package bootstrap

import (
	"fmt"
	"maps"
	"net/http"
	"testing"
	"time"
	"uuid"
)

// M3 design 3.5's rule for a change of a project role on the wired app,
// each half with its counterexample, one change after another on
// memberWorld; each refusal changes no row, and each change the target's
// role alone, by its caller within the request. Before each change, the
// target's last writer is checked to be an account other than its caller:
// alice, who made the membership, but for carol's last two changes, whose
// last writers are bob and then dave (erin's row, last written by bob, is
// only refused after his change):
//   - the rule's roles: carol, Web's member, may change no role there,
//     not even a guest's to a guest's, which the relative rule would let
//     her; as Ops's admin, she makes bob, its member, a guest;
//   - one who is not a workspace admin: bob, Web's admin, cannot change his
//     own role, though he is beta's admin; nor dave's, an admin's, nor make
//     carol an admin (project.role_too_high); he makes carol a guest, and
//     dave, Web's other admin, promotes her back to a member;
//   - the workspace guest's cap: bob keeps erin, acme's guest, a guest, but
//     neither he nor gina makes her a member (422 role not_allowed);
//   - the workspace admin's exception: gina, acme's admin and Web's member,
//     makes dave, an admin, a member, carol an admin, herself an admin, and
//     alice, acme's admin and Web's, a member.
func TestTheRelativeRuleOnTheComposedApp(t *testing.T) {
	w := newMemberWorld(t)
	for _, step := range []struct {
		name    string
		by      string
		project uuid.UUID
		member  string
		role    int
		status  int
		code    string // of a refusal
	}{
		{"carol, Web's member, keeps erin a guest", "carol", w.web, "erin", 5, http.StatusForbidden, "forbidden"},
		{"bob, Web's admin, keeps erin a guest", "bob", w.web, "erin", 5, http.StatusOK, ""},
		{"carol, Ops's admin, makes bob, its member, a guest", "carol", w.ops, "bob", 5, http.StatusOK, ""},
		{"bob, Web's admin and beta's, changes his own role", "bob", w.web, "bob", 15, http.StatusConflict, "project.own_membership"},
		{"bob changes dave, another admin", "bob", w.web, "dave", 15, http.StatusForbidden, "project.role_too_high"},
		{"bob makes carol an admin", "bob", w.web, "carol", 20, http.StatusForbidden, "project.role_too_high"},
		{"bob makes erin, acme's guest, a member", "bob", w.web, "erin", 15, http.StatusUnprocessableEntity, "validation_failed"},
		{"bob makes carol a guest", "bob", w.web, "carol", 5, http.StatusOK, ""},
		{"dave, Web's other admin, makes carol a member again", "dave", w.web, "carol", 15, http.StatusOK, ""},
		{"gina, Web's member and acme's admin, makes erin a member", "gina", w.web, "erin", 15, http.StatusUnprocessableEntity,
			"validation_failed"},
		{"gina makes dave, an admin, a member", "gina", w.web, "dave", 15, http.StatusOK, ""},
		{"gina makes carol an admin", "gina", w.web, "carol", 20, http.StatusOK, ""},
		{"gina makes herself an admin", "gina", w.web, "gina", 20, http.StatusOK, ""},
		{"gina makes alice, acme's admin and Web's, a member", "gina", w.web, "alice", 15, http.StatusOK, ""},
	} {
		id := w.membership(t, step.project, step.member)
		target, others := rowJSON(t, w.pool, "project_members", id), rowsBut(t, w.pool, []uuid.UUID{id})
		if step.code == "" && target["updated_by_id"] == w.ids[step.by].String() {
			t.Fatalf("%s: the target's membership was last written by %s before it; want another writer", step.name, step.by)
		}
		start := time.Now().Truncate(time.Microsecond)
		status, body := w.change(t, step.by, step.project, step.member, step.role)
		end := time.Now()
		if status != step.status || step.code != "" && problemCode(t, []byte(body)) != step.code {
			t.Fatalf("%s = %d %s, want %d %s", step.name, status, body, step.status, step.code)
		}
		if step.code == "validation_failed" {
			var problem struct {
				Errors []struct{ Field, Code string }
			}
			if decodeAnswer(t, body, &problem); len(problem.Errors) != 1 || problem.Errors[0].Field+" "+problem.Errors[0].Code != "role not_allowed" {
				t.Errorf("%s = %s, want its one error role not_allowed", step.name, body)
			}
		}
		if after := rowsBut(t, w.pool, []uuid.UUID{id}); !maps.Equal(after, others) {
			t.Errorf("%s changed rows besides the target's:\n%v\nwant them as they were:\n%v", step.name, after, others)
		}
		after := rowJSON(t, w.pool, "project_members", id)
		if step.code != "" {
			if !maps.Equal(after, target) {
				t.Errorf("%s changed the target's membership:\n%v\nwant it as it was:\n%v", step.name, after, target)
			}
			continue
		}
		written, err := time.Parse(time.RFC3339, fmt.Sprint(after["updated_at"]))
		if err != nil || after["role"] != float64(step.role) || after["updated_by_id"] != w.ids[step.by].String() || written.Before(start) ||
			written.After(end) {
			t.Errorf("%s: the target's membership %v; want role %d, written by %s within the request, %v to %v", step.name, after, step.role,
				step.by, start, end)
		}
		for _, column := range []string{"role", "updated_by_id", "updated_at"} {
			delete(after, column)
			delete(target, column)
		}
		if !maps.Equal(after, target) {
			t.Errorf("%s changed the target's other columns:\n%v\nwant them as they were:\n%v", step.name, after, target)
		}
	}
	if got, want := w.standing(t), "acme: alice 20, bob 15, carol 15, dave 15, erin 5, gina 20; beta: bob 20, carol 15; "+
		"Lab: bob 20, carol 15; Ops: alice 20, bob 5, carol 20; Web: alice 15, bob 20, carol 20, dave 15, erin 5, gina 20"; got != want {
		t.Errorf("the world after the changes: %s; want %s", got, want)
	}
}
