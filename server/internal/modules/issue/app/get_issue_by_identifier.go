package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// GetIssueByIdentifier reads a work item by its identifier, WEB-12, in a
// workspace (M4 design 3.7, 5.1).
type GetIssueByIdentifier struct {
	d GetIssueByIdentifierDeps
}

// GetIssueByIdentifierDeps are the use case's ports.
type GetIssueByIdentifierDeps struct {
	Locks      Locks
	Workspaces WorkspaceDirectory
	Projects   ProjectIdentifiers
	Issues     IssueNumbers
}

// NewGetIssueByIdentifier returns the use case.
func NewGetIssueByIdentifier(d GetIssueByIdentifierDeps) *GetIssueByIdentifier {
	return &GetIssueByIdentifier{d: d}
}

// Execute finds the workspace slug names and decides issue.search in it:
// one that is not there, or of which the caller is not an active member,
// is workspace.not_found. Then the work item identifier names: its
// project's identifier, compared folded with the workspace's undeleted
// projects' (domain.MatchIdentifier), and its number in that project. It
// reads the work item as getIssue does, without a lock or a transaction
// (Locks.see): an identifier that is no identifier, that names no
// project, or a number no undeleted work item of it has, is
// issue.not_found as one whose project the caller does not see, or that
// the guest rule hides from him, with the same answer; a role that may
// not read it, forbidden. An archived work item is read as any other.
func (u *GetIssueByIdentifier) Execute(ctx context.Context, slug, identifier string) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	workspaceID, found, err := u.d.Workspaces.WorkspaceBySlug(ctx, slug)
	switch {
	case err != nil:
		return domain.Issue{}, err
	case !found:
		return domain.Issue{}, domain.ErrWorkspaceNotFound
	}
	if err := u.d.Locks.seeWorkspace(ctx, actor, domain.ActionSearch, workspaceID); err != nil {
		return domain.Issue{}, err
	}
	id, err := u.find(ctx, workspaceID, identifier)
	if err != nil {
		return domain.Issue{}, err
	}
	return u.d.Locks.see(ctx, actor, id, domain.ActionRead)
}

// find is the id of the undeleted work item identifier names among the
// workspace's undeleted projects; domain.ErrIssueNotFound when it names
// none.
func (u *GetIssueByIdentifier) find(ctx context.Context, workspaceID uuid.UUID, identifier string) (uuid.UUID, error) {
	asked, number, ok := domain.ParseIdentifier(identifier)
	if !ok {
		return uuid.UUID{}, domain.ErrIssueNotFound
	}
	projects, err := u.d.Projects.Identifiers(ctx, workspaceID)
	if err != nil {
		return uuid.UUID{}, err
	}
	identifiers := make([]string, len(projects))
	for i, p := range projects {
		identifiers[i] = p.Identifier
	}
	i, ok := domain.MatchIdentifier(asked, identifiers)
	if !ok {
		return uuid.UUID{}, domain.ErrIssueNotFound
	}
	id, found, err := u.d.Issues.IssueByNumber(ctx, projects[i].ID, number)
	switch {
	case err != nil:
		return uuid.UUID{}, err
	case !found:
		return uuid.UUID{}, domain.ErrIssueNotFound
	}
	return id, nil
}
