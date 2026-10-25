package bootstrap

import (
	"maps"
	"net/http"
	"net/url"
	"slices"
	"testing"
	"uuid"
)

// Wrong creator/project/date columns, counting expanded rows, or replacing the
// original filter with a group selection must disagree with the literal table.
func TestIssueListAllGroupingFields(t *testing.T) {
	f := seedListGroups(t)
	project := "/api/v0/projects/" + f.w.web.String() + "/issues"
	member := memberIssuesPath(f.w, "bob")
	projectRaw := url.Values{"start_date": {"2030-01-01"}, "subscriber_id": {f.w.ids["bob"].String()}}
	memberRaw := url.Values{"start_date": {"2030-01-01"}, "relation": {"subscribed"}}
	projectIDs := slices.Clone(f.ids[:8])
	slices.Reverse(projectIDs) // Each literal timestamp increases with its seed index.
	memberIDs := slices.Clone(f.ids)
	slices.Reverse(memberIDs)
	fields := []string{"state_id", "state_group", "priority", "label_id", "assignee_id", "created_by_id", "target_date"}
	for _, field := range fields {
		t.Run("project/"+field, func(t *testing.T) {
			checkListGroups(t, f, project, projectRaw, field, "", projectIDs)
		})
	}
	for _, field := range []string{"state_group", "priority", "label_id", "target_date", "project_id"} {
		t.Run("member/"+field, func(t *testing.T) {
			checkListGroups(t, f, member, memberRaw, field, "", memberIDs)
		})
	}
	combinations := 0
	for _, group := range fields {
		for _, lane := range fields {
			if group == lane {
				continue
			}
			combinations++
			t.Run("cells/"+group+"/"+lane, func(t *testing.T) {
				checkListGroups(t, f, project, projectRaw, group, lane, projectIDs)
			})
		}
	}
	if combinations != 42 {
		t.Fatalf("executed %d legal ordered pairs", combinations)
	}
	t.Run("intersection", func(t *testing.T) {
		q := maps.Clone(projectRaw)
		q.Set("label_id", f.a.String())
		checkListGroups(t, f, project, q, "label_id", "assignee_id", []uuid.UUID{f.ids[7], f.ids[6], f.ids[1], f.ids[0]})
		q.Set("group_by", "label_id")
		q.Set("group_key", "none")
		checkListPages(t, f.w, "walt", project, q, nil, f.keys, f.rows)
	})
	t.Run("none-date-filter", func(t *testing.T) {
		q := maps.Clone(projectRaw)
		q.Set("target_date", "none")
		checkListGroups(t, f, project, q, "target_date", "created_by_id", []uuid.UUID{f.ids[7], f.ids[6], f.ids[3], f.ids[2]})
	})
	t.Run("missing-key", func(t *testing.T) {
		q := maps.Clone(projectRaw)
		q.Set("group_by", "created_by_id")
		q.Set("group_key", f.w.ids["olga"].String())
		checkListPages(t, f.w, "walt", project, q, nil, f.keys, f.rows)
	})
	t.Run("own-only", func(t *testing.T) {
		listSeedSQL(t, f.w, "UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2", f.w.web, f.w.ids["walt"])
		r := f.rows[f.ids[6]]
		r.children = 1
		f.rows[f.ids[6]] = r
		q := maps.Clone(projectRaw)
		checkListGroups(t, f, project, q, "label_id", "assignee_id", []uuid.UUID{f.ids[7], f.ids[6]})
		q.Set("order_by", "sort_order")
		q.Set("created_by_id", f.w.ids["walt"].String())
		p := checkListEnvelope(t, f.w.send(t, "walt", http.MethodGet, project+"?"+q.Encode(), "", 200), false, false)
		for _, row := range p.Data {
			if row.ID == f.ids[6] && row.ParentID != nil {
				t.Fatal("hidden parent leaked", row)
			}
		}
	})
}

