package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	assetapp "github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	assetdomain "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	issuedomain "github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	projectdomain "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type assetHostSpy struct {
	actor                   shared.Actor
	wid, pid, iid, did, cid uuid.UUID
	request                 any
	check                   any
	err                     error
	ctx                     context.Context
}
type assetWorkspaceSpy struct{ *assetHostSpy }
type assetProjectSpy struct{ *assetHostSpy }
type assetIssueSpy struct{ *assetHostSpy }

func TestAssetProjectLocksPreserveFailure(t *testing.T) {
	lost := errors.New("project lock unavailable")
	f := &fakeIssueProjects{web: uuid.New(), err: lost}
	ids, err := (assetProjectLocks{f}).LockMany(t.Context(), []uuid.UUID{f.web})
	if ids != nil || err != lost || len(f.asked) != 1 {
		t.Fatalf("failed project lock: ids=%v err=%v calls=%v", ids, err, f.asked)
	}
}

func (s assetWorkspaceSpy) Inspect(ctx context.Context, a shared.Actor, q workspace.AssetRequest) (workspace.AssetHost, error) {
	s.ctx, s.actor, s.request = ctx, a, q
	return workspace.AssetHost{WorkspaceID: s.wid}, s.err
}
func (s assetWorkspaceSpy) Check(ctx context.Context, a shared.Actor, q workspace.AssetRequest, h workspace.AssetHost) error {
	s.ctx, s.actor, s.request, s.check = ctx, a, q, h
	return s.err
}
func (s assetProjectSpy) Inspect(ctx context.Context, a shared.Actor, q project.AssetRequest) (project.AssetHost, error) {
	s.ctx, s.actor, s.request = ctx, a, q
	return project.AssetHost{WorkspaceID: s.wid, ProjectID: s.pid}, s.err
}
func (s assetProjectSpy) Check(ctx context.Context, a shared.Actor, q project.AssetRequest, h project.AssetHost) error {
	s.ctx, s.actor, s.request, s.check = ctx, a, q, h
	return s.err
}
func (s assetIssueSpy) Inspect(ctx context.Context, a shared.Actor, q issue.AssetRequest) (issue.AssetHost, error) {
	s.ctx, s.actor, s.request = ctx, a, q
	return issue.AssetHost{WorkspaceID: s.wid, ProjectID: s.pid, IssueID: s.iid, CommentID: s.cid, DraftID: s.did}, s.err
}
func (s assetIssueSpy) Check(ctx context.Context, a shared.Actor, q issue.AssetRequest, h issue.AssetHost) error {
	s.ctx, s.actor, s.request, s.check = ctx, a, q, h
	return s.err
}

