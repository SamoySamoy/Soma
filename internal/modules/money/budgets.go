package money

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/SamoySamoy/Soma/internal/authz"
	"github.com/SamoySamoy/Soma/internal/modules/money/store"
	"github.com/SamoySamoy/Soma/internal/platform/apperr"
)

// errMissingRate is returned when a figure needs a currency with no known rate.
// It is a typed apperr so callers can report it, and the summary can list it.
func errMissingRate(currency string) error {
	return apperr.Invalid("money.missing_rate",
		"Add an exchange rate for "+currency+" first, so it can be shown in your base currency.",
		map[string]string{"currency": currency})
}

// rateTable maps each currency to its latest rate. The base currency needs none.
type rateTable map[string]int64

func loadRates(ctx context.Context, tx pgx.Tx, spaceID uuid.UUID, asOf time.Time) (rateTable, error) {
	rows, err := store.New(tx).LatestRates(ctx, store.LatestRatesParams{SpaceID: spaceID, AsOf: asOf})
	if err != nil {
		return nil, fmt.Errorf("latest rates: %w", err)
	}
	out := rateTable{}
	for _, r := range rows {
		out[r.Currency] = r.RateE8
	}
	return out, nil
}

// toBase converts an amount in currency into the base currency.
func (r rateTable) toBase(amount int64, currency, base string) (int64, error) {
	if currency == base {
		return amount, nil
	}
	rate, ok := r[currency]
	if !ok {
		return 0, errMissingRate(currency)
	}
	return Convert(amount, currency, base, rate)
}

// budgetLines returns each expense category's budget and spending for the month,
// in the base currency. Spending in a category includes its subcategories.
func budgetLines(ctx context.Context, tx pgx.Tx, spaceID uuid.UUID, base string, month, asOf time.Time) ([]BudgetLine, error) {
	start, end := monthBounds(month)
	rates, err := loadRates(ctx, tx, spaceID, asOf)
	if err != nil {
		return nil, err
	}
	q := store.New(tx)
	budgets, err := q.MonthBudgets(ctx, store.MonthBudgetsParams{SpaceID: spaceID, Month: start})
	if err != nil {
		return nil, fmt.Errorf("month budgets: %w", err)
	}
	spent, err := q.MonthExpenseByCategory(ctx, store.MonthExpenseByCategoryParams{
		SpaceID: spaceID, MonthStart: start, MonthEnd: end,
	})
	if err != nil {
		return nil, fmt.Errorf("month spending: %w", err)
	}
	own := map[uuid.UUID]int64{}
	for _, row := range spent {
		if row.CategoryID == nil {
			continue
		}
		amount, err := rates.toBase(row.SpentMinor, row.Currency, base)
		if err != nil {
			return nil, err
		}
		own[*row.CategoryID] += amount
	}

	childSpent := map[uuid.UUID]int64{}
	for _, b := range budgets {
		if b.ParentID != nil {
			childSpent[*b.ParentID] += own[b.CategoryID]
		}
	}
	lines := make([]BudgetLine, 0, len(budgets))
	for _, b := range budgets {
		lines = append(lines, BudgetLine{
			CategoryID:  b.CategoryID,
			Name:        b.Name,
			ParentID:    b.ParentID,
			BudgetMinor: b.BudgetMinor,
			SpentMinor:  own[b.CategoryID] + childSpent[b.CategoryID],
		})
	}
	return lines, nil
}

// Budgets returns the budget lines for the month that contains month.
func (s *Service) Budgets(ctx context.Context, month time.Time) (string, []BudgetLine, error) {
	spaceID, _, err := scope(ctx)
	if err != nil {
		return "", nil, err
	}
	var base string
	var lines []BudgetLine
	err = s.db.InReadOnlyTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.MoneyRead); err != nil {
			return err
		}
		settings, err := loadSettings(ctx, tx, spaceID)
		if err != nil {
			return err
		}
		base = settings.BaseCurrency
		lines, err = budgetLines(ctx, tx, spaceID, base, month, s.today())
		return err
	})
	return base, lines, err
}

