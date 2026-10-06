package bootstrap

import (
	"maps"
	"net/http"
	"testing"
	"uuid"
)

// Each write on a label, as bootstrap wires it, changes no row but the
// rows it writes (M3 design 3.16): on memberWorld, whose acme has Web and
// Ops, each with worldLabels (seedLabelsOf), bob, Web's admin, creates QA
// at the top of Web, Icons under Feature and Fonts under Docs, renames
// Feature, moves UI to the top, deletes Fonts, moves Docs, whose one label
// under it is deleted, under Bug, and deletes Feature, which takes Icons
// with it, and Docs, whose Fonts, deleted before, keeps its deletion; one
// write after another. Between them, his move of Bug, which has UI under
// it, under Docs (422 parent_id not_allowed) and his creation of a label
// named as Bug in another case (409 project.label_name_taken) write
// nothing. After each write, every row of every table but the ones it
// writes is as it was before it: Web's other labels, the deleted ones
// among them, and Ops's, of the same names. The rows left out are its new
// label, the label it names, and the undeleted labels under the one it
// deletes; a refused write leaves none out.
func TestEachLabelWriteChangesItsRowsAlone(t *testing.T) {
	w := newMemberWorld(t)
	ids := w.seedLabelsOf(t, w.web)
	w.seedLabelsOf(t, w.ops)
	labels, feature := "/api/v0/projects/"+w.web.String()+"/labels", "/api/v0/labels/"+ids["Feature"].String()
	docs := "/api/v0/labels/" + ids["Docs"].String()
	for _, s := range []struct {
		name, method string
		path         string // "" for the label of the first row it writes, one an earlier write made
		body         string
		status       int
		writes       func() []uuid.UUID // the rows it writes that are there before it
		creates      string             // the name of the label it creates, a row it writes too
		// refusal, when set, is the one problem (field and code) a refused
		// write answers.
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
			`{"parent_id":"` + ids["Docs"].String() + `"}`, http.StatusUnprocessableEntity, nil, "", "parent_id not_allowed"},
		{"updateLabel, to the top", http.MethodPatch, "/api/v0/labels/" + ids["UI"].String(), `{"parent_id":null}`, http.StatusOK,
			func() []uuid.UUID { return []uuid.UUID{ids["UI"]} }, "", ""},
		{"createLabel, a name taken", http.MethodPost, labels, `{"name":"bug"}`, http.StatusConflict, nil, "", ""},
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
		if res.StatusCode != s.status || s.refusal != "" && oneError(t, answer) != s.refusal {
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
