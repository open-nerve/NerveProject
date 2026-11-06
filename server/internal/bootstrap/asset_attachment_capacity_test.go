package bootstrap

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Catches truncation at 1000, sorting after LIMIT, duplicate relation counts,
// unauthorized guest rows, per-cell recounting and unbounded SQL round trips.
// Fresh DB, no ANALYZE. This is cold planner statistics, not an OS-cache claim.
func TestAttachmentCapacity(t *testing.T) {
	for _, n := range []int{1000, 1001, 10000} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			f := newAttachmentCapacity(t)
			w := f.w
			f.seedProject(t, 0, w.web, n)
			ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
			for _, actor := range []string{"bob", "walt"} {
				w.makeMember(t, actor, ops, shared.RoleMember)
			}
			f.seedProject(t, 1, ops, 7)
			membership := queryIDs(t, w.pool, "SELECT id FROM project_members WHERE project_id=$1 AND member_id=$2", w.web, w.ids["walt"])[0]
			w.send(t, "alice", "PATCH", "/api/v0/project-members/"+membership.String(), `{"role":5}`, 200)
			path := "/api/v0/projects/" + w.web.String() + "/issues"
			for _, order := range []string{"attachment_count", "-attachment_count"} {
				want := f.ordered(f.projectRows[w.web], order)
				q := url.Values{"order_by": {order}, "limit": {"50"}}
				p := f.read(t, "bob", path, q, n, 1)
				if p.Total != int64(n+1) {
					t.Fatal("distinct total", p.Total)
				}
				checkAttachmentCountPage(t, p, want[:50], f.counts, order, true)
				q.Set("cursor", *p.NextCursor)
				p = f.read(t, "bob", path, q, n, 1)
				checkAttachmentCountPage(t, p, want[50:100], f.counts, order, true)
				q.Del("cursor")
				guest := f.ordered(f.ownRows[w.web], order)
				p = f.read(t, "walt", path, q, n, 1)
				if p.Total != int64(len(guest)) {
					t.Fatal("guest leaked total", p.Total, len(guest))
				}
				checkAttachmentCountPage(t, p, guest[:50], f.counts, order, true)
				q.Set("group_by", "assignee_id")
				p = f.read(t, "bob", path, q, n, 1)
				f.checkGroups(t, p, want, order, false)
				q.Set("sub_group_by", "label_id")
				p = f.read(t, "bob", path, q, n, 1)
				f.checkGroups(t, p, want, order, true)
				q.Set("group_key", p.Groups[0].Key)
				q.Set("sub_group_key", p.Groups[0].Subgroups[0].Key)
				q.Set("cursor", *p.Groups[0].Subgroups[0].NextCursor)
				p = f.read(t, "bob", path, q, n, 1)
				checkAttachmentCountPage(t, p, want[50:100], f.counts, order, true)
				q = url.Values{"order_by": {order}, "limit": {"50"}, "relation": {"assigned"}}
				p = f.read(t, "bob", memberIssuesPath(w, "bob"), q, n+7, 2)
				both := f.ordered(slices.Concat(f.projectRows[w.web], f.projectRows[ops]), order)
				if p.Total != int64(n+9) {
					t.Fatal("multi-project total", p.Total)
				}
				checkAttachmentCountPage(t, p, both[:50], f.counts, order, true)
			}
			// A normal order must query only the unique emitted IDs, even though
			// overlapping groups/cells repeat every item four times in JSON.
			p := f.read(t, "bob", path, url.Values{"order_by": {"-created_at"}, "limit": {"50"}, "group_by": {"assignee_id"}, "sub_group_by": {"label_id"}}, 0, 1)
			for _, g := range p.Groups {
				for _, cell := range g.Subgroups {
					for _, row := range cell.Data {
						if row.AttachmentCount != f.counts[row.ID] {
							t.Fatal("ordinary count", row)
						}
					}
				}
			}
			if n == 1001 {
				f.cancelRead(t, path, n)
			}
		})
	}
	t.Run("1001 projects", func(t *testing.T) {
		f := newAttachmentCapacity(t)
		f.seedManyProjects(t)
		var ids []uuid.UUID
		for id := range f.counts {
			ids = append(ids, id)
		}
		for _, order := range []string{"attachment_count", "-attachment_count"} {
			want := f.ordered(ids, order)
			q := url.Values{"order_by": {order}, "limit": {"50"}, "relation": {"assigned"}}
			p := f.read(t, "bob", memberIssuesPath(f.w, "bob"), q, 2002, 1001)
			if p.Total != 2002 {
				t.Fatal("1001 project total", p.Total)
			}
			checkAttachmentCountPage(t, p, want[:50], f.counts, order, true)
			q.Set("cursor", *p.NextCursor)
			p = f.read(t, "bob", memberIssuesPath(f.w, "bob"), q, 2002, 1001)
			checkAttachmentCountPage(t, p, want[50:100], f.counts, order, true)
		}
	})
}

