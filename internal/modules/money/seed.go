package money

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/SamoySamoy/Soma/internal/modules/money/store"
	"github.com/SamoySamoy/Soma/internal/platform/clock"
	"github.com/SamoySamoy/Soma/internal/platform/db"
	"github.com/SamoySamoy/Soma/internal/space"
)

// defaultCategories is the starting tree (FIN-03). It can be edited freely.
var defaultCategories = []struct {
	Name     string
	Kind     string
	Children []string
}{
	{"Food", "expense", []string{"Groceries", "Restaurants and cafes"}},
	{"Home", "expense", []string{"Rent", "Utilities"}},
	{"Transport", "expense", []string{"Fuel", "Public transport"}},
	{"Bills and subscriptions", "expense", nil},
	{"Health", "expense", nil},
	{"Shopping", "expense", nil},
	{"Entertainment", "expense", nil},
	{"Other", "expense", nil},
	{"Salary", "income", nil},
	{"Other income", "income", nil},
}

// SeedDefaults gives a space its money settings and default categories. It runs
// at startup for the local owner and does nothing once categories exist.
func SeedDefaults(ctx context.Context, d *db.DB, clk clock.Clock) error {
	spaceID := space.LocalSpaceID
	ctx = db.WithUserID(space.WithSpace(ctx, spaceID), space.LocalOwnerID)
	now := clk.Now()
	return d.InTx(ctx, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.InsertSettings(ctx, store.InsertSettingsParams{SpaceID: spaceID, Now: now}); err != nil {
			return fmt.Errorf("insert money settings: %w", err)
		}
		n, err := q.CountCategories(ctx, spaceID)
		if err != nil {
			return fmt.Errorf("count categories: %w", err)
		}
		if n > 0 {
			return nil
		}
		for _, top := range defaultCategories {
			parentID, err := insertCategory(ctx, q, spaceID, nil, top.Name, top.Kind, now)
			if err != nil {
				return err
			}
			for _, child := range top.Children {
				if _, err := insertCategory(ctx, q, spaceID, &parentID, child, top.Kind, now); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func insertCategory(ctx context.Context, q *store.Queries, spaceID uuid.UUID, parent *uuid.UUID, name, kind string, now time.Time) (uuid.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("category id: %w", err)
	}
	if err := q.InsertCategory(ctx, store.InsertCategoryParams{
		ID: id, SpaceID: spaceID, ParentID: parent, Name: name, Kind: kind, Now: now,
	}); err != nil {
		return uuid.Nil, fmt.Errorf("insert category %s: %w", name, err)
	}
	return id, nil
}
