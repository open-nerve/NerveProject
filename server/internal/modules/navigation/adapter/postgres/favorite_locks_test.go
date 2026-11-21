package postgresadapter_test

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// waits reports whether a statement taking lock on the favorite id waits for another transaction's lock on it.
func waits(t *testing.T, pool *pgxpool.Pool, id uuid.UUID, lock string) bool {
	t.Helper()
	return pgtest.RowWaits(t, pool, "user_favorites", id, lock)
}

// within runs fn in a transaction of its own and fails the test when it fails.
func within(t *testing.T, pool *pgxpool.Pool, fn func(ctx context.Context)) {
	t.Helper()
	if err := postgres.NewTxManager(pool, time.Second).WithinTx(context.Background(), func(ctx context.Context) error { fn(ctx); return nil }); err != nil {
		t.Fatal(err)
	}
}

// inIDOrder runs lock, which locks favorites, in a transaction of its own while another holds the favorite middle
// FOR UPDATE, written between lowest and highest in the reverse of their id order: lock must wait on it holding
// lowest and not yet highest, as a statement that locks in id order does (M3 design 3.6 convention 5). Once the
// holder lets go and lock has run, held runs while lock's transaction still holds what it took.
func inIDOrder(t *testing.T, pool *pgxpool.Pool, lowest, middle, highest uuid.UUID, lock func(ctx context.Context) error, held func()) {
	t.Helper()
	ctx := context.Background()
	holder, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = holder.Rollback(ctx) }()
	if _, err := holder.Exec(ctx, "SELECT 1 FROM user_favorites WHERE id = $1 FOR UPDATE", middle); err != nil {
		t.Fatal(err)
	}
	locked, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	released := false
	defer func() {
		if !released {
			close(release)
		}
	}()
	go func() {
		done <- postgres.NewTxManager(pool, 10*time.Second).WithinTx(ctx, func(ctx context.Context) error {
			if err := lock(ctx); err != nil {
				return err
			}
			close(locked)
			<-release
			return nil
		})
	}()
	pgtest.WaitForLockWaitBehind(t, pool, holder.Conn().PgConn(), 10*time.Second)
	if !waits(t, pool, lowest, "FOR UPDATE") || waits(t, pool, highest, "FOR UPDATE") {
		t.Fatal("waiting on the middle row, it does not hold the lowest id alone; want the rows locked in id order")
	}
	if err := holder.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-locked:
		held()
	case err := <-done:
		t.Fatalf("the lock ended before it was held: %v", err)
	}
	released = true
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

// The parent a creation or a change names is locked FOR SHARE before the favorite's own row (spec 2.5, 2.8): one's
// own undeleted folder in the workspace is found and held so, a change of it (FOR NO KEY UPDATE) waiting and another's
// FOR SHARE not. Any other row is no parent, not found and not locked: another's folder, his folder of another
// workspace, his favorite of a target, his deleted folder; so a parent's lock is always a folder row's.
func TestShareAFolder(t *testing.T) {
	s, pool := newStore(t)
	w := newViews(t, pool)
	folder := favorite(1, domain.FavoriteFolder, w.workspace, nil, w.owner, 1)
	create(t, s, folder, favorite(2, domain.FavoriteFolder, w.workspace, nil, w.other, 2), favorite(3, domain.FavoriteFolder, w.elsewhere, nil, w.owner, 3),
		favorite(4, domain.FavoriteProject, w.workspace, &w.project, w.owner, 4), favorite(5, domain.FavoriteFolder, w.workspace, nil, w.owner, 5))
	if err := s.DeleteFavorite(context.Background(), numberedID(5), now, w.owner); err != nil {
		t.Fatal(err)
	}
	within(t, pool, func(ctx context.Context) {
		for _, id := range []uuid.UUID{numberedID(2), numberedID(3), numberedID(4), numberedID(5), uuid.NewV7()} {
			if found, err := s.ShareFolder(ctx, w.owner, w.workspace, id); found || err != nil {
				t.Fatalf("%s found %v, %v; want no parent", id, found, err)
			}
		}
		if found, err := s.ShareFolder(ctx, w.owner, w.workspace, folder.ID); !found || err != nil {
			t.Fatalf("his folder: found %v, %v", found, err)
		}
		if !waits(t, pool, folder.ID, "FOR NO KEY UPDATE") || waits(t, pool, folder.ID, "FOR SHARE") {
			t.Fatal("the folder is not held FOR SHARE")
		}
		for _, n := range []int{2, 3, 4, 5} {
			if waits(t, pool, numberedID(n), "FOR UPDATE") {
				t.Fatalf("%d locked; want no row but the parent locked", n)
			}
		}
	})
}

