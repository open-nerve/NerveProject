package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeIdentityAccounts answers the states it holds, by id and by address, and
// records what it was asked.
type fakeIdentityAccounts struct {
	states []identityapp.AccountState
	err    error
	asked  []string
}

func (f *fakeIdentityAccounts) ShareAccount(_ context.Context, id uuid.UUID) (identityapp.AccountState, bool, error) {
	f.asked = append(f.asked, "id "+id.String())
	i := slices.IndexFunc(f.states, func(s identityapp.AccountState) bool { return s.ID == id })
	if i < 0 {
		return identityapp.AccountState{}, false, f.err
	}
	return f.states[i], true, f.err
}

func (f *fakeIdentityAccounts) ShareAccountByEmail(_ context.Context, email string) (identityapp.AccountState, bool, error) {
	f.asked = append(f.asked, "email "+email)
	i := slices.IndexFunc(f.states, func(s identityapp.AccountState) bool { return s.Email == email })
	if i < 0 {
		return identityapp.AccountState{}, false, f.err
	}
	return f.states[i], true, f.err
}

// workspaceAccounts hands workspace identity's answer to the same question:
// the account asked for, its state converted, found and the error as they
// came.
func TestWorkspaceAccountsConvertsIdentitysAnswer(t *testing.T) {
	alice := identityapp.AccountState{ID: uuid.NewV7(), Email: "alice@corp.com", Active: true}
	bob := identityapp.AccountState{ID: uuid.NewV7(), Email: "bob@corp.com", Active: false}
	fake := &fakeIdentityAccounts{states: []identityapp.AccountState{alice, bob}}
	a := workspaceAccounts{accounts: fake}
	ctx := context.Background()

	type answer struct {
		state workspace.AccountState
		found bool
	}
	var got []answer
	for _, id := range []uuid.UUID{alice.ID, bob.ID, uuid.NewV7()} {
		s, found, err := a.ShareAccount(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, answer{s, found})
	}
	for _, email := range []string{"bob@corp.com", "carol@corp.com"} {
		s, found, err := a.ShareAccountByEmail(ctx, email)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, answer{s, found})
	}
	want := []answer{
		{workspace.AccountState{ID: alice.ID, Email: "alice@corp.com", Active: true}, true},
		{workspace.AccountState{ID: bob.ID, Email: "bob@corp.com", Active: false}, true},
		{},
		{workspace.AccountState{ID: bob.ID, Email: "bob@corp.com", Active: false}, true},
		{},
	}
	if !slices.Equal(got, want) {
		t.Errorf("answers = %+v, want %+v", got, want)
	}
	if len(fake.asked) != 5 || fake.asked[0] != "id "+alice.ID.String() || fake.asked[3] != "email bob@corp.com" {
		t.Errorf("identity was asked %q", fake.asked)
	}

	failure := errors.New("connection reset")
	fake.err = failure
	if _, _, err := a.ShareAccount(ctx, alice.ID); !errors.Is(err, failure) {
		t.Errorf("ShareAccount() = %v, want %v", err, failure)
	}
	if _, _, err := a.ShareAccountByEmail(ctx, "alice@corp.com"); !errors.Is(err, failure) {
		t.Errorf("ShareAccountByEmail() = %v, want %v", err, failure)
	}
}

// fakeIdentityProfiles answers the profiles it holds of the ids asked for,
// in the order it holds them, and records the ids.
type fakeIdentityProfiles struct {
	profiles []identityapp.PublicProfile
	err      error
	asked    [][]uuid.UUID
}

func (f *fakeIdentityProfiles) PublicProfiles(_ context.Context, ids []uuid.UUID) ([]identityapp.PublicProfile, error) {
	f.asked = append(f.asked, ids)
	var out []identityapp.PublicProfile
	for _, p := range f.profiles {
		if slices.Contains(ids, p.ID) {
			out = append(out, p)
		}
	}
	return out, f.err
}

