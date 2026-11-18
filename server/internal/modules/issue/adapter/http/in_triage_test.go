package httpadapter_test

import (
	"net/http"
	"testing"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// Each of the six writes that guard their own work item answers a work
// item in its project's intake as its use case refuses it (M7 P2a spec
// 2.6): 409 issue.in_triage, as the contract declares it for each.
func TestWritesOnAWorkItemInTheIntakeAnswerInTriage(t *testing.T) {
	const want = `{"status":409,"code":"issue.in_triage","title":"Conflict",` +
		`"detail":"The work item is in the project's intake; triage it before changing it."}`
	issuePath := "/api/v0/issues/" + webEight.ID.String()
	for _, tt := range []struct {
		op, method, path, body string
		uc                     httpadapter.UseCases
	}{
		{"updateIssue", http.MethodPatch, issuePath, `{"name":"Mend"}`, httpadapter.UseCases{UpdateIssue: &fakeUpdate{err: domain.ErrInTriage}}},
		{"archiveIssue", http.MethodPost, issuePath + "/archive", "", httpadapter.UseCases{ArchiveIssue: &fakeArchive{err: domain.ErrInTriage}}},
		{"unarchiveIssue", http.MethodPost, issuePath + "/unarchive", "", httpadapter.UseCases{UnarchiveIssue: &fakeArchive{err: domain.ErrInTriage}}},
		{"deleteIssue", http.MethodDelete, issuePath, "", httpadapter.UseCases{DeleteIssue: &fakeDelete{err: domain.ErrInTriage}}},
		{"addSubIssues", http.MethodPost, issuePath + "/sub-issues", `{"sub_issue_ids":["0199a2b4-0000-7000-8000-0000000000e7"]}`,
			httpadapter.UseCases{AddSubIssues: &fakeAdd{err: domain.ErrInTriage}}},
		{"createIssueRelations", http.MethodPost, issuePath + "/relations", `{"relation_type":"relates_to","issue_ids":["0199a2b4-0000-7000-8000-0000000000e7"]}`,
			httpadapter.UseCases{CreateIssueRelations: &fakeRelate{err: domain.ErrInTriage}}},
	} {
		res, body := do(t, newServer(t, tt.uc), request(tt.method, tt.path, "alice", tt.body))
		if res.StatusCode != http.StatusConflict || body != want+"\n" {
			t.Errorf("%s: %s %s = %d %s, want 409 %s", tt.op, tt.method, tt.path, res.StatusCode, body, want)
		}
	}
}
