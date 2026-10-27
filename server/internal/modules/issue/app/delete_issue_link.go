package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DeleteIssueLinkDeps are the ports consumed by link deletion.
type DeleteIssueLinkDeps struct {
	Locks  Locks
	Links  LinkDeleter
	Clock  Clock
	Events shared.EventPublisher
}

// DeleteIssueLink soft-deletes a visible link under its project's write lock.
type DeleteIssueLink struct{ d DeleteIssueLinkDeps }

func NewDeleteIssueLink(d DeleteIssueLinkDeps) *DeleteIssueLink { return &DeleteIssueLink{d: d} }
func (u *DeleteIssueLink) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	current, found, err := u.d.Links.GetLink(ctx, id)
	if err != nil {
		return err
	}
	if !found {
		return domain.ErrLinkNotFound
	}
	w := write{issue: &current.IssueID, action: domain.ActionUpdate, notFound: domain.ErrLinkNotFound, check: linkTargetCheck(u.d.Links, id, &current)}
	return u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		if h.project.Archived {
			return domain.ErrProjectArchived
		}
		if h.issue.ArchivedAt != nil {
			return domain.ErrIssueArchived
		}
		at := u.d.Clock.Now()
		if err := u.d.Links.MarkChanged(ctx, ChangedRows{IDs: []uuid.UUID{h.issue.ID}, By: actor.UserID, At: at}); err != nil {
			return err
		}
		if err := u.d.Links.DeleteLink(ctx, LinkDeletion{ID: id, By: actor.UserID, At: at}); err != nil {
			return err
		}
		stamp := stamp{workspace: h.issue.WorkspaceID, project: h.issue.ProjectID, issue: h.issue.ID, by: actor.UserID, at: at}
		if err := u.d.Links.InsertActivities(ctx, stamp.activity([]domain.Activity{domain.LinkDeletedActivity(current)})); err != nil {
			return err
		}
		return u.d.Events.Publish(ctx, stamp.linkEvent(domain.EventLinkDeleted, id, []shared.Change{{Field: "url", Old: eventPtr(current.URL)}, {Field: "title", Old: current.Title}}))
	})
}
