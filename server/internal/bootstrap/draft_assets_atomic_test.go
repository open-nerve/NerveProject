package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/riverqueue/river"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Real issue subscribers precede this probe; a failure must undo their activity
// as well as the River insert, all migrated rows and the sole creation graph.
func draftAssetEventRoute(t *testing.T, w issueWorld, clock *eventCountingClock, draft uuid.UUID, ids []uuid.UUID, events *[]shared.Event, fault *error) moduleRoute {
	t.Helper()
	client, err := jobs.NewInserter(w.pool)
	if err != nil {
		t.Fatal(err)
	}
	inserter := issueInserter{client: client}
	subscribers := issue.Subscribers(issue.Provide(w.pool, nil), inserter)
	subscribers = append(subscribers, shared.EventSubscriber{Name: "draft migration probe", Types: []string{"issue.created", "asset.attachment_added"}, Handle: func(ctx context.Context, e shared.Event) error {
		if _, ok := postgres.Tx(ctx); !ok {
			t.Error("draft event escaped request transaction")
		}
		*events = append(*events, e)
		if e.Type == "asset.attachment_added" {
			if len(e.Changes) != 1 || e.Changes[0].Field != "issue" || e.Changes[0].NewID == nil || e.Changes[0].OldID != nil || e.Changes[0].Old != nil || e.Changes[0].New != nil || e.Entity.Kind != "asset" || e.ActorID == nil || *e.ActorID != w.ids["alice"] || e.ProjectID != w.web || e.WorkspaceID != w.acme {
				t.Error("attachment event lost exact ID-only envelope", e)
			}
			for _, id := range ids {
				row, err := assetpg.New(w.pool).FindAsset(ctx, id)
				if err != nil {
					return err
				}
				if row.Placement.IssueID != *e.Changes[0].NewID || row.Placement.DraftID != uuid.Nil() || !row.UpdatedAt.Equal(e.OccurredAt) {
					t.Error("attachment event preceded complete migration", row)
				}
				outside, err := assetpg.New(w.pool).FindAsset(t.Context(), id)
				if err != nil {
					return err
				}
				if outside.Placement.DraftID != draft {
					t.Error("migration visible before commit")
				}
			}
		}
		if err := inserter.InsertTx(ctx, issueEventProbeArgs{Event: e}, &river.InsertOpts{Queue: "issue_events"}); err != nil {
			return err
		}
		if e.Type == "asset.attachment_added" {
			return *fault
		}
		return nil
	}})
	dispatcher, err := newDispatcher(eventTypes(), subscribers)
	if err != nil {
		t.Fatal(err)
	}
	return newIssueRouteWithPublisher(t, w.pool, postgres.NewTxManager(w.pool, 2*time.Second), clock, dispatcher)
}

