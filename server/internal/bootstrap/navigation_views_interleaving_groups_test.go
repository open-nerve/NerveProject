package bootstrap

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
	"uuid"
)

// The groups of TestViewWritesInterleave (navigation_views_interleaving_test.go).

// viewsWaitForADeletion: alice's deletion of a project or a workspace, a
// new one of its own each time, holds its row past its writes; her write
// of a view under it, a project's view or a workspace's own, which passed
// its check, waits for that row: under a deleted project at
// ShareNavigationProject, after its workspace's FOR SHARE; under a deleted
// workspace at its first lock, the workspace's FOR SHARE, by its slug for
// a creation of a workspace's view (ShareDirectoryWorkspace), else by its
// id. Once the deletion has committed, the write finds its project or its
// workspace gone under its lock: 404, byte for byte as on what never was,
// navigation.view_not_found on a view's route, project.not_found or
// workspace.not_found on the views' (spec 2.2, 2.5). It wrote nothing: no
// view named Late, and the view it names as before, but for its deletion
// with its parent, at the parent's moment, by its deleter, which the
// parent's job may have written since (spec 2.7).
func viewsWaitForADeletion(t *testing.T, w issueWorld) {
	for n, c := range []struct {
		name string
		// workspace: the workspace is deleted, else the project; projects:
		// the write is on a project's views, else on a workspace's own.
		workspace, projects bool
		method              string
	}{
		{"a project's view created, its project deleted", false, true, http.MethodPost},
		{"a project's view changed, its project deleted", false, true, http.MethodPatch},
		{"a project's view deleted, its project deleted", false, true, http.MethodDelete},
		{"a workspace's view created, its workspace deleted", true, false, http.MethodPost},
		{"a workspace's view changed, its workspace deleted", true, false, http.MethodPatch},
		{"a workspace's view deleted, its workspace deleted", true, false, http.MethodDelete},
		{"a project's view changed, its workspace deleted", true, true, http.MethodPatch},
	} {
		t.Run(c.name, func(t *testing.T) {
			letter := string(rune('A' + n))
			slug, parents, parent := "acme", "workspaces", w.acme
			if c.workspace {
				slug = "doomed-" + strings.ToLower(letter)
				var doomed struct{ ID uuid.UUID }
				decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/workspaces", `{"name":"Doomed `+letter+`","slug":"`+slug+`"}`,
					http.StatusCreated), &doomed)
				parent = doomed.ID
			}
			views, absent, code := workspaceViewsPath(slug), workspaceViewsPath("absent"), "workspace.not_found"
			if c.projects {
				p := createdProject(t, w.contract, w.base, w.tokens["alice"], slug, "Doomed "+letter, "DM"+letter)
				w.changeProject(t, p, `{"issue_views_view":true}`)
				views, absent, code, parents, parent = projectViewsPath(p), projectViewsPath(uuid.NewV7()), "project.not_found", "projects", p
			}
			deletion, table, statement := alicesProjectWrite(http.MethodDelete, "/api/v0/projects/"+parent.String(), ""), "projects",
				"ShareNavigationProject"
			if c.workspace {
				deletion, table, statement = alicesWorkspaceWrite(http.MethodDelete, "/api/v0/workspaces/"+slug, ""), "workspaces",
					"ShareDirectoryWorkspaceByID"
			}
			path, body, view, before := views, `{"name":"Late"}`, (*uuid.UUID)(nil), ""
			switch c.method {
			case http.MethodPost:
				if !c.projects {
					statement = "ShareDirectoryWorkspace"
				}
			case http.MethodDelete:
				body = ""
				fallthrough
			default:
				v, _ := w.newView(t, "alice", views, `{"name":"Kept"}`)
				path, absent, code, view, before = viewPath(v.ID), viewPath(uuid.NewV7()), "navigation.view_not_found", &v.ID, viewColumns(t, w.pool, v.ID)
			}
			var deleted, written exchange
			w.viewWaits(t, table, statement, view, deletion, writesView("alice", c.method, path, body), &deleted, &written)
			w.answered(t, deleted, http.StatusNoContent)
			w.refusedAsAbsent(t, written, "alice", absent, body, code)
			if view != nil {
				unchangedButDeletedWith(t, w.pool, *view, before, parents, parent)
			} else if n := viewsNamed(t, w.pool, parent, "Late"); n != 0 {
				t.Errorf("%d views named Late; want none", n)
			}
		})
	}
}

