package postgresadapter_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func TestDescriptionVersions(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	a, b := newAccount(t, pool, "a@corp.com"), newAccount(t, pool, "b@corp.com")
	p := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	id, neighbor := uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, p, id, 1, a)
	newIssue(t, pool, p, neighbor, 2, a)
	if _, found, err := s.LatestDescriptionVersion(ctx, id); err != nil || found {
		t.Fatal(found, err)
	}
	base := domain.DescriptionVersion{ID: uuid.NewV7(), WorkspaceID: p.workspace, ProjectID: p.id, IssueID: id, OwnedByID: a, CreatedByID: a, UpdatedByID: a, HTML: "<p>First</p>", Text: "First", LastSavedAt: now, CreatedAt: now.Add(-time.Hour), UpdatedAt: now}
	if err := s.InsertDescriptionVersion(ctx, base); err != nil {
		t.Fatal(err)
	}
	got, found, err := s.LatestDescriptionVersion(ctx, id)
	if err != nil || !found || !reflect.DeepEqual(got, base) {
		t.Fatal(got, base, err)
	}
	merged := base
	merged.HTML, merged.Text, merged.UpdatedByID, merged.LastSavedAt, merged.UpdatedAt = "<p>Next</p>", "Next", b, now.Add(time.Minute), now.Add(time.Minute)
	if err := s.UpdateDescriptionVersion(ctx, merged); err != nil {
		t.Fatal(err)
	}
	got, _, err = s.LatestDescriptionVersion(ctx, id)
	if err != nil || !reflect.DeepEqual(got, merged) {
		t.Fatal("merge lost origin", got, merged, err)
	}
	higher := merged
	higher.ID = uuid.NewV7()
	if err := s.InsertDescriptionVersion(ctx, higher); err != nil {
		t.Fatal(err)
	}
	// Force a sort instead of incidental index order to distinguish the id tie-break.
	err = postgres.NewTxManager(pool, time.Second).WithinTx(ctx, func(txctx context.Context) error {
		if _, e := postgres.DB(txctx, pool).Exec(txctx, "SET LOCAL enable_indexscan=off; SET LOCAL enable_indexonlyscan=off; SET LOCAL enable_bitmapscan=off"); e != nil {
			return e
		}
		got, _, e := s.LatestDescriptionVersion(txctx, id)
		if e != nil || got.ID != higher.ID {
			t.Fatal("same-time version order", got, e)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	gone := higher
	gone.ID = uuid.NewV7()
	gone.LastSavedAt = now.Add(time.Hour)
	if err := s.InsertDescriptionVersion(ctx, gone); err != nil {
		t.Fatal(err)
	}
	exec(t, pool, "UPDATE issue_description_versions SET deleted_at=$2 WHERE id=$1", gone.ID, now)
	foreign := gone
	foreign.ID = uuid.NewV7()
	foreign.IssueID = neighbor
	if err := s.InsertDescriptionVersion(ctx, foreign); err != nil {
		t.Fatal(err)
	}
	if got, _, err := s.LatestDescriptionVersion(ctx, id); err != nil || got.ID != higher.ID {
		t.Fatal("deleted or neighbor version selected", got, err)
	}
	if got, found, err := s.GetDescriptionVersion(ctx, base.ID); err != nil || !found || !reflect.DeepEqual(got, merged) {
		t.Fatal("detail mapping", got, found, err)
	}
	if _, found, err := s.GetDescriptionVersion(ctx, gone.ID); err != nil || found {
		t.Fatal("deleted detail", found, err)
	}
	if _, found, err := s.GetDescriptionVersion(ctx, uuid.NewV7()); err != nil || found {
		t.Fatal("missing detail", found, err)
	}
	exec(t, pool, "UPDATE issue_description_versions SET description_stripped=NULL,created_by_id=NULL,updated_by_id=NULL WHERE id=$1", base.ID)
	nullable, found, err := s.GetDescriptionVersion(ctx, base.ID)
	if err != nil || !found || nullable.Text != "" || nullable.CreatedByID != (uuid.UUID{}) || nullable.UpdatedByID != (uuid.UUID{}) || !nullable.LastSavedAt.Equal(merged.LastSavedAt) || !nullable.CreatedAt.Equal(base.CreatedAt) {
		t.Fatal("nullable legacy audit mapping", nullable, err)
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, _, err := s.GetDescriptionVersion(cancelled, id); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := s.DescriptionVersions(cancelled, id, nil, 10); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}

	if _, _, err := s.LatestDescriptionVersion(cancelled, id); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := s.InsertDescriptionVersion(cancelled, base); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := s.UpdateDescriptionVersion(cancelled, base); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
