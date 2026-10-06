//go:build integration

package self_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/SamoySamoy/Soma/internal/modules/self"
	"github.com/SamoySamoy/Soma/internal/platform/apperr"
	"github.com/SamoySamoy/Soma/internal/platform/clock"
	"github.com/SamoySamoy/Soma/internal/platform/db"
	"github.com/SamoySamoy/Soma/internal/space"
	"github.com/SamoySamoy/Soma/internal/testutil/pgtest"
)

func as(ctx context.Context, user uuid.UUID) context.Context {
	return space.WithSpace(db.WithUserID(ctx, user), space.LocalSpaceID)
}

func newService(t *testing.T) *self.Service {
	t.Helper()
	dbs := pgtest.New(t)
	app, err := db.Open(t.Context(), dbs.AppURL)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(app.Close)
	clk := clock.NewFake(time.Date(2026, 10, 6, 9, 0, 0, 0, time.UTC))
	if err := space.EnsureLocal(t.Context(), app, clk); err != nil {
		t.Fatalf("EnsureLocal: %v", err)
	}
	return self.NewService(app, clk)
}

// The profile exists from the start, empty, so the Face of the map can read it.
func TestProfileExistsAndStartsEmpty(t *testing.T) {
	t.Parallel()
	svc := newService(t)
	p, err := svc.Get(as(t.Context(), space.LocalOwnerID))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if p.Version != 1 || p.PreferredName != "" {
		t.Errorf("Get() = %+v, want version 1 and empty fields", p)
	}
}

func TestUpdateAndStaleVersion(t *testing.T) {
	t.Parallel()
	svc := newService(t)
	ctx := as(t.Context(), space.LocalOwnerID)

	p, err := svc.Update(ctx, 1, self.Patch{"preferred_name": "Lan", "birth_date": "1990-04-23"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if p.Version != 2 || p.PreferredName != "Lan" || p.BirthDate != "1990-04-23" {
		t.Fatalf("Update() = %+v, want version 2 with name and birth date", p)
	}

	_, err = svc.Update(ctx, 1, self.Patch{"bio": "stale"})
	if e, ok := apperr.As(err); !ok || e.Kind != apperr.KindPreconditionFailed {
		t.Errorf("stale update: error = %v, want PreconditionFailed", err)
	}
}
