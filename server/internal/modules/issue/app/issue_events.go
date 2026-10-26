package app

import (
	"strconv"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// event keeps event identity separate from the entity and reuses the business
// write's actor and clock reading; UUID generation never supplies OccurredAt.
func (s stamp) event(kind string, changes []shared.Change) shared.Event {
	return shared.Event{ID: uuid.NewV7(), Type: kind, OccurredAt: s.at, WorkspaceID: s.workspace, ProjectID: s.project, ActorID: &s.by,
		Entity: shared.EntityRef{Kind: "issue", ID: s.issue}, Changes: changes}
}

// createdChanges is the complete initial state, including fields that have no
// activity rows. Names are snapshots already read under the project lock.
func createdChanges(row IssueRow, c checked, accounts []domain.Assignee) []shared.Change {
	changes := []shared.Change{
		{Field: "name", New: eventPtr(row.Name)},
		{Field: "description", New: eventPtr(row.DescriptionHTML)},
		{Field: "state", New: eventPtr(c.state.Name), NewID: &c.state.ID},
		{Field: "priority", New: eventPtr(string(row.Priority))},
		{Field: "start_date", New: eventDate(row.StartDate)},
		{Field: "target_date", New: eventDate(row.TargetDate)},
		{Field: "parent"},
		{Field: "sort_order", New: eventPtr(eventFloat(row.SortOrder))},
		{Field: "sequence_id", New: eventPtr(strconv.Itoa(row.SequenceID))},
		{Field: "completed_at"},
	}
	if c.parentName != nil {
		changes[6].New, changes[6].NewID = &c.parentName.Identifier, &c.parentName.ID
	}
	if row.CompletedAt != nil {
		changes[9].New = eventPtr(row.CompletedAt.UTC().Format(time.RFC3339Nano))
	}
	for _, a := range accounts {
		changes = append(changes, shared.Change{Field: "assignees", New: eventPtr(a.DisplayName), NewID: eventPtr(a.ID)})
	}
	for _, l := range c.labelNames {
		changes = append(changes, shared.Change{Field: "labels", New: eventPtr(l.Name), NewID: eventPtr(l.ID)})
	}
	return changes
}

// updatedChanges projects this write's changes, independent of persisted
// activity/version coalescing. Sorting has no activity and is appended last.
func updatedChanges(activity domain.Update, sortOrder *domain.Changed[float64]) []shared.Change {
	var changes []shared.Change
	for _, a := range domain.UpdateActivity(activity) {
		changes = append(changes, shared.Change{Field: *a.Field, Old: a.OldValue, New: a.NewValue, OldID: a.OldIdentifier, NewID: a.NewIdentifier})
	}
	if sortOrder != nil {
		changes = append(changes, shared.Change{Field: "sort_order", Old: eventPtr(eventFloat(sortOrder.Old)), New: eventPtr(eventFloat(sortOrder.New))})
	}
	return changes
}
func eventPtr[T any](v T) *T { return &v }
func eventDate(d *shared.Date) *string {
	if d == nil {
		return nil
	}
	return eventPtr(d.String())
}
func eventFloat(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }
