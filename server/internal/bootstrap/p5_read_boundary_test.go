package bootstrap

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

func TestP5ReadBoundary(t *testing.T) {
	t.Run("full-gate", p5ReadsThroughFullGate)
	t.Run("no-for-locks", p5ReadsWithoutRowLocks)
	t.Run("snapshot-connection", p5ReadsOnSnapshotConnection)
}

// The production app, not a replacement gate, admits one blocked sort write.
// A second legal write proves the gate is full while reads still finish.
func p5ReadsThroughFullGate(t *testing.T) {
	w := newIssueWorldWith(t, func(cfg *config.Config) {
		cfg.Server.RequestTimeout = 30 * time.Second
		cfg.Issue.WriteGate = config.WriteGateConfig{PerProject: 1, MaxWait: time.Minute}
		cfg.Issue.WriteLockTimeout = 25 * time.Second
	})
	hot := w.newWorkItem(t)
	archived := w.doneWorkItem(t)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+archived.String()+"/archive", "", 200)
	path := "/api/v0/issues/" + hot.String()
	row := holding(t, w.pool, "SELECT 1 FROM projects WHERE id=$1 FOR UPDATE", w.web)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request := newRequest(t, http.MethodPatch, w.base+path, w.tokens["alice"], []byte(`{"sort_order":19}`)).WithContext(ctx)
	w.contract.CheckRequest(t, request)
	written := sendInBackground(request)
	collected := false
	defer func() {
		if row != nil {
			release(t, row)
		}
		cancel()
		if !collected {
			receiveWithin(t, written, 5*time.Second, "blocked sort cleanup")
		}
	}()
	pgtest.WaitForLockWaitBehind(t, w.pool, row.Conn().PgConn(), 5*time.Second)
	second := newRequest(t, http.MethodPatch, w.base+path, w.tokens["bob"], []byte(`{"sort_order":37}`))
	w.contract.CheckRequest(t, second)
	res, body := send(t, second)
	w.wantBusy(t, "the second sort write", second, res, body)
	project := "/api/v0/projects/" + w.web.String() + "/issues"
	for _, read := range []struct {
		path string
		want string
	}{
		{project, hot.String()},
		{project + "?archived=true", archived.String()},
		{memberIssuesPath(w, "alice") + "?relation=created", hot.String()},
	} {
		p := checkListEnvelope(t, w.send(t, "alice", http.MethodGet, read.path, "", 200), false, false)
		if p.Total != 1 || len(p.Data) != 1 || p.Data[0].ID.String() != read.want || p.NextCursor != nil {
			t.Fatalf("read while gate full: %+v, want %s", p, read.want)
		}
	}
	// The holder and first write must still be present after all reads.
	pgtest.WaitForLockWaitBehind(t, w.pool, row.Conn().PgConn(), 5*time.Second)
	select {
	case a := <-written:
		collected = true
		t.Fatalf("sort write ended before holder released: %v %s", a.err, a.body)
	default:
	}
	if stored := sortStored(t, w, hot); stored.SortOrder == 19 || stored.SortOrder == 37 {
		t.Fatal("blocked/refused sort already stored", stored)
	}
	release(t, row)
	row = nil
	a := receiveWithin(t, written, 5*time.Second, "sort after holder released")
	collected = true
	if a.err != nil {
		t.Fatal(a.err)
	}
	w.contract.CheckResponse(t, request, a.res)
	if a.res.StatusCode != 200 || sortAnswer(t, string(a.body)).SortOrder != 19 || sortStored(t, w, hot).SortOrder != 19 {
		t.Fatal("sort did not use the original write path", a.res.StatusCode, string(a.body))
	}
}

func p5ReadsWithoutRowLocks(t *testing.T) {
	w := newIssueWorld(t)
	id := w.newWorkItem(t)
	project := "/api/v0/projects/" + w.web.String() + "/issues"
	// EXCLUSIVE admits AccessShare (ordinary SELECT), but conflicts with
	// every SELECT FOR mode's RowShare, as well as writes' RowExclusive.
	// users supplies profile names; it is deliberately absent from issueTables.
	tables := lockedTables(t, w.pool, issueTables+", users, project_user_properties", "EXCLUSIVE")
	defer release(t, tables)
	for _, query := range []string{"", "?group_by=label_id", "?group_by=label_id&sub_group_by=assignee_id"} {
		p := checkListEnvelope(t, w.send(t, "alice", http.MethodGet, project+query, "", 200), query != "", query == "?group_by=label_id&sub_group_by=assignee_id")
		if p.Total != 1 {
			t.Fatal(p)
		}
	}
	memberListIDs(t, w, "bob", "alice", "created", "", id)
	// Preferences GET is an ordinary read, not an issue-list RR transaction.
	samePreferencesJSON(t, w.send(t, "alice", http.MethodGet, "/api/v0/me/projects/"+w.web.String()+"/preferences", "", 200),
		preferencesJSON("work_items", "[]", 65535, `{}`, defaultDisplay, defaultProperties))
}
