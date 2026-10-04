package log

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
)

func TestNewRedactsSensitiveKeys(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	l := New(&buf, "info", "json")
	l.Info("login", "user_id", "u-1", "password", "hunter2", "Session_Token", "abc", "user_email", "a@b.c")

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal log line: %v", err)
	}

	if got["user_id"] != "u-1" {
		t.Errorf("user_id = %v, want u-1", got["user_id"])
	}
	for _, k := range []string{"password", "Session_Token", "user_email"} {
		if got[k] != Redacted {
			t.Errorf("%s = %v, want %q", k, got[k], Redacted)
		}
	}
}

func TestNewRespectsLevel(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	l := New(&buf, "warn", "text")
	l.Info("hidden")
	if buf.Len() != 0 {
		t.Errorf("info logged at warn level: %q", buf.String())
	}
}

func TestFromContext(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	l := New(&buf, "info", "json")
	ctx := WithLogger(context.Background(), l)

	if FromContext(ctx) != l {
		t.Error("FromContext did not return the stored logger")
	}
	if FromContext(context.Background()) == nil {
		t.Error("FromContext without logger returned nil")
	}
}
