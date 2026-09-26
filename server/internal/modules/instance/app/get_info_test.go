package app_test

import (
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/instance/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/domain"
)

type fixedSource domain.Build

func (s fixedSource) Build() domain.Build { return domain.Build(s) }

func TestGetInfoDescribesTheBuildAndTheSettings(t *testing.T) {
	settings := domain.Settings{SignupEnabled: true, FileSizeLimit: 5242880}
	uc := app.NewGetInfo(fixedSource{Version: "1.2.3", Commit: "4f2a9c1"}, settings)

	got := uc.Execute()

	want := domain.Info{Product: "Nerve", Version: "1.2.3", Commit: "4f2a9c1", APIVersion: "v0", Settings: settings}
	if got != want {
		t.Errorf("Execute() = %+v, want %+v", got, want)
	}
}
