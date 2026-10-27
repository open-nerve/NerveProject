package bootstrap

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"uuid"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Missing base/SubIssues/search projections, counting tombstones, filtering by
// author or metadata, and losing explicit zero must each break a public answer.
func TestIssueLinkCountsAcrossRepresentations(t *testing.T) {
	w := newIssueWorld(t)
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	project := "/api/v0/projects/" + w.web.String() + "/issues"
	created := w.send(t, "walt", http.MethodPost, project, `{"name":"LinkCount A"}`, 201)
	var a struct {
		ID         uuid.UUID
		SequenceID int `json:"sequence_id"`
	}
	decodeAnswer(t, created, &a)
	checkIssueLinkCounts(t, created, map[uuid.UUID]int64{a.ID: 0})
	b := w.newWorkItemBy(t, "walt", w.web, `{"name":"LinkCount B"}`)
	c := w.newWorkItemBy(t, "walt", w.web, `{"name":"LinkCount C"}`)
	live := createLink(t, w, "bob", a.ID, "http://169.254.169.254/count-a")
	pending := createLink(t, w, "alice", a.ID, "http://169.254.169.254/count-b")
	deleted := createLink(t, w, "bob", a.ID, "http://169.254.169.254/count-deleted")
	createLink(t, w, "bob", c, "http://169.254.169.254/count-c")
	w.send(t, "alice", http.MethodDelete, linkPath(deleted.ID), "", 204)
	waitUniqueLinkCrawl(t, w, live)
	waitUniqueLinkCrawl(t, w, pending)
	listSeedSQL(t, w, `UPDATE issue_links SET metadata=CASE WHEN id=$1 THEN '{"error":"blocked"}'::jsonb ELSE '{}'::jsonb END WHERE id IN ($1,$2)`, live.ID, pending.ID)
	path := "/api/v0/issues/" + a.ID.String()
	want := map[uuid.UUID]int64{a.ID: 2, b: 0, c: 1}
	detail := func(method, suffix, body string) {
		t.Helper()
		checkIssueLinkCounts(t, w.send(t, "walt", method, path+suffix, body, 200), map[uuid.UUID]int64{a.ID: want[a.ID]})
	}
	detail(http.MethodGet, "", "")
	checkIssueLinkCounts(t, w.send(t, "walt", http.MethodGet, "/api/v0/workspaces/acme/issues/WEB-"+strconv.Itoa(a.SequenceID), "", 200), map[uuid.UUID]int64{a.ID: 2})
	detail(http.MethodPatch, "", `{}`)
	detail(http.MethodPatch, "", `{"name":"LinkCount renamed"}`)
	detail(http.MethodPatch, "", fmt.Sprintf(`{"state_id":%q}`, w.done))
	for _, suffix := range []string{"/archive", "/archive", "/unarchive", "/unarchive"} {
		detail(http.MethodPost, suffix, "")
	}
	// GetIssues is used by relation writes and nested relation reads.
	related := w.send(t, "alice", http.MethodPost, relationsOf(b), relating("relates_to", a.ID, c), 201)
	checkIssueLinkCounts(t, related, map[uuid.UUID]int64{a.ID: 2, c: 1})
	checkIssueLinkCounts(t, w.send(t, "walt", http.MethodGet, relationsOf(b), "", 200), map[uuid.UUID]int64{a.ID: 2, c: 1})
	// Parent writes read back via GetIssues; the later GET has its own SubIssues SQL.
	checkIssueLinkCounts(t, w.send(t, "alice", http.MethodPost, subIssues(b), subIssueIDs(a.ID, c), 200), map[uuid.UUID]int64{a.ID: 2, c: 1})
	checkIssueLinkCounts(t, w.send(t, "walt", http.MethodGet, subIssues(b), "", 200), map[uuid.UUID]int64{a.ID: 2, c: 1})
	// OwnOnly constrains issues, never other people's live links on an admitted issue.
	listSeedSQL(t, w, "UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2", w.web, w.ids["walt"])
	checkLists := func() {
		t.Helper()
		for _, query := range []string{"?sub_issue=true", "?sub_issue=true&group_by=state_group", "?sub_issue=true&group_by=state_group&sub_group_by=priority"} {
			checkIssueLinkCounts(t, w.send(t, "walt", http.MethodGet, project+query, "", 200), want)
		}
		checkIssueLinkCounts(t, w.send(t, "walt", http.MethodGet, memberIssuesPath(w, "walt")+"?relation=created&sub_issue=true", "", 200), want)
		for _, q := range []string{"", "?q=LinkCount"} {
			checkIssueLinkCounts(t, w.send(t, "walt", http.MethodGet, searchPath+q, "", 200), want)
		}
	}
	checkLists()
	// Title, URL and the exact worker metadata port cannot change the live count.
	w.send(t, "alice", http.MethodPatch, linkPath(live.ID), `{"title":"User title"}`, 200)
	detail(http.MethodGet, "", "")
	w.send(t, "alice", http.MethodPatch, linkPath(live.ID), `{"url":"http://169.254.169.254/count-replaced"}`, 200)
	waitUniqueLinkCrawl(t, w, answeredLink{ID: live.ID, URL: "http://169.254.169.254/count-replaced"})
	detail(http.MethodGet, "", "")
	title := "Worker title"
	changed, err := postgresadapter.New(w.pool).SetLinkMetadata(pgtest.Soon(t), live.ID, "http://169.254.169.254/count-replaced", domain.LinkMetadata{"title": &title})
	if err != nil || !changed {
		t.Fatal("worker metadata update", changed, err)
	}
	detail(http.MethodGet, "", "")
	w.send(t, "alice", http.MethodDelete, linkPath(live.ID), "", 204)
	want[a.ID] = 1
	detail(http.MethodGet, "", "")
	checkIssueLinkCounts(t, w.send(t, "walt", http.MethodGet, subIssues(b), "", 200), map[uuid.UUID]int64{a.ID: 1, c: 1})
	checkIssueLinkCounts(t, w.send(t, "walt", http.MethodGet, relationsOf(b), "", 200), map[uuid.UUID]int64{a.ID: 1, c: 1})
	checkIssueLinkCounts(t, w.send(t, "walt", http.MethodGet, "/api/v0/workspaces/acme/issues/WEB-"+strconv.Itoa(a.SequenceID), "", 200), map[uuid.UUID]int64{a.ID: 1})
	checkLists()
}

