package domain

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Network is who of the workspace sees a project besides its members (M3
// design 3.4, 4.6).
type Network int

// The two networks, the values of projects.network.
const (
	// NetworkPrivate: the project's members and the workspace's admins.
	NetworkPrivate Network = 0
	// NetworkPublic: also the workspace's members.
	NetworkPublic Network = 2
)

// NewProject is what the caller asks for when creating a project (M3 design
// 5.1). A nil Network asks for a public project, the column's default; a
// nil Timezone asks for the workspace's (3.19).
type NewProject struct {
	Name        string
	Identifier  string
	Description string
	Network     *Network
	LeadID      *uuid.UUID
	LogoProps   LogoProps
	Timezone    *string
}

// The lengths of projects.name, varchar(255), and of an identifier, in
// characters (M3 design 3.19).
const (
	maxNameLength       = 255
	maxIdentifierLength = 10
)

// forbiddenNameCharacters may not occur in a name: Plane's
// FORBIDDEN_IDENTIFIER_CHARS_PATTERN (db/models/project.py:143), which the
// column's CHECK holds too (M3 design 3.19, 4.6).
const forbiddenNameCharacters = `&+,:;$^}{*=?@#|'<>.()%!-`

// identifierPattern is an identifier's spelling, in upper case: the web
// form's (core/components/project/create/common-attributes.tsx:96-107),
// which the column's CHECK holds too.
var identifierPattern = regexp.MustCompile(`^[A-Z0-9ÇŞĞİÖÜ]+$`)

// Identifier is the identifier that s stands for: s in upper case, as the
// web form sends it and the column stores it (M3 design 3.19).
func Identifier(s string) string {
	return strings.ToUpper(s)
}

// CheckNewProject checks p (M3 design 3.19) and returns it as it is stored:
// the identifier in upper case, the network given or public.
//   - a name of 1–255 characters, not blank, without NUL, which the
//     database cannot store, and without any of & + , : ; $ ^ } { * = ? @ #
//     | ' < > . ( ) % ! - (not_allowed);
//   - an identifier that is 1–10 of A-Z, 0-9 and ÇŞĞİÖÜ in upper case;
//   - a description without NUL; a network of 0 or 2; a time zone that
//     shared.ValidTimezone accepts; logo_props by checkLogoProps.
//
// Every problem is reported at once, as one 422 validation_failed. Whether
// the name or the identifier is taken is the database's to say; whether the
// lead may lead, the use case's, under its locks (M3 design 3.6).
func CheckNewProject(p NewProject) (NewProject, error) {
	p.Identifier = Identifier(p.Identifier)
	if p.Network == nil {
		public := NetworkPublic
		p.Network = &public
	}
	found := []*shared.FieldError{checkName(p.Name), checkIdentifier(p.Identifier), checkText("description", p.Description),
		checkNetwork(*p.Network), checkTimezone(p.Timezone)}
	if err := invalid(append(found, checkLogoProps(p.LogoProps)...)...); err != nil {
		return NewProject{}, err
	}
	return p, nil
}

// CanLead reports whether an account of workspace role role, an active
// member of the workspace, may lead a new project: an admin or a member,
// not a guest (M3 design 3.19). The set is named, not a bound.
func CanLead(role shared.Role) bool {
	return role == shared.RoleAdmin || role == shared.RoleMember
}

// invalid is the 422 of the problems found, in order, or nil when there is
// none.
func invalid(found ...*shared.FieldError) error {
	var fields []shared.FieldError
	for _, f := range found {
		if f != nil {
			fields = append(fields, *f)
		}
	}
	if len(fields) > 0 {
		return shared.Invalid(fields...)
	}
	return nil
}

func checkName(name string) *shared.FieldError {
	field := "name"
	switch {
	case strings.TrimSpace(name) == "":
		return &shared.FieldError{Field: field, Code: shared.FieldTooShort, Message: "must not be empty"}
	case utf8.RuneCountInString(name) > maxNameLength:
		return &shared.FieldError{Field: field, Code: shared.FieldTooLong, Message: fmt.Sprintf("must be at most %d characters", maxNameLength)}
	case strings.ContainsAny(name, forbiddenNameCharacters):
		return &shared.FieldError{Field: field, Code: shared.FieldNotAllowed,
			Message: "must not contain any of & + , : ; $ ^ } { * = ? @ # | ' < > . ( ) % ! -"}
	}
	return checkText(field, name)
}

func checkIdentifier(id string) *shared.FieldError {
	field := "identifier"
	switch {
	case id == "":
		return &shared.FieldError{Field: field, Code: shared.FieldTooShort, Message: "must not be empty"}
	case utf8.RuneCountInString(id) > maxIdentifierLength:
		return &shared.FieldError{Field: field, Code: shared.FieldTooLong,
			Message: fmt.Sprintf("must be at most %d characters", maxIdentifierLength)}
	case !identifierPattern.MatchString(id):
		return &shared.FieldError{Field: field, Code: shared.FieldInvalidFormat, Message: "may hold only A-Z, 0-9 and ÇŞĞİÖÜ"}
	}
	return nil
}

// checkText refuses NUL in the text of field, which Postgres cannot store
// in text nor in jsonb.
func checkText(field, s string) *shared.FieldError {
	if strings.ContainsRune(s, 0) {
		return &shared.FieldError{Field: field, Code: shared.FieldInvalidFormat, Message: "must not contain a NUL character"}
	}
	return nil
}

func checkNetwork(n Network) *shared.FieldError {
	if n != NetworkPrivate && n != NetworkPublic {
		return &shared.FieldError{Field: "network", Code: shared.FieldInvalidFormat, Message: "must be 0, private, or 2, public"}
	}
	return nil
}

func checkTimezone(zone *string) *shared.FieldError {
	if zone != nil && !shared.ValidTimezone(*zone) {
		return &shared.FieldError{Field: "timezone", Code: shared.FieldInvalidFormat, Message: "is not a known time zone"}
	}
	return nil
}
