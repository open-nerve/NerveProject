package postgresadapter_test

import (
	"context"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// readNow reads the ids of the notifications read at now, by bob at now, in id order.
func readNow(t *testing.T, pool *pgxpool.Pool, w readWorld) []uuid.UUID {
	t.Helper()
	var ids []uuid.UUID
	if err := pool.QueryRow(context.Background(), `SELECT coalesce(array_agg(id ORDER BY id), '{}') FROM notifications
		WHERE read_at = $1 AND updated_at = $1 AND updated_by_id = $2`, now, w.receiver).Scan(&ids); err != nil {
		t.Fatal(err)
	}
	return ids
}

// marks runs a batch of mark-all-read and wants it to mark want, as its count and its greatest id, last.
func marks(t *testing.T, s *postgresadapter.Store, ctx context.Context, b app.ReadBatch, want int, last uuid.UUID) {
	t.Helper()
	got, err := s.MarkReadBatch(ctx, b)
	if wantBatch := (app.MarkedBatch{Marked: want, Last: last}); err != nil || got != wantBatch {
		t.Fatalf("batch after %s: %+v, %v; want %+v", b.After, got, err, wantBatch)
	}
}

// A batch marks read, at now by bob, his unread, undeleted notifications of the projects he reads, own-only as the
// list, of an event at or before now, of the filter's senders, archived or not and snoozed at now or not as it says
// (M7 P1c spec 2.7, R2): one of an event a microsecond after now, carol's, another workspace's, a deleted one, one
// he may not see and one read before are not marked, nor is any written.
func TestMarkReadBatchMarksTheMatchingRows(t *testing.T) {
	s, pool := newStore(t)
	w := newReadWorld(t, pool)
	put(t, pool, w, append(readNotes(w),
		note{n: 20, created: now.Add(time.Microsecond), sender: domain.SenderAssigned, creator: &w.alice},
		note{n: 21, created: now, sender: domain.SenderAssigned, creator: &w.alice})...)
	all := domain.TabSenders(domain.TabAll)
	var untouched []uuid.UUID
	for n := 1; n <= 21; n++ {
		if !slices.Contains([]int{1, 3, 4, 5, 6, 7, 8, 9, 21}, n) {
			untouched = append(untouched, purgeID(n))
		}
	}
	before := stamps(t, pool, untouched...)
	ctx := context.Background()
	var marked []int
	for _, step := range []struct {
		name   string
		filter domain.Filter
		want   []int
	}{
		{"the mentions tab", domain.Filter{Senders: domain.TabSenders(domain.TabMentions)}, []int{4}},
		{"archived", domain.Filter{Senders: all, Archived: true}, []int{3}},
		{"snoozed", domain.Filter{Senders: all, Snoozed: true}, []int{5, 7}},
		{"assigned", domain.Filter{Senders: senders(domain.SenderAssigned)}, []int{1, 9, 21}},
		{"the all tab", domain.Filter{Senders: all}, []int{6, 8}},
		{"the all tab again", domain.Filter{Senders: all}, nil},
	} {
		last := uuid.UUID{}
		if len(step.want) > 0 {
			last = purgeID(slices.Max(step.want))
		}
		marks(t, s, ctx, app.ReadBatch{Inbox: w.inbox(), Filter: step.filter, Limit: 100}, len(step.want), last)
		marked = append(marked, step.want...)
		slices.Sort(marked)
		if got := readNow(t, pool, w); !slices.Equal(got, noteIDs(marked...)) {
			t.Fatalf("%s: read at now %v; want %v", step.name, got, noteIDs(marked...))
		}
	}
	if got := stamps(t, pool, untouched...); !slices.Equal(got, before) {
		t.Fatalf("the rows not marked %v; want them as they were: %v", got, before)
	}
	for _, limit := range []int{0, 2147483648} {
		if _, err := s.MarkReadBatch(ctx, app.ReadBatch{Inbox: w.inbox(), Filter: domain.Filter{Senders: all}, Limit: limit}); err == nil {
			t.Fatalf("accepted the batch %d", limit)
		}
	}
}

// Batches run from the greatest id of the one before, in id order, each at most its limit of rows it marks, so they
// mark every row once; a row another transaction marked read and committed before is rechecked in the locked
// subquery and takes no place in a batch, whose limit counts only the rows it marks. The batch after the last
// answers none and the id it ran after.
func TestMarkReadBatchesFromAfterInIDOrder(t *testing.T) {
	s, pool := newStore(t)
	w := newReadWorld(t, pool)
	for n := 1; n <= 7; n++ {
		put(t, pool, w, note{n: n, sender: domain.SenderSubscribed, creator: &w.alice})
	}
	exec(t, pool, "UPDATE notifications SET read_at = $2 WHERE id = ANY($1)", noteIDs(2, 5), minuteAgo)
	batch := func(after uuid.UUID) app.ReadBatch {
		return app.ReadBatch{Inbox: w.inbox(), Filter: domain.Filter{Senders: domain.TabSenders(domain.TabAll)}, After: after, Limit: 2}
	}
	ctx := context.Background()
	for _, step := range []struct {
		after, last uuid.UUID
		marked      int
		read        []int
	}{
		{uuid.UUID{}, purgeID(3), 2, []int{1, 3}},
		{purgeID(3), purgeID(6), 2, []int{1, 3, 4, 6}},
		{purgeID(6), purgeID(7), 1, []int{1, 3, 4, 6, 7}},
		{purgeID(7), purgeID(7), 0, []int{1, 3, 4, 6, 7}},
	} {
		marks(t, s, ctx, batch(step.after), step.marked, step.last)
		if got := readNow(t, pool, w); !slices.Equal(got, noteIDs(step.read...)) {
			t.Fatalf("after %s: read at now %v; want %v", step.after, got, noteIDs(step.read...))
		}
	}
}

// A batch never waits for a row lock (spec 2.7, R5; pre-flight re-84 I1): it passes over a row a single change
// holds FOR NO KEY UPDATE and one a deletion is soft-deleting, marks the others and returns at once, here within a
// lock timeout of a second, which a wait would fail with 55P03. The rows passed over stay unread.
func TestMarkReadBatchPassesOverLockedRows(t *testing.T) {
	s, pool := newStore(t)
	w := newReadWorld(t, pool)
	for n := 1; n <= 4; n++ {
		put(t, pool, w, note{n: n, sender: domain.SenderSubscribed, creator: &w.alice})
	}
	hold := func(sql string, args ...any) pgx.Tx {
		t.Helper()
		tx, err := pool.Begin(pgtest.Soon(t))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
		if _, err := tx.Exec(pgtest.Soon(t), sql, args...); err != nil {
			t.Fatal(err)
		}
		return tx
	}
	change := hold("SELECT 1 FROM notifications WHERE id = $1 FOR NO KEY UPDATE", purgeID(2))
	deletion := hold("UPDATE notifications SET deleted_at = $2, updated_at = $2 WHERE id = $1", purgeID(3), now)
	start := time.Now()
	err := postgres.NewTxManager(pool, time.Second, postgres.LockTimeout(time.Second)).WithinTx(pgtest.Soon(t), func(ctx context.Context) error {
		marks(t, s, ctx, app.ReadBatch{Inbox: w.inbox(), Filter: domain.Filter{Senders: domain.TabSenders(domain.TabAll)}, Limit: 10}, 2, purgeID(4))
		return nil
	})
	if err != nil {
		t.Fatalf("the batch waited for a held row (%v after %s)", err, time.Since(start))
	}
	if got := readNow(t, pool, w); !slices.Equal(got, noteIDs(1, 4)) {
		t.Fatalf("read at now %v; want 1 and 4, the held ones passed over", got)
	}
	if err := change.Rollback(pgtest.Soon(t)); err != nil {
		t.Fatal(err)
	}
	if err := deletion.Commit(pgtest.Soon(t)); err != nil {
		t.Fatal(err)
	}
	var unread []uuid.UUID
	if err := pool.QueryRow(context.Background(), "SELECT array_agg(id ORDER BY id) FROM notifications WHERE read_at IS NULL").Scan(&unread); err != nil ||
		!slices.Equal(unread, noteIDs(2, 3)) {
		t.Fatalf("unread %v, %v; want the two held rows", unread, err)
	}
}

// A batch locks and marks at most its limit whatever plan the planner picks: here one kept from hashing, merging,
// materializing, sorting and index scans, which runs a locking subquery joined to the table again for each of its
// rows. Each run would lock the next rows past those the statement already updated, marking past the limit; the
// locked rows are read once.
func TestMarkReadBatchMarksAtMostItsLimitWhateverThePlan(t *testing.T) {
	s, pool := newStore(t)
	w := newReadWorld(t, pool)
	for n := 1; n <= 7; n++ {
		put(t, pool, w, note{n: n, sender: domain.SenderSubscribed, creator: &w.alice})
	}
	exec(t, pool, "UPDATE notifications SET read_at = $2 WHERE id = ANY($1)", noteIDs(2, 5), minuteAgo)
	err := postgres.NewTxManager(pool, time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		tx, _ := postgres.Tx(ctx)
		if _, err := tx.Exec(ctx, `SET LOCAL enable_hashagg = off; SET LOCAL enable_hashjoin = off; SET LOCAL enable_mergejoin = off;
			SET LOCAL enable_material = off; SET LOCAL enable_sort = off; SET LOCAL enable_indexscan = off; SET LOCAL enable_bitmapscan = off`); err != nil {
			return err
		}
		marks(t, s, ctx, app.ReadBatch{Inbox: w.inbox(), Filter: domain.Filter{Senders: domain.TabSenders(domain.TabAll)}, Limit: 2}, 2, purgeID(3))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := readNow(t, pool, w); !slices.Equal(got, noteIDs(1, 3)) {
		t.Fatalf("read at now %v; want the batch's two, 1 and 3", got)
	}
}