// SetBudget sets a category's budget for the month that contains month. An
// amount of zero removes the budget. It returns the line after the change.
func (s *Service) SetBudget(ctx context.Context, categoryID uuid.UUID, month time.Time, amount int64) (BudgetLine, error) {
	if amount < 0 {
		return BudgetLine{}, invalid(map[string]string{"amount_minor": "Enter zero or more."})
	}
	spaceID, _, err := scope(ctx)
	if err != nil {
		return BudgetLine{}, err
	}
	start, _ := monthBounds(month)
	var line BudgetLine
	err = s.db.InTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.MoneyWrite); err != nil {
			return err
		}
		c, err := store.New(tx).GetCategory(ctx, store.GetCategoryParams{ID: categoryID, SpaceID: spaceID})
		if errors.Is(err, pgx.ErrNoRows) {
			return invalid(map[string]string{"category_id": "That category doesn't exist."})
		}
		if err != nil {
			return fmt.Errorf("load category: %w", err)
		}
		if c.Kind != "expense" {
			return invalid(map[string]string{"category_id": "Budgets are for expense categories."})
		}
		if amount == 0 {
			if _, err := store.New(tx).DeleteBudget(ctx, store.DeleteBudgetParams{
				SpaceID: spaceID, CategoryID: categoryID, Month: start,
			}); err != nil {
				return fmt.Errorf("delete budget: %w", err)
			}
		} else {
			if err := store.New(tx).UpsertBudget(ctx, store.UpsertBudgetParams{
				SpaceID: spaceID, CategoryID: categoryID, Month: start, AmountMinor: amount,
			}); err != nil {
				return fmt.Errorf("set budget: %w", err)
			}
		}
		settings, err := loadSettings(ctx, tx, spaceID)
		if err != nil {
			return err
		}
		lines, err := budgetLines(ctx, tx, spaceID, settings.BaseCurrency, start, s.today())
		if err != nil {
			return err
		}
		for _, l := range lines {
			if l.CategoryID == categoryID {
				line = l
			}
		}
		return nil
	})
	return line, err
}

// Summary is the money picture for one month, in the base currency.
type Summary struct {
	Month               time.Time
	BaseCurrency        string
	NetWorthMinor       int64
	IncomeMinor         int64
	ExpenseMinor        int64
	OverspentCategories int
	Accounts            []AccountTotal
	MissingRates        []string
}

// AccountTotal is an account's balance, with its base-currency value when known.
type AccountTotal struct {
	ID           uuid.UUID
	Name         string
	Currency     string
	BalanceMinor int64
	BaseMinor    *int64
}

// Summary returns net worth and the month's income and expenses. Currencies
// without a rate are left out and listed in MissingRates, never guessed.
func (s *Service) Summary(ctx context.Context, month time.Time) (Summary, error) {
	spaceID, _, err := scope(ctx)
	if err != nil {
		return Summary{}, err
	}
	out := Summary{Month: monthStart(month)}
	missing := map[string]bool{}
	err = s.db.InReadOnlyTx(ctx, func(tx pgx.Tx) error {
		if err := authz.Require(ctx, tx, spaceID, authz.MoneyRead); err != nil {
			return err
		}
		settings, err := loadSettings(ctx, tx, spaceID)
		if err != nil {
			return err
		}
		base := settings.BaseCurrency
		out.BaseCurrency = base
		asOf := s.today()
		rates, err := loadRates(ctx, tx, spaceID, asOf)
		if err != nil {
			return err
		}

		accounts, err := listAccounts(ctx, tx, spaceID)
		if err != nil {
			return err
		}
		for _, a := range accounts {
			total := AccountTotal{ID: a.ID, Name: a.Name, Currency: a.Currency, BalanceMinor: a.BalanceMinor}
			converted, err := rates.toBase(a.BalanceMinor, a.Currency, base)
			if err != nil {
				missing[a.Currency] = true
			} else {
				total.BaseMinor = &converted
				out.NetWorthMinor += converted
			}
			out.Accounts = append(out.Accounts, total)
		}

		start, end := monthBounds(month)
		flows, err := store.New(tx).MonthFlows(ctx, store.MonthFlowsParams{SpaceID: spaceID, MonthStart: start, MonthEnd: end})
		if err != nil {
			return fmt.Errorf("month flows: %w", err)
		}
		for _, f := range flows {
			amount, err := rates.toBase(f.TotalMinor, f.Currency, base)
			if err != nil {
				missing[f.Currency] = true
				continue
			}
			if f.Kind == "income" {
				out.IncomeMinor += amount
			} else {
				out.ExpenseMinor += -amount
			}
		}

		lines, err := budgetLines(ctx, tx, spaceID, base, month, asOf)
		if err != nil {
			if errMissingRateOf(err) {
				// Budgets need the rate too; the totals above still stand.
				return nil
			}
			return err
		}
		for _, l := range lines {
			if l.BudgetMinor > 0 && l.SpentMinor > l.BudgetMinor {
				out.OverspentCategories++
			}
		}
		return nil
	})
	if err != nil {
		return Summary{}, err
	}
	for c := range missing {
		out.MissingRates = append(out.MissingRates, c)
	}
	sort.Strings(out.MissingRates)
	return out, nil
}

func monthStart(t time.Time) time.Time {
	start, _ := monthBounds(t)
	return start
}

// errMissingRateOf reports whether err is the missing-rate error.
func errMissingRateOf(err error) bool {
	e, ok := apperr.As(err)
	return ok && e.Code == "money.missing_rate"
}
