package app_test

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The workspace the inbox tests' slug "acme" names and its projects WEB and OPS; the work items their notifications'
// changes name, the parent one alice reads; the business clock's moment and two before it.
var (
	acme, web, ops          = uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	parentItem, blockedItem = uuid.NewV7(), uuid.NewV7()
	parentRef               = domain.IssueRef{ID: parentItem, SequenceID: 3, Identifier: "WEB", Name: "Ship it"}
	inboxNow                = time.Date(2026, 11, 18, 10, 0, 0, 123456000, time.UTC)
	anHourAgo, twoHoursAgo  = inboxNow.Add(-time.Hour), inboxNow.Add(-2 * time.Hour)
	errInboxPort            = errors.New("the port failed")
)

// The changes of the inbox tests' notifications: one naming the parent, one naming no work item, one naming a work
// item alice does not read.
var (
	parentChange  = domain.ChangeSnapshot{Field: "parent", Verb: domain.VerbUpdated, NewID: &parentItem}
	renamedChange = domain.ChangeSnapshot{Field: "name", Verb: domain.VerbUpdated, OldValue: str("Fix it"), NewValue: str("Fix it now")}
	blockedChange = domain.ChangeSnapshot{Field: "blocked_by", Verb: domain.VerbCreated, NewID: &blockedItem}
)

// readerOf is a request's context acting as account.
func readerOf(account uuid.UUID) context.Context {
	return shared.WithActor(context.Background(), shared.Actor{UserID: account})
}

// note is alice's notification of the work item in WEB, of change, sent at at, unread.
func note(at time.Time, change domain.ChangeSnapshot) app.Notification {
	snapshot := domain.IssueSnapshot{ID: workItem, Name: "Fix it now", SequenceID: 12, Identifier: "WEB", StateName: "Doing", StateGroup: "started"}
	return app.Notification{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, IssueID: workItem, Sender: domain.SenderAssigned, TriggeredByID: &bob,
		Data: domain.Data{Issue: snapshot, Change: change}, CreatedAt: at}
}

// callIn is a call as inboxPorts records it when it is made in the transaction tx.
func callIn(tx, name string, args ...any) string { return callLine(name, args...) + " in " + tx }

// inboxTx is a transaction named name: its beginning and its end, "failed" when fn fails, are recorded among the
// ports' calls, and the calls fn makes know they are made in it.
type inboxTx struct {
	f    *inboxPorts
	name string
}

func (t inboxTx) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	t.f.calls = append(t.f.calls, "begin "+t.name)
	err := fn(context.WithValue(ctx, txKey{}, t.name))
	end := "end " + t.name
	if err != nil {
		end += " failed"
	}
	t.f.calls = append(t.f.calls, end)
	return err
}

// inboxPorts are every port of the inbox's use cases as fakes: the workspace "acme" is acme; the decision answers
// authErr; the projects alice reads are grants; the clock reads inboxNow; the list answers rows, at most its limit of
// them; the counts are counts; the single read and the lock answer stored and locked, nil for not found; the batches
// answer marks in turn, a batch past them marks none, and the failBatch-th, from 1, fails with errInboxPort; the work
// items read are refs; the guest rule is the issue
// module's: a work item is hidden from an own-only account unless its creator is known and is he. Each call is
// recorded in order, with its arguments and the transaction it is made in; the call named fail fails with
// errInboxPort.
type inboxPorts struct {
	authErr   error
	grants    []app.Grant
	rows      []app.Notification
	counts    app.UnreadCounts
	stored    *app.StoredNotification
	locked    *app.StoredNotification
	marks     []app.MarkedBatch
	failBatch int
	refs      map[uuid.UUID]domain.IssueRef
	fail      string
	calls     []string
	listed    []app.ListRead
	counted   []app.UnreadRead
	batches   []app.ReadBatch
	writes    []app.StateWrite
}

// newInboxPorts are the ports where alice reads WEB and, only her own work items, OPS, and the parent is a work item
// she reads.
func newInboxPorts() *inboxPorts {
	return &inboxPorts{grants: []app.Grant{{ProjectID: web}, {ProjectID: ops, OwnOnly: true}}, refs: map[uuid.UUID]domain.IssueRef{parentItem: parentRef}}
}

func (f *inboxPorts) call(ctx context.Context, name string, args ...any) error {
	line := callLine(name, args...)
	if tx, ok := ctx.Value(txKey{}).(string); ok {
		line += " in " + tx
	}
	f.calls = append(f.calls, line)
	if name == f.fail {
		return errInboxPort
	}
	return nil
}

func (f *inboxPorts) readTx() inboxTx  { return inboxTx{f: f, name: "read"} }
func (f *inboxPorts) writeTx() inboxTx { return inboxTx{f: f, name: "write"} }

