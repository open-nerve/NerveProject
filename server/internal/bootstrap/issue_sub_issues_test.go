package bootstrap

import (
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"uuid"
)

// subIssues is the path of the sub-items of the work item id.
func subIssues(id uuid.UUID) string {
	return "/api/v0/issues/" + id.String() + "/sub-issues"
}

// subIssueIDs is the body that names the work items ids, in their order.
func subIssueIDs(ids ...uuid.UUID) string {
	return `{"sub_issue_ids":` + idArray(ids...) + `}`
}

// countedIssue is a work item as a list answers it: its fields
// (storedIssue), and the number of its sub-items the caller sees.
type countedIssue struct {
	storedIssue
	SubIssuesCount int `json:"sub_issues_count"`
}

// listIn are the elements of type T a list answers, in its order.
func listIn[T any](t *testing.T, answer string) []T {
	t.Helper()
	var list struct {
		Data []T `json:"data"`
	}
	decodeAnswer(t, answer, &list)
	if list.Data == nil {
		t.Fatalf("the answer %s has no list", answer)
	}
	return list.Data
}

// itemIDs are the ids of items, in their order.
func itemIDs(items []countedIssue) []uuid.UUID {
	ids := make([]uuid.UUID, len(items))
	for n, i := range items {
		ids[n] = i.ID
	}
	return ids
}

// counts are items, in their order, each as "<name> <number of its
// sub-items the answer shows>".
func counts(items []countedIssue) []string {
	said := make([]string, len(items))
	for n, i := range items {
		said[n] = i.Name + " " + strconv.Itoa(i.SubIssuesCount)
	}
	return said
}

