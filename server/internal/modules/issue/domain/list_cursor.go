package domain

import (
	"encoding/json"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListCursor is the last row's [order,key,id] position. Key retains the
// order's JSON type and precision; it is never decoded via interface{}.
type ListCursor struct {
	Order ListOrder
	Key   json.RawMessage
	ID    uuid.UUID
}

// EncodeListCursor writes a typed, normalized payload in the shared envelope.
func EncodeListCursor(c ListCursor) (string, error) {
	value, err := shared.EncodeCursor(c)
	if err != nil {
		return "", shared.InvalidCursor()
	}
	return value, nil
}

// DecodeListCursor accepts only this order and its canonical spelling.
func DecodeListCursor(value string, order ListOrder) (ListCursor, error) {
	var c ListCursor
	if err := shared.DecodeCursor(value, &c); err != nil {
		return ListCursor{}, err
	}
	if c.Order != order {
		return ListCursor{}, shared.InvalidCursor()
	}
	return c, nil
}

// MarshalJSON reconstructs the payload from its order-specific key type so
// equivalent raw numeric/time spellings cannot bypass shared canonical checks.
func (c ListCursor) MarshalJSON() ([]byte, error) {
	if c.ID == (uuid.UUID{}) {
		return nil, shared.InvalidCursor()
	}
	key, err := canonicalListKey(c.Order, c.Key)
	if err != nil {
		return nil, err
	}
	return json.Marshal([3]any{c.Order, key, c.ID.String()})
}

// UnmarshalJSON validates and normalizes before the shared re-encode check.
func (c *ListCursor) UnmarshalJSON(raw []byte) error {
	var parts []json.RawMessage
	if err := json.Unmarshal(raw, &parts); err != nil || len(parts) != 3 {
		return shared.InvalidCursor()
	}
	var order ListOrder
	var idText string
	if err := json.Unmarshal(parts[0], &order); err != nil {
		return shared.InvalidCursor()
	}
	if err := json.Unmarshal(parts[2], &idText); err != nil {
		return shared.InvalidCursor()
	}
	id, err := uuid.Parse(idText)
	if err != nil || id == (uuid.UUID{}) {
		return shared.InvalidCursor()
	}
	key, err := canonicalListKey(order, parts[1])
	if err != nil {
		return err
	}
	*c = ListCursor{Order: order, Key: key, ID: id}
	return nil
}
