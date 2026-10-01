package bootstrap

import (
	"fmt"
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

// targetViolation is what is wrong with where path, a path of pattern that
// a cell of the column c sends, points; "" when nothing. A workspace named
// by its slug ({slug} right after workspaces) must be workspaceOf(c), a
// project named by its id ({project_id}) must be projectOf(c)'s, and any
// other row named by its id (a parameter ending in _id) must be a row of s
// under workspaceOf(c): a cell of the deleted workspace's column that named
// acme would get the 404 of a workspace its caller is not in, and pass
// whether deleted workspaces are hidden or not. A parameter passOver
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
			if id := s.project(projectOf(c)).String(); got[i] != id {
				return fmt.Sprintf("{project_id} %s is not its column's project %s, %s", got[i], projectOf(c), id)
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
