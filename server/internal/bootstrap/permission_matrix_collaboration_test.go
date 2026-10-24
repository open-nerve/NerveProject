package bootstrap

import (
	"net/http"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

func collaborationMatrixRows() []matrixRow {
	list := toWorkItem(http.MethodGet, "/comments", "")
	create := toWorkItem(http.MethodPost, "/comments", `{"comment_html":"<p>New comment</p>"}`)
	rows := []matrixRow{
		{op: "createIssue", variant: "description", write: true, columns: issueColumns, request: toProject(http.MethodPost, "/issues", `{"name":"Description","description_html":"<p>Rich</p>"}`), cells: onTheIssues(cellCreated, cellCreated, cellForbidden)},
		{op: "updateIssue", variant: "description", write: true, columns: issueColumns, request: toWorkItem(http.MethodPatch, "", `{"description_html":"<p>Rich</p>"}`), cells: onAWorkItem(cellOK, cellOK, cellOK, cellForbidden)},
		{op: "updateIssue", variant: "skip description", write: true, columns: issueColumns, request: toWorkItem(http.MethodPatch, "", `{"description_html":"<p></p>","skip_activity":true}`), cells: onAWorkItem(cellOK, cellOK, cellOK, cellForbidden)},
		{op: "updateIssue", variant: "description archived project", write: true, columns: archivedColumns, request: toWorkItem(http.MethodPatch, "", `{"description_html":"<p>Rich</p>"}`), cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		{op: "updateIssue", variant: "description archived issue", write: true, columns: issueColumns, prepare: inDone(true), request: toWorkItem(http.MethodPatch, "", `{"description_html":"<p>Rich</p>"}`), cells: onAWorkItem(cellIssueArchived, cellIssueArchived, cellIssueArchived, cellForbidden)},
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

	listReactions := toWorkItem(http.MethodGet, "/reactions", "")
	rows = append(rows, matrixRow{op: "listIssueReactions", columns: issueColumns, request: listReactions, cells: onAWorkItem(cellOK, cellOK, cellOK, cellOK), check: func(t *testing.T, _ caller, _ seeded, answer string) {
		got := listIn[answeredReaction](t, answer)
		if len(got) != 1 || got[0].Reaction != "1" {
			t.Fatal(answer)
		}
	}}, matrixRow{op: "listIssueReactions", variant: "archived", columns: archivedColumns, request: listReactions, cells: issueRows.ofArchived(cellOK, cellForbidden)})
	for _, remove := range []bool{false, true} {
		op, method, after, body, success := "addIssueReaction", http.MethodPost, "/reactions", `{"reaction":"128077"}`, cellCreated
		var prepare func(*testing.T, caller, seeded, string)
		if remove {
			op, method, after, body, success = "removeIssueReaction", http.MethodDelete, "/reactions/128077", "", cellNoContent
			prepare = ownReaction
		}
		request := toWorkItem(method, after, body)
		rows = append(rows, matrixRow{op: op, write: true, columns: issueColumns, prepare: prepare, request: request, cells: onAWorkItem(success, success, success, success)}, matrixRow{op: op, variant: "archived", write: true, columns: archivedColumns, prepare: prepare, request: request, cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)})
		archivedPrepare := func(t *testing.T, c caller, s seeded, url string) {
			if prepare != nil {
				prepare(t, c, s, url)
			}
			inDone(true)(t, c, s, url)
		}
		rows = append(rows, matrixRow{op: op, variant: "both archived", write: true, columns: archivedColumns, prepare: archivedPrepare, request: request, cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)}, matrixRow{op: op, variant: "archived issue", write: true, columns: issueColumns, prepare: archivedPrepare, request: request, cells: onAWorkItem(cellIssueArchived, cellIssueArchived, cellIssueArchived, cellIssueArchived)})
	}
	for _, remove := range []bool{false, true} {
		op, method, after, body, success := "addCommentReaction", http.MethodPost, "/reactions", `{"reaction":"128077"}`, cellCreated
		var prepare func(*testing.T, caller, seeded, string)
		if remove {
			op, method, after, body, success = "removeCommentReaction", http.MethodDelete, "/reactions/128077", "", cellNoContent
			prepare = ownCommentReaction
		}
		request := func(c caller, s seeded) (string, string, string) {
			return method, commentReactionsOf(s.comment(projectOf(c), issueOf(c))) + after[len("/reactions"):], body
		}
		rows = append(rows, matrixRow{op: op, write: true, columns: issueColumns, prepare: prepare, request: request, cells: commentCells(onAWorkItem(success, success, success, success))}, matrixRow{op: op, variant: "archived", write: true, columns: archivedColumns, prepare: prepare, request: request, cells: commentCells(issueRows.ofArchived(cellProjectArchived, cellForbidden))})
		archivedPrepare := func(t *testing.T, c caller, s seeded, url string) {
			if prepare != nil {
				prepare(t, c, s, url)
			}
			inDone(true)(t, c, s, url)
		}
		rows = append(rows, matrixRow{op: op, variant: "both archived", write: true, columns: archivedColumns, prepare: archivedPrepare, request: request, cells: commentCells(issueRows.ofArchived(cellProjectArchived, cellForbidden))}, matrixRow{op: op, variant: "archived issue", write: true, columns: issueColumns, prepare: archivedPrepare, request: request, cells: commentCells(onAWorkItem(cellIssueArchived, cellIssueArchived, cellIssueArchived, cellIssueArchived))})
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

// ownReaction puts the caller's live row in this write cell before its app starts.
func ownReaction(t *testing.T, c caller, s seeded, url string) {
	t.Helper()
	pool := openPool(t, url)
	defer pool.Close()
	if _, err := pool.Exec(pgtest.Soon(t), `INSERT INTO issue_reactions(id,workspace_id,project_id,issue_id,actor_id,reaction,created_by_id,updated_by_id) SELECT $1,workspace_id,project_id,id,$3,'128077',$3,$3 FROM issues WHERE id=$2`, uuid.NewV7(), s.issue(projectOf(c), issueOf(c)), s.account(accountOf(c))); err != nil {
		t.Fatal(err)
	}
}

func ownCommentReaction(t *testing.T, c caller, s seeded, url string) {
	t.Helper()
	pool := openPool(t, url)
	defer pool.Close()
	if _, err := pool.Exec(pgtest.Soon(t), `INSERT INTO comment_reactions(id,workspace_id,project_id,comment_id,actor_id,reaction,created_by_id,updated_by_id) SELECT $1,workspace_id,project_id,id,$3,'128077',$3,$3 FROM issue_comments WHERE id=$2`, uuid.NewV7(), s.comment(projectOf(c), issueOf(c)), s.account(accountOf(c))); err != nil {
		t.Fatal(err)
	}
}
