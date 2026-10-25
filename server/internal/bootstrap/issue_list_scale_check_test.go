package bootstrap

import (
	"encoding/json"
	"maps"
	"net/url"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

type listScaleContinuation struct {
	Query  url.Values
	Rows   []listScaleRow
	Cursor string
}

// Assert the captured response only after request timing and rows consumption end.
func (f listScaleFixture) checkPage(t *testing.T, s listScaleScenario, body string, want []listScaleRow) listScaleContinuation {
	t.Helper()
	grouped := s.Query.Get("group_by") != "" && s.Query.Get("group_key") == ""
	lanes := s.Query.Get("sub_group_by") != ""
	page := checkListEnvelope(t, body, grouped, lanes)
	if page.Total != int64(len(want)) {
		t.Fatalf("%s total %d want %d", s.Name, page.Total, len(want))
	}
	q := maps.Clone(s.Query)
	if q == nil {
		q = url.Values{}
	}
	if !grouped {
		f.checkRows(t, s, page, want, 0)
		return listScaleContinuation{q, want, cursorText(page)}
	}
	keys, groups := f.groupRows(want, s.Query.Get("group_by"))
	if len(page.Groups) != len(keys) {
		t.Fatal("group keys", page)
	}
	var next listScaleContinuation
	for i, key := range keys {
		g := page.Groups[i]
		rows := groups[key]
		if g.Key != key || g.Total != int64(len(rows)) {
			t.Fatal("group total", g, key, len(rows))
		}
		q.Set("group_key", key)
		if !lanes {
			f.checkRows(t, s, g, rows, 0)
			if i == 0 {
				next = listScaleContinuation{maps.Clone(q), rows, cursorText(g)}
			}
			continue
		}
		cells, cellRows := f.groupRows(rows, s.Query.Get("sub_group_by"))
		if len(g.Subgroups) != len(cells) {
			t.Fatal("cell keys", g)
		}
		for j, cell := range cells {
			sub := g.Subgroups[j]
			if sub.Key != cell || sub.Total != int64(len(cellRows[cell])) {
				t.Fatal("cell total", sub)
			}
			f.checkRows(t, s, sub, cellRows[cell], 0)
			if i == 0 && j == 0 {
				q.Set("sub_group_key", cell)
				next = listScaleContinuation{maps.Clone(q), cellRows[cell], cursorText(sub)}
			}
		}
	}
	if lanes {
		keys, rows := f.groupRows(want, s.Query.Get("sub_group_by"))
		if len(page.SubgroupTotals) != len(keys) {
			t.Fatal("lane keys", page)
		}
		for i, key := range keys {
			if page.SubgroupTotals[i].Key != key || page.SubgroupTotals[i].Total != int64(len(rows[key])) {
				t.Fatal("independent lane total", page)
			}
		}
	}
	return next
}
func cursorText(p issueListHTTP) string {
	if p.NextCursor == nil {
		return ""
	}
	return *p.NextCursor
}
func (f listScaleFixture) checkRows(t *testing.T, s listScaleScenario, p issueListHTTP, want []listScaleRow, offset int) {
	t.Helper()
	end := min(offset+50, len(want))
	if p.Total != int64(len(want)) || len(p.Data) != end-offset {
		t.Fatalf("page rows/total %d/%d want %d/%d", len(p.Data), p.Total, end-offset, len(want))
	}
	children := map[uuid.UUID]int{}
	creators := map[uuid.UUID]uuid.UUID{}
	for _, r := range f.rows {
		creators[r.ID] = r.Creator
		if !r.deleted() && !r.archived() && r.Parent != (uuid.UUID{}) && (r.Project != 2 || r.Creator == f.w.ids["walt"]) {
			children[r.Parent]++
		}
	}
	for i, got := range p.Data {
		r := want[offset+i]
		if got.ID != r.ID {
			t.Fatalf("row %d got %s want %s", offset+i, got.ID, r.ID)
		}
		one := p
		one.Data = p.Data[i : i+1]
		checkListPageRow(t, one, listRowWant{labels: r.Labels, assignees: r.Assignees, children: children[r.ID]})
		parent := r.Parent
		if r.Project == 2 && creators[parent] != f.w.ids["walt"] {
			parent = uuid.UUID{}
		}
		if parent == (uuid.UUID{}) {
			if got.ParentID != nil {
				t.Fatal("unexpected parent", got)
			}
		} else if got.ParentID == nil || *got.ParentID != parent {
			t.Fatal("missing parent", got)
		}
	}
	if end == len(want) {
		if p.NextCursor != nil {
			t.Fatal("terminal cursor", p)
		}
		return
	}
	r := want[end-1]
	decoded, err := domain.DecodeListCursor(cursorText(p), domain.ListOrder(s.order()))
	if err != nil || decoded.ID != r.ID || string(decoded.Key) != r.cursorKey(s.order()) {
		t.Fatalf("cursor %+v %v want %s/%s", decoded, err, r.ID, r.cursorKey(s.order()))
	}
}
func scaleCursor(t *testing.T, s listScaleScenario, r listScaleRow) string {
	t.Helper()
	v, err := domain.EncodeListCursor(domain.ListCursor{Order: domain.ListOrder(s.order()), ID: r.ID, Key: json.RawMessage(r.cursorKey(s.order()))})
	if err != nil {
		t.Fatal(err)
	}
	return v
}
