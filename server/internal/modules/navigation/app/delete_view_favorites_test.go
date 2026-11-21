package app_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ticking is a clock of the world that moves a second at each read, so that a second read is another moment.
type ticking struct {
	w     *world
	reads int
}

func (c *ticking) Now() time.Time {
	c.w.log("now")
	c.reads++
	return moment.Add(time.Duration(c.reads) * time.Second)
}

// failing is the world's FavoriteCascades answering err.
type failing struct {
	*world
	err error
}

func (f failing) DeleteFavoritesOfTarget(ctx context.Context, kind domain.FavoriteKind, id uuid.UUID, at time.Time, by uuid.UUID) error {
	if err := f.world.DeleteFavoritesOfTarget(ctx, kind, id, at, by); err != nil {
		return err
	}
	return f.err
}

// A view's deletion deletes every account's favorites of it with it (P3c spec 2.6, P3b spec 5): in its transaction,
// under its locks, right after the view, at the view's moment, the one read of the clock after the last lock, and by
// its writer, the caller, whoever owns the view: here alice, its admin, deletes bob's.
func TestDeletingAViewDeletesItsFavorites(t *testing.T) {
	for _, tt := range []struct {
		kind  string
		view  func(*world) domain.View
		grant shared.Grant
		calls []string
	}{
		{"project", webView, projectAdmin, []string{"view webView", "authorize alice view.delete acme/web",
			"begin write", "share workspace acme", "share project web", "lock view webView", "authorize alice view.delete acme/web",
			"now", "delete webView by alice", "delete favorites of view webView by alice", "end write"}},
		{"workspace", acmeView, workspaceAdmin, []string{"view acmeView", "authorize alice workspace_view.delete acme/-",
			"begin write", "share workspace acme", "lock view acmeView", "authorize alice workspace_view.delete acme/-",
			"now", "delete acmeView by alice", "delete favorites of view acmeView by alice", "end write"}},
	} {
		t.Run(tt.kind, func(t *testing.T) {
			w := newWorld()
			v := tt.view(w)
			decides(tt.grant, nil)(w)
			d := w.writes(noWords(w))
			d.Clock = &ticking{w: w}
			if err := app.NewDeleteView(d).Execute(as(t), v.ID); err != nil {
				t.Fatal(err)
			}
			wantCalls(t, w, tt.calls...)
			at := moment.Add(time.Second)
			if !reflect.DeepEqual(w.deleted, []deletion{{id: v.ID, by: alice, at: at}}) ||
				!reflect.DeepEqual(w.cascaded, []cascadeCall{{domain.FavoriteView, v.ID, at, alice}}) {
				t.Errorf("deleted %+v, favorites %+v; want the view and its favorites by alice at %s", w.deleted, w.cascaded, at)
			}
		})
	}
	failed := errors.New("the favorites' lock timed out")
	w := newWorld()
	v := own(w, webView(w))
	d := w.writes(noWords(w))
	d.Favorites = failing{w, failed}
	if err := app.NewDeleteView(d).Execute(as(t), v.ID); !errors.Is(err, failed) {
		t.Errorf("a failed deletion of its favorites = %v; want it, the view's deletion's", err)
	}
}
