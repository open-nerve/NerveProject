package postgresadapter_test

import (
	"context"
	"math"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

func TestIssueGroups(t *testing.T) {
	s, pool := newStore(t)
	a, b := newAccount(t, pool, "group-a"), newAccount(t, pool, "group-b")
	w := newWorkspace(t, pool, "groups")
	p := newProject(t, pool, w, "GROUP")
	l, c := newLabel(t, pool, p, "A"), newLabel(t, pool, p, "C")
	ids := make([]uuid.UUID, 6)
	for n := range ids {
		ids[n] = uuid.NewV7()
		newIssue(t, pool, p, ids[n], n+1, a)
		exec(t, pool, "UPDATE issues SET sort_order=$2 WHERE id=$1", ids[n], n)
	}
	for n := 0; n < 4; n++ {
		exec(t, pool, "INSERT INTO issue_labels(id,workspace_id,project_id,issue_id,label_id) VALUES($1,$2,$3,$4,$5)", uuid.NewV7(), w, p.id, ids[n], c)
		exec(t, pool, "INSERT INTO issue_assignees(id,workspace_id,project_id,issue_id,assignee_id) VALUES($1,$2,$3,$4,$5)", uuid.NewV7(), w, p.id, ids[n], b)
		if n < 2 {
			exec(t, pool, "INSERT INTO issue_labels(id,workspace_id,project_id,issue_id,label_id) VALUES($1,$2,$3,$4,$5)", uuid.NewV7(), w, p.id, ids[n], l)
			exec(t, pool, "INSERT INTO issue_assignees(id,workspace_id,project_id,issue_id,assignee_id) VALUES($1,$2,$3,$4,$5)", uuid.NewV7(), w, p.id, ids[n], a)
		}
	}
	// Deleted associations must neither create groups nor prevent the none group.
	exec(t, pool, "INSERT INTO issue_labels(id,workspace_id,project_id,issue_id,label_id,deleted_at) VALUES($1,$2,$3,$4,$5,now())", uuid.NewV7(), w, p.id, ids[4], l)
	in := app.ListRead{ActorID: a, Directories: app.ListDirectories{Projects: []app.VisibleProject{{ID: p.id}}, States: []app.ListState{{ID: p.state, Group: "backlog", Key: "backlog"}}}, Query: domain.ListQuery{Order: "sort_order", Limit: 1, GroupBy: "label_id", Filters: domain.ListFilters{SubIssues: true}}}
	for _, tc := range []struct {
		field domain.ListGroup
		keys  []string
	}{{"label_id", []string{l.String(), c.String(), "none"}}, {"assignee_id", []string{a.String(), b.String(), "none"}}} {
		t.Run(string(tc.field), func(t *testing.T) {
			in.Query.GroupBy = tc.field
			rows, err := s.Groups(context.Background(), in)
			if err != nil {
				t.Fatal(err)
			}
			keys := slices.Clone(tc.keys[:2])
			slices.Sort(keys)
			keys = append(keys, "none")
			if len(rows) != 6 {
				t.Fatalf("per-group lookahead: %d rows", len(rows))
			}
			for n, key := range keys {
				want := ids[:2]
				total := int64(2)
				if key == tc.keys[1] {
					total = 4
				}
				if key == "none" {
					want = ids[4:]
				}
				part := rows[n*2 : n*2+2]
				checkFlatIDs(t, part, want)
				for _, r := range part {
					if r.GroupKey != key || r.Total != 6 || r.GroupTotal != total {
						t.Fatalf("group counts %+v", r)
					}
					if r.Issue.AssigneeIDs == nil || r.Issue.LabelIDs == nil {
						t.Fatal("nil arrays")
					}
					if r.Issue.ID == ids[0] && (len(r.Issue.LabelIDs) != 2 || len(r.Issue.AssigneeIDs) != 2) {
						t.Fatalf("incomplete arrays %+v", r.Issue)
					}
				}
			}
		})
	}
	t.Run("intersection continuation", func(t *testing.T) {
		in.Query.GroupBy = "label_id"
		key := c.String()
		in.Query.GroupKey = &key
		in.Query.Filters.Labels = domain.IDFilter{IDs: []uuid.UUID{l}}
		checkGroupPages(t, s, in, ids[:2], 2)
		missing := uuid.NewV7().String()
		in.Query.GroupKey = &missing
		rows, err := s.Groups(context.Background(), in)
		if err != nil || len(rows) != 0 {
			t.Fatalf("absent group %+v %v", rows, err)
		}
	})
}

func TestIssueGroupKeyOrder(t *testing.T) {
	s, pool := newStore(t)
	a := newAccount(t, pool, "group-order")
	w := newWorkspace(t, pool, "group-order")
	p := newProject(t, pool, w, "GORDER")
	priorities := []string{"urgent", "high", "medium", "low", "none"}
	states := []string{"backlog", "unstarted", "started", "completed", "cancelled"}
	dirs := app.ListDirectories{Projects: []app.VisibleProject{{ID: p.id}}}
	ids := make([]uuid.UUID, 5)
	for n := range ids {
		state := uuid.NewV7()
		exec(t, pool, "INSERT INTO states(id,workspace_id,project_id,name,color,\"group\") VALUES($1,$2,$3,$4,'#000000',$4)", state, w, p.id, states[n])
		dirs.States = append(dirs.States, app.ListState{ID: state, Group: domain.StateGroup(states[n]), Key: states[n]})
		ids[n] = uuid.NewV7()
		newIssue(t, pool, p, ids[n], n+1, a)
		exec(t, pool, "UPDATE issues SET state_id=$2,priority=$3,target_date=$4,created_by_id=$5 WHERE id=$1", ids[n], state, priorities[n], []any{"2026-03-01", nil, "2025-01-01", nil, "2026-02-01"}[n], []any{a, nil, a, nil, a}[n])
	}
	stateKeys := make([]string, len(dirs.States))
	for n, state := range dirs.States {
		stateKeys[n] = state.ID.String()
	}
	slices.Sort(stateKeys)
	in := app.ListRead{ActorID: a, Directories: dirs, Query: domain.ListQuery{Order: "sort_order", Limit: 100, Filters: domain.ListFilters{SubIssues: true}}}
	for _, tc := range []struct {
		field domain.ListGroup
		keys  []string
	}{{"priority", priorities}, {"state_group", states}, {"state_id", stateKeys}, {"target_date", []string{"2025-01-01", "2026-02-01", "2026-03-01", "none"}}, {"created_by_id", []string{a.String(), "none"}}, {"project_id", []string{p.id.String()}}} {
		t.Run(string(tc.field), func(t *testing.T) {
			in.Query.GroupBy = tc.field
			rows, err := s.Groups(context.Background(), in)
			if err != nil {
				t.Fatal(err)
			}
			var keys []string
			for _, r := range rows {
				if len(keys) == 0 || keys[len(keys)-1] != r.GroupKey {
					keys = append(keys, r.GroupKey)
				}
			}
			if !slices.Equal(keys, tc.keys) {
				t.Fatalf("keys %v want %v", keys, tc.keys)
			}
		})
	}
	in.Query.GroupBy = "target_date"
	key := "none"
	in.Query.GroupKey = &key
	in.Query.Order = "created_at"
	in.Query.Filters.Target = []domain.DateFilter{{None: true}}
	checkGroupPages(t, s, in, []uuid.UUID{ids[1], ids[3]}, 2)
}

func TestIssueGroupPrecision(t *testing.T) {
	s, pool := newStore(t)
	a := newAccount(t, pool, "group-precision")
	w := newWorkspace(t, pool, "group-precision")
	p := newProject(t, pool, w, "GPREC")
	ids := []uuid.UUID{uuid.MustParse("00000000-0000-4000-8000-000000000003"), uuid.MustParse("00000000-0000-4000-8000-000000000002"), uuid.MustParse("00000000-0000-4000-8000-000000000001")}
	for n, id := range ids {
		newIssue(t, pool, p, id, n+1, a)
		exec(t, pool, "UPDATE issues SET sort_order=$2,created_at=$3::timestamptz,target_date=$4::date WHERE id=$1", id, []float64{1, math.Nextafter(1, 2), math.Nextafter(math.Nextafter(1, 2), 2)}[n], []string{"2255-06-05 23:47:34.740992Z", "2255-06-05 23:47:34.740993Z", "2255-06-05 23:47:34.740994Z"}[n], []any{"2026-01-01", "2026-01-02", nil}[n])
	}
	key := "none"
	in := app.ListRead{ActorID: a, Directories: app.ListDirectories{Projects: []app.VisibleProject{{ID: p.id}}, States: []app.ListState{{ID: p.state, Group: "backlog", Key: "backlog"}}}, Query: domain.ListQuery{GroupBy: "label_id", GroupKey: &key, Limit: 1, Filters: domain.ListFilters{SubIssues: true}}}
	for _, order := range []domain.ListOrder{"sort_order", "created_at", "-created_at", "target_date", "-target_date"} {
		t.Run(string(order), func(t *testing.T) {
			in.Query.Order = order
			want := slices.Clone(ids)
			if order == "-created_at" {
				slices.Reverse(want)
			}
			if order == "-target_date" {
				want = []uuid.UUID{ids[1], ids[0], ids[2]}
			}
			checkGroupPages(t, s, in, want, 3)
		})
	}
}

// Each step goes through the public canonical cursor, checks the lookahead,
// and compares fixed IDs rather than using the production ordering algorithm.
func checkGroupPages(t *testing.T, s interface {
	Groups(context.Context, app.ListRead) ([]app.ListRecord, error)
}, in app.ListRead, want []uuid.UUID, total int64) {
	t.Helper()
	in.Query.Limit = 1
	in.Query.Cursor = nil
	for n, id := range want {
		rows, err := s.Groups(context.Background(), in)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != min(2, len(want)-n) {
			t.Fatalf("page %d rows=%d", n, len(rows))
		}
		if rows[0].Issue.ID != id || rows[0].GroupTotal != total {
			t.Fatalf("page %d got %+v want %s total %d", n, rows[0], id, total)
		}
		encoded, err := domain.EncodeListCursor(rows[0].Position)
		if err != nil {
			t.Fatal(err)
		}
		c, err := domain.DecodeListCursor(encoded, in.Query.Order)
		if err != nil {
			t.Fatal(err)
		}
		in.Query.Cursor = &c
	}
	rows, err := s.Groups(context.Background(), in)
	if err != nil || len(rows) != 0 {
		t.Fatalf("past end %+v %v", rows, err)
	}
}
