// Package argon2adapter hashes passwords with argon2id (M2 design 3.8), with a cap
// on concurrent hashes and on how long a caller waits for one.
package argon2adapter

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/argon2"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Params are auth.password in the configuration.
type Params struct {
	MemoryKiB     uint32        // argon2_memory_kib
	Iterations    uint32        // argon2_iterations
	Parallelism   uint8         // argon2_parallelism
	MaxConcurrent int           // max_concurrent_hashes
	MaxWait       time.Duration // max_wait
}

const (
	saltLen = 16
	keyLen  = 32
	// retryAfter is the Retry-After of the 503 when no slot frees up.
	retryAfter = time.Second
)

// Hasher implements app.PasswordHasher.
type Hasher struct {
	p       Params
	logger  *slog.Logger
	slots   chan struct{}
	waiting atomic.Int64
}

// New returns a hasher with params p.
func New(p Params, logger *slog.Logger) *Hasher {
	return &Hasher{p: p, logger: logger, slots: make(chan struct{}, p.MaxConcurrent)}
}

// Hash returns the PHC string of password:
// $argon2id$v=19$m=<KiB>,t=<iterations>,p=<lanes>$<salt>$<key>, base64
// without padding. When no slot frees up within MaxWait it returns 503
// server_busy with Retry-After: 1 and logs the queue length.
func (h *Hasher) Hash(ctx context.Context, password string) (string, error) {
	if err := h.acquire(ctx); err != nil {
		return "", err
	}
	defer func() { <-h.slots }()
	salt := make([]byte, saltLen)
	_, _ = rand.Read(salt) // never fails since Go 1.24
	key := argon2.IDKey([]byte(password), salt, h.p.Iterations, h.p.MemoryKiB, h.p.Parallelism, keyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, h.p.MemoryKiB, h.p.Iterations, h.p.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

func (h *Hasher) acquire(ctx context.Context) error {
	select {
	case h.slots <- struct{}{}:
		return nil
	default:
	}
	queued := h.waiting.Add(1)
	defer h.waiting.Add(-1)
	timer := time.NewTimer(h.p.MaxWait)
	defer timer.Stop()
	select {
	case h.slots <- struct{}{}:
		return nil
	case <-timer.C:
		h.logger.LogAttrs(ctx, slog.LevelInfo, "password hashing is saturated", slog.Int64("queued", queued))
		return shared.ServerBusy(retryAfter)
	case <-ctx.Done():
		return ctx.Err()
	}
}
