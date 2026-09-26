package app_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func (f *adminFixture) setEmail() *app.SetEmail {
	return app.NewSetEmail(app.SetEmailDeps{Accounts: f.store, Users: f.store, Sessions: f.store, Tx: f.tx, Clock: clocktest.At(now), Logger: f.logger()})
}

// One transaction locks the account by its old address, writes the new
// one, both normalized, and revokes every session; the tokens stay (M2
// design 3.17). Neither address is logged (8.4).
func TestSetEmail(t *testing.T) {
	f := newAdminFixture()

	got, err := f.setEmail().Execute(context.Background(), " ALICE@corp.com", "Alice@New.Example ")

	if want := (app.SetEmailResult{From: "alice@corp.com", To: "alice@new.example", Sessions: 1}); err != nil || got != want {
		t.Fatalf("Execute() = %+v, %v; want %+v", got, err, want)
	}
	want := []string{
		"lock alice@corp.com", "email " + userID.String() + " alice@new.example",
		"revoke email_changed sessions of " + userID.String() + " but " + uuid.Nil().String(),
	}
	if !slices.Equal(f.log.calls, want) || f.tx.calls != 1 || !slices.Equal(f.store.writtenAt, []time.Time{now, now}) {
		t.Errorf("calls %q in %d transactions at %v; want %q in one at %v", f.log.calls, f.tx.calls, f.store.writtenAt, want, now)
	}
	logs := f.logs.String()
	if !strings.Contains(logs, `"msg":"e-mail address changed","user_id":"`+userID.String()+`","revoked_sessions":1,"by":"cli"`) ||
		strings.Contains(strings.ToLower(logs), "alice@") {
		t.Errorf("logs = %s, want the change with user_id and the sessions revoked, and no address", logs)
	}
}

// The new address is checked by the rules of registration and must differ
// from the old one once both are normalized; then nothing is locked.
func TestSetEmailChecksTheNewAddress(t *testing.T) {
	tests := []struct {
		name, newEmail string
		want           error
	}{
		{"invalid", "not-an-address", shared.Invalid(shared.FieldError{Field: "new_email", Code: shared.FieldInvalidFormat})},
		{"empty", "  ", shared.Invalid(shared.FieldError{Field: "new_email", Code: shared.FieldRequired})},
		{"the same", "Alice@CORP.com ", domain.ErrEmailUnchanged},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newAdminFixture()

			_, err := f.setEmail().Execute(context.Background(), "alice@corp.com", tt.newEmail)

			var got, want *shared.Error
			if !errors.As(err, &got) || !errors.As(tt.want, &want) || got.Code != want.Code || len(got.Fields) != len(want.Fields) ||
				len(want.Fields) == 1 && (got.Fields[0].Field != want.Fields[0].Field || got.Fields[0].Code != want.Fields[0].Code) || len(f.log.calls) != 0 {
				t.Errorf("Execute() = %+v after calls %q, want %+v and no call", err, f.log.calls, tt.want)
			}
		})
	}
}

func TestSetEmailFails(t *testing.T) {
	tests := []struct {
		name, email, newEmail string
		want                  error
		calls                 []string
	}{
		{"unknown account", "carol@corp.com", "carol@new.example", domain.ErrAccountNotFound, []string{"lock carol@corp.com"}},
		{"another account's address", "alice@corp.com", takenEmail, domain.ErrEmailTaken, []string{"lock alice@corp.com"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newAdminFixture()

			_, err := f.setEmail().Execute(context.Background(), tt.email, tt.newEmail)

			if !errors.Is(err, tt.want) || !slices.Equal(f.log.calls, tt.calls) || f.logs.Len() != 0 {
				t.Errorf("Execute() = %v after calls %q, logs %s; want %v after %q, nothing logged", err, f.log.calls, f.logs.String(), tt.want, tt.calls)
			}
		})
	}
}
