package journal

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	jstore "github.com/SamoySamoy/Soma/internal/modules/journal/store"
	"github.com/SamoySamoy/Soma/internal/platform/apperr"
)

func (e Entry) input() Input {
	return Input{EntryDate: e.EntryDate, Title: e.Title, Body: e.Body, Mood: e.Mood}
}

func fromGetRow(r jstore.GetEntryRow) Entry {
	return Entry{
		ID:        r.ID,
		EntryDate: r.EntryDate.Format(time.DateOnly),
		Title:     stringOf(r.Title),
		Body:      r.Body,
		Mood:      moodOf16(r.Mood),
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
		Version:   r.Version,
	}
}

func fromListRow(r jstore.ListEntriesRow) Entry {
	return Entry{
		ID:        r.ID,
		EntryDate: r.EntryDate.Format(time.DateOnly),
		Title:     stringOf(r.Title),
		Body:      r.Body,
		Mood:      moodOf16(r.Mood),
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
		Version:   r.Version,
	}
}

// dateOf parses a validated YYYY-MM-DD string for storage.
func dateOf(s string) (time.Time, error) {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return time.Time{}, apperr.Invalid("journal.invalid_entry", "Some fields need attention.",
			map[string]string{"entry_date": "Use the format YYYY-MM-DD."})
	}
	return t, nil
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

func moodPtr(n int) pgtype.Int2 {
	if n == 0 {
		return pgtype.Int2{}
	}
	return pgtype.Int2{Int16: int16(n), Valid: true} //nolint:gosec // G115: mood is validated to 1..5 before this point
}

func moodOf16(p pgtype.Int2) int {
	if !p.Valid {
		return 0
	}
	return int(p.Int16)
}
