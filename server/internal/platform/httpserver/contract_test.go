package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
)

// TestPlatformProblemsMatchTheContract keeps Problem in step with the Problem
// schema of api/common.yaml: every problem the platform writes must validate.
func TestPlatformProblemsMatchTheContract(t *testing.T) {
	contract := apitest.Load(t)
	discard := slog.New(slog.DiscardHandler)
	errs := NewAPIErrors(discard)
	notReady := Check{Name: "database", Run: func(context.Context) error { return errors.New("down") }}
	tests := []struct {
		name   string
		h      http.Handler
		target string
		status int
	}{
		{"unknown API path", NewMux(discard), "/api/v0/nope", http.StatusNotFound},
		{"not ready", NewMux(discard, notReady), "/readyz", http.StatusServiceUnavailable},
		{"panic", middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }), discard), "/api/v0/boom", http.StatusInternalServerError},
		{"bad request", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			errs.BadRequest(w, r, errors.New("Invalid format for parameter limit"))
		}), "/api/v0/things?limit=x", http.StatusBadRequest},
		{"internal error", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			errs.InternalError(w, r, errors.New("boom"))
		}), "/api/v0/things", http.StatusInternalServerError},
		{"field errors", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			WriteProblem(w, Problem{
				Status: http.StatusUnprocessableEntity,
				Code:   "issue.invalid",
				Title:  http.StatusText(http.StatusUnprocessableEntity),
				Detail: "the issue is invalid",
				Errors: []FieldError{{Field: "name", Message: "is required"}},
			})
		}), "/api/v0/issues", http.StatusUnprocessableEntity},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(tt.h, httptest.NewRequest(http.MethodGet, tt.target, nil))

			if rec.Code != tt.status || rec.Header().Get("Content-Type") != ContentTypeProblem {
				t.Fatalf("response = %d %s, want %d problem+json", rec.Code, rec.Header().Get("Content-Type"), tt.status)
			}
			contract.CheckSchema(t, "Problem", rec.Body.Bytes())
		})
	}
}
