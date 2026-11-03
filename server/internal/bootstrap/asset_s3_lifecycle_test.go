package bootstrap

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/s3test"
)

func TestAssetS3Lifecycle(t *testing.T) {
	service := s3test.New(t)
	t.Run("composition sources and ownership", func(t *testing.T) { assetS3Composition(t, service.NewBucket(t)) })
	t.Run("app shutdown and constructor failure", func(t *testing.T) {
		b := service.NewBucket(t)
		assetLifetimeFollowsApp(t, func(c *config.Config) { configureAssetS3(t, c, b) })
	})
	b := service.NewBucket(t)
	w := newAssetS3World(t, b, func(*config.Config) {}, func(*assetComponents) {})
	s3RoutesAndNamespace(t, w)
	item := w.newWorkItem(t)
	data := []byte("private immutable S3 application bytes")
	file := prepareAssetHTTP(t, w.issueWorld, "bob", "ISSUE_ATTACHMENT", assetScopeBody("issue", item), "report.txt", "text/plain", len(data))
	putAssetS3HTTP(t, w, file, data)
	body := assetAPICall(t, w.issueWorld, "bob", "POST", completeAssetPath(file.Asset.ID), `{}`, 200)
	decodeAnswer(t, body, &file.Asset)
	rows := assetpg.New(w.pool)
	row, err := rows.FindAsset(t.Context(), file.Asset.ID)
	if err != nil || !row.Uploaded || row.Metadata.Final == nil || row.Metadata.Staging == nil || len(row.Metadata.Candidates) != 0 {
		t.Fatalf("incomplete S3 publication: %+v %v", row, err)
	}
	digest := sha256.Sum256(data)
	if row.Metadata.Final.SHA256 != hex.EncodeToString(digest[:]) || row.Metadata.Final.ETag == "" {
		t.Fatal("published metadata lacks actual byte proof")
	}
	res, actual := readAssetS3HTTP(t, w, "bob", file.Asset.ID)
	if res.StatusCode != 200 || !bytes.Equal(actual, data) || res.Header.Get("Cache-Control") != "private, no-store" || !strings.HasPrefix(res.Header.Get("Content-Disposition"), "attachment;") {
		t.Fatalf("S3 read %d %q %v", res.StatusCode, actual, res.Header)
	}
	if repeat := assetAPICall(t, w.issueWorld, "bob", "POST", completeAssetPath(file.Asset.ID), `{}`, 200); repeat != body {
		t.Fatal("repeat changed public result")
	}
	s3ReadWithoutDML(t, w, file)
	after, err := rows.FindAsset(t.Context(), file.Asset.ID)
	if err != nil || !reflect.DeepEqual(row, after) {
		t.Fatal("read or repeat altered state")
	}
	// A replay can overwrite staging on S3. The published final stays immutable.
	putAssetS3HTTP(t, w, file, bytes.Repeat([]byte{'x'}, len(data)))
	_, actual = readAssetS3HTTP(t, w, "bob", file.Asset.ID)
	if !bytes.Equal(actual, data) {
		t.Fatal("old PUT replaced final bytes")
	}
	var copy preparedAssetAnswer
	decodeAnswer(t, assetAPICall(t, w.issueWorld, "bob", "POST", duplicateAssetPath(file.Asset.ID), fmt.Sprintf(`{"purpose":"ISSUE_ATTACHMENT","scope":%s}`, assetScopeBody("issue", item)), 201), &copy.Asset)
	copied, err := rows.FindAsset(t.Context(), copy.Asset.ID)
	if err != nil || copied.Metadata.Final == nil || copied.Metadata.Final.Key == row.Metadata.Final.Key || copied.Metadata.Final.SHA256 != row.Metadata.Final.SHA256 {
		t.Fatalf("copy not independent: %+v %v", copied, err)
	}
	_, actual = readAssetS3HTTP(t, w, "bob", copy.Asset.ID)
	if !bytes.Equal(actual, data) {
		t.Fatal("copy bytes differ")
	}
	for _, id := range []string{file.Asset.ID.String(), copy.Asset.ID.String()} {
		assetAPICall(t, w.issueWorld, "bob", "DELETE", "/api/v0/assets/"+id, "", 204)
		assetAPICall(t, w.issueWorld, "bob", "GET", "/api/v0/assets/"+id+"/content", "", 404)
	}
	reopened := restartAssetS3Cleanup(t, w, 61*24*time.Hour)
	for _, key := range []domain.ObjectKey{row.Metadata.Staging.Key, row.Metadata.Final.Key, copied.Metadata.Final.Key} {
		if _, err := reopened.Inspect(t.Context(), key); !errors.Is(err, domain.ErrObjectNotFound) {
			t.Fatalf("app cleanup retained deleted bytes %s: %v", key, err)
		}
	}
	for _, id := range []uuid.UUID{row.ID, copied.ID} {
		got, err := rows.FindAsset(t.Context(), id)
		if !errors.Is(err, domain.ErrAssetNotFound) && (err != nil || got.Metadata.Final != nil || got.Metadata.Staging != nil || len(got.Metadata.Candidates) != 0) {
			t.Fatalf("app cleanup retained recovery references: %+v %v", got, err)
		}
	}
}

