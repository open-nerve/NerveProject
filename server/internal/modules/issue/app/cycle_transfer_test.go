package app

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// cycleTransferRows are the transfer's store as fakes: each call in order, as "name count-or-ids at by"; what each
// write was given; the work items the removal answers; and the failure of a call by its name.
type cycleTransferRows struct {
	ctx        context.Context
	calls      []string
	removed    []CycleTransfer
	joins      []CycleJoins
	activities []ActivityRow
	changed    []ChangedRows
	issues     []uuid.UUID
	fail       map[string]error
}

func (r *cycleTransferRows) call(ctx context.Context, name string, args ...any) error {
	if ctx != r.ctx {
		return errors.New(name + " lost the transfer's transaction's context")
	}
	r.calls = append(r.calls, fmt.Sprint(append([]any{name}, args...)...))
	return r.fail[name]
}

func (r *cycleTransferRows) RemoveTransferredCycleIssues(ctx context.Context, t CycleTransfer) ([]uuid.UUID, error) {
	r.removed = append(r.removed, t)
	return slices.Clone(r.issues), r.call(ctx, "RemoveTransferredCycleIssues", " ", t.From.ID, " ", t.At.Format(time.RFC3339Nano), " ", t.By)
}

func (r *cycleTransferRows) InsertTransferredCycleIssues(ctx context.Context, j CycleJoins) error {
	r.joins = append(r.joins, j)
	return r.call(ctx, "InsertTransferredCycleIssues", " ", j.CycleID, " ", j.At.Format(time.RFC3339Nano), " ", j.By)
}

func (r *cycleTransferRows) InsertActivities(ctx context.Context, rows []ActivityRow) error {
	r.activities = append(r.activities, rows...)
	return r.call(ctx, "InsertActivities", " ", len(rows))
}

func (r *cycleTransferRows) MarkChanged(ctx context.Context, c ChangedRows) error {
	r.changed = append(r.changed, c)
	return r.call(ctx, "MarkChanged", " ", c.IDs, " ", c.At.Format(time.RFC3339Nano), " ", c.By)
}

// transferOf is a transfer of the work items in two states from Sprint 1 to Sprint 2, by an account at a moment of
// microseconds, as planning asks for it (M6 P4 spec 2.6).
func transferOf() CycleTransfer {
	return CycleTransfer{ProjectID: uuid.NewV7(), WorkspaceID: uuid.NewV7(), From: domain.Named{ID: uuid.NewV7(), Name: "Sprint 1"},
		To: domain.Named{ID: uuid.NewV7(), Name: "Sprint 2"}, StateIDs: []uuid.UUID{uuid.NewV7(), uuid.NewV7()}, By: uuid.NewV7(),
		At: time.Date(2026, 11, 10, 9, 30, 0, 123456000, time.UTC)}
}

