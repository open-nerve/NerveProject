package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

func preparationReadBarrier(t *testing.T, w issueWorld, trace *attachmentSnapshotTrace, req *http.Request) (<-chan answer, func()) {
	t.Helper()
	req.Header.Set(httpserver.HeaderRequestID, "attachment-snapshot-reader")
	trace.gate.deadline = time.Now().Add(10 * time.Second)
	var once sync.Once
	resume := func() { once.Do(func() { close(trace.gate.open) }) }
	t.Cleanup(resume)
	response := sendInBackground(req)
	select {
	case <-trace.gate.held:
	case got := <-response:
		t.Fatalf("preparation read ended before the real row query: %+v", got)
	case <-time.After(5 * time.Second):
		t.Fatal("preparation read missed its row query barrier")
	}
	trace.mu.Lock()
	pid := trace.pid
	trace.mu.Unlock()
	var snapshot bool
	if err := w.pool.QueryRow(t.Context(), `SELECT EXISTS(SELECT 1 FROM pg_stat_activity
WHERE pid=$1 AND datname=current_database() AND xact_start IS NOT NULL AND backend_xmin IS NOT NULL)`, pid).Scan(&snapshot); err != nil || !snapshot {
		t.Fatalf("owner read did not establish the paused transaction's snapshot: pid=%d snapshot=%t err=%v", pid, snapshot, err)
	}
	t.Logf("preparation read after host authorization, before ListDraftPreparations: actual backend PID=%d", pid)
	return response, resume
}

func assertPreparationSnapshot(t *testing.T, trace *attachmentSnapshotTrace) {
	t.Helper()
	trace.mu.Lock()
	queries, err, multiple := strings.ToLower(strings.Join(trace.queries, "\n")), trace.err, trace.multiple
	trace.mu.Unlock()
	if err != nil || multiple || !strings.Contains(queries, "begin isolation level repeatable read read only") || !strings.Contains(queries, "-- name: getdraft :one") || !strings.Contains(queries, "-- name: listdraftpreparations :many") {
		t.Fatalf("authorization/list did not share one production read-only snapshot: %s; %v", queries, err)
	}
	for _, lock := range []string{"for update", "for share", "for no key update", "for key share", "pg_advisory"} {
		if strings.Contains(queries, lock) {
			t.Fatal("preparation read acquired a write lock", queries)
		}
	}
}

func workspaceMemberHTTP(t *testing.T, w issueWorld, who string) uuid.UUID {
	t.Helper()
	var list struct {
		Data []struct {
			ID     uuid.UUID
			Member struct{ ID uuid.UUID }
		}
	}
	decodeAnswer(t, w.send(t, "alice", "GET", "/api/v0/workspaces/acme/members", "", 200), &list)
	for _, row := range list.Data {
		if row.Member.ID == w.ids[who] {
			return row.ID
		}
	}
	t.Fatalf("no workspace membership for %s", who)
	return uuid.Nil()
}

func TestDraftAssetPreparationsSnapshot(t *testing.T) {
	for _, change := range []string{"draft deleted", "workspace deleted", "project deleted", "moved", "published", "guest", "left", "deactivated", "logged out"} {
		t.Run(change, func(t *testing.T) {
			trace := &attachmentSnapshotTrace{gate: newGate(), pauseSQL: "ListDraftPreparations"}
			w := attachmentSnapshotWorld(t, trace)
			var session authTokens
			if change == "logged out" {
				decodeAnswer(t, w.send(t, "", "POST", "/api/v0/auth/login", `{"email":"bob@example.com","password":"Tr0ub4dor&3"}`, 200), &session)
				w.tokens["bob"] = session.AccessToken
			}
			d := createDraftHTTP(t, w, "bob", fmt.Sprintf(`{"project_id":%q,"payload":{"name":"Snapshot"}}`, w.web))
			first := uploadedAssetHTTP(t, w, "bob", "DRAFT_ISSUE_DESCRIPTION", attachmentScope("draft", d.ID), "old.png", "image/png", assetTestPNG(t))
			uploadedAssetHTTP(t, w, "bob", "DRAFT_ISSUE_DESCRIPTION", attachmentScope("draft", d.ID), "second.png", "image/png", assetTestPNG(t))
			member := workspaceMemberHTTP(t, w, "bob")
			req := newRequest(t, "GET", w.base+preparationsPath(d.ID)+"?limit=1", w.tokens["bob"], nil)
			response, resume := preparationReadBarrier(t, w, trace, req)
			// The paused reader holds no write lock: each real HTTP mutation must
			// commit before it is released, including publication and parent deletion.
			nextStatus := 404
			switch change {
			case "draft deleted":
				draftRequest(t, w, "bob", "DELETE", draftPath(d.ID), "", 204)
			case "workspace deleted":
				w.send(t, "alice", "DELETE", "/api/v0/workspaces/acme", "", 204)
			case "project deleted":
				w.send(t, "alice", "DELETE", "/api/v0/projects/"+w.web.String(), "", 204)
			case "moved":
				draftRequest(t, w, "bob", "PATCH", draftPath(d.ID), `{"project_id":null}`, 200)
				nextStatus = 200
			case "published":
				draftRequest(t, w, "bob", "POST", publishPath(d.ID), "", 201)
			case "guest":
				w.send(t, "alice", "PATCH", "/api/v0/workspace-members/"+member.String(), `{"role":5}`, 200)
			case "left":
				w.send(t, "bob", "POST", "/api/v0/workspaces/acme/leave", "", 204)
			case "deactivated":
				w.send(t, "bob", "POST", "/api/v0/me/deactivate", "", 204)
				nextStatus = 401
			case "logged out":
				w.send(t, "", "POST", "/api/v0/auth/logout", fmt.Sprintf(`{"refresh_token":%q}`, session.RefreshToken), 204)
				nextStatus = 401
			}
			resume()
			got := receiveWithin(t, response, 5*time.Second, "preparation snapshot after concurrent "+change)
			if got.err != nil || got.res.StatusCode != 200 {
				t.Fatalf("mixed owner/asset snapshot: %+v", got)
			}
			w.contract.CheckResponse(t, req, got.res)
			var page preparationPageHTTP
			decodeAnswer(t, string(got.body), &page)
			if len(page.Data) != 1 || page.Data[0].ID != first.Asset.ID || page.NextCursor == nil {
				t.Fatalf("mixed data snapshot: %s", got.body)
			}
			assertPreparationSnapshot(t, trace)
			raw := assetAPICall(t, w, "bob", "GET", preparationsPath(d.ID)+"?limit=1&cursor="+url.QueryEscape(*page.NextCursor), "", nextStatus)
			if nextStatus == 404 && problemCode(t, []byte(raw)) != "asset.not_found" {
				t.Fatal(raw)
			}
			if nextStatus == 200 {
				var next preparationPageHTTP
				decodeAnswer(t, raw, &next)
				if len(next.Data) != 1 || next.NextCursor != nil {
					t.Fatal("fresh null-project scope lost preparations", next)
				}
			}
			if change == "guest" {
				draftRequest(t, w, "bob", "GET", draftPath(d.ID), "", 403)
				w.send(t, "alice", "PATCH", "/api/v0/workspace-members/"+member.String(), `{"role":15}`, 200)
				if len(preparationsHTTP(t, w, "bob", d.ID, "").Data) != 2 {
					t.Fatal("restored author cannot rediscover preparations")
				}
			}
		})
	}
}

