package money

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/SamoySamoy/Soma/internal/platform/apperr"
)

// Account kinds (FIN-01). A loan is entered as a negative opening balance.
var accountKinds = map[string]bool{
	"cash": true, "bank": true, "credit_card": true, "e_wallet": true, "loan": true, "investment": true,
}

// Account is an account with its current balance in its own currency.
type Account struct {
	ID           uuid.UUID
	Name         string
	Kind         string
	Currency     string
	OpeningMinor int64
	OpenedOn     string
	BalanceMinor int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Version      int32
}

// AccountInput is the writable part of an account.
type AccountInput struct {
	Name         string
	Kind         string
	Currency     string
	OpeningMinor int64
	OpenedOn     string
}

// Category groups transactions and budgets. Categories form a two-level tree.
type Category struct {
	ID       uuid.UUID
	ParentID *uuid.UUID
	Name     string
	Kind     string // expense or income
}

// CategoryInput is the writable part of a category.
type CategoryInput struct {
	Name     string
	Kind     string
	ParentID *uuid.UUID
}

// Transaction kinds as the API accepts them when recording.
const (
	KindIncome   = "income"
	KindExpense  = "expense"
	KindTransfer = "transfer"
)

// Transaction is one row on an account. Amount is signed: income and transfers
// in are positive, expenses and transfers out negative.
type Transaction struct {
	ID         uuid.UUID
	AccountID  uuid.UUID
	Kind       string
	Amount     int64
	OccurredOn string
	CategoryID *uuid.UUID
	Payee      string
	Notes      string
	TransferID *uuid.UUID
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Version    int32
}

// TransactionInput records a new income, expense or transfer. Amount is positive.
type TransactionInput struct {
	Kind        string
	AccountID   uuid.UUID
	ToAccountID *uuid.UUID
	Amount      int64
	OccurredOn  string
	CategoryID  *uuid.UUID
	Payee       string
	Notes       string
}

// Settings are the space's money settings.
type Settings struct {
	BaseCurrency string
	UpdatedAt    time.Time
	Version      int32
}

// Rate is the exchange rate of one currency on one day.
type Rate struct {
	Currency string
	RateOn   string
	RateE8   int64
}

// BudgetLine is one category's budget and spending for a month, in the base
// currency. Spending includes subcategories.
type BudgetLine struct {
	CategoryID  uuid.UUID
	Name        string
	ParentID    *uuid.UUID
	BudgetMinor int64
	SpentMinor  int64
}

// Patch is a JSON Merge Patch. A missing key keeps the value; null clears it.
type Patch map[string]any

// Page is one page of transactions. Next is the cursor for the next page.
type Page struct {
	Items []Transaction
	Next  string
}

func invalid(fields map[string]string) error {
	return apperr.Invalid("money.invalid", "Some fields need attention.", fields)
}

func validDate(s string) bool {
	_, err := time.Parse(time.DateOnly, s)
	return err == nil
}

func validCode(s string) bool { return Supported(s) }

func (in AccountInput) normalized() AccountInput {
	in.Name = strings.TrimSpace(in.Name)
	in.OpenedOn = strings.TrimSpace(in.OpenedOn)
	return in
}

func (in AccountInput) validate() error {
	f := map[string]string{}
	if n := utf8.RuneCountInString(in.Name); n < 1 || n > 100 {
		f["name"] = "Name must be 1 to 100 characters."
	}
	if !accountKinds[in.Kind] {
		f["kind"] = "Choose a kind of account."
	}
	if !validCode(in.Currency) {
		f["currency"] = "Choose a supported currency."
	}
	if !validDate(in.OpenedOn) {
		f["opened_on"] = "Use the format YYYY-MM-DD."
	}
	if len(f) > 0 {
		return invalid(f)
	}
	return nil
}

// apply changes the editable fields of cur. The currency never changes.
func (in AccountInput) apply(p Patch) (AccountInput, error) {
	bad := map[string]string{}
	for key, v := range p {
		switch key {
		case "name":
			s, ok := textOf(v)
			if !ok {
				bad[key] = "Use text."
				continue
			}
			in.Name = s
		case "kind":
			s, ok := textOf(v)
			if !ok {
				bad[key] = "Use text."
				continue
			}
			in.Kind = s
		case "opening_minor":
			n, ok := integerOf(v)
			if !ok {
				bad[key] = "Use a whole number."
				continue
			}
			in.OpeningMinor = n
		case "opened_on":
			s, ok := textOf(v)
			if !ok {
				bad[key] = "Use text."
				continue
			}
			in.OpenedOn = s
		default:
			bad[key] = "This field can't be changed."
		}
	}
	if len(bad) > 0 {
		return AccountInput{}, invalid(bad)
	}
	return in, nil
}

func (in CategoryInput) normalized() CategoryInput {
	in.Name = strings.TrimSpace(in.Name)
	return in
}

func (in CategoryInput) validate() error {
	f := map[string]string{}
	if n := utf8.RuneCountInString(in.Name); n < 1 || n > 60 {
		f["name"] = "Name must be 1 to 60 characters."
	}
	if in.Kind != "expense" && in.Kind != "income" {
		f["kind"] = "Choose expense or income."
	}
	if len(f) > 0 {
		return invalid(f)
	}
	return nil
}

