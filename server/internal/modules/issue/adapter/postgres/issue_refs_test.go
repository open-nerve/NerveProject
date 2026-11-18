package postgresadapter_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// The work items another names, as IssueRefs reads them (M7 P1c spec 2.6, R4): the undeleted ones among the ids, an
// archived one too, of any project, another workspace's too, each with its project, its number, its name and its
// creator, nil when not known, in id order; a deleted one, or none, is not among them, and no ids read none. The
// guest rule and the grants are IssueRefs', not the statement's. It reads what is committed, without a lock.
func TestIssueRefRows(t *testing.T) {
	s, pool := newStore(t)
	by, other := newAccount(t, pool, "refs@corp.com"), newAccount(t, pool, "other@corp.com")
	acme := newWorkspace(t, pool, "refs")
	web, ops, far := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS"), newProject(t, pool, newWorkspace(t, pool, "far"), "FAR")
	plain, shelved, unknown, gone, theirs, elsewhere := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for n, i := range []uuid.UUID{plain, shelved, unknown, gone} {
		newIssue(t, pool, web, i, n+1, by)
	}
	newIssue(t, pool, ops, theirs, 1, other)
	newIssue(t, pool, far, elsewhere, 1, by)
	exec(t, pool, "UPDATE issues SET name = 'Fix the login' WHERE id = $1", plain)
	exec(t, pool, "UPDATE issues SET archived_at = now() WHERE id = $1", shelved)
	exec(t, pool, "UPDATE issues SET created_by_id = NULL WHERE id = $1", unknown)
	exec(t, pool, "UPDATE issues SET deleted_at = now() WHERE id = $1", gone)
	ref := func(id uuid.UUID, p project, sequence int, name string, creator *uuid.UUID) app.IssueRefRow {
		return app.IssueRefRow{IssueRef: app.IssueRef{ID: id, ProjectID: p.id, SequenceID: sequence, Name: name}, CreatedByID: creator}
	}
	want := []app.IssueRefRow{ref(plain, web, 1, "Fix the login", &by), ref(shelved, web, 2, "Work", &by), ref(unknown, web, 3, "Work", nil),
		ref(theirs, ops, 1, "Work", &other), ref(elsewhere, far, 1, "Work", &by)}
	got, err := s.IssueRefRows(context.Background(), []uuid.UUID{elsewhere, gone, uuid.NewV7(), theirs, unknown, shelved, plain, plain})
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("IssueRefRows() = %+v, %v; want %+v", got, err, want)
	}
	for _, ids := range [][]uuid.UUID{nil, {gone, uuid.NewV7()}} {
		if got, err := s.IssueRefRows(context.Background(), ids); err != nil || len(got) != 0 {
			t.Errorf("IssueRefRows(%v) = %+v, %v; want none", ids, got, err)
		}
	}
}

// The rows are read in the transaction ctx carries: a work item the
// transaction wrote is read before it commits, and none once it is rolled
// back. A failed read is its error, not none.
func TestIssueRefRowsReadInTheTransaction(t *testing.T) {
	s, pool := newStore(t)
	by := newAccount(t, pool, "refs@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "refs"), "WEB")
	item := uuid.NewV7()
	var inside []app.IssueRefRow
	err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		if _, err := postgres.DB(ctx, pool).Exec(ctx, `INSERT INTO issues (id, workspace_id, project_id, state_id, name, sequence_id,
			created_by_id) VALUES ($1, $2, $3, $4, 'Work', 1, $5)`, item, web.workspace, web.id, web.state, by); err != nil {
			return err
		}
		var err error
		inside, err = s.IssueRefRows(ctx, []uuid.UUID{item})
		return errors.Join(err, errors.New("roll back"))
	})
	want := []app.IssueRefRow{{IssueRef: app.IssueRef{ID: item, ProjectID: web.id, SequenceID: 1, Name: "Work"}, CreatedByID: &by}}
	if err == nil || !reflect.DeepEqual(inside, want) {
		t.Errorf("in the transaction: %+v, %v; want %+v", inside, err, want)
	}
	if got, err := s.IssueRefRows(context.Background(), []uuid.UUID{item}); err != nil || len(got) != 0 {
		t.Errorf("after the rollback: %+v, %v; want none", got, err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := s.IssueRefRows(cancelled, []uuid.UUID{item}); !errors.Is(err, context.Canceled) || got != nil {
		t.Errorf("on a cancelled context: %+v, %v; want context.Canceled", got, err)
	}
}
