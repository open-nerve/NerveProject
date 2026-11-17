package domain

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The events notified of are the fourteen of spec 2.3's table, the issue
// module's and the asset module's attachments.
func TestEventTypes(t *testing.T) {
	want := []string{"issue.created", "issue.updated", "issue.archived", "issue.unarchived",
		"issue.comment_created", "issue.comment_updated", "issue.comment_deleted",
		"issue.link_created", "issue.link_updated", "issue.link_deleted",
		"issue.relation_added", "issue.relation_removed", "asset.attachment_added", "asset.attachment_deleted"}
	if got := EventTypes(); !reflect.DeepEqual(got, want) {
		t.Errorf("EventTypes() = %v, want %v", got, want)
	}
}

// Each event notifies of spec 2.3's changes, each at its position in the
// event's changes: a creation, of itself at -1 and of each assignee with his
// display name and id; an update, of each change of its eight fields, as
// updated, the parent by its ids only (2.5), not of its description, cycle,
// modules and sorting; archiving and unarchiving, of archived_at; a
// comment's write, of its comment by id, without its HTML; a link's, of its
// address only; a relation's, on the end it was requested on, of the type
// that end reads, by the other end's id; an attachment's, of the
// attachment. The mention changes of a creation, an update and a comment's
// creation and update each notify their account, of the description or the
// comment, at their own positions. Nothing else notifies.
func TestNotifiableOf(t *testing.T) {
	updatedItem := func(seq int, field string, oldValue, newValue *string, oldID, newID *uuid.UUID) Item {
		return Item{Seq: seq, Field: field, Verb: VerbUpdated, OldValue: oldValue, NewValue: newValue, OldID: oldID, NewID: newID}
	}
	mention := func(account uuid.UUID, seq int, field string, commentID *uuid.UUID) Mention {
		return Mention{Account: account, Item: Item{Seq: seq, Field: field, Verb: VerbMentioned, CommentID: commentID}}
	}
	tests := map[string]Notifiable{
		"created": {
			Changes: []Item{{Seq: -1, Field: "issue", Verb: VerbCreated},
				{Seq: 10, Field: "assignees", Verb: VerbCreated, NewValue: text("Bob"), NewID: ref(bob)},
				{Seq: 11, Field: "assignees", Verb: VerbCreated, NewValue: text("Carol"), NewID: ref(carol)}},
			Mentions: []Mention{mention(carol, 15, "description", nil), mention(dave, 16, "description", nil)},
		},
		"created alone": {Changes: []Item{{Seq: -1, Field: "issue", Verb: VerbCreated}}},
		"updated": {
			Changes: []Item{
				updatedItem(0, "name", text("Fix it"), text("Fix it now"), nil, nil),
				updatedItem(2, "parent", nil, nil, ref(parent), ref(newParent)),
				updatedItem(3, "priority", text("high"), text("urgent"), nil, nil),
				updatedItem(4, "state", text("Todo"), text("Doing"), ref(todo), ref(doing)),
				updatedItem(5, "start_date", text(""), text("2026-11-20"), nil, nil),
				updatedItem(6, "target_date", text("2026-12-01"), text(""), nil, nil),
				updatedItem(7, "labels", text(""), text("bug"), nil, ref(label)),
				updatedItem(8, "labels", text("ui"), text(""), ref(oldLabel), nil),
				updatedItem(9, "assignees", text(""), text("Dave"), nil, ref(dave)),
				updatedItem(10, "assignees", text("Bob"), text(""), ref(bob), nil),
			},
			Mentions: []Mention{mention(eve, 14, "description", nil)},
		},
		"updated parent":       {Changes: []Item{updatedItem(0, "parent", nil, nil, ref(parent), nil)}},
		"updated silently":     {},
		"updated mention only": {Mentions: []Mention{mention(bob, 1, "description", nil)}},
		"archived":             {Changes: []Item{updatedItem(0, "archived_at", nil, text("manual_archive"), nil, nil)}},
		"auto archived":        {Changes: []Item{updatedItem(0, "archived_at", nil, text("archive"), nil, nil)}},
		"unarchived":           {Changes: []Item{updatedItem(0, "archived_at", text("archive"), text("restore"), nil, nil)}},
		"comment created": {
			Changes:  []Item{{Seq: 0, Field: "comment", Verb: VerbCreated, CommentID: ref(comment)}},
			Mentions: []Mention{mention(bob, 1, "comment", ref(comment)), mention(carol, 2, "comment", ref(comment))},
		},
		"comment updated": {
			Changes:  []Item{{Seq: 0, Field: "comment", Verb: VerbUpdated, CommentID: ref(comment)}},
			Mentions: []Mention{mention(dave, 1, "comment", ref(comment))},
		},
		"comment deleted": {Changes: []Item{{Seq: 0, Field: "comment", Verb: VerbDeleted, CommentID: ref(comment)}}},
		"link created":    {Changes: []Item{{Seq: 0, Field: "link", Verb: VerbCreated, NewValue: text("https://a.example")}}},
		"link updated": {Changes: []Item{
			{Seq: 0, Field: "link", Verb: VerbUpdated, OldValue: text("https://a.example"), NewValue: text("https://b.example")}}},
		"link retitled":      {},
		"link deleted":       {Changes: []Item{{Seq: 0, Field: "link", Verb: VerbDeleted, OldValue: text("https://b.example")}}},
		"relation blocking":  {Changes: []Item{{Seq: 3, Field: "blocking", Verb: VerbCreated, NewID: ref(other)}}},
		"relation relates":   {Changes: []Item{{Seq: 3, Field: "relates_to", Verb: VerbCreated, NewID: ref(other)}}},
		"relation removed":   {Changes: []Item{{Seq: 3, Field: "blocked_by", Verb: VerbDeleted, OldID: ref(other)}}},
		"attachment added":   {Changes: []Item{{Seq: 0, Field: "attachment", Verb: VerbCreated}}},
		"attachment deleted": {Changes: []Item{{Seq: 0, Field: "attachment", Verb: VerbDeleted}}},
		"deleted":            {},
		"reaction added":     {},
	}
	all := events()
	if len(tests) != len(all) {
		t.Fatalf("%d cases for %d events", len(tests), len(all))
	}
	for name, want := range tests {
		got := NotifiableOf(all[name])
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: NotifiableOf() = %s, want %s", name, show(got), show(want))
		}
		if e := all[name]; got.AnyBut(e.ActorID) != (len(want.Changes)+len(want.Mentions) > 0) {
			t.Errorf("%s: AnyBut(its actor) = %v", name, got.AnyBut(e.ActorID))
		}
	}
}

