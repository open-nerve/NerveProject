package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	issuepg "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type answeredVersion struct {
	ID          uuid.UUID `json:"id"`
	IssueID     uuid.UUID `json:"issue_id"`
	OwnedByID   uuid.UUID `json:"owned_by_id"`
	LastSavedAt time.Time `json:"last_saved_at"`
	CreatedAt   time.Time `json:"created_at"`
	HTML        string    `json:"description_html"`
}
type answeredVersionPage struct {
	Data []answeredVersion `json:"data"`
	Next *string           `json:"next_cursor"`
}

func versionsOf(id uuid.UUID) string {
	return "/api/v0/issues/" + id.String() + "/description-versions"
}
func versionPath(id uuid.UUID) string { return "/api/v0/issue-description-versions/" + id.String() }
func versionIDs(rows []answeredVersion) []uuid.UUID {
	ids := make([]uuid.UUID, len(rows))
	for n, v := range rows {
		ids[n] = v.ID
	}
	return ids
}

func TestDescriptionVersionsPages(t *testing.T) {
	w := newIssueWorld(t)
	id := w.newWorkItemIn(t, w.web, `{"name":"Versions","description_html":"<p>First</p>"}`)
	other := w.newWorkItemIn(t, w.web, `{"name":"Neighbor"}`)
	first := savedVersion(t, w, id)
	versions := []uuid.UUID{first.ID}
	// Alternating writers always create a version; insertion order disagrees with DESC.
	for n := 1; n < 101; n++ {
		actor := "alice"
		if n%2 == 1 {
			actor = "bob"
		}
		patchDescription(t, w, actor, id, fmt.Sprintf("<p>Version %d</p>", n), false)
		versions = append(versions, savedVersion(t, w, id).ID)
	}
	at := first.LastSavedAt
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issue_description_versions SET last_saved_at=$2 WHERE issue_id=$1", id, at); err != nil {
		t.Fatal(err)
	}
	slices.SortFunc(versions, func(a, b uuid.UUID) int { return b.Compare(a) })
	// Neighbor and deleted rows surround the page position, so neither filter can disappear.
	neighbor := savedVersion(t, w, other)
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issue_description_versions SET last_saved_at=$2 WHERE id=$1", neighbor.ID, at); err != nil {
		t.Fatal(err)
	}
	gone := versions[50]
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issue_description_versions SET deleted_at=$2 WHERE id=$1", gone, at); err != nil {
		t.Fatal(err)
	}
	versions = slices.Delete(versions, 50, 51)
	for _, limit := range []int{1, 50, 100} {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			collected := []uuid.UUID{}
			path := versionsOf(id) + fmt.Sprintf("?limit=%d", limit)
			for pages := 0; ; pages++ {
				if pages > 101 {
					t.Fatal("cursor did not advance")
				}
				var page answeredVersionPage
				decodeAnswer(t, w.send(t, "alice", http.MethodGet, path, "", 200), &page)
				collected = append(collected, versionIDs(page.Data)...)
				if len(page.Data) != limit {
					t.Fatal("page size", len(page.Data), limit)
				}
				if page.Next == nil {
					break
				}
				path = versionsOf(id) + fmt.Sprintf("?limit=%d&cursor=%s", limit, *page.Next)
			}
			if !slices.Equal(collected, versions) {
				t.Fatal("version order/filter", collected, versions)
			}
		})
	}
	cursor, err := shared.EncodeCursor(domain.DescriptionVersionCursor{LastSavedAt: at, ID: versions[len(versions)-1]})
	if err != nil {
		t.Fatal(err)
	}
	var empty answeredVersionPage
	decodeAnswer(t, w.send(t, "alice", http.MethodGet, versionsOf(id)+"?cursor="+cursor, "", 200), &empty)
	if empty.Data == nil || len(empty.Data) != 0 || empty.Next != nil {
		t.Fatal("empty terminal page", empty)
	}
	// Someone else's cursor is only a position and cannot broaden the issue's rows.
	foreign, _ := shared.EncodeCursor(domain.DescriptionVersionCursor{LastSavedAt: at.Add(time.Hour), ID: neighbor.ID})
	var page answeredVersionPage
	decodeAnswer(t, w.send(t, "alice", http.MethodGet, versionsOf(id)+"?limit=100&cursor="+foreign, "", 200), &page)
	if !slices.Equal(versionIDs(page.Data), versions) || page.Next != nil {
		t.Fatal("foreign cursor expanded rows", page)
	}
	// Save time dominates a contrary id order, and a page crosses the time boundary.
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issue_description_versions SET last_saved_at=$2 WHERE id=$1", first.ID, at.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodGet, versionsOf(id)+"?limit=1", "", 200), &page)
	if len(page.Data) != 1 || page.Data[0].ID != first.ID || page.Next == nil {
		t.Fatal("save time must precede id", page)
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodGet, versionsOf(id)+"?limit=100&cursor="+*page.Next, "", 200), &page)
	if !slices.Equal(versionIDs(page.Data), versions[:len(versions)-1]) || page.Next != nil {
		t.Fatal("time boundary cursor", page)
	}

	var detail answeredVersion
	decodeAnswer(t, w.send(t, "alice", http.MethodGet, versionPath(first.ID), "", 200), &detail)
	if detail.HTML != "<p>First</p>" || detail.OwnedByID != w.ids["alice"] || detail.IssueID != id || !detail.LastSavedAt.Equal(at.Add(time.Hour)) || !detail.CreatedAt.Equal(first.CreatedAt) {
		t.Fatal(detail)
	}
}

