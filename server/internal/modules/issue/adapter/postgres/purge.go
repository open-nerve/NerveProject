package postgresadapter

import (
	"context"
	"fmt"
	"math"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// PurgeCommentReactions commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeCommentReactions(ctx context.Context, b app.PurgeBatch) ([]uuid.UUID, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	ids, err := gen.New(s.pool).PurgeCommentReactions(ctx, gen.PurgeCommentReactionsParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("purge comment_reactions: %w", err)
	}
	return ids, nil
}

// PurgeIssueReactions commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeIssueReactions(ctx context.Context, b app.PurgeBatch) ([]uuid.UUID, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	ids, err := gen.New(s.pool).PurgeIssueReactions(ctx, gen.PurgeIssueReactionsParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("purge issue_reactions: %w", err)
	}
	return ids, nil
}

// PurgeIssueMentions commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeIssueMentions(ctx context.Context, b app.PurgeBatch) ([]uuid.UUID, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	ids, err := gen.New(s.pool).PurgeIssueMentions(ctx, gen.PurgeIssueMentionsParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("purge issue_mentions: %w", err)
	}
	return ids, nil
}

// PurgeIssueDescriptionVersions commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeIssueDescriptionVersions(ctx context.Context, b app.PurgeBatch) ([]uuid.UUID, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	ids, err := gen.New(s.pool).PurgeIssueDescriptionVersions(ctx, gen.PurgeIssueDescriptionVersionsParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("purge issue_description_versions: %w", err)
	}
	return ids, nil
}

// PurgeIssueActivities commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeIssueActivities(ctx context.Context, b app.PurgeBatch) ([]uuid.UUID, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	ids, err := gen.New(s.pool).PurgeIssueActivities(ctx, gen.PurgeIssueActivitiesParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("purge issue_activities: %w", err)
	}
	return ids, nil
}

// PurgeIssueComments commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeIssueComments(ctx context.Context, b app.PurgeBatch) ([]uuid.UUID, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	ids, err := gen.New(s.pool).PurgeIssueComments(ctx, gen.PurgeIssueCommentsParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("purge issue_comments: %w", err)
	}
	return ids, nil
}

// PurgeIssueLinks commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeIssueLinks(ctx context.Context, b app.PurgeBatch) ([]uuid.UUID, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	ids, err := gen.New(s.pool).PurgeIssueLinks(ctx, gen.PurgeIssueLinksParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("purge issue_links: %w", err)
	}
	return ids, nil
}

// PurgeIssueRelations commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeIssueRelations(ctx context.Context, b app.PurgeBatch) ([]uuid.UUID, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	ids, err := gen.New(s.pool).PurgeIssueRelations(ctx, gen.PurgeIssueRelationsParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("purge issue_relations: %w", err)
	}
	return ids, nil
}

// PurgeIssueAssignees commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeIssueAssignees(ctx context.Context, b app.PurgeBatch) ([]uuid.UUID, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	ids, err := gen.New(s.pool).PurgeIssueAssignees(ctx, gen.PurgeIssueAssigneesParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("purge issue_assignees: %w", err)
	}
	return ids, nil
}

// PurgeIssueLabels commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeIssueLabels(ctx context.Context, b app.PurgeBatch) ([]uuid.UUID, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	ids, err := gen.New(s.pool).PurgeIssueLabels(ctx, gen.PurgeIssueLabelsParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("purge issue_labels: %w", err)
	}
	return ids, nil
}

// PurgeIssueSubscribers commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeIssueSubscribers(ctx context.Context, b app.PurgeBatch) ([]uuid.UUID, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	ids, err := gen.New(s.pool).PurgeIssueSubscribers(ctx, gen.PurgeIssueSubscribersParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("purge issue_subscribers: %w", err)
	}
	return ids, nil
}

// PurgeDraftIssues commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeDraftIssues(ctx context.Context, b app.PurgeBatch) ([]uuid.UUID, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	ids, err := gen.New(s.pool).PurgeDraftIssues(ctx, gen.PurgeDraftIssuesParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("purge draft_issues: %w", err)
	}
	return ids, nil
}

// PurgeIssues commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeIssues(ctx context.Context, b app.PurgeBatch) ([]uuid.UUID, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	ids, err := gen.New(s.pool).PurgeIssues(ctx, gen.PurgeIssuesParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("purge issues: %w", err)
	}
	return ids, nil
}
