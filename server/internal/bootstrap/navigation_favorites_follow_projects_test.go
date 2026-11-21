package bootstrap

import (
	"context"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// followsAProjectsArchive is the cascade of a project's archive (spec 2.6,
// R3): Arc, alice's, whose member bob is, with its cycle, its module and
// bob's view of it, each favored by both, as Arc is. Alice archives it:
// the eight favorites are deleted at its archive's moment, by her; their
// folders, each one's Pins, and their favorites of Web's targets (kept) are
// not. Archived again, Arc is archived at a new moment, which deletes, by
// her at that moment, bob's favorite of Arc left behind by SQL meanwhile
// (leftFavorite), the eight keeping their own. Unarchived, it brings none
// back.
func followsAProjectsArchive(t *testing.T, w issueWorld) {
	arc := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Arc", "ARC")
	w.makeMember(t, "bob", arc, shared.RoleMember)
	w.turnCyclesOn(t, arc)
	w.turnModulesOn(t, arc)
	w.changeProject(t, arc, `{"issue_views_view":true}`)
	path := "/api/v0/projects/" + arc.String()
	var cycle cycleAnswer
	var module moduleAnswer
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, path+"/cycles", `{"name":"Arc's"}`, http.StatusCreated), &cycle)
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, path+"/modules", `{"name":"Arc's"}`, http.StatusCreated), &module)
	view, _ := w.newView(t, "bob", projectViewsPath(arc), `{"name":"Arc's"}`)
	var favorites []uuid.UUID
	for _, target := range []struct {
		kind string
		id   uuid.UUID
	}{{"project", arc}, {"cycle", cycle.ID}, {"module", module.ID}, {"view", view.ID}} {
		favorites = append(favorites, w.bothFavor(t, target.kind, target.id)...)
	}
	folders := []uuid.UUID{w.favorID(t, "alice", "acme", folderBody("Pins")), w.favorID(t, "bob", "acme", folderBody("Pins"))}
	rows := favoriteRows(t, w.pool, folders...)

	w.send(t, "alice", http.MethodPost, path+"/archive", "", http.StatusOK)
	at, by := stampOf(t, w.pool, "projects", "archived_at", arc)
	if by != w.ids["alice"] {
		t.Fatalf("Arc archived by %s; want alice", by)
	}
	favoritesDeletedAt(t, w.pool, at, by, favorites...)
	if got := favoriteRows(t, w.pool, folders...); got != rows {
		t.Errorf("Arc's archive changed the folders:\n%s\nwere\n%s", got, rows)
	}

	left := leftFavorite(t, w.pool, w.ids["bob"], "project", arc, arc)
	w.send(t, "alice", http.MethodPost, path+"/archive", "", http.StatusOK)
	again, by := stampOf(t, w.pool, "projects", "archived_at", arc)
	if !again.After(at) || by != w.ids["alice"] {
		t.Fatalf("Arc archived again at %s by %s; was at %s", again, by, at)
	}
	favoritesDeletedAt(t, w.pool, again, by, left)
	favoritesDeletedAt(t, w.pool, at, by, favorites...)
	w.send(t, "alice", http.MethodPost, path+"/unarchive", "", http.StatusOK)
	favoritesDeletedAt(t, w.pool, at, by, favorites...)
	for _, actor := range []string{"alice", "bob"} {
		if containsAny(w.listedFavoriteIDs(t, actor, "acme"), append(favorites, left)...) {
			t.Errorf("%s lists a favorite of Arc's, unarchived", actor)
		}
	}
}

