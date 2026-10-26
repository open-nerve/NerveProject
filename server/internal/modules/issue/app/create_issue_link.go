package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CreateIssueLinkDeps are the ports consumed by link creation.
type CreateIssueLinkDeps struct {
	Locks  Locks
	Links  LinkCreator
	Clock  Clock
	Events shared.EventPublisher
}

// CreateIssueLink creates a link under its work item's project lock.
type CreateIssueLink struct{ d CreateIssueLinkDeps }

func NewCreateIssueLink(d CreateIssueLinkDeps) *CreateIssueLink { return &CreateIssueLink{d: d} }
func (u *CreateIssueLink) Execute(ctx context.Context, id uuid.UUID, input domain.NewLink) (domain.Link, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Link{}, err
	}
	input, err = domain.CheckNewLink(input)
	if err != nil {
		return domain.Link{}, err
	}
	var created domain.Link
	err = u.d.Locks.run(ctx, actor, write{issue: &id, action: domain.ActionUpdate, notFound: domain.ErrIssueNotFound}, func(ctx context.Context, h held) error {
		if h.project.Archived {
			return domain.ErrProjectArchived
		}
		if h.issue.ArchivedAt != nil {
			return domain.ErrIssueArchived
		}
		exists, err := u.d.Links.LinkURLExists(ctx, id, uuid.UUID{}, input.URL)
		if err != nil {
			return err
		}
		if exists {
			return domain.ErrLinkExists
		}
		at := u.d.Clock.Now()
		created = domain.Link{ID: uuid.NewV7(), WorkspaceID: h.issue.WorkspaceID, ProjectID: h.issue.ProjectID, IssueID: id, URL: input.URL, Title: input.Title, Metadata: domain.LinkMetadata{}, CreatedByID: &actor.UserID, CreatedAt: at, UpdatedAt: at}
		if err := u.d.Links.MarkChanged(ctx, ChangedRows{IDs: []uuid.UUID{id}, By: actor.UserID, At: at}); err != nil {
			return err
		}
		if err := u.d.Links.InsertLink(ctx, LinkWrite{Link: created, By: actor.UserID, At: at}); err != nil {
			return err
		}
		stamp := stamp{workspace: created.WorkspaceID, project: created.ProjectID, issue: id, by: actor.UserID, at: at}
		if err := u.d.Links.InsertActivities(ctx, stamp.activity([]domain.Activity{domain.LinkCreatedActivity(created)})); err != nil {
			return err
		}
		changes := []shared.Change{{Field: "url", New: eventPtr(created.URL)}, {Field: "title", New: created.Title}}
		return u.d.Events.Publish(ctx, stamp.linkEvent(domain.EventLinkCreated, created.ID, changes))
	})
	if err != nil {
		return domain.Link{}, err
	}
	return created, nil
}