func TestDescriptionVersionsVisibility(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	own := w.newWorkItemIn(t, w.web, `{"name":"Own"}`)
	hidden := w.newWorkItemIn(t, w.web, `{"name":"Hidden"}`)
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issues SET created_by_id=$2 WHERE id=$1", own, w.ids["walt"]); err != nil {
		t.Fatal(err)
	}
	ownV, hiddenV := savedVersion(t, w, own), savedVersion(t, w, hidden)
	w.send(t, "walt", http.MethodGet, versionsOf(own), "", 200)
	w.send(t, "walt", http.MethodGet, versionPath(ownV.ID), "", 200)
	w.send(t, "walt", http.MethodGet, versionsOf(hidden), "", 404)
	var hiddenReply, missingReply map[string]any
	hiddenReq := newRequest(t, http.MethodGet, w.base+versionPath(hiddenV.ID), w.tokens["walt"], nil)
	hiddenRes, hiddenBody := send(t, hiddenReq)
	missingReq := newRequest(t, http.MethodGet, w.base+versionPath(uuid.NewV7()), w.tokens["walt"], nil)
	missingRes, missingBody := send(t, missingReq)
	if hiddenRes.StatusCode != 404 || missingRes.StatusCode != 404 {
		t.Fatal("hidden/missing version status", hiddenRes.StatusCode, missingRes.StatusCode)
	}
	decodeAnswer(t, string(hiddenBody), &hiddenReply)
	decodeAnswer(t, string(missingBody), &missingReply)
	if !reflect.DeepEqual(hiddenReply, missingReply) || hiddenReply["code"] != "issue.description_version_not_found" {
		t.Fatal("version leak", hiddenReply, missingReply)
	}
	w.contract.CheckResponse(t, hiddenReq, hiddenRes)
	w.contract.CheckResponse(t, missingReq, missingRes)
	// The same project guest without workspace-admin authority cannot read even his own.
	w.send(t, "alice", http.MethodPatch, "/api/v0/workspace-members/"+queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["walt"])[0].String(), `{"role":15}`, 200)
	w.send(t, "walt", http.MethodGet, versionsOf(own), "", 403)
	w.send(t, "walt", http.MethodGet, versionPath(ownV.ID), "", 403)
	w.send(t, "walt", http.MethodGet, versionPath(hiddenV.ID), "", 404)
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issue_description_versions SET deleted_at=now() WHERE id=$1", ownV.ID); err != nil {
		t.Fatal(err)
	}
	var deletedReply map[string]any
	decodeAnswer(t, w.send(t, "alice", http.MethodGet, versionPath(ownV.ID), "", 404), &deletedReply)
	if !reflect.DeepEqual(deletedReply, missingReply) {
		t.Fatal("deleted version leak", deletedReply, missingReply)
	}
	// Both archived levels remain readable.
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issues SET archived_at=now() WHERE id=$1", hidden); err != nil {
		t.Fatal(err)
	}
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", "", 200)
	w.send(t, "bob", http.MethodGet, versionsOf(hidden), "", 200)
	w.send(t, "bob", http.MethodGet, versionPath(hiddenV.ID), "", 200)
}

