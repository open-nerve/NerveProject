package httpadapter_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"testing"
	"time"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// sameTriage reports whether got is want: the same status and target,
// and the same time, wherever its zone.
func sameTriage(got, want domain.Triage) bool {
	switch {
	case got.Status != want.Status, (got.DuplicateToID == nil) != (want.DuplicateToID == nil), (got.SnoozedTill == nil) != (want.SnoozedTill == nil):
		return false
	case got.DuplicateToID != nil && *got.DuplicateToID != *want.DuplicateToID:
		return false
	}
	return got.SnoozedTill == nil || got.SnoozedTill.Equal(*want.SnoozedTill)
}

// POST goes to the use case for the caller and the path's work item, with
// the body's status as it came, its time and the work item it duplicates
// nil when not given; the answer is 200 with the work item the use case
// answers, its intake row as POST of a submission shows it: a snooze with
// its time, a duplicate with the work item, an acceptance with neither.
func TestTriageIntakeIssue(t *testing.T) {
	snoozed, snoozedJSON := inIntake(domain.IntakeFacts{Status: domain.IntakeSnoozed, SnoozedTill: &snoozeTill},
		`{"duplicate_to_id":null,"snoozed_till":"2026-10-20T09:00:00.123456Z","status":"snoozed"}`)
	duplicate, duplicateJSON := inIntake(domain.IntakeFacts{Status: domain.IntakeDuplicate, DuplicateToID: &webSixID},
		`{"duplicate_to_id":"0199a2b4-0000-7000-8000-0000000000e6","snoozed_till":null,"status":"duplicate"}`)
	accepted, acceptedJSON := inIntake(domain.IntakeFacts{Status: domain.IntakeAccepted}, `{"duplicate_to_id":null,"snoozed_till":null,"status":"accepted"}`)
	for _, tt := range []struct {
		body   string
		want   domain.Triage
		answer domain.Issue
		json   string
	}{
		{`{"status":"snoozed","snoozed_till":"2026-10-20T17:00:00.123456+08:00"}`, domain.Triage{Status: domain.IntakeSnoozed, SnoozedTill: &snoozeTill},
			snoozed, snoozedJSON},
		{`{"status":"duplicate","duplicate_to_id":"0199a2b4-0000-7000-8000-0000000000e6"}`, domain.Triage{Status: domain.IntakeDuplicate, DuplicateToID: &webSixID},
			duplicate, duplicateJSON},
		{`{"status":"accepted"}`, domain.Triage{Status: domain.IntakeAccepted}, accepted, acceptedJSON},
		{`{"status":"pending","snoozed_till":"2026-10-20T09:00:00.123456Z","duplicate_to_id":"0199a2b4-0000-7000-8000-0000000000e6"}`,
			domain.Triage{Status: domain.IntakePending, SnoozedTill: &snoozeTill, DuplicateToID: &webSixID}, snoozed, snoozedJSON},
	} {
		triage := &fakeTriage{answer: tt.answer}
		res, body := do(t, newServer(t, httpadapter.UseCases{TriageIntakeIssue: triage}), request(http.MethodPost, triagePath, "alice", tt.body))
		if res.StatusCode != http.StatusOK || body != tt.json+"\n" {
			t.Errorf("POST %s = %d %s, want 200 %s", tt.body, res.StatusCode, body, tt.json)
		}
		if want := []string{aliceID.String() + " " + webEight.ID.String()}; !slices.Equal(triage.calls, want) || len(triage.got) != 1 ||
			!sameTriage(triage.got[0], tt.want) {
			t.Errorf("POST %s: calls %q with %+v; want %q with %+v", tt.body, triage.calls, triage.got, want, tt.want)
		}
	}
}

