package bootstrap

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"uuid"
)

// Inspect every nested Issue, including relation wrappers and group cells;
// a missing required field must not silently decode as the valid zero count.
func checkAttachmentProjection(t *testing.T, raw string, want map[uuid.UUID]int64) map[uuid.UUID]int {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		t.Fatal(err)
	}
	found := map[uuid.UUID]int{}
	var visit func(any)
	visit = func(value any) {
		switch v := value.(type) {
		case []any:
			for _, x := range v {
				visit(x)
			}
		case map[string]any:
			if _, ok := v["sequence_id"]; ok {
				id, err := uuid.Parse(fmt.Sprint(v["id"]))
				if err != nil {
					t.Fatal(err)
				}
				expected, ok := want[id]
				if !ok {
					t.Fatalf("unexpected issue %s in response", id)
				}
				number, ok := v["attachment_count"].(json.Number)
				count, parse := number.Int64()
				if !ok || parse != nil || count != expected {
					t.Errorf("issue %s attachment_count=%v want=%d", id, v["attachment_count"], expected)
				}
				found[id]++
			}
			for _, x := range v {
				visit(x)
			}
		}
	}
	visit(value)
	if len(found) == 0 {
		t.Fatalf("response contains no Issue fixture: %s", raw)
	}
	return found
}

func TestIssueAttachmentProjection(t *testing.T) {
	w := newIssueWorld(t)
	raw := w.send(t, "bob", "POST", "/api/v0/projects/"+w.web.String()+"/issues", fmt.Sprintf(`{"name":"Projection root","assignee_ids":[%q,%q]}`, w.ids["alice"], w.ids["bob"]), 201)
	var root struct {
		ID       uuid.UUID `json:"id"`
		Sequence int64     `json:"sequence_id"`
	}
	decodeAnswer(t, raw, &root)
	checkAttachmentProjection(t, raw, map[uuid.UUID]int64{root.ID: 0})
	child := w.newWorkItemBy(t, "bob", w.web, `{"name":"Projection child"}`)
	other := w.newWorkItemBy(t, "bob", w.web, `{"name":"Projection zero"}`)
	want := map[uuid.UUID]int64{root.ID: 2, child: 1, other: 0}
	for id, count := range want {
		for range count {
			uploadedAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", attachmentScope("issue", id), "same.txt", "text/plain", []byte("abc"))
		}
	}
	// Completed but private description preparations never enter attachment counts.
	uploadedAssetHTTP(t, w, "bob", "ISSUE_DESCRIPTION", attachmentScope("issue", root.ID), "body.png", "image/png", assetTestPNG(t))
	base := "/api/v0/issues/" + root.ID.String()
	for _, q := range []struct {
		method, path, body string
		status             int
	}{
		{"GET", base, "", 200},
		{"GET", fmt.Sprintf("/api/v0/workspaces/acme/issues/WEB-%d", root.Sequence), "", 200},
		{"PATCH", base, `{}`, 200},
		{"PATCH", base, fmt.Sprintf(`{"state_id":%q}`, w.done), 200},
		{"POST", base + "/archive", "", 200},
		{"POST", base + "/unarchive", "", 200},
		{"POST", base + "/sub-issues", fmt.Sprintf(`{"sub_issue_ids":[%q]}`, child), 200},
		{"GET", base + "/sub-issues", "", 200},
		{"POST", base + "/relations", fmt.Sprintf(`{"relation_type":"relates_to","issue_ids":[%q]}`, other), 201},
		{"GET", base + "/relations", "", 200},
		{"GET", "/api/v0/workspaces/acme/issue-search?q=Projection", "", 200},
		{"GET", "/api/v0/projects/" + w.web.String() + "/issues?sub_issue=true", "", 200},
		{"GET", "/api/v0/projects/" + w.web.String() + "/issues?group_by=assignee_id", "", 200},
		{"GET", "/api/v0/projects/" + w.web.String() + "/issues?group_by=assignee_id&sub_group_by=label_id", "", 200},
		{"GET", "/api/v0/workspaces/acme/members/" + w.ids["bob"].String() + "/issues?relation=created", "", 200},
	} {
		t.Run(q.method+q.path, func(t *testing.T) {
			checkAttachmentProjection(t, w.send(t, "bob", q.method, q.path, q.body, q.status), want)
		})
	}
	d := createDraftHTTP(t, w, "bob", fmt.Sprintf(`{"project_id":%q,"payload":{"name":"Published projection"}}`, w.web))
	published := w.send(t, "bob", "POST", publishPath(d.ID), "", 201)
	var row struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, published, &row)
	checkAttachmentProjection(t, published, map[uuid.UUID]int64{row.ID: 0})
}
