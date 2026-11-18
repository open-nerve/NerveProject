package postgresadapter_test

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
)

// readWorld is the world and what the reads tell bob's notifications from (M7 P1c spec 2.3, 2.5): alice, who
// creates work items; carol, another receiver; Ops, a project of the workspace of whose work items bob reads only
// those he created; Secret, one he does not read; and Elsewhere, the project of another workspace, which bob reads,
// so its workspace alone keeps its notifications out.
type readWorld struct {
	world
	alice, carol, ops, secret, other, elsewhere uuid.UUID
}

func newReadWorld(t *testing.T, pool *pgxpool.Pool) readWorld {
	t.Helper()
	w := readWorld{world: newWorld(t, pool), alice: uuid.NewV7(), carol: uuid.NewV7(), ops: uuid.NewV7(), secret: uuid.NewV7(), other: uuid.NewV7(),
		elsewhere: uuid.NewV7()}
	exec(t, pool, "INSERT INTO users (id, email, password, display_name) VALUES ($1, 'alice@corp.com', 'x', 'alice'), ($2, 'carol@corp.com', 'x', 'carol')",
		w.alice, w.carol)
	exec(t, pool, "INSERT INTO projects (id, workspace_id, name, identifier) VALUES ($1, $3, 'Ops', 'OPS'), ($2, $3, 'Secret', 'SEC')", w.ops, w.secret, w.workspace)
	exec(t, pool, "INSERT INTO workspaces (id, name, slug) VALUES ($1, 'Other', 'other')", w.other)
	exec(t, pool, "INSERT INTO projects (id, workspace_id, name, identifier) VALUES ($1, $2, 'Elsewhere', 'ELS')", w.elsewhere, w.other)
	return w
}

// inbox is bob's notifications in the world's workspace at now: of Web whole, of Ops his own only, and of
// Elsewhere.
func (w readWorld) inbox() app.Inbox {
	return app.Inbox{ReceiverID: w.receiver, WorkspaceID: w.workspace, Now: now,
		Grants: []app.Grant{{ProjectID: w.project}, {ProjectID: w.ops, OwnOnly: true}, {ProjectID: w.elsewhere}}}
}

// note is a notification a read test writes: purgeID(n), of the work item purgeID(1000+n), created at created (n
// minutes before now when zero), of sender, in project (Web when zero) of workspace (the world's when zero), for
// receiver (bob when zero), whose work item's creator is creator (nil: not known), sent by by (nil: nerve); read,
// archived and deleted a minute before now when it says so; snoozed till snoozed.
type note struct {
	n                            int
	created                      time.Time
	sender                       domain.Sender
	project, workspace, receiver uuid.UUID
	creator, by                  *uuid.UUID
	read, archived, deleted      bool
	snoozed                      *time.Time
}

// minuteAgo is when a note is read, archived or deleted.
var minuteAgo = now.Add(-time.Minute)

// put writes the notes.
func put(t *testing.T, pool *pgxpool.Pool, w readWorld, notes ...note) {
	t.Helper()
	for _, n := range notes {
		created, project, workspace, receiver := n.created, n.project, n.workspace, n.receiver
		if created.IsZero() {
			created = now.Add(-time.Duration(n.n) * time.Minute)
		}
		if project == (uuid.UUID{}) {
			project = w.project
		}
		if workspace == (uuid.UUID{}) {
			workspace = w.workspace
		}
		if receiver == (uuid.UUID{}) {
			receiver = w.receiver
		}
		at := func(on bool) *time.Time {
			if on {
				return &minuteAgo
			}
			return nil
		}
		data, err := json.Marshal(noteData(n.n))
		if err != nil {
			t.Fatal(err)
		}
		exec(t, pool, `INSERT INTO notifications (id, workspace_id, project_id, receiver_id, triggered_by_id, issue_id, issue_created_by_id, sender, data,
			event_id, change_seq, read_at, archived_at, snoozed_till, created_at, updated_at, deleted_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, gen_random_uuid(), 0, $10, $11, $12, $13, $13, $14)`,
			purgeID(n.n), workspace, project, receiver, n.by, purgeID(1000+n.n), n.creator, string(n.sender), data, at(n.read), at(n.archived), n.snoozed,
			created, at(n.deleted))
	}
}

// noteData is the snapshot of note n.
func noteData(n int) domain.Data {
	done := "Done"
	return domain.Data{Issue: domain.IssueSnapshot{ID: purgeID(1000 + n), Name: fmt.Sprintf("item %d", n), SequenceID: n, Identifier: "WEB", StateName: "Todo",
		StateGroup: "unstarted"}, Change: domain.ChangeSnapshot{Field: "state", Verb: domain.VerbUpdated, NewValue: &done}}
}

