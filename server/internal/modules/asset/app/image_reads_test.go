package app

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func imageReadFixture(t *testing.T, purpose domain.Purpose) (*imageRows, ImageReference, time.Time) {
	t.Helper()
	_, f, request, now := imageFixture(t)
	row := f.rows[*request.Changes[0].CurrentID]
	row.Purpose = purpose
	ref := ImageReference{AssetID: row.ID, Purpose: purpose}
	switch purpose {
	case domain.UserAvatar, domain.UserCover:
		ref.UserID = row.Placement.UserID
	case domain.WorkspaceLogo:
		ref.WorkspaceID = accessID(30)
		row.Placement.WorkspaceID = ref.WorkspaceID
		row.Placement.Scope = domain.Scope{Kind: domain.ScopeWorkspace, ID: ref.WorkspaceID}
	case domain.ProjectCover:
		ref.WorkspaceID, ref.ProjectID = accessID(30), accessID(40)
		row.Placement.WorkspaceID, row.Placement.ProjectID = ref.WorkspaceID, ref.ProjectID
		row.Placement.Scope = domain.Scope{Kind: domain.ScopeProject, ID: ref.ProjectID}
	}
	f.rows = map[uuid.UUID]domain.Asset{row.ID: row}
	return f, ref, now
}

func TestReadBoundImages(t *testing.T) {
	for _, purpose := range []domain.Purpose{domain.UserAvatar, domain.UserCover, domain.WorkspaceLogo, domain.ProjectCover} {
		t.Run(string(purpose), func(t *testing.T) {
			f, ref, now := imageReadFixture(t, purpose)
			before, err := json.Marshal(f.rows)
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			u := NewImageReader(ImageReadDeps{Locks: f, RuntimeNow: func() time.Time { return now },
				Signer: readSignFunc(func(_ context.Context, grant domain.ReadGrant) (domain.SignedRequest, error) {
					calls++
					row := f.rows[ref.AssetID]
					if len(f.locks) != 1 || !reflect.DeepEqual(f.locks[0], []uuid.UUID{ref.AssetID}) {
						t.Fatal("sign before complete locks")
					}
					if grant.Key != row.Metadata.Final.Key || grant.Method != "GET" || !grant.Inline || grant.Name != row.Name || grant.ContentType != row.ContentType {
						t.Fatalf("grant=%+v", grant)
					}
					if err := domain.ValidateRead(grant, now); err != nil {
						t.Fatal(err)
					}
					return domain.SignedRequest{URL: "signed-image", ExpiresAt: grant.ExpiresAt}, nil
				})})
			got, err := u.ReadBoundImages(t.Context(), []ImageReference{ref, ref})
			if err != nil || len(got) != 1 || calls != 1 || got[ref.AssetID].URL != "signed-image" || !got[ref.AssetID].ExpiresAt.Equal(now.Truncate(time.Hour).Add(time.Hour)) {
				t.Fatalf("images=%+v calls=%d err=%v", got, calls, err)
			}
			after, err := json.Marshal(f.rows)
			if err != nil || string(before) != string(after) || len(f.writes) != 0 {
				t.Fatal("read changed asset/audit")
			}
		})
	}
}

func TestReadBoundImagesMissingOrInvalid(t *testing.T) {
	for _, name := range []string{"missing", "preparation", "deleted", "pending", "no final", "reclaiming", "purpose", "user", "workspace", "project", "extra parent", "new project scope"} {
		t.Run(name, func(t *testing.T) {
			purpose := domain.UserAvatar
			if name == "workspace" || name == "project" || name == "new project scope" {
				purpose = domain.ProjectCover
			}
			f, ref, now := imageReadFixture(t, purpose)
			a := f.rows[ref.AssetID]
			switch name {
			case "preparation":
				a.BoundAt = nil
			case "deleted":
				a.DeletedAt = &now
			case "pending":
				a.Uploaded = false
			case "no final":
				a.Metadata.Final = nil
			case "reclaiming":
				a.Metadata.Final.Phase = domain.ObjectReclaiming
			case "purpose":
				a.Purpose = domain.UserCover
			case "user":
				a.Placement.UserID = accessID(99)
			case "workspace":
				a.Placement.WorkspaceID = accessID(99)
			case "project":
				a.Placement.ProjectID = accessID(99)
				a.Placement.Scope.ID = accessID(99)
			case "extra parent":
				a.Placement.CommentID = accessID(99)
			case "new project scope":
				a.Placement.Scope = domain.Scope{Kind: domain.ScopeNewProject, ID: ref.WorkspaceID}
				a.Placement.ProjectID = uuid.Nil()
			}
			f.rows[a.ID] = a
			if name == "missing" {
				delete(f.rows, a.ID)
			}
			u := NewImageReader(ImageReadDeps{Locks: f, RuntimeNow: func() time.Time { return now }, Signer: readSignFunc(func(context.Context, domain.ReadGrant) (domain.SignedRequest, error) {
				t.Fatal("invalid reference reached signer")
				return domain.SignedRequest{}, nil
			})})
			got, err := u.ReadBoundImages(t.Context(), []ImageReference{ref})
			if err != nil || len(got) != 0 || len(f.writes) != 0 {
				t.Fatalf("invalid image=%+v %v", got, err)
			}
		})
	}
}

func TestReadBoundImagesErrors(t *testing.T) {
	for _, name := range []string{"database", "metadata", "signer"} {
		t.Run(name, func(t *testing.T) {
			f, ref, now := imageReadFixture(t, domain.UserAvatar)
			want := errors.New(name)
			if name == "metadata" {
				want = domain.ErrInvalidMetadata
			}
			if name != "signer" {
				f.lockErr = want
			}
			calls := 0
			u := NewImageReader(ImageReadDeps{Locks: f, RuntimeNow: func() time.Time { return now }, Signer: readSignFunc(func(context.Context, domain.ReadGrant) (domain.SignedRequest, error) {
				calls++
				return domain.SignedRequest{}, want
			})})
			got, err := u.ReadBoundImages(t.Context(), []ImageReference{ref})
			if !errors.Is(err, want) || got != nil {
				t.Fatalf("error swallowed: %+v %v", got, err)
			}
			if name != "signer" && calls != 0 {
				t.Fatal("signed after lock/decode failure")
			}
		})
	}
}

func TestReadBoundImagesConflictingReferences(t *testing.T) {
	for _, field := range []string{"purpose", "user", "workspace", "project"} {
		t.Run(field, func(t *testing.T) {
			f, ref, now := imageReadFixture(t, domain.UserAvatar)
			other := ref
			switch field {
			case "purpose":
				other.Purpose = domain.UserCover
			case "user":
				other.UserID = accessID(99)
			case "workspace":
				other.WorkspaceID = accessID(99)
			case "project":
				other.ProjectID = accessID(99)
			}
			for _, refs := range [][]ImageReference{{ref, other}, {other, ref}} {
				u := NewImageReader(ImageReadDeps{Locks: f, RuntimeNow: func() time.Time { return now }, Signer: readSignFunc(func(context.Context, domain.ReadGrant) (domain.SignedRequest, error) {
					t.Fatal("conflicting reference granted a shared capability")
					return domain.SignedRequest{}, nil
				})})
				got, err := u.ReadBoundImages(t.Context(), refs)
				if !errors.Is(err, domain.ErrInvalidAsset) || got != nil {
					t.Fatalf("conflict=%+v %v", got, err)
				}
			}
		})
	}
}
