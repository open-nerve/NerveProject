package shared_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestKindStatus(t *testing.T) {
	tests := []struct {
		kind shared.Kind
		want int
	}{
		{shared.KindInvalid, 422},
		{shared.KindBadRequest, 400},
		{shared.KindUnauthenticated, 401},
		{shared.KindForbidden, 403},
		{shared.KindNotFound, 404},
		{shared.KindConflict, 409},
		{shared.KindRateLimited, 429},
		{shared.KindUnavailable, 503},
		{shared.Kind(0), 500},
		{shared.Kind(99), 500},
	}
	for _, tt := range tests {
		if got := shared.NewError(tt.kind, "m.code", "d").ProblemStatus(); got != tt.want {
			t.Errorf("Kind %d: ProblemStatus() = %d, want %d", tt.kind, got, tt.want)
		}
	}
}

func TestConstructors(t *testing.T) {
	tests := []struct {
		name   string
		err    *shared.Error
		status int
		code   string
		retry  time.Duration
	}{
		{"Invalid", shared.Invalid(), 422, "validation_failed", 0},
		{"Unauthenticated", shared.Unauthenticated(), 401, "unauthorized", 0},
		{"ServerBusy", shared.ServerBusy(time.Second), 503, "server_busy", time.Second},
		{"NewError", shared.NewError(shared.KindConflict, "identity.email_taken", "taken"), 409, "identity.email_taken", 0},
	}
	for _, tt := range tests {
		if tt.err.ProblemStatus() != tt.status || tt.err.ProblemCode() != tt.code || tt.err.RetryAfter() != tt.retry || tt.err.Error() == "" {
			t.Errorf("%s = %d %q retry %s detail %q, want %d %q retry %s and a detail",
				tt.name, tt.err.ProblemStatus(), tt.err.ProblemCode(), tt.err.RetryAfter(), tt.err.Error(), tt.status, tt.code, tt.retry)
		}
	}
}

func TestProblemFields(t *testing.T) {
	err := shared.Invalid(
		shared.FieldError{Field: "email", Code: shared.FieldInvalidFormat, Message: "not an email address"},
		shared.FieldError{Field: "password", Code: shared.FieldWeakPassword, Message: "too weak"},
	)

	fields := err.ProblemFields()

	if len(fields) != 2 {
		t.Fatalf("ProblemFields() = %v, want 2 fields", fields)
	}
	type field interface {
		error
		ProblemField() string
		ProblemCode() string
	}
	var f field
	if !errors.As(fields[1], &f) || f.ProblemField() != "password" || f.ProblemCode() != "weak_password" || f.Error() != "too weak" {
		t.Errorf("second field = %v, want password weak_password \"too weak\"", fields[1])
	}
}

func TestIsMatchesKindAndCode(t *testing.T) {
	taken := shared.NewError(shared.KindConflict, "identity.email_taken", "taken")
	wrapped := fmt.Errorf("register: %w", shared.NewError(shared.KindConflict, "identity.email_taken", "another detail"))

	if !errors.Is(wrapped, taken) {
		t.Error("errors.Is(wrapped identity.email_taken, identity.email_taken) = false, want true")
	}
	if errors.Is(wrapped, shared.NewError(shared.KindForbidden, "identity.email_taken", "")) {
		t.Error("errors.Is matched another kind")
	}
	if errors.Is(wrapped, shared.NewError(shared.KindConflict, "identity.other", "")) {
		t.Error("errors.Is matched another code")
	}
}
