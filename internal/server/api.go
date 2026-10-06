package server

import (
	"context"

	"github.com/SamoySamoy/Soma/internal/apigen"
	"github.com/SamoySamoy/Soma/internal/bodymap"
	"github.com/SamoySamoy/Soma/internal/buildinfo"
	"github.com/SamoySamoy/Soma/internal/modules/journal"
	"github.com/SamoySamoy/Soma/internal/modules/money"
	"github.com/SamoySamoy/Soma/internal/modules/people"
	"github.com/SamoySamoy/Soma/internal/modules/self"
	"github.com/SamoySamoy/Soma/internal/modules/tasks"
	"github.com/SamoySamoy/Soma/internal/platform/config"
)

// api implements apigen.StrictServerInterface by delegating each operation to
// the handler of the module that owns it. Adding a module means one field and
// one method here.
type api struct {
	cfg     config.Config
	people  *people.Handler
	bodymap *bodymap.Handler
	journal *journal.Handler
	tasks   *tasks.Handler
	self    *self.Handler
	money   *money.Handler
}

var _ apigen.StrictServerInterface = (*api)(nil)

// GetMeta returns public information about this instance.
func (a *api) GetMeta(_ context.Context, _ apigen.GetMetaRequestObject) (apigen.GetMetaResponseObject, error) {
	return apigen.GetMeta200JSONResponse{
		Name:    "Soma",
		Version: buildinfo.Version,
		Mode:    apigen.MetaMode(a.cfg.Mode),
	}, nil
}

// GetBodyMap implements apigen.StrictServerInterface.
func (a *api) GetBodyMap(ctx context.Context, req apigen.GetBodyMapRequestObject) (apigen.GetBodyMapResponseObject, error) {
	return a.bodymap.GetBodyMap(ctx, req)
}

// ListContacts implements apigen.StrictServerInterface.
func (a *api) ListContacts(ctx context.Context, req apigen.ListContactsRequestObject) (apigen.ListContactsResponseObject, error) {
	return a.people.ListContacts(ctx, req)
}

// CreateContact implements apigen.StrictServerInterface.
func (a *api) CreateContact(ctx context.Context, req apigen.CreateContactRequestObject) (apigen.CreateContactResponseObject, error) {
	return a.people.CreateContact(ctx, req)
}

// GetContact implements apigen.StrictServerInterface.
func (a *api) GetContact(ctx context.Context, req apigen.GetContactRequestObject) (apigen.GetContactResponseObject, error) {
	return a.people.GetContact(ctx, req)
}

// UpdateContact implements apigen.StrictServerInterface.
func (a *api) UpdateContact(ctx context.Context, req apigen.UpdateContactRequestObject) (apigen.UpdateContactResponseObject, error) {
	return a.people.UpdateContact(ctx, req)
}

// DeleteContact implements apigen.StrictServerInterface.
func (a *api) DeleteContact(ctx context.Context, req apigen.DeleteContactRequestObject) (apigen.DeleteContactResponseObject, error) {
	return a.people.DeleteContact(ctx, req)
}

// ListJournalEntries implements apigen.StrictServerInterface.
func (a *api) ListJournalEntries(ctx context.Context, req apigen.ListJournalEntriesRequestObject) (apigen.ListJournalEntriesResponseObject, error) {
	return a.journal.ListJournalEntries(ctx, req)
}

// CreateJournalEntry implements apigen.StrictServerInterface.
func (a *api) CreateJournalEntry(ctx context.Context, req apigen.CreateJournalEntryRequestObject) (apigen.CreateJournalEntryResponseObject, error) {
	return a.journal.CreateJournalEntry(ctx, req)
}

// GetJournalEntry implements apigen.StrictServerInterface.
func (a *api) GetJournalEntry(ctx context.Context, req apigen.GetJournalEntryRequestObject) (apigen.GetJournalEntryResponseObject, error) {
	return a.journal.GetJournalEntry(ctx, req)
}

// UpdateJournalEntry implements apigen.StrictServerInterface.
func (a *api) UpdateJournalEntry(ctx context.Context, req apigen.UpdateJournalEntryRequestObject) (apigen.UpdateJournalEntryResponseObject, error) {
	return a.journal.UpdateJournalEntry(ctx, req)
}

