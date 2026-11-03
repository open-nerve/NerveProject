package domain

import (
	"encoding/base64"
	"encoding/json"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type attachmentCursor struct {
	V     int       `json:"v"`
	Kind  string    `json:"kind"`
	ID    uuid.UUID `json:"id"`
	After uuid.UUID `json:"after"`
}

func EncodeAttachmentCursor(kind string, id, after uuid.UUID) (string, error) {
	if (kind != "issue" && kind != "comment" && kind != "draft") || id == uuid.Nil() || after == uuid.Nil() {
		return "", shared.InvalidCursor()
	}
	raw, err := json.Marshal(attachmentCursor{V: 1, Kind: kind, ID: id, After: after})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// Compare the complete encoding, not just decoded fields: alternate UUIDs,
// repeated/unknown JSON keys, whitespace and padded base64 are not our cursor.
func DecodeAttachmentCursor(cursor, kind string, id uuid.UUID) (uuid.UUID, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return uuid.Nil(), shared.InvalidCursor()
	}
	var c attachmentCursor
	if json.Unmarshal(raw, &c) != nil || c.V != 1 || c.Kind != kind || c.ID != id {
		return uuid.Nil(), shared.InvalidCursor()
	}
	again, err := EncodeAttachmentCursor(c.Kind, c.ID, c.After)
	if err != nil || again != cursor {
		return uuid.Nil(), shared.InvalidCursor()
	}
	return c.After, nil
}
