package bootstrap

import (
	"context"
	"time"

	assets3 "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/s3"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

func newS3Assets(ctx context.Context, files config.FilesConfig) (assetComponents, error) {
	if err := files.S3.Validate(); err != nil {
		return assetComponents{}, err
	}
	credentials, err := loadS3Credentials(files.S3)
	if err != nil {
		return assetComponents{}, err
	}
	s := files.S3
	store, err := assets3.New(ctx, assets3.Config{
		Endpoint: s.Endpoint, Region: s.Region, Bucket: s.Bucket, Prefix: s.Prefix, UsePathStyle: s.PathStyle,
		AccessKeyID: credentials.accessKeyID, SecretAccessKey: credentials.secretAccessKey, SessionToken: credentials.sessionToken,
		SizeLimit: files.SizeLimit, OperationTimeout: files.OperationTimeout,
	}, assets3.Deps{RuntimeNow: time.Now})
	if err != nil {
		return assetComponents{}, err
	}
	return assetComponents{store: store, origin: store.Origin(), close: store.Close}, nil
}
