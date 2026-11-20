package app_test

import (
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// own is v made alice's, in w.
func own(w *world, v domain.View) domain.View {
	v.OwnedByID = alice
	w.views[v.ID] = v
	return v
}

// update runs alice's change p of the view id in w, with the kinds' words.
func update(t *testing.T, w *world, project, workspace *words, id uuid.UUID, p domain.Patch) (domain.View, error) {
	return app.NewUpdateView(w.writes(project, workspace)).Execute(as(t), id, p)
}

// rename is a change of a view's name alone.
func rename(name string) domain.Patch { return domain.Patch{Name: &name} }

// An update reads the view without a lock and decides its kind's update on it, then checks the body by its kind's
// words, outside any transaction; then, in its transaction, takes the workspace's row FOR SHARE, for a project's view
// the project's after it, and the view's row FOR UPDATE, last; decides again on the view as read under them; and only
// then reads the clock and writes (P3b spec 2.5, 2.6). Each field the body gives replaces the view's whole, never
// merged into it; what it does not give stays; the view changed at that moment, by the caller (spec 2.4).
func TestUpdatingAView(t *testing.T) {
	high, none := []string{"high"}, []string{"none"}
	for _, tt := range []struct {
		kind  string
		view  func(*world) domain.View
		calls []string
	}{
		{"project", webView, []string{"view webView", "authorize alice view.update acme/web", "words project",
			"begin write", "share workspace acme", "share project web", "lock view webView", "authorize alice view.update acme/web",
			"now", "update webView by alice", "end write"}},
		{"workspace", acmeView, []string{"view acmeView", "authorize alice workspace_view.update acme/-", "words workspace",
			"begin write", "share workspace acme", "lock view acmeView", "authorize alice workspace_view.update acme/-",
			"now", "update acmeView by alice", "end write"}},
	} {
		t.Run(tt.kind, func(t *testing.T) {
			w := newWorld()
			v := tt.view(w)
			v.Description = "kept"
			v.Filters = domain.Filters{ListFilters: shared.ListFilters{Priority: high, StateGroup: []string{"started"}}}
			v.CreatedAt = moment.AddDate(0, 0, -1)
			v.UpdatedAt = v.CreatedAt
			v = own(w, v)
			project, workspace := noWords(w)
			got, err := update(t, w, project, workspace, v.ID, domain.Patch{Name: new("  New  "),
				Filters: &domain.Filters{ListFilters: shared.ListFilters{StartDate: none}}})
			if err != nil {
				t.Fatal(err)
			}
			wantCalls(t, w, tt.calls...)
			want := v
			want.Name, want.Filters, want.UpdatedAt = "New", domain.Filters{ListFilters: shared.ListFilters{StartDate: none}}, moment
			if !reflect.DeepEqual(got, want) {
				t.Errorf("answered %+v\nwant %+v", got, want)
			}
			if !reflect.DeepEqual(w.updated, []insertion{{want, alice}}) {
				t.Errorf("updated %+v; want the view by alice", w.updated)
			}
		})
	}
}

// An update answers its body's 422 after the 404s and the 403s, the owner's among them, and before any lock (spec
// 2.5): a bad body on a view that is not there, or that the caller does not see, is 404; on another's, or on a view
// whose role the rule refuses, 403; on his own, 422, even while the project's views are off or the view is deleted
// meanwhile, which only the locks would read. A refusal before the 422 asks no words.
func TestTheRefusalOrderOfAnUpdate(t *testing.T) {
	for _, tt := range []struct {
		name    string
		project bool // the view is the project's, else the workspace's
		mine    bool
		world   func(*world, domain.View)
		body    domain.Patch
		want    func(*testing.T, error)
		words   bool // the words were asked
	}{
		{"bad words × no such view", true, true, func(w *world, v domain.View) { delete(w.views, v.ID) }, rename("v"), notFound, false},
		{"bad words × its project not seen", true, true, on(decides(shared.Grant{}, shared.ErrNotVisible)), rename("v"), notFound, false},
		{"bad name × its workspace not seen", false, true, on(decides(shared.Grant{}, shared.ErrNotVisible)), rename(" "), notFound, false},
		{"bad words × another's, own only", true, false, on(decides(guestOwnOnly, shared.Forbidden())), rename("v"), notFound, false},
		{"bad words × another's", true, false, nil, rename("v"), want403, false},
		{"bad name × another's workspace view", false, false, nil, rename(" "), want403, false},
		{"bad words × a role refused", true, true, on(decides(member, shared.Forbidden())), rename("v"), want403, false},
		{"bad words", true, true, nil, rename("v"), want422, true},
		{"bad name", false, true, nil, rename(" "), want422, true},
		{"bad words × views off", true, true, on(func(w *world) { w.projects[web] = app.ProjectFacts{WorkspaceID: acme} }), rename("v"), want422, true},
		{"bad words × deleted while a lock would wait", true, true, func(w *world, v domain.View) { w.viewsLocked = map[uuid.UUID]*domain.View{v.ID: nil} },
			rename("v"), want422, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newWorld()
			v := acmeView(w)
			if tt.project {
				v = webView(w)
			}
			if tt.mine {
				v = own(w, v)
			}
			if tt.world != nil {
				tt.world(w, v)
			}
			project, workspace := noWords(w)
			problem := []shared.FieldError{{Field: "filters.state_id[0]", Code: shared.FieldNotAllowed, Message: "is not allowed"}}
			project.problems, workspace.problems = problem, problem
			_, err := update(t, w, project, workspace, v.ID, tt.body)
			tt.want(t, err)
			wroteNothing(t, w)
			asked := slices.Contains(w.calls, "words project") || slices.Contains(w.calls, "words workspace")
			if asked != tt.words {
				t.Errorf("the words asked = %v; want %v (calls %q)", asked, tt.words, w.calls)
			}
			if slices.Contains(w.calls, "begin write") {
				t.Errorf("a lock was taken: %q", w.calls)
			}
		})
	}
}

