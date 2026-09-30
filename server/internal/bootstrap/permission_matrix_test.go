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

	"github.com/jackc/pgx/v5/pgxpool"

	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
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
// its row there, and what the row needs prepared here.

// matrixExempt are the modules whose operations have no row, each for its
// reason. Every other operation of the contract has one, so a module that
// adds operations is in the matrix unless it is added here (M3 design 9.2:
// every operation but the account-level and the public ones). An entry that
// no operation carries is reported, so a misspelled one fails. P3's public
// getWorkspaceInvitation is tagged workspace: P3 gives the matrix a column
// for a caller without a token, or an exemption by operation. This list
// exempts modules, and workspace on it would exempt all of its operations
// (spec P1 3 item 10).
var matrixExempt = []string{
	// Account-level (M2): each operation acts on the caller's own account,
	// sessions or tokens, and no workspace or project role decides it.
	"identity",
	// Public: it describes this instance to anyone, with a token or without.
	"instance",
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
	// cell's: what the answer holds for that caller.
	check func(t *testing.T, c caller, answer string)
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

// seeded are the ids of the rows prepareMatrix seeded that a request can
// name: each membership, by the workspace's slug and the column.
type seeded struct {
	memberships map[string]uuid.UUID
}

// membership is the id of c's membership of the workspace slug; the zero id
// when there is none, which no row has.
func (s seeded) membership(slug string, c caller) uuid.UUID {
	return s.memberships[slug+"/"+string(c)]
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

// matrixApps is how many writing cells may run an app at once. Each app's
// pool opens up to testConfig's MaxConns (4) connections, besides the reads
// app's and pgtest's admin pool (4 each), against the container's
// max_connections of 100.
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
// store, the workspace acme with its admin, member, guest and the member
// later removed, and the admin's display settings there; the workspace gone
// with its admin and the member; and the workspace other, whose admin was
// never a member of acme and where the removed member is still active, so
// that a role read in the wrong workspace lets either into acme. Through the
// API, gone deleted by its admin, which soft-deletes its memberships with
// it. Through SQL, until P5's store replaces it, the removed member's
// membership of acme ended. Everything that connected to the database is
// closed when it returns, so that it can be copied.
func prepareMatrix(t *testing.T) matrixData {
	t.Helper()
	d := matrixData{url: pgtest.NewDatabase(t), keyFile: writeFile(t, testKeyPEM), tokens: map[caller]string{}}
	prepared := t.Run("prepare", func(t *testing.T) {
		contract := apitest.Load(t)
		base := startApp(t, d.config(t, d.url, nil), migrations.FS())
		pool := openPool(t, d.url)
		ids := map[caller]uuid.UUID{}
		for _, c := range workspaceColumns {
			email := strings.ReplaceAll(string(c), " ", "-") + "@example.com"
			d.tokens[c] = registerAccount(t, contract, base, email).AccessToken
			var id uuid.UUID
			if err := pool.QueryRow(context.Background(), "SELECT id FROM users WHERE email = $1", email).Scan(&id); err != nil {
				t.Fatal(err)
			}
			ids[c] = id
		}
		seed := matrixSeed{t: t, store: workspacepg.New(pool), ids: ids, now: time.Now(),
			workspaces: map[string]uuid.UUID{}, memberships: map[string]uuid.UUID{}}
		seed.workspace("acme", callerAdmin)
		seed.join("acme", callerAdmin, shared.RoleAdmin)
		seed.join("acme", callerMember, shared.RoleMember)
		seed.join("acme", callerGuest, shared.RoleGuest)
		seed.join("acme", callerRemoved, shared.RoleMember)
		tabbed, three := "TABBED", 3
		seed.preferences("acme", callerAdmin, workspacedomain.PreferencesPatch{NavigationControl: &tabbed, NavigationProjectLimit: &three})
		seed.workspace("gone", callerDeleted)
		seed.join("gone", callerDeleted, shared.RoleAdmin)
		seed.join("gone", callerMember, shared.RoleMember)
		seed.workspace("other", callerNever)
		seed.join("other", callerNever, shared.RoleAdmin)
		seed.join("other", callerRemoved, shared.RoleMember)
		d.seeded = seeded{memberships: seed.memberships}
		// No store removes a member yet (P5), so SQL stands in until that
		// phase replaces it: it ends the removed member's membership of acme.
		seed.exec(pool, "UPDATE workspace_members SET is_active = false WHERE id = $1", d.seeded.membership("acme", callerRemoved))
		// The column's caller deletes gone as deleteWorkspace does it: its
		// membership goes with the workspace row, so every cell of the column
		// is asked about a workspace deleted the one way there is. A
		// membership left active in a deleted workspace is ActiveRole's
		// store test (P1).
		if status, body := call(t, contract, http.MethodDelete, base+"/api/v0/workspaces/gone", d.tokens[callerDeleted], ""); status != http.StatusNoContent {
			t.Fatalf("deleting gone = %d %s", status, body)
		}
	})
	if !prepared {
		t.FailNow()
	}
	return d
}

// matrixSeed writes the prepared workspaces, memberships and settings
// through the workspace store, and keeps their ids by slug (and column);
// exec runs the SQL that stands in for the store P5 adds.
type matrixSeed struct {
	t           *testing.T
	store       *workspacepg.Store
	ids         map[caller]uuid.UUID
	now         time.Time
	workspaces  map[string]uuid.UUID // by slug
	memberships map[string]uuid.UUID // by slug/column, as seeded.membership reads them
}

func (s matrixSeed) workspace(slug string, admin caller) {
	s.t.Helper()
	w, err := s.store.CreateWorkspace(context.Background(), workspaceapp.WorkspaceRow{
		ID: uuid.NewV7(), Name: slug, Slug: slug, Timezone: "UTC", CreatedBy: s.ids[admin], Now: s.now,
	})
	if err != nil {
		s.t.Fatal(err)
	}
	s.workspaces[slug] = w.ID
}

func (s matrixSeed) join(slug string, c caller, role shared.Role) {
	s.t.Helper()
	id := uuid.NewV7()
	if err := s.store.CreateMember(context.Background(), workspaceapp.MemberRow{
		ID: id, WorkspaceID: s.workspaces[slug], MemberID: s.ids[c], Role: role, CreatedBy: s.ids[c], Now: s.now,
	}); err != nil {
		s.t.Fatal(err)
	}
	s.memberships[slug+"/"+string(c)] = id
}

func (s matrixSeed) preferences(slug string, c caller, p workspacedomain.PreferencesPatch) {
	s.t.Helper()
	if _, err := s.store.UpsertPreferences(context.Background(), workspaceapp.PreferencesRow{
		ID: uuid.NewV7(), WorkspaceID: s.workspaces[slug], UserID: s.ids[c], Patch: p, Now: s.now,
	}); err != nil {
		s.t.Fatal(err)
	}
}

func (s matrixSeed) exec(pool *pgxpool.Pool, sql string, args ...any) {
	s.t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		s.t.Fatal(err)
	}
}

// Each cell of the matrix, the writing ones in parallel on their copies, at
// most matrixApps of them with an app at once. Every answer a row's check is
// for is checked, and counted: a harness that skipped the checks would fail.
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
				method, path, body := r.request(c, d.seeded)
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
					r.check(t, c, answer)
					checked.Add(1)
				}
			})
		}
	}
}
