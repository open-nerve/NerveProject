package httpserver

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIErrorsBadRequest(t *testing.T) {
	errs := NewAPIErrors(slog.New(slog.DiscardHandler))
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		errs.BadRequest(w, r, errors.New("Invalid format for parameter limit: not a number"))
	})

	rec := serve(h, httptest.NewRequest(http.MethodGet, "/api/v0/things?limit=x", nil))

	if rec.Code != http.StatusBadRequest || rec.Header().Get("Content-Type") != ContentTypeProblem {
		t.Errorf("response = %d %s, want 400 problem+json", rec.Code, rec.Header().Get("Content-Type"))
	}
	want := `{"status":400,"code":"bad_request","title":"Bad Request","detail":"Invalid format for parameter limit: not a number"}` + "\n"
	if rec.Body.String() != want {
		t.Errorf("body = %s, want %s", rec.Body, want)
	}
}

func TestAPIErrorsInternalErrorLogsAndHidesTheError(t *testing.T) {
	logger, logs := captureLogs(t)
	errs := NewAPIErrors(logger)
	h := withRequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		errs.InternalError(w, r, errors.New("dial tcp 10.0.0.5:5432: connection refused"))
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v0/things", nil)
	req.Header.Set(HeaderRequestID, "req-7")

	rec := serve(h, req)

	if rec.Code != http.StatusInternalServerError || rec.Header().Get("Content-Type") != ContentTypeProblem {
		t.Errorf("response = %d %s, want 500 problem+json", rec.Code, rec.Header().Get("Content-Type"))
	}
	want := `{"status":500,"code":"internal_error","title":"Internal Server Error"}` + "\n"
	if rec.Body.String() != want {
		t.Errorf("body = %s, want %s", rec.Body, want)
	}
	entry := findLog(logs(), "API handler failed")
	if entry == nil || entry["error"] != "dial tcp 10.0.0.5:5432: connection refused" ||
		entry["request_id"] != "req-7" || entry["method"] != "GET" || entry["path"] != "/api/v0/things" {
		t.Errorf("log = %v, want the error with request_id, method and path", entry)
	}
}
