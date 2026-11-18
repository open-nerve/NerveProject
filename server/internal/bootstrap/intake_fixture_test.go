package bootstrap

import (
	"net/http"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// seedIntake writes, directly, an intake row of the undeleted work item
// issue, by by, in the item's workspace and project: accepted, as the row of
// a work item in an ordinary state is once its triage took it out of Triage
// (M7 P2a spec 2.3). The deletion and purge guards need a row of
// intake_issues under the work item, its project and its workspace (spec
// 2.10), as P1b's need a notification (seedNotification): the guards write
// the issue module's intake rows as fixtures, so that they do not depend on
// the submission and the triage.
func seedIntake(t testing.TB, pool *pgxpool.Pool, issue, by uuid.UUID) {
	t.Helper()
	if tag, err := pool.Exec(pgtest.Soon(t), `INSERT INTO intake_issues (id, workspace_id, project_id, issue_id, status, created_by_id, updated_by_id)
		SELECT $1, workspace_id, project_id, id, 'accepted', $3, $3 FROM issues WHERE id = $2 AND deleted_at IS NULL`, uuid.NewV7(), issue, by); err != nil ||
		tag.RowsAffected() != 1 {
		t.Fatalf("seeding the intake row of %s: %d rows, %v; want 1", issue, tag.RowsAffected(), err)
	}
}

// intakeOn has alice, Web's admin, turn its intake on through the API
// (updateProject; M7 P2a spec 2.4: it is off as a project starts).
func (w issueWorld) intakeOn(t *testing.T) {
	t.Helper()
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"intake_view":true}`, http.StatusOK)
}

// submitted is the id of a work item alice submits to Web's intake through
// the API, once she has turned it on (intakeOn): in Web's triage state, its
// intake row pending.
func (w issueWorld) submitted(t *testing.T) uuid.UUID {
	t.Helper()
	w.intakeOn(t)
	var created struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/intake-issues", `{"name":"Login breaks"}`,
		http.StatusCreated), &created)
	return created.ID
}
