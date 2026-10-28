package app

import (
	"context"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// RetentionProjects includes every undeleted project, even archived projects.
type RetentionProjects interface {
	UndeletedProjectIDs(context.Context) ([]uuid.UUID, error)
}

// DescriptionVersionTrimmer removes older live versions within the project lock.
type DescriptionVersionTrimmer interface {
	TrimDescriptionVersions(context.Context, uuid.UUID) (int, error)
}

// TrimVersionsDeps supplies no clock or event publisher: retention only deletes.
type TrimVersionsDeps struct {
	Projects RetentionProjects
	Gate     ProjectGate
	Rows     DescriptionVersionTrimmer
	Tx       shared.TxManager
}

// TrimVersions retains the twenty latest live descriptions per issue.
type TrimVersions struct{ d TrimVersionsDeps }

// NewTrimVersions constructs the per-project retention use case.
func NewTrimVersions(d TrimVersionsDeps) *TrimVersions { return &TrimVersions{d: d} }

// Execute returns only deletions whose project transaction committed.
func (u *TrimVersions) Execute(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	projects, err := u.d.Projects.UndeletedProjectIDs(ctx)
	if err != nil {
		return 0, err
	}
	slices.SortFunc(projects, uuid.UUID.Compare)
	total := 0
	for _, id := range projects {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		count := 0
		err := u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
			_, found, err := u.d.Gate.Lock(ctx, id)
			if err != nil {
				return err
			}
			if !found {
				return nil
			}
			count, err = u.d.Rows.TrimDescriptionVersions(ctx, id)
			return err
		})
		if err != nil {
			return total, err
		}
		total += count
	}
	return total, nil
}
