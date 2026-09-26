# M2/P2 登录与会话 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 登录、续期、退出可用；刷新令牌的轮换和重复使用检测、会话的绝对期限、限流（令牌桶、认证之前的失败闸门、登录和注册的桶）、安全响应头全部到位；A3–A6、A15 的接口版本通过。

**Architecture:** 平台加两样东西：`platform/ratelimit`（只用标准库的令牌桶，`AllowAll` 全有或全无，`Reserve` 可退回）和 `httpserver` 里按路由的失败闸门与限流，平台通过自己声明的 `Limiter` 接口使用桶，两个平台包互不导入；固定链多一个安全响应头的中间件。`identity` 按端口与适配器分层：`domain` 解析刷新令牌、按 M2 设计 3.5 的表判定续期；`app` 的 `Login`、`Refresh`、`Logout` 编排账户行锁、条件轮换和重复使用的撤销，共用签发令牌的 `Issuance`；`adapter/postgres` 提供查询，`adapter/argon2`、`adapter/signing` 加上校验，`adapter/http` 挂三个操作、模块自己的桶和续期的期限。`bootstrap` 建一个限流器、六个桶，分给平台和模块。

**Tech Stack:** Go 1.27.1、pgx v5.11.0、sqlc v1.31.1（`CGO_ENABLED=0`）、oapi-codegen v2.8.0、golang-jwt/jwt/v5 v5.3.1、golang.org/x/crypto v0.57.0、golangci-lint 2.13.2、PostgreSQL 18.6；Node 24、pnpm 11.10.0、Playwright 1.63.0。P2 不加新的依赖。

**Spec:** `docs/v0/M2-auth/specs/P2-sessions.md`（上级：`docs/v0/M2-auth/M2-design.md`）

## Global Constraints

- **Go 版本**：`server/go.mod` 和 `server/tools/go.mod` 保持 `go 1.27` 和 `toolchain go1.27.1`。P2 不执行 `go get`；万一执行了 `go get` 或 `go mod tidy`，之后用 `grep -n "^go \|^toolchain" server/go.mod server/tools/go.mod` 核对（golangci-lint 2.13.2 由 go1.27.0 构建，`go` 行更高就拒绝运行）。两个 `go.mod`、`go.sum` 在本 plan 结束时与 `fd69736` 相同。
- **依赖**：不加任何 Go 模块和 npm 包（spec 2.2）。
- **每个 Task 提交前**：`make lint-go` 两段都输出 `0 issues.`，`make test` 全部通过。有生成物的 Task（5、8、9）先执行 `make gen`，核对生成物的 SHA-256（`shasum -a 256`）和行数，**提交之后**执行 `make gen-check`（它用 `git status` 判断生成物是否已提交，提交之前必然报差异）。改了 `schema.gen.ts` 或 `e2e/` 的 Task（5、9、11）另执行 `make lint-web`；Task 11 另执行 `make knip`。Task 5、9、11 之后执行 `make e2e`。
- **生成的文件不手写、不从本 plan 复制**：执行生成命令，提交它的输出。表中是生成物的 SHA-256 和行数；对不上时停下来，说明某个输入与本 plan 不一致。
- **容器**：`make test` 和 `make e2e` 用自己的 testcontainers。开发库 `nerve-dev-db-1` 可以用，但不要停止或重建它，不要执行 `make dev-db-down`、`make dev-db-reset`。不要碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`）。
- **git**：每次 Bash 调用只执行一个 git 命令，不用 `;`、`&&`、`|` 串联 git；不用 `git -C`、`stash`、`clean`、`reset --hard`。`cd` 不与别的命令组合。
- **安装**：除了 Docker、Go、Node 不做任何全局安装；不执行 `corepack enable`（pnpm 已在 PATH 上）。
- **规则**：不写 `init()`，不用全局可变状态，构造函数显式传入依赖；不建 `utils`、`common`、`helpers` 包；一个文件只做一件事，不超过约 400 行；依赖只能向内；平台包之间不互相导入（规则 7）；不留没有使用者的代码（spec 第 3 节第 7 条）。
- **注释**：Go、TS 代码、SQL 查询和接口描述用英文；`server/configs/*.yaml` 的中文注释和中文文档照本 plan 原样。
- **代码块**：标为"新文件"或"完整内容"的块是**完整的文件内容**，照原样写入，不要改动（原型中逐字节运行过）；标为"差异"的块是对这个文件当前版本（`fd69736` 或前一个 Task 写的版本）的统一差异，照差异修改，改完的文件与原型逐字节相同。差异块可以存成文件后在仓库根目录用 `patch -p1 < <文件>` 应用（块里的路径是 `a/<路径>`、`b/<路径>`；拼 plan 的脚本已用 `patch -p1` 逐个核对过，48 个差异块都得出原型中的文件）。
- **过渡版本**：少数文件先在较早的 Task 写成过渡版本，较晚的 Task 再写成最终版本；每个过渡版本都在原型的逐 Task 复现中运行过（spec 附录 A）。
- **提交**：提交信息用英文，末尾加一行：`Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明。

## 文件结构

路径相对于仓库根目录。"生成"表示由命令生成并提交；"过渡"表示这个 Task 写过渡版本，后面的 Task 写最终版本。

| 文件 | 职责 | Task |
|---|---|---|
| `server/internal/platform/config/config.go`、`load.go`、`validate.go` 及三个测试（修改） | P2 的配置项；YAML 空值、越界数、列表型环境变量；启动校验 | 1 |
| `server/configs/config.yaml`、`config.test.yaml`、`embed_test.go`（修改） | 默认值；test 的桶调高 | 1 |
| `server/internal/platform/ratelimit/ratelimit.go`、`ratelimit_test.go` | 令牌桶 | 2 |
| `server/internal/platform/httpserver/middleware.go`、`middleware_test.go`、`server.go`（修改） | 固定链的安全响应头 | 3 |
| `server/internal/bootstrap/headers_test.go` | 接好线的程序的每个响应都带安全响应头 | 3 |
| `server/internal/platform/httpserver/clientip.go` | 客户端 IP、可信代理、IP 键 | 4 |
| `server/internal/platform/httpserver/clientip_test.go` | 客户端 IP 的测试 | 4（过渡）、5 |
| `server/internal/platform/httpserver/api.go`、`api_test.go`（修改） | 请求元信息的 IP 键；失败闸门；`NewAPI` 检查依赖 | 4（过渡）、5 |
| `server/internal/bootstrap/app.go`、`app_test.go`（修改） | 可信代理；限流器和六个桶 | 4（过渡）、5（过渡）、9 |
| `server/internal/platform/httpserver/limit.go` | `Limiter`、限流中间件、429 | 5 |
| `server/internal/platform/httpserver/problem.go`、`contract_test.go`（修改） | `rate_limited`；平台 problem 的契约测试 | 5 |
| `server/internal/platform/httpserver/apitest/problems.go`、`problems_test.go`（修改） | 测试卫生 | 5 |
| `api/common.yaml`、`api/modules/instance.yaml`（修改） | `rate_limited` 的说明；problem 响应的响应头 | 5 |
| `api/openapi.yaml`、`api/modules/identity.yaml`（修改） | 顶层 `rate_limited`；响应头；三个新操作 | 5（过渡）、9 |
| `server/internal/modules/identity/adapter/authn/authenticator.go`、`authenticator_test.go`、`app/authenticate_test.go`（修改） | `ExpiredCredential` | 5 |
| `server/internal/modules/instance/adapter/http/handler_test.go`（修改） | 新的 `APIConfig` | 5 |
| `server/internal/modules/identity/adapter/http/handler_test.go`（修改） | 新的 `APIConfig`；三个新操作的测试服务 | 5（过渡）、9 |
| `server/internal/platform/httpserver/apigen/components.gen.go`、`*/adapter/http/gen/server.gen.go`、`api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`（生成） | | 5、9 |
| `server/internal/modules/identity/domain/session.go`、`errors.go`、`session_test.go`（修改） | 刷新令牌的解析、续期判定、三个模块错误 | 6 |
| `server/internal/modules/identity/app/ports.go`、`register.go`、`register_test.go`、`fakes_test.go`（修改） | 端口；注册改用 `Issuance` | 7 |
| `server/internal/modules/identity/app/issuer.go`、`login.go`、`refresh.go`、`logout.go` 及三个测试 | 用例 | 7 |
| `server/internal/modules/identity/adapter/argon2/hasher.go`、`hasher_test.go`、`adapter/signing/mac.go`、`signing_test.go`（修改） | `Verify` | 7 |
| `server/internal/modules/identity/module.go`（修改） | 注册改用 `Issuance`；接上三个用例和模块的桶 | 7（过渡）、9 |
| `server/internal/modules/identity/adapter/postgres/queries/users.sql`、`sessions.sql`、`users.go`、`sessions.go`（修改） | 登录和续期的查询 | 8 |
| `server/internal/modules/identity/adapter/postgres/credentials_test.go`、`refresh_test.go` | 集成测试 | 8 |
| `server/internal/modules/identity/adapter/postgres/gen/users.sql.go`、`sessions.sql.go`（生成） | | 8 |
| `server/internal/modules/identity/adapter/http/handler.go`（修改）、`auth.go`、`me.go`、`limits.go` 及 `auth_test.go`、`limits_test.go` | 三个操作、模块的桶、续期的期限 | 9 |
| `server/internal/modules/identity/adapter/http/gen/bodyshape.gen.go`（生成） | | 9 |
| `server/internal/shared/error.go`、`error_test.go`、`server/internal/bootstrap/errors_test.go`（修改） | `shared.RateLimited` | 9 |
| `server/internal/bootstrap/sessions_test.go` | 续期和登录耗时的整程序测试 | 10 |
| `e2e/fixtures/auth.ts`、`e2e/fixtures/assert/identity.ts`（修改） | 登录、续期；会话的断言 | 11 |
| `e2e/stories/identity/a3-sign-in.spec.ts`、`a4-refresh.spec.ts`、`a5-refresh-reuse.spec.ts`、`a6-sign-out.spec.ts`、`a15-sign-in-limits.spec.ts` | 五个故事的接口版本 | 11 |
| `docs/v0/v0-design.md`、`docs/v0/M0-foundation/M0-design.md`、`docs/v0/plane-diff.md`、`README.md`、`docs/v0/M2-auth/handoffs/M0-P{2,5,6}-*.md`（修改） | 文档同步、交接 | 12 |

---

### Task 1: 配置：限流的桶、可信代理、续期期限；配置加固

**Files:**
- Modify: `server/internal/platform/config/config.go`、`load.go`、`validate.go`、`config_test.go`、`load_test.go`、`validate_test.go`
- Modify: `server/configs/config.yaml`、`config.test.yaml`、`embed_test.go`

**Interfaces:**
- Produces（spec 2.3）：
  - `config.ServerConfig.TrustedProxies []netip.Prefix`（`server.trusted_proxies`）；`config.AuthConfig.RefreshDeadline`（`auth.refresh_deadline`）；
  - `config.RateLimitConfig{IPv6PrefixLen int; Anonymous, AuthFailure, Authenticated, LoginIP, LoginIPEmail, RegisterIP BucketConfig}`、`config.BucketConfig{PerMinute, Burst int}`，`Config.RateLimit`（`ratelimit`）；
  - 加载：`rejectNulls`（YAML 中没有值的键报错）、`numberHook`（目标整数类型装不下的数、小数报错）、`listHook`（环境变量按逗号切成列表）、`StringToNetIPPrefixHookFunc`（CIDR）；
  - 校验：`refresh_deadline` 为正且加 `database.commit_timeout` 小于 8 秒；每个桶两项至少 1；`ipv6_prefix_len` 1–128；可信代理不是 IPv4 映射的 IPv6 前缀。
- 这是 P1 评审第 6 节交给 P2 的"配置加固"（spec 第 7 节）。

**Tests:**
- `load_test.go`：`TestLoadAppliesLayersInOrder` 覆盖新键和两个环境变量（`NERVE_SERVER__TRUSTED_PROXIES`、`NERVE_RATELIMIT__LOGIN_IP__BURST`）；新增 `TestLoadReadsTrustedProxies`（4 个）；`TestLoadErrors` 新增 7 个（YAML 布尔空值、整节空值、负数给无符号键、超出 `int32`、小数、环境变量的负数、畸形 CIDR），每个核对完整的报错原文。
- `validate_test.go`：`TestValidateReportsEveryInvalidKey` 覆盖 `auth.refresh_deadline`、`ratelimit.ipv6_prefix_len` 和六个桶的两项；`TestValidateCrossKeyRules` 新增 3 个（续期期限加提交期限达到 8 秒、IPv4 映射的可信代理、前缀长度 129）。
- `config_test.go`：`TestLogValueMasksDatabaseURL` 核对 `trusted_proxies`、`refresh_deadline` 和桶的日志。
- `server/configs/embed_test.go`：`TestBuiltInProfiles` 核对三个环境的桶（dev、prod 是默认值，test 全部调高）和空的可信代理。

- [ ] **Step 1: 配置类型、加载和校验**

`server/internal/platform/config/config.go`（完整内容）：

```go
// Package config loads, validates and describes nerve's configuration.
package config

import (
	"log/slog"
	"net/netip"
	"strings"
	"time"
)

// Profiles, selected with NERVE_ENV.
const (
	EnvDev  = "dev"
	EnvTest = "test"
	EnvProd = "prod"
)

// Config is the effective configuration of a nerve process.
type Config struct {
	// Env is the profile the configuration was loaded for. It comes from
	// NERVE_ENV and is not a configuration key.
	Env       string          `koanf:"-"`
	Server    ServerConfig    `koanf:"server"`
	Database  DatabaseConfig  `koanf:"database"`
	Auth      AuthConfig      `koanf:"auth"`
	RateLimit RateLimitConfig `koanf:"ratelimit"`
	Log       LogConfig       `koanf:"log"`
}

// ServerConfig configures the HTTP server. The timeouts bound the reads and
// writes on a connection: reading the request headers, reading the whole
// request (headers and body), and writing the response; idle keep-alive
// connections have a fixed timeout in httpserver. RequestTimeout bounds each
// API request's context, which write_timeout does not cancel.
type ServerConfig struct {
	Addr              string        `koanf:"addr"`
	ReadHeaderTimeout time.Duration `koanf:"read_header_timeout"`
	ReadTimeout       time.Duration `koanf:"read_timeout"`
	WriteTimeout      time.Duration `koanf:"write_timeout"`
	ShutdownTimeout   time.Duration `koanf:"shutdown_timeout"`
	RequestTimeout    time.Duration `koanf:"request_timeout"`
	MaxBodyBytes      int64         `koanf:"max_body_bytes"`
	// AddrFile, when set, receives the address the server listens on once it
	// does, e.g. for addr ":0".
	AddrFile string `koanf:"addr_file"`
	// TrustedProxies are the reverse proxies whose X-Forwarded-For names the
	// client (M2 design 3.10). Empty: the client is the connection's peer.
	TrustedProxies []netip.Prefix `koanf:"trusted_proxies"`
}

// DatabaseConfig configures the PostgreSQL pool and schema migrations.
type DatabaseConfig struct {
	URL         string `koanf:"url"`
	MaxConns    int32  `koanf:"max_conns"`
	AutoMigrate bool   `koanf:"auto_migrate"`
	// CommitTimeout bounds COMMIT and ROLLBACK, which the request deadline
	// does not cancel.
	CommitTimeout time.Duration `koanf:"commit_timeout"`
}

// AuthConfig configures accounts and credentials (M2 design 6.5).
type AuthConfig struct {
	SignupEnabled  bool          `koanf:"signup_enabled"`
	AccessTokenTTL time.Duration `koanf:"access_token_ttl"`
	SessionTTL     time.Duration `koanf:"session_ttl"`
	// RefreshDeadline bounds the statements of a refresh or a logout; with
	// database.commit_timeout it must end before the web client gives up on
	// a refresh (M2 design 3.5).
	RefreshDeadline time.Duration  `koanf:"refresh_deadline"`
	JWT             JWTConfig      `koanf:"jwt"`
	Password        PasswordConfig `koanf:"password"`
}

// JWTConfig locates the Ed25519 signing key.
type JWTConfig struct {
	// PrivateKeyFile is a PKCS#8 PEM Ed25519 private key. Required in prod;
	// empty elsewhere means an ephemeral key generated at startup.
	PrivateKeyFile string `koanf:"private_key_file"`
}

// PasswordConfig configures argon2id and how many hashes run at once.
type PasswordConfig struct {
	Argon2MemoryKiB     uint32        `koanf:"argon2_memory_kib"`
	Argon2Iterations    uint32        `koanf:"argon2_iterations"`
	Argon2Parallelism   uint8         `koanf:"argon2_parallelism"`
	MaxConcurrentHashes int           `koanf:"max_concurrent_hashes"`
	MaxWait             time.Duration `koanf:"max_wait"`
}

// RateLimitConfig sizes the rate-limit buckets (M2 design 3.10).
type RateLimitConfig struct {
	// IPv6PrefixLen is how much of an IPv6 client address the per-IP buckets
	// count by: a host usually holds a whole /64.
	IPv6PrefixLen int `koanf:"ipv6_prefix_len"`
	// Anonymous limits the public operations, by client IP.
	Anonymous BucketConfig `koanf:"anonymous"`
	// AuthFailure is the gate before authentication: requests whose token
	// fails, by client IP.
	AuthFailure BucketConfig `koanf:"auth_failure"`
	// Authenticated limits the operations that need a token, by credential.
	Authenticated BucketConfig `koanf:"authenticated"`
	// LoginIP and LoginIPEmail limit sign-in by client IP, and by client IP
	// and address; RegisterIP limits sign-up by client IP.
	LoginIP      BucketConfig `koanf:"login_ip"`
	LoginIPEmail BucketConfig `koanf:"login_ip_email"`
	RegisterIP   BucketConfig `koanf:"register_ip"`
}

// BucketConfig is a token bucket: it holds at most Burst units and gains
// PerMinute units a minute.
type BucketConfig struct {
	PerMinute int `koanf:"per_minute"`
	Burst     int `koanf:"burst"`
}

// LogValue renders the bucket as its two settings.
func (b BucketConfig) LogValue() slog.Value {
	return slog.GroupValue(slog.Int("per_minute", b.PerMinute), slog.Int("burst", b.Burst))
}

// LogConfig configures the process logger.
type LogConfig struct {
	Level  string `koanf:"level"`  // debug, info, warn or error
	Format string `koanf:"format"` // text or json
}

// LogValue renders the configuration for logs with secrets masked, so the
// effective configuration can be logged at startup. Only the keys listed here
// reach the log; every *_file key logs whether it is set, never its path.
func (c Config) LogValue() slog.Value {
	proxies := make([]string, len(c.Server.TrustedProxies))
	for i, p := range c.Server.TrustedProxies {
		proxies[i] = p.String()
	}
	return slog.GroupValue(
		slog.String("env", c.Env),
		slog.Group("server",
			slog.String("addr", c.Server.Addr),
			slog.Duration("read_header_timeout", c.Server.ReadHeaderTimeout),
			slog.Duration("read_timeout", c.Server.ReadTimeout),
			slog.Duration("write_timeout", c.Server.WriteTimeout),
			slog.Duration("shutdown_timeout", c.Server.ShutdownTimeout),
			slog.Duration("request_timeout", c.Server.RequestTimeout),
			slog.Int64("max_body_bytes", c.Server.MaxBodyBytes),
			slog.Bool("addr_file_set", c.Server.AddrFile != ""),
			slog.String("trusted_proxies", strings.Join(proxies, ",")),
		),
		slog.Any("database", c.Database),
		slog.Group("auth",
			slog.Bool("signup_enabled", c.Auth.SignupEnabled),
			slog.Duration("access_token_ttl", c.Auth.AccessTokenTTL),
			slog.Duration("session_ttl", c.Auth.SessionTTL),
			slog.Duration("refresh_deadline", c.Auth.RefreshDeadline),
			slog.Group("jwt",
				slog.Bool("private_key_file_set", c.Auth.JWT.PrivateKeyFile != ""),
			),
			slog.Group("password",
				slog.Uint64("argon2_memory_kib", uint64(c.Auth.Password.Argon2MemoryKiB)),
				slog.Uint64("argon2_iterations", uint64(c.Auth.Password.Argon2Iterations)),
				slog.Uint64("argon2_parallelism", uint64(c.Auth.Password.Argon2Parallelism)),
				slog.Int("max_concurrent_hashes", c.Auth.Password.MaxConcurrentHashes),
				slog.Duration("max_wait", c.Auth.Password.MaxWait),
			),
		),
		slog.Group("ratelimit",
			slog.Int("ipv6_prefix_len", c.RateLimit.IPv6PrefixLen),
			slog.Any("anonymous", c.RateLimit.Anonymous),
			slog.Any("auth_failure", c.RateLimit.AuthFailure),
			slog.Any("authenticated", c.RateLimit.Authenticated),
			slog.Any("login_ip", c.RateLimit.LoginIP),
			slog.Any("login_ip_email", c.RateLimit.LoginIPEmail),
			slog.Any("register_ip", c.RateLimit.RegisterIP),
		),
		slog.Group("log",
			slog.String("level", c.Log.Level),
			slog.String("format", c.Log.Format),
		),
	)
}

// redacted stands in for a secret in log output.
const redacted = "xxxxx"

// LogValue renders the database settings with the URL masked as a whole, so
// they are safe to log on their own too. pgx parses the URL with its own
// libpq-compatible grammar, which accepts forms that other parsers read
// differently, so masking only the password another parser finds could leak
// the rest; log the target from the parsed pool configuration instead.
func (d DatabaseConfig) LogValue() slog.Value {
	url := ""
	if d.URL != "" {
		url = redacted
	}
	return slog.GroupValue(
		slog.String("url", url),
		slog.Int("max_conns", int(d.MaxConns)),
		slog.Bool("auto_migrate", d.AutoMigrate),
		slog.Duration("commit_timeout", d.CommitTimeout),
	)
}
```

`server/internal/platform/config/load.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/platform/config/load.go
+++ b/server/internal/platform/config/load.go
@@ -4,6 +4,7 @@
 	"errors"
 	"fmt"
 	"io/fs"
+	"math"
 	"os"
 	"path/filepath"
 	"reflect"
@@ -92,6 +93,9 @@
 	if err := k.Load(environ, nil); err != nil {
 		return Config{}, fmt.Errorf("read environment: %w", err)
 	}
+	if err := rejectNulls(k); err != nil {
+		return Config{}, fmt.Errorf("invalid configuration:\n%w", err)
+	}
 
 	cfg := Config{Env: profile}
 	if err := decode(k, &cfg); err != nil {
@@ -142,12 +146,26 @@
 	return strings.ToLower(strings.ReplaceAll(key, envKeySep, ".")), value
 }
 
+// rejectNulls fails every key that a layer left without a value. The decoder
+// would skip it and leave the key at its zero value: signup_enabled: in a
+// YAML file would silently mean false, whatever the layers below it say.
+func rejectNulls(k *koanf.Koanf) error {
+	var errs []error
+	for _, key := range k.Keys() {
+		if k.Get(key) == nil {
+			errs = append(errs, fmt.Errorf("%s: must not be null (a key without a value in YAML)", key))
+		}
+	}
+	return errors.Join(errs...)
+}
+
 // decode copies the merged layers into cfg. Unknown keys are errors, so a typo
 // never silently falls back to a default.
 func decode(k *koanf.Koanf, cfg *Config) error {
 	return k.UnmarshalWithConf("", cfg, koanf.UnmarshalConf{
 		DecoderConfig: &mapstructure.DecoderConfig{
-			DecodeHook:       mapstructure.ComposeDecodeHookFunc(emptyValueHook, durationHook),
+			DecodeHook: mapstructure.ComposeDecodeHookFunc(
+				emptyValueHook, durationHook, numberHook, listHook, mapstructure.StringToNetIPPrefixHookFunc()),
 			ErrorUnused:      true,
 			WeaklyTypedInput: true, // environment values are strings
 		},
@@ -181,3 +199,59 @@
 	}
 	return time.ParseDuration(s)
 }
+
+// numberHook rejects a YAML number that the key's integer type cannot hold.
+// Weakly typed decoding would convert it anyway: argon2_memory_kib: -1 would
+// wrap to 4294967295 KiB, max_conns: 5000000000 would lose its high bits and
+// 1.5 would become 1. Environment values are strings, which the decoder
+// parses with the type's bit size, so they pass through here.
+func numberHook(_ reflect.Type, to reflect.Type, data any) (any, error) {
+	var lo int64
+	var hi uint64
+	switch to.Kind() {
+	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
+		lo, hi = -1<<(to.Bits()-1), 1<<(to.Bits()-1)-1
+	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
+		hi = math.MaxUint64 >> (64 - to.Bits())
+	default:
+		return data, nil
+	}
+	if to == reflect.TypeFor[time.Duration]() { // durationHook has decoded it
+		return data, nil
+	}
+	var fits bool
+	switch v := reflect.ValueOf(data); {
+	case v.CanInt():
+		fits = v.Int() >= lo && (v.Int() < 0 || uint64(v.Int()) <= hi)
+	case v.CanUint():
+		fits = v.Uint() <= hi
+	case v.CanFloat():
+		f := v.Float()
+		fits = f == math.Trunc(f) && f >= float64(lo) && f <= float64(hi)
+	default:
+		return data, nil
+	}
+	if !fits {
+		return nil, fmt.Errorf("must be a whole number from %d to %d, got %v", lo, hi, data)
+	}
+	return data, nil
+}
+
+// listHook splits a string into a list at its commas, so that an environment
+// variable can set a list key, e.g.
+// NERVE_SERVER__TRUSTED_PROXIES=10.0.0.0/8,fd00::/8. The empty string is the
+// empty list.
+func listHook(from reflect.Type, to reflect.Type, data any) (any, error) {
+	if from.Kind() != reflect.String || to.Kind() != reflect.Slice {
+		return data, nil
+	}
+	s := reflect.ValueOf(data).String()
+	if s == "" {
+		return []string{}, nil
+	}
+	items := strings.Split(s, ",")
+	for i, item := range items {
+		items[i] = strings.TrimSpace(item)
+	}
+	return items, nil
+}
```

`server/internal/platform/config/validate.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/platform/config/validate.go
+++ b/server/internal/platform/config/validate.go
@@ -5,8 +5,14 @@
 	"fmt"
 	"log/slog"
 	"net"
+	"time"
 )
 
+// webRefreshTimeout is how long the web client waits for a refresh (M2 design
+// 7.1). The server must have finished a refresh before that: committed,
+// rolled back, or given up on its COMMIT (3.5).
+const webRefreshTimeout = 8 * time.Second
+
 // validate reports every invalid key at once, one "key: problem" line each.
 func (c Config) validate() error {
 	var errs []error
@@ -45,6 +51,13 @@
 	if c.Server.MaxBodyBytes < 1 {
 		fail("server.max_body_bytes", "must be at least 1, got %d", c.Server.MaxBodyBytes)
 	}
+	for _, p := range c.Server.TrustedProxies {
+		// Client addresses are compared unmapped (M2 design 3.10), so such a
+		// prefix would silently match nothing.
+		if p.Addr().Is4In6() {
+			fail("server.trusted_proxies", "%s is an IPv4-mapped IPv6 prefix, which no address matches: write the IPv4 prefix", p)
+		}
+	}
 	if c.Database.URL == "" {
 		fail("database.url", "is required")
 	}
@@ -55,6 +68,14 @@
 		fail("database.commit_timeout", "must be positive, got %s", c.Database.CommitTimeout)
 	}
 	c.Auth.validate(c.Env, fail)
+	switch {
+	case c.Auth.RefreshDeadline <= 0:
+		fail("auth.refresh_deadline", "must be positive, got %s", c.Auth.RefreshDeadline)
+	case c.Auth.RefreshDeadline+c.Database.CommitTimeout >= webRefreshTimeout:
+		fail("auth.refresh_deadline", "plus database.commit_timeout (%s) must be less than %s, the web client's refresh timeout, got %s",
+			c.Database.CommitTimeout, webRefreshTimeout, c.Auth.RefreshDeadline)
+	}
+	c.RateLimit.validate(fail)
 	var level slog.Level
 	if err := level.UnmarshalText([]byte(c.Log.Level)); err != nil {
 		fail("log.level", "must be one of debug, info, warn, error, got %q", c.Log.Level)
@@ -98,3 +119,27 @@
 		fail("auth.password.max_wait", "must be positive, got %s", p.MaxWait)
 	}
 }
+
+func (r RateLimitConfig) validate(fail func(key, format string, args ...any)) {
+	if r.IPv6PrefixLen < 1 || r.IPv6PrefixLen > 128 {
+		fail("ratelimit.ipv6_prefix_len", "must be from 1 to 128, got %d", r.IPv6PrefixLen)
+	}
+	for _, b := range []struct {
+		name   string
+		bucket BucketConfig
+	}{
+		{"anonymous", r.Anonymous},
+		{"auth_failure", r.AuthFailure},
+		{"authenticated", r.Authenticated},
+		{"login_ip", r.LoginIP},
+		{"login_ip_email", r.LoginIPEmail},
+		{"register_ip", r.RegisterIP},
+	} {
+		if b.bucket.PerMinute < 1 {
+			fail("ratelimit."+b.name+".per_minute", "must be at least 1, got %d", b.bucket.PerMinute)
+		}
+		if b.bucket.Burst < 1 {
+			fail("ratelimit."+b.name+".burst", "must be at least 1, got %d", b.bucket.Burst)
+		}
+	}
+}
```

- [ ] **Step 2: 测试**

`server/internal/platform/config/config_test.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/platform/config/config_test.go
+++ b/server/internal/platform/config/config_test.go
@@ -3,6 +3,7 @@
 import (
 	"bytes"
 	"log/slog"
+	"net/netip"
 	"strings"
 	"testing"
 )
@@ -37,6 +38,16 @@
 		"config.auth.password.argon2_parallelism=1",
 		"config.auth.password.max_concurrent_hashes=4",
 		"config.auth.password.max_wait=2s",
+		"config.auth.refresh_deadline=4s",
+		"config.server.trusted_proxies=10.0.0.0/8,2001:db8::/32",
+		"config.ratelimit.ipv6_prefix_len=64",
+		"config.ratelimit.anonymous.per_minute=600",
+		"config.ratelimit.anonymous.burst=100",
+		"config.ratelimit.auth_failure.per_minute=60",
+		"config.ratelimit.authenticated.burst=200",
+		"config.ratelimit.login_ip.per_minute=30",
+		"config.ratelimit.login_ip_email.burst=5",
+		"config.ratelimit.register_ip.per_minute=10",
 		"config.log.format=json",
 	} {
 		if !strings.Contains(out, want) {
@@ -49,6 +60,7 @@
 // M2 design 3.7): the path of a key file tells where secrets live.
 func TestLogValueHidesFilePaths(t *testing.T) {
 	cfg := validConfig()
+	cfg.Server.TrustedProxies = []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("fd00::/8")}
 	cfg.Server.AddrFile = "/run/nerve/addr-secret-dir"
 	cfg.Auth.JWT.PrivateKeyFile = "/etc/nerve/secret-dir/jwt.pem"
 	var buf bytes.Buffer
@@ -58,7 +70,11 @@
 	if strings.Contains(out, "secret-dir") {
 		t.Errorf("log output shows a file path: %s", out)
 	}
-	for _, want := range []string{"config.server.addr_file_set=true", "config.auth.jwt.private_key_file_set=true"} {
+	for _, want := range []string{
+		"config.server.addr_file_set=true",
+		"config.auth.jwt.private_key_file_set=true",
+		"config.server.trusted_proxies=10.0.0.0/8,fd00::/8", // not secret: the log should say whom nerve trusts
+	} {
 		if !strings.Contains(out, want) {
 			t.Errorf("log output lacks %q: %s", want, out)
 		}
```

`server/internal/platform/config/load_test.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/platform/config/load_test.go
+++ b/server/internal/platform/config/load_test.go
@@ -1,8 +1,10 @@
 package config
 
 import (
+	"net/netip"
 	"os"
 	"path/filepath"
+	"reflect"
 	"strings"
 	"testing"
 	"testing/fstest"
@@ -19,6 +21,7 @@
   request_timeout: 15s
   max_body_bytes: 1048576
   addr_file: ""
+  trusted_proxies: []
 database:
   url: ""
   max_conns: 10
@@ -28,6 +31,7 @@
   signup_enabled: false
   access_token_ttl: 15m
   session_ttl: 720h
+  refresh_deadline: 4s
   jwt:
     private_key_file: ""
   password:
@@ -36,6 +40,14 @@
     argon2_parallelism: 1
     max_concurrent_hashes: 4
     max_wait: 2s
+ratelimit:
+  ipv6_prefix_len: 64
+  anonymous: {per_minute: 600, burst: 100}
+  auth_failure: {per_minute: 60, burst: 60}
+  authenticated: {per_minute: 1200, burst: 200}
+  login_ip: {per_minute: 30, burst: 10}
+  login_ip_email: {per_minute: 10, burst: 5}
+  register_ip: {per_minute: 10, burst: 5}
 log:
   level: info
   format: json
@@ -72,6 +84,8 @@
 			"NERVE_DATABASE__AUTO_MIGRATE=false",
 			"NERVE_AUTH__SIGNUP_ENABLED=true",
 			"NERVE_AUTH__PASSWORD__ARGON2_MEMORY_KIB=64",
+			"NERVE_SERVER__TRUSTED_PROXIES=10.0.0.0/8,fd00::/8",
+			"NERVE_RATELIMIT__LOGIN_IP__BURST=3",
 		},
 		LocalFile: local,
 	})
@@ -89,6 +103,7 @@
 			ShutdownTimeout:   40 * time.Second,
 			RequestTimeout:    15 * time.Second,
 			MaxBodyBytes:      1048576,
+			TrustedProxies:    []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("fd00::/8")}, // environment
 		},
 		Database: DatabaseConfig{
 			URL:           "postgres://embedded-dev", // built-in config.dev.yaml
@@ -97,9 +112,10 @@
 			CommitTimeout: 2 * time.Second,
 		},
 		Auth: AuthConfig{
-			SignupEnabled:  true, // environment
-			AccessTokenTTL: 15 * time.Minute,
-			SessionTTL:     720 * time.Hour,
+			SignupEnabled:   true, // environment
+			AccessTokenTTL:  15 * time.Minute,
+			SessionTTL:      720 * time.Hour,
+			RefreshDeadline: 4 * time.Second,
 			Password: PasswordConfig{
 				Argon2MemoryKiB:     64, // environment
 				Argon2Iterations:    2,
@@ -108,10 +124,48 @@
 				MaxWait:             2 * time.Second,
 			},
 		},
+		RateLimit: RateLimitConfig{
+			IPv6PrefixLen: 64,
+			Anonymous:     BucketConfig{PerMinute: 600, Burst: 100},
+			AuthFailure:   BucketConfig{PerMinute: 60, Burst: 60},
+			Authenticated: BucketConfig{PerMinute: 1200, Burst: 200},
+			LoginIP:       BucketConfig{PerMinute: 30, Burst: 3}, // environment
+			LoginIPEmail:  BucketConfig{PerMinute: 10, Burst: 5},
+			RegisterIP:    BucketConfig{PerMinute: 10, Burst: 5},
+		},
 		Log: LogConfig{Level: "debug", Format: "text"},
 	}
-	if cfg != want {
+	if !reflect.DeepEqual(cfg, want) {
 		t.Errorf("Load() =\n%+v\nwant\n%+v", cfg, want)
+	}
+}
+
+// A list key comes from YAML as a list and from the environment as one
+// comma-separated value; the empty value is the empty list.
+func TestLoadReadsTrustedProxies(t *testing.T) {
+	yaml := "server:\n  trusted_proxies: [192.0.2.0/24, \"2001:db8::/32\"]\n"
+	tests := []struct {
+		name    string
+		profile string
+		environ []string
+		want    []netip.Prefix
+	}{
+		{"from YAML", yaml, nil, []netip.Prefix{netip.MustParsePrefix("192.0.2.0/24"), netip.MustParsePrefix("2001:db8::/32")}},
+		{"from the environment", "", []string{"NERVE_SERVER__TRUSTED_PROXIES= 192.0.2.0/24 ,2001:db8::/32"},
+			[]netip.Prefix{netip.MustParsePrefix("192.0.2.0/24"), netip.MustParsePrefix("2001:db8::/32")}},
+		{"emptied by the environment", yaml, []string{"NERVE_SERVER__TRUSTED_PROXIES="}, []netip.Prefix{}},
+		{"none", "", nil, []netip.Prefix{}},
+	}
+	for _, tt := range tests {
+		t.Run(tt.name, func(t *testing.T) {
+			cfg, err := Load(Sources{
+				Embedded: embedded(map[string]string{"test": tt.profile}),
+				Environ:  append([]string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://x"}, tt.environ...),
+			})
+			if err != nil || !reflect.DeepEqual(cfg.Server.TrustedProxies, tt.want) {
+				t.Errorf("Load() = %#v, %v; want %v", cfg.Server.TrustedProxies, err, tt.want)
+			}
+		})
 	}
 }
 
@@ -233,6 +287,50 @@
 			environ: []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://x", "NERVE_AUTH__SESSION_TTL="},
 			want:    "'auth.session_ttl' must not be empty",
 		},
+		// A YAML key without a value would leave its key at the zero value,
+		// whatever the layers below it say (M2/P1 review M2).
+		{
+			name:    "boolean without a value in YAML",
+			environ: []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://x"},
+			dirFile: "auth:\n  signup_enabled:\n",
+			want:    "invalid configuration:\nauth.signup_enabled: must not be null (a key without a value in YAML)",
+		},
+		{
+			name:    "section without a value in YAML",
+			environ: []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://x"},
+			dirFile: "ratelimit:\n",
+			want:    "invalid configuration:\nratelimit: must not be null (a key without a value in YAML)",
+		},
+		// Weakly typed decoding would wrap or cut a number its type cannot
+		// hold (M2/P1 review M2).
+		{
+			name:    "negative number for an unsigned key",
+			environ: []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://x"},
+			dirFile: "auth:\n  password:\n    argon2_memory_kib: -1\n",
+			want:    "'auth.password.argon2_memory_kib' must be a whole number from 0 to 4294967295, got -1",
+		},
+		{
+			name:    "number too large for its key",
+			environ: []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://x"},
+			dirFile: "database:\n  max_conns: 5000000000\n",
+			want:    "'database.max_conns' must be a whole number from -2147483648 to 2147483647, got 5000000000",
+		},
+		{
+			name:    "fraction for a whole number",
+			environ: []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://x"},
+			dirFile: "ratelimit:\n  login_ip:\n    burst: 1.5\n",
+			want:    "'ratelimit.login_ip.burst' must be a whole number from -9223372036854775808 to 9223372036854775807, got 1.5",
+		},
+		{
+			name:    "negative number for an unsigned key in the environment",
+			environ: []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://x", "NERVE_AUTH__PASSWORD__ARGON2_ITERATIONS=-1"},
+			want:    "'auth.password.argon2_iterations' cannot parse value as 'uint32'",
+		},
+		{
+			name:    "malformed CIDR",
+			environ: []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=postgres://x", "NERVE_SERVER__TRUSTED_PROXIES=10.0.0.1"},
+			want:    "'server.trusted_proxies[0]' netip.ParsePrefix: no '/'",
+		},
 	}
 	for _, tt := range tests {
 		t.Run(tt.name, func(t *testing.T) {
```

`server/internal/platform/config/validate_test.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/platform/config/validate_test.go
+++ b/server/internal/platform/config/validate_test.go
@@ -1,6 +1,7 @@
 package config
 
 import (
+	"net/netip"
 	"strings"
 	"testing"
 	"time"
@@ -17,12 +18,14 @@
 			ShutdownTimeout:   20 * time.Second,
 			RequestTimeout:    15 * time.Second,
 			MaxBodyBytes:      1 << 20,
+			TrustedProxies:    []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("2001:db8::/32")},
 		},
 		Database: DatabaseConfig{URL: "postgres://nerve:secret@localhost:5432/nerve", MaxConns: 10, CommitTimeout: 2 * time.Second},
 		Auth: AuthConfig{
-			SignupEnabled:  true,
-			AccessTokenTTL: 15 * time.Minute,
-			SessionTTL:     720 * time.Hour,
+			SignupEnabled:   true,
+			AccessTokenTTL:  15 * time.Minute,
+			SessionTTL:      720 * time.Hour,
+			RefreshDeadline: 4 * time.Second,
 			Password: PasswordConfig{
 				Argon2MemoryKiB:     19456,
 				Argon2Iterations:    2,
@@ -31,6 +34,15 @@
 				MaxWait:             2 * time.Second,
 			},
 		},
+		RateLimit: RateLimitConfig{
+			IPv6PrefixLen: 64,
+			Anonymous:     BucketConfig{PerMinute: 600, Burst: 100},
+			AuthFailure:   BucketConfig{PerMinute: 60, Burst: 60},
+			Authenticated: BucketConfig{PerMinute: 1200, Burst: 200},
+			LoginIP:       BucketConfig{PerMinute: 30, Burst: 10},
+			LoginIPEmail:  BucketConfig{PerMinute: 10, Burst: 5},
+			RegisterIP:    BucketConfig{PerMinute: 10, Burst: 5},
+		},
 		Log: LogConfig{Level: "info", Format: "json"},
 	}
 }
@@ -71,6 +83,20 @@
 		"auth.password.argon2_memory_kib: must be at least 8 per lane (8), got 0",
 		"auth.password.max_concurrent_hashes: must be at least 1, got 0",
 		"auth.password.max_wait: must be positive, got 0s",
+		"auth.refresh_deadline: must be positive, got 0s",
+		"ratelimit.ipv6_prefix_len: must be from 1 to 128, got 0",
+		"ratelimit.anonymous.per_minute: must be at least 1, got 0",
+		"ratelimit.anonymous.burst: must be at least 1, got 0",
+		"ratelimit.auth_failure.per_minute: must be at least 1, got 0",
+		"ratelimit.auth_failure.burst: must be at least 1, got 0",
+		"ratelimit.authenticated.per_minute: must be at least 1, got 0",
+		"ratelimit.authenticated.burst: must be at least 1, got 0",
+		"ratelimit.login_ip.per_minute: must be at least 1, got 0",
+		"ratelimit.login_ip.burst: must be at least 1, got 0",
+		"ratelimit.login_ip_email.per_minute: must be at least 1, got 0",
+		"ratelimit.login_ip_email.burst: must be at least 1, got 0",
+		"ratelimit.register_ip.per_minute: must be at least 1, got 0",
+		"ratelimit.register_ip.burst: must be at least 1, got 0",
 		`log.level: must be one of debug, info, warn, error, got "verbose"`,
 		`log.format: must be text or json, got "xml"`,
 	}
@@ -115,6 +141,26 @@
 			change: func(c *Config) { c.Env = EnvProd },
 			want:   "auth.jwt.private_key_file: is required in prod: a PKCS#8 PEM Ed25519 private key, e.g. from openssl genpkey -algorithm ed25519",
 		},
+		{
+			// The server must be done with a refresh before the web client
+			// gives up on it after 8 s (M2 design 3.5, 7.1).
+			name:   "refresh deadline and commit timeout reach the web client's timeout",
+			change: func(c *Config) { c.Auth.RefreshDeadline = 6 * time.Second },
+			want:   "auth.refresh_deadline: plus database.commit_timeout (2s) must be less than 8s, the web client's refresh timeout, got 6s",
+		},
+		{
+			// The client IP is unmapped before it is compared (M2 design 3.10).
+			name: "IPv4-mapped trusted proxy",
+			change: func(c *Config) {
+				c.Server.TrustedProxies = append(c.Server.TrustedProxies, netip.MustParsePrefix("::ffff:10.0.0.0/104"))
+			},
+			want: "server.trusted_proxies: ::ffff:10.0.0.0/104 is an IPv4-mapped IPv6 prefix, which no address matches: write the IPv4 prefix",
+		},
+		{
+			name:   "IPv6 prefix longer than an address",
+			change: func(c *Config) { c.RateLimit.IPv6PrefixLen = 129 },
+			want:   "ratelimit.ipv6_prefix_len: must be from 1 to 128, got 129",
+		},
 	}
 	for _, tt := range tests {
 		t.Run(tt.name, func(t *testing.T) {
```

- [ ] **Step 3: 内嵌的配置文件**

`server/configs/config.yaml`（完整内容）：

```yaml
# 基础配置：列出所有配置项及其默认值。各环境的覆盖项在 config.<env>.yaml 中。
# 任何一项都可以用环境变量覆盖：NERVE_ 加上配置路径，层级之间用双下划线，
# 例如 database.url 对应 NERVE_DATABASE__URL。列表用逗号分隔，
# 例如 NERVE_SERVER__TRUSTED_PROXIES=10.0.0.0/8,fd00::/8。
server:
  addr: ":8080"
  # 连接上的读写都有上限：读请求头、读整个请求（含请求体）、写响应。
  # 上传这类确实要更久的接口，由自己的 handler 单独放宽，不要调大全局值。
  # handler 自身的执行时间不受这些上限约束，它里面的阻塞调用要自带期限。
  read_header_timeout: 5s
  read_timeout: 30s
  write_timeout: 60s
  shutdown_timeout: 20s
  # 每个接口请求的期限：到期取消请求的 context，数据库调用随之结束。必须短于 write_timeout
  request_timeout: 15s
  # JSON 接口的请求体上限（字节），超出时 413
  max_body_bytes: 1048576
  # 非空时，监听成功后把实际地址写进这个文件（端到端测试用 127.0.0.1:0 监听）
  addr_file: ""
  # 可信的反向代理（CIDR 列表）。连接的对端在其中时，客户端 IP 取自 X-Forwarded-For；
  # 为空时就是连接的对端。前面有反向代理（例如 Caddy）时必须配置，否则所有人都按代理的地址限流
  trusted_proxies: []

database:
  # 必须提供。dev 环境写在 config.dev.yaml 中；test 和 prod 通过 NERVE_DATABASE__URL 提供。
  url: ""
  max_conns: 10
  # nerve serve 启动前是否自动执行迁移
  auto_migrate: true
  # COMMIT、ROLLBACK 自己的期限：它们不随请求期限取消
  commit_timeout: 2s

auth:
  # 是否开放注册。基础配置（也就是 prod）关闭；config.dev.yaml、config.test.yaml 覆盖为 true。
  # 关闭时第一个账户用 nerve users create 创建（M2/P3 加入；在那之前临时设 NERVE_AUTH__SIGNUP_ENABLED=true）
  signup_enabled: false
  access_token_ttl: 15m
  # 会话从登录起算的期限（30 天），续期不延长
  session_ttl: 720h
  # 续期、退出的语句期限；加上 database.commit_timeout 必须短于前端续期请求的 8 秒超时
  refresh_deadline: 4s
  jwt:
    # PKCS#8 PEM 格式的 Ed25519 私钥文件：openssl genpkey -algorithm ed25519 -out nerve-jwt.pem
    # prod 必填；dev、test 为空时，启动时生成一把临时密钥（重启后旧的访问令牌失效）
    private_key_file: ""
  password:
    # argon2id 的参数（OWASP 推荐的最低配置）
    argon2_memory_kib: 19456
    argon2_iterations: 2
    argon2_parallelism: 1
    # 同时进行的哈希计算上限；拿不到名额时最多等 max_wait，然后 503 server_busy
    max_concurrent_hashes: 4
    max_wait: 2s

# 限流的桶：每分钟补充 per_minute 次，最多攒 burst 次；超出时 429 rate_limited，带 Retry-After
ratelimit:
  # IPv6 的客户端按这个长度的前缀计数：一台主机通常拿到整个 /64
  ipv6_prefix_len: 64
  # 公开操作（注册、登录、续期、退出、实例配置），按客户端 IP
  anonymous: {per_minute: 600, burst: 100}
  # 认证之前的失败闸门：带了令牌、认证却失败的请求，按客户端 IP
  auth_failure: {per_minute: 60, burst: 60}
  # 需要令牌的操作，按凭证（会话或个人访问令牌）
  authenticated: {per_minute: 1200, burst: 200}
  # 登录：按客户端 IP，以及按客户端 IP 加邮箱，两个桶全有或全无
  login_ip: {per_minute: 30, burst: 10}
  login_ip_email: {per_minute: 10, burst: 5}
  # 注册：按客户端 IP
  register_ip: {per_minute: 10, burst: 5}

log:
  level: info    # debug | info | warn | error
  format: json   # text | json
```

`server/configs/config.test.yaml`（完整内容）：

```yaml
# 测试环境（NERVE_ENV=test）：集成测试和端到端测试使用。数据库地址通过 NERVE_DATABASE__URL 提供。
auth:
  # 测试环境开放注册（prod 默认关闭，见 config.yaml）
  signup_enabled: true
  password:
    # 测试用最低的 argon2 参数，让大量注册的测试不被哈希拖慢
    argon2_memory_kib: 64
    argon2_iterations: 1

# 测试从同一个 IP 注册、登录大量账户，限流全部调高；测限流的故事（A15）另起一个限流很低的 nerve
ratelimit:
  anonymous: {per_minute: 600000, burst: 100000}
  auth_failure: {per_minute: 600000, burst: 100000}
  authenticated: {per_minute: 600000, burst: 100000}
  login_ip: {per_minute: 600000, burst: 100000}
  login_ip_email: {per_minute: 600000, burst: 100000}
  register_ip: {per_minute: 600000, burst: 100000}

log:
  level: warn
  format: text
```

`server/configs/embed_test.go`（完整内容）：

```go
package configs_test

import (
	"net/netip"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/configs"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

// The default buckets (M2 design 3.10), and the test profile's, raised so
// that one IP can sign many accounts up and in.
var (
	defaultLimits = config.RateLimitConfig{
		IPv6PrefixLen: 64,
		Anonymous:     config.BucketConfig{PerMinute: 600, Burst: 100},
		AuthFailure:   config.BucketConfig{PerMinute: 60, Burst: 60},
		Authenticated: config.BucketConfig{PerMinute: 1200, Burst: 200},
		LoginIP:       config.BucketConfig{PerMinute: 30, Burst: 10},
		LoginIPEmail:  config.BucketConfig{PerMinute: 10, Burst: 5},
		RegisterIP:    config.BucketConfig{PerMinute: 10, Burst: 5},
	}
	high       = config.BucketConfig{PerMinute: 600000, Burst: 100000}
	testLimits = config.RateLimitConfig{
		IPv6PrefixLen: 64, Anonymous: high, AuthFailure: high, Authenticated: high, LoginIP: high, LoginIPEmail: high, RegisterIP: high,
	}
)

func TestBuiltInProfiles(t *testing.T) {
	const devURL = "postgres://nerve:nerve@localhost:55432/nerve?sslmode=disable"
	tests := []struct {
		env         string
		addr        string // expected server.addr
		url         string // expected database.url
		autoMigrate bool
		signup      bool   // auth.signup_enabled: closed in prod unless overridden (M2 design decision 2)
		argon2      uint32 // auth.password.argon2_memory_kib
		iterations  uint32 // auth.password.argon2_iterations
		keyFile     string // auth.jwt.private_key_file
		limits      config.RateLimitConfig
		level       string
		format      string
	}{
		{env: "dev", addr: "127.0.0.1:8080", url: devURL, autoMigrate: true, signup: true, argon2: 19456, iterations: 2, limits: defaultLimits, level: "debug", format: "text"},
		{env: "test", addr: ":8080", url: "postgres://from-env", autoMigrate: true, signup: true, argon2: 64, iterations: 1, limits: testLimits, level: "warn", format: "text"},
		{env: "prod", addr: ":8080", url: "postgres://from-env", autoMigrate: false, signup: false, argon2: 19456, iterations: 2, keyFile: "/etc/nerve/jwt.pem", limits: defaultLimits, level: "info", format: "json"},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			environ := []string{"NERVE_ENV=" + tt.env}
			if tt.env != "dev" {
				environ = append(environ, "NERVE_DATABASE__URL=postgres://from-env")
			}
			if tt.keyFile != "" {
				environ = append(environ, "NERVE_AUTH__JWT__PRIVATE_KEY_FILE="+tt.keyFile)
			}
			cfg, err := config.Load(config.Sources{Embedded: configs.FS(), Environ: environ})
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			want := config.Config{
				Env: tt.env,
				Server: config.ServerConfig{
					Addr:              tt.addr,
					ReadHeaderTimeout: 5 * time.Second,
					ReadTimeout:       30 * time.Second,
					WriteTimeout:      60 * time.Second,
					ShutdownTimeout:   20 * time.Second,
					RequestTimeout:    15 * time.Second,
					MaxBodyBytes:      1 << 20,
					TrustedProxies:    []netip.Prefix{}, // none: the client is the connection's peer
				},
				Database: config.DatabaseConfig{URL: tt.url, MaxConns: 10, AutoMigrate: tt.autoMigrate, CommitTimeout: 2 * time.Second},
				Auth: config.AuthConfig{
					SignupEnabled:   tt.signup,
					AccessTokenTTL:  15 * time.Minute,
					SessionTTL:      30 * 24 * time.Hour,
					RefreshDeadline: 4 * time.Second,
					JWT:             config.JWTConfig{PrivateKeyFile: tt.keyFile},
					Password: config.PasswordConfig{
						Argon2MemoryKiB:     tt.argon2,
						Argon2Iterations:    tt.iterations,
						Argon2Parallelism:   1,
						MaxConcurrentHashes: 4,
						MaxWait:             2 * time.Second,
					},
				},
				RateLimit: tt.limits,
				Log:       config.LogConfig{Level: tt.level, Format: tt.format},
			}
			if !reflect.DeepEqual(cfg, want) {
				t.Errorf("Load() =\n%+v\nwant\n%+v", cfg, want)
			}
		})
	}
}

func TestProdRequiresDatabaseURLAndSigningKey(t *testing.T) {
	_, err := config.Load(config.Sources{Embedded: configs.FS(), Environ: []string{"NERVE_ENV=prod"}})
	if err == nil {
		t.Fatal("Load() error = nil, want database.url and auth.jwt.private_key_file to be required")
	}
	for _, key := range []string{"database.url: is required", "auth.jwt.private_key_file: is required in prod"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("Load() error = %v, want it to report %q", err, key)
		}
	}
}
```

- [ ] **Step 4: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。`cmd/nerve` 的测试用内嵌的 test 配置启动 `nerve serve`，证明新键都有默认值、校验通过；新的配置项此时还没有使用者（Task 4、5、9 接上）。

- [ ] **Step 5: 提交**

```bash
git add server/internal/platform/config server/configs
```
```bash
git commit -m "feat(M2/P2): configuration for rate limits, trusted proxies and the refresh deadline

YAML keys without a value and numbers their key cannot hold are errors
now (M2/P1 review); list keys take a comma-separated environment value.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** `TestLoadErrors` 报出 YAML 空值、越界数和畸形 CIDR；`TestValidateCrossKeyRules` 报出续期期限的不等式和 IPv4 映射的可信代理；`TestBuiltInProfiles` 核对三个环境的桶；`make test` 通过。

---

### Task 2: `platform/ratelimit`：令牌桶、`AllowAll`、`Reserve`

**Files:**
- Create: `server/internal/platform/ratelimit/ratelimit.go`、`ratelimit_test.go`

**Interfaces:**
- Produces（spec 2.4）：`ratelimit.New(now func() time.Time) *Limiter`；`Rate{PerMinute, Burst int}`；`(*Limiter).Bucket(name, Rate) *Bucket`；`(*Bucket).Name()`、`Allow(key) (retry, ok)`、`Reserve(key) (refund func(), retry, ok)`；`Check{Bucket, Key}`；`(*Limiter).AllowAll(checks ...Check) (denied *Bucket, retry time.Duration)`。
- 只导入标准库（M2 设计 3.10）；不导入别的平台包（规则 7，archtest 已有）。

**Tests:**（`ratelimit_test.go`，用可推进的时钟）
- `TestBucketAllowsItsBurstThenItsRate`、`TestBucketNeverHoldsMoreThanItsBurst`、`TestKeysAndBucketsAreSeparate`；
- `TestAllowAllTakesFromEveryBucketOrNone`（一个桶空了，其余的都不扣）、`TestAllowAllReportsTheLongestWait`、`TestAllowAllRejectsABucketOfAnotherLimiter`；
- `TestReserveAndRefund`（退回只算一次；桶已回满时晚到的退回不超过突发）、`TestReserveUnderConcurrency`（突发 3、50 个并发的预留只成功 3 个）；
- `TestIdleKeysAreDropped`（回满的键在一分钟后被清掉，清掉之前没回满的键保留）。

- [ ] **Step 1: 限流器**

`server/internal/platform/ratelimit/ratelimit.go`（新文件）：

```go
// Package ratelimit is nerve's in-process rate limiter (M2 design 3.10):
// token buckets per key, each with a rate and a burst. A request takes one
// unit from a bucket; AllowAll takes one from several buckets or from none,
// and Reserve takes one that the caller may give back afterwards. A key
// whose bucket has filled up again is dropped from time to time, so the
// memory follows the callers of the last minutes. It uses only the
// standard library.
package ratelimit

import (
	"sync"
	"time"
)

// sweepEvery is how often AllowAll drops the keys whose bucket is full.
const sweepEvery = time.Minute

// Rate is a bucket's refill rate and size. Both are at least 1 (the
// configuration validates them).
type Rate struct {
	PerMinute int // units the bucket gains a minute
	Burst     int // units the bucket holds at most
}

// Limiter holds the state of all its buckets under one lock, so AllowAll
// can take from several at once.
type Limiter struct {
	mu        sync.Mutex
	now       func() time.Time
	levels    map[key]*level
	nextSweep time.Time
}

type key struct {
	bucket *Bucket
	id     string
}

// level is the units in one key's bucket, as of at.
type level struct {
	units float64
	at    time.Time
}

// New returns a limiter that tells the time with now.
func New(now func() time.Time) *Limiter {
	return &Limiter{now: now, levels: map[key]*level{}, nextSweep: now().Add(sweepEvery)}
}

// Bucket is one bucket per key, e.g. login_ip per client IP.
type Bucket struct {
	limiter  *Limiter
	name     string
	interval time.Duration // a unit comes back every interval
	burst    float64
}

// Bucket returns a bucket of l with rate r, named for logs, e.g. after its
// configuration key.
func (l *Limiter) Bucket(name string, r Rate) *Bucket {
	return &Bucket{
		limiter:  l,
		name:     name,
		interval: max(time.Minute/time.Duration(r.PerMinute), time.Nanosecond),
		burst:    float64(r.Burst),
	}
}

// Name is the bucket's name.
func (b *Bucket) Name() string { return b.name }

// Allow takes one unit from key's bucket. Without a unit it takes nothing
// and returns how long until one is back.
func (b *Bucket) Allow(key string) (retry time.Duration, ok bool) {
	if denied, retry := b.limiter.AllowAll(Check{Bucket: b, Key: key}); denied != nil {
		return retry, false
	}
	return 0, true
}

// Reserve takes one unit from key's bucket now; refund gives it back, at
// most once and never beyond the burst. Without a unit it takes nothing and
// returns how long until one is back.
func (b *Bucket) Reserve(key string) (refund func(), retry time.Duration, ok bool) {
	if retry, ok := b.Allow(key); !ok {
		return nil, retry, false
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			l := b.limiter
			l.mu.Lock()
			defer l.mu.Unlock()
			lv := l.level(Check{Bucket: b, Key: key}, l.now())
			lv.units = min(lv.units+1, b.burst)
		})
	}, 0, true
}

// Check is one key of one bucket.
type Check struct {
	Bucket *Bucket
	Key    string
}

// AllowAll takes one unit from the bucket of every check, or from none: when
// a bucket is empty it takes nothing and returns the empty bucket with the
// longest wait, and that wait. denied is nil when every bucket gave a unit.
// Every bucket must belong to l.
func (l *Limiter) AllowAll(checks ...Check) (denied *Bucket, retry time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.sweep(now)
	levels := make([]*level, len(checks))
	for i, c := range checks {
		if c.Bucket.limiter != l {
			panic("ratelimit: AllowAll with a bucket of another limiter")
		}
		levels[i] = l.level(c, now)
		if units := levels[i].units; units < 1 {
			if wait := time.Duration((1 - units) * float64(c.Bucket.interval)); denied == nil || wait > retry {
				denied, retry = c.Bucket, wait
			}
		}
	}
	if denied != nil {
		return denied, retry
	}
	for _, lv := range levels {
		lv.units--
	}
	return nil, 0
}

// level returns c's level brought up to now; a new key starts full. The
// caller holds l.mu.
func (l *Limiter) level(c Check, now time.Time) *level {
	k := key{bucket: c.Bucket, id: c.Key}
	lv, ok := l.levels[k]
	if !ok {
		lv = &level{units: c.Bucket.burst, at: now}
		l.levels[k] = lv
	}
	c.Bucket.refill(lv, now)
	return lv
}

// refill adds the units that came back since lv.at, up to the burst.
func (b *Bucket) refill(lv *level, now time.Time) {
	if now.After(lv.at) {
		lv.units = min(b.burst, lv.units+float64(now.Sub(lv.at))/float64(b.interval))
		lv.at = now
	}
}

// sweep drops, at most once every sweepEvery, the keys whose bucket is full
// again: a full bucket is what a key without a level starts with. The caller
// holds l.mu.
func (l *Limiter) sweep(now time.Time) {
	if now.Before(l.nextSweep) {
		return
	}
	l.nextSweep = now.Add(sweepEvery)
	for k, lv := range l.levels {
		k.bucket.refill(lv, now)
		if lv.units >= k.bucket.burst {
			delete(l.levels, k)
		}
	}
}
```

- [ ] **Step 2: 测试**

`server/internal/platform/ratelimit/ratelimit_test.go`（新文件）：

```go
package ratelimit

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeClock is a clock the test moves by hand.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newClock() *fakeClock { return &fakeClock{t: time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)} }

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

// units reads what key holds in b now, without taking any.
func units(b *Bucket, key string) float64 {
	l := b.limiter
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.level(Check{Bucket: b, Key: key}, l.now()).units
}

// A bucket lets its burst through at once, then one unit per interval.
func TestBucketAllowsItsBurstThenItsRate(t *testing.T) {
	clock := newClock()
	b := New(clock.now).Bucket("login_ip", Rate{PerMinute: 60, Burst: 3})

	for i := range 3 {
		if _, ok := b.Allow("192.0.2.1"); !ok {
			t.Fatalf("request %d of the burst was refused", i+1)
		}
	}
	if retry, ok := b.Allow("192.0.2.1"); ok || retry != time.Second {
		t.Errorf("after the burst: Allow() = %v, %v; want refused, retry in 1s", retry, ok)
	}
	clock.advance(400 * time.Millisecond)
	if retry, ok := b.Allow("192.0.2.1"); ok || retry != 600*time.Millisecond {
		t.Errorf("0.4 s later: Allow() = %v, %v; want refused, retry in 600ms", retry, ok)
	}
	clock.advance(600 * time.Millisecond)
	if _, ok := b.Allow("192.0.2.1"); !ok {
		t.Error("a second later: Allow() refused, want the unit that came back")
	}
	if _, ok := b.Allow("192.0.2.1"); ok {
		t.Error("the unit that came back was taken twice")
	}
}

func TestBucketNeverHoldsMoreThanItsBurst(t *testing.T) {
	clock := newClock()
	b := New(clock.now).Bucket("anonymous", Rate{PerMinute: 600, Burst: 2})
	b.Allow("k")

	clock.advance(time.Hour)

	if got := units(b, "k"); got != 2 {
		t.Errorf("after an hour the bucket holds %v, want its burst 2", got)
	}
}

func TestKeysAndBucketsAreSeparate(t *testing.T) {
	l := New(newClock().now)
	a := l.Bucket("a", Rate{PerMinute: 1, Burst: 1})
	b := l.Bucket("b", Rate{PerMinute: 1, Burst: 1})
	a.Allow("k")

	if _, ok := a.Allow("other"); !ok {
		t.Error("another key of the same bucket was refused")
	}
	if _, ok := b.Allow("k"); !ok {
		t.Error("the same key of another bucket was refused")
	}
	if _, ok := a.Allow("k"); ok {
		t.Error("the emptied key was allowed")
	}
}

// When one bucket refuses, AllowAll takes nothing from the others (M2
// design 3.10): a refused sign-in to one address must not spend the IP's.
func TestAllowAllTakesFromEveryBucketOrNone(t *testing.T) {
	l := New(newClock().now)
	ip := l.Bucket("login_ip", Rate{PerMinute: 30, Burst: 5})
	ipEmail := l.Bucket("login_ip_email", Rate{PerMinute: 10, Burst: 2})
	checks := func(email string) []Check {
		return []Check{{Bucket: ip, Key: "192.0.2.1"}, {Bucket: ipEmail, Key: "192.0.2.1 " + email}}
	}

	for range 2 {
		if denied, _ := l.AllowAll(checks("a@x")...); denied != nil {
			t.Fatalf("AllowAll() refused by %s within both bursts", denied.Name())
		}
	}
	for range 10 {
		if denied, retry := l.AllowAll(checks("a@x")...); denied != ipEmail || retry != 6*time.Second {
			t.Fatalf("AllowAll() = %v, %v; want refused by login_ip_email, retry in 6s", denied, retry)
		}
	}

	if got := units(ip, "192.0.2.1"); got != 3 {
		t.Errorf("login_ip holds %v after 2 allowed and 10 refused attempts, want 3", got)
	}
	if denied, _ := l.AllowAll(checks("b@x")...); denied != nil {
		t.Errorf("another address from the same IP was refused by %s", denied.Name())
	}
}

// Of several empty buckets, AllowAll names the one with the longest wait.
func TestAllowAllReportsTheLongestWait(t *testing.T) {
	l := New(newClock().now)
	fast := l.Bucket("fast", Rate{PerMinute: 60, Burst: 1})
	slow := l.Bucket("slow", Rate{PerMinute: 1, Burst: 1})
	both := []Check{{Bucket: fast, Key: "k"}, {Bucket: slow, Key: "k"}}
	l.AllowAll(both...)

	if denied, retry := l.AllowAll(both...); denied != slow || retry != time.Minute {
		t.Errorf("AllowAll() = %v, %v; want slow, 1m", denied, retry)
	}
}

func TestAllowAllRejectsABucketOfAnotherLimiter(t *testing.T) {
	l := New(newClock().now)
	other := New(newClock().now).Bucket("other", Rate{PerMinute: 1, Burst: 1})
	defer func() {
		if recover() == nil {
			t.Error("AllowAll() with another limiter's bucket did not panic")
		}
	}()
	l.AllowAll(Check{Bucket: other, Key: "k"})
}

// Reserve takes a unit at once; its refund gives it back once, and never
// beyond the burst (M2 design 3.6: the failure gate refunds unless the
// credential failed).
func TestReserveAndRefund(t *testing.T) {
	clock := newClock()
	b := New(clock.now).Bucket("auth_failure", Rate{PerMinute: 60, Burst: 2})

	refund, _, ok := b.Reserve("192.0.2.1")
	if !ok || units(b, "192.0.2.1") != 1 {
		t.Fatalf("Reserve() ok = %v, bucket holds %v; want a unit taken", ok, units(b, "192.0.2.1"))
	}
	refund()
	refund()
	if got := units(b, "192.0.2.1"); got != 2 {
		t.Errorf("after refunding twice the bucket holds %v, want 2: a refund counts once", got)
	}

	refundA, _, _ := b.Reserve("192.0.2.1")
	b.Reserve("192.0.2.1")
	if _, retry, ok := b.Reserve("192.0.2.1"); ok || retry != time.Second {
		t.Errorf("Reserve() of an empty bucket = %v, %v; want refused, retry in 1s", retry, ok)
	}
	clock.advance(2 * time.Second) // the bucket is full again
	refundA()
	if got := units(b, "192.0.2.1"); got != 2 {
		t.Errorf("a late refund left %v units, want at most the burst 2", got)
	}
}

// Reserving is one step under the lock: with a burst of 3, 50 concurrent
// reservations get exactly 3 units.
func TestReserveUnderConcurrency(t *testing.T) {
	b := New(newClock().now).Bucket("auth_failure", Rate{PerMinute: 60, Burst: 3})
	var granted atomic.Int32
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			if _, _, ok := b.Reserve("192.0.2.1"); ok {
				granted.Add(1)
			}
		})
	}
	wg.Wait()

	if got := granted.Load(); got != 3 {
		t.Errorf("%d of 50 concurrent reservations were granted, want the burst 3", got)
	}
}

// Keys whose bucket has filled up again are dropped once a minute; a key
// still short of its burst stays.
func TestIdleKeysAreDropped(t *testing.T) {
	clock := newClock()
	l := New(clock.now)
	b := l.Bucket("anonymous", Rate{PerMinute: 60, Burst: 10})
	for i := range 100 {
		b.Allow("192.0.2." + strconv.Itoa(i))
	}
	clock.advance(59 * time.Second)
	for range 5 {
		b.Allow("busy") // not yet a minute: no sweep
	}
	if len(l.levels) != 101 {
		t.Fatalf("%d keys before a minute has passed, want 101", len(l.levels))
	}

	clock.advance(time.Second)
	b.Allow("busy") // a minute has passed: the sweep runs first

	if len(l.levels) != 1 || units(b, "busy") != 5 {
		t.Errorf("after a minute %d keys are left and busy holds %v; want only busy, with 5", len(l.levels), units(b, "busy"))
	}
}
```

Run: `go -C server test -count=1 -race ./internal/platform/ratelimit/`
Expected: `ok  	github.com/open-nerve/NerveProject/server/internal/platform/ratelimit`

- [ ] **Step 3: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`（`internal/archtest` 的依赖规则覆盖新包）。

- [ ] **Step 4: 提交**

```bash
git add server/internal/platform/ratelimit
```
```bash
git commit -m "feat(M2/P2): platform/ratelimit, token buckets with AllowAll and Reserve

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 9 个测试在 `-race` 下通过；包只导入标准库。

---

### Task 3: 固定链上的安全响应头

**Files:**
- Modify: `server/internal/platform/httpserver/middleware.go`、`middleware_test.go`、`server.go`（注释）
- Create: `server/internal/bootstrap/headers_test.go`

**Interfaces:**
- Produces（spec 2.5，M2 设计 8.3）：固定链 `请求 ID → 异常恢复 → 访问日志 → 安全响应头`；`securityHeaders`（`X-Content-Type-Options: nosniff`、`Referrer-Policy: same-origin`、`X-Frame-Options: DENY`）；异常恢复清掉响应头之后重新设上它们。
- CSP 不在这里（P4 的 `webui`）。

**Tests:**
- `middleware_test.go`：新增 `TestSecurityHeadersOnEveryResponse`（200、404、problem）；`TestPanicDiscardsHeadersSetBeforeIt` 另核对 panic 的 500 带三个响应头。
- `bootstrap/headers_test.go`：`TestSecurityHeadersOnEveryResponse`，接好线的程序上的页面（`/`、`/settings/profile/general`）、`/healthz`、`/api/v0/instance`、`/api/v0/nope`（404）、`/api/v0/me`（401）。

- [ ] **Step 1: 中间件**

`server/internal/platform/httpserver/middleware.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/platform/httpserver/middleware.go
+++ b/server/internal/platform/httpserver/middleware.go
@@ -16,12 +16,31 @@
 
 type requestIDKey struct{}
 
+// securityHeaders go on every response (M2 design 8.3): no MIME sniffing, no
+// referrer beyond this site, and no framing by any page. The CSP goes on the
+// HTML pages only, and webui sets it.
+var securityHeaders = [...][2]string{
+	{"X-Content-Type-Options", "nosniff"},
+	{"Referrer-Policy", "same-origin"},
+	{"X-Frame-Options", "DENY"},
+}
+
 // middleware wraps h in the platform chain. The order is fixed, outermost
-// first: request ID -> recover -> access log.
+// first: request ID -> recover -> access log -> security headers.
 func middleware(h http.Handler, logger *slog.Logger) http.Handler {
-	return withRequestID(withRecover(logger, withAccessLog(logger, h)))
+	return withRequestID(withRecover(logger, withAccessLog(logger, withSecurityHeaders(h))))
 }
 
+// withSecurityHeaders sets securityHeaders before the handler writes.
+func withSecurityHeaders(next http.Handler) http.Handler {
+	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
+		for _, h := range securityHeaders {
+			w.Header().Set(h[0], h[1])
+		}
+		next.ServeHTTP(w, r)
+	})
+}
+
 // RequestID returns the ID the request ID middleware assigned to the request,
 // for log lines outside this package; "" outside a request.
 func RequestID(ctx context.Context) string {
@@ -61,9 +80,10 @@
 
 // withRecover turns a panic into a logged 500 problem. Headers the handler set
 // are dropped, except the request ID: a Set-Cookie must not leak, and a stale
-// Content-Length or Content-Encoding would corrupt the problem body. If the
-// response has already started, the connection is aborted instead so the
-// client cannot mistake a truncated body for a complete one.
+// Content-Length or Content-Encoding would corrupt the problem body. The
+// security headers, which the chain set before the handler ran, are set again.
+// If the response has already started, the connection is aborted instead so
+// the client cannot mistake a truncated body for a complete one.
 func withRecover(logger *slog.Logger, next http.Handler) http.Handler {
 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
 		rec := &statusRecorder{ResponseWriter: w}
@@ -91,6 +111,9 @@
 					delete(header, name)
 				}
 			}
+			for _, h := range securityHeaders {
+				header.Set(h[0], h[1])
+			}
 			WriteProblem(rec, Problem{
 				Status: http.StatusInternalServerError,
 				Code:   CodeInternal,
```

`server/internal/platform/httpserver/server.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/platform/httpserver/server.go
+++ b/server/internal/platform/httpserver/server.go
@@ -25,10 +25,10 @@
 	addrFile        string
 }
 
-// NewServer serves h behind the platform middleware chain
-// (request ID -> recover -> access log). Reading and writing on a connection
-// are bounded: request headers (server.read_header_timeout), the whole
-// request with its body (server.read_timeout), the response
+// NewServer serves h behind the platform middleware chain (request ID ->
+// recover -> access log -> security headers). Reading and writing on a
+// connection are bounded: request headers (server.read_header_timeout), the
+// whole request with its body (server.read_timeout), the response
 // (server.write_timeout) and idle keep-alive (idleTimeout). A handler that
 // legitimately needs longer, such as a file upload, extends its own deadlines
 // with http.ResponseController instead of raising them for every request.
```

- [ ] **Step 2: 测试**

`server/internal/platform/httpserver/middleware_test.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/platform/httpserver/middleware_test.go
+++ b/server/internal/platform/httpserver/middleware_test.go
@@ -88,6 +88,28 @@
 	}
 }
 
+// Every response carries the security headers (M2 design 8.3), whatever the
+// handler answers.
+func TestSecurityHeadersOnEveryResponse(t *testing.T) {
+	handlers := map[string]http.Handler{
+		"200": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) }),
+		"404": http.NotFoundHandler(),
+		"problem": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
+			WriteProblem(w, Problem{Status: 400, Code: CodeBadRequest})
+		}),
+	}
+	want := map[string]string{"X-Content-Type-Options": "nosniff", "Referrer-Policy": "same-origin", "X-Frame-Options": "DENY"}
+	for name, h := range handlers {
+		rec := serve(middleware(h, slog.New(slog.DiscardHandler)), httptest.NewRequest(http.MethodGet, "/", nil))
+
+		for header, value := range want {
+			if got := rec.Header().Get(header); got != value {
+				t.Errorf("%s response: %s = %q, want %q", name, header, got, value)
+			}
+		}
+	}
+}
+
 func TestPanicBecomes500Problem(t *testing.T) {
 	logger, logs := captureLogs(t)
 	h := middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
@@ -148,6 +170,11 @@
 	if id := resp.Header.Get(HeaderRequestID); id != "req-3" {
 		t.Errorf("X-Request-Id = %q, want req-3", id)
 	}
+	for _, h := range securityHeaders {
+		if got := resp.Header.Get(h[0]); got != h[1] {
+			t.Errorf("%s = %q, want %q: the 500 of a panic keeps the security headers", h[0], got, h[1])
+		}
+	}
 	var p Problem
 	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil || p.Code != CodeInternal {
 		t.Errorf("body = %+v, %v; want the complete internal_error problem", p, err)
```

`server/internal/bootstrap/headers_test.go`（新文件）：

```go
package bootstrap

import (
	"testing"
	"testing/fstest"
)

// Every response of the wired app carries the security headers of the fixed
// chain (M2 design 8.3): pages, probes, API answers and API problems.
func TestSecurityHeadersOnEveryResponse(t *testing.T) {
	base := startApp(t, testConfig(t, unreachableDB, false), fstest.MapFS{})
	want := map[string]string{"X-Content-Type-Options": "nosniff", "Referrer-Policy": "same-origin", "X-Frame-Options": "DENY"}
	for _, path := range []string{"/", "/settings/profile/general", "/healthz", "/api/v0/instance", "/api/v0/nope", "/api/v0/me"} {
		res, err := client.Get(base + path)
		if err != nil {
			t.Fatal(err)
		}
		_ = res.Body.Close()

		for header, value := range want {
			if got := res.Header.Get(header); got != value {
				t.Errorf("GET %s (%d): %s = %q, want %q", path, res.StatusCode, header, got, value)
			}
		}
	}
}
```

- [ ] **Step 3: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/platform/httpserver/middleware.go server/internal/platform/httpserver/middleware_test.go server/internal/platform/httpserver/server.go server/internal/bootstrap/headers_test.go
```
```bash
git commit -m "feat(M2/P2): security headers on every response, the fourth middleware of the fixed chain

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 两个 `TestSecurityHeadersOnEveryResponse` 和 `TestPanicDiscardsHeadersSetBeforeIt` 通过。

---

### Task 4: 客户端 IP、可信代理与 IP 键

**Files:**
- Create: `server/internal/platform/httpserver/clientip.go`
- Create: `server/internal/platform/httpserver/clientip_test.go`（过渡：Task 5 改用新的测试辅助函数）
- Modify: `server/internal/platform/httpserver/api.go`、`api_test.go`（过渡：Task 5 加上失败闸门和限流）
- Modify: `server/internal/bootstrap/app.go`、`app_test.go`（过渡：Task 5、9 加上限流器）

**Interfaces:**
- Consumes: `config.ServerConfig.TrustedProxies`、`config.RateLimitConfig.IPv6PrefixLen`（Task 1）。
- Produces（spec 2.6）：
  - `clientIPs{logger, trusted, v6Prefix, warned}`，方法 `of(r) netip.Addr`、`key(ip) string`；
  - `httpserver.APIConfig` 加上 `TrustedProxies []netip.Prefix`、`IPv6PrefixLen int`；
  - `httpserver.RequestMeta` 加上 `IPKey string`；请求元信息中间件填 `ClientIP` 和 `IPKey`。

**Tests:**（`clientip_test.go`）
- `TestClientIP`（13 个）、`TestClientIPOfAnUnparsablePeerIsZero`、`TestUntrustedForwardingIsWarnedOnce`、`TestIPKey`（6 个）、`TestRequestMetaCarriesTheClientAndItsKey`。
- `api_test.go` 的已有测试随 `APIConfig` 的新字段调整。

- [ ] **Step 1: 客户端 IP**

`server/internal/platform/httpserver/clientip.go`（新文件）：

```go
package httpserver

import (
	"log/slog"
	"net/http"
	"net/netip"
	"slices"
	"strings"
	"sync"
)

// clientIPs tells who the client of a request is (M2 design 3.10), and the
// key it counts under in the per-IP rate-limit buckets.
type clientIPs struct {
	logger   *slog.Logger
	trusted  []netip.Prefix // server.trusted_proxies
	v6Prefix int            // ratelimit.ipv6_prefix_len
	// warned makes the warning about X-Forwarded-For from an untrusted peer
	// once per process: bootstrap builds one API.
	warned sync.Once
}

// of returns the client of r: the connection's peer, unless the peer is a
// trusted proxy. Then it is the first address of X-Forwarded-For, from the
// right, that is not a trusted proxy: the proxies append the address they
// received from, so what lies left of the first untrusted one is the
// client's to write. When every address is a trusted proxy the leftmost is
// the client; a malformed entry ends the walk at the trusted hop that
// forwarded it. Addresses lose their zone, and an IPv4-mapped IPv6 address is
// its IPv4 address. The zero Addr when the peer address does not parse.
func (c *clientIPs) of(r *http.Request) netip.Addr {
	client := normalize(peerAddr(r.RemoteAddr))
	forwarded := r.Header.Values("X-Forwarded-For")
	if !c.isTrusted(client) {
		if len(forwarded) > 0 {
			c.warned.Do(func() {
				c.logger.WarnContext(r.Context(), "ignored X-Forwarded-For from a peer that is not a trusted proxy: "+
					"behind a reverse proxy, add its address to server.trusted_proxies, or every client counts as the proxy",
					slog.String("peer", client.String()))
			})
		}
		return client
	}
	hops := strings.Split(strings.Join(forwarded, ","), ",")
	for _, hop := range slices.Backward(hops) {
		addr, err := netip.ParseAddr(strings.TrimSpace(hop))
		if err != nil {
			break
		}
		client = normalize(addr)
		if !c.isTrusted(client) {
			break
		}
	}
	return client
}

func (c *clientIPs) isTrusted(ip netip.Addr) bool {
	return ip.IsValid() && slices.ContainsFunc(c.trusted, func(p netip.Prefix) bool { return p.Contains(ip) })
}

// key is the key of ip in the per-IP buckets: an IPv4 address is itself, an
// IPv6 address its prefix of ratelimit.ipv6_prefix_len bits, since a host
// usually holds a whole /64. "" for the zero Addr.
func (c *clientIPs) key(ip netip.Addr) string {
	switch {
	case !ip.IsValid():
		return ""
	case ip.Is4():
		return ip.String()
	}
	p, _ := ip.Prefix(c.v6Prefix) // the configuration holds it to 1-128
	return p.String()
}

// peerAddr is the address of RemoteAddr without its port; the zero Addr
// when it does not parse.
func peerAddr(remote string) netip.Addr {
	ap, err := netip.ParseAddrPort(remote)
	if err != nil {
		return netip.Addr{}
	}
	return ap.Addr()
}

func normalize(ip netip.Addr) netip.Addr {
	return ip.Unmap().WithZone("")
}
```

`server/internal/platform/httpserver/api.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/platform/httpserver/api.go
+++ b/server/internal/platform/httpserver/api.go
@@ -30,8 +30,10 @@
 	// PublicOperations are the route patterns that need no token, e.g.
 	// "POST /api/v0/auth/register": the union of every module's list.
 	PublicOperations []string
-	MaxBodyBytes     int64         // server.max_body_bytes
-	RequestTimeout   time.Duration // server.request_timeout
+	MaxBodyBytes     int64          // server.max_body_bytes
+	RequestTimeout   time.Duration  // server.request_timeout
+	TrustedProxies   []netip.Prefix // server.trusted_proxies
+	IPv6PrefixLen    int            // ratelimit.ipv6_prefix_len
 }
 
 // API is what the platform hands to every module's HTTP adapter: the error
@@ -43,6 +45,7 @@
 	public         map[string]bool
 	maxBodyBytes   int64
 	requestTimeout time.Duration
+	clients        *clientIPs
 }
 
 // NewAPI returns the API value for cfg.
@@ -58,6 +61,7 @@
 		public:         public,
 		maxBodyBytes:   cfg.MaxBodyBytes,
 		requestTimeout: cfg.RequestTimeout,
+		clients:        &clientIPs{logger: cfg.Logger, trusted: cfg.TrustedProxies, v6Prefix: cfg.IPv6PrefixLen},
 	}
 }
 
@@ -83,10 +87,14 @@
 
 // RequestMeta describes the client of a request.
 type RequestMeta struct {
-	// ClientIP is the connection's peer address, without port and zone; an
-	// IPv4-mapped IPv6 address is its IPv4 address. (M2/P2 adds trusted
-	// proxies.) The zero Addr when the peer address cannot be parsed.
-	ClientIP  netip.Addr
+	// ClientIP is the client's full address: the connection's peer, or what
+	// a trusted proxy forwarded (M2 design 3.10); without zone, and an
+	// IPv4-mapped IPv6 address is its IPv4 address. Logs and sessions record
+	// it. The zero Addr when the peer address cannot be parsed.
+	ClientIP netip.Addr
+	// IPKey is what the per-IP rate-limit buckets count the client by: an
+	// IPv4 address, or the prefix of an IPv6 one (ratelimit.ipv6_prefix_len).
+	IPKey     string
 	UserAgent string
 }
 
@@ -101,11 +109,8 @@
 
 func (a *API) requestMeta(next http.Handler) http.Handler {
 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
-		var ip netip.Addr
-		if ap, err := netip.ParseAddrPort(r.RemoteAddr); err == nil {
-			ip = ap.Addr().Unmap().WithZone("")
-		}
-		meta := RequestMeta{ClientIP: ip, UserAgent: r.UserAgent()}
+		ip := a.clients.of(r)
+		meta := RequestMeta{ClientIP: ip, IPKey: a.clients.key(ip), UserAgent: r.UserAgent()}
 		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), metaKey{}, meta)))
 	})
 }
```

- [ ] **Step 2: 测试**

`server/internal/platform/httpserver/clientip_test.go`（新文件）：

```go
package httpserver

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

func clientsTrusting(logger *slog.Logger, cidrs ...string) *clientIPs {
	c := &clientIPs{logger: logger, v6Prefix: 64}
	for _, s := range cidrs {
		c.trusted = append(c.trusted, netip.MustParsePrefix(s))
	}
	return c
}

func requestFrom(remote string, forwarded ...string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/api/v0/instance", nil)
	r.RemoteAddr = remote
	for _, f := range forwarded {
		r.Header.Add("X-Forwarded-For", f)
	}
	return r
}

// The client is the peer, unless the peer is a trusted proxy: then it is the
// first address of X-Forwarded-For, from the right, that is not a trusted
// proxy (M2 design 3.10).
func TestClientIP(t *testing.T) {
	tests := []struct {
		name      string
		remote    string
		forwarded []string
		want      string
	}{
		{"untrusted peer: its forwarding is ignored", "203.0.113.7:5555", []string{"198.51.100.1"}, "203.0.113.7"},
		{"trusted peer without forwarding", "10.0.0.1:5555", nil, "10.0.0.1"},
		{"trusted peer forwards the client", "10.0.0.1:5555", []string{"198.51.100.1"}, "198.51.100.1"},
		{"trusted hops are skipped", "10.0.0.1:5555", []string{"198.51.100.1, 10.0.0.2"}, "198.51.100.1"},
		{"what the client wrote on the left is not believed", "10.0.0.1:5555", []string{"6.6.6.6,198.51.100.1 , 10.0.0.2"}, "198.51.100.1"},
		{"several header lines are one list", "10.0.0.1:5555", []string{"6.6.6.6", "198.51.100.1, 10.0.0.2"}, "198.51.100.1"},
		{"every hop trusted: the leftmost", "10.0.0.1:5555", []string{"10.0.0.3, 10.0.0.2"}, "10.0.0.3"},
		{"a malformed entry ends at the hop that forwarded it", "10.0.0.1:5555", []string{"198.51.100.1, bogus, 10.0.0.2"}, "10.0.0.2"},
		{"an empty entry ends the walk too", "10.0.0.1:5555", []string{"198.51.100.1,"}, "10.0.0.1"},
		{"IPv6 proxy and client", "[fd00::1]:443", []string{"2001:db8::5"}, "2001:db8::5"},
		{"a mapped client is IPv4", "10.0.0.1:5555", []string{"::ffff:198.51.100.1"}, "198.51.100.1"},
		{"a mapped peer is IPv4, and trusted", "[::ffff:10.0.0.1]:80", []string{"198.51.100.1"}, "198.51.100.1"},
		{"the zone is dropped", "[fe80::1%en0]:80", nil, "fe80::1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := clientsTrusting(slog.New(slog.DiscardHandler), "10.0.0.0/8", "fd00::/8")

			if got := c.of(requestFrom(tt.remote, tt.forwarded...)); got != netip.MustParseAddr(tt.want) {
				t.Errorf("client = %v, want %s", got, tt.want)
			}
		})
	}
}

func TestClientIPOfAnUnparsablePeerIsZero(t *testing.T) {
	c := clientsTrusting(slog.New(slog.DiscardHandler))
	if got := c.of(requestFrom("@unix-socket")); got.IsValid() || c.key(got) != "" {
		t.Errorf("client = %v with key %q, want the zero Addr and the empty key", got, c.key(got))
	}
}

// X-Forwarded-For from a peer that is not trusted most likely means a proxy
// missing from server.trusted_proxies: warned once per process, never again.
func TestUntrustedForwardingIsWarnedOnce(t *testing.T) {
	logger, logs := captureLogs(t)
	c := clientsTrusting(logger, "10.0.0.0/8")

	c.of(requestFrom("203.0.113.7:5555", "198.51.100.1"))
	c.of(requestFrom("203.0.113.8:5555", "198.51.100.2"))
	c.of(requestFrom("10.0.0.1:5555", "198.51.100.3"))

	var warnings []map[string]any
	for _, e := range logs() {
		if e["level"] == "WARN" {
			warnings = append(warnings, e)
		}
	}
	if len(warnings) != 1 || warnings[0]["peer"] != "203.0.113.7" {
		t.Errorf("warnings = %v, want one, naming the peer 203.0.113.7", warnings)
	}
}

// The per-IP buckets count an IPv4 client by its address and an IPv6 client
// by its prefix: a host usually holds a whole /64 (M2 design 3.10).
func TestIPKey(t *testing.T) {
	tests := []struct {
		ip     string
		prefix int
		want   string
	}{
		{"198.51.100.1", 64, "198.51.100.1"},
		{"2001:db8:1:2::1", 64, "2001:db8:1:2::/64"},
		{"2001:db8:1:2:ffff:ffff:ffff:ffff", 64, "2001:db8:1:2::/64"},
		{"2001:db8:1:3::1", 64, "2001:db8:1:3::/64"},
		{"2001:db8:1:2::1", 48, "2001:db8:1::/48"},
		{"2001:db8:1:2::1", 128, "2001:db8:1:2::1/128"},
	}
	for _, tt := range tests {
		c := &clientIPs{v6Prefix: tt.prefix}
		if got := c.key(netip.MustParseAddr(tt.ip)); got != tt.want {
			t.Errorf("key(%s) with /%d = %q, want %q", tt.ip, tt.prefix, got, tt.want)
		}
	}
}

// The request meta middleware puts both in the context: the full address
// for logs and sessions, the key for the buckets.
func TestRequestMetaCarriesTheClientAndItsKey(t *testing.T) {
	auth := &fakeAuth{}
	api := newTestAPI(auth, slog.New(slog.DiscardHandler))
	router, _ := mount(t, api, slog.New(slog.DiscardHandler))
	req := post("/api/v0/things", "tok", `{"name":"a"}`)
	req.RemoteAddr = "[2001:db8:1:2::7]:443"

	serve(router, req)

	meta := RequestMetaFrom(auth.ctx)
	if meta.ClientIP != netip.MustParseAddr("2001:db8:1:2::7") || meta.IPKey != "2001:db8:1:2::/64" {
		t.Errorf("meta = %+v, want the full address and its /64", meta)
	}
}
```

`server/internal/platform/httpserver/api_test.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/platform/httpserver/api_test.go
+++ b/server/internal/platform/httpserver/api_test.go
@@ -90,6 +90,7 @@
 		PublicOperations: []string{publicRoute},
 		MaxBodyBytes:     64,
 		RequestTimeout:   2 * time.Second,
+		IPv6PrefixLen:    64,
 	})
 }
 
```

- [ ] **Step 3: 接线**

`server/internal/bootstrap/app.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/bootstrap/app.go
+++ b/server/internal/bootstrap/app.go
@@ -102,6 +102,8 @@
 		PublicOperations: a.publicOperations,
 		MaxBodyBytes:     cfg.Server.MaxBodyBytes,
 		RequestTimeout:   cfg.Server.RequestTimeout,
+		TrustedProxies:   cfg.Server.TrustedProxies,
+		IPv6PrefixLen:    cfg.RateLimit.IPv6PrefixLen,
 	})
 	// Modules mount their generated routes on this root router, next to the
 	// platform's /api/ fallback; an /api/v0/ sub-mux would shadow it.
```

`server/internal/bootstrap/app_test.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/bootstrap/app_test.go
+++ b/server/internal/bootstrap/app_test.go
@@ -64,7 +64,8 @@
 				MaxWait:             2 * time.Second,
 			},
 		},
-		Log: config.LogConfig{Level: "error", Format: "text"},
+		RateLimit: config.RateLimitConfig{IPv6PrefixLen: 64},
+		Log:       config.LogConfig{Level: "error", Format: "text"},
 	}
 }
 
```

- [ ] **Step 4: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。

- [ ] **Step 5: 提交**

```bash
git add server/internal/platform/httpserver/clientip.go server/internal/platform/httpserver/clientip_test.go server/internal/platform/httpserver/api.go server/internal/platform/httpserver/api_test.go server/internal/bootstrap/app.go server/internal/bootstrap/app_test.go
```
```bash
git commit -m "feat(M2/P2): the client IP behind trusted proxies, and its rate-limit key

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** `TestClientIP` 的 13 个情况和 `TestIPKey` 的 6 个情况通过；不可信的转发只警告一次。

---

### Task 5: 失败闸门、限流中间件与 `rate_limited`；契约的响应头

**Files:**
- Create: `server/internal/platform/httpserver/limit.go`
- Modify: `server/internal/platform/httpserver/api.go`、`api_test.go`、`clientip_test.go`、`problem.go`、`contract_test.go`
- Modify: `server/internal/platform/httpserver/apitest/problems.go`、`problems_test.go`
- Modify: `api/common.yaml`、`api/modules/instance.yaml`；`api/openapi.yaml`、`api/modules/identity.yaml`（过渡：Task 9 加上三个操作）
- Modify: `server/internal/modules/identity/adapter/authn/authenticator.go`、`authenticator_test.go`、`server/internal/modules/identity/app/authenticate_test.go`
- Modify: `server/internal/modules/instance/adapter/http/handler_test.go`；`server/internal/modules/identity/adapter/http/handler_test.go`（过渡）
- Modify: `server/internal/bootstrap/app.go`、`app_test.go`（过渡：Task 9 加上 identity 的桶）
- Generate: `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`、`server/internal/platform/httpserver/apigen/components.gen.go`、`server/internal/modules/{identity,instance}/adapter/http/gen/server.gen.go`

**Interfaces:**
- Consumes: `ratelimit`（Task 2）、`clientIPs` 和 `RequestMeta.IPKey`（Task 4）、`config.RateLimitConfig`（Task 1）。
- Produces（spec 2.7）：
  - `httpserver.Limiter{Allow; Reserve}`；`APIConfig` 加上 `Anonymous`、`Authenticated`、`AuthFailure`；
  - `NewAPI(cfg) (*API, error)`：缺依赖时 `httpserver: APIConfig lacks <名字>`；
  - 中间件顺序 `请求元信息 → 期限 → 请求体上限 → 失败闸门和认证 → 限流 → 请求体结构`；
  - 可选接口 `ExpiredCredential() bool`（`authn` 的 `expired` 实现它）；
  - `httpserver.CodeRateLimited`；429 `rate_limited`，`Retry-After` 向上取整；INFO "rate limited"（`request_id`、`bucket`、`ip`）；
  - 契约：顶层 `x-problem-codes` 加上 `rate_limited`；两个模块的 `Problem` 响应声明 `Retry-After`、`WWW-Authenticate`。
- P1 评审第 6 节的四项在这里完成：`NewAPI` 检查依赖、契约的响应头、`apitest` 的测试卫生、`TestAuthenticateTellsAnExpiredAccessToken` 的边界和注释（spec 第 7 节）。

**Tests:**
- `api_test.go`：`TestNewAPIRequiresItsDependencies`、`TestFailureGateKeepsTheUnitOfAFailedCredentialOnly`（6 个）、`TestFailureGateTurnsAwayWithoutAuthenticating`、`TestFailureGateHoldsUnderConcurrency`（额度 3、50 个并发：认证器调用 3 次、3 个 401、47 个 429）、`TestFailureGateCountsByTheIPKey`、`TestRateLimitPicksTheBucketAndKey`（5 个）、`TestRateLimitedRequestIs429`（2 个）；原有的测试改用 `newTestAPI`、`buildAPI`。
- `contract_test.go`：`TestPlatformProblemsMatchTheContract` 加上 429。
- `authenticator_test.go`：假用例核对收到的令牌；过期的访问令牌满足 `ExpiredCredential()`，其他 401 不满足。
- `authenticate_test.go`：`TestAuthenticateTellsAnExpiredAccessToken` 覆盖 `exp` 等于现在和早 1 秒。
- `problems_test.go`：`TestCheckResponseRecordsTheAnsweredCode` 用自己的操作和码，先断言记录为空。
- `instance` 的 `handler_test.go`：用真实的桶和一个不认识任何令牌的认证器搭 `API`；identity 的 `handler_test.go` 同样改用真实的桶。
- `bootstrap/app_test.go`：`testConfig` 的六个桶都是每分钟 600000、突发 100000（`roomy`），整程序测试不会被限流。

- [ ] **Step 1: 限流中间件和失败闸门**

`server/internal/platform/httpserver/limit.go`（新文件）：

```go
package httpserver

import (
	"log/slog"
	"net/http"
	"time"
)

// Limiter is one rate-limit bucket, with a unit count per key (M2 design
// 3.10). platform/ratelimit's buckets implement it; bootstrap sizes them
// from the configuration.
type Limiter interface {
	// Allow takes one unit of key's bucket, or tells how long until one is
	// back.
	Allow(key string) (retry time.Duration, ok bool)
	// Reserve takes one unit of key's bucket now; refund gives it back, at
	// most once.
	Reserve(key string) (refund func(), retry time.Duration, ok bool)
}

// rateLimit takes one unit of the caller's bucket (M2 design 3.10): of its
// credential in authenticated when authentication gave it one, else of its
// client IP in anonymous. Requests a module turns away later still count.
func (a *API) rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bucket, name, key := a.anonymous, "anonymous", RequestMetaFrom(r.Context()).IPKey
		if credential, ok := r.Context().Value(credentialKey{}).(string); ok {
			bucket, name, key = a.authenticated, "authenticated", credential
		}
		if retry, ok := bucket.Allow(key); !ok {
			a.tooManyRequests(w, r, name, retry)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// tooManyRequests answers 429 rate_limited with Retry-After and logs which
// bucket turned the client away (M2 design 8.4).
func (a *API) tooManyRequests(w http.ResponseWriter, r *http.Request, bucket string, retry time.Duration) {
	a.logger.LogAttrs(r.Context(), slog.LevelInfo, "rate limited",
		slog.String("request_id", RequestID(r.Context())), slog.String("bucket", bucket),
		slog.String("ip", RequestMetaFrom(r.Context()).ClientIP.String()))
	a.Errors.Write(w, r, rateLimited(retry))
}

// rateLimited is the ProblemError of a request over its rate: 429
// rate_limited, retry after the wait.
type rateLimited time.Duration

func (rateLimited) Error() string               { return "Too many requests; retry later." }
func (rateLimited) ProblemStatus() int          { return http.StatusTooManyRequests }
func (rateLimited) ProblemCode() string         { return CodeRateLimited }
func (r rateLimited) RetryAfter() time.Duration { return time.Duration(r) }
```

`server/internal/platform/httpserver/api.go`（完整内容）：

```go
package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"slices"
	"strings"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/bodyshape"
)

// Authenticator checks a bearer token (M2 design 3.6). The identity module
// implements it; the platform does not know what an account is.
type Authenticator interface {
	// Authenticate returns a context carrying the caller, and the caller's
	// rate-limit key (session:<id> or pat:<id>). An invalid token is an
	// error with ProblemStatus() 401; for a token that is valid but for its
	// expiry, the error also has ExpiredCredential() true. Any other error
	// is an internal fault.
	Authenticate(ctx context.Context, token string) (context.Context, string, error)
}

// expiredCredential is optional on the 401 error of an Authenticator: a
// validly signed access token whose exp has passed is the client's normal
// cue to refresh, and does not count as a failure.
type expiredCredential interface {
	ExpiredCredential() bool
}

// APIConfig is what the platform's per-route middlewares need.
type APIConfig struct {
	Logger        *slog.Logger
	Authenticator Authenticator
	// PublicOperations are the route patterns that need no token, e.g.
	// "POST /api/v0/auth/register": the union of every module's list.
	PublicOperations []string
	MaxBodyBytes     int64          // server.max_body_bytes
	RequestTimeout   time.Duration  // server.request_timeout
	TrustedProxies   []netip.Prefix // server.trusted_proxies
	IPv6PrefixLen    int            // ratelimit.ipv6_prefix_len
	// The rate-limit buckets of the platform (M2 design 3.10).
	Anonymous     Limiter // ratelimit.anonymous: public operations, by client IP
	Authenticated Limiter // ratelimit.authenticated: the rest, by credential
	AuthFailure   Limiter // ratelimit.auth_failure: the gate before authentication, by client IP
}

// API is what the platform hands to every module's HTTP adapter: the error
// mapping for the generated code, and the per-route middlewares.
type API struct {
	Errors         APIErrors
	logger         *slog.Logger
	authenticator  Authenticator
	public         map[string]bool
	maxBodyBytes   int64
	requestTimeout time.Duration
	clients        *clientIPs
	anonymous      Limiter
	authenticated  Limiter
	authFailure    Limiter
}

// NewAPI returns the API value for cfg; every dependency of cfg is required.
func NewAPI(cfg APIConfig) (*API, error) {
	var missing []string
	for _, dep := range []struct {
		name string
		nil  bool
	}{
		{"Logger", cfg.Logger == nil},
		{"Authenticator", cfg.Authenticator == nil},
		{"Anonymous", cfg.Anonymous == nil},
		{"Authenticated", cfg.Authenticated == nil},
		{"AuthFailure", cfg.AuthFailure == nil},
	} {
		if dep.nil {
			missing = append(missing, dep.name)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("httpserver: APIConfig lacks %s", strings.Join(missing, ", "))
	}
	public := make(map[string]bool, len(cfg.PublicOperations))
	for _, p := range cfg.PublicOperations {
		public[p] = true
	}
	return &API{
		Errors:         NewAPIErrors(cfg.Logger),
		logger:         cfg.Logger,
		authenticator:  cfg.Authenticator,
		public:         public,
		maxBodyBytes:   cfg.MaxBodyBytes,
		requestTimeout: cfg.RequestTimeout,
		clients:        &clientIPs{logger: cfg.Logger, trusted: cfg.TrustedProxies, v6Prefix: cfg.IPv6PrefixLen},
		anonymous:      cfg.Anonymous,
		authenticated:  cfg.Authenticated,
		authFailure:    cfg.AuthFailure,
	}, nil
}

// Middlewares returns the per-route middlewares for a module's generated
// StdHTTPServerOptions.Middlewares; bodies is the module's generated
// bodyshape table. They run in this order (M2 design 3.6):
//
//	request meta → request deadline → body limit → failure gate and
//	authentication → rate limit → body structure
//
// The generated code wraps the last middleware of its list outermost, so
// the list is in reverse.
func (a *API) Middlewares(bodies *bodyshape.Table) []func(http.Handler) http.Handler {
	inOrder := []func(http.Handler) http.Handler{
		a.requestMeta,
		a.deadline,
		a.bodyLimit,
		a.authenticate,
		a.rateLimit,
		bodyshape.Middleware(bodies, a.Errors.BodyError),
	}
	slices.Reverse(inOrder)
	return inOrder
}

// RequestMeta describes the client of a request.
type RequestMeta struct {
	// ClientIP is the client's full address: the connection's peer, or what
	// a trusted proxy forwarded (M2 design 3.10); without zone, and an
	// IPv4-mapped IPv6 address is its IPv4 address. Logs and sessions record
	// it. The zero Addr when the peer address cannot be parsed.
	ClientIP netip.Addr
	// IPKey is what the per-IP rate-limit buckets count the client by: an
	// IPv4 address, or the prefix of an IPv6 one (ratelimit.ipv6_prefix_len).
	IPKey     string
	UserAgent string
}

type metaKey struct{}

// RequestMetaFrom returns the request's meta; the zero value outside the
// per-route middlewares.
func RequestMetaFrom(ctx context.Context) RequestMeta {
	m, _ := ctx.Value(metaKey{}).(RequestMeta)
	return m
}

func (a *API) requestMeta(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := a.clients.of(r)
		meta := RequestMeta{ClientIP: ip, IPKey: a.clients.key(ip), UserAgent: r.UserAgent()}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), metaKey{}, meta)))
	})
}

// deadline bounds the handler: server.write_timeout only fails the writes
// and never cancels the request's context (M0-P2 handoff 5).
func (a *API) deadline(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), a.requestTimeout)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// bodyLimit makes reading more than server.max_body_bytes fail with
// *http.MaxBytesError, which APIErrors answers with 413.
func (a *API) bodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, a.maxBodyBytes)
		next.ServeHTTP(w, r)
	})
}

// credentialKey carries the rate-limit key of the request's credential from
// authenticate to rateLimit.
type credentialKey struct{}

// authenticate denies by default (M2 design 3.6): every operation needs a
// valid bearer token, except the public ones, which never look at it.
//
// The failure gate comes first: a request with a token reserves a unit of
// its client IP's auth_failure bucket before the authenticator runs, and
// gets 429 without running it when the bucket is empty. A credential that
// fails keeps the unit; success, an expired access token and an internal
// fault give it back. Reserving first holds concurrent requests to the
// bucket too, so failures never exceed it.
func (a *API) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.public[r.Pattern] {
			next.ServeHTTP(w, r)
			return
		}
		token, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok {
			a.unauthorized(w, r, errors.New("no bearer token"), false)
			return
		}
		refund, retry, ok := a.authFailure.Reserve(RequestMetaFrom(r.Context()).IPKey)
		if !ok {
			a.tooManyRequests(w, r, "auth_failure", retry)
			return
		}
		ctx, credential, err := a.authenticator.Authenticate(r.Context(), token)
		if err != nil {
			var pe ProblemError
			if errors.As(err, &pe) && pe.ProblemStatus() == http.StatusUnauthorized {
				var ec expiredCredential
				if errors.As(err, &ec) && ec.ExpiredCredential() {
					refund()
				}
				a.unauthorized(w, r, err, true)
				return
			}
			refund()
			a.Errors.Write(w, r, err)
			return
		}
		refund()
		next.ServeHTTP(w, r.WithContext(context.WithValue(ctx, credentialKey{}, credential)))
	})
}

// unauthorized answers 401 with WWW-Authenticate (RFC 6750 3). Why the
// credential failed goes to the debug log only.
func (a *API) unauthorized(w http.ResponseWriter, r *http.Request, reason error, invalidToken bool) {
	a.logger.LogAttrs(r.Context(), slog.LevelDebug, "authentication failed",
		slog.String("request_id", RequestID(r.Context())), slog.String("route", r.Pattern), slog.Any("error", reason))
	challenge, detail := "Bearer", "This operation requires a bearer token."
	if invalidToken {
		challenge, detail = `Bearer error="invalid_token"`, "The bearer token is invalid or has expired."
	}
	w.Header().Set("WWW-Authenticate", challenge)
	WriteProblem(w, Problem{
		Status: http.StatusUnauthorized,
		Code:   CodeUnauthorized,
		Title:  http.StatusText(http.StatusUnauthorized),
		Detail: detail,
	})
}

// bearerToken returns the token of an "Authorization: Bearer <token>"
// header; the scheme is case-insensitive (RFC 9110 11.1).
func bearerToken(header string) (string, bool) {
	scheme, token, ok := strings.Cut(header, " ")
	token = strings.TrimSpace(token)
	if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" || strings.ContainsAny(token, " \t") {
		return "", false
	}
	return token, true
}
```

`server/internal/platform/httpserver/problem.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/platform/httpserver/problem.go
+++ b/server/internal/platform/httpserver/problem.go
@@ -20,6 +20,7 @@
 	CodeUnauthorized    = "unauthorized"
 	CodeNotFound        = "not_found"
 	CodePayloadTooLarge = "payload_too_large"
+	CodeRateLimited     = "rate_limited"
 	CodeInternal        = "internal_error"
 	CodeNotReady        = "not_ready"
 )
```

- [ ] **Step 2: 平台的测试**

`server/internal/platform/httpserver/api_test.go`（完整内容）：

```go
package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/bodyshape"
)

const (
	privateRoute = "POST /api/v0/things"
	publicRoute  = "POST /api/v0/open"
)

type callerKey struct{}

// fakeAuth accepts every token except "bad" (401), "expired" (401 with
// ExpiredCredential) and "boom" (a fault).
type fakeAuth struct {
	calls int
	ctx   context.Context // what Authenticate was given
	token string
}

func (f *fakeAuth) Authenticate(ctx context.Context, token string) (context.Context, string, error) {
	f.calls++
	f.ctx, f.token = ctx, token
	switch token {
	case "bad":
		return nil, "", problemErr{status: http.StatusUnauthorized, code: "unauthorized", detail: "Authentication is required."}
	case "expired":
		return nil, "", fmt.Errorf("check token: %w", expiredErr{problemErr{status: http.StatusUnauthorized, code: "unauthorized", detail: "expired"}})
	case "boom":
		return nil, "", errors.New("database is down")
	}
	return context.WithValue(ctx, callerKey{}, "caller-"+token), "session:" + token, nil
}

// expiredErr is the 401 of an access token whose exp has passed.
type expiredErr struct{ problemErr }

func (expiredErr) ExpiredCredential() bool { return true }

// fakeLimiter gives each key burst units and never refills them. It records
// the keys it took a unit of and the keys it got one back for.
type fakeLimiter struct {
	mu      sync.Mutex
	burst   int
	retry   time.Duration
	used    map[string]int
	taken   []string
	refunds []string
}

func newFakeLimiter(burst int) *fakeLimiter {
	return &fakeLimiter{burst: burst, retry: 1500 * time.Millisecond, used: map[string]int{}}
}

func (f *fakeLimiter) take(key string) (time.Duration, bool) {
	if f.used[key] == f.burst {
		return f.retry, false
	}
	f.used[key]++
	f.taken = append(f.taken, key)
	return 0, true
}

func (f *fakeLimiter) Allow(key string) (time.Duration, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.take(key)
}

func (f *fakeLimiter) Reserve(key string) (func(), time.Duration, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if retry, ok := f.take(key); !ok {
		return nil, retry, false
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.used[key]--
			f.refunds = append(f.refunds, key)
		})
	}, 0, true
}

// left is what key has of its burst.
func (f *fakeLimiter) left(key string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.burst - f.used[key]
}

// thingBody is a table as bodyshapegen writes it: {"name": string}, closed.
func thingBody() *bodyshape.Table {
	return &bodyshape.Table{
		Nodes: []bodyshape.Node{
			{Types: bodyshape.Object, Extra: bodyshape.Closed, Items: bodyshape.Open, Props: map[string]int{"name": 1}, Required: []string{"name"}},
			{Types: bodyshape.String, Extra: bodyshape.Open, Items: bodyshape.Open},
		},
		Roots: map[string]int{privateRoute: 0, publicRoute: 0},
	}
}

type reached struct {
	called bool
	ctx    context.Context
	body   string
}

// mount registers h on a router behind api's middlewares, applied the way
// the generated code does: for each middleware in the list, h = m(h).
func mount(t *testing.T, api *API, logger *slog.Logger) (*Router, *reached) {
	t.Helper()
	got := &reached{}
	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.called, got.ctx = true, r.Context()
		data, err := io.ReadAll(r.Body)
		if err != nil {
			api.Errors.BodyError(w, r, err)
			return
		}
		got.body = string(data)
		w.WriteHeader(http.StatusNoContent)
	})
	for _, m := range api.Middlewares(thingBody()) {
		h = m(h)
	}
	router := NewRouter(logger)
	router.Handle(privateRoute, h)
	router.Handle(publicRoute, h)
	return router, got
}

// testAPIConfig has limiters that never run out in these tests.
func testAPIConfig(auth Authenticator, logger *slog.Logger) APIConfig {
	return APIConfig{
		Logger:           logger,
		Authenticator:    auth,
		PublicOperations: []string{publicRoute},
		MaxBodyBytes:     64,
		RequestTimeout:   2 * time.Second,
		IPv6PrefixLen:    64,
		Anonymous:        newFakeLimiter(100),
		Authenticated:    newFakeLimiter(100),
		AuthFailure:      newFakeLimiter(100),
	}
}

func buildAPI(t *testing.T, cfg APIConfig) *API {
	t.Helper()
	api, err := NewAPI(cfg)
	if err != nil {
		t.Fatalf("NewAPI() error = %v", err)
	}
	return api
}

func newTestAPI(t *testing.T, auth Authenticator, logger *slog.Logger) *API {
	t.Helper()
	return buildAPI(t, testAPIConfig(auth, logger))
}

func post(path, token, body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	r.RemoteAddr = "203.0.113.7:5555"
	r.Header.Set("User-Agent", "agent/1.0")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	return r
}

func decodeProblem(t *testing.T, rec *httptest.ResponseRecorder) Problem {
	t.Helper()
	var p Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("body %q: %v", rec.Body, err)
	}
	return p
}

func TestNewAPIRequiresItsDependencies(t *testing.T) {
	_, err := NewAPI(APIConfig{PublicOperations: []string{publicRoute}, MaxBodyBytes: 64})
	want := "httpserver: APIConfig lacks Logger, Authenticator, Anonymous, Authenticated, AuthFailure"
	if err == nil || err.Error() != want {
		t.Errorf("NewAPI(without dependencies) error = %v, want %q", err, want)
	}
	cfg := testAPIConfig(&fakeAuth{}, slog.New(slog.DiscardHandler))
	cfg.AuthFailure = nil
	if _, err := NewAPI(cfg); err == nil || err.Error() != "httpserver: APIConfig lacks AuthFailure" {
		t.Errorf("NewAPI(without AuthFailure) error = %v, want it named alone", err)
	}
}

// The authenticator already sees the request meta and the deadline: both run
// before authentication (M2 design 3.6).
func TestMetaAndDeadlineRunBeforeAuthentication(t *testing.T) {
	auth := &fakeAuth{}
	router, got := mount(t, newTestAPI(t, auth, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))

	rec := serve(router, post("/api/v0/things", "tok", `{"name":"a"}`))

	if rec.Code != http.StatusNoContent || !got.called {
		t.Fatalf("status = %d, handler called %v; want 204", rec.Code, got.called)
	}
	meta := RequestMetaFrom(auth.ctx)
	if meta.ClientIP != netip.MustParseAddr("203.0.113.7") || meta.UserAgent != "agent/1.0" {
		t.Errorf("meta seen by the authenticator = %+v", meta)
	}
	deadline, ok := auth.ctx.Deadline()
	if !ok || time.Until(deadline) > 2*time.Second {
		t.Errorf("authenticator's deadline = %v, %v; want within the 2s request timeout", deadline, ok)
	}
	if got.ctx.Value(callerKey{}) != "caller-tok" || got.body != `{"name":"a"}` {
		t.Errorf("handler saw caller %v and body %q; want the authenticator's context and the body unchanged", got.ctx.Value(callerKey{}), got.body)
	}
}

// Without a token the body is never looked at: authentication runs before
// the body structure check.
func TestAuthenticationRunsBeforeTheBodyCheck(t *testing.T) {
	router, got := mount(t, newTestAPI(t, &fakeAuth{}, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))

	rec := serve(router, post("/api/v0/things", "", `{"nope":1}`))

	if rec.Code != http.StatusUnauthorized || got.called {
		t.Errorf("status = %d, handler called %v; want 401 before the body is checked", rec.Code, got.called)
	}
}

// The body check reads the body through the body limit: the limit runs
// before it, and the check's own read is what fails.
func TestBodyLimitRunsBeforeTheBodyCheck(t *testing.T) {
	router, got := mount(t, newTestAPI(t, &fakeAuth{}, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))

	rec := serve(router, post("/api/v0/things", "tok", `{"name":"`+strings.Repeat("a", 100)+`"}`))

	if p := decodeProblem(t, rec); rec.Code != http.StatusRequestEntityTooLarge || p.Code != CodePayloadTooLarge || got.called {
		t.Errorf("response = %d %+v, handler called %v; want 413 payload_too_large", rec.Code, p, got.called)
	}
}

func TestBodyCheckAnswersEveryProblemAs400(t *testing.T) {
	router, got := mount(t, newTestAPI(t, &fakeAuth{}, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))

	rec := serve(router, post("/api/v0/things", "tok", `{"extra":1}`))

	want := `{"status":400,"code":"bad_request","title":"Bad Request","detail":"The request body does not match the API description.",` +
		`"errors":[{"field":"extra","code":"not_allowed","message":"is not a property of this request"},{"field":"name","code":"required","message":"is required"}]}` + "\n"
	if rec.Code != http.StatusBadRequest || rec.Body.String() != want || got.called {
		t.Errorf("response = %d %s, handler called %v; want 400 %s", rec.Code, rec.Body, got.called, want)
	}
}

func TestBodyThatIsNotJSONIs400WithAGenericDetail(t *testing.T) {
	router, _ := mount(t, newTestAPI(t, &fakeAuth{}, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))

	rec := serve(router, post("/api/v0/things", "tok", `{"name":`))

	if p := decodeProblem(t, rec); rec.Code != http.StatusBadRequest || p.Detail != "The request body could not be decoded." || len(p.Errors) != 0 {
		t.Errorf("response = %d %+v, want 400 with the generic detail", rec.Code, p)
	}
}

func TestAuthenticationDeniesByDefault(t *testing.T) {
	tests := []struct {
		name          string
		route, header string
		status        int
		challenge     string
		authCalls     int
	}{
		{"no token", "/api/v0/things", "", 401, "Bearer", 0},
		{"another scheme", "/api/v0/things", "Basic dXNlcjpwYXNz", 401, "Bearer", 0},
		{"empty bearer", "/api/v0/things", "Bearer ", 401, "Bearer", 0},
		{"invalid token", "/api/v0/things", "Bearer bad", 401, `Bearer error="invalid_token"`, 1},
		{"expired token", "/api/v0/things", "Bearer expired", 401, `Bearer error="invalid_token"`, 1},
		{"valid token", "/api/v0/things", "Bearer tok", 204, "", 1},
		{"scheme in lower case", "/api/v0/things", "bearer tok", 204, "", 1},
		{"authenticator fault", "/api/v0/things", "Bearer boom", 500, "", 1},
		{"public without a token", "/api/v0/open", "", 204, "", 0},
		{"public ignores a bad token", "/api/v0/open", "Bearer bad", 204, "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := &fakeAuth{}
			router, _ := mount(t, newTestAPI(t, auth, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))
			req := post(tt.route, "", `{"name":"a"}`)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			rec := serve(router, req)

			if rec.Code != tt.status || rec.Header().Get("WWW-Authenticate") != tt.challenge || auth.calls != tt.authCalls {
				t.Errorf("response = %d, WWW-Authenticate %q, authenticator called %d times; want %d, %q, %d",
					rec.Code, rec.Header().Get("WWW-Authenticate"), auth.calls, tt.status, tt.challenge, tt.authCalls)
			}
			if tt.status == 401 {
				if p := decodeProblem(t, rec); p.Code != CodeUnauthorized {
					t.Errorf("problem code = %q, want unauthorized", p.Code)
				}
			}
		})
	}
}

// Why a credential failed goes to the debug log, never into the response.
func TestAuthenticationFailureIsLoggedAtDebugLevel(t *testing.T) {
	logger, logs := captureLogs(t)
	router, _ := mount(t, newTestAPI(t, &fakeAuth{}, logger), slog.New(slog.DiscardHandler))

	rec := serve(router, post("/api/v0/things", "bad", `{"name":"a"}`))

	entry := findLog(logs(), "authentication failed")
	if entry == nil || entry["level"] != "DEBUG" || entry["route"] != privateRoute || entry["error"] != "Authentication is required." {
		t.Errorf("log = %v, want the reason at debug level", entry)
	}
	if strings.Contains(rec.Body.String(), "Authentication is required.") {
		t.Errorf("body %s carries the authenticator's reason", rec.Body)
	}
}

// The failure gate reserves a unit before the authenticator runs; only a
// credential that fails keeps it (M2 design 3.6).
func TestFailureGateKeepsTheUnitOfAFailedCredentialOnly(t *testing.T) {
	tests := []struct {
		name, route, token string
		status             int
		left               int // of the client's 3 units afterwards
		refunded           bool
	}{
		{"failed credential", "/api/v0/things", "bad", 401, 2, false},
		{"valid credential", "/api/v0/things", "tok", 204, 3, true},
		{"expired access token", "/api/v0/things", "expired", 401, 3, true},
		{"authenticator fault", "/api/v0/things", "boom", 500, 3, true},
		{"no token", "/api/v0/things", "", 401, 3, false},
		{"public operation", "/api/v0/open", "bad", 204, 3, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testAPIConfig(&fakeAuth{}, slog.New(slog.DiscardHandler))
			gate := newFakeLimiter(3)
			cfg.AuthFailure = gate
			router, _ := mount(t, buildAPI(t, cfg), slog.New(slog.DiscardHandler))

			rec := serve(router, post(tt.route, tt.token, `{"name":"a"}`))

			if rec.Code != tt.status || gate.left("203.0.113.7") != tt.left || (len(gate.refunds) == 1) != tt.refunded {
				t.Errorf("response %d, units left %d, refunds %q; want %d, %d, refunded %v",
					rec.Code, gate.left("203.0.113.7"), gate.refunds, tt.status, tt.left, tt.refunded)
			}
		})
	}
}

// An empty gate answers 429 without running the authenticator, for every
// token from that client, valid ones too; other clients are not affected.
func TestFailureGateTurnsAwayWithoutAuthenticating(t *testing.T) {
	logger, logs := captureLogs(t)
	auth := &fakeAuth{}
	cfg := testAPIConfig(auth, logger)
	cfg.AuthFailure = newFakeLimiter(2)
	router, got := mount(t, buildAPI(t, cfg), slog.New(slog.DiscardHandler))
	for range 2 {
		serve(router, post("/api/v0/things", "bad", `{"name":"a"}`))
	}

	for _, token := range []string{"bad", "tok"} {
		rec := serve(router, post("/api/v0/things", token, `{"name":"a"}`))

		if p := decodeProblem(t, rec); rec.Code != http.StatusTooManyRequests || p.Code != CodeRateLimited || rec.Header().Get("Retry-After") != "2" {
			t.Errorf("token %s: response = %d %+v, Retry-After %q; want 429 rate_limited, Retry-After 2", token, rec.Code, p, rec.Header().Get("Retry-After"))
		}
	}
	if auth.calls != 2 || got.called {
		t.Errorf("authenticator called %d times, handler called %v; want 2 and false", auth.calls, got.called)
	}
	entry := findLog(logs(), "rate limited")
	if entry == nil || entry["level"] != "INFO" || entry["bucket"] != "auth_failure" || entry["ip"] != "203.0.113.7" {
		t.Errorf("log = %v, want the auth_failure bucket and the client at info level", entry)
	}

	other := post("/api/v0/things", "tok", `{"name":"a"}`)
	other.RemoteAddr = "198.51.100.1:5555"
	if rec := serve(router, other); rec.Code != http.StatusNoContent {
		t.Errorf("another client: status = %d, want 204", rec.Code)
	}
}

// Reserving before authenticating holds concurrent failures to the burst:
// 50 invalid tokens at once, 3 units, 3 authentications.
func TestFailureGateHoldsUnderConcurrency(t *testing.T) {
	auth := &slowFailingAuth{}
	cfg := testAPIConfig(auth, slog.New(slog.DiscardHandler))
	cfg.AuthFailure = newFakeLimiter(3)
	router, _ := mount(t, buildAPI(t, cfg), slog.New(slog.DiscardHandler))

	var wg sync.WaitGroup
	var mu sync.Mutex
	codes := map[int]int{}
	for range 50 {
		wg.Go(func() {
			rec := serve(router, post("/api/v0/things", "bad", `{"name":"a"}`))
			mu.Lock()
			defer mu.Unlock()
			codes[rec.Code]++
		})
	}
	wg.Wait()

	if auth.calls() != 3 || codes[http.StatusUnauthorized] != 3 || codes[http.StatusTooManyRequests] != 47 {
		t.Errorf("authenticator called %d times, responses %v; want 3, three 401 and 47 429", auth.calls(), codes)
	}
}

// slowFailingAuth rejects every token, slowly enough that concurrent
// requests are inside it together.
type slowFailingAuth struct {
	mu sync.Mutex
	n  int
}

func (s *slowFailingAuth) Authenticate(context.Context, string) (context.Context, string, error) {
	s.mu.Lock()
	s.n++
	s.mu.Unlock()
	time.Sleep(20 * time.Millisecond)
	return nil, "", problemErr{status: http.StatusUnauthorized, code: "unauthorized", detail: "no such session"}
}

func (s *slowFailingAuth) calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.n
}

// The gate counts a client by its IP key: two addresses of one IPv6 /64 are
// one client.
func TestFailureGateCountsByTheIPKey(t *testing.T) {
	cfg := testAPIConfig(&fakeAuth{}, slog.New(slog.DiscardHandler))
	gate := newFakeLimiter(1)
	cfg.AuthFailure = gate
	router, _ := mount(t, buildAPI(t, cfg), slog.New(slog.DiscardHandler))
	first := post("/api/v0/things", "bad", `{"name":"a"}`)
	first.RemoteAddr = "[2001:db8:1:2::7]:443"
	second := post("/api/v0/things", "bad", `{"name":"a"}`)
	second.RemoteAddr = "[2001:db8:1:2::8]:443"

	serve(router, first)
	rec := serve(router, second)

	if rec.Code != http.StatusTooManyRequests || len(gate.taken) != 1 || gate.taken[0] != "2001:db8:1:2::/64" {
		t.Errorf("second address: status %d, units taken for %q; want 429 and one unit of 2001:db8:1:2::/64", rec.Code, gate.taken)
	}
}

// A request takes a unit of anonymous by its IP key when it carries no
// credential, else of authenticated by its credential (M2 design 3.10). A
// request authentication turns away takes neither.
func TestRateLimitPicksTheBucketAndKey(t *testing.T) {
	tests := []struct {
		name, route, token string
		status             int
		anonymous          []string
		authenticated      []string
	}{
		{"public operation", "/api/v0/open", "", 204, []string{"203.0.113.7"}, nil},
		{"public operation with a token", "/api/v0/open", "tok", 204, []string{"203.0.113.7"}, nil},
		{"authenticated", "/api/v0/things", "tok", 204, nil, []string{"session:tok"}},
		{"no token", "/api/v0/things", "", 401, nil, nil},
		{"failed credential", "/api/v0/things", "bad", 401, nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testAPIConfig(&fakeAuth{}, slog.New(slog.DiscardHandler))
			anonymous, authenticated := newFakeLimiter(5), newFakeLimiter(5)
			cfg.Anonymous, cfg.Authenticated = anonymous, authenticated
			router, _ := mount(t, buildAPI(t, cfg), slog.New(slog.DiscardHandler))

			rec := serve(router, post(tt.route, tt.token, `{"name":"a"}`))

			if rec.Code != tt.status || !slices.Equal(anonymous.taken, tt.anonymous) || !slices.Equal(authenticated.taken, tt.authenticated) {
				t.Errorf("response %d, anonymous took %q, authenticated took %q; want %d, %q, %q",
					rec.Code, anonymous.taken, authenticated.taken, tt.status, tt.anonymous, tt.authenticated)
			}
		})
	}
}

// Over its rate, a request gets 429 with Retry-After in whole seconds,
// rounded up, before its body is looked at; the log names the bucket.
func TestRateLimitedRequestIs429(t *testing.T) {
	tests := []struct {
		name, route, token, bucket string
	}{
		{"anonymous", "/api/v0/open", "", "anonymous"},
		{"authenticated", "/api/v0/things", "tok", "authenticated"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, logs := captureLogs(t)
			cfg := testAPIConfig(&fakeAuth{}, logger)
			cfg.Anonymous, cfg.Authenticated = newFakeLimiter(0), newFakeLimiter(0)
			router, got := mount(t, buildAPI(t, cfg), slog.New(slog.DiscardHandler))

			rec := serve(router, post(tt.route, tt.token, `{"nope":1}`))

			want := `{"status":429,"code":"rate_limited","title":"Too Many Requests","detail":"Too many requests; retry later."}` + "\n"
			if rec.Code != http.StatusTooManyRequests || rec.Body.String() != want || rec.Header().Get("Retry-After") != "2" || got.called {
				t.Errorf("response = %d %s, Retry-After %q, handler called %v; want 429 %s, Retry-After 2",
					rec.Code, rec.Body, rec.Header().Get("Retry-After"), got.called, want)
			}
			entry := findLog(logs(), "rate limited")
			if entry == nil || entry["level"] != "INFO" || entry["bucket"] != tt.bucket || entry["ip"] != "203.0.113.7" || entry["request_id"] == nil {
				t.Errorf("log = %v, want bucket %s, the client and the request id at info level", entry, tt.bucket)
			}
		})
	}
}

func TestRequestMetaClientIP(t *testing.T) {
	tests := []struct{ remote, want string }{
		{"203.0.113.7:5555", "203.0.113.7"},
		{"[2001:db8::1]:443", "2001:db8::1"},
		{"[::ffff:192.0.2.1]:80", "192.0.2.1"},
		{"[fe80::1%en0]:80", "fe80::1"},
	}
	for _, tt := range tests {
		auth := &fakeAuth{}
		router, _ := mount(t, newTestAPI(t, auth, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))
		req := post("/api/v0/things", "tok", `{"name":"a"}`)
		req.RemoteAddr = tt.remote

		serve(router, req)

		if got := RequestMetaFrom(auth.ctx).ClientIP; got != netip.MustParseAddr(tt.want) {
			t.Errorf("RemoteAddr %s: ClientIP = %v, want %s", tt.remote, got, tt.want)
		}
	}
}

func TestRequestMetaOutsideTheMiddlewaresIsZero(t *testing.T) {
	if m := RequestMetaFrom(context.Background()); m.ClientIP.IsValid() || m.IPKey != "" || m.UserAgent != "" {
		t.Errorf("RequestMetaFrom() = %+v, want the zero value", m)
	}
}
```

`server/internal/platform/httpserver/clientip_test.go`（对 Task 4 版本的差异）：

```diff
--- a/server/internal/platform/httpserver/clientip_test.go
+++ b/server/internal/platform/httpserver/clientip_test.go
@@ -115,7 +115,7 @@
 // for logs and sessions, the key for the buckets.
 func TestRequestMetaCarriesTheClientAndItsKey(t *testing.T) {
 	auth := &fakeAuth{}
-	api := newTestAPI(auth, slog.New(slog.DiscardHandler))
+	api := newTestAPI(t, auth, slog.New(slog.DiscardHandler))
 	router, _ := mount(t, api, slog.New(slog.DiscardHandler))
 	req := post("/api/v0/things", "tok", `{"name":"a"}`)
 	req.RemoteAddr = "[2001:db8:1:2::7]:443"
```

`server/internal/platform/httpserver/contract_test.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/platform/httpserver/contract_test.go
+++ b/server/internal/platform/httpserver/contract_test.go
@@ -18,6 +18,8 @@
 	discard := slog.New(slog.DiscardHandler)
 	errs := NewAPIErrors(discard)
 	notReady := Check{Name: "database", Run: func(context.Context) error { return errors.New("down") }}
+	limited := testAPIConfig(&fakeAuth{}, discard)
+	limited.Anonymous = newFakeLimiter(0)
 	tests := []struct {
 		name   string
 		h      http.Handler
@@ -39,7 +41,8 @@
 		{"payload too large", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
 			errs.Write(w, r, &http.MaxBytesError{Limit: 1024})
 		}), "/api/v0/things", http.StatusRequestEntityTooLarge},
-		{"unauthorized", newTestAPI(&fakeAuth{}, discard).authenticate(http.NotFoundHandler()), "/api/v0/things", http.StatusUnauthorized},
+		{"unauthorized", newTestAPI(t, &fakeAuth{}, discard).authenticate(http.NotFoundHandler()), "/api/v0/things", http.StatusUnauthorized},
+		{"rate limited", buildAPI(t, limited).rateLimit(http.NotFoundHandler()), "/api/v0/things", http.StatusTooManyRequests},
 		{"field errors", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
 			errs.Write(w, r, problemErr{
 				status: http.StatusUnprocessableEntity, code: "validation_failed", detail: "The request has invalid values.",
```

- [ ] **Step 3: `apitest` 的测试卫生**

`server/internal/platform/httpserver/apitest/problems.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/platform/httpserver/apitest/problems.go
+++ b/server/internal/platform/httpserver/apitest/problems.go
@@ -135,8 +135,8 @@
 		}
 		if missing := unanswered(doc, answered.snapshot()); len(missing) > 0 {
 			fmt.Fprintf(os.Stderr, "api/modules/%s.yaml declares problem codes that no test answered through CheckResponse:\n", module)
-			for _, m := range missing {
-				fmt.Fprintln(os.Stderr, "  "+m)
+			for _, entry := range missing {
+				fmt.Fprintln(os.Stderr, "  "+entry)
 			}
 			code = 1
 		}
```

`server/internal/platform/httpserver/apitest/problems_test.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/platform/httpserver/apitest/problems_test.go
+++ b/server/internal/platform/httpserver/apitest/problems_test.go
@@ -2,6 +2,7 @@
 
 import (
 	"io"
+	"maps"
 	"net/http"
 	"net/http/httptest"
 	"os"
@@ -96,17 +97,23 @@
 	}
 }
 
+// The operation and its code are this test's own: no other test answers
+// them, so what the recorder holds for them is what CheckResponse put there.
 func TestCheckResponseRecordsTheAnsweredCode(t *testing.T) {
-	c := contractFrom(t, thingsContract)
+	doc := strings.NewReplacer("operationId: createThing", "operationId: recordThing", "[things.taken]", "[things.recorded]").Replace(thingsContract)
+	c := contractFrom(t, doc)
+	if got := answered.snapshot()["recordThing"]; got != nil {
+		t.Fatalf("recordThing has answers %v before CheckResponse", got)
+	}
 	rec := httptest.NewRecorder()
 	rec.Header().Set("Content-Type", "application/problem+json")
 	rec.WriteHeader(http.StatusConflict)
-	_, _ = rec.WriteString(`{"code":"things.taken"}`)
+	_, _ = rec.WriteString(`{"code":"things.recorded"}`)
 
 	c.CheckResponse(t, httptest.NewRequest(http.MethodPost, "/api/v0/things", nil), rec.Result())
 
-	if !answered.snapshot()["createThing"]["things.taken"] {
-		t.Error("createThing: things.taken was not recorded")
+	if got := answered.snapshot()["recordThing"]; !maps.Equal(got, map[string]bool{"things.recorded": true}) {
+		t.Errorf("recordThing answered %v, want only things.recorded", got)
 	}
 }
 
```

- [ ] **Step 4: 过期的访问令牌退回单位**

`server/internal/modules/identity/adapter/authn/authenticator.go`（完整内容）：

```go
// Package authn implements the platform's httpserver.Authenticator with the
// identity module's authentication use case (M2 design 3.6).
package authn

import (
	"context"
	"errors"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// AuthenticateUseCase is app.Authenticate.
type AuthenticateUseCase interface {
	Execute(ctx context.Context, token string) (shared.Actor, error)
}

// Authenticator puts the request's actor in the context.
type Authenticator struct {
	uc AuthenticateUseCase
}

// New returns the authenticator over uc.
func New(uc AuthenticateUseCase) *Authenticator {
	return &Authenticator{uc: uc}
}

// Authenticate returns a context carrying the actor of token and the
// caller's rate-limit key, session:<id>. An invalid token is the use case's
// 401 *shared.Error; for an access token that is valid but for its expiry,
// that error also reports ExpiredCredential() true, so the platform's
// failure gate does not count it. Any other error passes through as an
// internal fault.
func (a *Authenticator) Authenticate(ctx context.Context, token string) (context.Context, string, error) {
	actor, err := a.uc.Execute(ctx, token)
	if errors.Is(err, app.ErrAccessTokenExpired) {
		return nil, "", expired{err}
	}
	if err != nil {
		return nil, "", err
	}
	return shared.WithActor(ctx, actor), "session:" + actor.SessionID.String(), nil
}

// expired is the 401 of an expired access token: the client's cue to
// refresh (M2 design 3.6).
type expired struct{ error }

func (e expired) Unwrap() error { return e.error }

// ExpiredCredential tells the platform's failure gate to give the unit back.
func (expired) ExpiredCredential() bool { return true }
```

`server/internal/modules/identity/adapter/authn/authenticator_test.go`（完整内容）：

```go
package authn_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/authn"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The platform's port, satisfied by structure.
var _ httpserver.Authenticator = (*authn.Authenticator)(nil)

// fakeUseCase answers its one token; any other token is a test error.
type fakeUseCase struct {
	token string
	actor shared.Actor
	err   error
}

func (f fakeUseCase) Execute(_ context.Context, token string) (shared.Actor, error) {
	if token != f.token {
		return shared.Actor{}, fmt.Errorf("fakeUseCase got token %q, want %q", token, f.token)
	}
	return f.actor, f.err
}

func TestAuthenticatePutsTheActorInTheContext(t *testing.T) {
	actor := shared.Actor{UserID: uuid.MustParse("0199a2b4-0000-7000-8000-000000000001"), SessionID: uuid.MustParse("0199a2b4-0000-7000-8000-000000000002")}

	ctx, key, err := authn.New(fakeUseCase{token: "token", actor: actor}).Authenticate(context.Background(), "token")

	got, actorErr := shared.RequireActor(ctx)
	if err != nil || actorErr != nil || got != actor || key != "session:0199a2b4-0000-7000-8000-000000000002" {
		t.Errorf("Authenticate() = actor %+v (%v), key %q, %v", got, actorErr, key, err)
	}
}

// expiredCredential is the platform's optional interface on a 401.
type expiredCredential interface{ ExpiredCredential() bool }

func TestAuthenticatePassesErrorsThrough(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		unauthorized bool // whether the error is a 401 ProblemError
		expired      bool // whether it reports ExpiredCredential() true
	}{
		{"invalid credential", fmt.Errorf("%w: %w", shared.Unauthenticated(), errors.New("session is revoked")), true, false},
		{"expired access token", fmt.Errorf("%w: %w", shared.Unauthenticated(), app.ErrAccessTokenExpired), true, true},
		{"internal fault", errors.New("database is down"), false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, key, err := authn.New(fakeUseCase{token: "token", err: tt.err}).Authenticate(context.Background(), "token")

			var ec expiredCredential
			isExpired := errors.As(err, &ec) && ec.ExpiredCredential()
			if !errors.Is(err, tt.err) || err.Error() != tt.err.Error() || ctx != nil || key != "" || isExpired != tt.expired {
				t.Errorf("Authenticate() = %v, %q, %v (expired %v); want nil, \"\", %v (expired %v)", ctx, key, err, isExpired, tt.err, tt.expired)
			}
			var pe httpserver.ProblemError
			if is401 := errors.As(err, &pe) && pe.ProblemStatus() == 401; is401 != tt.unauthorized {
				t.Errorf("Authenticate() error %v is a 401: %v, want %v", err, is401, tt.unauthorized)
			}
		})
	}
}
```

`server/internal/modules/identity/app/authenticate_test.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/modules/identity/app/authenticate_test.go
+++ b/server/internal/modules/identity/app/authenticate_test.go
@@ -91,17 +91,20 @@
 	}
 }
 
-// Only a valid signature with a past exp is "expired": the client's cue to
-// refresh, which the M2/P2 failure gate does not count.
+// Only a valid signature whose exp has come, at that instant or past it, is
+// "expired": the client's cue to refresh, which the failure gate does not
+// count (M2 design 3.6).
 func TestAuthenticateTellsAnExpiredAccessToken(t *testing.T) {
-	uc, tokens, _ := newAuthenticate(validCredential(), nil)
-	old, _ := tokens.Issue(app.AccessClaims{UserID: userID, SessionID: sessionID, ExpiresAt: now})
+	for _, exp := range []time.Time{now, now.Add(-time.Second)} {
+		uc, tokens, _ := newAuthenticate(validCredential(), nil)
+		old, _ := tokens.Issue(app.AccessClaims{UserID: userID, SessionID: sessionID, ExpiresAt: exp})
 
-	_, expired := uc.Execute(context.Background(), old)
-	_, forged := uc.Execute(context.Background(), "forged")
+		_, expired := uc.Execute(context.Background(), old)
+		_, forged := uc.Execute(context.Background(), "forged")
 
-	if !errors.Is(expired, app.ErrAccessTokenExpired) || errors.Is(forged, app.ErrAccessTokenExpired) {
-		t.Errorf("expired = %v, forged = %v; want only the first to be ErrAccessTokenExpired", expired, forged)
+		if !errors.Is(expired, app.ErrAccessTokenExpired) || errors.Is(forged, app.ErrAccessTokenExpired) {
+			t.Errorf("exp %v: expired = %v, forged = %v; want only the first to be ErrAccessTokenExpired", exp, expired, forged)
+		}
 	}
 }
 
```

- [ ] **Step 5: 契约**

`api/openapi.yaml`（对 `fd69736` 的差异）：

```diff
--- a/api/openapi.yaml
+++ b/api/openapi.yaml
@@ -13,8 +13,8 @@
   license:
     name: AGPL-3.0-only
     identifier: AGPL-3.0-only
-# 所有操作都可能返回的平台错误码，只写在这里（M2 设计 3.11）。rate_limited 随 M2/P2 的限流加入
-x-problem-codes: [bad_request, payload_too_large, internal_error]
+# 所有操作都可能返回的平台错误码，只写在这里（M2 设计 3.11）
+x-problem-codes: [bad_request, payload_too_large, rate_limited, internal_error]
 tags:
   - name: identity
     description: Accounts, sign-in and sessions.
```

`api/common.yaml`（对 `fd69736` 的差异）：

```diff
--- a/api/common.yaml
+++ b/api/common.yaml
@@ -24,9 +24,9 @@
           description: >-
             Stable error code. Platform codes have no prefix (bad_request,
             unauthorized, not_found, payload_too_large, validation_failed,
-            server_busy, internal_error, not_ready); module codes are prefixed
-            with the module, e.g. identity.email_taken. Each operation lists
-            the codes it can answer in x-problem-codes.
+            rate_limited, server_busy, internal_error, not_ready); module
+            codes are prefixed with the module, e.g. identity.email_taken.
+            Each operation lists the codes it can answer in x-problem-codes.
           type: string
         title:
           description: HTTP status phrase, e.g. "Not Found".
```

`api/modules/instance.yaml`（对 `fd69736` 的差异）：

```diff
--- a/api/modules/instance.yaml
+++ b/api/modules/instance.yaml
@@ -33,6 +33,21 @@
   responses:
     Problem:
       description: Error (RFC 9457 problem details).
+      headers:
+        Retry-After:
+          description: >-
+            Whole seconds to wait before trying again, rounded up; sent with
+            rate_limited and server_busy.
+          schema:
+            type: integer
+            minimum: 1
+        WWW-Authenticate:
+          description: >-
+            Sent with every 401 (RFC 9110 15.5.2): Bearer, or
+            Bearer error="invalid_token" when the bearer token sent is invalid
+            or has expired (RFC 6750 3).
+          schema:
+            type: string
       content:
         application/problem+json:
           schema:
```

`api/modules/identity.yaml`（对 `fd69736` 的差异）：

```diff
--- a/api/modules/identity.yaml
+++ b/api/modules/identity.yaml
@@ -60,6 +60,21 @@
   responses:
     Problem:
       description: Error (RFC 9457 problem details).
+      headers:
+        Retry-After:
+          description: >-
+            Whole seconds to wait before trying again, rounded up; sent with
+            rate_limited and server_busy.
+          schema:
+            type: integer
+            minimum: 1
+        WWW-Authenticate:
+          description: >-
+            Sent with every 401 (RFC 9110 15.5.2): Bearer, or
+            Bearer error="invalid_token" when the bearer token sent is invalid
+            or has expired (RFC 6750 3).
+          schema:
+            type: string
       content:
         application/problem+json:
           schema:
```

- [ ] **Step 6: 生成**

Run: `make gen`
Expected: 以下五个文件改变（`Problem` 的说明、两个模块的 `ProblemResponseHeaders`），其余生成物不变：

| 生成的文件 | SHA-256 | 行数 |
|---|---|---|
| `api/dist/openapi.yaml` | `69d0402019448b13a94519381a21ce0f424e11f6bc8fe68a5a260b25045b8cea` | 270 |
| `server/internal/modules/identity/adapter/http/gen/server.gen.go` | `be409d588bee19f2bf271499a4317255746f1d6c172687b9ca8090da9e8fb1c2` | 461 |
| `server/internal/modules/instance/adapter/http/gen/server.gen.go` | `08e296f49b255e2f30665caacca9ad52d8215eb86e408131183dac2512b7765e` | 336 |
| `server/internal/platform/httpserver/apigen/components.gen.go` | `bd818c8b4d91bbe5246e4f3685c220e8fb355a19c5aa779e2ed13984e65efea7` | 77 |
| `web/packages/api-client/src/schema.gen.ts` | `61ea9c9e2ac7a0ee947502ca80804a5b2fd070578d3199b7396e4a7b7af82fa7` | 241 |

其中 `api/dist/openapi.yaml`、`schema.gen.ts` 和 identity 的 `server.gen.go` 在 Task 9 再次生成；`components.gen.go` 和 instance 的 `server.gen.go` 是最终版本。

- [ ] **Step 7: 模块和接线的测试服务**

`server/internal/modules/instance/adapter/http/handler_test.go`（完整内容）：

```go
package httpadapter_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/instance/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/ratelimit"
)

type fixedSource domain.Build

func (s fixedSource) Build() domain.Build { return domain.Build(s) }

// noAccounts authenticates nobody: every operation of instance is public, so
// it is never asked.
type noAccounts struct{}

func (noAccounts) Authenticate(context.Context, string) (context.Context, string, error) {
	return nil, "", errors.New("instance has no operation that needs a token")
}

func TestGetInstanceMatchesTheContract(t *testing.T) {
	contract := apitest.Load(t)
	logger := slog.New(slog.DiscardHandler)
	router := httpserver.NewRouter(logger)
	limit := ratelimit.New(time.Now).Bucket("test", ratelimit.Rate{PerMinute: 600, Burst: 100})
	api, err := httpserver.NewAPI(httpserver.APIConfig{
		Logger:           logger,
		Authenticator:    noAccounts{},
		PublicOperations: httpadapter.PublicOperations(),
		MaxBodyBytes:     1 << 20,
		RequestTimeout:   time.Second,
		IPv6PrefixLen:    64,
		Anonymous:        limit,
		Authenticated:    limit,
		AuthFailure:      limit,
	})
	if err != nil {
		t.Fatal(err)
	}
	getInfo := app.NewGetInfo(fixedSource{Version: "1.2.3", Commit: "4f2a9c1"})
	httpadapter.Register(router, api, httpadapter.UseCases{GetInfo: getInfo})
	req := httptest.NewRequest(http.MethodGet, "/api/v0/instance", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	res := rec.Result()

	contract.CheckResponse(t, req, res)
	body, _ := io.ReadAll(res.Body)
	want := `{"api_version":"v0","commit":"4f2a9c1","product":"Nerve","version":"1.2.3"}` + "\n"
	if res.StatusCode != http.StatusOK || string(body) != want {
		t.Errorf("GET /api/v0/instance = %d %s, want 200 %s", res.StatusCode, body, want)
	}
}
```

`server/internal/modules/identity/adapter/http/handler_test.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/modules/identity/adapter/http/handler_test.go
+++ b/server/internal/modules/identity/adapter/http/handler_test.go
@@ -17,6 +17,7 @@
 	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
 	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
 	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
+	"github.com/open-nerve/NerveProject/server/internal/platform/ratelimit"
 	"github.com/open-nerve/NerveProject/server/internal/shared"
 )
 
@@ -61,16 +62,25 @@
 	return shared.WithActor(ctx, shared.Actor{UserID: userID, SessionID: sessionID}), "session:" + sessionID.String(), nil
 }
 
-func newServer(register *fakeRegister) http.Handler {
+func newServer(t *testing.T, register *fakeRegister) http.Handler {
+	t.Helper()
 	logger := slog.New(slog.DiscardHandler)
 	router := httpserver.NewRouter(logger)
-	api := httpserver.NewAPI(httpserver.APIConfig{
+	limit := ratelimit.New(time.Now).Bucket("test", ratelimit.Rate{PerMinute: 600, Burst: 100})
+	api, err := httpserver.NewAPI(httpserver.APIConfig{
 		Logger:           logger,
 		Authenticator:    fakeAuth{},
 		PublicOperations: httpadapter.PublicOperations(),
 		MaxBodyBytes:     1024,
 		RequestTimeout:   5 * time.Second,
+		IPv6PrefixLen:    64,
+		Anonymous:        limit,
+		Authenticated:    limit,
+		AuthFailure:      limit,
 	})
+	if err != nil {
+		t.Fatal(err)
+	}
 	httpadapter.Register(router, api, httpadapter.UseCases{Register: register, GetMe: fakeGetMe{}})
 	return router
 }
@@ -100,7 +110,7 @@
 	req := registerRequest(`{"email":"Alice@Corp.com","password":"Tr0ub4dor&3"}`)
 	apitest.Load(t).CheckRequest(t, req)
 
-	res, body := do(t, newServer(register), req)
+	res, body := do(t, newServer(t, register), req)
 
 	want := `{"access_token":"access","access_token_expires_in":900,"refresh_token":"nrv_rt_x",` +
 		`"refresh_token_expires_at":"2026-10-25T10:00:00.123456Z","token_type":"Bearer"}` + "\n"
@@ -129,7 +139,7 @@
 	}
 	for _, tt := range tests {
 		t.Run(tt.name, func(t *testing.T) {
-			res, body := do(t, newServer(&fakeRegister{err: tt.err}), registerRequest(`{"email":"a@b.co","password":"x"}`))
+			res, body := do(t, newServer(t, &fakeRegister{err: tt.err}), registerRequest(`{"email":"a@b.co","password":"x"}`))
 
 			if res.StatusCode != tt.status || !strings.Contains(body, `"code":"`+tt.code+`"`) || res.Header.Get("Retry-After") != tt.retryAfter {
 				t.Errorf("response = %d %s Retry-After %q, want %d %s", res.StatusCode, body, res.Header.Get("Retry-After"), tt.status, tt.code)
@@ -156,7 +166,7 @@
 	for _, tt := range tests {
 		t.Run(tt.name, func(t *testing.T) {
 			register := &fakeRegister{}
-			res, body := do(t, newServer(register), registerRequest(tt.body))
+			res, body := do(t, newServer(t, register), registerRequest(tt.body))
 
 			if res.StatusCode != tt.status || !strings.Contains(body, tt.want) || strings.Contains(body, "Go struct") {
 				t.Errorf("response = %d %s, want %d with %s", res.StatusCode, body, tt.status, tt.want)
@@ -173,7 +183,7 @@
 	req.Header.Set("Authorization", "Bearer valid")
 	apitest.Load(t).CheckRequest(t, req)
 
-	res, body := do(t, newServer(&fakeRegister{}), req)
+	res, body := do(t, newServer(t, &fakeRegister{}), req)
 
 	want := `{"avatar_url":null,"cover_image_url":null,"created_at":"2026-09-25T10:00:00.123456Z","display_name":"alice",` +
 		`"email":"alice@corp.com","first_name":"","id":"` + userID.String() + `","last_name":"","user_timezone":"UTC"}` + "\n"
@@ -189,7 +199,7 @@
 			req.Header.Set("Authorization", header)
 		}
 
-		res, body := do(t, newServer(&fakeRegister{}), req)
+		res, body := do(t, newServer(t, &fakeRegister{}), req)
 
 		if res.StatusCode != http.StatusUnauthorized || !strings.Contains(body, `"code":"unauthorized"`) {
 			t.Errorf("GET /me with %q = %d %s, want 401 unauthorized", header, res.StatusCode, body)
```

`server/internal/bootstrap/app.go`（对 Task 4 版本的差异）：

```diff
--- a/server/internal/bootstrap/app.go
+++ b/server/internal/bootstrap/app.go
@@ -12,6 +12,7 @@
 	"net/netip"
 	"os"
 	"slices"
+	"time"
 
 	"github.com/jackc/pgx/v5/pgxpool"
 
@@ -21,6 +22,7 @@
 	"github.com/open-nerve/NerveProject/server/internal/platform/config"
 	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
 	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
+	"github.com/open-nerve/NerveProject/server/internal/platform/ratelimit"
 	"github.com/open-nerve/NerveProject/server/internal/platform/webui"
 	"github.com/open-nerve/NerveProject/server/internal/shared"
 )
@@ -96,7 +98,10 @@
 		httpserver.Check{Name: "migrations", Run: migrator.CheckUpToDate},
 	)
 	a.publicOperations = slices.Concat(ident.PublicOperations(), inst.PublicOperations())
-	api := httpserver.NewAPI(httpserver.APIConfig{
+	// time.Now, not the Clock: its monotonic reading keeps a step of the wall
+	// clock from filling or draining the buckets.
+	limiter := ratelimit.New(time.Now)
+	api, err := httpserver.NewAPI(httpserver.APIConfig{
 		Logger:           logger,
 		Authenticator:    ident.Authenticator(),
 		PublicOperations: a.publicOperations,
@@ -104,7 +109,14 @@
 		RequestTimeout:   cfg.Server.RequestTimeout,
 		TrustedProxies:   cfg.Server.TrustedProxies,
 		IPv6PrefixLen:    cfg.RateLimit.IPv6PrefixLen,
+		Anonymous:        bucket(limiter, "anonymous", cfg.RateLimit.Anonymous),
+		Authenticated:    bucket(limiter, "authenticated", cfg.RateLimit.Authenticated),
+		AuthFailure:      bucket(limiter, "auth_failure", cfg.RateLimit.AuthFailure),
 	})
+	if err != nil {
+		a.close()
+		return nil, err
+	}
 	// Modules mount their generated routes on this root router, next to the
 	// platform's /api/ fallback; an /api/v0/ sub-mux would shadow it.
 	ident.Register(a.router, api)
@@ -132,6 +144,12 @@
 	return data, nil
 }
 
+// bucket is the rate-limit bucket name on limiter, sized by c (M2 design
+// 3.10).
+func bucket(limiter *ratelimit.Limiter, name string, c config.BucketConfig) *ratelimit.Bucket {
+	return limiter.Bucket(name, ratelimit.Rate{PerMinute: c.PerMinute, Burst: c.Burst})
+}
+
 // signupSwitch is auth.signup_enabled as identity's SignupPolicy.
 type signupSwitch bool
 
```

`server/internal/bootstrap/app_test.go`（对 Task 4 版本的差异）：

```diff
--- a/server/internal/bootstrap/app_test.go
+++ b/server/internal/bootstrap/app_test.go
@@ -35,6 +35,9 @@
 
 var client = &http.Client{Timeout: 5 * time.Second}
 
+// roomy is a rate no test here reaches; the tests of a limit set their own.
+var roomy = config.BucketConfig{PerMinute: 600000, Burst: 100000}
+
 // testConfig listens on a port the system picks and reports it through
 // server.addr_file. Password hashing is cheap: the tests hash many times.
 func testConfig(t *testing.T, dbURL string, autoMigrate bool) config.Config {
@@ -64,8 +67,12 @@
 				MaxWait:             2 * time.Second,
 			},
 		},
-		RateLimit: config.RateLimitConfig{IPv6PrefixLen: 64},
-		Log:       config.LogConfig{Level: "error", Format: "text"},
+		RateLimit: config.RateLimitConfig{
+			IPv6PrefixLen: 64,
+			Anonymous:     roomy, AuthFailure: roomy, Authenticated: roomy,
+			LoginIP: roomy, LoginIPEmail: roomy, RegisterIP: roomy,
+		},
+		Log: config.LogConfig{Level: "error", Format: "text"},
 	}
 }
 
```

- [ ] **Step 8: lint、测试和端到端**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。

Run: `make lint-web`
Expected: 通过（`schema.gen.ts` 的 `Problem` 响应多出两个响应头）。

Run: `make e2e`
Expected: P1 的故事全部通过（test 配置的桶很高，不会被限流）。

- [ ] **Step 9: 提交**

```bash
git add api web/packages/api-client/src/schema.gen.ts server/internal/platform/httpserver server/internal/modules/identity/adapter/authn server/internal/modules/identity/app/authenticate_test.go server/internal/modules/identity/adapter/http server/internal/modules/instance/adapter/http server/internal/bootstrap/app.go server/internal/bootstrap/app_test.go
```
```bash
git commit -m "feat(M2/P2): the failure gate before authentication and the per-route rate limit

A request with a token reserves a unit of its client IP's auth_failure
bucket first; only a credential that fails keeps it. Over its bucket a
request gets 429 rate_limited with Retry-After, which the contract now
declares with WWW-Authenticate. NewAPI checks its dependencies.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run: `make gen-check`
Expected: 没有差异。

**Done when:** 失败闸门的四个测试（留下、拒绝时不认证、并发、按 IP 键）和限流的两个测试通过；五个生成物与上表相同；`make e2e` 中 P1 的故事通过；`make gen-check` 干净。

---

### Task 6: `identity` 领域：刷新令牌的解析、续期判定表、登录与续期的错误

**Files:**
- Modify: `server/internal/modules/identity/domain/session.go`、`errors.go`、`session_test.go`

**Interfaces:**
- Produces（spec 2.8）：
  - `domain.ParseRefreshToken(s string) (RefreshToken, bool)`：只接受 `String` 写出的写法（98 个字符、严格的 base64url、68 字节、代数不超过 `math.MaxInt32`）；
  - `domain.SessionState{Generation uint32; TokenHash []byte; Revoked bool; ExpiresAt time.Time}`；`domain.Verdict`（`Rotate`、`Reuse`、`Reject`）；`domain.JudgeRefresh(s, token, tagValid, now) Verdict`（M2 设计 3.5 的表）；
  - `domain.ErrInvalidCredentials`（401）、`ErrAccountDeactivated`（403）、`ErrRefreshTokenInvalid`（401）。

**Tests:**（`session_test.go`）
- `TestParseRefreshTokenReadsWhatStringWrites`；`TestParseRefreshTokenRejects`（13 个，含中间的换行、两端的空格、没用到的位不为 0、超出 `integer` 的代数）；`TestJudgeRefresh`（11 行，含恰好到期和到期前 1 微秒）。

- [ ] **Step 1: 解析和判定**

`server/internal/modules/identity/domain/session.go`（完整内容）：

```go
package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"math"
	"strings"
	"time"
	"unicode/utf8"
	"uuid"
)

// RefreshTokenPrefix starts every refresh token (M2 design 3.4).
const RefreshTokenPrefix = "nrv_rt_"

// The layout of a refresh token's 68 bytes (M2 design 3.4): session id,
// generation (big endian), secret, and the MAC tag of the first 52 bytes.
const (
	refreshTokenLen = 16 + 4 + 32 + 16
	macMessageLen   = 16 + 4 + 32
)

// RefreshToken is the content of a refresh token. The server stores only
// SecretHash of the current generation; Tag proves that the server issued a
// generation without storing it.
type RefreshToken struct {
	SessionID  uuid.UUID
	Generation uint32
	Secret     [32]byte
	Tag        [16]byte
}

// MACMessage is what the tag authenticates: session id ‖ generation ‖ secret.
func (t RefreshToken) MACMessage() []byte {
	return t.bytes()[:macMessageLen]
}

// SecretHash is the SHA-256 of the secret, what auth_sessions.token_hash holds.
func (t RefreshToken) SecretHash() []byte {
	h := sha256.Sum256(t.Secret[:])
	return h[:]
}

// String is the token the client holds: nrv_rt_ and the 68 bytes in
// unpadded base64url, 91 characters.
func (t RefreshToken) String() string {
	return RefreshTokenPrefix + base64.RawURLEncoding.EncodeToString(t.bytes())
}

func (t RefreshToken) bytes() []byte {
	b := make([]byte, 0, refreshTokenLen)
	b = append(b, t.SessionID[:]...)
	b = binary.BigEndian.AppendUint32(b, t.Generation)
	b = append(b, t.Secret[:]...)
	return append(b, t.Tag[:]...)
}

// refreshTokenTextLen is the length of a refresh token as the client holds it.
var refreshTokenTextLen = len(RefreshTokenPrefix) + base64.RawURLEncoding.EncodedLen(refreshTokenLen)

// ParseRefreshToken reads a token that String wrote, without looking
// anything up (M2 design 3.5). It accepts only that spelling: the prefix,
// then 91 characters of unpadded base64url whose unused last bits are zero,
// nothing around them. A generation beyond auth_sessions.generation's
// integer is not one the server issued.
func ParseRefreshToken(s string) (RefreshToken, bool) {
	if len(s) != refreshTokenTextLen || !strings.HasPrefix(s, RefreshTokenPrefix) {
		return RefreshToken{}, false
	}
	// A decoder skips \r and \n: with them inside, fewer than 68 bytes come out.
	raw, err := base64.RawURLEncoding.Strict().DecodeString(s[len(RefreshTokenPrefix):])
	if err != nil || len(raw) != refreshTokenLen {
		return RefreshToken{}, false
	}
	var t RefreshToken
	copy(t.SessionID[:], raw[0:16])
	t.Generation = binary.BigEndian.Uint32(raw[16:20])
	copy(t.Secret[:], raw[20:52])
	copy(t.Tag[:], raw[52:68])
	if t.Generation > math.MaxInt32 {
		return RefreshToken{}, false
	}
	return t, true
}

// SessionState is what a refresh judges a token against: the session's row.
type SessionState struct {
	Generation uint32
	TokenHash  []byte // SHA-256 of the current generation's secret
	Revoked    bool
	ExpiresAt  time.Time
}

// Verdict is what a refresh does with the token it is given.
type Verdict int

// The verdicts of M2 design 3.5's table.
const (
	// Rotate: the current generation with its secret, of a live session.
	Rotate Verdict = iota + 1
	// Reuse: an older generation that the session did issue (its tag
	// holds), of a live session: revoke the session.
	Reuse
	// Reject: anything else; the session stays as it is.
	Reject
)

// JudgeRefresh applies M2 design 3.5's table to token and the state of its
// session at now. tagValid is whether token's MAC tag holds; only an older
// generation needs it: the stored hash proves the current one, which keeps
// working across a change of the signing key. A session that does not
// exist is the caller's Reject.
func JudgeRefresh(s SessionState, token RefreshToken, tagValid bool, now time.Time) Verdict {
	switch {
	case s.Revoked || !now.Before(s.ExpiresAt):
		return Reject
	case token.Generation == s.Generation && bytes.Equal(token.SecretHash(), s.TokenHash):
		return Rotate
	case token.Generation < s.Generation && tagValid:
		return Reuse
	}
	return Reject
}

// MaxUserAgentLength bounds auth_sessions.user_agent, in characters.
const MaxUserAgentLength = 512

// SanitizeUserAgent is the User-Agent a session records: valid UTF-8, no
// NUL (Postgres text cannot hold it), at most 512 characters.
func SanitizeUserAgent(ua string) string {
	ua = strings.ReplaceAll(strings.ToValidUTF8(ua, string(utf8.RuneError)), "\x00", "")
	if utf8.RuneCountInString(ua) <= MaxUserAgentLength {
		return ua
	}
	return string([]rune(ua)[:MaxUserAgentLength])
}
```

`server/internal/modules/identity/domain/errors.go`（完整内容）：

```go
package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The identity module's errors (M2 design 5.4). api/modules/identity.yaml
// declares their codes in x-problem-codes.
var (
	// ErrSignupDisabled answers a well-formed registration while sign-up is
	// off, before the address or the password is looked at (M2 design 3.9);
	// the platform's structural 400 or 413 can come first.
	ErrSignupDisabled = shared.NewError(shared.KindForbidden, "identity.signup_disabled", "Sign-up is disabled on this instance.")
	// ErrEmailTaken answers a registration with an address in use.
	ErrEmailTaken = shared.NewError(shared.KindConflict, "identity.email_taken", "An account with this e-mail address already exists.")
	// ErrInvalidCredentials answers a login with an unknown address or a
	// wrong password alike (M2 design 3.9).
	ErrInvalidCredentials = shared.NewError(shared.KindUnauthenticated, "identity.invalid_credentials", "The e-mail address or the password is incorrect.")
	// ErrAccountDeactivated answers a login with the right password for a
	// deactivated account; only then is the state revealed (M2 design 3.9).
	ErrAccountDeactivated = shared.NewError(shared.KindForbidden, "identity.account_deactivated", "This account is deactivated.")
	// ErrRefreshTokenInvalid answers every refresh that does not rotate:
	// unknown, expired, revoked, reused or forged (M2 design 3.5).
	ErrRefreshTokenInvalid = shared.NewError(shared.KindUnauthenticated, "identity.refresh_token_invalid", "The refresh token is not valid; sign in again.")
)
```

- [ ] **Step 2: 测试**

`server/internal/modules/identity/domain/session_test.go`（完整内容）：

```go
package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"math"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
	"uuid"
)

func sampleToken() RefreshToken {
	t := RefreshToken{SessionID: uuid.MustParse("0199a2b4-7c3e-7d2a-9f10-2b3c4d5e6f70"), Generation: 0x01020304}
	for i := range t.Secret {
		t.Secret[i] = byte(0x40 + i)
	}
	for i := range t.Tag {
		t.Tag[i] = byte(0xa0 + i)
	}
	return t
}

// The 68 bytes: session id 0–15, generation 16–19 big endian, secret 20–51,
// tag 52–67 (M2 design 3.4).
func TestRefreshTokenLayout(t *testing.T) {
	tok := sampleToken()
	s := tok.String()

	// The secret-scanning pattern of M2 design 8.6.
	if !regexp.MustCompile(`^nrv_rt_[A-Za-z0-9_-]{91}$`).MatchString(s) {
		t.Fatalf("token %q does not match nrv_rt_[A-Za-z0-9_-]{91}", s)
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(s, RefreshTokenPrefix))
	if err != nil || len(raw) != 68 {
		t.Fatalf("decoded %d bytes, %v; want 68", len(raw), err)
	}
	if !bytes.Equal(raw[0:16], tok.SessionID[:]) || !bytes.Equal(raw[16:20], []byte{1, 2, 3, 4}) ||
		!bytes.Equal(raw[20:52], tok.Secret[:]) || !bytes.Equal(raw[52:68], tok.Tag[:]) {
		t.Errorf("layout = % x", raw)
	}
	if !bytes.Equal(tok.MACMessage(), raw[:52]) {
		t.Errorf("MACMessage() = % x, want the first 52 bytes", tok.MACMessage())
	}
}

func TestRefreshTokenSecretHash(t *testing.T) {
	tok := sampleToken()
	want := sha256.Sum256(tok.Secret[:])
	if got := tok.SecretHash(); !bytes.Equal(got, want[:]) || len(got) != 32 {
		t.Errorf("SecretHash() = % x, want the SHA-256 of the secret", got)
	}
}

func TestSanitizeUserAgent(t *testing.T) {
	long := strings.Repeat("é", 600)
	tests := []struct{ in, want string }{
		{"Mozilla/5.0", "Mozilla/5.0"},
		{"agent\x00/1", "agent/1"},
		{"bad \xff byte", "bad " + string(utf8.RuneError) + " byte"},
		{long, strings.Repeat("é", 512)},
	}
	for _, tt := range tests {
		got := SanitizeUserAgent(tt.in)
		if got != tt.want || !utf8.ValidString(got) {
			t.Errorf("SanitizeUserAgent(%.20q) = %.20q (%d runes), want %.20q", tt.in, got, utf8.RuneCountInString(got), tt.want)
		}
	}
}

func TestParseRefreshTokenReadsWhatStringWrites(t *testing.T) {
	for _, generation := range []uint32{0, 0x01020304, math.MaxInt32} {
		tok := sampleToken()
		tok.Generation = generation

		got, ok := ParseRefreshToken(tok.String())

		if !ok || got != tok {
			t.Errorf("ParseRefreshToken(String()) at generation %d = %+v, %v; want the token back", generation, got, ok)
		}
	}
}

func TestParseRefreshTokenRejects(t *testing.T) {
	good := sampleToken().String()
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	last := strings.IndexByte(alphabet, good[len(good)-1])
	beyond := sampleToken()
	beyond.Generation = math.MaxInt32 + 1
	tests := []struct{ name, token string }{
		{"empty", ""},
		{"the prefix alone", RefreshTokenPrefix},
		{"another prefix", "nrv_xx_" + good[len(RefreshTokenPrefix):]},
		{"a personal access token", "nrv_pat_" + strings.Repeat("A", 43)},
		{"one character short", good[:len(good)-1]},
		{"one character more", good + "A"},
		{"padded", good[:len(good)-1] + "="},
		{"standard base64", good[:20] + "+" + good[21:]},
		{"a newline inside", good[:50] + "\n" + good[51:]},
		{"a space in front", " " + good[:len(good)-1]},
		{"a space behind", good[1:] + " "},
		// 91 characters carry 546 bits for 544: the last two must be zero,
		// so that one token has one spelling.
		{"unused bits set", good[:len(good)-1] + string(alphabet[last|1])},
		{"a generation beyond integer", beyond.String()},
	}
	for _, tt := range tests {
		if got, ok := ParseRefreshToken(tt.token); ok {
			t.Errorf("%s: ParseRefreshToken(%q) = %+v, want false", tt.name, tt.token, got)
		}
	}
}

// Every row of M2 design 3.5's table. The session is at generation 5; a
// session that does not exist is the use case's to reject.
func TestJudgeRefresh(t *testing.T) {
	now := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	current := sampleToken()
	current.Generation = 5
	live := SessionState{Generation: 5, TokenHash: current.SecretHash(), ExpiresAt: now.Add(time.Hour)}
	older, newer, otherSecret := current, current, current
	older.Generation, newer.Generation = 4, 6
	otherSecret.Secret[0] ^= 1
	revoked, expired, lastInstant := live, live, live
	revoked.Revoked = true
	expired.ExpiresAt = now
	lastInstant.ExpiresAt = now.Add(time.Microsecond)
	tests := []struct {
		name     string
		state    SessionState
		token    RefreshToken
		tagValid bool
		want     Verdict
	}{
		{"current generation", live, current, false, Rotate},
		{"current generation, tag valid too", live, current, true, Rotate},
		{"current generation, another secret", live, otherSecret, true, Reject},
		{"older generation the session issued", live, older, true, Reuse},
		{"older generation, forged", live, older, false, Reject},
		{"newer generation", live, newer, true, Reject},
		{"revoked, current generation", revoked, current, false, Reject},
		{"revoked, older generation the session issued", revoked, older, true, Reject},
		{"expired at this instant", expired, current, false, Reject},
		{"expired, older generation the session issued", expired, older, true, Reject},
		{"a microsecond before expiry", lastInstant, current, false, Rotate},
	}
	for _, tt := range tests {
		if got := JudgeRefresh(tt.state, tt.token, tt.tagValid, now); got != tt.want {
			t.Errorf("%s: JudgeRefresh() = %d, want %d", tt.name, got, tt.want)
		}
	}
}
```

- [ ] **Step 3: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`（`ParseRefreshToken`、`JudgeRefresh` 和三个错误的使用者在 Task 7 加入。）

Run: `make test`
Expected: 全部 `ok`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/modules/identity/domain
```
```bash
git commit -m "feat(M2/P2): parse refresh tokens and judge a refresh by the design's table

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** `TestParseRefreshTokenRejects` 的 13 个写法都被拒绝；`TestJudgeRefresh` 的 11 行与 M2 设计 3.5 的表逐行对应。

---

### Task 7: `identity` 的端口与用例：`Issuance`、`Login`、`Refresh`、`Logout`；argon2 和 MAC 的校验

**Files:**
- Modify: `server/internal/modules/identity/app/ports.go`、`register.go`、`register_test.go`、`fakes_test.go`
- Create: `server/internal/modules/identity/app/issuer.go`、`login.go`、`login_test.go`、`refresh.go`、`refresh_test.go`、`logout.go`、`logout_test.go`
- Modify: `server/internal/modules/identity/adapter/argon2/hasher.go`、`hasher_test.go`
- Modify: `server/internal/modules/identity/adapter/signing/mac.go`、`signing_test.go`
- Modify: `server/internal/modules/identity/module.go`（过渡：只把注册改用 `Issuance`；Task 9 接上三个新用例）

**Interfaces:**
- Consumes: `domain.ParseRefreshToken`、`JudgeRefresh`、三个错误（Task 6）；P1 的 `AccessTokens`、`RefreshTokenMAC.Tag`、`PasswordHasher.Hash`、`SessionCreator`、`shared.TxManager`。
- Produces（spec 2.9）：
  - 端口 `LoginAccountReader`、`CredentialLocker`、`PasswordHashWriter`、`SessionRotator`、`SessionEnder`；类型 `LoginAccount`、`LockedAccount`、`RefreshSession`、`SessionGeneration`；`PasswordHasher.Verify`、`RefreshTokenMAC.Verify`；
  - `app.Issuance{Tokens, MAC, AccessTTL, SessionTTL}`（`Tokens` 移到 `issuer.go`）；`RegisterDeps.Issuance` 替代原来的四个字段；
  - `app.NewLogin(LoginDeps{Accounts, Locker, Passwords, Sessions, Hasher, Tx, Issuance, Clock, Logger, DummyHash})`、`(*Login).Execute(ctx, LoginInput{Email, Password, UserAgent, IP})`；
  - `app.NewRefresh(RefreshDeps{Sessions, Tx, Issuance, Clock, Logger})`、`(*Refresh).Execute(ctx, token, ip)`；
  - `app.NewLogout(sessions, clock, logger)`、`(*Logout).Execute(ctx, token)`；
  - `argon2adapter.(*Hasher).Verify(ctx, password, hash) (ok, rehash bool, err error)`；`signing.(*RefreshTokenMAC).Verify(message, tag) bool`。

**Tests:**
- `login_test.go`（9 个）：`TestLoginSignsIn`、`TestLoginFailsAlikeForAnUnknownAddressAndAWrongPassword`（5 个）、`TestLoginRevealsDeactivationOnlyWithTheRightPassword`、`TestLoginRehashesWhenTheParametersChanged`、`TestLoginVerifiesAgainWhenAConcurrentLoginRehashed`、`TestLoginFailsWhenThePasswordChangedMeanwhile`、`TestLoginFailsWhenTheHashChangesTwice`、`TestLoginWhenTheHasherIsBusy`、`TestLoginLogsNoSecret`。
- `refresh_test.go`（9 个）：`TestRefreshRotates`、`TestRefreshEveryGenerationInTurn`、`TestRefreshDetectsReuse`、`TestRefreshRejectsWithoutRevoking`（8 个）、`TestRefreshAfterTheSigningKeyChanged`、`TestRefreshJudgesAgainWhenAConcurrentRefreshWins`、`TestRefreshJudgesAgainWhenARevocationWins`、`TestRefreshGivesUpWhenTheRotationMissesTwice`、`TestRefreshLogsNoSecret`。
- `logout_test.go`（2 个）：`TestLogoutEndsTheSessionOfTheCurrentGeneration`、`TestLogoutLeavesTheSessionForAnyOtherToken`（9 个）。
- `fakes_test.go`：`fakeLogins`（记下查过的地址、锁的次数、事务之外的写入）；`fakeSessions` 按真实的 `WHERE` 条件判断命中（`at`），`beforeWrite`、`afterWrite` 钩子模拟并发的写入；`fakeHasher.Verify`（`hashed:` 前缀是当前参数，`old:` 前缀要求重新哈希，`onVerify` 钩子）；`fakeMAC{key}`。
- argon2：`TestVerify`、`TestVerifyAsksForARehashWhenTheParametersChanged`、`TestVerifyRejectsAnotherFormat`（16 个）、`TestVerifyIsBusyWhenNoSlotFreesUp`。signing：`TestRefreshTokenMACVerify`。
- `register_test.go` 只改构造（`issuance(tokens, mac)`），7 个测试照旧通过。

- [ ] **Step 1: 端口和 `Issuance`**

`server/internal/modules/identity/app/ports.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/modules/identity/app/ports.go
+++ b/server/internal/modules/identity/app/ports.go
@@ -43,6 +43,40 @@
 	GetUser(ctx context.Context, id uuid.UUID) (domain.User, error)
 }
 
+// LoginAccount is what login reads of an account before its transaction:
+// the hash is the snapshot it verifies the password against (M2 design 3.5).
+type LoginAccount struct {
+	ID           uuid.UUID
+	PasswordHash string
+}
+
+// LoginAccountReader finds the account of an address.
+type LoginAccountReader interface {
+	// FindLoginAccount returns ErrNotFound when no account has email, a
+	// normalized address.
+	FindLoginAccount(ctx context.Context, email string) (LoginAccount, error)
+}
+
+// LockedAccount is an account's row under the credential lock.
+type LockedAccount struct {
+	PasswordHash string
+	Active       bool
+}
+
+// CredentialLocker takes the account row lock that every transaction
+// issuing or changing a credential takes first (M2 design 3.5).
+type CredentialLocker interface {
+	// LockForCredentials locks account id's row until the transaction ends
+	// (SELECT … FOR NO KEY UPDATE) and returns it; ErrNotFound when there
+	// is none. Call it inside a transaction.
+	LockForCredentials(ctx context.Context, id uuid.UUID) (LockedAccount, error)
+}
+
+// PasswordHashWriter stores a new hash of an account's password.
+type PasswordHashWriter interface {
+	UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string, now time.Time) error
+}
+
 // ProfileCreator inserts the preferences of a new account.
 type ProfileCreator interface {
 	// CreateDefaultProfile inserts profile id of userID with every default.
@@ -79,10 +113,50 @@
 	SessionCredential(ctx context.Context, id uuid.UUID) (SessionCredential, error)
 }
 
-// PasswordHasher hashes passwords with argon2id. Hash returns a *shared.Error
-// of 503 server_busy when no slot frees up within the wait limit (M2 design 3.8).
+// RefreshSession is what a refresh reads of the session its token names.
+type RefreshSession struct {
+	UserID uuid.UUID
+	State  domain.SessionState
+}
+
+// SessionGeneration is a session as a refresh token presents it: rotation
+// and logout change the session only while it is still at this generation
+// with this hash, unrevoked and unexpired at Now (M2 design 3.5).
+type SessionGeneration struct {
+	ID         uuid.UUID
+	Generation uint32
+	TokenHash  []byte
+	Now        time.Time
+}
+
+// SessionRotator is what a refresh reads and writes (M2 design 3.5).
+type SessionRotator interface {
+	// SessionForRefresh returns ErrNotFound when there is no such session.
+	SessionForRefresh(ctx context.Context, id uuid.UUID) (RefreshSession, error)
+	// RotateSession moves the session from g to the next generation with
+	// newHash; false when the session is no longer at g.
+	RotateSession(ctx context.Context, g SessionGeneration, newHash []byte) (bool, error)
+	// RevokeForReuse revokes session id with reason reuse_detected, unless
+	// it is revoked already.
+	RevokeForReuse(ctx context.Context, id uuid.UUID, now time.Time) error
+}
+
+// SessionEnder ends sessions at logout.
+type SessionEnder interface {
+	// EndSession revokes the session with reason logout while it is at g;
+	// false when it is not.
+	EndSession(ctx context.Context, g SessionGeneration) (bool, error)
+}
+
+// PasswordHasher hashes and verifies passwords with argon2id. Both return a
+// *shared.Error of 503 server_busy when no slot frees up within the wait
+// limit (M2 design 3.8).
 type PasswordHasher interface {
 	Hash(ctx context.Context, password string) (string, error)
+	// Verify reports whether password matches hash, and whether hash has
+	// other parameters than the current ones: then login hashes the
+	// password again.
+	Verify(ctx context.Context, password, hash string) (ok, rehash bool, err error)
 }
 
 // AccessClaims are the claims of an access token: nothing about permissions
@@ -110,6 +184,8 @@
 // HMAC-SHA256 under a key derived from the signing key.
 type RefreshTokenMAC interface {
 	Tag(message []byte) [16]byte
+	// Verify reports whether tag is message's tag, in constant time.
+	Verify(message []byte, tag [16]byte) bool
 }
 
 // SignupPolicy decides whether registration is open (M2 design 3.9). From
```

`server/internal/modules/identity/app/issuer.go`（新文件）：

```go
package app

import (
	"crypto/rand"
	"net/netip"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// Tokens are what registration, login and refresh return.
type Tokens struct {
	AccessToken      string
	AccessExpiresIn  time.Duration
	RefreshToken     string
	RefreshExpiresAt time.Time // the session's absolute end (M2 design 3.5)
}

// Issuance is how sessions and their tokens are made (M2 design 3.4, 3.5):
// registration, login and refresh share it.
type Issuance struct {
	Tokens     AccessTokens
	MAC        RefreshTokenMAC
	AccessTTL  time.Duration // auth.access_token_ttl
	SessionTTL time.Duration // auth.session_ttl
}

// refreshToken returns generation of session id, with a new secret and
// its tag.
func (i Issuance) refreshToken(sessionID uuid.UUID, generation uint32) domain.RefreshToken {
	t := domain.RefreshToken{SessionID: sessionID, Generation: generation}
	_, _ = rand.Read(t.Secret[:]) // never fails since Go 1.24
	t.Tag = i.MAC.Tag(t.MACMessage())
	return t
}

// tokens returns refresh with a new access token of userID; sessionEnd is
// the session's absolute end.
func (i Issuance) tokens(userID uuid.UUID, refresh domain.RefreshToken, now, sessionEnd time.Time) (Tokens, error) {
	access, err := i.Tokens.Issue(AccessClaims{UserID: userID, SessionID: refresh.SessionID, ExpiresAt: now.Add(i.AccessTTL)})
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: access, AccessExpiresIn: i.AccessTTL, RefreshToken: refresh.String(), RefreshExpiresAt: sessionEnd}, nil
}

// newSession returns a new session of userID at generation 0, which the
// caller inserts, and its tokens: a login from the client with userAgent
// at ip. The session ends SessionTTL from now, for good (M2 design 3.5).
func (i Issuance) newSession(userID uuid.UUID, userAgent string, ip netip.Addr, now time.Time) (NewSession, Tokens, error) {
	refresh := i.refreshToken(uuid.NewV7(), 0)
	s := NewSession{
		ID:        refresh.SessionID,
		UserID:    userID,
		TokenHash: refresh.SecretHash(),
		UserAgent: domain.SanitizeUserAgent(userAgent),
		IP:        ip,
		ExpiresAt: now.Add(i.SessionTTL),
		Now:       now,
	}
	tokens, err := i.tokens(userID, refresh, now, s.ExpiresAt)
	return s, tokens, err
}
```

`server/internal/modules/identity/app/register.go`（完整内容）：

```go
package app

import (
	"context"
	"log/slog"
	"net/netip"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// RegisterDeps are Register's collaborators and settings.
type RegisterDeps struct {
	Policy   SignupPolicy
	Rules    *domain.PasswordRules
	Hasher   PasswordHasher
	Tx       shared.TxManager
	Users    UserCreator
	Profiles ProfileCreator
	Sessions SessionCreator
	Issuance Issuance
	Clock    Clock
	Logger   *slog.Logger
}

// Register creates an account and signs it in: POST /api/v0/auth/register.
type Register struct {
	d        RegisterDeps
	accounts accounts
}

// NewRegister returns the use case.
func NewRegister(d RegisterDeps) *Register {
	return &Register{d: d, accounts: accounts{users: d.Users, profiles: d.Profiles}}
}

// RegisterInput is a registration and where it comes from.
type RegisterInput struct {
	Email     string
	Password  string
	UserAgent string
	IP        netip.Addr
}

// Execute registers in.Email:
//
//  1. closed sign-up answers 403 before any other check (M2 design 3.9);
//  2. the address and the password are validated, all fields at once (422);
//  3. the password is hashed outside the transaction (M2 design 3.5);
//  4. one transaction inserts the account, its profile and its first
//     session; an address in use is 409 identity.email_taken.
//
// The tokens are made before the transaction, so nothing can fail after it
// commits.
func (r *Register) Execute(ctx context.Context, in RegisterInput) (Tokens, error) {
	allowed, err := r.d.Policy.AllowSignup(ctx)
	if err != nil {
		return Tokens{}, err
	}
	if !allowed {
		return Tokens{}, domain.ErrSignupDisabled
	}
	email, err := domain.NewAccount(r.d.Rules, in.Email, in.Password)
	if err != nil {
		return Tokens{}, err
	}
	hash, err := r.d.Hasher.Hash(ctx, in.Password)
	if err != nil {
		return Tokens{}, err
	}

	now := r.d.Clock.Now()
	user := NewUser{ID: uuid.NewV7(), Email: email, PasswordHash: hash, DisplayName: domain.DisplayNameFromEmail(email), Now: now}
	session, tokens, err := r.d.Issuance.newSession(user.ID, in.UserAgent, in.IP, now)
	if err != nil {
		return Tokens{}, err
	}

	err = r.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := r.accounts.create(ctx, user); err != nil {
			return err
		}
		return r.d.Sessions.CreateSession(ctx, session)
	})
	if err != nil {
		return Tokens{}, err
	}
	r.d.Logger.InfoContext(ctx, "account registered", slog.String("user_id", user.ID.String()))
	return tokens, nil
}
```

- [ ] **Step 2: 三个用例**

`server/internal/modules/identity/app/login.go`（新文件）：

```go
package app

import (
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// LoginDeps are Login's collaborators and settings.
type LoginDeps struct {
	Accounts  LoginAccountReader
	Locker    CredentialLocker
	Passwords PasswordHashWriter
	Sessions  SessionCreator
	Hasher    PasswordHasher
	Tx        shared.TxManager
	Issuance  Issuance
	Clock     Clock
	Logger    *slog.Logger
	// DummyHash is a hash of a random password with the current parameters,
	// made at startup: an unknown address is verified against it, so that
	// it takes as long as a known one (M2 design 3.9).
	DummyHash string
}

// Login signs an account in with its password: POST /api/v0/auth/login.
type Login struct {
	d LoginDeps
}

// NewLogin returns the use case.
func NewLogin(d LoginDeps) *Login {
	return &Login{d: d}
}

// LoginInput is a login and where it comes from.
type LoginInput struct {
	Email     string
	Password  string
	UserAgent string
	IP        netip.Addr
}

// Execute signs in.Email in (M2 design 3.5, 3.9):
//
//  1. the account of the normalized address is read, its hash kept as the
//     snapshot; an unknown address is verified against DummyHash and is
//     401 identity.invalid_credentials, like a wrong password;
//  2. the password is verified against the snapshot outside the
//     transaction, and hashed again there when the parameters changed;
//  3. one transaction locks the account row, checks that the hash still
//     equals the snapshot and that the account is active (403
//     identity.account_deactivated, only now that the password is known to
//     be right), writes the new hash if any, and inserts the session.
//
// When the hash changed in between, a concurrent login hashed the password
// again, or the password changed: the password is verified against the new
// hash, and step 3 is done once more. A second change fails with 401.
func (l *Login) Execute(ctx context.Context, in LoginInput) (Tokens, error) {
	account, err := l.find(ctx, domain.NormalizeEmail(in.Email))
	if errors.Is(err, ErrNotFound) {
		if _, _, err := l.d.Hasher.Verify(ctx, in.Password, l.d.DummyHash); err != nil {
			return Tokens{}, err
		}
		return Tokens{}, l.failed(ctx, domain.ErrInvalidCredentials, "invalid_credentials", in.IP, uuid.Nil())
	}
	if err != nil {
		return Tokens{}, err
	}

	snapshot := account.PasswordHash
	for range 2 {
		ok, rehash, err := l.d.Hasher.Verify(ctx, in.Password, snapshot)
		if err != nil {
			return Tokens{}, err
		}
		if !ok {
			break
		}
		newHash := ""
		if rehash {
			if newHash, err = l.d.Hasher.Hash(ctx, in.Password); err != nil {
				return Tokens{}, err
			}
		}
		now := l.d.Clock.Now()
		session, tokens, err := l.d.Issuance.newSession(account.ID, in.UserAgent, in.IP, now)
		if err != nil {
			return Tokens{}, err
		}

		current := snapshot
		err = l.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
			locked, err := l.d.Locker.LockForCredentials(ctx, account.ID)
			if err != nil {
				return err
			}
			if current = locked.PasswordHash; current != snapshot {
				return nil
			}
			if !locked.Active {
				return domain.ErrAccountDeactivated
			}
			if newHash != "" {
				if err := l.d.Passwords.UpdatePasswordHash(ctx, account.ID, newHash, now); err != nil {
					return err
				}
			}
			return l.d.Sessions.CreateSession(ctx, session)
		})
		switch {
		case errors.Is(err, domain.ErrAccountDeactivated):
			return Tokens{}, l.failed(ctx, err, "deactivated", in.IP, account.ID)
		case err != nil:
			return Tokens{}, err
		case current == snapshot:
			l.d.Logger.InfoContext(ctx, "signed in", slog.String("user_id", account.ID.String()),
				slog.String("session_id", session.ID.String()), slog.String("ip", in.IP.String()))
			return tokens, nil
		}
		snapshot = current
	}
	return Tokens{}, l.failed(ctx, domain.ErrInvalidCredentials, "invalid_credentials", in.IP, account.ID)
}

// find reads the account of email. An address that cannot be valid is not
// looked up, so the database never sees what it could not store (a NUL, a
// byte that is not UTF-8).
func (l *Login) find(ctx context.Context, email string) (LoginAccount, error) {
	if !domain.ValidEmail(email) {
		return LoginAccount{}, ErrNotFound
	}
	return l.d.Accounts.FindLoginAccount(ctx, email)
}

// failed logs a failed login (M2 design 8.4) and returns err: the account
// only when there is one, never the address.
func (l *Login) failed(ctx context.Context, err error, reason string, ip netip.Addr, userID uuid.UUID) error {
	attrs := []slog.Attr{slog.String("reason", reason), slog.String("ip", ip.String())}
	if userID != uuid.Nil() {
		attrs = append(attrs, slog.String("user_id", userID.String()))
	}
	l.d.Logger.LogAttrs(ctx, slog.LevelInfo, "sign-in failed", attrs...)
	return err
}
```

`server/internal/modules/identity/app/refresh.go`（新文件）：

```go
package app

import (
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// RefreshDeps are Refresh's collaborators and settings.
type RefreshDeps struct {
	Sessions SessionRotator
	Tx       shared.TxManager
	Issuance Issuance
	Clock    Clock
	Logger   *slog.Logger
}

// Refresh exchanges a refresh token for the next pair: POST
// /api/v0/auth/refresh (M2 design 3.5).
type Refresh struct {
	d RefreshDeps
}

// NewRefresh returns the use case.
func NewRefresh(d RefreshDeps) *Refresh {
	return &Refresh{d: d}
}

// errRotatedTwice means the conditional rotation missed after a re-read
// that judged it possible: the session cannot move back to a generation,
// so this is a fault, not an answer.
var errRotatedTwice = errors.New("refresh: the session changed under the rotation twice")

// Execute rotates the session of token and returns the next generation's
// tokens. Every other outcome is 401 identity.refresh_token_invalid; only an
// older generation that the session did issue revokes the session first
// (reuse), with a WARN naming the account, the session and ip.
//
// Parsing looks nothing up. One transaction reads the session by its id,
// judges the token (domain.JudgeRefresh), and rotates with a conditional
// UPDATE. When the UPDATE misses, a concurrent refresh or revocation got
// there first: the row is read and judged again in the same transaction.
// The revocation of a reuse commits before the 401 goes out.
func (r *Refresh) Execute(ctx context.Context, token string, ip netip.Addr) (Tokens, error) {
	presented, ok := domain.ParseRefreshToken(token)
	if !ok {
		return Tokens{}, domain.ErrRefreshTokenInvalid
	}
	now := r.d.Clock.Now()
	tagValid := r.d.Issuance.MAC.Verify(presented.MACMessage(), presented.Tag)
	next := r.d.Issuance.refreshToken(presented.SessionID, presented.Generation+1)
	current := SessionGeneration{ID: presented.SessionID, Generation: presented.Generation, TokenHash: presented.SecretHash(), Now: now}

	var verdict domain.Verdict
	var userID uuid.UUID
	var tokens Tokens
	err := r.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		for range 2 {
			s, err := r.d.Sessions.SessionForRefresh(ctx, presented.SessionID)
			if errors.Is(err, ErrNotFound) {
				verdict = domain.Reject
				return nil
			}
			if err != nil {
				return err
			}
			userID = s.UserID
			switch verdict = domain.JudgeRefresh(s.State, presented, tagValid, now); verdict {
			case domain.Reject:
				return nil
			case domain.Reuse:
				return r.d.Sessions.RevokeForReuse(ctx, presented.SessionID, now)
			}
			if tokens, err = r.d.Issuance.tokens(s.UserID, next, now, s.State.ExpiresAt); err != nil {
				return err
			}
			rotated, err := r.d.Sessions.RotateSession(ctx, current, next.SecretHash())
			if err != nil || rotated {
				return err
			}
		}
		return errRotatedTwice
	})
	switch {
	case err != nil:
		return Tokens{}, err
	case verdict == domain.Rotate:
		return tokens, nil
	case verdict == domain.Reuse:
		r.d.Logger.WarnContext(ctx, "refresh token reused: session revoked",
			slog.String("user_id", userID.String()), slog.String("session_id", presented.SessionID.String()), slog.String("ip", ip.String()))
	}
	return Tokens{}, domain.ErrRefreshTokenInvalid
}
```

`server/internal/modules/identity/app/logout.go`（新文件）：

```go
package app

import (
	"context"
	"log/slog"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// Logout ends the session of a refresh token: POST /api/v0/auth/logout.
type Logout struct {
	sessions SessionEnder
	clock    Clock
	logger   *slog.Logger
}

// NewLogout returns the use case.
func NewLogout(sessions SessionEnder, clock Clock, logger *slog.Logger) *Logout {
	return &Logout{sessions: sessions, clock: clock, logger: logger}
}

// Execute revokes the session of token with reason logout, only while token
// is its current generation and the session is live: the first row of
// refresh's table (M2 design 3.5). Any other token changes nothing and is no
// error either, so the answer never tells what the token was; reuse is
// judged only by refresh. One statement, no transaction (M2 design 6.4).
func (l *Logout) Execute(ctx context.Context, token string) error {
	presented, ok := domain.ParseRefreshToken(token)
	if !ok {
		return nil
	}
	ended, err := l.sessions.EndSession(ctx, SessionGeneration{
		ID: presented.SessionID, Generation: presented.Generation, TokenHash: presented.SecretHash(), Now: l.clock.Now(),
	})
	if err != nil || !ended {
		return err
	}
	l.logger.InfoContext(ctx, "signed out", slog.String("session_id", presented.SessionID.String()))
	return nil
}
```

- [ ] **Step 3: 用例的测试**

`server/internal/modules/identity/app/fakes_test.go`（完整内容）：

```go
package app_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// fakeTx runs fn in a context marked as inside the transaction; fakeStore
// records whether each write happened there.
type fakeTx struct{ calls int }

type inTxKey struct{}

func (f *fakeTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	f.calls++
	return fn(context.WithValue(ctx, inTxKey{}, true))
}

func inTx(ctx context.Context) bool { return ctx.Value(inTxKey{}) == true }

// fakeStore is every repository port, in memory.
type fakeStore struct {
	users       []app.NewUser
	profiles    []uuid.UUID // user ids
	sessions    []app.NewSession
	outsideTx   []string // writes made outside a transaction
	createErr   error    // CreateUser's error
	getUser     domain.User
	getUserErr  error
	getUserIDs  []uuid.UUID // accounts looked up
	credential  app.SessionCredential
	credErr     error
	credentials []uuid.UUID // sessions looked up
}

func (s *fakeStore) CreateUser(ctx context.Context, u app.NewUser) error {
	if !inTx(ctx) {
		s.outsideTx = append(s.outsideTx, "user")
	}
	if s.createErr != nil {
		return s.createErr
	}
	s.users = append(s.users, u)
	return nil
}

func (s *fakeStore) CreateDefaultProfile(ctx context.Context, _, userID uuid.UUID, _ time.Time) error {
	if !inTx(ctx) {
		s.outsideTx = append(s.outsideTx, "profile")
	}
	s.profiles = append(s.profiles, userID)
	return nil
}

func (s *fakeStore) CreateSession(ctx context.Context, n app.NewSession) error {
	if !inTx(ctx) {
		s.outsideTx = append(s.outsideTx, "session")
	}
	s.sessions = append(s.sessions, n)
	return nil
}

func (s *fakeStore) GetUser(_ context.Context, id uuid.UUID) (domain.User, error) {
	s.getUserIDs = append(s.getUserIDs, id)
	return s.getUser, s.getUserErr
}

func (s *fakeStore) SessionCredential(_ context.Context, id uuid.UUID) (app.SessionCredential, error) {
	s.credentials = append(s.credentials, id)
	return s.credential, s.credErr
}

// fakeLogins is login's ports over one account, in memory.
type fakeLogins struct {
	account     app.LoginAccount // found by email
	email       string           // the account's normalized address
	active      bool
	hash        string   // the row's hash, as LockForCredentials reads it
	lookedUp    []string // addresses FindLoginAccount was given
	locks       int      // LockForCredentials calls
	hashUpdates []string // hashes UpdatePasswordHash wrote
	sessions    []app.NewSession
	outsideTx   []string
}

func (f *fakeLogins) FindLoginAccount(_ context.Context, email string) (app.LoginAccount, error) {
	f.lookedUp = append(f.lookedUp, email)
	if email != f.email {
		return app.LoginAccount{}, app.ErrNotFound
	}
	return f.account, nil
}

func (f *fakeLogins) LockForCredentials(ctx context.Context, id uuid.UUID) (app.LockedAccount, error) {
	f.locks++
	if !inTx(ctx) {
		f.outsideTx = append(f.outsideTx, "lock")
	}
	if id != f.account.ID {
		return app.LockedAccount{}, app.ErrNotFound
	}
	return app.LockedAccount{PasswordHash: f.hash, Active: f.active}, nil
}

func (f *fakeLogins) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string, _ time.Time) error {
	if !inTx(ctx) || id != f.account.ID {
		f.outsideTx = append(f.outsideTx, "password of "+id.String())
	}
	f.hashUpdates = append(f.hashUpdates, hash)
	f.hash = hash
	return nil
}

func (f *fakeLogins) CreateSession(ctx context.Context, n app.NewSession) error {
	if !inTx(ctx) {
		f.outsideTx = append(f.outsideTx, "session")
	}
	f.sessions = append(f.sessions, n)
	return nil
}

// fakeSession is a session row as refresh and logout see it.
type fakeSession struct {
	app.RefreshSession
	reason string // revoke_reason
}

// fakeSessions is refresh's and logout's ports, in memory. RotateSession and
// EndSession apply their conditions the way the SQL does.
type fakeSessions struct {
	rows        map[uuid.UUID]*fakeSession
	beforeWrite func() // runs before each conditional write: a transaction that commits first
	afterWrite  func() // runs after each conditional write
	reads       int
	outsideTx   []string
}

func (f *fakeSessions) SessionForRefresh(_ context.Context, id uuid.UUID) (app.RefreshSession, error) {
	f.reads++
	s, ok := f.rows[id]
	if !ok {
		return app.RefreshSession{}, app.ErrNotFound
	}
	return app.RefreshSession{UserID: s.UserID, State: s.State}, nil
}

// at reports whether the session is still at g: the WHERE of rotation and
// logout.
func (f *fakeSessions) at(g app.SessionGeneration) (*fakeSession, bool) {
	if f.beforeWrite != nil {
		f.beforeWrite()
	}
	if f.afterWrite != nil {
		defer f.afterWrite()
	}
	s, ok := f.rows[g.ID]
	return s, ok && s.State.Generation == g.Generation && bytes.Equal(s.State.TokenHash, g.TokenHash) &&
		!s.State.Revoked && g.Now.Before(s.State.ExpiresAt)
}

func (f *fakeSessions) RotateSession(ctx context.Context, g app.SessionGeneration, newHash []byte) (bool, error) {
	if !inTx(ctx) {
		f.outsideTx = append(f.outsideTx, "rotate")
	}
	s, ok := f.at(g)
	if ok {
		s.State.Generation++
		s.State.TokenHash = newHash
	}
	return ok, nil
}

func (f *fakeSessions) RevokeForReuse(ctx context.Context, id uuid.UUID, _ time.Time) error {
	if !inTx(ctx) {
		f.outsideTx = append(f.outsideTx, "revoke")
	}
	if s, ok := f.rows[id]; ok && !s.State.Revoked {
		s.State.Revoked, s.reason = true, "reuse_detected"
	}
	return nil
}

func (f *fakeSessions) EndSession(_ context.Context, g app.SessionGeneration) (bool, error) {
	s, ok := f.at(g)
	if ok {
		s.State.Revoked, s.reason = true, "logout"
	}
	return ok, nil
}

// fakeHasher "hashes" by prefixing "hashed:"; a hash prefixed "old:" has
// other parameters, so Verify asks for a rehash. It counts its calls.
type fakeHasher struct {
	calls     int // Hash calls
	err       error
	verified  []string // the hashes Verify was given
	verifyErr error
	onVerify  func() // runs inside every Verify: a transaction that commits meanwhile
}

func (h *fakeHasher) Hash(_ context.Context, password string) (string, error) {
	h.calls++
	if h.err != nil {
		return "", h.err
	}
	return "hashed:" + password, nil
}

func (h *fakeHasher) Verify(_ context.Context, password, hash string) (bool, bool, error) {
	h.verified = append(h.verified, hash)
	if h.onVerify != nil {
		h.onVerify()
	}
	if h.verifyErr != nil {
		return false, false, h.verifyErr
	}
	switch hash {
	case "hashed:" + password:
		return true, false, nil
	case "old:" + password:
		return true, true, nil
	}
	return false, false, nil
}

// fakeTokens issues "access:<sid>" and verifies what it issued at the instant
// it is given: like a JWT's exp, a token is expired from its ExpiresAt on. It
// records those instants.
type fakeTokens struct {
	issued     []app.AccessClaims
	claims     map[string]app.AccessClaims
	verifiedAt []time.Time
}

func newFakeTokens() *fakeTokens {
	return &fakeTokens{claims: map[string]app.AccessClaims{}}
}

func (f *fakeTokens) Issue(c app.AccessClaims) (string, error) {
	f.issued = append(f.issued, c)
	token := "access:" + c.SessionID.String()
	f.claims[token] = c
	return token, nil
}

var errBadSignature = errors.New("signature is invalid")

func (f *fakeTokens) Verify(token string, now time.Time) (app.AccessClaims, error) {
	f.verifiedAt = append(f.verifiedAt, now)
	c, ok := f.claims[token]
	if !ok {
		return app.AccessClaims{}, errBadSignature
	}
	if !now.Before(c.ExpiresAt) {
		return app.AccessClaims{}, app.ErrAccessTokenExpired
	}
	return c, nil
}

// fakeMAC tags with the first 16 bytes of SHA-256 over its key and the
// message: deterministic, and different for every message and key. Another
// key stands for a changed signing key.
type fakeMAC struct{ key string }

func (m fakeMAC) Tag(message []byte) [16]byte {
	sum := sha256.Sum256(append([]byte(m.key), message...))
	return [16]byte(sum[:16])
}

func (m fakeMAC) Verify(message []byte, tag [16]byte) bool { return m.Tag(message) == tag }

type fixedPolicy struct {
	allow bool
	err   error
}

func (p fixedPolicy) AllowSignup(context.Context) (bool, error) { return p.allow, p.err }
```

`server/internal/modules/identity/app/register_test.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/modules/identity/app/register_test.go
+++ b/server/internal/modules/identity/app/register_test.go
@@ -33,23 +33,25 @@
 func newRegister(policy app.SignupPolicy) *registerFixture {
 	f := &registerFixture{store: &fakeStore{}, tx: &fakeTx{}, hasher: &fakeHasher{}, tokens: newFakeTokens(), logs: &bytes.Buffer{}}
 	f.uc = app.NewRegister(app.RegisterDeps{
-		Policy:     policy,
-		Rules:      domain.NewPasswordRules(),
-		Hasher:     f.hasher,
-		Tx:         f.tx,
-		Users:      f.store,
-		Profiles:   f.store,
-		Sessions:   f.store,
-		Tokens:     f.tokens,
-		MAC:        fakeMAC{},
-		Clock:      clocktest.At(now),
-		Logger:     slog.New(slog.NewJSONHandler(f.logs, nil)),
-		AccessTTL:  15 * time.Minute,
-		SessionTTL: 720 * time.Hour,
+		Policy:   policy,
+		Rules:    domain.NewPasswordRules(),
+		Hasher:   f.hasher,
+		Tx:       f.tx,
+		Users:    f.store,
+		Profiles: f.store,
+		Sessions: f.store,
+		Issuance: issuance(f.tokens, fakeMAC{}),
+		Clock:    clocktest.At(now),
+		Logger:   slog.New(slog.NewJSONHandler(f.logs, nil)),
 	})
 	return f
 }
 
+// issuance has auth's default TTLs: 15 minutes, 30 days.
+func issuance(tokens *fakeTokens, mac fakeMAC) app.Issuance {
+	return app.Issuance{Tokens: tokens, MAC: mac, AccessTTL: 15 * time.Minute, SessionTTL: 720 * time.Hour}
+}
+
 // isV7 reports whether id is a version 7 UUID (RFC 9562 5.7): ids are
 // generated by the application, time-ordered.
 func isV7(id uuid.UUID) bool { return id[6]>>4 == 7 }
```

`server/internal/modules/identity/app/login_test.go`（新文件）：

```go
package app_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// dummyHash is the fixture's DummyHash: it verifies no password a test sends.
const dummyHash = "hashed:the dummy password"

type loginFixture struct {
	logins *fakeLogins
	hasher *fakeHasher
	tx     *fakeTx
	tokens *fakeTokens
	logs   *bytes.Buffer
	uc     *app.Login
}

// newLogin has one account, alice@corp.com, whose row holds hash.
func newLogin(hash string, active bool) *loginFixture {
	f := &loginFixture{
		logins: &fakeLogins{account: app.LoginAccount{ID: userID, PasswordHash: hash}, email: "alice@corp.com", active: active, hash: hash},
		hasher: &fakeHasher{},
		tx:     &fakeTx{},
		tokens: newFakeTokens(),
		logs:   &bytes.Buffer{},
	}
	f.uc = app.NewLogin(app.LoginDeps{
		Accounts:  f.logins,
		Locker:    f.logins,
		Passwords: f.logins,
		Sessions:  f.logins,
		Hasher:    f.hasher,
		Tx:        f.tx,
		Issuance:  issuance(f.tokens, fakeMAC{}),
		Clock:     clocktest.At(now),
		Logger:    slog.New(slog.NewJSONHandler(f.logs, nil)),
		DummyHash: dummyHash,
	})
	return f
}

var loginInput = app.LoginInput{
	Email:     "  Alice@Corp.com ",
	Password:  "Tr0ub4dor&3",
	UserAgent: "agent\x00/1",
	IP:        netip.MustParseAddr("203.0.113.7"),
}

func TestLoginSignsIn(t *testing.T) {
	f := newLogin("hashed:Tr0ub4dor&3", true)

	tokens, err := f.uc.Execute(context.Background(), loginInput)
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(f.logins.lookedUp, []string{"alice@corp.com"}) || !slices.Equal(f.hasher.verified, []string{"hashed:Tr0ub4dor&3"}) || f.hasher.calls != 0 {
		t.Errorf("looked up %q, verified against %q, hashed %d times; want the normalized address, the row's hash, no hash",
			f.logins.lookedUp, f.hasher.verified, f.hasher.calls)
	}
	if f.tx.calls != 1 || f.logins.locks != 1 || len(f.logins.outsideTx) != 0 || len(f.logins.hashUpdates) != 0 {
		t.Errorf("transactions %d, locks %d, outside one %q, hash updates %q; want the lock and the insert in one", f.tx.calls, f.logins.locks, f.logins.outsideTx, f.logins.hashUpdates)
	}
	if len(f.logins.sessions) != 1 {
		t.Fatalf("sessions = %d, want 1", len(f.logins.sessions))
	}
	s := f.logins.sessions[0]
	if s.UserID != userID || s.UserAgent != "agent/1" || s.IP != loginInput.IP || !s.ExpiresAt.Equal(now.Add(720*time.Hour)) || !s.Now.Equal(now) || !isV7(s.ID) {
		t.Errorf("session = %+v", s)
	}
	refresh, ok := domain.ParseRefreshToken(tokens.RefreshToken)
	if !ok || refresh.SessionID != s.ID || refresh.Generation != 0 || !bytes.Equal(refresh.SecretHash(), s.TokenHash) || !(fakeMAC{}).Verify(refresh.MACMessage(), refresh.Tag) {
		t.Errorf("refresh token %+v of session %+v; want generation 0, its secret's hash stored, tagged", refresh, s)
	}
	want := app.AccessClaims{UserID: userID, SessionID: s.ID, ExpiresAt: now.Add(15 * time.Minute)}
	if !slices.Equal(f.tokens.issued, []app.AccessClaims{want}) || tokens.AccessExpiresIn != 15*time.Minute || !tokens.RefreshExpiresAt.Equal(s.ExpiresAt) {
		t.Errorf("tokens = %+v, claims %+v; want claims %+v", tokens, f.tokens.issued, want)
	}
}

// An unknown address costs what a wrong password costs: one verification,
// with the current parameters, of the dummy hash (M2 design 3.9). An
// address that cannot be valid is not even looked up.
func TestLoginFailsAlikeForAnUnknownAddressAndAWrongPassword(t *testing.T) {
	tests := []struct {
		name, email, password string
		lookedUp              []string
		verified              string
	}{
		{"wrong password", "alice@corp.com", "Tr0ub4dor&4", []string{"alice@corp.com"}, "hashed:Tr0ub4dor&3"},
		{"unknown address", "bob@corp.com", "Tr0ub4dor&3", []string{"bob@corp.com"}, dummyHash},
		{"empty address", "  ", "Tr0ub4dor&3", nil, dummyHash},
		{"a NUL in the address", "alice\x00@corp.com", "Tr0ub4dor&3", nil, dummyHash},
		{"an address that is not UTF-8", "alice\xff@corp.com", "Tr0ub4dor&3", nil, dummyHash},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newLogin("hashed:Tr0ub4dor&3", true)

			_, err := f.uc.Execute(context.Background(), app.LoginInput{Email: tt.email, Password: tt.password})

			if !errors.Is(err, domain.ErrInvalidCredentials) {
				t.Errorf("Execute() = %v, want identity.invalid_credentials", err)
			}
			if !slices.Equal(f.logins.lookedUp, tt.lookedUp) || !slices.Equal(f.hasher.verified, []string{tt.verified}) || f.hasher.calls != 0 {
				t.Errorf("looked up %q, verified against %q, hashed %d times; want %q, [%q], 0", f.logins.lookedUp, f.hasher.verified, f.hasher.calls, tt.lookedUp, tt.verified)
			}
			if f.tx.calls != 0 || len(f.logins.sessions) != 0 {
				t.Errorf("transactions %d, sessions %d; want none", f.tx.calls, len(f.logins.sessions))
			}
		})
	}
}

// A deactivated account is told only to whoever knows its password, and
// only inside the transaction (M2 design 3.9).
func TestLoginRevealsDeactivationOnlyWithTheRightPassword(t *testing.T) {
	f := newLogin("hashed:Tr0ub4dor&3", false)

	_, wrong := f.uc.Execute(context.Background(), app.LoginInput{Email: "alice@corp.com", Password: "Tr0ub4dor&4"})
	locksAfterWrong := f.logins.locks
	_, right := f.uc.Execute(context.Background(), loginInput)

	if !errors.Is(wrong, domain.ErrInvalidCredentials) || locksAfterWrong != 0 {
		t.Errorf("wrong password: %v after %d locks; want identity.invalid_credentials before any lock", wrong, locksAfterWrong)
	}
	if !errors.Is(right, domain.ErrAccountDeactivated) || f.logins.locks != 1 || len(f.logins.sessions) != 0 {
		t.Errorf("right password: %v, %d locks, %d sessions; want identity.account_deactivated under the lock, no session", right, f.logins.locks, len(f.logins.sessions))
	}
}

// Other parameters: the password is hashed again outside the transaction
// and the new hash written under the lock (M2 design 3.8).
func TestLoginRehashesWhenTheParametersChanged(t *testing.T) {
	f := newLogin("old:Tr0ub4dor&3", true)

	if _, err := f.uc.Execute(context.Background(), loginInput); err != nil {
		t.Fatal(err)
	}

	if f.hasher.calls != 1 || !slices.Equal(f.logins.hashUpdates, []string{"hashed:Tr0ub4dor&3"}) || len(f.logins.outsideTx) != 0 || len(f.logins.sessions) != 1 {
		t.Errorf("hashed %d times, wrote %q, outside the transaction %q, sessions %d; want one new hash written in the transaction",
			f.hasher.calls, f.logins.hashUpdates, f.logins.outsideTx, len(f.logins.sessions))
	}
}

// A concurrent login rehashed the same password between the snapshot and
// the lock: the new hash is verified once more, and the login goes through
// without writing a hash of its own (M2 design 3.5, interleaving 5).
func TestLoginVerifiesAgainWhenAConcurrentLoginRehashed(t *testing.T) {
	f := newLogin("old:Tr0ub4dor&3", true)
	f.hasher.onVerify = func() { f.logins.hash = "hashed:Tr0ub4dor&3" }

	_, err := f.uc.Execute(context.Background(), loginInput)

	if err != nil || len(f.logins.sessions) != 1 {
		t.Fatalf("Execute() = %v with %d sessions, want a session", err, len(f.logins.sessions))
	}
	if !slices.Equal(f.hasher.verified, []string{"old:Tr0ub4dor&3", "hashed:Tr0ub4dor&3"}) || f.tx.calls != 2 || len(f.logins.hashUpdates) != 0 {
		t.Errorf("verified against %q in %d transactions, wrote %q; want the snapshot, then the new hash, and no write",
			f.hasher.verified, f.tx.calls, f.logins.hashUpdates)
	}
}

func TestLoginFailsWhenThePasswordChangedMeanwhile(t *testing.T) {
	f := newLogin("hashed:Tr0ub4dor&3", true)
	f.hasher.onVerify = func() { f.logins.hash = "hashed:N3w-password" }

	_, err := f.uc.Execute(context.Background(), loginInput)

	if !errors.Is(err, domain.ErrInvalidCredentials) || len(f.logins.sessions) != 0 {
		t.Errorf("Execute() = %v with %d sessions; want identity.invalid_credentials and none", err, len(f.logins.sessions))
	}
}

// The hash changes again under the second try: the login gives up.
func TestLoginFailsWhenTheHashChangesTwice(t *testing.T) {
	f := newLogin("hashed:Tr0ub4dor&3", true)
	next := []string{"old:Tr0ub4dor&3", "hashed:Tr0ub4dor&3"}
	f.hasher.onVerify = func() { f.logins.hash, next = next[0], next[1:] }

	_, err := f.uc.Execute(context.Background(), loginInput)

	if !errors.Is(err, domain.ErrInvalidCredentials) || len(f.logins.sessions) != 0 || f.tx.calls != 2 {
		t.Errorf("Execute() = %v with %d sessions in %d transactions; want identity.invalid_credentials after two", err, len(f.logins.sessions), f.tx.calls)
	}
}

func TestLoginWhenTheHasherIsBusy(t *testing.T) {
	for _, email := range []string{"alice@corp.com", "bob@corp.com"} {
		f := newLogin("hashed:Tr0ub4dor&3", true)
		f.hasher.verifyErr = shared.ServerBusy(time.Second)

		_, err := f.uc.Execute(context.Background(), app.LoginInput{Email: email, Password: "Tr0ub4dor&3"})

		var se *shared.Error
		if !errors.As(err, &se) || se.Code != shared.CodeServerBusy || f.tx.calls != 0 {
			t.Errorf("%s: Execute() = %v after %d transactions; want server_busy before any", email, err, f.tx.calls)
		}
	}
}

// The log names the account and the session, never the address, the
// password or a token (M2 design 8.4).
func TestLoginLogsNoSecret(t *testing.T) {
	f := newLogin("hashed:Tr0ub4dor&3", true)
	tokens, err := f.uc.Execute(context.Background(), loginInput)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.uc.Execute(context.Background(), app.LoginInput{Email: "alice@corp.com", Password: "Wr0ng-password", IP: loginInput.IP})
	_, _ = f.uc.Execute(context.Background(), app.LoginInput{Email: "bob@corp.com", Password: "Tr0ub4dor&3", IP: loginInput.IP})

	logs := f.logs.String()
	session := f.logins.sessions[0]
	for _, want := range []string{
		`"msg":"signed in","user_id":"` + userID.String() + `","session_id":"` + session.ID.String() + `","ip":"203.0.113.7"`,
		`"msg":"sign-in failed","reason":"invalid_credentials","ip":"203.0.113.7","user_id":"` + userID.String() + `"`,
		`"msg":"sign-in failed","reason":"invalid_credentials","ip":"203.0.113.7"}`,
	} {
		if !strings.Contains(logs, want) {
			t.Errorf("logs lack %s:\n%s", want, logs)
		}
	}
	for _, secret := range []string{"Tr0ub4dor", "Wr0ng", "alice", "bob", tokens.AccessToken, tokens.RefreshToken, string(session.TokenHash)} {
		if strings.Contains(logs, secret) {
			t.Errorf("logs contain %q:\n%s", secret, logs)
		}
	}
}
```

`server/internal/modules/identity/app/refresh_test.go`（新文件）：

```go
package app_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/netip"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var (
	clientIP   = netip.MustParseAddr("203.0.113.7")
	sessionEnd = now.Add(time.Hour)
	signedWith = fakeMAC{key: "the signing key"}
)

// issued is generation g of sessionID with secret byte s, tagged under mac:
// a token the server issued.
func issued(mac fakeMAC, g uint32, s byte) domain.RefreshToken {
	t := domain.RefreshToken{SessionID: sessionID, Generation: g, Secret: [32]byte{s}}
	t.Tag = mac.Tag(t.MACMessage())
	return t
}

// forged is generation g of sessionID with a random secret and tag.
func forged(g uint32) domain.RefreshToken {
	t := domain.RefreshToken{SessionID: sessionID, Generation: g}
	_, _ = rand.Read(t.Secret[:])
	_, _ = rand.Read(t.Tag[:])
	return t
}

type refreshFixture struct {
	sessions *fakeSessions
	row      *fakeSession
	tx       *fakeTx
	tokens   *fakeTokens
	logs     *bytes.Buffer
	uc       *app.Refresh
}

// newRefresh has one session of userID at generation 3, secret 3, until
// sessionEnd; the use case tags under mac.
func newRefresh(mac fakeMAC) *refreshFixture {
	row := &fakeSession{RefreshSession: app.RefreshSession{
		UserID: userID,
		State:  domain.SessionState{Generation: 3, TokenHash: issued(signedWith, 3, 3).SecretHash(), ExpiresAt: sessionEnd},
	}}
	f := &refreshFixture{
		sessions: &fakeSessions{rows: map[uuid.UUID]*fakeSession{sessionID: row}},
		row:      row,
		tx:       &fakeTx{},
		tokens:   newFakeTokens(),
		logs:     &bytes.Buffer{},
	}
	f.uc = app.NewRefresh(app.RefreshDeps{
		Sessions: f.sessions,
		Tx:       f.tx,
		Issuance: issuance(f.tokens, mac),
		Clock:    clocktest.At(now),
		Logger:   slog.New(slog.NewJSONHandler(f.logs, nil)),
	})
	return f
}

func TestRefreshRotates(t *testing.T) {
	f := newRefresh(signedWith)

	tokens, err := f.uc.Execute(context.Background(), issued(signedWith, 3, 3).String(), clientIP)
	if err != nil {
		t.Fatal(err)
	}

	next, ok := domain.ParseRefreshToken(tokens.RefreshToken)
	if !ok || next.SessionID != sessionID || next.Generation != 4 || !signedWith.Verify(next.MACMessage(), next.Tag) {
		t.Fatalf("refresh token = %+v, %v; want generation 4 of the session, tagged", next, ok)
	}
	if f.row.State.Generation != 4 || !bytes.Equal(f.row.State.TokenHash, next.SecretHash()) || f.row.State.Revoked || !f.row.State.ExpiresAt.Equal(sessionEnd) {
		t.Errorf("session = %+v; want generation 4 with the new secret's hash, its end unchanged", f.row.State)
	}
	want := app.AccessClaims{UserID: userID, SessionID: sessionID, ExpiresAt: now.Add(15 * time.Minute)}
	if !slices.Equal(f.tokens.issued, []app.AccessClaims{want}) || !tokens.RefreshExpiresAt.Equal(sessionEnd) || tokens.AccessExpiresIn != 15*time.Minute {
		t.Errorf("tokens = %+v, claims %+v; want claims %+v and the session's end", tokens, f.tokens.issued, want)
	}
	if f.tx.calls != 1 || len(f.sessions.outsideTx) != 0 {
		t.Errorf("transactions %d, writes outside one %q; want one", f.tx.calls, f.sessions.outsideTx)
	}
}

// Each refresh uses the token the last one returned; the session's end
// never moves (M2 design 3.5).
func TestRefreshEveryGenerationInTurn(t *testing.T) {
	f := newRefresh(signedWith)
	token := issued(signedWith, 3, 3).String()
	for g := uint32(4); g <= 6; g++ {
		tokens, err := f.uc.Execute(context.Background(), token, clientIP)
		if err != nil || f.row.State.Generation != g || !tokens.RefreshExpiresAt.Equal(sessionEnd) {
			t.Fatalf("refresh to %d: %v, session at %d, end %v", g, err, f.row.State.Generation, tokens.RefreshExpiresAt)
		}
		token = tokens.RefreshToken
	}
}

// An older generation that the session issued comes back: someone else has
// a copy. The session is revoked, and every token of it fails from now on.
func TestRefreshDetectsReuse(t *testing.T) {
	f := newRefresh(signedWith)

	_, err := f.uc.Execute(context.Background(), issued(signedWith, 2, 2).String(), clientIP)

	if !errors.Is(err, domain.ErrRefreshTokenInvalid) {
		t.Errorf("Execute() = %v, want identity.refresh_token_invalid", err)
	}
	if !f.row.State.Revoked || f.row.reason != "reuse_detected" || f.row.State.Generation != 3 || len(f.sessions.outsideTx) != 0 {
		t.Errorf("session = %+v (%s), writes outside the transaction %q; want revoked for reuse_detected in it", f.row.State, f.row.reason, f.sessions.outsideTx)
	}
	want := `"level":"WARN","msg":"refresh token reused: session revoked","user_id":"` + userID.String() +
		`","session_id":"` + sessionID.String() + `","ip":"203.0.113.7"`
	if !strings.Contains(f.logs.String(), want) {
		t.Errorf("logs = %s, want %s", f.logs, want)
	}
	if _, err := f.uc.Execute(context.Background(), issued(signedWith, 3, 3).String(), clientIP); !errors.Is(err, domain.ErrRefreshTokenInvalid) {
		t.Errorf("the current generation after the reuse: %v, want identity.refresh_token_invalid", err)
	}
}

// Everything else is 401 and leaves the session as it is: only a token the
// session did issue can end it (M2 design 3.5).
func TestRefreshRejectsWithoutRevoking(t *testing.T) {
	otherSecret := issued(signedWith, 3, 9)
	unknown := issued(signedWith, 3, 3)
	unknown.SessionID = uuid.NewV7()
	tests := []struct {
		name  string
		token string
		row   func(*fakeSession)
		reads int
	}{
		{"a forged older generation", forged(2).String(), nil, 1},
		{"the session id of an access token with generation 0", forged(0).String(), nil, 1},
		{"the current generation with another secret", otherSecret.String(), nil, 1},
		{"a newer generation", issued(signedWith, 4, 4).String(), nil, 1},
		{"an unknown session", unknown.String(), nil, 1},
		{"a revoked session", issued(signedWith, 3, 3).String(), func(s *fakeSession) { s.State.Revoked, s.reason = true, "logout" }, 1},
		{"a session ending now", issued(signedWith, 3, 3).String(), func(s *fakeSession) { s.State.ExpiresAt = now }, 1},
		{"a malformed token", "nrv_rt_garbage", nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRefresh(signedWith)
			if tt.row != nil {
				tt.row(f.row)
			}
			before := *f.row

			_, err := f.uc.Execute(context.Background(), tt.token, clientIP)

			if !errors.Is(err, domain.ErrRefreshTokenInvalid) || f.sessions.reads != tt.reads {
				t.Errorf("Execute() = %v after %d reads, want identity.refresh_token_invalid after %d", err, f.sessions.reads, tt.reads)
			}
			if f.row.State.Revoked != before.State.Revoked || f.row.reason != before.reason || f.row.State.Generation != 3 {
				t.Errorf("session = %+v (%s), want it unchanged", f.row.State, f.row.reason)
			}
			if strings.Contains(f.logs.String(), "WARN") {
				t.Errorf("logs = %s, want no reuse warning", f.logs)
			}
		})
	}
}

// After the signing key changed, an older generation's tag no longer
// holds: treated as forged, 401 without revoking. The current generation
// rotates as before: its stored hash proves it (M2 design 3.5, 3.7).
func TestRefreshAfterTheSigningKeyChanged(t *testing.T) {
	newKey := fakeMAC{key: "the new signing key"}
	f := newRefresh(newKey)

	_, oldGeneration := f.uc.Execute(context.Background(), issued(signedWith, 2, 2).String(), clientIP)
	tokens, current := f.uc.Execute(context.Background(), issued(signedWith, 3, 3).String(), clientIP)

	if !errors.Is(oldGeneration, domain.ErrRefreshTokenInvalid) || f.row.reason != "" {
		t.Errorf("older generation: %v, revoke reason %q; want 401 without revoking", oldGeneration, f.row.reason)
	}
	next, _ := domain.ParseRefreshToken(tokens.RefreshToken)
	if current != nil || f.row.State.Generation != 4 || !newKey.Verify(next.MACMessage(), next.Tag) {
		t.Errorf("current generation: %v, session at %d; want it rotated to 4 and tagged under the new key", current, f.row.State.Generation)
	}
}

// Another refresh with the same token commits between the read and the
// conditional UPDATE: the UPDATE misses, the row is read again, and the
// token is now an older generation the session issued: reuse (M2 design 3.5).
func TestRefreshJudgesAgainWhenAConcurrentRefreshWins(t *testing.T) {
	f := newRefresh(signedWith)
	f.sessions.beforeWrite = func() {
		f.sessions.beforeWrite = nil
		f.row.State.Generation, f.row.State.TokenHash = 4, issued(signedWith, 4, 4).SecretHash()
	}

	_, err := f.uc.Execute(context.Background(), issued(signedWith, 3, 3).String(), clientIP)

	if !errors.Is(err, domain.ErrRefreshTokenInvalid) || f.sessions.reads != 2 || f.tx.calls != 1 {
		t.Errorf("Execute() = %v after %d reads in %d transactions; want 401 after a second read in the same one", err, f.sessions.reads, f.tx.calls)
	}
	if !f.row.State.Revoked || f.row.reason != "reuse_detected" {
		t.Errorf("session = %+v (%s), want revoked for reuse_detected", f.row.State, f.row.reason)
	}
}

// A revocation commits first: the second read rejects, and the
// revocation keeps its reason.
func TestRefreshJudgesAgainWhenARevocationWins(t *testing.T) {
	f := newRefresh(signedWith)
	f.sessions.beforeWrite = func() {
		f.sessions.beforeWrite = nil
		f.row.State.Revoked, f.row.reason = true, "password_reset"
	}

	_, err := f.uc.Execute(context.Background(), issued(signedWith, 3, 3).String(), clientIP)

	if !errors.Is(err, domain.ErrRefreshTokenInvalid) || f.sessions.reads != 2 || f.row.reason != "password_reset" {
		t.Errorf("Execute() = %v after %d reads, reason %q; want 401 after a second read, password_reset kept", err, f.sessions.reads, f.row.reason)
	}
}

// The rotation cannot miss twice (a session never goes back to a
// generation); if it does, that is a fault, not an answer.
func TestRefreshGivesUpWhenTheRotationMissesTwice(t *testing.T) {
	f := newRefresh(signedWith)
	state := f.row.State
	f.sessions.beforeWrite = func() { f.row.State.Revoked = true }
	f.sessions.afterWrite = func() { f.row.State = state }

	_, err := f.uc.Execute(context.Background(), issued(signedWith, 3, 3).String(), clientIP)

	var se *shared.Error
	if err == nil || errors.As(err, &se) || f.sessions.reads != 2 {
		t.Errorf("Execute() = %v after %d reads, want a fault after two", err, f.sessions.reads)
	}
}

// Neither a token nor its hash reaches the log (M2 design 8.4).
func TestRefreshLogsNoSecret(t *testing.T) {
	f := newRefresh(signedWith)
	tokens, err := f.uc.Execute(context.Background(), issued(signedWith, 3, 3).String(), clientIP)
	if err != nil {
		t.Fatal(err)
	}
	old := issued(signedWith, 3, 3)
	_, _ = f.uc.Execute(context.Background(), old.String(), clientIP)

	logs := f.logs.String()
	for _, secret := range []string{old.String(), tokens.RefreshToken, tokens.AccessToken, hex.EncodeToString(old.SecretHash()), hex.EncodeToString(f.row.State.TokenHash)} {
		if strings.Contains(logs, secret) {
			t.Errorf("logs contain %q:\n%s", secret, logs)
		}
	}
}
```

`server/internal/modules/identity/app/logout_test.go`（新文件）：

```go
package app_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
)

// newLogout shares newRefresh's session: generation 3, secret 3.
func newLogout() (*app.Logout, *refreshFixture) {
	f := newRefresh(signedWith)
	return app.NewLogout(f.sessions, clocktest.At(now), slog.New(slog.NewJSONHandler(f.logs, nil))), f
}

func TestLogoutEndsTheSessionOfTheCurrentGeneration(t *testing.T) {
	uc, f := newLogout()

	err := uc.Execute(context.Background(), issued(signedWith, 3, 3).String())

	if err != nil || !f.row.State.Revoked || f.row.reason != "logout" || f.tx.calls != 0 {
		t.Errorf("Execute() = %v, session %+v (%s), %d transactions; want revoked for logout by one statement", err, f.row.State, f.row.reason, f.tx.calls)
	}
	if want := `"msg":"signed out","session_id":"` + sessionID.String() + `"`; !strings.Contains(f.logs.String(), want) {
		t.Errorf("logs = %s, want %s", f.logs, want)
	}
}

// Any other token changes nothing and is no error: the answer never tells
// what the token was, and an older generation is no reuse here (M2 design
// 3.5).
func TestLogoutLeavesTheSessionForAnyOtherToken(t *testing.T) {
	unknown := issued(signedWith, 3, 3)
	unknown.SessionID = uuid.NewV7()
	tests := []struct {
		name  string
		token string
		row   func(*fakeSession)
	}{
		{"an older generation the session issued", issued(signedWith, 2, 2).String(), nil},
		{"a forged older generation", forged(2).String(), nil},
		{"the current generation with another secret", issued(signedWith, 3, 9).String(), nil},
		{"a newer generation", issued(signedWith, 4, 4).String(), nil},
		{"an unknown session", unknown.String(), nil},
		{"a revoked session", issued(signedWith, 3, 3).String(), func(s *fakeSession) { s.State.Revoked, s.reason = true, "password_reset" }},
		{"a session ending now", issued(signedWith, 3, 3).String(), func(s *fakeSession) { s.State.ExpiresAt = now }},
		{"a malformed token", "not a token", nil},
		{"empty", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newLogout()
			if tt.row != nil {
				tt.row(f.row)
			}
			before := *f.row

			err := uc.Execute(context.Background(), tt.token)

			if err != nil || f.row.State.Revoked != before.State.Revoked || f.row.reason != before.reason || f.row.State.Generation != 3 {
				t.Errorf("Execute() = %v, session %+v (%s); want nil and the session unchanged", err, f.row.State, f.row.reason)
			}
			if f.logs.Len() != 0 {
				t.Errorf("logs = %s, want none", f.logs)
			}
		})
	}
}
```

Run: `go -C server test -count=1 -race ./internal/modules/identity/app/`
Expected: `ok  	github.com/open-nerve/NerveProject/server/internal/modules/identity/app`

- [ ] **Step 4: argon2 和 MAC 的校验**

`server/internal/modules/identity/adapter/argon2/hasher.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/modules/identity/adapter/argon2/hasher.go
+++ b/server/internal/modules/identity/adapter/argon2/hasher.go
@@ -5,9 +5,13 @@
 import (
 	"context"
 	"crypto/rand"
+	"crypto/subtle"
 	"encoding/base64"
+	"errors"
 	"fmt"
 	"log/slog"
+	"strconv"
+	"strings"
 	"sync/atomic"
 	"time"
 
@@ -61,6 +65,67 @@
 		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
 }
 
+// Verify reports whether password matches hash, a PHC string that Hash
+// wrote, and whether hash has other parameters than the current ones, so
+// that login hashes the password again (M2 design 3.8). It takes a slot like
+// Hash, with the same wait and the same 503. A hash in another format is an
+// error.
+func (h *Hasher) Verify(ctx context.Context, password, hash string) (ok, rehash bool, err error) {
+	p, err := parsePHC(hash)
+	if err != nil {
+		return false, false, err
+	}
+	if err := h.acquire(ctx); err != nil {
+		return false, false, err
+	}
+	defer func() { <-h.slots }()
+	key := argon2.IDKey([]byte(password), p.salt, p.iterations, p.memoryKiB, p.parallelism, keyLen)
+	ok = subtle.ConstantTimeCompare(key, p.key) == 1
+	rehash = p.memoryKiB != h.p.MemoryKiB || p.iterations != h.p.Iterations || p.parallelism != h.p.Parallelism
+	return ok, rehash, nil
+}
+
+// phc is a PHC string's parameters, salt and key.
+type phc struct {
+	memoryKiB   uint32
+	iterations  uint32
+	parallelism uint8
+	salt, key   []byte
+}
+
+// errNotOurHash never quotes the hash.
+var errNotOurHash = errors.New("password hash is not an argon2id PHC string of this hasher")
+
+// parsePHC reads $argon2id$v=19$m=<KiB>,t=<iterations>,p=<lanes>$<salt>$<key>
+// with a 16-byte salt and a 32-byte key, the only form Hash writes.
+func parsePHC(s string) (phc, error) {
+	parts := strings.Split(s, "$")
+	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != fmt.Sprintf("v=%d", argon2.Version) {
+		return phc{}, errNotOurHash
+	}
+	params := strings.Split(parts[3], ",")
+	if len(params) != 3 {
+		return phc{}, errNotOurHash
+	}
+	m, okM := param(params[0], "m", 32)
+	t, okT := param(params[1], "t", 32)
+	l, okP := param(params[2], "p", 8)
+	salt, errSalt := base64.RawStdEncoding.Strict().DecodeString(parts[4])
+	key, errKey := base64.RawStdEncoding.Strict().DecodeString(parts[5])
+	if !okM || !okT || !okP || errSalt != nil || errKey != nil || len(salt) != saltLen || len(key) != keyLen {
+		return phc{}, errNotOurHash
+	}
+	return phc{memoryKiB: uint32(m), iterations: uint32(t), parallelism: uint8(l), salt: salt, key: key}, nil
+}
+
+// param reads name=<n> with 0 < n < 2^bits: argon2 panics at zero rounds
+// or lanes.
+func param(s, name string, bits int) (uint64, bool) {
+	value, found := strings.CutPrefix(s, name+"=")
+	n, err := strconv.ParseUint(value, 10, bits)
+	return n, found && err == nil && n > 0
+}
+
 func (h *Hasher) acquire(ctx context.Context) error {
 	select {
 	case h.slots <- struct{}{}:
```

`server/internal/modules/identity/adapter/argon2/hasher_test.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/modules/identity/adapter/argon2/hasher_test.go
+++ b/server/internal/modules/identity/adapter/argon2/hasher_test.go
@@ -8,6 +8,7 @@
 	"errors"
 	"fmt"
 	"log/slog"
+	"slices"
 	"strings"
 	"testing"
 	"time"
@@ -42,7 +43,8 @@
 	if version != argon2.Version || m != 64 || iterations != 1 || p != 1 || errSalt != nil || errKey != nil || len(salt) != 16 || len(key) != 32 {
 		t.Errorf("hash %q: v=%d m=%d t=%d p=%d, salt %d bytes, key %d bytes", phc, version, m, iterations, p, len(salt), len(key))
 	}
-	// What login (M2/P2) will do: the same derivation gives the same key.
+	// Any argon2id implementation reads the string: the same derivation
+	// gives the same key.
 	if again := argon2.IDKey([]byte("Tr0ub4dor&3"), salt, iterations, m, p, 32); subtle.ConstantTimeCompare(again, key) != 1 {
 		t.Error("re-deriving the key from the PHC parameters gives another key")
 	}
@@ -107,6 +109,108 @@
 	}
 }
 
+func TestVerify(t *testing.T) {
+	h := New(testParams, slog.New(slog.DiscardHandler))
+	hash, err := h.Hash(context.Background(), "Tr0ub4dor&3")
+	if err != nil {
+		t.Fatal(err)
+	}
+	tests := []struct {
+		name, password string
+		ok             bool
+	}{
+		{"the password", "Tr0ub4dor&3", true},
+		{"another password", "Tr0ub4dor&4", false},
+		{"the password with a space", "Tr0ub4dor&3 ", false},
+		{"empty", "", false},
+	}
+	for _, tt := range tests {
+		ok, rehash, err := h.Verify(context.Background(), tt.password, hash)
+		if ok != tt.ok || rehash || err != nil {
+			t.Errorf("%s: Verify() = %v, rehash %v, %v; want %v, no rehash", tt.name, ok, rehash, err, tt.ok)
+		}
+	}
+}
+
+// A hash with other parameters still verifies, and asks for a new hash
+// with the current ones (M2 design 3.8).
+func TestVerifyAsksForARehashWhenTheParametersChanged(t *testing.T) {
+	old, err := New(testParams, slog.New(slog.DiscardHandler)).Hash(context.Background(), "Tr0ub4dor&3")
+	if err != nil {
+		t.Fatal(err)
+	}
+	for _, p := range []Params{
+		{MemoryKiB: 128, Iterations: 1, Parallelism: 1, MaxConcurrent: 1, MaxWait: time.Second},
+		{MemoryKiB: 64, Iterations: 2, Parallelism: 1, MaxConcurrent: 1, MaxWait: time.Second},
+		{MemoryKiB: 64, Iterations: 1, Parallelism: 2, MaxConcurrent: 1, MaxWait: time.Second},
+	} {
+		ok, rehash, err := New(p, slog.New(slog.DiscardHandler)).Verify(context.Background(), "Tr0ub4dor&3", old)
+		if !ok || !rehash || err != nil {
+			t.Errorf("Verify() with m=%d t=%d p=%d = %v, rehash %v, %v; want true, rehash", p.MemoryKiB, p.Iterations, p.Parallelism, ok, rehash, err)
+		}
+	}
+}
+
+func TestVerifyRejectsAnotherFormat(t *testing.T) {
+	h := New(testParams, slog.New(slog.DiscardHandler))
+	good, err := h.Hash(context.Background(), "Tr0ub4dor&3")
+	if err != nil {
+		t.Fatal(err)
+	}
+	parts := strings.Split(good, "$")
+	with := func(i int, s string) string {
+		p := slices.Clone(parts)
+		p[i] = s
+		return strings.Join(p, "$")
+	}
+	tests := []struct{ name, hash string }{
+		{"empty", ""},
+		{"bcrypt", "$2b$10$abcdefghijklmnopqrstuuABCDEFGHIJKLMNOPQRSTUVWXYZ01234"},
+		{"argon2i", with(1, "argon2i")},
+		{"another version", with(2, "v=16")},
+		{"no parameters", with(3, "")},
+		{"parameters in another order", with(3, "t=1,m=64,p=1")},
+		{"an extra parameter", with(3, "m=64,t=1,p=1,k=2")},
+		{"zero iterations", with(3, "m=64,t=0,p=1")},
+		{"zero lanes", with(3, "m=64,t=1,p=0")},
+		{"lanes beyond a byte", with(3, "m=64,t=1,p=256")},
+		{"a signed parameter", with(3, "m=+64,t=1,p=1")},
+		{"a short salt", with(4, parts[4][:10])},
+		{"a salt with padding", with(4, parts[4]+"==")},
+		{"a short key", with(5, parts[5][:20])},
+		{"an empty key", with(5, "")},
+		{"a trailing part", good + "$x"},
+	}
+	for _, tt := range tests {
+		ok, rehash, err := h.Verify(context.Background(), "Tr0ub4dor&3", tt.hash)
+		if ok || rehash || !errors.Is(err, errNotOurHash) {
+			t.Errorf("%s: Verify() = %v, rehash %v, %v; want the format error", tt.name, ok, rehash, err)
+		}
+		if tt.hash != "" && strings.Contains(err.Error(), tt.hash) {
+			t.Errorf("%s: the error %q quotes the hash", tt.name, err)
+		}
+	}
+}
+
+// Verify takes a slot like Hash: a login waits and gets 503 the same way,
+// whether its address exists or not.
+func TestVerifyIsBusyWhenNoSlotFreesUp(t *testing.T) {
+	h := New(testParams, slog.New(slog.DiscardHandler))
+	hash, err := h.Hash(context.Background(), "Tr0ub4dor&3")
+	if err != nil {
+		t.Fatal(err)
+	}
+	h.slots <- struct{}{}
+	h.slots <- struct{}{}
+
+	_, _, err = h.Verify(context.Background(), "Tr0ub4dor&3", hash)
+
+	var se *shared.Error
+	if !errors.As(err, &se) || se.Code != shared.CodeServerBusy {
+		t.Errorf("Verify() = %v, want 503 server_busy", err)
+	}
+}
+
 // BenchmarkHashDefaultParams measures one hash with the default parameters
 // (m = 19456 KiB, t = 2, p = 1): go test -bench . -run ^$ ./internal/modules/identity/adapter/argon2
 func BenchmarkHashDefaultParams(b *testing.B) {
```

`server/internal/modules/identity/adapter/signing/mac.go`（完整内容）：

```go
package signing

import (
	"crypto/hmac"
	"crypto/sha256"
)

// RefreshTokenMAC implements app.RefreshTokenMAC: the first 16 bytes of
// HMAC-SHA256 under K = HKDF-SHA256(the signing key's seed, info "nerve
// refresh-token mac v1") (M2 design 3.4). The tag lets the server recognize
// an old generation it issued without storing it (3.5).
type RefreshTokenMAC struct {
	keys *Keys
}

// NewRefreshTokenMAC returns the MAC of keys.
func NewRefreshTokenMAC(keys *Keys) *RefreshTokenMAC {
	return &RefreshTokenMAC{keys: keys}
}

// Tag returns the tag of message.
func (m *RefreshTokenMAC) Tag(message []byte) [16]byte {
	h := hmac.New(sha256.New, m.keys.mac)
	h.Write(message)
	return [16]byte(h.Sum(nil)[:16])
}

// Verify reports whether tag is the tag of message. It compares in constant
// time: the time of a comparison would tell a forger how much of a tag is
// right (M2 design 3.9).
func (m *RefreshTokenMAC) Verify(message []byte, tag [16]byte) bool {
	want := m.Tag(message)
	return hmac.Equal(want[:], tag[:])
}
```

`server/internal/modules/identity/adapter/signing/signing_test.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/modules/identity/adapter/signing/signing_test.go
+++ b/server/internal/modules/identity/adapter/signing/signing_test.go
@@ -255,5 +255,39 @@
 	}
 	if NewRefreshTokenMAC(EphemeralKeys()).Tag(msg) == tag {
 		t.Error("another key gives the same tag")
+	}
+}
+
+func TestRefreshTokenMACVerify(t *testing.T) {
+	m := NewRefreshTokenMAC(testKeys(t))
+	msg := bytes.Repeat([]byte{7}, 52)
+	tag := m.Tag(msg)
+	var random [16]byte
+	_, _ = rand.Read(random[:])
+	lastBit := tag
+	lastBit[15] ^= 1
+
+	if !m.Verify(msg, tag) {
+		t.Error("Verify() of the message's own tag = false")
+	}
+	for name, forged := range map[string][16]byte{
+		"a random tag":                      random,
+		"the tag with its last bit flipped": lastBit,
+		"the zero tag":                      {},
+	} {
+		if m.Verify(msg, forged) {
+			t.Errorf("Verify() of %s = true", name)
+		}
 	}
+	// Session id, generation and secret: a change anywhere voids the tag.
+	for i := range msg {
+		changed := bytes.Clone(msg)
+		changed[i] ^= 1
+		if m.Verify(changed, tag) {
+			t.Errorf("Verify() with byte %d of the message changed = true", i)
+		}
+	}
+	if NewRefreshTokenMAC(EphemeralKeys()).Verify(msg, tag) {
+		t.Error("Verify() under another key = true")
+	}
 }
```

- [ ] **Step 5: 模块入口（过渡）**

`server/internal/modules/identity/module.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/modules/identity/module.go
+++ b/server/internal/modules/identity/module.go
@@ -64,19 +64,21 @@
 	store := postgresadapter.New(d.Pool)
 	tokens := signing.NewAccessTokens(keys)
 	register := app.NewRegister(app.RegisterDeps{
-		Policy:     d.SignupPolicy,
-		Rules:      domain.NewPasswordRules(),
-		Hasher:     argon2adapter.New(argon2adapter.Params(d.Password), d.Logger),
-		Tx:         d.Tx,
-		Users:      store,
-		Profiles:   store,
-		Sessions:   store,
-		Tokens:     tokens,
-		MAC:        signing.NewRefreshTokenMAC(keys),
-		Clock:      d.Clock,
-		Logger:     d.Logger,
-		AccessTTL:  d.AccessTokenTTL,
-		SessionTTL: d.SessionTTL,
+		Policy:   d.SignupPolicy,
+		Rules:    domain.NewPasswordRules(),
+		Hasher:   argon2adapter.New(argon2adapter.Params(d.Password), d.Logger),
+		Tx:       d.Tx,
+		Users:    store,
+		Profiles: store,
+		Sessions: store,
+		Issuance: app.Issuance{
+			Tokens:     tokens,
+			MAC:        signing.NewRefreshTokenMAC(keys),
+			AccessTTL:  d.AccessTokenTTL,
+			SessionTTL: d.SessionTTL,
+		},
+		Clock:  d.Clock,
+		Logger: d.Logger,
 	})
 	return &Module{
 		uc:            httpadapter.UseCases{Register: register, GetMe: app.NewGetMe(store)},
```

- [ ] **Step 6: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`；注册的整程序测试照旧通过（`Issuance` 没有改变注册的行为）。

- [ ] **Step 7: 提交**

```bash
git add server/internal/modules/identity/app server/internal/modules/identity/adapter/argon2 server/internal/modules/identity/adapter/signing server/internal/modules/identity/module.go
```
```bash
git commit -m "feat(M2/P2): login, refresh and logout use cases; argon2 and MAC verification

Login locks the account row, checks the hash snapshot and reveals a
deactivated account only to the right password; an unknown address is
verified against a dummy hash. Refresh rotates conditionally and revokes
on a reuse the MAC tag proves; the revocation commits before the 401.
Registration, login and refresh share Issuance.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 三个用例的 20 个测试在 `-race` 下通过；`TestRefreshRejectsWithoutRevoking` 的伪造旧代、`g = 0` 两个情况和 `TestRefreshAfterTheSigningKeyChanged` 通过；注册的测试照旧通过。

---

### Task 8: 仓储：登录的查询、账户行锁、条件轮换、撤销与退出

**Files:**
- Modify: `server/internal/modules/identity/adapter/postgres/queries/users.sql`、`queries/sessions.sql`、`users.go`、`sessions.go`
- Create: `server/internal/modules/identity/adapter/postgres/credentials_test.go`、`refresh_test.go`
- Generate: `server/internal/modules/identity/adapter/postgres/gen/users.sql.go`、`sessions.sql.go`

**Interfaces:**
- Consumes: Task 7 的端口和类型。
- Produces（spec 2.10）：`*postgresadapter.Store` 按结构满足 `LoginAccountReader`、`CredentialLocker`、`PasswordHashWriter`、`SessionRotator`、`SessionEnder`（`FindLoginAccount`、`LockForCredentials`、`UpdatePasswordHash`、`SessionForRefresh`、`RotateSession`、`RevokeForReuse`、`EndSession`）；七条查询。
- sqlc 按参数第一次出现的列推断类型：`updated_at` 放在每个 `SET` 的第一位，`now` 才是 `time.Time`。

**Tests:**（真实数据库，`pgtest`；审计列等于固定时钟）
- `credentials_test.go`：`TestFindLoginAccount`、`TestLockForCredentialsReadsTheRow`、`TestTheCredentialLockBlocksLocksNotInserts`（持锁时另一个锁在 `lock_timeout = 500ms` 下得到 `55P03`，插入引用这个账户的会话不等待）、`TestUpdatePasswordHash`。
- `refresh_test.go`：`TestSessionForRefresh`、`TestRotateSession`、`TestRotateSessionMisses`（5 个）、`TestRevokeForReuse`、`TestRevokeForReuseKeepsAnEarlierRevocation`、`TestEndSession`、`TestEndSessionMisses`（同样 5 个）。

- [ ] **Step 1: 查询**

`server/internal/modules/identity/adapter/postgres/queries/users.sql`（完整内容）：

```sql
-- name: CreateUser :exec
-- The other columns take their defaults; the audit columns come from the use case's clock (M2 design 3.13).
INSERT INTO users (id, email, password, display_name, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(email), sqlc.arg(password), sqlc.arg(display_name), sqlc.arg(now), sqlc.arg(now));

-- name: GetUser :one
SELECT id, email, first_name, last_name, display_name, user_timezone, created_at
FROM users
WHERE id = sqlc.arg(id);

-- name: FindLoginAccount :one
-- What login reads before its transaction; the hash is its snapshot (M2 design 3.5).
SELECT id, password
FROM users
WHERE email = sqlc.arg(email);

-- name: LockUserForCredentials :one
-- The account row lock of M2 design 3.5. FOR NO KEY UPDATE conflicts with itself and with
-- FOR UPDATE, so the credential transactions of one account run one after another; it does not
-- conflict with the FOR KEY SHARE that foreign-key checks take, so inserting rows that reference
-- the account does not wait.
SELECT password, is_active
FROM users
WHERE id = sqlc.arg(id)
FOR NO KEY UPDATE;

-- name: UpdatePasswordHash :exec
UPDATE users
SET password = sqlc.arg(password), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);
```

`server/internal/modules/identity/adapter/postgres/queries/sessions.sql`（完整内容）：

```sql
-- name: CreateSession :exec
-- A new login: generation 0 (M2 design 3.5).
INSERT INTO auth_sessions (id, user_id, token_hash, generation, user_agent, ip, expires_at, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(token_hash), 0, sqlc.arg(user_agent), sqlc.arg(ip),
        sqlc.arg(expires_at), sqlc.arg(now), sqlc.arg(now));

-- name: GetSessionCredential :one
-- What authentication checks on every request, by primary key (M2 design 3.5);
-- the use case judges it against its clock.
SELECT s.user_id, s.expires_at, s.revoked_at, u.is_active AS user_active
FROM auth_sessions s
JOIN users u ON u.id = s.user_id
WHERE s.id = sqlc.arg(id);

-- name: GetSessionForRefresh :one
-- What a refresh judges its token against, by primary key (M2 design 3.5).
SELECT user_id, generation, token_hash, expires_at, revoked_at
FROM auth_sessions
WHERE id = sqlc.arg(id);

-- name: RotateSession :execrows
-- The conditional rotation of M2 design 3.5: it hits only while the session is still at the
-- generation and hash the refresh judged, unrevoked and unexpired. A concurrent writer's commit
-- makes it re-evaluate the WHERE on the new row, and miss.
-- sqlc types a parameter by its first use: updated_at (NOT NULL) comes first in each SET below,
-- so that now is a time.Time, not a *time.Time.
UPDATE auth_sessions
SET updated_at = sqlc.arg(now), last_refreshed_at = sqlc.arg(now),
    generation = generation + 1, token_hash = sqlc.arg(new_token_hash)
WHERE id = sqlc.arg(id) AND generation = sqlc.arg(generation) AND token_hash = sqlc.arg(token_hash)
  AND revoked_at IS NULL AND expires_at > sqlc.arg(now);

-- name: RevokeSessionForReuse :exec
-- A revoked session keeps the reason it was revoked for.
UPDATE auth_sessions
SET updated_at = sqlc.arg(now), revoked_at = sqlc.arg(now), revoke_reason = 'reuse_detected'
WHERE id = sqlc.arg(id) AND revoked_at IS NULL;

-- name: EndSession :execrows
-- Logout: the same conditions as the rotation (M2 design 3.5).
UPDATE auth_sessions
SET updated_at = sqlc.arg(now), revoked_at = sqlc.arg(now), revoke_reason = 'logout'
WHERE id = sqlc.arg(id) AND generation = sqlc.arg(generation) AND token_hash = sqlc.arg(token_hash)
  AND revoked_at IS NULL AND expires_at > sqlc.arg(now);
```

- [ ] **Step 2: 生成**

Run: `make gen`
Expected: 只有 sqlc 的两个文件改变：

| 生成的文件 | SHA-256 | 行数 |
|---|---|---|
| `server/internal/modules/identity/adapter/postgres/gen/sessions.sql.go` | `7195d33de8c518ef609691858e113b6b2d7f96df7d570a1424811e3e0e72de28` | 180 |
| `server/internal/modules/identity/adapter/postgres/gen/users.sql.go` | `553f4f0c7c86c3ba813ea6c81b64d7b4051205477d77450b29cd88401f19b575` | 128 |

Run: `grep -n "Now  *\*time.Time" server/internal/modules/identity/adapter/postgres/gen/*.go`
Expected: 没有输出：每个参数结构的 `Now` 都是 `time.Time`。

- [ ] **Step 3: 仓储方法**

`server/internal/modules/identity/adapter/postgres/users.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/modules/identity/adapter/postgres/users.go
+++ b/server/internal/modules/identity/adapter/postgres/users.go
@@ -44,6 +44,35 @@
 	}, nil
 }
 
+// FindLoginAccount reads the account of email, a normalized address;
+// app.ErrNotFound when there is none.
+func (s *Store) FindLoginAccount(ctx context.Context, email string) (app.LoginAccount, error) {
+	row, err := s.queries(ctx).FindLoginAccount(ctx, email)
+	if err != nil {
+		return app.LoginAccount{}, notFound(err)
+	}
+	return app.LoginAccount{ID: row.ID, PasswordHash: row.Password}, nil
+}
+
+// LockForCredentials locks account id's row until the transaction ends and
+// reads it; app.ErrNotFound when there is none. Outside a transaction the
+// lock would end with the statement: call it inside one.
+func (s *Store) LockForCredentials(ctx context.Context, id uuid.UUID) (app.LockedAccount, error) {
+	row, err := s.queries(ctx).LockUserForCredentials(ctx, id)
+	if err != nil {
+		return app.LockedAccount{}, notFound(err)
+	}
+	return app.LockedAccount{PasswordHash: row.Password, Active: row.IsActive}, nil
+}
+
+// UpdatePasswordHash stores hash as account id's password.
+func (s *Store) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string, now time.Time) error {
+	if err := s.queries(ctx).UpdatePasswordHash(ctx, gen.UpdatePasswordHashParams{Password: hash, Now: now, ID: id}); err != nil {
+		return fmt.Errorf("update password hash: %w", err)
+	}
+	return nil
+}
+
 // CreateDefaultProfile inserts the profile of a new account with Plane's
 // model defaults.
 func (s *Store) CreateDefaultProfile(ctx context.Context, id, userID uuid.UUID, now time.Time) error {
```

`server/internal/modules/identity/adapter/postgres/sessions.go`（完整内容）：

```go
package postgresadapter

import (
	"context"
	"fmt"
	"net/netip"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// CreateSession inserts a login at generation 0. An unknown client IP is
// stored as NULL.
func (s *Store) CreateSession(ctx context.Context, n app.NewSession) error {
	var ip *netip.Addr
	if n.IP.IsValid() {
		ip = &n.IP
	}
	err := s.queries(ctx).CreateSession(ctx, gen.CreateSessionParams{
		ID: n.ID, UserID: n.UserID, TokenHash: n.TokenHash, UserAgent: n.UserAgent, Ip: ip, ExpiresAt: n.ExpiresAt, Now: n.Now,
	})
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// SessionCredential reads what authentication checks of session id;
// app.ErrNotFound when there is none.
func (s *Store) SessionCredential(ctx context.Context, id uuid.UUID) (app.SessionCredential, error) {
	row, err := s.queries(ctx).GetSessionCredential(ctx, id)
	if err != nil {
		return app.SessionCredential{}, notFound(err)
	}
	return app.SessionCredential{
		UserID:     row.UserID,
		ExpiresAt:  row.ExpiresAt,
		Revoked:    row.RevokedAt != nil,
		UserActive: row.UserActive,
	}, nil
}

// SessionForRefresh reads what a refresh judges its token against;
// app.ErrNotFound when there is no session id.
func (s *Store) SessionForRefresh(ctx context.Context, id uuid.UUID) (app.RefreshSession, error) {
	row, err := s.queries(ctx).GetSessionForRefresh(ctx, id)
	if err != nil {
		return app.RefreshSession{}, notFound(err)
	}
	return app.RefreshSession{UserID: row.UserID, State: domain.SessionState{
		Generation: uint32(row.Generation), // CHECK (generation >= 0)
		TokenHash:  row.TokenHash,
		Revoked:    row.RevokedAt != nil,
		ExpiresAt:  row.ExpiresAt,
	}}, nil
}

// RotateSession moves the session from g to the next generation with
// newHash, stamping last_refreshed_at; false when it is no longer at g.
// g.Generation fits the column's integer: domain.ParseRefreshToken
// bounds it.
func (s *Store) RotateSession(ctx context.Context, g app.SessionGeneration, newHash []byte) (bool, error) {
	n, err := s.queries(ctx).RotateSession(ctx, gen.RotateSessionParams{
		Now: g.Now, NewTokenHash: newHash, ID: g.ID, Generation: int32(g.Generation), TokenHash: g.TokenHash,
	})
	if err != nil {
		return false, fmt.Errorf("rotate session: %w", err)
	}
	return n == 1, nil
}

// RevokeForReuse revokes session id with reason reuse_detected, unless it
// is revoked already.
func (s *Store) RevokeForReuse(ctx context.Context, id uuid.UUID, now time.Time) error {
	if err := s.queries(ctx).RevokeSessionForReuse(ctx, gen.RevokeSessionForReuseParams{Now: now, ID: id}); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

// EndSession revokes the session with reason logout while it is at g;
// false when it is not.
func (s *Store) EndSession(ctx context.Context, g app.SessionGeneration) (bool, error) {
	n, err := s.queries(ctx).EndSession(ctx, gen.EndSessionParams{
		Now: g.Now, ID: g.ID, Generation: int32(g.Generation), TokenHash: g.TokenHash,
	})
	if err != nil {
		return false, fmt.Errorf("end session: %w", err)
	}
	return n == 1, nil
}
```

- [ ] **Step 4: 集成测试**

`server/internal/modules/identity/adapter/postgres/credentials_test.go`（新文件）：

```go
package postgresadapter_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func TestFindLoginAccount(t *testing.T) {
	s, _ := newStore(t)
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)

	got, err := s.FindLoginAccount(context.Background(), "alice@corp.com")

	if want := (app.LoginAccount{ID: u.ID, PasswordHash: u.PasswordHash}); err != nil || got != want {
		t.Errorf("FindLoginAccount() = %+v, %v; want %+v", got, err, want)
	}
	// The address is matched as given: the use case normalizes it first.
	for _, email := range []string{"bob@corp.com", "Alice@corp.com", " alice@corp.com"} {
		if _, err := s.FindLoginAccount(context.Background(), email); !errors.Is(err, app.ErrNotFound) {
			t.Errorf("FindLoginAccount(%q) = %v, want app.ErrNotFound", email, err)
		}
	}
}

func TestLockForCredentialsReadsTheRow(t *testing.T) {
	s, pool := newStore(t)
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)
	tx := postgres.NewTxManager(pool, 2*time.Second)

	var got app.LockedAccount
	var unknown error
	err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
		var err error
		got, err = s.LockForCredentials(ctx, u.ID)
		_, unknown = s.LockForCredentials(ctx, uuid.NewV7())
		return err
	})

	if want := (app.LockedAccount{PasswordHash: u.PasswordHash, Active: true}); err != nil || got != want {
		t.Errorf("LockForCredentials() = %+v, %v; want %+v", got, err, want)
	}
	if !errors.Is(unknown, app.ErrNotFound) {
		t.Errorf("LockForCredentials(unknown) = %v, want app.ErrNotFound", unknown)
	}
}

// FOR NO KEY UPDATE (M2 design 3.5): while one transaction holds the lock,
// a second lock of the row waits, and inserting a session that references
// the account does not (interleaving 6). lock_timeout turns a wait into a
// failure.
func TestTheCredentialLockBlocksLocksNotInserts(t *testing.T) {
	s, pool := newStore(t)
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	locked, release := make(chan struct{}), make(chan struct{})
	held := make(chan error, 1)
	go func() {
		held <- tx.WithinTx(context.Background(), func(ctx context.Context) error {
			if _, err := s.LockForCredentials(ctx, u.ID); err != nil {
				return err
			}
			close(locked)
			<-release
			return nil
		})
	}()
	<-locked
	defer func() {
		close(release)
		if err := <-held; err != nil {
			t.Errorf("the transaction holding the lock: %v", err)
		}
	}()
	withTimeout := func(fn func(ctx context.Context) error) error {
		return tx.WithinTx(context.Background(), func(ctx context.Context) error {
			if _, err := postgres.DB(ctx, pool).Exec(ctx, "SET LOCAL lock_timeout = '500ms'"); err != nil {
				return err
			}
			return fn(ctx)
		})
	}
	hash := sha256.Sum256([]byte("secret"))

	insert := withTimeout(func(ctx context.Context) error {
		return s.CreateSession(ctx, app.NewSession{ID: uuid.NewV7(), UserID: u.ID, TokenHash: hash[:], ExpiresAt: now.Add(time.Hour), Now: now})
	})
	secondLock := withTimeout(func(ctx context.Context) error {
		_, err := s.LockForCredentials(ctx, u.ID)
		return err
	})

	if insert != nil {
		t.Errorf("inserting a session under the lock: %v, want no wait", insert)
	}
	var pgErr *pgconn.PgError
	if !errors.As(secondLock, &pgErr) || pgErr.Code != "55P03" {
		t.Errorf("a second lock: %v, want lock_not_available after waiting", secondLock)
	}
}

func TestUpdatePasswordHash(t *testing.T) {
	s, pool := newStore(t)
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)
	later := now.Add(time.Hour)

	if err := s.UpdatePasswordHash(context.Background(), u.ID, "$argon2id$new", later); err != nil {
		t.Fatal(err)
	}

	var password string
	var created, updated time.Time
	if err := pool.QueryRow(context.Background(), "SELECT password, created_at, updated_at FROM users WHERE id = $1", u.ID).
		Scan(&password, &created, &updated); err != nil {
		t.Fatal(err)
	}
	if password != "$argon2id$new" || !created.Equal(now) || !updated.Equal(later) {
		t.Errorf("row = %q, created %v, updated %v; want the new hash, updated at %v", password, created, updated, later)
	}
}
```

`server/internal/modules/identity/adapter/postgres/refresh_test.go`（新文件）：

```go
package postgresadapter_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

var (
	secretHash = sha256.Sum256([]byte("secret 0"))
	nextHash   = sha256.Sum256([]byte("secret 1"))
	sessionEnd = now.Add(time.Hour)
	later      = now.Add(time.Minute)
)

// sessionRow is what the refresh tests read back of a session.
type sessionRow struct {
	generation                int32
	tokenHash                 []byte
	expires, created, updated time.Time
	lastRefreshed, revoked    *time.Time
	reason                    *string
}

// newSession inserts a session of a new account at generation 0 with
// secretHash, until sessionEnd.
func newSession(t *testing.T, s *postgresadapter.Store) app.NewSession {
	t.Helper()
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)
	n := app.NewSession{ID: uuid.NewV7(), UserID: u.ID, TokenHash: secretHash[:], ExpiresAt: sessionEnd, Now: now}
	if err := s.CreateSession(context.Background(), n); err != nil {
		t.Fatal(err)
	}
	return n
}

func readSession(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) sessionRow {
	t.Helper()
	var r sessionRow
	err := pool.QueryRow(context.Background(), `SELECT generation, token_hash, expires_at, created_at, updated_at,
		last_refreshed_at, revoked_at, revoke_reason FROM auth_sessions WHERE id = $1`, id).
		Scan(&r.generation, &r.tokenHash, &r.expires, &r.created, &r.updated, &r.lastRefreshed, &r.revoked, &r.reason)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func exec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatal(err)
	}
}

func TestSessionForRefresh(t *testing.T) {
	s, pool := newStore(t)
	n := newSession(t, s)

	got, err := s.SessionForRefresh(context.Background(), n.ID)

	want := app.RefreshSession{UserID: n.UserID, State: domain.SessionState{Generation: 0, TokenHash: secretHash[:], ExpiresAt: sessionEnd}}
	if err != nil || got.UserID != want.UserID || got.State.Generation != 0 || !bytes.Equal(got.State.TokenHash, secretHash[:]) ||
		got.State.Revoked || !got.State.ExpiresAt.Equal(sessionEnd) {
		t.Errorf("SessionForRefresh() = %+v, %v; want %+v", got, err, want)
	}
	exec(t, pool, "UPDATE auth_sessions SET generation = 7, revoked_at = now(), revoke_reason = 'logout'")
	if got, err := s.SessionForRefresh(context.Background(), n.ID); err != nil || got.State.Generation != 7 || !got.State.Revoked {
		t.Errorf("SessionForRefresh() after a change = %+v, %v; want generation 7, revoked", got, err)
	}
	if _, err := s.SessionForRefresh(context.Background(), uuid.NewV7()); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("SessionForRefresh(unknown) = %v, want app.ErrNotFound", err)
	}
}

// The conditions of rotation and logout (M2 design 3.5): each case breaks
// one, and neither statement may touch the row.
var missedConditions = []struct {
	name  string
	g     func(*app.SessionGeneration)
	setup string
}{
	{"another generation", func(g *app.SessionGeneration) { g.Generation = 1 }, ""},
	{"another hash", func(g *app.SessionGeneration) { g.TokenHash = nextHash[:] }, ""},
	{"an unknown session", func(g *app.SessionGeneration) { g.ID = uuid.NewV7() }, ""},
	{"a revoked session", nil, "UPDATE auth_sessions SET revoked_at = now(), revoke_reason = 'password_reset'"},
	{"a session ending at this instant", func(g *app.SessionGeneration) { g.Now = sessionEnd }, ""},
}

func TestRotateSession(t *testing.T) {
	s, pool := newStore(t)
	n := newSession(t, s)
	g := app.SessionGeneration{ID: n.ID, Generation: 0, TokenHash: secretHash[:], Now: later}

	rotated, err := s.RotateSession(context.Background(), g, nextHash[:])

	r := readSession(t, pool, n.ID)
	if err != nil || !rotated || r.generation != 1 || !bytes.Equal(r.tokenHash, nextHash[:]) || r.lastRefreshed == nil || !r.lastRefreshed.Equal(later) ||
		!r.updated.Equal(later) || !r.expires.Equal(sessionEnd) || !r.created.Equal(now) || r.revoked != nil {
		t.Errorf("RotateSession() = %v, %v; row %+v; want generation 1 with the new hash, refreshed and updated at %v, the end unchanged", rotated, err, r, later)
	}
	if again, err := s.RotateSession(context.Background(), g, nextHash[:]); again || err != nil {
		t.Errorf("RotateSession() of the old generation again = %v, %v; want false", again, err)
	}
}

func TestRotateSessionMisses(t *testing.T) {
	for _, tt := range missedConditions {
		t.Run(tt.name, func(t *testing.T) {
			s, pool := newStore(t)
			n := newSession(t, s)
			if tt.setup != "" {
				exec(t, pool, tt.setup)
			}
			g := app.SessionGeneration{ID: n.ID, Generation: 0, TokenHash: secretHash[:], Now: later}
			if tt.g != nil {
				tt.g(&g)
			}
			before := readSession(t, pool, n.ID)

			rotated, err := s.RotateSession(context.Background(), g, nextHash[:])

			if after := readSession(t, pool, n.ID); rotated || err != nil || after.generation != before.generation || !after.updated.Equal(before.updated) {
				t.Errorf("RotateSession() = %v, %v; row %+v, want it unchanged from %+v", rotated, err, after, before)
			}
		})
	}
}

func TestRevokeForReuse(t *testing.T) {
	s, pool := newStore(t)
	n := newSession(t, s)

	if err := s.RevokeForReuse(context.Background(), n.ID, later); err != nil {
		t.Fatal(err)
	}

	r := readSession(t, pool, n.ID)
	if r.revoked == nil || !r.revoked.Equal(later) || r.reason == nil || *r.reason != "reuse_detected" || !r.updated.Equal(later) {
		t.Errorf("row = %+v, want revoked at %v for reuse_detected", r, later)
	}
}

// A session revoked for another reason keeps it.
func TestRevokeForReuseKeepsAnEarlierRevocation(t *testing.T) {
	s, pool := newStore(t)
	n := newSession(t, s)
	exec(t, pool, "UPDATE auth_sessions SET revoked_at = $1, revoke_reason = 'password_reset'", now)

	if err := s.RevokeForReuse(context.Background(), n.ID, later); err != nil {
		t.Fatal(err)
	}

	if r := readSession(t, pool, n.ID); !r.revoked.Equal(now) || *r.reason != "password_reset" {
		t.Errorf("row = %+v, want the password_reset revocation at %v kept", r, now)
	}
}

func TestEndSession(t *testing.T) {
	s, pool := newStore(t)
	n := newSession(t, s)

	ended, err := s.EndSession(context.Background(), app.SessionGeneration{ID: n.ID, Generation: 0, TokenHash: secretHash[:], Now: later})

	r := readSession(t, pool, n.ID)
	if err != nil || !ended || r.revoked == nil || !r.revoked.Equal(later) || *r.reason != "logout" || !r.updated.Equal(later) || r.generation != 0 {
		t.Errorf("EndSession() = %v, %v; row %+v; want revoked at %v for logout", ended, err, r, later)
	}
}

func TestEndSessionMisses(t *testing.T) {
	for _, tt := range missedConditions {
		t.Run(tt.name, func(t *testing.T) {
			s, pool := newStore(t)
			n := newSession(t, s)
			if tt.setup != "" {
				exec(t, pool, tt.setup)
			}
			g := app.SessionGeneration{ID: n.ID, Generation: 0, TokenHash: secretHash[:], Now: later}
			if tt.g != nil {
				tt.g(&g)
			}
			before := readSession(t, pool, n.ID)

			ended, err := s.EndSession(context.Background(), g)

			after := readSession(t, pool, n.ID)
			if ended || err != nil || !after.updated.Equal(before.updated) || (before.reason == nil) != (after.reason == nil) {
				t.Errorf("EndSession() = %v, %v; row %+v, want it unchanged from %+v", ended, err, after, before)
			}
		})
	}
}
```

- [ ] **Step 5: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`（`internal/archtest` 的 `TestSQLCSchemaScope` 照旧通过）。

- [ ] **Step 6: 提交**

```bash
git add server/internal/modules/identity/adapter/postgres
```
```bash
git commit -m "feat(M2/P2): queries for login, the account row lock, rotation, reuse and logout

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run: `make gen-check`
Expected: 没有差异。

**Done when:** 两个生成物与上表相同；`TestTheCredentialLockBlocksLocksNotInserts` 证明锁挡住锁、不挡插入；轮换和退出在五种条件下都不命中；`make gen-check` 干净。

---

### Task 9: 接口 `login`、`refreshTokens`、`logout`；模块的桶与续期的期限；接线

**Files:**
- Modify: `api/modules/identity.yaml`、`api/openapi.yaml`（最终版本）
- Modify: `server/internal/modules/identity/adapter/http/handler.go`、`handler_test.go`
- Create: `server/internal/modules/identity/adapter/http/auth.go`、`me.go`、`limits.go`、`auth_test.go`、`limits_test.go`
- Modify: `server/internal/modules/identity/module.go`（最终版本）
- Modify: `server/internal/shared/error.go`、`error_test.go`、`server/internal/bootstrap/errors_test.go`
- Modify: `server/internal/bootstrap/app.go`、`app_test.go`（最终版本）
- Generate: `server/internal/modules/identity/adapter/http/gen/server.gen.go`、`bodyshape.gen.go`、`api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Consumes: `app.Login`、`Refresh`、`Logout`（Task 7）；仓储（Task 8）；`ratelimit`（Task 2）；`httpserver.RequestMetaFrom`、`RequestID`（Task 4、5）。
- Produces（spec 2.11）：
  - 契约：`login`（`POST /api/v0/auth/login`，码 `identity.invalid_credentials`、`identity.account_deactivated`、`server_busy`，200 `AuthTokens`）、`refreshTokens`（`POST /api/v0/auth/refresh`，码 `identity.refresh_token_invalid`，200 `AuthTokens`）、`logout`（`POST /api/v0/auth/logout`，码 `[]`，204），都是 `security: []`；`LoginRequest`、`RefreshRequest`、`LogoutRequest`；
  - `httpadapter.UseCases{Register, Login, Refresh, Logout, GetMe}`、`Settings{Limits, RefreshDeadline, Logger}`、`Register(router, api, uc, s)`、`PublicOperations()`（四个）；
  - `httpadapter.RateLimiter{AllowAll}`、`Limits{Limiter, LoginIP, LoginIPEmail, RegisterIP}`；
  - `shared.CodeRateLimited`、`shared.RateLimited(retry) *Error`；
  - `identity.Deps` 加上 `RefreshDeadline`、`RateLimits{Limiter, LoginIP, LoginIPEmail, RegisterIP}`；假哈希在 `New` 中算出；
  - `bootstrap`：一个 `ratelimit.New(time.Now)`，六个桶（`bucket(limiter, name, cfg)`），三个给 `API`，三个给 `identity`。

**Tests:**
- `auth_test.go`（7 个）：`TestLoginAnswers200WithTheTokens`、`TestLoginProblems`（4 个）、`TestLoginWithoutAPassword`、`TestRefreshTokens`、`TestRefreshTokensInvalid`、`TestLogout`、`TestLogoutFault`。
- `limits_test.go`（3 个）：`TestLoginLimitsByIPAndByIPWithAddress`、`TestRegisterLimitsByIP`、`TestRefreshAndLogoutTakeNoUnitOfTheModule`。
- `handler_test.go`：测试服务改为 `newServer(t, fakes)`、`serverWith(t, fakes, Settings)`；注册和 `getMe` 的 5 个测试照旧；`TestMain` 的 `apitest.Main` 核对 identity 每个操作声明的每个码（现在共 8 项）都有测试答过。
- `shared/error_test.go`：`TestConstructors` 加上 `RateLimited`；`bootstrap/errors_test.go`：`TestEveryKindBecomesItsProblem` 的 429 用 `shared.RateLimited`。
- `bootstrap/app_test.go`：`testConfig` 加上 `RefreshDeadline: 4s`（缺了它，续期、退出的期限是 0，一进 handler 就到期）；P1 的四个整程序测试自动覆盖三个新操作。

- [ ] **Step 1: 契约**

`api/modules/identity.yaml`（对 Task 5 版本的差异）：

```diff
--- a/api/modules/identity.yaml
+++ b/api/modules/identity.yaml
@@ -18,6 +18,7 @@
         whether the address is registered. The password needs
         8–128 characters with an upper-case letter, a lower-case letter, a
         digit and a special character, and must not be a common password.
+        Registrations have a rate limit of their own per client IP.
       security: []
       x-problem-codes: [identity.signup_disabled, validation_failed, identity.email_taken, server_busy]
       requestBody:
@@ -35,6 +36,85 @@
                 $ref: '#/components/schemas/AuthTokens'
         default:
           $ref: '#/components/responses/Problem'
+  /api/v0/auth/login:
+    post:
+      operationId: login
+      tags: [identity]
+      summary: Sign in with an e-mail address and a password
+      description: >-
+        Starts a new session and returns its tokens. An unknown address and a
+        wrong password get the same identity.invalid_credentials; a
+        deactivated account answers identity.account_deactivated only to the
+        right password. Logins have rate limits of their own: per client IP,
+        and per client IP and address together.
+      security: []
+      x-problem-codes: [identity.invalid_credentials, identity.account_deactivated, server_busy]
+      requestBody:
+        required: true
+        content:
+          application/json:
+            schema:
+              $ref: '#/components/schemas/LoginRequest'
+      responses:
+        '200':
+          description: The new session's tokens.
+          content:
+            application/json:
+              schema:
+                $ref: '#/components/schemas/AuthTokens'
+        default:
+          $ref: '#/components/responses/Problem'
+  /api/v0/auth/refresh:
+    post:
+      operationId: refreshTokens
+      tags: [identity]
+      summary: Exchange a refresh token for the next pair
+      description: >-
+        Returns new tokens of the same session and retires the refresh token
+        sent: the next refresh uses the refresh_token of this response. The
+        session's end, refresh_token_expires_at, never moves. Any refresh
+        token but the current one of a session that has not ended answers
+        identity.refresh_token_invalid; one this session issued before also
+        ends the session, since someone else holds a copy of it.
+      security: []
+      x-problem-codes: [identity.refresh_token_invalid]
+      requestBody:
+        required: true
+        content:
+          application/json:
+            schema:
+              $ref: '#/components/schemas/RefreshRequest'
+      responses:
+        '200':
+          description: The session's next tokens.
+          content:
+            application/json:
+              schema:
+                $ref: '#/components/schemas/AuthTokens'
+        default:
+          $ref: '#/components/responses/Problem'
+  /api/v0/auth/logout:
+    post:
+      operationId: logout
+      tags: [identity]
+      summary: End the session of a refresh token
+      description: >-
+        Ends the session when refresh_token is its current one. Any other
+        token changes nothing and gets the same answer, which tells nothing
+        about the token.
+      security: []
+      x-problem-codes: []
+      requestBody:
+        required: true
+        content:
+          application/json:
+            schema:
+              $ref: '#/components/schemas/LogoutRequest'
+      responses:
+        '204':
+          description: Done, whatever the token was.
+        default:
+          $ref: '#/components/responses/Problem'
   /api/v0/me:
     get:
       operationId: getMe
@@ -91,10 +171,38 @@
           format: email
         password:
           type: string
+    LoginRequest:
+      type: object
+      additionalProperties: false
+      required: [email, password]
+      properties:
+        email:
+          description: The sign-in address, in any case, with or without surrounding blanks.
+          type: string
+          format: email
+        password:
+          type: string
+    RefreshRequest:
+      type: object
+      additionalProperties: false
+      required: [refresh_token]
+      properties:
+        refresh_token:
+          description: The refresh_token of the last AuthTokens of the session.
+          type: string
+    LogoutRequest:
+      type: object
+      additionalProperties: false
+      required: [refresh_token]
+      properties:
+        refresh_token:
+          description: The refresh_token of the last AuthTokens of the session.
+          type: string
     AuthTokens:
       description: >-
         A session's tokens. Send access_token as "Authorization: Bearer"; when
-        it expires, exchange refresh_token for a new pair (M2/P2).
+        it expires, exchange refresh_token for the next pair at
+        POST /api/v0/auth/refresh.
       type: object
       additionalProperties: false
       required: [token_type, access_token, access_token_expires_in, refresh_token, refresh_token_expires_at]
```

`api/openapi.yaml`（对 Task 5 版本的差异）：

```diff
--- a/api/openapi.yaml
+++ b/api/openapi.yaml
@@ -23,6 +23,12 @@
 paths:
   /api/v0/auth/register:
     $ref: 'modules/identity.yaml#/paths/~1api~1v0~1auth~1register'
+  /api/v0/auth/login:
+    $ref: 'modules/identity.yaml#/paths/~1api~1v0~1auth~1login'
+  /api/v0/auth/refresh:
+    $ref: 'modules/identity.yaml#/paths/~1api~1v0~1auth~1refresh'
+  /api/v0/auth/logout:
+    $ref: 'modules/identity.yaml#/paths/~1api~1v0~1auth~1logout'
   /api/v0/me:
     $ref: 'modules/identity.yaml#/paths/~1api~1v0~1me'
   /api/v0/instance:
```

- [ ] **Step 2: 生成**

Run: `make gen`
Expected: 以下四个文件改变，都是最终版本：

| 生成的文件 | SHA-256 | 行数 |
|---|---|---|
| `api/dist/openapi.yaml` | `00a465d77cffb202b5b725da6e4734ff189f99ba419cd7e91f622c45a5816e7f` | 373 |
| `server/internal/modules/identity/adapter/http/gen/bodyshape.gen.go` | `a35a9993252053ee9fa3272c234f18b33991341872d60a3e0140a0a1759dcaab` | 31 |
| `server/internal/modules/identity/adapter/http/gen/server.gen.go` | `adb2aa30856dc47ada82dd0677922f49ca5df52774506b9a27fafe406497779c` | 777 |
| `web/packages/api-client/src/schema.gen.ts` | `0eaaef5ce0e00a69daea6927a035e20696cde16a314e8227c6a26d5e35f77979` | 390 |

- [ ] **Step 3: `shared.RateLimited`**

`server/internal/shared/error.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/shared/error.go
+++ b/server/internal/shared/error.go
@@ -29,6 +29,7 @@
 const (
 	CodeValidationFailed = "validation_failed"
 	CodeUnauthorized     = "unauthorized"
+	CodeRateLimited      = "rate_limited"
 	CodeServerBusy       = "server_busy"
 )
 
@@ -121,6 +122,13 @@
 	return &Error{Kind: KindUnauthenticated, Code: CodeUnauthorized, Detail: "Authentication is required."}
 }
 
+// RateLimited reports a caller over one of a module's rate limits: 429
+// rate_limited with Retry-After, as the platform answers for its own
+// (M2 design 3.10).
+func RateLimited(retry time.Duration) *Error {
+	return &Error{Kind: KindRateLimited, Code: CodeRateLimited, Detail: "Too many requests; retry later.", RetryDelay: retry}
+}
+
 // ServerBusy reports that the server is temporarily overloaded, whoever the
 // caller is: 503 server_busy with Retry-After.
 func ServerBusy(retry time.Duration) *Error {
```

`server/internal/shared/error_test.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/shared/error_test.go
+++ b/server/internal/shared/error_test.go
@@ -49,6 +49,7 @@
 	}{
 		{"Invalid", shared.Invalid(), 422, "validation_failed", 0},
 		{"Unauthenticated", shared.Unauthenticated(), 401, "unauthorized", 0},
+		{"RateLimited", shared.RateLimited(1500 * time.Millisecond), 429, "rate_limited", 1500 * time.Millisecond},
 		{"ServerBusy", shared.ServerBusy(time.Second), 503, "server_busy", time.Second},
 		{"NewError", shared.NewError(shared.KindConflict, "identity.email_taken", "taken"), 409, "identity.email_taken", 0},
 	}
```

`server/internal/bootstrap/errors_test.go`（对 `fd69736` 的差异）：

```diff
--- a/server/internal/bootstrap/errors_test.go
+++ b/server/internal/bootstrap/errors_test.go
@@ -31,8 +31,8 @@
 		{shared.NewError(shared.KindForbidden, "things.forbidden", "d"), `{"status":403,"code":"things.forbidden","title":"Forbidden","detail":"d"}`, ""},
 		{shared.NewError(shared.KindNotFound, "things.missing", "d"), `{"status":404,"code":"things.missing","title":"Not Found","detail":"d"}`, ""},
 		{shared.NewError(shared.KindConflict, "things.taken", "d"), `{"status":409,"code":"things.taken","title":"Conflict","detail":"d"}`, ""},
-		{&shared.Error{Kind: shared.KindRateLimited, Code: "rate_limited", Detail: "d", RetryDelay: 1500 * time.Millisecond},
-			`{"status":429,"code":"rate_limited","title":"Too Many Requests","detail":"d"}`, "2"},
+		{shared.RateLimited(1500 * time.Millisecond),
+			`{"status":429,"code":"rate_limited","title":"Too Many Requests","detail":"Too many requests; retry later."}`, "2"},
 		{fmt.Errorf("register: %w", shared.ServerBusy(time.Second)),
 			`{"status":503,"code":"server_busy","title":"Service Unavailable","detail":"The server is busy; retry shortly."}`, "1"},
 	}
```

- [ ] **Step 4: HTTP 适配器**

`server/internal/modules/identity/adapter/http/handler.go`（完整内容）：

```go
// Package httpadapter serves the identity module's API: it implements the
// strict server that oapi-codegen generates from api/modules/identity.yaml
// into the gen package, translates between the generated types and the use
// cases, and applies the module's own rate limits and the refresh deadline.
package httpadapter

import (
	"context"
	"log/slog"
	"net/netip"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

// RegisterUseCase is app.Register.
type RegisterUseCase interface {
	Execute(ctx context.Context, in app.RegisterInput) (app.Tokens, error)
}

// LoginUseCase is app.Login.
type LoginUseCase interface {
	Execute(ctx context.Context, in app.LoginInput) (app.Tokens, error)
}

// RefreshUseCase is app.Refresh.
type RefreshUseCase interface {
	Execute(ctx context.Context, token string, ip netip.Addr) (app.Tokens, error)
}

// LogoutUseCase is app.Logout.
type LogoutUseCase interface {
	Execute(ctx context.Context, token string) error
}

// GetMeUseCase is app.GetMe.
type GetMeUseCase interface {
	Execute(ctx context.Context) (domain.User, error)
}

// UseCases are the use cases behind the module's operations.
type UseCases struct {
	Register RegisterUseCase
	Login    LoginUseCase
	Refresh  RefreshUseCase
	Logout   LogoutUseCase
	GetMe    GetMeUseCase
}

// Settings are what the handler applies around the use cases.
type Settings struct {
	Limits Limits
	// RefreshDeadline bounds refresh and logout (auth.refresh_deadline): it
	// is part of the rotation protocol, shorter than the web client's
	// timeout (M2 design 3.5).
	RefreshDeadline time.Duration
	Logger          *slog.Logger
}

// PublicOperations are the module's routes that need no token (M2 design
// 3.6), as the generated code registers them.
func PublicOperations() []string {
	return []string{
		"POST /api/v0/auth/register",
		"POST /api/v0/auth/login",
		"POST /api/v0/auth/refresh",
		"POST /api/v0/auth/logout",
	}
}

// Register mounts the module's routes on router behind api's per-route
// middlewares; api.Errors answers binding, decoding and handler errors.
func Register(router *httpserver.Router, api *httpserver.API, uc UseCases, s Settings) {
	strict := gen.NewStrictHandlerWithOptions(handler{uc: uc, s: s}, nil, gen.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  api.Errors.BodyError,
		ResponseErrorHandlerFunc: api.Errors.Write,
	})
	var middlewares []gen.MiddlewareFunc
	for _, m := range api.Middlewares(gen.BodyShapes()) {
		middlewares = append(middlewares, m)
	}
	gen.HandlerWithOptions(strict, gen.StdHTTPServerOptions{
		BaseRouter:       router,
		Middlewares:      middlewares,
		ErrorHandlerFunc: api.Errors.BadRequest,
	})
}

// handler implements gen.StrictServerInterface.
type handler struct {
	uc UseCases
	s  Settings
}
```

`server/internal/modules/identity/adapter/http/auth.go`（新文件）：

```go
package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

// Register serves POST /api/v0/auth/register.
func (h handler) Register(ctx context.Context, req gen.RegisterRequestObject) (gen.RegisterResponseObject, error) {
	meta := httpserver.RequestMetaFrom(ctx)
	if err := h.limitRegister(ctx); err != nil {
		return nil, err
	}
	tokens, err := h.uc.Register.Execute(ctx, app.RegisterInput{
		Email:     req.Body.Email,
		Password:  req.Body.Password,
		UserAgent: meta.UserAgent,
		IP:        meta.ClientIP,
	})
	if err != nil {
		return nil, err
	}
	return gen.Register201JSONResponse(authTokens(tokens)), nil
}

// Login serves POST /api/v0/auth/login.
func (h handler) Login(ctx context.Context, req gen.LoginRequestObject) (gen.LoginResponseObject, error) {
	meta := httpserver.RequestMetaFrom(ctx)
	if err := h.limitLogin(ctx, req.Body.Email); err != nil {
		return nil, err
	}
	tokens, err := h.uc.Login.Execute(ctx, app.LoginInput{
		Email:     req.Body.Email,
		Password:  req.Body.Password,
		UserAgent: meta.UserAgent,
		IP:        meta.ClientIP,
	})
	if err != nil {
		return nil, err
	}
	return gen.Login200JSONResponse(authTokens(tokens)), nil
}

// RefreshTokens serves POST /api/v0/auth/refresh, within the refresh
// deadline. Only the platform's anonymous bucket limits it (M2 design 3.10).
func (h handler) RefreshTokens(ctx context.Context, req gen.RefreshTokensRequestObject) (gen.RefreshTokensResponseObject, error) {
	ctx, cancel := context.WithTimeout(ctx, h.s.RefreshDeadline)
	defer cancel()
	tokens, err := h.uc.Refresh.Execute(ctx, req.Body.RefreshToken, httpserver.RequestMetaFrom(ctx).ClientIP)
	if err != nil {
		return nil, err
	}
	return gen.RefreshTokens200JSONResponse(authTokens(tokens)), nil
}

// Logout serves POST /api/v0/auth/logout, within the refresh deadline.
// Only the platform's anonymous bucket limits it (M2 design 3.10).
func (h handler) Logout(ctx context.Context, req gen.LogoutRequestObject) (gen.LogoutResponseObject, error) {
	ctx, cancel := context.WithTimeout(ctx, h.s.RefreshDeadline)
	defer cancel()
	if err := h.uc.Logout.Execute(ctx, req.Body.RefreshToken); err != nil {
		return nil, err
	}
	return gen.Logout204Response{}, nil
}

func authTokens(t app.Tokens) gen.AuthTokens {
	return gen.AuthTokens{
		TokenType:             gen.AuthTokensTokenTypeBearer,
		AccessToken:           t.AccessToken,
		AccessTokenExpiresIn:  int(t.AccessExpiresIn.Seconds()),
		RefreshToken:          t.RefreshToken,
		RefreshTokenExpiresAt: t.RefreshExpiresAt,
	}
}
```

`server/internal/modules/identity/adapter/http/me.go`（新文件）：

```go
package httpadapter

import (
	"context"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/http/gen"
)

// GetMe serves GET /api/v0/me.
func (h handler) GetMe(ctx context.Context, _ gen.GetMeRequestObject) (gen.GetMeResponseObject, error) {
	u, err := h.uc.GetMe.Execute(ctx)
	if err != nil {
		return nil, err
	}
	return gen.GetMe200JSONResponse{
		ID:           u.ID,
		Email:        u.Email,
		FirstName:    u.FirstName,
		LastName:     u.LastName,
		DisplayName:  u.DisplayName,
		UserTimezone: u.Timezone,
		// Required and null until M5. The zero Nullable is "unspecified" and
		// would marshal as "": set null explicitly.
		AvatarURL:     nullable.NewNullNullable[string](),
		CoverImageURL: nullable.NewNullNullable[string](),
		CreatedAt:     u.CreatedAt,
	}, nil
}
```

`server/internal/modules/identity/adapter/http/limits.go`（新文件）：

```go
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
```

- [ ] **Step 5: 适配器的测试**

`server/internal/modules/identity/adapter/http/handler_test.go`（对 Task 5 版本的差异）：

```diff
--- a/server/internal/modules/identity/adapter/http/handler_test.go
+++ b/server/internal/modules/identity/adapter/http/handler_test.go
@@ -62,8 +62,34 @@
 	return shared.WithActor(ctx, shared.Actor{UserID: userID, SessionID: sessionID}), "session:" + sessionID.String(), nil
 }
 
-func newServer(t *testing.T, register *fakeRegister) http.Handler {
+// fakes are the use cases behind a test server; newServer puts an idle fake
+// in place of each one left nil.
+type fakes struct {
+	register *fakeRegister
+	login    *fakeLogin
+	refresh  *fakeRefresh
+	logout   *fakeLogout
+}
+
+// newServer serves the module with limits no test here reaches.
+func newServer(t *testing.T, f fakes) http.Handler {
 	t.Helper()
+	limiter := ratelimit.New(time.Now)
+	roomy := func(name string) *ratelimit.Bucket {
+		return limiter.Bucket(name, ratelimit.Rate{PerMinute: 600, Burst: 100})
+	}
+	return serverWith(t, f, httpadapter.Settings{
+		Limits:          httpadapter.Limits{Limiter: limiter, LoginIP: roomy("login_ip"), LoginIPEmail: roomy("login_ip_email"), RegisterIP: roomy("register_ip")},
+		RefreshDeadline: refreshDeadline,
+		Logger:          slog.New(slog.DiscardHandler),
+	})
+}
+
+// refreshDeadline is shorter than the request timeout of serverWith.
+const refreshDeadline = 4 * time.Second
+
+func serverWith(t *testing.T, f fakes, s httpadapter.Settings) http.Handler {
+	t.Helper()
 	logger := slog.New(slog.DiscardHandler)
 	router := httpserver.NewRouter(logger)
 	limit := ratelimit.New(time.Now).Bucket("test", ratelimit.Rate{PerMinute: 600, Burst: 100})
@@ -81,7 +107,21 @@
 	if err != nil {
 		t.Fatal(err)
 	}
-	httpadapter.Register(router, api, httpadapter.UseCases{Register: register, GetMe: fakeGetMe{}})
+	if f.register == nil {
+		f.register = &fakeRegister{}
+	}
+	if f.login == nil {
+		f.login = &fakeLogin{}
+	}
+	if f.refresh == nil {
+		f.refresh = &fakeRefresh{}
+	}
+	if f.logout == nil {
+		f.logout = &fakeLogout{}
+	}
+	httpadapter.Register(router, api, httpadapter.UseCases{
+		Register: f.register, Login: f.login, Refresh: f.refresh, Logout: f.logout, GetMe: fakeGetMe{},
+	}, s)
 	return router
 }
 
@@ -95,14 +135,17 @@
 	return res, string(body)
 }
 
-func registerRequest(body string) *http.Request {
-	req := httptest.NewRequest(http.MethodPost, "/api/v0/auth/register", strings.NewReader(body))
+// postJSON is a request from 203.0.113.7 with agent/1.
+func postJSON(path, body string) *http.Request {
+	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
 	req.Header.Set("Content-Type", "application/json")
 	req.Header.Set("User-Agent", "agent/1")
 	req.RemoteAddr = "203.0.113.7:5555"
 	return req
 }
 
+func registerRequest(body string) *http.Request { return postJSON("/api/v0/auth/register", body) }
+
 func TestRegisterAnswers201WithTheTokens(t *testing.T) {
 	register := &fakeRegister{tokens: app.Tokens{
 		AccessToken: "access", AccessExpiresIn: 15 * time.Minute, RefreshToken: "nrv_rt_x", RefreshExpiresAt: created.Add(720 * time.Hour),
@@ -110,7 +153,7 @@
 	req := registerRequest(`{"email":"Alice@Corp.com","password":"Tr0ub4dor&3"}`)
 	apitest.Load(t).CheckRequest(t, req)
 
-	res, body := do(t, newServer(t, register), req)
+	res, body := do(t, newServer(t, fakes{register: register}), req)
 
 	want := `{"access_token":"access","access_token_expires_in":900,"refresh_token":"nrv_rt_x",` +
 		`"refresh_token_expires_at":"2026-10-25T10:00:00.123456Z","token_type":"Bearer"}` + "\n"
@@ -139,7 +182,7 @@
 	}
 	for _, tt := range tests {
 		t.Run(tt.name, func(t *testing.T) {
-			res, body := do(t, newServer(t, &fakeRegister{err: tt.err}), registerRequest(`{"email":"a@b.co","password":"x"}`))
+			res, body := do(t, newServer(t, fakes{register: &fakeRegister{err: tt.err}}), registerRequest(`{"email":"a@b.co","password":"x"}`))
 
 			if res.StatusCode != tt.status || !strings.Contains(body, `"code":"`+tt.code+`"`) || res.Header.Get("Retry-After") != tt.retryAfter {
 				t.Errorf("response = %d %s Retry-After %q, want %d %s", res.StatusCode, body, res.Header.Get("Retry-After"), tt.status, tt.code)
@@ -166,7 +209,7 @@
 	for _, tt := range tests {
 		t.Run(tt.name, func(t *testing.T) {
 			register := &fakeRegister{}
-			res, body := do(t, newServer(t, register), registerRequest(tt.body))
+			res, body := do(t, newServer(t, fakes{register: register}), registerRequest(tt.body))
 
 			if res.StatusCode != tt.status || !strings.Contains(body, tt.want) || strings.Contains(body, "Go struct") {
 				t.Errorf("response = %d %s, want %d with %s", res.StatusCode, body, tt.status, tt.want)
@@ -183,7 +226,7 @@
 	req.Header.Set("Authorization", "Bearer valid")
 	apitest.Load(t).CheckRequest(t, req)
 
-	res, body := do(t, newServer(t, &fakeRegister{}), req)
+	res, body := do(t, newServer(t, fakes{}), req)
 
 	want := `{"avatar_url":null,"cover_image_url":null,"created_at":"2026-09-25T10:00:00.123456Z","display_name":"alice",` +
 		`"email":"alice@corp.com","first_name":"","id":"` + userID.String() + `","last_name":"","user_timezone":"UTC"}` + "\n"
@@ -199,7 +242,7 @@
 			req.Header.Set("Authorization", header)
 		}
 
-		res, body := do(t, newServer(t, &fakeRegister{}), req)
+		res, body := do(t, newServer(t, fakes{}), req)
 
 		if res.StatusCode != http.StatusUnauthorized || !strings.Contains(body, `"code":"unauthorized"`) {
 			t.Errorf("GET /me with %q = %d %s, want 401 unauthorized", header, res.StatusCode, body)
```

`server/internal/modules/identity/adapter/http/auth_test.go`（新文件）：

```go
package httpadapter_test

import (
	"context"
	"errors"
	"net/http"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type fakeLogin struct {
	calls  int
	got    app.LoginInput
	tokens app.Tokens
	err    error
}

func (f *fakeLogin) Execute(_ context.Context, in app.LoginInput) (app.Tokens, error) {
	f.calls++
	f.got = in
	return f.tokens, f.err
}

// fakeRefresh records what it was given and how long its context had left.
type fakeRefresh struct {
	token  string
	ip     netip.Addr
	left   time.Duration
	tokens app.Tokens
	err    error
}

func (f *fakeRefresh) Execute(ctx context.Context, token string, ip netip.Addr) (app.Tokens, error) {
	f.token, f.ip, f.left = token, ip, timeLeft(ctx)
	return f.tokens, f.err
}

type fakeLogout struct {
	token string
	left  time.Duration
	err   error
}

func (f *fakeLogout) Execute(ctx context.Context, token string) error {
	f.token, f.left = token, timeLeft(ctx)
	return f.err
}

// timeLeft is what remains of ctx's deadline; zero without one.
func timeLeft(ctx context.Context) time.Duration {
	deadline, ok := ctx.Deadline()
	if !ok {
		return 0
	}
	return time.Until(deadline)
}

var sampleTokens = app.Tokens{
	AccessToken: "access", AccessExpiresIn: 15 * time.Minute, RefreshToken: "nrv_rt_x", RefreshExpiresAt: created.Add(720 * time.Hour),
}

const sampleTokensJSON = `{"access_token":"access","access_token_expires_in":900,"refresh_token":"nrv_rt_x",` +
	`"refresh_token_expires_at":"2026-10-25T10:00:00.123456Z","token_type":"Bearer"}` + "\n"

func TestLoginAnswers200WithTheTokens(t *testing.T) {
	login := &fakeLogin{tokens: sampleTokens}
	req := postJSON("/api/v0/auth/login", `{"email":" Alice@Corp.com","password":"Tr0ub4dor&3"}`)
	apitest.Load(t).CheckRequest(t, req)

	res, body := do(t, newServer(t, fakes{login: login}), req)

	if res.StatusCode != http.StatusOK || body != sampleTokensJSON {
		t.Errorf("POST /auth/login = %d %s, want 200 %s", res.StatusCode, body, sampleTokensJSON)
	}
	// The use case normalizes the address.
	want := app.LoginInput{Email: " Alice@Corp.com", Password: "Tr0ub4dor&3", UserAgent: "agent/1", IP: netip.MustParseAddr("203.0.113.7")}
	if login.got != want {
		t.Errorf("use case got %+v, want %+v", login.got, want)
	}
}

func TestLoginProblems(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		status     int
		code       string
		challenge  string
		retryAfter string
	}{
		{"unknown address or wrong password", domain.ErrInvalidCredentials, 401, "identity.invalid_credentials", "Bearer", ""},
		{"deactivated", domain.ErrAccountDeactivated, 403, "identity.account_deactivated", "", ""},
		{"hashing saturated", shared.ServerBusy(time.Second), 503, "server_busy", "", "1"},
		{"a fault", errors.New("database is down"), 500, "internal_error", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, body := do(t, newServer(t, fakes{login: &fakeLogin{err: tt.err}}), postJSON("/api/v0/auth/login", `{"email":"a@b.co","password":"x"}`))

			if res.StatusCode != tt.status || !strings.Contains(body, `"code":"`+tt.code+`"`) ||
				res.Header.Get("WWW-Authenticate") != tt.challenge || res.Header.Get("Retry-After") != tt.retryAfter {
				t.Errorf("response = %d %s, WWW-Authenticate %q, Retry-After %q; want %d %s, %q, %q", res.StatusCode, body,
					res.Header.Get("WWW-Authenticate"), res.Header.Get("Retry-After"), tt.status, tt.code, tt.challenge, tt.retryAfter)
			}
		})
	}
}

// A3's API version: a login without a password is the platform's 400.
func TestLoginWithoutAPassword(t *testing.T) {
	login := &fakeLogin{}

	res, body := do(t, newServer(t, fakes{login: login}), postJSON("/api/v0/auth/login", `{"email":"a@b.co"}`))

	want := `"errors":[{"field":"password","code":"required","message":"is required"}]`
	if res.StatusCode != http.StatusBadRequest || !strings.Contains(body, want) || login.calls != 0 {
		t.Errorf("response = %d %s after %d logins, want 400 with %s and none", res.StatusCode, body, login.calls, want)
	}
}

// Refresh and logout run within auth.refresh_deadline, shorter than the
// request timeout (M2 design 3.5).
func TestRefreshTokens(t *testing.T) {
	refresh := &fakeRefresh{tokens: sampleTokens}
	req := postJSON("/api/v0/auth/refresh", `{"refresh_token":"nrv_rt_old"}`)
	apitest.Load(t).CheckRequest(t, req)

	res, body := do(t, newServer(t, fakes{refresh: refresh}), req)

	if res.StatusCode != http.StatusOK || body != sampleTokensJSON {
		t.Errorf("POST /auth/refresh = %d %s, want 200 %s", res.StatusCode, body, sampleTokensJSON)
	}
	if refresh.token != "nrv_rt_old" || refresh.ip != netip.MustParseAddr("203.0.113.7") || refresh.left <= 0 || refresh.left > refreshDeadline {
		t.Errorf("use case got %q from %v with %v left; want the token, the client, at most %v", refresh.token, refresh.ip, refresh.left, refreshDeadline)
	}
}

func TestRefreshTokensInvalid(t *testing.T) {
	res, body := do(t, newServer(t, fakes{refresh: &fakeRefresh{err: domain.ErrRefreshTokenInvalid}}),
		postJSON("/api/v0/auth/refresh", `{"refresh_token":"nrv_rt_old"}`))

	if res.StatusCode != http.StatusUnauthorized || !strings.Contains(body, `"code":"identity.refresh_token_invalid"`) || res.Header.Get("WWW-Authenticate") != "Bearer" {
		t.Errorf("response = %d %s, WWW-Authenticate %q; want 401 identity.refresh_token_invalid with Bearer", res.StatusCode, body, res.Header.Get("WWW-Authenticate"))
	}
}

func TestLogout(t *testing.T) {
	logout := &fakeLogout{}
	req := postJSON("/api/v0/auth/logout", `{"refresh_token":"nrv_rt_current"}`)
	apitest.Load(t).CheckRequest(t, req)

	res, body := do(t, newServer(t, fakes{logout: logout}), req)

	if res.StatusCode != http.StatusNoContent || body != "" {
		t.Errorf("POST /auth/logout = %d %q, want 204 and no body", res.StatusCode, body)
	}
	if logout.token != "nrv_rt_current" || logout.left <= 0 || logout.left > refreshDeadline {
		t.Errorf("use case got %q with %v left, want the token and at most %v", logout.token, logout.left, refreshDeadline)
	}
}

func TestLogoutFault(t *testing.T) {
	res, body := do(t, newServer(t, fakes{logout: &fakeLogout{err: errors.New("database is down")}}),
		postJSON("/api/v0/auth/logout", `{"refresh_token":"nrv_rt_current"}`))

	if res.StatusCode != http.StatusInternalServerError || !strings.Contains(body, `"code":"internal_error"`) {
		t.Errorf("response = %d %s, want 500 internal_error", res.StatusCode, body)
	}
}
```

`server/internal/modules/identity/adapter/http/limits_test.go`（新文件）：

```go
package httpadapter_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/ratelimit"
)

// tightLimits are small buckets on a limiter whose clock stands still, so
// nothing refills during a test: login_ip 3, login_ip_email 2,
// register_ip 1, each regaining a unit a minute.
func tightLimits() httpadapter.Limits {
	limiter := ratelimit.New(func() time.Time { return created })
	return httpadapter.Limits{
		Limiter:      limiter,
		LoginIP:      limiter.Bucket("login_ip", ratelimit.Rate{PerMinute: 1, Burst: 3}),
		LoginIPEmail: limiter.Bucket("login_ip_email", ratelimit.Rate{PerMinute: 1, Burst: 2}),
		RegisterIP:   limiter.Bucket("register_ip", ratelimit.Rate{PerMinute: 1, Burst: 1}),
	}
}

func limitedServer(t *testing.T, f fakes, limits httpadapter.Limits, logs *bytes.Buffer) http.Handler {
	t.Helper()
	return serverWith(t, f, httpadapter.Settings{Limits: limits, RefreshDeadline: refreshDeadline, Logger: slog.New(slog.NewJSONHandler(logs, nil))})
}

// rateLimitLogs are the "rate limited" entries of logs.
func rateLimitLogs(t *testing.T, logs *bytes.Buffer) []map[string]any {
	t.Helper()
	var entries []map[string]any
	for line := range strings.Lines(logs.String()) {
		var e map[string]any
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("log line %q: %v", line, err)
		}
		if e["msg"] == "rate limited" {
			entries = append(entries, e)
		}
	}
	return entries
}

// Login takes a unit of login_ip and of login_ip_email together, or of
// neither (M2 design 3.10): one address fails twice and is refused; the
// refusal leaves login_ip as it was, so other addresses from the client
// get its last unit, then login_ip refuses.
func TestLoginLimitsByIPAndByIPWithAddress(t *testing.T) {
	var logs bytes.Buffer
	login := &fakeLogin{err: domain.ErrInvalidCredentials}
	h := limitedServer(t, fakes{login: login}, tightLimits(), &logs)
	attempt := func(email string) *http.Response {
		res, _ := do(t, h, postJSON("/api/v0/auth/login", `{"email":"`+email+`","password":"x"}`))
		return res
	}

	var statuses []int
	var retryAfter []string
	// " Alice@Corp.com" normalizes to alice@corp.com: the same bucket.
	for _, email := range []string{"alice@corp.com", " Alice@Corp.com", "alice@corp.com", "bob@corp.com", "carol@corp.com"} {
		res := attempt(email)
		statuses = append(statuses, res.StatusCode)
		retryAfter = append(retryAfter, res.Header.Get("Retry-After"))
	}

	if want := []int{401, 401, 429, 401, 429}; !slices.Equal(statuses, want) || login.calls != 3 {
		t.Errorf("statuses = %v after %d logins, want %v after 3", statuses, login.calls, want)
	}
	if retryAfter[2] != "60" || retryAfter[4] != "60" {
		t.Errorf("Retry-After = %q, want 60 on both refusals", retryAfter)
	}
	entries := rateLimitLogs(t, &logs)
	if len(entries) != 2 || entries[0]["bucket"] != "login_ip_email" || entries[1]["bucket"] != "login_ip" ||
		entries[0]["ip"] != "203.0.113.7" || entries[0]["request_id"] == nil {
		t.Errorf("rate limited logs = %v, want login_ip_email then login_ip, with the client and the request", entries)
	}
}

func TestRegisterLimitsByIP(t *testing.T) {
	var logs bytes.Buffer
	register := &fakeRegister{err: domain.ErrEmailTaken}
	h := limitedServer(t, fakes{register: register}, tightLimits(), &logs)

	first, _ := do(t, h, registerRequest(`{"email":"a@b.co","password":"x"}`))
	second, body := do(t, h, registerRequest(`{"email":"c@d.co","password":"x"}`))

	if first.StatusCode != 409 || second.StatusCode != 429 || !strings.Contains(body, `"code":"rate_limited"`) || second.Header.Get("Retry-After") != "60" {
		t.Errorf("responses = %d, %d %s Retry-After %q; want 409, then 429 rate_limited Retry-After 60", first.StatusCode, second.StatusCode, body, second.Header.Get("Retry-After"))
	}
	if entries := rateLimitLogs(t, &logs); len(entries) != 1 || entries[0]["bucket"] != "register_ip" {
		t.Errorf("rate limited logs = %v, want register_ip", entries)
	}
}

// Refresh and logout go through no bucket of the module, only the
// platform's anonymous one (M2 design 3.10): with every module bucket
// empty for the client, they still answer.
func TestRefreshAndLogoutTakeNoUnitOfTheModule(t *testing.T) {
	limits := tightLimits()
	for range 3 {
		limits.Limiter.AllowAll(ratelimit.Check{Bucket: limits.LoginIP, Key: "203.0.113.7"})
	}
	limits.Limiter.AllowAll(ratelimit.Check{Bucket: limits.RegisterIP, Key: "203.0.113.7"})
	var logs bytes.Buffer
	h := limitedServer(t, fakes{refresh: &fakeRefresh{tokens: sampleTokens}}, limits, &logs)

	login, _ := do(t, h, postJSON("/api/v0/auth/login", `{"email":"a@b.co","password":"x"}`))
	register, _ := do(t, h, registerRequest(`{"email":"a@b.co","password":"x"}`))
	refresh, _ := do(t, h, postJSON("/api/v0/auth/refresh", `{"refresh_token":"nrv_rt_x"}`))
	logout, _ := do(t, h, postJSON("/api/v0/auth/logout", `{"refresh_token":"nrv_rt_x"}`))

	if login.StatusCode != 429 || register.StatusCode != 429 || refresh.StatusCode != 200 || logout.StatusCode != 204 {
		t.Errorf("login %d, register %d, refresh %d, logout %d; want 429, 429, 200, 204", login.StatusCode, register.StatusCode, refresh.StatusCode, logout.StatusCode)
	}
}
```

- [ ] **Step 6: 模块入口和接线**

`server/internal/modules/identity/module.go`（完整内容）：

```go
// Package identity is the accounts module (M2 design 3.3, 6.2): accounts,
// profiles, sessions and, from later phases, personal access tokens. It
// brings registration, login, refresh, logout, GET /me and the
// authentication every other operation goes through.
package identity

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	argon2adapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/argon2"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/authn"
	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/http"
	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/signing"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/ratelimit"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Deps are what bootstrap builds for the module.
type Deps struct {
	Pool   *pgxpool.Pool
	Tx     shared.TxManager
	Clock  app.Clock
	Logger *slog.Logger
	// SignupPolicy is auth.signup_enabled (M2 decision 2).
	SignupPolicy app.SignupPolicy
	// SigningKeyPEM is the content of auth.jwt.private_key_file; nil for
	// none, then the key is ephemeral (dev and test only, M2 design 3.7).
	SigningKeyPEM   []byte
	AccessTokenTTL  time.Duration
	SessionTTL      time.Duration
	RefreshDeadline time.Duration // auth.refresh_deadline
	Password        PasswordHashing
	RateLimits      RateLimits
}

// PasswordHashing is auth.password: argon2id's parameters and the limits on
// concurrent hashes (M2 design 3.8).
type PasswordHashing struct {
	MemoryKiB     uint32
	Iterations    uint32
	Parallelism   uint8
	MaxConcurrent int
	MaxWait       time.Duration
}

// RateLimits are the module's own buckets, on the limiter they belong to
// (M2 design 3.10).
type RateLimits struct {
	Limiter      httpadapter.RateLimiter
	LoginIP      *ratelimit.Bucket
	LoginIPEmail *ratelimit.Bucket
	RegisterIP   *ratelimit.Bucket
}

// Module is the wired identity module.
type Module struct {
	uc            httpadapter.UseCases
	settings      httpadapter.Settings
	authenticator *authn.Authenticator
}

// New wires the module. A signing key that cannot be parsed is an error
// that never quotes the key.
func New(d Deps) (*Module, error) {
	keys, err := signingKeys(d)
	if err != nil {
		return nil, err
	}
	hasher := argon2adapter.New(argon2adapter.Params(d.Password), d.Logger)
	// Login verifies an unknown address against this, so that it takes as
	// long as a known one (M2 design 3.9).
	dummy, err := hasher.Hash(context.Background(), rand.Text())
	if err != nil {
		return nil, fmt.Errorf("hash the dummy password: %w", err)
	}
	store := postgresadapter.New(d.Pool)
	tokens := signing.NewAccessTokens(keys)
	issuance := app.Issuance{
		Tokens:     tokens,
		MAC:        signing.NewRefreshTokenMAC(keys),
		AccessTTL:  d.AccessTokenTTL,
		SessionTTL: d.SessionTTL,
	}
	return &Module{
		uc: httpadapter.UseCases{
			Register: app.NewRegister(app.RegisterDeps{
				Policy: d.SignupPolicy, Rules: domain.NewPasswordRules(), Hasher: hasher, Tx: d.Tx,
				Users: store, Profiles: store, Sessions: store, Issuance: issuance, Clock: d.Clock, Logger: d.Logger,
			}),
			Login: app.NewLogin(app.LoginDeps{
				Accounts: store, Locker: store, Passwords: store, Sessions: store, Hasher: hasher, Tx: d.Tx,
				Issuance: issuance, Clock: d.Clock, Logger: d.Logger, DummyHash: dummy,
			}),
			Refresh: app.NewRefresh(app.RefreshDeps{Sessions: store, Tx: d.Tx, Issuance: issuance, Clock: d.Clock, Logger: d.Logger}),
			Logout:  app.NewLogout(store, d.Clock, d.Logger),
			GetMe:   app.NewGetMe(store),
		},
		settings: httpadapter.Settings{
			Limits:          httpadapter.Limits(d.RateLimits),
			RefreshDeadline: d.RefreshDeadline,
			Logger:          d.Logger,
		},
		authenticator: authn.New(app.NewAuthenticate(tokens, store, d.Clock)),
	}, nil
}

func signingKeys(d Deps) (*signing.Keys, error) {
	if d.SigningKeyPEM == nil {
		d.Logger.Warn("auth.jwt.private_key_file is not set: signing with an ephemeral key; " +
			"access tokens stop verifying at restart (dev and test only)")
		return signing.EphemeralKeys(), nil
	}
	keys, err := signing.ParseKeys(d.SigningKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("auth.jwt.private_key_file: %w", err)
	}
	return keys, nil
}

// PublicOperations are the module's routes that need no token.
func (m *Module) PublicOperations() []string {
	return httpadapter.PublicOperations()
}

// Authenticator checks the bearer token of every non-public operation.
func (m *Module) Authenticator() httpserver.Authenticator {
	return m.authenticator
}

// Register mounts the module's API on router behind api's middlewares.
func (m *Module) Register(router *httpserver.Router, api *httpserver.API) {
	httpadapter.Register(router, api, m.uc, m.settings)
}
```

`server/internal/bootstrap/app.go`（对 Task 5 版本的差异）：

```diff
--- a/server/internal/bootstrap/app.go
+++ b/server/internal/bootstrap/app.go
@@ -69,16 +69,21 @@
 		return nil, err
 	}
 	a := &app{cfg: cfg, logger: logger, pool: pool, migrator: migrator}
+	// Every rate-limit bucket lives on this limiter (M2 design 3.10). It reads
+	// time.Now, not the Clock: the monotonic reading keeps a step of the wall
+	// clock from filling or draining the buckets.
+	limiter := ratelimit.New(time.Now)
 
 	ident, err := identity.New(identity.Deps{
-		Pool:           pool,
-		Tx:             postgres.NewTxManager(pool, cfg.Database.CommitTimeout),
-		Clock:          clock.System{},
-		Logger:         logger,
-		SignupPolicy:   signupSwitch(cfg.Auth.SignupEnabled),
-		SigningKeyPEM:  signingKey,
-		AccessTokenTTL: cfg.Auth.AccessTokenTTL,
-		SessionTTL:     cfg.Auth.SessionTTL,
+		Pool:            pool,
+		Tx:              postgres.NewTxManager(pool, cfg.Database.CommitTimeout),
+		Clock:           clock.System{},
+		Logger:          logger,
+		SignupPolicy:    signupSwitch(cfg.Auth.SignupEnabled),
+		SigningKeyPEM:   signingKey,
+		AccessTokenTTL:  cfg.Auth.AccessTokenTTL,
+		SessionTTL:      cfg.Auth.SessionTTL,
+		RefreshDeadline: cfg.Auth.RefreshDeadline,
 		Password: identity.PasswordHashing{
 			MemoryKiB:     cfg.Auth.Password.Argon2MemoryKiB,
 			Iterations:    cfg.Auth.Password.Argon2Iterations,
@@ -86,6 +91,12 @@
 			MaxConcurrent: cfg.Auth.Password.MaxConcurrentHashes,
 			MaxWait:       cfg.Auth.Password.MaxWait,
 		},
+		RateLimits: identity.RateLimits{
+			Limiter:      limiter,
+			LoginIP:      bucket(limiter, "login_ip", cfg.RateLimit.LoginIP),
+			LoginIPEmail: bucket(limiter, "login_ip_email", cfg.RateLimit.LoginIPEmail),
+			RegisterIP:   bucket(limiter, "register_ip", cfg.RateLimit.RegisterIP),
+		},
 	})
 	if err != nil {
 		a.close()
@@ -98,9 +109,6 @@
 		httpserver.Check{Name: "migrations", Run: migrator.CheckUpToDate},
 	)
 	a.publicOperations = slices.Concat(ident.PublicOperations(), inst.PublicOperations())
-	// time.Now, not the Clock: its monotonic reading keeps a step of the wall
-	// clock from filling or draining the buckets.
-	limiter := ratelimit.New(time.Now)
 	api, err := httpserver.NewAPI(httpserver.APIConfig{
 		Logger:           logger,
 		Authenticator:    ident.Authenticator(),
```

`server/internal/bootstrap/app_test.go`（对 Task 5 版本的差异）：

```diff
--- a/server/internal/bootstrap/app_test.go
+++ b/server/internal/bootstrap/app_test.go
@@ -56,9 +56,10 @@
 		},
 		Database: config.DatabaseConfig{URL: dbURL, MaxConns: 4, AutoMigrate: autoMigrate, CommitTimeout: 2 * time.Second},
 		Auth: config.AuthConfig{
-			SignupEnabled:  true,
-			AccessTokenTTL: 15 * time.Minute,
-			SessionTTL:     720 * time.Hour,
+			SignupEnabled:   true,
+			AccessTokenTTL:  15 * time.Minute,
+			SessionTTL:      720 * time.Hour,
+			RefreshDeadline: 4 * time.Second,
 			Password: config.PasswordConfig{
 				Argon2MemoryKiB:     64,
 				Argon2Iterations:    1,
```

- [ ] **Step 7: lint、测试和端到端**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`；`bootstrap` 的 `TestPublicOperationsAreTheContractsPublicOperations`、`TestAPIRoutesAreTheContractsOperations`、`TestBodiesThatBreakTheStructureAnswer400` 覆盖三个新操作。

Run: `make lint-web`
Expected: 通过。

Run: `make e2e`
Expected: P1 的故事全部通过。

- [ ] **Step 8: 提交**

```bash
git add api web/packages/api-client/src/schema.gen.ts server/internal/shared server/internal/modules/identity/adapter/http server/internal/modules/identity/module.go server/internal/bootstrap
```
```bash
git commit -m "feat(M2/P2): login, refreshTokens and logout, with the login and registration limits

Login takes a unit of login_ip and of login_ip_email, or neither;
registration one of register_ip. Refresh and logout run under
auth.refresh_deadline. bootstrap builds one limiter for all six buckets.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run: `make gen-check`
Expected: 没有差异。

**Done when:** 四个生成物与上表相同；`TestLoginLimitsByIPAndByIPWithAddress` 证明被 `login_ip_email` 拒绝的请求不扣 `login_ip`；`TestRefreshAndLogoutTakeNoUnitOfTheModule` 通过；`apitest.Main` 的覆盖核对通过；`make e2e` 中 P1 的故事通过；`make gen-check` 干净。

---

### Task 10: 整程序测试：续期的重复使用与并发、登录的耗时

**Files:**
- Create: `server/internal/bootstrap/sessions_test.go`

**Interfaces:**
- Consumes: 接好线的程序（Task 9）、`pgtest.NewDatabase`、`apitest.Load`、`bootstrap` 测试里已有的 `startApp`、`testConfig`、`registerAccount`、`newRequest`、`send`。

**Tests:**（spec 2.12，真实数据库和真实的接线）
- `TestRefreshReuseRevokesTheSession`：续期一次后再交出第 0 代 → 401，会话 `reuse_detected`（撤销已提交，虽然答的是错误）；之后最新的刷新令牌和访问令牌都是 401。
- `TestConcurrentRefreshesOfOneToken`：同一个令牌同时续期两次，5 轮，每轮恰好一个 200、一个 401，会话 `reuse_detected`。
- `TestLoginTakesAsLongForAnUnknownAddress`：默认的 argon2 参数（m = 19456 KiB、t = 2），已有地址的错误密码和未知地址交替各 15 次，两组中位数相差不超过较大者的四分之一；中位数写进 `t.Logf`。

- [ ] **Step 1: 测试**

`server/internal/bootstrap/sessions_test.go`（新文件）：

```go
package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// sessionApp runs the app on a database of its own and returns its base URL
// and a pool on the same database, for the assertions.
func sessionApp(t *testing.T) (string, *pgxpool.Pool) {
	t.Helper()
	url := pgtest.NewDatabase(t)
	base := startApp(t, testConfig(t, url, false), migrations.FS())
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return base, pool
}

// postTokens posts {"refresh_token": token} to the auth path and returns
// the status and, on 200, the tokens.
func postTokens(t *testing.T, contract *apitest.Contract, base, path, token string) (int, authTokens) {
	t.Helper()
	req := newRequest(t, http.MethodPost, base+path, "", []byte(`{"refresh_token":"`+token+`"}`))
	res, body := send(t, req)
	contract.CheckResponse(t, req, res)
	var tokens authTokens
	if res.StatusCode == http.StatusOK && json.Unmarshal(body, &tokens) != nil {
		t.Fatalf("POST %s = %s, want tokens", path, body)
	}
	return res.StatusCode, tokens
}

// revocation reads the revoke_reason of the only session of email; "" while
// it is live.
func revocation(t *testing.T, pool *pgxpool.Pool, email string) string {
	t.Helper()
	var reason *string
	err := pool.QueryRow(context.Background(), `SELECT s.revoke_reason FROM auth_sessions s JOIN users u ON u.id = s.user_id
		WHERE u.email = $1`, email).Scan(&reason)
	if err != nil {
		t.Fatal(err)
	}
	if reason == nil {
		return ""
	}
	return *reason
}

// A retired refresh token revokes its session, and the revocation commits
// although the answer is an error: afterwards the latest refresh token and
// the latest access token fail too (M2 design 3.5).
func TestRefreshReuseRevokesTheSession(t *testing.T) {
	contract := apitest.Load(t)
	base, pool := sessionApp(t)
	first := registerAccount(t, contract, base, "reuse@example.com")

	status, second := postTokens(t, contract, base, "/api/v0/auth/refresh", first.RefreshToken)
	if status != http.StatusOK {
		t.Fatalf("first refresh = %d, want 200", status)
	}
	reused, _ := postTokens(t, contract, base, "/api/v0/auth/refresh", first.RefreshToken)
	latest, _ := postTokens(t, contract, base, "/api/v0/auth/refresh", second.RefreshToken)
	me, _ := send(t, newRequest(t, http.MethodGet, base+"/api/v0/me", second.AccessToken, nil))

	if reused != http.StatusUnauthorized || revocation(t, pool, "reuse@example.com") != "reuse_detected" {
		t.Errorf("the retired token = %d, session revoked for %q; want 401 and reuse_detected", reused, revocation(t, pool, "reuse@example.com"))
	}
	if latest != http.StatusUnauthorized || me.StatusCode != http.StatusUnauthorized {
		t.Errorf("after the reuse: the latest refresh token = %d, GET /me = %d; want 401 for both", latest, me.StatusCode)
	}
}

// Two refreshes with one token at once: one rotates, the other sees an
// older generation that the session issued and revokes it, whichever way
// the two transactions interleave (M2 design 3.5).
func TestConcurrentRefreshesOfOneToken(t *testing.T) {
	contract := apitest.Load(t)
	base, pool := sessionApp(t)
	for i := range 5 {
		email := fmt.Sprintf("race%d@example.com", i)
		token := registerAccount(t, contract, base, email).RefreshToken
		statuses := make([]int, 2)
		var wg sync.WaitGroup
		for j := range statuses {
			wg.Go(func() { statuses[j], _ = postTokens(t, contract, base, "/api/v0/auth/refresh", token) })
		}
		wg.Wait()

		slices.Sort(statuses)
		if !slices.Equal(statuses, []int{http.StatusOK, http.StatusUnauthorized}) || revocation(t, pool, email) != "reuse_detected" {
			t.Errorf("round %d: statuses %v, session revoked for %q; want 200 and 401, reuse_detected", i, statuses, revocation(t, pool, email))
		}
	}
}

// login posts email and password and returns the status and how long the
// answer took.
func login(t *testing.T, base, email, password string) (int, time.Duration) {
	t.Helper()
	req := newRequest(t, http.MethodPost, base+"/api/v0/auth/login", "", []byte(`{"email":"`+email+`","password":"`+password+`"}`))
	start := time.Now()
	res, _ := send(t, req)
	return res.StatusCode, time.Since(start)
}

func median(d []time.Duration) time.Duration {
	s := slices.Clone(d)
	slices.Sort(s)
	return s[len(s)/2]
}

// With the default argon2id parameters, a login for an unknown address
// takes as long as one with a wrong password for a known one: both verify
// one hash (M2 design 3.9). The two kinds alternate, and their medians
// must lie within a quarter of each other.
func TestLoginTakesAsLongForAnUnknownAddress(t *testing.T) {
	cfg := testConfig(t, pgtest.NewDatabase(t), false)
	cfg.Auth.Password.Argon2MemoryKiB, cfg.Auth.Password.Argon2Iterations = 19456, 2
	base := startApp(t, cfg, migrations.FS())
	registerAccount(t, apitest.Load(t), base, "timing@example.com")

	var known, unknown []time.Duration
	for range 15 {
		statusKnown, k := login(t, base, "timing@example.com", "Wr0ng-password")
		statusUnknown, u := login(t, base, "nobody@example.com", "Wr0ng-password")
		if statusKnown != http.StatusUnauthorized || statusUnknown != http.StatusUnauthorized {
			t.Fatalf("logins = %d, %d; want 401 for both", statusKnown, statusUnknown)
		}
		known, unknown = append(known, k), append(unknown, u)
	}

	mk, mu := median(known), median(unknown)
	t.Logf("median login: known address %v, unknown address %v", mk, mu)
	if diff := max(mk, mu) - min(mk, mu); diff*4 > max(mk, mu) {
		t.Errorf("median login: known address %v, unknown address %v; want them within a quarter of each other", mk, mu)
	}
}
```

Run: `go -C server test -count=1 -v -run 'TestRefreshReuseRevokesTheSession|TestConcurrentRefreshesOfOneToken|TestLoginTakesAsLongForAnUnknownAddress' ./internal/bootstrap/`
Expected: 三个都 `PASS`；`TestLoginTakesAsLongForAnUnknownAddress` 的日志行写出两个中位数（原型中约 15–17 毫秒，相差不到 6%）。

- [ ] **Step 2: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。

- [ ] **Step 3: 提交**

```bash
git add server/internal/bootstrap/sessions_test.go
```
```bash
git commit -m "test(M2/P2): whole-program tests of refresh reuse, concurrent refreshes and login timing

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 三个测试通过；并发续期的 5 轮每轮都是一个 200、一个 401。

---

### Task 11: 端到端：登录与续期的 fixture；A3、A4、A5、A6、A15 的接口版本

**Files:**
- Modify: `e2e/fixtures/auth.ts`、`e2e/fixtures/assert/identity.ts`
- Create: `e2e/stories/identity/a3-sign-in.spec.ts`、`a4-refresh.spec.ts`、`a5-refresh-reuse.spec.ts`、`a6-sign-out.spec.ts`、`a15-sign-in-limits.spec.ts`

**Interfaces:**
- Consumes: P1 的 `api`、`db`、`nerveWith` fixture，`createApi`，`emailFor`、`register`、`password`。
- Produces（spec 2.13）：`login(api, email, headers)`、`refresh(api, refreshToken)`；`SignIn`（替代 `Registration`）、`sessionOf`、`expectRegistered`、`expectSignedIn`、`expectRefreshed`、`expectRevoked`。
- 页面版本随 P4 加入；PAT 的对等验收不适用于这五个故事（M2 设计 2：认证本身的故事发生在拿到任何令牌之前）。

**Tests:**（每个故事一个 `test`，名字以 "A<n> (API):" 开头）
- A3：登录成功（地址大小写和两端空白不同）、`/me` 可读；错误密码和未知地址同一个 401 和 `detail`；缺 `password` 400 `required`；失败不新增行。
- A4：三次续期，每次用上一次的刷新令牌；`refresh_token_expires_at` 不变；数据库代数为 3。
- A5：伪造的旧代 401、会话不变；真实的旧令牌 401、`reuse_detected`；之后的刷新令牌和访问令牌都是 401。
- A6：上一代退出 204、会话不变；当前一代退出 204、`logout`，访问令牌随即 401；再退出一次，不变。
- A15：限流很低的独立 nerve：同一邮箱 401、401、429；换邮箱 401；再换 429；`Retry-After: 60`；不新增行。

- [ ] **Step 1: fixture**

`e2e/fixtures/auth.ts`（对 `fd69736` 的差异）：

```diff
--- a/e2e/fixtures/auth.ts
+++ b/e2e/fixtures/auth.ts
@@ -28,3 +28,26 @@
   }
   return data;
 }
+
+/** Signs email in with the fixture's password and returns the new session's tokens. */
+export async function login(api: Api, email: string, headers: Record<string, string> = {}): Promise<AuthTokens> {
+  const { data, error, response } = await api.POST("/api/v0/auth/login", {
+    body: { email, password },
+    headers,
+  });
+  expect(response.status, `login ${email}: ${JSON.stringify(error)}`).toBe(200);
+  if (!data) {
+    throw new Error(`login ${email} answered 200 without tokens`);
+  }
+  return data;
+}
+
+/** Exchanges refreshToken for the session's next tokens. */
+export async function refresh(api: Api, refreshToken: string): Promise<AuthTokens> {
+  const { data, error, response } = await api.POST("/api/v0/auth/refresh", { body: { refresh_token: refreshToken } });
+  expect(response.status, `refresh: ${JSON.stringify(error)}`).toBe(200);
+  if (!data) {
+    throw new Error("refresh answered 200 without tokens");
+  }
+  return data;
+}
```

`e2e/fixtures/assert/identity.ts`（完整内容）：

```ts
import { createHash } from "node:crypto";

import { expect } from "@playwright/test";

import type { Database } from "../db";

// Database assertions of the identity stories. The page version and the API
// version of a story call the same function (v0 design 8.2).

const refreshTokenPrefix = "nrv_rt_";
const dayMs = 24 * 60 * 60 * 1000;

/** A refresh token's content (M2 design 3.4): nrv_rt_, then session id, generation, secret and tag in base64url. */
interface RefreshTokenParts {
  sessionId: string;
  generation: number;
  secret: Buffer;
}

function parseRefreshToken(token: string): RefreshTokenParts {
  expect(token.startsWith(refreshTokenPrefix), `${token} starts with ${refreshTokenPrefix}`).toBe(true);
  const raw = Buffer.from(token.slice(refreshTokenPrefix.length), "base64url");
  expect(raw.length, "a refresh token holds 68 bytes").toBe(68);
  const id = raw.subarray(0, 16).toString("hex");
  return {
    sessionId: `${id.slice(0, 8)}-${id.slice(8, 12)}-${id.slice(12, 16)}-${id.slice(16, 20)}-${id.slice(20)}`,
    generation: raw.readUInt32BE(16),
    secret: raw.subarray(20, 52),
  };
}

function secretHash(refreshToken: string): Buffer {
  return createHash("sha256").update(parseRefreshToken(refreshToken).secret).digest();
}

/** A session's row, as the session assertions read it. */
export interface SessionRow {
  id: string;
  user_id: string;
  token_hash: Buffer;
  generation: number;
  user_agent: string;
  ip: string;
  expires_at: Date;
  created_at: Date;
  last_refreshed_at: Date | null;
  revoked_at: Date | null;
  revoke_reason: string | null;
}

/** The row of the session that refreshToken belongs to. */
export async function sessionOf(db: Database, refreshToken: string): Promise<SessionRow> {
  const rows = await db.query<SessionRow>(
    `SELECT id, user_id, token_hash, generation, user_agent, ip, expires_at, created_at, last_refreshed_at,
            revoked_at, revoke_reason
       FROM auth_sessions WHERE id = $1`,
    [parseRefreshToken(refreshToken).sessionId]
  );
  expect(rows, "the session of the refresh token").toHaveLength(1);
  return rows[0] as SessionRow;
}

/** What a sign-up or a login sent and got back. */
export interface SignIn {
  /** The address as typed; the account holds it lowercased. */
  email: string;
  refreshToken: string;
  userAgent: string;
  ip: string;
}

/**
 * The session of s.refreshToken is new: generation 0 of the account userId,
 * holding the hash of the token's secret, with the caller's User-Agent and
 * IP, never refreshed, live, ending 30 days after it began.
 */
async function expectNewSession(db: Database, userId: string | undefined, s: SignIn): Promise<void> {
  const session = await sessionOf(db, s.refreshToken);
  expect(session.user_id).toBe(userId);
  expect(parseRefreshToken(s.refreshToken).generation).toBe(0);
  expect(session.generation).toBe(0);
  expect(session.token_hash.equals(secretHash(s.refreshToken))).toBe(true);
  expect(session.user_agent).toBe(s.userAgent);
  expect(session.ip).toBe(s.ip);
  // auth.session_ttl is 720h: the session ends 30 days after it began.
  expect(session.expires_at.getTime() - session.created_at.getTime()).toBe(30 * dayMs);
  expect(session.last_refreshed_at).toBeNull();
  expect(session.revoked_at).toBeNull();
}

/**
 * A1: registration added one account with the address lowercased, an
 * argon2id hash and the display name from the address; its default profile;
 * and its one session, a new one.
 */
export async function expectRegistered(db: Database, r: SignIn): Promise<void> {
  const email = r.email.toLowerCase();
  const users = await db.query<{ id: string; password: string; display_name: string; is_active: boolean }>(
    "SELECT id, password, display_name, is_active FROM users WHERE email = $1",
    [email]
  );
  expect(users).toHaveLength(1);
  const [user] = users;
  expect(user?.password).toMatch(/^\$argon2id\$/);
  expect(user?.display_name).toBe(email.slice(0, email.indexOf("@")));
  expect(user?.is_active).toBe(true);

  const profiles = await db.query(
    `SELECT theme, is_tour_completed, onboarding_step, is_onboarded, last_workspace_id, language, start_of_the_week
       FROM profiles WHERE user_id = $1`,
    [user?.id]
  );
  expect(profiles).toEqual([
    {
      theme: "system",
      is_tour_completed: false,
      onboarding_step: {
        profile_complete: false,
        workspace_create: false,
        workspace_invite: false,
        workspace_join: false,
      },
      is_onboarded: false,
      last_workspace_id: null,
      language: "en",
      start_of_the_week: 0,
    },
  ]);

  expect(await db.query("SELECT id FROM auth_sessions WHERE user_id = $1", [user?.id])).toHaveLength(1);
  await expectNewSession(db, user?.id, r);
}

/** A3: a login added a new session to the account of the address. */
export async function expectSignedIn(db: Database, s: SignIn): Promise<void> {
  const users = await db.query<{ id: string }>("SELECT id FROM users WHERE email = $1", [s.email.toLowerCase()]);
  expect(users).toHaveLength(1);
  await expectNewSession(db, users[0]?.id, s);
}

/**
 * A4: after `refreshes` refreshes, each with the token the last one
 * returned, the session is live at that generation, holds the hash of the
 * latest token's secret, was refreshed, and still ends at sessionEnd, the
 * refresh_token_expires_at of the login (M2 design 3.5).
 */
export async function expectRefreshed(
  db: Database,
  latest: string,
  refreshes: number,
  sessionEnd: string
): Promise<void> {
  const session = await sessionOf(db, latest);
  expect(parseRefreshToken(latest).generation).toBe(refreshes);
  expect(session.generation).toBe(refreshes);
  expect(session.token_hash.equals(secretHash(latest))).toBe(true);
  expect(session.last_refreshed_at).not.toBeNull();
  expect(session.expires_at.getTime()).toBe(new Date(sessionEnd).getTime());
  expect(session.revoked_at).toBeNull();
}

/** A5, A6: the session of refreshToken is revoked, for reason. */
export async function expectRevoked(
  db: Database,
  refreshToken: string,
  reason: "logout" | "reuse_detected"
): Promise<void> {
  const session = await sessionOf(db, refreshToken);
  expect(session.revoked_at).not.toBeNull();
  expect(session.revoke_reason).toBe(reason);
}

/** How many accounts, profiles and sessions there are. */
export interface IdentityCounts {
  users: number;
  profiles: number;
  sessions: number;
}

export async function countIdentity(db: Database): Promise<IdentityCounts> {
  const [counts] = await db.query<{ users: number; profiles: number; sessions: number }>(
    `SELECT (SELECT count(*)::int FROM users) AS users,
            (SELECT count(*)::int FROM profiles) AS profiles,
            (SELECT count(*)::int FROM auth_sessions) AS sessions`
  );
  if (!counts) {
    throw new Error("the counts query returned no row");
  }
  return counts;
}

/** A2, A3, A15: a refused sign-up or login added no account, profile or session. */
export async function expectNothingAdded(db: Database, before: IdentityCounts): Promise<void> {
  expect(await countIdentity(db)).toEqual(before);
}
```

- [ ] **Step 2: 故事**

`e2e/stories/identity/a3-sign-in.spec.ts`（新文件）：

```ts
import { countIdentity, expectNothingAdded, expectSignedIn } from "../../fixtures/assert/identity";
import { emailFor, login, password, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A3, signing in (M2 design 2). The page version, with next_path, joins in M2/P4.

test("A3 (API): a caller signs in; a wrong password and an unknown address answer alike", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo, "Alice");
  await register(api, email);
  const userAgent = "nerve-e2e/A3";

  // The address in another case and with blanks around it still signs in.
  const tokens = await login(api, ` ${email.toUpperCase()} `, { "User-Agent": userAgent });

  expect(tokens.token_type).toBe("Bearer");
  await expectSignedIn(db, { email, refreshToken: tokens.refresh_token, userAgent, ip: "127.0.0.1" });
  const me = await api.GET("/api/v0/me", { headers: { Authorization: `Bearer ${tokens.access_token}` } });
  expect(me.response.status).toBe(200);

  const before = await countIdentity(db);
  const refusals = await Promise.all([
    api.POST("/api/v0/auth/login", { body: { email, password: "Wr0ng-password" } }),
    api.POST("/api/v0/auth/login", { body: { email: emailFor(testInfo, "nobody"), password } }),
  ]);
  for (const { response, error } of refusals) {
    expect(response.status).toBe(401);
    expect(error?.code).toBe("identity.invalid_credentials");
    expect(error?.detail).toBe("The e-mail address or the password is incorrect.");
  }

  // A body without a password breaks the contract: the platform's 400.
  const missing = await api.POST("/api/v0/auth/login", {
    // @ts-expect-error -- the request leaves out a required field on purpose
    body: { email },
  });
  expect(missing.response.status).toBe(400);
  expect(missing.error?.code).toBe("bad_request");
  expect(missing.error?.errors?.map((e) => ({ field: e.field, code: e.code }))).toEqual([
    { field: "password", code: "required" },
  ]);
  await expectNothingAdded(db, before);
});
```

`e2e/stories/identity/a4-refresh.spec.ts`（新文件）：

```ts
import { expectRefreshed } from "../../fixtures/assert/identity";
import { emailFor, login, refresh, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A4, refreshing (M2 design 2). The page version, two tabs sharing the
// refresh, joins in M2/P4. Reusing an old token is A5.

test("A4 (API): each refresh uses the last token; the generation counts up and the session end stays", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  await register(api, email);
  const first = await login(api, email);

  const second = await refresh(api, first.refresh_token);
  const third = await refresh(api, second.refresh_token);
  const fourth = await refresh(api, third.refresh_token);

  // The access token holds only sub, sid and exp in seconds (M2 design 3.4):
  // within one second it can come out the same, so only the refresh token
  // must differ.
  for (const [before, after] of [
    [first, second],
    [second, third],
    [third, fourth],
  ] as const) {
    expect(after.refresh_token).not.toBe(before.refresh_token);
    expect(after.refresh_token_expires_at).toBe(first.refresh_token_expires_at);
  }
  await expectRefreshed(db, fourth.refresh_token, 3, first.refresh_token_expires_at);
  const me = await api.GET("/api/v0/me", { headers: { Authorization: `Bearer ${fourth.access_token}` } });
  expect(me.response.status).toBe(200);
});
```

`e2e/stories/identity/a5-refresh-reuse.spec.ts`（新文件）：

```ts
import { randomBytes } from "node:crypto";

import { expectRevoked, sessionOf } from "../../fixtures/assert/identity";
import { emailFor, refresh, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A5, a reused refresh token (M2 design 2). The page version joins in M2/P4.

/** token with its secret and tag replaced by random bytes: the session and the generation are real, the rest is not. */
function forgedFrom(token: string): string {
  const prefix = "nrv_rt_";
  const raw = Buffer.from(token.slice(prefix.length), "base64url");
  randomBytes(48).copy(raw, 20);
  return prefix + raw.toString("base64url");
}

test("A5 (API): a retired refresh token revokes its session; a forged older generation does not", async ({
  api,
  db,
}, testInfo) => {
  const first = await register(api, emailFor(testInfo));
  const second = await refresh(api, first.refresh_token);
  const refused = async (token: string) => {
    const { response, error } = await api.POST("/api/v0/auth/refresh", { body: { refresh_token: token } });
    expect(response.status).toBe(401);
    expect(error?.code).toBe("identity.refresh_token_invalid");
  };

  // A forged older generation proves nothing: 401, the session goes on.
  const before = await sessionOf(db, second.refresh_token);
  await refused(forgedFrom(first.refresh_token));
  expect(await sessionOf(db, second.refresh_token)).toEqual(before);

  // The real retired token comes back: someone else holds a copy.
  await refused(first.refresh_token);
  await expectRevoked(db, first.refresh_token, "reuse_detected");

  // Every token of the session fails from now on.
  await refused(second.refresh_token);
  const me = await api.GET("/api/v0/me", { headers: { Authorization: `Bearer ${second.access_token}` } });
  expect(me.response.status).toBe(401);
});
```

`e2e/stories/identity/a6-sign-out.spec.ts`（新文件）：

```ts
import { expectRevoked, sessionOf } from "../../fixtures/assert/identity";
import { emailFor, refresh, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A6, signing out (M2 design 2). The page version, with the other tab and
// the switch of accounts, joins in M2/P4.

test("A6 (API): logout ends the session; the previous generation's logout changes nothing", async ({
  api,
  db,
}, testInfo) => {
  const first = await register(api, emailFor(testInfo));
  const second = await refresh(api, first.refresh_token);
  const logout = async (token: string) => {
    const { response } = await api.POST("/api/v0/auth/logout", { body: { refresh_token: token } });
    expect(response.status).toBe(204);
  };

  // The previous generation: 204, and the session goes on (M2 design 3.5).
  const before = await sessionOf(db, second.refresh_token);
  await logout(first.refresh_token);
  expect(await sessionOf(db, second.refresh_token)).toEqual(before);

  // The current one ends the session; its access token fails on the next request.
  await logout(second.refresh_token);
  await expectRevoked(db, second.refresh_token, "logout");
  const me = await api.GET("/api/v0/me", { headers: { Authorization: `Bearer ${second.access_token}` } });
  expect(me.response.status).toBe(401);

  // Once more: the same answer, nothing changes.
  const ended = await sessionOf(db, second.refresh_token);
  await logout(second.refresh_token);
  expect(await sessionOf(db, second.refresh_token)).toEqual(ended);
});
```

`e2e/stories/identity/a15-sign-in-limits.spec.ts`（新文件）：

```ts
import { createApi } from "../../fixtures/api";
import { countIdentity, expectNothingAdded } from "../../fixtures/assert/identity";
import { emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A15, login limits (M2 design 2, 3.10). The page version joins in M2/P4.

test("A15 (API): logins are limited per client IP and address, then per client IP", async ({
  db,
  nerveWith,
}, testInfo) => {
  // A nerve with low login limits: login_ip 3, login_ip_email 2, each
  // regaining one unit a minute.
  const limited = createApi(
    (
      await nerveWith({
        NERVE_RATELIMIT__LOGIN_IP__PER_MINUTE: "1",
        NERVE_RATELIMIT__LOGIN_IP__BURST: "3",
        NERVE_RATELIMIT__LOGIN_IP_EMAIL__PER_MINUTE: "1",
        NERVE_RATELIMIT__LOGIN_IP_EMAIL__BURST: "2",
      })
    ).baseURL
  );
  const email = emailFor(testInfo);
  await register(limited, email);
  const before = await countIdentity(db);
  const attempt = async (address: string, want: number) => {
    const { response, error } = await limited.POST("/api/v0/auth/login", {
      body: { email: address, password: "Wr0ng-password" },
    });
    expect(response.status, address).toBe(want);
    if (want === 429) {
      expect(error?.code).toBe("rate_limited");
      expect(response.headers.get("Retry-After")).toBe("60");
    }
  };

  // One address fails up to login_ip_email's burst, then is refused.
  await attempt(email, 401);
  await attempt(email, 401);
  await attempt(email, 429);
  // That refusal took nothing from login_ip: another address gets its last
  // unit, and then login_ip refuses every address.
  await attempt(emailFor(testInfo, "other"), 401);
  await attempt(emailFor(testInfo, "third"), 429);

  await expectNothingAdded(db, before);
});
```

- [ ] **Step 3: 检查和端到端**

Run: `make lint-web`
Expected: 通过（`e2e` 的 oxlint 上限 0；循环里不 `await`，A4 逐次写出三次续期）。

Run: `make knip`
Expected: 通过（`Registration` 已删除，新的导出都有使用者）。

Run: `make e2e`
Expected: S1–S4、A1、A2、A3、A4、A5、A6、A15 共 12 个全部通过。

Run: `pnpm -C e2e exec playwright test stories/identity --repeat-each=5 --workers=1`（用 `make e2e` 刚构建的 `bin/nerve`）
Expected: 35 个全部通过（`emailFor` 带 `repeatEachIndex`，重跑不冲突）。

- [ ] **Step 4: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。

- [ ] **Step 5: 提交**

```bash
git add e2e
```
```bash
git commit -m "test(M2/P2): API versions of A3, A4, A5, A6 and A15; login and refresh in the fixtures

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 五个新故事通过，P1 的故事仍然通过；`--repeat-each=5` 通过；`make lint-web`、`make knip` 通过。

---

### Task 12: 上级文档、差异清单、README 与交接

**Files:**
- Modify: `docs/v0/v0-design.md`、`docs/v0/M0-foundation/M0-design.md`、`docs/v0/plane-diff.md`、`README.md`
- Modify: `docs/v0/M2-auth/handoffs/M0-P2-platform-notes.md`、`M0-P5-frontend-api-notes.md`、`M0-P6-e2e-notes.md`

**Interfaces:**
- M2 设计 3.20 中 P2 的 7 行、8.7 中 P2 的 1 行（spec 2.14）；交接的处理结果（spec 第 7 节）。三份交接的状态仍为 `open`。

- [ ] **Step 1: 总体设计**（3.5 `rate_limited`；3.6 限流；4.1 绝对期限；4.2 退出和重复使用检测；6.4 固定链和按路由的中间件）

`docs/v0/v0-design.md`（对 `fd69736` 的差异）：

````diff
--- a/docs/v0/v0-design.md
+++ b/docs/v0/v0-design.md
@@ -197,14 +197,18 @@
   ```
 - 看不到的资源返回 **404**，不泄露它是否存在；能看到但没权限执行操作，返回 **403**。
 - `title` 固定为 HTTP 状态短语，即 Go 的 `http.StatusText(status)`（不带 `type` 时符合 RFC 9457 的语义），具体说明放在 `detail`，程序按 `code` 分支。
-- 平台自己的错误码不带模块前缀：`bad_request`（400）、`unauthorized`（401）、`not_found`（404）、`payload_too_large`（413）、`validation_failed`（422）、`internal_error`（500）、`not_ready`（503，只用于 `/readyz`）、`server_busy`（503，带 `Retry-After`）；模块的错误码带模块前缀，例如 `identity.email_taken`（M2 设计 3.11）。
+- 平台自己的错误码不带模块前缀：`bad_request`（400）、`unauthorized`（401）、`not_found`（404）、`payload_too_large`（413）、`validation_failed`（422）、`rate_limited`（429，带 `Retry-After`）、`internal_error`（500）、`not_ready`（503，只用于 `/readyz`）、`server_busy`（503，带 `Retry-After`）；模块的错误码带模块前缀，例如 `identity.email_taken`（M2 设计 3.11）。
 - **结构在接口边界，取值在领域**（M2 设计 3.11）：请求体不是合法 JSON、有未声明的字段、不可为空的字段传了 `null`、缺少必填字段、生成为 Go 类型的格式（`date-time`、`uuid`）写错，一律 400 `bad_request`，`errors` 一次列出全部问题；长度、其余格式、枚举、取值范围和跨字段的规则由领域层校验，一次返回 422 `validation_failed`。
 - `errors` 的每一项是 `{field, code, message}`：`field` 是 JSON 路径，`code` 取自一个封闭的集合（`required`、`invalid_format`、`too_short`、`too_long`、`out_of_range`、`not_allowed`、`weak_password`、`common_password`、`must_be_future`、`contains_url`），前端按 `code` 显示文案。
 - **错误码写进接口描述**：每个操作用扩展字段 `x-problem-codes` 列出它可能返回的码；所有操作都可能返回的平台码只写在 `api/openapi.yaml` 的顶层，声明了 `bearer` 的操作另外隐含 `unauthorized`。`apitest` 核对码的写法、测试中返回的码都已声明、每个声明的码都有测试返回过（M2 设计 3.11）。
 - 请求 ID 只出现在 `X-Request-Id` 响应头中，不放进响应体。
 
 ### 3.6 其他
-- **限流**：按令牌计数，在进程内实现。登录接口单独按"IP + 邮箱"限流。具体数值在 M2 确定，默认参考 Plane。
+- **限流**：在进程内实现，按键的令牌桶，每个桶有速率和突发两个配置项（`ratelimit.<桶>.per_minute`、`burst`）。超出时 429 `rate_limited`，带 `Retry-After`（秒，向上取整）（M2 设计 3.10）：
+  - 公开操作按客户端 IP 计数（`anonymous`，每分钟 600、突发 100），其余操作按凭证计数（`authenticated`，每分钟 1200、突发 200）；
+  - 登录另按 IP（`login_ip`，30、10）和"IP + 邮箱"（`login_ip_email`，10、5）计数，两个桶全扣或全不扣；注册另按 IP 计数（`register_ip`，10、5）；修改密码按账户计数（`password_user`，M2/P3）；
+  - **认证之前的失败闸门**：非公开操作带了令牌时，先预留本 IP 的一个单位（`auth_failure`，60、60），闸门已空就直接 429，不再验证令牌；认证失败时单位留下，成功、只是过期的访问令牌、内部错误时退回（M2 设计 3.6）；
+  - 客户端 IP 取连接的对端，前面有反向代理时配置 `server.trusted_proxies`；IPv6 按前缀计数（`ratelimit.ipv6_prefix_len`，默认 64）。
 - **关联对象只返回 ID**：以后按需增加 `?expand=`。
 - **v0 不做乐观锁**：PATCH 只改传入的字段，同一字段并发修改时以后写入的为准。
 
@@ -218,13 +222,13 @@
 | 令牌 | 获取方式 | 形式 | 有效期 |
 |---|---|---|---|
 | 访问令牌 | 登录，或用刷新令牌换取 | JWT（Ed25519 签名），只包含用户 id、会话 id 和过期时间，**不含任何权限信息** | 15 分钟 |
-| 刷新令牌 | 注册、登录时一并下发 | `nrv_rt_` 加 68 字节的 base64url：会话 id、代数、32 字节的随机密文和 16 字节的 HMAC 标签，MAC 密钥从签名密钥派生；数据库只存当前一代密文的哈希（`auth_sessions`，M2 设计 3.4） | 30 天；每次使用后换新，并检测旧令牌是否被重复使用 |
+| 刷新令牌 | 注册、登录时一并下发 | `nrv_rt_` 加 68 字节的 base64url：会话 id、代数、32 字节的随机密文和 16 字节的 HMAC 标签，MAC 密钥从签名密钥派生；数据库只存当前一代密文的哈希（`auth_sessions`，M2 设计 3.4） | 从登录起 30 天，续期不延长（绝对期限，M2 设计 3.5）；每次使用后换新，并检测旧令牌是否被重复使用 |
 | 个人访问令牌（PAT） | 在设置页生成 | `nrv_pat_` 前缀的随机字符串，数据库只存哈希（`api_tokens`） | 由用户设定；可随时撤销 |
 
 ### 4.2 规则
 - **权限不放进令牌**：每个请求都从数据库读取成员关系和角色。所以移出项目、调整角色是立即生效的。
-- **会话撤销**：退出登录、修改密码、账户停用时，吊销该账户的刷新令牌；账户停用时，同时让该账户的所有会话失效。
-- **重复使用检测**：一旦发现某个已经换过新的刷新令牌又被使用，就作废这次登录派生出的所有令牌。
+- **会话撤销**：退出登录只结束当前这一处登录，同一账户的其他会话不受影响（M2 设计 11.1）；修改密码、账户停用时，吊销该账户的刷新令牌；账户停用时，同时让该账户的所有会话失效。
+- **重复使用检测**：一旦发现某个已经换过新的刷新令牌又被使用，就作废这次登录派生出的所有令牌。只有交出的旧令牌确是这个会话签发过的（它的 MAC 标签成立）才作废；会话 id 和代数对、密文和标签是伪造的旧令牌得到 401，会话不受影响（M2 设计 3.5）。
 - **密码哈希**：用 argon2id。
 - **登录方式**：v0 只有邮箱加密码。是否开放注册由配置 `auth.signup_enabled` 控制：prod 默认关闭，dev、test 默认开放；关闭时注册先答"注册已关闭"，不查邮箱。prod 的第一个账户由服务器管理员用 `nerve users create` 创建（M2/P3 加入；在那之前用 `NERVE_AUTH__SIGNUP_ENABLED=true` 临时打开注册）（M2 设计决策点 2）。被邀请的邮箱始终可以注册。
 - **忘记密码**：没有邮件服务，由服务器管理员通过命令行重置：`nerve users reset-password --email <email>`。
@@ -410,17 +414,18 @@
 
 ### 6.4 请求处理流程
 ```
-请求 → 请求 ID → 异常恢复 → 访问日志                            （固定链，httpserver.NewServer）
+请求 → 请求 ID → 异常恢复 → 访问日志 → 安全响应头              （固定链，httpserver.NewServer）
      → 路由匹配 → 生成的代码绑定路径参数和查询参数（格式错误 → 400）
-     → 请求元信息（客户端 IP、UA）→ 请求期限 → 请求体上限           （按路由，httpserver.API）
-     → 认证（识别 JWT 或 PAT，得到 Actor；公开操作不看令牌）→ 限流
+     → 请求元信息（客户端 IP 和限流用的 IP 键、UA）→ 请求期限 → 请求体上限（按路由，httpserver.API）
+     → 失败闸门（非公开操作带了令牌时，先预留本 IP 的一个单位，已空 → 429）
+     → 认证（识别 JWT 或 PAT，得到 Actor；公开操作不看令牌）→ 限流（有凭证按凭证，没有按 IP）
      → 请求体结构检查（不合契约 → 400）→ 生成的代码解码 JSON 请求体
      → handler（只做类型转换）
      → 用例：TxManager.WithinTx { 权限 → 领域校验（不合规 → 422）→ 业务规则 → 写数据 → 发布领域事件 } 提交
      → 响应；或者 error → APIErrors → problem+json
 ```
-- **请求 ID → 异常恢复 → 访问日志**这三个平台中间件固定在 `httpserver.NewServer` 内部，不可漏掉或调换。
-- **按路由的中间件**由 `httpserver.API.Middlewares` 按上图的顺序交给每个模块的生成代码，只作用于 `/api/v0` 的操作，不作用于健康检查和前端页面（M2 设计 3.6）。认证默认拒绝：除了模块声明为公开的操作，没有有效令牌一律 401。限流由 M2/P2 加入，接口调用日志由 M8 挂在限流之后。
+- **请求 ID → 异常恢复 → 访问日志 → 安全响应头**这四个平台中间件固定在 `httpserver.NewServer` 内部，不可漏掉或调换。安全响应头（`X-Content-Type-Options: nosniff`、`Referrer-Policy: same-origin`、`X-Frame-Options: DENY`）加在每个响应上，包括 panic 之后的 500；CSP 只加在页面上，由 `webui` 负责（M2 设计 8.3）。
+- **按路由的中间件**由 `httpserver.API.Middlewares` 按上图的顺序交给每个模块的生成代码，只作用于 `/api/v0` 的操作，不作用于健康检查和前端页面（M2 设计 3.6）。认证默认拒绝：除了模块声明为公开的操作，没有有效令牌一律 401。限流的桶见 3.6；接口调用日志由 M8 挂在限流之后。
 - **参数先于这些中间件绑定**：生成的代码在它们之前绑定路径参数和查询参数，参数格式错误的请求在认证之前就得到 400；请求体在它们之后才解码，没有通过认证的请求不会被解析请求体。
 - **每个写操作对应一个事务。** 业务数据、操作动态、历史版本、投递给 River 的任务，要么一起成功，要么一起回滚。
 - **事务的传递**：`TxManager` 端口声明在 `internal/shared`（由使用方定义接口），`platform/postgres` 提供实现，`bootstrap` 负责接线；仓储从 `ctx` 中取出当前事务。用例代码不接触任何数据库类型。
````

- [ ] **Step 2: M0 设计**（3.3 固定链的第四个中间件、按路由的失败闸门和限流、`rate_limited`）

`docs/v0/M0-foundation/M0-design.md`（对 `fd69736` 的差异）：

```diff
--- a/docs/v0/M0-foundation/M0-design.md
+++ b/docs/v0/M0-foundation/M0-design.md
@@ -174,9 +174,10 @@
   1. 请求 ID：读取 `X-Request-Id`，只有是 1–128 个 `[A-Za-z0-9._:-]` 字符时才采用，否则生成 UUIDv7，并写回响应头。
   2. 异常恢复：捕获 panic，返回 500 problem+json，并记录日志。
   3. 访问日志：用 slog 记录方法、路径、状态码、耗时、请求 ID。`/healthz`、`/readyz` 的记录是 DEBUG 级别，其余是 INFO（M2/P1）。
-- **按路由的中间件**（M2/P1，M2 设计 3.6）：`/api/v0` 的操作另有一串中间件，由 `httpserver.API.Middlewares` 交给每个模块的生成代码，在访问日志之后、按这个顺序：请求元信息（客户端 IP、UA）→ 请求期限（`server.request_timeout`）→ 请求体上限（`server.max_body_bytes`，超过是 413）→ 默认拒绝的认证（模块声明为公开的操作之外，没有有效令牌一律 401）→ 请求体结构检查（不合契约是 400）。生成代码在它们之前绑定路径参数和查询参数，在它们之后解码请求体。
+  4. 安全响应头（M2/P2，M2 设计 8.3）：每个响应都带 `X-Content-Type-Options: nosniff`、`Referrer-Policy: same-origin`、`X-Frame-Options: DENY`；异常恢复清掉已设的响应头之后重新设上它们，panic 的 500 同样带着。
+- **按路由的中间件**（M2/P1、P2，M2 设计 3.6）：`/api/v0` 的操作另有一串中间件，由 `httpserver.API.Middlewares` 交给每个模块的生成代码，在固定链之后、按这个顺序：请求元信息（客户端 IP 和限流用的 IP 键、UA）→ 请求期限（`server.request_timeout`）→ 请求体上限（`server.max_body_bytes`，超过是 413）→ 失败闸门和默认拒绝的认证（模块声明为公开的操作之外，没有有效令牌一律 401；带了令牌时先预留本 IP 的一个 `auth_failure` 单位，已空是 429）→ 限流（有凭证按凭证计数，没有按 IP 计数，超出是 429）→ 请求体结构检查（不合契约是 400）。生成代码在它们之前绑定路径参数和查询参数，在它们之后解码请求体。
 - **`/readyz`**：按顺序执行各项检查，全部共用一个 2 秒的超时预算，遇到第一个失败就停止，返回通用的 `detail`（`<检查名> is not ready`）；具体错误只写进日志，不返回给客户端。
-- **problem+json**：M0 定义统一的写出函数和 `Problem` 结构（与 `api/common.yaml` 中的定义一致），包含可选的 `detail`。平台自己的错误码不带模块前缀（`not_found`、`bad_request`、`internal_error`、`not_ready`）；M2/P1 加入 `unauthorized`、`payload_too_large`、`validation_failed`、`server_busy` 和领域错误码的体系（M2 设计 3.11）。
+- **problem+json**：M0 定义统一的写出函数和 `Problem` 结构（与 `api/common.yaml` 中的定义一致），包含可选的 `detail`。平台自己的错误码不带模块前缀（`not_found`、`bad_request`、`internal_error`、`not_ready`）；M2/P1 加入 `unauthorized`、`payload_too_large`、`validation_failed`、`server_busy` 和领域错误码的体系（M2 设计 3.11），M2/P2 加入 `rate_limited`（429，带 `Retry-After`）。
 - **生成代码的错误出口**（M0/P3，M2/P1 修改）：oapi-codegen 生成的代码在参数绑定、请求体解码、handler 返回错误三处默认输出纯文本；`httpserver.APIErrors` 把三处都接成 problem+json：参数绑定失败 → `BadRequest`（400 `bad_request`，`detail` 是失败原因）；请求体解码失败 → `BodyError`（400 `bad_request`，`detail` 是通用的一句话，不带出 Go 的类型名；超过请求体上限是 413 `payload_too_large`）；handler 出错或响应写出失败 → `Write`：满足 `httpserver.ProblemError` 的错误映射为它的状态、码、`detail` 和字段，其余是 500 `internal_error`（不带 `detail`，原因只进日志）；客户端断开（`context.Canceled`）不算 500；响应已经开始时改为记录日志并中断连接，不追加 problem。
 - **非规范的 `/api/` 路径**：例如 `/api/v0//instance`、`/api/v0/./instance`、`/api`，Go 的 `ServeMux` 会先返回 307 跳转到规范路径，而不是直接落进平台的 404 兜底。M0/P3 评审后接受这个行为，不作特殊处理。
 - **生命周期**：收到 SIGINT 或 SIGTERM 后停止接收新请求，在 `server.shutdown_timeout` 时间内处理完已有请求，然后退出。
```

- [ ] **Step 3: 差异清单**（四：M2 设计 4.6 中标 P2 的三行）

`docs/v0/plane-diff.md`（对 `fd69736` 的差异）：

```diff
--- a/docs/v0/plane-diff.md
+++ b/docs/v0/plane-diff.md
@@ -153,3 +153,6 @@
 | 注册默认是否开放 | `ENABLE_SIGNUP` 默认开放 | prod 默认关闭，dev、test 默认开放；关闭时先答"注册已关闭"，不查邮箱；第一个账户用 `nerve users create`（M2/P3 加入）（M2 设计决策点 2） |
 | 请求中的未知字段和不合法的 `null` | DRF 的序列化器忽略未知字段 | 按契约返回 400（M2 设计 3.11） |
 | 密码哈希过载 | 无并发上限 | 最多 4 个同时计算，等待 2 秒仍拿不到名额时 503 `server_busy`，带 `Retry-After: 1`（M2 设计 3.8） |
+| 登录时邮箱不存在 | 返回 `USER_DOES_NOT_EXIST` | 与密码错误相同的 401 `identity.invalid_credentials`，耗时也相同：对一个启动时生成的假哈希做一次同样参数的校验（M2 设计 3.9） |
+| 会话的期限 | Django 会话，从登录起固定 7 天（`SESSION_COOKIE_AGE`），请求不延长 | 访问令牌 15 分钟；会话从登录起 30 天，续期不延长；刷新令牌每次使用后换新，并检测重复使用；退出只结束当前这一处登录（M2 设计 3.5） |
+| 限流 | 认证接口合计每 IP 10/min，匿名 30/min，API Key 60/min；`/api/v1` 的响应带 `X-RateLimit-Remaining`、`X-RateLimit-Reset`（`plane/apps/api/plane/api/views/base.py:120-126`） | 进程内的令牌桶，每个桶有速率和突发：匿名按 IP、已认证按凭证、登录按 IP 和"IP + 邮箱"、注册按 IP，认证之前另有按 IP 的失败闸门；超出时 429 带 `Retry-After`，不加 `X-RateLimit-*`（M2 设计 3.10）。修改密码按账户的桶随 M2/P3 加入 |
```

- [ ] **Step 4: README**（部署：反向代理与 `server.trusted_proxies`）

`README.md`（对 `fd69736` 的差异）：

````diff
--- a/README.md
+++ b/README.md
@@ -104,6 +104,7 @@
   ```
 
   访问令牌由它签名，刷新令牌的 MAC 密钥也从它派生。换密钥（替换文件后重启）的后果：已签发的访问令牌验签失败，客户端续期一次即可；换钥之前的旧代刷新令牌不再能被认出是重复使用，所以刷新令牌正被盗用时换钥，受害者只是被登出（恢复靠修改密码或 `nerve users reset-password`）；同一出口 IP 后面的大量标签页同时续期，会短暂得到 429。所以在低峰时换。
+- **反向代理**：nerve 前面有反向代理（例如 Caddy）时，把代理的地址写进 `server.trusted_proxies`（CIDR 列表；环境变量用逗号分隔，例如 `NERVE_SERVER__TRUSTED_PROXIES=10.0.0.0/8,fd00::/8`）。只有连接的对端在这个列表中时，nerve 才从 `X-Forwarded-For` 自右向左取第一个不可信的地址作为客户端 IP。不配置时，所有请求都算作代理的地址：按 IP 的限流（匿名请求、登录、注册、认证失败）让所有人共用一份额度；nerve 第一次收到不可信对端带来的 `X-Forwarded-For` 时记一条 WARN 提醒。IPv6 客户端按前缀计数（`ratelimit.ipv6_prefix_len`，默认 64）。
 - **注册**：prod 默认关闭注册（`auth.signup_enabled: false`）。第一个账户用 `nerve users create` 创建（M2/P3 加入；在那之前临时设 `NERVE_AUTH__SIGNUP_ENABLED=true`）。
 - **迁移**：prod 默认不在启动时迁移（`database.auto_migrate: false`），先执行 `nerve migrate up`，再 `nerve serve`。用单独的数据库角色执行迁移时，运行服务的角色除了读写业务表，还要能读 `goose_db_version`（`GRANT SELECT ON goose_db_version TO <服务的角色>`）：`/readyz` 靠它判断迁移是否已完成。
 
````

- [ ] **Step 5: 交接**

`docs/v0/M2-auth/handoffs/M0-P2-platform-notes.md`（对 `fd69736` 的差异）：

```diff
--- a/docs/v0/M2-auth/handoffs/M0-P2-platform-notes.md
+++ b/docs/v0/M2-auth/handoffs/M0-P2-platform-notes.md
@@ -53,3 +53,11 @@
 仍未处理，状态保持 `open`：第 2 条的限流（M2/P2）和接口调用日志（M8，挂在限流之后）；第 5 条的 River、停机顺序、连接池关闭的时限和 River 的迁移（M2/P3）。
 
 来源：[M2/P1 spec](../specs/P1-platform-core.md) 第 7 节。
+
+## 处理结果（M2/P2）
+
+2. **按路由挂载**（限流完成）：`httpserver.API.Middlewares` 在认证之前加上按 IP 的失败闸门（`auth_failure`：先预留一个单位，只有认证失败才留下），在认证之后、请求体结构检查之前加上限流（有凭证按凭证计数的 `authenticated`，没有按 IP 计数的 `anonymous`）。桶由 `platform/ratelimit` 提供，`httpserver` 通过自己声明的 `Limiter` 接口使用它，`bootstrap` 接上；登录、注册的桶在 `identity` 的 HTTP 适配器里。限流没有改动 `NewServer`（固定链加上的安全响应头属于 M0-P5 交接，M2 设计 8.3）。
+
+仍未处理，状态保持 `open`：第 2 条的接口调用日志（M8，挂在限流之后）；第 5 条的 River、停机顺序、连接池关闭的时限和 River 的迁移（M2/P3）。
+
+来源：[M2/P2 spec](../specs/P2-sessions.md) 第 7 节。
```

`docs/v0/M2-auth/handoffs/M0-P5-frontend-api-notes.md`（对 `fd69736` 的差异）：

```diff
--- a/docs/v0/M2-auth/handoffs/M0-P5-frontend-api-notes.md
+++ b/docs/v0/M2-auth/handoffs/M0-P5-frontend-api-notes.md
@@ -19,3 +19,13 @@
 - 加 `X-Content-Type-Options: nosniff` 时，和 CSP 的决定放在一起做——大概率是同一层中间件，不要分两次改动路由或中间件链。
 
 来源：[M0/P5 评审记录](../../M0-foundation/reviews/P5-web-import-review.md)。
+
+## 处理结果（M2/P2）
+
+- **安全响应头**（完成）：`httpserver.NewServer` 的固定链加上第四个中间件，每个响应（接口、页面、健康检查、panic 之后的 500）都带 `X-Content-Type-Options: nosniff`、`Referrer-Policy: same-origin`、`X-Frame-Options: DENY`。
+- **与 CSP 分在两层**（有意的安排，M2 设计 8.3）：这三个响应头对接口的响应同样有意义，所以放在固定链上；CSP 只对页面有意义，而且要用 `index.html` 里内联脚本的哈希，只有 `webui` 知道这些脚本，所以由 M2/P4 在 `webui` 中加入。交接原文"和 CSP 放在同一层"的本意是两者都由服务端在 M2 加入，这一点照做；这一项在 P4 加入 CSP 时正式关闭。
+- **认证接口在 `/api/v0/` 下**（接口完成）：`register`、`login`、`refreshTokens`、`logout` 都在 `/api/v0/auth/` 下。
+
+仍未处理，状态保持 `open`：CSP（M2/P4）；前端改调 `/api/v0/instance` 和认证接口，以及同源部署的核对（M2/P4）。
+
+来源：[M2/P2 spec](../specs/P2-sessions.md) 第 7 节。
```

`docs/v0/M2-auth/handoffs/M0-P6-e2e-notes.md`（对 `fd69736` 的差异）：

```diff
--- a/docs/v0/M2-auth/handoffs/M0-P6-e2e-notes.md
+++ b/docs/v0/M2-auth/handoffs/M0-P6-e2e-notes.md
@@ -60,3 +60,12 @@
 仍未处理，状态保持 `open`：PAT 对等验收，认证 fixture 的登录、PAT 和页面的登录状态（M2/P2–P4）；S3 的 `signup_enabled`（M2/P3）；S2 的断言（M2/P4）；River 停机与 fixture 的预算（M2/P3）；fixture 写法的延伸（M4、M5、M8）。
 
 来源：[M2/P1 spec](../specs/P1-platform-core.md) 2.16。
+
+## 处理结果（M2/P2）
+
+- **认证 fixture 的登录**（完成）：`e2e/fixtures/auth.ts` 加上 `login` 和 `refresh`；A3、A4、A5、A6、A15 的接口版本调用 `e2e/fixtures/assert/identity.ts` 的断言函数（`expectSignedIn`、`expectRefreshed`、`expectRevoked`、`sessionOf`）；A15 用 `nerveWith` 另起一个限流很低的 nerve。
+- **新等待的期限**：P2 没有新增等待；A15 另起的 nerve 沿用 `nerveWith` 的预算。
+
+仍未处理，状态保持 `open`：PAT 对等验收和认证 fixture 的 PAT（M2/P3）；页面的登录状态（M2/P4）；S3 的 `signup_enabled`（M2/P3）；S2 的断言（M2/P4）；River 停机与 fixture 的预算（M2/P3）；fixture 写法的延伸（M4、M5、M8）。
+
+来源：[M2/P2 spec](../specs/P2-sessions.md) 第 7 节。
```

- [ ] **Step 6: 检查**

Run: `make lint-web`
Expected: 通过（关键词守卫扫描改过的文档）。

Run: `grep -n "按令牌计数\|访问日志                            （固定链" docs/v0/v0-design.md`
Expected: 没有输出（3.6 和 6.4 的旧写法都已改掉）。

Run: `make test`
Expected: 全部 `ok`（文档不影响测试；确认工作区干净）。

- [ ] **Step 7: 提交**

```bash
git add docs/v0/v0-design.md docs/v0/M0-foundation/M0-design.md docs/v0/plane-diff.md README.md docs/v0/M2-auth/handoffs
```
```bash
git commit -m "docs(M2/P2): sync the design documents, plane-diff and README; record the handoff results

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** M2 设计 3.20 中 P2 的 7 行和 8.7 中 P2 的一行都已同步；M0-P2、M0-P5、M0-P6 追加了"处理结果（M2/P2）"并保持 `open`。
