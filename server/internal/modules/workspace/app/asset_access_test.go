package app_test

import (
	"errors"
	"fmt"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestAssetHostAccessMatrix(t *testing.T) {
	uid, wid := uuid.New(), uuid.New()
	for _, intent := range []app.AssetIntent{app.AssetRead, app.AssetPrepare, app.AssetRemove} {
		for _, refusal := range []error{nil, shared.Forbidden(), shared.ErrNotVisible, errors.New("database unavailable")} {
			log := &callLog{}
			auth := &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{{uid, wid}: {}}, errs: map[grantKey]error{{uid, wid}: refusal}}
			a := app.NewAssetAccess(auth)
			q := app.AssetRequest{ID: wid, Intent: intent}
			actor := shared.Actor{UserID: uid}
			h, err := a.Inspect(t.Context(), actor, q)
			want := refusal
			if errors.Is(refusal, shared.ErrNotVisible) {
				want = domain.ErrNotFound
			}
			if !errors.Is(err, want) {
				t.Fatalf("intent=%d got %v want %v", intent, err, want)
			}
			if err == nil {
				if h.WorkspaceID != wid {
					t.Fatal("wrong host")
				}
				if err := a.Check(t.Context(), actor, q, h); err != nil {
					t.Fatal(err)
				}
			}
			action := "workspace.update"
			if intent == app.AssetRead {
				action = "workspace.read"
			}
			wantCall := fmt.Sprintf("Authorize %s %s on %s/%s outside tx", uid, action, wid, uuid.Nil())
			for _, call := range log.calls {
				if call != wantCall {
					t.Fatalf("wrong owner decision: %s", call)
				}
			}
			if len(log.calls) == 0 {
				t.Fatal("owner authorization bypassed")
			}
		}
	}
}
