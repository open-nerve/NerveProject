package bootstrap

import (
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

// planningNameOrder is how a list orders by the cycles' or the modules'
// names (M6 P3b spec 2.1): the order, planning's tables, and the statement
// that reads their names for it (R2).
type planningNameOrder struct {
	order, table, memberships, column, statement string
}

var planningNameOrders = []planningNameOrder{
	{"cycle_name", "cycles", "cycle_issues", "cycle_id", "CycleNames"},
	{"module_name", "modules", "module_issues", "module_id", "ModuleNames"},
}

// A list ordered by the cycles' or the modules' names reads their names in
// the read-only snapshot it reads its rows in (M6 P3b spec R2, A3). a is in
// Beta, b in Alpha, c in none. Held at the names' statement while another
// connection renames Alpha to Zulu and puts c in Aardvark, the list answers
// b first, of three, as before them, in each of its three forms; the next
// list answers c first. One transaction reads, the names after the other
// directories and before the rows, and locks nothing.
func TestListPlanningNamesSnapshot(t *testing.T) {
	for _, kind := range planningNameOrders {
		for _, shape := range []string{"FlatIssues", "GroupIssues", "SubgroupIssues"} {
			t.Run(kind.order+"/"+shape, func(t *testing.T) {
				trace := &attachmentSnapshotTrace{gate: newGate(), pauseSQL: kind.statement}
				w := attachmentSnapshotWorld(t, trace)
				a, b, c := w.newWorkItem(t), w.newWorkItem(t), w.newWorkItem(t)
				alpha, beta := w.seedPlanningTarget(t, kind.table, "Alpha"), w.seedPlanningTarget(t, kind.table, "Beta")
				w.seedMembership(t, kind.memberships, kind.column, a, beta, false)
				w.seedMembership(t, kind.memberships, kind.column, b, alpha, false)
				path := "/api/v0/projects/" + w.web.String() + "/issues?order_by=" + kind.order + "&limit=1" + planningShapeQuery(shape)
				req := newRequest(t, "GET", w.base+path, w.tokens["bob"], nil)
				req.Header.Set(httpserver.HeaderRequestID, "attachment-snapshot-reader")
				var once sync.Once
				resume := func() { once.Do(func() { close(trace.gate.open) }) }
				defer resume()
				trace.gate.deadline = time.Now().Add(10 * time.Second)
				answer := sendInBackground(req)
				select {
				case <-trace.gate.held:
				case got := <-answer:
					t.Fatalf("read missed barrier: %+v", got)
				case <-time.After(5 * time.Second):
					t.Fatal("read barrier deadline")
				}
				listSeedSQL(t, w, "UPDATE "+kind.table+" SET name = 'Zulu' WHERE id = $1", alpha)
				w.seedMembership(t, kind.memberships, kind.column, c, w.seedPlanningTarget(t, kind.table, "Aardvark"), false)
				resume()
				got := receiveWithin(t, answer, 5*time.Second, "name sort snapshot")
				if got.err != nil || got.res.StatusCode != 200 {
					t.Fatalf("reader %+v", got)
				}
				w.contract.CheckResponse(t, req, got.res)
				checkPlanningNamePage(t, planningShapeCell(t, string(got.body), shape), kind.order, b, `"alpha"`)
				checkPlanningNamePage(t, planningShapeCell(t, w.send(t, "bob", "GET", path, "", 200), shape), kind.order, c, `"aardvark"`)
				trace.mu.Lock()
				queries, traceErr, multiple := strings.Join(trace.queries, "\n"), trace.err, trace.multiple
				trace.mu.Unlock()
				begin := strings.Index(strings.ToLower(queries), "begin isolation level repeatable read read only")
				directories := strings.Index(queries, "-- name: PublicProfiles :many")
				names := strings.Index(queries, "-- name: "+kind.statement+" :many")
				rows := strings.Index(queries, "-- name: "+shape+" :many")
				if traceErr != nil || multiple || begin < 0 || directories < begin || names < directories || rows < names || strings.Contains(strings.ToLower(queries), "for update") {
					t.Fatalf("mixed transaction or order of reads: %v %s", traceErr, queries)
				}
			})
		}
	}
}

// Every order but the cycles' and the modules' names reads no planning
// directory, and each of those reads only its own (M6 P3b spec R2, A3): the
// statements of one list read, in each of the project's three forms and the
// member's list, as the database is sent them.
func TestListOtherOrdersReadNoPlanningNames(t *testing.T) {
	trace := &attachmentSnapshotTrace{gate: newGate(), pauseSQL: "NoSuchStatement"}
	w := attachmentSnapshotWorld(t, trace)
	w.newWorkItem(t)
	statements := func(t *testing.T, path string) string {
		t.Helper()
		trace.mu.Lock()
		trace.queries, trace.pid, trace.multiple = nil, 0, false
		trace.mu.Unlock()
		req := newRequest(t, "GET", w.base+path, w.tokens["bob"], nil)
		req.Header.Set(httpserver.HeaderRequestID, "attachment-snapshot-reader")
		got := receiveWithin(t, sendInBackground(req), 5*time.Second, path)
		if got.err != nil || got.res.StatusCode != 200 {
			t.Fatalf("%s: %+v", path, got)
		}
		trace.mu.Lock()
		defer trace.mu.Unlock()
		return strings.Join(trace.queries, "\n")
	}
	orders := strings.Fields("sort_order created_at -created_at updated_at -updated_at start_date -start_date target_date -target_date priority -priority " +
		"state_name -state_name label_name -label_name assignee_name -assignee_name sub_issues_count -sub_issues_count link_count -link_count " +
		"attachment_count -attachment_count cycle_name -cycle_name module_name -module_name")
	for _, order := range orders {
		for _, shape := range []string{"FlatIssues", "GroupIssues", "SubgroupIssues", "member"} {
			path, rows := "/api/v0/projects/"+w.web.String()+"/issues?order_by="+order+planningShapeQuery(shape), shape
			if shape == "member" {
				path, rows = memberIssuesPath(w, "bob")+"?relation=assigned&order_by="+order, "FlatIssues"
			}
			queries := statements(t, path)
			if !strings.Contains(queries, "-- name: "+rows+" :many") {
				t.Fatalf("%s: the probe saw no rows read: %s", path, queries)
			}
			for _, kind := range planningNameOrders {
				read := strings.Contains(queries, "-- name: "+kind.statement+" :many")
				if want := strings.TrimPrefix(order, "-") == kind.order; read != want {
					t.Errorf("%s: read %s = %v, want %v", path, kind.statement, read, want)
				}
			}
		}
	}
}

// planningShapeQuery is the query of a list in shape: grouped by Bug's label
// and in its lanes by assignee, where all the world's work items are.
func planningShapeQuery(shape string) string {
	switch shape {
	case "GroupIssues":
		return "&group_by=label_id"
	case "SubgroupIssues":
		return "&group_by=label_id&sub_group_by=assignee_id"
	}
	return ""
}

// planningShapeCell is the flat page, or the only group's, or its only cell's.
func planningShapeCell(t *testing.T, raw, shape string) issueListHTTP {
	t.Helper()
	p := checkListEnvelope(t, raw, shape != "FlatIssues", shape == "SubgroupIssues")
	if p.Total != 3 {
		t.Fatal("total", p)
	}
	if shape != "FlatIssues" {
		if len(p.Groups) != 1 {
			t.Fatal(p)
		}
		p = p.Groups[0]
	}
	if shape == "SubgroupIssues" {
		if len(p.Subgroups) != 1 {
			t.Fatal(p)
		}
		p = p.Subgroups[0]
	}
	return p
}

// checkPlanningNamePage checks that the page of one row of three is first's,
// continued after first's name key key.
func checkPlanningNamePage(t *testing.T, page issueListHTTP, order string, first uuid.UUID, key string) {
	t.Helper()
	if page.Total != 3 || len(page.Data) != 1 || page.Data[0].ID != first || page.NextCursor == nil {
		t.Fatalf("page %+v; want %s first of 3", page, first)
	}
	c, err := domain.DecodeListCursor(*page.NextCursor, domain.ListOrder(order))
	if err != nil || c.ID != first || string(c.Key) != key {
		t.Fatalf("cursor %+v, %v; want %s after %s", c, err, first, key)
	}
}
