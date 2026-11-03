package bootstrap

import (
	"bytes"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Invitations, acceptance and project membership all use the public API. This
// supplements the older issueWorld fixture without widening its SQL seeding.
func contentPermissionWorld(t *testing.T) issueWorld {
	t.Helper()
	w, _ := newImageStorageWorld(t, "local")
	w.send(t, "alice", "PATCH", "/api/v0/projects/"+w.web.String(), `{"network":0}`, 200)
	for _, name := range []string{"bob", "walt", "wanda", "olga"} {
		email := name + "@example.com"
		w.tokens[name] = registerAccount(t, w.contract, w.base, email).AccessToken
		w.ids[name] = accountID(t, w.contract, w.base, w.tokens[name])
		if name == "olga" {
			continue
		}
		role := shared.RoleMember
		if name == "wanda" {
			role = shared.RoleAdmin
		}
		link := inviteAs(t, w.contract, w.base, w.tokens["alice"], "acme", email, role)
		answerInvitation(t, w.contract, w.base, w.tokens[name], "accept", link, 200)
		w.makeMember(t, name, w.web, role)
	}
	return w
}

// Every rejected operation must leave the complete issue/asset graph intact.
// For hidden asset paths also compare the exact missing-resource reply, with a
// fixed request ID and only transport wall-clock Date excluded.
func contentPermissionCall(t *testing.T, w issueWorld, who, method, path, body string, want int, missingPath string) string {
	t.Helper()
	var before map[string]string
	if want >= 400 {
		before = contentAssetState(t, w)
	}
	request := func(path string) (*http.Response, []byte) {
		t.Helper()
		req := newRequest(t, method, w.base+path, w.tokens[who], []byte(body))
		req.Header.Set("X-Request-ID", "00000000-0000-4000-8000-000000000001")
		res, raw := send(t, req)
		w.contract.CheckResponse(t, req, res)
		return res, raw
	}
	res, raw := request(path)
	if res.StatusCode != want {
		t.Fatalf("%s %s as %s=%d %s, want %d", method, path, who, res.StatusCode, raw, want)
	}
	if want == 404 && missingPath != "" {
		missing, missingRaw := request(missingPath)
		res.Header.Del("Date")
		missing.Header.Del("Date")
		if missing.StatusCode != want || !bytes.Equal(raw, missingRaw) || !reflect.DeepEqual(res.Header, missing.Header) {
			t.Fatal("hidden and missing asset replies differ", res.Header, missing.Header, string(raw), string(missingRaw))
		}
	}
	if before != nil && !reflect.DeepEqual(before, contentAssetState(t, w)) {
		t.Fatal("refusal changed issue or asset graph")
	}
	return string(raw)
}

func TestContentAssetPermissions(t *testing.T) {
	for _, tc := range []struct {
		name, actor, creator string
		guest, all           bool
		read, write          int
	}{
		{"project admin", "alice", "bob", false, false, 200, 200},
		{"member", "bob", "alice", false, false, 200, 200},
		{"guest own", "walt", "walt", true, false, 200, 403},
		{"guest hidden", "walt", "bob", true, false, 404, 404},
		{"guest sees all", "walt", "bob", true, true, 200, 403},
		{"admin guest own", "wanda", "wanda", true, false, 200, 200},
		{"admin guest hidden", "wanda", "bob", true, false, 404, 404},
		{"admin guest sees all", "wanda", "bob", true, true, 200, 200},
		{"private project outsider", "olga", "bob", false, false, 404, 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := contentPermissionWorld(t)
			item := w.newWorkItemBy(t, tc.creator, w.web, `{"name":"Permission target"}`)
			uploader := "bob"
			if tc.actor == uploader {
				uploader = "alice"
			}
			comment := w.newComment(t, uploader, item)
			draft := createDraftHTTP(t, w, uploader, fmt.Sprintf(`{"project_id":%q,"payload":{"name":"Private draft"}}`, w.web))
			data := assetTestPNG(t)
			files := make(map[string]uuid.UUID)
			for _, host := range []struct {
				kind, prefix string
				id           uuid.UUID
			}{
				{"issue", "ISSUE", item}, {"comment", "COMMENT", comment.ID}, {"draft", "DRAFT_ISSUE", draft.ID},
			} {
				for _, suffix := range []string{"ATTACHMENT", "DESCRIPTION"} {
					purpose := host.prefix + "_" + suffix
					file := uploadedAssetHTTP(t, w, uploader, purpose, assetScopeBody(host.kind, host.id), "private-name.png", "image/png", data)
					files[purpose] = file.Asset.ID
					if suffix == "DESCRIPTION" {
						html := issueImageHTML(file.Asset.ID)
						switch host.kind {
						case "issue":
							saveIssueImage(t, w, uploader, item, html, false, 200)
						case "comment":
							w.send(t, uploader, "PATCH", commentPath(comment.ID), commentAssetsBody(t, html), 200)
						case "draft":
							draftRequest(t, w, uploader, "PATCH", draftPath(draft.ID), draftPayloadBody(t, map[string]any{"name": "Private draft", "description_html": html}), 200)
						}
					}
				}
			}
			private := uploadedAssetHTTP(t, w, uploader, "ISSUE_DESCRIPTION", assetScopeBody("issue", item), "unreferenced.png", "image/png", data)
			if tc.all {
				w.send(t, "alice", "PATCH", "/api/v0/projects/"+w.web.String(), `{"guest_view_all_features":true}`, 200)
			}
			if tc.guest {
				membership := queryIDs(t, w.pool, "SELECT id FROM project_members WHERE project_id=$1 AND member_id=$2 AND is_active AND deleted_at IS NULL", w.web, w.ids[tc.actor])[0]
				w.send(t, "alice", "PATCH", "/api/v0/project-members/"+membership.String(), `{"role":5}`, 200)
			}
			// Preparation/complete prove the actor's own write path independently
			// of host-readable files belonging to a different applicant.
			prepareWant := tc.write
			if prepareWant == 200 {
				prepareWant = 201
			}
			body := assetPrepareBody("ISSUE_ATTACHMENT", assetScopeBody("issue", item), "own.png", "image/png", int64(len(data)))
			prepared := contentPermissionCall(t, w, tc.actor, "POST", "/api/v0/assets", body, prepareWant, "")
			if prepareWant == 201 {
				var own preparedAssetAnswer
				decodeAnswer(t, prepared, &own)
				putAssetHTTP(t, w, own, data, 204)
				assetAPICall(t, w, tc.actor, "POST", completeAssetPath(own.Asset.ID), `{}`, 200)
			}
			copyBody := fmt.Sprintf(`{"purpose":"ISSUE_ATTACHMENT","scope":%s}`, assetScopeBody("issue", item))
			for _, purpose := range []string{"ISSUE_ATTACHMENT", "ISSUE_DESCRIPTION", "COMMENT_ATTACHMENT", "COMMENT_DESCRIPTION", "DRAFT_ISSUE_ATTACHMENT", "DRAFT_ISSUE_DESCRIPTION"} {
				t.Run(purpose, func(t *testing.T) {
					id := files[purpose]
					path, missing := "/api/v0/assets/"+id.String(), "/api/v0/assets/"+uuid.New().String()
					read, write := tc.read, tc.write
					if strings.HasPrefix(purpose, "DRAFT_") {
						read, write = 404, 404
					}
					contentPermissionCall(t, w, tc.actor, "GET", path, "", read, missing)
					contentPermissionCall(t, w, tc.actor, "GET", path+"/content", "", read, missing+"/content")
					if read == 200 {
						_, got := readAssetHTTP(t, w, tc.actor, id)
						if !bytes.Equal(got, data) {
							t.Fatal("host reader got different bytes")
						}
					}
					contentPermissionCall(t, w, tc.actor, "POST", path+"/complete", `{}`, 404, missing+"/complete")
					copyWant := write
					if copyWant == 200 {
						copyWant = 201
					}
					contentPermissionCall(t, w, tc.actor, "POST", path+"/duplicate", copyBody, copyWant, missing+"/duplicate")
					remove := write
					if read == 200 && strings.HasPrefix(purpose, "COMMENT_") {
						remove = 403
						if tc.actor == "alice" || tc.actor == "wanda" {
							remove = 200
						}
					}
					if remove == 200 {
						remove = 204
						if strings.HasSuffix(purpose, "DESCRIPTION") {
							remove = 409
						}
					}
					contentPermissionCall(t, w, tc.actor, "DELETE", path, "", remove, missing)
				})
			}
			path, missing := "/api/v0/assets/"+private.Asset.ID.String(), "/api/v0/assets/"+uuid.New().String()
			for _, op := range []struct{ method, suffix, body string }{{"GET", "", ""}, {"GET", "/content", ""}, {"POST", "/duplicate", copyBody}, {"DELETE", "", ""}} {
				contentPermissionCall(t, w, tc.actor, op.method, path+op.suffix, op.body, 404, missing+op.suffix)
			}
			privateHTMLStatus := tc.write
			if privateHTMLStatus == 200 {
				privateHTMLStatus = 404
			}
			contentPermissionCall(t, w, tc.actor, "PATCH", "/api/v0/issues/"+item.String(), fmt.Sprintf(`{"description_html":%q}`, issueImageHTML(private.Asset.ID)), privateHTMLStatus, "")
			contentPermissionCall(t, w, tc.actor, "PATCH", "/api/v0/issues/"+item.String(), fmt.Sprintf(`{"description_html":%q}`, issueImageHTML(files["ISSUE_DESCRIPTION"])), tc.write, "")
			commentWant := 403
			if tc.read == 404 {
				commentWant = 404
			}
			contentPermissionCall(t, w, tc.actor, "PATCH", commentPath(comment.ID), commentAssetsBody(t, issueImageHTML(files["COMMENT_DESCRIPTION"])), commentWant, "")
		})
	}
}
