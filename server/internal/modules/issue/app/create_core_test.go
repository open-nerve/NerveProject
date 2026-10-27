package app

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Unused ports panic through the embedded interface, so the core cannot
// silently acquire authorization, a gate or another transaction here.
type coreRows struct {
	IssueCreator
	row         IssueRow
	subscribers LinkRows
	activities  []ActivityRow
	version     domain.DescriptionVersion
	event       shared.Event
	clockReads  int
	at          time.Time
}

func (r *coreRows) InsertIssue(_ context.Context, row IssueRow) error { r.row = row; return nil }
func (r *coreRows) GreatestSortOrder(context.Context, uuid.UUID, uuid.UUID) (*float64, error) {
	return nil, nil
}
func (r *coreRows) InsertSubscribers(_ context.Context, rows LinkRows) error {
	r.subscribers = rows
	return nil
}
func (r *coreRows) InsertActivities(_ context.Context, rows []ActivityRow) error {
	r.activities = rows
	return nil
}
func (r *coreRows) InsertDescriptionVersion(_ context.Context, v domain.DescriptionVersion) error {
	r.version = v
	return nil
}
func (r *coreRows) GetIssue(_ context.Context, id uuid.UUID) (domain.Issue, bool, error) {
	return domain.Issue{ID: id, Name: "stored projection"}, true, nil
}
func (r *coreRows) States(context.Context, uuid.UUID) ([]StateRef, error) {
	return []StateRef{{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000c1"), Name: "Done", Default: true, Group: domain.GroupCompleted}}, nil
}
func (r *coreRows) Next(context.Context, uuid.UUID) (int, error) { return 7, nil }
func (r *coreRows) Now() time.Time {
	r.clockReads++
	return r.at.Add(time.Duration(r.clockReads-1) * time.Hour)
}
func (r *coreRows) Clean(string) domain.RichText {
	return domain.RichText{HTML: "<p>Hi</p>", Text: "Hi"}
}
func (r *coreRows) Publish(_ context.Context, e shared.Event) error { r.event = e; return nil }

// A second clock reading or returning a separate At would split the draft's
// eventual deletion time from the actual issue, child rows, version and event.
func TestCreateCoreReturnsItsWriteMoment(t *testing.T) {
	r := &coreRows{at: time.Date(2044, 5, 6, 7, 8, 9, 123456000, time.UTC)}
	core := NewIssueCreation(IssueCreationDeps{Issues: r, States: r, Sequences: r, Clock: r, Sanitizer: r, Events: r})
	in, err := domain.CheckNewIssue(domain.NewIssue{Name: "Hi"})
	if err != nil {
		t.Fatal(err)
	}
	actor, project, workspace := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	got, at, err := core.createLocked(context.Background(), actor, project, held{project: LockedProject{WorkspaceID: workspace}}, in)
	if err != nil {
		t.Fatal(err)
	}
	if r.clockReads != 1 || !at.Equal(r.at) || !r.row.At.Equal(at) || r.row.CompletedAt == nil || !r.row.CompletedAt.Equal(at) || !r.subscribers.At.Equal(at) || !r.version.LastSavedAt.Equal(at) || !r.version.CreatedAt.Equal(at) || !r.version.UpdatedAt.Equal(at) || !r.event.OccurredAt.Equal(at) {
		t.Fatalf("clock=%d at=%v row=%+v version=%+v event=%+v", r.clockReads, at, r.row, r.version, r.event)
	}
	if len(r.activities) != 1 || !r.activities[0].At.Equal(at) {
		t.Fatalf("activities=%+v", r.activities)
	}
	if got.ID != r.row.ID || got.Name != "stored projection" || r.row.By != actor || r.subscribers.By != actor || r.version.OwnedByID != actor || r.event.ActorID == nil || *r.event.ActorID != actor {
		t.Fatalf("result=%+v rows=%+v", got, r)
	}
}
