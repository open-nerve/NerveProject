package bootstrap

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The helpers of the recent visits' composition tests on the real app (M7
// P3d spec 4): TestRecentVisitsThroughTheApp and
// TestRecentVisitsFollowDeletedWorkItems.

// recentVisitHTTP is a recent visit as the API answers it (RecentVisit),
// its work item decoded, nil for null.
type recentVisitHTTP struct {
	ID         uuid.UUID              `json:"id"`
	EntityType string                 `json:"entity_type"`
	EntityID   uuid.UUID              `json:"entity_id"`
	VisitedAt  time.Time              `json:"visited_at"`
	Project    recentVisitAnswerOwner `json:"project"`
	Issue      *recentVisitIssueHTTP  `json:"issue"`
}

// recentVisitIssueHTTP is a recent visit's work item as the API answers it
// (RecentVisitIssue), and a work item's detail as far as it goes.
type recentVisitIssueHTTP struct {
	ID          uuid.UUID   `json:"id"`
	SequenceID  int         `json:"sequence_id"`
	Name        string      `json:"name"`
	StateID     uuid.UUID   `json:"state_id"`
	Priority    string      `json:"priority"`
	AssigneeIDs []uuid.UUID `json:"assignee_ids"`
}

// recordVisitPath is the recordings' route; recentVisitsPath a workspace's
// list's.
const recordVisitPath = "/api/v0/me/recent-visits"

func recentVisitsPath(slug string) string { return "/api/v0/workspaces/" + slug + "/recent-visits" }

// visitBody is the body of a recording of the target of kind with id.
func visitBody(kind string, id uuid.UUID) string {
	return fmt.Sprintf(`{"entity_type":%q,"entity_id":%q}`, kind, id)
}

// visit is actor's recording of his visit of the target of kind with id:
// 204.
func (w issueWorld) visit(t *testing.T, actor, kind string, id uuid.UUID) {
	t.Helper()
	w.send(t, actor, http.MethodPost, recordVisitPath, visitBody(kind, id), http.StatusNoContent)
}

// listedVisits is actor's list in the workspace slug with query ("" or
// "?entity_type=…"), in the answer's order, each moment in UTC.
func (w issueWorld) listedVisits(t *testing.T, actor, slug, query string) []recentVisitHTTP {
	t.Helper()
	var list struct{ Data []recentVisitHTTP }
	decodeAnswer(t, w.send(t, actor, http.MethodGet, recentVisitsPath(slug)+query, "", http.StatusOK), &list)
	for n := range list.Data {
		list.Data[n].VisitedAt = list.Data[n].VisitedAt.UTC()
	}
	return list.Data
}

// listedTargets is the targets of actor's list in the workspace slug, in
// the answer's order.
func (w issueWorld) listedTargets(t *testing.T, actor, slug string) []uuid.UUID {
	t.Helper()
	listed := w.listedVisits(t, actor, slug, "")
	ids := make([]uuid.UUID, len(listed))
	for n, v := range listed {
		ids[n] = v.EntityID
	}
	return ids
}

// issueNow is the work item id as actor reads it now (getIssue), as far as
// a recent visit answers it; projectNow, the project id (getProject).
func (w issueWorld) issueNow(t *testing.T, actor string, id uuid.UUID) recentVisitIssueHTTP {
	t.Helper()
	var i recentVisitIssueHTTP
	decodeAnswer(t, w.send(t, actor, http.MethodGet, "/api/v0/issues/"+id.String(), "", http.StatusOK), &i)
	return i
}

func (w issueWorld) projectNow(t *testing.T, actor string, id uuid.UUID) recentVisitAnswerOwner {
	t.Helper()
	var p recentVisitAnswerOwner
	decodeAnswer(t, w.send(t, actor, http.MethodGet, "/api/v0/projects/"+id.String(), "", http.StatusOK), &p)
	return p
}

// shownVisit is the visit r as its owner, actor, is to find it listed: its
// row's id, kind, target and moment, with its project and, a work item's,
// its work item as he reads them now.
func (w issueWorld) shownVisit(t *testing.T, actor string, r visitRow) recentVisitHTTP {
	t.Helper()
	v := recentVisitHTTP{ID: r.id, EntityType: r.kind, EntityID: r.entity, VisitedAt: r.visitedAt.UTC(), Project: w.projectNow(t, actor, r.projectID)}
	if r.kind == "issue" {
		i := w.issueNow(t, actor, r.entity)
		v.Issue = &i
	}
	return v
}