func (f attachmentCapacityFixture) ordered(ids []uuid.UUID, order string) []uuid.UUID {
	out := slices.Clone(ids)
	slices.SortFunc(out, func(a, b uuid.UUID) int {
		cmp := a.Compare(b)
		if f.counts[a] < f.counts[b] {
			cmp = -1
		} else if f.counts[a] > f.counts[b] {
			cmp = 1
		}
		if order[0] == '-' {
			return -cmp
		}
		return cmp
	})
	return out
}

func (f attachmentCapacityFixture) checkGroups(t *testing.T, p issueListHTTP, want []uuid.UUID, order string, lanes bool) {
	t.Helper()
	if p.Total != int64(len(want)) || len(p.Groups) != 2 {
		t.Fatal("overlapping total", p.Total, len(p.Groups))
	}
	if lanes && len(p.SubgroupTotals) != 2 {
		t.Fatal("lane totals", p.SubgroupTotals)
	}
	for _, total := range p.SubgroupTotals {
		if total.Total != int64(len(want)) {
			t.Fatal("duplicate lane total", total)
		}
	}
	for _, g := range p.Groups {
		if g.Total != int64(len(want)) {
			t.Fatal("group total", g.Total)
		}
		cells := []issueListHTTP{g}
		if lanes {
			cells = g.Subgroups
			if len(cells) != 2 {
				t.Fatal("cells", len(cells))
			}
		}
		for _, cell := range cells {
			if cell.Total != int64(len(want)) {
				t.Fatal("cell total", cell.Total)
			}
			checkAttachmentCountPage(t, cell, want[:50], f.counts, order, true)
		}
	}
}

func (f attachmentCapacityFixture) cancelRead(t *testing.T, path string, n int) {
	t.Helper()
	w := f.w
	// The app pool also runs the start-time jobs that read file_assets. One
	// still running would queue on this table lock and stay acquired.
	waitForCompletedKinds(t, w.pool, map[string]int{"asset.cleanup": 1, "cleanup.purge_deleted": 1}, &lockedBuffer{})
	held, err := w.pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer requestRelease(t, held)
	capacityExec(t, held, "LOCK TABLE file_assets IN ACCESS EXCLUSIVE MODE")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	req := newRequest(t, "GET", w.base+path+"?order_by=attachment_count", w.tokens["bob"], nil).WithContext(ctx)
	probe := w.observe(req)
	response := sendInBackground(req)
	requestTableBlocked(t, w.pool, probe, held.Conn().PgConn().PID(), "file_assets")
	cancel()
	if got := receiveWithin(t, response, 5*time.Second, "cancel attachment count read"); got.err == nil {
		t.Fatal("cancelled request succeeded")
	}
	requestEventually(t, "cancel releases list connection", func(context.Context) bool { return w.requests.appPool.Stat().AcquiredConns() == 0 })
	requestRelease(t, held)
	p := f.read(t, "bob", path, url.Values{"order_by": {"attachment_count"}, "limit": {"50"}}, n, 1)
	checkAttachmentCountPage(t, p, f.ordered(f.projectRows[w.web], "attachment_count")[:50], f.counts, "attachment_count", true)
}
