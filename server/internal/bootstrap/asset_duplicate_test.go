package bootstrap

import (
	"bytes"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func uploadedAssetHTTP(t *testing.T, w issueWorld, actor, purpose, scope, name, contentType string, data []byte) preparedAssetAnswer {
	t.Helper()
	prepared := prepareAssetHTTP(t, w, actor, purpose, scope, name, contentType, len(data))
	putAssetHTTP(t, w, prepared, data, 204)
	decodeAnswer(t, assetAPICall(t, w, actor, "POST", completeAssetPath(prepared.Asset.ID), `{}`, 200), &prepared.Asset)
	return prepared
}
func duplicateAssetPath(id uuid.UUID) string { return "/api/v0/assets/" + id.String() + "/duplicate" }
func assetScopeBody(kind string, id uuid.UUID) string {
	return fmt.Sprintf(`{"kind":%q,"id":%q}`, kind, id)
}

func TestAssetDuplicateAuthorizationAndBytes(t *testing.T) {
	w := newIssueWorld(t)
	w.send(t, "alice", "POST", "/api/v0/workspaces", `{"name":"Elsewhere","slug":"elsewhere"}`, 201)
	projects := []uuid.UUID{w.web, createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Second", "SECOND"), createdProject(t, w.contract, w.base, w.tokens["alice"], "elsewhere", "Elsewhere", "ELSE")}
	slices.SortFunc(projects, func(a, b uuid.UUID) int { return a.Compare(b) })
	items := make([]uuid.UUID, len(projects))
	for n, p := range projects {
		items[n] = w.newWorkItemBy(t, "alice", p, `{"name":"Copy host"}`)
	}
	seen := map[domain.ObjectKey]bool{}
	for n, pair := range [][2]int{{0, 1}, {1, 0}, {1, 2}, {2, 1}, {0, 0}} {
		t.Run(fmt.Sprintf("project order %d to %d", pair[0], pair[1]), func(t *testing.T) {
			data := []byte(fmt.Sprintf("same file name, distinct bytes %d", n))
			source := uploadedAssetHTTP(t, w, "alice", "ISSUE_ATTACHMENT", assetScopeBody("issue", items[pair[0]]), "same.txt", "text/plain", data)
			before, err := assetpg.New(w.pool).FindAsset(t.Context(), source.Asset.ID)
			if err != nil {
				t.Fatal(err)
			}
			body := fmt.Sprintf(`{"purpose":"ISSUE_ATTACHMENT","scope":%s}`, assetScopeBody("issue", items[pair[1]]))
			for range 2 {
				var got preparedAssetAnswer
				decodeAnswer(t, assetAPICall(t, w, "alice", "POST", duplicateAssetPath(source.Asset.ID), body, 201), &got.Asset)
				target, err := assetpg.New(w.pool).FindAsset(t.Context(), got.Asset.ID)
				if err != nil || got.Asset.ID == source.Asset.ID || got.Asset.Name != "same.txt" || got.Asset.State != "uploaded" || got.Asset.IsBound || !target.Uploaded || target.Metadata.Final == nil || target.Metadata.Final.Key == before.Metadata.Final.Key || target.Metadata.Origin != domain.OriginCopy || target.Metadata.Staging != nil || len(target.Metadata.Candidates) != 0 || target.UploadExpiresAt != nil || target.Placement.IssueID != items[pair[1]] || target.Metadata.Final.SHA256 != before.Metadata.Final.SHA256 {
					t.Fatalf("not an independent completed copy: %+v %v", target, err)
				}
				if seen[target.Metadata.Final.Key] {
					t.Fatal("repeated copy reused an object")
				}
				seen[target.Metadata.Final.Key] = true
				res, actual := readAssetHTTP(t, w, "alice", got.Asset.ID)
				if !bytes.Equal(actual, data) || !strings.HasPrefix(res.Header.Get("Content-Disposition"), "attachment;") {
					t.Fatal("copy bytes or disposition differ")
				}
			}
			after, err := assetpg.New(w.pool).FindAsset(t.Context(), source.Asset.ID)
			if err != nil || !reflect.DeepEqual(after, before) {
				t.Fatalf("copy changed its source: %+v %v", after, err)
			}
		})
	}
	t.Run("both sides authorize in source first order", func(t *testing.T) {
		item := w.newWorkItem(t)
		alice := uploadedAssetHTTP(t, w, "alice", "ISSUE_ATTACHMENT", assetScopeBody("issue", item), "private.png", "image/png", assetTestPNG(t))
		bob := uploadedAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", assetScopeBody("issue", item), "own.png", "image/png", assetTestPNG(t))
		secret := createdProjectFrom(t, w.contract, w.base, w.tokens["alice"], "acme", `{"name":"Secret","identifier":"SECRET","network":0}`)
		w.makeMember(t, "bob", secret, 15)
		secretItem := w.newWorkItemBy(t, "bob", secret, `{"name":"Hidden source"}`)
		hidden := uploadedAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", assetScopeBody("issue", secretItem), "hidden.png", "image/png", assetTestPNG(t))
		membership := queryIDs(t, w.pool, "SELECT id FROM project_members WHERE project_id=$1 AND member_id=$2 AND is_active AND deleted_at IS NULL", secret, w.ids["bob"])[0]
		w.send(t, "alice", "DELETE", "/api/v0/project-members/"+membership.String(), "", 204)
		for _, tc := range []struct {
			name   string
			source uuid.UUID
			target string
			status int
			code   string
		}{
			{"other applicant to writable own avatar", alice.Asset.ID, `{"purpose":"USER_AVATAR","scope":{"kind":"user"}}`, 404, "asset.not_found"},
			{"hidden source before forbidden target", alice.Asset.ID, fmt.Sprintf(`{"purpose":"PROJECT_COVER","scope":%s}`, assetScopeBody("project", w.web)), 404, "asset.not_found"},
			{"readable source to forbidden target", bob.Asset.ID, fmt.Sprintf(`{"purpose":"PROJECT_COVER","scope":%s}`, assetScopeBody("project", w.web)), 403, "forbidden"},
			{"own hidden host to writable own avatar", hidden.Asset.ID, `{"purpose":"USER_AVATAR","scope":{"kind":"user"}}`, 404, "asset.not_found"},
			{"own hidden host before forbidden target", hidden.Asset.ID, fmt.Sprintf(`{"purpose":"PROJECT_COVER","scope":%s}`, assetScopeBody("project", w.web)), 404, "asset.not_found"},
			{"missing source before forbidden target", uuid.New(), fmt.Sprintf(`{"purpose":"PROJECT_COVER","scope":%s}`, assetScopeBody("project", w.web)), 404, "asset.not_found"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				before := queryIDs(t, w.pool, "SELECT id FROM file_assets ORDER BY id")
				body := assetAPICall(t, w, "bob", "POST", duplicateAssetPath(tc.source), tc.target, tc.status)
				if problemCode(t, []byte(body)) != tc.code {
					t.Fatal(body)
				}
				after := queryIDs(t, w.pool, "SELECT id FROM file_assets ORDER BY id")
				if !slices.Equal(before, after) {
					t.Fatal("denied copy created a target")
				}
			})
		}
	})
}

func TestAssetDuplicateRevalidatesImagePurpose(t *testing.T) {
	w := newIssueWorld(t)
	item := w.newWorkItem(t)
	for _, tc := range []struct {
		name, contentType string
		data              []byte
		status            int
		code              string
	}{
		{"actual PNG", "image/png", assetTestPNG(t), 201, ""},
		{"fake PNG attachment", "image/png", []byte("looks like no PNG"), 422, "asset.content_invalid"},
		{"plain attachment", "text/plain", []byte("not raster"), 415, "asset.type_unsupported"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := uploadedAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", assetScopeBody("issue", item), "source", tc.contentType, tc.data)
			body := assetAPICall(t, w, "bob", "POST", duplicateAssetPath(source.Asset.ID), `{"purpose":"USER_AVATAR","scope":{"kind":"user"},"name":"copied.png"}`, tc.status)
			if tc.status != 201 {
				if problemCode(t, []byte(body)) != tc.code {
					t.Fatal(body)
				}
				rows := queryIDs(t, w.pool, "SELECT id FROM file_assets WHERE entity_type='USER_AVATAR' AND NOT is_uploaded AND user_id=$1", w.ids["bob"])
				if tc.status == 422 {
					if len(rows) != 1 {
						t.Fatalf("failed copy target lost: %v", rows)
					}
					target, err := assetpg.New(w.pool).FindAsset(t.Context(), rows[0])
					if err != nil || target.Metadata.Origin != domain.OriginCopy || target.Metadata.Staging != nil || len(target.Metadata.Candidates) != 1 || target.UploadExpiresAt != nil {
						t.Fatalf("failed copy has no recovery record: %+v %v", target, err)
					}
				}
				return
			}
			var got preparedAssetAnswer
			decodeAnswer(t, body, &got.Asset)
			if got.Asset.Width == nil || *got.Asset.Width != 2 || got.Asset.Height == nil || *got.Asset.Height != 3 || got.Asset.Name != "copied.png" {
				t.Fatal(body)
			}
			res, data := readAssetHTTP(t, w, "bob", got.Asset.ID)
			if !bytes.Equal(data, tc.data) || !strings.HasPrefix(res.Header.Get("Content-Disposition"), "inline;") {
				t.Fatal("avatar copy not verified")
			}
		})
	}
}
