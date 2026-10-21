package domain

import (
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A work item's parent (M4 design 3.9): an undeleted work item of its
// project that it is under, however deep, or none. A work item is never
// under itself, nor under a work item under it, so the work items of a
// project make trees; an archived work item takes no new work item under
// it (3.8), and a guest who sees only his own work items puts none under
// another's (3.4).

// ParentNotAllowed is the problem of a parent asked for that may not be
// the work item's (M4 design 3.9).
func ParentNotAllowed() shared.FieldError {
	return shared.FieldError{Field: "parent_id", Code: shared.FieldNotAllowed,
		Message: "must be a work item of the project that is not archived, and neither this one nor under it"}
}
