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

// twoProjects are the rows of web and ops: web has three work items, an
// assignee, a relation and two activity rows, ops a work item, a label on
// it, a relation and a subscriber. Web's work items and its activity each
// fill a batch of two, which fakeRows answers greatest first and least
// first, in that order.
func twoProjects(web, ops uuid.UUID) *fakeRows {
	w, o := ids(7), ids(4)
	return &fakeRows{deleted: map[uuid.UUID]string{}, rows: map[string]map[uuid.UUID][]uuid.UUID{
		"issues":      {web: w[0:3], ops: o[0:1]},
		"assignees":   {web: w[3:4]},
		"labels":      {ops: o[1:2]},
		"relations":   {web: w[6:7], ops: o[3:4]},
		"subscribers": {ops: o[2:3]},
		"activities":  {web: w[4:6]},
	}}
}

// batches are the calls deleteProjectRows makes on project's rows of f,
// in batches of two: per statement in the global order, a batch after the
// zero id, then after the greatest id of each batch, until one finds none.
func batches(f *fakeRows, project uuid.UUID) []string {
	var calls []string
	for _, statement := range []string{"issues", "assignees", "labels", "relations", "subscribers", "comments", "activities"} {
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
		{"a workspace's two", func(c *app.Cascade) error { return c.DeleteWorkspaceIssues(inTx, []uuid.UUID{ops, web}, by, now) },
			[]uuid.UUID{ops, web}},
		{"a workspace without projects", func(c *app.Cascade) error { return c.DeleteWorkspaceIssues(inTx, nil, by, now) }, nil},
	} {
		f := twoProjects(web, ops)
		if err := tt.run(app.NewCascade(f, nil, 2)); err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		var want []string
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
	for _, statement := range []string{"issues", "assignees", "labels", "relations", "subscribers", "comments", "activities"} {
		f := twoProjects(web, ops)
		f.fail = statement
		want := batches(f, web)
		want = want[:slices.IndexFunc(want, func(c string) bool { return strings.HasPrefix(c, statement+" ") })+1]
		err := app.NewCascade(f, nil, 2).DeleteWorkspaceIssues(context.WithValue(context.Background(), inTxKey{}, true), []uuid.UUID{web, ops}, by,
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
