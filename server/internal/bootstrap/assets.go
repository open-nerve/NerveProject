package bootstrap

import (
	"context"
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

// Only the composition root aggregates ports; each consumer still owns its
// small interface. The concrete driver never reaches business use cases.
type assetStore interface {
	assetapp.UploadSigner
	assetapp.ReadSigner
	assetapp.ObjectInspector
	assetapp.ObjectReader
	assetapp.ObjectSealer
	assetapp.ObjectDeleter
	assetapp.ObjectLister
	assetapp.TemporaryCleaner
}
type assetComponents struct {
	store      assetStore
	transfer   *assethttp.Deps
	sealing    assetapp.SealingDeps
	runtimeNow func() time.Time
	origin     string
	close      func() error
}

func newAssets(ctx context.Context, files config.FilesConfig, keys *identity.Keys, logger *slog.Logger) (assetComponents, error) {
	if err := ctx.Err(); err != nil {
		return assetComponents{}, err
	}
	var components assetComponents
	var err error
	switch files.Driver {
	case "local":
		components, err = newLocalAssets(files, keys, logger)
	case "s3":
		components, err = newS3Assets(ctx, files)
	default:
		return assetComponents{}, errors.New("files.driver must be local or s3")
	}
	if err != nil {
		return components, err
	}
	components.runtimeNow = time.Now
	components.sealing = assetapp.SealingDeps{Inspector: components.store, Reader: components.store, Sealer: components.store, Validator: content.Validator{}}
	return components, nil
}

func newLocalAssets(files config.FilesConfig, keys *identity.Keys, logger *slog.Logger) (assetComponents, error) {
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
	return assetComponents{store: store, transfer: &transfer, close: store.Close}, nil
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
