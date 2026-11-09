package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"testing"
	"time"
	"uuid"
)

// A cycle's deletion (M6 P3b spec 2.5): DeleteCyclePreferenceRows soft-deletes the deleted cycle's undeleted display
// settings' rows, each account's of its page, at the moment and by the account given; a row deleted before keeps its
// own deletion, the other cycle's rows theirs, and every other column of every row is kept. Again later, nothing
// changes. The statement answers a failed write as its failure.
func TestDeleteCyclePreferenceRows(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	at := now.Truncate(time.Microsecond)
	sprint, other := uuid.NewV7(), uuid.NewV7()
	kept, gone := account(t, pool, "kep"), account(t, pool, "gon")
	for _, c := range []uuid.UUID{sprint, other} {
		cycle(t, pool, w, w.web, c, nil)
		for _, user := range []uuid.UUID{w.by, kept, gone} {
			exec(t, pool, `INSERT INTO cycle_user_properties (id, workspace_id, project_id, cycle_id, user_id, filters, created_by_id, updated_by_id,
				created_at, updated_at) VALUES ($1, $2, $3, $4, $5, '{"priority":["urgent"]}', $5, $5, $6, $6)`, uuid.NewV7(), w.workspace, w.web, c, user,
				at.Add(-time.Hour))
		}
	}
	exec(t, pool, `UPDATE cycle_user_properties SET deleted_at = $1, updated_at = $1, updated_by_id = $2 WHERE cycle_id = $3 AND user_id = $4`,
		at.Add(-time.Minute), kept, sprint, gone)
	before, rest := audited(t, pool, "cycle_user_properties", w.web)
	live := queryIDs(t, pool, `SELECT id FROM cycle_user_properties WHERE cycle_id = $1 AND deleted_at IS NULL`, sprint)
	if len(before) != 6 || len(live) != 2 {
		t.Fatalf("rows %v, Sprint's live %v; want six, two of them Sprint's live", before, live)
	}
	deleter := account(t, pool, "del")
	for _, when := range []time.Time{at, at.Add(time.Hour)} {
		if err := s.DeleteCyclePreferenceRows(ctx, sprint, when, deleter); err != nil {
			t.Fatal(err)
		}
	}
	var stamp string
	if err := pool.QueryRow(ctx, `SELECT concat_ws(' ', $1::timestamptz, $1::timestamptz, $2::uuid)`, at, deleter).Scan(&stamp); err != nil {
		t.Fatal(err)
	}
	want := maps.Clone(before)
	for _, id := range live {
		want[id] = stamp
	}
	if after, afterRest := audited(t, pool, "cycle_user_properties", w.web); !maps.Equal(after, want) || afterRest != rest {
		t.Errorf("stamps %v, rest %s\nwant %v, %s", after, afterRest, want, rest)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := s.DeleteCyclePreferenceRows(cancelled, other, at, deleter); !errors.Is(err, context.Canceled) {
		t.Errorf("on a cancelled context = %v; want context.Canceled", err)
	}
}
