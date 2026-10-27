package app_test

import (
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Reordering W/auth/P/D, taking a gate, or locking a nil/duplicate project
// must fail these traces: the order protects both concurrency and disclosure.
func TestDraftLocks(t *testing.T) {
	t.Run("W auth sorted P D clock write", func(t *testing.T) {
		w := newDraftWorld()
		next := uuid.NewV7()
		w.p.rows[next] = acme
		_, err := w.update().Execute(as(alice), w.s.row.ID, domain.DraftPatch{SetProject: true, ProjectID: &next})
		if err != nil {
			t.Fatal(err)
		}
		projects := []uuid.UUID{web, next}
		slices.SortFunc(projects, func(a, b uuid.UUID) int { return a.Compare(b) })
		id := w.s.row.ID
		want := []string{line("GetDraft", id, "outside tx"), line("Find", web, "outside tx"), line("Authorize", alice, domain.ActionDraftWrite, acme, uuid.UUID{}, "outside tx"), line("Find", next, "outside tx"), line("Authorize", alice, shared.Action("project.read"), acme, next, "outside tx"), "Begin", line("ShareWorkspace", acme), line("GetDraft", id), line("Find", web), line("Authorize", alice, domain.ActionDraftWrite, acme, uuid.UUID{}), line("ShareMany", projects), line("LockDraft", id), line("Authorize", alice, shared.Action("project.read"), acme, next), "Now", "UpdateDraft"}
		if !slices.Equal(w.log.calls, want) {
			t.Fatalf("calls=%q\nwant=%q", w.log.calls, want)
		}
	})
	t.Run("same explicit project locked once", func(t *testing.T) {
		w := newDraftWorld()
		_, err := w.update().Execute(as(alice), w.s.row.ID, domain.DraftPatch{SetProject: true, ProjectID: &web})
		if err != nil || !slices.Equal(w.p.shared, []uuid.UUID{web}) {
			t.Fatalf("shared=%v err=%v", w.p.shared, err)
		}
	})
	for _, when := range []string{"workspace", "project", "draft"} {
		t.Run("changed after "+when, func(t *testing.T) {
			w := newDraftWorld()
			next := uuid.NewV7()
			w.p.rows[next] = acme
			change := func() { w.s.row.ProjectID = &next }
			switch when {
			case "workspace":
				w.w.waited = change
			case "project":
				w.p.waited = change
			case "draft":
				w.s.waited = change
			}
			_, err := w.update().Execute(as(alice), w.s.row.ID, domain.DraftPatch{})
			if !errors.Is(err, domain.ErrDraftChanged) || !slices.Equal(w.p.shared, []uuid.UUID{web}) || w.s.written.ID != (uuid.UUID{}) {
				t.Fatalf("err=%v shared=%v write=%+v", err, w.p.shared, w.s.written)
			}
		})
	}
	for _, deleted := range []bool{false, true} {
		t.Run("resource before demotion", func(t *testing.T) {
			w := newDraftWorld()
			w.auth.answers = []error{nil, shared.Forbidden()}
			w.w.waited = func() {
				if deleted {
					w.s.found = false
				}
			}
			_, err := w.update().Execute(as(alice), w.s.row.ID, domain.DraftPatch{})
			want := error(shared.Forbidden())
			if deleted {
				want = domain.ErrDraftNotFound
			}
			if !sameError(err, want) || len(w.p.shared) != 0 || slices.Contains(w.log.calls, line("LockDraft", w.s.row.ID)) {
				t.Fatalf("err=%v want=%v calls=%q", err, want, w.log.calls)
			}
		})
	}
	t.Run("deleted old project after W before auth", func(t *testing.T) {
		w := newDraftWorld()
		w.auth.answers = []error{nil, shared.Forbidden()}
		w.w.waited = func() { delete(w.p.rows, web) }
		err := w.delete().Execute(as(alice), w.s.row.ID)
		if !errors.Is(err, domain.ErrDraftNotFound) || w.auth.calls != 1 {
			t.Fatalf("err=%v auth=%d", err, w.auth.calls)
		}
	})
	for _, when := range []string{"P", "D"} {
		t.Run("deletion while waiting "+when, func(t *testing.T) {
			w := newDraftWorld()
			if when == "P" {
				w.p.waited = func() { delete(w.p.rows, web) }
			} else {
				w.s.waited = func() { w.s.found = false }
			}
			err := w.delete().Execute(as(alice), w.s.row.ID)
			if !errors.Is(err, domain.ErrDraftNotFound) || w.s.deleted.ID != (uuid.UUID{}) {
				t.Fatalf("err=%v deleted=%+v", err, w.s.deleted)
			}
		})
	}
}

func TestDraftLocksRejectProjectVisibilityLostWhileWaiting(t *testing.T) {
	w := newDraftWorld()
	w.auth.answers = []error{nil, nil, nil, shared.ErrNotVisible}
	_, err := w.update().Execute(as(alice), w.s.row.ID, domain.DraftPatch{SetProject: true, ProjectID: &web})
	var e *shared.Error
	if !errors.As(err, &e) || e.Kind != shared.KindInvalid || len(e.Fields) != 1 || e.Fields[0].Field != "project_id" || w.s.written.ID != (uuid.UUID{}) || slices.Contains(w.log.calls, "Now") {
		t.Fatalf("err=%v calls=%q", err, w.log.calls)
	}
}

func TestDraftLocksRejectOwnerLostWhileWaiting(t *testing.T) {
	w := newDraftWorld()
	w.s.waited = func() { w.s.row.CreatedByID = bob }
	_, err := w.update().Execute(as(alice), w.s.row.ID, domain.DraftPatch{})
	if !errors.Is(err, domain.ErrDraftNotFound) || w.s.written.ID != (uuid.UUID{}) {
		t.Fatalf("err=%v write=%+v", err, w.s.written)
	}
}
