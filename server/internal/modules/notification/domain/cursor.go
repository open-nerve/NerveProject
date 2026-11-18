package domain

import (
	"encoding/json"
	"errors"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListCursor is a position in the list's order, (created_at DESC, id
// DESC) (spec 2.3): the last row's of a page.
type ListCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

// errCursorShape is a list cursor that is not [created_at, id], or one of
// the zero id.
var errCursorShape = errors.New("the cursor must contain exactly a timestamp and a nonzero id")

// MarshalJSON writes the cursor as [created_at, id], created_at in UTC
// with its nanoseconds, so each position has exactly one spelling.
func (c ListCursor) MarshalJSON() ([]byte, error) {
	if c.ID == (uuid.UUID{}) {
		return nil, errCursorShape
	}
	return json.Marshal([2]string{c.CreatedAt.UTC().Format(time.RFC3339Nano), c.ID.String()})
}

// UnmarshalJSON reads an array of exactly two strings, an RFC 3339 time
// and an id. It also reads what MarshalJSON never writes, such as the time
// at an offset or the zero id; DecodeListCursor refuses those, as they do
// not encode back to the cursor.
func (c *ListCursor) UnmarshalJSON(b []byte) error {
	var parts []string
	if err := json.Unmarshal(b, &parts); err != nil {
		return err
	}
	if len(parts) != 2 {
		return errCursorShape
	}
	at, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return err
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return err
	}
	*c = ListCursor{CreatedAt: at.UTC(), ID: id}
	return nil
}

// EncodeListCursor is the page cursor of c, in the shared envelope.
func EncodeListCursor(c ListCursor) (string, error) { return shared.EncodeCursor(c) }

// DecodeListCursor is the position raw encodes: shared.InvalidCursor, 400
// bad_request, for any shape or spelling EncodeListCursor does not write.
func DecodeListCursor(raw string) (ListCursor, error) {
	var c ListCursor
	if err := shared.DecodeCursor(raw, &c); err != nil {
		return ListCursor{}, err
	}
	return c, nil
}
