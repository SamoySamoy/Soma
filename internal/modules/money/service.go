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
	"github.com/SamoySamoy/Soma/internal/platform/clock"
	"github.com/SamoySamoy/Soma/internal/platform/db"
	"github.com/SamoySamoy/Soma/internal/space"
)

const (
	kindAccount     = "money.account"
	kindTransaction = "money.transaction"
)

// Service implements money use cases for the space on the context. Each method
// checks authorization first and runs in one transaction.
type Service struct {
	db    *db.DB
	clock clock.Clock
}

// NewService returns a Service backed by d.
func NewService(d *db.DB, clk clock.Clock) *Service {
	return &Service{db: d, clock: clk}
}

// Now is the current time from the injected clock.
func (s *Service) Now() time.Time { return s.clock.Now() }

// today is the server's current UTC date. Time zones come with identity (M1).
func (s *Service) today() time.Time {
	now := s.clock.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// scope returns the space and the acting user.
func scope(ctx context.Context) (uuid.UUID, uuid.UUID, error) {
	spaceID, ok := space.FromContext(ctx)
	if !ok {
		return uuid.Nil, uuid.Nil, apperr.Unauthorized("space.required", "Choose a space to continue.")
	}
	user, ok := db.UserIDFrom(ctx)
	if !ok {
		return uuid.Nil, uuid.Nil, apperr.Unauthorized("auth.required", "Sign in to continue.")
	}
	return spaceID, user, nil
}

// Settings returns the money settings.
func (s *Service) Settings(ctx context.Context) (Settings, error) {
	spaceID, _, err := scope(ctx)
	if err != nil {
		return Settings{}, err
	}
	var out Settings
	err = s.db.InReadOnlyTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.MoneyRead); err != nil {
			return err
		}
		out, err = loadSettings(ctx, tx, spaceID)
		return err
	})
	return out, err
}

// UpdateSettings changes the base currency. expectedVersion must match.
func (s *Service) UpdateSettings(ctx context.Context, expectedVersion int32, patch Patch) (Settings, error) {
	spaceID, _, err := scope(ctx)
	if err != nil {
		return Settings{}, err
	}
	var out Settings
	err = s.db.InTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.MoneyWrite); err != nil {
			return err
		}
		cur, err := loadSettings(ctx, tx, spaceID)
		if err != nil {
			return err
		}
		if cur.Version != expectedVersion {
			return staleVersion("The money settings changed elsewhere. Reload and try again.")
		}
		base := cur.BaseCurrency
		if v, ok := patch["base_currency"]; ok {
			s, isText := textOf(v)
			if !isText {
				return invalid(map[string]string{"base_currency": "Choose a supported currency."})
			}
			base = s
		}
		for key := range patch {
			if key != "base_currency" {
				return invalid(map[string]string{key: "This field can't be changed."})
			}
		}
		if err := (Settings{}).validate(base); err != nil {
			return err
		}
		_, err = store.New(tx).UpdateSettings(ctx, store.UpdateSettingsParams{
			SpaceID: spaceID, BaseCurrency: base, Now: s.clock.Now(), ExpectedVersion: expectedVersion,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return staleVersion("The money settings changed elsewhere. Reload and try again.")
		}
		if err != nil {
			return fmt.Errorf("update settings: %w", err)
		}
		out, err = loadSettings(ctx, tx, spaceID)
		return err
	})
	return out, err
}

// Accounts lists the accounts with their balances.
func (s *Service) Accounts(ctx context.Context) ([]Account, error) {
	spaceID, _, err := scope(ctx)
	if err != nil {
		return nil, err
	}
	var out []Account
	err = s.db.InReadOnlyTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.MoneyRead); err != nil {
			return err
		}
		out, err = listAccounts(ctx, tx, spaceID)
		return err
	})
	return out, err
}

// Account returns one account with its balance.
func (s *Service) Account(ctx context.Context, id uuid.UUID) (Account, error) {
	spaceID, _, err := scope(ctx)
	if err != nil {
		return Account{}, err
	}
	var out Account
	err = s.db.InReadOnlyTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.MoneyRead); err != nil {
			return err
		}
		out, err = loadAccount(ctx, tx, spaceID, id)
		return err
	})
	return out, err
}

// CreateAccount adds an account.
func (s *Service) CreateAccount(ctx context.Context, in AccountInput) (Account, error) {
	in = in.normalized()
	if err := in.validate(); err != nil {
		return Account{}, err
	}
	spaceID, user, err := scope(ctx)
	if err != nil {
		return Account{}, err
	}
	var out Account
	err = s.db.InTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.MoneyWrite); err != nil {
			return err
		}
		ent, err := corestore.New(tx).InsertEntity(ctx, corestore.InsertEntityParams{
			SpaceID: spaceID, Kind: kindAccount, CreatedBy: user, Now: s.clock.Now(),
		})
		if err != nil {
			return fmt.Errorf("insert entity: %w", err)
		}
		opened, err := time.Parse(time.DateOnly, in.OpenedOn)
		if err != nil {
			return err
		}
		if err := store.New(tx).InsertAccount(ctx, store.InsertAccountParams{
			ID: ent.ID, SpaceID: spaceID, Name: in.Name, Kind: in.Kind, Currency: in.Currency,
			OpeningMinor: in.OpeningMinor, OpenedOn: opened,
		}); err != nil {
			return fmt.Errorf("insert account: %w", err)
		}
		out, err = loadAccount(ctx, tx, spaceID, ent.ID)
		return err
	})
	return out, err
}

