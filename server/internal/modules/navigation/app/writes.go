package app

import (
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// WriteDeps are what the writes of a view need: its creations, changes and
// deletions (spec 2.4, 2.5).
type WriteDeps struct {
	Locks Locks
	Words Words
	Views ViewWriter
	Clock Clock
}

// Words are each kind's words (P3b spec 2.4): a project's views' are a
// project's list's (the issue module's ListSettings), a workspace's views'
// the workspace list's (its WorkspaceListSettings).
type Words struct {
	Project, Workspace ListSettings
}

// of is the words of the views of kind k.
func (w Words) of(k domain.Kind) ListSettings {
	if k == domain.ProjectView {
		return w.Project
	}
	return w.Workspace
}

// checkNewView is the content n creates as a view of kind k
// (domain.CheckNewView); or every problem of n in one 422 validation_failed
// (spec 2.5): its name's, its description's and its icon's first, in the
// body's order, then those its kind's words find in its filters and its
// display filters. It reads nothing.
func (w Words) checkNewView(k domain.Kind, n domain.NewView) (domain.Content, error) {
	c, problems := domain.CheckNewView(n)
	problems = append(problems, w.of(k).Validate(n.Filters.Map(), n.DisplayFilters)...)
	if len(problems) > 0 {
		return domain.Content{}, shared.Invalid(problems...)
	}
	return c, nil
}

// checkPatch is p as a view of kind k stores it (domain.CheckPatch); or
// every problem of p in one 422 validation_failed (spec 2.5): its name's,
// its description's and its icon's first, in the body's order, then those
// the kind's words find in the filters and the display filters p gives,
// each it does not give left as the view has it. It reads nothing.
func (w Words) checkPatch(k domain.Kind, p domain.Patch) (domain.Patch, error) {
	checked, problems := domain.CheckPatch(p)
	var filters map[string][]string
	if p.Filters != nil {
		filters = p.Filters.Map()
	}
	problems = append(problems, w.of(k).Validate(filters, p.DisplayFilters)...)
	if len(problems) > 0 {
		return domain.Patch{}, shared.Invalid(problems...)
	}
	return checked, nil
}
