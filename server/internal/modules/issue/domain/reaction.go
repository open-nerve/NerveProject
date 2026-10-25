package domain

import (
	"regexp"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// IssueReaction is an actor's reaction on a work item.
type IssueReaction struct {
	ID, WorkspaceID, ProjectID, IssueID, ActorID uuid.UUID
	Reaction                                     string
	CreatedAt                                    time.Time
}

var reactionPattern = regexp.MustCompile(`^[0-9]{1,7}(-[0-9]{1,7}){0,15}$`)

// CheckReaction validates a decimal reaction code without an emoji registry.
func CheckReaction(reaction string) error {
	if len(reaction) > 128 || !reactionPattern.MatchString(reaction) {
		return shared.Invalid(shared.FieldError{Field: "reaction", Code: shared.FieldInvalidFormat, Message: "must contain 1 to 16 decimal segments of 1 to 7 digits, at most 128 characters"})
	}
	return nil
}
