# M2/P2 登录与会话：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M2/P2 `sessions` |
| 日期 | 2026-09-26 |
| 状态 | 进行中 |
| 上级文档 | [M2 设计文档](../M2-design.md) 第 2（A3–A6、A15）、3.4–3.6、3.9、3.10、3.20（P2 各行）、4.6、6.2–6.5、8.3、8.4、8.7、9.5、11.1、12（P2）、13.1 节；[v0 总体设计](../../v0-design.md) 3.5、3.6、4.1、4.2、6.4 节；[M0 设计](../../M0-foundation/M0-design.md) 3.3 节 |
| 前置交接 | [M0-P2-platform-notes](../handoffs/M0-P2-platform-notes.md)、[M0-P5-frontend-api-notes](../handoffs/M0-P5-frontend-api-notes.md)、[M0-P6-e2e-notes](../handoffs/M0-P6-e2e-notes.md)；[P1 评审记录](../reviews/P1-platform-core-review.md) 第 6 节中交给 P2 的五项（本 Phase 的处理见第 7 节） |
| 计划 | [P2 plan](../plans/P2-sessions.md) |

本 spec 只写 M2 设计交给 P2 决定的东西：名字、签名、SQL、配置项、测试名，以及原型证明了什么。规则本身以 M2 设计为准，这里引用节号，不重述。

## 1. 目标

按 M2 设计 12 节 P2 的目标：登录、续期、退出可用；刷新令牌的轮换和重复使用检测、会话的绝对期限、限流、安全响应头全部到位。具体是：

- 平台：`platform/ratelimit`（令牌桶、`AllowAll`、`Reserve`）；固定链上的安全响应头；客户端 IP 与 `server.trusted_proxies`；认证之前的失败闸门和按路由的限流；`rate_limited` 和契约中的 `Retry-After`、`WWW-Authenticate` 响应头；
- `identity`：刷新令牌的解析和续期判定表（领域）；`login`、`refreshTokens`、`logout` 三个用例和接口；账户行锁的查询；登录、注册的桶；续期、退出的 4 秒期限；
- 配置：3.10 的六个桶、`ipv6_prefix_len`、`trusted_proxies`、`refresh_deadline`，以及 P1 评审交来的配置加固；
- 测试：续期的并发与重复使用、登录耗时两个整程序测试；A3、A4、A5、A6、A15 的接口版本；
- 3.20 中 P2 的各行、8.7 中 P2 的 README 内容、交接的处理记录。

## 2. 交付物

### 2.1 文件总览

路径相对于仓库根目录。"生成"表示由命令生成并提交，不手改。"Task"是 plan 中负责它的任务；两个 Task 号表示先写过渡版本、后一个 Task 写成最终版本。

| 路径 | 内容 | Task |
|---|---|---|
| `server/internal/platform/config/`、`server/configs/config.yaml`、`config.test.yaml`、`embed_test.go` | P2 的配置项、校验；YAML 空值和越界数的拒绝；列表型环境变量 | 1 |
| `server/internal/platform/ratelimit/` | 令牌桶（只用标准库） | 2 |
| `server/internal/platform/httpserver/{middleware,server}.go`、`server/internal/bootstrap/headers_test.go` | 固定链的第四个中间件：安全响应头 | 3 |
| `server/internal/platform/httpserver/clientip.go`、`clientip_test.go` | 客户端 IP、可信代理、IP 键 | 4、5 |
| `server/internal/platform/httpserver/{api,limit,problem}.go`、`api_test.go`、`contract_test.go` | 失败闸门、限流中间件、`NewAPI` 检查依赖、`rate_limited` | 4、5 |
| `server/internal/platform/httpserver/apitest/problems.go`、`problems_test.go` | P1 评审交来的测试卫生 | 5 |
| `api/openapi.yaml`、`api/common.yaml`、`api/modules/{identity,instance}.yaml` | 顶层 `rate_limited`；problem 响应的两个响应头；三个新操作 | 5、9 |
| `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`、`*/adapter/http/gen/*.gen.go`、`apigen/components.gen.go` | 生成 | 5、9 |
| `server/internal/modules/identity/adapter/authn/`、`app/authenticate_test.go` | `ExpiredCredential` | 5 |
| `server/internal/modules/instance/adapter/http/handler_test.go` | 新的 `APIConfig` | 5 |
| `server/internal/modules/identity/domain/{session,errors}.go`、`session_test.go` | `ParseRefreshToken`、`JudgeRefresh`、三个模块错误 | 6 |
| `server/internal/modules/identity/app/` | 端口、`Issuance`、`Login`、`Refresh`、`Logout` | 7 |
| `server/internal/modules/identity/adapter/{argon2,signing}/` | `Verify` | 7 |
| `server/internal/modules/identity/adapter/postgres/`（含 `queries/`） | 登录和续期的查询、仓储方法 | 8 |
| `server/internal/modules/identity/adapter/postgres/gen/` | 生成 | 8 |
| `server/internal/modules/identity/adapter/http/`、`module.go` | 三个操作、模块的桶、续期和退出的期限 | 7、9 |
| `server/internal/shared/error.go`、`error_test.go`、`server/internal/bootstrap/errors_test.go` | `shared.RateLimited` | 9 |
| `server/internal/bootstrap/app.go`、`app_test.go` | 接线：限流器、六个桶、可信代理 | 4、5、9 |
| `server/internal/bootstrap/sessions_test.go` | 两个整程序测试（续期、登录耗时） | 10 |
| `e2e/fixtures/auth.ts`、`e2e/fixtures/assert/identity.ts`、`e2e/stories/identity/a{3,4,5,6,15}-*.spec.ts` | 登录、续期的 fixture；五个故事的接口版本 | 11 |
| `docs/…`、`README.md` | 3.20 的 P2 各行、8.7 的 P2 内容、交接的处理记录 | 12 |

### 2.2 依赖

P2 不加任何 Go 模块和 npm 包：限流器只用标准库（M2 设计 3.10），argon2 的校验用 P1 已有的 `golang.org/x/crypto` v0.57.0，MAC 的比较用标准库的 `crypto/hmac`。`server/go.mod`、`server/tools/go.mod` 不变，仍是 `go 1.27` / `toolchain go1.27.1`。M2 设计 6.6 中 P2 没有新依赖。

### 2.3 配置（M2 设计 6.5 中 P2 的部分；P1 评审第 6 节"配置加固"）

新配置项（`server/configs/config.yaml` 写默认值）：

| 键 | 默认值 | 校验 |
|---|---|---|
| `server.trusted_proxies` | `[]` | CIDR（解码时由 `netip.ParsePrefix` 检查）；不能是 IPv4 映射的 IPv6 前缀（见第 3 节第 4 条） |
| `auth.refresh_deadline` | `4s` | 大于 0；加上 `database.commit_timeout` 小于 8 秒（前端续期请求的超时，3.5、7.1） |
| `ratelimit.ipv6_prefix_len` | `64` | 1–128 |
| `ratelimit.anonymous` | `{per_minute: 600, burst: 100}` | 两项都至少 1 |
| `ratelimit.auth_failure` | `{per_minute: 60, burst: 60}` | 同上 |
| `ratelimit.authenticated` | `{per_minute: 1200, burst: 200}` | 同上 |
| `ratelimit.login_ip` | `{per_minute: 30, burst: 10}` | 同上 |
| `ratelimit.login_ip_email` | `{per_minute: 10, burst: 5}` | 同上 |
| `ratelimit.register_ip` | `{per_minute: 10, burst: 5}` | 同上 |

