package app_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type workspaceImageRows struct {
	*fakeWorkspaces
	onShare             func()
	ids, locked         []uuid.UUID
	shareErr            error
	listCalls, listFail int
	listErr             error
}

func (f *workspaceImageRows) ShareImageWorkspaces(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	f.log.add(ctx, "ShareImageWorkspaces %v", ids)
	f.ids = slices.Clone(ids)
	if f.onShare != nil {
		f.onShare()
	}
	if f.shareErr != nil {
		return nil, f.shareErr
	}
	if f.locked != nil {
		return f.locked, nil
	}
	return ids, nil
}
func (f *workspaceImageRows) ListWorkspaces(ctx context.Context, id uuid.UUID) ([]domain.Workspace, error) {
	f.listCalls++
	if f.listCalls == f.listFail {
		return nil, f.listErr
	}
	return f.fakeWorkspaces.ListWorkspaces(ctx, id)
}

type workspaceImageReader struct {
	refs []app.WorkspaceImageReference
	err  error
}

func (r *workspaceImageReader) ReadWorkspaceImages(ctx context.Context, refs []app.WorkspaceImageReference) (map[uuid.UUID]app.ImageURL, error) {
	if ctx.Value(inTxKey{}) != true {
		return nil, errors.New("image read escaped transaction")
	}
	r.refs = refs
	return nil, r.err
}

// A second listing alone may include newly joined owners we did not lock, or
// retain revoked candidates. Reverse ID/name ordering catches display reordering.
func TestWorkspaceImagesCandidates(t *testing.T) {
	low := domain.Workspace{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), Name: "Z"}
	high := domain.Workspace{ID: uuid.MustParse("00000000-0000-4000-8000-000000000003"), Name: "B"}
	added := domain.Workspace{ID: uuid.MustParse("00000000-0000-4000-8000-000000000002"), Name: "A"}
	for _, tc := range []struct {
		name   string
		next   []domain.Workspace
		want   []domain.Workspace
		locked []uuid.UUID
	}{
		{"order", []domain.Workspace{high, low}, []domain.Workspace{high, low}, nil},
		{"revoked", []domain.Workspace{high}, []domain.Workspace{high}, nil},
		{"new membership", []domain.Workspace{added, high, low}, []domain.Workspace{high, low}, nil},
		{"deleted while waiting", []domain.Workspace{high, low}, []domain.Workspace{high}, []uuid.UUID{high.ID}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &workspaceImageRows{fakeWorkspaces: &fakeWorkspaces{log: &callLog{}, lists: map[uuid.UUID][]domain.Workspace{alice.ID: {high, low}}}, locked: tc.locked}
			f.onShare = func() { f.lists[alice.ID] = tc.next }
			images := &workspaceImageReader{}
			uc := app.NewListWorkspaces(app.ListWorkspacesDeps{Workspaces: f, Caller: &fakeCallerLock{log: f.log}, Images: images, Tx: &fakeTx{}, Clock: clockAt{at: clockNow}})
			got, err := uc.Execute(as(alice))
			if err != nil || !slices.Equal(got, tc.want) || !slices.Equal(f.ids, []uuid.UUID{low.ID, high.ID}) {
				t.Fatal("candidate boundary/order", got, f.ids, err)
			}
			refs := make([]app.WorkspaceImageReference, len(tc.want))
			for i, w := range tc.want {
				refs[i] = app.WorkspaceImageReference{WorkspaceID: w.ID}
			}
			if !reflect.DeepEqual(images.refs, refs) {
				t.Fatal("signed outside visible locked set", images.refs)
			}
			if f.log.calls[0] != "ListWorkspaces "+alice.ID.String()+" outside tx" || f.log.calls[1] != callerImageCall(alice) {
				t.Fatal("account not first lock", f.log.calls)
			}
		})
	}
}

func TestWorkspaceImagesReadErrors(t *testing.T) {
	failure := errors.New("image dependency failed")
	for _, stage := range []string{"prelist", "caller", "share", "reread", "signer", "commit"} {
		t.Run(stage, func(t *testing.T) {
			f := &workspaceImageRows{fakeWorkspaces: &fakeWorkspaces{log: &callLog{}, lists: map[uuid.UUID][]domain.Workspace{alice.ID: {acme}}}}
			caller := &fakeCallerLock{log: f.log}
			images := &workspaceImageReader{}
			tx := &fakeTx{}
			switch stage {
			case "prelist":
				f.listFail = 1
				f.listErr = failure
			case "caller":
				caller.err = failure
			case "share":
				f.shareErr = failure
			case "reread":
				f.listFail = 2
				f.listErr = failure
			case "signer":
				images.err = failure
			case "commit":
				tx.commitErr = failure
			}
			uc := app.NewListWorkspaces(app.ListWorkspacesDeps{Workspaces: f, Caller: caller, Images: images, Tx: tx, Clock: clockAt{at: clockNow}})
			got, err := uc.Execute(as(alice))
			if err != failure || got != nil {
				t.Fatal("error/partial collection", got, err)
			}
		})
	}
}

func TestWorkspaceImagesGetRechecksCandidate(t *testing.T) {
	for _, stage := range []string{"revoked", "deleted", "slug reused", "share failed", "read failed", "signer failed", "commit failed", "caller failed"} {
		t.Run(stage, func(t *testing.T) {
			failure := errors.New("failed read dependency")
			f := &workspaceImageRows{fakeWorkspaces: &fakeWorkspaces{log: &callLog{}, workspaces: []domain.Workspace{acme}}}
			auth := &fakeAuthorizer{log: f.log, grants: map[grantKey]shared.Grant{{alice.ID, acme.ID}: {WorkspaceRole: shared.RoleAdmin}}}
			caller := &fakeCallerLock{log: f.log}
			images := &workspaceImageReader{}
			tx := &fakeTx{}
			want := error(domain.ErrNotFound)
			switch stage {
			case "revoked":
				f.onShare = func() { delete(auth.grants, grantKey{alice.ID, acme.ID}) }
			case "deleted":
				f.locked = []uuid.UUID{}
			case "slug reused":
				f.onShare = func() {
					f.workspaces[0].ID = beta.ID
					auth.grants[grantKey{alice.ID, beta.ID}] = shared.Grant{WorkspaceRole: shared.RoleAdmin}
				}
			case "share failed":
				f.shareErr = failure
				want = failure
			case "read failed":
				f.onShare = func() { f.slugErrs = map[string]error{"acme": failure} }
				want = failure
			case "signer failed":
				images.err = failure
				want = failure
			case "commit failed":
				tx.commitErr = failure
				want = failure
			case "caller failed":
				caller.err = failure
				want = failure
			}
			uc := app.NewGetWorkspace(app.GetWorkspaceDeps{Workspaces: f, Auth: auth, Caller: caller, Images: images, Tx: tx, Clock: clockAt{at: clockNow}})
			got, err := uc.Execute(as(alice), "acme")
			if !errors.Is(err, want) || got != (domain.Workspace{}) {
				t.Fatal("stale or partial single reply", got, err, want)
			}
		})
	}
}
