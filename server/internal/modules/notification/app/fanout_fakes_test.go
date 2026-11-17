package app_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The accounts, the work item, its state and its comment the fan-out tests' events name; the workspace and the project
// of the work item's row; and the events' moment.
var (
	alice, bob, carol, dave, eve, frank, gina, gus, hank = uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(),
		uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	workItem, todo, doing, comment = uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	rowWorkspace, rowProject       = uuid.NewV7(), uuid.NewV7()
	moment                         = time.Date(2026, 11, 17, 9, 30, 0, 123456000, time.UTC)
)

func str(s string) *string         { return &s }
func idOf(id uuid.UUID) *uuid.UUID { return &id }

// fanoutEvent is an event of kind on the work item's row's workspace and project at moment, by actor, nil for nerve.
func fanoutEvent(kind string, entity shared.EntityRef, actor *uuid.UUID, changes ...shared.Change) shared.Event {
	return shared.Event{ID: uuid.NewV7(), Type: kind, OccurredAt: moment, WorkspaceID: rowWorkspace, ProjectID: rowProject, ActorID: actor,
		Entity: entity, Changes: changes}
}

// renamed is alice's issue.updated of the work item: its name (0), its description (1, which notifies of nothing),
// its state (2), and mentions of eve (3) and gus (4).
func renamed() shared.Event {
	return fanoutEvent("issue.updated", shared.EntityRef{Kind: "issue", ID: workItem}, &alice,
		shared.Change{Field: "name", Old: str("Fix it"), New: str("Fix it now")},
		shared.Change{Field: "description"},
		shared.Change{Field: "state", Old: str("Todo"), New: str("Doing"), OldID: idOf(todo), NewID: idOf(doing)},
		shared.Change{Field: "mention", NewID: idOf(eve)},
		shared.Change{Field: "mention", NewID: idOf(gus)})
}

// commented is bob's issue.comment_created of the comment on the work item: its HTML (0), a mention of carol (1) and
// the work item (2).
func commented() shared.Event {
	return fanoutEvent("issue.comment_created", shared.EntityRef{Kind: "comment", ID: comment}, &bob,
		shared.Change{Field: "comment_html"}, shared.Change{Field: "mention", NewID: idOf(carol)},
		shared.Change{Field: "issue", NewID: idOf(workItem)})
}

// txKey marks the context of the fake transaction.
type txKey struct{}

// fanoutTx runs fn in a context the ports can tell is the transaction's.
type fanoutTx struct{}

func (fanoutTx) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(context.WithValue(ctx, txKey{}, true))
}

// fanoutPorts are every port of a fan-out, as fakes over the work item: its row (none unless exists), its columns,
// its subscribers, its comment's text (nil for none), its state's name (nil for none), its project's identifier (nil
// for none), the recheck's decisions and the guest rule, which hides the work item from the accounts hidden names.
// Each call is recorded, in order, as its name and arguments, marked when it is made outside the transaction; the
// call named fail fails. Each question of the guest rule, which reads nothing, is recorded apart, in asked.
type fanoutPorts struct {
	exists      bool
	locked      app.LockedIssue
	columns     app.IssueColumns
	subscribers []uuid.UUID
	text        *string
	state       *app.StateName
	identifier  *string
	decisions   map[uuid.UUID]domain.Decision
	fail        string
	failure     error
	hidden      []uuid.UUID
	calls       []string
	asked       []string
	inserted    []app.Notifications
}

// newFanoutPorts are the ports of the work item of the row's workspace and project, whose creator is carol and whose
// assignee is dave, in the state Doing of WEB.
func newFanoutPorts() *fanoutPorts {
	return &fanoutPorts{exists: true, locked: app.LockedIssue{WorkspaceID: rowWorkspace, ProjectID: rowProject},
		columns:    app.IssueColumns{Name: "Fix it now", SequenceID: 12, StateID: doing, CreatedByID: &carol, AssigneeIDs: []uuid.UUID{dave}},
		state:      &app.StateName{Name: "Doing", Group: "started"},
		identifier: str("WEB"), failure: errors.New("the port failed")}
}

// callLine is a call as fanoutPorts records it: its name and its arguments.
func callLine(name string, args ...any) string {
	parts := []string{name}
	for _, a := range args {
		parts = append(parts, fmt.Sprint(a))
	}
	return strings.Join(parts, " ")
}

