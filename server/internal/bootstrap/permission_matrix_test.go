package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// The permission matrix (M3 design 9.2): each operation of the contract but
// the exempt modules' below, called over HTTP on the wired app and a real
// database by each kind of caller, its status and problem code asserted cell
// by cell, and where a row says so, what the answer holds. The data is
// prepared once: the accounts through the API, the workspaces and
// memberships through the workspace store, the deleted workspace through
// the API, and the state no store writes yet through SQL (prepareMatrix).
// The cells that only read share one copy of it, and each cell that writes
// gets a copy of its own (pgtest.NewDatabaseFrom), so no cell sees
// another's writes. Each module's rows are in a file of their own
// (permission_matrix_<module>_test.go): a phase that adds an operation adds
// its row there, and what the row needs prepared here and in
// permission_matrix_seeded_test.go.

// matrixExempt is what has no row, each entry for its reason (M3 design
// 9.2: every operation but the account-level and the public ones).
var matrixExempt = matrixExemptions{
	modules: []string{
		// Account-level (M2): each operation acts on the caller's own
		// account, sessions or tokens, and no workspace or project role
		// decides it.
		"identity",
		// Public: it describes this instance to anyone, with a token or
		// without.
		"instance",
	},
	public: map[string]string{
		// The link's token stands for a credential (M3 design 3.8).
		"getWorkspaceInvitation": "TestTheInvitationLinkAnswersEveryCallerAlike",
	},
	notTargets: map[string]string{
		"/api/v0/workspace-slugs/{slug}": "a slug asked about, not a workspace: the answer is the same for every caller",
		"/api/v0/workspace-invitations/{invitation_id}/accept": "account level: each column answers an invitation to its own " +
			"address (ownInvitation), or acme's newcomer's, whatever workspace its column targets",
		"/api/v0/workspace-invitations/{invitation_id}/decline": "account level, as accept",
	},
}

// matrixExemptions are the operations without a row. modules exempts every
// operation of a module: every other operation of the contract has a row,
// so a module that adds operations is in the matrix unless it is listed,
// and an entry that no operation carries is reported, so a misspelled one
// fails. public exempts one public operation of a module the matrix
// covers, by its operationId, naming the test that stands for its row: its
// route runs no authentication (httpserver's PublicOperations), so every
// column would call it as nobody and the cells could not tell the columns
// apart. The test calls it with every column's token and without one, and
// wants one answer. An operation that needs a token cannot be listed.
// notTargets are the paths whose parameters name nothing a column's cell
// must aim at its workspace, each with its reason: targetViolation passes
// them over, and reports any other parameter it does not know.
type matrixExemptions struct {
	modules    []string
	public     map[string]string // operationId → the test that stands for its row
	notTargets map[string]string // path → why its parameters are no column's target
}

// caller is a column: an account, and how it stands to the workspace a row
// targets.
type caller string

const (
	callerAdmin   caller = "admin"
	callerMember  caller = "member"
	callerGuest   caller = "guest"
	callerNever   caller = "never a member"
	callerRemoved caller = "removed"
	callerDeleted caller = "workspace deleted"
)

// workspaceColumns are the columns of the workspace level, in the order of
// 9.2's table.
var workspaceColumns = []caller{callerAdmin, callerMember, callerGuest, callerNever, callerRemoved, callerDeleted}

// workspaceOf is the slug of the workspace a column's cells target: the
// prepared workspace, or the deleted one its caller was the admin of.
func workspaceOf(c caller) string {
	if c == callerDeleted {
		return "gone"
	}
	return "acme"
}

// cell is an answer: the status and, for a problem, its code.
type cell struct {
	status int
	code   string
}

// String is the answer as a failure prints it: the status, then the code
// of a problem.
func (c cell) String() string {
	if c.code == "" {
		return fmt.Sprint(c.status)
	}
	return fmt.Sprintf("%d %s", c.status, c.code)
}

var (
	cellOK        = cell{status: http.StatusOK}
	cellCreated   = cell{status: http.StatusCreated}
	cellNoContent = cell{status: http.StatusNoContent}
	cellForbidden = cell{http.StatusForbidden, "forbidden"}
)

