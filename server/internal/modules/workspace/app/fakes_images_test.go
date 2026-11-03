package app_test

import (
	"context"
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Legacy no-image cases still use complete dependencies and reject unexpected
// image references. They never silently exercise a fake image success path.
type noWorkspaceImages struct{ clock app.Clock }

func (noWorkspaceImages) check(q app.LogoImageChange) error {
	if q.Current != nil || q.Next != nil {
		return fmt.Errorf("unexpected logo in no-image fixture")
	}
	return nil
}
func (n noWorkspaceImages) LockImageBindings(_ context.Context, q app.LogoImageChange) (time.Time, error) {
	if err := n.check(q); err != nil {
		return time.Time{}, err
	}
	return n.clock.Now(), nil
}
func (n noWorkspaceImages) ApplyImageBindings(_ context.Context, q app.LogoImageChange, _ time.Time) error {
	return n.check(q)
}
func (noWorkspaceImages) ReadWorkspaceImages(_ context.Context, refs []app.WorkspaceImageReference) (map[uuid.UUID]app.ImageURL, error) {
	for _, ref := range refs {
		if ref.LogoAssetID != nil {
			return nil, fmt.Errorf("unexpected logo in no-image fixture")
		}
	}
	return nil, nil
}
func (f *fakeWorkspaces) ShareImageWorkspaces(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	f.log.add(ctx, "ShareImageWorkspaces %v", ids)
	return slices.Clone(ids), nil
}
func getWorkspace(store *fakeWorkspaces, auth shared.Authorizer) *app.GetWorkspace {
	return app.NewGetWorkspace(app.GetWorkspaceDeps{Workspaces: store, Auth: auth, Caller: &fakeCallerLock{log: store.log}, Images: noWorkspaceImages{}, Tx: &fakeTx{}, Clock: clockAt{at: clockNow}})
}
func listWorkspaces(store *fakeWorkspaces) *app.ListWorkspaces {
	return app.NewListWorkspaces(app.ListWorkspacesDeps{Workspaces: store, Caller: &fakeCallerLock{log: store.log}, Images: noWorkspaceImages{}, Tx: &fakeTx{}, Clock: clockAt{at: clockNow}})
}
func updateWorkspace(store *fakeWorkspaces, auth shared.Authorizer, tx shared.TxManager, clock app.Clock) *app.UpdateWorkspace {
	return app.NewUpdateWorkspace(app.UpdateWorkspaceDeps{Workspaces: store, Auth: auth, Caller: &fakeCallerLock{log: store.log}, Images: noWorkspaceImages{}, Bindings: noWorkspaceImages{clock: clock}, Tx: tx, Clock: clock})
}
func callerImageCall(user app.AccountState) string {
	return fmt.Sprintf("LockCaller %s session %s token %s at %s", user.ID, uuid.Nil(), uuid.Nil(), clockNow.Format(time.RFC3339Nano))
}
