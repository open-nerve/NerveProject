package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var errDisk = errors.New("disk full")

// inTxKey marks the context of the caller's transaction: a call that does
// not carry it ran outside it.
type inTxKey struct{}

// callLog records the calls of the fakes that share or embed it, in order,
// each with its arguments, and " outside tx" when it ran outside a
// transaction. It fails the call named in fail, or the one fail writes
// with its arguments as line does, with errDisk, after logging it. inTx is
// set while a fakeTx's transaction runs.
type callLog struct {
	calls []string
	fail  string
	inTx  bool
}

func (l *callLog) add(ctx context.Context, name string, args ...any) error {
	call := line(name, args...)
	failed := name == l.fail || call == l.fail
	if ctx.Value(inTxKey{}) != true {
		call += " outside tx"
	}
	l.calls = append(l.calls, call)
	if failed {
		return errDisk
	}
	return nil
}

// fakeTx runs fn in a context marked as inside the transaction.
type fakeTx struct{ log *callLog }

func (f fakeTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	f.log.calls = append(f.log.calls, "Begin")
	f.log.inTx = true
	defer func() { f.log.inTx = false }()
	return fn(context.WithValue(ctx, inTxKey{}, true))
}

// fakeGate is the write gate. Without a log, it lets every write in. With
// one, it logs each entry, " outside tx" when it is outside a transaction,
// then refuses the write with refuse, when set, or lets it in and logs its
// leaving, " in tx" when the transaction has not ended.
type fakeGate struct {
	log    *callLog
	refuse error
}

func (f fakeGate) Enter(ctx context.Context, projectIDs ...uuid.UUID) (func(), error) {
	if f.log == nil {
		return func() {}, nil
	}
	if err := f.log.add(ctx, "Enter", anys(projectIDs)...); err != nil {
		return nil, err
	}
	if f.refuse != nil {
		return nil, f.refuse
	}
	return func() {
		call := line("Leave", anys(projectIDs)...)
		if f.log.inTx {
			call += " in tx"
		}
		f.log.calls = append(f.log.calls, call)
	}, nil
}

// clockAt stands at an instant; each read is logged as "Now".
type clockAt struct {
	at  time.Time
	log *callLog
}

func (c clockAt) Now() time.Time {
	c.log.calls = append(c.log.calls, "Now")
	return c.at
}

// fakeAuthorizer answers its n-th call answers[n], nil or past the end
// for allowed, with grants[n], the Grant of what it read, which a refusal
// of the role carries too (shared.Authorizer), empty past the end; it logs
// every call, and keeps each call's target.
type fakeAuthorizer struct {
	log     *callLog
	answers []error
	grants  []shared.Grant
	targets []shared.Target
	calls   int
}

func (f *fakeAuthorizer) Authorize(ctx context.Context, actor shared.Actor, action shared.Action, t shared.Target) (shared.Grant, error) {
	n := f.calls
	f.calls++
	f.targets = append(f.targets, t)
	if err := f.log.add(ctx, "Authorize", actor.UserID, action, t.WorkspaceID, t.ProjectID); err != nil {
		return shared.Grant{}, err
	}
	var grant shared.Grant
	if n < len(f.grants) {
		grant = f.grants[n]
	}
	if n < len(f.answers) && f.answers[n] != nil {
		return grant, f.answers[n]
	}
	return grant, nil
}

// fakeProject is a project's side of the issue module's ports: its gate,
// its numbers and its catalogs, each call logged. found is whether Find
// answers it, locked what Lock answers, nil for none, and across the
// other projects LockMany answers; waited, when set, runs as a lock waits,
// before it answers: what commits meanwhile.
type fakeProject struct {
	log               *callLog
	id                uuid.UUID
	workspace         uuid.UUID
	found             bool
	locked            *app.LockedProject
	across            map[uuid.UUID]app.LockedProject
	waited            func()
	last              int
	states            []app.StateRef
	labels            []app.LabelRef
	roles             map[uuid.UUID]shared.Role
	mentionVisibility []app.MemberVisibility
}

func (f *fakeProject) Find(ctx context.Context, id uuid.UUID) (uuid.UUID, bool, error) {
	if err := f.log.add(ctx, "Find", id); err != nil || id != f.id || !f.found {
		return uuid.UUID{}, false, err
	}
	return f.workspace, true, nil
}

func (f *fakeProject) Lock(ctx context.Context, id uuid.UUID) (app.LockedProject, bool, error) {
	if f.waited != nil {
		f.waited()
	}
	if err := f.log.add(ctx, "Lock", id); err != nil || id != f.id || f.locked == nil {
		return app.LockedProject{}, false, err
	}
	return *f.locked, true, nil
}

func (f *fakeProject) Next(ctx context.Context, id uuid.UUID) (int, error) {
	if err := f.log.add(ctx, "Next", id); err != nil {
		return 0, err
	}
	f.last++
	return f.last, nil
}

