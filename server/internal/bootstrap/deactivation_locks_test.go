package bootstrap

import (
	"context"
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// Each lock of a deactivation is taken in the lock table's order, at its
// strength, on both paths (M3 design 3.6's lock table, convention 1,
// convention 6, the global order): his account first, then every workspace
// of his in id order, then the invitations to his address, the pending
// ones of a workspace he leaves with no active member, and his
// memberships, then every project of his in id order across the
// workspaces, then his memberships of them. alice has joined Ops and Lab;
// carol has made able, which bob joined, and delta, which he joined and
// she removed him from; bob has made kappa, where he is alone, and invited
// erin to it; acme and Lab have been written again. So acme's row lies
// after beta's, able, made after both, comes first by slug, and Lab's row
// and memberships lie after Docs's: the order is the ids', not the rows' or
// the slugs' (a statement without its ORDER BY reaches beta or able, and
// Docs, first). Other transactions hold, FOR SHARE, his account, beta, the
// invitation to gamma he declined, kappa's invitation, his membership of
// beta, Lab and his membership of Docs, which the deactivation's
// statements wait for in turn; they let go one at a time, and lockOn reads
// each row's strongest lock then. The deactivation waits for his account,
// holding nothing; then for beta, holding his account and acme, the first
// workspace by id, and not able nor kappa, the last; then for the
// invitation he declined, holding beta, able and kappa too; then for
// kappa's invitation, holding the one he declined; then for his
// membership, holding kappa's invitation; then for Lab, holding his
// membership, Web, Ops and Solo, of acme, before Lab, of beta, by id, and
// not Docs, of acme, after it; then for his membership of Docs, holding
// Lab and Docs; each FOR NO KEY UPDATE, no stronger, no weaker; never
// gamma, where he is no member, nor delta, where his membership ended.
// Then it is done, at a moment no earlier than beta's release: it read the
// clock under its last workspace's lock (3.3).
func TestEachLockOfADeactivationIsItsStrength(t *testing.T) {
	for _, p := range deactivationPaths {
		t.Run(p.name, func(t *testing.T) {
			w := newDeactivationWorld(t)
			w.clears(t, w.ops, w.lab)
			able, delta := w.carolsWithBob(t, "able", false), w.carolsWithBob(t, "delta", true)
			kappa, toKappa := w.bobsAlone(t, "kappa")
			bob, acme, beta := w.ids["bob"], w.workspace(t, "acme"), w.workspace(t, "beta")
			inBeta := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id = $1 AND member_id = $2", beta, bob)[0]
			inDocs := projectMemberships(t, w.pool, bob, w.docs)[0]
			for _, again := range []struct {
				sql string
				id  uuid.UUID
			}{
				{"UPDATE workspaces SET updated_at = updated_at WHERE id = $1", acme},
				{"UPDATE projects SET updated_at = updated_at WHERE id = $1", w.lab},
				{"UPDATE project_members SET updated_at = updated_at WHERE project_id = $1", w.lab},
			} {
				if _, err := w.pool.Exec(soon(t), again.sql, again.id); err != nil {
					t.Fatal(err)
				}
			}
			rows := []struct {
				name, from string
				id         uuid.UUID
			}{
				{"his account", "users", bob}, {"acme", "workspaces", acme}, {"beta", "workspaces", beta}, {"able", "workspaces", able},
				{"kappa", "workspaces", kappa}, {"gamma", "workspaces", w.gamma}, {"delta", "workspaces", delta},
				{"the invitation", "workspace_member_invites", w.bobsDeclined}, {"kappa's invitation", "workspace_member_invites", toKappa},
				{"his membership", "workspace_members", inBeta}, {"Web", "projects", w.web}, {"Ops", "projects", w.ops},
				{"Solo", "projects", w.solo}, {"Lab", "projects", w.lab}, {"Docs", "projects", w.docs},
			}
			locks := func() string {
				t.Helper()
				held := make([]string, len(rows))
				for i, r := range rows {
					held[i] = r.name + " " + lockOn(t, w.pool, r.from+" WHERE id = $1", r.id)
				}
				return strings.Join(held, ", ")
			}
			// state is the locks of rows once the deactivation has taken
			// took, FOR NO KEY UPDATE, while the holders of shared still hold
			// theirs, FOR SHARE: no lock on any other.
			state := func(took, shared []string) string {
				held := make([]string, len(rows))
				for i, r := range rows {
					switch {
					case slices.Contains(took, r.name):
						held[i] = r.name + " FOR NO KEY UPDATE"
					case slices.Contains(shared, r.name):
						held[i] = r.name + " FOR SHARE"
					default:
						held[i] = r.name + " no lock"
					}
				}
				return strings.Join(held, ", ")
			}
			holdsDocs := holding(t, w.pool, "SELECT 1 FROM project_members WHERE id = $1 FOR SHARE", inDocs)
			holdsLab := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR SHARE", w.lab)
			holdsMembership := holding(t, w.pool, "SELECT 1 FROM workspace_members WHERE id = $1 FOR SHARE", inBeta)
			holdsKappas := holding(t, w.pool, "SELECT 1 FROM workspace_member_invites WHERE id = $1 FOR SHARE", toKappa)
			holdsInvitation := holding(t, w.pool, "SELECT 1 FROM workspace_member_invites WHERE id = $1 FOR SHARE", w.bobsDeclined)
			holdsBeta := holding(t, w.pool, "SELECT 1 FROM workspaces WHERE id = $1 FOR SHARE", beta)
			holdsAccount := holding(t, w.pool, "SELECT 1 FROM users WHERE id = $1 FOR SHARE", bob)
			shared := []string{"his account", "beta", "the invitation", "kappa's invitation", "his membership", "Lab"}
			answer := p.sent(w, t, "bob")
			var released time.Time
			var took []string
			// Each step lets go of one holder, of the row frees, and names
			// the table the deactivation then waits on, and the rows it has
			// taken by then. A lock it took on a row held FOR SHARE before
			// its turn would hide behind the holder's, and show only as the
			// probe of the step that waits on that row's table timing out
			// (pgtest.WaitForLockWaitOn).
			for _, step := range []struct {
				release pgx.Tx
				frees   string
				waitsOn string
				takes   []string
			}{
				{nil, "", "users", nil},
				{holdsAccount, "his account", "workspaces", []string{"his account", "acme"}},
				{holdsBeta, "beta", "workspace_member_invites", []string{"beta", "able", "kappa"}},
				{holdsInvitation, "the invitation", "workspace_member_invites", []string{"the invitation"}},
				{holdsKappas, "kappa's invitation", "workspace_members", []string{"kappa's invitation"}},
				{holdsMembership, "his membership", "projects", []string{"his membership", "Web", "Ops", "Solo"}},
				{holdsLab, "Lab", "project_members", []string{"Lab", "Docs"}},
			} {
				if step.release != nil {
					if step.release == holdsBeta {
						released = time.Now()
					}
					if err := step.release.Rollback(context.Background()); err != nil {
						t.Fatal(err)
					}
					shared = slices.DeleteFunc(shared, func(name string) bool { return name == step.frees })
				}
				took = append(took, step.takes...)
				pgtest.WaitForLockWaitOn(t, w.pool, step.waitsOn, 5*time.Second)
				if got, want := locks(), state(took, shared); got != want {
					t.Errorf("the deactivation waiting on %s:\n%s\nwant\n%s", step.waitsOn, got, want)
				}
			}
			if err := holdsDocs.Rollback(context.Background()); err != nil {
				t.Fatal(err)
			}

			if got := answer(); got != "" {
				t.Fatalf("the deactivation = %q, want it done", got)
			}
			moment, _ := rowJSON(t, w.pool, "workspace_member_invites", w.bobsDeclined)["deleted_at"].(string)
			if at, err := time.Parse(time.RFC3339Nano, moment); err != nil || at.Before(released.Truncate(time.Microsecond)) {
				t.Errorf("the deactivation's moment %q (%v); want one no earlier than beta's release, %v", moment, err, released)
			}
		})
	}
}

// bobsAlone is bob's new workspace slug, where he is the only member, and
// his pending invitation of erin to it: their ids.
func (w deactivationWorld) bobsAlone(t *testing.T, slug string) (workspace, invitation uuid.UUID) {
	t.Helper()
	if status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/workspaces", w.tokens["bob"],
		`{"name":"`+slug+`","slug":"`+slug+`"}`); status != http.StatusCreated {
		t.Fatalf("creating %s = %d %s", slug, status, body)
	}
	return w.workspace(t, slug), invite(t, w.contract, w.base, w.tokens["bob"], slug, "erin@example.com").id
}

