package app_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The accounts, the workspace, the project and the moment of the views' tests: alice calls, bob owns the views of
// others. Their names are how the calls are logged.
var (
	alice, bob = uuid.NewV7(), uuid.NewV7()
	acme, web  = uuid.NewV7(), uuid.NewV7()
	moment     = time.Date(2026, 11, 20, 9, 30, 0, 123456000, time.UTC)
)

// world is the fakes of the ports of a call on the views, one value for all of them but the transactions and the
// words: Workspaces, Projects, ViewReader, ViewLocker, ViewWriter, the Authorizer and the Clock. calls logs each call,
// in order, with the transaction it is in (begin and end), so a test holds the whole order of a call's reads, locks,
// decisions, clock and writes.
type world struct {
	calls []string
	names map[uuid.UUID]string
	inTx  bool
	// slugs are the undeleted workspaces by slug as read without a lock; slugsLocked, when not nil, as read under the
	// lock; workspaceGone, the workspaces deleted while the lock waited.
	slugs, slugsLocked map[string]uuid.UUID
	workspaceGone      map[uuid.UUID]bool
	// projects are the undeleted projects' facts as read without a lock; underLock, those that differ under the
	// lock, nil for one deleted while it waited.
	projects  map[uuid.UUID]app.ProjectFacts
	underLock map[uuid.UUID]*app.ProjectFacts
	// views are the undeleted views as read without a lock; viewsLocked, those that differ as LockView reads them,
	// nil for one deleted while it waited; listed, what a list answers.
	views       map[uuid.UUID]domain.View
	viewsLocked map[uuid.UUID]*domain.View
	listed      []domain.View
	// decide answers Authorize, told whether the call is in a transaction (under the locks, for a write).
	decide func(action shared.Action, t shared.Target, inTx bool) (shared.Grant, error)
	// inserted and updated are the views InsertView and UpdateView stored, deleted the views DeleteView deleted.
	inserted, updated []insertion
	deleted           []deletion
}

// insertion is a view InsertView or UpdateView stored and its writer.
type insertion struct {
	view domain.View
	by   uuid.UUID
}

// deletion is a view DeleteView deleted, when and by whom.
type deletion struct {
	id, by uuid.UUID
	at     time.Time
}

// member is a project member's grant, which every rule of the views allows.
var member = shared.Grant{WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleMember}

// newWorld is acme, slug "acme", and its project web, whose views are turned on; alice is allowed every action as a
// member.
func newWorld() *world {
	return &world{
		names:    map[uuid.UUID]string{alice: "alice", bob: "bob", acme: "acme", web: "web"},
		slugs:    map[string]uuid.UUID{"acme": acme},
		projects: map[uuid.UUID]app.ProjectFacts{web: {WorkspaceID: acme, IssueViewsView: true}},
		views:    map[uuid.UUID]domain.View{},
		decide: func(shared.Action, shared.Target, bool) (shared.Grant, error) {
			return member, nil
		},
	}
}

func (w *world) name(id uuid.UUID) string {
	if id == (uuid.UUID{}) {
		return "-"
	}
	if n, ok := w.names[id]; ok {
		return n
	}
	return id.String()
}

func (w *world) log(format string, args ...any) {
	w.calls = append(w.calls, fmt.Sprintf(format, args...))
}

func (w *world) WorkspaceBySlug(_ context.Context, slug string) (uuid.UUID, bool, error) {
	w.log("slug %s", slug)
	id, found := w.slugs[slug]
	return id, found, nil
}

func (w *world) ShareWorkspaceBySlug(_ context.Context, slug string) (uuid.UUID, bool, error) {
	w.log("share slug %s", slug)
	slugs := w.slugs
	if w.slugsLocked != nil {
		slugs = w.slugsLocked
	}
	id, found := slugs[slug]
	return id, found, nil
}

func (w *world) ShareWorkspaceByID(_ context.Context, id uuid.UUID) (bool, error) {
	w.log("share workspace %s", w.name(id))
	return !w.workspaceGone[id], nil
}

func (w *world) Facts(_ context.Context, id uuid.UUID) (app.ProjectFacts, bool, error) {
	w.log("facts %s", w.name(id))
	f, found := w.projects[id]
	return f, found, nil
}

func (w *world) Share(_ context.Context, id uuid.UUID) (app.ProjectFacts, bool, error) {
	w.log("share project %s", w.name(id))
	if f, differs := w.underLock[id]; differs {
		if f == nil {
			return app.ProjectFacts{}, false, nil
		}
		return *f, true, nil
	}
	f, found := w.projects[id]
	return f, found, nil
}

func (w *world) Authorize(_ context.Context, actor shared.Actor, action shared.Action, t shared.Target) (shared.Grant, error) {
	w.log("authorize %s %s %s/%s", w.name(actor.UserID), action, w.name(t.WorkspaceID), w.name(t.ProjectID))
	return w.decide(action, t, w.inTx)
}

func (w *world) Now() time.Time {
	w.log("now")
	return moment
}

