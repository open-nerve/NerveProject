package app_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// This fixture models transaction commit/rollback, so swallowed errors or a
// single transaction across projects change the observable committed result.
type archiveWorld struct {
	*fakeProject
	candidates     []uuid.UUID
	lockedProjects map[uuid.UUID]app.LockedProject
	results        map[uuid.UUID][]uuid.UUID
	rows           []app.AutoArchiveRows
	activities     []app.ActivityRow
	events         []shared.Event
	committed      []uuid.UUID
	failProject    uuid.UUID
	cancel         context.CancelFunc
}

func archiveSetup() (*archiveWorld, app.AutoArchiveDeps) {
	l := &callLog{}
	f := &archiveWorld{fakeProject: &fakeProject{log: l, states: []app.StateRef{{ID: bob, Group: domain.GroupCompleted}, {ID: alice, Group: domain.GroupCancelled}, {ID: dave, Group: "started"}}}, candidates: []uuid.UUID{web, acme}, lockedProjects: map[uuid.UUID]app.LockedProject{web: {WorkspaceID: acme, ArchiveIn: 2}, acme: {WorkspaceID: acme, ArchiveIn: 1}}, results: map[uuid.UUID][]uuid.UUID{web: {dave, bob}, acme: {alice}}}
	return f, app.AutoArchiveDeps{Policies: f, Projects: f, States: f, Issues: f, Tx: f, Clock: clockAt{at: time.Date(2026, 3, 31, 10, 0, 0, 123456000, time.UTC), log: l}, Events: f}
}
func (f *archiveWorld) ArchiveProjects(ctx context.Context) ([]uuid.UUID, error) {
	return f.candidates, f.log.add(ctx, "Scan")
}
func (f *archiveWorld) Lock(ctx context.Context, id uuid.UUID) (app.LockedProject, bool, error) {
	if f.waited != nil {
		f.waited()
	}
	p, ok := f.lockedProjects[id]
	return p, ok, f.log.add(ctx, "Lock", id)
}
func (f *archiveWorld) AutoArchive(ctx context.Context, r app.AutoArchiveRows) ([]uuid.UUID, error) {
	f.rows = append(f.rows, r)
	if err := f.log.add(ctx, "Update", r.ProjectID); err != nil {
		return nil, err
	}
	if r.ProjectID == f.failProject {
		return nil, errDisk
	}
	ids := f.results[r.ProjectID]
	f.committed = append(f.committed, ids...)
	delete(f.results, r.ProjectID)
	return ids, nil
}
func (f *archiveWorld) InsertActivities(ctx context.Context, rows []app.ActivityRow) error {
	f.activities = append(f.activities, rows...)
	return f.log.add(ctx, "Activities")
}
func (f *archiveWorld) Publish(ctx context.Context, e shared.Event) error {
	f.events = append(f.events, e)
	return f.log.add(ctx, "Event", e.Entity.ID)
}
func (f *archiveWorld) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	oldA, oldE, oldC := len(f.activities), len(f.events), len(f.committed)
	oldR := map[uuid.UUID][]uuid.UUID{}
	for k, v := range f.results {
		oldR[k] = slices.Clone(v)
	}
	err := fakeTx{f.log}.WithinTx(ctx, func(ctx context.Context) error {
		if err := f.log.add(ctx, "Tx"); err != nil {
			return err
		}
		return fn(ctx)
	})
	if err != nil {
		f.activities = f.activities[:oldA]
		f.events = f.events[:oldE]
		f.committed = f.committed[:oldC]
		f.results = oldR
		f.log.calls = append(f.log.calls, "Rollback")
	} else {
		f.log.calls = append(f.log.calls, "Commit")
	}
	if f.cancel != nil {
		f.cancel()
	}
	return err
}