func (f *fakeProject) States(ctx context.Context, id uuid.UUID) ([]app.StateRef, error) {
	if err := f.log.add(ctx, "States", id); err != nil {
		return nil, err
	}
	return f.states, nil
}

func (f *fakeProject) Labels(ctx context.Context, id uuid.UUID, ids []uuid.UUID) ([]app.LabelRef, error) {
	if err := f.log.add(ctx, "Labels", id, ids); err != nil {
		return nil, err
	}
	return slices.DeleteFunc(slices.Clone(f.labels), func(l app.LabelRef) bool { return !slices.Contains(ids, l.ID) }), nil
}

func (f *fakeProject) Roles(ctx context.Context, id uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]shared.Role, error) {
	if err := f.log.add(ctx, "Roles", id, ids); err != nil {
		return nil, err
	}
	out := map[uuid.UUID]shared.Role{}
	for _, a := range ids {
		if r, ok := f.roles[a]; ok {
			out[a] = r
		}
	}
	return out, nil
}

// fakeProfiles answers the display names it has, each call logged.
type fakeProfiles struct {
	log   *callLog
	names map[uuid.UUID]string
}

func (f fakeProfiles) DisplayNames(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	if err := f.log.add(ctx, "DisplayNames", ids); err != nil {
		return nil, err
	}
	out := map[uuid.UUID]string{}
	for _, id := range ids {
		if n, ok := f.names[id]; ok {
			out[id] = n
		}
	}
	return out, nil
}

// fakeIssues is the work items' store: it keeps what it is given, each
// call logged, and answers the work item from it, its assignees and labels
// in id order; the subscribers it checks and logs, and keeps none. greatest
// is the greatest sort order of each state; lost, when set, loses the work
// items an activity names once it is written, not there to read back (lose).
// changed is the last update's fields, nil before any: the work item's
// last writer and moment; archival, when set, is the last archiving's or
// unarchiving's, and its last writer and moment instead; archivedAt the
// work item's archived_at, nil for none. subtree is the work item's
// subtree, nil for the work item alone, and deletions are the deletion's
// calls' arguments, one a table, in order (fakes_deletion_test.go). others
// are the store's other work items, answered as they are, ancestry their
// ancestries, counts the work items' numbers of sub-items, subItems the
// sub-items it answers, and parented the last addition of sub-items' rows
// (fakes_parents_test.go).
type fakeIssues struct {
	log        *callLog
	greatest   map[uuid.UUID]float64
	issue      *app.IssueRow
	changed    *app.IssueFields
	archival   *app.IssueArchival
	archivedAt *time.Time
	assignees  []uuid.UUID
	labels     []uuid.UUID
	activities []app.ActivityRow
	subtree    []uuid.UUID
	deletions  []app.IssueDeletion
	others     map[uuid.UUID]domain.Issue
	ancestry   map[uuid.UUID][]uuid.UUID
	counts     map[uuid.UUID]app.SubIssueCount
	subItems   []domain.Issue
	parented   *app.ParentRows
	lost       bool
}

func (f *fakeIssues) GreatestSortOrder(ctx context.Context, projectID, stateID uuid.UUID) (*float64, error) {
	if err := f.log.add(ctx, "GreatestSortOrder", projectID, stateID); err != nil {
		return nil, err
	}
	if g, ok := f.greatest[stateID]; ok {
		return &g, nil
	}
	return nil, nil
}

func (f *fakeIssues) InsertIssue(ctx context.Context, r app.IssueRow) error {
	if err := f.log.add(ctx, "InsertIssue", r.ID); err != nil {
		return err
	}
	f.issue = &r
	return nil
}

// writer is the account and the moment of the work item's last write: its
// last update's, or its creation's.
func (f *fakeIssues) writer() (uuid.UUID, time.Time) {
	switch {
	case f.archival != nil:
		return f.archival.By, f.archival.At
	case f.changed != nil:
		return f.changed.By, f.changed.At
	}
	return f.issue.By, f.issue.At
}

func (f *fakeIssues) UpdateIssue(ctx context.Context, r app.IssueFields) error {
	if err := f.log.add(ctx, "UpdateIssue", r.ID); err != nil {
		return err
	}
	if f.issue == nil || r.ID != f.issue.ID {
		return fmt.Errorf("UpdateIssue: %+v is not the work item %+v", r, f.issue)
	}
	f.issue.Name, f.issue.StateID, f.issue.Priority, f.issue.CompletedAt = r.Name, r.StateID, r.Priority, r.CompletedAt
	f.issue.StartDate, f.issue.TargetDate, f.issue.ParentID = r.StartDate, r.TargetDate, r.ParentID
	f.changed = &r
	return nil
}