- `config.test.yaml` 把六个桶都调到 `{per_minute: 600000, burst: 100000}`（3.10"测试环境"；A15 另起一个限流很低的 nerve）。
- `password_user` 随修改密码在 P3 加入（第 3 节第 7 条）；`auth.session_cleanup_interval`、`jobs.shutdown_timeout` 等属于 P3。
- **列表型的环境变量**：`listHook` 把字符串按逗号切开、去掉两端空白，空串是空列表，例如 `NERVE_SERVER__TRUSTED_PROXIES=10.0.0.0/8,fd00::/8`；解码链是 `emptyValueHook → durationHook → numberHook → listHook → StringToNetIPPrefixHookFunc`。
- **YAML 的空值**（P1 评审 M2）：`rejectNulls` 在解码之前拒绝任何没有值的键（`auth.signup_enabled: must not be null (a key without a value in YAML)`），整节为空（`ratelimit:`）同样拒绝。
- **越界的数**（P1 评审 M2）：`numberHook` 拒绝 YAML 中目标整数类型装不下的数和小数：`'auth.password.argon2_memory_kib' must be a whole number from 0 to 4294967295, got -1`；`max_conns: 5000000000` 和 `burst: 1.5` 同理。环境变量的值是字符串，由解码器按类型的位数解析，已经会报错。
- `BucketConfig` 实现 `LogValue`（两个数）；`Config.LogValue` 加上 `trusted_proxies`（逗号连接）、`refresh_deadline` 和 `ratelimit` 一组。
- 测试：`TestLoadReadsTrustedProxies`（YAML、环境变量、环境变量清空、没有，4 个）；`TestLoadErrors` 新增 7 个（布尔空值、整节空值、负数给无符号键、超出 `int32`、小数、环境变量的负数、畸形 CIDR）；`TestValidateReportsEveryInvalidKey` 覆盖 `refresh_deadline` 和每个桶的两项；`TestValidateCrossKeyRules` 新增 3 个（`refresh_deadline + commit_timeout` 达到 8 秒、IPv4 映射的可信代理、前缀长度 129）；`TestBuiltInProfiles` 核对三个环境的桶（test 全部调高）。

### 2.4 `platform/ratelimit`（M2 设计 3.10）

```go
type Rate struct{ PerMinute, Burst int }
func New(now func() time.Time) *Limiter
func (l *Limiter) Bucket(name string, r Rate) *Bucket
func (b *Bucket) Name() string
func (b *Bucket) Allow(key string) (retry time.Duration, ok bool)
func (b *Bucket) Reserve(key string) (refund func(), retry time.Duration, ok bool)
type Check struct{ Bucket *Bucket; Key string }
func (l *Limiter) AllowAll(checks ...Check) (denied *Bucket, retry time.Duration)
```

- 每个 `(bucket, key)` 一个水位（浮点的单位数和时刻），新键从满桶开始；每 `1 分钟 / PerMinute` 回来一个单位，最多 `Burst`。一个 `Limiter` 的全部桶共用一把锁，`AllowAll` 在这把锁下先看每个桶是否都有一个单位，都有才一起扣；有一个没有就都不扣，返回等待最长的桶和它的等待（`(1 - 余额) × 间隔`）。`Allow` 是只有一个检查的 `AllowAll`。
- `Reserve` 扣一个单位，返回的 `refund` 最多生效一次（`sync.Once`），退回时不超过突发。
- **清理**：`AllowAll` 最多每分钟清一次已经回满的键（满桶就是新键的初始状态，删掉不改变行为），内存只跟最近几分钟的调用方有关。
- `AllowAll` 收到别的 `Limiter` 的桶时 panic（接线错误）。
- 限流器的时钟是 `time.Now`，不是 `clock.System`：后者转 UTC、截到微秒，丢掉了单调时钟读数，墙上时钟被调整时桶会被错误地回满或抽干（第 3 节第 11 条）。
- 测试（9 个，用可推进的时钟）：`TestBucketAllowsItsBurstThenItsRate`、`TestBucketNeverHoldsMoreThanItsBurst`、`TestKeysAndBucketsAreSeparate`、`TestAllowAllTakesFromEveryBucketOrNone`、`TestAllowAllReportsTheLongestWait`、`TestAllowAllRejectsABucketOfAnotherLimiter`、`TestReserveAndRefund`（退回一次、第二次调用无效、不超过突发）、`TestReserveUnderConcurrency`（额度 3、50 个并发，只成功 3 个）、`TestIdleKeysAreDropped`。

### 2.5 安全响应头（M2 设计 8.3；M0-P5 交接）

- `httpserver` 的固定链变为 `请求 ID → 异常恢复 → 访问日志 → 安全响应头`：`withSecurityHeaders` 在 handler 写出之前设上 `X-Content-Type-Options: nosniff`、`Referrer-Policy: same-origin`、`X-Frame-Options: DENY`。
- 异常恢复在 panic 之后清掉 handler 设过的响应头（保留 `X-Request-Id`，P1 行为），再重新设上这三个，panic 的 500 同样带着（第 3 节第 1 条）。
- CSP 不在这里：只对页面有意义，由 P4 在 `webui` 中加入（8.3）。
- 测试：`httpserver` 的 `TestSecurityHeadersOnEveryResponse`（200、404、problem 三种）；`TestPanicDiscardsHeadersSetBeforeIt` 另核对三个响应头；`bootstrap` 的 `TestSecurityHeadersOnEveryResponse` 在接好线的程序上核对页面（`/`、深链接）、`/healthz`、`/api/v0/instance`、`/api/v0` 下的 404 和 401。

### 2.6 客户端 IP 与可信代理（M2 设计 3.10）

- `clientIPs.of(r)`：对端地址（去掉端口）不在 `server.trusted_proxies` 中时就是客户端；在其中时，把全部 `X-Forwarded-For` 行按逗号拼成一个列表，从右往左取第一个不可信的地址；全部可信时取最左边的一个。遇到解析不了的一项（空项、带端口、带方括号、主机名）就停在转发它的那个可信代理上（第 3 节第 3 条）。每个地址去掉 zone，IPv4 映射的 IPv6 地址转为 IPv4。对端地址解析不了时是零值。
- 对端不可信却带着 `X-Forwarded-For`：每个进程只记一次 WARN（`sync.Once`），带 `peer`，文字说明要把代理加进 `server.trusted_proxies`（8.4）。
- `clientIPs.key(ip)`：IPv4 是地址本身，IPv6 是 `ratelimit.ipv6_prefix_len` 位的前缀（例如 `2001:db8:1:2::/64`），零值是 `""`（第 3 节第 12 条）。
- `RequestMeta` 加上 `IPKey`；请求元信息中间件算出 `ClientIP` 和 `IPKey`。日志和 `auth_sessions.ip` 仍记完整的地址。
- `APIConfig` 加上 `TrustedProxies`、`IPv6PrefixLen`。
- 测试：`TestClientIP`（13 个：不可信对端的转发被忽略、可信代理没有转发、可信代理转发、跳过可信的多跳、左边客户端自己写的不采信、多行合为一个列表、全部可信取最左、畸形项、空项、IPv6、映射的客户端、映射的对端、zone）、`TestClientIPOfAnUnparsablePeerIsZero`、`TestUntrustedForwardingIsWarnedOnce`、`TestIPKey`（6 个，含同一 /64 的两个地址、不同 /64、/48、/128）、`TestRequestMetaCarriesTheClientAndItsKey`。

### 2.7 失败闸门、限流与 `rate_limited`（M2 设计 3.6、3.10、3.11）

**`httpserver.Limiter`**（`httpserver` 声明，`*ratelimit.Bucket` 按结构满足，平台包之间不互相导入，规则 7）：

```go
type Limiter interface {
	Allow(key string) (retry time.Duration, ok bool)
	Reserve(key string) (refund func(), retry time.Duration, ok bool)
}
```

