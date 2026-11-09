package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// CycleFilters are the saved list filters of a cycle's page (M6 P3b spec
// 2.3, R5, R8): the keys every page's list has (shared.ListFilters) and the
// modules'; not the cycles', the page's list being its cycle's already. A
// nil slice omits its key; an explicitly empty one is kept and means no
// filtering.
type CycleFilters struct {
	shared.ListFilters
	ModuleID []string `json:"module_id,omitzero"`
}

// Map returns the given keys by their JSON names, as the issue module's
// ListSettings checks them: the shared keys and the modules'. A nil slice is
// left out, an empty one is kept.
func (f CycleFilters) Map() map[string][]string {
	out := f.ListFilters.Map()
	if f.ModuleID != nil {
		out["module_id"] = f.ModuleID
	}
	return out
}

// ModuleFilters are the saved list filters of a module's page: the keys
// every page's list has and the cycles'; not the modules', the page's list
// being its module's already. A nil slice omits its key; an explicitly
// empty one is kept and means no filtering.
type ModuleFilters struct {
	shared.ListFilters
	CycleID []string `json:"cycle_id,omitzero"`
}

// Map returns the given keys by their JSON names, as the issue module's
// ListSettings checks them: the shared keys and the cycles'. A nil slice is
// left out, an empty one is kept.
func (f ModuleFilters) Map() map[string][]string {
	out := f.ListFilters.Map()
	if f.CycleID != nil {
		out["cycle_id"] = f.CycleID
	}
	return out
}

// PageFilters are the saved list filters of a cycle's or a module's page.
type PageFilters interface {
	CycleFilters | ModuleFilters
	Map() map[string][]string
}

// Preferences are an account's display settings on a cycle's or a module's
// page, each caller's own (M6 P3b spec 2.3): the page's list filters, and
// the display filters and properties every page has (R5).
type Preferences[F PageFilters] struct {
	Filters           F
	DisplayFilters    shared.ListDisplayFilters
	DisplayProperties shared.ListDisplayProperties
}

// PreferencesPatch is a change of Preferences: a nil field stays as it is;
// each given field replaces that field whole.
type PreferencesPatch[F PageFilters] struct {
	Filters           *F
	DisplayFilters    *shared.ListDisplayFilters
	DisplayProperties *shared.ListDisplayProperties
}

// The settings of a cycle's page and of a module's, and their changes.
type (
	CyclePreferences       = Preferences[CycleFilters]
	CyclePreferencesPatch  = PreferencesPatch[CycleFilters]
	ModulePreferences      = Preferences[ModuleFilters]
	ModulePreferencesPatch = PreferencesPatch[ModuleFilters]
)

// DefaultPreferences are an account's settings on a page while he has no
// row of them: the columns' defaults (00046, 00047), which the store's test
// holds equal: no filter, and the display defaults every page has.
func DefaultPreferences[F PageFilters]() Preferences[F] {
	return Preferences[F]{DisplayFilters: shared.DefaultListDisplayFilters(), DisplayProperties: shared.DefaultListDisplayProperties()}
}

// Apply returns p with the fields patch gives changed.
func (p Preferences[F]) Apply(patch PreferencesPatch[F]) Preferences[F] {
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
