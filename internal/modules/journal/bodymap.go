package journal

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/SamoySamoy/Soma/internal/bodymap"
	jstore "github.com/SamoySamoy/Soma/internal/modules/journal/store"
	"github.com/SamoySamoy/Soma/internal/platform/db"
)

// staleAfterDays is how long without an entry before the Mind area asks for one.
const staleAfterDays = 3

// BodymapProvider reports the Mind area for the acting user: attention when
// their last entry is more than staleAfterDays old. Someone who has never
// written does not get nagged.
type BodymapProvider struct{}

// Status implements bodymap.Provider. It reads the acting user's own entries.
func (BodymapProvider) Status(ctx context.Context, tx pgx.Tx, spaceID uuid.UUID, now time.Time) (bodymap.Result, error) {
	author, ok := db.UserIDFrom(ctx)
	if !ok {
		return bodymap.Result{}, fmt.Errorf("mind status: no acting user")
	}
	stats, err := jstore.New(tx).LastEntryStats(ctx, jstore.LastEntryStatsParams{
		SpaceID: spaceID, AuthorID: author,
	})
	if err != nil {
		return bodymap.Result{}, fmt.Errorf("entry stats: %w", err)
	}
	days := 0
	level := bodymap.LevelCalm
	if stats.Entries > 0 {
		days = daysSince(now, stats.LastEntryDate)
		if days > staleAfterDays {
			level = bodymap.LevelAttention
		}
	}
	return bodymap.Result{
		Level:  level,
		Counts: map[string]int{"entries": int(stats.Entries), "days_since_last": days},
	}, nil
}

// daysSince counts whole days from now's date back to day. Future days count as 0.
func daysSince(now, day time.Time) int {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	d := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	if d.After(today) {
		return 0
	}
	return int(today.Sub(d).Hours() / 24)
}
