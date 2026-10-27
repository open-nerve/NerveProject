package app_test

import (
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type refusingPayload struct{ calls int }

func (d *refusingPayload) DecodeCreatePayload([]byte) (domain.NewIssue, error) {
	d.calls++
	return domain.NewIssue{}, errDisk
}

// Authorization/resource/changed/archive refusals must precede decoding. These
// control-flow tests do not use fakeTx to claim transactional atomicity.
func TestPublishDraftRefusesBeforeDecoding(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*draftWorld)
		want  error
	}{
		{"absent", func(w *draftWorld) { w.s.found = false }, domain.ErrDraftNotFound},
		{"other owner", func(w *draftWorld) { w.s.row.CreatedByID = bob }, domain.ErrDraftNotFound},
		{"guest", func(w *draftWorld) { w.auth.answers = []error{shared.Forbidden()} }, shared.Forbidden()},
		{"deleted after W", func(w *draftWorld) { w.w.waited = func() { w.s.found = false } }, domain.ErrDraftNotFound},
		{"changed after D", func(w *draftWorld) { w.s.waited = func() { next := uuid.NewV7(); w.s.row.ProjectID = &next } }, domain.ErrDraftChanged},
		{"archived", func(w *draftWorld) { w.p.locked.Archived = true }, domain.ErrProjectArchived},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newDraftWorld()
			w.p.locked = &app.LockedProject{WorkspaceID: acme}
			w.p.found = true
			w.p.id = web
			tc.setup(w)
			decoder := &refusingPayload{}
			uc := app.NewPublishDraft(app.PublishDraftDeps{Locks: w.locks(), Decoder: decoder, Drafts: w.s})
			_, err := uc.Execute(as(alice), w.s.row.ID)
			if !sameError(err, tc.want) || decoder.calls != 0 || w.s.deleted.ID != uuid.Nil() {
				t.Fatalf("err=%v want=%v decoded=%d", err, tc.want, decoder.calls)
			}
		})
	}
	t.Run("missing project before gate", func(t *testing.T) {
		w := newDraftWorld()
		w.s.row.ProjectID = nil
		_, err := app.NewPublishDraft(app.PublishDraftDeps{Locks: w.locks()}).Execute(as(alice), w.s.row.ID)
		var e *shared.Error
		if !errors.As(err, &e) || len(e.Fields) != 1 || e.Fields[0].Field != "project_id" || e.Fields[0].Code != shared.FieldRequired || slices.Contains(w.log.calls, "Begin") {
			t.Fatal(err, w.log.calls)
		}
	})
}