func TestAutoArchiveProjectTransactionsAndSystemIdentity(t *testing.T) {
	f, d := archiveSetup()
	n, err := app.NewAutoArchive(d).Execute(context.Background())
	if n != 3 || err != nil {
		t.Fatalf("Execute = %d,%v", n, err)
	}
	want := []string{"Scan outside tx", "Begin", "Tx", line("Lock", web), line("States", web), "Now", line("Update", web), "Activities", line("Event", bob), line("Event", dave), "Commit", "Begin", "Tx", line("Lock", acme), line("States", acme), "Now", line("Update", acme), "Activities", line("Event", alice), "Commit"}
	if !slices.Equal(f.log.calls, want) {
		t.Fatalf("calls = %v; want %v", f.log.calls, want)
	}
	cutoffs := []time.Time{time.Date(2026, 1, 30, 10, 0, 0, 123456000, time.UTC), time.Date(2026, 3, 1, 10, 0, 0, 123456000, time.UTC)}
	for i, r := range f.rows {
		if !r.Cutoff.Equal(cutoffs[i]) || !r.At.Equal(d.Clock.(clockAt).at) || !slices.Equal(r.StateIDs, []uuid.UUID{bob, alice}) {
			t.Errorf("row %d = %+v", i, r)
		}
	}
	seen := map[uuid.UUID]bool{}
	for i, a := range f.activities {
		e := f.events[i]
		if a.ActorID != nil || a.Activity.Field == nil || *a.Activity.Field != "archived_at" || a.Activity.NewValue == nil || *a.Activity.NewValue != "archive" || a.Activity.OldValue != nil || !a.At.Equal(d.Clock.(clockAt).at) {
			t.Errorf("system activity = %+v", a)
		}
		if e.ActorID != nil || e.Type != domain.EventIssueArchived || e.Entity != (shared.EntityRef{Kind: "issue", ID: a.IssueID}) || !e.OccurredAt.Equal(a.At) || e.ID == (uuid.UUID{}) || e.ID == a.IssueID || seen[e.ID] || !reflect.DeepEqual(e.Changes, []shared.Change{{Field: "archived_at", New: eventString("archive")}}) {
			t.Errorf("event = %+v", e)
		}
		seen[e.ID] = true
	}
	n, err = app.NewAutoArchive(d).Execute(context.Background())
	if n != 0 || err != nil || len(f.activities) != 3 || len(f.events) != 3 {
		t.Fatalf("retry = %d,%v; activities/events=%d/%d", n, err, len(f.activities), len(f.events))
	}
}
func eventString(s string) *string { return &s }

func TestAutoArchiveRechecksPolicyUnderLock(t *testing.T) {
	for _, mode := range []string{"deleted", "archived", "disabled", "changed", "no terminal states", "no projects"} {
		t.Run(mode, func(t *testing.T) {
			f, d := archiveSetup()
			f.candidates = []uuid.UUID{web}
			f.waited = func() {
				p := f.lockedProjects[web]
				switch mode {
				case "deleted":
					delete(f.lockedProjects, web)
					return
				case "archived":
					p.Archived = true
				case "disabled":
					p.ArchiveIn = 0
				case "changed":
					p.ArchiveIn = 1
				case "no terminal states":
					f.states = []app.StateRef{{ID: dave, Group: "started"}}
				}
				f.lockedProjects[web] = p
			}
			if mode == "no projects" {
				f.candidates = nil
			}
			n, err := app.NewAutoArchive(d).Execute(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if mode == "changed" {
				if n != 2 || len(f.rows) != 1 || !f.rows[0].Cutoff.Equal(time.Date(2026, 3, 1, 10, 0, 0, 123456000, time.UTC)) {
					t.Fatalf("policy not read under lock: %d %+v", n, f.rows)
				}
			} else if n != 0 || len(f.rows) != 0 || slices.Contains(f.log.calls, "Now") {
				t.Fatalf("skip = %d, rows %v, calls %v", n, f.rows, f.log.calls)
			}
		})
	}
}
func TestAutoArchivePropagatesEveryPortFailure(t *testing.T) {
	for _, port := range []string{"Scan", "Tx", "Lock", "States", "Update", "Activities", line("Event", bob), line("Event", dave)} {
		t.Run(port, func(t *testing.T) {
			f, d := archiveSetup()
			f.log.fail = port
			n, err := app.NewAutoArchive(d).Execute(context.Background())
			if n != 0 || !errors.Is(err, errDisk) || len(f.committed) != 0 || len(f.activities) != 0 || len(f.events) != 0 {
				t.Fatalf("failure = %d,%v, committed=%v activities=%v events=%v", n, err, f.committed, f.activities, f.events)
			}
		})
	}
}
func TestAutoArchivePreservesCommittedProjectsOnFailureAndRetry(t *testing.T) {
	f, d := archiveSetup()
	f.failProject = acme
	n, err := app.NewAutoArchive(d).Execute(context.Background())
	if n != 2 || !errors.Is(err, errDisk) || len(f.activities) != 2 || len(f.events) != 2 {
		t.Fatalf("partial = %d,%v, activities/events=%d/%d", n, err, len(f.activities), len(f.events))
	}
	f.failProject = uuid.UUID{}
	n, err = app.NewAutoArchive(d).Execute(context.Background())
	if n != 1 || err != nil || len(f.activities) != 3 || len(f.events) != 3 {
		t.Fatalf("retry = %d,%v activities/events=%d/%d", n, err, len(f.activities), len(f.events))
	}
}
func TestAutoArchiveCancellation(t *testing.T) {
	for _, before := range []bool{true, false} {
		t.Run(map[bool]string{true: "before scan", false: "between projects"}[before], func(t *testing.T) {
			f, d := archiveSetup()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if before {
				cancel()
			} else {
				f.cancel = cancel
			}
			n, err := app.NewAutoArchive(d).Execute(ctx)
			want := 2
			if before {
				want = 0
			}
			if n != want || !errors.Is(err, context.Canceled) {
				t.Fatalf("cancel = %d,%v; want %d,Canceled", n, err, want)
			}
			if !before && len(f.rows) != 1 {
				t.Fatal("cancelled next project still updated")
			}
		})
	}
}
