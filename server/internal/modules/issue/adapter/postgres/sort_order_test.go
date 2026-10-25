package postgresadapter_test

import (
	"context"
	"math"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

func TestIssueSortOrderPatch(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	id := uuid.NewV7()
	createWebIssue(t, s, web, id, alice)
	for _, value := range []float64{0, -12.5, math.MaxFloat64, math.Nextafter(1, 2)} {
		if err := s.UpdateIssue(context.Background(), app.IssueFields{ID: id, Name: "sorted", StateID: web.state, Priority: domain.PriorityLow, SortOrder: value, By: alice, At: now}); err != nil {
			t.Fatal(err)
		}
		got, found, err := s.GetIssue(context.Background(), id)
		if err != nil || !found || got.SortOrder != value || got.Name != "sorted" || got.StateID != web.state {
			t.Fatal("stored sort", got, found, err)
		}
	}
}
