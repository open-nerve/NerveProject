package bootstrap

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestListAttachmentCountVisibilityAndRelations(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	counts := map[uuid.UUID]int64{}
	create := func(actor string, project uuid.UUID, count int64, extra string) uuid.UUID {
		id := w.newWorkItemBy(t, actor, project, fmt.Sprintf(`{"name":"Count visibility","assignee_ids":[%q]%s}`, w.ids["bob"], extra))
		counts[id] = count
		for range count {
			uploadedAssetHTTP(t, w, actor, "ISSUE_ATTACHMENT", attachmentScope("issue", id), "file.txt", "text/plain", []byte("abc"))
		}
		return id
	}
	hidden := create("bob", w.web, 5, "")
	own := create("walt", w.web, 1, "")
	child := create("walt", w.web, 2, "")
	w.send(t, "alice", "PATCH", "/api/v0/issues/"+child.String(), fmt.Sprintf(`{"parent_id":%q}`, hidden), 200)
	archived := create("walt", w.web, 4, fmt.Sprintf(`,"state_id":%q`, w.done))
	w.send(t, "walt", "POST", "/api/v0/issues/"+archived.String()+"/archive", "", 200)
	for _, tc := range []struct {
		path string
		ids  []uuid.UUID
	}{
		{"/api/v0/projects/" + w.web.String() + "/issues?sub_issue=false", []uuid.UUID{own}},
		{"/api/v0/projects/" + w.web.String() + "/issues?sub_issue=true", []uuid.UUID{child, own}},
		{"/api/v0/projects/" + w.web.String() + "/issues?archived=true&sub_issue=true", []uuid.UUID{archived}},
		{memberIssuesPath(w, "walt") + "?relation=created&sub_issue=true", []uuid.UUID{child, own}},
		{memberIssuesPath(w, "bob") + "?relation=assigned&sub_issue=true", []uuid.UUID{child, own}},
	} {
		p := checkListEnvelope(t, w.send(t, "walt", "GET", tc.path+"&order_by=-attachment_count", "", 200), false, false)
		if p.Total != int64(len(tc.ids)) {
			t.Fatal("hidden/archived total", p)
		}
		checkAttachmentCountPage(t, p, tc.ids, counts, "-attachment_count", false)
		for _, row := range p.Data {
			if row.ID == child && row.ParentID != nil {
				t.Fatal("count directory exposed hidden parent", row)
			}
		}
	}
	// The member list merges visible projects before ranking. Both ID and
	// count order oppose project traversal, and relation remains independent.
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	w.makeMember(t, "bob", ops, shared.RoleMember)
	cross := create("bob", ops, 3, "")
	for _, relation := range []string{"created", "assigned"} {
		want := []uuid.UUID{hidden, cross}
		if relation == "assigned" {
			want = append(want, child, own)
		}
		q := url.Values{"relation": {relation}, "sub_issue": {"true"}, "order_by": {"-attachment_count"}, "limit": {"1"}}
		for n, id := range want {
			p := checkListEnvelope(t, w.send(t, "bob", "GET", memberIssuesPath(w, "bob")+"?"+q.Encode(), "", 200), false, false)
			if p.Total != int64(len(want)) {
				t.Fatal("cross project total", p)
			}
			checkAttachmentCountPage(t, p, []uuid.UUID{id}, counts, "-attachment_count", n+1 < len(want))
			if p.NextCursor != nil {
				q.Set("cursor", *p.NextCursor)
			}
		}
	}
}

func TestListAttachmentCountPreferences(t *testing.T) {
	w := newIssueWorld(t)
	path := "/api/v0/me/projects/" + w.web.String() + "/preferences"
	for _, order := range []string{"attachment_count", "-attachment_count"} {
		display := strings.Replace(defaultDisplay, "-created_at", order, 1)
		want := preferencesJSON("work_items", "[]", 65535, `{}`, display, defaultProperties)
		samePreferencesJSON(t, w.send(t, "bob", "PATCH", path, `{"display_filters":`+display+`}`, 200), want)
		samePreferencesJSON(t, w.send(t, "bob", "GET", path, "", 200), want)
	}
	w.send(t, "bob", "PATCH", path, `{"display_filters":`+strings.Replace(defaultDisplay, "-created_at", "future_attachment_count", 1)+`}`, 422)
}
