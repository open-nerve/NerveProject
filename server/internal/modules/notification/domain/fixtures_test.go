package domain

import (
	"fmt"
	"strconv"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The accounts and rows the tests' events name.
var (
	alice, bob, carol, dave, eve                    = uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	workItem, other, parent, newParent              = uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	todo, doing, label, oldLabel, cycle, module     = uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	comment, link, relation, attachment             = uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	descriptionHTML, commentHTML, editedCommentHTML = `<p>hi <mention-component entity_identifier="x"></mention-component></p>`,
		"<p>look</p>", "<p>look again</p>"
)

func text(s string) *string       { return &s }
func ref(id uuid.UUID) *uuid.UUID { return &id }

// event is an event of kind by alice on the entity of kind entity and id
// entityID, with changes, as the issue and the asset modules publish it.
func event(kind, entity string, entityID uuid.UUID, changes ...shared.Change) shared.Event {
	return shared.Event{ID: uuid.NewV7(), Type: kind, OccurredAt: time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC),
		WorkspaceID: uuid.NewV7(), ProjectID: uuid.NewV7(), ActorID: &alice, Entity: shared.EntityRef{Kind: entity, ID: entityID},
		Changes: changes}
}

// created is issue.created as createdChanges builds it: the ten fields,
// then two assignees, a label, the cycle, a module and two mentions.
func created() shared.Event {
	return event("issue.created", "issue", workItem,
		shared.Change{Field: "name", New: text("Fix it")},
		shared.Change{Field: "description", New: text(descriptionHTML)},
		shared.Change{Field: "state", New: text("Todo"), NewID: ref(todo)},
		shared.Change{Field: "priority", New: text("high")},
		shared.Change{Field: "start_date"},
		shared.Change{Field: "target_date", New: text("2026-12-01")},
		shared.Change{Field: "parent", New: text("PROJ-1"), NewID: ref(parent)},
		shared.Change{Field: "sort_order", New: text("65535")},
		shared.Change{Field: "sequence_id", New: text("12")},
		shared.Change{Field: "completed_at"},
		shared.Change{Field: "assignees", New: text("Bob"), NewID: ref(bob)},     // 10
		shared.Change{Field: "assignees", New: text("Carol"), NewID: ref(carol)}, // 11
		shared.Change{Field: "labels", New: text("bug"), NewID: ref(label)},
		shared.Change{Field: "cycles", New: text("Sprint 1"), NewID: ref(cycle)},
		shared.Change{Field: "modules", New: text("Core"), NewID: ref(module)},
		shared.Change{Field: "mention", NewID: ref(carol)}, // 15
		shared.Change{Field: "mention", NewID: ref(dave)},  // 16
	)
}

// updated is issue.updated as updatedChanges builds it from every row
// UpdateActivity writes, in its order, then sort_order and a mention.
func updated() shared.Event {
	return event("issue.updated", "issue", workItem,
		shared.Change{Field: "name", Old: text("Fix it"), New: text("Fix it now")},
		shared.Change{Field: "description", Old: text(descriptionHTML), New: text("<p>more</p>")},
		shared.Change{Field: "parent", Old: text("PROJ-1"), New: text("PROJ-2"), OldID: ref(parent), NewID: ref(newParent)}, // 2
		shared.Change{Field: "priority", Old: text("high"), New: text("urgent")},
		shared.Change{Field: "state", Old: text("Todo"), New: text("Doing"), OldID: ref(todo), NewID: ref(doing)},
		shared.Change{Field: "start_date", Old: text(""), New: text("2026-11-20")},
		shared.Change{Field: "target_date", Old: text("2026-12-01"), New: text("")},
		shared.Change{Field: "labels", Old: text(""), New: text("bug"), NewID: ref(label)},
		shared.Change{Field: "labels", Old: text("ui"), New: text(""), OldID: ref(oldLabel)},
		shared.Change{Field: "assignees", Old: text(""), New: text("Dave"), NewID: ref(dave)},
		shared.Change{Field: "assignees", Old: text("Bob"), New: text(""), OldID: ref(bob)}, // 10
		shared.Change{Field: "cycles", Old: text(""), New: text("Sprint 1"), NewID: ref(cycle)},
		shared.Change{Field: "modules", Old: text(""), New: text("Core"), NewID: ref(module)},
		shared.Change{Field: "sort_order", Old: text("1"), New: text("2")},
		shared.Change{Field: "mention", NewID: ref(eve)}, // 14
	)
}

