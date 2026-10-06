package people

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/SamoySamoy/Soma/internal/bodymap"
	pstore "github.com/SamoySamoy/Soma/internal/modules/people/store"
)

// upcomingBirthdayDays is how far ahead a birthday counts as needing attention.
const upcomingBirthdayDays = 14

// BodymapProvider reports the Heart area: attention when a birthday falls in
// the next upcomingBirthdayDays, otherwise calm (BODY-02).
type BodymapProvider struct{}

// Status implements bodymap.Provider.
func (BodymapProvider) Status(ctx context.Context, tx pgx.Tx, spaceID uuid.UUID, now time.Time) (bodymap.Result, error) {
	birthdays, err := pstore.New(tx).ListBirthdays(ctx, spaceID)
	if err != nil {
		return bodymap.Result{}, fmt.Errorf("list birthdays: %w", err)
	}
	soon := 0
	for _, b := range birthdays {
		if b != nil && daysUntilBirthday(now, *b) <= upcomingBirthdayDays {
			soon++
		}
	}
	level := bodymap.LevelCalm
	if soon > 0 {
		level = bodymap.LevelAttention
	}
	return bodymap.Result{Level: level, Counts: map[string]int{"birthdays_soon": soon}}, nil
}

// daysUntilBirthday counts whole days from now's date to the next occurrence
// of birthday's month and day. A birthday today is 0. February 29 falls on
// March 1 in years that aren't leap years.
func daysUntilBirthday(now, birthday time.Time) int {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	next := time.Date(today.Year(), birthday.Month(), birthday.Day(), 0, 0, 0, 0, time.UTC)
	if next.Before(today) {
		next = next.AddDate(1, 0, 0)
	}
	return int(next.Sub(today).Hours() / 24)
}
