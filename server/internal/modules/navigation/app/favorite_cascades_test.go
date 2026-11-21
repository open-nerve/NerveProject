package app_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// cascadeCall is a call of the cascades' port: the favorites of a target of its kind it deleted, or of a project, with
// kind "", at a moment by an account.
type cascadeCall struct {
	kind domain.FavoriteKind
	id   uuid.UUID
	at   time.Time
	by   uuid.UUID
}

// cascadeLog is a fake FavoriteCascades: it keeps each call, made in want's context, and answers err.
type cascadeLog struct {
	t     *testing.T
	want  context.Context
	calls []cascadeCall
	err   error
}

func (c *cascadeLog) DeleteFavoritesOfTarget(ctx context.Context, kind domain.FavoriteKind, id uuid.UUID, at time.Time, by uuid.UUID) error {
	if ctx != c.want {
		c.t.Error("the publisher's context replaced")
	}
	c.calls = append(c.calls, cascadeCall{kind, id, at, by})
	return c.err
}

func (c *cascadeLog) DeleteFavoritesOfProject(ctx context.Context, project uuid.UUID, at time.Time, by uuid.UUID) error {
	if ctx != c.want {
		c.t.Error("the publisher's context replaced")
	}
	c.calls = append(c.calls, cascadeCall{"", project, at, by})
	return c.err
}

// The events of the cascades, each as its publisher writes it (P3c spec 2.6): a cycle's and a module's deletion,
// their one change the name before it (planning's deletionEvent), and their archives and a project's, no change.
func cascadeEvents() (events map[string]shared.Event, ws, p, target uuid.UUID, at time.Time, by uuid.UUID) {
	ws, p, target, by = uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	at = time.Date(2026, 11, 22, 9, 0, 0, 123456000, time.UTC)
	event := func(kind, entity string, id uuid.UUID, name string) shared.Event {
		e := shared.Event{ID: uuid.NewV7(), Type: kind, OccurredAt: at, WorkspaceID: ws, ProjectID: p, ActorID: &by, Entity: shared.EntityRef{Kind: entity, ID: id}}
		if name != "" {
			e.Changes = []shared.Change{{Field: "name", Old: &name}}
		}
		return e
	}
	return map[string]shared.Event{
		"planning.cycle_deleted":   event("planning.cycle_deleted", "cycle", target, "Sprint"),
		"planning.module_deleted":  event("planning.module_deleted", "module", target, "Platform"),
		"planning.cycle_archived":  event("planning.cycle_archived", "cycle", target, ""),
		"planning.module_archived": event("planning.module_archived", "module", target, ""),
		"project.archived":         event("project.archived", "project", p, ""),
	}, ws, p, target, at, by
}

// One subscriber, beside the deletion of a deleted project's rows, takes the five events (spec 2.6, DP-15): a cycle's
// or a module's deletion or archive deletes every account's favorites of it, a project's archive every account's
// favorites whose project it is; each in the publisher's context, its transaction, at the event's moment and by its
// actor, the target's id the event's entity's, the project's its project's.
func TestFavoriteCascades(t *testing.T) {
	events, _, p, target, at, by := cascadeEvents()
	for _, tt := range []struct {
		kind string
		want cascadeCall
	}{
		{"planning.cycle_deleted", cascadeCall{domain.FavoriteCycle, target, at, by}},
		{"planning.module_deleted", cascadeCall{domain.FavoriteModule, target, at, by}},
		{"planning.cycle_archived", cascadeCall{domain.FavoriteCycle, target, at, by}},
		{"planning.module_archived", cascadeCall{domain.FavoriteModule, target, at, by}},
		{"project.archived", cascadeCall{"", p, at, by}},
	} {
		t.Run(tt.kind, func(t *testing.T) {
			log := &cascadeLog{t: t, want: t.Context()}
			sub := app.NewFavoriteCascades(log)
			if sub.Name != "navigation_favorite_cascades" || sub.HandleAll != nil || !reflect.DeepEqual(sub.Types, []string{"planning.cycle_deleted",
				"planning.module_deleted", "planning.cycle_archived", "planning.module_archived", "project.archived"}) {
				t.Fatalf("subscriber %q of %v", sub.Name, sub.Types)
			}
			if err := sub.Handle(t.Context(), events[tt.kind]); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(log.calls, []cascadeCall{tt.want}) {
				t.Fatalf("calls %+v; want %+v", log.calls, tt.want)
			}
		})
	}
	failed := errors.New("the favorites' lock timed out")
	for kind, e := range events {
		log := &cascadeLog{t: t, want: t.Context(), err: failed}
		if err := app.NewFavoriteCascades(log).Handle(t.Context(), e); !errors.Is(err, failed) {
			t.Errorf("%s: the port's failure = %v; want it, the publisher's", kind, err)
		}
	}
}

