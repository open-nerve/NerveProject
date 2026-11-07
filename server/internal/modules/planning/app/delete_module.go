package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ModuleDeleter deletes a module under its project's lock, in the
// transaction ctx carries (M6 P2 spec 2.4): the module; its undeleted
// members' rows; its undeleted links. Each is one statement, the members'
// and the links' locking their rows in id order before writing them
// (convention 5); each row deleted and last changed by by at at, and one
// deleted before keeps its deletion.
type ModuleDeleter interface {
	DeleteModule(ctx context.Context, id uuid.UUID, at time.Time, by uuid.UUID) error
	DeleteModuleMemberRows(ctx context.Context, module uuid.UUID, at time.Time, by uuid.UUID) error
	DeleteModuleLinkRows(ctx context.Context, module uuid.UUID, at time.Time, by uuid.UUID) error
}

// DeleteModuleDeps are DeleteModule's ports.
type DeleteModuleDeps struct {
	Locks   Locks
	Modules ModuleDeleter
	Clock   Clock
}

// DeleteModule answers deleteModule (M6 P2 spec 2.3, 2.4).
type DeleteModule struct{ d DeleteModuleDeps }

// NewDeleteModule returns the use case over d.
func NewDeleteModule(d DeleteModuleDeps) *DeleteModule { return &DeleteModule{d: d} }

// Execute takes the lock path on the module id deciding module.delete,
// which its project's admins pass and its creator too (AllowCreator); then
// under the lock, the clock, and the module, its members' rows and its
// links deleted and last changed by the caller at that moment, in the
// write's transaction (R15), whatever its status and archive. P2 publishes
// no event.
func (u *DeleteModule) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	return u.d.Locks.run(ctx, actor, write{module: &id, action: domain.ActionModuleDelete}, func(ctx context.Context, _ held) error {
		at, by := u.d.Clock.Now(), actor.UserID
		if err := u.d.Modules.DeleteModule(ctx, id, at, by); err != nil {
			return err
		}
		if err := u.d.Modules.DeleteModuleMemberRows(ctx, id, at, by); err != nil {
			return err
		}
		return u.d.Modules.DeleteModuleLinkRows(ctx, id, at, by)
	})
}
