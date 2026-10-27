package postgresadapter_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// Dropping owner/scope filtering, counting after the cursor, merging payloads,
// or stamping a JSONB-equivalent update must change these observed results.
func TestDraftRowsAndCursorScope(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	a, b := newAccount(t, pool, "draft-a@corp.com"), newAccount(t, pool, "draft-b@corp.com")
	wid, other := newWorkspace(t, pool, "drafts"), newWorkspace(t, pool, "elsewhere")
	active, archived, gone := newProject(t, pool, wid, "ACT"), newProject(t, pool, wid, "ARC"), newProject(t, pool, wid, "DEL")
	exec(t, pool, "UPDATE projects SET archived_at=$2 WHERE id=$1", archived.id, now)
	exec(t, pool, "UPDATE projects SET deleted_at=$2 WHERE id=$1", gone.id, now)
	ids := []uuid.UUID{
		uuid.MustParse("0199a2b4-0000-7000-8000-000000000001"),
		uuid.MustParse("0199a2b4-0000-7000-8000-000000000002"),
		uuid.MustParse("0199a2b4-0000-7000-8000-000000000003"),
	}
	base := json.RawMessage(`{"name":"  中文🙂  ","description_html":"<script>原样</script>","priority":"later"}`)
	writes := []app.DraftWrite{
		{ID: ids[0], WorkspaceID: wid, By: a, Payload: base, At: now},
		{ID: ids[1], WorkspaceID: wid, ProjectID: &active.id, By: a, Payload: json.RawMessage(`{}`), At: now},
		{ID: ids[2], WorkspaceID: wid, ProjectID: &archived.id, By: a, Payload: json.RawMessage(`{}`), At: now},
	}
	for _, w := range writes {
		got, err := s.InsertDraft(ctx, w)
		if err != nil || got.ID != w.ID || got.WorkspaceID != wid || got.CreatedByID != a || got.UpdatedByID == nil || *got.UpdatedByID != a || !got.CreatedAt.Equal(now) || !got.UpdatedAt.Equal(now) || !reflect.DeepEqual(got.ProjectID, w.ProjectID) {
			t.Fatalf("insert = %+v, %v", got, err)
		}
	}
	foreign := []app.DraftWrite{
		{ID: uuid.NewV7(), WorkspaceID: wid, By: b, Payload: json.RawMessage(`{}`), At: now},
		{ID: uuid.NewV7(), WorkspaceID: wid, By: b, Payload: json.RawMessage(`{}`), At: now.Add(-time.Hour)},
		{ID: uuid.NewV7(), WorkspaceID: other, By: a, Payload: json.RawMessage(`{}`), At: now},
		{ID: uuid.NewV7(), WorkspaceID: wid, ProjectID: &gone.id, By: a, Payload: json.RawMessage(`{}`), At: now},
		{ID: uuid.NewV7(), WorkspaceID: wid, By: a, Payload: json.RawMessage(`{}`), At: now},
	}
	for _, w := range foreign {
		if _, err := s.InsertDraft(ctx, w); err != nil {
			t.Fatal(err)
		}
	}
	deleted := foreign[4].ID
	if err := s.DeleteDraft(ctx, app.DraftDeletion{ID: deleted, By: b, At: now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	// Readers deliberately expose live foreign-owned rows to permission orchestration.
	for _, read := range []func(context.Context, uuid.UUID) (domain.Draft, bool, error){s.GetDraft, s.LockDraft} {
		got, found, err := read(ctx, foreign[0].ID)
		if err != nil || !found || got.CreatedByID != b {
			t.Fatalf("foreign detail = %+v, %v, %v", got, found, err)
		}
		for _, id := range []uuid.UUID{deleted, uuid.NewV7()} {
			if _, found, err := read(ctx, id); err != nil || found {
				t.Fatalf("missing/deleted read = %v, %v", found, err)
			}
		}
	}
	q := app.DraftListRead{WorkspaceID: wid, ActorID: a, ProjectIDs: []uuid.UUID{active.id, archived.id}, Limit: 2}
	checkPage := func(after *domain.DraftCursor, want []uuid.UUID) {
		t.Helper()
		q.After = after
		err := postgres.NewTxManager(pool, time.Second, postgres.ReadOnlySnapshot()).WithinTx(ctx, func(txctx context.Context) error {
			rows, total, err := s.ListDrafts(txctx, q)
			if err != nil {
				return err
			}
			got := make([]uuid.UUID, len(rows))
			for n, row := range rows {
				got[n] = row.ID
			}
			if total != 3 || !reflect.DeepEqual(got, want) || rows == nil {
				t.Fatalf("page after %+v = %v, total %d; want %v / 3", after, got, total, want)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	checkPage(nil, []uuid.UUID{ids[2], ids[1]})
	checkPage(&domain.DraftCursor{CreatedAt: now, ID: ids[1]}, []uuid.UUID{ids[0]})
	checkPage(&domain.DraftCursor{CreatedAt: now, ID: ids[0]}, []uuid.UUID{})
	// Canonical cursors can name another owner's or an unknown row, never a scope.
	for _, c := range []domain.DraftCursor{
		{CreatedAt: foreign[0].At, ID: foreign[0].ID},
		{CreatedAt: foreign[1].At, ID: foreign[1].ID},
		{CreatedAt: now.Add(time.Hour), ID: uuid.NewV7()},
	} {
		encoded, err := domain.EncodeDraftCursor(c)
		if err != nil {
			t.Fatal(err)
		}
		after, err := domain.DecodeDraftCursor(encoded)
		if err != nil {
			t.Fatal(err)
		}
		want := []uuid.UUID{ids[2], ids[1]}
		if c.CreatedAt.Before(now) {
			want = []uuid.UUID{}
		}
		checkPage(&after, want)
	}
	q.ProjectIDs = nil
	rows, total, err := s.ListDrafts(ctx, q)
	if err != nil || total != 1 || len(rows) != 1 || rows[0].ID != ids[0] {
		t.Fatalf("empty project set = %+v, %d, %v", rows, total, err)
	}
	w := writes[0]
	w.Payload = json.RawMessage(`{"priority":"later","description_html":"<script>原样</script>","name":"  中文🙂  "}`)
	w.By, w.At = b, now.Add(2*time.Minute)
	got, err := s.UpdateDraft(ctx, w)
	if err != nil || !got.UpdatedAt.Equal(now) || got.UpdatedByID == nil || *got.UpdatedByID != a {
		t.Fatalf("JSONB noop audit = %+v, %v", got, err)
	}
	var stored map[string]string
	if err := json.Unmarshal(got.Payload, &stored); err != nil || stored["name"] != "  中文🙂  " || stored["description_html"] != "<script>原样</script>" {
		t.Fatalf("payload transformed: %s, %v", got.Payload, err)
	}
	w.Payload = json.RawMessage(`{}`)
	w.ProjectID = &active.id
	got, err = s.UpdateDraft(ctx, w)
	if err != nil || string(got.Payload) != "{}" || !got.UpdatedAt.Equal(w.At) || got.UpdatedByID == nil || *got.UpdatedByID != b || got.ProjectID == nil || *got.ProjectID != active.id || got.CreatedByID != a || !got.CreatedAt.Equal(now) {
		t.Fatalf("replacement = %+v, %v", got, err)
	}
	w.ProjectID = nil
	w.At = now.Add(3 * time.Minute)
	got, err = s.UpdateDraft(ctx, w)
	if err != nil || got.ProjectID != nil || !got.UpdatedAt.Equal(w.At) {
		t.Fatalf("clear project = %+v, %v", got, err)
	}
	got, found, err := s.GetDraft(ctx, w.ID)
	if err != nil || !found || string(got.Payload) != "{}" || !got.UpdatedAt.Equal(w.At) {
		t.Fatalf("readback = %+v, %v, %v", got, found, err)
	}
	// Repeated deletion preserves its original actor/time; update cannot resurrect it.
	if err := s.DeleteDraft(ctx, app.DraftDeletion{ID: deleted, By: a, At: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	var by uuid.UUID
	var stamp time.Time
	if err := pool.QueryRow(ctx, "SELECT updated_by_id,updated_at FROM draft_issues WHERE id=$1", deleted).Scan(&by, &stamp); err != nil || by != b || !stamp.Equal(now.Add(time.Minute)) {
		t.Fatalf("delete audit = %s, %s, %v", by, stamp, err)
	}
	w.ID = deleted
	if _, err := s.UpdateDraft(ctx, w); err == nil {
		t.Fatal("updated deleted draft")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, _, err := s.GetDraft(cancelled, w.ID); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, _, err := s.LockDraft(cancelled, w.ID); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, _, err := s.ListDrafts(cancelled, q); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := s.InsertDraft(cancelled, w); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := s.UpdateDraft(cancelled, w); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := s.DeleteDraft(cancelled, app.DraftDeletion{ID: w.ID}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
