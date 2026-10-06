package people

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/SamoySamoy/Soma/internal/authz"
	corestore "github.com/SamoySamoy/Soma/internal/core/store"
	pstore "github.com/SamoySamoy/Soma/internal/modules/people/store"
	"github.com/SamoySamoy/Soma/internal/platform/apperr"
	"github.com/SamoySamoy/Soma/internal/platform/clock"
	"github.com/SamoySamoy/Soma/internal/platform/db"
	"github.com/SamoySamoy/Soma/internal/space"
)

// kindContact is the entity kind for people contacts (entities.kind).
const kindContact = "people.contact"

// Service implements contact use cases. Each method runs in one transaction,
// checks authorization before touching data, and works only inside the space
// on the context.
type Service struct {
	db    *db.DB
	clock clock.Clock
}

// NewService returns a Service backed by d.
func NewService(d *db.DB, clk clock.Clock) *Service {
	return &Service{db: d, clock: clk}
}

// List returns up to limit contacts, newest first, after the before cursor.
func (s *Service) List(ctx context.Context, limit int32, before *uuid.UUID) (Page, error) {
	spaceID, err := spaceFrom(ctx)
	if err != nil {
		return Page{}, err
	}
	var page Page
	err = s.db.InReadOnlyTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.PeopleRead); err != nil {
			return err
		}
		rows, err := pstore.New(tx).ListContacts(ctx, pstore.ListContactsParams{
			SpaceID:  spaceID,
			Before:   before,
			RowLimit: limit + 1, // one extra row tells us whether another page exists
		})
		if err != nil {
			return fmt.Errorf("list contacts: %w", err)
		}
		if len(rows) > int(limit) {
			rows = rows[:limit]
			next := rows[len(rows)-1].ID
			page.Next = &next
		}
		page.Items = make([]Contact, 0, len(rows))
		for _, r := range rows {
			page.Items = append(page.Items, fromListRow(r))
		}
		return nil
	})
	return page, err
}

// Get returns one contact.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (Contact, error) {
	spaceID, err := spaceFrom(ctx)
	if err != nil {
		return Contact{}, err
	}
	var out Contact
	err = s.db.InReadOnlyTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.PeopleRead); err != nil {
			return err
		}
		out, err = loadContact(ctx, tx, spaceID, id)
		return err
	})
	return out, err
}

// Create adds a contact to the space.
func (s *Service) Create(ctx context.Context, in Input) (Contact, error) {
	in = in.normalized()
	if err := in.validate(); err != nil {
		return Contact{}, err
	}
	spaceID, err := spaceFrom(ctx)
	if err != nil {
		return Contact{}, err
	}
	userID, ok := db.UserIDFrom(ctx)
	if !ok {
		return Contact{}, apperr.Unauthorized("auth.required", "Sign in to continue.")
	}

	var out Contact
	err = s.db.InTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.PeopleWrite); err != nil {
			return err
		}
		ent, err := corestore.New(tx).InsertEntity(ctx, corestore.InsertEntityParams{
			SpaceID: spaceID, Kind: kindContact, CreatedBy: userID, Now: s.clock.Now(),
		})
		if err != nil {
			return fmt.Errorf("insert entity: %w", err)
		}
		if err := insertContact(ctx, tx, spaceID, ent.ID, in); err != nil {
			return err
		}
		out, err = loadContact(ctx, tx, spaceID, ent.ID)
		return err
	})
	return out, err
}