// DeleteJournalEntry implements apigen.StrictServerInterface.
func (a *api) DeleteJournalEntry(ctx context.Context, req apigen.DeleteJournalEntryRequestObject) (apigen.DeleteJournalEntryResponseObject, error) {
	return a.journal.DeleteJournalEntry(ctx, req)
}

// ListTasks implements apigen.StrictServerInterface.
func (a *api) ListTasks(ctx context.Context, req apigen.ListTasksRequestObject) (apigen.ListTasksResponseObject, error) {
	return a.tasks.ListTasks(ctx, req)
}

// CreateTask implements apigen.StrictServerInterface.
func (a *api) CreateTask(ctx context.Context, req apigen.CreateTaskRequestObject) (apigen.CreateTaskResponseObject, error) {
	return a.tasks.CreateTask(ctx, req)
}

// GetTask implements apigen.StrictServerInterface.
func (a *api) GetTask(ctx context.Context, req apigen.GetTaskRequestObject) (apigen.GetTaskResponseObject, error) {
	return a.tasks.GetTask(ctx, req)
}

// UpdateTask implements apigen.StrictServerInterface.
func (a *api) UpdateTask(ctx context.Context, req apigen.UpdateTaskRequestObject) (apigen.UpdateTaskResponseObject, error) {
	return a.tasks.UpdateTask(ctx, req)
}

// DeleteTask implements apigen.StrictServerInterface.
func (a *api) DeleteTask(ctx context.Context, req apigen.DeleteTaskRequestObject) (apigen.DeleteTaskResponseObject, error) {
	return a.tasks.DeleteTask(ctx, req)
}

// CompleteTask implements apigen.StrictServerInterface.
func (a *api) CompleteTask(ctx context.Context, req apigen.CompleteTaskRequestObject) (apigen.CompleteTaskResponseObject, error) {
	return a.tasks.CompleteTask(ctx, req)
}

// ReopenTask implements apigen.StrictServerInterface.
func (a *api) ReopenTask(ctx context.Context, req apigen.ReopenTaskRequestObject) (apigen.ReopenTaskResponseObject, error) {
	return a.tasks.ReopenTask(ctx, req)
}

// GetSelfProfile implements apigen.StrictServerInterface.
func (a *api) GetSelfProfile(ctx context.Context, req apigen.GetSelfProfileRequestObject) (apigen.GetSelfProfileResponseObject, error) {
	return a.self.GetSelfProfile(ctx, req)
}

// UpdateSelfProfile implements apigen.StrictServerInterface.
func (a *api) UpdateSelfProfile(ctx context.Context, req apigen.UpdateSelfProfileRequestObject) (apigen.UpdateSelfProfileResponseObject, error) {
	return a.self.UpdateSelfProfile(ctx, req)
}

// ListCurrencies implements apigen.StrictServerInterface.
func (a *api) ListCurrencies(ctx context.Context, req apigen.ListCurrenciesRequestObject) (apigen.ListCurrenciesResponseObject, error) {
	return a.money.ListCurrencies(ctx, req)
}

// GetMoneySettings implements apigen.StrictServerInterface.
func (a *api) GetMoneySettings(ctx context.Context, req apigen.GetMoneySettingsRequestObject) (apigen.GetMoneySettingsResponseObject, error) {
	return a.money.GetMoneySettings(ctx, req)
}

// UpdateMoneySettings implements apigen.StrictServerInterface.
func (a *api) UpdateMoneySettings(ctx context.Context, req apigen.UpdateMoneySettingsRequestObject) (apigen.UpdateMoneySettingsResponseObject, error) {
	return a.money.UpdateMoneySettings(ctx, req)
}

// ListAccounts implements apigen.StrictServerInterface.
func (a *api) ListAccounts(ctx context.Context, req apigen.ListAccountsRequestObject) (apigen.ListAccountsResponseObject, error) {
	return a.money.ListAccounts(ctx, req)
}

