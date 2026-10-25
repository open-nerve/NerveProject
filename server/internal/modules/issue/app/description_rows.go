package app

import "uuid"

// DescriptionWrite changes only the two stored description columns.
type DescriptionWrite struct {
	IssueID    uuid.UUID
	HTML, Text string
}
