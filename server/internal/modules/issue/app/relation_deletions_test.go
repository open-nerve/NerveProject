package app

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"
)

type ownershipRows struct {
	rows    []RelationDeletionRow
	locked  []uuid.UUID
	written map[uuid.UUID]IssueDeletion
	calls   []string
	fail    string
	err     error
}

func (r *ownershipRows) lock(after uuid.UUID, limit int) ([]RelationDeletionRow, error) {
	r.calls = append(r.calls, "lock")
	if r.fail == "lock" {
		return nil, r.err
	}
	r.locked = nil
	var out []RelationDeletionRow
	for _, row := range r.rows {
		if _, done := r.written[row.ID]; !done && row.ID.Compare(after) > 0 && len(out) < limit {
			out = append(out, row)
			r.locked = append(r.locked, row.ID)
		}
	}
	return out, nil
}
func (r *ownershipRows) LockProjectRelationRows(_ context.Context, b RowBatch) ([]RelationDeletionRow, error) {
	return r.lock(b.After, b.Limit)
}
func (r *ownershipRows) LockIssueRelationRows(context.Context, IssueDeletion) ([]RelationDeletionRow, error) {
	return r.lock(uuid.UUID{}, len(r.rows))
}
func (r *ownershipRows) DeleteLockedRelationRows(_ context.Context, ids []uuid.UUID, by uuid.UUID, at time.Time) (int, error) {
	r.calls = append(r.calls, "write")
	if r.fail == "write" {
		return 0, r.err
	}
	for _, id := range ids {
		if !slices.Contains(r.locked, id) {
			return 0, errors.New("write escaped locked IDs")
		}
		r.written[id] = IssueDeletion{By: by, At: at}
	}
	return len(ids), nil
}

type ownershipProjects struct {
	ProjectGate
	rows    *ownershipRows
	deleted map[uuid.UUID]time.Time
	asked   []uuid.UUID
	err     error
}

func (p *ownershipProjects) DeletedAt(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]time.Time, error) {
	p.rows.calls = append(p.rows.calls, "deleted")
	p.asked = append([]uuid.UUID(nil), ids...)
	return p.deleted, p.err
}

// Removing the earlier-end filter or reading it after the write corrupts audit ownership.
func TestRelationDeletionOwnership(t *testing.T) {
	at := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	for _, reverse := range []bool{false, true} {
		for _, swap := range []bool{false, true} {
			for _, mode := range []string{"live earlier", "live same", "job earlier", "job same", "job later", "internal", "absent"} {
				t.Run(fmt.Sprintf("%s/reverse=%v/swap=%v", mode, reverse, swap), func(t *testing.T) {
					p, q := uuid.NewV7(), uuid.NewV7()
					if swap {
						p, q = q, p
					}
					if mode == "internal" {
						q = p
					}
					row := RelationDeletionRow{ID: uuid.NewV7(), ProjectID: p, RelatedProjectID: q}
					if reverse {
						row.ProjectID, row.RelatedProjectID = q, p
					}
					r := &ownershipRows{rows: []RelationDeletionRow{row}, written: map[uuid.UUID]IssueDeletion{}}
					otherAt := at.Add(-time.Hour)
					if mode == "job same" || mode == "live same" {
						otherAt = at
					}
					if mode == "job later" {
						otherAt = at.Add(time.Hour)
					}
					projects := &ownershipProjects{rows: r, deleted: map[uuid.UUID]time.Time{q: otherAt}}
					if mode == "absent" {
						projects.deleted = nil
					}
					u := NewRelationDeletions(r, projects)
					by := uuid.NewV7()
					live := mode == "live earlier" || mode == "live same"
					skip := live || mode == "job earlier"
					if live {
						if err := u.ForIssues(context.Background(), p, IssueDeletion{IssueIDs: []uuid.UUID{uuid.NewV7()}, By: by, At: at}); err != nil {
							t.Fatal(err)
						}
					} else {
						got, err := u.ForProject(context.Background(), RowBatch{ProjectID: p, Limit: 2, By: by, At: at})
						want := BatchProgress{After: row.ID, Candidates: 1, Written: 1}
						if skip {
							want.Written = 0
						}
						if err != nil || got != want {
							t.Fatalf("progress %+v %v want %+v", got, err, want)
						}
					}
					want := map[uuid.UUID]IssueDeletion{}
					if !skip {
						want[row.ID] = IssueDeletion{By: by, At: at}
					}
					if !reflect.DeepEqual(r.written, want) {
						t.Fatalf("audit %+v want %+v", r.written, want)
					}
					calls := []string{"lock", "deleted"}
					if !skip {
						calls = append(calls, "write")
					}
					if !slices.Equal(r.calls, calls) {
						t.Fatal("lock/read/write order", r.calls)
					}
				})
			}
		}
	}
}

