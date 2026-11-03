package bootstrap

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
)

// The same business requests run against both real stores. Only the transfer
// endpoint and its documented successful PUT status differ between drivers.
func TestImageStorageContract(t *testing.T) {
	for _, driver := range []string{"local", "s3"} {
		t.Run(driver, func(t *testing.T) {
			w, origin := newImageStorageWorld(t, driver)
			data := assetTestPNG(t)
			upload := func(purpose, scope string) uuid.UUID {
				t.Helper()
				file := prepareAssetHTTP(t, w, "alice", purpose, scope, "image.png", "image/png", len(data))
				base, status := w.base, 204
				if driver == "s3" {
					base, status = "", 200
				}
				assetRequest(t, base, file.uploadRequest(), string(data), status)
				assetAPICall(t, w, "alice", "POST", completeAssetPath(file.Asset.ID), `{}`, 200)
				return file.Asset.ID
			}
			for _, tc := range []struct{ name, purpose, scope, path, field, table string }{
				{"avatar", "USER_AVATAR", `{"kind":"user"}`, "/api/v0/me", "avatar", "users"},
				{"cover", "USER_COVER", `{"kind":"user"}`, "/api/v0/me", "cover_image", "users"},
				{"logo", "WORKSPACE_LOGO", assetScopeBody("workspace", w.acme), "/api/v0/workspaces/acme", "logo", "workspaces"},
				{"project", "PROJECT_COVER", assetScopeBody("new_project", w.acme), "/api/v0/workspaces/acme/projects", "cover_image", "projects"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					old := upload(tc.purpose, tc.scope)
					path, method, want := tc.path, "PATCH", 200
					body := fmt.Sprintf(`{%q:%q}`, tc.field+"_asset_id", old)
					if tc.name == "project" {
						method, want = "POST", 201
						body = fmt.Sprintf(`{"name":"Stored cover","identifier":"STORED","cover_image_asset_id":%q}`, old)
						// A failed deferred INSERT must leave neither a partial project nor
						// migrated/bound preparation even after actual bytes were stored.
						imageStorageCommitFailure(t, w, tc.table, "INSERT", method, path, body)
					}
					bound := w.send(t, "alice", method, path, body, want)
					checkImageStorageReply(t, w, origin, bound, tc.field, old, data)
					var owner struct{ ID uuid.UUID }
					decodeAnswer(t, bound, &owner)
					scope := tc.scope
					if tc.name == "project" {
						path, scope = "/api/v0/projects/"+owner.ID.String(), assetScopeBody("project", owner.ID)
					}
					var ownerScope bool
					if err := w.pool.QueryRow(t.Context(), "SELECT "+tc.field+"_asset_id=$2 FROM "+tc.table+" WHERE id=$1", owner.ID, old).Scan(&ownerScope); err != nil || !ownerScope {
						t.Fatal("response reference differs from persisted owner", ownerScope, err)
					}
					var asset struct{ Scope json.RawMessage }
					decodeAnswer(t, assetAPICall(t, w, "alice", "GET", "/api/v0/assets/"+old.String(), "", 200), &asset)
					var expectedScope, gotScope map[string]string
					decodeAnswer(t, scope, &expectedScope)
					decodeAnswer(t, string(asset.Scope), &gotScope)
					if !maps.Equal(expectedScope, gotScope) {
						t.Fatal("asset did not retain or migrate to its final scope", gotScope, expectedScope)
					}
					waitForCompletedKinds(t, w.pool, map[string]int{"asset.cleanup": 1, "cleanup.purge_deleted": 1}, &lockedBuffer{})
					beforeRead := imageStorageSnapshot(t, w)
					checkImageStorageReply(t, w, origin, w.send(t, "alice", "GET", path, "", 200), tc.field, old, data)
					if imageStorageSnapshot(t, w) != beforeRead {
						t.Fatal("driver image read changed audit or recovery rows")
					}
					refusedBody := fmt.Sprintf(`{%q:%q}`, tc.field+"_asset_id", uuid.New())
					if code := problemCode(t, []byte(w.send(t, "alice", "PATCH", path, refusedBody, 404))); code != "asset.not_found" {
						t.Fatal(code)
					}
					if imageStorageSnapshot(t, w) != beforeRead {
						t.Fatal("refused image changed the owner or assets")
					}
					next := upload(tc.purpose, scope)
					replace := fmt.Sprintf(`{%q:%q}`, tc.field+"_asset_id", next)
					imageStorageCommitFailure(t, w, tc.table, "UPDATE", "PATCH", path, replace)
					checkImageStorageReply(t, w, origin, w.send(t, "alice", "PATCH", path, replace, 200), tc.field, next, data)
					rows := assetpg.New(w.pool)
					removed, err := rows.FindAsset(t.Context(), old)
					if err != nil || removed.DeletedAt == nil || removed.DeletedByID == nil || *removed.DeletedByID != w.ids["alice"] {
						t.Fatal("replacement lost the original tombstone", removed, err)
					}
					current, err := rows.FindAsset(t.Context(), next)
					if err != nil || current.BoundAt == nil || !current.BoundAt.Equal(*removed.DeletedAt) || !current.UpdatedAt.Equal(*current.BoundAt) {
						t.Fatal("binding/delete audit times differ", current, err)
					}
					var sameAt bool
					if err := w.pool.QueryRow(t.Context(), "SELECT updated_at=$2 FROM "+tc.table+" WHERE id=$1", owner.ID, *current.BoundAt).Scan(&sameAt); err != nil || !sameAt {
						t.Fatal("owner and binding audit differ", err)
					}
					cleared := w.send(t, "alice", "PATCH", path, fmt.Sprintf(`{%q:null}`, tc.field+"_asset_id"), 200)
					checkNullImageReply(t, cleared, tc.field)
				})
			}
		})
	}
}

