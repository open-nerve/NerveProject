package app_test

import (
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// creating is a creation of a view on the route of a kind: a project's views by the project's id, or a workspace's by
// its slug.
type creating struct {
	kind    string
	project uuid.UUID
	slug    string
}

var (
	onWeb  = creating{kind: "project", project: web}
	onAcme = creating{kind: "workspace", slug: "acme"}
)

// create runs the creation c of body as alice in w, with the kinds' words.
func (c creating) create(t *testing.T, w *world, project, workspace *words, body domain.NewView) (domain.View, error) {
	d := w.creations(project, workspace)
	if c.kind == "project" {
		return app.NewCreateProjectView(d).Execute(as(t), c.project, body)
	}
	return app.NewCreateWorkspaceView(d).Execute(as(t), c.slug, body)
}

// noWords are the kinds' words refusing nothing.
func noWords(w *world) (*words, *words) {
	return &words{w: w, kind: "project"}, &words{w: w, kind: "workspace"}
}

// A creation checks its body, its kind's words among it, before it reads anything; then reads what its route names
// and decides on it, outside any transaction; then, in its transaction, takes the workspace's row FOR SHARE and, for
// a project's view, the project's after it, reads them again under the locks and decides again; and only after the
// last lock reads the clock, once, and stores the view (P3b spec 2.5, 2.6): its name cleaned, the shared display
// defaults for what the body leaves out, of the workspace and the project as the locks read them, owned, created and
// last changed by the caller at that moment.
func TestCreatingAView(t *testing.T) {
	for _, tt := range []struct {
		route   creating
		project *uuid.UUID
		calls   []string
	}{
		{onWeb, &web, []string{"words project", "facts web", "authorize alice view.create acme/web",
			"begin write", "share workspace acme", "share project web", "authorize alice view.create acme/web", "now", "insert", "end write"}},
		{onAcme, nil, []string{"words workspace", "slug acme", "authorize alice workspace_view.create acme/-",
			"begin write", "share slug acme", "authorize alice workspace_view.create acme/-", "now", "insert", "end write"}},
	} {
		t.Run(tt.route.kind, func(t *testing.T) {
			w := newWorld()
			got, err := tt.route.create(t, w, &words{w: w, kind: "project"}, &words{w: w, kind: "workspace"}, domain.NewView{Name: "  Mine  "})
			if err != nil {
				t.Fatal(err)
			}
			wantCalls(t, w, tt.calls...)
			want := domain.View{ID: got.ID, WorkspaceID: acme, ProjectID: tt.project, OwnedByID: alice, CreatedAt: moment, UpdatedAt: moment,
				Content: domain.Content{Name: "Mine", DisplayFilters: shared.DefaultListDisplayFilters(),
					DisplayProperties: shared.DefaultListDisplayProperties()}}
			if got.ID == (uuid.UUID{}) || !reflect.DeepEqual(got, want) {
				t.Errorf("answered %+v\nwant %+v", got, want)
			}
			if !reflect.DeepEqual(w.inserted, []insertion{{want, alice}}) {
				t.Errorf("inserted %+v; want the view by alice", w.inserted)
			}
		})
	}
}

// The refusals of a creation, in their order (spec 2.5): the body's 422, its words among it, before any read, so a
// bad body on a route that names nothing, or names what the caller does not see, is 422; then the 404 of what the
// route names, project.not_found or workspace.not_found, also when the caller does not see it; then the 403 of the
// role; then, under the locks, what they read again and decide again, which is final: a project or a workspace
// deleted while a lock waited is its 404, a decision that refuses now is its answer; and last a project's views
// turned off, 409, which a workspace's views do not have. A refused creation writes nothing and reads no clock.
func TestTheRefusalsOfACreation(t *testing.T) {
	badName := domain.NewView{Name: " "}
	refuse := func(err error) func(*world) {
		return func(w *world) {
			w.decide = func(shared.Action, shared.Target, bool) (shared.Grant, error) { return shared.Grant{}, err }
		}
	}
	refuseUnderLock := func(err error) func(*world) {
		return func(w *world) {
			w.decide = func(_ shared.Action, _ shared.Target, inTx bool) (shared.Grant, error) {
				if inTx {
					return shared.Grant{}, err
				}
				return member, nil
			}
		}
	}
	viewsOff := func(w *world) { w.projects[web] = app.ProjectFacts{WorkspaceID: acme} }
	both := func(a, b func(*world)) func(*world) { return func(w *world) { a(w); b(w) } }
	off := app.ProjectFacts{WorkspaceID: acme}
	for _, tt := range []struct {
		name  string
		route creating
		world func(*world)
		body  domain.NewView
		words bool // the kind's words refuse
		want  func(*testing.T, error)
	}{
		{"bad words × a project not seen", onWeb, refuse(shared.ErrNotVisible), domain.NewView{Name: "v"}, true, want422},
		{"bad name × no such project", creating{kind: "project", project: uuid.NewV7()}, nil, badName, false, want422},
		{"bad words × a workspace not seen", onAcme, refuse(shared.ErrNotVisible), domain.NewView{Name: "v"}, true, want422},
		{"bad name × no such workspace", creating{kind: "workspace", slug: "nope"}, nil, badName, false, want422},
		{"no such project", creating{kind: "project", project: uuid.NewV7()}, nil, domain.NewView{Name: "v"}, false, wantErr(domain.ErrProjectNotFound)},
		{"a project not seen", onWeb, refuse(shared.ErrNotVisible), domain.NewView{Name: "v"}, false, wantErr(domain.ErrProjectNotFound)},
		{"a project's role refused", onWeb, refuse(shared.Forbidden()), domain.NewView{Name: "v"}, false, want403},
		{"a role refused × views off", onWeb, both(viewsOff, refuse(shared.Forbidden())), domain.NewView{Name: "v"}, false, want403},
		{"views off", onWeb, viewsOff, domain.NewView{Name: "v"}, false, wantErr(domain.ErrViewsDisabled)},
		{"views turned off while the lock waited", onWeb, func(w *world) { w.underLock = map[uuid.UUID]*app.ProjectFacts{web: &off} },
			domain.NewView{Name: "v"}, false, wantErr(domain.ErrViewsDisabled)},
		{"project deleted while the lock waited", onWeb, func(w *world) { w.underLock = map[uuid.UUID]*app.ProjectFacts{web: nil} },
			domain.NewView{Name: "v"}, false, wantErr(domain.ErrProjectNotFound)},
		{"its workspace deleted while the lock waited", onWeb, func(w *world) { w.workspaceGone = map[uuid.UUID]bool{acme: true} },
			domain.NewView{Name: "v"}, false, wantErr(domain.ErrProjectNotFound)},
		{"a project refused under the lock", onWeb, refuseUnderLock(shared.Forbidden()), domain.NewView{Name: "v"}, false, want403},
		{"a project not seen under the lock", onWeb, refuseUnderLock(shared.ErrNotVisible), domain.NewView{Name: "v"}, false,
			wantErr(domain.ErrProjectNotFound)},
		{"no such workspace", creating{kind: "workspace", slug: "nope"}, nil, domain.NewView{Name: "v"}, false, wantErr(domain.ErrWorkspaceNotFound)},
		{"a workspace not seen", onAcme, refuse(shared.ErrNotVisible), domain.NewView{Name: "v"}, false, wantErr(domain.ErrWorkspaceNotFound)},
		{"a workspace's role refused", onAcme, refuse(shared.Forbidden()), domain.NewView{Name: "v"}, false, want403},
		{"workspace deleted while the lock waited", onAcme, func(w *world) { w.slugsLocked = map[string]uuid.UUID{} },
			domain.NewView{Name: "v"}, false, wantErr(domain.ErrWorkspaceNotFound)},
		{"a workspace refused under the lock", onAcme, refuseUnderLock(shared.Forbidden()), domain.NewView{Name: "v"}, false, want403},
		{"a workspace not seen under the lock", onAcme, refuseUnderLock(shared.ErrNotVisible), domain.NewView{Name: "v"}, false,
			wantErr(domain.ErrWorkspaceNotFound)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newWorld()
			if tt.world != nil {
				tt.world(w)
			}
			project, workspace := noWords(w)
			if tt.words {
				problem := []shared.FieldError{{Field: "filters.state_id[0]", Code: shared.FieldNotAllowed, Message: "is not allowed"}}
				project.problems, workspace.problems = problem, problem
			}
			_, err := tt.route.create(t, w, project, workspace, tt.body)
			tt.want(t, err)
			if len(w.inserted) != 0 {
				t.Errorf("inserted %+v; want nothing", w.inserted)
			}
			for _, call := range w.calls {
				if call == "now" {
					t.Error("the clock was read")
				}
			}
		})
	}
}