// A status given that is none of the five is given: the body's shape does
// not check an enum's values, so it goes to the use case as it came, whose
// check refuses it (422 status invalid_format; the fake answers that
// check's own refusal).
func TestTriageIntakeIssueTakesAnyStatusAsGiven(t *testing.T) {
	in := domain.Triage{Status: "rejected"}
	triage := &fakeTriage{err: domain.CheckTriage(in)}
	res, body := do(t, newServer(t, httpadapter.UseCases{TriageIntakeIssue: triage}), request(http.MethodPost, triagePath, "alice", `{"status":"rejected"}`))
	want := `{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
		`"errors":[{"field":"status","code":"invalid_format","message":"must be one of pending, snoozed, accepted, declined, duplicate"}]}`
	if res.StatusCode != http.StatusUnprocessableEntity || body != want+"\n" || len(triage.got) != 1 || !sameTriage(triage.got[0], in) {
		t.Errorf("POST = %d %s, the use case got %+v; want 422 %s, the use case %+v", res.StatusCode, body, triage.got, want, in)
	}
}

// A body without its status, null for a field that is not nullable, a
// field of another type, a time that is no time or only a day, an id that
// is no id, or a field a triage does not have: refused as bad_request
// before the use case.
func TestTriageIntakeIssueHoldsTheBodyToItsStructure(t *testing.T) {
	triage := &fakeTriage{}
	h := newServer(t, httpadapter.UseCases{TriageIntakeIssue: triage})
	for _, body := range []string{`{}`, `{"snoozed_till":"2026-10-20T09:00:00Z"}`, `{"status":null}`, `{"status":1}`,
		`{"status":"snoozed","snoozed_till":null}`, `{"status":"snoozed","snoozed_till":"tomorrow"}`, `{"status":"snoozed","snoozed_till":"2026-10-20"}`,
		`{"status":"duplicate","duplicate_to_id":null}`, `{"status":"duplicate","duplicate_to_id":"WEB-6"}`,
		`{"status":"accepted","state_id":"0199a2b4-0000-7000-8000-0000000000c1"}`} {
		res, got := do(t, h, request(http.MethodPost, triagePath, "alice", body))
		var problem struct{ Code string }
		if err := json.Unmarshal([]byte(got), &problem); err != nil || res.StatusCode != http.StatusBadRequest || problem.Code != "bad_request" {
			t.Errorf("POST %s = %d %s, want 400 bad_request", body, res.StatusCode, got)
		}
	}
	if len(triage.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", triage.calls)
	}
}

// The use case's refusals, as the contract declares them, and its failure.
func TestTriageIntakeIssueRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"a snooze in the past", shared.Invalid(shared.FieldError{Field: "snoozed_till", Code: shared.FieldMustBeFuture, Message: "must be in the future"}),
			http.StatusUnprocessableEntity, `{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"snoozed_till","code":"must_be_future","message":"must be in the future"}]}`},
		{"no intake work item", domain.ErrIntakeNotFound, http.StatusNotFound,
			`{"status":404,"code":"issue.intake_not_found","title":"Not Found","detail":"The intake work item does not exist, or you cannot see it."}`},
		{"a member", shared.Forbidden(), http.StatusForbidden, forbidden},
		{"an archived project", domain.ErrProjectArchived, http.StatusConflict, archivedProject},
		{"its intake off", domain.ErrIntakeDisabled, http.StatusConflict, intakeOff},
		{"a move its status does not allow", domain.ErrIntakeTransitionNotAllowed, http.StatusConflict,
			`{"status":409,"code":"issue.intake_transition_not_allowed","title":"Conflict",` +
				`"detail":"The intake work item cannot move from its status to the one asked for."}`},
		{"a busy project", shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
		{"a failure", errGone, http.StatusInternalServerError, failed},
	} {
		h := newServer(t, httpadapter.UseCases{TriageIntakeIssue: &fakeTriage{err: tt.err}})
		res, body := do(t, h, request(http.MethodPost, triagePath, "alice", `{"status":"declined"}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
			t.Errorf("%s: POST = %d %s, Retry-After %q; want %d %s", tt.name, res.StatusCode, body, res.Header.Get("Retry-After"), tt.status, tt.want)
		}
	}
}
