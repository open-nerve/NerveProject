package app_test

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The write use cases' clock stands at clockNow, finer than a stored time:
// the fakes store a time as PostgreSQL's timestamptz does, to the
// microsecond, and answer the row as stored, at now. A use case that
// answered its own instant instead of the row as stored would answer
// clockNow.
var (
	clockNow = time.Date(2026, 9, 29, 10, 0, 0, 123456789, time.UTC)
	now      = stored(clockNow)
)

// stored is t as the database keeps it.
func stored(t time.Time) time.Time {
	return t.Truncate(time.Microsecond)
}

// clockAt stands at an instant, its nanoseconds kept, unlike
// clocktest.Fixed. With a log, each read is logged as "Now" among the
// fakes' calls, so a test sees when the use case reads it.
type clockAt struct {
	at  time.Time
	log *callLog
}

func (c clockAt) Now() time.Time {
	if c.log != nil {
		c.log.calls = append(c.log.calls, "Now")
	}
	return c.at
}

// fakeTx runs fn in a context marked as inside the transaction; the fakes
// record whether each call happened there. commitErr, when set, is the
// commit failing after fn succeeded.
type fakeTx struct {
	calls     int
	commitErr error
}

type inTxKey struct{}

func (f *fakeTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	f.calls++
	if err := fn(context.WithValue(ctx, inTxKey{}, true)); err != nil {
		return err
	}
	return f.commitErr
}

// callLog records the calls of the fakes that share it, in order, each with
// its arguments, and " outside tx" when it ran outside a transaction.
type callLog struct{ calls []string }

func (l *callLog) add(ctx context.Context, format string, args ...any) {
	call := fmt.Sprintf(format, args...)
	if ctx.Value(inTxKey{}) != true {
		call += " outside tx"
	}
	l.calls = append(l.calls, call)
}

// show is *s quoted, or <nil>.
func show(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%q", *s)
}

// grantKey is one (user, workspace) pair of fakeAuthorizer.
type grantKey struct{ user, workspace uuid.UUID }

// fakeAuthorizer answers each (user, workspace) pair its grant, or its
// error, and ErrNotVisible for any other; it logs every call.
type fakeAuthorizer struct {
	log    *callLog
	grants map[grantKey]shared.Grant
	errs   map[grantKey]error
}

func (f *fakeAuthorizer) Authorize(ctx context.Context, actor shared.Actor, action shared.Action, t shared.Target) (shared.Grant, error) {
	f.log.add(ctx, "Authorize %s %s on %s/%s", actor.UserID, action, t.WorkspaceID, t.ProjectID)
	key := grantKey{actor.UserID, t.WorkspaceID}
	if err := f.errs[key]; err != nil {
		return shared.Grant{}, err
	}
	if g, ok := f.grants[key]; ok {
		return g, nil
	}
	return shared.Grant{}, shared.ErrNotVisible
}
