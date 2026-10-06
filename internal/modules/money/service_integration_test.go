//go:build integration

package money_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	corestore "github.com/SamoySamoy/Soma/internal/core/store"
	"github.com/SamoySamoy/Soma/internal/modules/money"
	"github.com/SamoySamoy/Soma/internal/platform/apperr"
	"github.com/SamoySamoy/Soma/internal/platform/clock"
	"github.com/SamoySamoy/Soma/internal/platform/db"
	"github.com/SamoySamoy/Soma/internal/space"
	"github.com/SamoySamoy/Soma/internal/testutil/pgtest"
)

// env is one test's database with the local space set up and seeded.
type env struct {
	app   *db.DB
	owner *db.DB
	svc   *money.Service
	clk   *clock.Fake
}

func newEnv(t *testing.T) env {
	t.Helper()
	dbs := pgtest.New(t)
	app, err := db.Open(t.Context(), dbs.AppURL)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(app.Close)
	owner, err := db.Open(t.Context(), dbs.OwnerURL)
	if err != nil {
		t.Fatalf("open owner: %v", err)
	}
	t.Cleanup(owner.Close)
	clk := clock.NewFake(time.Date(2026, 10, 7, 9, 0, 0, 0, time.UTC))
	if err := space.EnsureLocal(t.Context(), app, clk); err != nil {
		t.Fatalf("EnsureLocal: %v", err)
	}
	if err := money.SeedDefaults(t.Context(), app, clk); err != nil {
		t.Fatalf("SeedDefaults: %v", err)
	}
	return env{app: app, owner: owner, svc: money.NewService(app, clk), clk: clk}
}

func as(ctx context.Context, user uuid.UUID) context.Context {
	return space.WithSpace(db.WithUserID(ctx, user), space.LocalSpaceID)
}

func owner(t *testing.T) context.Context {
	t.Helper()
	return as(t.Context(), space.LocalOwnerID)
}

func (e env) account(ctx context.Context, t *testing.T, name, currency string, opening int64) money.Account {
	t.Helper()
	a, err := e.svc.CreateAccount(ctx, money.AccountInput{
		Name: name, Kind: "bank", Currency: currency, OpeningMinor: opening, OpenedOn: "2026-01-01",
	})
	if err != nil {
		t.Fatalf("CreateAccount(%s): %v", name, err)
	}
	return a
}

