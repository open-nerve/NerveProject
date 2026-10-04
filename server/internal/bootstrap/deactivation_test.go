package bootstrap

import (
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// A deactivation refused leaves every row of every table as it was, on both
// paths (M3 design 3.7 rule 2, 3.9, 9.3): its refusal comes after its writes
// of the account, its profile and its sessions, and the workspace's refusal
// after its workspaces' locks, the project's after its deletion of the
// invitations and its end of the workspace memberships, so that each of
// those is shown rolled back. Each case has a world of its own.
//   - alice, acme's only active admin, dave's membership as its admin having
//     ended, which counts for nothing, with other active members:
//     workspace.sole_admin, its detail on both paths; beta, where bob is an
//     admin too, refuses nothing.
//   - bob, the only active admin of Ops, of acme, beside carol, its member,
//     and erin, an admin whose membership ended, and of Lab, of beta,
//     beside carol: project.sole_admin. So once alice has joined Lab, as its
//     admin: erin counts for nothing in Ops. So once she has joined Ops
//     instead: Lab is of beta, another of his workspaces, which the check
//     covers too.
//   - bob, once alice has left beta, its only active admin beside carol, its
//     member: workspace.sole_admin, though acme, his first workspace by id,
//     has alice; the check asks of each of his workspaces, and the
//     workspace's refusal comes before the project's.
func TestARefusedDeactivationChangesNothing(t *testing.T) {
	for _, p := range deactivationPaths {
		for _, tt := range []struct {
			name, who, want string
			before          func(w deactivationWorld, t *testing.T)
		}{
			{"acme's only admin", "alice", refusedWorkspaceSoleAdmin, nil},
			{"Ops's and Lab's only admin", "bob", refusedProjectSoleAdmin, nil},
			{"Ops's only admin", "bob", refusedProjectSoleAdmin, func(w deactivationWorld, t *testing.T) { w.clears(t, w.lab) }},
			{"Lab's only admin", "bob", refusedProjectSoleAdmin, func(w deactivationWorld, t *testing.T) { w.clears(t, w.ops) }},
			{"beta's only admin", "bob", refusedWorkspaceSoleAdmin, func(w deactivationWorld, t *testing.T) {
				if status, body := w.leave(t, "beta", "alice"); status != http.StatusNoContent {
					t.Fatalf("alice's leaving beta = %d %s", status, body)
				}
			}},
		} {
			t.Run(p.name+", "+tt.name, func(t *testing.T) {
				w := newDeactivationWorld(t)
				if tt.before != nil {
					tt.before(w, t)
				}
				before := tableRows(t, w.pool, riversOwn)
				if got := p.deactivate(w, t, tt.who); got != tt.want {
					t.Errorf("deactivating %s = %q, want %q", tt.who, got, tt.want)
				}
				if after := tableRows(t, w.pool, riversOwn); !maps.Equal(after, before) {
					t.Errorf("the tables after %s's refused deactivation changed:\n%v\nwant them as they were:\n%v", tt.who, after, before)
				}
			})
		}
	}
}

// A deactivation is one transaction, on both paths (M3 design 3.9): bob's,
// once alice has joined Ops and Lab, refused at its commit, after every
// statement ran, for each table it writes, fails, no problem of the
// contract, and no row of any table changes: no step wrote in a
// transaction of its own.
func TestADeactivationRefusedAtItsCommitChangesNothing(t *testing.T) {
	for _, p := range deactivationPaths {
		t.Run(p.name, func(t *testing.T) {
			w := newDeactivationWorld(t)
			w.clears(t, w.ops, w.lab)
			before := tableRows(t, w.pool, riversOwn)
			for _, table := range []string{"users", "profiles", "auth_sessions", "workspace_member_invites", "workspace_members", "project_members"} {
				restore := refusingCommits(t, w.pool, table)
				got := p.deactivate(w, t, "bob")
				restore()
				if after := tableRows(t, w.pool, riversOwn); got != "500" || !maps.Equal(after, before) {
					t.Errorf("the deactivation refused at its commit for %s = %q; want a failure and every table as it was", table, got)
				}
			}
		})
	}
}

// A deactivation ends every membership of the account and deletes every
// invitation to its address, on both paths, in one transaction at one
// moment, as the account (M3 design 3.6 convention 6, 3.8, 3.9, 9.3): once
// alice has joined Ops and Lab, bob's deactivation ends his memberships of
// acme and beta, and of Web, Ops, Solo, Lab and Docs, across the two
// workspaces, each row kept with its role; it deletes his invitations,
// pending ones to acme and beta and the one to gamma he declined, not
// being its member; each of those rows at one moment no earlier than the
// request, and as bob. Each was stamped by the test as last written by
// dave before, so that the claim of bob's writing can fail; his roles
// differ, a member's of acme and Docs, an admin's of beta and of his other
// projects, Solo among them archived (3.9 ends those too), so that a role
// written over shows. His account is inactive; every other row of every
// table, alice's, carol's, dave's and erin's memberships and the
// invitations to their addresses among them, is as it was. Deactivated, he
// is deactivated again: deactivateMe, his session revoked, is 401; the
// command goes through; neither writes a row of his the first one ended
// or deleted, nor any row of anyone else (the deactivation's statements
// write none on a repeat). Then carol's deactivation goes through: gamma's
// only active admin, she is its only active member too, dave's membership
// having ended (3.7 rule 2), and its membership ends with her others.
func TestADeactivationEndsEveryMembership(t *testing.T) {
	for _, p := range deactivationPaths {
		t.Run(p.name, func(t *testing.T) {
			w := newDeactivationWorld(t)
			w.clears(t, w.ops, w.lab)
			bob := w.ids["bob"]
			var written []uuid.UUID
			var tables []string
			for _, row := range []struct {
				table string
				sql   string
				args  []any
			}{
				{"workspace_members", "SELECT id FROM workspace_members WHERE member_id = $1 AND deleted_at IS NULL ORDER BY id", []any{bob}},
				{"project_members", "SELECT id FROM project_members WHERE member_id = $1 AND deleted_at IS NULL ORDER BY id", []any{bob}},
				{"workspace_member_invites", "SELECT id FROM workspace_member_invites WHERE email = 'bob@example.com' AND deleted_at IS NULL ORDER BY id",
					nil},
			} {
				ids := queryIDs(t, w.pool, row.sql, row.args...)
				written = append(written, ids...)
				tables = append(tables, slices.Repeat([]string{row.table}, len(ids))...)
			}
			if got := len(written); got != 2+5+3 {
				t.Fatalf("bob has %d memberships and invitations to his address; want 2 of workspaces, 5 of projects, 3 invitations", got)
			}
			rowsBefore := make([]map[string]any, len(written))
			for i, id := range written {
				stampWriter(t, w.pool, w.ids["dave"], 1, tables[i], "id = $2", id)
				rowsBefore[i] = rowJSON(t, w.pool, tables[i], id)
			}
			own := slices.Concat([]uuid.UUID{bob}, queryIDs(t, w.pool, "SELECT id FROM profiles WHERE user_id = $1", bob),
				queryIDs(t, w.pool, "SELECT id FROM auth_sessions WHERE user_id = $1", bob))
			others := rowsBut(t, w.pool, slices.Concat(written, own))
			started := time.Now()

			if got := p.deactivate(w, t, "bob"); got != "" {
				t.Fatalf("deactivating bob = %q, want it done", got)
			}

			moment, _ := rowJSON(t, w.pool, "workspace_member_invites", w.bobsDeclined)["deleted_at"].(string)
			at, err := time.Parse(time.RFC3339Nano, moment)
			if err != nil || at.Before(started.Truncate(time.Microsecond)) {
				t.Errorf("the declined invitation deleted at %q (%v); want a moment no earlier than the request, %v", moment, err, started)
			}
			// identity's moment, its account's, read before its transaction,
			// is no later than the Deactivator's, read after its last
			// workspace lock (3.3, 3.9): two moments, in that order.
			if account, err := time.Parse(time.RFC3339Nano, rowJSON(t, w.pool, "users", bob)["updated_at"].(string)); err != nil || account.After(at) {
				t.Errorf("bob's account deactivated at %v (%v), after his memberships ended at %v; want identity's moment no later", account, err, at)
			}
			for i, id := range written {
				want := maps.Clone(rowsBefore[i])
				want["updated_at"], want["updated_by_id"] = moment, bob.String()
				if tables[i] == "workspace_member_invites" {
					want["deleted_at"] = moment
				} else {
					want["is_active"] = false
				}
				if after := rowJSON(t, w.pool, tables[i], id); !maps.Equal(after, want) {
					t.Errorf("%s %s after the deactivation:\n%v\nwant\n%v", tables[i], id, after, want)
				}
			}
			if active, _ := rowJSON(t, w.pool, "users", bob)["is_active"].(bool); active {
				t.Error("bob's account after the deactivation: active; want it inactive")
			}
			if after := rowsBut(t, w.pool, slices.Concat(written, own)); !maps.Equal(after, others) {
				t.Errorf("every other row after the deactivation:\n%v\nwant them as they were:\n%v", after, others)
			}
			ended := make([]map[string]any, len(written))
			for i, id := range written {
				ended[i] = rowJSON(t, w.pool, tables[i], id)
			}
			if got, want := p.deactivate(w, t, "bob"), map[string]string{byAPI.name: "unauthorized: The bearer token is invalid or has expired."}[p.name]; got != want {
				t.Errorf("deactivating bob again = %q, want %q", got, want)
			}
			for i, id := range written {
				if after := rowJSON(t, w.pool, tables[i], id); !maps.Equal(after, ended[i]) {
					t.Errorf("%s %s after deactivating bob again:\n%v\nwant it as his deactivation left it:\n%v", tables[i], id, after, ended[i])
				}
			}
			if after := rowsBut(t, w.pool, slices.Concat(written, own)); !maps.Equal(after, others) {
				t.Errorf("every other row after deactivating bob again:\n%v\nwant them as they were:\n%v", after, others)
			}
			if got := p.deactivate(w, t, "carol"); got != "" {
				t.Fatalf("deactivating carol, gamma's only active member = %q, want it done", got)
			}
			if left := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE member_id = $1 AND is_active", w.ids["carol"]); len(left) != 0 {
				t.Errorf("carol's active memberships after her deactivation: %v; want none, gamma's ended too", left)
			}
		})
	}
}

