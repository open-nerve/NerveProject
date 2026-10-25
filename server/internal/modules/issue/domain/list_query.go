package domain

import (
	"slices"
	"strconv"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// IDFilter keeps UUID alternatives and the empty-value alternative together.
type IDFilter struct {
	IDs  []uuid.UUID
	None bool
}

// ListFilters contains alternatives within each field. Different fields and
// every element of Start/Target are conjunctive; disjoint dates remain valid.
type ListFilters struct {
	StateIDs, CreatorIDs, MentionIDs, SubscriberIDs, ProjectIDs []uuid.UUID
	StateGroups                                                 []StateGroup
	Priorities                                                  []Priority
	Labels, Assignees                                           IDFilter
	Start, Target                                               []DateFilter
	SubIssues                                                   bool
	Archived                                                    bool
}

// ListQuery is a validated list request, independent of HTTP binding.
type ListQuery struct {
	Filters               ListFilters
	Order                 ListOrder
	GroupBy, SubGroupBy   ListGroup
	GroupKey, SubGroupKey *string
	Relation              string
	Limit                 int
	Cursor                *ListCursor
}

// ParseListQuery preserves raw parameter multiplicity, resolves me to the
// caller, and checks page capabilities without looking up any referenced ID.
func ParseListQuery(raw map[string][]string, scope ListScope, actorID uuid.UUID) (ListQuery, error) {
	q := ListQuery{Order: "-created_at", Filters: ListFilters{SubIssues: true}}
	if scope != ProjectList && scope != MemberList {
		return ListQuery{}, listNotAllowed("scope")
	}
	// Stable validation order also makes requests with several bad fields
	// deterministic. Relationship checks run after all individual values.
	fields := make([]string, 0, len(raw))
	for field := range raw {
		fields = append(fields, field)
	}
	slices.Sort(fields)
	var limit *int
	for _, field := range fields {
		values := raw[field]
		date := field == "start_date" || field == "target_date"
		if len(values) == 0 || (!date && len(values) != 1) {
			return ListQuery{}, listNotAllowed(field)
		}
		if ListFilterAllowed(field, scope) {
			if err := q.Filters.parse(field, values, actorID); err != nil {
				return ListQuery{}, err
			}
			continue
		}
		value := values[0]
		switch field {
		case "order_by":
			if !ListOrderAllowed(ListOrder(value)) {
				return ListQuery{}, listNotAllowed(field)
			}
			q.Order = ListOrder(value)
		case "group_by":
			if !ListGroupAllowed(ListGroup(value), scope) {
				return ListQuery{}, listNotAllowed(field)
			}
			q.GroupBy = ListGroup(value)
		case "sub_group_by":
			if !ListSubGroupAllowed(ListGroup(value), scope) {
				return ListQuery{}, listNotAllowed(field)
			}
			q.SubGroupBy = ListGroup(value)
		case "group_key":
			q.GroupKey = &value
		case "sub_group_key":
			q.SubGroupKey = &value
		case "relation":
			if scope != MemberList || (value != "assigned" && value != "created" && value != "subscribed") {
				return ListQuery{}, listNotAllowed(field)
			}
			q.Relation = value
		case "limit":
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return ListQuery{}, listBindingError(field, shared.FieldInvalidFormat)
			}
			limit = &parsed
		case "cursor": // Decode after order and group/key relationships are known.
		default:
			return ListQuery{}, listNotAllowed(field)
		}
	}
	if scope == MemberList && q.Relation == "" {
		return ListQuery{}, listBindingError("relation", shared.FieldRequired)
	}
	var err error
	q.Limit, err = shared.PageSize(limit)
	if err != nil {
		return ListQuery{}, err
	}
	_, hasCursor := raw["cursor"]
	if err := q.validateGroups(hasCursor); err != nil {
		return ListQuery{}, err
	}
	if hasCursor {
		cursor, err := DecodeListCursor(raw["cursor"][0], q.Order)
		if err != nil {
			return ListQuery{}, err
		}
		q.Cursor = &cursor
	}
	return q, nil
}

func (q ListQuery) validateGroups(hasCursor bool) error {
	if q.SubGroupBy != "" && (q.GroupBy == "" || q.GroupBy == q.SubGroupBy) {
		return listNotAllowed("sub_group_by")
	}
	if q.GroupKey != nil && (q.GroupBy == "" || !ListGroupKeyAllowed(q.GroupBy, *q.GroupKey)) {
		return listNotAllowed("group_key")
	}
	if q.SubGroupKey != nil && (q.GroupKey == nil || q.SubGroupBy == "" || !ListGroupKeyAllowed(q.SubGroupBy, *q.SubGroupKey)) {
		return listNotAllowed("sub_group_key")
	}
	if hasCursor && ((q.GroupBy != "" && q.GroupKey == nil) || (q.SubGroupBy != "" && q.SubGroupKey == nil)) {
		return listNotAllowed("cursor")
	}
	if q.SubGroupBy != "" && q.GroupKey != nil && q.SubGroupKey == nil {
		return listNotAllowed("sub_group_key")
	}
	return nil
}

func listNotAllowed(field string) *shared.Error {
	return shared.Invalid(shared.FieldError{Field: field, Code: shared.FieldNotAllowed, Message: "is not allowed for this list"})
}

func listBindingError(field, code string) *shared.Error {
	return &shared.Error{Kind: shared.KindBadRequest, Code: shared.CodeBadRequest, Detail: "The list parameter is malformed.", Fields: []shared.FieldError{{Field: field, Code: code, Message: "must have the required parameter format"}}}
}
