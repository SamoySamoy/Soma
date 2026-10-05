//go:build integration

package db_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/SamoySamoy/Soma/internal/platform/db"
	"github.com/SamoySamoy/Soma/internal/testutil/pgtest"
	"github.com/SamoySamoy/Soma/migrations"
)

func open(t *testing.T, url string) *db.DB {
	t.Helper()
	d, err := db.Open(t.Context(), url)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(d.Close)
	return d
}

func TestInTxSetsActingUser(t *testing.T) {
	t.Parallel()
	d := open(t, pgtest.New(t).AppURL)

	id := uuid.New()
	tests := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{"with user", db.WithUserID(t.Context(), id), id.String()},
		{"without user", t.Context(), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			err := d.InTx(tt.ctx, func(tx pgx.Tx) error {
				return tx.QueryRow(tt.ctx, `SELECT current_setting('soma.user_id', true)`).Scan(&got)
			})
			if err != nil {
				t.Fatalf("InTx: %v", err)
			}
			if got != tt.want {
				t.Errorf("soma.user_id = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInTxRollsBackOnError(t *testing.T) {
	t.Parallel()
	dbs := pgtest.New(t)
	owner := open(t, dbs.OwnerURL)
	ctx := t.Context()

	if err := owner.InTx(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `CREATE TABLE notes (body text NOT NULL)`)
		return err
	}); err != nil {
		t.Fatalf("create table: %v", err)
	}

	errBoom := errors.New("boom")
	err := owner.InTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO notes VALUES ('kept?')`); err != nil {
			return err
		}
		return errBoom
	})
	if !errors.Is(err, errBoom) {
		t.Fatalf("InTx error = %v, want %v", err, errBoom)
	}

	var n int
	if err := owner.Pool().QueryRow(ctx, `SELECT count(*) FROM notes`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("rows after rollback = %d, want 0", n)
	}
}

func TestInReadOnlyTxRejectsWrites(t *testing.T) {
	t.Parallel()
	d := open(t, pgtest.New(t).OwnerURL)
	ctx := t.Context()

	err := d.InReadOnlyTx(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `CREATE TABLE should_fail (id int)`)
		return err
	})
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "25006" { // read_only_sql_transaction
		t.Fatalf("error = %v, want read-only transaction error 25006", err)
	}
}

func TestAppRoleCannotBypassRLSOrCreateTables(t *testing.T) {
	t.Parallel()
	d := open(t, pgtest.New(t).AppURL)
	ctx := t.Context()

	var bypass bool
	if err := d.Pool().QueryRow(ctx, `SELECT rolbypassrls FROM pg_roles WHERE rolname = current_user`).Scan(&bypass); err != nil {
		t.Fatalf("query role: %v", err)
	}
	if bypass {
		t.Error("soma_app can bypass row-level security")
	}

	_, err := d.Pool().Exec(ctx, `CREATE TABLE sneaky (id int)`)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "42501" { // insufficient_privilege
		t.Errorf("CREATE TABLE as soma_app: error = %v, want insufficient privilege", err)
	}
}

func TestSchemaVersionMatchesLatestMigration(t *testing.T) {
	t.Parallel()
	d := open(t, pgtest.New(t).AppURL)

	got, err := d.SchemaVersion(t.Context())
	if err != nil {
		t.Fatalf("SchemaVersion: %v", err)
	}
	want, err := migrations.Latest()
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if got != want {
		t.Errorf("SchemaVersion() = %d, want %d", got, want)
	}
}

func TestMigrationsRollBackAndReapply(t *testing.T) {
	t.Parallel()
	dbs := pgtest.New(t)
	ctx := t.Context()

	m, err := db.NewMigrator(dbs.OwnerURL, migrations.FS())
	if err != nil {
		t.Fatalf("NewMigrator: %v", err)
	}
	t.Cleanup(func() { _ = m.Close() })

	if err := m.DownTo(ctx, 0); err != nil {
		t.Fatalf("DownTo(0): %v", err)
	}
	if err := m.Up(ctx, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("Up after down: %v", err)
	}
}
