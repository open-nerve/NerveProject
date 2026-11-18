package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	notificationriver "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// holdNotificationDeletions has every notification.delete_rows job inserted
// into pool's database wait a day, as holdFanouts holds the fan-outs: a
// deleted project's notifications then stay undeleted (M7 P1b spec 2.8),
// the window in which the reads must already leave them out (P1c spec 2.5).
// The hold is put in place before the app starts, as holdFanouts' is.
func holdNotificationDeletions(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, sql := range []string{
		`CREATE SCHEMA deletion_hold`,
		`CREATE FUNCTION deletion_hold.later() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN NEW.scheduled_at = clock_timestamp() + interval '1 day'; RETURN NEW; END $$`,
		fmt.Sprintf(`CREATE TRIGGER hold_notification_deletions BEFORE INSERT ON river_job FOR EACH ROW WHEN (NEW.kind = '%s') EXECUTE FUNCTION deletion_hold.later()`,
			notificationriver.DeleteRowsKind),
	} {
		if _, err := pool.Exec(pgtest.Soon(t), sql); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := pool.Exec(ctx, `DROP TRIGGER hold_notification_deletions ON river_job; DROP SCHEMA deletion_hold CASCADE`); err != nil {
			t.Error(err)
		}
	})
}

// inboxIDs are the ids of the account name's notifications in acme that
// query lists, sorted (inbox).
func (w issueWorld) inboxIDs(t *testing.T, name, query string) []string {
	t.Helper()
	return slices.Sorted(slices.Values(noteIDs(w.inbox(t, name, query).rows)))
}

// unreadOf is the account name's unread counts in acme, the all tab's and
// the mentions tab's.
func (w issueWorld) unreadOf(t *testing.T, name string) (all, mentions int) {
	t.Helper()
	var counts struct {
		All      int `json:"all"`
		Mentions int `json:"mentions"`
	}
	decodeAnswer(t, w.send(t, name, http.MethodGet, inboxPath+"/unread-count", "", http.StatusOK), &counts)
	return counts.All, counts.Mentions
}

// marksAllRead is how many of the account name's notifications in acme
// mark-all-read marks, of the all tab by default.
func (w issueWorld) marksAllRead(t *testing.T, name string) int {
	t.Helper()
	var answer struct {
		Marked int `json:"marked"`
	}
	decodeAnswer(t, w.send(t, name, http.MethodPost, inboxPath+"/mark-all-read", `{}`, http.StatusOK), &answer)
	return answer.Marked
}

// refusesChange checks that the account name's change of each of the
// notifications ids, its read, answers 404 notification.not_found.
func (w issueWorld) refusesChange(t *testing.T, name string, ids ...string) {
	t.Helper()
	for _, id := range ids {
		status, body := call(t, w.contract, http.MethodPatch, w.base+"/api/v0/notifications/"+id, w.tokens[name], `{"read":true}`)
		if got := refusalOf(t, []byte(body), false); status != http.StatusNotFound || got != "notification.not_found" {
			t.Errorf("%s's change of notification %s = %d %s; want 404 notification.not_found", name, id, status, got)
		}
	}
}

// shows checks the account name's view of his notifications in acme
// through the API: its list is listed, in any order; its unread counts
// are unread, and none of the mentions tab; a change of each of hidden is
// refused (refusesChange); and then mark-all-read marks unread of them.
func (w issueWorld) shows(t *testing.T, name string, listed []string, unread int, hidden ...string) {
	t.Helper()
	if got, want := w.inboxIDs(t, name, ""), slices.Sorted(slices.Values(listed)); !slices.Equal(got, want) {
		t.Errorf("%s's list = %v; want %v", name, got, want)
	}
	if all, mentions := w.unreadOf(t, name); all != unread || mentions != 0 {
		t.Errorf("%s's unread counts = %d, %d; want %d, 0", name, all, mentions, unread)
	}
	w.refusesChange(t, name, hidden...)
	if marked := w.marksAllRead(t, name); marked != unread {
		t.Errorf("%s's mark-all-read marked %d; want %d", name, marked, unread)
	}
}

