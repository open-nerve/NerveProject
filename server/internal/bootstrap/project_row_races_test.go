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

// A write on a row under a project that waits for a lock, against what
// another transaction changes or holds meanwhile, on the wired app (M3
// design 3.6 convention 2, the lock table): what it finds once it has its
// locks, and each lock it takes, at its strength.

// rowWrite is one of the writes on a row under a project as the races send
// it on memberWorld: bob, Web's admin and acme's member, changes gina's
// role in Web to a guest's, or removes her; dave, Web's admin, leaves it;
// bob renames Web's QA, deletes it, or makes it Web's default; or bob
// creates a state in Web, a write addressed by its project, as leaving is.
// gina is Web's member and acme's admin; alice, Web's other admin, stays;
// QA is of the completed group, beside Done; nothing else refuses each
// write.
type rowWrite struct {
	op, by string
	// member is whose membership of Web a write on a membership changes,
	// the caller's own for leaving. A write on a state changes none: it
	// carries its caller, whose membership of acme the lock test reads, and
	// state is the name of Web's state it changes instead. A creation
	// changes neither and carries neither; the races end its caller's
	// membership through by, as every write's.
	member, state      string
	method, path, body string // path: %s the row's id for a path of one (rowPaths), else Web's
	status             int    // its answer, alone
	notFound           string // the code of its 404
	target             bool   // it changes the member's role: it shares his membership of acme (convention 3)
	clears             bool   // it makes its state Web's default: it writes Web's default, Backlog, before its state
	creates            bool   // it creates a state of Web: it has no row of its own to wait on
}

var rowWrites = []rowWrite{
	{op: "updateProjectMember", by: "bob", member: "gina", method: http.MethodPatch, path: "/api/v0/project-members/%s", body: `{"role":5}`,
		status: http.StatusOK, notFound: "project.member_not_found", target: true},
	{op: "removeProjectMember", by: "bob", member: "gina", method: http.MethodDelete, path: "/api/v0/project-members/%s",
		status: http.StatusNoContent, notFound: "project.member_not_found"},
	{op: "leaveProject", by: "dave", member: "dave", method: http.MethodPost, path: "/api/v0/projects/%s/leave", status: http.StatusNoContent,
		notFound: "project.not_found"},
	{op: "updateState", by: "bob", member: "bob", state: "QA", method: http.MethodPatch, path: "/api/v0/states/%s", body: `{"name":"Checked"}`,
		status: http.StatusOK, notFound: "project.state_not_found"},
	{op: "deleteState", by: "bob", member: "bob", state: "QA", method: http.MethodDelete, path: "/api/v0/states/%s",
		status: http.StatusNoContent, notFound: "project.state_not_found"},
	{op: "markDefaultState", by: "bob", member: "bob", state: "QA", method: http.MethodPost, path: "/api/v0/states/%s/mark-default",
		status: http.StatusNoContent, notFound: "project.state_not_found", clears: true},
	{op: "createState", by: "bob", method: http.MethodPost, path: "/api/v0/projects/%s/states",
		body: `{"name":"Checked","color":"#0EA5E9","group":"completed"}`, status: http.StatusCreated, notFound: "project.not_found",
		creates: true},
}

// row is the row of Web that m changes: its table and its id. A creation
// has no row of its own: asked for one, it fails the test.
func (m rowWrite) row(t *testing.T, w memberWorld) (string, uuid.UUID) {
	t.Helper()
	if m.creates {
		t.Fatalf("%s creates a state: it has no row of its own", m.op)
	}
	if m.state != "" {
		return "states", stateID(t, w.pool, w.web, m.state)
	}
	return "project_members", w.membership(t, w.web, m.member)
}

// sent sends m on w's app, checked against the contract, and hands over its
// answer.
func (m rowWrite) sent(t *testing.T, w memberWorld) (*http.Request, <-chan answer) {
	t.Helper()
	named := w.web
	if m.param() != "{project_id}" {
		_, named = m.row(t, w)
	}
	var body []byte
	if m.body != "" {
		body = []byte(m.body)
	}
	req := newRequest(t, m.method, w.base+strings.Replace(m.path, "%s", named.String(), 1), w.tokens[m.by], body)
	w.contract.CheckRequest(t, req)
	return req, sendInBackground(req)
}

