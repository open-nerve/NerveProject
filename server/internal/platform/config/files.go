package config

import (
	"errors"
	"log/slog"
	"strings"
	"time"
)

// FilesConfig bounds raw transfers separately from ordinary JSON requests.
// SizeLimit is also reported by the instance API.
type FilesConfig struct {
	UploadTTL                 time.Duration    `koanf:"upload_ttl"`
	OperationTimeout          time.Duration    `koanf:"operation_timeout"`
	CleanupInterval           time.Duration    `koanf:"cleanup_interval"`
	CleanupBatch              int              `koanf:"cleanup_batch"`
	CleanupBudget             time.Duration    `koanf:"cleanup_budget"`
	SizeLimit                 int64            `koanf:"size_limit"`
	Driver                    string           `koanf:"driver"`
	Local                     LocalFilesConfig `koanf:"local"`
	S3                        S3FilesConfig    `koanf:"s3"`
	MinTransferBytesPerSecond int64            `koanf:"min_transfer_bytes_per_second"`
	TransferGrace             time.Duration    `koanf:"transfer_grace"`
	TransferIdleTimeout       time.Duration    `koanf:"transfer_idle_timeout"`
	MaxTransferDuration       time.Duration    `koanf:"max_transfer_duration"`
	MaxConcurrentTransfers    int              `koanf:"max_concurrent_transfers"`
	TransferRate              BucketConfig     `koanf:"transfer_rate"`
}

type LocalFilesConfig struct {
	Root string `koanf:"root"`
}

const maxFileSize int64 = 100 << 20

// TransferBudget uses bounded integer arithmetic. Checking bounds first keeps
// the ceiling and duration conversion safe, even for directly constructed configs.
func (c FilesConfig) TransferBudget(size int64) (time.Duration, error) {
	if c.SizeLimit < 1 || c.SizeLimit > maxFileSize ||
		c.MaxTransferDuration <= 0 || c.MaxTransferDuration > time.Hour {
		return 0, errors.New("invalid file transfer budget")
	}
	uploadBudget, err := fileTransferBudget(c.SizeLimit, c.MinTransferBytesPerSecond, c.TransferGrace)
	if err != nil || uploadBudget > c.MaxTransferDuration {
		return 0, errors.New("invalid file transfer budget")
	}
	budget, err := fileTransferBudget(size, c.MinTransferBytesPerSecond, c.TransferGrace)
	if err != nil {
		return 0, err
	}
	// SizeLimit only limits new uploads. Previously stored objects may be larger;
	// they can still be read within the administrator's explicit duration cap.
	return min(budget, c.MaxTransferDuration), nil
}

func fileTransferBudget(size, rate int64, grace time.Duration) (time.Duration, error) {
	if size < 1 || size > maxFileSize || rate < 1024 || rate > 1<<30 || grace <= 0 || grace > time.Minute {
		return 0, errors.New("invalid file transfer budget")
	}
	seconds := size / rate
	if size%rate != 0 {
		seconds++
	}
	return grace + time.Duration(seconds)*time.Second, nil
}

func (c FilesConfig) validate(fail func(key, format string, args ...any)) {
	for _, bound := range []struct {
		key                 string
		value, lower, upper time.Duration
	}{
		{"upload_ttl", c.UploadTTL, 10 * time.Second, 15 * time.Minute},
		{"operation_timeout", c.OperationTimeout, time.Second, 10 * time.Second},
		{"cleanup_interval", c.CleanupInterval, time.Minute, 24 * time.Hour},
		{"cleanup_budget", c.CleanupBudget, time.Second, 30 * time.Second},
	} {
		if bound.value < bound.lower || bound.value > bound.upper {
			fail("files."+bound.key, "must be from %s to %s, got %s", bound.lower, bound.upper, bound.value)
		}
	}
	if c.CleanupBatch < 1 || c.CleanupBatch > 1000 {
		fail("files.cleanup_batch", "must be from 1 to 1000, got %d", c.CleanupBatch)
	}
	if c.SizeLimit < 1 || c.SizeLimit > maxFileSize {
		fail("files.size_limit", "must be from 1 to 104857600, got %d", c.SizeLimit)
	}
	switch c.Driver {
	case "local":
		if strings.TrimSpace(c.Local.Root) == "" || strings.ContainsRune(c.Local.Root, 0) {
			fail("files.local.root", "must be a nonempty directory path")
		}
		c.S3.validateCredentials(false, fail)
	case "s3":
		c.S3.validate(fail)
	default:
		fail("files.driver", "must be local or s3")
	}
	if c.MinTransferBytesPerSecond < 1024 || c.MinTransferBytesPerSecond > 1<<30 {
		fail("files.min_transfer_bytes_per_second", "must be from 1024 to 1073741824, got %d", c.MinTransferBytesPerSecond)
	}
	if c.TransferGrace <= 0 || c.TransferGrace > time.Minute {
		fail("files.transfer_grace", "must be positive and at most 1m, got %s", c.TransferGrace)
	}
	if c.TransferIdleTimeout <= 0 || c.TransferIdleTimeout > time.Minute {
		fail("files.transfer_idle_timeout", "must be positive and at most 1m, got %s", c.TransferIdleTimeout)
	}
	if c.MaxTransferDuration <= 0 || c.MaxTransferDuration > time.Hour {
		fail("files.max_transfer_duration", "must be positive and at most 1h, got %s", c.MaxTransferDuration)
	}
	if budget, err := fileTransferBudget(c.SizeLimit, c.MinTransferBytesPerSecond, c.TransferGrace); err == nil && c.MaxTransferDuration > 0 && budget > c.MaxTransferDuration {
		fail("files.max_transfer_duration", "must cover the maximum file's transfer budget (%s), got %s", budget, c.MaxTransferDuration)
	}
	if c.MaxConcurrentTransfers < 1 || c.MaxConcurrentTransfers > 256 {
		fail("files.max_concurrent_transfers", "must be from 1 to 256, got %d", c.MaxConcurrentTransfers)
	}
	c.TransferRate.validate("files.transfer_rate", fail)
}

func (c LocalFilesConfig) LogValue() slog.Value {
	return slog.GroupValue(slog.Bool("root_set", c.Root != ""))
}

func (c FilesConfig) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Duration("upload_ttl", c.UploadTTL), slog.Duration("operation_timeout", c.OperationTimeout),
		slog.Duration("cleanup_interval", c.CleanupInterval), slog.Int("cleanup_batch", c.CleanupBatch), slog.Duration("cleanup_budget", c.CleanupBudget),
		slog.Int64("size_limit", c.SizeLimit), slog.String("driver", c.Driver), slog.Any("local", c.Local), slog.Any("s3", c.S3),
		slog.Int64("min_transfer_bytes_per_second", c.MinTransferBytesPerSecond),
		slog.Duration("transfer_grace", c.TransferGrace), slog.Duration("transfer_idle_timeout", c.TransferIdleTimeout),
		slog.Duration("max_transfer_duration", c.MaxTransferDuration), slog.Int("max_concurrent_transfers", c.MaxConcurrentTransfers),
		slog.Any("transfer_rate", c.TransferRate),
	)
}
