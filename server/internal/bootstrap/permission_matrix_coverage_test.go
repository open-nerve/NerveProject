package bootstrap

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
)

// matrixViolations reports where the matrix and the contract part: an
// operation without a row, unless every tag it has is on exempt, so that a
// new module's operations need rows without anyone listing the module; an
// exempt module that no operation carries, which a misspelling would be; a
// row that names no operation; a row without a cell for a column; a cell
// whose request is not the operation its row names, so that no row tests
// another operation under its name; a row that sends anything but GET
// without write, whose cells could write on the copy the reading cells
// share.
func matrixViolations(ops []apitest.Operation, exempt []string, rows []matrixRow) []string {
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
			method, path, _ := r.request(c, seeded{})
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
		exempted := len(op.Tags) > 0 && !slices.ContainsFunc(op.Tags, func(tag string) bool { return !slices.Contains(exempt, tag) })
		if !exempted && !inMatrix[op.ID] {
			found = append(found, fmt.Sprintf("operation %s, tagged %v, has no row", op.ID, op.Tags))
		}
	}
	for _, module := range exempt {
		if !slices.ContainsFunc(ops, func(op apitest.Operation) bool { return slices.Contains(op.Tags, module) }) {
			found = append(found, fmt.Sprintf("the exempt module %s has no operation", module))
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

// Every operation but the exempt modules' has a row, each exempt module has
// operations, each row names an operation, has a cell for each column and
// sends that operation's request from each, and a row that writes says so
// (M3 design 9.2): a new operation without a row fails here, whatever its
// module, and so does a row that tests another operation under its name.
func TestThePermissionMatrixCoversEveryOperation(t *testing.T) {
	for _, v := range matrixViolations(apitest.Load(t).Operations(), matrixExempt, matrixRows()) {
		t.Error(v)
	}
}

// Each check of matrixViolations fails on its counterexample. identity is
// exempt unless a case says otherwise.
func TestMatrixViolationsCatchesEachGap(t *testing.T) {
	ops := []apitest.Operation{{ID: "getWorkspace", Tags: []string{"workspace"}, Method: http.MethodGet, Path: "/api/v0/workspaces/{slug}"},
		{ID: "getMe", Tags: []string{"identity"}, Method: http.MethodGet, Path: "/api/v0/me"}}
	exempt := []string{"identity"}
	get := sameRequest(http.MethodGet, "/api/v0/workspaces/acme", "")
	row := matrixRow{op: "getWorkspace", request: get, cells: every(cellOK)}
	// guestSends is row, but the guest's cell sends method path.
	guestSends := func(method, path string) matrixRow {
		r := row
		r.request = func(c caller, s seeded) (string, string, string) {
			if c == callerGuest {
				return method, path, ""
			}
			return get(c, s)
		}
		return r
	}
	// A matching matrix; a query is no part of the path.
	lists := apitest.Operation{ID: "listWorkspaces", Tags: []string{"workspace"}, Method: http.MethodGet, Path: "/api/v0/workspaces"}
	paged := matrixRow{op: "listWorkspaces", request: sameRequest(http.MethodGet, "/api/v0/workspaces?page=2", ""), cells: every(cellOK)}
	if got := matrixViolations(append(ops, lists), exempt, []matrixRow{row, paged}); len(got) != 0 {
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
			[]matrixRow{row}, []string{"operation updateWorkspace, tagged [workspace], has no row"}},
		{"an operation of a module no list names", append(ops, apitest.Operation{ID: "listProjects", Tags: []string{"project"}}),
			[]matrixRow{row}, []string{"operation listProjects, tagged [project], has no row"}},
		{"an operation without a tag", append(ops, apitest.Operation{ID: "getHealth"}),
			[]matrixRow{row}, []string{"operation getHealth, tagged [], has no row"}},
		{"an operation with an exempt tag and another", append(ops, apitest.Operation{ID: "getMyWorkspaces", Tags: []string{"identity", "workspace"}}),
			[]matrixRow{row}, []string{"operation getMyWorkspaces, tagged [identity workspace], has no row"}},
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
		if got := matrixViolations(tt.ops, exempt, tt.rows); !slices.Equal(got, tt.want) {
			t.Errorf("%s: %q, want %q", tt.name, got, tt.want)
		}
	}
	// An exempt entry that no operation carries: stale, or misspelled, when
	// the module it meant has operations without rows besides.
	for _, tt := range []struct {
		name   string
		exempt []string
		want   []string
	}{
		{"a stale exempt entry", []string{"identity", "instance"}, []string{"the exempt module instance has no operation"}},
		{"a misspelled exempt entry", []string{"identities"},
			[]string{"operation getMe, tagged [identity], has no row", "the exempt module identities has no operation"}},
	} {
		if got := matrixViolations(ops, tt.exempt, []matrixRow{row}); !slices.Equal(got, tt.want) {
			t.Errorf("%s: %q, want %q", tt.name, got, tt.want)
		}
	}
}
