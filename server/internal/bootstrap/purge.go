package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"
	"uuid"
)

type purgePageCursor struct {
	Table string
	After uuid.UUID
}
type purgePageProgress struct {
	Cursor                    purgePageCursor
	Scanned, Deleted, Blocked int
	Done                      bool
}
type Purger interface {
	Page(context.Context, time.Time, int, purgePageCursor) (purgePageProgress, error)
}
type purgeModule struct {
	name string
	rows Purger
}
type purgeClock interface{ Now() time.Time }
type purgeCursor struct {
	Version int       `json:"version"`
	Module  string    `json:"module"`
	Table   string    `json:"table"`
	After   uuid.UUID `json:"after"`
	Cutoff  time.Time `json:"cutoff"`
}
type purgeResult struct {
	Cursor   purgeCursor `json:"cursor"`
	Scanned  int         `json:"scanned"`
	Deleted  int         `json:"deleted"`
	Blocked  int         `json:"blocked"`
	Complete bool        `json:"complete"`
}
type purgeDeleted struct {
	modules    []purgeModule
	clock      purgeClock
	runtimeNow func() time.Time
	after      time.Duration
	batch      int
	budget     time.Duration
}

func newPurgeDeleted(modules []purgeModule, clock purgeClock, runtimeNow func() time.Time, after time.Duration, batch int, budget time.Duration) *purgeDeleted {
	return &purgeDeleted{modules: slices.Clone(modules), clock: clock, runtimeNow: runtimeNow, after: after, batch: batch, budget: budget}
}

// Execute preserves the cycle cutoff and returns only safely committed page
// progress. Its runtime budget never uses the business clock.
func (p *purgeDeleted) Execute(ctx context.Context, cursor purgeCursor) (purgeResult, error) {
	result := purgeResult{Cursor: cursor}
	if p.budget <= 0 || len(p.modules) == 0 {
		return result, errors.New("invalid purge worker configuration")
	}
	if cursor.Version == 0 {
		if cursor != (purgeCursor{}) {
			return result, errors.New("uninitialized purge cursor has state")
		}
		cursor = purgeCursor{Version: 1, Module: p.modules[0].name, Cutoff: p.clock.Now().Add(-p.after).UTC()}
	}
	index, err := p.cursorModule(cursor)
	if err != nil {
		return result, err
	}
	result.Cursor = cursor
	until := p.runtimeNow().Add(p.budget)
	workctx, cancel := context.WithTimeout(ctx, p.budget)
	defer cancel()
	for {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if !p.runtimeNow().Before(until) || workctx.Err() != nil {
			return result, nil
		}
		page, err := p.modules[index].rows.Page(workctx, cursor.Cutoff, p.batch, purgePageCursor{Table: cursor.Table, After: cursor.After})
		cursor.Table, cursor.After = page.Cursor.Table, page.Cursor.After
		result.Cursor = cursor
		result.Scanned += page.Scanned
		result.Deleted += page.Deleted
		result.Blocked += page.Blocked
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) && workctx.Err() == context.DeadlineExceeded && ctx.Err() == nil {
				return result, nil
			}
			return result, err
		}
		if page.Done {
			index++
			if index == len(p.modules) {
				result.Complete = true
				return result, nil
			}
			cursor.Module, cursor.Table, cursor.After = p.modules[index].name, "", uuid.Nil()
			result.Cursor = cursor
		}
	}
}

func (p *purgeDeleted) cursorModule(cursor purgeCursor) (int, error) {
	if cursor.Version != 1 || cursor.Cutoff.IsZero() || (cursor.Table == "" && cursor.After != uuid.Nil()) {
		return -1, errors.New("invalid purge cursor version or position")
	}
	index := slices.IndexFunc(p.modules, func(m purgeModule) bool { return m.name == cursor.Module })
	if index < 0 {
		return -1, fmt.Errorf("unknown purge cursor module %q", cursor.Module)
	}
	return index, nil
}
