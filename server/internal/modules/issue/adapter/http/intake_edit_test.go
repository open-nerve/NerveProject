package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeIntakeUpdate is updateIntakeIssue: each call is recorded as "caller
// work item", with the edit it got; it answers answer, or err.
type fakeIntakeUpdate struct {
	calls  []string
	got    []domain.IntakeIssueUpdate
	answer domain.Issue
	err    error
}

func (f *fakeIntakeUpdate) Execute(ctx context.Context, id uuid.UUID, edit domain.IntakeIssueUpdate) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String())
	f.got = append(f.got, edit)
	return f.answer, f.err
}

// intakeItemPath is WEB-8's, under /intake-issues.
var intakeItemPath = "/api/v0/intake-issues/" + webEight.ID.String()

// PATCH goes to the use case for the caller and the path's work item, with
// the body's fields, each nil when not given, a list given empty given,
// the target date given null given as none; the answer is 200 with the
// work item the use case answers, its intake row with it.
func TestUpdateIntakeIssue(t *testing.T) {
	pending, pendingJSON := inIntake(domain.IntakeFacts{Status: domain.IntakePending}, `{"duplicate_to_id":null,"snoozed_till":null,"status":"pending"}`)
	html, skip, name, high, none := "<p>Rich</p>", true, " Mend ", domain.PriorityHigh, []uuid.UUID{}
	for _, tt := range []struct {
		body string
		want domain.IntakeIssueUpdate
	}{
		{`{"name":" Mend ","priority":"high","assignee_ids":["0199a2b4-0000-7000-8000-000000000002"],"label_ids":["0199a2b4-0000-7000-8000-0000000000d1"],` +
			`"target_date":"2026-10-19"}`,
			domain.IntakeIssueUpdate{Name: &name, Priority: &high, AssigneeIDs: &[]uuid.UUID{bobID}, LabelIDs: &[]uuid.UUID{bugID}, SetTargetDate: true,
				TargetDate: &targetDay}},
		{`{}`, domain.IntakeIssueUpdate{}},
		{`{"description_html":"<p>Rich</p>","skip_activity":true}`, domain.IntakeIssueUpdate{DescriptionHTML: &html, SkipActivity: &skip}},
		{`{"assignee_ids":[],"label_ids":[]}`, domain.IntakeIssueUpdate{AssigneeIDs: &none, LabelIDs: &none}},
		{`{"target_date":null}`, domain.IntakeIssueUpdate{SetTargetDate: true}},
	} {
		update := &fakeIntakeUpdate{answer: pending}
		res, body := do(t, newServer(t, httpadapter.UseCases{UpdateIntakeIssue: update}), request(http.MethodPatch, intakeItemPath, "alice", tt.body))
		if res.StatusCode != http.StatusOK || body != pendingJSON+"\n" {
			t.Errorf("PATCH %s = %d %s, want 200 %s", tt.body, res.StatusCode, body, pendingJSON)
		}
		if want := []string{aliceID.String() + " " + webEight.ID.String()}; !slices.Equal(update.calls, want) ||
			!reflect.DeepEqual(update.got, []domain.IntakeIssueUpdate{tt.want}) {
			t.Errorf("PATCH %s: calls %q with %+v; want %q with %+v", tt.body, update.calls, update.got, want, tt.want)
		}
	}
}

// A body with a field the content of an intake item does not have (its
// state, start date, parent, cycle, modules, sort order or status), null
// for a field that is not nullable, a value of the wrong type, an id that
// is no id, or a day that is no day: refused as bad_request before the
// use case.
func TestUpdateIntakeIssueHoldsTheBodyToItsStructure(t *testing.T) {
	update := &fakeIntakeUpdate{}
	h := newServer(t, httpadapter.UseCases{UpdateIntakeIssue: update})
	for _, body := range []string{`{"state_id":"0199a2b4-0000-7000-8000-0000000000c1"}`, `{"start_date":"2026-10-12"}`,
		`{"parent_id":"0199a2b4-0000-7000-8000-0000000000e6"}`, `{"cycle_id":null}`, `{"module_ids":[]}`, `{"sort_order":0}`, `{"status":"accepted"}`,
		`{"name":null}`, `{"description_html":null}`, `{"priority":null}`, `{"assignee_ids":null}`, `{"skip_activity":"yes"}`, `{"name":1}`,
		`{"assignee_ids":"bob"}`, `{"label_ids":["Bug"]}`, `{"target_date":"2026-02-30"}`, `{"target_date":"2026-10-19T00:00:00Z"}`} {
		res, got := do(t, h, request(http.MethodPatch, intakeItemPath, "alice", body))
		var problem struct{ Code string }
		if err := json.Unmarshal([]byte(got), &problem); err != nil || res.StatusCode != http.StatusBadRequest || problem.Code != "bad_request" {
			t.Errorf("PATCH %s = %d %s, want 400 bad_request", body, res.StatusCode, got)
		}
	}
	if len(update.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", update.calls)
	}
}

