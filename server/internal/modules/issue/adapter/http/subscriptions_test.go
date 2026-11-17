package httpadapter_test

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeSubscription is subscribeIssue, or unsubscribeIssue: each call is
// recorded as "caller work item"; it answers err.
type fakeSubscription struct {
	calls []string
	err   error
}

func (f *fakeSubscription) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String())
	return f.err
}

// POST subscribe goes to subscribeIssue, and POST unsubscribe to
// unsubscribeIssue, for the caller and the path's work item, with no
// body; the answer is 204, without a body, and the other use case is not
// called.
func TestSubscribeAndUnsubscribeIssue(t *testing.T) {
	for _, path := range subscriptionPaths {
		subscribe, unsubscribe := &fakeSubscription{}, &fakeSubscription{}
		h := newServer(t, httpadapter.UseCases{SubscribeIssue: subscribe, UnsubscribeIssue: unsubscribe})
		res, body := do(t, h, request(http.MethodPost, "/api/v0/issues/"+webSeven.ID.String()+path, "alice", ""))
		if res.StatusCode != http.StatusNoContent || body != "" {
			t.Errorf("POST %s = %d %q, want 204 without a body", path, res.StatusCode, body)
		}
		called, other := subscribe, unsubscribe
		if path == "/unsubscribe" {
			called, other = unsubscribe, subscribe
		}
		if want := []string{aliceID.String() + " " + webSeven.ID.String()}; !slices.Equal(called.calls, want) || len(other.calls) != 0 {
			t.Errorf("POST %s: calls %q and %q, want %q and none", path, called.calls, other.calls, want)
		}
	}
}

// The use cases' refusals, as the contract declares them, and their
// failure, each through both operations.
func TestSubscriptionRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"no work item", domain.ErrIssueNotFound, http.StatusNotFound,
			`{"status":404,"code":"issue.not_found","title":"Not Found","detail":"The work item does not exist, or you cannot see it."}`},
		{"a workspace admin, outside the project", shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"a wait for the work item's row", shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
		{"a failure", errGone, http.StatusInternalServerError,
			`{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
	} {
		for _, path := range subscriptionPaths {
			uc := &fakeSubscription{err: tt.err}
			h := newServer(t, httpadapter.UseCases{SubscribeIssue: uc, UnsubscribeIssue: uc})
			res, body := do(t, h, request(http.MethodPost, "/api/v0/issues/"+webSeven.ID.String()+path, "alice", ""))
			if res.StatusCode != tt.status || body != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
				t.Errorf("%s: POST %s = %d %s, Retry-After %q; want %d %s", tt.name, path, res.StatusCode, body, res.Header.Get("Retry-After"),
					tt.status, tt.want)
			}
		}
	}
}

// A request without a token, or naming no work item by its id, is
// answered before either use case.
func TestSubscriptionBinding(t *testing.T) {
	for _, path := range subscriptionPaths {
		uc := &fakeSubscription{}
		h := newServer(t, httpadapter.UseCases{SubscribeIssue: uc, UnsubscribeIssue: uc})
		if res, body := do(t, h, request(http.MethodPost, "/api/v0/issues/"+webSeven.ID.String()+path, "", "")); res.StatusCode != http.StatusUnauthorized {
			t.Errorf("POST %s without a token = %d %s, want 401", path, res.StatusCode, body)
		}
		if res, body := do(t, h, request(http.MethodPost, "/api/v0/issues/not-a-uuid"+path, "alice", "")); res.StatusCode != http.StatusBadRequest {
			t.Errorf("POST %s of no id = %d %s, want 400", path, res.StatusCode, body)
		}
		if len(uc.calls) != 0 {
			t.Errorf("POST %s: calls %q, want none", path, uc.calls)
		}
	}
}

// subscriptionPaths are the paths of subscribeIssue and unsubscribeIssue,
// after a work item's.
var subscriptionPaths = []string{"/subscribe", "/unsubscribe"}
