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

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

type attachmentSnapshotTrace struct {
	gate     *gate
	pauseSQL string
	mu       sync.Mutex
	queries  []string
	err      error
	pid      uint32
	multiple bool
}

func (p *attachmentSnapshotTrace) TraceQueryStart(ctx context.Context, c *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	if httpserver.RequestID(ctx) != "attachment-snapshot-reader" {
		return ctx
	}
	p.mu.Lock()
	p.queries = append(p.queries, d.SQL)
	// Authentication can borrow a different connection before BEGIN. Track
	// only the read transaction, from its BEGIN through COMMIT/ROLLBACK.
	if strings.HasPrefix(strings.ToLower(d.SQL), "begin ") {
		p.pid = c.PgConn().PID()
	}
	if p.pid != 0 && p.pid != c.PgConn().PID() {
		p.multiple = true
	}
	p.mu.Unlock()
	pauseSQL := p.pauseSQL
	if pauseSQL == "" {
		pauseSQL = "ListAttachments"
	}
	if strings.HasPrefix(d.SQL, "-- name: "+pauseSQL+" :many") {
		err := p.gate.wait(ctx)
		p.mu.Lock()
		p.err = err
		p.mu.Unlock()
	}
	return ctx
}
func (*attachmentSnapshotTrace) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

func TestAttachmentListsUseOneSnapshot(t *testing.T) {
	for _, kind := range []string{"issue", "comment", "draft"} {
		t.Run(kind, func(t *testing.T) {
			trace := &attachmentSnapshotTrace{gate: newGate()}
			w := attachmentSnapshotWorld(t, trace)
			item := w.newWorkItem(t)
			id := item
			host := "/api/v0/issues/" + id.String()
			purpose := "ISSUE_ATTACHMENT"
			if kind == "comment" {
				id = w.newComment(t, "bob", item).ID
				host = commentPath(id)
				purpose = "COMMENT_ATTACHMENT"
			}
			if kind == "draft" {
				id = createDraftHTTP(t, w, "bob", `{"payload":{}}`).ID
				host = draftPath(id)
				purpose = "DRAFT_ISSUE_ATTACHMENT"
			}
			old := uploadedAssetHTTP(t, w, "bob", purpose, attachmentScope(kind, id), "old.txt", "text/plain", []byte("old"))
			req := newRequest(t, "GET", w.base+host+"/attachments", w.tokens["bob"], nil)
			req.Header.Set(httpserver.HeaderRequestID, "attachment-snapshot-reader")
			var once sync.Once
			resume := func() { once.Do(func() { close(trace.gate.open) }) }
			defer resume()
			trace.gate.deadline = time.Now().Add(10 * time.Second)
			response := sendInBackground(req)
			select {
			case <-trace.gate.held:
			case got := <-response:
				t.Fatalf("list ended before asset query barrier: %+v", got)
			case <-time.After(5 * time.Second):
				t.Fatal("list missed asset query barrier")
			}
			uploadedAssetHTTP(t, w, "bob", purpose, attachmentScope(kind, id), "new.txt", "text/plain", []byte("new"))
			assetAPICall(t, w, "bob", "DELETE", "/api/v0/assets/"+old.Asset.ID.String(), "", 204)
			deleter := "bob"
			if kind == "issue" {
				deleter = "alice"
			}
			w.send(t, deleter, "DELETE", host, "", 204)
			resume()
			got := receiveWithin(t, response, 5*time.Second, "attachment snapshot")
			if got.err != nil || got.res.StatusCode != 200 {
				t.Fatalf("mixed owner/asset snapshot: %+v", got)
			}
			w.contract.CheckResponse(t, req, got.res)
			var page attachmentListPage
			if json.Unmarshal([]byte(got.body), &page) != nil || len(page.Data) != 1 || page.Data[0].ID != old.Asset.ID {
				t.Fatalf("mixed asset snapshot: %s", got.body)
			}
			trace.mu.Lock()
			queries, traceErr, multiple := strings.Join(trace.queries, "\n"), trace.err, trace.multiple
			trace.mu.Unlock()
			if traceErr != nil || multiple || !strings.Contains(strings.ToLower(queries), "begin isolation level repeatable read read only") || strings.Contains(strings.ToLower(queries), "for update") {
				t.Fatalf("not a single read-only snapshot: %s; %v", queries, traceErr)
			}
			attachmentListRead(t, w, "bob", host+"/attachments", 404)
		})
	}
}

func attachmentSnapshotWorld(t *testing.T, trace *attachmentSnapshotTrace) issueWorld {
	t.Helper()
	dbURL := pgtest.NewDatabase(t)
	pool := openPool(t, dbURL)
	cfg := testConfig(t, dbURL, false)
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
	return prepareIssueWorld(t, issueWorld{base: runBuiltApp(t, a), pool: pool, contract: apitest.Load(t), tokens: map[string]string{}, ids: map[string]uuid.UUID{}})
}
