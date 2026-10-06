package bootstrap

import (
	"maps"
	"net/http"
	"testing"
	"uuid"
)

// Each write on a label, as bootstrap wires it, changes no row outside the
// rows it may write (M3 design 3.16): on memberWorld, whose acme has Web
// and Ops, each with worldLabels (seedLabelsOf), bob, Web's admin, sends
// one write after another, each answered as its row says: the creations of
// QA at the top of Web, of Icons under Feature and of Fonts under Docs
// (201); a new name for Feature (200); Bug, which has UI under it, given
// Docs as its parent (422 validation_failed, parent_id not_allowed); UI
// given no parent (200); a label named as Bug in another case (409
// project.label_name_taken); the deletion of Fonts (204); Docs, the parent
// Fonts was created under, given Bug as its parent (200); the deletions of
// Feature and of Docs (204). After each write, every row of every table
// outside the rows it may write is as it was before it, the deleted ones
// included: Web's other labels, Fonts among them after its deletion, and
// Ops's, of the same names. The rows it may write, an upper bound, are its
// new label, the label it names, and the undeleted labels under the label
// it deletes; a refused write may write none. What a write does to the
// rows it may write is not read here.
func TestEachLabelWriteChangesItsRowsAlone(t *testing.T) {
	w := newMemberWorld(t)
	ids := w.seedLabelsOf(t, w.web)
	w.seedLabelsOf(t, w.ops)
	labels, feature := "/api/v0/projects/"+w.web.String()+"/labels", "/api/v0/labels/"+ids["Feature"].String()
	docs := "/api/v0/labels/" + ids["Docs"].String()
	for _, s := range []struct {
		name, method string
		path         string // "" for the label of the first row it may write, one an earlier write made
		body         string
		status       int
		writes       func() []uuid.UUID // the rows there before it that it may write
		creates      string             // the name of the label it creates, a row it may write too
		// refusal, when set, is the problem a refused write answers (refusalOf):
		// its code, then its field error (field and code) when it has one.
		refusal string
	}{
		{"createLabel", http.MethodPost, labels, `{"name":"QA"}`, http.StatusCreated, nil, "QA", ""},
		{"createLabel, under Feature", http.MethodPost, labels, `{"name":"Icons","parent_id":"` + ids["Feature"].String() + `"}`,
			http.StatusCreated, nil, "Icons", ""},
		{"createLabel, under Docs", http.MethodPost, labels, `{"name":"Fonts","parent_id":"` + ids["Docs"].String() + `"}`,
			http.StatusCreated, nil, "Fonts", ""},
		{"updateLabel", http.MethodPatch, feature, `{"name":"Story"}`, http.StatusOK, func() []uuid.UUID { return []uuid.UUID{ids["Feature"]} }, "",
			""},
		{"updateLabel, a label with labels under it given a parent", http.MethodPatch, "/api/v0/labels/" + ids["Bug"].String(),
			`{"parent_id":"` + ids["Docs"].String() + `"}`, http.StatusUnprocessableEntity, nil, "", "validation_failed parent_id not_allowed"},
		{"updateLabel, to the top", http.MethodPatch, "/api/v0/labels/" + ids["UI"].String(), `{"parent_id":null}`, http.StatusOK,
			func() []uuid.UUID { return []uuid.UUID{ids["UI"]} }, "", ""},
		{"createLabel, a name taken", http.MethodPost, labels, `{"name":"bug"}`, http.StatusConflict, nil, "", "project.label_name_taken"},
		{"deleteLabel, a label under another", http.MethodDelete, "", "", http.StatusNoContent,
			func() []uuid.UUID { return []uuid.UUID{labelID(t, w.pool, w.web, "Fonts")} }, "", ""},
		{"updateLabel, a label whose one label under it is deleted given a parent", http.MethodPatch, docs,
			`{"parent_id":"` + ids["Bug"].String() + `"}`, http.StatusOK, func() []uuid.UUID { return []uuid.UUID{ids["Docs"]} }, "", ""},
		{"deleteLabel", http.MethodDelete, feature, "", http.StatusNoContent,
			func() []uuid.UUID { return []uuid.UUID{ids["Feature"], labelID(t, w.pool, w.web, "Icons")} }, "", ""},
		{"deleteLabel, a label whose label under it is deleted", http.MethodDelete, docs, "", http.StatusNoContent,
			func() []uuid.UUID { return []uuid.UUID{ids["Docs"]} }, "", ""},
	} {
		var writes []uuid.UUID
		if s.writes != nil {
			writes = s.writes()
		}
		path := s.path
		if path == "" {
			path = "/api/v0/labels/" + writes[0].String()
		}
		before := rowsBut(t, w.pool, writes)
		var body []byte
		if s.body != "" {
			body = []byte(s.body)
		}
		req := newRequest(t, s.method, w.base+path, w.tokens["bob"], body)
		w.contract.CheckRequest(t, req)
		res, answer := send(t, req)
		w.contract.CheckResponse(t, req, res)
		if res.StatusCode != s.status || s.refusal != "" && refusalOf(t, answer) != s.refusal {
			t.Fatalf("%s = %d %s, want %d %s", s.name, res.StatusCode, answer, s.status, s.refusal)
		}
		if s.creates != "" {
			writes = append(writes, labelID(t, w.pool, w.web, s.creates))
		}
		if after := rowsBut(t, w.pool, writes); !maps.Equal(after, before) {
			t.Errorf("%s: every other row after it:\n%v\nwant them as they were:\n%v", s.name, after, before)
		}
	}
}

// refusalOf is the problem body answers: its code (problemCode), then its
// field error (oneError) when it has exactly one, as "validation_failed
// parent_id not_allowed" or "project.label_name_taken".
func refusalOf(t *testing.T, body []byte) string {
	t.Helper()
	if one := oneError(t, body); one != "" {
		return problemCode(t, body) + " " + one
	}
	return problemCode(t, body)
}
