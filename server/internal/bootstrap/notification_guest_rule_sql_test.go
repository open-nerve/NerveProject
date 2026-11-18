package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	notificationpg "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/postgres"
	notificationapp "github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
	notificationdomain "github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// guestCell is a cell of the guest rule's equivalence (M7 P1c spec R2):
// a notification of gail's of a work item whose creator is creator, nil
// when not known, the one notification of its project, which gail reads,
// own-only or not.
type guestCell struct {
	name          string
	creator       *uuid.UUID
	ownOnly       bool
	project, note uuid.UUID
}

// errRolledBack ends a transaction that must leave nothing behind.
var errRolledBack = errors.New("rolled back")

// The guest rule's copy in the notification module's SQL selects what the
// rule itself shows (M7 P1c spec 2.5, R2; pre-flight re-84 m2): in each of
// the six cells, {creator not known, gail, alice} × {own-only, not}, the
// list, the unread counts and mark-all-read's batch select gail's
// notification exactly when issue's HiddenFrom, as the notification
// module's GuestRule is handed it (notificationIssues over issue.Provide),
// does not hide its work item from her. Each cell has a project and a
// notification of its own and is read alone, its batch in a transaction
// rolled back; then the six projects at once, of paired ids and flags,
// select the same, and the batch, committed, marks those. The comparison
// is against the predicate, not a copy of its table: a rule that changed
// would hold the SQL to the change.
func TestNotificationGuestRuleSQLIsHiddenFrom(t *testing.T) {
	pool := openPool(t, pgtest.NewDatabase(t))
	ctx := context.Background()
	gail, alice, acme := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	for id, name := range map[uuid.UUID]string{gail: "gail", alice: "alice"} {
		exec("INSERT INTO users (id, email, password, display_name) VALUES ($1, $2, 'x', $3)", id, name+"@example.com", name)
	}
	exec("INSERT INTO workspaces (id, name, slug) VALUES ($1, 'acme', 'acme')", acme)
	var cells []guestCell
	for _, own := range []bool{true, false} {
		for _, c := range []struct {
			name    string
			creator *uuid.UUID
		}{{"not known", nil}, {"gail", &gail}, {"alice", &alice}} {
			cell := guestCell{name: fmt.Sprintf("created by %s, own-only %t", c.name, own), creator: c.creator, ownOnly: own,
				project: uuid.NewV7(), note: uuid.NewV7()}
			exec("INSERT INTO projects (id, workspace_id, name, identifier) VALUES ($1, $2, $3, $3)", cell.project, acme, fmt.Sprintf("P%d", len(cells)))
			exec(`INSERT INTO notifications (id, workspace_id, project_id, receiver_id, issue_id, issue_created_by_id, sender, data, event_id, change_seq,
					created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, 'subscribed', jsonb_build_object('issue', jsonb_build_object('id', $5::uuid, 'name', 'Fix', 'sequence_id', 1),
					'change', jsonb_build_object('field', 'name', 'verb', 'updated')), $7, 0, now() - interval '1 hour', now() - interval '1 hour')`,
				cell.note, acme, cell.project, gail, uuid.NewV7(), cell.creator, uuid.NewV7())
			cells = append(cells, cell)
		}
	}

	store := notificationpg.New(pool)
	rule := notificationIssues{issues: issue.Provide(pool, nil).NotificationIssues}
	tx := postgres.NewTxManager(pool, 5*time.Second)
	filter := notificationdomain.Filter{Senders: notificationdomain.TabSenders(notificationdomain.TabAll)}
	inbox := func(cells ...guestCell) notificationapp.Inbox {
		grants := make([]notificationapp.Grant, len(cells))
		for n, c := range cells {
			grants[n] = notificationapp.Grant{ProjectID: c.project, OwnOnly: c.ownOnly}
		}
		return notificationapp.Inbox{ReceiverID: gail, WorkspaceID: acme, Grants: grants, Now: time.Now()}
	}
	listed := func(in notificationapp.Inbox) []uuid.UUID {
		rows, err := store.ListNotifications(ctx, notificationapp.ListRead{Inbox: in, Filter: notificationdomain.ListFilter{Filter: filter}, Limit: len(cells) + 1})
		if err != nil {
			t.Fatal(err)
		}
		ids := make([]uuid.UUID, len(rows))
		for n, r := range rows {
			ids[n] = r.ID
		}
		return slices.SortedFunc(slices.Values(ids), uuid.UUID.Compare)
	}
	counted := func(in notificationapp.Inbox) int {
		counts, err := store.CountUnread(ctx, notificationapp.UnreadRead{Inbox: in, All: filter.Senders,
			Mentions: notificationdomain.TabSenders(notificationdomain.TabMentions)})
		if err != nil {
			t.Fatal(err)
		}
		return counts.All
	}
	marked := func(ctx context.Context, in notificationapp.Inbox) int {
		batch, err := store.MarkReadBatch(ctx, notificationapp.ReadBatch{Inbox: in, Filter: filter, Limit: 1000})
		if err != nil {
			t.Fatal(err)
		}
		return batch.Marked
	}

	var shown []uuid.UUID
	for _, c := range cells {
		want := !rule.HiddenFrom(c.creator, gail, c.ownOnly)
		in := inbox(c)
		var batch int
		if err := tx.WithinTx(ctx, func(ctx context.Context) error { batch = marked(ctx, in); return errRolledBack }); !errors.Is(err, errRolledBack) {
			t.Fatal(err)
		}
		for statement, got := range map[string]bool{"the list": slices.Equal(listed(in), []uuid.UUID{c.note}), "the unread count": counted(in) == 1,
			"mark-all-read's batch": batch == 1} {
			if got != want {
				t.Errorf("%s: %s selects gail's notification %t; HiddenFrom shows its work item %t", c.name, statement, got, want)
			}
		}
		if want {
			shown = append(shown, c.note)
		}
	}
	slices.SortFunc(shown, uuid.UUID.Compare)
	all := inbox(cells...)
	if got := listed(all); !slices.Equal(got, shown) {
		t.Errorf("of the six projects at once, the list selects %v; want %v", got, shown)
	}
	if got := counted(all); got != len(shown) {
		t.Errorf("of the six projects at once, the unread count is %d; want %d", got, len(shown))
	}
	if got := marked(ctx, all); got != len(shown) {
		t.Errorf("of the six projects at once, mark-all-read's batch marked %d; want %d", got, len(shown))
	}
	read := queryIDs(t, pool, "SELECT id FROM notifications WHERE receiver_id = $1 AND read_at IS NOT NULL ORDER BY id", gail)
	if !slices.Equal(read, shown) {
		t.Errorf("of the six projects at once, mark-all-read's batch marked %v read; want %v", read, shown)
	}
}
