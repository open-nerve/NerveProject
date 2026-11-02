package app

import (
	"context"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type AccessDeps struct {
	Hosts                  HostAccess
	Workspaces             WorkspaceLocks
	Projects               ProjectLocks
	Drafts                 DraftLocks
	Assets                 AssetLocker
	Caller                 CallerLock
	Gate                   WriteGate
	Tx                     shared.TxManager
	CredentialClock, Clock Clock
}
type Access struct{ d AccessDeps }
type AccessHeld struct {
	Hosts  []Host
	Assets []domain.Asset
	At     time.Time
}

func NewAccess(d AccessDeps) *Access { return &Access{d: d} }

// Within is the asset module's transaction boundary. Its callback belongs to
// this module; owner modules supply capabilities, never arbitrary callbacks.
func (a *Access) Within(ctx context.Context, actor shared.Actor, requests []HostRequest, assetIDs []uuid.UUID, fn func(context.Context, AccessHeld) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if actor.UserID == uuid.Nil() {
		return shared.Unauthenticated()
	}
	if len(requests) == 0 || fn == nil {
		return domain.ErrInvalidAsset
	}
	h := AccessHeld{Hosts: make([]Host, 0, len(requests))}
	var workspaceIDs, projectIDs, draftIDs []uuid.UUID
	for _, request := range requests {
		if request.Intent < ReadAsset || request.Intent > RemoveAsset {
			return domain.ErrInvalidAsset
		}
		host, err := a.d.Hosts.Inspect(ctx, actor, request)
		if err != nil {
			return err
		}
		if host.Scope != request.Scope {
			return domain.ErrAssetScopeChanged
		}
		h.Hosts = append(h.Hosts, host)
		workspaceIDs = append(workspaceIDs, host.WorkspaceID)
		projectIDs = append(projectIDs, host.ProjectID)
		draftIDs = append(draftIDs, host.DraftID)
	}
	workspaceIDs, projectIDs, draftIDs = orderedIDs(workspaceIDs), orderedIDs(projectIDs), orderedIDs(draftIDs)
	for _, id := range assetIDs {
		if id == uuid.Nil() {
			return domain.ErrInvalidAsset
		}
	}
	assetIDs = orderedIDs(assetIDs)
	if err := ctx.Err(); err != nil {
		return err
	}
	leave, err := a.d.Gate.Enter(ctx, projectIDs...)
	if err != nil {
		return err
	}
	defer leave()
	return a.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := a.d.Caller.LockCaller(ctx, actor, a.d.CredentialClock.Now()); err != nil {
			return err
		}
		missing := false
		for _, locks := range []struct {
			ids  []uuid.UUID
			take func(context.Context, []uuid.UUID) ([]uuid.UUID, error)
		}{
			{workspaceIDs, a.d.Workspaces.ShareMany}, {projectIDs, a.d.Projects.LockMany}, {draftIDs, a.d.Drafts.LockMany},
		} {
			if len(locks.ids) == 0 {
				continue
			}
			got, err := locks.take(ctx, locks.ids)
			if err != nil {
				return err
			}
			missing = missing || !slices.Equal(orderedIDs(got), locks.ids)
		}
		// Re-read and authorize in semantic order (source, then target). The
		// physical lock order is independent of this refusal order.
		for n, request := range requests {
			if err := a.d.Hosts.Check(ctx, actor, request, h.Hosts[n]); err != nil {
				return err
			}
		}
		if missing {
			return domain.ErrAssetNotFound
		}
		if len(assetIDs) > 0 {
			var err error
			h.Assets, err = a.d.Assets.LockAssets(ctx, assetIDs)
			if err != nil {
				return err
			}
			got := make([]uuid.UUID, 0, len(h.Assets))
			for _, row := range h.Assets {
				if row.Placement.UserID != actor.UserID || row.DeletedAt != nil {
					return domain.ErrAssetNotFound
				}
				got = append(got, row.ID)
			}
			if len(got) != len(assetIDs) || !slices.Equal(orderedIDs(got), assetIDs) {
				return domain.ErrAssetNotFound
			}
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		h.At = a.d.Clock.Now()
		return fn(ctx, h)
	})
}

func orderedIDs(ids []uuid.UUID) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id != uuid.Nil() {
			out = append(out, id)
		}
	}
	slices.SortFunc(out, func(a, b uuid.UUID) int { return a.Compare(b) })
	return slices.Compact(out)
}
