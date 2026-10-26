package bootstrap

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"uuid"

	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

func linkMatrixRows() []matrixRow {
	list := toWorkItem(http.MethodGet, "/links", "")
	create := toWorkItem(http.MethodPost, "/links", `{"url":"http://169.254.169.254/matrix-create"}`)
	return []matrixRow{
		{op: "listIssueLinks", columns: issueColumns, request: list, cells: onAWorkItem(cellOK, cellOK, cellOK, cellOK), check: func(t *testing.T, c caller, s seeded, answer string) {
			got := listIn[answeredLink](t, answer)
			if len(got) != 1 || got[0].URL != "http://169.254.169.254/matrix-seed" || got[0].IssueID != s.issue(projectOf(c), issueOf(c)) {
				t.Fatal(answer)
			}
		}},
		{op: "listIssueLinks", variant: "archived", columns: archivedColumns, request: list, cells: issueRows.ofArchived(cellOK, cellForbidden)},
		{op: "createIssueLink", write: true, columns: issueColumns, request: create, cells: onAWorkItem(cellCreated, cellCreated, cellCreated, cellForbidden)},
		{op: "createIssueLink", variant: "archived", write: true, columns: archivedColumns, request: create, cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		{op: "createIssueLink", variant: "both archived", write: true, columns: archivedColumns, prepare: inDone(true), request: create, cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		{op: "createIssueLink", variant: "archived issue", write: true, columns: issueColumns, prepare: inDone(true), request: create, cells: onAWorkItem(cellIssueArchived, cellIssueArchived, cellIssueArchived, cellForbidden)},
	}
}

// Matrix fixtures seed through the same store as the existing issue rows;
// deletion preconditions must carry and verify the new dependent table too.
func (s issueSeed) links(sd seeded) {
	for _, i := range seededItems() {
		slug, _, _ := strings.Cut(i.key, "/")
		by := s.ids[i.by]
		l := domain.Link{ID: uuid.NewV7(), WorkspaceID: s.workspaces[slug], ProjectID: s.projects[i.key], IssueID: sd.issue(i.key, i.name), URL: "http://169.254.169.254/matrix-seed"}
		if err := s.store.InsertLink(context.Background(), issueapp.LinkWrite{Link: l, By: by, At: s.now}); err != nil {
			s.t.Fatal(err)
		}
	}
}
func (s issueSeed) seededLinkRows(sd seeded) {
	for _, i := range seededItems() {
		rows, err := s.store.Links(context.Background(), sd.issue(i.key, i.name))
		gone := strings.HasPrefix(i.key, "gone/")
		if err != nil || !gone && (len(rows) != 1 || rows[0].URL != "http://169.254.169.254/matrix-seed") || gone && len(rows) != 0 {
			s.t.Fatalf("seeded links %s/%s: %+v %v", i.key, i.name, rows, err)
		}
	}
}
