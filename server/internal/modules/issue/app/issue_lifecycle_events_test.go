package app_test

import (
	"errors"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestIssueDeletionEvents(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(line("publisher fails", fail), func(t *testing.T) {
			_, w, id := newDelete()
			w.issues.subtree = []uuid.UUID{uuid.NewV7(), id, uuid.NewV7()}
			p := &recordedEvents{t: t, before: func(shared.Event) {
				if len(w.issues.deletions) != 11 || w.issues.relationWrites != 1 {
					t.Fatal("event preceded complete subtree batch writes")
				}
			}}
			if fail {
				p.fail = errDisk
			}
			uc := app.NewDeleteIssue(app.DeleteIssueDeps{Events: p, Locks: w.locks, Rows: w.issues, Relations: app.NewRelationDeletions(w.issues, w.project), Clock: clockAt{at: clockNow, log: w.log}})
			err := uc.Execute(as(alice), id)
			if fail {
				if !errors.Is(err, errDisk) || len(p.rows) != 1 {
					t.Fatalf("failure=%v events=%d", err, len(p.rows))
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(p.rows) != len(w.issues.subtree) {
				t.Fatalf("deleted events=%d want %d", len(p.rows), len(w.issues.subtree))
			}
			seen := map[uuid.UUID]bool{}
			for n, e := range p.rows {
				wantIssueEvent(t, &recordedEvents{rows: []shared.Event{e}}, "issue.deleted", w.issues.subtree[n], nil)
				if seen[e.ID] {
					t.Fatal("subtree events reused identity")
				}
				seen[e.ID] = true
			}
			if eventClockReads(w) != 1 {
				t.Fatal("deletion must share one business clock reading")
			}
		})
	}
}

func TestIssueArchivalEvents(t *testing.T) {
	for _, on := range []bool{true, false} {
		for _, noop := range []bool{false, true} {
			for _, fail := range []bool{false, true} {
				t.Run(line("archive noop fail", []bool{on, noop, fail}), func(t *testing.T) {
					_, _, w, id := newArchive()
					if on == noop {
						w.issues.archivedAt = &earlier
					}
					p := &recordedEvents{t: t, before: func(shared.Event) {
						if w.issues.archival == nil || len(w.issues.activities) != 1 {
							t.Fatal("event preceded archival and activity writes")
						}
					}}
					if fail {
						p.fail = errDisk
					}
					d := app.ArchiveIssueDeps{Events: p, Locks: w.locks, Issues: w.issues, States: w.project, Clock: clockAt{at: clockNow, log: w.log}}
					execute := app.NewUnarchiveIssue(d).Execute
					kind := "issue.unarchived"
					changes := []shared.Change{{Field: "archived_at", Old: evptr("archive"), New: evptr("restore")}}
					if on {
						execute = app.NewArchiveIssue(d).Execute
						kind = "issue.archived"
						changes = []shared.Change{{Field: "archived_at", New: evptr("manual_archive")}}
					}
					_, err := execute(as(alice), id)
					if noop {
						if err != nil || len(p.rows) != 0 || eventClockReads(w) != 0 {
							t.Fatalf("noop err=%v events=%d clock=%d", err, len(p.rows), eventClockReads(w))
						}
						return
					}
					if fail {
						if !errors.Is(err, errDisk) {
							t.Fatalf("publisher error=%v", err)
						}
					} else if err != nil {
						t.Fatal(err)
					}
					wantIssueEvent(t, p, kind, id, changes)
					if eventClockReads(w) != 1 {
						t.Fatal("archival must share one business clock reading")
					}
				})
			}
		}
	}
}
