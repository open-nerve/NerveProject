package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// fakeRows is the deleted project's rows, by statement and project, as
// ProjectRowsDeleter's statements see them: each batch takes the
// undeleted rows of the project after the batch's id, at most its limit,
// in id order, marks them deleted, and answers their ids in no set order:
// of its answers of more than one id, the first, third and so on come
// greatest first, the others least first. A caller who took the first or
// the last id of an answer for the next batch's bound, not the greatest,
// would so read again what it deleted. It logs each batch in its callLog,
// which fails the statement named in fail.
type fakeRows struct {
	callLog
	rows    map[string]map[uuid.UUID][]uuid.UUID // statement → project → ids, ascending
	deleted map[uuid.UUID]string                 // id → "by at"
	turns   int                                  // the answers of more than one id so far
}

func (f *fakeRows) batch(ctx context.Context, statement string, b app.RowBatch) ([]uuid.UUID, error) {
	if err := f.add(ctx, statement, b.ProjectID, "after", b.After, "limit", b.Limit); err != nil {
		return nil, err
	}
	var ids []uuid.UUID
	for _, id := range f.rows[statement][b.ProjectID] {
		if _, gone := f.deleted[id]; !gone && id.Compare(b.After) > 0 && len(ids) < b.Limit {
			ids = append(ids, id)
			f.deleted[id] = fmt.Sprintf("%s at %s", b.By, b.At.Format(time.RFC3339Nano))
		}
	}
	if len(ids) > 1 {
		f.turns++
		if f.turns%2 == 1 {
			slices.Reverse(ids)
		}
	}
	return ids, nil
}

func (f *fakeRows) DeleteProjectIssueRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.batch(ctx, "issues", b)
}

func (f *fakeRows) DeleteProjectAssigneeRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.batch(ctx, "assignees", b)
}

func (f *fakeRows) DeleteProjectLabelRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.batch(ctx, "labels", b)
}

func (f *fakeRows) DeleteProjectRelationRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.batch(ctx, "relations", b)
}

func (f *fakeRows) DeleteProjectSubscriberRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.batch(ctx, "subscribers", b)
}

func (f *fakeRows) DeleteProjectActivityRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.batch(ctx, "activities", b)
}

// ids are n ids drawn in order: ascending.
func ids(n int) []uuid.UUID {
	out := make([]uuid.UUID, n)
	for i := range out {
		out[i] = uuid.NewV7()
	}
	return out
}

// twoProjects are the rows of web and ops, each with three drafts. Web has
// three work items, an assignee, a relation and two activity rows; ops a work
// item, a label on it, a relation and a subscriber. Several tables fill a
// batch of two, which fakeRows alternates greatest first and least first.
func twoProjects(web, ops uuid.UUID) *fakeRows {
	w, o := ids(13), ids(8)
	return &fakeRows{deleted: map[uuid.UUID]string{}, rows: map[string]map[uuid.UUID][]uuid.UUID{
		"drafts":      {web: w[10:13], ops: o[5:8]},
		"issues":      {web: w[0:3], ops: o[0:1]},
		"assignees":   {web: w[3:4]},
		"labels":      {ops: o[1:2]},
		"relations":   {web: w[6:7], ops: o[3:4]},
		"subscribers": {ops: o[2:3]},
		"links":       {web: w[7:10], ops: o[4:5]},
		"activities":  {web: w[4:6]},
	}}
}

// batches are the calls deleteProjectRows makes on project's rows of f,
// in batches of two: per statement in the global order, a batch after the
// zero id, then after the greatest id of each batch, until one finds none.
func batches(f *fakeRows, project uuid.UUID) []string {
	var calls []string
	for _, statement := range []string{"drafts", "issues", "assignees", "labels", "relations", "links", "subscribers", "mentions", "comments", "reactions", "comment_reactions", "activities", "description_versions"} {
		rows := f.rows[statement][project]
		after := uuid.UUID{}
		for i := 0; ; i += 2 {
			calls = append(calls, fmt.Sprintf("%s %s after %s limit 2", statement, project, after))
			if i >= len(rows) {
				break
			}
			after = rows[min(i+2, len(rows))-1]
		}
	}
	return calls
}

// DeleteProjectIssues soft-deletes every row of the project's work-item
// side, and none of another project's, in the caller's transaction, at its
// moment and by its account: statement by statement in the global order of
// M4 design 3.5, each in batches of the limit, the next after the greatest
// id of the last, until a batch finds none (M4 design 3.6).
// DeleteWorkspaceIssues does it for each project it is given, in that
// order.
func TestTheCascadeDeletesEachProjectsRowsInBatches(t *testing.T) {
	web, ops, by := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	now := time.Date(2026, 10, 17, 10, 0, 0, 123456000, time.UTC)
	inTx := context.WithValue(context.Background(), inTxKey{}, true)
	for _, tt := range []struct {
		name     string
		run      func(c *app.Cascade) error
		projects []uuid.UUID
	}{
		{"one project", func(c *app.Cascade) error { return c.DeleteProjectIssues(inTx, web, by, now) }, []uuid.UUID{web}},
		{"a workspace's two", func(c *app.Cascade) error {
			return c.DeleteWorkspaceIssues(inTx, uuid.UUID{}, []uuid.UUID{ops, web}, by, now)
		},
			[]uuid.UUID{ops, web}},
		{"a workspace without projects", func(c *app.Cascade) error { return c.DeleteWorkspaceIssues(inTx, uuid.UUID{}, nil, by, now) }, nil},
	} {
		f := twoProjects(web, ops)
		if err := tt.run(app.NewCascade(f, nil, 2)); err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		var want []string
		if tt.name != "one project" {
			want = []string{fmt.Sprintf("workspace_drafts %s after %s limit 2", uuid.UUID{}, uuid.UUID{})}
		}
		deleted := map[uuid.UUID]string{}
		for _, p := range tt.projects {
			want = append(want, batches(f, p)...)
			for _, rows := range f.rows {
				for _, id := range rows[p] {
					deleted[id] = fmt.Sprintf("%s at %s", by, now.Format(time.RFC3339Nano))
				}
			}
		}
		if !slices.Equal(f.calls, want) {
			t.Errorf("%s: the batches\n%q\nwant\n%q", tt.name, f.calls, want)
		}
		if fmt.Sprint(f.deleted) != fmt.Sprint(deleted) {
			t.Errorf("%s: deleted %v, want %v", tt.name, f.deleted, deleted)
		}
	}
}

