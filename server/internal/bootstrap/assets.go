package bootstrap

import (
	"github.com/open-nerve/NerveProject/server/internal/modules/asset"
	assetapp "github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
)

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
