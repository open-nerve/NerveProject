package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// PublishDraftDeps reuses the one write boundary and creation core.
type PublishDraftDeps struct {
	Locks    DraftLocks
	Decoder  IssueCreatePayloadDecoder
	Creation *IssueCreation
	Drafts   DraftWriter
}
type PublishDraft struct{ d PublishDraftDeps }

func NewPublishDraft(d PublishDraftDeps) *PublishDraft { return &PublishDraft{d: d} }

// Execute decodes only the locked stored payload. Creation and consuming the
// draft share the transaction, actor and sole moment returned by the core.
func (u *PublishDraft) Execute(ctx context.Context, id uuid.UUID) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	var created domain.Issue
	err = u.d.Locks.publish(ctx, actor, id, func(ctx context.Context, h publishHeld) error {
		if h.Issue.project.Archived {
			return domain.ErrProjectArchived
		}
		in, err := u.d.Decoder.DecodeCreatePayload(h.Draft.Payload)
		if err != nil {
			return err
		}
		in, err = domain.CheckNewIssue(in)
		if err != nil {
			return err
		}
		row, at, err := u.d.Creation.createLocked(ctx, actor.UserID, *h.Draft.ProjectID, h.Issue, in)
		if err != nil {
			return err
		}
		if err = u.d.Drafts.DeleteDraft(ctx, DraftDeletion{ID: id, By: actor.UserID, At: at}); err != nil {
			return err
		}
		created = row
		return nil
	})
	if err != nil {
		return domain.Issue{}, err
	}
	return created, nil
}
