package postgresadapter

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// CreateUser inserts u. A taken address is domain.ErrEmailTaken. The domain
// validated every value, so a CHECK violation is a bug: an internal error
// (500), not a domain error.
func (s *Store) CreateUser(ctx context.Context, u app.NewUser) error {
	err := s.queries(ctx).CreateUser(ctx, gen.CreateUserParams{
		ID: u.ID, Email: u.Email, Password: u.PasswordHash, DisplayName: u.DisplayName, Now: u.Now,
	})
	switch {
	case uniqueViolation(err, "users_email_key"):
		return domain.ErrEmailTaken
	case err != nil:
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// GetUser reads account id; app.ErrNotFound when there is none.
func (s *Store) GetUser(ctx context.Context, id uuid.UUID) (domain.User, error) {
	row, err := s.queries(ctx).GetUser(ctx, id)
	if err != nil {
		return domain.User{}, notFound(err)
	}
	return domain.User{
		ID:          row.ID,
		Email:       row.Email,
		FirstName:   row.FirstName,
		LastName:    row.LastName,
		DisplayName: row.DisplayName,
		Timezone:    row.UserTimezone,
		CreatedAt:   row.CreatedAt,
	}, nil
}

// CreateDefaultProfile inserts the profile of a new account with Plane's
// model defaults.
func (s *Store) CreateDefaultProfile(ctx context.Context, id, userID uuid.UUID, now time.Time) error {
	if err := s.queries(ctx).CreateProfile(ctx, gen.CreateProfileParams{ID: id, UserID: userID, Now: now}); err != nil {
		return fmt.Errorf("create profile: %w", err)
	}
	return nil
}
