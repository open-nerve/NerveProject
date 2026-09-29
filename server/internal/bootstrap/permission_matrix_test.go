package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// The permission matrix (M3 design 9.2): each operation of the modules
// below, called over HTTP on the wired app and a real database by each kind
// of caller, its status and problem code asserted cell by cell. The data is
// prepared once: the accounts through the API, the workspaces and
// memberships through the workspace store, and the two states no store
// writes yet through SQL (prepareMatrix). The cells that only read share
// one copy of it, and each cell that writes gets a copy of its own
// (pgtest.NewDatabaseFrom), so no cell sees another's writes. A phase that
// adds an operation adds its row, and what the row needs prepared.

// matrixModules are the modules whose every operation has a row: their
// operations are authorized in a workspace, or act on the caller's
// workspaces. project adds itself with its operations.
var matrixModules = []string{"workspace"}

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
	cellOK                = cell{status: http.StatusOK}
	cellCreated           = cell{status: http.StatusCreated}
	cellWorkspaceNotFound = cell{http.StatusNotFound, "workspace.not_found"}
	cellCreationDisabled  = cell{http.StatusForbidden, "workspace.creation_disabled"}
)

// matrixRow is an operation's row: the request each caller sends and the
// answer each gets.
type matrixRow struct {
	op      string // operationId
	variant string // what sets the row apart from the operation's other rows
	write   bool   // each cell on a copy of its own
	config  func(*config.Config)
	request func(c caller) (method, path, body string)
	cells   map[caller]cell
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
func sameRequest(method, path, body string) func(caller) (string, string, string) {
	return func(caller) (string, string, string) { return method, path, body }
}

// matrixRows are the rows, a phase's operations added by that phase.
func matrixRows() []matrixRow {
	return []matrixRow{
		// The account level: any valid credential (6.4).
		{op: "listWorkspaces", request: sameRequest(http.MethodGet, "/api/v0/workspaces", ""), cells: every(cellOK)},
		{op: "checkWorkspaceSlug", request: sameRequest(http.MethodGet, "/api/v0/workspace-slugs/acme", ""), cells: every(cellOK)},
		{op: "createWorkspace", write: true, request: sameRequest(http.MethodPost, "/api/v0/workspaces", `{"name":"New","slug":"new"}`),
			cells: every(cellCreated)},
		{op: "createWorkspace", variant: "creation switched off", write: true,
			config:  func(cfg *config.Config) { cfg.Workspace.CreationEnabled = false },
			request: sameRequest(http.MethodPost, "/api/v0/workspaces", `{"name":"New","slug":"new"}`), cells: every(cellCreationDisabled)},
		// The workspace level.
		{op: "getWorkspace",
			request: func(c caller) (string, string, string) {
				return http.MethodGet, "/api/v0/workspaces/" + workspaceOf(c), ""
			},
			cells: map[caller]cell{callerAdmin: cellOK, callerMember: cellOK, callerGuest: cellOK,
				callerNever: cellWorkspaceNotFound, callerRemoved: cellWorkspaceNotFound, callerDeleted: cellWorkspaceNotFound}},
	}
}

// matrixData is the prepared database, the signing key every app on a copy
// of it shares, and each column's access token.
type matrixData struct {
	url     string
	keyFile string
	tokens  map[caller]string
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
// later removed; the workspace gone with its admin; and the workspace
// other, whose admin was never a member of acme and where the removed
// member is still active, so that a role read in the wrong workspace lets
// either into acme. Through SQL, until the stores of P5 and P2 replace it:
// the removed member's membership of acme ended, and gone's workspace row
// alone soft-deleted. Everything that connected to the database is closed
// when it returns, so that it can be copied.
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
		seed := matrixSeed{t: t, store: workspacepg.New(pool), ids: ids, now: time.Now()}
		acme := seed.workspace("acme", callerAdmin)
		seed.join(acme, callerAdmin, shared.RoleAdmin)
		seed.join(acme, callerMember, shared.RoleMember)
		seed.join(acme, callerGuest, shared.RoleGuest)
		seed.join(acme, callerRemoved, shared.RoleMember)
		gone := seed.workspace("gone", callerDeleted)
		seed.join(gone, callerDeleted, shared.RoleAdmin)
		other := seed.workspace("other", callerNever)
		seed.join(other, callerNever, shared.RoleAdmin)
		seed.join(other, callerRemoved, shared.RoleMember)
		// No store removes a member (P5) or deletes a workspace (P2) yet, so
		// SQL stands in until those phases replace it. The first statement
		// ends the removed member's membership of acme. The second
		// soft-deletes the workspace row of gone alone, which leaves its
		// admin's membership active in a deleted workspace; P2's delete
		// also soft-deletes the memberships, invitations and preferences,
		// so replacing it changes the state this column is asked about.
		seed.exec(pool, "UPDATE workspace_members SET is_active = false WHERE workspace_id = $1 AND member_id = $2",
			acme, ids[callerRemoved])
		seed.exec(pool, "UPDATE workspaces SET deleted_at = now() WHERE id = $1", gone)
	})
	if !prepared {
		t.FailNow()
	}
	return d
}

