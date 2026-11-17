package bootstrap

import (
	"net/http"
	"strconv"
	"testing"
	"uuid"
)

// subscribedIn is the is_subscribed of the answer raw, a work item in full,
// which must carry it.
func subscribedIn(t *testing.T, raw string) bool {
	t.Helper()
	var answer struct {
		IsSubscribed *bool `json:"is_subscribed"`
	}
	decodeAnswer(t, raw, &answer)
	if answer.IsSubscribed == nil {
		t.Fatalf("the answer %s has no is_subscribed", raw)
	}
	return *answer.IsSubscribed
}

// Each of the seven answers of a work item in full says whether its caller
// subscribes to it (M7 P1a spec 2.3, R5). alice's creation of Fix, in Done,
// and her draft's publication answer true: a creation subscribes its
// creator, and its answer is read back in its transaction. bob, Web's
// member, not subscribed to Fix, is answered false by its read, its read by
// its identifier, his change of its name, its archiving and its
// unarchiving, none of which subscribes him; once he subscribes, true by
// each; once he unsubscribes, false by its read. alice, who created it, is
// answered true by her read throughout.
func TestEachDetailAnswerSaysWhetherTheCallerSubscribes(t *testing.T) {
	w := newIssueWorld(t)
	raw := w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues",
		`{"name":"Fix","state_id":"`+w.done.String()+`"}`, http.StatusCreated)
	var fix struct {
		ID         uuid.UUID `json:"id"`
		SequenceID int       `json:"sequence_id"`
	}
	decodeAnswer(t, raw, &fix)
	if !subscribedIn(t, raw) {
		t.Errorf("alice's creation of Fix answers is_subscribed false, want true: %s", raw)
	}
	d := w.draftIn(t, `{"name":"Drafted"}`)
	if published := draftRequest(t, w, "alice", http.MethodPost, publishPath(d.ID), "", http.StatusCreated); !subscribedIn(t, published) {
		t.Errorf("alice's publication of her draft answers is_subscribed false, want true: %s", published)
	}

	path := "/api/v0/issues/" + fix.ID.String()
	answers := func(round string, want bool) {
		t.Helper()
		for _, step := range []struct{ name, method, path, body string }{
			{"read", http.MethodGet, path, ""},
			{"read by its identifier", http.MethodGet, "/api/v0/workspaces/acme/issues/WEB-" + strconv.Itoa(fix.SequenceID), ""},
			{"changed", http.MethodPatch, path, `{"name":"Fix ` + round + `"}`},
			{"archived", http.MethodPost, path + "/archive", ""},
			{"unarchived", http.MethodPost, path + "/unarchive", ""},
		} {
			if got := subscribedIn(t, w.send(t, "bob", step.method, step.path, step.body, http.StatusOK)); got != want {
				t.Errorf("%s, %s by bob: is_subscribed %v, want %v", round, step.name, got, want)
			}
		}
		if !subscribedIn(t, w.send(t, "alice", http.MethodGet, path, "", http.StatusOK)) {
			t.Errorf("%s, read by alice: is_subscribed false, want true", round)
		}
	}
	answers("not subscribed", false)
	w.send(t, "bob", http.MethodPost, path+"/subscribe", "", http.StatusNoContent)
	answers("subscribed", true)
	w.send(t, "bob", http.MethodPost, path+"/unsubscribe", "", http.StatusNoContent)
	if subscribedIn(t, w.send(t, "bob", http.MethodGet, path, "", http.StatusOK)) {
		t.Error("unsubscribed, read by bob: is_subscribed true, want false")
	}
}
