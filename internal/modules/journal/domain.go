// Package journal keeps the owner's private diary (JRN-01..04). Entries are
// visible only to their author; the service and row-level security both
// enforce that.
package journal

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
	maxBody  = 100000
)

// Entry is one journal entry. Mood 0 means none was recorded.
type Entry struct {
	ID        uuid.UUID
	EntryDate string // YYYY-MM-DD
	Title     string
	Body      string
	Mood      int
	CreatedAt time.Time
	UpdatedAt time.Time
	Version   int32
}

// Input is the writable part of an entry.
type Input struct {
	EntryDate string
	Title     string
	Body      string
	Mood      int
}

// Patch is a JSON Merge Patch over the writable fields. A missing key keeps
// the current value; null clears it. Values are whatever encoding/json gives:
// string, float64 or nil.
type Patch map[string]any

// Page is one page of entries. Next is set when more remain.
type Page struct {
	Items []Entry
	Next  *uuid.UUID
}

func (in Input) normalized() Input {
	return Input{
		EntryDate: strings.TrimSpace(in.EntryDate),
		Title:     strings.TrimSpace(in.Title),
		Body:      strings.TrimSpace(in.Body),
		Mood:      in.Mood,
	}
}

func (in Input) validate() error {
	fields := map[string]string{}
	if _, err := time.Parse(time.DateOnly, in.EntryDate); err != nil {
		fields["entry_date"] = "Use the format YYYY-MM-DD."
	}
	if utf8.RuneCountInString(in.Title) > maxTitle {
		fields["title"] = fmt.Sprintf("Title can be at most %d characters.", maxTitle)
	}
	if utf8.RuneCountInString(in.Body) > maxBody {
		fields["body"] = fmt.Sprintf("Entry can be at most %d characters.", maxBody)
	}
	if in.Title == "" && in.Body == "" {
		fields["body"] = "Write something first."
	}
	if in.Mood < 0 || in.Mood > 5 {
		fields["mood"] = "Choose a mood from 1 to 5."
	}
	if len(fields) > 0 {
		return invalid(fields)
	}
	return nil
}

// apply returns cur with the patch applied. Unknown keys and wrong value types
// are refused, so a typo can't silently do nothing.
func (p Patch) apply(cur Input) (Input, error) {
	bad := map[string]string{}
	for key, v := range p {
		switch key {
		case "entry_date":
			s, ok := textOf(v)
			if !ok {
				bad[key] = "Use text."
				continue
			}
			cur.EntryDate = s
		case "title":
			s, ok := textOf(v)
			if !ok {
				bad[key] = "Use text."
				continue
			}
			cur.Title = s
		case "body":
			s, ok := textOf(v)
			if !ok {
				bad[key] = "Use text."
				continue
			}
			cur.Body = s
		case "mood":
			n, ok := moodOf(v)
			if !ok {
				bad[key] = "Choose a mood from 1 to 5."
				continue
			}
			cur.Mood = n
		default:
			bad[key] = "This field can't be changed."
		}
	}
	if len(bad) > 0 {
		return Input{}, invalid(bad)
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

func moodOf(v any) (int, bool) {
	switch x := v.(type) {
	case nil:
		return 0, true
	case float64:
		n := int(x)
		if float64(n) != x || n < 1 || n > 5 {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func invalid(fields map[string]string) error {
	return apperr.Invalid("journal.invalid_entry", "Some fields need attention.", fields)
}
