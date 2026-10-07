package app

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// checkParent reads the label parentID, the parent a write asks for the
// label id of project (uuid.Nil for a new label), under the project's FOR
// NO KEY UPDATE, which every write of the project's labels takes, so that
// the labels stay as read until the commit (M3 design 3.16); and checks it
// (domain.CheckParent), hasChildren saying whether the label has labels
// under it. A parent read for another id than asked is an error.
func checkParent(ctx context.Context, labels LabelFinder, id, project, parentID uuid.UUID, hasChildren bool) error {
	parent, found, err := labels.LabelByID(ctx, parentID)
	if err != nil {
		return err
	}
	var read *domain.Label
	if found {
		if parent.ID != parentID {
			return fmt.Errorf("parent label %s read as %s", parentID, parent.ID)
		}
		read = &parent
	}
	return domain.CheckParent(id, project, read, hasChildren)
}