// One's favorite is read locked FOR UPDATE under a change's or a deletion's locks (spec 2.8): even a foreign key's
// check (FOR KEY SHARE) waits for it. Another's and a deleted one are not found and not locked.
func TestLockAFavorite(t *testing.T) {
	s, pool := newStore(t)
	w := newViews(t, pool)
	mine, hers, gone := favorite(1, domain.FavoriteCycle, w.workspace, &w.project, w.owner, 1), favorite(2, domain.FavoriteCycle, w.workspace, &w.project, w.other, 2),
		favorite(3, domain.FavoriteFolder, w.workspace, nil, w.owner, 3)
	create(t, s, mine, hers, gone)
	if err := s.DeleteFavorite(context.Background(), gone.ID, now, w.owner); err != nil {
		t.Fatal(err)
	}
	within(t, pool, func(ctx context.Context) {
		for _, id := range []uuid.UUID{hers.ID, gone.ID, uuid.NewV7()} {
			if got, found, err := s.LockFavorite(ctx, w.owner, id); found || err != nil {
				t.Fatalf("%s locked: %+v, %v", id, got, err)
			}
		}
		if got, found, err := s.LockFavorite(ctx, w.owner, mine.ID); err != nil || !found || !reflect.DeepEqual(got, mine) {
			t.Fatalf("locked %+v, %v, %v; want %+v", got, found, err, mine)
		}
		if !waits(t, pool, mine.ID, "FOR KEY SHARE") || waits(t, pool, hers.ID, "FOR UPDATE") || waits(t, pool, gone.ID, "FOR UPDATE") {
			t.Fatal("want his own row alone held FOR UPDATE")
		}
	})
}

// A folder's deletion locks the favorites it holds FOR UPDATE, in id order, after the folder's own (spec 2.8): it
// waits on the middle one holding the lowest alone, then holds each so. One deleted before, another folder's and one
// at the top are not locked, nor the folder by this statement.
func TestLockAFoldersFavoritesInIDOrder(t *testing.T) {
	s, pool := newStore(t)
	w := newViews(t, pool)
	folder, other := favorite(10, domain.FavoriteFolder, w.workspace, nil, w.owner, 1), favorite(20, domain.FavoriteFolder, w.workspace, nil, w.owner, 2)
	create(t, s, folder, other, in(favorite(3, domain.FavoriteCycle, w.workspace, &w.project, w.owner, 3), folder.ID),
		in(favorite(2, domain.FavoriteModule, w.workspace, &w.project, w.owner, 4), folder.ID), in(favorite(1, domain.FavoriteView, w.workspace, nil, w.owner, 5), folder.ID),
		in(favorite(4, domain.FavoriteView, w.workspace, &w.ops, w.owner, 6), folder.ID), in(favorite(21, domain.FavoriteView, w.workspace, nil, w.owner, 7), other.ID),
		favorite(5, domain.FavoriteProject, w.workspace, &w.project, w.owner, 8))
	if _, err := pool.Exec(context.Background(), "UPDATE user_favorites SET deleted_at = $2 WHERE id = $1", numberedID(4), now); err != nil {
		t.Fatal(err)
	}
	inIDOrder(t, pool, numberedID(1), numberedID(2), numberedID(3), func(ctx context.Context) error { return s.LockFolderChildren(ctx, folder.ID) }, func() {
		for _, n := range []int{1, 2, 3} {
			if !waits(t, pool, numberedID(n), "FOR KEY SHARE") {
				t.Errorf("%d not held FOR UPDATE", n)
			}
		}
		for _, id := range []uuid.UUID{numberedID(4), numberedID(21), numberedID(5), folder.ID, other.ID} {
			if waits(t, pool, id, "FOR UPDATE") {
				t.Errorf("%s locked; want the folder's undeleted favorites alone", id)
			}
		}
	})
}

