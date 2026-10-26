package app_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/content"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func newComment() (*app.CreateIssueComment, *world, *fakeComments, uuid.UUID) {
	_, w, id := newUpdate()
	f := &fakeComments{fakeRelations: &fakeRelations{fakeIssues: w.issues}}
	return app.NewCreateIssueComment(app.CreateIssueCommentDeps{Events: discardEvents{}, Locks: w.locks, Comments: f, Members: w.project, Sanitizer: content.NewSanitizer(), Clock: clockAt{at: clockNow, log: w.log}}), w, f, id
}

func TestCreateComment(t *testing.T) {
	for _, tt := range []struct {
		name, html string
		code       string
	}{
		{"blank", " <p> </p>", shared.FieldRequired}, {"script", "<script>alert(1)</script>", shared.FieldRequired},
		{"image", `<img src="https://example.com/a.png">`, ""},
		{"mention", `<mention-component entity_name="user_mention" entity_identifier="` + bob.String() + `">Bob</mention-component>`, ""},
		{"unicode boundary", strings.Repeat("中", 500000), ""}, {"unicode over boundary", strings.Repeat("中", 500001), shared.FieldTooLong},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, f, id := newComment()
			got, err := uc.Execute(as(alice), id, tt.html)
			if tt.code != "" {
				var se *shared.Error
				if !errors.As(err, &se) || len(se.Fields) != 1 || se.Fields[0].Field != "comment_html" || se.Fields[0].Code != tt.code {
					t.Fatalf("error = %#v", err)
				}
				if len(w.log.calls) != 0 {
					t.Fatal(w.log.calls)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.ID == (uuid.UUID{}) || got.IssueID != id || got.ActorID != alice || got.EditedAt != nil || got.HTML != content.NewSanitizer().Clean(tt.html).HTML || !got.CreatedAt.Equal(clockNow) || !got.UpdatedAt.Equal(clockNow) || got.Reactions == nil || len(got.Reactions) != 0 {
				t.Fatalf("creation shape: %+v", got)
			}
			if f.inserted == nil || f.inserted.By != alice || !f.inserted.At.Equal(clockNow) || f.marked == nil || !f.marked.At.Equal(clockNow) || f.subscribed == nil || !f.subscribed.At.Equal(clockNow) {
				t.Fatal("stamps")
			}
			if w.issues.activities != nil {
				t.Fatal("comment created activity")
			}
		})
	}
}

func TestCreateCommentSubscribesOnlyVisibleMembersAndWritesInOrder(t *testing.T) {
	uc, w, f, id := newComment()
	w.project.mentionVisibility = []app.MemberVisibility{{ID: gus, Role: shared.RoleGuest, OwnOnly: true}, {ID: carol, Role: shared.RoleGuest, OwnOnly: true}, {ID: bob, Role: shared.RoleMember}, {ID: dave, Role: shared.RoleGuest}}
	html := "<p>Hi</p>"
	for _, a := range []uuid.UUID{gus, carol, bob, bob, dave, out} {
		html += `<mention-component entity_name="user_mention" entity_identifier="` + a.String() + `">Hi</mention-component>`
	}
	if _, err := uc.Execute(as(alice), id, html); err != nil {
		t.Fatal(err)
	}
	if f.subscribed == nil {
		t.Fatal("no subscribers written")
	}
	if !slices.Equal(f.subscribed.Targets, []uuid.UUID{bob, alice, carol, dave}) {
		t.Fatalf("subscription visibility: %v", f.subscribed.Targets)
	}
	want := append(lockedAs(domain.ActionCreateComment, id), line("MentionMembers", web, []uuid.UUID{bob, carol, gus, out, dave}), "Now", line("MarkChanged", []uuid.UUID{id}), line("InsertSubscribers", []uuid.UUID{bob, alice, carol, dave}), line("InsertComment", id))
	if !slices.Equal(w.log.calls, want) {
		t.Fatalf("calls = %q; want %q", w.log.calls, want)
	}
}

func TestCreateCommentStopsAtFailingPort(t *testing.T) {
	for _, port := range []string{"GetIssue", "Authorize", "Lock", "MentionMembers", "MarkChanged", "InsertSubscribers", "InsertComment"} {
		t.Run(port, func(t *testing.T) {
			uc, w, _, id := newComment()
			w.log.fail = port
			_, err := uc.Execute(as(alice), id, `<mention-component entity_name="user_mention" entity_identifier="`+bob.String()+`">Bob</mention-component>`)
			wantStoppedAt(t, w, err, port)
		})
	}
}

