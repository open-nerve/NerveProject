package bootstrap

import (
	"context"
	"fmt"
	"net/http"
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

func TestActivityPagesFilterBeforeLimit(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	id := w.newWorkItemBy(t, "walt", w.web, `{"name":"Target"}`)
	hidden := w.newWorkItemIn(t, w.web, `{"name":"Hidden"}`)
	neighbor := w.newWorkItemIn(t, w.web, `{"name":"Neighbor"}`)
	initial := activityPage(t, w, "walt", activitiesOf(id)).Data[0]
	at := initial.CreatedAt.Add(time.Second)
	store := issuepg.New(w.pool)
	rows := []issueapp.ActivityRow{}
	want := []uuid.UUID{initial.ID}
	for n := 0; n < 102; n++ {
		field := "name"
		a := domain.Activity{Verb: "updated", Field: &field}
		if n%2 == 0 {
			field = "parent"
			a.NewIdentifier = &hidden
		} else {
			want = append(want, uuid.UUID{})
		}
		r := issueapp.ActivityRow{ID: uuid.NewV7(), WorkspaceID: w.acme, ProjectID: w.web, IssueID: id, ActorID: w.ids["alice"], At: at, Activity: a}
		if n%2 == 1 {
			want[len(want)-1] = r.ID
		}
		rows = append(rows, r)
	}
	// Same timestamps and reverse insertion force UUID ordering rather than insertion order.
	slices.Reverse(rows)
	if err := store.InsertActivities(pgtest.Soon(t), rows); err != nil {
		t.Fatal(err)
	}
	gone := rows[0]
	gone.ID = uuid.NewV7()
	if err := store.InsertActivities(pgtest.Soon(t), []issueapp.ActivityRow{gone}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issue_activities SET deleted_at=now() WHERE id=$1", gone.ID); err != nil {
		t.Fatal(err)
	}
	gone.ID, gone.IssueID = uuid.NewV7(), neighbor
	if err := store.InsertActivities(pgtest.Soon(t), []issueapp.ActivityRow{gone}); err != nil {
		t.Fatal(err)
	}
	for _, limit := range []int{1, 50, 100} {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			path := activitiesOf(id) + fmt.Sprintf("?limit=%d", limit)
			got := []uuid.UUID{}
			for n := 0; ; n++ {
				if n > 103 {
					t.Fatal("cursor did not advance")
				}
				page := activityPage(t, w, "walt", path)
				got = append(got, activityIDs(page.Data)...)
				if page.Next == nil {
					again := activityPage(t, w, "walt", path)
					if !slices.Equal(activityIDs(again.Data), activityIDs(page.Data)) || again.Next != nil {
						t.Fatal("last page request cursor not stable", page, again)
					}
					break
				}
				if len(page.Data) != limit {
					t.Fatal("filtered after limit", page)
				}
				var decoded domain.ActivityCursor
				if err := shared.DecodeCursor(*page.Next, &decoded); err != nil || decoded.ID != page.Data[len(page.Data)-1].ID {
					t.Fatal("cursor names hidden or probe row", decoded, err)
				}
				path = activitiesOf(id) + fmt.Sprintf("?limit=%d&cursor=%s", limit, *page.Next)
			}
			if !slices.Equal(got, want) {
				t.Fatal("visible order/filter", got, want)
			}
		})
	}
}

func TestActivityIncrementalLastPageRequestCursor(t *testing.T) {
	w := newIssueWorld(t)
	id := w.newWorkItemIn(t, w.web, `{"name":"Incremental"}`)
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+id.String(), `{"name":"Renamed"}`, 200)
	patchDescription(t, w, "alice", id, "<p>First</p>", false)
	cache := map[uuid.UUID]answeredActivity{}
	path := activitiesOf(id) + "?limit=2"
	for {
		page := activityPage(t, w, "alice", path)
		for _, r := range page.Data {
			cache[r.ID] = r
		}
		if page.Next == nil {
			break
		}
		path = activitiesOf(id) + "?limit=2&cursor=" + *page.Next
	}
	var description answeredActivity
	for _, r := range cache {
		if r.Field != nil && *r.Field == "description" {
			description = r
		}
	}
	if description.ID == (uuid.UUID{}) {
		t.Fatal("description fixture")
	}
	patchDescription(t, w, "alice", id, "<p>Merged</p>", false)
	w.send(t, "bob", http.MethodPatch, "/api/v0/issues/"+id.String(), `{"name":"Newest"}`, 200)
	page := activityPage(t, w, "alice", path)
	if page.Next != nil || len(page.Data) != 2 {
		t.Fatal("incremental terminal page", page)
	}
	for _, r := range page.Data {
		cache[r.ID] = r
	}
	if len(cache) != 4 || cache[description.ID].NewValue == nil || *cache[description.ID].NewValue != "<p>Merged</p>" || !cache[description.ID].CreatedAt.After(description.CreatedAt) {
		t.Fatal("incremental merge did not replace by id", cache)
	}
	ordered := []answeredActivity{}
	for _, r := range cache {
		ordered = append(ordered, r)
	}
	slices.SortFunc(ordered, func(a, b answeredActivity) int {
		if c := a.CreatedAt.Compare(b.CreatedAt); c != 0 {
			return c
		}
		return a.ID.Compare(b.ID)
	})
	full := activityPage(t, w, "alice", activitiesOf(id))
	if !slices.Equal(activityIDs(ordered), activityIDs(full.Data)) {
		t.Fatal("incremental reordering differs from full read", ordered, full)
	}
}

