package bootstrap

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
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
