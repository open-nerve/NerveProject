package app_test

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

type deletionWorld struct {
	other           map[uuid.UUID]uuid.UUID
	deletedProjects map[uuid.UUID]time.Time
	*fakeRows
	commits, begins, failCommit int
	pendingCalls                int
	remainder                   app.ProjectRemainder
	draftRemainder              app.WorkspaceDraftBatch
	pending                     bool
	pendingErr                  error
	runtime                     time.Time
	advance                     time.Duration
}

func (w *deletionWorld) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	w.begins++
	before := maps.Clone(w.deleted)
	err := fn(context.WithValue(ctx, inTxKey{}, true))
	if err == nil && w.begins == w.failCommit {
		err = errDisk
	}
	if err != nil {
		w.deleted = before
		return err
	}
	w.commits++
	w.runtime = w.runtime.Add(w.advance)
	return nil
}
func (w *deletionWorld) ProjectDeletionRemaining(_ context.Context, r app.ProjectRemainder) (bool, error) {
	w.pendingCalls++
	w.remainder = r
	return w.pending, w.pendingErr
}
func (w *deletionWorld) WorkspaceDraftDeletionRemaining(_ context.Context, b app.WorkspaceDraftBatch) (bool, error) {
	w.pendingCalls++
	w.draftRemainder = b
	return w.pending, w.pendingErr
}
func (w *deletionWorld) LockProjectRelationRows(ctx context.Context, b app.RowBatch) ([]app.RelationDeletionRow, error) {
	if err := w.add(ctx, "relations", b.ProjectID, "after", b.After, "limit", b.Limit); err != nil {
		return nil, err
	}
	var rows []app.RelationDeletionRow
	for _, id := range w.rows["relations"][b.ProjectID] {
		if _, gone := w.deleted[id]; gone || id.Compare(b.After) <= 0 || len(rows) >= b.Limit {
			continue
		}
		other := b.ProjectID
		if q, ok := w.other[id]; ok {
			other = q
		}
		rows = append(rows, app.RelationDeletionRow{ID: id, ProjectID: b.ProjectID, RelatedProjectID: other})
	}
	return rows, nil
}
func (w *deletionWorld) LockIssueRelationRows(context.Context, app.IssueDeletion) ([]app.RelationDeletionRow, error) {
	panic("unused")
}
func (w *deletionWorld) DeleteLockedRelationRows(_ context.Context, ids []uuid.UUID, by uuid.UUID, at time.Time) (int, error) {
	for _, id := range ids {
		w.deleted[id] = fmt.Sprintf("%s at %s", by, at.Format(time.RFC3339Nano))
	}
	return len(ids), nil
}
func (w *deletionWorld) useCase() *app.DeleteRows {
	return app.NewDeleteRows(app.DeleteRowsDeps{Rows: w, Relations: app.NewRelationDeletions(w, &fakeProject{log: &callLog{}, deleted: w.deletedProjects}), Remaining: w, Tx: w, RuntimeNow: func() time.Time { return w.runtime }, Batch: 2, Budget: 45 * time.Second})
}
func newDeletionWorld(p uuid.UUID) *deletionWorld {
	return &deletionWorld{fakeRows: twoProjects(p, uuid.NewV7()), runtime: time.Now()}
}

func TestDeleteRowsCommitsEachBatch(t *testing.T) {
	p, by := uuid.NewV7(), uuid.NewV7()
	at := time.Now()
	w := newDeletionWorld(p)
	want := batches(w.fakeRows, p)
	result, err := w.useCase().Execute(t.Context(), "project", p, at, by)
	if err != nil || !result.Complete || result.Deleted != 13 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if !reflect.DeepEqual(w.calls, want) || w.commits != len(want) || w.begins != w.commits || w.pendingCalls != 0 {
		t.Fatalf("calls=%v commits=%d begins=%d pending=%d want=%v", w.calls, w.commits, w.begins, w.pendingCalls, want)
	}
	stamps := maps.Clone(w.deleted)
	again, err := w.useCase().Execute(t.Context(), "project", p, at.Add(time.Hour), uuid.NewV7())
	if err != nil || !again.Complete || again.Deleted != 0 || !maps.Equal(stamps, w.deleted) {
		t.Fatalf("redo=%+v err=%v", again, err)
	}
	for _, stamp := range stamps {
		if stamp != fmt.Sprintf("%s at %s", by, at.Format(time.RFC3339Nano)) {
			t.Fatal(stamp)
		}
	}
	for _, failure := range []string{"commit", "statement"} {
		t.Run(failure, func(t *testing.T) {
			w := newDeletionWorld(p)
			if failure == "commit" {
				w.failCommit = 2
			} else {
				w.fail = "drafts " + p.String() + " after " + w.rows["drafts"][p][1].String() + " limit 2"
			}
			got, err := w.useCase().Execute(t.Context(), "project", p, at, by)
			if !errors.Is(err, errDisk) || got.Deleted != 2 || got.Complete || w.commits != 1 || w.begins != 2 || len(w.deleted) != 2 {
				t.Fatalf("got=%+v err=%v commits=%d deleted=%v", got, err, w.commits, w.deleted)
			}
		})
	}
}

