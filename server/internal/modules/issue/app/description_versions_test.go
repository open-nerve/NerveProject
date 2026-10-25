package app_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type versionReads struct {
	log   *callLog
	rows  []domain.DescriptionVersion
	found bool
	limit int
	after *domain.DescriptionVersionCursor
}

func (f *versionReads) DescriptionVersions(ctx context.Context, id uuid.UUID, after *domain.DescriptionVersionCursor, limit int) ([]domain.DescriptionVersion, error) {
	f.limit, f.after = limit, after
	return f.rows, f.log.add(ctx, "DescriptionVersions", id)
}
func (f *versionReads) GetDescriptionVersion(ctx context.Context, id uuid.UUID) (domain.DescriptionVersion, bool, error) {
	err := f.log.add(ctx, "GetDescriptionVersion", id)
	if len(f.rows) == 0 {
		return domain.DescriptionVersion{}, false, err
	}
	return f.rows[0], f.found, err
}
func versionList(w *world, f *versionReads) *app.ListDescriptionVersions {
	return app.NewListDescriptionVersions(app.ListDescriptionVersionsDeps{Locks: w.locks, Versions: f, ReadTx: fakeTx{log: w.log}})
}
func TestDescriptionVersionPages(t *testing.T) {
	for _, limit := range []int{1, 50, 100} {
		t.Run(line("limit", limit), func(t *testing.T) {
			_, w, id := newUpdate()
			f := &versionReads{log: w.log, rows: []domain.DescriptionVersion{{ID: alice, IssueID: id, LastSavedAt: clockNow}, {ID: bob, IssueID: id, LastSavedAt: clockNow}}}
			page, err := versionList(w, f).Execute(as(alice), id, &limit, nil)
			if err != nil {
				t.Fatal(err)
			}
			if f.limit != limit+1 {
				t.Fatalf("query must fetch limit+1: %d", f.limit)
			}
			want := []uuid.UUID{alice, bob}
			if limit == 1 {
				want = want[:1]
			}
			got := []uuid.UUID{}
			for _, v := range page.Data {
				got = append(got, v.ID)
			}
			if !slices.Equal(got, want) {
				t.Fatal("version order", got, want)
			}
			if limit == 1 {
				var c domain.DescriptionVersionCursor
				if page.NextCursor == "" {
					t.Fatal("missing next cursor")
				}
				if err := shared.DecodeCursor(page.NextCursor, &c); err != nil || c.ID != alice || !c.LastSavedAt.Equal(clockNow) {
					t.Fatal(c, err)
				}
			} else if page.NextCursor != "" {
				t.Fatal("full terminal page has cursor")
			}
			wantCalls := []string{"Begin", line("GetIssue", id), line("Authorize", alice, shared.Action("issue_description_version.read"), acme, web), line("DescriptionVersions", id)}
			if !slices.Equal(w.log.calls, wantCalls) {
				t.Fatal("snapshot must include first visibility read without write gate/lock", w.log.calls)
			}
		})
	}
	_, w, id := newUpdate()
	f := &versionReads{log: w.log}
	page, err := versionList(w, f).Execute(as(alice), id, nil, nil)
	if err != nil || f.limit != 51 || page.Data == nil || len(page.Data) != 0 || page.NextCursor != "" {
		t.Fatal("empty page", page, err, f.limit)
	}
}
func TestDescriptionVersionVisibility(t *testing.T) {
	notFound := shared.NewError(shared.KindNotFound, "issue.description_version_not_found", "The description version does not exist, or you cannot see it.")
	for _, tt := range []struct {
		name                     string
		found, hidden, forbidden bool
		want                     error
	}{
		{"member", true, false, false, nil}, {"guest own", true, false, true, shared.Forbidden()}, {"guest other", true, true, true, notFound}, {"admin guest own", true, false, false, nil}, {"admin guest other", true, true, false, notFound}, {"missing", false, false, false, notFound},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, w, id := newUpdate()
			f := &versionReads{log: w.log, found: tt.found, rows: []domain.DescriptionVersion{{ID: carol, IssueID: id, HTML: "<p>Saved</p>"}}}
			if tt.hidden {
				w.auth.grants = []shared.Grant{ownOnly}
			}
			if tt.forbidden {
				w.auth.answers = []error{shared.Forbidden()}
			}
			uc := app.NewGetDescriptionVersion(app.GetDescriptionVersionDeps{Locks: w.locks, Versions: f})
			v, err := uc.Execute(as(alice), carol)
			if tt.want != nil {
				wantRefused(t, err, tt.want)
			} else if err != nil || v.HTML != "<p>Saved</p>" {
				t.Fatal(v, err)
			}
			for _, call := range w.log.calls {
				if call == "Begin" {
					t.Fatal("get entered transaction")
				}
			}
		})
	}
}
func TestDescriptionVersionReadValidationAndFailures(t *testing.T) {
	_, w, id := newUpdate()
	f := &versionReads{log: w.log}
	uc := versionList(w, f)
	bad, zero := "bad", 0
	_, err := uc.Execute(as(alice), id, &zero, &bad)
	wantRefused(t, err, shared.InvalidCursor())
	if len(w.log.calls) != 0 {
		t.Fatal(w.log.calls)
	}
	_, err = uc.Execute(as(alice), id, &zero, nil)
	var se *shared.Error
	if !errors.As(err, &se) || se.Code != "validation_failed" {
		t.Fatal(err)
	}
	for _, port := range []string{"GetIssue", "Authorize", "DescriptionVersions", "GetDescriptionVersion"} {
		t.Run(port, func(t *testing.T) {
			_, w, id := newUpdate()
			w.log.fail = port
			f := &versionReads{log: w.log, found: true, rows: []domain.DescriptionVersion{{ID: carol, IssueID: id}}}
			if port == "GetDescriptionVersion" {
				_, err = app.NewGetDescriptionVersion(app.GetDescriptionVersionDeps{Locks: w.locks, Versions: f}).Execute(as(alice), carol)
			} else {
				_, err = versionList(w, f).Execute(as(alice), id, nil, nil)
			}
			if !errors.Is(err, errDisk) {
				t.Fatalf("port error lost: %v", err)
			}
		})
	}
	cursor, _ := shared.EncodeCursor(domain.DescriptionVersionCursor{LastSavedAt: clockNow, ID: out})
	_, err = uc.Execute(as(alice), id, nil, &cursor)
	if err != nil || f.after == nil || f.after.ID != out {
		t.Fatal("cursor must be a position only", f.after, err)
	}
}

