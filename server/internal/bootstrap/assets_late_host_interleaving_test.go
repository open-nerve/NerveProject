package bootstrap

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestAssetLateHostInterleaving(t *testing.T) {
	for _, change := range []string{"issue delete", "issue archive", "comment delete", "comment issue archive", "project delete", "project archive", "workspace delete", "draft delete", "draft publish", "draft ascending", "draft descending", "copy target draft move", "copy source revoke"} {
		t.Run(change, func(t *testing.T) {
			var armed atomic.Bool
			attempts := make(chan *assetSealAttempt, 1)
			w, store := newAssetCompleteWorld(t, func(c *assetComponents) {
				controlled := controlledAssetSeal(c.store, attempts)
				base := c.store
				c.sealing.Sealer = assetSealFunc(func(ctx context.Context, src, dst domain.ObjectKey, etag string) (domain.ObjectInfo, error) {
					if armed.Load() {
						return controlled(ctx, src, dst, etag)
					}
					return base.Seal(ctx, src, dst, etag)
				})
			})
			item := w.newWorkItem(t)
			if change == "issue archive" || change == "comment issue archive" {
				w.send(t, "alice", "PATCH", "/api/v0/issues/"+item.String(), fmt.Sprintf(`{"state_id":%q}`, w.done), 200)
			}
			purpose, scope := "ISSUE_ATTACHMENT", assetScopeBody("issue", item)
			method, path, body, changeStatus := "DELETE", "/api/v0/issues/"+item.String(), "", 204
			want, code, actor := 404, "asset.not_found", "alice"
			switch change {
			case "issue archive":
				method, path, changeStatus, want, code = "POST", path+"/archive", 200, 409, "issue.archived"
			case "comment delete", "comment issue archive":
				comment := w.newComment(t, "bob", item)
				purpose, scope = "COMMENT_ATTACHMENT", assetScopeBody("comment", comment.ID)
				if change == "comment delete" {
					path = commentPath(comment.ID)
				} else {
					method, path, changeStatus, want, code = "POST", path+"/archive", 200, 409, "issue.archived"
				}
			case "project delete":
				path = "/api/v0/projects/" + w.web.String()
			case "project archive":
				method, path, changeStatus, want, code = "POST", "/api/v0/projects/"+w.web.String()+"/archive", 200, 409, "project.archived"
			case "workspace delete":
				path = "/api/v0/workspaces/acme"
			case "copy source revoke":
				w.send(t, "alice", "PATCH", "/api/v0/projects/"+w.web.String(), `{"network":0}`, 200)
				member := queryIDs(t, w.pool, "SELECT id FROM project_members WHERE project_id=$1 AND member_id=$2 AND is_active AND deleted_at IS NULL", w.web, w.ids["bob"])[0]
				path = "/api/v0/project-members/" + member.String()
			default:
				initial, target := w.web, w.web
				if change == "draft ascending" || change == "draft descending" || change == "copy target draft move" {
					target = createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Other", "OTHER")
					w.makeMember(t, "bob", target, shared.RoleMember)
					if change == "draft descending" {
						initial, target = target, initial
					}
				}
				d := createDraftHTTP(t, w, "bob", fmt.Sprintf(`{"project_id":%q,"payload":{"name":"Late host"}}`, initial))
				purpose, scope, path, actor = "DRAFT_ISSUE_ATTACHMENT", assetScopeBody("draft", d.ID), draftPath(d.ID), "bob"
				if change == "draft publish" {
					method, path, changeStatus = "POST", publishPath(d.ID), 201
				}
				if change == "draft ascending" || change == "draft descending" || change == "copy target draft move" {
					method, body, changeStatus, want, code = "PATCH", fmt.Sprintf(`{"project_id":%q}`, target), 200, 409, "asset.scope_changed"
				}
			}
			requestPath, requestBody := "", `{}`
			if change == "copy target draft move" || change == "copy source revoke" {
				source := uploadedAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", assetScopeBody("issue", item), "source.txt", "text/plain", []byte("old"))
				requestPath = duplicateAssetPath(source.Asset.ID)
				requestBody = fmt.Sprintf(`{"purpose":%q,"scope":%s}`, purpose, scope)
			} else {
				file := prepareAssetHTTP(t, w, "bob", purpose, scope, "x.txt", "text/plain", 3)
				putAssetHTTP(t, w, file, []byte("old"), 204)
				requestPath = completeAssetPath(file.Asset.ID)
			}
			armed.Store(true)
			req := newRequest(t, "POST", w.base+requestPath, w.tokens["bob"], []byte(requestBody))
			response := sendInBackground(req)
			attempt := awaitAssetSeal(t, attempts, response)
			defer attempt.resume(nil)
			w.send(t, actor, method, path, body, changeStatus)
			attempt.resume(nil)
			got := receiveWithin(t, response, 5*time.Second, "publication after host change")
			if got.err != nil || got.res.StatusCode != want || problemCode(t, got.body) != code {
				t.Fatalf("late host=%+v want=%d/%s", got, want, code)
			}
			w.contract.CheckResponse(t, req, got.res)
			row, err := assetpg.New(w.pool).FindAsset(t.Context(), attempt.key.AssetID())
			if err != nil || row.Uploaded || row.Metadata.Final != nil || len(row.Metadata.Candidates) != 1 || row.Metadata.Candidates[0].Key != attempt.key {
				t.Fatalf("late host lost recovery: %+v %v", row, err)
			}
			if _, err := store.Inspect(t.Context(), attempt.key); err != nil {
				t.Fatal(err)
			}
		})
	}
}
