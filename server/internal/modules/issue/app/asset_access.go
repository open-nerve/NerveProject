package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type AssetIntent uint8

const (
	AssetRead AssetIntent = iota + 1
	AssetPrepare
	AssetRemove
)

type AssetHostKind uint8

const (
	AssetIssue AssetHostKind = iota + 1
	AssetComment
	AssetNewComment
	AssetDraft
)

type AssetRequest struct {
	ID     uuid.UUID
	Kind   AssetHostKind
	Intent AssetIntent
}
type AssetHost struct{ WorkspaceID, ProjectID, IssueID, CommentID, DraftID uuid.UUID }

var ErrAssetScopeChanged = errors.New("issue asset host scope changed")

// AssetProject is the project of an asset's host as AssetProjectReader
// reads it: whether it is archived, and whether its intake is turned on,
// read in one statement (M7 P2c spec 2.4, R4).
type AssetProject struct {
	Archived   bool
	IntakeView bool
}

// AssetProjectReader reads the project id under a project lock already held
// by the caller; found is false when there is none. It must not take a
// second project lock or enter a write gate.
type AssetProjectReader interface {
	AssetProject(ctx context.Context, id uuid.UUID) (p AssetProject, found bool, err error)
}
type AssetAccessDeps struct {
	Locks    Locks
	Drafts   DraftLocks
	Comments CommentReader
	Projects AssetProjectReader
}
type AssetAccess struct{ d AssetAccessDeps }

func NewAssetAccess(d AssetAccessDeps) *AssetAccess { return &AssetAccess{d: d} }

// Inspect decides q for actor before any lock (read): the host's 404, its
// role's 403, and on a work item in its project's intake the submitter's
// 403 (M7 P2c spec 2.2). Neither the intake's toggle nor its item's status
// is judged here: Check judges them, under the lock.
func (a *AssetAccess) Inspect(ctx context.Context, actor shared.Actor, q AssetRequest) (AssetHost, error) {
	h, _, err := a.read(ctx, actor, q)
	return h, err
}

// Check decides q again for actor under the locks the asset's write holds,
// its host's project's among them, in this order (M7 P2c spec 2.2): read's
// refusals, on the work item and the grant as they are now, so that one
// accepted while the lock waited is decided on issue.update; a host other
// than before, ErrAssetScopeChanged; then, for a write, its project not
// there, the host's 404; for a write on a work item in its project's
// intake (inIntake), the project archived (409 project.archived), then its
// intake turned off (409 issue.intake_disabled) (refuseIntakeOff), then
// the item declined or a duplicate (409 issue.intake_closed), by its
// stored status alone, no clock read (domain.IntakeFacts.CheckOpen), so
// that a snooze whose time has come is written; for any other host, the
// project archived; then the work item archived (409 issue.archived).
func (a *AssetAccess) Check(ctx context.Context, actor shared.Actor, q AssetRequest, before AssetHost) error {
	h, item, err := a.read(ctx, actor, q)
	if err != nil {
		return err
	}
	if h != before {
		return ErrAssetScopeChanged
	}
	if q.Kind == AssetDraft || q.Intent == AssetRead {
		return nil
	}
	p, found, err := a.d.Projects.AssetProject(ctx, h.ProjectID)
	if err != nil {
		return err
	}
	if !found {
		if q.Kind == AssetComment {
			return domain.ErrCommentNotFound
		}
		return domain.ErrIssueNotFound
	}
	switch {
	case inIntake(q, item):
		// AssetProject read p under the project's lock the asset's write
		// holds, the one a submission, a triage and an edit take.
		if err := refuseIntakeOff(LockedProject{Archived: p.Archived, IntakeView: p.IntakeView}); err != nil {
			return err
		}
		if err := item.Intake.CheckOpen(); err != nil {
			return err
		}
	case p.Archived:
		return domain.ErrProjectArchived
	}
	if item.ArchivedAt != nil {
		return domain.ErrIssueArchived
	}
	return nil
}

