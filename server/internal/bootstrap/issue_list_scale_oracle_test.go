package bootstrap

import (
	"fmt"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
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
		first, second := project.People[0], project.People[1]
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
			r.Assignees = [][]uuid.UUID{{}, {first}, {second}, {first, second}}[n/100%4]
			r.Subscribers = [][]uuid.UUID{{}, {first}, {second}, {first, second}}[n/7%4]
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
	for _, p := range f.projects {
		out["scope"][fmt.Sprintf("%s/%t/%d/%t/false", p.ID, p.Archived, p.Role, !p.Left)] = 1
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

// listScaleRoute is the endpoint a scenario reads, as walt, and so the
// scope its expected rows come from (inScope).
type listScaleRoute int

const (
	// scaleProjectList is the project list of the scenario's Project.
	scaleProjectList listScaleRoute = iota
	// scaleMemberList is bob's member list in acme, of the projects walt sees.
	scaleMemberList
	// scaleWorkspaceList is acme's workspace list (M7 P3a), of the projects
	// walt sees, like the member list.
	scaleWorkspaceList
)

type listScaleScenario struct {
	Name    string
	Route   listScaleRoute
	Project int // the project a project scenario reads
	// Tail is the rows of a last page: the request's cursor is the row
	// before them. 0 is the first page.
	Tail int
	// Pair is the project scenario a workspace scenario is judged against:
	// its list-SQL median at the same total, in the same run (M7 P3a spec 2.4).
	Pair   string
	Query  url.Values
	Accept func(listScaleRow) bool
}

func (s listScaleScenario) order() string {
	if v := s.Query.Get("order_by"); v != "" {
		return v
	}
	return "-created_at"
}

// scenarios are the scope's, or the sub-seed's when f is spread. The scope's
// are m4Scenarios and three more of Web's project list. The workspace list's
// five (M7 P3a spec 2.4) are in each, paired with Web's project list at the
// same total; assigned and subscribed are by bob's id in the scope, where
// walt is neither (me is the caller's id once parsed: the same SQL), and by
// me in spread, where walt is both in bob's place, as the default views ask.
func (f listScaleFixture) scenarios() []listScaleScenario {
	assigned := func(id uuid.UUID) func(listScaleRow) bool {
		return func(r listScaleRow) bool { return slices.Contains(r.Assignees, id) }
	}
	subscribed := func(id uuid.UUID) func(listScaleRow) bool {
		return func(r listScaleRow) bool { return slices.Contains(r.Subscribers, id) }
	}
	if f.spread {
		walt := f.w.ids["walt"]
		return []listScaleScenario{
			{Name: "spread_default", Route: scaleWorkspaceList, Pair: "default"},
			{Name: "spread_assigned", Route: scaleWorkspaceList, Pair: "assigned", Query: url.Values{"assignee_id": {"me"}}, Accept: assigned(walt)},
			{Name: "spread_subscribed", Route: scaleWorkspaceList, Pair: "subscribed", Query: url.Values{"subscriber_id": {"me"}}, Accept: subscribed(walt)},
			{Name: "spread_label_name", Route: scaleWorkspaceList, Pair: "label_name", Query: url.Values{"order_by": {"label_name"}}},
			{Name: "spread_last_page", Route: scaleWorkspaceList, Pair: "last_page", Tail: 25},
		}
	}
	bob := f.w.ids["bob"]
	return append(f.m4Scenarios(), []listScaleScenario{
		{Name: "assigned", Query: url.Values{"assignee_id": {bob.String()}}, Accept: assigned(bob)},
		{Name: "subscribed", Query: url.Values{"subscriber_id": {bob.String()}}, Accept: subscribed(bob)},
		{Name: "last_page", Tail: 25},
		{Name: "workspace_default", Route: scaleWorkspaceList, Pair: "default"},
		{Name: "workspace_assigned", Route: scaleWorkspaceList, Pair: "assigned", Query: url.Values{"assignee_id": {bob.String()}}, Accept: assigned(bob)},
		{Name: "workspace_subscribed", Route: scaleWorkspaceList, Pair: "subscribed", Query: url.Values{"subscriber_id": {bob.String()}}, Accept: subscribed(bob)},
		{Name: "workspace_label_name", Route: scaleWorkspaceList, Pair: "label_name", Query: url.Values{"order_by": {"label_name"}}},
		{Name: "workspace_last_page", Route: scaleWorkspaceList, Pair: "last_page", Tail: 25},
	}...)
}

// m4Scenarios are the scope's 13 scenarios of the project and member lists
// M4 P5 measured, the only ones the crowded database runs (listScaleCrowded).
func (f listScaleFixture) m4Scenarios() []listScaleScenario {
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
		{Name: "member_created", Route: scaleMemberList, Query: url.Values{"relation": {"created"}, "group_by": {"project_id"}}, Accept: func(r listScaleRow) bool { return r.Creator == bob }},
		{Name: "member_assigned", Route: scaleMemberList, Query: url.Values{"relation": {"assigned"}}, Accept: func(r listScaleRow) bool { return slices.Contains(r.Assignees, bob) }},
		{Name: "member_subscribed", Route: scaleMemberList, Query: url.Values{"relation": {"subscribed"}}, Accept: func(r listScaleRow) bool { return slices.Contains(r.Subscribers, bob) }},
		{Name: "own_only", Project: 2},
	}
}

// path is the endpoint s reads.
func (f listScaleFixture) path(s listScaleScenario) string {
	switch s.Route {
	case scaleMemberList:
		return memberIssuesPath(f.w, "bob")
	case scaleWorkspaceList:
		return "/api/v0/workspaces/acme/issues"
	}
	return "/api/v0/projects/" + f.projects[s.Project].ID.String() + "/issues"
}

// inScope is whether r is of a project s reads: a project scenario's
// Project; for the member and the workspace list, a project walt sees, one
// he has not left that is not archived.
func (f listScaleFixture) inScope(s listScaleScenario, r listScaleRow) bool {
	if s.Route == scaleProjectList {
		return r.Project == s.Project
	}
	p := f.projects[r.Project]
	return !p.Left && !p.Archived
}

// ownOnly is whether walt sees only his own work items of the project: he
// is its guest.
func (p listScaleProject) ownOnly() bool { return p.Role == shared.RoleGuest }
func (f listScaleFixture) expected(s listScaleScenario) []listScaleRow {
	var rows []listScaleRow
	for _, r := range f.rows {
		if r.deleted() || r.archived() || r.State == 5 {
			continue
		}
		if !f.inScope(s, r) {
			continue
		}
		if f.projects[r.Project].ownOnly() && r.Creator != f.w.ids["walt"] {
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
