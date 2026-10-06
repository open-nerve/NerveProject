package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateLabel changes exactly the fields the patch gives of Web's UI, and
// the audit columns to the moment and the account given, and answers the
// label as stored, made at earlier and written at that moment; every other
// column keeps its value, and every other label every column. Each change
// is by another account than the change before it, at a later moment, so
// that the audit columns it writes are its own. A parent given sets it, a
// parent given as none clears it, and a parent not given keeps it. A patch
// that gives nothing changes the audit columns alone: it comes after one
// that gave every field, so that the parent, the color and the sort order
// it keeps are neither none nor the columns' defaults.
func TestUpdateLabel(t *testing.T) {
	w := newStateWorld(t)
	bob := newAccount(t, w.pool, "bob@corp.com")
	bug := w.addLabel(t, w.web, uuid.NewV7(), "Bug", nil, 1)
	ui := w.addLabel(t, w.web, uuid.NewV7(), "UI", nil, 2).ID
	others := tableRows(t, w.pool, "labels", ui)
	for i, tt := range []struct {
		name   string
		patch  domain.LabelPatch
		change map[string]string
	}{
		{"every field", domain.LabelPatch{Name: ptr("Front"), Color: ptr("#123456"), SetParent: true, ParentID: &bug.ID, SortOrder: ptr(-2.5)},
			map[string]string{"name": `"Front"`, "color": `"#123456"`, "parent_id": `"` + bug.ID.String() + `"`, "sort_order": "-2.5"}},
		{"nothing", domain.LabelPatch{}, nil},
		{"the name alone, its parent kept", domain.LabelPatch{Name: ptr("UI")}, map[string]string{"name": `"UI"`}},
		{"its parent cleared", domain.LabelPatch{SetParent: true}, map[string]string{"parent_id": "null"}},
		{"the sort order alone", domain.LabelPatch{SortOrder: ptr(5.0)}, map[string]string{"sort_order": "5"}},
		{"an empty color", domain.LabelPatch{Color: ptr("")}, map[string]string{"color": `""`}},
	} {
		by, at := []uuid.UUID{w.alice, bob}[i%2], now.Add(time.Duration(i)*time.Minute)
		before := columns(t, w.pool, "labels", ui)
		got, err := w.s.UpdateLabel(context.Background(), ui, tt.patch, by, at)
		want := changed(before, changed(audit(by, at), tt.change))
		if after := columns(t, w.pool, "labels", ui); err != nil || !maps.Equal(after, want) {
			t.Errorf("%s: UpdateLabel() = %v, the row %v\nwant %v", tt.name, err, after, want)
		}
		if stored, _, _ := w.s.LabelByID(context.Background(), ui); jsonOf(t, got) != jsonOf(t, stored) || !got.CreatedAt.Equal(earlier) ||
			!got.UpdatedAt.Equal(at) {
			t.Errorf("%s: UpdateLabel() answered %+v; want the row as stored, %+v, made at %v and written at %v", tt.name, got, stored, earlier, at)
		}
	}
	if after := tableRows(t, w.pool, "labels", ui); after != others {
		t.Errorf("the other labels:\n%s\nwant\n%s", after, others)
	}
}

// A name another undeleted label of the project has, in any case, is
// project.label_name_taken, the first problem the API would answer, and
// changes nothing; a deleted label's name and another project's are free,
// as is the label's own in another case.
func TestUpdateLabelNameTaken(t *testing.T) {
	w := newStateWorld(t)
	ui := w.addLabel(t, w.web, uuid.NewV7(), "UI", nil, 1).ID
	w.addLabel(t, w.web, uuid.NewV7(), "Feature", nil, 2)
	old := w.addLabel(t, w.web, uuid.NewV7(), "Old", nil, 3)
	exec(t, w.pool, "UPDATE labels SET deleted_at = $2 WHERE id = $1", old.ID, earlier)
	w.addLabel(t, w.ops, uuid.NewV7(), "Docs", nil, 1)
	before := tableRows(t, w.pool, "labels")
	var se *shared.Error
	if _, err := w.s.UpdateLabel(context.Background(), ui, domain.LabelPatch{Name: ptr("FEATURE")}, w.alice, now); !errors.As(err, &se) ||
		!errors.Is(se, domain.ErrLabelNameTaken) {
		t.Errorf("FEATURE: %v, want project.label_name_taken", err)
	}
	if after := tableRows(t, w.pool, "labels"); after != before {
		t.Errorf("the labels after the refusal:\n%s\nwant\n%s", after, before)
	}
	for _, free := range []string{"ui", "Old", "Docs"} {
		if _, err := w.s.UpdateLabel(context.Background(), ui, domain.LabelPatch{Name: ptr(free)}, w.alice, now); err != nil {
			t.Errorf("%s: %v, want it free", free, err)
		}
	}
}