- `APIConfig` 加上 `Anonymous`、`Authenticated`、`AuthFailure` 三个 `Limiter`。
- **`NewAPI(cfg) (*API, error)`**（P1 评审交来的一项）：`Logger`、`Authenticator` 和三个桶缺任何一个都返回 `httpserver: APIConfig lacks <按字段顺序的名字>`，`bootstrap` 关闭已打开的资源后返回这个错误。
- **中间件顺序**：`请求元信息 → 期限 → 请求体上限 → 失败闸门和认证 → 限流 → 请求体结构`，仍按相反的顺序返回（M2 设计 3.6）。
- **失败闸门**（在 `authenticate` 里，公开操作和没有令牌的请求不经过它）：`AuthFailure.Reserve(IPKey)`；拿不到 → 429，不调用认证器。认证器返回 401 的 `ProblemError` → 单位留下，只有同时满足可选接口 `ExpiredCredential() bool` 并返回 true 时退回；认证器返回其他错误 → 退回，交给 `Errors.Write`；成功 → 退回。
- **限流**（`rateLimit`）：认证给出限流键（`session:<id>`）时扣 `Authenticated` 的这个键，否则扣 `Anonymous` 的 `IPKey`。被模块自己的桶拒绝的请求在这里仍然算一次（3.10）。
- **429**：`tooManyRequests` 记 INFO "rate limited"（`request_id`、`bucket`、`ip`，8.4），再由 `Errors.Write` 写出 `rateLimited(retry)`：429 `rate_limited`，`detail` "Too many requests; retry later."，`Retry-After` 向上取整到秒。平台码加上 `CodeRateLimited`。
- **`authn`**：`app.ErrAccessTokenExpired` 包成 `expired{err}`（`Unwrap` 保留原错误，`ExpiredCredential()` 为 true）；其余照旧。
- **契约**（P1 评审 M6）：`api/openapi.yaml` 顶层 `x-problem-codes` 为 `[bad_request, payload_too_large, rate_limited, internal_error]`（P1 spec 第 3 节第 2 条说的"随 P2 加入"）；`api/common.yaml` 的 `Problem.code` 说明列出 `rate_limited`；`identity.yaml`、`instance.yaml` 的 `Problem` 响应声明两个响应头：`Retry-After`（`integer`，`minimum: 1`，"Whole seconds to wait before trying again, rounded up; sent with rate_limited and server_busy."）、`WWW-Authenticate`（`string`，"Sent with every 401 (RFC 9110 15.5.2): Bearer, or Bearer error=\"invalid_token\" when the bearer token sent is invalid or has expired (RFC 6750 3)."）。响应头写在每个模块的 `Problem` 响应里，`common.yaml` 只放 schema（第 3 节第 13 条）。
- **`apitest` 的测试卫生**（P1 评审交来的一项）：`TestCheckResponseRecordsTheAnsweredCode` 用自己独有的操作和码（`recordThing` / `things.recorded`），先断言记录为空，不依赖别的测试留下的状态；`Main` 中的循环变量 `m` 改名 `entry`。
- **P1 评审交来的注释**：`TestAuthenticateTellsAnExpiredAccessToken` 覆盖 `exp` 恰好等于现在和早于现在 1 秒两种，注释改为"exp has come, at that instant or past it"。
- 测试（`api_test.go`）：`TestNewAPIRequiresItsDependencies`；`TestFailureGateKeepsTheUnitOfAFailedCredentialOnly`（6 个：失败的凭证留下、成功退回、过期退回、认证器故障退回、没有令牌和公开操作不预留）；`TestFailureGateTurnsAwayWithoutAuthenticating`；`TestFailureGateHoldsUnderConcurrency`（额度 3、认证 20 毫秒、50 个并发的无效令牌：认证器只被调用 3 次，3 个 401、47 个 429）；`TestFailureGateCountsByTheIPKey`（同一 /64 的两个地址共用闸门）；`TestRateLimitPicksTheBucketAndKey`（5 个）；`TestRateLimitedRequestIs429`（匿名、已认证各一次，`Retry-After` 向上取整，请求体不合契约也先得到 429，日志带桶名）。`contract_test.go` 的 `TestPlatformProblemsMatchTheContract` 加上 429。假的限流器和认证器都核对自己收到的键和令牌。

### 2.8 `identity` 领域：刷新令牌的解析与续期判定（M2 设计 3.4、3.5、5.4）

- `ParseRefreshToken(s string) (RefreshToken, bool)`：长度恰好是 98（`nrv_rt_` 加 91 个字符）；前缀正确；其余用 `base64.RawURLEncoding.Strict()` 解码（最后一个字符没用到的位必须为 0，`\r`、`\n` 不会被跳过），解出恰好 68 字节；代数不超过 `math.MaxInt32`（`auth_sessions.generation` 是 `integer`，更大的代数不可能由服务端签发，第 3 节第 5 条）。不查库，不核对标签。
- `SessionState{Generation uint32; TokenHash []byte; Revoked bool; ExpiresAt time.Time}`；`Verdict`：`Rotate`、`Reuse`、`Reject`。
- `JudgeRefresh(s, token, tagValid, now) Verdict`，就是 3.5 的表：已撤销或 `now >= ExpiresAt` → `Reject`；代数相同且 `SHA-256(密文) = TokenHash` → `Rotate`（不看标签）；代数较旧且标签成立 → `Reuse`；其余 → `Reject`。会话不存在由调用方判为 `Reject`。
- 模块错误：`ErrInvalidCredentials`（401 `identity.invalid_credentials`，"The e-mail address or the password is incorrect."）、`ErrAccountDeactivated`（403 `identity.account_deactivated`，"This account is deactivated."）、`ErrRefreshTokenInvalid`（401 `identity.refresh_token_invalid`，"The refresh token is not valid; sign in again."）。
- 测试：`TestParseRefreshTokenReadsWhatStringWrites`；`TestParseRefreshTokenRejects`（13 个：空串、只有前缀、别的前缀、PAT 形状、少一个字符、多一个字符、`=` 补位、标准 base64 的字符、中间换行、前面的空格、后面的空格、没用到的位不为 0、代数超出 `integer`）；`TestJudgeRefresh`（11 行，逐行对应 3.5 的表，含"恰好在期限那一刻"）。

### 2.9 `identity` 用例与端口（M2 设计 3.5、3.8、3.9、6.3）

**端口**（`app/ports.go` 新增）：

```go
type LoginAccount struct{ ID uuid.UUID; PasswordHash string }
type LoginAccountReader interface { FindLoginAccount(ctx, email string) (LoginAccount, error) }   // 没有时 ErrNotFound
type LockedAccount struct{ PasswordHash string; Active bool }
type CredentialLocker interface { LockForCredentials(ctx, id uuid.UUID) (LockedAccount, error) }  // 在事务里调用
type PasswordHashWriter interface { UpdatePasswordHash(ctx, id uuid.UUID, hash string, now time.Time) error }
type RefreshSession struct{ UserID uuid.UUID; State domain.SessionState }
type SessionGeneration struct{ ID uuid.UUID; Generation uint32; TokenHash []byte; Now time.Time }
type SessionRotator interface {
	SessionForRefresh(ctx, id uuid.UUID) (RefreshSession, error)            // 没有时 ErrNotFound
	RotateSession(ctx, g SessionGeneration, newHash []byte) (bool, error)   // 会话已不在 g 时 false
	RevokeForReuse(ctx, id uuid.UUID, now time.Time) error
}
type SessionEnder interface { EndSession(ctx, g SessionGeneration) (bool, error) }
// PasswordHasher 加上 Verify(ctx, password, hash) (ok, rehash bool, err error)
// RefreshTokenMAC 加上 Verify(message []byte, tag [16]byte) bool
```

**`Issuance`**（`app/issuer.go`，注册、登录、续期共用）：`Issuance{Tokens AccessTokens; MAC RefreshTokenMAC; AccessTTL, SessionTTL time.Duration}`，方法 `refreshToken(sessionID, generation)`（新密文和标签）、`tokens(userID, refresh, now, sessionEnd)`（签出访问令牌，拼成 `Tokens`）、`newSession(userID, userAgent, ip, now)`（第 0 代的会话行和它的令牌，`expires_at = now + SessionTTL`）。`Tokens` 移到这里；`RegisterDeps` 的 `Tokens`、`MAC`、`AccessTTL`、`SessionTTL` 四个字段合成一个 `Issuance`（第 3 节第 14 条）。

