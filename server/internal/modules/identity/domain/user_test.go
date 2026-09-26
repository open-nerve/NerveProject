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

func TestCheckUserPatchAcceptsAValidPatch(t *testing.T) {
	name, long, tz := "Ada", strings.Repeat("界", 255), "America/St_Johns"
	for _, p := range []UserPatch{
		{},
		{FirstName: &name, LastName: &long, DisplayName: &long, Timezone: &tz},
		{FirstName: ptr(""), LastName: ptr("")},
		{Timezone: ptr("UTC")},
	} {
		if err := CheckUserPatch(p); err != nil {
			t.Errorf("CheckUserPatch(%+v) = %v, want nil", p, err)
		}
	}
}

// Every problem is reported at once, as one 422.
func TestCheckUserPatchReportsEveryField(t *testing.T) {
	long := strings.Repeat("界", 256)
	tests := []struct {
		name  string
		patch UserPatch
		want  []shared.FieldError
	}{
		{"a web address in the first name", UserPatch{FirstName: ptr("see x.io")},
			[]shared.FieldError{{Field: "first_name", Code: "contains_url", Message: "must not contain a web address"}}},
		{"a web address in the last name", UserPatch{LastName: ptr("https://x")},
			[]shared.FieldError{{Field: "last_name", Code: "contains_url", Message: "must not contain a web address"}}},
		{"a long first name", UserPatch{FirstName: &long},
			[]shared.FieldError{{Field: "first_name", Code: "too_long", Message: "must be at most 255 characters"}}},
		{"NUL in the last name", UserPatch{LastName: ptr("a\x00")},
			[]shared.FieldError{{Field: "last_name", Code: "invalid_format", Message: "must not contain a NUL character"}}},
		{"an empty display name", UserPatch{DisplayName: ptr("")},
			[]shared.FieldError{{Field: "display_name", Code: "too_short", Message: "must not be empty"}}},
		{"a long display name", UserPatch{DisplayName: &long},
			[]shared.FieldError{{Field: "display_name", Code: "too_long", Message: "must be at most 255 characters"}}},
		{"an unknown time zone", UserPatch{Timezone: ptr("Mars/Olympus")},
			[]shared.FieldError{{Field: "user_timezone", Code: "invalid_format", Message: "is not a known time zone"}}},
		{"the host's zone", UserPatch{Timezone: ptr("Local")},
			[]shared.FieldError{{Field: "user_timezone", Code: "invalid_format", Message: "is not a known time zone"}}},
		{"no time zone", UserPatch{Timezone: ptr("")},
			[]shared.FieldError{{Field: "user_timezone", Code: "invalid_format", Message: "is not a known time zone"}}},
		{"all at once", UserPatch{FirstName: ptr("x.io"), LastName: &long, DisplayName: ptr(""), Timezone: ptr("Nowhere")}, []shared.FieldError{
			{Field: "first_name", Code: "contains_url", Message: "must not contain a web address"},
			{Field: "last_name", Code: "too_long", Message: "must be at most 255 characters"},
			{Field: "display_name", Code: "too_short", Message: "must not be empty"},
			{Field: "user_timezone", Code: "invalid_format", Message: "is not a known time zone"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckUserPatch(tt.patch)

			var se *shared.Error
			if !errors.As(err, &se) || se.Code != shared.CodeValidationFailed || !slices.Equal(se.Fields, tt.want) {
				t.Errorf("CheckUserPatch() = %+v, want validation_failed with %+v", err, tt.want)
			}
		})
	}
}
