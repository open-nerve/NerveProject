package bootstrap

import (
	"net/http"
	"testing"
)

// A broken escape must not silently remove a filter or reset a cursor. These
// requests go over HTTP through the real binder, authentication and list use cases.
func TestIssueListRejectsMalformedQueryEncoding(t *testing.T) {
	w := newIssueWorld(t)
	high := w.newWorkItemIn(t, w.web, `{"name":"High","priority":"high","start_date":"2026-10-12"}`)
	w.newWorkItemIn(t, w.web, `{"name":"Low","priority":"low","start_date":"2026-10-20"}`)
	for _, route := range []struct{ name, path, prefix string }{
		{"project", "/api/v0/projects/" + w.web.String() + "/issues", ""},
		{"member", "/api/v0/workspaces/acme/members/" + w.ids["alice"].String() + "/issues", "relation=created&"},
	} {
		t.Run(route.name, func(t *testing.T) {
			// A correctly encoded filter still narrows the real stored rows.
			page := checkListEnvelope(t, w.send(t, "alice", http.MethodGet, route.path+"?"+route.prefix+"priority=high&start_date=2026-10-01..&start_date=..2026-10-15", "", 200), false, false)
			if page.Total != 1 || len(page.Data) != 1 || page.Data[0].ID != high {
				t.Fatal(page)
			}
			for _, query := range []string{
				"priority=%ZZ",
				"priority=high;low",
				"cursor=%",
				"cursor=%2",
				"limit=%ZZ",
				"%ZZ=high",
				"unknown=%ZZ",
				"priority=high&start_date=2026-10-01..&start_date=%ZZ",
				"priority=high&target_date=2026-10-01;other=ignored",
			} {
				t.Run(query, func(t *testing.T) {
					req := newRequest(t, http.MethodGet, w.base+route.path+"?"+route.prefix+query, w.tokens["alice"], nil)
					raw := req.URL.RawQuery
					res, body := send(t, req)
					w.contract.CheckResponse(t, req, res)
					if res.StatusCode != 400 || problemCode(t, body) != "bad_request" || req.URL.RawQuery != raw {
						t.Fatalf("status=%d body=%s query=%q want unchanged %q", res.StatusCode, body, req.URL.RawQuery, raw)
					}
				})
			}
			// Escaped punctuation is decoded successfully, then rejected by the
			// existing vocabulary rules; it is not an encoding error.
			for _, query := range []string{"priority=high%3Blow", "priority=%25ZZ"} {
				body := w.send(t, "alice", http.MethodGet, route.path+"?"+route.prefix+query, "", 422)
				if problemCode(t, []byte(body)) != "validation_failed" || oneError(t, []byte(body)) != "priority not_allowed" {
					t.Fatal(body)
				}
			}
			// The new raw-query check runs inside the strict list middleware,
			// after the existing authentication step for successfully bound input.
			req := newRequest(t, http.MethodGet, w.base+route.path+"?"+route.prefix+"priority=%ZZ", "", nil)
			res, body := send(t, req)
			w.contract.CheckResponse(t, req, res)
			if res.StatusCode != 401 || problemCode(t, body) != "unauthorized" {
				t.Fatal(res.StatusCode, string(body))
			}
		})
	}
}
