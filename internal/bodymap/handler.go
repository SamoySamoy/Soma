package bodymap

import (
	"context"

	"github.com/SamoySamoy/Soma/internal/apigen"
)

// Handler serves the body map. It holds no rules; Service does the work.
type Handler struct {
	svc *Service
}

// NewHandler returns a Handler for svc.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GetBodyMap implements apigen.StrictServerInterface.
func (h *Handler) GetBodyMap(ctx context.Context, _ apigen.GetBodyMapRequestObject) (apigen.GetBodyMapResponseObject, error) {
	snap, err := h.svc.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	areas := make([]apigen.BodyArea, 0, len(snap.Areas))
	for _, a := range snap.Areas {
		areas = append(areas, apigen.BodyArea{
			Key:     apigen.BodyAreaKey(a.Area),
			Phase:   apigen.BodyAreaPhase(a.Phase),
			Status:  apigen.BodyAreaStatus(a.Level),
			Enabled: a.Enabled,
			Counts:  a.Counts,
		})
	}
	return apigen.GetBodyMap200JSONResponse{AsOf: snap.AsOf, Areas: areas}, nil
}
