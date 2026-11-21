package bootstrap

import (
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// filtersOnRead is the read-time filter's cell (M7 design 3.8, spec 2.2,
// 2.5): bob's favorites of Web, Sprint, Core, alice's view and his own
// view of Web are kept as rows when he leaves Web, and not listed; his
// change and deletion of one answer as an id no favorite has; joined
// again, he lists them as before. Made Web's guest, who sees only his own
// (Grant.OwnOnly), his favorite of alice's view alone is not listed, and
// answers so; Web's member again, he lists it.
func filtersOnRead(t *testing.T, w favoritesWorld, bobs map[string]favoriteHTTP) {
	own := w.favorID(t, "bob", "acme", favoriteBody("view", w.bobsView))
	webs := []uuid.UUID{bobs["project"].ID, bobs["cycle"].ID, bobs["module"].ID, bobs["view"].ID, own}
	before := w.listedFavorites(t, "bob", "acme")
	if listed := favoriteIDsOf(before); !containsAll(listed, webs...) {
		t.Fatalf("bob lists %v; want %v among them", listed, webs)
	}
	w.send(t, "bob", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/leave", "", http.StatusNoContent)
	listed := w.listedFavoriteIDs(t, "bob", "acme")
	if containsAny(listed, webs...) || !slices.Contains(listed, bobs["workspace view"].ID) || !slices.Contains(listed, w.bobsFolders["Daily"]) {
		t.Errorf("bob lists %v out of Web; want none of %v, his folders and acme's view", listed, webs)
	}
	favoritesLive(t, w.pool, webs...)
	w.favoriteAnsweredAsAbsent(t, "bob", bobs["project"].ID)
	w.send(t, "bob", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/join", "", http.StatusOK)
	if got := w.listedFavorites(t, "bob", "acme"); !reflect.DeepEqual(got, before) {
		t.Errorf("bob lists %+v in Web again; want %+v", got, before)
	}

	membership := projectMemberships(t, w.pool, w.ids["bob"], w.web)[0]
	w.send(t, "alice", http.MethodPatch, "/api/v0/project-members/"+membership.String(), `{"role":5}`, http.StatusOK)
	listed = w.listedFavoriteIDs(t, "bob", "acme")
	if slices.Contains(listed, bobs["view"].ID) || !containsAll(listed, bobs["project"].ID, bobs["cycle"].ID, bobs["module"].ID, own) {
		t.Errorf("bob, Web's guest, lists %v; want all of %v but alice's view's %s", listed, webs, bobs["view"].ID)
	}
	w.favoriteAnsweredAsAbsent(t, "bob", bobs["view"].ID)
	w.send(t, "alice", http.MethodPatch, "/api/v0/project-members/"+membership.String(), `{"role":15}`, http.StatusOK)
	if got := w.listedFavorites(t, "bob", "acme"); !reflect.DeepEqual(got, before) {
		t.Errorf("bob lists %+v, Web's member again; want %+v", got, before)
	}
}

// favorsAsTheGuestRuleSays is H19's project-level cells, which the matrix
// of workspace columns cannot hold (spec 2.5): walt, made Web's guest, who
// sees only his own, favors Web, Sprint, Core and his own view (seeded: a
// guest makes no view), but neither bob's view nor alice's, each the one
// 422 of a target he may not favor; so adam, acme's admin who is Web's
// guest, while Web does not show its guests everything. Shown everything,
// each favors another's view; no longer shown it, neither lists it, the
// rows kept. Removed from Web, walt lists none of his favorites of it.
func favorsAsTheGuestRuleSays(t *testing.T, w favoritesWorld) {
	w.makeMember(t, "walt", w.web, shared.RoleGuest)
	walts := seedView(t, w.pool, w.acme, &w.web, "Walt's", w.ids["walt"], time.Now().UTC().Truncate(time.Microsecond))
	var mine []uuid.UUID
	for _, body := range []string{favoriteBody("project", w.web), favoriteBody("cycle", w.sprint), favoriteBody("module", w.core), favoriteBody("view", walts)} {
		mine = append(mine, w.favorID(t, "walt", "acme", body))
	}
	w.refusedAlike(t, "walt", "acme", w.noTarget(t, "walt", "acme"), favoriteBody("view", w.bobsView), favoriteBody("view", w.alicesView))
	w.favorID(t, "adam", "acme", favoriteBody("project", w.web))
	w.refusedAlike(t, "adam", "acme", w.noTarget(t, "adam", "acme"), favoriteBody("view", w.alicesView))

	w.changeProject(t, w.web, `{"guest_view_all_features":true}`)
	waltsOfBobs := w.favorID(t, "walt", "acme", favoriteBody("view", w.bobsView))
	adamsOfAlices := w.favorID(t, "adam", "acme", favoriteBody("view", w.alicesView))
	w.changeProject(t, w.web, `{"guest_view_all_features":false}`)
	for actor, other := range map[string]uuid.UUID{"walt": waltsOfBobs, "adam": adamsOfAlices} {
		if slices.Contains(w.listedFavoriteIDs(t, actor, "acme"), other) {
			t.Errorf("%s lists his favorite %s of another's view, no longer shown it", actor, other)
		}
	}
	favoritesLive(t, w.pool, waltsOfBobs, adamsOfAlices)
	if got := w.listedFavoriteIDs(t, "walt", "acme"); !containsAll(got, mine...) {
		t.Errorf("walt lists %v; want %v among them", got, mine)
	}

	membership := projectMemberships(t, w.pool, w.ids["walt"], w.web)[0]
	w.send(t, "alice", http.MethodDelete, "/api/v0/project-members/"+membership.String(), "", http.StatusNoContent)
	if got := w.listedFavoriteIDs(t, "walt", "acme"); containsAny(got, append(mine, waltsOfBobs)...) {
		t.Errorf("walt, removed from Web, lists %v", got)
	}
	favoritesLive(t, w.pool, mine...)
}

// ignoresTheToggles is the toggles' cell (spec 2.5, R5): with Web's
// cycles, modules and views turned off, bob lists his favorites of them
// still, and favors Later, Edge and alice's view Off, made before; turned
// on again.
func ignoresTheToggles(t *testing.T, w favoritesWorld, bobs map[string]favoriteHTTP) {
	later := w.newCycle(t, "alice", `{"name":"Later"}`)
	edge := w.newModule(t, `{"name":"Edge"}`)
	off, _ := w.newView(t, "alice", projectViewsPath(w.web), `{"name":"Off"}`)
	w.changeProject(t, w.web, `{"cycle_view":false,"module_view":false,"issue_views_view":false}`)
	made := []uuid.UUID{w.favorID(t, "bob", "acme", favoriteBody("cycle", later)), w.favorID(t, "bob", "acme", favoriteBody("module", edge)),
		w.favorID(t, "bob", "acme", favoriteBody("view", off.ID))}
	if got := w.listedFavoriteIDs(t, "bob", "acme"); !containsAll(got, append(made, bobs["cycle"].ID, bobs["module"].ID, bobs["view"].ID)...) {
		t.Errorf("bob lists %v, Web's cycles, modules and views turned off", got)
	}
	w.changeProject(t, w.web, `{"cycle_view":true,"module_view":true,"issue_views_view":true}`)
}

// listsNoArchivedTarget is the cell of a favorite whose target is archived
// (M7 design 3.8, spec 2.5): bob's favorites of Old, Web's archived cycle,
// and of Shelf, his archived project, left behind by SQL as a creation and
// an archive interleaved would leave them (leftFavorite), are undeleted
// rows he does not list. It returns them.
func listsNoArchivedTarget(t *testing.T, w favoritesWorld) []uuid.UUID {
	left := []uuid.UUID{leftFavorite(t, w.pool, w.ids["bob"], "cycle", w.old, w.web), leftFavorite(t, w.pool, w.ids["bob"], "project", w.shelf, w.shelf)}
	if got := w.listedFavoriteIDs(t, "bob", "acme"); containsAny(got, left...) {
		t.Errorf("bob lists %v; want none of his favorites %v of archived targets", got, left)
	}
	favoritesLive(t, w.pool, left...)
	return left
}
