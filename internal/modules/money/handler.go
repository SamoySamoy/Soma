package money

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/SamoySamoy/Soma/internal/apigen"
	"github.com/SamoySamoy/Soma/internal/platform/apperr"
)

const (
	defaultPageSize = 50
	maxPageSize     = 200
)

// Handler adapts the generated strict API to the money services.
type Handler struct {
	svc *Service
}

// NewHandler returns a Handler for svc.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ListCurrencies implements apigen.StrictServerInterface.
func (h *Handler) ListCurrencies(_ context.Context, _ apigen.ListCurrenciesRequestObject) (apigen.ListCurrenciesResponseObject, error) {
	out := apigen.ListCurrencies200JSONResponse{Items: []apigen.Currency{}}
	for _, c := range Currencies() {
		out.Items = append(out.Items, apigen.Currency{Code: c.Code, Digits: c.Digits})
	}
	return out, nil
}

// GetMoneySettings implements apigen.StrictServerInterface.
func (h *Handler) GetMoneySettings(ctx context.Context, _ apigen.GetMoneySettingsRequestObject) (apigen.GetMoneySettingsResponseObject, error) {
	st, err := h.svc.Settings(ctx)
	if err != nil {
		return nil, err
	}
	return apigen.GetMoneySettings200JSONResponse{
		Body:    settingsToAPI(st),
		Headers: apigen.GetMoneySettings200ResponseHeaders{ETag: etag(st.Version)},
	}, nil
}

// UpdateMoneySettings implements apigen.StrictServerInterface.
func (h *Handler) UpdateMoneySettings(ctx context.Context, req apigen.UpdateMoneySettingsRequestObject) (apigen.UpdateMoneySettingsResponseObject, error) {
	version, err := parseIfMatch(req.Params.IfMatch)
	if err != nil {
		return nil, err
	}
	if req.Body == nil {
		return nil, apperr.Invalid("money.invalid", "Send the changes as JSON.", nil)
	}
	st, err := h.svc.UpdateSettings(ctx, version, Patch(*req.Body))
	if err != nil {
		return nil, err
	}
	return apigen.UpdateMoneySettings200JSONResponse{
		Body:    settingsToAPI(st),
		Headers: apigen.UpdateMoneySettings200ResponseHeaders{ETag: etag(st.Version)},
	}, nil
}

// ListAccounts implements apigen.StrictServerInterface.
func (h *Handler) ListAccounts(ctx context.Context, _ apigen.ListAccountsRequestObject) (apigen.ListAccountsResponseObject, error) {
	accounts, err := h.svc.Accounts(ctx)
	if err != nil {
		return nil, err
	}
	out := apigen.ListAccounts200JSONResponse{Items: []apigen.Account{}}
	for _, a := range accounts {
		out.Items = append(out.Items, accountToAPI(a))
	}
	return out, nil
}

// CreateAccount implements apigen.StrictServerInterface.
func (h *Handler) CreateAccount(ctx context.Context, req apigen.CreateAccountRequestObject) (apigen.CreateAccountResponseObject, error) {
	if req.Body == nil {
		return nil, apperr.Invalid("money.invalid", "Send the account as JSON.", nil)
	}
	in := AccountInput{
		Name:     req.Body.Name,
		Kind:     string(req.Body.Kind),
		Currency: req.Body.Currency,
		OpenedOn: req.Body.OpenedOn,
	}
	if req.Body.OpeningMinor != nil {
		in.OpeningMinor = int64(*req.Body.OpeningMinor)
	}
	a, err := h.svc.CreateAccount(ctx, in)
	if err != nil {
		return nil, err
	}
	return apigen.CreateAccount201JSONResponse{
		Body:    accountToAPI(a),
		Headers: apigen.CreateAccount201ResponseHeaders{ETag: etag(a.Version)},
	}, nil
}

