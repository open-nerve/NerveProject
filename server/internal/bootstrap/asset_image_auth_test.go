package bootstrap

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

// A hidden avatar must take precedence over an unuploaded cover in either UUID
// order. The reply is identical to a missing avatar, including problem headers.
func TestImageAuthHiddenPriority(t *testing.T) {
	for _, hiddenFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("hidden first=%t", hiddenFirst), func(t *testing.T) {
			w := newIssueWorld(t)
			var hidden, pending uuid.UUID
			prepareHidden := func() {
				hidden = uploadedAssetHTTP(t, w, "bob", "USER_AVATAR", `{"kind":"user"}`, "hidden.png", "image/png", assetTestPNG(t)).Asset.ID
			}
			preparePending := func() {
				pending = prepareAssetHTTP(t, w, "alice", "USER_COVER", `{"kind":"user"}`, "pending.png", "image/png", len(assetTestPNG(t))).Asset.ID
			}
			if hiddenFirst {
				prepareHidden()
				preparePending()
			} else {
				preparePending()
				prepareHidden()
			}
			if (hidden.Compare(pending) < 0) != hiddenFirst {
				t.Fatal("fixture lacks reversed UUID order")
			}
			waitForCompletedKinds(t, w.pool, map[string]int{"asset.cleanup": 1, "cleanup.purge_deleted": 1}, &lockedBuffer{})
			before := profileImageSnapshot(t, w)
			checkImageRefusalsEqual(t, w, "alice", "/api/v0/me", fmt.Sprintf(`{"avatar_asset_id":%q,"cover_image_asset_id":%q}`, hidden, pending), fmt.Sprintf(`{"avatar_asset_id":%q,"cover_image_asset_id":%q}`, uuid.New(), pending), 404, "asset.not_found")
			if profileImageSnapshot(t, w) != before {
				t.Fatal("mixed hidden input partially changed bindings")
			}
			checkImageRefusalsEqual(t, w, "bob", "/api/v0/workspaces/acme", fmt.Sprintf(`{"logo_asset_id":%q}`, pending), fmt.Sprintf(`{"logo_asset_id":%q}`, uuid.New()), 403, "forbidden")
		})
	}
}

func checkImageRefusalsEqual(t *testing.T, w issueWorld, who, path, firstBody, secondBody string, status int, code string) {
	t.Helper()
	var first *http.Response
	var payload []byte
	for _, body := range []string{firstBody, secondBody} {
		req := newRequest(t, "PATCH", w.base+path, w.tokens[who], []byte(body))
		req.Header.Set("X-Request-ID", "00000000-0000-4000-8000-000000000001")
		res, raw := send(t, req)
		w.contract.CheckResponse(t, req, res)
		if res.StatusCode != status || problemCode(t, raw) != code {
			t.Fatal(res.StatusCode, string(raw))
		}
		res.Header.Del("Date") // Transport wall time is not a resource disclosure.
		if first == nil {
			first, payload = res, raw
		} else if !bytes.Equal(payload, raw) || !reflect.DeepEqual(first.Header, res.Header) {
			t.Fatal("different hidden problem body/headers", first.Header, res.Header, string(payload), string(raw))
		}
	}
}

// Project images, work-item writes and generic asset content must consume the
// same gate. Cancellation must release both the active SQL and its gate slot.
func TestImageAuthSharedGateAndCancel(t *testing.T) {
	w := newObservedIssueWorldWith(t, func(c *config.Config) {
		c.Issue.WriteGate = config.WriteGateConfig{PerProject: 1, MaxQueue: 0, MaxWait: time.Second}
	})
	p := "/api/v0/projects/" + w.web.String()
	image := bindRaceImage(t, w, "alice", "PROJECT_COVER", assetScopeBody("project", w.web), p, "cover_image")
	item := w.newWorkItem(t)
	file := uploadedAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", assetScopeBody("issue", item), "gate.txt", "text/plain", []byte("shared gate"))
	for _, op := range []string{"get", "list", "update"} {
		t.Run(op, func(t *testing.T) {
			method, path, body := "GET", p, ""
			if op == "list" {
				path = "/api/v0/workspaces/acme/projects"
			}
			if op == "update" {
				method, body = "PATCH", `{}`
			}
			held := holding(t, w.pool, "SELECT id FROM file_assets WHERE id=$1 FOR SHARE", image)
			defer requestRelease(t, held)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			req := newRequest(t, method, w.base+path, w.tokens["alice"], []byte(body)).WithContext(ctx)
			probe := w.observe(req)
			result := sendInBackground(req)
			requestBlocked(t, w.pool, probe, held.Conn().PgConn().PID(), "file_assets")
			for _, target := range []struct{ method, path, body string }{
				{"PATCH", "/api/v0/issues/" + item.String(), `{"name":"Busy"}`},
				{"GET", "/api/v0/assets/" + file.Asset.ID.String() + "/content", ""},
			} {
				busy := newRequest(t, target.method, w.base+target.path, w.tokens["bob"], []byte(target.body))
				got := imageRaceAnswer(t, w, busy, sendInBackground(busy), 503)
				if problemCode(t, got.body) != "server_busy" || got.res.Header.Get("Retry-After") != "1" {
					t.Fatal("lost shared gate refusal", got)
				}
			}
			w.send(t, "olga", "GET", p, "", 404) // Precheck does not enter the occupied gate.
			cancel()
			got := receiveWithin(t, result, 5*time.Second, "cancelled image response")
			if got.err == nil {
				t.Fatal("cancelled request succeeded", got)
			}
			requestRelease(t, held)
			requestEventually(t, "cancelled image releases connection", func(context.Context) bool { return w.requests.appPool.Stat().AcquiredConns() == 0 })
			checkImageReply(t, w, w.send(t, "alice", "GET", p, "", 200), "cover_image", image, assetTestPNG(t))
			w.send(t, "bob", "PATCH", "/api/v0/issues/"+item.String(), `{"name":"After cancellation"}`, 200)
		})
	}
}
