package tasks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/SamoySamoy/Soma/internal/authz"
	corestore "github.com/SamoySamoy/Soma/internal/core/store"
	tstore "github.com/SamoySamoy/Soma/internal/modules/tasks/store"
	"github.com/SamoySamoy/Soma/internal/platform/apperr"
	"github.com/SamoySamoy/Soma/internal/platform/clock"
	"github.com/SamoySamoy/Soma/internal/platform/db"
	"github.com/SamoySamoy/Soma/internal/space"
)

const kindTask = "tasks.task"

// Service implements task use cases. Each method runs in one transaction,
// checks authorization first, and works only inside the space on the context.
type Service struct {
	db    *db.DB
	clock clock.Clock
}

// NewService returns a Service backed by d.
func NewService(d *db.DB, clk clock.Clock) *Service {
	return &Service{db: d, clock: clk}
}

// List returns up to limit tasks, newest saved first. With openOnly, done
// tasks are left out.
func (s *Service) List(ctx context.Context, limit int32, before *uuid.UUID, openOnly bool) (Page, error) {
	spaceID, err := spaceFrom(ctx)
	if err != nil {
		return Page{}, err
	}
	var page Page
	err = s.db.InReadOnlyTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.TasksRead); err != nil {
			return err
		}
		rows, err := tstore.New(tx).ListTasks(ctx, tstore.ListTasksParams{
			SpaceID:  spaceID,
			Before:   before,
			OnlyOpen: openOnly,
			RowLimit: limit + 1,
		})
		if err != nil {
			return fmt.Errorf("list tasks: %w", err)
		}
		if len(rows) > int(limit) {
			rows = rows[:limit]
			next := rows[len(rows)-1].ID
			page.Next = &next
		}
		page.Items = make([]Task, 0, len(rows))
		for _, r := range rows {
			page.Items = append(page.Items, fromListRow(r))
		}
		return nil
	})
	return page, err
}

// Get returns one task.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (Task, error) {
	spaceID, err := spaceFrom(ctx)
	if err != nil {
		return Task{}, err
	}
	var out Task
	err = s.db.InReadOnlyTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.TasksRead); err != nil {
			return err
		}
		out, err = loadTask(ctx, tx, spaceID, id)
		return err
	})
	return out, err
}

// Create adds a task to the space.
func (s *Service) Create(ctx context.Context, in Input) (Task, error) {
	in = in.normalized()
	if err := in.validate(); err != nil {
		return Task{}, err
	}
	spaceID, err := spaceFrom(ctx)
	if err != nil {
		return Task{}, err
	}
	userID, ok := db.UserIDFrom(ctx)
	if !ok {
		return Task{}, apperr.Unauthorized("auth.required", "Sign in to continue.")
	}
	var out Task
	err = s.db.InTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.TasksWrite); err != nil {
			return err
		}
		ent, err := corestore.New(tx).InsertEntity(ctx, corestore.InsertEntityParams{
			SpaceID: spaceID, Kind: kindTask, CreatedBy: userID, Now: s.clock.Now(),
		})
		if err != nil {
			return fmt.Errorf("insert entity: %w", err)
		}
		due, err := dueOf(in.DueOn)
		if err != nil {
			return err
		}
		if err := tstore.New(tx).InsertTask(ctx, tstore.InsertTaskParams{
			ID: ent.ID, SpaceID: spaceID, Title: in.Title, Notes: nullable(in.Notes), DueOn: due,
		}); err != nil {
			return fmt.Errorf("insert task: %w", err)
		}
		out, err = loadTask(ctx, tx, spaceID, ent.ID)
		return err
	})
	return out, err
}

// Update applies patch to a task. expectedVersion must equal the stored
// version; otherwise the update is refused with 412.
func (s *Service) Update(ctx context.Context, id uuid.UUID, expectedVersion int32, patch Patch) (Task, error) {
	spaceID, err := spaceFrom(ctx)
	if err != nil {
		return Task{}, err
	}
	var out Task
	err = s.db.InTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.TasksWrite); err != nil {
			return err
		}
		cur, err := loadTask(ctx, tx, spaceID, id)
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
		if err := writeTask(ctx, tx, spaceID, id, next); err != nil {
			return err
		}
		if err := bump(ctx, tx, id, expectedVersion, s.clock.Now()); err != nil {
			return err
		}
		out, err = loadTask(ctx, tx, spaceID, id)
		return err
	})
	return out, err
}