// Sub-items as bootstrap wires them (M4 design 3.5, 3.8, 3.9, 3.13). alice
// makes Root, Top under Root, Up, A, B under Up, and C, WEB-1 to WEB-6.
// bob puts C, B and A under Top: the answer is A, B and C, in id order,
// each under Top and changed last by bob, and each a row of his activity,
// from none, from WEB-3 for B, to WEB-2. To alice, Top lists them and
// shows 3 sub-items: D, which bob puts under it too and archives, is in
// neither. Putting A under it again answers none and writes nothing;
// putting A and E answers E alone, and does not write A. Naming none, Up
// twice, or Up 101 times is refused, 422, nothing written. Then, once
// alice puts Root under Base, each work item Top may not take is refused
// at its place, in one 422, nothing written: Ops's work item, Top itself,
// Shelved, archived, Root, above it, Base, above Root, a deleted one, one
// not there; Up, which it may take, beside them, stays under none.
// Shelved, which alice archives with Kept under it, takes none (409), but
// lists Kept to bob, unarchived, under it, as getIssue answers it.
func TestWorkItemsArePutUnderAWorkItem(t *testing.T) {
	w := newIssueWorld(t)
	root := w.newWorkItemIn(t, w.web, `{"name":"Root"}`)
	top := w.newWorkItemIn(t, w.web, underParent("Top", root))
	up := w.newWorkItemIn(t, w.web, `{"name":"Up"}`)
	a := w.newWorkItemIn(t, w.web, `{"name":"A"}`)
	b := w.newWorkItemIn(t, w.web, underParent("B", up))
	c := w.newWorkItemIn(t, w.web, `{"name":"C"}`)
	moved := listIn[countedIssue](t, w.send(t, "bob", http.MethodPost, subIssues(top), subIssueIDs(c, b, a), http.StatusOK))
	if got := itemIDs(moved); !slices.Equal(got, []uuid.UUID{a, b, c}) {
		t.Errorf("the answer is %v, want A, B and C, %v", got, []uuid.UUID{a, b, c})
	}
	for _, i := range moved {
		if !reflect.DeepEqual(i.ParentID, &top) || !reflect.DeepEqual(w.parentOf(t, i.ID), &top) || i.UpdatedByID == nil ||
			*i.UpdatedByID != w.ids["bob"] {
			t.Errorf("%s answered under %v, changed last by %v, stored under %v; want under Top, by bob", i.Name, i.ParentID, i.UpdatedByID,
				w.parentOf(t, i.ID))
		}
	}
	for _, tt := range []struct {
		item uuid.UUID
		from string
	}{{a, " → WEB-2, none → "}, {b, "WEB-3 → WEB-2, " + up.String() + " → "}, {c, " → WEB-2, none → "}} {
		if said, want := w.parentRows(t, tt.item), []string{"bob: " + tt.from + top.String()}; !slices.Equal(said, want) {
			t.Errorf("the rows of %s's parent say %q, want %q", tt.item, said, want)
		}
	}
	d := w.doneWorkItem(t)
	w.send(t, "bob", http.MethodPost, subIssues(top), subIssueIDs(d), http.StatusOK)
	w.send(t, "bob", http.MethodPost, "/api/v0/issues/"+d.String()+"/archive", "", http.StatusOK)
	if got := itemIDs(listIn[countedIssue](t, w.send(t, "alice", http.MethodGet, subIssues(top), "", http.StatusOK))); !slices.Equal(got,
		[]uuid.UUID{a, b, c}) {
		t.Errorf("Top lists %v, want A, B and C", got)
	}
	if got := subItems(t, w.send(t, "alice", http.MethodGet, "/api/v0/issues/"+top.String(), "", http.StatusOK)); got != 3 {
		t.Errorf("Top shows %d sub-items, want 3", got)
	}
	_, changed, by := archivalOf(t, w, a)
	if again := listIn[countedIssue](t, w.send(t, "alice", http.MethodPost, subIssues(top), subIssueIDs(a), http.StatusOK)); len(again) != 0 {
		t.Errorf("A put under Top again answers %v, want none", itemIDs(again))
	}
	e := w.newWorkItemIn(t, w.web, `{"name":"E"}`)
	added := listIn[countedIssue](t, w.send(t, "alice", http.MethodPost, subIssues(top), subIssueIDs(a, e), http.StatusOK))
	if got := itemIDs(added); !slices.Equal(got, []uuid.UUID{e}) {
		t.Errorf("A and E put under Top answer %v, want E alone, %s", got, e)
	}
	if _, at, who := archivalOf(t, w, a); !at.Equal(changed) || who != by || len(w.parentRows(t, a)) != 1 {
		t.Errorf("A put under Top again is changed last by %s at %v, with %d rows of its parent; want as before, one row", who, at,
			len(w.parentRows(t, a)))
	}
	for body, refusal := range map[string]string{subIssueIDs(): "sub_issue_ids too_short", subIssueIDs(up, up): "sub_issue_ids[1] duplicate",
		subIssueIDs(slices.Repeat([]uuid.UUID{up}, 101)...): "sub_issue_ids too_long"} {
		answer := w.send(t, "bob", http.MethodPost, subIssues(top), body, http.StatusUnprocessableEntity)
		if got := refusalOf(t, []byte(answer), false); got != "validation_failed "+refusal || w.parentOf(t, up) != nil {
			t.Errorf("%.60s answers %s, and Up is under %v; want %s, and under none", body, got, w.parentOf(t, up), refusal)
		}
	}

	far := w.newWorkItemIn(t, createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS"), `{"name":"Far"}`)
	shelved := w.doneWorkItem(t)
	kept := w.newWorkItemIn(t, w.web, underParent("Kept", shelved))
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+shelved.String()+"/archive", "", http.StatusOK)
	gone := w.newWorkItemIn(t, w.web, `{"name":"Gone"}`)
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+gone.String(), "", http.StatusNoContent)
	base := w.newWorkItemIn(t, w.web, `{"name":"Base"}`)
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+root.String(), `{"parent_id":"`+base.String()+`"}`, http.StatusOK)
	answer := w.send(t, "bob", http.MethodPost, subIssues(top), subIssueIDs(far, top, shelved, root, base, gone, uuid.NewV7(), up),
		http.StatusUnprocessableEntity)
	refused := []string{}
	for _, field := range []string{"[0]", "[1]", "[2]", "[3]", "[4]", "[5]", "[6]"} {
		refused = append(refused, "sub_issue_ids"+field+" not_allowed")
	}
	if got, want := refusalOf(t, []byte(answer), false), "validation_failed "+strings.Join(refused, "; "); got != want ||
		w.parentOf(t, up) != nil {
		t.Errorf("the work items Top may not take answer %s, and Up is under %v; want %s, and under none", got, w.parentOf(t, up), want)
	}
	if answer := w.send(t, "bob", http.MethodPost, subIssues(shelved), subIssueIDs(up), http.StatusConflict); refusalOf(t, []byte(answer),
		false) != "issue.archived" || w.parentOf(t, up) != nil {
		t.Errorf("Up under an archived work item answers %s, and is under %v; want issue.archived, under none", answer, w.parentOf(t, up))
	}
	var read countedIssue
	decodeAnswer(t, w.send(t, "bob", http.MethodGet, "/api/v0/issues/"+kept.String(), "", http.StatusOK), &read)
	if got := listIn[countedIssue](t, w.send(t, "bob", http.MethodGet, subIssues(shelved), "", http.StatusOK)); !reflect.DeepEqual(got,
		[]countedIssue{read}) || read.ArchivedAt != nil || !reflect.DeepEqual(read.ParentID, &shelved) {
		t.Errorf("Shelved, archived, lists %+v; want Kept alone, unarchived, under it, as getIssue answers it: %+v", got, read)
	}
}

