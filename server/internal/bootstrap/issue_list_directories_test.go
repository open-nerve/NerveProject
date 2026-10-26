package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	issuepostgres "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Keep ordinary answers separate from historical and write catalogs: a
// converter routed to an older provider method must not accidentally pass.
type listProjectAnswers struct {
	visible                          []project.VisibleIssueProject
	listStates                       []project.IssueState
	listLabels                       []project.IssueLabel
	visibleCtx, statesCtx, labelsCtx context.Context
	workspace, actor                 uuid.UUID
	stateProjects, labelProjects     []uuid.UUID
}

func (f *fakeIssueProjects) VisibleIssueProjects(ctx context.Context, wid, actor uuid.UUID) ([]project.VisibleIssueProject, error) {
	f.visibleCtx, f.workspace, f.actor = ctx, wid, actor
	return f.visible, f.err
}
func (f *fakeIssueProjects) IssueListStates(ctx context.Context, pids []uuid.UUID) ([]project.IssueState, error) {
	f.statesCtx, f.stateProjects = ctx, slices.Clone(pids)
	return f.listStates, f.err
}
func (f *fakeIssueProjects) IssueListLabels(ctx context.Context, pids []uuid.UUID) ([]project.IssueLabel, error) {
	f.labelsCtx, f.labelProjects = ctx, slices.Clone(pids)
	return f.listLabels, f.err
}

func TestVisibleIssueProjectsConversion(t *testing.T) {
	wid, actor, first, second := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	f := &fakeIssueProjects{listProjectAnswers: listProjectAnswers{visible: []project.VisibleIssueProject{{ID: second, Role: shared.RoleGuest, OwnOnly: true}, {ID: first, Role: shared.RoleMember}}}}
	var p issue.ProjectPorts = issueProjects{projects: f}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	got, err := p.Visible(ctx, wid, actor)
	want := []issue.VisibleProject{{ID: second, Role: shared.RoleGuest, OwnOnly: true}, {ID: first, Role: shared.RoleMember}}
	if err != nil || !slices.Equal(got, want) || f.visibleCtx != ctx || f.workspace != wid || f.actor != actor {
		t.Fatalf("visible conversion/arguments/context: %+v, %v; fake %+v", got, err, f)
	}
	f.visible = nil
	if got, err := p.Visible(ctx, wid, actor); err != nil || got == nil || len(got) != 0 {
		t.Errorf("empty visible projects: %+v, %v", got, err)
	}
	f.err = errors.New("visible unavailable")
	if got, err := p.Visible(ctx, wid, actor); err != f.err || got != nil {
		t.Errorf("visible failure: %+v, %v", got, err)
	}
}

func TestListDirectoriesConversion(t *testing.T) {
	first, second := uuid.NewV7(), uuid.NewV7()
	pids := []uuid.UUID{second, first}
	f := &fakeIssueProjects{listProjectAnswers: listProjectAnswers{
		listStates: []project.IssueState{{ID: second, Name: "Zed", Group: "started", Default: true}, {ID: first, Name: "Alpha", Group: "completed"}},
		listLabels: []project.IssueLabel{{ID: second, Name: "Zed"}, {ID: first, Name: "Alpha"}},
	}}
	var p issue.ProjectPorts = issueProjects{projects: f}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	states, err := p.ListStates(ctx, pids)
	wantStates := []issue.StateRef{{ID: second, Name: "Zed", Group: "started", Default: true}, {ID: first, Name: "Alpha", Group: "completed"}}
	if err != nil || !slices.Equal(states, wantStates) || f.statesCtx != ctx || !slices.Equal(f.stateProjects, pids) {
		t.Fatalf("list states conversion/context/IDs: %+v, %v; fake %+v", states, err, f)
	}
	labels, err := p.ListLabels(ctx, pids)
	wantLabels := []issue.LabelRef{{ID: second, Name: "Zed"}, {ID: first, Name: "Alpha"}}
	if err != nil || !slices.Equal(labels, wantLabels) || f.labelsCtx != ctx || !slices.Equal(f.labelProjects, pids) {
		t.Fatalf("list labels conversion/context/IDs: %+v, %v; fake %+v", labels, err, f)
	}
	f.listStates, f.listLabels = nil, nil
	for _, empty := range [][]uuid.UUID{nil, {}} {
		states, err := p.ListStates(ctx, empty)
		if err != nil || states == nil || len(states) != 0 || !slices.Equal(f.stateProjects, empty) {
			t.Errorf("empty converted states: %+v, %v", states, err)
		}
		labels, err := p.ListLabels(ctx, empty)
		if err != nil || labels == nil || len(labels) != 0 || !slices.Equal(f.labelProjects, empty) {
			t.Errorf("empty converted labels: %+v, %v", labels, err)
		}
	}
	f.err = errors.New("list catalogs unavailable")
	if got, err := p.ListStates(ctx, pids); err != f.err || got != nil {
		t.Errorf("list states failure: %+v, %v", got, err)
	}
	if got, err := p.ListLabels(ctx, pids); err != f.err || got != nil {
		t.Errorf("list labels failure: %+v, %v", got, err)
	}
	profiles := &fakeIdentityProfiles{profiles: []identityapp.PublicProfile{{ID: second, FirstName: "Wrong source", DisplayName: "Display"}, {ID: first, DisplayName: "First"}}}
	var names issueapp.PublicProfiles = issueProfiles{profiles: profiles}
	got, err := names.DisplayNames(ctx, pids)
	if err != nil || !maps.Equal(got, map[uuid.UUID]string{second: "Display", first: "First"}) || profiles.ctx != ctx || !slices.Equal(profiles.asked[len(profiles.asked)-1], pids) {
		t.Fatalf("profile conversion/context/IDs: %v, %v", got, err)
	}
	profiles.err = errors.New("identity unavailable")
	if got, err := names.DisplayNames(ctx, pids); err != profiles.err || got != nil {
		t.Errorf("names failure: %v, %v", got, err)
	}
}

