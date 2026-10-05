// Package db owns the PostgreSQL connection pool and transaction helpers.
// Every transaction carries the acting user's ID in the soma.user_id setting,
// which row-level security policies read.
package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps a pgx connection pool.
type DB struct {
	pool *pgxpool.Pool
}

// Open connects to url and verifies the connection.
func Open(ctx context.Context, url string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &DB{pool: pool}, nil
}

// Close closes all connections.
func (d *DB) Close() { d.pool.Close() }

// Ping checks that the database is reachable.
func (d *DB) Ping(ctx context.Context) error { return d.pool.Ping(ctx) }

// Pool exposes the underlying pool for libraries that need it (job queue).
// Application code must use InTx instead.
func (d *DB) Pool() *pgxpool.Pool { return d.pool }

// SchemaVersion returns the newest applied migration version.
func (d *DB) SchemaVersion(ctx context.Context) (int64, error) {
	var v int64
	err := d.pool.QueryRow(ctx,
		`SELECT coalesce(max(version_id), 0) FROM goose_db_version WHERE is_applied`).Scan(&v)
	if err != nil {
		return 0, fmt.Errorf("query schema version: %w", err)
	}
	return v, nil
}

type userKey struct{}

// WithUserID returns a copy of ctx whose transactions act as user id.
func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userKey{}, id)
}

// UserIDFrom returns the acting user set by WithUserID.
func UserIDFrom(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userKey{}).(uuid.UUID)
	return id, ok && id != uuid.Nil
}

// InTx runs fn in a read-write transaction. It commits when fn returns nil and
// rolls back otherwise, including when fn panics.
func (d *DB) InTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	return d.inTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadWrite}, fn)
}

// InReadOnlyTx runs fn in a read-only transaction. Any write fails.
func (d *DB) InReadOnlyTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	return d.inTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly}, fn)
}

func (d *DB) inTx(ctx context.Context, opts pgx.TxOptions, fn func(tx pgx.Tx) error) (err error) {
	tx, err := d.pool.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(context.WithoutCancel(ctx))
			panic(p)
		}
		if err != nil {
			if rbErr := tx.Rollback(context.WithoutCancel(ctx)); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
				err = errors.Join(err, fmt.Errorf("rollback: %w", rbErr))
			}
		}
	}()

	// An empty setting makes RLS policies match no rows, so a request without
	// a user can't read user data even if a policy check is forgotten.
	userID := ""
	if id, ok := UserIDFrom(ctx); ok {
		userID = id.String()
	}
	if _, err = tx.Exec(ctx, `SELECT set_config('soma.user_id', $1, true)`, userID); err != nil {
		return fmt.Errorf("set acting user: %w", err)
	}

	if err = fn(tx); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
