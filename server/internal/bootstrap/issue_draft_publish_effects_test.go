package bootstrap

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Losing any creation side effect or giving draft deletion a second clock
// fails through stored rows and the real dispatcher/River transaction bridge.
func TestDraftPublishKeepsCreationEffects(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE projects SET default_assignee_id=$2 WHERE id=$1", w.web, w.ids["bob"]); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2044, 5, 6, 7, 8, 9, 123456000, time.UTC)
	tracked := &eventCountingClock{base: clockAt(at)}
	var events []shared.Event
	var failure error
	route := issueEventRoute(t, w, tracked, &events, &failure)
	html := mentionHTML(w.ids["walt"], w.ids["bob"], w.ids["bob"], w.ids["olga"])
	d := createDraftHTTP(t, w, "alice", fmt.Sprintf(`{"project_id":%q,"payload":{"name":"  Published  ","description_html":%q,"label_ids":[%q]}}`, w.web, html+`<script>bad()</script>`, w.bug))
	var got storedIssue
	decodeAnswer(t, route.answerWithin(t, w.contract, "POST", publishPath(d.ID), w.ids["alice"], "", 201), &got)
	if tracked.calls.Load() != 1 || got.Name != "Published" || got.SequenceID != 1 || got.SortOrder != 65535 || !got.CreatedAt.Equal(at) || !got.UpdatedAt.Equal(at) || !slices.Equal(got.AssigneeIDs, []uuid.UUID{w.ids["bob"]}) {
		t.Fatalf("issue=%+v clock=%d", got, tracked.calls.Load())
	}
	var stored, text string
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT description_html,description_stripped FROM issues WHERE id=$1", got.ID).Scan(&stored, &text); err != nil {
		t.Fatal(err)
	}
	version := savedVersion(t, w, got.ID)
	if stored != html || text != "Hello" || version.HTML != html || version.Text != "Hello" || !version.LastSavedAt.Equal(at) {
		t.Fatal(stored, text, version)
	}
	mentions := queryIDs(t, w.pool, "SELECT mention_id FROM issue_mentions WHERE issue_id=$1 ORDER BY mention_id", got.ID)
	wantMentions := []uuid.UUID{w.ids["walt"], w.ids["bob"]}
	slices.SortFunc(wantMentions, uuid.UUID.Compare)
	subscribers := queryIDs(t, w.pool, "SELECT subscriber_id FROM issue_subscribers WHERE issue_id=$1 ORDER BY subscriber_id", got.ID)
	wantSubscribers := []uuid.UUID{w.ids["alice"], w.ids["bob"]}
	slices.SortFunc(wantSubscribers, uuid.UUID.Compare)
	if !slices.Equal(mentions, wantMentions) || !slices.Equal(subscribers, wantSubscribers) {
		t.Fatal(mentions, subscribers)
	}
	activity := "issue_activities made by alice made, changed by alice made, of alice"
	want := []string{activity, activity, "issue_assignees made by alice made, changed by alice made, of bob", "issue_description_versions made by alice made, changed by alice made, of alice", "issue_labels made by alice made, changed by alice made", "issue_mentions made by alice made, changed by alice made, of bob", "issue_mentions made by alice made, changed by alice made, of walt", "issue_subscribers made by alice made, changed by alice made, of alice", "issue_subscribers made by alice made, changed by alice made, of bob", "issues made by alice made, changed by alice made"}
	slices.Sort(want)
	if rows := w.workItemStamps(t, got.ID); !slices.Equal(rows, want) {
		t.Fatalf("rows=%q want=%q", rows, want)
	}
	var deleted, updated time.Time
	var by uuid.UUID
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at,updated_at,updated_by_id FROM draft_issues WHERE id=$1", d.ID).Scan(&deleted, &updated, &by); err != nil {
		t.Fatal(err)
	}
	if !deleted.Equal(at) || !updated.Equal(at) || by != w.ids["alice"] {
		t.Fatal(deleted, updated, by)
	}
	if len(events) != 1 || events[0].Type != "issue.created" || events[0].Entity.ID != got.ID || !events[0].OccurredAt.Equal(at) || events[0].ActorID == nil || *events[0].ActorID != w.ids["alice"] || issueEventCount(t, w) != 1 {
		t.Fatal(events)
	}
}