// The relation events as relationEvent builds them: the stored row, issue
// then related_issue, then the end it was requested on, with the type that
// end reads; all on the New side for an addition, the Old for a removal.

// blockingAdded is relation_added asked on workItem as blocking other,
// stored the other way round, as other's blocked_by.
func blockingAdded() shared.Event {
	return event("issue.relation_added", "issue_relation", relation,
		shared.Change{Field: "relation_type", New: text("blocked_by")},
		shared.Change{Field: "issue", NewID: ref(other)},
		shared.Change{Field: "related_issue", NewID: ref(workItem)},
		shared.Change{Field: "requested_on", New: text("blocking"), NewID: ref(workItem)},
	)
}

// relatesAdded is relation_added asked on workItem, stored from it.
func relatesAdded() shared.Event {
	return event("issue.relation_added", "issue_relation", relation,
		shared.Change{Field: "relation_type", New: text("relates_to")},
		shared.Change{Field: "issue", NewID: ref(workItem)},
		shared.Change{Field: "related_issue", NewID: ref(other)},
		shared.Change{Field: "requested_on", New: text("relates_to"), NewID: ref(workItem)},
	)
}

// blockedByRemoved is relation_removed of a relation stored from workItem
// as other's blocking: requested on its issue_id end, workItem, which reads
// it as blocked_by.
func blockedByRemoved() shared.Event {
	return event("issue.relation_removed", "issue_relation", relation,
		shared.Change{Field: "relation_type", Old: text("blocked_by")},
		shared.Change{Field: "issue", OldID: ref(workItem)},
		shared.Change{Field: "related_issue", OldID: ref(other)},
		shared.Change{Field: "requested_on", Old: text("blocked_by"), OldID: ref(workItem)},
	)
}

// change is a change of the tests' events.
type change = shared.Change

