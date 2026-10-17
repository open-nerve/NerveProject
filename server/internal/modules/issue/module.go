// Package issue is the work items module (M4 design 3.3, 6.2): it owns the
// work items' tables, and soft-deletes a deleted project's rows of them for
// the project module's deletions (its IssueCascade port).
package issue

import (
	"context"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// deleteBatch is the rows a statement of a project's deletion soft-deletes
// at most (M4 design 3.6).
const deleteBatch = 1000

// Cascade is what the project module's deletions ask of the work items:
// project's IssueCascade port (M4 design 3.3). Each method runs in the
// caller's transaction, which ctx carries, writes by and now into the rows
// it deletes, and returns a failure as itself, so the caller's whole write
// rolls back.
type Cascade interface {
	// DeleteProjectIssues soft-deletes the project's work-item-side rows.
	DeleteProjectIssues(ctx context.Context, projectID, by uuid.UUID, now time.Time) error
	// DeleteWorkspaceIssues soft-deletes the work-item-side rows of each
	// of the projects.
	DeleteWorkspaceIssues(ctx context.Context, projectIDs []uuid.UUID, by uuid.UUID, now time.Time) error
}

// Provided are the adapters issue offers the other modules. They depend on
// the pool alone, so bootstrap builds them before any module (M4 design
// 6.6).
type Provided struct {
	IssueCascade Cascade
}

// Provide builds issue's adapters for the other modules.
func Provide(pool *pgxpool.Pool) Provided {
	return Provided{IssueCascade: app.NewCascade(postgresadapter.New(pool), deleteBatch)}
}