// memberRow is the id of the account name's undeleted row of table, a
// table of memberships whose column scope holds the project or the
// workspace of id.
func (w issueWorld) memberRow(t *testing.T, table, scope string, id uuid.UUID, name string) string {
	t.Helper()
	rows := queryIDs(t, w.pool, "SELECT id FROM "+table+" WHERE "+scope+" = $1 AND member_id = $2 AND deleted_at IS NULL", id, w.ids[name])
	if len(rows) != 1 {
		t.Fatalf("%s's memberships of %s in %s: %v; want one", name, id, table, rows)
	}
	return rows[0].String()
}

// A receiver's notifications in acme show what he reads now (M7 P1c spec
// 2.5, R1, R2; acceptance "可见"): the list, the unread counts and
// mark-all-read's marked leave out, and a change of one refuses, each of
// a project he does not read now, and in a project whose guests see only
// their own work items each of a work item he did not create. Before
// anything changes, alice makes a work item assigned to each receiver in
// the project his case is about, and to olga in Ops too, which sends him
// its creation and his assignment, and she comments on the work item walt
// made in Web. Then: olga, removed from Web, sees only Ops's, until she is
// added back and sees Web's again, unread; walt, made Web's guest, and
// then acme's admin as well, sees only his own work item's, until he is
// Web's admin and sees the rest, unread; pia, who left the private
// Secret, sees none; bob sees Ops's once it is archived, and changes one;
// wanda sees none of Gone's once it is deleted, while its
// notification.delete_rows is held and its notifications are undeleted
// and unread; and bob, no longer acme's member, is answered
// workspace.not_found.
func TestNotificationsShowWhatTheReaderReadsNow(t *testing.T) {
	dbURL := pgtest.NewDatabase(t)
	pool := openPool(t, dbURL)
	holdNotificationDeletions(t, pool)
	w := newIssueWorldOn(t, dbURL, pool, func(*config.Config) {})
	alice := w.ids["alice"]
	w.tokens["pia"] = registerAccount(t, w.contract, w.base, "pia@example.com").AccessToken
	w.ids["pia"] = accountID(t, w.contract, w.base, w.tokens["pia"])
	for _, name := range []string{"olga", "pia"} {
		inWorkspaceOf(t, pool, w.web, w.ids[name], alice, shared.RoleMember)
	}
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	secret := createdProjectFrom(t, w.contract, w.base, w.tokens["alice"], "acme", `{"name":"Secret","identifier":"SEC","network":0}`)
	gone := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Gone", "GONE")
	for _, m := range []struct {
		name    string
		project uuid.UUID
		role    shared.Role
	}{{"walt", w.web, shared.RoleMember}, {"olga", w.web, shared.RoleMember}, {"olga", ops, shared.RoleMember}, {"bob", ops, shared.RoleMember},
		{"pia", secret, shared.RoleMember}, {"wanda", gone, shared.RoleAdmin}} {
		w.makeMember(t, m.name, m.project, m.role)
	}
	walts := w.newWorkItemBy(t, "walt", w.web, `{"name":"Walt's"}`)
	w.send(t, "alice", http.MethodPost, commentsOf(walts), `{"comment_html":"<p>Seen</p>"}`, http.StatusCreated)
	for _, a := range []struct {
		project  uuid.UUID
		assignee string
	}{{w.web, "walt"}, {w.web, "olga"}, {ops, "olga"}, {ops, "bob"}, {secret, "pia"}, {gone, "wanda"}} {
		w.newWorkItemIn(t, a.project, `{"name":"Task","assignee_ids":`+idArray(w.ids[a.assignee])+`}`)
	}
	waitForFanouts(t, pool)
	sent := map[string][]map[string]any{}
	for name, want := range map[string]int{"olga": 4, "walt": 3, "pia": 2, "bob": 2, "wanda": 2} {
		if sent[name] = w.inbox(t, name, "").rows; len(sent[name]) != want {
			t.Fatalf("%s is sent %v; want %d notifications", name, noteIDs(sent[name]), want)
		}
	}
	of := func(name, key string, value uuid.UUID, in bool) []string {
		return notesWhere(sent[name], func(r map[string]any) bool { return (r[key] == value.String()) == in })
	}

	t.Run("removed from a project, and back", func(t *testing.T) {
		inWeb, inOps := of("olga", "project_id", w.web, true), of("olga", "project_id", w.web, false)
		w.send(t, "alice", http.MethodDelete, "/api/v0/project-members/"+w.memberRow(t, "project_members", "project_id", w.web, "olga"), "",
			http.StatusNoContent)
		w.shows(t, "olga", inOps, len(inOps), inWeb...)
		w.makeMember(t, "olga", w.web, shared.RoleMember)
		if got, want := w.inboxIDs(t, "olga", "?read=false"), slices.Sorted(slices.Values(inWeb)); !slices.Equal(got, want) {
			t.Errorf("olga's unread, back in Web = %v; want Web's %v", got, want)
		}
		w.shows(t, "olga", noteIDs(sent["olga"]), len(inWeb))
	})
	own, others := of("walt", "issue_id", walts, true), of("walt", "issue_id", walts, false)
	waltInWeb := w.memberRow(t, "project_members", "project_id", w.web, "walt")
	t.Run("demoted to a guest", func(t *testing.T) {
		w.send(t, "alice", http.MethodPatch, "/api/v0/project-members/"+waltInWeb, `{"role":5}`, http.StatusOK)
		w.shows(t, "walt", own, len(own), others...)
	})
	t.Run("a workspace admin who is a guest", func(t *testing.T) {
		w.send(t, "alice", http.MethodPatch, "/api/v0/workspace-members/"+w.memberRow(t, "workspace_members", "workspace_id", w.acme, "walt"),
			`{"role":20}`, http.StatusOK)
		w.shows(t, "walt", own, 0, others...)
		w.send(t, "alice", http.MethodPatch, "/api/v0/project-members/"+waltInWeb, `{"role":20}`, http.StatusOK)
		if got, want := w.inboxIDs(t, "walt", "?read=false"), slices.Sorted(slices.Values(others)); !slices.Equal(got, want) {
			t.Errorf("walt's unread, Web's admin = %v; want the rest %v", got, want)
		}
	})
	t.Run("left a private project", func(t *testing.T) {
		w.send(t, "pia", http.MethodPost, "/api/v0/projects/"+secret.String()+"/leave", "", http.StatusNoContent)
		w.shows(t, "pia", nil, 0, noteIDs(sent["pia"])...)
	})
	bobs := noteIDs(sent["bob"])
	t.Run("an archived project", func(t *testing.T) {
		w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+ops.String()+"/archive", "", http.StatusOK)
		w.shows(t, "bob", bobs, len(bobs))
		w.send(t, "bob", http.MethodPatch, "/api/v0/notifications/"+bobs[0], `{"archived":true}`, http.StatusOK)
	})
	t.Run("a deleted project", func(t *testing.T) {
		wandas := noteIDs(sent["wanda"])
		w.send(t, "alice", http.MethodDelete, "/api/v0/projects/"+gone.String(), "", http.StatusNoContent)
		w.shows(t, "wanda", nil, 0, wandas...)
		if job, ok := notificationDeletionJob(pgtest.Soon(t), t, pool, gone); !ok || job.Attempt != 0 {
			t.Errorf("Gone's notification deletion job %+v, found %t; want it enqueued and never run", job, ok)
		}
		var live int
		if err := pool.QueryRow(pgtest.Soon(t), `SELECT count(*) FROM notifications WHERE id = ANY($1::uuid[]) AND deleted_at IS NULL AND read_at IS NULL`,
			wandas).Scan(&live); err != nil || live != len(wandas) {
			t.Errorf("Gone's notifications to wanda: %d undeleted and unread, %v; want all %d, their deletion held", live, err, len(wandas))
		}
	})
	t.Run("no longer the workspace's member", func(t *testing.T) {
		w.send(t, "alice", http.MethodDelete, "/api/v0/workspace-members/"+w.memberRow(t, "workspace_members", "workspace_id", w.acme, "bob"), "",
			http.StatusNoContent)
		for _, r := range []struct{ method, path, body string }{
			{http.MethodGet, inboxPath, ""}, {http.MethodGet, inboxPath + "/unread-count", ""}, {http.MethodPost, inboxPath + "/mark-all-read", `{}`},
		} {
			status, body := call(t, w.contract, r.method, w.base+r.path, w.tokens["bob"], r.body)
			if got := refusalOf(t, []byte(body), false); status != http.StatusNotFound || got != "workspace.not_found" {
				t.Errorf("bob's %s %s = %d %s; want 404 workspace.not_found", r.method, r.path, status, got)
			}
		}
		w.refusesChange(t, "bob", bobs...)
	})
}
