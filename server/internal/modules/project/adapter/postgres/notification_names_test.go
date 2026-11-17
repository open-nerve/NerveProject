package postgresadapter_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// The names a notification's snapshot is completed with (M7 P1b spec 2.5,
// 2.6), each read alone, by the ids it is asked: the name and the group of
// a project's undeleted state, its triage state too, and none for another
// project's state, a deleted state or none; the identifier of a workspace's
// undeleted project, an archived one's too, and none for a deleted
// project, another workspace's or none. On stateWorld, whose Web, Ops and
// Site each have the six default states, with acme's archived Old and
// deleted Gone besides: a read that answered the workspace's first project
// or the project's first state, by id, would answer Web's or Backlog.
func TestNotificationNames(t *testing.T) {
	w := newStateWorld(t)
	ctx := context.Background()
	deleted := uuid.NewV7()
	exec(t, w.pool, `INSERT INTO states (id, workspace_id, project_id, name, color, "group", deleted_at) VALUES ($1, $2, $3, 'Gone', '#000', 'started', $4)`,
		deleted, w.acme, w.ops, now)
	for _, tt := range []struct {
		name           string
		project, state uuid.UUID
		want           app.NotificationState
		found          bool
	}{
		{"Ops' In Progress", w.ops, w.states[w.ops]["In Progress"], app.NotificationState{Name: "In Progress", Group: "started"}, true},
		{"Ops' Triage", w.ops, w.states[w.ops]["Triage"], app.NotificationState{Name: "Triage", Group: "triage"}, true},
		{"Web's Done", w.web, w.states[w.web]["Done"], app.NotificationState{Name: "Done", Group: "completed"}, true},
		{"Web's In Progress, of Ops", w.ops, w.states[w.web]["In Progress"], app.NotificationState{}, false},
		{"a deleted state", w.ops, deleted, app.NotificationState{}, false},
		{"no state", w.ops, uuid.NewV7(), app.NotificationState{}, false},
	} {
		if got, found, err := w.s.NotificationState(ctx, tt.project, tt.state); err != nil || found != tt.found || got != tt.want {
			t.Errorf("%s: NotificationState() = %+v, %v, %v; want %+v, %v", tt.name, got, found, err, tt.want, tt.found)
		}
	}
	old, gone := newProject(t, w.s, w.acme, "Old", "OLD", w.maker), newProject(t, w.s, w.acme, "Gone", "GONE", w.maker)
	exec(t, w.pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", old, now)
	exec(t, w.pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", gone, now)
	for _, tt := range []struct {
		name               string
		workspace, project uuid.UUID
		want               string
		found              bool
	}{
		{"Ops", w.acme, w.ops, "OPS", true},
		{"Web", w.acme, w.web, "WEB", true},
		{"Old, archived", w.acme, old, "OLD", true},
		{"Gone, deleted", w.acme, gone, "", false},
		{"Ops, of beta", w.beta, w.ops, "", false},
		{"Site, of acme", w.acme, w.site, "", false},
		{"no project", w.acme, uuid.NewV7(), "", false},
	} {
		if got, found, err := w.s.NotificationIdentifier(ctx, tt.workspace, tt.project); err != nil || found != tt.found || got != tt.want {
			t.Errorf("%s: NotificationIdentifier() = %q, %v, %v; want %q, %v", tt.name, got, found, err, tt.want, tt.found)
		}
	}
}

// The names are read in the transaction ctx carries: a project and a state
// the transaction wrote are read before it commits, and none once it is
// rolled back. A failed read is its error, not "not found".
func TestNotificationNamesReadInTheTransaction(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	project, state := uuid.NewV7(), uuid.NewV7()
	var name app.NotificationState
	var identifier string
	var stateFound, projectFound bool
	err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		if err := s.CreateProject(ctx, app.ProjectRow{ID: project, WorkspaceID: acme, Name: "Web", Identifier: "WEB", Network: domain.NetworkPublic,
			Timezone: "UTC", CreatedBy: alice, Now: now}); err != nil {
			return err
		}
		todo := domain.DefaultStates()[1]
		if err := s.CreateStates(ctx, []app.StateRow{{ID: state, WorkspaceID: acme, ProjectID: project, State: todo,
			CreatedBy: alice, Now: now}}); err != nil {
			return err
		}
		var errState, errProject error
		name, stateFound, errState = s.NotificationState(ctx, project, state)
		identifier, projectFound, errProject = s.NotificationIdentifier(ctx, acme, project)
		return errors.Join(errState, errProject, errors.New("roll back"))
	})
	if want := (app.NotificationState{Name: "Todo", Group: "unstarted"}); err == nil ||
		!stateFound || name != want || !projectFound || identifier != "WEB" {
		t.Errorf("in the transaction: %+v, %v, %q, %v, %v; want %+v and WEB", name, stateFound, identifier, projectFound, err, want)
	}
	if got, found, err := s.NotificationState(context.Background(), project, state); err != nil || found || got != (app.NotificationState{}) {
		t.Errorf("the state after the rollback: %+v, %v, %v; want none", got, found, err)
	}
	if got, found, err := s.NotificationIdentifier(context.Background(), acme, project); err != nil || found || got != "" {
		t.Errorf("the identifier after the rollback: %q, %v, %v; want none", got, found, err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if got, found, err := s.NotificationState(cancelled, project, state); !errors.Is(err, context.Canceled) || found || got != (app.NotificationState{}) {
		t.Errorf("NotificationState on a cancelled context: %+v, %v, %v; want context.Canceled", got, found, err)
	}
	if got, found, err := s.NotificationIdentifier(cancelled, acme, project); !errors.Is(err, context.Canceled) || found || got != "" {
		t.Errorf("NotificationIdentifier on a cancelled context: %q, %v, %v; want context.Canceled", got, found, err)
	}
}