// category finds a seeded category by name.
func (e env) category(ctx context.Context, t *testing.T, name string) money.Category {
	t.Helper()
	cats, err := e.svc.Categories(ctx)
	if err != nil {
		t.Fatalf("Categories: %v", err)
	}
	for _, c := range cats {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("no category %q", name)
	return money.Category{}
}

func kindOf(err error) apperr.Kind { return apperr.KindOf(err) }

func TestBalancesFollowTransactions(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := owner(t)
	cash := e.account(ctx, t, "Wallet", "VND", 1_000_000)
	food := e.category(ctx, t, "Groceries")
	salary := e.category(ctx, t, "Salary")

	if _, err := e.svc.RecordTransaction(ctx, money.TransactionInput{
		Kind: money.KindIncome, AccountID: cash.ID, Amount: 500_000, OccurredOn: "2026-10-05", CategoryID: &salary.ID,
	}); err != nil {
		t.Fatalf("income: %v", err)
	}
	if _, err := e.svc.RecordTransaction(ctx, money.TransactionInput{
		Kind: money.KindExpense, AccountID: cash.ID, Amount: 125_000, OccurredOn: "2026-10-06", CategoryID: &food.ID,
	}); err != nil {
		t.Fatalf("expense: %v", err)
	}

	got, err := e.svc.Account(ctx, cash.ID)
	if err != nil {
		t.Fatalf("Account: %v", err)
	}
	// 1,000,000 + 500,000 - 125,000 = 1,375,000 VND.
	if got.BalanceMinor != 1_375_000 {
		t.Errorf("balance = %d, want 1375000", got.BalanceMinor)
	}
}

// TestTransferMovesBothSidesTogether checks that a transfer writes two linked
// rows that move the money, and that deleting one side removes both.
func TestTransferMovesBothSidesTogether(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := owner(t)
	from := e.account(ctx, t, "Checking", "VND", 2_000_000)
	to := e.account(ctx, t, "Savings", "VND", 0)

	out, err := e.svc.RecordTransaction(ctx, money.TransactionInput{
		Kind: money.KindTransfer, AccountID: from.ID, ToAccountID: &to.ID, Amount: 300_000, OccurredOn: "2026-10-06",
	})
	if err != nil {
		t.Fatalf("transfer: %v", err)
	}
	if out.Amount != -300_000 || out.TransferID == nil {
		t.Fatalf("outgoing side = %+v, want -300000 with a transfer id", out)
	}
	if a, _ := e.svc.Account(ctx, from.ID); a.BalanceMinor != 1_700_000 {
		t.Errorf("checking = %d, want 1700000", a.BalanceMinor)
	}
	if a, _ := e.svc.Account(ctx, to.ID); a.BalanceMinor != 300_000 {
		t.Errorf("savings = %d, want 300000", a.BalanceMinor)
	}

	if err := e.svc.DeleteTransaction(ctx, out.ID); err != nil {
		t.Fatalf("delete transfer: %v", err)
	}
	if a, _ := e.svc.Account(ctx, from.ID); a.BalanceMinor != 2_000_000 {
		t.Errorf("checking after delete = %d, want 2000000", a.BalanceMinor)
	}
	if a, _ := e.svc.Account(ctx, to.ID); a.BalanceMinor != 0 {
		t.Errorf("savings after delete = %d, want 0", a.BalanceMinor)
	}
}

func TestTransferBetweenCurrenciesIsRefused(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := owner(t)
	vnd := e.account(ctx, t, "Wallet", "VND", 1_000_000)
	usd := e.account(ctx, t, "Card", "USD", 0)
	_, err := e.svc.RecordTransaction(ctx, money.TransactionInput{
		Kind: money.KindTransfer, AccountID: vnd.ID, ToAccountID: &usd.ID, Amount: 1000, OccurredOn: "2026-10-06",
	})
	if kindOf(err) != apperr.KindInvalid {
		t.Errorf("cross-currency transfer: kind = %v, want Invalid (err %v)", kindOf(err), err)
	}
}

func TestEditKeepsSignAndTransfersAreLocked(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := owner(t)
	cash := e.account(ctx, t, "Wallet", "VND", 0)
	to := e.account(ctx, t, "Savings", "VND", 0)
	food := e.category(ctx, t, "Groceries")

	spent, err := e.svc.RecordTransaction(ctx, money.TransactionInput{
		Kind: money.KindExpense, AccountID: cash.ID, Amount: 50_000, OccurredOn: "2026-10-06", CategoryID: &food.ID,
	})
	if err != nil {
		t.Fatalf("expense: %v", err)
	}
	edited, err := e.svc.UpdateTransaction(ctx, spent.ID, spent.Version, money.Patch{"amount_minor": 70_000.0})
	if err != nil {
		t.Fatalf("UpdateTransaction: %v", err)
	}
	if edited.Amount != -70_000 || edited.Version != 2 {
		t.Errorf("edited = %+v, want -70000 at version 2", edited)
	}

	transfer, err := e.svc.RecordTransaction(ctx, money.TransactionInput{
		Kind: money.KindTransfer, AccountID: cash.ID, ToAccountID: &to.ID, Amount: 10_000, OccurredOn: "2026-10-06",
	})
	if err != nil {
		t.Fatalf("transfer: %v", err)
	}
	if _, err := e.svc.UpdateTransaction(ctx, transfer.ID, transfer.Version, money.Patch{"notes": "x"}); kindOf(err) != apperr.KindInvalid {
		t.Errorf("editing a transfer: kind = %v, want Invalid", kindOf(err))
	}
}

func TestBudgetsCountSubcategoriesAndFlagOverspending(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := owner(t)
	cash := e.account(ctx, t, "Wallet", "VND", 5_000_000)
	food := e.category(ctx, t, "Food")
	groceries := e.category(ctx, t, "Groceries")
	month := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	if _, err := e.svc.SetBudget(ctx, food.ID, month, 400_000); err != nil {
		t.Fatalf("SetBudget: %v", err)
	}
	if _, err := e.svc.RecordTransaction(ctx, money.TransactionInput{
		Kind: money.KindExpense, AccountID: cash.ID, Amount: 300_000, OccurredOn: "2026-10-03", CategoryID: &groceries.ID,
	}); err != nil {
		t.Fatalf("grocery expense: %v", err)
	}
	if _, err := e.svc.RecordTransaction(ctx, money.TransactionInput{
		Kind: money.KindExpense, AccountID: cash.ID, Amount: 200_000, OccurredOn: "2026-09-30", CategoryID: &groceries.ID,
	}); err != nil {
		t.Fatalf("last month's expense: %v", err)
	}

	_, lines, err := e.svc.Budgets(ctx, month)
	if err != nil {
		t.Fatalf("Budgets: %v", err)
	}
	var foodLine *money.BudgetLine
	for i := range lines {
		if lines[i].CategoryID == food.ID {
			foodLine = &lines[i]
		}
	}
	if foodLine == nil {
		t.Fatal("no budget line for Food")
	}
	// Only this month's grocery spending counts toward Food.
	if foodLine.SpentMinor != 300_000 || foodLine.BudgetMinor != 400_000 {
		t.Errorf("food = spent %d budget %d, want 300000 of 400000", foodLine.SpentMinor, foodLine.BudgetMinor)
	}

	sum, err := e.svc.Summary(ctx, month)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if sum.ExpenseMinor != 300_000 || sum.OverspentCategories != 0 {
		t.Errorf("summary expense = %d, overspent = %d; want 300000 and 0", sum.ExpenseMinor, sum.OverspentCategories)
	}

	if _, err := e.svc.SetBudget(ctx, food.ID, month, 250_000); err != nil {
		t.Fatalf("lower budget: %v", err)
	}
	sum, err = e.svc.Summary(ctx, month)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if sum.OverspentCategories != 1 {
		t.Errorf("overspent = %d, want 1", sum.OverspentCategories)
	}
}

// TestForeignAccountsNeedARate checks that net worth never guesses: without a
// rate, the currency is listed as missing; with one, it converts exactly.
func TestForeignAccountsNeedARate(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := owner(t)
	e.account(ctx, t, "Wallet", "VND", 1_000_000)
	usd := e.account(ctx, t, "Card", "USD", 10_000) // 100.00 USD

	sum, err := e.svc.Summary(ctx, e.clk.Now())
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if len(sum.MissingRates) != 1 || sum.MissingRates[0] != "USD" {
		t.Fatalf("missing rates = %v, want [USD]", sum.MissingRates)
	}
	if sum.NetWorthMinor != 1_000_000 {
		t.Errorf("net worth = %d, want the VND account alone (1000000)", sum.NetWorthMinor)
	}

	// 1 USD = 25,000 VND.
	if _, err := e.svc.SetRate(ctx, "USD", "2026-10-06", 25_000*100_000_000); err != nil {
		t.Fatalf("SetRate: %v", err)
	}
	sum, err = e.svc.Summary(ctx, e.clk.Now())
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if len(sum.MissingRates) != 0 || sum.NetWorthMinor != 1_000_000+2_500_000 {
		t.Errorf("after rate: missing %v, net worth %d; want none and 3500000", sum.MissingRates, sum.NetWorthMinor)
	}
	for _, a := range sum.Accounts {
		if a.ID == usd.ID && (a.BaseMinor == nil || *a.BaseMinor != 2_500_000) {
			t.Errorf("USD account base value = %v, want 2500000", a.BaseMinor)
		}
	}
}

func TestViewerCannotRecordMoney(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := owner(t)
	cash := e.account(ctx, t, "Wallet", "VND", 0)

	viewer := uuid.New()
	now := e.clk.Now()
	if err := e.owner.InTx(t.Context(), func(tx pgx.Tx) error {
		q := corestore.New(tx)
		if err := q.UpsertUser(t.Context(), corestore.UpsertUserParams{ID: viewer, DisplayName: "V", Now: now}); err != nil {
			return err
		}
		return q.InsertSpaceMember(t.Context(), corestore.InsertSpaceMemberParams{
			SpaceID: space.LocalSpaceID, UserID: viewer, Role: "viewer", Now: now,
		})
	}); err != nil {
		t.Fatalf("seed viewer: %v", err)
	}

	vctx := as(t.Context(), viewer)
	if _, err := e.svc.RecordTransaction(vctx, money.TransactionInput{
		Kind: money.KindIncome, AccountID: cash.ID, Amount: 1, OccurredOn: "2026-10-06",
	}); kindOf(err) != apperr.KindForbidden {
		t.Errorf("viewer record: kind = %v, want Forbidden (err %v)", kindOf(err), err)
	}
	if _, err := e.svc.Summary(vctx, now); err != nil {
		t.Errorf("viewer summary: %v, want success", err)
	}
}

// Regression: an empty space must still produce a summary and budgets.
func TestSummaryOfAnEmptySpace(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := owner(t)
	sum, err := e.svc.Summary(ctx, e.clk.Now())
	if err != nil {
		t.Fatalf("Summary of an empty space: %v", err)
	}
	if sum.NetWorthMinor != 0 || sum.BaseCurrency != "VND" {
		t.Errorf("empty summary = %+v", sum)
	}
	if _, lines, err := e.svc.Budgets(ctx, e.clk.Now()); err != nil || len(lines) == 0 {
		t.Errorf("empty budgets = %d lines, %v; want the default categories with no budget", len(lines), err)
	}
}