func TestGetDescriptionVersionPreservesParentPortErrors(t *testing.T) {
	for _, port := range []string{"GetIssue", "Authorize"} {
		t.Run(port, func(t *testing.T) {
			_, w, id := newUpdate()
			w.log.fail = port
			f := &versionReads{log: w.log, found: true, rows: []domain.DescriptionVersion{{ID: carol, IssueID: id}}}
			_, err := app.NewGetDescriptionVersion(app.GetDescriptionVersionDeps{Locks: w.locks, Versions: f}).Execute(as(alice), carol)
			if !errors.Is(err, errDisk) {
				t.Fatalf("parent port error swallowed: %v", err)
			}
		})
	}
	_, w, id := newUpdate()
	f := &versionReads{log: w.log, found: true, rows: []domain.DescriptionVersion{{ID: carol, IssueID: uuid.NewV7()}}}
	_, err := app.NewGetDescriptionVersion(app.GetDescriptionVersionDeps{Locks: w.locks, Versions: f}).Execute(as(alice), carol)
	wantRefused(t, err, domain.ErrDescriptionVersionNotFound)
	w.log.calls = nil
	_, err = versionList(w, f).Execute(context.Background(), id, nil, nil)
	wantRefused(t, err, shared.Unauthenticated())
	_, err = app.NewGetDescriptionVersion(app.GetDescriptionVersionDeps{Locks: w.locks, Versions: f}).Execute(context.Background(), carol)
	wantRefused(t, err, shared.Unauthenticated())
	if len(w.log.calls) != 0 {
		t.Fatal("unauthenticated request read ports", w.log.calls)
	}
}
