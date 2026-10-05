package server

import (
	"context"

	"github.com/SamoySamoy/Soma/internal/apigen"
	"github.com/SamoySamoy/Soma/internal/buildinfo"
	"github.com/SamoySamoy/Soma/internal/platform/config"
)

// api implements apigen.StrictServerInterface. As modules arrive it embeds
// each module's handler so the generated interface is satisfied piece by piece.
type api struct {
	cfg config.Config
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
