package bootstrap

import (
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// moduleActions are the actions each module declares (M3 design 3.4). A
// module with actions adds its line here.
func moduleActions() map[string][]shared.Action {
	return map[string][]shared.Action{
		"workspace": workspace.Actions(),
	}
}

// actionViolations reports where the modules' actions and the rule table's
// keys part (M3 design 3.4): an action without a row, which the Authorizer
// would refuse at run time; a row without an action, which nothing asks
// for; an action declared twice, in one module or in two.
func actionViolations(modules map[string][]shared.Action, rules []shared.Action) []string {
	var found []string
	declared := map[shared.Action]string{}
	for _, module := range slices.Sorted(maps.Keys(modules)) {
		for _, action := range modules[module] {
			if other, ok := declared[action]; ok {
				found = append(found, fmt.Sprintf("action %q is declared by %s and by %s", action, other, module))
				continue
			}
			declared[action] = module
			if !slices.Contains(rules, action) {
				found = append(found, fmt.Sprintf("action %q of %s has no row in the rule table", action, module))
			}
		}
	}
	for _, rule := range rules {
		if _, ok := declared[rule]; !ok {
			found = append(found, fmt.Sprintf("the rule table's row %q is no module's action", rule))
		}
	}
	return found
}

// The union of the modules' Actions() is exactly the rule table's keys, and
// no action is declared twice (M3 design 3.4, 9.1).
func TestEveryActionHasARuleAndEveryRuleAnAction(t *testing.T) {
	if len(access.RuleKeys()) == 0 {
		t.Fatal("the rule table has no row")
	}
	for _, v := range actionViolations(moduleActions(), access.RuleKeys()) {
		t.Error(v)
	}
}

// Each check of actionViolations fails on its counterexample.
func TestActionViolationsCatchesEachMismatch(t *testing.T) {
	const read, update, create = shared.Action("workspace.read"), shared.Action("workspace.update"), shared.Action("project.create")
	if got := actionViolations(map[string][]shared.Action{"workspace": {read}, "project": {create}}, []shared.Action{create, read}); len(got) != 0 {
		t.Fatalf("a matching layout: %q, want none", got)
	}
	tests := []struct {
		name    string
		modules map[string][]shared.Action
		rules   []shared.Action
		want    string
	}{
		{"an action without a row", map[string][]shared.Action{"workspace": {read, update}}, []shared.Action{read},
			`action "workspace.update" of workspace has no row in the rule table`},
		{"a row without an action", map[string][]shared.Action{"workspace": {read}}, []shared.Action{read, update},
			`the rule table's row "workspace.update" is no module's action`},
		{"an action of two modules", map[string][]shared.Action{"workspace": {read}, "project": {read}}, []shared.Action{read},
			`action "workspace.read" is declared by project and by workspace`},
		{"an action twice in one module", map[string][]shared.Action{"workspace": {read, read}}, []shared.Action{read},
			`action "workspace.read" is declared by workspace and by workspace`},
	}
	for _, tt := range tests {
		if got := actionViolations(tt.modules, tt.rules); !slices.Equal(got, []string{tt.want}) {
			t.Errorf("%s: %q, want %q", tt.name, got, tt.want)
		}
	}
}
