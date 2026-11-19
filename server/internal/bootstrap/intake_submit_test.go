package bootstrap

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	notificationdomain "github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// submission is a work item a submission to an intake answered: its id,
// number and state, and the answer itself.
type submission struct {
	ID         uuid.UUID `json:"id"`
	SequenceID int       `json:"sequence_id"`
	StateID    uuid.UUID `json:"state_id"`
	raw        string
}

// intakePath is the path a work item is submitted to project's intake at;
// triagePath, the path the work item item is triaged at.
func intakePath(project uuid.UUID) string {
	return "/api/v0/projects/" + project.String() + "/intake-issues"
}
func triagePath(item uuid.UUID) string { return "/api/v0/intake-issues/" + item.String() + "/triage" }

// snoozedTill is a triage's body snoozing a work item till at; duplicateOf,
// one marking it a duplicate of target.
func snoozedTill(at time.Time) string {
	return fmt.Sprintf(`{"status":"snoozed","snoozed_till":%q}`, at.Format(time.RFC3339Nano))
}
func duplicateOf(target uuid.UUID) string {
	return fmt.Sprintf(`{"status":"duplicate","duplicate_to_id":%q}`, target)
}

// submit has the account name submit a work item with body to Web's
// intake through the API, answered 201.
func (w issueWorld) submit(t *testing.T, name, body string) submission {
	t.Helper()
	s := submission{raw: w.send(t, name, http.MethodPost, intakePath(w.web), body, http.StatusCreated)}
	decodeAnswer(t, s.raw, &s)
	return s
}

// triage has the account name triage the work item item with body through
// the API, checks the answer is want, and returns its body.
func (w issueWorld) triage(t *testing.T, name string, item uuid.UUID, body string, want int) string {
	t.Helper()
	return w.send(t, name, http.MethodPost, triagePath(item), body, want)
}

// refusesAs has the account name send method path with body through the
// API and checks it is refused with status and the refusal (refusalOf).
func (w issueWorld) refusesAs(t *testing.T, name, method, path, body string, status int, refusal string) {
	t.Helper()
	if got := refusalOf(t, []byte(w.send(t, name, method, path, body, status)), false); got != refusal {
		t.Errorf("%s %s %s as %s = %s, want %s", method, path, body, name, got, refusal)
	}
}

// intakeIs checks the answer raw, a work item in full, carries the intake
// row want, nil for null: its times compared as instants.
func intakeIs(t *testing.T, what, raw string, want *answeredIntake) {
	t.Helper()
	got := intakeIn(t, raw)
	same := got == nil && want == nil
	if got != nil && want != nil {
		same = got.Status == want.Status && (got.SnoozedTill == nil) == (want.SnoozedTill == nil) &&
			(got.SnoozedTill == nil || got.SnoozedTill.Equal(*want.SnoozedTill)) && (got.DuplicateToID == nil) == (want.DuplicateToID == nil) &&
			(got.DuplicateToID == nil || *got.DuplicateToID == *want.DuplicateToID)
	}
	if !same {
		t.Errorf("%s answers intake %+v, want %+v", what, got, want)
	}
}

// projectListIDs checks the account name's list of Web's work items is
// want, in any order.
func projectListIDs(t *testing.T, w issueWorld, name string, want ...uuid.UUID) {
	t.Helper()
	page := checkListEnvelope(t, w.send(t, name, http.MethodGet, "/api/v0/projects/"+w.web.String()+"/issues", "", http.StatusOK), false, false)
	got := make([]uuid.UUID, len(page.Data))
	for n, row := range page.Data {
		got[n] = row.ID
	}
	slices.SortFunc(got, uuid.UUID.Compare)
	want = slices.Clone(want)
	slices.SortFunc(want, uuid.UUID.Compare)
	if !slices.Equal(got, want) || page.Total != int64(len(want)) {
		t.Errorf("%s lists Web's work items %v (total %d), want %v", name, got, page.Total, want)
	}
}

