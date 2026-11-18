package bootstrap

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The rows of the permission matrix of one's own notifications (M7 P1c
// spec 2.8), on the workspace level's columns alone: a write's row with a
// project table's column would be taken for a write on a project
// (writesOnAProject). notification.read and notification.write let the
// workspace's admin, member and guest in (design 3.10); which of his
// notifications each of them sees is the projects he reads (spec 2.5). In
// the prepared data acme's admin and member read none of acme's projects
// (the admin is a member of none, the member's membership of the private
// one is deleted, partingStates), and the guest reads the public one,
// which shows him only his own work items, so each row's check tells the
// three apart: the guest's list, counts and marking hold his one
// notification there (seenBy), not his other one, on another's work item;
// the admin's and the member's none. A change of one's own notification is
// the guest's alone, then: the admin's and the member's are
// notification.not_found, as a notification of a project its receiver
// does not read is, unless the row's prepare makes them members of the
// public project on the cell's copy (prepareDraftPublisher). The lists
// have no such row: the guest's cells already tell what one sees from what
// one does not, and each prepared cell would be a write's, on a copy of its
// own. The callers the workspace is not visible to are workspace.not_found
// on its lists, and notification.not_found on their own notification:
// never's and the removed member's in acme, the deleted workspace's
// admin's, deleted with gone.
func notificationMatrixRows() []matrixRow {
	notFound := cell{http.StatusNotFound, "notification.not_found"}
	own := func(c caller, s seeded) (string, string, string) {
		return http.MethodPatch, "/api/v0/notifications/" + s.notification(c).String(), `{"read":true}`
	}
	changed := func(t *testing.T, c caller, s seeded, answer string) {
		var n apiNotification
		decodeAnswer(t, answer, &n)
		if n.ID != s.notification(c) || n.ReadAt == nil || n.Data.Issue.ID != s.issue(ownNotification(c).project, ownNotification(c).item) {
			t.Fatalf("%s's change of his notification = %s; want %s, read", c, answer, s.notification(c))
		}
	}
	return []matrixRow{
		{op: "listNotifications", request: toWorkspace(http.MethodGet, "/notifications", ""), cells: inWorkspace(cellOK, cellOK, cellOK),
			check: func(t *testing.T, c caller, s seeded, answer string) {
				var page notificationPage
				decodeAnswer(t, answer, &page)
				var got []uuid.UUID
				for _, n := range page.Data {
					got = append(got, n.ID)
				}
				if page.Data == nil || page.NextCursor != nil || !slices.Equal(got, seenBy(c, s)) {
					t.Fatalf("%s's notifications in %s = %s; want %v, without a cursor", c, workspaceOf(c), answer, seenBy(c, s))
				}
			}},
		{op: "getNotificationUnreadCount", request: toWorkspace(http.MethodGet, "/notifications/unread-count", ""),
			cells: inWorkspace(cellOK, cellOK, cellOK), check: func(t *testing.T, c caller, s seeded, answer string) {
				var got unreadCounts
				decodeAnswer(t, answer, &got)
				if want := (unreadCounts{All: len(seenBy(c, s))}); got != want {
					t.Fatalf("%s's unread counts in %s = %+v; want %+v", c, workspaceOf(c), got, want)
				}
			}},
		{op: "markAllNotificationsRead", write: true, request: toWorkspace(http.MethodPost, "/notifications/mark-all-read", `{}`),
			cells: inWorkspace(cellOK, cellOK, cellOK), check: func(t *testing.T, c caller, s seeded, answer string) {
				var got struct {
					Marked int `json:"marked"`
				}
				decodeAnswer(t, answer, &got)
				if got.Marked != len(seenBy(c, s)) {
					t.Fatalf("%s's mark-all-read in %s marked %d; want %d", c, workspaceOf(c), got.Marked, len(seenBy(c, s)))
				}
			}},
		{op: "updateNotification", write: true, request: own, cells: workspaceCells(notFound, notFound, cellOK, notFound, notFound, notFound),
			check: changed},
		{op: "updateNotification", variant: "admin and member in its project", write: true, prepare: prepareDraftPublisher, request: own,
			cells: workspaceCells(cellOK, cellOK, cellOK, notFound, notFound, notFound), check: changed},
	}
}

// seenBy are the notifications c's cells see in the workspace they target,
// in the prepared data: the guest's own one, of his own work item in a
// project that shows him only his own; none for the admin and the member,
// who read none of acme's projects.
func seenBy(c caller, s seeded) []uuid.UUID {
	if c == callerGuest {
		return []uuid.UUID{s.notification(c)}
	}
	return nil
}

// seededNotification is a notification prepareMatrix seeds: the key
// newSeeded names it by, its workspace's slug, "/" and whose; its
// receiver; and its work item, by its project's key and its name.
type seededNotification struct {
	key           string
	receiver      caller
	project, item string
}

