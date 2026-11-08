package bootstrap

import (
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// joiningPayload is a draft's payload named Fix, in cycle unless it is
// nil, and in modules, in their order, when any are given.
func joiningPayload(cycle *uuid.UUID, modules ...uuid.UUID) string {
	payload := `{"name":"Fix"`
	if cycle != nil {
		payload += `,"cycle_id":"` + cycle.String() + `"`
	}
	if len(modules) > 0 {
		payload += `,"module_ids":` + idArray(modules...)
	}
	return payload + "}"
}

// draftIn is alice's draft in Web with payload, as saved.
func (w issueWorld) draftIn(t *testing.T, payload string) draftHTTP {
	t.Helper()
	return createDraftHTTP(t, w, "alice", `{"project_id":"`+w.web.String()+`","payload":`+payload+`}`)
}

// keptDraft checks alice's draft d is as saved with payload: read with
// that payload and its last change, and listed.
func (w issueWorld) keptDraft(t *testing.T, d draftHTTP, payload string) {
	t.Helper()
	var kept draftHTTP
	decodeAnswer(t, draftRequest(t, w, "alice", http.MethodGet, draftPath(d.ID), "", http.StatusOK), &kept)
	var want map[string]any
	decodeAnswer(t, payload, &want)
	if !reflect.DeepEqual(kept.Payload, want) || !kept.UpdatedAt.Equal(d.UpdatedAt) {
		t.Errorf("draft %s is %+v, want as saved with %s at %s", d.ID, kept, payload, d.UpdatedAt)
	}
	var page struct {
		Data []draftHTTP `json:"data"`
	}
	decodeAnswer(t, draftRequest(t, w, "alice", http.MethodGet, draftsPath+"?limit=100", "", http.StatusOK), &page)
	if !slices.ContainsFunc(page.Data, func(l draftHTTP) bool { return l.ID == d.ID }) {
		t.Errorf("draft %s is not listed", d.ID)
	}
}

// opsTargets are a cycle and a module alice creates in Ops, a project of
// acme's she creates, its cycles and modules turned on.
func (w issueWorld) opsTargets(t *testing.T) (cycle, module uuid.UUID) {
	t.Helper()
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	w.turnCyclesOn(t, ops)
	w.turnModulesOn(t, ops)
	var c cycleAnswer
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+ops.String()+"/cycles", `{"name":"Theirs"}`, http.StatusCreated), &c)
	var m moduleAnswer
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+ops.String()+"/modules", `{"name":"Theirs"}`, http.StatusCreated), &m)
	return c.ID, m.ID
}

// refusedPublication has alice publish her draft d, saved with payload,
// through publish, checks it is refused with status and the refusal want
// (refusalOf: the code, then each field's), that nothing is written and
// that the draft is kept as saved; it returns the refusal.
func (w issueWorld) refusedPublication(t *testing.T, publish func(path string, status int) string, d draftHTTP, payload string, status int,
	want string) string {
	t.Helper()
	before, written := publishSnapshot(t, w), w.cycleWritten(t)
	answer := publish(publishPath(d.ID), status)
	if got := refusalOf(t, []byte(answer), false); got != want {
		t.Errorf("publishing %s = %s, want %s", payload, answer, want)
	}
	if after := publishSnapshot(t, w); !reflect.DeepEqual(after, before) || w.cycleWritten(t) != written {
		t.Errorf("the refused publication of %s wrote", payload)
	}
	w.keptDraft(t, d, payload)
	return answer
}

