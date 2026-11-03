package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func profileImageSnapshot(t *testing.T, w issueWorld) string {
	t.Helper()
	var data string
	if err := w.pool.QueryRow(t.Context(), `SELECT jsonb_build_object(
 'user',(SELECT to_jsonb(u) FROM users u WHERE id=$1),
 'assets',(SELECT jsonb_agg(to_jsonb(a) ORDER BY id) FROM file_assets a))::text`, w.ids["alice"]).Scan(&data); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestProfileImagesRejectInvalidPreparations(t *testing.T) {
	w := newIssueWorld(t)
	data := assetTestPNG(t)
	makeImage := func(actor, purpose string) uuid.UUID {
		return uploadedAssetHTTP(t, w, actor, purpose, `{"kind":"user"}`, "image.png", "image/png", data).Asset.ID
	}
	other := makeImage("bob", "USER_AVATAR")
	wrong := makeImage("alice", "USER_COVER")
	old := makeImage("alice", "USER_AVATAR")
	removed := makeImage("alice", "USER_AVATAR")
	assetAPICall(t, w, "alice", "DELETE", "/api/v0/assets/"+removed.String(), "", 204)
	pending := prepareAssetHTTP(t, w, "alice", "USER_AVATAR", `{"kind":"user"}`, "pending.png", "image/png", len(data)).Asset.ID
	if _, err := w.pool.Exec(t.Context(), "UPDATE file_assets SET created_at=$1 WHERE id=$2", time.Now().Add(-8*24*time.Hour), old); err != nil {
		t.Fatal(err)
	}
	missing := uuid.NewV7()
	baseline := w.send(t, "alice", "PATCH", "/api/v0/me", fmt.Sprintf(`{"avatar_asset_id":%q}`, missing), 404)
	for _, tc := range []struct {
		name, body, code string
		status           int
	}{
		{"other applicant", fmt.Sprintf(`{"avatar_asset_id":%q}`, other), "asset.not_found", 404},
		{"expired", fmt.Sprintf(`{"avatar_asset_id":%q}`, old), "asset.not_found", 404},
		{"deleted", fmt.Sprintf(`{"avatar_asset_id":%q}`, removed), "asset.not_found", 404},
		{"wrong purpose", fmt.Sprintf(`{"avatar_asset_id":%q}`, wrong), "validation_failed", 422},
		{"pending", fmt.Sprintf(`{"avatar_asset_id":%q}`, pending), "asset.not_uploaded", 409},
		{"hidden second beats pending first", fmt.Sprintf(`{"avatar_asset_id":%q,"cover_image_asset_id":%q}`, pending, other), "asset.not_found", 404},
		{"hidden first beats wrong purpose second", fmt.Sprintf(`{"avatar_asset_id":%q,"cover_image_asset_id":%q}`, other, pending), "asset.not_found", 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := profileImageSnapshot(t, w)
			got := w.send(t, "alice", "PATCH", "/api/v0/me", tc.body, tc.status)
			if problemCode(t, []byte(got)) != tc.code || (tc.status == 404 && got != baseline) {
				t.Fatalf("wrong refusal %s; hidden baseline %s", got, baseline)
			}
			if after := profileImageSnapshot(t, w); after != before {
				t.Fatal("refusal changed owner or assets")
			}
		})
	}
}

type profileSigningFault struct {
	assetStore
	fail atomic.Bool
}

func (s *profileSigningFault) SignRead(ctx context.Context, grant domain.ReadGrant) (domain.SignedRequest, error) {
	if s.fail.Load() {
		return domain.SignedRequest{}, errors.New("injected profile signing failure")
	}
	return s.assetStore.SignRead(ctx, grant)
}

func TestProfileImagesRollback(t *testing.T) {
	for _, stage := range []string{"owner", "second asset", "signer", "commit"} {
		t.Run(stage, func(t *testing.T) {
			var signing *profileSigningFault
			w, _ := newAssetCompleteWorld(t, func(c *assetComponents) { signing = &profileSigningFault{assetStore: c.store}; c.store = signing })
			data := assetTestPNG(t)
			makeImage := func(purpose string) uuid.UUID {
				return uploadedAssetHTTP(t, w, "alice", purpose, `{"kind":"user"}`, "image.png", "image/png", data).Asset.ID
			}
			oldAvatar, oldCover := makeImage("USER_AVATAR"), makeImage("USER_COVER")
			w.send(t, "alice", "PATCH", "/api/v0/me", fmt.Sprintf(`{"avatar_asset_id":%q,"cover_image_asset_id":%q}`, oldAvatar, oldCover), 200)
			nextAvatar, nextCover := makeImage("USER_AVATAR"), makeImage("USER_COVER")
			waitForCompletedKinds(t, w.pool, map[string]int{"asset.cleanup": 1, "cleanup.purge_deleted": 1}, &lockedBuffer{})
			before := profileImageSnapshot(t, w)
			table := "users"
			if stage == "signer" {
				signing.fail.Store(true)
			} else {
				trigger := "CREATE TRIGGER reject_profile_write BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION reject_profile_write()"
				if stage == "second asset" {
					table = "file_assets"
					trigger = fmt.Sprintf("CREATE TRIGGER reject_profile_write BEFORE UPDATE ON file_assets FOR EACH ROW WHEN (NEW.id='%s') EXECUTE FUNCTION reject_profile_write()", nextCover)
				}
				if stage == "commit" {
					trigger = "CREATE CONSTRAINT TRIGGER reject_profile_write AFTER UPDATE ON users DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_profile_write()"
				}
				if _, err := w.pool.Exec(t.Context(), `CREATE FUNCTION reject_profile_write() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'profile write rejected'; END $$;`+trigger); err != nil {
					t.Fatal(err)
				}
			}
			body := fmt.Sprintf(`{"avatar_asset_id":%q,"cover_image_asset_id":%q,"first_name":"must roll back"}`, nextAvatar, nextCover)
			got := w.send(t, "alice", "PATCH", "/api/v0/me", body, 500)
			if problemCode(t, []byte(got)) != "internal_error" || strings.Contains(got, "url") || strings.Contains(got, "payload") {
				t.Fatal("failure leaked partial capability", got)
			}
			if profileImageSnapshot(t, w) != before {
				t.Fatal("owner/old images/new candidates diverged after rollback")
			}
			if stage == "signer" {
				w.send(t, "alice", "GET", "/api/v0/me", "", 500)
				signing.fail.Store(false)
			} else if _, err := w.pool.Exec(t.Context(), "DROP TRIGGER reject_profile_write ON "+table); err != nil {
				t.Fatal(err)
			}
			got = w.send(t, "alice", "PATCH", "/api/v0/me", body, 200)
			checkImageReply(t, w, got, "avatar", nextAvatar, data)
			checkImageReply(t, w, got, "cover_image", nextCover, data)
		})
	}
}
