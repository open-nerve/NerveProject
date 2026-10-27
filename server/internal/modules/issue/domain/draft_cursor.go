package domain

import (
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DraftCursor is a position in descending [created_at, id] order.
type DraftCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

// MarshalJSON uses the collaboration lists' UTC timestamp and nonzero id shape.
func (c DraftCursor) MarshalJSON() ([]byte, error) {
	return marshalCollaborationCursor(c.CreatedAt, c.ID)
}

// UnmarshalJSON shares the collaboration lists' two-element shape validation.
func (c *DraftCursor) UnmarshalJSON(b []byte) error {
	at, id, err := unmarshalCollaborationCursor(b)
	if err == nil {
		*c = DraftCursor{CreatedAt: at, ID: id}
	}
	return err
}

// EncodeDraftCursor encodes a position in the shared versioned envelope.
func EncodeDraftCursor(c DraftCursor) (string, error) { return shared.EncodeCursor(c) }

// DecodeDraftCursor rejects any shape or spelling the canonical encoder cannot write.
func DecodeDraftCursor(raw string) (DraftCursor, error) {
	var c DraftCursor
	if err := shared.DecodeCursor(raw, &c); err != nil {
		return DraftCursor{}, err
	}
	return c, nil
}
