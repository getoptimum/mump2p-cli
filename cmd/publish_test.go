package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testMaxBytes = 1 << 20

// stdinFrom returns the read end of a pipe pre-filled with content, simulating piped stdin.
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

// terminalStdin returns a char device (like an interactive terminal), or skips if unavailable.
func terminalStdin(t *testing.T) *os.File {
	t.Helper()
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Skipf("cannot open %s: %v", os.DevNull, err)
	}
	t.Cleanup(func() { f.Close() })
	if !isTerminal(f) {
		t.Skipf("%s is not a char device on this platform", os.DevNull)
	}
	return f
}

// messageSrc builds a payloadSource for an explicit --message.
func messageSrc(msg string) payloadSource {
	return payloadSource{message: msg, messageSet: true}
}

// fileSrc builds a payloadSource for an explicit --file.
func fileSrc(path string) payloadSource {
	return payloadSource{filePath: path, fileSet: true}
}

// TestResolvePublishPayload_MessageFlag checks --message is used when stdin is a terminal.
func TestResolvePublishPayload_MessageFlag(t *testing.T) {
	got, err := resolvePublishPayload(messageSrc("hello"), terminalStdin(t), testMaxBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("want %q, got %q", "hello", got)
	}
}

// TestResolvePublishPayload_StdinOverridesMessage checks piped stdin wins over --message (#92).
func TestResolvePublishPayload_StdinOverridesMessage(t *testing.T) {
	got, err := resolvePublishPayload(messageSrc("flag"), stdinFrom(t, "piped"), testMaxBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "piped" {
		t.Fatalf("want %q, got %q", "piped", got)
	}
}

// TestResolvePublishPayload_EmptyStdinFallsBackToMessage checks an empty pipe does not discard --message.
func TestResolvePublishPayload_EmptyStdinFallsBackToMessage(t *testing.T) {
	got, err := resolvePublishPayload(messageSrc("flag"), stdinFrom(t, ""), testMaxBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "flag" {
		t.Fatalf("want %q, got %q", "flag", got)
	}
}

// TestResolvePublishPayload_ExplicitEmptyMessage checks an explicitly empty --message is an error, not a stdin fallback.
func TestResolvePublishPayload_ExplicitEmptyMessage(t *testing.T) {
	_, err := resolvePublishPayload(messageSrc(""), terminalStdin(t), testMaxBytes)
	if err == nil {
		t.Fatal("expected error for explicit empty --message")
	}
	_, err = resolvePublishPayload(messageSrc(""), stdinFrom(t, ""), testMaxBytes)
	if err == nil {
		t.Fatal("expected error for explicit empty --message with empty stdin")
	}
}

// TestResolvePublishPayload_FileFlag checks --file contents are used as-is.
func TestResolvePublishPayload_FileFlag(t *testing.T) {
	path := filepath.Join(t.TempDir(), "payload.json")
	if err := os.WriteFile(path, []byte(`{"a":1}`), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	got, err := resolvePublishPayload(fileSrc(path), stdinFrom(t, "ignored"), testMaxBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != `{"a":1}` {
		t.Fatalf("want %q, got %q", `{"a":1}`, got)
	}
}

// TestResolvePublishPayload_FileMissing checks a missing --file path is reported.
func TestResolvePublishPayload_FileMissing(t *testing.T) {
	_, err := resolvePublishPayload(fileSrc(filepath.Join(t.TempDir(), "nope")), stdinFrom(t, ""), testMaxBytes)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

// TestValidatePayloadFlags checks rejected flag combinations, including empty explicit values.
func TestValidatePayloadFlags(t *testing.T) {
	cases := []struct {
		name string
		src  payloadSource
	}{
		{"message and file", payloadSource{message: "m", messageSet: true, filePath: "f", fileSet: true}},
		{"empty message and file", payloadSource{message: "", messageSet: true, filePath: "f", fileSet: true}},
		{"empty file path", payloadSource{filePath: "", fileSet: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validatePayloadFlags(tc.src); err == nil {
				t.Fatal("expected validation error")
			}
			if _, err := resolvePublishPayload(tc.src, stdinFrom(t, "data"), testMaxBytes); err == nil {
				t.Fatal("expected resolver to reject invalid flags")
			}
		})
	}
	if err := validatePayloadFlags(messageSrc("ok")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestResolvePublishPayload_StdinWhenNoFlags checks stdin is used when no payload flags are set.
func TestResolvePublishPayload_StdinWhenNoFlags(t *testing.T) {
	got, err := resolvePublishPayload(payloadSource{}, stdinFrom(t, "from stdin\n"), testMaxBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "from stdin\n" {
		t.Fatalf("want %q, got %q", "from stdin\n", got)
	}
}

// TestResolvePublishPayload_FileDashReadsStdin checks --file=- reads stdin explicitly.
func TestResolvePublishPayload_FileDashReadsStdin(t *testing.T) {
	got, err := resolvePublishPayload(fileSrc("-"), stdinFrom(t, "dash"), testMaxBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "dash" {
		t.Fatalf("want %q, got %q", "dash", got)
	}
}

// TestResolvePublishPayload_EmptyStdin checks empty stdin without a fallback is an error.
func TestResolvePublishPayload_EmptyStdin(t *testing.T) {
	if _, err := resolvePublishPayload(payloadSource{}, stdinFrom(t, ""), testMaxBytes); err == nil {
		t.Fatal("expected error for empty stdin")
	}
	if _, err := resolvePublishPayload(fileSrc("-"), stdinFrom(t, ""), testMaxBytes); err == nil {
		t.Fatal("expected error for empty stdin with --file=-")
	}
}

// TestResolvePublishPayload_TerminalNoFlags checks the command fails fast instead of blocking on a terminal.
func TestResolvePublishPayload_TerminalNoFlags(t *testing.T) {
	if _, err := resolvePublishPayload(payloadSource{}, terminalStdin(t), testMaxBytes); err == nil {
		t.Fatal("expected error when no flags and stdin is a terminal")
	}
	if _, err := resolvePublishPayload(payloadSource{}, nil, testMaxBytes); err == nil {
		t.Fatal("expected error when no flags and stdin is nil")
	}
}

// TestResolvePublishPayload_StdinSizeLimit checks stdin larger than maxBytes is rejected.
func TestResolvePublishPayload_StdinSizeLimit(t *testing.T) {
	const limit = 16
	if _, err := resolvePublishPayload(payloadSource{}, stdinFrom(t, strings.Repeat("x", limit+1)), limit); err == nil {
		t.Fatal("expected error for oversized stdin")
	}
	got, err := resolvePublishPayload(payloadSource{}, stdinFrom(t, strings.Repeat("x", limit)), limit)
	if err != nil {
		t.Fatalf("unexpected error at exact limit: %v", err)
	}
	if len(got) != limit {
		t.Fatalf("want %d bytes, got %d", limit, len(got))
	}
}
