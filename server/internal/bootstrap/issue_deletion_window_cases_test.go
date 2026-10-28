package bootstrap

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
)

// Each entry owns one real operation; variants never stand in for a missing entry.
type deletionWindowCase struct {
	op, method, pattern, body string
}

func deletionWindowCases() []deletionWindowCase {
	return []deletionWindowCase{
		{"searchIssues", "GET", "/api/v0/workspaces/{slug}/issue-search", ""},
		{"listDrafts", "GET", "/api/v0/workspaces/{slug}/drafts", ""},
		{"getDraft", "GET", "/api/v0/drafts/{draft_id}", ""},
		{"listMemberIssues", "GET", "/api/v0/workspaces/{slug}/members/{user_id}/issues", ""},
		{"listIssues", "GET", "/api/v0/projects/{project_id}/issues", ""},
		{"getIssue", "GET", "/api/v0/issues/{issue_id}", ""},
		{"listSubIssues", "GET", "/api/v0/issues/{issue_id}/sub-issues", ""},
		{"listIssueRelations", "GET", "/api/v0/issues/{issue_id}/relations", ""},
		{"getIssueByIdentifier", "GET", "/api/v0/workspaces/{slug}/issues/{identifier}", ""},
		{"listIssueReactions", "GET", "/api/v0/issues/{issue_id}/reactions", ""},
		{"listIssueLinks", "GET", "/api/v0/issues/{issue_id}/links", ""},
		{"listIssueComments", "GET", "/api/v0/issues/{issue_id}/comments", ""},
		{"listIssueActivities", "GET", "/api/v0/issues/{issue_id}/activities", ""},
		{"listDescriptionVersions", "GET", "/api/v0/issues/{issue_id}/description-versions", ""},
		{"getDescriptionVersion", "GET", "/api/v0/issue-description-versions/{version_id}", ""},
		{"createDraft", "POST", "/api/v0/workspaces/{slug}/drafts", `{"project_id":"{project_id}","payload":{"name":"Saved"}}`},
		{"updateDraft", "PATCH", "/api/v0/drafts/{draft_id}", `{"payload":{"name":"Changed"}}`},
		{"deleteDraft", "DELETE", "/api/v0/drafts/{draft_id}", ""},
		{"publishDraft", "POST", "/api/v0/drafts/{draft_id}/publish", ""},
		{"createIssue", "POST", "/api/v0/projects/{project_id}/issues", `{"name":"Created"}`},
		{"updateIssue", "PATCH", "/api/v0/issues/{issue_id}", `{"name":"Changed"}`},
		{"deleteIssue", "DELETE", "/api/v0/issues/{issue_id}", ""},
		{"archiveIssue", "POST", "/api/v0/issues/{issue_id}/archive", ""},
		{"unarchiveIssue", "POST", "/api/v0/issues/{issue_id}/unarchive", ""},
		{"addSubIssues", "POST", "/api/v0/issues/{issue_id}/sub-issues", `{"sub_issue_ids":["{other_id}"]}`},
		{"createIssueRelations", "POST", "/api/v0/issues/{issue_id}/relations", `{"relation_type":"relates_to","issue_ids":["{other_id}"]}`},
		{"deleteIssueRelation", "DELETE", "/api/v0/issue-relations/{relation_id}", ""},
		{"addIssueReaction", "POST", "/api/v0/issues/{issue_id}/reactions", `{"reaction":"128077"}`},
		{"removeIssueReaction", "DELETE", "/api/v0/issues/{issue_id}/reactions/{reaction}", ""},
		{"addCommentReaction", "POST", "/api/v0/issue-comments/{comment_id}/reactions", `{"reaction":"128077"}`},
		{"removeCommentReaction", "DELETE", "/api/v0/issue-comments/{comment_id}/reactions/{reaction}", ""},
		{"createIssueLink", "POST", "/api/v0/issues/{issue_id}/links", `{"url":"http://169.254.169.254/window-new"}`},
		{"updateIssueLink", "PATCH", "/api/v0/issue-links/{link_id}", `{"title":"Changed"}`},
		{"deleteIssueLink", "DELETE", "/api/v0/issue-links/{link_id}", ""},
		{"createIssueComment", "POST", "/api/v0/issues/{issue_id}/comments", `{"comment_html":"<p>Changed</p>"}`},
		{"updateIssueComment", "PATCH", "/api/v0/issue-comments/{comment_id}", `{"comment_html":"<p>Changed</p>"}`},
		{"deleteIssueComment", "DELETE", "/api/v0/issue-comments/{comment_id}", ""},
	}
}

func TestDeletionWindowCoversEveryIssueOperation(t *testing.T) {
	contract := apitest.Load(t)
	rows := deletionWindowCases()
	var reads, writes, actual, want []string
	byID := map[string]apitest.Operation{}
	for _, op := range contract.Operations() {
		if slices.Contains(op.Tags, "issue") {
			want = append(want, op.ID)
			byID[op.ID] = op
		}
	}
	for _, row := range rows {
		op, ok := byID[row.op]
		if !ok || row.method != op.Method || row.pattern != op.Path {
			t.Errorf("window case %+v is not its real issue operation %+v", row, op)
		}
		actual = append(actual, row.op)
		if row.method == http.MethodGet {
			reads = append(reads, row.op)
		} else {
			writes = append(writes, row.op)
		}
	}
	slices.Sort(actual)
	slices.Sort(want)
	if len(reads) != 15 || len(writes) != 22 || !slices.Equal(actual, want) {
		t.Fatalf("window reads=%d writes=%d operations=%q; contract=%q", len(reads), len(writes), actual, want)
	}
	checkIssueWriteDirectories(t, contract)
	slices.Sort(writes)
	if !slices.Equal(writes, writesOnIssues(contract.Operations())) {
		t.Fatal("window writes lost the ordinary/draft lock directory union")
	}
}

type deletionWindowTargets struct {
	project, issue, archived, other, comment, link, version, relation, draft, user uuid.UUID
	identifier, slug                                                               string
}

func (x deletionWindowTargets) replace(s string) string {
	return strings.NewReplacer("{project_id}", x.project.String(), "{issue_id}", x.issue.String(),
		"{other_id}", x.other.String(), "{comment_id}", x.comment.String(), "{link_id}", x.link.String(),
		"{version_id}", x.version.String(), "{relation_id}", x.relation.String(), "{draft_id}", x.draft.String(),
		"{user_id}", x.user.String(), "{identifier}", x.identifier, "{slug}", x.slug, "{reaction}", "128077").Replace(s)
}

func (x deletionWindowTargets) absent(workspace bool) deletionWindowTargets {
	x.project, x.issue, x.archived, x.other = uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	x.comment, x.link, x.version, x.relation, x.draft = uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	x.identifier = "ABSENT-999999"
	if workspace {
		x.slug = "absent"
	}
	return x
}

func (r deletionWindowCase) request(x deletionWindowTargets) (string, string) {
	if r.op == "unarchiveIssue" {
		x.issue = x.archived
	}
	path := x.replace(r.pattern)
	switch r.op {
	case "searchIssues":
		path += "?q=Window&limit=100"
	case "listMemberIssues":
		path += "?relation=created&order_by=-created_at"
	}
	return path, x.replace(r.body)
}
