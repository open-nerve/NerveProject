package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The accounts and workspaces of createProject's tests: alice is acme's
// member, bob its admin, carol its guest, dave not an active member; erin
// is beta's admin.
var (
	alice, bob, carol, dave, erin = uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	acme                          = app.Workspace{ID: uuid.NewV7(), Timezone: "Asia/Shanghai"}
	beta                          = app.Workspace{ID: uuid.NewV7(), Timezone: "UTC"}
)

// createFixture is CreateProject over fakes sharing one log.
type createFixture struct {
	log        *callLog
	tx         *fakeTx
	workspaces *fakeDirectory
	members    *fakeMembers
	projects   *fakeProjects
	auth       *fakeAuthorizer
}

func newCreate() (*app.CreateProject, *createFixture) {
	log := &callLog{}
	f := &createFixture{log: log, tx: &fakeTx{log: log},
		workspaces: &fakeDirectory{log: log, workspaces: map[string]app.Workspace{"acme": acme, "beta": beta}},
		members: &fakeMembers{log: log, roles: map[uuid.UUID]map[uuid.UUID]shared.Role{
			acme.ID: {alice: shared.RoleMember, bob: shared.RoleAdmin, carol: shared.RoleGuest},
			beta.ID: {erin: shared.RoleAdmin, dave: shared.RoleMember},
		}},
		projects: &fakeProjects{log: log, lowest: map[uuid.UUID]*float64{bob: ptr(100.0)}},
		auth: &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{
			{alice, acme.ID}: {WorkspaceRole: shared.RoleMember}, {bob, acme.ID}: {WorkspaceRole: shared.RoleAdmin},
			{erin, beta.ID}: {WorkspaceRole: shared.RoleAdmin},
		}, errs: map[grantKey]error{{carol, acme.ID}: shared.Forbidden()}},
	}
	return app.NewCreateProject(app.CreateProjectDeps{Workspaces: f.workspaces, Members: f.members, Projects: f.projects, Auth: f.auth, Tx: f.tx,
		Clock: clockAt{clockNow, log}}), f
}

func ptr[T any](v T) *T { return &v }

// created are the calls of a creation of the project id in w by user, lead
// the other admin or none: its locks and decision, then the inserts.
func created(id uuid.UUID, w app.Workspace, user uuid.UUID, lead *uuid.UUID, row string, sortOrders map[uuid.UUID]string) []string {
	at := clockNow.Format(timeFormat)
	admins := []uuid.UUID{user}
	if lead != nil && *lead != user {
		admins = append(admins, *lead)
	}
	calls := []string{"Now", "Begin", "ShareWorkspaceBySlug " + slugOf(w),
		fmt.Sprintf("Authorize %s project.create on %s/%s", user, w.ID, uuid.UUID{}), fmt.Sprintf("ShareMembers %s %v", w.ID, admins),
		"CreateProject " + id.String() + " in " + w.ID.String() + " " + row + " by " + user.String() + " at " + at}
	for _, a := range admins {
		calls = append(calls, fmt.Sprintf("CreateMember %s in %s/%s as 20 by %s at %s", a, w.ID, id, user, at),
			fmt.Sprintf("LowestSortOrder %s %s", w.ID, a),
			fmt.Sprintf("CreatePreferences %s in %s/%s at %s by %s at %s", a, w.ID, id, sortOrders[a], user, at))
	}
	var states []string
	for _, s := range []string{"Backlog", "Todo", "In Progress", "Done", "Cancelled", "Triage"} {
		states = append(states, fmt.Sprintf("%s/%s %s by %s at %s", w.ID, id, s, user, at))
	}
	return append(calls, fmt.Sprintf("CreateStates %q", states), fmt.Sprintf("GetProject %s for %s", id, user))
}

func slugOf(w app.Workspace) string {
	if w == acme {
		return "acme"
	}
	return "beta"
}

// createdID is the id of the project the fixture's store was given.
func (f *createFixture) createdID(t *testing.T) uuid.UUID {
	t.Helper()
	if f.projects.project == nil {
		t.Fatal("no project was inserted")
	}
	return f.projects.project.ID
}

