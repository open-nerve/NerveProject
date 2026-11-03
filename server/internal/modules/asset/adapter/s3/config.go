package s3

import (
	"errors"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

type Config struct {
	Endpoint, Region, Bucket, Prefix, AccessKeyID, SecretAccessKey, SessionToken string
	UsePathStyle                                                                 bool
	SizeLimit                                                                    int64
	OperationTimeout                                                             time.Duration
}

func (c Config) validate() error {
	if c.SizeLimit < 1 || c.SizeLimit > domain.MaxObjectSize || c.OperationTimeout < time.Second || c.OperationTimeout > 10*time.Second {
		return errors.New("invalid S3 object size or operation timeout")
	}
	// Reuse the pure deployment-format validator. This adapter accepts only
	// resolved credentials, never a secret file or ambient source.
	return (config.S3FilesConfig{Endpoint: c.Endpoint, Region: c.Region, Bucket: c.Bucket, Prefix: c.Prefix, PathStyle: c.UsePathStyle,
		AccessKeyID: c.AccessKeyID, SecretAccessKey: c.SecretAccessKey, SessionToken: c.SessionToken}).Validate()
}
