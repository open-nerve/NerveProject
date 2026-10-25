package app

import (
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CommentWrite is a new comment and the account and moment writing it.
type CommentWrite struct {
	Comment domain.Comment
	By      uuid.UUID
	At      time.Time
}

// MemberVisibility is an active project member's role and guest restriction.
type MemberVisibility struct {
	ID      uuid.UUID
	Role    shared.Role
	OwnOnly bool
}

// CommentPage is comments in ascending creation order, and the next position.
type CommentPage struct {
	Data       []domain.Comment
	NextCursor string
}

// CommentDeletion identifies the comment and its deletion audit stamp.
type CommentDeletion struct {
	ID, By uuid.UUID
	At     time.Time
}
