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
}

// ListModules answers listModules (M6 P2 spec 2.3, R9).
type ListModules struct{ d ListModulesDeps }

// NewListModules returns the use case over d.
func NewListModules(d ListModulesDeps) *ListModules { return &ListModules{d: d} }

// Execute decides module.read on the project (Reads) and lists its modules,
// archived or not as asked, in the store's order, in the same snapshot.
func (u *ListModules) Execute(ctx context.Context, projectID uuid.UUID, archived bool) ([]domain.Module, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	var out []domain.Module
	err = u.d.Reads.run(ctx, actor, domain.ActionModuleRead, named{project: projectID}, func(ctx context.Context, _ held) error {
		modules, err := u.d.Modules.ListModules(ctx, projectID, archived)
		out = modules
		return err
	})
	return out, err
}
