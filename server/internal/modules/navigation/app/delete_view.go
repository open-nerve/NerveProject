package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DeleteView deletes a view of either kind by its id (deleteView).
type DeleteView struct{ d WriteDeps }

// NewDeleteView returns the use case over d.
func NewDeleteView(d WriteDeps) *DeleteView { return &DeleteView{d: d} }

// Execute soft-deletes the view id (P3b spec 2.4, 2.5), through its locks
// and decision (Locks.run): the view read without a lock and its kind's
// deletion decided on it, a 404 navigation.view_not_found for every view
// the caller cannot have, then 403 for a role the rule refuses and for
// anyone but its owner or its admin, Grant.ProjectAdmin for a project's
// view, the workspace's admin for a workspace's; then the locks, the view
// read again under them and the decision again; then 409 while its
// project's views are turned off. The view is deleted, and last changed,
// by the caller at the moment read after the last lock.
func (u *DeleteView) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	return u.d.Locks.run(ctx, actor, write{named: named{view: &id}, verb: domain.Delete}, func(ctx context.Context, h held) error {
		return u.d.Views.DeleteView(ctx, h.view.ID, u.d.Clock.Now(), actor.UserID)
	})
}
