package tasks

import (
	"time"

	tstore "github.com/SamoySamoy/Soma/internal/modules/tasks/store"
	"github.com/SamoySamoy/Soma/internal/platform/apperr"
)

func (t Task) input() Input {
	return Input{Title: t.Title, Notes: t.Notes, DueOn: t.DueOn}
}

func fromGetRow(r tstore.GetTaskRow) Task {
	return Task{
		ID:          r.ID,
		Title:       r.Title,
		Notes:       stringOf(r.Notes),
		DueOn:       dateOf(r.DueOn),
		CompletedAt: r.CompletedAt,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
		Version:     r.Version,
	}
}

func fromListRow(r tstore.ListTasksRow) Task {
	return Task{
		ID:          r.ID,
		Title:       r.Title,
		Notes:       stringOf(r.Notes),
		DueOn:       dateOf(r.DueOn),
		CompletedAt: r.CompletedAt,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
		Version:     r.Version,
	}
}

// dueOf parses a validated YYYY-MM-DD string for storage; "" stays NULL.
func dueOf(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return nil, apperr.Invalid("tasks.invalid_task", "Some fields need attention.",
			map[string]string{"due_on": "Use the format YYYY-MM-DD."})
	}
	return &t, nil
}

func dateOf(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.DateOnly)
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func stringOf(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