// The guest rule holds for sub-items (M4 design 3.4, 3.9): walt, acme's
// admin and Web's guest (adminGuest), sees only his own work items in Web.
// He makes His and Mine, alice Hers between them; under Mine, alice makes
// Low and walt Below. walt puts Mine under His, answered with the one
// sub-item he sees; Hers, which he does not see, is refused at its place
// with the very answer of a work item not there, and stays under none;
// alice puts it there. walt lists Mine alone under His, with its one
// sub-item, and alice and bob Hers and Mine, with its two; Up, alice's,
// walt finds neither to list nor to put a work item under: the very 404
// of a work item not there.
func TestTheGuestRuleHoldsForSubItems(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	up := w.newWorkItemIn(t, w.web, `{"name":"Up"}`)
	his := w.newWorkItemBy(t, "walt", w.web, `{"name":"His"}`)
	hers := w.newWorkItemIn(t, w.web, `{"name":"Hers"}`)
	mine := w.newWorkItemBy(t, "walt", w.web, `{"name":"Mine"}`)
	w.newWorkItemIn(t, w.web, underParent("Low", mine))
	w.newWorkItemBy(t, "walt", w.web, underParent("Below", mine))
	added := listIn[countedIssue](t, w.send(t, "walt", http.MethodPost, subIssues(his), subIssueIDs(mine), http.StatusOK))
	if got := counts(added); !slices.Equal(got, []string{"Mine 1"}) {
		t.Errorf("walt's putting Mine under His answers %q, want Mine with the one sub-item he sees", got)
	}
	answer := w.send(t, "walt", http.MethodPost, subIssues(his), subIssueIDs(hers), http.StatusUnprocessableEntity)
	notThere := w.send(t, "walt", http.MethodPost, subIssues(his), subIssueIDs(uuid.NewV7()), http.StatusUnprocessableEntity)
	if got := refusalOf(t, []byte(answer), false); got != "validation_failed sub_issue_ids[0] not_allowed" || answer != notThere ||
		w.parentOf(t, hers) != nil {
		t.Errorf("walt's putting Hers under His answers %s, and Hers is under %v; want sub_issue_ids[0] not_allowed, as for one not there: %s",
			answer, w.parentOf(t, hers), notThere)
	}
	w.send(t, "alice", http.MethodPost, subIssues(his), subIssueIDs(hers), http.StatusOK)
	for _, tt := range []struct {
		name string
		want []string
	}{{"walt", []string{"Mine 1"}}, {"alice", []string{"Hers 0", "Mine 2"}}, {"bob", []string{"Hers 0", "Mine 2"}}} {
		listed := listIn[countedIssue](t, w.send(t, tt.name, http.MethodGet, subIssues(his), "", http.StatusOK))
		if got := counts(listed); !slices.Equal(got, tt.want) {
			t.Errorf("%s lists %q under His, want %q", tt.name, got, tt.want)
		}
	}
	for _, tt := range []struct{ method, body string }{{http.MethodGet, ""}, {http.MethodPost, subIssueIDs(mine)}} {
		hidden := w.send(t, "walt", tt.method, subIssues(up), tt.body, http.StatusNotFound)
		if absent := w.send(t, "walt", tt.method, subIssues(uuid.NewV7()), tt.body, http.StatusNotFound); hidden != absent {
			t.Errorf("walt's %s of Up's sub-items answers %s, want as for a work item not there: %s", tt.method, hidden, absent)
		}
	}
}
