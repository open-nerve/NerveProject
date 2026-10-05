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
// hold them, by id, beside fakeStore's projects, whose log, failures,
// reads of a row by its id and changedAs it shares; the triage states are
// none of them, as the store reads none. passOver makes the guarded
// writes, DeleteState and MarkDefaultState's second statement, write no
// row: "kept" leaves the state as it is, "gone" takes it away, as a
// deletion meanwhile would.
type fakeStates struct {
	*fakeStore
	states   map[uuid.UUID]domain.State
	passOver string
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

// StateByID is the state id, read again under the locks as f.reread says.
func (f *fakeStates) StateByID(ctx context.Context, id uuid.UUID) (domain.State, bool, error) {
	f.log.add(ctx, "StateByID %s", id)
	f.rowReadCount++
	again := f.rowReadCount > 1
	if err := f.fail("StateByID"); err != nil {
		return domain.State{}, false, err
	}
	if again && f.reread.err != nil {
		return domain.State{}, false, fmt.Errorf("StateByID: %w", f.reread.err)
	}
	s, ok := f.states[id]
	if !ok || (again && f.reread.gone) {
		return domain.State{}, false, nil
	}
	if again && f.reread.project != (uuid.UUID{}) {
		s.ProjectID = f.reread.project
	}
	if f.answersAs != (uuid.UUID{}) {
		s.ID = f.answersAs
	}
	return s, true, nil
}

func (f *fakeStates) CountGroupStates(ctx context.Context, projectID uuid.UUID, group domain.StateGroup) (int, error) {
	f.log.add(ctx, "CountGroupStates %s %s", projectID, group)
	if err := f.fail("CountGroupStates"); err != nil {
		return 0, err
	}
	n := 0
	for _, s := range f.of(projectID) {
		if s.Group == group {
			n++
		}
	}
	return n, nil
}

// UpdateState changes the state id as p gives, stored, unless another state
// of its project has the name p gives: domain.ErrStateNameTaken.
func (f *fakeStates) UpdateState(ctx context.Context, id uuid.UUID, p domain.StatePatch, by uuid.UUID, now time.Time) (domain.State, error) {
	f.log.add(ctx, "UpdateState %s %s by %s at %s", id, statePatch(p), by, now.Format(timeFormat))
	if err := f.fail("UpdateState"); err != nil {
		return domain.State{}, err
	}
	s, ok := f.states[id]
	if !ok {
		return domain.State{}, fmt.Errorf("UpdateState: no state %s", id)
	}
	if p.Name != nil {
		for _, other := range f.of(s.ProjectID) {
			if other.ID != id && other.Name == *p.Name {
				return domain.State{}, domain.ErrStateNameTaken
			}
		}
		s.Name = *p.Name
	}
	if p.Color != nil {
		s.Color = *p.Color
	}
	if p.Group != nil {
		s.Group = *p.Group
	}
	if p.Description != nil {
		s.Description = *p.Description
	}
	if p.Sequence != nil {
		s.Sequence = *p.Sequence
	}
	s.UpdatedAt = now.Truncate(time.Microsecond)
	f.states[id] = s
	if f.changedAs != (uuid.UUID{}) {
		s.ID = f.changedAs
	}
	return s, nil
}

// DeleteState deletes the state id, unless it is its project's default or
// f.passOver passes over it.
func (f *fakeStates) DeleteState(ctx context.Context, id, by uuid.UUID, now time.Time) (bool, error) {
	f.log.add(ctx, "DeleteState %s by %s at %s", id, by, now.Format(timeFormat))
	if err := f.fail("DeleteState"); err != nil {
		return false, err
	}
	s, ok := f.states[id]
	if !ok || s.Default || f.passedOver(id) {
		return false, nil
	}
	delete(f.states, id)
	return true, nil
}

// MarkDefaultState makes the state id of projectID its default, after the
// project's default the default no longer, unless f.passOver passes over
// the second statement: the first stays written, as the store leaves it for
// the caller to roll back.
func (f *fakeStates) MarkDefaultState(ctx context.Context, projectID, id, by uuid.UUID, now time.Time) (bool, error) {
	f.log.add(ctx, "MarkDefaultState %s %s by %s at %s", projectID, id, by, now.Format(timeFormat))
	if err := f.fail("MarkDefaultState"); err != nil {
		return false, err
	}
	at := now.Truncate(time.Microsecond)
	for _, s := range f.of(projectID) {
		if s.Default {
			s.Default, s.UpdatedAt = false, at
			f.states[s.ID] = s
		}
	}
	s, ok := f.states[id]
	if !ok || s.ProjectID != projectID || f.passedOver(id) {
		return false, nil
	}
	s.Default, s.UpdatedAt = true, at
	f.states[id] = s
	return true, nil
}

// passedOver reports whether f.passOver passes over the state id, taking it
// away for "gone".
func (f *fakeStates) passedOver(id uuid.UUID) bool {
	if f.passOver == "gone" {
		delete(f.states, id)
	}
	return f.passOver != ""
}

// statePatch is p as UpdateState logs it: each field it gives.
func statePatch(p domain.StatePatch) string {
	out := "{"
	for _, field := range []struct {
		name  string
		value any
	}{{"name", p.Name}, {"color", p.Color}, {"group", p.Group}, {"description", p.Description}, {"sequence", p.Sequence}} {
		switch v := field.value.(type) {
		case *string:
			if v != nil {
				out += fmt.Sprintf(" %s %q", field.name, *v)
			}
		case *domain.StateGroup:
			if v != nil {
				out += fmt.Sprintf(" %s %s", field.name, *v)
			}
		case *float64:
			if v != nil {
				out += fmt.Sprintf(" %s %v", field.name, *v)
			}
		}
	}
	return out + " }"
}

// stateRow is r as CreateState logs it: every field but its id, which the
// use case makes.
func stateRow(r app.StateRow) string {
	return fmt.Sprintf("%s/%s %q %s %s at %v default %v %q by %s at %s", r.WorkspaceID, r.ProjectID, r.State.Name, r.State.Color, r.State.Group,
		r.State.Sequence, r.State.Default, r.State.Description, r.CreatedBy, r.Now.Format(timeFormat))
}
