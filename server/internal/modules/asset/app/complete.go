package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type CompleteDeps struct {
	Publication      AttachmentPublication
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
	before, err := accessibleAsset(ctx, u.d.Rows, actor, id, PrepareAsset)
	if err != nil {
		return domain.Asset{}, err
	}
	var reserved domain.Asset
	var candidate domain.Candidate
	request := HostRequest{Scope: before.Placement.Scope, Intent: PrepareAsset}
	err = u.d.Access.Within(ctx, actor, []HostRequest{request}, []AssetAccessRequest{{ID: id, Host: request}}, func(ctx context.Context, held AccessHeld) error {
		row := held.Assets[id]
		if err := completionPlacement(before, row, held.Hosts[0]); err != nil {
			return err
		}
		if isPublished(row) {
			reserved, err = u.d.Publication.Bind(ctx, row, actor.UserID, held.At)
			return err
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
		source := *reserved.Metadata.Staging
		source.Size, source.ContentType = reserved.Size, reserved.ContentType
		verified, externalErr = sealContent(external, u.d.Storage, source, candidate.Key, reserved.Purpose.ImageRequired(), u.d.SizeLimit)
		cancel()
		externalErr = publicationError(ctx, externalErr)
	}
	return u.finish(ctx, actor, reserved, candidate, verified, externalErr)
}

// A reservation's revision is not a publication precondition: other attempts
// can append candidates meanwhile. Apply only this candidate to the locked row.
func (u *Complete) finish(ctx context.Context, actor shared.Actor, before domain.Asset, candidate domain.Candidate, verified verifiedObject, externalErr error) (domain.Asset, error) {
	var result domain.Asset
	request := HostRequest{Scope: before.Placement.Scope, Intent: PrepareAsset}
	err := u.d.Access.Within(ctx, actor, []HostRequest{request}, []AssetAccessRequest{{ID: before.ID, Host: request}}, func(ctx context.Context, held AccessHeld) error {
		row := held.Assets[before.ID]
		if err := completionPlacement(before, row, held.Hosts[0]); err != nil {
			return err
		}
		if isPublished(row) {
			var err error
			result, err = u.d.Publication.Bind(ctx, row, actor.UserID, held.At)
			return err
		}
		if externalErr != nil {
			return externalErr
		}
		if err := publishCandidate(&row, candidate, verified, u.d.RuntimeNow(), held.At, actor.UserID); err != nil {
			return err
		}
		var err error
		result, err = u.d.Writer.SaveAsset(ctx, row)
		if err != nil {
			return err
		}
		result, err = u.d.Publication.Bind(ctx, result, actor.UserID, held.At)
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
