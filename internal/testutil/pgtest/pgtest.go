//go:build integration

// Package pgtest gives integration tests a real, migrated PostgreSQL 18
// database. One container starts per test binary; each test gets a fresh
// database cloned from a migrated template, so tests never share state.
package pgtest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/SamoySamoy/Soma/internal/platform/db"
	"github.com/SamoySamoy/Soma/migrations"
)

const (
	image        = "postgres:18"
	ownerUser    = "soma"
	ownerPass    = "soma"
	appUser      = "soma_app"
	appPass      = "soma_app"
	templateName = "soma_template"
)

// Database is one test's private database.
type Database struct {
	// OwnerURL connects as the schema owner (migrations, test setup).
	OwnerURL string
	// AppURL connects as soma_app, subject to row-level security.
	AppURL string
}

var (
	once     sync.Once
	baseURL  *url.URL
	startErr error
)

// New returns a fresh migrated database, dropped when the test ends.
func New(t *testing.T) Database {
	t.Helper()
	ctx := t.Context()

	once.Do(func() { startErr = start(context.Background()) })
	if startErr != nil {
		t.Fatalf("start postgres: %v", startErr)
	}

	name := "t_" + randomSuffix(t)
	admin := connect(ctx, t, withDB(baseURL, ownerUser, ownerPass, "postgres"))
	if _, err := admin.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %s TEMPLATE %s OWNER %s`,
		pgx.Identifier{name}.Sanitize(), templateName, ownerUser)); err != nil {
		t.Fatalf("create test database: %v", err)
	}
	t.Cleanup(func() {
		// t.Context is already canceled when cleanups run.
		ctx := context.WithoutCancel(ctx)
		admin := connectNoCleanup(ctx, withDB(baseURL, ownerUser, ownerPass, "postgres"))
		if admin == nil {
			return
		}
		defer func() { _ = admin.Close(ctx) }()
		_, _ = admin.Exec(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS %s WITH (FORCE)`, pgx.Identifier{name}.Sanitize()))
	})

	return Database{
		OwnerURL: withDB(baseURL, ownerUser, ownerPass, name),
		AppURL:   withDB(baseURL, appUser, appPass, name),
	}
}

func start(ctx context.Context) error {
	ctr, err := postgres.Run(ctx, image,
		postgres.WithDatabase("postgres"),
		postgres.WithUsername(ownerUser),
		postgres.WithPassword(ownerPass),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return fmt.Errorf("run container: %w", err)
	}
	connStr, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return fmt.Errorf("connection string: %w", err)
	}
	baseURL, err = url.Parse(connStr)
	if err != nil {
		return fmt.Errorf("parse connection string: %w", err)
	}

	admin := connectNoCleanup(ctx, withDB(baseURL, ownerUser, ownerPass, "postgres"))
	if admin == nil {
		return fmt.Errorf("connect as owner")
	}
	defer func() { _ = admin.Close(ctx) }()

	// Mirrors deploy/postgres/init: an application role that can log in but
	// can't bypass row-level security or own objects.
	stmts := []string{
		fmt.Sprintf(`CREATE ROLE %s LOGIN PASSWORD '%s' NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS`, appUser, appPass),
		fmt.Sprintf(`CREATE DATABASE %s OWNER %s`, templateName, ownerUser),
	}
	for _, s := range stmts {
		if _, err := admin.Exec(ctx, s); err != nil {
			return fmt.Errorf("bootstrap %q: %w", s, err)
		}
	}

	m, err := db.NewMigrator(withDB(baseURL, ownerUser, ownerPass, templateName), migrations.FS())
	if err != nil {
		return err
	}
	if err := m.Up(ctx, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		_ = m.Close()
		return err
	}
	// The template must have no open connections to be cloned.
	return m.Close()
}

func withDB(base *url.URL, user, pass, name string) string {
	u := *base
	u.User = url.UserPassword(user, pass)
	u.Path = "/" + name
	return u.String()
}

func connect(ctx context.Context, t *testing.T, connURL string) *pgx.Conn {
	t.Helper()
	conn, err := pgx.Connect(ctx, connURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(context.WithoutCancel(ctx)) })
	return conn
}

func connectNoCleanup(ctx context.Context, connURL string) *pgx.Conn {
	conn, err := pgx.Connect(ctx, connURL)
	if err != nil {
		return nil
	}
	return conn
}

func randomSuffix(t *testing.T) string {
	t.Helper()
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("random: %v", err)
	}
	return hex.EncodeToString(b)
}
