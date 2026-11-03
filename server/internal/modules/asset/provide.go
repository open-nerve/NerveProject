package asset

import (
	"context"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

type PurgeCursor = app.PurgeCursor
type PurgeProgress = app.PurgeProgress
type DraftAssets interface {
	DeleteDraftAssets(context.Context, uuid.UUID, time.Time, uuid.UUID) error
}
type Provided struct {
	Purger      *app.Purger
	DraftAssets DraftAssets
	store       *postgresadapter.Store
}

func Provide(pool *pgxpool.Pool) Provided {
	store := postgresadapter.New(pool)
	return Provided{Purger: app.NewPurger(store), DraftAssets: draftAssets{store}, store: store}
}

type draftAssets struct{ rows app.ScopeDeletions }

func (d draftAssets) DeleteDraftAssets(ctx context.Context, id uuid.UUID, at time.Time, by uuid.UUID) error {
	return d.rows.DeleteScopeRows(ctx, domain.ScopeDraft, id, app.DeletionStamp{At: at, By: &by})
}

type ProtectedParentReference struct{ Schema, Table, Constraint, Column, Parent string }

// Each caller owns a copy. The catalog guard verifies the actual constraints
// and derives the transitive protected ancestor set independently.
func ProtectedParentReferences() []ProtectedParentReference {
	return []ProtectedParentReference{
		{"public", "file_assets", "file_assets_user_fk", "user_id", "users"},
		{"public", "file_assets", "file_assets_workspace_fk", "workspace_id", "workspaces"},
		{"public", "file_assets", "file_assets_project_fk", "project_id", "projects"},
		{"public", "file_assets", "file_assets_issue_fk", "issue_id", "issues"},
		{"public", "file_assets", "file_assets_comment_fk", "comment_id", "issue_comments"},
		{"public", "file_assets", "file_assets_draft_issue_fk", "draft_issue_id", "draft_issues"},
	}
}
