package app_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// A list ordered by the cycles' or the modules' names reads their names
// from planning in the list's snapshot (M6 P3b spec R2): of the projects the
// other directories are of, all of them for the member's list too, keyed in
// Go as states and labels are, after the other directories and before the
// rows. The cycles only for ±cycle_name, the modules only for ±module_name,
// neither for any other order. A failed read reaches no rows.
func TestListPlanningDirectories(t *testing.T) {
	cycle, archived, module, alpha := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	cycles := []app.PlanningName{{ID: cycle, Name: "ÄBC Sprint"}, {ID: archived, Name: "Old ZULU"}}
	modules := []app.PlanningName{{ID: module, Name: "Core"}, {ID: alpha, Name: "ALPHA"}}
	wantCycles := []app.ListName{{ID: cycle, Key: "Äbc sprint"}, {ID: archived, Key: "old zulu"}}
	wantModules := []app.ListName{{ID: module, Key: "core"}, {ID: alpha, Key: "alpha"}}
	for _, tc := range []struct {
		order           domain.ListOrder
		read            string
		cycles, modules []app.ListName
	}{
		{"cycle_name", "cycles", wantCycles, nil},
		{"-cycle_name", "cycles", wantCycles, nil},
		{"module_name", "modules", nil, wantModules},
		{"-module_name", "modules", nil, wantModules},
		{"label_name", "", nil, nil},
		{"-state_name", "", nil, nil},
		{"sort_order", "", nil, nil},
		{"-created_at", "", nil, nil},
	} {
		for _, member := range []bool{false, true} {
			t.Run(string(tc.order)+map[bool]string{false: "/project", true: "/member"}[member], func(t *testing.T) {
				_, f, in := listingFixture(t)
				in.Query.Order = tc.order
				if member {
					other, target := uuid.NewV7(), uuid.NewV7()
					in.TargetMemberID, in.Query.Relation = &target, "created"
					in.Projects = append(in.Projects, app.VisibleProject{ID: other})
					f.pids = []uuid.UUID{web, other}
				}
				directory := fakeDirectory{ports: f, cycles: cycles, modules: modules}
				l := app.NewListing(app.ListingDeps{Attachments: noIssueAttachments{}, Rows: f, States: f, Labels: f, Assignees: f, Profiles: f, Issues: f, Planning: directory})
				if _, err := l.Read(f.ctx, in); err != nil {
					t.Fatal(err)
				}
				want := []string{"states", "labels", "assignees", "profiles", "flat"}
				if tc.read != "" {
					want = slices.Insert(want, 4, tc.read)
				}
				if !slices.Equal(f.calls, want) {
					t.Fatalf("calls %v; want %v", f.calls, want)
				}
				got := f.reads[0].Directories
				if !reflect.DeepEqual(got.Cycles, tc.cycles) || !reflect.DeepEqual(got.Modules, tc.modules) {
					t.Fatalf("cycles %+v, modules %+v; want %+v, %+v", got.Cycles, got.Modules, tc.cycles, tc.modules)
				}
			})
		}
		if tc.read == "" {
			continue
		}
		t.Run(string(tc.order)+"/failed", func(t *testing.T) {
			_, f, in := listingFixture(t)
			in.Query.Order = tc.order
			f.fail = tc.read
			l := app.NewListing(app.ListingDeps{Attachments: noIssueAttachments{}, Rows: f, States: f, Labels: f, Assignees: f, Profiles: f, Issues: f, Planning: fakeDirectory{ports: f, cycles: cycles, modules: modules}})
			if _, err := l.Read(f.ctx, in); !errors.Is(err, errDisk) || len(f.reads) != 0 {
				t.Fatalf("err %v, reads %d; want the directory's failure and no rows read", err, len(f.reads))
			}
		})
	}
}
