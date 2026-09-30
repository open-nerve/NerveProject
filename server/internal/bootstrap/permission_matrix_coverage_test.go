package bootstrap

import (
	"fmt"
	"maps"
	"net/http"
	"runtime"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
)

// matrixViolations reports where the matrix and the contract part: an
// operation without a row, unless every tag it has is on exempt.modules, so
// that a new module's operations need rows without anyone listing the
// module, or it is on exempt.public; an exempt module that no operation
// carries, which a misspelling would be; a public exemption that names no
// operation, one that needs a token, one that has a row, or one whose test
// is none of tests (the names of the package's tests); a not-target
// path that is no operation's; a row that names no operation; a row without a cell for a column; a cell
// whose request is not the operation its row names, so that no row tests
// another operation under its name; a cell that does not target its
// column's workspace (targetViolation), so that no column quietly tests
// another's case; a row that sends anything but GET without write, whose
// cells could write on the copy the reading cells share. The requests name
// the rows of s.
func matrixViolations(ops []apitest.Operation, exempt matrixExemptions, rows []matrixRow, s seeded, tests map[string]bool) []string {
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
			method, path, _ := r.request(c, s)
			switch {
			case !named:
			case method != op.Method || !pathOf(op.Path, path):
				found = append(found, fmt.Sprintf("row %s, %s: %s %s is not %s", r.name(), c, method, path, op.Pattern()))
			default:
				if _, listed := exempt.notTargets[op.Path]; listed {
					break
				}
				if v := targetViolation(op.Path, path, c, s); v != "" {
					found = append(found, fmt.Sprintf("row %s, %s: %s", r.name(), c, v))
				}
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
		exempted := len(op.Tags) > 0 && !slices.ContainsFunc(op.Tags, func(tag string) bool { return !slices.Contains(exempt.modules, tag) })
		if _, public := exempt.public[op.ID]; !exempted && !public && !inMatrix[op.ID] {
			found = append(found, fmt.Sprintf("operation %s, tagged %v, has no row", op.ID, op.Tags))
		}
	}
	for _, id := range slices.Sorted(maps.Keys(exempt.public)) {
		op, named := byID[id]
		switch {
		case !named:
			found = append(found, fmt.Sprintf("the public exemption %s names no operation of the contract", id))
		case !op.Public:
			found = append(found, fmt.Sprintf("operation %s is exempt as public, but needs a token", id))
		case inMatrix[id]:
			found = append(found, fmt.Sprintf("operation %s is exempt as public, and has a row", id))
		case !tests[exempt.public[id]]:
			found = append(found, fmt.Sprintf("the public exemption %s names %s, which no test of the package is", id, exempt.public[id]))
		}
	}
	for _, path := range slices.Sorted(maps.Keys(exempt.notTargets)) {
		if !slices.ContainsFunc(ops, func(op apitest.Operation) bool { return op.Path == path }) {
			found = append(found, fmt.Sprintf("the not-target path %s is no operation's", path))
		}
	}
	for _, module := range exempt.modules {
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

// targetViolation is what is wrong with where path, a path of pattern that
// a cell of the column c sends, points; "" when nothing. A workspace named
// by its slug ({slug} right after workspaces) must be workspaceOf(c), and a
// row named by its id (a parameter ending in _id) must be a row of s under
// workspaceOf(c): a cell of the deleted workspace's column that named acme
// would get the 404 of a workspace its caller is not in, and pass whether
// deleted workspaces are hidden or not. Any other parameter is reported: a
// path whose parameters are no column's target is listed as such, with its
// reason (matrixExemptions.notTargets), and not given here.
func targetViolation(pattern, path string, c caller, s seeded) string {
	path, _, _ = strings.Cut(path, "?")
	want, got := strings.Split(pattern, "/"), strings.Split(path, "/")
	for i, segment := range want {
		switch {
		case !strings.HasPrefix(segment, "{") || !strings.HasSuffix(segment, "}"):
		case segment == "{slug}" && i > 0 && want[i-1] == "workspaces":
			if got[i] != workspaceOf(c) {
				return fmt.Sprintf("targets the workspace %s, not its column's %s", got[i], workspaceOf(c))
			}
		case strings.HasSuffix(segment, "_id}"):
			id, err := uuid.Parse(got[i])
			slug, isRow := s.workspaceOfRow(id)
			if err != nil || !isRow || slug != workspaceOf(c) {
				return fmt.Sprintf("%s %s is no row seeded under its column's workspace %s", segment, got[i], workspaceOf(c))
			}
		default:
			return fmt.Sprintf("%s is no target the matrix knows: list %s as not a target, with its reason", segment, pattern)
		}
	}
	return ""
}

// Every operation but the exempt modules' has a row, each exempt module has
// operations, each row names an operation, has a cell for each column and
// sends that operation's request from each, to the column's workspace, and
// a row that writes says so (M3 design 9.2): a new operation without a row
// fails here, whatever its module, and so does a row that tests another
// operation under its name, or a column's case in another workspace; the
// test a public exemption names is one of the package's, so that renaming
// or deleting it fails here too.
func TestThePermissionMatrixCoversEveryOperation(t *testing.T) {
	for _, v := range matrixViolations(apitest.Load(t).Operations(), matrixExempt, matrixRows(), newSeeded().in(t), packageTests(t)) {
		t.Error(v)
	}
}

// Each check of matrixViolations fails on its counterexample. identity is
// exempt unless a case says otherwise, and so is the public
// getWorkspaceInvitation, which each ops holds, for TestOfItsOwn, which
// each case but one has among the package's tests.
func TestMatrixViolationsCatchesEachGap(t *testing.T) {
	ops := []apitest.Operation{{ID: "getWorkspace", Tags: []string{"workspace"}, Method: http.MethodGet, Path: "/api/v0/workspaces/{slug}"},
		{ID: "getMe", Tags: []string{"identity"}, Method: http.MethodGet, Path: "/api/v0/me"},
		{ID: "getWorkspaceInvitation", Tags: []string{"workspace"}, Method: http.MethodGet, Path: "/api/v0/workspace-invitations/{invitation_id}",
			Public: true}}
	exempt := matrixExemptions{modules: []string{"identity"}, public: map[string]string{"getWorkspaceInvitation": "TestOfItsOwn"}}
	own := map[string]bool{"TestOfItsOwn": true}
	// exemptPublic is exempt, but with the public exemptions public.
	exemptPublic := func(public ...string) matrixExemptions {
		e := matrixExemptions{modules: exempt.modules, public: map[string]string{}}
		for _, id := range public {
			e.public[id] = "TestOfItsOwn"
		}
		return e
	}
	// checkSlug is a row of checkWorkspaceSlug, whose {slug} is no
	// workspace; notTarget is exempt with its path listed as not a target.
	checkSlug := apitest.Operation{ID: "checkWorkspaceSlug", Tags: []string{"workspace"}, Method: http.MethodGet,
		Path: "/api/v0/workspace-slugs/{slug}"}
	checks := matrixRow{op: "checkWorkspaceSlug", request: sameRequest(http.MethodGet, "/api/v0/workspace-slugs/acme", ""), cells: every(cellOK)}
	notTarget := exemptPublic("getWorkspaceInvitation")
	notTarget.notTargets = map[string]string{checkSlug.Path: "a slug asked about"}
	// accepts is a row that sends POST to a path answers lists as not a
	// target, and does not say it writes: the listing spares its cells the
	// target check only, never the check of a write.
	accept := apitest.Operation{ID: "acceptWorkspaceInvitation", Tags: []string{"workspace"}, Method: http.MethodPost,
		Path: "/api/v0/workspace-invitations/{invitation_id}/accept"}
	accepts := matrixRow{op: accept.ID, request: sameRequest(http.MethodPost, "/api/v0/workspace-invitations/"+uuid.Nil().String()+"/accept",
		`{"token":"t"}`), cells: every(cellOK)}
	answers := exemptPublic("getWorkspaceInvitation")
	answers.notTargets = map[string]string{accept.Path: "account level"}
	var unknown []string
	for _, c := range workspaceColumns {
		unknown = append(unknown, fmt.Sprintf("row checkWorkspaceSlug, %s: {slug} is no target the matrix knows: "+
			"list /api/v0/workspace-slugs/{slug} as not a target, with its reason", c))
	}
	getInvitation := matrixRow{op: "getWorkspaceInvitation", cells: every(cellOK), request: func(c caller, s seeded) (string, string, string) {
		return http.MethodGet, "/api/v0/workspace-invitations/" + s.invitation(workspaceOf(c), "newcomer@example.com").String() + "?token=t", ""
	}}
	s := newSeeded().in(t)
	get := toWorkspace(http.MethodGet, "", "")
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
	// A matching matrix; a query is no part of the path; a membership named
	// by its id, in each column's workspace.
	lists := apitest.Operation{ID: "listWorkspaces", Tags: []string{"workspace"}, Method: http.MethodGet, Path: "/api/v0/workspaces"}
	paged := matrixRow{op: "listWorkspaces", request: sameRequest(http.MethodGet, "/api/v0/workspaces?page=2", ""), cells: every(cellOK)}
	membership := apitest.Operation{ID: "updateWorkspaceMember", Tags: []string{"workspace"}, Method: http.MethodPatch,
		Path: "/api/v0/workspace-members/{workspace_member_id}"}
	demotes := matrixRow{op: "updateWorkspaceMember", write: true, request: toMembership(anotherMember, `{"role":5}`), cells: every(cellOK)}
	if got := matrixViolations(append(ops, lists, membership), exempt, []matrixRow{row, paged, demotes}, s, own); len(got) != 0 {
		t.Fatalf("a matching matrix: %q, want none", got)
	}
	// deletedNames is demotes, but the deleted workspace's column names the
	// membership of who in slug.
	deletedNames := func(slug string, who caller) matrixRow {
		r := demotes
		r.request = toMembership(func(c caller) (string, caller) {
			if c == callerDeleted {
				return slug, who
			}
			return anotherMember(c)
		}, `{"role":5}`)
		return r
	}
	// guestNames is demotes, but the guest's cell sends path.
	guestNames := func(path string) matrixRow {
		r := demotes
		r.request = func(c caller, s seeded) (string, string, string) {
			if c == callerGuest {
				return http.MethodPatch, path, `{"role":5}`
			}
			return demotes.request(c, s)
		}
		return r
	}
	preferences := apitest.Operation{ID: "getWorkspacePreferences", Tags: []string{"workspace"}, Method: http.MethodGet,
		Path: "/api/v0/me/workspaces/{slug}/preferences"}
	otherPreferences := matrixRow{op: "getWorkspacePreferences", cells: every(cellOK), request: func(c caller, s seeded) (string, string, string) {
		if c == callerGuest {
			return http.MethodGet, "/api/v0/me/workspaces/other/preferences", ""
		}
		return toPreferences(http.MethodGet, "")(c, s)
	}}
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
		name   string
		ops    []apitest.Operation
		exempt matrixExemptions
		rows   []matrixRow
		want   []string
	}{
		{"an operation without a row", append(ops, apitest.Operation{ID: "updateWorkspace", Tags: []string{"workspace"}}), exempt,
			[]matrixRow{row}, []string{"operation updateWorkspace, tagged [workspace], has no row"}},
		{"a public operation without a row or an exemption", ops, exemptPublic(), []matrixRow{row},
			[]string{"operation getWorkspaceInvitation, tagged [workspace], has no row"}},
		{"a public exemption of no operation", ops, exemptPublic("getWorkspaceInvitation", "getInvitation"), []matrixRow{row},
			[]string{"the public exemption getInvitation names no operation of the contract"}},
		{"a public exemption of an operation that needs a token",
			append(ops, apitest.Operation{ID: "deleteWorkspace", Tags: []string{"workspace"}, Method: http.MethodDelete, Path: "/api/v0/workspaces/{slug}"}),
			exemptPublic("getWorkspaceInvitation", "deleteWorkspace"), []matrixRow{row},
			[]string{"operation deleteWorkspace is exempt as public, but needs a token"}},
		{"a public exemption with a row", ops, exempt, []matrixRow{row, getInvitation},
			[]string{"operation getWorkspaceInvitation is exempt as public, and has a row"}},
		{"a parameter the matrix does not know", append(ops, checkSlug), exempt, []matrixRow{row, checks}, unknown},
		{"a path listed as not a target", append(ops, checkSlug), notTarget, []matrixRow{row, checks}, nil},
		{"a not-target path of no operation", ops, notTarget, []matrixRow{row},
			[]string{"the not-target path /api/v0/workspace-slugs/{slug} is no operation's"}},
		{"an operation of a module no list names", append(ops, apitest.Operation{ID: "listProjects", Tags: []string{"project"}}), exempt,
			[]matrixRow{row}, []string{"operation listProjects, tagged [project], has no row"}},
		{"an operation without a tag", append(ops, apitest.Operation{ID: "getHealth"}), exempt,
			[]matrixRow{row}, []string{"operation getHealth, tagged [], has no row"}},
		{"an operation with an exempt tag and another", append(ops, apitest.Operation{ID: "getMyWorkspaces", Tags: []string{"identity", "workspace"}}), exempt,
			[]matrixRow{row}, []string{"operation getMyWorkspaces, tagged [identity workspace], has no row"}},
		{"a row of no operation", ops, exempt, []matrixRow{row, {op: "renameWorkspace", request: get, cells: every(cellOK)}},
			[]string{"row renameWorkspace names no operation of the contract"}},
		{"a row without a cell", ops, exempt, []matrixRow{partial}, []string{
			"row getWorkspace has no cell for member", "row getWorkspace has no cell for guest", "row getWorkspace has no cell for never a member",
			"row getWorkspace has no cell for removed", "row getWorkspace has no cell for workspace deleted",
		}},
		{"a row that sends another operation", ops, exempt, []matrixRow{{op: "getWorkspace", request: sameRequest(http.MethodGet, "/api/v0/workspaces", ""),
			cells: every(cellOK)}}, mislabelled},
		{"a cell of another method", ops, exempt, []matrixRow{posting},
			[]string{"row getWorkspace, guest: POST /api/v0/workspaces/acme is not GET /api/v0/workspaces/{slug}"}},
		{"a cell of another path", ops, exempt, []matrixRow{guestSends(http.MethodGet, "/api/v0/workspace-slugs/acme")},
			[]string{"row getWorkspace, guest: GET /api/v0/workspace-slugs/acme is not GET /api/v0/workspaces/{slug}"}},
		{"a cell with an empty parameter", ops, exempt, []matrixRow{guestSends(http.MethodGet, "/api/v0/workspaces/")},
			[]string{"row getWorkspace, guest: GET /api/v0/workspaces/ is not GET /api/v0/workspaces/{slug}"}},
		{"a cell with a longer path", ops, exempt, []matrixRow{guestSends(http.MethodGet, "/api/v0/workspaces/acme/members")},
			[]string{"row getWorkspace, guest: GET /api/v0/workspaces/acme/members is not GET /api/v0/workspaces/{slug}"}},
		{"a write without write", append(ops, creates), exempt, []matrixRow{row,
			{op: "createWorkspace", request: sameRequest(http.MethodPost, "/api/v0/workspaces", `{}`), cells: every(cellCreated)}},
			[]string{"row createWorkspace sends POST without write: its cells could run on the reads' copy"}},
		{"a write without write on a path listed as not a target", append(ops, accept), answers, []matrixRow{row, accepts},
			[]string{"row acceptWorkspaceInvitation sends POST without write: its cells could run on the reads' copy"}},
		{"a cell that targets another column's workspace", ops, exempt,
			[]matrixRow{{op: "getWorkspace", request: sameRequest(http.MethodGet, "/api/v0/workspaces/acme", ""), cells: every(cellOK)}},
			[]string{"row getWorkspace, workspace deleted: targets the workspace acme, not its column's gone"}},
		{"a cell of one's settings in another column's workspace", append(ops, preferences), exempt, []matrixRow{row, otherPreferences},
			[]string{"row getWorkspacePreferences, guest: targets the workspace other, not its column's acme"}},
		{"a membership of another column's workspace", append(ops, membership), exempt, []matrixRow{row, deletedNames("acme", callerMember)},
			[]string{"row updateWorkspaceMember, workspace deleted: {workspace_member_id} " + s.membership("acme", callerMember).String() +
				" is no row seeded under its column's workspace gone"}},
		{"an id no seeded row has", append(ops, membership), exempt, []matrixRow{row, guestNames("/api/v0/workspace-members/" + uuid.Nil().String())},
			[]string{"row updateWorkspaceMember, guest: {workspace_member_id} " + uuid.Nil().String() + " is no row seeded under its column's workspace acme"}},
	}
	for _, tt := range tests {
		if got := matrixViolations(tt.ops, tt.exempt, tt.rows, s, own); !slices.Equal(got, tt.want) {
			t.Errorf("%s: %q, want %q", tt.name, got, tt.want)
		}
	}
	// A public exemption whose test the package does not have: renamed or
	// deleted, the operation would have neither a row nor its test.
	if got := matrixViolations(ops, exempt, []matrixRow{row}, s, map[string]bool{}); !slices.Equal(got, []string{
		"the public exemption getWorkspaceInvitation names TestOfItsOwn, which no test of the package is"}) {
		t.Errorf("a public exemption of no test: %q", got)
	}
	// A membership never seeded fails the test at once, and names it: its id
	// would name no row, and an outsider's cell would pass on its 404.
	failed := fatalOf(func(tb testing.TB) {
		matrixViolations(append(ops, membership), exempt, []matrixRow{deletedNames("acme", callerNever)}, newSeeded().in(tb), own)
	})
	if want := "no membership of acme by never a member is seeded"; failed != want {
		t.Errorf("a membership never seeded: failed with %q, want %q", failed, want)
	}
	// An exempt entry that no operation carries: stale, or misspelled, when
	// the module it meant has operations without rows besides.
	for _, tt := range []struct {
		name    string
		modules []string
		want    []string
	}{
		{"a stale exempt entry", []string{"identity", "instance"}, []string{"the exempt module instance has no operation"}},
		{"a misspelled exempt entry", []string{"identities"},
			[]string{"operation getMe, tagged [identity], has no row", "the exempt module identities has no operation"}},
	} {
		if got := matrixViolations(ops, matrixExemptions{modules: tt.modules, public: exempt.public}, []matrixRow{row}, s, own); !slices.Equal(got, tt.want) {
			t.Errorf("%s: %q, want %q", tt.name, got, tt.want)
		}
	}
}

// fatalOf runs f on a goroutine of its own with a testing.TB whose Fatalf
// records the message and ends that goroutine, as testing.T's does, without
// failing the test; it returns the message, "" when f did not fail.
func fatalOf(f func(testing.TB)) string {
	p := &fatalProbe{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		f(p)
	}()
	<-done
	return p.message
}

type fatalProbe struct {
	testing.TB
	message string
}

func (*fatalProbe) Helper() {}

func (p *fatalProbe) Fatalf(format string, args ...any) {
	p.message = fmt.Sprintf(format, args...)
	runtime.Goexit()
}
