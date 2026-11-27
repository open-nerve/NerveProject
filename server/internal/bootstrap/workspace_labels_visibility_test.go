package bootstrap

import (
	"net/http"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// The workspace's list of labels and each project's agree (M7/P6b spec
// 2.1): for every account of the matrix, the labels listWorkspaceLabels
// lists of acme are exactly those listLabels lists of each of acme's
// unarchived projects, by project in the order of their ids, a project that
// answers him 403 or 404 adding none; partingStates' memberships among
// them. Unlike listStates, listLabels lists an archived project's labels as
// any other's (M3 design 3.19), and the workspace's list leaves them out:
// the archived project is not joined, and PA, who lists its labels, finds
// none of them in acme's list. The workspace's list reads the membership in
// its query, each project's list in access's rule: here the two cannot
// part. Some accounts list no label, some both unarchived projects'.
func TestListingWorkspaceLabelsListsEachProjects(t *testing.T) {
	d := prepareMatrix(t)
	contract := apitest.Load(t)
	base := startApp(t, d.config(t, pgtest.NewDatabaseFrom(t, d.url), nil), migrations.FS())
	s := d.seeded.in(t)
	archived := labelIDs(t, contract, base+"/api/v0/projects/"+s.project("acme/archived").String()+"/labels", d.tokens[callerProjectAdmin])
	if len(archived) != len(matrixLabels) {
		t.Fatalf("PA lists %v of the archived project; the test needs its labels listed", archived)
	}
	var sizes []int
	for _, account := range matrixAccounts {
		token := d.tokens[account]
		listed := labelIDs(t, contract, base+"/api/v0/workspaces/acme/labels", token)
		var each []uuid.UUID
		for _, key := range byProjectID(s, "acme/public", "acme/private") {
			each = append(each, labelIDs(t, contract, base+"/api/v0/projects/"+s.project(key).String()+"/labels", token)...)
		}
		if !slices.Equal(listed, each) {
			t.Errorf("%s: acme's list %v, each unarchived project's %v", account, listed, each)
		}
		sizes = append(sizes, len(each))
	}
	if slices.Min(sizes) != 0 || slices.Max(sizes) != 2*len(matrixLabels) {
		t.Errorf("the accounts list %v labels; want some none and some both unarchived projects' %d", sizes, 2*len(matrixLabels))
	}
}

// Each workspace's list of labels holds its own projects' alone (M7/P6b
// spec 2.1), on memberWorld, whose bob and carol are members of acme's Web
// and Ops and of beta's Lab: for each of its accounts, acme's list is
// exactly what listLabels lists of Web and of Ops, by project id, and
// beta's what it lists of Lab. A list that took in the projects of another
// workspace would hold Lab's Notes in acme's, Web's Bug and Ops's Runbook in
// beta's; Web has a deleted label, Dropped, which alice created and
// deleted, and which neither list holds.
func TestWorkspaceLabelsKeepToTheirWorkspace(t *testing.T) {
	w := newMemberWorld(t)
	var dropped uuid.UUID
	for _, l := range []struct {
		by, name string
		project  uuid.UUID
	}{{"alice", "Bug", w.web}, {"alice", "Dropped", w.web}, {"alice", "Runbook", w.ops}, {"bob", "Notes", w.lab}} {
		status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/projects/"+l.project.String()+"/labels", w.tokens[l.by],
			`{"name":"`+l.name+`"}`)
		if status != http.StatusCreated {
			t.Fatalf("%s's creating %s = %d %s", l.by, l.name, status, body)
		}
		var created struct {
			ID uuid.UUID `json:"id"`
		}
		if decodeAnswer(t, body, &created); l.name == "Dropped" {
			dropped = created.ID
		}
	}
	if status, body := call(t, w.contract, http.MethodDelete, w.base+"/api/v0/labels/"+dropped.String(), w.tokens["alice"],
		""); status != http.StatusNoContent {
		t.Fatalf("alice's deleting Dropped = %d %s", status, body)
	}
	acme := slices.SortedFunc(slices.Values([]uuid.UUID{w.web, w.ops}), func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) })
	for _, p := range []uuid.UUID{w.web, w.ops, w.lab} {
		if len(labelIDs(t, w.contract, w.base+"/api/v0/projects/"+p.String()+"/labels", w.tokens["bob"])) != 1 {
			t.Fatalf("bob does not list the one label of %s; the test needs him to list each project's", p)
		}
	}
	for _, name := range []string{"alice", "bob", "carol", "dave", "erin", "gina"} {
		for _, ws := range []struct {
			slug     string
			projects []uuid.UUID
		}{{"acme", acme}, {"beta", []uuid.UUID{w.lab}}} {
			listed := labelIDs(t, w.contract, w.base+"/api/v0/workspaces/"+ws.slug+"/labels", w.tokens[name])
			var each []uuid.UUID
			for _, p := range ws.projects {
				each = append(each, labelIDs(t, w.contract, w.base+"/api/v0/projects/"+p.String()+"/labels", w.tokens[name])...)
			}
			if !slices.Equal(listed, each) || slices.Contains(listed, dropped) {
				t.Errorf("%s: %s's list %v, each project's %v; Dropped, %s, is deleted", name, ws.slug, listed, each, dropped)
			}
		}
	}
}

// labelIDs are the ids of the labels GET url answers token, in its order;
// none when it answers 403 or 404, a project he sees and is not a member
// of, or a project or workspace he does not see. Any other answer fails the
// test.
func labelIDs(t *testing.T, contract *apitest.Contract, url, token string) []uuid.UUID {
	t.Helper()
	status, body := call(t, contract, http.MethodGet, url, token, "")
	switch status {
	case http.StatusForbidden, http.StatusNotFound:
		return nil
	case http.StatusOK:
	default:
		t.Fatalf("GET %s = %d %s; want 200, 403 or 404", url, status, body)
	}
	var list struct {
		Data []struct {
			ID uuid.UUID `json:"id"`
		} `json:"data"`
	}
	decodeAnswer(t, body, &list)
	ids := make([]uuid.UUID, len(list.Data))
	for i, l := range list.Data {
		ids[i] = l.ID
	}
	return ids
}