**`Login.Execute(ctx, LoginInput{Email, Password, UserAgent, IP}) (Tokens, error)`**（3.5 的"用密码签发"一行，3.9）：

1. 邮箱规范化；`ValidEmail` 不成立的地址不查库（NUL、非 UTF-8 等数据库存不下的输入不会到达数据库），按不存在处理；
2. 不存在 → 用 `LoginDeps.DummyHash` 做一次 `Verify`（结果不用），再 401 `identity.invalid_credentials`；
3. 在事务外对快照 `Verify`；需要重新哈希时，新哈希也在事务外算好；
4. 事务：`LockForCredentials` → 哈希不等于快照时退出事务、不写任何东西 → 停用 → 403 `identity.account_deactivated`（此时密码已知正确）→ 需要时 `UpdatePasswordHash` → 插入会话；
5. 哈希变了：以新哈希为快照回到第 3 步，最多两轮（`for range 2`）；第二轮哈希又变了，或新哈希下密码不对，都是 401。
6. 日志：成功 INFO "signed in"（`user_id`、`session_id`、`ip`）；失败 INFO "sign-in failed"（`reason` 是 `invalid_credentials` 或 `deactivated`、`ip`，已知账户时加 `user_id`，不记邮箱）。哈希器忙（503 `server_busy`）和数据库错误原样返回。

**`Refresh.Execute(ctx, token string, ip netip.Addr) (Tokens, error)`**（3.5 的续期表）：

1. `ParseRefreshToken` 失败 → 401，不查库；
2. 事务外核对标签（`MAC.Verify`，`hmac.Equal`），算好下一代令牌；
3. 事务：按主键读会话（不存在 → `Reject`）→ `JudgeRefresh` → `Reject` 结束；`Reuse` → `RevokeForReuse`，事务**正常提交**；`Rotate` → 签出令牌，条件 `RotateSession`；没有命中 → 在同一个事务里重读、重判，最多两轮，仍不命中返回内部错误 `errRotatedTwice`（500，第 3 节第 15 条）；
4. 事务之后：`Rotate` 返回新令牌，`refresh_token_expires_at` 仍是会话原来的 `expires_at`；`Reuse` 记 WARN "refresh token reused: session revoked"（`user_id`、`session_id`、`ip`），再 401；其余 401。

**`Logout.Execute(ctx, token string) error`**（3.5"退出"，6.4"单条语句的写入不开事务"）：解析失败 → nil；否则一条条件 `UPDATE`（`EndSession`，条件与轮换相同）；命中时记 INFO "signed out"（`session_id`）；没命中也是 nil。数据库错误原样返回（500）。

**`argon2adapter.Hasher.Verify`**：`parsePHC` 只接受 `Hash` 写出的形式：6 段；`argon2id`；`v=19`；`m=`、`t=`、`p=` 按这个顺序，都是大于 0 的十进制数，`p` 小于 256（argon2 在 0 轮、0 条并行时 panic）；盐 16 字节、密钥 32 字节，严格的无补位标准 base64。别的形式返回 `errNotOurHash`（不引用哈希原文）。校验与 `Hash` 共用名额和等待上限（3.8）；`rehash` 在参数与当前配置不同时为 true；比较用 `subtle.ConstantTimeCompare`。

**`signing.RefreshTokenMAC.Verify`**：重新算标签，`hmac.Equal` 比较（3.9）。

测试：`TestLoginSignsIn`；`TestLoginFailsAlikeForAnUnknownAddressAndAWrongPassword`（5 个：错误密码、未知地址、只有空白的地址、带 NUL 的地址、不是 UTF-8 的地址；每个都恰好做一次 `Verify`（后四种对假哈希）、不做哈希、不开事务，后三种不查库）；`TestLoginRevealsDeactivationOnlyWithTheRightPassword`；`TestLoginRehashesWhenTheParametersChanged`；`TestLoginVerifiesAgainWhenAConcurrentLoginRehashed`；`TestLoginFailsWhenThePasswordChangedMeanwhile`；`TestLoginFailsWhenTheHashChangesTwice`；`TestLoginWhenTheHasherIsBusy`；`TestLoginLogsNoSecret`；`TestRefreshRotates`；`TestRefreshEveryGenerationInTurn`；`TestRefreshDetectsReuse`；`TestRefreshRejectsWithoutRevoking`（8 个，含"伪造的旧代"和"访问令牌里的会话 id 加 g = 0"）；`TestRefreshAfterTheSigningKeyChanged`；`TestRefreshJudgesAgainWhenAConcurrentRefreshWins`；`TestRefreshJudgesAgainWhenARevocationWins`；`TestRefreshGivesUpWhenTheRotationMissesTwice`；`TestRefreshLogsNoSecret`；`TestLogoutEndsTheSessionOfTheCurrentGeneration`；`TestLogoutLeavesTheSessionForAnyOtherToken`（9 种：会话发过的旧代、伪造的旧代、当前代数配别的密文、更新的代数、不存在的会话、已撤销、恰好到期、畸形、空串）；argon2 的 `TestVerify`、`TestVerifyAsksForARehashWhenTheParametersChanged`、`TestVerifyRejectsAnotherFormat`（16 个）、`TestVerifyIsBusyWhenNoSlotFreesUp`；`TestRefreshTokenMACVerify`（自己的标签成立；随机标签、最后一位翻转的标签、全零标签、改动消息（会话 id、代数、密文）52 个字节中任何一个、换一把密钥，都不成立）。假的仓储按真实的 `WHERE` 条件判断命中（`fakeSessions.at`），并有 `beforeWrite` / `afterWrite` 钩子模拟并发的写入。

### 2.10 仓储与查询（M2 设计 3.5、3.13）

```sql
-- name: FindLoginAccount :one
SELECT id, password FROM users WHERE email = sqlc.arg(email);

-- name: LockUserForCredentials :one
SELECT password, is_active FROM users WHERE id = sqlc.arg(id) FOR NO KEY UPDATE;

-- name: UpdatePasswordHash :exec
UPDATE users SET password = sqlc.arg(password), updated_at = sqlc.arg(now) WHERE id = sqlc.arg(id);

-- name: GetSessionForRefresh :one
SELECT user_id, generation, token_hash, expires_at, revoked_at FROM auth_sessions WHERE id = sqlc.arg(id);

-- name: RotateSession :execrows
UPDATE auth_sessions
SET updated_at = sqlc.arg(now), last_refreshed_at = sqlc.arg(now),
    generation = generation + 1, token_hash = sqlc.arg(new_token_hash)
WHERE id = sqlc.arg(id) AND generation = sqlc.arg(generation) AND token_hash = sqlc.arg(token_hash)
  AND revoked_at IS NULL AND expires_at > sqlc.arg(now);

-- name: RevokeSessionForReuse :exec
UPDATE auth_sessions
SET updated_at = sqlc.arg(now), revoked_at = sqlc.arg(now), revoke_reason = 'reuse_detected'
WHERE id = sqlc.arg(id) AND revoked_at IS NULL;

-- name: EndSession :execrows
UPDATE auth_sessions
SET updated_at = sqlc.arg(now), revoked_at = sqlc.arg(now), revoke_reason = 'logout'
WHERE id = sqlc.arg(id) AND generation = sqlc.arg(generation) AND token_hash = sqlc.arg(token_hash)
  AND revoked_at IS NULL AND expires_at > sqlc.arg(now);
```

