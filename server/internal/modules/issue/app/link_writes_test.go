package app_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type writableLinks struct {
	*fakeLinks
	current          domain.Link
	found            bool
	reads            int
	updated          *app.LinkWrite
	deleted          *app.LinkDeletion
	returnedMetadata domain.LinkMetadata
}

func (f *writableLinks) GetLink(ctx context.Context, id uuid.UUID) (domain.Link, bool, error) {
	f.reads++
	err := f.log.add(ctx, "GetLink", id)
	return f.current, f.found, err
}
func (f *writableLinks) UpdateLink(ctx context.Context, w app.LinkWrite) (domain.Link, error) {
	if err := f.log.add(ctx, "UpdateLink", w.Link.ID); err != nil {
		return domain.Link{}, err
	}
	f.updated = &w
	result := w.Link
	result.Metadata = f.returnedMetadata
	return result, nil
}
func (f *writableLinks) DeleteLink(ctx context.Context, d app.LinkDeletion) error {
	if err := f.log.add(ctx, "DeleteLink", d.ID); err != nil {
		return err
	}
	f.deleted = &d
	return nil
}
func newLinkWrites(t *testing.T) (*app.UpdateIssueLink, *app.DeleteIssueLink, *world, *writableLinks, *recordedEvents) {
	t.Helper()
	_, w, f, p, id := newLinkCreate(t)
	gated(w, nil)
	c := &writableLinks{fakeLinks: f, found: true, current: domain.Link{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, IssueID: id, URL: "http://169.254.169.254/before", Title: evptr("Before"), Metadata: domain.LinkMetadata{"title": evptr("old snapshot")}, CreatedByID: &bob, CreatedAt: earlier, UpdatedAt: earlier}, returnedMetadata: domain.LinkMetadata{"title": evptr("concurrent preview")}}
	return app.NewUpdateIssueLink(app.UpdateIssueLinkDeps{Locks: w.locks, Links: c, Clock: clockAt{at: clockNow, log: w.log}, Events: p}), app.NewDeleteIssueLink(app.DeleteIssueLinkDeps{Locks: w.locks, Links: c, Clock: clockAt{at: clockNow, log: w.log}, Events: p}), w, c, p
}
func TestUpdateIssueLink(t *testing.T) {
	for _, tc := range []struct {
		name     string
		patch    domain.LinkPatch
		changes  []shared.Change
		activity bool
	}{
		{"title null", domain.LinkPatch{SetTitle: true}, []shared.Change{{Field: "title", Old: evptr("Before")}}, false},
		{"title empty", domain.LinkPatch{SetTitle: true, Title: evptr("")}, []shared.Change{{Field: "title", Old: evptr("Before"), New: evptr("")}}, false},
		{"URL title omitted", domain.LinkPatch{URL: evptr("169.254.169.254/after")}, []shared.Change{{Field: "url", Old: evptr("http://169.254.169.254/before"), New: evptr("http://169.254.169.254/after")}}, true},
		{"URL and title", domain.LinkPatch{URL: evptr("169.254.169.254/after"), SetTitle: true, Title: evptr("After")}, []shared.Change{{Field: "url", Old: evptr("http://169.254.169.254/before"), New: evptr("http://169.254.169.254/after")}, {Field: "title", Old: evptr("Before"), New: evptr("After")}}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u, _, w, f, p := newLinkWrites(t)
			old := f.current
			p.before = func(shared.Event) {
				if f.updated == nil || tc.activity && len(w.issues.activities) != 1 {
					t.Fatal("event preceded link/activity write")
				}
			}
			got, err := u.Execute(as(alice), old.ID, tc.patch)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got.Metadata, f.returnedMetadata) || f.updated == nil || f.marked == nil || f.updated.By != alice || !f.updated.At.Equal(clockNow) || !got.UpdatedAt.Equal(clockNow) || !got.CreatedAt.Equal(earlier) || got.CreatedByID == nil || *got.CreatedByID != bob || f.reads != 3 || eventClockReads(w) != 1 {
				t.Fatalf("write/result stamp or locked reads: %+v %v", got, w.log.calls)
			}
			wantCollaborationEvent(t, p, "issue.link_updated", "issue_link", old.ID, tc.changes)
			if tc.activity {
				want := domain.Activity{Field: evptr("link"), Verb: domain.VerbUpdated, OldValue: evptr(old.URL), NewValue: evptr(got.URL), OldIdentifier: &old.ID, NewIdentifier: &old.ID, Comment: "updated a link"}
				if len(w.issues.activities) != 1 || !reflect.DeepEqual(w.issues.activities[0].Activity, want) {
					t.Fatal("URL activity", w.issues.activities)
				}
			} else if len(w.issues.activities) != 0 {
				t.Fatal("title-only inserted URL activity")
			}
			calls := w.log.calls
			mark, write, activity := -1, -1, -1
			for i, c := range calls {
				if c == line("MarkChanged", []uuid.UUID{old.IssueID}) {
					mark = i
				}
				if c == line("UpdateLink", old.ID) {
					write = i
				}
				if c == line("InsertActivities", 1) {
					activity = i
				}
			}
			if mark < 0 || write <= mark || tc.activity && activity <= write {
				t.Fatal("table order", calls)
			}
		})
	}
}
func TestDeleteIssueLink(t *testing.T) {
	_, d, w, f, p := newLinkWrites(t)
	old := f.current
	p.before = func(shared.Event) {
		if f.deleted == nil || len(w.issues.activities) != 1 {
			t.Fatal("event preceded deletion/activity write")
		}
	}
	if err := d.Execute(as(alice), old.ID); err != nil {
		t.Fatal(err)
	}
	if f.deleted == nil || !reflect.DeepEqual(*f.deleted, app.LinkDeletion{ID: old.ID, By: alice, At: clockNow}) || f.marked == nil || f.reads != 3 || eventClockReads(w) != 1 {
		t.Fatal("deletion stamp/reads")
	}
	want := domain.Activity{Field: evptr("link"), Verb: domain.VerbDeleted, OldValue: evptr(old.URL), NewValue: evptr(""), Comment: "deleted the link"}
	if len(w.issues.activities) != 1 || !reflect.DeepEqual(w.issues.activities[0].Activity, want) {
		t.Fatal("deletion activity empty/nil slots", w.issues.activities)
	}
	wantCollaborationEvent(t, p, "issue.link_deleted", "issue_link", old.ID, []shared.Change{{Field: "url", Old: evptr(old.URL)}, {Field: "title", Old: old.Title}})
}
func TestUpdateIssueLinkNoopAndArchiveOrder(t *testing.T) {
	for _, state := range []string{"noop", "project and issue archived", "issue archived"} {
		t.Run(state, func(t *testing.T) {
			u, _, w, f, p := newLinkWrites(t)
			if state != "noop" {
				w.issues.archivedAt = &earlier
			}
			if state == "project and issue archived" {
				w.project.locked.Archived = true
			}
			got, err := u.Execute(as(alice), f.current.ID, domain.LinkPatch{URL: evptr(" 169.254.169.254/before ")})
			switch state {
			case "noop":
				if err != nil || !reflect.DeepEqual(got, f.current) {
					t.Fatal(got, err)
				}
			case "issue archived":
				wantRefused(t, err, domain.ErrIssueArchived)
			default:
				wantRefused(t, err, domain.ErrProjectArchived)
			}
			if f.updated != nil || f.marked != nil || len(p.rows) != 0 || eventClockReads(w) != 0 || slices.ContainsFunc(w.log.calls, func(s string) bool { return s == line("LinkURLExists", f.current.IssueID, f.current.ID, f.current.URL) }) {
				t.Fatal("noop/refusal wrote or queried uniqueness")
			}
		})
	}
}
func TestUpdateIssueLinkAndDeleteIssueLinkTarget(t *testing.T) {
	for _, edit := range []bool{true, false} {
		for _, state := range []string{"missing", "hidden forbidden", "forbidden", "deleted under lock", "moved under lock", "changed under lock", "removed under lock"} {
			t.Run(line("edit", edit, state), func(t *testing.T) {
				u, d, w, f, p := newLinkWrites(t)
				want := error(domain.ErrLinkNotFound)
				switch state {
				case "missing":
					f.found = false
				case "hidden forbidden":
					w.auth.grants = []shared.Grant{ownOnly}
					w.auth.answers = []error{shared.Forbidden()}
				case "forbidden":
					w.auth.answers = []error{shared.Forbidden()}
					want = shared.Forbidden()
				default:
					w.project.waited = func() {
						switch state {
						case "deleted under lock":
							f.found = false
						case "moved under lock":
							f.current.IssueID = uuid.NewV7()
						case "changed under lock":
							f.current.URL = "http://169.254.169.254/locked"
						case "removed under lock":
							w.auth.answers = []error{nil, shared.ErrNotVisible}
						}
					}
				}
				var err error
				if edit {
					_, err = u.Execute(as(alice), f.current.ID, domain.LinkPatch{SetTitle: true, Title: evptr("After")})
				} else {
					err = d.Execute(as(alice), f.current.ID)
				}
				if state == "changed under lock" {
					if err != nil || len(p.rows) != 1 {
						t.Fatal(err)
					}
					if !edit && *p.rows[0].Changes[0].Old != "http://169.254.169.254/locked" {
						t.Fatal("event used stale pre-lock URL")
					}
					return
				}
				wantRefused(t, err, want)
				if state == "missing" || state == "hidden forbidden" || state == "forbidden" {
					noCommentResources(t, w)
				}
				if f.updated != nil || f.deleted != nil || f.marked != nil || len(p.rows) != 0 || eventClockReads(w) != 0 {
					t.Fatal("refused target wrote/published")
				}
			})
		}
	}
}
func TestUpdateIssueLinkAndDeleteIssueLinkPortFailures(t *testing.T) {
	for _, edit := range []bool{true, false} {
		ports := []string{"GetLink", "GetIssue", "Authorize", "Lock", "MarkChanged", "InsertActivities", "Publish"}
		if edit {
			ports = append(ports, "LinkURLExists", "UpdateLink")
		} else {
			ports = append(ports, "DeleteLink")
		}
		for _, port := range ports {
			t.Run(line("edit", edit, port), func(t *testing.T) {
				u, d, w, f, p := newLinkWrites(t)
				if port == "Publish" {
					p.fail = errDisk
				} else {
					w.log.fail = port
				}
				var err error
				if edit {
					_, err = u.Execute(as(alice), f.current.ID, domain.LinkPatch{URL: evptr("169.254.169.254/after")})
				} else {
					err = d.Execute(as(alice), f.current.ID)
				}
				if !errors.Is(err, errDisk) {
					t.Fatal("lost port error", err)
				}
				if port != "Publish" && len(p.rows) != 0 {
					t.Fatal("published after failed port")
				}
			})
		}
	}
}

func TestUpdateIssueLinkValidationBeforeResources(t *testing.T) {
	u, _, w, f, p := newLinkWrites(t)
	_, err := u.Execute(as(alice), f.current.ID, domain.LinkPatch{URL: evptr("ftp://169.254.169.254/x"), SetTitle: true, Title: evptr(strings.Repeat("界", 256))})
	if err == nil || len(w.log.calls) != 0 || len(p.rows) != 0 {
		t.Fatal("domain validation acquired resources", err, w.log.calls)
	}
	want := shared.Invalid(shared.FieldError{Field: "url", Code: shared.FieldNotAllowed, Message: "must be an HTTP or HTTPS URL without credentials, zones or control characters, of at most 2048 characters"}, shared.FieldError{Field: "title", Code: shared.FieldTooLong, Message: "must be at most 255 characters"})
	wantRefused(t, err, want)
}
