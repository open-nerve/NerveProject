package bootstrap

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
	"time"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func TestAssetCompleteRaces(t *testing.T) {
	for _, failLoser := range []bool{false, true} {
		t.Run(fmt.Sprintf("loser external failure=%t", failLoser), func(t *testing.T) {
			attempts := make(chan *assetSealAttempt, 2)
			w, store := newAssetCompleteWorld(t, func(c *assetComponents) { c.sealing.Sealer = controlledAssetSeal(c.store, attempts) })
			item := w.newWorkItem(t)
			data := []byte("both requests seal these bytes")
			prepared := prepareAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", fmt.Sprintf(`{"kind":"issue","id":%q}`, item), "same.txt", "text/plain", len(data))
			putAssetHTTP(t, w, prepared, data, 204)
			firstRequest := newRequest(t, "POST", w.base+completeAssetPath(prepared.Asset.ID), w.tokens["bob"], []byte(`{}`))
			firstResponse := sendInBackground(firstRequest)
			first := awaitAssetSeal(t, attempts, firstResponse)
			defer first.resume(nil)
			secondRequest := newRequest(t, "POST", w.base+completeAssetPath(prepared.Asset.ID), w.tokens["bob"], []byte(`{}`))
			secondResponse := sendInBackground(secondRequest)
			second := awaitAssetSeal(t, attempts, secondResponse)
			defer second.resume(nil)
			pending, err := assetpg.New(w.pool).FindAsset(t.Context(), prepared.Asset.ID)
			if err != nil || pending.Uploaded || len(pending.Metadata.Candidates) != 2 || first.key == second.key {
				t.Fatalf("independent candidates missing: %+v %v", pending, err)
			}
			second.resume(nil)
			winner := receiveWithin(t, secondResponse, 5*time.Second, "winning complete")
			if winner.err != nil || winner.res.StatusCode != 200 {
				t.Fatalf("winner=%+v", winner)
			}
			w.contract.CheckResponse(t, secondRequest, winner.res)
			published, err := assetpg.New(w.pool).FindAsset(t.Context(), prepared.Asset.ID)
			if err != nil || published.Metadata.Final == nil || published.Metadata.Final.Key != second.key || len(published.Metadata.Candidates) != 1 || published.Metadata.Candidates[0].Key != first.key {
				t.Fatalf("winner lost another candidate: %+v %v", published, err)
			}
			var externalError error
			if failLoser {
				externalError = domain.ErrObjectChanged
			}
			first.resume(externalError)
			loser := receiveWithin(t, firstResponse, 5*time.Second, "losing complete")
			if loser.err != nil || loser.res.StatusCode != 200 || !bytes.Equal(loser.body, winner.body) {
				t.Fatalf("loser did not return the authorized winner: %+v", loser)
			}
			w.contract.CheckResponse(t, firstRequest, loser.res)
			after, err := assetpg.New(w.pool).FindAsset(t.Context(), prepared.Asset.ID)
			if err != nil || !reflect.DeepEqual(after, published) {
				t.Fatalf("loser overwrote final/metadata/audit: %+v %v", after, err)
			}
			for _, key := range []domain.ObjectKey{first.key, second.key} {
				if _, err := store.Inspect(t.Context(), key); err != nil {
					t.Fatalf("sealed candidate lost before recovery: %v", err)
				}
			}
			_, actual := readAssetHTTP(t, w, "bob", prepared.Asset.ID)
			if !bytes.Equal(actual, data) {
				t.Fatal("published bytes changed")
			}
		})
	}
}

func TestAssetCompleteRechecksHost(t *testing.T) {
	attempts := make(chan *assetSealAttempt, 1)
	w, store := newAssetCompleteWorld(t, func(c *assetComponents) { c.sealing.Sealer = controlledAssetSeal(c.store, attempts) })
	item := w.newWorkItemBy(t, "bob", w.web, `{"name":"Still visible after demotion"}`)
	prepared := prepareAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", fmt.Sprintf(`{"kind":"issue","id":%q}`, item), "x.txt", "text/plain", 3)
	putAssetHTTP(t, w, prepared, []byte("old"), 204)
	req := newRequest(t, "POST", w.base+completeAssetPath(prepared.Asset.ID), w.tokens["bob"], []byte(`{}`))
	response := sendInBackground(req)
	attempt := awaitAssetSeal(t, attempts, response)
	defer attempt.resume(nil)
	membership := queryIDs(t, w.pool, "SELECT id FROM project_members WHERE project_id=$1 AND member_id=$2 AND is_active AND deleted_at IS NULL", w.web, w.ids["bob"])[0]
	w.send(t, "alice", "PATCH", "/api/v0/project-members/"+membership.String(), `{"role":5}`, 200)
	assetAPICall(t, w, "bob", "GET", "/api/v0/assets/"+prepared.Asset.ID.String(), "", 200)
	attempt.resume(nil)
	got := receiveWithin(t, response, 5*time.Second, "revoked complete")
	if got.err != nil || got.res.StatusCode != 403 || problemCode(t, got.body) != "forbidden" {
		t.Fatalf("published after write access was revoked: %+v", got)
	}
	w.contract.CheckResponse(t, req, got.res)
	row, err := assetpg.New(w.pool).FindAsset(t.Context(), prepared.Asset.ID)
	if err != nil || row.Uploaded || row.Metadata.Final != nil || len(row.Metadata.Candidates) != 1 || row.Metadata.Candidates[0].Key != attempt.key {
		t.Fatalf("revoked result lost recovery: %+v %v", row, err)
	}
	if _, err := store.Inspect(t.Context(), attempt.key); err != nil {
		t.Fatal(err)
	}
}
