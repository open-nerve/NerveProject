package httpserver

import (
	"log/slog"
	"net/http"
)

// APIErrors answers the errors that code generated from the OpenAPI
// description reports, as problem+json instead of oapi-codegen's default
// text/plain http.Error. Every module wires it into its generated handler:
//
//   - StdHTTPServerOptions.ErrorHandlerFunc: BadRequest
//   - StrictHTTPServerOptions.RequestErrorHandlerFunc: BadRequest
//   - StrictHTTPServerOptions.ResponseErrorHandlerFunc: InternalError
type APIErrors struct {
	logger *slog.Logger
}

// NewAPIErrors returns APIErrors that log internal errors to logger.
func NewAPIErrors(logger *slog.Logger) APIErrors {
	return APIErrors{logger: logger}
}

// BadRequest answers 400 bad_request for a request whose parameters or body
// the generated code could not bind or decode; err says what was wrong and
// becomes the detail.
func (APIErrors) BadRequest(w http.ResponseWriter, _ *http.Request, err error) {
	WriteProblem(w, Problem{
		Status: http.StatusBadRequest,
		Code:   CodeBadRequest,
		Title:  http.StatusText(http.StatusBadRequest),
		Detail: err.Error(),
	})
}

// InternalError logs err and answers 500 internal_error. Like a recovered
// panic, the answer carries no detail: err may describe internals.
//
// Generated strict code also calls it when writing a response that has
// already started fails (WriteHeader(200), then the body write errors). A
// problem appended there would corrupt the response, so, following the
// recover middleware, it aborts the connection instead and logs at warn: the
// usual cause is a client that went away, not a server fault.
func (e APIErrors) InternalError(w http.ResponseWriter, r *http.Request, err error) {
	attrs := []slog.Attr{
		slog.String("request_id", requestID(r.Context())),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.Any("error", err),
	}
	if responseStarted(w) {
		e.logger.LogAttrs(r.Context(), slog.LevelWarn, "response failed after it started", attrs...)
		panic(http.ErrAbortHandler)
	}
	e.logger.LogAttrs(r.Context(), slog.LevelError, "API handler failed", attrs...)
	WriteProblem(w, Problem{
		Status: http.StatusInternalServerError,
		Code:   CodeInternal,
		Title:  http.StatusText(http.StatusInternalServerError),
	})
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