// Traverse actual envelopes so the same assertion checks detail, list, group,
// subgroup, and relation-nested Issue values. Compare integers without float64.
func checkIssueLinkCounts(t *testing.T, answer string, want map[uuid.UUID]int64) {
	t.Helper()
	var body any
	decoder := json.NewDecoder(strings.NewReader(answer))
	decoder.UseNumber()
	if err := decoder.Decode(&body); err != nil {
		t.Fatal(err)
	}
	seen := map[uuid.UUID]bool{}
	var visit func(any)
	visit = func(value any) {
		switch v := value.(type) {
		case []any:
			for _, child := range v {
				visit(child)
			}
		case map[string]any:
			if _, isIssue := v["sequence_id"]; isIssue {
				id, err := uuid.Parse(fmt.Sprint(v["id"]))
				if err != nil {
					t.Fatal(err)
				}
				expected, ok := want[id]
				if !ok {
					t.Fatalf("unexpected issue %s in %s", id, answer)
				}
				number, ok := v["link_count"].(json.Number)
				if !ok {
					t.Fatalf("issue %s lacks required integer link_count: %s", id, answer)
				}
				count, err := number.Int64()
				if err != nil || count != expected {
					t.Fatalf("issue %s link_count=%s want %d: %v", id, number, expected, err)
				}
				seen[id] = true
			}
			for _, child := range v {
				visit(child)
			}
		}
	}
	visit(body)
	if len(seen) != len(want) {
		t.Fatalf("issues %v want %v in %s", seen, want, answer)
	}
}

