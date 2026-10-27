package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateIssueLinkDeps are the ports consumed by link editing.
type UpdateIssueLinkDeps struct {
	Locks  Locks
	Links  LinkUpdater
	Clock  Clock
	Events shared.EventPublisher
}

// UpdateIssueLink edits a visible link under its project's write lock.
type UpdateIssueLink struct{ d UpdateIssueLinkDeps }

func NewUpdateIssueLink(d UpdateIssueLinkDeps) *UpdateIssueLink { return &UpdateIssueLink{d: d} }
func (u *UpdateIssueLink) Execute(ctx context.Context, id uuid.UUID, patch domain.LinkPatch) (domain.Link, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Link{}, err
	}
	patch, err = domain.CheckLinkPatch(patch)
	if err != nil {
		return domain.Link{}, err
	}
	current, found, err := u.d.Links.GetLink(ctx, id)
	if err != nil {
		return domain.Link{}, err
	}
	if !found {
		return domain.Link{}, domain.ErrLinkNotFound
	}
	w := write{issue: &current.IssueID, action: domain.ActionUpdate, notFound: domain.ErrLinkNotFound, check: linkTargetCheck(u.d.Links, id, &current)}
	err = u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		if h.project.Archived {
			return domain.ErrProjectArchived
		}
		if h.issue.ArchivedAt != nil {
			return domain.ErrIssueArchived
		}
		next := current
		if patch.URL != nil {
			next.URL = *patch.URL
		}
		if patch.SetTitle {
			next.Title = patch.Title
		}
		urlChanged := next.URL != current.URL
		titleChanged := next.Title != nil && current.Title == nil || next.Title == nil && current.Title != nil || next.Title != nil && current.Title != nil && *next.Title != *current.Title
		if !urlChanged && !titleChanged {
			return nil
		}
		if urlChanged {
			exists, err := u.d.Links.LinkURLExists(ctx, h.issue.ID, id, next.URL)
			if err != nil {
				return err
			}
			if exists {
				return domain.ErrLinkExists
			}
		}
		at := u.d.Clock.Now()
		next.UpdatedAt = at
		if err := u.d.Links.MarkChanged(ctx, ChangedRows{IDs: []uuid.UUID{h.issue.ID}, By: actor.UserID, At: at}); err != nil {
			return err
		}
		updated, err := u.d.Links.UpdateLink(ctx, LinkWrite{Link: next, By: actor.UserID, At: at})
		if err != nil {
			return err
		}
		stamp := stamp{workspace: h.issue.WorkspaceID, project: h.issue.ProjectID, issue: h.issue.ID, by: actor.UserID, at: at}
		changes := make([]shared.Change, 0, 2)
		if urlChanged {
			if err := u.d.Links.InsertActivities(ctx, stamp.activity([]domain.Activity{domain.LinkUpdatedActivity(current, updated)})); err != nil {
				return err
			}
			changes = append(changes, shared.Change{Field: "url", Old: eventPtr(current.URL), New: eventPtr(updated.URL)})
		}
		if titleChanged {
			changes = append(changes, shared.Change{Field: "title", Old: current.Title, New: updated.Title})
		}
		current = updated
		return u.d.Events.Publish(ctx, stamp.linkEvent(domain.EventLinkUpdated, id, changes))
	})
	if err != nil {
		return domain.Link{}, err
	}
	return current, nil
}
