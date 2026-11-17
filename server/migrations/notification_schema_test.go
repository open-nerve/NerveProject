package migrations_test

import (
	"slices"
	"testing"
)

// The ids the notifications table's tests write besides newIssueDatabase's,
// as SQL literals: the receiver of a notification, the account whose write
// triggered it, and the source event.
const (
	notificationReceiver = "'0199a2b4-0000-7000-8000-0000000000e1'"
	notificationActor    = "'0199a2b4-0000-7000-8000-0000000000e2'"
	notificationEvent    = "'0199a2b4-0000-7000-8000-0000000000e3'"
	notificationRow      = "'0199a2b4-0000-7000-8000-0000000000e4'"
)

// notificationOf is the insertion of an undeleted notification of web's
// work item for receiver, of the change at seq of the event, triggered by
// the actor.
func notificationOf(id, event, seq, receiver string) string {
	return "INSERT INTO notifications (id, workspace_id, project_id, receiver_id, triggered_by_id, issue_id, issue_created_by_id, sender, data, event_id, change_seq) VALUES (" +
		id + ", " + issueAcme + ", " + issueWeb + ", " + receiver + ", " + notificationActor + ", " + webIssue + ", " + issueBob +
		`, 'subscribed', '{"issue": {"id": "x"}, "change": {"field": "state"}}', ` + event + ", " + seq + ")"
}

