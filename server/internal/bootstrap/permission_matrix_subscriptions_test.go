package bootstrap

import (
	"net/http"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The rows of the permission matrix of subscribing to a work item and
// unsubscribing from it (M7 P1a spec 2.3, R3, R4), naming it by its id: its
// column's work item (issueOf). Each is issue.read's (getIssue's) rule, the
// admins, members and guests of its project allowed: who does not see it
// (the guest rule's other case, a project he does not see, or no member of
// the workspace) is issue.not_found; who sees the project and is none of its
// members, the workspace's admin or member on a public one, forbidden. A
// work item of an archived project, or one archived itself, is subscribed
// to and unsubscribed from as any other (R3). Each successful cell answers
// 204 without a body; an unsubscription's cell removes the caller's own
// row, which its prepare puts there (ownSubscription).
func subscriptionMatrixRows() []matrixRow {
	var rows []matrixRow
	for _, unsubscribe := range []bool{false, true} {
		op, after := "subscribeIssue", "/subscribe"
		var own func(*testing.T, caller, seeded, string)
		if unsubscribe {
			op, after, own = "unsubscribeIssue", "/unsubscribe", ownSubscription
		}
		request := toWorkItem(http.MethodPost, after, "")
		archivedIssue := func(t *testing.T, c caller, s seeded, url string) {
			if own != nil {
				own(t, c, s, url)
			}
			inDone(true)(t, c, s, url)
		}
		rows = append(rows,
			matrixRow{op: op, write: true, columns: issueColumns, prepare: own, request: request,
				cells: onAWorkItem(cellNoContent, cellNoContent, cellNoContent, cellNoContent)},
			matrixRow{op: op, variant: "archived", write: true, columns: archivedColumns, prepare: own, request: request,
				cells: issueRows.ofArchived(cellNoContent, cellForbidden)},
			matrixRow{op: op, variant: "archived work item", write: true, columns: issueColumns, prepare: archivedIssue, request: request,
				cells: onAWorkItem(cellNoContent, cellNoContent, cellNoContent, cellNoContent)})
	}
	return rows
}

// ownSubscription puts the caller's undeleted row among the subscribers of
// his column's work item in this write cell, before its app starts, unless
// he has one already.
func ownSubscription(t *testing.T, c caller, s seeded, url string) {
	t.Helper()
	pool := openPool(t, url)
	defer pool.Close()
	tag, err := pool.Exec(pgtest.Soon(t), `INSERT INTO issue_subscribers (id, workspace_id, project_id, issue_id, subscriber_id, created_by_id, updated_by_id)
		SELECT $1, workspace_id, project_id, id, $3, $3, $3 FROM issues WHERE id = $2
		ON CONFLICT DO NOTHING`, uuid.NewV7(), s.issue(projectOf(c), issueOf(c)), s.account(accountOf(c)))
	if err != nil {
		t.Fatal(err)
	}
	var subscribed bool
	if err := pool.QueryRow(pgtest.Soon(t), `SELECT EXISTS (SELECT 1 FROM issue_subscribers WHERE issue_id = $1 AND subscriber_id = $2
		AND deleted_at IS NULL)`, s.issue(projectOf(c), issueOf(c)), s.account(accountOf(c))).Scan(&subscribed); err != nil || !subscribed {
		t.Fatalf("%s's subscription: %v, %d rows inserted, subscribed %v", c, err, tag.RowsAffected(), subscribed)
	}
}
