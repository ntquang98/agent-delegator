package hub

import (
	"bytes"
	"strings"
	"testing"
)

func TestServerRespondsToInitialize(t *testing.T) {
	manager, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	err = NewServer(manager).Serve(strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`), &output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"protocolVersion":"2025-03-26"`) {
		t.Fatalf("initialize response missing: %q", output.String())
	}
}
