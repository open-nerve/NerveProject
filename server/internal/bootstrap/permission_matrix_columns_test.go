package bootstrap

import (
	"fmt"
	"maps"
	"net/http"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
)

// The columns of the project level (M3 design 9.2), the accounts that call
// from them, and the projects they target.

// The project level's columns that are not the workspace level's. PA, PM
// and the member before are workspace members; PG, and WG-, the workspace's
// guests; PM+WA and WA-, its admins. X is the workspace level's three
// outsiders: never a member, removed, and the deleted workspace's.
const (
	callerProjectAdmin   caller = "project admin"                      // PA
	callerProjectMember  caller = "project member"                     // PM
	callerProjectGuest   caller = "project guest"                      // PG: the workspace's guest
	callerMemberAndAdmin caller = "project member and workspace admin" // PM+WA
	callerAdminOnly      caller = "workspace admin only"               // WA-: the workspace's admin
	callerMemberPublic   caller = "workspace member only, public"      // WM-公: the workspace's member
	callerMemberPrivate  caller = "workspace member only, private"     // WM-私: the workspace's member
	callerGuestOnly      caller = "workspace guest only"               // WG-
	callerBefore         caller = "project member before"              // P-前: ended in the private project
	// callerArchivedAdmin is PA on acme's archived project: the column of
	// 9.2's small table of the archived project.
	callerArchivedAdmin caller = "archived project admin"
)

// projectColumns are the columns of the project level, in the order of
// 9.2's table, X as its three accounts.
var projectColumns = []caller{callerProjectAdmin, callerProjectMember, callerProjectGuest, callerMemberAndAdmin, callerAdminOnly,
	callerMemberPublic, callerMemberPrivate, callerGuestOnly, callerBefore, callerNever, callerRemoved, callerDeleted}

// archivedColumns are the columns of the archived project's table (9.2).
var archivedColumns = []caller{callerArchivedAdmin}

// matrixAccounts are the accounts prepareMatrix registers: each workspace
// column's, and each project column's that is none of those (M3 design
// 9.2: eleven).
var matrixAccounts = append(slices.Clone(workspaceColumns), callerProjectAdmin, callerProjectMember, callerMemberAndAdmin, callerGuestOnly,
	callerBefore)

// accountOf is the account a column's cells call as: a project column that
// is a workspace column's account under the project level's name is that
// account.
func accountOf(c caller) caller {
	switch c {
	case callerArchivedAdmin:
		return callerProjectAdmin
	case callerProjectGuest:
		return callerGuest
	case callerAdminOnly:
		return callerAdmin
	case callerMemberPublic, callerMemberPrivate:
		return callerMember
	}
	return c
}

// projectOf is the key of the project a project column's cells target:
// acme's private project for the columns about it, its archived one for
// the archived project's, gone's for the deleted workspace's, acme's public
// one for every other; PA, PM, PG and PM+WA answer alike in both, WG- too
// (9.2).
func projectOf(c caller) string {
	switch c {
	case callerMemberPrivate, callerBefore:
		return "acme/private"
	case callerArchivedAdmin:
		return "acme/archived"
	case callerDeleted:
		return "gone/project"
	}
	return "acme/public"
}

// Every account of a column is registered, and every account registered is
// some column's: a column whose account prepareMatrix did not register would
// call with no token, and its cells answer 401 whatever the rule.
func TestEveryColumnCallsAsARegisteredAccount(t *testing.T) {
	var used []caller
	for _, c := range slices.Concat(workspaceColumns, projectColumns, archivedColumns) {
		if !slices.Contains(matrixAccounts, accountOf(c)) {
			t.Errorf("column %s calls as %s, which prepareMatrix does not register", c, accountOf(c))
		}
		used = append(used, accountOf(c))
	}
	for _, a := range matrixAccounts {
		if !slices.Contains(used, a) {
			t.Errorf("the account %s is no column's", a)
		}
	}
	if len(matrixAccounts) != 11 {
		t.Errorf("%d accounts, want 9.2's 11", len(matrixAccounts))
	}
	// account names a registered account: PG's column, whose account is the
	// workspace's guest, is none, and fails the test at once; an account is
	// uuid.Nil until prepareMatrix registers its id, so that a request built
	// without a database names one still (matrixViolations).
	if failed, want := fatalOf(func(tb testing.TB) { newSeeded().in(tb).account(callerProjectGuest) }),
		"no account project guest is registered"; failed != want {
		t.Errorf("the account of PG's column itself: failed with %q, want %q", failed, want)
	}
	s := newSeeded().in(t)
	for _, a := range matrixAccounts {
		if id := s.account(a); id != uuid.Nil() {
			t.Errorf("the account %s before prepareMatrix registers it = %s, want uuid.Nil", a, id)
		}
	}
}

