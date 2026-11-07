package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// GetModuleDeps are GetModule's ports.
type GetModuleDeps struct {
	Reads Reads
}

// GetModule answers getModule (M6 P2 spec 2.3).
type GetModule struct{ d GetModuleDeps }

// NewGetModule returns the use case over d.
func NewGetModule(d GetModuleDeps) *GetModule { return &GetModule{d: d} }

// Execute reads the module id for the caller (Reads, module.read): an
// archived one too, which the archive's list opens (M6 P2 spec 2.3), with its
// undeleted members in id order.
func (u *GetModule) Execute(ctx context.Context, id uuid.UUID) (domain.Module, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Module{}, err
	}
	var out domain.Module
	err = u.d.Reads.run(ctx, actor, domain.ActionModuleRead, named{module: &id}, func(_ context.Context, h held) error {
		out = h.module
		return nil
	})
	return out, err
}
