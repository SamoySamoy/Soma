// Package apperr defines the domain error type shared by every layer. Services
// return these errors; the HTTP layer maps them to RFC 9457 problem details.
package apperr

import (
	"errors"
	"fmt"
)

// Kind classifies an error. Each kind maps to exactly one HTTP status.
type Kind int

// Error kinds. KindInternal is the zero value so an unclassified error is never
// mistaken for a client error.
const (
	KindInternal Kind = iota
	KindInvalid
	KindUnauthorized
	KindForbidden
	KindNotFound
	KindConflict
	KindPreconditionFailed
	KindRateLimited
)

func (k Kind) String() string {
	switch k {
	case KindInvalid:
		return "invalid"
	case KindUnauthorized:
		return "unauthorized"
	case KindForbidden:
		return "forbidden"
	case KindNotFound:
		return "not_found"
	case KindConflict:
		return "conflict"
	case KindPreconditionFailed:
		return "precondition_failed"
	case KindRateLimited:
		return "rate_limited"
	default:
		return "internal"
	}
}

// Error is a classified domain error.
type Error struct {
	Kind Kind
	// Code is a stable, machine-readable identifier such as
	// "money.zero_amount". Clients may depend on it.
	Code string
	// Message is safe to show to the user. It must not contain secrets or
	// personal data.
	Message string
	// Fields holds per-field validation messages, keyed by JSON field name.
	Fields map[string]string
	// Err is the underlying cause, kept for logs and errors.Is/As.
	Err error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.Err }

// New returns an Error of the given kind.
func New(kind Kind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message}
}

// Wrap returns an Error of the given kind that wraps cause.
func Wrap(cause error, kind Kind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message, Err: cause}
}

// Invalid returns a validation error with per-field messages.
func Invalid(code, message string, fields map[string]string) *Error {
	return &Error{Kind: KindInvalid, Code: code, Message: message, Fields: fields}
}

// NotFound returns a not-found error. Also used when the caller lacks access,
// so the existence of a record is never revealed.
func NotFound(code, message string) *Error { return New(KindNotFound, code, message) }

// Unauthorized returns an error for a missing or invalid authentication.
func Unauthorized(code, message string) *Error { return New(KindUnauthorized, code, message) }

// Forbidden returns an error for an authenticated caller who may not act.
func Forbidden(code, message string) *Error { return New(KindForbidden, code, message) }

// Conflict returns an error for a state conflict such as a duplicate.
func Conflict(code, message string) *Error { return New(KindConflict, code, message) }

// As extracts an *Error from err's chain.
func As(err error) (*Error, bool) {
	var e *Error
	ok := errors.As(err, &e)
	return e, ok
}

// KindOf returns the kind of the first *Error in err's chain, or KindInternal.
func KindOf(err error) Kind {
	if e, ok := As(err); ok {
		return e.Kind
	}
	return KindInternal
}
