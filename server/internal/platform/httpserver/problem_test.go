package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteProblem(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteProblem(rec, Problem{
		Status: http.StatusUnprocessableEntity,
		Code:   "issue.state_not_in_project",
		Title:  "State is not in the project",
		Errors: []FieldError{{Field: "state_id", Code: "not_allowed", Message: "unknown state"}},
	})

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}
	want := `{"status":422,"code":"issue.state_not_in_project","title":"State is not in the project",` +
		`"errors":[{"field":"state_id","code":"not_allowed","message":"unknown state"}]}` + "\n"
	if got := rec.Body.String(); got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
}

func TestWriteProblemOmitsEmptyOptionalMembers(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteProblem(rec, Problem{Status: http.StatusNotFound, Code: CodeNotFound, Title: "Not Found"})

	want := `{"status":404,"code":"not_found","title":"Not Found"}` + "\n"
	if got := rec.Body.String(); got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
}
