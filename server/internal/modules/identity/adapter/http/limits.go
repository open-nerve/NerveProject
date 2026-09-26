package httpadapter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/ratelimit"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// RateLimiter takes a unit from several buckets at once, or from none:
// platform/ratelimit's Limiter (M2 design 3.10).
type RateLimiter interface {
	AllowAll(checks ...ratelimit.Check) (denied *ratelimit.Bucket, retry time.Duration)
}

// Limits are the module's own buckets, on Limiter (M2 design 3.10). The
// platform's anonymous bucket has counted the request already: one that a
// bucket here refuses still counts there.
type Limits struct {
	Limiter      RateLimiter
	LoginIP      *ratelimit.Bucket // ratelimit.login_ip, by client IP key
	LoginIPEmail *ratelimit.Bucket // ratelimit.login_ip_email, by client IP key and address
	RegisterIP   *ratelimit.Bucket // ratelimit.register_ip, by client IP key
}

// limitLogin takes a unit of login_ip and one of login_ip_email, or
// neither: a login that login_ip_email refuses leaves login_ip as it was,
// so failing on one address does not lock the client out of the others.
func (h handler) limitLogin(ctx context.Context, email string) error {
	key := httpserver.RequestMetaFrom(ctx).IPKey
	return h.allow(ctx,
		ratelimit.Check{Bucket: h.s.Limits.LoginIP, Key: key},
		ratelimit.Check{Bucket: h.s.Limits.LoginIPEmail, Key: ipEmailKey(key, email)})
}

// limitRegister takes a unit of register_ip.
func (h handler) limitRegister(ctx context.Context) error {
	return h.allow(ctx, ratelimit.Check{Bucket: h.s.Limits.RegisterIP, Key: httpserver.RequestMetaFrom(ctx).IPKey})
}

// allow answers 429 rate_limited with Retry-After when a bucket refuses,
// and logs which bucket turned the client away (M2 design 8.4), as the
// platform does for its own.
func (h handler) allow(ctx context.Context, checks ...ratelimit.Check) error {
	denied, retry := h.s.Limits.Limiter.AllowAll(checks...)
	if denied == nil {
		return nil
	}
	h.s.Logger.LogAttrs(ctx, slog.LevelInfo, "rate limited",
		slog.String("request_id", httpserver.RequestID(ctx)), slog.String("bucket", denied.Name()),
		slog.String("ip", httpserver.RequestMetaFrom(ctx).ClientIP.String()))
	return shared.RateLimited(retry)
}

// ipEmailKey is login_ip_email's key: the client IP key and the normalized
// address, hashed so that the key's size does not depend on what the
// client sends.
func ipEmailKey(ipKey, email string) string {
	sum := sha256.Sum256([]byte(domain.NormalizeEmail(email)))
	return ipKey + " " + hex.EncodeToString(sum[:])
}
