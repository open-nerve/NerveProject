package bootstrap

import (
	"errors"
	"log/slog"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/content"
	assethttp "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/local"
	assetapp "github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

type assetComponents struct {
	store    *local.Store
	transfer assethttp.Deps
	sealing  assetapp.SealingDeps
	close    func() error
}

func newAssets(files config.FilesConfig, keys *identity.Keys, logger *slog.Logger) (assetComponents, error) {
	if files.Driver != "local" {
		return assetComponents{}, errors.New("files.driver must be local")
	}
	upload, read, err := assetMACs(keys)
	if err != nil {
		return assetComponents{}, err
	}
	store, err := local.New(local.Config{Root: files.Local.Root, SizeLimit: files.SizeLimit}, local.Deps{UploadMAC: upload, ReadMAC: read, RuntimeNow: time.Now})
	if err != nil {
		return assetComponents{}, err
	}
	transfer := assethttp.Deps{
		UploadVerifier: store, ReadVerifier: store, Uploader: store, Reader: store,
		Files: files, RuntimeNow: time.Now, Errors: httpserver.NewAPIErrors(logger),
	}
	return assetComponents{store: store, transfer: transfer, sealing: assetapp.SealingDeps{Inspector: store, Reader: store, Sealer: store, Validator: content.Validator{}}, close: store.Close}, nil
}

// assetMACs is the only composition point that selects file MAC purposes.
// It uses already loaded keys; no storage adapter reads credentials or key files.
func assetMACs(keys *identity.Keys) (upload, read assetapp.MAC, err error) {
	upload, err = keys.MAC(asset.UploadMACPurpose)
	if err != nil {
		return nil, nil, err
	}
	read, err = keys.MAC(asset.ReadMACPurpose)
	if err != nil {
		return nil, nil, err
	}
	return upload, read, nil
}
