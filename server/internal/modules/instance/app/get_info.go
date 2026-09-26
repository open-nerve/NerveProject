package app

import "github.com/open-nerve/NerveProject/server/internal/modules/instance/domain"

// GetInfo tells API clients what this instance runs and how it is set up.
type GetInfo struct {
	source   InfoSource
	settings domain.Settings
}

// NewGetInfo returns the use case, reading the build from source.
func NewGetInfo(source InfoSource, settings domain.Settings) *GetInfo {
	return &GetInfo{source: source, settings: settings}
}

// Execute describes the instance.
func (uc *GetInfo) Execute() domain.Info {
	build := uc.source.Build()
	return domain.Info{
		Product:    domain.Product,
		Version:    build.Version,
		Commit:     build.Commit,
		APIVersion: domain.APIVersion,
		Settings:   uc.settings,
	}
}