// webCount is how many work items Web has, deleted or not.
func (w issueWorld) webCount(t *testing.T) int {
	t.Helper()
	var n int
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM issues WHERE project_id = $1", w.web).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// H8 through the app (M7 P2a spec 2.4, 4). With Web's intake turned on,
// alice, its admin, bob, its member, and walt, its guest, whom Web shows
// only his own work items, each submit a work item: 201, in Web's triage
// state, its intake pending. None is in Web's list, alice's or walt's, in
// bob's lists of the work items assigned to him or that he created, nor
// in a search, alone or with another work item as its source: Assigned,
// alice's, assigned to bob, and Bob's, his, are. alice's Login breaks,
// assigned to bob, mentions wanda, acme's admin and so Web's admin, and
// walt: once its fan-out has run, bob is sent its creation and his
// assignment, wanda her mention, each showing it in Triage, triage; walt,
// from whom the guest rule hides it, nothing. A body with the state, the
// parent, the cycle or the modules is refused 400 before anything is
// looked at. With the intake turned off a submission is refused 409
// issue.intake_disabled, and the work items submitted read as they were;
// with Web archived too, 409 project.archived comes first. A refused
// submission writes no work item.
func TestSubmittingToTheIntakeThroughTheApp(t *testing.T) {
	w := newIssueWorld(t)
	alice, bob, walt, wanda := w.ids["alice"], w.ids["bob"], w.ids["walt"], w.ids["wanda"]
	w.makeMember(t, "walt", w.web, shared.RoleGuest)
	w.makeMember(t, "wanda", w.web, shared.RoleAdmin)
	triage := stateID(t, w.pool, w.web, "Triage")
	assigned := w.newWorkItemIn(t, w.web, fmt.Sprintf(`{"name":"Assigned","assignee_ids":[%q]}`, bob))
	bobs := w.newWorkItemBy(t, "bob", w.web, `{"name":"Bob's"}`)
	w.intakeOn(t)
	login := w.submit(t, "alice", fmt.Sprintf(`{"name":"Login breaks","assignee_ids":[%q],"description_html":%q}`, bob, mentionHTML(wanda, walt)))
	submitted := map[string]submission{"alice": login, "bob": w.submit(t, "bob", `{"name":"Reported"}`),
		"walt": w.submit(t, "walt", `{"name":"Guessed"}`)}
	pending := &answeredIntake{Status: "pending"}

	t.Run("each role submits", func(t *testing.T) {
		for name, s := range submitted {
			if s.StateID != triage {
				t.Errorf("%s's submission answers state %s, want Web's triage state %s", name, s.StateID, triage)
			}
			intakeIs(t, name+"'s submission", s.raw, pending)
		}
	})
	t.Run("out of the lists and searches", func(t *testing.T) {
		projectListIDs(t, w, "alice", assigned, bobs)
		projectListIDs(t, w, "walt")
		memberListIDs(t, w, "alice", "bob", "assigned", "", assigned)
		memberListIDs(t, w, "alice", "bob", "created", "", bobs)
		searchHTTPIDs(t, w, "alice", "", assigned, bobs)
		searchHTTPIDs(t, w, "alice", "for=relation&issue_id="+assigned.String(), bobs)
		searchHTTPIDs(t, w, "walt", "")
	})
	t.Run("notifications", func(t *testing.T) {
		waitForFanouts(t, w.pool)
		job := fanoutOf(t, w.pool, "issue.created", login.ID)
		mentionedAt(t, job, walt) // the creation mentions walt, whom the recheck leaves out
		bobName := displayNames(t, w, bob)[0]
		shown := notificationdomain.IssueSnapshot{ID: login.ID, Name: "Login breaks", SequenceID: login.SequenceID, Identifier: "WEB",
			StateName: "Triage", StateGroup: "triage"}
		assertSent(t, w.pool, login.ID, []uuid.UUID{job.EventID}, w.sentOf(job, &alice, &alice, shown,
			sending{bob, "assigned", -1, madeIt}, sending{bob, "assigned", assignedAt(t, job, bob), assigning(&bobName, &bob)},
			sending{wanda, "mentioned", mentionedAt(t, job, wanda), notificationdomain.ChangeSnapshot{Field: "description", Verb: "mentioned"}}))
	})
	t.Run("a body with a field a submission may not have", func(t *testing.T) {
		before := w.webCount(t)
		for field, value := range map[string]string{"state_id": `"` + w.todo.String() + `"`, "parent_id": `"` + assigned.String() + `"`,
			"cycle_id": `"` + uuid.NewV7().String() + `"`, "module_ids": "[]"} {
			body := `{"name":"Mend","` + field + `":` + value + `}`
			if got := refusalOf(t, []byte(draftRequest(t, w, "bob", http.MethodPost, intakePath(w.web), body, http.StatusBadRequest)), false); got !=
				"bad_request "+field+" not_allowed" {
				t.Errorf("bob's submission of %s = %s, want bad_request %s not_allowed", body, got, field)
			}
		}
		if after := w.webCount(t); after != before {
			t.Errorf("the refused bodies wrote %d work items", after-before)
		}
	})
	t.Run("the intake turned off, then the project archived", func(t *testing.T) {
		before := w.webCount(t)
		w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"intake_view":false}`, http.StatusOK)
		for name := range submitted {
			w.refusesAs(t, name, http.MethodPost, intakePath(w.web), `{"name":"Late"}`, http.StatusConflict, "issue.intake_disabled")
		}
		for name, s := range submitted {
			raw := w.send(t, name, http.MethodGet, "/api/v0/issues/"+s.ID.String(), "", http.StatusOK)
			var read submission
			decodeAnswer(t, raw, &read)
			if read.StateID != triage {
				t.Errorf("%s reads his submission in state %s, the intake off; want Web's triage state", name, read.StateID)
			}
			intakeIs(t, name+"'s read of his submission, the intake off", raw, pending)
		}
		w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", `{}`, http.StatusOK)
		w.refusesAs(t, "alice", http.MethodPost, intakePath(w.web), `{"name":"Late"}`, http.StatusConflict, "project.archived")
		if after := w.webCount(t); after != before {
			t.Errorf("the refused submissions wrote %d work items", after-before)
		}
	})
}

// The detail's intake through the app (M7 P2a spec 2.3, 2.8), as
// TestEachDetailAnswerCarriesItsIntakeRow does not show it. bob's Snoozed,
// which alice snoozes till an hour on, reads snoozed till then; through
// the issue module wired as bootstrap wires it (newIssueRouteWithClock) on
// a clock a microsecond before that time, the same, read and read by its
// identifier; on the clock at that time, its snooze over, pending, without
// a time, its row as stored. walt, Web's guest, whom Web shows only his own
// work items, submits Guessed, which alice marks a duplicate of Target,
// hers: alice reads it naming Target, walt, from whom the guest rule hides
// Target, naming none; once alice deletes Target, she reads it naming none
// too.
func TestTheDetailsIntakeThroughTheApp(t *testing.T) {
	w := newIssueWorld(t)
	alice := w.ids["alice"]
	w.makeMember(t, "walt", w.web, shared.RoleGuest)
	target := w.newWorkItemIn(t, w.web, `{"name":"Target"}`)
	w.intakeOn(t)

	t.Run("a snooze whose time has come", func(t *testing.T) {
		s := w.submit(t, "bob", `{"name":"Snoozed"}`)
		till := time.Now().Add(time.Hour).UTC().Truncate(time.Microsecond)
		snoozed := &answeredIntake{Status: "snoozed", SnoozedTill: &till}
		intakeIs(t, "alice's snooze", w.triage(t, "alice", s.ID, snoozedTill(till), http.StatusOK), snoozed)
		paths := []string{"/api/v0/issues/" + s.ID.String(), fmt.Sprintf("/api/v0/workspaces/acme/issues/WEB-%d", s.SequenceID)}
		intakeIs(t, "alice's read, the snooze on", w.send(t, "alice", http.MethodGet, paths[0], "", http.StatusOK), snoozed)
		c := clockAt(till.Add(-time.Microsecond))
		route := newIssueRouteWithClock(t, w.pool, postgres.NewTxManager(w.pool, 2*time.Second), c)
		for _, at := range []struct {
			name string
			want *answeredIntake
		}{{"a microsecond before its time", snoozed}, {"at its time", &answeredIntake{Status: "pending"}}} {
			for _, path := range paths {
				intakeIs(t, "GET "+path+" "+at.name, route.answerWithin(t, w.contract, http.MethodGet, path, alice, "", http.StatusOK), at.want)
			}
			c.advance(time.Microsecond)
		}
		var status string
		var stored time.Time
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT status, snoozed_till FROM intake_issues WHERE issue_id = $1", s.ID).Scan(&status,
			&stored); err != nil || status != "snoozed" || !stored.Equal(till) {
			t.Errorf("the row stored %s till %s, %v; want snoozed till %s", status, stored, err, till)
		}
	})
	t.Run("a duplicate's target as its reader sees it", func(t *testing.T) {
		s := w.submit(t, "walt", `{"name":"Guessed"}`)
		named, none := &answeredIntake{Status: "duplicate", DuplicateToID: &target}, &answeredIntake{Status: "duplicate"}
		intakeIs(t, "alice's triage of Guessed", w.triage(t, "alice", s.ID, duplicateOf(target), http.StatusOK), named)
		path := "/api/v0/issues/" + s.ID.String()
		intakeIs(t, "alice's read of Guessed", w.send(t, "alice", http.MethodGet, path, "", http.StatusOK), named)
		intakeIs(t, "walt's read of Guessed", w.send(t, "walt", http.MethodGet, path, "", http.StatusOK), none)
		w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+target.String(), "", http.StatusNoContent)
		intakeIs(t, "alice's read of Guessed, Target deleted", w.send(t, "alice", http.MethodGet, path, "", http.StatusOK), none)
	})
}
