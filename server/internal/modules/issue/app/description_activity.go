package app

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// writeDescriptionActivity writes only the description, after any title row and before later fields.
func writeDescriptionActivity(ctx context.Context, store DescriptionActivityStore, s stamp, change domain.Changed[string]) error {
	latest, found, err := store.LatestActivity(ctx, s.issue)
	if err != nil {
		return err
	}
	if found && latest.ActorID != nil && *latest.ActorID == s.by && latest.Activity.Field != nil && *latest.Activity.Field == "description" {
		return store.MergeDescriptionActivity(ctx, DescriptionActivityMerge{ID: latest.ID, By: s.by, NewHTML: change.New, At: s.at})
	}
	return store.InsertActivities(ctx, s.activity(domain.UpdateActivity(domain.Update{Description: &change})))
}
