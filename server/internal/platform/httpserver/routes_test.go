package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthzDoesNotRunChecks(t *testing.T) {
	failing := Check{Name: "database", Run: func(context.Context) error { return errors.New("down") }}
	mux := NewMux(slog.New(slog.DiscardHandler), failing)

	rec := serve(mux, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK || rec.Body.String() != `{"status":"ok"}`+"\n" {
		t.Errorf("GET /healthz = %d %s, want 200 {\"status\":\"ok\"}", rec.Code, rec.Body)
	}
}

func TestReadyzWhenAllChecksPass(t *testing.T) {
	var ran []string
	check := func(name string) Check {
		return Check{Name: name, Run: func(ctx context.Context) error {
			if _, ok := ctx.Deadline(); !ok {
				t.Errorf("check %s ran without a deadline", name)
			}
			ran = append(ran, name)
			return nil
		}}
	}
	mux := NewMux(slog.New(slog.DiscardHandler), check("database"), check("migrations"))

	rec := serve(mux, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if rec.Code != http.StatusOK || rec.Body.String() != `{"status":"ok"}`+"\n" {
		t.Errorf("GET /readyz = %d %s, want 200 {\"status\":\"ok\"}", rec.Code, rec.Body)
	}
	if strings.Join(ran, ",") != "database,migrations" {
		t.Errorf("checks ran = %v, want database then migrations", ran)
	}
}

func TestReadyzReportsFirstFailingCheck(t *testing.T) {
	logger, logs := captureLogs(t)
	secondRan := false
	mux := NewMux(logger,
		Check{Name: "database", Run: func(context.Context) error { return errors.New("connection refused") }},
		Check{Name: "migrations", Run: func(context.Context) error { secondRan = true; return nil }},
	)

	rec := serve(mux, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != ContentTypeProblem {
		t.Errorf("Content-Type = %q, want %q", ct, ContentTypeProblem)
	}
	want := `{"status":503,"code":"not_ready","title":"Service Unavailable","detail":"database is not ready"}` + "\n"
	if rec.Body.String() != want {
		t.Errorf("body = %s, want %s", rec.Body, want)
	}
	if secondRan {
		t.Error("checks after the failing one ran")
	}
	entry := findLog(logs(), "readiness check failed")
	if entry == nil || entry["check"] != "database" || entry["error"] != "connection refused" {
		t.Errorf("log = %v, want the failing check and its error", entry)
	}
}

func TestUnknownAPIPathIsProblem404(t *testing.T) {
	mux := NewMux(slog.New(slog.DiscardHandler))
	for _, target := range []string{"/api/", "/api/v0/nope"} {
		rec := serve(mux, httptest.NewRequest(http.MethodPost, target, nil))

		if rec.Code != http.StatusNotFound || rec.Header().Get("Content-Type") != ContentTypeProblem {
			t.Errorf("POST %s = %d %s, want 404 problem+json", target, rec.Code, rec.Header().Get("Content-Type"))
		}
		want := `{"status":404,"code":"not_found","title":"Not Found","detail":"no API endpoint for POST ` + target + `"}` + "\n"
		if rec.Body.String() != want {
			t.Errorf("body = %s, want %s", rec.Body, want)
		}
	}
}

func TestOtherPathsAreLeftForTheWebUI(t *testing.T) {
	rec := serve(NewMux(slog.New(slog.DiscardHandler)), httptest.NewRequest(http.MethodGet, "/projects", nil))

	if rec.Code != http.StatusNotFound || rec.Header().Get("Content-Type") == ContentTypeProblem {
		t.Errorf("GET /projects = %d %s, want the mux's plain 404", rec.Code, rec.Header().Get("Content-Type"))
	}
}
