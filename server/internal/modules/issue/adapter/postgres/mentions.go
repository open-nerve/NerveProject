package postgresadapter

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// MentionIDs reads the live mention set in account order.
func (s *Store) MentionIDs(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error) {
	return s.queries(ctx).MentionIDs(ctx, id)
}

// InsertMentions inserts new live links in target order.
func (s *Store) InsertMentions(ctx context.Context, r app.LinkRows) error {
	return s.queries(ctx).InsertMentions(ctx, gen.InsertMentionsParams{Ids: r.IDs, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, IssueID: r.IssueID, MentionIds: r.Targets, By: &r.By, At: r.At})
}

// RemoveMentions soft deletes only removed live links.
func (s *Store) RemoveMentions(ctx context.Context, r app.UnlinkRows) error {
	return s.queries(ctx).RemoveMentions(ctx, gen.RemoveMentionsParams{IssueID: r.IssueID, MentionIds: r.Targets, By: r.By, At: r.At})
}