// readNotes are bob's notifications the list, the unread counts and mark-all-read tell from one another and from
// those he may not see.
func readNotes(w readWorld) []note {
	alice, bob := &w.alice, &w.receiver
	later, soon, at, past := now.Add(time.Hour), now.Add(time.Microsecond), now, now.Add(-time.Hour)
	return []note{
		{n: 1, sender: domain.SenderAssigned, creator: alice, by: alice},
		{n: 2, sender: domain.SenderCreated, creator: alice, read: true},
		{n: 3, sender: domain.SenderSubscribed, creator: alice, archived: true},
		{n: 4, sender: domain.SenderMentioned, creator: alice},
		{n: 5, sender: domain.SenderSubscribed, snoozed: &later},
		{n: 6, sender: domain.SenderSubscribed, creator: alice, snoozed: &at},
		{n: 7, sender: domain.SenderSubscribed, creator: alice, snoozed: &soon},
		{n: 8, sender: domain.SenderSubscribed, creator: alice, snoozed: &past},
		{n: 9, sender: domain.SenderAssigned, project: w.ops, creator: bob},
		{n: 10, sender: domain.SenderAssigned, project: w.ops, creator: alice},
		{n: 11, sender: domain.SenderAssigned, project: w.ops},
		{n: 12, sender: domain.SenderAssigned, read: true},
		{n: 13, sender: domain.SenderAssigned, project: w.secret, creator: alice},
		{n: 14, sender: domain.SenderAssigned, creator: alice, deleted: true},
		{n: 15, sender: domain.SenderAssigned, creator: alice, receiver: w.carol},
		{n: 16, sender: domain.SenderAssigned, creator: alice, project: w.elsewhere, workspace: w.other},
		{n: 17, sender: domain.SenderMentioned, creator: alice, by: alice, read: true, archived: true, snoozed: &later},
		{n: 18, sender: domain.SenderAssigned, creator: alice, archived: true, snoozed: &later},
		{n: 19, sender: domain.SenderMentioned, project: w.ops, creator: alice},
	}
}

// noteIDs are the ids of notes ns, in their order.
func noteIDs(ns ...int) []uuid.UUID {
	ids := make([]uuid.UUID, len(ns))
	for i, n := range ns {
		ids[i] = purgeID(n)
	}
	return ids
}

// listed lists r and answers the notifications' ids in the order listed.
func listed(t *testing.T, s *postgresadapter.Store, r app.ListRead) []uuid.UUID {
	t.Helper()
	page, err := s.ListNotifications(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]uuid.UUID, len(page))
	for i, n := range page {
		ids[i] = n.ID
	}
	return ids
}

func senders(s ...domain.Sender) []domain.Sender { return s }

// The list answers bob's undeleted notifications in the workspace, newest first, of the projects he reads and, in
// an own-only one, only of the work items he created, an unknown creator's not; of the filter's senders, so the all
// tab has no mention; read, unread or both; archived or not; snoozed at now or not, a snooze till now itself
// being over and one a microsecond later not (M7 P1c spec 2.3, 2.5, R2, R6). Carol's, another workspace's, a
// deleted one and one of a project he does not read never show.
func TestListNotifications(t *testing.T) {
	s, pool := newStore(t)
	w := newReadWorld(t, pool)
	put(t, pool, w, readNotes(w)...)
	all, mentions := domain.TabSenders(domain.TabAll), domain.TabSenders(domain.TabMentions)
	yes, no := true, false
	for _, tt := range []struct {
		name   string
		filter domain.ListFilter
		want   []int
	}{
		{"the all tab", domain.ListFilter{Filter: domain.Filter{Senders: all}}, []int{1, 2, 6, 8, 9, 12}},
		{"the mentions tab", domain.ListFilter{Filter: domain.Filter{Senders: mentions}}, []int{4}},
		{"assigned", domain.ListFilter{Filter: domain.Filter{Senders: senders(domain.SenderAssigned)}}, []int{1, 9, 12}},
		{"created and subscribed", domain.ListFilter{Filter: domain.Filter{Senders: senders(domain.SenderCreated, domain.SenderSubscribed)}}, []int{2, 6, 8}},
		{"read", domain.ListFilter{Filter: domain.Filter{Senders: all}, Read: &yes}, []int{2, 12}},
		{"unread", domain.ListFilter{Filter: domain.Filter{Senders: all}, Read: &no}, []int{1, 6, 8, 9}},
		{"archived", domain.ListFilter{Filter: domain.Filter{Senders: all, Archived: true}}, []int{3}},
		{"snoozed", domain.ListFilter{Filter: domain.Filter{Senders: all, Snoozed: true}}, []int{5, 7}},
		{"assigned and unread", domain.ListFilter{Filter: domain.Filter{Senders: senders(domain.SenderAssigned)}, Read: &no}, []int{1, 9}},
		{"archived and snoozed", domain.ListFilter{Filter: domain.Filter{Senders: all, Archived: true, Snoozed: true}}, []int{18}},
		{"mentions read, archived and snoozed", domain.ListFilter{Filter: domain.Filter{Senders: mentions, Archived: true, Snoozed: true}, Read: &yes}, []int{17}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := listed(t, s, app.ListRead{Inbox: w.inbox(), Filter: tt.filter, Limit: 50}); !slices.Equal(got, noteIDs(tt.want...)) {
				t.Fatalf("listed %v; want %v", got, noteIDs(tt.want...))
			}
		})
	}
	page, err := s.ListNotifications(context.Background(), app.ListRead{Inbox: w.inbox(), Limit: 50,
		Filter: domain.ListFilter{Filter: domain.Filter{Senders: mentions, Archived: true, Snoozed: true}}})
	later := now.Add(time.Hour)
	want := []app.Notification{{ID: purgeID(17), WorkspaceID: w.workspace, ProjectID: w.project, IssueID: purgeID(1017), Sender: domain.SenderMentioned,
		TriggeredByID: &w.alice, Data: noteData(17), State: domain.State{ReadAt: &minuteAgo, ArchivedAt: &minuteAgo, SnoozedTill: &later},
		CreatedAt: now.Add(-17 * time.Minute)}}
	if err != nil || !reflect.DeepEqual(page, want) {
		t.Fatalf("listed %+v, %v; want %+v", page, err, want)
	}
	for _, limit := range []int{0, 2147483648} {
		if _, err := s.ListNotifications(context.Background(), app.ListRead{Inbox: w.inbox(), Filter: domain.ListFilter{Filter: domain.Filter{Senders: all}},
			Limit: limit}); err == nil {
			t.Fatalf("accepted the limit %d", limit)
		}
	}
}

