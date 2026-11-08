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
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// membershipRemovalRows are the memberships' store as fakes: each call in order, as "name target-or-count at by"; the
// rows each write was given; the work items a removal answers; and the failure of a call by its name.
type membershipRemovalRows struct {
	ctx        context.Context
	calls      []string
	activities []ActivityRow
	changed    []ChangedRows
	issues     []uuid.UUID
	fail       map[string]error
}

func (r *membershipRemovalRows) call(ctx context.Context, name string, args ...any) error {
	if ctx != r.ctx {
		return errors.New(name + " lost the deleting transaction's context")
	}
	r.calls = append(r.calls, fmt.Sprint(append([]any{name}, args...)...))
	return r.fail[name]
}

func (r *membershipRemovalRows) RemoveCycleMemberships(ctx context.Context, cycle uuid.UUID, at time.Time, by uuid.UUID) ([]uuid.UUID, error) {
	return slices.Clone(r.issues), r.call(ctx, "RemoveCycleMemberships", " ", cycle, " ", at.Format(time.RFC3339Nano), " ", by)
}

func (r *membershipRemovalRows) RemoveModuleMemberships(ctx context.Context, module uuid.UUID, at time.Time, by uuid.UUID) ([]uuid.UUID, error) {
	return slices.Clone(r.issues), r.call(ctx, "RemoveModuleMemberships", " ", module, " ", at.Format(time.RFC3339Nano), " ", by)
}

func (r *membershipRemovalRows) InsertActivities(ctx context.Context, rows []ActivityRow) error {
	r.activities = append(r.activities, rows...)
	return r.call(ctx, "InsertActivities", " ", len(rows))
}

func (r *membershipRemovalRows) MarkChanged(ctx context.Context, c ChangedRows) error {
	r.changed = append(r.changed, c)
	return r.call(ctx, "MarkChanged", " ", len(c.IDs), " ", c.At.Format(time.RFC3339Nano), " ", c.By)
}

// deletionOf is planning's event of the deletion of a cycle, kind "cycle", or a module, named name, by an account at
// a moment of microseconds, as planning publishes it (M6 P3a spec 2.5).
func deletionOf(kind, name string) shared.Event {
	by := uuid.NewV7()
	return shared.Event{ID: uuid.NewV7(), Type: "planning." + kind + "_deleted", OccurredAt: time.Date(2026, 11, 8, 9, 30, 0, 123456000, time.UTC),
		WorkspaceID: uuid.NewV7(), ProjectID: uuid.NewV7(), ActorID: &by, Entity: shared.EntityRef{Kind: kind, ID: uuid.NewV7()},
		Changes: []shared.Change{{Field: "name", Old: &name}}}
}

// A cycle's or a module's deletion takes the work items out of it in the deleting transaction (M6 P3a spec 2.5, R6):
// its undeleted memberships removed at the deletion's moment by its actor, not the subscriber's own; then one row of
// activity for each work item that was in it, in the order the removal answers them, the row of its leaving the
// target named as the event names it, by the actor at that moment, in the event's workspace and project; then those
// work items last changed by the deletion. Nothing else is written: no event, no other table.
func TestMembershipRemovalsOfADeletedCycleOrModule(t *testing.T) {
	for _, kind := range []string{"cycle", "module"} {
		t.Run(kind, func(t *testing.T) {
			e := deletionOf(kind, "Sprint")
			a, b := uuid.NewV7(), uuid.NewV7()
			rows := &membershipRemovalRows{ctx: t.Context(), issues: []uuid.UUID{a, b}}
			if err := NewMembershipRemovals(rows).Handle(rows.ctx, e); err != nil {
				t.Fatal(err)
			}
			at, by := e.OccurredAt.Format(time.RFC3339Nano), *e.ActorID
			remove := map[string]string{"cycle": "RemoveCycleMemberships", "module": "RemoveModuleMemberships"}[kind]
			want := []string{fmt.Sprint(remove, " ", e.Entity.ID, " ", at, " ", by), "InsertActivities 2", fmt.Sprint("MarkChanged 2 ", at, " ", by)}
			if !slices.Equal(rows.calls, want) {
				t.Fatalf("calls\n%q\nwant\n%q", rows.calls, want)
			}
			left := domain.Named{ID: e.Entity.ID, Name: "Sprint"}
			activity := map[string]domain.Activity{"cycle": domain.CycleLeft(left), "module": domain.ModuleLeft(left)}[kind]
			for i, item := range []uuid.UUID{a, b} {
				got := rows.activities[i]
				want := ActivityRow{ID: got.ID, WorkspaceID: e.WorkspaceID, ProjectID: e.ProjectID, IssueID: item, Activity: activity, ActorID: &by,
					At: e.OccurredAt}
				if got.ID == uuid.Nil() || !reflect.DeepEqual(got, want) {
					t.Errorf("activity %d: %+v\nwant %+v", i, got, want)
				}
			}
			if rows.activities[0].ID == rows.activities[1].ID {
				t.Error("two rows of activity share an id")
			}
			if !reflect.DeepEqual(rows.changed, []ChangedRows{{IDs: []uuid.UUID{a, b}, By: by, At: e.OccurredAt}}) {
				t.Errorf("changed %+v, want both work items by the deleter at the deletion", rows.changed)
			}
		})
	}
}

