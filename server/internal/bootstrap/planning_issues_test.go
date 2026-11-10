package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning"
)

// fakePlanningTransfer is issue's PlanningTransfer answering what it holds, recording the context and the transfer
// it was given.
type fakePlanningTransfer struct {
	moved int
	err   error
	ctx   []context.Context
	given []issue.CycleTransfer
}

func (f *fakePlanningTransfer) TransferCycleIssues(ctx context.Context, t issue.CycleTransfer) (int, error) {
	f.ctx, f.given = append(f.ctx, ctx), append(f.given, t)
	return f.moved, f.err
}

// planningTransfer hands issue planning's transfer (M6 P4 spec R2), in the context it came in, which carries
// planning's transaction: every field of it, the project's and the workspace's ids, both cycles' ids and names, the
// states in their order, the account and the moment; issue's count as it came; the error as it came.
func TestPlanningTransferConvertsTheTransfer(t *testing.T) {
	type key struct{}
	ctx := context.WithValue(context.Background(), key{}, "planning's transaction")
	web, acme, sprint, next, todo, doing, bob := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	at := time.Date(2026, 11, 10, 16, 30, 0, 123456000, time.FixedZone("CST", 8*3600))
	tr := planning.Transfer{ProjectID: web, WorkspaceID: acme, From: planning.NamedCycle{ID: sprint, Name: "Sprint 1"},
		To: planning.NamedCycle{ID: next, Name: "Sprint 2"}, StateIDs: []uuid.UUID{doing, todo}, By: bob, At: at}
	fake := &fakePlanningTransfer{moved: 3}
	var p planning.IssueTransfer = planningTransfer{transfer: fake}
	n, err := p.TransferCycleIssues(ctx, tr)
	want := issue.CycleTransfer{ProjectID: web, WorkspaceID: acme, From: issue.Named{ID: sprint, Name: "Sprint 1"},
		To: issue.Named{ID: next, Name: "Sprint 2"}, StateIDs: []uuid.UUID{doing, todo}, By: bob, At: at}
	if err != nil || n != 3 {
		t.Errorf("TransferCycleIssues() = %d, %v; want issue's 3", n, err)
	}
	if !reflect.DeepEqual(fake.given, []issue.CycleTransfer{want}) || len(fake.ctx) != 1 || fake.ctx[0] != ctx {
		t.Errorf("issue was given %+v in %v; want %+v in planning's context", fake.given, fake.ctx, want)
	}

	boom := errors.New("boom")
	fake.moved, fake.err = 0, boom
	if n, err := p.TransferCycleIssues(ctx, tr); !errors.Is(err, boom) || n != 0 {
		t.Errorf("TransferCycleIssues() = %d, %v; want the error alone", n, err)
	}
}
