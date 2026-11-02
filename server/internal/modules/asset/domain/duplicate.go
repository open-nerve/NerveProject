package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

type DuplicateAsset struct {
	Purpose Purpose
	Scope   Scope
	Name    *string
}

func DuplicateAssetFields(in DuplicateAsset) []shared.FieldError {
	// Size and MIME come only from the authorized source. This validation
	// applies the same scope/name rules before any source data is read.
	name := "copy"
	if in.Name != nil {
		name = *in.Name
	}
	return NewAssetFields(NewAsset{Purpose: in.Purpose, Scope: in.Scope, Name: name, Size: 1, ContentType: "application/octet-stream"})
}
