package bootstrap

import (
	"encoding/json"
	"net/http"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

const workspaceIssuesPath = "/api/v0/workspaces/acme/issues"

// workspaceListWorld is TestWorkspaceIssueList's world: issueWorld's acme
// and Web, and four more projects of acme, alice's (newWorkspaceListWorld).
// Its work items are named by project and case: a… Alpha's, w… Web's.
type workspaceListWorld struct {
	issueWorld
	alpha, open, shelf, former, ops, alphaDone uuid.UUID
	// Alpha's: a1 alice's, walt subscribed by hand; a2 bob's, assigned to
	// walt, Ops, urgent, dated; a3 walt's, Done, mentioning bob; a4 walt's,
	// assigned to him, archived; a5 walt's, assigned to him, deleted; a6
	// walt's submission, in Triage; a7 bob's, assigned to bob.
	a1, a2, a3, a4, a5, a6, a7 uuid.UUID
	// Web's: w1 alice's, assigned to walt, Bug, urgent, dated, mentioning
	// bob; w2 walt's, assigned to him, Bug, high; w3 walt's, a child of w1;
	// w4 bob's, assigned to bob.
	w1, w2, w3, w4 uuid.UUID
	// shelved, walt's in Shelf; left, walt's in Former; outside, alice's in
	// Open; each assigned to walt but outside's.
	shelved, left, outside uuid.UUID
}

// newWorkspaceListWorld builds the world through the API (M7 P3a spec 4):
//   - Web, public: walt a member while he makes w2 and w3 and is assigned
//     w1, then its guest, who sees only his own (OwnOnly);
//   - Alpha, private: walt and bob its members, wanda, acme's admin, its
//     admin (a workspace admin joins a project as its admin); its intake on;
//   - Open, public: none of them a member, not even wanda;
//   - Shelf: walt a member, the project archived after he made shelved;
//   - Former: walt a member who left it after he made left.
//
// The work items are made in the order a1, w1, w2, a2, w4, w3, a3, shelved,
// left, outside, a4, a5, a6, a7, so that walt's list by -created_at is a7,
// a3, w3, a2, w2, a1: two projects alternate in it, and w4 and w1, which
// he does not see, fall between his rows. Walt is subscribed to each work
// item he makes or is assigned to, as the API subscribes them, and to a1,
// which he subscribes to himself.
func newWorkspaceListWorld(t *testing.T) workspaceListWorld {
	t.Helper()
	w := workspaceListWorld{issueWorld: newIssueWorld(t)}
	token := w.tokens["alice"]
	w.alpha = createdProjectFrom(t, w.contract, w.base, token, "acme", `{"name":"Alpha","identifier":"ALP","network":0}`)
	w.open = createdProject(t, w.contract, w.base, token, "acme", "Open", "OPN")
	w.shelf = createdProject(t, w.contract, w.base, token, "acme", "Shelf", "SHF")
	w.former = createdProject(t, w.contract, w.base, token, "acme", "Former", "FOR")
	for _, m := range []struct {
		name    string
		project uuid.UUID
		role    shared.Role
	}{{"walt", w.web, shared.RoleMember}, {"walt", w.alpha, shared.RoleMember}, {"bob", w.alpha, shared.RoleMember},
		{"wanda", w.alpha, shared.RoleAdmin}, {"walt", w.shelf, shared.RoleMember}, {"walt", w.former, shared.RoleMember}} {
		w.makeMember(t, m.name, m.project, m.role)
	}
	var label struct{ ID uuid.UUID }
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.alpha.String()+"/labels", `{"name":"Ops"}`, http.StatusCreated), &label)
	w.ops, w.alphaDone = label.ID, stateID(t, w.pool, w.alpha, "Done")
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.alpha.String(), `{"intake_view":true}`, http.StatusOK)
	walt, bob := idArray(w.ids["walt"]), idArray(w.ids["bob"])
	mention, _ := json.Marshal(mentionHTML(w.ids["bob"]))
	dated := `,"priority":"urgent","start_date":"2026-10-12","target_date":"2026-10-15"`
	w.a1 = w.newWorkItemIn(t, w.alpha, `{"name":"Alpha plain"}`)
	w.w1 = w.newWorkItemIn(t, w.web, `{"name":"Web hidden","assignee_ids":`+walt+`,"label_ids":`+idArray(w.bug)+dated+`,"description_html":`+string(mention)+`}`)
	w.w2 = w.newWorkItemBy(t, "walt", w.web, `{"name":"Web own","assignee_ids":`+walt+`,"label_ids":`+idArray(w.bug)+`,"priority":"high"}`)
	w.a2 = w.newWorkItemBy(t, "bob", w.alpha, `{"name":"Alpha assigned","assignee_ids":`+walt+`,"label_ids":`+idArray(w.ops)+dated+`}`)
	w.w4 = w.newWorkItemBy(t, "bob", w.web, `{"name":"Web bob","assignee_ids":`+bob+`}`)
	w.w3 = w.newWorkItemBy(t, "walt", w.web, `{"name":"Web child","parent_id":"`+w.w1.String()+`"}`)
	w.a3 = w.newWorkItemBy(t, "walt", w.alpha, `{"name":"Alpha done","state_id":"`+w.alphaDone.String()+`","description_html":`+string(mention)+`}`)
	w.shelved = w.newWorkItemBy(t, "walt", w.shelf, `{"name":"Shelved","assignee_ids":`+walt+`}`)
	w.left = w.newWorkItemBy(t, "walt", w.former, `{"name":"Left","assignee_ids":`+walt+`}`)
	w.outside = w.newWorkItemIn(t, w.open, `{"name":"Open"}`)
	w.a4 = w.newWorkItemBy(t, "walt", w.alpha, `{"name":"Alpha archived","state_id":"`+w.alphaDone.String()+`","assignee_ids":`+walt+`}`)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+w.a4.String()+"/archive", "", http.StatusOK)
	w.a5 = w.newWorkItemBy(t, "walt", w.alpha, `{"name":"Alpha deleted","assignee_ids":`+walt+`}`)
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+w.a5.String(), "", http.StatusNoContent)
	var submitted struct{ ID uuid.UUID }
	decodeAnswer(t, w.send(t, "walt", http.MethodPost, "/api/v0/projects/"+w.alpha.String()+"/intake-issues", `{"name":"Alpha triage"}`, http.StatusCreated), &submitted)
	w.a6 = submitted.ID
	w.a7 = w.newWorkItemBy(t, "bob", w.alpha, `{"name":"Alpha bob","assignee_ids":`+bob+`}`)
	w.send(t, "walt", http.MethodPost, "/api/v0/issues/"+w.a1.String()+"/subscribe", "", http.StatusNoContent)
	pm := queryIDs(t, w.pool, "SELECT id FROM project_members WHERE project_id=$1 AND member_id=$2", w.web, w.ids["walt"])[0]
	w.send(t, "alice", http.MethodPatch, "/api/v0/project-members/"+pm.String(), `{"role":5}`, http.StatusOK)
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.shelf.String()+"/archive", "", http.StatusOK)
	w.send(t, "walt", http.MethodPost, "/api/v0/projects/"+w.former.String()+"/leave", "", http.StatusNoContent)
	return w
}

