package bootstrap

import (
	"maps"
	"net/http"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// A write on a project membership decides on the membership as it reads it
// under its locks (M3 design 3.5, 3.6 convention 2), on the wired app:
// another transaction holds Web FOR NO KEY UPDATE, as a write on it does,
// and changes the membership the write names, which the write has read
// before its locks and waits for Web's row. Once the other commits:
//   - carol, Web's member, made its admin: bob, Web's admin and no
//     workspace admin, may not change an admin's role, and gina, Web's
//     member and acme's admin, may not remove one above her own role: each
//     403 project.role_too_high, the relative rule reading the role under
//     the locks, not the one read before them;
//   - gina's membership deleted: carol, Web's member, who may change no
//     role and remove nobody, gets the membership's 404, not her 403: the
//     membership is read again before the decision.
//
// The membership is as the other left it, and every other row as it was.
func TestAWriteOnAProjectMembershipDecidesOnWhatItReadUnderItsLocks(t *testing.T) {
	const madeAdmin, deleted = "UPDATE project_members SET role = 20 WHERE id = $1", "UPDATE project_members SET deleted_at = now() WHERE id = $1"
	for _, tt := range []struct {
		name, by, member, method, body, sql string
		status                              int
		code                                string
	}{
		{"bob's change of carol's role, carol made an admin", "bob", "carol", http.MethodPatch, `{"role":5}`, madeAdmin, http.StatusForbidden,
			"project.role_too_high"},
		{"gina's removal of carol, carol made an admin", "gina", "carol", http.MethodDelete, "", madeAdmin, http.StatusForbidden,
			"project.role_too_high"},
		{"carol's change of gina's role, gina's membership deleted", "carol", "gina", http.MethodPatch, `{"role":5}`, deleted,
			http.StatusNotFound, "project.member_not_found"},
		{"carol's removal of gina, gina's membership deleted", "carol", "gina", http.MethodDelete, "", deleted, http.StatusNotFound,
			"project.member_not_found"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newMemberWorld(t)
			id := w.membership(t, w.web, tt.member)
			others := rowsBut(t, w.pool, []uuid.UUID{id})
			other := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR NO KEY UPDATE", w.web)
			if tag, err := other.Exec(soon(t), tt.sql, id); err != nil || tag.RowsAffected() != 1 {
				t.Fatalf("%s: %v, %v; want one row changed", tt.sql, tag, err)
			}
			changed := rowJSON(t, other, "project_members", id)
			var body []byte
			if tt.body != "" {
				body = []byte(tt.body)
			}
			req := newRequest(t, tt.method, w.base+"/api/v0/project-members/"+id.String(), w.tokens[tt.by], body)
			w.contract.CheckRequest(t, req)
			answered := sendInBackground(req)
			pgtest.WaitForLockWaitOn(t, w.pool, "projects", 5*time.Second)
			if err := other.Commit(soon(t)); err != nil {
				t.Fatal(err)
			}
			a := receiveWithin(t, answered, 10*time.Second, "the answer to "+tt.name)
			if a.err != nil {
				t.Fatal(a.err)
			}
			w.contract.CheckResponse(t, req, a.res)
			if a.res.StatusCode != tt.status || problemCode(t, a.body) != tt.code {
				t.Errorf("%s = %d %s, want %d %s", tt.name, a.res.StatusCode, a.body, tt.status, tt.code)
			}
			if after := rowJSON(t, w.pool, "project_members", id); !reflect.DeepEqual(after, changed) {
				t.Errorf("the membership after the write:\n%v\nwant it as the other left it:\n%v", after, changed)
			}
			if after := rowsBut(t, w.pool, []uuid.UUID{id}); !maps.Equal(after, others) {
				t.Errorf("every other row after the write:\n%v\nwant them as they were:\n%v", after, others)
			}
		})
	}
}