// activitySnapshotProjects verifies the historical catalog uses the first read's snapshot.
type activitySnapshotProjects struct {
	issueProjects
	inspect func(context.Context)
}

func (p activitySnapshotProjects) Historical(ctx context.Context, ws, actor uuid.UUID) ([]issueapp.VisibleProject, error) {
	p.inspect(ctx)
	return p.issueProjects.Historical(ctx, ws, actor)
}

type activitySnapshotRows struct {
	*issuepg.Store
	inspect func(context.Context)
}

func (s activitySnapshotRows) Activities(ctx context.Context, id, actor uuid.UUID, p []issueapp.VisibleProject, c *domain.ActivityCursor, n int) ([]issueapp.ActivityRecord, error) {
	s.inspect(ctx)
	return s.Store.Activities(ctx, id, actor, p, c, n)
}
func TestActivityPagesReadSnapshot(t *testing.T) {
	w := newIssueWorld(t)
	id := w.newWorkItemIn(t, w.web, `{"name":"Snapshot"}`)
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	ref := w.newWorkItemIn(t, ops, `{"name":"Reference"}`)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+id.String()+"/relations", fmt.Sprintf(`{"relation_type":"relates_to","issue_ids":[%q]}`, ref), 201)
	before := activityPage(t, w, "alice", activitiesOf(id))
	store := issuepg.New(w.pool)
	checks := 0
	inspect := func(ctx context.Context) {
		checks++
		var ro, isolation string
		if err := postgres.DB(ctx, w.pool).QueryRow(ctx, "SELECT current_setting('transaction_read_only'),current_setting('transaction_isolation')").Scan(&ro, &isolation); err != nil || ro != "on" || isolation != "repeatable read" {
			t.Fatal("read escaped snapshot", ro, isolation, err)
		}
	}
	projects := activitySnapshotProjects{issueProjects: issueProjects{projects: project.Provide(w.pool).IssueProjects}, inspect: inspect}
	wrapped := versionSnapshotStore{Store: store, inspect: inspect, afterIssue: func() {
		if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2", ops, w.ids["alice"]); err != nil {
			t.Fatal(err)
		}
		if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issue_activities SET deleted_at=now() WHERE id=$1", before.Data[1].ID); err != nil {
			t.Fatal(err)
		}
	}}
	locks := issueapp.NewLocks(projects, wrapped, authorizerOn(w.pool), versionReadOnly{}, versionReadOnly{})
	list := issueapp.NewListIssueActivities(issueapp.ListIssueActivitiesDeps{Locks: locks, Projects: projects, Activities: activitySnapshotRows{Store: store, inspect: inspect}, ReadTx: postgres.NewTxManager(w.pool, 2*time.Second, postgres.ReadOnlySnapshot())})
	ctx := shared.WithActor(pgtest.Soon(t), shared.Actor{UserID: w.ids["alice"]})
	page, err := list.Execute(ctx, id, nil, nil)
	if err != nil || checks != 3 || len(page.Data) != 2 || page.Data[1].ID != before.Data[1].ID {
		t.Fatal("historical directory/list left first visibility snapshot", page, checks, err)
	}
	fresh := activityPage(t, w, "alice", activitiesOf(id))
	if len(fresh.Data) != 1 {
		t.Fatal("concurrent commits missing", fresh)
	}
	tx, err := w.pool.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(pgtest.Soon(t), "SELECT id FROM projects WHERE id=$1 FOR UPDATE", w.web); err != nil {
		t.Fatal(err)
	}
	w.send(t, "bob", http.MethodGet, activitiesOf(id), "", 200)
}