// matrixRow is an operation's row: the request each caller sends, which can
// name a row prepareMatrix seeded, and the answer each gets.
type matrixRow struct {
	op      string // operationId
	variant string // what sets the row apart from the operation's other rows
	write   bool   // each cell on a copy of its own
	config  func(*config.Config)
	request func(c caller, s seeded) (method, path, body string)
	cells   map[caller]cell
	// check, when set, runs on each answer that is not a problem and is its
	// cell's: what the answer holds for that caller, the seeded ids to
	// compare its ids with.
	check func(t *testing.T, c caller, s seeded, answer string)
}

func (r matrixRow) name() string {
	if r.variant == "" {
		return r.op
	}
	return r.op + ", " + r.variant
}

// every is the same answer in every workspace column.
func every(answer cell) map[caller]cell {
	cells := map[caller]cell{}
	for _, c := range workspaceColumns {
		cells[c] = answer
	}
	return cells
}

// sameRequest is the request of a row whose callers all send the same.
func sameRequest(method, path, body string) func(caller, seeded) (string, string, string) {
	return func(caller, seeded) (string, string, string) { return method, path, body }
}

// decodeAnswer decodes a cell's answer into v for a row's check.
func decodeAnswer(t *testing.T, answer string, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(answer), v); err != nil {
		t.Fatalf("the answer %s: %v", answer, err)
	}
}

// matrixRows are the rows, each module's from its file.
func matrixRows() []matrixRow {
	return slices.Concat(workspaceMatrixRows())
}

// matrixApps is how many cells may run an app of their own at once: each
// cell that writes, or whose row has a config, takes a slot while its app
// runs. Each app's pool opens up to testConfig's MaxConns (4) connections,
// besides the reads app's and pgtest's admin pool (4 each), against the
// container's max_connections of 100.
const matrixApps = 8

// matrixData is the prepared database, the signing key every app on a copy
// of it shares, each column's access token, and the seeded ids.
type matrixData struct {
	url     string
	keyFile string
	tokens  map[caller]string
	seeded  seeded
}

// config is the configuration of an app on the database at url.
func (d matrixData) config(t *testing.T, url string, change func(*config.Config)) config.Config {
	t.Helper()
	cfg := testConfig(t, url, false)
	cfg.Auth.JWT.PrivateKeyFile = d.keyFile
	if change != nil {
		change(&cfg)
	}
	return cfg
}

