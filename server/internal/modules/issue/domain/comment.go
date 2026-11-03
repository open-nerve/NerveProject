package domain

import (
	"time"
	"unicode/utf8"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CommentReaction is an actor's reaction on a comment.
type CommentReaction struct {
	ID, WorkspaceID, ProjectID, CommentID, ActorID uuid.UUID
	Reaction                                       string
	CreatedAt                                      time.Time
}

// Comment is a work item's comment, with its current reactions.
type Comment struct {
	ID, WorkspaceID, ProjectID, IssueID, ActorID uuid.UUID
	HTML, Text                                   string
	EditedAt                                     *time.Time
	CreatedAt, UpdatedAt                         time.Time
	Reactions                                    []CommentReaction
}

// NewComment names the content and completed preparations claimed on creation.
// Attachment IDs contain ordinary files only; images come from sanitized HTML.
type NewComment struct {
	HTML          string
	AttachmentIDs []uuid.UUID
}

func CheckNewComment(in NewComment) error {
	if err := CheckCommentHTML(in.HTML); err != nil {
		return err
	}
	return invalid(checkIDs("attachment_ids", in.AttachmentIDs, 100)...)
}

// CheckCommentHTML bounds the input before sanitization or any read.
func CheckCommentHTML(html string) error {
	if utf8.RuneCountInString(html) > 500000 {
		return shared.Invalid(shared.FieldError{Field: "comment_html", Code: shared.FieldTooLong, Message: "must be at most 500000 characters"})
	}
	return nil
}

// RequiredComment reports content with no text, mention or image after cleaning.
func RequiredComment() error {
	return shared.Invalid(shared.FieldError{Field: "comment_html", Code: shared.FieldRequired, Message: "must contain text, a mention or an image"})
}
