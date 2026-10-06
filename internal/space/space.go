// Package space owns spaces (the containers that hold records) and the
// identity stand-in used until authentication exists (ADR-014).
package space

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/SamoySamoy/Soma/internal/core/store"
	"github.com/SamoySamoy/Soma/internal/platform/clock"
	"github.com/SamoySamoy/Soma/internal/platform/db"
)

// Fixed identifiers for the implicit local owner and their personal space.
// They are stable so the database survives restarts, and they are used only
// in local mode.
var (
	LocalOwnerID = uuid.MustParse("00000000-0000-7000-8000-000000000001")
	LocalSpaceID = uuid.MustParse("00000000-0000-7000-8000-000000000002")
)

type spaceKey struct{}

// WithSpace returns a copy of ctx that targets the space with id.
func WithSpace(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, spaceKey{}, id)
}

// FromContext returns the space targeted by ctx.
func FromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(spaceKey{}).(uuid.UUID)
	return id, ok && id != uuid.Nil
}

// EnsureLocal creates the implicit local owner, their personal space and the
// membership between them. It does nothing when they already exist, so it
// runs safely at every start.
func EnsureLocal(ctx context.Context, d *db.DB, clk clock.Clock) error {
	ctx = db.WithUserID(ctx, LocalOwnerID)
	now := clk.Now()
	err := d.InTx(ctx, func(tx pgx.Tx) error {
		q := store.New(tx)
		_, err := q.GetMemberRole(ctx, store.GetMemberRoleParams{SpaceID: LocalSpaceID, UserID: LocalOwnerID})
		if err == nil {
			return nil // already set up
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("check owner membership: %w", err)
		}
		if err := q.UpsertUser(ctx, store.UpsertUserParams{
			ID: LocalOwnerID, DisplayName: "You", Now: now,
		}); err != nil {
			return fmt.Errorf("upsert local owner: %w", err)
		}
		if err := q.InsertSpace(ctx, store.InsertSpaceParams{
			ID: LocalSpaceID, Name: "Personal", CreatedBy: LocalOwnerID, Now: now,
		}); err != nil {
			return fmt.Errorf("insert personal space: %w", err)
		}
		if err := q.InsertSpaceMember(ctx, store.InsertSpaceMemberParams{
			SpaceID: LocalSpaceID, UserID: LocalOwnerID, Role: "owner", Now: now,
		}); err != nil {
			return fmt.Errorf("insert owner membership: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("ensure local owner: %w", err)
	}
	return nil
}
