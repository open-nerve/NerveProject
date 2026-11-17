package app_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/access/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeRolesMany answers, of the accounts asked, the roles it holds in the
// workspace asked about, and records every call with the value of ctxKey in
// its context, as fakeRoles does.
type fakeRolesMany struct {
	roles map[uuid.UUID]map[uuid.UUID]shared.Role // by workspace, then account
	err   error
	calls []string
}

func (f *fakeRolesMany) ActiveRoles(ctx context.Context, workspaceID uuid.UUID, accountIDs []uuid.UUID) (map[uuid.UUID]shared.Role, error) {
	f.calls = append(f.calls, workspaceID.String()+" "+asked(accountIDs)+" "+valueOf(ctx))
	if f.err != nil {
		return nil, f.err
	}
	out := map[uuid.UUID]shared.Role{}
	for _, id := range accountIDs {
		if role, ok := f.roles[workspaceID][id]; ok {
			out[id] = role
		}
	}
	return out, nil
}

// fakeProjectsMany answers the facts it holds of the project asked about,
// with the roles of the accounts asked, and records every call as fakeRoles
// does. A project in deleted is not found, though its facts come with the
// answer: a decision must not take them.
type fakeProjectsMany struct {
	facts   map[uuid.UUID]app.ProjectFactsMany
	deleted map[uuid.UUID]bool
	err     error
	calls   []string
}

func (f *fakeProjectsMany) ProjectFactsMany(ctx context.Context, projectID uuid.UUID, accountIDs []uuid.UUID) (app.ProjectFactsMany, bool, error) {
	f.calls = append(f.calls, projectID.String()+" "+asked(accountIDs)+" "+valueOf(ctx))
	if f.err != nil {
		return app.ProjectFactsMany{}, false, f.err
	}
	p, ok := f.facts[projectID]
	roles := map[uuid.UUID]shared.Role{}
	for _, id := range accountIDs {
		if role, member := p.Roles[id]; member {
			roles[id] = role
		}
	}
	p.Roles = roles
	return p, ok && !f.deleted[projectID], nil
}

func valueOf(ctx context.Context) string {
	if value, ok := ctx.Value(ctxKey{}).(string); ok {
		return value
	}
	return "(none)"
}

func asked(ids []uuid.UUID) string {
	s := ""
	for _, id := range ids {
		s += id.String()[:8] + ","
	}
	return s
}

// world is a workspace w1 of seven accounts, each in another state, and four
// projects: private, of w1, which hides from its guests the work items they
// did not create; viewAll, public, of w1, which shows them every one; other,
// of w2; and gone, of w1 but deleted. outsider is no member of w1 but an
// active member of private (a membership a removal would have ended); odd
// is private's member of a role outside the three.
type world struct {
	roles                         *fakeRolesMany
	projects                      *fakeProjectsMany
	private, viewAll, other, gone uuid.UUID
	admin, member, guest          uuid.UUID
	adminGuest, outsider, odd     uuid.UUID
	nobody                        uuid.UUID
}

func newWorld() world {
	w := world{private: uuid.NewV7(), viewAll: uuid.NewV7(), other: uuid.NewV7(), gone: uuid.NewV7(), admin: uuid.NewV7(),
		member: uuid.NewV7(), guest: uuid.NewV7(), adminGuest: uuid.NewV7(), outsider: uuid.NewV7(), odd: uuid.NewV7(), nobody: uuid.NewV7()}
	w.roles = &fakeRolesMany{roles: map[uuid.UUID]map[uuid.UUID]shared.Role{
		w1: {w.admin: shared.RoleAdmin, w.member: shared.RoleMember, w.guest: shared.RoleGuest, w.adminGuest: shared.RoleAdmin,
			w.odd: shared.RoleMember},
		w2: {w.member: shared.RoleMember},
	}}
	members := map[uuid.UUID]shared.Role{w.member: shared.RoleMember, w.guest: shared.RoleGuest, w.adminGuest: shared.RoleGuest,
		w.outsider: shared.RoleMember, w.odd: 10}
	w.projects = &fakeProjectsMany{facts: map[uuid.UUID]app.ProjectFactsMany{
		w.private: {WorkspaceID: w1, Roles: members},
		w.viewAll: {WorkspaceID: w1, Public: true, GuestViewAll: true, Roles: map[uuid.UUID]shared.Role{w.member: shared.RoleMember,
			w.guest: shared.RoleGuest}},
		w.other: {WorkspaceID: w2, Public: true, Roles: members},
		w.gone:  {WorkspaceID: w1, Public: true, GuestViewAll: true, Roles: members},
	}, deleted: map[uuid.UUID]bool{w.gone: true}}
	return w
}

