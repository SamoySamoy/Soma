package money

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/SamoySamoy/Soma/internal/authz"
	corestore "github.com/SamoySamoy/Soma/internal/core/store"
	"github.com/SamoySamoy/Soma/internal/modules/money/store"
	"github.com/SamoySamoy/Soma/internal/platform/apperr"
)

// Transactions lists transactions newest first. A non-nil accountID limits the
// list to one account. cursor is the value from a previous page's Next.
func (s *Service) Transactions(ctx context.Context, limit int32, accountID *uuid.UUID, cursor string) (Page, error) {
	spaceID, _, err := scope(ctx)
	if err != nil {
		return Page{}, err
	}
	beforeDate, beforeID, err := parseCursor(cursor)
	if err != nil {
		return Page{}, err
	}
	var page Page
	err = s.db.InReadOnlyTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.MoneyRead); err != nil {
			return err
		}
		rows, err := store.New(tx).ListTransactions(ctx, store.ListTransactionsParams{
			SpaceID: spaceID, AccountID: accountID, BeforeDate: beforeDate, BeforeID: beforeID, RowLimit: limit + 1,
		})
		if err != nil {
			return fmt.Errorf("list transactions: %w", err)
		}
		if len(rows) > int(limit) {
			rows = rows[:limit]
			last := rows[len(rows)-1]
			page.Next = fmt.Sprintf("%s_%s", last.OccurredOn.Format(time.DateOnly), last.ID)
		}
		page.Items = make([]Transaction, 0, len(rows))
		for _, r := range rows {
			page.Items = append(page.Items, fromListRow(r))
		}
		return nil
	})
	return page, err
}

// Transaction returns one transaction.
func (s *Service) Transaction(ctx context.Context, id uuid.UUID) (Transaction, error) {
	spaceID, _, err := scope(ctx)
	if err != nil {
		return Transaction{}, err
	}
	var out Transaction
	err = s.db.InReadOnlyTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.MoneyRead); err != nil {
			return err
		}
		out, err = loadTransaction(ctx, tx, spaceID, id)
		return err
	})
	return out, err
}

// RecordTransaction records an income, an expense or a transfer. A transfer
// writes two linked rows; the returned row is the outgoing side.
func (s *Service) RecordTransaction(ctx context.Context, in TransactionInput) (Transaction, error) {
	if err := in.validate(); err != nil {
		return Transaction{}, err
	}
	spaceID, user, err := scope(ctx)
	if err != nil {
		return Transaction{}, err
	}
	var out Transaction
	err = s.db.InTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.MoneyWrite); err != nil {
			return err
		}
		from, err := accountCurrency(ctx, tx, spaceID, in.AccountID)
		if err != nil {
			return err
		}
		if in.Kind == KindTransfer {
			return s.recordTransfer(ctx, tx, spaceID, user, in, from, &out)
		}
		if err := checkCategory(ctx, tx, spaceID, in.CategoryID, in.Kind); err != nil {
			return err
		}
		kind := KindIncome
		if in.Kind == KindExpense {
			kind = "expense"
		}
		id, err := s.insertRow(ctx, tx, spaceID, user, in.AccountID, kind, in.Amount, in.OccurredOn, in.CategoryID, in.Payee, in.Notes, nil)
		if err != nil {
			return err
		}
		out, err = loadTransaction(ctx, tx, spaceID, id)
		return err
	})
	return out, err
}

