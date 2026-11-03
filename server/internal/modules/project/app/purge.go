package app

import (
	"context"
	"fmt"
	"math"
	"slices"
	"time"
	"uuid"
)

type PurgeBatch struct {
	Cutoff time.Time
	After  uuid.UUID
	Limit  int
}
type PurgeCursor struct {
	Table string
	After uuid.UUID
}
type PurgePage struct {
	Scanned, Deleted, Blocked int
	After                     uuid.UUID
}
type PurgeProgress struct {
	Cursor                    PurgeCursor
	Scanned, Deleted, Blocked int
	Done                      bool
}

// PurgeStep binds one owned table to its fixed query and protection protocol.
type PurgeStep struct {
	Table     string
	Protected bool
	Delete    func(context.Context, PurgeBatch) (PurgePage, error)
}
type Purger struct{ steps []PurgeStep }

func NewPurger(steps []PurgeStep) *Purger { return &Purger{steps: slices.Clone(steps)} }
func (p *Purger) Tables() []string {
	var out []string
	for _, s := range p.steps {
		out = append(out, s.Table)
	}
	return out
}
func (p *Purger) ProtectedTables() []string {
	var out []string
	for _, s := range p.steps {
		if s.Protected {
			out = append(out, s.Table)
		}
	}
	return out
}
func (p *Purger) Page(ctx context.Context, cutoff time.Time, limit int, cursor PurgeCursor) (PurgeProgress, error) {
	result := PurgeProgress{Cursor: cursor}
	if limit < 1 || limit > math.MaxInt32 {
		return result, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", limit)
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if cursor.Table == "" {
		if cursor.After != uuid.Nil() {
			return result, fmt.Errorf("purge cursor has ID without table")
		}
		if len(p.steps) == 0 {
			result.Done = true
			return result, nil
		}
		cursor.Table = p.steps[0].Table
		result.Cursor = cursor
	}
	index := slices.IndexFunc(p.steps, func(s PurgeStep) bool { return s.Table == cursor.Table })
	if index < 0 {
		return result, fmt.Errorf("unknown purge cursor table %q", cursor.Table)
	}
	page, err := p.steps[index].Delete(ctx, PurgeBatch{Cutoff: cutoff, After: cursor.After, Limit: limit})
	if page.Scanned < 0 || page.Scanned > limit || page.Deleted < 0 || page.Blocked < 0 || page.Deleted+page.Blocked > page.Scanned ||
		(page.Scanned == 0 && page.After != cursor.After) || (page.Scanned > 0 && page.After.Compare(cursor.After) <= 0) {
		return result, fmt.Errorf("invalid purge progress from %s", cursor.Table)
	}
	result.Cursor.After = page.After
	result.Scanned, result.Deleted, result.Blocked = page.Scanned, page.Deleted, page.Blocked
	if err != nil {
		return result, err
	}
	if page.Scanned == 0 {
		if index+1 == len(p.steps) {
			result.Done = true
		} else {
			result.Cursor = PurgeCursor{Table: p.steps[index+1].Table}
		}
	}
	return result, nil
}