// GetAccount implements apigen.StrictServerInterface.
func (h *Handler) GetAccount(ctx context.Context, req apigen.GetAccountRequestObject) (apigen.GetAccountResponseObject, error) {
	a, err := h.svc.Account(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return apigen.GetAccount200JSONResponse{
		Body:    accountToAPI(a),
		Headers: apigen.GetAccount200ResponseHeaders{ETag: etag(a.Version)},
	}, nil
}

// UpdateAccount implements apigen.StrictServerInterface.
func (h *Handler) UpdateAccount(ctx context.Context, req apigen.UpdateAccountRequestObject) (apigen.UpdateAccountResponseObject, error) {
	version, err := parseIfMatch(req.Params.IfMatch)
	if err != nil {
		return nil, err
	}
	if req.Body == nil {
		return nil, apperr.Invalid("money.invalid", "Send the changes as JSON.", nil)
	}
	a, err := h.svc.UpdateAccount(ctx, req.Id, version, Patch(*req.Body))
	if err != nil {
		return nil, err
	}
	return apigen.UpdateAccount200JSONResponse{
		Body:    accountToAPI(a),
		Headers: apigen.UpdateAccount200ResponseHeaders{ETag: etag(a.Version)},
	}, nil
}

// DeleteAccount implements apigen.StrictServerInterface.
func (h *Handler) DeleteAccount(ctx context.Context, req apigen.DeleteAccountRequestObject) (apigen.DeleteAccountResponseObject, error) {
	if err := h.svc.DeleteAccount(ctx, req.Id); err != nil {
		return nil, err
	}
	return apigen.DeleteAccount204Response{}, nil
}

// ListCategories implements apigen.StrictServerInterface.
func (h *Handler) ListCategories(ctx context.Context, _ apigen.ListCategoriesRequestObject) (apigen.ListCategoriesResponseObject, error) {
	cats, err := h.svc.Categories(ctx)
	if err != nil {
		return nil, err
	}
	out := apigen.ListCategories200JSONResponse{Items: []apigen.Category{}}
	for _, c := range cats {
		out.Items = append(out.Items, categoryToAPI(c))
	}
	return out, nil
}

// CreateCategory implements apigen.StrictServerInterface.
func (h *Handler) CreateCategory(ctx context.Context, req apigen.CreateCategoryRequestObject) (apigen.CreateCategoryResponseObject, error) {
	if req.Body == nil {
		return nil, apperr.Invalid("money.invalid", "Send the category as JSON.", nil)
	}
	in := CategoryInput{Name: req.Body.Name, Kind: string(req.Body.Kind)}
	if req.Body.ParentId != nil {
		id := *req.Body.ParentId
		in.ParentID = &id
	}
	c, err := h.svc.CreateCategory(ctx, in)
	if err != nil {
		return nil, err
	}
	return apigen.CreateCategory201JSONResponse(categoryToAPI(c)), nil
}

// ListTransactions implements apigen.StrictServerInterface.
func (h *Handler) ListTransactions(ctx context.Context, req apigen.ListTransactionsRequestObject) (apigen.ListTransactionsResponseObject, error) {
	limit := int32(defaultPageSize)
	if req.Params.Limit != nil {
		limit = int32(*req.Params.Limit) //nolint:gosec // G115: the range check below rejects overflow
	}
	if limit < 1 || limit > maxPageSize {
		return nil, apperr.Invalid("money.invalid_page", "Choose a page size between 1 and 200.",
			map[string]string{"limit": "Use a number from 1 to 200."})
	}
	var accountID *uuid.UUID
	if req.Params.AccountId != nil {
		id := *req.Params.AccountId
		accountID = &id
	}
	cursor := ""
	if req.Params.Cursor != nil {
		cursor = *req.Params.Cursor
	}
	page, err := h.svc.Transactions(ctx, limit, accountID, cursor)
	if err != nil {
		return nil, err
	}
	out := apigen.ListTransactions200JSONResponse{Items: []apigen.Transaction{}}
	for _, t := range page.Items {
		out.Items = append(out.Items, transactionToAPI(t))
	}
	if page.Next != "" {
		next := page.Next
		out.NextCursor = &next
	}
	return out, nil
}

// CreateTransaction implements apigen.StrictServerInterface.
func (h *Handler) CreateTransaction(ctx context.Context, req apigen.CreateTransactionRequestObject) (apigen.CreateTransactionResponseObject, error) {
	if req.Body == nil {
		return nil, apperr.Invalid("money.invalid", "Send the transaction as JSON.", nil)
	}
	b := req.Body
	in := TransactionInput{
		Kind:       string(b.Kind),
		AccountID:  b.AccountId,
		Amount:     int64(b.AmountMinor),
		OccurredOn: b.OccurredOn,
		Payee:      derefString(b.Payee),
		Notes:      derefString(b.Notes),
	}
	if b.ToAccountId != nil {
		id := *b.ToAccountId
		in.ToAccountID = &id
	}
	if b.CategoryId != nil {
		id := *b.CategoryId
		in.CategoryID = &id
	}
	t, err := h.svc.RecordTransaction(ctx, in)
	if err != nil {
		return nil, err
	}
	return apigen.CreateTransaction201JSONResponse{
		Body:    transactionToAPI(t),
		Headers: apigen.CreateTransaction201ResponseHeaders{ETag: etag(t.Version)},
	}, nil
}

// GetTransaction implements apigen.StrictServerInterface.
func (h *Handler) GetTransaction(ctx context.Context, req apigen.GetTransactionRequestObject) (apigen.GetTransactionResponseObject, error) {
	t, err := h.svc.Transaction(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return apigen.GetTransaction200JSONResponse{
		Body:    transactionToAPI(t),
		Headers: apigen.GetTransaction200ResponseHeaders{ETag: etag(t.Version)},
	}, nil
}

// UpdateTransaction implements apigen.StrictServerInterface.
func (h *Handler) UpdateTransaction(ctx context.Context, req apigen.UpdateTransactionRequestObject) (apigen.UpdateTransactionResponseObject, error) {
	version, err := parseIfMatch(req.Params.IfMatch)
	if err != nil {
		return nil, err
	}
	if req.Body == nil {
		return nil, apperr.Invalid("money.invalid", "Send the changes as JSON.", nil)
	}
	t, err := h.svc.UpdateTransaction(ctx, req.Id, version, Patch(*req.Body))
	if err != nil {
		return nil, err
	}
	return apigen.UpdateTransaction200JSONResponse{
		Body:    transactionToAPI(t),
		Headers: apigen.UpdateTransaction200ResponseHeaders{ETag: etag(t.Version)},
	}, nil
}

// DeleteTransaction implements apigen.StrictServerInterface.
func (h *Handler) DeleteTransaction(ctx context.Context, req apigen.DeleteTransactionRequestObject) (apigen.DeleteTransactionResponseObject, error) {
	if err := h.svc.DeleteTransaction(ctx, req.Id); err != nil {
		return nil, err
	}
	return apigen.DeleteTransaction204Response{}, nil
}

// GetBudgets implements apigen.StrictServerInterface.
func (h *Handler) GetBudgets(ctx context.Context, req apigen.GetBudgetsRequestObject) (apigen.GetBudgetsResponseObject, error) {
	month, err := h.monthParam(req.Params.Month)
	if err != nil {
		return nil, err
	}
	base, lines, err := h.svc.Budgets(ctx, month)
	if err != nil {
		return nil, err
	}
	out := apigen.GetBudgets200JSONResponse{
		Month:        formatMonth(monthStart(month)),
		BaseCurrency: base,
		Items:        []apigen.BudgetLine{},
	}
	for _, l := range lines {
		out.Items = append(out.Items, budgetLineToAPI(l))
	}
	return out, nil
}

// SetBudget implements apigen.StrictServerInterface.
func (h *Handler) SetBudget(ctx context.Context, req apigen.SetBudgetRequestObject) (apigen.SetBudgetResponseObject, error) {
	if req.Body == nil {
		return nil, apperr.Invalid("money.invalid", "Send the budget as JSON.", nil)
	}
	month, err := parseMonthDate(req.Body.Month)
	if err != nil {
		return nil, err
	}
	line, err := h.svc.SetBudget(ctx, req.Body.CategoryId, month, int64(req.Body.AmountMinor))
	if err != nil {
		return nil, err
	}
	return apigen.SetBudget200JSONResponse(budgetLineToAPI(line)), nil
}

// ListRates implements apigen.StrictServerInterface.
func (h *Handler) ListRates(ctx context.Context, _ apigen.ListRatesRequestObject) (apigen.ListRatesResponseObject, error) {
	rates, err := h.svc.LatestRates(ctx)
	if err != nil {
		return nil, err
	}
	out := apigen.ListRates200JSONResponse{Items: []apigen.Rate{}}
	for _, r := range rates {
		out.Items = append(out.Items, rateToAPI(r))
	}
	return out, nil
}

// SetRate implements apigen.StrictServerInterface.
func (h *Handler) SetRate(ctx context.Context, req apigen.SetRateRequestObject) (apigen.SetRateResponseObject, error) {
	if req.Body == nil {
		return nil, apperr.Invalid("money.invalid", "Send the rate as JSON.", nil)
	}
	r, err := h.svc.SetRate(ctx, req.Body.Currency, req.Body.RateOn, int64(req.Body.RateE8))
	if err != nil {
		return nil, err
	}
	return apigen.SetRate200JSONResponse(rateToAPI(r)), nil
}

// GetMoneySummary implements apigen.StrictServerInterface.
func (h *Handler) GetMoneySummary(ctx context.Context, req apigen.GetMoneySummaryRequestObject) (apigen.GetMoneySummaryResponseObject, error) {
	month, err := h.monthParam(req.Params.Month)
	if err != nil {
		return nil, err
	}
	sum, err := h.svc.Summary(ctx, month)
	if err != nil {
		return nil, err
	}
	return apigen.GetMoneySummary200JSONResponse(summaryToAPI(sum)), nil
}

// monthParam reads an optional month, defaulting to the current one.
func (h *Handler) monthParam(p *string) (time.Time, error) {
	if p == nil {
		return h.svc.Now(), nil
	}
	return parseMonth(*p)
}

// parseMonthDate reads a required date that names the month to use.
func parseMonthDate(s string) (time.Time, error) {
	return parseMonth(s)
}

// parseIfMatch reads the version from an If-Match header. Missing or malformed is 412.
func parseIfMatch(header *string) (int32, error) {
	missing := apperr.New(apperr.KindPreconditionFailed, "money.version_required",
		"Send If-Match with the version you are editing.")
	if header == nil {
		return 0, missing
	}
	v, err := strconv.ParseInt(strings.Trim(strings.TrimSpace(*header), `"`), 10, 32)
	if err != nil || v < 1 {
		return 0, missing
	}
	return int32(v), nil
}

func etag(version int32) *string {
	s := fmt.Sprintf(`"%d"`, version)
	return &s
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func settingsToAPI(s Settings) apigen.MoneySettings {
	return apigen.MoneySettings{
		BaseCurrency: s.BaseCurrency,
		UpdatedAt:    s.UpdatedAt,
		Version:      int(s.Version),
	}
}

func accountToAPI(a Account) apigen.Account {
	return apigen.Account{
		Id:           a.ID,
		Name:         a.Name,
		Kind:         apigen.AccountKind(a.Kind),
		Currency:     a.Currency,
		OpeningMinor: int(a.OpeningMinor),
		OpenedOn:     a.OpenedOn,
		BalanceMinor: int(a.BalanceMinor),
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
		Version:      int(a.Version),
	}
}

func categoryToAPI(c Category) apigen.Category {
	out := apigen.Category{Id: c.ID, Name: c.Name, Kind: apigen.CategoryKind(c.Kind)}
	out.ParentId = c.ParentID
	return out
}

func transactionToAPI(t Transaction) apigen.Transaction {
	out := apigen.Transaction{
		Id:          t.ID,
		AccountId:   t.AccountID,
		Kind:        apigen.TransactionKind(t.Kind),
		AmountMinor: int(t.Amount),
		OccurredOn:  t.OccurredOn,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		Version:     int(t.Version),
	}
	out.CategoryId = t.CategoryID
	out.TransferId = t.TransferID
	if t.Payee != "" {
		out.Payee = &t.Payee
	}
	if t.Notes != "" {
		out.Notes = &t.Notes
	}
	return out
}

func budgetLineToAPI(l BudgetLine) apigen.BudgetLine {
	out := apigen.BudgetLine{
		CategoryId:  l.CategoryID,
		Name:        l.Name,
		BudgetMinor: int(l.BudgetMinor),
		SpentMinor:  int(l.SpentMinor),
	}
	out.ParentId = l.ParentID
	return out
}

func rateToAPI(r Rate) apigen.Rate {
	return apigen.Rate{Currency: r.Currency, RateOn: r.RateOn, RateE8: int(r.RateE8)}
}

func summaryToAPI(s Summary) apigen.MoneySummary {
	out := apigen.MoneySummary{
		Month:               formatMonth(s.Month),
		BaseCurrency:        s.BaseCurrency,
		NetWorthMinor:       int(s.NetWorthMinor),
		IncomeMinor:         int(s.IncomeMinor),
		ExpenseMinor:        int(s.ExpenseMinor),
		OverspentCategories: s.OverspentCategories,
		MissingRates:        append([]string{}, s.MissingRates...),
		Accounts: []struct {
			BalanceMinor int       `json:"balance_minor"`
			BaseMinor    *int      `json:"base_minor,omitempty"`
			Currency     string    `json:"currency"`
			Id           uuid.UUID `json:"id"` //nolint:revive // matches the generated field name
			Name         string    `json:"name"`
		}{},
	}
	for _, a := range s.Accounts {
		row := struct {
			BalanceMinor int       `json:"balance_minor"`
			BaseMinor    *int      `json:"base_minor,omitempty"`
			Currency     string    `json:"currency"`
			Id           uuid.UUID `json:"id"` //nolint:revive // matches the generated field name
			Name         string    `json:"name"`
		}{BalanceMinor: int(a.BalanceMinor), Currency: a.Currency, Id: a.ID, Name: a.Name}
		if a.BaseMinor != nil {
			v := int(*a.BaseMinor)
			row.BaseMinor = &v
		}
		out.Accounts = append(out.Accounts, row)
	}
	return out
}
