package httpadapter_test

import (
	"fmt"
	"testing"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// These are consumer-side error contracts, without importing asset's domain.
// Real binding eligibility and rollback are exercised through bootstrap HTTP.
func TestContentAssetRefusals(t *testing.T) {
	for _, tc := range []struct {
		code, detail string
		kind         shared.Kind
		status       int
		ops          []string
	}{
		{"asset.not_found", "The asset does not exist, or you cannot see it.", shared.KindNotFound, 404, []string{"create", "update", "publish"}},
		{"asset.not_uploaded", "The asset's content has not been completed.", shared.KindConflict, 409, []string{"update"}},
	} {
		for _, op := range tc.ops {
			t.Run(op+"/"+tc.code, func(t *testing.T) {
				refusal := shared.NewError(tc.kind, tc.code, tc.detail)
				uc := httpadapter.UseCases{CreateIssue: &fakeCreate{err: refusal}, UpdateIssue: &fakeUpdate{err: refusal}, PublishDraft: &fakeGet{err: refusal}}
				method, path, body := "POST", "/api/v0/projects/"+webID.String()+"/issues", `{"name":"Body"}`
				if op == "update" {
					method, path, body = "PATCH", "/api/v0/issues/"+webSeven.ID.String(), `{"description_html":"<p>Body</p>"}`
				}
				if op == "publish" {
					path, body = "/api/v0/drafts/"+webID.String()+"/publish", ""
				}
				res, got := do(t, newServer(t, uc), request(method, path, "alice", body))
				title := "Not Found"
				if tc.status == 409 {
					title = "Conflict"
				}
				want := fmt.Sprintf("{\"status\":%d,\"code\":%q,\"title\":%q,\"detail\":%q}\n", tc.status, tc.code, title, tc.detail)
				if res.StatusCode != tc.status || got != want {
					t.Fatalf("%d %s want %d %s", res.StatusCode, got, tc.status, want)
				}
			})
		}
	}
}
