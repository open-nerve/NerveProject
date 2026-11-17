package postgresadapter_test

import (
	"context"
	"maps"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// ShareIssue (M7 P1a spec 2.4), in the transaction the caller holds,
// answers the undeleted work item, archived or not, as its own row has it:
// its id, workspace, project and creator, nil when not known; and locks
// that row FOR SHARE and nothing else: a FOR NO KEY UPDATE of it, which
// every UPDATE of issues takes, waits; another FOR SHARE and a foreign
// key's FOR KEY SHARE do not; the other work item's row, its project's and
// its workspace's are not locked. A deleted work item, and one not there,
// answer nothing.
func TestShareIssueLocksTheWorkItemsRowAlone(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	id, archived, deleted, other := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for n, i := range []uuid.UUID{id, archived, deleted, other} {
		newIssue(t, pool, web, i, n+1, alice)
	}
	exec(t, pool, "UPDATE issues SET archived_at = $2, created_by_id = NULL WHERE id = $1", archived, now)
	exec(t, pool, "UPDATE issues SET deleted_at = $2 WHERE id = $1", deleted, now)
	err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		got, found, err := s.ShareIssue(ctx, id)
		if err != nil {
			return err
		}
		if want := (app.SharedIssue{ID: id, WorkspaceID: web.workspace, ProjectID: web.id, CreatedByID: &alice}); !found || !reflect.DeepEqual(got, want) {
			t.Errorf("ShareIssue() = %+v, %v; want %+v", got, found, want)
		}
		if !pgtest.RowWaits(t, pool, "issues", id, "FOR NO KEY UPDATE") || pgtest.RowWaits(t, pool, "issues", id, "FOR SHARE") ||
			pgtest.RowWaits(t, pool, "issues", id, "FOR KEY SHARE") {
			t.Error("the work item's row must be held FOR SHARE: a FOR NO KEY UPDATE waits, a FOR SHARE and a FOR KEY SHARE do not")
		}
		if pgtest.RowWaits(t, pool, "issues", other, "FOR UPDATE") || pgtest.RowWaits(t, pool, "projects", web.id, "FOR UPDATE") ||
			pgtest.RowWaits(t, pool, "workspaces", web.workspace, "FOR UPDATE") {
			t.Error("a row ShareIssue was not asked for is locked")
		}
		got, found, err = s.ShareIssue(ctx, archived)
		if want := (app.SharedIssue{ID: archived, WorkspaceID: web.workspace, ProjectID: web.id}); err != nil || !found || !reflect.DeepEqual(got, want) {
			t.Errorf("ShareIssue(archived) = %+v, %v, %v; want %+v", got, found, err, want)
		}
		for _, gone := range []uuid.UUID{deleted, uuid.NewV7()} {
			if got, found, err := s.ShareIssue(ctx, gone); err != nil || found || got != (app.SharedIssue{}) {
				t.Errorf("ShareIssue(%s) = %+v, %v, %v; want nothing", gone, got, found, err)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// ShareIssue and a write of the work item's row wait for each other (M7
// P1a spec 2.4). Held by ShareIssue, the row's UPDATE waits until its
// transaction ends. Held by an UPDATE, ShareIssue waits for it, then reads
// the row's new version: a deletion that committed meanwhile answers
// nothing; any other write, the work item.
func TestShareIssueAndAWriteOfTheRowWaitForEachOther(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	t.Run("an update waits for ShareIssue", func(t *testing.T) {
		id := uuid.NewV7()
		newIssue(t, pool, web, id, 1, alice)
		updated := make(chan error, 1)
		err := postgres.NewTxManager(pool, 5*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
			if _, found, err := s.ShareIssue(ctx, id); err != nil || !found {
				t.Fatalf("ShareIssue() = %v, %v", found, err)
			}
			go func() {
				_, err := pool.Exec(pgtest.Soon(t), "UPDATE issues SET name = 'Mend' WHERE id = $1", id)
				updated <- err
			}()
			pgtest.WaitForLockWaitOn(t, pool, "issues", 5*time.Second)
			select {
			case err := <-updated:
				t.Errorf("the update ended (%v) while ShareIssue held the row", err)
			default:
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		select {
		case err := <-updated:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("the update did not end once ShareIssue's transaction had")
		}
	})
	for n, tt := range []struct {
		name, write string
		found       bool
	}{
		{"a deletion committed meanwhile", "UPDATE issues SET deleted_at = $2, updated_at = $2 WHERE id = $1", false},
		{"an archiving committed meanwhile", "UPDATE issues SET archived_at = $2, updated_at = $2 WHERE id = $1", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			id := uuid.NewV7()
			newIssue(t, pool, web, id, n+2, alice)
			soon := pgtest.Soon(t)
			held, err := pool.Begin(soon)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = held.Rollback(context.Background()) }()
			if _, err := held.Exec(soon, tt.write, id, now); err != nil {
				t.Fatal(err)
			}
			type answer struct {
				found bool
				err   error
			}
			shared := make(chan answer, 1)
			go func() {
				var a answer
				a.err = postgres.NewTxManager(pool, 5*time.Second).WithinTx(soon, func(ctx context.Context) error {
					_, a.found, a.err = s.ShareIssue(ctx, id)
					return a.err
				})
				shared <- a
			}()
			pgtest.WaitForLockWaitOn(t, pool, "issues", 5*time.Second)
			if err := held.Commit(soon); err != nil {
				t.Fatal(err)
			}
			select {
			case a := <-shared:
				if a.err != nil || a.found != tt.found {
					t.Errorf("ShareIssue() found %v, %v; want found %v", a.found, a.err, tt.found)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("ShareIssue did not end once the write had committed")
			}
		})
	}
}

// RemoveSubscriber (M7 P1a spec 2.4) soft-deletes the account's one
// undeleted row of the work item's subscribers, at the moment and by the
// account given. His row deleted before keeps its deletion; another
// account's row and his row of another work item are not touched; and a
// second call, with none of his rows left, changes nothing.
func TestRemoveSubscriberDeletesTheOneUndeletedRow(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	id, other := uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, web, id, 1, alice)
	newIssue(t, pool, web, other, 2, alice)
	earlier := now.Add(-time.Hour)
	old, live, elsewhere := linkRows(web, id, alice, earlier, bob), linkRows(web, id, alice, earlier, bob, carol), linkRows(web, other, alice, earlier, bob)
	if err := s.InsertSubscribers(context.Background(), old); err != nil {
		t.Fatal(err)
	}
	exec(t, pool, "UPDATE issue_subscribers SET deleted_at = $2 WHERE id = $1", old.IDs[0], earlier)
	for _, r := range []app.LinkRows{live, elsewhere} {
		if err := s.InsertSubscribers(context.Background(), r); err != nil {
			t.Fatal(err)
		}
	}
	untouched := "undeleted, written at " + earlier.UTC().Format(time.RFC3339Nano) + " by " + alice.String()
	want := map[uuid.UUID]string{old.IDs[0]: deletion(earlier, alice), live.IDs[0]: deletion(now, bob), live.IDs[1]: untouched,
		elsewhere.IDs[0]: untouched}
	for _, at := range []time.Time{now, now.Add(time.Hour)} {
		if err := s.RemoveSubscriber(context.Background(), app.SubscriberRemoval{IssueID: id, SubscriberID: bob, By: bob, At: at}); err != nil {
			t.Fatal(err)
		}
		if got := audits(t, pool, "issue_subscribers"); !maps.Equal(got, want) {
			t.Errorf("after RemoveSubscriber at %s:\n%v\nwant\n%v", at, got, want)
		}
	}
}

// IsSubscribed (M7 P1a spec R5) answers whether the account has an
// undeleted row of the work item's subscribers: not for his deleted row
// alone, nor for another account's row, nor for his row of another work
// item.
func TestIsSubscribedReadsAnUndeletedRow(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	id, other := uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, web, id, 1, alice)
	newIssue(t, pool, web, other, 2, alice)
	unsubscribed := linkRows(web, id, alice, now, bob)
	for _, r := range []app.LinkRows{linkRows(web, id, alice, now, alice), unsubscribed, linkRows(web, other, alice, now, carol)} {
		if err := s.InsertSubscribers(context.Background(), r); err != nil {
			t.Fatal(err)
		}
	}
	exec(t, pool, "UPDATE issue_subscribers SET deleted_at = $2 WHERE id = $1", unsubscribed.IDs[0], now)
	for _, tt := range []struct {
		name           string
		issue, account uuid.UUID
		want           bool
	}{
		{"an undeleted row", id, alice, true},
		{"a deleted row alone", id, bob, false},
		{"a row of another work item alone", id, carol, false},
		{"another account's row alone", other, alice, false},
	} {
		if got, err := s.IsSubscribed(context.Background(), tt.issue, tt.account); err != nil || got != tt.want {
			t.Errorf("%s: IsSubscribed() = %v, %v; want %v", tt.name, got, err, tt.want)
		}
	}
}
