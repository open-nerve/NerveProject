package bootstrap

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
	"time"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

func TestDraftAssetsLockBeforeMomentAndWrites(t *testing.T) {
	for _, publish := range []bool{false, true} {
		t.Run(fmt.Sprint("publish=", publish), func(t *testing.T) {
			w := newIssueWorld(t)
			d := createDraftHTTP(t, w, "alice", fmt.Sprintf(`{"project_id":%q,"payload":{"name":"Locked"}}`, w.web))
			image, last := draftAsset(t, w, d.ID, true), draftAsset(t, w, d.ID, false)
			if image.Asset.ID.Compare(last.Asset.ID) >= 0 {
				t.Fatal("fixture must put referenced image before final asset lock")
			}
			held := holding(t, w.pool, "SELECT 1 FROM file_assets WHERE id=$1 FOR UPDATE", last.Asset.ID)
			clock := &eventCountingClock{base: clockAt(time.Date(2046, 8, 9, 10, 11, 12, 0, time.UTC))}
			tx := requestTx(w.pool)
			route := newIssueRouteWithClock(t, w.pool, tx, clock)
			body := draftPayloadBody(t, map[string]any{"name": "Locked", "description_html": issueImageHTML(image.Asset.ID)})
			method, path, want := "PATCH", draftPath(d.ID), 200
			if publish {
				method, path, body, want = "POST", publishPath(d.ID), "", 201
			}
			var response exchange
			collect := requestBackground(t, w.sending(&response, route, "alice", method, path, body), func() { requestRelease(t, held) })
			requestBlocked(t, w.pool, tx, held.Conn().PgConn().PID(), "file_assets")
			if clock.calls.Load() != 0 || !pgtest.RowWaits(t, w.pool, "draft_issues", d.ID, "FOR UPDATE") || !pgtest.RowWaits(t, w.pool, "file_assets", image.Asset.ID, "FOR UPDATE") {
				t.Fatal("moment preceded final lock or set was not locked in UUID order")
			}
			for _, table := range requestWrittenTables(t, w.pool, requestPID(t, tx)) {
				var sequenceRelation bool
				if err := w.pool.QueryRow(t.Context(), `SELECT EXISTS(SELECT 1 FROM pg_class r LEFT JOIN pg_index i ON i.indexrelid=r.oid WHERE r.relname=$1 AND (r.oid='projects'::regclass OR i.indrelid='projects'::regclass))`, table).Scan(&sequenceRelation); err != nil {
					t.Fatal(err)
				}
				if !publish || !sequenceRelation {
					t.Fatal("business write preceded complete asset set", table)
				}
			}
			requestRelease(t, held)
			collect()
			w.answered(t, response, want)
			if clock.calls.Load() != 1 {
				t.Fatal("save read multiple business moments", clock.calls.Load())
			}
		})
	}
}

func TestDraftAssetPublishTwiceWithFiles(t *testing.T) {
	w := newObservedIssueWorld(t)
	d := createDraftHTTP(t, w, "alice", fmt.Sprintf(`{"project_id":%q,"payload":{"name":"Only once"}}`, w.web))
	file := draftAsset(t, w, d.ID, false)
	held := holding(t, w.pool, "SELECT 1 FROM file_assets WHERE id=$1 FOR UPDATE", file.Asset.ID)
	firstReq := newRequest(t, "POST", w.base+publishPath(d.ID), w.tokens["alice"], nil)
	firstProbe := w.observe(firstReq)
	first := sendInBackground(firstReq)
	pid := requestBlocked(t, w.pool, firstProbe, held.Conn().PgConn().PID(), "file_assets")
	secondReq := newRequest(t, "POST", w.base+publishPath(d.ID), w.tokens["alice"], nil)
	secondProbe := w.observe(secondReq)
	second := sendInBackground(secondReq)
	requestBlocked(t, w.pool, secondProbe, pid, "projects")
	release(t, held)
	a, b := receiveWithin(t, first, 5*time.Second, "first publish"), receiveWithin(t, second, 5*time.Second, "second publish")
	if a.err != nil || b.err != nil || a.res.StatusCode != 201 || b.res.StatusCode != 404 {
		t.Fatal("publish did not serialize", a, b)
	}
	w.contract.CheckResponse(t, firstReq, a.res)
	w.contract.CheckResponse(t, secondReq, b.res)
	var got storedIssue
	decodeAnswer(t, string(a.body), &got)
	if got.SequenceID != 1 || len(queryIDs(t, w.pool, "SELECT id FROM issues WHERE project_id=$1", w.web)) != 1 || attachmentActivityCount(t, w, got.ID) != 1 {
		t.Fatal("two creation graphs or duplicate activity")
	}
	row, err := assetpg.New(w.pool).FindAsset(t.Context(), file.Asset.ID)
	if err != nil || row.Placement.IssueID != got.ID {
		t.Fatal("winning file lost", row, err)
	}
}

