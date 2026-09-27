package app_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
)

func (f *adminFixture) activate() *app.Activate {
	return app.NewActivate(app.ActivateDeps{Accounts: f.store, Users: f.store, APITokens: f.store, Tx: f.tx, Clock: clocktest.At(now), Logger: f.logger()})
}

// One transaction locks the account by its normalized address, activates
// it and counts the tokens that authenticate again (M2 decision 3).
func TestActivate(t *testing.T) {
	f := newAdminFixture()

	got, err := f.activate().Execute(context.Background(), " Alice@Corp.com")

	if want := (app.ActivateResult{Email: "alice@corp.com", APITokens: 2}); err != nil || got != want {
		t.Fatalf("Execute() = %+v, %v; want %+v", got, err, want)
	}
	want := []string{"lock alice@corp.com", "activate " + userID.String(), "count the tokens of " + userID.String()}
	if !slices.Equal(f.log.calls, want) || f.tx.calls != 1 || !slices.Equal(f.store.writtenAt, []time.Time{now, now}) {
		t.Errorf("calls %q in %d transactions at %v; want %q in one at %v", f.log.calls, f.tx.calls, f.store.writtenAt, want, now)
	}
	if logs := f.logs.String(); !strings.Contains(logs, `"msg":"account activated","user_id":"`+userID.String()+`","usable_api_tokens":2,"by":"cli"`) {
		t.Errorf("logs = %s, want the activation with user_id and the usable tokens, by cli", logs)
	}
}

func TestActivateFails(t *testing.T) {
	boom := errors.New("connection reset")
	tests := []struct {
		name  string
		email string
		fail  func(*fakeAdmin)
		want  error
	}{
		{"unknown account", "carol@corp.com", func(*fakeAdmin) {}, domain.ErrAccountNotFound},
		{"a failed write", "alice@corp.com", func(s *fakeAdmin) { s.activateErr = boom }, boom},
		{"a failed count", "alice@corp.com", func(s *fakeAdmin) { s.countErr = boom }, boom},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newAdminFixture()
			tt.fail(f.store)

			if _, err := f.activate().Execute(context.Background(), tt.email); !errors.Is(err, tt.want) || f.logs.Len() != 0 {
				t.Errorf("Execute() = %v, logs %s; want %v and nothing logged", err, f.logs.String(), tt.want)
			}
		})
	}
}
