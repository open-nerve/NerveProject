package app_test

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/content"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type writableComments struct {
	*fakeComments
	current    domain.Comment
	found      bool
	reads      int
	updated    *app.CommentWrite
	deleted    *app.CommentDeletion
	duringRead func(int)
}

func (f *writableComments) GetComment(ctx context.Context, id uuid.UUID) (domain.Comment, bool, error) {
	f.reads++
	if f.duringRead != nil {
		f.duringRead(f.reads)
	}
	err := f.log.add(ctx, "GetComment", id)
	return f.current, f.found, err
}
func (f *writableComments) UpdateComment(ctx context.Context, w app.CommentWrite) error {
	if err := f.log.add(ctx, "UpdateComment", w.Comment.ID); err != nil {
		return err
	}
	f.updated = &w
	return nil
}
func (f *writableComments) DeleteComment(ctx context.Context, d app.CommentDeletion) error {
	if err := f.log.add(ctx, "DeleteComment", d.ID); err != nil {
		return err
	}
	f.deleted = &d
	return nil
}
func newCommentWrites() (*app.UpdateIssueComment, *app.DeleteIssueComment, *world, *writableComments) {
	_, w, f, id := newComment()
	gated(w, nil)
	c := &writableComments{fakeComments: f, found: true, current: domain.Comment{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, IssueID: id, ActorID: alice, HTML: "<p>Before</p>", Text: "Before", CreatedAt: earlier, UpdatedAt: earlier, Reactions: []domain.CommentReaction{}}}
	return app.NewUpdateIssueComment(app.UpdateIssueCommentDeps{Locks: w.locks, Comments: c, Members: w.project, Sanitizer: content.NewSanitizer(), Clock: clockAt{at: clockNow, log: w.log}}), app.NewDeleteIssueComment(app.DeleteIssueCommentDeps{Locks: w.locks, Comments: c, Clock: clockAt{at: clockNow, log: w.log}}), w, c
}
func executeCommentWrite(edit bool, u *app.UpdateIssueComment, d *app.DeleteIssueComment, id uuid.UUID) error {
	if edit {
		_, err := u.Execute(as(alice), id, "<p>After</p>")
		return err
	}
	return d.Execute(as(alice), id)
}
func noCommentResources(t *testing.T, w *world) {
	t.Helper()
	for _, c := range w.log.calls {
		for _, s := range []string{"Enter", "Begin", "Now", "Lock", "Leave"} {
			if strings.HasPrefix(c, s) {
				t.Fatal("refusal acquired resources", w.log.calls)
			}
		}
	}
}
func TestCommentWritePermissions(t *testing.T) {
	for _, edit := range []bool{true, false} {
		for _, tt := range []struct {
			name   string
			author uuid.UUID
			grant  shared.Grant
			own    bool
			want   error
		}{
			{"author", alice, shared.Grant{}, false, nil}, {"admin other author", bob, shared.Grant{ProjectAdmin: true}, false, shared.NewError(shared.KindForbidden, "issue.not_author", "Only the author can edit a comment; only the author or a project administrator can delete it.")},
			{"creator is not author", bob, shared.Grant{}, true, shared.NewError(shared.KindForbidden, "issue.not_author", "Only the author can edit a comment; only the author or a project administrator can delete it.")},
			{"admin guest own", bob, adminGuest, true, nil}, {"admin guest hidden", alice, adminGuest, false, shared.NewError(shared.KindNotFound, "issue.comment_not_found", "The comment does not exist, or you cannot see it.")},
		} {
			t.Run(line("edit", edit, tt.name), func(t *testing.T) {
				u, d, w, f := newCommentWrites()
				f.current.ActorID = tt.author
				w.auth.grants = []shared.Grant{tt.grant, tt.grant}
				if tt.own {
					w.issues.issue.By = alice
				}
				want := tt.want
				if tt.name == "admin other author" && !edit {
					want = nil
				}
				if tt.name == "admin guest own" && edit {
					want = shared.NewError(shared.KindForbidden, "issue.not_author", "Only the author can edit a comment; only the author or a project administrator can delete it.")
				}
				err := executeCommentWrite(edit, u, d, f.current.ID)
				if want != nil {
					wantRefused(t, err, want)
					noCommentResources(t, w)
				} else if err != nil || edit && f.updated == nil || !edit && f.deleted == nil {
					t.Fatal(err, "missing write")
				}
			})
		}
	}
}
func TestCommentWritePrecheck(t *testing.T) {
	for _, edit := range []bool{true, false} {
		var missing error
		for _, kind := range []string{"missing", "hidden", "issue missing", "deleted"} {
			u, d, w, f := newCommentWrites()
			switch kind {
			case "missing", "deleted":
				f.found = false
			case "hidden":
				w.auth.grants = []shared.Grant{ownOnly}
			case "issue missing":
				w.issues.issue.ID = uuid.UUID{}
			}
			err := executeCommentWrite(edit, u, d, f.current.ID)
			if missing == nil {
				missing = err
			}
			if err == nil || !reflect.DeepEqual(err, missing) {
				t.Fatal("child existence leaked", kind, err, missing)
			}
			noCommentResources(t, w)
		}
		for _, kind := range []string{"comment deleted", "comment moved", "demoted", "removed", "hidden"} {
			t.Run(line("edit", edit, kind), func(t *testing.T) {
				u, d, w, f := newCommentWrites()
				f.current.ActorID = bob
				w.auth.grants = []shared.Grant{{ProjectAdmin: true}, {ProjectAdmin: true}}
				if edit {
					f.current.ActorID = alice
				}
				w.project.waited = func() {
					switch kind {
					case "comment deleted":
						f.found = false
					case "comment moved":
						f.current.IssueID = uuid.NewV7()
					case "demoted":
						w.auth.grants[1] = shared.Grant{}
						if edit {
							w.auth.answers = []error{nil, shared.Forbidden()}
						}
					case "removed":
						w.auth.answers = []error{nil, shared.ErrNotVisible}
					case "hidden":
						w.auth.grants[1] = ownOnly
					}
				}
				err := executeCommentWrite(edit, u, d, f.current.ID)
				want := error(domain.ErrCommentNotFound)
				if kind == "demoted" {
					want = domain.ErrNotAuthor
					if edit {
						want = shared.Forbidden()
					}
				}
				wantRefused(t, err, want)
				if err == nil || f.updated != nil || f.deleted != nil {
					t.Fatal("locked refusal missing", kind, err)
				}
				for _, c := range w.log.calls {
					if c == "Now" {
						t.Fatal("refusal read clock")
					}
				}
			})
		}
	}
}
func TestUnchangedCommentWritesNothing(t *testing.T) {
	for _, archived := range []string{"", "project", "issue"} {
		u, _, w, f := newCommentWrites()
		if archived == "project" {
			w.project.locked.Archived = true
		}
		if archived == "issue" {
			w.issues.archivedAt = &earlier
		}
		got, err := u.Execute(as(alice), f.current.ID, "<p>Before</p><script>bad()</script>")
		if archived == "" {
			if err != nil || !reflect.DeepEqual(got, f.current) {
				t.Fatal(got, err)
			}
		} else if err == nil {
			t.Fatal("archive accepted no-op")
		}
		if f.updated != nil || f.marked != nil || f.subscribed != nil {
			t.Fatal("no-op wrote rows")
		}
		for _, c := range w.log.calls {
			if c == "Now" {
				t.Fatal("no-op read clock")
			}
		}
	}
}
func TestCommentWriteStampsAndOrder(t *testing.T) {
	for _, edit := range []bool{true, false} {
		u, d, w, f := newCommentWrites()
		err := executeCommentWrite(edit, u, d, f.current.ID)
		if err != nil {
			t.Fatal(err)
		}
		if f.marked == nil || f.marked.By != alice || !f.marked.At.Equal(clockNow) {
			t.Fatal("issue stamp")
		}
		if edit {
			if f.updated == nil || f.updated.Comment.EditedAt == nil || !f.updated.Comment.EditedAt.Equal(clockNow) || !f.updated.Comment.UpdatedAt.Equal(clockNow) || f.updated.Comment.ActorID != alice || !f.updated.Comment.CreatedAt.Equal(earlier) {
				t.Fatal("edit stamp", f.updated)
			}
		} else if f.deleted == nil || f.deleted.By != alice || !f.deleted.At.Equal(clockNow) {
			t.Fatal("delete stamp")
		}
		mark, write := -1, -1
		for n, c := range w.log.calls {
			if strings.HasPrefix(c, "MarkChanged") {
				mark = n
			}
			if strings.HasPrefix(c, "UpdateComment") || strings.HasPrefix(c, "DeleteComment") {
				write = n
			}
		}
		if mark < 0 || write <= mark {
			t.Fatal("table order", w.log.calls)
		}
		if f.reads != 3 {
			t.Fatal("must locate, precheck and re-read", f.reads)
		}
	}
}
func TestCommentWritePortFailures(t *testing.T) {
	for _, edit := range []bool{true, false} {
		ports := []string{"GetComment", "GetIssue", "Authorize", "Lock", "MarkChanged"}
		if edit {
			ports = append(ports, "InsertSubscribers", "UpdateComment")
		} else {
			ports = append(ports, "DeleteComment")
		}
		for _, port := range ports {
			t.Run(line("edit", edit, port), func(t *testing.T) {
				u, d, w, f := newCommentWrites()
				w.log.fail = port
				err := executeCommentWrite(edit, u, d, f.current.ID)
				if len(w.log.calls) > 0 && strings.HasPrefix(w.log.calls[len(w.log.calls)-1], "Leave") {
					want := line("Leave", web)
					n := 0
					for _, c := range w.log.calls {
						if strings.HasPrefix(c, "Leave") {
							n++
						}
					}
					if n != 1 || w.log.calls[len(w.log.calls)-1] != want {
						t.Fatal("gate release", w.log.calls)
					}
					w.log.calls = w.log.calls[:len(w.log.calls)-1]
				}
				wantStoppedAt(t, w, err, port)
			})
		}
	}
}

