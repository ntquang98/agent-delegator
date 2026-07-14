package hub

import (
	"strings"
	"testing"
)

func TestRedactSecretsHandlesAssignmentAndJSON(t *testing.T) {
	input := `XAI_API_KEY=abc123 {"OPENAI_API_KEY":"def456"}`
	got := redactSecrets(input)
	if got == input || strings.Contains(got, "abc123") || strings.Contains(got, "def456") {
		t.Fatalf("secret was not redacted: %q", got)
	}
}

func TestPersistJobWritesState(t *testing.T) {
	manager, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.persistJob(Job{ID: "test-job", Status: "running"}); err != nil {
		t.Fatal(err)
	}
}
