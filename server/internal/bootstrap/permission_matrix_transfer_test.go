package bootstrap

import (
	"slices"
	"sync"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The cells of transferCycleIssues' rows (M6 P4 spec 6): the seeded
// cycles have ended and hold no work item, and listsTheCycles holds each
// project's list to them, so each cell's prepare makes, in its own copy,
// the cycle it transfers to and the work items it moves.

// matrixNext is the id of Next, the cycle every cell transfers to, which
// toTransfer makes in the cell's copy.
var matrixNext = uuid.NewV7()

// transferToNext is the body of a transfer to Next.
var transferToNext = transferBody(matrixNext)

// transferCopies are the copies of the cells, by column: each cell's
// prepare keeps its copy's url here, for its check to read the moved
// memberships from; a row's check is given its answer alone.
var transferCopies sync.Map

// toTransfer is a cell's prepare: in its copy at url, through SQL, Next, a
// draft of the column's project with the id matrixNext; every undeleted
// work item of the project in the column's own cycle (cycleOf), its
// matrixIssue in Todo as seeded, the others moved into Done. The copy is
// kept for the check (transferCopies).
func toTransfer(t *testing.T, c caller, s seeded, url string) {
	t.Helper()
	pool := openPool(t, url)
	defer pool.Close()
	ctx, key := pgtest.Soon(t), projectOf(c)
	for _, step := range []struct {
		sql  string
		args []any
	}{
		{"INSERT INTO cycles (id, workspace_id, project_id, name) SELECT $1, workspace_id, id, 'Next' FROM projects WHERE id = $2",
			[]any{matrixNext, s.project(key)}},
		{`INSERT INTO cycle_issues (id, workspace_id, project_id, cycle_id, issue_id)
			SELECT gen_random_uuid(), workspace_id, project_id, $2, id FROM issues WHERE project_id = $1 AND deleted_at IS NULL`,
			[]any{s.project(key), s.cycle(key, cycleOf(c))}},
		{"UPDATE issues SET state_id = $2, completed_at = updated_at WHERE project_id = $1 AND deleted_at IS NULL AND id <> $3",
			[]any{s.project(key), s.state(key, "Done"), s.issue(key, matrixIssue)}},
	} {
		if _, err := pool.Exec(ctx, step.sql, step.args...); err != nil {
			t.Fatalf("preparing %s's transfer: %v", c, err)
		}
	}
	transferCopies.Store(c, url)
}

// transfersTheWorkItems: one work item moved, the column's project's
// matrixIssue, now Next's alone; the project's others, in Done, left in
// the column's own cycle (toTransfer).
func transfersTheWorkItems(t *testing.T, c caller, s seeded, answer string) {
	var got struct {
		Transferred int64 `json:"transferred_issues"`
	}
	decodeAnswer(t, answer, &got)
	url, ok := transferCopies.Load(c)
	if !ok {
		t.Fatalf("no copy of %s's cell was prepared", c)
	}
	pool := openPool(t, url.(string))
	key := projectOf(c)
	const in = "SELECT issue_id FROM cycle_issues WHERE cycle_id = $1 AND deleted_at IS NULL ORDER BY issue_id"
	next, left := queryIDs(t, pool, in, matrixNext), queryIDs(t, pool, in, s.cycle(key, cycleOf(c)))
	others := queryIDs(t, pool, "SELECT id FROM issues WHERE project_id = $1 AND deleted_at IS NULL AND id <> $2 ORDER BY id", s.project(key),
		s.issue(key, matrixIssue))
	if got.Transferred != 1 || !slices.Equal(next, []uuid.UUID{s.issue(key, matrixIssue)}) || len(others) == 0 || !slices.Equal(left, others) {
		t.Errorf("%s answers %s; Next has %v, %s %v; want %s's %s alone moved, and %v left", c, answer, next, cycleOf(c), left, key,
			matrixIssue, others)
	}
}
