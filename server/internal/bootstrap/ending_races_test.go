package bootstrap

import (
	"context"
	"maps"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// An ending that waits for acme's row, against what another transaction
// changes or holds meanwhile, on the wired app (M3 design 3.6's lock table,
// convention 1, convention 2): what it finds once it has the lock, and
// each lock it takes, at its strength.

// sent sends e's ending of name's membership on w's app, checked against
// the contract, and hands over its answer.
func (e ending) sent(w endingWorld, t *testing.T, name string) (*http.Request, <-chan answer) {
	t.Helper()
	method, path, token := e.request(w, t, name)
	req := newRequest(t, method, w.base+path, token, nil)
	w.contract.CheckRequest(t, req)
	return req, sendInBackground(req)
}

// An ending that waits for acme's row reads, once it has it, what another
// transaction committed meanwhile, and answers its 404, as for a membership
// or a workspace never there, never the 403 that would tell its caller
// that it was there; and it changes no row (M3 design 3.6 convention 2,
// the lock table). The other transaction holds acme FOR NO KEY UPDATE and
// ends bob's membership, ends alice's, the remover's, or deletes acme; bob's
// removal by alice, or his leaving, has passed authentication and waits for
// acme's row. Once the other commits, the removal is 404
// workspace.member_not_found, the leaving 404 workspace.not_found; the row
// the other changed is as it left it, and every other row as it was. alice
// has joined Ops first, so that nothing else refuses the ending.
func TestAnEndingFindsWhatEndedMeanwhile(t *testing.T) {
	type change struct {
		name, sql string
		row       func(w endingWorld, t *testing.T) (table string, id uuid.UUID)
	}
	membershipOf := func(name string) func(w endingWorld, t *testing.T) (string, uuid.UUID) {
		return func(w endingWorld, t *testing.T) (string, uuid.UUID) {
			return "workspace_members", w.membership(t, name)
		}
	}
	ends := func(name string) change {
		return change{name + "'s membership ended", "UPDATE workspace_members SET is_active = false WHERE id = $1", membershipOf(name)}
	}
	deletesAcme := change{"acme deleted", "UPDATE workspaces SET deleted_at = now(), updated_at = now() WHERE id = $1",
		func(w endingWorld, t *testing.T) (string, uuid.UUID) { return "workspaces", w.workspace(t, "acme") }}
	for _, tt := range []struct {
		ending ending
		change change
	}{
		{endings[0], ends("bob")}, {endings[0], ends("alice")}, {endings[0], deletesAcme},
		{endings[1], ends("bob")}, {endings[1], deletesAcme},
	} {
		t.Run(tt.ending.name+", "+tt.change.name, func(t *testing.T) {
			w := newEndingWorld(t)
			w.joinsOps(t)
			table, id := tt.change.row(w, t)
			others := rowsBut(t, w.pool, []uuid.UUID{id})
			other := holding(t, w.pool, "SELECT 1 FROM workspaces WHERE slug = 'acme' FOR NO KEY UPDATE")
			if tag, err := other.Exec(context.Background(), tt.change.sql, id); err != nil || tag.RowsAffected() != 1 {
				t.Fatalf("%s: %v, %v; want one row changed", tt.change.sql, tag, err)
			}
			changed := rowJSON(t, other, table, id)
			req, answered := tt.ending.sent(w, t, "bob")
			pgtest.WaitForLockWaitOn(t, w.pool, "workspaces", 5*time.Second)
			if err := other.Commit(context.Background()); err != nil {
				t.Fatal(err)
			}

			a := receiveWithin(t, answered, 10*time.Second, "answer to the ending")
			if a.err != nil {
				t.Fatal(a.err)
			}
			w.contract.CheckResponse(t, req, a.res)
			if a.res.StatusCode != http.StatusNotFound || problemCode(t, a.body) != tt.ending.notFound {
				t.Errorf("the ending = %d %s, want 404 %s", a.res.StatusCode, a.body, tt.ending.notFound)
			}
			if after := rowJSON(t, w.pool, table, id); !maps.Equal(after, changed) {
				t.Errorf("%s %s after the ending:\n%v\nwant it as the other transaction left it:\n%v", table, id, after, changed)
			}
			if after := rowsBut(t, w.pool, []uuid.UUID{id}); !maps.Equal(after, others) {
				t.Errorf("every other row after the ending:\n%v\nwant them as they were:\n%v", after, others)
			}
		})
	}
}

// Each lock of an ending is taken in the lock table's order, at its
// strength, as bootstrap wires it (M3 design 3.6's lock table, convention
// 1, convention 6, the global order: the invitations before the members,
// the projects before their members). Other transactions hold acme's row
// FOR NO KEY UPDATE and, FOR SHARE, the pending invitation to bob's
// address, his membership of acme and his membership of Ops, which the
// ending's statements wait for in turn; they let go one at a time, and
// lockOn reads each row's strongest lock then. alice's removal of bob, or
// his leaving, waits for acme's row, holding nothing; then for the
// invitation, holding acme's row alone; then for his membership, holding
// acme's row and the invitation FOR NO KEY UPDATE; then for his membership
// of Ops, holding acme's row, the invitation, his membership, Web and Ops,
// each FOR NO KEY UPDATE, no stronger, no weaker; never Site, alice's
// project of acme, of which he is no member; and never his account at
// FOR SHARE or above, which a deactivation's FOR NO KEY UPDATE of it would
// wait for: the ending reads his address unlocked (convention 1). Then it
// is 204, and the moment it wrote is no earlier than acme's release: it
// read the clock under acme's lock (3.3).
func TestEachLockOfAnEndingIsItsStrength(t *testing.T) {
	for _, e := range endings {
		t.Run(e.name, func(t *testing.T) {
			w := newEndingWorld(t)
			w.joinsOps(t)
			acme, bobs := w.workspace(t, "acme"), w.membership(t, "bob")
			opsMembership := projectMemberships(t, w.pool, w.ids["bob"], w.ops)[0]
			site := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Site", "SITE")
			locks := func() string {
				t.Helper()
				return "acme " + lockOn(t, w.pool, "workspaces WHERE id = $1", acme) +
					", his membership " + lockOn(t, w.pool, "workspace_members WHERE id = $1", bobs) +
					", the invitation " + lockOn(t, w.pool, "workspace_member_invites WHERE id = $1", w.bobsInvitation) +
					", Web " + lockOn(t, w.pool, "projects WHERE id = $1", w.web) +
					", Ops " + lockOn(t, w.pool, "projects WHERE id = $1", w.ops) +
					", Site " + lockOn(t, w.pool, "projects WHERE id = $1", site)
			}
			holdsOps := holding(t, w.pool, "SELECT 1 FROM project_members WHERE id = $1 FOR SHARE", opsMembership)
			holdsMembership := holding(t, w.pool, "SELECT 1 FROM workspace_members WHERE id = $1 FOR SHARE", bobs)
			holdsInvitation := holding(t, w.pool, "SELECT 1 FROM workspace_member_invites WHERE id = $1 FOR SHARE", w.bobsInvitation)
			holdsAcme := holding(t, w.pool, "SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", acme)
			req, answered := e.sent(w, t, "bob")
			var released time.Time
			// Each step lets go of one holder and names the table the
			// ending then waits on, and its locks: acme's is the holder's
			// until the first step; the FOR SHARE ones are the holders'.
			// A lock the ending took on a row held FOR SHARE before it
			// had acme's would hide behind the holder's at the first step,
			// and show only as the next step's probe timing out: its
			// upgrade of a row lock it shares with the holder waits
			// without a tuple lock, which pgtest.WaitForLockWaitOn does
			// not see.
			for _, step := range []struct {
				release pgx.Tx
				waitsOn string
				want    string
			}{
				{nil, "workspaces", "acme FOR NO KEY UPDATE, his membership FOR SHARE, the invitation FOR SHARE, Web no lock, Ops no lock, " +
					"Site no lock"},
				{holdsAcme, "workspace_member_invites", "acme FOR NO KEY UPDATE, his membership FOR SHARE, the invitation FOR SHARE, Web no lock, " +
					"Ops no lock, Site no lock"},
				{holdsInvitation, "workspace_members", "acme FOR NO KEY UPDATE, his membership FOR SHARE, the invitation FOR NO KEY UPDATE, " +
					"Web no lock, Ops no lock, Site no lock"},
				{holdsMembership, "project_members", "acme FOR NO KEY UPDATE, his membership FOR NO KEY UPDATE, the invitation FOR NO KEY UPDATE, " +
					"Web FOR NO KEY UPDATE, Ops FOR NO KEY UPDATE, Site no lock"},
			} {
				if step.release != nil {
					if step.release == holdsAcme {
						released = time.Now()
					}
					if err := step.release.Rollback(context.Background()); err != nil {
						t.Fatal(err)
					}
				}
				pgtest.WaitForLockWaitOn(t, w.pool, step.waitsOn, 5*time.Second)
				if got := locks(); got != step.want {
					t.Errorf("the ending waiting on %s: %s; want %s", step.waitsOn, got, step.want)
				}
				if heldBy(t, w.pool, "SELECT 1 FROM users WHERE id = $1 FOR NO KEY UPDATE", w.ids["bob"]) {
					t.Errorf("the ending waiting on %s holds bob's account at FOR SHARE or above; want it unlocked, or held FOR KEY SHARE at most",
						step.waitsOn)
				}
			}
			if err := holdsOps.Rollback(context.Background()); err != nil {
				t.Fatal(err)
			}

			a := receiveWithin(t, answered, 10*time.Second, "answer to the ending")
			if a.err != nil {
				t.Fatal(a.err)
			}
			w.contract.CheckResponse(t, req, a.res)
			if a.res.StatusCode != http.StatusNoContent {
				t.Errorf("the ending = %d %s, want 204", a.res.StatusCode, a.body)
			}
			moment, _ := rowJSON(t, w.pool, "workspace_members", bobs)["updated_at"].(string)
			if at, err := time.Parse(time.RFC3339Nano, moment); err != nil || at.Before(released.Truncate(time.Microsecond)) {
				t.Errorf("the ending's moment %q (%v); want one no earlier than acme's release, %v", moment, err, released)
			}
		})
	}
}
