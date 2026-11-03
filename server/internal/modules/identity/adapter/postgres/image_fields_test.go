package postgresadapter_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// Catches dropped fields in SQL/row conversion, NULL interpreted as omission,
// or an update that bypasses the caller's transaction. Assets are FK fixtures;
// authorization and actual binding are tested through the API in later tasks.
func TestProfileImageFields(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	u := newUser("profile-fields@example.com")
	mustCreate(t, s, u)
	avatar, cover := uuid.NewV7(), uuid.NewV7()
	for _, id := range []uuid.UUID{avatar, cover} {
		_, err := pool.Exec(ctx, `INSERT INTO file_assets(id,user_id,entity_type,scope_kind,size,attributes,storage_metadata)
 VALUES($1,$2,'USER_AVATAR','user',1,'{"name":"image.png","content_type":"image/png"}',
 '{"schema_version":1,"origin":"copy","staging":null,"final":null,"candidates":[]}')`, id, u.ID)
		if err != nil {
			t.Fatal(err)
		}
	}
	assertImages := func(t *testing.T, wantA, wantC *uuid.UUID) {
		t.Helper()
		got, err := s.GetUser(ctx, u.ID)
		if err != nil || !reflect.DeepEqual(got.AvatarAssetID, wantA) || !reflect.DeepEqual(got.CoverImageAssetID, wantC) {
			t.Fatalf("GetUser images=%v/%v err=%v want=%v/%v", got.AvatarAssetID, got.CoverImageAssetID, err, wantA, wantC)
		}
		var a, c *uuid.UUID
		if err := pool.QueryRow(ctx, "SELECT avatar_asset_id,cover_image_asset_id FROM users WHERE id=$1", u.ID).Scan(&a, &c); err != nil || !reflect.DeepEqual(a, wantA) || !reflect.DeepEqual(c, wantC) {
			t.Fatalf("database images=%v/%v err=%v", a, c, err)
		}
	}
	steps := []struct {
		name  string
		patch domain.UserPatch
		a, c  *uuid.UUID
	}{
		{"set both", domain.UserPatch{SetAvatar: true, AvatarAssetID: &avatar, SetCoverImage: true, CoverImageAssetID: &cover}, &avatar, &cover},
		{"omit both", domain.UserPatch{}, &avatar, &cover},
		{"clear avatar", domain.UserPatch{SetAvatar: true}, nil, &cover},
		{"clear cover", domain.UserPatch{SetCoverImage: true}, nil, nil},
	}
	for _, tc := range steps {
		t.Run(tc.name, func(t *testing.T) {
			got, err := s.UpdateUser(ctx, u.ID, tc.patch, now.Add(time.Minute))
			if err != nil || !reflect.DeepEqual(got.AvatarAssetID, tc.a) || !reflect.DeepEqual(got.CoverImageAssetID, tc.c) {
				t.Fatalf("UpdateUser=%+v %v", got, err)
			}
			assertImages(t, tc.a, tc.c)
		})
	}
	boom := errors.New("rollback image row")
	err := postgres.NewTxManager(pool, time.Second).WithinTx(ctx, func(txctx context.Context) error {
		if _, err := s.UpdateUser(txctx, u.ID, domain.UserPatch{SetAvatar: true, AvatarAssetID: &avatar}, now); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatal(err)
	}
	assertImages(t, nil, nil)
}