func TestDeleteRowsBudgetUsesRuntimeElapsed(t *testing.T) {
	for _, days := range []int{-61, 61} {
		for _, pending := range []bool{false, true} {
			for _, scope := range []string{"project", "workspace_drafts"} {
				t.Run(fmt.Sprintf("%d/%t/%s", days, pending, scope), func(t *testing.T) {
					p, by := uuid.NewV7(), uuid.NewV7()
					w := newDeletionWorld(p)
					w.rows["workspace_drafts"] = map[uuid.UUID][]uuid.UUID{p: ids(3)}
					w.advance = 46 * time.Second
					w.pending = pending
					at := w.runtime.Add(time.Duration(days) * 24 * time.Hour)
					got, err := w.useCase().Execute(t.Context(), scope, p, at, by)
					if err != nil || got.Deleted != 2 || got.Complete == pending || w.begins != 1 || w.pendingCalls != 1 {
						t.Fatalf("got=%+v err=%v begins=%d pending=%d", got, err, w.begins, w.pendingCalls)
					}
					if scope == "project" {
						if w.remainder.ProjectID != p || w.remainder.Step != "draft_issues" || w.remainder.After != w.rows["drafts"][p][1] {
							t.Fatalf("remainder=%+v", w.remainder)
						}
					} else if w.draftRemainder.WorkspaceID != p || w.draftRemainder.After != w.rows["workspace_drafts"][p][1] {
						t.Fatal(w.draftRemainder)
					}
					for _, stamp := range w.deleted {
						if !strings.Contains(stamp, at.Format(time.RFC3339Nano)) {
							t.Fatal(stamp)
						}
					}
				})
			}
		}
	}
	t.Run("pending error", func(t *testing.T) {
		p := uuid.NewV7()
		w := newDeletionWorld(p)
		w.advance = time.Minute
		w.pendingErr = errDisk
		got, err := w.useCase().Execute(t.Context(), "project", p, time.Now(), uuid.NewV7())
		if err != errDisk || got.Complete || got.Deleted != 2 {
			t.Fatalf("%+v %v", got, err)
		}
	})
}

func TestDeleteRowsRejectsInvalidAndCancelled(t *testing.T) {
	p, by := uuid.NewV7(), uuid.NewV7()
	at := time.Now()
	for _, v := range []struct {
		scope string
		id    uuid.UUID
		at    time.Time
		by    uuid.UUID
	}{{"other", p, at, by}, {"project", uuid.UUID{}, at, by}, {"project", p, time.Time{}, by}, {"workspace_drafts", p, at, uuid.UUID{}}} {
		w := newDeletionWorld(p)
		if _, err := w.useCase().Execute(t.Context(), v.scope, v.id, v.at, v.by); err == nil || w.begins != 0 {
			t.Fatalf("invalid %+v wrote", v)
		}
	}
	w := newDeletionWorld(p)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := w.useCase().Execute(ctx, "project", p, at, by); err != context.Canceled || w.begins != 0 {
		t.Fatalf("cancel=%v begins=%d", err, w.begins)
	}
}

// A full skipped relation batch must advance by candidates, including at a
// budget boundary; a later owned relation is still reachable on retry.
func TestDeleteRowsSkippedRelationsAdvance(t *testing.T) {
	for _, budget := range []bool{false, true} {
		t.Run(fmt.Sprint(budget), func(t *testing.T) {
			p, q, by := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
			at := time.Now()
			r := ids(3)
			w := newDeletionWorld(p)
			w.rows = map[string]map[uuid.UUID][]uuid.UUID{"relations": {p: r}}
			w.other = map[uuid.UUID]uuid.UUID{r[0]: q, r[1]: q}
			w.deletedProjects = map[uuid.UUID]time.Time{q: at.Add(-time.Hour)}
			if budget {
				w.advance = 9 * time.Second
				w.pending = true
			}
			got, err := w.useCase().Execute(t.Context(), "project", p, at, by)
			if err != nil {
				t.Fatal(err)
			}
			if budget {
				if got.Complete || got.Deleted != 0 || w.remainder.Step != "issue_relations" || w.remainder.After != r[1] || w.pendingCalls != 1 {
					t.Fatalf("%+v remainder=%+v", got, w.remainder)
				}
				w.advance = 0
				got, err = w.useCase().Execute(t.Context(), "project", p, at, by)
			}
			if err != nil || !got.Complete || got.Deleted != 1 || len(w.deleted) != 1 || w.deleted[r[2]] == "" {
				t.Fatalf("%+v %v rows=%v", got, err, w.deleted)
			}
		})
	}
}