// listsVisits checks that actor's list in the workspace slug with query is
// his visits rows, each as shownVisit answers it, in their order.
func (w issueWorld) listsVisits(t *testing.T, actor, slug, query string, rows ...visitRow) {
	t.Helper()
	want := make([]recentVisitHTTP, len(rows))
	for n, r := range rows {
		want[n] = w.shownVisit(t, actor, r)
	}
	if got := w.listedVisits(t, actor, slug, query); !reflect.DeepEqual(got, want) {
		t.Errorf("%s lists in %s%s:\n%+v\nwant\n%+v", actor, slug, query, got, want)
	}
}

// wholeOffContract is actor's method on path with body as a wholeAnswer,
// a request that breaks the contract on purpose (an entity_type of
// neither kind), so unchecked; its answer is checked against the contract.
func (w issueWorld) wholeOffContract(t *testing.T, actor, method, path, body string) wholeAnswer {
	t.Helper()
	var b []byte
	if body != "" {
		b = []byte(body)
	}
	req := newRequest(t, method, w.base+path, w.tokens[actor], b)
	res, raw := send(t, req)
	w.contract.CheckResponse(t, req, res)
	return wholeAnswerOf(res, raw)
}

// visitAnsweredAsAbsent checks that actor's recording of his visit of the
// target of kind with id answers its own 404, issue.not_found or
// project.not_found, the same answer byte for byte, its status, body and
// headers, as of an id no target of the kind ever had (spec 2.7), and that
// neither writes a visit.
func (w issueWorld) visitAnsweredAsAbsent(t *testing.T, actor, kind string, id uuid.UUID) {
	t.Helper()
	code := map[string]string{"issue": "issue.not_found", "project": "project.not_found"}[kind]
	before := visitTable(t, w.pool)
	got, want := w.whole(t, actor, http.MethodPost, recordVisitPath, visitBody(kind, id)), w.whole(t, actor, http.MethodPost, recordVisitPath,
		visitBody(kind, uuid.NewV7()))
	if want.status != http.StatusNotFound || problemCode(t, []byte(want.body)) != code {
		t.Fatalf("%s's recording of an id no %s ever had: %d %s; want 404 %s", actor, kind, want.status, want.body, code)
	}
	if got != want {
		t.Errorf("%s's recording of the %s %s differs from one of an id no %s ever had:\n%+v\n%+v", actor, kind, id, kind, got, want)
	}
	if after := visitTable(t, w.pool); after != before {
		t.Errorf("%s's refused recordings of the %s %s wrote:\n%s\nwere\n%s", actor, kind, id, after, before)
	}
}

// visitRow is a recent visit's row as the API does not show it all: whose
// it is, where, who made it and changed it last and when, when it was
// deleted.
type visitRow struct {
	id, workspaceID, projectID, userID uuid.UUID
	kind                               string
	entity                             uuid.UUID
	visitedAt, createdAt, updatedAt    time.Time
	createdBy, updatedBy               *uuid.UUID
	deletedAt                          *time.Time
}

func (r visitRow) String() string {
	stamp := func(at *time.Time, by *uuid.UUID) string {
		s := "none"
		if at != nil {
			s = at.UTC().Format(time.RFC3339Nano)
		}
		if by != nil {
			s += " by " + by.String()
		}
		return s
	}
	return fmt.Sprintf("{%s: %s's of the %s %s in %s/%s, visited %s, made %s, changed %s, deleted %s}", r.id, r.userID, r.kind, r.entity,
		r.workspaceID, r.projectID, r.visitedAt.UTC().Format(time.RFC3339Nano), stamp(&r.createdAt, r.createdBy), stamp(&r.updatedAt, r.updatedBy),
		stamp(r.deletedAt, nil))
}