// Complete marks a task done. Completing a done task changes nothing.
func (s *Service) Complete(ctx context.Context, id uuid.UUID) (Task, error) {
	return s.setDone(ctx, id, true)
}

// Reopen marks a done task open again. Reopening an open task changes nothing.
func (s *Service) Reopen(ctx context.Context, id uuid.UUID) (Task, error) {
	return s.setDone(ctx, id, false)
}

func (s *Service) setDone(ctx context.Context, id uuid.UUID, done bool) (Task, error) {
	spaceID, err := spaceFrom(ctx)
	if err != nil {
		return Task{}, err
	}
	var out Task
	err = s.db.InTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.TasksWrite); err != nil {
			return err
		}
		cur, err := loadTask(ctx, tx, spaceID, id)
		if err != nil {
			return err
		}
		if cur.Completed() == done {
			out = cur
			return nil
		}
		now := s.clock.Now()
		var completedAt *time.Time
		if done {
			completedAt = &now
		}
		if err := tstore.New(tx).SetTaskCompleted(ctx, tstore.SetTaskCompletedParams{
			ID: id, SpaceID: spaceID, CompletedAt: completedAt,
		}); err != nil {
			return fmt.Errorf("set completed: %w", err)
		}
		if err := bump(ctx, tx, id, cur.Version, now); err != nil {
			return err
		}
		out, err = loadTask(ctx, tx, spaceID, id)
		return err
	})
	return out, err
}

// Delete moves a task to the trash.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	spaceID, err := spaceFrom(ctx)
	if err != nil {
		return err
	}
	return s.db.InTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.TasksWrite); err != nil {
			return err
		}
		if _, err := loadTask(ctx, tx, spaceID, id); err != nil {
			return err
		}
		if _, err := corestore.New(tx).SoftDeleteEntity(ctx, corestore.SoftDeleteEntityParams{
			ID: id, Now: ptrNow(s.clock.Now()),
		}); err != nil {
			return fmt.Errorf("move task to trash: %w", err)
		}
		return nil
	})
}

// today is the server's current UTC date. Time zones come with identity (M1).
func (s *Service) today() time.Time {
	now := s.clock.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

func ptrNow(t time.Time) *time.Time { return &t }

func spaceFrom(ctx context.Context) (uuid.UUID, error) {
	id, ok := space.FromContext(ctx)
	if !ok {
		return uuid.Nil, apperr.Unauthorized("space.required", "Choose a space to continue.")
	}
	return id, nil
}

func loadTask(ctx context.Context, tx pgx.Tx, spaceID, id uuid.UUID) (Task, error) {
	r, err := tstore.New(tx).GetTask(ctx, tstore.GetTaskParams{ID: id, SpaceID: spaceID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Task{}, apperr.NotFound("tasks.task_not_found", "Task not found.")
	}
	if err != nil {
		return Task{}, fmt.Errorf("load task: %w", err)
	}
	return fromGetRow(r), nil
}

func writeTask(ctx context.Context, tx pgx.Tx, spaceID, id uuid.UUID, in Input) error {
	due, err := dueOf(in.DueOn)
	if err != nil {
		return err
	}
	if err := tstore.New(tx).UpdateTask(ctx, tstore.UpdateTaskParams{
		ID: id, SpaceID: spaceID, Title: in.Title, Notes: nullable(in.Notes), DueOn: due,
	}); err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	return nil
}

// bump increments the version, or refuses with 412 if it no longer matches.
func bump(ctx context.Context, tx pgx.Tx, id uuid.UUID, expected int32, now time.Time) error {
	_, err := corestore.New(tx).BumpEntityVersion(ctx, corestore.BumpEntityVersionParams{
		ID: id, ExpectedVersion: expected, Now: now,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return staleVersion()
	}
	if err != nil {
		return fmt.Errorf("bump version: %w", err)
	}
	return nil
}

func staleVersion() error {
	return apperr.New(apperr.KindPreconditionFailed, "tasks.stale_version",
		"This task was changed elsewhere. Reload it and try again.")
}
