package app_test

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// The labels of web and ops, by id, the same in every fixture: made in
// this order, so their ids are too.
var webBug, webUI, webFeature, opsDocs = uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()

// fakeLabels is the project store's labels as the label operations' tests
// hold them, by id, beside fakeStore's projects, whose log, failures,
// reads of a row by its id and changedAs it shares. failsFor, when set, is
// the label whose read fails (errDisk), the others read as they are: how a
// test fails the read of a parent and not the label's own.
type fakeLabels struct {
	*fakeStore
	labels   map[uuid.UUID]domain.Label
	failsFor uuid.UUID
}

// newLabels is newWrites with web's labels, Bug at the top with UI under
// it and Feature at the top, in this order of sort order; and ops's Docs at
// the top. Each was made when the fakes' states were (stateSince).
func newLabels() (*writeFixture, *fakeLabels) {
	f := newWrites()
	l := &fakeLabels{fakeStore: f.store, labels: map[uuid.UUID]domain.Label{}}
	for _, label := range []domain.Label{
		{ID: webBug, ProjectID: webID, Name: "Bug", SortOrder: 65535},
		{ID: webUI, ProjectID: webID, ParentID: &webBug, Name: "UI", SortOrder: 75535},
		{ID: webFeature, ProjectID: webID, Name: "Feature", SortOrder: 85535},
		{ID: opsDocs, ProjectID: opsID, Name: "Docs", SortOrder: 65535},
	} {
		label.WorkspaceID, label.Color, label.CreatedAt, label.UpdatedAt = acme.ID, "#111111", stateSince, stateSince
		l.labels[label.ID] = label
	}
	return f, l
}

// of are the labels of project, in the store's order: by name, which
// neither a sort by sort order, either way, nor one by id gives, so that a
// use case that sorted them would answer another.
func (f *fakeLabels) of(project uuid.UUID) []domain.Label {
	var out []domain.Label
	for _, l := range f.labels {
		if l.ProjectID == project {
			out = append(out, l)
		}
	}
	slices.SortFunc(out, func(a, b domain.Label) int { return cmp.Compare(a.Name, b.Name) })
	return out
}

func (f *fakeLabels) GreatestSortOrder(ctx context.Context, projectID uuid.UUID) (*float64, error) {
	f.log.add(ctx, "GreatestSortOrder %s", projectID)
	if err := f.fail("GreatestSortOrder"); err != nil {
		return nil, err
	}
	var greatest *float64
	for _, l := range f.of(projectID) {
		if greatest == nil || l.SortOrder > *greatest {
			greatest = &l.SortOrder
		}
	}
	return greatest, nil
}

// CreateLabel stores r's label, as stored, unless its project has a label
// of its name in any case: domain.ErrLabelNameTaken.
func (f *fakeLabels) CreateLabel(ctx context.Context, r app.LabelRow) (domain.Label, error) {
	f.log.add(ctx, "CreateLabel %s", labelRow(r))
	if err := f.fail("CreateLabel"); err != nil {
		return domain.Label{}, err
	}
	if f.taken(r.ProjectID, uuid.Nil(), r.Name) {
		return domain.Label{}, domain.ErrLabelNameTaken
	}
	at := r.Now.Truncate(time.Microsecond)
	l := domain.Label{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, ParentID: r.ParentID, Name: r.Name, Color: r.Color,
		SortOrder: r.SortOrder, CreatedAt: at, UpdatedAt: at}
	f.labels[r.ID] = l
	if f.changedAs != (uuid.UUID{}) {
		l.ID = f.changedAs
	}
	return l, nil
}

// LabelByID is the label id; the reads after the first, under the locks,
// answer as f.reread says (readRow); the read of f.failsFor fails.
func (f *fakeLabels) LabelByID(ctx context.Context, id uuid.UUID) (domain.Label, bool, error) {
	stored, ok := f.labels[id]
	l, found, err := readRow(ctx, f.fakeStore, "LabelByID", id, stored, ok,
		func(l *domain.Label) (*uuid.UUID, *uuid.UUID) { return &l.ProjectID, &l.ID })
	if err == nil && f.failsFor != (uuid.UUID{}) && id == f.failsFor {
		return domain.Label{}, false, fmt.Errorf("LabelByID %s: %w", id, errDisk)
	}
	return l, found, err
}

