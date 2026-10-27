package bootstrap

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Missing cleaning, defaults, mentions, versions, activity or event breaks this
// real module composition, rather than a call log that omits those writes.
func TestCreateCoreKeepsEffects(t *testing.T) {
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
	body := fmt.Sprintf(`{"name":"  Core  ","description_html":%q,"label_ids":[%q]}`, html+`<script>bad()</script>`, w.bug)
	var got storedIssue
	decodeAnswer(t, route.answerWithin(t, w.contract, http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", w.ids["alice"], body, 201), &got)
	if tracked.calls.Load() != 1 || got.Name != "Core" || got.SequenceID != 1 || got.SortOrder != 65535 || !got.CreatedAt.Equal(at) || !got.UpdatedAt.Equal(at) || !slices.Equal(got.AssigneeIDs, []uuid.UUID{w.ids["bob"]}) {
		t.Fatalf("created=%+v clock=%d", got, tracked.calls.Load())
	}
	var stored, text string
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT description_html,description_stripped FROM issues WHERE id=$1", got.ID).Scan(&stored, &text); err != nil {
		t.Fatal(err)
	}
	version := savedVersion(t, w, got.ID)
	if stored != html || text != "Hello" || version.HTML != html || version.Text != "Hello" || !version.LastSavedAt.Equal(at) {
		t.Fatalf("clean body=%q text=%q version=%+v", stored, text, version)
	}
	mentions := queryIDs(t, w.pool, "SELECT mention_id FROM issue_mentions WHERE issue_id=$1 ORDER BY mention_id", got.ID)
	wantMentions := []uuid.UUID{w.ids["walt"], w.ids["bob"]}
	slices.SortFunc(wantMentions, uuid.UUID.Compare)
	if !slices.Equal(mentions, wantMentions) {
		t.Fatalf("mentions=%v want=%v", mentions, wantMentions)
	}
	subscribers := queryIDs(t, w.pool, "SELECT subscriber_id FROM issue_subscribers WHERE issue_id=$1 ORDER BY subscriber_id", got.ID)
	wantSubscribers := []uuid.UUID{w.ids["alice"], w.ids["bob"]}
	slices.SortFunc(wantSubscribers, uuid.UUID.Compare)
	if !slices.Equal(subscribers, wantSubscribers) {
		t.Fatalf("subscribers=%v want=%v", subscribers, wantSubscribers)
	}
	activity := "issue_activities made by alice made, changed by alice made, of alice"
	want := []string{activity, activity, "issue_assignees made by alice made, changed by alice made, of bob", "issue_description_versions made by alice made, changed by alice made, of alice", "issue_labels made by alice made, changed by alice made"}
	for _, id := range wantMentions {
		name := "bob"
		if id == w.ids["walt"] {
			name = "walt"
		}
		want = append(want, "issue_mentions made by alice made, changed by alice made, of "+name)
	}
	want = append(want, "issue_subscribers made by alice made, changed by alice made, of alice", "issue_subscribers made by alice made, changed by alice made, of bob", "issues made by alice made, changed by alice made")
	slices.Sort(want)
	if rows := w.workItemStamps(t, got.ID); !slices.Equal(rows, want) {
		t.Fatalf("rows=%q want=%q", rows, want)
	}
	if len(events) != 1 || events[0].Type != "issue.created" || events[0].Entity.ID != got.ID || !events[0].OccurredAt.Equal(at) || events[0].ActorID == nil || *events[0].ActorID != w.ids["alice"] || issueEventCount(t, w) != 1 {
		t.Fatalf("events=%+v", events)
	}
}