func s3ReadWithoutDML(t *testing.T, w assetS3World, file preparedAssetAnswer) {
	t.Helper()
	waitForCompletedKinds(t, w.pool, map[string]int{"asset.cleanup": 1, "cleanup.purge_deleted": 1}, &lockedBuffer{})
	_, err := w.pool.Exec(t.Context(), `CREATE TABLE s3_read_audit (op text NOT NULL);
CREATE FUNCTION audit_s3_write() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN INSERT INTO s3_read_audit VALUES(TG_OP); RETURN NULL; END $$;
CREATE TRIGGER audit_s3_write AFTER INSERT OR UPDATE OR DELETE ON file_assets FOR EACH STATEMENT EXECUTE FUNCTION audit_s3_write();`)
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		readAssetS3HTTP(t, w, "bob", file.Asset.ID)
		assetAPICall(t, w.issueWorld, "bob", "GET", "/api/v0/assets/"+file.Asset.ID.String(), "", 200)
	}
	var count int
	if err := w.pool.QueryRow(t.Context(), "SELECT count(*) FROM s3_read_audit").Scan(&count); err != nil || count != 0 {
		t.Fatalf("S3 content/metadata read DML=%d %v", count, err)
	}
	if _, err := w.pool.Exec(t.Context(), "DROP TRIGGER audit_s3_write ON file_assets"); err != nil {
		t.Fatal(err)
	}
}

func s3RoutesAndNamespace(t *testing.T, w assetS3World) {
	t.Helper()
	assertStorageDriverRoutes(t, w.program, "s3")
	assertAPIRoutes(t, w.program)
	assertPublicOperations(t, w.program)
	assertOperationsRequireToken(t, w.base)
	noRedirect := *client
	noRedirect.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	for _, path := range []string{"/static", "/static/", "/static/local/" + string(assetKey(t, 'u')), "/static//missing", "/static/local/../../settings", "//static/local/../../settings", "/elsewhere/../static/missing", "/%73tatic/local/../../settings"} {
		res, err := noRedirect.Get(w.base + path + "?payload=private&tag=private")
		if err != nil {
			t.Fatal(err)
		}
		_ = res.Body.Close()
		if res.StatusCode != 404 || res.Header.Get("Location") != "" || res.Header.Get("Cache-Control") != "no-store" {
			t.Fatalf("S3 reserved namespace redirected/served local: %s %d %v", path, res.StatusCode, res.Header)
		}
	}
	w.send(t, "alice", "POST", "/api/v0/workspaces", `{"name":"Storage","slug":"storage"}`, 201)
	for _, path := range []string{"/storage", "/storage/projects", "/storage/settings", "/static-more/projects"} {
		res, body := send(t, newRequest(t, "GET", w.base+path, "", nil))
		if res.StatusCode != 200 || !strings.Contains(string(body), testIndexHTML) {
			t.Fatalf("S3 intercepted workspace page %s", path)
		}
	}
}
