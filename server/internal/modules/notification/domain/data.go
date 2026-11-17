package domain

import "uuid"

// ExcerptLength is how many characters of a comment's stored plain text
// its notification keeps (spec 2.5, R6).
const ExcerptLength = 200

// Data is a notification's snapshot, notifications.data (spec 2.5): the
// work item and the change at the moment it is sent, exactly the two
// objects the table's CHECK allows.
type Data struct {
	Issue  IssueSnapshot  `json:"issue"`
	Change ChangeSnapshot `json:"change"`
}

// IssueSnapshot is the work item as its notification shows it: its id,
// name and number, its project's identifier, and its state's name and
// group.
type IssueSnapshot struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	SequenceID int       `json:"sequence_id"`
	Identifier string    `json:"identifier"`
	StateName  string    `json:"state_name"`
	StateGroup string    `json:"state_group"`
}

// ChangeSnapshot is the change a notification is of: its field and verb,
// and only when it has them, its values, its ids, its comment and the
// comment's excerpt; a key it does not have is not written.
type ChangeSnapshot struct {
	Field     string     `json:"field"`
	Verb      Verb       `json:"verb"`
	OldValue  *string    `json:"old_value,omitempty"`
	NewValue  *string    `json:"new_value,omitempty"`
	OldID     *uuid.UUID `json:"old_id,omitempty"`
	NewID     *uuid.UUID `json:"new_id,omitempty"`
	CommentID *uuid.UUID `json:"comment_id,omitempty"`
	Excerpt   *string    `json:"excerpt,omitempty"`
}

// NewData is the snapshot of the notification of item on the work item
// issue (spec 2.5). text is the stored plain text of item's comment, nil
// when it is not there: a comment's creation, update and mention keep its
// excerpt, its first ExcerptLength characters, in which a mention shows no
// name (R6); a comment's deletion keeps none.
func NewData(issue IssueSnapshot, item Item, text *string) Data {
	change := ChangeSnapshot{Field: item.Field, Verb: item.Verb, OldValue: item.OldValue, NewValue: item.NewValue, OldID: item.OldID,
		NewID: item.NewID, CommentID: item.CommentID}
	if item.CommentID != nil && item.Verb != VerbDeleted && text != nil {
		e := excerpt(*text)
		change.Excerpt = &e
	}
	return Data{Issue: issue, Change: change}
}

// excerpt is the first ExcerptLength characters of text, cut between
// characters, never inside one.
func excerpt(text string) string {
	n := 0
	for at := range text {
		if n == ExcerptLength {
			return text[:at]
		}
		n++
	}
	return text
}
