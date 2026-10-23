package bootstrap

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// relationsOf is the path of the relations of the work item id.
func relationsOf(id uuid.UUID) string {
	return "/api/v0/issues/" + id.String() + "/relations"
}

// relating is the body that relates a work item, as kind, to the work
// items ids, in their order.
func relating(kind string, ids ...uuid.UUID) string {
	return `{"relation_type":"` + kind + `","issue_ids":` + idArray(ids...) + `}`
}

// answeredRelation is a relation as an answer has it.
type answeredRelation struct {
	ID           uuid.UUID    `json:"id"`
	RelationType string       `json:"relation_type"`
	Issue        countedIssue `json:"issue"`
	CreatedAt    time.Time    `json:"created_at"`
}

// othersOf are the other work items of relations, in their order.
func othersOf(relations []answeredRelation) []countedIssue {
	others := make([]countedIssue, len(relations))
	for n, r := range relations {
		others[n] = r.Issue
	}
	return others
}

// storedRelations are the undeleted relations, sorted, each as "<work
// item> <type> <work item>, in <project's identifier>, by <maker>, <last
// writer> at <moment>", the work items by their names, an account by its
// name in the world, the moment "the moment" when it was made and changed
// last at at, else "another moment".
func (w issueWorld) storedRelations(t *testing.T, at time.Time) []string {
	t.Helper()
	rows, err := w.pool.Query(pgtest.Soon(t), `SELECT i.name, r.relation_type, o.name, p.identifier, r.created_by_id, r.updated_by_id,
		r.created_at, r.updated_at FROM issue_relations r JOIN issues i ON i.id = r.issue_id JOIN issues o ON o.id = r.related_issue_id
		JOIN projects p ON p.id = r.project_id WHERE r.deleted_at IS NULL`)
	if err != nil {
		t.Fatal(err)
	}
	names := w.names()
	lines, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (string, error) {
		var from, kind, to, project string
		var madeBy, changedBy uuid.UUID
		var madeAt, changedAt time.Time
		if err := row.Scan(&from, &kind, &to, &project, &madeBy, &changedBy, &madeAt, &changedAt); err != nil {
			return "", err
		}
		when := "at the moment"
		if !madeAt.Equal(at) || !changedAt.Equal(at) {
			when = "at another moment"
		}
		return from + " " + kind + " " + to + ", in " + project + ", by " + names[madeBy] + ", " + names[changedBy] + " " + when, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return slices.Sorted(slices.Values(lines))
}

// relationRows are the rows of the work item item's activity on its
// relations (rows 20–23 of M4 design 3.13), in their ids' order, each as
// "<actor>: <verb> <field> <old value> → <new value>, <old work item> →
// <new work item>, in <project's identifier>, <comment>", a work item by
// its name, none for NULL, and " at the moment" when it was made at at.
func (w issueWorld) relationRows(t *testing.T, item uuid.UUID, at time.Time) []string {
	t.Helper()
	rows, err := w.pool.Query(pgtest.Soon(t), `SELECT a.actor_id, a.verb, a.field, a.old_value, a.new_value, coalesce(o.name, 'none'),
		coalesce(n.name, 'none'), p.identifier, a.comment, a.created_at FROM issue_activities a JOIN projects p ON p.id = a.project_id
		LEFT JOIN issues o ON o.id = a.old_identifier LEFT JOIN issues n ON n.id = a.new_identifier
		WHERE a.issue_id = $1 AND a.field IN ('relates_to', 'duplicate', 'blocked_by', 'blocking') ORDER BY a.id`, item)
	if err != nil {
		t.Fatal(err)
	}
	names := w.names()
	said, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (string, error) {
		var actor uuid.UUID
		var verb, field, oldValue, newValue, oldItem, newItem, project, comment string
		var madeAt time.Time
		if err := row.Scan(&actor, &verb, &field, &oldValue, &newValue, &oldItem, &newItem, &project, &comment, &madeAt); err != nil {
			return "", err
		}
		line := names[actor] + ": " + verb + " " + field + " " + oldValue + " → " + newValue + ", " + oldItem + " → " + newItem + ", in " +
			project + ", " + comment
		if madeAt.Equal(at) {
			line += " at the moment"
		}
		return line, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return said
}

// Relations as bootstrap wires them (M4 design 3.5, 3.7, 3.10, 3.13).
// alice makes Ops, of acme, with bob its member, then Near and Mid in Web,
// Far in Ops. bob relates Near to Far and Mid, blocking them: the answer
// is both, Mid then Far, in id order, each blocking, the other work item
// changed last by bob, at one moment; each is stored the other way round,
// as the other's blocked_by, a row of the other's project; Near, Mid and
// Far are changed last by bob then, and each relation is a row of Near's
// activity and one of the other's, in each one's project, the type as it
// reads it. Mid relating to Near, of any type, and Near to Far again, are
// refused as related already, at their places, nothing written. Then bob
// relates Far, of Ops, to Mid, of Web, as blocked by it, at another
// moment: the answer is blocked_by Mid; the relation is stored as asked,
// Far's blocked_by, a row of Ops, and is a row of Far's activity,
// blocked_by, and one of Mid's, blocking.
func TestWorkItemsAreRelatedAcrossProjects(t *testing.T) {
	w := newIssueWorld(t)
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+ops.String()+"/members",
		`{"members":[{"member_id":"`+w.ids["bob"].String()+`","role":15}]}`, http.StatusCreated)
	near := w.newWorkItemIn(t, w.web, `{"name":"Near"}`)
	mid := w.newWorkItemIn(t, w.web, `{"name":"Mid"}`)
	far := w.newWorkItemIn(t, ops, `{"name":"Far"}`)

	made := listIn[answeredRelation](t, w.send(t, "bob", http.MethodPost, relationsOf(near), relating("blocking", far, mid), http.StatusCreated))
	if len(made) != 2 {
		t.Fatalf("the answer is %d relations, want 2", len(made))
	}
	at := made[0].CreatedAt
	for n, want := range []uuid.UUID{mid, far} {
		r := made[n]
		if r.RelationType != "blocking" || r.Issue.ID != want || r.Issue.UpdatedByID == nil || *r.Issue.UpdatedByID != w.ids["bob"] ||
			!r.Issue.UpdatedAt.Equal(at) || !r.CreatedAt.Equal(at) {
			t.Errorf("relation %d is %+v; want blocking %s, changed last by bob, at %v", n, r, want, at)
		}
	}
	for _, item := range []uuid.UUID{near, mid, far} {
		if _, changed, by := archivalOf(t, w, item); !changed.Equal(at) || by != w.ids["bob"] {
			t.Errorf("the work item %s is changed last by %s at %v, want by bob at %v", item, by, changed, at)
		}
	}

	for _, tt := range []struct {
		item uuid.UUID
		body string
	}{{mid, relating("relates_to", near)}, {near, relating("duplicate", far)}} {
		answer := w.send(t, "bob", http.MethodPost, relationsOf(tt.item), tt.body, http.StatusUnprocessableEntity)
		if got, want := refusalOf(t, []byte(answer), false), "validation_failed issue_ids[0] duplicate"; got != want ||
			len(w.storedRelations(t, at)) != 2 || len(w.relationRows(t, near, at)) != 2 {
			t.Errorf("%s answers %s, with %d relations; want %s, and the two", tt.body, got, len(w.storedRelations(t, at)), want)
		}
	}

	back := listIn[answeredRelation](t, w.send(t, "bob", http.MethodPost, relationsOf(far), relating("blocked_by", mid), http.StatusCreated))
	if len(back) != 1 || back[0].RelationType != "blocked_by" || back[0].Issue.ID != mid {
		t.Fatalf("Far's relation to Mid is %+v; want blocked_by Mid alone", back)
	}
	if got, want := w.storedRelations(t, at), []string{"Far blocked_by Mid, in OPS, by bob, bob at another moment",
		"Far blocked_by Near, in OPS, by bob, bob at the moment",
		"Mid blocked_by Near, in WEB, by bob, bob at the moment"}; !slices.Equal(got, want) {
		t.Errorf("the relations stored:\n%q\nwant\n%q", got, want)
	}
	for _, tt := range []struct {
		item uuid.UUID
		want []string
	}{
		{near, []string{"bob: updated blocking  → WEB-2, none → Mid, in WEB, added blocking relation at the moment",
			"bob: updated blocking  → OPS-1, none → Far, in WEB, added blocking relation at the moment"}},
		{mid, []string{"bob: updated blocked_by  → WEB-1, none → Near, in WEB, added blocked_by relation at the moment",
			"bob: updated blocking  → OPS-1, none → Far, in WEB, added blocking relation"}},
		{far, []string{"bob: updated blocked_by  → WEB-1, none → Near, in OPS, added blocked_by relation at the moment",
			"bob: updated blocked_by  → WEB-2, none → Mid, in OPS, added blocked_by relation"}},
	} {
		if got := w.relationRows(t, tt.item, at); !slices.Equal(got, tt.want) {
			t.Errorf("the rows of %s's relations:\n%q\nwant\n%q", tt.item, got, tt.want)
		}
	}
}

// Each work item a work item may not be related to is refused at its
// place, in one 422, nothing written (M4 design 3.4, 3.8, 3.10): Near
// itself, a deleted work item, one not there, an archived one, one of an
// archived project bob is a member of, one of a project of acme bob is no
// member of, one of another workspace; Up, which it may be related to,
// beside them, is not. The one of another workspace is refused to alice
// too, who sees it. Naming no work item, Up twice, or Up 21 times is
// refused, 422, nothing written. An archived work item, and one of an
// archived project, relate to none (409).
func TestWhatAWorkItemMayNotBeRelatedTo(t *testing.T) {
	w := newIssueWorld(t)
	near := w.newWorkItemIn(t, w.web, `{"name":"Near"}`)
	up := w.newWorkItemIn(t, w.web, `{"name":"Up"}`)
	gone := w.newWorkItemIn(t, w.web, `{"name":"Gone"}`)
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+gone.String(), "", http.StatusNoContent)
	shelved := w.doneWorkItem(t)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+shelved.String()+"/archive", "", http.StatusOK)
	old := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Old", "OLD")
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+old.String()+"/members",
		`{"members":[{"member_id":"`+w.ids["bob"].String()+`","role":15}]}`, http.StatusCreated)
	olds := w.newWorkItemIn(t, old, `{"name":"Old's"}`)
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+old.String()+"/archive", "", http.StatusOK)
	labs := w.newWorkItemIn(t, createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Lab", "LAB"), `{"name":"Lab's"}`)
	w.send(t, "alice", http.MethodPost, "/api/v0/workspaces", `{"name":"Beta","slug":"beta"}`, http.StatusCreated)
	betas := w.newWorkItemIn(t, createdProject(t, w.contract, w.base, w.tokens["alice"], "beta", "Beta", "BETA"), `{"name":"Beta's"}`)

	answer := w.send(t, "bob", http.MethodPost, relationsOf(near), relating("relates_to", near, gone, uuid.NewV7(), shelved, olds, labs, betas,
		up), http.StatusUnprocessableEntity)
	var refused []string
	for _, field := range []string{"[0]", "[1]", "[2]", "[3]", "[4]", "[5]", "[6]"} {
		refused = append(refused, "issue_ids"+field+" not_allowed")
	}
	if got, want := refusalOf(t, []byte(answer), false), "validation_failed "+strings.Join(refused, "; "); got != want ||
		len(w.storedRelations(t, time.Time{})) != 0 || len(w.relationRows(t, up, time.Time{})) != 0 {
		t.Errorf("the work items Near may not be related to answer %s, with %d relations; want %s, and none", got,
			len(w.storedRelations(t, time.Time{})), want)
	}
	answer = w.send(t, "alice", http.MethodPost, relationsOf(near), relating("relates_to", betas), http.StatusUnprocessableEntity)
	if got := refusalOf(t, []byte(answer), false); got != "validation_failed issue_ids[0] not_allowed" ||
		len(w.storedRelations(t, time.Time{})) != 0 {
		t.Errorf("alice's relating Near to Beta's, of another workspace, answers %s, with %d relations; want issue_ids[0] not_allowed, and none",
			got, len(w.storedRelations(t, time.Time{})))
	}
	for body, refusal := range map[string]string{relating("relates_to"): "issue_ids too_short", relating("relates_to", up, up): "issue_ids[1] duplicate",
		relating("relates_to", slices.Repeat([]uuid.UUID{up}, 21)...): "issue_ids too_long"} {
		answer = w.send(t, "bob", http.MethodPost, relationsOf(near), body, http.StatusUnprocessableEntity)
		if got := refusalOf(t, []byte(answer), false); got != "validation_failed "+refusal || len(w.storedRelations(t, time.Time{})) != 0 {
			t.Errorf("%.60s answers %s, with %d relations; want %s, and none", body, got, len(w.storedRelations(t, time.Time{})), refusal)
		}
	}
	for _, tt := range []struct {
		item uuid.UUID
		code string
	}{{shelved, "issue.archived"}, {olds, "project.archived"}} {
		answer := w.send(t, "alice", http.MethodPost, relationsOf(tt.item), relating("relates_to", up), http.StatusConflict)
		if got := refusalOf(t, []byte(answer), false); got != tt.code || len(w.storedRelations(t, time.Time{})) != 0 {
			t.Errorf("relating %s answers %s, with %d relations; want %s, and none", tt.item, got, len(w.storedRelations(t, time.Time{})),
				tt.code)
		}
	}
}

