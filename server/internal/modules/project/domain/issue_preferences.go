package domain

// Filters contains optional arrays of list-filter atoms. A nil slice omits
// its key; an explicitly empty slice is retained and means no filtering.
type Filters struct {
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

// DisplayCalendar is the calendar part of the caller's list layout.
type DisplayCalendar struct {
	Layout       string `json:"layout"`
	ShowWeekends bool   `json:"show_weekends"`
}

// DisplayFilters is replaced whole; nil grouping pointers represent JSON null.
type DisplayFilters struct {
	Layout          string          `json:"layout"`
	GroupBy         *string         `json:"group_by"`
	SubGroupBy      *string         `json:"sub_group_by"`
	OrderBy         string          `json:"order_by"`
	SubIssue        bool            `json:"sub_issue"`
	ShowEmptyGroups bool            `json:"show_empty_groups"`
	Calendar        DisplayCalendar `json:"calendar"`
}

// DisplayProperties controls presentation only, including link visibility;
// it does not request additional list-query fields.
type DisplayProperties struct {
	Assignee      bool `json:"assignee"`
	StartDate     bool `json:"start_date"`
	DueDate       bool `json:"due_date"`
	Labels        bool `json:"labels"`
	Key           bool `json:"key"`
	Priority      bool `json:"priority"`
	State         bool `json:"state"`
	SubIssueCount bool `json:"sub_issue_count"`
	Link          bool `json:"link"`
	CreatedOn     bool `json:"created_on"`
	UpdatedOn     bool `json:"updated_on"`
}
