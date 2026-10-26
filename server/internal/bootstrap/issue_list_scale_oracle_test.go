package bootstrap

import (
	"fmt"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"time"
	"uuid"
)

type listScaleRow struct {
	ID, Parent, Creator            uuid.UUID
	Project, N, Seq, Kind, State   int
	Priority, Start, Target        string
	Labels, Assignees, Subscribers []uuid.UUID
	Created                        int64
	Sort                           int
}

const scaleEpochMicros int64 = 1767225600000000

func (f listScaleFixture) expectedRows() []listScaleRow {
	var out []listScaleRow
	alice, bob, walt := f.w.ids["alice"], f.w.ids["bob"], f.w.ids["walt"]
	for p, project := range f.projects {
		appendRow := func(n, seq, kind int) {
			r := listScaleRow{ID: scaleID(p, seq), Project: p, N: n, Seq: seq, Kind: kind, State: n / 5 % 5,
				Priority: []string{"urgent", "high", "medium", "low", "none"}[n%5], Creator: []uuid.UUID{alice, bob, walt, {}}[n%4], Created: scaleEpochMicros + int64(n/4), Sort: 1000 + n%8}
			if kind == 1 {
				r.State = 5
			}
			if kind >= 4 {
				r.Parent = scaleID(p, n+1)
			}
			if kind == 4 {
				r.Creator = walt
			}
			if kind == 5 {
				r.Creator = alice
			}
			if n%3 != 0 {
				r.Start = fmt.Sprintf("2026-01-%02d", 1+n%11)
			}
			if n%4 != 0 {
				r.Target = fmt.Sprintf("2026-01-%02d", 21+n%7)
			}
			r.Labels = [][]uuid.UUID{{}, {project.Labels[0]}, {project.Labels[1]}, {project.Labels[0], project.Labels[1]}}[n/25%4]
			r.Assignees = [][]uuid.UUID{{}, {alice}, {bob}, {alice, bob}}[n/100%4]
			r.Subscribers = [][]uuid.UUID{{}, {alice}, {bob}, {alice, bob}}[n/7%4]
			out = append(out, r)
		}
		for n := 0; n < project.Roots; n++ {
			appendRow(n, n+1, 0)
		}
		for block := 0; block < project.Roots/100; block++ {
			for kind := 1; kind <= 7; kind++ {
				appendRow(block*100, project.Roots+block*7+kind, kind)
			}
		}
	}
	return out
}
func (r listScaleRow) archived() bool { return r.Kind == 2 || r.Kind == 6 }
func (r listScaleRow) deleted() bool  { return r.Kind == 3 || r.Kind == 7 }
func (f listScaleFixture) expectedDistribution() map[string]map[string]int64 {
	out := map[string]map[string]int64{}
	for _, name := range []string{"rows", "degrees", "creators", "subscribers", "mentions", "relations", "scope"} {
		out[name] = map[string]int64{}
	}
	for i, p := range f.projects {
		role := 15
		if i == 2 {
			role = 5
		}
		out["scope"][fmt.Sprintf("%s/%t/%d/%t/false", p.ID, i == 4, role, i != 3)] = 1
	}
	for _, r := range f.rows {
		p := f.projects[r.Project].ID.String()
		state := []string{"backlog", "unstarted", "started", "completed", "cancelled", "triage"}[r.State]
		key := fmt.Sprintf("%s/%s/%s/%t/%t/%t/%t/%t", p, state, r.Priority, r.Parent != (uuid.UUID{}), r.archived(), r.deleted(), r.Start == "", r.Target == "")
		out["rows"][key]++
		out["degrees"][fmt.Sprintf("%s/%d/%d", p, len(r.Labels), len(r.Assignees))]++
		creator := "none"
		if r.Creator != (uuid.UUID{}) {
			creator = r.Creator.String()
		}
		out["creators"][p+"/"+creator]++
		for _, id := range r.Subscribers {
			out["subscribers"][p+"/"+id.String()]++
		}
		if r.N%13 == 0 {
			out["mentions"][p+"/"+f.w.ids["bob"].String()]++
		}
		out["relations"]["labels/false"] += int64(len(r.Labels))
		out["relations"]["assignees/false"] += int64(len(r.Assignees))
		if r.Kind == 0 && r.N%100 == 0 {
			out["relations"]["labels/true"]++
		}
	}
	return out
}

type listScaleScenario struct {
	Name    string
	Project int // -1 is the member endpoint, target bob, actor walt.
	Query   url.Values
	Accept  func(listScaleRow) bool
}

