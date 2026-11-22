package app_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// navigationIssueRows is the store NavigationIssues reads through, as a fake: it answers those of its places and rows
// whose ids it is asked, in id order, as the store does, and records each read; err fails each read.
type navigationIssueRows struct {
	places map[uuid.UUID]app.NavigationIssuePlace
	rows   []app.NavigationIssueRow
	err    error
	asked  []string
}

func (f *navigationIssueRows) NavigationIssuePlaces(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]app.NavigationIssuePlace, error) {
	f.asked = append(f.asked, "places")
	if f.err != nil {
		return nil, f.err
	}
	out := map[uuid.UUID]app.NavigationIssuePlace{}
	for _, id := range ids {
		if p, found := f.places[id]; found {
			out[id] = p
		}
	}
	return out, nil
}

func (f *navigationIssueRows) NavigationIssueRows(_ context.Context, ids []uuid.UUID) ([]app.NavigationIssueRow, error) {
	f.asked = append(f.asked, "rows")
	if f.err != nil {
		return nil, f.err
	}
	var out []app.NavigationIssueRow
	for _, r := range f.rows {
		if slices.Contains(ids, r.ID) {
			out = append(out, r)
		}
	}
	return out, nil
}

// NavigationIssues answers what a caller sees now of the work items a recent visit names (M7 P3d spec 2.5, R3), by the
// grants he is given: of web's, which gail reads as a guest who sees only her own, hers but not alice's, nor one whose
// creator is not known; every one of web's to bob, its member; none of ops', which neither reads, nor of another
// workspace's project, which is among no grant. Each is answered as its row has it, its priority as written, and in
// triage while its undeleted intake row is not accepted (domain.Issue.InTriage): pending and declined are, accepted and
// none are not. The ids it is asked are the reader's, in one read.
func TestNavigationIssuesAnswerWhatTheReaderSees(t *testing.T) {
	gail, alice, bob := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	web, ops, far := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	row := func(project uuid.UUID, name string, creator *uuid.UUID, intake domain.IntakeStatus) app.NavigationIssueRow {
		r := app.NavigationIssueRow{ID: uuid.NewV7(), ProjectID: project, SequenceID: 3, Name: name, StateID: uuid.NewV7(),
			Priority: domain.PriorityUrgent, AssigneeIDs: []uuid.UUID{alice, bob}, CreatedByID: creator}
		if intake != "" {
			r.Intake = &domain.IntakeFacts{Status: intake}
		}
		return r
	}
	hers, alices, unknown := row(web, "Hers", &gail, ""), row(web, "Alice's", &alice, ""), row(web, "Unknown", nil, "")
	pending, declined, accepted := row(web, "Pending", &gail, domain.IntakePending), row(web, "Declined", &gail, domain.IntakeDeclined),
		row(web, "Accepted", &gail, domain.IntakeAccepted)
	opsHers, farHers := row(ops, "Ops'", &gail, ""), row(far, "Far", &gail, "")
	rows := &navigationIssueRows{rows: []app.NavigationIssueRow{hers, alices, unknown, pending, declined, accepted, opsHers, farHers}}
	issues := app.NewNavigationIssues(rows)
	ids := []uuid.UUID{hers.ID, alices.ID, unknown.ID, pending.ID, declined.ID, accepted.ID, opsHers.ID, farHers.ID, uuid.NewV7()}
	answer := func(r app.NavigationIssueRow, inTriage bool) app.NavigationIssue {
		return app.NavigationIssue{ID: r.ID, ProjectID: r.ProjectID, SequenceID: r.SequenceID, Name: r.Name, StateID: r.StateID, Priority: "urgent",
			AssigneeIDs: r.AssigneeIDs, InTriage: inTriage}
	}
	gails := []app.NavigationIssue{answer(hers, false), answer(pending, true), answer(declined, true), answer(accepted, false)}
	for _, tt := range []struct {
		name   string
		reader uuid.UUID
		grants []app.ProjectGrant
		want   []app.NavigationIssue
	}{
		{"gail, web's guest who sees only her own", gail, []app.ProjectGrant{{ProjectID: web, OwnOnly: true}}, gails},
		{"bob, web's member", bob, []app.ProjectGrant{{ProjectID: web}}, append(gails[:4:4], answer(alices, false), answer(unknown, false))},
		{"alice, who sees only her own in web", alice, []app.ProjectGrant{{ProjectID: web, OwnOnly: true}}, []app.NavigationIssue{answer(alices, false)}},
		{"gail, reading none", gail, nil, nil},
	} {
		rows.asked = nil
		got, err := issues.NavigationIssues(context.Background(), tt.reader, tt.grants, ids)
		want := map[uuid.UUID]app.NavigationIssue{}
		for _, i := range tt.want {
			want[i.ID] = i
		}
		if err != nil || len(got) != len(want) || !maps.EqualFunc(got, want, sameIssue) {
			t.Errorf("%s: NavigationIssues() = %+v, %v; want %+v", tt.name, got, err, want)
		}
		if !slices.Equal(rows.asked, []string{"rows"}) {
			t.Errorf("%s: the store was asked %v; want the rows once", tt.name, rows.asked)
		}
	}
}