// listed checks actor's flat page of the workspace's list with query:
// its rows are want, in this order, its total is total, and it has a
// next cursor exactly when more is told. It answers the page.
func (w workspaceListWorld) listed(t *testing.T, actor, query string, total int64, more bool, want ...uuid.UUID) issueListHTTP {
	t.Helper()
	p := checkListEnvelope(t, w.send(t, actor, http.MethodGet, workspaceIssuesPath+"?"+query, "", http.StatusOK), false, false)
	got := make([]uuid.UUID, len(p.Data))
	for n, row := range p.Data {
		got[n] = row.ID
	}
	if p.Total != total || !slices.Equal(got, want) || (p.NextCursor != nil) != more {
		t.Fatalf("%s lists ?%s = %+v; want %v of %d, more %t", actor, query, p, want, total, more)
	}
	return p
}

// all is actor's whole list, of no filter, as listed checks it.
func (w workspaceListWorld) all(t *testing.T, actor string, want ...uuid.UUID) {
	t.Helper()
	w.listed(t, actor, "", int64(len(want)), false, want...)
}

// cursorAt is the list's cursor of -created_at at the work item id, whose
// creation alice, the admin of each project, reads through the API: the
// one a page ending on it would give, though walt may not see it.
func (w workspaceListWorld) cursorAt(t *testing.T, id uuid.UUID) string {
	t.Helper()
	var item struct {
		CreatedAt string `json:"created_at"`
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodGet, "/api/v0/issues/"+id.String(), "", http.StatusOK), &item)
	key, err := json.Marshal(item.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	cursor, err := domain.EncodeListCursor(domain.ListCursor{Order: "-created_at", Key: key, ID: id})
	if err != nil {
		t.Fatal(err)
	}
	return cursor
}
