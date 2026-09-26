// Package domain holds the identity module's rules (M2 design 6.2): pure
// functions and values, no I/O.
package domain

import (
	"time"
	"unicode/utf8"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// User is an account as the API shows it.
type User struct {
	ID          uuid.UUID
	Email       string
	FirstName   string
	LastName    string
	DisplayName string
	Timezone    string
	CreatedAt   time.Time
}

// UserPatch is a partial update of an account: a nil field stays as it is.
// The e-mail address is not in it: only the administrator's command changes
// it (M2 decision 1).
type UserPatch struct {
	FirstName   *string
	LastName    *string
	DisplayName *string
	Timezone    *string
}

// maxNameLength is the length of users' name columns, varchar(255), in
// characters.
const maxNameLength = 255

// CheckUserPatch checks p (M2 design 4.2): first and last names of at most
// 255 characters without an address in them, as Plane checks them
// (contains_url); a display name of 1–255 characters; no NUL, which the
// database cannot store; and a time zone that Go's time package knows, not
// "Local". Every problem is reported at once, as one 422
// validation_failed.
func CheckUserPatch(p UserPatch) error {
	var fields []shared.FieldError
	for _, name := range []struct {
		field string
		value *string
	}{{"first_name", p.FirstName}, {"last_name", p.LastName}} {
		if name.value == nil {
			continue
		}
		if f := checkText(name.field, *name.value, false, maxNameLength); f != nil {
			fields = append(fields, *f)
		} else if containsURL(*name.value) {
			fields = append(fields, shared.FieldError{Field: name.field, Code: shared.FieldContainsURL, Message: "must not contain a web address"})
		}
	}
	if p.DisplayName != nil {
		if f := checkText("display_name", *p.DisplayName, true, maxNameLength); f != nil {
			fields = append(fields, *f)
		}
	}
	if p.Timezone != nil && !validTimezone(*p.Timezone) {
		fields = append(fields, shared.FieldError{Field: "user_timezone", Code: shared.FieldInvalidFormat, Message: "is not a known time zone"})
	}
	if len(fields) > 0 {
		return shared.Invalid(fields...)
	}
	return nil
}

// validTimezone reports whether name is an IANA time zone that Go's time
// package loads (M2 design 4.2): "UTC" and every zone of the embedded
// tzdata, not "Local" or "", which LoadLocation takes for the host's zone
// and for UTC.
func validTimezone(name string) bool {
	if name == "" || name == "Local" {
		return false
	}
	_, err := time.LoadLocation(name)
	return err == nil
}

// NewAccount checks the e-mail address and the password of a new account
// and returns the normalized address. Every problem is reported at once, as
// one 422 validation_failed.
func NewAccount(rules *PasswordRules, email, password string) (string, error) {
	email = NormalizeEmail(email)
	var fields []shared.FieldError
	switch {
	case email == "":
		fields = append(fields, shared.FieldError{Field: "email", Code: shared.FieldRequired, Message: "is required"})
	case utf8.RuneCountInString(email) > MaxEmailLength:
		fields = append(fields, shared.FieldError{Field: "email", Code: shared.FieldTooLong, Message: "must be at most 255 characters"})
	case !ValidEmail(email):
		fields = append(fields, shared.FieldError{Field: "email", Code: shared.FieldInvalidFormat, Message: "is not a valid e-mail address"})
	}
	if f := rules.Check("password", password, email); f != nil {
		fields = append(fields, *f)
	}
	if len(fields) > 0 {
		return "", shared.Invalid(fields...)
	}
	return email, nil
}
