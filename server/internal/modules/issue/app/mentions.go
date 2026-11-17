package app

import (
	"context"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// commentMentions are, of the accounts ids name, the work item i's
// project's active members, a guest who sees only his own among them
// (descriptionTargets): the mention changes of the comment's event (M7 P1a
// spec 2.2); and the accounts the comment subscribes: the caller and those
// of them who can see i. The project role determines OwnOnly, even for an
// admin of the workspace. These mentions do not create issue_mentions or
// activities.
func commentMentions(ctx context.Context, members ProjectMembers, i domain.Issue, actor uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, []uuid.UUID, error) {
	mentions, visible, err := descriptionTargets(ctx, members, i, ids)
	if err != nil {
		return nil, nil, err
	}
	return mentions, ascending(append(visible, actor)), nil
}

// newlyMentioned are the ids next mentions that old does not: what a
// comment's edit mentions anew, its body before the edit old's mentions.
func newlyMentioned(next, old []uuid.UUID) []uuid.UUID {
	return slices.DeleteFunc(slices.Clone(next), func(id uuid.UUID) bool { return slices.Contains(old, id) })
}
