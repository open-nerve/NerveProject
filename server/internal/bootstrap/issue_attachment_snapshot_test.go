package bootstrap

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

func TestIssueAttachmentProjectionSnapshot(t *testing.T) {
	for _, kind := range []string{"detail", "identifier", "children"} {
		t.Run(kind, func(t *testing.T) {
			dbURL := pgtest.NewDatabase(t)
			pool := openPool(t, dbURL)
			cfg := testConfig(t, dbURL, false)
			trace := &attachmentSnapshotTrace{gate: newGate(), pauseSQL: "AttachmentCounts"}
			factory := func(ctx context.Context, c config.DatabaseConfig) (*pgxpool.Pool, error) {
				source, err := postgres.NewPool(ctx, c)
				if err != nil {
					return nil, err
				}
				pc := source.Config()
				source.Close()
				pc.ConnConfig.Tracer = trace
				return pgxpool.NewWithConfig(ctx, pc)
			}
			a, err := newAppWithPoolFactory(t.Context(), cfg, slog.New(slog.DiscardHandler), migrations.FS(), testWebUI, factory)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(a.close)
			w := prepareIssueWorld(t, issueWorld{base: runBuiltApp(t, a), pool: pool, contract: apitest.Load(t), tokens: map[string]string{}, ids: map[string]uuid.UUID{}})
			parent := w.newWorkItem(t)
			item := parent
			path := "/api/v0/issues/" + parent.String()
			firstRead := "-- name: GetIssue "
			if kind == "identifier" {
				path = "/api/v0/workspaces/acme/issues/WEB-1"
				firstRead = "-- name: DirectoryWorkspace "
			}
			if kind == "children" {
				item = w.newWorkItemIn(t, w.web, underParent("Count child", parent))
				path += "/sub-issues"
			}
			old := uploadedAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", attachmentScope("issue", item), "old.txt", "text/plain", []byte("old"))
			oldAt, _ := attachmentIssueStamp(t, w, item)
			req := newRequest(t, "GET", w.base+path, w.tokens["bob"], nil)
			req.Header.Set(httpserver.HeaderRequestID, "attachment-snapshot-reader")
			var once sync.Once
			resume := func() { once.Do(func() { close(trace.gate.open) }) }
			defer resume()
			trace.gate.deadline = time.Now().Add(10 * time.Second)
			response := sendInBackground(req)
			select {
			case <-trace.gate.held:
			case got := <-response:
				t.Fatalf("read ended before count barrier: %+v", got)
			case <-time.After(5 * time.Second):
				t.Fatal("missed count barrier")
			}
			uploadedAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", attachmentScope("issue", item), "new.txt", "text/plain", []byte("new"))
			assetAPICall(t, w, "bob", "DELETE", "/api/v0/assets/"+old.Asset.ID.String(), "", 204)
			w.send(t, "alice", "DELETE", "/api/v0/issues/"+parent.String(), "", 204)
			resume()
			got := receiveWithin(t, response, 5*time.Second, "issue count snapshot")
			if got.err != nil || got.res.StatusCode != 200 {
				t.Fatalf("mixed owner/issue/count snapshot: %+v", got)
			}
			w.contract.CheckResponse(t, req, got.res)
			checkAttachmentProjection(t, string(got.body), map[uuid.UUID]int64{item: 1})
			var detail struct {
				UpdatedAt time.Time `json:"updated_at"`
			}
			var page struct {
				Data []struct {
					UpdatedAt time.Time `json:"updated_at"`
				} `json:"data"`
			}
			if kind == "children" {
				decodeAnswer(t, string(got.body), &page)
				if len(page.Data) != 1 {
					t.Fatal(page)
				}
				detail.UpdatedAt = page.Data[0].UpdatedAt
			} else {
				if json.Unmarshal(got.body, &detail) != nil {
					t.Fatal("bad detail")
				}
			}
			if !detail.UpdatedAt.Equal(oldAt) {
				t.Fatal("issue audit came from another snapshot", detail.UpdatedAt, oldAt)
			}
			trace.mu.Lock()
			queries, traceErr, multiple := strings.Join(trace.queries, "\n"), trace.err, trace.multiple
			trace.mu.Unlock()
			begin := strings.Index(strings.ToLower(queries), "begin isolation level repeatable read read only")
			first := strings.Index(queries, firstRead)
			if traceErr != nil || multiple || begin < 0 || first < begin || strings.Contains(strings.ToLower(queries), "for update") || strings.Count(queries, "-- name: AttachmentCounts :many") != 1 {
				t.Fatalf("not one snapshot from first host read: %s %v", queries, traceErr)
			}
			w.send(t, "bob", "GET", path, "", 404)
		})
	}
}

func TestIssueAttachmentProjectionHiddenBeforeCounterFailure(t *testing.T) {
	w := newIssueWorld(t)
	id := w.newWorkItem(t)
	if _, err := w.pool.Exec(t.Context(), "ALTER TABLE file_assets RENAME TO unavailable_issue_assets"); err != nil {
		t.Fatal(err)
	}
	path := "/api/v0/issues/" + id.String()
	w.send(t, "olga", "GET", path, "", 404)
	w.send(t, "bob", "GET", path, "", 500)
}
