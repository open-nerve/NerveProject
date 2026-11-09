package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"uuid"
)

// directoryRows is IssueDirectory's rows as a fake that logs its calls and answers what it holds: the store keeps to
// the projects it is asked about, whatever their toggles, which the directory does not read.
type directoryRows struct {
	calls           []string
	cycles, modules []IssueDirectoryName
	fail            error
}

func (r *directoryRows) CycleNames(_ context.Context, projects []uuid.UUID) ([]IssueDirectoryName, error) {
	r.calls = append(r.calls, fmt.Sprint("CycleNames ", projects))
	return r.cycles, r.fail
}

func (r *directoryRows) ModuleNames(_ context.Context, projects []uuid.UUID) ([]IssueDirectoryName, error) {
	r.calls = append(r.calls, fmt.Sprint("ModuleNames ", projects))
	return r.modules, r.fail
}

// The directory answers, of the projects asked about, the store's cycles' names and its modules' names, each as read
// and each of its own read, asked of those projects alone; a read's failure is the answer.
func TestIssueDirectoryAnswersTheProjectsNames(t *testing.T) {
	ctx := context.Background()
	web, ops := uuid.NewV7(), uuid.NewV7()
	rows := &directoryRows{
		cycles:  []IssueDirectoryName{{ID: uuid.NewV7(), Name: "Sprint 1"}, {ID: uuid.NewV7(), Name: "Sprint 2"}},
		modules: []IssueDirectoryName{{ID: uuid.NewV7(), Name: "Core"}},
	}
	d := NewIssueDirectory(rows)
	if got, err := d.CycleNames(ctx, []uuid.UUID{web, ops}); err != nil || !slices.Equal(got, rows.cycles) {
		t.Errorf("CycleNames() = %+v, %v; want %+v", got, err, rows.cycles)
	}
	if got, err := d.ModuleNames(ctx, []uuid.UUID{ops}); err != nil || !slices.Equal(got, rows.modules) {
		t.Errorf("ModuleNames() = %+v, %v; want %+v", got, err, rows.modules)
	}
	if want := []string{fmt.Sprint("CycleNames ", []uuid.UUID{web, ops}), fmt.Sprint("ModuleNames ", []uuid.UUID{ops})}; !slices.Equal(rows.calls, want) {
		t.Errorf("calls %q, want %q", rows.calls, want)
	}

	boom := errors.New("boom")
	rows.fail = boom
	if _, err := d.CycleNames(ctx, []uuid.UUID{web}); !errors.Is(err, boom) {
		t.Errorf("CycleNames() = %v; want the read's failure", err)
	}
	if _, err := d.ModuleNames(ctx, []uuid.UUID{web}); !errors.Is(err, boom) {
		t.Errorf("ModuleNames() = %v; want the read's failure", err)
	}
}