// intakeNotFound and intakeClosed, as the domain answers them.
const (
	intakeNotFound = `{"status":404,"code":"issue.intake_not_found","title":"Not Found","detail":"The intake work item does not exist, or you cannot see it."}`
	intakeClosed   = `{"status":409,"code":"issue.intake_closed","title":"Conflict",` +
		`"detail":"The intake work item has been accepted, declined or marked a duplicate; it can no longer be edited in the intake."}`
)

// The use case's refusals, as the contract declares them, and its failure.
func TestUpdateIntakeIssueRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"a label refused", shared.Invalid(domain.UnknownLabel(0)), http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"label_ids[0]","code":"not_allowed","message":"must be a label of the project"}]}`},
		{"no intake work item", domain.ErrIntakeNotFound, http.StatusNotFound, intakeNotFound},
		{"not its submitter", shared.Forbidden(), http.StatusForbidden, forbidden},
		{"an archived project", domain.ErrProjectArchived, http.StatusConflict, archivedProject},
		{"its intake off", domain.ErrIntakeDisabled, http.StatusConflict, intakeOff},
		{"a triaged one", domain.ErrIntakeClosed, http.StatusConflict, intakeClosed},
		{"a busy project", shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
		{"an image not the work item's", shared.NewError(shared.KindNotFound, "asset.not_found", "The asset does not exist, or you cannot see it."),
			http.StatusNotFound, `{"status":404,"code":"asset.not_found","title":"Not Found","detail":"The asset does not exist, or you cannot see it."}`},
		{"an image not uploaded", shared.NewError(shared.KindConflict, "asset.not_uploaded", "The asset's content has not been completed."),
			http.StatusConflict, `{"status":409,"code":"asset.not_uploaded","title":"Conflict","detail":"The asset's content has not been completed."}`},
		{"a failure", errGone, http.StatusInternalServerError, failed},
	} {
		h := newServer(t, httpadapter.UseCases{UpdateIntakeIssue: &fakeIntakeUpdate{err: tt.err}})
		res, body := do(t, h, request(http.MethodPatch, intakeItemPath, "alice", `{"name":"Mend"}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
			t.Errorf("%s: PATCH = %d %s, Retry-After %q; want %d %s", tt.name, res.StatusCode, body, res.Header.Get("Retry-After"), tt.status, tt.want)
		}
	}
}

// DELETE goes to the use case for the caller and the path's work item; the
// answer is 204 without a body. Its refusals, as the contract declares
// them, and its failure.
func TestDeleteIntakeIssue(t *testing.T) {
	remove := &fakeDelete{}
	res, body := do(t, newServer(t, httpadapter.UseCases{DeleteIntakeIssue: remove}), request(http.MethodDelete, intakeItemPath, "alice", ""))
	if want := []string{aliceID.String() + " " + webEight.ID.String()}; res.StatusCode != http.StatusNoContent || body != "" || !slices.Equal(remove.calls, want) {
		t.Errorf("DELETE = %d %q, calls %q; want 204 and %q", res.StatusCode, body, remove.calls, want)
	}
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"no intake work item", domain.ErrIntakeNotFound, http.StatusNotFound, intakeNotFound},
		{"not its submitter", shared.Forbidden(), http.StatusForbidden, forbidden},
		{"an archived project", domain.ErrProjectArchived, http.StatusConflict, archivedProject},
		{"its intake off", domain.ErrIntakeDisabled, http.StatusConflict, intakeOff},
		{"a busy project", shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
		{"a failure", errGone, http.StatusInternalServerError, failed},
	} {
		h := newServer(t, httpadapter.UseCases{DeleteIntakeIssue: &fakeDelete{err: tt.err}})
		res, body := do(t, h, request(http.MethodDelete, intakeItemPath, "alice", ""))
		if res.StatusCode != tt.status || body != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
			t.Errorf("%s: DELETE = %d %s, Retry-After %q; want %d %s", tt.name, res.StatusCode, body, res.Header.Get("Retry-After"), tt.status, tt.want)
		}
	}
}
