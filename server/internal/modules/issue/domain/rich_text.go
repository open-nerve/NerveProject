package domain

import (
	"slices"
	"uuid"
)

// ContentIdentity excludes formatting and generated block identifiers.
type ContentIdentity struct {
	Text       string
	MentionIDs []uuid.UUID
	Links      []string
	Checked    []bool
	Images     []string
}

// RichText contains safe HTML and the content used by issue writes.
type RichText struct {
	HTML, Text           string
	Identity             ContentIdentity
	HasMention, HasImage bool
}

func (r RichText) EmptyComment() bool { return r.Text == "" && !r.HasMention && !r.HasImage }
func (i ContentIdentity) Equal(other ContentIdentity) bool {
	return i.Text == other.Text && slices.Equal(i.MentionIDs, other.MentionIDs) &&
		slices.Equal(i.Links, other.Links) && slices.Equal(i.Checked, other.Checked) && slices.Equal(i.Images, other.Images)
}

// AssetImageIDs selects canonical UUID references in content order, retaining
// duplicates. Non-asset sources still participate in the content's identity.
func (i ContentIdentity) AssetImageIDs() []uuid.UUID {
	var ids []uuid.UUID
	for _, source := range i.Images {
		if id, err := uuid.Parse(source); err == nil && id.String() == source {
			ids = append(ids, id)
		}
	}
	return ids
}
