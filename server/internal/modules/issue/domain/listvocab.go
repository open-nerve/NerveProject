package domain

import (
	"slices"
	"strings"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListScope identifies the page whose list capabilities are being checked.
// The zero value and unknown scopes have no capabilities.
type ListScope uint8

const (
	ProjectList ListScope = iota + 1
	MemberList
)

// ListOrder is a list ordering word, with a leading minus for descending.
type ListOrder string

// ListGroup is a list grouping field.
type ListGroup string

// ListFilterAllowed reports whether field is a filter on scope. Control
// parameters and display-settings filter schemas are checked by their callers.
func ListFilterAllowed(field string, scope ListScope) bool {
	switch scope {
	case ProjectList:
		switch field {
		case "state_id", "state_group", "priority", "label_id", "assignee_id", "created_by_id", "mention_id", "subscriber_id", "start_date", "target_date", "sub_issue", "archived":
			return true
		}
	case MemberList:
		switch field {
		case "state_group", "priority", "label_id", "project_id", "start_date", "target_date", "sub_issue":
			return true
		}
	}
	return false
}

// ListGroupAllowed reports whether group is available on scope.
func ListGroupAllowed(group ListGroup, scope ListScope) bool {
	switch scope {
	case ProjectList:
		switch group {
		case "state_id", "state_group", "priority", "label_id", "assignee_id", "created_by_id", "target_date":
			return true
		}
	case MemberList:
		switch group {
		case "state_group", "priority", "label_id", "target_date", "project_id":
			return true
		}
	}
	return false
}

// ListSubGroupAllowed checks page capability only; the caller checks that
// a primary group is present and differs from the subgroup.
func ListSubGroupAllowed(group ListGroup, scope ListScope) bool {
	return scope == ProjectList && ListGroupAllowed(group, scope)
}

// ListOrderAllowed accepts exactly the 21 P8 ordering words. Ascending
// words have no prefix; sort_order has no descending form.
func ListOrderAllowed(value ListOrder) bool {
	if value == "sort_order" {
		return true
	}
	switch strings.TrimPrefix(string(value), "-") {
	case "created_at", "updated_at", "start_date", "target_date", "priority", "state_name", "label_name", "assignee_name", "sub_issues_count", "link_count":
		return true
	}
	return false
}

// ListFilterValueAllowed checks one CSV atom or one whole date condition.
// It neither splits CSV nor resolves me or looks up an object's existence.
// A query parser or settings adapter supplies the context for error fields.
func ListFilterValueAllowed(field, value string) bool {
	switch field {
	case "state_id", "project_id":
		return listUUIDAllowed(value)
	case "label_id":
		return value == "none" || listUUIDAllowed(value)
	case "assignee_id":
		return value == "none" || value == "me" || listUUIDAllowed(value)
	case "created_by_id", "mention_id", "subscriber_id":
		return value == "me" || listUUIDAllowed(value)
	case "priority":
		return slices.Contains(priorities, Priority(value))
	case "state_group":
		return listStateGroupAllowed(StateGroup(value))
	case "start_date", "target_date":
		_, err := ParseListDateFilter(value)
		return err == nil
	case "sub_issue", "archived":
		return value == "true" || value == "false"
	}
	return false
}

// ListGroupKeyAllowed checks a single group key's syntax. Unlike filter
// atoms, creator groups may be none, me is never a key, and dates are days.
func ListGroupKeyAllowed(group ListGroup, key string) bool {
	switch group {
	case "state_id", "project_id":
		return listUUIDAllowed(key)
	case "label_id", "assignee_id", "created_by_id":
		return key == "none" || listUUIDAllowed(key)
	case "priority":
		return slices.Contains(priorities, Priority(key))
	case "state_group":
		return listStateGroupAllowed(StateGroup(key))
	case "target_date":
		if key == "none" {
			return true
		}
		_, err := shared.ParseDate(key)
		return err == nil
	}
	return false
}

func listUUIDAllowed(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}

func listStateGroupAllowed(group StateGroup) bool {
	switch group {
	case "backlog", "unstarted", "started", GroupCompleted, GroupCancelled:
		return true
	}
	return false
}