// The cascades (spec 2.6): of a target's deletion or archiving, every account's undeleted favorites of the target, of
// its kind alone; of a project's archiving, every account's undeleted favorites whose project it is, the project's
// own, its cycles', modules' and views'. Each at the cascade's moment by its actor. One deleted before keeps its
// deletion, and the others stay as they were: a folder holding one, another target's, another project's, a
// workspace's view's.
func TestTheCascades(t *testing.T) {
	for _, tt := range []struct {
		name         string
		cascade      func(*postgresadapter.Store, context.Context, views, uuid.UUID, time.Time, uuid.UUID) error
		mine, others func(w views, target uuid.UUID) []domain.Favorite
		before       domain.FavoriteKind // the kind of the actor's favorite of the target, deleted before
	}{
		{"of a target", func(s *postgresadapter.Store, ctx context.Context, _ views, target uuid.UUID, at time.Time, by uuid.UUID) error {
			return s.DeleteFavoritesOfTarget(ctx, domain.FavoriteCycle, target, at, by)
		}, func(w views, target uuid.UUID) []domain.Favorite {
			return []domain.Favorite{of(favorite(3, domain.FavoriteCycle, w.workspace, &w.project, w.owner, 3), target),
				in(of(favorite(1, domain.FavoriteCycle, w.workspace, &w.project, w.other, 1), target), numberedID(8))}
		}, func(w views, target uuid.UUID) []domain.Favorite {
			return []domain.Favorite{favorite(8, domain.FavoriteFolder, w.workspace, nil, w.other, 8), of(favorite(4, domain.FavoriteModule, w.workspace, &w.project, w.owner, 4), target),
				favorite(5, domain.FavoriteCycle, w.workspace, &w.project, w.owner, 5)}
		}, domain.FavoriteCycle},
		{"of a project", func(s *postgresadapter.Store, ctx context.Context, w views, _ uuid.UUID, at time.Time, by uuid.UUID) error {
			return s.DeleteFavoritesOfProject(ctx, w.project, at, by)
		}, func(w views, _ uuid.UUID) []domain.Favorite {
			return []domain.Favorite{favorite(13, domain.FavoriteProject, w.workspace, &w.project, w.owner, 13), favorite(11, domain.FavoriteProject, w.workspace, &w.project, w.other, 11),
				in(favorite(12, domain.FavoriteCycle, w.workspace, &w.project, w.owner, 12), numberedID(19)), favorite(14, domain.FavoriteModule, w.workspace, &w.project, w.owner, 14),
				favorite(15, domain.FavoriteView, w.workspace, &w.project, w.other, 15)}
		}, func(w views, _ uuid.UUID) []domain.Favorite {
			return []domain.Favorite{favorite(19, domain.FavoriteFolder, w.workspace, nil, w.owner, 19), favorite(17, domain.FavoriteProject, w.workspace, &w.ops, w.owner, 17),
				favorite(18, domain.FavoriteView, w.workspace, nil, w.owner, 18), favorite(20, domain.FavoriteProject, w.elsewhere, &w.elsewhereProject, w.owner, 20)}
		}, domain.FavoriteView},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s, pool := newStore(t)
			ctx := context.Background()
			w := newViews(t, pool)
			actor, target := uuid.NewV7(), uuid.NewV7()
			exec(t, pool, "INSERT INTO users (id, email, password, display_name) VALUES ($1, 'cid@corp.com', 'x', 'cid')", actor)
			mine, others := tt.mine(w, target), tt.others(w, target)
			before := of(favorite(2, tt.before, w.workspace, &w.project, actor, 2), target)
			create(t, s, append(append(others, mine...), before)...)
			if err := s.DeleteFavorite(ctx, before.ID, now.Add(-time.Minute), w.other); err != nil {
				t.Fatal(err)
			}
			kept := append(favoriteIDs(others), before.ID)
			unchanged := stamps(t, pool, "user_favorites", kept...)
			at := now.Add(time.Hour)
			if err := tt.cascade(s, ctx, w, target, at, actor); err != nil {
				t.Fatal(err)
			}
			var stamped int
			if err := pool.QueryRow(ctx, "SELECT count(*) FROM user_favorites WHERE id = ANY($1) AND deleted_at = $2 AND updated_at = $2 AND updated_by_id = $3",
				favoriteIDs(mine), at, actor).Scan(&stamped); err != nil || stamped != len(mine) {
				t.Fatalf("deleted at %s by %s: %d of %d, %v", at, actor, stamped, len(mine), err)
			}
			if got := stamps(t, pool, "user_favorites", kept...); !slices.Equal(got, unchanged) {
				t.Fatalf("deleted before and the others=%v; want %v", got, unchanged)
			}
		})
	}
}

// of is f, a favorite of a target, of target.
func of(f domain.Favorite, target uuid.UUID) domain.Favorite {
	f.EntityID = &target
	return f
}

// Each cascade locks its rows in id order (M3 design 3.6 convention 5, spec 2.8), every account's: written in the
// reverse of their id order with the middle one held, it waits on it holding the lowest alone.
func TestTheCascadesLockInIDOrder(t *testing.T) {
	for _, tt := range []struct {
		name    string
		kinds   []domain.FavoriteKind
		cascade func(*postgresadapter.Store, context.Context, views, uuid.UUID) error
	}{
		{"of a target", []domain.FavoriteKind{domain.FavoriteModule, domain.FavoriteModule, domain.FavoriteModule},
			func(s *postgresadapter.Store, ctx context.Context, w views, target uuid.UUID) error {
				return s.DeleteFavoritesOfTarget(ctx, domain.FavoriteModule, target, now, w.owner)
			}},
		{"of a project", []domain.FavoriteKind{domain.FavoriteModule, domain.FavoriteCycle, domain.FavoriteProject},
			func(s *postgresadapter.Store, ctx context.Context, w views, _ uuid.UUID) error {
				return s.DeleteFavoritesOfProject(ctx, w.project, now, w.owner)
			}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s, pool := newStore(t)
			w := newViews(t, pool)
			third, target := uuid.NewV7(), uuid.NewV7()
			exec(t, pool, "INSERT INTO users (id, email, password, display_name) VALUES ($1, 'cid@corp.com', 'x', 'cid')", third)
			for i, user := range []uuid.UUID{w.owner, w.other, third} {
				f := favorite(3-i, tt.kinds[i], w.workspace, &w.project, user, 1)
				if f.Kind != domain.FavoriteProject {
					f = of(f, target)
				}
				create(t, s, f)
			}
			inIDOrder(t, pool, numberedID(1), numberedID(2), numberedID(3), func(ctx context.Context) error { return tt.cascade(s, ctx, w, target) }, func() {})
		})
	}
}
