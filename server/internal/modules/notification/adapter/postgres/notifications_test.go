package postgresadapter_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// stored is a notification as the table holds it, its id aside.
type stored struct {
	Workspace, Project, Receiver, Issue, Event uuid.UUID
	Sender                                     string
	Seq                                        int16
	Data                                       domain.Data
	Changed                                    []string
	TriggeredBy, IssueCreatedBy, By            *uuid.UUID
	Created, Updated                           time.Time
	Read, Archived, Snoozed, Deleted           *time.Time
}

// storedNotifications reads the issue's notifications, deleted or not, in the order of their receivers and positions,
// and of their ids, and the keys each change keeps.
func storedNotifications(t *testing.T, pool *pgxpool.Pool, issue uuid.UUID) ([]uuid.UUID, []stored) {
	t.Helper()
	rows, err := pool.Query(context.Background(), `SELECT id, workspace_id, project_id, receiver_id, issue_id, event_id, sender, change_seq, data,
		array(SELECT jsonb_object_keys(data->'change') ORDER BY 1), triggered_by_id, issue_created_by_id, coalesce(created_by_id, updated_by_id),
		created_at, updated_at, read_at, archived_at, snoozed_till, deleted_at
		FROM notifications WHERE issue_id = $1 ORDER BY receiver_id, change_seq, id`, issue)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var ids []uuid.UUID
	var got []stored
	for rows.Next() {
		var id uuid.UUID
		var s stored
		var data []byte
		if err := rows.Scan(&id, &s.Workspace, &s.Project, &s.Receiver, &s.Issue, &s.Event, &s.Sender, &s.Seq, &data, &s.Changed, &s.TriggeredBy,
			&s.IssueCreatedBy, &s.By, &s.Created, &s.Updated, &s.Read, &s.Archived, &s.Snoozed, &s.Deleted); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &s.Data); err != nil {
			t.Fatal(err)
		}
		s.Created, s.Updated = s.Created.UTC(), s.Updated.UTC()
		ids, got = append(ids, id), append(got, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return ids, got
}

// A fan-out's notifications are inserted in the transaction ctx carries, each as sent: the shared workspace, project,
// work item, event, sender account, creator and moment, its own receiver, sender, position and snapshot, keeping only
// the keys its change has; nerve's audit, none. The same notifications sent again, with ids of their own, as a retried
// job sends them, add none; once one is deleted, it is sent anew, and only it (M7 P1b spec 2.9, R4).
func TestInsertNotifications(t *testing.T) {
	s, pool := newStore(t)
	w := newWorld(t, pool)
	alice, carol := uuid.NewV7(), uuid.NewV7()
	exec(t, pool, "INSERT INTO users (id, email, password, display_name) VALUES ($1, 'alice@corp.com', 'x', 'alice'), ($2, 'carol@corp.com', 'x', 'carol')", alice, carol)
	issue, event := uuid.NewV7(), uuid.NewV7()
	at := time.Date(2026, 11, 17, 9, 30, 0, 123456000, time.UTC)
	snapshot := domain.IssueSnapshot{ID: issue, Name: "Fix it", SequenceID: 12, Identifier: "WEB", StateName: "Todo", StateGroup: "unstarted"}
	created := domain.Data{Issue: snapshot, Change: domain.ChangeSnapshot{Field: "issue", Verb: domain.VerbCreated}}
	assigned := domain.Data{Issue: snapshot, Change: domain.ChangeSnapshot{Field: "assignees", Verb: domain.VerbCreated, NewValue: str("bob"), NewID: &w.receiver}}
	mentioned := domain.Data{Issue: snapshot, Change: domain.ChangeSnapshot{Field: "description", Verb: domain.VerbMentioned}}
	sent := func() app.Notifications {
		return app.Notifications{WorkspaceID: w.workspace, ProjectID: w.project, IssueID: issue, EventID: event, IssueCreatedByID: &alice,
			TriggeredByID: &alice, At: at, Rows: []app.NotificationRow{
				{ID: uuid.NewV7(), ReceiverID: w.receiver, Sender: domain.SenderAssigned, ChangeSeq: domain.CreationSeq, Data: created},
				{ID: uuid.NewV7(), ReceiverID: w.receiver, Sender: domain.SenderAssigned, ChangeSeq: 10, Data: assigned},
				{ID: uuid.NewV7(), ReceiverID: carol, Sender: domain.SenderMentioned, ChangeSeq: 15, Data: mentioned}}}
	}
	rollback := errors.New("rollback the caller")
	err := postgres.NewTxManager(pool, time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		if err := s.InsertNotifications(ctx, sent()); err != nil {
			t.Fatal(err)
		}
		return rollback
	})
	if ids, _ := storedNotifications(t, pool, issue); !errors.Is(err, rollback) || len(ids) != 0 {
		t.Fatalf("an insert the caller rolled back: %v, %v; want nothing", err, ids)
	}
	first := sent()
	if err := s.InsertNotifications(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	row := func(receiver uuid.UUID, sender string, seq int16, data domain.Data, keys ...string) stored {
		return stored{Workspace: w.workspace, Project: w.project, Receiver: receiver, Issue: issue, Event: event, Sender: sender, Seq: seq,
			Data: data, Changed: keys, TriggeredBy: &alice, IssueCreatedBy: &alice, Created: at, Updated: at}
	}
	want := []stored{row(w.receiver, "assigned", -1, created, "field", "verb"),
		row(w.receiver, "assigned", 10, assigned, "field", "new_id", "new_value", "verb"),
		row(carol, "mentioned", 15, mentioned, "field", "verb")}
	if w.receiver.Compare(carol) > 0 {
		want = append(want[2:], want[:2]...)
	}
	ids, got := storedNotifications(t, pool, issue)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("notifications\n%+v\nwant\n%+v", got, want)
	}
	if !slices.ContainsFunc(first.Rows, func(r app.NotificationRow) bool { return r.ID == ids[0] }) {
		t.Fatalf("ids %v; want the ones sent", ids)
	}
	if err := s.InsertNotifications(context.Background(), sent()); err != nil {
		t.Fatal(err)
	}
	if again, got := storedNotifications(t, pool, issue); !slices.Equal(again, ids) || !reflect.DeepEqual(got, want) {
		t.Fatalf("sent again: %v %+v; want the first copy alone", again, got)
	}
	deleted := at.Add(time.Hour)
	exec(t, pool, "UPDATE notifications SET deleted_at = $2, updated_at = $2 WHERE receiver_id = $1 AND issue_id = $3", carol, deleted, issue)
	retry := sent()
	if err := s.InsertNotifications(context.Background(), retry); err != nil {
		t.Fatal(err)
	}
	after, got := storedNotifications(t, pool, issue)
	var fresh []uuid.UUID
	for n, id := range after {
		if !slices.Contains(ids, id) {
			fresh = append(fresh, id)
			if got[n].Receiver != carol || got[n].Deleted != nil || got[n].Seq != 15 {
				t.Fatalf("sent anew %+v; want carol's mention", got[n])
			}
		}
	}
	if len(after) != 4 || !slices.Equal(fresh, []uuid.UUID{retry.Rows[2].ID}) {
		t.Fatalf("after carol's was deleted: %v (new %v); want hers alone sent anew", after, fresh)
	}
}

func str(s string) *string { return &s }