// Each event is validated whole, by its type, before any write (spec 2.6, re-91 m3): an actor, a moment, the
// workspace and the project, and an entity of the type's kind with an id; a deletion's one change its name before
// it, as planning writes it, an archive none; and a project's archive of the event's project. A malformed event is
// refused, and the publisher's write with it, and nothing is deleted.
func TestFavoriteCascadesRefuseAMalformedEvent(t *testing.T) {
	events, _, _, _, _, _ := cascadeEvents()
	name, empty, other := "Sprint", "", uuid.NewV7()
	every := map[string]func(*shared.Event){
		"no actor":        func(e *shared.Event) { e.ActorID = nil },
		"a zero actor":    func(e *shared.Event) { e.ActorID = &uuid.UUID{} },
		"no moment":       func(e *shared.Event) { e.OccurredAt = time.Time{} },
		"no workspace":    func(e *shared.Event) { e.WorkspaceID = uuid.UUID{} },
		"no project":      func(e *shared.Event) { e.ProjectID = uuid.UUID{} },
		"no entity":       func(e *shared.Event) { e.Entity.ID = uuid.UUID{} },
		"an issue":        func(e *shared.Event) { e.Entity.Kind = "issue" },
		"a view":          func(e *shared.Event) { e.Entity.Kind = "view" },
		"an unknown type": func(e *shared.Event) { e.Type = "planning.cycle_updated" },
	}
	deletion := map[string]func(*shared.Event){
		"no change":         func(e *shared.Event) { e.Changes = nil },
		"two changes":       func(e *shared.Event) { e.Changes = append(e.Changes, shared.Change{Field: "name", Old: &name}) },
		"another field":     func(e *shared.Event) { e.Changes = []shared.Change{{Field: "status", Old: &name}} },
		"no old name":       func(e *shared.Event) { e.Changes = []shared.Change{{Field: "name"}} },
		"an empty old name": func(e *shared.Event) { e.Changes = []shared.Change{{Field: "name", Old: &empty}} },
		"a new name":        func(e *shared.Event) { e.Changes = []shared.Change{{Field: "name", Old: &name, New: &name}} },
		"an old id":         func(e *shared.Event) { e.Changes = []shared.Change{{Field: "name", Old: &name, OldID: &other}} },
		"a new id":          func(e *shared.Event) { e.Changes = []shared.Change{{Field: "name", Old: &name, NewID: &other}} },
	}
	archive := map[string]func(*shared.Event){
		"a change":        func(e *shared.Event) { e.Changes = []shared.Change{{Field: "name", Old: &name}} },
		"an empty change": func(e *shared.Event) { e.Changes = []shared.Change{{}} },
	}
	project := map[string]func(*shared.Event){
		"another project": func(e *shared.Event) { e.Entity.ID = other },
		"a cycle":         func(e *shared.Event) { e.Entity.Kind = "cycle" },
	}
	cases := map[string][]map[string]func(*shared.Event){
		"planning.cycle_deleted":   {every, deletion, {"a module": func(e *shared.Event) { e.Entity.Kind = "module" }}},
		"planning.module_deleted":  {every, deletion, {"a cycle": func(e *shared.Event) { e.Entity.Kind = "cycle" }}},
		"planning.cycle_archived":  {every, archive, {"a project": func(e *shared.Event) { e.Entity.Kind = "project" }}},
		"planning.module_archived": {every, archive, {"a cycle": func(e *shared.Event) { e.Entity.Kind = "cycle" }}},
		"project.archived":         {every, archive, project},
	}
	for kind, groups := range cases {
		for _, group := range groups {
			for defect, change := range group {
				t.Run(kind+"/"+defect, func(t *testing.T) {
					e := events[kind]
					e.Changes = append([]shared.Change(nil), e.Changes...)
					change(&e)
					log := &cascadeLog{t: t, want: t.Context()}
					if err := app.NewFavoriteCascades(log).Handle(t.Context(), e); err == nil || len(log.calls) != 0 {
						t.Fatalf("a malformed event: err=%v, calls %+v; want it refused before any write", err, log.calls)
					}
				})
			}
		}
	}
}