// viewsWaitForTheToggle: alice's change of Web's issue_views_view holds
// Web's row past its writes; bob's write of a view of Web, which passed
// its check, waits for it at ShareNavigationProject. Turned off, the
// write reads it off under its lock and is refused, 409
// navigation.views_disabled, and wrote nothing; turned on, though off at
// the write's check, which does not judge it, the write goes through, at a
// moment after the change: the toggle is judged under the lock alone (spec
// 2.5).
func viewsWaitForTheToggle(t *testing.T, w issueWorld) {
	web := projectViewsPath(w.web)
	changed, _ := w.newView(t, "bob", web, `{"name":"Changed while turned off"}`)
	deleted, _ := w.newView(t, "bob", web, `{"name":"Deleted while turned on"}`)
	for _, c := range []struct {
		name   string
		on     bool
		method string
		view   *uuid.UUID
	}{
		{"turned off, a creation waiting", false, http.MethodPost, nil},
		{"turned off, a change waiting", false, http.MethodPatch, &changed.ID},
		{"turned on, a creation waiting", true, http.MethodPost, nil},
		{"turned on, a deletion waiting", true, http.MethodDelete, &deleted.ID},
	} {
		t.Run(c.name, func(t *testing.T) {
			w.changeProject(t, w.web, fmt.Sprintf(`{"issue_views_view":%t}`, !c.on))
			path, body, before := web, `{"name":"`+c.name+`"}`, ""
			if c.view != nil {
				path, before = viewPath(*c.view), viewColumns(t, w.pool, *c.view)
			}
			if c.method == http.MethodDelete {
				body = ""
			}
			var turned, written exchange
			w.viewWaits(t, "projects", "ShareNavigationProject", c.view, alicesProjectWrite(http.MethodPatch, "/api/v0/projects/"+w.web.String(),
				fmt.Sprintf(`{"issue_views_view":%t}`, c.on)), writesView("bob", c.method, path, body), &turned, &written)
			w.answered(t, turned, http.StatusOK)
			switch {
			case !c.on:
				w.refused(t, written, http.StatusConflict, "navigation.views_disabled")
				if c.view != nil {
					unchanged(t, w.pool, *c.view, before)
				} else if n := viewsNamed(t, w.pool, w.web, c.name); n != 0 {
					t.Errorf("%d views named %q; want none", n, c.name)
				}
			case c.view == nil:
				var v viewHTTP
				decodeAnswer(t, w.answered(t, written, http.StatusCreated), &v)
				if v.Name != c.name || v.CreatedAt.Before(turned.released) {
					t.Errorf("created %+v; want it named %q, at a moment no earlier than the change's commit at %s", v, c.name, turned.released)
				}
			default:
				w.answered(t, written, http.StatusNoContent)
				viewDeletedBy(t, w.pool, *c.view, w.ids["bob"], w.ids["bob"], turned.released, time.Now())
			}
		})
	}
}

// viewsWaitForAMember: alice's change of a member, or of what Vault shows
// its guests, holds the row of Vault, private, or of acme past its writes;
// his write of a view, which passed its check as he was, waits for it at
// ShareNavigationProject or ShareDirectoryWorkspaceByID, and once the
// change has committed is decided as he is now (spec 2.2, 2.5): bob, made
// Vault's guest, may change his own view no more, 403 forbidden; walt,
// removed from Vault, sees neither it nor his view of it, 404
// navigation.view_not_found byte for byte as a view that never was; adam,
// acme's admin and Vault's guest, whose deletion of bob's view of Vault
// passed its check while Vault showed its guests every view, finds it
// hidden from him once Vault shows them only their own (View.HiddenFrom),
// 404 as well; wanda, made acme's member, may delete bob's view of acme no
// more, 403 forbidden. Each refused write wrote nothing.
// walt, made acme's admin, changes his own view of acme as he would
// before, at a moment after the change.
func viewsWaitForAMember(t *testing.T, w issueWorld, vault uuid.UUID) {
	inVault, inAcme := projectViewsPath(vault), workspaceViewsPath("acme")
	bobs, _ := w.newView(t, "bob", inVault, `{"name":"Bob's"}`)
	forAdam, _ := w.newView(t, "bob", inVault, `{"name":"For adam"}`)
	walts, _ := w.newView(t, "walt", inVault, `{"name":"Walt's"}`)
	forWanda, _ := w.newView(t, "bob", inAcme, `{"name":"For wanda"}`)
	waltsOfAcme, _ := w.newView(t, "walt", inAcme, `{"name":"Walt's of acme"}`)
	w.changeProject(t, vault, `{"guest_view_all_features":true}`)
	ofVault := func(name string) string {
		return "/api/v0/project-members/" + projectMemberships(t, w.pool, w.ids[name], vault)[0].String()
	}
	ofAcme := func(name string) string {
		return "/api/v0/workspace-members/" + queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id = $1 AND member_id = $2",
			w.acme, w.ids[name])[0].String()
	}
	for _, c := range []struct {
		name, writer, method string
		view                 uuid.UUID
		change               racingWrite
		changed              int
		table, statement     string // what the write waits for, and at
		status               int
		code                 string
	}{
		{"made a guest", "bob", http.MethodPatch, bobs.ID, alicesProjectWrite(http.MethodPatch, ofVault("bob"), `{"role":5}`), http.StatusOK,
			"projects", "ShareNavigationProject", http.StatusForbidden, "forbidden"},
		{"removed", "walt", http.MethodDelete, walts.ID, alicesProjectWrite(http.MethodDelete, ofVault("walt"), ""), http.StatusNoContent,
			"projects", "ShareNavigationProject", http.StatusNotFound, "navigation.view_not_found"},
		{"shown only his own", "adam", http.MethodDelete, forAdam.ID, alicesProjectWrite(http.MethodPatch, "/api/v0/projects/"+vault.String(),
			`{"guest_view_all_features":false}`), http.StatusOK, "projects", "ShareNavigationProject", http.StatusNotFound, "navigation.view_not_found"},
		{"made a member of acme", "wanda", http.MethodDelete, forWanda.ID, alicesWorkspaceWrite(http.MethodPatch, ofAcme("wanda"), `{"role":15}`),
			http.StatusOK, "workspaces", "ShareDirectoryWorkspaceByID", http.StatusForbidden, "forbidden"},
		{"made an admin of acme", "walt", http.MethodPatch, waltsOfAcme.ID, alicesWorkspaceWrite(http.MethodPatch, ofAcme("walt"), `{"role":20}`),
			http.StatusOK, "workspaces", "ShareDirectoryWorkspaceByID", http.StatusOK, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			body := `{"name":"Late"}`
			if c.method == http.MethodDelete {
				body = ""
			}
			before := viewColumns(t, w.pool, c.view)
			var changed, written exchange
			w.viewWaits(t, c.table, c.statement, &c.view, c.change, writesView(c.writer, c.method, viewPath(c.view), body), &changed, &written)
			w.answered(t, changed, c.changed)
			switch c.status {
			case http.StatusNotFound:
				w.refusedAsAbsent(t, written, c.writer, viewPath(uuid.NewV7()), body, c.code)
			case http.StatusForbidden:
				w.refused(t, written, c.status, c.code)
			default:
				var v viewHTTP
				decodeAnswer(t, w.answered(t, written, c.status), &v)
				if read, _ := w.readView(t, c.writer, c.view); v.Name != "Late" || v.UpdatedAt.Before(changed.released) || !sameView(read, v) {
					t.Errorf("changed %+v, read %+v; want it named Late, at a moment no earlier than the change's commit at %s", v, read, changed.released)
				}
				return
			}
			unchanged(t, w.pool, c.view, before)
		})
	}
}

