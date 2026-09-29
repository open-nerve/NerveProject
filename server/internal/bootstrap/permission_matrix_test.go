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
// prepared once, through the modules' stores; the cells that only read
// share one copy of it, and each cell that writes gets a copy of its own
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

// prepareMatrix fills a database for the matrix: an account for each
// column, registered through the API for its token; the workspace acme with
// its admin, member and guest, and the removed member's ended membership;
// the deleted workspace gone with its admin. Everything that connected to
// the database is closed when it returns, so that it can be copied.
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
		// No store removes a member or deletes a workspace yet: SQL does
		// what they will, until the phases that add them.
		seed.exec(pool, "UPDATE workspace_members SET is_active = false WHERE member_id = $1", ids[callerRemoved])
		seed.exec(pool, "UPDATE workspaces SET deleted_at = now() WHERE id = $1", gone)
	})
	if !prepared {
		t.FailNow()
	}
	return d
}

// matrixSeed writes the prepared data through the modules' stores.
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
					t.Errorf("%s %s = %d %s, want %d %s", method, path, status, answer, want.status, want.code)
				}
			})
		}
	}
}

// matrixViolations reports where the matrix and the contract part: an
// operation of matrixModules without a row, a row that names no operation,
// a row without a cell for a column.
func matrixViolations(ops []apitest.Operation, rows []matrixRow) []string {
	var found []string
	inContract, inMatrix := map[string]bool{}, map[string]bool{}
	for _, r := range rows {
		inMatrix[r.op] = true
		for _, c := range workspaceColumns {
			if _, ok := r.cells[c]; !ok {
				found = append(found, fmt.Sprintf("row %s has no cell for %s", r.name(), c))
			}
		}
	}
	for _, op := range ops {
		inContract[op.ID] = true
		if slices.ContainsFunc(op.Tags, func(tag string) bool { return slices.Contains(matrixModules, tag) }) && !inMatrix[op.ID] {
			found = append(found, fmt.Sprintf("operation %s of %s has no row", op.ID, strings.Join(op.Tags, ", ")))
		}
	}
	for _, r := range rows {
		if !inContract[r.op] {
			found = append(found, fmt.Sprintf("row %s names no operation of the contract", r.name()))
		}
	}
	return found
}

// Every operation of the matrix's modules has a row, each row names an
// operation and has a cell for each column (M3 design 9.2): a new
// operation without a row fails here.
func TestThePermissionMatrixCoversEveryOperation(t *testing.T) {
	for _, v := range matrixViolations(apitest.Load(t).Operations(), matrixRows()) {
		t.Error(v)
	}
}

// Each check of matrixViolations fails on its counterexample.
func TestMatrixViolationsCatchesEachGap(t *testing.T) {
	ops := []apitest.Operation{{ID: "getWorkspace", Tags: []string{"workspace"}}, {ID: "getMe", Tags: []string{"identity"}}}
	row := matrixRow{op: "getWorkspace", cells: every(cellOK)}
	if got := matrixViolations(ops, []matrixRow{row}); len(got) != 0 {
		t.Fatalf("a matching matrix: %q, want none", got)
	}
	partial := row
	partial.cells = map[caller]cell{callerAdmin: cellOK}
	tests := []struct {
		name string
		ops  []apitest.Operation
		rows []matrixRow
		want []string
	}{
		{"an operation without a row", append(ops, apitest.Operation{ID: "updateWorkspace", Tags: []string{"workspace"}}),
			[]matrixRow{row}, []string{"operation updateWorkspace of workspace has no row"}},
		{"a row of no operation", ops, []matrixRow{row, {op: "renameWorkspace", cells: every(cellOK)}},
			[]string{"row renameWorkspace names no operation of the contract"}},
		{"a row without a cell", ops, []matrixRow{partial}, []string{
			"row getWorkspace has no cell for member", "row getWorkspace has no cell for guest", "row getWorkspace has no cell for never a member",
			"row getWorkspace has no cell for removed", "row getWorkspace has no cell for workspace deleted",
		}},
	}
	for _, tt := range tests {
		if got := matrixViolations(tt.ops, tt.rows); !slices.Equal(got, tt.want) {
			t.Errorf("%s: %q, want %q", tt.name, got, tt.want)
		}
	}
}
