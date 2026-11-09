package app

import (
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// IssueListSettings checks the words of a page's saved list settings
// without a directory or an actor: the issue module's ListSettings, which
// bootstrap hands over as it is (M6 P3b spec R5). A nil top-level value is
// omitted; a non-nil empty filters map clears them.
type IssueListSettings interface {
	Validate(filters map[string][]string, display *shared.ListDisplayFilters) []shared.FieldError
}

// checkListSettings is the refusal of the words p gives (M6 P3b spec 2.3):
// each filter value and display word settings refuses at its field, in its
// order, as one 422 validation_failed; nil when it refuses none. Display
// properties are booleans whose keys the shared type fixes, with no word to
// check.
func checkListSettings[F domain.PageFilters](p domain.PreferencesPatch[F], settings IssueListSettings) error {
	var filters map[string][]string
	if p.Filters != nil {
		filters = (*p.Filters).Map()
	}
	if fields := settings.Validate(filters, p.DisplayFilters); len(fields) > 0 {
		return shared.Invalid(fields...)
	}
	return nil
}
