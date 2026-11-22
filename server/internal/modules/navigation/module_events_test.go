package navigation_test

import (
	"reflect"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation"
)

// The module's subscribers, in the order bootstrap's dispatcher delivers to them, each with the types it takes: the
// deletion of a deleted project's or workspace's rows (P3b spec 2.7), the favorites' cascades (P3c spec 2.6), and the
// forgetting of every account's recent visits of a deleted work item (P3d spec 2.6, R10). Built over no pool and no
// inserter: none of them writes before an event comes.
func TestTheModulesSubscribers(t *testing.T) {
	type subscriber struct {
		Name  string
		Types []string
	}
	var got []subscriber
	for _, s := range navigation.Subscribers(navigation.Provide(nil), nil) {
		got = append(got, subscriber{s.Name, s.Types})
	}
	want := []subscriber{
		{"navigation_delete_rows_on_deletion", []string{"project.deleted", "workspace.deleted"}},
		{"navigation_favorite_cascades", []string{"planning.cycle_archived", "planning.cycle_deleted", "planning.module_archived",
			"planning.module_deleted", "project.archived"}},
		{"navigation_forget_deleted_issue_visits", []string{"issue.deleted"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("subscribers %+v; want %+v", got, want)
	}
}
