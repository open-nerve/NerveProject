package postgresadapter_test

import (
	"context"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// visit records v, made by its owner, as a recording writes it, in one transaction of tm's: his visits' advisory
// lock, the upsert, then the trim to the cap (P3d spec 2.8). What the trim deleted is read from the table.
func visit(s *postgresadapter.Store, tm *postgres.TxManager, v domain.RecentVisit) error {
	return tm.WithinTx(context.Background(), func(ctx context.Context) error {
		if err := s.LockRecentVisits(ctx, v.UserID, v.WorkspaceID); err != nil {
			return err
		}
		if err := s.RecordRecentVisit(ctx, v, v.UserID); err != nil {
			return err
		}
		return s.TrimRecentVisits(ctx, v.UserID, v.WorkspaceID, domain.RecentVisitCap)
	})
}

// visits are n of user's visits of work items in workspace's project, numbered from first: the i-th visited i minutes
// after base, so the greater the number the newer the visit, and made i minutes before it, so the newer the visit the
// older the row. A list or a trim by creation reads them in the reverse of the visit's order.
func visits(workspace, project, user uuid.UUID, first, n int, base time.Time) []visitRow {
	out := make([]visitRow, n)
	for i := range out {
		at := time.Duration(i+1) * time.Minute
		out[i] = visitRow{numberedID(first + i), workspace, project, user, domain.RecentVisitIssue, uuid.NewV7(), base.Add(-at), base.Add(at), nil}
	}
	return out
}

func rowIDs(rows []visitRow) []uuid.UUID {
	out := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		out[i] = r.id
	}
	return out
}

// tableIDs are the ids of user's rows in workspace, deleted or not, in id order.
func tableIDs(t *testing.T, pool *pgxpool.Pool, user, workspace uuid.UUID) []uuid.UUID {
	t.Helper()
	var ids []uuid.UUID
	if err := pool.QueryRow(context.Background(), `SELECT coalesce(array_agg(id ORDER BY id), '{}') FROM user_recent_visits
		WHERE user_id = $1 AND workspace_id = $2`, user, workspace).Scan(&ids); err != nil {
		t.Fatal(err)
	}
	return ids
}

// sorted is ids in id order.
func sorted(ids ...[]uuid.UUID) []uuid.UUID {
	out := slices.Concat(ids...)
	slices.SortFunc(out, uuid.UUID.Compare)
	return out
}

// A recording's trim keeps one's 20 newest undeleted visits in the workspace, by visit, not by creation (D35), and
// deletes for good every other undeleted one (spec 2.5, 2.8): from 23, with the oldest revisited, the three oldest of
// the others, not only the oldest one (Plane's == 20); from 21, as a rolled-back deletion leaves them, with a new
// target, the two oldest, so the transient 21st heals at the next recording. Deleted visits, the newest of all, take
// no place among the 20 and stay. Another account's visits in the workspace and his own in another workspace, each
// newer than his there, stay as they are.
func TestTheTrimKeepsTheTwentyNewestVisits(t *testing.T) {
	for _, tt := range []struct {
		name    string
		live    int  // his undeleted visits there, before the recording
		revisit bool // the recording revisits his oldest one; else it is of a new target
		deleted int  // his deleted visits there, the newest of all
		trimmed int
	}{
		{"23 with a revisit", 23, true, 0, 3},
		{"a transient 21st and a new target", 21, false, 0, 2},
		{"newer deleted ones take no place", 21, true, 3, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s, pool := newStore(t)
			w := newViews(t, pool)
			mine := visits(w.workspace, w.project, w.owner, 100, tt.live, now.Add(-time.Hour))
			gone := visits(w.workspace, w.ops, w.owner, 200, tt.deleted, now)
			for i := range gone {
				gone[i].deleted = ptr(now.Add(time.Hour))
			}
			hers := visits(w.workspace, w.project, w.other, 300, 3, now)
			elsewhere := visits(w.elsewhere, w.elsewhereProject, w.owner, 400, 3, now)
			seedVisit(t, pool, slices.Concat(mine, gone, hers, elsewhere)...)
			unchanged := stamps(t, pool, "user_recent_visits", rowIDs(slices.Concat(gone, hers, elsewhere))...)
			recorded := domain.RecentVisit{ID: numberedID(500), WorkspaceID: w.workspace, ProjectID: w.ops, UserID: w.owner, Kind: domain.RecentVisitIssue,
				EntityID: uuid.NewV7(), VisitedAt: now}
			kept := []uuid.UUID{recorded.ID}
			if tt.revisit {
				// A revisit of his oldest one: the row keeps its id, its visit the newest.
				recorded.ProjectID, recorded.EntityID = mine[0].project, mine[0].target
				kept, mine = nil, slices.Concat(mine[1:], mine[:1])
			}
			if err := visit(s, postgres.NewTxManager(pool, time.Second), recorded); err != nil {
				t.Fatal(err)
			}
			kept = sorted(kept, rowIDs(mine[tt.trimmed:]), rowIDs(gone))
			if got := tableIDs(t, pool, w.owner, w.workspace); !slices.Equal(got, kept) {
				t.Fatalf("his rows there %v; want %v: his %d oldest visits %v gone for good, no other row", got, kept, tt.trimmed,
					rowIDs(mine[:tt.trimmed]))
			}
			live, err := s.RecentVisits(context.Background(), w.owner, w.workspace, nil)
			if err != nil || len(live) != 20 {
				t.Fatalf("his visits there %d, %v; want 20", len(live), err)
			}
			for _, scope := range []struct{ user, workspace uuid.UUID }{{w.other, w.workspace}, {w.owner, w.elsewhere}} {
				if got := tableIDs(t, pool, scope.user, scope.workspace); len(got) != 3 {
					t.Fatalf("%s's rows in %s %v; want the 3 as they were", scope.user, scope.workspace, got)
				}
			}
			if got := stamps(t, pool, "user_recent_visits", rowIDs(slices.Concat(gone, hers, elsewhere))...); !slices.Equal(got, unchanged) {
				t.Fatalf("his deleted ones, another's and his in another workspace %v; want %v", got, unchanged)
			}
		})
	}
}