// HasChildren reports whether a label has id as its parent.
func (f *fakeLabels) HasChildren(ctx context.Context, id uuid.UUID) (bool, error) {
	f.log.add(ctx, "HasChildren %s", id)
	if err := f.fail("HasChildren"); err != nil {
		return false, err
	}
	for _, l := range f.labels {
		if l.ParentID != nil && *l.ParentID == id {
			return true, nil
		}
	}
	return false, nil
}

// UpdateLabel changes the fields p gives of the label id, as stored,
// unless another label of its project has the name p gives, in any case:
// domain.ErrLabelNameTaken.
func (f *fakeLabels) UpdateLabel(ctx context.Context, id uuid.UUID, p domain.LabelPatch, by uuid.UUID, now time.Time) (domain.Label, error) {
	f.log.add(ctx, "UpdateLabel %s %s by %s at %s", id, labelPatch(p), by, now.Format(timeFormat))
	if err := f.fail("UpdateLabel"); err != nil {
		return domain.Label{}, err
	}
	l, ok := f.labels[id]
	if !ok {
		return domain.Label{}, fmt.Errorf("UpdateLabel: no label %s", id)
	}
	if p.Name != nil {
		if f.taken(l.ProjectID, id, *p.Name) {
			return domain.Label{}, domain.ErrLabelNameTaken
		}
		l.Name = *p.Name
	}
	if p.Color != nil {
		l.Color = *p.Color
	}
	if p.SetParent {
		l.ParentID = p.ParentID
	}
	if p.SortOrder != nil {
		l.SortOrder = *p.SortOrder
	}
	l.UpdatedAt = now.Truncate(time.Microsecond)
	f.labels[id] = l
	if f.changedAs != (uuid.UUID{}) {
		l.ID = f.changedAs
	}
	return l, nil
}

// DeleteLabel deletes the label id and the labels under it.
func (f *fakeLabels) DeleteLabel(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "DeleteLabel %s by %s at %s", id, by, now.Format(timeFormat))
	if err := f.fail("DeleteLabel"); err != nil {
		return err
	}
	for key, l := range f.labels {
		if key == id || (l.ParentID != nil && *l.ParentID == id) {
			delete(f.labels, key)
		}
	}
	return nil
}

// taken reports whether a label of project but except has name, in any
// case.
func (f *fakeLabels) taken(project, except uuid.UUID, name string) bool {
	return slices.ContainsFunc(f.of(project), func(l domain.Label) bool { return l.ID != except && strings.EqualFold(l.Name, name) })
}

// labelRow is r as CreateLabel logs it: every field but its id, which the
// use case makes.
func labelRow(r app.LabelRow) string {
	return fmt.Sprintf("%s/%s %q %q under %s at %v by %s at %s", r.WorkspaceID, r.ProjectID, r.Name, r.Color, parentOf(r.ParentID), r.SortOrder,
		r.CreatedBy, r.Now.Format(timeFormat))
}

// labelPatch is p as UpdateLabel logs it: the fields it gives, the parent
// "none" for a label moved to the top.
func labelPatch(p domain.LabelPatch) string {
	out := "{"
	if p.Name != nil {
		out += fmt.Sprintf(" name %q", *p.Name)
	}
	if p.Color != nil {
		out += fmt.Sprintf(" color %q", *p.Color)
	}
	if p.SetParent {
		out += " parent " + parentOf(p.ParentID)
	}
	if p.SortOrder != nil {
		out += fmt.Sprintf(" sort order %v", *p.SortOrder)
	}
	return out + " }"
}

// labelJSON is l as the tests compare labels: its parent by its id, not by
// the pointer.
func labelJSON(t *testing.T, l domain.Label) string {
	t.Helper()
	b, err := json.Marshal(l)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// parentOf is a parent as the fakes log it: its id, or "none".
func parentOf(id *uuid.UUID) string {
	if id == nil {
		return "none"
	}
	return id.String()
}
