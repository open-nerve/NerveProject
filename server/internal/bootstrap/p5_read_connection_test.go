package bootstrap

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

func p5ReadsOnSnapshotConnection(t *testing.T) {
	w := newIssueWorld(t)
	parent := w.newWorkItem(t)
	w.adminGuest(t)
	own := w.newWorkItemBy(t, "walt", w.web, `{"name":"Own","label_ids":`+idArray(w.bug)+`,"assignee_ids":`+idArray(w.ids["bob"])+`}`)
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+own.String(), `{"parent_id":"`+parent.String()+`"}`, 200)
	trace := &listTrace{}
	one := listTracePool(t, w.pool.Config().ConnString(), trace, func(cfg *pgxpool.Config) { cfg.MaxConns = 1 })
	// A real ReadTx and real ports; only the forbidden write transaction
	// is replaced with the existing fail-fast boundary sentinel.
	route := newIssueRoute(t, one, versionReadOnly{})
	cursor, err := domain.EncodeListCursor(domain.ListCursor{Order: "sort_order", Key: []byte("9999999999"), ID: own})
	if err != nil {
		t.Fatal(err)
	}
	project := "/api/v0/projects/" + w.web.String() + "/issues"
	for _, tt := range []struct {
		name, path, query, sql string
		grouped, lanes, empty  bool
	}{
		{"flat", project, "", "FlatIssues", false, false, false},
		{"groups", project, "group_by=label_id", "GroupIssues", true, false, false},
		{"lanes", project, "group_by=label_id&sub_group_by=assignee_id", "SubgroupIssues", true, true, false},
		{"member", memberIssuesPath(w, "bob"), "relation=assigned", "FlatIssues", false, false, false},
		{"fallback", project, "label_id=" + w.bug.String() + "&order_by=sort_order&cursor=" + url.QueryEscape(cursor), "FlatIssues", false, false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), listRequestKey{}, tt.name)
			path := tt.path
			if tt.query != "" {
				path += "?" + tt.query
			}
			req, rec := route.sendContext(ctx, http.MethodGet, path, w.ids["walt"], "")
			w.contract.CheckResponse(t, req, rec.Result())
			if rec.Code != 200 {
				t.Fatal(rec.Code, rec.Body.String())
			}
			page := checkListEnvelope(t, rec.Body.String(), tt.grouped, tt.lanes)
			if page.Total != 1 {
				t.Fatal(page)
			}
			rows := page.Data
			if tt.grouped {
				if len(page.Groups) != 1 {
					t.Fatal(page)
				}
				rows = page.Groups[0].Data
				if tt.lanes {
					if len(page.Groups[0].Subgroups) != 1 {
						t.Fatal(page)
					}
					rows = page.Groups[0].Subgroups[0].Data
				}
			}
			if tt.empty {
				if len(rows) != 0 || page.NextCursor != nil {
					t.Fatal(page)
				}
			} else if len(rows) != 1 || rows[0].ID != own || rows[0].ParentID != nil || !slices.Equal(rows[0].LabelIDs, []uuid.UUID{w.bug}) || !slices.Equal(rows[0].AssigneeIDs, []uuid.UUID{w.ids["bob"]}) {
				t.Fatal("wrong row, hidden parent or arrays", page)
			}
			records := trace.take(tt.name)
			if len(records) < 8 || !strings.Contains(strings.ToUpper(records[0].SQL), "REPEATABLE READ READ ONLY") || strings.ToUpper(records[len(records)-1].SQL) != "COMMIT" {
				t.Fatal("missing independent RO/RR transaction", records)
			}
			reads := 0
			for _, r := range records {
				if r.PID != records[0].PID || r.Ended.IsZero() || r.Error != "" {
					t.Fatal("source/directory/SQL escaped connection", r)
				}
				if strings.Contains(strings.ToUpper(r.SQL), "FOR UPDATE") || strings.Contains(strings.ToUpper(r.SQL), "FOR SHARE") || strings.Contains(strings.ToUpper(r.SQL), "FOR NO KEY UPDATE") || strings.Contains(strings.ToUpper(r.SQL), "FOR KEY SHARE") {
					t.Fatal("read acquired row lock", r.SQL)
				}
				if listStatementName(r.SQL) == tt.sql {
					reads++
				}
			}
			wantReads := 1
			if tt.empty {
				wantReads = 2
			}
			if reads != wantReads {
				t.Fatalf("%s SQL reads=%d want %d", tt.name, reads, wantReads)
			}
		})
	}
}
