package app_test

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeDirectory finds the workspaces it holds by slug, logs each call and
// fails with err.
type fakeDirectory struct {
	log        *callLog
	workspaces map[string]app.Workspace
	err        error
}

func (f *fakeDirectory) WorkspaceBySlug(ctx context.Context, slug string) (app.Workspace, bool, error) {
	f.log.add(ctx, "WorkspaceBySlug %s", slug)
	if f.err != nil {
		return app.Workspace{}, false, f.err
	}
	w, ok := f.workspaces[slug]
	return w, ok, nil
}

func (f *fakeDirectory) ShareWorkspaceBySlug(ctx context.Context, slug string) (app.Workspace, bool, error) {
	f.log.add(ctx, "ShareWorkspaceBySlug %s", slug)
	if f.err != nil {
		return app.Workspace{}, false, f.err
	}
	w, ok := f.workspaces[slug]
	return w, ok, nil
}

// fakeMembers answers the active members' roles it holds by workspace,
// only of the accounts asked for; it logs each call and fails with err.
type fakeMembers struct {
	log   *callLog
	roles map[uuid.UUID]map[uuid.UUID]shared.Role // workspace → account → role
	err   error
}

func (f *fakeMembers) ShareMembers(ctx context.Context, workspaceID uuid.UUID, userIDs []uuid.UUID) (map[uuid.UUID]shared.Role, error) {
	f.log.add(ctx, "ShareMembers %s %v", workspaceID, userIDs)
	if f.err != nil {
		return nil, f.err
	}
	out := map[uuid.UUID]shared.Role{}
	for _, id := range userIDs {
		if role, ok := f.roles[workspaceID][id]; ok {
			out[id] = role
		}
	}
	return out, nil
}

// fakeProjects is createProject's repository: it logs each call with its
// arguments, keeps what it is given, and answers GetProject from it, the
// times as stored (to the microsecond); errs fails a method by its name.
type fakeProjects struct {
	log     *callLog
	lowest  map[uuid.UUID]*float64 // by account
	project *app.ProjectRow
	members []app.MemberRow
	prefs   []app.PreferencesRow
	states  []app.StateRow
	errs    map[string]error
	missing bool // GetProject finds nothing
}

func (f *fakeProjects) fail(name string) error {
	if err := f.errs[name]; err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func (f *fakeProjects) CreateProject(ctx context.Context, p app.ProjectRow) error {
	logo, _ := json.Marshal(p.LogoProps)
	f.log.add(ctx, "CreateProject %s in %s %q %q %q network %d lead %v logo %s timezone %s by %s at %s", p.ID, p.WorkspaceID, p.Name,
		p.Identifier, p.Description, p.Network, lead(p.LeadID), logo, p.Timezone, p.CreatedBy, p.Now.Format(timeFormat))
	f.project = &p
	return f.fail("CreateProject")
}

func (f *fakeProjects) CreateMember(ctx context.Context, m app.MemberRow) error {
	f.log.add(ctx, "CreateMember %s in %s/%s as %d by %s at %s", m.MemberID, m.WorkspaceID, m.ProjectID, m.Role, m.CreatedBy, m.Now.Format(timeFormat))
	f.members = append(f.members, m)
	return f.fail("CreateMember")
}

func (f *fakeProjects) LowestSortOrder(ctx context.Context, workspaceID, userID uuid.UUID) (*float64, error) {
	f.log.add(ctx, "LowestSortOrder %s %s", workspaceID, userID)
	return f.lowest[userID], f.fail("LowestSortOrder")
}

func (f *fakeProjects) CreatePreferences(ctx context.Context, p app.PreferencesRow) error {
	f.log.add(ctx, "CreatePreferences %s in %s/%s at %v by %s at %s", p.UserID, p.WorkspaceID, p.ProjectID, p.SortOrder, p.CreatedBy,
		p.Now.Format(timeFormat))
	f.prefs = append(f.prefs, p)
	return f.fail("CreatePreferences")
}

func (f *fakeProjects) CreateStates(ctx context.Context, rows []app.StateRow) error {
	var names []string
	for _, r := range rows {
		names = append(names, fmt.Sprintf("%s/%s %s by %s at %s", r.WorkspaceID, r.ProjectID, r.State.Name, r.CreatedBy, r.Now.Format(timeFormat)))
	}
	f.log.add(ctx, "CreateStates %q", names)
	f.states = append(f.states, rows...)
	return f.fail("CreateStates")
}

func (f *fakeProjects) GetProject(ctx context.Context, id, userID uuid.UUID) (domain.Project, bool, error) {
	f.log.add(ctx, "GetProject %s for %s", id, userID)
	if err := f.fail("GetProject"); err != nil || f.missing || f.project == nil || f.project.ID != id {
		return domain.Project{}, false, err
	}
	p := domain.Project{ID: f.project.ID, WorkspaceID: f.project.WorkspaceID, Name: f.project.Name, Description: f.project.Description,
		Identifier: f.project.Identifier, Network: f.project.Network, LeadID: f.project.LeadID, LogoProps: f.project.LogoProps,
		Timezone: f.project.Timezone, CreatedAt: f.project.Now.Truncate(time.Microsecond), UpdatedAt: f.project.Now.Truncate(time.Microsecond)}
	for _, m := range f.members {
		p.MemberIDs = append(p.MemberIDs, m.MemberID)
		if m.MemberID == userID {
			role := m.Role
			p.MemberRole = &role
		}
	}
	if i := slices.IndexFunc(f.prefs, func(r app.PreferencesRow) bool { return r.UserID == userID }); i >= 0 {
		p.SortOrder = &f.prefs[i].SortOrder
	}
	return p, true, nil
}

const timeFormat = "2006-01-02T15:04:05.999999999"

// lead is the lead's id, or <nil>.
func lead(id *uuid.UUID) string {
	if id == nil {
		return "<nil>"
	}
	return id.String()
}
