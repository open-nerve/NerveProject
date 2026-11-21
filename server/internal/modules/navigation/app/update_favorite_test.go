package app_test

import (
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// into is a change of a favorite's folder to parent, nil for none.
func into(parent *uuid.UUID) domain.FavoritePatch {
	return domain.FavoritePatch{SetParentID: true, ParentID: parent}
}

// A change replaces each field it gives whole (spec 2.4, 2.5): her favorite moved into a folder, out of its folder, or
// into another at another sequence; its sequence alone; a folder renamed without its surrounding white space, renamed
// to its own name in another case, which is no conflict, or to the name of another's folder, of her deleted one and
// of her folder in another workspace; a folder's sequence and name. It is written, last changed by her at the one
// moment read after the last lock, and answered with its target's name and icon now. Its locks are spec 2.8's: the
// target's project's and view's FOR SHARE, the new parent's FOR SHARE, then its own FOR UPDATE.
func TestChangeAFavorite(t *testing.T) {
	for _, tt := range []struct {
		name, row string
		p         func(a alices) domain.FavoritePatch
		change    func(f *domain.Favorite, a alices)
		parent    string
	}{
		{"her project's into a folder", "proj", func(a alices) domain.FavoritePatch { return into(&a.home.ID) },
			func(f *domain.Favorite, a alices) { f.ParentID = &a.home.ID }, "home"},
		{"her cycle's out of its folder", "cyc", func(alices) domain.FavoritePatch { return into(nil) }, func(f *domain.Favorite, _ alices) { f.ParentID = nil }, ""},
		{"her module's into another folder at another sequence", "mod", func(a alices) domain.FavoritePatch {
			p := into(&a.home.ID)
			p.Sequence = ptr(-2.5)
			return p
		}, func(f *domain.Favorite, a alices) { f.ParentID, f.Sequence = &a.home.ID, -2.5 }, "home"},
		{"her view's sequence", "view", func(alices) domain.FavoritePatch { return domain.FavoritePatch{Sequence: ptr(1e9)} },
			func(f *domain.Favorite, _ alices) { f.Sequence = 1e9 }, ""},
		{"a workspace's view's into a folder", "wsview", func(a alices) domain.FavoritePatch { return into(&a.work.ID) },
			func(f *domain.Favorite, a alices) { f.ParentID = &a.work.ID }, "work"},
		{"a folder renamed", "work", func(alices) domain.FavoritePatch { return domain.FavoritePatch{Name: ptr("  Desk ")} },
			func(f *domain.Favorite, _ alices) { f.Name = "Desk" }, ""},
		{"a folder renamed to its own name in another case", "work", func(alices) domain.FavoritePatch { return domain.FavoritePatch{Name: ptr("WORK")} },
			func(f *domain.Favorite, _ alices) { f.Name = "WORK" }, ""},
		{"a folder renamed to another's folder's name", "home", func(alices) domain.FavoritePatch { return domain.FavoritePatch{Name: ptr("office")} },
			func(f *domain.Favorite, _ alices) { f.Name = "office" }, ""},
		{"a folder renamed to her deleted folder's name", "home", func(alices) domain.FavoritePatch { return domain.FavoritePatch{Name: ptr("GONE")} },
			func(f *domain.Favorite, _ alices) { f.Name = "GONE" }, ""},
		{"a folder renamed to her other workspace's folder's name", "home", func(alices) domain.FavoritePatch { return domain.FavoritePatch{Name: ptr("library")} },
			func(f *domain.Favorite, _ alices) { f.Name = "library" }, ""},
		{"a folder's sequence and name", "home", func(alices) domain.FavoritePatch { return domain.FavoritePatch{Sequence: ptr(1.0), Name: ptr("Desk")} },
			func(f *domain.Favorite, _ alices) { f.Sequence, f.Name = 1, "Desk" }, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s, a := stocked()
			row := map[string]domain.Favorite{"proj": a.proj, "cyc": a.cyc, "mod": a.mod, "view": a.view, "wsview": a.wsview, "work": a.work, "home": a.home}[tt.row]
			want := row
			tt.change(&want, a)
			want.UpdatedAt = moment
			got, err := s.updateFavorite().Execute(as(t), row.ID, tt.p(a))
			if err != nil {
				t.Fatal(err)
			}
			if answer := answerOf(tt.row, want); !reflect.DeepEqual(got, answer) {
				t.Errorf("answered %+v\nwant %+v", got, answer)
			}
			if len(s.changed) != 1 || !reflect.DeepEqual(s.changed[0].favorite, want) || s.changed[0].by != alice || !reflect.DeepEqual(s.storedOf(row.ID), want) {
				t.Errorf("changed %+v; want %+v, by alice", s.changed, want)
			}
			var before []string
			if tt.parent != "" {
				before = []string{"share folder " + tt.parent}
			}
			wantCalls(t, s.world, rowCalls(tt.row, before, nil, "now", "update favorite "+tt.row+" by alice")...)
		})
	}
}

