package app

import (
	"context"
	"errors"
	"math"
	"time"
	"uuid"
)

type PurgeCursor struct {
	Table string
	After uuid.UUID
}
type PurgeProgress struct {
	Cursor                    PurgeCursor
	Scanned, Deleted, Blocked int
	Done                      bool
}
type Purger struct{ rows PurgeRows }

func NewPurger(rows PurgeRows) *Purger { return &Purger{rows: rows} }
func (p *Purger) Tables() []string     { return []string{"file_assets"} }
func (p *Purger) Page(ctx context.Context, cutoff time.Time, batch int, cursor PurgeCursor) (PurgeProgress, error) {
	result := PurgeProgress{Cursor: cursor}
	if batch < 1 || batch > math.MaxInt32 {
		return result, errors.New("invalid asset purge batch")
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if cursor.Table == "" && cursor.After == uuid.Nil() {
		cursor.Table = "file_assets"
		result.Cursor = cursor
	}
	if cursor.Table != "file_assets" {
		return result, errors.New("unknown asset purge table")
	}
	page, err := p.rows.PurgeAssets(ctx, cutoff, cursor.After, batch)
	if page.Scanned < 0 || page.Scanned > batch || page.Deleted < 0 || page.Deleted > page.Scanned ||
		(page.Scanned == 0 && page.After != cursor.After) || (page.Scanned > 0 && page.After.Compare(cursor.After) <= 0) {
		return result, errors.New("invalid asset purge progress")
	}
	result.Cursor.After = page.After
	result.Scanned = page.Scanned
	result.Deleted = page.Deleted
	result.Done = err == nil && page.Scanned == 0
	return result, err
}
