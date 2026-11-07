package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// held is what a planning call holds once decided: its project's id and
// facts; the cycle it names, zero for a call on a project; and the caller's
// grant. A write holds them under its lock (Locks), a read in its snapshot
// (Reads).
type held struct {
	projectID uuid.UUID
	project   ProjectFacts
	cycle     domain.Cycle
	grant     shared.Grant
}

// see reads what a call names, the cycle cycle or the project project, and
// its project's facts through facts, and authorizes action on them for
// actor, the cycle's creator in the target (AllowCreator). Under a write's
// lock, locked is the project its precheck found, whose facts are read (and
// so its row locked) before the cycle is read again; zero otherwise, when
// the cycle is read first, for its project. A cycle that is not there, or no
// longer in the locked project, and a project that is not there or that
// actor does not see, is the 404 of what the caller named:
// planning.cycle_not_found for a cycle, project.not_found for a project. A
// role the rule refuses is the Authorizer's 403.
func see(ctx context.Context, cycles CycleReader, auth shared.Authorizer, actor shared.Actor, action shared.Action, project uuid.UUID,
	cycle *uuid.UUID, locked uuid.UUID, facts func(context.Context, uuid.UUID) (ProjectFacts, bool, error)) (held, error) {
	notFound := domain.ErrProjectNotFound
	h := held{projectID: project}
	switch {
	case cycle != nil && locked == (uuid.UUID{}):
		// The cycle first: its project is the call's.
		notFound = domain.ErrCycleNotFound
		c, err := findCycle(ctx, cycles, *cycle, uuid.UUID{})
		if err != nil {
			return held{}, err
		}
		h.cycle, h.projectID = c, c.ProjectID
	case cycle != nil:
		// Under the lock: the project first, then the cycle read again.
		notFound, h.projectID = domain.ErrCycleNotFound, locked
	}
	f, found, err := facts(ctx, h.projectID)
	switch {
	case err != nil:
		return held{}, err
	case !found:
		return held{}, notFound
	}
	h.project = f
	if cycle != nil && locked != (uuid.UUID{}) {
		if h.cycle, err = findCycle(ctx, cycles, *cycle, locked); err != nil {
			return held{}, err
		}
	}
	target := shared.Target{WorkspaceID: f.WorkspaceID, ProjectID: h.projectID}
	if h.cycle.CreatedByID != nil {
		target.CreatorID = *h.cycle.CreatedByID
	}
	h.grant, err = auth.Authorize(ctx, actor, action, target)
	switch {
	case errors.Is(err, shared.ErrNotVisible):
		return held{}, notFound
	case err != nil:
		return held{}, err
	}
	return h, nil
}

// findCycle reads the cycle id; one that is not there, or, when project is
// not zero, not in project, is planning.cycle_not_found.
func findCycle(ctx context.Context, cycles CycleReader, id, project uuid.UUID) (domain.Cycle, error) {
	c, found, err := cycles.GetCycle(ctx, id)
	switch {
	case err != nil:
		return domain.Cycle{}, err
	case !found || (project != (uuid.UUID{}) && c.ProjectID != project):
		return domain.Cycle{}, domain.ErrCycleNotFound
	}
	return c, nil
}
