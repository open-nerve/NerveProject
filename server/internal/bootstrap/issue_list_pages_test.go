package bootstrap

import (
	"encoding/json"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

type listRowWant struct {
	labels, assignees []uuid.UUID
	children          int
}

func listSeedSQL(t *testing.T, w issueWorld, sql string, args ...any) {
	t.Helper()
	if _, err := w.pool.Exec(pgtest.Soon(t), sql, args...); err != nil {
		t.Fatal(err)
	}
}

// Fixed seed values supply both the public terminal cursor and independent row
// expectations. Returned IDs are never sorted before comparing the page order.
func checkListPages(t *testing.T, w issueWorld, actor, path string, raw url.Values, want []uuid.UUID, keys map[uuid.UUID]json.RawMessage, rows map[uuid.UUID]listRowWant) {
	t.Helper()
	raw = maps.Clone(raw)
	raw.Set("limit", "1")
	order := raw.Get("order_by")
	if order == "" {
		order = "-created_at"
	}
	start := 0
	if cursor := raw.Get("cursor"); cursor != "" {
		// The first row was checked in the nested response; consume that
		// response's actual cursor instead of fetching a new selected first page.
		c, err := domain.DecodeListCursor(cursor, domain.ListOrder(order))
		if len(want) == 0 || err != nil || c.ID != want[0] || string(c.Key) != string(keys[want[0]]) {
			t.Fatalf("initial group/cell cursor %+v / %v", c, err)
		}
		start = 1
	}
	for n := start; n < len(want); n++ {
		id := want[n]
		p := checkListEnvelope(t, w.send(t, actor, http.MethodGet, path+"?"+raw.Encode(), "", 200), false, false)
		if p.Total != int64(len(want)) || len(p.Data) != 1 || p.Data[0].ID != id {
			t.Fatalf("page %d: %+v; want ID %s / total %d", n, p, id, len(want))
		}
		checkListPageRow(t, p, rows[id])
		if n == len(want)-1 {
			if p.NextCursor != nil {
				t.Fatalf("terminal cursor: %s", *p.NextCursor)
			}
		} else {
			if p.NextCursor == nil {
				t.Fatal("lost lookahead/continuation")
			}
			c, err := domain.DecodeListCursor(*p.NextCursor, domain.ListOrder(order))
			if err != nil || c.ID != id || string(c.Key) != string(keys[id]) {
				t.Fatalf("cursor %+v / %v; want %s %s", c, err, id, keys[id])
			}
			raw.Set("cursor", *p.NextCursor)
		}
	}
	if len(want) > 0 {
		last := want[len(want)-1]
		cursor, err := domain.EncodeListCursor(domain.ListCursor{Order: domain.ListOrder(order), ID: last, Key: keys[last]})
		if err != nil {
			t.Fatal(err)
		}
		raw.Set("cursor", cursor)
	}
	p := checkListEnvelope(t, w.send(t, actor, http.MethodGet, path+"?"+raw.Encode(), "", 200), false, false)
	if p.Total != int64(len(want)) || len(p.Data) != 0 || p.NextCursor != nil {
		t.Fatalf("past end (original filter/selection must survive): %+v want total %d", p, len(want))
	}
}

func checkListPageRow(t *testing.T, p issueListHTTP, want listRowWant) {
	t.Helper()
	got := p.Data[0]
	labels, assignees := slices.Clone(want.labels), slices.Clone(want.assignees)
	slices.SortFunc(labels, uuid.UUID.Compare)
	slices.SortFunc(assignees, uuid.UUID.Compare)
	if !slices.Equal(got.LabelIDs, labels) || !slices.Equal(got.AssigneeIDs, assignees) || got.SubIssuesCount != want.children {
		t.Fatalf("complete arrays/count: %+v want %+v", got, want)
	}
}
