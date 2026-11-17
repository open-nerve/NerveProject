// Package domain holds the notification module's rules (M7 P1b spec
// 2.3–2.5): which changes of an event notify, with which field and verb, at
// which position; who receives each, and as what; and the snapshot each
// notification keeps. It reads the issue and the asset modules' events by
// their published names and shapes, and imports neither.
package domain

import (
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The events a notification is sent for (spec 2.3): the issue module's and
// the asset module's attachments'.
const (
	issueCreated      = "issue.created"
	issueUpdated      = "issue.updated"
	issueArchived     = "issue.archived"
	issueUnarchived   = "issue.unarchived"
	commentCreated    = "issue.comment_created"
	commentUpdated    = "issue.comment_updated"
	commentDeleted    = "issue.comment_deleted"
	linkCreated       = "issue.link_created"
	linkUpdated       = "issue.link_updated"
	linkDeleted       = "issue.link_deleted"
	relationAdded     = "issue.relation_added"
	relationRemoved   = "issue.relation_removed"
	attachmentAdded   = "asset.attachment_added"
	attachmentDeleted = "asset.attachment_deleted"
)

// EventTypes are the events the module notifies of, those its subscriber
// reads (spec 2.9).
func EventTypes() []string {
	return []string{issueCreated, issueUpdated, issueArchived, issueUnarchived, commentCreated, commentUpdated, commentDeleted,
		linkCreated, linkUpdated, linkDeleted, relationAdded, relationRemoved, attachmentAdded, attachmentDeleted}
}

// Verb is what a notification says was done (spec 2.3): the activity's
// verbs, and mentioned.
type Verb string

// The verbs of notifications.
const (
	VerbCreated   Verb = "created"
	VerbUpdated   Verb = "updated"
	VerbDeleted   Verb = "deleted"
	VerbMentioned Verb = "mentioned"
)

// CreationSeq is the position of a work item's creation itself, which is
// none of its event's changes (R4).
const CreationSeq = -1

// Item is a change an event notifies of (spec 2.3): its position in the
// event's changes, CreationSeq for the creation itself; the field and the
// verb its notifications name; the values and the ids they keep, each nil
// for none; and the comment it is of, nil for none.
type Item struct {
	Seq                int
	Field              string
	Verb               Verb
	OldValue, NewValue *string
	OldID, NewID       *uuid.UUID
	CommentID          *uuid.UUID
}

// Mention is the notification of an account an event newly mentions: the
// account, and the item of the mention change, its own position.
type Mention struct {
	Account uuid.UUID
	Item    Item
}

// Notifiable is what an event notifies of: Changes, each of the work item's
// subscribers of each; Mentions, each its account.
type Notifiable struct {
	Changes  []Item
	Mentions []Mention
}

// AnyBut tells whether there is anything to notify anyone but actor, nil
// for nerve's own write, of: a change, which the work item's subscribers are
// notified of, or a mention of another account than actor. The subscriber
// enqueues the job only then (spec 2.9): a write whose only notifiable item
// is its actor's mention of himself notifies no one (Recipients leaves the
// actor out).
func (n Notifiable) AnyBut(actor *uuid.UUID) bool {
	if len(n.Changes) > 0 {
		return true
	}
	for _, m := range n.Mentions {
		if actor == nil || m.Account != *actor {
			return true
		}
	}
	return false
}

// notified is how an event of a type notifies (spec 2.3): the verb of its
// changes; the field its mentions are of, "" when it mentions none; and the
// item of a change at the position seq, if that change notifies.
type notified struct {
	verb    Verb
	mention string
	item    func(e shared.Event, seq int, verb Verb) (Item, bool)
}

// notifiedBy is how each event of EventTypes notifies, spec 2.3's table.
var notifiedBy = map[string]notified{
	issueCreated:      {VerbCreated, "description", assigneeItem},
	issueUpdated:      {VerbUpdated, "description", updatedItem},
	issueArchived:     {VerbUpdated, "", archivedItem},
	issueUnarchived:   {VerbUpdated, "", archivedItem},
	commentCreated:    {VerbCreated, "comment", commentItem},
	commentUpdated:    {VerbUpdated, "comment", commentItem},
	commentDeleted:    {VerbDeleted, "", commentItem},
	linkCreated:       {VerbCreated, "", linkItem},
	linkUpdated:       {VerbUpdated, "", linkItem},
	linkDeleted:       {VerbDeleted, "", linkItem},
	relationAdded:     {VerbCreated, "", relationItem},
	relationRemoved:   {VerbDeleted, "", relationItem},
	attachmentAdded:   {VerbCreated, "", attachmentItem},
	attachmentDeleted: {VerbDeleted, "", attachmentItem},
}

// NotifiableOf is what the event e notifies of (spec 2.3), from its type,
// its entity and its changes, each at its position in them, the creation of
// a work item first, at CreationSeq. An event of another type notifies of
// nothing.
func NotifiableOf(e shared.Event) Notifiable {
	rule, ok := notifiedBy[e.Type]
	if !ok {
		return Notifiable{}
	}
	var n Notifiable
	if e.Type == issueCreated {
		n.Changes = append(n.Changes, Item{Seq: CreationSeq, Field: "issue", Verb: VerbCreated})
	}
	for seq, c := range e.Changes {
		if c.Field == "mention" {
			if rule.mention != "" && c.NewID != nil {
				n.Mentions = append(n.Mentions, Mention{Account: *c.NewID,
					Item: Item{Seq: seq, Field: rule.mention, Verb: VerbMentioned, CommentID: commentOf(e)}})
			}
			continue
		}
		if item, ok := rule.item(e, seq, rule.verb); ok {
			n.Changes = append(n.Changes, item)
		}
	}
	return n
}

// asItem is the change at seq of e as an item of verb, with its values and
// ids.
func asItem(e shared.Event, seq int, verb Verb) Item {
	c := e.Changes[seq]
	return Item{Seq: seq, Field: c.Field, Verb: verb, OldValue: c.Old, NewValue: c.New, OldID: c.OldID, NewID: c.NewID}
}

// assigneeItem notifies, of a creation, each account it is assigned to.
func assigneeItem(e shared.Event, seq int, verb Verb) (Item, bool) {
	return asItem(e, seq, verb), e.Changes[seq].Field == "assignees"
}

// updatedFields are the fields whose changes an update notifies of; not its
// description (only its mentions), cycle, modules or sorting.
var updatedFields = map[string]bool{"name": true, "parent": true, "priority": true, "state": true, "start_date": true,
	"target_date": true, "labels": true, "assignees": true}

// updatedItem notifies, of an update, each change of updatedFields; a
// parent's by its ids only, its identifiers being resolved when read for
// the reader (spec 2.5).
func updatedItem(e shared.Event, seq int, verb Verb) (Item, bool) {
	item := asItem(e, seq, verb)
	if item.Field == "parent" {
		item.OldValue, item.NewValue = nil, nil
	}
	return item, updatedFields[item.Field]
}

// archivedItem notifies, of archiving and unarchiving, archived_at.
func archivedItem(e shared.Event, seq int, verb Verb) (Item, bool) {
	return asItem(e, seq, verb), e.Changes[seq].Field == "archived_at"
}

// commentItem notifies, of a comment's write, the comment, by its id, the
// event's entity: never its HTML.
func commentItem(e shared.Event, seq int, verb Verb) (Item, bool) {
	return Item{Seq: seq, Field: "comment", Verb: verb, CommentID: commentOf(e)}, e.Changes[seq].Field == "comment_html"
}

// commentOf is the comment e is of, its entity, nil unless it is a
// comment's event.
func commentOf(e shared.Event) *uuid.UUID {
	switch e.Type {
	case commentCreated, commentUpdated, commentDeleted:
		id := e.Entity.ID
		return &id
	}
	return nil
}

// linkItem notifies, of a link's write, its address, with its values: a
// change of its title alone notifies of nothing (the activity the same).
func linkItem(e shared.Event, seq int, verb Verb) (Item, bool) {
	c := e.Changes[seq]
	return Item{Seq: seq, Field: "link", Verb: verb, OldValue: c.Old, NewValue: c.New}, c.Field == "url"
}

// relationItem notifies, of a relation's write, on the end it was requested
// on, requested_on's: of the type that end reads, as the field, and of the
// other end by its id, the one of issue and related_issue that is not
// requested_on's (spec 2.3, R11).
func relationItem(e shared.Event, seq int, verb Verb) (Item, bool) {
	c := e.Changes[seq]
	at, typ := idOf(c), valueOf(c)
	if c.Field != "requested_on" || at == nil || typ == nil {
		return Item{}, false
	}
	for _, end := range e.Changes {
		if id := idOf(end); (end.Field == "issue" || end.Field == "related_issue") && id != nil && *id != *at {
			item := Item{Seq: seq, Field: *typ, Verb: verb}
			if verb == VerbDeleted {
				item.OldID = id
			} else {
				item.NewID = id
			}
			return item, true
		}
	}
	return Item{}, false
}

// attachmentItem notifies, of an attachment's write, the attachment.
func attachmentItem(e shared.Event, seq int, verb Verb) (Item, bool) {
	return Item{Seq: seq, Field: "attachment", Verb: verb}, e.Changes[seq].Field == "issue"
}

// idOf and valueOf are the id and the value c names, its New one, else its
// Old one, nil for none.
func idOf(c shared.Change) *uuid.UUID {
	if c.NewID != nil {
		return c.NewID
	}
	return c.OldID
}

func valueOf(c shared.Change) *string {
	if c.New != nil {
		return c.New
	}
	return c.Old
}

// WorkItem is the work item the event e is about (spec 2.9), which its
// notifications are of: an issue event's entity; the work item a comment,
// a link or an attachment belongs to, its issue change's id, the Old one at
// a deletion; a relation's end it was requested on, requested_on's, not the
// stored issue end. It is none for an event the module does not notify of,
// or one that does not name it.
func WorkItem(e shared.Event) (uuid.UUID, bool) {
	field := "issue"
	switch e.Type {
	case issueCreated, issueUpdated, issueArchived, issueUnarchived:
		return e.Entity.ID, true
	case relationAdded, relationRemoved:
		field = "requested_on"
	default:
		if _, ok := notifiedBy[e.Type]; !ok {
			return uuid.Nil(), false
		}
	}
	for _, c := range e.Changes {
		if id := idOf(c); c.Field == field && id != nil {
			return *id, true
		}
	}
	return uuid.Nil(), false
}

// MayNameAnotherProject tells whether the event e may name another project
// than its work item's (WorkItem): a relation's names its stored row's,
// its issue end's, while its work item is the end it was requested on,
// which is of another project of the workspace when a relation across
// projects is stored from the other end (R11). Every other event names its
// work item's.
func MayNameAnotherProject(e shared.Event) bool {
	return e.Type == relationAdded || e.Type == relationRemoved
}

// WithoutHTML is changes as the job's arguments carry them (spec 2.9, R4):
// every change at its position, so the positions notifications are numbered
// with are the event's, the values of description and comment_html, the
// HTML of a description and a comment, blanked, as river_job keeps no
// HTML. changes is left as it is.
func WithoutHTML(changes []shared.Change) []shared.Change {
	kept := make([]shared.Change, 0, len(changes))
	for _, c := range changes {
		if c.Field == "description" || c.Field == "comment_html" {
			c.Old, c.New = nil, nil
		}
		kept = append(kept, c)
	}
	return kept
}
