package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type ListIssuesDeps struct {
	ReadTx  shared.TxManager
	Locks   Locks
	Listing *Listing
}

// ListIssues reads an explicitly authorized project, including an archived one.
type ListIssues struct{ d ListIssuesDeps }

func NewListIssues(d ListIssuesDeps) *ListIssues { return &ListIssues{d: d} }
func (u *ListIssues) Execute(ctx context.Context, projectID uuid.UUID, raw map[string][]string) (IssueListPage, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return IssueListPage{}, err
	}
	q, err := domain.ParseListQuery(raw, domain.ProjectList, actor.UserID)
	if err != nil {
		return IssueListPage{}, err
	}
	var page IssueListPage
	err = u.d.ReadTx.WithinTx(ctx, func(ctx context.Context) error {
		wid, found, err := u.d.Locks.projects.Find(ctx, projectID)
		if err != nil {
			return err
		}
		if !found {
			return domain.ErrProjectNotFound
		}
		action := domain.ActionRead
		if q.Filters.Archived {
			action = domain.ActionArchive
		}
		grant, err := u.d.Locks.decide(ctx, actor, action, wid, projectID, nil, domain.ErrProjectNotFound)
		if err != nil {
			return err
		}
		page, err = u.d.Listing.Read(ctx, ListingInput{ActorID: actor.UserID, Projects: []VisibleProject{{ID: projectID, Role: grant.ProjectRole, OwnOnly: grant.OwnOnly}}, Query: q})
		return err
	})
	if err != nil {
		return IssueListPage{}, err
	}
	return page, nil
}
