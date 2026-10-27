package app_test

import (
	"context"
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

// publishPrecheckProjects models deletion between resource's first Find and
// the publication precheck's second Find, without changing shared CRUD fakes.
type publishPrecheckProjects struct {
	*draftProjects
	finds          int
	deleteOnSecond bool
}

func (p *publishPrecheckProjects) Find(ctx context.Context, id uuid.UUID) (uuid.UUID, bool, error) {
	p.finds++
	if p.finds == 2 && p.deleteOnSecond {
		delete(p.rows, id)
	}
	return p.draftProjects.Find(ctx, id)
}

// Physical disappearance conceals the named draft; an existing project whose
// create decision is invisible retains create's project404. Both stop before
// taking the project gate, opening a transaction or decoding stored content.
func TestPublishDraftPrecheckProjectDeletion(t *testing.T) {
	for _, tc := range []struct {
		name    string
		deleted bool
		want    error
	}{
		{"deleted between project reads", true, domain.ErrDraftNotFound},
		{"existing project create invisible", false, domain.ErrProjectNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newDraftWorld()
			p := &publishPrecheckProjects{draftProjects: w.p, deleteOnSecond: tc.deleted}
			if !tc.deleted {
				w.auth.answers = []error{nil, shared.ErrNotVisible}
			}
			locks := app.NewDraftLocks(app.NewLocks(p, nil, w.auth, fakeGate{log: w.log}, fakeTx{log: w.log}), w.w, p, w.s)
			decoder := &refusingPayload{}
			uc := app.NewPublishDraft(app.PublishDraftDeps{Locks: locks, Decoder: decoder, Drafts: w.s})
			_, err := uc.Execute(as(alice), w.s.row.ID)
			if !errors.Is(err, tc.want) {
				t.Errorf("Execute=%v want %v", err, tc.want)
			}
			if p.finds != 2 || decoder.calls != 0 || slices.Contains(w.log.calls, "Begin") || slices.Contains(w.log.calls, line("Enter", web, "outside tx")) {
				t.Fatalf("finds=%d decoder=%d calls=%q", p.finds, decoder.calls, w.log.calls)
			}
			wantAuth := 2
			if tc.deleted {
				wantAuth = 1
			}
			if w.auth.calls != wantAuth {
				t.Fatalf("authorization calls=%d want %d", w.auth.calls, wantAuth)
			}
		})
	}
}
