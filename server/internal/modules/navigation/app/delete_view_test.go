package app_test

import (
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The grants a decision gives on a view's change and deletion (shared.Grant, access decide.go): the project's admin;
// the workspace's admin who is the project's member, or its guest, the last seeing only his own while the project
// does not show its guests everything; the project's guest, likewise; and the workspace's admin, member and guest on
// a workspace's view.
var (
	projectAdmin    = shared.Grant{WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleAdmin, ProjectAdmin: true}
	adminAsMember   = shared.Grant{WorkspaceRole: shared.RoleAdmin, ProjectRole: shared.RoleMember, ProjectAdmin: true}
	adminAsGuest    = shared.Grant{WorkspaceRole: shared.RoleAdmin, ProjectRole: shared.RoleGuest, ProjectAdmin: true, OwnOnly: true}
	adminAsGuestAll = shared.Grant{WorkspaceRole: shared.RoleAdmin, ProjectRole: shared.RoleGuest, ProjectAdmin: true}
	guestOwnOnly    = shared.Grant{WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleGuest, OwnOnly: true}
	guestSeesAll    = shared.Grant{WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleGuest}
	workspaceAdmin  = shared.Grant{WorkspaceRole: shared.RoleAdmin}
	workspaceMember = shared.Grant{WorkspaceRole: shared.RoleMember}
	workspaceGuest  = shared.Grant{WorkspaceRole: shared.RoleGuest}
)

// notFound is the one 404 of /views/{view_id}.
var notFound = wantErr(domain.ErrViewNotFound)

// remove runs alice's deletion of the view id in w.
func remove(t *testing.T, w *world, id uuid.UUID) error {
	project, workspace := noWords(w)
	return app.NewDeleteView(w.writes(project, workspace)).Execute(as(t), id)
}

// change is a write of a view by its id as alice, an update or a deletion, and who wrote what it wrote.
type change struct {
	name    string
	run     func(*testing.T, *world, uuid.UUID) error
	writers func(*world) []uuid.UUID
}

var changes = []change{
	{"update", func(t *testing.T, w *world, id uuid.UUID) error {
		project, workspace := noWords(w)
		_, err := update(t, w, project, workspace, id, rename("Changed"))
		return err
	}, func(w *world) []uuid.UUID {
		var by []uuid.UUID
		for _, u := range w.updated {
			by = append(by, u.by)
		}
		return by
	}},
	{"delete", remove, func(w *world) []uuid.UUID {
		var by []uuid.UUID
		for _, d := range w.deleted {
			by = append(by, d.by)
		}
		return by
	}},
}

// wantDone fails t unless c is done, the one view written by alice, the caller, whoever owns it.
func (c change) wantDone(t *testing.T, w *world, err error) {
	t.Helper()
	if by := c.writers(w); err != nil || !reflect.DeepEqual(by, []uuid.UUID{alice}) {
		t.Errorf("%s: %v, written by %v; want it done by alice", c.name, err, by)
	}
}

// decides is a world whose decisions answer g and err, before the locks and under them.
func decides(g shared.Grant, err error) func(*world) {
	return func(w *world) {
		w.decide = func(shared.Action, shared.Target, bool) (shared.Grant, error) { return g, err }
	}
}

// A deletion reads the view and decides as an update does, takes the same locks in the same order and decides again
// under them; then soft-deletes the view at the moment read after its last lock, by the caller (P3b spec 2.5, 2.6).
func TestDeletingAView(t *testing.T) {
	for _, tt := range []struct {
		kind  string
		view  func(*world) domain.View
		calls []string
	}{
		{"project", webView, []string{"view webView", "authorize alice view.delete acme/web",
			"begin write", "share workspace acme", "share project web", "lock view webView", "authorize alice view.delete acme/web",
			"now", "delete webView by alice", "end write"}},
		{"workspace", acmeView, []string{"view acmeView", "authorize alice workspace_view.delete acme/-",
			"begin write", "share workspace acme", "lock view acmeView", "authorize alice workspace_view.delete acme/-",
			"now", "delete acmeView by alice", "end write"}},
	} {
		t.Run(tt.kind, func(t *testing.T) {
			w := newWorld()
			v := own(w, tt.view(w))
			if err := remove(t, w, v.ID); err != nil {
				t.Fatal(err)
			}
			wantCalls(t, w, tt.calls...)
			if !reflect.DeepEqual(w.deleted, []deletion{{id: v.ID, by: alice, at: moment}}) {
				t.Errorf("deleted %+v; want the view by alice at the moment", w.deleted)
			}
		})
	}
}

// Who changes a view and who deletes one (spec 2.2, D29, DP-15, re-90 m11): only its owner changes it, whoever
// administers it; its owner or its admin deletes it, Grant.ProjectAdmin for a project's view, the workspace's admin
// for a workspace's. Another's view is not there for a grant that sees only his own (Grant.OwnOnly), the 404 before
// any 403, also when the rule refuses his role: the project's guest's, and the workspace's admin's who is the
// project's guest, whom the rule lets through and only OwnOnly hides; once the project shows its guests everything,
// he deletes another's view, as its admin, and changes none. A role the rule refuses is 403, his own view too.
func TestWhoChangesAndDeletesAView(t *testing.T) {
	refused := shared.Forbidden()
	for _, tt := range []struct {
		name           string
		project, mine  bool
		grant          shared.Grant
		rule           error
		update, delete func(*testing.T, error) // nil: done
	}{
		{"a member, another's", true, false, member, nil, want403, want403},
		{"the project's admin, another's", true, false, projectAdmin, nil, want403, nil},
		{"the workspace's admin as the project's member, another's", true, false, adminAsMember, nil, want403, nil},
		{"a guest seeing all, another's", true, false, guestSeesAll, refused, want403, want403},
		{"a guest seeing his own, another's", true, false, guestOwnOnly, refused, notFound, notFound},
		{"the workspace's admin as the project's guest, another's", true, false, adminAsGuest, nil, notFound, notFound},
		{"the workspace's admin as the project's guest seeing all, another's", true, false, adminAsGuestAll, nil, want403, nil},
		{"a member, his own", true, true, member, nil, nil, nil},
		{"a guest seeing his own, his own", true, true, guestOwnOnly, refused, want403, want403},
		{"the workspace's admin as the project's guest, his own", true, true, adminAsGuest, nil, nil, nil},
		{"the workspace's member, another's", false, false, workspaceMember, nil, want403, want403},
		{"the workspace's admin, another's", false, false, workspaceAdmin, nil, want403, nil},
		{"the workspace's guest, another's", false, false, workspaceGuest, refused, want403, want403},
		{"the workspace's member, his own", false, true, workspaceMember, nil, nil, nil},
	} {
		for _, c := range changes {
			t.Run(c.name+"/"+tt.name, func(t *testing.T) {
				w := newWorld()
				v := acmeView(w)
				if tt.project {
					v = webView(w)
				}
				if tt.mine {
					v = own(w, v)
				}
				decides(tt.grant, tt.rule)(w)
				err := c.run(t, w, v.ID)
				want := tt.update
				if c.name == "delete" {
					want = tt.delete
				}
				if want == nil {
					c.wantDone(t, w, err)
					return
				}
				want(t, err)
				wroteNothing(t, w)
			})
		}
	}
}

// The refusals of a change and of a deletion of the caller's own view, beyond who he is, in their order (spec 2.5):
// a view that is not there, whose project or workspace he does not see, is 404 navigation.view_not_found; a role
// refused is 403; then, under the locks, what they read again and decide again, which is final: a view, its project
// or its workspace deleted while a lock waited is the same 404, and so is one he no longer sees; a role refused now is
// 403; and last a project's views turned off, 409, also when they were turned on before the locks, after every 404
// and 403, those under the locks too. A refused write writes nothing and reads no clock.
func TestTheRefusalsOfAChange(t *testing.T) {
	refuseUnderLock := func(err error) func(*world) {
		return func(w *world) {
			w.decide = func(_ shared.Action, t shared.Target, inTx bool) (shared.Grant, error) {
				if inTx {
					return shared.Grant{}, err
				}
				if t.ProjectID == (uuid.UUID{}) {
					return workspaceMember, nil
				}
				return member, nil
			}
		}
	}
	off := app.ProjectFacts{WorkspaceID: acme}
	viewsOff := func(w *world) { w.projects[web] = off }
	offUnderLock := func(w *world) { w.underLock = map[uuid.UUID]*app.ProjectFacts{web: &off} }
	both := func(a, b func(*world)) func(*world) { return func(w *world) { a(w); b(w) } }
	for _, tt := range []struct {
		name    string
		project bool // the view is the project's, else the workspace's
		world   func(*world, domain.View)
		want    func(*testing.T, error) // nil: done
	}{
		{"no such view", true, func(w *world, v domain.View) { delete(w.views, v.ID) }, notFound},
		{"its project not seen", true, on(decides(shared.Grant{}, shared.ErrNotVisible)), notFound},
		{"its workspace not seen", false, on(decides(shared.Grant{}, shared.ErrNotVisible)), notFound},
		{"a project's role refused", true, on(decides(member, shared.Forbidden())), want403},
		{"a workspace's role refused", false, on(decides(workspaceGuest, shared.Forbidden())), want403},
		{"views off", true, on(viewsOff), wantErr(domain.ErrViewsDisabled)},
		{"views off × a role refused", true, on(both(viewsOff, decides(member, shared.Forbidden()))), want403},
		{"views off × its project not seen", true, on(both(viewsOff, decides(shared.Grant{}, shared.ErrNotVisible))), notFound},
		{"views off × another's", true, func(w *world, v domain.View) { viewsOff(w); v.OwnedByID = bob; w.views[v.ID] = v }, want403},
		{"views turned on while the lock waited", true, on(both(viewsOff, func(w *world) {
			w.underLock = map[uuid.UUID]*app.ProjectFacts{web: {WorkspaceID: acme, IssueViewsView: true}}
		})), nil},
		{"the view deleted while the lock waited", true, func(w *world, v domain.View) { w.viewsLocked = map[uuid.UUID]*domain.View{v.ID: nil} }, notFound},
		{"a workspace's view deleted while the lock waited", false, func(w *world, v domain.View) { w.viewsLocked = map[uuid.UUID]*domain.View{v.ID: nil} },
			notFound},
		{"its project deleted while the lock waited", true, on(func(w *world) { w.underLock = map[uuid.UUID]*app.ProjectFacts{web: nil} }), notFound},
		{"its workspace deleted while the lock waited", true, on(func(w *world) { w.workspaceGone = map[uuid.UUID]bool{acme: true} }), notFound},
		{"a workspace's view's workspace deleted while the lock waited", false, on(func(w *world) { w.workspaceGone = map[uuid.UUID]bool{acme: true} }),
			notFound},
		{"views turned off while the lock waited", true, on(offUnderLock), wantErr(domain.ErrViewsDisabled)},
		{"a project's role refused under the lock", true, on(refuseUnderLock(shared.Forbidden())), want403},
		{"a workspace's role refused under the lock", false, on(refuseUnderLock(shared.Forbidden())), want403},
		{"its project not seen under the lock", true, on(refuseUnderLock(shared.ErrNotVisible)), notFound},
		{"its workspace not seen under the lock", false, on(refuseUnderLock(shared.ErrNotVisible)), notFound},
		{"views turned off × a role refused under the lock", true, on(both(offUnderLock, refuseUnderLock(shared.Forbidden()))), want403},
		{"views turned off × not seen under the lock", true, on(both(offUnderLock, refuseUnderLock(shared.ErrNotVisible))), notFound},
	} {
		for _, c := range changes {
			t.Run(c.name+"/"+tt.name, func(t *testing.T) {
				w := newWorld()
				v := acmeView(w)
				if tt.project {
					v = webView(w)
				}
				v = own(w, v)
				if !tt.project {
					decides(workspaceMember, nil)(w)
				}
				tt.world(w, v)
				err := c.run(t, w, v.ID)
				if tt.want == nil {
					c.wantDone(t, w, err)
					return
				}
				tt.want(t, err)
				wroteNothing(t, w)
			})
		}
	}
}

// on is a change of the world alone, whatever the view.
func on(f func(*world)) func(*world, domain.View) {
	return func(w *world, _ domain.View) { f(w) }
}

// The owner rules are decided again under the locks, and that decision is final (spec 2.5): an admin who deletes
// another's view and was demoted while the lock waited is refused 403; the workspace's admin who is the project's
// guest, once the project no longer shows its guests everything, no longer sees another's view, 404.
func TestTheOwnerRulesUnderTheLock(t *testing.T) {
	for _, tt := range []struct {
		name          string
		project       bool
		before, after shared.Grant
		want          func(*testing.T, error)
	}{
		{"the project's admin demoted", true, projectAdmin, member, want403},
		{"the workspace's admin as the project's guest, no longer seeing all", true, adminAsGuestAll, adminAsGuest, notFound},
		{"the workspace's admin demoted", false, workspaceAdmin, workspaceMember, want403},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newWorld()
			v := acmeView(w)
			if tt.project {
				v = webView(w)
			}
			w.decide = func(_ shared.Action, _ shared.Target, inTx bool) (shared.Grant, error) {
				if inTx {
					return tt.after, nil
				}
				return tt.before, nil
			}
			tt.want(t, remove(t, w, v.ID))
			wroteNothing(t, w)
		})
	}
}
