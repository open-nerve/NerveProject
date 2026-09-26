package httpserver

import (
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

// Without a token the body is never looked at: authentication runs before
// the body structure check.
func TestAuthenticationRunsBeforeTheBodyCheck(t *testing.T) {
	router, got := mount(t, newTestAPI(t, &fakeAuth{}, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))

	rec := serve(router, post("/api/v0/things", "", `{"nope":1}`))

	if rec.Code != http.StatusUnauthorized || got.called {
		t.Errorf("status = %d, handler called %v; want 401 before the body is checked", rec.Code, got.called)
	}
}

// The body check reads the body through the body limit: the limit runs
// before it, and the check's own read is what fails.
func TestBodyLimitRunsBeforeTheBodyCheck(t *testing.T) {
	router, got := mount(t, newTestAPI(t, &fakeAuth{}, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))

	rec := serve(router, post("/api/v0/things", "tok", `{"name":"`+strings.Repeat("a", 100)+`"}`))

	if p := decodeProblem(t, rec); rec.Code != http.StatusRequestEntityTooLarge || p.Code != CodePayloadTooLarge || got.called {
		t.Errorf("response = %d %+v, handler called %v; want 413 payload_too_large", rec.Code, p, got.called)
	}
}

func TestBodyCheckAnswersEveryProblemAs400(t *testing.T) {
	router, got := mount(t, newTestAPI(t, &fakeAuth{}, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))

	rec := serve(router, post("/api/v0/things", "tok", `{"extra":1}`))

	want := `{"status":400,"code":"bad_request","title":"Bad Request","detail":"The request body does not match the API description.",` +
		`"errors":[{"field":"extra","code":"not_allowed","message":"is not a property of this request"},{"field":"name","code":"required","message":"is required"}]}` + "\n"
	if rec.Code != http.StatusBadRequest || rec.Body.String() != want || got.called {
		t.Errorf("response = %d %s, handler called %v; want 400 %s", rec.Code, rec.Body, got.called, want)
	}
}

func TestBodyThatIsNotJSONIs400WithAGenericDetail(t *testing.T) {
	router, _ := mount(t, newTestAPI(t, &fakeAuth{}, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))

	rec := serve(router, post("/api/v0/things", "tok", `{"name":`))

	if p := decodeProblem(t, rec); rec.Code != http.StatusBadRequest || p.Detail != "The request body could not be decoded." || len(p.Errors) != 0 {
		t.Errorf("response = %d %+v, want 400 with the generic detail", rec.Code, p)
	}
}
