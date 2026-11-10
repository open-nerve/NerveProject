package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ModuleLister lists a project's undeleted modules, each with its undeleted
// members in id order, in the transaction ctx carries: the unarchived ones by
// sort_order, created_at, id, or, when archived, the archived ones alone by
// archived_at DESC, id (M6 P2 spec 2.3, R9).
type ModuleLister interface {
	ListModules(ctx context.Context, project uuid.UUID, archived bool) ([]domain.Module, error)
}

// ListModulesDeps are ListModules's ports.
type ListModulesDeps struct {
	Reads   Reads
	Modules ModuleLister
	Groups  StateGroupReader
	Stats   IssueStats
}

// ListModules answers listModules (M6 P2 spec 2.3, R9).
type ListModules struct{ d ListModulesDeps }

// NewListModules returns the use case over d.
func NewListModules(d ListModulesDeps) *ListModules { return &ListModules{d: d} }

// Execute decides module.read on the project (Reads) and lists its modules,
// archived or not as asked, in the store's order, in the same snapshot, each
// with the counts of its work items that the caller sees (held.viewer), read
// once for the whole page, whatever its length, and summed by the groups of
// the project's states (domain.CountsOf; M6 P4 spec R2, R9); an empty list
// for none, which reads nothing more.
func (u *ListModules) Execute(ctx context.Context, projectID uuid.UUID, archived bool) ([]domain.ModuleListItem, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	out := []domain.ModuleListItem{}
	err = u.d.Reads.run(ctx, actor, domain.ActionModuleRead, named{project: projectID}, func(ctx context.Context, h held) error {
		modules, err := u.d.Modules.ListModules(ctx, projectID, archived)
		if err != nil || len(modules) == 0 {
			return err
		}
		ids := make([]uuid.UUID, len(modules))
		for i, m := range modules {
			ids[i] = m.ID
		}
		groups, err := u.d.Groups.StateGroups(ctx, h.projectID)
		if err != nil {
			return err
		}
		counts, err := u.d.Stats.ModuleCounts(ctx, h.projectID, ids, h.viewer(actor))
		if err != nil {
			return err
		}
		live, err := domain.CountsOf(counts, groups)
		if err != nil {
			return err
		}
		for _, m := range modules {
			out = append(out, domain.ModuleListItem{Module: m, Live: live[m.ID]})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
