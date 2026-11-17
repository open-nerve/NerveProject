package domain

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
	"uuid"
)

// The rows of an event before the recheck (spec 2.4): each mention its
// account's, as mentioned, but the actor's; each subscriber, but the actor
// and those the event mentions, one row for each change it notifies of,
// none when it notifies of none: as created for the work item's creator,
// else as assigned for one of its assignees, else as subscribed. Nerve's
// own write has no actor to leave out.
func TestRecipients(t *testing.T) {
	state := Item{Seq: 4, Field: "state", Verb: VerbUpdated}
	assignee := Item{Seq: 9, Field: "assignees", Verb: VerbUpdated}
	mentioned := func(seq int) Item { return Item{Seq: seq, Field: "description", Verb: VerbMentioned} }
	n := Notifiable{Changes: []Item{state, assignee},
		Mentions: []Mention{{Account: alice, Item: mentioned(14)}, {Account: dave, Item: mentioned(15)}}}
	issue := Issue{CreatedByID: &bob, AssigneeIDs: []uuid.UUID{carol, eve}}
	everyone := []uuid.UUID{alice, bob, carol, dave, eve}
	tests := []struct {
		name        string
		n           Notifiable
		actor       *uuid.UUID
		subscribers []uuid.UUID
		issue       Issue
		want        []Row
	}{
		{"by alice", n, &alice, everyone, issue, []Row{
			{dave, SenderMentioned, mentioned(15)},
			{bob, SenderCreated, state}, {bob, SenderCreated, assignee},
			{carol, SenderAssigned, state}, {carol, SenderAssigned, assignee},
			{eve, SenderAssigned, state}, {eve, SenderAssigned, assignee},
		}},
		{"by nerve", n, nil, everyone, issue, []Row{
			{alice, SenderMentioned, mentioned(14)}, {dave, SenderMentioned, mentioned(15)},
			{bob, SenderCreated, state}, {bob, SenderCreated, assignee},
			{carol, SenderAssigned, state}, {carol, SenderAssigned, assignee},
			{eve, SenderAssigned, state}, {eve, SenderAssigned, assignee},
		}},
		{"a creator assigned too", Notifiable{Changes: []Item{state}}, &dave, everyone,
			Issue{CreatedByID: &carol, AssigneeIDs: []uuid.UUID{carol, eve}}, []Row{
				{alice, SenderSubscribed, state}, {bob, SenderSubscribed, state}, {carol, SenderCreated, state}, {eve, SenderAssigned, state},
			}},
		{"mentioning the creator", Notifiable{Changes: []Item{state}, Mentions: []Mention{{Account: bob, Item: mentioned(5)}}}, &alice,
			everyone, issue, []Row{
				{bob, SenderMentioned, mentioned(5)},
				{carol, SenderAssigned, state}, {dave, SenderSubscribed, state}, {eve, SenderAssigned, state},
			}},
		{"of a creator not known", Notifiable{Changes: []Item{state}}, &alice, everyone, Issue{}, []Row{
			{bob, SenderSubscribed, state}, {carol, SenderSubscribed, state}, {dave, SenderSubscribed, state}, {eve, SenderSubscribed, state},
		}},
		{"mentions only", Notifiable{Mentions: n.Mentions}, &alice, everyone, issue, []Row{{dave, SenderMentioned, mentioned(15)}}},
		{"no subscriber", n, &alice, nil, issue, []Row{{dave, SenderMentioned, mentioned(15)}}},
		{"nothing", Notifiable{}, &alice, everyone, issue, nil},
	}
	for _, tt := range tests {
		if got := Recipients(tt.n, tt.actor, tt.subscribers, tt.issue); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: Recipients() = %s, want %s", tt.name, showRows(got), showRows(tt.want))
		}
	}
}

// The recheck decides on each receiver once, in id order.
func TestReceivers(t *testing.T) {
	item := Item{Seq: 0, Field: "name", Verb: VerbUpdated}
	rows := []Row{{eve, SenderMentioned, item}, {bob, SenderCreated, item}, {eve, SenderSubscribed, item}, {bob, SenderCreated, item}}
	want := slices.SortedFunc(slices.Values([]uuid.UUID{bob, eve}), uuid.UUID.Compare)
	if got := Receivers(rows); !reflect.DeepEqual(got, want) {
		t.Errorf("Receivers() = %v, want %v", got, want)
	}
	if got := Receivers(nil); len(got) != 0 {
		t.Errorf("Receivers(nil) = %v, want none", got)
	}
}

// The recheck keeps the rows of the receivers allowed to read the work
// item (spec 2.4) from whom the guest rule it is given does not hide it:
// the rule is asked of each allowed row, with the creator the work item
// names, none when it names none, the row's receiver and his grant's
// OwnOnly, and its answer is the recheck's, whatever the grant: Visible
// applies no rule of its own (M4 design 3.4). One refused, or not decided
// on, is not allowed, and the rule is not asked of him.
func TestVisible(t *testing.T) {
	item := Item{Seq: 0, Field: "name", Verb: VerbUpdated}
	rows := []Row{{alice, SenderSubscribed, item}, {bob, SenderCreated, item}, {carol, SenderAssigned, item},
		{dave, SenderMentioned, item}, {eve, SenderSubscribed, item}}
	decisions := map[uuid.UUID]Decision{alice: {Allowed: true}, bob: {Allowed: true, OwnOnly: true}, carol: {Allowed: true, OwnOnly: true},
		dave: {}}
	tests := []struct {
		name   string
		issue  Issue
		hidden []uuid.UUID
		want   []Row
	}{
		{"none hidden", Issue{CreatedByID: &bob}, nil, []Row{rows[0], rows[1], rows[2]}},
		{"one who sees his own hidden", Issue{CreatedByID: &bob}, []uuid.UUID{carol}, []Row{rows[0], rows[1]}},
		{"one of each grant hidden", Issue{}, []uuid.UUID{alice, bob}, []Row{rows[2]}},
		{"the refused and the undecided hidden too", Issue{CreatedByID: &eve}, []uuid.UUID{dave, eve}, []Row{rows[0], rows[1], rows[2]}},
	}
	for _, tt := range tests {
		var asked []string
		hiddenFrom := func(creator *uuid.UUID, account uuid.UUID, ownOnly bool) bool {
			asked = append(asked, fmt.Sprintf("%s %s %t", named(creator), name(account), ownOnly))
			return slices.Contains(tt.hidden, account)
		}
		creator := named(tt.issue.CreatedByID)
		wantAsked := []string{creator + " alice false", creator + " bob true", creator + " carol true"}
		if got := Visible(rows, decisions, tt.issue, hiddenFrom); !reflect.DeepEqual(got, tt.want) || !slices.Equal(asked, wantAsked) {
			t.Errorf("%s: Visible() = %s asking %q, want %s asking %q", tt.name, showRows(got), asked, showRows(tt.want), wantAsked)
		}
	}
}