// followsDeletionsThroughTheJob is the deletion job's favorites step (spec
// 2.6, R11): Doomed, alice's project in acme, whose member bob is, and its
// cycle, each favored by both; alice deletes it, and its job deletes the
// four at its deletion's moment, by her, as the event gave them. Gone,
// alice's workspace, holds her project Far with her view Held of it, her
// favorite of Far and her folder Pins with her favorite of Gone's own view
// in it. Its deletion's job for Far held (holdNavigationDeletion), the
// workspace's own step deletes the folder and the favorite in it, at its
// deletion's moment, by her, and not her favorite of Far, which Far's job
// deletes once released, at Far's. The start-time runs have run first, so
// none queues on the lock the test holds (P2a review §4).
func followsDeletionsThroughTheJob(t *testing.T, w issueWorld) {
	waitForStartupRuns(t, w.pool)
	doomed := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Doomed", "DMD")
	w.makeMember(t, "bob", doomed, shared.RoleMember)
	w.turnCyclesOn(t, doomed)
	var cycle cycleAnswer
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+doomed.String()+"/cycles", `{"name":"Doomed's"}`, http.StatusCreated), &cycle)
	favorites := append(w.bothFavor(t, "project", doomed), w.bothFavor(t, "cycle", cycle.ID)...)
	w.send(t, "alice", http.MethodDelete, "/api/v0/projects/"+doomed.String(), "", http.StatusNoContent)
	waitNavigationDeletion(t, w.pool, "project", doomed)
	at, by := stampOf(t, w.pool, "projects", "deleted_at", doomed)
	favoritesDeletedAt(t, w.pool, at, by, favorites...)

	var gone struct{ ID uuid.UUID }
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/workspaces", `{"name":"Gone","slug":"gone"}`, http.StatusCreated), &gone)
	far := createdProject(t, w.contract, w.base, w.tokens["alice"], "gone", "Far", "FAR")
	w.changeProject(t, far, `{"issue_views_view":true}`)
	w.newView(t, "alice", projectViewsPath(far), `{"name":"Held"}`)
	own, _ := w.newView(t, "alice", workspaceViewsPath("gone"), `{"name":"Gone's"}`)
	ofFar := w.favorID(t, "alice", "gone", favoriteBody("project", far))
	pins := w.favorID(t, "alice", "gone", folderBody("Pins"))
	ofGone := []uuid.UUID{pins, w.favorID(t, "alice", "gone", favoriteBody("view", own.ID, pins))}
	holder := holdNavigationDeletion(t, w.pool, far)
	w.send(t, "alice", http.MethodDelete, "/api/v0/workspaces/gone", "", http.StatusNoContent)
	waitNavigationDeletion(t, w.pool, "workspace", gone.ID)
	at, by = stampOf(t, w.pool, "workspaces", "deleted_at", gone.ID)
	favoritesDeletedAt(t, w.pool, at, by, ofGone...)
	favoritesLive(t, w.pool, ofFar)
	requestRelease(t, holder)
	waitNavigationDeletion(t, w.pool, "project", far)
	at, by = stampOf(t, w.pool, "projects", "deleted_at", far)
	favoritesDeletedAt(t, w.pool, at, by, ofFar)
}

// refusesAMalformedArchive is a project's archive whose event is malformed
// (spec 2.6, R13): Steady, alice's, whose member bob is, favored by both,
// archived through the project module's routes on the app's database,
// whose publisher is the app's dispatcher and its subscribers
// (projectDeletionPublisher) behind amendedEvents, which gives the archive
// a change, as no archive has. The favorites' cascade refuses it before
// any write and the archive fails, 500: Steady is not archived and the
// favorites are as they were. Published as written, the archive deletes
// them at its moment, by her.
func refusesAMalformedArchive(t *testing.T, w issueWorld) {
	steady := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Steady", "STD")
	w.makeMember(t, "bob", steady, shared.RoleMember)
	favorites := w.bothFavor(t, "project", steady)
	rows := favoriteRows(t, w.pool, favorites...)
	path := "/api/v0/projects/" + steady.String() + "/archive"
	route := func(amend bool) moduleRoute {
		return newProjectRouteWithPublisher(t, w.pool, postgres.NewTxManager(w.pool, 2*time.Second), authorizerOn(w.pool),
			workspace.Provide(w.pool, nil).WorkspaceMembers, amendedEvents{next: projectDeletionPublisher(t, w.pool), amend: amend})
	}
	route(true).answerWithin(t, w.contract, http.MethodPost, path, w.ids["alice"], "", http.StatusInternalServerError)
	var archived *time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT archived_at FROM projects WHERE id = $1", steady).Scan(&archived); err != nil || archived != nil {
		t.Fatalf("Steady after a malformed archive: archived at %v, %v; want not archived", archived, err)
	}
	if got := favoriteRows(t, w.pool, favorites...); got != rows {
		t.Fatalf("a malformed archive changed the favorites:\n%s\nwere\n%s", got, rows)
	}
	route(false).answerWithin(t, w.contract, http.MethodPost, path, w.ids["alice"], "", http.StatusOK)
	at, by := stampOf(t, w.pool, "projects", "archived_at", steady)
	favoritesDeletedAt(t, w.pool, at, by, favorites...)
}

// amendedEvents publishes each event through next, a project's archive
// given a change first when amend is set: the app's publisher with one
// defect.
type amendedEvents struct {
	next  shared.EventPublisher
	amend bool
}

func (p amendedEvents) Publish(ctx context.Context, e shared.Event) error {
	if p.amend && e.Type == "project.archived" {
		name := "Steady"
		e.Changes = []shared.Change{{Field: "name", Old: &name}}
	}
	return p.next.Publish(ctx, e)
}
