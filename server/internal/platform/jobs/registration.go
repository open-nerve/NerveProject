package jobs

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/riverqueue/river"
)

// Job declares a worker, a periodic schedule, or both. Schedule integrity is
// checked independently of the workers actually registered with River.
type Job struct {
	Worker   *WorkerRegistration
	Periodic *PeriodicSpec
}

// WorkerRegistration binds a kind and its registration to the same argument
// type. Modules cannot supply metadata separately from the worker being added.
type WorkerRegistration struct {
	kind string
	add  func(*river.Workers) error
	err  error
}

// Register describes a worker. New rejects invalid descriptions before it
// registers any workers or builds the worker client.
func Register[T river.JobArgs](worker river.Worker[T]) *WorkerRegistration {
	r := &WorkerRegistration{}
	if nilValue(worker) {
		r.err = errors.New("nil worker")
		return r
	}
	var args T
	if nilValue(args) {
		r.err = errors.New("nil worker args")
		return r
	}
	r.kind = args.Kind()
	if r.kind == "" {
		r.err = errors.New("empty worker kind")
		return r
	}
	r.add = func(workers *river.Workers) error { return river.AddWorkerSafely(workers, worker) }
	return r
}

// PeriodicSpec is a public schedule description, independent of a worker.
// InsertOpts is passed to River when the schedule inserts a job.
type PeriodicSpec struct {
	Args       river.JobArgs
	Interval   time.Duration
	ID         string
	RunOnStart bool
	InsertOpts *river.InsertOpts
}

func nilValue(v any) bool {
	if v == nil {
		return true
	}
	value := reflect.ValueOf(v)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	}
	return false
}

func registerJobs(definitions []Job, enqueuedKinds []string) (*river.Workers, []*river.PeriodicJob, error) {
	// Validate every descriptor first. A Register(nil) must never look like an
	// intentionally absent worker on a schedule-only declaration.
	for _, j := range definitions {
		if w := j.Worker; w != nil {
			if w.err != nil {
				return nil, nil, fmt.Errorf("register a job: %w", w.err)
			}
			if w.kind == "" || w.add == nil {
				return nil, nil, errors.New("invalid worker registration")
			}
		}
	}
	workers := river.NewWorkers()
	kinds := make(map[string]bool)
	for _, j := range definitions {
		if w := j.Worker; w != nil {
			if err := w.add(workers); err != nil {
				return nil, nil, fmt.Errorf("register a job: %w", err)
			}
			kinds[w.kind] = true
		}
	}
	var periodic []*river.PeriodicJob
	ids := make(map[string]bool)
	for _, j := range definitions {
		if spec := j.Periodic; spec != nil {
			if nilValue(spec.Args) {
				return nil, nil, errors.New("nil periodic args")
			}
			kind := spec.Args.Kind()
			if kind == "" {
				return nil, nil, errors.New("empty periodic kind")
			}
			if !kinds[kind] {
				return nil, nil, fmt.Errorf("periodic kind %s has no worker", kind)
			}
			if spec.Interval < time.Second {
				return nil, nil, errors.New("periodic interval must be at least 1s")
			}
			if spec.ID == "" {
				return nil, nil, errors.New("empty periodic ID")
			}
			if ids[spec.ID] {
				return nil, nil, fmt.Errorf("duplicate periodic ID %s", spec.ID)
			}
			ids[spec.ID] = true
			args, opts := spec.Args, spec.InsertOpts
			periodic = append(periodic, river.NewPeriodicJob(river.PeriodicInterval(spec.Interval), func() (river.JobArgs, *river.InsertOpts) { return args, opts }, &river.PeriodicJobOpts{ID: spec.ID, RunOnStart: spec.RunOnStart}))
		}
	}
	for _, kind := range enqueuedKinds {
		if !kinds[kind] {
			return nil, nil, fmt.Errorf("enqueued kind %s has no worker", kind)
		}
	}
	return workers, periodic, nil
}
