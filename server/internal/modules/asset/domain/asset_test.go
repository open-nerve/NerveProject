package domain

import (
	"errors"
	"strings"
	"testing"
	"uuid"
)

// Removing the purpose/scope gate would allow a user image to name another host.
func TestAssetPurposeScopes(t *testing.T) {
	id := uuid.MustParse("9ec63b2c-4dc0-4eb2-b1d9-65f55f12f42e")
	for _, tc := range []struct {
		purpose Purpose
		kinds   []ScopeKind
		image   bool
	}{
		{UserAvatar, []ScopeKind{ScopeUser}, true}, {UserCover, []ScopeKind{ScopeUser}, true},
		{WorkspaceLogo, []ScopeKind{ScopeWorkspace}, true}, {ProjectCover, []ScopeKind{ScopeProject, ScopeNewProject}, true},
		{IssueAttachment, []ScopeKind{ScopeIssue}, false}, {IssueDescription, []ScopeKind{ScopeIssue}, true},
		{CommentDescription, []ScopeKind{ScopeComment, ScopeNewComment}, true}, {CommentAttachment, []ScopeKind{ScopeComment, ScopeNewComment}, false},
		{DraftIssueDescription, []ScopeKind{ScopeDraft}, true}, {DraftIssueAttachment, []ScopeKind{ScopeDraft}, false},
	} {
		t.Run(string(tc.purpose), func(t *testing.T) {
			if tc.purpose.ImageRequired() != tc.image {
				t.Fatalf("image policy %s", tc.purpose)
			}
			for _, kind := range []ScopeKind{ScopeUser, ScopeWorkspace, ScopeProject, ScopeNewProject, ScopeIssue, ScopeComment, ScopeNewComment, ScopeDraft, "page"} {
				scope := Scope{Kind: kind, ID: id}
				if kind == ScopeUser {
					scope.ID = uuid.Nil()
				}
				want := false
				for _, k := range tc.kinds {
					if k == kind {
						want = true
					}
				}
				if got := scope.ValidFor(tc.purpose); got != want {
					t.Errorf("scope %s valid=%v want %v", kind, got, want)
				}
			}
		})
	}
	for _, s := range []Scope{{Kind: ScopeUser, ID: id}, {Kind: ScopeIssue}, {Kind: ""}} {
		if s.ValidFor(UserAvatar) || s.ValidFor(IssueAttachment) {
			t.Errorf("bad scope accepted: %+v", s)
		}
	}
	if (Scope{Kind: ScopeIssue, ID: id}).ValidFor("PAGE_DESCRIPTION") {
		t.Fatal("unknown purpose accepted")
	}
}

// The declared size must never be truncated and attachment MIME never grants inline policy.
func TestAssetApplicationBounds(t *testing.T) {
	good := NewAsset{Purpose: UserAvatar, Scope: Scope{Kind: ScopeUser}, Name: "头像.png", Size: 3, ContentType: "image/png"}
	if err := ValidateNewAsset(good, 5242880); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*NewAsset)
		want   error
	}{
		{"zero", func(a *NewAsset) { a.Size = 0 }, ErrInvalidAsset},
		{"limit", func(a *NewAsset) { a.Size = 5242881 }, ErrAssetTooLarge},
		{"hard limit", func(a *NewAsset) { a.Size = 104857601 }, ErrAssetTooLarge},
		{"empty name", func(a *NewAsset) { a.Name = "" }, ErrInvalidAsset},
		{"long unicode", func(a *NewAsset) { a.Name = strings.Repeat("图", 256) }, ErrInvalidAsset},
		{"path", func(a *NewAsset) { a.Name = "../image.png" }, ErrInvalidAsset},
		{"control", func(a *NewAsset) { a.Name = "x\x00.png" }, ErrInvalidAsset},
		{"bad type", func(a *NewAsset) { a.ContentType = "image/png; charset=utf-8" }, ErrInvalidAsset},
		{"svg", func(a *NewAsset) { a.ContentType = "image/svg+xml" }, ErrAssetTypeUnsupported},
		{"wrong scope", func(a *NewAsset) { a.Scope.Kind = ScopeIssue }, ErrInvalidAsset},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := good
			tc.change(&a)
			if err := ValidateNewAsset(a, 5242880); !errors.Is(err, tc.want) {
				t.Fatalf("error=%v want %v", err, tc.want)
			}
		})
	}
	a := good
	a.Name = strings.Repeat("图", 255)
	a.Size = 1
	if err := ValidateNewAsset(a, 1); err != nil {
		t.Fatal(err)
	}
	a.Purpose = IssueAttachment
	a.Scope = Scope{Kind: ScopeIssue, ID: uuid.NewV7()}
	a.ContentType = "image/svg+xml"
	if err := ValidateNewAsset(a, 1); err != nil {
		t.Fatal("attachment unnecessarily rejected", err)
	}
	for _, n := range []int64{0, -1, 104857601} {
		if ValidateNewAsset(good, n) == nil {
			t.Errorf("invalid config %d accepted", n)
		}
	}
}

