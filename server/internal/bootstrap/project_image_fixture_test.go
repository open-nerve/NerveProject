package bootstrap

import (
	"context"
	"errors"
	"time"
	"uuid"

	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// Existing M3 route fixtures contain no images and no credentials. Their
// actual account and gate locks remain; complete image/credential composition
// is proved separately by newIssueWorld's authenticated HTTP requests.
type projectFixtureImages struct{}

func (projectFixtureImages) LockImageBindings(ctx context.Context, q projectapp.CoverImageChange) (time.Time, error) {
	if _, inTx := postgres.Tx(ctx); !inTx || q.Current != nil || q.Next != nil {
		return time.Time{}, errors.New("unexpected image in no-image fixture")
	}
	return clock.System{}.Now(), nil
}
func (projectFixtureImages) ApplyImageBindings(ctx context.Context, q projectapp.CoverImageChange, _ time.Time) error {
	if _, inTx := postgres.Tx(ctx); !inTx || q.Current != nil || q.Next != nil {
		return errors.New("unexpected image in no-image fixture")
	}
	return nil
}
func (projectFixtureImages) ReadProjectImages(_ context.Context, refs []projectapp.ProjectImageReference) (map[uuid.UUID]projectapp.ImageURL, error) {
	for _, ref := range refs {
		if ref.CoverImageAssetID != nil {
			return nil, errors.New("unexpected image in no-image fixture")
		}
	}
	return nil, nil
}
func fixtureProjectImageGate() projectImageGate {
	return projectImageGate{gate: newWriteGate(config.WriteGateConfig{PerProject: 1, MaxQueue: 32, MaxWait: 5 * time.Second})}
}