// A change that leaves her favorite as its lock read it, after its name is put without its surrounding white space,
// answers it as it is (spec 2.4): {}, every field as it is, out of any folder at the top, a folder's name as it is,
// and a folder given no parent, the none it has. Nothing is written and the clock is not read: its updated_at stays.
// A parent it names is locked all the same.
func TestChangeAFavoriteLeavingItAsItIs(t *testing.T) {
	for _, tt := range []struct {
		name, row string
		p         func(a alices) domain.FavoritePatch
		parent    string
	}{
		{"{}", "proj", func(alices) domain.FavoritePatch { return domain.FavoritePatch{} }, ""},
		{"every field as it is", "cyc", func(a alices) domain.FavoritePatch {
			p := into(&a.work.ID)
			p.Sequence = ptr(5.0)
			return p
		}, "work"},
		{"out of any folder at the top", "proj", func(alices) domain.FavoritePatch { return into(nil) }, ""},
		{"a folder's name as it is", "work", func(alices) domain.FavoritePatch {
			return domain.FavoritePatch{Name: ptr(" Work "), Sequence: ptr(100.0)}
		}, ""},
		{"a folder given no parent", "work", func(alices) domain.FavoritePatch { return into(nil) }, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s, a := stocked()
			row := map[string]domain.Favorite{"proj": a.proj, "cyc": a.cyc, "work": a.work}[tt.row]
			got, err := s.updateFavorite().Execute(as(t), row.ID, tt.p(a))
			if err != nil {
				t.Fatal(err)
			}
			if answer := answerOf(tt.row, row); !reflect.DeepEqual(got, answer) {
				t.Errorf("answered %+v\nwant %+v, as it is", got, answer)
			}
			wroteNoFavorite(t, s)
			var before []string
			if tt.parent != "" {
				before = []string{"share folder " + tt.parent}
			}
			wantCalls(t, s.world, rowCalls(tt.row, before, nil)...)
		})
	}
}

// A folder renamed to the name, in any case, of another of her undeleted folders in the workspace is 409, from the
// store, and nothing is changed (spec 2.5).
func TestChangeAFolderToATakenName(t *testing.T) {
	s, a := stocked()
	_, err := s.updateFavorite().Execute(as(t), a.home.ID, domain.FavoritePatch{Name: ptr(" wORK "), Sequence: ptr(1.0)})
	wantSame(t, err, domain.ErrFavoriteFolderNameTaken)
	if len(s.changed) != 0 || !reflect.DeepEqual(s.storedOf(a.home.ID), a.home) {
		t.Errorf("changed %+v; want nothing", s.changed)
	}
	wantCalls(t, s.world, rowCalls("home", nil, nil, "now", "update favorite home by alice")...)
}

// The parent a change names is one of her own undeleted folders in the workspace (spec 2.5), as its lock under the
// others finds it (the fake mirrors the store's rule): another's folder, her favorite of a target, the favorite itself,
// her deleted folder, her folder of another workspace and an id of none are each one 422, parent_id not_allowed, with
// nothing written. A folder given a parent is refused before any lock: no folder holds one (D42).
func TestChangeAFavoriteIntoAParentSheMayNot(t *testing.T) {
	for _, tt := range []struct {
		name   string
		parent func(a alices) uuid.UUID
	}{
		{"another's folder", func(a alices) uuid.UUID { return a.bobs.ID }},
		{"her favorite of a target", func(a alices) uuid.UUID { return a.proj.ID }},
		{"the favorite itself", func(a alices) uuid.UUID { return a.mod.ID }},
		{"her deleted folder", func(a alices) uuid.UUID { return a.gone.ID }},
		{"her folder of another workspace", func(a alices) uuid.UUID { return a.betas.ID }},
		{"an id of none", func(alices) uuid.UUID { return nowhere }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s, a := stocked()
			p := into(ptr(tt.parent(a)))
			p.Sequence = ptr(1.0)
			_, err := s.updateFavorite().Execute(as(t), a.mod.ID, p)
			if fields := fieldsOf(t, err); !reflect.DeepEqual(fields, []shared.FieldError{parentNotAllowed}) {
				t.Errorf("fields %+v; want %+v", fields, parentNotAllowed)
			}
			wroteNoFavorite(t, s)
			wantCalls(t, s.world, rowCalls("mod", []string{"share folder " + s.name(tt.parent(a))}, nil)...)
		})
	}
	t.Run("a folder", func(t *testing.T) {
		s, a := stocked()
		_, err := s.updateFavorite().Execute(as(t), a.work.ID, into(&a.home.ID))
		want := shared.FieldError{Field: "parent_id", Code: shared.FieldNotAllowed, Message: "is not given for a folder"}
		if fields := fieldsOf(t, err); !reflect.DeepEqual(fields, []shared.FieldError{want}) {
			t.Errorf("fields %+v; want %+v", fields, want)
		}
		wroteNoFavorite(t, s)
		wantCalls(t, s.world, checked("work")...)
	})
}

