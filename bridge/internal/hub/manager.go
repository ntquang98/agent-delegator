package hub

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Manager struct {
	mu        sync.RWMutex
	jobs      map[string]*Result
	processes map[string]runningProcess
	stateDir  string
	lookPath  func(string) (string, error)
}

type runningProcess struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc
}

func NewManager(stateDir string) (*Manager, error) {
	if stateDir == "" {
		cache, err := os.UserCacheDir()
		if err != nil {
			return nil, fmt.Errorf("resolve user cache directory: %w", err)
		}
		stateDir = filepath.Join(cache, "delegate-hub", "jobs")
	}
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return nil, fmt.Errorf("create job directory: %w", err)
	}
	return &Manager{jobs: make(map[string]*Result), processes: make(map[string]runningProcess), stateDir: stateDir, lookPath: exec.LookPath}, nil
}

func (m *Manager) Start(request StartRequest) (Job, error) {
	if strings.TrimSpace(request.Task) == "" {
		return Job{}, fmt.Errorf("task is required")
	}
	if request.Workspace == "" {
		return Job{}, fmt.Errorf("workspace is required")
	}
	workspace, err := filepath.Abs(request.Workspace)
	if err != nil {
		return Job{}, fmt.Errorf("resolve workspace: %w", err)
	}
	info, err := os.Stat(workspace)
	if err != nil || !info.IsDir() {
		return Job{}, fmt.Errorf("workspace must be an existing directory")
	}
	request.Workspace = workspace
	if request.Mode == "" {
		request.Mode = ModeReadOnly
	}
	if request.Mode != ModeReadOnly && request.Mode != ModeWrite {
		return Job{}, fmt.Errorf("mode must be read_only or write")
	}
	if request.Provider == "" || request.Provider == ProviderAuto {
		request.Provider = ProviderGrok
		request.FallbackToCursor = true
	} else {
		request.FallbackToCursor = false
	}
	if request.Provider == ProviderGrok {
		if _, err := m.lookPath("grok"); err != nil && request.FallbackToCursor {
			request.Provider = ProviderCursor
		}
	}
	if _, err := m.lookPath(executableFor(request.Provider)); err != nil {
		return Job{}, fmt.Errorf("%s CLI is not available on PATH: %w", request.Provider, err)
	}

	spec, err := BuildCommand(request)
	if err != nil {
		return Job{}, err
	}
	id, err := jobID()
	if err != nil {
		return Job{}, err
	}
	logPath := filepath.Join(m.stateDir, id+".log")
	job := Job{ID: id, Provider: request.Provider, Mode: request.Mode, Workspace: workspace, Status: "running", StartedAt: time.Now().UTC(), LogPath: logPath}
	result := &Result{Job: job}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, spec.Executable, spec.Args...)
	cmd.Dir = spec.Dir

	m.mu.Lock()
	m.jobs[id] = result
	m.processes[id] = runningProcess{cmd: cmd, cancel: cancel}
	if err := m.persistJob(job); err != nil {
		delete(m.jobs, id)
		delete(m.processes, id)
		m.mu.Unlock()
		cancel()
		return Job{}, err
	}
	m.mu.Unlock()

	go m.run(id, cmd)
	return job, nil
}

func executableFor(provider Provider) string {
	if provider == ProviderCursor {
		return "cursor-agent.cmd"
	}
	return "grok"
}

func (m *Manager) run(id string, cmd *exec.Cmd) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	outputText := redactSecrets(stdout.String())
	stderrText := redactSecrets(stderr.String())

	m.mu.Lock()
	defer m.mu.Unlock()
	result := m.jobs[id]
	result.Output = outputText
	populateNormalizedResult(result)
	result.Job.FinishedAt = time.Now().UTC()
	result.Job.Status = "completed"
	if err != nil {
		if result.Job.Status != "cancelled" {
			result.Job.Status = "failed"
			result.Job.Error = err.Error()
		}
		var exitError *exec.ExitError
		if os.IsNotExist(err) {
			result.Job.Error = "delegated executable was not found"
		} else if errors.As(err, &exitError) {
			code := exitError.ExitCode()
			result.Job.ExitCode = &code
		}
	}
	if err := os.WriteFile(result.Job.LogPath, []byte(outputText), 0o600); err != nil {
		result.Job.Error = "write local job log: " + err.Error()
		result.Job.Status = "failed"
	}
	if stderrText != "" {
		_ = os.WriteFile(result.Job.LogPath+".stderr", []byte(stderrText), 0o600)
	}
	if err := m.persistJob(result.Job); err != nil {
		result.Job.Error = "persist job state: " + err.Error()
		result.Job.Status = "failed"
	}
	delete(m.processes, id)
}

func (m *Manager) Status(id string) (Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result, ok := m.jobs[id]
	if !ok {
		return Job{}, fmt.Errorf("job %q was not found", id)
	}
	return result.Job, nil
}

func (m *Manager) Result(id string) (Result, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result, ok := m.jobs[id]
	if !ok {
		return Result{}, fmt.Errorf("job %q was not found", id)
	}
	return *result, nil
}

func (m *Manager) Cancel(id string) (Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	result, ok := m.jobs[id]
	if !ok {
		return Job{}, fmt.Errorf("job %q was not found", id)
	}
	process, running := m.processes[id]
	if !running {
		return result.Job, nil
	}
	process.cancel()
	killProcessTree(process.cmd)
	result.Job.Status = "cancelled"
	result.Job.FinishedAt = time.Now().UTC()
	if err := m.persistJob(result.Job); err != nil {
		result.Job.Error = "persist job state: " + err.Error()
		result.Job.Status = "failed"
	}
	return result.Job, nil
}

func jobID() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func redactSecrets(value string) string {
	pattern := regexp.MustCompile(`(?i)(XAI_API_KEY|OPENAI_API_KEY|ANTHROPIC_API_KEY|CURSOR_API_KEY)(["']?\s*[:=]\s*["']?)([^\s"',}]+)`)
	return pattern.ReplaceAllString(value, "$1$2[REDACTED]")
}

func (m *Manager) persistJob(job Job) error {
	encoded, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(m.stateDir, job.ID+".json"), encoded, 0o600)
}

func populateNormalizedResult(result *Result) {
	var output struct {
		Text       string `json:"text"`
		Result     string `json:"result"`
		SessionID  string `json:"sessionId"`
		SessionID2 string `json:"session_id"`
		RequestID  string `json:"requestId"`
		RequestID2 string `json:"request_id"`
	}
	if json.Unmarshal([]byte(result.Output), &output) != nil {
		return
	}
	result.Text = output.Text
	if result.Text == "" {
		result.Text = output.Result
	}
	result.SessionID = output.SessionID
	if result.SessionID == "" {
		result.SessionID = output.SessionID2
	}
	result.RequestID = output.RequestID
	if result.RequestID == "" {
		result.RequestID = output.RequestID2
	}
}
