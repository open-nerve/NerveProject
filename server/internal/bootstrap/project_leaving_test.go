package bootstrap

import (
	"net/http"
	"testing"
	"uuid"
)

// leave is name's leaving project: its status and body.
func (w memberWorld) leave(t *testing.T, name string, project uuid.UUID) (int, string) {
	t.Helper()
	return call(t, w.contract, http.MethodPost, w.base+"/api/v0/projects/"+project.String()+"/leave", w.tokens[name], "")
}

// Leaving a project on the wired app (M3 design 3.7 rule 1), one leaving
// after another on memberWorld, each a memberEnding by the member himself:
//   - bob, Lab's only admin, cannot leave it while carol is its member;
//     once she has left it, he cannot either: rule 1 holds when he is
//     alone too;
//   - carol and erin, Web's member and guest, leave it; bob and dave, two
//     of its three admins, leave it; alice, its only admin now, cannot,
//     gina its member: bob's and dave's ended memberships, admins', do not
//     count;
//   - gina makes alice acme's guest, and so Web's and Ops' guest (3.3):
//     carol, Ops' admin and acme's member, is its only admin now, and
//     cannot leave it, bob its member. bob and alice, refused above, are
//     workspace admins, of beta and of acme, and carol is none: rule 1
//     holds for a project admin of either kind;
//   - Web has no admin now: gina, its member and acme's admin, leaves it as
//     a member; bob, Ops' member, leaves it, carol its admin; alice, Web's
//     guest now and its only member, leaves it, though she could not as its
//     only admin. The rule is the project role's, not the workspace's.
func TestLeavingAProject(t *testing.T) {
	w := newMemberWorld(t)
	step := func(name, member string, project uuid.UUID, status int, code string) {
		memberEnding{step: name, name: member, by: member, project: project, status: status, code: code,
			send: func() (int, string) { return w.leave(t, member, project) }}.check(t, w)
	}
	step("bob leaves Lab, its only admin; carol its member", "bob", w.lab, http.StatusConflict, "project.sole_admin")
	step("carol leaves Lab, its member", "carol", w.lab, http.StatusNoContent, "")
	step("bob leaves Lab, its only admin and member", "bob", w.lab, http.StatusConflict, "project.sole_admin")
	step("carol leaves Web, its member", "carol", w.web, http.StatusNoContent, "")
	step("erin leaves Web, its guest", "erin", w.web, http.StatusNoContent, "")
	step("bob leaves Web, an admin; alice and dave its others", "bob", w.web, http.StatusNoContent, "")
	step("dave leaves Web, an admin; alice its other", "dave", w.web, http.StatusNoContent, "")
	step("alice leaves Web, its only active admin; gina its member", "alice", w.web, http.StatusConflict, "project.sole_admin")
	if status, body := call(t, w.contract, http.MethodPatch, w.base+"/api/v0/workspace-members/"+w.acmeMembership(t, "alice").String(),
		w.tokens["gina"], `{"role":5}`); status != http.StatusOK {
		t.Fatalf("gina's making alice acme's guest = %d %s", status, body)
	}
	step("carol leaves Ops, its only admin now and acme's member; bob its member", "carol", w.ops, http.StatusConflict, "project.sole_admin")
	step("gina leaves Web, its member and acme's admin; alice its guest now", "gina", w.web, http.StatusNoContent, "")
	step("bob leaves Ops, its member; carol its admin", "bob", w.ops, http.StatusNoContent, "")
	step("alice leaves Web, its guest now and only member", "alice", w.web, http.StatusNoContent, "")
	if got, want := w.standing(t), "acme: alice 5, bob 15, carol 15, dave 15, erin 5, gina 20; beta: bob 20, carol 15; Lab: bob 20; "+
		"Ops: alice 5, carol 20"; got != want {
		t.Errorf("the world after the leavings: %s; want %s", got, want)
	}
}