// The notifications table (M7 design 4.2, P1b spec 2.2): its columns and
// indexes; the four senders and no other; the snapshot an object of exactly
// an issue object and a change object; the change's position the creation's
// -1 or a list position; the work item and its creator no foreign keys; the
// triggering account's and the audit accounts' SET NULL keeping the
// notification, the receiver's and the project's purge taking it.
func TestNotificationsSchema(t *testing.T) {
	pool := newIssueDatabase(t)
	if got, want := columnsOf(t, pool, "notifications"), []string{
		"id uuid NO -", "workspace_id uuid NO -", "project_id uuid NO -", "receiver_id uuid NO -", "triggered_by_id uuid YES -",
		"issue_id uuid NO -", "issue_created_by_id uuid YES -", "sender character varying NO -", "data jsonb NO -",
		"event_id uuid NO -", "change_seq smallint NO -", "read_at timestamp with time zone YES -",
		"archived_at timestamp with time zone YES -", "snoozed_till timestamp with time zone YES -",
		"created_by_id uuid YES -", "updated_by_id uuid YES -", "created_at timestamp with time zone NO now()", "updated_at timestamp with time zone NO now()",
		"deleted_at timestamp with time zone YES -",
	}; !slices.Equal(got, want) {
		t.Fatalf("notifications columns\n%q\nwant\n%q", got, want)
	}
	if got, want := indexesOf(t, pool, "notifications"), []string{
		"CREATE INDEX notifications_deleted_at_idx ON public.notifications USING btree (deleted_at) WHERE (deleted_at IS NOT NULL)",
		"CREATE UNIQUE INDEX notifications_event_id_change_seq_receiver_id_key ON public.notifications USING btree (event_id, change_seq, receiver_id) WHERE (deleted_at IS NULL)",
		"CREATE UNIQUE INDEX notifications_pkey ON public.notifications USING btree (id)",
		"CREATE INDEX notifications_project_id_idx ON public.notifications USING btree (project_id)",
		"CREATE INDEX notifications_receiver_id_idx ON public.notifications USING btree (receiver_id)",
		"CREATE INDEX notifications_receiver_list_idx ON public.notifications USING btree (receiver_id, workspace_id, created_at DESC, id DESC) WHERE (deleted_at IS NULL)",
		"CREATE INDEX notifications_receiver_unread_idx ON public.notifications USING btree (receiver_id, workspace_id) WHERE ((deleted_at IS NULL) AND (read_at IS NULL) AND (archived_at IS NULL))",
		"CREATE INDEX notifications_workspace_id_idx ON public.notifications USING btree (workspace_id)",
		"CREATE INDEX notifications_workspace_id_issue_id_idx ON public.notifications USING btree (workspace_id, issue_id) WHERE (deleted_at IS NULL)",
	}; !slices.Equal(got, want) {
		t.Errorf("notifications indexes\n%q\nwant\n%q", got, want)
	}

	accepted(t, pool,
		"INSERT INTO users (id, email, password, display_name) VALUES ("+notificationReceiver+", 'carol@corp.com', 'x', 'carol'), ("+
			notificationActor+", 'dave@corp.com', 'x', 'dave')",
		notificationOf(notificationRow, notificationEvent, "0", notificationReceiver))
	update := func(set string) string { return "UPDATE notifications SET " + set + " WHERE id=" + notificationRow }
	// Accepted: every sender; a snapshot of the two objects, whatever they
	// hold; the creation's -1 and any list position; a read, archived and
	// snoozed state.
	for _, set := range []string{"sender='created'", "sender='assigned'", "sender='mentioned'", "sender='subscribed'",
		`data='{"issue": {}, "change": {}}'`,
		`data='{"issue": {"id": "x", "sequence_id": 12, "identifier": "WEB"}, "change": {"field": "comment", "verb": "created", "excerpt": "hi"}}'`,
		"change_seq=-1", "change_seq=32767", "change_seq=0",
		"read_at=now(), archived_at=now(), snoozed_till=now() + interval '1 day'",
		"triggered_by_id=NULL, issue_created_by_id=NULL"} {
		accepted(t, pool, update(set))
	}
	for _, tt := range []struct{ set, constraint string }{
		{"sender='in_app:created'", "notifications_sender_check"},
		{"sender='Created'", "notifications_sender_check"},
		{"sender='commented'", "notifications_sender_check"},
		{"sender=''", "notifications_sender_check"},
		{"data='{}'", "notifications_data_check"},
		{`data='{"issue": {}}'`, "notifications_data_check"},
		{`data='{"change": {}}'`, "notifications_data_check"},
		{`data='{"issue": {}, "change": {}, "issue_activity": {}}'`, "notifications_data_check"},
		{`data='{"issue": [], "change": {}}'`, "notifications_data_check"},
		{`data='{"issue": {}, "change": "state"}'`, "notifications_data_check"},
		{`data='{"issue": {}, "change": null}'`, "notifications_data_check"},
		{`data='[{"issue": {}, "change": {}}]'`, "notifications_data_check"},
		{`data='"issue"'`, "notifications_data_check"},
		{"data='1'", "notifications_data_check"},
		{"data='null'", "notifications_data_check"},
		{"change_seq=-2", "notifications_change_seq_check"},
	} {
		refusedBy(t, pool, update(tt.set), "23514", tt.constraint)
	}
	for _, column := range []string{"workspace_id", "project_id", "receiver_id", "issue_id", "sender", "data", "event_id", "change_seq",
		"created_at", "updated_at"} {
		nullRefused(t, pool, update(column+"=NULL"), column)
	}
	// The work item and its creator are no foreign keys: a notification
	// outlives its work item's row, which the 60-day purge removes later.
	accepted(t, pool, update("issue_id=gen_random_uuid(), issue_created_by_id=gen_random_uuid()"))

	// Purging the triggering account, which is also the row's last writer,
	// keeps the notification without them; purging the receiver's account
	// takes the receiver's notifications.
	accepted(t, pool, update("triggered_by_id="+notificationActor+", created_by_id="+notificationActor+", updated_by_id="+notificationActor),
		"DELETE FROM users WHERE id="+notificationActor)
	if got := names(t, pool, "SELECT concat_ws(' ', coalesce(triggered_by_id::text, 'nerve'), coalesce(created_by_id::text, 'nerve'), coalesce(updated_by_id::text, 'nerve')) FROM notifications"); !slices.Equal(got, []string{"nerve nerve nerve"}) {
		t.Fatalf("triggered_by, created_by, updated_by after the actor's purge = %q, want [nerve nerve nerve]", got)
	}
	accepted(t, pool, "DELETE FROM users WHERE id="+notificationReceiver)
	if got := names(t, pool, "SELECT id::text FROM notifications"); len(got) != 0 {
		t.Fatalf("notifications after the receiver's purge = %q, want none", got)
	}
	// Purging the project takes its notifications; the other project's stay.
	accepted(t, pool, "INSERT INTO notifications (id, workspace_id, project_id, receiver_id, issue_id, sender, data, event_id, change_seq) VALUES "+
		"(gen_random_uuid(), "+issueAcme+", "+issueWeb+", "+issueBob+", "+webIssue+`, 'created', '{"issue": {}, "change": {}}', gen_random_uuid(), -1), `+
		"(gen_random_uuid(), "+issueAcme+", "+issueOps+", "+issueBob+", "+opsIssue+`, 'created', '{"issue": {}, "change": {}}', gen_random_uuid(), -1)`,
		"DELETE FROM projects WHERE id="+issueWeb)
	if got := names(t, pool, "SELECT issue_id::text FROM notifications"); !slices.Equal(got, []string{opsIssue[1 : len(opsIssue)-1]}) {
		t.Fatalf("notifications after web's purge = %q, want ops' only", got)
	}
}

