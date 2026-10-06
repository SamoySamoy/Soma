package journal

import (
	"strings"
	"testing"
	"time"

	"github.com/SamoySamoy/Soma/internal/platform/apperr"
)

func TestInputValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		in         Input
		wantFields []string
	}{
		{"a body is enough", Input{EntryDate: "2026-10-06", Body: "Quiet day."}, nil},
		{"a title alone is enough", Input{EntryDate: "2026-10-06", Title: "Hello"}, nil},
		{"an empty entry is refused", Input{EntryDate: "2026-10-06"}, []string{"body"}},
		{"date must be a date", Input{EntryDate: "06/10/2026", Body: "x"}, []string{"entry_date"}},
		{"title is limited", Input{EntryDate: "2026-10-06", Title: strings.Repeat("t", 201)}, []string{"title"}},
		{"mood is 1 to 5", Input{EntryDate: "2026-10-06", Body: "x", Mood: 6}, []string{"mood"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.in.normalized().validate()
			if len(tt.wantFields) == 0 {
				if err != nil {
					t.Fatalf("validate() = %v, want nil", err)
				}
				return
			}
			e, ok := apperr.As(err)
			if !ok || e.Kind != apperr.KindInvalid {
				t.Fatalf("validate() = %v, want an Invalid error", err)
			}
			for _, f := range tt.wantFields {
				if _, has := e.Fields[f]; !has {
					t.Errorf("fields = %v, want a message for %q", e.Fields, f)
				}
			}
		})
	}
}

func TestPatchApply(t *testing.T) {
	t.Parallel()

	cur := Input{EntryDate: "2026-10-06", Title: "Old", Body: "Old body", Mood: 2}

	got, err := Patch{"mood": 4.0, "title": nil}.apply(cur)
	if err != nil {
		t.Fatalf("apply() error = %v", err)
	}
	want := Input{EntryDate: "2026-10-06", Title: "", Body: "Old body", Mood: 4}
	if got != want {
		t.Errorf("apply() = %+v, want %+v", got, want)
	}

	for _, bad := range []Patch{
		{"mood": 4.5},
		{"mood": 9.0},
		{"mood": "happy"},
		{"body": 12.0},
		{"author": "someone"},
	} {
		if _, err := bad.apply(cur); err == nil {
			t.Errorf("apply(%v) succeeded, want an error", bad)
		}
	}
}

func TestDaysSince(t *testing.T) {
	t.Parallel()

	day := func(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }
	tests := []struct {
		name string
		now  time.Time
		last time.Time
		want int
	}{
		{"written today", day(2026, 10, 6), day(2026, 10, 6), 0},
		{"written three days ago", day(2026, 10, 6), day(2026, 10, 3), 3},
		{"crosses a month", day(2026, 11, 2), day(2026, 10, 30), 3},
		{"a future date counts as today", day(2026, 10, 6), day(2026, 10, 9), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := daysSince(tt.now, tt.last); got != tt.want {
				t.Errorf("daysSince() = %d, want %d", got, tt.want)
			}
		})
	}
}
