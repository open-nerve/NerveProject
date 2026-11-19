package bootstrap

import (
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// countsAre checks read counts acme's intakes as want, in project id order.
func countsAre(t *testing.T, what string, read reader, want ...answeredCount) {
	t.Helper()
	want = slices.Clone(want)
	slices.SortFunc(want, func(a, b answeredCount) int { return a.ProjectID.Compare(b.ProjectID) })
	if want == nil {
		want = []answeredCount{}
	}
	if got := countsIn(t, read(t, "/api/v0/workspaces/acme/intake-counts")); !slices.Equal(got, want) {
		t.Errorf("%s counts %+v; want %+v", what, got, want)
	}
}

// H9's counts through the app (M7 P2b spec 2.6, 4). In acme, alice makes
// Ops, Far, Shut and Old besides Web, and turns each one's intake on; bob
// is a member and walt a guest of each but Far, each showing its guests
// only their own work items. In Web, alice, bob and walt each submit a
// work item, and bob three more, which alice snoozes till an hour on,
// declines and accepts; alice submits one to Far, bob one to Shut and one
// to Old; then alice turns Shut's intake off and archives Old. alice
// counts Web's three pending, Ops' none and Far's one; bob, Web's three
// and Ops' none; walt, his own one of Web's and Ops' none; wanda, acme's
// admin and a member of none of them, nothing. Shut and Old are listed to
// no one. Through the issue module on a clock a microsecond before the
// snooze's time, the same; at its time, alice and bob count it among
// Web's, and walt, whose it is not, does not.
func TestCountingTheIntakesThroughTheApp(t *testing.T) {
	w := newIssueWorld(t)
	w.makeMember(t, "walt", w.web, shared.RoleGuest)
	project := func(name, identifier string) uuid.UUID {
		return createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", name, identifier)
	}
	ops, far, shut, old := project("Ops", "OPS"), project("Far", "FAR"), project("Shut", "SHUT"), project("Old", "OLD")
	for _, p := range []uuid.UUID{ops, shut, old} {
		w.makeMember(t, "bob", p, shared.RoleMember)
		w.makeMember(t, "walt", p, shared.RoleGuest)
	}
	w.intakeOn(t)
	for _, p := range []uuid.UUID{ops, far, shut, old} {
		w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+p.String(), `{"intake_view":true}`, http.StatusOK)
	}
	submitTo := func(name string, project uuid.UUID) uuid.UUID {
		var s submission
		decodeAnswer(t, w.send(t, name, http.MethodPost, intakePath(project), `{"name":"Report"}`, http.StatusCreated), &s)
		return s.ID
	}
	for _, name := range []string{"alice", "bob", "walt"} {
		submitTo(name, w.web)
	}
	snoozed, declined, accepted := submitTo("bob", w.web), submitTo("bob", w.web), submitTo("bob", w.web)
	till := time.Now().Add(time.Hour).UTC().Truncate(time.Microsecond)
	w.triage(t, "alice", snoozed, snoozedTill(till), http.StatusOK)
	w.triage(t, "alice", declined, `{"status":"declined"}`, http.StatusOK)
	w.triage(t, "alice", accepted, `{"status":"accepted"}`, http.StatusOK)
	submitTo("alice", far)
	submitTo("bob", shut)
	submitTo("bob", old)
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+shut.String(), `{"intake_view":false}`, http.StatusOK)
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+old.String()+"/archive", `{}`, http.StatusOK)
	counted := func(webs int64) map[string][]answeredCount {
		return map[string][]answeredCount{
			"alice": {{ProjectID: w.web, Pending: webs}, {ProjectID: ops}, {ProjectID: far, Pending: 1}},
			"bob":   {{ProjectID: w.web, Pending: webs}, {ProjectID: ops}},
			"walt":  {{ProjectID: w.web, Pending: 1}, {ProjectID: ops}},
			"wanda": nil,
		}
	}

	t.Run("each reader", func(t *testing.T) {
		for name, want := range counted(3) {
			countsAre(t, name, w.reading(name), want...)
		}
	})
	t.Run("a snooze whose time has come", func(t *testing.T) {
		c := clockAt(till.Add(-time.Microsecond))
		route := newIssueRouteWithClock(t, w.pool, postgres.NewTxManager(w.pool, 2*time.Second), c)
		for name, want := range counted(3) {
			countsAre(t, name+", a microsecond before the snooze's time", w.readingOn(route, name), want...)
		}
		c.advance(time.Microsecond)
		for name, want := range counted(4) {
			countsAre(t, name+", at the snooze's time", w.readingOn(route, name), want...)
		}
	})
}