func (w *world) ProjectViews(_ context.Context, project uuid.UUID, ownedBy *uuid.UUID) ([]domain.View, error) {
	w.log("project views %s owned %s", w.name(project), w.owner(ownedBy))
	return w.listed, nil
}

func (w *world) WorkspaceViews(_ context.Context, workspace uuid.UUID, ownedBy *uuid.UUID) ([]domain.View, error) {
	w.log("workspace views %s owned %s", w.name(workspace), w.owner(ownedBy))
	return w.listed, nil
}

func (w *world) owner(ownedBy *uuid.UUID) string {
	if ownedBy == nil {
		return "any"
	}
	return w.name(*ownedBy)
}

func (w *world) View(_ context.Context, id uuid.UUID) (domain.View, bool, error) {
	w.log("view %s", w.name(id))
	v, found := w.views[id]
	return v, found, nil
}

func (w *world) LockView(_ context.Context, id uuid.UUID) (domain.View, bool, error) {
	w.log("lock view %s", w.name(id))
	if v, differs := w.viewsLocked[id]; differs {
		if v == nil {
			return domain.View{}, false, nil
		}
		return *v, true, nil
	}
	v, found := w.views[id]
	return v, found, nil
}

func (w *world) InsertView(_ context.Context, v domain.View, by uuid.UUID) error {
	w.log("insert")
	w.inserted = append(w.inserted, insertion{v, by})
	return nil
}

func (w *world) UpdateView(_ context.Context, v domain.View, by uuid.UUID) error {
	w.log("update %s by %s", w.name(v.ID), w.name(by))
	w.updated = append(w.updated, insertion{v, by})
	return nil
}

func (w *world) DeleteView(_ context.Context, id uuid.UUID, at time.Time, by uuid.UUID) error {
	w.log("delete %s by %s", w.name(id), w.name(by))
	w.deleted = append(w.deleted, deletion{id: id, by: by, at: at})
	return nil
}

// tx is a fake transaction manager of the world: it logs where its transaction begins and ends, named, and marks
// the calls in it.
type tx struct {
	w    *world
	name string
}

func (t tx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	t.w.log("begin %s", t.name)
	t.w.inTx = true
	err := fn(ctx)
	t.w.inTx = false
	t.w.log("end %s", t.name)
	return err
}

// words are a kind's fake words: they log that they were asked, keep what they were given and refuse with problems.
type words struct {
	w        *world
	kind     string
	problems []shared.FieldError
	filters  map[string][]string
	display  *shared.ListDisplayFilters
}

func (s *words) Validate(filters map[string][]string, display *shared.ListDisplayFilters) []shared.FieldError {
	s.w.log("words %s", s.kind)
	s.filters, s.display = filters, display
	return s.problems
}

// as is ctx with alice, the caller.
func as(t *testing.T) context.Context {
	return shared.WithActor(t.Context(), shared.Actor{UserID: alice})
}

// reads are the world's reads, in its "read" transactions.
func (w *world) reads() app.Reads { return app.NewReads(w, w, w, w, tx{w, "read"}) }

// writes are the world's writes of a view, in its "write" transactions, with the kinds' words.
func (w *world) writes(project, workspace *words) app.WriteDeps {
	return app.WriteDeps{Locks: app.NewLocks(w, w, w, w, w, tx{w, "write"}), Words: app.Words{Project: project, Workspace: workspace}, Views: w,
		Clock: w}
}

// wroteNothing fails t unless the world stored, changed and deleted no view and read no clock.
func wroteNothing(t *testing.T, w *world) {
	t.Helper()
	if len(w.inserted)+len(w.updated)+len(w.deleted) != 0 {
		t.Errorf("wrote: inserted %+v, updated %+v, deleted %+v; want nothing", w.inserted, w.updated, w.deleted)
	}
	for _, call := range w.calls {
		if call == "now" {
			t.Error("the clock was read")
		}
	}
}

// wantCalls fails t unless the world's calls are want, in order.
func wantCalls(t *testing.T, w *world, want ...string) {
	t.Helper()
	if !reflect.DeepEqual(w.calls, want) {
		t.Errorf("calls:\n got %q\nwant %q", w.calls, want)
	}
}

// wantSame fails t unless err is want itself, the one value its code answers with.
func wantSame(t *testing.T, err, want error) {
	t.Helper()
	if err != want {
		t.Errorf("err = %v (%T); want %v itself", err, err, want)
	}
}

// wantForbidden fails t unless err is the Authorizer's 403.
func wantForbidden(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, shared.Forbidden()) {
		t.Errorf("err = %v; want forbidden", err)
	}
}

// fieldsOf is the fields of the 422 err, failing t when it is not one.
func fieldsOf(t *testing.T, err error) []shared.FieldError {
	t.Helper()
	var e *shared.Error
	if !errors.As(err, &e) || e.Code != shared.CodeValidationFailed {
		t.Fatalf("err = %v; want validation_failed", err)
	}
	return e.Fields
}
