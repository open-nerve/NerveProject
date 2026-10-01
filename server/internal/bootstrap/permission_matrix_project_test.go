package bootstrap

import (
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"
)

// The project module's rows of the permission matrix (M3 design 9.2).

var (
	cellProjectNotFound = cell{http.StatusNotFound, "project.not_found"}
	cellProjectArchived = cell{http.StatusConflict, "project.archived"}
)

// ofProject are the cells of a project-level row: the answers of PA, PM,
// PG, PM+WA, WA- and WM-公, and project.not_found for the columns that do
// not see their project: WM-私, WG-, P-前 and X.
func ofProject(pa, pm, pg, pmwa, wa, wm cell) map[caller]cell {
	return map[caller]cell{callerProjectAdmin: pa, callerProjectMember: pm, callerProjectGuest: pg, callerMemberAndAdmin: pmwa,
		callerAdminOnly: wa, callerMemberPublic: wm, callerMemberPrivate: cellProjectNotFound, callerGuestOnly: cellProjectNotFound,
		callerBefore: cellProjectNotFound, callerNever: cellProjectNotFound, callerRemoved: cellProjectNotFound,
		callerDeleted: cellProjectNotFound}
}

// toProject is the request of a row whose callers each send method to the
// path under the project their column targets.
func toProject(method, path, body string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return method, "/api/v0/projects/" + s.project(projectOf(c)).String() + path, body
	}
}

func projectMatrixRows() []matrixRow {
	return []matrixRow{
		{op: "listProjects", request: toWorkspace(http.MethodGet, "/projects", ""), cells: inWorkspace(cellOK, cellOK, cellOK),
			check: listsTheProjects(false)},
		{op: "listProjects", variant: "archived", request: toWorkspace(http.MethodGet, "/projects?archived=true", ""),
			cells: inWorkspace(cellOK, cellOK, cellOK), check: listsTheProjects(true)},
		{op: "createProject", write: true, request: toWorkspace(http.MethodPost, "/projects", `{"name":"New","identifier":"new"}`),
			cells: inWorkspace(cellCreated, cellCreated, cellForbidden), check: createsItsProject},
		// A lead who is no member of the workspace is refused after the
		// decision (M3 design 3.6 convention 3): the guest is still refused
		// as a guest, and learns nothing of the lead.
		{op: "createProject", variant: "a lead who is no member", write: true,
			request: toWorkspace(http.MethodPost, "/projects", `{"name":"New","identifier":"NEW","project_lead_id":"`+uuid.Nil().String()+`"}`),
			cells:   inWorkspace(cellValidationFailed, cellValidationFailed, cellForbidden)},
		// acme has WEB, and web is WEB in any case; gone has it too, deleted
		// with gone.
		{op: "checkProjectIdentifier", variant: "taken", request: toWorkspace(http.MethodGet, "/project-identifiers/web", ""),
			cells: inWorkspace(cellOK, cellOK, cellForbidden), check: identifierAvailable(false)},
		{op: "checkProjectIdentifier", variant: "free", request: toWorkspace(http.MethodGet, "/project-identifiers/NEW", ""),
			cells: inWorkspace(cellOK, cellOK, cellForbidden), check: identifierAvailable(true)},
		{op: "getProject", columns: projectColumns, request: toProject(http.MethodGet, "", ""),
			cells: ofProject(cellOK, cellOK, cellOK, cellOK, cellOK, cellOK), check: readsItsProject},
		{op: "getProject", variant: "archived", columns: archivedColumns, request: toProject(http.MethodGet, "", ""),
			cells: map[caller]cell{callerArchivedAdmin: cellOK}, check: readsItsProject},
		// The project's admins, and its members who are the workspace's
		// admins (M3 design 3.4): PM+WA is a member of the project and WA- is
		// not, so the row parts the workspace's admin who joined from the one
		// who did not.
		{op: "updateProject", write: true, columns: projectColumns, request: toProject(http.MethodPatch, "", `{"name":"Renamed"}`),
			cells: ofProject(cellOK, cellForbidden, cellForbidden, cellOK, cellForbidden, cellForbidden), check: renamesItsProject},
		// A lead who is no member of the project is refused after the
		// decision: who may not change the project learns nothing of the lead.
		{op: "updateProject", variant: "a lead who is no member", write: true, columns: projectColumns,
			request: toProject(http.MethodPatch, "", `{"project_lead_id":"`+uuid.Nil().String()+`"}`),
			cells:   ofProject(cellValidationFailed, cellForbidden, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden)},
		{op: "updateProject", variant: "archived", write: true, columns: archivedColumns, request: toProject(http.MethodPatch, "", `{"name":"Renamed"}`),
			cells: map[caller]cell{callerArchivedAdmin: cellProjectArchived}},
		// As updateProject (M3 design 3.4); an archived project archives
		// again, and an unarchived one unarchives.
		{op: "archiveProject", write: true, columns: projectColumns, request: toProject(http.MethodPost, "/archive", ""),
			cells: ofProject(cellOK, cellForbidden, cellForbidden, cellOK, cellForbidden, cellForbidden), check: archivesItsProject(true)},
		{op: "archiveProject", variant: "archived", write: true, columns: archivedColumns, request: toProject(http.MethodPost, "/archive", ""),
			cells: map[caller]cell{callerArchivedAdmin: cellOK}, check: archivesItsProject(true)},
		{op: "unarchiveProject", write: true, columns: projectColumns, request: toProject(http.MethodPost, "/unarchive", ""),
			cells: ofProject(cellOK, cellForbidden, cellForbidden, cellOK, cellForbidden, cellForbidden), check: archivesItsProject(false)},
		{op: "unarchiveProject", variant: "archived", write: true, columns: archivedColumns, request: toProject(http.MethodPost, "/unarchive", ""),
			cells: map[caller]cell{callerArchivedAdmin: cellOK}, check: archivesItsProject(false)},
		// As updateProject; an archived project is deleted as any other.
		{op: "deleteProject", write: true, columns: projectColumns, request: toProject(http.MethodDelete, "", ""),
			cells: ofProject(cellNoContent, cellForbidden, cellForbidden, cellNoContent, cellForbidden, cellForbidden)},
		{op: "deleteProject", variant: "archived", write: true, columns: archivedColumns, request: toProject(http.MethodDelete, "", ""),
			cells: map[caller]cell{callerArchivedAdmin: cellNoContent}},
	}
}

