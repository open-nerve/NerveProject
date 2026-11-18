package domain

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func ptr[T any](v T) *T { return &v }

// fieldCodes are the fields of err and their codes, "field:code", when it
// is one 422 validation_failed; nil, and the test fails, when it is not.
func fieldCodes(t *testing.T, err error) []string {
	t.Helper()
	var se *shared.Error
	if !errors.As(err, &se) || se.ProblemStatus() != 422 || se.Code != shared.CodeValidationFailed {
		t.Errorf("error = %v, want one 422 validation_failed", err)
		return nil
	}
	var got []string
	for _, f := range se.Fields {
		got = append(got, f.Field+":"+f.Code)
	}
	return got
}

// The tabs split the senders (spec 2.3, R3): all is every sender but
// mentioned, mentions is mentioned alone; each of the four is in exactly
// one; another tab has none.
func TestTabSenders(t *testing.T) {
	if got := TabSenders(TabAll); !slices.Equal(got, []Sender{SenderCreated, SenderAssigned, SenderSubscribed}) {
		t.Errorf("TabSenders(all) = %v", got)
	}
	if got := TabSenders(TabMentions); !slices.Equal(got, []Sender{SenderMentioned}) {
		t.Errorf("TabSenders(mentions) = %v", got)
	}
	for _, tab := range []Tab{"", "mention", "All"} {
		if got := TabSenders(tab); got != nil {
			t.Errorf("TabSenders(%q) = %v, want none", tab, got)
		}
	}
	for _, s := range []Sender{SenderCreated, SenderAssigned, SenderSubscribed, SenderMentioned} {
		if in := slices.Contains(TabSenders(TabAll), s); in == slices.Contains(TabSenders(TabMentions), s) {
			t.Errorf("%s is in all: %v, and in mentions too or in neither", s, in)
		}
	}
}

// The list's query (spec 2.3): each parameter optional, a string. tab is
// all by default, archived and snoozed false; read absent lists read and
// unread alike. type, only with tab=all, is a comma-separated non-empty
// set of assigned, created and subscribed, each once, and names exactly
// the senders listed, in the canonical order.
func TestParseListFilter(t *testing.T) {
	all := []Sender{SenderCreated, SenderAssigned, SenderSubscribed}
	tests := []struct {
		name  string
		query ListQuery
		want  ListFilter
	}{
		{"nothing", ListQuery{}, ListFilter{Filter: Filter{Senders: all}}},
		{"tab all", ListQuery{Tab: ptr("all")}, ListFilter{Filter: Filter{Senders: all}}},
		{"tab mentions", ListQuery{Tab: ptr("mentions")}, ListFilter{Filter: Filter{Senders: []Sender{SenderMentioned}}}},
		{"one type", ListQuery{Tab: ptr("all"), Type: ptr("assigned")}, ListFilter{Filter: Filter{Senders: []Sender{SenderAssigned}}}},
		{"type without tab", ListQuery{Type: ptr("created")}, ListFilter{Filter: Filter{Senders: []Sender{SenderCreated}}}},
		{"two types in canonical order", ListQuery{Type: ptr("subscribed,created")},
			ListFilter{Filter: Filter{Senders: []Sender{SenderCreated, SenderSubscribed}}}},
		{"every type", ListQuery{Type: ptr("subscribed,assigned,created")}, ListFilter{Filter: Filter{Senders: all}}},
		{"read", ListQuery{Read: ptr("true")}, ListFilter{Filter: Filter{Senders: all}, Read: ptr(true)}},
		{"unread", ListQuery{Read: ptr("false")}, ListFilter{Filter: Filter{Senders: all}, Read: ptr(false)}},
		{"archived", ListQuery{Archived: ptr("true")}, ListFilter{Filter: Filter{Senders: all, Archived: true}}},
		{"not archived", ListQuery{Archived: ptr("false")}, ListFilter{Filter: Filter{Senders: all}}},
		{"snoozed", ListQuery{Snoozed: ptr("true")}, ListFilter{Filter: Filter{Senders: all, Snoozed: true}}},
		{"not snoozed", ListQuery{Snoozed: ptr("false")}, ListFilter{Filter: Filter{Senders: all}}},
		{"everything", ListQuery{Tab: ptr("all"), Type: ptr("subscribed"), Read: ptr("false"), Archived: ptr("true"), Snoozed: ptr("true")},
			ListFilter{Filter: Filter{Senders: []Sender{SenderSubscribed}, Archived: true, Snoozed: true}, Read: ptr(false)}},
		{"mentions, read, archived, snoozed", ListQuery{Tab: ptr("mentions"), Read: ptr("true"), Archived: ptr("true"), Snoozed: ptr("true")},
			ListFilter{Filter: Filter{Senders: []Sender{SenderMentioned}, Archived: true, Snoozed: true}, Read: ptr(true)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseListFilter(tt.query)
			if err != nil || !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseListFilter() = %+v, %v; want %+v", got, err, tt.want)
			}
		})
	}
}

