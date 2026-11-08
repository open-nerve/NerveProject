package domain

import (
	"reflect"
	"testing"
	"uuid"
)

// A work item taken out of a cycle or a module that is deleted writes the row of its leaving it (M6 P3a spec 2.4,
// 2.5): deleted, from the name to "", the id the old identifier, Plane's sentence cut before the name: the row a PATCH
// that takes it out writes.
func TestLeavingADeletedCycleOrModule(t *testing.T) {
	sprint, auth := Named{ID: uuid.NewV7(), Name: "Sprint"}, Named{ID: uuid.NewV7(), Name: "Auth"}
	cycles, modules, empty := "cycles", "modules", ""
	for _, tc := range []struct {
		name             string
		got, want, patch Activity
	}{
		{"cycle", CycleLeft(sprint), Activity{Field: &cycles, Verb: VerbDeleted, OldValue: &sprint.Name, NewValue: &empty, OldIdentifier: &sprint.ID,
			Comment: "removed this work item from cycle "}, UpdateActivity(Update{Cycle: &Changed[*Named]{Old: &sprint}})[0]},
		{"module", ModuleLeft(auth), Activity{Field: &modules, Verb: VerbDeleted, OldValue: &auth.Name, NewValue: &empty, OldIdentifier: &auth.ID,
			Comment: "removed this work item from module "}, UpdateActivity(Update{ModulesRemoved: []Named{auth}})[0]},
	} {
		if !reflect.DeepEqual(tc.got, tc.want) || !reflect.DeepEqual(tc.got, tc.patch) {
			t.Errorf("leaving the %s: %+v; want %+v, as a PATCH's %+v", tc.name, tc.got, tc.want, tc.patch)
		}
	}
}
