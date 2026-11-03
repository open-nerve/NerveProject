package bootstrap

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/local"
	assetapp "github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

func newAssetCompleteWorld(t *testing.T, alter func(*assetComponents)) (issueWorld, *local.Store) {
	t.Helper()
	return newAssetCompleteWorldWith(t, func(*config.Config) {}, alter)
}

func newAssetCompleteWorldWith(t *testing.T, configure func(*config.Config), alter func(*assetComponents)) (issueWorld, *local.Store) {
	t.Helper()
	dbURL := pgtest.NewDatabase(t)
	cfg := testConfig(t, dbURL, false)
	configure(&cfg)
	requests := &requestQueries{}
	var store *local.Store
	factory := func(files config.FilesConfig, keys *identity.Keys, logger *slog.Logger) (assetComponents, error) {
		components, err := newAssets(files, keys, logger)
		if err == nil {
			store = components.store
			alter(&components)
		}
		return components, err
	}
	a, err := newAppWithFactories(t.Context(), cfg, slog.New(slog.DiscardHandler), migrations.FS(), testWebUI, appFactories{pool: requests.pool, assets: factory})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(a.close)
	w := prepareIssueWorld(t, issueWorld{requests: requests, base: runBuiltApp(t, a), pool: openPool(t, dbURL), contract: apitest.Load(t), tokens: map[string]string{}, ids: map[string]uuid.UUID{}})
	return w, store
}

type assetSealFunc func(context.Context, domain.ObjectKey, domain.ObjectKey, string) (domain.ObjectInfo, error)

func (f assetSealFunc) Seal(ctx context.Context, src, dst domain.ObjectKey, etag string) (domain.ObjectInfo, error) {
	return f(ctx, src, dst, etag)
}

type assetSealAttempt struct {
	key  domain.ObjectKey
	gate *gate
	mu   sync.Mutex
	err  error
	once sync.Once
}

func (a *assetSealAttempt) resume(err error) {
	a.once.Do(func() { a.mu.Lock(); a.err = err; a.mu.Unlock(); close(a.gate.open) })
}
func controlledAssetSeal(base assetapp.ObjectSealer, attempts chan<- *assetSealAttempt) assetSealFunc {
	return func(ctx context.Context, src, dst domain.ObjectKey, etag string) (domain.ObjectInfo, error) {
		info, err := base.Seal(ctx, src, dst, etag)
		if err != nil {
			return info, err
		}
		attempt := &assetSealAttempt{key: dst, gate: newGate()}
		select {
		case attempts <- attempt:
		case <-ctx.Done():
			return domain.ObjectInfo{}, ctx.Err()
		}
		if err := attempt.gate.wait(ctx); err != nil {
			return domain.ObjectInfo{}, err
		}
		attempt.mu.Lock()
		defer attempt.mu.Unlock()
		return info, attempt.err
	}
}
func awaitAssetSeal(t *testing.T, attempts <-chan *assetSealAttempt, response <-chan answer) *assetSealAttempt {
	t.Helper()
	select {
	case attempt := <-attempts:
		t.Cleanup(func() { attempt.resume(context.Canceled) })
		return attempt
	case answer := <-response:
		t.Fatalf("confirmation ended before reaching real Seal: %+v", answer)
	case <-time.After(5 * time.Second):
		t.Fatal("confirmation did not reach real Seal without holding resources")
	}
	return nil
}
