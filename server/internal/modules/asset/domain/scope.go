package domain

import "uuid"

// Purpose fixes both the authorized host category and the image requirement.
type Purpose string

const (
	UserAvatar            Purpose = "USER_AVATAR"
	UserCover             Purpose = "USER_COVER"
	WorkspaceLogo         Purpose = "WORKSPACE_LOGO"
	ProjectCover          Purpose = "PROJECT_COVER"
	IssueAttachment       Purpose = "ISSUE_ATTACHMENT"
	IssueDescription      Purpose = "ISSUE_DESCRIPTION"
	CommentDescription    Purpose = "COMMENT_DESCRIPTION"
	CommentAttachment     Purpose = "COMMENT_ATTACHMENT"
	DraftIssueDescription Purpose = "DRAFT_ISSUE_DESCRIPTION"
	DraftIssueAttachment  Purpose = "DRAFT_ISSUE_ATTACHMENT"
)

type ScopeKind string

const (
	ScopeUser       ScopeKind = "user"
	ScopeWorkspace  ScopeKind = "workspace"
	ScopeProject    ScopeKind = "project"
	ScopeNewProject ScopeKind = "new_project"
	ScopeIssue      ScopeKind = "issue"
	ScopeComment    ScopeKind = "comment"
	ScopeNewComment ScopeKind = "new_comment"
	ScopeDraft      ScopeKind = "draft"
)

// Scope names one existing host; user names only the authenticated applicant.
type Scope struct {
	Kind ScopeKind
	ID   uuid.UUID
}

// Placement contains owner-resolved parents, never independently trusted client IDs.
type Placement struct {
	Scope                                                       Scope
	UserID, WorkspaceID, ProjectID, IssueID, CommentID, DraftID uuid.UUID
}

func (p Purpose) Valid() bool {
	switch p {
	case UserAvatar, UserCover, WorkspaceLogo, ProjectCover, IssueAttachment, IssueDescription,
		CommentDescription, CommentAttachment, DraftIssueDescription, DraftIssueAttachment:
		return true
	default:
		return false
	}
}

func (p Purpose) ImageRequired() bool {
	switch p {
	case UserAvatar, UserCover, WorkspaceLogo, ProjectCover, IssueDescription, CommentDescription, DraftIssueDescription:
		return true
	default:
		return false
	}
}

func (s Scope) ValidFor(p Purpose) bool {
	if (s.Kind == ScopeUser) != (s.ID == uuid.Nil()) {
		return false
	}
	switch p {
	case UserAvatar, UserCover:
		return s.Kind == ScopeUser
	case WorkspaceLogo:
		return s.Kind == ScopeWorkspace
	case ProjectCover:
		return s.Kind == ScopeProject || s.Kind == ScopeNewProject
	case IssueAttachment, IssueDescription:
		return s.Kind == ScopeIssue
	case CommentDescription, CommentAttachment:
		return s.Kind == ScopeComment || s.Kind == ScopeNewComment
	case DraftIssueDescription, DraftIssueAttachment:
		return s.Kind == ScopeDraft
	default:
		return false
	}
}

func (p Placement) Validate(purpose Purpose) error {
	if p.UserID == uuid.Nil() || !p.Scope.ValidFor(purpose) {
		return ErrInvalidAsset
	}
	var match bool
	switch p.Scope.Kind {
	case ScopeUser:
		match = p.parents(false, false, false, false, false)
	case ScopeWorkspace, ScopeNewProject:
		match = p.parents(true, false, false, false, false) && p.Scope.ID == p.WorkspaceID
	case ScopeProject:
		match = p.parents(true, true, false, false, false) && p.Scope.ID == p.ProjectID
	case ScopeIssue, ScopeNewComment:
		match = p.parents(true, true, true, false, false) && p.Scope.ID == p.IssueID
	case ScopeComment:
		match = p.parents(true, true, true, true, false) && p.Scope.ID == p.CommentID
	case ScopeDraft:
		match = p.parents(true, p.ProjectID != uuid.Nil(), false, false, true) && p.Scope.ID == p.DraftID
	}
	if !match {
		return ErrInvalidAsset
	}
	return nil
}

func (p Placement) parents(workspace, project, issue, comment, draft bool) bool {
	return (p.WorkspaceID != uuid.Nil()) == workspace && (p.ProjectID != uuid.Nil()) == project &&
		(p.IssueID != uuid.Nil()) == issue && (p.CommentID != uuid.Nil()) == comment && (p.DraftID != uuid.Nil()) == draft
}
