package bootstrap

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// laterApp is the address of an app over cfg's database whose business
// clock is ahead of cfg's by ahead: built as newApp builds it, signing
// and checking tokens with cfg's key, so that the tokens of cfg's app are
// its too, and serving its routes without running its workers: the jobs
// its writes enqueue are the running app's to run, on the same database.
// It stops when the test ends.
func laterApp(t *testing.T, cfg config.Config, ahead time.Duration) string {
	t.Helper()
	cfg.Clock.Offset += ahead
	server := httptest.NewServer(buildApp(t, cfg, migrations.FS()).router)
	t.Cleanup(server.Close)
	return server.URL
}

// markedBy is how many of the account name's notifications in acme his
// mark-all-read with body marks, as its answer, {marked} and no other
// key, says.
func (w issueWorld) markedBy(t *testing.T, name, body string) int {
	t.Helper()
	answer := w.send(t, name, http.MethodPost, inboxPath+"/mark-all-read", body, http.StatusOK)
	var marked int
	decodeAnswer(t, string(listKeys(t, []byte(answer), "marked")["marked"]), &marked)
	return marked
}

// unreadRows are the ids of the account name's undeleted unread
// notifications in acme as stored, whether he reads them now or not,
// sorted.
func (w issueWorld) unreadRows(t *testing.T, name string) []string {
	t.Helper()
	var ids []string
	for _, id := range queryIDs(t, w.pool, `SELECT id FROM notifications WHERE receiver_id = $1 AND workspace_id = $2 AND read_at IS NULL
		AND deleted_at IS NULL`, w.ids[name], w.acme) {
		ids = append(ids, id.String())
	}
	return slices.Sorted(slices.Values(ids))
}

// sortedIDs are ids, sorted.
func sortedIDs(ids ...string) []string { return slices.Sorted(slices.Values(ids)) }

