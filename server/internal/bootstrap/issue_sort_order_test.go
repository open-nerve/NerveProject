package bootstrap

import (
	"fmt"
	"maps"
	"math"
	"net/http"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// sortRows includes row contents, not just counts: a refused patch must not
// rewrite the issue, its audit or any of the update's dependent rows.
func sortRows(t *testing.T, w issueWorld, id uuid.UUID) map[string]string {
	t.Helper()
	rows := map[string]string{}
	for _, table := range []string{"issues", "issue_assignees", "issue_labels", "issue_subscribers", "issue_mentions", "issue_activities", "issue_description_versions"} {
		rows[table] = descriptionRows(t, w, table, id)
	}
	return rows
}

func sortStored(t *testing.T, w issueWorld, id uuid.UUID) storedIssue {
	t.Helper()
	var rows []storedIssue
	decodeAnswer(t, descriptionRows(t, w, "issues", id), &rows)
	if len(rows) != 1 {
		t.Fatalf("want one stored issue, got %v", rows)
	}
	rows[0].inUTC()
	return rows[0]
}

func sortAnswer(t *testing.T, body string) storedIssue {
	t.Helper()
	var i storedIssue
	decodeAnswer(t, body, &i)
	i.inUTC()
	return i
}

func TestIssueSortOrderPatch(t *testing.T) {
	t.Run("finite values and unchanged rows", func(t *testing.T) {
		w := newIssueWorld(t)
		id := w.newWorkItem(t)
		for _, value := range []float64{0, -12.5, math.MaxFloat64, 1, math.Nextafter(1, 2)} {
			before, old := sortRows(t, w, id), sortStored(t, w, id)
			start := time.Now().Truncate(time.Microsecond)
			got := sortAnswer(t, w.send(t, "bob", http.MethodPatch, "/api/v0/issues/"+id.String(), fmt.Sprintf(`{"sort_order":%g}`, value), 200))
			stored := sortStored(t, w, id)
			if got.SortOrder != value || stored.SortOrder != value || got.UpdatedByID == nil || *got.UpdatedByID != w.ids["bob"] ||
				!got.UpdatedAt.Equal(stored.UpdatedAt) || got.UpdatedAt.Before(start) || got.UpdatedAt.After(time.Now()) {
				t.Fatalf("sort %g: response %+v, stored %+v", value, got, stored)
			}
			stored.SortOrder, stored.UpdatedAt, stored.UpdatedByID = old.SortOrder, old.UpdatedAt, old.UpdatedByID
			if !reflect.DeepEqual(stored, old) {
				t.Fatal("sort changed another issue field", old, stored)
			}
			after := sortRows(t, w, id)
			sortUnchangedColumns(t, before["issues"], after["issues"], "sort_order", "updated_at", "updated_by_id")
			delete(before, "issues")
			delete(after, "issues")
			if !maps.Equal(before, after) {
				t.Fatal("sort changed dependent rows", before, after)
			}
			unchanged := sortRows(t, w, id)
			w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+id.String(), fmt.Sprintf(`{"sort_order":%g}`, value), 200)
			if !maps.Equal(unchanged, sortRows(t, w, id)) {
				t.Fatal("same value rewrote rows")
			}
		}
		// An ordinary full-row update must preserve the latest non-default value.
		w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+id.String(), `{"name":"Preserved"}`, 200)
		if got := sortStored(t, w, id); got.SortOrder != math.Nextafter(1, 2) {
			t.Fatal("omission lost sort", got)
		}
	})
	t.Run("state and sort", func(t *testing.T) {
		w := newIssueWorld(t)
		id := w.newWorkItem(t)
		before := sortRows(t, w, id)
		got := sortAnswer(t, w.send(t, "bob", http.MethodPatch, "/api/v0/issues/"+id.String(), fmt.Sprintf(`{"sort_order":0,"state_id":%q}`, w.done), 200))
		stored := sortStored(t, w, id)
		if got.SortOrder != 0 || stored.SortOrder != 0 || stored.StateID != w.done || stored.CompletedAt == nil ||
			!stored.CompletedAt.Equal(stored.UpdatedAt) || !got.UpdatedAt.Equal(stored.UpdatedAt) || *stored.UpdatedByID != w.ids["bob"] {
			t.Fatal("state + sort", got, stored)
		}
		if got := w.activityBy(t, id, w.ids["bob"]); !reflect.DeepEqual(got, []string{"state: Todo → Done"}) {
			t.Fatal("want only state activity", got)
		}
		after := sortRows(t, w, id)
		sortUnchangedColumns(t, before["issues"], after["issues"], "sort_order", "state_id", "completed_at", "updated_at", "updated_by_id")
		sortAddedActivity(t, before["issue_activities"], after["issue_activities"], "state", w.ids["bob"], stored.UpdatedAt)
		for table, rows := range before {
			if table != "issues" && table != "issue_activities" && rows != descriptionRows(t, w, table, id) {
				t.Fatal("state + sort changed unrelated rows", table)
			}
		}
		activity, completed := after["issue_activities"], stored.CompletedAt
		got = sortAnswer(t, w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+id.String(), fmt.Sprintf(`{"sort_order":19,"state_id":%q}`, w.done), 200))
		if got.SortOrder != 19 || !reflect.DeepEqual(got.CompletedAt, completed) || activity != descriptionRows(t, w, "issue_activities", id) {
			t.Fatal("unchanged state created activity or changed completion", got)
		}
	})
	t.Run("ordinary description and sort", func(t *testing.T) {
		w := newIssueWorld(t)
		id := w.newWorkItem(t)
		before := descriptionRows(t, w, "issue_activities", id)
		got := sortAnswer(t, w.send(t, "bob", http.MethodPatch, "/api/v0/issues/"+id.String(), `{"sort_order":0,"description_html":"<p>Sorted description</p>"}`, 200))
		stored := sortStored(t, w, id)
		v := savedVersion(t, w, id)
		if got.SortOrder != 0 || stored.SortOrder != 0 || v.HTML != "<p>Sorted description</p>" {
			t.Fatal("description+sort lost write", got, stored, v)
		}
		assertStoredDescription(t, pgtest.Soon(t), w, id, v.HTML, &v.Text)
		sortAddedActivity(t, before, descriptionRows(t, w, "issue_activities", id), "description", w.ids["bob"], got.UpdatedAt)
	})
}

func TestSkipActivityRejectsSortOrder(t *testing.T) {
	w := newIssueWorld(t)
	id := w.newWorkItemIn(t, w.web, `{"name":"Skip","description_html":"<p>Original</p>"}`)
	before := sortRows(t, w, id)
	for _, skip := range []bool{false, true} {
		for _, value := range []float64{0, 65535} {
			body := w.send(t, "bob", http.MethodPatch, "/api/v0/issues/"+id.String(), fmt.Sprintf(`{"description_html":"<p data-id=\"migration\">Original</p>","skip_activity":%t,"sort_order":%g}`, skip, value), 422)
			var problem struct {
				Code   string
				Errors []struct{ Field, Code string }
			}
			decodeAnswer(t, body, &problem)
			if problem.Code != "validation_failed" || len(problem.Errors) != 1 || problem.Errors[0].Field != "skip_activity" || problem.Errors[0].Code != "not_allowed" {
				t.Fatal("wrong skip refusal", body)
			}
			if !maps.Equal(before, sortRows(t, w, id)) {
				t.Fatalf("skip=%t, sort=%g wrote rows", skip, value)
			}
		}
	}
}
