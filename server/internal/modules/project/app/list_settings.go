package app

import (
	"errors"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// IssueListSettings checks saved words without querying a directory or interpreting an actor.
// Nil top-level values mean omitted; a nonnil empty filters map means clear.
type IssueListSettings interface {
	Validate(filters map[string][]string, display *shared.ListDisplayFilters) []shared.FieldError
}

// checkListSettings preserves navigation's error order, then the settings port's order.
// Display properties are booleans whose keys the shared type fixes; there is no word in them to check.
func checkListSettings(p domain.PreferencesPatch, settings IssueListSettings) error {
	var fields []shared.FieldError
	if err := domain.CheckPreferencesPatch(p); err != nil {
		var invalid *shared.Error
		if !errors.As(err, &invalid) || invalid.Code != shared.CodeValidationFailed {
			return err
		}
		fields = append(fields, invalid.Fields...)
	}
	var filters map[string][]string
	if f := p.Filters; f != nil {
		filters = f.Map()
	}
	fields = append(fields, settings.Validate(filters, p.DisplayFilters)...)
	if len(fields) > 0 {
		return shared.Invalid(fields...)
	}
	return nil
}
