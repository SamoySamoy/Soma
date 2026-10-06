package tasks

import (
	"testing"
	"time"

	"github.com/SamoySamoy/Soma/internal/platform/apperr"
)

func TestOverdue(t *testing.T) {
	t.Parallel()

	today := time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC)
	done := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		task Task
		want bool
	}{
		{"due yesterday and open", Task{DueOn: "2026-10-05"}, true},
		{"due today is not overdue", Task{DueOn: "2026-10-06"}, false},
		{"no due date is never overdue", Task{}, false},
		{"done tasks are never overdue", Task{DueOn: "2026-10-01", CompletedAt: &done}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.task.Overdue(today); got != tt.want {
				t.Errorf("Overdue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInputValidate(t *testing.T) {
	t.Parallel()

	if err := (Input{Title: "Renew passport", DueOn: "2026-12-01"}).normalized().validate(); err != nil {
		t.Fatalf("valid task refused: %v", err)
	}
	for name, in := range map[string]Input{
		"title required": {Title: "  "},
		"date format":    {Title: "x", DueOn: "1 Dec"},
	} {
		e, ok := apperr.As(in.normalized().validate())
		if !ok || e.Kind != apperr.KindInvalid {
			t.Errorf("%s: want an Invalid error", name)
		}
	}
}

func TestPatchApply(t *testing.T) {
	t.Parallel()

	cur := Input{Title: "Call the bank", Notes: "ask about fees", DueOn: "2026-10-07"}
	got, err := Patch{"due_on": nil, "notes": "ask about the card"}.apply(cur)
	if err != nil {
		t.Fatalf("apply() error = %v", err)
	}
	want := Input{Title: "Call the bank", Notes: "ask about the card"}
	if got != want {
		t.Errorf("apply() = %+v, want %+v", got, want)
	}
	if _, err := (Patch{"completed": true}).apply(cur); err == nil {
		t.Error("apply(completed) succeeded; completion goes through its own endpoint")
	}
}
