package bootstrap

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// collaborationClock makes Fixed safe while the controller advances it and
// independent request goroutines read it. No production clock is changed.
type collaborationClock struct {
	mu    sync.Mutex
	fixed *clocktest.Fixed
}

func clockAt(at time.Time) *collaborationClock { return &collaborationClock{fixed: clocktest.At(at)} }
func (c *collaborationClock) Now() time.Time   { c.mu.Lock(); defer c.mu.Unlock(); return c.fixed.Now() }
func (c *collaborationClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fixed.Advance(d)
}

func TestDescriptionSavesSerialize(t *testing.T) {
	for _, secondActor := range []string{"bob", "alice"} {
		t.Run(secondActor, func(t *testing.T) {
			w := newIssueWorld(t)
			initialHTML, firstHTML, secondHTML := "<p>Initial</p>", "<p>First save</p>", "<p>Second save</p>"
			c := clockAt(time.Date(2044, 5, 6, 7, 8, 9, 0, time.UTC))
			route := newIssueRouteWithClock(t, w.pool, postgres.NewTxManager(w.pool, 2*time.Second), c)
			var created struct {
				ID uuid.UUID `json:"id"`
			}
			decodeAnswer(t, route.answerWithin(t, w.contract, http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", w.ids["alice"], fmt.Sprintf(`{"name":"Serialized description","description_html":%q}`, initialHTML), 201), &created)
			id := created.ID
			// An older row has the greatest id: latest must follow time before id.
			high := uuid.MustParse("ffffffff-ffff-7fff-bfff-ffffffffffff")
			for _, table := range []string{"issue_activities", "issue_description_versions"} {
				if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE "+table+" SET id=$2 WHERE issue_id=$1", id, high); err != nil {
					t.Fatal(err)
				}
			}
			initialVersion := savedVersion(t, w, id)
			c.advance(time.Hour)
			firstAt := c.Now()
			path := "/api/v0/issues/" + id.String()
			first := collaborationAt("bob", http.MethodPatch, path, fmt.Sprintf(`{"description_html":%q}`, firstHTML), c)
			second := collaborationAt(secondActor, http.MethodPatch, path, fmt.Sprintf(`{"description_html":%q}`, secondHTML), c)
			var a, b exchange
			w.serializeAtAfter(t, "projects", first, second, &a, &b, func() {
				if !pgtest.RowWaits(t, w.pool, "projects", w.web, "FOR NO KEY UPDATE") {
					t.Fatal("description holder lost project N lock")
				}
				c.advance(500 * time.Second)
			})
			var one, two struct {
				HTML string    `json:"description_html"`
				At   time.Time `json:"updated_at"`
			}
			decodeAnswer(t, w.answered(t, a, 200), &one)
			decodeAnswer(t, w.answered(t, b, 200), &two)
			if one.HTML != firstHTML || two.HTML != secondHTML || !one.At.Equal(firstAt) || !two.At.Equal(c.Now()) || two.At.Before(one.At) {
				t.Fatal("lost serialized description or lock-time clock", one, two, c.Now())
			}
			desc := queryIDs(t, w.pool, "SELECT id FROM issue_activities WHERE issue_id=$1 AND field='description' ORDER BY created_at,id", id)
			versions := queryIDs(t, w.pool, "SELECT id FROM issue_description_versions WHERE issue_id=$1 AND id<>$2 ORDER BY last_saved_at,id", id, initialVersion.ID)
			want := 1
			if secondActor != "bob" {
				want = 2
			}
			if len(desc) != want || len(versions) != want {
				t.Fatal("serialized coalescing", desc, versions, want)
			}
			var old, new string
			var actor uuid.UUID
			var at time.Time
			if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT old_value,new_value,actor_id,created_at FROM issue_activities WHERE id=$1", desc[len(desc)-1]).Scan(&old, &new, &actor, &at); err != nil {
				t.Fatal(err)
			}
			wantOld := initialHTML
			if want == 2 {
				wantOld = firstHTML
			}
			if old != wantOld || new != secondHTML || actor != w.ids[secondActor] || !at.Equal(c.Now()) {
				t.Fatal("lost description transition", old, new, actor, at)
			}
			v := savedVersion(t, w, id)
			if v.HTML != secondHTML || v.OwnedByID != w.ids[secondActor] || v.UpdatedByID != w.ids[secondActor] || !v.LastSavedAt.Equal(c.Now()) || !v.UpdatedAt.Equal(c.Now()) {
				t.Fatal("wrong lock-time version", v)
			}
			if want == 1 {
				if !v.CreatedAt.Equal(firstAt) || v.CreatedByID != w.ids["bob"] {
					t.Fatal("merged version lost origin", v)
				}
				// Now the origin is 1000 seconds old, but the previous save only 500:
				// a creation-time window would incorrectly insert another version.
				c.advance(500 * time.Second)
				route.answerWithin(t, w.contract, http.MethodPatch, path, w.ids["bob"], `{"description_html":"<p>Third sliding save</p>"}`, 200)
				after := savedVersion(t, w, id)
				if after.ID != v.ID || !after.CreatedAt.Equal(v.CreatedAt) || !after.LastSavedAt.Equal(c.Now()) {
					t.Fatal("window used version creation instead of last save", v, after)
				}
			}
		})
	}
}

