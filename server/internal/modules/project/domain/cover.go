package domain

import (
	"strings"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// checkCover accepts one image source and only stable, canonical built-in IDs.
// File ownership and readiness belong to the transaction's binding port.
func checkCover(asset *uuid.UUID, builtin *string) []*shared.FieldError {
	if asset != nil && builtin != nil {
		return []*shared.FieldError{
			{Field: "cover_image_asset_id", Code: shared.FieldNotAllowed, Message: "choose either an uploaded or a built-in cover"},
			{Field: "cover_builtin_id", Code: shared.FieldNotAllowed, Message: "choose either an uploaded or a built-in cover"},
		}
	}
	if builtin != nil && !validCoverID(*builtin) {
		return []*shared.FieldError{{Field: "cover_builtin_id", Code: shared.FieldNotAllowed, Message: "must be cover-01 through cover-29"}}
	}
	return nil
}

func validCoverID(id string) bool {
	return len(id) == 8 && strings.HasPrefix(id, "cover-") &&
		id[6] >= '0' && id[6] <= '2' && id[7] >= '0' && id[7] <= '9' && id != "cover-00"
}

// normalizeCoverPatch clears the other source only when a non-null source
// was explicitly selected. An omitted field or explicit null is independent.
func normalizeCoverPatch(p ProjectPatch) ProjectPatch {
	if p.SetCoverImage && p.CoverImageAssetID != nil {
		p.SetCoverBuiltin, p.CoverBuiltinID = true, nil
	}
	if p.SetCoverBuiltin && p.CoverBuiltinID != nil {
		p.SetCoverImage, p.CoverImageAssetID = true, nil
	}
	return p
}
