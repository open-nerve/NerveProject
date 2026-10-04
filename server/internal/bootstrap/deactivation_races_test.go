package bootstrap

import (
	"context"
	"net/http"
	"testing"
	"time"
	"uuid"

	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// A deactivation against what another transaction changes meanwhile, on
// both paths (M3 design 3.6's lock table, convention 6, 3.9): what it
// finds once it has a lock, and whose each ending stays.

// endings is every membership of bob's and every invitation to his
// addresses, old and new, but the ones he accepted, which their acceptance
// deleted, one line each, in byte order: the project's name
// or the workspace's slug, or "invitation of <address> to <slug>"; then
// "active", "pending" or "declined" while it stands, else "ended by" or
// "deleted by" and who wrote it last.
func (w deactivationWorld) endings(t *testing.T) string {
	t.Helper()
	var got string
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT string_agg(place || ' ' || CASE WHEN deleted THEN 'deleted by ' || by
			WHEN active THEN standing ELSE 'ended by ' || by END, '; ' ORDER BY place COLLATE "C") FROM (
		SELECT s.slug AS place, m.deleted_at IS NOT NULL AS deleted, m.is_active AS active, 'active' AS standing, m.updated_by_id AS writer
			FROM workspace_members m JOIN workspaces s ON s.id = m.workspace_id WHERE m.member_id = $1
		UNION ALL SELECT p.name, m.deleted_at IS NOT NULL, m.is_active, 'active', m.updated_by_id
			FROM project_members m JOIN projects p ON p.id = m.project_id WHERE m.member_id = $1
		UNION ALL SELECT 'invitation of ' || i.email || ' to ' || s.slug, i.deleted_at IS NOT NULL, true,
			CASE WHEN i.responded_at IS NULL THEN 'pending' ELSE 'declined' END, i.updated_by_id
			FROM workspace_member_invites i JOIN workspaces s ON s.id = i.workspace_id WHERE i.email IN ('bob@example.com', 'robert@example.com')
				AND NOT i.accepted) r
		LEFT JOIN LATERAL (SELECT split_part(u.email, '@', 1) AS by FROM users u WHERE u.id = r.writer) b ON true`, w.ids["bob"]).
		Scan(&got); err != nil {
		t.Fatal(err)
	}
	return got
}

// bobsStanding is endings once alice has joined Ops and Lab, before any
// ending.
const bobsStanding = "Docs active; Lab active; Ops active; Solo active; Web active; acme active; beta active; " +
	"invitation of bob@example.com to acme pending; invitation of bob@example.com to beta pending; " +
	"invitation of bob@example.com to gamma declined"

// A deactivation that waits for a workspace's row reads, once it has it,
// what alice's write committed meanwhile, and goes through; each row she
// wrote stays hers (3.9; review spike 15). Once she has joined Ops and Lab,
// her write, through the API, holds acme's row or beta's and waits for a
// row another transaction holds FOR SHARE; bob's deactivation waits for
// that workspace's row: neither writes a row of workspaces before then, so
// only that lock's wait satisfies the probe. Then the holder lets go, her
// write is done, and so is the deactivation.
//   - acme deleted: its deletion holds acme FOR NO KEY UPDATE and waits for
//     the invitation to bob's address in acme. The deactivation leaves acme
//     out, as deleted while its lock waited, no 404 and no failure, and
//     ends the rest; his memberships of acme and of its projects and the
//     invitation to acme are as the deletion left them.
//   - Docs deleted: its deletion holds acme FOR SHARE and waits for Docs's
//     row. The deactivation finds Docs deleted once it has acme: his
//     membership of Docs is as the deletion left it.
//   - bob removed from beta: the removal holds beta FOR NO KEY UPDATE, has
//     ended his membership of beta and deleted the invitation to beta, and
//     waits for his membership of Lab. The deactivation has acme, waits for
//     beta, then finds those ended: the removal's endings stay alice's.
func TestADeactivationFindsWhatChangedMeanwhile(t *testing.T) {
	for _, tt := range []struct {
		name            string
		write           func(w deactivationWorld, t *testing.T) (path string, held uuid.UUID)
		waitsOn, ending string
	}{
		{"acme deleted", func(w deactivationWorld, t *testing.T) (string, uuid.UUID) {
			return "/api/v0/workspaces/acme", w.bobsInvitation
		}, "workspace_member_invites", "Docs deleted by alice; Lab ended by bob; Ops deleted by alice; Solo deleted by alice; " +
			"Web deleted by alice; acme deleted by alice; beta ended by bob; invitation of bob@example.com to acme deleted by alice; " +
			"invitation of bob@example.com to beta deleted by bob; invitation of bob@example.com to gamma deleted by bob"},
		{"Docs deleted", func(w deactivationWorld, t *testing.T) (string, uuid.UUID) {
			return "/api/v0/projects/" + w.docs.String(), w.docs
		}, "projects", "Docs deleted by alice; Lab ended by bob; Ops ended by bob; Solo ended by bob; Web ended by bob; acme ended by bob; " +
			"beta ended by bob; invitation of bob@example.com to acme deleted by bob; invitation of bob@example.com to beta deleted by bob; " +
			"invitation of bob@example.com to gamma deleted by bob"},
		{"bob removed from beta", func(w deactivationWorld, t *testing.T) (string, uuid.UUID) {
			beta := queryIDs(t, w.pool, `SELECT m.id FROM workspace_members m JOIN workspaces s ON s.id = m.workspace_id
				WHERE s.slug = 'beta' AND m.member_id = $1`, w.ids["bob"])[0]
			return "/api/v0/workspace-members/" + beta.String(), projectMemberships(t, w.pool, w.ids["bob"], w.lab)[0]
		}, "project_members", "Docs ended by bob; Lab ended by alice; Ops ended by bob; Solo ended by bob; Web ended by bob; acme ended by bob; " +
			"beta ended by alice; invitation of bob@example.com to acme deleted by bob; invitation of bob@example.com to beta deleted by alice; " +
			"invitation of bob@example.com to gamma deleted by bob"},
	} {
		for _, p := range deactivationPaths {
			t.Run(tt.name+", "+p.name, func(t *testing.T) {
				w := newDeactivationWorld(t)
				w.clears(t, w.ops, w.lab)
				if got := w.endings(t); got != bobsStanding {
					t.Fatalf("bob's memberships and invitations before: %s; want %s", got, bobsStanding)
				}
				path, row := tt.write(w, t)
				holder := holding(t, w.pool, "SELECT 1 FROM "+tt.waitsOn+" WHERE id = $1 FOR SHARE", row)
				req := newRequest(t, http.MethodDelete, w.base+path, w.tokens["alice"], nil)
				w.contract.CheckRequest(t, req)
				written := sendInBackground(req)
				pgtest.WaitForLockWaitOn(t, w.pool, tt.waitsOn, 5*time.Second)
				answer := p.sent(w, t, "bob")
				pgtest.WaitForLockWaitOn(t, w.pool, "workspaces", 5*time.Second)
				if err := holder.Rollback(context.Background()); err != nil {
					t.Fatal(err)
				}

				a := receiveWithin(t, written, 10*time.Second, "the answer to alice's write")
				if a.err != nil {
					t.Fatal(a.err)
				}
				w.contract.CheckResponse(t, req, a.res)
				if a.res.StatusCode != http.StatusNoContent {
					t.Errorf("alice's write = %d %s, want 204", a.res.StatusCode, a.body)
				}
				if got := answer(); got != "" {
					t.Errorf("the deactivation = %q, want it done", got)
				}
				if got := w.endings(t); got != tt.ending {
					t.Errorf("bob's memberships and invitations after both:\n%s\nwant\n%s", got, tt.ending)
				}
			})
		}
	}
}

// The deactivation deletes the invitations to the address it reads under
// his account's lock (M3 design 3.6 convention 1, 3.9; review M5): the
// change of bob's address to robert@example.com, `nerve users set-email`,
// holds his account row, its address written, at its gate, before it
// revokes his sessions; the deactivation waits for that row, and the
// change commits. alice has invited robert@example.com to acme and beta,
// and joined Ops and Lab.
//   - deactivateMe, with his personal access token, which the change
//     leaves: it reads his new address under the lock and deletes the
//     invitations to it; those to bob@example.com, an address no account
//     has now, stay as they were; his memberships end.
//   - `nerve users deactivate --email bob@example.com`: it locks the account
//     by that address and finds none once the change has committed:
//     identity.account_not_found, and nothing of his changes.
func TestADeactivationReadsTheAddressUnderItsLock(t *testing.T) {
	for _, tt := range []struct {
		path            deactivationPath
		answer, endings string
	}{
		{byAPI, "", "Docs ended by robert; Lab ended by robert; Ops ended by robert; Solo ended by robert; Web ended by robert; " +
			"acme ended by robert; beta ended by robert; invitation of bob@example.com to acme pending; " +
			"invitation of bob@example.com to beta pending; invitation of bob@example.com to gamma declined; " +
			"invitation of robert@example.com to acme deleted by robert; invitation of robert@example.com to beta deleted by robert"},
		{byCommand, "identity.account_not_found: No account has this e-mail address.", "Docs active; Lab active; Ops active; " +
			"Solo active; Web active; acme active; beta active; invitation of bob@example.com to acme pending; " +
			"invitation of bob@example.com to beta pending; invitation of bob@example.com to gamma declined; " +
			"invitation of robert@example.com to acme pending; invitation of robert@example.com to beta pending"},
	} {
		t.Run(tt.path.name, func(t *testing.T) {
			w := newDeactivationWorld(t)
			w.clears(t, w.ops, w.lab)
			for _, slug := range []string{"acme", "beta"} {
				invite(t, w.contract, w.base, w.tokens["alice"], slug, "robert@example.com")
			}
			w.tokens["bob"] = createPAT(t, w.contract, w.base, w.tokens["bob"]).Token
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			changed := run(func() error { return setBobsEmail(ctx, w.pool, gatedSessions{identitypg.New(w.pool), g}) })
			held(t, ctx, g, changed, "the change of address")
			answer := tt.path.sent(w, t, "bob")
			pgtest.WaitForLockWaitOn(t, w.pool, "users", 5*time.Second)
			close(g.open)

			if err := result(t, ctx, changed, "the change of address"); err != nil {
				t.Fatalf("the change of address = %v, want it done", err)
			}
			if got := answer(); got != tt.answer {
				t.Errorf("the deactivation = %q, want %q", got, tt.answer)
			}
			if got := w.endings(t); got != tt.endings {
				t.Errorf("bob's memberships and invitations after both:\n%s\nwant\n%s", got, tt.endings)
			}
		})
	}
}
