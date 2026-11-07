package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ModuleLinkWriter is the module links' store as their writes use it, in
// the transaction ctx carries: a new link, written by its creator at its
// creation; the change of a link's address and title, by by at its
// UpdatedAt, which empties its metadata when the address changes and keeps
// the metadata stored when it does not (M6 P2 spec R11), answered as
// written; and its soft deletion, by by at at. An address another undeleted
// link of the module has, as it is spelt, is domain.ErrModuleLinkExists.
type ModuleLinkWriter interface {
	InsertModuleLink(ctx context.Context, l domain.ModuleLink) error
	UpdateModuleLink(ctx context.Context, l domain.ModuleLink, by uuid.UUID) (domain.ModuleLink, error)
	DeleteModuleLink(ctx context.Context, id uuid.UUID, at time.Time, by uuid.UUID) error
}

// ModuleLinkLister reads the module's undeleted links, the latest created
// first, then by id descending, in the transaction ctx carries.
type ModuleLinkLister interface {
	ListModuleLinks(ctx context.Context, module uuid.UUID) ([]domain.ModuleLink, error)
}

// ModuleLinkDeps are what the writes of a module's links need.
type ModuleLinkDeps struct {
	Locks Locks
	Links ModuleLinkWriter
	Clock Clock
}

// CreateModuleLink adds a link to a module (createModuleLink).
type CreateModuleLink struct{ d ModuleLinkDeps }

// NewCreateModuleLink returns the use case over d.
func NewCreateModuleLink(d ModuleLinkDeps) *CreateModuleLink { return &CreateModuleLink{d: d} }

// Execute checks in's values, takes the lock and decision of a write on the
// module (Locks, module.write), refuses an archived module, and writes the
// link, its metadata empty, created and last changed by the caller at the
// moment read after the lock. The address is the store's to refuse
// (ModuleLinkWriter).
func (u *CreateModuleLink) Execute(ctx context.Context, moduleID uuid.UUID, in domain.NewModuleLink) (domain.ModuleLink, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.ModuleLink{}, err
	}
	if in, err = domain.CheckNewModuleLink(in); err != nil {
		return domain.ModuleLink{}, err
	}
	var out domain.ModuleLink
	err = u.d.Locks.run(ctx, actor, write{module: &moduleID, action: domain.ActionModuleWrite}, func(ctx context.Context, h held) error {
		if err := writable(h); err != nil {
			return err
		}
		now := u.d.Clock.Now()
		by := actor.UserID
		l := domain.ModuleLink{ID: uuid.NewV7(), WorkspaceID: h.module.WorkspaceID, ProjectID: h.module.ProjectID, ModuleID: h.module.ID,
			URL: in.URL, Title: in.Title, Metadata: shared.LinkMetadata{}, CreatedByID: &by, CreatedAt: now, UpdatedAt: now}
		if err := u.d.Links.InsertModuleLink(ctx, l); err != nil {
			return err
		}
		out = l
		return nil
	})
	return out, err
}

// UpdateModuleLink changes a module's link (updateModuleLink).
type UpdateModuleLink struct{ d ModuleLinkDeps }

// NewUpdateModuleLink returns the use case over d.
func NewUpdateModuleLink(d ModuleLinkDeps) *UpdateModuleLink { return &UpdateModuleLink{d: d} }

// Execute checks what p gives, takes the lock and decision of a write on the
// link (Locks, module.write), refuses an archived module, a change of
// nothing too, and writes what changes, last changed by the caller at the
// moment read after the lock. A change of nothing writes nothing and
// answers the link as the lock read it.
func (u *UpdateModuleLink) Execute(ctx context.Context, id uuid.UUID, p domain.ModuleLinkPatch) (domain.ModuleLink, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.ModuleLink{}, err
	}
	if p, err = domain.CheckModuleLinkPatch(p); err != nil {
		return domain.ModuleLink{}, err
	}
	var out domain.ModuleLink
	err = u.d.Locks.run(ctx, actor, write{link: &id, action: domain.ActionModuleWrite}, func(ctx context.Context, h held) error {
		if err := writable(h); err != nil {
			return err
		}
		next := h.link
		if p.URL != nil {
			next.URL = *p.URL
		}
		if p.SetTitle {
			next.Title = p.Title
		}
		if next.URL == h.link.URL && sameTitle(next.Title, h.link.Title) {
			out = h.link
			return nil
		}
		next.UpdatedAt = u.d.Clock.Now()
		written, err := u.d.Links.UpdateModuleLink(ctx, next, actor.UserID)
		if err != nil {
			return err
		}
		out = written
		return nil
	})
	return out, err
}

// DeleteModuleLink removes a module's link (deleteModuleLink).
type DeleteModuleLink struct{ d ModuleLinkDeps }

// NewDeleteModuleLink returns the use case over d.
func NewDeleteModuleLink(d ModuleLinkDeps) *DeleteModuleLink { return &DeleteModuleLink{d: d} }

// Execute takes the lock and decision of a write on the link (Locks,
// module.write), refuses an archived module, and soft-deletes the link,
// deleted and last changed by the caller at the moment read after the lock.
func (u *DeleteModuleLink) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	return u.d.Locks.run(ctx, actor, write{link: &id, action: domain.ActionModuleWrite}, func(ctx context.Context, h held) error {
		if err := writable(h); err != nil {
			return err
		}
		return u.d.Links.DeleteModuleLink(ctx, id, u.d.Clock.Now(), actor.UserID)
	})
}

// ListModuleLinksDeps are what ListModuleLinks needs.
type ListModuleLinksDeps struct {
	Reads Reads
	Links ModuleLinkLister
}

// ListModuleLinks lists a module's links (listModuleLinks).
type ListModuleLinks struct{ d ListModuleLinksDeps }

// NewListModuleLinks returns the use case over d.
func NewListModuleLinks(d ListModuleLinksDeps) *ListModuleLinks { return &ListModuleLinks{d: d} }

// Execute decides module.read on the module (Reads), archived or not, and
// reads its links in the same snapshot.
func (u *ListModuleLinks) Execute(ctx context.Context, moduleID uuid.UUID) ([]domain.ModuleLink, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	var out []domain.ModuleLink
	err = u.d.Reads.run(ctx, actor, domain.ActionModuleRead, named{module: &moduleID}, func(ctx context.Context, _ held) error {
		links, err := u.d.Links.ListModuleLinks(ctx, moduleID)
		out = links
		return err
	})
	return out, err
}

// writable refuses a write on the links of an archived module (M6 P2 spec
// R2): planning.archived, after the decision.
func writable(h held) error {
	if h.module.ArchivedAt != nil {
		return domain.ErrArchived
	}
	return nil
}

// sameTitle reports whether a and b are the same title, none for both too.
func sameTitle(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
