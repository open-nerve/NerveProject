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

// fakeMemberProjects answers, of the world's projects, those of the
// workspace asked about that are not deleted and that the account asked
// about is a member of, in id order, as the real read does; and records
// every call as fakeRoles does.
type fakeMemberProjects struct {
	w     world
	err   error
	calls []string
}

func (f *fakeMemberProjects) MemberProjectFacts(ctx context.Context, workspaceID, userID uuid.UUID) ([]app.MemberProjectFacts, error) {
	f.calls = append(f.calls, workspaceID.String()+" "+userID.String()+" "+valueOf(ctx))
	if f.err != nil {
		return nil, f.err
	}
	var out []app.MemberProjectFacts
	for _, id := range slices.SortedFunc(maps.Keys(f.w.projects.facts), uuid.UUID.Compare) {
		p := f.w.projects.facts[id]
		if role, member := p.Roles[userID]; member && p.WorkspaceID == workspaceID && !f.w.projects.deleted[id] {
			out = append(out, app.MemberProjectFacts{ProjectID: id, Public: p.Public, Role: role, GuestViewAll: p.GuestViewAll})
		}
	}
	return out, nil
}

// readable is the ReadableProjects over w's facts, one account's at a time:
// his role as w holds it, and the projects he is a member of.
func (w world) readable() (*app.ReadableProjects, *fakeRoles, *fakeMemberProjects) {
	roles := &fakeRoles{roles: map[membership]shared.Role{}}
	for ws, byAccount := range w.roles.roles {
		for id, role := range byAccount {
			roles.roles[membership{ws, id}] = role
		}
	}
	projects := &fakeMemberProjects{w: w}
	return app.NewReadableProjects(roles, projects), roles, projects
}

// Readable answers issue.read on each project the account is a member of
// as the rule table does (M7 P1c spec 2.5): a project's member and guest
// read it, the guest only his own work items where the project hides the
// others (OwnOnly, by the project role: the workspace's admin who is its
// guest too); no project of the workspace's admin who is a member of none,
// none of a project member whose workspace membership is not active, none
// where his project role is outside the three; a deleted project is no
// one's.
func TestReadableAnswersEachAccount(t *testing.T) {
	w := newWorld()
	r, _, _ := w.readable()
	tests := []struct {
		name      string
		workspace uuid.UUID
		account   uuid.UUID
		want      []app.Grant // in any order
	}{
		{name: "the workspace's admin, a member of no project", workspace: w1, account: w.admin},
		{name: "a member", workspace: w1, account: w.member, want: []app.Grant{{ProjectID: w.private}, {ProjectID: w.viewAll}}},
		{name: "a guest", workspace: w1, account: w.guest, want: []app.Grant{{ProjectID: w.private, OwnOnly: true}, {ProjectID: w.viewAll}}},
		{name: "the workspace's admin, a project's guest", workspace: w1, account: w.adminGuest,
			want: []app.Grant{{ProjectID: w.private, OwnOnly: true}}},
		{name: "a project's member, no member of the workspace", workspace: w1, account: w.outsider},
		{name: "a role outside the three", workspace: w1, account: w.odd},
		{name: "no one", workspace: w1, account: w.nobody},
		{name: "another workspace's member", workspace: w2, account: w.member, want: []app.Grant{{ProjectID: w.other}}},
		{name: "another workspace's project's guest, no member of it", workspace: w2, account: w.guest},
	}
	for _, tt := range tests {
		got, err := r.Readable(context.Background(), shared.Actor{UserID: tt.account}, tt.workspace)
		want := slices.SortedFunc(slices.Values(tt.want), func(a, b app.Grant) int { return a.ProjectID.Compare(b.ProjectID) })
		if err != nil || !slices.Equal(got, want) {
			t.Errorf("%s: Readable() = %+v, %v; want %+v", tt.name, got, err, want)
		}
	}
}

// Readable is Authorize's issue.read on each project (M7 P1c spec 2.5,
// R1): for each account, in each workspace, the projects it answers, in id
// order, are exactly those of the world on which Authorize allows him
// issue.read with the Target of that workspace and project, another
// workspace's and the deleted one among them, each OwnOnly as its Grant
// is. The comparison meets each answer: allowed, own-only and refused.
func TestReadableIsAuthorizeForEachAccount(t *testing.T) {
	w := newWorld()
	r, _, _ := w.readable()
	auth := w.authorizer()
	projects := slices.SortedFunc(maps.Keys(w.projects.facts), uuid.UUID.Compare)
	answers := map[string]int{}
	for _, ws := range []uuid.UUID{w1, w2} {
		for _, id := range w.accounts() {
			actor := shared.Actor{UserID: id}
			got, err := r.Readable(context.Background(), actor, ws)
			if err != nil {
				t.Fatalf("Readable(%s, %s) = %v", id, ws, err)
			}
			var want []app.Grant
			for _, p := range projects {
				grant, err := auth.Authorize(context.Background(), actor, "issue.read", shared.Target{WorkspaceID: ws, ProjectID: p})
				switch {
				case err == nil && grant.OwnOnly:
					answers["own-only"]++
				case err == nil:
					answers["allowed"]++
				case errors.Is(err, shared.ErrNotVisible) || errors.Is(err, shared.Forbidden()):
					answers["refused"]++
					continue
				default:
					t.Fatalf("Authorize() = %v", err)
				}
				want = append(want, app.Grant{ProjectID: p, OwnOnly: grant.OwnOnly})
			}
			if !slices.Equal(got, want) {
				t.Errorf("%s in %s: Readable() = %+v, Authorize allows %+v", id, ws, got, want)
			}
		}
	}
	for _, a := range []string{"allowed", "own-only", "refused"} {
		if answers[a] == 0 {
			t.Errorf("no project was answered %s (%v); the comparison needs each answer", a, answers)
		}
	}
}

// Readable reads each port once, in the caller's context, for the caller
// and the workspace asked about; each port's failure is its own, and
// answers nothing.
func TestReadableReadsOnceInTheCallersContext(t *testing.T) {
	w := newWorld()
	r, roles, projects := w.readable()
	ctx := context.WithValue(context.Background(), ctxKey{}, "tx")
	if _, err := r.Readable(ctx, shared.Actor{UserID: w.guest}, w1); err != nil {
		t.Fatal(err)
	}
	want := []string{w1.String() + " " + w.guest.String() + " tx"}
	if !slices.Equal(roles.calls, want) || !slices.Equal(projects.calls, want) {
		t.Errorf("ActiveRole calls = %q, MemberProjectFacts calls = %q; want %q each", roles.calls, projects.calls, want)
	}
	failure := errors.New("connection reset")
	roles.err = failure
	if got, err := r.Readable(ctx, shared.Actor{UserID: w.guest}, w1); !errors.Is(err, failure) || got != nil {
		t.Errorf("with the roles failing: %v, %v; want %v", got, err, failure)
	}
	roles.err, projects.err = nil, failure
	if got, err := r.Readable(ctx, shared.Actor{UserID: w.guest}, w1); !errors.Is(err, failure) || got != nil {
		t.Errorf("with the projects failing: %v, %v; want %v", got, err, failure)
	}
}
