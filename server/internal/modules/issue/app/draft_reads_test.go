package app_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Moving directories/authorization/total outside ReadTx, filtering by project
// membership, or checking owner before guest authorization must fail here.
func TestDraftReadVisibility(t *testing.T) {
	t.Run("own exited archived project in snapshot", func(t *testing.T) {
		w := newDraftWorld()
		d, err := app.NewGetDraft(app.GetDraftDeps{ReadTx: fakeTx{log: w.log}, Locks: w.locks()}).Execute(as(alice), w.s.row.ID)
		if err != nil || !reflect.DeepEqual(d, w.s.row) {
			t.Fatalf("draft=%+v err=%v", d, err)
		}
		want := []string{"Begin", line("GetDraft", d.ID), line("Find", web), line("Authorize", alice, domain.ActionDraftWrite, acme, uuid.UUID{})}
		if !slices.Equal(w.log.calls, want) {
			t.Fatalf("calls=%q", w.log.calls)
		}
	})
	for _, guest := range []bool{false, true} {
		t.Run("other owner", func(t *testing.T) {
			w := newDraftWorld()
			w.s.row.CreatedByID = bob
			want := error(domain.ErrDraftNotFound)
			if guest {
				w.auth.answers = []error{shared.Forbidden()}
				want = shared.Forbidden()
			}
			d, err := app.NewGetDraft(app.GetDraftDeps{ReadTx: fakeTx{log: w.log}, Locks: w.locks()}).Execute(as(alice), w.s.row.ID)
			if !sameError(err, want) || !reflect.DeepEqual(d, domain.Draft{}) {
				t.Fatalf("draft=%+v err=%v want=%v", d, err, want)
			}
		})
	}
	t.Run("deleted project and random id indistinguishable", func(t *testing.T) {
		w := newDraftWorld()
		delete(w.p.rows, web)
		u := app.NewGetDraft(app.GetDraftDeps{ReadTx: fakeTx{log: w.log}, Locks: w.locks()})
		d, a := u.Execute(as(alice), w.s.row.ID)
		other, b := u.Execute(as(alice), uuid.NewV7())
		if !errors.Is(a, domain.ErrDraftNotFound) || !sameError(a, b) || !reflect.DeepEqual(d, other) {
			t.Fatalf("%+v %v versus %+v %v", d, a, other, b)
		}
	})
	t.Run("list snapshot complete directory limit lookahead and cursor total", func(t *testing.T) {
		w := newDraftWorld()
		p := uuid.NewV7()
		w.p.rows[p] = acme
		w.s.rows = []domain.Draft{w.s.row, {ID: uuid.NewV7(), CreatedAt: earlier}}
		w.s.total = 7
		limit := 1
		u := app.NewListDrafts(app.ListDraftsDeps{ReadTx: fakeTx{log: w.log}, Locks: w.locks(), Workspaces: w.w, Projects: w.p, Drafts: w.s})
		page, err := u.Execute(as(alice), "acme", &limit, nil)
		if err != nil || len(page.Data) != 1 || page.Total != 7 || page.NextCursor == nil || w.s.list.Limit != 2 || w.s.list.ActorID != alice || w.s.list.WorkspaceID != acme || len(w.s.list.ProjectIDs) != 2 {
			t.Fatalf("page=%+v read=%+v err=%v", page, w.s.list, err)
		}
		cursor, err := domain.DecodeDraftCursor(*page.NextCursor)
		if err != nil || cursor.ID != w.s.row.ID || !cursor.CreatedAt.Equal(earlier) {
			t.Fatalf("cursor=%+v err=%v", cursor, err)
		}
		want := []string{"Begin", line("Workspace", "acme"), line("Authorize", alice, domain.ActionDraftWrite, acme, uuid.UUID{}), line("UndeletedIDs", acme), "ListDrafts"}
		if !slices.Equal(w.log.calls, want) {
			t.Fatalf("calls=%q", w.log.calls)
		}
		w.s.rows = nil
		page, err = u.Execute(as(alice), "acme", nil, page.NextCursor)
		if err != nil || page.Data == nil || len(page.Data) != 0 || page.Total != 7 || page.NextCursor != nil || w.s.list.After == nil || w.s.list.Limit != 51 {
			t.Fatalf("last page=%+v read=%+v err=%v", page, w.s.list, err)
		}
	})
	t.Run("bad pagination before snapshot", func(t *testing.T) {
		for _, badCursor := range []bool{false, true} {
			w := newDraftWorld()
			limit := 0
			c := "bad"
			var lp *int
			var cp *string
			if badCursor {
				cp = &c
			} else {
				lp = &limit
			}
			_, err := app.NewListDrafts(app.ListDraftsDeps{ReadTx: fakeTx{log: w.log}, Locks: w.locks(), Workspaces: w.w, Projects: w.p, Drafts: w.s}).Execute(as(alice), "acme", lp, cp)
			if err == nil || len(w.log.calls) != 0 {
				t.Fatalf("err=%v calls=%q", err, w.log.calls)
			}
		}
	})
}

func TestDraftReadVisibilityWorkspaceMissMatchesUnknownDraft(t *testing.T) {
	w := newDraftWorld()
	w.auth.answers = []error{shared.ErrNotVisible}
	u := app.NewGetDraft(app.GetDraftDeps{ReadTx: fakeTx{log: w.log}, Locks: w.locks()})
	d, a := u.Execute(as(alice), w.s.row.ID)
	unknown, b := u.Execute(as(alice), uuid.NewV7())
	if !sameError(a, b) || !reflect.DeepEqual(d, unknown) {
		t.Fatalf("hidden workspace=%+v %v unknown=%+v %v", d, a, unknown, b)
	}
}

func TestDraftReadVisibilityCrossWorkspaceProjectMatchesUnknownDraft(t *testing.T) {
	w := newDraftWorld()
	w.p.rows[web] = uuid.NewV7()
	u := app.NewGetDraft(app.GetDraftDeps{ReadTx: fakeTx{log: w.log}, Locks: w.locks()})
	d, a := u.Execute(as(alice), w.s.row.ID)
	unknown, b := u.Execute(as(alice), uuid.NewV7())
	if !sameError(a, b) || !reflect.DeepEqual(d, unknown) || w.auth.calls != 0 {
		t.Fatalf("cross workspace=%+v %v unknown=%+v %v calls=%d", d, a, unknown, b, w.auth.calls)
	}
}
