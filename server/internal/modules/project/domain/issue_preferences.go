package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// Filters are a project's saved list filters: the keys every page's list
// has (shared.ListFilters) and the project's own two, its cycles and
// modules. A nil slice omits its key; an explicitly empty slice is
// retained and means no filtering.
type Filters struct {
	shared.ListFilters
	CycleID  []string `json:"cycle_id,omitzero"`
	ModuleID []string `json:"module_id,omitzero"`
}

// Map returns the given keys by their JSON names, as the issue module's
// ListSettings checks them: the shared keys and the project's own two. A
// nil slice is left out, an empty one is kept.
func (f Filters) Map() map[string][]string {
	out := f.ListFilters.Map()
	if f.CycleID != nil {
		out["cycle_id"] = f.CycleID
	}
	if f.ModuleID != nil {
		out["module_id"] = f.ModuleID
	}
	return out
}
