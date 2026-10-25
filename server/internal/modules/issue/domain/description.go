package domain

import (
	"unicode/utf8"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

const defaultDescriptionHTML = "<p></p>"
const maxDescription = 500000

// DescriptionChange separates persistence from ordinary audit changes.
type DescriptionChange struct {
	Content        RichText
	Write, Changed bool
}

// CompareDescription compares sanitized values without parsing HTML. Migration
// saves always persist, but only content identity changes produce audit work.
func CompareDescription(old, next RichText, skip bool) DescriptionChange {
	if skip {
		return DescriptionChange{Content: next, Write: true, Changed: !old.Identity.Equal(next.Identity)}
	}
	changed := old.HTML != next.HTML
	return DescriptionChange{Content: next, Write: changed, Changed: changed}
}

func checkDescription(html *string) *shared.FieldError {
	if html != nil && utf8.RuneCountInString(*html) > maxDescription {
		return &shared.FieldError{Field: "description_html", Code: shared.FieldTooLong, Message: "must be at most 500000 characters"}
	}
	return nil
}

func checkSkipActivity(p IssuePatch) *shared.FieldError {
	if p.SkipActivity != nil && (p.DescriptionHTML == nil || p.Name != nil || p.StateID != nil || p.Priority != nil || p.SortOrder != nil ||
		p.AssigneeIDs != nil || p.LabelIDs != nil || p.SetStartDate || p.SetTargetDate || p.SetParentID) {
		return &shared.FieldError{Field: "skip_activity", Code: shared.FieldNotAllowed, Message: "requires description_html without other patch fields"}
	}
	return nil
}
