package bootstrap

import (
	"context"
	"net/http"
	"testing"
	"time"
	"uuid"

	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// newCrossedWorld is the wired app on a database of its own, all through
// the API: bob made acme, and dave joined it by his invitation and was
// removed by him; carol made gamma, and erin joined it and was removed by
// her; then bob invited carol's address to acme, and carol bob's to gamma.
// So each is the only active member of his own workspace, beside an ended
// one, and the two invitations are pending. Of a deactivationWorld it has
// only what a deactivationPath reads: the contract, the app, the pool, the
// tokens, the ids and the database's url.
func newCrossedWorld(t *testing.T) deactivationWorld {
	t.Helper()
	dbURL := pgtest.NewDatabase(t)
	w := deactivationWorld{endingWorld: endingWorld{contract: apitest.Load(t), pool: openPool(t, dbURL), tokens: map[string]string{},
		ids: map[string]uuid.UUID{}}}
	w.base, w.url = startApp(t, testConfig(t, dbURL, false), migrations.FS()), w.pool.Config().ConnString()
	for _, name := range []string{"bob", "carol", "dave", "erin"} {
		w.tokens[name] = registerAccount(t, w.contract, w.base, name+"@example.com").AccessToken
		w.ids[name] = accountID(t, w.contract, w.base, w.tokens[name])
	}
	for _, c := range []struct{ admin, slug, ended, invites string }{{"bob", "acme", "dave", "carol"}, {"carol", "gamma", "erin", "bob"}} {
		if status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/workspaces", w.tokens[c.admin],
			`{"name":"`+c.slug+`","slug":"`+c.slug+`"}`); status != http.StatusCreated {
			t.Fatalf("creating %s = %d %s", c.slug, status, body)
		}
		answerInvitation(t, w.contract, w.base, w.tokens[c.ended], "accept", invite(t, w.contract, w.base, w.tokens[c.admin], c.slug,
			c.ended+"@example.com"), http.StatusOK)
		ended := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id = $1 AND member_id = $2", w.workspace(t, c.slug),
			w.ids[c.ended])[0]
		if status, body := call(t, w.contract, http.MethodDelete, w.base+"/api/v0/workspace-members/"+ended.String(), w.tokens[c.admin],
			""); status != http.StatusNoContent {
			t.Fatalf("%s's removal of %s = %d %s", c.admin, c.ended, status, body)
		}
	}
	for _, c := range []struct{ admin, slug, invites string }{{"bob", "acme", "carol"}, {"carol", "gamma", "bob"}} {
		invite(t, w.contract, w.base, w.tokens[c.admin], c.slug, c.invites+"@example.com")
	}
	if got, want := w.crossed(t), "bob active, carol active, dave active, erin active; acme bob 20 active, acme dave 15 ended, "+
		"gamma carol 20 active, gamma erin 15 ended; acme carol@example.com pending, gamma bob@example.com pending"; got != want {
		t.Fatalf("the crossed world: %s; want %s", got, want)
	}
	return w
}

