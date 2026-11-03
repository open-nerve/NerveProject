package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type DuplicateDeps struct {
	Publication      AttachmentPublication
	Rows             AssetReader
	Creator          AssetCreator
	Writer           MetadataWriter
	Access           *Access
	Storage          SealingDeps
	SizeLimit        int64
	OperationTimeout time.Duration
	RuntimeNow       func() time.Time
}
type Duplicate struct{ d DuplicateDeps }

func NewDuplicate(d DuplicateDeps) *Duplicate { return &Duplicate{d: d} }

func (u *Duplicate) Execute(ctx context.Context, sourceID uuid.UUID, in domain.DuplicateAsset) (domain.Asset, error) {
	if fields := domain.DuplicateAssetFields(in); len(fields) != 0 {
		return domain.Asset{}, shared.Invalid(fields...)
	}
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Asset{}, err
	}
	before, err := accessibleAsset(ctx, u.d.Rows, actor, sourceID, ReadAsset)
	if err != nil {
		return domain.Asset{}, err
	}
	var source, target domain.Asset
	var candidate domain.Candidate
	requests := []HostRequest{{Scope: before.Placement.Scope, Intent: ReadAsset}, {Scope: in.Scope, Intent: PrepareAsset}}
	err = u.d.Access.Within(ctx, actor, requests, []AssetAccessRequest{{ID: sourceID, Host: requests[0]}}, func(ctx context.Context, held AccessHeld) error {
		source = held.Assets[sourceID]
		if err := currentPlacement(before, source, held.Hosts[0]); err != nil {
			return err
		}
		if !isPublished(source) {
			return domain.ErrAssetNotUploaded
		}
		name := source.Name
		if in.Name != nil {
			name = *in.Name
		}
		if err := domain.ValidateNewAsset(domain.NewAsset{Purpose: in.Purpose, Scope: in.Scope, Name: name, Size: source.Size, ContentType: source.ContentType}, u.d.SizeLimit); err != nil {
			return err
		}
		h := held.Hosts[1]
		target = domain.Asset{ID: uuid.NewV7(), Purpose: in.Purpose, Name: name, Size: source.Size, ContentType: source.ContentType,
			Placement: domain.Placement{Scope: h.Scope, UserID: actor.UserID, WorkspaceID: h.WorkspaceID, ProjectID: h.ProjectID, IssueID: h.IssueID, CommentID: h.CommentID, DraftID: h.DraftID},
			CreatedAt: held.At, UpdatedAt: held.At, CreatedByID: &actor.UserID, UpdatedByID: &actor.UserID,
			Metadata: domain.StorageMetadata{SchemaVersion: 1, Origin: domain.OriginCopy, Candidates: []domain.Candidate{}},
		}
		candidate, err = reserveCandidate(&target, u.d.RuntimeNow())
		if err != nil {
			return err
		}
		target, err = u.d.Creator.CreateAsset(ctx, target)
		return err
	})
	if err != nil {
		return domain.Asset{}, err
	}
	var verified verifiedObject
	var externalErr error
	if !u.d.RuntimeNow().Before(candidate.LeaseUntil) {
		externalErr = domain.ErrAssetScopeChanged
	} else {
		external, cancel := context.WithTimeout(ctx, u.d.OperationTimeout)
		object := *source.Metadata.Final
		object.Size, object.ContentType = source.Size, source.ContentType
		verified, externalErr = sealContent(external, u.d.Storage, object, candidate.Key, target.Purpose.ImageRequired(), u.d.SizeLimit)
		cancel()
		externalErr = publicationError(ctx, externalErr)
	}
	return u.finish(ctx, actor, source, target, candidate, verified, externalErr)
}

func (u *Duplicate) finish(ctx context.Context, actor shared.Actor, source, target domain.Asset, candidate domain.Candidate, verified verifiedObject, externalErr error) (domain.Asset, error) {
	var result domain.Asset
	requests := []HostRequest{{Scope: source.Placement.Scope, Intent: ReadAsset}, {Scope: target.Placement.Scope, Intent: PrepareAsset}}
	err := u.d.Access.Within(ctx, actor, requests, []AssetAccessRequest{{ID: source.ID, Host: requests[0]}, {ID: target.ID, Host: requests[1]}}, func(ctx context.Context, held AccessHeld) error {
		currentSource, currentTarget := held.Assets[source.ID], held.Assets[target.ID]
		if err := currentPlacement(source, currentSource, held.Hosts[0]); err != nil {
			return err
		}
		if !isPublished(currentSource) || currentSource.Metadata.Final.Key != source.Metadata.Final.Key || currentSource.Metadata.Final.SHA256 != source.Metadata.Final.SHA256 || currentSource.Metadata.Final.ETag != source.Metadata.Final.ETag {
			return domain.ErrAssetScopeChanged
		}
		if err := currentPlacement(target, currentTarget, held.Hosts[1]); err != nil {
			return err
		}
		if externalErr != nil {
			return externalErr
		}
		if currentTarget.Uploaded {
			return domain.ErrAssetScopeChanged
		}
		if err := publishCandidate(&currentTarget, candidate, verified, u.d.RuntimeNow(), held.At, actor.UserID); err != nil {
			return err
		}
		var err error
		result, err = u.d.Writer.SaveAsset(ctx, currentTarget)
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