// Update applies patch to a contact. expectedVersion must equal the stored
// version (from If-Match); if it doesn't, the update is refused with 412.
func (s *Service) Update(ctx context.Context, id uuid.UUID, expectedVersion int32, patch Patch) (Contact, error) {
	spaceID, err := spaceFrom(ctx)
	if err != nil {
		return Contact{}, err
	}
	var out Contact
	err = s.db.InTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.PeopleWrite); err != nil {
			return err
		}
		cur, err := loadContact(ctx, tx, spaceID, id)
		if err != nil {
			return err
		}
		if cur.Version != expectedVersion {
			return staleVersion()
		}
		next, err := patch.apply(cur.input())
		if err != nil {
			return err
		}
		next = next.normalized()
		if err := next.validate(); err != nil {
			return err
		}
		if err := writeContact(ctx, tx, spaceID, id, next); err != nil {
			return err
		}
		// The version bump is conditional too, so a concurrent edit that slips
		// in after the read above still gets 412 rather than being overwritten.
		if _, err := corestore.New(tx).BumpEntityVersion(ctx, corestore.BumpEntityVersionParams{
			ID: id, ExpectedVersion: expectedVersion, Now: s.clock.Now(),
		}); errors.Is(err, pgx.ErrNoRows) {
			return staleVersion()
		} else if err != nil {
			return fmt.Errorf("bump version: %w", err)
		}
		out, err = loadContact(ctx, tx, spaceID, id)
		return err
	})
	return out, err
}

// Delete moves a contact to the trash. The row is kept until the trash is purged.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	spaceID, err := spaceFrom(ctx)
	if err != nil {
		return err
	}
	return s.db.InTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.PeopleWrite); err != nil {
			return err
		}
		if _, err := loadContact(ctx, tx, spaceID, id); err != nil {
			return err
		}
		if _, err := corestore.New(tx).SoftDeleteEntity(ctx, corestore.SoftDeleteEntityParams{
			ID: id, Now: ptrTo(s.clock.Now()),
		}); err != nil {
			return fmt.Errorf("move contact to trash: %w", err)
		}
		return nil
	})
}

func spaceFrom(ctx context.Context) (uuid.UUID, error) {
	id, ok := space.FromContext(ctx)
	if !ok {
		return uuid.Nil, apperr.Unauthorized("space.required", "Choose a space to continue.")
	}
	return id, nil
}

func loadContact(ctx context.Context, tx pgx.Tx, spaceID, id uuid.UUID) (Contact, error) {
	r, err := pstore.New(tx).GetContact(ctx, pstore.GetContactParams{ID: id, SpaceID: spaceID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Contact{}, apperr.NotFound("people.contact_not_found", "Contact not found.")
	}
	if err != nil {
		return Contact{}, fmt.Errorf("load contact: %w", err)
	}
	return fromGetRow(r), nil
}

func insertContact(ctx context.Context, tx pgx.Tx, spaceID, id uuid.UUID, in Input) error {
	bday, err := birthdayOf(in.Birthday)
	if err != nil {
		return err
	}
	if err := pstore.New(tx).InsertContact(ctx, pstore.InsertContactParams{
		ID:          id,
		SpaceID:     spaceID,
		DisplayName: in.DisplayName,
		Nickname:    nullable(in.Nickname),
		Email:       nullable(in.Email),
		Phone:       nullable(in.Phone),
		Birthday:    bday,
		HowWeMet:    nullable(in.HowWeMet),
		Notes:       nullable(in.Notes),
	}); err != nil {
		return fmt.Errorf("insert contact: %w", err)
	}
	return nil
}

func writeContact(ctx context.Context, tx pgx.Tx, spaceID, id uuid.UUID, in Input) error {
	bday, err := birthdayOf(in.Birthday)
	if err != nil {
		return err
	}
	if err := pstore.New(tx).UpdateContact(ctx, pstore.UpdateContactParams{
		ID:          id,
		SpaceID:     spaceID,
		DisplayName: in.DisplayName,
		Nickname:    nullable(in.Nickname),
		Email:       nullable(in.Email),
		Phone:       nullable(in.Phone),
		Birthday:    bday,
		HowWeMet:    nullable(in.HowWeMet),
		Notes:       nullable(in.Notes),
	}); err != nil {
		return fmt.Errorf("update contact: %w", err)
	}
	return nil
}

func staleVersion() error {
	return apperr.New(apperr.KindPreconditionFailed, "people.stale_version",
		"This contact was changed elsewhere. Reload it and try again.")
}
