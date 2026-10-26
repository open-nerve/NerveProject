package domain

import (
	"fmt"
	"slices"
	"strings"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DefaultSortOrder is a project's place in a member's sidebar when he has
// no other place in the workspace's projects:
// project_user_properties.sort_order's default (M3 design 3.18).
const DefaultSortOrder = 65535

// SortOrderFirst is the place in a member's sidebar of a project he is made
// a member of by its creation: before the others, 10000 less than the least
// of his places in the workspace's projects, lowest, or DefaultSortOrder
// when he has none (Plane's ProjectMember.save, db/models/project.py:226-
// 239; M3 design 3.18).
func SortOrderFirst(lowest *float64) float64 {
	if lowest == nil {
		return DefaultSortOrder
	}
	return *lowest - 10000
}

// tabs are a project's tabs, in the web app's order (M3 design 5.2;
// core/components/navigation/use-navigation-items.ts:38-78). The first,
// the work items, is the default tab and never hidden.
var tabs = []string{"work_items", "cycles", "modules", "views", "intake"}

// Navigation is the tab bar of a project's header, as one account has it:
// the tab the project opens on, and the tabs moved under "more".
type Navigation struct {
	DefaultTab     string
	HideInMoreMenu []string
}

// Preferences are an account's display settings in a project (M3 design
// 3.18, 5.2; M4/P5): its tabs, sidebar place and issue-list presentation.
type Preferences struct {
	Navigation        Navigation
	SortOrder         float64
	Filters           Filters
	DisplayFilters    DisplayFilters
	DisplayProperties DisplayProperties
}

// PreferencesPatch is a change of Preferences: a nil field stays as it
// is; each given top-level field replaces that field whole.
type PreferencesPatch struct {
	Navigation        *Navigation
	SortOrder         *float64
	Filters           *Filters
	DisplayFilters    *DisplayFilters
	DisplayProperties *DisplayProperties
}

// DefaultPreferences are an account's settings in a project while he has
// no row of them: the columns' defaults (M3 design 4.8), which the store's
// test holds equal.
func DefaultPreferences() Preferences {
	return Preferences{Navigation: Navigation{DefaultTab: tabs[0], HideInMoreMenu: []string{}}, SortOrder: DefaultSortOrder,
		DisplayFilters:    DisplayFilters{Layout: "list", OrderBy: "-created_at", SubIssue: true, ShowEmptyGroups: true, Calendar: DisplayCalendar{Layout: "month"}},
		DisplayProperties: DisplayProperties{Assignee: true, StartDate: true, DueDate: true, Labels: true, Key: true, Priority: true, State: true, SubIssueCount: true, Link: true, CreatedOn: true, UpdatedOn: true}}
}

// Apply returns p with the fields patch gives changed.
func (p Preferences) Apply(patch PreferencesPatch) Preferences {
	if patch.Navigation != nil {
		p.Navigation = *patch.Navigation
	}
	if patch.SortOrder != nil {
		p.SortOrder = *patch.SortOrder
	}
	if patch.Filters != nil {
		p.Filters = *patch.Filters
	}
	if patch.DisplayFilters != nil {
		p.DisplayFilters = *patch.DisplayFilters
	}
	if patch.DisplayProperties != nil {
		p.DisplayProperties = *patch.DisplayProperties
	}
	return p
}

// CheckPreferencesPatch checks the navigation p gives: its default tab one
// of the tabs; each tab it hides one of the tabs but the work items, and
// hidden once. Every problem is reported at once, as one 422
// validation_failed; a tab the web app does not have is refused here, not
// stored (M3 design 12, P4b).
func CheckPreferencesPatch(p PreferencesPatch) error {
	if p.Navigation == nil {
		return nil
	}
	var found []*shared.FieldError
	if !slices.Contains(tabs, p.Navigation.DefaultTab) {
		found = append(found, &shared.FieldError{Field: "navigation.default_tab", Code: shared.FieldInvalidFormat,
			Message: "is not one of " + strings.Join(tabs, ", ")})
	}
	hideable := tabs[1:]
	for i, tab := range p.Navigation.HideInMoreMenu {
		field := fmt.Sprintf("navigation.hide_in_more_menu[%d]", i)
		switch {
		case !slices.Contains(hideable, tab):
			found = append(found, &shared.FieldError{Field: field, Code: shared.FieldInvalidFormat, Message: "is not one of " + strings.Join(hideable, ", ")})
		case slices.Contains(p.Navigation.HideInMoreMenu[:i], tab):
			found = append(found, &shared.FieldError{Field: field, Code: shared.FieldDuplicate, Message: "is listed before"})
		}
	}
	return invalid(found...)
}
