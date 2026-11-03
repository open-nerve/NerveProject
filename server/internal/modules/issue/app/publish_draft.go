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
	Transfer DraftAssetTransfer
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
		prepared, err := u.d.Creation.prepareLocked(ctx, actor.UserID, *h.Draft.ProjectID, h.Issue, in)
		if err != nil {
			return err
		}
		q := DraftAssetRequest{ActorID: actor.UserID, WorkspaceID: h.Draft.WorkspaceID, DraftID: id,
			OldProjectID: h.Draft.ProjectID, NewProjectID: h.Draft.ProjectID, ImageIDs: prepared.content.Identity.AssetImageIDs(), Publish: true}
		at, err := u.d.Transfer.LockDraftAssets(ctx, q)
		if err != nil {
			return err
		}
		row, err := u.d.Creation.writePreparedAt(ctx, prepared, at)
		if err != nil {
			return err
		}
		if err := u.d.Transfer.MoveDraftAssets(ctx, q, row.ID, at); err != nil {
			return err
		}
		if err = u.d.Drafts.DeleteDraft(ctx, DraftDeletion{ID: id, By: actor.UserID, At: at}); err != nil {
			return err
		}
		items := []domain.Issue{row}
		if err := projectAttachments(ctx, u.d.Creation.d.Attachments, items); err != nil {
			return err
		}
		created = items[0]
		return nil
	})
	if err != nil {
		return domain.Issue{}, err
	}
	return created, nil
}