// crossed is the accounts of a crossed world, their memberships, and the
// invitations not deleted, each in byte order.
func (w deactivationWorld) crossed(t *testing.T) string {
	t.Helper()
	var got string
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT concat_ws('; ',
		(SELECT string_agg(split_part(email, '@', 1) || CASE WHEN is_active THEN ' active' ELSE ' deactivated' END, ', '
			ORDER BY email COLLATE "C") FROM users),
		(SELECT string_agg(s.slug || ' ' || split_part(u.email, '@', 1) || ' ' || m.role || CASE WHEN m.is_active THEN ' active' ELSE ' ended' END,
			', ' ORDER BY s.slug COLLATE "C", u.email COLLATE "C") FROM workspace_members m JOIN workspaces s ON s.id = m.workspace_id
			JOIN users u ON u.id = m.member_id),
		coalesce((SELECT string_agg(s.slug || ' ' || i.email || CASE WHEN i.responded_at IS NULL THEN ' pending' ELSE ' declined' END, ', '
			ORDER BY s.slug COLLATE "C") FROM workspace_member_invites i JOIN workspaces s ON s.id = i.workspace_id WHERE i.deleted_at IS NULL),
			'no invitation'))`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	return got
}

// Two deactivations whose workspaces invite each other's address both go
// through (M3 design 3.6's global order, 3.9; the P6 final review's I1).
// Each deletes the invitation to its address, of the other's workspace,
// which it does not lock, and the pending one of its own, which it leaves
// with no active member, its ended member not counted: the two rows the
// other deletes too. The first, bob's or carol's, `nerve users deactivate`
// as bootstrap wires it, stops at a gate once it has deleted its
// invitations (deletedInvitations); the second, on each path, waits for an
// invitation the first holds. Every deactivation locks both rows in id
// order before it deletes either. Had each deleted the invitation to its
// address first, and its own workspace's then, the second would hold the
// one to its address, of the first's workspace, and wait for its own
// workspace's, to the first's address, which the first deleted; the first,
// stopped between its two deletes, would wait for the second's: 40P01, one
// of them rolled back. Once the gate opens both are done: both accounts
// deactivated, their memberships ended, no invitation left; and the
// second's moment is no earlier than the gate's opening: it read the clock
// after the invitations' lock it waited for (3.3).
func TestTwoDeactivationsWithCrossedInvitationsBothGoThrough(t *testing.T) {
	for _, first := range []string{"bob", "carol"} {
		second := map[string]string{"bob": "carol", "carol": "bob"}[first]
		for _, p := range deactivationPaths {
			t.Run(first+" first, "+second+" by "+p.name, func(t *testing.T) {
				w := newCrossedWorld(t)
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				g := newGate()
				gated := deactivating(w.pool, identitypg.New(w.pool), deletedInvitations{workspacepg.New(w.pool), g})
				done := run(func() error {
					_, err := gated.ExecuteByEmail(ctx, first+"@example.com")
					return err
				})
				held(t, ctx, g, done, "the first deactivation")
				answer := p.sent(w, t, second)
				pgtest.WaitForLockWaitOn(t, w.pool, "workspace_member_invites", 5*time.Second)
				opened := time.Now()
				close(g.open)

				if err := result(t, ctx, done, "the first deactivation"); err != nil {
					t.Errorf("%s's deactivation, the first = %v; want it done", first, err)
				}
				if got := answer(); got != "" {
					t.Errorf("%s's deactivation, the second = %q; want it done", second, got)
				}
				if got, want := w.crossed(t), "bob deactivated, carol deactivated, dave active, erin active; acme bob 20 ended, acme dave 15 ended, "+
					"gamma carol 20 ended, gamma erin 15 ended; no invitation"; got != want {
					t.Errorf("after both: %s; want %s", got, want)
				}
				moment, _ := rowJSON(t, w.pool, "workspace_members", queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE member_id = $1",
					w.ids[second])[0])["updated_at"].(string)
				if at, err := time.Parse(time.RFC3339Nano, moment); err != nil || at.Before(opened.Truncate(time.Microsecond)) {
					t.Errorf("%s's moment %q (%v); want one no earlier than the gate's opening, %v", second, moment, err, opened)
				}
			})
		}
	}
}

// An invitation to bob's address created after his deactivation locked its
// invitations is not his deactivation's to delete, and carol's, which
// deletes it, does not wait for his in a cycle (ruling F-1; P6 spec section
// 3 item 20). In a crossed world where carol has withdrawn her invitation
// of bob to gamma, bob's deactivation, `nerve users deactivate` as
// bootstrap wires it, stops at a gate once it has locked acme's invitation
// to carol, x, the one it finds (lockedInvitations). Then z, a new
// invitation of carol's to his address in gamma, is committed; its id is
// smaller than x's, drawn before x's was, as a request draws it before its
// transaction waits. carol's deactivation, on each path, locks z, then
// waits for x. bob's deletes x, the invitation its lock returned, and
// commits; had it deleted every invitation to his address, it would have
// waited for z, which carol's holds: 40P01, one of them rolled back.
// carol's then leaves x out, deleted meanwhile, and deletes z. Both are
// done, and each invitation was deleted at the end of its deleter's
// membership: z by carol, x by bob.
func TestAnInvitationCreatedAfterADeactivationsLockIsLeftToTheOther(t *testing.T) {
	for _, p := range deactivationPaths {
		t.Run("carol by "+p.name, func(t *testing.T) {
			z := uuid.NewV7()
			w := newCrossedWorld(t)
			x := queryIDs(t, w.pool, "SELECT id FROM workspace_member_invites WHERE email = 'carol@example.com'")[0]
			y := queryIDs(t, w.pool, "SELECT id FROM workspace_member_invites WHERE email = 'bob@example.com'")[0]
			if status, body := call(t, w.contract, http.MethodDelete, w.base+"/api/v0/workspace-invitations/"+y.String(), w.tokens["carol"],
				""); status != http.StatusNoContent || z.Compare(x) >= 0 {
				t.Fatalf("carol's withdrawal of her invitation to bob = %d %s; z %s, x %s: want it done, and z's id the smaller", status, body,
					z, x)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			gated := deactivating(w.pool, identitypg.New(w.pool), lockedInvitations{workspacepg.New(w.pool), g})
			done := run(func() error {
				_, err := gated.ExecuteByEmail(ctx, "bob@example.com")
				return err
			})
			held(t, ctx, g, done, "bob's deactivation")
			if _, err := w.pool.Exec(pgtest.Soon(t), `INSERT INTO workspace_member_invites (id, workspace_id, email, role, created_by_id,
				updated_by_id) VALUES ($1, $2, 'bob@example.com', 15, $3, $3)`, z, w.workspace(t, "gamma"), w.ids["carol"]); err != nil {
				t.Fatal(err)
			}
			answer := p.sent(w, t, "carol")
			pgtest.WaitForLockWaitOn(t, w.pool, "workspace_member_invites", 5*time.Second)
			close(g.open)

			if err := result(t, ctx, done, "bob's deactivation"); err != nil {
				t.Errorf("bob's deactivation = %v; want it done", err)
			}
			if got := answer(); got != "" {
				t.Errorf("carol's deactivation = %q; want it done", got)
			}
			if got, want := w.crossed(t), "bob deactivated, carol deactivated, dave active, erin active; acme bob 20 ended, acme dave 15 ended, "+
				"gamma carol 20 ended, gamma erin 15 ended; no invitation"; got != want {
				t.Errorf("after both: %s; want %s", got, want)
			}
			var deleters string
			if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT string_agg(s.slug || ' ' || i.email || ' by ' || split_part(u.email, '@', 1) ||
				CASE WHEN i.deleted_at = m.updated_at AND i.updated_at = m.updated_at THEN ' at the end of the membership' ELSE ' at '
				|| i.deleted_at::text END, ', ' ORDER BY i.id)
				FROM workspace_member_invites i JOIN workspaces s ON s.id = i.workspace_id JOIN users u ON u.id = i.updated_by_id
				JOIN workspace_members m ON m.member_id = i.updated_by_id WHERE i.id = ANY ($1)`, []uuid.UUID{z, x}).Scan(&deleters); err != nil {
				t.Fatal(err)
			}
			if want := "gamma bob@example.com by carol at the end of the membership, acme carol@example.com by bob at the end of the membership"; deleters != want {
				t.Errorf("z and x by their deleters: %s; want %s", deleters, want)
			}
		})
	}
}