- sqlc 按参数**第一次出现**的列推断类型：`updated_at`（NOT NULL）放在每个 `SET` 的第一位，`now` 才是 `time.Time`；先写 `last_refreshed_at`（可空）会生成 `*time.Time`（原型实测）。查询文件的注释写明。
- `Store` 的方法：`FindLoginAccount`、`LockForCredentials`（`ErrNoRows` → `app.ErrNotFound`）、`UpdatePasswordHash`、`SessionForRefresh`（`uint32(row.Generation)`，CHECK 保证非负）、`RotateSession`、`RevokeForReuse`、`EndSession`（`int32(g.Generation)`，`ParseRefreshToken` 保证装得下）。
- `RevokeForReuse` 只在未撤销时写：已经因为别的原因撤销的会话保留原来的原因（第 3 节第 16 条）。
- 测试（真实数据库，`pgtest`）：`TestFindLoginAccount`；`TestLockForCredentialsReadsTheRow`；**`TestTheCredentialLockBlocksLocksNotInserts`**（M2 设计 3.5 测试第 6 条：一个事务持有锁时，另一个事务的 `LockForCredentials` 在 `lock_timeout = 500ms` 下得到 `55P03`，而插入引用这个账户的会话立即完成）；`TestUpdatePasswordHash`；`TestSessionForRefresh`；`TestRotateSession`；`TestRotateSessionMisses`（5 个：别的代数、别的哈希、不存在的会话、已撤销、恰好到期；行不变）；`TestRevokeForReuse`；`TestRevokeForReuseKeepsAnEarlierRevocation`；`TestEndSession`；`TestEndSessionMisses`。审计列都等于固定时钟。

### 2.11 接口与 HTTP 适配器（M2 设计 3.5、3.10、5.1、5.2、5.4）

| 操作 | `security` | `x-problem-codes` | 成功 |
|---|---|---|---|
| `login`：`POST /api/v0/auth/login` | `[]` | `identity.invalid_credentials`、`identity.account_deactivated`、`server_busy` | 200 `AuthTokens` |
| `refreshTokens`：`POST /api/v0/auth/refresh` | `[]` | `identity.refresh_token_invalid` | 200 `AuthTokens` |
| `logout`：`POST /api/v0/auth/logout` | `[]` | `[]` | 204 |

- `LoginRequest{email (format: email), password}`、`RefreshRequest{refresh_token}`、`LogoutRequest{refresh_token}`，都必填，`additionalProperties: false`。`AuthTokens` 的说明改为指向 `POST /api/v0/auth/refresh`；`register` 的说明加上"Registrations have a rate limit of their own per client IP."；三个操作的说明逐句与实现核对过（第 6 节"契约说明"）。
- 适配器按资源分文件（M2 设计 6.2）：`handler.go`（用例的小接口、`UseCases{Register, Login, Refresh, Logout, GetMe}`、`Settings{Limits, RefreshDeadline, Logger}`、`PublicOperations()`（四个）、`Register(router, api, uc, s)`）、`auth.go`（四个认证操作）、`me.go`（`getMe`）、`limits.go`（模块的桶）。
- **模块的桶**：`RateLimiter` 接口只有 `AllowAll(checks ...ratelimit.Check) (*ratelimit.Bucket, time.Duration)`（6.3）；`Limits{Limiter, LoginIP, LoginIPEmail, RegisterIP}`。登录在解码请求体之后、调用用例之前扣 `login_ip`（IP 键）和 `login_ip_email`（IP 键 + 空格 + 规范化邮箱的 SHA-256 十六进制，第 3 节第 2 条），全有或全无；注册扣 `register_ip`。拒绝时记 INFO "rate limited"（`request_id`、`bucket`、`ip`），返回 `shared.RateLimited(retry)`（429 `rate_limited`，走"一条路"）。续期、退出不经过模块的桶。
- **续期、退出的期限**：handler 调用用例之前 `context.WithTimeout(ctx, RefreshDeadline)`（3.5）。
- `login` 从 `RequestMetaFrom(ctx)` 取 UA 和 IP；`refreshTokens` 取 IP（重复使用的 WARN 用）。
- `shared.RateLimited(retry) *Error`（`KindRateLimited`、`CodeRateLimited`、"Too many requests; retry later."）；`bootstrap` 的 `TestEveryKindBecomesItsProblem` 用它核对 429 和 `Retry-After`。
- 模块入口：`Deps` 加上 `RefreshDeadline` 和 `RateLimits{Limiter httpadapter.RateLimiter; LoginIP, LoginIPEmail, RegisterIP *ratelimit.Bucket}`（转换成 `httpadapter.Limits`，第 3 节第 17 条）；假哈希在 `New` 中用当前参数对 `rand.Text()` 哈希一次（3.9"启动时生成"）。
- `bootstrap`：先建 `ratelimit.New(time.Now)`，六个桶都建在它上面（`bucket(limiter, name, cfg)`）；三个给 `API`，三个给 `identity`；`testConfig` 加上 `RefreshDeadline: 4s`。
- 测试：`auth_test.go` 的 `TestLoginAnswers200WithTheTokens`（用例收到的 `LoginInput` 逐字段核对）、`TestLoginProblems`（401 带 `WWW-Authenticate: Bearer`、403、503 带 `Retry-After: 1`、500）、`TestLoginWithoutAPassword`（400 `required`）、`TestRefreshTokens`（期限不超过 4 秒、IP 传到用例）、`TestRefreshTokensInvalid`、`TestLogout`、`TestLogoutFault`；`limits_test.go` 的 `TestLoginLimitsByIPAndByIPWithAddress`（冻结的时钟，`login_ip` 3、`login_ip_email` 2：同一地址（第二次换了大小写和空白）401、401、429，换地址 401，再换 429；用例只被调用 3 次；`Retry-After: 60`；两条日志依次是 `login_ip_email`、`login_ip`）、`TestRegisterLimitsByIP`、`TestRefreshAndLogoutTakeNoUnitOfTheModule`。`apitest.Main` 核对 identity 声明的每个码都有测试答过。

### 2.12 整程序测试（M2 设计 3.5、3.9，12 节 P2 完成线）

`server/internal/bootstrap/sessions_test.go`，真实数据库和真实的接线：

- `TestRefreshReuseRevokesTheSession`：续期一次后再交出第 0 代 → 401，会话 `reuse_detected`（撤销已提交）；之后最新的刷新令牌和访问令牌都是 401。
- `TestConcurrentRefreshesOfOneToken`：同一个令牌同时续期两次，5 轮：每轮恰好一个 200、一个 401，会话都是 `reuse_detected`（无论两个事务怎样交错）。
- `TestLoginTakesAsLongForAnUnknownAddress`：默认参数（m = 19456 KiB、t = 2），已有地址的错误密码与未知地址交替各 15 次，两组中位数相差不超过较大者的四分之一。

P1 的四个整程序测试自动覆盖三个新操作：公开集合等于契约、路由等于契约，`TestBodiesThatBreakTheStructureAnswer400` 对 `login`、`refreshTokens`、`logout` 的请求体逐项改坏。

### 2.13 端到端（M2 设计 2、9.5；M0-P6 交接）

- `auth.ts` 加上 `login(api, email, headers)`、`refresh(api, refreshToken)`（都断言 200 并返回 `AuthTokens`）。
- `assert/identity.ts`：`SignIn`（替代 P1 的 `Registration`）、`sessionOf`（按令牌里的会话 id 读出会话行）、`expectRegistered`、`expectSignedIn`（第 0 代、`token_hash`、UA、IP、恰好 30 天、未撤销）、`expectRefreshed(db, latest, refreshes, sessionEnd)`（代数、最新令牌的哈希、`last_refreshed_at` 已写、`expires_at` 不变、未撤销）、`expectRevoked(db, token, reason)`；`countIdentity`、`expectNothingAdded` 照旧。
- 故事（接口版本，页面版本随 P4 加入）：
  - `a3-sign-in.spec.ts`：大小写和空白不同的地址登录成功，访问令牌能读 `/me`；错误密码和未知地址同一个 401 和 `detail`；缺 `password` 是 400 `required`；失败不新增任何行。
  - `a4-refresh.spec.ts`：登录后连续续期三次，每次用上一次的刷新令牌；刷新令牌都不同，`refresh_token_expires_at` 不变，数据库代数为 3。访问令牌在同一秒内可能相同，不断言（第 3 节第 10 条）。
  - `a5-refresh-reuse.spec.ts`：伪造的旧代（真的会话 id 和代数，随机的密文和标签）401、会话不变；真实的旧令牌 401、`reuse_detected`；之后最新的刷新令牌和访问令牌都是 401。
  - `a6-sign-out.spec.ts`：上一代退出 204、会话不变；当前一代退出 204、`logout`，访问令牌下一个请求就是 401；再退出一次，什么都不变。
  - `a15-sign-in-limits.spec.ts`：`nerveWith` 另起一个 nerve（`NERVE_RATELIMIT__LOGIN_IP__PER_MINUTE=1`、`LOGIN_IP__BURST=3`、`LOGIN_IP_EMAIL__PER_MINUTE=1`、`LOGIN_IP_EMAIL__BURST=2`）：同一邮箱 401、401、429；换邮箱 401（证明被 `login_ip_email` 拒绝的请求没有扣 `login_ip`）；再换 429；429 带 `rate_limited` 和 `Retry-After: 60`；不新增任何行。
