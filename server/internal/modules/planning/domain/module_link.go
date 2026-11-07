package domain

import (
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ModuleLink is an undeleted link of a module (M6 P2 spec 2.3): its address,
// its title, none or empty too, and its preview metadata, an empty object
// until the crawl writes it.
type ModuleLink struct {
	ID, WorkspaceID, ProjectID, ModuleID uuid.UUID
	URL                                  string
	Title                                *string
	Metadata                             shared.LinkMetadata
	CreatedByID                          *uuid.UUID
	CreatedAt, UpdatedAt                 time.Time
}

// NewModuleLink is a link as a creation gives it: its address, and its
// title, nil for none.
type NewModuleLink struct {
	URL   string
	Title *string
}

// ModuleLinkPatch is a change of a link: URL nil when it is not given;
// SetTitle tells whether the title is given, Title its value, nil to clear
// it. The title may be given alone (M6 P2 spec 2.3).
type ModuleLinkPatch struct {
	URL      *string
	SetTitle bool
	Title    *string
}

// CheckNewModuleLink is n with its address normalized, or the problems of
// its address and title, as validation_failed, by a work item's link's rules
// (M6 design 3.2): shared.CheckLinkURL, shared.CheckLinkTitle.
func CheckNewModuleLink(n NewModuleLink) (NewModuleLink, error) {
	u, problem := shared.CheckLinkURL(n.URL)
	if err := invalid(problem, shared.CheckLinkTitle(n.Title)); err != nil {
		return NewModuleLink{}, err
	}
	n.URL = u
	return n, nil
}

// CheckModuleLinkPatch is p with its address normalized, or the problems of
// the address and the title it gives, as validation_failed, by a creation's
// rules; what it does not give is not looked at.
func CheckModuleLinkPatch(p ModuleLinkPatch) (ModuleLinkPatch, error) {
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
		return ModuleLinkPatch{}, err
	}
	return p, nil
}
