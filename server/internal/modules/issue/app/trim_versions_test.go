package app_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

type trimWorld struct {
	*fakeProject
	projects  []uuid.UUID
	locked    map[uuid.UUID]app.LockedProject
	counts    map[uuid.UUID]int
	committed []uuid.UUID
	fail      uuid.UUID
	commitErr error
}

func trimSetup() (*trimWorld, app.TrimVersionsDeps) {
	f := &trimWorld{fakeProject: &fakeProject{log: &callLog{}}, projects: []uuid.UUID{web, acme}, locked: map[uuid.UUID]app.LockedProject{web: {Archived: true}, acme: {}}, counts: map[uuid.UUID]int{web: 3, acme: 5}}
	return f, app.TrimVersionsDeps{Projects: f, Gate: f, Rows: f, Tx: f}
}
func (f *trimWorld) UndeletedProjectIDs(ctx context.Context) ([]uuid.UUID, error) {
	return slices.Clone(f.projects), f.log.add(ctx, "Scan")
}
func (f *trimWorld) Lock(ctx context.Context, id uuid.UUID) (app.LockedProject, bool, error) {
	if f.waited != nil {
		f.waited()
	}
	p, ok := f.locked[id]
	return p, ok, f.log.add(ctx, "Lock", id)
}
func (f *trimWorld) TrimDescriptionVersions(ctx context.Context, id uuid.UUID) (int, error) {
	if err := f.log.add(ctx, "Trim", id); err != nil {
		return 0, err
	}
	if id == f.fail {
		return 0, errDisk
	}
	f.committed = append(f.committed, id)
	return f.counts[id], nil
}
func (f *trimWorld) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	if f.log.fail == "Begin" {
		return errDisk
	}
	before := slices.Clone(f.committed)
	err := fakeTx{f.log}.WithinTx(ctx, fn)
	if err == nil {
		err = f.commitErr
	}
	if err != nil {
		f.committed = before
		f.log.calls = append(f.log.calls, "Rollback")
	} else {
		f.log.calls = append(f.log.calls, "Commit")
	}
	return err
}

func TestTrimVersionsCommitsPerProject(t *testing.T) {
	for _, failure := range []bool{false, true} {
		t.Run(map[bool]string{false: "all projects including archived", true: "later failure preserves earlier commit"}[failure], func(t *testing.T) {
			f, d := trimSetup()
			wantIDs := slices.Clone(f.projects)
			slices.SortFunc(wantIDs, uuid.UUID.Compare)
			want := 8
			if failure {
				f.fail = wantIDs[1]
				want = f.counts[wantIDs[0]]
			}
			n, err := app.NewTrimVersions(d).Execute(context.Background())
			if n != want || failure && !errors.Is(err, errDisk) || !failure && err != nil {
				t.Fatalf("Execute=%d,%v want %d failure=%v", n, err, want, failure)
			}
			wantCommitted := wantIDs
			if failure {
				wantCommitted = wantIDs[:1]
			}
			if !slices.Equal(f.committed, wantCommitted) {
				t.Fatalf("committed=%v want %v", f.committed, wantCommitted)
			}
			calls := []string{"Scan outside tx"}
			for i, id := range wantIDs {
				end := "Commit"
				if failure && i == 1 {
					end = "Rollback"
				}
				calls = append(calls, "Begin", line("Lock", id), line("Trim", id), end)
			}
			if !slices.Equal(f.log.calls, calls) {
				t.Fatalf("calls=%v want %v", f.log.calls, calls)
			}
		})
	}
}
func TestTrimVersionsSkipsProjectGoneAfterWait(t *testing.T) {
	f, d := trimSetup()
	f.projects = []uuid.UUID{web}
	f.waited = func() { delete(f.locked, web) }
	if n, err := app.NewTrimVersions(d).Execute(context.Background()); n != 0 || err != nil || len(f.committed) != 0 {
		t.Fatal(n, err, f.committed)
	}
	if want := []string{"Scan outside tx", "Begin", line("Lock", web), "Commit"}; !slices.Equal(f.log.calls, want) {
		t.Fatal(f.log.calls)
	}
}
func TestTrimVersionsPropagatesFailures(t *testing.T) {
	for _, step := range []string{"Scan", "Begin", "Lock", "Trim", "Commit", "canceled"} {
		t.Run(step, func(t *testing.T) {
			f, d := trimSetup()
			ctx := context.Background()
			want := errDisk
			switch step {
			case "Commit":
				f.commitErr = errDisk
			case "canceled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
				want = context.Canceled
			default:
				f.log.fail = step
			}
			if n, err := app.NewTrimVersions(d).Execute(ctx); n != 0 || !errors.Is(err, want) {
				t.Fatalf("Execute=%d,%v want 0,%v", n, err, want)
			}
		})
	}
}
