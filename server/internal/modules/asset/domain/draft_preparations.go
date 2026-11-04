package domain

import (
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type draftPreparationCursor struct {
	Kind    string    `json:"kind"`
	DraftID uuid.UUID `json:"draft_id"`
	After   uuid.UUID `json:"after"`
}

func EncodeDraftPreparationCursor(draftID, after uuid.UUID) (string, error) {
	if draftID == uuid.Nil() || after == uuid.Nil() {
		return "", shared.InvalidCursor()
	}
	return shared.EncodeCursor(draftPreparationCursor{Kind: "draft_asset_preparations", DraftID: draftID, After: after})
}

func DecodeDraftPreparationCursor(draftID uuid.UUID, cursor *string) (uuid.UUID, error) {
	if cursor == nil {
		return uuid.Nil(), nil
	}
	var value draftPreparationCursor
	if err := shared.DecodeCursor(*cursor, &value); err != nil {
		return uuid.Nil(), err
	}
	if value.Kind != "draft_asset_preparations" || value.DraftID != draftID || draftID == uuid.Nil() || value.After == uuid.Nil() {
		return uuid.Nil(), shared.InvalidCursor()
	}
	return value.After, nil
}
