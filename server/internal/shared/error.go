// Package shared is the shared kernel: the few values that cross module
// boundaries (M2 design 3.3). Every module returns its errors as *Error, the
// authentication puts the Actor in the context, and TxManager carries one
// transaction through the repositories of several modules. It imports only
// the standard library, and the platform does not import it: the platform
// declares the small interfaces these types satisfy by structure.
package shared

import "time"

// Kind is the category of a domain error. It decides the HTTP status of the
// problem the error becomes (M2 design 3.11).
type Kind int

// The kinds of domain errors and the status each one maps to.
const (
	KindInvalid         Kind = iota + 1 // 422: a value breaks a domain rule
	KindBadRequest                      // 400: a malformed value, e.g. a cursor
	KindUnauthenticated                 // 401: no valid credential
	KindForbidden                       // 403
	KindNotFound                        // 404: missing, or not visible to the caller
	KindConflict                        // 409
	KindRateLimited                     // 429, with Retry-After
	KindUnavailable                     // 503, with Retry-After
)

// Codes of the platform problems that domain errors carry. Module codes are
// prefixed with the module, e.g. "identity.email_taken".
const (
	CodeValidationFailed = "validation_failed"
	CodeUnauthorized     = "unauthorized"
	CodeServerBusy       = "server_busy"
)

// Field codes: the closed set of FieldError.Code values that clients
// translate (M2 design 3.11). api/common.yaml lists the same set.
const (
	FieldRequired       = "required"
	FieldInvalidFormat  = "invalid_format"
	FieldTooShort       = "too_short"
	FieldTooLong        = "too_long"
	FieldOutOfRange     = "out_of_range"
	FieldNotAllowed     = "not_allowed"
	FieldWeakPassword   = "weak_password"
	FieldCommonPassword = "common_password"
	FieldMustBeFuture   = "must_be_future"
	FieldContainsURL    = "contains_url"
)

// FieldCodes returns the closed set of field codes: every Field* constant.
// The bootstrap tests hold it equal to the contract's enum.
func FieldCodes() []string {
	return []string{
		FieldRequired, FieldInvalidFormat, FieldTooShort, FieldTooLong, FieldOutOfRange,
		FieldNotAllowed, FieldWeakPassword, FieldCommonPassword, FieldMustBeFuture, FieldContainsURL,
	}
}

// status is the HTTP status of the problem an error of kind k becomes. The
// numbers are literal: shared must not import net/http (architecture rule 10).
func (k Kind) status() int {
	switch k {
	case KindInvalid:
		return 422
	case KindBadRequest:
		return 400
	case KindUnauthenticated:
		return 401
	case KindForbidden:
		return 403
	case KindNotFound:
		return 404
	case KindConflict:
		return 409
	case KindRateLimited:
		return 429
	case KindUnavailable:
		return 503
	}
	return 500
}

// FieldError is the problem with one field of a request. Message is English
// for humans; clients translate Code.
type FieldError struct {
	Field   string // JSON path, e.g. "email" or "onboarding_step.profile_complete"
	Code    string // one of the Field* codes
	Message string
}

func (f FieldError) Error() string        { return f.Message }
func (f FieldError) ProblemField() string { return f.Field }
func (f FieldError) ProblemCode() string  { return f.Code }

// Error is the error every module returns for an expected failure. The
// platform maps it to a problem+json response through the httpserver
// ProblemError interface, which Error satisfies by structure. Carrying the
// HTTP status (ProblemStatus) is the one deliberate trade-off of M2 design
// 3.11: the platform can map it without importing shared.
type Error struct {
	Kind       Kind
	Code       string        // the problem code
	Detail     string        // becomes the problem's detail
	Fields     []FieldError  // the invalid fields, if any
	RetryDelay time.Duration // becomes Retry-After when positive
}

// NewError returns an error of kind with a module code, e.g.
// NewError(KindConflict, "identity.email_taken", "…").
func NewError(kind Kind, code, detail string) *Error {
	return &Error{Kind: kind, Code: code, Detail: detail}
}

// Invalid reports the fields that break domain rules: 422 validation_failed.
func Invalid(fields ...FieldError) *Error {
	return &Error{Kind: KindInvalid, Code: CodeValidationFailed, Detail: "The request has invalid values.", Fields: fields}
}

// Unauthenticated reports a request without a valid credential: 401 unauthorized.
func Unauthenticated() *Error {
	return &Error{Kind: KindUnauthenticated, Code: CodeUnauthorized, Detail: "Authentication is required."}
}

// ServerBusy reports that the server is temporarily overloaded, whoever the
// caller is: 503 server_busy with Retry-After.
func ServerBusy(retry time.Duration) *Error {
	return &Error{Kind: KindUnavailable, Code: CodeServerBusy, Detail: "The server is busy; retry shortly.", RetryDelay: retry}
}

func (e *Error) Error() string { return e.Detail }

// Is makes errors.Is match an error of the same kind and code, so a caller
// can test for, say, identity.email_taken without comparing pointers.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	return ok && t.Kind == e.Kind && t.Code == e.Code
}

// ProblemStatus is the HTTP status of the problem.
func (e *Error) ProblemStatus() int { return e.Kind.status() }

// ProblemCode is the problem's code.
func (e *Error) ProblemCode() string { return e.Code }

// ProblemFields lists the invalid fields; each element has ProblemField and
// ProblemCode.
func (e *Error) ProblemFields() []error {
	out := make([]error, len(e.Fields))
	for i, f := range e.Fields {
		out[i] = f
	}
	return out
}

// RetryAfter is how long the caller should wait before retrying; zero for none.
func (e *Error) RetryAfter() time.Duration { return e.RetryDelay }
