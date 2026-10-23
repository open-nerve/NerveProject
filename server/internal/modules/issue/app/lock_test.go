package app_test

import (
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// gated rebuilds w's locks with a write gate that logs, and refuses each
// write with refuse, when set: the use cases' creation and update over
// them.
func gated(w *world, refuse error) (*app.CreateIssue, *app.UpdateIssue) {
	w.locks = app.NewLocks(w.project, w.issues, w.auth, fakeGate{log: w.log, refuse: refuse}, fakeTx{log: w.log})
	create := app.NewCreateIssue(app.CreateIssueDeps{Locks: w.locks, Issues: w.issues, Sequences: w.project, States: w.project,
		Labels: w.project, Members: w.project, Profiles: w.profiles, Clock: clockAt{at: clockNow, log: w.log}})
	update := app.NewUpdateIssue(app.UpdateIssueDeps{Locks: w.locks, Issues: w.issues, States: w.project, Labels: w.project, Members: w.project,
		Profiles: w.profiles, Clock: clockAt{at: clockNow, log: w.log}})
	return create, update
}

// A write enters its project's write gate after its precheck, outside any
// transaction, holding nothing, and begins its transaction only then (M4
// design 3.5, 11.11); it leaves the gate once its transaction has ended,
// after its last statement. A creation enters the gate of the project it
// names, an update that of its work item's project.
func TestAWriteEntersItsProjectsGateBetweenItsPrecheckAndItsTransaction(t *testing.T) {
	_, w, _ := newUpdate()
	create, _ := gated(w, nil)
	if _, err := create.Execute(as(alice), web, domain.NewIssue{Name: "Fix"}); err != nil {
		t.Fatal(err)
	}
	if want := []string{line("Find", web, "outside tx"), line("Authorize", alice, domain.ActionCreate, acme, web, "outside tx"),
		line("Enter", web, "outside tx"), "Begin", line("Lock", web)}; !slices.Equal(w.log.calls[:5], want) ||
		w.log.calls[len(w.log.calls)-1] != line("Leave", web) || slices.Index(w.log.calls, line("Leave", web)) != len(w.log.calls)-1 {
		t.Errorf("a creation's calls:\n%q\nwant them to begin %q and end with Leave web alone", w.log.calls, want)
	}

	_, w, id := newUpdate()
	_, update := gated(w, nil)
	if _, err := update.Execute(as(alice), id, domain.IssuePatch{Name: name("Mend")}); err != nil {
		t.Fatal(err)
	}
	want := slices.Concat(locked(id)[:2], []string{line("Enter", web, "outside tx")}, locked(id)[2:], []string{"Now", line("UpdateIssue", id),
		line("InsertActivities", 1), line("GetIssue", id), counted(id), line("Leave", web)})
	if !slices.Equal(w.log.calls, want) {
		t.Errorf("an update's calls:\n%q\nwant\n%q", w.log.calls, want)
	}
}

// Each write P2 adds on a work item enters its project's write gate as an
// update does (TestAWriteEntersItsProjectsGateBetweenItsPrecheckAndItsTransaction):
// after its precheck, outside any transaction, before its transaction
// begins; and leaves it last, once its transaction has ended.
func TestEachWriteOnAWorkItemEntersItsProjectsGate(t *testing.T) {
	for _, tt := range []struct {
		name   string
		action shared.Action
		write  func(w *world, id uuid.UUID) error
	}{
		{"a deletion", domain.ActionDelete, func(w *world, id uuid.UUID) error {
			return deleteOver(w).Execute(as(alice), id)
		}},
		{"an archiving", domain.ActionArchive, func(w *world, id uuid.UUID) error {
			archive, _ := archiveOver(w)
			_, err := archive.Execute(as(alice), id)
			return err
		}},
		{"an unarchiving", domain.ActionArchive, func(w *world, id uuid.UUID) error {
			w.issues.archivedAt = &earlier
			_, unarchive := archiveOver(w)
			_, err := unarchive.Execute(as(alice), id)
			return err
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, _, w, id := newArchive()
			gated(w, nil)
			if err := tt.write(w, id); err != nil {
				t.Fatal(err)
			}
			want := slices.Concat(lockedAs(tt.action, id)[:2], []string{line("Enter", web, "outside tx")}, lockedAs(tt.action, id)[2:])
			if len(w.log.calls) < len(want) || !slices.Equal(w.log.calls[:len(want)], want) ||
				slices.Index(w.log.calls, line("Leave", web)) != len(w.log.calls)-1 {
				t.Errorf("calls:\n%q\nwant them to begin\n%q\nand end with Leave web alone", w.log.calls, want)
			}
		})
	}
}

// A write leaves the gate on every path once it has entered it: refused
// under the lock, ended by a failing port, or by a panic, which goes on.
func TestAWriteLeavesTheGateOnEveryPath(t *testing.T) {
	for _, tt := range []struct {
		name      string
		set       func(w *world)
		recovered any // what goes on past Execute: the write's panic; nil for none
	}{
		{"refused under the lock", func(w *world) { w.project.locked.Archived = true }, nil},
		{"a failing port", func(w *world) { w.log.fail = "UpdateIssue" }, nil},
		{"a panic", func(w *world) { w.project.waited = func() { panic("connection lost") } }, "connection lost"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, w, id := newUpdate()
			_, update := gated(w, nil)
			tt.set(w)
			func() {
				defer func() {
					if r := recover(); r != tt.recovered {
						t.Errorf("recovered %v past Execute(); want %v", r, tt.recovered)
					}
				}()
				if _, err := update.Execute(as(alice), id, domain.IssuePatch{Name: name("Mend")}); err == nil {
					t.Error("Execute() = nil, want its refusal or failure")
				}
			}()
			if last := w.log.calls[len(w.log.calls)-1]; !slices.Contains(w.log.calls, line("Enter", web, "outside tx")) || last != line("Leave", web) {
				t.Errorf("calls:\n%q\nwant the gate entered, and left last", w.log.calls)
			}
		})
	}
}