// workspaceProfiles asks identity for the ids workspace asks for and hands
// over each profile, every field converted, in identity's order; an error
// as it came, without profiles.
func TestWorkspaceProfilesConvertsIdentitysAnswer(t *testing.T) {
	alice := identityapp.PublicProfile{ID: uuid.NewV7(), Email: "alice@corp.com", FirstName: "Alice", LastName: "Liddell", DisplayName: "al"}
	bob := identityapp.PublicProfile{ID: uuid.NewV7(), Email: "bob@corp.com", DisplayName: "bob"}
	fake := &fakeIdentityProfiles{profiles: []identityapp.PublicProfile{alice, bob}}
	p := workspaceProfiles{profiles: fake}
	ids := []uuid.UUID{bob.ID, uuid.NewV7(), alice.ID}

	got, err := p.PublicProfiles(context.Background(), ids)

	want := []workspace.PublicProfile{
		{ID: alice.ID, Email: "alice@corp.com", FirstName: "Alice", LastName: "Liddell", DisplayName: "al"},
		{ID: bob.ID, Email: "bob@corp.com", DisplayName: "bob"},
	}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("PublicProfiles() = %+v, %v; want %+v", got, err, want)
	}
	if len(fake.asked) != 1 || !slices.Equal(fake.asked[0], ids) {
		t.Errorf("identity was asked %v, want %v", fake.asked, ids)
	}
	failure := errors.New("connection reset")
	fake.err = failure
	if got, err := p.PublicProfiles(context.Background(), ids); !errors.Is(err, failure) || got != nil {
		t.Errorf("PublicProfiles() = %+v, %v; want no profiles and %v", got, err, failure)
	}
}

// fakeWorkspaceDirectory answers the workspaces it holds by slug, or by the
// id names gives a slug, and records what it was asked: "read" without a
// lock, "share" with one, "share by id" by the id's name.
type fakeWorkspaceDirectory struct {
	workspaces map[string]workspace.DirectoryEntry
	names      map[uuid.UUID]string
	err        error
	asked      []string
}

func (f *fakeWorkspaceDirectory) WorkspaceBySlug(_ context.Context, slug string) (workspace.DirectoryEntry, bool, error) {
	f.asked = append(f.asked, "read "+slug)
	w, found := f.workspaces[slug]
	return w, found, f.err
}

func (f *fakeWorkspaceDirectory) ShareWorkspaceBySlug(_ context.Context, slug string) (workspace.DirectoryEntry, bool, error) {
	f.asked = append(f.asked, "share "+slug)
	w, found := f.workspaces[slug]
	return w, found, f.err
}

func (f *fakeWorkspaceDirectory) ShareWorkspaceByID(_ context.Context, id uuid.UUID) (workspace.DirectoryEntry, bool, error) {
	f.asked = append(f.asked, "share by id "+f.names[id])
	w, found := f.workspaces[f.names[id]]
	return w, found, f.err
}

// projectWorkspaces hands project workspace's answer to the same question,
// through the same lock or without one, by slug or by id: the workspace
// converted, found and the error as they came.
func TestProjectWorkspacesConvertsWorkspacesAnswer(t *testing.T) {
	acme := workspace.DirectoryEntry{ID: uuid.NewV7(), Timezone: "Asia/Shanghai"}
	ids := map[string]uuid.UUID{"acme": acme.ID, "gone": uuid.NewV7()}
	fake := &fakeWorkspaceDirectory{workspaces: map[string]workspace.DirectoryEntry{"acme": acme},
		names: map[uuid.UUID]string{ids["acme"]: "acme", ids["gone"]: "gone"}}
	d := projectWorkspaces{directory: fake}
	ctx := context.Background()
	want := project.Workspace{ID: acme.ID, Timezone: "Asia/Shanghai"}

	for name, find := range map[string]func(context.Context, string) (project.Workspace, bool, error){
		"share": d.ShareWorkspaceBySlug, "read": d.WorkspaceBySlug,
		"share by id": func(ctx context.Context, slug string) (project.Workspace, bool, error) {
			return d.ShareWorkspaceByID(ctx, ids[slug])
		},
	} {
		fake.asked, fake.err = nil, nil
		if w, found, err := find(ctx, "acme"); err != nil || !found || w != want {
			t.Errorf("%s acme = %+v, %v, %v; want %+v, found", name, w, found, err, want)
		}
		if w, found, err := find(ctx, "gone"); err != nil || found || w != (project.Workspace{}) {
			t.Errorf("%s gone = %+v, %v, %v; want not found", name, w, found, err)
		}
		if want := []string{name + " acme", name + " gone"}; !slices.Equal(fake.asked, want) {
			t.Errorf("workspace was asked %q, want %q", fake.asked, want)
		}
		failure := errors.New("connection reset")
		fake.err = failure
		if _, _, err := find(ctx, "acme"); !errors.Is(err, failure) {
			t.Errorf("%s = %v, want %v", name, err, failure)
		}
	}
}

// fakeProjectAccess answers the facts it holds by project, and records what
// it was asked.
type fakeProjectAccess struct {
	facts map[uuid.UUID]project.AccessFacts
	err   error
	asked []string
}

