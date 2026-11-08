package bootstrap

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	assetriver "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/river"
	identityriver "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/river"
	issueriver "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func publishPath(id uuid.UUID) string { return draftPath(id) + "/publish" }

// Catches request-body substitution, using an earlier saved payload, retaining
// the draft, and duplicate publication. Database lock races extend this below.
func TestDraftPublishUsesStoredPayload(t *testing.T) {
	w := newIssueWorld(t)
	d := createDraftHTTP(t, w, "alice", `{"project_id":"`+w.web.String()+`","payload":{"name":"Old"}}`)
	draftRequest(t, w, "alice", "PATCH", draftPath(d.ID), `{"payload":{"name":"Locked new"}}`, 200)
	before := draftState(t, w)
	draftRequest(t, w, "alice", "POST", publishPath(d.ID), `{"name":"Injected"}`, 400)
	if !reflect.DeepEqual(before, draftState(t, w)) {
		t.Fatal("body refusal wrote rows")
	}
	var got storedIssue
	decodeAnswer(t, draftRequest(t, w, "alice", "POST", publishPath(d.ID), "", 201), &got)
	if got.Name != "Locked new" || got.SequenceID != 1 {
		t.Fatal(got)
	}
	draftRequest(t, w, "alice", "GET", draftPath(d.ID), "", 404)
	after := draftState(t, w)
	draftRequest(t, w, "alice", "POST", publishPath(d.ID), "", 404)
	if !reflect.DeepEqual(after, draftState(t, w)) {
		t.Fatal("repeat publish wrote rows")
	}
}