// A bad value is a 422 (spec 2.3): invalid_format for a tab or a boolean
// that is not one, and for a type with an empty, an unknown or a repeated
// member (mentioned is no type); not_allowed for any type with
// tab=mentions. Every bad parameter is in the one 422, in the order tab,
// type, read, archived, snoozed.
func TestParseListFilterRefuses(t *testing.T) {
	tests := []struct {
		name  string
		query ListQuery
		want  []string
	}{
		{"empty tab", ListQuery{Tab: ptr("")}, []string{"tab:invalid_format"}},
		{"unknown tab", ListQuery{Tab: ptr("mention")}, []string{"tab:invalid_format"}},
		{"tab in another case", ListQuery{Tab: ptr("All")}, []string{"tab:invalid_format"}},
		{"type with mentions", ListQuery{Tab: ptr("mentions"), Type: ptr("assigned")}, []string{"type:not_allowed"}},
		{"bad type with mentions", ListQuery{Tab: ptr("mentions"), Type: ptr("assigned,assigned")}, []string{"type:not_allowed"}},
		{"empty type", ListQuery{Type: ptr("")}, []string{"type:invalid_format"}},
		{"trailing comma", ListQuery{Type: ptr("assigned,")}, []string{"type:invalid_format"}},
		{"leading comma", ListQuery{Type: ptr(",assigned")}, []string{"type:invalid_format"}},
		{"empty member", ListQuery{Type: ptr("assigned,,created")}, []string{"type:invalid_format"}},
		{"repeated member", ListQuery{Type: ptr("assigned,assigned")}, []string{"type:invalid_format"}},
		{"member repeated later", ListQuery{Type: ptr("created,assigned,created")}, []string{"type:invalid_format"}},
		{"mentioned is no type", ListQuery{Type: ptr("mentioned")}, []string{"type:invalid_format"}},
		{"unknown sender", ListQuery{Type: ptr("assigned,watching")}, []string{"type:invalid_format"}},
		{"type in another case", ListQuery{Type: ptr("Assigned")}, []string{"type:invalid_format"}},
		{"blank in type", ListQuery{Type: ptr("assigned, created")}, []string{"type:invalid_format"}},
		{"type under a bad tab", ListQuery{Tab: ptr("x"), Type: ptr("assigned")}, []string{"tab:invalid_format"}},
		{"bad type under a bad tab", ListQuery{Tab: ptr("x"), Type: ptr("assigned,assigned")}, []string{"tab:invalid_format", "type:invalid_format"}},
		{"empty read", ListQuery{Read: ptr("")}, []string{"read:invalid_format"}},
		{"read as a number", ListQuery{Read: ptr("1")}, []string{"read:invalid_format"}},
		{"read in another case", ListQuery{Read: ptr("TRUE")}, []string{"read:invalid_format"}},
		{"archived as a word", ListQuery{Archived: ptr("yes")}, []string{"archived:invalid_format"}},
		{"snoozed in another case", ListQuery{Snoozed: ptr("False")}, []string{"snoozed:invalid_format"}},
		{"all but tab at once", ListQuery{Tab: ptr("mentions"), Type: ptr("x"), Read: ptr("x"), Archived: ptr("x"), Snoozed: ptr("x")},
			[]string{"type:not_allowed", "read:invalid_format", "archived:invalid_format", "snoozed:invalid_format"}},
		{"all at once", ListQuery{Tab: ptr("x"), Type: ptr(","), Read: ptr("x"), Archived: ptr("x"), Snoozed: ptr("x")},
			[]string{"tab:invalid_format", "type:invalid_format", "read:invalid_format", "archived:invalid_format", "snoozed:invalid_format"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseListFilter(tt.query)
			if fields := fieldCodes(t, err); !slices.Equal(fields, tt.want) || !reflect.DeepEqual(got, ListFilter{}) {
				t.Errorf("ParseListFilter() = %+v with %v, want nothing with %v", got, fields, tt.want)
			}
		})
	}
}

