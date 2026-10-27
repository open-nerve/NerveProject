package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	issuepg "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Using issue.read or a membership-only directory would reject Walt's public
// project, or hide his drafts once it becomes private. Real access facts and
// the runtime role also prove that no-op saves preserve persisted audit stamps.
func TestDraftAppPersonalProjectsWithRealAuthorization(t *testing.T) {
	r := newSplitRoles(t)
	w := newIssueWorldOn(t, r.serverURL, r.owner, func(*config.Config) {})
	store := issuepg.New(r.server)
	p := issueProjects{projects: project.Provide(r.server).IssueProjects}
	ws := issueWorkspaces{directory: workspace.Provide(r.server).WorkspaceDirectory}
	tx := postgres.NewTxManager(r.server, time.Second)
	readTx := postgres.NewTxManager(r.server, time.Second, postgres.ReadOnlySnapshot())
	locks := issueapp.NewDraftLocks(issueapp.NewLocks(p, store, authorizerOn(r.server), nil, tx), ws, p, store)
	create := issueapp.NewCreateDraft(issueapp.CreateDraftDeps{Locks: locks, Workspaces: ws, Drafts: store, Clock: clock.System{}})
	update := issueapp.NewUpdateDraft(issueapp.UpdateDraftDeps{Locks: locks, Drafts: store, Clock: clock.System{}})
	get := issueapp.NewGetDraft(issueapp.GetDraftDeps{Locks: locks, ReadTx: readTx})
	list := issueapp.NewListDrafts(issueapp.ListDraftsDeps{Locks: locks, ReadTx: readTx, Workspaces: ws, Projects: p, Drafts: store})
	ctx := shared.WithActor(context.Background(), shared.Actor{UserID: w.ids["walt"]})
	d, err := create.Execute(ctx, "acme", domain.NewDraft{ProjectID: &w.web, Payload: []byte(`{"name":""}`)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.owner.Exec(ctx, "UPDATE projects SET archived_at=now() WHERE id=$1", w.web); err != nil {
		t.Fatal(err)
	}
	second, err := create.Execute(ctx, "acme", domain.NewDraft{ProjectID: &w.web, Payload: []byte(`{}`)})
	if err != nil {
		t.Fatalf("save archived public: %v", err)
	}
	if _, err := r.owner.Exec(ctx, "UPDATE projects SET network=0 WHERE id=$1", w.web); err != nil {
		t.Fatal(err)
	}
	unchanged, err := update.Execute(ctx, d.ID, domain.DraftPatch{})
	if err != nil || !reflect.DeepEqual(unchanged, d) {
		t.Fatalf("no-op=%+v want=%+v err=%v", unchanged, d, err)
	}
	own, err := get.Execute(ctx, d.ID)
	if err != nil || !reflect.DeepEqual(own, d) {
		t.Fatalf("own=%+v err=%v", own, err)
	}
	page, err := list.Execute(ctx, "acme", nil, nil)
	if err != nil || page.Total != 2 || len(page.Data) != 2 {
		t.Fatalf("page=%+v err=%v", page, err)
	}
	_, err = update.Execute(ctx, d.ID, domain.DraftPatch{SetProject: true, ProjectID: &w.web})
	var invalid *shared.Error
	if !errors.As(err, &invalid) || invalid.Kind != shared.KindInvalid || len(invalid.Fields) != 1 || invalid.Fields[0].Code != shared.FieldNotAllowed {
		t.Fatalf("explicit invisible project=%v", err)
	}
	cleared, err := update.Execute(ctx, d.ID, domain.DraftPatch{SetProject: true})
	if err != nil || cleared.ProjectID != nil {
		t.Fatalf("clear=%+v err=%v", cleared, err)
	}
	if err := issueapp.NewDeleteDraft(issueapp.DeleteDraftDeps{Locks: locks, Drafts: store, Clock: clock.System{}}).Execute(ctx, second.ID); err != nil {
		t.Fatalf("delete after loss of visibility=%v", err)
	}
	_, err = get.Execute(shared.WithActor(context.Background(), shared.Actor{UserID: w.ids["wanda"]}), d.ID)
	if !errors.Is(err, domain.ErrDraftNotFound) {
		t.Fatalf("admin reading other's draft=%v", err)
	}
	if _, err := r.owner.Exec(ctx, "UPDATE workspace_members SET role=5 WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["bob"]); err != nil {
		t.Fatal(err)
	}
	_, err = get.Execute(shared.WithActor(context.Background(), shared.Actor{UserID: w.ids["bob"]}), d.ID)
	var forbidden *shared.Error
	if !errors.As(err, &forbidden) || forbidden.Kind != shared.KindForbidden {
		t.Fatalf("guest reading other's draft=%v", err)
	}
}