// Changing a valid count/UUID only changes a starting position. Scope,
// OwnOnly, filters, member relation and selected group/cell still apply.
func TestIssueLinkCountCursorAuthorization(t *testing.T) {
	w := newIssueWorld(t)
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	visible := w.newWorkItemBy(t, "walt", w.web, `{"name":"Visible"}`)
	hidden := w.newWorkItem(t)
	filtered := w.newWorkItemBy(t, "walt", w.web, `{"name":"Filtered","priority":"high"}`)
	grouped := w.newWorkItemBy(t, "walt", w.web, fmt.Sprintf(`{"name":"Other group","state_id":%q}`, w.done))
	unrel := w.newWorkItemBy(t, "walt", w.web, `{"name":"No subscription"}`)
	for _, id := range []uuid.UUID{visible, filtered, grouped} {
		listSeedSQL(t, w, `INSERT INTO issue_subscribers(id,workspace_id,project_id,issue_id,subscriber_id) VALUES($1,$2,$3,$4,$5)`, uuid.NewV7(), w.acme, w.web, id, w.ids["bob"])
	}
	outside := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Private", "PRIVATE")
	private := w.newWorkItemIn(t, outside, `{"name":"Private"}`)
	for _, id := range []uuid.UUID{visible, hidden, filtered, private, grouped, unrel} {
		createLink(t, w, "alice", id, "http://169.254.169.254/cursor-"+id.String())
	}
	listSeedSQL(t, w, "UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2", w.web, w.ids["walt"])
	project := "/api/v0/projects/" + w.web.String() + "/issues"
	for _, order := range []domain.ListOrder{"link_count", "-link_count"} {
		for _, shape := range []string{"flat", "group", "cell", "member"} {
			t.Run(string(order)+"/"+shape, func(t *testing.T) {
				path := project
				q := url.Values{"order_by": {string(order)}, "priority": {"none"}, "sub_issue": {"true"}}
				expected := map[uuid.UUID]int64{visible: 1, unrel: 1}
				if shape == "flat" || shape == "member" {
					q.Set("state_group", "backlog")
				}
				if shape == "member" {
					expected = map[uuid.UUID]int64{visible: 1}
					path = memberIssuesPath(w, "bob")
					q.Set("relation", "subscribed")
					q.Set("project_id", w.web.String())
				}
				if shape == "group" || shape == "cell" {
					q.Set("group_by", "state_group")
					q.Set("group_key", "backlog")
				}
				if shape == "cell" {
					q.Del("priority")
					q.Set("sub_group_by", "priority")
					q.Set("sub_group_key", "none")
				}
				for _, key := range []string{"-1", "null", "1.5", "1.0", `"1"`, "9223372036854775808"} {
					raw := fmt.Sprintf(`{"v":1,"p":[%q,%s,%q]}`, order, key, hidden.String())
					q.Set("cursor", base64.RawURLEncoding.EncodeToString([]byte(raw)))
					answer := w.send(t, "walt", http.MethodGet, path+"?"+q.Encode(), "", 400)
					if oneError(t, []byte(answer)) != "cursor invalid_format" {
						t.Fatal(answer)
					}
				}
				key := "0"
				if order[0] == '-' {
					key = "9223372036854775807"
				}
				cursor, err := domain.EncodeListCursor(domain.ListCursor{Order: order, Key: json.RawMessage(key), ID: hidden})
				if err != nil {
					t.Fatal(err)
				}
				q.Set("cursor", cursor)
				checkIssueLinkCounts(t, w.send(t, "walt", http.MethodGet, path+"?"+q.Encode(), "", 200), expected)
				key = "9223372036854775807"
				if order[0] == '-' {
					key = "0"
				}
				cursor, err = domain.EncodeListCursor(domain.ListCursor{Order: order, Key: json.RawMessage(key), ID: private})
				if err != nil {
					t.Fatal(err)
				}
				q.Set("cursor", cursor)
				page := checkListEnvelope(t, w.send(t, "walt", http.MethodGet, path+"?"+q.Encode(), "", 200), false, false)
				if page.Total != int64(len(expected)) || len(page.Data) != 0 || page.NextCursor != nil {
					t.Fatal("forged cursor changed scope", page)
				}
			})
		}
	}
}
