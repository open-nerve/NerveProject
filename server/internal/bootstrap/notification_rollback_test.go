package bootstrap

import (
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	notificationriver "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/river"
	notificationdomain "github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// witnessFanouts counts each notification.fanout job inserted into pool's database from then on, in sequences, which
// no rollback takes back: in_write when the transaction inserting it is the one that wrote a work item of its project
// (an issues row's xmin is the job row's), elsewhere when not. A write of the app's is one transaction with no
// savepoint (postgres.TxManager.WithinTx), and River inserts in the transaction it is handed (Client.InsertTx): a
// fan-out its subscriber enqueues counts in_write, and one inserted in a transaction of its own elsewhere. It declares
// the faults' function too, fanout_rollback.reject, which raises.
func witnessFanouts(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, sql := range []string{
		`CREATE SCHEMA fanout_rollback`,
		`CREATE SEQUENCE fanout_rollback.in_write`,
		`CREATE SEQUENCE fanout_rollback.elsewhere`,
		`CREATE FUNCTION fanout_rollback.witness() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
			IF EXISTS (SELECT 1 FROM river_job j JOIN issues i ON i.xmin = j.xmin
				WHERE j.id = NEW.id AND i.project_id = (NEW.args->>'project_id')::uuid) THEN
				PERFORM nextval('fanout_rollback.in_write');
			ELSE
				PERFORM nextval('fanout_rollback.elsewhere');
			END IF;
			RETURN NULL;
		END $$`,
		`CREATE FUNCTION fanout_rollback.reject() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'notification rollback fault'; END $$`,
		fmt.Sprintf(`CREATE TRIGGER fanout_a_witness AFTER INSERT ON river_job FOR EACH ROW WHEN (NEW.kind = '%s')
			EXECUTE FUNCTION fanout_rollback.witness()`, notificationriver.FanoutKind),
	} {
		if _, err := pool.Exec(pgtest.Soon(t), sql); err != nil {
			t.Fatal(err)
		}
	}
}

// witnessed answers how many fan-outs witnessFanouts has counted inserted in their writes' transactions, and
// elsewhere.
func witnessed(t *testing.T, pool *pgxpool.Pool) (inWrite, elsewhere int64) {
	t.Helper()
	if err := pool.QueryRow(pgtest.Soon(t), `SELECT (SELECT CASE WHEN is_called THEN last_value ELSE 0 END FROM fanout_rollback.in_write),
		(SELECT CASE WHEN is_called THEN last_value ELSE 0 END FROM fanout_rollback.elsewhere)`).Scan(&inWrite, &elsewhere); err != nil {
		t.Fatal(err)
	}
	return inWrite, elsewhere
}

// rolledBackTables are the tables the writes of TestARolledBackWriteLeavesNoFanoutAndNoNotification write, the
// project's counter of work items among them, and the notifications.
var rolledBackTables = []string{"projects", "issues", "issue_assignees", "issue_subscribers", "issue_activities", "issue_comments", "notifications"}

// writtenState is every row of rolledBackTables, and every notification.fanout job, by table.
func writtenState(t *testing.T, pool *pgxpool.Pool) map[string]string {
	t.Helper()
	state := map[string]string{}
	for _, table := range rolledBackTables {
		var rows string
		if err := pool.QueryRow(pgtest.Soon(t), fmt.Sprintf(`SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY r.id), '[]')::text FROM %s r`, table)).
			Scan(&rows); err != nil {
			t.Fatal(err)
		}
		state[table] = rows
	}
	var jobs string
	if err := pool.QueryRow(pgtest.Soon(t), `SELECT coalesce(jsonb_agg(jsonb_build_array(id, state, args) ORDER BY id), '[]')::text
		FROM river_job WHERE kind = $1`, notificationriver.FanoutKind).Scan(&jobs); err != nil {
		t.Fatal(err)
	}
	state[notificationriver.FanoutKind] = jobs
	return state
}

// A write that would notify someone and fails leaves no notification.fanout job, no notification and none of its own
// rows (M7 P1b spec 2.9; M4 → M7 M7-02, M5 → M7 M5-M7-02). Through the app, alice's creation of a work item assigned
// to bob, her change of the priority of hers assigned to bob, and her comment on it, each of which sends bob, fail at
// two faults. One fails in the write's transaction once its fan-out is in: the fan-out's insert is the write's last
// (its subscriber is the dispatcher's last, and the write only reads once it has published), so a trigger raises after
// the job's row is inserted. The other fails at the COMMIT, the write done: a deferred trigger on the work item's row
// raises. Each write answers 500; its fan-out was inserted, once, in its own transaction (witnessFanouts), and none is
// left, nor a notification, and every row it writes is as it was. The same writes, the faults gone, then send bob what
// M7 P1b spec 2.3–2.5 say, read back once their fan-outs have run, and the creation takes the number the failed ones
// did not: what the failures held back were writes that notify.
//
// A fan-out enqueued in a transaction of its own (the jobs inserter's Insert in place of InsertTx) fails at both
// faults: counted elsewhere, and at the commit left behind.
func TestARolledBackWriteLeavesNoFanoutAndNoNotification(t *testing.T) {
	w := newIssueWorld(t)
	alice, bob := w.ids["alice"], w.ids["bob"]
	var bobName string
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT display_name FROM users WHERE id = $1", bob).Scan(&bobName); err != nil {
		t.Fatal(err)
	}
	todo := webState{w.todo, "Todo", "unstarted"}
	item := w.itemMade(t, "alice", "Fix", todo, bob)
	waitForFanouts(t, w.pool)
	witnessFanouts(t, w.pool)
	writes := []struct {
		name, method, path, body string
	}{
		// itemMade's body, which the creation sends again below.
		{"creation", http.MethodPost, "/api/v0/projects/" + w.web.String() + "/issues",
			fmt.Sprintf(`{"name":"Retry","state_id":%q,"assignee_ids":%s}`, w.todo, idArray(bob))},
		{"update", http.MethodPatch, "/api/v0/issues/" + item.ID.String(), `{"priority":"high"}`},
		{"comment", http.MethodPost, commentsOf(item.ID), `{"comment_html":"<p>Ship it</p>"}`},
	}
	for _, fault := range []struct{ name, install, remove string }{
		{"after its fan-out's insert", fmt.Sprintf(`CREATE TRIGGER fanout_b_reject AFTER INSERT ON river_job FOR EACH ROW WHEN (NEW.kind = '%s')
			EXECUTE FUNCTION fanout_rollback.reject()`, notificationriver.FanoutKind), `DROP TRIGGER fanout_b_reject ON river_job`},
		{"at its commit", `CREATE CONSTRAINT TRIGGER fanout_reject_at_commit AFTER INSERT OR UPDATE ON issues DEFERRABLE INITIALLY DEFERRED
			FOR EACH ROW EXECUTE FUNCTION fanout_rollback.reject()`, `DROP TRIGGER fanout_reject_at_commit ON issues`},
	} {
		for _, write := range writes {
			t.Run(fault.name+"/"+write.name, func(t *testing.T) {
				before := writtenState(t, w.pool)
				inWrite, elsewhere := witnessed(t, w.pool)
				if _, err := w.pool.Exec(pgtest.Soon(t), fault.install); err != nil {
					t.Fatal(err)
				}
				defer func() {
					if _, err := w.pool.Exec(pgtest.Soon(t), fault.remove); err != nil {
						t.Error(err)
					}
				}()
				if answer := w.send(t, "alice", write.method, write.path, write.body, http.StatusInternalServerError); strings.Contains(answer, "rollback fault") {
					t.Errorf("the fault's detail escaped: %s", answer)
				}
				if in, out := witnessed(t, w.pool); in != inWrite+1 || out != elsewhere {
					t.Errorf("the write's fan-outs: %d inserted in its transaction, %d elsewhere; want one, in it", in-inWrite, out-elsewhere)
				}
				after := writtenState(t, w.pool)
				for _, table := range slices.Sorted(maps.Keys(before)) {
					if after[table] != before[table] {
						t.Errorf("the failed write left %s\n%s\nwas\n%s", table, after[table], before[table])
					}
				}
			})
		}
	}

	retried := w.itemMade(t, "alice", "Retry", todo, bob)
	w.send(t, "alice", writes[1].method, writes[1].path, writes[1].body, http.StatusOK)
	var said answeredComment
	decodeAnswer(t, w.send(t, "alice", writes[2].method, writes[2].path, writes[2].body, http.StatusCreated), &said)
	waitForFanouts(t, w.pool)
	if retried.SequenceID != item.SequenceID+1 {
		t.Errorf("the creation took number %d after %d's; the failed ones took one", retried.SequenceID, item.SequenceID)
	}
	if in, out := witnessed(t, w.pool); in != 9 || out != 0 {
		t.Errorf("fan-outs inserted in their writes' transactions %d, elsewhere %d; want the six failed writes' and the three's after", in, out)
	}
	v := eventValue[string]
	made := func(job notificationriver.FanoutArgs, issue notificationdomain.IssueSnapshot) []sentNotification {
		return w.sentOf(job, &alice, &alice, issue, sending{bob, "assigned", -1, madeIt},
			sending{bob, "assigned", assignedAt(t, job, bob), assigning(&bobName, &bob)})
	}
	again := fanoutOf(t, w.pool, "issue.created", retried.ID)
	assertSent(t, w.pool, retried.ID, []uuid.UUID{again.EventID}, made(again, retried))
	first, update := fanoutOf(t, w.pool, "issue.created", item.ID), fanoutOf(t, w.pool, "issue.updated", item.ID)
	comment := fanoutOf(t, w.pool, "issue.comment_created", said.ID)
	assertSent(t, w.pool, item.ID, []uuid.UUID{first.EventID, update.EventID, comment.EventID}, slices.Concat(made(first, item),
		w.sentOf(update, &alice, &alice, item, sending{bob, "assigned", 0, changed("priority", v("none"), v("high"), nil, nil)}),
		w.sentOf(comment, &alice, &alice, item, sending{bob, "assigned", 0, notificationdomain.ChangeSnapshot{Field: "comment", Verb: "created",
			CommentID: &said.ID, Excerpt: v("Ship it")}})))
}
