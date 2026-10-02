package app_test

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeStore's growth of a project's memberships: it logs each write and
// keeps it in the project, so that ListMembers and GetProject answer it.

func (f *fakeStore) CreateMember(ctx context.Context, m app.MemberRow) error {
	f.log.add(ctx, "CreateMember %s in %s/%s as %d by %s at %s", m.MemberID, m.WorkspaceID, m.ProjectID, m.Role, m.CreatedBy, m.Now.Format(timeFormat))
	if err := f.fail("CreateMember"); err != nil {
		return err
	}
	f.projects[m.ProjectID].members[m.MemberID] = app.Membership{ID: m.ID, Role: m.Role, Active: true}
	return nil
}

func (f *fakeStore) RestoreMember(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "RestoreMember %s as %d by %s at %s", id, role, by, now.Format(timeFormat))
	if err := f.fail("RestoreMember"); err != nil {
		return err
	}
	for _, p := range f.projects {
		for user, m := range p.members {
			if m.ID == id {
				p.members[user] = app.Membership{ID: id, Role: role, Active: true}
			}
		}
	}
	return nil
}

func (f *fakeStore) EnsurePreferences(ctx context.Context, p app.PreferencesRow) error {
	f.log.add(ctx, "EnsurePreferences %s in %s/%s at %v by %s at %s", p.UserID, p.WorkspaceID, p.ProjectID, p.SortOrder, p.CreatedBy,
		p.Now.Format(timeFormat))
	if err := f.fail("EnsurePreferences"); err != nil {
		return err
	}
	project := f.projects[p.ProjectID]
	if project.prefs == nil {
		project.prefs = map[uuid.UUID]domain.Preferences{}
	}
	if _, ok := project.prefs[p.UserID]; !ok {
		project.prefs[p.UserID] = domain.Preferences{Navigation: domain.DefaultPreferences().Navigation, SortOrder: p.SortOrder}
	}
	return nil
}

func (f *fakeStore) LowestSortOrder(ctx context.Context, workspaceID, userID uuid.UUID) (*float64, error) {
	f.log.add(ctx, "LowestSortOrder %s %s", workspaceID, userID)
	return f.lowest[userID], f.fail("LowestSortOrder")
}

var gina, hank, ivy = uuid.NewV7(), uuid.NewV7(), uuid.NewV7()

// newGrowth is newWrites with the workspace's members (fakeMembers) in
// acme, by account: bob, alice and dave members, carol a guest; gina an
// admin, hank a member and ivy a guest, of no project; erin none. hank's
// least place in his sidebar is -5.
func newGrowth() *writeFixture {
	f := newWrites()
	f.store.lowest = map[uuid.UUID]*float64{hank: ptr(-5.0)}
	f.members.roles = map[uuid.UUID]map[uuid.UUID]shared.Role{acme.ID: {
		bob: shared.RoleMember, alice: shared.RoleMember, dave: shared.RoleMember, carol: shared.RoleGuest,
		gina: shared.RoleAdmin, hank: shared.RoleMember, ivy: shared.RoleGuest,
	}}
	return f
}

// grown are the calls of one account's growth, by by: his membership,
// restored when ended is set, or made; his display settings at sortOrder.
func grown(user uuid.UUID, ended *app.Membership, role shared.Role, sortOrder float64, by uuid.UUID) []string {
	write := fmt.Sprintf("CreateMember %s in %s/%s as %d by %s at %s", user, acme.ID, webID, role, by, clockNow.Format(timeFormat))
	if ended != nil {
		write = fmt.Sprintf("RestoreMember %s as %d by %s at %s", ended.ID, role, by, clockNow.Format(timeFormat))
	}
	return []string{write, fmt.Sprintf("EnsurePreferences %s in %s/%s at %v by %s at %s", user, acme.ID, webID, sortOrder, by,
		clockNow.Format(timeFormat))}
}
