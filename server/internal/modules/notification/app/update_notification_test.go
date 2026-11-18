package app_test

import (
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func updateNotification(f *inboxPorts) *app.UpdateNotification {
	return app.NewUpdateNotification(app.UpdateNotificationDeps{Authorizer: f, Readable: f, GuestRule: f, Clock: f, Rows: f, Refs: f, Tx: f.writeTx()})
}

// stored is n as its row stores it, of a work item creator created, nil for not known; the single read and the lock
// both answer it.
func stored(f *inboxPorts, n app.Notification, creator *uuid.UUID) {
	f.stored = &app.StoredNotification{Notification: n, IssueCreatedByID: creator}
	locked := *f.stored
	f.locked = &locked
}

// reachedCalls are the calls of steps 1 to 3 (spec 2.7) for alice's notification id, of a work item creator created
// ("-" for not known), in a project she reads, own-only or not: without a lock and outside any transaction.
func reachedCalls(id uuid.UUID, creator string, ownOnly bool) []string {
	return []string{callLine("GetNotification", id, alice), callLine("Authorize", alice, domain.ActionNotificationWrite, shared.Target{WorkspaceID: acme}),
		callLine("Readable", alice, acme), callLine("HiddenFrom", creator, alice, ownOnly)}
}

// A change follows spec 2.7's seven steps: the row read without a lock, the decision on its workspace, the projects
// she reads and the guest rule, all before the transaction; in it the row locked, the clock read after the lock,
// the write of what changed at that moment by her, and the work items the row names asked for with the projects of
// step 3; she is answered the row as written, its parent she reads named.
func TestUpdateNotificationDecidesBeforeTheLockAndReadsTheClockAfterIt(t *testing.T) {
	f := newInboxPorts()
	n := note(anHourAgo, parentChange)
	stored(f, n, &bob)
	v, err := updateNotification(f).Execute(readerOf(alice), n.ID, domain.Patch{Read: new(true)})
	if err != nil {
		t.Fatal(err)
	}
	sameCalls(t, f.calls, append(reachedCalls(n.ID, bob.String(), false), "begin write", callIn("write", "LockNotification", n.ID, alice), "Now",
		callIn("write", "UpdateNotification", n.ID, alice), callIn("write", "IssueRefs", alice, f.grants, []uuid.UUID{parentItem}), "end write"))
	read := domain.State{ReadAt: &inboxNow}
	if want := []app.StateWrite{{ID: n.ID, ReceiverID: alice, State: read, At: inboxNow}}; !reflect.DeepEqual(f.writes, want) {
		t.Fatalf("wrote %+v, want %+v", f.writes, want)
	}
	n.State = read
	if want := viewOf(n, map[uuid.UUID]domain.IssueRef{parentItem: parentRef}); !reflect.DeepEqual(v, want) {
		t.Fatalf("answered %+v, want %+v", v, want)
	}
}

// A change to the state the locked row is already in writes nothing (spec 2.7, step 7) and answers the row as the
// lock read it, here archived since step 1; a row that names no work item asks for none.
func TestUpdateNotificationThatChangesNothingAnswersTheLockedRow(t *testing.T) {
	for name, p := range map[string]domain.Patch{"{}": {}, "archived again": {Archived: new(true)}, "unread again": {Read: new(false)}} {
		t.Run(name, func(t *testing.T) {
			f := newInboxPorts()
			n := note(anHourAgo, renamedChange)
			stored(f, n, &bob)
			f.locked.State = domain.State{ArchivedAt: &twoHoursAgo}
			v, err := updateNotification(f).Execute(readerOf(alice), n.ID, p)
			if err != nil {
				t.Fatal(err)
			}
			sameCalls(t, f.calls, append(reachedCalls(n.ID, bob.String(), false), "begin write", callIn("write", "LockNotification", n.ID, alice), "Now", "end write"))
			if want := viewOf(f.locked.Notification, nil); !reflect.DeepEqual(v, want) {
				t.Fatalf("answered %+v, want the locked row %+v", v, want)
			}
		})
	}
}

// Every notification alice cannot reach is the one notification.not_found, the same value (spec 2.2, 2.7, R5):
// another receiver's, a deleted one and none at all, which the store does not tell apart; one in a workspace she is
// not a member of; one of a project she does not read; where she sees only her own work items, one of another's work
// item and one whose creator is not known; and one deleted between the read and the lock. Each asks a past
// snoozed_till, a 422 if it were checked: none is, the clock is never read and nothing is written (404 before 422).
func TestEveryNotificationSheCannotReachIsTheSameNotFound(t *testing.T) {
	past := inboxNow.Add(-time.Minute)
	patch := domain.Patch{Read: new(true), SetSnoozed: true, SnoozedTill: &past}
	n := note(anHourAgo, parentChange)
	get, auth := callLine("GetNotification", n.ID, alice), callLine("Authorize", alice, domain.ActionNotificationWrite, shared.Target{WorkspaceID: acme})
	readable := callLine("Readable", alice, acme)
	for _, c := range []struct {
		name    string
		creator *uuid.UUID
		arrange func(f *inboxPorts)
		calls   []string
	}{
		{"another's, a deleted one or none", &bob, func(f *inboxPorts) { f.stored = nil }, []string{get}},
		{"in a workspace she is not a member of", &bob, func(f *inboxPorts) { f.authErr = shared.ErrNotVisible }, []string{get, auth}},
		{"of a project she does not read", &bob, func(f *inboxPorts) { f.grants = []app.Grant{{ProjectID: ops}} }, []string{get, auth, readable}},
		{"of another's work item where she sees her own", &bob, func(f *inboxPorts) { f.grants = []app.Grant{{ProjectID: web, OwnOnly: true}} },
			reachedCalls(n.ID, bob.String(), true)},
		{"of a work item of no known creator where she sees her own", nil, func(f *inboxPorts) { f.grants = []app.Grant{{ProjectID: web, OwnOnly: true}} },
			reachedCalls(n.ID, "-", true)},
		{"deleted between the read and the lock", &bob, func(f *inboxPorts) { f.locked = nil },
			append(reachedCalls(n.ID, bob.String(), false), "begin write", callIn("write", "LockNotification", n.ID, alice), "end write failed")},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newInboxPorts()
			stored(f, n, c.creator)
			c.arrange(f)
			_, err := updateNotification(f).Execute(readerOf(alice), n.ID, patch)
			if err != domain.ErrNotificationNotFound {
				t.Fatalf("Execute = %v, want notification.not_found itself", err)
			}
			sameCalls(t, f.calls, c.calls)
			if len(f.writes) != 0 {
				t.Fatalf("wrote %+v, want nothing", f.writes)
			}
		})
	}
}

