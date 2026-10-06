package domain

import (
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Place answers the label's own id, workspace and project, each in its
// place.
func TestLabelPlace(t *testing.T) {
	l := Label{ID: uuid.NewV7(), WorkspaceID: uuid.NewV7(), ProjectID: uuid.NewV7()}
	if id, workspace, project := l.Place(); id != l.ID || workspace != l.WorkspaceID || project != l.ProjectID {
		t.Errorf("Place() = %s, %s, %s; want %s, %s, %s", id, workspace, project, l.ID, l.WorkspaceID, l.ProjectID)
	}
}

// CheckNewLabel accepts a name of one and of 255 characters, a color empty
// or of 255 characters, and leaves the parent to others.
func TestCheckNewLabelAcceptsValidLabels(t *testing.T) {
	parent := uuid.NewV7()
	for _, l := range []LabelCreate{
		{Name: "Bug", Color: "#FF0000"},
		{Name: "B"},
		{Name: strings.Repeat("标", 255), Color: strings.Repeat("c", 255), ParentID: &parent},
	} {
		if err := CheckNewLabel(l); err != nil {
			t.Errorf("CheckNewLabel(%+v) = %v, want nil", l, err)
		}
	}
}

// CheckNewLabel and CheckLabelPatch report every field with a problem at
// once: a name empty, blank, of 256 characters or with NUL; a color of 256
// characters or with NUL, which may be empty. A field with two problems
// reports its first: too long before NUL. CheckLabelPatch checks only the
// fields given: an empty patch, a sort order, a parent or none pass.
func TestCheckLabelReportsEveryField(t *testing.T) {
	for _, tt := range []struct {
		name string
		l    LabelCreate
		want []shared.FieldError
	}{
		{"empty name", LabelCreate{Name: ""}, []shared.FieldError{nameEmpty}},
		{"blank name", LabelCreate{Name: " \n"}, []shared.FieldError{nameEmpty}},
		{"name of 256 characters", LabelCreate{Name: strings.Repeat("标", 256)}, []shared.FieldError{nameLong}},
		{"name with NUL", LabelCreate{Name: "B\x00ug"}, []shared.FieldError{nameNUL}},
		{"name of 256 characters with NUL", LabelCreate{Name: strings.Repeat("标", 255) + "\x00"}, []shared.FieldError{nameLong}},
		{"color of 256 characters", LabelCreate{Name: "Bug", Color: strings.Repeat("c", 256)}, []shared.FieldError{colorLong}},
		{"color with NUL", LabelCreate{Name: "Bug", Color: "#\x00"}, []shared.FieldError{colorNUL}},
		{"color of 256 characters with NUL", LabelCreate{Name: "Bug", Color: strings.Repeat("c", 255) + "\x00"},
			[]shared.FieldError{colorLong}},
		{"all at once", LabelCreate{Name: "", Color: "\x00"}, []shared.FieldError{nameEmpty, colorNUL}},
	} {
		if got := fieldsOf(t, CheckNewLabel(tt.l)); !slices.Equal(got, tt.want) {
			t.Errorf("%s: CheckNewLabel() fields %+v, want %+v", tt.name, got, tt.want)
		}
		p := LabelPatch{Name: &tt.l.Name, Color: &tt.l.Color}
		if got := fieldsOf(t, CheckLabelPatch(p)); !slices.Equal(got, tt.want) {
			t.Errorf("%s: CheckLabelPatch() fields %+v, want %+v", tt.name, got, tt.want)
		}
	}
	parent := uuid.NewV7()
	for _, p := range []LabelPatch{{}, {SortOrder: ptr(-1.0)}, {SetParent: true}, {SetParent: true, ParentID: &parent},
		{Name: ptr("Bug"), Color: ptr("")}} {
		if err := CheckLabelPatch(p); err != nil {
			t.Errorf("CheckLabelPatch(%+v) = %v, want nil", p, err)
		}
	}
}

// A new label comes 10000 after the greatest sort order of its project's
// labels, and at the column's default, 65535, when the project has none.
func TestSortOrderAfter(t *testing.T) {
	for _, tt := range []struct {
		greatest *float64
		want     float64
	}{{nil, 65535}, {ptr(65535.0), 75535}, {ptr(-1.5), 9998.5}, {ptr(0.0), 10000}} {
		if got := SortOrderAfter(tt.greatest); got != tt.want {
			t.Errorf("SortOrderAfter(%v) = %v, want %v", tt.greatest, got, tt.want)
		}
	}
}

// CheckParent takes a label of the project at the top as the parent of a
// new label, or of a label without children, and refuses, each as one 422
// parent_id not_allowed: no parent, a label of another project, the label
// itself, a label that has a parent, and any parent of a label with
// children (M3 design 3.16).
func TestCheckParent(t *testing.T) {
	web, ops, bug, ui := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	top := &Label{ID: bug, ProjectID: web}
	child := &Label{ID: ui, ProjectID: web, ParentID: &bug}
	refused := func(why string) []shared.FieldError {
		return []shared.FieldError{{Field: "parent_id", Code: "not_allowed", Message: why}}
	}
	for _, tt := range []struct {
		name        string
		id          uuid.UUID
		parent      *Label
		hasChildren bool
		want        []shared.FieldError
	}{
		{"a new label under a label at the top", uuid.Nil(), top, false, nil},
		{"a label without children under a label at the top", uuid.NewV7(), top, false, nil},
		{"no parent", uuid.Nil(), nil, false, refused("must be a label of the project")},
		{"a label of another project", uuid.Nil(), &Label{ID: uuid.NewV7(), ProjectID: ops}, false, refused("must be a label of the project")},
		{"the label itself", bug, top, false, refused("must not be the label itself")},
		{"a label that has a parent", uuid.Nil(), child, false, refused("must be a label without a parent: labels have two levels")},
		{"a label with children", uuid.NewV7(), top, true,
			refused("must be null: the label has labels under it, and labels have two levels")},
	} {
		if got := fieldsOf(t, CheckParent(tt.id, web, tt.parent, tt.hasChildren)); !slices.Equal(got, tt.want) {
			t.Errorf("%s: CheckParent() fields %+v, want %+v", tt.name, got, tt.want)
		}
	}
}
