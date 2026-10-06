package tasks

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

// Handler adapts the generated strict API to Service.
type Handler struct {
	svc *Service
}

// NewHandler returns a Handler for svc.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ListTasks implements apigen.StrictServerInterface.
func (h *Handler) ListTasks(ctx context.Context, req apigen.ListTasksRequestObject) (apigen.ListTasksResponseObject, error) {
	limit := int32(defaultPageSize)
	if req.Params.Limit != nil {
		limit = int32(*req.Params.Limit) //nolint:gosec // G115: the range check below rejects overflow
	}
	if limit < 1 || limit > maxPageSize {
		return nil, apperr.Invalid("tasks.invalid_page", "Choose a page size between 1 and 200.",
			map[string]string{"limit": "Use a number from 1 to 200."})
	}
	var before *uuid.UUID
	if req.Params.Cursor != nil {
		id := *req.Params.Cursor
		before = &id
	}
	openOnly := req.Params.OpenOnly != nil && *req.Params.OpenOnly
	page, err := h.svc.List(ctx, limit, before, openOnly)
	if err != nil {
		return nil, err
	}
	today := h.svc.today()
	out := apigen.ListTasks200JSONResponse{Items: make([]apigen.Task, 0, len(page.Items))}
	for _, t := range page.Items {
		out.Items = append(out.Items, toAPI(t, today))
	}
	if page.Next != nil {
		next := *page.Next
		out.NextCursor = &next
	}
	return out, nil
}

// CreateTask implements apigen.StrictServerInterface.
func (h *Handler) CreateTask(ctx context.Context, req apigen.CreateTaskRequestObject) (apigen.CreateTaskResponseObject, error) {
	if req.Body == nil {
		return nil, apperr.Invalid("tasks.invalid_task", "Send the task as JSON.", nil)
	}
	t, err := h.svc.Create(ctx, inputFromAPI(*req.Body))
	if err != nil {
		return nil, err
	}
	return apigen.CreateTask201JSONResponse{
		Body:    toAPI(t, h.svc.today()),
		Headers: apigen.CreateTask201ResponseHeaders{ETag: etag(t.Version)},
	}, nil
}

// GetTask implements apigen.StrictServerInterface.
func (h *Handler) GetTask(ctx context.Context, req apigen.GetTaskRequestObject) (apigen.GetTaskResponseObject, error) {
	t, err := h.svc.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return apigen.GetTask200JSONResponse{
		Body:    toAPI(t, h.svc.today()),
		Headers: apigen.GetTask200ResponseHeaders{ETag: etag(t.Version)},
	}, nil
}

// UpdateTask implements apigen.StrictServerInterface.
func (h *Handler) UpdateTask(ctx context.Context, req apigen.UpdateTaskRequestObject) (apigen.UpdateTaskResponseObject, error) {
	version, err := parseIfMatch(req.Params.IfMatch)
	if err != nil {
		return nil, err
	}
	if req.Body == nil {
		return nil, apperr.Invalid("tasks.invalid_task", "Send the changes as JSON.", nil)
	}
	t, err := h.svc.Update(ctx, req.Id, version, Patch(*req.Body))
	if err != nil {
		return nil, err
	}
	return apigen.UpdateTask200JSONResponse{
		Body:    toAPI(t, h.svc.today()),
		Headers: apigen.UpdateTask200ResponseHeaders{ETag: etag(t.Version)},
	}, nil
}

// DeleteTask implements apigen.StrictServerInterface.
func (h *Handler) DeleteTask(ctx context.Context, req apigen.DeleteTaskRequestObject) (apigen.DeleteTaskResponseObject, error) {
	if err := h.svc.Delete(ctx, req.Id); err != nil {
		return nil, err
	}
	return apigen.DeleteTask204Response{}, nil
}

// CompleteTask implements apigen.StrictServerInterface.
func (h *Handler) CompleteTask(ctx context.Context, req apigen.CompleteTaskRequestObject) (apigen.CompleteTaskResponseObject, error) {
	t, err := h.svc.Complete(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return apigen.CompleteTask200JSONResponse{
		Body:    toAPI(t, h.svc.today()),
		Headers: apigen.CompleteTask200ResponseHeaders{ETag: etag(t.Version)},
	}, nil
}

// ReopenTask implements apigen.StrictServerInterface.
func (h *Handler) ReopenTask(ctx context.Context, req apigen.ReopenTaskRequestObject) (apigen.ReopenTaskResponseObject, error) {
	t, err := h.svc.Reopen(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return apigen.ReopenTask200JSONResponse{
		Body:    toAPI(t, h.svc.today()),
		Headers: apigen.ReopenTask200ResponseHeaders{ETag: etag(t.Version)},
	}, nil
}

// parseIfMatch reads the version from an If-Match header. Missing or malformed
// is refused with 412.
func parseIfMatch(header *string) (int32, error) {
	missing := apperr.New(apperr.KindPreconditionFailed, "tasks.version_required",
		"Send If-Match with the version of the task you are editing.")
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

func inputFromAPI(in apigen.TaskInput) Input {
	return Input{
		Title: in.Title,
		Notes: derefString(in.Notes),
		DueOn: derefString(in.DueOn),
	}
}

func toAPI(t Task, today time.Time) apigen.Task {
	out := apigen.Task{
		Id:        t.ID,
		Title:     t.Title,
		Completed: t.Completed(),
		Overdue:   t.Overdue(today),
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
		Version:   int(t.Version),
	}
	if t.Notes != "" {
		out.Notes = &t.Notes
	}
	if t.DueOn != "" {
		out.DueOn = &t.DueOn
	}
	out.CompletedAt = t.CompletedAt
	return out
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