// A write notifies anyone but its actor of a change, which the work item's
// subscribers receive, or of a mention of another account than its actor:
// a write whose only notifiable item is its actor's mention of himself
// notifies no one (spec 2.9; Recipients leaves the actor out), nerve's own
// write any mention.
func TestAnyBut(t *testing.T) {
	mentionOf := func(account uuid.UUID) Mention {
		return Mention{Account: account, Item: Item{Seq: 1, Field: "description", Verb: VerbMentioned}}
	}
	change := Item{Seq: 0, Field: "name", Verb: VerbUpdated}
	for _, tt := range []struct {
		name  string
		n     Notifiable
		actor *uuid.UUID
		want  bool
	}{
		{"nothing", Notifiable{}, &alice, false},
		{"nothing by nerve", Notifiable{}, nil, false},
		{"the actor mentioning himself", Notifiable{Mentions: []Mention{mentionOf(alice)}}, &alice, false},
		{"the actor mentioning himself twice", Notifiable{Mentions: []Mention{mentionOf(alice), mentionOf(alice)}}, &alice, false},
		{"the actor mentioning another", Notifiable{Mentions: []Mention{mentionOf(bob)}}, &alice, true},
		{"the actor mentioning himself and another", Notifiable{Mentions: []Mention{mentionOf(alice), mentionOf(bob)}}, &alice, true},
		{"nerve mentioning an account", Notifiable{Mentions: []Mention{mentionOf(alice)}}, nil, true},
		{"a change", Notifiable{Changes: []Item{change}}, &alice, true},
		{"a change and the actor's mention of himself", Notifiable{Changes: []Item{change}, Mentions: []Mention{mentionOf(alice)}}, &alice, true},
		{"a change by nerve", Notifiable{Changes: []Item{change}}, nil, true},
	} {
		if got := tt.n.AnyBut(tt.actor); got != tt.want {
			t.Errorf("%s: AnyBut() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// An event is about the work item spec 2.9's job locks: an issue event's
// entity; a comment's, a link's or an attachment's, the issue change's id,
// the Old one at a deletion; a relation's, the end it was requested on, not
// the stored issue end. An event notified of without one is about none.
func TestWorkItem(t *testing.T) {
	all := events()
	for _, kind := range []string{"created", "updated", "archived", "auto archived", "unarchived", "comment created", "comment updated",
		"comment deleted", "link created", "link retitled", "link deleted", "relation blocking", "relation relates", "relation removed",
		"attachment added", "attachment deleted"} {
		if got, ok := WorkItem(all[kind]); !ok || got != workItem {
			t.Errorf("%s: WorkItem() = %s, %v, want the work item", kind, name(got), ok)
		}
	}
	noIssue := all["comment created"]
	noIssue.Changes = noIssue.Changes[:3]
	none := map[string]shared.Event{"deleted": all["deleted"], "reaction added": all["reaction added"], "comment without its issue": noIssue}
	for kind, e := range none {
		if got, ok := WorkItem(e); ok {
			t.Errorf("%s: WorkItem() = %s, want none", kind, name(got))
		}
	}
}

// A relation's event names its stored row's project, its issue end's,
// while its work item is the end it was requested on (R11), which is of
// another project when a blocking relation across projects is asked on its
// blocking end: it may name another project than its work item's. Every
// other event names its work item's.
func TestMayNameAnotherProject(t *testing.T) {
	for kind, e := range events() {
		want := strings.HasPrefix(kind, "relation ")
		if got := MayNameAnotherProject(e); got != want {
			t.Errorf("%s: MayNameAnotherProject() = %v, want %v", kind, got, want)
		}
	}
}

// The job's arguments keep every change at its position, the HTML of a
// description and a comment blanked (R4), and the event's changes as they
// are: what they notify of is what the event does, at the same positions.
func TestWithoutHTML(t *testing.T) {
	for name, e := range events() {
		before := slices.Clone(e.Changes)
		got := WithoutHTML(e.Changes)
		if !reflect.DeepEqual(e.Changes, before) {
			t.Errorf("%s: WithoutHTML() changed the event's changes", name)
		}
		if len(got) != len(e.Changes) {
			t.Errorf("%s: WithoutHTML() kept %d of %d changes", name, len(got), len(e.Changes))
		} else {
			for n, c := range got {
				want := e.Changes[n]
				if want.Field == "description" || want.Field == "comment_html" {
					want.Old, want.New = nil, nil
				}
				if !reflect.DeepEqual(c, want) {
					t.Errorf("%s: change %d = %s, want %s", name, n, showChange(c), showChange(want))
				}
			}
		}
		args := e
		args.Changes = got
		if a, b := NotifiableOf(args), NotifiableOf(e); !reflect.DeepEqual(a, b) {
			t.Errorf("%s: NotifiableOf() over the arguments = %s, over the event %s", name, show(a), show(b))
		}
	}
}