func (f *fanoutPorts) call(ctx context.Context, name string, args ...any) error {
	line := callLine(name, args...)
	if ctx.Value(txKey{}) == nil {
		line += " outside the transaction"
	}
	f.calls = append(f.calls, line)
	if name == f.fail {
		return f.failure
	}
	return nil
}

func (f *fanoutPorts) ShareIssue(ctx context.Context, id uuid.UUID) (app.LockedIssue, bool, error) {
	if err := f.call(ctx, "ShareIssue", id); err != nil || !f.exists {
		return app.LockedIssue{}, false, err
	}
	return f.locked, true, nil
}

func (f *fanoutPorts) NotificationIssue(ctx context.Context, id uuid.UUID) (app.IssueColumns, bool, error) {
	if err := f.call(ctx, "NotificationIssue", id); err != nil || !f.exists {
		return app.IssueColumns{}, false, err
	}
	return f.columns, true, nil
}

func (f *fanoutPorts) SubscriberIDs(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error) {
	if err := f.call(ctx, "SubscriberIDs", id); err != nil {
		return nil, err
	}
	return f.subscribers, nil
}

func (f *fanoutPorts) CommentText(ctx context.Context, id uuid.UUID) (string, bool, error) {
	if err := f.call(ctx, "CommentText", id); err != nil || f.text == nil {
		return "", false, err
	}
	return *f.text, true, nil
}

func (f *fanoutPorts) State(ctx context.Context, project, state uuid.UUID) (app.StateName, bool, error) {
	if err := f.call(ctx, "State", project, state); err != nil || f.state == nil {
		return app.StateName{}, false, err
	}
	return *f.state, true, nil
}

func (f *fanoutPorts) Identifier(ctx context.Context, workspace, project uuid.UUID) (string, bool, error) {
	if err := f.call(ctx, "Identifier", workspace, project); err != nil || f.identifier == nil {
		return "", false, err
	}
	return *f.identifier, true, nil
}

func (f *fanoutPorts) DecideMany(ctx context.Context, action shared.Action, workspace, project uuid.UUID, accounts []uuid.UUID) (map[uuid.UUID]domain.Decision, error) {
	if err := f.call(ctx, "DecideMany", action, workspace, project, accounts); err != nil {
		return nil, err
	}
	return f.decisions, nil
}

// HiddenFrom is the guest rule as a fake: it hides the work item from the accounts f.hidden names, whatever its
// creator and the grant; the question, its creator ("-" for none), the account and the grant's OwnOnly, is recorded.
func (f *fanoutPorts) HiddenFrom(creator *uuid.UUID, account uuid.UUID, ownOnly bool) bool {
	who := "-"
	if creator != nil {
		who = creator.String()
	}
	f.asked = append(f.asked, callLine("HiddenFrom", who, account, ownOnly))
	return slices.Contains(f.hidden, account)
}

func (f *fanoutPorts) InsertNotifications(ctx context.Context, n app.Notifications) error {
	if err := f.call(ctx, "InsertNotifications", len(n.Rows)); err != nil {
		return err
	}
	f.inserted = append(f.inserted, n)
	return nil
}

// fanout is the use case over the ports, logging into logs.
func fanout(f *fanoutPorts, logs *bytes.Buffer) *app.Fanout {
	return app.NewFanout(app.FanoutDeps{Lock: f, Issues: f, Projects: f, Readers: f, GuestRule: f, Rows: f, Tx: fanoutTx{},
		Logger: slog.New(slog.NewTextHandler(logs, nil))})
}

// sent is a row as a test expects it, its id aside: its receiver, its sender, its position and its change.
type sent struct {
	Receiver uuid.UUID
	Sender   domain.Sender
	Seq      int16
	Change   domain.ChangeSnapshot
}

// sentRows are n's rows as sent, in their order, each with the snapshot of the work item as newFanoutPorts has it;
// a row with another snapshot, or an id that is zero or another row's, is reported as bad.
func sentRows(n app.Notifications) (rows []sent, bad []string) {
	want := domain.IssueSnapshot{ID: workItem, Name: "Fix it now", SequenceID: 12, Identifier: "WEB", StateName: "Doing", StateGroup: "started"}
	seen := map[uuid.UUID]bool{}
	for _, r := range n.Rows {
		if r.ID == (uuid.UUID{}) || seen[r.ID] || r.Data.Issue != want {
			bad = append(bad, fmt.Sprintf("%+v", r))
		}
		seen[r.ID] = true
		rows = append(rows, sent{Receiver: r.ReceiverID, Sender: r.Sender, Seq: r.ChangeSeq, Change: r.Data.Change})
	}
	return rows, bad
}
