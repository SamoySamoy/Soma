package people

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/SamoySamoy/Soma/internal/apigen"
	"github.com/SamoySamoy/Soma/internal/platform/apperr"
)

const (
	defaultPageSize = 50
	maxPageSize     = 200
)

// Handler adapts the generated strict API to Service. It holds no business
// rules: validation and authorization live in the service.
type Handler struct {
	svc *Service
}

// NewHandler returns a Handler for svc.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ListContacts implements apigen.StrictServerInterface.
func (h *Handler) ListContacts(ctx context.Context, req apigen.ListContactsRequestObject) (apigen.ListContactsResponseObject, error) {
	limit := int32(defaultPageSize)
	if req.Params.Limit != nil {
		limit = int32(*req.Params.Limit) //nolint:gosec // G115: the range check below rejects overflow
	}
	if limit < 1 || limit > maxPageSize {
		return nil, apperr.Invalid("people.invalid_page", "Choose a page size between 1 and 200.",
			map[string]string{"limit": "Use a number from 1 to 200."})
	}

	var before *uuid.UUID
	if req.Params.Cursor != nil {
		id := *req.Params.Cursor
		before = &id
	}

	page, err := h.svc.List(ctx, limit, before)
	if err != nil {
		return nil, err
	}
	out := apigen.ListContacts200JSONResponse{Items: make([]apigen.Contact, 0, len(page.Items))}
	for _, c := range page.Items {
		out.Items = append(out.Items, toAPI(c))
	}
	if page.Next != nil {
		next := *page.Next
		out.NextCursor = &next
	}
	return out, nil
}

// CreateContact implements apigen.StrictServerInterface.
func (h *Handler) CreateContact(ctx context.Context, req apigen.CreateContactRequestObject) (apigen.CreateContactResponseObject, error) {
	if req.Body == nil {
		return nil, apperr.Invalid("people.invalid_contact", "Send the contact as JSON.", nil)
	}
	c, err := h.svc.Create(ctx, inputFromAPI(*req.Body))
	if err != nil {
		return nil, err
	}
	return apigen.CreateContact201JSONResponse{
		Body:    toAPI(c),
		Headers: apigen.CreateContact201ResponseHeaders{ETag: etag(c.Version)},
	}, nil
}

// GetContact implements apigen.StrictServerInterface.
func (h *Handler) GetContact(ctx context.Context, req apigen.GetContactRequestObject) (apigen.GetContactResponseObject, error) {
	c, err := h.svc.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return apigen.GetContact200JSONResponse{
		Body:    toAPI(c),
		Headers: apigen.GetContact200ResponseHeaders{ETag: etag(c.Version)},
	}, nil
}

// UpdateContact implements apigen.StrictServerInterface.
func (h *Handler) UpdateContact(ctx context.Context, req apigen.UpdateContactRequestObject) (apigen.UpdateContactResponseObject, error) {
	version, err := parseIfMatch(req.Params.IfMatch)
	if err != nil {
		return nil, err
	}
	if req.Body == nil {
		return nil, apperr.Invalid("people.invalid_contact", "Send the changes as JSON.", nil)
	}
	c, err := h.svc.Update(ctx, req.Id, version, Patch(*req.Body))
	if err != nil {
		return nil, err
	}
	return apigen.UpdateContact200JSONResponse{
		Body:    toAPI(c),
		Headers: apigen.UpdateContact200ResponseHeaders{ETag: etag(c.Version)},
	}, nil
}

// DeleteContact implements apigen.StrictServerInterface.
func (h *Handler) DeleteContact(ctx context.Context, req apigen.DeleteContactRequestObject) (apigen.DeleteContactResponseObject, error) {
	if err := h.svc.Delete(ctx, req.Id); err != nil {
		return nil, err
	}
	return apigen.DeleteContact204Response{}, nil
}

// parseIfMatch reads the version from an If-Match header such as "3". A
// missing or malformed header is refused with 412, per the contract.
func parseIfMatch(header *string) (int32, error) {
	missing := apperr.New(apperr.KindPreconditionFailed, "people.version_required",
		"Send If-Match with the version of the contact you are editing.")
	if header == nil {
		return 0, missing
	}
	raw := strings.Trim(strings.TrimSpace(*header), `"`)
	v, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || v < 1 {
		return 0, missing
	}
	return int32(v), nil
}

func etag(version int32) *string {
	s := fmt.Sprintf(`"%d"`, version)
	return &s
}

func inputFromAPI(in apigen.ContactInput) Input {
	return Input{
		DisplayName: in.DisplayName,
		Nickname:    derefString(in.Nickname),
		Email:       derefString(in.Email),
		Phone:       derefString(in.Phone),
		Birthday:    derefString(in.Birthday),
		HowWeMet:    derefString(in.HowWeMet),
		Notes:       derefString(in.Notes),
	}
}

func toAPI(c Contact) apigen.Contact {
	return apigen.Contact{
		Id:          c.ID,
		DisplayName: c.DisplayName,
		Nickname:    optionalString(c.Nickname),
		Email:       optionalString(c.Email),
		Phone:       optionalString(c.Phone),
		Birthday:    optionalString(c.Birthday),
		HowWeMet:    optionalString(c.HowWeMet),
		Notes:       optionalString(c.Notes),
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
		Version:     int(c.Version),
	}
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
