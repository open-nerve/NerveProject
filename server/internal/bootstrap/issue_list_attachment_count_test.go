package bootstrap

import (
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// The last created item is a zero, and the equal-count pair is reversed by
// descending UUID. Sorting only an already selected page cannot pass.
func TestListAttachmentCount(t *testing.T) {
	w := newIssueWorld(t)
	counts := map[uuid.UUID]int64{}
	var ids []uuid.UUID
	for n, count := range []int64{0, 2, 1, 2, 3, 0} {
		id := w.newWorkItemBy(t, "bob", w.web, fmt.Sprintf(`{"name":"Attachment order %d","assignee_ids":[%q,%q],"label_ids":[%q]}`, n, w.ids["alice"], w.ids["bob"], w.bug))
		ids = append(ids, id)
		counts[id] = count
		for range count {
			uploadedAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", attachmentScope("issue", id), "file.txt", "text/plain", []byte("abc"))
		}
	}
	for _, path := range []string{"/api/v0/projects/" + w.web.String() + "/issues", memberIssuesPath(w, "bob")} {
		member := path == memberIssuesPath(w, "bob")
		for _, shape := range []int{0, 1, 2} {
			if member && shape == 2 {
				continue
			} // Member lists have no subgroups.
			for _, order := range []string{"attachment_count", "-attachment_count"} {
				t.Run(path+fmt.Sprint(shape)+order, func(t *testing.T) {
					want := slices.Clone(ids)
					slices.SortFunc(want, func(a, b uuid.UUID) int {
						comparison := 0
						if counts[a] < counts[b] {
							comparison = -1
						} else if counts[a] > counts[b] {
							comparison = 1
						} else {
							comparison = a.Compare(b)
						}
						if order[0] == '-' {
							return -comparison
						}
						return comparison
					})
					q := url.Values{"order_by": {order}, "limit": {"2"}}
					if path == memberIssuesPath(w, "bob") {
						q.Set("relation", "assigned")
					}
					if shape > 0 {
						q.Set("group_by", "assignee_id")
						if member {
							q.Set("group_by", "label_id")
						}
					}
					if shape > 1 {
						q.Set("sub_group_by", "label_id")
					}
					for start := 0; start < len(want); start += 2 {
						p := checkListEnvelope(t, draftRequest(t, w, "bob", "GET", path+"?"+q.Encode(), "", 200), shape > 0 && start == 0, shape > 1 && start == 0)
						if p.Total != 6 {
							t.Fatal("distinct total", p.Total)
						}
						cell := p
						if shape > 0 && start == 0 {
							groups := 2
							if member {
								groups = 1
							}
							if len(p.Groups) != groups {
								t.Fatal("overlapping groups", p)
							}
							for _, g := range p.Groups {
								if g.Total != 6 {
									t.Fatal("group total", g)
								}
								part := g
								if shape > 1 {
									if len(g.Subgroups) != 1 || len(p.SubgroupTotals) != 1 || p.SubgroupTotals[0].Total != 6 {
										t.Fatal("distinct lane total", p)
									}
									part = g.Subgroups[0]
								}
								checkAttachmentCountPage(t, part, want[:2], counts, order, true)
							}
							cell = p.Groups[0]
							q.Set("group_key", cell.Key)
							if shape > 1 {
								cell = cell.Subgroups[0]
								q.Set("sub_group_key", cell.Key)
							}
						}
						checkAttachmentCountPage(t, cell, want[start:start+2], counts, order, start+2 < len(want))
						if cell.NextCursor != nil {
							q.Set("cursor", *cell.NextCursor)
						}
					}
					last := want[len(want)-1]
					cursor, err := domain.EncodeListCursor(domain.ListCursor{Order: domain.ListOrder(order), Key: json.RawMessage(fmt.Sprint(counts[last])), ID: last})
					if err != nil {
						t.Fatal(err)
					}
					q.Set("cursor", cursor)
					p := checkListEnvelope(t, w.send(t, "bob", "GET", path+"?"+q.Encode(), "", 200), false, false)
					if p.Total != 6 || len(p.Data) != 0 || p.NextCursor != nil {
						t.Fatal("empty continuation lost total", p)
					}
				})
			}
		}
	}
}

func checkAttachmentCountPage(t *testing.T, page issueListHTTP, want []uuid.UUID, counts map[uuid.UUID]int64, order string, more bool) {
	t.Helper()
	if len(page.Data) != len(want) || (page.NextCursor != nil) != more {
		t.Fatalf("page/lookahead %+v want %v more=%v", page, want, more)
	}
	for n, id := range want {
		if page.Data[n].ID != id || page.Data[n].AttachmentCount != counts[id] {
			t.Fatalf("count order row %+v want %s/%d", page.Data[n], id, counts[id])
		}
	}
	if page.NextCursor != nil {
		c, err := domain.DecodeListCursor(*page.NextCursor, domain.ListOrder(order))
		last := want[len(want)-1]
		if err != nil || c.ID != last || string(c.Key) != fmt.Sprint(counts[last]) {
			t.Fatalf("cursor not last emitted count %+v %v", c, err)
		}
	}
}