// The trim never waits for a row (spec 2.8): one of his visits past the 20th that another transaction holds is
// passed over, the recording answering within its lock timeout, and stays, his transient 21st; once the holder lets
// go, his next recording deletes it. A trim that waited would fail at the timeout.
func TestTheTrimPassesOverAHeldVisit(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newViews(t, pool)
	mine := visits(w.workspace, w.project, w.owner, 100, 21, now.Add(-time.Hour))
	seedVisit(t, pool, mine...)
	holder, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = holder.Rollback(ctx) }()
	if _, err := holder.Exec(ctx, "SELECT 1 FROM user_recent_visits WHERE id = $1 FOR UPDATE", mine[0].id); err != nil {
		t.Fatal(err)
	}
	tm := postgres.NewTxManager(pool, time.Second, postgres.LockTimeout(time.Second))
	revisit := visitOf(mine[20])
	revisit.VisitedAt = now
	if err := visit(s, tm, revisit); err != nil {
		t.Fatalf("the recording while the 21st is held: %v; want it answered without waiting", err)
	}
	if got := tableIDs(t, pool, w.owner, w.workspace); !slices.Equal(got, rowIDs(mine)) {
		t.Fatalf("his rows %v; want the 21, none deleted, the held one among them", got)
	}
	if err := holder.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	revisit.VisitedAt = now.Add(time.Minute)
	if err := visit(s, tm, revisit); err != nil {
		t.Fatalf("the recording once the 21st is let go: %v", err)
	}
	if got := tableIDs(t, pool, w.owner, w.workspace); !slices.Equal(got, rowIDs(mine[1:])) {
		t.Fatalf("his rows %v; want the 20 newest, the 21st %s alone deleted", got, mine[0].id)
	}
	for _, keep := range []int{0, 2147483648} {
		if err := s.TrimRecentVisits(ctx, w.owner, w.workspace, keep); err == nil {
			t.Fatalf("accepted the trim to %d", keep)
		}
	}
}

