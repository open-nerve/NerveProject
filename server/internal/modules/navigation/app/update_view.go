package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateView changes a view of either kind by its id (updateView).
type UpdateView struct{ d WriteDeps }

// NewUpdateView returns the use case over d.
func NewUpdateView(d WriteDeps) *UpdateView { return &UpdateView{d: d} }

// Execute changes the view id by p (P3b spec 2.4, 2.5), through its locks
// and decision (Locks.run): the view read without a lock and its kind's
// update decided on it, a 404 navigation.view_not_found for every view the
// caller cannot have, then 403 for a role the rule refuses and for anyone
// but its owner; then p's every problem in one 422, its kind's words
// among them (Words.checkPatch); then the locks, the view read again under
// them and the decision again; then 409 while its project's views are
// turned off. A p that leaves the view as the locks read it, {} among
// them, answers that view and writes nothing (domain.Patch.Apply); a
// change replaces each field p gives whole, last changed by the caller at
// the moment read after the last lock.
func (u *UpdateView) Execute(ctx context.Context, id uuid.UUID, p domain.Patch) (domain.View, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.View{}, err
	}
	var checked domain.Patch
	w := write{named: named{view: &id}, verb: domain.Update, check: func(h held) error {
		checked, err = u.d.Words.checkPatch(h.kind(), p)
		return err
	}}
	var out domain.View
	err = u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		out = *h.view
		content, changed := checked.Apply(out.Content)
		if !changed {
			return nil
		}
		out.Content, out.UpdatedAt = content, u.d.Clock.Now()
		return u.d.Views.UpdateView(ctx, out, actor.UserID)
	})
	if err != nil {
		return domain.View{}, err
	}
	return out, nil
}
