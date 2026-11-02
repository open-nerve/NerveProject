package riveradapter

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

type cleanupCall struct {
	result app.SweepResult
	err    error
	calls  int
	cursor app.SweepCursor
}

func (f *cleanupCall) Execute(_ context.Context, c app.SweepCursor) (app.SweepResult, error) {
	f.calls++
	f.cursor = c
	return f.result, f.err
}

type scopeCall struct {
	cleanupCall
	kind  domain.ScopeKind
	id    uuid.UUID
	stamp app.DeletionStamp
}

func (f *scopeCall) Execute(ctx context.Context, k domain.ScopeKind, id uuid.UUID, s app.DeletionStamp, c app.SweepCursor) (app.SweepResult, error) {
	f.kind, f.id, f.stamp = k, id, s
	return f.cleanupCall.Execute(ctx, c)
}

func TestAssetWorkersRequireDurableOutput(t *testing.T) {
	for _, failed := range []bool{false, true} {
		fault := errors.New("object delete failed")
		c := app.SweepCursor{Version: 1, Stage: "rows", After: uuid.NewV7(), Cutoff: time.Now().UTC(), RuntimeCutoff: time.Now().Add(-24 * time.Hour).UTC()}
		uc := &cleanupCall{result: app.SweepResult{Cursor: c, Scanned: 1}}
		if failed {
			uc.err = fault
		}
		err := NewCleanupWorker(uc, time.Second).Work(t.Context(), &river.Job[CleanupArgs]{JobRow: &rivertype.JobRow{}})
		if err == nil || uc.calls != 1 || uc.cursor != (app.SweepCursor{}) || (failed && !errors.Is(err, fault)) {
			t.Fatalf("cleanup output failure=%v calls=%d cursor=%+v", err, uc.calls, uc.cursor)
		}
		args := DeleteScopeArgs{Scope: domain.ScopeProject, ID: uuid.NewV7(), At: c.Cutoff}
		c.Stage = "scope"
		c.RuntimeCutoff = time.Time{}
		scope := &scopeCall{cleanupCall: cleanupCall{result: app.SweepResult{Cursor: c, Scanned: 1}}}
		if failed {
			scope.err = fault
		}
		err = NewDeleteScopeWorker(scope, time.Second).Work(t.Context(), &river.Job[DeleteScopeArgs]{JobRow: &rivertype.JobRow{}, Args: args})
		if err == nil || scope.calls != 1 || scope.id != args.ID || scope.kind != args.Scope || !scope.stamp.At.Equal(args.At) || (failed && !errors.Is(err, fault)) {
			t.Fatalf("scope output failure=%v calls=%d", err, scope.calls)
		}
	}
}
func TestAssetWorkersRejectInvalidProgress(t *testing.T) {
	for _, raw := range []string{`null`, `{"output":null}`, `{"output":{}}`, `{"output":{"cursor":{"version":2}}}`, `{"output":{"cursor":{"version":1},"extra":true}}`} {
		uc := &cleanupCall{}
		err := NewCleanupWorker(uc, time.Second).Work(t.Context(), &river.Job[CleanupArgs]{JobRow: &rivertype.JobRow{Metadata: []byte(raw)}})
		if err == nil || uc.calls != 0 {
			t.Fatalf("accepted progress %s: %v calls=%d", raw, err, uc.calls)
		}
	}
}
