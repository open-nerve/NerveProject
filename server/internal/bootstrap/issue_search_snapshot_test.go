package bootstrap

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"uuid"

	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Reuse the existing second-connection/channel helper; only its result shape
// is adapted. Every actual port observes PID, RR and RO on its caller's ctx.
func interleaveSearch(t *testing.T, w issueWorld, p *listSnapshotProbe, raw map[string][]string, writes []listSnapshotWrite) []domain.Issue {
	t.Helper()
	u := snapshotSearchUseCase(w, p)
	page := interleaveList(t, w, p, "walt", func(ctx context.Context) (issueapp.IssueListPage, error) {
		rows, err := u.Execute(ctx, "acme", raw)
		return issueapp.IssueListPage{Data: rows}, err
	}, writes)
	return page.Data
}
func TestSearchReadSnapshot(t *testing.T) {
	t.Run("scope source rows and seen", func(t *testing.T) {
		w := newIssueWorld(t)
		w.makeMember(t, "walt", w.web, shared.RoleMember)
		other := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Other", "OTHER")
		w.makeMember(t, "walt", other, shared.RoleMember)
		source := w.newWorkItemBy(t, "walt", w.web, `{"name":"Source"}`)
		parent := w.newWorkItemBy(t, "bob", w.web, `{"name":"Parent"}`)
		target := w.newWorkItemBy(t, "walt", w.web, fmt.Sprintf(`{"name":"Pick","parent_id":%q}`, parent))
		w.newWorkItemBy(t, "walt", w.web, fmt.Sprintf(`{"name":"Child","parent_id":%q}`, target))
		w.newWorkItemBy(t, "bob", w.web, fmt.Sprintf(`{"name":"Child","parent_id":%q}`, target))
		outside := w.newWorkItemIn(t, other, `{"name":"Pick"}`)
		listSeedSQL(t, w, `INSERT INTO issue_links(id,workspace_id,project_id,issue_id,url,created_by_id) VALUES($1,$2,$3,$4,'https://example.com/old',$5)`, uuid.NewV7(), w.acme, w.web, target, w.ids["bob"])
		listSeedSQL(t, w, "UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2", w.web, w.ids["walt"])
		p := newListProbe(w, "visible")
		raw := map[string][]string{"q": {"Pick"}, "for": {"relation"}, "issue_id": {source.String()}}
		old := interleaveSearch(t, w, p, raw, []listSnapshotWrite{
			{"INSERT INTO issue_links(id,workspace_id,project_id,issue_id,url,created_by_id) VALUES($1,$2,$3,$4,'https://example.com/new',$5)", []any{uuid.NewV7(), w.acme, w.web, target, w.ids["bob"]}},
			{"UPDATE project_members SET role=15 WHERE project_id=$1 AND member_id=$2", []any{w.web, w.ids["walt"]}},
			{"UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2", []any{other, w.ids["walt"]}},
			{"UPDATE issues SET parent_id=$2,name='Pick changed' WHERE id=$1", []any{target, source}},
			{"INSERT INTO issues(id,workspace_id,project_id,state_id,sequence_id,name,parent_id,created_by_id) VALUES($1,$2,$3,$4,9000,'Child',$5,$6)", []any{uuid.NewV7(), w.acme, w.web, w.todo, target, w.ids["bob"]}},
		})
		checkSearchIDs(t, old, outside, target)
		if old[1].ParentID != nil || old[1].SubIssuesCount != 1 || old[1].Name != "Pick" || old[1].LinkCount != 1 {
			t.Fatal("mixed old scope/seen", old)
		}
		wantSteps := []string{"workspace", "auth", "visible", "source", "identifiers", "rows", "counts", "parents"}
		if !slices.Equal(p.steps, wantSteps) {
			t.Fatal("missing or repeated snapshot stage", p.steps, wantSteps)
		}
		next := searchExecute(t, w, "walt", raw)
		checkSearchIDs(t, next, target)
		if next[0].ParentID == nil || *next[0].ParentID != source || next[0].SubIssuesCount != 3 || next[0].Name != "Pick changed" || next[0].LinkCount != 2 {
			t.Fatal("next snapshot missed committed scope/seen", next)
		}
	})
	t.Run("live relation committed after directory", func(t *testing.T) {
		w := newIssueWorld(t)
		w.makeMember(t, "walt", w.web, shared.RoleMember)
		source := w.newWorkItemBy(t, "walt", w.web, `{"name":"Source"}`)
		target := w.newWorkItemIn(t, w.web, `{"name":"Pick"}`)
		raw := map[string][]string{"q": {"Pick"}, "for": {"relation"}, "issue_id": {source.String()}}
		p := newListProbe(w, "visible")
		old := interleaveSearch(t, w, p, raw, []listSnapshotWrite{{"INSERT INTO issue_relations(id,workspace_id,project_id,issue_id,related_issue_id,relation_type) VALUES($1,$2,$3,$4,$5,'blocked_by')", []any{uuid.NewV7(), w.acme, w.web, target, source}}})
		checkSearchIDs(t, old, target)
		checkSearchIDs(t, searchExecute(t, w, "walt", raw))
	})
}