- 新增的等待：没有（A15 的 nerve 沿用 `nerveWith` 的预算）。

### 2.14 文档、README 与交接（M2 设计 3.20、8.7、13.1）

plan 的 Task 12 逐行给出文字。要点：

- **总体设计**：3.5 平台码加上 `rate_limited`；3.6 限流一条改写为桶、突发、默认值、失败闸门、客户端 IP；4.1 刷新令牌的期限改为从登录起 30 天的绝对期限；4.2 退出只结束当前会话、重复使用只在标签成立时撤销；6.4 固定链变为四个中间件，按路由加上失败闸门和限流。
- **M0 设计** 3.3：固定链的第四个中间件；按路由的中间件加上失败闸门和限流；平台码加上 `rate_limited`。
- **差异清单** 四：4.6 中标 P2 的三行（登录时邮箱不存在、会话的期限、限流）。
- **README** 部署：反向代理与 `server.trusted_proxies`（8.7 的 P2 一行）。
- **交接**：M0-P2、M0-P5、M0-P6 追加"处理结果（M2/P2）"一节，状态仍为 `open`（第 7 节）。

## 3. 与设计的差异和补充（请控制者裁定）

以下都没有改变 M2 设计的架构。

1. **panic 的 500 也带安全响应头**：异常恢复清掉 handler 设过的响应头（P1 的行为，防止半截的 `Content-Length`、`Set-Cookie` 漏出）之后重新设上三个安全响应头。8.3 说"所有响应"，这里把 panic 的 500 也算在内。
2. **`login_ip_email` 的键是"IP 键 + 空格 + 规范化邮箱的 SHA-256 十六进制"**：3.10 只写"客户端 IP 键 + 规范化后的邮箱"。取哈希让键的长度固定（请求体上限 1 MiB 之内的邮箱都能进键），也不在内存里留邮箱原文。
3. **`X-Forwarded-For` 中解析不了的一项结束查找**：客户端取转发了这一项的可信代理。带端口（`1.2.3.4:5678`）、带方括号的 IPv6、主机名、空项都算解析不了。Caddy 写的是纯地址，不受影响；在地址后面加端口的代理即使配置为可信，客户端也会取成代理本身，等于没有配置可信代理，README 的说明够用。
4. **可信代理不能是 IPv4 映射的 IPv6 前缀**（例如 `::ffff:10.0.0.0/104`）：客户端地址比较之前已转为 IPv4，这样的前缀什么也匹配不上，启动校验报出并提示写 IPv4 前缀。3.10 只要求"CIDR 合法"。
5. **代数超过 `math.MaxInt32` 的令牌在解析时拒绝**：`auth_sessions.generation` 是 `integer`，服务端签发不出更大的代数；这样仓储把代数转为 `int32` 时不会回绕。第 `MaxInt32` 代再续期会让 `generation + 1` 溢出，这次续期答 500。这个前提核对过：续期只受 `anonymous`（每分钟 600 次）约束，连续以这个速率续期也要约 6.8 年才到 2³¹ 代，默认 30 天的会话只能到约 2,600 万代；即使运维把 `session_ttl` 设得很长，后果也只是这个会话自己续不了期，不影响别人，不处理。
6. **退出是一条条件 `UPDATE`，不开事务**：6.4 已把退出列为"单条语句的写入"；判定与续期表的第一行相同，写在 SQL 的 `WHERE` 里。执行中被取消时可能已提交而答 500，重试得到 204（3.6 说明的无害情况）。
7. **`password_user` 桶和 `app/credential_lock.go` 随 P3 加入**：`password_user` 唯一的使用者是修改密码（3.10）；`credential_lock.go` 是"锁账户行、复核调用者的凭证"的共用部分（6.2），P2 只有登录一个使用者，而登录的锁内步骤（核对哈希快照）是它自己的。P2 在仓储上提供 `LockForCredentials`（6.3 的 `Users` 端口的一部分）并用集成测试证明锁的性质；共用的部分等 P3 有了第二个使用者再提取，不留没有使用者的代码。3.5 的六个交错测试是 P3 的完成线，P2 用会阻塞的钩子在用例测试里覆盖其中登录的三种（重新哈希、密码已改、哈希变两次）。
8. **`NewAPI` 返回 error**：P1 评审要求构造时检查依赖不为空；缺依赖是接线错误，`newApp` 把它和其他启动错误一样返回。
9. **`rate_limited` 的 `detail` 在两处**："Too many requests; retry later." 同时写在 `httpserver`（平台的桶）和 `shared.RateLimited`（模块的桶）。平台不导入 `shared`（规则 4），`shared` 不导入 `net/http`（规则 10），两处各写一次，`TestEveryKindBecomesItsProblem` 和 `TestRateLimitedRequestIs429` 分别核对。
10. **访问令牌的 `exp` 不截到会话的期限；同一秒内签出的访问令牌可能逐字节相同**：访问令牌只有 `sub`、`sid`、`exp`（秒）（3.4），同一个会话一秒内续期两次得到相同的访问令牌，A4 因此只断言刷新令牌不同。会话期限之后的访问令牌由每个请求的会话检查拒绝（`now >= expires_at` → 401，P1 的 `Authenticate`），所以不需要截短。
11. **限流器用 `time.Now`**：见 2.4。它不是业务判断用的时间（3.5 的"业务判断用的时间都来自用例的时钟"），而是速率的度量，需要单调时钟。
12. **解析不了的对端地址的 IP 键是 `""`**：这些请求共用一个桶。`net/http` 的 TCP 连接总有可解析的对端地址，只有测试或将来的 Unix 套接字会走到这里。
13. **契约的两个响应头写在每个模块的 `Problem` 响应里**：`api/common.yaml` 只放 schema，没有 `responses`（P1 的布局），所以 `identity.yaml`、`instance.yaml` 各声明一次；以后的模块照抄模板中的 `Problem` 响应。
14. **`Issuance` 重构了注册**：注册、登录、续期三处签发令牌的代码合成 `app.Issuance`，`RegisterDeps` 随之变小。注册的行为和测试不变（`register_test.go` 只改了构造）。
15. **续期两次都没有命中时是 500**：条件 `UPDATE` 没有命中说明有并发的写入；重读后第二次仍不命中，意味着第三个并发的写入者也抢先了，这在同一会话上几乎不可能（同一浏览器内有跨标签页的锁，7.1）。返回内部错误而不是无限重试；错误记 ERROR，客户端按 500 退避重试（3.5 的第 4 种情况同理）。
16. **`RevokeForReuse` 是专门的方法**：只写 `reuse_detected`，不做通用的"按原因撤销"：P3 的其他撤销（修改密码、停用、重置）按账户批量撤销，形状不同，到时各自加查询。已经撤销的会话不改原因。
17. **`identity.RateLimits` 转换为 `httpadapter.Limits`**：两个结构的字段相同，`module.go` 用类型转换，不逐字段复制；`bootstrap` 只依赖模块入口的类型。适配器的 `RateLimiter` 接口的参数和返回值是 `platform/ratelimit` 的类型（适配器可以导入平台，6.3 写明由 `platform/ratelimit` 实现）。
18. **非法邮箱的登录不查库**：`ValidEmail` 不成立的地址不可能存在于 `users`（注册时校验过），查库只会把 NUL、非 UTF-8 这类输入交给 Postgres（两者都会让查询报错，成为 500）。仍然做一次假哈希校验，耗时与其他失败相同。
19. **测试环境的桶全部调高**：3.10 的"测试环境"写的是做法，数值定为每分钟 600000、突发 100000；A15 用环境变量另起一个 nerve（2.13）。

