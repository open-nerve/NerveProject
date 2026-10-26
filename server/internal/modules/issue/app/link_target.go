package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// linkTargetCheck rereads a live link and its parent before and under the lock.
func linkTargetCheck(links LinkReader, id uuid.UUID, current *domain.Link) func(context.Context, domain.Issue, shared.Grant) error {
	return func(ctx context.Context, i domain.Issue, _ shared.Grant) error {
		l, found, err := links.GetLink(ctx, id)
		if err != nil {
			return err
		}
		if !found || l.IssueID != i.ID {
			return domain.ErrLinkNotFound
		}
		*current = l
		return nil
	}
}
