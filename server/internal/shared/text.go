package shared

import "strings"

// CheckText refuses NUL in the text of field, which Postgres cannot store
// in text nor in jsonb.
func CheckText(field, s string) *FieldError {
	if strings.ContainsRune(s, 0) {
		return &FieldError{Field: field, Code: FieldInvalidFormat, Message: "must not contain a NUL character"}
	}
	return nil
}
