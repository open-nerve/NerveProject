package issue

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// NavigationIssues is what the module tells the navigation module about
// the work items its recent visits name (M7 P3d spec 2.5, 2.7, R3): where
// each is, and which of them a caller sees now, each with what a recent
// visit is answered with. Each reads once, without a lock, in the
// transaction ctx carries or on the pool, and not at all for no ids.
// issue.Provide builds it over the pool; bootstrap converts it into
// navigation's Issues.
type NavigationIssues interface {
	// NavigationIssuePlaces answers, by id, where each of the undeleted
	// work items among ids is, archived or not, of any workspace: its
	// workspace and its project. It decides nothing.
	NavigationIssuePlaces(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]NavigationIssuePlace, error)
	// NavigationIssues answers, by id, those of the work items ids that
	// are undeleted, archived or not, in a project among grants, archived
	// or not, and that the guest rule, domain.Issue.HiddenFrom, does not
	// hide from reader by his grant there; another workspace's work item
	// is in no project of grants. Each tells whether it is in its
	// project's intake (domain.Issue.InTriage).
	NavigationIssues(ctx context.Context, reader uuid.UUID, grants []ProjectGrant, ids []uuid.UUID) (map[uuid.UUID]NavigationIssue, error)
}

// NavigationIssuePlace, where a work item is, and NavigationIssue, a work
// item as a recent visit is answered with it, are the values
// NavigationIssues hands over: bootstrap converts them.
type (
	NavigationIssuePlace = app.NavigationIssuePlace
	NavigationIssue      = app.NavigationIssue
)