// param is the parameter m's path names, as projectWrite.param.
func (m rowWrite) param() string {
	return projectWrite{path: m.path}.param()
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

// A write on a row under a project that waits for a lock reads, once it has
// it, what another transaction committed meanwhile, and answers its 404, as
// for a row or a project never there, never the 403 that would tell its
// caller that it was there; and it changes no row (M3 design 3.6
// convention 2, 8.2). The other transaction holds Web FOR NO KEY UPDATE, as
// a write on it does, and ends (a membership), deletes or moves to Ops the
// row the write changes, or ends the caller's own membership, or deletes
// Web; or it holds acme FOR NO KEY UPDATE, as a cascade does, and deletes
// acme. SQL makes each change inside the other transaction, which must hold
// its lock open across the probe: a real write cannot without a hook in
// product code. The write has passed authentication and its read without a
// lock, of the row it names or, leaving or creating a state, of Web's
// workspace, and waits for the row the other holds. Once the other commits,
// the write is 404; the row the other changed is as it left it, and every
// other row as it was, no state created among them. Web is private: bob or
// dave, his membership ended, does not see it.
func TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile(t *testing.T) {
	type change struct {
		name, holds, sql string // holds: the table of the row the other transaction locks first, acme's or Web's; sql: %s the table
		table            string // the table sql changes; empty for the write's row's
		ofRow            bool   // the change is of the write's row, not its caller's own membership
		membership       bool   // it ends its row: a membership's change alone
		row              func(w memberWorld, t *testing.T, m rowWrite) uuid.UUID
	}
	ofRow := func(w memberWorld, t *testing.T, m rowWrite) uuid.UUID {
		_, id := m.row(t, w)
		return id
	}
	callers := func(w memberWorld, t *testing.T, m rowWrite) uuid.UUID { return w.membership(t, w.web, m.by) }
	web := func(w memberWorld, _ *testing.T, _ rowWrite) uuid.UUID { return w.web }
	acme := func(w memberWorld, t *testing.T, _ rowWrite) uuid.UUID {
		var id uuid.UUID
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT id FROM workspaces WHERE slug = 'acme'").Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	changes := []change{
		{"the row ended", "projects", "UPDATE %s SET is_active = false WHERE id = $1", "", true, true, ofRow},
		{"the row deleted", "projects", "UPDATE %s SET deleted_at = now() WHERE id = $1", "", true, false, ofRow},
		{"the row moved to Ops", "projects", "UPDATE %s SET project_id = (SELECT id FROM projects WHERE name = 'Ops') WHERE id = $1", "", true,
			false, ofRow},
		{"the caller's membership ended", "projects", "UPDATE %s SET is_active = false WHERE id = $1", "project_members", false, false, callers},
		{"Web deleted", "projects", "UPDATE %s SET deleted_at = now(), updated_at = now() WHERE id = $1", "projects", false, false, web},
		{"acme deleted", "workspaces", "UPDATE %s SET deleted_at = now(), updated_at = now() WHERE id = $1", "workspaces", false, false, acme},
	}
	for _, m := range rowWrites {
		for _, c := range changes {
			switch {
			case c.ofRow && m.creates:
				continue // a creation has no row of its own
			case c.ofRow && m.state == "" && m.member == m.by:
				continue // leaving: its row is the caller's own membership, which "the caller's membership ended" ends
			case c.membership && m.state != "":
				continue // a state does not end
			}
			t.Run(m.op+", "+c.name, func(t *testing.T) {
				w := newPrivateWorld(t)
				table := c.table
				if table == "" {
					table, _ = m.row(t, w)
				}
				id := c.row(w, t, m)
				others := rowsBut(t, w.pool, []uuid.UUID{id})
				first := "SELECT 1 FROM projects WHERE id = $1 FOR NO KEY UPDATE"
				firstID := w.web
				if c.holds == "workspaces" {
					first, firstID = "SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", acme(w, t, m)
				}
				other := holding(t, w.pool, first, firstID)
				sql := strings.Replace(c.sql, "%s", table, 1)
				if tag, err := other.Exec(pgtest.Soon(t), sql, id); err != nil || tag.RowsAffected() != 1 {
					t.Fatalf("%s: %v, %v; want one row changed", sql, tag, err)
				}
				changed := rowJSON(t, other, table, id)
				req, answered := m.sent(t, w)
				pgtest.WaitForLockWaitOn(t, w.pool, c.holds, 5*time.Second)
				if err := other.Commit(pgtest.Soon(t)); err != nil {
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
				if after := rowJSON(t, w.pool, table, id); !reflect.DeepEqual(after, changed) {
					t.Errorf("%s %s after the write:\n%v\nwant it as the other transaction left it:\n%v", table, id, after, changed)
				}
				if after := rowsBut(t, w.pool, []uuid.UUID{id}); !maps.Equal(after, others) {
					t.Errorf("every other row after the write:\n%v\nwant them as they were:\n%v", after, others)
				}
			})
		}
	}
}

// Each lock of a write on a row under a project is taken in the lock
// table's order, at its strength, as bootstrap wires it (M3 design 3.6's
// lock table, conventions 2 and 3). Other transactions hold acme's row FOR
// NO KEY UPDATE and, FOR SHARE, Web's row and the row of Web the write
// changes, which the write waits for in turn; they let go one at a time,
// and lockOn reads each row's strongest lock then:
//   - waiting for acme's row, the write holds neither the member's
//     membership of acme, nor Web's default state, nor Ops, and nothing of
//     Web or of its row that conflicts with the others' shares, or it would
//     wait there; a share of either, hidden under the others', is
//     TestEachWriteOnAProjectSharesItsWorkspaceFirst's to see;
//   - waiting for Web's row, it holds acme's FOR SHARE and, for a change of
//     a role, the member's membership of acme FOR SHARE (convention 3), no
//     stronger; that membership not at all for any other write;
//   - waiting for its row, it holds Web FOR NO KEY UPDATE too, and still not
//     Ops, alice's other project of acme, of which gina and dave are no
//     members; Web's default, Backlog, only a write that makes its state
//     the default holds, FOR NO KEY UPDATE, having written it first (M3
//     design 3.17).
//
// Then it answers as alone. The moment it wrote is no earlier than Web's
// release, as it read the clock under Web's lock (3.3), and earlier than
// its row's release: it read the clock before it waited for its row, which
// only its write's UPDATE locks; nothing before the decision and the clock
// does, not the read of it under the locks either.
func TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength(t *testing.T) {
	for _, m := range rowWrites {
		if m.creates {
			// A creation has no row of its own to wait on: the order and the
			// strength of its locks are TestEachWriteOnAProjectSharesItsWorkspaceFirst's
			// and the two creations' of TestStateWritesOnOneProjectSerialize.
			continue
		}
		t.Run(m.op, func(t *testing.T) {
			w := newMemberWorld(t)
			var acme, inAcme uuid.UUID
			if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT s.id, m.id FROM workspaces s JOIN workspace_members m ON m.workspace_id = s.id
				WHERE s.slug = 'acme' AND m.member_id = $1`, w.ids[m.member]).Scan(&acme, &inAcme); err != nil {
				t.Fatal(err)
			}
			table, row := m.row(t, w)
			backlog := stateID(t, w.pool, w.web, "Backlog")
			locks := func() string {
				t.Helper()
				return "acme " + lockOn(t, w.pool, "workspaces WHERE id = $1", acme) +
					", in acme " + lockOn(t, w.pool, "workspace_members WHERE id = $1", inAcme) +
					", Web " + lockOn(t, w.pool, "projects WHERE id = $1", w.web) +
					", its row " + lockOn(t, w.pool, table+" WHERE id = $1", row) +
					", Web's default " + lockOn(t, w.pool, "states WHERE id = $1", backlog) +
					", Ops " + lockOn(t, w.pool, "projects WHERE id = $1", w.ops)
			}
			inAcmeShared, defaultHeld := "no lock", "no lock"
			if m.target {
				inAcmeShared = "FOR SHARE"
			}
			if m.clears {
				defaultHeld = "FOR NO KEY UPDATE"
			}
			holdsRow := holding(t, w.pool, "SELECT 1 FROM "+table+" WHERE id = $1 FOR SHARE", row)
			holdsWeb := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR SHARE", w.web)
			holdsAcme := holding(t, w.pool, "SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", acme)
			req, answered := m.sent(t, w)
			var releasedWeb time.Time
			for _, step := range []struct {
				release pgx.Tx
				waitsOn string
				want    string
			}{
				{nil, "workspaces", "acme FOR NO KEY UPDATE, in acme no lock, Web FOR SHARE, its row FOR SHARE, Web's default no lock, Ops no lock"},
				{holdsAcme, "projects", "acme FOR SHARE, in acme " + inAcmeShared + ", Web FOR SHARE, its row FOR SHARE, Web's default no lock, " +
					"Ops no lock"},
				{holdsWeb, table, "acme FOR SHARE, in acme " + inAcmeShared + ", Web FOR NO KEY UPDATE, its row FOR SHARE, Web's default " +
					defaultHeld + ", Ops no lock"},
			} {
				if step.release != nil {
					if step.release == holdsWeb {
						releasedWeb = time.Now()
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
			releasedRow := time.Now()
			if err := holdsRow.Rollback(context.Background()); err != nil {
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
			moment, _ := rowJSON(t, w.pool, table, row)["updated_at"].(string)
			if at, err := time.Parse(time.RFC3339Nano, moment); err != nil || at.Before(releasedWeb.Truncate(time.Microsecond)) ||
				!at.Before(releasedRow) {
				t.Errorf("%s's moment %q (%v); want one no earlier than Web's release, %v, and earlier than its row's, %v", m.op, moment,
					err, releasedWeb, releasedRow)
			}
		})
	}
}
