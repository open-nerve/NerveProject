package bootstrap

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

type s3Credentials struct {
	accessKeyID, secretAccessKey, sessionToken string
}

func loadS3Credentials(c config.S3FilesConfig) (s3Credentials, error) {
	if err := c.ValidateCredentials(); err != nil {
		return s3Credentials{}, err
	}
	for _, source := range []struct {
		key   string
		value *string
		file  *string
		limit int
	}{
		{"access_key_id", &c.AccessKeyID, &c.AccessKeyIDFile, config.S3AccessKeyIDMaxBytes},
		{"secret_access_key", &c.SecretAccessKey, &c.SecretAccessKeyFile, config.S3SecretAccessKeyMaxBytes},
		{"session_token", &c.SessionToken, &c.SessionTokenFile, config.S3SessionTokenMaxBytes},
	} {
		if *source.file == "" {
			continue
		}
		value, err := readS3Secret(*source.file, source.limit)
		if err != nil {
			return s3Credentials{}, fmt.Errorf("files.s3.%s_file: %w", source.key, err)
		}
		*source.value, *source.file = value, ""
	}
	if err := c.ValidateCredentials(); err != nil {
		return s3Credentials{}, err
	}
	return s3Credentials{accessKeyID: c.AccessKeyID, secretAccessKey: c.SecretAccessKey, sessionToken: c.SessionToken}, nil
}

func readS3Secret(path string, limit int) (string, error) {
	// Check before opening too: a configured FIFO/device must not block startup.
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("must name a readable regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("cannot open secret file")
	}
	defer func() { _ = f.Close() }()
	info, err = f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("must name a readable regular file")
	}
	if info.Size() > int64(limit+2) {
		return "", fmt.Errorf("secret file exceeds its byte limit")
	}
	data, err := io.ReadAll(io.LimitReader(f, int64(limit+3)))
	if err != nil {
		return "", fmt.Errorf("cannot read secret file")
	}
	value := string(data)
	if strings.HasSuffix(value, "\r\n") {
		value = strings.TrimSuffix(value, "\r\n")
	} else {
		value = strings.TrimSuffix(value, "\n")
	}
	if value == "" || len(data) > limit+2 || len(value) > limit {
		return "", fmt.Errorf("secret file is empty or exceeds its byte limit")
	}
	return value, nil
}
