package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The notification module's actions: the keys of its rows in the access
// module's rule table (M7 design 3.10). Both are the workspace's, for its
// admins, members and guests, and only ever on the caller's own
// notifications, which the module's reads and writes keep to.
const (
	// ActionNotificationRead is listing one's own notifications in a
	// workspace and counting the unread ones.
	ActionNotificationRead shared.Action = "notification.read"
	// ActionNotificationWrite is marking one's own notifications read or
	// unread, archiving and snoozing them, and marking them all read.
	ActionNotificationWrite shared.Action = "notification.write"
)

// Actions lists the module's actions. bootstrap's test holds the union of
// every module's Actions equal to the rule table's keys (M3 design 3.4).
func Actions() []shared.Action {
	return []shared.Action{ActionNotificationRead, ActionNotificationWrite}
}
