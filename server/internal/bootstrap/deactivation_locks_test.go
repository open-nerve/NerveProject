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
// of his in id order, then every invitation it deletes, in id order, then
// his memberships, then every project of his in id order across the
// workspaces, then his memberships of them. alice has joined Ops and Lab;
// carol has made able, which bob joined, and delta, which he joined and she
// removed him from; bob has made kappa, where he is alone, and invited erin
// to it; then carol has invited his address to delta again; acme, Lab and
// kappa's invitation have been written again. So acme's row lies after
// beta's, able, made after both, comes first by slug, and Lab's row and
// memberships lie after Docs's: the order is the ids', not the rows' or the
// slugs' (a statement without its ORDER BY reaches beta or able, and Docs,
// first). kappa's invitation, of the workspace he leaves with no active
// member, has the smaller id but lies after delta's, to his address: one
// statement takes both, in id order, before either delete, though
// DeleteInvitationsTo, which writes delta's, runs before the fifth
// statement, which writes kappa's (the final review's I1). Other
// transactions hold, FOR SHARE, his account, beta, kappa's invitation,
// delta's invitation, his membership of beta, Lab and his membership of
// Docs, which the deactivation's statements wait for in turn; they let go
// one at a time, each step's probe naming the holder it waits behind, and
// lockOn reads each row's strongest lock then. The deactivation waits for
// his account, holding nothing; then for beta, holding his account and
// acme, the first workspace by id, and not able nor kappa, the last, nor
// any invitation; then for kappa's invitation, holding beta, able and kappa
// too, and gamma's, the invitation he declined, whose id is smaller; then
// for delta's, holding kappa's; then for his membership, holding delta's;
// then for Lab, holding his membership, Web, Ops and Solo, of acme, before
// Lab, of beta, by id, and not Docs, of acme, after it; then for his
// membership of Docs, holding Lab and Docs; each FOR NO KEY UPDATE, no
// stronger, no weaker; never gamma, where he is no member, nor delta, where
// his membership ended. Then it is done, at a moment no earlier than the
// release of delta's invitation, its last lock before the clock, nor of
// beta, his last workspace's (3.3).
func TestEachLockOfADeactivationIsItsStrength(t *testing.T) {
	for _, p := range deactivationPaths {
		t.Run(p.name, func(t *testing.T) {
			w := newDeactivationWorld(t)
			w.clears(t, w.ops, w.lab)
			able, delta := w.carolsWithBob(t, "able", false), w.carolsWithBob(t, "delta", true)
			kappa, toKappa := w.bobsAlone(t, "kappa")
			toDelta := invite(t, w.contract, w.base, w.tokens["carol"], "delta", "bob@example.com").id
			bob, acme, beta := w.ids["bob"], w.workspace(t, "acme"), w.workspace(t, "beta")
			inBeta := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id = $1 AND member_id = $2", beta, bob)[0]
			inDocs := projectMemberships(t, w.pool, bob, w.docs)[0]
			if w.bobsDeclined.Compare(toKappa) >= 0 || toKappa.Compare(toDelta) >= 0 {
				t.Fatalf("gamma's invitation %s, kappa's %s, delta's %s: want them in that id order", w.bobsDeclined, toKappa, toDelta)
			}
			w.writtenAgain(t, acme, beta, toKappa, toDelta)
			rows := []struct {
				name, from string
				id         uuid.UUID
			}{
				{"his account", "users", bob}, {"acme", "workspaces", acme}, {"beta", "workspaces", beta}, {"able", "workspaces", able},
				{"kappa", "workspaces", kappa}, {"gamma", "workspaces", w.gamma}, {"delta", "workspaces", delta},
				{"gamma's invitation", "workspace_member_invites", w.bobsDeclined}, {"kappa's invitation", "workspace_member_invites", toKappa},
				{"delta's invitation", "workspace_member_invites", toDelta}, {"his membership", "workspace_members", inBeta},
				{"Web", "projects", w.web}, {"Ops", "projects", w.ops}, {"Solo", "projects", w.solo}, {"Lab", "projects", w.lab},
				{"Docs", "projects", w.docs},
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
			holdsDeltas := holding(t, w.pool, "SELECT 1 FROM workspace_member_invites WHERE id = $1 FOR SHARE", toDelta)
			holdsKappas := holding(t, w.pool, "SELECT 1 FROM workspace_member_invites WHERE id = $1 FOR SHARE", toKappa)
			holdsBeta := holding(t, w.pool, "SELECT 1 FROM workspaces WHERE id = $1 FOR SHARE", beta)
			holdsAccount := holding(t, w.pool, "SELECT 1 FROM users WHERE id = $1 FOR SHARE", bob)
			shared := []string{"his account", "beta", "kappa's invitation", "delta's invitation", "his membership", "Lab"}
			answer := p.sent(w, t, "bob")
			released := map[pgx.Tx]time.Time{}
			var took []string
			// Each step lets go of one holder, of the row frees, and names
			// the holder the deactivation then waits behind, of the row
			// waitsFor, and the rows it has taken by then. The probe names
			// the holder (pgtest.WaitForLockWaitBehind): a wait for another
			// row of the same table, or one that ended with the holder the
			// step let go, does not satisfy it. A lock taken on a row held
			// FOR SHARE before its turn would hide behind the holder's, and
			// show only as the probe of the step that waits behind that
			// holder timing out.
			for _, step := range []struct {
				release, behind pgx.Tx
				frees, waitsFor string
				takes           []string
			}{
				{nil, holdsAccount, "", "his account", nil},
				{holdsAccount, holdsBeta, "his account", "beta", []string{"his account", "acme"}},
				{holdsBeta, holdsKappas, "beta", "kappa's invitation", []string{"beta", "able", "kappa", "gamma's invitation"}},
				{holdsKappas, holdsDeltas, "kappa's invitation", "delta's invitation", []string{"kappa's invitation"}},
				{holdsDeltas, holdsMembership, "delta's invitation", "his membership", []string{"delta's invitation"}},
				{holdsMembership, holdsLab, "his membership", "Lab", []string{"his membership", "Web", "Ops", "Solo"}},
				{holdsLab, holdsDocs, "Lab", "his membership of Docs", []string{"Lab", "Docs"}},
			} {
				if step.release != nil {
					released[step.release] = time.Now()
					if err := step.release.Rollback(context.Background()); err != nil {
						t.Fatal(err)
					}
					shared = slices.DeleteFunc(shared, func(name string) bool { return name == step.frees })
				}
				took = append(took, step.takes...)
				pgtest.WaitForLockWaitBehind(t, w.pool, step.behind.Conn().PgConn(), 5*time.Second)
				if got, want := locks(), state(took, shared); got != want {
					t.Errorf("the deactivation waiting for %s:\n%s\nwant\n%s", step.waitsFor, got, want)
				}
			}
			if err := holdsDocs.Rollback(context.Background()); err != nil {
				t.Fatal(err)
			}

			if got := answer(); got != "" {
				t.Fatalf("the deactivation = %q, want it done", got)
			}
			moment, _ := rowJSON(t, w.pool, "workspace_member_invites", w.bobsDeclined)["deleted_at"].(string)
			at, err := time.Parse(time.RFC3339Nano, moment)
			for _, last := range []struct {
				name   string
				holder pgx.Tx
			}{{"beta, his last workspace", holdsBeta}, {"delta's invitation, his last invitation", holdsDeltas}} {
				if err != nil || at.Before(released[last.holder].Truncate(time.Microsecond)) {
					t.Errorf("the deactivation's moment %q (%v); want one no earlier than the release of %s, %v", moment, err, last.name,
						released[last.holder])
				}
			}
		})
	}
}

// writtenAgain writes acme's row, Lab's, Lab's memberships and kappa's
// invitation again, each update one row of Lab's three memberships or the
// row named, and checks the rows then lie, in each table, out of their id
// order: beta's before acme's, Docs's before Lab's, bob's membership of
// Docs before his of Lab, and delta's invitation, to his address, before
// kappa's, to erin's. A plan that reads the rows in the table's order meets
// the larger id first.
func (w deactivationWorld) writtenAgain(t *testing.T, acme, beta, toKappa, toDelta uuid.UUID) {
	t.Helper()
	for _, again := range []struct {
		sql  string
		id   uuid.UUID
		rows int64
	}{
		{"UPDATE workspaces SET updated_at = updated_at WHERE id = $1", acme, 1},
		{"UPDATE projects SET updated_at = updated_at WHERE id = $1", w.lab, 1},
		{"UPDATE project_members SET updated_at = updated_at WHERE project_id = $1", w.lab, 3},
		{"UPDATE workspace_member_invites SET updated_at = updated_at WHERE id = $1", toKappa, 1},
	} {
		if tag, err := w.pool.Exec(pgtest.Soon(t), again.sql, again.id); err != nil || tag.RowsAffected() != again.rows {
			t.Fatalf("%s: %v, %v; want %d rows", again.sql, tag, err, again.rows)
		}
	}
	var heap string
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT concat_ws('; ',
		(SELECT string_agg(slug, ' ' ORDER BY ctid) FROM workspaces WHERE id = ANY ($1)),
		(SELECT string_agg(name, ' ' ORDER BY ctid) FROM projects WHERE id = ANY ($2)),
		(SELECT string_agg(p.name, ' ' ORDER BY m.ctid) FROM project_members m JOIN projects p ON p.id = m.project_id
			WHERE m.member_id = $3 AND m.project_id = ANY ($2)),
		(SELECT string_agg(email, ' ' ORDER BY ctid) FROM workspace_member_invites WHERE id = ANY ($4)))`,
		[]uuid.UUID{acme, beta}, []uuid.UUID{w.lab, w.docs}, w.ids["bob"], []uuid.UUID{toKappa, toDelta}).Scan(&heap); err != nil {
		t.Fatal(err)
	}
	if want := "beta acme; Docs Lab; Docs Lab; bob@example.com erin@example.com"; heap != want {
		t.Fatalf("the rows in the tables' order: %s; want %s", heap, want)
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
