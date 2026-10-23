package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"testing"
	"uuid"
)

// The conversion of a list's read across projects (M4 design 3.8): which
// projects are archived, as bootstrap/ports.go hands it to issue.

// ArchivedIssueProjects answers fakeIssueProjects' web archived, of
// whatever projects it is asked.
func (f *fakeIssueProjects) ArchivedIssueProjects(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
	f.asked = append(f.asked, fmt.Sprint("archived ", ids))
	if f.err != nil {
		return nil, f.err
	}
	return map[uuid.UUID]bool{f.web: true}, nil
}

// issueProjects asks project for the projects issue asks about and hands
// over its answer as it came; an error as it came, without projects.
func TestIssueProjectsConvertsArchivedProjects(t *testing.T) {
	web, ops := uuid.NewV7(), uuid.NewV7()
	fake := &fakeIssueProjects{web: web}
	p := issueProjects{projects: fake}
	got, err := p.Archived(context.Background(), []uuid.UUID{web, ops})
	if want := map[uuid.UUID]bool{web: true}; err != nil || !maps.Equal(got, want) {
		t.Errorf("Archived() = %v, %v; want %v", got, err, want)
	}
	if want := []string{fmt.Sprint("archived ", []uuid.UUID{web, ops})}; !slices.Equal(fake.asked, want) {
		t.Errorf("project was asked %q, want %q", fake.asked, want)
	}
	failure := errors.New("connection reset")
	fake.err = failure
	if got, err := p.Archived(context.Background(), []uuid.UUID{web}); !errors.Is(err, failure) || got != nil {
		t.Errorf("Archived() = %v, %v; want no projects and %v", got, err, failure)
	}
}
