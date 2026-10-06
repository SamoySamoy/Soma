// Package tasks keeps the things the owner needs to do (TSK-01..02). Recurrence
// and reminders come later.
package tasks

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/SamoySamoy/Soma/internal/platform/apperr"
)

// Field limits. They match the contract in api/openapi.yaml.
const (
	maxTitle = 200
	maxNotes = 5000
)

// Task is one thing to do.
type Task struct {
	ID          uuid.UUID
	Title       string
	Notes       string
	DueOn       string // YYYY-MM-DD, or empty
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Version     int32
}

// Completed reports whether the task is done.
func (t Task) Completed() bool { return t.CompletedAt != nil }

// Overdue reports whether an open task is due before today.
func (t Task) Overdue(today time.Time) bool {
	if t.Completed() || t.DueOn == "" {
		return false
	}
	due, err := time.Parse(time.DateOnly, t.DueOn)
	return err == nil && due.Before(today)
}

// Input is the writable part of a task.
type Input struct {
	Title string
	Notes string
	DueOn string
}

// Patch is a JSON Merge Patch over the writable fields. A missing key keeps the
// current value; null clears it. Values are string or nil.
type Patch map[string]any

// Page is one page of tasks. Next is set when more remain.
type Page struct {
	Items []Task
	Next  *uuid.UUID
}

func (in Input) normalized() Input {
	return Input{
		Title: strings.TrimSpace(in.Title),
		Notes: strings.TrimSpace(in.Notes),
		DueOn: strings.TrimSpace(in.DueOn),
	}
}

func (in Input) validate() error {
	fields := map[string]string{}
	if in.Title == "" {
		fields["title"] = "Enter what needs doing."
	}
	if utf8.RuneCountInString(in.Title) > maxTitle {
		fields["title"] = fmt.Sprintf("Title can be at most %d characters.", maxTitle)
	}
	if utf8.RuneCountInString(in.Notes) > maxNotes {
		fields["notes"] = fmt.Sprintf("Notes can be at most %d characters.", maxNotes)
	}
	if in.DueOn != "" {
		if _, err := time.Parse(time.DateOnly, in.DueOn); err != nil {
			fields["due_on"] = "Use the format YYYY-MM-DD."
		}
	}
	if len(fields) > 0 {
		return apperr.Invalid("tasks.invalid_task", "Some fields need attention.", fields)
	}
	return nil
}

// apply returns cur with the patch applied. Unknown keys and wrong value types
// are refused.
func (p Patch) apply(cur Input) (Input, error) {
	bad := map[string]string{}
	for key, v := range p {
		s, isText := textOf(v)
		switch key {
		case "title":
			if !isText {
				bad[key] = "Use text."
				continue
			}
			cur.Title = s
		case "notes":
			if !isText {
				bad[key] = "Use text."
				continue
			}
			cur.Notes = s
		case "due_on":
			if !isText {
				bad[key] = "Use text."
				continue
			}
			cur.DueOn = s
		default:
			bad[key] = "This field can't be changed."
		}
	}
	if len(bad) > 0 {
		return Input{}, apperr.Invalid("tasks.invalid_task", "Some fields need attention.", bad)
	}
	return cur, nil
}

func textOf(v any) (string, bool) {
	switch x := v.(type) {
	case nil:
		return "", true
	case string:
		return x, true
	default:
		return "", false
	}
}
