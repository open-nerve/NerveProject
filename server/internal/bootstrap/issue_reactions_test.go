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

type answeredReaction struct {
	ID        uuid.UUID `json:"id"`
	IssueID   uuid.UUID `json:"issue_id"`
	ActorID   uuid.UUID `json:"actor_id"`
	Reaction  string    `json:"reaction"`
	CreatedAt time.Time `json:"created_at"`
}

func reactionsOf(id uuid.UUID) string { return "/api/v0/issues/" + id.String() + "/reactions" }
func reactionSnapshot(t *testing.T, w issueWorld, id uuid.UUID, table string) string {
	t.Helper()
	var s string
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY id),'[]'::jsonb)::text FROM "+table+" r WHERE issue_id=$1", id).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}
func TestIssueReactions(t *testing.T) {
	w := newIssueWorld(t)
	id := w.newWorkItemIn(t, w.web, w.fullIssue())
	activities, subscribers := reactionSnapshot(t, w, id, "issue_activities"), reactionSnapshot(t, w, id, "issue_subscribers")
	if len(queryIDs(t, w.pool, "SELECT id FROM issue_activities WHERE issue_id=$1", id)) < 2 {
		t.Fatal("need the complete issue's initial activity rows")
	}
	var empty struct{ Data []answeredReaction }
	decodeAnswer(t, w.send(t, "alice", http.MethodGet, reactionsOf(id), "", 200), &empty)
	if empty.Data == nil || len(empty.Data) != 0 {
		t.Fatal(empty)
	}
	var first answeredReaction
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, reactionsOf(id), `{"reaction":"10084-65039"}`, 201), &first)
	if first.ID == (uuid.UUID{}) || first.ActorID != w.ids["alice"] || first.IssueID != id || first.Reaction != "10084-65039" {
		t.Fatal("wrong reaction", first)
	}
	var at time.Time
	var by uuid.UUID
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT updated_at,updated_by_id FROM issues WHERE id=$1", id).Scan(&at, &by); err != nil || !at.Equal(first.CreatedAt) || by != first.ActorID {
		t.Fatal("audit", at, by, err)
	}
	w.send(t, "alice", http.MethodPost, reactionsOf(id), `{"reaction":"10084-65039"}`, 409)
	w.send(t, "bob", http.MethodDelete, reactionsOf(id)+"/10084-65039", "", 404)
	var other answeredReaction
	decodeAnswer(t, w.send(t, "bob", http.MethodPost, reactionsOf(id), `{"reaction":"10084-65039"}`, 201), &other)
	w.send(t, "alice", http.MethodDelete, reactionsOf(id)+"/10084-65039", "", 204)
	var deletedAt time.Time
	var deletedBy uuid.UUID
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT updated_at,updated_by_id FROM issues WHERE id=$1", id).Scan(&deletedAt, &deletedBy); err != nil || deletedBy != w.ids["alice"] {
		t.Fatal(err, deletedBy)
	}
	var n int
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM issue_reactions WHERE id=$1 AND deleted_at=$2 AND updated_at=$2 AND updated_by_id=$3", first.ID, deletedAt, deletedBy).Scan(&n); err != nil || n != 1 {
		t.Fatal("delete audit", n, err)
	}
	var again answeredReaction
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, reactionsOf(id), `{"reaction":"10084-65039"}`, 201), &again)
	if again.ID == first.ID {
		t.Fatal("reused deleted id")
	}
	var page struct{ Data []answeredReaction }
	decodeAnswer(t, w.send(t, "alice", http.MethodGet, reactionsOf(id), "", 200), &page)
	if len(page.Data) != 2 || page.Data[0].ID != other.ID || page.Data[1].ID != again.ID {
		t.Fatal(page)
	}
	// A previously unsubscribed member's reaction must not create a subscription.
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	w.send(t, "walt", http.MethodPost, reactionsOf(id), `{"reaction":"128077"}`, 201)
	if reactionSnapshot(t, w, id, "issue_subscribers") != subscribers {
		t.Fatal("reaction subscribed its actor")
	}
	w.send(t, "walt", http.MethodDelete, reactionsOf(id)+"/128077", "", 204)
	if reactionSnapshot(t, w, id, "issue_activities") != activities || reactionSnapshot(t, w, id, "issue_subscribers") != subscribers {
		t.Fatal("reaction changed activities or subscriptions")
	}
	for _, value := range []string{"", "-1", "12345678", "1--2"} {
		req := newRequest(t, http.MethodPost, w.base+reactionsOf(id), w.tokens["alice"], []byte(fmt.Sprintf(`{"reaction":%q}`, value)))
		res, body := send(t, req)
		w.contract.CheckResponse(t, req, res)
		if res.StatusCode != 422 {
			t.Fatal(res.StatusCode, string(body))
		}
	}
}
func TestIssueReactionsListOrdersAndFilters(t *testing.T) {
	w := newIssueWorld(t)
	id, neighbor := w.newWorkItem(t), w.newWorkItem(t)
	ids := []uuid.UUID{uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()}
	at := time.Now().UTC().Truncate(time.Microsecond)
	for _, n := range []int{3, 2, 1, 0} {
		target := id
		if n == 2 {
			target = neighbor
		}
		if _, err := w.pool.Exec(pgtest.Soon(t), "INSERT INTO issue_reactions(id,workspace_id,project_id,issue_id,actor_id,reaction,created_at) VALUES($1,$2,$3,$4,$5,$6,$7)", ids[n], w.acme, w.web, target, w.ids["alice"], fmt.Sprint(n), at); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issue_reactions SET deleted_at=$2 WHERE id=$1", ids[3], at); err != nil {
		t.Fatal(err)
	}
	var page struct{ Data []answeredReaction }
	decodeAnswer(t, w.send(t, "alice", http.MethodGet, reactionsOf(id), "", 200), &page)
	var got []uuid.UUID
	for _, r := range page.Data {
		got = append(got, r.ID)
	}
	if !slices.Equal(got, ids[:2]) {
		t.Fatal(got, ids[:2])
	}
}

func (w issueWorld) newReactionPath(t *testing.T) string {
	t.Helper()
	id := w.newWorkItem(t)
	w.send(t, "alice", http.MethodPost, reactionsOf(id), `{"reaction":"128077"}`, http.StatusCreated)
	return reactionsOf(id) + "/128077"
}
