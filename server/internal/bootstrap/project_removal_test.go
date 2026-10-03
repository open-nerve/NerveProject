package bootstrap

import (
	"maps"
	"net/http"
	"testing"
	"time"
	"uuid"
)

// remove is by's removal of name's membership of project: its status and
// body.
func (w memberWorld) remove(t *testing.T, by string, project uuid.UUID, name string) (int, string) {
	t.Helper()
	return call(t, w.contract, http.MethodDelete, w.base+"/api/v0/project-members/"+w.membership(t, project, name).String(), w.tokens[by], "")
}

// memberEnding is a step that ends name's membership of project, by by,
// or is refused with code: send sends its request, which answers status.
type memberEnding struct {
	step, name, by string
	project        uuid.UUID
	status         int
	code           string // of a refusal
	send           func() (int, string)
}

// check runs e and checks it: it changes no row besides the membership's;
// a refusal leaves the membership as it was; an ending ends it, not deleted,
// by e.by at a moment within its request, its other columns as they were,
// its role among them. The member has his display settings in the project
// before, which stay; before an ending, the membership was last written by
// an account other than e.by, so that its writer shows.
func (e memberEnding) check(t *testing.T, w memberWorld) {
	t.Helper()
	id := w.membership(t, e.project, e.name)
	target, others := rowJSON(t, w.pool, "project_members", id), rowsBut(t, w.pool, []uuid.UUID{id})
	var settings int
	if err := w.pool.QueryRow(soon(t), "SELECT count(*) FROM project_user_properties WHERE project_id = $1 AND user_id = $2 AND "+
		"deleted_at IS NULL", e.project, w.ids[e.name]).Scan(&settings); err != nil || settings != 1 {
		t.Fatalf("%s: %s's display settings in the project: %d, %v; want his one", e.step, e.name, settings, err)
	}
	if e.code == "" && target["updated_by_id"] == w.ids[e.by].String() {
		t.Fatalf("%s: the target's membership was last written by %s before it; want another writer", e.step, e.by)
	}
	before := time.Now().Truncate(time.Microsecond)
	status, body := e.send()
	after := time.Now()
	if status != e.status || e.code != "" && problemCode(t, []byte(body)) != e.code {
		t.Fatalf("%s = %d %s, want %d %s", e.step, status, body, e.status, e.code)
	}
	if got := rowsBut(t, w.pool, []uuid.UUID{id}); !maps.Equal(got, others) {
		t.Errorf("%s changed rows besides the target's:\n%v\nwant them as they were:\n%v", e.step, got, others)
	}
	ended := rowJSON(t, w.pool, "project_members", id)
	if e.code != "" {
		if !maps.Equal(ended, target) {
			t.Errorf("%s changed the target's membership:\n%v\nwant it as it was:\n%v", e.step, ended, target)
		}
		return
	}
	at, err := time.Parse(time.RFC3339Nano, ended["updated_at"].(string))
	if ended["is_active"] != false || ended["deleted_at"] != nil || ended["updated_by_id"] != w.ids[e.by].String() || err != nil ||
		at.Before(before) || at.After(after) {
		t.Errorf("%s: the target's membership %v; want it ended, not deleted, by %s within %v to %v", e.step, ended, e.by, before, after)
	}
	for _, column := range []string{"is_active", "updated_by_id", "updated_at"} {
		delete(ended, column)
		delete(target, column)
	}
	if !maps.Equal(ended, target) {
		t.Errorf("%s changed the target's other columns, its role among them:\n%v\nwant them as they were:\n%v", e.step, ended, target)
	}
}

// Removing a project member on the wired app (M3 design 3.5), one removal
// after another on memberWorld, each a memberEnding; the project keeps an
// admin.
//   - carol, Web's member, may remove nobody: erin, a guest, 403;
//   - bob, Web's admin, cannot remove himself (409); nor can gina, its
//     member and acme's admin (409);
//   - gina cannot remove bob, Web's admin, though she is acme's admin
//     (403 project.role_too_high: 3.5 gives no exception here);
//   - bob removes dave, another admin, and erin, a guest; gina removes
//     carol, a member of her own role;
//   - dave's ended membership is 404 to bob, and to gina, before the
//     rule that would refuse her an admin's (403);
//   - dave, a member of acme, joins Web again: his row is back, a
//     member's, 15, the lesser of its 20 and his workspace role (3.5,
//     3.6 convention 6), made when it was.
func TestRemovingAProjectMember(t *testing.T) {
	w := newMemberWorld(t)
	daves := w.membership(t, w.web, "dave")
	made := rowJSON(t, w.pool, "project_members", daves)["created_at"]
	for _, step := range []struct {
		name, by, member string
		status           int
		code             string // of a refusal
	}{
		{"carol, Web's member, removes erin", "carol", "erin", http.StatusForbidden, "forbidden"},
		{"bob, Web's admin, removes himself", "bob", "bob", http.StatusConflict, "project.own_membership"},
		{"gina, Web's member and acme's admin, removes herself", "gina", "gina", http.StatusConflict, "project.own_membership"},
		{"gina removes bob, Web's admin", "gina", "bob", http.StatusForbidden, "project.role_too_high"},
		{"bob removes dave, another admin", "bob", "dave", http.StatusNoContent, ""},
		{"bob removes erin, a guest", "bob", "erin", http.StatusNoContent, ""},
		{"gina removes carol, a member", "gina", "carol", http.StatusNoContent, ""},
		{"bob removes dave again", "bob", "dave", http.StatusNotFound, "project.member_not_found"},
		{"gina removes dave, an admin's ended membership", "gina", "dave", http.StatusNotFound, "project.member_not_found"},
	} {
		memberEnding{step: step.name, name: step.member, by: step.by, project: w.web, status: step.status, code: step.code,
			send: func() (int, string) { return w.remove(t, step.by, w.web, step.member) }}.check(t, w)
	}
	status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/projects/"+w.web.String()+"/join", w.tokens["dave"], "")
	var joined struct {
		MemberRole *int `json:"member_role"`
	}
	if status != http.StatusOK {
		t.Fatalf("dave's joining Web again = %d %s", status, body)
	}
	again := rowJSON(t, w.pool, "project_members", daves)
	if decodeAnswer(t, body, &joined); joined.MemberRole == nil || *joined.MemberRole != 15 || w.membership(t, w.web, "dave") != daves ||
		again["created_at"] != made {
		t.Errorf("dave joins Web again as %s, his membership %s made at %v; want his row back, %s, made at %v, as a member", body,
			w.membership(t, w.web, "dave"), again["created_at"], daves, made)
	}
	if got, want := w.standing(t), "acme: alice 20, bob 15, carol 15, dave 15, erin 5, gina 20; beta: bob 20, carol 15; "+
		"Lab: bob 20, carol 15; Ops: alice 20, bob 15, carol 20; Web: alice 20, bob 20, dave 15, gina 15"; got != want {
		t.Errorf("the world after the removals: %s; want %s", got, want)
	}
}
