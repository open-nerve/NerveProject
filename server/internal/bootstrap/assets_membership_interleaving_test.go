package bootstrap

import (
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

func TestAssetMembershipInterleaving(t *testing.T) {
	for _, scope := range []string{"workspace", "project"} {
		for _, revokeFirst := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s revoke first=%t", scope, revokeFirst), func(t *testing.T) {
				url := pgtest.NewDatabase(t)
				w := newIssueWorldOnRequests(t, url, openPool(t, url), func(*config.Config) {}, &requestQueries{})
				w.send(t, "alice", "PATCH", "/api/v0/projects/"+w.web.String(), `{"network":0}`, 200)
				item := w.newWorkItem(t)
				file := uploadedAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", assetScopeBody("issue", item), "private.txt", "text/plain", []byte("private"))
				contentPath := "/api/v0/assets/" + file.Asset.ID.String() + "/content"
				var oldGrant struct {
					URL string `json:"url"`
				}
				decodeAnswer(t, assetAPICall(t, w, "bob", "GET", contentPath, "", 200), &oldGrant)
				parentID, parentTable := w.web, "projects"
				if scope == "workspace" {
					parentID, parentTable = w.acme, "workspaces"
				}
				memberTable := scope + "_members"
				member := queryIDs(t, w.pool, "SELECT id FROM "+memberTable+" WHERE "+scope+"_id=$1 AND member_id=$2 AND is_active AND deleted_at IS NULL", parentID, w.ids["bob"])[0]
				revoke := newRequest(t, "DELETE", w.base+"/api/v0/"+scope+"-members/"+member.String(), w.tokens["alice"], nil)
				sign := newRequest(t, "GET", w.base+contentPath, w.tokens["bob"], nil)
				revokeProbe, signProbe := w.observe(revoke), w.observe(sign)
				var barrier pgx.Tx
				var revoked, signed <-chan answer
				if revokeFirst {
					// Keep the membership readable by the signer's precheck, while
					// the real revocation owns the parent and waits to update it.
					barrier = lockedTables(t, w.pool, memberTable, "SHARE")
					defer requestRelease(t, barrier)
					revoked = sendInBackground(revoke)
					pid := requestBlocked(t, w.pool, revokeProbe, barrier.Conn().PgConn().PID(), memberTable)
					signed = sendInBackground(sign)
					requestBlocked(t, w.pool, signProbe, pid, parentTable)
				} else {
					barrier = holding(t, w.pool, "SELECT id FROM file_assets WHERE id=$1 FOR SHARE", file.Asset.ID)
					defer requestRelease(t, barrier)
					signed = sendInBackground(sign)
					pid := requestBlocked(t, w.pool, signProbe, barrier.Conn().PgConn().PID(), "file_assets")
					revoked = sendInBackground(revoke)
					requestBlocked(t, w.pool, revokeProbe, pid, parentTable)
				}
				requestRelease(t, barrier)
				r := receiveWithin(t, revoked, 5*time.Second, "membership revocation")
				if r.err != nil || r.res.StatusCode != 204 {
					t.Fatalf("revoke=%+v", r)
				}
				w.contract.CheckResponse(t, revoke, r.res)
				s := receiveWithin(t, signed, 5*time.Second, "serialized content signature")
				want := 200
				if revokeFirst {
					want = 404
				}
				if s.err != nil || s.res.StatusCode != want {
					t.Fatalf("sign=%+v want=%d", s, want)
				}
				w.contract.CheckResponse(t, sign, s.res)
				assetAPICall(t, w, "bob", "GET", contentPath, "", 404)
				// A prior capability is intentionally valid until its short expiry.
				raw, bytes := send(t, newRequest(t, "GET", w.base+oldGrant.URL, "", nil))
				if raw.StatusCode != 200 || string(bytes) != "private" {
					t.Fatalf("prior grant=%d %s", raw.StatusCode, bytes)
				}
			})
		}
	}
}