// matrixNotifications are the notifications prepareMatrix seeds: for each
// workspace column, one of its caller's in the workspace his cells target,
// the one they name (ownNotification); and the guest's other one in acme,
// on its public project's matrixIssue, its admin's work item, which that
// project, showing him only his own, hides from him: a list, a count or a
// marking that took in more than he sees would hold it.
func matrixNotifications() []seededNotification {
	ns := make([]seededNotification, 0, len(workspaceColumns)+1)
	for _, c := range workspaceColumns {
		ns = append(ns, ownNotification(c))
	}
	return append(ns, seededNotification{key: workspaceOf(callerGuest) + "/" + string(callerGuest) + ", another's work item",
		receiver: callerGuest, project: projectOf(callerGuest), item: matrixIssue})
}

// ownNotification is the notification of c's that his cells name, on a work
// item of his column's project (projectOf): the guest's on his own work
// item (issueOf), which the public project shows him; every other's on
// the project's matrixIssue: acme's public one's for the admin and the
// member, which they read once they are its members, and for never and the
// removed member, who are not acme's members; gone's for its admin,
// deleted with it.
func ownNotification(c caller) seededNotification {
	item := matrixIssue
	if c == callerGuest {
		item = issueOf(callerGuestOwn)
	}
	return seededNotification{key: workspaceOf(c) + "/" + string(c), receiver: c, project: projectOf(c), item: item}
}

// notification is the id of the notification of c's that his cells name;
// one never seeded fails the test at once, as membership's does.
func (s seeded) notification(c caller) uuid.UUID {
	id, ok := s.notifications[ownNotification(c).key]
	if !ok {
		s.t.Helper()
		s.t.Fatalf("no notification of %s in %s is seeded", c, workspaceOf(c))
	}
	return id
}

// seedMatrixNotifications writes each of matrixNotifications with the id
// newSeeded named, directly, as the P1b deletion guards write theirs
// (seedNotification): nerve's own, an automatic archive's to a subscriber,
// unread, at at, in its work item's workspace and project, its creator
// the work item's maker, its snapshot the work item's number, project
// identifier and state. prepareMatrix writes them before it deletes gone,
// which deletes gone's with its project.
func seedMatrixNotifications(t *testing.T, pool *pgxpool.Pool, s seeded, at time.Time) {
	t.Helper()
	for _, n := range matrixNotifications() {
		tag, err := pool.Exec(pgtest.Soon(t), `INSERT INTO notifications (id, workspace_id, project_id, receiver_id, issue_id, issue_created_by_id, sender,
				data, event_id, change_seq, created_at, updated_at)
			SELECT $1, i.workspace_id, i.project_id, $2, i.id, i.created_by_id, 'subscribed',
				jsonb_build_object('issue', jsonb_build_object('id', i.id, 'name', i.name, 'sequence_id', i.sequence_id, 'identifier', p.identifier,
					'state_name', st.name, 'state_group', st."group"),
					'change', jsonb_build_object('field', 'archived_at', 'verb', 'updated', 'new_value', 'archive')),
				$3, 0, $4, $4
			FROM issues i JOIN projects p ON p.id = i.project_id JOIN states st ON st.id = i.state_id
			WHERE i.id = $5`, s.notifications[n.key], s.account(n.receiver), uuid.NewV7(), at, s.issue(n.project, n.item))
		if err != nil || tag.RowsAffected() != 1 {
			t.Fatalf("seeding the notification %s on %s's %s: %d rows, %v; want 1", n.key, n.project, n.item, tag.RowsAffected(), err)
		}
	}
}

// seededNotificationRows checks the notifications the cells rest on, read
// back by the ids newSeeded named: each of matrixNotifications its
// receiver's, in its work item's workspace and project, its creator the
// work item's maker, unread, unarchived and not snoozed; undeleted, but
// gone's, which gone's deletion deleted. One missing, or stored otherwise,
// would let a cell answer as it wants for another reason.
func seededNotificationRows(t *testing.T, pool *pgxpool.Pool, s seeded) {
	t.Helper()
	for _, n := range matrixNotifications() {
		slug, _, _ := strings.Cut(n.project, "/")
		_, maker := numberOf(n.project, n.item)
		var stored bool
		if err := pool.QueryRow(pgtest.Soon(t), `SELECT EXISTS (SELECT 1 FROM notifications WHERE id = $1 AND receiver_id = $2 AND workspace_id = $3
			AND project_id = $4 AND issue_id = $5 AND issue_created_by_id = $6 AND read_at IS NULL AND archived_at IS NULL AND snoozed_till IS NULL
			AND (deleted_at IS NOT NULL) = $7)`, s.notifications[n.key], s.account(n.receiver), s.workspace(slug), s.project(n.project),
			s.issue(n.project, n.item), s.account(maker), slug == "gone").Scan(&stored); err != nil || !stored {
			t.Fatalf("the notification %s: %v, %v; want %s's on %s's %s, made by %s, unread, deleted %v", n.key, stored, err, n.receiver,
				n.project, n.item, maker, slug == "gone")
		}
	}
}
