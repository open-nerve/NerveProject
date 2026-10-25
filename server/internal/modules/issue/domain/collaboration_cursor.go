package domain

import (
	"encoding/json"
	"errors"
	"time"
	"uuid"
)

// CommentCursor is the last comment's position in [created_at, id] order.
type CommentCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

// ActivityCursor is the last activity's position in [created_at, id] order.
type ActivityCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

// DescriptionVersionCursor is the last version's [last_saved_at, id] position.
type DescriptionVersionCursor struct {
	LastSavedAt time.Time
	ID          uuid.UUID
}

// MarshalJSON writes the two-element payload for shared.EncodeCursor in UTC.
func (c CommentCursor) MarshalJSON() ([]byte, error) {
	return marshalCollaborationCursor(c.CreatedAt, c.ID)
}

// UnmarshalJSON reads the two-element payload for shared.DecodeCursor.
func (c *CommentCursor) UnmarshalJSON(b []byte) error {
	at, id, err := unmarshalCollaborationCursor(b)
	if err == nil {
		*c = CommentCursor{CreatedAt: at, ID: id}
	}
	return err
}

// MarshalJSON writes the two-element payload for shared.EncodeCursor in UTC.
func (c ActivityCursor) MarshalJSON() ([]byte, error) {
	return marshalCollaborationCursor(c.CreatedAt, c.ID)
}

// UnmarshalJSON reads the two-element payload for shared.DecodeCursor.
func (c *ActivityCursor) UnmarshalJSON(b []byte) error {
	at, id, err := unmarshalCollaborationCursor(b)
	if err == nil {
		*c = ActivityCursor{CreatedAt: at, ID: id}
	}
	return err
}

// MarshalJSON writes the two-element payload for shared.EncodeCursor in UTC.
func (c DescriptionVersionCursor) MarshalJSON() ([]byte, error) {
	return marshalCollaborationCursor(c.LastSavedAt, c.ID)
}

// UnmarshalJSON reads the two-element payload for shared.DecodeCursor.
func (c *DescriptionVersionCursor) UnmarshalJSON(b []byte) error {
	at, id, err := unmarshalCollaborationCursor(b)
	if err == nil {
		*c = DescriptionVersionCursor{LastSavedAt: at, ID: id}
	}
	return err
}

var errCollaborationCursorShape = errors.New("the cursor must contain exactly a timestamp and a nonzero UUID")

func marshalCollaborationCursor(at time.Time, id uuid.UUID) ([]byte, error) {
	if id == (uuid.UUID{}) {
		return nil, errCollaborationCursorShape
	}
	return json.Marshal([2]string{at.UTC().Format(time.RFC3339Nano), id.String()})
}

func unmarshalCollaborationCursor(b []byte) (time.Time, uuid.UUID, error) {
	var parts []string
	if err := json.Unmarshal(b, &parts); err != nil {
		return time.Time{}, uuid.UUID{}, err
	}
	if len(parts) != 2 {
		return time.Time{}, uuid.UUID{}, errCollaborationCursorShape
	}
	at, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, uuid.UUID{}, err
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return time.Time{}, uuid.UUID{}, err
	}
	if id == (uuid.UUID{}) {
		return time.Time{}, uuid.UUID{}, errCollaborationCursorShape
	}
	return at.UTC(), id, nil
}