func TestCommentWriteValidationAndMentionSubscriptions(t *testing.T) {
	for _, html := range []string{"<script>empty()</script>", strings.Repeat("中", 500001)} {
		u, _, w, f := newCommentWrites()
		_, err := u.Execute(as(alice), f.current.ID, html)
		if err == nil || len(w.log.calls) != 0 {
			t.Fatal("validation acquired resources", err, w.log.calls)
		}
	}
	u, _, w, f := newCommentWrites()
	w.project.mentionVisibility = []app.MemberVisibility{{ID: bob, Role: shared.RoleMember}, {ID: gus, Role: shared.RoleGuest, OwnOnly: true}}
	html := `<p>After</p><mention-component entity_name="user_mention" entity_identifier="` + bob.String() + `">Bob</mention-component><mention-component entity_name="user_mention" entity_identifier="` + gus.String() + `">Hidden</mention-component>`
	if _, err := u.Execute(as(alice), f.current.ID, html); err != nil {
		t.Fatal(err)
	}
	if f.subscribed == nil || !reflect.DeepEqual(f.subscribed.Targets, []uuid.UUID{bob, alice}) {
		t.Fatal("edit mention visibility", f.subscribed)
	}
	w.log.fail = "MentionMembers"
	if _, err := u.Execute(as(alice), f.current.ID, html); err != errDisk {
		t.Fatal("mention port error lost", err)
	}
}