// visitRowsWhere reads the rows of user_recent_visits where cond holds,
// with args, deleted or not, in id order.
func visitRowsWhere(t *testing.T, pool *pgxpool.Pool, cond string, args ...any) []visitRow {
	t.Helper()
	rows, err := pool.Query(pgtest.Soon(t), `SELECT id, workspace_id, project_id, user_id, entity_type, entity_identifier, visited_at, created_at,
		updated_at, created_by_id, updated_by_id, deleted_at FROM user_recent_visits WHERE `+cond+` ORDER BY id`, args...)
	if err != nil {
		t.Fatal(err)
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (visitRow, error) {
		var r visitRow
		err := row.Scan(&r.id, &r.workspaceID, &r.projectID, &r.userID, &r.kind, &r.entity, &r.visitedAt, &r.createdAt, &r.updatedAt,
			&r.createdBy, &r.updatedBy, &r.deletedAt)
		return r, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// liveVisit is the row of owner's one undeleted visit of the target id.
func liveVisit(t *testing.T, pool *pgxpool.Pool, owner, id uuid.UUID) visitRow {
	t.Helper()
	rows := visitRowsWhere(t, pool, "user_id = $1 AND entity_identifier = $2 AND deleted_at IS NULL", owner, id)
	if len(rows) != 1 {
		t.Fatalf("%s's undeleted visits of %s: %v; want one", owner, id, rows)
	}
	return rows[0]
}

// liveVisits is the rows of each owner's one undeleted visit of each of
// the targets ids, owner by owner.
func liveVisits(t *testing.T, pool *pgxpool.Pool, owners []uuid.UUID, ids ...uuid.UUID) []visitRow {
	t.Helper()
	var rows []visitRow
	for _, owner := range owners {
		for _, id := range ids {
			rows = append(rows, liveVisit(t, pool, owner, id))
		}
	}
	return rows
}

// visitTable is the whole of user_recent_visits, each row as text, in id
// order: what a request that writes no visit leaves byte for byte.
func visitTable(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var table string
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT coalesce(string_agg(v::text, E'\\n' ORDER BY v.id), '') FROM user_recent_visits AS v").
		Scan(&table); err != nil {
		t.Fatal(err)
	}
	return table
}

// visitsText is the whole rows of the visits rows, as they are now, in id
// order, as text: what a write that leaves them untouched keeps byte for
// byte.
func visitsText(t *testing.T, pool *pgxpool.Pool, rows ...visitRow) string {
	t.Helper()
	ids := make([]uuid.UUID, len(rows))
	for n, r := range rows {
		ids[n] = r.id
	}
	var text string
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT coalesce(string_agg(v::text, E'\\n' ORDER BY v.id), '') FROM user_recent_visits AS v WHERE v.id = ANY($1)",
		ids).Scan(&text); err != nil {
		t.Fatal(err)
	}
	return text
}

// visitsDeletedAt checks that each of the visits rows, as it was before,
// is now deleted and last changed at at by by, every other column kept:
// its owner, target, visit and creation (spec 2.6).
func visitsDeletedAt(t *testing.T, pool *pgxpool.Pool, at time.Time, by uuid.UUID, rows ...visitRow) {
	t.Helper()
	for _, was := range rows {
		now := visitRowsWhere(t, pool, "id = $1", was.id)
		want := was
		want.deletedAt, want.updatedAt, want.updatedBy = &at, at, &by
		if len(now) != 1 || !sameVisitRow(now[0], want) {
			t.Errorf("the visit %s now %v; want it deleted and last changed at %s by %s", was, now, at.UTC().Format(time.RFC3339Nano), by)
		}
	}
}

// sameVisitRow tells whether a and b are the same row, column by column,
// their moments as instants.
func sameVisitRow(a, b visitRow) bool {
	sameAt := func(x, y *time.Time) bool { return x == nil && y == nil || x != nil && y != nil && x.Equal(*y) }
	sameBy := func(x, y *uuid.UUID) bool { return x == nil && y == nil || x != nil && y != nil && *x == *y }
	return a.id == b.id && a.workspaceID == b.workspaceID && a.projectID == b.projectID && a.userID == b.userID && a.kind == b.kind &&
		a.entity == b.entity && a.visitedAt.Equal(b.visitedAt) && a.createdAt.Equal(b.createdAt) && a.updatedAt.Equal(b.updatedAt) &&
		sameBy(a.createdBy, b.createdBy) && sameBy(a.updatedBy, b.updatedBy) && sameAt(a.deletedAt, b.deletedAt)
}
