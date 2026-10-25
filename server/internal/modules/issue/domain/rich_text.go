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
}

// RichText contains safe HTML and the content used by issue writes.
type RichText struct {
	HTML, Text           string
	Identity             ContentIdentity
	HasMention, HasImage bool
}

func (r RichText) EmptyComment() bool { return r.Text == "" && !r.HasMention && !r.HasImage }
func (i ContentIdentity) Equal(other ContentIdentity) bool {
	return i.Text == other.Text && slices.Equal(i.MentionIDs, other.MentionIDs) && slices.Equal(i.Links, other.Links) && slices.Equal(i.Checked, other.Checked)
}