// Its refusals before the locks come in spec 2.7's order, each way: every favorite she may not have is one 404, the
// value an id of none answers, whatever the body, a bad one too, which is checked only once her favorite is read, its
// workspace decided and its target seen; a guest is 403 the same. A bad body on her own favorite is one 422 with every
// problem of it by the favorite's kind, before any lock. None writes or reads the clock.
func TestChangeAFavoriteRefusals(t *testing.T) {
	for _, tt := range refusals {
		for body, p := range map[string]domain.FavoritePatch{"": {Sequence: ptr(1.0)}, " with a bad body": badItem} {
			t.Run(tt.name+body, func(t *testing.T) {
				s, a := stocked()
				if tt.change != nil {
					tt.change(s)
				}
				_, err := s.updateFavorite().Execute(as(t), tt.row(a), p)
				wantRefused(t, err, tt.want)
				wroteNoFavorite(t, s)
				wantCalls(t, s.world, tt.calls...)
			})
		}
	}
	t.Run("a bad body", func(t *testing.T) {
		s, a := stocked()
		_, err := s.updateFavorite().Execute(as(t), a.proj.ID, badItem)
		fields := fieldsOf(t, err)
		if len(fields) != 2 || fields[0].Field != "sequence" || fields[1].Field != "name" {
			t.Errorf("fields %+v; want sequence and name", fields)
		}
		wroteNoFavorite(t, s)
		wantCalls(t, s.world, checked("proj")...)
	})
}

// What the decision under the locks reads is final (spec 2.7, 2.8): another writer's change, or her grants',
// committed while the check ran and the lock waited, is what it decides on. Each 404 is the one value; a parent
// deleted meanwhile is the 422; and whether it changes anything is told against the favorite as its lock read it.
func TestChangeAFavoriteDecidesUnderItsLocks(t *testing.T) {
	for _, tt := range meanwhiles {
		t.Run(tt.name, func(t *testing.T) {
			s, a := stocked()
			s.meanwhile = func(s *shelf) { tt.change(s, a) }
			_, err := s.updateFavorite().Execute(as(t), s.idOf(tt.row), domain.FavoritePatch{Sequence: ptr(1.0)})
			wantRefused(t, err, tt.want)
			wroteNoFavorite(t, s)
		})
	}
	t.Run("the parent deleted meanwhile", func(t *testing.T) {
		s, a := stocked()
		s.meanwhile = func(s *shelf) { s.deleted[a.home.ID] = true }
		_, err := s.updateFavorite().Execute(as(t), a.proj.ID, into(&a.home.ID))
		if fields := fieldsOf(t, err); !reflect.DeepEqual(fields, []shared.FieldError{parentNotAllowed}) {
			t.Errorf("fields %+v; want %+v", fields, parentNotAllowed)
		}
		wroteNoFavorite(t, s)
	})
	for _, tt := range []struct {
		name, row string
		meanwhile func(f *domain.Favorite, a alices)
		p         func(a alices) domain.FavoritePatch
		written   bool
	}{
		{"its sequence changed meanwhile, changed back", "proj", func(f *domain.Favorite, _ alices) { f.Sequence = 8 },
			func(alices) domain.FavoritePatch { return domain.FavoritePatch{Sequence: ptr(7.0)} }, true},
		{"a folder renamed meanwhile, renamed back", "work", func(f *domain.Favorite, _ alices) { f.Name = "Desk" },
			func(alices) domain.FavoritePatch { return domain.FavoritePatch{Name: ptr("Work")} }, true},
		{"moved into the folder meanwhile", "proj", func(f *domain.Favorite, a alices) { f.ParentID = &a.home.ID },
			func(a alices) domain.FavoritePatch { return into(&a.home.ID) }, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s, a := stocked()
			id := s.idOf(tt.row)
			s.meanwhile = func(s *shelf) { s.edit(id, func(f *domain.Favorite) { tt.meanwhile(f, a) }) }
			got, err := s.updateFavorite().Execute(as(t), id, tt.p(a))
			if err != nil {
				t.Fatal(err)
			}
			if !tt.written {
				wroteNoFavorite(t, s)
				if !reflect.DeepEqual(got.Favorite, s.storedOf(id)) {
					t.Errorf("answered %+v; want %+v, as its lock read it", got.Favorite, s.storedOf(id))
				}
				return
			}
			if len(s.changed) != 1 || !reflect.DeepEqual(got.Favorite, s.changed[0].favorite) {
				t.Fatalf("changed %+v, answered %+v; want the one change answered", s.changed, got.Favorite)
			}
			want := map[string]domain.Favorite{"proj": a.proj, "work": a.work}[tt.row]
			want.UpdatedAt = moment
			if !reflect.DeepEqual(got.Favorite, want) {
				t.Errorf("answered %+v; want %+v", got.Favorite, want)
			}
		})
	}
}
