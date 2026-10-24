package bootstrap

import (
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

type answeredActivity struct {
	ID            uuid.UUID  `json:"id"`
	IssueID       uuid.UUID  `json:"issue_id"`
	ProjectID     uuid.UUID  `json:"project_id"`
	Verb          string     `json:"verb"`
	Field         *string    `json:"field"`
	OldValue      *string    `json:"old_value"`
	NewValue      *string    `json:"new_value"`
	OldIdentifier *uuid.UUID `json:"old_identifier"`
	NewIdentifier *uuid.UUID `json:"new_identifier"`
	ActorID       *uuid.UUID `json:"actor_id"`
	CreatedAt     time.Time  `json:"created_at"`
}
type answeredActivityPage struct {
	Data []answeredActivity `json:"data"`
	Next *string            `json:"next_cursor"`
}

func activitiesOf(id uuid.UUID) string { return "/api/v0/issues/" + id.String() + "/activities" }
func activityIDs(rows []answeredActivity) []uuid.UUID {
	ids := make([]uuid.UUID, len(rows))
	for n, r := range rows {
		ids[n] = r.ID
	}
	return ids
}
func activityPage(t *testing.T, w issueWorld, actor, path string) answeredActivityPage {
	t.Helper()
	var page answeredActivityPage
	decodeAnswer(t, w.send(t, actor, http.MethodGet, path, "", 200), &page)
	return page
}

func TestActivityReferencesGuest(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	target := w.newWorkItemBy(t, "walt", w.web, `{"name":"Own target"}`)
	owned := w.newWorkItemBy(t, "walt", w.web, `{"name":"Own reference"}`)
	hidden := w.newWorkItemIn(t, w.web, `{"name":"Hidden parent"}`)
	path := "/api/v0/issues/" + target.String()
	// Both parent endpoints have to pass, including a hidden old parent number.
	for _, parent := range []uuid.UUID{hidden, owned} {
		w.send(t, "alice", http.MethodPatch, path, fmt.Sprintf(`{"parent_id":%q}`, parent), 200)
	}
	for _, kind := range []string{"relates_to", "duplicate", "blocked_by", "blocking"} {
		hidden := w.newWorkItemIn(t, w.web, `{"name":"Hidden relation"}`)
		w.send(t, "alice", http.MethodPost, path+"/relations", fmt.Sprintf(`{"relation_type":%q,"issue_ids":[%q]}`, kind, hidden), 201)
	}
	full := activityPage(t, w, "alice", activitiesOf(target))
	if len(full.Data) != 7 {
		t.Fatal("fixture", full)
	}
	guest := activityPage(t, w, "walt", activitiesOf(target))
	if len(guest.Data) != 1 || guest.Data[0].Verb != "created" {
		t.Fatal("hidden old/new references leaked", guest)
	}
	raw := w.send(t, "walt", http.MethodGet, activitiesOf(target), "", 200)
	if strings.Contains(raw, hidden.String()) || strings.Contains(raw, "WEB-3") {
		t.Fatal("hidden parent identifier or old number leaked", raw)
	}
	// Historical directory never grants normal access to another creator's target.
	var missing, refused map[string]any
	decodeAnswer(t, w.send(t, "walt", http.MethodGet, activitiesOf(uuid.NewV7()), "", 404), &missing)
	decodeAnswer(t, w.send(t, "walt", http.MethodGet, activitiesOf(hidden), "", 404), &refused)
	if !reflect.DeepEqual(missing, refused) {
		t.Fatal("hidden target distinguishable", missing, refused)
	}

	w.send(t, "alice", http.MethodPatch, path, fmt.Sprintf(`{"state_id":%q}`, w.done), 200)
	w.send(t, "alice", http.MethodPost, path+"/archive", "", 200)
	w.send(t, "walt", http.MethodGet, activitiesOf(target), "", 200)
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", "", 200)
	w.send(t, "walt", http.MethodGet, activitiesOf(target), "", 200)
}

func TestActivityReferencesProjectDeletion(t *testing.T) {
	w := newIssueWorld(t)
	target := w.newWorkItemIn(t, w.web, `{"name":"Target"}`)
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	w.makeMember(t, "bob", ops, 15)
	reference := w.newWorkItemIn(t, ops, `{"name":"History"}`)
	refs := []uuid.UUID{reference}
	for n, kind := range []string{"relates_to", "duplicate", "blocked_by", "blocking"} {
		ref := reference
		if n > 0 {
			ref = w.newWorkItemIn(t, ops, `{"name":"History relation"}`)
			refs = append(refs, ref)
		}
		w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+target.String()+"/relations", fmt.Sprintf(`{"relation_type":%q,"issue_ids":[%q]}`, kind, ref), 201)
	}
	before := activityPage(t, w, "bob", activitiesOf(target))
	if len(before.Data) != 5 {
		t.Fatal("reference fixture", before)
	}
	// Exercise current synchronous project deletion orchestration, not a future job.
	w.send(t, "alice", http.MethodDelete, "/api/v0/projects/"+ops.String(), "", 204)
	after := activityPage(t, w, "bob", activitiesOf(target))
	if !reflect.DeepEqual(after, before) {
		t.Fatal("soft-deleted reference/project lost history", before, after)
	}
	if w.deletedAt(t, "issues", reference).IsZero() {
		t.Fatal("project deletion did not cascade")
	}
	var active bool
	var deleted *time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT is_active,deleted_at FROM project_members WHERE project_id=$1 AND member_id=$2", ops, w.ids["bob"]).Scan(&active, &deleted); err != nil || !active || deleted == nil {
		t.Fatal("historical member fixture", active, deleted, err)
	}
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2", ops, w.ids["bob"]); err != nil {
		t.Fatal(err)
	}
	after = activityPage(t, w, "bob", activitiesOf(target))
	if len(after.Data) != 1 || after.Data[0].Verb != "created" {
		t.Fatal("inactive historical membership leaked", after)
	}
	// A valid cursor from a broader reader cannot widen this request's current directory.
	cursor := activityPage(t, w, "alice", activitiesOf(target)+"?limit=1").Next
	if cursor == nil {
		t.Fatal("missing broader-reader cursor")
	}
	after = activityPage(t, w, "bob", activitiesOf(target)+"?limit=1&cursor="+*cursor)
	if len(after.Data) != 0 || after.Next != nil {
		t.Fatal("cursor granted reference access", after)
	}
	// Physical cleanup removes the reference even for the still-active historical member.
	if _, err := w.pool.Exec(pgtest.Soon(t), "DELETE FROM issues WHERE id=ANY($1::uuid[])", refs); err != nil {
		t.Fatal(err)
	}
	after = activityPage(t, w, "alice", activitiesOf(target))
	if len(after.Data) != 1 {
		t.Fatal("physically absent reference leaked", after)
	}
	if !slices.Equal(activityIDs(after.Data), activityIDs(before.Data[:1])) {
		t.Fatal(after)
	}
}