func (s *Service) recordTransfer(ctx context.Context, tx pgx.Tx, spaceID, user uuid.UUID, in TransactionInput, from string, out *Transaction) error {
	to, err := accountCurrency(ctx, tx, spaceID, *in.ToAccountID)
	if err != nil {
		return err
	}
	if to != from {
		return apperr.Invalid("money.transfer_currency",
			"Transfers need two accounts in the same currency for now.", map[string]string{"to_account_id": "Choose an account in " + from + "."})
	}
	transferID, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("transfer id: %w", err)
	}
	outID, err := s.insertRow(ctx, tx, spaceID, user, in.AccountID, "transfer_out", in.Amount, in.OccurredOn, nil, in.Payee, in.Notes, &transferID)
	if err != nil {
		return err
	}
	if _, err := s.insertRow(ctx, tx, spaceID, user, *in.ToAccountID, "transfer_in", in.Amount, in.OccurredOn, nil, in.Payee, in.Notes, &transferID); err != nil {
		return err
	}
	loaded, err := loadTransaction(ctx, tx, spaceID, outID)
	if err != nil {
		return err
	}
	*out = loaded
	return nil
}

// UpdateTransaction edits a transaction. Transfers and kinds can't change.
func (s *Service) UpdateTransaction(ctx context.Context, id uuid.UUID, expectedVersion int32, patch Patch) (Transaction, error) {
	spaceID, _, err := scope(ctx)
	if err != nil {
		return Transaction{}, err
	}
	var out Transaction
	err = s.db.InTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.MoneyWrite); err != nil {
			return err
		}
		cur, err := loadTransaction(ctx, tx, spaceID, id)
		if err != nil {
			return err
		}
		if cur.Version != expectedVersion {
			return staleVersion("This transaction changed elsewhere. Reload and try again.")
		}
		next, err := cur.apply(patch)
		if err != nil {
			return err
		}
		if !validDate(next.OccurredOn) {
			return invalid(map[string]string{"occurred_on": "Use the format YYYY-MM-DD."})
		}
		if next.Amount != cur.Amount {
			// Keep the sign the kind requires; a new amount is always positive.
			if next.Amount < 1 {
				return invalid(map[string]string{"amount_minor": "Enter an amount greater than zero."})
			}
		}
		if next.CategoryID != nil || cur.CategoryID != nil {
			if err := checkCategory(ctx, tx, spaceID, next.CategoryID, categoryKindOf(cur.Kind)); err != nil {
				return err
			}
		}
		stored := signed(cur.Kind, next.Amount)
		if cur.Kind == "adjustment" {
			stored = sign(cur.Amount) * next.Amount
		}
		if err := store.New(tx).UpdateTransaction(ctx, store.UpdateTransactionParams{
			ID: id, SpaceID: spaceID, AmountMinor: stored, OccurredOn: mustDate(next.OccurredOn),
			CategoryID: next.CategoryID, Payee: nullable(next.Payee), Notes: nullable(next.Notes),
		}); err != nil {
			return fmt.Errorf("update transaction: %w", err)
		}
		if err := bump(ctx, tx, id, expectedVersion, s.clock.Now()); err != nil {
			return err
		}
		out, err = loadTransaction(ctx, tx, spaceID, id)
		return err
	})
	return out, err
}

// DeleteTransaction moves a transaction to the trash. A transfer's two sides go together.
func (s *Service) DeleteTransaction(ctx context.Context, id uuid.UUID) error {
	spaceID, _, err := scope(ctx)
	if err != nil {
		return err
	}
	return s.db.InTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.MoneyWrite); err != nil {
			return err
		}
		cur, err := loadTransaction(ctx, tx, spaceID, id)
		if err != nil {
			return err
		}
		ids := []uuid.UUID{id}
		if cur.TransferID != nil {
			legs, err := store.New(tx).TransferLegs(ctx, store.TransferLegsParams{TransferID: cur.TransferID, SpaceID: spaceID})
			if err != nil {
				return fmt.Errorf("load transfer legs: %w", err)
			}
			ids = legs
		}
		now := s.clock.Now()
		for _, leg := range ids {
			if _, err := corestore.New(tx).SoftDeleteEntity(ctx, corestore.SoftDeleteEntityParams{
				ID: leg, Now: ptrNow(now),
			}); err != nil {
				return fmt.Errorf("move transaction to trash: %w", err)
			}
		}
		return nil
	})
}

