package postgresadapter

import (
	"context"
	"fmt"
	"strings"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// Search consumes the authorized directory without opening another transaction.
// Escaping is literal SQL LIKE escaping, independent of trigram ranking.
func (s *Store) Search(ctx context.Context, in app.SearchRead) ([]domain.Issue, error) {
	q := in.Query
	p := gen.SearchIssuesParams{
		Query: q.Text, ActorID: in.ActorID, ProjectIds: make([]uuid.UUID, 0, len(in.Projects)), ProjectOwn: make([]bool, 0, len(in.Projects)),
		SourceID: q.IssueID, Picker: string(q.For), ExactProjectID: in.ExactProjectID,
		IdentifierNumber: searchNumber(q.IdentifierNumber), SequenceNumber: searchNumber(q.SequenceNumber), PageSize: int32(q.Limit),
		Pattern: "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(q.Text) + "%",
	}
	for _, project := range in.Projects {
		p.ProjectIds = append(p.ProjectIds, project.ID)
		p.ProjectOwn = append(p.ProjectOwn, project.OwnOnly)
	}
	p.StartNone, p.StartFrom, p.StartTo = dateParameters(q.StartDate)
	p.TargetNone, p.TargetFrom, p.TargetTo = dateParameters(q.TargetDate)
	rows, err := s.queries(ctx).SearchIssues(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("search work items: %w", err)
	}
	out := make([]domain.Issue, len(rows))
	for n, r := range rows {
		// Similarity is a native float32 SQL result, used only for ranking. Map
		// the remaining full projection through the shared Issue conversion.
		out[n] = issueOf(gen.GetIssueRow{
			ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, SequenceID: r.SequenceID, Name: r.Name,
			DescriptionHtml: r.DescriptionHtml, DescriptionStripped: r.DescriptionStripped, StateID: r.StateID, Priority: r.Priority,
			StartDate: r.StartDate, TargetDate: r.TargetDate, ParentID: r.ParentID, SortOrder: r.SortOrder, CompletedAt: r.CompletedAt, ArchivedAt: r.ArchivedAt,
			CreatedByID: r.CreatedByID, UpdatedByID: r.UpdatedByID, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, AssigneeIds: r.AssigneeIds, LabelIds: r.LabelIds, LinkCount: r.LinkCount,
		})
	}
	return out, nil
}
func searchNumber(n *int) *int32 {
	if n == nil {
		return nil
	}
	out := int32(*n)
	return &out
}
