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

type fakeLinks struct {
	*fakeRelations
	inserted *app.LinkWrite
	rows     []domain.Link
	exists   bool
	checked  string
}

func (f *fakeLinks) LinkURLExists(ctx context.Context, id, exclude uuid.UUID, url string) (bool, error) {
	if err := f.log.add(ctx, "LinkURLExists", id, exclude, url); err != nil {
		return false, err
	}
	f.checked = url
	return f.exists, nil
}
func (f *fakeLinks) InsertLink(ctx context.Context, w app.LinkWrite) error {
	if err := f.log.add(ctx, "InsertLink", w.Link.IssueID); err != nil {
		return err
	}
	f.inserted = &w
	return nil
}
func (f *fakeLinks) Links(ctx context.Context, id uuid.UUID) ([]domain.Link, error) {
	if err := f.log.add(ctx, "Links", id); err != nil {
		return nil, err
	}
	return f.rows, nil
}
func newLinkCreate(t *testing.T) (*app.CreateIssueLink, *world, *fakeLinks, *recordedEvents, uuid.UUID) {
	t.Helper()
	_, w, id := newUpdate()
	f := &fakeLinks{fakeRelations: &fakeRelations{fakeIssues: w.issues}}
	p := &recordedEvents{t: t}
	return app.NewCreateIssueLink(app.CreateIssueLinkDeps{Locks: w.locks, Links: f, Clock: clockAt{at: clockNow, log: w.log}, Events: p}), w, f, p, id
}