## 4. 验收标准（完成线，M2 设计 12 节 P2）

- [ ] A3–A6、A15 的接口版本通过，P1 的故事（S1–S4、A1、A2）仍然通过。
- [ ] 登录的耗时测试（`TestLoginTakesAsLongForAnUnknownAddress`）通过。
- [ ] 续期的四个测试通过：伪造的旧代 401、不撤销（`TestRefreshRejectsWithoutRevoking`、A5）；过期访问令牌里的 `sid` 加 `g = 0` 401、不撤销（`TestRefreshRejectsWithoutRevoking`）；真实的旧令牌撤销（`TestRefreshDetectsReuse`、`TestRefreshReuseRevokesTheSession`、A5）；换了签名密钥后换钥之前的旧代 401、不撤销，当前一代照常续期（`TestRefreshAfterTheSigningKeyChanged`）。
- [ ] 3.10 表中除 `password_user` 外的每个桶都有测试：`anonymous`、`authenticated`（`TestRateLimitPicksTheBucketAndKey`、`TestRateLimitedRequestIs429`）、`auth_failure`（失败闸门的四个测试）、`login_ip`、`login_ip_email`（`TestLoginLimitsByIPAndByIPWithAddress`、A15）、`register_ip`（`TestRegisterLimitsByIP`）；续期、退出只经过 `anonymous`（`TestRefreshAndLogoutTakeNoUnitOfTheModule`、`TestRateLimitPicksTheBucketAndKey`）；IPv6 前缀（`TestIPKey`、`TestFailureGateCountsByTheIPKey`）。
- [ ] 失败闸门：超额后不再调用认证器（`TestFailureGateTurnsAwayWithoutAuthenticating`）；50 个并发的无效令牌不超过额度（`TestFailureGateHoldsUnderConcurrency`）；过期的 JWT、成功、认证器故障都退回（`TestFailureGateKeepsTheUnitOfAFailedCredentialOnly`）。
- [ ] `platform/ratelimit` 的测试覆盖突发、`AllowAll`、`Reserve` 的退回和闲置键的清理。
- [ ] 每个响应都带三个安全响应头（`httpserver` 和 `bootstrap` 的 `TestSecurityHeadersOnEveryResponse`）。
- [ ] `make lint`、`make test`、`make gen-check`、`make knip`、`make e2e` 通过。
- [ ] 3.20 中 P2 的各行、8.7 中 P2 的 README 内容在同一次合并中写好；交接按第 7 节处理。

## 5. 不在 P2 范围内

- A3–A6、A15 的页面版本、令牌管理器、`next_path`（P4）。
- `password_user`、`credential_lock.go`、3.5 在真实数据库上的六个交错测试、PAT、修改密码、停用、管理命令、会话的清理任务、River（P3）。
- CSP（P4，8.3）；接口调用日志（M8）。
- 多进程部署时的共享限流：限流器在进程内，v0 只有一个进程（总体设计 3.6）。

## 6. 风险

| 风险 | 应对 |
|---|---|
| 登录耗时测试在持续集成的机器上抖动 | 两组交替、各 15 次、比较中位数、容差四分之一；本机两批各 8 次运行中两组中位数相差都不到 6%，而且差别的方向只取决于先后（附录 A）。持续集成上失败时先看日志里的两个中位数，写进 P2 review |
| 数据库变慢时失败闸门的预留挡住同一 IP 的正常请求 | 3.6"预留的代价"已说明，§16 登记；上限是突发 60 |
| 反向代理没有配置成可信，所有人共用代理地址的额度 | 第一次出现不可信的 `X-Forwarded-For` 时记 WARN；README 的部署一节写明 |
| 并发续期的整程序测试依赖真实的交错 | 两种交错的结果相同（一个 200、一个 401、`reuse_detected`），测试断言的是结果，不是交错；原型中 `-count=5`（25 轮）全部通过（附录 A） |
| 契约说明与实现不一致（P1 评审发现过） | 三个操作和两个响应头的说明逐句与代码核对：`Retry-After` 只随 `rate_limited`、`server_busy` 发出；每个 401 都带 `WWW-Authenticate`（包括登录和续期的 401）；续期说明不声称"当前一代总能续期"，写明"未结束的会话的当前一代" |

## 7. 交接的处理

| 交接 | P2 处理的条目 | 留下的条目 | 状态 |
|---|---|---|---|
| M0-P2-platform-notes | 2 中限流：失败闸门和限流按路由挂在 `API.Middlewares` 上，`NewServer` 没有为限流改动（Task 5、9） | 2 中接口调用日志（M8，挂在限流之后）；5 中 River、停机顺序、连接池关闭的时限、River 的迁移（P3） | open |
| M0-P5-frontend-api-notes | 安全响应头：固定链的第四个中间件（Task 3）；认证接口都在 `/api/v0/auth/` 下（Task 9） | CSP（P4，这一项在 P4 正式关闭，8.3）；前端改调新接口和同源部署的核对（P4） | open |
| M0-P6-e2e-notes | 认证 fixture 的登录（`login`、`refresh`，Task 11）；新等待的期限（P2 没有新增等待） | PAT 对等验收和 fixture 的 PAT（P3）；页面的登录状态（P4）；S3 的新字段（P3）；S2 的断言（P4）；River 停机（P3）；fixture 写法的延伸（M4、M5、M8） | open |

P1 评审第 6 节交给 P2 的五项：

| 事项 | 落在 |
|---|---|
| 配置加固：负数或溢出的数、YAML 布尔空值、`refresh_deadline + commit_timeout < 8 秒`、`trusted_proxies` 的 CIDR 校验 | Task 1（2.3） |
| `NewAPI` 在构造时检查依赖 | Task 5（2.7） |
| 契约声明 `Retry-After`、`WWW-Authenticate`，与 `rate_limited` 一起 | Task 5（2.7） |
| `apitest` 的测试卫生（独有的操作和码；循环变量改名） | Task 5（2.7） |
| `TestAuthenticateTellsAnExpiredAccessToken` 的注释与边界 | Task 5（2.7） |

## 附录 A：原型验证记录（2026-09-26）

原型在 `$M2TMP/p2proto`（`fd69736` 的副本，Go 1.27.1、Node 24.15.0、Docker 29.7.2、Apple M5 Max）。plan 中的代码就是原型中运行过的代码，由脚本从原型文件原样拼入 plan（新文件和改动大的文件给完整内容，改动小的给对前一个版本的统一差异；拼接脚本用 `patch -p1` 把 48 个差异块逐个应用到前一个版本上，结果都与原型的文件逐字节相同）。另在 `$M2TMP/p2stage` 从 `fd69736` 的副本出发，**按 plan 的 Task 1–12 逐个应用**（最终文件加上 plan 写明的过渡版本），有生成物的 Task（5、8、9）先运行 `make gen` 并核对生成物与 plan 的 SHA-256 相同，每个 Task 之后都运行 `make lint-go` 和 `make test`，全部通过；Task 5、9、11 之后另跑了前端检查和端到端，Task 11 之后跑了 `make knip`，Task 12 之后跑了关键词守卫。Task 12 之后 `p2stage` 与原型逐文件相同（含生成物；不提交的构建产物除外）。另用 plan 写明的命令 `pnpm -C e2e exec playwright test stories/identity --repeat-each=5 --workers=1` 在 `p2stage` 上重跑，35 个全部通过。

