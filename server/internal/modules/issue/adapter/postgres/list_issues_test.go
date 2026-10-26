package postgresadapter_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// Each filter has an admitted row and a neighbor; filtering after LIMIT
// loses the admitted row when a rejected neighbor sorts before it.
func TestFlatIssueList(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	a, b := newAccount(t, pool, "flat-a"), newAccount(t, pool, "flat-b")
	w := newWorkspace(t, pool, "flat")
	p, other := newProject(t, pool, w, "FLAT"), newProject(t, pool, w, "OTHER")
	l, m := newLabel(t, pool, p, "A"), newLabel(t, pool, p, "B")
	ids := make([]uuid.UUID, 9)
	for n := range ids {
		ids[n] = uuid.NewV7()
		newIssue(t, pool, p, ids[n], n+1, a)
	}
	exec(t, pool, "UPDATE issues SET priority='urgent', start_date='2026-01-02', target_date='2026-02-02', sort_order=20 WHERE id=$1", ids[0])
	exec(t, pool, "UPDATE issues SET created_by_id=$2, sort_order=10 WHERE id=$1", ids[1], b)
	exec(t, pool, "UPDATE issues SET parent_id=$2 WHERE id=ANY($1::uuid[])", ids[2:7], ids[0])
	exec(t, pool, "UPDATE issues SET created_by_id=$2 WHERE id=$1", ids[3], b)
	exec(t, pool, "UPDATE issues SET archived_at=now() WHERE id=$1", ids[4])
	exec(t, pool, "UPDATE issues SET deleted_at=now() WHERE id=$1", ids[5])
	triage := uuid.NewV7()
	exec(t, pool, "INSERT INTO states(id,workspace_id,project_id,name,color,\"group\") VALUES($1,$2,$3,'Triage','#000000','triage')", triage, w, p.id)
	exec(t, pool, "UPDATE issues SET state_id=$2 WHERE id=$1", ids[6], triage)
	exec(t, pool, "UPDATE issues SET parent_id=$2 WHERE id=$1", ids[7], ids[2])
	exec(t, pool, "UPDATE issues SET created_by_id=NULL WHERE id=$1", ids[8])
	outside := uuid.NewV7()
	newIssue(t, pool, other, outside, 1, a)
	for _, target := range []uuid.UUID{a, b} {
		exec(t, pool, "INSERT INTO issue_assignees(id,workspace_id,project_id,issue_id,assignee_id) VALUES($1,$2,$3,$4,$5)", uuid.NewV7(), w, p.id, ids[0], target)
	}
	for _, target := range []uuid.UUID{l, m} {
		exec(t, pool, "INSERT INTO issue_labels(id,workspace_id,project_id,issue_id,label_id) VALUES($1,$2,$3,$4,$5)", uuid.NewV7(), w, p.id, ids[0], target)
	}
	for _, table := range []string{"issue_assignees", "issue_labels", "issue_mentions", "issue_subscribers"} {
		column, target := "assignee_id", b
		switch table {
		case "issue_labels":
			column, target = "label_id", l
		case "issue_mentions":
			column = "mention_id"
		case "issue_subscribers":
			column = "subscriber_id"
		}
		exec(t, pool, "INSERT INTO "+table+"(id,workspace_id,project_id,issue_id,"+column+",deleted_at) VALUES($1,$2,$3,$4,$5,now())", uuid.NewV7(), w, p.id, ids[1], target)
		if table == "issue_mentions" || table == "issue_subscribers" {
			exec(t, pool, "INSERT INTO "+table+"(id,workspace_id,project_id,issue_id,"+column+") VALUES($1,$2,$3,$4,$5)", uuid.NewV7(), w, p.id, ids[0], b)
		}
	}
	dirs := app.ListDirectories{Projects: []app.VisibleProject{{ID: p.id}}, States: []app.ListState{{ID: p.state, Group: "backlog", Key: "backlog"}}, Labels: []app.ListName{{ID: l, Key: "a"}, {ID: m, Key: "b"}}, Assignees: []app.ListName{{ID: a, Key: "a"}, {ID: b, Key: "b"}}}
	base := func() app.ListRead {
		return app.ListRead{ActorID: a, Directories: dirs, Query: domain.ListQuery{Order: "sort_order", Limit: 100, Filters: domain.ListFilters{SubIssues: true}}}
	}
	query := func(t *testing.T, in app.ListRead) []app.ListRecord {
		t.Helper()
		got, err := s.Flat(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	wantAll := []uuid.UUID{ids[1], ids[0], ids[2], ids[3], ids[7], ids[8]}
	rows := query(t, base())
	checkFlatIDs(t, rows, wantAll)
	if rows[1].Issue.SubIssuesCount != 3 || len(rows[1].Issue.AssigneeIDs) != 2 || len(rows[1].Issue.LabelIDs) != 2 || rows[1].Issue.DescriptionHTML != "" {
		t.Fatalf("projection/count/complete arrays: %+v", rows[1])
	}
	for _, row := range rows {
		if row.Total != 6 || row.Issue.AssigneeIDs == nil || row.Issue.LabelIDs == nil {
			t.Fatalf("total/empty arrays: %+v", row)
		}
	}
	for _, tc := range []struct {
		name  string
		value string
		want  []uuid.UUID
	}{
		{"state_id", p.state.String(), wantAll}, {"state_id", triage.String(), nil}, {"state_group", "backlog", wantAll}, {"state_group", "started", nil},
		{"priority", "urgent", ids[:1]}, {"priority", "low", nil}, {"label_id", l.String(), ids[:1]}, {"label_id", "none", []uuid.UUID{ids[1], ids[2], ids[3], ids[7], ids[8]}},
		{"assignee_id", b.String(), ids[:1]}, {"assignee_id", "none", []uuid.UUID{ids[1], ids[2], ids[3], ids[7], ids[8]}},
		{"created_by_id", b.String(), []uuid.UUID{ids[1], ids[3]}}, {"mention_id", b.String(), ids[:1]}, {"subscriber_id", b.String(), ids[:1]},
		{"start_date", "2026-01-02", ids[:1]}, {"start_date", "2026-01-03", nil}, {"target_date", "2026-02-02", ids[:1]}, {"target_date", "2026-02-03", nil},
		{"sub_issue", "false", []uuid.UUID{ids[1], ids[0], ids[8]}}, {"archived", "true", []uuid.UUID{ids[4]}},
	} {
		t.Run(tc.name+tc.value, func(t *testing.T) {
			in := base()
			var err error
			in.Query, err = domain.ParseListQuery(map[string][]string{tc.name: {tc.value}, "order_by": {"sort_order"}, "limit": {"100"}}, domain.ProjectList, a)
			if err != nil {
				t.Fatal(err)
			}
			checkFlatIDs(t, query(t, in), tc.want)
		})
	}
	t.Run("date conditions", func(t *testing.T) {
		for _, field := range []string{"start_date", "target_date"} {
			month := "01"
			if field == "target_date" {
				month = "02"
			}
			for _, tc := range []struct {
				values []string
				want   []uuid.UUID
			}{
				{[]string{"2026-" + month + "-01.."}, ids[:1]}, {[]string{"..2026-" + month + "-03"}, ids[:1]},
				{[]string{"2026-" + month + "-01..", "..2026-" + month + "-03"}, ids[:1]},
				{[]string{"2026-" + month + "-03..", "..2026-" + month + "-01"}, nil},
				{[]string{"2026-" + month + "-02..", "..2026-" + month + "-01"}, nil},
				{[]string{"none"}, []uuid.UUID{ids[1], ids[2], ids[3], ids[7], ids[8]}},
			} {
				in := base()
				q, err := domain.ParseListQuery(map[string][]string{field: tc.values, "order_by": {"sort_order"}, "limit": {"100"}}, domain.ProjectList, a)
				if err != nil {
					t.Fatal(err)
				}
				in.Query = q
				checkFlatIDs(t, query(t, in), tc.want)
			}
		}
	})
	t.Run("alternatives", func(t *testing.T) {
		for _, field := range []string{"label_id", "assignee_id"} {
			target := l
			if field == "assignee_id" {
				target = b
			}
			in := base()
			q, err := domain.ParseListQuery(map[string][]string{field: {target.String() + ",none"}, "order_by": {"sort_order"}, "limit": {"100"}}, domain.ProjectList, a)
			if err != nil {
				t.Fatal(err)
			}
			in.Query = q
			checkFlatIDs(t, query(t, in), wantAll)
		}
	})
	t.Run("combined before limit", func(t *testing.T) {
		exec(t, pool, "UPDATE issues SET sort_order=100000 WHERE id=$1", ids[0])
		defer exec(t, pool, "UPDATE issues SET sort_order=20 WHERE id=$1", ids[0])
		in := base()
		in.Query.Limit = 1
		in.Query.Filters.Labels = domain.IDFilter{IDs: []uuid.UUID{l}}
		in.Query.Filters.Priorities = []domain.Priority{"urgent"}
		rows := query(t, in)
		checkFlatIDs(t, rows, ids[:1])
		if rows[0].Total != 1 {
			t.Fatalf("filtered total %d", rows[0].Total)
		}
	})
	t.Run("limit plus one", func(t *testing.T) { in := base(); in.Query.Limit = 1; checkFlatIDs(t, query(t, in), wantAll[:2]) })
	t.Run("own only", func(t *testing.T) {
		in := base()
		in.Directories.Projects = []app.VisibleProject{{ID: p.id, OwnOnly: true}}
		got := query(t, in)
		checkFlatIDs(t, got, []uuid.UUID{ids[0], ids[2], ids[7]})
		if got[0].Issue.SubIssuesCount != 2 {
			t.Fatalf("own direct children including triage: %+v", got[0])
		}
		for _, order := range []domain.ListOrder{"sub_issues_count", "-sub_issues_count"} {
			in.Query.Order = order
			want := []uuid.UUID{ids[7], ids[2], ids[0]}
			if order[0] == '-' {
				slices.Reverse(want)
			}
			ordered := query(t, in)
			checkFlatIDs(t, ordered, want)
			for _, r := range ordered {
				if r.Issue.ID == ids[0] && r.Issue.SubIssuesCount != 2 {
					t.Fatalf("own sorting count %+v", r)
				}
			}
			checkFlatPages(t, s, in, want)
		}

	})
	t.Run("empty scope", func(t *testing.T) { in := base(); in.Directories.Projects = nil; checkFlatIDs(t, query(t, in), nil) })
	t.Run("project filter", func(t *testing.T) {
		in := base()
		in.Query.Filters.ProjectIDs = []uuid.UUID{other.id}
		checkFlatIDs(t, query(t, in), nil)
	})
	for _, relation := range []string{"assigned", "created", "subscribed"} {
		t.Run(relation, func(t *testing.T) {
			in := base()
			in.TargetMemberID = &b
			in.Query.Relation = relation
			want := ids[:1]
			if relation == "created" {
				want = []uuid.UUID{ids[1], ids[3]}
			}
			checkFlatIDs(t, query(t, in), want)
		})
	}
	for _, relation := range []string{"", "assigned", "invalid"} {
		t.Run("pair "+relation, func(t *testing.T) {
			in := base()
			in.Query.Relation = relation
			if relation != "assigned" {
				in.TargetMemberID = &b
			}
			if _, err := s.Flat(ctx, in); err == nil {
				t.Fatal("invalid relation/target accepted")
			}
		})
	}
	t.Run("transaction context", func(t *testing.T) {
		rollback := errors.New("rollback sentinel")
		err := postgres.NewTxManager(pool, time.Second).WithinTx(ctx, func(txctx context.Context) error {
			_, err := postgres.DB(txctx, pool).Exec(txctx, "UPDATE issues SET priority='low' WHERE id=$1", ids[0])
			if err != nil {
				return err
			}
			in := base()
			in.Query.Filters.Priorities = []domain.Priority{"low"}
			got, err := s.Flat(txctx, in)
			if err != nil {
				return err
			}
			checkFlatIDs(t, got, ids[:1])
			return rollback
		})
		if !errors.Is(err, rollback) {
			t.Fatalf("expected rollback sentinel, got %v", err)
		}
	})
	t.Run("cancelled", func(t *testing.T) {
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		_, err := s.Flat(cancelled, base())
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error %v", err)
		}
	})
}

func checkFlatIDs(t *testing.T, rows []app.ListRecord, want []uuid.UUID) {
	t.Helper()
	got := make([]uuid.UUID, len(rows))
	for n, r := range rows {
		got[n] = r.Issue.ID
	}
	if !slices.Equal(got, want) {
		t.Fatalf("IDs %v; want %v", got, want)
	}
}

// The direct sqlc boundary must still accept NULL filter arrays. This is
// independent of the adapter's own normalization, tested in-package.
func TestListArrayParameters(t *testing.T) {
	s, pool := newStore(t)
	a := newAccount(t, pool, "arrays")
	wid := newWorkspace(t, pool, "arrays")
	p := newProject(t, pool, wid, "ARRAY")
	id := uuid.NewV7()
	newIssue(t, pool, p, id, 1, a)
	base := gen.FlatIssuesParams{ActorID: a, ProjectIds: []uuid.UUID{p.id}, ProjectOwn: []bool{false}, StateIds: []uuid.UUID{p.state}, StateGroups: []string{"backlog"}, StateKeys: []string{"backlog"}, PageSize: 2, OrderKey: "sort_order", SubIssues: true}
	v := reflect.ValueOf(&base).Elem()
	for n := 0; n < v.NumField(); n++ {
		if f := v.Field(n); f.Kind() == reflect.Slice && f.IsNil() {
			f.Set(reflect.MakeSlice(f.Type(), 0, 0))
		}
	}
	for n := 0; n < v.NumField(); n++ {
		if v.Field(n).Kind() != reflect.Slice {
			continue
		}
		t.Run(v.Type().Field(n).Name, func(t *testing.T) {
			null, empty := base, base
			reflect.ValueOf(&null).Elem().Field(n).SetZero()
			field := reflect.ValueOf(&empty).Elem().Field(n)
			field.Set(reflect.MakeSlice(field.Type(), 0, 0))
			left, err := gen.New(pool).FlatIssues(context.Background(), null)
			if err != nil {
				t.Fatal(err)
			}
			right, err := gen.New(pool).FlatIssues(context.Background(), empty)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(left, right) {
				t.Fatalf("NULL versus empty: %+v / %+v", left, right)
			}
		})
	}
	in := app.ListRead{ActorID: a, Directories: app.ListDirectories{Projects: []app.VisibleProject{{ID: p.id}}, States: []app.ListState{{ID: p.state, Group: "backlog", Key: "backlog"}}}, Query: domain.ListQuery{Order: "sort_order", Limit: 1, Filters: domain.ListFilters{SubIssues: true}}}
	checkFlatIDs(t, func() []app.ListRecord {
		rows, err := s.Flat(context.Background(), in)
		if err != nil {
			t.Fatal(err)
		}
		return rows
	}(), []uuid.UUID{id})
}
