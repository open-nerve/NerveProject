package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeStore's reads and writes of one project membership: it logs each
// call, finds a membership among its projects' by id, and writes into it.

// memberSince is when the fakes' memberships were made, as stored: no
// clock's time.
var memberSince = time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)

// membership finds the membership id among the projects': its project, its
// member and itself.
func (f *fakeStore) membership(id uuid.UUID) (project, user uuid.UUID, m app.Membership, ok bool) {
	for pid, p := range f.projects {
		for uid, m := range p.members {
			if m.ID == id {
				return pid, uid, m, true
			}
		}
	}
	return uuid.UUID{}, uuid.UUID{}, app.Membership{}, false
}

// memberReads is how a membership read again under the locks answers: err
// fails it, gone finds none, as if it was deleted meanwhile, project,
// when set, is the project it is found in, as if it had moved there,
// role, when set, its role, as if it had been changed meanwhile, ended
// finds it ended, as if it had been ended meanwhile, and id, when set, is
// the id it answers for the one asked.
type memberReads struct {
	err     error
	gone    bool
	project uuid.UUID
	role    shared.Role
	ended   bool
	id      uuid.UUID
}

func (f *fakeStore) MemberByID(ctx context.Context, id uuid.UUID) (app.ProjectMembership, bool, error) {
	f.log.add(ctx, "MemberByID %s", id)
	f.memberReadCount++
	again := f.memberReadCount > 1
	if err := f.fail("MemberByID"); err != nil {
		return app.ProjectMembership{}, false, err
	}
	if again && f.reread.err != nil {
		return app.ProjectMembership{}, false, fmt.Errorf("MemberByID: %w", f.reread.err)
	}
	project, user, m, ok := f.membership(id)
	if !ok || (again && f.reread.gone) {
		return app.ProjectMembership{}, false, nil
	}
	out := app.ProjectMembership{ID: m.ID, WorkspaceID: f.projects[project].workspace, ProjectID: project, MemberID: user, Role: m.Role,
		Active: m.Active}
	if again && f.reread.project != (uuid.UUID{}) {
		out.ProjectID = f.reread.project
	}
	if again && f.reread.role != 0 {
		out.Role = f.reread.role
	}
	if again && f.reread.ended {
		out.Active = false
	}
	if again && f.reread.id != (uuid.UUID{}) {
		out.ID = f.reread.id
	}
	if f.answersAs != (uuid.UUID{}) {
		out.ID = f.answersAs
	}
	return out, true, nil
}

func (f *fakeStore) UpdateMemberRole(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) (domain.Member, error) {
	f.log.add(ctx, "UpdateMemberRole %s as %d by %s at %s", id, role, by, now.Format(timeFormat))
	if err := f.fail("UpdateMemberRole"); err != nil {
		return domain.Member{}, err
	}
	project, user, m, ok := f.membership(id)
	if !ok || !m.Active {
		return domain.Member{}, fmt.Errorf("UpdateMemberRole: no active membership %s", id)
	}
	m.Role = role
	f.projects[project].members[user] = m
	answer := domain.Member{ID: id, ProjectID: project, MemberID: user, Role: role, CreatedAt: memberSince}
	if f.changedAs != (uuid.UUID{}) {
		answer.ID = f.changedAs
	}
	return answer, nil
}

// frank is a project admin in newMemberWrites; the memberships it adds,
// by id, the same in every fixture.
var (
	frank                           = uuid.NewV7()
	ginaInWeb, frankInWeb, ivyInWeb = uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
)

// newMemberWrites is newGrowth with more members of web: gina, the
// workspace's admin, its member; frank, the workspace's member, a second
// admin; ivy, the workspace's guest, its guest. gina's grant in acme is a
// workspace admin's who is a project member.
func newMemberWrites() *writeFixture {
	f := newGrowth()
	web := f.store.projects[webID]
	web.members[gina] = app.Membership{ID: ginaInWeb, Role: shared.RoleMember, Active: true}
	web.members[frank] = app.Membership{ID: frankInWeb, Role: shared.RoleAdmin, Active: true}
	web.members[ivy] = app.Membership{ID: ivyInWeb, Role: shared.RoleGuest, Active: true}
	f.members.roles[acme.ID][frank] = shared.RoleMember
	f.auth.grants[grantKey{gina, acme.ID}] = shared.Grant{WorkspaceRole: shared.RoleAdmin, ProjectRole: shared.RoleMember, ProjectAdmin: true}
	return f
}

// memberOf is user's membership of project in f, by id.
func (f *writeFixture) memberOf(project, user uuid.UUID) uuid.UUID {
	return f.store.projects[project].members[user].ID
}

// memberLocked are the calls of a write by caller on the membership id,
// user's of project, up to its decision on action: the transaction, the
// membership read, acme's row FOR SHARE, user's membership of acme FOR
// SHARE when target is set, the project FOR NO KEY UPDATE, the membership
// read again, the decision.
func memberLocked(id, user, project, caller uuid.UUID, action shared.Action, target bool) []string {
	calls := []string{"Begin", "MemberByID " + id.String(), "ShareWorkspaceByID " + acme.ID.String()}
	if target {
		calls = append(calls, fmt.Sprintf("ShareMembers %s %v", acme.ID, []uuid.UUID{user}))
	}
	return append(calls, "LockProject "+project.String(), "MemberByID "+id.String(),
		fmt.Sprintf("Authorize %s %s on %s/%s", caller, action, acme.ID, project))
}

// outcome is how a write ends in a case: want is the refusal the API
// answers (a *shared.Error), a failure that comes back as itself (errDisk),
// or nil for the write's own error, neither of them: a 500. calls are the
// calls up to the end.
type outcome struct {
	name  string
	want  error
	calls []string
}

// check fails the test unless err, the write's answer, is tt's: a refusal
// is the first *shared.Error in err's chain, the one the API answers, of
// want's kind, code and fields (sameError); a failure is want, and the
// write's own error is an error, each with no *shared.Error in the chain.
// Once the write began its transaction, err is what came out of it
// (fakeTx.answered). The calls are tt.calls, each once, in order.
func (tt outcome) check(t *testing.T, err error, f *writeFixture) {
	t.Helper()
	var se, refusal *shared.Error
	switch {
	case errors.As(tt.want, &refusal):
		if !sameError(err, tt.want) {
			t.Errorf("%s: Execute() = %v, answered as another problem; want %v", tt.name, err, tt.want)
		}
	case errors.As(err, &se):
		t.Errorf("%s: Execute() = %v, answered as %s; want a 500", tt.name, err, se.Code)
	case err == nil, tt.want != nil && !errors.Is(err, tt.want):
		t.Errorf("%s: Execute() = %v; want %v, or an error of the write's own for nil", tt.name, err, tt.want)
	}
	if len(tt.calls) > 0 && !f.tx.answered(err) {
		t.Errorf("%s: Execute() = %v, the transaction's function returned %v, its commit failing with %v; want the answer to come out "+
			"of the transaction as itself", tt.name, err, f.tx.returned, f.tx.commitErr)
	}
	if !slices.Equal(f.log.calls, tt.calls) {
		t.Errorf("%s: calls\n%q\nwant\n%q", tt.name, f.log.calls, tt.calls)
	}
}
