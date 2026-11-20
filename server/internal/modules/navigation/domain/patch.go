package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// Patch is a change of a view (updateView's body, spec 2.4): each
// top-level field nil when it is not given; one given replaces that field
// of the view whole, never merged into it.
type Patch struct {
	Name, Description *string
	LogoProps         *shared.LogoProps
	Filters           *Filters
	DisplayFilters    *shared.ListDisplayFilters
	DisplayProperties *shared.ListDisplayProperties
}

// CheckPatch is p as stored, its name without its surrounding white space;
// or the problems of the name, description and icon it gives, by
// CheckNewView's rules and in its order (checkContent). The use case
// answers them in one 422 with its view's kind's words after them, once
// it has read the view and the decision has let the caller change it
// (spec 2.5).
func CheckPatch(p Patch) (Patch, []shared.FieldError) {
	name, problems := checkContent(p.Name, p.Description, p.LogoProps)
	if problems != nil {
		return Patch{}, problems
	}
	p.Name = name
	return p, nil
}

// Apply is c with each field p gives in its place, and whether that changes
// it (Content.Equal): a field given as it is changes nothing, and {}
// nothing at all (spec 2.4). The use case applies a checked patch to the
// view it reads under its lock, and writes only a change.
func (p Patch) Apply(c Content) (Content, bool) {
	next := c
	if p.Name != nil {
		next.Name = *p.Name
	}
	if p.Description != nil {
		next.Description = *p.Description
	}
	if p.LogoProps != nil {
		next.LogoProps = *p.LogoProps
	}
	if p.Filters != nil {
		next.Filters = *p.Filters
	}
	if p.DisplayFilters != nil {
		next.DisplayFilters = *p.DisplayFilters
	}
	if p.DisplayProperties != nil {
		next.DisplayProperties = *p.DisplayProperties
	}
	return next, !next.Equal(c)
}
