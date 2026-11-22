package postgresadapter_test

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

var (
	_ app.RecentVisitReader   = (*postgresadapter.Store)(nil)
	_ app.RecentVisitLocker   = (*postgresadapter.Store)(nil)
	_ app.RecentVisitWriter   = (*postgresadapter.Store)(nil)
	_ app.RecentVisitCascades = (*postgresadapter.Store)(nil)
)

// visitRow is a row of user_recent_visits as a test writes it through SQL, the one shape of it: user's visit of
// target of kind, in workspace's project, made at made and last visited at visited, by user, his last writing; deleted
// at deleted when it is not nil, its last change then.
type visitRow struct {
	id, workspace, project, user uuid.UUID
	kind                         domain.RecentVisitKind
	target                       uuid.UUID
	made, visited                time.Time
	deleted                      *time.Time
}

// seedVisit writes each row.
func seedVisit(t *testing.T, pool *pgxpool.Pool, rows ...visitRow) {
	t.Helper()
	for _, r := range rows {
		exec(t, pool, `INSERT INTO user_recent_visits (id, workspace_id, project_id, user_id, entity_type, entity_identifier, visited_at,
				created_by_id, updated_by_id, created_at, updated_at, deleted_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $4, $4, $8, coalesce($9::timestamptz, $7::timestamptz), $9)`,
			r.id, r.workspace, r.project, r.user, string(r.kind), r.target, r.visited, r.made, r.deleted)
	}
}

// visitOf is the visit row r as the store answers it.
func visitOf(r visitRow) domain.RecentVisit {
	return domain.RecentVisit{ID: r.id, WorkspaceID: r.workspace, ProjectID: r.project, UserID: r.user, Kind: r.kind, EntityID: r.target, VisitedAt: r.visited}
}

func visitIDs(vs []domain.RecentVisit) []uuid.UUID {
	out := make([]uuid.UUID, len(vs))
	for i, v := range vs {
		out[i] = v.ID
	}
	return out
}

