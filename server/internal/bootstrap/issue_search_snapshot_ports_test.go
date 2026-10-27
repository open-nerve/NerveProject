package bootstrap

import (
	"context"
	"time"
	"uuid"

	issuepg "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

type snapshotSearchProjects struct{ snapshotListProjects }

func (d snapshotSearchProjects) Identifiers(ctx context.Context, wid uuid.UUID) ([]issueapp.ProjectIdentifier, error) {
	return listObserved(ctx, d.p, "identifiers", func() ([]issueapp.ProjectIdentifier, error) { return d.issueProjects.Identifiers(ctx, wid) })
}

type snapshotSearchStore struct {
	*issuepg.Store
	p *listSnapshotProbe
}

func (s snapshotSearchStore) GetIssue(ctx context.Context, id uuid.UUID) (domain.Issue, bool, error) {
	row, found, err := s.Store.GetIssue(ctx, id)
	if err == nil {
		err = s.p.after(ctx, "source")
	}
	return row, found, err
}
func (s snapshotSearchStore) GetIssues(ctx context.Context, ids []uuid.UUID) ([]domain.Issue, error) {
	return listObserved(ctx, s.p, "parents", func() ([]domain.Issue, error) { return s.Store.GetIssues(ctx, ids) })
}
func (s snapshotSearchStore) SubIssueCounts(ctx context.Context, actor uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]issueapp.SubIssueCount, error) {
	return listObserved(ctx, s.p, "counts", func() (map[uuid.UUID]issueapp.SubIssueCount, error) { return s.Store.SubIssueCounts(ctx, actor, ids) })
}
func (s snapshotSearchStore) Search(ctx context.Context, in issueapp.SearchRead) ([]domain.Issue, error) {
	return listObserved(ctx, s.p, "rows", func() ([]domain.Issue, error) { return s.Store.Search(ctx, in) })
}
func snapshotSearchUseCase(w issueWorld, p *listSnapshotProbe) *issueapp.SearchIssues {
	projects, workspaces := project.Provide(w.pool), workspace.Provide(w.pool)
	dirs := snapshotSearchProjects{snapshotListProjects{issueProjects{projects: projects.IssueProjects}, p}}
	store := snapshotSearchStore{issuepg.New(w.pool), p}
	auth := snapshotListAuth{authorizerOn(w.pool), p}
	return issueapp.NewSearchIssues(issueapp.SearchIssuesDeps{
		ReadTx: postgres.NewTxManager(w.pool, 3*time.Second, postgres.ReadOnlySnapshot()), Authorizer: auth,
		Workspaces: snapshotListWorkspace{issueWorkspaces{directory: workspaces.WorkspaceDirectory}, p},
		Projects:   dirs, Identifiers: dirs, Issues: store, Locks: issueapp.NewLocks(dirs, store, auth, nil, nil), Rows: store,
	})
}
