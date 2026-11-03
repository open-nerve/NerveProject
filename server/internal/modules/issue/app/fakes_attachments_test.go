package app_test

import (
	"context"
	"strings"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// Old issue-only unit fixtures have no assets. Production and bootstrap tests
// always use the actual asset reader; count-specific unit tests inject values.
type noIssueAttachments struct{}

func (noIssueAttachments) CountIssueAttachments(_ context.Context, ids []uuid.UUID) ([]app.AttachmentCount, error) {
	rows := make([]app.AttachmentCount, len(ids))
	for n, id := range ids {
		rows[n].IssueID = id
	}
	return rows, nil
}

// The three reads now enter their snapshot before the first port call. Keep
// the original refusal-order lists and assert that every call carries its ctx.
func readCalls(calls []string) []string {
	if len(calls) == 0 {
		return nil
	}
	out := []string{"Begin"}
	for _, call := range calls {
		out = append(out, strings.TrimSuffix(call, " outside tx"))
	}
	return out
}