// One's recent visits in a workspace (P3d spec 2.5): his undeleted ones there, of both kinds, by visit, the newest
// first, and at one moment by id descending; none of another's of the same targets, of his in another workspace, nor
// his deleted one. The order is the query's whatever plan reads it: the list is read as planned and again with index
// scans off, the rows reaching the sort as written. The two at one moment are written in their id order, the reverse
// of the list's, and the rows that reach the sort are too few for it to reorder equal ones, so an order by the visit
// alone fails here (hard rules §5).
func TestTheRecentVisitsByVisitThenByID(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newViews(t, pool)
	item, other := uuid.NewV7(), uuid.NewV7()
	at := func(minutes int) time.Time { return now.Add(time.Duration(minutes) * time.Minute) }
	mine := []visitRow{
		{numberedID(40), w.workspace, w.project, w.owner, domain.RecentVisitIssue, item, at(-60), at(3), nil},
		{numberedID(30), w.workspace, w.project, w.owner, domain.RecentVisitProject, w.project, at(-50), at(3), nil},
		{numberedID(10), w.workspace, w.ops, w.owner, domain.RecentVisitIssue, other, at(-70), at(2), nil},
		{numberedID(20), w.workspace, w.ops, w.owner, domain.RecentVisitProject, w.ops, at(-90), at(1), nil},
	}
	seedVisit(t, pool, mine[1], mine[0], mine[2], mine[3],
		visitRow{numberedID(50), w.workspace, w.ops, w.owner, domain.RecentVisitIssue, uuid.NewV7(), at(-5), at(9), ptr(at(10))},
		visitRow{numberedID(60), w.workspace, w.project, w.other, domain.RecentVisitIssue, item, at(-5), at(8), nil},
		visitRow{numberedID(70), w.workspace, w.project, w.other, domain.RecentVisitProject, w.project, at(-5), at(7), nil},
		visitRow{numberedID(80), w.elsewhere, w.elsewhereProject, w.owner, domain.RecentVisitProject, w.elsewhereProject, at(-5), at(6), nil})
	want := []domain.RecentVisit{visitOf(mine[0]), visitOf(mine[1]), visitOf(mine[2]), visitOf(mine[3])}
	for _, plan := range []string{"as planned", "without index scans"} {
		err := postgres.NewTxManager(pool, time.Second).WithinTx(ctx, func(txctx context.Context) error {
			if plan == "without index scans" {
				if _, err := postgres.DB(txctx, pool).Exec(txctx,
					"SELECT set_config('enable_indexscan', 'off', true), set_config('enable_bitmapscan', 'off', true)"); err != nil {
					return err
				}
			}
			if got, err := s.RecentVisits(txctx, w.owner, w.workspace, nil); err != nil || !reflect.DeepEqual(got, want) {
				t.Errorf("%s: %v, %v; want %v", plan, visitIDs(got), err, visitIDs(want))
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, tt := range []struct {
		user, workspace uuid.UUID
		want            []uuid.UUID
	}{{w.other, w.workspace, []uuid.UUID{numberedID(60), numberedID(70)}}, {w.owner, w.elsewhere, []uuid.UUID{numberedID(80)}}} {
		if got, err := s.RecentVisits(ctx, tt.user, tt.workspace, nil); err != nil || !slices.Equal(visitIDs(got), tt.want) {
			t.Errorf("%s in %s: %v, %v; want %v", tt.user, tt.workspace, visitIDs(got), err, tt.want)
		}
	}
	if got, err := s.RecentVisits(ctx, w.other, w.elsewhere, nil); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("no visits %#v, %v; want an empty list", got, err)
	}
}

// The list's entity_type keeps its kind alone (spec 2.4, 2.5), in the list's order; none of another's of it, nor of
// his in another workspace.
func TestTheRecentVisitsOfAKind(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newViews(t, pool)
	visits := []visitRow{
		{numberedID(1), w.workspace, w.project, w.owner, domain.RecentVisitIssue, uuid.NewV7(), now, now.Add(4 * time.Minute), nil},
		{numberedID(2), w.workspace, w.project, w.owner, domain.RecentVisitProject, w.project, now, now.Add(3 * time.Minute), nil},
		{numberedID(3), w.workspace, w.ops, w.owner, domain.RecentVisitIssue, uuid.NewV7(), now, now.Add(2 * time.Minute), nil},
		{numberedID(4), w.workspace, w.ops, w.owner, domain.RecentVisitProject, w.ops, now, now.Add(time.Minute), nil},
		{numberedID(5), w.workspace, w.project, w.other, domain.RecentVisitIssue, uuid.NewV7(), now, now.Add(5 * time.Minute), nil},
		{numberedID(6), w.elsewhere, w.elsewhereProject, w.owner, domain.RecentVisitProject, w.elsewhereProject, now, now.Add(6 * time.Minute), nil},
	}
	seedVisit(t, pool, visits...)
	for _, tt := range []struct {
		kind domain.RecentVisitKind
		want []domain.RecentVisit
	}{{domain.RecentVisitIssue, []domain.RecentVisit{visitOf(visits[0]), visitOf(visits[2])}},
		{domain.RecentVisitProject, []domain.RecentVisit{visitOf(visits[1]), visitOf(visits[3])}}} {
		if got, err := s.RecentVisits(ctx, w.owner, w.workspace, &tt.kind); err != nil || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: %v, %v; want %v", tt.kind, visitIDs(got), err, visitIDs(tt.want))
		}
	}
}

// visitColumns are a row of user_recent_visits: the visit as the store answers it, its creation, its last change and
// its deletion, every moment in UTC.
type visitColumns struct {
	domain.RecentVisit
	Made, Changed  time.Time
	Maker, Changer *uuid.UUID
	Deleted        *time.Time
}

// readVisit reads the row id.
func readVisit(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) visitColumns {
	t.Helper()
	var c visitColumns
	var kind string
	if err := pool.QueryRow(context.Background(), `SELECT id, workspace_id, project_id, user_id, entity_type, entity_identifier, visited_at,
			created_at, created_by_id, updated_at, updated_by_id, deleted_at
		FROM user_recent_visits WHERE id = $1`, id).Scan(&c.ID, &c.WorkspaceID, &c.ProjectID, &c.UserID, &kind, &c.EntityID, &c.VisitedAt,
		&c.Made, &c.Maker, &c.Changed, &c.Changer, &c.Deleted); err != nil {
		t.Fatalf("read visit %s: %v", id, err)
	}
	c.Kind, c.VisitedAt, c.Made, c.Changed = domain.RecentVisitKind(kind), c.VisitedAt.UTC(), c.Made.UTC(), c.Changed.UTC()
	if c.Deleted != nil {
		c.Deleted = ptr(c.Deleted.UTC())
	}
	return c
}

// A recording stores a visit, made by its writer at its visit, its creation, change and visit one moment (spec 2.3,
// 2.5). Its owner's next recording of the target in the workspace revisits that row: its id, creation and maker kept,
// its visit and its last change stamped at the new visit's moment by the new writer, and no other row. Another
// account's visit of the target is another row. Once the visit is deleted, a recording of the target is a new row,
// the deleted one keeping its deletion: the recording's conflict is the partial unique index's, of undeleted rows
// alone.
func TestRecordAndRevisitARecentVisit(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newViews(t, pool)
	item := uuid.NewV7()
	first := domain.RecentVisit{ID: numberedID(1), WorkspaceID: w.workspace, ProjectID: w.project, UserID: w.owner, Kind: domain.RecentVisitIssue, EntityID: item,
		VisitedAt: now.Add(-time.Hour)}
	if err := s.RecordRecentVisit(ctx, first, w.owner); err != nil {
		t.Fatal(err)
	}
	made := visitColumns{RecentVisit: first, Made: first.VisitedAt, Maker: &w.owner, Changed: first.VisitedAt, Changer: &w.owner}
	if got := readVisit(t, pool, first.ID); !reflect.DeepEqual(got, made) {
		t.Fatalf("recorded %+v; want %+v", got, made)
	}
	again := first
	again.ID, again.VisitedAt = numberedID(2), now
	if err := s.RecordRecentVisit(ctx, again, w.other); err != nil {
		t.Fatal(err)
	}
	revisited := made
	revisited.VisitedAt, revisited.Changed, revisited.Changer = now, now, &w.other
	if got := readVisit(t, pool, first.ID); !reflect.DeepEqual(got, revisited) {
		t.Fatalf("revisited %+v; want %+v", got, revisited)
	}
	hers := first
	hers.ID, hers.UserID = numberedID(3), w.other
	if err := s.RecordRecentVisit(ctx, hers, w.other); err != nil {
		t.Fatal(err)
	}
	if got, err := s.RecentVisits(ctx, w.owner, w.workspace, nil); err != nil || !reflect.DeepEqual(got, []domain.RecentVisit{revisited.RecentVisit}) {
		t.Fatalf("his visits %+v, %v; want the revisited one alone", got, err)
	}
	if err := s.DeleteIssueRecentVisits(ctx, w.workspace, item, now.Add(time.Minute), &w.other); err != nil {
		t.Fatal(err)
	}
	deleted := readVisit(t, pool, first.ID)
	anew := first
	anew.ID, anew.VisitedAt = numberedID(4), now.Add(2*time.Minute)
	if err := s.RecordRecentVisit(ctx, anew, w.owner); err != nil {
		t.Fatal(err)
	}
	if got, err := s.RecentVisits(ctx, w.owner, w.workspace, nil); err != nil || !reflect.DeepEqual(got, []domain.RecentVisit{anew}) {
		t.Fatalf("after its deletion %+v, %v; want the new one alone", got, err)
	}
	if got := readVisit(t, pool, first.ID); !reflect.DeepEqual(got, deleted) || deleted.Deleted == nil {
		t.Fatalf("the deleted one %+v; want %+v", got, deleted)
	}
}
