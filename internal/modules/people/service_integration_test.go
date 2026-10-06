//go:build integration

package people_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	corestore "github.com/SamoySamoy/Soma/internal/core/store"
	"github.com/SamoySamoy/Soma/internal/modules/people"
	pstore "github.com/SamoySamoy/Soma/internal/modules/people/store"
	"github.com/SamoySamoy/Soma/internal/platform/apperr"
	"github.com/SamoySamoy/Soma/internal/platform/clock"
	"github.com/SamoySamoy/Soma/internal/platform/db"
	"github.com/SamoySamoy/Soma/internal/space"
	"github.com/SamoySamoy/Soma/internal/testutil/pgtest"
)

// env is one test's database. app connects as soma_app (RLS applies); owner
// connects as the schema owner, used only to seed members.
type env struct {
	app   *db.DB
	owner *db.DB
	svc   *people.Service
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
	return env{app: app, owner: owner, svc: people.NewService(app, clk), clk: clk}
}

func openDB(t *testing.T, url string) *db.DB {
	t.Helper()
	d, err := db.Open(t.Context(), url)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(d.Close)
	return d
}

// as returns a context acting as user in the local personal space.
func as(ctx context.Context, user uuid.UUID) context.Context {
	return space.WithSpace(db.WithUserID(ctx, user), space.LocalSpaceID)
}

// addMember makes user a member of the local space with role.
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
		t.Fatalf("add %s member: %v", role, err)
	}
}

func kindOf(err error) apperr.Kind {
	return apperr.KindOf(err)
}

func sampleInput(name string) people.Input {
	return people.Input{DisplayName: name, Email: "lan@example.com", Birthday: "1990-04-23"}
}

func TestContactLifecycle(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := as(t.Context(), space.LocalOwnerID)

	created, err := e.svc.Create(ctx, sampleInput("  Lan Nguyen "))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.DisplayName != "Lan Nguyen" || created.Version != 1 || created.Birthday != "1990-04-23" {
		t.Fatalf("Create() = %+v, want trimmed name, version 1, birthday kept", created)
	}

	got, err := e.svc.Get(ctx, created.ID)
	if err != nil || got.ID != created.ID || got.Email != "lan@example.com" {
		t.Fatalf("Get() = %+v, %v; want the created contact", got, err)
	}

	phone := "+84 90 000 0000"
	updated, err := e.svc.Update(ctx, created.ID, 1, people.Patch{"phone": &phone, "email": nil})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Version != 2 || updated.Phone != phone || updated.Email != "" || updated.DisplayName != "Lan Nguyen" {
		t.Fatalf("Update() = %+v, want version 2, phone set, email cleared, name kept", updated)
	}

	if _, err := e.svc.Update(ctx, created.ID, 1, people.Patch{"notes": nil}); kindOf(err) != apperr.KindPreconditionFailed {
		t.Errorf("Update with stale version: kind = %v, want PreconditionFailed (err %v)", kindOf(err), err)
	}

	if err := e.svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := e.svc.Get(ctx, created.ID); kindOf(err) != apperr.KindNotFound {
		t.Errorf("Get after delete: kind = %v, want NotFound", kindOf(err))
	}
	page, err := e.svc.List(ctx, 50, nil)
	if err != nil || len(page.Items) != 0 {
		t.Errorf("List after delete = %d items, %v; want none", len(page.Items), err)
	}
}

func TestListPaginatesNewestFirst(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := as(t.Context(), space.LocalOwnerID)

	created := make([]uuid.UUID, 0, 5)
	for _, name := range []string{"A", "B", "C", "D", "E"} {
		c, err := e.svc.Create(ctx, sampleInput(name))
		if err != nil {
			t.Fatalf("Create(%s): %v", name, err)
		}
		created = append(created, c.ID)
	}

	var seen []uuid.UUID
	var cursor *uuid.UUID
	for pages := 0; ; pages++ {
		if pages > 5 {
			t.Fatal("pagination did not terminate")
		}
		page, err := e.svc.List(ctx, 2, cursor)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		for _, c := range page.Items {
			seen = append(seen, c.ID)
		}
		if page.Next == nil {
			break
		}
		cursor = page.Next
	}

	if len(seen) != 5 {
		t.Fatalf("paged through %d contacts, want 5", len(seen))
	}
	for i := range seen {
		if seen[i] != created[len(created)-1-i] {
			t.Errorf("position %d = %v, want newest-first %v", i, seen[i], created[len(created)-1-i])
		}
	}
}

