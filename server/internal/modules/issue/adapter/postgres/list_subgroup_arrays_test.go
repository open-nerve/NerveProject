package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func TestIssueSubgroupsArrayParameters(t *testing.T) {
	s, pool, in, ids, _, _ := subgroupFixture(t)
	base := gen.SubgroupIssuesParams{ActorID: in.ActorID, ProjectIds: []uuid.UUID{in.Directories.Projects[0].ID}, ProjectOwn: []bool{false}, StateIds: []uuid.UUID{in.Directories.States[0].ID}, StateGroups: []string{"backlog"}, StateKeys: []string{"backlog"}, GroupBy: "label_id", SubGroupBy: "assignee_id", OrderKey: "sort_order", PageSize: 100, SubIssues: true}
	// Direct sqlc NULL filters are distinct from the adapter's nonnil arrays.
	// Each setter takes both inputs from nil to an explicit empty array.
	for _, tc := range []struct {
		name  string
		empty func(*gen.SubgroupIssuesParams)
	}{
		{"projects", func(p *gen.SubgroupIssuesParams) { p.FilterProjects = []uuid.UUID{} }},
		{"states", func(p *gen.SubgroupIssuesParams) { p.FilterStates = []uuid.UUID{} }},
		{"groups", func(p *gen.SubgroupIssuesParams) { p.FilterGroups = []string{} }},
		{"priorities", func(p *gen.SubgroupIssuesParams) { p.FilterPriorities = []string{} }},
		{"creators", func(p *gen.SubgroupIssuesParams) { p.FilterCreators = []uuid.UUID{} }},
		{"mentions", func(p *gen.SubgroupIssuesParams) { p.FilterMentions = []uuid.UUID{} }},
		{"subscribers", func(p *gen.SubgroupIssuesParams) { p.FilterSubscribers = []uuid.UUID{} }},
		{"labels", func(p *gen.SubgroupIssuesParams) { p.FilterLabels = []uuid.UUID{} }},
		{"assignees", func(p *gen.SubgroupIssuesParams) { p.FilterAssignees = []uuid.UUID{} }},
		{"start", func(p *gen.SubgroupIssuesParams) {
			p.StartNone, p.StartFrom, p.StartTo = []bool{}, []string{}, []string{}
		}},
		{"target", func(p *gen.SubgroupIssuesParams) {
			p.TargetNone, p.TargetFrom, p.TargetTo = []bool{}, []string{}, []string{}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			empty := base
			tc.empty(&empty)
			left, err := gen.New(pool).SubgroupIssues(context.Background(), base)
			if err != nil {
				t.Fatal(err)
			}
			right, err := gen.New(pool).SubgroupIssues(context.Background(), empty)
			if err != nil {
				t.Fatal(err)
			}
			if len(left) != 11 || len(right) != 11 {
				t.Fatalf("NULL/empty rows %d/%d want11", len(left), len(right))
			}
			for n, r := range left {
				v := right[n]
				if r.ID != v.ID || r.Total != 5 || v.Total != 5 || r.GroupKey != v.GroupKey || r.SubGroupKey != v.SubGroupKey || r.CellTotal != v.CellTotal || !slices.Equal(r.LabelIds, v.LabelIds) || !slices.Equal(r.AssigneeIds, v.AssigneeIds) {
					t.Fatalf("NULL/empty row %+v / %+v", r, v)
				}
			}
		})
	}
	t.Run("empty scope", func(t *testing.T) {
		in := in
		in.Directories.Projects = nil
		rows, err := s.Subgroups(context.Background(), in)
		if err != nil || rows == nil || len(rows) != 0 {
			t.Fatalf("empty scope %+v %v", rows, err)
		}
	})
	t.Run("transaction ctx", func(t *testing.T) {
		rollback := errors.New("rollback cell read")
		err := postgres.NewTxManager(pool, time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
			if _, err := postgres.DB(ctx, pool).Exec(ctx, "UPDATE issues SET priority='high' WHERE id=$1", ids[1]); err != nil {
				return err
			}
			in.Query.Filters.Priorities = []domain.Priority{"high"}
			rows, err := s.Subgroups(ctx, in)
			if err != nil {
				return err
			}
			checkFlatIDs(t, rows, []uuid.UUID{ids[1], ids[1], ids[1], ids[1]})
			for _, r := range rows {
				if r.Total != 1 || r.GroupTotal != 1 || r.SubgroupTotal != 1 || r.CellTotal != 1 {
					t.Fatalf("transaction totals %+v", r)
				}
			}
			return rollback
		})
		if !errors.Is(err, rollback) {
			t.Fatalf("rollback sentinel: %v", err)
		}
	})
	t.Run("cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := s.Subgroups(ctx, in); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled error %v", err)
		}
	})
}
