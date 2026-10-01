package domain

import (
	"fmt"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ProjectPatch is what updateProject changes (M3 design 5.2): every field
// of the project the caller may write, each nil when it is not given and
// keeps its value. The lead and the default assignee can be cleared, so
// each has a flag: set, it changes to the id, or to none when the id is
// nil.
type ProjectPatch struct {
	Name                 *string
	Description          *string
	Identifier           *string
	Network              *Network
	SetLead              bool
	LeadID               *uuid.UUID
	SetDefaultAssignee   bool
	DefaultAssigneeID    *uuid.UUID
	CycleView            *bool
	ModuleView           *bool
	IssueViewsView       *bool
	IntakeView           *bool
	GuestViewAllFeatures *bool
	ArchiveIn            *int
	LogoProps            *LogoProps
	Timezone             *string
}

// MaxArchiveIn is the most months archive_in takes: Plane's validators,
// which the column's CHECK holds too (M3 design 4.6).
const MaxArchiveIn = 12

// CheckProjectPatch checks the fields p gives, by createProject's rules
// (CheckNewProject), and archive_in from 0 to MaxArchiveIn; it returns p as
// it is stored, the identifier in upper case. Every field with a problem is
// reported at once, in one 422 validation_failed. Whether the lead and the
// default assignee may be them is the use case's, under its lock (M3 design
// 3.19).
func CheckProjectPatch(p ProjectPatch) (ProjectPatch, error) {
	var found []*shared.FieldError
	if p.Name != nil {
		found = append(found, checkName(*p.Name))
	}
	if p.Identifier != nil {
		identifier := Identifier(*p.Identifier)
		p.Identifier = &identifier
		found = append(found, checkIdentifier(identifier))
	}
	if p.Description != nil {
		found = append(found, checkText("description", *p.Description))
	}
	if p.Network != nil {
		found = append(found, checkNetwork(*p.Network))
	}
	if p.ArchiveIn != nil && (*p.ArchiveIn < 0 || *p.ArchiveIn > MaxArchiveIn) {
		found = append(found, &shared.FieldError{Field: "archive_in", Code: shared.FieldOutOfRange,
			Message: fmt.Sprintf("must be between 0 and %d", MaxArchiveIn)})
	}
	found = append(found, checkTimezone(p.Timezone))
	if p.LogoProps != nil {
		found = append(found, checkLogoProps(*p.LogoProps)...)
	}
	if err := invalid(found...); err != nil {
		return ProjectPatch{}, err
	}
	return p, nil
}

// assignableRoles are the project roles of the members a project's lead and
// default assignee are chosen from on update: not its guests (Plane
// core/components/project/member-select.tsx:32-42; M3 design 3.19).
var assignableRoles = []shared.Role{shared.RoleAdmin, shared.RoleMember}

// CanAssign reports whether an active member of a project, of project role
// role, may be its lead or its default assignee. The set is named, not a
// bound.
func CanAssign(role shared.Role) bool {
	return slices.Contains(assignableRoles, role)
}