func TestListComments(t *testing.T) {
	for _, limit := range []int{1, 50, 100} {
		t.Run(line("limit", limit), func(t *testing.T) {
			_, w, f, id := newComment()
			f.listed = []domain.Comment{{ID: bob, CreatedAt: earlier}, {ID: alice, CreatedAt: clockNow}}
			uc := app.NewListIssueComments(app.ListIssueCommentsDeps{Locks: w.locks, Comments: f, ReadTx: fakeTx{log: w.log}})
			page, err := uc.Execute(as(alice), id, &limit, nil)
			if err != nil {
				t.Fatal(err)
			}
			if f.limit != limit+1 {
				t.Fatalf("query limit = %d", f.limit)
			}
			if limit == 1 {
				if len(page.Data) != 1 || page.NextCursor == "" {
					t.Fatal("limit+1 page")
				}
				var c domain.CommentCursor
				if err := shared.DecodeCursor(page.NextCursor, &c); err != nil || c.ID != bob {
					t.Fatal(c, err)
				}
			} else if len(page.Data) != 2 || page.NextCursor != "" {
				t.Fatal(page)
			}
			if !slices.Equal(w.log.calls, []string{"Begin", line("GetIssue", id), line("Authorize", alice, domain.ActionRead, acme, web), line("Comments", id), line("CommentReactions", f.reactionIDs)}) {
				t.Fatal(w.log.calls)
			}
		})
	}
}

func TestListCommentsValidatesCursorBeforeLimitAndDoesNotAuthorizeCursor(t *testing.T) {
	_, w, f, id := newComment()
	uc := app.NewListIssueComments(app.ListIssueCommentsDeps{Locks: w.locks, Comments: f, ReadTx: fakeTx{log: w.log}})
	bad, zero := "bad", 0
	_, err := uc.Execute(as(alice), id, &zero, &bad)
	wantRefused(t, err, shared.InvalidCursor())
	if len(w.log.calls) != 0 {
		t.Fatal(w.log.calls)
	}
	cursor, _ := shared.EncodeCursor(domain.CommentCursor{CreatedAt: earlier, ID: uuid.NewV7()})
	w.auth.answers = []error{shared.ErrNotVisible}
	_, err = uc.Execute(as(alice), id, nil, &cursor)
	wantRefused(t, err, domain.ErrIssueNotFound)
	if len(w.log.calls) != 3 {
		t.Fatal(w.log.calls)
	}
}

func TestListCommentsDefaultAndPortFailures(t *testing.T) {
	_, w, f, id := newComment()
	uc := app.NewListIssueComments(app.ListIssueCommentsDeps{Locks: w.locks, Comments: f, ReadTx: fakeTx{log: w.log}})
	page, err := uc.Execute(as(alice), id, nil, nil)
	if err != nil || f.limit != 51 || page.Data == nil || len(page.Data) != 0 || page.NextCursor != "" {
		t.Fatal(page, err, f.limit)
	}
	for _, port := range []string{"GetIssue", "Authorize", "Comments"} {
		t.Run(port, func(t *testing.T) {
			_, w, f, id := newComment()
			w.log.fail = port
			uc := app.NewListIssueComments(app.ListIssueCommentsDeps{Locks: w.locks, Comments: f, ReadTx: fakeTx{log: w.log}})
			_, err := uc.Execute(as(alice), id, nil, nil)
			wantStoppedAt(t, w, err, port)
		})
	}
}

func TestCreateCommentHiddenGuestNeverEntersGate(t *testing.T) {
	_, w, f, id := newComment()
	w.auth.grants = []shared.Grant{ownOnly}
	w.auth.answers = []error{shared.Forbidden()}
	locks := app.NewLocks(w.project, w.issues, w.auth, fakeGate{log: w.log}, fakeTx{log: w.log})
	uc := app.NewCreateIssueComment(app.CreateIssueCommentDeps{Events: discardEvents{}, Locks: locks, Comments: f, Members: w.project, Sanitizer: content.NewSanitizer(), Clock: clockAt{at: clockNow, log: w.log}})
	_, err := uc.Execute(as(alice), id, "<p>Hidden</p>")
	wantRefused(t, err, domain.ErrIssueNotFound)
	if len(w.log.calls) != 2 || f.inserted != nil || f.subscribed != nil || f.marked != nil {
		t.Fatal(w.log.calls)
	}
}
