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
// transaction. It fails the call named in fail with errDisk, after logging
// it.
type callLog struct {
	calls []string
	fail  string
}

func (l *callLog) add(ctx context.Context, name string, args ...any) error {
	call := line(name, args...)
	if ctx.Value(inTxKey{}) != true {
		call += " outside tx"
	}
	l.calls = append(l.calls, call)
	if name == l.fail {
		return errDisk
	}
	return nil
}

// fakeTx runs fn in a context marked as inside the transaction.
type fakeTx struct{ log *callLog }

func (f fakeTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	f.log.calls = append(f.log.calls, "Begin")
	return fn(context.WithValue(ctx, inTxKey{}, true))
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
// for allowed, with an empty grant, which Locks does not read; it logs
// every call.
type fakeAuthorizer struct {
	log     *callLog
	answers []error
	calls   int
}

func (f *fakeAuthorizer) Authorize(ctx context.Context, actor shared.Actor, action shared.Action, t shared.Target) (shared.Grant, error) {
	n := f.calls
	f.calls++
	if err := f.log.add(ctx, "Authorize", actor.UserID, action, t.WorkspaceID, t.ProjectID); err != nil {
		return shared.Grant{}, err
	}
	if n < len(f.answers) && f.answers[n] != nil {
		return shared.Grant{}, f.answers[n]
	}
	return shared.Grant{}, nil
}

// fakeProject is a project's side of the issue module's ports: its gate,
// its numbers and its catalogs, each call logged. found is whether Find
// answers it, locked what Lock answers, nil for none.
type fakeProject struct {
	log       *callLog
	id        uuid.UUID
	workspace uuid.UUID
	found     bool
	locked    *app.LockedProject
	last      int
	states    []app.StateRef
	labels    []app.LabelRef
	roles     map[uuid.UUID]shared.Role
}

func (f *fakeProject) Find(ctx context.Context, id uuid.UUID) (uuid.UUID, bool, error) {
	if err := f.log.add(ctx, "Find", id); err != nil || id != f.id || !f.found {
		return uuid.UUID{}, false, err
	}
	return f.workspace, true, nil
}

func (f *fakeProject) Lock(ctx context.Context, id uuid.UUID) (app.LockedProject, bool, error) {
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
// item once its activity is written, so that it is not there to read back.
type fakeIssues struct {
	log        *callLog
	greatest   map[uuid.UUID]float64
	issue      *app.IssueRow
	assignees  []uuid.UUID
	labels     []uuid.UUID
	activities []app.ActivityRow
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

// link logs an insert of name's rows r and answers their targets, or the
// failure: each row under the work item inserted, by its creator, then.
func (f *fakeIssues) link(ctx context.Context, name string, r app.LinkRows) ([]uuid.UUID, error) {
	if err := f.log.add(ctx, name, r.Targets); err != nil {
		return nil, err
	}
	if f.issue == nil || r.IssueID != f.issue.ID || r.WorkspaceID != f.issue.WorkspaceID || r.ProjectID != f.issue.ProjectID ||
		r.By != f.issue.By || !r.At.Equal(f.issue.At) || len(r.IDs) != len(r.Targets) || slices.Contains(r.IDs, uuid.UUID{}) {
		return nil, fmt.Errorf("%s: %+v not under the work item %+v", name, r, f.issue)
	}
	return r.Targets, nil
}

func (f *fakeIssues) InsertAssignees(ctx context.Context, r app.LinkRows) (err error) {
	f.assignees, err = f.link(ctx, "InsertAssignees", r)
	return err
}

func (f *fakeIssues) InsertLabels(ctx context.Context, r app.LinkRows) (err error) {
	f.labels, err = f.link(ctx, "InsertLabels", r)
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
		f.issue = nil
	}
	return nil
}

func (f *fakeIssues) GetIssue(ctx context.Context, id uuid.UUID) (domain.Issue, bool, error) {
	if err := f.log.add(ctx, "GetIssue", id); err != nil || f.issue == nil || f.issue.ID != id {
		return domain.Issue{}, false, err
	}
	r := f.issue
	return domain.Issue{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, SequenceID: r.SequenceID, Name: r.Name,
		StateID: r.StateID, Priority: r.Priority, SortOrder: r.SortOrder, CompletedAt: r.CompletedAt,
		AssigneeIDs: slices.SortedFunc(slices.Values(f.assignees), uuid.UUID.Compare), LabelIDs: slices.SortedFunc(slices.Values(f.labels),
			uuid.UUID.Compare), CreatedByID: &r.By, UpdatedByID: &r.By, CreatedAt: r.At, UpdatedAt: r.At}, true, nil
}
