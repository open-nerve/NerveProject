package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	issuepg "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
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

// Ordinary scope excludes ended/deleted memberships and archived/deleted
// projects, while the explicitly authorized archived source keeps its scope.
func TestRelationListUsesVisibleProjects(t *testing.T) {
	w := newIssueWorld(t)
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	w.makeMember(t, "walt", ops, shared.RoleGuest)
	w.adminGuest(t)
	source := w.newWorkItemBy(t, "walt", w.web, `{"name":"Source"}`)
	local := w.newWorkItemBy(t, "walt", w.web, `{"name":"Local"}`)
	localOther := w.newWorkItemIn(t, w.web, `{"name":"Local other"}`)
	parent := w.newWorkItemIn(t, ops, `{"name":"Hidden parent"}`)
	own := w.newWorkItemBy(t, "walt", ops, `{"name":"Own"}`)
	other := w.newWorkItemIn(t, ops, `{"name":"Other"}`)
	unknown := w.newWorkItemIn(t, ops, `{"name":"Unknown"}`)
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+own.String(), fmt.Sprintf(`{"parent_id":%q}`, parent), 200)
	w.newWorkItemBy(t, "walt", ops, fmt.Sprintf(`{"name":"Own child","parent_id":%q}`, own))
	w.newWorkItemIn(t, ops, fmt.Sprintf(`{"name":"Other child","parent_id":%q}`, own))
	archived := w.newWorkItemIn(t, ops, fmt.Sprintf(`{"name":"Archived","state_id":%q,"parent_id":%q}`, stateID(t, w.pool, ops, "Done"), own))
	gone := w.newWorkItemIn(t, ops, fmt.Sprintf(`{"name":"Gone","parent_id":%q}`, own))
	for _, target := range []uuid.UUID{local, other, own, unknown, archived, gone, localOther} {
		w.send(t, "alice", http.MethodPost, relationsOf(source), relating("blocking", target), 201)
	}
	for _, kind := range []string{"Left", "Deleted member", "Deleted project", "Archived project"} {
		p := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", kind, "P"+fmt.Sprint(len(kind)))
		w.makeMember(t, "walt", p, shared.RoleAdmin)
		id := w.newWorkItemBy(t, "walt", p, `{"name":"Excluded"}`)
		w.send(t, "alice", http.MethodPost, relationsOf(source), relating("relates_to", id), 201)
		statement := "UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2"
		switch kind {
		case "Deleted member":
			statement = "UPDATE project_members SET deleted_at=now() WHERE project_id=$1 AND member_id=$2"
		case "Deleted project":
			statement = "UPDATE projects SET deleted_at=now(),updated_by_id=$2 WHERE id=$1"
		case "Archived project":
			statement = "UPDATE projects SET archived_at=now(),updated_by_id=$2 WHERE id=$1"
		}
		if _, err := w.pool.Exec(context.Background(), statement, p, w.ids["walt"]); err != nil {
			t.Fatal(err)
		}
	}
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+archived.String()+"/archive", "", 200)
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+gone.String(), "", 204)
	if _, err := w.pool.Exec(context.Background(), "UPDATE issues SET created_by_id=NULL WHERE id=$1", unknown); err != nil {
		t.Fatal(err)
	}
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", "", 200)
	rows := listIn[answeredRelation](t, w.send(t, "walt", http.MethodGet, relationsOf(source), "", 200))
	if len(rows) != 2 || rows[0].Issue.ID != local || rows[1].Issue.ID != own || rows[1].RelationType != "blocking" || rows[1].Issue.ParentID != nil || rows[1].Issue.SubIssuesCount != 1 {
		t.Fatalf("caller scope/order/parent/count: %+v", rows)
	}
}

// One connection is occupied by the independent RO/RR transaction. Any source,
// authorization, directory, relation, count or parent read escaping ctx times out.
func TestRelationListReadSnapshot(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	source := w.newWorkItemBy(t, "walt", w.web, `{"name":"Source"}`)
	parent := w.newWorkItem(t)
	other := w.newWorkItemBy(t, "walt", w.web, `{"name":"Other"}`)
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+other.String(), fmt.Sprintf(`{"parent_id":%q}`, parent), 200)
	w.send(t, "alice", http.MethodPost, relationsOf(source), relating("duplicate", other), 201)
	one := poolOfOne(t, w.pool.Config().ConnString())
	dirs := issueProjects{projects: project.Provide(one).IssueProjects}
	store := issuepg.New(one)
	tx := &checkedListSnapshot{t: t, pool: one, tx: postgres.NewTxManager(one, 3*time.Second, postgres.ReadOnlySnapshot())}
	u := issueapp.NewListIssueRelations(issueapp.ListIssueRelationsDeps{Locks: issueapp.NewLocks(dirs, store, authorizerOn(one), versionReadOnly{}, versionReadOnly{}), Relations: store, Projects: dirs, ReadTx: tx})
	ctx, cancel := context.WithTimeout(shared.WithActor(context.Background(), shared.Actor{UserID: w.ids["walt"]}), 5*time.Second)
	defer cancel()
	rows, err := u.Execute(ctx, source)
	if err != nil || len(rows) != 1 || rows[0].Issue.ID != other || rows[0].Issue.ParentID != nil || tx.calls != 1 {
		t.Fatal("relation read escaped snapshot", rows, err, tx.calls)
	}
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
// Archived, Shelved is in no other's list, but its own still shows Near,
// whose duplicate it is. Unarchived, it shows again. Once Ops is
// archived, Far, of a project other than Near's, is no longer in Near's
// list, but Far's own list still shows Near and Beside, of Far's own
// project. walt, acme's admin and Web's guest (adminGuest), who sees only
// his own work items in Web, relates his His to his Other, which alice
// puts under her Up: he lists the relation as alice does, compared whole,
// but with Other under none, since Up is hidden from him.
func TestARelationListShowsWhatTheCallerSees(t *testing.T) {
	w := newIssueWorld(t)
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	w.makeMember(t, "bob", ops, shared.RoleMember)
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
		relationList{"bob", mid, []string{"blocking Near"}}, relationList{"alice", shelved, []string{"duplicate Near"}})
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
