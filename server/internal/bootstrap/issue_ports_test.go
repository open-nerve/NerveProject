package bootstrap

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
)

// The conversions of a read by identifier (M4 design 3.7): the ports
// getIssueByIdentifier asks, as bootstrap/ports.go hands them to issue.

// ProjectIdentifiers answers fakeIssueProjects' identifiers, of whatever
// workspace it is asked.
func (f *fakeIssueProjects) ProjectIdentifiers(_ context.Context, workspaceID uuid.UUID) ([]project.IssueProjectIdentifier, error) {
	f.asked = append(f.asked, "identifiers "+workspaceID.String())
	return f.identifiers, f.err
}

// issueProjects asks project for the workspace issue asks about and hands
// over each project's id and identifier, in their order; an error as it
// came, without identifiers.
func TestIssueProjectsConvertsProjectIdentifiers(t *testing.T) {
	acme, web, ops := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	fake := &fakeIssueProjects{identifiers: []project.IssueProjectIdentifier{{ID: web, Identifier: "WEB"}, {ID: ops, Identifier: "İŞ"}}}
	p := issueProjects{projects: fake}
	got, err := p.Identifiers(context.Background(), acme)
	if want := []issue.ProjectIdentifier{{ID: web, Identifier: "WEB"}, {ID: ops, Identifier: "İŞ"}}; err != nil || !slices.Equal(got, want) {
		t.Errorf("Identifiers() = %+v, %v; want %+v", got, err, want)
	}
	if want := []string{"identifiers " + acme.String()}; !slices.Equal(fake.asked, want) {
		t.Errorf("project was asked %q, want %q", fake.asked, want)
	}
	failure := errors.New("connection reset")
	fake.err = failure
	if got, err := p.Identifiers(context.Background(), acme); !errors.Is(err, failure) || got != nil {
		t.Errorf("Identifiers() = %+v, %v; want no identifiers and %v", got, err, failure)
	}
}

// issueWorkspaces hands issue workspace's answer to its read without a
// lock: the workspace's id, found and the error as they came.
func TestIssueWorkspacesConvertsWorkspacesAnswer(t *testing.T) {
	acme := workspace.DirectoryEntry{ID: uuid.NewV7(), Timezone: "Asia/Shanghai"}
	fake := &fakeWorkspaceDirectory{workspaces: map[string]workspace.DirectoryEntry{"acme": acme}}
	d := issueWorkspaces{directory: fake}
	ctx := context.Background()
	if id, found, err := d.WorkspaceBySlug(ctx, "acme"); err != nil || !found || id != acme.ID {
		t.Errorf("WorkspaceBySlug(acme) = %s, %v, %v; want %s, found", id, found, err, acme.ID)
	}
	if id, found, err := d.WorkspaceBySlug(ctx, "gone"); err != nil || found || id != (uuid.UUID{}) {
		t.Errorf("WorkspaceBySlug(gone) = %s, %v, %v; want not found", id, found, err)
	}
	if want := []string{"read acme", "read gone"}; !slices.Equal(fake.asked, want) {
		t.Errorf("workspace was asked %q, want %q", fake.asked, want)
	}
	failure := errors.New("connection reset")
	fake.err = failure
	if _, _, err := d.WorkspaceBySlug(ctx, "acme"); !errors.Is(err, failure) {
		t.Errorf("WorkspaceBySlug() = %v, want %v", err, failure)
	}
}