// A failing batch comes back as itself, and no batch runs after it.
func TestTheCascadeStopsAtAFailingBatch(t *testing.T) {
	web, ops, by := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for _, statement := range []string{"workspace_drafts", "drafts", "issues", "assignees", "labels", "relations", "links", "subscribers", "mentions", "comments", "reactions", "comment_reactions", "activities", "description_versions"} {
		f := twoProjects(web, ops)
		f.fail = statement
		want := append([]string{fmt.Sprintf("workspace_drafts %s after %s limit 2", uuid.UUID{}, uuid.UUID{})}, batches(f, web)...)
		want = want[:slices.IndexFunc(want, func(c string) bool { return strings.HasPrefix(c, statement+" ") })+1]
		err := app.NewCascade(f, nil, 2).DeleteWorkspaceIssues(context.WithValue(context.Background(), inTxKey{}, true), uuid.UUID{}, []uuid.UUID{web, ops}, by,
			time.Now())
		if !errors.Is(err, errDisk) || !slices.Equal(f.calls, want) {
			t.Errorf("failing %s: %v, the batches\n%q\nwant %v after\n%q", statement, err, f.calls, errDisk, want)
		}
	}
}

// DeleteProjectCommentRows soft-deletes comment rows in the existing deletion sequence.
func (f *fakeRows) DeleteProjectCommentRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.batch(ctx, "comments", b)
}

// DeleteProjectReactionRows soft-deletes reaction rows in the existing deletion sequence.
func (f *fakeRows) DeleteProjectReactionRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.batch(ctx, "reactions", b)
}

// DeleteProjectCommentReactionRows soft-deletes reaction rows in the existing deletion sequence.
func (f *fakeRows) DeleteProjectCommentReactionRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.batch(ctx, "comment_reactions", b)
}

func (f *fakeRows) DeleteProjectMentionRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.batch(ctx, "mentions", b)
}

func (f *fakeRows) DeleteProjectDescriptionVersionRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.batch(ctx, "description_versions", b)
}

func (f *fakeRows) DeleteProjectLinkRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.batch(ctx, "links", b)
}

func (f *fakeRows) DeleteProjectDraftRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.batch(ctx, "drafts", b)
}
func (f *fakeRows) DeleteWorkspaceDraftRows(ctx context.Context, b app.WorkspaceDraftBatch) ([]uuid.UUID, error) {
	return f.batch(ctx, "workspace_drafts", app.RowBatch{ProjectID: b.WorkspaceID, After: b.After, Limit: b.Limit, By: b.By, At: b.At})
}

// Dropping workspaceID or skipping the batch for an empty project list leaves
// these drafts alive; running issues first violates the shared lock order.
func TestWorkspaceDraftCascadeCarriesWorkspaceID(t *testing.T) {
	workspace, project, by := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	now := time.Date(2026, 10, 27, 1, 0, 0, 0, time.UTC)
	for _, projects := range [][]uuid.UUID{nil, {project}} {
		f := &fakeRows{deleted: map[uuid.UUID]string{}, rows: map[string]map[uuid.UUID][]uuid.UUID{
			"workspace_drafts": {workspace: ids(2)}, "drafts": {project: ids(2)}, "issues": {project: ids(1)},
		}}
		if err := app.NewCascade(f, nil, 1).DeleteWorkspaceIssues(context.WithValue(context.Background(), inTxKey{}, true), workspace, projects, by, now); err != nil {
			t.Fatal(err)
		}
		want := 2
		if len(projects) > 0 {
			want = 5
		}
		if len(f.deleted) != want {
			t.Fatalf("deleted %d rows; want %d", len(f.deleted), want)
		}
		for _, stamp := range f.deleted {
			if stamp != fmt.Sprintf("%s at %s", by, now.Format(time.RFC3339Nano)) {
				t.Errorf("stamp %s", stamp)
			}
		}
		if !strings.HasPrefix(f.calls[0], "workspace_drafts "+workspace.String()+" ") {
			t.Errorf("first batch: %v", f.calls)
		}
		if len(projects) > 0 && (!strings.HasPrefix(f.calls[3], "drafts ") || !strings.HasPrefix(f.calls[6], "issues ")) {
			t.Errorf("drafts must precede issues: %v", f.calls)
		}
	}
}
