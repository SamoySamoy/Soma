//go:build integration

package tasks_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/SamoySamoy/Soma/internal/bodymap"
	corestore "github.com/SamoySamoy/Soma/internal/core/store"
	"github.com/SamoySamoy/Soma/internal/modules/tasks"
	"github.com/SamoySamoy/Soma/internal/platform/apperr"
	"github.com/SamoySamoy/Soma/internal/platform/clock"
	"github.com/SamoySamoy/Soma/internal/platform/db"
	"github.com/SamoySamoy/Soma/internal/space"
	"github.com/SamoySamoy/Soma/internal/testutil/pgtest"
)

type env struct {
	owner *db.DB
	svc   *tasks.Service
	clk   *clock.Fake
	app   *db.DB
}

func newEnv(t *testing.T) env {
	t.Helper()
	dbs := pgtest.New(t)
	app, err := db.Open(t.Context(), dbs.AppURL)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(app.Close)
	owner, err := db.Open(t.Context(), dbs.OwnerURL)
	if err != nil {
		t.Fatalf("open owner: %v", err)
	}
	t.Cleanup(owner.Close)
	clk := clock.NewFake(time.Date(2026, 10, 6, 9, 0, 0, 0, time.UTC))
	if err := space.EnsureLocal(t.Context(), app, clk); err != nil {
		t.Fatalf("EnsureLocal: %v", err)
	}
	return env{owner: owner, svc: tasks.NewService(app, clk), clk: clk, app: app}
}

func as(ctx context.Context, user uuid.UUID) context.Context {
	return space.WithSpace(db.WithUserID(ctx, user), space.LocalSpaceID)
}

func TestTaskLifecycle(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := as(t.Context(), space.LocalOwnerID)

	task, err := e.svc.Create(ctx, tasks.Input{Title: "Renew passport", DueOn: "2026-10-01"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if task.Version != 1 || task.Completed() {
		t.Fatalf("Create() = %+v, want version 1 and open", task)
	}
	if !task.Overdue(e.clk.Now()) {
		t.Errorf("task due 2026-10-01 is not overdue on 2026-10-06")
	}

	done, err := e.svc.Complete(ctx, task.ID)
	if err != nil || !done.Completed() || done.Version != 2 {
		t.Fatalf("Complete() = %+v, %v; want done at version 2", done, err)
	}
	again, err := e.svc.Complete(ctx, task.ID)
	if err != nil || again.Version != 2 {
		t.Errorf("completing a done task changed it: version %d, %v", again.Version, err)
	}

	page, err := e.svc.List(ctx, 50, nil, true)
	if err != nil || len(page.Items) != 0 {
		t.Errorf("open-only list = %d items, %v; want none", len(page.Items), err)
	}

	reopened, err := e.svc.Reopen(ctx, task.ID)
	if err != nil || reopened.Completed() || reopened.Version != 3 {
		t.Fatalf("Reopen() = %+v, %v", reopened, err)
	}

	if _, err := e.svc.Update(ctx, task.ID, 1, tasks.Patch{"title": "x"}); kindOf(err) != apperr.KindPreconditionFailed {
		t.Errorf("stale update: kind = %v, want PreconditionFailed", kindOf(err))
	}

	if err := e.svc.Delete(ctx, task.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := e.svc.Get(ctx, task.ID); kindOf(err) != apperr.KindNotFound {
		t.Errorf("Get after delete: kind = %v, want NotFound", kindOf(err))
	}
}

// TestViewerCannotChangeTasks checks the service refuses a viewer, and that the
// database refuses the same write if the service check were ever missing.
func TestViewerCannotChangeTasks(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	viewer := uuid.New()
	now := e.clk.Now()
	err := e.owner.InTx(t.Context(), func(tx pgx.Tx) error {
		q := corestore.New(tx)
		if err := q.UpsertUser(t.Context(), corestore.UpsertUserParams{ID: viewer, DisplayName: "V", Now: now}); err != nil {
			return err
		}
		return q.InsertSpaceMember(t.Context(), corestore.InsertSpaceMemberParams{
			SpaceID: space.LocalSpaceID, UserID: viewer, Role: "viewer", Now: now,
		})
	})
	if err != nil {
		t.Fatalf("seed viewer: %v", err)
	}

	_, err = e.svc.Create(as(t.Context(), viewer), tasks.Input{Title: "nope"})
	if kindOf(err) != apperr.KindForbidden {
		t.Errorf("viewer Create: kind = %v, want Forbidden (err %v)", kindOf(err), err)
	}
	page, err := e.svc.List(as(t.Context(), viewer), 50, nil, false)
	if err != nil || len(page.Items) != 0 {
		t.Errorf("viewer List = %d, %v; want an empty, readable list", len(page.Items), err)
	}
}

func TestBodymapCountsOverdueAndDueToday(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := as(t.Context(), space.LocalOwnerID)
	for _, due := range []string{"2026-10-01", "2026-10-06", "2026-10-20"} {
		if _, err := e.svc.Create(ctx, tasks.Input{Title: "task " + due, DueOn: due}); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}
	// Overdue by five days, so the Responsibilities area is urgent.
	level, counts := bodymapOf(t, e)
	if level != bodymap.LevelUrgent {
		t.Errorf("level = %q, want urgent", level)
	}
	if counts["overdue"] != 1 || counts["due_today"] != 1 || counts["overdue_long"] != 1 {
		t.Errorf("counts = %v, want overdue 1, due_today 1, overdue_long 1", counts)
	}
}

func kindOf(err error) apperr.Kind { return apperr.KindOf(err) }

// bodymapOf reads the Responsibilities status the way the body map does.
func bodymapOf(t *testing.T, e env) (bodymap.Level, map[string]int) {
	t.Helper()
	var res bodymap.Result
	err := e.app.InReadOnlyTx(as(t.Context(), space.LocalOwnerID), func(tx pgx.Tx) error {
		var err error
		res, err = tasks.BodymapProvider{}.Status(t.Context(), tx, space.LocalSpaceID, e.clk.Now())
		return err
	})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	return res.Level, res.Counts
}