// inIntake reports whether q, its work item read as i, is an asset's write
// on a work item in its project's intake (M7 design 3.6, P2c spec 2.2):
// on the work item itself (AssetIssue), not on its comments, which keep
// their rules, while it is in the intake (domain.Issue.InTriage), not
// accepted. intake.update and its submitter rule decide such a write, and,
// under the lock, the intake's toggle and its item's status (Check).
func inIntake(q AssetRequest, i domain.Issue) bool {
	return q.Kind == AssetIssue && q.Intent != AssetRead && i.InTriage()
}

// read reads the host q names for actor, without a lock of its own, as
// Inspect and Check do: a draft through DraftLocks; otherwise the work
// item, its own or a comment's, read once and decided (Locks.seeChoosing)
// on the action q asks of it: a read issue.read; a comment's write
// issue_comment.write, with its author rule; a new comment's
// issue_comment.create; a write on the work item itself issue.update, or
// intake.update while it is in its project's intake (inIntake), whose
// submitter rule (refuseNonSubmitter) follows the decision and its guest
// rule's 404 (M7 P2c spec 2.2). A work item's 404 is issue.not_found,
// never the intake's, so that AssetHostNotFound knows it. It answers the
// host and the work item as read, none for a draft.
func (a *AssetAccess) read(ctx context.Context, actor shared.Actor, q AssetRequest) (AssetHost, domain.Issue, error) {
	var none domain.Issue
	if q.ID == uuid.Nil() || q.Intent < AssetRead || q.Intent > AssetRemove {
		return AssetHost{}, none, errors.New("invalid issue asset request")
	}
	if q.Kind == AssetDraft {
		d, err := a.d.Drafts.read(ctx, actor, q.ID)
		if err != nil {
			return AssetHost{}, none, err
		}
		h := AssetHost{WorkspaceID: d.WorkspaceID, DraftID: d.ID}
		if d.ProjectID != nil {
			h.ProjectID = *d.ProjectID
		}
		return h, none, nil
	}
	id, action, notFound := q.ID, domain.ActionRead, error(domain.ErrIssueNotFound)
	var current domain.Comment
	var check func(context.Context, domain.Issue, shared.Grant) error
	switch q.Kind {
	case AssetIssue:
		if q.Intent != AssetRead {
			action = domain.ActionUpdate
		}
	case AssetNewComment:
		if q.Intent != AssetRead {
			action = domain.ActionCreateComment
		}
	case AssetComment:
		c, found, err := a.d.Comments.GetComment(ctx, q.ID)
		if err != nil {
			return AssetHost{}, none, err
		}
		if !found {
			return AssetHost{}, none, domain.ErrCommentNotFound
		}
		id, notFound = c.IssueID, domain.ErrCommentNotFound
		check = commentTargetCheck(a.d.Comments, q.ID, &current)
		if q.Intent != AssetRead {
			action = domain.ActionWriteComment
			check = commentCheck(a.d.Comments, q.ID, actor.UserID, q.Intent == AssetRemove, &current)
		}
	default:
		return AssetHost{}, none, errors.New("invalid issue asset host kind")
	}
	choose := func(i domain.Issue) shared.Action {
		if inIntake(q, i) {
			return domain.ActionUpdateIntake
		}
		return action
	}
	i, grant, err := a.d.Locks.seeChoosing(ctx, actor, id, choose, notFound)
	if err != nil {
		return AssetHost{}, none, err
	}
	if inIntake(q, i) {
		if err := refuseNonSubmitter(actor.UserID, i, grant); err != nil {
			return AssetHost{}, none, err
		}
	}
	if check != nil {
		if err := check(ctx, i, grant); err != nil {
			return AssetHost{}, none, err
		}
	}
	h := AssetHost{WorkspaceID: i.WorkspaceID, ProjectID: i.ProjectID, IssueID: i.ID}
	if q.Kind == AssetComment {
		h.CommentID = current.ID
	}
	return h, i, nil
}
