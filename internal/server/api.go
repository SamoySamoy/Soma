package server

import (
	"context"

	"github.com/SamoySamoy/Soma/internal/apigen"
	"github.com/SamoySamoy/Soma/internal/bodymap"
	"github.com/SamoySamoy/Soma/internal/buildinfo"
	"github.com/SamoySamoy/Soma/internal/modules/people"
	"github.com/SamoySamoy/Soma/internal/platform/config"
)

// api implements apigen.StrictServerInterface by delegating each operation to
// the handler of the module that owns it. Adding a module means one field and
// one method here.
type api struct {
	cfg     config.Config
	people  *people.Handler
	bodymap *bodymap.Handler
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
