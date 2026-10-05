package bootstrap

import (
	"maps"
	"net/http"
	"testing"
	"uuid"
)

// Each write on a state, as bootstrap wires it, changes no row but the
// rows it writes (M3 design 3.17): on memberWorld, whose acme has Web and
// Ops and whose beta has Lab, each with its states, bob, Web's admin,
// creates Shipped in Web, renames QA, deletes it, renames Backlog, the
// only state of its group, giving it its own group again, which is no
// move (answered 200, it is not refused as its group's last), and makes
// Done Web's default, one write after another. Between them, his creation
// of a state in the triage group and his move of Done to it, bodies the
// contract's enum leaves out and the server must still answer as the
// contract says, are each 422 group not_allowed and write nothing. That
// they are refused before the write's transaction begins is the unit
// tests' to show: TestCreateStateRefuses/the_triage_group and
// TestUpdateStateRefuses/the_triage_group, whose call logs are empty.
// After each write, every row of every table but the ones it writes is as
// it was before it: Web's other states, its triage state among them,
// Ops's, in acme too, and Lab's, in beta. The rows left out are its new
// state, the state it names, and, for the default, Web's default before
// it, Backlog; a refused write leaves none out.
func TestEachStateWriteChangesItsRowsAlone(t *testing.T) {
	w := newMemberWorld(t)
	qa, done, backlog := stateID(t, w.pool, w.web, "QA"), stateID(t, w.pool, w.web, "Done"), stateID(t, w.pool, w.web, "Backlog")
	for _, s := range []struct {
		name, method, path, body string
		status                   int
		writes                   []uuid.UUID // the rows it writes that are there before it
		creates                  string      // the name of the state it creates, a row it writes too
		// refusal, when set, is the one problem (field and code) a refused
		// write answers, for a body the contract's enum leaves out.
		refusal string
	}{
		{"createState", http.MethodPost, "/api/v0/projects/" + w.web.String() + "/states", `{"name":"Shipped","color":"#46A758","group":"completed"}`,
			http.StatusCreated, nil, "Shipped", ""},
		{"updateState", http.MethodPatch, "/api/v0/states/" + qa.String(), `{"name":"Checked"}`, http.StatusOK, []uuid.UUID{qa}, "", ""},
		{"deleteState", http.MethodDelete, "/api/v0/states/" + qa.String(), "", http.StatusNoContent, []uuid.UUID{qa}, "", ""},
		{"updateState, its own group", http.MethodPatch, "/api/v0/states/" + backlog.String(), `{"name":"Later","group":"backlog"}`, http.StatusOK,
			[]uuid.UUID{backlog}, "", ""},
		{"createState, the triage group", http.MethodPost, "/api/v0/projects/" + w.web.String() + "/states",
			`{"name":"Intake","color":"#4E5355","group":"triage"}`, http.StatusUnprocessableEntity, nil, "", "group not_allowed"},
		{"updateState, to the triage group", http.MethodPatch, "/api/v0/states/" + done.String(), `{"group":"triage"}`,
			http.StatusUnprocessableEntity, nil, "", "group not_allowed"},
		{"markDefaultState", http.MethodPost, "/api/v0/states/" + done.String() + "/mark-default", "", http.StatusNoContent,
			[]uuid.UUID{done, backlog}, "", ""},
	} {
		before := rowsBut(t, w.pool, s.writes)
		var body []byte
		if s.body != "" {
			body = []byte(s.body)
		}
		req := newRequest(t, s.method, w.base+s.path, w.tokens["bob"], body)
		if s.refusal == "" {
			w.contract.CheckRequest(t, req)
		}
		res, answer := send(t, req)
		w.contract.CheckResponse(t, req, res)
		if res.StatusCode != s.status || s.refusal != "" && oneError(t, answer) != s.refusal {
			t.Fatalf("%s = %d %s, want %d %s", s.name, res.StatusCode, answer, s.status, s.refusal)
		}
		written := s.writes
		if s.creates != "" {
			written = append(written, stateID(t, w.pool, w.web, s.creates))
		}
		if after := rowsBut(t, w.pool, written); !maps.Equal(after, before) {
			t.Errorf("%s: every other row after it:\n%v\nwant them as they were:\n%v", s.name, after, before)
		}
	}
}
