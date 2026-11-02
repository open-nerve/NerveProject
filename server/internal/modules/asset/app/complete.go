package app

import (
	"context"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type CompleteDeps struct {
	Rows             AssetReader
	Writer           MetadataWriter
	Access           *Access
	Storage          SealingDeps
	SizeLimit        int64
	OperationTimeout time.Duration
	RuntimeNow       func() time.Time
}
type Complete struct{ d CompleteDeps }

func NewComplete(d CompleteDeps) *Complete { return &Complete{d: d} }

func (u *Complete) Execute(ctx context.Context, id uuid.UUID) (domain.Asset, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Asset{}, err
	}
	before, err := ownedAsset(ctx, u.d.Rows, actor, id)
	if err != nil {
		return domain.Asset{}, err
	}
	var reserved domain.Asset
	var candidate domain.Candidate
	err = u.d.Access.Within(ctx, actor, []HostRequest{{Scope: before.Placement.Scope, Intent: PrepareAsset}}, []uuid.UUID{id}, func(ctx context.Context, held AccessHeld) error {
		row := held.Assets[0]
		if err := currentPlacement(before, row, held.Hosts[0]); err != nil {
			return err
		}
		if isPublished(row) {
			reserved = row
			return nil
		}
		if row.Metadata.Staging == nil || row.Metadata.Staging.Phase != domain.ObjectActive {
			return domain.ErrAssetObjectMissing
		}
		if row.Size > u.d.SizeLimit {
			return domain.ErrAssetTooLarge
		}
		if len(row.Metadata.Candidates) >= domain.MaxCandidates {
			return domain.AssetBusy()
		}
		candidate, err = reserveCandidate(&row, u.d.RuntimeNow())
		if err != nil {
			return err
		}
		reserved, err = u.d.Writer.SaveAsset(ctx, row)
		return err
	})
	if err != nil {
		return domain.Asset{}, err
	}
	if isPublished(reserved) {
		return reserved, nil
	}
	var verified verifiedObject
	var externalErr error
	if !u.d.RuntimeNow().Before(candidate.LeaseUntil) {
		externalErr = domain.ErrAssetScopeChanged
	} else {
		external, cancel := context.WithTimeout(ctx, u.d.OperationTimeout)
		verified, externalErr = sealContent(external, u.d.Storage, reserved.Metadata.Staging.Key, candidate.Key, reserved.Size, reserved.ContentType, reserved.Purpose.ImageRequired(), u.d.SizeLimit)
		cancel()
		externalErr = publicationError(ctx, externalErr)
	}
	return u.finish(ctx, actor, reserved, candidate, verified, externalErr)
}

// A reservation's revision is not a publication precondition: other attempts
// can append candidates meanwhile. Apply only this candidate to the locked row.
func (u *Complete) finish(ctx context.Context, actor shared.Actor, before domain.Asset, candidate domain.Candidate, verified verifiedObject, externalErr error) (domain.Asset, error) {
	var result domain.Asset
	err := u.d.Access.Within(ctx, actor, []HostRequest{{Scope: before.Placement.Scope, Intent: PrepareAsset}}, []uuid.UUID{before.ID}, func(ctx context.Context, held AccessHeld) error {
		row := held.Assets[0]
		if err := currentPlacement(before, row, held.Hosts[0]); err != nil {
			return err
		}
		if isPublished(row) {
			result = row
			return nil
		}
		if externalErr != nil {
			return externalErr
		}
		index := slices.IndexFunc(row.Metadata.Candidates, func(c domain.Candidate) bool { return c.Key == candidate.Key })
		if index < 0 || row.Metadata.Candidates[index].Phase != domain.ObjectActive || !row.Metadata.Candidates[index].LeaseUntil.Equal(candidate.LeaseUntil) || !u.d.RuntimeNow().Before(candidate.LeaseUntil) {
			return domain.ErrAssetScopeChanged
		}
		final := row.Metadata.Candidates[index].ObjectRecord
		final.ETag, final.SHA256 = verified.info.ETag, verified.proof.SHA256
		row.Metadata.Final = &final
		row.Metadata.Candidates = slices.Delete(row.Metadata.Candidates, index, index+1)
		row.Uploaded = true
		row.Width, row.Height = verified.proof.Width, verified.proof.Height
		row.UpdatedAt, row.UpdatedByID = held.At, &actor.UserID
		var err error
		result, err = u.d.Writer.SaveAsset(ctx, row)
		return err
	})
	if err != nil {
		return domain.Asset{}, err
	}
	return result, nil
}

func reserveCandidate(row *domain.Asset, now time.Time) (domain.Candidate, error) {
	key, err := domain.NewObjectKey('o', row.ID, uuid.New())
	if err != nil {
		return domain.Candidate{}, err
	}
	lease := now.UTC().Add(domain.CandidateLease)
	candidate := domain.Candidate{ObjectRecord: domain.ObjectRecord{Key: key, Phase: domain.ObjectActive, SafeAfter: lease.Add(domain.CandidateSettleGrace), Size: row.Size, ContentType: row.ContentType}, LeaseUntil: lease}
	row.Metadata.Candidates = append(row.Metadata.Candidates, candidate)
	return candidate, nil
}
func isPublished(row domain.Asset) bool {
	return row.Uploaded && row.Metadata.Final != nil && row.Metadata.Final.Phase == domain.ObjectActive
}