// events are one event of each kind the tests read, by name.
func events() map[string]shared.Event {
	auto := event("issue.archived", "issue", workItem, change{Field: "archived_at", New: text("archive")})
	auto.ActorID = nil
	toIssue, fromIssue := change{Field: "issue", NewID: ref(workItem)}, change{Field: "issue", OldID: ref(workItem)}
	return map[string]shared.Event{
		"created": created(),
		"created alone": event("issue.created", "issue", workItem,
			change{Field: "name", New: text("Fix it")}, change{Field: "description"}),
		"updated": updated(),
		"updated parent": event("issue.updated", "issue", workItem,
			change{Field: "parent", Old: text("PROJ-1"), New: text(""), OldID: ref(parent)}),
		"updated silently": event("issue.updated", "issue", workItem,
			change{Field: "description", Old: text("<p>a</p>"), New: text("<p>b</p>")},
			change{Field: "cycles", Old: text("Sprint 1"), New: text(""), OldID: ref(cycle)},
			change{Field: "modules", Old: text("Core"), New: text(""), OldID: ref(module)},
			change{Field: "sort_order", Old: text("1"), New: text("2")}),
		"updated mention only": event("issue.updated", "issue", workItem,
			change{Field: "description", Old: text("<p>a</p>"), New: text(descriptionHTML)}, change{Field: "mention", NewID: ref(bob)}),
		"archived":      event("issue.archived", "issue", workItem, change{Field: "archived_at", New: text("manual_archive")}),
		"auto archived": auto,
		"unarchived":    event("issue.unarchived", "issue", workItem, change{Field: "archived_at", Old: text("archive"), New: text("restore")}),
		"comment created": event("issue.comment_created", "issue_comment", comment,
			change{Field: "comment_html", New: text(commentHTML)}, change{Field: "mention", NewID: ref(bob)},
			change{Field: "mention", NewID: ref(carol)}, toIssue),
		"comment updated": event("issue.comment_updated", "issue_comment", comment,
			change{Field: "comment_html", Old: text(commentHTML), New: text(editedCommentHTML)}, change{Field: "mention", NewID: ref(dave)},
			toIssue),
		"comment deleted": event("issue.comment_deleted", "issue_comment", comment,
			change{Field: "comment_html", Old: text(commentHTML)}, fromIssue),
		"link created": event("issue.link_created", "issue_link", link,
			change{Field: "url", New: text("https://a.example")}, change{Field: "title", New: text("A")}, toIssue),
		"link updated": event("issue.link_updated", "issue_link", link,
			change{Field: "url", Old: text("https://a.example"), New: text("https://b.example")},
			change{Field: "title", Old: text("A"), New: text("B")}, toIssue),
		"link retitled": event("issue.link_updated", "issue_link", link, change{Field: "title", Old: text("A"), New: text("B")}, toIssue),
		"link deleted": event("issue.link_deleted", "issue_link", link,
			change{Field: "url", Old: text("https://b.example")}, change{Field: "title", Old: text("B")}, fromIssue),
		"relation blocking":  blockingAdded(),
		"relation relates":   relatesAdded(),
		"relation removed":   blockedByRemoved(),
		"attachment added":   event("asset.attachment_added", "asset", attachment, toIssue),
		"attachment deleted": event("asset.attachment_deleted", "asset", attachment, fromIssue),
		"deleted":            event("issue.deleted", "issue", workItem),
		"reaction added": event("issue.reaction_added", "issue_reaction", uuid.NewV7(),
			change{Field: "reaction", New: text("+1")}, toIssue),
	}
}

// name is the name the tests give id, or its text.
func name(id uuid.UUID) string {
	names := map[uuid.UUID]string{alice: "alice", bob: "bob", carol: "carol", dave: "dave", eve: "eve", workItem: "workItem", other: "other",
		parent: "parent", newParent: "newParent", todo: "todo", doing: "doing", label: "label", oldLabel: "oldLabel", cycle: "cycle",
		module: "module", comment: "comment", link: "link", relation: "relation", attachment: "attachment"}
	if n, ok := names[id]; ok {
		return n
	}
	return id.String()
}

// value and named write a value and an id, "-" for none.
func value(p *string) string {
	if p == nil {
		return "-"
	}
	return strconv.Quote(*p)
}

func named(p *uuid.UUID) string {
	if p == nil {
		return "-"
	}
	return name(*p)
}

// show, showItem, showChange and showRows write what the tests compare
// with its values and the names of its ids, not their addresses.
func show(n Notifiable) string {
	s := ""
	for _, i := range n.Changes {
		s += showItem(i) + " "
	}
	for _, m := range n.Mentions {
		s += name(m.Account) + ":" + showItem(m.Item) + " "
	}
	return "[" + s + "]"
}

func showItem(i Item) string {
	return fmt.Sprintf("{%d %s %s %s %s %s %s %s}", i.Seq, i.Field, i.Verb, value(i.OldValue), value(i.NewValue), named(i.OldID), named(i.NewID),
		named(i.CommentID))
}

func showChange(c shared.Change) string {
	return fmt.Sprintf("{%s %s %s %s %s}", c.Field, value(c.Old), value(c.New), named(c.OldID), named(c.NewID))
}

func showRows(rows []Row) string {
	s := ""
	for _, r := range rows {
		s += fmt.Sprintf("%s/%s%s ", name(r.Receiver), r.Sender, showItem(r.Item))
	}
	return "[" + s + "]"
}
