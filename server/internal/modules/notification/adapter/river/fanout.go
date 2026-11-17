package riveradapter

import (
	"context"
	"log/slog"
	"time"
	"uuid"

	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// FanoutKind identifies the fan-out of an event's notifications (M7 P1b
// spec 2.9).
const FanoutKind = "notification.fanout"

// FanoutMaxAttempts bounds the retries of a fan-out, a lock timeout's among
// them (spec 2.9).
const FanoutMaxAttempts = 5

// fanoutTimeout bounds a fan-out's run (spec 2.9).
const fanoutTimeout = 30 * time.Second

// FanoutArgs are the event a fan-out sends the notifications of, as the
// subscriber hands it over (app.NewNotifyOnIssueEvents): its id, type,
// moment, workspace, project, actor, nil for nerve's own write, entity, and
// every change at its position, the HTML of a description and a comment
// blanked, so river_job keeps none (R4). The work item is the one the event
// names (domain.WorkItem of the rebuilt event).
type FanoutArgs struct {
	EventID     uuid.UUID      `json:"event_id"`
	Type        string         `json:"type"`
	OccurredAt  time.Time      `json:"occurred_at"`
	WorkspaceID uuid.UUID      `json:"workspace_id"`
	ProjectID   uuid.UUID      `json:"project_id"`
	ActorID     *uuid.UUID     `json:"actor_id"`
	EntityKind  string         `json:"entity_kind"`
	EntityID    uuid.UUID      `json:"entity_id"`
	Changes     []FanoutChange `json:"changes"`
}

// FanoutChange is a change of FanoutArgs: a nil value or id is left out, so
// a value "" stays apart from none.
type FanoutChange struct {
	Field string     `json:"field"`
	Old   *string    `json:"old,omitempty"`
	New   *string    `json:"new,omitempty"`
	OldID *uuid.UUID `json:"old_id,omitempty"`
	NewID *uuid.UUID `json:"new_id,omitempty"`
}

func (FanoutArgs) Kind() string { return FanoutKind }

// InsertOpts keeps the fan-outs on the default queue, each with its own
// attempts, without deduplication: each event is its own.
func (FanoutArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: river.QueueDefault, MaxAttempts: FanoutMaxAttempts}
}

// fanoutArgs are the arguments of the event e.
func fanoutArgs(e shared.Event) FanoutArgs {
	changes := make([]FanoutChange, len(e.Changes))
	for n, c := range e.Changes {
		changes[n] = FanoutChange{Field: c.Field, Old: c.Old, New: c.New, OldID: c.OldID, NewID: c.NewID}
	}
	return FanoutArgs{EventID: e.ID, Type: e.Type, OccurredAt: e.OccurredAt, WorkspaceID: e.WorkspaceID, ProjectID: e.ProjectID,
		ActorID: e.ActorID, EntityKind: e.Entity.Kind, EntityID: e.Entity.ID, Changes: changes}
}

// event is the event a carries, as it was handed over.
func (a FanoutArgs) event() shared.Event {
	changes := make([]shared.Change, len(a.Changes))
	for n, c := range a.Changes {
		changes[n] = shared.Change{Field: c.Field, Old: c.Old, New: c.New, OldID: c.OldID, NewID: c.NewID}
	}
	return shared.Event{ID: a.EventID, Type: a.Type, OccurredAt: a.OccurredAt, WorkspaceID: a.WorkspaceID, ProjectID: a.ProjectID,
		ActorID: a.ActorID, Entity: shared.EntityRef{Kind: a.EntityKind, ID: a.EntityID}, Changes: changes}
}

// Fanouts adapt the subscriber's enqueueing to a transactional River insert
// (app.Fanouts).
type Fanouts struct{ inserter Inserter }

func NewFanouts(i Inserter) *Fanouts { return &Fanouts{inserter: i} }

func (f *Fanouts) Enqueue(ctx context.Context, e shared.Event) error {
	args := fanoutArgs(e)
	opts := args.InsertOpts()
	return f.inserter.InsertTx(ctx, args, &opts)
}

// FanoutUseCase keeps River out of the application layer.
type FanoutUseCase interface {
	Execute(ctx context.Context, e shared.Event) error
}

// FanoutWorker runs the fan-out of the event a job carries, and logs the
// last failed attempt.
type FanoutWorker struct {
	river.WorkerDefaults[FanoutArgs]
	uc     FanoutUseCase
	logger *slog.Logger
}

func NewFanoutWorker(uc FanoutUseCase, logger *slog.Logger) *FanoutWorker {
	return &FanoutWorker{uc: uc, logger: logger}
}

func (w *FanoutWorker) Work(ctx context.Context, job *river.Job[FanoutArgs]) error {
	err := w.uc.Execute(ctx, job.Args.event())
	if err != nil && job.Attempt >= job.MaxAttempts {
		w.logger.ErrorContext(ctx, "notification fan-out exhausted", slog.Int64("job_id", job.ID), slog.String("event_id", job.Args.EventID.String()),
			slog.String("type", job.Args.Type), slog.Any("error", err))
	}
	return err
}

func (w *FanoutWorker) Timeout(*river.Job[FanoutArgs]) time.Duration {
	return fanoutTimeout
}

// FanoutJob declares the request-driven worker.
func FanoutJob(uc FanoutUseCase, logger *slog.Logger) jobs.Job {
	return jobs.Job{Worker: jobs.Register(NewFanoutWorker(uc, logger))}
}
