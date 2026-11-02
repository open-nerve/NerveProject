package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	assetapp "github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	assetdomain "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	issuedomain "github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// These are the actual module facades and actual access rules on API-created
// hosts. No asset HTTP API or binding is claimed by this host-only matrix.
func assetOwners(w issueWorld) assetHostAccess {
	wp, pp := workspace.Provide(w.pool), project.Provide(w.pool)
	auth := access.New(access.Deps{WorkspaceRoles: wp.WorkspaceRoles, ProjectAccess: accessProjects{projects: pp.ProjectAccess}})
	ws := workspace.New(workspace.Deps{Pool: w.pool, Authorizer: auth})
	pr := project.New(project.Deps{Pool: w.pool, Authorizer: auth})
	is := issue.New(issue.Deps{Pool: w.pool, Authorizer: auth, Projects: issueProjects{projects: pp.IssueProjects}, Workspaces: issueWorkspaces{directory: wp.WorkspaceDirectory}})
	return assetHostAccess{ws.AssetAccess(), pr.AssetAccess(), is.AssetAccess()}
}

func checkAssetOwner(t *testing.T, w issueWorld, a assetHostAccess, actor shared.Actor, q assetapp.HostRequest, h assetapp.Host) error {
	t.Helper()
	return postgres.NewTxManager(w.pool, 2*time.Second).WithinTx(t.Context(), func(ctx context.Context) error {
		if h.WorkspaceID != uuid.Nil() {
			if _, err := (assetWorkspaceLocks{workspace.Provide(w.pool).AssetParents}).ShareMany(ctx, []uuid.UUID{h.WorkspaceID}); err != nil {
				return err
			}
		}
		if h.ProjectID != uuid.Nil() {
			if _, err := (assetProjectLocks{project.Provide(w.pool).IssueProjects}).LockMany(ctx, []uuid.UUID{h.ProjectID}); err != nil {
				return err
			}
		}
		if h.DraftID != uuid.Nil() {
			if _, err := (assetDraftLocks{issue.Provide(w.pool).AssetParents}).LockMany(ctx, []uuid.UUID{h.DraftID}); err != nil {
				return err
			}
		}
		return a.Check(ctx, actor, q, h)
	})
}