// A deactivation that leaves a workspace with no active member deletes its
// pending invitations too, whoever sent them, on both paths (M3 design 3.7,
// 3.9; the P6 pre-flight's M1): carol is gamma's admin and its only active
// member, dave's and erin's memberships of it having ended by her removal
// of them; she invites the two again, and is deactivated, which rule 2
// allows. The two invitations are deleted at her memberships' moment, as
// hers, though the test stamped them as last written by alice, each other
// column kept; every other row of every table but hers is as it was: bob's
// invitation to gamma, which he declined and no one can accept, her
// invitation of frank to it, which she deleted before, and the invitations
// to acme and beta, which keep their members, among them.
// dave's and erin's acceptances answer workspace.invitation_not_found, the
// code of a deleted invitation, and gamma keeps no active member: no
// invitation lets anyone into a workspace with no admin. The server's
// administrator still can (a known limit, 3.11): reactivate-member of the
// two makes gamma's active members two members and no admin, and dave's
// deactivation goes through, rule 2 counting admins alone.
func TestADeactivationDeletesTheInvitationsOfAWorkspaceItEmpties(t *testing.T) {
	for _, p := range deactivationPaths {
		t.Run(p.name, func(t *testing.T) {
			w := newDeactivationWorld(t)
			carol, gammaRows := w.ids["carol"], `SELECT count(*) FILTER (WHERE m.role = 20), count(*) FROM workspace_members m
				WHERE m.workspace_id = $1 AND m.is_active AND m.deleted_at IS NULL`
			answerInvitation(t, w.contract, w.base, w.tokens["erin"], "accept", invite(t, w.contract, w.base, w.tokens["carol"], "gamma",
				"erin@example.com"), http.StatusOK)
			erinIn := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id = $1 AND member_id = $2", w.gamma, w.ids["erin"])[0]
			if status, body := call(t, w.contract, http.MethodDelete, w.base+"/api/v0/workspace-members/"+erinIn.String(), w.tokens["carol"],
				""); status != http.StatusNoContent {
				t.Fatalf("carol's removal of erin from gamma = %d %s", status, body)
			}
			names := []string{"dave", "erin"}
			links, invited := map[string]invitationLink{}, []uuid.UUID{}
			for _, name := range names {
				links[name] = invite(t, w.contract, w.base, w.tokens["carol"], "gamma", name+"@example.com")
				invited = append(invited, links[name].id)
			}
			toFrank := invite(t, w.contract, w.base, w.tokens["carol"], "gamma", "frank@example.com")
			if status, body := call(t, w.contract, http.MethodDelete, w.base+"/api/v0/workspace-invitations/"+toFrank.id.String(),
				w.tokens["carol"], ""); status != http.StatusNoContent {
				t.Fatalf("carol's deletion of her invitation of frank to gamma = %d %s", status, body)
			}
			stampWriter(t, w.pool, w.ids["alice"], 2, "workspace_member_invites", "id = ANY ($2::uuid[])", invited)
			var admins, members int
			if err := w.pool.QueryRow(pgtest.Soon(t), gammaRows, w.gamma).Scan(&admins, &members); err != nil || admins != 1 || members != 1 {
				t.Fatalf("gamma: %d active admins of %d active members (%v); want carol alone", admins, members, err)
			}
			rowsBefore := []map[string]any{rowJSON(t, w.pool, "workspace_member_invites", invited[0]),
				rowJSON(t, w.pool, "workspace_member_invites", invited[1])}
			own := slices.Concat([]uuid.UUID{carol}, invited, queryIDs(t, w.pool, `SELECT id FROM profiles WHERE user_id = $1
				UNION ALL SELECT id FROM auth_sessions WHERE user_id = $1 UNION ALL SELECT id FROM workspace_members WHERE member_id = $1
				UNION ALL SELECT id FROM project_members WHERE member_id = $1
				UNION ALL SELECT id FROM workspace_member_invites WHERE email = 'carol@example.com'`, carol))
			others := rowsBut(t, w.pool, own)

			if got := p.deactivate(w, t, "carol"); got != "" {
				t.Fatalf("deactivating carol, gamma's only active member = %q, want it done", got)
			}

			inGamma := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id = $1 AND member_id = $2", w.gamma, carol)[0]
			moment, _ := rowJSON(t, w.pool, "workspace_members", inGamma)["updated_at"].(string)
			for i, id := range invited {
				want := maps.Clone(rowsBefore[i])
				want["deleted_at"], want["updated_at"], want["updated_by_id"] = moment, moment, carol.String()
				if after := rowJSON(t, w.pool, "workspace_member_invites", id); !maps.Equal(after, want) {
					t.Errorf("the invitation of %s to gamma after carol's deactivation:\n%v\nwant\n%v", names[i], after, want)
				}
			}
			if after := rowsBut(t, w.pool, own); !maps.Equal(after, others) {
				t.Errorf("every other row after carol's deactivation:\n%v\nwant them as they were:\n%v", after, others)
			}
			for _, name := range names {
				if status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/workspace-invitations/"+links[name].id.String()+"/accept",
					w.tokens[name], `{"token":"`+links[name].token+`"}`); status != http.StatusNotFound ||
					!strings.Contains(body, `"code":"workspace.invitation_not_found"`) {
					t.Errorf("%s's acceptance of carol's invitation to gamma = %d %s; want 404 workspace.invitation_not_found", name, status, body)
				}
			}
			if err := w.pool.QueryRow(pgtest.Soon(t), gammaRows, w.gamma).Scan(&admins, &members); err != nil || members != 0 {
				t.Errorf("gamma after the acceptances: %d active members (%v); want none", members, err)
			}

			for _, name := range names {
				if out, _, err := runWorkspaces(t, w.url, ReactivateMember("gamma", name+"@example.com")); err != nil ||
					!strings.HasPrefix(out, "reactivated "+name+"@example.com in gamma as member;") {
					t.Fatalf("reactivate-member of %s in gamma = %q, %v", name, out, err)
				}
			}
			if err := w.pool.QueryRow(pgtest.Soon(t), gammaRows, w.gamma).Scan(&admins, &members); err != nil || admins != 0 || members != 2 {
				t.Fatalf("gamma after the two reactivations: %d active admins of %d active members (%v); want none of two", admins, members, err)
			}
			if got := p.deactivate(w, t, "dave"); got != "" {
				t.Errorf("deactivating dave, a member of gamma beside erin and no admin = %q, want it done", got)
			}
		})
	}
}

// queryIDs is the ids sql reads on pool, in its order.
func queryIDs(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) []uuid.UUID {
	t.Helper()
	rows, err := pool.Query(pgtest.Soon(t), sql, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return ids
}
