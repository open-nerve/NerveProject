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
func (e APIErrors) InternalError(w http.ResponseWriter, r *http.Request, err error) {
	e.logger.ErrorContext(r.Context(), "API handler failed",
		slog.String("request_id", requestID(r.Context())),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.Any("error", err),
	)
	WriteProblem(w, Problem{
		Status: http.StatusInternalServerError,
		Code:   CodeInternal,
		Title:  http.StatusText(http.StatusInternalServerError),
	})
}
