package bootstrap

import (
	"net/http"
	"testing"
)

func collaborationMatrixRows() []matrixRow {
	list := toWorkItem(http.MethodGet, "/comments", "")
	create := toWorkItem(http.MethodPost, "/comments", `{"comment_html":"<p>New comment</p>"}`)
	return []matrixRow{
		{op: "listIssueComments", columns: issueColumns, request: list, cells: onAWorkItem(cellOK, cellOK, cellOK, cellOK), check: func(t *testing.T, _ caller, _ seeded, answer string) {
			if got := listIn[answeredComment](t, answer); len(got) != 1 || got[0].HTML != "<p>Seeded comment</p>" {
				t.Fatal(answer)
			}
		}},
		{op: "listIssueComments", variant: "archived", columns: archivedColumns, request: list, cells: issueRows.ofArchived(cellOK, cellForbidden)},
		{op: "createIssueComment", write: true, columns: issueColumns, request: create, cells: onAWorkItem(cellCreated, cellCreated, cellCreated, cellCreated)},
		{op: "createIssueComment", variant: "archived", write: true, columns: archivedColumns, request: create, cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		{op: "createIssueComment", variant: "both archived", write: true, columns: archivedColumns, prepare: inDone(true), request: create, cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		{op: "createIssueComment", variant: "archived work item", write: true, columns: issueColumns, prepare: inDone(true), request: create, cells: onAWorkItem(cellIssueArchived, cellIssueArchived, cellIssueArchived, cellIssueArchived)},
	}
}
