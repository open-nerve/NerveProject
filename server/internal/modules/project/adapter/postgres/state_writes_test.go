package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateState changes exactly the fields the patch gives of Web's default
// state, Backlog, and the audit columns to the moment and the account
// given, and answers the state as stored; every other column keeps its
// value, the default flag too, which the column's default, false, would
// not show, and every other state every column. A patch that gives
// nothing changes the audit columns alone: it comes after one that gave
// every field, so that the group and the sequence it keeps are neither
// backlog nor 0.
func TestUpdateState(t *testing.T) {
	w := newStateWorld(t)
	backlog := w.states[w.web]["Backlog"]
	if cols := columns(t, w.pool, "states", backlog); cols["default"] != "true" {
		t.Fatalf("Web's Backlog: %v; the test needs it the default", cols)
	}
	others := tableRows(t, w.pool, "states", backlog)
	for _, tt := range []struct {
		name   string
		patch  domain.StatePatch
		change map[string]string
	}{
		{"every field", domain.StatePatch{Name: ptr("Next"), Description: ptr("Up next"), Color: ptr("#123456"), Group: ptr(domain.GroupStarted),
			Sequence: ptr(-2.5)}, map[string]string{"name": `"Next"`, "description": `"Up next"`, "color": `"#123456"`, "group": `"started"`,
			"sequence": "-2.5"}},
		{"nothing", domain.StatePatch{}, nil},
		{"the name alone", domain.StatePatch{Name: ptr("Later")}, map[string]string{"name": `"Later"`}},
		{"the group alone", domain.StatePatch{Group: ptr(domain.GroupCancelled)}, map[string]string{"group": `"cancelled"`}},
		{"the sequence alone", domain.StatePatch{Sequence: ptr(25000.0)}, map[string]string{"sequence": "25000"}},
	} {
		before := columns(t, w.pool, "states", backlog)
		got, err := w.s.UpdateState(context.Background(), backlog, tt.patch, w.alice, now)
		want := changed(before, changed(audit(w.alice, now), tt.change))
		if after := columns(t, w.pool, "states", backlog); err != nil || !maps.Equal(after, want) {
			t.Errorf("%s: UpdateState() = %v, the row %v\nwant %v", tt.name, err, after, want)
		}
		if stored, _, _ := w.s.StateByID(context.Background(), backlog); got != stored {
			t.Errorf("%s: UpdateState() answered %+v; want the row as stored, %+v", tt.name, got, stored)
		}
	}
	if after := tableRows(t, w.pool, "states", backlog); after != others {
		t.Errorf("the other states:\n%s\nwant\n%s", after, others)
	}
}

// A name another undeleted state of the project has, its triage state's
// too, is project.state_name_taken, the first problem the API would
// answer, and changes nothing; the same name in another case, a deleted
// state's and another project's are free.
func TestUpdateStateNameTaken(t *testing.T) {
	w := newStateWorld(t)
	todo := w.states[w.web]["Todo"]
	exec(t, w.pool, "UPDATE states SET deleted_at = $2 WHERE id = $1", w.states[w.web]["Cancelled"], earlier)
	w.addState(t, w.ops, uuid.NewV7(), domain.NewState{Name: "Review", Color: "#000", Sequence: 1, Group: domain.GroupStarted})
	before := tableRows(t, w.pool, "states")
	for _, taken := range []string{"Done", "Triage"} {
		var se *shared.Error
		if _, err := w.s.UpdateState(context.Background(), todo, domain.StatePatch{Name: ptr(taken)}, w.alice, now); !errors.As(err, &se) ||
			!errors.Is(se, domain.ErrStateNameTaken) {
			t.Errorf("%s: %v, want project.state_name_taken", taken, err)
		}
	}
	if after := tableRows(t, w.pool, "states"); after != before {
		t.Errorf("the states after the refusal:\n%s\nwant\n%s", after, before)
	}
	for _, free := range []string{"done", "Cancelled", "Review"} {
		if _, err := w.s.UpdateState(context.Background(), todo, domain.StatePatch{Name: ptr(free)}, w.alice, now); err != nil {
			t.Errorf("%s: %v, want it free", free, err)
		}
	}
}

