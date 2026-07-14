package hub

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

type Server struct{ manager *Manager }

func NewServer(manager *Manager) *Server { return &Server{manager: manager} }

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func (s *Server) Serve(input io.Reader, output io.Writer) error {
	scanner := bufio.NewScanner(input)
	buffer := make([]byte, 64*1024)
	scanner.Buffer(buffer, 4*1024*1024)
	for scanner.Scan() {
		var request rpcRequest
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil {
			continue
		}
		if len(request.ID) == 0 {
			continue
		}
		result, rpcErr := s.handle(request)
		response := map[string]any{"jsonrpc": "2.0", "id": json.RawMessage(request.ID)}
		if rpcErr != nil {
			response["error"] = map[string]any{"code": -32602, "message": rpcErr.Error()}
		} else {
			response["result"] = result
		}
		encoded, _ := json.Marshal(response)
		if _, err := fmt.Fprintln(output, string(encoded)); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (s *Server) handle(request rpcRequest) (any, error) {
	switch request.Method {
	case "initialize":
		return map[string]any{"protocolVersion": "2025-03-26", "serverInfo": map[string]string{"name": "delegate-hub", "version": "0.1.0"}, "capabilities": map[string]any{"tools": map[string]any{}}}, nil
	case "tools/list":
		return map[string]any{"tools": tools()}, nil
	case "tools/call":
		var params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(request.Params, &params); err != nil {
			return nil, fmt.Errorf("invalid tools/call parameters")
		}
		return s.call(params.Name, params.Arguments)
	default:
		return nil, fmt.Errorf("method %q is not supported", request.Method)
	}
}

func (s *Server) call(name string, args json.RawMessage) (any, error) {
	var value any
	switch name {
	case "delegate_start":
		var request StartRequest
		if err := json.Unmarshal(args, &request); err != nil {
			return nil, fmt.Errorf("invalid delegate_start arguments")
		}
		job, err := s.manager.Start(request)
		if err != nil {
			return nil, err
		}
		value = job
	case "delegate_status":
		var request struct {
			JobID string `json:"jobId"`
		}
		if err := json.Unmarshal(args, &request); err != nil {
			return nil, fmt.Errorf("invalid delegate_status arguments")
		}
		job, err := s.manager.Status(request.JobID)
		if err != nil {
			return nil, err
		}
		value = job
	case "delegate_result":
		var request struct {
			JobID string `json:"jobId"`
		}
		if err := json.Unmarshal(args, &request); err != nil {
			return nil, fmt.Errorf("invalid delegate_result arguments")
		}
		result, err := s.manager.Result(request.JobID)
		if err != nil {
			return nil, err
		}
		value = result
	case "delegate_cancel":
		var request struct {
			JobID string `json:"jobId"`
		}
		if err := json.Unmarshal(args, &request); err != nil {
			return nil, fmt.Errorf("invalid delegate_cancel arguments")
		}
		job, err := s.manager.Cancel(request.JobID)
		if err != nil {
			return nil, err
		}
		value = job
	default:
		return nil, fmt.Errorf("tool %q is not supported", name)
	}
	encoded, _ := json.Marshal(value)
	return map[string]any{"content": []map[string]string{{"type": "text", "text": string(encoded)}}, "structuredContent": value}, nil
}

func tools() []map[string]any {
	return []map[string]any{
		{"name": "delegate_start", "description": "Start a local coding task. Defaults to Grok and falls back to Cursor only if Grok is unavailable. Read-only is the default.", "inputSchema": map[string]any{"type": "object", "required": []string{"workspace", "task"}, "properties": map[string]any{"provider": map[string]any{"type": "string", "enum": []string{"auto", "grok", "cursor"}}, "fallbackToCursor": map[string]string{"type": "boolean"}, "mode": map[string]any{"type": "string", "enum": []string{"read_only", "write"}}, "workspace": map[string]string{"type": "string"}, "task": map[string]string{"type": "string"}, "maxTurns": map[string]any{"type": "integer", "minimum": 1, "maximum": 50}}}},
		{"name": "delegate_status", "description": "Read the current status of a delegated job.", "inputSchema": map[string]any{"type": "object", "required": []string{"jobId"}, "properties": map[string]any{"jobId": map[string]string{"type": "string"}}}},
		{"name": "delegate_result", "description": "Read the normalized result and redacted local output of a delegated job.", "inputSchema": map[string]any{"type": "object", "required": []string{"jobId"}, "properties": map[string]any{"jobId": map[string]string{"type": "string"}}}},
		{"name": "delegate_cancel", "description": "Cancel a running delegated job.", "inputSchema": map[string]any{"type": "object", "required": []string{"jobId"}, "properties": map[string]any{"jobId": map[string]string{"type": "string"}}}},
	}
}
