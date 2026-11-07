package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ModuleArchiver writes a module's archive under its project's lock: its
// archived_at, and its last change; nothing else of it.
type ModuleArchiver interface {
	SetModuleArchived(ctx context.Context, m domain.Module) error
}

// ArchiveModuleDeps are ArchiveModule's ports.
type ArchiveModuleDeps struct {
	Locks   Locks
	Modules ModuleArchiver
	Clock   Clock
}

// ArchiveModule answers archiveModule, or unarchiveModule (M6 P2 spec 2.3,
// R3).
type ArchiveModule struct {
	d       ArchiveModuleDeps
	archive bool
}

// NewArchiveModule returns archiveModule's use case over d.
func NewArchiveModule(d ArchiveModuleDeps) *ArchiveModule { return &ArchiveModule{d: d, archive: true} }

// NewUnarchiveModule returns unarchiveModule's use case over d.
func NewUnarchiveModule(d ArchiveModuleDeps) *ArchiveModule { return &ArchiveModule{d: d} }

// Execute takes the lock path on the module id (Locks, module.write); then
// under the lock: a module already as asked answers as the lock read it and
// writes nothing (R3); one to archive whose status is neither completed nor
// cancelled is refused (planning.module_not_closed), while one to unarchive
// is not, whatever its status; then, at the moment read then, its
// archived_at is that moment, or none, and its last change the caller's at
// that moment. The answer is the module as stored.
func (u *ArchiveModule) Execute(ctx context.Context, id uuid.UUID) (domain.Module, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Module{}, err
	}
	var out domain.Module
	err = u.d.Locks.run(ctx, actor, write{module: &id, action: domain.ActionModuleWrite}, func(ctx context.Context, h held) error {
		m := h.module
		switch {
		case (m.ArchivedAt != nil) == u.archive:
			out = m
			return nil
		case u.archive && !m.Status.Closed():
			return domain.ErrModuleNotClosed
		}
		now, by := u.d.Clock.Now(), actor.UserID
		m.ArchivedAt = nil
		if u.archive {
			m.ArchivedAt = &now
		}
		m.UpdatedAt, m.UpdatedByID = now, &by
		if err := u.d.Modules.SetModuleArchived(ctx, m); err != nil {
			return err
		}
		out = m
		return nil
	})
	return out, err
}