// UpdateState writes neither a deleted state nor the triage state: each is
// an internal error, no domain error (project.state_name_taken or any
// other), and every state keeps every column.
func TestUpdateStateWritesNoDeletedOrTriageState(t *testing.T) {
	w := newStateWorld(t)
	exec(t, w.pool, "UPDATE states SET deleted_at = $2 WHERE id = $1", w.states[w.web]["Todo"], earlier)
	before := tableRows(t, w.pool, "states")
	for name, id := range map[string]uuid.UUID{"the deleted Todo": w.states[w.web]["Todo"], "the triage state": w.states[w.web]["Triage"]} {
		var se *shared.Error
		if got, err := w.s.UpdateState(context.Background(), id, domain.StatePatch{Name: ptr("Other")}, w.alice, now); err == nil ||
			errors.As(err, &se) || got != (domain.State{}) {
			t.Errorf("UpdateState() of %s = %+v, %v; want an internal error", name, got, err)
		}
	}
	if after := tableRows(t, w.pool, "states"); after != before {
		t.Errorf("the states after the refusals:\n%s\nwant\n%s", after, before)
	}
}

// DeleteState deletes Web's Todo at the moment and by the account given,
// deleted_at and updated_at alike, and touches no other column, nor any
// other state. It deletes neither the default state, nor the triage state,
// nor a state deleted before, which keeps its moment, nor an id of none:
// each answers false and writes nothing.
func TestDeleteState(t *testing.T) {
	w := newStateWorld(t)
	todo := w.states[w.web]["Todo"]
	exec(t, w.pool, "UPDATE states SET deleted_at = $2 WHERE id = $1", w.states[w.web]["Done"], earlier)
	others := tableRows(t, w.pool, "states", todo)
	before := columns(t, w.pool, "states", todo)
	if deleted, err := w.s.DeleteState(context.Background(), todo, w.alice, now); err != nil || !deleted {
		t.Errorf("DeleteState(Todo) = %v, %v; want true", deleted, err)
	}
	if got, want := columns(t, w.pool, "states", todo), changed(before, changed(audit(w.alice, now),
		map[string]string{"deleted_at": jsonTime(now)})); !maps.Equal(got, want) {
		t.Errorf("Todo: %v\nwant %v", got, want)
	}
	if after := tableRows(t, w.pool, "states", todo); after != others {
		t.Errorf("the other states:\n%s\nwant\n%s", after, others)
	}
	all := tableRows(t, w.pool, "states")
	for name, id := range map[string]uuid.UUID{"the default Backlog": w.states[w.web]["Backlog"], "the triage state": w.states[w.web]["Triage"],
		"Done, deleted before": w.states[w.web]["Done"], "no state": uuid.NewV7()} {
		if deleted, err := w.s.DeleteState(context.Background(), id, w.alice, now.Add(1)); err != nil || deleted {
			t.Errorf("DeleteState() of %s = %v, %v; want false", name, deleted, err)
		}
	}
	if after := tableRows(t, w.pool, "states"); after != all {
		t.Errorf("the states after the refusals:\n%s\nwant\n%s", after, all)
	}
}

// MarkDefaultState makes Web's Todo its default and Backlog the default no
// longer, each stamped with the moment and the account given; no other
// state changes: Ops's default stays, as does a deleted state of Web that
// was its default. Then Backlog takes it back, and Backlog made the
// default while it is the default stays the only one.
func TestMarkDefaultState(t *testing.T) {
	w := newStateWorld(t)
	todo, backlog := w.states[w.web]["Todo"], w.states[w.web]["Backlog"]
	old := w.addState(t, w.web, uuid.NewV7(), domain.NewState{Name: "Old", Color: "#000", Sequence: 1, Group: domain.GroupStarted})
	exec(t, w.pool, `UPDATE states SET deleted_at = $2, "default" = true WHERE id = $1`, old.ID, earlier)
	others := tableRows(t, w.pool, "states", todo, backlog)
	before := map[uuid.UUID]map[string]string{todo: columns(t, w.pool, "states", todo), backlog: columns(t, w.pool, "states", backlog)}

	if marked, err := w.s.MarkDefaultState(context.Background(), w.web, todo, w.alice, now); err != nil || !marked {
		t.Errorf("MarkDefaultState(Todo) = %v, %v; want true", marked, err)
	}
	for id, isDefault := range map[uuid.UUID]string{todo: "true", backlog: "false"} {
		if got, want := columns(t, w.pool, "states", id), changed(before[id], changed(audit(w.alice, now),
			map[string]string{"default": isDefault})); !maps.Equal(got, want) {
			t.Errorf("%s: %v\nwant %v", id, got, want)
		}
	}
	if after := tableRows(t, w.pool, "states", todo, backlog); after != others {
		t.Errorf("the other states:\n%s\nwant\n%s", after, others)
	}
	for range 2 {
		if marked, err := w.s.MarkDefaultState(context.Background(), w.web, backlog, w.alice, now); err != nil || !marked {
			t.Errorf("MarkDefaultState(Backlog) = %v, %v; want true", marked, err)
		}
		var defaults []uuid.UUID
		if err := w.pool.QueryRow(context.Background(), `SELECT array_agg(id) FROM states WHERE project_id = $1 AND "default"
			AND deleted_at IS NULL`, w.web).Scan(&defaults); err != nil || len(defaults) != 1 || defaults[0] != backlog {
			t.Errorf("Web's defaults after marking Backlog: %v, %v; want Backlog alone", defaults, err)
		}
	}
}

