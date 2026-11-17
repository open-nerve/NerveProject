package bootstrap

import (
	"fmt"
	"net/http"
	"testing"
	"uuid"

	notificationriver "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/river"
	notificationdomain "github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// mentionCase is a write of alice's on a work item she made and watchers subscribed to through the API, whose HTML
// names the accounts mentioned, in their order, repeats too; and what it sends: receivers their mention, each once at
// its position, and subscribed the write's change.
type mentionCase struct {
	name                                       string
	watchers, mentioned, receivers, subscribed []string
}

// mentionWrite is a mention case written in a form, a description's or a comment's, on its own work item; comment is
// the comment a comment's write made.
type mentionWrite struct {
	form    string
	c       mentionCase
	item    notificationdomain.IssueSnapshot
	comment uuid.UUID
}

// The mentions of a write through the API send what M7 P1b spec 2.3–2.4 say, read back from the table once the app's
// fan-outs have run, in each form: alice's change of a work item's name and description, whose mentions are of the
// description, after its name at 0 and its description at 1; and her comment on it, whose mentions are of the comment,
// with its id and its text's excerpt, after its HTML at 0. Each account mentioned is sent, as mentioned, the mention at
// the position of its own mention change in the event, not the first; a subscriber mentioned is sent his mention and
// not the write's change too, which a subscriber not mentioned is sent; an account mentioned twice in one write is sent
// one mention; alice, who wrote it, mentioning herself is sent nothing; wanda, a guest of Web who sees only the work
// items she created, mentioned on alice's, is sent nothing (spec 2.4's recheck).
func TestMentionsNotifyThroughTheApp(t *testing.T) {
	w := newIssueWorld(t)
	alice := w.ids["alice"]
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	membership := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["wanda"])[0]
	w.send(t, "alice", http.MethodPatch, "/api/v0/workspace-members/"+membership.String(), `{"role":15}`, http.StatusOK)
	w.makeMember(t, "wanda", w.web, shared.RoleGuest)
	cases := []mentionCase{
		{name: "each at its own position", mentioned: []string{"walt", "bob"}, receivers: []string{"walt", "bob"}},
		{name: "a subscriber once", watchers: []string{"bob", "walt"}, mentioned: []string{"bob"}, receivers: []string{"bob"},
			subscribed: []string{"walt"}},
		{name: "twice in one write once", mentioned: []string{"walt", "walt"}, receivers: []string{"walt"}},
		{name: "the actor nothing", mentioned: []string{"alice", "walt"}, receivers: []string{"walt"}},
		{name: "one who cannot read it nothing", mentioned: []string{"wanda", "walt"}, receivers: []string{"walt"}},
	}
	todo := webState{w.todo, "Todo", "unstarted"}
	var writes []mentionWrite
	for _, form := range []string{"description", "comment"} {
		for _, c := range cases {
			writes = append(writes, mentionWrite{form: form, c: c, item: w.itemMade(t, "alice", "Fix", todo)})
		}
	}
	// A fan-out reads the work item's subscribers when it runs: the creations' have run before anyone else subscribes.
	waitForFanouts(t, w.pool)
	for n, m := range writes {
		path := "/api/v0/issues/" + m.item.ID.String()
		mentioned := make([]uuid.UUID, len(m.c.mentioned))
		for i, name := range m.c.mentioned {
			mentioned[i] = w.ids[name]
		}
		for _, name := range m.c.watchers {
			w.send(t, name, http.MethodPost, path+"/subscribe", "", http.StatusNoContent)
		}
		if m.form == "description" {
			w.send(t, "alice", http.MethodPatch, path, fmt.Sprintf(`{"name":"Fix it","description_html":%q}`, mentionHTML(mentioned...)), http.StatusOK)
			continue
		}
		var c answeredComment
		decodeAnswer(t, w.send(t, "alice", http.MethodPost, commentsOf(m.item.ID), fmt.Sprintf(`{"comment_html":%q}`, mentionHTML(mentioned...)),
			http.StatusCreated), &c)
		writes[n].comment = c.ID
	}
	waitForFanouts(t, w.pool)

	hello := "Hello" // mentionHTML's text: a mention shows nothing in it (R6)
	for _, m := range writes {
		t.Run(m.form+"/"+m.c.name, func(t *testing.T) {
			var job notificationriver.FanoutArgs
			after := m.item
			mention := notificationdomain.ChangeSnapshot{Field: "description", Verb: "mentioned"}
			change := changed("name", eventValue("Fix"), eventValue("Fix it"), nil, nil)
			if m.form == "description" {
				job, after.Name = fanoutOf(t, w.pool, "issue.updated", m.item.ID), "Fix it"
			} else {
				job = fanoutOf(t, w.pool, "issue.comment_created", m.comment)
				mention = notificationdomain.ChangeSnapshot{Field: "comment", Verb: "mentioned", CommentID: &m.comment, Excerpt: &hello}
				change = notificationdomain.ChangeSnapshot{Field: "comment", Verb: "created", CommentID: &m.comment, Excerpt: &hello}
			}
			var sends []sending
			for _, name := range m.c.receivers {
				sends = append(sends, sending{w.ids[name], "mentioned", mentionedAt(t, job, w.ids[name]), mention})
			}
			for _, name := range m.c.subscribed {
				sends = append(sends, sending{w.ids[name], "subscribed", 0, change})
			}
			assertSent(t, w.pool, m.item.ID, []uuid.UUID{job.EventID}, w.sentOf(job, &alice, &alice, after, sends...))
		})
	}
}
