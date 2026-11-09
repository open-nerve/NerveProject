package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// Filters are a project's saved list filters: the keys every page's list
// has (shared.ListFilters). A nil slice omits its key; an explicitly empty
// slice is retained and means no filtering.
type Filters struct {
	shared.ListFilters
}
