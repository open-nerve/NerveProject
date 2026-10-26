package postgresadapter_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func TestIssueGroupArrayParameters(t *testing.T) {
	s, pool := newStore(t)
	a := newAccount(t, pool, "group-arrays")
	w := newWorkspace(t, pool, "group-arrays")
	p := newProject(t, pool, w, "GARRAY")
	id := uuid.NewV7()
	newIssue(t, pool, p, id, 1, a)
	base := gen.GroupIssuesParams{ActorID: a, ProjectIds: []uuid.UUID{p.id}, ProjectOwn: []bool{false}, StateIds: []uuid.UUID{p.state}, StateGroups: []string{"backlog"}, StateKeys: []string{"backlog"}, GroupBy: "label_id", OrderKey: "sort_order", PageSize: 2, SubIssues: true}
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
			left, err := gen.New(pool).GroupIssues(context.Background(), null)
			if err != nil {
				t.Fatal(err)
			}
			right, err := gen.New(pool).GroupIssues(context.Background(), empty)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(left, right) {
				t.Fatalf("NULL versus empty %+v / %+v", left, right)
			}
		})
	}
	in := app.ListRead{ActorID: a, Directories: app.ListDirectories{Projects: []app.VisibleProject{{ID: p.id, OwnOnly: true}}, States: []app.ListState{{ID: p.state, Group: "backlog", Key: "backlog"}}}, Query: domain.ListQuery{Order: "sort_order", GroupBy: "label_id", Limit: 1, Filters: domain.ListFilters{SubIssues: true}}}
	t.Run("transaction ctx", func(t *testing.T) {
		rollback := errors.New("rollback group read")
		err := postgres.NewTxManager(pool, time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
			if _, err := postgres.DB(ctx, pool).Exec(ctx, "UPDATE issues SET priority='high' WHERE id=$1", id); err != nil {
				return err
			}
			in.Query.Filters.Priorities = []domain.Priority{"high"}
			rows, err := s.Groups(ctx, in)
			if err != nil {
				return err
			}
			checkFlatIDs(t, rows, []uuid.UUID{id})
			if rows[0].Total != 1 || rows[0].GroupTotal != 1 {
				t.Fatalf("group totals %+v", rows)
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
		if _, err := s.Groups(ctx, in); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled error %v", err)
		}
	})
}
