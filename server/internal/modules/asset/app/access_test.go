package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type accessRecording struct {
	log          []string
	hosts        map[uuid.UUID]Host
	rows         []domain.Asset
	fail         string
	failure      error
	inTx, inGate bool
}

func (r *accessRecording) step(name string) error {
	r.log = append(r.log, name)
	if name == r.fail {
		return r.failure
	}
	return nil
}
func (r *accessRecording) Inspect(_ context.Context, _ shared.Actor, q HostRequest) (Host, error) {
	return r.hosts[q.Scope.ID], r.step("inspect:" + q.Scope.ID.String())
}
func (r *accessRecording) Check(_ context.Context, _ shared.Actor, q HostRequest, h Host) error {
	if h != r.hosts[q.Scope.ID] {
		return errors.New("wrong host")
	}
	return r.step("check:" + q.Scope.ID.String())
}
func (r *accessRecording) Enter(_ context.Context, ids ...uuid.UUID) (func(), error) {
	if r.inTx {
		return nil, errors.New("gate entered in transaction")
	}
	if err := r.step(fmt.Sprint("gate:", ids)); err != nil {
		return nil, err
	}
	r.inGate = true
	return func() { r.inGate = false; _ = r.step("leave") }, nil
}
func (r *accessRecording) WithinTx(ctx context.Context, fn func(context.Context) error) (err error) {
	if err = r.step("begin"); err != nil {
		return err
	}
	r.inTx = true
	defer func() { r.inTx = false; _ = r.step("end") }()
	return fn(ctx)
}
func (r *accessRecording) LockCaller(_ context.Context, _ shared.Actor, at time.Time) error {
	if at.Unix() != 11 {
		return errors.New("wrong credential clock")
	}
	return r.step("caller")
}
func (r *accessRecording) LockAssets(_ context.Context, ids []uuid.UUID) ([]domain.Asset, error) {
	return r.rows, r.step(fmt.Sprint("assets:", ids))
}

type accessParents struct {
	r    *accessRecording
	name string
}

func (p accessParents) LockMany(_ context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	return ids, p.r.step(fmt.Sprint(p.name, ":", ids))
}
func (p accessParents) ShareMany(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	return p.LockMany(ctx, ids)
}

type accessClock struct {
	r    *accessRecording
	name string
	at   int64
}

func (c accessClock) Now() time.Time { _ = c.r.step(c.name); return time.Unix(c.at, 0) }
func accessID(i byte) uuid.UUID      { var id uuid.UUID; id[15] = i; return id }
func accessFixture() (*Access, *accessRecording, shared.Actor, []HostRequest, []uuid.UUID) {
	r := &accessRecording{hosts: map[uuid.UUID]Host{}, failure: errors.New("injected dependency failure")}
	actor := shared.Actor{UserID: accessID(20)}
	requests := []HostRequest{{Scope: domain.Scope{Kind: domain.ScopeDraft, ID: accessID(9)}, Intent: ReadAsset}, {Scope: domain.Scope{Kind: domain.ScopeDraft, ID: accessID(2)}, Intent: PrepareAsset}}
	r.hosts[accessID(9)] = Host{Scope: requests[0].Scope, WorkspaceID: accessID(8), ProjectID: accessID(6), DraftID: accessID(9)}
	r.hosts[accessID(2)] = Host{Scope: requests[1].Scope, WorkspaceID: accessID(1), ProjectID: accessID(3), DraftID: accessID(2)}
	ids := []uuid.UUID{accessID(12), accessID(10), accessID(12)}
	for _, id := range []uuid.UUID{accessID(10), accessID(12)} {
		r.rows = append(r.rows, domain.Asset{ID: id, Placement: domain.Placement{UserID: actor.UserID}})
	}
	a := NewAccess(AccessDeps{Hosts: r, Workspaces: accessParents{r, "W"}, Projects: accessParents{r, "P"}, Drafts: accessParents{r, "D"}, Assets: r, Caller: r, Gate: r, Tx: r, CredentialClock: accessClock{r, "credential-clock", 11}, Clock: accessClock{r, "audit-clock", 29}})
	return a, r, actor, requests, ids
}

func TestAssetAccessOrdering(t *testing.T) {
	a, r, actor, requests, ids := accessFixture()
	err := a.Within(t.Context(), actor, requests, ids, func(_ context.Context, h AccessHeld) error {
		if !r.inTx || !r.inGate || h.At.Unix() != 29 || len(h.Assets) != 2 || len(h.Hosts) != 2 {
			t.Fatalf("wrong held state %+v", h)
		}
		return r.step("write")
	})
	pair := func(name string, a, b byte) string {
		return fmt.Sprint(name, ":", []uuid.UUID{accessID(a), accessID(b)})
	}
	want := []string{"inspect:" + accessID(9).String(), "inspect:" + accessID(2).String(), pair("gate", 3, 6), "begin", "credential-clock", "caller", pair("W", 1, 8), pair("P", 3, 6), pair("D", 2, 9), "check:" + accessID(9).String(), "check:" + accessID(2).String(), pair("assets", 10, 12), "audit-clock", "write", "end", "leave"}
	if err != nil || !slices.Equal(r.log, want) || r.inTx || r.inGate {
		t.Fatalf("Within=%v\ngot %q\nwant %q", err, r.log, want)
	}
}

