package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// earlier is when the fixtures' states were made: no write of the tests
// is at that moment.
var earlier = now.Add(-time.Hour)

// stateWorld is acme's Web and Ops and beta's Site, each with the six
// default states, made by maker at earlier: no write of the tests is by
// maker. alice writes; TestUpdateLabel writes as bob too, every other
// change, so that each change's audit columns are its own. The label
// tests add their labels the same way (addLabel).
type stateWorld struct {
	s              *postgresadapter.Store
	pool           *pgxpool.Pool
	alice, maker   uuid.UUID
	acme, beta     uuid.UUID
	web, ops, site uuid.UUID
	states         map[uuid.UUID]map[string]uuid.UUID // by project, by name
	// early is an id made before Web's states, for a state added after them.
	early uuid.UUID
}

func newStateWorld(t *testing.T) stateWorld {
	t.Helper()
	s, pool := newStore(t)
	w := stateWorld{s: s, pool: pool, alice: newAccount(t, pool, "alice@corp.com"), maker: newAccount(t, pool, "maker@corp.com"),
		acme: newWorkspace(t, pool, "acme"), beta: newWorkspace(t, pool, "beta"), states: map[uuid.UUID]map[string]uuid.UUID{}}
	w.early = uuid.NewV7()
	w.web, w.ops = newProject(t, s, w.acme, "Web", "WEB", w.maker), newProject(t, s, w.acme, "Ops", "OPS", w.maker)
	w.site = newProject(t, s, w.beta, "Site", "SITE", w.maker)
	for _, p := range []struct{ workspace, project uuid.UUID }{{w.acme, w.web}, {w.acme, w.ops}, {w.beta, w.site}} {
		w.states[p.project] = seedDefaultStates(t, s, p.workspace, p.project, w.maker)
	}
	return w
}

// seedDefaultStates stores project's six default states, made by by at
// earlier, and returns their ids by name.
func seedDefaultStates(t *testing.T, s *postgresadapter.Store, workspace, project, by uuid.UUID) map[string]uuid.UUID {
	t.Helper()
	ids := map[string]uuid.UUID{}
	var rows []app.StateRow
	for _, st := range domain.DefaultStates() {
		ids[st.Name] = uuid.NewV7()
		rows = append(rows, app.StateRow{ID: ids[st.Name], WorkspaceID: workspace, ProjectID: project, State: st, CreatedBy: by, Now: earlier})
	}
	if err := s.CreateStates(context.Background(), rows); err != nil {
		t.Fatal(err)
	}
	return ids
}

// workspaceOf is the workspace of w's project: beta's for Site, acme's
// for the others.
func (w stateWorld) workspaceOf(project uuid.UUID) uuid.UUID {
	if project == w.site {
		return w.beta
	}
	return w.acme
}

