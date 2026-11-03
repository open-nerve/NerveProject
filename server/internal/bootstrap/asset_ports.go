package bootstrap

import (
	"context"
	"errors"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

type assetWorkspaceLocks struct{ parents workspace.AssetParents }

func (l assetWorkspaceLocks) ShareMany(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	return l.parents.ShareAssetWorkspaces(ctx, ids)
}

type assetDraftLocks struct{ parents issue.AssetParents }

func (l assetDraftLocks) LockMany(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	return l.parents.LockAssetDrafts(ctx, ids)
}

type assetProjectLocks struct{ projects project.IssueProjects }

func (l assetProjectLocks) LockMany(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	rows, err := l.projects.LockIssueProjects(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]uuid.UUID, 0, len(rows))
	for id := range rows {
		out = append(out, id)
	}
	return sortedAssetParents(out), nil
}

type assetWriteGate struct{ gate writeGate }

func forAssetWrites(g writeGate) assetWriteGate { return assetWriteGate{gate: g} }
func (g assetWriteGate) Enter(ctx context.Context, ids ...uuid.UUID) (func(), error) {
	if _, nested := postgres.Tx(ctx); nested {
		return nil, errors.New("asset operation cannot join an existing transaction")
	}
	return g.gate.Enter(ctx, ids...)
}

func sortedAssetParents(ids []uuid.UUID) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id != uuid.Nil() {
			out = append(out, id)
		}
	}
	slices.SortFunc(out, func(a, b uuid.UUID) int { return a.Compare(b) })
	return slices.Compact(out)
}
