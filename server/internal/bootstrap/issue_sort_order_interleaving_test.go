package bootstrap

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"reflect"
	"testing"
	"time"
	"uuid"
)

func sortBody(value float64) func(*testing.T, issueWorld) string {
	return func(*testing.T, issueWorld) string { return fmt.Sprintf(`{"sort_order":%g}`, value) }
}

// Compare raw columns as well as the typed output: typed Issue omits description.
func sortUnchangedColumns(t *testing.T, before, after string, allowed ...string) {
	t.Helper()
	var a, b []map[string]json.RawMessage
	decodeAnswer(t, before, &a)
	decodeAnswer(t, after, &b)
	for _, rows := range [][]map[string]json.RawMessage{a, b} {
		for _, row := range rows {
			for _, key := range allowed {
				delete(row, key)
			}
		}
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatal("unexpected issue column change", a, b)
	}
}

// A legitimate state/archive write adds exactly its one activity. Every
// pre-existing raw row, including its stamps, must still be identical.
func sortAddedActivity(t *testing.T, before, after string, field string, actor uuid.UUID, at time.Time) {
	t.Helper()
	var a, b []map[string]json.RawMessage
	decodeAnswer(t, before, &a)
	decodeAnswer(t, after, &b)
	old := map[string]map[string]json.RawMessage{}
	for _, r := range a {
		old[string(r["id"])] = r
	}
	added := 0
	for _, r := range b {
		if was, ok := old[string(r["id"])]; ok {
			if !reflect.DeepEqual(was, r) {
				t.Fatal("old activity rewritten", was, r)
			}
			delete(old, string(r["id"]))
			continue
		}
		added++
		var got struct {
			Field       string
			ActorID     uuid.UUID `json:"actor_id"`
			CreatedByID uuid.UUID `json:"created_by_id"`
			UpdatedByID uuid.UUID `json:"updated_by_id"`
			CreatedAt   time.Time `json:"created_at"`
			UpdatedAt   time.Time `json:"updated_at"`
		}
		encoded, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		decodeAnswer(t, string(encoded), &got)
		if got.Field != field || got.ActorID != actor || got.CreatedByID != actor || got.UpdatedByID != actor || !got.CreatedAt.Equal(at) || !got.UpdatedAt.Equal(at) {
			t.Fatal("wrong added activity", got, field, actor, at)
		}
	}
	if len(old) != 0 || added != 1 {
		t.Fatal("want all old activities plus exactly one", len(old), added)
	}
}

