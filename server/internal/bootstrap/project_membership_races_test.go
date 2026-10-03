package bootstrap

import (
	"context"
	"maps"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// A write on a project membership that waits for a lock, against what
// another transaction changes or holds meanwhile, on the wired app (M3
// design 3.6 convention 2, the lock table): what it finds once it has its
// locks, and each lock it takes, at its strength.

// membershipWrite is one of the writes on a project membership as the
// races send it on memberWorld: bob, Web's admin and acme's member,
// changes gina's role in Web to a guest's, or removes her; dave, Web's
// admin, leaves it. gina is Web's member and acme's admin; alice, Web's
// other admin, stays; nothing else refuses each write.
type membershipWrite struct {
	op, by, member     string // the caller; whose membership of Web the write changes
	method, path, body string // path: %s the membership's id for a path of one, else Web's
	status             int    // its answer, alone
	notFound           string // the code of its 404
	target             bool   // it changes the member's role: it shares his membership of acme (convention 3)
}

var membershipWrites = []membershipWrite{
	{"updateProjectMember", "bob", "gina", http.MethodPatch, "/api/v0/project-members/%s", `{"role":5}`, http.StatusOK, "project.member_not_found",
		true},
	{"removeProjectMember", "bob", "gina", http.MethodDelete, "/api/v0/project-members/%s", "", http.StatusNoContent, "project.member_not_found",
		false},
	{"leaveProject", "dave", "dave", http.MethodPost, "/api/v0/projects/%s/leave", "", http.StatusNoContent, "project.not_found", false},
}

// sent sends m on w's app, checked against the contract, and hands over its
// answer.
func (m membershipWrite) sent(t *testing.T, w memberWorld) (*http.Request, <-chan answer) {
	t.Helper()
	named := w.web
	if strings.HasPrefix(m.path, "/api/v0/project-members/") {
		named = w.membership(t, w.web, m.member)
	}
	var body []byte
	if m.body != "" {
		body = []byte(m.body)
	}
	req := newRequest(t, m.method, w.base+strings.Replace(m.path, "%s", named.String(), 1), w.tokens[m.by], body)
	w.contract.CheckRequest(t, req)
	return req, sendInBackground(req)
}

// newPrivateWorld is memberWorld with Web private, made so by alice: a
// member of acme who is no member of Web does not see it.
func newPrivateWorld(t *testing.T) memberWorld {
	t.Helper()
	w := newMemberWorld(t)
	if status, body := call(t, w.contract, http.MethodPatch, w.base+"/api/v0/projects/"+w.web.String(), w.tokens["alice"], `{"network":0}`); status !=
		http.StatusOK {
		t.Fatalf("alice's making Web private = %d %s", status, body)
	}
	return w
}

// A write on a project membership that waits for a lock reads, once it has
// it, what another transaction committed meanwhile, and answers its 404, as
// for a membership or a project never there, never the 403 that would tell
// its caller that it was there; and it changes no row (M3 design 3.6
// convention 2, 8.2). The other transaction holds Web FOR NO KEY UPDATE, as
// a write on it does, and ends, deletes or moves to Ops the membership the
// write changes, or ends the caller's own membership, or deletes Web; or it
// holds acme FOR NO KEY UPDATE, as a cascade does, and deletes acme. The
// write has passed authentication, read the membership it names unlocked,
// and waits for that row. Once the other commits, the write is 404; the row
// the other changed is as it left it, and every other row as it was. Web is
// private: bob or dave, his membership ended, does not see it.
func TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile(t *testing.T) {
	type change struct {
		name, holds, table, sql string // holds: the table of the row the other transaction locks first, acme's or Web's
		member                  bool   // the change is of the write's member's membership, not its caller's own
		row                     func(w memberWorld, t *testing.T, m membershipWrite) uuid.UUID
	}
	membershipOf := func(member bool) func(w memberWorld, t *testing.T, m membershipWrite) uuid.UUID {
		return func(w memberWorld, t *testing.T, m membershipWrite) uuid.UUID {
			if member {
				return w.membership(t, w.web, m.member)
			}
			return w.membership(t, w.web, m.by)
		}
	}
	web := func(w memberWorld, _ *testing.T, _ membershipWrite) uuid.UUID { return w.web }
	acme := func(w memberWorld, t *testing.T, _ membershipWrite) uuid.UUID {
		var id uuid.UUID
		if err := w.pool.QueryRow(soon(t), "SELECT id FROM workspaces WHERE slug = 'acme'").Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	changes := []change{
		{"the member's membership ended", "projects", "project_members", "UPDATE project_members SET is_active = false WHERE id = $1", true,
			membershipOf(true)},
		{"the member's membership deleted", "projects", "project_members", "UPDATE project_members SET deleted_at = now() WHERE id = $1", true,
			membershipOf(true)},
		{"the member's membership moved to Ops", "projects", "project_members",
			"UPDATE project_members SET project_id = (SELECT id FROM projects WHERE name = 'Ops') WHERE id = $1", true, membershipOf(true)},
		{"the caller's membership ended", "projects", "project_members", "UPDATE project_members SET is_active = false WHERE id = $1", false,
			membershipOf(false)},
		{"Web deleted", "projects", "projects", "UPDATE projects SET deleted_at = now(), updated_at = now() WHERE id = $1", false, web},
		{"acme deleted", "workspaces", "workspaces", "UPDATE workspaces SET deleted_at = now(), updated_at = now() WHERE id = $1", false, acme},
	}
	for _, m := range membershipWrites {
		for _, c := range changes {
			if c.member && m.member == m.by {
				continue // leaving: the member's membership is the caller's
			}
			t.Run(m.op+", "+c.name, func(t *testing.T) {
				w := newPrivateWorld(t)
				id := c.row(w, t, m)
				others := rowsBut(t, w.pool, []uuid.UUID{id})
				first := "SELECT 1 FROM projects WHERE id = $1 FOR NO KEY UPDATE"
				firstID := w.web
				if c.holds == "workspaces" {
					first, firstID = "SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", acme(w, t, m)
				}
				other := holding(t, w.pool, first, firstID)
				if tag, err := other.Exec(soon(t), c.sql, id); err != nil || tag.RowsAffected() != 1 {
					t.Fatalf("%s: %v, %v; want one row changed", c.sql, tag, err)
				}
				changed := rowJSON(t, other, c.table, id)
				req, answered := m.sent(t, w)
				pgtest.WaitForLockWaitOn(t, w.pool, c.holds, 5*time.Second)
				if err := other.Commit(soon(t)); err != nil {
					t.Fatal(err)
				}

				a := receiveWithin(t, answered, 10*time.Second, "the answer to "+m.op)
				if a.err != nil {
					t.Fatal(a.err)
				}
				w.contract.CheckResponse(t, req, a.res)
				if a.res.StatusCode != http.StatusNotFound || problemCode(t, a.body) != m.notFound {
					t.Errorf("%s = %d %s, want 404 %s", m.op, a.res.StatusCode, a.body, m.notFound)
				}
				if after := rowJSON(t, w.pool, c.table, id); !reflect.DeepEqual(after, changed) {
					t.Errorf("%s %s after the write:\n%v\nwant it as the other transaction left it:\n%v", c.table, id, after, changed)
				}
				if after := rowsBut(t, w.pool, []uuid.UUID{id}); !maps.Equal(after, others) {
					t.Errorf("every other row after the write:\n%v\nwant them as they were:\n%v", after, others)
				}
			})
		}
	}
}

// Each lock of a write on a project membership is taken in the lock
// table's order, at its strength, as bootstrap wires it (M3 design 3.6's
// lock table, conventions 2 and 3). Other transactions hold acme's row FOR
// NO KEY UPDATE and, FOR SHARE, Web's row and the membership of Web the
// write changes, which the write waits for in turn; they let go one at a
// time, and lockOn reads each row's strongest lock then. The write waits
// for acme's row, holding nothing; then for Web's, holding acme's FOR
// SHARE and, for a change of a role, the member's membership of acme FOR
// SHARE (convention 3), no stronger; then for the membership of Web, holding
// Web FOR NO KEY UPDATE too; never the member's membership of acme for a
// removal or a leaving, nor Ops, alice's other project of acme, of which
// gina and dave are no members. Then it answers as alone, and the moment it
// wrote is no earlier than Web's release: it read the clock under Web's
// lock (3.3).
func TestEachLockOfAWriteOnAProjectMembershipIsItsStrength(t *testing.T) {
	for _, m := range membershipWrites {
		t.Run(m.op, func(t *testing.T) {
			w := newMemberWorld(t)
			var acme, inAcme uuid.UUID
			if err := w.pool.QueryRow(soon(t), `SELECT s.id, m.id FROM workspaces s JOIN workspace_members m ON m.workspace_id = s.id
				WHERE s.slug = 'acme' AND m.member_id = $1`, w.ids[m.member]).Scan(&acme, &inAcme); err != nil {
				t.Fatal(err)
			}
			inWeb := w.membership(t, w.web, m.member)
			locks := func() string {
				t.Helper()
				return "acme " + lockOn(t, w.pool, "workspaces WHERE id = $1", acme) +
					", in acme " + lockOn(t, w.pool, "workspace_members WHERE id = $1", inAcme) +
					", Web " + lockOn(t, w.pool, "projects WHERE id = $1", w.web) +
					", in Web " + lockOn(t, w.pool, "project_members WHERE id = $1", inWeb) +
					", Ops " + lockOn(t, w.pool, "projects WHERE id = $1", w.ops)
			}
			inAcmeShared := "no lock"
			if m.target {
				inAcmeShared = "FOR SHARE"
			}
			holdsInWeb := holding(t, w.pool, "SELECT 1 FROM project_members WHERE id = $1 FOR SHARE", inWeb)
			holdsWeb := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR SHARE", w.web)
			holdsAcme := holding(t, w.pool, "SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", acme)
			req, answered := m.sent(t, w)
			var released time.Time
			for _, step := range []struct {
				release pgx.Tx
				waitsOn string
				want    string
			}{
				{nil, "workspaces", "acme FOR NO KEY UPDATE, in acme no lock, Web FOR SHARE, in Web FOR SHARE, Ops no lock"},
				{holdsAcme, "projects", "acme FOR SHARE, in acme " + inAcmeShared + ", Web FOR SHARE, in Web FOR SHARE, Ops no lock"},
				{holdsWeb, "project_members", "acme FOR SHARE, in acme " + inAcmeShared + ", Web FOR NO KEY UPDATE, in Web FOR SHARE, Ops no lock"},
			} {
				if step.release != nil {
					if step.release == holdsWeb {
						released = time.Now()
					}
					if err := step.release.Rollback(context.Background()); err != nil {
						t.Fatal(err)
					}
				}
				pgtest.WaitForLockWaitOn(t, w.pool, step.waitsOn, 5*time.Second)
				if got := locks(); got != step.want {
					t.Errorf("%s waiting on %s: %s; want %s", m.op, step.waitsOn, got, step.want)
				}
			}
			if err := holdsInWeb.Rollback(context.Background()); err != nil {
				t.Fatal(err)
			}

			a := receiveWithin(t, answered, 10*time.Second, "the answer to "+m.op)
			if a.err != nil {
				t.Fatal(a.err)
			}
			w.contract.CheckResponse(t, req, a.res)
			if a.res.StatusCode != m.status {
				t.Errorf("%s = %d %s, want %d", m.op, a.res.StatusCode, a.body, m.status)
			}
			moment, _ := rowJSON(t, w.pool, "project_members", inWeb)["updated_at"].(string)
			if at, err := time.Parse(time.RFC3339Nano, moment); err != nil || at.Before(released.Truncate(time.Microsecond)) {
				t.Errorf("%s's moment %q (%v); want one no earlier than Web's release, %v", m.op, moment, err, released)
			}
		})
	}
}