// UpdateAccount edits an account. The currency can't change.
func (s *Service) UpdateAccount(ctx context.Context, id uuid.UUID, expectedVersion int32, patch Patch) (Account, error) {
	spaceID, _, err := scope(ctx)
	if err != nil {
		return Account{}, err
	}
	var out Account
	err = s.db.InTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.MoneyWrite); err != nil {
			return err
		}
		cur, err := loadAccount(ctx, tx, spaceID, id)
		if err != nil {
			return err
		}
		if cur.Version != expectedVersion {
			return staleVersion("This account changed elsewhere. Reload and try again.")
		}
		in, err := AccountInput{Name: cur.Name, Kind: cur.Kind, Currency: cur.Currency,
			OpeningMinor: cur.OpeningMinor, OpenedOn: cur.OpenedOn}.apply(patch)
		if err != nil {
			return err
		}
		in = in.normalized()
		if err := in.validate(); err != nil {
			return err
		}
		opened, err := time.Parse(time.DateOnly, in.OpenedOn)
		if err != nil {
			return err
		}
		if err := store.New(tx).UpdateAccount(ctx, store.UpdateAccountParams{
			ID: id, SpaceID: spaceID, Name: in.Name, Kind: in.Kind,
			OpeningMinor: in.OpeningMinor, OpenedOn: opened,
		}); err != nil {
			return fmt.Errorf("update account: %w", err)
		}
		if err := bump(ctx, tx, id, expectedVersion, s.clock.Now()); err != nil {
			return err
		}
		out, err = loadAccount(ctx, tx, spaceID, id)
		return err
	})
	return out, err
}

// DeleteAccount moves an account to the trash.
func (s *Service) DeleteAccount(ctx context.Context, id uuid.UUID) error {
	spaceID, _, err := scope(ctx)
	if err != nil {
		return err
	}
	return s.db.InTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.MoneyWrite); err != nil {
			return err
		}
		if _, err := loadAccount(ctx, tx, spaceID, id); err != nil {
			return err
		}
		_, err := corestore.New(tx).SoftDeleteEntity(ctx, corestore.SoftDeleteEntityParams{
			ID: id, Now: ptrNow(s.clock.Now()),
		})
		if err != nil {
			return fmt.Errorf("move account to trash: %w", err)
		}
		return nil
	})
}

// Categories lists the categories, parents before their children.
func (s *Service) Categories(ctx context.Context) ([]Category, error) {
	spaceID, _, err := scope(ctx)
	if err != nil {
		return nil, err
	}
	var out []Category
	err = s.db.InReadOnlyTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.MoneyRead); err != nil {
			return err
		}
		rows, err := store.New(tx).ListCategories(ctx, spaceID)
		if err != nil {
			return fmt.Errorf("list categories: %w", err)
		}
		out = make([]Category, 0, len(rows))
		for _, r := range rows {
			out = append(out, Category{ID: r.ID, ParentID: r.ParentID, Name: r.Name, Kind: r.Kind})
		}
		return nil
	})
	return out, err
}

// CreateCategory adds a category. A subcategory's parent must be a top-level
// category of the same kind.
func (s *Service) CreateCategory(ctx context.Context, in CategoryInput) (Category, error) {
	in = in.normalized()
	if err := in.validate(); err != nil {
		return Category{}, err
	}
	spaceID, _, err := scope(ctx)
	if err != nil {
		return Category{}, err
	}
	var out Category
	err = s.db.InTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.MoneyWrite); err != nil {
			return err
		}
		if in.ParentID != nil {
			parent, err := store.New(tx).GetCategory(ctx, store.GetCategoryParams{ID: *in.ParentID, SpaceID: spaceID})
			if errors.Is(err, pgx.ErrNoRows) {
				return invalid(map[string]string{"parent_id": "That category doesn't exist."})
			}
			if err != nil {
				return fmt.Errorf("load parent: %w", err)
			}
			if parent.ParentID != nil {
				return invalid(map[string]string{"parent_id": "Categories can only be nested two levels deep."})
			}
			if parent.Kind != in.Kind {
				return invalid(map[string]string{"kind": "A subcategory must have the same kind as its parent."})
			}
		}
		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("new category id: %w", err)
		}
		if err := store.New(tx).InsertCategory(ctx, store.InsertCategoryParams{
			ID: id, SpaceID: spaceID, ParentID: in.ParentID, Name: in.Name, Kind: in.Kind, Now: s.clock.Now(),
		}); err != nil {
			return fmt.Errorf("insert category: %w", err)
		}
		out = Category{ID: id, ParentID: in.ParentID, Name: in.Name, Kind: in.Kind}
		return nil
	})
	return out, err
}