func (f *inboxPorts) inbox() app.InboxDeps {
	return app.InboxDeps{Workspaces: f, Authorizer: f, Readable: f, Clock: f}
}

func (f *inboxPorts) WorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, bool, error) {
	if err := f.call(ctx, "WorkspaceBySlug", slug); err != nil || slug != "acme" {
		return uuid.UUID{}, false, err
	}
	return acme, true, nil
}

func (f *inboxPorts) Authorize(ctx context.Context, actor shared.Actor, action shared.Action, t shared.Target) (shared.Grant, error) {
	if err := f.call(ctx, "Authorize", actor.UserID, action, t); err != nil {
		return shared.Grant{}, err
	}
	return shared.Grant{}, f.authErr
}

func (f *inboxPorts) Readable(ctx context.Context, actor shared.Actor, workspaceID uuid.UUID) ([]app.Grant, error) {
	if err := f.call(ctx, "Readable", actor.UserID, workspaceID); err != nil {
		return nil, err
	}
	return f.grants, nil
}

func (f *inboxPorts) Now() time.Time {
	f.calls = append(f.calls, "Now")
	return inboxNow
}

func (f *inboxPorts) ListNotifications(ctx context.Context, r app.ListRead) ([]app.Notification, error) {
	f.listed = append(f.listed, r)
	if err := f.call(ctx, "ListNotifications", r.Limit); err != nil {
		return nil, err
	}
	return f.rows[:min(len(f.rows), r.Limit)], nil
}

func (f *inboxPorts) CountUnread(ctx context.Context, r app.UnreadRead) (app.UnreadCounts, error) {
	f.counted = append(f.counted, r)
	if err := f.call(ctx, "CountUnread"); err != nil {
		return app.UnreadCounts{}, err
	}
	return f.counts, nil
}

func (f *inboxPorts) GetNotification(ctx context.Context, id, receiverID uuid.UUID) (app.StoredNotification, bool, error) {
	return f.answer(ctx, "GetNotification", id, receiverID, f.stored)
}

func (f *inboxPorts) LockNotification(ctx context.Context, id, receiverID uuid.UUID) (app.StoredNotification, bool, error) {
	return f.answer(ctx, "LockNotification", id, receiverID, f.locked)
}

func (f *inboxPorts) answer(ctx context.Context, name string, id, receiverID uuid.UUID, n *app.StoredNotification) (app.StoredNotification, bool, error) {
	if err := f.call(ctx, name, id, receiverID); err != nil || n == nil {
		return app.StoredNotification{}, false, err
	}
	return *n, true, nil
}

// UpdateNotification answers the locked row with the states written.
func (f *inboxPorts) UpdateNotification(ctx context.Context, w app.StateWrite) (app.Notification, error) {
	f.writes = append(f.writes, w)
	if err := f.call(ctx, "UpdateNotification", w.ID, w.ReceiverID); err != nil {
		return app.Notification{}, err
	}
	n := f.locked.Notification
	n.State = w.State
	return n, nil
}

func (f *inboxPorts) MarkReadBatch(ctx context.Context, b app.ReadBatch) (app.MarkedBatch, error) {
	f.batches = append(f.batches, b)
	n := len(f.batches)
	if err := f.call(ctx, "MarkReadBatch", b.After, b.Limit); err != nil {
		return app.MarkedBatch{}, err
	}
	switch {
	case n == f.failBatch:
		return app.MarkedBatch{}, errInboxPort
	case n <= len(f.marks):
		return f.marks[n-1], nil
	}
	return app.MarkedBatch{Last: b.After}, nil
}

func (f *inboxPorts) IssueRefs(ctx context.Context, reader uuid.UUID, grants []app.Grant, ids []uuid.UUID) (map[uuid.UUID]domain.IssueRef, error) {
	if err := f.call(ctx, "IssueRefs", reader, grants, ids); err != nil {
		return nil, err
	}
	refs := map[uuid.UUID]domain.IssueRef{}
	for _, id := range ids {
		if ref, ok := f.refs[id]; ok {
			refs[id] = ref
		}
	}
	return refs, nil
}

// HiddenFrom is the issue module's guest rule (domain.Issue.HiddenFrom): hidden from an own-only account unless the
// creator is known and is he. The question, its creator ("-" for none), is recorded among the calls.
func (f *inboxPorts) HiddenFrom(creator *uuid.UUID, account uuid.UUID, ownOnly bool) bool {
	who := "-"
	if creator != nil {
		who = creator.String()
	}
	f.calls = append(f.calls, callLine("HiddenFrom", who, account, ownOnly))
	return ownOnly && (creator == nil || *creator != account)
}
