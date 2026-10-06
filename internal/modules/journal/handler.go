package journal

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

// Handler adapts the generated strict API to Service.
type Handler struct {
	svc *Service
}

// NewHandler returns a Handler for svc.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ListJournalEntries implements apigen.StrictServerInterface.
func (h *Handler) ListJournalEntries(ctx context.Context, req apigen.ListJournalEntriesRequestObject) (apigen.ListJournalEntriesResponseObject, error) {
	limit := int32(defaultPageSize)
	if req.Params.Limit != nil {
		limit = int32(*req.Params.Limit) //nolint:gosec // G115: the range check below rejects overflow
	}
	if limit < 1 || limit > maxPageSize {
		return nil, apperr.Invalid("journal.invalid_page", "Choose a page size between 1 and 200.",
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
	out := apigen.ListJournalEntries200JSONResponse{Items: make([]apigen.JournalEntry, 0, len(page.Items))}
	for _, e := range page.Items {
		out.Items = append(out.Items, toAPI(e))
	}
	if page.Next != nil {
		next := *page.Next
		out.NextCursor = &next
	}
	return out, nil
}

// CreateJournalEntry implements apigen.StrictServerInterface.
func (h *Handler) CreateJournalEntry(ctx context.Context, req apigen.CreateJournalEntryRequestObject) (apigen.CreateJournalEntryResponseObject, error) {
	if req.Body == nil {
		return nil, apperr.Invalid("journal.invalid_entry", "Send the entry as JSON.", nil)
	}
	e, err := h.svc.Create(ctx, inputFromAPI(*req.Body))
	if err != nil {
		return nil, err
	}
	return apigen.CreateJournalEntry201JSONResponse{
		Body:    toAPI(e),
		Headers: apigen.CreateJournalEntry201ResponseHeaders{ETag: etag(e.Version)},
	}, nil
}

// GetJournalEntry implements apigen.StrictServerInterface.
func (h *Handler) GetJournalEntry(ctx context.Context, req apigen.GetJournalEntryRequestObject) (apigen.GetJournalEntryResponseObject, error) {
	e, err := h.svc.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return apigen.GetJournalEntry200JSONResponse{
		Body:    toAPI(e),
		Headers: apigen.GetJournalEntry200ResponseHeaders{ETag: etag(e.Version)},
	}, nil
}

// UpdateJournalEntry implements apigen.StrictServerInterface.
func (h *Handler) UpdateJournalEntry(ctx context.Context, req apigen.UpdateJournalEntryRequestObject) (apigen.UpdateJournalEntryResponseObject, error) {
	version, err := parseIfMatch(req.Params.IfMatch)
	if err != nil {
		return nil, err
	}
	if req.Body == nil {
		return nil, apperr.Invalid("journal.invalid_entry", "Send the changes as JSON.", nil)
	}
	e, err := h.svc.Update(ctx, req.Id, version, Patch(*req.Body))
	if err != nil {
		return nil, err
	}
	return apigen.UpdateJournalEntry200JSONResponse{
		Body:    toAPI(e),
		Headers: apigen.UpdateJournalEntry200ResponseHeaders{ETag: etag(e.Version)},
	}, nil
}

// DeleteJournalEntry implements apigen.StrictServerInterface.
func (h *Handler) DeleteJournalEntry(ctx context.Context, req apigen.DeleteJournalEntryRequestObject) (apigen.DeleteJournalEntryResponseObject, error) {
	if err := h.svc.Delete(ctx, req.Id); err != nil {
		return nil, err
	}
	return apigen.DeleteJournalEntry204Response{}, nil
}

// parseIfMatch reads the version from an If-Match header such as "3". A missing
// or malformed header is refused with 412.
func parseIfMatch(header *string) (int32, error) {
	missing := apperr.New(apperr.KindPreconditionFailed, "journal.version_required",
		"Send If-Match with the version of the entry you are editing.")
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

func inputFromAPI(in apigen.JournalEntryInput) Input {
	mood := 0
	if in.Mood != nil {
		mood = *in.Mood
	}
	return Input{
		EntryDate: in.EntryDate,
		Title:     derefString(in.Title),
		Body:      in.Body,
		Mood:      mood,
	}
}

func toAPI(e Entry) apigen.JournalEntry {
	out := apigen.JournalEntry{
		Id:        e.ID,
		EntryDate: e.EntryDate,
		Body:      e.Body,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
		Version:   int(e.Version),
	}
	if e.Title != "" {
		out.Title = &e.Title
	}
	if e.Mood != 0 {
		m := e.Mood
		out.Mood = &m
	}
	return out
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