// CreateProject checks the request, then, in one transaction and in the
// order of M3 design 3.6, locks the workspace, decides, locks the creator's
// and the lead's memberships, and inserts the project, each of them its
// admin with his display settings first in his sidebar, and the six
// default states, by the caller at the clock's time read once before the
// transaction; the answer is the project as stored, read back as the
// caller sees it. The project takes the workspace's time zone unless the
// request gives one.
func TestCreateProject(t *testing.T) {
	tests := []struct {
		name       string
		user       uuid.UUID
		slug       string
		in         domain.NewProject
		w          app.Workspace
		lead       *uuid.UUID
		row        string
		sortOrders map[uuid.UUID]string
	}{
		{"a member, the admin as the lead", alice, "acme",
			domain.NewProject{Name: "Web", Identifier: "web", Description: "The app", LeadID: &bob,
				LogoProps: domain.LogoProps{InUse: ptr("emoji"), Emoji: &domain.Emoji{Value: ptr("128640")}}}, acme, &bob,
			`"Web" "WEB" "The app" network 2 lead ` + bob.String() + ` logo {"in_use":"emoji","emoji":{"value":"128640"}} timezone Asia/Shanghai`,
			map[uuid.UUID]string{alice: "65535", bob: "-9900"}},
		{"the admin, himself the lead, private, in his time zone", bob, "acme",
			domain.NewProject{Name: "Ops", Identifier: "OPS", LeadID: &bob, Network: ptr(domain.NetworkPrivate), Timezone: ptr("Europe/Paris")},
			acme, &bob, `"Ops" "OPS" "" network 0 lead ` + bob.String() + ` logo {} timezone Europe/Paris`, map[uuid.UUID]string{bob: "-9900"}},
		{"another workspace's admin, no lead", erin, "beta", domain.NewProject{Name: "Web", Identifier: "WEB"}, beta, nil,
			`"Web" "WEB" "" network 2 lead <nil> logo {} timezone UTC`, map[uuid.UUID]string{erin: "65535"}},
		{"a member of another workspace as the lead", erin, "beta", domain.NewProject{Name: "Web", Identifier: "WEB", LeadID: &dave}, beta, &dave,
			`"Web" "WEB" "" network 2 lead ` + dave.String() + ` logo {} timezone UTC`, map[uuid.UUID]string{erin: "65535", dave: "65535"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newCreate()

			got, err := uc.Execute(as(tt.user), tt.slug, tt.in)

			if err != nil {
				t.Fatalf("Execute() = %v", err)
			}
			id := f.createdID(t)
			if want := created(id, tt.w, tt.user, tt.lead, tt.row, tt.sortOrders); !slices.Equal(f.log.calls, want) {
				t.Errorf("calls\n%q\nwant\n%q", f.log.calls, want)
			}
			if got.ID != id || !got.CreatedAt.Equal(now) || got.CreatedAt != now || got.MemberRole == nil || *got.MemberRole != shared.RoleAdmin {
				t.Errorf("Execute() = %+v; want the project %s as stored at %v, the caller its admin", got, id, now)
			}
		})
	}
}

// The six default states are stored as the domain gives them, each its own
// row of the project.
func TestCreateProjectStoresTheDefaultStates(t *testing.T) {
	uc, f := newCreate()
	if _, err := uc.Execute(as(alice), "acme", domain.NewProject{Name: "Web", Identifier: "WEB"}); err != nil {
		t.Fatal(err)
	}
	var got []domain.NewState
	ids := map[uuid.UUID]bool{}
	for _, r := range f.projects.states {
		got = append(got, r.State)
		ids[r.ID] = true
	}
	if !slices.Equal(got, domain.DefaultStates()) || len(ids) != 6 {
		t.Errorf("the states %+v, %d ids; want the six defaults, each its own id", got, len(ids))
	}
}

// Refusals, each in its place, and nothing written:
//   - the request's values, before the clock or the transaction;
//   - a workspace not there, or not visible: 404 workspace.not_found,
//     before the members' lock;
//   - a guest: the Authorizer's 403, before the members' lock;
//   - a lead who is not an active admin or member of the workspace: 422,
//     after the decision and the members' lock.
func TestCreateProjectRefuses(t *testing.T) {
	leadIs := func(lead uuid.UUID) domain.NewProject {
		return domain.NewProject{Name: "Web", Identifier: "WEB", LeadID: &lead}
	}
	decided := func(user uuid.UUID, w app.Workspace, slug string) []string {
		return []string{"Now", "Begin", "ShareWorkspaceBySlug " + slug, fmt.Sprintf("Authorize %s project.create on %s/%s", user, w.ID, uuid.UUID{})}
	}
	tests := []struct {
		name  string
		user  uuid.UUID
		slug  string
		in    domain.NewProject
		want  error
		calls []string
	}{
		{"an invalid identifier", alice, "acme", domain.NewProject{Name: "Web", Identifier: "WEB-2"},
			shared.Invalid(shared.FieldError{Field: "identifier", Code: "invalid_format"}), nil},
		{"no workspace", alice, "gone", domain.NewProject{Name: "Web", Identifier: "WEB"}, domain.ErrWorkspaceNotFound,
			[]string{"Now", "Begin", "ShareWorkspaceBySlug gone"}},
		{"a workspace he is not in", dave, "acme", domain.NewProject{Name: "Web", Identifier: "WEB"}, domain.ErrWorkspaceNotFound,
			decided(dave, acme, "acme")},
		{"a guest", carol, "acme", domain.NewProject{Name: "Web", Identifier: "WEB"}, shared.Forbidden(), decided(carol, acme, "acme")},
		{"a guest as the lead", alice, "acme", leadIs(carol), domain.LeadNotAllowed(),
			append(decided(alice, acme, "acme"), fmt.Sprintf("ShareMembers %s %v", acme.ID, []uuid.UUID{alice, carol}))},
		{"no active member as the lead", alice, "acme", leadIs(dave), domain.LeadNotAllowed(),
			append(decided(alice, acme, "acme"), fmt.Sprintf("ShareMembers %s %v", acme.ID, []uuid.UUID{alice, dave}))},
		{"another workspace's admin as the lead", alice, "acme", leadIs(erin), domain.LeadNotAllowed(),
			append(decided(alice, acme, "acme"), fmt.Sprintf("ShareMembers %s %v", acme.ID, []uuid.UUID{alice, erin}))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newCreate()

			got, err := uc.Execute(as(tt.user), tt.slug, tt.in)

			if !sameError(err, tt.want) || got.ID != (uuid.UUID{}) {
				t.Errorf("Execute() = %+v, %v; want %v", got, err, tt.want)
			}
			if !slices.Equal(f.log.calls, tt.calls) || f.projects.project != nil {
				t.Errorf("calls %q, a project inserted %v; want %q and none", f.log.calls, f.projects.project != nil, tt.calls)
			}
		})
	}
}