// SetRate stores the exchange rate of a currency on a day.
func (s *Service) SetRate(ctx context.Context, currency, on string, rateE8 int64) (Rate, error) {
	if err := validRate(currency, rateE8, on); err != nil {
		return Rate{}, err
	}
	spaceID, _, err := scope(ctx)
	if err != nil {
		return Rate{}, err
	}
	day, err := time.Parse(time.DateOnly, on)
	if err != nil {
		return Rate{}, err
	}
	err = s.db.InTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.MoneyWrite); err != nil {
			return err
		}
		return store.New(tx).UpsertRate(ctx, store.UpsertRateParams{
			SpaceID: spaceID, Currency: currency, RateOn: day, RateE8: rateE8,
		})
	})
	if err != nil {
		return Rate{}, err
	}
	return Rate{Currency: currency, RateOn: on, RateE8: rateE8}, nil
}

// LatestRates returns the newest rate on or before today, per currency.
func (s *Service) LatestRates(ctx context.Context) ([]Rate, error) {
	spaceID, _, err := scope(ctx)
	if err != nil {
		return nil, err
	}
	var out []Rate
	err = s.db.InReadOnlyTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.MoneyRead); err != nil {
			return err
		}
		rows, err := store.New(tx).LatestRates(ctx, store.LatestRatesParams{SpaceID: spaceID, AsOf: s.today()})
		if err != nil {
			return fmt.Errorf("latest rates: %w", err)
		}
		for _, r := range rows {
			out = append(out, Rate{Currency: r.Currency, RateOn: r.RateOn.Format(time.DateOnly), RateE8: r.RateE8})
		}
		return nil
	})
	return out, err
}

// loadSettings reads the settings, which are created with the space.
func loadSettings(ctx context.Context, tx pgx.Tx, spaceID uuid.UUID) (Settings, error) {
	r, err := store.New(tx).GetSettings(ctx, spaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Settings{}, apperr.NotFound("money.settings_not_found", "Money settings not found.")
	}
	if err != nil {
		return Settings{}, fmt.Errorf("load settings: %w", err)
	}
	return Settings{BaseCurrency: r.BaseCurrency, UpdatedAt: r.UpdatedAt, Version: r.Version}, nil
}

func listAccounts(ctx context.Context, tx pgx.Tx, spaceID uuid.UUID) ([]Account, error) {
	rows, err := store.New(tx).ListAccounts(ctx, spaceID)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	balances, err := balancesOf(ctx, tx, spaceID)
	if err != nil {
		return nil, err
	}
	out := make([]Account, 0, len(rows))
	for _, r := range rows {
		out = append(out, Account{
			ID: r.ID, Name: r.Name, Kind: r.Kind, Currency: r.Currency, OpeningMinor: r.OpeningMinor,
			OpenedOn: r.OpenedOn.Format(time.DateOnly), BalanceMinor: balances[r.ID],
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, Version: r.Version,
		})
	}
	return out, nil
}

func loadAccount(ctx context.Context, tx pgx.Tx, spaceID, id uuid.UUID) (Account, error) {
	r, err := store.New(tx).GetAccount(ctx, store.GetAccountParams{ID: id, SpaceID: spaceID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, apperr.NotFound("money.account_not_found", "Account not found.")
	}
	if err != nil {
		return Account{}, fmt.Errorf("load account: %w", err)
	}
	balances, err := balancesOf(ctx, tx, spaceID)
	if err != nil {
		return Account{}, err
	}
	return Account{
		ID: r.ID, Name: r.Name, Kind: r.Kind, Currency: r.Currency, OpeningMinor: r.OpeningMinor,
		OpenedOn: r.OpenedOn.Format(time.DateOnly), BalanceMinor: balances[r.ID],
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, Version: r.Version,
	}, nil
}

// balancesOf maps each account to its balance in its own currency.
func balancesOf(ctx context.Context, tx pgx.Tx, spaceID uuid.UUID) (map[uuid.UUID]int64, error) {
	rows, err := store.New(tx).AccountBalances(ctx, spaceID)
	if err != nil {
		return nil, fmt.Errorf("account balances: %w", err)
	}
	out := make(map[uuid.UUID]int64, len(rows))
	for _, r := range rows {
		out[r.ID] = r.BalanceMinor
	}
	return out, nil
}

// bump increments the version, or refuses with 412 if it no longer matches.
func bump(ctx context.Context, tx pgx.Tx, id uuid.UUID, expected int32, now time.Time) error {
	_, err := corestore.New(tx).BumpEntityVersion(ctx, corestore.BumpEntityVersionParams{
		ID: id, ExpectedVersion: expected, Now: now,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return staleVersion("This changed elsewhere. Reload and try again.")
	}
	if err != nil {
		return fmt.Errorf("bump version: %w", err)
	}
	return nil
}

func staleVersion(msg string) error {
	return apperr.New(apperr.KindPreconditionFailed, "money.stale_version", msg)
}

func ptrNow(t time.Time) *time.Time { return &t }

// keep pgtype in the import graph for the date helpers in mapping.go.
var _ = pgtype.Date{}
