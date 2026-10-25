package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// visibleMentionSubscribers includes the caller and each active mention candidate
// who can see this issue. The project role determines OwnOnly, even for an admin
// of the workspace. These mentions do not create issue_mentions or activities.
func visibleMentionSubscribers(ctx context.Context, members ProjectMembers, i domain.Issue, actor uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	targets := []uuid.UUID{actor}
	if len(ids) > 0 {
		candidates, err := members.MentionMembers(ctx, i.ProjectID, ascending(ids))
		if err != nil {
			return nil, err
		}
		for _, m := range candidates {
			if !i.HiddenFrom(m.ID, m.OwnOnly) {
				targets = append(targets, m.ID)
			}
		}
	}
	return ascending(targets), nil
}