func imageStorageSnapshot(t *testing.T, w issueWorld) string {
	t.Helper()
	return profileImageSnapshot(t, w) + workspaceImageSnapshot(t, w) + projectCoverSnapshot(t, w)
}

func imageStorageCommitFailure(t *testing.T, w issueWorld, table, operation, method, path, body string) {
	t.Helper()
	waitForCompletedKinds(t, w.pool, map[string]int{"asset.cleanup": 1, "cleanup.purge_deleted": 1}, &lockedBuffer{})
	before := imageStorageSnapshot(t, w)
	if _, err := w.pool.Exec(t.Context(), `CREATE OR REPLACE FUNCTION reject_image_storage_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'image deferred commit rejected'; END $$;
CREATE CONSTRAINT TRIGGER reject_image_storage_commit AFTER `+operation+" ON "+table+" DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_image_storage_commit()"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := w.pool.Exec(t.Context(), "DROP TRIGGER reject_image_storage_commit ON "+table); err != nil {
			t.Error(err)
		}
	}()
	got := w.send(t, "alice", method, path, body, 500)
	if problemCode(t, []byte(got)) != "internal_error" || strings.Contains(got, "url") || imageStorageSnapshot(t, w) != before {
		t.Fatal("deferred commit exposed a capability or retained a partial binding", got)
	}
}

func checkImageStorageReply(t *testing.T, w issueWorld, origin, body, field string, id uuid.UUID, data []byte) {
	t.Helper()
	var reply map[string]json.RawMessage
	decodeAnswer(t, body, &reply)
	var address string
	var expiry time.Time
	var gotID uuid.UUID
	for key, value := range map[string]any{field + "_asset_id": &gotID, field + "_url": &address, field + "_url_expires_at": &expiry} {
		if err := json.Unmarshal(reply[key], value); err != nil {
			t.Fatal(key, err)
		}
	}
	if gotID != id || address == "" || time.Until(expiry) < 290*time.Second || expiry.After(time.Now().Add(65*time.Minute)) {
		t.Fatal("invalid image capability", body)
	}
	base, err := url.Parse(w.base)
	if err != nil {
		t.Fatal(err)
	}
	target, err := base.Parse(address)
	if err != nil || target.Scheme+"://"+target.Host != origin {
		t.Fatal("image capability escaped the configured driver origin", address, err)
	}
	if target.Query().Get("X-Amz-Algorithm") != "" {
		start, err := time.Parse("20060102T150405Z", target.Query().Get("X-Amz-Date"))
		seconds, durationErr := strconv.Atoi(target.Query().Get("X-Amz-Expires"))
		if err != nil || durationErr != nil || !start.Add(time.Duration(seconds)*time.Second).Equal(expiry) {
			t.Fatal("S3 capability expiry differs from DTO", err, durationErr, expiry)
		}
	} else {
		checkImageReply(t, w, body, field, id, data)
	}
	res, actual := send(t, newRequest(t, "GET", target.String(), "", nil))
	if res.StatusCode != 200 || !bytes.Equal(actual, data) || !strings.HasPrefix(res.Header.Get("Content-Disposition"), "inline;") || res.Header.Get("Content-Type") != "image/png" {
		t.Fatal("real driver returned different image bytes or disposition", res.StatusCode, res.Header)
	}
	a, err := assetpg.New(w.pool).FindAsset(t.Context(), id)
	digest := sha256.Sum256(actual)
	if err != nil || a.Metadata.Final == nil || a.Metadata.Final.SHA256 != hex.EncodeToString(digest[:]) || a.BoundAt == nil || a.CreatedByID == nil || *a.CreatedByID != w.ids["alice"] {
		t.Fatal("persisted image does not match actual bytes/binding", a, err)
	}
}
