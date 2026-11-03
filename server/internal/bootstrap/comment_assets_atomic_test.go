package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestCommentAssetsRollback(t *testing.T) {
	for _, edit := range []bool{false, true} {
		for _, stage := range []string{"comment", "second asset", "commit", "subscriber"} {
			t.Run(fmt.Sprintf("edit=%t/%s", edit, stage), func(t *testing.T) {
				w := newIssueWorld(t)
				item := w.newWorkItem(t)
				kind, host, method, path, event := "new_comment", item, "POST", commentsOf(item), "issue.comment_created"
				if edit {
					kind, host, method, event = "comment", w.newComment(t, "bob", item).ID, "PATCH", "issue.comment_updated"
					path = commentPath(host)
				}
				a := commentAsset(t, w, "bob", kind, host, true)
				b := commentAsset(t, w, "bob", kind, host, true)
				body := commentAssetsBody(t, issueImageHTML(a.Asset.ID, b.Asset.ID)+mentionHTML(w.ids["walt"]))
				before := contentAssetState(t, w)
				at := time.Date(2045, 6, 7, 8, 9, 10, 123456000, time.UTC)
				clock := &eventCountingClock{base: clockAt(at)}
				var events []shared.Event
				var failure error
				if stage == "subscriber" {
					failure = errors.New("comment binding subscriber failure")
				}
				route := issueEventRouteFor(t, w, clock, &events, &failure, []string{event}, func(ctx context.Context, e shared.Event) error {
					for _, id := range []uuid.UUID{a.Asset.ID, b.Asset.ID} {
						row, err := assetpg.New(w.pool).FindAsset(ctx, id)
						if err != nil {
							return err
						}
						if row.BoundAt == nil || !row.BoundAt.Equal(e.OccurredAt) || row.Placement.CommentID != e.Entity.ID {
							t.Error("comment event preceded binding or used another moment")
						}
						outside, err := assetpg.New(w.pool).FindAsset(t.Context(), id)
						if err != nil {
							return err
						}
						if outside.BoundAt != nil {
							t.Error("binding visible before commit")
						}
					}
					return nil
				})
				table, operation := "file_assets", "UPDATE"
				if stage != "subscriber" {
					if stage == "comment" {
						table = "issue_comments"
						if !edit {
							operation = "INSERT"
						}
					}
					threshold := 1
					if stage == "second asset" {
						threshold = 2
					}
					fn := fmt.Sprintf(`CREATE SEQUENCE comment_asset_fault; CREATE FUNCTION reject_comment_asset() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF nextval('comment_asset_fault') >= %d THEN RAISE EXCEPTION 'comment asset fault'; END IF; RETURN NEW; END $$;`, threshold)
					trigger := "CREATE TRIGGER reject_comment_asset AFTER " + operation + " ON " + table + " FOR EACH ROW EXECUTE FUNCTION reject_comment_asset()"
					if stage == "commit" {
						trigger = "CREATE CONSTRAINT TRIGGER reject_comment_asset AFTER UPDATE ON file_assets DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_comment_asset()"
					}
					if _, err := w.pool.Exec(t.Context(), fn+trigger); err != nil {
						t.Fatal(err)
					}
				}
				route.answerWithin(t, w.contract, method, path, w.ids["bob"], body, 500)
				if clock.calls.Load() != 1 {
					t.Fatal("comment did not use exactly one locked business moment", clock.calls.Load())
				}
				if stage != "subscriber" {
					var reached bool
					if err := w.pool.QueryRow(t.Context(), "SELECT is_called FROM comment_asset_fault").Scan(&reached); err != nil || !reached {
						t.Fatal("fault statement not reached", err)
					}
					if _, err := w.pool.Exec(t.Context(), "DROP TRIGGER reject_comment_asset ON "+table); err != nil {
						t.Fatal(err)
					}
				}
				if !reflect.DeepEqual(before, contentAssetState(t, w)) || issueEventCount(t, w) != 0 {
					t.Fatal("failure left a partial comment, claim, audit, subscriber or task")
				}
				failure = nil
				events = nil
				status := 201
				if edit {
					status = 200
				}
				var got answeredComment
				decodeAnswer(t, route.answerWithin(t, w.contract, method, path, w.ids["bob"], body, status), &got)
				if len(events) != 1 || issueEventCount(t, w) != 1 || clock.calls.Load() != 2 || !got.UpdatedAt.Equal(at) {
					t.Fatal("retry did not commit exactly once", events, got, clock.calls.Load())
				}
				if attachmentActivityCount(t, w, item) != 0 {
					t.Fatal("comment claim produced issue attachment activity")
				}
			})
		}
	}
}

func TestCommentAssetsUnchangedLegacyReference(t *testing.T) {
	w := newIssueWorld(t)
	item := w.newWorkItem(t)
	c := w.newComment(t, "bob", item)
	image := commentAsset(t, w, "bob", "comment", c.ID, true)
	html := issueImageHTML(image.Asset.ID)
	// Only historical state can have stored HTML referring to an unbound image.
	if _, err := w.pool.Exec(t.Context(), "UPDATE issue_comments SET comment_html=$2 WHERE id=$1", c.ID, html); err != nil {
		t.Fatal(err)
	}
	before := collaborationState(t, w)
	w.send(t, "bob", "PATCH", commentPath(c.ID), commentAssetsBody(t, html), 200)
	row, err := assetpg.New(w.pool).FindAsset(t.Context(), image.Asset.ID)
	if err != nil || row.BoundAt == nil {
		t.Fatal("unchanged comment skipped reference binding", err)
	}
	if !reflect.DeepEqual(before, collaborationState(t, w)) {
		t.Fatal("legacy binding changed comment audit or published another action")
	}
	other := w.newComment(t, "bob", item)
	private := commentAsset(t, w, "bob", "comment", other.ID, true)
	html = issueImageHTML(private.Asset.ID)
	if _, err := w.pool.Exec(t.Context(), "UPDATE issue_comments SET comment_html=$2 WHERE id=$1", c.ID, html); err != nil {
		t.Fatal(err)
	}
	old := contentAssetState(t, w)
	w.send(t, "bob", "PATCH", commentPath(c.ID), commentAssetsBody(t, html), 404)
	if !reflect.DeepEqual(old, contentAssetState(t, w)) {
		t.Fatal("unchanged cross-comment reference changed state")
	}
}