// Only the name's unique key is a 409 for a change too: a CHECK the domain
// should have kept (an empty name, a label its own parent) and a parent of
// none are each an internal error, never a domain error.
func TestUpdateLabelBreakingAnotherConstraintIsInternal(t *testing.T) {
	w := newStateWorld(t)
	ui := w.addLabel(t, w.web, uuid.NewV7(), "UI", nil, 1).ID
	none := uuid.NewV7()
	for _, tt := range []struct {
		name       string
		patch      domain.LabelPatch
		constraint string
	}{
		{"an empty name", domain.LabelPatch{Name: ptr("")}, "labels_name_check"},
		{"its own parent", domain.LabelPatch{SetParent: true, ParentID: &ui}, "labels_not_own_parent_check"},
		{"a parent of none", domain.LabelPatch{SetParent: true, ParentID: &none}, "labels_parent_id_fkey"},
	} {
		_, err := w.s.UpdateLabel(context.Background(), ui, tt.patch, w.alice, now)
		var se *shared.Error
		var pgErr *pgconn.PgError
		if errors.As(err, &se) || !errors.As(err, &pgErr) || pgErr.ConstraintName != tt.constraint {
			t.Errorf("%s: UpdateLabel() = %v; want the violation of %s, not a domain error", tt.name, err, tt.constraint)
		}
	}
}

// UpdateLabel writes no deleted label: an internal error, no domain error
// (project.label_name_taken or any other), and every label keeps every
// column.
func TestUpdateLabelWritesNoDeletedLabel(t *testing.T) {
	w := newStateWorld(t)
	old := w.addLabel(t, w.web, uuid.NewV7(), "Old", nil, 1)
	exec(t, w.pool, "UPDATE labels SET deleted_at = $2 WHERE id = $1", old.ID, earlier)
	before := tableRows(t, w.pool, "labels")
	var se *shared.Error
	if got, err := w.s.UpdateLabel(context.Background(), old.ID, domain.LabelPatch{Name: ptr("Other")}, w.alice, now); err == nil ||
		errors.As(err, &se) || got.ID != (uuid.UUID{}) {
		t.Errorf("UpdateLabel() of the deleted Old = %+v, %v; want an internal error", got, err)
	}
	if after := tableRows(t, w.pool, "labels"); after != before {
		t.Errorf("the labels after the refusal:\n%s\nwant\n%s", after, before)
	}
}

// DeleteLabel deletes Web's Bug and UI under it, at the moment and by the
// account given, deleted_at and updated_at alike, and touches no other
// column: Gone, under Bug and deleted before, keeps its moment; Feature,
// Feature's Docs, and Ops's Bug keep every column. Deleted again, at a
// later moment, Bug changes nothing. Deleting a child, Docs, deletes it
// alone.
func TestDeleteLabel(t *testing.T) {
	w := newStateWorld(t)
	bug := w.addLabel(t, w.web, uuid.NewV7(), "Bug", nil, 1)
	ui := w.addLabel(t, w.web, uuid.NewV7(), "UI", &bug.ID, 2)
	gone := w.addLabel(t, w.web, uuid.NewV7(), "Gone", &bug.ID, 3)
	exec(t, w.pool, "UPDATE labels SET deleted_at = $2 WHERE id = $1", gone.ID, earlier)
	feature := w.addLabel(t, w.web, uuid.NewV7(), "Feature", nil, 4)
	docs := w.addLabel(t, w.web, uuid.NewV7(), "Docs", &feature.ID, 5)
	w.addLabel(t, w.ops, uuid.NewV7(), "Bug", nil, 1)
	others := tableRows(t, w.pool, "labels", bug.ID, ui.ID)
	before := map[uuid.UUID]map[string]string{bug.ID: columns(t, w.pool, "labels", bug.ID), ui.ID: columns(t, w.pool, "labels", ui.ID)}

	if err := w.s.DeleteLabel(context.Background(), bug.ID, w.alice, now); err != nil {
		t.Errorf("DeleteLabel(Bug) = %v", err)
	}
	for id, cols := range before {
		if got, want := columns(t, w.pool, "labels", id), changed(cols, changed(audit(w.alice, now),
			map[string]string{"deleted_at": jsonTime(now)})); !maps.Equal(got, want) {
			t.Errorf("%s: %v\nwant %v", id, got, want)
		}
	}
	if after := tableRows(t, w.pool, "labels", bug.ID, ui.ID); after != others {
		t.Errorf("the other labels:\n%s\nwant\n%s", after, others)
	}
	all := tableRows(t, w.pool, "labels")
	if err := w.s.DeleteLabel(context.Background(), bug.ID, w.alice, now.Add(time.Minute)); err != nil {
		t.Errorf("DeleteLabel(Bug) again = %v", err)
	}
	if after := tableRows(t, w.pool, "labels"); after != all {
		t.Errorf("the labels after Bug deleted again:\n%s\nwant\n%s", after, all)
	}
	all = tableRows(t, w.pool, "labels", docs.ID)
	if err := w.s.DeleteLabel(context.Background(), docs.ID, w.alice, now); err != nil {
		t.Errorf("DeleteLabel(Docs) = %v", err)
	}
	if after := tableRows(t, w.pool, "labels", docs.ID); after != all || columns(t, w.pool, "labels", docs.ID)["deleted_at"] != jsonTime(now) {
		t.Errorf("the labels after Docs deleted:\n%s\nwant Docs deleted and the others\n%s", after, all)
	}
}