// Saving a draft checks only the shape of its cycle and its modules, as of
// its other fields; its publication applies a creation's rules (M6 P3a
// spec 2.3, M6 design 3.3): with Web's cycles and modules turned off, alice
// saves, and replaces a draft's payload with, payloads naming an ended
// cycle, an archived one, a deleted one, Ops' and none at all, an archived
// module, a deleted one, Ops' and none at all, one named twice, and more
// than a hundred; each is saved and answered as given, and no work item,
// membership or activity is written.
func TestDraftsSaveAnyCycleAndModules(t *testing.T) {
	w := newIssueWorld(t)
	w.turnCyclesOn(t, w.web)
	w.turnModulesOn(t, w.web)
	day := func(days int) string { return shanghaiDay(t, time.Now(), days) }
	ended := w.newCycle(t, "alice", datesBody("Ended", day(-30), day(-25)))
	shelved := w.newCycle(t, "alice", datesBody("Shelved", day(-24), day(-20)))
	w.send(t, "alice", http.MethodPost, "/api/v0/cycles/"+shelved.String()+"/archive", "", http.StatusOK)
	gone := w.newCycle(t, "alice", `{"name":"Gone"}`)
	w.send(t, "alice", http.MethodDelete, "/api/v0/cycles/"+gone.String(), "", http.StatusNoContent)
	archived := w.newModule(t, `{"name":"Shelved","status":"completed"}`)
	w.send(t, "alice", http.MethodPost, "/api/v0/modules/"+archived.String()+"/archive", "", http.StatusOK)
	dropped := w.newModule(t, `{"name":"Gone"}`)
	w.send(t, "alice", http.MethodDelete, "/api/v0/modules/"+dropped.String(), "", http.StatusNoContent)
	theirCycle, theirModule := w.opsTargets(t)
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"cycle_view":false,"module_view":false}`, http.StatusOK)
	many := make([]uuid.UUID, 101)
	for i := range many {
		many[i] = uuid.NewV7()
	}
	none := uuid.NewV7()
	before := w.cycleWritten(t)
	replaced := w.draftIn(t, `{}`)
	for _, payload := range []string{
		joiningPayload(&ended), joiningPayload(&shelved), joiningPayload(&gone), joiningPayload(&theirCycle), joiningPayload(&none),
		joiningPayload(nil, archived, dropped, theirModule, none), joiningPayload(&ended, archived, none, archived), joiningPayload(nil, many...),
	} {
		w.keptDraft(t, w.draftIn(t, payload), payload)
		var changed draftHTTP
		decodeAnswer(t, draftRequest(t, w, "alice", http.MethodPatch, draftPath(replaced.ID), `{"payload":`+payload+`}`, http.StatusOK), &changed)
		w.keptDraft(t, changed, payload)
	}
	if after := w.cycleWritten(t); after != before {
		t.Errorf("saving drafts wrote: %s, was %s", after, before)
	}
}

// Publishing a draft creates its work item in the draft's cycle and
// modules as a creation does (M6 P3a spec 2.3, 2.4, R5): alice's draft
// names Sprint, then Billing and Auth in that order; published, the work
// item is answered in Sprint and in Auth and Billing, in id order; its
// cycle's and modules' rows and its activity, the creation's then
// Sprint's and each module's joining, are hers at the publication's
// moment, the moment the draft is consumed; issue.created ends with the
// cycle's and the modules' changes.
func TestDraftPublishJoinsTheCycleAndTheModules(t *testing.T) {
	w := newIssueWorld(t)
	w.turnCyclesOn(t, w.web)
	w.turnModulesOn(t, w.web)
	at := time.Now().Truncate(time.Microsecond)
	sprint := w.newCycle(t, "alice", datesBody("Sprint", shanghaiDay(t, at, -1), shanghaiDay(t, at, 1)))
	auth, billing := w.newModule(t, `{"name":"Auth"}`), w.newModule(t, `{"name":"Billing"}`)
	n := cycleNames{w: w, cycles: map[uuid.UUID]string{sprint: "Sprint", auth: "Auth", billing: "Billing"}, moments: map[string]string{}}
	n.moment("published", at)
	d := w.draftIn(t, joiningPayload(&sprint, billing, auth))
	var events []shared.Event
	var failure error
	route := issueEventRoute(t, w, clockAt(at), &events, &failure)
	var item joinedItem
	decodeAnswer(t, route.answerWithin(t, w.contract, http.MethodPost, publishPath(d.ID), w.ids["alice"], "", http.StatusCreated), &item)
	if item.CycleID == nil || *item.CycleID != sprint || !slices.Equal(item.ModuleIDs, []uuid.UUID{auth, billing}) || !item.CreatedAt.Equal(at) {
		t.Fatalf("published in %v and %v at %s, want in Sprint, Auth and Billing at %s", item.CycleID, item.ModuleIDs, item.CreatedAt, at)
	}
	wantRows(t, "cycle rows", n.cycleRows(t, item.ID), []string{"Sprint made by alice at published"})
	wantRows(t, "module rows", n.moduleRows(t, item.ID), []string{"Auth made by alice at published", "Billing made by alice at published"})
	wantRows(t, "activity", n.cycleActivity(t, item.ID), []string{` created "-"→"-" -→- "created the work item" by alice at published`,
		`cycles created ""→"Sprint" -→Sprint "added cycle " by alice at published`,
		`modules created ""→"Auth" -→Auth "added module " by alice at published`,
		`modules created ""→"Billing" -→Billing "added module " by alice at published`})
	draftRequest(t, w, "alice", http.MethodGet, draftPath(d.ID), "", http.StatusNotFound)
	var consumed time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at FROM draft_issues WHERE id = $1", d.ID).Scan(&consumed); err != nil || !consumed.Equal(at) {
		t.Errorf("the draft consumed at %s (%v), want at the publication, %s", consumed, err, at)
	}
	if len(events) != 1 || events[0].Type != "issue.created" {
		t.Fatalf("events %+v, want the creation", events)
	}
	changes := events[0].Changes
	want := []shared.Change{{Field: "cycles", New: eventValue("Sprint"), NewID: &sprint},
		{Field: "modules", New: eventValue("Auth"), NewID: &auth}, {Field: "modules", New: eventValue("Billing"), NewID: &billing}}
	if len(changes) < len(want) || !reflect.DeepEqual(changes[len(changes)-len(want):], want) {
		t.Errorf("issue.created's changes %+v, want them ending %+v", changes, want)
	}
}

// Publishing a draft refuses its cycle and modules by a creation's rules,
// as they are when it is published (M6 P3a spec R2, M6 design 3.3): alice
// saves drafts in an ended cycle, in Shelved and in Gone, which she then
// archives and deletes, in Ops' cycle, and in Auth with a module she then
// archives, with one she then deletes, and with Ops' module; each
// publication is refused with the answer a creation of its payload gets.
// Changed to Sprint and Auth, each is published. A draft in Sprint, open
// today, is refused as ended when published on a day after Sprint's end
// in Shanghai; with Web's cycles turned off, then its modules, drafts in
// Sprint and in Auth are refused; turned on again, both are published.
// Each refusal writes nothing and keeps the draft as saved, and listed.
func TestDraftPublishIsRefusedAsACreationIs(t *testing.T) {
	w := newIssueWorld(t)
	w.turnCyclesOn(t, w.web)
	w.turnModulesOn(t, w.web)
	now := time.Now()
	day := func(days int) string { return shanghaiDay(t, now, days) }
	sprint := w.newCycle(t, "alice", datesBody("Sprint", day(-1), day(1)))
	ended := w.newCycle(t, "alice", datesBody("Ended", day(-30), day(-25)))
	shelved := w.newCycle(t, "alice", datesBody("Shelved", day(-24), day(-20)))
	gone := w.newCycle(t, "alice", `{"name":"Gone"}`)
	auth := w.newModule(t, `{"name":"Auth"}`)
	archived := w.newModule(t, `{"name":"Shelved","status":"completed"}`)
	dropped := w.newModule(t, `{"name":"Gone"}`)
	theirCycle, theirModule := w.opsTargets(t)
	unknown := "validation_failed cycle_id not_allowed"
	refused := []struct {
		payload string
		status  int
		want    string
	}{
		{joiningPayload(&ended), http.StatusConflict, "planning.cycle_completed"},
		{joiningPayload(&shelved), http.StatusConflict, "planning.archived"},
		{joiningPayload(&gone), http.StatusUnprocessableEntity, unknown},
		{joiningPayload(&theirCycle), http.StatusUnprocessableEntity, unknown},
		{joiningPayload(nil, auth, archived), http.StatusConflict, "planning.archived"},
		{joiningPayload(nil, auth, dropped), http.StatusUnprocessableEntity, "validation_failed module_ids[1] not_allowed"},
		{joiningPayload(nil, theirModule, auth), http.StatusUnprocessableEntity, "validation_failed module_ids[0] not_allowed"},
	}
	drafts := make([]draftHTTP, len(refused))
	for i, r := range refused {
		drafts[i] = w.draftIn(t, r.payload)
	}
	w.send(t, "alice", http.MethodPost, "/api/v0/cycles/"+shelved.String()+"/archive", "", http.StatusOK)
	w.send(t, "alice", http.MethodDelete, "/api/v0/cycles/"+gone.String(), "", http.StatusNoContent)
	w.send(t, "alice", http.MethodPost, "/api/v0/modules/"+archived.String()+"/archive", "", http.StatusOK)
	w.send(t, "alice", http.MethodDelete, "/api/v0/modules/"+dropped.String(), "", http.StatusNoContent)
	alice := func(path string, status int) string { return w.send(t, "alice", http.MethodPost, path, "", status) }
	create := "/api/v0/projects/" + w.web.String() + "/issues"
	for i, r := range refused {
		answer := w.refusedPublication(t, alice, drafts[i], r.payload, r.status, r.want)
		if creation := w.send(t, "alice", http.MethodPost, create, r.payload, r.status); !reflect.DeepEqual(problemObject(t, answer), problemObject(t, creation)) {
			t.Errorf("publishing %s = %s, a creation %s", r.payload, answer, creation)
		}
	}
	published := func(d draftHTTP, cycle uuid.UUID, modules ...uuid.UUID) {
		t.Helper()
		var item joinedItem
		decodeAnswer(t, alice(publishPath(d.ID), http.StatusCreated), &item)
		if item.CycleID == nil || *item.CycleID != cycle || !slices.Equal(item.ModuleIDs, modules) {
			t.Errorf("draft %s published in %v and %v, want in %s and %v", d.ID, item.CycleID, item.ModuleIDs, cycle, modules)
		}
		draftRequest(t, w, "alice", http.MethodGet, draftPath(d.ID), "", http.StatusNotFound)
	}
	for _, d := range drafts {
		draftRequest(t, w, "alice", http.MethodPatch, draftPath(d.ID), `{"payload":`+joiningPayload(&sprint, auth)+`}`, http.StatusOK)
		published(d, sprint, auth)
	}

	inSprint, inAuth := w.draftIn(t, joiningPayload(&sprint)), w.draftIn(t, joiningPayload(&sprint, auth))
	later := newIssueRouteWithClock(t, w.pool, postgres.NewTxManager(w.pool, 2*time.Second), clockAt(now.AddDate(0, 0, 7)))
	w.refusedPublication(t, func(path string, status int) string {
		return later.answerWithin(t, w.contract, http.MethodPost, path, w.ids["alice"], "", status)
	}, inSprint, joiningPayload(&sprint), http.StatusConflict, "planning.cycle_completed")
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"cycle_view":false}`, http.StatusOK)
	w.refusedPublication(t, alice, inSprint, joiningPayload(&sprint), http.StatusConflict, "planning.cycles_disabled")
	w.turnCyclesOn(t, w.web)
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"module_view":false}`, http.StatusOK)
	w.refusedPublication(t, alice, inAuth, joiningPayload(&sprint, auth), http.StatusConflict, "planning.modules_disabled")
	w.turnModulesOn(t, w.web)
	published(inSprint, sprint)
	published(inAuth, sprint, auth)
}