// Pages follow one another by the cursor of the last row, (created_at, id) descending, notifications of one moment
// by id, with no row twice and none left out; every condition precedes the LIMIT, so rows he may not see between
// them take no place; a page is at most the limit, which takes the caller's lookahead.
func TestListNotificationsPages(t *testing.T) {
	s, pool := newStore(t)
	w := newReadWorld(t, pool)
	t0 := now.Add(-time.Hour)
	alice := &w.alice
	put(t, pool, w,
		note{n: 1, created: t0, sender: domain.SenderSubscribed, creator: alice},
		note{n: 2, created: t0, sender: domain.SenderSubscribed, creator: alice},
		note{n: 3, created: t0, sender: domain.SenderSubscribed, creator: alice},
		note{n: 4, created: t0, sender: domain.SenderSubscribed, creator: alice},
		note{n: 5, created: t0.Add(-time.Minute), sender: domain.SenderSubscribed, creator: alice},
		note{n: 6, created: t0.Add(-time.Minute), sender: domain.SenderSubscribed, creator: alice},
		note{n: 7, created: t0.Add(-2 * time.Minute), sender: domain.SenderSubscribed, creator: alice},
		note{n: 8, created: t0, sender: domain.SenderSubscribed, creator: alice, deleted: true},
		note{n: 9, created: t0, sender: domain.SenderSubscribed, creator: alice, receiver: w.carol},
		note{n: 10, created: t0.Add(-time.Minute), sender: domain.SenderSubscribed, creator: alice, project: w.secret},
		note{n: 11, created: t0.Add(-2 * time.Minute), sender: domain.SenderSubscribed, creator: alice, project: w.ops})
	filter := domain.ListFilter{Filter: domain.Filter{Senders: domain.TabSenders(domain.TabAll)}}
	if got, want := listed(t, s, app.ListRead{Inbox: w.inbox(), Filter: filter, Limit: 3}), noteIDs(4, 3, 2); !slices.Equal(got, want) {
		t.Fatalf("the first page with its lookahead %v; want %v", got, want)
	}
	var got []uuid.UUID
	var after *domain.ListCursor
	for range 10 {
		page, err := s.ListNotifications(context.Background(), app.ListRead{Inbox: w.inbox(), Filter: filter, After: after, Limit: 2})
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range page {
			got = append(got, n.ID)
		}
		if len(page) < 2 {
			break
		}
		last := page[len(page)-1]
		after = &domain.ListCursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}
	if want := noteIDs(4, 3, 2, 1, 6, 5, 7); !slices.Equal(got, want) {
		t.Fatalf("paged %v; want %v", got, want)
	}
}

// The two unread counts are bob's unread, unarchived notifications of each tab's senders not snoozed at now, a
// snooze till now itself or before being over, of the projects he reads, own-only as the list (spec 2.2, R2, R3,
// R6): read, archived, snoozed, deleted, carol's, another workspace's and those he may not see are not counted.
func TestCountUnreadNotifications(t *testing.T) {
	s, pool := newStore(t)
	w := newReadWorld(t, pool)
	put(t, pool, w, readNotes(w)...)
	got, err := s.CountUnread(context.Background(), app.UnreadRead{Inbox: w.inbox(), All: domain.TabSenders(domain.TabAll),
		Mentions: domain.TabSenders(domain.TabMentions)})
	if want := (app.UnreadCounts{All: 4, Mentions: 1}); err != nil || got != want {
		t.Fatalf("unread %+v, %v; want %+v (1, 6, 8, 9 and 4)", got, err, want)
	}
	none := app.Inbox{ReceiverID: w.receiver, WorkspaceID: w.workspace, Now: now}
	got, err = s.CountUnread(context.Background(), app.UnreadRead{Inbox: none, All: domain.TabSenders(domain.TabAll), Mentions: domain.TabSenders(domain.TabMentions)})
	if err != nil || got != (app.UnreadCounts{}) {
		t.Fatalf("unread of no project %+v, %v; want none", got, err)
	}
}
