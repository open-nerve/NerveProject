package domain

import (
	"testing"
	"time"
	"uuid"
)

func TestBoundAssetAccessCategories(t *testing.T) {
	u, w, p, i, c, d := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	at := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		purpose             Purpose
		placement           Placement
		attachment, content bool
	}{
		{IssueAttachment, Placement{UserID: u, WorkspaceID: w, ProjectID: p, IssueID: i, Scope: Scope{Kind: ScopeIssue, ID: i}}, true, true},
		{IssueDescription, Placement{UserID: u, WorkspaceID: w, ProjectID: p, IssueID: i, Scope: Scope{Kind: ScopeIssue, ID: i}}, false, true},
		{CommentAttachment, Placement{UserID: u, WorkspaceID: w, ProjectID: p, IssueID: i, CommentID: c, Scope: Scope{Kind: ScopeComment, ID: c}}, true, true},
		{CommentDescription, Placement{UserID: u, WorkspaceID: w, ProjectID: p, IssueID: i, CommentID: c, Scope: Scope{Kind: ScopeComment, ID: c}}, false, true},
		{DraftIssueAttachment, Placement{UserID: u, WorkspaceID: w, DraftID: d, Scope: Scope{Kind: ScopeDraft, ID: d}}, true, true},
		{DraftIssueDescription, Placement{UserID: u, WorkspaceID: w, ProjectID: p, DraftID: d, Scope: Scope{Kind: ScopeDraft, ID: d}}, false, true},
		{CommentAttachment, Placement{UserID: u, WorkspaceID: w, ProjectID: p, IssueID: i, Scope: Scope{Kind: ScopeNewComment, ID: i}}, false, false},
		{CommentDescription, Placement{UserID: u, WorkspaceID: w, ProjectID: p, IssueID: i, Scope: Scope{Kind: ScopeNewComment, ID: i}}, false, false},
		{UserAvatar, Placement{UserID: u, Scope: Scope{Kind: ScopeUser}}, false, false},
		{UserCover, Placement{UserID: u, Scope: Scope{Kind: ScopeUser}}, false, false},
		{WorkspaceLogo, Placement{UserID: u, WorkspaceID: w, Scope: Scope{Kind: ScopeWorkspace, ID: w}}, false, false},
		{ProjectCover, Placement{UserID: u, WorkspaceID: w, ProjectID: p, Scope: Scope{Kind: ScopeProject, ID: p}}, false, false},
	} {
		t.Run(string(tc.purpose)+"/"+string(tc.placement.Scope.Kind), func(t *testing.T) {
			a := Asset{Purpose: tc.purpose, Placement: tc.placement, BoundAt: &at}
			if a.IsOrdinaryAttachment() != tc.attachment || a.IsBoundContent() != tc.content {
				t.Fatal("content category differs from purpose and existing host")
			}
			a.BoundAt = nil
			if a.IsBoundContent() {
				t.Fatal("unbound preparation is not shared content")
			}
			a.BoundAt = &at
			a.Placement.UserID = uuid.Nil()
			if a.IsBoundContent() || a.IsOrdinaryAttachment() {
				t.Fatal("invalid placement became shared content")
			}
		})
	}
}