// Where alice sees only her own work items, the notification of one she created is hers to change.
func TestUpdateNotificationOfHerOwnWorkItemWhereSheSeesHerOwn(t *testing.T) {
	f := newInboxPorts()
	n := note(anHourAgo, renamedChange)
	stored(f, n, &alice)
	f.grants = []app.Grant{{ProjectID: web, OwnOnly: true}}
	if _, err := updateNotification(f).Execute(readerOf(alice), n.ID, domain.Patch{Archived: new(true)}); err != nil {
		t.Fatal(err)
	}
	sameCalls(t, f.calls, append(reachedCalls(n.ID, alice.String(), true), "begin write", callIn("write", "LockNotification", n.ID, alice), "Now",
		callIn("write", "UpdateNotification", n.ID, alice), "end write"))
}

// A snoozed_till not after the moment read under the lock is the 422, checked only there (step 6), and nothing is
// written.
func TestUpdateNotificationRefusesAPastSnoozeOnlyUnderTheLock(t *testing.T) {
	f := newInboxPorts()
	n := note(anHourAgo, parentChange)
	stored(f, n, &bob)
	_, err := updateNotification(f).Execute(readerOf(alice), n.ID, domain.Patch{SetSnoozed: true, SnoozedTill: &inboxNow})
	var problem *shared.Error
	if !errors.As(err, &problem) || len(problem.Fields) != 1 || problem.Fields[0].Field != "snoozed_till" || problem.Fields[0].Code != shared.FieldMustBeFuture {
		t.Fatalf("Execute = %v, want 422 must_be_future on snoozed_till", err)
	}
	sameCalls(t, f.calls, append(reachedCalls(n.ID, bob.String(), false), "begin write", callIn("write", "LockNotification", n.ID, alice), "Now",
		"end write failed"))
}

// A refusal other than not seeing the workspace, and every port's failure, is answered as it is, not as the 404.
func TestUpdateNotificationAnswersOtherFailuresAsTheyAre(t *testing.T) {
	n := note(anHourAgo, parentChange)
	f := newInboxPorts()
	stored(f, n, &bob)
	f.authErr = shared.Forbidden()
	if _, err := updateNotification(f).Execute(readerOf(alice), n.ID, domain.Patch{Read: new(true)}); !errors.Is(err, shared.Forbidden()) {
		t.Fatalf("a refused role: %v, want forbidden", err)
	}
	for _, port := range []string{"GetNotification", "Authorize", "Readable", "LockNotification", "UpdateNotification", "IssueRefs"} {
		f := newInboxPorts()
		stored(f, n, &bob)
		f.fail = port
		if _, err := updateNotification(f).Execute(readerOf(alice), n.ID, domain.Patch{Read: new(true)}); !errors.Is(err, errInboxPort) {
			t.Fatalf("%s failing: %v, want its error", port, err)
		}
	}
}
