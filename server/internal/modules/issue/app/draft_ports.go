package app

import (
	"context"
	"encoding/json"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// DraftReader reads live drafts before workspace and ownership decisions.
type DraftReader interface {
	GetDraft(context.Context, uuid.UUID) (domain.Draft, bool, error)
	LockDraft(context.Context, uuid.UUID) (domain.Draft, bool, error)
}

// DraftListRead scopes a personal list to the workspace's undeleted projects.
// ProjectIDs is the complete undeleted set, including archived projects, rather
// than the actor's project membership directory. Limit includes any lookahead.
type DraftListRead struct {
	WorkspaceID, ActorID uuid.UUID
	ProjectIDs           []uuid.UUID
	After                *domain.DraftCursor
	Limit                int
}

// DraftLister returns the page and its cursor-independent total in the read snapshot.
type DraftLister interface {
	ListDrafts(context.Context, DraftListRead) ([]domain.Draft, int64, error)
}

// DraftWrite replaces the complete stored payload and optional project.
type DraftWrite struct {
	ID, WorkspaceID, By uuid.UUID
	ProjectID           *uuid.UUID
	Payload             json.RawMessage
	At                  time.Time
}

// DraftDeletion stamps a live draft with the request's actor and time.
type DraftDeletion struct {
	ID, By uuid.UUID
	At     time.Time
}

// DraftWriter persists drafts with no issue creation side effects.
type DraftWriter interface {
	InsertDraft(context.Context, DraftWrite) (domain.Draft, error)
	UpdateDraft(context.Context, DraftWrite) (domain.Draft, error)
	DeleteDraft(context.Context, DraftDeletion) error
}
