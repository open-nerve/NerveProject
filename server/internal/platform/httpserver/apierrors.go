package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"time"
)

// ProblemError is an error the platform maps to a problem+json response
// (M2 design 3.11). Error() becomes the problem's detail. The platform does
// not import internal/shared: shared.Error satisfies this by structure, and so
// can any other error type, e.g. bodyshape's.
//
// Two methods are optional:
//
//	ProblemFields() []error    each element has ProblemField() and
//	                           ProblemCode() string, and Error() is the message
//	RetryAfter() time.Duration a positive value becomes Retry-After
type ProblemError interface {
	error
	ProblemStatus() int
	ProblemCode() string
}

type problemFields interface {
	ProblemFields() []error
}

type problemField interface {
	error
	ProblemField() string
	ProblemCode() string
}

type retryAfter interface {
	RetryAfter() time.Duration
}

// statusClientClosedRequest is logged when the client went away before the
// answer (nginx's 499); the client never sees it.
const statusClientClosedRequest = 499

// APIErrors answers errors as problem+json. Every module wires it into its
// generated handler, and the per-route middlewares use it:
//
//   - StdHTTPServerOptions.ErrorHandlerFunc: BadRequest (parameter binding)
//   - StrictHTTPServerOptions.RequestErrorHandlerFunc: BodyError
//   - StrictHTTPServerOptions.ResponseErrorHandlerFunc: Write
//
// Write is the one road from an error to a problem (M2 design 3.11): a
// ProblemError becomes its own problem, anything else a logged 500.
type APIErrors struct {
	logger *slog.Logger
}

// NewAPIErrors returns APIErrors that log internal errors to logger.
func NewAPIErrors(logger *slog.Logger) APIErrors {
	return APIErrors{logger: logger}
}

// BadRequest answers 400 bad_request for a request whose parameters the
// generated code could not bind; err says what was wrong and becomes the
// detail. (M2/P3 derives the field from binding errors instead.)
func (APIErrors) BadRequest(w http.ResponseWriter, _ *http.Request, err error) {
	WriteProblem(w, Problem{
		Status: http.StatusBadRequest,
		Code:   CodeBadRequest,
		Title:  http.StatusText(http.StatusBadRequest),
		Detail: err.Error(),
	})
}

// BodyError answers a request body that could not be read, parsed or
// decoded, from the bodyshape middleware or the generated strict handler. A
// ProblemError (bodyshape's structural 400 with its fields) and
// *http.MaxBytesError (413) go through Write. Anything else, e.g. a body
// that is not JSON or is empty, is 400 bad_request with a generic detail:
// the decoder's message names Go types, so it is logged at debug level only.
func (e APIErrors) BodyError(w http.ResponseWriter, r *http.Request, err error) {
	var pe ProblemError
	var tooLarge *http.MaxBytesError
	if errors.As(err, &pe) || errors.As(err, &tooLarge) {
		e.Write(w, r, err)
		return
	}
	e.logger.LogAttrs(r.Context(), slog.LevelDebug, "request body not decoded",
		slog.String("request_id", RequestID(r.Context())), slog.Any("error", err))
	WriteProblem(w, Problem{
		Status: http.StatusBadRequest,
		Code:   CodeBadRequest,
		Title:  http.StatusText(http.StatusBadRequest),
		Detail: "The request body could not be decoded.",
	})
}

// Write answers err, returned by a handler or a per-route middleware:
//
//   - a ProblemError: its status, code, detail, fields and Retry-After;
//   - *http.MaxBytesError: 413 payload_too_large;
//   - context.Canceled while the request's context is cancelled: the client
//     went away; logged at debug level, no 500;
//   - anything else: logged, and 500 internal_error without detail.
//
// If the response has already started (the generated code reports a failed
// write that way), a problem appended would corrupt it: the connection is
// aborted instead and the error logged at warn level, as the recover
// middleware does.
func (e APIErrors) Write(w http.ResponseWriter, r *http.Request, err error) {
	attrs := []slog.Attr{
		slog.String("request_id", RequestID(r.Context())),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.Any("error", err),
	}
	if responseStarted(w) {
		e.logger.LogAttrs(r.Context(), slog.LevelWarn, "response failed after it started", attrs...)
		panic(http.ErrAbortHandler)
	}
	var pe ProblemError
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &pe):
		p, retry := problemOf(pe)
		if retry > 0 {
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(retry.Seconds()))))
		}
		WriteProblem(w, p)
	case errors.As(err, &tooLarge):
		WriteProblem(w, Problem{
			Status: http.StatusRequestEntityTooLarge,
			Code:   CodePayloadTooLarge,
			Title:  http.StatusText(http.StatusRequestEntityTooLarge),
			Detail: fmt.Sprintf("The request body exceeds %d bytes.", tooLarge.Limit),
		})
	case errors.Is(err, context.Canceled) && r.Context().Err() != nil:
		e.logger.LogAttrs(r.Context(), slog.LevelDebug, "client went away", attrs...)
		w.WriteHeader(statusClientClosedRequest)
	default:
		e.logger.LogAttrs(r.Context(), slog.LevelError, "API handler failed", attrs...)
		WriteProblem(w, Problem{
			Status: http.StatusInternalServerError,
			Code:   CodeInternal,
			Title:  http.StatusText(http.StatusInternalServerError),
		})
	}
}

// problemOf builds the problem for pe and returns its retry delay.
func problemOf(pe ProblemError) (Problem, time.Duration) {
	status := pe.ProblemStatus()
	p := Problem{Status: status, Code: pe.ProblemCode(), Title: http.StatusText(status), Detail: pe.Error()}
	if pf, ok := pe.(problemFields); ok {
		for _, fe := range pf.ProblemFields() {
			var f problemField
			if errors.As(fe, &f) {
				p.Errors = append(p.Errors, FieldError{Field: f.ProblemField(), Code: f.ProblemCode(), Message: f.Error()})
			}
		}
	}
	var retry time.Duration
	if ra, ok := pe.(retryAfter); ok {
		retry = ra.RetryAfter()
	}
	return p, retry
}

// responseStarted reports whether a status has already gone out through w.
// Generated code receives the access log's statusRecorder, possibly wrapped
// by a module's StdHTTPServerOptions.Middlewares, so wrappers are unwrapped
// the way http.ResponseController does it.
func responseStarted(w http.ResponseWriter) bool {
	for {
		switch rw := w.(type) {
		case *statusRecorder:
			return rw.status != 0
		case interface{ Unwrap() http.ResponseWriter }:
			w = rw.Unwrap()
		default:
			return false
		}
	}
}