func TestAssetHostAccessRealMatrix(t *testing.T) {
	w := newIssueWorld(t)
	id := w.newWorkItem(t)
	comment := w.newComment(t, "bob", id)
	draft := createDraftHTTP(t, w, "alice", `{"project_id":"`+w.web.String()+`","payload":{"name":"Draft"}}`)
	w.makeMember(t, "walt", w.web, shared.RoleGuest)
	a := assetOwners(w)
	for _, row := range []struct {
		purpose assetdomain.Purpose
		kind    assetdomain.ScopeKind
		id      uuid.UUID
	}{
		{assetdomain.UserAvatar, assetdomain.ScopeUser, uuid.Nil()}, {assetdomain.UserCover, assetdomain.ScopeUser, uuid.Nil()},
		{assetdomain.WorkspaceLogo, assetdomain.ScopeWorkspace, w.acme}, {assetdomain.ProjectCover, assetdomain.ScopeProject, w.web}, {assetdomain.ProjectCover, assetdomain.ScopeNewProject, w.acme},
		{assetdomain.IssueAttachment, assetdomain.ScopeIssue, id}, {assetdomain.IssueDescription, assetdomain.ScopeIssue, id},
		{assetdomain.CommentDescription, assetdomain.ScopeComment, comment.ID}, {assetdomain.CommentAttachment, assetdomain.ScopeComment, comment.ID}, {assetdomain.CommentDescription, assetdomain.ScopeNewComment, id}, {assetdomain.CommentAttachment, assetdomain.ScopeNewComment, id},
		{assetdomain.DraftIssueDescription, assetdomain.ScopeDraft, draft.ID}, {assetdomain.DraftIssueAttachment, assetdomain.ScopeDraft, draft.ID},
	} {
		for _, intent := range []assetapp.AccessIntent{assetapp.ReadAsset, assetapp.PrepareAsset, assetapp.RemoveAsset} {
			for _, who := range []string{"alice", "bob", "walt", "olga"} {
				t.Run(fmt.Sprintf("%s/%s/%d/%s", row.purpose, row.kind, intent, who), func(t *testing.T) {
					q := assetapp.HostRequest{Scope: assetdomain.Scope{Kind: row.kind, ID: row.id}, Intent: intent}
					if !q.Scope.ValidFor(row.purpose) {
						t.Fatal("invalid purpose fixture")
					}
					var want error
					switch row.kind {
					case assetdomain.ScopeUser:
					case assetdomain.ScopeWorkspace:
						if who == "olga" {
							want = assetdomain.ErrAssetNotFound
						} else if intent != assetapp.ReadAsset && who != "alice" {
							want = shared.Forbidden()
						}
					case assetdomain.ScopeProject:
						if who == "olga" {
							want = assetdomain.ErrAssetNotFound
						} else if intent != assetapp.ReadAsset && who != "alice" {
							want = shared.Forbidden()
						}
					case assetdomain.ScopeNewProject:
						if who == "olga" {
							want = assetdomain.ErrAssetNotFound
						}
					case assetdomain.ScopeDraft:
						if who != "alice" {
							want = assetdomain.ErrAssetNotFound
						}
					default:
						if who == "walt" || who == "olga" {
							want = assetdomain.ErrAssetNotFound
						} else if row.kind == assetdomain.ScopeComment && who == "alice" && intent == assetapp.PrepareAsset {
							want = issuedomain.ErrNotAuthor
						}
					}
					actor := shared.Actor{UserID: w.ids[who]}
					h, err := a.Inspect(t.Context(), actor, q)
					if !errors.Is(err, want) {
						t.Fatalf("Inspect=%v want %v", err, want)
					}
					if err == nil {
						if err := checkAssetOwner(t, w, a, actor, q, h); err != nil {
							t.Fatal(err)
						}
					}
				})
			}
		}
	}
	t.Run("archive refusal follows authority and drafts remain private", func(t *testing.T) {
		q := assetapp.HostRequest{Scope: assetdomain.Scope{Kind: assetdomain.ScopeIssue, ID: id}, Intent: assetapp.PrepareAsset}
		actor := shared.Actor{UserID: w.ids["alice"]}
		h, err := a.Inspect(t.Context(), actor, q)
		if err != nil {
			t.Fatal(err)
		}
		w.send(t, "alice", "POST", "/api/v0/projects/"+w.web.String()+"/archive", `{}`, 200)
		if err := checkAssetOwner(t, w, a, actor, q, h); !errors.Is(err, issuedomain.ErrProjectArchived) {
			t.Fatalf("project archive=%v", err)
		}
		q = assetapp.HostRequest{Scope: assetdomain.Scope{Kind: assetdomain.ScopeDraft, ID: draft.ID}, Intent: assetapp.PrepareAsset}
		h, err = a.Inspect(t.Context(), actor, q)
		if err != nil {
			t.Fatal(err)
		}
		if err := checkAssetOwner(t, w, a, actor, q, h); err != nil {
			t.Fatalf("archived draft denied: %v", err)
		}
	})
	t.Run("asset gate rejects an existing transaction", func(t *testing.T) {
		g := forAssetWrites(newWriteGate(config.WriteGateConfig{PerProject: 1, MaxQueue: 1, MaxWait: time.Second}))
		if err := postgres.NewTxManager(w.pool, time.Second).WithinTx(pgtest.Soon(t), func(ctx context.Context) error {
			leave, err := g.Enter(ctx, w.web)
			if err == nil {
				leave()
				return errors.New("asset joined an existing transaction")
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	})
}
