package postgresadapter_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// favorite is user's favorite numbered n of kind in workspace, at sequence, made an hour before now: a folder named
// for n; a project's, of project; any other's, of a new target of project, nil for a workspace's view.
func favorite(n int, kind domain.FavoriteKind, workspace uuid.UUID, project *uuid.UUID, user uuid.UUID, sequence float64) domain.Favorite {
	at := now.Add(-time.Hour)
	f := domain.Favorite{ID: numberedID(n), WorkspaceID: workspace, ProjectID: project, UserID: user, Kind: kind, Sequence: sequence, CreatedAt: at, UpdatedAt: at}
	switch kind {
	case domain.FavoriteFolder:
		f.Name = fmt.Sprintf("Folder %d", n)
	case domain.FavoriteProject:
		f.EntityID = project
	default:
		f.EntityID = ptr(uuid.NewV7())
	}
	return f
}

// in is f in the folder parent.
func in(f domain.Favorite, parent uuid.UUID) domain.Favorite {
	f.ParentID = &parent
	return f
}

// create stores each favorite, made by its owner, which must be new.
func create(t *testing.T, s *postgresadapter.Store, fs ...domain.Favorite) {
	t.Helper()
	for _, f := range fs {
		if got, created, err := s.InsertFavorite(context.Background(), f, f.UserID); err != nil || !created || !reflect.DeepEqual(got, f) {
			t.Fatalf("create %s: %+v, %v, %v", f.ID, got, created, err)
		}
	}
}

func favoriteIDs(fs []domain.Favorite) []uuid.UUID {
	out := make([]uuid.UUID, len(fs))
	for i, f := range fs {
		out[i] = f.ID
	}
	return out
}

