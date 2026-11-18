package domain

import (
	"slices"
	"strings"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Tab is a page of a receiver's notifications (P1c spec 2.3, R3; Plane's
// two tabs): all, of every sender but mentioned; mentions, of mentioned
// alone.
type Tab string

// The tabs.
const (
	TabAll      Tab = "all"
	TabMentions Tab = "mentions"
)

// typeSenders are the senders a filter's type may name, only in the all
// tab: its senders, in their canonical order.
var typeSenders = []Sender{SenderCreated, SenderAssigned, SenderSubscribed}

// TabSenders are the senders of the tab t's notifications, in their
// canonical order: all's every sender but mentioned, mentions' mentioned
// alone; none for another value. The unread counts are one for each tab
// (R3).
func TabSenders(t Tab) []Sender {
	switch t {
	case TabAll:
		return slices.Clone(typeSenders)
	case TabMentions:
		return []Sender{SenderMentioned}
	}
	return nil
}

// Filter is which of a receiver's notifications a list and mark-all-read
// select (spec 2.3, 2.7), besides those he may see (spec 2.5): those of
// one of Senders, never empty, in their canonical order, as the SQL takes
// them; archived ones when Archived, else ones not archived; ones snoozed
// at the moment of the read when Snoozed, else ones not snoozed (R6).
type Filter struct {
	Senders  []Sender
	Archived bool
	Snoozed  bool
}

// ListFilter is the list's filter (spec 2.3): Filter, and Read, nil for
// read and unread ones alike, else read ones only (true) or unread ones
// only (false). Mark-all-read has no Read: it marks unread ones.
type ListFilter struct {
	Filter
	Read *bool
}

// FilterValues are what a filter is made of, each nil when it is not
// given: mark-all-read's body's values, as the contract types them (spec
// 2.7), and the list's, once ParseListFilter has read its query's strings.
// The contract types them, it does not check them: a tab or a type may be
// any string.
type FilterValues struct {
	Tab      *Tab
	Type     *[]Sender
	Archived *bool
	Snoozed  *bool
}

// NewFilter is the filter of v (spec 2.3, 2.7): of tab's senders, all by
// default; in the all tab, of type's when it gives them, one or more of
// assigned, created and subscribed, each once; not archived and not
// snoozed by default. A tab that is not one, and a type that is not such
// a set, are invalid_format; a type with tab=mentions is not_allowed.
// Every problem is reported at once, as one 422 validation_failed.
func NewFilter(v FilterValues) (Filter, error) {
	f, problems := newFilter(v)
	if len(problems) > 0 {
		return Filter{}, shared.Invalid(problems...)
	}
	return f, nil
}

// newFilter is NewFilter's filter and its problems, tab's then type's: the
// one set of rules a list's query and mark-all-read's body share.
func newFilter(v FilterValues) (Filter, []shared.FieldError) {
	var problems []shared.FieldError
	tab := TabAll
	if v.Tab != nil {
		tab = *v.Tab
	}
	senders := TabSenders(tab)
	if senders == nil {
		problems = append(problems, shared.FieldError{Field: "tab", Code: shared.FieldInvalidFormat, Message: "must be all or mentions"})
	}
	if v.Type != nil {
		switch {
		case tab == TabMentions:
			problems = append(problems, shared.FieldError{Field: "type", Code: shared.FieldNotAllowed, Message: "is only allowed in the all tab"})
		case !isTypeSet(*v.Type):
			problems = append(problems, shared.FieldError{Field: "type", Code: shared.FieldInvalidFormat,
				Message: "must be one or more of assigned, created and subscribed, each once"})
		default:
			senders = slices.DeleteFunc(slices.Clone(typeSenders), func(s Sender) bool { return !slices.Contains(*v.Type, s) })
		}
	}
	return Filter{Senders: senders, Archived: v.Archived != nil && *v.Archived, Snoozed: v.Snoozed != nil && *v.Snoozed}, problems
}

// isTypeSet tells whether types is a filter's type: one or more of
// typeSenders, each once.
func isTypeSet(types []Sender) bool {
	for i, s := range types {
		if !slices.Contains(typeSenders, s) || slices.Contains(types[:i], s) {
			return false
		}
	}
	return len(types) > 0
}

// ListQuery is the list's filter as its query string carries it (spec
// 2.3): each parameter nil when it is absent, else its one value, a
// string. Whether a parameter is one of the list's and given once is the
// handler's.
type ListQuery struct {
	Tab, Type, Read, Archived, Snoozed *string
}

// ParseListFilter is the list's filter of q (spec 2.3): NewFilter's rules
// on its tab, its type, split at each comma, and its archived and
// snoozed; and its read. A boolean is true or false. Every problem is
// reported at once, as one 422 validation_failed, in the order tab, type,
// read, archived, snoozed.
func ParseListFilter(q ListQuery) (ListFilter, error) {
	var problems []shared.FieldError
	boolean := func(field string, value *string) *bool {
		if value == nil {
			return nil
		}
		switch *value {
		case "true":
			return new(true)
		case "false":
			return new(false)
		}
		problems = append(problems, shared.FieldError{Field: field, Code: shared.FieldInvalidFormat, Message: "must be true or false"})
		return nil
	}
	read, archived, snoozed := boolean("read", q.Read), boolean("archived", q.Archived), boolean("snoozed", q.Snoozed)
	v := FilterValues{Tab: (*Tab)(q.Tab), Archived: archived, Snoozed: snoozed}
	if q.Type != nil {
		var types []Sender
		for member := range strings.SplitSeq(*q.Type, ",") {
			types = append(types, Sender(member))
		}
		v.Type = &types
	}
	f, invalid := newFilter(v)
	if invalid = append(invalid, problems...); len(invalid) > 0 {
		return ListFilter{}, shared.Invalid(invalid...)
	}
	return ListFilter{Filter: f, Read: read}, nil
}