func (w world) accounts() []uuid.UUID {
	return []uuid.UUID{w.admin, w.member, w.guest, w.adminGuest, w.outsider, w.odd, w.nobody}
}

// authorizer is the Authorizer over the same facts, one account at a time:
// gone, deleted, is not found.
func (w world) authorizer() *app.Authorizer {
	roles := &fakeRoles{roles: map[membership]shared.Role{}}
	for ws, byAccount := range w.roles.roles {
		for id, role := range byAccount {
			roles.roles[membership{ws, id}] = role
		}
	}
	projects := &fakeProjects{facts: map[projectKey]app.ProjectFacts{}}
	for p, f := range w.projects.facts {
		for _, id := range w.accounts() {
			if w.projects.deleted[p] {
				continue
			}
			role, member := f.Roles[id]
			projects.facts[projectKey{p, id}] = app.ProjectFacts{WorkspaceID: f.WorkspaceID, Public: f.Public, Member: member, Role: role,
				GuestViewAll: f.GuestViewAll}
		}
	}
	return app.NewAuthorizer(roles, projects)
}

// DecideMany answers issue.read for each account of w1 on each project as
// the rule table does (M7 P1b spec 2.7): a project's member and guest are
// allowed, the guest who sees only his own work items where the project
// hides the others (OwnOnly, by the project role: the workspace's admin who
// is its guest too), and no one else: not the workspace's admin who is no
// member of the project, not its member whose workspace membership is not
// active, not a role outside the three. A project of another workspace and
// a deleted one are seen by no one. Each account asked is answered once.
func TestDecideManyAnswersEachAccount(t *testing.T) {
	w := newWorld()
	d := app.NewDecider(w.roles, w.projects)
	allowed, own := app.Decision{Allowed: true}, app.Decision{Allowed: true, OwnOnly: true}
	refused := app.Decision{}
	tests := []struct {
		name    string
		project uuid.UUID
		want    []app.Decision // by w.accounts()
	}{
		{"private", w.private, []app.Decision{refused, allowed, own, own, refused, refused, refused}},
		{"viewAll", w.viewAll, []app.Decision{refused, allowed, allowed, refused, refused, refused, refused}},
		{"another workspace's", w.other, []app.Decision{refused, refused, refused, refused, refused, refused, refused}},
		{"deleted", w.gone, []app.Decision{refused, refused, refused, refused, refused, refused, refused}},
	}
	for _, tt := range tests {
		got, err := d.DecideMany(context.Background(), "issue.read", w1, tt.project, append(w.accounts(), w.member))
		want := map[uuid.UUID]app.Decision{}
		for i, id := range w.accounts() {
			want[id] = tt.want[i]
		}
		if err != nil || !maps.Equal(got, want) {
			t.Errorf("%s: DecideMany() = %v, %v; want %v", tt.name, got, err, want)
		}
	}
}