// One's recordings in a workspace take one advisory lock until their transactions end (spec 2.8): a second
// transaction's waits for the first's, and takes it once the first commits. Another account's lock in the
// workspace, and his own in another workspace, are other locks: taken at once while his is held, within a lock
// timeout under which his own would fail. No app runs on this database, so any lock wait is the second's; a lock that
// outlived its transaction would fail the second at its own timeout.
func TestTheRecentVisitsLockIsOnePersonsInOneWorkspace(t *testing.T) {
	s, pool := newStore(t)
	w := newViews(t, pool)
	held, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	released := false
	defer func() {
		if !released {
			close(release)
		}
	}()
	go func() {
		done <- postgres.NewTxManager(pool, 10*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
			err := s.LockRecentVisits(ctx, w.owner, w.workspace)
			close(held)
			if err != nil {
				return err
			}
			<-release
			return nil
		})
	}()
	<-held
	quick := postgres.NewTxManager(pool, time.Second, postgres.LockTimeout(time.Second))
	for _, other := range []struct{ user, workspace uuid.UUID }{{w.other, w.workspace}, {w.owner, w.elsewhere}} {
		if err := quick.WithinTx(context.Background(), func(ctx context.Context) error { return s.LockRecentVisits(ctx, other.user, other.workspace) }); err != nil {
			t.Fatalf("%s's lock in %s: %v; want it taken at once", other.user, other.workspace, err)
		}
	}
	err := quick.WithinTx(context.Background(), func(ctx context.Context) error { return s.LockRecentVisits(ctx, w.owner, w.workspace) })
	if !postgres.LockNotAvailable(err) {
		t.Fatalf("his own lock while held: %v; want it to wait past the lock timeout", err)
	}
	second, took := make(chan error, 1), make(chan struct{})
	go func() {
		second <- postgres.NewTxManager(pool, 10*time.Second, postgres.LockTimeout(10*time.Second)).WithinTx(context.Background(), func(ctx context.Context) error {
			if err := s.LockRecentVisits(ctx, w.owner, w.workspace); err != nil {
				return err
			}
			close(took)
			return nil
		})
	}()
	pgtest.WaitForLockWait(t, pool, 10*time.Second)
	select {
	case <-took:
		t.Fatal("a second transaction took his lock while the first held it")
	default:
	}
	released = true
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := <-second; err != nil {
		t.Fatalf("the second transaction once the first committed: %v", err)
	}
}

// The trim and the 60-day purge take disjoint rows, in either order (spec 2.6): the trim his undeleted visits past
// the 20 newest, the purge those deleted before the cutoff. His 20 newest and his visits deleted since the cutoff
// stay; each step deletes its own rows alone, whichever runs first.
func TestTrimAndPurgeOfRecentVisitsTakeDisjointRows(t *testing.T) {
	for _, first := range []string{"trim", "purge"} {
		t.Run(first+" first", func(t *testing.T) {
			s, pool := newStore(t)
			ctx := context.Background()
			w := newWorld(t, pool)
			cutoff := now.Add(-60 * 24 * time.Hour)
			live := visits(w.workspace, w.project, w.owner, 100, 22, now.Add(-time.Hour))
			old := visits(w.workspace, w.project, w.owner, 200, 2, now)
			recent := visits(w.workspace, w.project, w.owner, 300, 2, now)
			for i := range old {
				old[i].deleted, recent[i].deleted = ptr(cutoff.Add(-24*time.Hour)), ptr(cutoff.Add(24*time.Hour))
			}
			seedVisit(t, pool, slices.Concat(live, old, recent)...)
			trim := func() []uuid.UUID {
				if err := s.TrimRecentVisits(ctx, w.owner, w.workspace, domain.RecentVisitCap); err != nil {
					t.Fatal(err)
				}
				return rowIDs(live[:2])
			}
			purge := func() []uuid.UUID {
				if page, err := s.PurgeUserRecentVisits(ctx, app.PurgeBatch{Cutoff: cutoff, Limit: 100}); err != nil || page.Deleted != 2 {
					t.Fatalf("purged %+v, %v; want 2", page, err)
				}
				return rowIDs(old)
			}
			steps := []func() []uuid.UUID{trim, purge}
			if first == "purge" {
				steps[0], steps[1] = purge, trim
			}
			left := tableIDs(t, pool, w.owner, w.workspace)
			for i, step := range steps {
				gone := step()
				want := slices.DeleteFunc(slices.Clone(left), func(id uuid.UUID) bool { return slices.Contains(gone, id) })
				if got := tableIDs(t, pool, w.owner, w.workspace); !slices.Equal(got, want) {
					t.Fatalf("after step %d %v; want %v, its own rows alone gone", i+1, got, want)
				}
				left = want
			}
			if want := sorted(rowIDs(live[2:]), rowIDs(recent)); !slices.Equal(left, want) {
				t.Fatalf("left %v; want his 20 newest and the visits deleted since the cutoff", left)
			}
		})
	}
}