// A receiver's unread counts and his mark-all-read through the app (M7
// P1c spec 2.2, 2.7, R3, R5, R6; H6, H7). bob is sent, by alice, Fix's
// creation and his assignment, both in Web; the same of Far, in Ops; her
// change of the name and the priority of Move, which he subscribes to,
// two notifications; and of Talk, which he made, her comment and her
// comment mentioning him, which he is sent as a mention alone. He archives
// his assignment to Fix, snoozes Move's renaming for five minutes and
// reads its new priority.
//
// The unread counts leave out what is archived, snoozed, read, or of a
// project he no longer reads, and count the mention in its own tab. On the
// business clock ten minutes on, through an app whose clock is that, the
// snooze is over: the renaming is listed again and counted unread, and no
// longer listed as snoozed.
//
// mark-all-read marks his unread notifications of the tab and the
// filters it is given, and only those: the mentions; the comment, of the
// type created; the archived one; the snoozed one; the rest of the all
// tab, leaving the mention, unread again, as it is; and then none. A
// comment of alice's made through the later app is of an event after the
// moment of bob's mark-all-read, which leaves it; through the later app,
// it is marked. A notification of an event before that moment that is
// written while his mark-all-read waits to mark, as a late fan-out's
// would be, with an id after every other, is marked: the known property
// the spec records (re-84 I2). Marking 1005 of walt's notifications takes
// two batches, and touches none of bob's.
func TestMarkingAllReadAndTheUnreadCountsThroughTheApp(t *testing.T) {
	var cfg config.Config
	key := writeFile(t, testKeyPEM)
	w := newIssueWorldWith(t, func(c *config.Config) { c.Auth.JWT.PrivateKeyFile = key; cfg = *c })
	later := w
	later.base = laterApp(t, cfg, 10*time.Minute)
	bob := w.ids["bob"]
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	w.makeMember(t, "bob", ops, shared.RoleMember)
	toBob := fmt.Sprintf(`"assignee_ids":%s}`, idArray(bob))
	fix, far := w.newWorkItemIn(t, w.web, `{"name":"Fix",`+toBob), w.newWorkItemIn(t, ops, `{"name":"Far",`+toBob)
	move, talk := w.newWorkItemIn(t, w.web, `{"name":"Move"}`), w.newWorkItemBy(t, "bob", w.web, `{"name":"Talk"}`)
	// A fan-out reads the work item's subscribers when it runs: the creations' have run before bob subscribes.
	waitForFanouts(t, w.pool)
	w.send(t, "bob", http.MethodPost, "/api/v0/issues/"+move.String()+"/subscribe", "", http.StatusNoContent)
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+move.String(), `{"name":"Moved","priority":"high"}`, http.StatusOK)
	for _, html := range []string{"<p>Seen</p>", mentionHTML(bob)} {
		w.send(t, "alice", http.MethodPost, commentsOf(talk), fmt.Sprintf(`{"comment_html":%q}`, html), http.StatusCreated)
	}
	waitForFanouts(t, w.pool)
	note := func(item uuid.UUID, change string) string { return w.noteIDOf(t, "bob", item, change) }
	issued, assigned, renamed, prioritized := note(fix, "issue created"), note(fix, "assignees created"), note(move, "name updated"), note(move, "priority updated")
	commented, mention, farOnes := note(talk, "comment created"), note(talk, "comment mentioned"), []string{note(far, "issue created"), note(far, "assignees created")}
	w.changeNote(t, "bob", assigned, `{"archived":true}`)
	w.changeNote(t, "bob", renamed, snoozing(time.Now().Add(5*time.Minute)))
	w.changeNote(t, "bob", prioritized, `{"read":true}`)
	counted := func(t *testing.T, at issueWorld, when string, all, mentions int) {
		t.Helper()
		if gotAll, gotMentions := at.unreadOf(t, "bob"); gotAll != all || gotMentions != mentions {
			t.Errorf("bob's unread counts %s = %d, %d; want %d, %d", when, gotAll, gotMentions, all, mentions)
		}
	}

	t.Run("unread counts", func(t *testing.T) {
		counted(t, w, "with Far's", 4, 1)
		w.send(t, "bob", http.MethodPost, "/api/v0/projects/"+ops.String()+"/leave", "", http.StatusNoContent)
		counted(t, w, "once he left Ops", 2, 1)
	})
	t.Run("a snooze over by the business clock", func(t *testing.T) {
		for _, c := range []struct {
			at              issueWorld
			when            string
			listed, snoozed []string
			unread          int
		}{
			{w, "now", sortedIDs(issued, prioritized, commented), []string{renamed}, 2},
			{later, "ten minutes on", sortedIDs(issued, prioritized, commented, renamed), nil, 3},
		} {
			if listed, snoozed := c.at.inboxIDs(t, "bob", ""), c.at.inboxIDs(t, "bob", "?snoozed=true"); !slices.Equal(listed, c.listed) ||
				!slices.Equal(snoozed, c.snoozed) {
				t.Errorf("bob's list %s = %v, snoozed %v; want %v, snoozed %v", c.when, listed, snoozed, c.listed, c.snoozed)
			}
			counted(t, c.at, c.when, c.unread, 1)
		}
	})
	// marks checks that the account name's mark-all-read with body, through
	// the app at, marks his unread notifications want and no other.
	marks := func(t *testing.T, at issueWorld, name, body string, want ...string) {
		t.Helper()
		before := w.unreadRows(t, name)
		if slices.ContainsFunc(want, func(id string) bool { return !slices.Contains(before, id) }) {
			t.Fatalf("%s's notifications %v to mark are not all unread, %v", name, want, before)
		}
		left := slices.DeleteFunc(slices.Clone(before), func(id string) bool { return slices.Contains(want, id) })
		if marked := at.markedBy(t, name, body); marked != len(want) || !slices.Equal(w.unreadRows(t, name), left) {
			t.Errorf("%s's mark-all-read %s marked %d, leaving %v unread; want %d, leaving %v", name, body, marked, w.unreadRows(t, name), len(want), left)
		}
	}
	t.Run("of a tab and filters", func(t *testing.T) {
		if got, want := w.unreadRows(t, "bob"), sortedIDs(append([]string{issued, assigned, renamed, commented, mention}, farOnes...)...); !slices.Equal(got, want) {
			t.Fatalf("bob's unread notifications %v; want %v", got, want)
		}
		marks(t, w, "bob", `{"tab":"mentions"}`, mention)
		marks(t, w, "bob", `{"type":["created"]}`, commented)
		marks(t, w, "bob", `{"archived":true}`, assigned)
		marks(t, w, "bob", `{"snoozed":true}`, renamed)
		w.changeNote(t, "bob", mention, `{"read":false}`)
		marks(t, w, "bob", `{"tab":"all"}`, issued)
		marks(t, w, "bob", `{"tab":"all"}`)
		marks(t, w, "bob", `{}`)
		marks(t, w, "bob", `{"tab":"mentions"}`, mention)
		marks(t, w, "bob", `{"tab":"mentions"}`)
	})
	t.Run("an event after its moment", func(t *testing.T) {
		before := w.unreadRows(t, "bob")
		later.send(t, "alice", http.MethodPost, commentsOf(talk), `{"comment_html":"<p>Later</p>"}`, http.StatusCreated)
		waitForFanouts(t, w.pool)
		made := slices.DeleteFunc(w.unreadRows(t, "bob"), func(id string) bool { return slices.Contains(before, id) })
		var ahead bool
		if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT bool_and(created_at > now()) FROM notifications WHERE id = ANY($1::uuid[])`, made).Scan(&ahead); err != nil ||
			len(made) != 1 || !ahead {
			t.Fatalf("the later comment sent bob %v, after now %t, %v; want one, after now", made, ahead, err)
		}
		marks(t, w, "bob", `{}`)
		marks(t, later, "bob", `{}`, made...)
	})
	t.Run("a late row of an earlier event", func(t *testing.T) {
		const late = "ffffffff-ffff-7fff-bfff-ffffffffffff"
		held := lockedTables(t, w.pool, "notifications", "EXCLUSIVE")
		req := newRequest(t, http.MethodPost, w.base+inboxPath+"/mark-all-read", w.tokens["bob"], []byte(`{}`))
		w.contract.CheckRequest(t, req)
		answered := sendInBackground(req)
		pgtest.WaitForStatementLockWaitBehind(t, w.pool, held.Conn().PgConn(), 5*time.Second, "MarkNotificationsRead")
		if _, err := held.Exec(pgtest.Soon(t), `INSERT INTO notifications (id, workspace_id, project_id, receiver_id, triggered_by_id, issue_id,
				issue_created_by_id, sender, data, event_id, change_seq, created_at, updated_at)
			SELECT $1, workspace_id, project_id, receiver_id, triggered_by_id, issue_id, issue_created_by_id, sender, data, gen_random_uuid(), change_seq,
				created_at - interval '1 minute', created_at - interval '1 minute'
			FROM notifications WHERE id = $2`, late, issued); err != nil {
			t.Fatal(err)
		}
		if err := held.Commit(pgtest.Soon(t)); err != nil {
			t.Fatal(err)
		}
		a := receiveWithin(t, answered, 5*time.Second, "answer to bob's mark-all-read")
		if a.err != nil {
			t.Fatal(a.err)
		}
		w.contract.CheckResponse(t, req, a.res)
		var marked int
		if a.res.StatusCode == http.StatusOK {
			decodeAnswer(t, string(listKeys(t, a.body, "marked")["marked"]), &marked)
		}
		if a.res.StatusCode != http.StatusOK || marked != 1 || slices.Contains(w.unreadRows(t, "bob"), late) {
			t.Errorf("bob's mark-all-read with a row of an earlier event written as it waited = %d %s, %v unread; want it marked, marked 1",
				a.res.StatusCode, a.body, w.unreadRows(t, "bob"))
		}
	})
	t.Run("across batches", func(t *testing.T) {
		start := time.Now()
		if _, err := w.pool.Exec(pgtest.Soon(t), `INSERT INTO notifications (id, workspace_id, project_id, receiver_id, triggered_by_id, issue_id,
				issue_created_by_id, sender, data, event_id, change_seq, created_at, updated_at)
			SELECT gen_random_uuid(), n.workspace_id, n.project_id, CASE WHEN g <= 1005 THEN $2::uuid ELSE $3::uuid END, n.triggered_by_id, n.issue_id,
				n.issue_created_by_id, n.sender, n.data, gen_random_uuid(), n.change_seq, n.created_at, n.created_at
			FROM notifications n, generate_series(1, 1010) AS g WHERE n.id = $1`, issued, w.ids["walt"], bob); err != nil {
			t.Fatal(err)
		}
		bobs := w.unreadRows(t, "bob")
		counted(t, w, "with five more", 5, 0)
		if all, mentions := w.unreadOf(t, "walt"); all != 1005 || mentions != 0 {
			t.Fatalf("walt's unread counts = %d, %d; want 1005, 0", all, mentions)
		}
		if marked := w.markedBy(t, "walt", `{}`); marked != 1005 {
			t.Errorf("walt's mark-all-read marked %d; want all 1005, in two batches", marked)
		}
		if all, mentions := w.unreadOf(t, "walt"); all != 0 || mentions != 0 || len(w.unreadRows(t, "walt")) != 0 {
			t.Errorf("walt's unread counts after his mark-all-read = %d, %d, %d rows unread; want none", all, mentions, len(w.unreadRows(t, "walt")))
		}
		if got := w.unreadRows(t, "bob"); len(bobs) != 7 || !slices.Equal(got, bobs) {
			t.Errorf("bob's unread notifications after walt's mark-all-read %v; want his seven, %v", got, bobs)
		}
		counted(t, w, "after walt's mark-all-read", 5, 0)
		t.Logf("seeding 1010 notifications and marking 1005 of them read took %v", time.Since(start))
	})
}