| 核对 | 命令 | 结果 |
|---|---|---|
| 生成物没有差异 | `make gen` 前后逐目录比较 `apigen`、`modules/*/adapter/{http,postgres}/gen`、`api/dist`、`schema.gen.ts`（原型不是 git 仓库，用脚本代替 `git status`） | 没有差异 |
| Go 静态检查 | `make lint-go` | server 0 issues；server/tools 0 issues |
| Go 测试 | `make test`（testcontainers，`postgres:18.6`） | 全部通过 |
| 前端检查 | `turbo run check:types check:lint check:format check:sync`；`make knip`；关键词守卫（遍历目录的等价脚本） | 54 个任务通过；knip 通过；5 个命中都有例外，没有新的命中 |
| 端到端 | `make build`，再 `playwright test` | S1、S2（两个）、S4、A1、A2、A3、A4、A5、A6、A15 共 11 个通过；`stories/identity --repeat-each=5 --workers=1` 35 个全部通过。S3 在原型中失败，原因与 P1 相同：副本不是 git 仓库，`go build` 取不到提交号 |
| 登录耗时 | `go test -count=8 -v -run TestLoginTakesAsLongForAnUnknownAddress ./internal/bootstrap/` | 原型中两批各 8 次都通过：已有地址的中位数 15.4–17.3 毫秒，未知地址 15.4–17.7 毫秒，每次两者相差不到 6%。每一对里后发的那个略慢（约 0.4–0.8 毫秒）：临时把两者的先后交换，差别的方向随之反过来，所以这是测量顺序的影响（多半是回收上一次 argon2 的 19 MiB 内存），不是可区分两种失败的差别 |
| 并发续期 | `go test -count=5 -run 'TestConcurrentRefreshesOfOneToken\|TestRefreshReuseRevokesTheSession' ./internal/bootstrap/` | 全部通过（25 轮并发续期，每轮一个 200、一个 401、`reuse_detected`） |
| 失败闸门的并发 | `TestFailureGateHoldsUnderConcurrency` | 额度 3、50 个并发：认证器调用 3 次，3 个 401、47 个 429（与设计的 spike 相同） |
| 账户行锁 | `TestTheCredentialLockBlocksLocksNotInserts` | 第二个锁在 500 毫秒后得到 `55P03`；插入引用这个账户的会话不等待 |
| sqlc 的参数类型 | 把 `last_refreshed_at` 放在 `SET` 的第一位再生成 | `Now` 变成 `*time.Time`；`updated_at` 在前时是 `time.Time` |

**变异核对**（`$M2TMP/p2tools/mutate.py`：改一处代码，跑相关的包，恢复）：

| 变异 | 结果 |
|---|---|
| 先认证、后预留闸门的单位 | `TestFailureGateTurnsAwayWithoutAuthenticating`、`TestFailureGateHoldsUnderConcurrency` 失败 |
| 过期的访问令牌不退回单位 | `TestFailureGateKeepsTheUnitOfAFailedCredentialOnly/expired_access_token` 失败 |
| 解析刷新令牌不用 `Strict()` | `TestParseRefreshTokenRejects` 失败 |
| 未知地址不做假哈希校验 | `TestLoginFailsAlikeForAnUnknownAddressAndAWrongPassword` 的 4 个子测试和 `TestLoginWhenTheHasherIsBusy` 失败；整程序的 `TestLoginTakesAsLongForAnUnknownAddress` 也失败 |
| `FOR NO KEY UPDATE` 改为 `FOR UPDATE` | `TestTheCredentialLockBlocksLocksNotInserts` 失败（插入等锁） |
| 去掉锁 | `TestTheCredentialLockBlocksLocksNotInserts` 失败 |
| 重复使用时事务返回错误（撤销被回滚） | `TestRefreshReuseRevokesTheSession`、`TestConcurrentRefreshesOfOneToken` 失败 |
| 异常恢复不重新设安全响应头 | `TestPanicDiscardsHeadersSetBeforeIt` 失败 |
| 去掉 `Middlewares` 中的 `slices.Reverse` | 16 个测试失败（顺序、闸门、限流） |
| `AllowAll` 在检查中途就扣（拒绝时前面的桶已扣） | `TestBucketAllowsItsBurstThenItsRate`、`TestAllowAllTakesFromEveryBucketOrNone` 失败 |
| 请求体结构检查放到限流之前 | `TestRateLimitedRequestIs429` 失败 |
| 限流放到认证之前 | `TestRateLimitPicksTheBucketAndKey`、`TestRateLimitedRequestIs429` 失败 |

**P1 的缺陷类别**（P1 评审发现过的六类，对本 plan 逐类核对）：

| 类别 | 核对了什么 | 结果 |
|---|---|---|
| 测试漏掉的空白和边界 | 刷新令牌的写法（换行、两端空格、`=`、没用到的位、代数上限）；登录的地址（大小写、两端空白、只有空白、NUL、非 UTF-8）；`X-Forwarded-For`（空项、项两边的空白、多行、映射地址、zone）；期限的边界（判定表、条件 `UPDATE`、退出都测了"恰好到期"）；`Retry-After` 向上取整；可信代理的写法 | 发现一处并修正：IPv4 映射的可信代理前缀什么也匹配不上而没有报错，现由启动校验拒绝（第 3 节第 4 条） |
| 忽略参数的假实现 | `authn` 的假用例、`fakeLogins`（按地址和账户 id）、`fakeSessions`（按真实的 `WHERE` 条件）、`fakeHasher`（按密码和哈希）、假限流器（记下键）、HTTP 适配器的假用例（记下入参和剩余的期限） | P1 的 `authn` 假用例原来不看令牌，已改为核对；其余都核对自己的参数 |
| 日志里的库错误原文 | 新增的日志："rate limited"、"signed in"、"sign-in failed"（原因是固定的两个词）、"refresh token reused: session revoked"、"signed out"、不可信转发的 WARN | 都不带库的错误原文；`argon2` 的 `errNotOurHash` 不引用哈希；认证失败的原因仍只进 DEBUG（P1） |
| 契约说明声称链上没有的检查顺序或行为 | 三个新操作、`register` 的补充、`AuthTokens`、两个响应头的说明逐句对照代码和测试 | 发现一处并修正：`refreshTokens` 原来写"不是当前一代的令牌答 401"，漏了"已结束的会话的当前一代也答 401"；`WWW-Authenticate` 写"每个 401 都带"，由 `TestLoginProblems`、`TestRefreshTokensInvalid` 核对登录和续期的 401 也带 `WWW-Authenticate: Bearer` |
| 死链接 | 脚本检查 spec、plan 和改动的文档中的相对链接 | 没有 |
| 建立在未核对前提上的推迟 | `password_user`（3.10 表：只有修改密码用它）；`credential_lock.go`（P2 只有登录一个使用者）；代数溢出（见第 3 节第 5 条的速率计算）；访问令牌的 `exp` 不截短（P1 的 `Authenticate` 按 `now >= expires_at` 拒绝，有测试）；IP 键为空（TCP 连接的对端地址总能解析） | 前提都成立 |

**没有证明的**：

- 持续集成（ubuntu）上的运行，特别是登录耗时测试的稳定性（第 6 节）。P2 合并后看持续集成的结果。
- S3 在原型中因为不是 git 仓库而失败，在仓库中照常。
- 真实反向代理（Caddy）后面的客户端 IP：只有单元测试和 README 的说明；P4 或 M8 的部署核对时再看。
- 3.5 的六个交错测试在真实数据库上的版本（P3 的完成线）；P2 只有用例层的钩子测试和锁的集成测试。
- 大量不同键时限流器的内存：清理有测试，规模没有压测；v0 的部署规模下不是问题。
