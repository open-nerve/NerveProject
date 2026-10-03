package app_test

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeProject is a project as fakeStore holds it: its workspace, whether it
// is archived, its memberships and its members' display settings by
// account, and when it was last written, as stored.
type fakeProject struct {
	workspace uuid.UUID
	archived  bool
	members   map[uuid.UUID]app.Membership
	prefs     map[uuid.UUID]domain.Preferences
	updated   time.Time
}

// The projects of the writes' tests: acme's web, whose admin is bob, whose
// member is alice and whose guest is carol, dave's membership ended; acme's
// ops, archived, whose admin is bob. Bob has display settings in web
// (bobsTabs), the others none.
var webID, opsID = uuid.NewV7(), uuid.NewV7()

// The memberships of web and ops, by id, the same in every fixture.
var bobInWeb, aliceInWeb, carolInWeb, daveInWeb, bobInOps = uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()

var bobsTabs = domain.Preferences{Navigation: domain.Navigation{DefaultTab: "modules", HideInMoreMenu: []string{"views"}}, SortOrder: 10}

// writeFixture is a write use case's fakes, sharing one log.
type writeFixture struct {
	log        *callLog
	tx         *fakeTx
	store      *fakeStore
	workspaces *fakeWorkspaces
	members    *fakeMembers
	auth       *fakeAuthorizer
}

// newWrites is writeFixture with web and ops as stored at now, acme's row
// to lock, no workspace member to lock, and the Authorizer's answers: bob's
// grant in acme, the project's admin; alice's 403, a project member; any
// other caller sees nothing.
func newWrites() *writeFixture {
	log := &callLog{}
	return &writeFixture{log: log, tx: &fakeTx{log: log}, workspaces: &fakeWorkspaces{log: log}, members: &fakeMembers{log: log},
		store: &fakeStore{log: log, projects: map[uuid.UUID]*fakeProject{
			webID: {workspace: acme.ID, updated: now, members: map[uuid.UUID]app.Membership{
				bob:   {ID: bobInWeb, Role: shared.RoleAdmin, Active: true},
				alice: {ID: aliceInWeb, Role: shared.RoleMember, Active: true},
				carol: {ID: carolInWeb, Role: shared.RoleGuest, Active: true},
				dave:  {ID: daveInWeb, Role: shared.RoleMember},
			}, prefs: map[uuid.UUID]domain.Preferences{bob: bobsTabs}},
			opsID: {workspace: acme.ID, archived: true, updated: now, members: map[uuid.UUID]app.Membership{
				bob: {ID: bobInOps, Role: shared.RoleAdmin, Active: true},
			}},
		}},
		auth: &fakeAuthorizer{log: log,
			grants: map[grantKey]shared.Grant{{bob, acme.ID}: {WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleAdmin, ProjectAdmin: true}},
			errs:   map[grantKey]error{{alice, acme.ID}: shared.Forbidden()}},
	}
}

// locks is Locks over the fixture's fakes.
func (f *writeFixture) locks() app.Locks {
	return app.NewLocks(f.store, f.workspaces, f.members, f.auth)
}

// fakeWorkspaces is the workspace module's lock of a workspace's row by its
// id: it logs each call, finds acme unless gone, and fails with err.
type fakeWorkspaces struct {
	log  *callLog
	gone bool
	err  error
}

func (f *fakeWorkspaces) ShareWorkspaceByID(ctx context.Context, id uuid.UUID) (app.Workspace, bool, error) {
	f.log.add(ctx, "ShareWorkspaceByID %s", id)
	if f.err != nil {
		return app.Workspace{}, false, f.err
	}
	if f.gone || id != acme.ID {
		return app.Workspace{}, false, nil
	}
	return acme, true, nil
}

// lockedTo are the calls of Locks on project before its own lock: the
// transaction, the project's workspace, acme's row FOR SHARE.
func lockedTo(project uuid.UUID) []string {
	return []string{"Begin", "ProjectWorkspace " + project.String(), "ShareWorkspaceByID " + acme.ID.String()}
}

// lockedDecision are the calls of a write on project by user before its
// checks: its locks (the workspace's, the project's FOR NO KEY UPDATE) and
// the decision on action.
func lockedDecision(user, project uuid.UUID, action shared.Action) []string {
	return append(lockedTo(project), "LockProject "+project.String(), fmt.Sprintf("Authorize %s %s on %s/%s", user, action, acme.ID, project))
}

// noProject are the calls of a write on a project that is not there: the
// transaction, the project's workspace, which finds none.
var noProject = []string{"Begin", "ProjectWorkspace " + uuid.Nil().String()}

// fakeStore is the project store of the writes on a project: it logs each
// call with its arguments, holds the projects by id, writes what it is
// given into them and answers GetProject from them, the time as stored (to
// the microsecond). errs fails a method by its name, after logging the
// call; missing makes GetProject and ListMembers find nothing, as if the
// writes were lost; lowest is each account's least place in his sidebar;
// moved, when set, is the workspace each project's lock reads, as if the
// project had moved there since ProjectWorkspace read it.
type fakeStore struct {
	log      *callLog
	projects map[uuid.UUID]*fakeProject
	errs     map[string]error
	missing  bool
	lowest   map[uuid.UUID]*float64
	moved    uuid.UUID
	deleted  bool // each project's lock finds nothing, as if it was deleted while the lock waited
	// The reads of a membership by its id (fakes_member_test.go): how many
	// ran, how the second one answers, and answersAs, when set, the id each
	// answers for the one asked; changedAs, when set, is the id
	// UpdateMemberRole answers for the one it changed.
	memberReadCount int
	reread          memberReads
	answersAs       uuid.UUID
	changedAs       uuid.UUID
}