// prepareMatrix fills a database for the matrix. Through the API, an
// account for each column, registered for its token. Through the workspace
// store, the workspaces, memberships and invitations of matrixMemberships
// and matrixInvitations, with the ids newSeeded named, and acme's admin's
// display settings; other's admin and removed member are there so that a
// role read in the wrong workspace lets either into acme. Through the API,
// gone deleted by its admin, which soft-deletes its memberships with it.
// Through SQL, until P5's store replaces it, the removed member's
// membership of acme ended. Everything that connected to the database is
// closed when it returns, so that it can be copied. A -run that leaves out
// prepare fails here, not with a 401 in every cell.
func prepareMatrix(t *testing.T) matrixData {
	t.Helper()
	d := matrixData{url: pgtest.NewDatabase(t), keyFile: writeFile(t, testKeyPEM), tokens: map[caller]string{}, seeded: newSeeded()}
	prepared := t.Run("prepare", func(t *testing.T) {
		contract := apitest.Load(t)
		base := startApp(t, d.config(t, d.url, nil), migrations.FS())
		pool := openPool(t, d.url)
		ids := map[caller]uuid.UUID{}
		for _, c := range workspaceColumns {
			email := emailOf(c)
			d.tokens[c] = registerAccount(t, contract, base, email).AccessToken
			var id uuid.UUID
			if err := pool.QueryRow(context.Background(), "SELECT id FROM users WHERE email = $1", email).Scan(&id); err != nil {
				t.Fatal(err)
			}
			ids[c] = id
		}
		seed := matrixSeed{t: t, store: workspacepg.New(pool), ids: ids, now: time.Now(), workspaces: map[string]uuid.UUID{}}
		s := d.seeded.in(t)
		for _, m := range matrixMemberships {
			if _, created := seed.workspaces[m.slug]; !created {
				seed.workspace(s.workspace(m.slug), m.slug, m.c)
			}
			seed.join(s.membership(m.slug, m.c), m.slug, m.c, m.role)
		}
		for _, i := range matrixInvitations {
			seed.invite(s.invitation(i.slug, i.email), i.slug, i.email, i.role)
		}
		tabbed, three := "TABBED", 3
		seed.preferences("acme", callerAdmin, workspacedomain.PreferencesPatch{NavigationControl: &tabbed, NavigationProjectLimit: &three})
		// No store removes a member yet (P5), so SQL stands in until that
		// phase replaces it: it ends the removed member's membership of acme.
		seed.exec(pool, "UPDATE workspace_members SET is_active = false WHERE id = $1", s.membership("acme", callerRemoved))
		// The column's caller deletes gone as deleteWorkspace does it: its
		// membership and invitations go with the workspace row, so every
		// cell of the column is asked about a workspace deleted the one way
		// there is. A membership left active in a deleted workspace is
		// ActiveRole's store test (P1).
		if status, body := call(t, contract, http.MethodDelete, base+"/api/v0/workspaces/gone", d.tokens[callerDeleted], ""); status != http.StatusNoContent {
			t.Fatalf("deleting gone = %d %s", status, body)
		}
	})
	if !prepared {
		t.FailNow()
	}
	if len(d.tokens) != len(workspaceColumns) {
		t.Fatal("prepare did not run: a -run of some cells must select prepare too, e.g. -run 'TestPermissionMatrix/(prepare|getWorkspace)'")
	}
	return d
}

// Each cell of the matrix, in parallel: a reading cell on the reads' copy,
// a cell with an app of its own (a writing cell, or one whose row has a
// config) on its copy, at most matrixApps of those at once. Every answer a
// row's check is for is checked, and counted: a harness that skipped the
// checks would fail.
func TestPermissionMatrix(t *testing.T) {
	d := prepareMatrix(t)
	contract := apitest.Load(t)
	reads := startApp(t, d.config(t, pgtest.NewDatabaseFrom(t, d.url), nil), migrations.FS())
	apps := make(chan struct{}, matrixApps)
	var checked, toCheck atomic.Int64
	t.Cleanup(func() {
		if !t.Failed() && checked.Load() != toCheck.Load() {
			t.Errorf("%d answers checked, want %d", checked.Load(), toCheck.Load())
		}
	})
	for _, r := range matrixRows() {
		for _, c := range workspaceColumns {
			want, ok := r.cells[c]
			if !ok {
				continue // TestThePermissionMatrixCoversEveryOperation reports it
			}
			t.Run(r.name()+"/"+string(c), func(t *testing.T) {
				t.Parallel()
				// Counted in the cell: a -run of some cells expects only
				// their checks.
				if r.check != nil && want.code == "" {
					toCheck.Add(1)
				}
				base := reads
				if r.write || r.config != nil {
					apps <- struct{}{}
					// Registered before the app's: cleanups run last first,
					// so the slot is freed once the app is closed.
					t.Cleanup(func() { <-apps })
					base = startApp(t, d.config(t, pgtest.NewDatabaseFrom(t, d.url), r.config), migrations.FS())
				}
				method, path, body := r.request(c, d.seeded.in(t))
				status, answer := call(t, contract, method, base+path, d.tokens[c], body)
				got := cell{status: status}
				if status >= http.StatusBadRequest {
					got.code = problemCode(t, []byte(answer))
				}
				if got != want {
					t.Errorf("%s %s = %d %s, want %s", method, path, status, strings.TrimSpace(answer), want)
					return
				}
				if r.check != nil && got.code == "" {
					r.check(t, c, d.seeded.in(t), answer)
					checked.Add(1)
				}
			})
		}
	}
}
