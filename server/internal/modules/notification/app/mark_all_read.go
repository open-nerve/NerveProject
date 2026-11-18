package app

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// markBatch is how many notifications a batch of mark-all-read marks at
// most, each batch in its transaction (spec 2.7): a constant, not
// configuration.
const markBatch = 1000

// MarkAllReadDeps supply mark-all-read: its inbox, opened in a read
// snapshot (REPEATABLE READ, READ ONLY); the transactions of its batches,
// read-write, READ COMMITTED; and the store's batch.
type MarkAllReadDeps struct {
	InboxDeps
	ReadTx shared.TxManager
	Tx     shared.TxManager
	Marks  ReadMarks
}

// MarkAllRead marks the caller's unread notifications in a workspace that
// a filter selects read (spec 2.2, 2.7): POST
// /workspaces/{slug}/notifications/mark-all-read.
type MarkAllRead struct{ d MarkAllReadDeps }

// NewMarkAllRead constructs mark-all-read.
func NewMarkAllRead(d MarkAllReadDeps) *MarkAllRead {
	return &MarkAllRead{d: d}
}

// Execute checks the filter of v first, whatever the workspace, a 422.
// Then it opens, in one read snapshot, the inbox of the workspace slug
// names for notification.write: the projects the caller reads and the
// clock's moment are read once, at the start, and every batch shares them
// (R5). The moment is read before any lock, not after the last one as a
// write's is (hard rule 03 §4): it is also the bound of what is marked,
// the events at or before it, the same for every batch; a moment read
// after each batch's lock would move the bound between batches. Then it
// marks batch after batch, each in its own transaction, from the greatest
// id the one before marked, until one marks fewer than markBatch, and
// answers how many the batches marked in all. A batch never waits for a
// row lock: it passes over one another transaction holds. One that fails
// answers its error; the batches before it stay committed, and a request
// again marks the rest.
func (u *MarkAllRead) Execute(ctx context.Context, slug string, v domain.FilterValues) (int, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return 0, err
	}
	filter, err := domain.NewFilter(v)
	if err != nil {
		return 0, err
	}
	var inbox Inbox
	err = u.d.ReadTx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		inbox, err = u.d.open(ctx, actor, slug, domain.ActionNotificationWrite)
		return err
	})
	if err != nil {
		return 0, err
	}
	b := ReadBatch{Inbox: inbox, Filter: filter, Limit: markBatch}
	marked := 0
	for {
		var m MarkedBatch
		err := u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
			var err error
			m, err = u.d.Marks.MarkReadBatch(ctx, b)
			return err
		})
		if err != nil {
			return 0, err
		}
		marked += m.Marked
		if m.Marked < markBatch {
			return marked, nil
		}
		b.After = m.Last
	}
}
