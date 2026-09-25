package domain

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestNewAccountNormalizesTheEmail(t *testing.T) {
	email, err := NewAccount(rules, "  Alice@Corp.COM ", "Tr0ub4dor&3")
	if err != nil || email != "alice@corp.com" {
		t.Errorf("NewAccount() = %q, %v; want alice@corp.com", email, err)
	}
}

// Every problem is reported at once, as one 422.
func TestNewAccountReportsEveryField(t *testing.T) {
	tests := []struct {
		name, email, password string
		want                  []shared.FieldError
	}{
		{"both empty", "", "", []shared.FieldError{
			{Field: "email", Code: "required", Message: "is required"},
			{Field: "password", Code: "required", Message: "is required"},
		}},
		{"bad address, weak password", "not-an-address", "short", []shared.FieldError{
			{Field: "email", Code: "invalid_format", Message: "is not a valid e-mail address"},
			{Field: "password", Code: "weak_password", Message: "must be 8-128 characters with an upper-case letter, a lower-case letter, a digit and one of " + passwordSpecials},
		}},
		{"too long", strings.Repeat("a", 250) + "@x.com", "Tr0ub4dor&3", []shared.FieldError{
			{Field: "email", Code: "too_long", Message: "must be at most 255 characters"},
		}},
		{"common password", "bob@corp.com", "Password1!", []shared.FieldError{
			{Field: "password", Code: "common_password", Message: "is too common"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAccount(rules, tt.email, tt.password)
			var se *shared.Error
			if !errors.As(err, &se) || se.Kind != shared.KindInvalid || se.Code != shared.CodeValidationFailed || !slices.Equal(se.Fields, tt.want) {
				t.Errorf("NewAccount() = %+v, want validation_failed with %+v", err, tt.want)
			}
		})
	}
}