// A successful final with a losing candidate is publishable. The old complete
// cannot attach itself again after the draft is consumed, and recovery retains
// both immutable objects until the normal cleanup protocol removes the loser.
func TestDraftAssetPublishWithLateCandidate(t *testing.T) {
	attempts := make(chan *assetSealAttempt, 2)
	w, store := newAssetCompleteWorld(t, func(c *assetComponents) { c.sealing.Sealer = controlledAssetSeal(c.store, attempts) })
	d := createDraftHTTP(t, w, "alice", fmt.Sprintf(`{"project_id":%q,"payload":{"name":"Winner"}}`, w.web))
	data := []byte("winner bytes")
	file := prepareAssetHTTP(t, w, "alice", "DRAFT_ISSUE_ATTACHMENT", attachmentScope("draft", d.ID), "winner.txt", "text/plain", len(data))
	putAssetHTTP(t, w, file, data, 204)
	firstReq := newRequest(t, "POST", w.base+completeAssetPath(file.Asset.ID), w.tokens["alice"], []byte(`{}`))
	firstResponse := sendInBackground(firstReq)
	first := awaitAssetSeal(t, attempts, firstResponse)
	defer first.resume(nil)
	secondReq := newRequest(t, "POST", w.base+completeAssetPath(file.Asset.ID), w.tokens["alice"], []byte(`{}`))
	secondResponse := sendInBackground(secondReq)
	second := awaitAssetSeal(t, attempts, secondResponse)
	defer second.resume(nil)
	second.resume(nil)
	winner := receiveWithin(t, secondResponse, 5*time.Second, "completed candidate")
	if winner.err != nil || winner.res.StatusCode != 200 {
		t.Fatal(winner)
	}
	w.contract.CheckResponse(t, secondReq, winner.res)
	var issue storedIssue
	decodeAnswer(t, draftRequest(t, w, "alice", "POST", publishPath(d.ID), "", 201), &issue)
	before, err := assetpg.New(w.pool).FindAsset(t.Context(), file.Asset.ID)
	if err != nil || before.Metadata.Final == nil || before.Metadata.Final.Key != second.key || len(before.Metadata.Candidates) != 1 || before.Metadata.Candidates[0].Key != first.key || before.Placement.IssueID != issue.ID {
		t.Fatal("migration lost published final or recovery candidate", before, err)
	}
	first.resume(nil)
	loser := receiveWithin(t, firstResponse, 5*time.Second, "late old draft candidate")
	if loser.err != nil || loser.res.StatusCode != 404 || problemCode(t, loser.body) != "asset.not_found" {
		t.Fatal("late complete reused consumed draft", loser)
	}
	w.contract.CheckResponse(t, firstReq, loser.res)
	after, err := assetpg.New(w.pool).FindAsset(t.Context(), file.Asset.ID)
	if err != nil || !reflect.DeepEqual(before, after) || attachmentActivityCount(t, w, issue.ID) != 1 {
		t.Fatal("late complete changed migrated state or emitted duplicate", after, err)
	}
	if _, err := store.Inspect(t.Context(), first.key); err != nil {
		t.Fatal("lost recoverable candidate bytes", err)
	}
	_, actual := readAssetHTTP(t, w, "bob", file.Asset.ID)
	if !bytes.Equal(actual, data) {
		t.Fatal("late complete changed published bytes")
	}
}

