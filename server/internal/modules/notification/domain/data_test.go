package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

// A notification's data is spec 2.5's: the work item's id, name, number,
// project identifier and state, then the change's field and verb and, only
// when it has them, its values ("" is one), its ids, its comment and the
// excerpt of the comment's text: for a comment's creation, update and
// mention, not for its deletion, nor when the comment is not there.
func TestNewData(t *testing.T) {
	issue := IssueSnapshot{ID: workItem, Name: "Fix it", SequenceID: 12, Identifier: "PROJ", StateName: "In Progress", StateGroup: "started"}
	issueJSON := `{"id":"` + workItem.String() + `","name":"Fix it","sequence_id":12,"identifier":"PROJ","state_name":"In Progress",` +
		`"state_group":"started"}`
	c := `"comment_id":"` + comment.String() + `"`
	tests := []struct {
		name   string
		item   Item
		text   *string
		change string
	}{
		{"state", Item{Seq: 4, Field: "state", Verb: VerbUpdated, OldValue: text("Todo"), NewValue: text("Doing"), OldID: ref(todo),
			NewID: ref(doing)}, nil,
			`{"field":"state","verb":"updated","old_value":"Todo","new_value":"Doing","old_id":"` + todo.String() + `","new_id":"` + doing.String() + `"}`},
		{"creation", Item{Seq: -1, Field: "issue", Verb: VerbCreated}, nil, `{"field":"issue","verb":"created"}`},
		{"assignee added", Item{Seq: 9, Field: "assignees", Verb: VerbUpdated, OldValue: text(""), NewValue: text("Dave"), NewID: ref(dave)}, nil,
			`{"field":"assignees","verb":"updated","old_value":"","new_value":"Dave","new_id":"` + dave.String() + `"}`},
		{"relation", Item{Seq: 3, Field: "blocking", Verb: VerbCreated, NewID: ref(other)}, nil,
			`{"field":"blocking","verb":"created","new_id":"` + other.String() + `"}`},
		{"comment created", Item{Field: "comment", Verb: VerbCreated, CommentID: ref(comment)}, text("look"),
			`{"field":"comment","verb":"created",` + c + `,"excerpt":"look"}`},
		{"comment updated", Item{Field: "comment", Verb: VerbUpdated, CommentID: ref(comment)}, text("look again"),
			`{"field":"comment","verb":"updated",` + c + `,"excerpt":"look again"}`},
		{"comment mention", Item{Seq: 1, Field: "comment", Verb: VerbMentioned, CommentID: ref(comment)}, text("look"),
			`{"field":"comment","verb":"mentioned",` + c + `,"excerpt":"look"}`},
		{"comment of no text", Item{Field: "comment", Verb: VerbCreated, CommentID: ref(comment)}, text(""),
			`{"field":"comment","verb":"created",` + c + `,"excerpt":""}`},
		{"comment not there", Item{Field: "comment", Verb: VerbUpdated, CommentID: ref(comment)}, nil,
			`{"field":"comment","verb":"updated",` + c + `}`},
		{"comment deleted", Item{Field: "comment", Verb: VerbDeleted, CommentID: ref(comment)}, text("look"),
			`{"field":"comment","verb":"deleted",` + c + `}`},
		{"description mention", Item{Seq: 15, Field: "description", Verb: VerbMentioned}, text("look"),
			`{"field":"description","verb":"mentioned"}`},
	}
	for _, tt := range tests {
		got, err := json.Marshal(NewData(issue, tt.item, tt.text))
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		if want := `{"issue":` + issueJSON + `,"change":` + tt.change + `}`; string(got) != want {
			t.Errorf("%s: data =\n%s\nwant\n%s", tt.name, got, want)
		}
	}
}

// The excerpt is the first 200 characters of the comment's stored text,
// however many bytes they are (R6).
func TestExcerpt(t *testing.T) {
	tests := []struct {
		name, text, want string
	}{
		{"short", "look", "look"},
		{"200 letters", strings.Repeat("a", 200), strings.Repeat("a", 200)},
		{"201 letters", strings.Repeat("a", 201), strings.Repeat("a", 200)},
		{"200 characters, 201 bytes", strings.Repeat("a", 199) + "é", strings.Repeat("a", 199) + "é"},
		{"250 Chinese characters", strings.Repeat("评", 250), strings.Repeat("评", 200)},
		{"mixed", strings.Repeat("x", 150) + strings.Repeat("评", 100), strings.Repeat("x", 150) + strings.Repeat("评", 50)},
	}
	item := Item{Field: "comment", Verb: VerbCreated, CommentID: ref(comment)}
	for _, tt := range tests {
		got := NewData(IssueSnapshot{}, item, &tt.text).Change.Excerpt
		if got == nil || *got != tt.want {
			t.Errorf("%s: excerpt = %s, want %d characters, %s", tt.name, value(got), utf8.RuneCountInString(tt.want), value(&tt.want))
			continue
		}
		if !utf8.ValidString(*got) {
			t.Errorf("%s: excerpt is not valid UTF-8", tt.name)
		}
	}
}
