package journal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/SamoySamoy/Soma/internal/authz"
	corestore "github.com/SamoySamoy/Soma/internal/core/store"
	jstore "github.com/SamoySamoy/Soma/internal/modules/journal/store"
	"github.com/SamoySamoy/Soma/internal/platform/apperr"
	"github.com/SamoySamoy/Soma/internal/platform/clock"
	"github.com/SamoySamoy/Soma/internal/platform/db"
	"github.com/SamoySamoy/Soma/internal/space"
)

const kindEntry = "journal.entry"

// Service implements journal use cases. Every method works only on the acting
// user's own entries, inside the space on the context.
type Service struct {
	db    *db.DB
	clock clock.Clock
}

// NewService returns a Service backed by d.
func NewService(d *db.DB, clk clock.Clock) *Service {
	return &Service{db: d, clock: clk}
}

// List returns up to limit of the acting user's entries, newest saved first.
func (s *Service) List(ctx context.Context, limit int32, before *uuid.UUID) (Page, error) {
	spaceID, author, err := scope(ctx)
	if err != nil {
		return Page{}, err
	}
	var page Page
	err = s.db.InReadOnlyTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.JournalRead); err != nil {
			return err
		}
		rows, err := jstore.New(tx).ListEntries(ctx, jstore.ListEntriesParams{
			SpaceID:  spaceID,
			AuthorID: author,
			Before:   before,
			RowLimit: limit + 1,
		})
		if err != nil {
			return fmt.Errorf("list entries: %w", err)
		}
		if len(rows) > int(limit) {
			rows = rows[:limit]
			next := rows[len(rows)-1].ID
			page.Next = &next
		}
		page.Items = make([]Entry, 0, len(rows))
		for _, r := range rows {
			page.Items = append(page.Items, fromListRow(r))
		}
		return nil
	})
	return page, err
}

// Get returns one of the acting user's entries.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (Entry, error) {
	spaceID, author, err := scope(ctx)
	if err != nil {
		return Entry{}, err
	}
	var out Entry
	err = s.db.InReadOnlyTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.JournalRead); err != nil {
			return err
		}
		out, err = loadEntry(ctx, tx, spaceID, author, id)
		return err
	})
	return out, err
}

// Create saves a new entry for the acting user.
func (s *Service) Create(ctx context.Context, in Input) (Entry, error) {
	in = in.normalized()
	if err := in.validate(); err != nil {
		return Entry{}, err
	}
	spaceID, author, err := scope(ctx)
	if err != nil {
		return Entry{}, err
	}
	var out Entry
	err = s.db.InTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.JournalWrite); err != nil {
			return err
		}
		ent, err := corestore.New(tx).InsertEntity(ctx, corestore.InsertEntityParams{
			SpaceID: spaceID, Kind: kindEntry, CreatedBy: author, Now: s.clock.Now(),
		})
		if err != nil {
			return fmt.Errorf("insert entity: %w", err)
		}
		if err := insertEntry(ctx, tx, spaceID, author, ent.ID, in); err != nil {
			return err
		}
		out, err = loadEntry(ctx, tx, spaceID, author, ent.ID)
		return err
	})
	return out, err
}

// Update applies patch to one of the acting user's entries. expectedVersion
// must equal the stored version; otherwise the update is refused with 412.
func (s *Service) Update(ctx context.Context, id uuid.UUID, expectedVersion int32, patch Patch) (Entry, error) {
	spaceID, author, err := scope(ctx)
	if err != nil {
		return Entry{}, err
	}
	var out Entry
	err = s.db.InTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.JournalWrite); err != nil {
			return err
		}
		cur, err := loadEntry(ctx, tx, spaceID, author, id)
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
		if err := writeEntry(ctx, tx, spaceID, author, id, next); err != nil {
			return err
		}
		if _, err := corestore.New(tx).BumpEntityVersion(ctx, corestore.BumpEntityVersionParams{
			ID: id, ExpectedVersion: expectedVersion, Now: s.clock.Now(),
		}); errors.Is(err, pgx.ErrNoRows) {
			return staleVersion()
		} else if err != nil {
			return fmt.Errorf("bump version: %w", err)
		}
		out, err = loadEntry(ctx, tx, spaceID, author, id)
		return err
	})
	return out, err
}

// Delete moves one of the acting user's entries to the trash.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	spaceID, author, err := scope(ctx)
	if err != nil {
		return err
	}
	return s.db.InTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.JournalWrite); err != nil {
			return err
		}
		if _, err := loadEntry(ctx, tx, spaceID, author, id); err != nil {
			return err
		}
		if _, err := corestore.New(tx).SoftDeleteEntity(ctx, corestore.SoftDeleteEntityParams{
			ID: id, Now: ptrNow(s.clock.Now()),
		}); err != nil {
			return fmt.Errorf("move entry to trash: %w", err)
		}
		return nil
	})
}

func ptrNow(t time.Time) *time.Time { return &t }

// scope returns the space and acting user every journal call needs.
func scope(ctx context.Context) (uuid.UUID, uuid.UUID, error) {
	spaceID, ok := space.FromContext(ctx)
	if !ok {
		return uuid.Nil, uuid.Nil, apperr.Unauthorized("space.required", "Choose a space to continue.")
	}
	author, ok := db.UserIDFrom(ctx)
	if !ok {
		return uuid.Nil, uuid.Nil, apperr.Unauthorized("auth.required", "Sign in to continue.")
	}
	return spaceID, author, nil
}

func loadEntry(ctx context.Context, tx pgx.Tx, spaceID, author, id uuid.UUID) (Entry, error) {
	r, err := jstore.New(tx).GetEntry(ctx, jstore.GetEntryParams{ID: id, SpaceID: spaceID, AuthorID: author})
	if errors.Is(err, pgx.ErrNoRows) {
		// Another member's entry looks the same as a missing one.
		return Entry{}, apperr.NotFound("journal.entry_not_found", "Entry not found.")
	}
	if err != nil {
		return Entry{}, fmt.Errorf("load entry: %w", err)
	}
	return fromGetRow(r), nil
}

func insertEntry(ctx context.Context, tx pgx.Tx, spaceID, author, id uuid.UUID, in Input) error {
	date, err := dateOf(in.EntryDate)
	if err != nil {
		return err
	}
	if err := jstore.New(tx).InsertEntry(ctx, jstore.InsertEntryParams{
		ID: id, SpaceID: spaceID, AuthorID: author, EntryDate: date,
		Title: nullable(in.Title), Body: in.Body, Mood: moodPtr(in.Mood),
	}); err != nil {
		return fmt.Errorf("insert entry: %w", err)
	}
	return nil
}

func writeEntry(ctx context.Context, tx pgx.Tx, spaceID, author, id uuid.UUID, in Input) error {
	date, err := dateOf(in.EntryDate)
	if err != nil {
		return err
	}
	if err := jstore.New(tx).UpdateEntry(ctx, jstore.UpdateEntryParams{
		ID: id, SpaceID: spaceID, AuthorID: author, EntryDate: date,
		Title: nullable(in.Title), Body: in.Body, Mood: moodPtr(in.Mood),
	}); err != nil {
		return fmt.Errorf("update entry: %w", err)
	}
	return nil
}

func staleVersion() error {
	return apperr.New(apperr.KindPreconditionFailed, "journal.stale_version",
		"This entry was changed elsewhere. Reload it and try again.")
}