// The permission decision precedes archived/payload checks. Admin plus project
// guest follows create's elevated grant while retaining OwnOnly for parents.
func TestDraftPublishRefusalOrder(t *testing.T) {
	w := newIssueWorld(t)
	noProject := createDraftHTTP(t, w, "alice", `{"payload":{}}`)
	if raw := draftRequest(t, w, "alice", "POST", publishPath(noProject.ID), "", 422); oneError(t, []byte(raw)) != "project_id required" {
		t.Fatal(raw)
	}
	guest := createDraftHTTP(t, w, "walt", `{"project_id":"`+w.web.String()+`","payload":{}}`)
	valid := createDraftHTTP(t, w, "alice", `{"project_id":"`+w.web.String()+`","payload":{}}`)
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE projects SET archived_at=now() WHERE id=$1", w.web); err != nil {
		t.Fatal(err)
	}
	before := publishSnapshot(t, w)
	if raw := draftRequest(t, w, "walt", "POST", publishPath(guest.ID), "", 403); !strings.Contains(raw, `"code":"forbidden"`) {
		t.Fatal(raw)
	}
	if raw := draftRequest(t, w, "alice", "POST", publishPath(valid.ID), "", 409); !strings.Contains(raw, `"code":"project.archived"`) {
		t.Fatal(raw)
	}
	if after := publishSnapshot(t, w); !reflect.DeepEqual(before, after) {
		for table, rows := range before {
			if after[table] != rows {
				t.Errorf("refusals changed %s: before=%s after=%s", table, rows, after[table])
			}
		}
		t.Fatal("refusals wrote rows")
	}
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE projects SET archived_at=NULL WHERE id=$1", w.web); err != nil {
		t.Fatal(err)
	}
	parent := w.newWorkItem(t)
	w.adminGuest(t)
	d := createDraftHTTP(t, w, "walt", `{"project_id":"`+w.web.String()+`","payload":{"name":"Own"}}`)
	var own storedIssue
	ownRaw := draftRequest(t, w, "walt", "POST", publishPath(d.ID), "", 201)
	decodeAnswer(t, ownRaw, &own)
	if own.ParentID != nil || subItems(t, ownRaw) != 0 {
		t.Fatal(own)
	}
	child := createDraftHTTP(t, w, "walt", fmt.Sprintf(`{"project_id":%q,"payload":{"name":"Own child","parent_id":%q}}`, w.web, own.ID))
	childRaw := draftRequest(t, w, "walt", "POST", publishPath(child.ID), "", 201)
	var ownChild storedIssue
	decodeAnswer(t, childRaw, &ownChild)
	if ownChild.ParentID == nil || *ownChild.ParentID != own.ID || subItems(t, childRaw) != 0 {
		t.Fatal(childRaw)
	}
	hidden := createDraftHTTP(t, w, "walt", fmt.Sprintf(`{"project_id":%q,"payload":{"name":"Hidden parent","parent_id":%q}}`, w.web, parent))
	raw := draftRequest(t, w, "walt", "POST", publishPath(hidden.ID), "", 422)
	if oneError(t, []byte(raw)) != "parent_id not_allowed" {
		t.Fatal(raw)
	}
	// Existing references may disappear after save; semantic validation is current.
	for _, which := range []string{"state", "parent"} {
		body := fmt.Sprintf(`{"name":"Stale","parent_id":%q}`, parent)
		if which == "state" {
			body = fmt.Sprintf(`{"name":"Stale","state_id":%q}`, w.done)
		}
		draft := createDraftHTTP(t, w, "alice", fmt.Sprintf(`{"project_id":%q,"payload":%s}`, w.web, body))
		if which == "state" {
			w.send(t, "alice", "POST", "/api/v0/projects/"+w.web.String()+"/states", `{"name":"Spare completed","color":"#0EA5E9","group":"completed"}`, 201)
			w.send(t, "alice", "DELETE", "/api/v0/states/"+w.done.String(), "", 204)
		} else {
			w.send(t, "alice", "DELETE", "/api/v0/issues/"+parent.String(), "", 204)
		}
		got := draftRequest(t, w, "alice", "POST", publishPath(draft.ID), "", 422)
		if oneError(t, []byte(got)) != which+"_id not_allowed" {
			t.Fatal(got)
		}
	}
}