// An update answers every problem of its body in one 422 (spec 2.5): those of its name, its description and its icon
// first, then its view's kind's words', given the filters and the display filters the body gives, nil for those it
// does not, and never the other kind's.
func TestEveryProblemOfAChangeInOneRefusal(t *testing.T) {
	inUse := "neither"
	display := shared.DefaultListDisplayFilters()
	for _, tt := range []struct {
		kind    string
		view    func(*world) domain.View
		body    domain.Patch
		filters map[string][]string
		display *shared.ListDisplayFilters
	}{
		{"project", webView, domain.Patch{Name: new(" "), Description: new("a\x00b"), LogoProps: &shared.LogoProps{InUse: &inUse},
			Filters: &domain.Filters{ProjectID: []string{}}}, map[string][]string{"project_id": {}}, nil},
		{"workspace", acmeView, domain.Patch{Name: new(" "), Description: new("a\x00b"), LogoProps: &shared.LogoProps{InUse: &inUse},
			DisplayFilters: &display}, nil, &display},
	} {
		t.Run(tt.kind, func(t *testing.T) {
			w := newWorld()
			v := own(w, tt.view(w))
			project, workspace := noWords(w)
			wordProblem := shared.FieldError{Field: "filters.project_id", Code: shared.FieldNotAllowed}
			project.problems, workspace.problems = []shared.FieldError{wordProblem}, []shared.FieldError{wordProblem}
			_, err := update(t, w, project, workspace, v.ID, tt.body)
			var got []string
			for _, f := range fieldsOf(t, err) {
				got = append(got, f.Field+" "+f.Code)
			}
			want := []string{"name too_short", "description invalid_format", "logo_props.in_use invalid_format", "filters.project_id not_allowed"}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("fields %q\nwant %q", got, want)
			}
			asked, other := project, workspace
			if tt.kind == "workspace" {
				asked, other = workspace, project
			}
			if !reflect.DeepEqual(asked.filters, tt.filters) || asked.display != tt.display {
				t.Errorf("the %s words were given %v, %p; want %v, %p", tt.kind, asked.filters, asked.display, tt.filters, tt.display)
			}
			if slices.Contains(w.calls, "words "+other.kind) {
				t.Errorf("the other kind's words were asked: %q", w.calls)
			}
			wroteNothing(t, w)
		})
	}
}

