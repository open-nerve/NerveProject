package app_test

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type recordedEvents struct {
	t      *testing.T
	rows   []shared.Event
	fail   error
	before func(shared.Event)
}

func (p *recordedEvents) Publish(ctx context.Context, e shared.Event) error {
	p.t.Helper()
	if ctx.Value(inTxKey{}) != true {
		p.t.Fatal("event lost business transaction context")
	}
	if p.before != nil {
		p.before(e)
	}
	p.rows = append(p.rows, e)
	return p.fail
}
func eventCreate(w *world, p shared.EventPublisher) *app.CreateIssue {
	return app.NewCreateIssue(app.CreateIssueDeps{Locks: w.locks, Creation: app.NewIssueCreation(app.IssueCreationDeps{Events: p, Issues: w.issues, Sequences: w.project, States: w.project, Labels: w.project, Members: w.project, Profiles: w.profiles, Clock: clockAt{at: clockNow, log: w.log}, Sanitizer: descriptionSanitizer{}})})
}
func eventUpdate(w *world, p shared.EventPublisher) *app.UpdateIssue {
	return app.NewUpdateIssue(app.UpdateIssueDeps{Events: p, Locks: w.locks, Issues: w.issues, States: w.project, Labels: w.project, Members: w.project, Profiles: w.profiles, Clock: clockAt{at: clockNow, log: w.log}, Sanitizer: descriptionSanitizer{}})
}
func evptr[T any](v T) *T { return &v }
func wantIssueEvent(t *testing.T, p *recordedEvents, kind string, id uuid.UUID, changes []shared.Change) shared.Event {
	t.Helper()
	if len(p.rows) != 1 {
		t.Fatalf("published %d events, want one %s", len(p.rows), kind)
	}
	e := p.rows[0]
	if e.ID == (uuid.UUID{}) || e.ID == id {
		t.Errorf("event identity %s must be nonzero and distinct from entity %s", e.ID, id)
	}
	if e.Type != kind || e.Entity != (shared.EntityRef{Kind: "issue", ID: id}) || e.WorkspaceID != acme || e.ProjectID != web || e.ActorID == nil || *e.ActorID != alice || !e.OccurredAt.Equal(clockNow) {
		t.Errorf("event envelope = %+v", e)
	}
	if !reflect.DeepEqual(e.Changes, changes) {
		t.Errorf("changes = %+v\nwant %+v", e.Changes, changes)
	}
	return e
}
func TestIssueCreatedEvent(t *testing.T) {
	for _, full := range []bool{false, true} {
		t.Run(line("full", full), func(t *testing.T) {
			_, w := newCreate()
			p := &recordedEvents{t: t, before: func(e shared.Event) {
				if len(w.issues.versions) != 1 {
					t.Fatal("create event preceded initial version")
				}
			}}
			in := domain.NewIssue{Name: "  Fix  "}
			want := []shared.Change{
				{Field: "name", New: evptr("Fix")}, {Field: "description", New: evptr("<p></p>")},
				{Field: "state", New: evptr("Backlog"), NewID: &backlog.ID}, {Field: "priority", New: evptr("none")},
				{Field: "start_date"}, {Field: "target_date"}, {Field: "parent"}, {Field: "sort_order", New: evptr("65535")},
				{Field: "sequence_id", New: evptr("7")}, {Field: "completed_at"},
			}
			if full {
				parent := uuid.NewV7()
				w.issues.others = map[uuid.UUID]domain.Issue{parent: {ID: parent, ProjectID: web, SequenceID: 3}}
				w.project.locked.DefaultAssigneeID = &bob
				in.DescriptionHTML = evptr("<p>Clean</p>")
				in.StateID = &done.ID
				in.Priority = evptr(domain.PriorityUrgent)
				in.StartDate, in.TargetDate = october(18), october(19)
				in.ParentID = &parent
				in.LabelIDs = []uuid.UUID{ui, bug}
				want = []shared.Change{
					{Field: "name", New: evptr("Fix")}, {Field: "description", New: evptr("<p>Clean</p>")},
					{Field: "state", New: evptr("Done"), NewID: &done.ID}, {Field: "priority", New: evptr("urgent")},
					{Field: "start_date", New: evptr("2026-10-18")}, {Field: "target_date", New: evptr("2026-10-19")},
					{Field: "parent", New: evptr("WEB-3"), NewID: &parent}, {Field: "sort_order", New: evptr("95535")},
					{Field: "sequence_id", New: evptr("7")}, {Field: "completed_at", New: evptr("2026-10-17T10:00:00.123456789Z")},
					{Field: "assignees", New: evptr("bob"), NewID: &bob}, {Field: "labels", New: evptr("Bug"), NewID: &bug}, {Field: "labels", New: evptr("UI"), NewID: &ui},
				}
			}
			got, err := eventCreate(w, p).Execute(as(alice), web, in)
			if err != nil {
				t.Fatal(err)
			}
			wantIssueEvent(t, p, "issue.created", got.ID, want)
			if n := eventClockReads(w); n != 1 {
				t.Fatalf("clock reads = %d, want one", n)
			}
		})
	}
}