// sameError reports whether err is want: the same code, and for a 422 the
// same fields, their messages left out.
func sameError(err, want error) bool {
	var e, w *shared.Error
	if !errors.As(err, &e) || !errors.As(want, &w) || e.Kind != w.Kind || e.Code != w.Code || len(e.Fields) != len(w.Fields) {
		return false
	}
	for i := range e.Fields {
		if e.Fields[i].Field != w.Fields[i].Field || e.Fields[i].Code != w.Fields[i].Code {
			return false
		}
	}
	return true
}

// Every port's failure comes back as itself, the transaction's commit's
// too, and nothing is answered.
func TestCreateProjectReturnsEachFailure(t *testing.T) {
	failure := errors.New("disk full")
	type failing struct {
		name string
		fail func(f *createFixture)
	}
	tests := []failing{
		{"the workspace's lock", func(f *createFixture) { f.workspaces.err = failure }},
		{"the decision", func(f *createFixture) { f.auth.errs = map[grantKey]error{{alice, acme.ID}: failure} }},
		{"the members' lock", func(f *createFixture) { f.members.err = failure }},
		{"the commit", func(f *createFixture) { f.tx.commitErr = failure }},
	}
	for _, method := range []string{"CreateProject", "CreateMember", "LowestSortOrder", "CreatePreferences", "CreateStates", "GetProject"} {
		tests = append(tests, failing{method, func(f *createFixture) { f.projects.errs = map[string]error{method: failure} }})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newCreate()
			tt.fail(f)
			got, err := uc.Execute(as(alice), "acme", domain.NewProject{Name: "Web", Identifier: "WEB", LeadID: &bob})
			if !errors.Is(err, failure) || got.ID != (uuid.UUID{}) {
				t.Errorf("Execute() = %+v, %v; want %v", got, err, failure)
			}
		})
	}
	// A project the store cannot read back is an internal error, not an
	// answer.
	uc, f := newCreate()
	f.projects.missing = true
	var e *shared.Error
	if got, err := uc.Execute(as(alice), "acme", domain.NewProject{Name: "Web", Identifier: "WEB"}); err == nil || errors.As(err, &e) ||
		got.ID != (uuid.UUID{}) {
		t.Errorf("Execute() with the project gone = %+v, %v; want an internal error", got, err)
	}
}

// The store's conflicts come back as themselves: 409 of the identifier or
// the name.
func TestCreateProjectAnswersTheStoresConflicts(t *testing.T) {
	for _, conflict := range []error{domain.ErrIdentifierTaken, domain.ErrNameTaken} {
		uc, f := newCreate()
		f.projects.errs = map[string]error{"CreateProject": conflict}
		if _, err := uc.Execute(as(alice), "acme", domain.NewProject{Name: "Web", Identifier: "WEB"}); !errors.Is(err, conflict) {
			t.Errorf("Execute() = %v, want %v", err, conflict)
		}
	}
}

// Without a caller nothing is read: 401, from the use case on its own.
func TestCreateProjectWithoutACaller(t *testing.T) {
	uc, f := newCreate()
	if _, err := uc.Execute(context.Background(), "acme", domain.NewProject{Name: "Web", Identifier: "WEB"}); !errors.Is(err, shared.Unauthenticated()) {
		t.Errorf("Execute() without an actor = %v, want 401 unauthorized", err)
	}
	if len(f.log.calls) != 0 {
		t.Errorf("calls = %q, want none", f.log.calls)
	}
}
