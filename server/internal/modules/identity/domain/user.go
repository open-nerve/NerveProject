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