// A refusal of the body reads nothing: only the kind's words are asked.
func TestABadBodyReadsNothing(t *testing.T) {
	for _, route := range []creating{onWeb, onAcme} {
		t.Run(route.kind, func(t *testing.T) {
			w := newWorld()
			w.decide = func(shared.Action, shared.Target, bool) (shared.Grant, error) {
				return shared.Grant{}, shared.ErrNotVisible
			}
			project, workspace := noWords(w)
			_, err := route.create(t, w, project, workspace, domain.NewView{Name: " "})
			want422(t, err)
			wantCalls(t, w, "words "+route.kind)
		})
	}
}

// The toggle refuses only under the locks, after them and the decision: a project whose views were turned on while
// the lock waited takes the view, as its lock read it; and only a project's views have it.
func TestTheToggleUnderTheLock(t *testing.T) {
	w := newWorld()
	w.projects[web] = app.ProjectFacts{WorkspaceID: acme}
	w.underLock = map[uuid.UUID]*app.ProjectFacts{web: {WorkspaceID: acme, IssueViewsView: true}}
	project, workspace := noWords(w)
	if _, err := onWeb.create(t, w, project, workspace, domain.NewView{Name: "v"}); err != nil || len(w.inserted) != 1 {
		t.Fatalf("views turned on while the lock waited: %v, inserted %d; want the view", err, len(w.inserted))
	}
	w = newWorld()
	w.projects[web] = app.ProjectFacts{WorkspaceID: acme}
	project, workspace = noWords(w)
	_, err := onWeb.create(t, w, project, workspace, domain.NewView{Name: "v"})
	wantSame(t, err, domain.ErrViewsDisabled)
	wantCalls(t, w, "words project", "facts web", "authorize alice view.create acme/web",
		"begin write", "share workspace acme", "share project web", "authorize alice view.create acme/web", "end write")
}

