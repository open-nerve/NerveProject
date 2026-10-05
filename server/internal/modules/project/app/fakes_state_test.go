package app_test

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// stateSince is when the fakes' states were made, as stored: no clock's
// time.
var stateSince = time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)

// The states of web and ops, by id, the same in every fixture: made in
// this order, so their ids are too.
var webBacklog, webTodo, webStarted, webReview, webDone, webCancelled, opsBacklog = uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(),
	uuid.NewV7(), uuid.NewV7(), uuid.NewV7()

// fakeStates is the project store's states as the state operations' tests
// hold them, by id, beside fakeStore's projects, whose log, failures and
// changedAs it shares; the triage states are none of them, as the store
// reads none.
type fakeStates struct {
	*fakeStore
	states map[uuid.UUID]domain.State
}

// newStates is newWrites with web's states, Backlog its default, In
// Progress and Review the started group's, each other group's one, and
// ops's Backlog, its default.
func newStates() (*writeFixture, *fakeStates) {
	f := newWrites()
	s := &fakeStates{fakeStore: f.store, states: map[uuid.UUID]domain.State{}}
	for _, st := range []domain.State{
		{ID: webBacklog, ProjectID: webID, Name: "Backlog", Group: domain.GroupBacklog, Default: true, Sequence: 15000},
		{ID: webTodo, ProjectID: webID, Name: "Todo", Group: domain.GroupUnstarted, Sequence: 25000},
		{ID: webStarted, ProjectID: webID, Name: "In Progress", Group: domain.GroupStarted, Sequence: 35000},
		{ID: webReview, ProjectID: webID, Name: "Review", Group: domain.GroupStarted, Sequence: 40000},
		{ID: webDone, ProjectID: webID, Name: "Done", Group: domain.GroupCompleted, Sequence: 45000},
		{ID: webCancelled, ProjectID: webID, Name: "Cancelled", Group: domain.GroupCancelled, Sequence: 55000},
		{ID: opsBacklog, ProjectID: opsID, Name: "Backlog", Group: domain.GroupBacklog, Default: true, Sequence: 15000},
	} {
		st.WorkspaceID, st.Color, st.CreatedAt, st.UpdatedAt = acme.ID, "#60646C", stateSince, stateSince
		s.states[st.ID] = st
	}
	return f, s
}

// of are the states of project, in the store's order: by name, which
// neither a sort by sequence, either way, nor one by id gives, so that a
// use case that sorted them would answer another.
func (f *fakeStates) of(project uuid.UUID) []domain.State {
	var out []domain.State
	for _, s := range f.states {
		if s.ProjectID == project {
			out = append(out, s)
		}
	}
	slices.SortFunc(out, func(a, b domain.State) int { return cmp.Compare(a.Name, b.Name) })
	return out
}

func (f *fakeStates) GreatestSequence(ctx context.Context, projectID uuid.UUID) (*float64, error) {
	f.log.add(ctx, "GreatestSequence %s", projectID)
	if err := f.fail("GreatestSequence"); err != nil {
		return nil, err
	}
	var greatest *float64
	for _, s := range f.of(projectID) {
		if greatest == nil || s.Sequence > *greatest {
			greatest = &s.Sequence
		}
	}
	return greatest, nil
}

// CreateState stores r's state, as stored, unless its project has a state
// of its name: domain.ErrStateNameTaken.
func (f *fakeStates) CreateState(ctx context.Context, r app.StateRow) (domain.State, error) {
	f.log.add(ctx, "CreateState %s", stateRow(r))
	if err := f.fail("CreateState"); err != nil {
		return domain.State{}, err
	}
	for _, s := range f.of(r.ProjectID) {
		if s.Name == r.State.Name {
			return domain.State{}, domain.ErrStateNameTaken
		}
	}
	at := r.Now.Truncate(time.Microsecond)
	s := domain.State{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, Name: r.State.Name, Description: r.State.Description,
		Color: r.State.Color, Group: r.State.Group, Default: r.State.Default, Sequence: r.State.Sequence, CreatedAt: at, UpdatedAt: at}
	f.states[r.ID] = s
	if f.changedAs != (uuid.UUID{}) {
		s.ID = f.changedAs
	}
	return s, nil
}

// stateRow is r as CreateState logs it: every field but its id, which the
// use case makes.
func stateRow(r app.StateRow) string {
	return fmt.Sprintf("%s/%s %q %s %s at %v default %v %q by %s at %s", r.WorkspaceID, r.ProjectID, r.State.Name, r.State.Color, r.State.Group,
		r.State.Sequence, r.State.Default, r.State.Description, r.CreatedBy, r.Now.Format(timeFormat))
}
