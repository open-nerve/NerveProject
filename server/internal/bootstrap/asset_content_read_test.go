package bootstrap

import (
	"net/url"
	"strings"
	"testing"
	"time"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

// SQL triggers count even no-op UPDATE statements; the row snapshot alone
// would miss a write that restores its original values. This is a storage
// fixture, not evidence that complete or F1 has shipped.
func TestAssetContentReadNoWrites(t *testing.T) {
	var cfg config.Config
	w := newIssueWorldWith(t, func(c *config.Config) { c.Auth.JWT.PrivateKeyFile = writeFile(t, testKeyPEM); cfg = *c })
	waitForCompletedKinds(t, w.pool, map[string]int{"asset.cleanup": 1}, &lockedBuffer{})
	id := w.newWorkItem(t)
	row := assetRowForHost(t, w, domain.ScopeIssue, id, id)
	key, err := domain.NewObjectKey('o', row.ID, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	row.Uploaded = true
	row.Metadata.Final = &domain.ObjectRecord{Key: key, Phase: domain.ObjectActive, SafeAfter: time.Now().Add(time.Hour), Size: 3, ContentType: "text/plain", ETag: "fixture-etag", SHA256: strings.Repeat("a", 64)}
	if _, err := assetpg.New(w.pool).SaveAsset(t.Context(), row); err != nil {
		t.Fatal(err)
	}
	_, err = w.pool.Exec(t.Context(), `CREATE TABLE asset_read_audit (op text NOT NULL);
CREATE FUNCTION audit_asset_write() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN INSERT INTO asset_read_audit VALUES(TG_OP); RETURN NULL; END $$;
CREATE TRIGGER audit_asset_write AFTER INSERT OR UPDATE OR DELETE ON file_assets FOR EACH STATEMENT EXECUTE FUNCTION audit_asset_write();`)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := func() string {
		var data string
		if err := w.pool.QueryRow(t.Context(), "SELECT row_to_json(a)::text FROM file_assets a WHERE id=$1", row.ID).Scan(&data); err != nil {
			t.Fatal(err)
		}
		return data
	}
	before := snapshot()
	store := assetClientStore(t, cfg)
	for range 3 {
		body := assetAPICall(t, w, "bob", "GET", "/api/v0/assets/"+row.ID.String()+"/content", "", 200)
		var got struct {
			URL       string    `json:"url"`
			ExpiresAt time.Time `json:"expires_at"`
		}
		decodeAnswer(t, body, &got)
		parsed, err := url.Parse(got.URL)
		if err != nil {
			t.Fatal(err)
		}
		grant, err := store.VerifyRead(string(key), parsed.RawQuery, "GET")
		if err != nil || grant.Key != key || grant.Inline || grant.Name != row.Name || grant.ContentType != row.ContentType || !grant.ExpiresAt.Equal(got.ExpiresAt) {
			t.Fatalf("wrong read grant: %+v err=%v", grant, err)
		}
		if got.ExpiresAt.Location() != time.UTC || got.ExpiresAt.Minute() != 0 || got.ExpiresAt.Second() != 0 || got.ExpiresAt.Nanosecond() != 0 {
			t.Fatalf("expiry is not a UTC whole hour: %s", got.ExpiresAt)
		}
		assetAPICall(t, w, "bob", "GET", "/api/v0/assets/"+row.ID.String(), "", 200)
	}
	var count int
	if err := w.pool.QueryRow(t.Context(), "SELECT count(*) FROM asset_read_audit").Scan(&count); err != nil || count != 0 || snapshot() != before {
		t.Fatalf("content reads wrote rows: audit=%d err=%v", count, err)
	}
}
