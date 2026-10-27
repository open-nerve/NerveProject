package bootstrap

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

const draftsPath = "/api/v0/workspaces/acme/drafts"

type draftHTTP struct {
	ID          uuid.UUID      `json:"id"`
	WorkspaceID uuid.UUID      `json:"workspace_id"`
	ProjectID   *uuid.UUID     `json:"project_id"`
	Payload     map[string]any `json:"payload"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

func draftPath(id uuid.UUID) string { return "/api/v0/drafts/" + id.String() }

// Invalid bodies and raw queries deliberately bypass request-contract checking.
func draftRequest(t *testing.T, w issueWorld, who, method, path, body string, want int) string {
	t.Helper()
	req := newRequest(t, method, w.base+path, w.tokens[who], []byte(body))
	res, raw := send(t, req)
	if res.StatusCode != want {
		t.Fatalf("%s %s as %s = %d %s, want %d", method, path, who, res.StatusCode, raw, want)
	}
	w.contract.CheckResponse(t, req, res)
	return string(raw)
}
func createDraftHTTP(t *testing.T, w issueWorld, who, body string) draftHTTP {
	t.Helper()
	raw := draftRequest(t, w, who, "POST", draftsPath, body, 201)
	listKeys(t, []byte(raw), "id", "workspace_id", "project_id", "payload", "created_at", "updated_at")
	var d draftHTTP
	decodeAnswer(t, raw, &d)
	return d
}
func draftState(t *testing.T, w issueWorld) map[string]string {
	t.Helper()
	out := collaborationState(t, w)
	var rows string
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY id),'[]'::jsonb)::text FROM draft_issues r").Scan(&rows); err != nil {
		t.Fatal(err)
	}
	out["draft_issues"] = rows
	return out
}

// Catches structural widening, create validation on save, merged payloads,
// lost project tri-state, audit churn, and malformed/repeated raw query loss.
func TestDraftHTTPShapes(t *testing.T) {
	w := newIssueWorld(t)
	initialIssues := collaborationState(t, w)
	for _, body := range []string{`{"payload":{}}`, `{"payload":{"name":""}}`} {
		createDraftHTTP(t, w, "alice", body)
	}
	for _, body := range []string{
		`{}`, `{"payload":null}`, `{"payload":[]}`, `{"payload":{"unknown":1}}`, `{"payload":{"project_id":null}}`,
		`{"payload":{"skip_activity":true}}`, `{"payload":{"sort_order":1}}`, `{"payload":{"name":null}}`,
		`{"payload":{"description_html":null}}`, `{"payload":{"priority":null}}`, `{"payload":{"state_id":"bad"}}`,
		`{"payload":{"parent_id":null}}`, `{"payload":{"label_ids":[null]}}`, `{"payload":{"assignee_ids":["bad"]}}`,
		`{"payload":{"start_date":"2026-02-30"}}`, `{"payload":{"target_date":null}}`, `{"payload":{"name":3}}`, `{"payload":{"state_id":null}}`, `{"payload":{"assignee_ids":null}}`, `{"payload":{"label_ids":null}}`, `{"payload":{"start_date":null}}`,
		`{"payload":{},"payload":{}}`, `{"payload":{"name":"a","name":"b"}}`, `{"project_id":"bad","payload":{}}`,
		`{"extra":1,"payload":{}}`,
	} {
		t.Run("shape/"+body, func(t *testing.T) { draftRequest(t, w, "alice", "POST", draftsPath, body, 400) })
	}
	bad := fmt.Sprintf(`{"name":"%s","description_html":"<script>raw</script>","priority":"arbitrary","state_id":"%s","parent_id":"%s","label_ids":%s,"assignee_ids":[],"start_date":"2030-12-12","target_date":"2000-01-01"}`, strings.Repeat("x", 256), uuid.NewV7(), uuid.NewV7(), idArray(w.bug, w.bug))
	d := createDraftHTTP(t, w, "alice", `{"project_id":"`+w.web.String()+`","payload":`+bad+`}`)
	if len(d.Payload["name"].(string)) != 256 || d.Payload["description_html"] != "<script>raw</script>" {
		t.Fatal(d)
	}
	// More than 100 ids is a create concern, as are duplicates and unknown references.
	ids := make([]uuid.UUID, 101)
	for i := range ids {
		ids[i] = w.bug
	}
	createDraftHTTP(t, w, "alice", `{"payload":{"label_ids":`+idArray(ids...)+`}}`)
	before := draftState(t, w)
	for _, field := range []string{"name", "description_html", "priority"} {
		body := `{"payload":{"` + field + `":"\u0000"}}`
		for _, method := range []string{"POST", "PATCH"} {
			path := draftsPath
			if method == "PATCH" {
				path = draftPath(d.ID)
			}
			raw := draftRequest(t, w, "alice", method, path, body, 422)
			if oneError(t, []byte(raw)) != "payload."+field+" invalid_format" {
				t.Fatal(raw)
			}
		}
	}
	if !reflect.DeepEqual(before, draftState(t, w)) {
		t.Fatal("NUL refusal changed persisted rows")
	}
	for _, body := range []string{`{}`, `{"payload":` + bad + `}`} {
		var got draftHTTP
		decodeAnswer(t, draftRequest(t, w, "alice", "PATCH", draftPath(d.ID), body, 200), &got)
		if !got.UpdatedAt.Equal(d.UpdatedAt) || got.ProjectID == nil || *got.ProjectID != w.web {
			t.Fatal("no-op/project omission", got, d)
		}
	}
	var empty draftHTTP
	decodeAnswer(t, draftRequest(t, w, "alice", "PATCH", draftPath(d.ID), `{"payload":{}}`, 200), &empty)
	if len(empty.Payload) != 0 || empty.ProjectID == nil || *empty.ProjectID != w.web {
		t.Fatal("payload must replace", empty)
	}
	var cleared draftHTTP
	decodeAnswer(t, draftRequest(t, w, "alice", "PATCH", draftPath(d.ID), `{"project_id":null}`, 200), &cleared)
	if cleared.ProjectID != nil || len(cleared.Payload) != 0 {
		t.Fatal(cleared)
	}
	var reassigned draftHTTP
	decodeAnswer(t, draftRequest(t, w, "alice", "PATCH", draftPath(d.ID), `{"project_id":"`+w.web.String()+`"}`, 200), &reassigned)
	if reassigned.ProjectID == nil || *reassigned.ProjectID != w.web || len(reassigned.Payload) != 0 {
		t.Fatal("explicit project/payload omission", reassigned)
	}
	draftRequest(t, w, "alice", "PATCH", draftPath(d.ID), `{"payload":null}`, 400)
	for _, tc := range []struct {
		query string
		want  int
	}{
		{"", 200}, {"?limit=1", 200}, {"?limit=100", 200}, {"?limit=0", 422}, {"?limit=101", 422}, {"?limit=x", 400},
		{"?unknown=x", 422}, {"?limit=1&limit=2", 422}, {"?cursor=x&cursor=y", 422}, {"?cursor=x", 400},
		{"?limit=1&limit=2&bad=%zz", 400}, {"?cursor=a&cursor=b&bad=%zz", 400}, {"?bad=%zz", 400}, {"?bad=1;2", 400},
	} {
		t.Run("query/"+tc.query, func(t *testing.T) { draftRequest(t, w, "alice", "GET", draftsPath+tc.query, "", tc.want) })
	}
	// A real page over >50 rows proves the default and cursor-independent total.
	for range 48 {
		createDraftHTTP(t, w, "alice", `{"payload":{}}`)
	}
	var page struct {
		Data  []draftHTTP `json:"data"`
		Next  *string     `json:"next_cursor"`
		Total int         `json:"total"`
	}
	raw := draftRequest(t, w, "alice", "GET", draftsPath, "", 200)
	listKeys(t, []byte(raw), "data", "next_cursor", "total")
	decodeAnswer(t, raw, &page)
	if len(page.Data) != 50 || page.Total != 52 || page.Next == nil {
		t.Fatal(raw)
	}
	for _, row := range listArray(t, listKeys(t, []byte(raw), "data", "next_cursor", "total")["data"]) {
		listKeys(t, row, "id", "workspace_id", "project_id", "payload", "created_at", "updated_at")
	}
	if !reflect.DeepEqual(initialIssues, collaborationState(t, w)) {
		t.Fatal("saving drafts changed issue resources")
	}
	draftRequest(t, w, "alice", "DELETE", draftPath(d.ID), "", 204)
	draftRequest(t, w, "alice", "GET", draftPath(d.ID), "", 404)
}

// A valid cursor is only a sort position. Its id must never supply scope,
// owner or existence information, even when it belongs to a different actor.
func TestDraftCursorNeverReadsOtherOwner(t *testing.T) {
	w := newIssueWorld(t)
	a := createDraftHTTP(t, w, "alice", `{"payload":{"name":"A"}}`)
	b := createDraftHTTP(t, w, "bob", `{"payload":{"name":"B private"}}`)
	for _, id := range []uuid.UUID{b.ID, uuid.NewV7()} {
		c, err := domain.EncodeDraftCursor(domain.DraftCursor{CreatedAt: b.CreatedAt, ID: id})
		if err != nil {
			t.Fatal(err)
		}
		raw := draftRequest(t, w, "alice", "GET", draftsPath+"?cursor="+url.QueryEscape(c), "", 200)
		var p struct {
			Data  []draftHTTP `json:"data"`
			Total int         `json:"total"`
		}
		decodeAnswer(t, raw, &p)
		if p.Total != 1 || len(p.Data) != 1 || p.Data[0].ID != a.ID || strings.Contains(raw, "B private") {
			t.Fatal(raw)
		}
		envelope := listKeys(t, []byte(raw), "data", "next_cursor", "total")
		for _, row := range listArray(t, envelope["data"]) {
			listKeys(t, row, "id", "workspace_id", "project_id", "payload", "created_at", "updated_at")
		}
	}
}