// Missing audit, wrong activity slots, event entity/changes or an extra clock
// read must fail even when the inserted link itself is correct.
func TestCreateIssueLink(t *testing.T) {
	for n, title := range []*string{nil, evptr(""), evptr("  User title  ")} {
		t.Run([]string{"null title", "empty title", "title value"}[n], func(t *testing.T) {
			uc, w, f, p, id := newLinkCreate(t)
			p.before = func(shared.Event) {
				if f.inserted == nil || len(w.issues.activities) != 1 {
					t.Fatal("publication preceded business/activity SQL")
				}
			}
			got, err := uc.Execute(as(alice), id, domain.NewLink{URL: "  169.254.169.254/P?q=A  ", Title: title})
			if err != nil {
				t.Fatal(err)
			}
			want := domain.Link{ID: got.ID, WorkspaceID: acme, ProjectID: web, IssueID: id, URL: "http://169.254.169.254/P?q=A", Title: title, Metadata: domain.LinkMetadata{}, CreatedByID: &alice, CreatedAt: clockNow, UpdatedAt: clockNow}
			if got.ID == (uuid.UUID{}) || !reflect.DeepEqual(got, want) {
				t.Fatalf("link=%#v want=%#v", got, want)
			}
			if f.inserted == nil || !reflect.DeepEqual(*f.inserted, app.LinkWrite{Link: want, By: alice, At: clockNow}) || f.marked == nil || !reflect.DeepEqual(*f.marked, app.ChangedRows{IDs: []uuid.UUID{id}, By: alice, At: clockNow}) {
				t.Fatalf("business stamps: %+v %+v", f.inserted, f.marked)
			}
			if len(w.issues.activities) != 1 {
				t.Fatalf("activities=%+v", w.issues.activities)
			}
			row := w.issues.activities[0]
			activity := domain.Activity{Field: evptr("link"), Verb: domain.VerbCreated, NewValue: evptr(want.URL), NewIdentifier: &got.ID, Comment: "created a link"}
			if row.ID == (uuid.UUID{}) || row.ID == got.ID || row.WorkspaceID != acme || row.ProjectID != web || row.IssueID != id || row.ActorID == nil || *row.ActorID != alice || !row.At.Equal(clockNow) || !reflect.DeepEqual(row.Activity, activity) {
				t.Fatalf("activity=%+v", row)
			}
			wantCollaborationEvent(t, p, "issue.link_created", "issue_link", got.ID, []shared.Change{{Field: "url", New: evptr(want.URL)}, {Field: "title", New: title}})
			calls := append(locked(id), line("LinkURLExists", id, uuid.UUID{}, want.URL), "Now", line("MarkChanged", []uuid.UUID{id}), line("InsertLink", id), line("InsertActivities", 1))
			if !slices.Equal(w.log.calls, calls) || eventClockReads(w) != 1 {
				t.Fatalf("calls=%q want=%q", w.log.calls, calls)
			}
		})
	}
}
func TestCreateIssueLinkRefusals(t *testing.T) {
	for _, reason := range []string{"validation", "hidden forbidden", "forbidden", "project archived", "issue archived", "duplicate"} {
		t.Run(reason, func(t *testing.T) {
			uc, w, f, p, id := newLinkCreate(t)
			input := domain.NewLink{URL: "169.254.169.254/a"}
			want := error(domain.ErrIssueNotFound)
			switch reason {
			case "validation":
				input.URL = "ftp://example.com"
				want = shared.Invalid(shared.FieldError{Field: "url", Code: shared.FieldNotAllowed, Message: "must be an HTTP or HTTPS URL without credentials, zones or control characters, of at most 2048 characters"})
			case "hidden forbidden":
				w.auth.grants = []shared.Grant{ownOnly}
				w.auth.answers = []error{shared.Forbidden()}
			case "forbidden":
				w.auth.answers = []error{shared.Forbidden()}
				want = shared.Forbidden()
			case "project archived":
				w.project.locked.Archived = true
				w.issues.archivedAt = &earlier
				want = domain.ErrProjectArchived
			case "issue archived":
				w.issues.archivedAt = &earlier
				want = domain.ErrIssueArchived
			case "duplicate":
				f.exists = true
				want = domain.ErrLinkExists
			}
			_, err := uc.Execute(as(alice), id, input)
			wantRefused(t, err, want)
			if f.inserted != nil || f.marked != nil || len(w.issues.activities) != 0 || len(p.rows) != 0 || eventClockReads(w) != 0 {
				t.Fatal("refusal wrote/published/read clock")
			}
			if reason == "hidden forbidden" || reason == "forbidden" {
				if len(w.log.calls) != 2 {
					t.Fatal("refusal passed precheck", w.log.calls)
				}
			}
			if reason == "validation" && len(w.log.calls) != 0 {
				t.Fatal(w.log.calls)
			}
		})
	}
}
func TestCreateIssueLinkPortFailures(t *testing.T) {
	for _, port := range []string{"GetIssue", "Authorize", "Lock", "LinkURLExists", "MarkChanged", "InsertLink", "InsertActivities", "Publish"} {
		t.Run(port, func(t *testing.T) {
			uc, w, _, p, id := newLinkCreate(t)
			if port == "Publish" {
				p.fail = errDisk
			} else {
				w.log.fail = port
			}
			_, err := uc.Execute(as(alice), id, domain.NewLink{URL: "169.254.169.254/a"})
			if !errors.Is(err, errDisk) {
				t.Fatalf("lost port error: %v", err)
			}
			if port != "Publish" && len(p.rows) != 0 {
				t.Fatal("published after failing port")
			}
		})
	}
}
func TestCreateIssueLinkUnicodeLimit(t *testing.T) {
	for _, extra := range []string{"", "界"} {
		t.Run(line("extra", extra), func(t *testing.T) {
			uc, _, f, _, id := newLinkCreate(t)
			prefix := "http://169.254.169.254/"
			url := prefix + strings.Repeat("界", 2048-len(prefix)) + extra
			got, err := uc.Execute(as(alice), id, domain.NewLink{URL: url})
			if extra == "" {
				if err != nil || got.URL != url || f.checked != url {
					t.Fatal("boundary changed", err)
				}
			} else if err == nil || f.inserted != nil {
				t.Fatal("accepted overlong URL")
			}
		})
	}
}
func TestListIssueLinks(t *testing.T) {
	for _, reason := range []string{"rows", "empty", "own only hidden", "own only own", "guest all", "forbidden", "port"} {
		t.Run(reason, func(t *testing.T) {
			_, w, f, _, id := newLinkCreate(t)
			f.rows = []domain.Link{{ID: bob, URL: "http://169.254.169.254/new"}, {ID: alice, URL: "http://169.254.169.254/old"}}
			switch reason {
			case "empty":
				f.rows = nil
			case "own only hidden":
				w.auth.grants = []shared.Grant{ownOnly}
			case "own only own":
				w.auth.grants = []shared.Grant{ownOnly}
				w.issues.issue.By = alice
			case "guest all":
				w.auth.grants = []shared.Grant{{OwnOnly: false}}
			case "forbidden":
				w.auth.answers = []error{shared.Forbidden()}
			case "port":
				w.log.fail = "Links"
			}
			uc := app.NewListIssueLinks(app.ListIssueLinksDeps{Locks: w.locks, Links: f, ReadTx: fakeTx{log: w.log}})
			got, err := uc.Execute(as(alice), id)
			switch reason {
			case "own only hidden":
				wantRefused(t, err, domain.ErrIssueNotFound)
				return
			case "forbidden":
				wantRefused(t, err, shared.Forbidden())
				return
			case "port":
				if !errors.Is(err, errDisk) {
					t.Fatal(err)
				}
				return
			}
			if err != nil || got == nil || len(got) != len(f.rows) || (len(got) > 0 && !reflect.DeepEqual(got, f.rows)) {
				t.Fatal(got, err)
			}
			if !slices.Equal(w.log.calls, []string{"Begin", line("GetIssue", id), line("Authorize", alice, domain.ActionRead, acme, web), line("Links", id)}) {
				t.Fatal(w.log.calls)
			}
		})
	}
}
