package postgresadapter

import (
	"context"
	"fmt"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// WriteDescription persists sanitized description columns without auditing.
func (s *Store) WriteDescription(ctx context.Context, w app.DescriptionWrite) error {
	if err := s.queries(ctx).WriteDescription(ctx, gen.WriteDescriptionParams{IssueID: w.IssueID, Html: w.HTML, Text: w.Text}); err != nil {
		return fmt.Errorf("write description of work item %s: %w", w.IssueID, err)
	}
	return nil
}
