package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/riverqueue/river"
)

type otherArgs struct{}

func (otherArgs) Kind() string { return "jobs_test.other" }

type emptyArgs struct{}

func (emptyArgs) Kind() string { return "" }

type emptyWorker struct {
	river.WorkerDefaults[emptyArgs]
}

func (*emptyWorker) Work(context.Context, *river.Job[emptyArgs]) error { return nil }

type otherWorker struct {
	river.WorkerDefaults[otherArgs]
}

func (*otherWorker) Work(context.Context, *river.Job[otherArgs]) error { return nil }

func runnerConfig() Config {
	return Config{DefaultWorkers: 10, PeriodicWorkers: 2, ShutdownTimeout: time.Second, Logger: slog.New(slog.DiscardHandler)}
}

// Missing worker checks must be independent: removing either check must let
// its case construct successfully, with another actual worker still present.
func TestJobRegistration(t *testing.T) {
	pool := newPool(t, "postgres://nobody@127.0.0.1:1/nowhere")
	valid := func() Job {
		return probeJob(time.Hour, func(context.Context, *river.Job[probeArgs]) error { return nil })
	}
	periodic := func(args river.JobArgs) *PeriodicSpec {
		return &PeriodicSpec{Args: args, Interval: time.Second, ID: "schedule"}
	}
	var nilWorker *probeWorker
	var nilInterface river.Worker[probeArgs]
	var nilArgs *probeArgs
	type registrationCase struct {
		name   string
		jobs   []Job
		kinds  []string
		change func(*Config)
		want   string
	}
	tests := []registrationCase{
		{name: "valid", jobs: []Job{valid()}, kinds: []string{"jobs_test.probe"}},
		{name: "enqueued only", jobs: []Job{{Worker: Register(&otherWorker{})}}, kinds: []string{"jobs_test.other"}},
		{name: "periodic worker removed", jobs: []Job{{Worker: Register(&otherWorker{})}, {Periodic: periodic(probeArgs{})}}, want: "periodic kind jobs_test.probe has no worker"},
		{name: "enqueued worker removed", jobs: []Job{{Worker: Register(&otherWorker{})}}, kinds: []string{"jobs_test.probe"}, want: "enqueued kind jobs_test.probe has no worker"},
		{name: "wrong register", jobs: []Job{{Worker: Register(&otherWorker{}), Periodic: periodic(probeArgs{})}}, want: "periodic kind jobs_test.probe has no worker"},
		{name: "duplicate worker", jobs: []Job{valid(), valid()}, want: "register a job"},
		{name: "duplicate periodic ID", jobs: []Job{valid(), {Periodic: valid().Periodic}}, want: "duplicate periodic ID"},
		{name: "nil worker", jobs: []Job{{Worker: Register(nilInterface)}}, want: "nil worker"},
		{name: "typed nil worker", jobs: []Job{{Worker: Register(nilWorker)}}, want: "nil worker"},
		{name: "empty descriptor", jobs: []Job{{Worker: &WorkerRegistration{}}}, want: "invalid worker registration"},
		{name: "empty worker kind", jobs: []Job{{Worker: Register(&emptyWorker{})}}, want: "empty worker kind"},
		{name: "nil args", jobs: []Job{{Worker: Register(&otherWorker{}), Periodic: periodic(nil)}}, want: "nil periodic args"},
		{name: "typed nil args", jobs: []Job{{Worker: Register(&otherWorker{}), Periodic: periodic(nilArgs)}}, want: "nil periodic args"},
		{name: "empty args kind", jobs: []Job{{Worker: Register(&otherWorker{}), Periodic: periodic(emptyArgs{})}}, want: "empty periodic kind"},
		{name: "empty periodic ID", jobs: []Job{{Worker: Register(&otherWorker{}), Periodic: &PeriodicSpec{Args: otherArgs{}, Interval: time.Second}}}, want: "empty periodic ID"},
		{name: "subsecond interval", jobs: []Job{{Worker: Register(&otherWorker{}), Periodic: &PeriodicSpec{Args: otherArgs{}, Interval: time.Millisecond, ID: "short"}}}, want: "interval must be at least 1s"},
		{name: "zero interval", jobs: []Job{{Worker: Register(&otherWorker{}), Periodic: &PeriodicSpec{Args: otherArgs{}, ID: "zero"}}}, want: "interval must be at least 1s"},
		{name: "negative interval", jobs: []Job{{Worker: Register(&otherWorker{}), Periodic: &PeriodicSpec{Args: otherArgs{}, Interval: -time.Second, ID: "negative"}}}, want: "interval must be at least 1s"},
	}
	for _, key := range []string{"default", "periodic"} {
		for _, n := range []int{0, -1} {
			tests = append(tests, registrationCase{name: fmt.Sprintf("%s_%d", key, n), jobs: []Job{valid()}, change: func(c *Config) {
				if key == "default" {
					c.DefaultWorkers = n
				} else {
					c.PeriodicWorkers = n
				}
			}, want: key + " workers must be positive"})
		}
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := runnerConfig()
			if tt.change != nil {
				tt.change(&cfg)
			}
			_, err := New(pool, cfg, tt.jobs, tt.kinds)
			if tt.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("New()=%v; want %q", err, tt.want)
			}
		})
	}
}
