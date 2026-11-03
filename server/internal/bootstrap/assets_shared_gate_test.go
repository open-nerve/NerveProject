package bootstrap

import (
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

func TestAssetSharedGateInterleaving(t *testing.T) {
	for _, assetFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("asset first=%t", assetFirst), func(t *testing.T) {
			url := pgtest.NewDatabase(t)
			w := newIssueWorldOnRequests(t, url, openPool(t, url), func(c *config.Config) {
				c.Issue.WriteGate = config.WriteGateConfig{PerProject: 1, MaxQueue: 0, MaxWait: time.Second}
			}, &requestQueries{})
			item := w.newWorkItem(t)
			file := uploadedAssetHTTP(t, w, "bob", "ISSUE_DESCRIPTION", assetScopeBody("issue", item), "gate.png", "image/png", assetTestPNG(t))
			content := "/api/v0/assets/" + file.Asset.ID.String() + "/content"
			issuePath := "/api/v0/issues/" + item.String()
			firstMethod, firstPath, firstBody, firstWant := "PATCH", issuePath, `{"name":"Gate owner"}`, 200
			secondMethod, secondPath, secondBody := "GET", content, ""
			table := "issues"
			var holder pgx.Tx
			if assetFirst {
				firstMethod, firstPath, firstBody = secondMethod, secondPath, secondBody
				secondMethod, secondPath, secondBody = "PATCH", issuePath, `{"name":"Must not enter"}`
				table = "file_assets"
				holder = holding(t, w.pool, "SELECT id FROM file_assets WHERE id=$1 FOR SHARE", file.Asset.ID)
			} else {
				holder = lockedTables(t, w.pool, "issues", "SHARE")
			}
			defer requestRelease(t, holder)
			first := newRequest(t, firstMethod, w.base+firstPath, w.tokens["bob"], []byte(firstBody))
			probe := w.observe(first)
			result := sendInBackground(first)
			requestBlocked(t, w.pool, probe, holder.Conn().PgConn().PID(), table)
			second := newRequest(t, secondMethod, w.base+secondPath, w.tokens["bob"], []byte(secondBody))
			refused := receiveWithin(t, sendInBackground(second), time.Second, "shared gate's immediate refusal")
			if refused.err != nil || refused.res.StatusCode != 503 || problemCode(t, refused.body) != "server_busy" || refused.res.Header.Get("Retry-After") != "1" {
				t.Fatalf("separate gates admitted competing write: %+v", refused)
			}
			w.contract.CheckResponse(t, second, refused.res)
			// Alice may write the target but may not inspect Bob's preparation.
			// This must finish before the occupied project gate is considered.
			copyBody := fmt.Sprintf(`{"purpose":"ISSUE_ATTACHMENT","scope":%s}`, assetScopeBody("issue", item))
			foreign := newRequest(t, "POST", w.base+duplicateAssetPath(file.Asset.ID), w.tokens["alice"], []byte(copyBody))
			hidden := receiveWithin(t, sendInBackground(foreign), time.Second, "private source precheck")
			if hidden.err != nil || hidden.res.StatusCode != 404 || problemCode(t, hidden.body) != "asset.not_found" {
				t.Fatalf("hidden source entered the occupied gate: %+v", hidden)
			}
			w.contract.CheckResponse(t, foreign, hidden.res)
			if got := queryIDs(t, w.pool, "SELECT id FROM file_assets"); len(got) != 1 || got[0] != file.Asset.ID {
				t.Fatalf("refused copy inserted target: %v", got)
			}
			requestRelease(t, holder)
			got := receiveWithin(t, result, 5*time.Second, "gate owner completion")
			if got.err != nil || got.res.StatusCode != firstWant {
				t.Fatalf("gate owner=%+v", got)
			}
			w.contract.CheckResponse(t, first, got.res)
		})
	}
}
