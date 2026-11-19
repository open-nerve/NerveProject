package domain

import (
	"encoding/json"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// IntakeCursor is the last row's position in an intake list's order (M7
// P2b spec 2.3): [order, key, id] in the shared envelope, the key the work
// item's created_at or updated_at (At), in UTC at the microsecond, spelt as
// a list cursor's, or its sequence_id (Seq), as the order says; the id
// the work item's. Its own type, so that the intake list's words stay out
// of ListCursor's (DP-13).
type IntakeCursor struct {
	Order IntakeOrder
	At    time.Time
	Seq   int64
	ID    uuid.UUID
}

// EncodeIntakeCursor writes c in the shared envelope.
func EncodeIntakeCursor(c IntakeCursor) (string, error) {
	value, err := shared.EncodeCursor(c)
	if err != nil {
		return "", shared.InvalidCursor()
	}
	return value, nil
}

// DecodeIntakeCursor reads a cursor EncodeIntakeCursor wrote for order,
// in its one spelling, else the cursor's 400.
func DecodeIntakeCursor(value string, order IntakeOrder) (IntakeCursor, error) {
	var c IntakeCursor
	if err := shared.DecodeCursor(value, &c); err != nil {
		return IntakeCursor{}, err
	}
	if c.Order != order {
		return IntakeCursor{}, shared.InvalidCursor()
	}
	return c, nil
}

// MarshalJSON writes [order, key, id]: an order of the intake list's, a
// number of 1 or more for sequence_id, and a nonzero id.
func (c IntakeCursor) MarshalJSON() ([]byte, error) {
	if c.ID == (uuid.UUID{}) || !IntakeOrderAllowed(c.Order) {
		return nil, shared.InvalidCursor()
	}
	var key any = c.At.UTC().Format(listTimestampLayout)
	if c.Order.Key() == "sequence_id" {
		if c.Seq < 1 {
			return nil, shared.InvalidCursor()
		}
		key = c.Seq
	}
	return json.Marshal([3]any{c.Order, key, c.ID.String()})
}

// UnmarshalJSON reads what MarshalJSON writes; the envelope's re-encoding
// refuses any other spelling of it.
func (c *IntakeCursor) UnmarshalJSON(raw []byte) error {
	var parts []json.RawMessage
	if err := json.Unmarshal(raw, &parts); err != nil || len(parts) != 3 {
		return shared.InvalidCursor()
	}
	var read IntakeCursor
	var idText string
	if json.Unmarshal(parts[0], &read.Order) != nil || !IntakeOrderAllowed(read.Order) || json.Unmarshal(parts[2], &idText) != nil {
		return shared.InvalidCursor()
	}
	id, err := uuid.Parse(idText)
	if err != nil || id == (uuid.UUID{}) {
		return shared.InvalidCursor()
	}
	read.ID = id
	if read.Order.Key() == "sequence_id" {
		if json.Unmarshal(parts[1], &read.Seq) != nil || read.Seq < 1 {
			return shared.InvalidCursor()
		}
	} else {
		var value string
		var ok bool
		if json.Unmarshal(parts[1], &value) != nil {
			return shared.InvalidCursor()
		}
		if read.At, ok = listTimestamp(value); !ok {
			return shared.InvalidCursor()
		}
	}
	*c = read
	return nil
}
