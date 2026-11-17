package app

// NotificationState is a work item's state as project.Provide reads it for
// the notification module's snapshot (M7 P1b spec 2.5): its name and its
// group. bootstrap converts it into notification's value.
type NotificationState struct {
	Name, Group string
}