// sameIssue tells whether a and b are the same answer, field by field.
func sameIssue(a, b app.NavigationIssue) bool {
	return a.ID == b.ID && a.ProjectID == b.ProjectID && a.SequenceID == b.SequenceID && a.Name == b.Name && a.StateID == b.StateID &&
		a.Priority == b.Priority && slices.Equal(a.AssigneeIDs, b.AssigneeIDs) && a.InTriage == b.InTriage
}

// The places are the store's, as it reads them, by id: no grants and no guest rule are asked (spec 2.7).
func TestNavigationIssuePlacesAreTheStores(t *testing.T) {
	item, other := uuid.NewV7(), uuid.NewV7()
	place := app.NavigationIssuePlace{WorkspaceID: uuid.NewV7(), ProjectID: uuid.NewV7()}
	rows := &navigationIssueRows{places: map[uuid.UUID]app.NavigationIssuePlace{item: place}}
	got, err := app.NewNavigationIssues(rows).NavigationIssuePlaces(context.Background(), []uuid.UUID{item, other})
	if err != nil || !maps.Equal(got, map[uuid.UUID]app.NavigationIssuePlace{item: place}) || !slices.Equal(rows.asked, []string{"places"}) {
		t.Errorf("NavigationIssuePlaces() = %+v, %v after %v; want %s's place in one read", got, err, rows.asked, item)
	}
}

// No ids are no read and no answer, of either; a failed read is its error, and no answer.
func TestNavigationIssuesReadNothingForNoIDsAndFailAsTheStore(t *testing.T) {
	rows := &navigationIssueRows{}
	issues := app.NewNavigationIssues(rows)
	gail, web := uuid.NewV7(), uuid.NewV7()
	grants := []app.ProjectGrant{{ProjectID: web}}
	for _, ids := range [][]uuid.UUID{nil, {}} {
		if got, err := issues.NavigationIssues(context.Background(), gail, grants, ids); err != nil || len(got) != 0 {
			t.Errorf("NavigationIssues(%v) = %+v, %v; want none", ids, got, err)
		}
		if got, err := issues.NavigationIssuePlaces(context.Background(), ids); err != nil || len(got) != 0 {
			t.Errorf("NavigationIssuePlaces(%v) = %+v, %v; want none", ids, got, err)
		}
	}
	if len(rows.asked) != 0 {
		t.Errorf("the store was asked %v; want no read for no ids", rows.asked)
	}
	failure := errors.New("connection reset")
	rows.err = failure
	if got, err := issues.NavigationIssues(context.Background(), gail, grants, []uuid.UUID{uuid.NewV7()}); !errors.Is(err, failure) || got != nil {
		t.Errorf("NavigationIssues() = %+v, %v; want %v and none", got, err, failure)
	}
	if got, err := issues.NavigationIssuePlaces(context.Background(), []uuid.UUID{uuid.NewV7()}); !errors.Is(err, failure) || got != nil {
		t.Errorf("NavigationIssuePlaces() = %+v, %v; want %v and none", got, err, failure)
	}
}