func TestIssueSortOrderInterleaving(t *testing.T) {
	t.Run("project archive", func(t *testing.T) {
		var before map[string]string
		inBothOrders(t, func(t *testing.T, w issueWorld) { aWorkItem(t, w); before = sortRows(t, w, w.workItem(t)) }, "sort", changesIssue("bob", sortBody(0)), "archive", archivesWeb,
			func(t *testing.T, w issueWorld, sortFirst bool, sorted, archived exchange) {
				w.answered(t, archived, 200)
				id := w.workItem(t)
				if !sortFirst {
					w.refused(t, sorted, 409, "project.archived")
					if !maps.Equal(before, sortRows(t, w, id)) {
						t.Fatal("refused sort changed raw rows")
					}
					return
				}
				got := sortAnswer(t, w.answered(t, sorted, 200))
				stored := sortStored(t, w, id)
				if stored.SortOrder != 0 || !stored.UpdatedAt.Equal(got.UpdatedAt) || *stored.UpdatedByID != w.ids["bob"] {
					t.Fatal("sort lost after project archive", got, stored)
				}
				after := sortRows(t, w, id)
				sortUnchangedColumns(t, before["issues"], after["issues"], "sort_order", "updated_at", "updated_by_id")
				delete(before, "issues")
				delete(after, "issues")
				if !maps.Equal(before, after) {
					t.Fatal("sort changed side rows")
				}
			})
	})
	t.Run("issue archive", func(t *testing.T) {
		var before map[string]string
		var initial storedIssue
		inBothOrders(t, func(t *testing.T, w issueWorld) {
			id := w.doneWorkItem(t)
			before = sortRows(t, w, id)
			initial = sortStored(t, w, id)
		}, "sort", changesIssue("bob", sortBody(0)), "archive", archivesIssue("alice"),
			func(t *testing.T, w issueWorld, sortFirst bool, sorted, archived exchange) {
				archive := sortAnswer(t, w.answered(t, archived, 200))
				if sortFirst {
					w.answered(t, sorted, 200)
				} else {
					w.refused(t, sorted, 409, "issue.archived")
				}
				id := w.workItem(t)
				stored := sortStored(t, w, id)
				want := initial.SortOrder
				if sortFirst {
					want = 0
				}
				if stored.SortOrder != want || archive.SortOrder != want || stored.StateID != w.done || !reflect.DeepEqual(stored.CompletedAt, initial.CompletedAt) || stored.ArchivedAt == nil || !stored.UpdatedAt.Equal(archive.UpdatedAt) || *stored.UpdatedByID != w.ids["alice"] {
					t.Fatal("archive/sort final state", stored, archive)
				}
				after := sortRows(t, w, id)
				sortUnchangedColumns(t, before["issues"], after["issues"], "sort_order", "updated_at", "updated_by_id", "archived_at")
				sortAddedActivity(t, before["issue_activities"], after["issue_activities"], "archived_at", w.ids["alice"], archive.UpdatedAt)
				delete(before, "issues")
				delete(after, "issues")
				delete(before, "issue_activities")
				delete(after, "issue_activities")
				if !maps.Equal(before, after) {
					t.Fatal("archive/sort changed side rows")
				}
			})
	})
	for _, own := range []bool{false, true} {
		for _, change := range []struct {
			name   string
			write  racingWrite
			status int
		}{{"removed", removesBob, http.StatusNoContent}, {"demoted", demotesBob, http.StatusOK}} {
			t.Run(fmt.Sprintf("%s own=%t", change.name, own), func(t *testing.T) {
				var before map[string]string
				inBothOrders(t, func(t *testing.T, w issueWorld) {
					w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"network":0}`, 200)
					if own {
						bobsWorkItem(t, w)
					} else {
						aWorkItem(t, w)
					}
					before = sortRows(t, w, w.workItem(t))
				}, "sort", changesIssue("bob", sortBody(-7)), change.name, change.write,
					func(t *testing.T, w issueWorld, sortFirst bool, sorted, membership exchange) {
						w.answered(t, membership, change.status)
						id := w.workItem(t)
						if !sortFirst {
							status, code := 404, "issue.not_found"
							if own && change.name == "demoted" {
								status, code = 403, "forbidden"
							}
							w.refused(t, sorted, status, code)
							if !maps.Equal(before, sortRows(t, w, id)) {
								t.Fatal("refused sort changed raw rows")
							}
							return
						}
						got := sortAnswer(t, w.answered(t, sorted, 200))
						stored := sortStored(t, w, id)
						if stored.SortOrder != -7 || !stored.UpdatedAt.Equal(got.UpdatedAt) || *stored.UpdatedByID != w.ids["bob"] {
							t.Fatal("successful sort lost on membership change", got, stored)
						}
						after := sortRows(t, w, id)
						sortUnchangedColumns(t, before["issues"], after["issues"], "sort_order", "updated_at", "updated_by_id")
						delete(before, "issues")
						delete(after, "issues")
						if !maps.Equal(before, after) {
							t.Fatal("membership/sort changed side rows")
						}
					})
			})
		}
	}
	for _, state := range []bool{false, true} {
		t.Run(map[bool]string{false: "another sort", true: "state only"}[state], func(t *testing.T) {
			var before map[string]string
			second := changesIssue("alice", sortBody(19))
			if state {
				second = changesIssue("alice", func(_ *testing.T, w issueWorld) string { return fmt.Sprintf(`{"state_id":%q}`, w.done) })
			}
			inBothOrders(t, func(t *testing.T, w issueWorld) { bobsWorkItem(t, w); before = sortRows(t, w, w.workItem(t)) }, "bob sort", changesIssue("bob", sortBody(0)), "alice write", second,
				func(t *testing.T, w issueWorld, bobFirst bool, bob, alice exchange) {
					b := sortAnswer(t, w.answered(t, bob, 200))
					a := sortAnswer(t, w.answered(t, alice, 200))
					if b.SortOrder != 0 || b.UpdatedByID == nil || *b.UpdatedByID != w.ids["bob"] || a.UpdatedByID == nil || *a.UpdatedByID != w.ids["alice"] || !state && a.SortOrder != 19 {
						t.Fatal("response lost its requested value/writer", b, a)
					}
					first, last, released := b, a, bob.released
					if !bobFirst {
						first, last, released = a, b, alice.released
					}
					if last.UpdatedAt.Before(released) || last.UpdatedAt.Before(first.UpdatedAt) {
						t.Fatal("clock read before final lock", first, last, released)
					}
					id := w.workItem(t)
					stored := sortStored(t, w, id)
					want := last.SortOrder
					if state {
						want = 0
					}
					if stored.SortOrder != want || last.SortOrder != want || !stored.UpdatedAt.Equal(last.UpdatedAt) || !reflect.DeepEqual(stored.UpdatedByID, last.UpdatedByID) {
						t.Fatal("lost concurrent write", stored, last)
					}
					after := sortRows(t, w, id)
					allowed := []string{"sort_order", "updated_at", "updated_by_id"}
					if state {
						if stored.StateID != w.done || stored.CompletedAt == nil || !stored.CompletedAt.Equal(a.UpdatedAt) {
							t.Fatal("sort lost state completion", stored, a)
						}
						sortAddedActivity(t, before["issue_activities"], after["issue_activities"], "state", w.ids["alice"], a.UpdatedAt)
						if said := w.activityBy(t, id, w.ids["alice"]); !reflect.DeepEqual(said, []string{"state: Backlog → Done"}) {
							t.Fatal("state activity", said)
						}
						allowed = append(allowed, "state_id", "completed_at")
						delete(before, "issue_activities")
						delete(after, "issue_activities")
					}
					sortUnchangedColumns(t, before["issues"], after["issues"], allowed...)
					delete(before, "issues")
					delete(after, "issues")
					if !maps.Equal(before, after) {
						t.Fatal("concurrent sort changed unrelated raw rows")
					}
				})
		})
	}
}