func (s listScaleScenario) order() string {
	if v := s.Query.Get("order_by"); v != "" {
		return v
	}
	return "-created_at"
}
func (f listScaleFixture) scenarios() []listScaleScenario {
	a, c := f.projects[0].Labels[0], f.projects[0].Labels[1]
	bob := f.w.ids["bob"]
	return []listScaleScenario{
		{Name: "default"}, {Name: "label_name", Query: url.Values{"order_by": {"label_name"}}},
		{Name: "sort_order", Query: url.Values{"order_by": {"sort_order"}}},
		{Name: "state_groups", Query: url.Values{"group_by": {"state_id"}}},
		{Name: "label_groups", Query: url.Values{"group_by": {"label_id"}}},
		{Name: "swimlanes", Query: url.Values{"group_by": {"label_id"}, "sub_group_by": {"assignee_id"}}},
		{Name: "intersection", Query: url.Values{"group_by": {"label_id"}, "group_key": {c.String()}, "label_id": {a.String()}}, Accept: func(r listScaleRow) bool { return slices.Contains(r.Labels, a) && slices.Contains(r.Labels, c) }},
		{Name: "none", Query: url.Values{"label_id": {a.String() + ",none"}}, Accept: func(r listScaleRow) bool { return len(r.Labels) == 0 || slices.Contains(r.Labels, a) }},
		{Name: "date_and", Query: url.Values{"target_date": {"2026-01-22..2026-01-26", "2026-01-24.."}}, Accept: func(r listScaleRow) bool { return r.Target >= "2026-01-24" && r.Target <= "2026-01-26" }},
		{Name: "member_created", Project: -1, Query: url.Values{"relation": {"created"}, "group_by": {"project_id"}}, Accept: func(r listScaleRow) bool { return r.Creator == bob }},
		{Name: "member_assigned", Project: -1, Query: url.Values{"relation": {"assigned"}}, Accept: func(r listScaleRow) bool { return slices.Contains(r.Assignees, bob) }},
		{Name: "member_subscribed", Project: -1, Query: url.Values{"relation": {"subscribed"}}, Accept: func(r listScaleRow) bool { return slices.Contains(r.Subscribers, bob) }},
		{Name: "own_only", Project: 2},
	}
}
func (f listScaleFixture) expected(s listScaleScenario) []listScaleRow {
	var rows []listScaleRow
	for _, r := range f.rows {
		if r.deleted() || r.archived() || r.State == 5 {
			continue
		}
		if s.Project >= 0 && r.Project != s.Project || s.Project < 0 && r.Project > 2 {
			continue
		}
		if r.Project == 2 && r.Creator != f.w.ids["walt"] {
			continue
		}
		if s.Accept != nil && !s.Accept(r) {
			continue
		}
		rows = append(rows, r)
	}
	order := s.order()
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		switch order {
		case "label_name":
			name := func(r listScaleRow) string {
				if len(r.Labels) == 0 {
					return "~"
				}
				if slices.Contains(r.Labels, f.projects[r.Project].Labels[0]) {
					return "alpha"
				}
				return "charlie"
			}
			if name(a) != name(b) {
				return name(a) < name(b)
			}
		case "sort_order":
			if a.Sort != b.Sort {
				return a.Sort < b.Sort
			}
		default:
			if a.Created != b.Created {
				return a.Created > b.Created
			}
		}
		if order == "-created_at" {
			return a.ID.Compare(b.ID) > 0
		}
		return a.ID.Compare(b.ID) < 0
	})
	return rows
}
func (f listScaleFixture) groupRows(rows []listScaleRow, field string) (keys []string, groups map[string][]listScaleRow) {
	groups = map[string][]listScaleRow{}
	for _, r := range rows {
		var values []string
		switch field {
		case "state_id":
			values = []string{f.projects[r.Project].States[r.State].String()}
		case "project_id":
			values = []string{f.projects[r.Project].ID.String()}
		case "label_id":
			for _, id := range r.Labels {
				values = append(values, id.String())
			}
		case "assignee_id":
			for _, id := range r.Assignees {
				values = append(values, id.String())
			}
		}
		if len(values) == 0 {
			values = []string{"none"}
		}
		for _, v := range values {
			groups[v] = append(groups[v], r)
		}
	}
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[j] == "none" {
			return keys[i] != "none"
		}
		if keys[i] == "none" {
			return false
		}
		return keys[i] < keys[j]
	})
	return keys, groups
}
func (r listScaleRow) cursorKey(order string) string {
	switch order {
	case "sort_order":
		return strconv.Itoa(r.Sort)
	case "label_name":
		if len(r.Labels) == 0 {
			return "null"
		}
		if r.N/25%4 == 2 {
			return `"charlie"`
		}
		return `"alpha"`
	default:
		return strconv.Quote(time.UnixMicro(r.Created).UTC().Format("2006-01-02T15:04:05.000000Z"))
	}
}