func TestDraftAssetPublishRollback(t *testing.T) {
	for _, stage := range []string{"sequence", "issue", "version", "second asset", "draft", "event", "commit"} {
		t.Run(stage, func(t *testing.T) {
			w := newIssueWorld(t)
			d := createDraftHTTP(t, w, "alice", fmt.Sprintf(`{"project_id":%q,"payload":{"name":"Atomic","description_html":%q,"label_ids":[%q],"assignee_ids":[%q]}}`, w.web, mentionHTML(w.ids["bob"]), w.bug, w.ids["bob"]))
			a, b := draftAsset(t, w, d.ID, false), draftAsset(t, w, d.ID, false)
			image := draftAsset(t, w, d.ID, true)
			ids := []uuid.UUID{a.Asset.ID, b.Asset.ID, image.Asset.ID}
			at := time.Date(2046, 7, 8, 9, 10, 11, 123456000, time.UTC)
			clock := &eventCountingClock{base: clockAt(at)}
			var events []shared.Event
			var fault error
			route := draftAssetEventRoute(t, w, clock, d.ID, ids, &events, &fault)
			table, operation, threshold := "file_assets", "UPDATE", 1
			if stage == "event" {
				fault = errors.New("reject migrated attachment after activity and job")
			} else {
				switch stage {
				case "sequence":
					table = "projects"
				case "issue":
					table, operation = "issues", "INSERT"
				case "version":
					table, operation = "issue_description_versions", "INSERT"
				case "second asset":
					threshold = 2
				case "draft":
					table = "draft_issues"
				}
				fn := fmt.Sprintf(`CREATE SEQUENCE draft_asset_fault; CREATE FUNCTION reject_draft_asset() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF nextval('draft_asset_fault') >= %d THEN RAISE EXCEPTION 'draft asset fault'; END IF; RETURN NEW; END $$;`, threshold)
				trigger := "CREATE TRIGGER reject_draft_asset AFTER " + operation + " ON " + table + " FOR EACH ROW EXECUTE FUNCTION reject_draft_asset()"
				if stage == "commit" {
					trigger = "CREATE CONSTRAINT TRIGGER reject_draft_asset AFTER UPDATE ON file_assets DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_draft_asset()"
				}
				if _, err := w.pool.Exec(t.Context(), fn+trigger); err != nil {
					t.Fatal(err)
				}
			}
			before := draftAssetSnapshot(t, w)
			route.answerWithin(t, w.contract, "POST", publishPath(d.ID), w.ids["alice"], "", 500)
			if !reflect.DeepEqual(before, draftAssetSnapshot(t, w)) || issueEventCount(t, w) != 0 {
				t.Fatal("failure committed part of graph, sequence, audit, event or asset")
			}
			var clockReads int32 = 1
			if stage == "sequence" {
				clockReads = 0
			}
			if clock.calls.Load() != clockReads {
				t.Fatal("wrong clock position/count", clock.calls.Load())
			}
			if stage != "event" {
				var called bool
				if err := w.pool.QueryRow(t.Context(), "SELECT is_called FROM draft_asset_fault").Scan(&called); err != nil || !called {
					t.Fatal("fault not reached", err)
				}
				if _, err := w.pool.Exec(t.Context(), "DROP TRIGGER reject_draft_asset ON "+table); err != nil {
					t.Fatal(err)
				}
			}
			fault, events = nil, nil
			var got storedIssue
			decodeAnswer(t, route.answerWithin(t, w.contract, "POST", publishPath(d.ID), w.ids["alice"], "", 201), &got)
			if got.SequenceID != 1 || !got.CreatedAt.Equal(at) || len(events) != 3 || issueEventCount(t, w) != 3 || attachmentActivityCount(t, w, got.ID) != 2 || clock.calls.Load() != clockReads+1 {
				t.Fatal("retry did not publish once with one time", got, events, clock.calls.Load())
			}
			want := []uuid.UUID{a.Asset.ID, b.Asset.ID}
			slices.SortFunc(want, uuid.UUID.Compare)
			if events[0].Type != "issue.created" || events[1].Entity.ID != want[0] || events[2].Entity.ID != want[1] {
				t.Fatal("attachment event order", events)
			}
		})
	}
}

func TestDraftAssetsMoveRollback(t *testing.T) {
	for _, stage := range []string{"second asset", "commit"} {
		t.Run(stage, func(t *testing.T) {
			w := newIssueWorld(t)
			d := createDraftHTTP(t, w, "alice", `{"payload":{}}`)
			draftAsset(t, w, d.ID, false)
			draftAsset(t, w, d.ID, true)
			threshold := 2
			trigger := "CREATE TRIGGER reject_draft_move AFTER UPDATE ON file_assets FOR EACH ROW EXECUTE FUNCTION reject_draft_move()"
			if stage == "commit" {
				threshold = 1
				trigger = "CREATE CONSTRAINT TRIGGER reject_draft_move AFTER UPDATE ON file_assets DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_draft_move()"
			}
			fn := fmt.Sprintf(`CREATE SEQUENCE draft_move_fault; CREATE FUNCTION reject_draft_move() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF nextval('draft_move_fault') >= %d THEN RAISE EXCEPTION 'move fault'; END IF; RETURN NEW; END $$;`, threshold)
			if _, err := w.pool.Exec(t.Context(), fn+trigger); err != nil {
				t.Fatal(err)
			}
			before := draftAssetSnapshot(t, w)
			body := fmt.Sprintf(`{"project_id":%q}`, w.web)
			draftRequest(t, w, "alice", "PATCH", draftPath(d.ID), body, 500)
			var called bool
			if err := w.pool.QueryRow(t.Context(), "SELECT is_called FROM draft_move_fault").Scan(&called); err != nil || !called || !reflect.DeepEqual(before, draftAssetSnapshot(t, w)) {
				t.Fatal("move fault not atomic or not reached", err)
			}
			if _, err := w.pool.Exec(t.Context(), "DROP TRIGGER reject_draft_move ON file_assets"); err != nil {
				t.Fatal(err)
			}
			draftRequest(t, w, "alice", "PATCH", draftPath(d.ID), body, 200)
		})
	}
}
