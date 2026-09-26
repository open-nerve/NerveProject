package app

import (
	"context"
	"log/slog"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// Logout ends the session of a refresh token: POST /api/v0/auth/logout.
type Logout struct {
	sessions SessionEnder
	clock    Clock
	logger   *slog.Logger
}

// NewLogout returns the use case.
func NewLogout(sessions SessionEnder, clock Clock, logger *slog.Logger) *Logout {
	return &Logout{sessions: sessions, clock: clock, logger: logger}
}

// Execute revokes the session of token with reason logout, only while token
// is its current generation and the session is live: the first row of
// refresh's table (M2 design 3.5). Any other token changes nothing and is no
// error either, so the answer never tells what the token was; reuse is
// judged only by refresh. One statement, no transaction (M2 design 6.4).
func (l *Logout) Execute(ctx context.Context, token string) error {
	presented, ok := domain.ParseRefreshToken(token)
	if !ok {
		return nil
	}
	ended, err := l.sessions.EndSession(ctx, SessionGeneration{
		ID: presented.SessionID, Generation: presented.Generation, TokenHash: presented.SecretHash(), Now: l.clock.Now(),
	})
	if err != nil || !ended {
		return err
	}
	l.logger.InfoContext(ctx, "signed out", slog.String("session_id", presented.SessionID.String()))
	return nil
}