// Each check of the project level's columns and of the not-target
// parameters fails on its counterexample: a project row's columns are
// projectColumns, each cell aims at its column's project, and a parameter
// listed as not a target leaves the path's others checked.
func TestMatrixViolationsCatchesEachColumnGap(t *testing.T) {
	s := newSeeded().in(t)
	getProject := apitest.Operation{ID: "getProject", Tags: []string{"project"}, Method: http.MethodGet, Path: "/api/v0/projects/{project_id}"}
	cells := map[caller]cell{}
	for _, c := range projectColumns {
		cells[c] = cellOK
	}
	toProject := func(c caller, s seeded) (string, string, string) {
		return http.MethodGet, "/api/v0/projects/" + s.project(projectOf(c)).String(), ""
	}
	row := matrixRow{op: getProject.ID, columns: projectColumns, request: toProject, cells: cells}
	// with is row with change made to a copy of it.
	with := func(change func(r *matrixRow)) matrixRow {
		r := row
		r.cells = maps.Clone(cells)
		change(&r)
		return r
	}
	// aimed is row, but the column at's cell aims at the project key.
	aimed := func(at caller, key string) matrixRow {
		return with(func(r *matrixRow) {
			r.request = func(c caller, s seeded) (string, string, string) {
				if c == at {
					return http.MethodGet, "/api/v0/projects/" + s.project(key).String(), ""
				}
				return toProject(c, s)
			}
		})
	}
	identifiers := apitest.Operation{ID: "checkProjectIdentifier", Tags: []string{"project"}, Method: http.MethodGet,
		Path: "/api/v0/workspaces/{slug}/project-identifiers/{identifier}"}
	checks := matrixRow{op: identifiers.ID, request: toWorkspace(http.MethodGet, "/project-identifiers/WEB", ""), cells: every(cellOK)}
	listed := matrixExemptions{notTargets: map[notTarget]string{{identifiers.Path, "{identifier}"}: "an identifier asked about"}}
	ops := []apitest.Operation{getProject, identifiers}
	if got := matrixViolations(ops, listed, []matrixRow{row, checks}, s, nil); len(got) != 0 {
		t.Fatalf("a matching matrix: %q, want none", got)
	}
	public, private := s.project("acme/public").String(), s.project("acme/private").String()
	tests := []struct {
		name   string
		exempt matrixExemptions
		rows   []matrixRow
		want   []string
	}{
		{"a project row without a cell for PA", listed, []matrixRow{with(func(r *matrixRow) { delete(r.cells, callerProjectAdmin) }), checks},
			[]string{"row getProject has no cell for project admin"}},
		{"a project row with a workspace column's cell", listed, []matrixRow{with(func(r *matrixRow) { r.cells[callerAdmin] = cellOK }), checks},
			[]string{"row getProject has a cell for admin, which is none of its columns"}},
		{"a workspace row with a project column's cell", listed, []matrixRow{row, func() matrixRow {
			r := checks
			r.cells = maps.Clone(checks.cells)
			r.cells[callerBefore] = cellOK
			return r
		}()}, []string{"row checkProjectIdentifier has a cell for project member before, which is none of its columns"}},
		{"a cell of another column's project", listed, []matrixRow{with(func(r *matrixRow) {
			r.request = func(c caller, s seeded) (string, string, string) {
				if c == callerMemberPrivate {
					return http.MethodGet, "/api/v0/projects/" + public, ""
				}
				return toProject(c, s)
			}
		}), checks}, []string{fmt.Sprintf("row getProject, %s: {project_id} %s is not its column's project acme/private, %s",
			callerMemberPrivate, public, private)}},
		// The X columns answer alike wherever they aim; what they aim at is
		// what they test: gone's project, deleted with it, hidden from its
		// admin; the public project, which only his ended membership of acme
		// keeps the removed member out of.
		{"the deleted workspace's column at acme's project", listed, []matrixRow{aimed(callerDeleted, "acme/public"), checks},
			[]string{fmt.Sprintf("row getProject, %s: {project_id} %s is not its column's project gone/project, %s", callerDeleted, public,
				s.project("gone/project"))}},
		{"the removed member's column at the private project", listed, []matrixRow{aimed(callerRemoved, "acme/private"), checks},
			[]string{fmt.Sprintf("row getProject, %s: {project_id} %s is not its column's project acme/public, %s", callerRemoved, private,
				public)}},
		{"a project row of PA's column alone", listed, []matrixRow{with(func(r *matrixRow) {
			r.columns, r.cells = projectColumns[:1], map[caller]cell{callerProjectAdmin: cellOK}
		}), checks}, []string{`row getProject has the columns ["project admin"], which are no table of the matrix`}},
		// A project's operation in a workspace-level row: each of its six
		// cells aims at its column's project, and the project level's own
		// columns are never asked.
		{"a project operation in a workspace-level row", listed, []matrixRow{with(func(r *matrixRow) {
			r.columns, r.cells = nil, every(cellOK)
		}), checks}, func() []string {
			var want []string
			for _, c := range []caller{callerAdmin, callerMember, callerGuest} {
				want = append(want, fmt.Sprintf("row getProject, %s: {project_id} from a column of no project table (matrixTables): "+
					"a project's row names its columns", c))
			}
			return want
		}()},
		{"a {slug} of another column's workspace beside a listed {identifier}", listed, []matrixRow{row, func() matrixRow {
			r := checks
			r.request = sameRequest(http.MethodGet, "/api/v0/workspaces/acme/project-identifiers/WEB", "")
			return r
		}()}, []string{"row checkProjectIdentifier, workspace deleted: targets the workspace acme, not its column's gone"}},
		{"an {identifier} not listed", matrixExemptions{}, []matrixRow{row, checks}, func() []string {
			var want []string
			for _, c := range workspaceColumns {
				want = append(want, fmt.Sprintf("row checkProjectIdentifier, %s: {identifier} is no target the matrix knows: "+
					"list /api/v0/workspaces/{slug}/project-identifiers/{identifier} as not a target, with its reason", c))
			}
			return want
		}()},
		{"a not-target parameter its path does not have", matrixExemptions{notTargets: map[notTarget]string{
			{identifiers.Path, "{identifier}"}: "an identifier asked about", {identifiers.Path, "{project_id}"}: "a mistake"}},
			[]matrixRow{row, checks},
			[]string{"the not-target {project_id} of /api/v0/workspaces/{slug}/project-identifiers/{identifier} is no parameter of an operation's path"}},
	}
	for _, tt := range tests {
		if got := matrixViolations(ops, tt.exempt, tt.rows, s, nil); !slices.Equal(got, tt.want) {
			t.Errorf("%s: %q, want %q", tt.name, got, tt.want)
		}
	}
	// A parameter passed over before another leaves that one checked too.
	if v := targetViolation("/api/v0/project-identifiers/{identifier}/projects/{project_id}", "/api/v0/project-identifiers/WEB/projects/"+public,
		callerMemberPrivate, s, func(param string) bool { return param == "{identifier}" }); v == "" {
		t.Error("a {project_id} of another column's project after a parameter passed over: no violation")
	}
	// A project never seeded fails the test at once, and names it.
	failed := fatalOf(func(tb testing.TB) {
		matrixViolations(ops, listed, []matrixRow{with(func(r *matrixRow) {
			r.request = func(c caller, s seeded) (string, string, string) {
				return http.MethodGet, "/api/v0/projects/" + s.project("acme/nothing").String(), ""
			}
		})}, newSeeded().in(tb), nil)
	})
	if want := "no project acme/nothing is seeded"; failed != want {
		t.Errorf("a project never seeded: failed with %q, want %q", failed, want)
	}
}