func TestCollaborationReadsClockUnderLock(t *testing.T) {
	for _, higherProject := range []bool{true, false} {
		t.Run(fmt.Sprintf("newer_project=%t", higherProject), func(t *testing.T) {
			for _, operation := range []string{"description", "comment create", "comment edit", "comment delete", "issue reaction add", "issue reaction remove", "comment reaction add", "comment reaction remove"} {
				t.Run(operation, func(t *testing.T) {
					w := newIssueWorld(t)
					neighbor := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Later", "LATER")
					if higherProject {
						w.web, neighbor = neighbor, w.web
						w.makeMember(t, "bob", w.web, 15)
					}
					w.makeMember(t, "walt", w.web, 15)
					id := w.newWorkItemIn(t, w.web, `{"name":"Clock target"}`)
					comment := w.newComment(t, "bob", id)
					c := clockAt(time.Date(2044, 6, 7, 8, 9, 10, 0, time.UTC))
					method, path, body := http.MethodPatch, "/api/v0/issues/"+id.String(), fmt.Sprintf(`{"description_html":%q}`, mentionHTML(w.ids["walt"]))
					expected := map[string]int{"issues": 1, "issue_subscribers": 1, "issue_mentions": 1, "issue_activities": 1, "issue_description_versions": 1}
					switch operation {
					case "comment create", "comment edit":
						method, path, body = http.MethodPost, commentsOf(id), fmt.Sprintf(`{"comment_html":%q}`, mentionHTML(w.ids["walt"]))
						if operation == "comment edit" {
							method, path = http.MethodPatch, commentPath(comment.ID)
						}
						expected = map[string]int{"issues": 1, "issue_subscribers": 1, "issue_comments": 1}
					case "comment delete":
						addCommentReaction(t, w, "alice", comment.ID, "128077")
						method, path, body = http.MethodDelete, commentPath(comment.ID), ""
						expected = map[string]int{"issues": 1, "issue_comments": 1, "comment_reactions": 1}
					case "issue reaction add", "issue reaction remove":
						method, path, body = http.MethodPost, reactionsOf(id), `{"reaction":"128077"}`
						if operation == "issue reaction remove" {
							w.send(t, "bob", method, path, body, 201)
							method, path, body = http.MethodDelete, path+"/128077", ""
						}
						expected = map[string]int{"issues": 1, "issue_reactions": 1}
					case "comment reaction add", "comment reaction remove":
						method, path, body = http.MethodPost, commentReactionsOf(comment.ID), `{"reaction":"128077"}`
						if operation == "comment reaction remove" {
							addCommentReaction(t, w, "bob", comment.ID, "128077")
							method, path, body = http.MethodDelete, path+"/128077", ""
						}
						expected = map[string]int{"comment_reactions": 1}
					}
					var holder, writer exchange
					w.serializeAtAfter(t, "projects", changesIssue("alice", named("Holder committed first")), collaborationAt("bob", method, path, body, c), &holder, &writer, func() {
						if !pgtest.RowWaits(t, w.pool, "projects", w.web, "FOR NO KEY UPDATE") {
							t.Fatal("holder does not hold project N lock")
						}
						if pgtest.RowWaits(t, w.pool, "projects", neighbor, "FOR NO KEY UPDATE") {
							t.Fatal("unrelated project locked")
						}
						c.advance(time.Hour)
					})
					w.answered(t, holder, 200)
					status := 200
					if method == http.MethodPost {
						status = 201
					}
					if method == http.MethodDelete {
						status = 204
					}
					w.answered(t, writer, status)
					// Every changed audit, including a new subscription/mention and a cascaded
					// reaction deletion, must use the same clock read made after waiting.
					for table, want := range expected {
						condition := "issue_id=$1"
						if table == "issues" {
							condition = "id=$1"
						}
						if table == "comment_reactions" {
							condition = "comment_id IN (SELECT id FROM issue_comments WHERE issue_id=$1)"
						}
						var stamped int
						if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM "+table+" WHERE "+condition+" AND updated_by_id=$2 AND updated_at=$3", id, w.ids["bob"], c.Now()).Scan(&stamped); err != nil || stamped != want {
							t.Fatalf("%s clock read before lock: %d at %s, want %d: %v", table, stamped, c.Now(), want, err)
						}
						extra := ""
						switch {
						case operation == "comment delete" && table != "issues", operation == "issue reaction remove" && table == "issue_reactions", operation == "comment reaction remove":
							extra = "deleted_at=$3"
						case operation == "comment edit" && table == "issue_comments":
							extra = "edited_at=$3"
						case table != "issues":
							extra = "created_at=$3"
						}
						if extra != "" {
							var n int
							if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM "+table+" WHERE "+condition+" AND updated_by_id=$2 AND updated_at=$3 AND "+extra, id, w.ids["bob"], c.Now()).Scan(&n); err != nil || n != want {
								t.Fatal("side-table event time differs from audit", table, n, want, err)
							}
						}
					}
				})
			}

		})
	}
}

