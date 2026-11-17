package bootstrap

import (
	"cmp"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	notificationriver "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/river"
	notificationdomain "github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fanoutOf is the arguments of the one notification.fanout job of the event of type on the work item item: the event
// as the issue module published it but its HTML (TestTheFanoutJobKeepsTheEventsPositionsWithoutItsHTML), its id, its
// moment, its actor and every change at its position.
func fanoutOf(t *testing.T, pool *pgxpool.Pool, eventType string, item uuid.UUID) notificationriver.FanoutArgs {
	t.Helper()
	rows, err := pool.Query(pgtest.Soon(t), `SELECT args FROM river_job WHERE kind = $1 AND args->>'type' = $2 AND args->>'entity_id' = $3`,
		notificationriver.FanoutKind, eventType, item.String())
	if err != nil {
		t.Fatal(err)
	}
	raws, err := pgx.CollectRows(rows, pgx.RowTo[[]byte])
	if err != nil || len(raws) != 1 {
		t.Fatalf("the fan-outs of %s on %s: %d, %v; want one", eventType, item, len(raws), err)
	}
	var job notificationriver.FanoutArgs
	if err := json.Unmarshal(raws[0], &job); err != nil {
		t.Fatal(err)
	}
	return job
}

// assignedAt is the position of the job's change assigning the account, as positionOf reads an event's.
func assignedAt(t *testing.T, job notificationriver.FanoutArgs, account uuid.UUID) int16 {
	t.Helper()
	for n, c := range job.Changes {
		if c.Field == "assignees" && c.NewID != nil && *c.NewID == account {
			return int16(n)
		}
	}
	t.Fatalf("%s has no assignment of %s: %+v", job.Type, account, job.Changes)
	return 0
}

// sending is a notification a fan-out sends: its receiver, its sender, its position and its change.
type sending struct {
	receiver uuid.UUID
	sender   string
	seq      int16
	change   notificationdomain.ChangeSnapshot
}

// sentOf are the rows job's fan-out sends of the work item creator made, showing it as issue, written by actor, nil
// for nerve: sends, each in Web of acme at the event's moment, with no audit account, unread, live.
func (w issueWorld) sentOf(job notificationriver.FanoutArgs, actor, creator *uuid.UUID, issue notificationdomain.IssueSnapshot,
	sends ...sending) []sentNotification {
	rows := make([]sentNotification, len(sends))
	for i, s := range sends {
		rows[i] = sentNotification{Receiver: s.receiver, Sender: s.sender, Seq: s.seq, Data: notificationdomain.Data{Issue: issue, Change: s.change},
			Workspace: w.acme, Project: w.web, Issue: issue.ID, Event: job.EventID, TriggeredBy: actor, IssueCreatedBy: creator,
			Created: job.OccurredAt.UTC(), Updated: job.OccurredAt.UTC()}
	}
	return rows
}

// assertSent checks that the work item's notifications, deleted or not, are want and no other, in the order
// sentNotifications reads them: by event, in the order of events, then by receiver and position.
func assertSent(t *testing.T, pool *pgxpool.Pool, item uuid.UUID, events []uuid.UUID, want []sentNotification) {
	t.Helper()
	slices.SortStableFunc(want, func(a, b sentNotification) int {
		return cmp.Or(cmp.Compare(slices.Index(events, a.Event), slices.Index(events, b.Event)), a.Receiver.Compare(b.Receiver), cmp.Compare(a.Seq, b.Seq))
	})
	if _, got := sentNotifications(t, pool, item, events...); !reflect.DeepEqual(got, want) {
		shown, _ := json.Marshal(got)
		wanted, _ := json.Marshal(want)
		t.Errorf("the notifications of %s\n%s\nwant\n%s", item, shown, wanted)
	}
}

// sentData is the snapshot of the one notification of the event to the receiver of the change at seq, as the table
// holds it: every key it writes, decoded.
func sentData(t *testing.T, pool *pgxpool.Pool, event, receiver uuid.UUID, seq int16) map[string]any {
	t.Helper()
	var raw []byte
	if err := pool.QueryRow(pgtest.Soon(t), `SELECT data FROM notifications WHERE event_id = $1 AND receiver_id = $2 AND change_seq = $3`,
		event, receiver, seq).Scan(&raw); err != nil {
		t.Fatalf("the notification of %s to %s at %d: %v", event, receiver, seq, err)
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	return data
}

// webState is one of Web's states: its id, name and group.
type webState struct {
	id          uuid.UUID
	name, group string
}

// changed is a change of field an update notifies of, verb updated, with the values and ids it has, nil for none.
func changed(field string, before, after *string, beforeID, afterID *uuid.UUID) notificationdomain.ChangeSnapshot {
	return notificationdomain.ChangeSnapshot{Field: field, Verb: "updated", OldValue: before, NewValue: after, OldID: beforeID, NewID: afterID}
}

// fieldWrite is a write of a work item's fields through the API, and what it sends: the body of alice's PATCH of a
// work item she made as Fix in Todo assigned to bob, the work item's name and state after it, and the notifications.
type fieldWrite struct {
	name, body string
	after      string
	state      webState
	sends      []sending
}

// Each notifying write of a work item's through the API sends what M7 P1b spec 2.3–2.5 say, read back from the table
// once the app's fan-outs have run. bob's creation of a work item assigned to himself and walt sends walt, its
// assignee, the creation itself at -1 and each assignment at its position in the event, not bob, who wrote it. alice's
// change of each field an update notifies of, on a work item she made assigned to bob, sends bob that change at its
// position, updated, with its old and new values and ids, the parent by its id alone, and the snapshot of the work item
// as it is then; alice, its creator and subscriber, receives nothing she wrote. Changes that notify of nothing keep
// their positions: of a write of the name, the description, the priority and the sorting, the name is sent at 0 and the
// priority at 2. An assignee alice takes off is sent his removal as subscribed, the one she adds, subscribed by it, as
// assigned. Archiving and restoring send manual_archive, then archive to restore. The sender is settled when a change is
// sent, created before assigned before subscribed: alice's change of the priority of bob's work item sends bob, its
// creator and assignee, created, walt assigned, and wanda, who subscribed, subscribed. nerve's own archiving of a work
// item done long ago sends its creator alice too, as created, triggered by no one, at the run's moment. Each row is in
// the work item's workspace and project, at its event's moment, by its actor; its snapshot writes exactly the keys it
// has.
func TestWorkItemWritesNotifyThroughTheApp(t *testing.T) {
	w := newIssueWorld(t)
	alice, bob, walt, wanda := w.ids["alice"], w.ids["bob"], w.ids["walt"], w.ids["wanda"]
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	w.makeMember(t, "wanda", w.web, shared.RoleAdmin)
	names := map[uuid.UUID]string{}
	for _, account := range []uuid.UUID{bob, walt} {
		var name string
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT display_name FROM users WHERE id = $1", account).Scan(&name); err != nil {
			t.Fatal(err)
		}
		names[account] = name
	}
	bobName, waltName := names[bob], names[walt]
	todo, done := webState{w.todo, "Todo", "unstarted"}, webState{w.done, "Done", "completed"}
	started := webState{stateID(t, w.pool, w.web, "In Progress"), "In Progress", "started"}
	made := func(by, name string, s webState, assignees ...uuid.UUID) notificationdomain.IssueSnapshot {
		t.Helper()
		var answer struct {
			ID         uuid.UUID `json:"id"`
			SequenceID int       `json:"sequence_id"`
		}
		body := fmt.Sprintf(`{"name":%q,"state_id":%q,"assignee_ids":%s}`, name, s.id, idArray(assignees...))
		decodeAnswer(t, w.send(t, by, http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", body, http.StatusCreated), &answer)
		return notificationdomain.IssueSnapshot{ID: answer.ID, Name: name, SequenceID: answer.SequenceID, Identifier: "WEB", StateName: s.name,
			StateGroup: s.group}
	}
	parent := made("alice", "Parent", todo)
	team := made("bob", "Fix the login", todo, bob, walt)
	none, v := eventValue(""), eventValue[string]
	writes := []fieldWrite{
		{"name", `{"name":"Fix it"}`, "Fix it", todo, []sending{{bob, "assigned", 0, changed("name", v("Fix"), v("Fix it"), nil, nil)}}},
		{"parent", fmt.Sprintf(`{"parent_id":%q}`, parent.ID), "Fix", todo, []sending{{bob, "assigned", 0, changed("parent", nil, nil, nil, &parent.ID)}}},
		{"priority", `{"priority":"high"}`, "Fix", todo, []sending{{bob, "assigned", 0, changed("priority", v("none"), v("high"), nil, nil)}}},
		{"state", fmt.Sprintf(`{"state_id":%q}`, started.id), "Fix", started,
			[]sending{{bob, "assigned", 0, changed("state", v("Todo"), v("In Progress"), &todo.id, &started.id)}}},
		{"start_date", `{"start_date":"2044-01-02"}`, "Fix", todo, []sending{{bob, "assigned", 0, changed("start_date", none, v("2044-01-02"), nil, nil)}}},
		{"target_date", `{"target_date":"2044-02-03"}`, "Fix", todo,
			[]sending{{bob, "assigned", 0, changed("target_date", none, v("2044-02-03"), nil, nil)}}},
		{"labels", fmt.Sprintf(`{"label_ids":[%q]}`, w.bug), "Fix", todo, []sending{{bob, "assigned", 0, changed("labels", none, v("Bug"), nil, &w.bug)}}},
		{"assignees", fmt.Sprintf(`{"assignee_ids":[%q]}`, walt), "Fix", todo, []sending{
			{bob, "subscribed", 0, changed("assignees", none, &waltName, nil, &walt)}, {bob, "subscribed", 1, changed("assignees", &bobName, none, &bob, nil)},
			{walt, "assigned", 0, changed("assignees", none, &waltName, nil, &walt)}, {walt, "assigned", 1, changed("assignees", &bobName, none, &bob, nil)}}},
		{"positions", `{"name":"Fix it","description_html":"<p>Steps to reproduce</p>","priority":"urgent","sort_order":5}`, "Fix it", todo,
			[]sending{{bob, "assigned", 0, changed("name", v("Fix"), v("Fix it"), nil, nil)},
				{bob, "assigned", 2, changed("priority", v("none"), v("urgent"), nil, nil)}}},
	}
	items := make([]notificationdomain.IssueSnapshot, len(writes))
	for i := range writes {
		items[i] = made("alice", "Fix", todo, bob)
	}
	archived, auto := made("alice", "Fix", done, bob), made("alice", "Fix", done, bob)
	// A fan-out reads the work item's snapshot, subscribers and assignees when it runs: the creations' have run before
	// any work item is written again, and each is written once more, or, archived, by writes that change none of them.
	waitForFanouts(t, w.pool)

	w.send(t, "wanda", http.MethodPost, "/api/v0/issues/"+team.ID.String()+"/subscribe", "", http.StatusNoContent)
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+team.ID.String(), `{"priority":"low"}`, http.StatusOK)
	for i, write := range writes {
		w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+items[i].ID.String(), write.body, http.StatusOK)
	}
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+archived.ID.String()+"/archive", "", http.StatusOK)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+archived.ID.String()+"/unarchive", "", http.StatusOK)
	// nerve's own archiving (M4's autoArchiveOn, with the app's subscribers), 20 days ahead: auto, done 40 days before,
	// is the one work item due then; on the real clock, which the app's start-time run reads, it is not due.
	at := time.Now().Add(20 * 24 * time.Hour).UTC().Truncate(time.Microsecond)
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issues SET updated_at = $2 WHERE id = $1", auto.ID, at.Add(-40*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"archive_in":1}`, http.StatusOK)
	if n, err := autoArchiveOn(w, clocktest.At(at), projectDeletionPublisher(t, w.pool)).Execute(pgtest.Soon(t)); err != nil || n != 1 {
		t.Fatalf("nerve's archiving = %d, %v; want auto alone", n, err)
	}
	waitForFanouts(t, w.pool)

	// madeIt is the change of a work item's creation itself; assigning, of its assignment to the account of the name.
	madeIt := notificationdomain.ChangeSnapshot{Field: "issue", Verb: "created"}
	assigning := func(name *string, account *uuid.UUID) notificationdomain.ChangeSnapshot {
		return notificationdomain.ChangeSnapshot{Field: "assignees", Verb: "created", NewValue: name, NewID: account}
	}
	creation := func(t *testing.T, item notificationdomain.IssueSnapshot) (notificationriver.FanoutArgs, []sentNotification) {
		t.Helper()
		job := fanoutOf(t, w.pool, "issue.created", item.ID)
		return job, w.sentOf(job, &alice, &alice, item, sending{bob, "assigned", -1, madeIt},
			sending{bob, "assigned", assignedAt(t, job, bob), assigning(&bobName, &bob)})
	}
	t.Run("creation and senders", func(t *testing.T) {
		created, priority := fanoutOf(t, w.pool, "issue.created", team.ID), fanoutOf(t, w.pool, "issue.updated", team.ID)
		low := changed("priority", v("none"), v("low"), nil, nil)
		assertSent(t, w.pool, team.ID, []uuid.UUID{created.EventID, priority.EventID}, slices.Concat(
			w.sentOf(created, &bob, &bob, team, sending{walt, "assigned", -1, madeIt},
				sending{walt, "assigned", assignedAt(t, created, bob), assigning(&bobName, &bob)},
				sending{walt, "assigned", assignedAt(t, created, walt), assigning(&waltName, &walt)}),
			w.sentOf(priority, &alice, &bob, team, sending{bob, "created", 0, low}, sending{walt, "assigned", 0, low}, sending{wanda, "subscribed", 0, low})))
	})
	for i, write := range writes {
		t.Run(write.name, func(t *testing.T) {
			job, want := creation(t, items[i])
			update := fanoutOf(t, w.pool, "issue.updated", items[i].ID)
			after := items[i]
			after.Name, after.StateName, after.StateGroup = write.after, write.state.name, write.state.group
			assertSent(t, w.pool, items[i].ID, []uuid.UUID{job.EventID, update.EventID},
				append(want, w.sentOf(update, &alice, &alice, after, write.sends...)...))
		})
	}
	t.Run("archiving and restoring", func(t *testing.T) {
		job, want := creation(t, archived)
		on, off := fanoutOf(t, w.pool, "issue.archived", archived.ID), fanoutOf(t, w.pool, "issue.unarchived", archived.ID)
		assertSent(t, w.pool, archived.ID, []uuid.UUID{job.EventID, on.EventID, off.EventID}, slices.Concat(want,
			w.sentOf(on, &alice, &alice, archived, sending{bob, "assigned", 0, changed("archived_at", nil, v("manual_archive"), nil, nil)}),
			w.sentOf(off, &alice, &alice, archived, sending{bob, "assigned", 0, changed("archived_at", v("archive"), v("restore"), nil, nil)})))
	})
	t.Run("nerve's archiving", func(t *testing.T) {
		job, want := creation(t, auto)
		run := fanoutOf(t, w.pool, "issue.archived", auto.ID)
		if run.ActorID != nil || !run.OccurredAt.Equal(at) {
			t.Fatalf("nerve's archiving of %s by %v at %s; want no one at %s", auto.ID, run.ActorID, run.OccurredAt, at)
		}
		archive := changed("archived_at", nil, v("archive"), nil, nil)
		assertSent(t, w.pool, auto.ID, []uuid.UUID{job.EventID, run.EventID}, append(want,
			w.sentOf(run, nil, &alice, auto, sending{alice, "created", 0, archive}, sending{bob, "assigned", 0, archive})...))
	})
	t.Run("snapshot keys", func(t *testing.T) {
		snapshot := func(s notificationdomain.IssueSnapshot) map[string]any {
			return map[string]any{"id": s.ID.String(), "name": s.Name, "sequence_id": float64(s.SequenceID), "identifier": s.Identifier,
				"state_name": s.StateName, "state_group": s.StateGroup}
		}
		state, parentOf := slices.IndexFunc(writes, func(f fieldWrite) bool { return f.name == "state" }),
			slices.IndexFunc(writes, func(f fieldWrite) bool { return f.name == "parent" })
		moved := items[state]
		moved.StateName, moved.StateGroup = started.name, started.group
		for _, c := range []struct {
			event, receiver uuid.UUID
			seq             int16
			want            map[string]any
		}{
			{fanoutOf(t, w.pool, "issue.created", team.ID).EventID, walt, -1, map[string]any{"issue": snapshot(team),
				"change": map[string]any{"field": "issue", "verb": "created"}}},
			{fanoutOf(t, w.pool, "issue.updated", moved.ID).EventID, bob, 0, map[string]any{"issue": snapshot(moved),
				"change": map[string]any{"field": "state", "verb": "updated", "old_value": "Todo", "new_value": "In Progress", "old_id": todo.id.String(),
					"new_id": started.id.String()}}},
			{fanoutOf(t, w.pool, "issue.updated", items[parentOf].ID).EventID, bob, 0, map[string]any{"issue": snapshot(items[parentOf]),
				"change": map[string]any{"field": "parent", "verb": "updated", "new_id": parent.ID.String()}}},
			{fanoutOf(t, w.pool, "issue.archived", auto.ID).EventID, alice, 0, map[string]any{"issue": snapshot(auto),
				"change": map[string]any{"field": "archived_at", "verb": "updated", "new_value": "archive"}}},
		} {
			if got := sentData(t, w.pool, c.event, c.receiver, c.seq); !reflect.DeepEqual(got, c.want) {
				t.Errorf("the snapshot of %s to %s at %d = %v; want %v", c.event, c.receiver, c.seq, got, c.want)
			}
		}
	})
}