// addState stores a state of w's project, made by maker at earlier, with
// the id given, and returns it as CreateState answers it.
func (w stateWorld) addState(t *testing.T, project, id uuid.UUID, st domain.NewState) domain.State {
	t.Helper()
	got, err := w.s.CreateState(context.Background(), app.StateRow{ID: id, WorkspaceID: w.workspaceOf(project), ProjectID: project, State: st,
		CreatedBy: w.maker, Now: earlier})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// names are the names of states, in order.
func names(states []domain.State) []string {
	out := make([]string, len(states))
	for i, s := range states {
		out[i] = s.Name
	}
	return out
}

// CreateState stores the row with its values, its description too, by the
// account and at the moment given, never deleted, and answers it as
// stored; every other state keeps every column.
func TestCreateState(t *testing.T) {
	w := newStateWorld(t)
	id := uuid.NewV7()
	others := tableRows(t, w.pool, "states", id)

	got, err := w.s.CreateState(context.Background(), app.StateRow{ID: id, WorkspaceID: w.acme, ProjectID: w.web, CreatedBy: w.alice, Now: now,
		State: domain.NewState{Name: "Review", Description: "Waiting for a review", Color: "#F59E0B", Sequence: 70000, Group: domain.GroupStarted}})

	want := domain.State{ID: id, WorkspaceID: w.acme, ProjectID: w.web, Name: "Review", Description: "Waiting for a review", Color: "#F59E0B",
		Group: domain.GroupStarted, Sequence: 70000, CreatedAt: now, UpdatedAt: now}
	if err != nil || got != want {
		t.Errorf("CreateState() = %+v, %v; want %+v", got, err, want)
	}
	cols := columns(t, w.pool, "states", id)
	if a := `"` + w.alice.String() + `"`; cols["created_by_id"] != a || cols["updated_by_id"] != a || cols["deleted_at"] != "null" ||
		cols["created_at"] != jsonTime(now) || cols["updated_at"] != jsonTime(now) {
		t.Errorf("the stored row: %v; want it made and written by alice at now, undeleted", cols)
	}
	if after := tableRows(t, w.pool, "states", id); after != others {
		t.Errorf("the other states:\n%s\nwant\n%s", after, others)
	}
}

// A name is taken by another undeleted state of the same project only,
// written as it is: In Progress again in Web, and Triage, its triage
// state's, are each project.state_name_taken, the first problem the API
// would answer, and store nothing; in progress, in another case, is free,
// as are a deleted state's name and a name of another project's state.
func TestCreateStateNameTaken(t *testing.T) {
	w := newStateWorld(t)
	exec(t, w.pool, "UPDATE states SET deleted_at = $2 WHERE id = $1", w.states[w.web]["Todo"], earlier)
	w.addState(t, w.ops, uuid.NewV7(), domain.NewState{Name: "Review", Color: "#000", Sequence: 1, Group: domain.GroupStarted})
	create := func(name string) error {
		_, err := w.s.CreateState(context.Background(), app.StateRow{ID: uuid.NewV7(), WorkspaceID: w.acme, ProjectID: w.web, CreatedBy: w.alice,
			Now: now, State: domain.NewState{Name: name, Color: "#000", Sequence: 1, Group: domain.GroupStarted}})
		return err
	}
	before := tableRows(t, w.pool, "states")
	for _, taken := range []string{"In Progress", "Triage"} {
		var se *shared.Error
		if err := create(taken); !errors.As(err, &se) || !errors.Is(se, domain.ErrStateNameTaken) {
			t.Errorf("%s again: %v, want project.state_name_taken", taken, err)
		}
	}
	if after := tableRows(t, w.pool, "states"); after != before {
		t.Errorf("the states after the refusal:\n%s\nwant\n%s", after, before)
	}
	for _, free := range []string{"in progress", "Todo", "Review"} {
		if err := create(free); err != nil {
			t.Errorf("%s: %v, want it free", free, err)
		}
	}
}

// Only the name's unique key is a 409: another unique key (a duplicate id,
// a second default state) and a CHECK the domain should have kept (an
// unknown group) are each an internal error, never a domain error.
func TestCreateStateBreakingAnotherConstraintIsInternal(t *testing.T) {
	w := newStateWorld(t)
	for _, tt := range []struct {
		name       string
		id         uuid.UUID
		state      domain.NewState
		constraint string
	}{
		{"a duplicate id", w.states[w.web]["Todo"], domain.NewState{Name: "Review", Color: "#000", Group: domain.GroupStarted}, "states_pkey"},
		{"a second default", uuid.NewV7(), domain.NewState{Name: "Review", Color: "#000", Group: domain.GroupStarted, Default: true},
			"states_project_id_default_key"},
		{"an unknown group", uuid.NewV7(), domain.NewState{Name: "Review", Color: "#000", Group: "review"}, "states_group_check"},
	} {
		_, err := w.s.CreateState(context.Background(), app.StateRow{ID: tt.id, WorkspaceID: w.acme, ProjectID: w.web, State: tt.state,
			CreatedBy: w.alice, Now: now})
		if !internalViolation(err, tt.constraint) {
			t.Errorf("%s: CreateState() = %v; want the violation of %s, not a domain error", tt.name, err, tt.constraint)
		}
	}
}

// StateByID reads the undeleted state the id names, every column as
// CreateState answered it: not another state, which a read of whichever
// row lies first would answer for one of the two asked about; not the
// triage state, nor a deleted state, nor an id of none.
func TestStateByID(t *testing.T) {
	w := newStateWorld(t)
	review := w.addState(t, w.web, uuid.NewV7(), domain.NewState{Name: "Review", Description: "d", Color: "#111", Sequence: 70000.5,
		Group: domain.GroupStarted})
	exec(t, w.pool, "UPDATE states SET deleted_at = $2 WHERE id = $1", w.states[w.web]["Todo"], earlier)
	backlog := domain.State{ID: w.states[w.ops]["Backlog"], WorkspaceID: w.acme, ProjectID: w.ops, Name: "Backlog", Color: "#60646C",
		Group: domain.GroupBacklog, Default: true, Sequence: 15000, CreatedAt: earlier, UpdatedAt: earlier}
	for _, want := range []domain.State{review, backlog} {
		if got, found, err := w.s.StateByID(context.Background(), want.ID); err != nil || !found || got != want {
			t.Errorf("StateByID(%s) = %+v, %v, %v; want %+v", want.Name, got, found, err, want)
		}
	}
	for name, id := range map[string]uuid.UUID{"Web's triage state": w.states[w.web]["Triage"], "Web's deleted Todo": w.states[w.web]["Todo"],
		"no state": uuid.NewV7()} {
		if got, found, err := w.s.StateByID(context.Background(), id); err != nil || found || got != (domain.State{}) {
			t.Errorf("StateByID() of %s = %+v, %v, %v; want none", name, got, found, err)
		}
	}
}

// ListStates lists the project's undeleted states but its triage state, by
// sequence, then id: Review, added last at In Progress's sequence with an
// id below its, comes before it and before Done, which the order the rows
// lie in does not give; not Web's deleted Todo, nor another project's
// states. An archived project has none: an empty list.
func TestListStates(t *testing.T) {
	w := newStateWorld(t)
	w.addState(t, w.web, w.early, domain.NewState{Name: "Review", Color: "#111", Sequence: 35000, Group: domain.GroupStarted})
	exec(t, w.pool, "UPDATE states SET deleted_at = $2 WHERE id = $1", w.states[w.web]["Todo"], earlier)
	var heap []string
	if err := w.pool.QueryRow(context.Background(), "SELECT array_agg(name ORDER BY ctid) FROM states WHERE project_id = $1 AND sequence = 35000",
		w.web).Scan(&heap); err != nil || !slices.Equal(heap, []string{"In Progress", "Review"}) {
		t.Fatalf("the rows of sequence 35000 lie as %q, %v; the test needs In Progress first", heap, err)
	}

	got, err := w.s.ListStates(context.Background(), w.web)
	if want := []string{"Backlog", "Review", "In Progress", "Done", "Cancelled"}; err != nil || !slices.Equal(names(got), want) {
		t.Errorf("ListStates(Web) = %q, %v; want %q", names(got), err, want)
	}
	if len(got) > 0 && got[0] != (domain.State{ID: w.states[w.web]["Backlog"], WorkspaceID: w.acme, ProjectID: w.web, Name: "Backlog",
		Color: "#60646C", Group: domain.GroupBacklog, Default: true, Sequence: 15000, CreatedAt: earlier, UpdatedAt: earlier}) {
		t.Errorf("Web's Backlog listed as %+v", got[0])
	}
	exec(t, w.pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", w.ops, now)
	if got, err := w.s.ListStates(context.Background(), w.ops); err != nil || got == nil || len(got) != 0 {
		t.Errorf("ListStates(Ops, archived) = %+v, %v; want an empty list", got, err)
	}
}

// GreatestSequence is the greatest sequence of the project's undeleted
// states but its triage state: Web's Cancelled, 55000, neither the first
// nor the last of Web's states made, so that a read of whichever row was
// made first or last misses it; not its Triage, 65000, nor a deleted
// state's 99999, nor Ops's 80000. A project without states, and one with
// its triage state alone, have none.
func TestGreatestSequence(t *testing.T) {
	w := newStateWorld(t)
	w.addState(t, w.web, uuid.NewV7(), domain.NewState{Name: "Gone", Color: "#111", Sequence: 99999, Group: domain.GroupStarted})
	exec(t, w.pool, "UPDATE states SET deleted_at = $1 WHERE name = 'Gone'", earlier)
	w.addState(t, w.web, uuid.NewV7(), domain.NewState{Name: "Early", Color: "#111", Sequence: 5000, Group: domain.GroupStarted})
	w.addState(t, w.ops, uuid.NewV7(), domain.NewState{Name: "Late", Color: "#111", Sequence: 80000, Group: domain.GroupStarted})
	empty := newProject(t, w.s, w.acme, "Empty", "EMPTY", w.maker)
	triage := newProject(t, w.s, w.acme, "Intake", "INTAKE", w.maker)
	w.addState(t, triage, uuid.NewV7(), domain.NewState{Name: "Triage", Color: "#111", Sequence: 65000, Group: domain.GroupTriage})

	if got, err := w.s.GreatestSequence(context.Background(), w.web); err != nil || got == nil || *got != 55000 {
		t.Errorf("GreatestSequence(Web) = %s, %v; want 55000", jsonOf(t, got), err)
	}
	for name, project := range map[string]uuid.UUID{"a project without states": empty, "a project with its triage state alone": triage} {
		if got, err := w.s.GreatestSequence(context.Background(), project); err != nil || got != nil {
			t.Errorf("GreatestSequence() of %s = %s, %v; want none", name, jsonOf(t, got), err)
		}
	}
}

// CountGroupStates counts the project's undeleted states of the group:
// Web's started group holds In Progress and Review, not Web's deleted
// started state, nor Ops's In Progress, nor a state of another group.
func TestCountGroupStates(t *testing.T) {
	w := newStateWorld(t)
	w.addState(t, w.web, uuid.NewV7(), domain.NewState{Name: "Review", Color: "#111", Sequence: 1, Group: domain.GroupStarted})
	w.addState(t, w.web, uuid.NewV7(), domain.NewState{Name: "Gone", Color: "#111", Sequence: 2, Group: domain.GroupStarted})
	exec(t, w.pool, "UPDATE states SET deleted_at = $1 WHERE name = 'Gone'", earlier)
	for _, tt := range []struct {
		project uuid.UUID
		group   domain.StateGroup
		want    int
	}{{w.web, domain.GroupStarted, 2}, {w.web, domain.GroupBacklog, 1}, {w.ops, domain.GroupStarted, 1}} {
		if got, err := w.s.CountGroupStates(context.Background(), tt.project, tt.group); err != nil || got != tt.want {
			t.Errorf("CountGroupStates(%s, %s) = %d, %v; want %d", tt.project, tt.group, got, err, tt.want)
		}
	}
}

// ListWorkspaceStates lists the states but the triage states of the
// workspace's undeleted, unarchived projects alice is an active member of,
// by project id, then sequence, then id: Web's and Ops's, Review, added last
// at In Progress's sequence with an id below its, before In Progress in
// Web, which the order the rows lie in does not give; not the deleted
// Todo; not the states of Docs, where her membership ended, of Arch,
// archived, of Gone, deleted though its rows were left as no deletion
// leaves them, of Lab, where her only membership is deleted, nor of Team,
// where bob is a member and she is not; nor of beta's Site. bob, whose
// only project is Team, lists Team's. Every membership was made and last
// written by maker, who is no member: a query reading created_by_id or
// updated_by_id where it means member_id lists nothing.
func TestListWorkspaceStates(t *testing.T) {
	w := newStateWorld(t)
	bob := newAccount(t, w.pool, "bob@corp.com")
	w.addState(t, w.web, w.early, domain.NewState{Name: "Review", Color: "#111", Sequence: 35000, Group: domain.GroupStarted})
	exec(t, w.pool, "UPDATE states SET deleted_at = $2 WHERE id = $1", w.states[w.web]["Todo"], earlier)
	var heap []string
	if err := w.pool.QueryRow(context.Background(), "SELECT array_agg(name ORDER BY ctid) FROM states WHERE project_id = $1 AND sequence = 35000",
		w.web).Scan(&heap); err != nil || !slices.Equal(heap, []string{"In Progress", "Review"}) {
		t.Fatalf("the rows of sequence 35000 lie as %q, %v; the test needs In Progress first", heap, err)
	}
	more := map[string]uuid.UUID{}
	for _, name := range []string{"Docs", "Arch", "Gone", "Lab", "Team"} {
		more[name] = newProject(t, w.s, w.acme, name, strings.ToUpper(name), w.maker)
		w.states[more[name]] = seedDefaultStates(t, w.s, w.acme, more[name], w.maker)
	}
	for _, p := range []uuid.UUID{w.web, w.ops, more["Arch"], more["Gone"]} {
		seedMember(t, w.pool, w.acme, p, w.alice, 15, true)
	}
	seedMember(t, w.pool, w.beta, w.site, w.alice, 15, true)
	seedMember(t, w.pool, w.acme, more["Docs"], w.alice, 20, false)
	seedDeleted(t, w.pool, w.acme, more["Lab"], w.alice, 20)
	seedMember(t, w.pool, w.acme, more["Team"], bob, 15, true)
	exec(t, w.pool, "UPDATE project_members SET created_by_id = $1, updated_by_id = $1", w.maker)
	exec(t, w.pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", more["Arch"], now)
	exec(t, w.pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", more["Gone"], now)

	got, err := w.s.ListWorkspaceStates(context.Background(), w.acme, w.alice)
	want := []string{"Backlog", "Review", "In Progress", "Done", "Cancelled", "Backlog", "Todo", "In Progress", "Done", "Cancelled"}
	if err != nil || !slices.Equal(names(got), want) || got[0].ProjectID != w.web || got[5].ProjectID != w.ops {
		t.Errorf("ListWorkspaceStates(acme, alice) = %+v, %v; want Web's then Ops's, %q", got, err, want)
	}
	got, err = w.s.ListWorkspaceStates(context.Background(), w.acme, bob)
	if want := []string{"Backlog", "Todo", "In Progress", "Done", "Cancelled"}; err != nil || !slices.Equal(names(got), want) ||
		got[0].ProjectID != more["Team"] || got[0] != (domain.State{ID: w.states[more["Team"]]["Backlog"], WorkspaceID: w.acme,
		ProjectID: more["Team"], Name: "Backlog", Color: "#60646C", Group: domain.GroupBacklog, Default: true, Sequence: 15000,
		CreatedAt: earlier, UpdatedAt: earlier}) {
		t.Errorf("ListWorkspaceStates(acme, bob) = %+v, %v; want Team's, %q", got, err, want)
	}
	if got, err := w.s.ListWorkspaceStates(context.Background(), w.beta, bob); err != nil || got == nil || len(got) != 0 {
		t.Errorf("ListWorkspaceStates(beta, bob) = %+v, %v; want an empty list", got, err)
	}
}
