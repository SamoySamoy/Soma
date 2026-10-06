// Package authz decides what the acting user may do in a space. Services call
// Require before every read or write; row-level security is the second layer
// (ADR-008).
package authz

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/SamoySamoy/Soma/internal/core/store"
	"github.com/SamoySamoy/Soma/internal/platform/apperr"
	"github.com/SamoySamoy/Soma/internal/platform/db"
)

// Permission names an action on a module, matching x-soma-permission in the
// API contract.
type Permission string

// Permissions. Add one per module action as modules arrive.
const (
	PeopleRead  Permission = "people:read"
	PeopleWrite Permission = "people:write"
)

// Role is a member's role in a space.
type Role string

// Roles, matching the space_members.role check constraint.
const (
	RoleOwner  Role = "owner"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

var grants = map[Role]map[Permission]bool{
	RoleOwner:  {PeopleRead: true, PeopleWrite: true},
	RoleEditor: {PeopleRead: true, PeopleWrite: true},
	RoleViewer: {PeopleRead: true},
}

// Allows reports whether role grants perm. Unknown roles grant nothing.
func Allows(role Role, perm Permission) bool {
	return grants[role][perm]
}

// Require checks that the acting user (from ctx) may use perm in spaceID.
// A user who isn't a member gets NotFound, so the space's existence isn't
// revealed. A member without the permission gets Forbidden.
func Require(ctx context.Context, tx pgx.Tx, spaceID uuid.UUID, perm Permission) error {
	userID, ok := db.UserIDFrom(ctx)
	if !ok {
		return apperr.Unauthorized("auth.required", "Sign in to continue.")
	}
	raw, err := store.New(tx).GetMemberRole(ctx, store.GetMemberRoleParams{
		SpaceID: spaceID, UserID: userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return apperr.NotFound("space.not_found", "Space not found.")
	}
	if err != nil {
		return fmt.Errorf("load membership: %w", err)
	}
	if !Allows(Role(raw), perm) {
		return apperr.Forbidden("authz.forbidden", "Your role in this space doesn't allow this.")
	}
	return nil
}
