package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset"
	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	assetdomain "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestAssetParentEventRollsBackAfterSynchronousMark(t *testing.T) {
	w := newIssueWorld(t)
	id := w.newWorkItem(t)
	a := assetRowForHost(t, w, assetdomain.ScopeIssue, id, id)
	client, err := jobs.NewInserter(w.pool)
	if err != nil {
		t.Fatal(err)
	}
	subs := asset.Subscribers(asset.Provide(w.pool), assetInserter{client: client})
	fault := errors.New("later subscriber failed")
	seen := false
	subs = append(subs, shared.EventSubscriber{Name: "fail_after_asset", Types: []string{"issue.deleted"}, Handle: func(ctx context.Context, e shared.Event) error {
		got, err := assetpg.New(w.pool).FindAsset(ctx, a.ID)
		if err != nil {
			return err
		}
		if got.DeletedAt == nil || !got.DeletedAt.Equal(e.OccurredAt) || got.DeletedByID == nil || *got.DeletedByID != w.ids["alice"] {
			t.Fatalf("event did not mark asset in same tx: %+v", got)
		}
		seen = true
		return fault
	}})
	events, err := newDispatcher(eventTypes(), subs)
	if err != nil {
		t.Fatal(err)
	}
	route := newIssueRouteWithPublisher(t, w.pool, postgres.NewTxManager(w.pool, time.Second), clock.System{}, events)
	var response exchange
	if err := w.sending(&response, route, "alice", http.MethodDelete, "/api/v0/issues/"+id.String(), "")(); err != nil {
		t.Fatal(err)
	}
	if response.rec.Code != http.StatusInternalServerError || !seen {
		t.Fatalf("response=%d seen=%t", response.rec.Code, seen)
	}
	got, err := assetpg.New(w.pool).FindAsset(t.Context(), a.ID)
	if err != nil || got.DeletedAt != nil || got.StorageRevision != 0 {
		t.Fatalf("asset escaped rollback: %+v %v", got, err)
	}
	var live bool
	if err := w.pool.QueryRow(t.Context(), "SELECT deleted_at IS NULL FROM issues WHERE id=$1", id).Scan(&live); err != nil || !live {
		t.Fatalf("parent escaped rollback: %t %v", live, err)
	}
}

func TestAssetParentJobsUseOriginalTransaction(t *testing.T) {
	w := newIssueWorld(t)
	client, err := jobs.NewInserter(w.pool)
	if err != nil {
		t.Fatal(err)
	}
	events, err := newDispatcher(eventTypes(), asset.Subscribers(asset.Provide(w.pool), assetInserter{client: client}))
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"project", "workspace"} {
		for _, rollback := range []bool{true, false} {
			id := w.web
			project := w.web
			if kind == "workspace" {
				id = w.acme
				project = uuid.Nil()
			}
			at := time.Now().UTC().Truncate(time.Microsecond)
			by := w.ids["alice"]
			e := shared.Event{ID: uuid.NewV7(), Type: kind + ".deleted", OccurredAt: at, ActorID: &by, WorkspaceID: w.acme, ProjectID: project, Entity: shared.EntityRef{Kind: kind, ID: id}}
			fault := errors.New("abort after asset enqueue")
			err := postgres.NewTxManager(w.pool, time.Second).WithinTx(t.Context(), func(ctx context.Context) error {
				if err := events.Publish(ctx, e); err != nil {
					return err
				}
				var inside, outside int
				if err := postgres.DB(ctx, w.pool).QueryRow(ctx, "SELECT count(*) FROM river_job WHERE kind='asset.delete_scope' AND args->>'at'=$1", at.Format(time.RFC3339Nano)).Scan(&inside); err != nil {
					return err
				}
				if err := w.pool.QueryRow(ctx, "SELECT count(*) FROM river_job WHERE kind='asset.delete_scope' AND args->>'at'=$1", at.Format(time.RFC3339Nano)).Scan(&outside); err != nil {
					return err
				}
				if inside != 1 || outside != 0 {
					t.Fatalf("insert visibility inside=%d outside=%d", inside, outside)
				}
				if rollback {
					return fault
				}
				return nil
			})
			if (rollback && !errors.Is(err, fault)) || (!rollback && err != nil) {
				t.Fatalf("transaction error=%v", err)
			}
			var n int
			if err := w.pool.QueryRow(t.Context(), "SELECT count(*) FROM river_job WHERE kind='asset.delete_scope' AND args->>'at'=$1 AND args->>'by'=$2 AND args->>'id'=$3 AND args->>'scope'=$4", at.Format(time.RFC3339Nano), by.String(), id.String(), kind).Scan(&n); err != nil {
				t.Fatal(err)
			}
			want := 1
			if rollback {
				want = 0
			}
			if n != want {
				t.Fatalf("kind=%s rollback=%t count=%d", kind, rollback, n)
			}
		}
	}
}