// A caller the precheck refuses takes no place in the gate; a write the
// gate refuses, its project too busy (503 server_busy), begins no
// transaction and has nothing to leave.
func TestTheGateTakesOnlyWritesThePrecheckLetThrough(t *testing.T) {
	_, w, id := newUpdate()
	_, update := gated(w, nil)
	w.auth.answers = []error{shared.Forbidden()}
	if _, err := update.Execute(as(alice), id, domain.IssuePatch{Name: name("Mend")}); !errors.Is(err, shared.Forbidden()) ||
		!slices.Equal(w.log.calls, locked(id)[:2]) {
		t.Errorf("Execute() = %v after\n%q\nwant forbidden after the precheck alone", err, w.log.calls)
	}

	_, w, id = newUpdate()
	busy := shared.ServerBusy(time.Second)
	_, update = gated(w, busy)
	if _, err := update.Execute(as(alice), id, domain.IssuePatch{Name: name("Mend")}); !errors.Is(err, busy) ||
		!slices.Equal(w.log.calls, append(locked(id)[:2], line("Enter", web, "outside tx"))) {
		t.Errorf("Execute() = %v after\n%q\nwant server_busy once the gate refuses it", err, w.log.calls)
	}
}

// A write across projects (convention 7, M4 design 3.5, 11.11) enters the
// gates of its work item's project and of each project of the work items
// it reaches, all at once in ascending id order, after its precheck and
// outside any transaction; it locks the same projects all at once, in the
// same order (LockMany), and leaves the gates last, once its transaction
// has ended. alice relates web's work item 7 to far, of ops, then near, of
// web: web and ops. Its own project takes its place in that order like
// the others': moved to ops, the higher (web's id is drawn first), the
// work item takes web, then ops, too. A work item the precheck does not
// reach takes neither a lock nor a place in a gate: far, which alice does
// not see, or which is not there, leaves ops out of both, and the refusal
// (422) is answered under web's lock alone.
func TestAWriteAcrossProjectsTakesOnlyTheProjectsItReaches(t *testing.T) {
	for _, tt := range []struct {
		name     string
		set      func(w *world)
		projects []uuid.UUID
	}{
		{"both reached", func(*world) {}, sorted([]uuid.UUID{web, ops})},
		{"both reached from ops", func(w *world) { w.issues.issue.ProjectID = ops }, sorted([]uuid.UUID{web, ops})},
		{"far not seen", func(w *world) { w.auth.answers = []error{1: shared.ErrNotVisible} }, []uuid.UUID{web}},
		{"far not there", func(w *world) { delete(w.issues.others, farID) }, []uuid.UUID{web}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, w, rel, id, _ := newRelate()
			gated(w, nil)
			uc := relateOver(w, rel)
			tt.set(w)
			_, _ = uc.Execute(as(alice), id, domain.RelatesTo, []uuid.UUID{farID, nearID})
			wantAllAtOnce(t, w, tt.projects)
		})
	}
}

// A write on either of two work items (deleteIssueRelation's, M4 design
// 3.10, convention 7) takes the projects of both as a write across
// projects does (TestAWriteAcrossProjectsTakesOnlyTheProjectsItReaches),
// in ascending id order, whichever work item the decision let it through
// on: far's, of ops, or near's, of web, and whichever the relation is
// stored from: far, or near (fromNear), the smaller id of the smaller
// project. Two work items of one project take its gate and its lock once.
func TestAWriteOnEitherWorkItemTakesBothProjects(t *testing.T) {
	for _, tt := range []struct {
		name     string
		set      func(w *world, rel *fakeRelations)
		projects []uuid.UUID
	}{
		{"decided on far", func(*world, *fakeRelations) {}, sorted([]uuid.UUID{web, ops})},
		{"decided on near", func(w *world, _ *fakeRelations) {
			w.auth.answers = []error{0: shared.Forbidden(), 2: shared.Forbidden()}
		}, sorted([]uuid.UUID{web, ops})},
		{"stored from near, decided on near", fromNear, sorted([]uuid.UUID{web, ops})},
		{"both of web", func(w *world, _ *fakeRelations) {
			far := w.issues.others[farID]
			far.ProjectID = web
			w.issues.others[farID] = far
		}, []uuid.UUID{web}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, w, rel := newUnrelate()
			gated(w, nil)
			tt.set(w, rel)
			if err := unrelateOver(w, rel).Execute(as(alice), relationID); err != nil {
				t.Fatal(err)
			}
			wantAllAtOnce(t, w, tt.projects)
		})
	}
}

// wantAllAtOnce checks the calls of a write over w that takes projects all
// at once, in their order: it enters their gates after its precheck,
// outside any transaction, then begins it and locks them (LockMany), and
// leaves the gates last, once it has ended.
func wantAllAtOnce(t *testing.T, w *world, projects []uuid.UUID) {
	t.Helper()
	enter, lock := line("Enter", anys(projects)...)+" outside tx", line("LockMany", anys(projects)...)
	at := slices.Index(w.log.calls, enter)
	if at < 0 || at+2 >= len(w.log.calls) || w.log.calls[at+1] != "Begin" || w.log.calls[at+2] != lock ||
		w.log.calls[len(w.log.calls)-1] != line("Leave", anys(projects)...) || slices.Contains(w.log.calls[:at], "Begin") {
		t.Errorf("calls:\n%q\nwant %q after the precheck, then the transaction and %q, and the gates left last", w.log.calls, enter, lock)
	}
}