// The guest rule holds for relations (M4 design 3.4, 3.10): walt, acme's
// admin and Web's guest (adminGuest), sees only his own work items in Web.
// He makes His and Mine, alice Hers between them; under Mine, alice makes
// Low and walt Below. walt relates His to Mine and Hers, which he does
// not see, refused at its place with the very answer of a work item not
// there: nothing written. alice relates Hers to His; then walt His to Mine
// alone, answered with the one sub-item of Mine he sees. walt lists His's
// relations: Mine alone, with its one sub-item, though Hers, of Web too,
// comes first; alice lists Hers and Mine, with its two. Up, alice's, walt
// finds not to relate: the very 404 of a work item not there.
func TestTheGuestRuleHoldsForRelations(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	up := w.newWorkItemIn(t, w.web, `{"name":"Up"}`)
	his := w.newWorkItemBy(t, "walt", w.web, `{"name":"His"}`)
	hers := w.newWorkItemIn(t, w.web, `{"name":"Hers"}`)
	mine := w.newWorkItemBy(t, "walt", w.web, `{"name":"Mine"}`)
	w.newWorkItemIn(t, w.web, underParent("Low", mine))
	w.newWorkItemBy(t, "walt", w.web, underParent("Below", mine))
	answer := w.send(t, "walt", http.MethodPost, relationsOf(his), relating("relates_to", mine, hers), http.StatusUnprocessableEntity)
	notThere := w.send(t, "walt", http.MethodPost, relationsOf(his), relating("relates_to", mine, uuid.NewV7()),
		http.StatusUnprocessableEntity)
	if got := refusalOf(t, []byte(answer), false); got != "validation_failed issue_ids[1] not_allowed" || answer != notThere ||
		len(w.storedRelations(t, time.Time{})) != 0 {
		t.Errorf("walt's relating His to Mine and Hers answers %s, with %d relations; want issue_ids[1] not_allowed, as for one not there: %s",
			answer, len(w.storedRelations(t, time.Time{})), notThere)
	}
	w.send(t, "alice", http.MethodPost, relationsOf(hers), relating("duplicate", his), http.StatusCreated)
	made := listIn[answeredRelation](t, w.send(t, "walt", http.MethodPost, relationsOf(his), relating("relates_to", mine), http.StatusCreated))
	if got := counts(othersOf(made)); !slices.Equal(got, []string{"Mine 1"}) {
		t.Fatalf("walt's relating His to Mine answers relations to %q; want one, to Mine with the one sub-item he sees", got)
	}
	if got, want := w.storedRelations(t, made[0].CreatedAt), []string{"Hers duplicate His, in WEB, by alice, alice at another moment",
		"His relates_to Mine, in WEB, by walt, walt at the moment"}; !slices.Equal(got, want) {
		t.Errorf("the relations stored:\n%q\nwant\n%q", got, want)
	}
	for _, tt := range []struct {
		name string
		want []string
	}{{"walt", []string{"Mine 1"}}, {"alice", []string{"Hers 0", "Mine 2"}}} {
		listed := listIn[answeredRelation](t, w.send(t, tt.name, http.MethodGet, relationsOf(his), "", http.StatusOK))
		if got := counts(othersOf(listed)); !slices.Equal(got, tt.want) {
			t.Errorf("%s lists His's relations to %q, want %q", tt.name, got, tt.want)
		}
	}
	hidden := w.send(t, "walt", http.MethodPost, relationsOf(up), relating("relates_to", mine), http.StatusNotFound)
	absent := w.send(t, "walt", http.MethodPost, relationsOf(uuid.NewV7()), relating("relates_to", mine), http.StatusNotFound)
	if hidden != absent {
		t.Errorf("walt's relating Up answers %s, want as for a work item not there: %s", hidden, absent)
	}
}
