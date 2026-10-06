package domain

import (
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Label is a label of a project as stored and as the API shows it (M3
// design 3.16, 5.2): a project's alone, of two levels at most, its parent
// nil at the top.
type Label struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	ParentID    *uuid.UUID
	Name        string
	Color       string
	SortOrder   float64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Place is the label's id, its project and the project's workspace: where
// a write on it takes its locks.
func (l Label) Place() (id, workspaceID, projectID uuid.UUID) {
	return l.ID, l.WorkspaceID, l.ProjectID
}

// LabelCreate is what the caller asks for when creating a label (M3 design
// 5.1): its parent nil for a label at the top. Its place is the project's
// to give (SortOrderAfter; M3 design 4.10).
type LabelCreate struct {
	Name     string
	Color    string
	ParentID *uuid.UUID
}

// LabelPatch is what updateLabel changes (M3 design 5.1): each field nil
// when it is not given, and kept; SetParent says whether the parent is
// given, ParentID nil then for none.
type LabelPatch struct {
	Name      *string
	Color     *string
	SetParent bool
	ParentID  *uuid.UUID
	SortOrder *float64
}

// maxLabelText is the length of labels.name and labels.color,
// varchar(255), in characters.
const maxLabelText = 255

// sortOrderStep is the gap between a project's last label and a new one,
// and firstSortOrder the sort order of a project's first label: Plane's
// Label.save() and the column's default (M3 design 4.10).
const (
	sortOrderStep  = 10000
	firstSortOrder = 65535
)

// CheckNewLabel checks l (M3 design 3.16): a name of 1–255 characters, not
// blank, without NUL; a color of at most 255 characters, empty for none,
// without NUL. Every field with a problem is reported at once, in one 422
// validation_failed. Whether the name is taken, in any case, is the
// database's to say; whether the parent may be, the use case's, under the
// project's lock (CheckParent).
func CheckNewLabel(l LabelCreate) error {
	return invalid(checkRequiredText("name", l.Name, maxLabelText), checkLabelColor(l.Color))
}

// CheckLabelPatch checks the fields p gives by CheckNewLabel's rules; a
// sort order is any number. Every field with a problem is reported at
// once.
func CheckLabelPatch(p LabelPatch) error {
	var found []*shared.FieldError
	if p.Name != nil {
		found = append(found, checkRequiredText("name", *p.Name, maxLabelText))
	}
	if p.Color != nil {
		found = append(found, checkLabelColor(*p.Color))
	}
	return invalid(found...)
}

// SortOrderAfter is the sort order of a new label of a project whose labels
// have greatest as their greatest sort order, nil when it has none: one
// step after it, or the column's default (Plane's Label.save(); M3 design
// 4.10).
func SortOrderAfter(greatest *float64) float64 {
	if greatest == nil {
		return firstSortOrder
	}
	return *greatest + sortOrderStep
}

// CheckParent refuses parent as the parent of the label id of project,
// uuid.Nil for a new label, which has children when hasChildren (M3 design
// 3.16): the parent of a label is an undeleted label of its project, not
// itself, that has no parent of its own; a label with children has none.
// parent is nil for a parent that is not there or is deleted. Each refusal
// is one 422 validation_failed, parent_id not_allowed.
func CheckParent(id, project uuid.UUID, parent *Label, hasChildren bool) error {
	var why string
	switch {
	case parent == nil || parent.ProjectID != project:
		why = "must be a label of the project"
	case parent.ID == id:
		why = "must not be the label itself"
	case parent.ParentID != nil:
		why = "must be a label without a parent: labels have two levels"
	case hasChildren:
		why = "must be null: the label has labels under it, and labels have two levels"
	default:
		return nil
	}
	return shared.Invalid(shared.FieldError{Field: "parent_id", Code: shared.FieldNotAllowed, Message: why})
}

// checkLabelColor refuses a color longer than labels.color or with NUL; it
// may be empty, for none.
func checkLabelColor(c string) *shared.FieldError {
	if f := checkMaxLength("color", c, maxLabelText); f != nil {
		return f
	}
	return checkText("color", c)
}
