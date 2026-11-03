package bootstrap

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

func TestContentAssetRaces(t *testing.T) {
	for _, cancelAtLast := range []bool{false, true} {
		t.Run(fmt.Sprintf("reverse references cancel=%t", cancelAtLast), func(t *testing.T) {
			w := newObservedIssueWorldWith(t, func(c *config.Config) {
				c.Issue.WriteGate = config.WriteGateConfig{PerProject: 1, MaxQueue: 0, MaxWait: time.Second}
			})
			item := w.newWorkItem(t)
			low := uploadedAssetHTTP(t, w, "bob", "ISSUE_DESCRIPTION", assetScopeBody("issue", item), "low.png", "image/png", assetTestPNG(t))
			high := uploadedAssetHTTP(t, w, "bob", "ISSUE_DESCRIPTION", assetScopeBody("issue", item), "high.png", "image/png", assetTestPNG(t))
			if low.Asset.ID.Compare(high.Asset.ID) >= 0 {
				t.Fatal("fixture needs two distinct ordered assets")
			}
			before := contentAssetState(t, w)
			lower := holding(t, w.pool, "SELECT id FROM file_assets WHERE id=$1 FOR SHARE", low.Asset.ID)
			defer requestRelease(t, lower)
			upper := holding(t, w.pool, "SELECT id FROM file_assets WHERE id=$1 FOR SHARE", high.Asset.ID)
			defer requestRelease(t, upper)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			body := fmt.Sprintf(`{"description_html":%q}`, issueImageHTML(high.Asset.ID, low.Asset.ID))
			path := "/api/v0/issues/" + item.String()
			req := newRequest(t, "PATCH", w.base+path, w.tokens["bob"], []byte(body)).WithContext(ctx)
			probe := w.observe(req)
			response := sendInBackground(req)
			pid := requestBlocked(t, w.pool, probe, lower.Conn().PgConn().PID(), "file_assets")
			if got := requestRowLock(t, w.pool, w.pool, probe, "projects WHERE id=$1", w.web); got != "FOR NO KEY UPDATE" {
				t.Fatal("content did not own project before assets", got)
			}
			if tables := requestWrittenTables(t, w.pool, pid); len(tables) != 0 {
				t.Fatal("content wrote before complete set", tables)
			}
			requestRelease(t, lower)
			lastPID := requestBlocked(t, w.pool, probe, upper.Conn().PgConn().PID(), "file_assets")
			if lastPID != pid {
				t.Fatal("content switched transaction between asset locks", pid, lastPID)
			}
			if got := requestRowLock(t, w.pool, w.pool, probe, "file_assets WHERE id=$1", low.Asset.ID); got != "FOR UPDATE" {
				t.Fatal("higher asset preceded lower reference", got)
			}
			if tables := requestWrittenTables(t, w.pool, pid); len(tables) != 0 {
				t.Fatal("content wrote before last asset", tables)
			}
			t.Logf("content request=%s pid=%d ordered assets=%s,%s last blocker=%d", probe.id, pid, low.Asset.ID, high.Asset.ID, upper.Conn().PgConn().PID())
			if cancelAtLast {
				cancel()
				got := receiveWithin(t, response, 5*time.Second, "cancelled content request")
				if got.err == nil {
					t.Fatal("cancelled content succeeded", got)
				}
				requestEventually(t, "cancelled content releases its pool connection", func(context.Context) bool { return w.requests.appPool.Stat().AcquiredConns() == 0 })
				if !reflect.DeepEqual(before, contentAssetState(t, w)) {
					t.Fatal("cancellation retained partial content or binding")
				}
				requestRelease(t, upper)
				w.send(t, "bob", "PATCH", path, body, 200) // Same gate slot and full set must be reusable.
			} else {
				requestRelease(t, upper)
				imageRaceAnswer(t, w, req, response, 200)
			}
			a, err := assetpg.New(w.pool).FindAsset(t.Context(), low.Asset.ID)
			if err != nil {
				t.Fatal(err)
			}
			b, err := assetpg.New(w.pool).FindAsset(t.Context(), high.Asset.ID)
			if err != nil || a.BoundAt == nil || b.BoundAt == nil || !a.BoundAt.Equal(*b.BoundAt) {
				t.Fatal("retry or completion did not bind the full set at one moment", a, b, err)
			}
		})
	}
	for _, change := range []string{"issue delete", "issue archive", "project archive"} {
		for _, changeFirst := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s first=%t", change, changeFirst), func(t *testing.T) {
				w := newObservedIssueWorld(t)
				item := w.newWorkItem(t)
				path := "/api/v0/issues/" + item.String()
				if change == "issue archive" {
					w.send(t, "alice", "PATCH", path, fmt.Sprintf(`{"state_id":%q}`, w.done), 200)
				}
				file := uploadedAssetHTTP(t, w, "bob", "ISSUE_DESCRIPTION", assetScopeBody("issue", item), "pending-reference.png", "image/png", assetTestPNG(t))
				changeMethod, changePath, changeBody, changeStatus := "DELETE", path, "", 204
				barrierTable := "issues"
				if change == "issue archive" {
					changeMethod, changePath, changeStatus = "POST", path+"/archive", 200
				}
				if change == "project archive" {
					changeMethod, changePath, changeStatus, barrierTable = "POST", "/api/v0/projects/"+w.web.String()+"/archive", 200, "projects"
				}
				changeReq := newRequest(t, changeMethod, w.base+changePath, w.tokens["alice"], []byte(changeBody))
				bindReq := newRequest(t, "PATCH", w.base+path, w.tokens["bob"], []byte(fmt.Sprintf(`{"description_html":%q}`, issueImageHTML(file.Asset.ID))))
				changeProbe, bindProbe := w.observe(changeReq), w.observe(bindReq)
				var barrier pgx.Tx
				var changed, bound <-chan answer
				if changeFirst {
					barrier = lockedTables(t, w.pool, barrierTable, "SHARE")
					defer requestRelease(t, barrier)
					changed = sendInBackground(changeReq)
					pid := requestBlocked(t, w.pool, changeProbe, barrier.Conn().PgConn().PID(), barrierTable)
					bound = sendInBackground(bindReq)
					bindPID := requestBlocked(t, w.pool, bindProbe, pid, "projects")
					t.Logf("parent first request=%s pid=%d content=%s pid=%d", changeProbe.id, pid, bindProbe.id, bindPID)
				} else {
					barrier = holding(t, w.pool, "SELECT id FROM file_assets WHERE id=$1 FOR SHARE", file.Asset.ID)
					defer requestRelease(t, barrier)
					bound = sendInBackground(bindReq)
					pid := requestBlocked(t, w.pool, bindProbe, barrier.Conn().PgConn().PID(), "file_assets")
					changed = sendInBackground(changeReq)
					changePID := requestBlocked(t, w.pool, changeProbe, pid, "projects")
					t.Logf("content first request=%s pid=%d parent=%s pid=%d", bindProbe.id, pid, changeProbe.id, changePID)
				}
				requestRelease(t, barrier)
				imageRaceAnswer(t, w, changeReq, changed, changeStatus)
				want := 200
				if changeFirst {
					want = 409
					if change == "issue delete" {
						want = 404
					}
				}
				imageRaceAnswer(t, w, bindReq, bound, want)
				row, err := assetpg.New(w.pool).FindAsset(t.Context(), file.Asset.ID)
				if err != nil || (row.BoundAt == nil) != changeFirst {
					t.Fatal("parent ordering consumed a refused reference", row, err)
				}
				if (row.DeletedAt != nil) != (change == "issue delete") {
					t.Fatal("parent deletion/archive conflated", row)
				}
			})
		}
	}
}
