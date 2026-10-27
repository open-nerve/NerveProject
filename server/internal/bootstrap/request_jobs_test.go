package bootstrap

import (
	"slices"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// A rejected request must not hide a newly enqueued job or a replaced payload.
// Worker progress and periodic scheduling must not look like request writes.
func TestRegistrationRowsSeeRequestJobs(t *testing.T) {
	t.Run("declared request kinds", func(t *testing.T) {
		if kinds := append(identity.EnqueuedKinds(), issue.EnqueuedKinds()...); !slices.Contains(kinds, "issue.crawl_link") || !slices.Contains(kinds, "issue.delete_rows") {
			t.Fatalf("request kinds=%v; want independently expected issue.crawl_link and issue.delete_rows", kinds)
		}
	})
	pool := openPool(t, pgtest.NewDatabase(t))
	exec := func(sql string) {
		t.Helper()
		if _, err := pool.Exec(pgtest.Soon(t), sql); err != nil {
			t.Fatal(err)
		}
	}
	before := registrationRows(t, pool)
	exec(`INSERT INTO river_job(id,kind,args,max_attempts) VALUES(41,'issue.delete_rows','{"scope":"workspace_drafts","id":"0199a2b4-0000-7000-8000-000000000001","at":"2026-10-28T01:00:00Z","by":"0199a2b4-0000-7000-8000-000000000002"}',25)`)
	insertedDeletion := registrationRows(t, pool)
	for _, part := range []string{`"id":41`, `"kind":"issue.delete_rows"`, `"scope": "workspace_drafts"`, `"at": "2026-10-28T01:00:00Z"`, `"by": "0199a2b4-0000-7000-8000-000000000002"`, `"id": "0199a2b4-0000-7000-8000-000000000001"`} {
		if !strings.Contains(insertedDeletion, part) {
			t.Errorf("deletion snapshot lacks %s", part)
		}
	}
	exec(`UPDATE river_job SET args=jsonb_set(args,'{scope}','"project"') WHERE id=41`)
	replacedDeletion := registrationRows(t, pool)
	if before == insertedDeletion || insertedDeletion == replacedDeletion || !strings.Contains(replacedDeletion, `"scope": "project"`) {
		t.Fatal("deletion insert/replacement invisible")
	}
	before = replacedDeletion
	exec(`INSERT INTO river_job (id,kind,args,max_attempts) VALUES
		(29,'issue.crawl_link','{"link_id":"0199a2b4-0000-7000-8000-000000000001","url":"http://169.254.169.254/first"}',3)`)
	inserted := registrationRows(t, pool)
	t.Run("inserting a request", func(t *testing.T) {
		if before == inserted {
			t.Fatal("inserting issue.crawl_link did not change registrationRows")
		}
		for _, part := range []string{`"id":29`, `"kind":"issue.crawl_link"`, `"link_id": "0199a2b4-0000-7000-8000-000000000001"`, `"url": "http://169.254.169.254/first"`} {
			if !strings.Contains(inserted, part) {
				t.Errorf("request snapshot lacks stable identity/payload %s: %s", part, inserted)
			}
		}
	})
	exec(`UPDATE river_job SET args=jsonb_set(args,'{url}','"http://169.254.169.254/replaced"') WHERE id=29`)
	replaced := registrationRows(t, pool)
	t.Run("replacing args of the same job", func(t *testing.T) {
		if inserted == replaced || !strings.Contains(replaced, "http://169.254.169.254/replaced") || strings.Contains(replaced, "http://169.254.169.254/first") {
			t.Fatal("replacing request args at the same id did not change registrationRows to the new payload")
		}
	})
	exec(`UPDATE river_job SET state='completed',attempt=2,attempted_at=now(),finalized_at=now(),attempted_by=ARRAY['worker'] WHERE id=29`)
	t.Run("worker progress", func(t *testing.T) {
		if after := registrationRows(t, pool); after != replaced {
			t.Fatalf("worker progress changed registrationRows:\n%s\nwas:\n%s", after, replaced)
		}
	})
	exec(`INSERT INTO river_job (id,kind,args,max_attempts) VALUES
		(7,'issue.crawl_link','{"link_id":"0199a2b4-0000-7000-8000-000000000002","url":"http://169.254.169.254/second"}',3)`)
	replaced = registrationRows(t, pool)
	t.Run("stable identities ordered by id", func(t *testing.T) {
		first, second := strings.Index(replaced, `"id":7,"kind":"issue.crawl_link"`), strings.Index(replaced, `"id":29,"kind":"issue.crawl_link"`)
		if first < 0 || second < 0 || first >= second {
			t.Fatalf("request identities are missing or not ordered by id: %s", replaced)
		}
	})
	exec(`INSERT INTO river_job (kind,args,max_attempts,queue) VALUES
		('identity.cleanup_expired_sessions','{}',3,'periodic'),('issue.auto_archive','{}',3,'periodic'),('issue.trim_description_versions','{}',25,'periodic'),('cleanup.purge_deleted','{}',25,'periodic')`)
	exec(`INSERT INTO river_leader (leader_id,elected_at,expires_at) VALUES ('new-leader',now(),now()+interval '1 minute')`)
	t.Run("periodic and leader noise", func(t *testing.T) {
		if after := registrationRows(t, pool); after != replaced {
			t.Fatalf("periodic jobs/leader changed registrationRows:\n%s\nwas:\n%s", after, replaced)
		}
	})
}
