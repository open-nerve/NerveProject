package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/netip"
	"os"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/ratelimit"
)

// The configuration as the composition hands it to the modules and the
// platform: the signing key, the invitation MAC, the password hashing, the
// rate-limit buckets, and the warning of a non-prod nerve listening beyond
// this machine.

// readSigningKey returns the content of auth.jwt.private_key_file, or nil
// when it is not set. Errors name the key, never the path (M0-P2 handoff 4).
func readSigningKey(path string) ([]byte, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		var pathErr *fs.PathError
		if errors.As(err, &pathErr) {
			err = pathErr.Err
		}
		return nil, fmt.Errorf("auth.jwt.private_key_file: %w", err)
	}
	return data, nil
}

// invitationMAC is the invitation MAC the workspace module takes: keys'
// MAC of workspace.InvitationMACPurpose (M3 design 3.8, 6.6), the one place
// the composition names the purpose.
func invitationMAC(keys *identity.Keys) (identity.MAC, error) {
	return keys.MAC(workspace.InvitationMACPurpose)
}

// passwordHashing is auth.password as identity takes it.
func passwordHashing(p config.PasswordConfig) identity.PasswordHashing {
	return identity.PasswordHashing{
		MemoryKiB:     p.Argon2MemoryKiB,
		Iterations:    p.Argon2Iterations,
		Parallelism:   p.Argon2Parallelism,
		MaxConcurrent: p.MaxConcurrentHashes,
		MaxWait:       p.MaxWait,
	}
}

// bucket is the rate-limit bucket name on limiter, sized by c (M2 design
// 3.10).
func bucket(limiter *ratelimit.Limiter, name string, c config.BucketConfig) *ratelimit.Bucket {
	return limiter.Bucket(name, ratelimit.Rate{PerMinute: c.PerMinute, Burst: c.Burst})
}

// warnIfExposed warns once when a non-prod nerve listens beyond loopback:
// most likely a deployment without NERVE_ENV=prod (M2 design 6.1).
func warnIfExposed(ctx context.Context, logger *slog.Logger, cfg config.Config) {
	if cfg.Env == config.EnvProd || loopback(cfg.Server.Addr) {
		return
	}
	logger.WarnContext(ctx, "not running as prod but listening beyond this machine: sign-up is open by default, "+
		"and without auth.jwt.private_key_file the signing key is ephemeral; set NERVE_ENV=prod to deploy",
		slog.String("env", cfg.Env), slog.String("addr", cfg.Server.Addr))
}

func loopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip, err := netip.ParseAddr(host)
	return err == nil && ip.IsLoopback()
}
