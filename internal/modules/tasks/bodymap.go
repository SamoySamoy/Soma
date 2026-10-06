package tasks

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/SamoySamoy/Soma/internal/bodymap"
	tstore "github.com/SamoySamoy/Soma/internal/modules/tasks/store"
)

// BodymapProvider reports the Responsibilities area: urgent when a task has been
// overdue for more than three days, attention when something is overdue or due
// today, otherwise calm.
type BodymapProvider struct{}

// Status implements bodymap.Provider.
func (BodymapProvider) Status(ctx context.Context, tx pgx.Tx, spaceID uuid.UUID, now time.Time) (bodymap.Result, error) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	c, err := tstore.New(tx).TaskCounts(ctx, tstore.TaskCountsParams{SpaceID: spaceID, Today: today})
	if err != nil {
		return bodymap.Result{}, fmt.Errorf("task counts: %w", err)
	}
	level := bodymap.LevelCalm
	switch {
	case c.OverdueLong > 0:
		level = bodymap.LevelUrgent
	case c.Overdue > 0 || c.DueToday > 0:
		level = bodymap.LevelAttention
	}
	return bodymap.Result{Level: level, Counts: map[string]int{
		"overdue":      int(c.Overdue),
		"due_today":    int(c.DueToday),
		"overdue_long": int(c.OverdueLong),
	}}, nil
}
