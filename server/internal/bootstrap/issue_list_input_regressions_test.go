package bootstrap

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"uuid"
)

// A canonical JSON name position containing NUL must be rejected before the
// real PostgreSQL query can turn the client's cursor into an internal error.
func TestIssueListNULCursor(t *testing.T) {
	w := newIssueWorld(t)
	w.newWorkItem(t)
	path := "/api/v0/projects/" + w.web.String() + "/issues"
	for _, order := range []string{"state_name", "-state_name", "label_name", "-label_name", "assignee_name", "-assignee_name"} {
		for _, shape := range []string{"flat", "group", "cell"} {
			t.Run(order+"/"+shape, func(t *testing.T) {
				payload := fmt.Sprintf(`{"v":1,"p":[%q,"\u0000","00000000-0000-0000-0000-000000000001"]}`, order)
				q := url.Values{"order_by": {order}, "cursor": {base64.RawURLEncoding.EncodeToString([]byte(payload))}}
				if shape != "flat" {
					q.Set("group_by", "label_id")
					q.Set("group_key", w.bug.String())
				}
				if shape == "cell" {
					q.Set("sub_group_by", "assignee_id")
					q.Set("sub_group_key", w.ids["bob"].String())
				}
				body := w.send(t, "alice", http.MethodGet, path+"?"+q.Encode(), "", 400)
				if got := oneError(t, []byte(body)); got != "cursor invalid_format" {
					t.Fatal(body)
				}
				if got := problemCode(t, []byte(body)); got != "bad_request" {
					t.Fatal(body)
				}
			})
		}
	}
}

// Literal fixture membership and timestamps drive first pages, real returned
// continuation cursors, and a terminal empty page whose total must survive.
func TestIssueListUUIDGroupKeySpellings(t *testing.T) {
	f := seedListGroups(t)
	project := "/api/v0/projects/" + f.w.web.String() + "/issues"
	cases := []struct {
		name, path string
		q          url.Values
		want       []uuid.UUID
	}{
		{"project", project, url.Values{"start_date": {"2030-01-01"}, "group_by": {"label_id"}, "group_key": {f.a.String()}}, []uuid.UUID{f.ids[7], f.ids[6], f.ids[1], f.ids[0]}},
		{"cell", project, url.Values{"start_date": {"2030-01-01"}, "group_by": {"state_id"}, "group_key": {f.w.todo.String()}, "sub_group_by": {"label_id"}, "sub_group_key": {f.a.String()}}, []uuid.UUID{f.ids[7], f.ids[6], f.ids[1], f.ids[0]}},
		{"member", memberIssuesPath(f.w, "bob"), url.Values{"start_date": {"2030-01-01"}, "relation": {"subscribed"}, "group_by": {"project_id"}, "group_key": {f.w.web.String()}}, []uuid.UUID{f.ids[7], f.ids[6], f.ids[5], f.ids[4], f.ids[3], f.ids[2], f.ids[1], f.ids[0]}},
	}
	for _, tc := range cases {
		for _, spelling := range []string{"canonical", "uppercase", "compact"} {
			t.Run(tc.name+"/"+spelling, func(t *testing.T) {
				q := url.Values{}
				for key, values := range tc.q {
					q[key] = append([]string(nil), values...)
				}
				for _, key := range []string{"group_key", "sub_group_key"} {
					if value := q.Get(key); value != "" {
						switch spelling {
						case "uppercase":
							q.Set(key, strings.ToUpper(value))
						case "compact":
							q.Set(key, strings.ReplaceAll(value, "-", ""))
						}
						if spelling != "canonical" && q.Get(key) == value {
							t.Fatal("fixture must exercise another spelling", value)
						}
					}
				}
				checkListPages(t, f.w, "walt", tc.path, q, tc.want, f.keys, f.rows)
			})
		}
	}
}
