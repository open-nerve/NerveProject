package bootstrap

import (
	"net/http"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestIssueListPermissionsMemberRoles(t *testing.T) {
	w := newIssueWorld(t)
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	w.newWorkItemBy(t, "bob", w.web, `{"name":"Existing target"}`)
	for _, state := range []struct {
		name, update, code string
		status             int
	}{
		{"guest", "role=5", "forbidden", 403},
		{"inactive", "role=15,is_active=false", "workspace.not_found", 404},
		{"deleted", "is_active=true,deleted_at=now()", "workspace.not_found", 404},
	} {
		t.Run(state.name, func(t *testing.T) {
			if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE workspace_members SET "+state.update+" WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["walt"]); err != nil {
				t.Fatal(err)
			}
			for _, relation := range []string{"assigned", "created", "subscribed"} {
				body := w.send(t, "walt", http.MethodGet, memberIssuesPath(w, "bob")+"?relation="+relation, "", state.status)
				if problemCode(t, []byte(body)) != state.code {
					t.Fatal(body)
				}
			}
		})
	}
}
