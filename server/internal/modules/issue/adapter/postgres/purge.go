package postgresadapter

import (
	"context"
	"fmt"
	"math"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// PurgeCommentReactions commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeCommentReactions(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	row, err := gen.New(s.pool).PurgeCommentReactions(ctx, gen.PurgeCommentReactionsParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge comment_reactions: %w", err)
	}
	return app.PurgePage{Scanned: int(row.Scanned), Deleted: int(row.Deleted), After: row.AfterID}, nil
}

// PurgeIssueReactions commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeIssueReactions(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	row, err := gen.New(s.pool).PurgeIssueReactions(ctx, gen.PurgeIssueReactionsParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge issue_reactions: %w", err)
	}
	return app.PurgePage{Scanned: int(row.Scanned), Deleted: int(row.Deleted), After: row.AfterID}, nil
}

// PurgeIssueMentions commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeIssueMentions(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	row, err := gen.New(s.pool).PurgeIssueMentions(ctx, gen.PurgeIssueMentionsParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge issue_mentions: %w", err)
	}
	return app.PurgePage{Scanned: int(row.Scanned), Deleted: int(row.Deleted), After: row.AfterID}, nil
}

// PurgeIssueDescriptionVersions commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeIssueDescriptionVersions(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	row, err := gen.New(s.pool).PurgeIssueDescriptionVersions(ctx, gen.PurgeIssueDescriptionVersionsParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge issue_description_versions: %w", err)
	}
	return app.PurgePage{Scanned: int(row.Scanned), Deleted: int(row.Deleted), After: row.AfterID}, nil
}

// PurgeIssueActivities commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeIssueActivities(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	row, err := gen.New(s.pool).PurgeIssueActivities(ctx, gen.PurgeIssueActivitiesParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge issue_activities: %w", err)
	}
	return app.PurgePage{Scanned: int(row.Scanned), Deleted: int(row.Deleted), After: row.AfterID}, nil
}

// PurgeIssueComments scans candidates before deleting, retaining blocked progress.
func (s *Store) PurgeIssueComments(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	q := gen.New(s.pool)
	page, err := postgres.PurgeProtected(ctx, protectedPurgeQueries{
		candidates: func(ctx context.Context, cutoff time.Time, after uuid.UUID, limit int) ([]uuid.UUID, error) {
			return q.PurgeIssueCommentsCandidates(ctx, gen.PurgeIssueCommentsCandidatesParams{Cutoff: &cutoff, AfterID: after, BatchLimit: int32(limit)})
		},
		remove: func(ctx context.Context, cutoff time.Time, ids []uuid.UUID) ([]uuid.UUID, error) {
			return q.DeletePurgeIssueComments(ctx, gen.DeletePurgeIssueCommentsParams{Cutoff: &cutoff, Ids: ids})
		},
	}, s.protected, b.Cutoff, b.After, b.Limit)
	if err != nil {
		return app.PurgePage(page), fmt.Errorf("purge issue_comments: %w", err)
	}
	return app.PurgePage(page), nil
}

// PurgeIssueLinks commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeIssueLinks(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	row, err := gen.New(s.pool).PurgeIssueLinks(ctx, gen.PurgeIssueLinksParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge issue_links: %w", err)
	}
	return app.PurgePage{Scanned: int(row.Scanned), Deleted: int(row.Deleted), After: row.AfterID}, nil
}

// PurgeIssueRelations commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeIssueRelations(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	row, err := gen.New(s.pool).PurgeIssueRelations(ctx, gen.PurgeIssueRelationsParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge issue_relations: %w", err)
	}
	return app.PurgePage{Scanned: int(row.Scanned), Deleted: int(row.Deleted), After: row.AfterID}, nil
}

// PurgeIssueAssignees commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeIssueAssignees(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	row, err := gen.New(s.pool).PurgeIssueAssignees(ctx, gen.PurgeIssueAssigneesParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge issue_assignees: %w", err)
	}
	return app.PurgePage{Scanned: int(row.Scanned), Deleted: int(row.Deleted), After: row.AfterID}, nil
}

// PurgeIssueLabels commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeIssueLabels(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	row, err := gen.New(s.pool).PurgeIssueLabels(ctx, gen.PurgeIssueLabelsParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge issue_labels: %w", err)
	}
	return app.PurgePage{Scanned: int(row.Scanned), Deleted: int(row.Deleted), After: row.AfterID}, nil
}

// PurgeIssueSubscribers commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeIssueSubscribers(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	row, err := gen.New(s.pool).PurgeIssueSubscribers(ctx, gen.PurgeIssueSubscribersParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge issue_subscribers: %w", err)
	}
	return app.PurgePage{Scanned: int(row.Scanned), Deleted: int(row.Deleted), After: row.AfterID}, nil
}

// PurgeModuleIssues commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeModuleIssues(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	row, err := gen.New(s.pool).PurgeModuleIssues(ctx, gen.PurgeModuleIssuesParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge module_issues: %w", err)
	}
	return app.PurgePage{Scanned: int(row.Scanned), Deleted: int(row.Deleted), After: row.AfterID}, nil
}

// PurgeCycleIssues commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeCycleIssues(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	row, err := gen.New(s.pool).PurgeCycleIssues(ctx, gen.PurgeCycleIssuesParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge cycle_issues: %w", err)
	}
	return app.PurgePage{Scanned: int(row.Scanned), Deleted: int(row.Deleted), After: row.AfterID}, nil
}

// PurgeDraftIssues scans candidates before deleting, retaining blocked progress.
func (s *Store) PurgeDraftIssues(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	q := gen.New(s.pool)
	page, err := postgres.PurgeProtected(ctx, protectedPurgeQueries{
		candidates: func(ctx context.Context, cutoff time.Time, after uuid.UUID, limit int) ([]uuid.UUID, error) {
			return q.PurgeDraftIssuesCandidates(ctx, gen.PurgeDraftIssuesCandidatesParams{Cutoff: &cutoff, AfterID: after, BatchLimit: int32(limit)})
		},
		remove: func(ctx context.Context, cutoff time.Time, ids []uuid.UUID) ([]uuid.UUID, error) {
			return q.DeletePurgeDraftIssues(ctx, gen.DeletePurgeDraftIssuesParams{Cutoff: &cutoff, Ids: ids})
		},
	}, s.protected, b.Cutoff, b.After, b.Limit)
	if err != nil {
		return app.PurgePage(page), fmt.Errorf("purge draft_issues: %w", err)
	}
	return app.PurgePage(page), nil
}

// PurgeIssues scans candidates before deleting, retaining blocked progress.
func (s *Store) PurgeIssues(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	q := gen.New(s.pool)
	page, err := postgres.PurgeProtected(ctx, protectedPurgeQueries{
		candidates: func(ctx context.Context, cutoff time.Time, after uuid.UUID, limit int) ([]uuid.UUID, error) {
			return q.PurgeIssuesCandidates(ctx, gen.PurgeIssuesCandidatesParams{Cutoff: &cutoff, AfterID: after, BatchLimit: int32(limit)})
		},
		remove: func(ctx context.Context, cutoff time.Time, ids []uuid.UUID) ([]uuid.UUID, error) {
			return q.DeletePurgeIssues(ctx, gen.DeletePurgeIssuesParams{Cutoff: &cutoff, Ids: ids})
		},
	}, s.protected, b.Cutoff, b.After, b.Limit)
	if err != nil {
		return app.PurgePage(page), fmt.Errorf("purge issues: %w", err)
	}
	return app.PurgePage(page), nil
}