func TestIssueUpdatedEvent(t *testing.T) {
	_, w, id := newUpdate()
	parent, oldParent := uuid.NewV7(), uuid.NewV7()
	w.issues.issue.ParentID = &oldParent
	w.issues.issue.DescriptionHTML = "<p>Before</p>"
	w.issues.issue.StartDate, w.issues.issue.TargetDate = october(18), october(19)
	w.issues.others = map[uuid.UUID]domain.Issue{parent: {ID: parent, ProjectID: web, SequenceID: 9}, oldParent: {ID: oldParent, ProjectID: web, SequenceID: 3}}
	p := &recordedEvents{t: t}
	_, err := eventUpdate(w, p).Execute(as(alice), id, domain.IssuePatch{Name: evptr("Mend"), DescriptionHTML: evptr("<p>After</p>"), StateID: &done.ID, Priority: evptr(domain.PriorityLow), SortOrder: evptr(0.0), SetStartDate: true, StartDate: october(17), SetTargetDate: true, SetParentID: true, ParentID: &parent, AssigneeIDs: set(carol, bob), LabelIDs: set(ui)})
	if err != nil {
		t.Fatal(err)
	}
	wantIssueEvent(t, p, "issue.updated", id, []shared.Change{
		{Field: "name", Old: evptr("Fix"), New: evptr("Mend")}, {Field: "description", Old: evptr("<p>Before</p>"), New: evptr("<p>After</p>")},
		{Field: "parent", Old: evptr("WEB-3"), New: evptr("WEB-9"), OldID: &oldParent, NewID: &parent},
		{Field: "priority", Old: evptr("high"), New: evptr("low")}, {Field: "state", Old: evptr("Backlog"), New: evptr("Done"), OldID: &backlog.ID, NewID: &done.ID},
		{Field: "start_date", Old: evptr("2026-10-18"), New: evptr("2026-10-17")}, {Field: "target_date", Old: evptr("2026-10-19"), New: evptr("")},
		{Field: "labels", Old: evptr(""), New: evptr("UI"), NewID: &ui}, {Field: "labels", Old: evptr("Bug"), New: evptr(""), OldID: &bug},
		{Field: "assignees", Old: evptr(""), New: evptr("Carol C."), NewID: &carol}, {Field: "assignees", Old: evptr("out"), New: evptr(""), OldID: &out},
		{Field: "sort_order", Old: evptr("65535"), New: evptr("0")},
	})
	if n := eventClockReads(w); n != 1 {
		t.Fatalf("clock reads = %d, want one", n)
	}
}

