package bootstrap

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strconv"
	"testing"
	"time"
	"uuid"
)

// answeredIntake is a detail's intake row, as the contract's IssueIntake
// has it.
type answeredIntake struct {
	Status        string     `json:"status"`
	SnoozedTill   *time.Time `json:"snoozed_till"`
	DuplicateToID *uuid.UUID `json:"duplicate_to_id"`
}

// intakeIn is the intake of the answer raw, a work item in full, which
// must carry it, null or not: nil for null.
func intakeIn(t *testing.T, raw string) *answeredIntake {
	t.Helper()
	var answer map[string]json.RawMessage
	decodeAnswer(t, raw, &answer)
	field, ok := answer["intake"]
	if !ok {
		t.Fatalf("the answer %s has no intake", raw)
	}
	var in *answeredIntake
	decodeAnswer(t, string(field), &in)
	return in
}

// Each answer of a work item in full carries its intake row (M7 P2a spec
// 2.8): null for a work item never submitted, by alice's creation of Fix,
// in Done, her draft's publication, and bob's read, read by its identifier,
// change, archiving and unarchiving of Fix. With Web's intake turned on,
// bob's submission of Login breaks answers it pending, without a time or a
// work item it duplicates, and so do its read and its read by its
// identifier, by bob and by alice; alice's acceptance answers it accepted,
// and so do then bob's read, read by its identifier, change into Done,
// archiving and unarchiving of it.
func TestEachDetailAnswerCarriesItsIntakeRow(t *testing.T) {
	w := newIssueWorld(t)
	var fix struct {
		ID         uuid.UUID `json:"id"`
		SequenceID int       `json:"sequence_id"`
	}
	raw := w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", `{"name":"Fix","state_id":"`+w.done.String()+`"}`,
		http.StatusCreated)
	decodeAnswer(t, raw, &fix)
	if got := intakeIn(t, raw); got != nil {
		t.Errorf("alice's creation of Fix answers intake %+v, want null", got)
	}
	d := w.draftIn(t, `{"name":"Drafted"}`)
	if got := intakeIn(t, draftRequest(t, w, "alice", http.MethodPost, publishPath(d.ID), "", http.StatusCreated)); got != nil {
		t.Errorf("alice's publication of her draft answers intake %+v, want null", got)
	}
	answers := func(name string, item uuid.UUID, sequence int, want *answeredIntake) {
		t.Helper()
		path := "/api/v0/issues/" + item.String()
		for _, step := range []struct{ name, method, path, body string }{
			{"read", http.MethodGet, path, ""},
			{"read by its identifier", http.MethodGet, "/api/v0/workspaces/acme/issues/WEB-" + strconv.Itoa(sequence), ""},
			{"changed", http.MethodPatch, path, `{"name":"` + name + ` again","state_id":"` + w.done.String() + `"}`},
			{"archived", http.MethodPost, path + "/archive", ""},
			{"unarchived", http.MethodPost, path + "/unarchive", ""},
		} {
			if got := intakeIn(t, w.send(t, "bob", step.method, step.path, step.body, http.StatusOK)); !reflect.DeepEqual(got, want) {
				t.Errorf("%s, %s by bob: intake %+v, want %+v", name, step.name, got, want)
			}
		}
	}
	answers("Fix", fix.ID, fix.SequenceID, nil)

	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"intake_view":true}`, http.StatusOK)
	var login struct {
		ID         uuid.UUID `json:"id"`
		SequenceID int       `json:"sequence_id"`
	}
	raw = w.send(t, "bob", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/intake-issues", `{"name":"Login breaks"}`, http.StatusCreated)
	decodeAnswer(t, raw, &login)
	pending := &answeredIntake{Status: "pending"}
	if got := intakeIn(t, raw); !reflect.DeepEqual(got, pending) {
		t.Errorf("bob's submission of Login breaks answers intake %+v, want %+v", got, pending)
	}
	for _, name := range []string{"bob", "alice"} {
		for _, path := range []string{"/api/v0/issues/" + login.ID.String(), "/api/v0/workspaces/acme/issues/WEB-" + strconv.Itoa(login.SequenceID)} {
			if got := intakeIn(t, w.send(t, name, http.MethodGet, path, "", http.StatusOK)); !reflect.DeepEqual(got, pending) {
				t.Errorf("GET %s by %s: intake %+v, want %+v", path, name, got, pending)
			}
		}
	}
	accepted := &answeredIntake{Status: "accepted"}
	raw = w.send(t, "alice", http.MethodPost, "/api/v0/intake-issues/"+login.ID.String()+"/triage", `{"status":"accepted"}`, http.StatusOK)
	if got := intakeIn(t, raw); !reflect.DeepEqual(got, accepted) {
		t.Errorf("alice's acceptance of Login breaks answers intake %+v, want %+v", got, accepted)
	}
	answers("Login breaks", login.ID, login.SequenceID, accepted)
}
