package bootstrap

import (
	"net/http"
	"testing"
)

func collaborationMatrixRows() []matrixRow {
	list := toWorkItem(http.MethodGet, "/comments", "")
	create := toWorkItem(http.MethodPost, "/comments", `{"comment_html":"<p>New comment</p>"}`)
	rows := []matrixRow{
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
	notAuthor := cell{http.StatusForbidden, "issue.not_author"}
	editCells := commentCells(onAWorkItem(notAuthor, notAuthor, cellOK, notAuthor))
	editCells[callerGuestOwn], editCells[callerAdminGuestOwn] = cellOK, cellOK
	deleteCells := commentCells(onAWorkItem(cellNoContent, notAuthor, cellNoContent, notAuthor))
	deleteCells[callerGuestOwn] = cellNoContent
	for _, edit := range []bool{true, false} {
		op, method, body, cells := "deleteIssueComment", http.MethodDelete, "", deleteCells
		if edit {
			op, method, body, cells = "updateIssueComment", http.MethodPatch, `{"comment_html":"<p>Updated</p>"}`, editCells
		}
		rows = append(rows, matrixRow{op: op, write: true, columns: issueColumns, request: toComment(method, body), cells: cells})
		archived := commentCells(issueRows.ofArchived(cellProjectArchived, cellForbidden))
		if edit {
			archived = commentCells(issueRows.ofArchived(notAuthor, cellForbidden))
		}
		rows = append(rows, matrixRow{op: op, variant: "archived", write: true, columns: archivedColumns, request: toComment(method, body), cells: archived},
			matrixRow{op: op, variant: "both archived", write: true, columns: archivedColumns, prepare: inDone(true), request: toComment(method, body), cells: archived})
		onArchived := make(map[caller]cell, len(cells))
		for c, v := range cells {
			if v == cellOK || v == cellNoContent {
				v = cellIssueArchived
			}
			onArchived[c] = v
		}
		rows = append(rows, matrixRow{op: op, variant: "archived issue", write: true, columns: issueColumns, prepare: inDone(true), request: toComment(method, body), cells: onArchived})
	}
	return rows
}
func toComment(method, body string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return method, commentPath(s.comment(projectOf(c), issueOf(c))), body
	}
}
func commentCells(cells map[caller]cell) map[caller]cell {
	for c, v := range cells {
		if v == cellIssueNotFound {
			cells[c] = cell{http.StatusNotFound, "issue.comment_not_found"}
		}
	}
	return cells
}
