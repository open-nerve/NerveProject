package app

import (
	"context"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

type descriptionRelations struct {
	mentions    domain.Diff
	subscribers []uuid.UUID
}

// descriptionTargets keeps all active members as mentions, including hidden guests.
// Only candidates who can see the work item may become subscribers.
func descriptionTargets(ctx context.Context, members ProjectMembers, i domain.Issue, ids []uuid.UUID) ([]uuid.UUID, []uuid.UUID, error) {
	var mentions, subscribers []uuid.UUID
	if len(ids) == 0 {
		return mentions, subscribers, nil
	}
	candidates, err := members.MentionMembers(ctx, i.ProjectID, ascending(ids))
	if err != nil {
		return nil, nil, err
	}
	for _, m := range candidates {
		mentions = append(mentions, m.ID)
		if !i.HiddenFrom(m.ID, m.OwnOnly) {
			subscribers = append(subscribers, m.ID)
		}
	}
	return ascending(mentions), ascending(subscribers), nil
}

func prepareDescription(ctx context.Context, store MentionStore, members ProjectMembers, i domain.Issue, c *domain.DescriptionChange) (descriptionRelations, error) {
	var r descriptionRelations
	if c == nil || !c.Write {
		return r, nil
	}
	targets, subscribers, err := descriptionTargets(ctx, members, i, c.Content.Identity.MentionIDs)
	if err != nil {
		return r, err
	}
	old, err := store.MentionIDs(ctx, i.ID)
	if err != nil {
		return r, err
	}
	r.subscribers = subscribers
	for _, id := range targets {
		if !slices.Contains(old, id) {
			r.mentions.Added = append(r.mentions.Added, id)
		}
	}
	for _, id := range old {
		if !slices.Contains(targets, id) {
			r.mentions.Removed = append(r.mentions.Removed, id)
		}
	}
	return r, nil
}

func writeMentions(ctx context.Context, store MentionStore, s stamp, d domain.Diff) error {
	if len(d.Removed) > 0 {
		if err := store.RemoveMentions(ctx, s.unlinks(d.Removed)); err != nil {
			return err
		}
	}
	if len(d.Added) > 0 {
		return store.InsertMentions(ctx, s.links(d.Added))
	}
	return nil
}

func descriptionVersion(s stamp, clean domain.RichText) domain.DescriptionVersion {
	return domain.DescriptionVersion{ID: uuid.NewV7(), WorkspaceID: s.workspace, ProjectID: s.project, IssueID: s.issue,
		OwnedByID: s.by, CreatedByID: s.by, UpdatedByID: s.by, HTML: clean.HTML, Text: clean.Text,
		LastSavedAt: s.at, CreatedAt: s.at, UpdatedAt: s.at}
}

func saveDescriptionVersion(ctx context.Context, store DescriptionVersions, s stamp, clean domain.RichText) error {
	v, found, err := store.LatestDescriptionVersion(ctx, s.issue)
	if err != nil {
		return err
	}
	if found && domain.CanMergeDescriptionVersion(v, s.by, s.at) {
		v.HTML, v.Text, v.LastSavedAt, v.UpdatedAt, v.UpdatedByID = clean.HTML, clean.Text, s.at, s.at, s.by
		return store.UpdateDescriptionVersion(ctx, v)
	}
	return store.InsertDescriptionVersion(ctx, descriptionVersion(s, clean))
}

// writeUpdateActivities preserves the field order around the mergeable description.
func writeUpdateActivities(ctx context.Context, store DescriptionActivityStore, s stamp, activity domain.Update) error {
	if activity.Description == nil {
		return store.InsertActivities(ctx, s.activity(domain.UpdateActivity(activity)))
	}
	if activity.Name != nil {
		if err := store.InsertActivities(ctx, s.activity(domain.UpdateActivity(domain.Update{Name: activity.Name}))); err != nil {
			return err
		}
		activity.Name = nil
	}
	if err := writeDescriptionActivity(ctx, store, s, *activity.Description); err != nil {
		return err
	}
	activity.Description = nil
	rows := domain.UpdateActivity(activity)
	if len(rows) > 0 {
		return store.InsertActivities(ctx, s.activity(rows))
	}
	return nil
}