// An update that changes nothing, {} or each field it gives as the view already has it, after its name is cleaned,
// answers the view as read under its lock and writes nothing, nor reads the clock (spec 2.4): it is compared with
// that view, not the one read before the locks, so a change made while the lock waited is what it is compared with
// and what it changes. It is refused as any update first: another's is 403, and views turned off are 409.
func TestAnUpdateThatChangesNothing(t *testing.T) {
	high := []string{"high"}
	viewOf := func(w *world) domain.View {
		v := webView(w)
		v.Description, v.Filters = "d", domain.Filters{ListFilters: shared.ListFilters{Priority: high}, CycleID: []string{}}
		v.DisplayFilters, v.DisplayProperties = shared.DefaultListDisplayFilters(), shared.DefaultListDisplayProperties()
		return own(w, v)
	}
	same := func(v domain.View) domain.Patch {
		return domain.Patch{Name: new(" " + v.Name + " "), Description: &v.Description, LogoProps: &v.LogoProps,
			Filters:        &domain.Filters{ListFilters: shared.ListFilters{Priority: []string{"high"}}, CycleID: []string{}},
			DisplayFilters: &v.DisplayFilters, DisplayProperties: &v.DisplayProperties}
	}
	for _, tt := range []struct {
		name string
		body func(domain.View) domain.Patch
	}{
		{"{}", func(domain.View) domain.Patch { return domain.Patch{} }},
		{"every field as it is", same},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newWorld()
			v := viewOf(w)
			project, workspace := noWords(w)
			got, err := update(t, w, project, workspace, v.ID, tt.body(v))
			if err != nil || !reflect.DeepEqual(got, v) {
				t.Fatalf("answered %+v, %v; want the view as it is", got, err)
			}
			wroteNothing(t, w)
			if last := w.calls[len(w.calls)-2:]; !reflect.DeepEqual(last, []string{"authorize alice view.update acme/web", "end write"}) {
				t.Errorf("calls %q; want the decision under the locks last", w.calls)
			}
		})
	}
	t.Run("as changed while the lock waited", func(t *testing.T) {
		w := newWorld()
		v := viewOf(w)
		meanwhile := v
		meanwhile.Name, meanwhile.Description = "Renamed meanwhile", "changed meanwhile"
		w.viewsLocked = map[uuid.UUID]*domain.View{v.ID: &meanwhile}
		project, workspace := noWords(w)
		got, err := update(t, w, project, workspace, v.ID, rename("Renamed meanwhile"))
		if err != nil || !reflect.DeepEqual(got, meanwhile) {
			t.Fatalf("answered %+v, %v; want the view as the lock read it", got, err)
		}
		wroteNothing(t, w)
		got, err = update(t, w, project, workspace, v.ID, rename(v.Name))
		want := meanwhile
		want.Name, want.UpdatedAt = v.Name, moment
		if err != nil || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(w.updated, []insertion{{want, alice}}) {
			t.Errorf("answered %+v, %v, updated %+v; want the view as the lock read it, renamed back", got, err, w.updated)
		}
	})
	for _, tt := range []struct {
		name  string
		world func(*world, domain.View)
		want  func(*testing.T, error)
	}{
		{"another's", func(w *world, v domain.View) { v.OwnedByID = bob; w.views[v.ID] = v }, want403},
		{"views off", func(w *world, _ domain.View) { w.projects[web] = app.ProjectFacts{WorkspaceID: acme} }, wantErr(domain.ErrViewsDisabled)},
	} {
		t.Run("{} × "+tt.name, func(t *testing.T) {
			w := newWorld()
			v := viewOf(w)
			tt.world(w, v)
			project, workspace := noWords(w)
			_, err := update(t, w, project, workspace, v.ID, domain.Patch{})
			tt.want(t, err)
			wroteNothing(t, w)
		})
	}
}
