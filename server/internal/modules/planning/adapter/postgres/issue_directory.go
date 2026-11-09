package postgresadapter

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
)

// CycleNames reads the names of the projects' undeleted cycles, archived
// or not, in id order (issue_directory.sql).
func (s *Store) CycleNames(ctx context.Context, projects []uuid.UUID) ([]app.IssueDirectoryName, error) {
	rows, err := s.queries(ctx).CycleNames(ctx, projects)
	if err != nil {
		return nil, fmt.Errorf("read the names of the cycles of %d projects: %w", len(projects), err)
	}
	out := make([]app.IssueDirectoryName, len(rows))
	for i, r := range rows {
		out[i] = app.IssueDirectoryName(r)
	}
	return out, nil
}

// ModuleNames reads the names of the projects' undeleted modules, archived
// or not, in id order (issue_directory.sql).
func (s *Store) ModuleNames(ctx context.Context, projects []uuid.UUID) ([]app.IssueDirectoryName, error) {
	rows, err := s.queries(ctx).ModuleNames(ctx, projects)
	if err != nil {
		return nil, fmt.Errorf("read the names of the modules of %d projects: %w", len(projects), err)
	}
	out := make([]app.IssueDirectoryName, len(rows))
	for i, r := range rows {
		out[i] = app.IssueDirectoryName(r)
	}
	return out, nil
}
