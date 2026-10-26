package bootstrap

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
	"github.com/open-nerve/NerveProject/server/migrations"
)

func TestMemberIssueListsAsRuntimeRole(t *testing.T) {
	roles := newSplitRoles(t)
	w := issueWorld{contract: apitest.Load(t), base: startApp(t, testConfig(t, roles.serverURL, false), migrations.FS()), pool: roles.server, tokens: map[string]string{}, ids: map[string]uuid.UUID{}}
	for _, name := range []string{"alice", "bob", "walt"} {
		email := name + "-member-runtime@example.com"
		w.tokens[name] = registerAccount(t, w.contract, w.base, email).AccessToken
		w.ids[name] = accountID(t, w.contract, w.base, w.tokens[name])
	}
	var made struct{ ID uuid.UUID }
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/workspaces", `{"name":"Acme","slug":"acme"}`, 201), &made)
	w.acme = made.ID
	for _, name := range []string{"bob", "walt"} {
		link := invite(t, w.contract, w.base, w.tokens["alice"], "acme", name+"-member-runtime@example.com")
		answerInvitation(t, w.contract, w.base, w.tokens[name], "accept", link, 200)
	}
	w.web = createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Web", "WEB")
	w.makeMember(t, "bob", w.web, shared.RoleMember)
	ids := memberRelations(t, w)
	for relation, want := range map[string][]uuid.UUID{"assigned": {ids[0]}, "created": {ids[1]}, "subscribed": ids[:3]} {
		memberListIDs(t, w, "walt", "bob", relation, "", want...)
	}
	path := memberIssuesPath(w, "bob") + "?relation=subscribed&group_by=project_id&order_by=sort_order&limit=1"
	p := checkListEnvelope(t, w.send(t, "walt", http.MethodGet, path, "", 200), true, false)
	if p.Total != 3 || len(p.Groups) != 1 || p.Groups[0].Key != w.web.String() || p.Groups[0].Total != 3 || len(p.Groups[0].Data) != 1 || p.Groups[0].NextCursor == nil {
		t.Fatal(p)
	}
	cursor := p.Groups[0].NextCursor
	seen := []uuid.UUID{p.Groups[0].Data[0].ID}
	for cursor != nil {
		p = checkListEnvelope(t, w.send(t, "walt", http.MethodGet, path+"&group_key="+w.web.String()+"&cursor="+url.QueryEscape(*cursor), "", 200), false, false)
		if p.Total != 3 || len(p.Data) != 1 {
			t.Fatal(p)
		}
		seen = append(seen, p.Data[0].ID)
		cursor = p.NextCursor
	}
	want := slices.Clone(ids[:3])
	slices.SortFunc(want, uuid.UUID.Compare)
	slices.SortFunc(seen, uuid.UUID.Compare)
	if !slices.Equal(seen, want) {
		t.Fatal(seen, want)
	}
	// Build as a member before demotion. A's own assignment to B survives OwnOnly.
	own := w.newWorkItemBy(t, "walt", w.web, fmt.Sprintf(`{"name":"Own","assignee_ids":%s}`, idArray(w.ids["bob"])))
	pm := queryIDs(t, w.pool, "SELECT id FROM project_members WHERE project_id=$1 AND member_id=$2", w.web, w.ids["walt"])[0]
	w.send(t, "alice", http.MethodPatch, "/api/v0/project-members/"+pm.String(), `{"role":5}`, 200)
	memberListIDs(t, w, "walt", "bob", "assigned", "", own)
	memberListIDs(t, w, "walt", "bob", "created", "")
	memberListIDs(t, w, "walt", "bob", "subscribed", "", own)
}
