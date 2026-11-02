package riveradapter

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func TestAssetSweepOutputStagesAndBounds(t *testing.T) {
	base := app.SweepResult{Cursor: app.SweepCursor{Version: 1, Stage: "rows", Cutoff: time.Now().UTC(), RuntimeCutoff: time.Now().Add(-24 * time.Hour).UTC()}}
	key, _ := domain.NewObjectKey('o', uuid.NewV7(), uuid.NewV7())
	for _, stage := range []string{"rows", "objects", "tmp", "done"} {
		r := base
		r.Cursor.Stage, r.Complete = stage, stage == "done"
		if stage == "objects" {
			r.Cursor.ObjectAfter, r.Cursor.Token = key, "adapter-token"
		}
		if stage == "tmp" {
			r.Cursor.Token = "temporary-token"
		}
		r.Failed = 2
		raw, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		got, found, err := decodeOutput([]byte(`{"output":`+string(raw)+`}`), "rows")
		if err != nil || !found || got != r {
			t.Fatalf("stage=%s roundtrip=%+v %v", stage, got, err)
		}
	}
	for _, mutate := range []func(*app.SweepResult){
		func(r *app.SweepResult) { r.Complete = true },
		func(r *app.SweepResult) { r.Cursor.Stage = "done" },
		func(r *app.SweepResult) { r.Failed = -1 },
		func(r *app.SweepResult) { r.Cursor.Stage, r.Cursor.After = "objects", uuid.NewV7() },
		func(r *app.SweepResult) { r.Cursor.Stage, r.Cursor.ObjectAfter = "objects", "invalid" },
		func(r *app.SweepResult) { r.Cursor.Stage, r.Cursor.ObjectAfter = "tmp", key },
		func(r *app.SweepResult) { r.Cursor.Stage, r.Cursor.Token = "objects", strings.Repeat("x", 8<<10) },
		func(r *app.SweepResult) { r.Cursor.Stage, r.Cursor.Token = "objects", strings.Repeat("\x00", 2000) },
		func(r *app.SweepResult) { r.Cursor.RuntimeCutoff = time.Time{} },
		func(r *app.SweepResult) { r.Cursor.Stage = "scope" },
	} {
		r := base
		mutate(&r)
		raw, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := decodeOutput([]byte(`{"output":`+string(raw)+`}`), "rows"); err == nil {
			t.Fatalf("accepted invalid progress: %s", raw)
		}
	}
	base.Cursor.Stage, base.Cursor.RuntimeCutoff = "scope", time.Time{}
	if err := validOutput(base, "scope"); err != nil {
		t.Fatal(err)
	}
	base.Cursor.ObjectAfter = key
	if err := validOutput(base, "scope"); err == nil {
		t.Fatal("scope accepted inventory state")
	}
}
