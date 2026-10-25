package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type fakeActivityList struct {
	id     uuid.UUID
	limit  *int
	cursor *string
	calls  int
	page   app.ActivityPage
	err    error
}

func (f *fakeActivityList) Execute(_ context.Context, id uuid.UUID, limit *int, cursor *string) (app.ActivityPage, error) {
	f.id, f.limit, f.cursor, f.calls = id, limit, cursor, f.calls+1
	return f.page, f.err
}

var activitiesPath = "/api/v0/issues/" + webSixID.String() + "/activities"

func TestActivitiesHTTP(t *testing.T) {
	for _, query := range []string{"", "?limit=1&cursor=", "?limit=100&cursor=other-position"} {
		f := &fakeActivityList{}
		res, body := do(t, newServer(t, httpadapter.UseCases{ListIssueActivities: f}), request(http.MethodGet, activitiesPath+query, "alice", ""))
		if res.StatusCode != 200 || body != "{\"data\":[],\"next_cursor\":null}\n" || f.id != webSixID {
			t.Fatal(res.StatusCode, body, f)
		}
		if query == "" {
			if f.limit != nil || f.cursor != nil {
				t.Fatal("absence lost")
			}
		} else if f.limit == nil || f.cursor == nil {
			t.Fatal("presence lost")
		}
	}
	field, old, next := "parent", "", "after"
	oldID, newID := bobComment, webSixID
	row := app.ActivityRecord{ID: bobComment, IssueID: webSixID, ProjectID: webID, WorkspaceID: acmeID, CreatedAt: created, UpdatedAt: created, Activity: domain.Activity{Verb: "future_verb", Comment: "secret"}}
	for _, nonnull := range []bool{false, true} {
		if nonnull {
			row.ActorID = &aliceID
			row.Activity.Field = &field
			row.Activity.OldValue = &old
			row.Activity.NewValue = &next
			row.Activity.OldIdentifier = &oldID
			row.Activity.NewIdentifier = &newID
		}
		f := &fakeActivityList{page: app.ActivityPage{Data: []app.ActivityRecord{row}, NextCursor: "next"}}
		res, body := do(t, newServer(t, httpadapter.UseCases{ListIssueActivities: f}), request(http.MethodGet, activitiesPath, "alice", ""))
		var page struct {
			Data []map[string]any
			Next *string `json:"next_cursor"`
		}
		err := json.Unmarshal([]byte(body), &page)
		if err != nil || res.StatusCode != 200 || len(page.Data) != 1 || len(page.Data[0]) != 11 || page.Next == nil || *page.Next != "next" {
			t.Fatal(body, err)
		}
		for _, key := range []string{"id", "issue_id", "project_id", "verb", "field", "old_value", "new_value", "old_identifier", "new_identifier", "actor_id", "created_at"} {
			v, ok := page.Data[0][key]
			if !ok {
				t.Fatal("missing field", key)
			}
			if key == "verb" && v != "future_verb" {
				t.Fatal("verb is not string", v)
			}
		}
		if page.Data[0]["id"] != bobComment.String() || page.Data[0]["issue_id"] != webSixID.String() || page.Data[0]["project_id"] != webID.String() || page.Data[0]["created_at"] != created.Format("2006-01-02T15:04:05.999999999Z07:00") {
			t.Fatal("activity identity mapping", body)
		}
		if nonnull && (page.Data[0]["field"] != field || page.Data[0]["old_value"] != old || page.Data[0]["new_value"] != next || page.Data[0]["old_identifier"] != oldID.String() || page.Data[0]["new_identifier"] != newID.String() || page.Data[0]["actor_id"] != aliceID.String()) {
			t.Fatal("activity value mapping", body)
		}
		for _, key := range []string{"field", "old_value", "new_value", "old_identifier", "new_identifier", "actor_id"} {
			if (page.Data[0][key] != nil) != nonnull {
				t.Fatal("null presence lost", key, body)
			}
		}
	}
}
func TestActivitiesHTTPRefusals(t *testing.T) {
	for _, tt := range []struct {
		err    error
		status int
	}{{shared.InvalidCursor(), 400}, {shared.Invalid(shared.FieldError{Field: "limit", Code: shared.FieldOutOfRange, Message: "out of range"}), 422}, {domain.ErrIssueNotFound, 404}, {shared.Forbidden(), 403}, {errGone, 500}} {
		res, body := do(t, newServer(t, httpadapter.UseCases{ListIssueActivities: &fakeActivityList{err: tt.err}}), request(http.MethodGet, activitiesPath, "alice", ""))
		if res.StatusCode != tt.status {
			t.Fatal(res.StatusCode, body)
		}
	}
}
func TestActivitiesHTTPBinding(t *testing.T) {
	f := &fakeActivityList{}
	h := newServer(t, httpadapter.UseCases{ListIssueActivities: f})
	for _, tt := range []struct {
		path, actor string
		status      int
	}{{activitiesPath, "", 401}, {"/api/v0/issues/bad/activities", "alice", 400}, {activitiesPath + "?limit=abc", "alice", 400}} {
		res, body := do(t, h, request(http.MethodGet, tt.path, tt.actor, ""))
		if res.StatusCode != tt.status {
			t.Fatal(res.StatusCode, body)
		}
	}
	if f.calls != 0 {
		t.Fatal("bad request reached use case")
	}
}