func (f *fakeProjectAccess) ProjectFacts(_ context.Context, projectID, userID uuid.UUID) (project.AccessFacts, bool, error) {
	f.asked = append(f.asked, projectID.String()+" "+userID.String())
	p, found := f.facts[projectID]
	return p, found, f.err
}

// accessProjects hands access project's answer to the same question: the
// facts converted, found and the error as they came.
func TestAccessProjectsConvertsProjectsAnswer(t *testing.T) {
	web, acme, alice := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	fake := &fakeProjectAccess{facts: map[uuid.UUID]project.AccessFacts{web: {WorkspaceID: acme, Public: true, Member: true, Role: 15}}}
	a := accessProjects{projects: fake}
	ctx := context.Background()

	f, found, err := a.ProjectFacts(ctx, web, alice)
	if want := (access.ProjectFacts{WorkspaceID: acme, Public: true, Member: true, Role: 15}); err != nil || !found || f != want {
		t.Errorf("ProjectFacts(web) = %+v, %v, %v; want %+v, found", f, found, err, want)
	}
	gone := uuid.NewV7()
	if f, found, err := a.ProjectFacts(ctx, gone, alice); err != nil || found || f != (access.ProjectFacts{}) {
		t.Errorf("ProjectFacts(gone) = %+v, %v, %v; want not found", f, found, err)
	}
	if want := []string{web.String() + " " + alice.String(), gone.String() + " " + alice.String()}; !slices.Equal(fake.asked, want) {
		t.Errorf("project was asked %q, want %q", fake.asked, want)
	}
	failure := errors.New("connection reset")
	fake.err = failure
	if _, _, err := a.ProjectFacts(ctx, web, alice); !errors.Is(err, failure) {
		t.Errorf("ProjectFacts() = %v, want %v", err, failure)
	}
}

// fakeIssueProjects answers project's IssueProjects of one project, web,
// and records each question; err, when set, is every answer's failure.
type fakeIssueProjects struct {
	web    uuid.UUID
	locked project.IssueProject
	states []project.IssueState
	labels []project.IssueLabel
	roles  map[uuid.UUID]shared.Role
	err    error
	asked  []string
}

func (f *fakeIssueProjects) ProjectWorkspace(_ context.Context, id uuid.UUID) (uuid.UUID, bool, error) {
	f.asked = append(f.asked, "workspace")
	return f.locked.WorkspaceID, id == f.web, f.err
}

func (f *fakeIssueProjects) LockIssueProject(_ context.Context, id uuid.UUID) (project.IssueProject, bool, error) {
	f.asked = append(f.asked, "lock")
	return f.locked, id == f.web, f.err
}

func (f *fakeIssueProjects) NextIssueSequence(context.Context, uuid.UUID) (int, error) {
	f.asked = append(f.asked, "next")
	return 7, f.err
}

func (f *fakeIssueProjects) IssueStates(context.Context, uuid.UUID) ([]project.IssueState, error) {
	f.asked = append(f.asked, "states")
	return f.states, f.err
}

func (f *fakeIssueProjects) IssueLabels(_ context.Context, _ uuid.UUID, ids []uuid.UUID) ([]project.IssueLabel, error) {
	f.asked = append(f.asked, fmt.Sprint("labels ", ids))
	return f.labels, f.err
}

func (f *fakeIssueProjects) ActiveRoles(_ context.Context, _ uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]shared.Role, error) {
	f.asked = append(f.asked, fmt.Sprint("roles ", ids))
	return f.roles, f.err
}

