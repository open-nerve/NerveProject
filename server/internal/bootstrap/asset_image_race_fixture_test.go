package bootstrap

import (
	"fmt"
	"net/http"
	"testing"
	"time"
	"uuid"
)

func bindRaceImage(t *testing.T, w issueWorld, actor, purpose, scope, path, field string) uuid.UUID {
	t.Helper()
	id := uploadedAssetHTTP(t, w, actor, purpose, scope, "race.png", "image/png", assetTestPNG(t)).Asset.ID
	w.send(t, actor, "PATCH", path, fmt.Sprintf(`{%q:%q}`, field+"_asset_id", id), 200)
	return id
}

func imageRaceAnswer(t *testing.T, w issueWorld, req *http.Request, ch <-chan answer, want int) answer {
	t.Helper()
	got := receiveWithin(t, ch, 5*time.Second, "image transaction completion")
	if got.err != nil || got.res.StatusCode != want {
		t.Fatalf("%s %s: status=%v error=%v body=%s, want %d", req.Method, req.URL.Path, got.res, got.err, got.body, want)
	}
	w.contract.CheckResponse(t, req, got.res)
	return got
}
