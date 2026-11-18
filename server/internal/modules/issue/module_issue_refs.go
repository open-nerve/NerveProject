package issue

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// IssueRefs is what the module tells the notification module about the
// work items a notification's change names, its parent and its relations'
// other ends (M7 P1c spec 2.6, R4): which of them a reader sees now, each
// with its project, its number and its name. issue.Provide builds it over
// the pool; bootstrap composes it with project's NotificationIdentifiers
// for the notification module.
type IssueRefs interface {
	// IssueRefs answers, by id, those of the work items ids that are
	// undeleted, archived or not, in a project among grants, archived or
	// not, and that the guest rule, domain.Issue.HiddenFrom, does not hide
	// from reader by his grant there; another workspace's work item is in
	// no project of grants. One read, in the transaction ctx carries or on
	// the pool; none for no ids.
	IssueRefs(ctx context.Context, reader uuid.UUID, grants []ProjectGrant, ids []uuid.UUID) (map[uuid.UUID]IssueRef, error)
}

// ProjectGrant, a project whose work items a reader may read, and
// IssueRef, a work item as IssueRefs names it, are the values IssueRefs
// takes and hands over: bootstrap converts them.
type (
	ProjectGrant = app.ProjectGrant
	IssueRef     = app.IssueRef
)