// archivesItsProject: the column's project, with the caller's role in it,
// archived when archived is true, as of its last change, and not archived
// otherwise.
func archivesItsProject(archived bool) func(t *testing.T, c caller, s seeded, answer string) {
	return func(t *testing.T, c caller, s seeded, answer string) {
		var p struct {
			ID         uuid.UUID  `json:"id"`
			MemberRole *int       `json:"member_role"`
			ArchivedAt *time.Time `json:"archived_at"`
			UpdatedAt  time.Time  `json:"updated_at"`
		}
		decodeAnswer(t, answer, &p)
		if p.ID != s.project(projectOf(c)) || p.MemberRole == nil || *p.MemberRole != memberRoles[c] || (p.ArchivedAt != nil) != archived ||
			(archived && !p.ArchivedAt.Equal(p.UpdatedAt)) {
			t.Errorf("%s archives (%v) %s; want %s, his role %d, archived %v as of its change", c, archived, answer, projectOf(c), memberRoles[c],
				archived)
		}
	}
}

// memberRoles are the project roles of the columns that are their
// project's members.
var memberRoles = map[caller]int{callerProjectAdmin: 20, callerProjectMember: 15, callerProjectGuest: 5, callerMemberAndAdmin: 15,
	callerArchivedAdmin: 20}

// renamesItsProject: the column's project, renamed, with the caller's role
// in it.
func renamesItsProject(t *testing.T, c caller, s seeded, answer string) {
	var p struct {
		ID         uuid.UUID `json:"id"`
		Name       string    `json:"name"`
		MemberRole *int      `json:"member_role"`
	}
	decodeAnswer(t, answer, &p)
	if p.ID != s.project(projectOf(c)) || p.Name != "Renamed" || p.MemberRole == nil || *p.MemberRole != memberRoles[c] {
		t.Errorf("%s renames %s; want %s renamed, his role %d", c, answer, projectOf(c), memberRoles[c])
	}
}

// readsItsProject: the column's project, with the caller's role in it, null
// for who is not its member, and its archived_at, set for the archived one
// alone.
func readsItsProject(t *testing.T, c caller, s seeded, answer string) {
	var p struct {
		ID         uuid.UUID  `json:"id"`
		MemberRole *int       `json:"member_role"`
		ArchivedAt *time.Time `json:"archived_at"`
	}
	decodeAnswer(t, answer, &p)
	role, member := memberRoles[c]
	if p.ID != s.project(projectOf(c)) || (p.MemberRole != nil) != member || (member && *p.MemberRole != role) ||
		(p.ArchivedAt != nil) != (c == callerArchivedAdmin) {
		t.Errorf("%s reads %s; want %s, his role %d (0: none), archived only for the archived project", c, answer, projectOf(c), role)
	}
}

// createsItsProject: the project is created as asked, with its caller as
// its admin and only member.
func createsItsProject(t *testing.T, c caller, _ seeded, answer string) {
	var p struct {
		Identifier string      `json:"identifier"`
		MemberRole *int        `json:"member_role"`
		MemberIDs  []uuid.UUID `json:"member_ids"`
	}
	decodeAnswer(t, answer, &p)
	if p.Identifier != "NEW" || p.MemberRole == nil || *p.MemberRole != 20 || len(p.MemberIDs) != 1 {
		t.Errorf("%s creates %s; want NEW, with him its admin and only member", c, answer)
	}
}

// listsTheProjects: acme's projects the column's caller sees, the archived
// one alone or the others (9.2): every one to the admin, the public one to
// the member, those he is a member of to the guest; by name, as every
// place in a sidebar is the same.
func listsTheProjects(archived bool) func(t *testing.T, c caller, _ seeded, answer string) {
	return func(t *testing.T, c caller, _ seeded, answer string) {
		var list struct {
			Data []struct {
				Name string `json:"name"`
			} `json:"data"`
		}
		decodeAnswer(t, answer, &list)
		var got []string
		for _, p := range list.Data {
			got = append(got, p.Name)
		}
		want := map[caller][]string{callerAdmin: {"Secret", "Web"}, callerMember: {"Web"}, callerGuest: {"Secret", "Web"}}[c]
		if archived {
			want = map[caller][]string{callerAdmin: {"Old"}, callerMember: {"Old"}}[c]
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s lists %q, want %q", c, got, want)
		}
	}
}

// identifierAvailable: the answer is want.
func identifierAvailable(want bool) func(t *testing.T, c caller, _ seeded, answer string) {
	return func(t *testing.T, c caller, _ seeded, answer string) {
		var a struct {
			Available bool `json:"available"`
		}
		decodeAnswer(t, answer, &a)
		if a.Available != want {
			t.Errorf("%s is answered %s, want available %v", c, answer, want)
		}
	}
}
