package money

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/SamoySamoy/Soma/internal/bodymap"
)

// BodymapProvider reports the Money area: attention when spending in a category
// has gone past its budget this month, otherwise calm. A budget whose currency
// has no rate yet doesn't fail the whole map; it is simply not counted.
type BodymapProvider struct{}

// Status implements bodymap.Provider.
func (BodymapProvider) Status(ctx context.Context, tx pgx.Tx, spaceID uuid.UUID, now time.Time) (bodymap.Result, error) {
	settings, err := loadSettings(ctx, tx, spaceID)
	if err != nil {
		return bodymap.Result{}, err
	}
	lines, err := budgetLines(ctx, tx, spaceID, settings.BaseCurrency, now, now)
	if err != nil {
		if errMissingRateOf(err) {
			return bodymap.Result{Level: bodymap.LevelCalm, Counts: map[string]int{"budgets": 0, "overspent": 0}}, nil
		}
		return bodymap.Result{}, err
	}
	budgets, overspent := 0, 0
	for _, l := range lines {
		if l.BudgetMinor > 0 {
			budgets++
			if l.SpentMinor > l.BudgetMinor {
				overspent++
			}
		}
	}
	level := bodymap.LevelCalm
	if overspent > 0 {
		level = bodymap.LevelAttention
	}
	return bodymap.Result{Level: level, Counts: map[string]int{"budgets": budgets, "overspent": overspent}}, nil
}