func TestAssetAccessStopsAndReleases(t *testing.T) {
	for _, failure := range []string{"inspect:" + accessID(9).String(), "inspect:" + accessID(2).String(), fmt.Sprint("gate:", []uuid.UUID{accessID(3), accessID(6)}), "begin", "caller", fmt.Sprint("W:", []uuid.UUID{accessID(1), accessID(8)}), fmt.Sprint("P:", []uuid.UUID{accessID(3), accessID(6)}), fmt.Sprint("D:", []uuid.UUID{accessID(2), accessID(9)}), "check:" + accessID(9).String(), "check:" + accessID(2).String(), fmt.Sprint("assets:", []uuid.UUID{accessID(10), accessID(12)}), "write"} {
		t.Run(failure, func(t *testing.T) {
			a, r, actor, q, ids := accessFixture()
			r.fail = failure
			err := a.Within(t.Context(), actor, q, ids, func(context.Context, AccessHeld) error { return r.step("write") })
			if !errors.Is(err, r.failure) || r.inTx || r.inGate {
				t.Fatalf("failure leaked resources: %v %q", err, r.log)
			}
			at := slices.Index(r.log, failure)
			for _, later := range r.log[at+1:] {
				if later != "end" && later != "leave" {
					t.Fatalf("continued after failure: %q", r.log)
				}
			}
		})
	}
	a, r, actor, q, ids := accessFixture()
	func() {
		defer func() {
			if recover() != "test panic" {
				t.Fatal("panic lost")
			}
		}()
		_ = a.Within(t.Context(), actor, q, ids, func(context.Context, AccessHeld) error { panic("test panic") })
	}()
	if r.inTx || r.inGate || !slices.Equal(r.log[len(r.log)-2:], []string{"end", "leave"}) {
		t.Fatalf("panic leaked resources %q", r.log)
	}
}

func TestAssetAccessRechecksPrivateRows(t *testing.T) {
	for _, kind := range []string{"other applicant", "deleted", "missing", "duplicate"} {
		t.Run(kind, func(t *testing.T) {
			a, r, actor, q, ids := accessFixture()
			switch kind {
			case "other applicant":
				r.rows[0].Placement.UserID = accessID(99)
			case "deleted":
				at := time.Unix(1, 0)
				r.rows[0].DeletedAt = &at
			case "missing":
				r.rows = nil
			case "duplicate":
				r.rows[1] = r.rows[0]
			}
			called := false
			err := a.Within(t.Context(), actor, q, ids, func(context.Context, AccessHeld) error { called = true; return nil })
			if !errors.Is(err, domain.ErrAssetNotFound) || called || r.inTx || r.inGate {
				t.Fatalf("private row leaked: %v %q", err, r.log)
			}
		})
	}
}

func TestAssetAccessDeduplicatesParentsWithoutChangingRefusalOrder(t *testing.T) {
	a, r, actor, q, ids := accessFixture()
	h := r.hosts[accessID(2)]
	h.WorkspaceID = accessID(8)
	h.ProjectID = accessID(6)
	r.hosts[accessID(2)] = h
	slices.Reverse(q)
	if err := a.Within(t.Context(), actor, q, ids, func(context.Context, AccessHeld) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if r.log[0] != "inspect:"+accessID(2).String() || r.log[1] != "inspect:"+accessID(9).String() {
		t.Fatalf("semantic order changed %q", r.log)
	}
	for _, want := range []string{fmt.Sprint("gate:", []uuid.UUID{accessID(6)}), fmt.Sprint("W:", []uuid.UUID{accessID(8)}), fmt.Sprint("P:", []uuid.UUID{accessID(6)})} {
		if !slices.Contains(r.log, want) {
			t.Fatalf("parent not deduplicated: %s in %q", want, r.log)
		}
	}
}

func TestAssetAccessCancellationBeforeResources(t *testing.T) {
	a, r, actor, q, ids := accessFixture()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := a.Within(ctx, actor, q, ids, func(context.Context, AccessHeld) error { return nil }); !errors.Is(err, context.Canceled) || len(r.log) != 0 {
		t.Fatalf("canceled request used resources: %v %q", err, r.log)
	}
}
