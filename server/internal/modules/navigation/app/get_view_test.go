package app_test

import (
	"errors"
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// webView and acmeView are bob's views: one of the project web, one of the workspace acme's own.
func webView(w *world) domain.View {
	id := uuid.NewV7()
	w.names[id] = "webView"
	v := domain.View{ID: id, WorkspaceID: acme, ProjectID: &web, OwnedByID: bob, CreatedAt: moment, UpdatedAt: moment,
		Content: domain.Content{Name: "Web's"}}
	w.views[id] = v
	return v
}

func acmeView(w *world) domain.View {
	id := uuid.NewV7()
	w.names[id] = "acmeView"
	v := domain.View{ID: id, WorkspaceID: acme, OwnedByID: bob, CreatedAt: moment, UpdatedAt: moment, Content: domain.Content{Name: "Acme's"}}
	w.views[id] = v
	return v
}

func get(t *testing.T, w *world, id uuid.UUID) (domain.View, error) {
	return app.NewGetView(w.reads()).Execute(as(t), id)
}

// getView reads the view without a lock first, then chooses its action by its kind (P3b spec 2.5): a project's view
// is view.read on its project, a workspace's is workspace_view.read on the workspace alone. All in one read
// transaction; a project whose views are turned off is read as ever.
func TestGettingAViewChoosesItsActionByItsKind(t *testing.T) {
	w := newWorld()
	w.projects[web] = app.ProjectFacts{WorkspaceID: acme}
	pv, wv := webView(w), acmeView(w)
	for _, tt := range []struct {
		view  domain.View
		calls []string
	}{
		{pv, []string{"begin read", "view webView", "authorize alice view.read acme/web", "end read"}},
		{wv, []string{"begin read", "view acmeView", "authorize alice workspace_view.read acme/-", "end read"}},
	} {
		w.calls = nil
		got, err := get(t, w, tt.view.ID)
		if err != nil || !reflect.DeepEqual(got, tt.view) {
			t.Errorf("get %s = %+v, %v; want the view", tt.view.Name, got, err)
		}
		wantCalls(t, w, tt.calls...)
	}
}

// Every view getView cannot answer is navigation.view_not_found, the one value a view that is not there answers
// (spec 2.5): a view whose project or workspace the caller does not see, never project.not_found nor
// workspace.not_found; and another's view while the caller sees only his own (Grant.OwnOnly), also when the rule
// refuses his role, before the 403. His own he gets, and a role refused on a view he sees is 403.
func TestGettingAViewNotSeen(t *testing.T) {
	ownOnly := shared.Grant{WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleGuest, OwnOnly: true}
	for _, tt := range []struct {
		name    string
		project bool // the view is the project's, else the workspace's
		mine    bool
		grant   shared.Grant
		refusal error
		want    func(*testing.T, error)
	}{
		{"its project not seen", true, false, shared.Grant{}, shared.ErrNotVisible, wantErr(domain.ErrViewNotFound)},
		{"its workspace not seen", false, false, shared.Grant{}, shared.ErrNotVisible, wantErr(domain.ErrViewNotFound)},
		{"another's, own only", true, false, ownOnly, nil, wantErr(domain.ErrViewNotFound)},
		{"another's, own only, role refused", true, false, ownOnly, shared.Forbidden(), wantErr(domain.ErrViewNotFound)},
		{"his own, own only, role refused", true, true, ownOnly, shared.Forbidden(), want403},
		{"a project's role refused", true, false, member, shared.Forbidden(), want403},
		{"a workspace's role refused", false, false, shared.Grant{WorkspaceRole: shared.RoleGuest}, shared.Forbidden(), want403},
		{"his own, own only", true, true, ownOnly, nil, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newWorld()
			v := acmeView(w)
			if tt.project {
				v = webView(w)
			}
			if tt.mine {
				v.OwnedByID = alice
				w.views[v.ID] = v
			}
			w.decide = func(shared.Action, shared.Target, bool) (shared.Grant, error) { return tt.grant, tt.refusal }
			got, err := get(t, w, v.ID)
			if tt.want == nil {
				if err != nil || !reflect.DeepEqual(got, v) {
					t.Fatalf("get = %+v, %v; want his view", got, err)
				}
				return
			}
			tt.want(t, err)
			if errors.Is(err, domain.ErrProjectNotFound) || errors.Is(err, domain.ErrWorkspaceNotFound) {
				t.Errorf("err = %v: it names the view's container", err)
			}
		})
	}
	w := newWorld()
	_, err := get(t, w, uuid.NewV7())
	wantSame(t, err, domain.ErrViewNotFound)
	if len(w.calls) != 3 || w.calls[2] != "end read" {
		t.Errorf("a view that is not there: calls %q; want its read alone", w.calls)
	}
}
