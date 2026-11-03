package app

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type assetReadFunc func(context.Context, uuid.UUID) (domain.Asset, error)

func (f assetReadFunc) FindAsset(ctx context.Context, id uuid.UUID) (domain.Asset, error) {
	return f(ctx, id)
}

type readSignFunc func(context.Context, domain.ReadGrant) (domain.SignedRequest, error)

func (f readSignFunc) SignRead(ctx context.Context, g domain.ReadGrant) (domain.SignedRequest, error) {
	return f(ctx, g)
}

// Exercise the expiry in the use case's issued grant, not only its helper.
// Literal boundaries catch <= instead of < and non-UTC rounding.
func TestAssetContentExpiry(t *testing.T) {
	for _, tc := range []struct{ now, expiry string }{
		{"2026-11-03T10:00:00Z", "2026-11-03T11:00:00Z"},
		{"2026-11-03T10:55:00Z", "2026-11-03T11:00:00Z"},
		{"2026-11-03T10:55:00.000000001Z", "2026-11-03T12:00:00Z"},
		{"2026-11-03T18:59:59+08:00", "2026-11-03T12:00:00Z"},
		{"2026-11-03T23:59:59Z", "2026-11-04T01:00:00Z"},
	} {
		t.Run(tc.now, func(t *testing.T) {
			access, record, actor, requests, _ := accessFixture()
			h := record.hosts[requests[0].Scope.ID]
			row := domain.Asset{ID: accessID(10), Purpose: domain.DraftIssueAttachment, Name: "still-a-download.png", ContentType: "image/png", Uploaded: true,
				Placement: domain.Placement{Scope: h.Scope, UserID: actor.UserID, WorkspaceID: h.WorkspaceID, ProjectID: h.ProjectID, DraftID: h.DraftID}}
			key, err := domain.NewObjectKey('o', row.ID, accessID(31))
			if err != nil {
				t.Fatal(err)
			}
			row.Metadata.Final = &domain.ObjectRecord{Key: key, Phase: domain.ObjectActive}
			record.rows = []domain.Asset{row}
			now, err := time.Parse(time.RFC3339Nano, tc.now)
			if err != nil {
				t.Fatal(err)
			}
			u := NewResolveContent(ResolveContentDeps{Rows: assetReadFunc(func(context.Context, uuid.UUID) (domain.Asset, error) { return row, nil }), Access: access, RuntimeNow: func() time.Time { return now },
				Signer: readSignFunc(func(_ context.Context, grant domain.ReadGrant) (domain.SignedRequest, error) {
					if !record.inTx || !record.inGate || grant.Key != key || grant.Method != "GET" || grant.Inline || grant.Name != row.Name || grant.ContentType != "image/png" {
						t.Fatalf("bad grant or unprotected signature: %+v", grant)
					}
					if err := domain.ValidateRead(grant, now); err != nil {
						t.Fatal(err)
					}
					return domain.SignedRequest{Method: grant.Method, URL: "signed", ExpiresAt: grant.ExpiresAt}, nil
				})})
			got, err := u.Execute(shared.WithActor(t.Context(), actor), row.ID)
			if err != nil || got.URL != "signed" || got.ExpiresAt.Format(time.RFC3339Nano) != tc.expiry || record.inTx || record.inGate {
				t.Fatalf("content=%+v err=%v want=%s", got, err, tc.expiry)
			}
		})
	}
}

func TestAssetContentRejectsChangedStateBeforeSigning(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*domain.Asset)
		want   error
	}{
		{"pending", func(a *domain.Asset) { a.Uploaded = false }, domain.ErrAssetNotUploaded},
		{"no final", func(a *domain.Asset) { a.Metadata.Final = nil }, domain.ErrAssetNotUploaded},
		{"reclaiming", func(a *domain.Asset) { a.Metadata.Final.Phase = domain.ObjectReclaiming }, domain.ErrAssetNotUploaded},
		{"revision", func(a *domain.Asset) { a.StorageRevision++ }, domain.ErrAssetScopeChanged},
		{"project changed", func(a *domain.Asset) { a.Placement.ProjectID = accessID(42) }, domain.ErrAssetScopeChanged},
		{"purpose changed", func(a *domain.Asset) { a.Purpose = domain.DraftIssueDescription }, domain.ErrAssetScopeChanged},
		{"bound changed", func(a *domain.Asset) { at := time.Unix(1, 0); a.BoundAt = &at }, domain.ErrAssetScopeChanged},
		{"deleted", func(a *domain.Asset) { at := time.Unix(1, 0); a.DeletedAt = &at }, domain.ErrAssetNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			access, record, actor, requests, _ := accessFixture()
			h := record.hosts[requests[0].Scope.ID]
			row := domain.Asset{ID: accessID(10), Purpose: domain.DraftIssueAttachment, Uploaded: true,
				Placement: domain.Placement{Scope: h.Scope, UserID: actor.UserID, WorkspaceID: h.WorkspaceID, ProjectID: h.ProjectID, DraftID: h.DraftID},
				Metadata:  domain.StorageMetadata{Final: &domain.ObjectRecord{Phase: domain.ObjectActive}}}
			current := row
			final := *row.Metadata.Final
			current.Metadata.Final = &final
			tc.change(&current)
			record.rows = []domain.Asset{current}
			u := NewResolveContent(ResolveContentDeps{Rows: assetReadFunc(func(context.Context, uuid.UUID) (domain.Asset, error) { return row, nil }), Access: access, RuntimeNow: time.Now,
				Signer: readSignFunc(func(context.Context, domain.ReadGrant) (domain.SignedRequest, error) {
					t.Fatal("invalid state reached signer")
					return domain.SignedRequest{}, nil
				})})
			got, err := u.Execute(shared.WithActor(t.Context(), actor), row.ID)
			if !errors.Is(err, tc.want) || got.URL != "" || record.inTx || record.inGate {
				t.Fatalf("got=%+v err=%v want=%v", got, err, tc.want)
			}
		})
	}
}
