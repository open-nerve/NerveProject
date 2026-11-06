package domain

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// maxName is the length of cycles.name, in characters.
const maxName = 255

// CleanName is name without its surrounding white space, or the problem of
// that: empty, longer than cycles.name, or with NUL, which Postgres cannot
// store (M6 design 3.1).
func CleanName(name string) (string, *shared.FieldError) {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return "", &shared.FieldError{Field: "name", Code: shared.FieldTooShort, Message: "must not be empty"}
	case utf8.RuneCountInString(name) > maxName:
		return "", &shared.FieldError{Field: "name", Code: shared.FieldTooLong, Message: fmt.Sprintf("must be at most %d characters", maxName)}
	case strings.ContainsRune(name, 0):
		return "", &shared.FieldError{Field: "name", Code: shared.FieldInvalidFormat, Message: "must not contain a NUL character"}
	}
	return name, nil
}
