package domain

import (
	"strings"
	"uuid"
)

func (f *ListFilters) parse(field string, values []string, actorID uuid.UUID) error {
	if field == "start_date" || field == "target_date" {
		conditions := make([]DateFilter, 0, len(values))
		for _, value := range values {
			condition, err := ParseListDateFilter(value)
			if err != nil || (condition.None && len(values) > 1) {
				return listNotAllowed(field)
			}
			conditions = append(conditions, condition)
		}
		if field == "start_date" {
			f.Start = conditions
		} else {
			f.Target = conditions
		}
		return nil
	}
	if field == "sub_issue" || field == "archived" {
		if !ListFilterValueAllowed(field, values[0]) {
			return listNotAllowed(field)
		}
		if field == "sub_issue" {
			f.SubIssues = values[0] == "true"
		} else {
			f.Archived = values[0] == "true"
		}
		return nil
	}
	for _, value := range strings.Split(values[0], ",") {
		if !ListFilterValueAllowed(field, value) {
			return listNotAllowed(field)
		}
		switch field {
		case "state_group":
			f.StateGroups = append(f.StateGroups, StateGroup(value))
		case "priority":
			f.Priorities = append(f.Priorities, Priority(value))
		default:
			if err := f.appendID(field, value, actorID); err != nil {
				return err
			}
		}
	}
	return nil
}

// appendID records one checked atom on its own field: none only on the four
// fields whose atoms admit it, each its own, never another field's.
func (f *ListFilters) appendID(field, value string, actorID uuid.UUID) error {
	if value == "none" {
		switch field {
		case "label_id":
			f.Labels.None = true
		case "assignee_id":
			f.Assignees.None = true
		case "cycle_id":
			f.Cycles.None = true
		case "module_id":
			f.Modules.None = true
		}
		return nil
	}
	id := actorID
	if value != "me" {
		parsed, err := uuid.Parse(value)
		if err != nil {
			return listNotAllowed(field)
		}
		id = parsed
	}
	switch field {
	case "state_id":
		f.StateIDs = append(f.StateIDs, id)
	case "created_by_id":
		f.CreatorIDs = append(f.CreatorIDs, id)
	case "mention_id":
		f.MentionIDs = append(f.MentionIDs, id)
	case "subscriber_id":
		f.SubscriberIDs = append(f.SubscriberIDs, id)
	case "project_id":
		f.ProjectIDs = append(f.ProjectIDs, id)
	case "label_id":
		f.Labels.IDs = append(f.Labels.IDs, id)
	case "assignee_id":
		f.Assignees.IDs = append(f.Assignees.IDs, id)
	case "cycle_id":
		f.Cycles.IDs = append(f.Cycles.IDs, id)
	case "module_id":
		f.Modules.IDs = append(f.Modules.IDs, id)
	}
	return nil
}
