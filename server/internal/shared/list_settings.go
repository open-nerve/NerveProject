package shared

// The display settings of a saved work-item list, the same on a project's,
// a cycle's and a module's page (M6/P3b spec R5): the contract's
// IssueListCalendar, IssueListDisplayFilters and IssueListDisplayProperties
// (api/common.yaml). Each page stores them as jsonb under these JSON names,
// and the issue module's ListSettings checks their words.

// ListFilters are the filter keys every page's saved list has, each an
// optional array of list-filter atoms. A nil slice omits its key; an
// explicitly empty slice is kept and means no filtering for it. A page's
// filters embed it and add the page's own keys.
type ListFilters struct {
	StateID      []string `json:"state_id,omitzero"`
	StateGroup   []string `json:"state_group,omitzero"`
	Priority     []string `json:"priority,omitzero"`
	LabelID      []string `json:"label_id,omitzero"`
	AssigneeID   []string `json:"assignee_id,omitzero"`
	CreatedByID  []string `json:"created_by_id,omitzero"`
	MentionID    []string `json:"mention_id,omitzero"`
	SubscriberID []string `json:"subscriber_id,omitzero"`
	StartDate    []string `json:"start_date,omitzero"`
	TargetDate   []string `json:"target_date,omitzero"`
}

// Map returns the given keys by their JSON names, as the issue module's
// ListSettings takes them: a nil slice is left out, an empty one is kept.
// The map is never nil, so a page adds its own keys to it.
func (f ListFilters) Map() map[string][]string {
	out := make(map[string][]string)
	for _, entry := range []struct {
		key    string
		values []string
	}{
		{"state_id", f.StateID}, {"state_group", f.StateGroup}, {"priority", f.Priority}, {"label_id", f.LabelID}, {"assignee_id", f.AssigneeID},
		{"created_by_id", f.CreatedByID}, {"mention_id", f.MentionID}, {"subscriber_id", f.SubscriberID}, {"start_date", f.StartDate}, {"target_date", f.TargetDate},
	} {
		if entry.values != nil {
			out[entry.key] = entry.values
		}
	}
	return out
}

// ListCalendar is the calendar part of a list's layout.
type ListCalendar struct {
	Layout       string `json:"layout"`
	ShowWeekends bool   `json:"show_weekends"`
}

// ListDisplayFilters are a list's layout, grouping and order, replaced
// whole; nil grouping pointers represent JSON null.
type ListDisplayFilters struct {
	Layout          string       `json:"layout"`
	GroupBy         *string      `json:"group_by"`
	SubGroupBy      *string      `json:"sub_group_by"`
	OrderBy         string       `json:"order_by"`
	SubIssue        bool         `json:"sub_issue"`
	ShowEmptyGroups bool         `json:"show_empty_groups"`
	Calendar        ListCalendar `json:"calendar"`
}

// ListDisplayProperties control presentation only, including link
// visibility; they do not request additional list-query fields.
type ListDisplayProperties struct {
	Assignee      bool `json:"assignee"`
	StartDate     bool `json:"start_date"`
	DueDate       bool `json:"due_date"`
	Labels        bool `json:"labels"`
	Key           bool `json:"key"`
	Priority      bool `json:"priority"`
	State         bool `json:"state"`
	SubIssueCount bool `json:"sub_issue_count"`
	Link          bool `json:"link"`
	// AttachmentCount shows a work item's number of live ordinary attachments.
	AttachmentCount bool `json:"attachment_count"`
	CreatedOn       bool `json:"created_on"`
	UpdatedOn       bool `json:"updated_on"`
}

// DefaultListDisplayFilters are the display filters of a list nobody has
// set: a list, ungrouped, newest first, with sub-work items and empty
// groups shown, the calendar by month without weekends.
func DefaultListDisplayFilters() ListDisplayFilters {
	return ListDisplayFilters{Layout: "list", OrderBy: "-created_at", SubIssue: true, ShowEmptyGroups: true, Calendar: ListCalendar{Layout: "month"}}
}

// DefaultListDisplayProperties show every property.
func DefaultListDisplayProperties() ListDisplayProperties {
	return ListDisplayProperties{Assignee: true, StartDate: true, DueDate: true, Labels: true, Key: true, Priority: true, State: true,
		SubIssueCount: true, Link: true, AttachmentCount: true, CreatedOn: true, UpdatedOn: true}
}
