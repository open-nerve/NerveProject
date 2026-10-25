package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// ListStateCatalog reads all undeleted, non-triage states in the selected
// projects, in the transaction carried by ctx.
type ListStateCatalog interface {
	ListStates(context.Context, []uuid.UUID) ([]StateRef, error)
}

// ListLabelCatalog reads all undeleted labels in the selected projects,
// including labels unused by any work item.
type ListLabelCatalog interface {
	ListLabels(context.Context, []uuid.UUID) ([]LabelRef, error)
}

// AssigneeDirectory reads distinct account IDs from undeleted assignee rows
// in the selected projects, including former members and current guests.
type AssigneeDirectory interface {
	AssigneeIDs(context.Context, []uuid.UUID) ([]uuid.UUID, error)
}

// ListState pairs a state with its group and normalized name ordering key.
type ListState struct {
	ID    uuid.UUID
	Group domain.StateGroup
	Key   string
}

// ListName pairs a label or account with its normalized name ordering key.
type ListName struct {
	ID  uuid.UUID
	Key string
}

// ListDirectories contains the complete project scope and name catalogs of
// a list read, independently of the page's filters and returned rows.
type ListDirectories struct {
	Projects          []VisibleProject
	States            []ListState
	Labels, Assignees []ListName
}
