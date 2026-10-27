package httpadapter

import (
	"maps"
	"slices"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

const draftListPattern = "GET /api/v0/workspaces/{slug}/drafts"

func checkDraftQuery(raw map[string][]string) error {
	var fields []shared.FieldError
	for _, key := range slices.Sorted(maps.Keys(raw)) {
		if key != "limit" && key != "cursor" {
			fields = append(fields, shared.FieldError{Field: key, Code: shared.FieldNotAllowed, Message: "is not allowed for this list"})
		} else if len(raw[key]) != 1 {
			fields = append(fields, shared.FieldError{Field: key, Code: shared.FieldInvalidFormat, Message: "must be given once"})
		}
	}
	if len(fields) > 0 {
		return shared.Invalid(fields...)
	}
	return nil
}
