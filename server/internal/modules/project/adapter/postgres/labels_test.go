package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// addLabel stores a label of w's project, made by maker at earlier, with
// the id, name, parent and sort order given, and returns it as CreateLabel
// answers it.
func (w stateWorld) addLabel(t *testing.T, project, id uuid.UUID, name string, parent *uuid.UUID, sortOrder float64) domain.Label {
	t.Helper()
	got, err := w.s.CreateLabel(context.Background(), app.LabelRow{ID: id, WorkspaceID: w.workspaceOf(project), ProjectID: project, ParentID: parent,
		Name: name, Color: "#111", SortOrder: sortOrder, CreatedBy: w.maker, Now: earlier})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// labelNames are the names of labels, in order.
func labelNames(labels []domain.Label) []string {
	out := make([]string, len(labels))
	for i, l := range labels {
		out[i] = l.Name
	}
	return out
}

// CreateLabel stores the row with its values, its parent too, by the
// account and at the moment given, never deleted, and answers it as
// stored; every other label keeps every column.
func TestCreateLabel(t *testing.T) {
	w := newStateWorld(t)
	bug := w.addLabel(t, w.web, uuid.NewV7(), "Bug", nil, 65535)
	id := uuid.NewV7()
	others := tableRows(t, w.pool, "labels", id)

	got, err := w.s.CreateLabel(context.Background(), app.LabelRow{ID: id, WorkspaceID: w.acme, ProjectID: w.web, ParentID: &bug.ID, Name: "UI",
		Color: "#F59E0B", SortOrder: 75535, CreatedBy: w.alice, Now: now})

	want := domain.Label{ID: id, WorkspaceID: w.acme, ProjectID: w.web, ParentID: &bug.ID, Name: "UI", Color: "#F59E0B", SortOrder: 75535,
		CreatedAt: now, UpdatedAt: now}
	if err != nil || jsonOf(t, got) != jsonOf(t, want) {
		t.Errorf("CreateLabel() = %+v, %v; want %+v", got, err, want)
	}
	cols := columns(t, w.pool, "labels", id)
	if a := `"` + w.alice.String() + `"`; cols["created_by_id"] != a || cols["updated_by_id"] != a || cols["deleted_at"] != "null" ||
		cols["created_at"] != jsonTime(now) || cols["updated_at"] != jsonTime(now) || cols["parent_id"] != `"`+bug.ID.String()+`"` {
		t.Errorf("the stored row: %v; want it under Bug, made and written by alice at now, undeleted", cols)
	}
	if after := tableRows(t, w.pool, "labels", id); after != others {
		t.Errorf("the other labels:\n%s\nwant\n%s", after, others)
	}
}

// A name is taken by another undeleted label of the same project in any
// case: bug and BUG beside Bug are each project.label_name_taken, the
// first problem the API would answer, and store nothing; a deleted label's
// name and a name of another project's label are free.
func TestCreateLabelNameTaken(t *testing.T) {
	w := newStateWorld(t)
	w.addLabel(t, w.web, uuid.NewV7(), "Bug", nil, 1)
	old := w.addLabel(t, w.web, uuid.NewV7(), "Old", nil, 2)
	exec(t, w.pool, "UPDATE labels SET deleted_at = $2 WHERE id = $1", old.ID, earlier)
	w.addLabel(t, w.ops, uuid.NewV7(), "Feature", nil, 1)
	create := func(name string) error {
		_, err := w.s.CreateLabel(context.Background(), app.LabelRow{ID: uuid.NewV7(), WorkspaceID: w.acme, ProjectID: w.web, Name: name,
			SortOrder: 3, CreatedBy: w.alice, Now: now})
		return err
	}
	before := tableRows(t, w.pool, "labels")
	for _, taken := range []string{"bug", "BUG"} {
		var se *shared.Error
		if err := create(taken); !errors.As(err, &se) || !errors.Is(se, domain.ErrLabelNameTaken) {
			t.Errorf("%s: %v, want project.label_name_taken", taken, err)
		}
	}
	if after := tableRows(t, w.pool, "labels"); after != before {
		t.Errorf("the labels after the refusals:\n%s\nwant\n%s", after, before)
	}
	for _, free := range []string{"old", "Feature", "Bugs"} {
		if err := create(free); err != nil {
			t.Errorf("%s: %v, want it free", free, err)
		}
	}
}

// Only the name's unique key is a 409: another unique key (a duplicate
// id), a CHECK the domain should have kept (an empty name, a label its own
// parent) and a parent of none are each an internal error, never a domain
// error.
func TestCreateLabelBreakingAnotherConstraintIsInternal(t *testing.T) {
	w := newStateWorld(t)
	bug := w.addLabel(t, w.web, uuid.NewV7(), "Bug", nil, 1)
	own, none := uuid.NewV7(), uuid.NewV7()
	for _, tt := range []struct {
		name       string
		row        app.LabelRow
		constraint string
	}{
		{"a duplicate id", app.LabelRow{ID: bug.ID, Name: "Feature"}, "labels_pkey"},
		{"an empty name", app.LabelRow{ID: uuid.NewV7(), Name: ""}, "labels_name_check"},
		{"its own parent", app.LabelRow{ID: own, ParentID: &own, Name: "Feature"}, "labels_not_own_parent_check"},
		{"a parent of none", app.LabelRow{ID: uuid.NewV7(), ParentID: &none, Name: "Feature"}, "labels_parent_id_fkey"},
	} {
		tt.row.WorkspaceID, tt.row.ProjectID, tt.row.CreatedBy, tt.row.Now = w.acme, w.web, w.alice, now
		_, err := w.s.CreateLabel(context.Background(), tt.row)
		if !internalViolation(err, tt.constraint) {
			t.Errorf("%s: CreateLabel() = %v; want the violation of %s, not a domain error", tt.name, err, tt.constraint)
		}
	}
}

// LabelByID reads the undeleted label the id names, every column as
// CreateLabel answered it, its parent too: not another label, which a read
// of whichever row lies first would answer for one of the two asked about;
// not a deleted label, nor an id of none.
func TestLabelByID(t *testing.T) {
	w := newStateWorld(t)
	bug := w.addLabel(t, w.web, uuid.NewV7(), "Bug", nil, 1)
	ui := w.addLabel(t, w.web, uuid.NewV7(), "UI", &bug.ID, 2.5)
	old := w.addLabel(t, w.web, uuid.NewV7(), "Old", nil, 3)
	exec(t, w.pool, "UPDATE labels SET deleted_at = $2 WHERE id = $1", old.ID, earlier)
	for _, want := range []domain.Label{bug, ui} {
		if got, found, err := w.s.LabelByID(context.Background(), want.ID); err != nil || !found || jsonOf(t, got) != jsonOf(t, want) {
			t.Errorf("LabelByID(%s) = %+v, %v, %v; want %+v", want.Name, got, found, err, want)
		}
	}
	for name, id := range map[string]uuid.UUID{"the deleted Old": old.ID, "no label": uuid.NewV7()} {
		if got, found, err := w.s.LabelByID(context.Background(), id); err != nil || found || got.ID != (uuid.UUID{}) {
			t.Errorf("LabelByID() of %s = %+v, %v, %v; want none", name, got, found, err)
		}
	}
}

// ListLabels lists the project's undeleted labels, parents and children
// alike, by sort order, then id: Wiki, added last at UI's sort order with
// an id below its and a name after its, comes before it, which neither the
// order the rows lie in nor the names give; not the deleted Old, nor
// another project's labels. An archived project lists its labels as any
// other (M3 design 3.19).
func TestListLabels(t *testing.T) {
	w := newStateWorld(t)
	bug := w.addLabel(t, w.web, uuid.NewV7(), "Bug", nil, 3)
	w.addLabel(t, w.web, uuid.NewV7(), "UI", &bug.ID, 2)
	w.addLabel(t, w.web, w.early, "Wiki", nil, 2)
	old := w.addLabel(t, w.web, uuid.NewV7(), "Old", nil, 1)
	exec(t, w.pool, "UPDATE labels SET deleted_at = $2 WHERE id = $1", old.ID, earlier)
	w.addLabel(t, w.ops, uuid.NewV7(), "Docs", nil, 1)
	var heap []string
	if err := w.pool.QueryRow(context.Background(), "SELECT array_agg(name ORDER BY ctid) FROM labels WHERE project_id = $1 AND sort_order = 2",
		w.web).Scan(&heap); err != nil || !slices.Equal(heap, []string{"UI", "Wiki"}) {
		t.Fatalf("the rows of sort order 2 lie as %q, %v; the test needs UI first", heap, err)
	}

	got, err := w.s.ListLabels(context.Background(), w.web)
	if want := []string{"Wiki", "UI", "Bug"}; err != nil || !slices.Equal(labelNames(got), want) {
		t.Errorf("ListLabels(Web) = %q, %v; want %q", labelNames(got), err, want)
	}
	if len(got) > 1 && (got[1].ParentID == nil || *got[1].ParentID != bug.ID) {
		t.Errorf("Web's UI listed as %+v; want it under Bug", got[1])
	}
	exec(t, w.pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", w.ops, now)
	if got, err := w.s.ListLabels(context.Background(), w.ops); err != nil || !slices.Equal(labelNames(got), []string{"Docs"}) {
		t.Errorf("ListLabels(Ops, archived) = %q, %v; want Docs", labelNames(got), err)
	}
	if got, err := w.s.ListLabels(context.Background(), w.site); err != nil || got == nil || len(got) != 0 {
		t.Errorf("ListLabels(Site) = %+v, %v; want an empty list", got, err)
	}
}

// GreatestSortOrder is the greatest sort order of the project's undeleted
// labels, a child's too: Web's UI, 75535, not a deleted label's 99999, nor
// Ops's 80000. A project without labels has none.
func TestGreatestSortOrder(t *testing.T) {
	w := newStateWorld(t)
	bug := w.addLabel(t, w.web, uuid.NewV7(), "Bug", nil, 65535)
	w.addLabel(t, w.web, uuid.NewV7(), "UI", &bug.ID, 75535)
	old := w.addLabel(t, w.web, uuid.NewV7(), "Old", nil, 99999)
	exec(t, w.pool, "UPDATE labels SET deleted_at = $2 WHERE id = $1", old.ID, earlier)
	w.addLabel(t, w.ops, uuid.NewV7(), "Docs", nil, 80000)

	if got, err := w.s.GreatestSortOrder(context.Background(), w.web); err != nil || got == nil || *got != 75535 {
		t.Errorf("GreatestSortOrder(Web) = %s, %v; want 75535", jsonOf(t, got), err)
	}
	if got, err := w.s.GreatestSortOrder(context.Background(), w.site); err != nil || got != nil {
		t.Errorf("GreatestSortOrder(Site) = %s, %v; want none", jsonOf(t, got), err)
	}
}

// HasChildren is whether an undeleted label is under the label: Bug has
// UI; UI has none; Feature's only child is deleted; an id of none has
// none.
func TestHasChildren(t *testing.T) {
	w := newStateWorld(t)
	bug := w.addLabel(t, w.web, uuid.NewV7(), "Bug", nil, 1)
	ui := w.addLabel(t, w.web, uuid.NewV7(), "UI", &bug.ID, 2)
	feature := w.addLabel(t, w.web, uuid.NewV7(), "Feature", nil, 3)
	gone := w.addLabel(t, w.web, uuid.NewV7(), "Gone", &feature.ID, 4)
	exec(t, w.pool, "UPDATE labels SET deleted_at = $2 WHERE id = $1", gone.ID, earlier)
	for _, tt := range []struct {
		name string
		id   uuid.UUID
		want bool
	}{{"Bug", bug.ID, true}, {"UI", ui.ID, false}, {"Feature", feature.ID, false}, {"no label", uuid.NewV7(), false}} {
		if got, err := w.s.HasChildren(context.Background(), tt.id); err != nil || got != tt.want {
			t.Errorf("HasChildren(%s) = %v, %v; want %v", tt.name, got, err, tt.want)
		}
	}
}