// Written=0 must not stop the actual production batch scanner: the first two
// low IDs belong to the earlier project; the high ID still belongs to this one.
func TestRelationDeletionCursorUsesCandidates(t *testing.T) {
	at := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	for _, mode := range []string{"skip then owned", "all skipped", "mixed", "empty"} {
		t.Run(mode, func(t *testing.T) {
			p, old, live := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
			a, b, c := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
			rows := []RelationDeletionRow{{a, p, old}, {b, old, p}, {c, p, live}}
			if mode == "all skipped" {
				rows[2].RelatedProjectID = old
			}
			if mode == "mixed" {
				rows[1].ProjectID = live
			}
			if mode == "empty" {
				rows = nil
			}
			r := &ownershipRows{rows: rows, written: map[uuid.UUID]IssueDeletion{}}
			u := NewRelationDeletions(r, &ownershipProjects{rows: r, deleted: map[uuid.UUID]time.Time{old: at.Add(-time.Hour)}})
			var progress []BatchProgress
			step := func(ctx context.Context, b RowBatch) (BatchProgress, error) {
				p, err := u.ForProject(ctx, b)
				progress = append(progress, p)
				return p, err
			}
			batch := RowBatch{ProjectID: p, Limit: 2, At: at, By: uuid.NewV7()}
			if err := deleteRowBatches(context.Background(), step, batch); err != nil {
				t.Fatal(err)
			}
			want := []BatchProgress{{b, 2, 0}, {c, 1, 1}, {Candidates: 0}}
			if mode == "all skipped" {
				want[1].Written = 0
			}
			if mode == "mixed" {
				want[0].Written = 1
			}
			if mode == "empty" {
				want = []BatchProgress{{}}
			}
			if !reflect.DeepEqual(progress, want) {
				t.Fatalf("production scan progress %+v want %+v", progress, want)
			}
			expected := 0
			if mode == "skip then owned" {
				expected = 1
			}
			if mode == "mixed" {
				expected = 2
			}
			if len(r.written) != expected {
				t.Fatalf("wrote %d want %d", len(r.written), expected)
			}
			before := len(r.written)
			if err := deleteRowBatches(context.Background(), u.ForProject, batch); err != nil || len(r.written) != before {
				t.Fatal("retry changed audit", err, r.written)
			}
		})
	}
}

func TestRelationDeletionPortErrors(t *testing.T) {
	failure := errors.New("relation port failed")
	for _, live := range []bool{false, true} {
		for _, port := range []string{"lock", "deleted", "write"} {
			t.Run(fmt.Sprintf("%s/live=%v", port, live), func(t *testing.T) {
				p := uuid.NewV7()
				r := &ownershipRows{rows: []RelationDeletionRow{{uuid.NewV7(), p, uuid.NewV7()}}, written: map[uuid.UUID]IssueDeletion{}, fail: port, err: failure}
				projects := &ownershipProjects{rows: r}
				if port == "deleted" {
					projects.err = failure
				}
				u := NewRelationDeletions(r, projects)
				var err error
				if live {
					err = u.ForIssues(context.Background(), p, IssueDeletion{})
				} else {
					_, err = u.ForProject(context.Background(), RowBatch{ProjectID: p, Limit: 2})
				}
				if err != failure || len(r.written) != 0 {
					t.Fatalf("error=%v writes=%v", err, r.written)
				}
			})
		}
	}
}
