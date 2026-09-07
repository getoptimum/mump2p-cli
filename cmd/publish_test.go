package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// stdinFrom returns an *os.File whose read end yields the given content,
// simulating `echo "content" | mump2p publish ...`. An empty content string
// yields a pipe that is already closed on the write side.
func stdinFrom(t *testing.T, content string) *os.File {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	if content != "" {
		if _, err := w.WriteString(content); err != nil {
			t.Fatalf("write to pipe: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	t.Cleanup(func() { r.Close() })
	return r
}

func TestResolvePublishPayload_MessageFlag(t *testing.T) {
	got, err := resolvePublishPayload("hello", "", stdinFrom(t, "ignored"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("want %q, got %q", "hello", got)
	}
}

func TestResolvePublishPayload_FileFlag(t *testing.T) {
	path := filepath.Join(t.TempDir(), "payload.json")
	if err := os.WriteFile(path, []byte(`{"a":1}`), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	got, err := resolvePublishPayload("", path, stdinFrom(t, "ignored"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != `{"a":1}` {
		t.Fatalf("want %q, got %q", `{"a":1}`, got)
	}
}

func TestResolvePublishPayload_FileMissing(t *testing.T) {
	_, err := resolvePublishPayload("", filepath.Join(t.TempDir(), "nope"), stdinFrom(t, ""))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestResolvePublishPayload_StdinWhenNoFlags(t *testing.T) {
	got, err := resolvePublishPayload("", "", stdinFrom(t, "from stdin\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "from stdin\n" {
		t.Fatalf("want %q, got %q", "from stdin\n", got)
	}
}

func TestResolvePublishPayload_FileDashReadsStdin(t *testing.T) {
	got, err := resolvePublishPayload("", "-", stdinFrom(t, "dash"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "dash" {
		t.Fatalf("want %q, got %q", "dash", got)
	}
}

func TestResolvePublishPayload_EmptyStdin(t *testing.T) {
	_, err := resolvePublishPayload("", "", stdinFrom(t, ""))
	if err == nil {
		t.Fatal("expected error for empty stdin")
	}
}

func TestResolvePublishPayload_NilStdinIsTerminal(t *testing.T) {
	// A nil stdin is treated as an interactive terminal: with no flags we
	// must fail fast instead of blocking on input that can never arrive.
	_, err := resolvePublishPayload("", "", nil)
	if err == nil {
		t.Fatal("expected error when no flags and stdin is a terminal")
	}
}

func TestIsTerminal(t *testing.T) {
	if !isTerminal(nil) {
		t.Fatal("nil file should be treated as a terminal")
	}
	if isTerminal(stdinFrom(t, "x")) {
		t.Fatal("a pipe should not be treated as a terminal")
	}
}
