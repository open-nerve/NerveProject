package app_test

import (
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// listing is a list of the views of a route's kind: a project's by its id, or a workspace's own by its slug.
func (c creating) list(t *testing.T, w *world) ([]domain.View, error) {
	if c.kind == "project" {
		return app.NewListProjectViews(w.reads()).Execute(as(t), c.project)
	}
	return app.NewListWorkspaceViews(w.reads()).Execute(as(t), c.slug)
}

// A list decides view.read on the project, or workspace_view.read on the workspace, and lists their views as the
// store answers them, all in one read transaction (P3b spec 2.2, 2.4): every view to a caller who sees all, only his
// own, filtered by the store, to one who sees only his own (Grant.OwnOnly). A project whose views are turned off is
// listed as ever.
func TestListingTheViews(t *testing.T) {
	ownOnly := shared.Grant{WorkspaceRole: shared.RoleAdmin, ProjectRole: shared.RoleGuest, ProjectAdmin: true, OwnOnly: true}
	for _, tt := range []struct {
		name  string
		route creating
		world func(*world)
		calls []string
	}{
		{"a project's", onWeb, nil, []string{"begin read", "facts web", "authorize alice view.read acme/web", "project views web owned any", "end read"}},
		{"a project's, own only", onWeb, func(w *world) {
			w.decide = func(shared.Action, shared.Target, bool) (shared.Grant, error) { return ownOnly, nil }
		}, []string{"begin read", "facts web", "authorize alice view.read acme/web", "project views web owned alice", "end read"}},
		{"a project's, views off", onWeb, func(w *world) { w.projects[web] = app.ProjectFacts{WorkspaceID: acme} },
			[]string{"begin read", "facts web", "authorize alice view.read acme/web", "project views web owned any", "end read"}},
		{"a workspace's", onAcme, nil, []string{"begin read", "slug acme", "authorize alice workspace_view.read acme/-",
			"workspace views acme owned any", "end read"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newWorld()
			if tt.world != nil {
				tt.world(w)
			}
			w.listed = []domain.View{{ID: uuid.NewV7(), Content: domain.Content{Name: "newer"}}, {ID: uuid.NewV7(), Content: domain.Content{Name: "older"}}}
			got, err := tt.route.list(t, w)
			if err != nil || !reflect.DeepEqual(got, w.listed) {
				t.Errorf("list = %+v, %v; want the store's, in its order", got, err)
			}
			wantCalls(t, w, tt.calls...)
		})
	}
}

// A list's refusals (spec 2.2): a project or a workspace that is not there, or that the caller does not see, is its
// route's 404, project.not_found or workspace.not_found; a role refused, 403. A refused list lists nothing.
func TestTheRefusalsOfAList(t *testing.T) {
	refuse := func(err error) func(*world) {
		return func(w *world) {
			w.decide = func(shared.Action, shared.Target, bool) (shared.Grant, error) { return shared.Grant{}, err }
		}
	}
	for _, tt := range []struct {
		name  string
		route creating
		world func(*world)
		want  func(*testing.T, error)
	}{
		{"no such project", creating{kind: "project", project: uuid.NewV7()}, nil, wantErr(domain.ErrProjectNotFound)},
		{"a project not seen", onWeb, refuse(shared.ErrNotVisible), wantErr(domain.ErrProjectNotFound)},
		{"a project's role refused", onWeb, refuse(shared.Forbidden()), want403},
		{"no such workspace", creating{kind: "workspace", slug: "nope"}, nil, wantErr(domain.ErrWorkspaceNotFound)},
		{"a workspace not seen", onAcme, refuse(shared.ErrNotVisible), wantErr(domain.ErrWorkspaceNotFound)},
		{"a workspace's guest", onAcme, refuse(shared.Forbidden()), want403},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newWorld()
			if tt.world != nil {
				tt.world(w)
			}
			got, err := tt.route.list(t, w)
			tt.want(t, err)
			if got != nil {
				t.Errorf("listed %+v; want nothing", got)
			}
			for _, call := range w.calls {
				if call == "project views web owned any" || call == "workspace views acme owned any" {
					t.Errorf("listed: %s", call)
				}
			}
		})
	}
}
