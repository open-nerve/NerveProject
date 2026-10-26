package bootstrap

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

func TestIssueListsAsRuntimeRole(t *testing.T) {
	roles := newSplitRoles(t)
	w := issueWorld{contract: apitest.Load(t), base: startApp(t, testConfig(t, roles.serverURL, false), migrations.FS()), pool: roles.server, tokens: map[string]string{}, ids: map[string]uuid.UUID{}}
	w.tokens["alice"] = registerAccount(t, w.contract, w.base, "list-runtime@example.com").AccessToken
	w.ids["alice"] = accountID(t, w.contract, w.base, w.tokens["alice"])
	var made struct{ ID uuid.UUID }
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/workspaces", `{"name":"Acme","slug":"acme"}`, 201), &made)
	w.acme = made.ID
	w.web = createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Web", "WEB")
	w.done = stateID(t, roles.server, w.web, "Done")
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/labels", `{"name":"Bug"}`, 201), &made)
	w.bug = made.ID
	body := fmt.Sprintf(`{"name":"Runtime","state_id":%q,"label_ids":%s,"assignee_ids":%s}`, w.done, idArray(w.bug), idArray(w.ids["alice"]))
	ids := []uuid.UUID{w.newWorkItemIn(t, w.web, body), w.newWorkItemIn(t, w.web, body)}
	path := "/api/v0/projects/" + w.web.String() + "/issues"
	for _, shape := range []string{"", "&group_by=label_id", "&group_by=label_id&sub_group_by=assignee_id"} {
		answer := w.send(t, "alice", http.MethodGet, path+"?limit=1&order_by=sort_order"+shape, "", 200)
		p := checkListEnvelope(t, answer, shape != "", strings.Contains(shape, "sub_group"))
		if p.Total != 2 {
			t.Fatal(answer)
		}
		row := p
		more := ""
		if shape != "" {
			if len(p.Groups) != 1 {
				t.Fatal(answer)
			}
			row = p.Groups[0]
			more = "&group_key=" + row.Key
			if row.Total != 2 {
				t.Fatal(answer)
			}
		}
		if strings.Contains(shape, "sub_group") {
			if len(row.Subgroups) != 1 || len(p.SubgroupTotals) != 1 || p.SubgroupTotals[0].Total != 2 {
				t.Fatal(answer)
			}
			row = row.Subgroups[0]
			more += "&sub_group_key=" + row.Key
		}
		if row.Total != 2 || len(row.Data) != 1 || row.Data[0].ID != ids[0] || row.NextCursor == nil || !slices.Equal(row.Data[0].LabelIDs, []uuid.UUID{w.bug}) || !slices.Equal(row.Data[0].AssigneeIDs, []uuid.UUID{w.ids["alice"]}) {
			t.Fatal(answer)
		}
		next := checkListEnvelope(t, w.send(t, "alice", http.MethodGet, path+"?limit=1&order_by=sort_order"+shape+more+"&cursor="+url.QueryEscape(*row.NextCursor), "", 200), false, false)
		if next.Total != 2 || len(next.Data) != 1 || next.Data[0].ID != ids[1] || next.NextCursor != nil {
			t.Fatal(next)
		}
	}
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+ids[0].String()+"/archive", "", 200)
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", "", 200)
	for _, archived := range []bool{false, true} {
		p := checkListEnvelope(t, w.send(t, "alice", http.MethodGet, path+fmt.Sprintf("?archived=%t", archived), "", 200), false, false)
		want := ids[1]
		if archived {
			want = ids[0]
		}
		if p.Total != 1 || len(p.Data) != 1 || p.Data[0].ID != want || p.NextCursor != nil {
			t.Fatal(p)
		}
	}
}