// CreateAccount implements apigen.StrictServerInterface.
func (a *api) CreateAccount(ctx context.Context, req apigen.CreateAccountRequestObject) (apigen.CreateAccountResponseObject, error) {
	return a.money.CreateAccount(ctx, req)
}

// GetAccount implements apigen.StrictServerInterface.
func (a *api) GetAccount(ctx context.Context, req apigen.GetAccountRequestObject) (apigen.GetAccountResponseObject, error) {
	return a.money.GetAccount(ctx, req)
}

// UpdateAccount implements apigen.StrictServerInterface.
func (a *api) UpdateAccount(ctx context.Context, req apigen.UpdateAccountRequestObject) (apigen.UpdateAccountResponseObject, error) {
	return a.money.UpdateAccount(ctx, req)
}

// DeleteAccount implements apigen.StrictServerInterface.
func (a *api) DeleteAccount(ctx context.Context, req apigen.DeleteAccountRequestObject) (apigen.DeleteAccountResponseObject, error) {
	return a.money.DeleteAccount(ctx, req)
}

// ListCategories implements apigen.StrictServerInterface.
func (a *api) ListCategories(ctx context.Context, req apigen.ListCategoriesRequestObject) (apigen.ListCategoriesResponseObject, error) {
	return a.money.ListCategories(ctx, req)
}

// CreateCategory implements apigen.StrictServerInterface.
func (a *api) CreateCategory(ctx context.Context, req apigen.CreateCategoryRequestObject) (apigen.CreateCategoryResponseObject, error) {
	return a.money.CreateCategory(ctx, req)
}

// ListTransactions implements apigen.StrictServerInterface.
func (a *api) ListTransactions(ctx context.Context, req apigen.ListTransactionsRequestObject) (apigen.ListTransactionsResponseObject, error) {
	return a.money.ListTransactions(ctx, req)
}

// CreateTransaction implements apigen.StrictServerInterface.
func (a *api) CreateTransaction(ctx context.Context, req apigen.CreateTransactionRequestObject) (apigen.CreateTransactionResponseObject, error) {
	return a.money.CreateTransaction(ctx, req)
}

// GetTransaction implements apigen.StrictServerInterface.
func (a *api) GetTransaction(ctx context.Context, req apigen.GetTransactionRequestObject) (apigen.GetTransactionResponseObject, error) {
	return a.money.GetTransaction(ctx, req)
}

// UpdateTransaction implements apigen.StrictServerInterface.
func (a *api) UpdateTransaction(ctx context.Context, req apigen.UpdateTransactionRequestObject) (apigen.UpdateTransactionResponseObject, error) {
	return a.money.UpdateTransaction(ctx, req)
}

// DeleteTransaction implements apigen.StrictServerInterface.
func (a *api) DeleteTransaction(ctx context.Context, req apigen.DeleteTransactionRequestObject) (apigen.DeleteTransactionResponseObject, error) {
	return a.money.DeleteTransaction(ctx, req)
}

// GetBudgets implements apigen.StrictServerInterface.
func (a *api) GetBudgets(ctx context.Context, req apigen.GetBudgetsRequestObject) (apigen.GetBudgetsResponseObject, error) {
	return a.money.GetBudgets(ctx, req)
}

// SetBudget implements apigen.StrictServerInterface.
func (a *api) SetBudget(ctx context.Context, req apigen.SetBudgetRequestObject) (apigen.SetBudgetResponseObject, error) {
	return a.money.SetBudget(ctx, req)
}

// ListRates implements apigen.StrictServerInterface.
func (a *api) ListRates(ctx context.Context, req apigen.ListRatesRequestObject) (apigen.ListRatesResponseObject, error) {
	return a.money.ListRates(ctx, req)
}

// SetRate implements apigen.StrictServerInterface.
func (a *api) SetRate(ctx context.Context, req apigen.SetRateRequestObject) (apigen.SetRateResponseObject, error) {
	return a.money.SetRate(ctx, req)
}

// GetMoneySummary implements apigen.StrictServerInterface.
func (a *api) GetMoneySummary(ctx context.Context, req apigen.GetMoneySummaryRequestObject) (apigen.GetMoneySummaryResponseObject, error) {
	return a.money.GetMoneySummary(ctx, req)
}
