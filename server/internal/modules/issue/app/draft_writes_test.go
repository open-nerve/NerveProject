package app_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func sameError(a, b error) bool { return reflect.DeepEqual(a, b) }

// These assertions reject early no-op returns, partial payload merges, missing
// audit stamps, and pure validation after resource reads.
func TestDraftCRUD(t *testing.T) {
	t.Run("create nil skips P D and stamps actor time", func(t *testing.T) {
		w := newDraftWorld()
		d, err := w.create().Execute(as(alice), "acme", domain.NewDraft{Payload: []byte(`{}`)})
		if err != nil || d.ID == (uuid.UUID{}) || d.WorkspaceID != acme || d.CreatedByID != alice || !d.CreatedAt.Equal(clockNow) || d.ProjectID != nil || string(d.Payload) != "{}" {
			t.Fatalf("draft=%+v err=%v", d, err)
		}
		want := []string{line("Workspace", "acme", "outside tx"), line("Authorize", alice, domain.ActionDraftWrite, acme, uuid.UUID{}, "outside tx"), "Begin", line("ShareWorkspace", acme), line("Authorize", alice, domain.ActionDraftWrite, acme, uuid.UUID{}), "Now", "InsertDraft"}
		if !slices.Equal(w.log.calls, want) {
			t.Fatalf("calls=%q want=%q", w.log.calls, want)
		}
	})
	t.Run("replace payload and clear project", func(t *testing.T) {
		w := newDraftWorld()
		d, err := w.update().Execute(as(alice), w.s.row.ID, domain.DraftPatch{SetProject: true, SetPayload: true, Payload: []byte(`{}`)})
		if err != nil || d.ProjectID != nil || string(d.Payload) != "{}" || w.s.written.By != alice || !w.s.written.At.Equal(clockNow) {
			t.Fatalf("draft=%+v err=%v write=%+v", d, err, w.s.written)
		}
	})
	t.Run("omitted patch uses locked payload", func(t *testing.T) {
		w := newDraftWorld()
		w.s.waited = func() { w.s.row.Payload = []byte(`{"priority":"anything"}`) }
		d, err := w.update().Execute(as(alice), w.s.row.ID, domain.DraftPatch{})
		if err != nil || string(d.Payload) != `{"priority":"anything"}` {
			t.Fatalf("draft=%+v err=%v", d, err)
		}
	})
	t.Run("delete has no gate and stamps time", func(t *testing.T) {
		w := newDraftWorld()
		err := w.delete().Execute(as(alice), w.s.row.ID)
		if err != nil || w.s.deleted.ID != w.s.row.ID || w.s.deleted.By != alice || !w.s.deleted.At.Equal(clockNow) {
			t.Fatalf("delete=%+v err=%v", w.s.deleted, err)
		}
		for _, call := range w.log.calls {
			if call == line("Enter", web, "outside tx") {
				t.Fatal("delete entered gate")
			}
		}
	})
	for _, op := range []string{"create", "update"} {
		t.Run("NUL before reads "+op, func(t *testing.T) {
			w := newDraftWorld()
			raw := []byte(`{"name":"x\u0000"}`)
			var err error
			if op == "create" {
				_, err = w.create().Execute(as(alice), "acme", domain.NewDraft{Payload: raw})
			} else {
				_, err = w.update().Execute(as(alice), w.s.row.ID, domain.DraftPatch{SetPayload: true, Payload: raw})
			}
			var e *shared.Error
			if !errors.As(err, &e) || e.Kind != shared.KindInvalid || len(w.log.calls) != 0 {
				t.Fatalf("err=%v calls=%q", err, w.log.calls)
			}
		})
	}
	for _, op := range []string{"create", "update", "delete"} {
		for _, deny := range []error{shared.Forbidden(), shared.ErrNotVisible} {
			t.Run(op+" auth refusal "+deny.Error(), func(t *testing.T) {
				w := newDraftWorld()
				w.auth.answers = []error{deny}
				var err error
				switch op {
				case "create":
					_, err = w.create().Execute(as(alice), "acme", domain.NewDraft{Payload: []byte(`{}`)})
				case "update":
					_, err = w.update().Execute(as(alice), w.s.row.ID, domain.DraftPatch{})
				case "delete":
					err = w.delete().Execute(as(alice), w.s.row.ID)
				}
				if err == nil || slices.Contains(w.log.calls, "Begin") {
					t.Fatalf("err=%v calls=%q", err, w.log.calls)
				}
			})
		}
	}
	for _, op := range []string{"update", "delete"} {
		t.Run("owner refusal outside tx "+op, func(t *testing.T) {
			w := newDraftWorld()
			w.s.row.CreatedByID = bob
			var err error
			if op == "update" {
				_, err = w.update().Execute(as(alice), w.s.row.ID, domain.DraftPatch{})
			} else {
				err = w.delete().Execute(as(alice), w.s.row.ID)
			}
			if !errors.Is(err, domain.ErrDraftNotFound) || slices.Contains(w.log.calls, "Begin") {
				t.Fatalf("err=%v calls=%q", err, w.log.calls)
			}
		})
	}
	for _, op := range []string{"omit", "clear", "delete", "explicit public", "explicit private", "cross workspace", "missing"} {
		t.Run("exited archived project "+op, func(t *testing.T) {
			w := newDraftWorld()
			patch := domain.DraftPatch{}
			wantInvalid := false
			switch op {
			case "clear":
				patch.SetProject = true
			case "explicit public":
				patch.SetProject = true
				patch.ProjectID = &web
			case "explicit private":
				patch.SetProject = true
				patch.ProjectID = &web
				w.auth.answers = []error{nil, shared.ErrNotVisible}
				wantInvalid = true
			case "cross workspace":
				p := uuid.NewV7()
				w.p.rows[p] = uuid.NewV7()
				patch.SetProject = true
				patch.ProjectID = &p
				wantInvalid = true
			case "missing":
				p := uuid.NewV7()
				patch.SetProject = true
				patch.ProjectID = &p
				wantInvalid = true
			}
			var err error
			if op == "delete" {
				err = w.delete().Execute(as(alice), w.s.row.ID)
			} else {
				_, err = w.update().Execute(as(alice), w.s.row.ID, patch)
			}
			if wantInvalid {
				var e *shared.Error
				if !errors.As(err, &e) || e.Kind != shared.KindInvalid || len(e.Fields) != 1 || e.Fields[0].Field != "project_id" || e.Fields[0].Code != shared.FieldNotAllowed {
					t.Fatalf("err=%#v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDraftCRUDFailuresDoNotReturnDraftData(t *testing.T) {
	for _, port := range []string{"GetDraft", "Find", "Authorize", "ShareWorkspace", "ShareMany", "LockDraft", "UpdateDraft"} {
		t.Run(port, func(t *testing.T) {
			w := newDraftWorld()
			w.log.fail = port
			d, err := w.update().Execute(as(alice), w.s.row.ID, domain.DraftPatch{})
			if !errors.Is(err, errDisk) || !reflect.DeepEqual(d, domain.Draft{}) {
				t.Fatalf("draft=%+v err=%v", d, err)
			}
		})
	}
	for _, op := range []string{"create", "delete"} {
		t.Run(op+" write failure", func(t *testing.T) {
			w := newDraftWorld()
			if op == "create" {
				w.log.fail = "InsertDraft"
				d, err := w.create().Execute(as(alice), "acme", domain.NewDraft{Payload: []byte(`{}`)})
				if !errors.Is(err, errDisk) || !reflect.DeepEqual(d, domain.Draft{}) {
					t.Fatalf("draft=%+v err=%v", d, err)
				}
			} else {
				w.log.fail = "DeleteDraft"
				if err := w.delete().Execute(as(alice), w.s.row.ID); !errors.Is(err, errDisk) {
					t.Fatal(err)
				}
			}
		})
	}
}
