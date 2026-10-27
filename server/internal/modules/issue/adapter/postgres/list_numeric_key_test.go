package postgresadapter

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

func TestListNumericCursor(t *testing.T) {
	for _, tc := range []struct {
		value float64
		key   int64
	}{
		{-math.MaxFloat64, -9218868437227405311}, {-1, -4607182418800017408},
		{-math.SmallestNonzeroFloat64, -1}, {math.Copysign(0, -1), 0}, {0, 0},
		{math.SmallestNonzeroFloat64, 1}, {math.Float64frombits(0x0010000000000000), 4503599627370496},
		{1, 4607182418800017408}, {math.MaxFloat64, 9218868437227405311},
	} {
		got, err := sortOrderNumericKey(tc.value)
		if err != nil || got != tc.key {
			t.Fatalf("float %g key %d; want %d: %v", tc.value, got, tc.key, err)
		}
	}
	for _, value := range []float64{math.Inf(1), math.Inf(-1), math.NaN()} {
		if _, err := sortOrderNumericKey(value); err == nil {
			t.Fatal("nonfinite accepted")
		}
	}
	for _, order := range []domain.ListOrder{"sub_issues_count", "-sub_issues_count", "link_count", "-link_count"} {
		for _, count := range []int64{9007199254740991, 9007199254740992, 9007199254740993, math.MaxInt64} {
			p := gen.FlatIssuesParams{OrderKey: strings.TrimPrefix(string(order), "-")}
			cursor := domain.ListCursor{Order: order, Key: json.RawMessage(strconv.FormatInt(count, 10)), ID: uuid.NewV7()}
			if err := cursorParameters(&p, cursor, cursor.Order); err != nil || p.CursorNum == nil || *p.CursorNum != count {
				t.Fatalf("count %d cursor %+v, %v", count, p, err)
			}
			row := gen.FlatIssuesRow{ID: cursor.ID, SubIssuesCount: 47, LinkCount: 31}
			if strings.TrimPrefix(string(order), "-") == "link_count" {
				row.LinkCount = count
			} else {
				row.SubIssuesCount = count
			}
			position, err := listPosition(row, order)
			if err != nil || string(position.Key) != strconv.FormatInt(count, 10) {
				t.Fatalf("%s row count %d position %+v %v", order, count, position, err)
			}
			in := app.ListRead{ActorID: uuid.NewV7(), Query: domain.ListQuery{Order: cursor.Order, Cursor: &cursor, Limit: 1, GroupBy: "label_id", GroupKey: new("none")}}
			flat := in
			flat.Query.GroupBy, flat.Query.GroupKey = "", nil
			f, err := flatParameters(flat)
			if err != nil || f.CursorNum == nil || *f.CursorNum != count || f.PageSize != 2 || f.Descending != strings.HasPrefix(string(order), "-") {
				t.Fatalf("flat %s count %d: %+v %v", order, count, f, err)
			}
			g, err := groupParameters(in)
			if err != nil || g.CursorNum == nil || *g.CursorNum != count || g.PageSize != 2 {
				t.Fatalf("group typed count/lookahead %d: %+v %v", count, g, err)
			}
			in.Query.SubGroupBy, in.Query.SubGroupKey = "assignee_id", new("none")
			s, err := subgroupParameters(in)
			if err != nil || s.CursorNum == nil || *s.CursorNum != count || s.PageSize != 2 {
				t.Fatalf("subgroup typed count/lookahead %d: %+v %v", count, s, err)
			}
		}
	}
}
