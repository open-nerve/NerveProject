package postgresadapter_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// Where the work items the navigation module names are (M7 P3d spec 2.7, R3): the undeleted ones among the ids, an
// archived one too, of any project, another workspace's too, each with its workspace and its project, once though asked
// twice; a deleted one, or none, is not among them, and no ids read none. No decision: the grants and the guest rule
// are not asked. It reads what is committed, without a lock.
func TestNavigationIssuePlaces(t *testing.T) {
	s, pool := newStore(t)
	by := newAccount(t, pool, "places@corp.com")
	acme := newWorkspace(t, pool, "places")
	web, ops, far := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS"), newProject(t, pool, newWorkspace(t, pool, "far"), "FAR")
	plain, shelved, gone, theirs, elsewhere := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for n, i := range []uuid.UUID{plain, shelved, gone} {
		newIssue(t, pool, web, i, n+1, by)
	}
	newIssue(t, pool, ops, theirs, 1, by)
	newIssue(t, pool, far, elsewhere, 1, by)
	exec(t, pool, "UPDATE issues SET archived_at = now() WHERE id = $1", shelved)
	exec(t, pool, "UPDATE issues SET deleted_at = now() WHERE id = $1", gone)
	place := func(p project) app.NavigationIssuePlace {
		return app.NavigationIssuePlace{WorkspaceID: p.workspace, ProjectID: p.id}
	}
	want := map[uuid.UUID]app.NavigationIssuePlace{plain: place(web), shelved: place(web), theirs: place(ops), elsewhere: place(far)}
	got, err := s.NavigationIssuePlaces(context.Background(), []uuid.UUID{elsewhere, gone, uuid.NewV7(), theirs, shelved, plain, plain})
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("NavigationIssuePlaces() = %+v, %v; want %+v", got, err, want)
	}
	for _, ids := range [][]uuid.UUID{nil, {gone, uuid.NewV7()}} {
		if got, err := s.NavigationIssuePlaces(context.Background(), ids); err != nil || len(got) != 0 {
			t.Errorf("NavigationIssuePlaces(%v) = %+v, %v; want none", ids, got, err)
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := s.NavigationIssuePlaces(cancelled, []uuid.UUID{plain}); !errors.Is(err, context.Canceled) || got != nil {
		t.Errorf("on a cancelled context: %+v, %v; want context.Canceled", got, err)
	}
}

// The work items as a recent visit answers them now (M7 P3d spec 2.4, 2.5, R3): the undeleted ones among the ids, an
// archived one too, of any project, another workspace's too, in id order, each with its project, its number, its name,
// its state, its priority, the accounts of its undeleted assignees' rows, each once, in id order, none for none, its
// creator, nil when not known, and its undeleted intake row's status, nil for none and for a deleted row alone; a
// deleted one, or none, is not among them. The grants, the guest rule and InTriage are NavigationIssues', not the
// statement's.
func TestNavigationIssueRows(t *testing.T) {
	s, pool := newStore(t)
	by := newAccount(t, pool, "rows@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "rows"), "WEB")
	far := newProject(t, pool, newWorkspace(t, pool, "far"), "FAR")
	item, quiet, gone, elsewhere := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for n, i := range []uuid.UUID{item, quiet, gone} {
		newIssue(t, pool, web, i, n+7, by)
	}
	newIssue(t, pool, far, elsewhere, 1, by)
	exec(t, pool, "UPDATE issues SET name = 'Fix the login', priority = 'high', archived_at = now() WHERE id = $1", item)
	exec(t, pool, "UPDATE issues SET created_by_id = NULL WHERE id = $1", quiet)
	exec(t, pool, "UPDATE issues SET deleted_at = now() WHERE id = $1", gone)
	first, second, again, removed := newAccount(t, pool, "first@corp.com"), newAccount(t, pool, "second@corp.com"),
		newAccount(t, pool, "again@corp.com"), newAccount(t, pool, "removed@corp.com")
	assign := func(p project, issue, account uuid.UUID, deleted bool) {
		exec(t, pool, `INSERT INTO issue_assignees (id, workspace_id, project_id, issue_id, assignee_id, deleted_at)
			VALUES ($1, $2, $3, $4, $5, CASE WHEN $6 THEN now() END)`, uuid.NewV7(), p.workspace, p.id, issue, account, deleted)
	}
	assign(web, item, again, true)
	assign(web, item, second, false)
	assign(web, item, removed, true)
	assign(web, item, first, false)
	assign(web, item, again, false)
	assign(web, gone, first, false)
	assign(far, elsewhere, removed, false)
	row := func(id uuid.UUID, p project, sequence int, name string, priority domain.Priority, assignees []uuid.UUID, creator *uuid.UUID,
		intake *domain.IntakeFacts) app.NavigationIssueRow {
		return app.NavigationIssueRow{ID: id, ProjectID: p.id, SequenceID: sequence, Name: name, StateID: p.state, Priority: priority,
			AssigneeIDs: assignees, CreatedByID: creator, Intake: intake}
	}
	parent := uuid.NewV7()
	newIssue(t, pool, web, parent, 99, by)
	cases := newIntakeCases(t, pool, web, parent, by)
	ids := []uuid.UUID{elsewhere, gone, uuid.NewV7(), quiet, item, item}
	want := []app.NavigationIssueRow{
		row(item, web, 7, "Fix the login", domain.PriorityHigh, []uuid.UUID{first, second, again}, &by, nil),
		row(quiet, web, 8, "Work", domain.PriorityNone, []uuid.UUID{}, nil, nil),
	}
	for n, c := range cases {
		ids = append(ids, c.id)
		var intake *domain.IntakeFacts
		if c.want != nil {
			intake = &domain.IntakeFacts{Status: c.want.Status}
		}
		want = append(want, row(c.id, web, 100+n, "Work", domain.PriorityNone, []uuid.UUID{}, &by, intake))
	}
	want = append(want, row(elsewhere, far, 1, "Work", domain.PriorityNone, []uuid.UUID{removed}, &by, nil))
	slices.SortFunc(want, func(a, b app.NavigationIssueRow) int { return a.ID.Compare(b.ID) })
	got, err := s.NavigationIssueRows(context.Background(), ids)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("NavigationIssueRows() =\n %+v, %v\nwant\n %+v", got, err, want)
	}
	for _, ids := range [][]uuid.UUID{nil, {gone, uuid.NewV7()}} {
		if got, err := s.NavigationIssueRows(context.Background(), ids); err != nil || len(got) != 0 {
			t.Errorf("NavigationIssueRows(%v) = %+v, %v; want none", ids, got, err)
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := s.NavigationIssueRows(cancelled, []uuid.UUID{item}); !errors.Is(err, context.Canceled) || got != nil {
		t.Errorf("on a cancelled context: %+v, %v; want context.Canceled", got, err)
	}
}
