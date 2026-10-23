package bootstrap

import (
	"net/http"
	"reflect"
	"slices"
	"testing"
	"uuid"
)

// listedRelations are the relations the account name lists of the work
// item id, in their order, each as "<type> <other work item's name>".
func (w issueWorld) listedRelations(t *testing.T, name string, id uuid.UUID) []string {
	t.Helper()
	var said []string
	for _, r := range listIn[answeredRelation](t, w.send(t, name, http.MethodGet, relationsOf(id), "", http.StatusOK)) {
		said = append(said, r.RelationType+" "+r.Issue.Name)
	}
	return said
}

// relationList is a list of relations a test expects: the account name
// lists the work item item's, which are want (listedRelations).
type relationList struct {
	name string
	item uuid.UUID
	want []string
}

// A work item's relations as bootstrap wires them (M4 design 3.4, 3.8,
// 3.10): alice makes Ops, bob its member, and Lab, private, of which he is
// none; Near, Mid, Shelved and Gone in Web, Far and Beside in Ops, Lab's
// in Lab. alice relates Near to Far, and Far to Beside; bob Mid to Near,
// as blocking it; alice Near to Lab's, Shelved and Gone, as their
// duplicate. Then Shelved is archived and Gone deleted. alice lists Near's
// relations in the order they were made, each as Near reads it: Far, Mid,
// which blocks it, and Lab's; bob, who does not see Lab, lists Far and
// Mid alone. Far lists Near and Beside, and Mid Near, which it blocks.
// Shelved unarchived shows again. Once Ops is archived, Far, of a project
// other than Near's, is no longer in Near's list, but Far's own list
// still shows Near and Beside, of Far's own project. walt, acme's admin
// and Web's guest (adminGuest), who sees only his own work items in Web,
// relates his His to his Other, which alice puts under her Up: he lists
// the relation as alice does, compared whole, but with Other under none,
// since Up is hidden from him.
func TestARelationListShowsWhatTheCallerSees(t *testing.T) {
	w := newIssueWorld(t)
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+ops.String()+"/members",
		`{"members":[{"member_id":"`+w.ids["bob"].String()+`","role":15}]}`, http.StatusCreated)
	lab := createdProjectFrom(t, w.contract, w.base, w.tokens["alice"], "acme", `{"name":"Lab","identifier":"LAB","network":0}`)
	near := w.newWorkItemIn(t, w.web, `{"name":"Near"}`)
	mid := w.newWorkItemIn(t, w.web, `{"name":"Mid"}`)
	far := w.newWorkItemIn(t, ops, `{"name":"Far"}`)
	beside := w.newWorkItemIn(t, ops, `{"name":"Beside"}`)
	labs := w.newWorkItemIn(t, lab, `{"name":"Lab's"}`)
	shelved := w.newWorkItemIn(t, w.web, `{"name":"Shelved","state_id":"`+w.done.String()+`"}`)
	gone := w.newWorkItemIn(t, w.web, `{"name":"Gone"}`)
	w.send(t, "alice", http.MethodPost, relationsOf(near), relating("relates_to", far), http.StatusCreated)
	w.send(t, "alice", http.MethodPost, relationsOf(far), relating("relates_to", beside), http.StatusCreated)
	w.send(t, "bob", http.MethodPost, relationsOf(mid), relating("blocking", near), http.StatusCreated)
	w.send(t, "alice", http.MethodPost, relationsOf(near), relating("duplicate", gone, shelved, labs), http.StatusCreated)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+shelved.String()+"/archive", "", http.StatusOK)
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+gone.String(), "", http.StatusNoContent)

	lists := func(when string, expected ...relationList) {
		t.Helper()
		for _, l := range expected {
			if got := w.listedRelations(t, l.name, l.item); !slices.Equal(got, l.want) {
				t.Errorf("%s, %s lists %s's relations %q, want %q", when, l.name, l.item, got, l.want)
			}
		}
	}
	lists("at first", relationList{"alice", near, []string{"relates_to Far", "blocked_by Mid", "duplicate Lab's"}},
		relationList{"bob", near, []string{"relates_to Far", "blocked_by Mid"}},
		relationList{"bob", far, []string{"relates_to Near", "relates_to Beside"}},
		relationList{"bob", mid, []string{"blocking Near"}})
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+shelved.String()+"/unarchive", "", http.StatusOK)
	lists("Shelved unarchived", relationList{"bob", near, []string{"relates_to Far", "blocked_by Mid", "duplicate Shelved"}})
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+ops.String()+"/archive", "", http.StatusOK)
	lists("Ops archived", relationList{"bob", near, []string{"blocked_by Mid", "duplicate Shelved"}},
		relationList{"bob", far, []string{"relates_to Near", "relates_to Beside"}})

	w.adminGuest(t)
	up := w.newWorkItemIn(t, w.web, `{"name":"Up"}`)
	his, other := w.newWorkItemBy(t, "walt", w.web, `{"name":"His"}`), w.newWorkItemBy(t, "walt", w.web, `{"name":"Other"}`)
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+other.String(), `{"parent_id":"`+up.String()+`"}`, http.StatusOK)
	w.send(t, "walt", http.MethodPost, relationsOf(his), relating("relates_to", other), http.StatusCreated)
	var listed [2]answeredRelation
	for n, name := range []string{"walt", "alice"} {
		relations := listIn[answeredRelation](t, w.send(t, name, http.MethodGet, relationsOf(his), "", http.StatusOK))
		if len(relations) != 1 {
			t.Fatalf("%s lists %d relations of His, want 1", name, len(relations))
		}
		listed[n] = relations[0]
	}
	want := listed[1]
	if want.Issue.ParentID == nil || *want.Issue.ParentID != up {
		t.Errorf("alice lists Other under %v, want under Up, %s", want.Issue.ParentID, up)
	}
	want.Issue.ParentID = nil
	if !reflect.DeepEqual(listed[0], want) {
		t.Errorf("walt lists His's relation %+v; want alice's, with Other under none, %+v", listed[0], want)
	}
}
