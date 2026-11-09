package app

import (
	"context"
	"uuid"
)

// IssueDirectoryName is a cycle or a module as IssueDirectory answers it:
// its id and its name, which a list of work items ordered by the cycles'
// or the modules' names keys them by (M6 P3b spec R2).
type IssueDirectoryName struct {
	ID   uuid.UUID
	Name string
}

// IssueDirectoryRows are the reads of IssueDirectory, in the transaction
// ctx carries: of the projects, every undeleted cycle, or module, archived
// or not, with its name, in id order; another project's and a deleted one
// not among them, nor anything when no project is asked about.
type IssueDirectoryRows interface {
	CycleNames(ctx context.Context, projects []uuid.UUID) ([]IssueDirectoryName, error)
	ModuleNames(ctx context.Context, projects []uuid.UUID) ([]IssueDirectoryName, error)
}

// IssueDirectory is what the module tells the issue module about the
// cycles and modules of the projects a list of work items reads, when the
// list is ordered by their names (M6 P3b spec R2): their names, read in the
// list's read-only snapshot, which ctx carries, so they are those of the
// moment the list's rows are. Archived ones are among them, as their work
// items are; the projects' toggles do not matter, reading is not held to
// them (M6 design 3.8). planning.Provide hands it to bootstrap.
type IssueDirectory struct {
	rows IssueDirectoryRows
}

// NewIssueDirectory returns the issue directory over the planning rows.
func NewIssueDirectory(rows IssueDirectoryRows) *IssueDirectory {
	return &IssueDirectory{rows: rows}
}

// CycleNames reads the names of the projects' undeleted cycles, archived
// or not, in id order (IssueDirectoryRows).
func (d *IssueDirectory) CycleNames(ctx context.Context, projects []uuid.UUID) ([]IssueDirectoryName, error) {
	return d.rows.CycleNames(ctx, projects)
}

// ModuleNames reads the names of the projects' undeleted modules, archived
// or not, in id order (IssueDirectoryRows).
func (d *IssueDirectory) ModuleNames(ctx context.Context, projects []uuid.UUID) ([]IssueDirectoryName, error) {
	return d.rows.ModuleNames(ctx, projects)
}