// DecideMany is Authorize for each account (M7 P1b spec R7): for rules of
// each level without AllowCreator, on each project, every account's
// decision is allowed exactly when Authorize answers him no error, and
// OwnOnly then as its Grant's; a refusal, ErrNotVisible or Forbidden, is the
// zero Decision.
func TestDecideManyIsAuthorizeForEachAccount(t *testing.T) {
	w := newWorld()
	d, auth := app.NewDecider(w.roles, w.projects), w.authorizer()
	var answers []app.Decision
	for _, action := range []shared.Action{"issue.read", "issue.update", "project.read", "workspace.update"} {
		for _, p := range []uuid.UUID{w.private, w.viewAll, w.other, w.gone} {
			got, err := d.DecideMany(context.Background(), action, w1, p, w.accounts())
			if err != nil {
				t.Fatalf("%s on %s: DecideMany() = %v", action, p, err)
			}
			for _, id := range w.accounts() {
				grant, err := auth.Authorize(context.Background(), shared.Actor{UserID: id}, action, shared.Target{WorkspaceID: w1, ProjectID: p})
				if err != nil && !errors.Is(err, shared.ErrNotVisible) && !errors.Is(err, shared.Forbidden()) {
					t.Fatalf("Authorize() = %v", err)
				}
				want := app.Decision{Allowed: err == nil, OwnOnly: err == nil && grant.OwnOnly}
				if got[id] != want {
					t.Errorf("%s on %s for %s: DecideMany() = %+v, Authorize() = %+v, %v", action, p, id, got[id], grant, err)
				}
				answers = append(answers, got[id])
			}
		}
	}
	for _, d := range []app.Decision{{}, {Allowed: true}, {Allowed: true, OwnOnly: true}} {
		if !slices.Contains(answers, d) {
			t.Errorf("no account was answered %+v; the comparison needs each answer", d)
		}
	}
}

// DecideMany reads each port once, in the caller's context, for the
// accounts asked; each port's failure is its own. A rule that allows the
// creator is refused before anything is read, as an action without a rule
// is: DecideMany knows no creator (M7 P1b spec 2.7).
func TestDecideManyReadsOnceAndRefusesTheCreatorsRules(t *testing.T) {
	w := newWorld()
	d := app.NewDecider(w.roles, w.projects)
	ctx := context.WithValue(context.Background(), ctxKey{}, "tx")
	accounts := []uuid.UUID{w.member, w.guest}
	if _, err := d.DecideMany(ctx, "issue.read", w1, w.private, accounts); err != nil {
		t.Fatal(err)
	}
	if want := []string{w1.String() + " " + asked(accounts) + " tx"}; !slices.Equal(w.roles.calls, want) {
		t.Errorf("ActiveRoles calls = %q, want %q", w.roles.calls, want)
	}
	if want := []string{w.private.String() + " " + asked(accounts) + " tx"}; !slices.Equal(w.projects.calls, want) {
		t.Errorf("ProjectFactsMany calls = %q, want %q", w.projects.calls, want)
	}
	for _, action := range []shared.Action{"issue.delete", "cycle.delete", "no.such.action"} {
		w.roles.calls, w.projects.calls = nil, nil
		got, err := d.DecideMany(ctx, action, w1, w.private, accounts)
		var se *shared.Error
		if err == nil || errors.As(err, &se) || got != nil {
			t.Errorf("%s: DecideMany() = %v, %v; want an internal error", action, got, err)
		}
		if len(w.roles.calls)+len(w.projects.calls) != 0 {
			t.Errorf("%s: ActiveRoles calls %q, ProjectFactsMany calls %q; want none", action, w.roles.calls, w.projects.calls)
		}
	}
	failure := errors.New("connection reset")
	w.roles.err = failure
	if got, err := d.DecideMany(ctx, "issue.read", w1, w.private, accounts); !errors.Is(err, failure) || got != nil {
		t.Errorf("with the roles failing: %v, %v; want %v", got, err, failure)
	}
	w.roles.err, w.projects.err = nil, failure
	if got, err := d.DecideMany(ctx, "issue.read", w1, w.private, accounts); !errors.Is(err, failure) || got != nil {
		t.Errorf("with the project failing: %v, %v; want %v", got, err, failure)
	}
}
