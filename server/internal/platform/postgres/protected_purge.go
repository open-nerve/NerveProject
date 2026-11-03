package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
)

// ProtectedReference identifies one database constraint, never an error text.
type ProtectedReference struct{ Schema, Table, Constraint string }
type PurgePage struct {
	Scanned, Deleted, Blocked int
	After                     uuid.UUID
}
type PurgeQueries interface {
	Candidates(context.Context, time.Time, uuid.UUID, int) ([]uuid.UUID, error)
	Delete(context.Context, time.Time, []uuid.UUID) ([]uuid.UUID, error)
}

func ProtectedForeignKey(err error, references []ProtectedReference) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr == nil || pgErr.Code != "23503" {
		return false
	}
	for _, ref := range references {
		if ref.Schema != "" && ref.Table != "" && ref.Constraint != "" &&
			pgErr.SchemaName == ref.Schema && pgErr.TableName == ref.Table && pgErr.ConstraintName == ref.Constraint {
			return true
		}
	}
	return false
}

// PurgeProtected uses independent statements supplied by an owner adapter.
// Delete must run on the pool: a failed bulk statement rolls back entirely,
// while each successful fallback statement stays committed after a later error.
func PurgeProtected(ctx context.Context, queries PurgeQueries, references []ProtectedReference, cutoff time.Time, after uuid.UUID, limit int) (PurgePage, error) {
	page := PurgePage{After: after}
	if limit < 1 || limit > math.MaxInt32 {
		return page, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", limit)
	}
	if err := ctx.Err(); err != nil {
		return page, err
	}
	ids, err := queries.Candidates(ctx, cutoff, after, limit)
	if err != nil {
		return page, err
	}
	if len(ids) == 0 {
		return page, nil
	}
	if len(ids) > limit {
		return page, errors.New("purge candidate page exceeds limit")
	}
	last := after
	for _, id := range ids {
		if id.Compare(last) <= 0 {
			return page, errors.New("purge candidates must advance in ID order")
		}
		last = id
	}
	deleted, err := queries.Delete(ctx, cutoff, ids)
	if err == nil {
		return PurgePage{Scanned: len(ids), Deleted: len(deleted), After: last}, nil
	}
	if !ProtectedForeignKey(err, references) {
		return page, err
	}
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return page, err
		}
		deleted, err := queries.Delete(ctx, cutoff, []uuid.UUID{id})
		switch {
		case err == nil:
			page.Deleted += len(deleted)
		case ProtectedForeignKey(err, references):
			page.Blocked++
		default:
			return page, err
		}
		page.Scanned++
		page.After = id
	}
	return page, nil
}