// TestAuthorizationMatrix runs every contact operation as each role that can
// exist without sessions (ADR-014). The anonymous and wrong-scope cases are
// added with M1.
func TestAuthorizationMatrix(t *testing.T) {
	t.Parallel()

	editor := uuid.New()
	viewer := uuid.New()
	stranger := uuid.New()

	type actor struct {
		name string
		user uuid.UUID
		// writes and reads state what each operation must return: nil, or the
		// kind of error the caller must receive.
		writeKind apperr.Kind
		readKind  apperr.Kind
	}
	actors := []actor{
		{"owner", space.LocalOwnerID, -1, -1},
		{"editor", editor, -1, -1},
		{"viewer", viewer, apperr.KindForbidden, -1},
		{"user of another space", stranger, apperr.KindNotFound, apperr.KindNotFound},
	}
	const noError apperr.Kind = -1

	for _, a := range actors {
		t.Run(a.name, func(t *testing.T) {
			t.Parallel()
			e := newEnv(t)
			e.addMember(t, editor, "editor")
			e.addMember(t, viewer, "viewer")
			owner := as(t.Context(), space.LocalOwnerID)
			contact, err := e.svc.Create(owner, sampleInput("Lan"))
			if err != nil {
				t.Fatalf("seed contact: %v", err)
			}
			ctx := as(t.Context(), a.user)

			check := func(op string, err error, want apperr.Kind) {
				t.Helper()
				if want == noError {
					if err != nil {
						t.Errorf("%s: error = %v, want success", op, err)
					}
					return
				}
				if kindOf(err) != want {
					t.Errorf("%s: kind = %v (err %v), want %v", op, kindOf(err), err, want)
				}
			}

			_, err = e.svc.List(ctx, 50, nil)
			check("list", err, a.readKind)
			_, err = e.svc.Get(ctx, contact.ID)
			check("get", err, a.readKind)
			_, err = e.svc.Create(ctx, sampleInput("New"))
			check("create", err, a.writeKind)
			_, err = e.svc.Update(ctx, contact.ID, contact.Version, people.Patch{"notes": nil})
			check("update", err, a.writeKind)
			err = e.svc.Delete(ctx, contact.ID)
			check("delete", err, a.writeKind)
		})
	}
}

// TestDatabaseRefusesWritesFromViewers checks row-level security directly,
// without the service's authorization check, so a forgotten check still can't
// write (ADR-008).
func TestDatabaseRefusesWritesFromViewers(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	viewer := uuid.New()
	e.addMember(t, viewer, "viewer")
	ctx := db.WithUserID(t.Context(), viewer)

	err := e.app.InTx(ctx, func(tx pgx.Tx) error {
		_, err := corestore.New(tx).InsertEntity(ctx, corestore.InsertEntityParams{
			SpaceID: space.LocalSpaceID, Kind: "people.contact", CreatedBy: viewer, Now: e.clk.Now(),
		})
		return err
	})
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
		t.Fatalf("viewer insert into entities: error = %v, want insufficient privilege (42501)", err)
	}
}

// TestDatabaseHidesSpacesFromStrangers checks that rows of another space are
// invisible to a user who isn't a member, even when the query names the space.
func TestDatabaseHidesSpacesFromStrangers(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	if _, err := e.svc.Create(as(t.Context(), space.LocalOwnerID), sampleInput("Lan")); err != nil {
		t.Fatalf("seed: %v", err)
	}

	stranger := uuid.New() // never made a member of any space

	err := e.app.InReadOnlyTx(db.WithUserID(t.Context(), stranger), func(tx pgx.Tx) error {
		rows, err := pstore.New(tx).ListContacts(t.Context(), pstore.ListContactsParams{
			SpaceID: space.LocalSpaceID, RowLimit: 10,
		})
		if err != nil {
			return err
		}
		if len(rows) != 0 {
			t.Errorf("stranger sees %d contacts, want 0", len(rows))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("query as stranger: %v", err)
	}
}