// twoWritesOfAView: two writes of one of bob's views of Web serialize on
// the view's row (spec 2.6): the first holds it past its writes; the
// second, which passed its check on the view as it was before the first,
// waits at LockView, and once the first has committed reads the view
// again under its lock and writes over what the first wrote (spec 2.5):
// bob's change of the description keeps his change of the name before it,
// no update lost; his change of the name back to the one his check read
// is a change of the view his lock read, written; after alice's deletion
// of the view, his change finds none, 404 navigation.view_not_found byte
// for byte as a view that never was, and the view is as she deleted it.
func twoWritesOfAView(t *testing.T, w issueWorld) {
	w.changeProject(t, w.web, `{"issue_views_view":true}`)
	web := projectViewsPath(w.web)
	// changes is bob's view named name changed by him to First, then, the
	// second write, by body: the second's answer, as he then reads it.
	changes := func(t *testing.T, name, body string) viewHTTP {
		t.Helper()
		v, _ := w.newView(t, "bob", web, `{"name":"`+name+`"}`)
		var first, second exchange
		w.viewWaits(t, "issue_views", "LockView", nil, writesView("bob", http.MethodPatch, viewPath(v.ID), `{"name":"First"}`),
			writesView("bob", http.MethodPatch, viewPath(v.ID), body), &first, &second)
		w.answered(t, first, http.StatusOK)
		var answered viewHTTP
		decodeAnswer(t, w.answered(t, second, http.StatusOK), &answered)
		if read, _ := w.readView(t, "bob", v.ID); answered.UpdatedAt.Before(first.released) || !sameView(read, answered) {
			t.Errorf("the second change answers %+v, read %+v; want them alike, at a moment no earlier than the first's commit at %s", answered, read,
				first.released)
		}
		return answered
	}
	t.Run("two changes", func(t *testing.T) {
		if v := changes(t, "Twice changed", `{"description":"Second"}`); v.Name != "First" || v.Description != "Second" {
			t.Errorf("after two changes: %+v; want it named First, described Second", v)
		}
	})
	t.Run("a change back", func(t *testing.T) {
		if v := changes(t, "Changed back", `{"name":"Changed back"}`); v.Name != "Changed back" {
			t.Errorf("after a change back: %+v; want it named Changed back", v)
		}
	})
	t.Run("a deletion, then a change", func(t *testing.T) {
		v, _ := w.newView(t, "bob", web, `{"name":"Deleted first"}`)
		from := time.Now().UTC().Truncate(time.Microsecond)
		var deleted, changed exchange
		w.viewWaits(t, "issue_views", "LockView", nil, writesView("alice", http.MethodDelete, viewPath(v.ID), ""),
			writesView("bob", http.MethodPatch, viewPath(v.ID), `{"name":"Late"}`), &deleted, &changed)
		w.answered(t, deleted, http.StatusNoContent)
		w.refusedAsAbsent(t, changed, "bob", viewPath(uuid.NewV7()), `{"name":"Late"}`, "navigation.view_not_found")
		viewDeletedBy(t, w.pool, v.ID, w.ids["bob"], w.ids["alice"], from, deleted.released)
	})
}
