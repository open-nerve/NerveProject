package app_test

import (
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A relation's events name, last, the end the request was on, requested_on, with the type that end reads (M7 P1b spec
// 2.3, R11): the work item an addition was asked on, though blocking is stored the other way round, as the other's
// blocked_by; and the end a removal through the relation's id wrote first, the one stored as issue_id (h.own()), whether
// the caller may change it or only the other. Either way it is the work item and the type of the first activity row the
// use case writes.
func TestRelationEventsNameTheEndTheyWereRequestedOn(t *testing.T) {
	t.Run("blocking asked on the work item, stored as the other's blocked_by", func(t *testing.T) {
		_, w, rel, id, r := newRelate()
		p := &recordedEvents{t: t}
		uc := app.NewCreateIssueRelations(app.CreateIssueRelationsDeps{Events: p, Locks: w.locks, Relations: rel, Clock: clockAt{at: clockNow, log: w.log}})
		if _, err := uc.Execute(as(alice), id, domain.Blocking, []uuid.UUID{r.far.ID}); err != nil {
			t.Fatal(err)
		}
		if s := rel.inserted[0]; s.IssueID != r.far.ID || s.RelatedID != id || s.Type != domain.BlockedBy || len(p.rows) != 1 {
			t.Fatalf("stored %+v, events %d; want far blocked_by the work item, one event", s, len(p.rows))
		}
		wantRequestedOn(t, p.rows[0], w.issues.activities[0], id, "blocking")
	})
	for _, otherOnly := range []bool{false, true} {
		t.Run(line("removal through the relation's id, the caller may change only the other end", otherOnly), func(t *testing.T) {
			_, w, rel := newUnrelate()
			r := rel.relations[0]
			if otherOnly {
				w.auth.answers = []error{0: shared.Forbidden(), 2: shared.Forbidden()}
			}
			p := &recordedEvents{t: t}
			uc := app.NewDeleteIssueRelation(app.DeleteIssueRelationDeps{Events: p, Locks: w.locks, Relations: rel, Clock: clockAt{at: clockNow, log: w.log}})
			if err := uc.Execute(as(alice), r.ID); err != nil || len(p.rows) != 1 {
				t.Fatalf("removal = %v, events %d; want one", err, len(p.rows))
			}
			wantRequestedOn(t, p.rows[0], w.issues.activities[0], r.IssueID, "blocked_by")
		})
	}
}

// wantRequestedOn holds e's last change the requesting end on, reading onType, on the New side of an addition, the Old
// side of a removal, and the use case's first activity row first of on, of onType.
func wantRequestedOn(t *testing.T, e shared.Event, first app.ActivityRow, on uuid.UUID, onType string) {
	t.Helper()
	c := e.Changes[len(e.Changes)-1]
	value, id, other, otherID := c.New, c.NewID, c.Old, c.OldID
	if e.Type == "issue.relation_removed" {
		value, id, other, otherID = c.Old, c.OldID, c.New, c.NewID
	}
	if c.Field != "requested_on" || value == nil || *value != onType || id == nil || *id != on || other != nil || otherID != nil {
		t.Errorf("%s's last change = %+v; want requested_on %s reading %s", e.Type, c, on, onType)
	}
	if first.IssueID != on || first.Activity.Field == nil || *first.Activity.Field != onType {
		t.Errorf("first activity row %+v; want it of %s, reading %s, as requested_on", first, on, onType)
	}
}
