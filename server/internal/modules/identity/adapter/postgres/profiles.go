package postgresadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// GetProfile reads userID's preferences; app.ErrNotFound when there are
// none.
func (s *Store) GetProfile(ctx context.Context, userID uuid.UUID) (domain.Profile, error) {
	row, err := s.queries(ctx).GetProfile(ctx, userID)
	if err != nil {
		return domain.Profile{}, notFound(err)
	}
	return profileOf(row)
}

// UpdateProfile applies p to userID's preferences at now, in one statement
// that merges the steps (M2 design 3.14), and returns them;
// app.ErrNotFound when there are none.
func (s *Store) UpdateProfile(ctx context.Context, userID uuid.UUID, p domain.ProfilePatch, now time.Time) (domain.Profile, error) {
	arg := gen.UpdateProfileParams{Now: now, UserID: userID, OnboardingStepPatch: p.OnboardingStep.JSON()}
	arg.SetTheme, arg.Theme = set(p.Theme)
	arg.SetLanguage, arg.Language = set(p.Language)
	if p.StartOfTheWeek != nil {
		arg.SetStartOfTheWeek, arg.StartOfTheWeek = true, int16(*p.StartOfTheWeek) // 0–6, checked by the domain
	}
	arg.SetIsOnboarded, arg.IsOnboarded = set(p.IsOnboarded)
	arg.SetIsTourCompleted, arg.IsTourCompleted = set(p.IsTourCompleted)
	arg.SetLastWorkspaceID, arg.LastWorkspaceID = p.LastWorkspaceSet, p.LastWorkspaceID
	row, err := s.queries(ctx).UpdateProfile(ctx, arg)
	if err != nil {
		return domain.Profile{}, notFound(err)
	}
	return profileOf(gen.GetProfileRow(row))
}

// ResetOnboarding puts userID's onboarding back to the defaults it had at
// registration, at now.
func (s *Store) ResetOnboarding(ctx context.Context, userID uuid.UUID, now time.Time) error {
	if err := s.queries(ctx).ResetOnboarding(ctx, gen.ResetOnboardingParams{Now: now, UserID: userID}); err != nil {
		return fmt.Errorf("reset onboarding: %w", err)
	}
	return nil
}

// profileOf reads a profile row. The steps are the object that the
// column's CHECK guarantees: four booleans.
func profileOf(row gen.GetProfileRow) (domain.Profile, error) {
	var steps domain.OnboardingSteps
	if err := json.Unmarshal(row.OnboardingStep, &steps); err != nil {
		return domain.Profile{}, fmt.Errorf("read onboarding_step: %w", err)
	}
	return domain.Profile{
		Theme:           row.Theme,
		Language:        row.Language,
		StartOfTheWeek:  int(row.StartOfTheWeek),
		OnboardingStep:  steps,
		IsOnboarded:     row.IsOnboarded,
		IsTourCompleted: row.IsTourCompleted,
		LastWorkspaceID: row.LastWorkspaceID,
		UpdatedAt:       row.UpdatedAt,
	}, nil
}