// A creation answers every problem of its body in one 422 (spec 2.5): those of its name, its description and its
// icon first, in the body's order, then its kind's words', in theirs. The words are its kind's: a project's for a
// project's views, a workspace's for a workspace's, given the body's filters by key, an empty filter kept, and its
// display filters as given.
func TestEveryProblemOfABodyInOneRefusal(t *testing.T) {
	inUse := "neither"
	display := shared.DefaultListDisplayFilters()
	display.Layout = "gantt"
	body := domain.NewView{Name: " ", Description: "a\x00b", LogoProps: shared.LogoProps{InUse: &inUse},
		Filters:        domain.Filters{CycleID: []string{"c"}, ProjectID: []string{}},
		DisplayFilters: &display}
	ownProblems := []shared.FieldError{
		{Field: "name", Code: shared.FieldTooShort, Message: "must not be empty"},
		{Field: "description", Code: shared.FieldInvalidFormat},
		{Field: "logo_props.in_use", Code: shared.FieldInvalidFormat, Message: "must be emoji or icon"},
	}
	wordProblems := map[string][]shared.FieldError{
		"project":   {{Field: "filters.project_id", Code: shared.FieldNotAllowed}, {Field: "display_filters.layout", Code: shared.FieldNotAllowed}},
		"workspace": {{Field: "filters.cycle_id", Code: shared.FieldNotAllowed}, {Field: "display_filters.layout", Code: shared.FieldNotAllowed}},
	}
	for _, route := range []creating{onWeb, onAcme} {
		t.Run(route.kind, func(t *testing.T) {
			w := newWorld()
			project, workspace := noWords(w)
			project.problems, workspace.problems = wordProblems["project"], wordProblems["workspace"]
			_, err := route.create(t, w, project, workspace, body)
			fields := fieldsOf(t, err)
			want := append(append([]shared.FieldError(nil), ownProblems...), wordProblems[route.kind]...)
			if len(fields) != len(want) {
				t.Fatalf("fields %+v\nwant %+v", fields, want)
			}
			for i := range want {
				if fields[i].Field != want[i].Field || fields[i].Code != want[i].Code {
					t.Errorf("field %d = %s %s; want %s %s", i, fields[i].Field, fields[i].Code, want[i].Field, want[i].Code)
				}
			}
			asked, other := project, workspace
			if route.kind == "workspace" {
				asked, other = workspace, project
			}
			wantFilters := map[string][]string{"cycle_id": {"c"}, "project_id": {}}
			if !reflect.DeepEqual(asked.filters, wantFilters) || asked.display != &display {
				t.Errorf("the %s words were given %v, %p; want %v, %p", route.kind, asked.filters, asked.display, wantFilters, &display)
			}
			if other.filters != nil {
				t.Errorf("the other kind's words were asked: %v", other.filters)
			}
			wantCalls(t, w, "words "+route.kind)
		})
	}
}

func want422(t *testing.T, err error) {
	t.Helper()
	fieldsOf(t, err)
}

func want403(t *testing.T, err error) {
	t.Helper()
	wantForbidden(t, err)
}

func wantErr(want error) func(*testing.T, error) {
	return func(t *testing.T, err error) {
		t.Helper()
		wantSame(t, err, want)
	}
}