// A transfer moves the work items of the cycle it leaves into the one it joins, in its transaction (M6 P4 spec 2.6,
// R8): the removal asked of the transfer as planning gave it; then one new row of the joined cycle for each work item
// the removal answers, in its order, each with a fresh id, in the transfer's workspace and project, by its account at
// its moment; then one row of activity for each, in that order, updated, field cycles, from the left cycle's name to
// the joined one's, their ids the old and the new identifier, by the account at the moment; then exactly those work
// items last changed by the transfer. It answers how many moved. Nothing else is written: no event, no other table.
func TestTransferringACyclesWorkItems(t *testing.T) {
	tr := transferOf()
	a, b := uuid.NewV7(), uuid.NewV7()
	rows := &cycleTransferRows{ctx: t.Context(), issues: []uuid.UUID{a, b}}
	n, err := NewCycleTransfers(rows).TransferCycleIssues(rows.ctx, tr)
	if err != nil || n != 2 {
		t.Fatalf("TransferCycleIssues() = %d, %v; want 2", n, err)
	}
	at, by := tr.At.Format(time.RFC3339Nano), tr.By
	want := []string{fmt.Sprint("RemoveTransferredCycleIssues ", tr.From.ID, " ", at, " ", by), fmt.Sprint("InsertTransferredCycleIssues ", tr.To.ID, " ", at, " ", by),
		"InsertActivities 2", fmt.Sprint("MarkChanged ", []uuid.UUID{a, b}, " ", at, " ", by)}
	if !slices.Equal(rows.calls, want) {
		t.Fatalf("calls\n%q\nwant\n%q", rows.calls, want)
	}
	if !reflect.DeepEqual(rows.removed, []CycleTransfer{tr}) {
		t.Errorf("removal asked of %+v, want the transfer %+v", rows.removed, tr)
	}
	j := rows.joins[0]
	if wantJoins := (CycleJoins{IDs: j.IDs, IssueIDs: []uuid.UUID{a, b}, CycleID: tr.To.ID, WorkspaceID: tr.WorkspaceID, ProjectID: tr.ProjectID,
		By: by, At: tr.At}); !reflect.DeepEqual(j, wantJoins) || len(j.IDs) != 2 || slices.Contains(j.IDs, uuid.Nil()) || j.IDs[0] == j.IDs[1] {
		t.Errorf("joins %+v; want %+v, two fresh ids", j, wantJoins)
	}
	field, from, to := "cycles", "Sprint 1", "Sprint 2"
	moved := domain.Activity{Field: &field, Verb: domain.VerbUpdated, OldValue: &from, NewValue: &to, OldIdentifier: &tr.From.ID,
		NewIdentifier: &tr.To.ID, Comment: "updated cycle from "}
	if !reflect.DeepEqual(moved, domain.CycleMoved(tr.From, tr.To)) {
		t.Fatalf("domain.CycleMoved() = %+v, want %+v", domain.CycleMoved(tr.From, tr.To), moved)
	}
	for i, item := range []uuid.UUID{a, b} {
		got := rows.activities[i]
		want := ActivityRow{ID: got.ID, WorkspaceID: tr.WorkspaceID, ProjectID: tr.ProjectID, IssueID: item, Activity: moved, ActorID: &by, At: tr.At}
		if got.ID == uuid.Nil() || slices.Contains(j.IDs, got.ID) || !reflect.DeepEqual(got, want) {
			t.Errorf("activity %d: %+v\nwant %+v", i, got, want)
		}
	}
	if rows.activities[0].ID == rows.activities[1].ID {
		t.Error("two rows of activity share an id")
	}
	if !reflect.DeepEqual(rows.changed, []ChangedRows{{IDs: []uuid.UUID{a, b}, By: by, At: tr.At}}) {
		t.Errorf("changed %+v, want both work items by the transferring account at the transfer", rows.changed)
	}
}

// A transfer that finds no work item to move writes nothing past the removal: no membership, no activity, no work
// item changed. It answers 0, which is no refusal (M6 P4 spec 2.6: 200 with 0).
func TestTransferringNoWorkItem(t *testing.T) {
	rows := &cycleTransferRows{ctx: t.Context()}
	n, err := NewCycleTransfers(rows).TransferCycleIssues(rows.ctx, transferOf())
	if err != nil || n != 0 {
		t.Fatalf("TransferCycleIssues() = %d, %v; want 0", n, err)
	}
	if len(rows.calls) != 1 || rows.joins != nil || rows.activities != nil || rows.changed != nil {
		t.Errorf("calls %q, joins %+v, activity %+v, changed %+v; want the removal alone", rows.calls, rows.joins, rows.activities, rows.changed)
	}
}

// A write that fails stops the transfer with its error, which fails planning's transaction: nothing after it is
// written, and no count is answered.
func TestTransferStopsAtAFailure(t *testing.T) {
	for _, step := range []string{"RemoveTransferredCycleIssues", "InsertTransferredCycleIssues", "InsertActivities", "MarkChanged"} {
		failure := errors.New(step + " failed")
		rows := &cycleTransferRows{ctx: t.Context(), issues: []uuid.UUID{uuid.NewV7()}, fail: map[string]error{step: failure}}
		if n, err := NewCycleTransfers(rows).TransferCycleIssues(rows.ctx, transferOf()); !errors.Is(err, failure) || n != 0 {
			t.Fatalf("%s: TransferCycleIssues() = %d, %v; want 0 and its failure", step, n, err)
		}
		if last := rows.calls[len(rows.calls)-1]; !strings.HasPrefix(last, step) {
			t.Errorf("%s: calls %q, want them to end at it", step, rows.calls)
		}
	}
}