// carolsWithBob is carol's new workspace slug, which bob joined by
// accepting her invitation, and, when removed, she then removed him from:
// its id.
func (w deactivationWorld) carolsWithBob(t *testing.T, slug string, removed bool) uuid.UUID {
	t.Helper()
	if status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/workspaces", w.tokens["carol"],
		`{"name":"`+slug+`","slug":"`+slug+`"}`); status != http.StatusCreated {
		t.Fatalf("creating %s = %d %s", slug, status, body)
	}
	id := w.workspace(t, slug)
	answerInvitation(t, w.contract, w.base, w.tokens["bob"], "accept", invite(t, w.contract, w.base, w.tokens["carol"], slug, "bob@example.com"),
		http.StatusOK)
	if removed {
		bobs := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id = $1 AND member_id = $2", id, w.ids["bob"])[0]
		if status, body := call(t, w.contract, http.MethodDelete, w.base+"/api/v0/workspace-members/"+bobs.String(), w.tokens["carol"],
			""); status != http.StatusNoContent {
			t.Fatalf("carol's removal of bob from %s = %d %s", slug, status, body)
		}
	}
	return id
}

// The command has no deadline of its own (M3 design 3.9, README): while
// bob's membership of Lab is held it waits, still after twice the request
// timeout its configuration gives the server, having written his account,
// his profile, his sessions, the invitations to his address and his
// memberships of acme and beta; and an interruption, its context's end as
// SIGINT or SIGTERM ends it (cmd/nerve), rolls it all back: it fails, and
// no row of any table changes. Run again once the row is free, it
// deactivates bob.
func TestAnInterruptedDeactivationChangesNothing(t *testing.T) {
	w := newDeactivationWorld(t)
	w.clears(t, w.ops, w.lab)
	before := tableRows(t, w.pool, riversOwn)
	holdsLab := holding(t, w.pool, "SELECT 1 FROM project_members WHERE id = $1 FOR SHARE", projectMemberships(t, w.pool, w.ids["bob"], w.lab)[0])
	ctx, interrupt := context.WithCancel(context.Background())
	defer interrupt()
	cfg := testConfig(t, w.url, false)
	cfg.Server.RequestTimeout = 200 * time.Millisecond
	done := deactivatingUser(ctx, cfg, "bob@example.com")
	pgtest.WaitForLockWaitOn(t, w.pool, "project_members", 5*time.Second)
	select {
	case run := <-done:
		t.Fatalf("the deactivation ended while bob's membership of Lab was held = %q, %v; want it waiting until interrupted", run.out, run.err)
	case <-time.After(2 * cfg.Server.RequestTimeout):
	}
	interrupt()

	if run := receiveWithin(t, done, 10*time.Second, "the end of nerve users deactivate"); run.err == nil || run.out != "" {
		t.Errorf("the deactivation interrupted = %q, %v; want no line and an error", run.out, run.err)
	}
	if err := holdsLab.Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
	if after := tableRows(t, w.pool, riversOwn); !maps.Equal(after, before) {
		t.Errorf("the tables after the interrupted deactivation changed:\n%v\nwant them as they were:\n%v", after, before)
	}
	if got := byCommand.deactivate(w, t, "bob"); got != "" {
		t.Errorf("the deactivation again = %q, want it done", got)
	}
}

// The deactivation runs every statement on its transaction's connection
// (M3 design 3.6 convention 2, 3.9): identity's, the Deactivator's and the
// cascade's. The command's composition runs on a pool of one connection: a
// statement sent through the pool rather than the transaction would wait
// for a second connection. On the command's context it waits until that
// context ends, after 5 seconds, and the command fails; on a context that
// never ends it waits for good, and the test fails at its 10-second wait
// for the command's end, that goroutine left until the test binary ends.
func TestTheDeactivationRunsOnItsTransactionsConnection(t *testing.T) {
	w := newDeactivationWorld(t)
	w.clears(t, w.ops, w.lab)
	cfg := testConfig(t, w.url, false)
	cfg.Database.MaxConns = 1
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	want := "deactivated bob@example.com: revoked 1 sessions and ended its memberships; to bring it back, run nerve users activate, then " +
		"nerve workspaces reactivate-member in each workspace\n"
	if run := receiveWithin(t, deactivatingUser(ctx, cfg, "bob@example.com"), 10*time.Second, "the end of nerve users deactivate"); run.out != want ||
		run.err != nil {
		t.Errorf("the deactivation on a pool of one connection = %q, %v; want %q", run.out, run.err, want)
	}
}