// Missing and extra parents must not silently create assets outside the authorized host.
func TestAssetPlacement(t *testing.T) {
	u, w, p, i, c, d := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	cases := []struct {
		purpose Purpose
		place   Placement
	}{
		{UserAvatar, Placement{UserID: u, Scope: Scope{Kind: ScopeUser}}},
		{WorkspaceLogo, Placement{UserID: u, WorkspaceID: w, Scope: Scope{Kind: ScopeWorkspace, ID: w}}},
		{ProjectCover, Placement{UserID: u, WorkspaceID: w, Scope: Scope{Kind: ScopeNewProject, ID: w}}},
		{ProjectCover, Placement{UserID: u, WorkspaceID: w, ProjectID: p, Scope: Scope{Kind: ScopeProject, ID: p}}},
		{IssueAttachment, Placement{UserID: u, WorkspaceID: w, ProjectID: p, IssueID: i, Scope: Scope{Kind: ScopeIssue, ID: i}}},
		{CommentDescription, Placement{UserID: u, WorkspaceID: w, ProjectID: p, IssueID: i, Scope: Scope{Kind: ScopeNewComment, ID: i}}},
		{CommentAttachment, Placement{UserID: u, WorkspaceID: w, ProjectID: p, IssueID: i, CommentID: c, Scope: Scope{Kind: ScopeComment, ID: c}}},
		{DraftIssueDescription, Placement{UserID: u, WorkspaceID: w, DraftID: d, Scope: Scope{Kind: ScopeDraft, ID: d}}},
		{DraftIssueAttachment, Placement{UserID: u, WorkspaceID: w, ProjectID: p, DraftID: d, Scope: Scope{Kind: ScopeDraft, ID: d}}},
	}
	for _, tc := range cases {
		if err := tc.place.Validate(tc.purpose); err != nil {
			t.Fatalf("%+v: %v", tc.place, err)
		}
		bad := tc.place
		bad.UserID = uuid.Nil()
		if bad.Validate(tc.purpose) == nil {
			t.Fatal("no applicant accepted")
		}
		bad = tc.place
		bad.Scope.ID = uuid.NewV7()
		if bad.Validate(tc.purpose) == nil {
			t.Fatal("scope id disagrees with host")
		}
	}
	for _, change := range []func(*Placement){func(x *Placement) { x.WorkspaceID = uuid.Nil() }, func(x *Placement) { x.ProjectID = uuid.Nil() }, func(x *Placement) { x.CommentID = c }, func(x *Placement) { x.DraftID = d }} {
		x := cases[4].place
		change(&x)
		if x.Validate(IssueAttachment) == nil {
			t.Fatalf("invalid issue parents %+v", x)
		}
	}
}
