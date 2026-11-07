package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// named is what a planning call names (M6 P2 spec R1): the project project;
// or, whichever is not nil, the cycle cycle, the module module or the module
// link link, whose project the call is then on.
type named struct {
	project             uuid.UUID
	cycle, module, link *uuid.UUID
}

// object reports whether n names a cycle, a module or a module link, not a
// project.
func (n named) object() bool { return n.cycle != nil || n.module != nil || n.link != nil }

// notFound is the 404 of what n names: planning.cycle_not_found,
// planning.module_not_found, planning.module_link_not_found, or
// project.not_found for a project.
func (n named) notFound() error {
	switch {
	case n.cycle != nil:
		return domain.ErrCycleNotFound
	case n.module != nil:
		return domain.ErrModuleNotFound
	case n.link != nil:
		return domain.ErrModuleLinkNotFound
	}
	return domain.ErrProjectNotFound
}

// held is what a planning call holds once decided: its project's id and
// facts; what it names below the project — the cycle; the module; the module
// link and its module — each zero when it names none; and the caller's grant.
// A write holds them under its lock (Locks), a read in its snapshot (Reads).
type held struct {
	projectID uuid.UUID
	project   ProjectFacts
	cycle     domain.Cycle
	module    domain.Module
	link      domain.ModuleLink
	grant     shared.Grant
}

// see reads what a call names, n, and its project's facts through facts, and
// authorizes action on them for actor, with the creator of what it names in
// the target (AllowCreator, held.creator). Under a write's lock, locked is
// the project its precheck found, whose facts are read (and so its row
// locked) before what the call names is read again; zero otherwise, when
// what it names is read first, for its project: a link, then its module.
// What the call names that is not there, or no longer in the locked project
// — a link whose module is not, too — and a project that is not there or
// that actor does not see, is the 404 of what the caller named (named.notFound).
// A role the rule refuses is the Authorizer's 403.
func see(ctx context.Context, objects NamedReader, auth shared.Authorizer, actor shared.Actor, action shared.Action, n named,
	locked uuid.UUID, facts func(context.Context, uuid.UUID) (ProjectFacts, bool, error)) (held, error) {
	h := held{projectID: n.project}
	switch {
	case n.object() && locked == (uuid.UUID{}):
		// What the call names first: its project is the call's.
		if err := h.find(ctx, objects, n, uuid.UUID{}); err != nil {
			return held{}, err
		}
	case n.object():
		// Under the lock: the project first, then what the call names read again.
		h.projectID = locked
	}
	f, found, err := facts(ctx, h.projectID)
	switch {
	case err != nil:
		return held{}, err
	case !found:
		return held{}, n.notFound()
	}
	h.project = f
	if n.object() && locked != (uuid.UUID{}) {
		if err := h.find(ctx, objects, n, locked); err != nil {
			return held{}, err
		}
	}
	target := shared.Target{WorkspaceID: f.WorkspaceID, ProjectID: h.projectID}
	if creator := h.creator(n); creator != nil {
		target.CreatorID = *creator
	}
	h.grant, err = auth.Authorize(ctx, actor, action, target)
	switch {
	case errors.Is(err, shared.ErrNotVisible):
		return held{}, n.notFound()
	case err != nil:
		return held{}, err
	}
	return h, nil
}

// find reads what n names into h, and its project as h's: the cycle; the
// module; the module link, then its module. One that is not there, or, when
// project is not zero, not in project, is n's 404; a link's module that is
// not there or not in project is the link's.
func (h *held) find(ctx context.Context, objects NamedReader, n named, project uuid.UUID) error {
	switch {
	case n.cycle != nil:
		c, err := findCycle(ctx, objects, *n.cycle, project)
		if err != nil {
			return err
		}
		h.cycle, h.projectID = c, c.ProjectID
	case n.module != nil:
		m, err := findModule(ctx, objects, *n.module, project, domain.ErrModuleNotFound)
		if err != nil {
			return err
		}
		h.module, h.projectID = m, m.ProjectID
	case n.link != nil:
		l, found, err := objects.GetModuleLink(ctx, *n.link)
		switch {
		case err != nil:
			return err
		case !found:
			return domain.ErrModuleLinkNotFound
		}
		m, err := findModule(ctx, objects, l.ModuleID, project, domain.ErrModuleLinkNotFound)
		if err != nil {
			return err
		}
		h.link, h.module, h.projectID = l, m, m.ProjectID
	}
	return nil
}

// creator is the creator of what n names, whom a rule's AllowCreator lets
// through: the cycle's; the module's, for the module or a link of it, whose
// writes are the module's (module.write); none for a project, nor one not
// known.
func (h held) creator(n named) *uuid.UUID {
	switch {
	case n.cycle != nil:
		return h.cycle.CreatedByID
	case n.module != nil, n.link != nil:
		return h.module.CreatedByID
	}
	return nil
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

// findModule reads the module id; one that is not there, or, when project is
// not zero, not in project, is notFound: planning.module_not_found for a call
// on the module, planning.module_link_not_found for one on a link of it.
func findModule(ctx context.Context, modules ModuleReader, id, project uuid.UUID, notFound error) (domain.Module, error) {
	m, found, err := modules.GetModule(ctx, id)
	switch {
	case err != nil:
		return domain.Module{}, err
	case !found || (project != (uuid.UUID{}) && m.ProjectID != project):
		return domain.Module{}, notFound
	}
	return m, nil
}
