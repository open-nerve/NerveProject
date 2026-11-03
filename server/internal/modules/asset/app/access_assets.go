package app

import (
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func accessAssetReferences(hosts []HostRequest, refs []AssetAccessRequest) ([]uuid.UUID, []AssetAccessRequest, error) {
	seen := make(map[uuid.UUID]HostRequest, len(refs))
	var ids []uuid.UUID
	var unique []AssetAccessRequest
	for _, ref := range refs {
		if ref.ID == uuid.Nil() || !slices.Contains(hosts, ref.Host) {
			return nil, nil, domain.ErrInvalidAsset
		}
		if previous, ok := seen[ref.ID]; ok {
			if previous != ref.Host {
				return nil, nil, domain.ErrInvalidAsset
			}
			continue
		}
		seen[ref.ID] = ref.Host
		ids, unique = append(ids, ref.ID), append(unique, ref)
	}
	return orderedIDs(ids), unique, nil
}

func checkAccessAssets(actor shared.Actor, requests []HostRequest, hosts []Host, refs []AssetAccessRequest, rows []domain.Asset) (map[uuid.UUID]domain.Asset, error) {
	if len(rows) != len(refs) {
		return nil, domain.ErrAssetNotFound
	}
	byID := make(map[uuid.UUID]domain.Asset, len(rows))
	for _, row := range rows {
		if _, duplicate := byID[row.ID]; duplicate {
			return nil, domain.ErrAssetNotFound
		}
		byID[row.ID] = row
	}
	byRequest := make(map[HostRequest]Host, len(hosts))
	for n, host := range hosts {
		byRequest[requests[n]] = host
	}
	// Keep semantic source/target order independent of the physical row locks.
	for _, ref := range refs {
		row, ok := byID[ref.ID]
		if !ok || !assetVisibleFor(row, actor, ref.Host.Intent) {
			return nil, domain.ErrAssetNotFound
		}
		if hostOf(row.Placement) != byRequest[ref.Host] {
			return nil, domain.ErrAssetScopeChanged
		}
	}
	return byID, nil
}

func assetVisibleFor(row domain.Asset, actor shared.Actor, intent AccessIntent) bool {
	if row.DeletedAt != nil && intent != RemoveAsset {
		return false
	}
	if row.Placement.UserID == actor.UserID {
		return true
	}
	return intent != PrepareAsset && row.IsBoundContent() &&
		(isPublished(row) || (intent == RemoveAsset && row.DeletedAt != nil))
}
