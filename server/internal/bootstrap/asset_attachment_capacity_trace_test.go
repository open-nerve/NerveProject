package bootstrap

import (
	"context"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

type attachmentCapacityTrace struct {
	requestQueryTracer
	list  listTrace
	mu    sync.Mutex
	names map[string][]string
}

func (tr *attachmentCapacityTrace) pool(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	source, err := tr.requests.pool(ctx, cfg)
	if err != nil {
		return nil, err
	}
	pc := source.Config()
	source.Close()
	pc.ConnConfig.Tracer = tr
	p, err := pgxpool.NewWithConfig(ctx, pc)
	if err == nil {
		tr.requests.appPool = p
	}
	return p, err
}

func (tr *attachmentCapacityTrace) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	id := httpserver.RequestID(ctx)
	if id != "" && tr.requests.lookup(id) != nil {
		name, _, _ := strings.Cut(data.SQL, "\n")
		tr.mu.Lock()
		tr.names[id] = append(tr.names[id], name)
		tr.mu.Unlock()
		if strings.Contains(data.SQL, "AS a(id, attachment_count)") || strings.HasPrefix(name, "-- name: AttachmentCounts ") || strings.HasPrefix(name, "-- name: ProjectAttachmentCounts ") {
			ctx = tr.list.TraceQueryStart(context.WithValue(ctx, listRequestKey{}, id), conn, data)
		}
	}
	return tr.requestQueryTracer.TraceQueryStart(ctx, conn, data)
}

func (tr *attachmentCapacityTrace) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	tr.list.TraceQueryEnd(ctx, conn, data)
	tr.requestQueryTracer.TraceQueryEnd(ctx, conn, data)
}

func (f attachmentCapacityFixture) read(t *testing.T, actor, path string, q url.Values, catalog int, projectCount int) issueListHTTP {
	t.Helper()
	f.assertColdStatistics(t)
	w := f.w
	req := newRequest(t, "GET", w.base+path+"?"+q.Encode(), w.tokens[actor], nil)
	probe := w.observe(req)
	started := time.Now()
	res, raw := send(t, req)
	elapsed := time.Since(started)
	if res.StatusCode != 200 || elapsed > 5*time.Second {
		t.Fatal("capacity request", res.StatusCode, elapsed, string(raw))
	}
	w.contract.CheckResponse(t, req, res)
	grouped := q.Get("group_by") != "" && q.Get("group_key") == ""
	p := checkListEnvelope(t, string(raw), grouped, grouped && q.Get("sub_group_by") != "")
	f.checkTrace(t, probe.id, p, catalog, projectCount, elapsed)
	f.assertColdStatistics(t)
	return p
}

func capacityEmitted(page issueListHTTP) []uuid.UUID {
	var ids []uuid.UUID
	for _, row := range page.Data {
		ids = append(ids, row.ID)
	}
	for _, g := range page.Groups {
		ids = append(ids, capacityEmitted(g)...)
	}
	for _, g := range page.Subgroups {
		ids = append(ids, capacityEmitted(g)...)
	}
	slices.SortFunc(ids, uuid.UUID.Compare)
	return slices.Compact(ids)
}

func (f attachmentCapacityFixture) checkTrace(t *testing.T, id string, page issueListHTTP, catalog, projectCount int, elapsed time.Duration) {
	t.Helper()
	var returned []uuid.UUID
	var chunks []int
	listQueries := 0
	for _, q := range f.trace.list.take(id) {
		if q.Error != "" || q.Ended.IsZero() {
			t.Fatal("SQL capture", q.Error, q.SQL)
		}
		switch {
		case strings.HasPrefix(q.SQL, "-- name: AttachmentCounts "):
			ids := q.Args[0].([]uuid.UUID)
			if len(ids) > 1000 {
				t.Fatal("oversized returned-ID batch", len(ids))
			}
			returned = append(returned, ids...)
		case strings.HasPrefix(q.SQL, "-- name: ProjectAttachmentCounts "):
			ids := q.Args[0].([]uuid.UUID)
			chunks = append(chunks, len(ids))
			if len(ids) > 1000 || q.Args[2] != int32(1000) {
				t.Fatal("oversized directory input", len(ids), q.Args[2])
			}
		default:
			listQueries++
			found := false
			for i, arg := range q.Args {
				if counts, ok := arg.([]int64); ok {
					found = true
					if len(counts) != catalog {
						t.Fatalf("full directory truncated: got %d want %d", len(counts), catalog)
					}
					ids, ok := q.Args[i-1].([]uuid.UUID)
					if !ok || len(ids) != catalog {
						t.Fatal("directory pair length")
					}
					for n, issue := range ids {
						if counts[n] != f.counts[issue] || counts[n] <= 0 {
							t.Fatal("directory count", issue, counts[n])
						}
					}
				}
			}
			if !found {
				t.Fatal("list SQL lacks count directory")
			}
		}
	}
	wantChunks := 0
	if catalog > 0 {
		wantChunks = (catalog/1000 + 1) * ((projectCount + 999) / 1000)
	}
	wantReturned := capacityEmitted(page)
	slices.SortFunc(returned, uuid.UUID.Compare)
	if len(chunks) != wantChunks || listQueries != 1 || !slices.Equal(returned, wantReturned) {
		t.Fatalf("bulk query shape: chunks=%v want=%d list=%d returned=%d want=%d", chunks, wantChunks, listQueries, len(returned), len(wantReturned))
	}
	f.trace.mu.Lock()
	queries := len(f.trace.names[id])
	f.trace.mu.Unlock()
	if queries > 40+wantChunks {
		t.Fatal("capacity N+1", queries)
	}
	t.Logf("capacity: project_inputs=%d catalog=%d paired_payload_bytes=%d SQL=%d asset_project_chunks=%v returned_unique=%d elapsed=%s", projectCount, catalog, catalog*24, queries, chunks, len(returned), elapsed)
}
