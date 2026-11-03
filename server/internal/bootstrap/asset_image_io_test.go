package bootstrap

import (
	"context"
	"io"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	assetapp "github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

// Instrument the real driver at its complete I/O boundary. A read signature is
// CPU-only; a successful signer call does not permit storage I/O under SQL locks.
type imageStorageCalls struct {
	assetStore
	io, signs atomic.Int64
}

func (s *imageStorageCalls) SignRead(ctx context.Context, g domain.ReadGrant) (domain.SignedRequest, error) {
	s.signs.Add(1)
	return s.assetStore.SignRead(ctx, g)
}
func (s *imageStorageCalls) Inspect(ctx context.Context, k domain.ObjectKey) (domain.ObjectInfo, error) {
	s.io.Add(1)
	return s.assetStore.Inspect(ctx, k)
}
func (s *imageStorageCalls) Open(ctx context.Context, k domain.ObjectKey, etag string) (io.ReadCloser, error) {
	s.io.Add(1)
	return s.assetStore.Open(ctx, k, etag)
}
func (s *imageStorageCalls) Seal(ctx context.Context, from, to domain.ObjectKey, etag string) (domain.ObjectInfo, error) {
	s.io.Add(1)
	return s.assetStore.Seal(ctx, from, to, etag)
}
func (s *imageStorageCalls) Delete(ctx context.Context, k domain.ObjectKey) error {
	s.io.Add(1)
	return s.assetStore.Delete(ctx, k)
}
func (s *imageStorageCalls) List(ctx context.Context, cursor string, limit int) (assetapp.ObjectPage, error) {
	s.io.Add(1)
	return s.assetStore.List(ctx, cursor, limit)
}
func (s *imageStorageCalls) SweepTemporary(ctx context.Context, at time.Time, limit int, cursor string) (string, int, error) {
	s.io.Add(1)
	return s.assetStore.SweepTemporary(ctx, at, limit, cursor)
}

func TestImageAuthNoStorageIOOrAuditWrites(t *testing.T) {
	var storage *imageStorageCalls
	w, _ := newAssetCompleteWorld(t, func(c *assetComponents) {
		storage = &imageStorageCalls{assetStore: c.store}
		c.store = storage
		c.sealing.Inspector, c.sealing.Reader, c.sealing.Sealer = storage, storage, storage
	})
	bindRaceImage(t, w, "alice", "USER_AVATAR", `{"kind":"user"}`, "/api/v0/me", "avatar")
	bindRaceImage(t, w, "alice", "USER_COVER", `{"kind":"user"}`, "/api/v0/me", "cover_image")
	bindRaceImage(t, w, "alice", "WORKSPACE_LOGO", assetScopeBody("workspace", w.acme), "/api/v0/workspaces/acme", "logo")
	p := "/api/v0/projects/" + w.web.String()
	bindRaceImage(t, w, "alice", "PROJECT_COVER", assetScopeBody("project", w.web), p, "cover_image")
	link := invite(t, w.contract, w.base, w.tokens["alice"], "acme", "olga@example.com")
	waitForCompletedKinds(t, w.pool, map[string]int{"asset.cleanup": 1, "cleanup.purge_deleted": 1}, &lockedBuffer{})
	if storage.io.Load() == 0 {
		t.Fatal("the real upload did not exercise the storage instrumentation")
	}
	if _, err := w.pool.Exec(t.Context(), `CREATE TABLE image_auth_writes(name text NOT NULL);
CREATE FUNCTION audit_image_auth() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN INSERT INTO image_auth_writes VALUES(TG_TABLE_NAME); RETURN NULL; END $$;
CREATE TRIGGER image_auth_user AFTER INSERT OR UPDATE OR DELETE ON users FOR EACH STATEMENT EXECUTE FUNCTION audit_image_auth();
CREATE TRIGGER image_auth_profile AFTER INSERT OR UPDATE OR DELETE ON profiles FOR EACH STATEMENT EXECUTE FUNCTION audit_image_auth();
CREATE TRIGGER image_auth_workspace AFTER INSERT OR UPDATE OR DELETE ON workspaces FOR EACH STATEMENT EXECUTE FUNCTION audit_image_auth();
CREATE TRIGGER image_auth_project AFTER INSERT OR UPDATE OR DELETE ON projects FOR EACH STATEMENT EXECUTE FUNCTION audit_image_auth();
CREATE TRIGGER image_auth_asset AFTER INSERT OR UPDATE OR DELETE ON file_assets FOR EACH STATEMENT EXECUTE FUNCTION audit_image_auth();`); err != nil {
		t.Fatal(err)
	}
	before := profileImageSnapshot(t, w) + workspaceImageSnapshot(t, w) + projectCoverSnapshot(t, w)
	for _, target := range []struct{ who, path string }{
		{"alice", "/api/v0/me"},
		{"bob", "/api/v0/workspaces"},
		{"bob", "/api/v0/workspaces/acme"},
		{"alice", "/api/v0/workspaces/acme/members"},
		{"bob", p},
		{"bob", "/api/v0/workspaces/acme/projects"},
		{"", "/api/v0/workspace-invitations/" + link.id.String() + "?token=" + url.QueryEscape(link.token)},
	} {
		beforeIO, beforeSign := storage.io.Load(), storage.signs.Load()
		w.send(t, target.who, "GET", target.path, "", 200)
		if storage.io.Load() != beforeIO || storage.signs.Load() <= beforeSign {
			t.Fatalf("%s: I/O=%d signs=%d", target.path, storage.io.Load()-beforeIO, storage.signs.Load()-beforeSign)
		}
	}
	var writes int
	if err := w.pool.QueryRow(t.Context(), "SELECT count(*) FROM image_auth_writes").Scan(&writes); err != nil || writes != 0 || before != profileImageSnapshot(t, w)+workspaceImageSnapshot(t, w)+projectCoverSnapshot(t, w) {
		t.Fatal("image reads changed audit state", writes, err)
	}
}
