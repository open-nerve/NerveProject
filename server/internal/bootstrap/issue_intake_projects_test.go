package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
)

// IntakeIssueProjects makes fakeIssueProjects project's IssueProjects: it
// records the question and answers no project, or its error.
func (f *fakeIssueProjects) IntakeIssueProjects(_ context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	f.asked = append(f.asked, fmt.Sprint("intake ", ids))
	return nil, f.err
}

// intakeProjectsFake answers project's IntakeIssueProjects with on and
// records what it was asked and in which context; every other question as
// fakeIssueProjects.
type intakeProjectsFake struct {
	*fakeIssueProjects
	on  []uuid.UUID
	ids []uuid.UUID
	ctx context.Context
}

func (f *intakeProjectsFake) IntakeIssueProjects(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	f.ctx, f.ids = ctx, slices.Clone(ids)
	return f.on, f.err
}

// issueProjects hands project the projects issue asks which of have their
// intake turned on (M7 P2b spec 2.6), in the same context, and issue
// project's answer as it came, an error with none.
func TestIntakeProjectsConversion(t *testing.T) {
	web, ops, quiet := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	f := &intakeProjectsFake{fakeIssueProjects: &fakeIssueProjects{}, on: []uuid.UUID{ops, web}}
	var p issue.ProjectPorts = issueProjects{projects: f}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	asked := []uuid.UUID{web, quiet, ops}
	got, err := p.IntakeOn(ctx, asked)
	if err != nil || !slices.Equal(got, []uuid.UUID{ops, web}) || f.ctx != ctx || !slices.Equal(f.ids, asked) {
		t.Fatalf("IntakeOn() = %v, %v; project asked %v in %v", got, err, f.ids, f.ctx)
	}
	f.err = errors.New("intake toggles unavailable")
	if got, err := p.IntakeOn(ctx, asked); !errors.Is(err, f.err) || got != nil {
		t.Errorf("IntakeOn() = %v, %v; want no projects and %v", got, err, f.err)
	}
}