// validate checks a transaction before it is recorded. Amounts must be positive:
// the kind sets the sign.
func (in TransactionInput) validate() error {
	f := map[string]string{}
	switch in.Kind {
	case KindIncome, KindExpense, KindTransfer:
	default:
		f["kind"] = "Choose income, expense or transfer."
	}
	if in.Amount < 1 {
		f["amount_minor"] = "Enter an amount greater than zero."
	}
	if !validDate(in.OccurredOn) {
		f["occurred_on"] = "Use the format YYYY-MM-DD."
	}
	if in.Kind == KindTransfer {
		if in.ToAccountID == nil {
			f["to_account_id"] = "Choose the account to receive the money."
		} else if *in.ToAccountID == in.AccountID {
			f["to_account_id"] = "Choose a different account."
		}
		if in.CategoryID != nil {
			f["category_id"] = "Transfers don't have a category."
		}
	}
	if utf8Len(in.Payee) > 120 {
		f["payee"] = "Payee can be at most 120 characters."
	}
	if utf8Len(in.Notes) > 2000 {
		f["notes"] = "Notes can be at most 2000 characters."
	}
	if len(f) > 0 {
		return invalid(f)
	}
	return nil
}

// apply changes the editable fields of a transaction. The kind and transfers
// can't change, so the sign stays with the kind.
func (cur Transaction) apply(p Patch) (Transaction, error) {
	bad := map[string]string{}
	for key, v := range p {
		switch key {
		case "amount_minor":
			n, ok := integerOf(v)
			if !ok || n < 1 {
				bad[key] = "Enter an amount greater than zero."
				continue
			}
			cur.Amount = n
		case "occurred_on":
			s, ok := textOf(v)
			if !ok || !validDate(s) {
				bad[key] = "Use the format YYYY-MM-DD."
				continue
			}
			cur.OccurredOn = s
		case "category_id":
			id, ok := uuidOf(v)
			if !ok {
				bad[key] = "Choose a category."
				continue
			}
			cur.CategoryID = id
		case "payee":
			s, ok := textOf(v)
			if !ok || utf8Len(s) > 120 {
				bad[key] = "Payee can be at most 120 characters."
				continue
			}
			cur.Payee = s
		case "notes":
			s, ok := textOf(v)
			if !ok || utf8Len(s) > 2000 {
				bad[key] = "Notes can be at most 2000 characters."
				continue
			}
			cur.Notes = s
		default:
			bad[key] = "This field can't be changed."
		}
	}
	if len(bad) > 0 {
		return Transaction{}, invalid(bad)
	}
	if (cur.Kind == "transfer_in" || cur.Kind == "transfer_out") && len(p) > 0 {
		return Transaction{}, apperr.Invalid("money.transfer_locked",
			"Transfers can't be edited. Delete it and record it again.", nil)
	}
	return cur, nil
}

// signed returns amount with the sign that its kind requires.
func signed(kind string, amount int64) int64 {
	if kind == KindExpense || kind == "transfer_out" {
		return -amount
	}
	return amount
}

// sign returns -1 or 1 for a stored amount, so an edited amount keeps its sign.
func sign(amount int64) int64 {
	if amount < 0 {
		return -1
	}
	return 1
}

func (s Settings) validate(base string) error {
	if !validCode(base) {
		return invalid(map[string]string{"base_currency": "Choose a supported currency."})
	}
	return nil
}

func validRate(currency string, rateE8 int64, on string) error {
	f := map[string]string{}
	if !validCode(currency) {
		f["currency"] = "Choose a supported currency."
	}
	if rateE8 < 1 {
		f["rate_e8"] = "The rate must be greater than zero."
	}
	if !validDate(on) {
		f["rate_on"] = "Use the format YYYY-MM-DD."
	}
	if len(f) > 0 {
		return invalid(f)
	}
	return nil
}

// monthBounds returns the first day of t's month and the first day of the next.
func monthBounds(t time.Time) (time.Time, time.Time) {
	start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	return start, start.AddDate(0, 1, 0)
}

func parseMonth(s string) (time.Time, error) {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return time.Time{}, invalid(map[string]string{"month": "Use the format YYYY-MM-DD."})
	}
	return t, nil
}

func utf8Len(s string) int { return utf8.RuneCountInString(s) }

func textOf(v any) (string, bool) {
	switch x := v.(type) {
	case nil:
		return "", true
	case string:
		return strings.TrimSpace(x), true
	default:
		return "", false
	}
}

// integerOf reads a JSON number that must be a whole number.
func integerOf(v any) (int64, bool) {
	f, ok := v.(float64)
	if !ok || f != float64(int64(f)) {
		return 0, false
	}
	return int64(f), true
}

// uuidOf reads a JSON string holding a UUID; null means no value.
func uuidOf(v any) (*uuid.UUID, bool) {
	if v == nil {
		return nil, true
	}
	s, ok := v.(string)
	if !ok {
		return nil, false
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil, false
	}
	return &id, true
}

func formatMonth(t time.Time) string { return fmt.Sprintf("%04d-%02d-01", t.Year(), t.Month()) }
