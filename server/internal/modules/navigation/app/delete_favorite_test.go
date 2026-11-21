package app_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
)

// Her favorite of each target is soft-deleted (spec 2.5, 2.7): deleted, and last changed, by her at the one moment read
// after the last lock, under its target's project's and view's FOR SHARE and its own FOR UPDATE; no other favorite is.
func TestDeleteAFavorite(t *testing.T) {
	for _, row := range []string{"proj", "cyc", "mod", "view", "wsview"} {
		t.Run(row, func(t *testing.T) {
			s, _ := stocked()
			id := s.idOf(row)
			if err := s.deleteFavorite().Execute(as(t), id); err != nil {
				t.Fatal(err)
			}
			if want := []deletion{{id: id, by: alice, at: moment}}; !reflect.DeepEqual(s.removals, want) {
				t.Errorf("deleted %+v; want %+v", s.removals, want)
			}
			wantCalls(t, s.world, rowCalls(row, nil, nil, "now", "delete favorite "+row+" by alice")...)
		})
	}
}

// A folder's deletion soft-deletes it and, in the one write, every undeleted favorite it holds, all at the one moment
// read after the last lock, by her (spec 2.5): a favorite in it whose target she no longer sees too, a folder being
// always hers to reach. One it held that was deleted before keeps its deletion; another folder's favorites and those
// at the top stay. The folder is locked FOR UPDATE, then the favorites it holds (spec 2.8).
func TestDeleteAFolderWithItsFavorites(t *testing.T) {
	s, a := stocked()
	old := favored(uuid.NewV7(), acme, domain.FavoriteProject, ptr(ops), "", &a.work.ID, 2)
	s.names[old.ID], s.stored, s.deleted[old.ID] = "old", append(s.stored, old), true
	s.edit(a.view.ID, func(f *domain.Favorite) { f.ParentID = &a.home.ID })
	archive(s.cycles, sprint)
	if err := s.deleteFavorite().Execute(as(t), a.work.ID); err != nil {
		t.Fatal(err)
	}
	var deleted []string
	for _, d := range s.removals {
		if d.at != moment || d.by != alice {
			t.Errorf("%s deleted at %s by %s; want at %s by alice", s.name(d.id), d.at, s.name(d.by), moment)
		}
		deleted = append(deleted, s.name(d.id))
	}
	if slices.Sort(deleted); !slices.Equal(deleted, []string{"cyc", "mod", "work"}) {
		t.Errorf("deleted %v; want work with cyc and mod", deleted)
	}
	wantCalls(t, s.world, rowCalls("work", nil, []string{"lock children work"}, "now", "delete favorite work by alice")...)
}

// Every favorite she may not delete is refused before its locks as a change of it is (spec 2.7): one 404, the value an
// id of none answers, whatever she may not have; a guest 403. Nothing is deleted and the clock is not read.
func TestDeleteAFavoriteRefusals(t *testing.T) {
	for _, tt := range refusals {
		t.Run(tt.name, func(t *testing.T) {
			s, a := stocked()
			if tt.change != nil {
				tt.change(s)
			}
			wantRefused(t, s.deleteFavorite().Execute(as(t), tt.row(a)), tt.want)
			wroteNoFavorite(t, s)
			wantCalls(t, s.world, tt.calls...)
		})
	}
}

// What the decision under the locks reads is final (spec 2.7, 2.8): another writer's change, or her grants',
// committed while the check ran and the lock waited, refuses the deletion as it would a change; nothing is deleted.
func TestDeleteAFavoriteDecidesUnderItsLocks(t *testing.T) {
	for _, tt := range meanwhiles {
		t.Run(tt.name, func(t *testing.T) {
			s, a := stocked()
			s.meanwhile = func(s *shelf) { tt.change(s, a) }
			wantRefused(t, s.deleteFavorite().Execute(as(t), s.idOf(tt.row)), tt.want)
			wroteNoFavorite(t, s)
		})
	}
}

// The favorites' rows are locked in one total order (spec 2.8, re-91 I2): a folder's before an item's. A change of a
// favorite in a folder that names the folder it is in, as a whole replacement may, locks the folder FOR SHARE, then
// the favorite FOR UPDATE, each in its own statement, though the favorite's id is less than the folder's; the folder's
// deletion locks the folder, then the favorites it holds. So neither waits for the other while holding a row the other
// waits for. A fake cannot deadlock: this pins the order of the lock calls; T12's interleaving runs the two at once on
// the database.
func TestTheWritesOfAFavoriteLockAFolderFirst(t *testing.T) {
	locks := func(calls []string) []string {
		return slices.DeleteFunc(slices.Clone(calls), func(c string) bool {
			return !strings.HasPrefix(c, "share folder ") && !strings.HasPrefix(c, "lock ")
		})
	}
	s, a := stocked()
	if a.cyc.ID.Compare(a.work.ID) >= 0 {
		t.Fatal("cyc's id is not less than work's")
	}
	p := into(&a.work.ID)
	p.Sequence = ptr(9.0)
	if _, err := s.updateFavorite().Execute(as(t), a.cyc.ID, p); err != nil {
		t.Fatal(err)
	}
	if got := locks(s.calls); !slices.Equal(got, []string{"share folder work", "lock favorite cyc"}) {
		t.Errorf("the change locks %q; want the folder, then the favorite", got)
	}
	s.calls = nil
	if err := s.deleteFavorite().Execute(as(t), a.work.ID); err != nil {
		t.Fatal(err)
	}
	if got := locks(s.calls); !slices.Equal(got, []string{"lock favorite work", "lock children work"}) {
		t.Errorf("the deletion locks %q; want the folder, then its favorites", got)
	}
}
