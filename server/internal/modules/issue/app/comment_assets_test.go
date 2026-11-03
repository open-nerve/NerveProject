package app_test

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/content"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type commentBindingProbe struct {
	lock func(app.ContentBindRequest) (time.Time, error)
	bind func(app.ContentBindRequest, time.Time) error
}

func (p commentBindingProbe) LockContent(_ context.Context, q app.ContentBindRequest) (time.Time, error) {
	return p.lock(q)
}
func (p commentBindingProbe) BindContent(_ context.Context, q app.ContentBindRequest, at time.Time) error {
	return p.bind(q, at)
}

// This boundary test catches moving the first asset lock after a comment write
// or replacing the lock-returned moment with a second clock read. Real SQL
// atomicity and eligibility are covered through bootstrap's actual HTTP routes.
func TestCommentAssetsLockBeforeWrite(t *testing.T) {
	for _, edit := range []bool{false, true} {
		for _, fail := range []string{"", "lock", "bind"} {
			t.Run(fmt.Sprintf("edit=%t/%s", edit, fail), func(t *testing.T) {
				_, _, w, f := newCommentWrites()
				at := clockNow.Add(time.Hour)
				var locked app.ContentBindRequest
				locks, binds := 0, 0
				p := &recordedEvents{t: t}
				probe := commentBindingProbe{
					lock: func(q app.ContentBindRequest) (time.Time, error) {
						locks++
						locked = q
						if f.inserted != nil || f.updated != nil || f.marked != nil || f.subscribed != nil || slices.Contains(w.log.calls, "Now") {
							t.Error("comment wrote or read clock before complete asset lock")
						}
						if q.ActorID != alice || q.Host.WorkspaceID != acme || q.Host.ProjectID != web || q.Host.IssueID != f.current.IssueID || q.Host.CommentID == uuid.Nil() || q.NewComment == edit || !slices.Equal(q.ImageIDs, []uuid.UUID{bob, carol}) {
							t.Error("wrong locked content scope", q)
						}
						if edit && (q.Host.CommentID != f.current.ID || len(q.AttachmentIDs) != 0) || !edit && !slices.Equal(q.AttachmentIDs, []uuid.UUID{dave}) {
							t.Error("wrong comment preparation set", q)
						}
						if fail == "lock" {
							return time.Time{}, errDisk
						}
						return at, nil
					},
					bind: func(q app.ContentBindRequest, got time.Time) error {
						binds++
						if locks != 1 || !reflect.DeepEqual(q, locked) || !got.Equal(at) {
							t.Error("binding did not reuse locked set and time")
						}
						written := f.inserted
						if edit {
							written = f.updated
						}
						if written == nil || written.Comment.ID != q.Host.CommentID || !written.At.Equal(at) || len(p.rows) != 0 {
							t.Error("binding must follow INSERT/UPDATE and precede event")
						}
						if fail == "bind" {
							return errDisk
						}
						return nil
					},
				}
				p.before = func(e shared.Event) {
					if binds != 1 || !e.OccurredAt.Equal(at) || e.Entity.ID != locked.Host.CommentID {
						t.Error("event preceded content or had another stamp")
					}
				}
				html := fmt.Sprintf(`<p>Images</p><image-component src="%s"></image-component><image-component src="%s"></image-component>`, bob, carol)
				var got domain.Comment
				var err error
				if edit {
					u := app.NewUpdateIssueComment(app.UpdateIssueCommentDeps{ContentBindings: probe, Events: p, Locks: w.locks, Comments: f, Members: w.project, Sanitizer: content.NewSanitizer(), Clock: clockAt{at: earlier, log: w.log}})
					got, err = u.Execute(as(alice), f.current.ID, html)
				} else {
					u := app.NewCreateIssueComment(app.CreateIssueCommentDeps{ContentBindings: probe, Events: p, Locks: w.locks, Comments: f, Members: w.project, Sanitizer: content.NewSanitizer(), Clock: clockAt{at: earlier, log: w.log}})
					got, err = u.Execute(as(alice), f.current.IssueID, domain.NewComment{HTML: html, AttachmentIDs: []uuid.UUID{dave}})
				}
				if fail != "" {
					if err != errDisk || len(p.rows) != 0 {
						t.Fatal("binding failure was swallowed", err, p.rows)
					}
					return
				}
				if err != nil || locks != 1 || binds != 1 || !got.UpdatedAt.Equal(at) || slices.Contains(w.log.calls, "Now") || len(p.rows) != 1 {
					t.Fatal("content write did not use one locked moment", got, err, locks, binds, w.log.calls)
				}
			})
		}
	}
}
