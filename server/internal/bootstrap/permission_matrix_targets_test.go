package bootstrap

import (
	"fmt"
	"slices"
	"strings"
	"uuid"
)

// Where a cell's request points: the path of its operation, and the target
// of its column.

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

// ofAProjectTable reports whether c is a column of a project table
// (projectTables).
func ofAProjectTable(c caller) bool {
	return slices.ContainsFunc(projectTables, func(table []caller) bool { return slices.Contains(table, c) })
}

// underProject are the rows under a project that a path names by their
// id, by their parameter: what each is called, and the seeded rows among
// which it must be, keyed by their project's key.
var underProject = map[string]struct {
	what string
	rows func(s seeded) map[string]uuid.UUID
}{
	"{project_member_id}": {"membership", func(s seeded) map[string]uuid.UUID { return s.projectMembers }},
	"{state_id}":          {"state", func(s seeded) map[string]uuid.UUID { return s.states }},
	"{label_id}":          {"label", func(s seeded) map[string]uuid.UUID { return s.labels }},
}

// targetViolation is what is wrong with where path, a path of pattern that
// a cell of the column c sends, points; "" when nothing. A workspace named
// by its slug ({slug} right after workspaces) must be workspaceOf(c). A
// project named by its id ({project_id}) must be projectOf(c)'s, and a row
// under a project (underProject: a project membership, a state, a label)
// one seeded in projectOf(c), each from a column of a project table
// (projectTables): a project's operation in a workspace-level row, or the
// only admin's, would leave the project level's own columns unasked.
// Any other row named by its id (a parameter ending in _id) must be a row
// of s under workspaceOf(c): a cell of the deleted workspace's column that
// named acme would get the 404 of a workspace its caller is not in, and
// pass whether deleted workspaces are hidden or not. A parameter passOver
// reports is passed over, each other checked. Any other parameter is
// reported: a parameter that is no column's target is listed as such, with
// its reason (matrixExemptions.notTargets), and not given here.
func targetViolation(pattern, path string, c caller, s seeded, passOver func(param string) bool) string {
	path, _, _ = strings.Cut(path, "?")
	want, got := strings.Split(pattern, "/"), strings.Split(path, "/")
	for i, segment := range want {
		switch {
		case !strings.HasPrefix(segment, "{") || !strings.HasSuffix(segment, "}"):
		case passOver(segment):
		case segment == "{project_id}":
			if !ofAProjectTable(c) {
				return "{project_id} from a column of no project table (projectTables): a project's row names its columns"
			}
			if id := s.project(projectOf(c)).String(); got[i] != id {
				return fmt.Sprintf("{project_id} %s is not its column's project %s, %s", got[i], projectOf(c), id)
			}
		case underProject[segment].rows != nil:
			if !ofAProjectTable(c) {
				return segment + " from a column of no project table (projectTables): a project's row names its columns"
			}
			id, err := uuid.Parse(got[i])
			if key, isRow := projectOfRow(underProject[segment].rows(s), id); err != nil || !isRow || key != projectOf(c) {
				return fmt.Sprintf("%s %s is no %s seeded in its column's project %s", segment, got[i], underProject[segment].what, projectOf(c))
			}
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