// A cycle or a module no undeleted work item is in writes nothing past the removal, which finds no membership: no
// activity, no work item changed.
func TestMembershipRemovalsOfAnEmptyTarget(t *testing.T) {
	for _, kind := range []string{"cycle", "module"} {
		rows := &membershipRemovalRows{ctx: t.Context()}
		if err := NewMembershipRemovals(rows).Handle(rows.ctx, deletionOf(kind, "Sprint")); err != nil {
			t.Fatal(err)
		}
		if len(rows.calls) != 1 || rows.activities != nil || rows.changed != nil {
			t.Errorf("%s: calls %q, activity %+v, changed %+v; want the removal alone", kind, rows.calls, rows.activities, rows.changed)
		}
	}
}

// A write that fails stops the subscriber with its error, which fails the deletion that published the event: nothing
// after it is written.
func TestMembershipRemovalsStopAtAFailure(t *testing.T) {
	for _, step := range []string{"RemoveCycleMemberships", "InsertActivities", "MarkChanged"} {
		failure := errors.New(step + " failed")
		rows := &membershipRemovalRows{ctx: t.Context(), issues: []uuid.UUID{uuid.NewV7()}, fail: map[string]error{step: failure}}
		if err := NewMembershipRemovals(rows).Handle(rows.ctx, deletionOf("cycle", "Sprint")); !errors.Is(err, failure) {
			t.Fatalf("%s: Handle() = %v, want its failure", step, err)
		}
		if last := rows.calls[len(rows.calls)-1]; !strings.HasPrefix(last, step) {
			t.Errorf("%s: calls %q, want them to end at it", step, rows.calls)
		}
	}
}

// The payload is read strictly, before any write (precedents deletedProjects, attachmentIssue): an event of another
// type, an entity not of its kind, a zero id, workspace, project, actor or moment, and changes other than the one name
// before the deletion are an error, which fails the deletion.
func TestMembershipRemovalsRejectMalformed(t *testing.T) {
	for _, tc := range []struct {
		name  string
		alter func(*shared.Event)
	}{
		{"unknown type", func(e *shared.Event) { e.Type = "planning.cycle_archived" }},
		{"module's type on a cycle", func(e *shared.Event) { e.Type = "planning.module_deleted" }},
		{"wrong entity", func(e *shared.Event) { e.Entity.Kind = "module" }},
		{"missing event", func(e *shared.Event) { e.ID = uuid.Nil() }},
		{"zero entity", func(e *shared.Event) { e.Entity.ID = uuid.Nil() }},
		{"zero workspace", func(e *shared.Event) { e.WorkspaceID = uuid.Nil() }},
		{"zero project", func(e *shared.Event) { e.ProjectID = uuid.Nil() }},
		{"missing actor", func(e *shared.Event) { e.ActorID = nil }},
		{"zero actor", func(e *shared.Event) { id := uuid.Nil(); e.ActorID = &id }},
		{"zero time", func(e *shared.Event) { e.OccurredAt = time.Time{} }},
		{"missing change", func(e *shared.Event) { e.Changes = nil }},
		{"extra change", func(e *shared.Event) { e.Changes = append(e.Changes, e.Changes[0]) }},
		{"wrong field", func(e *shared.Event) { e.Changes[0].Field = "title" }},
		{"missing name", func(e *shared.Event) { e.Changes[0].Old = nil }},
		{"empty name", func(e *shared.Event) { empty := ""; e.Changes[0].Old = &empty }},
		{"new name", func(e *shared.Event) { e.Changes[0].New = e.Changes[0].Old }},
		{"old id", func(e *shared.Event) { e.Changes[0].OldID = &e.Entity.ID }},
		{"new id", func(e *shared.Event) { e.Changes[0].NewID = &e.Entity.ID }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := deletionOf("cycle", "Sprint")
			tc.alter(&e)
			rows := &membershipRemovalRows{ctx: t.Context(), issues: []uuid.UUID{uuid.NewV7()}}
			if err := NewMembershipRemovals(rows).Handle(rows.ctx, e); err == nil || rows.calls != nil {
				t.Fatalf("malformed event reached storage: %v %q", err, rows.calls)
			}
		})
	}
	if !reflect.DeepEqual(NewMembershipRemovals(nil).Types, []string{"planning.cycle_deleted", "planning.module_deleted"}) {
		t.Fatal("subscriber omitted an event")
	}
}
