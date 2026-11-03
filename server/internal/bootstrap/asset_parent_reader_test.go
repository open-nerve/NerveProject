package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	assetapp "github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	assetdomain "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
)

type assetFactSpy struct {
	by    uuid.UUID
	at    time.Time
	err   error
	omit  bool
	calls []string
	ctx   context.Context
}

func (f *assetFactSpy) read(ctx context.Context, kind string, ids []uuid.UUID) error {
	f.ctx = ctx
	f.calls = append(f.calls, kind)
	if !slices.IsSortedFunc(ids, func(a, b uuid.UUID) int { return a.Compare(b) }) || len(ids) != 2 {
		return fmt.Errorf("unsorted or repeated %s parents: %v", kind, ids)
	}
	return f.err
}
func (f *assetFactSpy) AssetWorkspaceFacts(ctx context.Context, ids []uuid.UUID) ([]workspace.AssetParent, error) {
	if err := f.read(ctx, "W", ids); err != nil {
		return nil, err
	}
	if f.omit {
		return nil, nil
	}
	return []workspace.AssetParent{{ID: ids[0], DeletedAt: &f.at, DeletedByID: &f.by}, {ID: ids[1]}}, nil
}
func (f *assetFactSpy) AssetProjectFacts(ctx context.Context, ids []uuid.UUID) ([]project.AssetParent, error) {
	if err := f.read(ctx, "P", ids); err != nil {
		return nil, err
	}
	return []project.AssetParent{{ID: ids[0], DeletedAt: &f.at, DeletedByID: &f.by}, {ID: ids[1]}}, nil
}
func (f *assetFactSpy) AssetIssueFacts(ctx context.Context, ids []uuid.UUID) ([]issue.AssetParent, error) {
	if err := f.read(ctx, "I", ids); err != nil {
		return nil, err
	}
	return []issue.AssetParent{{ID: ids[0], DeletedAt: &f.at, DeletedByID: &f.by}, {ID: ids[1]}}, nil
}
func (f *assetFactSpy) AssetCommentFacts(ctx context.Context, ids []uuid.UUID) ([]issue.AssetParent, error) {
	if err := f.read(ctx, "C", ids); err != nil {
		return nil, err
	}
	return []issue.AssetParent{{ID: ids[0], DeletedAt: &f.at, DeletedByID: &f.by}, {ID: ids[1]}}, nil
}
func (f *assetFactSpy) AssetDraftFacts(ctx context.Context, ids []uuid.UUID) ([]issue.AssetParent, error) {
	if err := f.read(ctx, "D", ids); err != nil {
		return nil, err
	}
	return []issue.AssetParent{{ID: ids[0], DeletedAt: &f.at, DeletedByID: &f.by}, {ID: ids[1]}}, nil
}
func (f *assetFactSpy) ShareAssetWorkspaces(context.Context, []uuid.UUID) ([]uuid.UUID, error) {
	return nil, errors.New("fact reader took workspace locks")
}
func (f *assetFactSpy) LockAssetDrafts(context.Context, []uuid.UUID) ([]uuid.UUID, error) {
	return nil, errors.New("fact reader took draft locks")
}

func TestAssetParentDeletionReader(t *testing.T) {
	first, last := uuid.NewV7(), uuid.NewV7()
	host := func(id uuid.UUID) assetapp.Host {
		return assetapp.Host{WorkspaceID: id, ProjectID: id, IssueID: id, CommentID: id, DraftID: id}
	}
	hosts := []assetapp.Host{host(last), host(first), host(last)}
	spy := &assetFactSpy{by: uuid.New(), at: time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)}
	reader := assetParentReader{spy, spy, spy}
	got, err := reader.Deletions(t.Context(), hosts)
	if err != nil || len(got) != 5 || spy.ctx != t.Context() || !slices.Equal(spy.calls, []string{"W", "P", "I", "C", "D"}) {
		t.Fatalf("facts %+v %v calls %v", got, err, spy.calls)
	}
	for _, kind := range []assetdomain.ScopeKind{assetdomain.ScopeWorkspace, assetdomain.ScopeProject, assetdomain.ScopeIssue, assetdomain.ScopeComment, assetdomain.ScopeDraft} {
		fact, ok := got[assetapp.ParentKey{Kind: kind, ID: first}]
		if !ok || !fact.At.Equal(spy.at) || fact.By == nil || *fact.By != spy.by {
			t.Fatalf("original deletion lost for %s: %+v", kind, fact)
		}
		if _, ok := got[assetapp.ParentKey{Kind: kind, ID: last}]; ok {
			t.Fatal("live parent became a deletion")
		}
	}
	spy.omit = true
	if _, err := reader.Deletions(t.Context(), hosts); err == nil {
		t.Fatal("missing parent guessed as deleted or alive")
	}
	spy.omit = false
	spy.err = errors.New("query failed")
	if _, err := reader.Deletions(t.Context(), hosts); !errors.Is(err, spy.err) {
		t.Fatalf("read error swallowed: %v", err)
	}
}
