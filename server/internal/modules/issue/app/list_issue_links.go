package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListIssueLinksDeps are the read ports and independent snapshot manager.
type ListIssueLinksDeps struct {
	Locks  Locks
	Links  LinkLister
	ReadTx shared.TxManager
}

// ListIssueLinks reads a visible work item's links in a consistent snapshot.
type ListIssueLinks struct{ d ListIssueLinksDeps }

func NewListIssueLinks(d ListIssueLinksDeps) *ListIssueLinks { return &ListIssueLinks{d: d} }
func (u *ListIssueLinks) Execute(ctx context.Context, id uuid.UUID) ([]domain.Link, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	rows := []domain.Link{}
	err = u.d.ReadTx.WithinTx(ctx, func(ctx context.Context) error {
		if _, _, err := u.d.Locks.see(ctx, actor, id, domain.ActionRead); err != nil {
			return err
		}
		links, err := u.d.Links.Links(ctx, id)
		if err != nil {
			return err
		}
		rows = append(rows, links...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rows, nil
}
