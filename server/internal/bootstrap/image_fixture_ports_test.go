package bootstrap

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The old repository interleavings deliberately construct account-only actors.
// Preserve their actual first account row lock; authenticated HTTP image tests
// exercise the real credential recheck, which those fixtures do not supply.
type imageFixtureCaller struct{ pool *pgxpool.Pool }

func (f imageFixtureCaller) LockCaller(ctx context.Context, a shared.Actor, _ time.Time) error {
	account, err := identitypg.New(f.pool).LockForCredentials(ctx, a.UserID)
	if err != nil {
		return err
	}
	if !account.Active {
		return shared.Unauthenticated()
	}
	return nil
}

// These fixtures have no images. An unexpected reference is an error, never a
// dummy signed success; real image cases always use the fully wired app.
type imageFixtureReader struct{}

func (imageFixtureReader) ReadWorkspaceImages(_ context.Context, refs []workspaceapp.WorkspaceImageReference) (map[uuid.UUID]workspaceapp.ImageURL, error) {
	for _, ref := range refs {
		if ref.LogoAssetID != nil {
			return nil, fmt.Errorf("unexpected logo in no-image fixture")
		}
	}
	return nil, nil
}
func (imageFixtureReader) ReadMemberAvatars(_ context.Context, refs []workspaceapp.MemberAvatarReference) (map[uuid.UUID]workspaceapp.ImageURL, error) {
	for _, ref := range refs {
		if ref.AssetID != nil {
			return nil, fmt.Errorf("unexpected avatar in no-image fixture")
		}
	}
	return nil, nil
}