func TestDraftAssetsMoveWhole1001(t *testing.T) {
	w := newIssueWorld(t)
	d := createDraftHTTP(t, w, "alice", `{"payload":{}}`)
	image := draftAsset(t, w, d.ID, true)
	ids := []uuid.UUID{image.Asset.ID}
	for n := 0; n < 1000; n++ {
		file := prepareAssetHTTP(t, w, "alice", "DRAFT_ISSUE_ATTACHMENT", attachmentScope("draft", d.ID), "pending.txt", "text/plain", 1)
		ids = append(ids, file.Asset.ID)
	}
	body := fmt.Sprintf(`{"project_id":%q,"payload":{"description_html":%q}}`, w.web, issueImageHTML(image.Asset.ID))
	var got draftHTTP
	decodeAnswer(t, draftRequest(t, w, "alice", "PATCH", draftPath(d.ID), body, 200), &got)
	var count int
	if err := w.pool.QueryRow(t.Context(), "SELECT count(*) FROM file_assets WHERE id=ANY($1::uuid[]) AND project_id=$2 AND draft_issue_id=$3 AND updated_at=$4 AND updated_by_id=$5", ids, w.web, d.ID, got.UpdatedAt, w.ids["alice"]).Scan(&count); err != nil || count != 1001 {
		t.Fatal("move truncated full set", count, err)
	}
	row, err := assetpg.New(w.pool).FindAsset(t.Context(), image.Asset.ID)
	if err != nil || row.BoundAt == nil || !row.BoundAt.Equal(got.UpdatedAt) {
		t.Fatal("move omitted lower referenced image", err)
	}
}

func TestDraftAssetRaces(t *testing.T) {
	for _, publishFirst := range []bool{false, true} {
		t.Run(fmt.Sprintf("publish versus delete publish first=%t", publishFirst), func(t *testing.T) {
			w := newObservedIssueWorld(t)
			d := createDraftHTTP(t, w, "alice", fmt.Sprintf(`{"project_id":%q,"payload":{"name":"One outcome"}}`, w.web))
			file := draftAsset(t, w, d.ID, false)
			barrier := holding(t, w.pool, "SELECT id FROM file_assets WHERE id=$1 FOR SHARE", file.Asset.ID)
			defer requestRelease(t, barrier)
			publish := newRequest(t, "POST", w.base+publishPath(d.ID), w.tokens["alice"], nil)
			remove := newRequest(t, "DELETE", w.base+draftPath(d.ID), w.tokens["alice"], nil)
			publishProbe, removeProbe := w.observe(publish), w.observe(remove)
			firstReq, secondReq, firstProbe, secondProbe := remove, publish, removeProbe, publishProbe
			if publishFirst {
				firstReq, secondReq, firstProbe, secondProbe = publish, remove, publishProbe, removeProbe
			}
			first := sendInBackground(firstReq)
			pid := requestBlocked(t, w.pool, firstProbe, barrier.Conn().PgConn().PID(), "file_assets")
			second := sendInBackground(secondReq)
			secondPID := requestBlocked(t, w.pool, secondProbe, pid, "projects")
			t.Logf("draft first=%s pid=%d second=%s pid=%d", firstProbe.id, pid, secondProbe.id, secondPID)
			requestRelease(t, barrier)
			firstStatus := 204
			if publishFirst {
				firstStatus = 201
			}
			winner := imageRaceAnswer(t, w, firstReq, first, firstStatus)
			imageRaceAnswer(t, w, secondReq, second, 404)
			row, err := assetpg.New(w.pool).FindAsset(t.Context(), file.Asset.ID)
			if err != nil {
				t.Fatal(err)
			}
			issues := queryIDs(t, w.pool, "SELECT id FROM issues WHERE project_id=$1", w.web)
			if publishFirst {
				var item storedIssue
				decodeAnswer(t, string(winner.body), &item)
				if len(issues) != 1 || issues[0] != item.ID || row.Placement.IssueID != item.ID || row.Placement.DraftID != uuid.Nil() || row.DeletedAt != nil || attachmentActivityCount(t, w, item.ID) != 1 {
					t.Fatal("losing delete touched published graph", row, issues)
				}
				_, data := readAssetHTTP(t, w, "bob", file.Asset.ID)
				if string(data) != "draft bytes" {
					t.Fatal("publication lost bytes")
				}
			} else if len(issues) != 0 || row.Placement.DraftID != d.ID || row.DeletedAt == nil {
				t.Fatal("losing publish created graph or revived file", row, issues)
			}
		})
	}
}