func TestIssueUpdatedEventEachField(t *testing.T) {
	for _, tt := range []struct {
		name  string
		patch domain.IssuePatch
		want  []shared.Change
	}{
		{"name", domain.IssuePatch{Name: evptr("Mend")}, []shared.Change{{Field: "name", Old: evptr("Fix"), New: evptr("Mend")}}},
		{"priority", domain.IssuePatch{Priority: evptr(domain.PriorityNone)}, []shared.Change{{Field: "priority", Old: evptr("high"), New: evptr("none")}}},
		{"state", domain.IssuePatch{StateID: &done.ID}, []shared.Change{{Field: "state", Old: evptr("Backlog"), New: evptr("Done"), OldID: &backlog.ID, NewID: &done.ID}}},
		{"start date", domain.IssuePatch{SetStartDate: true, StartDate: october(18)}, []shared.Change{{Field: "start_date", Old: evptr(""), New: evptr("2026-10-18")}}},
		{"target date", domain.IssuePatch{SetTargetDate: true, TargetDate: october(19)}, []shared.Change{{Field: "target_date", Old: evptr(""), New: evptr("2026-10-19")}}},
		{"assignees", domain.IssuePatch{AssigneeIDs: set(bob, carol)}, []shared.Change{{Field: "assignees", Old: evptr(""), New: evptr("Carol C."), NewID: &carol}, {Field: "assignees", Old: evptr("out"), New: evptr(""), OldID: &out}}},
		{"labels", domain.IssuePatch{LabelIDs: set(ui)}, []shared.Change{{Field: "labels", Old: evptr(""), New: evptr("UI"), NewID: &ui}, {Field: "labels", Old: evptr("Bug"), New: evptr(""), OldID: &bug}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, w, id := newUpdate()
			p := &recordedEvents{t: t}
			if _, err := eventUpdate(w, p).Execute(as(alice), id, tt.patch); err != nil {
				t.Fatal(err)
			}
			wantIssueEvent(t, p, "issue.updated", id, tt.want)
		})
	}
}

func TestIssueUpdatedEventSortOrder(t *testing.T) {
	_, w, id := newUpdate()
	p := &recordedEvents{t: t}
	uc := eventUpdate(w, p)
	seen := map[uuid.UUID]bool{}
	for _, tt := range []struct {
		old, new         float64
		oldText, newText string
	}{{65535, 0, "65535", "0"}, {0, 1, "0", "1"}, {1, math.Nextafter(1, 2), "1", "1.0000000000000002"}} {
		p.rows = nil
		if _, err := uc.Execute(as(alice), id, domain.IssuePatch{SortOrder: &tt.new}); err != nil {
			t.Fatal(err)
		}
		e := wantIssueEvent(t, p, "issue.updated", id, []shared.Change{{Field: "sort_order", Old: &tt.oldText, New: &tt.newText}})
		if seen[e.ID] {
			t.Fatal("continuous writes reused event identity")
		}
		seen[e.ID] = true
	}
	if len(w.issues.activities) != 0 {
		t.Fatal("sort-only event added activity")
	}
}

func TestIssueUpdatedEventDescriptionWindow(t *testing.T) {
	for _, elapsed := range []time.Duration{599 * time.Second, 600 * time.Second, 600*time.Second + time.Microsecond} {
		t.Run(line("elapsed", elapsed), func(t *testing.T) {
			_, w, id := newUpdate()
			w.issues.issue.DescriptionHTML = "<p>Old</p>"
			versionID := uuid.NewV7()
			w.issues.versions = []domain.DescriptionVersion{{ID: versionID, OwnedByID: alice, LastSavedAt: clockNow.Add(-elapsed)}}
			p := &recordedEvents{t: t, before: func(e shared.Event) {
				v := w.issues.versions[len(w.issues.versions)-1]
				if v.HTML != "<p>New</p>" {
					t.Fatal("published before description version")
				}
			}}
			if _, err := eventUpdate(w, p).Execute(as(alice), id, domain.IssuePatch{DescriptionHTML: evptr("<p>New</p>")}); err != nil {
				t.Fatal(err)
			}
			wantIssueEvent(t, p, "issue.updated", id, []shared.Change{{Field: "description", Old: evptr("<p>Old</p>"), New: evptr("<p>New</p>")}})
			if merged := w.issues.versions[len(w.issues.versions)-1].ID == versionID; merged != (elapsed <= 600*time.Second) {
				t.Fatalf("version merged=%v at %s", merged, elapsed)
			}
		})
	}
}

