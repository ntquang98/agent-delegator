package hub

import "time"

// Provider identifies the local coding CLI that performs a delegated task.
type Provider string

const (
	ProviderGrok   Provider = "grok"
	ProviderCursor Provider = "cursor"
	ProviderAuto   Provider = "auto"
)

// Mode controls the permissions requested from the delegated CLI.
type Mode string

const (
	ModeReadOnly Mode = "read_only"
	ModeWrite    Mode = "write"
)

type StartRequest struct {
	Provider         Provider `json:"provider"`
	FallbackToCursor bool     `json:"fallbackToCursor"`
	Mode             Mode     `json:"mode"`
	Workspace        string   `json:"workspace"`
	Task             string   `json:"task"`
	MaxTurns         int      `json:"maxTurns"`
}

type Job struct {
	ID         string    `json:"id"`
	Provider   Provider  `json:"provider"`
	Mode       Mode      `json:"mode"`
	Workspace  string    `json:"workspace"`
	Status     string    `json:"status"`
	StartedAt  time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt,omitempty"`
	ExitCode   *int      `json:"exitCode,omitempty"`
	Error      string    `json:"error,omitempty"`
	LogPath    string    `json:"logPath"`
}

type Result struct {
	Job       Job    `json:"job"`
	Output    string `json:"output"`
	Text      string `json:"text,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	RequestID string `json:"requestId,omitempty"`
}