// Mark-all-read's body (spec 2.7) carries the list's filter but read,
// typed: the same rules, the same defaults and the same 422s.
func TestNewFilter(t *testing.T) {
	all := []Sender{SenderCreated, SenderAssigned, SenderSubscribed}
	mentions := Tab("mentions")
	tests := []struct {
		name   string
		values FilterValues
		want   Filter
	}{
		{"nothing", FilterValues{}, Filter{Senders: all}},
		{"tab all", FilterValues{Tab: ptr(TabAll)}, Filter{Senders: all}},
		{"tab mentions", FilterValues{Tab: &mentions}, Filter{Senders: []Sender{SenderMentioned}}},
		{"one type", FilterValues{Type: &[]Sender{SenderSubscribed}}, Filter{Senders: []Sender{SenderSubscribed}}},
		{"two types in canonical order", FilterValues{Tab: ptr(TabAll), Type: &[]Sender{SenderSubscribed, SenderAssigned}},
			Filter{Senders: []Sender{SenderAssigned, SenderSubscribed}}},
		{"archived and snoozed", FilterValues{Archived: ptr(true), Snoozed: ptr(true)}, Filter{Senders: all, Archived: true, Snoozed: true}},
		{"neither", FilterValues{Archived: ptr(false), Snoozed: ptr(false)}, Filter{Senders: all}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewFilter(tt.values)
			if err != nil || !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewFilter() = %+v, %v; want %+v", got, err, tt.want)
			}
		})
	}
	refused := []struct {
		name   string
		values FilterValues
		want   []string
	}{
		{"unknown tab", FilterValues{Tab: ptr(Tab("mention"))}, []string{"tab:invalid_format"}},
		{"type with mentions", FilterValues{Tab: &mentions, Type: &[]Sender{SenderAssigned}}, []string{"type:not_allowed"}},
		{"empty type with mentions", FilterValues{Tab: &mentions, Type: &[]Sender{}}, []string{"type:not_allowed"}},
		{"empty type", FilterValues{Type: &[]Sender{}}, []string{"type:invalid_format"}},
		{"empty member", FilterValues{Type: &[]Sender{""}}, []string{"type:invalid_format"}},
		{"repeated member", FilterValues{Type: &[]Sender{SenderAssigned, SenderCreated, SenderAssigned}}, []string{"type:invalid_format"}},
		{"mentioned is no type", FilterValues{Type: &[]Sender{SenderMentioned}}, []string{"type:invalid_format"}},
		{"unknown sender", FilterValues{Type: &[]Sender{"watching"}}, []string{"type:invalid_format"}},
		{"both", FilterValues{Tab: ptr(Tab("x")), Type: &[]Sender{SenderMentioned}}, []string{"tab:invalid_format", "type:invalid_format"}},
	}
	for _, tt := range refused {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewFilter(tt.values)
			if fields := fieldCodes(t, err); !slices.Equal(fields, tt.want) || !reflect.DeepEqual(got, Filter{}) {
				t.Errorf("NewFilter() = %+v with %v, want nothing with %v", got, fields, tt.want)
			}
		})
	}
}
