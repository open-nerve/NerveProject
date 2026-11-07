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

// maxName is the length of cycles.name and modules.name, in characters.
const maxName = 255

// CleanName is name without its surrounding white space, or the problem of
// that: empty, longer than a cycle's or a module's name may be, or with NUL,
// which Postgres cannot store (M6 design 3.1, 3.2).
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

// ReadCycle is a cycle as the module answers it: with its status on the
// project's day.
type ReadCycle struct {
	Cycle
	Status Status
}

// NewCycle is a cycle as a creation gives it (createCycle's body).
type NewCycle struct {
	Name, Description string
	Dates             Dates
}

// CheckNewCycle is n with its name cleaned, or every problem of its values,
// as validation_failed: checked before anything is read (M6 P1 spec R2).
func CheckNewCycle(n NewCycle) (NewCycle, error) {
	var fields []shared.FieldError
	name, problem := CleanName(n.Name)
	if problem != nil {
		fields = append(fields, *problem)
	}
	n.Name = name
	if problem := checkDescription(n.Description); problem != nil {
		fields = append(fields, *problem)
	}
	if problem := CheckDates(n.Dates); problem != nil {
		fields = append(fields, *problem)
	}
	if len(fields) > 0 {
		return NewCycle{}, shared.Invalid(fields...)
	}
	return n, nil
}

// checkDescription refuses a description with NUL, which Postgres cannot
// store.
func checkDescription(d string) *shared.FieldError {
	if strings.ContainsRune(d, 0) {
		return &shared.FieldError{Field: "description", Code: shared.FieldInvalidFormat, Message: "must not contain a NUL character"}
	}
	return nil
}

// firstSortOrder is the sort order of a project's first cycle, and
// sortOrderStep how far before the lowest a new cycle goes (Plane
// cycle.py:88-95).
const (
	firstSortOrder = 65535
	sortOrderStep  = 10000
)

// NextSortOrder is the sort order of a new cycle: before lowest, the lowest
// of the project's undeleted cycles, archived ones too, or firstSortOrder
// when it has none.
func NextSortOrder(lowest *float64) float64 {
	if lowest == nil {
		return firstSortOrder
	}
	return *lowest - sortOrderStep
}
