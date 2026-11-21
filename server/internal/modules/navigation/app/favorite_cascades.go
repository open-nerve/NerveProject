package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// cascade is what an event of the cascades takes away (spec 2.6): the
// favorites of a target of kind, a cycle or a module, or of a project,
// whose events are of an entity of that kind; and whether it is a
// deletion, whose one change is its target's name before it, or an
// archive, which has none.
type cascade struct {
	kind     domain.FavoriteKind
	deletion bool
}

// cascades are the events of the cascades by their types (spec 2.6): planning
// declares a cycle's and a module's deletion and archive, project a
// project's archive. Neither module is imported: the types are the strings
// they publish, as the deletion's of a project and of a workspace are
// (NewDeleteRowsOnDeletion).
var cascades = map[string]cascade{
	"planning.cycle_deleted":   {domain.FavoriteCycle, true},
	"planning.module_deleted":  {domain.FavoriteModule, true},
	"planning.cycle_archived":  {domain.FavoriteCycle, false},
	"planning.module_archived": {domain.FavoriteModule, false},
	"project.archived":         {domain.FavoriteProject, false},
}

// NewFavoriteCascades deletes, within the transaction of what takes a
// target away, every account's favorites it takes with it (spec 2.6,
// DP-15): of a cycle or a module deleted or archived, every account's
// favorites of it; of a project archived, every account's favorites whose
// project it is, its own and its cycles', modules' and views'; folders and
// another project's stay. Each at the event's moment and by its actor, the
// target's writer, through favorites, which locks them in id order. The
// event is validated whole, by its type, before any write (cascadeOf); a
// malformed one is refused, and its publisher's write with it. An
// unarchive publishes nothing, and brings no favorite back.
func NewFavoriteCascades(favorites FavoriteCascades) shared.EventSubscriber {
	return shared.EventSubscriber{Name: "navigation_favorite_cascades", Types: []string{"planning.cycle_deleted", "planning.module_deleted",
		"planning.cycle_archived", "planning.module_archived", "project.archived"}, Handle: func(ctx context.Context, e shared.Event) error {
		c, err := cascadeOf(e)
		if err != nil {
			return err
		}
		if c.kind == domain.FavoriteProject {
			return favorites.DeleteFavoritesOfProject(ctx, e.ProjectID, e.OccurredAt, *e.ActorID)
		}
		return favorites.DeleteFavoritesOfTarget(ctx, c.kind, e.Entity.ID, e.OccurredAt, *e.ActorID)
	}}
}

// cascadeOf is the cascade of e, validated whole before any write (spec
// 2.6, re-91 m3): its type one of the cascades'; its actor, its moment, its
// workspace and its project set, and its entity of its type's kind, with an
// id; a deletion's one change its target's name before it, old and not
// empty, nothing else, as planning writes it (its deletionEvent); an
// archive's none; and a project's archive of the event's project. Any
// other shape is an error.
func cascadeOf(e shared.Event) (cascade, error) {
	invalid := errors.New("invalid favorite cascade event")
	c, known := cascades[e.Type]
	if !known || e.ActorID == nil || *e.ActorID == uuid.Nil() || e.OccurredAt.IsZero() || e.WorkspaceID == uuid.Nil() ||
		e.ProjectID == uuid.Nil() || e.Entity.Kind != string(c.kind) || e.Entity.ID == uuid.Nil() {
		return cascade{}, invalid
	}
	switch {
	case c.kind == domain.FavoriteProject && e.Entity.ID != e.ProjectID:
		return cascade{}, invalid
	case !c.deletion && len(e.Changes) != 0:
		return cascade{}, invalid
	case c.deletion && !renamedFrom(e.Changes):
		return cascade{}, invalid
	}
	return c, nil
}

// renamedFrom tells whether changes are a deletion's: one change, its
// target's name before it, old and not empty, and nothing else.
func renamedFrom(changes []shared.Change) bool {
	if len(changes) != 1 {
		return false
	}
	c := changes[0]
	return c.Field == "name" && c.Old != nil && *c.Old != "" && c.New == nil && c.OldID == nil && c.NewID == nil
}
