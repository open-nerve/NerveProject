package httpserver

import (
	"errors"
	"io"
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

// Nothing was written yet, so the platform chain's recorder reports the
// response as not started and InternalError answers the 500 problem.
func TestAPIErrorsInternalErrorLogsAndHidesTheError(t *testing.T) {
	logger, logs := captureLogs(t)
	errs := NewAPIErrors(logger)
	h := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		errs.InternalError(w, r, errors.New("dial tcp 10.0.0.5:5432: connection refused"))
	}), logger)
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
	if entry == nil || entry["level"] != "ERROR" || entry["error"] != "dial tcp 10.0.0.5:5432: connection refused" ||
		entry["request_id"] != "req-7" || entry["method"] != "GET" || entry["path"] != "/api/v0/things" {
		t.Errorf("log = %v, want the error with request_id, method and path at level ERROR", entry)
	}
}

// Generated strict code also reports a write that failed after the response
// started (WriteHeader(200), then the buffered body fails to go out). A
// problem appended there would corrupt the 200, so the response is aborted.
func TestAPIErrorsInternalErrorAbortsAStartedResponse(t *testing.T) {
	logger, logs := captureLogs(t)
	errs := NewAPIErrors(logger)
	srv := httptest.NewServer(middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"product":`)
		_ = http.NewResponseController(w).Flush()
		errs.InternalError(w, r, errors.New("write tcp: broken pipe"))
	}), logger))
	t.Cleanup(srv.Close)
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/v0/things", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(HeaderRequestID, "req-8")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	srv.Close() // waits for the handler, so its log lines are complete

	if resp.StatusCode != http.StatusOK || readErr == nil {
		t.Errorf("response = %d, read error %v; want the started 200 cut off", resp.StatusCode, readErr)
	}
	if string(body) != `{"product":` {
		t.Errorf("body = %q, want only what the handler wrote: no problem appended", body)
	}
	entries := logs()
	for _, e := range entries {
		if e["level"] == "ERROR" {
			t.Errorf("log %v at level ERROR, want none: the client may simply be gone", e)
		}
	}
	entry := findLog(entries, "response failed after it started")
	if entry == nil || entry["level"] != "WARN" || entry["error"] != "write tcp: broken pipe" ||
		entry["request_id"] != "req-8" || entry["method"] != "GET" || entry["path"] != "/api/v0/things" {
		t.Errorf("log = %v, want the error with request_id, method and path at level WARN", entry)
	}
}

// wrappingWriter stands for a module middleware that wraps the writer.
type wrappingWriter struct{ http.ResponseWriter }

func (w wrappingWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func TestResponseStartedSeesThroughWrappers(t *testing.T) {
	rec := &statusRecorder{ResponseWriter: httptest.NewRecorder()}
	w := wrappingWriter{rec}
	if responseStarted(w) {
		t.Error("responseStarted() = true before anything was written")
	}
	rec.WriteHeader(http.StatusOK)
	if !responseStarted(w) {
		t.Error("responseStarted() = false after WriteHeader, through a wrapper")
	}
	if responseStarted(httptest.NewRecorder()) {
		t.Error("responseStarted() = true for a writer outside the platform chain")
	}
}
