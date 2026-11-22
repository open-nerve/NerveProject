package domain

import (
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// RecentVisitKind is what a recent visit is of (P3d spec 2.3, 2.4): a work
// item or a project, the two kinds a visit is recorded of (M7 design 3.9,
// D37). The body of a recording and the list's query name it alike.
type RecentVisitKind string

// The two kinds of recent visits.
const (
	RecentVisitIssue   RecentVisitKind = "issue"
	RecentVisitProject RecentVisitKind = "project"
)

// RecentVisitCap is the most undeleted recent visits one keeps in a
// workspace, of both kinds together and whether he sees their targets now
// or not (spec 2.5): a recording deletes, in its transaction, those past
// this many by visit, the newest first.
const RecentVisitCap = 20

// CheckRecentVisitKind is nil for one of the two kinds, as they are
// written; for any other, the one problem of entity_type, which the use
// case answers as a 422 before it reads anything (spec 2.4, 2.7).
func CheckRecentVisitKind(k RecentVisitKind) error {
	if k == RecentVisitIssue || k == RecentVisitProject {
		return nil
	}
	return shared.Invalid(shared.FieldError{Field: "entity_type", Code: shared.FieldInvalidFormat, Message: "must be one of issue, project"})
}

// RecentVisit is one account's latest visit of a target in a workspace as
// the module stores it (spec 2.3, 2.5). Only he records and lists it (spec
// 2.2); he has at most one undeleted visit of a target there.
type RecentVisit struct {
	ID, WorkspaceID uuid.UUID
	// ProjectID is its target's project: the project itself for a
	// project's visit, a work item's project for a work item's.
	ProjectID uuid.UUID
	UserID    uuid.UUID
	Kind      RecentVisitKind
	// EntityID is its target's id.
	EntityID uuid.UUID
	// VisitedAt is the moment of his latest visit. A recording stamps the
	// row's creation, or its revisit's change, at this moment too.
	VisitedAt time.Time
}