// The notification key holds among undeleted rows only (P1b spec 2.2, R4):
// a second undeleted row of the same change of the same event for the same
// receiver is refused, so a retried or redelivered job writes nothing more;
// a row short of a column of the key is another notification; soft-deleting
// the first frees the key.
func TestNotificationKeyHoldsAmongUndeletedRowsOnly(t *testing.T) {
	pool := newIssueDatabase(t)
	accepted(t, pool,
		"INSERT INTO users (id, email, password, display_name) VALUES ("+notificationReceiver+", 'carol@corp.com', 'x', 'carol'), ("+
			notificationActor+", 'dave@corp.com', 'x', 'dave')",
		notificationOf(notificationRow, notificationEvent, "0", notificationReceiver),
		// Another receiver of the change, another change of the event for the
		// receiver, the same position of another event.
		notificationOf("gen_random_uuid()", notificationEvent, "0", notificationActor),
		notificationOf("gen_random_uuid()", notificationEvent, "-1", notificationReceiver),
		notificationOf("gen_random_uuid()", "gen_random_uuid()", "0", notificationReceiver))
	again := notificationOf("gen_random_uuid()", notificationEvent, "0", notificationReceiver)
	refusedBy(t, pool, again, "23505", "notifications_event_id_change_seq_receiver_id_key")
	accepted(t, pool, "UPDATE notifications SET read_at=now(), archived_at=now() WHERE id="+notificationRow)
	refusedBy(t, pool, again, "23505", "notifications_event_id_change_seq_receiver_id_key")
	accepted(t, pool, "UPDATE notifications SET deleted_at=now() WHERE id="+notificationRow, again)
}

// notificationNames are the notification module's constraint and index
// names with what each is (TestConstraintAndIndexNames).
var notificationNames = []string{
	"notifications_change_seq_check c",
	"notifications_created_by_id_fkey f n",
	"notifications_data_check c",
	"notifications_deleted_at_idx iw",
	"notifications_event_id_change_seq_receiver_id_key iuw",
	"notifications_pkey iu",
	"notifications_pkey p",
	"notifications_project_id_fkey f c",
	"notifications_project_id_idx i",
	"notifications_receiver_id_fkey f c",
	"notifications_receiver_id_idx i",
	"notifications_receiver_list_idx iw",
	"notifications_receiver_unread_idx iw",
	"notifications_sender_check c",
	"notifications_triggered_by_id_fkey f n",
	"notifications_updated_by_id_fkey f n",
	"notifications_workspace_id_fkey f c",
	"notifications_workspace_id_idx i",
	"notifications_workspace_id_issue_id_idx iw",
}
