package server

import (
	"context"

	"github.com/SamoySamoy/Soma/internal/apigen"
	"github.com/SamoySamoy/Soma/internal/buildinfo"
	"github.com/SamoySamoy/Soma/internal/modules/people"
	"github.com/SamoySamoy/Soma/internal/platform/config"
)

// api implements apigen.StrictServerInterface by combining the core handlers
// with each module's handler. Each module's methods are promoted through its
// embedded handler, so adding a module doesn't touch the rest.
type api struct {
	cfg config.Config
	*people.Handler
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