func TestListDirectoriesComposition(t *testing.T) {
	w := newIssueWorld(t)
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	w.newWorkItemIn(t, w.web, fmt.Sprintf(`{"name":"Assigned","assignee_ids":%s}`, idArray(w.ids["bob"], w.ids["walt"])))
	w.newWorkItem(t) // bob is assigned twice but has only one directory entry.
	for _, statement := range []struct {
		sql  string
		args []any
	}{
		{"UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2", []any{w.web, w.ids["bob"]}},
		{"UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2", []any{w.web, w.ids["walt"]}},
		{"UPDATE users SET is_active=false,first_name='Wrong source',display_name='Former Member' WHERE id=$1", []any{w.ids["bob"]}},
		{"UPDATE users SET display_name='Now Guest' WHERE id=$1", []any{w.ids["walt"]}},
	} {
		if _, err := w.pool.Exec(context.Background(), statement.sql, statement.args...); err != nil {
			t.Fatal(err)
		}
	}
	// One borrowed connection is occupied by the transaction. Any directory
	// escaping its ctx would wait for another and hit the deadline.
	one := poolOfOne(t, w.pool.Config().ConnString())
	projects := issueProjects{projects: project.Provide(one).IssueProjects}
	var assignees issueapp.AssigneeDirectory = issuepostgres.New(one)
	profiles := issueProfiles{profiles: identity.Provide(one).PublicProfiles}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := postgres.NewTxManager(one, 2*time.Second, postgres.ReadOnlySnapshot()).WithinTx(ctx, func(ctx context.Context) error {
		visible, err := projects.Visible(ctx, w.acme, w.ids["walt"])
		if err != nil {
			return err
		}
		if want := []issue.VisibleProject{{ID: w.web, Role: shared.RoleGuest, OwnOnly: true}}; !slices.Equal(visible, want) {
			t.Errorf("guest scope: %+v; want %+v", visible, want)
		}
		states, err := projects.ListStates(ctx, []uuid.UUID{w.web})
		if err != nil {
			return err
		}
		if len(states) != 5 || slices.ContainsFunc(states, func(s issue.StateRef) bool { return s.Group == "triage" }) {
			t.Errorf("list state catalog: %+v", states)
		}
		labels, err := projects.ListLabels(ctx, []uuid.UUID{w.web})
		if err != nil {
			return err
		}
		if want := []issue.LabelRef{{ID: w.bug, Name: "Bug"}}; !slices.Equal(labels, want) {
			t.Errorf("label catalog: %+v; want %+v", labels, want)
		}
		ids, err := assignees.AssigneeIDs(ctx, []uuid.UUID{w.web})
		if err != nil {
			return err
		}
		names, err := profiles.DisplayNames(ctx, ids)
		if err != nil {
			return err
		}
		if want := map[uuid.UUID]string{w.ids["bob"]: "Former Member", w.ids["walt"]: "Now Guest"}; !maps.Equal(names, want) {
			t.Errorf("former assignee display names: %v; want %v", names, want)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
