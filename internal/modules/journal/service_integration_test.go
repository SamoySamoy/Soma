//go:build integration

package journal_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	corestore "github.com/SamoySamoy/Soma/internal/core/store"
	"github.com/SamoySamoy/Soma/internal/modules/journal"
	jstore "github.com/SamoySamoy/Soma/internal/modules/journal/store"
	"github.com/SamoySamoy/Soma/internal/platform/apperr"
	"github.com/SamoySamoy/Soma/internal/platform/clock"
	"github.com/SamoySamoy/Soma/internal/platform/db"
	"github.com/SamoySamoy/Soma/internal/space"
	"github.com/SamoySamoy/Soma/internal/testutil/pgtest"
)

type env struct {
	app   *db.DB
	owner *db.DB
	svc   *journal.Service
	clk   *clock.Fake
}

func newEnv(t *testing.T) env {
	t.Helper()
	dbs := pgtest.New(t)
	app := openDB(t, dbs.AppURL)
	owner := openDB(t, dbs.OwnerURL)
	clk := clock.NewFake(time.Date(2026, 10, 6, 9, 0, 0, 0, time.UTC))
	if err := space.EnsureLocal(t.Context(), app, clk); err != nil {
		t.Fatalf("EnsureLocal: %v", err)
	}
	return env{app: app, owner: owner, svc: journal.NewService(app, clk), clk: clk}
}

func openDB(t *testing.T, url string) *db.DB {
	t.Helper()
	d, err := db.Open(t.Context(), url)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(d.Close)
	return d
}

func as(ctx context.Context, user uuid.UUID) context.Context {
	return space.WithSpace(db.WithUserID(ctx, user), space.LocalSpaceID)
}

func (e env) addMember(t *testing.T, user uuid.UUID, role string) {
	t.Helper()
	ctx := t.Context()
	now := e.clk.Now()
	err := e.owner.InTx(ctx, func(tx pgx.Tx) error {
		q := corestore.New(tx)
		if err := q.UpsertUser(ctx, corestore.UpsertUserParams{ID: user, DisplayName: "Member", Now: now}); err != nil {
			return err
		}
		return q.InsertSpaceMember(ctx, corestore.InsertSpaceMemberParams{
			SpaceID: space.LocalSpaceID, UserID: user, Role: role, Now: now,
		})
	})
	if err != nil {
		t.Fatalf("add member: %v", err)
	}
}

func input(body string) journal.Input {
	return journal.Input{EntryDate: "2026-10-06", Body: body, Mood: 4}
}

func TestEntryLifecycle(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := as(t.Context(), space.LocalOwnerID)

	created, err := e.svc.Create(ctx, input("Went for a long walk."))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Version != 1 || created.Mood != 4 || created.EntryDate != "2026-10-06" {
		t.Fatalf("Create() = %+v, want version 1, mood 4, date kept", created)
	}

	got, err := e.svc.Get(ctx, created.ID)
	if err != nil || got.Body != "Went for a long walk." {
		t.Fatalf("Get() = %+v, %v", got, err)
	}

	updated, err := e.svc.Update(ctx, created.ID, 1, journal.Patch{"mood": 2.0, "title": "Slow day"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Version != 2 || updated.Mood != 2 || updated.Title != "Slow day" || updated.Body != created.Body {
		t.Fatalf("Update() = %+v, want version 2, mood 2, title set, body kept", updated)
	}

	if _, err := e.svc.Update(ctx, created.ID, 1, journal.Patch{"mood": 5.0}); kindOf(err) != apperr.KindPreconditionFailed {
		t.Errorf("stale update: kind = %v, want PreconditionFailed (err %v)", kindOf(err), err)
	}

	if err := e.svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := e.svc.Get(ctx, created.ID); kindOf(err) != apperr.KindNotFound {
		t.Errorf("Get after delete: kind = %v, want NotFound", kindOf(err))
	}
}

// TestEntriesAreVisibleOnlyToTheirAuthor is the privacy rule from the business
// spec: another member of the same space never sees an entry, even by ID.
func TestEntriesAreVisibleOnlyToTheirAuthor(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	owner := as(t.Context(), space.LocalOwnerID)
	entry, err := e.svc.Create(owner, input("Private."))
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	editor := uuid.New()
	e.addMember(t, editor, "editor")
	ctx := as(t.Context(), editor)

	page, err := e.svc.List(ctx, 50, nil)
	if err != nil || len(page.Items) != 0 {
		t.Errorf("editor List = %d items, %v; want none", len(page.Items), err)
	}
	if _, err := e.svc.Get(ctx, entry.ID); kindOf(err) != apperr.KindNotFound {
		t.Errorf("editor Get of another's entry: kind = %v, want NotFound", kindOf(err))
	}
	if _, err := e.svc.Update(ctx, entry.ID, entry.Version, journal.Patch{"body": "changed"}); kindOf(err) != apperr.KindNotFound {
		t.Errorf("editor Update of another's entry: kind = %v, want NotFound", kindOf(err))
	}
}

// TestDatabaseHidesOtherAuthorsEntries checks the same rule in row-level
// security, without the service's checks.
func TestDatabaseHidesOtherAuthorsEntries(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	if _, err := e.svc.Create(as(t.Context(), space.LocalOwnerID), input("Private.")); err != nil {
		t.Fatalf("seed: %v", err)
	}

	editor := uuid.New()
	e.addMember(t, editor, "editor")
	err := e.app.InReadOnlyTx(db.WithUserID(t.Context(), editor), func(tx pgx.Tx) error {
		rows, err := jstore.New(tx).ListEntries(t.Context(), jstore.ListEntriesParams{
			SpaceID: space.LocalSpaceID, AuthorID: space.LocalOwnerID, RowLimit: 10,
		})
		if err != nil {
			return err
		}
		if len(rows) != 0 {
			t.Errorf("editor reads %d entries of the owner through the database, want 0", len(rows))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
}

func TestListPagesNewestFirst(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := as(t.Context(), space.LocalOwnerID)
	var ids []uuid.UUID
	for i := range 5 {
		entry, err := e.svc.Create(ctx, input("entry "+string(rune('a'+i))))
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		ids = append(ids, entry.ID)
	}
	var seen []uuid.UUID
	var cursor *uuid.UUID
	for range 5 {
		page, err := e.svc.List(ctx, 2, cursor)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		for _, it := range page.Items {
			seen = append(seen, it.ID)
		}
		if page.Next == nil {
			break
		}
		cursor = page.Next
	}
	if len(seen) != 5 {
		t.Fatalf("paged through %d entries, want 5", len(seen))
	}
	for i := range seen {
		if seen[i] != ids[len(ids)-1-i] {
			t.Errorf("position %d = %v, want newest-first", i, seen[i])
		}
	}
}

func kindOf(err error) apperr.Kind { return apperr.KindOf(err) }
