package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ModuleUpdater is the modules' store as a change uses it, in the
// transaction ctx carries: the module as changed, whose name another
// undeleted module of the project has, in any case, is
// domain.ErrModuleNameTaken (M6 P2 spec R8) — its own row is no other, so
// its own name in another case is not; the soft deletion of the module's
// rows of the accounts it no longer has as members, at at by by; and its new
// members' rows.
type ModuleUpdater interface {
	UpdateModule(ctx context.Context, m domain.Module) error
	RemoveModuleMembers(ctx context.Context, module uuid.UUID, accounts []uuid.UUID, at time.Time, by uuid.UUID) error
	InsertModuleMembers(ctx context.Context, r MemberRows) error
}

// UpdateModuleDeps are what UpdateModule needs.
type UpdateModuleDeps struct {
	Locks   Locks
	Roles   RoleReader
	Modules ModuleUpdater
	Clock   Clock
}

// UpdateModule changes a module (updateModule).
type UpdateModule struct{ d UpdateModuleDeps }

// NewUpdateModule returns the use case over d.
func NewUpdateModule(d UpdateModuleDeps) *UpdateModule { return &UpdateModule{d: d} }

// Execute checks what p gives, before anything is read (CheckModulePatch);
// takes the lock path on the module id (Locks, module.write); then under the
// lock, in this order (M6 P2 spec R2): an archived module refuses any change,
// one of nothing too (planning.archived); the project's roles of those p
// makes anew (Newcomers), in one read when there is one — a lead or a member
// the module keeps is not looked at, even one who has left the project
// (R5); p applied to the module, every problem of its merged dates and of
// its newcomers at once (R4, R5); a change of nothing, member_ids naming the
// module's members in any order too, writes nothing and answers the module
// as the lock read it. Else, at the moment read then and by the caller, the
// module is written, last changed by them, the rows of the members it
// removes soft-deleted and rows written for those it adds, in account order,
// each with an id of its own; a member it keeps keeps its row. The name is
// the store's to refuse (ModuleUpdater). The answer is the module as stored,
// its members in id order.
func (u *UpdateModule) Execute(ctx context.Context, id uuid.UUID, p domain.ModulePatch) (domain.Module, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Module{}, err
	}
	if p, err = domain.CheckModulePatch(p); err != nil {
		return domain.Module{}, err
	}
	var out domain.Module
	err = u.d.Locks.run(ctx, actor, write{module: &id, action: domain.ActionModuleWrite}, func(ctx context.Context, h held) error {
		if err := writable(h); err != nil {
			return err
		}
		var roles map[uuid.UUID]shared.Role
		if newcomers := p.Newcomers(h.module); len(newcomers) > 0 {
			read, err := u.d.Roles.Roles(ctx, h.projectID, newcomers)
			if err != nil {
				return err
			}
			roles = read
		}
		next, changed, err := p.Apply(h.module, roles)
		switch {
		case err != nil:
			return err
		case !changed:
			out = h.module
			return nil
		}
		now, by := u.d.Clock.Now(), actor.UserID
		next.UpdatedAt, next.UpdatedByID = now, &by
		if err := u.d.Modules.UpdateModule(ctx, next); err != nil {
			return err
		}
		diff := domain.DiffMembers(h.module.MemberIDs, next.MemberIDs)
		if len(diff.Removed) > 0 {
			if err := u.d.Modules.RemoveModuleMembers(ctx, id, diff.Removed, now, by); err != nil {
				return err
			}
		}
		if len(diff.Added) > 0 {
			if err := u.d.Modules.InsertModuleMembers(ctx, memberRows(next, diff.Added, by, now)); err != nil {
				return err
			}
		}
		out = next
		return nil
	})
	return out, err
}