// Saving remains permissive; publishing has exactly the same problem object
// as create for both shape failures and aggregate business/reference failures,
// a cycle's and modules' too, with Web's cycles and modules turned on (M6 P3a
// spec 2.3). A cycle or modules whose shape create refuses cannot be saved:
// saving refuses them as create does, each problem under payload.
func TestDraftPublishMatchesCreateBodyErrors(t *testing.T) {
	w := newIssueWorld(t)
	w.turnCyclesOn(t, w.web)
	w.turnModulesOn(t, w.web)
	ids := make([]uuid.UUID, 101)
	for i := range ids {
		ids[i] = uuid.NewV7()
	}
	cycle, module := uuid.NewV7(), uuid.NewV7()
	for _, tc := range []struct {
		body   string
		status int
	}{
		{`{}`, 400},
		{fmt.Sprintf(`{"name":"Refs","state_id":%q,"label_ids":[%q],"parent_id":%q,"assignee_ids":[%q]}`, uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), w.ids["olga"]), 422},
		{`{"name":"` + strings.Repeat("x", 256) + `","priority":"bad","label_ids":` + idArray(ids...) + `}`, 422},
		{fmt.Sprintf(`{"cycle_id":%q,"module_ids":[%q]}`, cycle, module), 400},
		{fmt.Sprintf(`{"name":"Refs","label_ids":[%q],"cycle_id":%q,"module_ids":[%q,%q]}`, uuid.NewV7(), cycle, module, w.bug), 422},
		{`{"name":"Many","module_ids":` + idArray(ids...) + `}`, 422},
		{fmt.Sprintf(`{"name":"Twice","module_ids":[%q,%q,%q]}`, module, cycle, module), 422},
	} {
		d := createDraftHTTP(t, w, "alice", `{"project_id":"`+w.web.String()+`","payload":`+tc.body+`}`)
		before := draftState(t, w)
		got := draftRequest(t, w, "alice", "POST", publishPath(d.ID), "", tc.status)
		want := draftRequest(t, w, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", tc.body, tc.status)
		if !reflect.DeepEqual(problemObject(t, got), problemObject(t, want)) {
			t.Fatal(got, want)
		}
		if !reflect.DeepEqual(before, draftState(t, w)) {
			t.Fatal("validation wrote rows")
		}
	}
	for _, body := range []string{`{"name":"Fix","cycle_id":null}`, `{"name":"Fix","cycle_id":"Sprint"}`, `{"name":"Fix","module_ids":null}`,
		`{"name":"Fix","module_ids":[null]}`, `{"name":"Fix","module_ids":["Auth"]}`, `{"name":"Fix","module_ids":{}}`} {
		before := draftState(t, w)
		saved := problemObject(t, draftRequest(t, w, "alice", http.MethodPost, draftsPath, `{"project_id":"`+w.web.String()+`","payload":`+body+`}`, 400))
		want := problemObject(t, draftRequest(t, w, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", body, 400))
		for _, e := range want["errors"].([]any) {
			e.(map[string]any)["field"] = "payload." + e.(map[string]any)["field"].(string)
		}
		if !reflect.DeepEqual(saved, want) {
			t.Errorf("saving %s = %v, want create's refusal under payload: %v", body, saved, want)
		}
		if !reflect.DeepEqual(before, draftState(t, w)) {
			t.Fatalf("the refusal of %s wrote rows", body)
		}
	}
}

// All persistence, including project counter and queued event, must roll back
// at each failure point. A 500 alone would miss an independently committed create.
func TestDraftPublishRollsBackWithCreation(t *testing.T) {
	for _, fault := range []string{"event", "soft delete", "after sequence"} {
		t.Run(fault, func(t *testing.T) {
			w := newIssueWorld(t)
			body := fmt.Sprintf(`{"project_id":%q,"payload":{"name":"Atomic","description_html":%q,"assignee_ids":[%q],"label_ids":[%q]}}`, w.web, mentionHTML(w.ids["bob"]), w.ids["bob"], w.bug)
			d := createDraftHTTP(t, w, "alice", body)
			var events []shared.Event
			var failure error
			route := issueEventRoute(t, w, clockAt(time.Date(2044, 5, 6, 7, 8, 9, 0, time.UTC)), &events, &failure)
			switch fault {
			case "event":
				failure = errors.New("event rejected after River insertion")
			case "soft delete":
				_, err := w.pool.Exec(pgtest.Soon(t), `CREATE FUNCTION reject_publish_delete() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.deleted_at IS NOT NULL THEN RAISE EXCEPTION 'draft delete rejected'; END IF; RETURN NEW; END $$; CREATE TRIGGER reject_publish_delete BEFORE UPDATE ON draft_issues FOR EACH ROW EXECUTE FUNCTION reject_publish_delete()`)
				if err != nil {
					t.Fatal(err)
				}
			case "after sequence":
				_, err := w.pool.Exec(pgtest.Soon(t), `ALTER TABLE issue_description_versions ADD CONSTRAINT reject_publish_version CHECK (description_html NOT LIKE '%Hello%')`)
				if err != nil {
					t.Fatal(err)
				}
			}
			before := publishSnapshot(t, w)
			route.answerWithin(t, w.contract, "POST", publishPath(d.ID), w.ids["alice"], "", 500)
			if after := publishSnapshot(t, w); !reflect.DeepEqual(before, after) {
				for table, rows := range before {
					if after[table] != rows {
						t.Errorf("%s changed after %s failure", table, fault)
					}
				}
			}
			wantEvents := 1
			if fault == "after sequence" {
				wantEvents = 0
			}
			if len(events) != wantEvents {
				t.Fatalf("events=%d want %d", len(events), wantEvents)
			}
		})
	}
}

// A refused request must preserve all business rows and every non-periodic
// job's insertion identity and payload, including test event probes and unknown
// kinds. Only the complete signature of a declared periodic schedule is noise;
// worker attempt/state/timing updates are not writes by the request.
func publishSnapshot(t *testing.T, w issueWorld) map[string]string {
	t.Helper()
	out := draftState(t, w)
	var projects, queued string
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY id),'[]'::jsonb)::text FROM projects r").Scan(&projects); err != nil {
		t.Fatal(err)
	}
	periodic := []string{assetriver.CleanupKind, identityriver.CleanupKind, issueriver.AutoArchiveKind, issueriver.TrimVersionsKind, (purgeArgs{}).Kind()}
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT coalesce(jsonb_agg(to_jsonb(j) ORDER BY id),'[]'::jsonb)::text
 FROM (SELECT id,kind,args,queue,max_attempts,priority,tags,created_at FROM river_job
 WHERE NOT coalesce(kind=ANY($1::text[]) AND queue='periodic' AND args='{}'::jsonb
 AND metadata @> jsonb_build_object('periodic',true,'river:periodic_job_id',kind),false)) j`, periodic).Scan(&queued); err != nil {
		t.Fatal(err)
	}
	out["projects"], out["river_job"] = projects, queued
	return out
}
