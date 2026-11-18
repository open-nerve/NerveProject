package app_test

import (
	"errors"
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// markBatch is mark-all-read's batch (spec 2.7): a constant, not configuration.
const markBatch = 1000

func markAllRead(f *inboxPorts) *app.MarkAllRead {
	return app.NewMarkAllRead(app.MarkAllReadDeps{InboxDeps: f.inbox(), ReadTx: f.readTx(), Tx: f.writeTx(), Marks: f})
}

// batchCalls are the calls of a batch from after: its own transaction, the batch alone in it.
func batchCalls(after uuid.UUID, end string) []string {
	return []string{"begin write", callIn("write", "MarkReadBatch", after, markBatch), end}
}

// Mark-all-read opens the inbox once, its clock and the projects she reads read once at the start, before any
// batch (R5); then marks batch after batch, each in its own transaction from the last id the batch before marked,
// until one marks fewer than the batch; it answers the sum. Each batch is the same inbox, at the same moment, and the
// body's filter.
func TestMarkAllReadMarksBatchAfterBatchTillAShortOne(t *testing.T) {
	first, second, third := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for _, c := range []struct {
		name   string
		marks  []app.MarkedBatch
		afters []uuid.UUID
		sum    int
	}{
		{"none", nil, []uuid.UUID{{}}, 0},
		{"fewer than a batch", []app.MarkedBatch{{Marked: 7, Last: first}}, []uuid.UUID{{}}, 7},
		{"two batches and a short one", []app.MarkedBatch{{Marked: markBatch, Last: first}, {Marked: markBatch, Last: second}, {Marked: 7, Last: third}},
			[]uuid.UUID{{}, first, second}, 2*markBatch + 7},
		{"a batch and an empty one", []app.MarkedBatch{{Marked: markBatch, Last: first}}, []uuid.UUID{{}, first}, markBatch},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newInboxPorts()
			f.marks = c.marks
			mentions := domain.TabMentions
			marked, err := markAllRead(f).Execute(readerOf(alice), "acme", domain.FilterValues{Tab: &mentions, Snoozed: new(true)})
			if err != nil || marked != c.sum {
				t.Fatalf("Execute = %d, %v; want %d", marked, err, c.sum)
			}
			want := append(inboxOpened(domain.ActionNotificationWrite), "end read")
			var batches []app.ReadBatch
			for _, after := range c.afters {
				want = append(want, batchCalls(after, "end write")...)
				batches = append(batches, app.ReadBatch{Inbox: alicesInbox(f), Filter: domain.Filter{Senders: domain.TabSenders(domain.TabMentions), Snoozed: true},
					After: after, Limit: markBatch})
			}
			sameCalls(t, f.calls, want)
			if !reflect.DeepEqual(f.batches, batches) {
				t.Fatalf("batches %+v, want %+v", f.batches, batches)
			}
		})
	}
}

// A batch that fails answers its error; the batches before it are committed, and none follows.
func TestMarkAllReadAnswersTheFailedBatchsError(t *testing.T) {
	f := newInboxPorts()
	first := uuid.NewV7()
	f.marks = []app.MarkedBatch{{Marked: markBatch, Last: first}}
	f.failBatch = 2
	if _, err := markAllRead(f).Execute(readerOf(alice), "acme", domain.FilterValues{}); !errors.Is(err, errInboxPort) {
		t.Fatalf("Execute = %v, want the batch's error", err)
	}
	want := append(append(inboxOpened(domain.ActionNotificationWrite), "end read"), batchCalls(uuid.UUID{}, "end write")...)
	sameCalls(t, f.calls, append(want, batchCalls(first, "end write failed")...))
}

// The filter is checked before anything is read: a 422 of the body's own values, whatever the workspace.
func TestMarkAllReadRefusesItsFilterBeforeReading(t *testing.T) {
	f := newInboxPorts()
	mentions := domain.TabMentions
	_, err := markAllRead(f).Execute(readerOf(alice), "nowhere", domain.FilterValues{Tab: &mentions, Type: &[]domain.Sender{domain.SenderCreated}})
	var problem *shared.Error
	if !errors.As(err, &problem) || problem.Code != shared.CodeValidationFailed || len(f.calls) != 0 {
		t.Fatalf("Execute = %v, calls %v; want a 422 and no call", err, f.calls)
	}
}

// A workspace that is not there or not seen is workspace.not_found, the same value, and nothing is marked; another
// refusal, and a read's failure, is answered as it is.
func TestMarkAllReadInAWorkspaceNotSeenIsNotFound(t *testing.T) {
	f := newInboxPorts()
	if _, err := markAllRead(f).Execute(readerOf(alice), "nowhere", domain.FilterValues{}); err != domain.ErrWorkspaceNotFound || len(f.batches) != 0 {
		t.Fatalf("an unknown slug: %v, batches %v; want workspace.not_found and none", err, f.batches)
	}
	f = newInboxPorts()
	f.authErr = shared.ErrNotVisible
	if _, err := markAllRead(f).Execute(readerOf(alice), "acme", domain.FilterValues{}); err != domain.ErrWorkspaceNotFound || len(f.batches) != 0 {
		t.Fatalf("a workspace not seen: %v, batches %v; want workspace.not_found and none", err, f.batches)
	}
	f = newInboxPorts()
	f.authErr = shared.Forbidden()
	if _, err := markAllRead(f).Execute(readerOf(alice), "acme", domain.FilterValues{}); !errors.Is(err, shared.Forbidden()) || len(f.batches) != 0 {
		t.Fatalf("a refused role: %v, batches %v; want forbidden and none", err, f.batches)
	}
	f = newInboxPorts()
	f.fail = "Readable"
	if _, err := markAllRead(f).Execute(readerOf(alice), "acme", domain.FilterValues{}); !errors.Is(err, errInboxPort) || len(f.batches) != 0 {
		t.Fatalf("Readable failing: %v, batches %v; want its error and none", err, f.batches)
	}
}
