package app_test

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

type invitationImageRows struct {
	*fakeInvitations
	shareErr, readErr error
	otherWorkspace    bool
}

func (r invitationImageRows) ShareWorkspace(ctx context.Context, id uuid.UUID) error {
	if r.shareErr != nil {
		return r.shareErr
	}
	return r.fakeInvitations.ShareWorkspace(ctx, id)
}
func (r invitationImageRows) ShareInvitationPreview(ctx context.Context, id uuid.UUID) (domain.InvitationPreview, error) {
	if r.readErr != nil {
		return domain.InvitationPreview{}, r.readErr
	}
	got, err := r.fakeInvitations.ShareInvitationPreview(ctx, id)
	if r.otherWorkspace {
		got.WorkspaceID = beta.ID
	}
	return got, err
}

func TestInvitationImageReadErrors(t *testing.T) {
	for _, stage := range []string{"deleted workspace", "share failed", "revoked invitation", "read failed", "changed workspace", "signer", "commit"} {
		t.Run(stage, func(t *testing.T) {
			f := newInvitations()
			failure := errors.New("preview dependency failed")
			rows := invitationImageRows{fakeInvitations: f.invitations}
			images := &workspaceImageReader{}
			want := failure
			switch stage {
			case "deleted workspace":
				rows.shareErr, want = app.ErrNotFound, domain.ErrInvitationNotFound
			case "share failed":
				rows.shareErr = failure
			case "revoked invitation":
				rows.readErr, want = app.ErrNotFound, domain.ErrInvitationNotFound
			case "read failed":
				rows.readErr = failure
			case "changed workspace":
				rows.otherWorkspace, want = true, domain.ErrInvitationNotFound
			case "signer":
				images.err = failure
			case "commit":
				f.tx.commitErr = failure
			}
			uc := app.NewGetWorkspaceInvitation(app.GetInvitationDeps{Invitations: rows, Images: images, Tx: f.tx, MAC: f.mac})
			got, err := uc.Execute(context.Background(), carolToAcme.ID, tokenOf(f.mac, carolToAcme.ID))
			if !errors.Is(err, want) || got != (domain.InvitationPreview{}) || !f.tx.answered(err) {
				t.Fatal("partial or misclassified preview", got, err, want)
			}
		})
	}
}

type memberImageReader struct{ err error }

func (r memberImageReader) ReadMemberAvatars(ctx context.Context, _ []app.MemberAvatarReference) (map[uuid.UUID]app.ImageURL, error) {
	if ctx.Value(inTxKey{}) != true {
		return nil, errors.New("member avatars escaped transaction")
	}
	return nil, r.err
}

func TestMemberAvatarReadErrors(t *testing.T) {
	for _, stage := range []string{"caller", "deleted workspace", "share failed", "signer", "commit"} {
		t.Run(stage, func(t *testing.T) {
			f := newMembers()
			failure := errors.New("avatar dependency failed")
			rows := &workspaceImageRows{fakeWorkspaces: f.workspaces}
			caller := &fakeCallerLock{log: f.log}
			images, tx := memberImageReader{}, &fakeTx{}
			want := failure
			switch stage {
			case "caller":
				caller.err = failure
			case "deleted workspace":
				rows.locked, want = []uuid.UUID{}, domain.ErrNotFound
			case "share failed":
				rows.shareErr = failure
			case "signer":
				images.err = failure
			case "commit":
				tx.commitErr = failure
			}
			uc := app.NewListWorkspaceMembers(app.ListMembersDeps{Members: rows, Profiles: f.profiles, Auth: f.auth, Caller: caller, Images: images, Tx: tx, Clock: clockAt{at: now}})
			got, err := uc.Execute(as(alice), "acme")
			if !errors.Is(err, want) || got != nil || !tx.answered(err) {
				t.Fatal("partial or misclassified member collection", got, err, want)
			}
		})
	}
}
