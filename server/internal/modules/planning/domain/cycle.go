package domain

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Cycle is a cycle of a project as the module stores it (M6 P1 spec 2.2): its
// status is not stored but read from its dates on the project's day
// (StatusOf).
type Cycle struct {
	ID, ProjectID, WorkspaceID uuid.UUID
	Name, Description          string
	Dates                      Dates
	OwnedByID                  *uuid.UUID
	SortOrder                  float64
	ArchivedAt                 *time.Time
	CreatedAt, UpdatedAt       time.Time
	CreatedByID, UpdatedByID   *uuid.UUID
}

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