// issueProjects hands issue project's answers to the same questions: the
// locked project, each state and each label converted, every field; the
// workspace, the number and the roles as they came; found and the error as
// they came, and no state or label with an error.
func TestIssueProjectsConvertsProjectsAnswers(t *testing.T) {
	web, acme, ann, bug, ui := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	backlog := project.IssueState{ID: uuid.NewV7(), Name: "Backlog", Group: "backlog", Default: true}
	done := project.IssueState{ID: uuid.NewV7(), Name: "Done", Group: "completed"}
	fake := &fakeIssueProjects{web: web, locked: project.IssueProject{WorkspaceID: acme, Archived: true, DefaultAssigneeID: &ann},
		states: []project.IssueState{backlog, done}, labels: []project.IssueLabel{{ID: bug, Name: "Bug"}, {ID: ui, Name: "UI"}},
		roles: map[uuid.UUID]shared.Role{ann: shared.RoleMember}}
	p := issueProjects{projects: fake}
	ctx := context.Background()

	if w, found, err := p.Find(ctx, web); err != nil || !found || w != acme {
		t.Errorf("Find() = %s, %v, %v; want acme, found", w, found, err)
	}
	if l, found, err := p.Lock(ctx, web); err != nil || !found || l != (issue.LockedProject{WorkspaceID: acme, Archived: true, DefaultAssigneeID: &ann}) {
		t.Errorf("Lock() = %+v, %v, %v; want web as locked, found", l, found, err)
	}
	if _, found, _ := p.Lock(ctx, uuid.NewV7()); found {
		t.Error("Lock() of another project found it")
	}
	if n, err := p.Next(ctx, web); err != nil || n != 7 {
		t.Errorf("Next() = %d, %v; want 7", n, err)
	}
	want := []issue.StateRef{{ID: backlog.ID, Name: "Backlog", Group: "backlog", Default: true}, {ID: done.ID, Name: "Done", Group: "completed"}}
	if got, err := p.States(ctx, web); err != nil || !slices.Equal(got, want) {
		t.Errorf("States() = %+v, %v; want %+v", got, err, want)
	}
	if got, err := p.Labels(ctx, web, []uuid.UUID{bug, ui}); err != nil || !slices.Equal(got, []issue.LabelRef{{ID: bug, Name: "Bug"}, {ID: ui, Name: "UI"}}) {
		t.Errorf("Labels() = %+v, %v; want Bug and UI", got, err)
	}
	if got, err := p.Roles(ctx, web, []uuid.UUID{ann}); err != nil || !maps.Equal(got, fake.roles) {
		t.Errorf("Roles() = %v, %v; want ann's", got, err)
	}
	if want := []string{"workspace", "lock", "lock", "next", "states", fmt.Sprint("labels ", []uuid.UUID{bug, ui}),
		fmt.Sprint("roles ", []uuid.UUID{ann})}; !slices.Equal(fake.asked, want) {
		t.Errorf("project was asked %q, want %q", fake.asked, want)
	}

	failure := errors.New("connection reset")
	fake.err = failure
	if got, err := p.States(ctx, web); !errors.Is(err, failure) || got != nil {
		t.Errorf("States() = %+v, %v; want no states and %v", got, err, failure)
	}
	if got, err := p.Labels(ctx, web, nil); !errors.Is(err, failure) || got != nil {
		t.Errorf("Labels() = %+v, %v; want no labels and %v", got, err, failure)
	}
	for name, call := range map[string]func() error{
		"Find":  func() error { _, _, err := p.Find(ctx, web); return err },
		"Lock":  func() error { _, _, err := p.Lock(ctx, web); return err },
		"Next":  func() error { _, err := p.Next(ctx, web); return err },
		"Roles": func() error { _, err := p.Roles(ctx, web, nil); return err },
	} {
		if err := call(); !errors.Is(err, failure) {
			t.Errorf("%s() = %v, want %v", name, err, failure)
		}
	}
}

// issueProfiles asks identity for the ids issue asks for and hands over
// each account's display name, by account; an error as it came, without
// names.
func TestIssueProfilesConvertsIdentitysAnswer(t *testing.T) {
	alice := identityapp.PublicProfile{ID: uuid.NewV7(), Email: "alice@corp.com", FirstName: "Alice", DisplayName: "al"}
	bob := identityapp.PublicProfile{ID: uuid.NewV7(), Email: "bob@corp.com", DisplayName: "bob"}
	fake := &fakeIdentityProfiles{profiles: []identityapp.PublicProfile{alice, bob}}
	p := issueProfiles{profiles: fake}
	ids := []uuid.UUID{bob.ID, uuid.NewV7(), alice.ID}

	got, err := p.DisplayNames(context.Background(), ids)
	if want := map[uuid.UUID]string{alice.ID: "al", bob.ID: "bob"}; err != nil || !maps.Equal(got, want) {
		t.Errorf("DisplayNames() = %v, %v; want %v", got, err, want)
	}
	if len(fake.asked) != 1 || !slices.Equal(fake.asked[0], ids) {
		t.Errorf("identity was asked %v, want %v", fake.asked, ids)
	}
	failure := errors.New("connection reset")
	fake.err = failure
	if got, err := p.DisplayNames(context.Background(), ids); !errors.Is(err, failure) || got != nil {
		t.Errorf("DisplayNames() = %v, %v; want no names and %v", got, err, failure)
	}
}
