package httpadapter_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
)

// invalidQuery is the 422 of a query whose parameters are errors, as the
// drafts' list answers it.
func invalidQuery(errors string) string {
	return `{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.","errors":[` +
		errors + `]}`
}

// unknown and repeated are the errors of a parameter the operation does
// not take, and of one it takes given more than once.
func unknown(name string) string {
	return `{"field":"` + name + `","code":"not_allowed","message":"is not allowed for this operation"}`
}

func repeated(name string) string {
	return `{"field":"` + name + `","code":"invalid_format","message":"must be given once"}`
}

// undecodable is the answer to a query that does not decode.
const undecodable = `{"status":400,"code":"bad_request","title":"Bad Request","detail":"The request query could not be decoded."}`

// Each workspace operation takes its own query parameters alone, each once
// (P1c spec 2.3, 2.4; the drafts' list's practice): the list its filters,
// limit and cursor; the unread counts and mark-all-read none. Another
// parameter is 422 not_allowed on it, one of its own given more than once
// 422 invalid_format, every such parameter at once in name order, also
// where the list's binding would otherwise have answered 400 first; a
// query that does not decode is 400. None of them comes to a use case. A
// list parameter that does not bind, given once and with no such error, is
// still the binding's 400 on it.
func TestEachWorkspaceOperationTakesItsOwnQueryOnce(t *testing.T) {
	for _, tt := range []struct {
		name, method, target, body string
		status                     int
		want                       string
	}{
		{"list, another", http.MethodGet, listPath + "?tab=all&page=2", "", http.StatusUnprocessableEntity, invalidQuery(unknown("page"))},
		{"list, Plane's mentioned", http.MethodGet, listPath + "?mentioned=true", "", http.StatusUnprocessableEntity, invalidQuery(unknown("mentioned"))},
		{"list, a filter twice", http.MethodGet, listPath + "?tab=all&tab=mentions", "", http.StatusUnprocessableEntity, invalidQuery(repeated("tab"))},
		{"list, type twice", http.MethodGet, listPath + "?type=assigned&type=created", "", http.StatusUnprocessableEntity, invalidQuery(repeated("type"))},
		{"list, read twice alike", http.MethodGet, listPath + "?read=true&read=true", "", http.StatusUnprocessableEntity, invalidQuery(repeated("read"))},
		{"list, limit twice", http.MethodGet, listPath + "?limit=1&limit=2", "", http.StatusUnprocessableEntity, invalidQuery(repeated("limit"))},
		{"list, cursor twice", http.MethodGet, listPath + "?cursor=a&cursor=b", "", http.StatusUnprocessableEntity, invalidQuery(repeated("cursor"))},
		{"list, several", http.MethodGet, listPath + "?snoozed=true&per_page=5&archived=true&snoozed=false", "", http.StatusUnprocessableEntity,
			invalidQuery(unknown("per_page") + "," + repeated("snoozed"))},
		{"list, a limit that does not bind and another", http.MethodGet, listPath + "?limit=ten&page=2", "", http.StatusUnprocessableEntity,
			invalidQuery(unknown("page"))},
		{"list, undecodable", http.MethodGet, listPath + "?tab=all&x=%zz", "", http.StatusBadRequest, undecodable},
		{"count, any", http.MethodGet, countPath + "?tab=all", "", http.StatusUnprocessableEntity, invalidQuery(unknown("tab"))},
		{"count, another twice", http.MethodGet, countPath + "?all=1&all=2", "", http.StatusUnprocessableEntity, invalidQuery(unknown("all"))},
		{"count, undecodable", http.MethodGet, countPath + "?%zz", "", http.StatusBadRequest, undecodable},
		{"mark, any", http.MethodPost, markPath + "?tab=all", `{}`, http.StatusUnprocessableEntity, invalidQuery(unknown("tab"))},
		{"mark, undecodable", http.MethodPost, markPath + "?a=%zz", `{}`, http.StatusBadRequest, undecodable},
	} {
		f := newFakes()
		res, body := do(t, newServer(t, f.useCases()), request(tt.method, tt.target, "alice", tt.body))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: %s %s = %d %s, want %d %s", tt.name, tt.method, tt.target, res.StatusCode, body, tt.status, tt.want)
		}
		if f.asked() != 0 {
			t.Errorf("%s: the use cases got %d calls, want none", tt.name, f.asked())
		}
	}

	f := newFakes()
	res, body := do(t, newServer(t, f.useCases()), request(http.MethodGet, listPath+"?limit=ten", "alice", ""))
	var problem struct {
		Code   string
		Errors []struct{ Field string }
	}
	if err := json.Unmarshal([]byte(body), &problem); err != nil || res.StatusCode != http.StatusBadRequest || problem.Code != "bad_request" ||
		len(problem.Errors) != 1 || problem.Errors[0].Field != "limit" || f.asked() != 0 {
		t.Errorf("GET ?limit=ten = %d %s with %d calls, want 400 bad_request on limit and none", res.StatusCode, body, f.asked())
	}
}

// A change takes no query of its own and reads none: a parameter beside
// it changes nothing.
func TestAChangeIgnoresItsQuery(t *testing.T) {
	f := newFakes()
	f.update.answer = bareView()
	res, body := do(t, newServer(t, f.useCases()), request(http.MethodPatch, updatePath+"?read=true&read=false", "alice", `{}`))
	if res.StatusCode != http.StatusOK || body != bareJSON+"\n" || len(f.update.calls) != 1 || !samePatch(f.update.calls[0].Patch, domain.Patch{}) {
		t.Errorf("PATCH ?read=true&read=false {} = %d %s, the use case got %+v; want 200 and no change", res.StatusCode, body, f.update.calls)
	}
}
