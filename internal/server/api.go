package server

import (
	"context"

	"github.com/SamoySamoy/Soma/internal/apigen"
	"github.com/SamoySamoy/Soma/internal/bodymap"
	"github.com/SamoySamoy/Soma/internal/buildinfo"
	"github.com/SamoySamoy/Soma/internal/modules/journal"
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
