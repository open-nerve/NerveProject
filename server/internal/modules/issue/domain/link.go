package domain

import (
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Link is a live work-item link and its current preview metadata.
type Link struct {
	ID, WorkspaceID, ProjectID, IssueID uuid.UUID
	URL                                 string
	Title                               *string
	Metadata                            shared.LinkMetadata
	CreatedByID                         *uuid.UUID
	CreatedAt, UpdatedAt                time.Time
}

// NewLink contains the caller's URL and optional title.
type NewLink struct {
	URL   string
	Title *string
}

// LinkPatch distinguishes an omitted title from an explicit null.
type LinkPatch struct {
	URL      *string
	SetTitle bool
	Title    *string
}

// CheckNewLink normalizes only surrounding URL whitespace and an omitted
// HTTP scheme, preserving the exact stored URL and title otherwise.
func CheckNewLink(n NewLink) (NewLink, error) {
	u, problem := shared.CheckLinkURL(n.URL)
	if err := invalid(problem, shared.CheckLinkTitle(n.Title)); err != nil {
		return NewLink{}, err
	}
	n.URL = u
	return n, nil
}

// CheckLinkPatch checks only supplied fields and preserves title presence.
func CheckLinkPatch(p LinkPatch) (LinkPatch, error) {
	var found []*shared.FieldError
	if p.URL != nil {
		u, problem := shared.CheckLinkURL(*p.URL)
		p.URL = &u
		found = append(found, problem)
	}
	if p.SetTitle {
		found = append(found, shared.CheckLinkTitle(p.Title))
	}
	if err := invalid(found...); err != nil {
		return LinkPatch{}, err
	}
	return p, nil
}