func TestActivityCreatedAtDoesNotGoBack(t *testing.T) {
	w := newIssueWorld(t)
	id := w.newWorkItemIn(t, w.web, `{"name":"Activity clock"}`)
	// The request starts at an earlier clock value than the committed predecessor.
	// Its clock is moved to the predecessor's time only after its project wait.
	c := clockAt(time.Date(2044, 7, 8, 9, 10, 11, 0, time.UTC))
	old := c.Now().Add(time.Hour)
	path := "/api/v0/issues/" + id.String()
	var first, second exchange
	a := collaborationAt("bob", http.MethodPatch, path, `{"description_html":"<p>First</p>"}`, clockAt(old))
	b := collaborationAt("bob", http.MethodPatch, path, `{"description_html":"<p>Second</p>"}`, c)
	w.serializeAtAfter(t, "projects", a, b, &first, &second, func() { c.advance(time.Hour) })
	w.answered(t, first, 200)
	w.answered(t, second, 200)
	var at time.Time
	var value string
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT created_at,new_value FROM issue_activities WHERE issue_id=$1 AND field='description'", id).Scan(&at, &value); err != nil {
		t.Fatal(err)
	}
	if at.Before(old) || !at.Equal(c.Now()) || value != "<p>Second</p>" {
		t.Fatal("activity time moved before lock", at, old, value)
	}
	// Equal times still use descending id to select the last activity. Two
	// preceding title rows have the opposite insertion/id order.
	high, low := uuid.MustParse("ffffffff-ffff-7fff-bfff-ffffffffffff"), uuid.MustParse("00000000-0000-7000-8000-000000000001")
	for n, actor := range []string{"alice", "bob"} {
		route := newIssueRouteWithClock(t, w.pool, postgres.NewTxManager(w.pool, 2*time.Second), c)
		route.answerWithin(t, w.contract, http.MethodPatch, path, w.ids[actor], fmt.Sprintf(`{"name":"Title %d"}`, n), 200)
		rowID := low
		if n == 0 {
			rowID = high
		}
		if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issue_activities SET id=$2 WHERE issue_id=$1 AND field='name' AND actor_id=$3", id, rowID, w.ids[actor]); err != nil {
			t.Fatal(err)
		}
	}
	route := newIssueRouteWithClock(t, w.pool, postgres.NewTxManager(w.pool, 2*time.Second), c)
	route.answerWithin(t, w.contract, http.MethodPatch, path, w.ids["bob"], `{"description_html":"<p>After titles</p>"}`, 200)
	// Either title interrupts coalescing; no earlier description can be merged.
	if rows := queryIDs(t, w.pool, "SELECT id FROM issue_activities WHERE issue_id=$1 AND field='description'", id); len(rows) != 2 {
		t.Fatal("same-time title did not interrupt description merge", rows)
	}
}