// versionSnapshotStore observes the first issue read before a concurrent commit.
type versionSnapshotStore struct {
	*issuepg.Store
	afterIssue func()
	inspect    func(context.Context)
}

func (s versionSnapshotStore) GetIssue(ctx context.Context, id uuid.UUID) (domain.Issue, bool, error) {
	if s.inspect != nil {
		s.inspect(ctx)
	}
	v, found, err := s.Store.GetIssue(ctx, id)
	if s.afterIssue != nil {
		s.afterIssue()
	}
	return v, found, err
}

type versionReadOnly struct{}

func (versionReadOnly) Enter(context.Context, ...uuid.UUID) (func(), error) {
	return nil, errors.New("read used write gate")
}
func (versionReadOnly) WithinTx(context.Context, func(context.Context) error) error {
	return errors.New("read used write transaction")
}

func TestDescriptionVersionsReadSnapshot(t *testing.T) {
	w := newIssueWorld(t)
	id := w.newWorkItem(t)
	v := savedVersion(t, w, id)
	store := issuepg.New(w.pool)
	projects := issueProjects{projects: project.Provide(w.pool).IssueProjects}
	changed := false
	wrapped := versionSnapshotStore{Store: store, inspect: func(ctx context.Context) {
		var readOnly, isolation string
		if err := postgres.DB(ctx, w.pool).QueryRow(ctx, "SELECT current_setting('transaction_read_only'),current_setting('transaction_isolation')").Scan(&readOnly, &isolation); err != nil || readOnly != "on" || isolation != "repeatable read" {
			t.Fatal("first visibility read outside independent read snapshot", readOnly, isolation, err)
		}
	}, afterIssue: func() {
		changed = true
		if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issue_description_versions SET deleted_at=now() WHERE id=$1", v.ID); err != nil {
			t.Fatal(err)
		}
	}}
	locks := issueapp.NewLocks(projects, wrapped, authorizerOn(w.pool), versionReadOnly{}, versionReadOnly{})
	list := issueapp.NewListDescriptionVersions(issueapp.ListDescriptionVersionsDeps{Locks: locks, Versions: store, ReadTx: postgres.NewTxManager(w.pool, 2*time.Second, postgres.ReadOnlySnapshot())})
	ctx := shared.WithActor(pgtest.Soon(t), shared.Actor{UserID: w.ids["alice"]})
	page, err := list.Execute(ctx, id, nil, nil)
	if err != nil || !changed || len(page.Data) != 1 || page.Data[0].ID != v.ID {
		t.Fatal("page left first visibility snapshot", page, err)
	}
	if _, found, err := store.GetDescriptionVersion(ctx, v.ID); err != nil || found {
		t.Fatal("concurrent deletion did not commit", found, err)
	}
	// A detail has no transaction dependency and takes no write gate or row lock.
	fresh := w.newWorkItem(t)
	detail := savedVersion(t, w, fresh)
	locks = issueapp.NewLocks(projects, store, authorizerOn(w.pool), versionReadOnly{}, versionReadOnly{})
	get := issueapp.NewGetDescriptionVersion(issueapp.GetDescriptionVersionDeps{Locks: locks, Versions: store})
	got, err := get.Execute(ctx, detail.ID)
	if err != nil || got.ID != detail.ID {
		t.Fatal("detail used write resources", got, err)
	}
	tx, err := w.pool.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(pgtest.Soon(t), "SELECT id FROM projects WHERE id=$1 FOR UPDATE", w.web); err != nil {
		t.Fatal(err)
	}
	w.send(t, "bob", http.MethodGet, versionsOf(fresh), "", 200)
	w.send(t, "bob", http.MethodGet, versionPath(detail.ID), "", 200)
}
