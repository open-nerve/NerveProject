package app

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func TestBoundAssetAccessAssociatesIntentAndHost(t *testing.T) {
	for _, name := range []string{"read source", "foreign target", "wrong host", "unbound", "pending", "reclaiming", "reversed rows"} {
		t.Run(name, func(t *testing.T) {
			a, r, actor, q, refs := accessFixture()
			at := time.Unix(1, 0)
			r.rows[0].Placement.UserID = accessID(99)
			r.rows[0].BoundAt = &at
			r.rows[0].Uploaded = true
			r.rows[0].Metadata.Final = &domain.ObjectRecord{Phase: domain.ObjectActive}
			var want error
			switch name {
			case "foreign target":
				r.rows[1].Placement.UserID = accessID(99)
				r.rows[1].BoundAt = &at
				r.rows[1].Uploaded = true
				r.rows[1].Metadata.Final = &domain.ObjectRecord{Phase: domain.ObjectActive}
				want = domain.ErrAssetNotFound
			case "wrong host":
				refs[1].Host = q[1]
				r.rows[0].Placement.UserID = actor.UserID
				want = domain.ErrAssetScopeChanged
			case "unbound":
				r.rows[0].BoundAt = nil
				want = domain.ErrAssetNotFound
			case "pending":
				r.rows[0].Uploaded = false
				want = domain.ErrAssetNotFound
			case "reclaiming":
				r.rows[0].Metadata.Final.Phase = domain.ObjectReclaiming
				want = domain.ErrAssetNotFound
			case "reversed rows":
				slices.Reverse(r.rows)
			}
			called := false
			err := a.Within(t.Context(), actor, q, refs, func(_ context.Context, h AccessHeld) error {
				called = true
				if h.Assets[accessID(10)].Placement.Scope != q[0].Scope || h.Assets[accessID(12)].Placement.Scope != q[1].Scope {
					t.Fatal("physical ordering changed source/target identity")
				}
				return nil
			})
			if !errors.Is(err, want) || called != (want == nil) {
				t.Fatalf("access=%v callback=%t want=%v", err, called, want)
			}
		})
	}
}

func TestBoundAssetAccessRejectsUnassociatedRequests(t *testing.T) {
	for _, name := range []string{"zero ID", "unknown host", "conflicting intent"} {
		t.Run(name, func(t *testing.T) {
			a, r, actor, q, refs := accessFixture()
			switch name {
			case "zero ID":
				refs[0].ID = accessID(0)
			case "unknown host":
				refs[0].Host.Scope.ID = accessID(77)
			case "conflicting intent":
				refs[2].Host = q[0]
			}
			called := false
			err := a.Within(t.Context(), actor, q, refs, func(context.Context, AccessHeld) error { called = true; return nil })
			if !errors.Is(err, domain.ErrInvalidAsset) || called || len(r.log) != 0 {
				t.Fatalf("ambiguous request acquired resources: %v %v", err, r.log)
			}
		})
	}
}
