package bootstrap

import (
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

func TestListAttachmentCountSnapshot(t *testing.T) {
	for _, shape := range []string{"FlatIssues", "GroupIssues", "SubgroupIssues"} {
		for _, stage := range []string{shape, "AttachmentCounts"} {
			t.Run(shape+"/"+stage, func(t *testing.T) {
				trace := &attachmentSnapshotTrace{gate: newGate(), pauseSQL: stage}
				w := attachmentSnapshotWorld(t, trace)
				a, b, c := w.newWorkItem(t), w.newWorkItem(t), w.newWorkItem(t)
				put := func(id uuid.UUID) preparedAssetAnswer {
					return uploadedAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", attachmentScope("issue", id), "file.txt", "text/plain", []byte("abc"))
				}
				one, two := put(a), put(b)
				put(b)
				path := "/api/v0/projects/" + w.web.String() + "/issues?order_by=-attachment_count&limit=1"
				if shape != "FlatIssues" {
					path += "&group_by=label_id"
				}
				if shape == "SubgroupIssues" {
					path += "&sub_group_by=assignee_id"
				}
				req := newRequest(t, "GET", w.base+path, w.tokens["bob"], nil)
				req.Header.Set(httpserver.HeaderRequestID, "attachment-snapshot-reader")
				var once sync.Once
				resume := func() { once.Do(func() { close(trace.gate.open) }) }
				defer resume()
				trace.gate.deadline = time.Now().Add(10 * time.Second)
				answer := sendInBackground(req)
				select {
				case <-trace.gate.held:
				case got := <-answer:
					t.Fatalf("read missed barrier: %+v", got)
				case <-time.After(5 * time.Second):
					t.Fatal("read barrier deadline")
				}
				assetAPICall(t, w, "bob", "DELETE", "/api/v0/assets/"+one.Asset.ID.String(), "", 204)
				assetAPICall(t, w, "bob", "DELETE", "/api/v0/assets/"+two.Asset.ID.String(), "", 204)
				put(c)
				put(c)
				put(c)
				resume()
				got := receiveWithin(t, answer, 5*time.Second, "count sort snapshot")
				if got.err != nil || got.res.StatusCode != 200 {
					t.Fatalf("reader %+v", got)
				}
				w.contract.CheckResponse(t, req, got.res)
				cell := func(raw string) issueListHTTP {
					p := checkListEnvelope(t, raw, shape != "FlatIssues", shape == "SubgroupIssues")
					if p.Total != 3 {
						t.Fatal("total", p)
					}
					if shape != "FlatIssues" {
						if len(p.Groups) != 1 {
							t.Fatal(p)
						}
						p = p.Groups[0]
					}
					if shape == "SubgroupIssues" {
						if len(p.Subgroups) != 1 {
							t.Fatal(p)
						}
						p = p.Subgroups[0]
					}
					return p
				}
				checkAttachmentCountPage(t, cell(string(got.body)), []uuid.UUID{b}, map[uuid.UUID]int64{b: 2}, "-attachment_count", true)
				checkAttachmentCountPage(t, cell(w.send(t, "bob", "GET", path, "", 200)), []uuid.UUID{c}, map[uuid.UUID]int64{c: 3}, "-attachment_count", true)
				trace.mu.Lock()
				queries, traceErr, multiple := strings.Join(trace.queries, "\n"), trace.err, trace.multiple
				trace.mu.Unlock()
				begin := strings.Index(strings.ToLower(queries), "begin isolation level repeatable read read only")
				directory := strings.Index(queries, "-- name: ProjectAttachmentCounts :many")
				rows := strings.Index(queries, "-- name: "+shape+" :many")
				projection := strings.Index(queries, "-- name: AttachmentCounts :many")
				if traceErr != nil || multiple || begin < 0 || directory < begin || rows < directory || projection < rows || strings.Contains(strings.ToLower(queries), "for update") {
					t.Fatalf("mixed transaction count/order/cursor: %v %s", traceErr, queries)
				}
			})
		}
	}
}
