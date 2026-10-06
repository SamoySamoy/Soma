package self

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/SamoySamoy/Soma/internal/apigen"
	"github.com/SamoySamoy/Soma/internal/platform/apperr"
)

// Handler adapts the generated strict API to Service.
type Handler struct {
	svc *Service
}

// NewHandler returns a Handler for svc.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GetSelfProfile implements apigen.StrictServerInterface.
func (h *Handler) GetSelfProfile(ctx context.Context, _ apigen.GetSelfProfileRequestObject) (apigen.GetSelfProfileResponseObject, error) {
	p, err := h.svc.Get(ctx)
	if err != nil {
		return nil, err
	}
	return apigen.GetSelfProfile200JSONResponse{
		Body:    toAPI(p),
		Headers: apigen.GetSelfProfile200ResponseHeaders{ETag: etag(p.Version)},
	}, nil
}

// UpdateSelfProfile implements apigen.StrictServerInterface.
func (h *Handler) UpdateSelfProfile(ctx context.Context, req apigen.UpdateSelfProfileRequestObject) (apigen.UpdateSelfProfileResponseObject, error) {
	version, err := parseIfMatch(req.Params.IfMatch)
	if err != nil {
		return nil, err
	}
	if req.Body == nil {
		return nil, apperr.Invalid("self.invalid_profile", "Send the changes as JSON.", nil)
	}
	p, err := h.svc.Update(ctx, version, Patch(*req.Body))
	if err != nil {
		return nil, err
	}
	return apigen.UpdateSelfProfile200JSONResponse{
		Body:    toAPI(p),
		Headers: apigen.UpdateSelfProfile200ResponseHeaders{ETag: etag(p.Version)},
	}, nil
}

// parseIfMatch reads the version from an If-Match header. Missing or malformed
// is refused with 412.
func parseIfMatch(header *string) (int32, error) {
	missing := apperr.New(apperr.KindPreconditionFailed, "self.version_required",
		"Send If-Match with the version of your profile you are editing.")
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

func toAPI(p Profile) apigen.SelfProfile {
	out := apigen.SelfProfile{
		UpdatedAt: p.UpdatedAt,
		Version:   int(p.Version),
	}
	if p.PreferredName != "" {
		out.PreferredName = &p.PreferredName
	}
	if p.BirthDate != "" {
		out.BirthDate = &p.BirthDate
	}
	if p.CoreValues != "" {
		out.CoreValues = &p.CoreValues
	}
	if p.Bio != "" {
		out.Bio = &p.Bio
	}
	return out
}
