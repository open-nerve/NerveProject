// Package domain holds the workspace module's rules (M3 design 6.2): pure
// functions and values.
package domain

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Workspace is a workspace as a member sees it (M3 design 5.2): its columns,
// the caller's role and the number of active members.
type Workspace struct {
	ID               uuid.UUID
	Name             string
	Slug             string
	OrganizationSize *string // nil: not given
	Timezone         string
	Role             shared.Role // the caller's
	TotalMembers     int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// NewWorkspace is what the caller asks for when creating a workspace (M3
// design 5.1). A nil Timezone asks for DefaultTimezone.
type NewWorkspace struct {
	Name             string
	Slug             string
	OrganizationSize *string
	Timezone         *string
}

// DefaultTimezone is a new workspace's time zone when none is given, the
// column's default (M3 design 4.2).
const DefaultTimezone = "UTC"

// The lengths of workspaces.name, varchar(80), and workspaces.slug,
// varchar(48), in characters.
const (
	maxNameLength = 80
	maxSlugLength = 48
)

// organizationSizes are the web form's options (web/packages/constants/src/
// workspace.ts:10), which the column's CHECK holds too (M3 design 4.2).
var organizationSizes = []string{"Just myself", "2-10", "11-50", "51-200", "201-500", "500+"}

// slugPattern is a slug's spelling: lower case only (M3 design 3.10).
var slugPattern = regexp.MustCompile(`^[a-z0-9_-]+$`)

// SlugReason is why a slug cannot name a new workspace (M3 design 5.1, the
// reason of SlugAvailability).
type SlugReason string

// The reasons.
const (
	SlugInvalid  SlugReason = "invalid"  // not 1–48 of a-z, 0-9, - and _
	SlugReserved SlugReason = "reserved" // on the reserved list
	SlugTaken    SlugReason = "taken"    // an undeleted workspace has it
)

// CheckSlug returns why slug cannot name a new workspace without looking at
// the workspaces, invalid or reserved, or "" when it can. Whether it is taken
// is the repository's to say.
func CheckSlug(slug string) SlugReason {
	switch {
	case !spelledAsSlug(slug):
		return SlugInvalid
	case isReserved(slug):
		return SlugReserved
	}
	return ""
}

// spelledAsSlug reports whether s is 1–48 of a-z, 0-9, - and _: the slug's
// spelling, which the reserved list's names have too.
func spelledAsSlug(s string) bool {
	return utf8.RuneCountInString(s) <= maxSlugLength && slugPattern.MatchString(s)
}

// CheckNewWorkspace checks w (M3 design 3.10, 3.13), as Plane's serializer
// does (serializers/workspace.py:48-67) with the slug in lower case only:
//   - a name of 1–80 characters, with a letter or a digit (Unicode), without
//     a web address, and without NUL or invalid UTF-8, which the database
//     cannot store (the API's body check refuses invalid UTF-8 first; the
//     command line passes it here);
//   - a slug of 1–48 lower-case letters, digits, - and _, not reserved;
//   - an organization size of the web form's options;
//   - a time zone that shared.ValidTimezone accepts.
//
// Every problem is reported at once, as one 422 validation_failed. Whether
// the slug is taken is the database's to say.
func CheckNewWorkspace(w NewWorkspace) error {
	var fields []shared.FieldError
	if f := checkName(w.Name); f != nil {
		fields = append(fields, *f)
	}
	if f := checkSlug(w.Slug); f != nil {
		fields = append(fields, *f)
	}
	if w.OrganizationSize != nil && !slices.Contains(organizationSizes, *w.OrganizationSize) {
		fields = append(fields, shared.FieldError{Field: "organization_size", Code: shared.FieldInvalidFormat, Message: "is not a known organization size"})
	}
	if w.Timezone != nil && !shared.ValidTimezone(*w.Timezone) {
		fields = append(fields, shared.FieldError{Field: "timezone", Code: shared.FieldInvalidFormat, Message: "is not a known time zone"})
	}
	if len(fields) > 0 {
		return shared.Invalid(fields...)
	}
	return nil
}

func checkName(name string) *shared.FieldError {
	field := "name"
	switch {
	case name == "":
		return &shared.FieldError{Field: field, Code: shared.FieldTooShort, Message: "must not be empty"}
	case utf8.RuneCountInString(name) > maxNameLength:
		return &shared.FieldError{Field: field, Code: shared.FieldTooLong, Message: fmt.Sprintf("must be at most %d characters", maxNameLength)}
	case strings.ContainsRune(name, 0):
		return &shared.FieldError{Field: field, Code: shared.FieldInvalidFormat, Message: "must not contain a NUL character"}
	case !utf8.ValidString(name):
		return &shared.FieldError{Field: field, Code: shared.FieldInvalidFormat, Message: "must be valid UTF-8"}
	case !strings.ContainsFunc(name, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsNumber(r) }):
		return &shared.FieldError{Field: field, Code: shared.FieldInvalidFormat, Message: "must contain a letter or a digit"}
	case shared.ContainsURL(name):
		return &shared.FieldError{Field: field, Code: shared.FieldContainsURL, Message: "must not contain a web address"}
	}
	return nil
}

func checkSlug(slug string) *shared.FieldError {
	field := "slug"
	switch reason := CheckSlug(slug); {
	case slug == "":
		return &shared.FieldError{Field: field, Code: shared.FieldTooShort, Message: "must not be empty"}
	case utf8.RuneCountInString(slug) > maxSlugLength:
		return &shared.FieldError{Field: field, Code: shared.FieldTooLong, Message: fmt.Sprintf("must be at most %d characters", maxSlugLength)}
	case reason == SlugInvalid:
		return &shared.FieldError{Field: field, Code: shared.FieldInvalidFormat, Message: "may hold only lower-case letters, digits, - and _"}
	case reason == SlugReserved:
		return &shared.FieldError{Field: field, Code: shared.FieldNotAllowed, Message: "is reserved"}
	}
	return nil
}
