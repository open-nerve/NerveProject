package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// keptDrafts records the payload each save of a draft gives its use case
// and answers a draft holding it.
type keptDrafts struct{ got []json.RawMessage }

func (k *keptDrafts) keep(payload json.RawMessage) (domain.Draft, error) {
	k.got = append(k.got, payload)
	return domain.Draft{ID: webSeven.ID, WorkspaceID: acmeID, CreatedByID: aliceID, ProjectID: &webID, Payload: payload, CreatedAt: created, UpdatedAt: created}, nil
}

// keptDraftCreate is createDraft, keptDraftUpdate updateDraft, of k.
type keptDraftCreate struct{ k *keptDrafts }
type keptDraftUpdate struct{ k *keptDrafts }

func (c keptDraftCreate) Execute(_ context.Context, _ string, in domain.NewDraft) (domain.Draft, error) {
	return c.k.keep(in.Payload)
}

func (u keptDraftUpdate) Execute(_ context.Context, _ uuid.UUID, in domain.DraftPatch) (domain.Draft, error) {
	return u.k.keep(in.Payload)
}

// A draft's payload carries the cycle and the modules its work item is to
// be created in, as a creation's body does (M6 P3a spec 2.3, M6 design
// 3.3): saved, and replaced, as given, the modules in their order and one
// of them twice, which only its publication checks, and answered so.
func TestDraftPayloadTakesTheCycleAndTheModules(t *testing.T) {
	payload := `{"name":"Fix","cycle_id":` + sprintJSON + `,"module_ids":["` + moduleIDs[1].String() + `","` + moduleIDs[0].String() + `","` +
		moduleIDs[1].String() + `"]}`
	var want map[string]any
	if err := json.Unmarshal([]byte(payload), &want); err != nil {
		t.Fatal(err)
	}
	kept := &keptDrafts{}
	h := newServer(t, httpadapter.UseCases{CreateDraft: keptDraftCreate{kept}, UpdateDraft: keptDraftUpdate{kept}})
	for _, req := range []*http.Request{
		request(http.MethodPost, "/api/v0/workspaces/acme/drafts", "alice", `{"payload":`+payload+`}`),
		request(http.MethodPatch, "/api/v0/drafts/"+webSeven.ID.String(), "alice", `{"payload":`+payload+`}`),
	} {
		res, got := do(t, h, req)
		var answer struct{ Payload map[string]any }
		if err := json.Unmarshal([]byte(got), &answer); err != nil || res.StatusCode >= 300 || !reflect.DeepEqual(answer.Payload, want) {
			t.Errorf("%s %s = %d %s; want the payload %s", req.Method, req.URL.Path, res.StatusCode, got, payload)
		}
	}
	if len(kept.got) != 2 {
		t.Fatalf("the use cases got %d payloads, want 2", len(kept.got))
	}
	for i, raw := range kept.got {
		var stored map[string]any
		if err := json.Unmarshal(raw, &stored); err != nil || !reflect.DeepEqual(stored, want) {
			t.Errorf("save %d gave its use case %s, want %s", i, raw, payload)
		}
	}
}

// Each refusal of the cycle and of the modules, answered to a draft's
// publication as to a creation (M6 P3a spec 2.3): planning's codes, the
// work item's details.
func TestPublishDraftPlanningRefusals(t *testing.T) {
	for _, tt := range slices.Concat(planningRefusals, moduleRefusals) {
		t.Run(tt.name, func(t *testing.T) {
			h := newServer(t, httpadapter.UseCases{PublishDraft: &fakeGet{err: tt.err}})
			path := "/api/v0/drafts/" + webSeven.ID.String() + "/publish"
			if res, got := do(t, h, request(http.MethodPost, path, "alice", "")); res.StatusCode != http.StatusConflict || got != tt.want+"\n" {
				t.Errorf("POST %s = %d %s, want 409 %s", path, res.StatusCode, got, tt.want)
			}
		})
	}
}