func TestIssueUpdatedEventNoopAndSkip(t *testing.T) {
	for _, tt := range []struct {
		name   string
		patch  domain.IssuePatch
		events int
	}{
		{"noop", domain.IssuePatch{Name: evptr("Fix")}, 0},
		{"same sort", domain.IssuePatch{SortOrder: evptr(65535.0)}, 0},
		{"same HTML", domain.IssuePatch{DescriptionHTML: evptr("<p>Old</p>")}, 0},
		{"migration ids", domain.IssuePatch{DescriptionHTML: evptr(`<p data-id="new">Old</p>`), SkipActivity: evptr(true)}, 0},
		{"skip content changed", domain.IssuePatch{DescriptionHTML: evptr("<p>New</p>"), SkipActivity: evptr(true)}, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, w, id := newUpdate()
			w.issues.issue.DescriptionHTML = "<p>Old</p>"
			p := &recordedEvents{t: t}
			if _, err := eventUpdate(w, p).Execute(as(alice), id, tt.patch); err != nil {
				t.Fatal(err)
			}
			if len(p.rows) != tt.events {
				t.Fatalf("events=%d want %d", len(p.rows), tt.events)
			}
			if tt.events == 1 {
				wantIssueEvent(t, p, "issue.updated", id, []shared.Change{{Field: "description", Old: evptr("<p>Old</p>"), New: evptr("<p>New</p>")}})
			}
		})
	}
}

func TestIssueCreatedEventUniqueAndFailures(t *testing.T) {
	seen := map[uuid.UUID]bool{}
	for range 3 {
		_, w := newCreate()
		p := &recordedEvents{t: t}
		got, err := eventCreate(w, p).Execute(as(alice), web, domain.NewIssue{Name: "Fix"})
		if err != nil {
			t.Fatal(err)
		}
		if len(p.rows) != 1 {
			t.Fatal("missing create event")
		}
		e := p.rows[0]
		if e.ID == (uuid.UUID{}) || e.ID == got.ID || seen[e.ID] {
			t.Fatal("invalid or reused event ID")
		}
		seen[e.ID] = true
	}
	for _, create := range []bool{false, true} {
		t.Run(line("create", create), func(t *testing.T) {
			_, w, id := newUpdate()
			p := &recordedEvents{t: t, fail: errDisk}
			var err error
			if create {
				_, w = newCreate()
				_, err = eventCreate(w, p).Execute(as(alice), web, domain.NewIssue{Name: "Fix"})
			} else {
				_, err = eventUpdate(w, p).Execute(as(alice), id, domain.IssuePatch{Name: evptr("Mend")})
			}
			if !errors.Is(err, errDisk) {
				t.Fatalf("publisher failure lost: %v", err)
			}
		})
	}
	for _, create := range []bool{false, true} {
		t.Run(line("version fails create", create), func(t *testing.T) {
			_, w, id := newUpdate()
			if create {
				_, w = newCreate()
			}
			w.log.fail = "InsertDescriptionVersion"
			p := &recordedEvents{t: t}
			var err error
			if create {
				_, err = eventCreate(w, p).Execute(as(alice), web, domain.NewIssue{Name: "Fix"})
			} else {
				_, err = eventUpdate(w, p).Execute(as(alice), id, domain.IssuePatch{DescriptionHTML: evptr("<p>New</p>")})
			}
			if !errors.Is(err, errDisk) || len(p.rows) != 0 {
				t.Fatalf("version failure=%v published=%d", err, len(p.rows))
			}
		})
	}
}

func eventClockReads(w *world) int {
	n := 0
	for _, call := range w.log.calls {
		if call == "Now" {
			n++
		}
	}
	return n
}

func TestIssueUpdatedEventParent(t *testing.T) {
	for _, remove := range []bool{false, true} {
		t.Run(line("remove", remove), func(t *testing.T) {
			_, w, id := newUpdate()
			parent := uuid.NewV7()
			w.issues.others = map[uuid.UUID]domain.Issue{parent: {ID: parent, ProjectID: web, SequenceID: 3}}
			patch := domain.IssuePatch{SetParentID: true, ParentID: &parent}
			want := []shared.Change{{Field: "parent", Old: evptr(""), New: evptr("WEB-3"), NewID: &parent}}
			if remove {
				w.issues.issue.ParentID = &parent
				patch.ParentID = nil
				want = []shared.Change{{Field: "parent", Old: evptr("WEB-3"), New: evptr(""), OldID: &parent}}
			}
			p := &recordedEvents{t: t}
			if _, err := eventUpdate(w, p).Execute(as(alice), id, patch); err != nil {
				t.Fatal(err)
			}
			wantIssueEvent(t, p, "issue.updated", id, want)
		})
	}
}