// fanoutCount is how many notification.fanout jobs were ever enqueued.
func fanoutCount(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM river_job WHERE kind = $1", notificationriver.FanoutKind).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// Writes of a work item's through the API that notify no one (M7 P1b spec 2.3, 4) enqueue no fan-out and send
// nothing: of its description alone, its sorting, its modules, the transfer of its cycle's work items to another cycle,
// its cycle, and its deletion. bob, its assignee, is sent its creation and his assignment, and nothing else.
func TestWorkItemWritesThatNotifyNoOne(t *testing.T) {
	w := newIssueWorld(t)
	w.turnCyclesOn(t, w.web)
	w.turnModulesOn(t, w.web)
	now := time.Now()
	day := func(days int) string { return shanghaiDay(t, now, days) }
	sprint := w.newCycle(t, "alice", datesBody("Sprint", day(-10), day(2)))
	next := w.newCycle(t, "alice", datesBody("Next", day(5), day(10)))
	login := w.newModule(t, `{"name":"Login"}`)
	item := w.newWorkItemIn(t, w.web, fmt.Sprintf(`{"name":"Fix","state_id":%q,"assignee_ids":[%q],"cycle_id":%q}`, w.todo, w.ids["bob"], sprint))
	waitForFanouts(t, w.pool)
	sent, rows := sentNotifications(t, w.pool, item)
	if len(rows) != 2 || rows[0].Receiver != w.ids["bob"] || rows[1].Receiver != w.ids["bob"] {
		t.Fatalf("the creation's notifications %+v; want bob's two", rows)
	}
	path := "/api/v0/issues/" + item.String()
	for _, write := range []struct {
		name string
		send func(t *testing.T)
	}{
		{"its description alone", func(t *testing.T) {
			w.send(t, "alice", http.MethodPatch, path, `{"description_html":"<p>Steps to reproduce</p>"}`, http.StatusOK)
		}},
		{"its sorting", func(t *testing.T) { w.send(t, "alice", http.MethodPatch, path, `{"sort_order":12345}`, http.StatusOK) }},
		{"its modules", func(t *testing.T) { w.send(t, "alice", http.MethodPatch, path, modulesBody(login), http.StatusOK) }},
		{"the transfer of its cycle's", func(t *testing.T) {
			w.send(t, "alice", http.MethodPatch, "/api/v0/cycles/"+sprint.String(), `{"start_date":"`+day(-10)+`","end_date":"`+day(-2)+`"}`, http.StatusOK)
			if moved := w.transfer(t, "alice", sprint, next); moved != 1 {
				t.Fatalf("the transfer moved %d; want the work item", moved)
			}
		}},
		{"its cycle", func(t *testing.T) { w.send(t, "alice", http.MethodPatch, path, `{"cycle_id":null}`, http.StatusOK) }},
		{"its deletion", func(t *testing.T) { w.send(t, "alice", http.MethodDelete, path, "", http.StatusNoContent) }},
	} {
		t.Run(write.name, func(t *testing.T) {
			before := fanoutCount(t, w.pool)
			write.send(t)
			if after := fanoutCount(t, w.pool); after != before {
				t.Errorf("%s enqueued %d fan-outs; want none", write.name, after-before)
			}
		})
	}
	waitForFanouts(t, w.pool)
	if again, rows := sentNotifications(t, w.pool, item); !slices.Equal(again, sent) {
		t.Errorf("the work item's notifications %v %+v; want its creation's alone, %v", again, rows, sent)
	}
}
