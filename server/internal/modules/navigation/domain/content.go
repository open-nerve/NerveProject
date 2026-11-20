package domain

import (
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Filters are a view's saved list filters (spec 2.3, 2.4): the keys every
// page's saved list has (shared.ListFilters) and the three only one kind
// of view has, a project view's cycle_id and module_id and a workspace
// view's project_id, the contract's union of thirteen. Which of them a
// view may hold, and their values, are its kind's words: the issue
// module's settings of that kind check them (the use case, through its
// port), and the column's CHECK holds each kind's keys. A nil slice omits
// its key; an explicitly empty one is kept and means no filtering for it.
type Filters struct {
	shared.ListFilters
	CycleID   []string `json:"cycle_id,omitzero"`
	ModuleID  []string `json:"module_id,omitzero"`
	ProjectID []string `json:"project_id,omitzero"`
}

// Map returns the given keys by their JSON names, as the issue module's
// settings check them: a nil slice is left out, an empty one is kept.
func (f Filters) Map() map[string][]string {
	out := f.ListFilters.Map()
	for _, entry := range []struct {
		key    string
		values []string
	}{{"cycle_id", f.CycleID}, {"module_id", f.ModuleID}, {"project_id", f.ProjectID}} {
		if entry.values != nil {
			out[entry.key] = entry.values
		}
	}
	return out
}

// Content is what a view's owner writes of it (spec 2.3, 2.4): its name,
// its plain-text description, its icon ({} for none) and the list
// settings it shows its work items with.
type Content struct {
	Name, Description string
	LogoProps         shared.LogoProps
	Filters           Filters
	DisplayFilters    shared.ListDisplayFilters
	DisplayProperties shared.ListDisplayProperties
}

// Equal tells whether c and o are the same content as stored: every field
// equal in value, pointers by what they point to, and an explicitly empty
// filter apart from an omitted one, as their jsonb keeps them. A change
// that leaves a view's content Equal is no change (spec 2.4).
func (c Content) Equal(o Content) bool { return reflect.DeepEqual(c, o) }

// NewView is a view as a creation gives it (createProjectView's and
// createWorkspaceView's body): DisplayFilters and DisplayProperties nil
// when it does not give them; no icon and no filter are the zero values.
type NewView struct {
	Name, Description string
	LogoProps         shared.LogoProps
	Filters           Filters
	DisplayFilters    *shared.ListDisplayFilters
	DisplayProperties *shared.ListDisplayProperties
}

// CheckNewView is the content n creates (spec 2.4, DP-22): its name without
// its surrounding white space, the shared display defaults for the display
// settings it does not give, the rest as given. Or it is the problems of
// its name, description and icon, in that order (checkContent); the use
// case answers them in one 422 with its kind's words after them, before it
// reads anything (spec 2.5).
func CheckNewView(n NewView) (Content, []shared.FieldError) {
	name, problems := checkContent(&n.Name, &n.Description, &n.LogoProps)
	if problems != nil {
		return Content{}, problems
	}
	c := Content{Name: *name, Description: n.Description, LogoProps: n.LogoProps, Filters: n.Filters,
		DisplayFilters: shared.DefaultListDisplayFilters(), DisplayProperties: shared.DefaultListDisplayProperties()}
	if n.DisplayFilters != nil {
		c.DisplayFilters = *n.DisplayFilters
	}
	if n.DisplayProperties != nil {
		c.DisplayProperties = *n.DisplayProperties
	}
	return c, nil
}

// checkContent checks the name, description and icon given, each nil when
// it is not: the name by cleanName, NUL in the description (plain text of
// any length, without assets), the icon by shared.LogoProps.Check. It
// returns the name cleaned, nil when not given, or every problem, in that
// order, nil when there is none.
func checkContent(name, description *string, logo *shared.LogoProps) (*string, []shared.FieldError) {
	var found []*shared.FieldError
	if name != nil {
		cleaned, problem := cleanName(*name)
		name = &cleaned
		found = append(found, problem)
	}
	if description != nil {
		found = append(found, shared.CheckText("description", *description))
	}
	if logo != nil {
		found = append(found, logo.Check()...)
	}
	var problems []shared.FieldError
	for _, f := range found {
		if f != nil {
			problems = append(problems, *f)
		}
	}
	return name, problems
}

// maxName is the length of issue_views.name, in characters.
const maxName = 255

// cleanName is name without its surrounding white space, or the problem of
// that: empty, longer than maxName characters, or with NUL, which Postgres
// cannot store (spec 2.3; the column's CHECK holds only that it is not
// empty).
func cleanName(name string) (string, *shared.FieldError) {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return "", &shared.FieldError{Field: "name", Code: shared.FieldTooShort, Message: "must not be empty"}
	case utf8.RuneCountInString(name) > maxName:
		return "", &shared.FieldError{Field: "name", Code: shared.FieldTooLong, Message: fmt.Sprintf("must be at most %d characters", maxName)}
	}
	return name, shared.CheckText("name", name)
}
