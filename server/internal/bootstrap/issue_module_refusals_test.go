package bootstrap

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"uuid"
)

// The app refuses a work item's modules as spec R2 orders it, on a
// creation and a change alike, writing nothing: an archived module joined
// by its refusal, before every 422; a deleted module, another project's and
// none at all at their places in module_ids as given, with the label's and
// the cycle's problems in one 422; more than a hundred, or one named twice,
// at their places, before the work item is looked up. The project's
// modules turned off refuse a change, leaving too, and a creation in
// modules, before an archived module; a change of nothing, a creation in
// none and a change of another field are not refused.
func TestJoiningModulesIsRefusedInOrder(t *testing.T) {
	w := newIssueWorld(t)
	w.turnCyclesOn(t, w.web)
	w.turnModulesOn(t, w.web)
	auth := w.newModule(t, `{"name":"Auth"}`)
	shelved := w.newModule(t, `{"name":"Shelved","status":"completed"}`)
	w.send(t, "alice", http.MethodPost, "/api/v0/modules/"+shelved.String()+"/archive", "", http.StatusOK)
	gone := w.newModule(t, `{"name":"Gone"}`)
	w.send(t, "alice", http.MethodDelete, "/api/v0/modules/"+gone.String(), "", http.StatusNoContent)
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	w.turnModulesOn(t, ops)
	var theirs moduleAnswer
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+ops.String()+"/modules", `{"name":"Theirs"}`, http.StatusCreated), &theirs)
	item := w.newWorkItemIn(t, w.web, `{"name":"Fix","module_ids":["`+auth.String()+`"]}`)
	create, change := "/api/v0/projects/"+w.web.String()+"/issues", "/api/v0/issues/"+item.String()
	in := func(more string, modules ...uuid.UUID) string {
		quoted := make([]string, len(modules))
		for i, id := range modules {
			quoted[i] = fmt.Sprintf("%q", id)
		}
		return fmt.Sprintf(`{"name":"Fix","module_ids":[%s]%s}`, strings.Join(quoted, ","), more)
	}
	ghost := `,"label_ids":["` + uuid.NewV7().String() + `"]`
	nowhere := `,"cycle_id":"` + uuid.NewV7().String() + `"`
	none := uuid.NewV7()
	for _, path := range []string{create, change} {
		method := map[string]string{create: http.MethodPost, change: http.MethodPatch}[path]
		for _, tt := range []struct {
			body   string
			status int
			want   string
		}{
			{in("", shelved), http.StatusConflict, "planning.archived"},
			{in(ghost+nowhere, auth, gone, shelved), http.StatusConflict, "planning.archived"},
			{in("", gone), http.StatusUnprocessableEntity, "validation_failed module_ids[0] not_allowed"},
			{in("", auth, theirs.ID), http.StatusUnprocessableEntity, "validation_failed module_ids[1] not_allowed"},
			{in("", none, auth, gone), http.StatusUnprocessableEntity, "validation_failed module_ids[0] not_allowed; module_ids[2] not_allowed"},
			{in(ghost+nowhere, auth, theirs.ID, gone), http.StatusUnprocessableEntity,
				"validation_failed label_ids[0] not_allowed; cycle_id not_allowed; module_ids[1] not_allowed; module_ids[2] not_allowed"},
		} {
			w.refusedJoin(t, "bob", method, path, tt.body, tt.status, tt.want)
		}
	}
	many := make([]uuid.UUID, 101)
	for i := range many {
		many[i] = uuid.NewV7()
	}
	twice := in("", gone, auth, gone, none, auth)
	for _, path := range []string{create, "/api/v0/issues/" + uuid.NewV7().String()} {
		method := map[bool]string{true: http.MethodPost, false: http.MethodPatch}[path == create]
		w.refusedJoin(t, "bob", method, path, in("", many...), http.StatusUnprocessableEntity, "validation_failed module_ids too_long")
		w.refusedJoin(t, "bob", method, path, twice, http.StatusUnprocessableEntity,
			"validation_failed module_ids[2] duplicate; module_ids[4] duplicate")
	}

	// Web's modules turned off: a change of the modules, leaving them too,
	// and a creation in modules, refused before an archived module; a
	// change of nothing, a creation in none and a change of another field
	// are not.
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"module_view":false}`, http.StatusOK)
	for _, tt := range []struct{ method, path, body string }{{http.MethodPost, create, in("", shelved)},
		{http.MethodPatch, change, in("", auth, shelved)}, {http.MethodPatch, change, `{"module_ids":[]}`}} {
		w.refusedJoin(t, "bob", tt.method, tt.path, tt.body, http.StatusConflict, "planning.modules_disabled")
	}
	before := w.cycleWritten(t)
	w.send(t, "bob", http.MethodPatch, change, in("", auth), http.StatusOK)
	if after := w.cycleWritten(t); after != before {
		t.Errorf("a change of nothing wrote with modules off: %s, was %s", after, before)
	}
	w.newWorkItemIn(t, w.web, `{"name":"Out of modules","module_ids":[]}`)
	w.send(t, "bob", http.MethodPatch, change, `{"name":"Mend"}`, http.StatusOK)
}