func TestDraftAssetPreparationsCancelReleasesSnapshot(t *testing.T) {
	trace := &attachmentSnapshotTrace{gate: newGate(), pauseSQL: "ListDraftPreparations"}
	w := attachmentSnapshotWorld(t, trace)
	d := createDraftHTTP(t, w, "bob", `{"payload":{}}`)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	req := newRequest(t, "GET", w.base+preparationsPath(d.ID), w.tokens["bob"], nil).WithContext(ctx)
	response, resume := preparationReadBarrier(t, w, trace, req)
	defer resume()
	trace.mu.Lock()
	pid := trace.pid
	trace.mu.Unlock()
	cancel()
	got := receiveWithin(t, response, 5*time.Second, "canceled preparation request")
	if !errors.Is(got.err, context.Canceled) {
		t.Fatalf("request ignored cancellation: %+v", got)
	}
	requestEventuallyWithin(t, "canceled preparation releases its exact backend transaction", 5*time.Second, func(ctx context.Context) bool {
		var held bool
		if err := w.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE pid=$1 AND xact_start IS NOT NULL)", pid).Scan(&held); err != nil {
			t.Fatal(err)
		}
		return !held
	})
	trace.mu.Lock()
	err := trace.err
	trace.mu.Unlock()
	if !errors.Is(err, context.Canceled) {
		t.Fatal("row barrier did not receive request cancellation", err)
	}
	if page := preparationsHTTP(t, w, "bob", d.ID, ""); len(page.Data) != 0 {
		t.Fatal(page)
	}
}

func TestDraftAssetPreparationsOtherWorkspaceUnaffected(t *testing.T) {
	w := newIssueWorld(t)
	acme := createDraftHTTP(t, w, "bob", `{"payload":{}}`)
	prepareAssetHTTP(t, w, "bob", "DRAFT_ISSUE_ATTACHMENT", attachmentScope("draft", acme.ID), "acme.txt", "text/plain", 1)
	w.send(t, "bob", "POST", "/api/v0/workspaces", `{"name":"Other","slug":"other"}`, 201)
	var other draftHTTP
	decodeAnswer(t, draftRequest(t, w, "bob", "POST", "/api/v0/workspaces/other/drafts", `{"payload":{}}`, 201), &other)
	prepareAssetHTTP(t, w, "bob", "DRAFT_ISSUE_ATTACHMENT", attachmentScope("draft", other.ID), "other.txt", "text/plain", 1)
	before := preparationsHTTP(t, w, "bob", other.ID, "")
	w.send(t, "alice", "PATCH", "/api/v0/workspace-members/"+workspaceMemberHTTP(t, w, "bob").String(), `{"role":5}`, 200)
	preparationRefusal(t, w, "bob", acme.ID)
	after := preparationsHTTP(t, w, "bob", other.ID, "")
	if len(after.Data) != 1 || len(before.Data) != 1 || after.Data[0].ID != before.Data[0].ID || after.Data[0].Name != "other.txt" {
		t.Fatal("one workspace's role change affected another preparation scope", after)
	}
}