func (f *fakeStore) fail(name string) error {
	if err := f.errs[name]; err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// lock is LockProject and ShareProject, named by method.
func (f *fakeStore) lock(ctx context.Context, method string, id uuid.UUID) (app.LockedProject, bool, error) {
	f.log.add(ctx, "%s %s", method, id)
	if err := f.fail(method); err != nil {
		return app.LockedProject{}, false, err
	}
	p, ok := f.projects[id]
	if !ok || f.deleted {
		return app.LockedProject{}, false, nil
	}
	if f.moved != (uuid.UUID{}) {
		return app.LockedProject{WorkspaceID: f.moved, Archived: p.archived}, true, nil
	}
	return app.LockedProject{WorkspaceID: p.workspace, Archived: p.archived}, true, nil
}

func (f *fakeStore) LockProject(ctx context.Context, id uuid.UUID) (app.LockedProject, bool, error) {
	return f.lock(ctx, "LockProject", id)
}

func (f *fakeStore) ShareProject(ctx context.Context, id uuid.UUID) (app.LockedProject, bool, error) {
	return f.lock(ctx, "ShareProject", id)
}

func (f *fakeStore) ProjectWorkspace(ctx context.Context, id uuid.UUID) (uuid.UUID, bool, error) {
	f.log.add(ctx, "ProjectWorkspace %s", id)
	if err := f.fail("ProjectWorkspace"); err != nil {
		return uuid.UUID{}, false, err
	}
	p, ok := f.projects[id]
	if !ok {
		return uuid.UUID{}, false, nil
	}
	return p.workspace, true, nil
}

func (f *fakeStore) Memberships(ctx context.Context, projectID uuid.UUID, userIDs []uuid.UUID) (map[uuid.UUID]app.Membership, error) {
	f.log.add(ctx, "Memberships %s %v", projectID, userIDs)
	if err := f.fail("Memberships"); err != nil {
		return nil, err
	}
	out := map[uuid.UUID]app.Membership{}
	if p, ok := f.projects[projectID]; ok {
		for _, user := range userIDs {
			if m, ok := p.members[user]; ok {
				out[user] = m
			}
		}
	}
	return out, nil
}

func (f *fakeStore) UpdateProject(ctx context.Context, id uuid.UUID, p domain.ProjectPatch, by uuid.UUID, now time.Time) error {
	patch, _ := json.Marshal(p)
	f.log.add(ctx, "UpdateProject %s %s by %s at %s", id, patch, by, now.Format(timeFormat))
	if err := f.fail("UpdateProject"); err != nil {
		return err
	}
	f.projects[id].updated = now.Truncate(time.Microsecond)
	return nil
}

func (f *fakeStore) SetArchived(ctx context.Context, id uuid.UUID, archived bool, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "SetArchived %s %v by %s at %s", id, archived, by, now.Format(timeFormat))
	if err := f.fail("SetArchived"); err != nil {
		return err
	}
	f.projects[id].archived, f.projects[id].updated = archived, now.Truncate(time.Microsecond)
	return nil
}

// deleting logs a step of a deletion: the workspace, the project, or every
// project of the workspace ("*"), the account and the moment.
func (f *fakeStore) deleting(ctx context.Context, step string, d app.Deletion) error {
	project := "*"
	if d.ProjectID != nil {
		project = d.ProjectID.String()
	}
	f.log.add(ctx, "%s %s/%s by %s at %s", step, d.WorkspaceID, project, d.By, d.Now.Format(timeFormat))
	return f.fail(step)
}

func (f *fakeStore) DeleteProjects(ctx context.Context, d app.Deletion) error {
	return f.deleting(ctx, "DeleteProjects", d)
}

func (f *fakeStore) DeleteProjectMembers(ctx context.Context, d app.Deletion) error {
	return f.deleting(ctx, "DeleteProjectMembers", d)
}

func (f *fakeStore) DeleteProjectPreferences(ctx context.Context, d app.Deletion) error {
	return f.deleting(ctx, "DeleteProjectPreferences", d)
}

func (f *fakeStore) DeleteStates(ctx context.Context, d app.Deletion) error {
	return f.deleting(ctx, "DeleteStates", d)
}

func (f *fakeStore) GetProject(ctx context.Context, id, userID uuid.UUID) (domain.Project, bool, error) {
	f.log.add(ctx, "GetProject %s for %s", id, userID)
	if err := f.fail("GetProject"); err != nil {
		return domain.Project{}, false, err
	}
	p, ok := f.projects[id]
	if !ok || f.missing {
		return domain.Project{}, false, nil
	}
	out := domain.Project{ID: id, WorkspaceID: p.workspace, UpdatedAt: p.updated}
	if p.archived {
		out.ArchivedAt = &p.updated
	}
	if m, ok := p.members[userID]; ok && m.Active {
		role := m.Role
		out.MemberRole = &role
	}
	return out, true, nil
}