// A favorite of each kind is stored and read back as written (P3c spec 2.3): its project, target, folder's name,
// parent, sequence and moments, a favorite of a target without a name; its creation's writer stamps both audit
// columns. It is one's own: another's, a deleted one and one not there are not found. A change writes its parent,
// sequence and name and stamps its moment and writer, keeping the creation.
func TestAFavoriteRoundTrips(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newViews(t, pool)
	folder := favorite(1, domain.FavoriteFolder, w.workspace, nil, w.owner, 65535)
	mine := []domain.Favorite{folder, favorite(2, domain.FavoriteProject, w.workspace, &w.project, w.owner, 75535.25),
		in(favorite(3, domain.FavoriteCycle, w.workspace, &w.project, w.owner, -1), folder.ID), favorite(4, domain.FavoriteModule, w.workspace, &w.ops, w.owner, 0),
		favorite(5, domain.FavoriteView, w.workspace, &w.project, w.owner, 1e300), in(favorite(6, domain.FavoriteView, w.workspace, nil, w.owner, 3), folder.ID)}
	hers := favorite(7, domain.FavoriteCycle, w.workspace, &w.project, w.other, 65535)
	create(t, s, append(mine, hers)...)
	for _, want := range mine {
		if got, found, err := s.Favorite(ctx, w.owner, want.ID); err != nil || !found || !reflect.DeepEqual(got, want) {
			t.Fatalf("read %+v, %v, %v; want %+v", got, found, err, want)
		}
	}
	var stamped int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM user_favorites WHERE created_by_id = user_id AND updated_by_id = user_id
		AND (name IS NULL) = (entity_type <> 'folder')`).Scan(&stamped); err != nil || stamped != 7 {
		t.Fatalf("made by their owners, a name for the folder alone: %d of 7, %v", stamped, err)
	}
	for _, tt := range []struct {
		user, id uuid.UUID
	}{{w.owner, hers.ID}, {w.other, folder.ID}, {w.owner, uuid.NewV7()}} {
		if got, found, err := s.Favorite(ctx, tt.user, tt.id); found || err != nil {
			t.Fatalf("%s read %s: %+v, %v", tt.user, tt.id, got, err)
		}
	}
	moved, renamed := mine[2], folder
	moved.ParentID, moved.Sequence, moved.UpdatedAt = nil, 1.5, now
	renamed.Name, renamed.Sequence, renamed.UpdatedAt = "Home", 85535, now
	for _, f := range []domain.Favorite{moved, renamed} {
		if err := s.UpdateFavorite(ctx, f, w.other); err != nil {
			t.Fatal(err)
		}
		if got, found, err := s.Favorite(ctx, w.owner, f.ID); err != nil || !found || !reflect.DeepEqual(got, f) {
			t.Fatalf("changed %+v, %v, %v; want %+v", got, found, err, f)
		}
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM user_favorites WHERE id = ANY($1) AND created_by_id = $2 AND updated_by_id = $3",
		[]uuid.UUID{moved.ID, renamed.ID}, w.owner, w.other).Scan(&stamped); err != nil || stamped != 2 {
		t.Fatalf("made by the owner, changed by the writer: %d of 2, %v", stamped, err)
	}
	if err := s.DeleteFavorite(ctx, moved.ID, now, w.owner); err != nil {
		t.Fatal(err)
	}
	if got, found, err := s.Favorite(ctx, w.owner, moved.ID); found || err != nil {
		t.Fatalf("read a deleted one: %+v, %v", got, err)
	}
}

// One's list in a workspace (spec 2.4): his undeleted favorites there, those in his folders too, by sequence
// descending and, at one sequence, by id descending; none of another's in the workspace, his deleted ones or his in
// another workspace. The order is the query's whatever plan reads it: the list is read as planned and again with
// index scans off, the rows reaching the sort as written. The three at one sequence are written in neither id
// order, and the rows that reach the sort are too few for it to reorder equal ones, so an order by sequence alone
// fails here.
func TestTheListBySequenceThenByID(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newViews(t, pool)
	folder := favorite(60, domain.FavoriteFolder, w.workspace, nil, w.owner, 50)
	create(t, s, favorite(20, domain.FavoriteCycle, w.workspace, &w.project, w.owner, 100), favorite(10, domain.FavoriteProject, w.workspace, &w.project, w.owner, 100),
		favorite(30, domain.FavoriteView, w.workspace, nil, w.owner, 100), favorite(5, domain.FavoriteModule, w.workspace, &w.ops, w.owner, 200),
		folder, in(favorite(70, domain.FavoriteView, w.workspace, &w.ops, w.owner, 150), folder.ID),
		favorite(50, domain.FavoriteProject, w.workspace, &w.ops, w.owner, 300), favorite(80, domain.FavoriteProject, w.workspace, &w.project, w.other, 400),
		favorite(90, domain.FavoriteProject, w.elsewhere, &w.elsewhereProject, w.owner, 500))
	if err := s.DeleteFavorite(ctx, numberedID(50), now, w.owner); err != nil {
		t.Fatal(err)
	}
	want := []uuid.UUID{numberedID(5), numberedID(70), numberedID(30), numberedID(20), numberedID(10), numberedID(60)}
	for _, plan := range []string{"as planned", "without index scans"} {
		err := postgres.NewTxManager(pool, time.Second).WithinTx(ctx, func(txctx context.Context) error {
			if plan == "without index scans" {
				if _, err := postgres.DB(txctx, pool).Exec(txctx,
					"SELECT set_config('enable_indexscan', 'off', true), set_config('enable_bitmapscan', 'off', true)"); err != nil {
					return err
				}
			}
			if got, err := s.Favorites(txctx, w.owner, w.workspace); err != nil || !slices.Equal(favoriteIDs(got), want) {
				t.Errorf("%s: %v, %v; want %v", plan, favoriteIDs(got), err, want)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if got, err := s.Favorites(ctx, w.other, w.elsewhere); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("no favorites %#v, %v; want an empty list", got, err)
	}
}

// A new favorite's sequence is read from one's own undeleted favorites in the workspace (spec 2.5): another's in the
// workspace, his in another workspace and his deleted ones, each greater, are not his greatest; none is nil.
func TestTheGreatestSequence(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newViews(t, pool)
	if got, err := s.GreatestFavoriteSequence(ctx, w.owner, w.workspace); err != nil || got != nil {
		t.Fatalf("none: %v, %v; want nil", got, err)
	}
	create(t, s, favorite(1, domain.FavoriteProject, w.workspace, &w.project, w.owner, 100), favorite(2, domain.FavoriteFolder, w.workspace, nil, w.owner, -5),
		favorite(3, domain.FavoriteProject, w.workspace, &w.project, w.other, 900), favorite(4, domain.FavoriteProject, w.elsewhere, &w.elsewhereProject, w.owner, 1000),
		favorite(5, domain.FavoriteProject, w.workspace, &w.ops, w.owner, 800),
		favorite(6, domain.FavoriteFolder, w.elsewhere, nil, w.other, -7), favorite(7, domain.FavoriteFolder, w.elsewhere, nil, w.other, -3))
	if err := s.DeleteFavorite(ctx, numberedID(5), now, w.owner); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		user, workspace uuid.UUID
		want            float64
	}{{w.owner, w.workspace, 100}, {w.other, w.workspace, 900}, {w.owner, w.elsewhere, 1000}, {w.other, w.elsewhere, -3}} {
		if got, err := s.GreatestFavoriteSequence(ctx, tt.user, tt.workspace); err != nil || got == nil || *got != tt.want {
			t.Errorf("%s in %s: %v, %v; want %v", tt.user, tt.workspace, got, err, tt.want)
		}
	}
}

// One favors a target once (spec 2.3, 2.5): a favorite of a target he has an undeleted favorite of stores nothing,
// and its insertion answers the one he has, unchanged, its parent and sequence too. Another account's of the target,
// his of another kind with the same id, and his after he deleted his are new. A creation's last lock (spec 2.8) finds
// and holds FOR SHARE the one he has, a change of it (FOR NO KEY UPDATE) waiting and another's FOR SHARE not; his
// deleted one, another account's of the target, his of another kind with the same id and his of another target are
// not found, nor locked.
func TestAFavoriteOfATargetIsStoredOnce(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newViews(t, pool)
	folder := favorite(9, domain.FavoriteFolder, w.workspace, nil, w.owner, 65535)
	first := favorite(1, domain.FavoriteCycle, w.workspace, &w.project, w.owner, 100)
	create(t, s, folder, first)
	unchanged := stamps(t, pool, "user_favorites", first.ID)
	again := in(favorite(2, domain.FavoriteCycle, w.workspace, &w.project, w.owner, 999), folder.ID)
	again.EntityID = first.EntityID
	if got, created, err := s.InsertFavorite(ctx, again, w.owner); err != nil || created || !reflect.DeepEqual(got, first) {
		t.Fatalf("again: %+v, %v, %v; want the first, not created", got, created, err)
	}
	var rows int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM user_favorites WHERE entity_identifier = $1 OR id = $2", *first.EntityID, again.ID).Scan(&rows); err != nil || rows != 1 {
		t.Fatalf("rows of the target: %d, %v; want the first alone", rows, err)
	}
	if got := stamps(t, pool, "user_favorites", first.ID); !slices.Equal(got, unchanged) {
		t.Fatalf("the first %v; want %v", got, unchanged)
	}
	// The insertion passes over the target's conflict alone: an id taken is raised.
	clash := favorite(9, domain.FavoriteModule, w.workspace, &w.project, w.owner, 1)
	if got, created, err := s.InsertFavorite(ctx, clash, w.owner); !postgres.UniqueViolation(err, "user_favorites_pkey") || created {
		t.Fatalf("an id taken: %+v, %v, %v; want the primary key's violation", got, created, err)
	}
	hers := favorite(3, domain.FavoriteCycle, w.workspace, &w.project, w.other, 100)
	module := favorite(4, domain.FavoriteModule, w.workspace, &w.project, w.owner, 100)
	hers.EntityID, module.EntityID = first.EntityID, first.EntityID
	another := favorite(6, domain.FavoriteCycle, w.workspace, &w.project, w.owner, 100)
	create(t, s, hers, module, another)
	if err := s.DeleteFavorite(ctx, first.ID, now, w.owner); err != nil {
		t.Fatal(err)
	}
	others := []uuid.UUID{first.ID, hers.ID, module.ID, another.ID}
	within(t, pool, func(ctx context.Context) {
		if got, found, err := s.ShareFavoriteOfTarget(ctx, w.owner, domain.FavoriteCycle, *first.EntityID); found || err != nil {
			t.Fatalf("his deleted one, another's, of another kind or target: %+v, %v; want none", got, err)
		}
		for _, id := range others {
			if waits(t, pool, id, "FOR UPDATE") {
				t.Fatalf("%s locked; want none", id)
			}
		}
	})
	again.ID = numberedID(5)
	create(t, s, again)
	within(t, pool, func(ctx context.Context) {
		if got, found, err := s.ShareFavoriteOfTarget(ctx, w.owner, domain.FavoriteCycle, *first.EntityID); err != nil || !found || !reflect.DeepEqual(got, again) {
			t.Fatalf("his own: %+v, %v, %v; want %+v", got, found, err, again)
		}
		if !waits(t, pool, again.ID, "FOR NO KEY UPDATE") || waits(t, pool, again.ID, "FOR SHARE") {
			t.Fatal("his own is not held FOR SHARE")
		}
		for _, id := range others {
			if waits(t, pool, id, "FOR UPDATE") {
				t.Fatalf("%s locked; want his own alone", id)
			}
		}
	})
}

// One's folder names in a workspace differ in any case (spec 2.5, D41): a creation or a renaming to a name another
// of his undeleted folders there has is domain.ErrFavoriteFolderNameTaken, by the partial unique index, and stores
// nothing; the folder insert passes no conflict over. Another account's folder, his in another workspace and his
// deleted one do not take a name, nor does a folder's own name in another case.
func TestAFolderNameIsTakenInAnyCase(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newViews(t, pool)
	named := func(f domain.Favorite, name string) domain.Favorite { f.Name = name; return f }
	work := named(favorite(1, domain.FavoriteFolder, w.workspace, nil, w.owner, 1), "Work")
	home := named(favorite(5, domain.FavoriteFolder, w.workspace, nil, w.owner, 5), "Home")
	create(t, s, work, home, named(favorite(3, domain.FavoriteFolder, w.workspace, nil, w.other, 3), "Work"),
		named(favorite(4, domain.FavoriteFolder, w.elsewhere, nil, w.owner, 4), "Work"))
	taken := named(favorite(2, domain.FavoriteFolder, w.workspace, nil, w.owner, 2), "wORK")
	if got, created, err := s.InsertFavorite(ctx, taken, w.owner); !errors.Is(err, domain.ErrFavoriteFolderNameTaken) || created {
		t.Fatalf("a taken name: %+v, %v, %v; want %v", got, created, err, domain.ErrFavoriteFolderNameTaken)
	}
	unchanged := stamps(t, pool, "user_favorites", home.ID)
	if err := s.UpdateFavorite(ctx, named(home, "WORK"), w.owner); !errors.Is(err, domain.ErrFavoriteFolderNameTaken) {
		t.Fatalf("renamed to a taken name: %v; want %v", err, domain.ErrFavoriteFolderNameTaken)
	}
	var names []string
	if err := pool.QueryRow(ctx, `SELECT array_agg(name ORDER BY id) FROM user_favorites WHERE user_id = $1 AND workspace_id = $2`, w.owner, w.workspace).
		Scan(&names); err != nil || !slices.Equal(names, []string{"Work", "Home"}) || !slices.Equal(stamps(t, pool, "user_favorites", home.ID), unchanged) {
		t.Fatalf("his folders %v, %v; want Work and Home as they were", names, err)
	}
	if err := s.UpdateFavorite(ctx, named(work, "WORK"), w.owner); err != nil {
		t.Fatalf("its own name in another case: %v", err)
	}
	if err := s.DeleteFavorite(ctx, work.ID, now, w.owner); err != nil {
		t.Fatal(err)
	}
	create(t, s, named(favorite(6, domain.FavoriteFolder, w.workspace, nil, w.owner, 6), "work"))
}

// A deletion soft-deletes the favorite, and a folder's the favorites it holds with it, every row at the deletion's
// moment by its writer (spec 2.5); a favorite deleted before keeps its deletion, as does one deleted again. Another
// folder's favorites and those at the top stay.
func TestDeleteAFavoriteAndAFoldersFavorites(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newViews(t, pool)
	folder, other := favorite(10, domain.FavoriteFolder, w.workspace, nil, w.owner, 1), favorite(20, domain.FavoriteFolder, w.workspace, nil, w.owner, 2)
	create(t, s, folder, other, in(favorite(3, domain.FavoriteCycle, w.workspace, &w.project, w.owner, 3), folder.ID),
		in(favorite(2, domain.FavoriteModule, w.workspace, &w.project, w.owner, 4), folder.ID), in(favorite(1, domain.FavoriteView, w.workspace, nil, w.owner, 5), folder.ID),
		in(favorite(21, domain.FavoriteView, w.workspace, nil, w.owner, 6), other.ID), favorite(5, domain.FavoriteProject, w.workspace, &w.project, w.owner, 7))
	before, at := now.Add(-time.Minute), now
	if err := s.DeleteFavorite(ctx, numberedID(2), before, w.other); err != nil {
		t.Fatal(err)
	}
	kept := []uuid.UUID{numberedID(2), other.ID, numberedID(21), numberedID(5)}
	unchanged := stamps(t, pool, "user_favorites", kept...)
	if err := s.DeleteFavorite(ctx, folder.ID, at, w.owner); err != nil {
		t.Fatal(err)
	}
	deleted := []uuid.UUID{folder.ID, numberedID(1), numberedID(3)}
	var stamped int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM user_favorites WHERE id = ANY($1) AND deleted_at = $2 AND updated_at = $2 AND updated_by_id = $3",
		deleted, at, w.owner).Scan(&stamped); err != nil || stamped != 3 {
		t.Fatalf("the folder and its favorites deleted at %s by %s: %d of 3, %v", at, w.owner, stamped, err)
	}
	if got := stamps(t, pool, "user_favorites", kept...); !slices.Equal(got, unchanged) {
		t.Fatalf("deleted before, another folder's and at the top=%v; want %v", got, unchanged)
	}
	first := stamps(t, pool, "user_favorites", deleted...)
	if err := s.DeleteFavorite(ctx, folder.ID, now.Add(time.Hour), w.other); err != nil {
		t.Fatal(err)
	}
	if got := stamps(t, pool, "user_favorites", deleted...); !slices.Equal(got, first) {
		t.Fatalf("deleted again %v; want %v", got, first)
	}
}
