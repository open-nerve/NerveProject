package app_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The use cases of the caller's account and preferences: UpdateMe,
// GetProfile and UpdateProfile.

func aliceAccount() *fakeAccount {
	workspace := uuid.MustParse("0199a2b4-0000-7000-8000-000000000007")
	return &fakeAccount{
		user: domain.User{ID: userID, Email: "alice@corp.com", FirstName: "Ann", DisplayName: "alice", Timezone: "Asia/Shanghai", CreatedAt: now},
		profile: domain.Profile{
			Theme: "dark", Language: "zh-CN", StartOfTheWeek: 1, OnboardingStep: domain.OnboardingSteps{ProfileComplete: true},
			LastWorkspaceID: &workspace, UpdatedAt: now,
		},
	}
}

func TestUpdateMe(t *testing.T) {
	account := aliceAccount()
	patch := domain.UserPatch{FirstName: ptr("Ann"), Timezone: ptr("Asia/Shanghai")}

	got, err := app.NewUpdateMe(account, clocktest.At(now)).Execute(shared.WithActor(context.Background(), tokenActor), patch)

	if err != nil || got != account.user {
		t.Errorf("Execute() = %+v, %v; want %+v", got, err, account.user)
	}
	if want := []userUpdate{{userID, patch, now}}; !slices.Equal(account.userUpdates, want) {
		t.Errorf("updates = %+v, want %+v", account.userUpdates, want)
	}
}

func TestGetProfile(t *testing.T) {
	account := aliceAccount()

	got, err := app.NewGetProfile(account).Execute(shared.WithActor(context.Background(), sessionActor))

	if err != nil || got != account.profile {
		t.Errorf("Execute() = %+v, %v; want %+v", got, err, account.profile)
	}
	if want := []uuid.UUID{userID}; !slices.Equal(account.profileReads, want) {
		t.Errorf("profiles read = %v, want %v", account.profileReads, want)
	}
}

func TestUpdateProfile(t *testing.T) {
	account := aliceAccount()
	patch := domain.ProfilePatch{
		Theme: ptr("dark"), OnboardingStep: domain.OnboardingStepsPatch{ProfileComplete: ptr(true)}, LastWorkspaceSet: true,
	}

	got, err := app.NewUpdateProfile(account, clocktest.At(now)).Execute(shared.WithActor(context.Background(), tokenActor), patch)

	if err != nil || got != account.profile {
		t.Errorf("Execute() = %+v, %v; want %+v", got, err, account.profile)
	}
	if want := []profileUpdate{{userID, patch, now}}; !slices.Equal(account.profileUpdates, want) {
		t.Errorf("updates = %+v, want %+v", account.profileUpdates, want)
	}
}

// An invalid patch is one 422 with every field, and writes nothing (M2
// design 4.2, 4.3).
func TestAccountPatchesAreChecked(t *testing.T) {
	ctx := shared.WithActor(context.Background(), sessionActor)
	tests := []struct {
		name   string
		run    func(*fakeAccount) error
		fields []string
	}{
		{"UpdateMe", func(f *fakeAccount) error {
			_, err := app.NewUpdateMe(f, clocktest.At(now)).Execute(ctx, domain.UserPatch{DisplayName: ptr(""), Timezone: ptr("Mars/Olympus")})
			return err
		}, []string{"display_name", "user_timezone"}},
		{"UpdateProfile", func(f *fakeAccount) error {
			_, err := app.NewUpdateProfile(f, clocktest.At(now)).Execute(ctx, domain.ProfilePatch{Theme: ptr("neon"), StartOfTheWeek: ptr(7)})
			return err
		}, []string{"theme", "start_of_the_week"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := aliceAccount()

			err := tt.run(account)

			var se *shared.Error
			var fields []string
			if errors.As(err, &se) {
				for _, f := range se.Fields {
					fields = append(fields, f.Field)
				}
			}
			if se == nil || se.Code != shared.CodeValidationFailed || !slices.Equal(fields, tt.fields) {
				t.Errorf("Execute() = %v, want validation_failed on %v", err, tt.fields)
			}
			if len(account.userUpdates)+len(account.profileUpdates) != 0 {
				t.Errorf("wrote %+v %+v, want nothing", account.userUpdates, account.profileUpdates)
			}
		})
	}
}

// Without an actor, and when the account is gone since authentication, the
// caller is unauthenticated; a database error stays one (500). Without an
// actor the store is never asked.
func TestAccountUseCasesErrors(t *testing.T) {
	boom := errors.New("connection refused")
	gone := uuid.MustParse("0199a2b4-0000-7000-8000-000000000009")
	authed := shared.WithActor(context.Background(), sessionActor)
	useCases := []struct {
		name string
		run  func(context.Context, *fakeAccount) error
	}{
		{"UpdateMe", func(ctx context.Context, f *fakeAccount) error {
			_, err := app.NewUpdateMe(f, clocktest.At(now)).Execute(ctx, domain.UserPatch{})
			return err
		}},
		{"GetProfile", func(ctx context.Context, f *fakeAccount) error {
			_, err := app.NewGetProfile(f).Execute(ctx)
			return err
		}},
		{"UpdateProfile", func(ctx context.Context, f *fakeAccount) error {
			_, err := app.NewUpdateProfile(f, clocktest.At(now)).Execute(ctx, domain.ProfilePatch{})
			return err
		}},
	}
	tests := []struct {
		name    string
		ctx     context.Context
		account uuid.UUID // the id of the fake's account
		err     error     // the fake's error
		want    error
		asked   bool // whether the store is asked
	}{
		{"without an actor", context.Background(), userID, nil, shared.Unauthenticated(), false},
		{"when the account is gone", authed, gone, nil, shared.Unauthenticated(), true},
		{"when the database fails", authed, userID, boom, boom, true},
	}
	for _, uc := range useCases {
		for _, tt := range tests {
			t.Run(uc.name+" "+tt.name, func(t *testing.T) {
				account := &fakeAccount{user: domain.User{ID: tt.account}, err: tt.err}

				if err := uc.run(tt.ctx, account); !errors.Is(err, tt.want) {
					t.Errorf("Execute() = %v, want %v", err, tt.want)
				}
				if calls := len(account.userUpdates) + len(account.profileReads) + len(account.profileUpdates); (calls == 1) != tt.asked {
					t.Errorf("the store was asked %d times, want asked: %v", calls, tt.asked)
				}
			})
		}
	}
}
