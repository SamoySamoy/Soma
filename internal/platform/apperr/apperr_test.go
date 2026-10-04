package apperr

import (
	"errors"
	"fmt"
	"testing"
)

func TestKindOf(t *testing.T) {
	t.Parallel()

	cause := errors.New("connection reset")

	tests := []struct {
		name string
		err  error
		want Kind
	}{
		{"plain error is internal", cause, KindInternal},
		{"nil is internal", nil, KindInternal},
		{"direct app error", NotFound("x.not_found", "Not found"), KindNotFound},
		{"wrapped app error", fmt.Errorf("load: %w", Conflict("x.dup", "Duplicate")), KindConflict},
		{"wrap keeps kind", Wrap(cause, KindInvalid, "x.bad", "Bad"), KindInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := KindOf(tt.err); got != tt.want {
				t.Errorf("KindOf() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWrapPreservesCause(t *testing.T) {
	t.Parallel()

	cause := errors.New("disk full")
	err := fmt.Errorf("save: %w", Wrap(cause, KindInternal, "x.io", "Could not save"))

	if !errors.Is(err, cause) {
		t.Error("errors.Is(err, cause) = false, want true")
	}
}

func TestKindString(t *testing.T) {
	t.Parallel()

	if got := KindPreconditionFailed.String(); got != "precondition_failed" {
		t.Errorf("String() = %q, want precondition_failed", got)
	}
	if got := Kind(99).String(); got != "internal" {
		t.Errorf("unknown kind String() = %q, want internal", got)
	}
}
