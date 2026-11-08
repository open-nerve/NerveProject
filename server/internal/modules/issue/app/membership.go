package app

import (
	"context"
	"fmt"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// membership is what a write changes in the cycle and the modules a work
// item is in, as its activity names them (M6 P3a spec 2.4): the cycle it
// leaves and the one it joins, nil when it is left as it is; the modules it
// joins and those it leaves, each in id order, none for none.
type membership struct {
	cycle         *cycleChange
	modulesJoined []domain.Named
	modulesLeft   []domain.Named
}

// readMembership reads what planning holds about a write's change of the
// work item's cycle, the cycle given, nil when it is left as it is, and of its
// modules, the Diff modules, in the project projectID (M6 P3a spec R1–R3):
// once, under the project's lock the write holds, before its clock, about
// the cycles and the modules together, nothing when neither changes. The
// project's cycles turned off refuse any change of the cycle, leaving one
// too, then its modules turned off any change of the modules (409
// planning.cycles_disabled, planning.modules_disabled, design 3.8); then a
// cycle or a module joined, archived, refuses it (409 planning.archived).
// One joined that is not the project's undeleted cycle or module is
// unknown: the problems answered, for the write's one 422 after its other
// problems, are the cycle's at cycle_id (domain.CycleNotAllowed), then
// each module's at its index in listed, the module_ids the request gives
// (domain.ModuleNotAllowed). Those left are not checked (R3), only named;
// each is the project's undeleted cycle or module, else an error: a
// deletion takes its work items out of it. Modules kept are not asked
// about.
func readMembership(ctx context.Context, planning PlanningTargets, projectID uuid.UUID, cycle *domain.CycleChange, modules domain.Diff,
	listed []uuid.UUID) (membership, []shared.FieldError, error) {
	changesModules := len(modules.Added) > 0 || len(modules.Removed) > 0
	if cycle == nil && !changesModules {
		return membership{}, nil, nil
	}
	var cycles, asked []uuid.UUID
	if cycle != nil {
		for _, id := range []*uuid.UUID{cycle.From, cycle.To} {
			if id != nil {
				cycles = append(cycles, *id)
			}
		}
	}
	if changesModules {
		asked = ascending(slices.Concat(modules.Added, modules.Removed))
	}
	facts, err := planning.Targets(ctx, projectID, ascending(cycles), asked)
	switch {
	case err != nil:
		return membership{}, nil, err
	case cycle != nil && !facts.CyclesEnabled:
		return membership{}, nil, domain.ErrCyclesDisabled
	case changesModules && !facts.ModulesEnabled:
		return membership{}, nil, domain.ErrModulesDisabled
	}
	var m membership
	var problems []shared.FieldError
	if cycle != nil {
		change, unknown, err := namedCycleChange(facts.Cycles, projectID, *cycle)
		if err != nil {
			return membership{}, nil, err
		}
		m.cycle = &change
		if unknown {
			problems = append(problems, domain.CycleNotAllowed())
		}
	}
	for at, id := range listed {
		joined, found := targetOf(facts.Modules, id)
		switch {
		case !slices.Contains(modules.Added, id):
		case !found:
			problems = append(problems, domain.ModuleNotAllowed(at))
		case joined.Archived:
			return membership{}, nil, domain.ErrPlanningArchived
		}
	}
	for _, id := range modules.Added {
		if joined, found := targetOf(facts.Modules, id); found {
			m.modulesJoined = append(m.modulesJoined, domain.Named{ID: id, Name: joined.Name})
		}
	}
	for _, id := range modules.Removed {
		left, found := targetOf(facts.Modules, id)
		if !found {
			return membership{}, nil, fmt.Errorf("work item's module %s is not an undeleted module of project %s", id, projectID)
		}
		m.modulesLeft = append(m.modulesLeft, domain.Named{ID: id, Name: left.Name})
	}
	return m, problems, nil
}

// targetOf is the target id among targets; found is false when it is not
// there.
func targetOf(targets []PlanningTarget, id uuid.UUID) (PlanningTarget, bool) {
	at := slices.IndexFunc(targets, func(t PlanningTarget) bool { return t.ID == id })
	if at < 0 {
		return PlanningTarget{}, false
	}
	return targets[at], true
}

// idsOf are the ids of named, in their order.
func idsOf(named []domain.Named) []uuid.UUID {
	ids := make([]uuid.UUID, len(named))
	for i, n := range named {
		ids[i] = n.ID
	}
	return ids
}