func TestAssetHostRouting(t *testing.T) {
	actor := shared.Actor{UserID: uuid.New(), SessionID: uuid.New()}
	id, wid, pid, iid, cid, did := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	for _, kind := range []assetdomain.ScopeKind{assetdomain.ScopeWorkspace, assetdomain.ScopeProject, assetdomain.ScopeNewProject, assetdomain.ScopeIssue, assetdomain.ScopeComment, assetdomain.ScopeNewComment, assetdomain.ScopeDraft} {
		for _, intent := range []assetapp.AccessIntent{assetapp.ReadAsset, assetapp.PrepareAsset, assetapp.RemoveAsset} {
			spy := &assetHostSpy{wid: wid, pid: pid, iid: iid, cid: cid, did: did}
			a := assetHostAccess{assetWorkspaceSpy{spy}, assetProjectSpy{spy}, assetIssueSpy{spy}}
			q := assetapp.HostRequest{Scope: assetdomain.Scope{Kind: kind, ID: id}, Intent: intent}
			h, err := a.Inspect(t.Context(), actor, q)
			if err != nil || h.Scope != q.Scope || h.WorkspaceID != wid || spy.actor != actor || spy.ctx != t.Context() {
				t.Fatalf("routing %s %d: %+v %v", kind, intent, h, err)
			}
			var want any
			switch kind {
			case assetdomain.ScopeWorkspace:
				want = workspace.AssetRequest{ID: id, Intent: workspace.AssetIntent(intent)}
			case assetdomain.ScopeProject, assetdomain.ScopeNewProject:
				want = project.AssetRequest{ID: id, NewProject: kind == assetdomain.ScopeNewProject, Intent: project.AssetIntent(intent)}
				if h.ProjectID != pid {
					t.Fatal("project lost")
				}
			default:
				k := map[assetdomain.ScopeKind]issue.AssetHostKind{assetdomain.ScopeIssue: issue.AssetIssue, assetdomain.ScopeComment: issue.AssetComment, assetdomain.ScopeNewComment: issue.AssetNewComment, assetdomain.ScopeDraft: issue.AssetDraft}[kind]
				want = issue.AssetRequest{ID: id, Kind: k, Intent: issue.AssetIntent(intent)}
				if h.ProjectID != pid || h.IssueID != iid || h.CommentID != cid || h.DraftID != did {
					t.Fatal("issue fields lost")
				}
			}
			if !reflect.DeepEqual(spy.request, want) {
				t.Fatalf("request=%+v want=%+v", spy.request, want)
			}
			if err := a.Check(t.Context(), actor, q, h); err != nil || spy.check == nil {
				t.Fatalf("Check bypassed %v", err)
			}
			switch check := spy.check.(type) {
			case workspace.AssetHost:
				if check.WorkspaceID != wid {
					t.Fatal("W mismatch")
				}
			case project.AssetHost:
				if check.WorkspaceID != wid || check.ProjectID != pid {
					t.Fatal("P mismatch")
				}
			case issue.AssetHost:
				if check.WorkspaceID != wid || check.ProjectID != pid || check.IssueID != iid || check.CommentID != cid || check.DraftID != did {
					t.Fatal("I mismatch")
				}
			}
		}
	}
	for _, source := range []error{workspacedomain.ErrNotFound, projectdomain.ErrNotFound, projectdomain.ErrWorkspaceNotFound, issuedomain.ErrIssueNotFound, issuedomain.ErrCommentNotFound, issuedomain.ErrDraftNotFound, errors.New("connection lost"), shared.Forbidden()} {
		spy := &assetHostSpy{err: source}
		a := assetHostAccess{assetWorkspaceSpy{spy}, assetProjectSpy{spy}, assetIssueSpy{spy}}
		q := assetapp.HostRequest{Scope: assetdomain.Scope{Kind: assetdomain.ScopeIssue, ID: id}, Intent: assetapp.ReadAsset}
		_, got := a.Inspect(t.Context(), actor, q)
		want := source
		if workspace.AssetHostNotFound(source) || project.AssetHostNotFound(source) || issue.AssetHostNotFound(source) {
			want = assetdomain.ErrAssetNotFound
		}
		if !errors.Is(got, want) {
			t.Fatalf("error conversion %v => %v want %v", source, got, want)
		}
	}
	a := assetHostAccess{}
	q := assetapp.HostRequest{Scope: assetdomain.Scope{Kind: assetdomain.ScopeUser}, Intent: assetapp.ReadAsset}
	if h, err := a.Inspect(t.Context(), actor, q); err != nil || h.Scope != q.Scope {
		t.Fatalf("current user host: %+v %v", h, err)
	}
	q.Scope.ID = id
	if _, err := a.Inspect(t.Context(), actor, q); err == nil {
		t.Fatal("client selected another user")
	}
}

func TestAssetAndIssueShareWriteGate(t *testing.T) {
	gate := newWriteGate(config.WriteGateConfig{PerProject: 1, MaxQueue: 0, MaxWait: time.Second})
	files := forAssetWrites(gate)
	id := uuid.New()
	leave, err := gate.Enter(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	defer leave()
	if release, err := files.Enter(t.Context(), id); err == nil {
		release()
		t.Fatal("asset uses an independent gate")
	}
	release, err := files.Enter(t.Context(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	release()
}