// Intersect only explicit member sets, retaining the independent fixed order.
func listIntersection(order []uuid.UUID, sets ...[]uuid.UUID) []uuid.UUID {
	out := []uuid.UUID{}
	for _, id := range order {
		keep := true
		for _, set := range sets {
			keep = keep && slices.Contains(set, id)
		}
		if keep {
			out = append(out, id)
		}
	}
	return out
}

func checkListGroups(t *testing.T, f listGroupFixture, path string, raw url.Values, group, lane string, order []uuid.UUID) {
	t.Helper()
	raw = maps.Clone(raw)
	raw.Set("limit", "1")
	raw.Set("group_by", group)
	if lane != "" {
		raw.Set("sub_group_by", lane)
	}
	p := checkListEnvelope(t, f.w.send(t, "walt", http.MethodGet, path+"?"+raw.Encode(), "", 200), true, lane != "")
	if p.Total != int64(len(order)) {
		t.Fatalf("overall total %d want %d", p.Total, len(order))
	}
	groupKeys := []string{}
	for _, key := range f.groups[group].keys {
		if len(listIntersection(order, f.groups[group].members[key])) > 0 {
			groupKeys = append(groupKeys, key)
		}
	}
	if len(p.Groups) != len(groupKeys) {
		t.Fatalf("groups %+v want %v", p.Groups, groupKeys)
	}
	for n, key := range groupKeys {
		g := p.Groups[n]
		want := listIntersection(order, f.groups[group].members[key])
		if g.Key != key || g.Total != int64(len(want)) {
			t.Fatalf("group %+v want %s / %v", g, key, want)
		}
		selected := maps.Clone(raw)
		selected.Set("group_key", key)
		if lane == "" {
			checkFirstGroupRow(t, g, want, f)
			if g.NextCursor != nil {
				selected.Set("cursor", *g.NextCursor)
			}
			checkListPages(t, f.w, "walt", path, selected, want, f.keys, f.rows)
			continue
		}
		cell := 0
		for _, laneKey := range f.groups[lane].keys {
			members := listIntersection(want, f.groups[lane].members[laneKey])
			if len(members) == 0 {
				continue
			}
			if cell >= len(g.Subgroups) || g.Subgroups[cell].Key != laneKey {
				t.Fatalf("cell key %s: %+v", laneKey, g.Subgroups)
			}
			checkFirstGroupRow(t, g.Subgroups[cell], members, f)
			selected.Set("sub_group_key", laneKey)
			selected.Del("cursor")
			if g.Subgroups[cell].NextCursor != nil {
				selected.Set("cursor", *g.Subgroups[cell].NextCursor)
			}
			checkListPages(t, f.w, "walt", path, selected, members, f.keys, f.rows)
			cell++
		}
		if cell != len(g.Subgroups) {
			t.Fatalf("unexpected cells %+v", g.Subgroups)
		}
	}
	if lane != "" {
		n := 0
		for _, key := range f.groups[lane].keys {
			want := listIntersection(order, f.groups[lane].members[key])
			if len(want) == 0 {
				continue
			}
			if n >= len(p.SubgroupTotals) || p.SubgroupTotals[n].Key != key || p.SubgroupTotals[n].Total != int64(len(want)) {
				t.Fatalf("independent lane %s/%d: %+v", key, len(want), p.SubgroupTotals)
			}
			n++
		}
		if n != len(p.SubgroupTotals) {
			t.Fatal("extra lane totals", p.SubgroupTotals)
		}
	}
}

func checkFirstGroupRow(t *testing.T, g issueListHTTP, want []uuid.UUID, f listGroupFixture) {
	t.Helper()
	if g.Total != int64(len(want)) || len(g.Data) != 1 || g.Data[0].ID != want[0] || (g.NextCursor == nil) != (len(want) == 1) {
		t.Fatalf("initial per-group/cell limit: %+v want %v", g, want)
	}
	checkListPageRow(t, g, f.rows[want[0]])
}
