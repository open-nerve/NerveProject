package app_test

import (
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// createProject only inserts, so it reads the clock once, before its
// transaction and its locks (P3's ruling (c)): no row it reads under them
// has a time its rows must follow. Every row it writes has that one time.
func TestCreatingAProjectReadsTheClockBeforeItsTransaction(t *testing.T) {
	uc, f := newCreate()
	if _, err := uc.Execute(as(alice), "acme", domain.NewProject{Name: "Web", Identifier: "WEB", LeadID: &bob}); err != nil {
		t.Fatal(err)
	}
	if len(f.log.calls) < 2 || f.log.calls[0] != "Now" || f.log.calls[1] != "Begin" || slices.Contains(f.log.calls[1:], "Now") {
		t.Errorf("calls %q; want the clock read once, first, before the transaction", f.log.calls)
	}
	for _, r := range f.projects.members {
		if r.Now != clockNow {
			t.Errorf("a membership at %v, want %v", r.Now, clockNow)
		}
	}
	for _, r := range f.projects.prefs {
		if r.Now != clockNow {
			t.Errorf("display settings at %v, want %v", r.Now, clockNow)
		}
	}
	for _, r := range f.projects.states {
		if r.Now != clockNow {
			t.Errorf("a state at %v, want %v", r.Now, clockNow)
		}
	}
}
