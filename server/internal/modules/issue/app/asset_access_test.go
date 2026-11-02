package app_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type assetArchiveReader struct {
	archived bool
	found    bool
}

func (p assetArchiveReader) AssetProjectArchived(context.Context, uuid.UUID) (bool, bool, error) {
	return p.archived, p.found, nil
}

func TestAssetHostAccessMatrix(t *testing.T) {
	for _, kind := range []app.AssetHostKind{app.AssetIssue, app.AssetComment, app.AssetNewComment} {
		for _, intent := range []app.AssetIntent{app.AssetRead, app.AssetPrepare, app.AssetRemove} {
			_, _, w, c := newCommentWrites()
			c.current.ActorID = alice
			a := app.NewAssetAccess(app.AssetAccessDeps{Locks: w.locks, Comments: c, Projects: assetArchiveReader{found: true}})
			q := app.AssetRequest{Kind: kind, Intent: intent, ID: c.current.IssueID}
			if kind == app.AssetComment {
				q.ID = c.current.ID
			}
			h, err := a.Inspect(t.Context(), shared.Actor{UserID: alice}, q)
			if err != nil || h.IssueID != c.current.IssueID || h.ProjectID != web || h.WorkspaceID != acme {
				t.Fatalf("kind=%d intent=%d host=%+v err=%v", kind, intent, h, err)
			}
			if err := a.Check(t.Context(), shared.Actor{UserID: alice}, q, h); err != nil {
				t.Fatal(err)
			}
			action := "issue.read"
			if intent != app.AssetRead {
				switch kind {
				case app.AssetIssue:
					action = "issue.update"
				case app.AssetComment:
					action = "issue_comment.write"
				case app.AssetNewComment:
					action = "issue_comment.create"
				}
			}
			seen := false
			for _, call := range w.log.calls {
				if strings.HasPrefix(call, "Authorize") {
					seen = true
					if !strings.Contains(call, action) {
						t.Fatalf("wrong action %s", call)
					}
				}
			}
			if !seen {
				t.Fatal("authorization bypassed")
			}
			noCommentResources(t, w)
		}
	}
	for _, intent := range []app.AssetIntent{app.AssetRead, app.AssetPrepare, app.AssetRemove} {
		_, _, w, c := newCommentWrites()
		c.current.ActorID = bob
		w.auth.grants = []shared.Grant{{ProjectAdmin: true}, {ProjectAdmin: true}}
		a := app.NewAssetAccess(app.AssetAccessDeps{Locks: w.locks, Comments: c, Projects: assetArchiveReader{found: true}})
		_, err := a.Inspect(t.Context(), shared.Actor{UserID: alice}, app.AssetRequest{Kind: app.AssetComment, ID: c.current.ID, Intent: intent})
		want := error(nil)
		if intent == app.AssetPrepare {
			want = domain.ErrNotAuthor
		}
		if !errors.Is(err, want) {
			t.Fatalf("admin intent=%d got=%v want=%v", intent, err, want)
		}
	}
	t.Run("hidden before forbidden", func(t *testing.T) {
		_, _, w, c := newCommentWrites()
		w.auth.grants = []shared.Grant{ownOnly}
		w.auth.answers = []error{shared.Forbidden()}
		a := app.NewAssetAccess(app.AssetAccessDeps{Locks: w.locks, Comments: c})
		_, err := a.Inspect(t.Context(), shared.Actor{UserID: alice}, app.AssetRequest{Kind: app.AssetComment, ID: c.current.ID, Intent: app.AssetPrepare})
		if !errors.Is(err, domain.ErrCommentNotFound) {
			t.Fatalf("hidden leaked: %v", err)
		}
		noCommentResources(t, w)
	})
}

func TestAssetDraftAccess(t *testing.T) {
	for _, intent := range []app.AssetIntent{app.AssetRead, app.AssetPrepare, app.AssetRemove} {
		w := newDraftWorld()
		a := app.NewAssetAccess(app.AssetAccessDeps{Drafts: w.locks()})
		q := app.AssetRequest{Kind: app.AssetDraft, ID: w.s.row.ID, Intent: intent}
		actor := shared.Actor{UserID: alice}
		h, err := a.Inspect(t.Context(), actor, q)
		if err != nil || h.DraftID != w.s.row.ID || h.ProjectID != web {
			t.Fatalf("host %+v %v", h, err)
		}
		if err := a.Check(t.Context(), actor, q, h); err != nil {
			t.Fatal(err)
		}
		w.s.row.CreatedByID = bob
		if err := a.Check(t.Context(), actor, q, h); !errors.Is(err, domain.ErrDraftNotFound) {
			t.Fatalf("private draft visible: %v", err)
		}
		w.s.row.CreatedByID = alice
		w.s.row.ProjectID = nil
		if err := a.Check(t.Context(), actor, q, h); !errors.Is(err, app.ErrAssetScopeChanged) {
			t.Fatalf("changed scope=%v", err)
		}
		w.s.row.ProjectID = &web
		delete(w.p.rows, web)
		if err := a.Check(t.Context(), actor, q, h); !errors.Is(err, domain.ErrDraftNotFound) {
			t.Fatalf("deleted project=%v", err)
		}
	}
}
