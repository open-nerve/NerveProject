package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type TransferBudget interface {
	TransferBudget(int64) (time.Duration, error)
}
type PrepareDeps struct {
	Access     *Access
	Rows       AssetCreator
	Signer     UploadSigner
	Budget     TransferBudget
	SizeLimit  int64
	UploadTTL  time.Duration
	RuntimeNow func() time.Time
}
type Prepare struct{ d PrepareDeps }
type PreparedAsset struct {
	Asset  domain.Asset
	Upload domain.SignedRequest
}

func NewPrepare(d PrepareDeps) *Prepare { return &Prepare{d: d} }

func (u *Prepare) Execute(ctx context.Context, in domain.NewAsset) (PreparedAsset, error) {
	if fields := domain.NewAssetFields(in); len(fields) != 0 {
		return PreparedAsset{}, shared.Invalid(fields...)
	}
	if err := domain.ValidateNewAsset(in, u.d.SizeLimit); err != nil {
		return PreparedAsset{}, err
	}
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return PreparedAsset{}, err
	}
	budget, err := u.d.Budget.TransferBudget(in.Size)
	if err != nil {
		return PreparedAsset{}, err
	}
	var result PreparedAsset
	err = u.d.Access.Within(ctx, actor, []HostRequest{{Scope: in.Scope, Intent: PrepareAsset}}, nil, func(ctx context.Context, held AccessHeld) error {
		id := uuid.NewV7()
		key, err := domain.NewObjectKey('u', id, uuid.New())
		if err != nil {
			return err
		}
		expires := u.d.RuntimeNow().UTC().Add(u.d.UploadTTL).Truncate(time.Second)
		h := held.Hosts[0]
		row := domain.Asset{
			ID: id, Purpose: in.Purpose, Name: in.Name, Size: in.Size, ContentType: in.ContentType,
			Placement: domain.Placement{Scope: h.Scope, UserID: actor.UserID, WorkspaceID: h.WorkspaceID, ProjectID: h.ProjectID, IssueID: h.IssueID, CommentID: h.CommentID, DraftID: h.DraftID},
			CreatedAt: held.At, UpdatedAt: held.At, CreatedByID: &actor.UserID, UpdatedByID: &actor.UserID, UploadExpiresAt: &expires,
			Metadata: domain.StorageMetadata{SchemaVersion: 1, Origin: domain.OriginUpload, Candidates: []domain.Candidate{}, Staging: &domain.ObjectRecord{
				Key: key, Phase: domain.ObjectActive, SafeAfter: expires.Add(budget + domain.CandidateSettleGrace), Size: in.Size, ContentType: in.ContentType,
			}},
		}
		result.Asset, err = u.d.Rows.CreateAsset(ctx, row)
		if err != nil {
			return err
		}
		result.Upload, err = u.d.Signer.SignUpload(ctx, domain.UploadGrant{Key: key, Size: in.Size, ContentType: in.ContentType, ExpiresAt: expires})
		if err != nil {
			return err
		}
		if !result.Upload.ExpiresAt.Equal(expires) {
			return domain.ErrInvalidUpload
		}
		return nil
	})
	if err != nil {
		return PreparedAsset{}, err
	}
	return result, nil
}