// MarkDefaultState makes no state the default that is deleted, the triage
// state, of another project than the one given, or none: each answers
// false, after its first statement. In the caller's transaction the
// project given has no default then, its Backlog the default no longer and
// no state made it, and the other project keeps its own; the caller rolls
// back, and the states are as they were: both statements ran in its
// transaction.
func TestMarkDefaultStateRefuses(t *testing.T) {
	w := newStateWorld(t)
	exec(t, w.pool, "UPDATE states SET deleted_at = $2 WHERE id = $1", w.states[w.web]["Todo"], earlier)
	before := tableRows(t, w.pool, "states")
	rollBack := errors.New("roll back")
	for _, tt := range []struct {
		name        string
		project, id uuid.UUID
		kept        uuid.UUID // the other project's default
	}{
		{"the deleted Todo", w.web, w.states[w.web]["Todo"], w.states[w.ops]["Backlog"]},
		{"the triage state", w.web, w.states[w.web]["Triage"], w.states[w.ops]["Backlog"]},
		{"Ops's Done, for Web", w.web, w.states[w.ops]["Done"], w.states[w.ops]["Backlog"]},
		{"Web's Done, for Ops", w.ops, w.states[w.web]["Done"], w.states[w.web]["Backlog"]},
		{"no state", w.web, uuid.NewV7(), w.states[w.ops]["Backlog"]},
	} {
		var marked bool
		var defaults []uuid.UUID
		err := postgres.NewTxManager(w.pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
			var err error
			if marked, err = w.s.MarkDefaultState(ctx, tt.project, tt.id, w.alice, now); err != nil {
				return err
			}
			if err := postgres.DB(ctx, w.pool).QueryRow(ctx, `SELECT coalesce(array_agg(id), '{}') FROM states WHERE "default"
				AND deleted_at IS NULL AND project_id = ANY($1)`, []uuid.UUID{w.web, w.ops}).Scan(&defaults); err != nil {
				return err
			}
			return rollBack
		})
		if !errors.Is(err, rollBack) || marked || !slices.Equal(defaults, []uuid.UUID{tt.kept}) {
			t.Errorf("MarkDefaultState() of %s = %v, %v, the defaults of Web and Ops then %v; want false, %v alone", tt.name, marked, err,
				defaults, tt.kept)
		}
		if after := tableRows(t, w.pool, "states"); after != before {
			t.Errorf("the states after %s:\n%s\nwant\n%s", tt.name, after, before)
		}
	}
}

// MarkDefaultState answers the failure of its second statement as itself,
// never "not marked", which markDefaultState would answer as
// project.state_not_found: its first statement makes Web's Backlog the
// default no longer, and its second waits for Todo, which another
// transaction holds FOR UPDATE, until the caller's lock_timeout ends it. A
// cancelled context (TestAFailedWriteIsAnError) fails the first.
func TestMarkDefaultStateAnswersTheFailureOfItsSecondStatement(t *testing.T) {
	w := newStateWorld(t)
	todo := w.states[w.web]["Todo"]
	other, err := w.pool.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = other.Rollback(context.Background()) }()
	if _, err := other.Exec(pgtest.Soon(t), "SELECT 1 FROM states WHERE id = $1 FOR UPDATE", todo); err != nil {
		t.Fatal(err)
	}
	var marked bool
	err = postgres.NewTxManager(w.pool, 2*time.Second).WithinTx(pgtest.Soon(t), func(ctx context.Context) error {
		if _, err := postgres.DB(ctx, w.pool).Exec(ctx, "SET LOCAL lock_timeout = '200ms'"); err != nil {
			return err
		}
		var err error
		marked, err = w.s.MarkDefaultState(ctx, w.web, todo, w.alice, now)
		return err
	})
	var pgErr *pgconn.PgError
	if marked || !errors.As(err, &pgErr) || pgErr.Code != "55P03" {
		t.Errorf("MarkDefaultState(Todo) with Todo held = %v, %v; want lock_not_available (55P03)", marked, err)
	}
}
