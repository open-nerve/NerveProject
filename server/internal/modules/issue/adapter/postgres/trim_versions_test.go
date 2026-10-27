package postgresadapter_test

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// Every exclusion has over twenty versions. Heap order is the reverse of the
// id tie-break at the 20/21 boundary, and created_at reverses last_saved_at.
func TestTrimDescriptionVersionsKeepsLatestTwenty(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	actor := newAccount(t, pool, "retention@corp.com")
	workspace := newWorkspace(t, pool, "acme")
	live := newProject(t, pool, workspace, "LIVE")
	archived := newProject(t, pool, workspace, "ARCH")
	deleted := newProject(t, pool, workspace, "GONE")
	neighbor := newProject(t, pool, workspace, "OTHER")
	exec(t, pool, "UPDATE projects SET archived_at=$2 WHERE id=$1", archived.id, now)
	exec(t, pool, "UPDATE projects SET deleted_at=$2 WHERE id=$1", deleted.id, now)
	wantDeleted := map[uuid.UUID]bool{}
	for index, c := range []struct {
		p                       project
		archived, deleted, trim bool
	}{
		{live, false, false, true}, {live, true, false, true}, {live, false, true, false}, {archived, false, false, true}, {deleted, false, false, false}, {neighbor, false, false, false},
	} {
		issue := uuid.NewV7()
		newIssue(t, pool, c.p, issue, index+1, actor)
		if c.archived {
			exec(t, pool, "UPDATE issues SET archived_at=$2 WHERE id=$1", issue, now)
		}
		if c.deleted {
			exec(t, pool, "UPDATE issues SET deleted_at=$2 WHERE id=$1", issue, now)
		}
		ids := make([]uuid.UUID, 23)
		for i := range ids {
			ids[i] = uuid.NewV7()
		}
		slices.SortFunc(ids, uuid.UUID.Compare)
		for i := 0; i < 23; i++ {
			saved := now.Add(time.Duration(i) * time.Minute)
			if i == 2 {
				saved = now.Add(3 * time.Minute)
			}
			v := domain.DescriptionVersion{ID: ids[i], WorkspaceID: workspace, ProjectID: c.p.id, IssueID: issue, OwnedByID: actor, CreatedByID: actor, UpdatedByID: actor, HTML: "<p>Keep audit</p>", Text: "Keep audit", LastSavedAt: saved, CreatedAt: now.Add(-time.Duration(i) * time.Hour), UpdatedAt: now}
			if err := s.InsertDescriptionVersion(ctx, v); err != nil {
				t.Fatal(err)
			}
			if c.trim && i < 3 {
				wantDeleted[v.ID] = true
			}
		}
		for _, saved := range []time.Time{now.Add(-time.Hour), now.Add(time.Hour)} {
			v := domain.DescriptionVersion{ID: uuid.NewV7(), WorkspaceID: workspace, ProjectID: c.p.id, IssueID: issue, OwnedByID: actor, CreatedByID: actor, UpdatedByID: actor, LastSavedAt: saved, CreatedAt: now, UpdatedAt: now}
			if err := s.InsertDescriptionVersion(ctx, v); err != nil {
				t.Fatal(err)
			}
			exec(t, pool, "UPDATE issue_description_versions SET deleted_at=$2 WHERE id=$1", v.ID, now)
		}
	}
	before := rowsAsText(t, pool, "issue_description_versions")
	beforeAudit := audits(t, pool, "issue_description_versions")
	err := postgres.NewTxManager(pool, time.Second).WithinTx(ctx, func(ctx context.Context) error {
		// Prevent incidental index ordering from supplying a missing tie-break.
		if _, err := postgres.DB(ctx, pool).Exec(ctx, "SET LOCAL enable_indexscan=off; SET LOCAL enable_indexonlyscan=off; SET LOCAL enable_bitmapscan=off"); err != nil {
			return err
		}
		for _, p := range []struct {
			id   uuid.UUID
			want int
		}{{live.id, 6}, {archived.id, 3}} {
			n, err := s.TrimDescriptionVersions(ctx, p.id)
			if err != nil {
				return err
			}
			if n != p.want {
				t.Errorf("trim %s=%d want %d", p.id, n, p.want)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[uuid.UUID]string{}
	wantAudit := map[uuid.UUID]string{}
	for id, row := range before {
		if !wantDeleted[id] {
			want[id] = row
			wantAudit[id] = beforeAudit[id]
		}
	}
	if got := rowsAsText(t, pool, "issue_description_versions"); !reflect.DeepEqual(got, want) {
		t.Fatalf("remaining version ids/rows differ: got %d rows want %d; wrong ordering or changed excluded/audit rows", len(got), len(want))
	}
	if got := audits(t, pool, "issue_description_versions"); !reflect.DeepEqual(got, wantAudit) {
		t.Fatal("retention changed survivor/soft-deleted audit columns")
	}
}