// matrixSeed writes the prepared workspaces and memberships through the
// workspace store; exec runs the SQL that stands in for the stores P2 and
// P5 add.
type matrixSeed struct {
	t     *testing.T
	store *workspacepg.Store
	ids   map[caller]uuid.UUID
	now   time.Time
}

func (s matrixSeed) workspace(slug string, admin caller) uuid.UUID {
	s.t.Helper()
	w, err := s.store.CreateWorkspace(context.Background(), workspaceapp.WorkspaceRow{
		ID: uuid.NewV7(), Name: slug, Slug: slug, Timezone: "UTC", CreatedBy: s.ids[admin], Now: s.now,
	})
	if err != nil {
		s.t.Fatal(err)
	}
	return w.ID
}

func (s matrixSeed) join(workspace uuid.UUID, c caller, role shared.Role) {
	s.t.Helper()
	if err := s.store.CreateMember(context.Background(), workspaceapp.MemberRow{
		ID: uuid.NewV7(), WorkspaceID: workspace, MemberID: s.ids[c], Role: role, CreatedBy: s.ids[c], Now: s.now,
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

// Each cell of the matrix, the writing ones in parallel on their copies.
func TestPermissionMatrix(t *testing.T) {
	d := prepareMatrix(t)
	contract := apitest.Load(t)
	reads := startApp(t, d.config(t, pgtest.NewDatabaseFrom(t, d.url), nil), migrations.FS())
	for _, r := range matrixRows() {
		for _, c := range workspaceColumns {
			want, ok := r.cells[c]
			if !ok {
				continue // TestThePermissionMatrixCoversEveryOperation reports it
			}
			t.Run(r.name()+"/"+string(c), func(t *testing.T) {
				// Connections: at most -parallel cells (GOMAXPROCS by
				// default) run at once, and each that starts an app of its
				// own (a writing cell) opens a pool of up to testConfig's
				// MaxConns (4), besides the reads app's pool and pgtest's
				// admin pool (4 each), against the container's
				// max_connections of 100. P2 bounds the cells that start an
				// app with a semaphore.
				t.Parallel()
				base := reads
				if r.write || r.config != nil {
					base = startApp(t, d.config(t, pgtest.NewDatabaseFrom(t, d.url), r.config), migrations.FS())
				}
				method, path, body := r.request(c)
				status, answer := call(t, contract, method, base+path, d.tokens[c], body)
				got := cell{status: status}
				if status >= http.StatusBadRequest {
					got.code = problemCode(t, []byte(answer))
				}
				if got != want {
					t.Errorf("%s %s = %d %s, want %s", method, path, status, strings.TrimSpace(answer), want)
				}
			})
		}
	}
}

// matrixViolations reports where the matrix and the contract part: an
// operation of matrixModules without a row; a row that names no operation;
// a row without a cell for a column; a cell whose request is not the
// operation its row names, so that no row tests another operation under
// its name; a row that sends anything but GET without write, whose cells
// could write on the copy the reading cells share.
func matrixViolations(ops []apitest.Operation, rows []matrixRow) []string {
	var found []string
	byID, inMatrix := map[string]apitest.Operation{}, map[string]bool{}
	for _, op := range ops {
		byID[op.ID] = op
	}
	for _, r := range rows {
		inMatrix[r.op] = true
		op, named := byID[r.op]
		unsafe := ""
		for _, c := range workspaceColumns {
			if _, ok := r.cells[c]; !ok {
				found = append(found, fmt.Sprintf("row %s has no cell for %s", r.name(), c))
			}
			method, path, _ := r.request(c)
			if named && (method != op.Method || !pathOf(op.Path, path)) {
				found = append(found, fmt.Sprintf("row %s, %s: %s %s is not %s", r.name(), c, method, path, op.Pattern()))
			}
			if method != http.MethodGet && !r.write && unsafe == "" {
				unsafe = method
			}
		}
		if unsafe != "" {
			found = append(found, fmt.Sprintf("row %s sends %s without write: its cells could run on the reads' copy", r.name(), unsafe))
		}
	}
	for _, op := range ops {
		if slices.ContainsFunc(op.Tags, func(tag string) bool { return slices.Contains(matrixModules, tag) }) && !inMatrix[op.ID] {
			found = append(found, fmt.Sprintf("operation %s of %s has no row", op.ID, strings.Join(op.Tags, ", ")))
		}
	}
	for _, r := range rows {
		if _, named := byID[r.op]; !named {
			found = append(found, fmt.Sprintf("row %s names no operation of the contract", r.name()))
		}
	}
	return found
}

// pathOf reports whether path, its query left out, is a path of the
// contract's pattern: each {parameter} one segment that is not empty, every
// other segment the same.
func pathOf(pattern, path string) bool {
	path, _, _ = strings.Cut(path, "?")
	want, got := strings.Split(pattern, "/"), strings.Split(path, "/")
	if len(want) != len(got) {
		return false
	}
	for i, segment := range want {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			if got[i] == "" {
				return false
			}
		} else if segment != got[i] {
			return false
		}
	}
	return true
}

// Every operation of the matrix's modules has a row, each row names an
// operation, has a cell for each column and sends that operation's request
// from each, and a row that writes says so (M3 design 9.2): a new
// operation without a row fails here, and so does a row that tests another
// operation under its name.
func TestThePermissionMatrixCoversEveryOperation(t *testing.T) {
	for _, v := range matrixViolations(apitest.Load(t).Operations(), matrixRows()) {
		t.Error(v)
	}
}

// Each check of matrixViolations fails on its counterexample.
func TestMatrixViolationsCatchesEachGap(t *testing.T) {
	ops := []apitest.Operation{{ID: "getWorkspace", Tags: []string{"workspace"}, Method: http.MethodGet, Path: "/api/v0/workspaces/{slug}"},
		{ID: "getMe", Tags: []string{"identity"}, Method: http.MethodGet, Path: "/api/v0/me"}}
	get := sameRequest(http.MethodGet, "/api/v0/workspaces/acme", "")
	row := matrixRow{op: "getWorkspace", request: get, cells: every(cellOK)}
	// guestSends is row, but the guest's cell sends method path.
	guestSends := func(method, path string) matrixRow {
		r := row
		r.request = func(c caller) (string, string, string) {
			if c == callerGuest {
				return method, path, ""
			}
			return get(c)
		}
		return r
	}
	// A matching matrix; a query is no part of the path.
	lists := apitest.Operation{ID: "listWorkspaces", Tags: []string{"workspace"}, Method: http.MethodGet, Path: "/api/v0/workspaces"}
	paged := matrixRow{op: "listWorkspaces", request: sameRequest(http.MethodGet, "/api/v0/workspaces?page=2", ""), cells: every(cellOK)}
	if got := matrixViolations(append(ops, lists), []matrixRow{row, paged}); len(got) != 0 {
		t.Fatalf("a matching matrix: %q, want none", got)
	}
	partial := row
	partial.cells = map[caller]cell{callerAdmin: cellOK}
	var mislabelled []string
	for _, c := range workspaceColumns {
		mislabelled = append(mislabelled, fmt.Sprintf("row getWorkspace, %s: GET /api/v0/workspaces is not GET /api/v0/workspaces/{slug}", c))
	}
	posting := guestSends(http.MethodPost, "/api/v0/workspaces/acme")
	posting.write = true
	creates := apitest.Operation{ID: "createWorkspace", Tags: []string{"workspace"}, Method: http.MethodPost, Path: "/api/v0/workspaces"}
	tests := []struct {
		name string
		ops  []apitest.Operation
		rows []matrixRow
		want []string
	}{
		{"an operation without a row", append(ops, apitest.Operation{ID: "updateWorkspace", Tags: []string{"workspace"}}),
			[]matrixRow{row}, []string{"operation updateWorkspace of workspace has no row"}},
		{"a row of no operation", ops, []matrixRow{row, {op: "renameWorkspace", request: get, cells: every(cellOK)}},
			[]string{"row renameWorkspace names no operation of the contract"}},
		{"a row without a cell", ops, []matrixRow{partial}, []string{
			"row getWorkspace has no cell for member", "row getWorkspace has no cell for guest", "row getWorkspace has no cell for never a member",
			"row getWorkspace has no cell for removed", "row getWorkspace has no cell for workspace deleted",
		}},
		{"a row that sends another operation", ops, []matrixRow{{op: "getWorkspace", request: sameRequest(http.MethodGet, "/api/v0/workspaces", ""),
			cells: every(cellOK)}}, mislabelled},
		{"a cell of another method", ops, []matrixRow{posting},
			[]string{"row getWorkspace, guest: POST /api/v0/workspaces/acme is not GET /api/v0/workspaces/{slug}"}},
		{"a cell of another path", ops, []matrixRow{guestSends(http.MethodGet, "/api/v0/workspace-slugs/acme")},
			[]string{"row getWorkspace, guest: GET /api/v0/workspace-slugs/acme is not GET /api/v0/workspaces/{slug}"}},
		{"a cell with an empty parameter", ops, []matrixRow{guestSends(http.MethodGet, "/api/v0/workspaces/")},
			[]string{"row getWorkspace, guest: GET /api/v0/workspaces/ is not GET /api/v0/workspaces/{slug}"}},
		{"a cell with a longer path", ops, []matrixRow{guestSends(http.MethodGet, "/api/v0/workspaces/acme/members")},
			[]string{"row getWorkspace, guest: GET /api/v0/workspaces/acme/members is not GET /api/v0/workspaces/{slug}"}},
		{"a write without write", append(ops, creates), []matrixRow{row,
			{op: "createWorkspace", request: sameRequest(http.MethodPost, "/api/v0/workspaces", `{}`), cells: every(cellCreated)}},
			[]string{"row createWorkspace sends POST without write: its cells could run on the reads' copy"}},
	}
	for _, tt := range tests {
		if got := matrixViolations(tt.ops, tt.rows); !slices.Equal(got, tt.want) {
			t.Errorf("%s: %q, want %q", tt.name, got, tt.want)
		}
	}
}