// link logs an insert of name's rows r and answers their targets, or the
// failure: each row under the work item inserted, by its last writer, then.
func (f *fakeIssues) link(ctx context.Context, name string, r app.LinkRows) ([]uuid.UUID, error) {
	if err := f.log.add(ctx, name, r.Targets); err != nil {
		return nil, err
	}
	if f.issue == nil || r.IssueID != f.issue.ID || r.WorkspaceID != f.issue.WorkspaceID || r.ProjectID != f.issue.ProjectID ||
		len(r.IDs) != len(r.Targets) || slices.Contains(r.IDs, uuid.UUID{}) {
		return nil, fmt.Errorf("%s: %+v not under the work item %+v", name, r, f.issue)
	}
	if by, at := f.writer(); r.By != by || !r.At.Equal(at) {
		return nil, fmt.Errorf("%s: %+v not by the work item's last writer %s at %s", name, r, by, at)
	}
	return r.Targets, nil
}

// unlink logs a removal of name's rows r and answers the targets of from
// left, or the failure: each a row of the work item, by its last writer,
// then.
func (f *fakeIssues) unlink(ctx context.Context, name string, r app.UnlinkRows, from []uuid.UUID) ([]uuid.UUID, error) {
	if err := f.log.add(ctx, name, r.Targets); err != nil {
		return nil, err
	}
	if by, at := f.writer(); f.issue == nil || r.IssueID != f.issue.ID || r.By != by || !r.At.Equal(at) {
		return nil, fmt.Errorf("%s: %+v not of the work item %+v by its last writer", name, r, f.issue)
	}
	return slices.DeleteFunc(slices.Clone(from), func(id uuid.UUID) bool { return slices.Contains(r.Targets, id) }), nil
}

func (f *fakeIssues) InsertAssignees(ctx context.Context, r app.LinkRows) error {
	added, err := f.link(ctx, "InsertAssignees", r)
	f.assignees = append(f.assignees, added...)
	return err
}

func (f *fakeIssues) RemoveAssignees(ctx context.Context, r app.UnlinkRows) error {
	left, err := f.unlink(ctx, "RemoveAssignees", r, f.assignees)
	if err == nil {
		f.assignees = left
	}
	return err
}

func (f *fakeIssues) InsertLabels(ctx context.Context, r app.LinkRows) error {
	added, err := f.link(ctx, "InsertLabels", r)
	f.labels = append(f.labels, added...)
	return err
}

func (f *fakeIssues) RemoveLabels(ctx context.Context, r app.UnlinkRows) error {
	left, err := f.unlink(ctx, "RemoveLabels", r, f.labels)
	if err == nil {
		f.labels = left
	}
	return err
}

func (f *fakeIssues) InsertSubscribers(ctx context.Context, r app.LinkRows) error {
	_, err := f.link(ctx, "InsertSubscribers", r)
	return err
}

func (f *fakeIssues) InsertActivities(ctx context.Context, rows []app.ActivityRow) error {
	if err := f.log.add(ctx, "InsertActivities", len(rows)); err != nil {
		return err
	}
	f.activities = rows
	if f.lost {
		f.lose(rows)
	}
	return nil
}

func (f *fakeIssues) GetIssue(ctx context.Context, id uuid.UUID) (domain.Issue, bool, error) {
	if err := f.log.add(ctx, "GetIssue", id); err != nil {
		return domain.Issue{}, false, err
	}
	i, found := f.read(id)
	return i, found, nil
}

// read is the work item id as the fake has it: one of its others, or the
// work item it keeps; found is false when it has none.
func (f *fakeIssues) read(id uuid.UUID) (domain.Issue, bool) {
	if o, ok := f.others[id]; ok {
		return o, true
	}
	if f.issue == nil || f.issue.ID != id {
		return domain.Issue{}, false
	}
	r := f.issue
	by, at := f.writer()
	return domain.Issue{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, SequenceID: r.SequenceID, Name: r.Name,
		StateID: r.StateID, Priority: r.Priority, StartDate: r.StartDate, TargetDate: r.TargetDate, ParentID: r.ParentID, SortOrder: r.SortOrder,
		CompletedAt: r.CompletedAt, ArchivedAt: f.archivedAt,
		AssigneeIDs: slices.SortedFunc(slices.Values(f.assignees), uuid.UUID.Compare), LabelIDs: slices.SortedFunc(slices.Values(f.labels),
			uuid.UUID.Compare), CreatedByID: creatorOf(r.By), UpdatedByID: &by, CreatedAt: r.At, UpdatedAt: at}, true
}

// creatorOf is a stored work item's creator as a read answers it: none for
// the zero id, as for a work item whose creator's account is purged (P9).
func creatorOf(by uuid.UUID) *uuid.UUID {
	if by == (uuid.UUID{}) {
		return nil
	}
	return &by
}
