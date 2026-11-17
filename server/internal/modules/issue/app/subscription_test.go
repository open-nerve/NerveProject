package app_test

import (
	"context"
	"maps"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// subscriptions is subscribeIssue and unsubscribeIssue over web's world
// (newCreate's), its write gate one that logs, with web's work item 7
// stored in it, made by carol at earlier; subs the subscribers' store.
type subscriptions struct {
	subscribe   *app.SubscribeIssue
	unsubscribe *app.UnsubscribeIssue
	w           *world
	subs        *fakeSubscriptions
	id          uuid.UUID
}

func newSubscriptions() subscriptions {
	_, w := newCreate()
	w.locks = app.NewLocks(w.project, w.issues, w.auth, fakeGate{log: w.log}, fakeTx{log: w.log}, noIssueAttachments{})
	id := uuid.NewV7()
	w.issues.issue = &app.IssueRow{ID: id, WorkspaceID: acme, ProjectID: web, SequenceID: 7, Name: "Fix", StateID: backlog.ID,
		Priority: domain.PriorityHigh, SortOrder: 65535, By: carol, At: earlier}
	subs := &fakeSubscriptions{log: w.log, subscribed: map[uuid.UUID]bool{}}
	d := app.SubscriptionDeps{Locks: w.locks, Subscriptions: subs, Clock: clockAt{at: clockNow, log: w.log}}
	return subscriptions{subscribe: app.NewSubscribeIssue(d), unsubscribe: app.NewUnsubscribeIssue(d), w: w, subs: subs, id: id}
}

// subscriptionWrite is one of the two writes: its name, how it runs on s's
// work item, and the call of its one row's write by alice, as the fakes
// log it.
type subscriptionWrite struct {
	name  string
	run   func(s subscriptions, ctx context.Context) error
	write string
}

// subscriptionWrites are the subscription and the unsubscription.
func subscriptionWrites() []subscriptionWrite {
	return []subscriptionWrite{
		{"subscribe", func(s subscriptions, ctx context.Context) error { return s.subscribe.Execute(ctx, s.id) },
			line("InsertSubscribers", []uuid.UUID{alice})},
		{"unsubscribe", func(s subscriptions, ctx context.Context) error { return s.unsubscribe.Execute(ctx, s.id) },
			line("RemoveSubscriber", alice)},
	}
}

// sharedCalls are the calls of a subscription's write of the work item id by
// alice up to the decision under the lock: the work item read and the
// decision on issue.read, before any lock and outside the transaction; in
// it, the work item's row FOR SHARE and the decision again. No write gate,
// no project lock.
func sharedCalls(id uuid.UUID) []string {
	return []string{line("GetIssue", id, "outside tx"), line("Authorize", alice, domain.ActionRead, acme, web, "outside tx"), "Begin",
		line("ShareIssue", id), line("Authorize", alice, domain.ActionRead, acme, web)}
}

// A subscription and an unsubscription, in the order of M7 P1a spec 2.4:
// the precheck, before any lock and outside the transaction; then, in it,
// the work item's row FOR SHARE and the decision again on it; the clock;
// and the caller's one row, written: a subscription's row of alice under
// web's work item, at the moment and by her; an unsubscription's removal
// of her row, at the moment and by her. Neither enters the write gate, nor
// locks the project, marks the work item changed, writes its activity or
// publishes an event (the use cases hold no port to).
func TestASubscriptionWritesTheCallersRowUnderTheWorkItemsShare(t *testing.T) {
	for _, tt := range subscriptionWrites() {
		t.Run(tt.name, func(t *testing.T) {
			s := newSubscriptions()
			if err := tt.run(s, as(alice)); err != nil {
				t.Fatal(err)
			}
			if want := append(sharedCalls(s.id), "Now", tt.write); !slices.Equal(s.w.log.calls, want) {
				t.Errorf("calls:\n%q\nwant\n%q", s.w.log.calls, want)
			}
			switch tt.name {
			case "subscribe":
				want := app.LinkRows{WorkspaceID: acme, ProjectID: web, IssueID: s.id, Targets: []uuid.UUID{alice}, By: alice, At: clockNow}
				if len(s.subs.inserted) != 1 || len(s.subs.inserted[0].IDs) != 1 || s.subs.inserted[0].IDs[0] == (uuid.UUID{}) {
					t.Fatalf("inserted %+v; want one row with an id", s.subs.inserted)
				}
				want.IDs = s.subs.inserted[0].IDs
				if !reflect.DeepEqual(s.subs.inserted[0], want) {
					t.Errorf("inserted %+v; want %+v", s.subs.inserted[0], want)
				}
			case "unsubscribe":
				want := []app.SubscriberRemoval{{IssueID: s.id, SubscriberID: alice, By: alice, At: clockNow}}
				if !reflect.DeepEqual(s.subs.removed, want) {
					t.Errorf("removed %+v; want %+v", s.subs.removed, want)
				}
			}
		})
	}
}

// Subscribing twice and unsubscribing twice succeed (M7 P1a spec 2.3): each
// writes its row again, which the store's statements make no-ops; neither
// reads first nor refuses. alice subscribes once, then not at all.
func TestSubscribingAndUnsubscribingTwiceSucceed(t *testing.T) {
	s := newSubscriptions()
	for _, step := range []struct {
		run  func(context.Context, uuid.UUID) error
		want map[uuid.UUID]bool
	}{
		{s.subscribe.Execute, map[uuid.UUID]bool{alice: true}},
		{s.subscribe.Execute, map[uuid.UUID]bool{alice: true}},
		{s.unsubscribe.Execute, map[uuid.UUID]bool{}},
		{s.unsubscribe.Execute, map[uuid.UUID]bool{}},
	} {
		if err := step.run(as(alice), s.id); err != nil || !maps.Equal(s.subs.subscribed, step.want) {
			t.Fatalf("Execute() = %v, subscribed %v; want nil, %v", err, s.subs.subscribed, step.want)
		}
	}
	if len(s.subs.inserted) != 2 || len(s.subs.removed) != 2 {
		t.Errorf("inserted %d rows, removed %d; want each write written", len(s.subs.inserted), len(s.subs.removed))
	}
}

// A subscription the precheck refuses (M7 P1a spec 2.3, 2.4) is answered
// there, before any lock: it begins no transaction and writes nothing. A
// work item not there, whose project alice does not see, or that the guest
// rule hides from her (carol made it), is issue.not_found; a role that may
// not read it, forbidden; a request without a caller, unauthenticated
// before any read; a read or a decision that fails, its failure.
func TestASubscriptionRefusedByThePrecheckTakesNoLock(t *testing.T) {
	for _, write := range subscriptionWrites() {
		for _, tt := range []struct {
			name  string
			set   func(s *subscriptions)
			ctx   context.Context
			want  error
			calls int // of the precheck's two
		}{
			{"no work item", func(s *subscriptions) { s.id = uuid.NewV7() }, as(alice), domain.ErrIssueNotFound, 1},
			{"a project not seen", func(s *subscriptions) { s.w.auth.answers = []error{shared.ErrNotVisible} }, as(alice), domain.ErrIssueNotFound, 2},
			{"hidden by the guest rule", func(s *subscriptions) { s.w.auth.grants = []shared.Grant{{OwnOnly: true}} }, as(alice), domain.ErrIssueNotFound, 2},
			{"a role not allowed", func(s *subscriptions) { s.w.auth.answers = []error{shared.Forbidden()} }, as(alice), shared.Forbidden(), 2},
			{"the read fails", func(s *subscriptions) { s.w.log.fail = "GetIssue" }, as(alice), errDisk, 1},
			{"the decision fails", func(s *subscriptions) { s.w.log.fail = "Authorize" }, as(alice), errDisk, 2},
			{"no caller", func(*subscriptions) {}, context.Background(), shared.Unauthenticated(), 0},
		} {
			t.Run(write.name+": "+tt.name, func(t *testing.T) {
				s := newSubscriptions()
				tt.set(&s)
				if err := write.run(s, tt.ctx); !reflect.DeepEqual(err, tt.want) {
					t.Errorf("Execute() = %v; want %v", err, tt.want)
				}
				if want := sharedCalls(s.id)[:tt.calls]; !slices.Equal(s.w.log.calls, want) {
					t.Errorf("calls:\n%q\nwant\n%q", s.w.log.calls, want)
				}
			})
		}
	}
}

// What the precheck let through is decided again on the work item's row
// as locked (M7 P1a spec 2.4), which is final: a work item deleted while
// the lock waited is issue.not_found; so is one whose project alice no
// longer sees, or that the guest rule now hides from her, the guest who
// sees only her own she was made meanwhile; a role no longer allowed is
// forbidden; a lock that fails, its failure. Each writes nothing.
func TestASubscriptionDecidesAgainOnTheLockedRow(t *testing.T) {
	for _, write := range subscriptionWrites() {
		for _, tt := range []struct {
			name  string
			set   func(s *subscriptions)
			want  error
			calls int // of sharedCalls'
		}{
			{"deleted while the lock waited", func(s *subscriptions) { s.w.issues.sharing = func() { s.w.issues.issue = nil } }, domain.ErrIssueNotFound, 4},
			{"the lock fails", func(s *subscriptions) { s.w.log.fail = "ShareIssue" }, errDisk, 4},
			{"removed from the project meanwhile", func(s *subscriptions) { s.w.auth.answers = []error{1: shared.ErrNotVisible} }, domain.ErrIssueNotFound, 5},
			{"made a guest who sees only her own meanwhile", func(s *subscriptions) { s.w.auth.grants = []shared.Grant{1: {OwnOnly: true}} },
				domain.ErrIssueNotFound, 5},
			{"a role no longer allowed", func(s *subscriptions) { s.w.auth.answers = []error{1: shared.Forbidden()} }, shared.Forbidden(), 5},
		} {
			t.Run(write.name+": "+tt.name, func(t *testing.T) {
				s := newSubscriptions()
				tt.set(&s)
				if err := write.run(s, as(alice)); !reflect.DeepEqual(err, tt.want) {
					t.Errorf("Execute() = %v; want %v", err, tt.want)
				}
				if want := sharedCalls(s.id)[:tt.calls]; !slices.Equal(s.w.log.calls, want) {
					t.Errorf("calls:\n%q\nwant\n%q", s.w.log.calls, want)
				}
			})
		}
	}
}
