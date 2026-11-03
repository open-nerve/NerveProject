package app_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type profileImagePorts struct {
	*fakeAccount
	log     *callLog
	fail    string
	boom    error
	changes []app.ProfileImageChanges
	at      time.Time
}

func (p *profileImagePorts) step(ctx context.Context, name string) error {
	p.log.add(ctx, name)
	if p.fail == name {
		return p.boom
	}
	return nil
}
func (p *profileImagePorts) GetUser(ctx context.Context, id uuid.UUID) (domain.User, error) {
	if err := p.step(ctx, "profile"); err != nil {
		return domain.User{}, err
	}
	return p.fakeAccount.GetUser(ctx, id)
}
func (p *profileImagePorts) UpdateUser(ctx context.Context, id uuid.UUID, patch domain.UserPatch, at time.Time) (domain.User, error) {
	if err := p.step(ctx, "update"); err != nil {
		return domain.User{}, err
	}
	u, err := p.fakeAccount.UpdateUser(ctx, id, patch, at)
	if patch.SetAvatar {
		u.AvatarAssetID = patch.AvatarAssetID
	}
	if patch.SetCoverImage {
		u.CoverImageAssetID = patch.CoverImageAssetID
	}
	return u, err
}
func (p *profileImagePorts) LockImageBindings(ctx context.Context, q app.ProfileImageChanges) (time.Time, error) {
	p.changes = append(p.changes, q)
	return p.at, p.step(ctx, "images lock")
}
func (p *profileImagePorts) ApplyImageBindings(ctx context.Context, q app.ProfileImageChanges, at time.Time) error {
	p.changes = append(p.changes, q)
	if at != p.at {
		return errors.New("apply did not reuse final lock time")
	}
	return p.step(ctx, "images apply")
}
func (p *profileImagePorts) ReadProfileImages(ctx context.Context, refs []app.ProfileImageReference) (map[uuid.UUID]app.ImageURL, error) {
	if err := p.step(ctx, "images read"); err != nil {
		return nil, err
	}
	out := map[uuid.UUID]app.ImageURL{}
	for _, ref := range refs {
		for _, id := range []*uuid.UUID{ref.AvatarAssetID, ref.CoverImageAssetID} {
			if id != nil {
				out[*id] = app.ImageURL{URL: "signed " + id.String(), ExpiresAt: p.at.Add(time.Hour)}
			}
		}
	}
	return out, nil
}

type profileCommitError struct {
	tx  *fakeTx
	err error
}

func (c profileCommitError) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	if err := c.tx.WithinTx(ctx, fn); err != nil {
		return err
	}
	return c.err
}

func TestProfileImagesProtocol(t *testing.T) {
	f := newCredentialFixture()
	old, next, cover := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	p := &profileImagePorts{fakeAccount: aliceAccount(), log: f.log, at: now.Add(time.Minute)}
	p.user.AvatarAssetID, p.user.CoverImageAssetID = &old, &cover
	u := app.NewUpdateMe(app.UpdateMeDeps{Users: p, Bindings: p, Images: p, Lock: f.lock(), Tx: f.tx, Clock: clocktest.At(now)})
	got, err := u.Execute(shared.WithActor(t.Context(), sessionActor), domain.UserPatch{SetAvatar: true, AvatarAssetID: &next})
	want := []string{"lock " + userID.String(), "session " + sessionID.String(), "profile", "images lock", "update", "images apply", "images read"}
	change := app.ProfileImageChanges{UserID: userID, AvatarCurrent: &old, AvatarNext: &next, CoverCurrent: &cover, CoverNext: &cover}
	if err != nil || !slices.Equal(f.log.calls, want) || !reflect.DeepEqual(p.changes, []app.ProfileImageChanges{change, change}) {
		t.Fatalf("protocol: calls=%v changes=%+v err=%v", f.log.calls, p.changes, err)
	}
	if len(p.userUpdates) != 1 || p.userUpdates[0].now != p.at || got.Avatar == nil || got.Avatar.URL != "signed "+next.String() || got.CoverImage == nil || got.CoverImage.URL != "signed "+cover.String() {
		t.Fatalf("wrong time/result %+v %+v", p.userUpdates, got)
	}
}

func TestProfileImagesPortErrors(t *testing.T) {
	for _, operation := range []string{"get", "update"} {
		for _, stage := range []string{"credential", "profile", "images lock", "update", "images apply", "images read", "commit"} {
			if operation == "get" && (stage == "images lock" || stage == "update" || stage == "images apply") {
				continue
			}
			t.Run(operation+"/"+stage, func(t *testing.T) {
				f := newCredentialFixture()
				boom := errors.New("original profile port failure")
				p := &profileImagePorts{fakeAccount: aliceAccount(), log: f.log, fail: stage, boom: boom, at: now}
				if stage == "credential" {
					f.creds.accountErr = boom
				}
				var tx shared.TxManager = f.tx
				if stage == "commit" {
					tx = profileCommitError{f.tx, boom}
				}
				ctx := shared.WithActor(t.Context(), tokenActor)
				var got domain.User
				var err error
				if operation == "get" {
					got, err = app.NewGetMe(app.GetMeDeps{Users: p, Images: p, Lock: f.lock(), Tx: tx, Clock: clocktest.At(now)}).Execute(ctx)
				} else {
					got, err = app.NewUpdateMe(app.UpdateMeDeps{Users: p, Bindings: p, Images: p, Lock: f.lock(), Tx: tx, Clock: clocktest.At(now)}).Execute(ctx, domain.UserPatch{})
				}
				if err != boom || got != (domain.User{}) {
					t.Fatalf("partial result or changed cause: %+v %v", got, err)
				}
			})
		}
	}
}
