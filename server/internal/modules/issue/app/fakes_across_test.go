package app_test

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// anys are ids as a logged call's arguments, each its own.
func anys(ids []uuid.UUID) []any {
	out := make([]any, len(ids))
	for n, id := range ids {
		out[n] = id
	}
	return out
}

// LockMany logs the lock, each project its own argument, and answers
// those of ids the fake has locked: its project, as Lock answers it, and
// those of across, as they are (fakeProject).
func (f *fakeProject) LockMany(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]app.LockedProject, error) {
	if f.waited != nil {
		f.waited()
	}
	if err := f.log.add(ctx, "LockMany", anys(ids)...); err != nil {
		return nil, err
	}
	locked := map[uuid.UUID]app.LockedProject{}
	for _, id := range ids {
		if p, ok := f.across[id]; ok {
			locked[id] = p
		} else if id == f.id && f.locked != nil {
			locked[id] = *f.locked
		}
	}
	return locked, nil
}