// insertRow writes one entity and its transaction row.
func (s *Service) insertRow(ctx context.Context, tx pgx.Tx, spaceID, user, accountID uuid.UUID, kind string, amount int64,
	occurredOn string, categoryID *uuid.UUID, payee, notes string, transferID *uuid.UUID) (uuid.UUID, error) {
	ent, err := corestore.New(tx).InsertEntity(ctx, corestore.InsertEntityParams{
		SpaceID: spaceID, Kind: kindTransaction, CreatedBy: user, Now: s.clock.Now(),
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert entity: %w", err)
	}
	if err := store.New(tx).InsertTransaction(ctx, store.InsertTransactionParams{
		ID: ent.ID, SpaceID: spaceID, AccountID: accountID, Kind: kind,
		AmountMinor: signed(kind, amount), OccurredOn: mustDate(occurredOn),
		CategoryID: categoryID, Payee: nullable(payee), Notes: nullable(notes), TransferID: transferID,
	}); err != nil {
		return uuid.Nil, fmt.Errorf("insert transaction: %w", err)
	}
	return ent.ID, nil
}

// accountCurrency returns an account's currency, or NotFound.
func accountCurrency(ctx context.Context, tx pgx.Tx, spaceID, id uuid.UUID) (string, error) {
	a, err := loadAccount(ctx, tx, spaceID, id)
	if err != nil {
		return "", err
	}
	return a.Currency, nil
}

// checkCategory verifies that a category exists and matches the kind of entry.
// A nil category is always fine.
func checkCategory(ctx context.Context, tx pgx.Tx, spaceID uuid.UUID, categoryID *uuid.UUID, kind string) error {
	if categoryID == nil {
		return nil
	}
	c, err := store.New(tx).GetCategory(ctx, store.GetCategoryParams{ID: *categoryID, SpaceID: spaceID})
	if errors.Is(err, pgx.ErrNoRows) {
		return invalid(map[string]string{"category_id": "That category doesn't exist."})
	}
	if err != nil {
		return fmt.Errorf("load category: %w", err)
	}
	want := "expense"
	if kind == KindIncome || kind == "income" {
		want = "income"
	}
	if c.Kind != want {
		return invalid(map[string]string{"category_id": "Choose a " + want + " category."})
	}
	return nil
}

// categoryKindOf maps a stored transaction kind to the category kind it uses.
func categoryKindOf(kind string) string {
	if kind == "expense" {
		return KindExpense
	}
	return KindIncome
}

func loadTransaction(ctx context.Context, tx pgx.Tx, spaceID, id uuid.UUID) (Transaction, error) {
	r, err := store.New(tx).GetTransaction(ctx, store.GetTransactionParams{ID: id, SpaceID: spaceID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Transaction{}, apperr.NotFound("money.transaction_not_found", "Transaction not found.")
	}
	if err != nil {
		return Transaction{}, fmt.Errorf("load transaction: %w", err)
	}
	return fromGetRow(r), nil
}

// parseCursor reads "YYYY-MM-DD_<uuid>". An empty cursor starts at the newest row.
func parseCursor(c string) (pgtype.Date, *uuid.UUID, error) {
	if c == "" {
		return pgtype.Date{}, nil, nil
	}
	if len(c) < 11 {
		return pgtype.Date{}, nil, badCursor()
	}
	day, err := time.Parse(time.DateOnly, c[:10])
	if err != nil {
		return pgtype.Date{}, nil, badCursor()
	}
	id, err := uuid.Parse(c[11:])
	if err != nil {
		return pgtype.Date{}, nil, badCursor()
	}
	return pgtype.Date{Time: day, Valid: true}, &id, nil
}

func badCursor() error {
	return apperr.Invalid("money.invalid_cursor", "This page link is no longer valid.",
		map[string]string{"cursor": "Start again from the first page."})
}

func mustDate(s string) time.Time {
	t, _ := time.Parse(time.DateOnly, s) // validated before every call
	return t
}
