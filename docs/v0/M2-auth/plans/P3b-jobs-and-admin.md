# M2/P3b River 与管理命令 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 服务器管理员的五个命令（`nerve users create`、`reset-password`、`set-email`、`deactivate`、`activate`）可用；River 的第一个定时任务（清理过期会话）按配置运行，nerve 按 M2 设计 3.15 的顺序停机；账户行锁的交错测试 1–3 在真实数据库上通过，六个交错测试至此全部通过；存储的哈希不可用时登录的耗时不变；A12–A14、A16、A17 的接口版本通过。

**Architecture:** 平台加 `platform/jobs`：服务用的 River 客户端（只导入 River 和 pgx），数据库暂时不可达时在后台重试启动，停止时给正在运行的任务 `jobs.shutdown_timeout`。迁移 `00005` 是 River 主线第 2–7 版的原样导出。`identity` 按端口与适配器分层：`app` 加清理用例和管理员的五个用例（与注册共用"建账户"一步，停用与自助停用共用写入），`adapter/postgres` 加按邮箱锁账户行、改邮箱、恢复、撤销全部 PAT、数可用的 PAT、分批删除过期会话，`adapter/river` 是清理任务的 worker；模块入口导出 `Jobs()` 和只用连接池组合的 `NewAdmin`。`bootstrap` 让 HTTP 和任务一起运行并按顺序停机，`Users` 是命令行的最小组合（连接池和 `Admin`，没有 River 客户端）；`cmd/nerve` 只解析 `nerve users` 的参数、读密码。

**Tech Stack:** Go 1.27.1、pgx v5.11.0、River v0.47.0 与 riverpgxv5 v0.47.0（新加）、golang.org/x/term v0.46.0（由间接依赖改为直接依赖）、sqlc v1.31.1（`CGO_ENABLED=0`）、oapi-codegen v2.8.0、goose v3.28.0、golangci-lint 2.13.2、PostgreSQL 18.6；Node 24、pnpm 11.10.0、Playwright 1.63.0。

**Spec:** `docs/v0/M2-auth/specs/P3b-jobs-and-admin.md`（上级：`docs/v0/M2-auth/M2-design.md`）

## Global Constraints

- **Go 版本**：`server/go.mod` 和 `server/tools/go.mod` 保持 `go 1.27` 和 `toolchain go1.27.1`。只有两处改依赖：Task 3 的 `go -C server get github.com/riverqueue/river@v0.47.0 github.com/riverqueue/river/riverdriver/riverpgxv5@v0.47.0` 加 `go -C server mod tidy`；Task 8 的 `go -C server get golang.org/x/term@v0.46.0` 加 `go -C server mod tidy`。**每次**之后都执行 `grep -n "^go \|^toolchain" server/go.mod server/tools/go.mod`，四行必须是 `go 1.27` 和 `toolchain go1.27.1`（golangci-lint 2.13.2 由 go1.27.0 构建，`go` 行更高就拒绝运行）。本 plan 结束时，`server/go.mod`、`go.sum` 对 `d6f313b` 的差异只有 Task 3 和 Task 8 的差异块；`server/tools/go.mod`、`go.sum` 不变。
- **依赖**：只加 `github.com/riverqueue/river` v0.47.0 和 `github.com/riverqueue/river/riverdriver/riverpgxv5` v0.47.0（M2 设计 3.15、6.6），它们带进间接依赖 `riverdriver`、`rivershared`、`rivertype`（都是 v0.47.0）和 `github.com/tidwall/{gjson v1.19.0, match v1.2.0, pretty v1.2.1, sjson v1.2.5}`；`golang.org/x/term` v0.46.0 本来就在 `go.sum` 里，Task 8 把它写进 `require`。River 的命令行 `github.com/riverqueue/river/cmd/river` **锁定 v0.47.0**，只在 Task 2 用 `go run …@v0.47.0` 导出迁移 SQL，不进任何 `go.mod`。不加 npm 包。
- **每个 Task 提交前**：`make lint-go` 两段都输出 `0 issues.`，`make test` 全部通过。有生成物的 Task（2、4、6、8）核对生成物的 SHA-256（`shasum -a 256`）和行数；Task 4、6、8 的生成物由 `make gen` 生成，**提交之后**执行 `make gen-check`（它用 `git status` 判断生成物是否已提交，提交之前必然报差异）。Task 2 的迁移由锁定版本的 River 命令行导出（Task 2 的命令），不由 `make gen` 生成。改了 `schema.gen.ts` 或 `e2e/` 的 Task（8、11）另执行 `make lint-web`、`make knip` 和 `make e2e`；Task 12 执行 `make lint-web`（关键词守卫）。
- **生成的文件不手写、不从本 plan 复制**：执行生成命令，提交它的输出。表中是生成物的 SHA-256 和行数；对不上时停下来，说明某个输入与本 plan 不一致。
- **容器**：`make test` 和 `make e2e` 用自己的 testcontainers。开发库 `nerve-dev-db-1` 可以用，但不要停止或重建它，不要执行 `make dev-db-down`、`make dev-db-reset`。不要碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`）。
- **git**：每次 Bash 调用只执行一个 git 命令，不用 `;`、`&&`、`|` 串联 git；不用 `git -C`、`stash`、`clean`、`reset --hard`。`cd` 不与别的命令组合。
- **安装**：除了 Docker、Go、Node 不做任何全局安装；不执行 `corepack enable`（pnpm 已在 PATH 上）。
- **规则**：不写 `init()`，不用全局可变状态，构造函数显式传入依赖；不建 `utils`、`common`、`helpers` 包；一个文件只做一件事，不超过约 400 行；依赖只能向内；平台包之间不互相导入（规则 7：`platform/jobs` 只导入 River 和 pgx）；不留没有使用者的代码。**例外**：接口描述按模块一个文件（M0-P3 交接 5），`api/modules/identity.yaml` 超过 400 行不拆；`config/load_test.go` 的分层用例表因两个新键到 403 行，不拆（一张表拆开反而难读）。
- **注释**：Go、TS 代码、SQL 查询和接口描述用英文；迁移文件、`server/configs/*.yaml` 的中文注释和中文文档照本 plan 原样。
- **代码块**：标为"新文件"或"完整内容"的块是**完整的文件内容**，照原样写入，不要改动（原型中逐字节运行过）；标为"差异"的块是对这个文件当前版本（`d6f313b` 或前一个 Task 写的版本）的统一差异，照差异修改，改完的文件与原型逐字节相同。差异块可以存成文件后在仓库根目录用 `patch -p1 < <文件>` 应用（块里的路径是 `a/<路径>`、`b/<路径>`；拼 plan 的脚本已用 `patch -p1` 逐个核对过，48 个差异块都得出原型中的文件）。
- **过渡版本**：少数文件先在较早的 Task 写成过渡版本，较晚的 Task 再写成最终版本；每个过渡版本都在原型的逐 Task 复现中运行过（spec 附录 A）。
- **提交**：提交信息用英文，末尾加一行：`Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明。

## 文件结构

路径相对于仓库根目录。"生成"表示由命令生成并提交；"过渡"表示这个 Task 写过渡版本，后面的 Task 写最终版本。

| 文件 | 职责 | Task |
|---|---|---|
| `server/internal/platform/config/config.go`、`validate.go` 及三个测试（修改） | `auth.session_cleanup_interval`、`jobs.shutdown_timeout` | 1 |
| `server/configs/config.yaml`（修改） | 两个新键的默认值；注册一条的注释 | 1（过渡）、8 |
| `server/configs/config.test.yaml`、`embed_test.go`（修改） | test 的清理间隔 2 秒 | 1 |
| `server/migrations/sql/00005_river_main_v2_to_v7.sql`（生成） | River 主线第 2–7 版 | 2 |
| `server/migrations/schema_test.go`（修改） | 五个迁移 up、down、再 up；River 的表、枚举和函数 | 2 |
| `server/internal/archtest/sqlc_test.go`、`sqlc_cases_test.go`（修改） | `CREATE UNLOGGED TABLE` 也算建表 | 2 |
| `server/go.mod`、`server/go.sum`（修改） | River v0.47.0；`golang.org/x/term` 进 `require` | 3（`go.mod` 过渡）、8 |
| `server/internal/platform/jobs/jobs.go`、`jobs_test.go` | 服务用的 River 客户端 | 3 |
| `server/internal/modules/identity/adapter/postgres/queries/sessions.sql`、`sessions.go`（修改）、`cleanup_test.go` | 分批删除过期会话（`SKIP LOCKED`） | 4 |
| `server/internal/modules/identity/adapter/postgres/gen/sessions.sql.go`（生成） | | 4 |
| `server/internal/modules/identity/app/ports.go`（修改） | 清理、管理员命令的端口 | 4（过渡）、6 |
| `server/internal/modules/identity/app/cleanup_sessions.go`、`cleanup_sessions_test.go` | 清理用例 | 4 |
| `server/internal/modules/identity/adapter/river/cleanup.go`、`cleanup_test.go` | 清理任务的 worker 和定时任务 | 4 |
| `server/internal/modules/identity/module.go`（修改） | `Deps.SessionCleanupInterval`、`Jobs()` | 4 |
| `server/internal/bootstrap/app.go`（修改） | 任务的接线、运行和停机顺序；连接池关闭的时限；`passwordHashing` | 5（过渡）、8 |
| `server/internal/bootstrap/app_test.go`（修改）、`jobs_test.go` | 清理按配置运行；停机顺序；连接池不挂住 | 5 |
| `server/cmd/nerve/main_test.go`（修改） | `serve` 的日志顺序；标准输入 | 5（过渡）、8 |
| `server/internal/modules/identity/adapter/postgres/queries/users.sql`、`api_tokens.sql`、`users.go`、`api_tokens.go`（修改）、`admin_test.go` | 管理员命令的查询和存储 | 6 |
| `server/internal/modules/identity/adapter/postgres/gen/users.sql.go`、`api_tokens.sql.go`（生成） | | 6 |
| `server/internal/modules/identity/domain/user.go`、`user_test.go`、`session.go`、`errors.go`（修改） | `NewEmail`；两个撤销原因；两个模块错误 | 7 |
| `server/internal/modules/identity/app/create_user.go`、`reset_password.go`、`set_email.go`、`activate.go` 及测试、`fakes_admin_test.go` | 管理员的四个用例 | 7 |
| `server/internal/modules/identity/app/register.go`、`deactivate.go`、`deactivate_test.go`（修改） | 注册与创建共用"建账户"；按邮箱停用 | 7 |
| `server/internal/modules/identity/admin.go` | `NewAdmin`：只用连接池的组合 | 8 |
| `server/internal/bootstrap/users.go`、`users_test.go` | 命令行的最小组合、输出的一行、错误的一行 | 8 |
| `server/cmd/nerve/users.go`、`users_test.go`、`commands.go`、`main.go`（修改） | `nerve users` 的五个命令；读密码 | 8 |
| `api/modules/identity.yaml`（修改） | `updateMe`、`deactivateMe` 的说明写上命令 | 8 |
| `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`（生成） | | 8 |
| `server/internal/modules/identity/adapter/argon2/hasher.go`、`hasher_test.go`（修改） | 不可用的密码：同样的工作量，不匹配 | 9 |
| `server/internal/bootstrap/sessions_test.go`（修改） | 三种登录耗时相同 | 9 |
| `server/internal/modules/identity/interleavings_test.go`（修改）、`interleavings_reset_test.go` | 交错测试 1–3 | 10 |
| `e2e/fixtures/server.ts`、`e2e/fixtures/assert/identity.ts`（修改）、`e2e/fixtures/users.ts` | 命令的标准输入；管理员命令；断言 | 11 |
| `e2e/stories/identity/a12-deactivate.spec.ts`、`a13-reset-password.spec.ts`、`a14-session-cleanup.spec.ts`、`a16-set-email.spec.ts`、`a17-create-user.spec.ts` | 五个故事 | 11 |
| `docs/v0/v0-design.md`、`docs/v0/plane-diff.md`、`README.md`、`docs/v0/M2-auth/handoffs/{M0-P2,M0-P6,M1-P3}-*.md`（修改） | 文档同步、交接 | 12 |

---

### Task 1: 配置：`auth.session_cleanup_interval`、`jobs.shutdown_timeout`

**Files:**
- Modify: `server/internal/platform/config/config.go`、`validate.go`、`config_test.go`、`load_test.go`、`validate_test.go`
- Modify: `server/configs/config.yaml`（过渡）、`config.test.yaml`、`embed_test.go`

**Interfaces:**
- Produces（spec 2.3，M2 设计 6.5）：
  - `config.AuthConfig.SessionCleanupInterval time.Duration`（`auth.session_cleanup_interval`，默认 `1h`，test 配置 `2s`）；
  - `config.JobsConfig{ShutdownTimeout time.Duration}`（`jobs.shutdown_timeout`，默认 `10s`），`Config.Jobs`；
  - 校验：清理间隔至少 1 秒（River 的定时任务最多每秒一次，`auth.session_cleanup_interval: must be at least 1s, got …`）；`jobs.shutdown_timeout` 为正（`must be positive, got …`）；
  - `Config.LogValue` 加上 `auth.session_cleanup_interval` 和 `jobs.shutdown_timeout`。
- 使用者：清理间隔在 Task 4 交给清理任务，停止时限在 Task 3、5 交给 `platform/jobs`。

**Tests:**
- `load_test.go`：`TestLoadAppliesLayersInOrder` 的 YAML 写上两个新键，两个环境变量（`NERVE_AUTH__SESSION_CLEANUP_INTERVAL=90s`、`NERVE_JOBS__SHUTDOWN_TIMEOUT=7s`）覆盖它们。
- `validate_test.go`：`validConfig` 的两个新值（`3h`、`12s`）都不同于默认值、也不同于 `server.shutdown_timeout`，日志从错的字段取值时测试失败；`TestValidateReportsEveryInvalidKey` 加上两个零值；`TestValidateCrossKeyRules` 加上 500 毫秒的清理间隔和负的停止时限。
- `config_test.go`：`TestLogValueMasksDatabaseURL` 核对两个新的日志项。
- `server/configs/embed_test.go`：`TestBuiltInProfiles` 核对三个环境的清理间隔（dev、prod `1h`，test `2s`）和停止时限（`10s`）。

- [ ] **Step 1: 配置类型、校验和默认值**

`server/internal/platform/config/config.go`（对 `d6f313b` 的差异）：

```diff
--- a/server/internal/platform/config/config.go
+++ b/server/internal/platform/config/config.go
@@ -24,6 +24,7 @@
 	Database  DatabaseConfig  `koanf:"database"`
 	Auth      AuthConfig      `koanf:"auth"`
 	RateLimit RateLimitConfig `koanf:"ratelimit"`
+	Jobs      JobsConfig      `koanf:"jobs"`
 	Workspace WorkspaceConfig `koanf:"workspace"`
 	Files     FilesConfig     `koanf:"files"`
 	Log       LogConfig       `koanf:"log"`
@@ -68,9 +69,13 @@
 	// RefreshDeadline bounds the statements of a refresh or a logout; with
 	// database.commit_timeout it must end before the web client gives up on
 	// a refresh (M2 design 3.5).
-	RefreshDeadline time.Duration  `koanf:"refresh_deadline"`
-	JWT             JWTConfig      `koanf:"jwt"`
-	Password        PasswordConfig `koanf:"password"`
+	RefreshDeadline time.Duration `koanf:"refresh_deadline"`
+	// SessionCleanupInterval is how often the periodic job deletes the
+	// expired sessions (M2 design 3.15). River runs periodic jobs at most
+	// once a second.
+	SessionCleanupInterval time.Duration  `koanf:"session_cleanup_interval"`
+	JWT                    JWTConfig      `koanf:"jwt"`
+	Password               PasswordConfig `koanf:"password"`
 }
 
 // JWTConfig locates the Ed25519 signing key.
@@ -123,6 +128,14 @@
 	return slog.GroupValue(slog.Int("per_minute", b.PerMinute), slog.Int("burst", b.Burst))
 }
 
+// JobsConfig configures the background jobs (M2 design 3.15).
+type JobsConfig struct {
+	// ShutdownTimeout is how long nerve waits at shutdown for the running
+	// jobs, after the HTTP server has stopped; then their contexts are
+	// cancelled.
+	ShutdownTimeout time.Duration `koanf:"shutdown_timeout"`
+}
+
 // WorkspaceConfig configures workspaces. The instance API reports it to
 // clients; creating workspaces arrives, and honours it, in M3 (M2 design 5.3).
 type WorkspaceConfig struct {
@@ -168,6 +181,7 @@
 			slog.Duration("access_token_ttl", c.Auth.AccessTokenTTL),
 			slog.Duration("session_ttl", c.Auth.SessionTTL),
 			slog.Duration("refresh_deadline", c.Auth.RefreshDeadline),
+			slog.Duration("session_cleanup_interval", c.Auth.SessionCleanupInterval),
 			slog.Group("jwt",
 				slog.Bool("private_key_file_set", c.Auth.JWT.PrivateKeyFile != ""),
 			),
@@ -189,6 +203,9 @@
 			slog.Any("register_ip", c.RateLimit.RegisterIP),
 			slog.Any("password_user", c.RateLimit.PasswordUser),
 		),
+		slog.Group("jobs",
+			slog.Duration("shutdown_timeout", c.Jobs.ShutdownTimeout),
+		),
 		slog.Group("workspace",
 			slog.Bool("creation_enabled", c.Workspace.CreationEnabled),
 		),
```

`server/internal/platform/config/validate.go`（对 `d6f313b` 的差异）：

```diff
--- a/server/internal/platform/config/validate.go
+++ b/server/internal/platform/config/validate.go
@@ -81,6 +81,9 @@
 			c.Database.CommitTimeout, webRefreshTimeout, c.Auth.RefreshDeadline)
 	}
 	c.RateLimit.validate(fail)
+	if c.Jobs.ShutdownTimeout <= 0 {
+		fail("jobs.shutdown_timeout", "must be positive, got %s", c.Jobs.ShutdownTimeout)
+	}
 	if c.Files.SizeLimit < 1 {
 		fail("files.size_limit", "must be at least 1, got %d", c.Files.SizeLimit)
 	}
@@ -104,6 +107,10 @@
 	case a.SessionTTL <= a.AccessTokenTTL:
 		fail("auth.session_ttl", "must be longer than auth.access_token_ttl (%s), got %s", a.AccessTokenTTL, a.SessionTTL)
 	}
+	if a.SessionCleanupInterval < time.Second {
+		// River runs a periodic job at most once a second.
+		fail("auth.session_cleanup_interval", "must be at least 1s, got %s", a.SessionCleanupInterval)
+	}
 	if env == EnvProd && a.JWT.PrivateKeyFile == "" {
 		// The file itself is read when nerve starts (bootstrap), not here.
 		fail("auth.jwt.private_key_file", "is required in prod: a PKCS#8 PEM Ed25519 private key, e.g. from openssl genpkey -algorithm ed25519")
```

`server/configs/config.yaml`（对 `d6f313b` 的差异）：

```diff
--- a/server/configs/config.yaml
+++ b/server/configs/config.yaml
@@ -39,6 +39,8 @@
   session_ttl: 720h
   # 续期、退出的语句期限；加上 database.commit_timeout 必须短于前端续期请求的 8 秒超时
   refresh_deadline: 4s
+  # 定时任务删除过期会话的间隔，至少 1 秒
+  session_cleanup_interval: 1h
   jwt:
     # PKCS#8 PEM 格式的 Ed25519 私钥文件：openssl genpkey -algorithm ed25519 -out nerve-jwt.pem
     # prod 必填；dev、test 为空时，启动时生成一把临时密钥（重启后旧的访问令牌失效）
@@ -71,6 +73,11 @@
   # 需要校验密码的已认证操作（M2 中是修改密码），按账户
   password_user: {per_minute: 5, burst: 5}
 
+# 后台任务（River）
+jobs:
+  # 停机时，HTTP 服务停下之后等正在运行的任务结束的时长；到期后取消它们
+  shutdown_timeout: 10s
+
 workspace:
   # 是否允许创建工作区。实例配置接口报告它；创建工作区在 M3 加入时照它执行
   creation_enabled: true
```

`server/configs/config.test.yaml`（对 `d6f313b` 的差异）：

```diff
--- a/server/configs/config.test.yaml
+++ b/server/configs/config.test.yaml
@@ -2,6 +2,8 @@
 auth:
   # 测试环境开放注册（prod 默认关闭，见 config.yaml）
   signup_enabled: true
+  # 端到端测试等清理任务删掉过期的会话（A14），间隔调短
+  session_cleanup_interval: 2s
   password:
     # 测试用最低的 argon2 参数，让大量注册的测试不被哈希拖慢
     argon2_memory_kib: 64
```

- [ ] **Step 2: 测试**

`server/internal/platform/config/config_test.go`（对 `d6f313b` 的差异）：

```diff
--- a/server/internal/platform/config/config_test.go
+++ b/server/internal/platform/config/config_test.go
@@ -39,6 +39,7 @@
 		"config.auth.password.max_concurrent_hashes=4",
 		"config.auth.password.max_wait=2s",
 		"config.auth.refresh_deadline=4s",
+		"config.auth.session_cleanup_interval=3h0m0s",
 		"config.server.trusted_proxies=10.0.0.0/8,2001:db8::/32",
 		"config.ratelimit.ipv6_prefix_len=64",
 		"config.ratelimit.anonymous.per_minute=600",
@@ -50,6 +51,7 @@
 		"config.ratelimit.register_ip.per_minute=10",
 		"config.ratelimit.password_user.per_minute=7",
 		"config.ratelimit.password_user.burst=3",
+		"config.jobs.shutdown_timeout=12s",
 		"config.workspace.creation_enabled=false",
 		"config.files.size_limit=7340032",
 		"config.log.format=json",
```

`server/internal/platform/config/load_test.go`（对 `d6f313b` 的差异）：

```diff
--- a/server/internal/platform/config/load_test.go
+++ b/server/internal/platform/config/load_test.go
@@ -32,6 +32,7 @@
   access_token_ttl: 15m
   session_ttl: 720h
   refresh_deadline: 4s
+  session_cleanup_interval: 1h
   jwt:
     private_key_file: ""
   password:
@@ -49,6 +50,8 @@
   login_ip_email: {per_minute: 10, burst: 5}
   register_ip: {per_minute: 10, burst: 5}
   password_user: {per_minute: 5, burst: 5}
+jobs:
+  shutdown_timeout: 10s
 workspace:
   creation_enabled: true
 files:
@@ -92,6 +95,8 @@
 			"NERVE_SERVER__TRUSTED_PROXIES=10.0.0.0/8,fd00::/8",
 			"NERVE_RATELIMIT__LOGIN_IP__BURST=3",
 			"NERVE_RATELIMIT__PASSWORD_USER__PER_MINUTE=7",
+			"NERVE_AUTH__SESSION_CLEANUP_INTERVAL=90s",
+			"NERVE_JOBS__SHUTDOWN_TIMEOUT=7s",
 			"NERVE_WORKSPACE__CREATION_ENABLED=false",
 			"NERVE_FILES__SIZE_LIMIT=1024",
 		},
@@ -120,10 +125,11 @@
 			CommitTimeout: 2 * time.Second,
 		},
 		Auth: AuthConfig{
-			SignupEnabled:   true, // environment
-			AccessTokenTTL:  15 * time.Minute,
-			SessionTTL:      720 * time.Hour,
-			RefreshDeadline: 4 * time.Second,
+			SignupEnabled:          true, // environment
+			AccessTokenTTL:         15 * time.Minute,
+			SessionTTL:             720 * time.Hour,
+			RefreshDeadline:        4 * time.Second,
+			SessionCleanupInterval: 90 * time.Second, // environment
 			Password: PasswordConfig{
 				Argon2MemoryKiB:     64, // environment
 				Argon2Iterations:    2,
@@ -142,8 +148,9 @@
 			RegisterIP:    BucketConfig{PerMinute: 10, Burst: 5},
 			PasswordUser:  BucketConfig{PerMinute: 7, Burst: 5}, // environment
 		},
-		Workspace: WorkspaceConfig{CreationEnabled: false}, // environment
-		Files:     FilesConfig{SizeLimit: 1024},            // environment
+		Jobs:      JobsConfig{ShutdownTimeout: 7 * time.Second}, // environment
+		Workspace: WorkspaceConfig{CreationEnabled: false},      // environment
+		Files:     FilesConfig{SizeLimit: 1024},                 // environment
 		Log:       LogConfig{Level: "debug", Format: "text"},
 	}
 	if !reflect.DeepEqual(cfg, want) {
```

`server/internal/platform/config/validate_test.go`（对 `d6f313b` 的差异）：

```diff
--- a/server/internal/platform/config/validate_test.go
+++ b/server/internal/platform/config/validate_test.go
@@ -23,10 +23,11 @@
 		},
 		Database: DatabaseConfig{URL: "postgres://nerve:secret@localhost:5432/nerve", MaxConns: 10, CommitTimeout: 2 * time.Second},
 		Auth: AuthConfig{
-			SignupEnabled:   true,
-			AccessTokenTTL:  15 * time.Minute,
-			SessionTTL:      720 * time.Hour,
-			RefreshDeadline: 4 * time.Second,
+			SignupEnabled:          true,
+			AccessTokenTTL:         15 * time.Minute,
+			SessionTTL:             720 * time.Hour,
+			RefreshDeadline:        4 * time.Second,
+			SessionCleanupInterval: 3 * time.Hour,
 			Password: PasswordConfig{
 				Argon2MemoryKiB:     19456,
 				Argon2Iterations:    2,
@@ -45,6 +46,8 @@
 			RegisterIP:    BucketConfig{PerMinute: 10, Burst: 5},
 			PasswordUser:  BucketConfig{PerMinute: 7, Burst: 3},
 		},
+		// Unlike server.shutdown_timeout and the default.
+		Jobs: JobsConfig{ShutdownTimeout: 12 * time.Second},
 		// Unlike the defaults and unlike auth.signup_enabled, so that a key
 		// logged from the wrong field shows.
 		Workspace: WorkspaceConfig{CreationEnabled: false},
@@ -83,6 +86,7 @@
 		"database.commit_timeout: must be positive, got 0s",
 		"auth.access_token_ttl: must be positive, got 0s",
 		"auth.session_ttl: must be positive, got 0s",
+		"auth.session_cleanup_interval: must be at least 1s, got 0s",
 		"auth.jwt.private_key_file: is required in prod: a PKCS#8 PEM Ed25519 private key, e.g. from openssl genpkey -algorithm ed25519",
 		"auth.password.argon2_iterations: must be at least 1, got 0",
 		"auth.password.argon2_parallelism: must be at least 1, got 0",
@@ -105,6 +109,7 @@
 		"ratelimit.register_ip.burst: must be at least 1, got 0",
 		"ratelimit.password_user.per_minute: must be at least 1, got 0",
 		"ratelimit.password_user.burst: must be at least 1, got 0",
+		"jobs.shutdown_timeout: must be positive, got 0s",
 		"files.size_limit: must be at least 1, got 0",
 		`log.level: must be one of debug, info, warn, error, got "verbose"`,
 		`log.format: must be text or json, got "xml"`,
@@ -181,6 +186,17 @@
 			want: "server.trusted_proxies: ::/0 trusts every address, so any client could choose its own IP; list only your proxies' addresses",
 		},
 		{
+			// River runs a periodic job at most once a second.
+			name:   "session cleanup more often than once a second",
+			change: func(c *Config) { c.Auth.SessionCleanupInterval = 500 * time.Millisecond },
+			want:   "auth.session_cleanup_interval: must be at least 1s, got 500ms",
+		},
+		{
+			name:   "negative jobs shutdown timeout",
+			change: func(c *Config) { c.Jobs.ShutdownTimeout = -time.Second },
+			want:   "jobs.shutdown_timeout: must be positive, got -1s",
+		},
+		{
 			name:   "negative file size limit",
 			change: func(c *Config) { c.Files.SizeLimit = -1 },
 			want:   "files.size_limit: must be at least 1, got -1",
```

`server/configs/embed_test.go`（对 `d6f313b` 的差异）：

```diff
--- a/server/configs/embed_test.go
+++ b/server/configs/embed_test.go
@@ -38,17 +38,18 @@
 		addr        string // expected server.addr
 		url         string // expected database.url
 		autoMigrate bool
-		signup      bool   // auth.signup_enabled: closed in prod unless overridden (M2 design decision 2)
-		argon2      uint32 // auth.password.argon2_memory_kib
-		iterations  uint32 // auth.password.argon2_iterations
-		keyFile     string // auth.jwt.private_key_file
+		signup      bool          // auth.signup_enabled: closed in prod unless overridden (M2 design decision 2)
+		argon2      uint32        // auth.password.argon2_memory_kib
+		iterations  uint32        // auth.password.argon2_iterations
+		keyFile     string        // auth.jwt.private_key_file
+		cleanup     time.Duration // auth.session_cleanup_interval
 		limits      config.RateLimitConfig
 		level       string
 		format      string
 	}{
-		{env: "dev", addr: "127.0.0.1:8080", url: devURL, autoMigrate: true, signup: true, argon2: 19456, iterations: 2, limits: defaultLimits, level: "debug", format: "text"},
-		{env: "test", addr: ":8080", url: "postgres://from-env", autoMigrate: true, signup: true, argon2: 64, iterations: 1, limits: testLimits, level: "warn", format: "text"},
-		{env: "prod", addr: ":8080", url: "postgres://from-env", autoMigrate: false, signup: false, argon2: 19456, iterations: 2, keyFile: "/etc/nerve/jwt.pem", limits: defaultLimits, level: "info", format: "json"},
+		{env: "dev", addr: "127.0.0.1:8080", url: devURL, autoMigrate: true, signup: true, argon2: 19456, iterations: 2, cleanup: time.Hour, limits: defaultLimits, level: "debug", format: "text"},
+		{env: "test", addr: ":8080", url: "postgres://from-env", autoMigrate: true, signup: true, argon2: 64, iterations: 1, cleanup: 2 * time.Second, limits: testLimits, level: "warn", format: "text"},
+		{env: "prod", addr: ":8080", url: "postgres://from-env", autoMigrate: false, signup: false, argon2: 19456, iterations: 2, keyFile: "/etc/nerve/jwt.pem", cleanup: time.Hour, limits: defaultLimits, level: "info", format: "json"},
 	}
 	for _, tt := range tests {
 		t.Run(tt.env, func(t *testing.T) {
@@ -77,11 +78,12 @@
 				},
 				Database: config.DatabaseConfig{URL: tt.url, MaxConns: 10, AutoMigrate: tt.autoMigrate, CommitTimeout: 2 * time.Second},
 				Auth: config.AuthConfig{
-					SignupEnabled:   tt.signup,
-					AccessTokenTTL:  15 * time.Minute,
-					SessionTTL:      30 * 24 * time.Hour,
-					RefreshDeadline: 4 * time.Second,
-					JWT:             config.JWTConfig{PrivateKeyFile: tt.keyFile},
+					SignupEnabled:          tt.signup,
+					AccessTokenTTL:         15 * time.Minute,
+					SessionTTL:             30 * 24 * time.Hour,
+					RefreshDeadline:        4 * time.Second,
+					SessionCleanupInterval: tt.cleanup,
+					JWT:                    config.JWTConfig{PrivateKeyFile: tt.keyFile},
 					Password: config.PasswordConfig{
 						Argon2MemoryKiB:     tt.argon2,
 						Argon2Iterations:    tt.iterations,
@@ -91,6 +93,7 @@
 					},
 				},
 				RateLimit: tt.limits,
+				Jobs:      config.JobsConfig{ShutdownTimeout: 10 * time.Second},
 				Workspace: config.WorkspaceConfig{CreationEnabled: true},
 				Files:     config.FilesConfig{SizeLimit: 5 << 20},
 				Log:       config.LogConfig{Level: tt.level, Format: tt.format},
```

- [ ] **Step 3: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/platform/config server/configs
```
```bash
git commit -m "feat(M2/P3b): configure the session cleanup interval and the jobs' shutdown timeout

auth.session_cleanup_interval (1h, 2s in test, at least 1s) and
jobs.shutdown_timeout (10s, positive), both logged.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 两个键能从 YAML 和环境变量读入、校验、记进日志；三个环境的默认值由 `TestBuiltInProfiles` 核对。

---

### Task 2: River 的迁移 `00005`；archtest 认出 `CREATE UNLOGGED TABLE`

**Files:**
- Create（生成）: `server/migrations/sql/00005_river_main_v2_to_v7.sql`
- Modify: `server/migrations/schema_test.go`
- Modify: `server/internal/archtest/sqlc_test.go`、`sqlc_cases_test.go`

**Interfaces:**
- Produces（spec 2.4，M2 设计 3.15、4.1；M0-P2 交接 5）：River 主线第 2–7 版的表 `river_job`、`river_leader`（`UNLOGGED`）、`river_queue`、`river_notification`，枚举 `river_job_state`，函数 `river_job_state_in_bitmask`。版本 1 只建 `river_migration`，River 的迁移命令才用它，不导出。
- `TestSQLCSchemaScope`（M2 设计 3.14）从迁移里找建表语句，认出表的所有者；它原来的正则不认 `CREATE UNLOGGED TABLE`，于是 `00005` 中 `ALTER TABLE river_leader` 被当作"改别的文件建的表"而失败。正则加上可选的 `UNLOGGED`（spec 第 3 节第 2 条）。
- 使用者：Task 3 的客户端、Task 4 的清理任务在这些表上运行；`pgtest` 的模板库经同一条迁移链带上它们。

**Tests:**
- `schema_test.go`：`TestMigrationsGoUpDownAndUpAgain` 改为五个迁移，Up 之后 River 的四张表、枚举和函数都在，Down 之后一个都不剩（`names()` 按三个目录查询：表、枚举、函数），再 Up 成功。
- `sqlc_cases_test.go`：基准布局加上一个带 `CREATE UNLOGGED TABLE river_leader` 和 `ALTER TABLE river_leader` 的 River 迁移，`TestSQLCScopeOfTheBaseLayoutPasses` 仍然通过。

- [ ] **Step 1: 导出迁移**

在仓库根目录执行（`go run …@v0.47.0` 下载锁定版本的 River 命令行，不改任何 `go.mod`）：

```bash
bash -euo pipefail -c '
out=server/migrations/sql/00005_river_main_v2_to_v7.sql
river="go run github.com/riverqueue/river/cmd/river@v0.47.0 migrate-get --line main --all --exclude-version 1"
{
  printf "%s\n" \
    "-- River 的表：主线迁移第 2–7 版（M2 设计 3.15），建出 river_job、river_leader、river_queue、" \
    "-- river_notification 和枚举 river_job_state。第 1 版只建 river_migration，客户端不读它，不导出。" \
    "-- 两段 SQL 由锁定版本的 River 命令行导出，原样粘贴，不手改：" \
    "--   go run github.com/riverqueue/river/cmd/river@v0.47.0 migrate-get --line main --all --exclude-version 1 --up（--down）" \
    "-- 整段 Up、整段 Down 各包在一对 StatementBegin/StatementEnd 里：goose 按分号切分语句，会切断 \$\$ 函数体。" \
    "-- 以后升级 River，用 --version N 导出新增的版本，另写一个迁移，不改这个文件。" \
    "" \
    "-- +goose Up" \
    "-- +goose StatementBegin"
  $river --up
  printf "%s\n" "-- +goose StatementEnd" "" "-- +goose Down" "-- +goose StatementBegin"
  $river --down
  printf "%s\n" "-- +goose StatementEnd"
} >"$out"
'
```

Run: `shasum -a 256 server/migrations/sql/00005_river_main_v2_to_v7.sql` 和 `wc -l server/migrations/sql/00005_river_main_v2_to_v7.sql`
Expected: 与下表相同。

| 生成的文件 | SHA-256 | 行数 |
|---|---|---|
| `server/migrations/sql/00005_river_main_v2_to_v7.sql` | `03e6c6a07d5a8a03addf7482d381f1106dcb53cb0f2164a84abd78e0ec7a9699` | 521 |

前 10 行是这样（其后是 River 的 SQL，原样，不手改）：

```sql
-- River 的表：主线迁移第 2–7 版（M2 设计 3.15），建出 river_job、river_leader、river_queue、
-- river_notification 和枚举 river_job_state。第 1 版只建 river_migration，客户端不读它，不导出。
-- 两段 SQL 由锁定版本的 River 命令行导出，原样粘贴，不手改：
--   go run github.com/riverqueue/river/cmd/river@v0.47.0 migrate-get --line main --all --exclude-version 1 --up（--down）
-- 整段 Up、整段 Down 各包在一对 StatementBegin/StatementEnd 里：goose 按分号切分语句，会切断 $$ 函数体。
-- 以后升级 River，用 --version N 导出新增的版本，另写一个迁移，不改这个文件。

-- +goose Up
-- +goose StatementBegin
-- River main migration 002 [up]
```

- [ ] **Step 2: 迁移测试**

`server/migrations/schema_test.go`（对 `d6f313b` 的差异）：

```diff
--- a/server/migrations/schema_test.go
+++ b/server/migrations/schema_test.go
@@ -25,10 +25,16 @@
 	return pool
 }
 
-func tables(t *testing.T, pool *pgxpool.Pool) []string {
+// The schema's tables (but goose's own), enum types and functions.
+const (
+	tablesQuery    = "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_name <> 'goose_db_version' ORDER BY 1"
+	enumsQuery     = "SELECT typname FROM pg_type WHERE typnamespace = 'public'::regnamespace AND typtype = 'e' ORDER BY 1"
+	functionsQuery = "SELECT proname FROM pg_proc WHERE pronamespace = 'public'::regnamespace ORDER BY 1"
+)
+
+func names(t *testing.T, pool *pgxpool.Pool, query string) []string {
 	t.Helper()
-	rows, err := pool.Query(context.Background(),
-		"SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_name <> 'goose_db_version' ORDER BY 1")
+	rows, err := pool.Query(context.Background(), query)
 	if err != nil {
 		t.Fatal(err)
 	}
@@ -46,7 +52,8 @@
 	return names
 }
 
-// Every migration can go up, down and up again (M2 design 4.1).
+// Every migration can go up, down and up again (M2 design 4.1), River's
+// too: goose runs each of its halves as one statement (M2 design 3.15).
 func TestMigrationsGoUpDownAndUpAgain(t *testing.T) {
 	ctx := context.Background()
 	pool := newPool(t, pgtest.NewEmptyDatabase(t))
@@ -57,22 +64,33 @@
 	t.Cleanup(func() { _ = m.Close() })
 
 	up, err := m.Up(ctx)
-	if err != nil || len(up) != 4 {
-		t.Fatalf("Up() = %d migrations, %v; want 4", len(up), err)
+	if err != nil || len(up) != 5 {
+		t.Fatalf("Up() = %d migrations, %v; want 5", len(up), err)
 	}
-	if got, want := tables(t, pool), []string{"api_tokens", "auth_sessions", "profiles", "users"}; !slices.Equal(got, want) {
-		t.Errorf("tables after Up = %q, want %q", got, want)
+	for _, want := range []struct {
+		query string
+		names []string
+	}{
+		{tablesQuery, []string{"api_tokens", "auth_sessions", "profiles", "river_job", "river_leader", "river_notification", "river_queue", "users"}},
+		{enumsQuery, []string{"river_job_state"}},
+		{functionsQuery, []string{"river_job_state_in_bitmask"}},
+	} {
+		if got := names(t, pool, want.query); !slices.Equal(got, want.names) {
+			t.Errorf("after Up, %s = %q, want %q", want.query, got, want.names)
+		}
 	}
 	for range up {
 		if _, err := m.Down(ctx); err != nil {
 			t.Fatalf("Down() error = %v", err)
 		}
 	}
-	if got := tables(t, pool); len(got) != 0 {
-		t.Errorf("tables after every Down = %q, want none", got)
+	for _, query := range []string{tablesQuery, enumsQuery, functionsQuery} {
+		if got := names(t, pool, query); len(got) != 0 {
+			t.Errorf("after every Down, %s = %q, want none", query, got)
+		}
 	}
-	if again, err := m.Up(ctx); err != nil || len(again) != 4 {
-		t.Errorf("Up() again = %d migrations, %v; want 4", len(again), err)
+	if again, err := m.Up(ctx); err != nil || len(again) != 5 {
+		t.Errorf("Up() again = %d migrations, %v; want 5", len(again), err)
 	}
 }
 
```

- [ ] **Step 3: archtest**

`server/internal/archtest/sqlc_test.go`（对 `d6f313b` 的差异）：

```diff
--- a/server/internal/archtest/sqlc_test.go
+++ b/server/internal/archtest/sqlc_test.go
@@ -97,7 +97,7 @@
 var (
 	migrationFileName = regexp.MustCompile(`^\d{5}_([a-z][a-z0-9]*)_[a-z0-9_]+\.sql$`)
 	sqlComment        = regexp.MustCompile(`--[^\n]*`)
-	createTable       = regexp.MustCompile(`(?i)\bCREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:public\.)?([a-z_][a-z0-9_]*)`)
+	createTable       = regexp.MustCompile(`(?i)\bCREATE\s+(?:UNLOGGED\s+)?TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:public\.)?([a-z_][a-z0-9_]*)`)
 	alterTable        = regexp.MustCompile(`(?i)\bALTER\s+TABLE\s+(?:IF\s+EXISTS\s+)?(?:ONLY\s+)?(?:public\.)?([a-z_][a-z0-9_]*)`)
 	moduleQueries     = regexp.MustCompile(`^internal/modules/([a-z][a-z0-9]*)/adapter/postgres/queries$`)
 )
```

`server/internal/archtest/sqlc_cases_test.go`（对 `d6f313b` 的差异）：

```diff
--- a/server/internal/archtest/sqlc_cases_test.go
+++ b/server/internal/archtest/sqlc_cases_test.go
@@ -7,7 +7,8 @@
 
 // The spike's layout (M2 design 3.14): identity creates users; a later
 // module, asset, creates assets; identity's own later migration alters
-// users to reference assets; River's migration belongs to no entry.
+// users to reference assets; River's migration belongs to no entry, and
+// alters an unlogged table it creates, as its real one does.
 func sqlcBase() ([]sqlcEntry, []migrationFile, []string) {
 	entries := []sqlcEntry{
 		{
@@ -23,7 +24,8 @@
 	}
 	migrations := []migrationFile{
 		{"00001_identity_users.sql", "-- +goose Up\nCREATE TABLE users (id uuid PRIMARY KEY);\n-- +goose Down\nDROP TABLE users;\n"},
-		{"00005_river_main_v2_to_v7.sql", "-- +goose Up\nCREATE TABLE river_job (id bigint);\nALTER TABLE river_job ADD COLUMN x int;\n"},
+		{"00005_river_main_v2_to_v7.sql", "-- +goose Up\nCREATE TABLE river_job (id bigint);\nCREATE UNLOGGED TABLE river_leader (name text);\n" +
+			"ALTER TABLE river_job ADD COLUMN x int;\nALTER TABLE river_leader ADD COLUMN y int;\n"},
 		{"00020_asset_assets.sql", "-- +goose Up\nCREATE TABLE IF NOT EXISTS assets (id uuid PRIMARY KEY);\n"},
 		{"00021_identity_users_avatar_asset.sql", "-- +goose Up\n-- ALTER TABLE assets would be wrong; a comment is not SQL\n" +
 			"ALTER TABLE users ADD COLUMN avatar_asset_id uuid REFERENCES assets ON DELETE SET NULL;\n" +
```

- [ ] **Step 4: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`；`server/migrations` 的 `TestMigrationsGoUpDownAndUpAgain` 覆盖五个迁移。River 的迁移不在 `server/sqlc.yaml` 的任何 `schema` 中，sqlc 的生成物不受影响。

- [ ] **Step 5: 提交**

```bash
git add server/migrations server/internal/archtest
```
```bash
git commit -m "feat(M2/P3b): River's tables as migration 00005, exported by the pinned River CLI

Main-line versions 2 to 7, Up and Down each wrapped in one goose statement.
TestSQLCSchemaScope recognises CREATE UNLOGGED TABLE as a table's creation.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 五个迁移 up、down、再 up 都通过；迁移文件的 SHA-256 与表中相同；`TestSQLCSchemaScope` 通过。

---

### Task 3: `platform/jobs`：服务用的 River 客户端

**Files:**
- Create: `server/internal/platform/jobs/jobs.go`、`jobs_test.go`
- Modify: `server/go.mod`（过渡）、`server/go.sum`

**Interfaces:**
- Produces（spec 2.5，M2 设计 3.15）：
  - `jobs.Job{Add func(*river.Workers) error; Periodic *river.PeriodicJob}`：模块的 river 适配器造它；
  - `jobs.Config{ShutdownTimeout time.Duration; Logger *slog.Logger}`；
  - `jobs.New(pool *pgxpool.Pool, cfg Config, jobs []Job) (*Runner, error)`：默认队列最多 2 个 worker，定时任务交给 River（由 leader 投递，多台服务也只投一次），`SoftStopTimeout` 是 `cfg.ShutdownTimeout`；两个 worker 同一种任务时报错；
  - `(*Runner).Start(ctx)`：在后台启动，立即返回。River 的 `Start` 在数据库不可达时失败（spec 第 3 节第 1 条），runner 就间隔 1 秒、加倍到 30 秒重试，每次记 WARN "jobs did not start; trying again"（`error`、`retry_in`）；成功时记 INFO "jobs started"（`shutdown_timeout`）。客户端运行在 `context.WithoutCancel(ctx)` 上，调用者的 ctx 结束不停止它，只有 `Stop` 能停；
  - `(*Runner).Stop(ctx) error`：结束重试；已启动时停止客户端：不再取任务，正在运行的任务有 `ShutdownTimeout` 结束，之后取消它们的 ctx，再等 1 秒（`cancelGrace`）；还不返回就报错（`jobs still running 1s after jobs.shutdown_timeout (…)`），不挂住停机。成功时记 INFO "jobs stopped"。
  - 只导入 River 和 pgx，不导入别的平台包（archtest 规则 7）。
- 使用者：Task 5 的 `bootstrap`。

**Tests:**（`jobs_test.go`，同包；真实数据库；每个等待都有期限）
- `TestRunnerWorksAPeriodicJobUntilStopped`：定时任务启动时运行一次，之后每个间隔一次；`Stop` 返回之后不再运行；日志有 "jobs started"、"jobs stopped"。
- `TestStopCancelsARunningJobAfterTheShutdownTimeout`：两个设置（1 秒、3 秒，并行）：正在运行的任务在时限到期时收到取消，`Stop` 在它返回后返回，耗时在时限与时限加 1 秒之间。
- `TestStopGivesUpOnAJobThatIgnoresCancellation`：不理会取消的任务不挂住 `Stop`：时限加 1 秒后报错。
- `TestStartWithoutADatabase`：数据库不可达时 `Start` 立即返回，日志有重试的 WARN，`Stop` 立即返回 nil。
- `TestNewRejectsTwoWorkersOfOneKind`。
- `TestStartTriesAgainUntilTheClientStarts`（假客户端，前三次失败）：等待是 10、20、25 毫秒（加倍，封顶）；调用者的 ctx 取消后客户端的 ctx 仍然有效；"jobs started" 带 `shutdown_timeout=7s`；`Stop` 给客户端的期限是时限加 1 秒（误差 100 毫秒以内）。
- `TestStopEndsTheAttemptsToStart`：一直失败时，`Stop` 不等完两次重试之间的停顿，也不调用客户端的 `Stop`。

- [ ] **Step 1: 客户端**

`server/internal/platform/jobs/jobs.go`（新文件）：

```go
// Package jobs runs the modules' background jobs on River (M2 design 3.15):
// the server's client, which works the jobs and enqueues the periodic ones.
// It imports River and pgx, and no other platform package.
package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

// Job is one kind of job a module works: its worker and, when the job is
// periodic, its schedule. A module's river adapter builds it.
type Job struct {
	// Add registers the job's worker, e.g. with river.AddWorkerSafely.
	Add func(*river.Workers) error
	// Periodic, when set, enqueues the job on its schedule. River's leader
	// enqueues it, so one runs per schedule however many servers there are.
	Periodic *river.PeriodicJob
}

// Config is the runner's settings.
type Config struct {
	// ShutdownTimeout is how long Stop lets the running jobs finish before
	// it cancels their contexts: jobs.shutdown_timeout.
	ShutdownTimeout time.Duration
	Logger          *slog.Logger
}

const (
	// maxWorkers is how many jobs run at once. M2 has one periodic job.
	maxWorkers = 2
	// cancelGrace is how long Stop waits, once ShutdownTimeout has passed and
	// the jobs' contexts are cancelled, for the jobs to return.
	cancelGrace = time.Second
	// The first and the longest wait between two attempts to start.
	firstRetry = time.Second
	lastRetry  = 30 * time.Second
)

// client is what the runner uses of *river.Client.
type client interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// Runner is the server's River client.
type Runner struct {
	client          client
	logger          *slog.Logger
	shutdownTimeout time.Duration
	firstRetry      time.Duration
	lastRetry       time.Duration

	cancel  context.CancelFunc // ends the attempts to start
	done    chan struct{}      // closed once the attempts have ended
	started bool               // written before done is closed
}

// New builds the client on pool for jobs; Start runs it.
func New(pool *pgxpool.Pool, cfg Config, jobs []Job) (*Runner, error) {
	workers := river.NewWorkers()
	var periodic []*river.PeriodicJob
	for _, j := range jobs {
		if err := j.Add(workers); err != nil {
			return nil, fmt.Errorf("register a job: %w", err)
		}
		if j.Periodic != nil {
			periodic = append(periodic, j.Periodic)
		}
	}
	c, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Logger:       cfg.Logger,
		Queues:       map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: maxWorkers}},
		Workers:      workers,
		PeriodicJobs: periodic,
		// Stopping lets the running jobs finish for this long, then cancels
		// their contexts.
		SoftStopTimeout: cfg.ShutdownTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("create the jobs client: %w", err)
	}
	return newRunner(c, cfg), nil
}

func newRunner(c client, cfg Config) *Runner {
	return &Runner{client: c, logger: cfg.Logger, shutdownTimeout: cfg.ShutdownTimeout, firstRetry: firstRetry, lastRetry: lastRetry}
}

// Start starts the client in the background and returns at once. River's
// Start fails while the database cannot be reached; the runner then tries
// again, waiting from firstRetry up to lastRetry between attempts, so that
// nerve serves meanwhile and /readyz reports the database, as without jobs.
// Call Start once, and Stop after it.
func (r *Runner) Start(ctx context.Context) {
	ctx, r.cancel = context.WithCancel(context.WithoutCancel(ctx))
	r.done = make(chan struct{})
	go func() {
		defer close(r.done)
		for wait := r.firstRetry; ; wait = min(2*wait, r.lastRetry) {
			err := r.client.Start(ctx)
			if err == nil {
				r.started = true
				r.logger.InfoContext(ctx, "jobs started", slog.Duration("shutdown_timeout", r.shutdownTimeout))
				return
			}
			if ctx.Err() != nil {
				return
			}
			r.logger.WarnContext(ctx, "jobs did not start; trying again", slog.Any("error", err), slog.Duration("retry_in", wait))
			select {
			case <-ctx.Done():
				return
			case <-time.After(wait):
			}
		}
	}()
}

// Stop ends the attempts to start and stops the client: no job is fetched
// any more, the running ones get ShutdownTimeout to finish and then their
// contexts are cancelled. It returns once they have returned, or with an
// error when they still run cancelGrace after that.
func (r *Runner) Stop(ctx context.Context) error {
	r.cancel()
	<-r.done
	if !r.started {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), r.shutdownTimeout+cancelGrace)
	defer cancel()
	if err := r.client.Stop(ctx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			err = fmt.Errorf("jobs still running %s after jobs.shutdown_timeout (%s): %w", cancelGrace, r.shutdownTimeout, err)
		}
		return err
	}
	r.logger.InfoContext(ctx, "jobs stopped")
	return nil
}
```

- [ ] **Step 2: 依赖**

Run: `go -C server get github.com/riverqueue/river@v0.47.0 github.com/riverqueue/river/riverdriver/riverpgxv5@v0.47.0`

Run: `go -C server mod tidy`

Run: `grep -n "^go \|^toolchain" server/go.mod server/tools/go.mod`
Expected: 四行，`go 1.27` 和 `toolchain go1.27.1` 各两行。

`server/go.mod`、`server/go.sum` 应与下面的差异相同：

`server/go.mod`（对 `d6f313b` 的差异）：

```diff
--- a/server/go.mod
+++ b/server/go.mod
@@ -16,6 +16,8 @@
 	github.com/oapi-codegen/nullable v1.2.0
 	github.com/oapi-codegen/runtime v1.7.0
 	github.com/pressly/goose/v3 v3.28.0
+	github.com/riverqueue/river v0.47.0
+	github.com/riverqueue/river/riverdriver/riverpgxv5 v0.47.0
 	github.com/spf13/cobra v1.10.2
 	github.com/testcontainers/testcontainers-go/modules/postgres v0.44.0
 	golang.org/x/crypto v0.57.0
@@ -70,6 +72,9 @@
 	github.com/opencontainers/go-digest v1.0.0 // indirect
 	github.com/opencontainers/image-spec v1.1.1 // indirect
 	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
+	github.com/riverqueue/river/riverdriver v0.47.0 // indirect
+	github.com/riverqueue/river/rivershared v0.47.0 // indirect
+	github.com/riverqueue/river/rivertype v0.47.0 // indirect
 	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3 // indirect
 	github.com/sethvargo/go-retry v0.4.0 // indirect
 	github.com/shirou/gopsutil/v4 v4.26.6 // indirect
@@ -77,6 +82,10 @@
 	github.com/spf13/pflag v1.0.9 // indirect
 	github.com/stretchr/testify v1.12.1 // indirect
 	github.com/testcontainers/testcontainers-go v0.44.0 // indirect
+	github.com/tidwall/gjson v1.19.0 // indirect
+	github.com/tidwall/match v1.2.0 // indirect
+	github.com/tidwall/pretty v1.2.1 // indirect
+	github.com/tidwall/sjson v1.2.5 // indirect
 	github.com/tklauser/go-sysconf v0.4.0 // indirect
 	github.com/tklauser/numcpus v0.12.0 // indirect
 	github.com/yusufpapurcu/wmi v1.2.4 // indirect
```

`server/go.sum`（对 `d6f313b` 的差异）：

```diff
--- a/server/go.sum
+++ b/server/go.sum
@@ -70,6 +70,8 @@
 github.com/gorilla/mux v1.8.0/go.mod h1:DVbg23sWSpFRCP0SfiEN6jmj59UnW/n46BH5rLB71So=
 github.com/inconshreveable/mousetrap v1.1.0 h1:wN+x4NVGpMsO7ErUn/mUI3vEoE6Jt13X2s0bqwp9tc8=
 github.com/inconshreveable/mousetrap v1.1.0/go.mod h1:vpF70FUmC8bwa3OWnCshd2FqLfsEA9PFc4w1p2J65bw=
+github.com/jackc/pgerrcode v0.0.0-20240316143900-6e2875d9b438 h1:Dj0L5fhJ9F82ZJyVOmBx6msDp/kfd1t9GRfny/mfJA0=
+github.com/jackc/pgerrcode v0.0.0-20240316143900-6e2875d9b438/go.mod h1:a/s9Lp5W7n/DD0VrVoyJ00FbP2ytTPDVOivvn2bMlds=
 github.com/jackc/pgpassfile v1.0.0 h1:/6Hmqy13Ss2zCq62VdNG8tM1wchn8zjSGOBJ6icpsIM=
 github.com/jackc/pgpassfile v1.0.0/go.mod h1:CEx0iS5ambNFdcRtxPj5JhEz+xB6uRky5eyVu/W2HEg=
 github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 h1:iCEnooe7UlwOQYpKFhBabPMi4aNAfoODPEFNiAnClxo=
@@ -150,6 +152,18 @@
 github.com/pressly/goose/v3 v3.28.0/go.mod h1:v26MOuB8bL3kzzrt3Vqhb3R0PRVsl8hFQKdrht/L6Rk=
 github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec h1:W09IVJc94icq4NjY3clb7Lk8O1qJ8BdBEF8z0ibU0rE=
 github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec/go.mod h1:qqbHyh8v60DhA7CoWK5oRCqLrMHRGoxYCSS9EjAz6Eo=
+github.com/riverqueue/river v0.47.0 h1:j8HOEyiOE8gRRhVS2wllKams372TH1WWiH6xCakXHqc=
+github.com/riverqueue/river v0.47.0/go.mod h1:Wgmwx475ZBd8lQnNrJgyG2DWH7BfyNiSAtv0rC9bJBQ=
+github.com/riverqueue/river/riverdriver v0.47.0 h1:qU8VkjdMl9plqeRg57SxsDUM/i/eECaSYejZ7HynC60=
+github.com/riverqueue/river/riverdriver v0.47.0/go.mod h1:NOXl0fUiF1AT/TaQOjdx2A/c0Davn+SKbW8nAXWjfC4=
+github.com/riverqueue/river/riverdriver/riverpgxv5 v0.47.0 h1:5N9nvemhQwbUElMxASw4oEaYJ/v6hiS5Y9VcOQfdC5g=
+github.com/riverqueue/river/riverdriver/riverpgxv5 v0.47.0/go.mod h1:ZboiXXZKC4+fTkxBxGRVmAsCuUu0NPYliqbWYuQAZyw=
+github.com/riverqueue/river/rivershared v0.47.0 h1:jdtFsBexCvLqTXf8wnDnGXvB/eeOtPKQZAmThkjFpLs=
+github.com/riverqueue/river/rivershared v0.47.0/go.mod h1:w8Pi1T+6ypyko5/hs9Mv7IIIKo4fAL9eXYnkVV/Y418=
+github.com/riverqueue/river/rivertype v0.47.0 h1:SzNavtLGR4nMT1QkrEYQ7n96OMatYsn/z3aJWhewmv0=
+github.com/riverqueue/river/rivertype v0.47.0/go.mod h1:XKkcRQR6zm8RR/JQa1Q2ywpj8uXQu21quPa4Lpw1Xhw=
+github.com/robfig/cron/v3 v3.0.1 h1:WdRxkvbJztn8LMz/QEvLN5sBU+xKpSqwwUO1Pjr4qDs=
+github.com/robfig/cron/v3 v3.0.1/go.mod h1:eQICP3HwyT7UooqI/z+Ov+PtYAWygg1TEWWzGIFLtro=
 github.com/rogpeppe/go-internal v1.14.1 h1:UQB4HGPB6osV0SQTLymcB4TgvyWu6ZyliaW0tI/otEQ=
 github.com/rogpeppe/go-internal v1.14.1/go.mod h1:MaRKkUm5W0goXpeCfT7UZI6fk/L7L7so1lCWt35ZSgc=
 github.com/russross/blackfriday/v2 v2.1.0/go.mod h1:+Rmxgy9KzJVeS9/2gXHxylqXiyQDYRxCVz55jmeOWTM=
@@ -177,6 +191,17 @@
 github.com/testcontainers/testcontainers-go v0.44.0/go.mod h1:IcnwQrYTO86xHXu5bvMaBH7ATlbS3Qn1M1QWW3c66rE=
 github.com/testcontainers/testcontainers-go/modules/postgres v0.44.0 h1:8fdv/9y3JMxjQ+ULAcOG8RtgeNu5t9XF9LolSXDuTwM=
 github.com/testcontainers/testcontainers-go/modules/postgres v0.44.0/go.mod h1:CFr2LncGYokw+OKjXcr8ARCKG1SaC2UEnGxFBovE86g=
+github.com/tidwall/gjson v1.14.2/go.mod h1:/wbyibRr2FHMks5tjHJ5F8dMZh3AcwJEMf5vlfC0lxk=
+github.com/tidwall/gjson v1.19.0 h1:xwxm7n691Uf3u5OFjzngavjGTh55KX5q/9w9xHW88JU=
+github.com/tidwall/gjson v1.19.0/go.mod h1:V37/opeE/JbLUOfH0QTXiNez2l0RUjYUhpT4szFQAfc=
+github.com/tidwall/match v1.1.1/go.mod h1:eRSPERbgtNPcGhD8UCthc6PmLEQXEWd3PRB5JTxsfmM=
+github.com/tidwall/match v1.2.0 h1:0pt8FlkOwjN2fPt4bIl4BoNxb98gGHN2ObFEDkrfZnM=
+github.com/tidwall/match v1.2.0/go.mod h1:eRSPERbgtNPcGhD8UCthc6PmLEQXEWd3PRB5JTxsfmM=
+github.com/tidwall/pretty v1.2.0/go.mod h1:ITEVvHYasfjBbM0u2Pg8T2nJnzm8xPwvNhhsoaGGjNU=
+github.com/tidwall/pretty v1.2.1 h1:qjsOFOWWQl+N3RsoF5/ssm1pHmJJwhjlSbZ51I6wMl4=
+github.com/tidwall/pretty v1.2.1/go.mod h1:ITEVvHYasfjBbM0u2Pg8T2nJnzm8xPwvNhhsoaGGjNU=
+github.com/tidwall/sjson v1.2.5 h1:kLy8mja+1c9jlljvWTlSazM7cKDRfJuR/bOJhcY5NcY=
+github.com/tidwall/sjson v1.2.5/go.mod h1:Fvgq9kS/6ociJEDnK0Fk1cpYF4FIW6ZF7LAe+6jwd28=
 github.com/tklauser/go-sysconf v0.4.0 h1:7H0uAN+7RkwWRaxhYXDLqa5V3LPrJeV8wmD9dRUgPQU=
 github.com/tklauser/go-sysconf v0.4.0/go.mod h1:8mTNWyog7H+MpKijp4VmKJAd2bbYQ2zuUwkYRbUArPI=
 github.com/tklauser/numcpus v0.12.0 h1:NR85qdvHA9pFse3x3weVZ0r0ST8R6l5RHbZrlRaqob4=
@@ -197,6 +222,8 @@
 go.opentelemetry.io/otel/sdk/metric v1.46.0/go.mod h1:I1PbKrdVc8Qu8HYVDNtqVIwLwjNrhsV/uFuxfwg8mO4=
 go.opentelemetry.io/otel/trace v1.46.0 h1:OULy7ccdJnZtJ0UDYFOIGaCmiWzJ8Vi2G/Rsu60qs1c=
 go.opentelemetry.io/otel/trace v1.46.0/go.mod h1:J7GAXweO77XSFkB/rmAqk9D6ihszhFjLU+d9WuUxDLI=
+go.uber.org/goleak v1.3.0 h1:2K3zAYmnTNqV73imy9J1T3WC+gmCePx2hEGkimedGto=
+go.uber.org/goleak v1.3.0/go.mod h1:CoHD4mav9JJNrW/WLlf7HGZPjdw8EucARQHekz1X6bE=
 go.uber.org/multierr v1.11.0 h1:blXXJkSxSSfBVBlC76pxqeO+LN3aDfLQo+309xJstO0=
 go.uber.org/multierr v1.11.0/go.mod h1:20+QtiLqy0Nd6FdQB9TLXag12DsQkrbs3htMFfDN80Y=
 go.yaml.in/yaml/v3 v3.0.4/go.mod h1:DhzuOOF2ATzADvBadXxruRBLzYTpT36CKvDb3+aBEFg=
```

- [ ] **Step 3: 测试**

`server/internal/platform/jobs/jobs_test.go`（新文件）：

```go
package jobs

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// logBuffer collects the log lines that River's goroutines and the runner
// write concurrently.
type logBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *logBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *logBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// waitFor fails the test unless the logs hold want within limit.
func (b *logBuffer) waitFor(t *testing.T, want string, limit time.Duration) {
	t.Helper()
	for deadline := time.Now().Add(limit); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if strings.Contains(b.String(), want) {
			return
		}
	}
	t.Fatalf("logs lack %q after %s:\n%s", want, limit, b.String())
}

func newLogger(b *logBuffer) *slog.Logger { return slog.New(slog.NewTextHandler(b, nil)) }

// newPool connects to a database of its own, migrated, River's tables too.
func newPool(t *testing.T, url string) *pgxpool.Pool {
	t.Helper()
	pool, err := postgres.NewPool(context.Background(), config.DatabaseConfig{URL: url, MaxConns: 4})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

type probeArgs struct{}

func (probeArgs) Kind() string { return "jobs_test.probe" }

// probeWorker runs work for every probe job.
type probeWorker struct {
	river.WorkerDefaults[probeArgs]
	work func(ctx context.Context) error
}

func (w *probeWorker) Work(ctx context.Context, _ *river.Job[probeArgs]) error { return w.work(ctx) }

// probeJob is a periodic job every interval, the first at once.
func probeJob(interval time.Duration, work func(ctx context.Context) error) Job {
	return Job{
		Add: func(w *river.Workers) error { return river.AddWorkerSafely(w, &probeWorker{work: work}) },
		Periodic: river.NewPeriodicJob(river.PeriodicInterval(interval),
			func() (river.JobArgs, *river.InsertOpts) { return probeArgs{}, nil },
			&river.PeriodicJobOpts{ID: "jobs_test.probe", RunOnStart: true}),
	}
}

// receive fails the test unless ch yields within limit.
func receive[T any](t *testing.T, ch <-chan T, limit time.Duration, what string) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(limit):
		t.Fatalf("no %s within %s", what, limit)
	}
	var zero T
	return zero
}

// start runs r.Start(ctx) and fails the test unless it returns at once:
// Start must not wait for River, which may not start for a long while.
func start(t *testing.T, r *Runner, ctx context.Context) {
	t.Helper()
	returned := make(chan struct{})
	go func() {
		r.Start(ctx)
		close(returned)
	}()
	receive(t, returned, 100*time.Millisecond, "return from Start")
}

// The periodic job runs at once, then every interval, until Stop; Stop
// returns once the client has stopped, and nothing runs after it.
func TestRunnerWorksAPeriodicJobUntilStopped(t *testing.T) {
	t.Parallel()
	runs := make(chan time.Time, 100)
	var logs logBuffer
	r, err := New(newPool(t, pgtest.NewDatabase(t)), Config{ShutdownTimeout: 5 * time.Second, Logger: newLogger(&logs)},
		[]Job{probeJob(time.Second, func(context.Context) error { runs <- time.Now(); return nil })})
	if err != nil {
		t.Fatal(err)
	}
	start(t, r, context.Background())

	first := receive(t, runs, 15*time.Second, "first run")
	second := receive(t, runs, 10*time.Second, "second run")
	if gap := second.Sub(first); gap < 500*time.Millisecond {
		t.Errorf("runs %s apart, want about the 1s interval", gap)
	}
	if err := r.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() = %v", err)
	}
	for len(runs) > 0 { // a run that was fetched before Stop
		<-runs
	}
	select {
	case at := <-runs:
		t.Errorf("a run at %v after Stop returned", at)
	case <-time.After(1500 * time.Millisecond):
	}
	out := logs.String()
	if !strings.Contains(out, `msg="jobs started"`) || !strings.Contains(out, `msg="jobs stopped"`) {
		t.Errorf("logs lack the start and the stop:\n%s", out)
	}
}

// Stop lets a running job finish for jobs.shutdown_timeout, then cancels
// its context and returns once it has returned. Two settings, so the value
// reaches River whichever it is.
func TestStopCancelsARunningJobAfterTheShutdownTimeout(t *testing.T) {
	t.Parallel()
	for _, timeout := range []time.Duration{time.Second, 3 * time.Second} {
		t.Run(timeout.String(), func(t *testing.T) {
			t.Parallel()
			running, cancelled := make(chan struct{}, 1), make(chan time.Time, 1)
			var logs logBuffer
			r, err := New(newPool(t, pgtest.NewDatabase(t)), Config{ShutdownTimeout: timeout, Logger: newLogger(&logs)},
				[]Job{probeJob(time.Hour, func(ctx context.Context) error {
					running <- struct{}{}
					select {
					case <-ctx.Done():
						cancelled <- time.Now()
						return ctx.Err()
					case <-time.After(15 * time.Second): // never cancelled: end, so the test fails instead of hanging
						return errors.New("the job's context was never cancelled")
					}
				})})
			if err != nil {
				t.Fatal(err)
			}
			start(t, r, context.Background())
			receive(t, running, 15*time.Second, "running job")

			stopping := time.Now()
			err = r.Stop(context.Background())
			took := time.Since(stopping)

			at := receive(t, cancelled, time.Second, "cancellation")
			if err != nil || took < timeout || took > timeout+cancelGrace {
				t.Errorf("Stop() = %v after %s, want nil after %s to %s", err, took, timeout, timeout+cancelGrace)
			}
			if waited := at.Sub(stopping); waited < timeout || waited > timeout+500*time.Millisecond {
				t.Errorf("the job's context was cancelled %s after Stop began, want %s", waited, timeout)
			}
		})
	}
}

// A job that ignores its cancelled context does not hold nerve's shutdown:
// Stop gives up cancelGrace after the timeout.
func TestStopGivesUpOnAJobThatIgnoresCancellation(t *testing.T) {
	t.Parallel()
	running, release := make(chan struct{}, 1), make(chan struct{})
	var logs logBuffer
	r, err := New(newPool(t, pgtest.NewDatabase(t)), Config{ShutdownTimeout: time.Second, Logger: newLogger(&logs)},
		[]Job{probeJob(time.Hour, func(context.Context) error {
			running <- struct{}{}
			<-release
			return nil
		})})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { close(release) }) // before the pool closes: cleanups run last first
	start(t, r, context.Background())
	receive(t, running, 15*time.Second, "running job")

	stopping := time.Now()
	stopped := make(chan error, 1)
	go func() { stopped <- r.Stop(context.Background()) }()
	err = receive(t, stopped, time.Second+cancelGrace+2*time.Second, "return from Stop")
	took := time.Since(stopping)

	if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "after jobs.shutdown_timeout (1s)") ||
		took < time.Second+cancelGrace || took > time.Second+cancelGrace+500*time.Millisecond {
		t.Errorf("Stop() = %v after %s, want the deadline after %s", err, took, time.Second+cancelGrace)
	}
}

// River's Start fails while the database cannot be reached: the runner
// keeps trying in the background, and Stop ends that at once.
func TestStartWithoutADatabase(t *testing.T) {
	var logs logBuffer
	r, err := New(newPool(t, "postgres://nobody@127.0.0.1:1/nowhere"), Config{ShutdownTimeout: 5 * time.Second, Logger: newLogger(&logs)},
		[]Job{probeJob(time.Hour, func(context.Context) error { return nil })})
	if err != nil {
		t.Fatal(err)
	}
	start(t, r, context.Background())
	logs.waitFor(t, `msg="jobs did not start; trying again"`, 5*time.Second)

	stopping := time.Now()
	if err := r.Stop(context.Background()); err != nil || time.Since(stopping) > 500*time.Millisecond {
		t.Errorf("Stop() = %v after %s, want nil at once", err, time.Since(stopping))
	}
	if strings.Contains(logs.String(), `msg="jobs started"`) {
		t.Errorf("logs claim the jobs started:\n%s", logs.String())
	}
}

func TestNewRejectsTwoWorkersOfOneKind(t *testing.T) {
	job := probeJob(time.Hour, func(context.Context) error { return nil })
	_, err := New(newPool(t, "postgres://nobody@127.0.0.1:1/nowhere"), Config{ShutdownTimeout: time.Second, Logger: slog.New(slog.DiscardHandler)},
		[]Job{job, {Add: job.Add}})
	if err == nil || !strings.Contains(err.Error(), "register a job") {
		t.Errorf("New() = %v, want the second worker of jobs_test.probe refused", err)
	}
}

// fakeClient fails Start until fails runs out; it records the contexts
// that Start and Stop get.
type fakeClient struct {
	mu       sync.Mutex
	fails    int
	starts   []context.Context
	stops    []context.Context
	attempts chan struct{}
}

func (f *fakeClient) Start(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.starts = append(f.starts, ctx)
	f.attempts <- struct{}{}
	if f.fails > 0 {
		f.fails--
		return errors.New("dial tcp 127.0.0.1:1: connection refused")
	}
	return nil
}

func (f *fakeClient) Stop(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stops = append(f.stops, ctx)
	return nil
}

// The runner tries again, waiting twice as long each time up to its
// longest wait, until River starts; the client runs on, whatever happens to
// the caller's context, until Stop stops it within the shutdown budget.
func TestStartTriesAgainUntilTheClientStarts(t *testing.T) {
	fake := &fakeClient{fails: 3, attempts: make(chan struct{}, 10)}
	var logs logBuffer
	r := newRunner(fake, Config{ShutdownTimeout: 7 * time.Second, Logger: newLogger(&logs)})
	r.firstRetry, r.lastRetry = 10*time.Millisecond, 25*time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())

	start(t, r, ctx)
	for range 4 {
		receive(t, fake.attempts, 5*time.Second, "attempt to start")
	}
	logs.waitFor(t, `msg="jobs started" shutdown_timeout=7s`, 5*time.Second)
	cancel() // the server's context ends at shutdown, before the jobs stop

	fake.mu.Lock()
	running := fake.starts[3]
	fake.mu.Unlock()
	if running.Err() != nil {
		t.Errorf("the client's context ended with the caller's: %v", running.Err())
	}
	out := logs.String()
	for _, wait := range []string{"retry_in=10ms", "retry_in=20ms", "retry_in=25ms"} {
		if !strings.Contains(out, `msg="jobs did not start; trying again" error="dial tcp 127.0.0.1:1: connection refused" `+wait) {
			t.Errorf("logs lack the retry with %s:\n%s", wait, out)
		}
	}

	stopping := time.Now()
	if err := r.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() = %v", err)
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.stops) != 1 {
		t.Fatalf("Stop() called the client's Stop %d times, want once", len(fake.stops))
	}
	deadline, ok := fake.stops[0].Deadline()
	if budget := deadline.Sub(stopping); !ok || budget < 7*time.Second+cancelGrace || budget > 7*time.Second+cancelGrace+100*time.Millisecond {
		t.Errorf("the client's Stop got %s to stop (deadline set: %v), want 7s and cancelGrace", budget, ok)
	}
}

// Stop ends the attempts to start without waiting out the pause between
// them, and has nothing to stop.
func TestStopEndsTheAttemptsToStart(t *testing.T) {
	fake := &fakeClient{fails: 1000, attempts: make(chan struct{}, 1000)}
	r := newRunner(fake, Config{ShutdownTimeout: time.Second, Logger: slog.New(slog.DiscardHandler)})
	start(t, r, context.Background())
	receive(t, fake.attempts, 5*time.Second, "attempt to start")

	stopping := time.Now()
	if err := r.Stop(context.Background()); err != nil || time.Since(stopping) > 100*time.Millisecond {
		t.Errorf("Stop() = %v after %s, want nil at once, not after the %s pause", err, time.Since(stopping), firstRetry)
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.stops) != 0 || len(fake.starts) != 1 {
		t.Errorf("client started %d times, stopped %d times; want one attempt and no stop", len(fake.starts), len(fake.stops))
	}
}
```

- [ ] **Step 4: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`；`internal/platform/jobs` 的 7 个测试通过，archtest 通过（`platform/jobs` 不导入别的平台包；River 不在禁止链接的名单里）。

- [ ] **Step 5: 提交**

```bash
git add server/go.mod server/go.sum server/internal/platform/jobs
```
```bash
git commit -m "feat(M2/P3b): platform/jobs runs the server's River client

It starts in the background and tries again while the database cannot be
reached; Stop gives running jobs jobs.shutdown_timeout, then cancels them.
River v0.47.0.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** `platform/jobs` 的 7 个测试通过；`go.mod` 仍是 `go 1.27` / `toolchain go1.27.1`，依赖的差异与 plan 相同。

---

### Task 4: 清理过期会话：查询、用例、River 的 worker、模块的 `Jobs()`

**Files:**
- Modify: `server/internal/modules/identity/adapter/postgres/queries/sessions.sql`、`sessions.go`
- Create: `server/internal/modules/identity/adapter/postgres/cleanup_test.go`
- Modify（生成）: `server/internal/modules/identity/adapter/postgres/gen/sessions.sql.go`
- Modify: `server/internal/modules/identity/app/ports.go`（过渡）
- Create: `server/internal/modules/identity/app/cleanup_sessions.go`、`cleanup_sessions_test.go`
- Create: `server/internal/modules/identity/adapter/river/cleanup.go`、`cleanup_test.go`
- Modify: `server/internal/modules/identity/module.go`

**Interfaces:**
- Produces（spec 2.6，M2 设计 3.15）：
  - 查询 `DeleteExpiredSessions :execrows`：`DELETE FROM auth_sessions WHERE id IN (SELECT s.id FROM auth_sessions s WHERE s.expires_at < $now LIMIT $batch FOR UPDATE SKIP LOCKED)`。别的事务锁着的会话（续期、撤销）跳过，下一轮再删；清理不进入 3.5 的加锁顺序。子查询要用别名，不然 sqlc 认为 `expires_at` 有歧义。
  - 端口 `app.ExpiredSessionDeleter{DeleteExpiredSessions(ctx, now, limit int) (int, error)}`；`Store.DeleteExpiredSessions`。
  - `app.NewCleanupSessions(sessions, clock, logger)`；`Execute(ctx) (int, error)`：取一次时钟的 `now`，每批 1000 行，直到某一批不满；删掉了就记 INFO "expired sessions deleted"（`sessions`）；某一批失败时返回错误和之前各批删掉的数目。
  - `riveradapter`（`identity/adapter/river`）：`CleanupKind = "identity.cleanup_expired_sessions"`（也是定时任务的 id）；`CleanupArgs`；`CleanupWorker`（`Work` 只调用用例，失败时 River 重试）；`CleanupJob(uc, interval) jobs.Job`：`river.AddWorkerSafely` 加 `river.NewPeriodicJob(river.PeriodicInterval(interval), …, &river.PeriodicJobOpts{ID: CleanupKind, RunOnStart: true})`。
  - 模块入口：`Deps.SessionCleanupInterval`；`(*Module).Jobs() []jobs.Job`。
- 使用者：Task 5 的 `bootstrap` 把 `Jobs()` 交给 `platform/jobs`。

**Tests:**
- `cleanup_test.go`（真实数据库）：`TestDeleteExpiredSessions`：两个账户的三个过期会话（其中一个已撤销、只早于 `now` 1 微秒），和四个留下的（恰好在 `now` 到期、未到期、已撤销但未到期、bob 未到期）；每批 2 行，三次调用得 2、1、0，最后只剩那四个（`<` 不含 `now`）。`TestDeleteExpiredSessionsSkipsLockedRows`：另一个事务正在更新一个过期会话（持有它的行锁）时，清理（`lock_timeout` 500 毫秒，等锁就会失败）删掉另一个、跳过它；那个事务提交后，下一次删掉它。
- `cleanup_sessions_test.go`：`TestCleanupSessionsDeletesBatchAfterBatch`（没有过期的、不满一批、恰好一批、两批多：调用次数、每次的 `now` 和批大小、返回值、日志）；`TestCleanupSessionsKeepsWhatExpiresFromNowOn`（整个清理用同一个 `now`）；`TestCleanupSessionsWhenABatchFails`（返回错误和已删的数目）。
- `adapter/river/cleanup_test.go`：`TestCleanupKind`；`TestCleanupWorkerDeletesTheExpiredSessions`；`TestCleanupWorkerFailsWithTheCleanup`。

- [ ] **Step 1: 查询**

`server/internal/modules/identity/adapter/postgres/queries/sessions.sql`（对 `d6f313b` 的差异）：

```diff
--- a/server/internal/modules/identity/adapter/postgres/queries/sessions.sql
+++ b/server/internal/modules/identity/adapter/postgres/queries/sessions.sql
@@ -50,3 +50,16 @@
 SET updated_at = sqlc.arg(now), revoked_at = sqlc.arg(now), revoke_reason = 'logout'
 WHERE id = sqlc.arg(id) AND generation = sqlc.arg(generation) AND token_hash = sqlc.arg(token_hash)
   AND revoked_at IS NULL AND expires_at > sqlc.arg(now);
+
+-- name: DeleteExpiredSessions :execrows
+-- The periodic cleanup (M2 design 3.15): up to batch sessions that expired before now. A row
+-- another transaction holds (a refresh, a revocation) is skipped, not waited for: the next run
+-- deletes it, and the cleanup never joins the lock order of 3.5. The subquery needs its alias:
+-- without it sqlc finds expires_at ambiguous.
+DELETE FROM auth_sessions
+WHERE id IN (
+    SELECT s.id FROM auth_sessions s
+    WHERE s.expires_at < sqlc.arg(now)
+    LIMIT sqlc.arg(batch)
+    FOR UPDATE SKIP LOCKED
+);
```

- [ ] **Step 2: 生成**

Run: `make gen`
Expected: 只有 `gen/sessions.sql.go` 改变：

| 生成的文件 | SHA-256 | 行数 |
|---|---|---|
| `server/internal/modules/identity/adapter/postgres/gen/sessions.sql.go` | `1fa0e922d49c28ff5edbb19f6a6ab67170f03d61f401fb47a617bf1d0a6e872a` | 236 |

- [ ] **Step 3: 端口、存储和用例**

`server/internal/modules/identity/app/ports.go`（对 `d6f313b` 的差异）：

```diff
--- a/server/internal/modules/identity/app/ports.go
+++ b/server/internal/modules/identity/app/ports.go
@@ -194,6 +194,14 @@
 	// that is neither revoked nor expired, except keep (uuid.Nil keeps
 	// none), and returns how many it revoked.
 	RevokeSessions(ctx context.Context, userID, keep uuid.UUID, reason domain.RevokeReason, now time.Time) (int, error)
+}
+
+// ExpiredSessionDeleter deletes expired sessions (M2 design 3.15).
+type ExpiredSessionDeleter interface {
+	// DeleteExpiredSessions deletes up to limit sessions that expired
+	// before now, skipping those another transaction has locked, and
+	// returns how many it deleted.
+	DeleteExpiredSessions(ctx context.Context, now time.Time, limit int) (int, error)
 }
 
 // SessionEnder ends sessions at logout.
```

`server/internal/modules/identity/adapter/postgres/sessions.go`（对 `d6f313b` 的差异）：

```diff
--- a/server/internal/modules/identity/adapter/postgres/sessions.go
+++ b/server/internal/modules/identity/adapter/postgres/sessions.go
@@ -102,3 +102,14 @@
 	}
 	return n == 1, nil
 }
+
+// DeleteExpiredSessions deletes up to limit sessions that expired before
+// now, skipping those another transaction has locked, and returns how many
+// it deleted.
+func (s *Store) DeleteExpiredSessions(ctx context.Context, now time.Time, limit int) (int, error) {
+	n, err := s.queries(ctx).DeleteExpiredSessions(ctx, gen.DeleteExpiredSessionsParams{Now: now, Batch: int32(limit)}) // app's batch, 1000
+	if err != nil {
+		return 0, fmt.Errorf("delete expired sessions: %w", err)
+	}
+	return int(n), nil
+}
```

`server/internal/modules/identity/app/cleanup_sessions.go`（新文件）：

```go
package app

import (
	"context"
	"log/slog"
)

// cleanupBatch is the most sessions one statement deletes: each batch is a
// short statement of its own, however many sessions have expired.
const cleanupBatch = 1000

// CleanupSessions deletes the expired sessions: the periodic job
// identity.cleanup_expired_sessions (M2 design 3.15). An expired session
// authenticates nothing and refreshes nothing; only its row is left.
type CleanupSessions struct {
	sessions ExpiredSessionDeleter
	clock    Clock
	logger   *slog.Logger
}

// NewCleanupSessions returns the use case.
func NewCleanupSessions(sessions ExpiredSessionDeleter, clock Clock, logger *slog.Logger) *CleanupSessions {
	return &CleanupSessions{sessions: sessions, clock: clock, logger: logger}
}

// Execute deletes, batch after batch, the sessions that expired before the
// clock's now, until a batch comes back short. A session another
// transaction has locked is skipped and left to the next run. It returns
// how many it deleted, those of the batches before a failure too.
func (u *CleanupSessions) Execute(ctx context.Context) (int, error) {
	now := u.clock.Now()
	deleted := 0
	for {
		n, err := u.sessions.DeleteExpiredSessions(ctx, now, cleanupBatch)
		deleted += n
		if err != nil {
			return deleted, err
		}
		if n < cleanupBatch {
			break
		}
	}
	if deleted > 0 {
		u.logger.InfoContext(ctx, "expired sessions deleted", slog.Int("sessions", deleted))
	}
	return deleted, nil
}
```

- [ ] **Step 4: River 的 worker 和模块的 `Jobs()`**

`server/internal/modules/identity/adapter/river/cleanup.go`（新文件）：

```go
// Package riveradapter works the identity module's background jobs on River
// (M2 design 3.15): each worker only runs a use case.
package riveradapter

import (
	"context"
	"time"

	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
)

// CleanupKind is the kind of the job that deletes the expired sessions, and
// the id of its schedule.
const CleanupKind = "identity.cleanup_expired_sessions"

// CleanupUseCase is the use case the job runs: app.CleanupSessions.
type CleanupUseCase interface {
	Execute(ctx context.Context) (int, error)
}

// CleanupArgs are the job's arguments: none.
type CleanupArgs struct{}

// Kind is CleanupKind.
func (CleanupArgs) Kind() string { return CleanupKind }

// CleanupWorker runs the cleanup.
type CleanupWorker struct {
	river.WorkerDefaults[CleanupArgs]
	uc CleanupUseCase
}

// NewCleanupWorker returns the worker of uc.
func NewCleanupWorker(uc CleanupUseCase) *CleanupWorker {
	return &CleanupWorker{uc: uc}
}

// Work deletes the expired sessions. A failure makes River retry the job.
func (w *CleanupWorker) Work(ctx context.Context, _ *river.Job[CleanupArgs]) error {
	_, err := w.uc.Execute(ctx)
	return err
}

// CleanupJob is the periodic cleanup: when the server starts, then every
// interval (auth.session_cleanup_interval).
func CleanupJob(uc CleanupUseCase, interval time.Duration) jobs.Job {
	return jobs.Job{
		Add: func(w *river.Workers) error { return river.AddWorkerSafely(w, NewCleanupWorker(uc)) },
		Periodic: river.NewPeriodicJob(river.PeriodicInterval(interval),
			func() (river.JobArgs, *river.InsertOpts) { return CleanupArgs{}, nil },
			&river.PeriodicJobOpts{ID: CleanupKind, RunOnStart: true}),
	}
}
```

`server/internal/modules/identity/module.go`（对 `d6f313b` 的差异）：

```diff
--- a/server/internal/modules/identity/module.go
+++ b/server/internal/modules/identity/module.go
@@ -1,7 +1,8 @@
 // Package identity is the accounts module (M2 design 3.3, 6.2): accounts,
 // profiles, sessions and personal access tokens. It brings registration,
-// login, refresh, logout, the caller's account, preferences and tokens, and
-// the authentication every other operation goes through.
+// login, refresh, logout, the caller's account, preferences and tokens, the
+// authentication every other operation goes through, and the job that
+// deletes the expired sessions.
 package identity
 
 import (
@@ -17,10 +18,12 @@
 	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/authn"
 	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/http"
 	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
+	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/river"
 	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/signing"
 	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
 	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
 	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
+	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
 	"github.com/open-nerve/NerveProject/server/internal/platform/ratelimit"
 	"github.com/open-nerve/NerveProject/server/internal/shared"
 )
@@ -39,8 +42,10 @@
 	AccessTokenTTL  time.Duration
 	SessionTTL      time.Duration
 	RefreshDeadline time.Duration // auth.refresh_deadline
-	Password        PasswordHashing
-	RateLimits      RateLimits
+	// SessionCleanupInterval is auth.session_cleanup_interval.
+	SessionCleanupInterval time.Duration
+	Password               PasswordHashing
+	RateLimits             RateLimits
 }
 
 // PasswordHashing is auth.password: argon2id's parameters and the limits on
@@ -68,6 +73,7 @@
 	uc            httpadapter.UseCases
 	settings      httpadapter.Settings
 	authenticator *authn.Authenticator
+	jobs          []jobs.Job
 }
 
 // New wires the module. A signing key that cannot be parsed is an error
@@ -131,6 +137,9 @@
 		authenticator: authn.New(app.NewAuthenticate(app.AuthenticateDeps{
 			AccessTokens: tokens, Sessions: store, APITokens: store, Touch: store, Clock: d.Clock, Logger: d.Logger,
 		})),
+		jobs: []jobs.Job{
+			riveradapter.CleanupJob(app.NewCleanupSessions(store, d.Clock, d.Logger), d.SessionCleanupInterval),
+		},
 	}, nil
 }
 
@@ -157,6 +166,11 @@
 	return m.authenticator
 }
 
+// Jobs are the module's background jobs, for platform/jobs to run.
+func (m *Module) Jobs() []jobs.Job {
+	return m.jobs
+}
+
 // Register mounts the module's API on router behind api's middlewares.
 func (m *Module) Register(router *httpserver.Router, api *httpserver.API) {
 	httpadapter.Register(router, api, m.uc, m.settings)
```

- [ ] **Step 5: 测试**

`server/internal/modules/identity/adapter/postgres/cleanup_test.go`（新文件）：

```go
package postgresadapter_test

import (
	"context"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// sessionUntil inserts a session of u that expires at expires.
func sessionUntil(t *testing.T, s *postgresadapter.Store, u app.NewUser, expires time.Time) uuid.UUID {
	t.Helper()
	n := app.NewSession{ID: uuid.NewV7(), UserID: u.ID, TokenHash: secretHash[:], ExpiresAt: expires, Now: now}
	if err := s.CreateSession(context.Background(), n); err != nil {
		t.Fatal(err)
	}
	return n.ID
}

// sessionIDs lists every session left, sorted.
func sessionIDs(t *testing.T, pool *pgxpool.Pool) []uuid.UUID {
	t.Helper()
	rows, err := pool.Query(context.Background(), "SELECT id FROM auth_sessions")
	if err != nil {
		t.Fatal(err)
	}
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	slices.SortFunc(ids, uuid.UUID.Compare)
	return ids
}

func sorted(ids ...uuid.UUID) []uuid.UUID {
	slices.SortFunc(ids, uuid.UUID.Compare)
	return ids
}

// The cleanup deletes the sessions that expired before now, any account's,
// revoked or not, at most limit a call; what expires at now or later stays.
func TestDeleteExpiredSessions(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newUser("alice@corp.com"), newUser("bob@corp.com")
	mustCreate(t, s, alice)
	mustCreate(t, s, bob)
	sessionUntil(t, s, alice, now.Add(-time.Hour))
	revokedExpired := sessionUntil(t, s, alice, now.Add(-time.Microsecond))
	sessionUntil(t, s, bob, now.Add(-time.Minute))
	atNow := sessionUntil(t, s, alice, now)
	live := sessionUntil(t, s, alice, now.Add(time.Hour))
	revokedLive := sessionUntil(t, s, alice, now.Add(time.Hour))
	bobsLive := sessionUntil(t, s, bob, now.Add(time.Hour))
	exec(t, pool, `UPDATE auth_sessions SET revoked_at = $2, revoke_reason = 'logout' WHERE id = ANY($1)`,
		[]uuid.UUID{revokedExpired, revokedLive}, now.Add(-time.Hour))

	var deleted []int
	for range 3 {
		n, err := s.DeleteExpiredSessions(context.Background(), now, 2)
		if err != nil {
			t.Fatal(err)
		}
		deleted = append(deleted, n)
	}

	if !slices.Equal(deleted, []int{2, 1, 0}) {
		t.Errorf("DeleteExpiredSessions(limit 2) deleted %v in three calls, want [2 1 0]", deleted)
	}
	if got, want := sessionIDs(t, pool), sorted(atNow, live, revokedLive, bobsLive); !slices.Equal(got, want) {
		t.Errorf("sessions left = %v, want %v: those from now on", got, want)
	}
}

// A session another transaction holds, e.g. being refreshed or revoked, is
// skipped rather than waited for (FOR UPDATE SKIP LOCKED); the next run
// deletes it. lock_timeout turns a wait into a failure.
func TestDeleteExpiredSessionsSkipsLockedRows(t *testing.T) {
	s, pool := newStore(t)
	alice := newUser("alice@corp.com")
	mustCreate(t, s, alice)
	held, free := sessionUntil(t, s, alice, now.Add(-time.Hour)), sessionUntil(t, s, alice, now.Add(-time.Hour))
	tx := postgres.NewTxManager(pool, 2*time.Second)
	locked, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- tx.WithinTx(context.Background(), func(ctx context.Context) error {
			if _, err := postgres.DB(ctx, pool).Exec(ctx, "UPDATE auth_sessions SET updated_at = $2 WHERE id = $1", held, later); err != nil {
				return err
			}
			close(locked)
			<-release
			return nil
		})
	}()
	select {
	case <-locked:
	case err := <-done:
		t.Fatalf("locking the session: %v", err)
	}

	var n int
	err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
		if _, err := postgres.DB(ctx, pool).Exec(ctx, "SET LOCAL lock_timeout = '500ms'"); err != nil {
			return err
		}
		var err error
		n, err = s.DeleteExpiredSessions(ctx, now, 1000)
		return err
	})
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("the transaction holding the session: %v", err)
	}

	if err != nil || n != 1 || !slices.Equal(sessionIDs(t, pool), []uuid.UUID{held}) {
		t.Errorf("DeleteExpiredSessions() = %d, %v leaving %v; want %v deleted without waiting, %v left", n, err, sessionIDs(t, pool), free, held)
	}
	if n, err := s.DeleteExpiredSessions(context.Background(), now, 1000); err != nil || n != 1 || len(sessionIDs(t, pool)) != 0 {
		t.Errorf("the next run = %d, %v; want the released session deleted", n, err)
	}
}
```

`server/internal/modules/identity/app/cleanup_sessions_test.go`（新文件）：

```go
package app_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
)

// fakeExpiries is the sessions' expiry times, in memory. It deletes the
// way the query does: up to limit of those before now. failAt fails that
// call, counting from 1.
type fakeExpiries struct {
	expires []time.Time
	calls   []int // the limit of each call
	nows    []time.Time
	failAt  int
}

var errDatabaseGone = errors.New("connection reset")

func (f *fakeExpiries) DeleteExpiredSessions(_ context.Context, now time.Time, limit int) (int, error) {
	f.calls = append(f.calls, limit)
	f.nows = append(f.nows, now)
	if len(f.calls) == f.failAt {
		return 0, errDatabaseGone
	}
	deleted := 0
	f.expires = slices.DeleteFunc(f.expires, func(at time.Time) bool {
		if deleted < limit && at.Before(now) {
			deleted++
			return true
		}
		return false
	})
	return deleted, nil
}

// sessionsExpiring returns n expiry times at, then the live ones.
func sessionsExpiring(n int, at time.Time, live int) []time.Time {
	var out []time.Time
	for range n {
		out = append(out, at)
	}
	for range live {
		out = append(out, now.Add(time.Hour))
	}
	return out
}

func TestCleanupSessionsDeletesBatchAfterBatch(t *testing.T) {
	tests := []struct {
		name    string
		expired int
		calls   []int
	}{
		{"none expired", 0, []int{1000}},
		{"fewer than a batch", 3, []int{1000}},
		{"exactly a batch", 1000, []int{1000, 1000}},
		{"two batches and more", 2003, []int{1000, 1000, 1000}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeExpiries{expires: sessionsExpiring(tt.expired, now.Add(-time.Second), 2)}
			var logs bytes.Buffer

			n, err := app.NewCleanupSessions(f, clocktest.At(now), slog.New(slog.NewJSONHandler(&logs, nil))).Execute(context.Background())

			if err != nil || n != tt.expired || len(f.expires) != 2 || !slices.Equal(f.calls, tt.calls) {
				t.Errorf("Execute() = %d, %v with %d sessions left, calls %v; want %d, the 2 live left, calls %v",
					n, err, len(f.expires), f.calls, tt.expired, tt.calls)
			}
			for _, at := range f.nows {
				if !at.Equal(now) {
					t.Errorf("a batch deleted before %v, want the clock's %v", at, now)
				}
			}
			want := fmt.Sprintf(`"msg":"expired sessions deleted","sessions":%d}`, tt.expired)
			if logged := strings.Contains(logs.String(), want); logged != (tt.expired > 0) || tt.expired == 0 && logs.Len() > 0 {
				t.Errorf("logs = %s, want %s when any was deleted, else nothing", logs.String(), want)
			}
		})
	}
}

// Sessions that expire at the clock's now, or after, stay.
func TestCleanupSessionsKeepsWhatExpiresFromNowOn(t *testing.T) {
	f := &fakeExpiries{expires: []time.Time{now.Add(-time.Microsecond), now, now.Add(time.Microsecond)}}

	n, err := app.NewCleanupSessions(f, clocktest.At(now), slog.New(slog.DiscardHandler)).Execute(context.Background())

	if err != nil || n != 1 || !slices.Equal(f.expires, []time.Time{now, now.Add(time.Microsecond)}) {
		t.Errorf("Execute() = %d, %v leaving %v; want 1 deleted, those from now on left", n, err, f.expires)
	}
}

// A failed batch ends the run with the error and what the batches before
// it deleted; nothing is logged as deleted by the failed one.
func TestCleanupSessionsWhenABatchFails(t *testing.T) {
	f := &fakeExpiries{expires: sessionsExpiring(2500, now.Add(-time.Second), 0), failAt: 2}
	var logs bytes.Buffer

	n, err := app.NewCleanupSessions(f, clocktest.At(now), slog.New(slog.NewJSONHandler(&logs, nil))).Execute(context.Background())

	if !errors.Is(err, errDatabaseGone) || n != 1000 || len(f.calls) != 2 || strings.Contains(logs.String(), "expired sessions deleted") {
		t.Errorf("Execute() = %d, %v after %d calls, logs %s; want 1000 and the error after the second call, no log", n, err, len(f.calls), logs.String())
	}
}
```

`server/internal/modules/identity/adapter/river/cleanup_test.go`（新文件）：

```go
package riveradapter_test

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/riverqueue/river"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

var now = clocktest.At(time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)).Now()

func TestCleanupKind(t *testing.T) {
	if got := (riveradapter.CleanupArgs{}).Kind(); got != "identity.cleanup_expired_sessions" {
		t.Errorf("Kind() = %q, want identity.cleanup_expired_sessions (M2 design 3.15)", got)
	}
}

// The worker's integration test calls Work (M2 design 3.15): the expired
// sessions of both accounts go, the live ones stay.
func TestCleanupWorkerDeletesTheExpiredSessions(t *testing.T) {
	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, config.DatabaseConfig{URL: pgtest.NewDatabase(t), MaxConns: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	store := postgresadapter.New(pool)
	var live []uuid.UUID
	for _, email := range []string{"alice@corp.com", "bob@corp.com"} {
		u := app.NewUser{ID: uuid.NewV7(), Email: email, PasswordHash: "$argon2id$v=19$m=64,t=1,p=1$c2FsdA$a2V5", DisplayName: "x", Now: now}
		if err := store.CreateUser(ctx, u); err != nil {
			t.Fatal(err)
		}
		for _, expires := range []time.Time{now.Add(-time.Minute), now.Add(time.Minute)} {
			id := uuid.NewV7()
			if err := store.CreateSession(ctx, app.NewSession{ID: id, UserID: u.ID, TokenHash: make([]byte, 32), ExpiresAt: expires, Now: now}); err != nil {
				t.Fatal(err)
			}
			if expires.After(now) {
				live = append(live, id)
			}
		}
	}
	w := riveradapter.NewCleanupWorker(app.NewCleanupSessions(store, clocktest.At(now), slog.New(slog.DiscardHandler)))

	if err := w.Work(ctx, &river.Job[riveradapter.CleanupArgs]{}); err != nil {
		t.Fatalf("Work() = %v", err)
	}

	rows, err := pool.Query(ctx, "SELECT id FROM auth_sessions")
	if err != nil {
		t.Fatal(err)
	}
	var left []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		left = append(left, id)
	}
	slices.SortFunc(left, uuid.UUID.Compare)
	slices.SortFunc(live, uuid.UUID.Compare)
	if !slices.Equal(left, live) {
		t.Errorf("sessions left = %v, want the live ones %v", left, live)
	}
}

type failingCleanup struct{ err error }

func (f failingCleanup) Execute(context.Context) (int, error) { return 3, f.err }

// A failed cleanup fails the job, so that River retries it.
func TestCleanupWorkerFailsWithTheCleanup(t *testing.T) {
	boom := errors.New("connection reset")

	if err := riveradapter.NewCleanupWorker(failingCleanup{boom}).Work(context.Background(), &river.Job[riveradapter.CleanupArgs]{}); !errors.Is(err, boom) {
		t.Errorf("Work() = %v, want %v", err, boom)
	}
}
```

- [ ] **Step 6: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。`bootstrap` 还不接这个任务（Task 5），`Deps.SessionCleanupInterval` 为零时只构造、不运行。

- [ ] **Step 7: 提交**

```bash
git add server/internal/modules/identity
```
```bash
git commit -m "feat(M2/P3b): the periodic job that deletes the expired sessions

Batches of 1000 taken with FOR UPDATE SKIP LOCKED, so the cleanup never
waits for the account row lock order; the identity module exports it.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**Done when:** 清理只删过期的会话、跳过锁住的会话（真实数据库）；用例分批、日志和错误的测试通过。

---

### Task 5: `bootstrap`：任务与 HTTP 一起运行，按顺序停机

**Files:**
- Modify: `server/internal/bootstrap/app.go`（过渡）、`app_test.go`
- Create: `server/internal/bootstrap/jobs_test.go`
- Modify: `server/cmd/nerve/main_test.go`（过渡）

**Interfaces:**
- Produces（spec 2.7，M2 设计 3.15；M0-P2 交接 5）：
  - `newApp` 把 `cfg.Auth.SessionCleanupInterval` 交给 identity，`jobs.New(pool, jobs.Config{ShutdownTimeout: cfg.Jobs.ShutdownTimeout, Logger: logger}, ident.Jobs())`；
  - `run`：需要时自动迁移 → `a.jobs.Start(ctx)` → HTTP 服务直到 ctx 结束（HTTP 自己的优雅停机，M0）→ `a.jobs.Stop(ctx)`；两个错误用 `errors.Join` 一起返回。HTTP 先停：没有请求还在投递任务时再停任务；
  - `close`：迁移执行器 → 连接池。`pool.Close()` 在一个协程里，最多等 `poolCloseTimeout`（5 秒）：不理会 ctx 的 handler 可能还占着连接；等到了记 INFO "database pool closed"，超时记 WARN "database pool not closed: connections still in use"（`waited`），退出不被挂住。
- 使用者：`nerve serve`（`bootstrap.Serve`，不变）。

**Tests:**
- `jobs_test.go`：
  - `TestTheSessionCleanupRunsAsConfigured`：两个设置（间隔 1 秒、停止时限 2 秒；间隔 1 小时、停止时限 3 秒）：清理在启动时运行一次；1 秒的间隔在第一次之后 3 秒内再运行，1 小时的不再运行；过期的会话被删掉，未过期的留下；日志的 "jobs started" 带着配置的 `shutdown_timeout`（两个设置各自出现，接线没有写死）。
  - `TestCloseDoesNotWaitForAConnectionInUse`：有连接被占着时，`close` 在 `poolCloseTimeout`（测试设 200 毫秒）之后返回并记 WARN。
  - `TestRunStopsTheJobsAfterHTTP`：一个登录在账户行锁上等着（测试另开事务 `FOR UPDATE` 锁住账户行，`pgtest.WaitForLockWait` 确认登录在等锁）时取消 `run`：HTTP 等这个请求，任务仍在运行（"jobs stopped" 还没出现）；放开锁后登录完成（200），任务随后停止，`run` 返回。
- `app_test.go`：`testConfig` 带上两个新值；`buildAppLogging`、`startAppLogging` 让测试读日志。
- `main_test.go`：`TestServeUntilCancelled` 另外等到 `river_job` 里有一个完成的任务（清理在启动时运行过），日志里 "http server stopped"、"jobs stopped"、"database pool closed" 依次出现。

- [ ] **Step 1: 运行和停机**

`server/internal/bootstrap/app.go`（对 `d6f313b` 的差异）：

```diff
--- a/server/internal/bootstrap/app.go
+++ b/server/internal/bootstrap/app.go
@@ -21,6 +21,7 @@
 	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
 	"github.com/open-nerve/NerveProject/server/internal/platform/config"
 	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
+	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
 	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
 	"github.com/open-nerve/NerveProject/server/internal/platform/ratelimit"
 	"github.com/open-nerve/NerveProject/server/internal/platform/webui"
@@ -34,13 +35,20 @@
 	_ httpserver.ProblemError = (*shared.Error)(nil)
 )
 
+// poolCloseTimeout bounds the wait for the pool's connections at shutdown:
+// a handler that ignores its context may still hold one (M2 design 3.15).
+const poolCloseTimeout = 5 * time.Second
+
 // app is a fully wired nerve server.
 type app struct {
 	cfg      config.Config
 	logger   *slog.Logger
 	pool     *pgxpool.Pool
 	migrator *postgres.Migrator
+	jobs     *jobs.Runner
 	router   *httpserver.Router
+	// poolCloseTimeout is the package's, but for tests.
+	poolCloseTimeout time.Duration
 	// publicOperations are the routes that need no token: the union of what
 	// the modules declare, checked against the contract by a test.
 	publicOperations []string
@@ -68,7 +76,7 @@
 		pool.Close()
 		return nil, err
 	}
-	a := &app{cfg: cfg, logger: logger, pool: pool, migrator: migrator}
+	a := &app{cfg: cfg, logger: logger, pool: pool, migrator: migrator, poolCloseTimeout: poolCloseTimeout}
 	// Every rate-limit bucket lives on this limiter (M2 design 3.10). It reads
 	// time.Now, not the Clock: the monotonic reading keeps a step of the wall
 	// clock from filling or draining the buckets.
@@ -84,6 +92,8 @@
 		AccessTokenTTL:  cfg.Auth.AccessTokenTTL,
 		SessionTTL:      cfg.Auth.SessionTTL,
 		RefreshDeadline: cfg.Auth.RefreshDeadline,
+		// The periodic job that deletes the expired sessions (M2 design 3.15).
+		SessionCleanupInterval: cfg.Auth.SessionCleanupInterval,
 		Password: identity.PasswordHashing{
 			MemoryKiB:     cfg.Auth.Password.Argon2MemoryKiB,
 			Iterations:    cfg.Auth.Password.Argon2Iterations,
@@ -103,6 +113,11 @@
 		a.close()
 		return nil, err
 	}
+	a.jobs, err = jobs.New(pool, jobs.Config{ShutdownTimeout: cfg.Jobs.ShutdownTimeout, Logger: logger}, ident.Jobs())
+	if err != nil {
+		a.close()
+		return nil, err
+	}
 	inst, err := instance.New(instance.Deps{
 		SignupEnabled:            cfg.Auth.SignupEnabled,
 		WorkspaceCreationEnabled: cfg.Workspace.CreationEnabled,
@@ -197,7 +212,10 @@
 }
 
 // run applies pending migrations when database.auto_migrate is on, then
-// serves HTTP on server.addr until ctx is done (see httpserver.Server.Serve).
+// runs the jobs and serves HTTP on server.addr until ctx is done. It stops
+// in the order of M2 design 3.15: HTTP first (see httpserver.Server.Serve),
+// so no request is left to enqueue work, then the jobs; close then releases
+// the migrator and the pool.
 func (a *app) run(ctx context.Context) error {
 	warnIfExposed(ctx, a.logger, a.cfg)
 	if a.cfg.Database.AutoMigrate {
@@ -210,13 +228,27 @@
 			return err
 		}
 	}
-	return httpserver.NewServer(a.cfg.Server, a.router, a.logger).ListenAndServe(ctx)
+	a.jobs.Start(ctx)
+	served := httpserver.NewServer(a.cfg.Server, a.router, a.logger).ListenAndServe(ctx)
+	return errors.Join(served, a.jobs.Stop(ctx))
 }
 
-// close releases the database resources. Call it after run has returned.
+// close releases the database resources: the migrator, then the pool, whose
+// wait for connections still in use is bounded. Call it after run has
+// returned.
 func (a *app) close() {
 	if err := a.migrator.Close(); err != nil {
 		a.logger.Warn("close migrator", slog.Any("error", err))
 	}
-	a.pool.Close()
+	closed := make(chan struct{})
+	go func() {
+		a.pool.Close()
+		close(closed)
+	}()
+	select {
+	case <-closed:
+		a.logger.Info("database pool closed")
+	case <-time.After(a.poolCloseTimeout):
+		a.logger.Warn("database pool not closed: connections still in use", slog.Duration("waited", a.poolCloseTimeout))
+	}
 }
```

- [ ] **Step 2: 测试**

`server/internal/bootstrap/app_test.go`（对 `d6f313b` 的差异）：

```diff
--- a/server/internal/bootstrap/app_test.go
+++ b/server/internal/bootstrap/app_test.go
@@ -46,6 +46,8 @@
 
 // testConfig listens on a port the system picks and reports it through
 // server.addr_file. Password hashing is cheap: the tests hash many times.
+// The session cleanup runs when the app starts and then hourly: only the
+// test of the cleanup waits for a second run.
 func testConfig(t *testing.T, dbURL string, autoMigrate bool) config.Config {
 	t.Helper()
 	return config.Config{
@@ -62,10 +64,11 @@
 		},
 		Database: config.DatabaseConfig{URL: dbURL, MaxConns: 4, AutoMigrate: autoMigrate, CommitTimeout: 2 * time.Second},
 		Auth: config.AuthConfig{
-			SignupEnabled:   true,
-			AccessTokenTTL:  15 * time.Minute,
-			SessionTTL:      720 * time.Hour,
-			RefreshDeadline: 4 * time.Second,
+			SignupEnabled:          true,
+			AccessTokenTTL:         15 * time.Minute,
+			SessionTTL:             720 * time.Hour,
+			RefreshDeadline:        4 * time.Second,
+			SessionCleanupInterval: time.Hour,
 			Password: config.PasswordConfig{
 				Argon2MemoryKiB:     64,
 				Argon2Iterations:    1,
@@ -79,6 +82,7 @@
 			Anonymous:     roomy, AuthFailure: roomy, Authenticated: roomy,
 			LoginIP: roomy, LoginIPEmail: roomy, RegisterIP: roomy, PasswordUser: roomy,
 		},
+		Jobs:      config.JobsConfig{ShutdownTimeout: 5 * time.Second},
 		Workspace: config.WorkspaceConfig{CreationEnabled: true},
 		Files:     config.FilesConfig{SizeLimit: 5242880},
 		Log:       config.LogConfig{Level: "error", Format: "text"},
@@ -88,7 +92,13 @@
 // buildApp wires the app, serving testWebUI, and closes it when the test ends.
 func buildApp(t *testing.T, cfg config.Config, migrations fs.FS) *app {
 	t.Helper()
-	a, err := newApp(context.Background(), cfg, slog.New(slog.DiscardHandler), migrations, testWebUI)
+	return buildAppLogging(t, cfg, migrations, slog.New(slog.DiscardHandler))
+}
+
+// buildAppLogging is buildApp logging to logger.
+func buildAppLogging(t *testing.T, cfg config.Config, migrations fs.FS, logger *slog.Logger) *app {
+	t.Helper()
+	a, err := newApp(context.Background(), cfg, logger, migrations, testWebUI)
 	if err != nil {
 		t.Fatalf("newApp() error = %v", err)
 	}
@@ -101,7 +111,13 @@
 // down cleanly.
 func startApp(t *testing.T, cfg config.Config, migrations fs.FS) string {
 	t.Helper()
-	a := buildApp(t, cfg, migrations)
+	return startAppLogging(t, cfg, migrations, slog.New(slog.DiscardHandler))
+}
+
+// startAppLogging is startApp logging to logger.
+func startAppLogging(t *testing.T, cfg config.Config, migrations fs.FS, logger *slog.Logger) string {
+	t.Helper()
+	a := buildAppLogging(t, cfg, migrations, logger)
 	ctx, cancel := context.WithCancel(context.Background())
 	done := make(chan error, 1)
 	go func() { done <- a.run(ctx) }()
```

`server/internal/bootstrap/jobs_test.go`（新文件）：

```go
package bootstrap

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// lockedBuffer collects the log lines that the app's goroutines write.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// cleanupRuns counts the cleanup jobs that River has completed.
func cleanupRuns(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	err := pool.QueryRow(context.Background(),
		"SELECT count(*) FROM river_job WHERE kind = 'identity.cleanup_expired_sessions' AND state = 'completed'").Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func sessionExists(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) bool {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(context.Background(), "SELECT EXISTS (SELECT 1 FROM auth_sessions WHERE id = $1)", id).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	return exists
}

// The session cleanup is registered with River and runs when the app
// starts, then every auth.session_cleanup_interval; the jobs get
// jobs.shutdown_timeout (M2 design 3.15). Two settings of each, so that
// both reach the jobs whichever they are.
func TestTheSessionCleanupRunsAsConfigured(t *testing.T) {
	tests := []struct {
		interval, shutdown time.Duration
		runs               int // within 3 seconds after the first
	}{
		{time.Second, 2 * time.Second, 2},
		{time.Hour, 3 * time.Second, 1},
	}
	for _, tt := range tests {
		t.Run(tt.interval.String(), func(t *testing.T) {
			t.Parallel()
			url := pgtest.NewDatabase(t)
			pool := openPool(t, url)
			user, expired, live := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
			for _, insert := range []struct {
				sql  string
				args []any
			}{
				{"INSERT INTO users (id, email, password, display_name) VALUES ($1, 'cleanup@example.com', 'x', 'cleanup')", []any{user}},
				{`INSERT INTO auth_sessions (id, user_id, token_hash, expires_at) VALUES
					($1, $3, sha256('a'), now() - interval '1 minute'), ($2, $3, sha256('b'), now() + interval '1 hour')`, []any{expired, live, user}},
			} {
				if _, err := pool.Exec(context.Background(), insert.sql, insert.args...); err != nil {
					t.Fatal(err)
				}
			}
			cfg := testConfig(t, url, false)
			cfg.Auth.SessionCleanupInterval, cfg.Jobs.ShutdownTimeout = tt.interval, tt.shutdown
			var logs lockedBuffer
			startAppLogging(t, cfg, migrations.FS(), slog.New(slog.NewTextHandler(&logs, nil)))

			for deadline := time.Now().Add(15 * time.Second); cleanupRuns(t, pool) == 0; time.Sleep(50 * time.Millisecond) {
				if time.Now().After(deadline) {
					t.Fatal("the cleanup did not run when the app started")
				}
			}
			n := 1
			for deadline := time.Now().Add(3 * time.Second); n < 2 && time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
				n = cleanupRuns(t, pool)
			}

			if n != tt.runs || sessionExists(t, pool, expired) || !sessionExists(t, pool, live) {
				t.Errorf("%d runs within 3s of the first (expired session left: %v, live one left: %v); want %d, only the live one left",
					n, sessionExists(t, pool, expired), sessionExists(t, pool, live), tt.runs)
			}
			if want := `msg="jobs started" shutdown_timeout=` + tt.shutdown.String(); !strings.Contains(logs.String(), want) {
				t.Errorf("logs lack %s:\n%s", want, logs.String())
			}
		})
	}
}

// close waits for the pool's connections for poolCloseTimeout at most: a
// handler that ignores its context may hold one (M2 design 3.15).
func TestCloseDoesNotWaitForAConnectionInUse(t *testing.T) {
	var logs bytes.Buffer
	a, err := newApp(context.Background(), testConfig(t, pgtest.NewDatabase(t), false), slog.New(slog.NewTextHandler(&logs, nil)), sampleMigrations, testWebUI)
	if err != nil {
		t.Fatalf("newApp() error = %v", err)
	}
	a.poolCloseTimeout = 200 * time.Millisecond
	conn, err := a.pool.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()

	closing := time.Now()
	closed := make(chan struct{})
	go func() {
		a.close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("close() still waits for the connection after 2s")
	}

	if took := time.Since(closing); took < 200*time.Millisecond {
		t.Errorf("close() returned after %s, want it to wait 200ms for the connection", took)
	}
	if !strings.Contains(logs.String(), `msg="database pool not closed: connections still in use" waited=200ms`) {
		t.Errorf("logs lack the warning:\n%s", logs.String())
	}
}

// run stops HTTP first, draining the requests in flight, and the jobs only
// then (M2 design 3.15): while a login waits for its account's row lock at
// shutdown, the jobs run on.
func TestRunStopsTheJobsAfterHTTP(t *testing.T) {
	var logs lockedBuffer
	url := pgtest.NewDatabase(t)
	cfg := testConfig(t, url, false)
	a := buildAppLogging(t, cfg, migrations.FS(), slog.New(slog.NewTextHandler(&logs, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- a.run(ctx) }()
	waitForLog := func(want string) {
		t.Helper()
		for deadline := time.Now().Add(10 * time.Second); !strings.Contains(logs.String(), want); time.Sleep(10 * time.Millisecond) {
			if time.Now().After(deadline) {
				t.Fatalf("logs lack %s:\n%s", want, logs.String())
			}
		}
	}
	waitForLog(`msg="jobs started"`)
	waitForLog(`msg="http server listening"`)
	addr, err := os.ReadFile(cfg.Server.AddrFile)
	if err != nil {
		t.Fatal(err)
	}
	base := "http://" + string(addr)
	registerAccount(t, apitest.Load(t), base, "drain@example.com")
	pool := openPool(t, url)
	holder, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = holder.Rollback(context.Background()) }()
	if _, err := holder.Exec(context.Background(), "SELECT 1 FROM users WHERE email = 'drain@example.com' FOR UPDATE"); err != nil {
		t.Fatal(err)
	}
	answered := make(chan int, 1)
	go func() {
		status, _ := login(t, base, "drain@example.com", "Tr0ub4dor&3")
		answered <- status
	}()
	pgtest.WaitForLockWait(t, pool, 5*time.Second) // the login is in flight

	cancel()
	waitForLog(`msg="http server shutting down"`)
	time.Sleep(time.Second) // the jobs would stop meanwhile if they did not wait for HTTP
	stoppedEarly := strings.Contains(logs.String(), `msg="jobs stopped"`)
	if err := holder.Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
	status := receiveWithin(t, answered, 10*time.Second, "login answer")
	if err := receiveWithin(t, done, 10*time.Second, "return from run"); err != nil {
		t.Fatalf("run() = %v", err)
	}

	out := logs.String()
	if stoppedEarly || status != http.StatusOK ||
		strings.Index(out, `msg="http server stopped"`) > strings.Index(out, `msg="jobs stopped"`) {
		t.Errorf("jobs stopped while the login was in flight: %v; the login %d; want the jobs stopped after HTTP, the login 200:\n%s",
			stoppedEarly, status, out)
	}
}

// receiveWithin fails the test unless ch yields within limit.
func receiveWithin[T any](t *testing.T, ch <-chan T, limit time.Duration, what string) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(limit):
		t.Fatalf("no %s within %s", what, limit)
	}
	var zero T
	return zero
}
```

`server/cmd/nerve/main_test.go`（对 `d6f313b` 的差异）：

```diff
--- a/server/cmd/nerve/main_test.go
+++ b/server/cmd/nerve/main_test.go
@@ -9,6 +9,8 @@
 	"testing"
 	"time"
 
+	"github.com/jackc/pgx/v5/pgxpool"
+
 	"github.com/open-nerve/NerveProject/server/internal/platform/buildinfo"
 	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
 )
@@ -71,6 +73,8 @@
 	}
 }
 
+// nerve serve runs the API and the jobs until it is cancelled, then stops in
+// the order of M2 design 3.15: HTTP, the jobs, the pool.
 func TestServeUntilCancelled(t *testing.T) {
 	ln, err := net.Listen("tcp", "127.0.0.1:0")
 	if err != nil {
@@ -78,9 +82,15 @@
 	}
 	addr := ln.Addr().String()
 	_ = ln.Close()
+	url := pgtest.NewDatabase(t)
+	pool, err := pgxpool.New(context.Background(), url)
+	if err != nil {
+		t.Fatal(err)
+	}
+	defer pool.Close()
 	environ := []string{
 		"NERVE_ENV=test",
-		"NERVE_DATABASE__URL=" + pgtest.NewDatabase(t),
+		"NERVE_DATABASE__URL=" + url,
 		"NERVE_SERVER__ADDR=" + addr,
 		"NERVE_LOG__LEVEL=info",
 	}
@@ -97,25 +107,35 @@
 	}()
 
 	client := &http.Client{Timeout: time.Second}
-	ready := false
-	for deadline := time.Now().Add(10 * time.Second); !ready && time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
+	ready, worked := false, false
+	for deadline := time.Now().Add(15 * time.Second); (!ready || !worked) && time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
 		if resp, err := client.Get("http://" + addr + "/readyz"); err == nil {
 			_ = resp.Body.Close()
 			ready = resp.StatusCode == http.StatusOK
 		}
+		// The jobs run: the session cleanup ran when nerve started.
+		_ = pool.QueryRow(context.Background(), "SELECT EXISTS (SELECT 1 FROM river_job WHERE state = 'completed')").Scan(&worked)
 	}
 	cancel()
 	r := <-done
 
-	if !ready {
-		t.Fatalf("nerve serve never became ready; stderr:\n%s", r.stderr)
+	if !ready || !worked {
+		t.Fatalf("nerve serve never became ready (%v) or never worked a job (%v); stderr:\n%s", ready, worked, r.stderr)
 	}
 	if r.code != 0 {
 		t.Errorf("nerve serve exit code = %d, want 0; stderr:\n%s", r.code, r.stderr)
 	}
-	for _, want := range []string{"configuration loaded", "config.database.url=xxxxx", "database pool created", "http server stopped"} {
+	for _, want := range []string{"configuration loaded", "config.database.url=xxxxx", "database pool created"} {
 		if !strings.Contains(r.stderr, want) {
 			t.Errorf("stderr lacks %q:\n%s", want, r.stderr)
 		}
 	}
+	last := -1
+	for _, step := range []string{`msg="http server stopped"`, `msg="jobs stopped"`, `msg="database pool closed"`} {
+		at := strings.Index(r.stderr, step)
+		if at <= last {
+			t.Errorf("stderr has %s at %d, want it after the step before (at %d):\n%s", step, at, last, r.stderr)
+		}
+		last = at
+	}
 }
```

- [ ] **Step 3: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`；数据库不可达时的 `TestNotReadyWhenDatabaseIsUnavailable`、`TestNotReadyWhenMigrationsArePending` 仍然通过（任务在后台重试，HTTP 照常服务）。

- [ ] **Step 4: 提交**

```bash
git add server/internal/bootstrap server/cmd/nerve/main_test.go
```
```bash
git commit -m "feat(M2/P3b): nerve serve runs the jobs and stops HTTP, the jobs, the migrator, then the pool

The pool's close waits five seconds at most for connections still in use.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 清理任务按配置运行；停机顺序由日志和等锁的登录核对；连接池关闭不挂住。

---

### Task 6: 管理员命令的存储

**Files:**
- Modify: `server/internal/modules/identity/adapter/postgres/queries/users.sql`、`queries/api_tokens.sql`、`users.go`、`api_tokens.go`
- Modify（生成）: `server/internal/modules/identity/adapter/postgres/gen/users.sql.go`、`gen/api_tokens.sql.go`
- Modify: `server/internal/modules/identity/app/ports.go`
- Create: `server/internal/modules/identity/adapter/postgres/admin_test.go`

**Interfaces:**
- Produces（spec 2.8，M2 设计 3.5、3.17）：
  - `LockAccountByEmail :one`：`SELECT id FROM users WHERE email = $email FOR NO KEY UPDATE`：一条语句找到并锁住账户行，账户在两步之间不会变。端口 `app.AccountLocker{LockAccount(ctx, email) (uuid.UUID, error)}`，没有这个邮箱是 `app.ErrNotFound`。
  - `ChangeEmail :exec`（`email`、`updated_at`）：端口 `app.EmailChanger`，`users_email_key` 冲突译为 `domain.ErrEmailTaken`。
  - `ActivateUser :exec`（`is_active = true`、`updated_at`）：端口 `app.UserActivator`。
  - `RevokeAllAPITokens :execrows`：`SET updated_at = $now, deleted_at = $now, updated_by_id = NULL WHERE user_id = $user_id AND deleted_at IS NULL`：过期的也撤销，已撤销的保留原来的撤销；**`updated_by_id` 为 NULL**：这不是任何账户做的改动，与 Plane 在没有登录用户时（例如管理命令）`BaseModel.save` 写的相同（`plane/apps/api/plane/db/models/base.py:31-33`；spec 第 3 节第 5 条）。端口 `app.AllAPITokensRevoker`。
  - `CountUsableAPITokens :one`：未撤销、`expired_at` 为空或晚于 `now` 的令牌数（认证的判断：到期那一刻就不能用）。`now` 写成 `sqlc.arg(now)::timestamptz`，生成为 `time.Time` 而不是指针。端口 `app.UsableAPITokenCounter`。
- 使用者：Task 7 的用例。

**Tests:**（`admin_test.go`，真实数据库；每个测试两个账户，另一个不受影响）
- `TestLockAccountFindsTheAccountByAddress`：找到 alice 的 id；不认识的邮箱是 `ErrNotFound`。
- `TestLockAccountIsTheAccountRowLock`：持有它时，同一账户的 `LockForCredentials` 等待（`lock_timeout` 500 毫秒后失败），别的账户的不等，插入 alice 的会话也不等（`FOR NO KEY UPDATE`，不是 `FOR UPDATE`）。
- `TestChangeEmail`：改成新邮箱，`updated_at` 是给定的时刻；改成 bob 的邮箱是 `ErrEmailTaken`；bob 不变。
- `TestActivateUser`：只恢复 alice，`updated_at` 是给定的时刻。
- `TestRevokeAllAPITokens`：alice 的未过期和已过期的令牌都撤销（`deleted_at`、`updated_at` 是给定的时刻，`updated_by_id` 为 NULL），先撤销的保留原来的时刻和撤销者，返回 2；bob 的不变。
- `TestCountUsableAPITokens`：永不过期和以后过期的算，恰好在 `now` 过期的、已撤销的、bob 的不算。

- [ ] **Step 1: 查询**

`server/internal/modules/identity/adapter/postgres/queries/users.sql`（对 `d6f313b` 的差异）：

```diff
--- a/server/internal/modules/identity/adapter/postgres/queries/users.sql
+++ b/server/internal/modules/identity/adapter/postgres/queries/users.sql
@@ -51,3 +51,23 @@
 UPDATE users
 SET password = sqlc.arg(password), updated_at = sqlc.arg(now)
 WHERE id = sqlc.arg(id);
+
+-- name: LockAccountByEmail :one
+-- The account row lock of M2 design 3.5 for the server administrator's commands, which name the
+-- account by its address: one statement finds and locks the row, inside the command's
+-- transaction, so the account cannot change between the two.
+SELECT id
+FROM users
+WHERE email = sqlc.arg(email)
+FOR NO KEY UPDATE;
+
+-- name: ChangeEmail :exec
+-- nerve users set-email (M2 decision 1). users_email_key rejects an address another account has.
+UPDATE users
+SET email = sqlc.arg(email), updated_at = sqlc.arg(now)
+WHERE id = sqlc.arg(id);
+
+-- name: ActivateUser :exec
+UPDATE users
+SET is_active = true, updated_at = sqlc.arg(now)
+WHERE id = sqlc.arg(id);
```

`server/internal/modules/identity/adapter/postgres/queries/api_tokens.sql`（对 `d6f313b` 的差异）：

```diff
--- a/server/internal/modules/identity/adapter/postgres/queries/api_tokens.sql
+++ b/server/internal/modules/identity/adapter/postgres/queries/api_tokens.sql
@@ -41,3 +41,17 @@
 UPDATE api_tokens
 SET last_used = sqlc.arg(now)::timestamptz
 WHERE id = sqlc.arg(id) AND (last_used IS NULL OR last_used < sqlc.arg(stale_before)::timestamptz);
+
+-- name: RevokeAllAPITokens :execrows
+-- nerve users reset-password revokes every token of the account, expired ones too (M2 design 3.5).
+-- No account makes the change, so updated_by_id is NULL, as Plane's BaseModel.save writes it when
+-- no user is signed in, e.g. in a management command (plane/apps/api/plane/db/models/base.py:31-33).
+UPDATE api_tokens
+SET updated_at = sqlc.arg(now), deleted_at = sqlc.arg(now), updated_by_id = NULL
+WHERE user_id = sqlc.arg(user_id) AND deleted_at IS NULL;
+
+-- name: CountUsableAPITokens :one
+-- The tokens that authenticate while the account is active: unrevoked, and unexpired at now.
+SELECT count(*)
+FROM api_tokens
+WHERE user_id = sqlc.arg(user_id) AND deleted_at IS NULL AND (expired_at IS NULL OR expired_at > sqlc.arg(now)::timestamptz);
```

- [ ] **Step 2: 生成**

Run: `make gen`
Expected: 只有这两个生成文件改变：

| 生成的文件 | SHA-256 | 行数 |
|---|---|---|
| `server/internal/modules/identity/adapter/postgres/gen/api_tokens.sql.go` | `236763db28a9c7d10289d7dd861a8fad6b88b7d4d7950c118370fb6a19d1c1a3` | 246 |
| `server/internal/modules/identity/adapter/postgres/gen/users.sql.go` | `3c1b3cf00e7777895e0f493494a34dd84e44ddcf5a56c3ec2ee899d68cce27ff` | 276 |

- [ ] **Step 3: 端口和存储**

`server/internal/modules/identity/app/ports.go`（对 Task 4 版本的差异）：

```diff
--- a/server/internal/modules/identity/app/ports.go
+++ b/server/internal/modules/identity/app/ports.go
@@ -106,6 +106,29 @@
 	LockForCredentials(ctx context.Context, id uuid.UUID) (LockedAccount, error)
 }
 
+// AccountLocker takes the account row lock of M2 design 3.5 by address:
+// the server administrator's commands name accounts by address (3.17).
+type AccountLocker interface {
+	// LockAccount locks the row of the account with email, a normalized
+	// address, until the transaction ends (SELECT … FOR NO KEY UPDATE) and
+	// returns its id; ErrNotFound when there is none. Call it inside a
+	// transaction, before any other statement of the account.
+	LockAccount(ctx context.Context, email string) (uuid.UUID, error)
+}
+
+// EmailChanger changes accounts' addresses.
+type EmailChanger interface {
+	// ChangeEmail sets account id's address to email, a normalized one, at
+	// now; domain.ErrEmailTaken when another account has it.
+	ChangeEmail(ctx context.Context, id uuid.UUID, email string, now time.Time) error
+}
+
+// UserActivator activates accounts.
+type UserActivator interface {
+	// ActivateUser sets account id active at now.
+	ActivateUser(ctx context.Context, id uuid.UUID, now time.Time) error
+}
+
 // UserDeactivator deactivates accounts.
 type UserDeactivator interface {
 	// DeactivateUser sets account id inactive at now.
@@ -243,6 +266,22 @@
 	RevokeAPIToken(ctx context.Context, id, userID uuid.UUID, now time.Time) (bool, error)
 }
 
+// AllAPITokensRevoker revokes all of an account's personal access tokens:
+// the server administrator's reset of the password (M2 design 3.5).
+type AllAPITokensRevoker interface {
+	// RevokeAllAPITokens revokes at now every unrevoked token of userID,
+	// expired ones too, and returns how many. No account makes the change:
+	// updated_by_id is NULL.
+	RevokeAllAPITokens(ctx context.Context, userID uuid.UUID, now time.Time) (int, error)
+}
+
+// UsableAPITokenCounter counts an account's usable personal access tokens.
+type UsableAPITokenCounter interface {
+	// CountUsableAPITokens counts the tokens of userID that authenticate
+	// while the account is active: unrevoked and unexpired at now.
+	CountUsableAPITokens(ctx context.Context, userID uuid.UUID, now time.Time) (int, error)
+}
+
 // APITokenCredential is what authentication and the credential lock check
 // of a personal access token.
 type APITokenCredential struct {
```

`server/internal/modules/identity/adapter/postgres/users.go`（对 `d6f313b` 的差异）：

```diff
--- a/server/internal/modules/identity/adapter/postgres/users.go
+++ b/server/internal/modules/identity/adapter/postgres/users.go
@@ -107,6 +107,38 @@
 	return app.LockedAccount{PasswordHash: row.Password, Active: row.IsActive}, nil
 }
 
+// LockAccount locks the row of the account with email, a normalized
+// address, until the transaction ends and returns its id; app.ErrNotFound
+// when there is none. Call it inside a transaction, as LockForCredentials.
+func (s *Store) LockAccount(ctx context.Context, email string) (uuid.UUID, error) {
+	id, err := s.queries(ctx).LockAccountByEmail(ctx, email)
+	if err != nil {
+		return uuid.Nil(), notFound(err)
+	}
+	return id, nil
+}
+
+// ChangeEmail sets account id's address to email, a normalized one, at now.
+// An address another account has is domain.ErrEmailTaken.
+func (s *Store) ChangeEmail(ctx context.Context, id uuid.UUID, email string, now time.Time) error {
+	err := s.queries(ctx).ChangeEmail(ctx, gen.ChangeEmailParams{Email: email, Now: now, ID: id})
+	switch {
+	case uniqueViolation(err, "users_email_key"):
+		return domain.ErrEmailTaken
+	case err != nil:
+		return fmt.Errorf("change email: %w", err)
+	}
+	return nil
+}
+
+// ActivateUser sets account id active at now.
+func (s *Store) ActivateUser(ctx context.Context, id uuid.UUID, now time.Time) error {
+	if err := s.queries(ctx).ActivateUser(ctx, gen.ActivateUserParams{Now: now, ID: id}); err != nil {
+		return fmt.Errorf("activate user: %w", err)
+	}
+	return nil
+}
+
 // DeactivateUser sets account id inactive at now.
 func (s *Store) DeactivateUser(ctx context.Context, id uuid.UUID, now time.Time) error {
 	if err := s.queries(ctx).DeactivateUser(ctx, gen.DeactivateUserParams{Now: now, ID: id}); err != nil {
```

`server/internal/modules/identity/adapter/postgres/api_tokens.go`（对 `d6f313b` 的差异）：

```diff
--- a/server/internal/modules/identity/adapter/postgres/api_tokens.go
+++ b/server/internal/modules/identity/adapter/postgres/api_tokens.go
@@ -53,6 +53,26 @@
 	return n == 1, nil
 }
 
+// RevokeAllAPITokens soft-deletes every unrevoked token of userID, expired
+// ones too, at now, with updated_by_id NULL, and returns how many.
+func (s *Store) RevokeAllAPITokens(ctx context.Context, userID uuid.UUID, now time.Time) (int, error) {
+	n, err := s.queries(ctx).RevokeAllAPITokens(ctx, gen.RevokeAllAPITokensParams{Now: now, UserID: userID})
+	if err != nil {
+		return 0, fmt.Errorf("revoke all API tokens: %w", err)
+	}
+	return int(n), nil
+}
+
+// CountUsableAPITokens counts userID's tokens that are unrevoked and
+// unexpired at now.
+func (s *Store) CountUsableAPITokens(ctx context.Context, userID uuid.UUID, now time.Time) (int, error) {
+	n, err := s.queries(ctx).CountUsableAPITokens(ctx, gen.CountUsableAPITokensParams{UserID: userID, Now: now})
+	if err != nil {
+		return 0, fmt.Errorf("count usable API tokens: %w", err)
+	}
+	return int(n), nil
+}
+
 // APITokenByHash reads what authentication checks of the token with hash;
 // app.ErrNotFound when there is none.
 func (s *Store) APITokenByHash(ctx context.Context, hash []byte) (app.APITokenCredential, error) {
```

- [ ] **Step 4: 测试**

`server/internal/modules/identity/adapter/postgres/admin_test.go`（新文件）：

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
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// holdLock runs lock in a transaction and keeps the transaction open until
// the test ends.
func holdLock(t *testing.T, tx *postgres.TxManager, lock func(ctx context.Context) error) {
	t.Helper()
	locked, release, held := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		held <- tx.WithinTx(context.Background(), func(ctx context.Context) error {
			if err := lock(ctx); err != nil {
				return err
			}
			close(locked)
			<-release
			return nil
		})
	}()
	select {
	case <-locked:
	case err := <-held:
		t.Fatalf("taking the lock: %v", err)
	}
	t.Cleanup(func() {
		close(release)
		if err := <-held; err != nil {
			t.Errorf("the transaction holding the lock: %v", err)
		}
	})
}

// withLockTimeout runs fn in a transaction whose lock waits fail after
// 500ms.
func withLockTimeout(tx *postgres.TxManager, pool *pgxpool.Pool, fn func(ctx context.Context) error) error {
	return tx.WithinTx(context.Background(), func(ctx context.Context) error {
		if _, err := postgres.DB(ctx, pool).Exec(ctx, "SET LOCAL lock_timeout = '500ms'"); err != nil {
			return err
		}
		return fn(ctx)
	})
}

func isLockTimeout(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "55P03"
}

func TestLockAccountFindsTheAccountByAddress(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newUser("alice@corp.com"), newUser("bob@corp.com")
	mustCreate(t, s, alice)
	mustCreate(t, s, bob)
	tx := postgres.NewTxManager(pool, 2*time.Second)

	var got uuid.UUID
	var unknown error
	err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
		var err error
		got, err = s.LockAccount(ctx, "bob@corp.com")
		_, unknown = s.LockAccount(ctx, "carol@corp.com")
		return err
	})

	if err != nil || got != bob.ID || !errors.Is(unknown, app.ErrNotFound) {
		t.Errorf("LockAccount() = %v, %v, unknown address %v; want bob's id and app.ErrNotFound", got, err, unknown)
	}
}

// LockAccount takes the lock of M2 design 3.5 by address: while it is held,
// the credential lock of the same account waits, another account's does
// not, and inserting a session of the account does not either (FOR NO KEY
// UPDATE, not FOR UPDATE).
func TestLockAccountIsTheAccountRowLock(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newUser("alice@corp.com"), newUser("bob@corp.com")
	mustCreate(t, s, alice)
	mustCreate(t, s, bob)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	holdLock(t, tx, func(ctx context.Context) error {
		_, err := s.LockAccount(ctx, "alice@corp.com")
		return err
	})
	hash := sha256.Sum256([]byte("secret"))

	same := withLockTimeout(tx, pool, func(ctx context.Context) error {
		_, err := s.LockForCredentials(ctx, alice.ID)
		return err
	})
	byAddress := withLockTimeout(tx, pool, func(ctx context.Context) error {
		_, err := s.LockAccount(ctx, "alice@corp.com")
		return err
	})
	other := withLockTimeout(tx, pool, func(ctx context.Context) error {
		_, err := s.LockForCredentials(ctx, bob.ID)
		return err
	})
	insert := withLockTimeout(tx, pool, func(ctx context.Context) error {
		return s.CreateSession(ctx, app.NewSession{ID: uuid.NewV7(), UserID: alice.ID, TokenHash: hash[:], ExpiresAt: now.Add(time.Hour), Now: now})
	})

	if !isLockTimeout(same) || !isLockTimeout(byAddress) {
		t.Errorf("locking the held account by id: %v, by address: %v; want both to wait", same, byAddress)
	}
	if other != nil || insert != nil {
		t.Errorf("locking another account: %v; inserting a session of the held one: %v; want neither to wait", other, insert)
	}
}

func readUser(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) (email string, active bool, updated time.Time) {
	t.Helper()
	if err := pool.QueryRow(context.Background(), "SELECT email, is_active, updated_at FROM users WHERE id = $1", id).
		Scan(&email, &active, &updated); err != nil {
		t.Fatal(err)
	}
	return email, active, updated
}

func TestChangeEmail(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newUser("alice@corp.com"), newUser("bob@corp.com")
	mustCreate(t, s, alice)
	mustCreate(t, s, bob)

	changed := s.ChangeEmail(context.Background(), alice.ID, "alice@new.example", later)
	taken := s.ChangeEmail(context.Background(), alice.ID, "bob@corp.com", later.Add(time.Minute))

	if changed != nil || !errors.Is(taken, domain.ErrEmailTaken) {
		t.Errorf("ChangeEmail() = %v, to bob's address %v; want nil, then identity.email_taken", changed, taken)
	}
	if email, _, updated := readUser(t, pool, alice.ID); email != "alice@new.example" || !updated.Equal(later) {
		t.Errorf("alice = %s updated %v, want alice@new.example at %v", email, updated, later)
	}
	if email, _, updated := readUser(t, pool, bob.ID); email != "bob@corp.com" || !updated.Equal(now) {
		t.Errorf("bob = %s updated %v, want unchanged", email, updated)
	}
}

func TestActivateUser(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newUser("alice@corp.com"), newUser("bob@corp.com")
	mustCreate(t, s, alice)
	mustCreate(t, s, bob)
	for _, u := range []app.NewUser{alice, bob} {
		if err := s.DeactivateUser(context.Background(), u.ID, now); err != nil {
			t.Fatal(err)
		}
	}

	if err := s.ActivateUser(context.Background(), alice.ID, later); err != nil {
		t.Fatal(err)
	}

	if _, active, updated := readUser(t, pool, alice.ID); !active || !updated.Equal(later) {
		t.Errorf("alice active %v updated %v, want active at %v", active, updated, later)
	}
	if _, active, _ := readUser(t, pool, bob.ID); active {
		t.Error("bob is active, want him left inactive")
	}
}

// tokenRow is what the administrator's revocation writes.
type tokenRow struct {
	deleted   *time.Time
	updated   time.Time
	updatedBy *uuid.UUID
}

func readToken(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) tokenRow {
	t.Helper()
	var r tokenRow
	if err := pool.QueryRow(context.Background(), "SELECT deleted_at, updated_at, updated_by_id FROM api_tokens WHERE id = $1", id).
		Scan(&r.deleted, &r.updated, &r.updatedBy); err != nil {
		t.Fatal(err)
	}
	return r
}

// Every token of the account that is not revoked yet, expired or not, is
// revoked, by no account: updated_by_id NULL. A token revoked before keeps
// its revocation; another account's tokens stay.
func TestRevokeAllAPITokens(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newUser("alice@corp.com"), newUser("bob@corp.com")
	mustCreate(t, s, alice)
	mustCreate(t, s, bob)
	live, expired, revoked, bobs := newToken(alice.ID, "live", now), newToken(alice.ID, "expired", now),
		newToken(alice.ID, "revoked", now), newToken(bob.ID, "bobs", now)
	for _, n := range []app.NewAPIToken{live, expired, revoked, bobs} {
		mustCreateToken(t, s, n)
	}
	exec(t, pool, "UPDATE api_tokens SET expired_at = $2 WHERE id = $1", expired.ID, now.Add(-time.Minute))
	if ok, err := s.RevokeAPIToken(context.Background(), revoked.ID, alice.ID, now); err != nil || !ok {
		t.Fatalf("RevokeAPIToken() = %v, %v", ok, err)
	}

	n, err := s.RevokeAllAPITokens(context.Background(), alice.ID, later)

	if err != nil || n != 2 {
		t.Fatalf("RevokeAllAPITokens() = %d, %v; want 2", n, err)
	}
	for _, id := range []uuid.UUID{live.ID, expired.ID} {
		if r := readToken(t, pool, id); r.deleted == nil || !r.deleted.Equal(later) || !r.updated.Equal(later) || r.updatedBy != nil {
			t.Errorf("token %v = deleted %v updated %v by %v; want revoked at %v by nobody", id, r.deleted, r.updated, r.updatedBy, later)
		}
	}
	if r := readToken(t, pool, revoked.ID); r.deleted == nil || !r.deleted.Equal(now) || r.updatedBy == nil || *r.updatedBy != alice.ID {
		t.Errorf("the token revoked before = deleted %v by %v, want its own revocation kept", r.deleted, r.updatedBy)
	}
	if r := readToken(t, pool, bobs.ID); r.deleted != nil || !r.updated.Equal(now) {
		t.Errorf("bob's token = deleted %v updated %v, want untouched", r.deleted, r.updated)
	}
}

// Usable means unrevoked and unexpired at now, as authentication judges: a
// token that expires at now is not.
func TestCountUsableAPITokens(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newUser("alice@corp.com"), newUser("bob@corp.com")
	mustCreate(t, s, alice)
	mustCreate(t, s, bob)
	forever, expiring, expiresNow, revoked, bobs := newToken(alice.ID, "forever", now), newToken(alice.ID, "expiring", now),
		newToken(alice.ID, "now", now), newToken(alice.ID, "revoked", now), newToken(bob.ID, "bobs", now)
	for _, n := range []app.NewAPIToken{forever, expiring, expiresNow, revoked, bobs} {
		mustCreateToken(t, s, n)
	}
	exec(t, pool, "UPDATE api_tokens SET expired_at = $2 WHERE id = $1", expiring.ID, later)
	exec(t, pool, "UPDATE api_tokens SET expired_at = $2 WHERE id = $1", expiresNow.ID, now)
	exec(t, pool, "UPDATE api_tokens SET deleted_at = $2 WHERE id = $1", revoked.ID, now)

	n, err := s.CountUsableAPITokens(context.Background(), alice.ID, now)

	if err != nil || n != 2 {
		t.Errorf("CountUsableAPITokens() = %d, %v; want 2: the one that never expires and the one that expires later", n, err)
	}
}
```

- [ ] **Step 5: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。

- [ ] **Step 6: 提交**

```bash
git add server/internal/modules/identity
```
```bash
git commit -m "feat(M2/P3b): the store of the administrator's commands

Lock an account by its address, change the address, activate, revoke every
token (updated_by_id NULL, as Plane without a user) and count the usable ones.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**Done when:** 六个存储测试在真实数据库上通过，每个都有第二个账户证明 `WHERE` 只碰到一个账户。

---

### Task 7: 管理员的用例：创建、重置密码、改邮箱、按邮箱停用、恢复

**Files:**
- Modify: `server/internal/modules/identity/domain/user.go`、`user_test.go`、`session.go`、`errors.go`
- Create: `server/internal/modules/identity/app/create_user.go`、`reset_password.go`、`set_email.go`、`activate.go` 及四个测试、`fakes_admin_test.go`
- Modify: `server/internal/modules/identity/app/register.go`、`deactivate.go`、`deactivate_test.go`

**Interfaces:**
- Produces（spec 2.9，M2 设计 3.5、3.17，决策点 1–3）：
  - `domain.NewEmail(field, email) (string, error)`：按注册的规则规范化、校验地址（`checkEmail` 与 `NewAccount` 共用），问题是 422 `validation_failed`，`field` 是给定的字段名；
  - `domain.RevokePasswordReset`（`password_reset`）、`domain.RevokeEmailChanged`（`email_changed`）；
  - `domain.ErrAccountNotFound`（`KindNotFound`，`identity.account_not_found`，"No account has this e-mail address."）、`domain.ErrEmailUnchanged`（`KindInvalid`，`identity.email_unchanged`，"The new e-mail address is the account's current one."）：只有命令行用，不进接口描述（spec 第 3 节第 4 条）；
  - **"建账户"一步**（`app.accounts{rules, hasher, clock, users, profiles}`）：`prepare`（一次报出地址和密码的全部问题，在任何事务之外哈希）、`create`（在调用者的事务里插入账户和默认资料）。注册（`register.go`）和创建共用它（M2 设计 3.17）；
  - `app.NewCreateUser(CreateUserDeps{Rules, Hasher, Tx, Users, Profiles, Clock, Logger})`；`Execute(ctx, email, password) (string, error)` 返回规范化的地址；不经过 `SignupPolicy`（注册关闭时也能建）；不建会话；INFO "account created"（`user_id`、`by: cli`）；
  - `app.NewResetPassword(ResetPasswordDeps{Accounts, Passwords, Sessions, APITokens, Hasher, Rules, Tx, Clock, Logger})`；`Execute(ctx, email, password) (ResetPasswordResult{Email, Sessions, APITokens}, error)`：先按密码规则检查（带地址，查词干），在事务外哈希；一个事务按全局加锁顺序：`LockAccount`（按地址锁账户行）→ 写哈希 → 撤销全部会话（`password_reset`）→ 撤销全部 PAT。INFO "password reset"（`user_id`、两个数目、`by: cli`）；
  - `app.NewSetEmail(SetEmailDeps{Accounts, Users, Sessions, Tx, Clock, Logger})`；`Execute(ctx, email, newEmail) (SetEmailResult{From, To, Sessions}, error)`：新地址用 `NewEmail("new_email", …)`，与旧地址规范化后相同时 `ErrEmailUnchanged`；一个事务：按旧地址锁账户行 → 改地址 → 撤销全部会话（`email_changed`）；PAT 不撤销。日志不记两个地址（M2 设计 8.4）；
  - `(*Deactivate).ExecuteByEmail(ctx, email) (DeactivateResult{Email, Sessions}, error)`：按地址锁账户行，然后与自助停用写同样的三步（`deactivate` 私有方法，两个入口共用）；INFO "account deactivated" 带 `by: cli`。`DeactivateDeps` 加 `Accounts`：`Execute`（接口）用 `Lock`，`ExecuteByEmail`（命令行）用 `Accounts`；
  - `app.NewActivate(ActivateDeps{Accounts, Users, APITokens, Tx, Clock, Logger})`；`Execute(ctx, email) (ActivateResult{Email, APITokens}, error)`：一个事务：按地址锁账户行 → 恢复 → 数可用的 PAT；INFO "account activated"（`usable_api_tokens`）；
  - 私有的 `lockAccount`：`app.ErrNotFound` 译为 `domain.ErrAccountNotFound`。
- 使用者：Task 8 的 `identity.NewAdmin`。

**Tests:**
- `user_test.go`：`TestNewEmail`（规范化；空、太长、不合规，字段名照传）。
- `create_user_test.go`：`TestCreateUser`（一个事务里插入账户和资料，地址已规范化，没有会话，日志 `by=cli`）；`TestCreateUserChecksTheAddressAndThePassword`（坏地址和弱密码一起报出，不哈希、不插入）；`TestCreateUserWithATakenAddress`（`identity.email_taken`，不记日志）。
- `reset_password_test.go`：`TestResetPassword`（调用顺序：哈希 → 锁 → 写哈希 → 撤销会话 → 撤销 PAT；一个事务；地址已规范化；原因 `password_reset`；结果的两个数目；日志）；`TestResetPasswordChecksTheNewPassword`（弱密码、常见密码、含地址词干的密码：都不哈希、不写）；`TestResetPasswordOfAnUnknownAccount`；`TestResetPasswordWhenAWriteFails`。
- `set_email_test.go`：`TestSetEmail`（按旧地址锁、写新地址、撤销会话；日志里没有两个地址）；`TestSetEmailChecksTheNewAddress`（不合规、空、规范化后相同：什么都不锁）；`TestSetEmailFails`（不认识的账户、新地址已被占用）。
- `activate_test.go`：`TestActivate`、`TestActivateFails`。
- `deactivate_test.go`：`TestDeactivateByEmail`（与自助停用同样的三步、同样的顺序，日志 `by=cli`）；`TestDeactivateByEmailOfAnUnknownAccount`。
- `fakes_admin_test.go`：`fakeAdmin` 包着 P3a 的 `fakeCredentials`，只按它持有的地址（`alice@corp.com`）找到账户，`bob@corp.com` 是被占用的地址；每次调用把参数（账户 id、地址、原因）记进同一份调用记录，测试逐条核对。

- [ ] **Step 1: 领域**

`server/internal/modules/identity/domain/user.go`（对 `d6f313b` 的差异）：

```diff
--- a/server/internal/modules/identity/domain/user.go
+++ b/server/internal/modules/identity/domain/user.go
@@ -92,13 +92,8 @@
 func NewAccount(rules *PasswordRules, email, password string) (string, error) {
 	email = NormalizeEmail(email)
 	var fields []shared.FieldError
-	switch {
-	case email == "":
-		fields = append(fields, shared.FieldError{Field: "email", Code: shared.FieldRequired, Message: "is required"})
-	case utf8.RuneCountInString(email) > MaxEmailLength:
-		fields = append(fields, shared.FieldError{Field: "email", Code: shared.FieldTooLong, Message: "must be at most 255 characters"})
-	case !ValidEmail(email):
-		fields = append(fields, shared.FieldError{Field: "email", Code: shared.FieldInvalidFormat, Message: "is not a valid e-mail address"})
+	if f := checkEmail("email", email); f != nil {
+		fields = append(fields, *f)
 	}
 	if f := rules.Check("password", password, email); f != nil {
 		fields = append(fields, *f)
@@ -108,3 +103,27 @@
 	}
 	return email, nil
 }
+
+// NewEmail checks the new address of an account by the rules of
+// registration and returns it normalized: `nerve users set-email` (M2
+// decision 1). A problem is 422 validation_failed on field.
+func NewEmail(field, email string) (string, error) {
+	email = NormalizeEmail(email)
+	if f := checkEmail(field, email); f != nil {
+		return "", shared.Invalid(*f)
+	}
+	return email, nil
+}
+
+// checkEmail checks a normalized address.
+func checkEmail(field, email string) *shared.FieldError {
+	switch {
+	case email == "":
+		return &shared.FieldError{Field: field, Code: shared.FieldRequired, Message: "is required"}
+	case utf8.RuneCountInString(email) > MaxEmailLength:
+		return &shared.FieldError{Field: field, Code: shared.FieldTooLong, Message: "must be at most 255 characters"}
+	case !ValidEmail(email):
+		return &shared.FieldError{Field: field, Code: shared.FieldInvalidFormat, Message: "is not a valid e-mail address"}
+	}
+	return nil
+}
```

`server/internal/modules/identity/domain/session.go`（对 `d6f313b` 的差异）：

```diff
--- a/server/internal/modules/identity/domain/session.go
+++ b/server/internal/modules/identity/domain/session.go
@@ -24,6 +24,12 @@
 	RevokePasswordChanged RevokeReason = "password_changed"
 	// RevokeDeactivated revokes every session of a deactivated account.
 	RevokeDeactivated RevokeReason = "deactivated"
+	// RevokePasswordReset revokes every session of an account whose
+	// password the server's administrator reset.
+	RevokePasswordReset RevokeReason = "password_reset"
+	// RevokeEmailChanged revokes every session of an account whose address
+	// the server's administrator changed.
+	RevokeEmailChanged RevokeReason = "email_changed"
 )
 
 // RefreshTokenPrefix starts every refresh token (M2 design 3.4).
```

`server/internal/modules/identity/domain/errors.go`（完整内容）：

```go
package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The identity module's errors (M2 design 5.4). api/modules/identity.yaml
// declares the codes of those the API answers with in x-problem-codes; the
// last two are the server administrator's commands' only (3.17).
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
	// ErrCurrentPasswordIncorrect answers a change of password whose current
	// password is wrong, or was changed concurrently since it was verified
	// (M2 design 3.5).
	ErrCurrentPasswordIncorrect = shared.NewError(shared.KindInvalid, "identity.current_password_incorrect", "The current password is incorrect.")
	// ErrAPITokenNotFound answers a revocation of a token that does not
	// exist, is revoked already or belongs to another account: what the
	// caller cannot see is not found (v0 design 3.5).
	ErrAPITokenNotFound = shared.NewError(shared.KindNotFound, "identity.api_token_not_found", "The API token does not exist.")
	// ErrAccountNotFound answers a command of the server's administrator
	// for an address no account has.
	ErrAccountNotFound = shared.NewError(shared.KindNotFound, "identity.account_not_found", "No account has this e-mail address.")
	// ErrEmailUnchanged answers `nerve users set-email` when the new address
	// is, once normalized, the account's own.
	ErrEmailUnchanged = shared.NewError(shared.KindInvalid, "identity.email_unchanged", "The new e-mail address is the account's current one.")
)
```

`server/internal/modules/identity/domain/user_test.go`（对 `d6f313b` 的差异）：

```diff
--- a/server/internal/modules/identity/domain/user_test.go
+++ b/server/internal/modules/identity/domain/user_test.go
@@ -112,3 +112,26 @@
 		})
 	}
 }
+
+// A new address follows the rules of registration, reported on the field
+// the caller names (M2 decision 1).
+func TestNewEmail(t *testing.T) {
+	if email, err := NewEmail("new_email", "  Alice@New.EXAMPLE "); err != nil || email != "alice@new.example" {
+		t.Errorf("NewEmail() = %q, %v; want alice@new.example", email, err)
+	}
+	tests := []struct {
+		email string
+		want  shared.FieldError
+	}{
+		{" ", shared.FieldError{Field: "new_email", Code: "required", Message: "is required"}},
+		{"not-an-address", shared.FieldError{Field: "new_email", Code: "invalid_format", Message: "is not a valid e-mail address"}},
+		{strings.Repeat("a", 250) + "@x.com", shared.FieldError{Field: "new_email", Code: "too_long", Message: "must be at most 255 characters"}},
+	}
+	for _, tt := range tests {
+		_, err := NewEmail("new_email", tt.email)
+		var se *shared.Error
+		if !errors.As(err, &se) || se.Code != shared.CodeValidationFailed || !slices.Equal(se.Fields, []shared.FieldError{tt.want}) {
+			t.Errorf("NewEmail(%q) = %+v, want validation_failed with %+v", tt.email, err, tt.want)
+		}
+	}
+}
```

- [ ] **Step 2: "建账户"一步；创建账户**

`server/internal/modules/identity/app/create_user.go`（完整内容）：

```go
package app

import (
	"context"
	"log/slog"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// accounts is the step that registration and `nerve users create` share
// (M2 design 3.17, 6.2): check the address and the password, hash the
// password, insert the account and its default profile.
type accounts struct {
	rules    *domain.PasswordRules
	hasher   PasswordHasher
	clock    Clock
	users    UserCreator
	profiles ProfileCreator
}

// prepare checks email and password, every problem at once (422), and
// hashes the password outside any transaction (M2 design 3.5). The account
// it returns has the normalized address, and the clock's time once the
// hash is done for its audit columns.
func (a accounts) prepare(ctx context.Context, email, password string) (NewUser, error) {
	email, err := domain.NewAccount(a.rules, email, password)
	if err != nil {
		return NewUser{}, err
	}
	hash, err := a.hasher.Hash(ctx, password)
	if err != nil {
		return NewUser{}, err
	}
	return NewUser{ID: uuid.NewV7(), Email: email, PasswordHash: hash, DisplayName: domain.DisplayNameFromEmail(email), Now: a.clock.Now()}, nil
}

// create inserts u and its default profile; an address in use is
// identity.email_taken. Run it inside the caller's transaction.
func (a accounts) create(ctx context.Context, u NewUser) error {
	if err := a.users.CreateUser(ctx, u); err != nil {
		return err
	}
	return a.profiles.CreateDefaultProfile(ctx, uuid.NewV7(), u.ID, u.Now)
}

// CreateUserDeps are CreateUser's collaborators.
type CreateUserDeps struct {
	Rules    *domain.PasswordRules
	Hasher   PasswordHasher
	Tx       shared.TxManager
	Users    UserCreator
	Profiles ProfileCreator
	Clock    Clock
	Logger   *slog.Logger
}

// CreateUser creates an account for the server's administrator:
// `nerve users create` (M2 decision 2, design 3.17).
type CreateUser struct {
	d        CreateUserDeps
	accounts accounts
}

// NewCreateUser returns the use case.
func NewCreateUser(d CreateUserDeps) *CreateUser {
	return &CreateUser{d: d, accounts: accounts{rules: d.Rules, hasher: d.Hasher, clock: d.Clock, users: d.Users, profiles: d.Profiles}}
}

// Execute creates the account of email with password, and its profile, in
// one transaction; it opens no session. It is registration's first step
// without the sign-up policy: the administrator creates accounts whether
// sign-up is open or not. It returns the normalized address.
func (u *CreateUser) Execute(ctx context.Context, email, password string) (string, error) {
	user, err := u.accounts.prepare(ctx, email, password)
	if err != nil {
		return "", err
	}
	if err := u.d.Tx.WithinTx(ctx, func(ctx context.Context) error { return u.accounts.create(ctx, user) }); err != nil {
		return "", err
	}
	u.d.Logger.InfoContext(ctx, "account created", slog.String("user_id", user.ID.String()), slog.String("by", byCLI))
	return user.Email, nil
}

// byCLI is the value of "by" in the logs of the administrator's commands
// (M2 design 8.4): the command line, not the account itself.
const byCLI = "cli"
```

`server/internal/modules/identity/app/register.go`（对 `d6f313b` 的差异）：

```diff
--- a/server/internal/modules/identity/app/register.go
+++ b/server/internal/modules/identity/app/register.go
@@ -4,7 +4,6 @@
 	"context"
 	"log/slog"
 	"net/netip"
-	"uuid"
 
 	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
 	"github.com/open-nerve/NerveProject/server/internal/shared"
@@ -32,7 +31,7 @@
 
 // NewRegister returns the use case.
 func NewRegister(d RegisterDeps) *Register {
-	return &Register{d: d, accounts: accounts{users: d.Users, profiles: d.Profiles}}
+	return &Register{d: d, accounts: accounts{rules: d.Rules, hasher: d.Hasher, clock: d.Clock, users: d.Users, profiles: d.Profiles}}
 }
 
 // RegisterInput is a registration and where it comes from.
@@ -61,22 +60,15 @@
 	if !allowed {
 		return Tokens{}, domain.ErrSignupDisabled
 	}
-	email, err := domain.NewAccount(r.d.Rules, in.Email, in.Password)
+	user, err := r.accounts.prepare(ctx, in.Email, in.Password)
 	if err != nil {
 		return Tokens{}, err
 	}
-	hash, err := r.d.Hasher.Hash(ctx, in.Password)
+	session, tokens, err := r.d.Issuance.newSession(user.ID, in.UserAgent, in.IP, user.Now)
 	if err != nil {
 		return Tokens{}, err
 	}
 
-	now := r.d.Clock.Now()
-	user := NewUser{ID: uuid.NewV7(), Email: email, PasswordHash: hash, DisplayName: domain.DisplayNameFromEmail(email), Now: now}
-	session, tokens, err := r.d.Issuance.newSession(user.ID, in.UserAgent, in.IP, now)
-	if err != nil {
-		return Tokens{}, err
-	}
-
 	err = r.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
 		if err := r.accounts.create(ctx, user); err != nil {
 			return err
```

- [ ] **Step 3: 重置密码、改邮箱、恢复、按邮箱停用**

`server/internal/modules/identity/app/reset_password.go`（新文件）：

```go
package app

import (
	"context"
	"errors"
	"log/slog"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ResetPasswordDeps are ResetPassword's collaborators.
type ResetPasswordDeps struct {
	Accounts  AccountLocker
	Passwords PasswordHashWriter
	Sessions  SessionRevoker
	APITokens AllAPITokensRevoker
	Hasher    PasswordHasher
	Rules     *domain.PasswordRules
	Tx        shared.TxManager
	Clock     Clock
	Logger    *slog.Logger
}

// ResetPassword sets an account's password for the server's administrator:
// `nerve users reset-password`, the way back into an account whose
// credentials leaked (M2 design 3.5, 3.17, 8.5).
type ResetPassword struct {
	d ResetPasswordDeps
}

// NewResetPassword returns the use case.
func NewResetPassword(d ResetPasswordDeps) *ResetPassword {
	return &ResetPassword{d: d}
}

// ResetPasswordResult is the account's address and what the reset revoked.
type ResetPasswordResult struct {
	Email     string // normalized
	Sessions  int
	APITokens int
}

// Execute resets the password of the account with email:
//
//  1. the new password is checked by the password rules, with the address
//     for the stem rule: 422 validation_failed on password;
//  2. it is hashed outside the transaction (M2 design 3.5);
//  3. one transaction locks the account row by its address, writes the
//     hash, then revokes every session (password_reset) and every personal
//     access token, in the global lock order. No such account is
//     identity.account_not_found.
//
// A login or a token creation that verified the old password holds or waits
// for the same lock, and finds its credential revoked (interleavings 1-3).
func (u *ResetPassword) Execute(ctx context.Context, email, password string) (ResetPasswordResult, error) {
	email = domain.NormalizeEmail(email)
	if f := u.d.Rules.Check("password", password, email); f != nil {
		return ResetPasswordResult{}, shared.Invalid(*f)
	}
	hash, err := u.d.Hasher.Hash(ctx, password)
	if err != nil {
		return ResetPasswordResult{}, err
	}
	now := u.d.Clock.Now()
	result := ResetPasswordResult{Email: email}
	var id uuid.UUID
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		if id, err = lockAccount(ctx, u.d.Accounts, email); err != nil {
			return err
		}
		if err := u.d.Passwords.UpdatePasswordHash(ctx, id, hash, now); err != nil {
			return err
		}
		if result.Sessions, err = u.d.Sessions.RevokeSessions(ctx, id, uuid.Nil(), domain.RevokePasswordReset, now); err != nil {
			return err
		}
		result.APITokens, err = u.d.APITokens.RevokeAllAPITokens(ctx, id, now)
		return err
	})
	if err != nil {
		return ResetPasswordResult{}, err
	}
	u.d.Logger.InfoContext(ctx, "password reset", slog.String("user_id", id.String()),
		slog.Int("revoked_sessions", result.Sessions), slog.Int("revoked_api_tokens", result.APITokens), slog.String("by", byCLI))
	return result, nil
}

// lockAccount takes the account row lock by address: identity.account_not_found
// when no account has it. Call it first inside the transaction.
func lockAccount(ctx context.Context, accounts AccountLocker, email string) (uuid.UUID, error) {
	id, err := accounts.LockAccount(ctx, email)
	if errors.Is(err, ErrNotFound) {
		return uuid.Nil(), domain.ErrAccountNotFound
	}
	return id, err
}
```

`server/internal/modules/identity/app/set_email.go`（新文件）：

```go
package app

import (
	"context"
	"log/slog"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// SetEmailDeps are SetEmail's collaborators.
type SetEmailDeps struct {
	Accounts AccountLocker
	Users    EmailChanger
	Sessions SessionRevoker
	Tx       shared.TxManager
	Clock    Clock
	Logger   *slog.Logger
}

// SetEmail changes an account's address for the server's administrator:
// `nerve users set-email`, the only way to change it (M2 decision 1,
// design 3.17).
type SetEmail struct {
	d SetEmailDeps
}

// NewSetEmail returns the use case.
func NewSetEmail(d SetEmailDeps) *SetEmail {
	return &SetEmail{d: d}
}

// SetEmailResult is the old and the new address, normalized, and how many
// sessions the change revoked.
type SetEmailResult struct {
	From, To string
	Sessions int
}

// Execute changes the address email to newEmail:
//
//  1. both are normalized; the new one is checked by the rules of
//     registration (422 validation_failed on new_email), and must differ
//     from the old one (identity.email_unchanged);
//  2. one transaction locks the account row by the old address
//     (identity.account_not_found), writes the new one
//     (identity.email_taken when another account has it) and revokes every
//     session (email_changed).
//
// The personal access tokens stay: changing the address is no recovery
// from a leak; reset-password is (M2 design 3.17).
func (u *SetEmail) Execute(ctx context.Context, email, newEmail string) (SetEmailResult, error) {
	from := domain.NormalizeEmail(email)
	to, err := domain.NewEmail("new_email", newEmail)
	if err != nil {
		return SetEmailResult{}, err
	}
	if to == from {
		return SetEmailResult{}, domain.ErrEmailUnchanged
	}
	now := u.d.Clock.Now()
	result := SetEmailResult{From: from, To: to}
	var id uuid.UUID
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		if id, err = lockAccount(ctx, u.d.Accounts, from); err != nil {
			return err
		}
		if err := u.d.Users.ChangeEmail(ctx, id, to, now); err != nil {
			return err
		}
		result.Sessions, err = u.d.Sessions.RevokeSessions(ctx, id, uuid.Nil(), domain.RevokeEmailChanged, now)
		return err
	})
	if err != nil {
		return SetEmailResult{}, err
	}
	// Neither address is logged (M2 design 8.4).
	u.d.Logger.InfoContext(ctx, "e-mail address changed", slog.String("user_id", id.String()),
		slog.Int("revoked_sessions", result.Sessions), slog.String("by", byCLI))
	return result, nil
}
```

`server/internal/modules/identity/app/activate.go`（新文件）：

```go
package app

import (
	"context"
	"log/slog"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ActivateDeps are Activate's collaborators.
type ActivateDeps struct {
	Accounts  AccountLocker
	Users     UserActivator
	APITokens UsableAPITokenCounter
	Tx        shared.TxManager
	Clock     Clock
	Logger    *slog.Logger
}

// Activate activates an account for the server's administrator:
// `nerve users activate` (M2 decision 3, design 3.17).
type Activate struct {
	d ActivateDeps
}

// NewActivate returns the use case.
func NewActivate(d ActivateDeps) *Activate {
	return &Activate{d: d}
}

// ActivateResult is the account's address and how many of its personal
// access tokens authenticate again.
type ActivateResult struct {
	Email     string // normalized
	APITokens int
}

// Execute activates the account with email in one transaction: it locks the
// account row by the address (identity.account_not_found), sets it active
// and counts the tokens that are unrevoked and unexpired. Deactivation kept
// them, so they authenticate again; when the account may be compromised,
// reset-password revokes them (M2 design 3.17).
func (u *Activate) Execute(ctx context.Context, email string) (ActivateResult, error) {
	email = domain.NormalizeEmail(email)
	now := u.d.Clock.Now()
	result := ActivateResult{Email: email}
	var id uuid.UUID
	err := u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		if id, err = lockAccount(ctx, u.d.Accounts, email); err != nil {
			return err
		}
		if err := u.d.Users.ActivateUser(ctx, id, now); err != nil {
			return err
		}
		result.APITokens, err = u.d.APITokens.CountUsableAPITokens(ctx, id, now)
		return err
	})
	if err != nil {
		return ActivateResult{}, err
	}
	u.d.Logger.InfoContext(ctx, "account activated", slog.String("user_id", id.String()),
		slog.Int("usable_api_tokens", result.APITokens), slog.String("by", byCLI))
	return result, nil
}
```

`server/internal/modules/identity/app/deactivate.go`（完整内容）：

```go
package app

import (
	"context"
	"log/slog"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DeactivateDeps are Deactivate's collaborators. Execute locks with Lock,
// ExecuteByEmail with Accounts: a composition sets the one its entry uses.
type DeactivateDeps struct {
	Lock     CredentialLock
	Accounts AccountLocker
	Users    UserDeactivator
	Profiles OnboardingResetter
	Sessions SessionRevoker
	Tx       shared.TxManager
	Clock    Clock
	Logger   *slog.Logger
}

// Deactivate deactivates an account: the caller's own,
// POST /api/v0/me/deactivate, or one the server's administrator names,
// `nerve users deactivate` (M2 decision 3). The caller may use any
// credential, a token too, and no password is asked for, as in Plane
// (M2 design 8.6).
type Deactivate struct {
	d DeactivateDeps
}

// NewDeactivate returns the use case.
func NewDeactivate(d DeactivateDeps) *Deactivate {
	return &Deactivate{d: d}
}

// Execute deactivates the caller's account: it takes the credential lock,
// then writes as deactivate does.
func (u *Deactivate) Execute(ctx context.Context) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	now := u.d.Clock.Now()
	var revoked int
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if _, err := u.d.Lock.Lock(ctx, actor, now); err != nil {
			return err
		}
		var err error
		revoked, err = u.deactivate(ctx, actor.UserID, now)
		return err
	})
	if err != nil {
		return err
	}
	u.d.Logger.InfoContext(ctx, "account deactivated", slog.String("user_id", actor.UserID.String()),
		slog.Int("revoked_sessions", revoked), slog.String("by", "self"))
	return nil
}

// DeactivateResult is the account's address and how many sessions the
// deactivation revoked.
type DeactivateResult struct {
	Email    string // normalized
	Sessions int
}

// ExecuteByEmail deactivates the account with email for the server's
// administrator: it locks the account row by the address
// (identity.account_not_found), then writes as deactivate does.
func (u *Deactivate) ExecuteByEmail(ctx context.Context, email string) (DeactivateResult, error) {
	email = domain.NormalizeEmail(email)
	now := u.d.Clock.Now()
	result := DeactivateResult{Email: email}
	var id uuid.UUID
	err := u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		if id, err = lockAccount(ctx, u.d.Accounts, email); err != nil {
			return err
		}
		result.Sessions, err = u.deactivate(ctx, id, now)
		return err
	})
	if err != nil {
		return DeactivateResult{}, err
	}
	u.d.Logger.InfoContext(ctx, "account deactivated", slog.String("user_id", id.String()),
		slog.Int("revoked_sessions", result.Sessions), slog.String("by", byCLI))
	return result, nil
}

// deactivate writes in the global lock order (M2 design 3.5, 6.4): the
// account inactive, its onboarding started over, every session revoked
// with reason deactivated; it returns how many. The password and the
// personal access tokens stay; authentication refuses the tokens while the
// account is inactive. Call it under the account row lock.
func (u *Deactivate) deactivate(ctx context.Context, id uuid.UUID, now time.Time) (int, error) {
	if err := u.d.Users.DeactivateUser(ctx, id, now); err != nil {
		return 0, err
	}
	if err := u.d.Profiles.ResetOnboarding(ctx, id, now); err != nil {
		return 0, err
	}
	return u.d.Sessions.RevokeSessions(ctx, id, uuid.Nil(), domain.RevokeDeactivated, now)
}
```

- [ ] **Step 4: 测试**

`server/internal/modules/identity/app/fakes_admin_test.go`（新文件）：

```go
package app_test

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// fakeAdmin is the ports of the server administrator's commands over one
// account, userID with the address email, and the writes of fakeCredentials,
// logging every call to the same log. A second account has taken.
type fakeAdmin struct {
	*fakeCredentials
	tokens      int   // RevokeAllAPITokens revokes this many
	usable      int   // CountUsableAPITokens counts this many
	activateErr error // ActivateUser fails with it
}

const takenEmail = "bob@corp.com"

func newFakeAdmin(log *callLog) *fakeAdmin {
	return &fakeAdmin{fakeCredentials: &fakeCredentials{log: log, email: "alice@corp.com"}, tokens: 3, usable: 2}
}

func (f *fakeAdmin) LockAccount(ctx context.Context, email string) (uuid.UUID, error) {
	f.log.add(ctx, "lock "+email)
	if email != f.email {
		return uuid.Nil(), app.ErrNotFound
	}
	return userID, nil
}

func (f *fakeAdmin) ChangeEmail(ctx context.Context, id uuid.UUID, email string, now time.Time) error {
	if email == takenEmail {
		return domain.ErrEmailTaken
	}
	f.log.add(ctx, "email "+id.String()+" "+email)
	f.writtenAt = append(f.writtenAt, now)
	return nil
}

func (f *fakeAdmin) ActivateUser(ctx context.Context, id uuid.UUID, now time.Time) error {
	if f.activateErr != nil {
		return f.activateErr
	}
	f.log.add(ctx, "activate "+id.String())
	f.writtenAt = append(f.writtenAt, now)
	return nil
}

func (f *fakeAdmin) RevokeAllAPITokens(ctx context.Context, userID uuid.UUID, now time.Time) (int, error) {
	f.log.add(ctx, "revoke the tokens of "+userID.String())
	f.writtenAt = append(f.writtenAt, now)
	return f.tokens, nil
}

func (f *fakeAdmin) CountUsableAPITokens(ctx context.Context, userID uuid.UUID, now time.Time) (int, error) {
	f.log.add(ctx, "count the tokens of "+userID.String())
	f.writtenAt = append(f.writtenAt, now)
	return f.usable, nil
}
```

`server/internal/modules/identity/app/create_user_test.go`（新文件）：

```go
package app_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type createUserFixture struct {
	store  *fakeStore
	tx     *fakeTx
	hasher *fakeHasher
	logs   *bytes.Buffer
	uc     *app.CreateUser
}

func newCreateUser() *createUserFixture {
	f := &createUserFixture{store: &fakeStore{}, tx: &fakeTx{}, hasher: &fakeHasher{}, logs: &bytes.Buffer{}}
	f.uc = app.NewCreateUser(app.CreateUserDeps{
		Rules: domain.NewPasswordRules(), Hasher: f.hasher, Tx: f.tx, Users: f.store, Profiles: f.store,
		Clock: clocktest.At(now), Logger: slog.New(slog.NewJSONHandler(f.logs, nil)),
	})
	return f
}

// The account and its profile are inserted in one transaction, with the
// normalized address and the hash; no session is opened. There is no
// sign-up policy to ask: the administrator creates accounts either way
// (M2 decision 2).
func TestCreateUser(t *testing.T) {
	f := newCreateUser()

	email, err := f.uc.Execute(context.Background(), "  Carol@Corp.COM ", "Tr0ub4dor&3")

	if err != nil || email != "carol@corp.com" {
		t.Fatalf("Execute() = %q, %v; want carol@corp.com", email, err)
	}
	if len(f.store.users) != 1 || len(f.store.profiles) != 1 || len(f.store.sessions) != 0 || f.tx.calls != 1 || len(f.store.outsideTx) != 0 {
		t.Fatalf("%d accounts, %d profiles, %d sessions in %d transactions (outside: %v); want one account and profile in one, no session",
			len(f.store.users), len(f.store.profiles), len(f.store.sessions), f.tx.calls, f.store.outsideTx)
	}
	u := f.store.users[0]
	if u.Email != "carol@corp.com" || u.PasswordHash != "hashed:Tr0ub4dor&3" || u.DisplayName != "carol" || !u.Now.Equal(now) ||
		!isV7(u.ID) || f.store.profiles[0] != u.ID {
		t.Errorf("account = %+v with profile of %v, want carol's with the hash, at %v", u, f.store.profiles[0], now)
	}
	logs := f.logs.String()
	if !strings.Contains(logs, `"msg":"account created","user_id":"`+u.ID.String()+`","by":"cli"`) {
		t.Errorf("logs = %s, want the creation with user_id, by cli", logs)
	}
	assertNoSecret(t, logs, "password", []byte("Tr0ub4dor&3"))
	assertNoSecret(t, logs, "hash", []byte(u.PasswordHash))
}

// A bad address and a weak password are reported at once, before hashing;
// nothing is inserted.
func TestCreateUserChecksTheAddressAndThePassword(t *testing.T) {
	f := newCreateUser()

	_, err := f.uc.Execute(context.Background(), "not-an-address", "short")

	var se *shared.Error
	if !errors.As(err, &se) || se.Code != shared.CodeValidationFailed || len(se.Fields) != 2 || f.hasher.calls != 0 || len(f.store.users) != 0 {
		t.Errorf("Execute() = %+v after %d hashes, %d accounts; want 422 on both fields, before hashing, nothing inserted", err, f.hasher.calls, len(f.store.users))
	}
}

// An address in use is identity.email_taken, and nothing is logged.
func TestCreateUserWithATakenAddress(t *testing.T) {
	f := newCreateUser()
	f.store.createErr = domain.ErrEmailTaken

	if _, err := f.uc.Execute(context.Background(), "carol@corp.com", "Tr0ub4dor&3"); !errors.Is(err, domain.ErrEmailTaken) || f.logs.Len() != 0 ||
		len(f.store.profiles) != 0 {
		t.Errorf("Execute() = %v, logs %s, %d profiles; want identity.email_taken, no log, no profile", err, f.logs.String(), len(f.store.profiles))
	}
}
```

`server/internal/modules/identity/app/reset_password_test.go`（新文件）：

```go
package app_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
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

// adminFixture is the administrator's use cases over fakeAdmin, sharing
// one call log.
type adminFixture struct {
	log    *callLog
	store  *fakeAdmin
	tx     *fakeTx
	hasher *fakeHasher
	logs   *bytes.Buffer
}

func newAdminFixture() *adminFixture {
	log := &callLog{}
	return &adminFixture{log: log, store: newFakeAdmin(log), tx: &fakeTx{}, hasher: &fakeHasher{}, logs: &bytes.Buffer{}}
}

func (f *adminFixture) logger() *slog.Logger { return slog.New(slog.NewJSONHandler(f.logs, nil)) }

func (f *adminFixture) resetPassword() *app.ResetPassword {
	return app.NewResetPassword(app.ResetPasswordDeps{
		Accounts: f.store, Passwords: f.store, Sessions: f.store, APITokens: f.store, Hasher: f.hasher,
		Rules: domain.NewPasswordRules(), Tx: f.tx, Clock: clocktest.At(now), Logger: f.logger(),
	})
}

const newPassword = "N3w-Passw0rd!"

// One transaction locks the account by its normalized address, writes the
// hash, then revokes every session and every token, in the global lock
// order (M2 design 3.5).
func TestResetPassword(t *testing.T) {
	f := newAdminFixture()

	got, err := f.resetPassword().Execute(context.Background(), "  Alice@Corp.COM ", newPassword)

	if want := (app.ResetPasswordResult{Email: "alice@corp.com", Sessions: 1, APITokens: 3}); err != nil || got != want {
		t.Fatalf("Execute() = %+v, %v; want %+v", got, err, want)
	}
	want := []string{
		"lock alice@corp.com", "password " + userID.String() + " hashed:" + newPassword,
		"revoke password_reset sessions of " + userID.String() + " but " + uuid.Nil().String(),
		"revoke the tokens of " + userID.String(),
	}
	if !slices.Equal(f.log.calls, want) || f.tx.calls != 1 || !slices.Equal(f.store.writtenAt, []time.Time{now, now, now}) {
		t.Errorf("calls %q in %d transactions at %v; want %q in one at %v", f.log.calls, f.tx.calls, f.store.writtenAt, want, now)
	}
	logs := f.logs.String()
	if !strings.Contains(logs, `"msg":"password reset","user_id":"`+userID.String()+`","revoked_sessions":1,"revoked_api_tokens":3,"by":"cli"`) {
		t.Errorf("logs = %s, want the reset with user_id and what it revoked, by cli", logs)
	}
	assertNoSecret(t, logs, "password", []byte(newPassword))
	assertNoSecret(t, logs, "hash", []byte("hashed:"+newPassword))
}

// The password rules apply, with the account's address for the stem rule;
// a refused password is neither hashed nor written.
func TestResetPasswordChecksTheNewPassword(t *testing.T) {
	tests := []struct {
		name, email, password, code string
	}{
		{"weak", "alice@corp.com", "short", shared.FieldWeakPassword},
		{"the address's stem", "zebracorn@corp.com", "Zebracorn1!", shared.FieldCommonPassword},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newAdminFixture()

			_, err := f.resetPassword().Execute(context.Background(), tt.email, tt.password)

			var se *shared.Error
			if !errors.As(err, &se) || len(se.Fields) != 1 || se.Fields[0].Field != "password" || se.Fields[0].Code != tt.code ||
				f.hasher.calls != 0 || len(f.log.calls) != 0 {
				t.Errorf("Execute() = %+v after %d hashes and calls %q; want 422 %s on password, nothing else", err, f.hasher.calls, f.log.calls, tt.code)
			}
		})
	}
}

// An address no account has is identity.account_not_found: nothing is
// written, nothing logged.
func TestResetPasswordOfAnUnknownAccount(t *testing.T) {
	f := newAdminFixture()

	_, err := f.resetPassword().Execute(context.Background(), "carol@corp.com", newPassword)

	if !errors.Is(err, domain.ErrAccountNotFound) || !slices.Equal(f.log.calls, []string{"lock carol@corp.com"}) || f.logs.Len() != 0 {
		t.Errorf("Execute() = %v after calls %q, logs %s; want identity.account_not_found after the lock alone", err, f.log.calls, f.logs.String())
	}
}

func TestResetPasswordWhenAWriteFails(t *testing.T) {
	boom := errors.New("connection reset")
	for _, fail := range []func(*fakeAdmin){
		func(s *fakeAdmin) { s.hashErr = boom },
		func(s *fakeAdmin) { s.revokeErr = boom },
	} {
		f := newAdminFixture()
		fail(f.store)

		if _, err := f.resetPassword().Execute(context.Background(), "alice@corp.com", newPassword); !errors.Is(err, boom) || f.logs.Len() != 0 {
			t.Errorf("Execute() = %v, logs %s; want %v and nothing logged", err, f.logs.String(), boom)
		}
	}
}
```

`server/internal/modules/identity/app/set_email_test.go`（新文件）：

```go
package app_test

import (
	"context"
	"errors"
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

func (f *adminFixture) setEmail() *app.SetEmail {
	return app.NewSetEmail(app.SetEmailDeps{Accounts: f.store, Users: f.store, Sessions: f.store, Tx: f.tx, Clock: clocktest.At(now), Logger: f.logger()})
}

// One transaction locks the account by its old address, writes the new
// one, both normalized, and revokes every session; the tokens stay (M2
// design 3.17). Neither address is logged (8.4).
func TestSetEmail(t *testing.T) {
	f := newAdminFixture()

	got, err := f.setEmail().Execute(context.Background(), " ALICE@corp.com", "Alice@New.Example ")

	if want := (app.SetEmailResult{From: "alice@corp.com", To: "alice@new.example", Sessions: 1}); err != nil || got != want {
		t.Fatalf("Execute() = %+v, %v; want %+v", got, err, want)
	}
	want := []string{
		"lock alice@corp.com", "email " + userID.String() + " alice@new.example",
		"revoke email_changed sessions of " + userID.String() + " but " + uuid.Nil().String(),
	}
	if !slices.Equal(f.log.calls, want) || f.tx.calls != 1 || !slices.Equal(f.store.writtenAt, []time.Time{now, now}) {
		t.Errorf("calls %q in %d transactions at %v; want %q in one at %v", f.log.calls, f.tx.calls, f.store.writtenAt, want, now)
	}
	logs := f.logs.String()
	if !strings.Contains(logs, `"msg":"e-mail address changed","user_id":"`+userID.String()+`","revoked_sessions":1,"by":"cli"`) ||
		strings.Contains(strings.ToLower(logs), "alice@") {
		t.Errorf("logs = %s, want the change with user_id and the sessions revoked, and no address", logs)
	}
}

// The new address is checked by the rules of registration and must differ
// from the old one once both are normalized; then nothing is locked.
func TestSetEmailChecksTheNewAddress(t *testing.T) {
	tests := []struct {
		name, newEmail string
		want           error
	}{
		{"invalid", "not-an-address", shared.Invalid(shared.FieldError{Field: "new_email", Code: shared.FieldInvalidFormat})},
		{"empty", "  ", shared.Invalid(shared.FieldError{Field: "new_email", Code: shared.FieldRequired})},
		{"the same", "Alice@CORP.com ", domain.ErrEmailUnchanged},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newAdminFixture()

			_, err := f.setEmail().Execute(context.Background(), "alice@corp.com", tt.newEmail)

			var got, want *shared.Error
			if !errors.As(err, &got) || !errors.As(tt.want, &want) || got.Code != want.Code || len(got.Fields) != len(want.Fields) ||
				len(want.Fields) == 1 && (got.Fields[0].Field != want.Fields[0].Field || got.Fields[0].Code != want.Fields[0].Code) || len(f.log.calls) != 0 {
				t.Errorf("Execute() = %+v after calls %q, want %+v and no call", err, f.log.calls, tt.want)
			}
		})
	}
}

func TestSetEmailFails(t *testing.T) {
	tests := []struct {
		name, email, newEmail string
		want                  error
		calls                 []string
	}{
		{"unknown account", "carol@corp.com", "carol@new.example", domain.ErrAccountNotFound, []string{"lock carol@corp.com"}},
		{"another account's address", "alice@corp.com", takenEmail, domain.ErrEmailTaken, []string{"lock alice@corp.com"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newAdminFixture()

			_, err := f.setEmail().Execute(context.Background(), tt.email, tt.newEmail)

			if !errors.Is(err, tt.want) || !slices.Equal(f.log.calls, tt.calls) || f.logs.Len() != 0 {
				t.Errorf("Execute() = %v after calls %q, logs %s; want %v after %q, nothing logged", err, f.log.calls, f.logs.String(), tt.want, tt.calls)
			}
		})
	}
}
```

`server/internal/modules/identity/app/activate_test.go`（新文件）：

```go
package app_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
)

func (f *adminFixture) activate() *app.Activate {
	return app.NewActivate(app.ActivateDeps{Accounts: f.store, Users: f.store, APITokens: f.store, Tx: f.tx, Clock: clocktest.At(now), Logger: f.logger()})
}

// One transaction locks the account by its normalized address, activates
// it and counts the tokens that authenticate again (M2 decision 3).
func TestActivate(t *testing.T) {
	f := newAdminFixture()

	got, err := f.activate().Execute(context.Background(), " Alice@Corp.com")

	if want := (app.ActivateResult{Email: "alice@corp.com", APITokens: 2}); err != nil || got != want {
		t.Fatalf("Execute() = %+v, %v; want %+v", got, err, want)
	}
	want := []string{"lock alice@corp.com", "activate " + userID.String(), "count the tokens of " + userID.String()}
	if !slices.Equal(f.log.calls, want) || f.tx.calls != 1 || !slices.Equal(f.store.writtenAt, []time.Time{now, now}) {
		t.Errorf("calls %q in %d transactions at %v; want %q in one at %v", f.log.calls, f.tx.calls, f.store.writtenAt, want, now)
	}
	if logs := f.logs.String(); !strings.Contains(logs, `"msg":"account activated","user_id":"`+userID.String()+`","usable_api_tokens":2,"by":"cli"`) {
		t.Errorf("logs = %s, want the activation with user_id and the usable tokens, by cli", logs)
	}
}

func TestActivateFails(t *testing.T) {
	boom := errors.New("connection reset")
	tests := []struct {
		name  string
		email string
		fail  func(*fakeAdmin)
		want  error
	}{
		{"unknown account", "carol@corp.com", func(*fakeAdmin) {}, domain.ErrAccountNotFound},
		{"a failed write", "alice@corp.com", func(s *fakeAdmin) { s.activateErr = boom }, boom},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newAdminFixture()
			tt.fail(f.store)

			if _, err := f.activate().Execute(context.Background(), tt.email); !errors.Is(err, tt.want) || f.logs.Len() != 0 {
				t.Errorf("Execute() = %v, logs %s; want %v and nothing logged", err, f.logs.String(), tt.want)
			}
		})
	}
}
```

`server/internal/modules/identity/app/deactivate_test.go`（对 `d6f313b` 的差异）：

```diff
--- a/server/internal/modules/identity/app/deactivate_test.go
+++ b/server/internal/modules/identity/app/deactivate_test.go
@@ -10,6 +10,7 @@
 	"uuid"
 
 	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
+	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
 	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
 	"github.com/open-nerve/NerveProject/server/internal/shared"
 )
@@ -20,6 +21,14 @@
 	})
 }
 
+// The administrator's deactivation takes no credential lock: it has no
+// caller's credential to check again.
+func (f *adminFixture) deactivate() *app.Deactivate {
+	return app.NewDeactivate(app.DeactivateDeps{
+		Accounts: f.store, Users: f.store, Profiles: f.store, Sessions: f.store, Tx: f.tx, Clock: clocktest.At(now), Logger: f.logger(),
+	})
+}
+
 // One transaction takes the lock and checks the caller's credential again,
 // then writes in the global lock order: the account, the profile, the
 // sessions, all of them, whatever the credential (M2 design 3.5, 6.4).
@@ -96,3 +105,36 @@
 		t.Errorf("Execute() = %v after calls %q, want 401 before any", err, f.log.calls)
 	}
 }
+
+// `nerve users deactivate` locks the account by its normalized address and
+// writes what the self-service deactivation writes, in the same order
+// (M2 decision 3, design 3.17).
+func TestDeactivateByEmail(t *testing.T) {
+	f := newAdminFixture()
+
+	got, err := f.deactivate().ExecuteByEmail(context.Background(), " Alice@Corp.com ")
+
+	if want := (app.DeactivateResult{Email: "alice@corp.com", Sessions: 1}); err != nil || got != want {
+		t.Fatalf("ExecuteByEmail() = %+v, %v; want %+v", got, err, want)
+	}
+	want := []string{
+		"lock alice@corp.com", "deactivate " + userID.String(), "reset onboarding of " + userID.String(),
+		"revoke deactivated sessions of " + userID.String() + " but " + uuid.Nil().String(),
+	}
+	if !slices.Equal(f.log.calls, want) || f.tx.calls != 1 || !slices.Equal(f.store.writtenAt, []time.Time{now, now, now}) {
+		t.Errorf("calls %q in %d transactions at %v; want %q in one at %v", f.log.calls, f.tx.calls, f.store.writtenAt, want, now)
+	}
+	if logs := f.logs.String(); !strings.Contains(logs, `"msg":"account deactivated","user_id":"`+userID.String()+`","revoked_sessions":1,"by":"cli"`) {
+		t.Errorf("logs = %s, want the deactivation with user_id and the sessions revoked, by cli", logs)
+	}
+}
+
+func TestDeactivateByEmailOfAnUnknownAccount(t *testing.T) {
+	f := newAdminFixture()
+
+	_, err := f.deactivate().ExecuteByEmail(context.Background(), "carol@corp.com")
+
+	if !errors.Is(err, domain.ErrAccountNotFound) || !slices.Equal(f.log.calls, []string{"lock carol@corp.com"}) || f.logs.Len() != 0 {
+		t.Errorf("ExecuteByEmail() = %v after calls %q, logs %s; want identity.account_not_found after the lock alone", err, f.log.calls, f.logs.String())
+	}
+}
```

- [ ] **Step 5: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`；注册的测试不变而通过（"建账户"一步的抽取不改变注册的行为）。

- [ ] **Step 6: 提交**

```bash
git add server/internal/modules/identity
```
```bash
git commit -m "feat(M2/P3b): the administrator's use cases: create, reset-password, set-email, deactivate, activate

Each locks the account row by its normalized address first. Creating shares
registration's step; deactivating by address shares the self-service writes.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 五个用例的测试通过；注册的测试不变而通过。

---

### Task 8: 命令行 `nerve users`；`identity.NewAdmin`；契约说明

**Files:**
- Create: `server/internal/modules/identity/admin.go`
- Create: `server/internal/bootstrap/users.go`、`users_test.go`
- Modify: `server/internal/bootstrap/app.go`
- Create: `server/cmd/nerve/users.go`、`users_test.go`
- Modify: `server/cmd/nerve/commands.go`、`main.go`、`main_test.go`
- Modify: `server/go.mod`
- Modify: `api/modules/identity.yaml`；生成：`api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`
- Modify: `server/configs/config.yaml`

**Interfaces:**
- Produces（spec 2.10，M2 设计 3.17）：
  - `identity.AdminDeps{Pool, Tx, Clock, Logger, Password}`、`identity.Admin{CreateUser, ResetPassword, SetEmail, Deactivate, Activate}`、`identity.NewAdmin(d) *Admin`：只用连接池和密码哈希组合，不要签名密钥、限流和注册策略（spec 第 3 节第 3 条）；
  - `bootstrap.UserCommand func(ctx, *identity.Admin) (string, error)`；`bootstrap.Users(ctx, cfg, logOut, out io.Writer, cmd) error`：最小组合（日志、连接池、`TxManager`、`NewAdmin`；没有 HTTP，没有 River 客户端），命令的一行写到 `out`，日志写到 `logOut`；
  - 五个命令和它们输出的一行（M2 设计 3.17）：`CreateUser` → `created user <email>`；`ResetPassword` → `password reset for <email>: revoked <n> sessions, <m> API tokens`；`SetEmail` → `email changed from <旧> to <新>: revoked <n> sessions`；`DeactivateUser` → `deactivated <email>: revoked <n> sessions`；`ActivateUser` → `activated <email>: <m> API tokens are usable again`；
  - 错误是一行：领域错误的字段按命令行的名字写成 `<名字> <问题>`，用 `; ` 连接（`email` → `--email`，`new_email` → `--new-email`，`password` → `the password`），没有字段时是错误的说明；`main` 打印 `nerve: <这一行>`，退出码 1；
  - `passwordHashing(config.PasswordConfig) identity.PasswordHashing`：`newApp` 和 `Users` 共用；
  - `cmd/nerve`：`nerve users` 下的五个子命令，都要 `--email`（`set-email` 另要 `--new-email`）；`create`、`reset-password` 读密码：标准输入是终端时用 `golang.org/x/term` 不回显地提示两次，两次不同则失败；否则读一行，只去掉行尾的 `\n` 或 `\r\n`（最后一行没有换行也算），什么都读不到是错误；空行由密码规则报 `the password is required`。`deactivate`、`activate`、`set-email` 不读标准输入。只有 `nerve users` 时打印帮助。`run` 和 `newRootCommand` 多一个 `stdin`；
  - 契约：`updateMe` 的说明写明邮箱由 `nerve users set-email` 修改并结束全部会话；`deactivateMe` 的说明写明 `nerve users activate` 恢复、之后未过期的 PAT 重新可用（P3a 评审第 6 节的复核）；
  - `config.yaml` 中注册一条的注释改为 `nerve users create --email <邮箱>` 的用法。
- 使用者：Task 11 的端到端（`bin/nerve users …`）。

**Tests:**
- `bootstrap/users_test.go`：`TestUsersCommands`：五个命令在同一个账户上依次运行（2 个会话、3 个 PAT），每个输出自己的一行，账户的地址、状态、会话的原因和 PAT 逐步如期变化；第二个账户（bob）始终不变；`river_job` 是空的（命令行没有任务客户端）。`TestActivateCountsTheUsableTokens`（未撤销、未过期的才算）。`TestUsersCommandErrors`（9 个：创建时地址被占用、弱密码和坏地址一起、重置不认识的账户、常见密码、改成别人的地址、规范化后相同的地址、不合规的新地址、停用和恢复不认识的账户：每个只有一行说明，没有输出行；之后库里只有测试建的账户，都没变）。`TestCommandErrorKeepsOtherErrors`。
- `cmd/nerve/users_test.go`：`TestUsersReadThePasswordFromStandardInput`（`\n`、`\r\n`、没有换行的最后一行、前后的空格保留；存的哈希用 test 配置的 argon2 参数 `m=64,t=1,p=1`，证明命令按 `auth.password` 哈希）；`TestUsersCommandsWithoutAPassword`（空的标准输入不妨碍它们）；`TestUsersCommandsFail`（没有密码、空密码、缺 `--email`、缺 `--new-email`、不认识的账户、不认识的子命令：退出码 1，标准输出为空，标准错误的最后一行）；`TestBareUsersPrintsHelp`。
- `cmd/nerve/main_test.go`：`executeWithInput` 给命令标准输入。

- [ ] **Step 1: 模块的管理员组合**

`server/internal/modules/identity/admin.go`（新文件）：

```go
package identity

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	argon2adapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/argon2"
	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// AdminDeps are what the server administrator's commands need: the pool and
// the password hashing, and no signing key, rate limit or sign-up policy
// (M2 design 3.17).
type AdminDeps struct {
	Pool     *pgxpool.Pool
	Tx       shared.TxManager
	Clock    app.Clock
	Logger   *slog.Logger
	Password PasswordHashing
}

// Admin is the server administrator's use cases, behind `nerve users`
// (M2 design 3.17). The API shares them where it has the operation:
// deactivation is one use case with two entries.
type Admin struct {
	CreateUser    *app.CreateUser
	ResetPassword *app.ResetPassword
	SetEmail      *app.SetEmail
	Deactivate    *app.Deactivate
	Activate      *app.Activate
}

// NewAdmin wires the administrator's use cases on the pool alone: the
// command line builds no HTTP server and no jobs client, so it is the
// module's own minimal composition, not New's.
func NewAdmin(d AdminDeps) *Admin {
	hasher := argon2adapter.New(argon2adapter.Params(d.Password), d.Logger)
	rules := domain.NewPasswordRules()
	store := postgresadapter.New(d.Pool)
	return &Admin{
		CreateUser: app.NewCreateUser(app.CreateUserDeps{
			Rules: rules, Hasher: hasher, Tx: d.Tx, Users: store, Profiles: store, Clock: d.Clock, Logger: d.Logger,
		}),
		ResetPassword: app.NewResetPassword(app.ResetPasswordDeps{
			Accounts: store, Passwords: store, Sessions: store, APITokens: store, Hasher: hasher,
			Rules: rules, Tx: d.Tx, Clock: d.Clock, Logger: d.Logger,
		}),
		SetEmail: app.NewSetEmail(app.SetEmailDeps{
			Accounts: store, Users: store, Sessions: store, Tx: d.Tx, Clock: d.Clock, Logger: d.Logger,
		}),
		Deactivate: app.NewDeactivate(app.DeactivateDeps{
			Accounts: store, Users: store, Profiles: store, Sessions: store, Tx: d.Tx, Clock: d.Clock, Logger: d.Logger,
		}),
		Activate: app.NewActivate(app.ActivateDeps{
			Accounts: store, Users: store, APITokens: store, Tx: d.Tx, Clock: d.Clock, Logger: d.Logger,
		}),
	}
}
```

- [ ] **Step 2: 最小组合与输出**

`server/internal/bootstrap/users.go`（新文件）：

```go
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/logging"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UserCommand is one `nerve users` command on the administrator's use
// cases: it runs and returns the line it prints.
type UserCommand func(ctx context.Context, admin *identity.Admin) (string, error)

// Users runs cmd on the minimal composition of M2 design 3.17: a pool and
// identity's administrator use cases; no HTTP server, no jobs client. The
// command's line goes to out, the logs to logOut. An error is one line for
// the administrator, and the database is unchanged.
func Users(ctx context.Context, cfg config.Config, logOut, out io.Writer, cmd UserCommand) error {
	logger, err := logging.New(logOut, cfg.Log)
	if err != nil {
		return err
	}
	pool, err := postgres.NewPool(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()
	admin := identity.NewAdmin(identity.AdminDeps{
		Pool:     pool,
		Tx:       postgres.NewTxManager(pool, cfg.Database.CommitTimeout),
		Clock:    clock.System{},
		Logger:   logger,
		Password: passwordHashing(cfg.Auth.Password),
	})
	line, err := cmd(ctx, admin)
	if err != nil {
		return commandError(err)
	}
	return writeLine(out, line)
}

// CreateUser is `nerve users create` (M2 decision 2).
func CreateUser(email, password string) UserCommand {
	return func(ctx context.Context, admin *identity.Admin) (string, error) {
		created, err := admin.CreateUser.Execute(ctx, email, password)
		return "created user " + created, err
	}
}

// ResetPassword is `nerve users reset-password` (M2 design 3.5).
func ResetPassword(email, password string) UserCommand {
	return func(ctx context.Context, admin *identity.Admin) (string, error) {
		r, err := admin.ResetPassword.Execute(ctx, email, password)
		return fmt.Sprintf("password reset for %s: revoked %d sessions, %d API tokens", r.Email, r.Sessions, r.APITokens), err
	}
}

// SetEmail is `nerve users set-email` (M2 decision 1).
func SetEmail(email, newEmail string) UserCommand {
	return func(ctx context.Context, admin *identity.Admin) (string, error) {
		r, err := admin.SetEmail.Execute(ctx, email, newEmail)
		return fmt.Sprintf("email changed from %s to %s: revoked %d sessions", r.From, r.To, r.Sessions), err
	}
}

// DeactivateUser is `nerve users deactivate` (M2 decision 3).
func DeactivateUser(email string) UserCommand {
	return func(ctx context.Context, admin *identity.Admin) (string, error) {
		r, err := admin.Deactivate.ExecuteByEmail(ctx, email)
		return fmt.Sprintf("deactivated %s: revoked %d sessions", r.Email, r.Sessions), err
	}
}

// ActivateUser is `nerve users activate` (M2 decision 3).
func ActivateUser(email string) UserCommand {
	return func(ctx context.Context, admin *identity.Admin) (string, error) {
		r, err := admin.Activate.Execute(ctx, email)
		return fmt.Sprintf("activated %s: %d API tokens are usable again", r.Email, r.APITokens), err
	}
}

// fieldNames name the use cases' fields as the command line knows them.
var fieldNames = map[string]string{"email": "--email", "new_email": "--new-email", "password": "the password"}

// commandError is err as one line for the administrator: the invalid
// fields of a domain error, each as "<field> <problem>", or its detail.
func commandError(err error) error {
	var se *shared.Error
	if !errors.As(err, &se) || len(se.Fields) == 0 {
		return err
	}
	problems := make([]string, len(se.Fields))
	for i, f := range se.Fields {
		name, ok := fieldNames[f.Field]
		if !ok {
			name = f.Field
		}
		problems[i] = name + " " + f.Message
	}
	return errors.New(strings.Join(problems, "; "))
}
```

`server/internal/bootstrap/app.go`（对 Task 5 版本的差异）：

```diff
--- a/server/internal/bootstrap/app.go
+++ b/server/internal/bootstrap/app.go
@@ -94,13 +94,7 @@
 		RefreshDeadline: cfg.Auth.RefreshDeadline,
 		// The periodic job that deletes the expired sessions (M2 design 3.15).
 		SessionCleanupInterval: cfg.Auth.SessionCleanupInterval,
-		Password: identity.PasswordHashing{
-			MemoryKiB:     cfg.Auth.Password.Argon2MemoryKiB,
-			Iterations:    cfg.Auth.Password.Argon2Iterations,
-			Parallelism:   cfg.Auth.Password.Argon2Parallelism,
-			MaxConcurrent: cfg.Auth.Password.MaxConcurrentHashes,
-			MaxWait:       cfg.Auth.Password.MaxWait,
-		},
+		Password:               passwordHashing(cfg.Auth.Password),
 		RateLimits: identity.RateLimits{
 			Limiter:      limiter,
 			LoginIP:      bucket(limiter, "login_ip", cfg.RateLimit.LoginIP),
@@ -177,6 +171,17 @@
 	return data, nil
 }
 
+// passwordHashing is auth.password as identity takes it.
+func passwordHashing(p config.PasswordConfig) identity.PasswordHashing {
+	return identity.PasswordHashing{
+		MemoryKiB:     p.Argon2MemoryKiB,
+		Iterations:    p.Argon2Iterations,
+		Parallelism:   p.Argon2Parallelism,
+		MaxConcurrent: p.MaxConcurrentHashes,
+		MaxWait:       p.MaxWait,
+	}
+}
+
 // bucket is the rate-limit bucket name on limiter, sized by c (M2 design
 // 3.10).
 func bucket(limiter *ratelimit.Limiter, name string, c config.BucketConfig) *ratelimit.Bucket {
```

- [ ] **Step 3: 命令行**

`server/cmd/nerve/users.go`（新文件）：

```go
package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/open-nerve/NerveProject/server/internal/bootstrap"
)

// newUsersCommand is `nerve users`, the server administrator's commands
// (M2 design 3.17). Each names the account with --email; the ones that set
// a password read it from stdin.
func newUsersCommand(load configLoader, stdin io.Reader) *cobra.Command {
	users := &cobra.Command{
		Use:   "users",
		Short: "Manage accounts as the server's administrator",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	users.AddCommand(
		userCommand(load, stdin, "create", "Create an account without signing it in; works while sign-up is closed",
			true, bootstrap.CreateUser),
		userCommand(load, stdin, "reset-password", "Set an account's password and revoke all its sessions and API tokens",
			true, bootstrap.ResetPassword),
		setEmailCommand(load),
		userCommand(load, stdin, "deactivate", "Deactivate an account and revoke its sessions; its API tokens stay",
			false, func(email, _ string) bootstrap.UserCommand { return bootstrap.DeactivateUser(email) }),
		userCommand(load, stdin, "activate", "Activate an account; its unexpired API tokens work again, so run reset-password too if it may be compromised",
			false, func(email, _ string) bootstrap.UserCommand { return bootstrap.ActivateUser(email) }),
	)
	return users
}

// userCommand builds one `nerve users` subcommand: it requires --email and,
// when withPassword, reads the password before it runs.
func userCommand(load configLoader, stdin io.Reader, use, short string, withPassword bool,
	command func(email, password string) bootstrap.UserCommand) *cobra.Command {
	var email string
	cmd := &cobra.Command{
		Use:   use + " --email <address>",
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := load()
			if err != nil {
				return err
			}
			var password string
			if withPassword {
				if password, err = readPassword(stdin, cmd.ErrOrStderr()); err != nil {
					return err
				}
			}
			return bootstrap.Users(cmd.Context(), cfg, cmd.ErrOrStderr(), cmd.OutOrStdout(), command(email, password))
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "the account's e-mail address")
	_ = cmd.MarkFlagRequired("email") // the flag exists
	return cmd
}

// setEmailCommand is `nerve users set-email` (M2 decision 1).
func setEmailCommand(load configLoader) *cobra.Command {
	var email, newEmail string
	cmd := &cobra.Command{
		Use: "set-email --email <address> --new-email <address>",
		Short: "Change an account's e-mail address and revoke its sessions; its API tokens stay, " +
			"so run reset-password too if the change is about a compromise",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := load()
			if err != nil {
				return err
			}
			return bootstrap.Users(cmd.Context(), cfg, cmd.ErrOrStderr(), cmd.OutOrStdout(), bootstrap.SetEmail(email, newEmail))
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "the account's e-mail address")
	cmd.Flags().StringVar(&newEmail, "new-email", "", "the new e-mail address")
	_ = cmd.MarkFlagRequired("email") // the flags exist
	_ = cmd.MarkFlagRequired("new-email")
	return cmd
}

// readPassword reads the new password (M2 design 3.17): on a terminal it
// asks twice without echo, and the two must match; otherwise it reads one
// line, for scripts and tests. Only the line ending is removed.
func readPassword(in io.Reader, prompt io.Writer) (string, error) {
	if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		first, err := promptPassword(f, prompt, "Password: ")
		if err != nil {
			return "", err
		}
		again, err := promptPassword(f, prompt, "Password again: ")
		if err != nil {
			return "", err
		}
		if !bytes.Equal(first, again) {
			return "", errors.New("the passwords do not match")
		}
		return string(first), nil
	}
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && (!errors.Is(err, io.EOF) || line == "") {
		return "", fmt.Errorf("read the password from standard input: %w", err)
	}
	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
}

func promptPassword(f *os.File, prompt io.Writer, label string) ([]byte, error) {
	_, _ = io.WriteString(prompt, label)
	password, err := term.ReadPassword(int(f.Fd()))
	_, _ = io.WriteString(prompt, "\n")
	if err != nil {
		return nil, fmt.Errorf("read the password: %w", err)
	}
	return password, nil
}
```

`server/cmd/nerve/commands.go`（对 `d6f313b` 的差异）：

```diff
--- a/server/cmd/nerve/commands.go
+++ b/server/cmd/nerve/commands.go
@@ -19,7 +19,7 @@
 
 type configLoader func() (config.Config, error)
 
-func newRootCommand(environ []string) *cobra.Command {
+func newRootCommand(environ []string, stdin io.Reader) *cobra.Command {
 	root := &cobra.Command{
 		Use:           "nerve",
 		Short:         "Nerve: a lightweight project management server",
@@ -31,7 +31,7 @@
 	load := func() (config.Config, error) {
 		return config.Load(config.Sources{Embedded: configs.FS(), Environ: environ, LocalFile: localConfigFile})
 	}
-	root.AddCommand(newServeCommand(load), newMigrateCommand(load), newVersionCommand())
+	root.AddCommand(newServeCommand(load), newMigrateCommand(load), newUsersCommand(load, stdin), newVersionCommand())
 	return root
 }
 
```

`server/cmd/nerve/main.go`（完整内容）：

```go
// Command nerve runs the Nerve server and its operational commands.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	_ "time/tzdata" // every zone is known, whatever the host has (M2 design 4.2)
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	// After the first signal starts the graceful shutdown, restore the
	// default handling so a second signal stops the process at once.
	context.AfterFunc(ctx, stop)
	code := run(ctx, os.Args[1:], os.Environ(), os.Stdin, os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

// run executes one command line and returns the process exit code. A
// password comes from stdin; results go to stdout; logs and errors go to
// stderr.
func run(ctx context.Context, args, environ []string, stdin io.Reader, stdout, stderr io.Writer) int {
	root := newRootCommand(environ, stdin)
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)
	if err := root.ExecuteContext(ctx); err != nil {
		_, _ = fmt.Fprintf(stderr, "nerve: %v\n", err)
		return 1
	}
	return 0
}
```

- [ ] **Step 4: 依赖**

Run: `go -C server get golang.org/x/term@v0.46.0`

Run: `go -C server mod tidy`

Run: `grep -n "^go \|^toolchain" server/go.mod server/tools/go.mod`
Expected: 四行，`go 1.27` 和 `toolchain go1.27.1` 各两行。

`server/go.mod` 应与下面的差异相同（`go.sum` 不变：x/term v0.46.0 本来就在里面）：

`server/go.mod`（对 Task 3 版本的差异）：

```diff
--- a/server/go.mod
+++ b/server/go.mod
@@ -21,6 +21,7 @@
 	github.com/spf13/cobra v1.10.2
 	github.com/testcontainers/testcontainers-go/modules/postgres v0.44.0
 	golang.org/x/crypto v0.57.0
+	golang.org/x/term v0.46.0
 	golang.org/x/tools v0.50.0
 )
 
```

- [ ] **Step 5: 契约说明和配置注释**

`api/modules/identity.yaml`（对 `d6f313b` 的差异）：

```diff
--- a/api/modules/identity.yaml
+++ b/api/modules/identity.yaml
@@ -138,7 +138,8 @@
       summary: Change the caller's names and time zone
       description: >-
         Changes the fields sent and leaves the others. The e-mail address
-        cannot change here: the server's administrator changes it.
+        cannot change here: the server's administrator changes it with
+        `nerve users set-email`, which signs every session out.
       security: [{bearer: []}]
       x-problem-codes: [validation_failed]
       requestBody:
@@ -166,7 +167,8 @@
         asked for. Every session is signed out and onboarding starts over.
         The password and the personal access tokens stay, but nothing
         authenticates as the account until the server's administrator
-        activates it again.
+        activates it again with `nerve users activate`; then its unexpired
+        personal access tokens authenticate again.
       security: [{bearer: []}]
       x-problem-codes: []
       responses:
```

`server/configs/config.yaml`（对 Task 1 版本的差异）：

```diff
--- a/server/configs/config.yaml
+++ b/server/configs/config.yaml
@@ -32,7 +32,7 @@
 
 auth:
   # 是否开放注册。基础配置（也就是 prod）关闭；config.dev.yaml、config.test.yaml 覆盖为 true。
-  # 关闭时第一个账户用 nerve users create 创建（M2/P3 加入；在那之前临时设 NERVE_AUTH__SIGNUP_ENABLED=true）
+  # 关闭时第一个账户用 nerve users create --email <邮箱> 创建，密码从终端或标准输入读取
   signup_enabled: false
   access_token_ttl: 15m
   # 会话从登录起算的期限（30 天），续期不延长
```

Run: `make gen`
Expected: 只有两个生成文件改变：

| 生成的文件 | SHA-256 | 行数 |
|---|---|---|
| `api/dist/openapi.yaml` | `4170c15093c75216a8c9b9d2721f5557aa68b15f3175792ce074dd3bb93d8f61` | 878 |
| `web/packages/api-client/src/schema.gen.ts` | `c27e42b4792f13c2be63d9b23dad755a899bb945cd1318b22b3f9c32261c4407` | 883 |

- [ ] **Step 6: 测试**

`server/internal/bootstrap/users_test.go`（新文件）：

```go
package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// runUsers runs cmd on the database at url and returns its line, its logs
// and its error.
func runUsers(t *testing.T, url string, cmd UserCommand) (out, logs string, err error) {
	t.Helper()
	cfg := testConfig(t, url, false)
	cfg.Log.Level = "info"
	var stdout, stderr bytes.Buffer
	err = Users(context.Background(), cfg, &stderr, &stdout, cmd)
	return stdout.String(), stderr.String(), err
}

// accountState is what the commands change of an account: its address, its
// state, its sessions' reasons and its tokens.
type accountState struct {
	email, sessions, tokens string
	active                  bool
}

func stateOf(t *testing.T, pool *pgxpool.Pool, id string) accountState {
	t.Helper()
	var s accountState
	err := pool.QueryRow(context.Background(), `SELECT u.email, u.is_active,
		coalesce((SELECT string_agg(coalesce(revoke_reason, 'live'), ',' ORDER BY coalesce(revoke_reason, 'live')) FROM auth_sessions WHERE user_id = u.id), ''),
		coalesce((SELECT string_agg(CASE WHEN deleted_at IS NULL THEN 'kept' ELSE 'revoked' END, ',' ORDER BY deleted_at) FROM api_tokens WHERE user_id = u.id), '')
		FROM users u WHERE u.id = $1`, id).Scan(&s.email, &s.active, &s.sessions, &s.tokens)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// The five commands run on the minimal composition, one after another on
// one account, each printing its line (M2 design 3.17); a second account
// shows that nothing reaches beyond the one named. The composition has no
// jobs client: nothing is enqueued.
func TestUsersCommands(t *testing.T) {
	url := pgtest.NewDatabase(t)
	pool := openPool(t, url)
	bob := createdAccount(t, url, pool, "bob@corp.com")

	out, logs, err := runUsers(t, url, CreateUser(" Carol@Corp.COM ", "Tr0ub4dor&3"))
	if err != nil || out != "created user carol@corp.com\n" || !strings.Contains(logs, `msg="account created"`) {
		t.Fatalf("create = %q, %v, logs %s", out, err, logs)
	}
	var carol string
	if err := pool.QueryRow(context.Background(), "SELECT id FROM users WHERE email = 'carol@corp.com'").Scan(&carol); err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{
		`INSERT INTO auth_sessions (id, user_id, token_hash, expires_at) VALUES (uuidv7(), $1, sha256(uuidv7()::text::bytea), now() + interval '1 hour'),
			(uuidv7(), $1, sha256(uuidv7()::text::bytea), now() + interval '1 hour')`,
		`INSERT INTO api_tokens (id, user_id, token_hash, label) VALUES (uuidv7(), $1, sha256(uuidv7()::text::bytea), 'c'),
			(uuidv7(), $1, sha256(uuidv7()::text::bytea), 'd'), (uuidv7(), $1, sha256(uuidv7()::text::bytea), 'e')`,
	} {
		for _, id := range []string{carol, bob} {
			if _, err := pool.Exec(context.Background(), sql, id); err != nil {
				t.Fatal(err)
			}
		}
	}
	// Each step after the reset signs carol in once more first: a live
	// session for it to revoke, or not.
	steps := []struct {
		name string
		cmd  UserCommand
		out  string
		want accountState
	}{
		{"reset-password", ResetPassword("CAROL@corp.com", "N3w-Passw0rd!"),
			"password reset for carol@corp.com: revoked 2 sessions, 3 API tokens\n",
			accountState{"carol@corp.com", "password_reset,password_reset", "revoked,revoked,revoked", true}},
		{"set-email", SetEmail("carol@corp.com", "Carol@New.Example"),
			"email changed from carol@corp.com to carol@new.example: revoked 1 sessions\n",
			accountState{"carol@new.example", "email_changed,password_reset,password_reset", "revoked,revoked,revoked", true}},
		{"deactivate", DeactivateUser("carol@new.example"),
			"deactivated carol@new.example: revoked 1 sessions\n",
			accountState{"carol@new.example", "deactivated,email_changed,password_reset,password_reset", "revoked,revoked,revoked", false}},
		{"activate", ActivateUser("carol@new.example"),
			"activated carol@new.example: 0 API tokens are usable again\n",
			accountState{"carol@new.example", "deactivated,email_changed,live,password_reset,password_reset", "revoked,revoked,revoked", true}},
	}
	for i, step := range steps {
		if i > 0 {
			if _, err := pool.Exec(context.Background(), `INSERT INTO auth_sessions (id, user_id, token_hash, expires_at)
				VALUES (uuidv7(), $1, sha256(uuidv7()::text::bytea), now() + interval '1 hour')`, carol); err != nil {
				t.Fatal(err)
			}
		}
		out, _, err := runUsers(t, url, step.cmd)
		if got := stateOf(t, pool, carol); err != nil || out != step.out || got != step.want {
			t.Errorf("%s = %q, %v leaving %+v; want %q leaving %+v", step.name, out, err, got, step.out, step.want)
		}
	}
	if got, want := stateOf(t, pool, bob), (accountState{"bob@corp.com", "live,live", "kept,kept,kept", true}); got != want {
		t.Errorf("bob = %+v, want %+v: untouched", got, want)
	}
	var jobs int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM river_job").Scan(&jobs); err != nil || jobs != 0 {
		t.Errorf("river_job holds %d rows (%v), want none: the commands have no jobs client", jobs, err)
	}
}

// createdAccount creates email through the command and returns its id.
func createdAccount(t *testing.T, url string, pool *pgxpool.Pool, email string) string {
	t.Helper()
	if _, _, err := runUsers(t, url, CreateUser(email, "Tr0ub4dor&3")); err != nil {
		t.Fatal(err)
	}
	var id string
	if err := pool.QueryRow(context.Background(), "SELECT id FROM users WHERE email = $1", email).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// A deactivated account's tokens are counted as usable again when it is
// activated: unrevoked and unexpired ones only.
func TestActivateCountsTheUsableTokens(t *testing.T) {
	url := pgtest.NewDatabase(t)
	pool := openPool(t, url)
	id := createdAccount(t, url, pool, "dave@corp.com")
	if _, err := pool.Exec(context.Background(), `INSERT INTO api_tokens (id, user_id, token_hash, label, expired_at, deleted_at) VALUES
		(uuidv7(), $1, sha256('a'), 'a', NULL, NULL), (uuidv7(), $1, sha256('b'), 'b', now() + interval '1 day', NULL),
		(uuidv7(), $1, sha256('c'), 'c', now() - interval '1 day', NULL), (uuidv7(), $1, sha256('d'), 'd', NULL, now())`, id); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runUsers(t, url, DeactivateUser("dave@corp.com")); err != nil {
		t.Fatal(err)
	}

	out, _, err := runUsers(t, url, ActivateUser("dave@corp.com"))

	if err != nil || out != "activated dave@corp.com: 2 API tokens are usable again\n" {
		t.Errorf("activate = %q, %v; want 2 usable tokens", out, err)
	}
}

// A refused command prints no line and says why in one line: the invalid
// fields by the names the command line knows, or the error's detail.
func TestUsersCommandErrors(t *testing.T) {
	url := pgtest.NewDatabase(t)
	pool := openPool(t, url)
	createdAccount(t, url, pool, "erin@corp.com")
	createdAccount(t, url, pool, "frank@corp.com")
	tests := []struct {
		name string
		cmd  UserCommand
		want string
	}{
		{"a taken address", CreateUser("erin@corp.com", "Tr0ub4dor&3"), "An account with this e-mail address already exists."},
		{"a weak password and a bad address", CreateUser("nobody", "short"),
			"--email is not a valid e-mail address; the password must be 8-128 characters with an upper-case letter, a lower-case letter, a digit and one of !@#$%^&*()-_+=[]{}|;:'\",.<>?/"},
		{"an unknown account", ResetPassword("nobody@corp.com", "N3w-Passw0rd!"), "No account has this e-mail address."},
		{"a common password", ResetPassword("erin@corp.com", "Password1!"), "the password is too common"},
		{"another account's address", SetEmail("erin@corp.com", "frank@corp.com"), "An account with this e-mail address already exists."},
		{"the same address", SetEmail("erin@corp.com", "ERIN@corp.com"), "The new e-mail address is the account's current one."},
		{"a bad new address", SetEmail("erin@corp.com", "erin"), "--new-email is not a valid e-mail address"},
		{"deactivating nobody", DeactivateUser("nobody@corp.com"), "No account has this e-mail address."},
		{"activating nobody", ActivateUser("nobody@corp.com"), "No account has this e-mail address."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _, err := runUsers(t, url, tt.cmd)
			if err == nil || err.Error() != tt.want || out != "" {
				t.Errorf("= %q, %v; want no line and %q", out, err, tt.want)
			}
		})
	}
	if got := stateOf(t, pool, createdAccount(t, url, pool, "gina@corp.com")); got.email != "gina@corp.com" {
		t.Fatalf("gina = %+v", got)
	}
	var emails string
	if err := pool.QueryRow(context.Background(), "SELECT string_agg(email, ',' ORDER BY email) FROM users").Scan(&emails); err != nil ||
		emails != "erin@corp.com,frank@corp.com,gina@corp.com" {
		t.Errorf("accounts = %s (%v), want the three created and nothing changed", emails, err)
	}
}

func TestCommandErrorKeepsOtherErrors(t *testing.T) {
	boom := errors.New("connection refused")
	if err := commandError(boom); !errors.Is(err, boom) {
		t.Errorf("commandError(%v) = %v, want it unchanged", boom, err)
	}
	if err := commandError(domain.ErrAccountNotFound); !errors.Is(err, domain.ErrAccountNotFound) {
		t.Errorf("commandError(%v) = %v, want it unchanged", domain.ErrAccountNotFound, err)
	}
	unknown := shared.Invalid(shared.FieldError{Field: "label", Message: "is too long"})
	if err := commandError(unknown); err == nil || err.Error() != "label is too long" {
		t.Errorf("commandError(%v) = %v, want the field's own name", unknown, err)
	}
}
```

`server/cmd/nerve/users_test.go`（新文件）：

```go
package main

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	argon2adapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/argon2"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// usersDatabase is a migrated database of its own, the environment that
// points nerve at it, and a pool on it for the assertions.
func usersDatabase(t *testing.T) ([]string, *pgxpool.Pool) {
	t.Helper()
	url := pgtest.NewDatabase(t)
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=" + url}, pool
}

// hasPassword reports whether the account with email has password, hashed
// with the test profile's argon2 parameters: the command hashes with
// auth.password.
func hasPassword(t *testing.T, pool *pgxpool.Pool, email, password string) bool {
	t.Helper()
	var hash string
	if err := pool.QueryRow(context.Background(), "SELECT password FROM users WHERE email = $1", email).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=64,t=1,p=1$") {
		t.Errorf("hash %s, want the test profile's parameters m=64,t=1,p=1", hash)
	}
	hasher := argon2adapter.New(argon2adapter.Params{MemoryKiB: 64, Iterations: 1, Parallelism: 1, MaxConcurrent: 1, MaxWait: time.Second},
		slog.New(slog.DiscardHandler))
	ok, _, err := hasher.Verify(context.Background(), password, hash)
	return err == nil && ok
}

// Without a terminal, the password is one line of standard input: only its
// line ending is removed, and a last line without one counts too
// (M2 design 3.17).
func TestUsersReadThePasswordFromStandardInput(t *testing.T) {
	environ, pool := usersDatabase(t)
	tests := []struct {
		email, input, password string
	}{
		{"ida@corp.com", "Tr0ub4dor&3\n", "Tr0ub4dor&3"},
		{"jan@corp.com", "Pass word1!\r\nignored\n", "Pass word1!"},
		{"kim@corp.com", " Tr0ub4dor&3 ", " Tr0ub4dor&3 "},
	}
	for _, tt := range tests {
		code, stdout, stderr := executeWithInput(context.Background(), environ, tt.input, "users", "create", "--email", tt.email)
		if code != 0 || stdout != "created user "+tt.email+"\n" || !hasPassword(t, pool, tt.email, tt.password) {
			t.Errorf("create %s from %q = %d %q (stderr %q); want the account with password %q", tt.email, tt.input, code, stdout, stderr, tt.password)
		}
	}

	code, stdout, _ := executeWithInput(context.Background(), environ, "N3w-Passw0rd!\n", "users", "reset-password", "--email", "ida@corp.com")
	if code != 0 || stdout != "password reset for ida@corp.com: revoked 0 sessions, 0 API tokens\n" || !hasPassword(t, pool, "ida@corp.com", "N3w-Passw0rd!") {
		t.Errorf("reset-password = %d %q, want ida's new password", code, stdout)
	}
}

// The commands that set no password do not read standard input: empty
// input is no error for them.
func TestUsersCommandsWithoutAPassword(t *testing.T) {
	environ, _ := usersDatabase(t)
	if code, _, stderr := executeWithInput(context.Background(), environ, "Tr0ub4dor&3\n", "users", "create", "--email", "lee@corp.com"); code != 0 {
		t.Fatalf("create = %d: %s", code, stderr)
	}
	for _, args := range [][]string{
		{"users", "deactivate", "--email", "lee@corp.com"},
		{"users", "activate", "--email", "lee@corp.com"},
		{"users", "set-email", "--email", "lee@corp.com", "--new-email", "lee@new.example"},
	} {
		if code, stdout, stderr := execute(context.Background(), environ, args...); code != 0 || stdout == "" {
			t.Errorf("nerve %s = %d %q (stderr %q), want 0 and its line", strings.Join(args, " "), code, stdout, stderr)
		}
	}
}

// A refused command exits 1 with one line on stderr and nothing on stdout.
func TestUsersCommandsFail(t *testing.T) {
	environ, _ := usersDatabase(t)
	tests := []struct {
		name  string
		input string
		args  []string
		want  string
	}{
		{"no password", "", []string{"users", "create", "--email", "may@corp.com"}, "nerve: read the password from standard input: EOF\n"},
		{"an empty password", "\n", []string{"users", "create", "--email", "may@corp.com"}, "nerve: the password is required\n"},
		{"no address", "Tr0ub4dor&3\n", []string{"users", "create"}, "nerve: required flag(s) \"email\" not set\n"},
		{"no new address", "", []string{"users", "set-email", "--email", "may@corp.com"}, "nerve: required flag(s) \"new-email\" not set\n"},
		{"an unknown account", "", []string{"users", "activate", "--email", "may@corp.com"}, "nerve: No account has this e-mail address.\n"},
		{"an unknown command", "", []string{"users", "delete"}, "nerve: unknown command \"delete\" for \"nerve users\"\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := executeWithInput(context.Background(), environ, tt.input, tt.args...)
			if code != 1 || stdout != "" || !strings.HasSuffix(stderr, tt.want) {
				t.Errorf("nerve %s = %d, stdout %q, stderr %q; want 1 and %q", strings.Join(tt.args, " "), code, stdout, stderr, tt.want)
			}
		})
	}
}

func TestBareUsersPrintsHelp(t *testing.T) {
	code, stdout, stderr := execute(context.Background(), nil, "users")

	if code != 0 || !strings.Contains(stdout, "reset-password") || stderr != "" {
		t.Errorf("nerve users = %d, stdout %q, stderr %q; want 0 and the help", code, stdout, stderr)
	}
}
```

`server/cmd/nerve/main_test.go`（对 Task 5 版本的差异）：

```diff
--- a/server/cmd/nerve/main_test.go
+++ b/server/cmd/nerve/main_test.go
@@ -16,8 +16,13 @@
 )
 
 func execute(ctx context.Context, environ []string, args ...string) (code int, stdout, stderr string) {
+	return executeWithInput(ctx, environ, "", args...)
+}
+
+// executeWithInput runs a command line with input on its standard input.
+func executeWithInput(ctx context.Context, environ []string, input string, args ...string) (code int, stdout, stderr string) {
 	var out, errOut bytes.Buffer
-	code = run(ctx, args, environ, &out, &errOut)
+	code = run(ctx, args, environ, strings.NewReader(input), &out, &errOut)
 	return code, out.String(), errOut.String()
 }
 
```

- [ ] **Step 7: lint、测试和端到端**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。

Run: `make lint-web`
Expected: 通过（`schema.gen.ts` 只有说明文字改变）。

Run: `make knip`
Expected: 通过。

Run: `make e2e`
Expected: 全部通过（此前的故事；新故事在 Task 11）。

- [ ] **Step 8: 提交**

```bash
git add server/internal/modules/identity/admin.go server/internal/bootstrap server/cmd/nerve server/go.mod server/configs/config.yaml api web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M2/P3b): nerve users create, reset-password, set-email, deactivate and activate

The command line composes the pool and identity's administrator use cases
only: no HTTP server, no jobs client. A password comes from the terminal,
asked twice, or from one line of standard input.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**Done when:** 五个命令在真实数据库上输出各自的一行并改对数据；错误是一行、退出码 1、数据库不变；命令行的组合里没有 River 客户端。

---

### Task 9: 不可用的密码保持登录的耗时

**Files:**
- Modify: `server/internal/modules/identity/adapter/argon2/hasher.go`、`hasher_test.go`
- Modify: `server/internal/bootstrap/sessions_test.go`

**Interfaces:**
- Produces（spec 2.11；P2 评审第 6 节）：存的哈希不是本 hasher 写的 argon2id PHC 字符串时，它是"不可用的密码"（v0 不写这种值）：`Verify` 对一个替身（当前参数、启动时生成的随机盐和密钥）做一次同样的 argon2 计算，占用同样的名额（同样的等待和 503），然后答"不匹配"、没有错误，并记 WARN "a stored password hash is not an argon2id PHC string: no password matches it"（不带哈希）。原来它立即返回 `errNotOurHash`，登录变成很快的 500。
- 使用者：登录、修改密码（经 `PasswordHasher`）。

**Tests:**
- `hasher_test.go`：`TestVerifyMatchesNothingAgainstAnotherFormat`（原来的格式错误的各例：不匹配、没有错误、`parsePHC` 仍认出格式错误、WARN 不带哈希）；`TestVerifyOfAnotherFormatIsBusyWhenNoSlotFreesUp`（没有名额时同样是 503 `server_busy`）；`TestTheUnusableStandInHasTheCurrentParameters`（替身带当前参数，每个 hasher 的盐和密钥各不相同）。
- `bootstrap/sessions_test.go`：`TestLoginTakesAsLongForAnUnknownAddress` 加上第三种登录：密码列被改成 `!` 的账户。三种交替各 15 次，都是 401，中位数相差不到四分之一。

- [ ] **Step 1: hasher**

`server/internal/modules/identity/adapter/argon2/hasher.go`（对 `d6f313b` 的差异）：

```diff
--- a/server/internal/modules/identity/adapter/argon2/hasher.go
+++ b/server/internal/modules/identity/adapter/argon2/hasher.go
@@ -42,11 +42,17 @@
 	logger  *slog.Logger
 	slots   chan struct{}
 	waiting atomic.Int64
+	// unusable stands in for a stored hash in another format: Verify does
+	// the same work against it and matches no password.
+	unusable phc
 }
 
 // New returns a hasher with params p.
 func New(p Params, logger *slog.Logger) *Hasher {
-	return &Hasher{p: p, logger: logger, slots: make(chan struct{}, p.MaxConcurrent)}
+	unusable := phc{memoryKiB: p.MemoryKiB, iterations: p.Iterations, parallelism: p.Parallelism, salt: make([]byte, saltLen), key: make([]byte, keyLen)}
+	_, _ = rand.Read(unusable.salt) // never fails since Go 1.24
+	_, _ = rand.Read(unusable.key)
+	return &Hasher{p: p, logger: logger, slots: make(chan struct{}, p.MaxConcurrent), unusable: unusable}
 }
 
 // Hash returns the PHC string of password:
@@ -68,18 +74,27 @@
 // Verify reports whether password matches hash, a PHC string that Hash
 // wrote, and whether hash has other parameters than the current ones, so
 // that login hashes the password again (M2 design 3.8). It takes a slot like
-// Hash, with the same wait and the same 503. A hash in another format is an
-// error.
+// Hash, with the same wait and the same 503.
+//
+// A hash in another format is an unusable password (v0 writes none): it
+// matches no password, and Verify says so only after the work of a real
+// verification at the current parameters, so that a login's answer takes
+// as long as for any other account (M2 design 3.9). It is logged as a
+// warning, without the hash.
 func (h *Hasher) Verify(ctx context.Context, password, hash string) (ok, rehash bool, err error) {
-	p, err := parsePHC(hash)
-	if err != nil {
-		return false, false, err
+	p, parseErr := parsePHC(hash)
+	if parseErr != nil {
+		p = h.unusable
 	}
 	if err := h.acquire(ctx); err != nil {
 		return false, false, err
 	}
 	defer func() { <-h.slots }()
 	key := argon2.IDKey([]byte(password), p.salt, p.iterations, p.memoryKiB, p.parallelism, keyLen)
+	if parseErr != nil {
+		h.logger.WarnContext(ctx, "a stored password hash is not an argon2id PHC string: no password matches it")
+		return false, false, nil
+	}
 	ok = subtle.ConstantTimeCompare(key, p.key) == 1
 	rehash = p.memoryKiB != h.p.MemoryKiB || p.iterations != h.p.Iterations || p.parallelism != h.p.Parallelism
 	return ok, rehash, nil
@@ -93,7 +108,7 @@
 	salt, key   []byte
 }
 
-// errNotOurHash never quotes the hash.
+// errNotOurHash is parsePHC's error for a hash in another format.
 var errNotOurHash = errors.New("password hash is not an argon2id PHC string of this hasher")
 
 // parsePHC reads $argon2id$v=19$m=<KiB>,t=<iterations>,p=<lanes>$<salt>$<key>
```

- [ ] **Step 2: 测试**

`server/internal/modules/identity/adapter/argon2/hasher_test.go`（对 `d6f313b` 的差异）：

```diff
--- a/server/internal/modules/identity/adapter/argon2/hasher_test.go
+++ b/server/internal/modules/identity/adapter/argon2/hasher_test.go
@@ -151,8 +151,13 @@
 	}
 }
 
-func TestVerifyRejectsAnotherFormat(t *testing.T) {
-	h := New(testParams, slog.New(slog.DiscardHandler))
+// A hash in another format is an unusable password: no password matches
+// it, and it is no error, so a login with it answers 401 like any wrong
+// password rather than a quick 500 (M2 design 3.9). The warning never
+// quotes the hash.
+func TestVerifyMatchesNothingAgainstAnotherFormat(t *testing.T) {
+	var logs bytes.Buffer
+	h := New(testParams, slog.New(slog.NewJSONHandler(&logs, nil)))
 	good, err := h.Hash(context.Background(), "Tr0ub4dor&3")
 	if err != nil {
 		t.Fatal(err)
@@ -188,16 +193,48 @@
 		{"a leading part", "x" + good},
 	}
 	for _, tt := range tests {
+		logs.Reset()
 		ok, rehash, err := h.Verify(context.Background(), "Tr0ub4dor&3", tt.hash)
-		if ok || rehash || !errors.Is(err, errNotOurHash) {
-			t.Errorf("%s: Verify() = %v, rehash %v, %v; want the format error", tt.name, ok, rehash, err)
+		if ok || rehash || err != nil {
+			t.Errorf("%s: Verify() = %v, rehash %v, %v; want no match and no error", tt.name, ok, rehash, err)
 		}
-		if tt.hash != "" && strings.Contains(err.Error(), tt.hash) {
-			t.Errorf("%s: the error %q quotes the hash", tt.name, err)
+		if _, err := parsePHC(tt.hash); !errors.Is(err, errNotOurHash) {
+			t.Errorf("%s: parsePHC() = %v, want the format error", tt.name, err)
 		}
+		if out := logs.String(); !strings.Contains(out, `"level":"WARN","msg":"a stored password hash is not an argon2id PHC string: no password matches it"`) ||
+			tt.hash != "" && strings.Contains(out, tt.hash) {
+			t.Errorf("%s: logs = %s, want the warning without the hash", tt.name, out)
+		}
 	}
 }
 
+// An unusable password costs a slot like any other: with none free, the
+// login waits and gets 503 the same way.
+func TestVerifyOfAnotherFormatIsBusyWhenNoSlotFreesUp(t *testing.T) {
+	h := New(testParams, slog.New(slog.DiscardHandler))
+	h.slots <- struct{}{}
+	h.slots <- struct{}{}
+
+	_, _, err := h.Verify(context.Background(), "Tr0ub4dor&3", "!")
+
+	var se *shared.Error
+	if !errors.As(err, &se) || se.Code != shared.CodeServerBusy {
+		t.Errorf("Verify() = %v, want 503 server_busy", err)
+	}
+}
+
+// The stand-in for an unusable password has the current parameters, so
+// verifying against it costs what verifying a current hash costs, and its
+// own salt and key.
+func TestTheUnusableStandInHasTheCurrentParameters(t *testing.T) {
+	p := Params{MemoryKiB: 128, Iterations: 3, Parallelism: 2, MaxConcurrent: 1, MaxWait: time.Second}
+	a, b := New(p, slog.New(slog.DiscardHandler)).unusable, New(p, slog.New(slog.DiscardHandler)).unusable
+	if a.memoryKiB != 128 || a.iterations != 3 || a.parallelism != 2 || len(a.salt) != saltLen || len(a.key) != keyLen ||
+		bytes.Equal(a.salt, b.salt) || bytes.Equal(a.key, b.key) {
+		t.Errorf("stand-ins %+v and %+v; want the parameters m=128,t=3,p=2 and random salts and keys", a, b)
+	}
+}
+
 // Verify takes a slot like Hash: a login waits and gets 503 the same way,
 // whether its address exists or not.
 func TestVerifyIsBusyWhenNoSlotFreesUp(t *testing.T) {
```

`server/internal/bootstrap/sessions_test.go`（对 `d6f313b` 的差异）：

```diff
--- a/server/internal/bootstrap/sessions_test.go
+++ b/server/internal/bootstrap/sessions_test.go
@@ -132,30 +132,39 @@
 	return s[len(s)/2]
 }
 
-// With the default argon2id parameters, a login for an unknown address
-// takes as long as one with a wrong password for a known one: both verify
-// one hash (M2 design 3.9). The two kinds alternate, and their medians
-// must lie within a quarter of each other.
+// With the default argon2id parameters, a login for an unknown address, or
+// for an account whose stored password is unusable (a hash in another
+// format, which v0 never writes), takes as long as one with a wrong
+// password for a known account: each verifies one hash (M2 design 3.9; P2
+// review section 6). The three kinds alternate, and their medians must lie
+// within a quarter of each other.
 func TestLoginTakesAsLongForAnUnknownAddress(t *testing.T) {
-	cfg := testConfig(t, pgtest.NewDatabase(t), false)
+	url := pgtest.NewDatabase(t)
+	cfg := testConfig(t, url, false)
 	cfg.Auth.Password.Argon2MemoryKiB, cfg.Auth.Password.Argon2Iterations = 19456, 2
 	base := startApp(t, cfg, migrations.FS())
-	registerAccount(t, apitest.Load(t), base, "timing@example.com")
+	contract := apitest.Load(t)
+	registerAccount(t, contract, base, "timing@example.com")
+	registerAccount(t, contract, base, "unusable@example.com")
+	if _, err := openPool(t, url).Exec(context.Background(), "UPDATE users SET password = '!' WHERE email = 'unusable@example.com'"); err != nil {
+		t.Fatal(err)
+	}
 
-	var known, unknown []time.Duration
+	var known, unknown, unusable []time.Duration
 	for range 15 {
 		statusKnown, k := login(t, base, "timing@example.com", "Wr0ng-password")
 		statusUnknown, u := login(t, base, "nobody@example.com", "Wr0ng-password")
-		if statusKnown != http.StatusUnauthorized || statusUnknown != http.StatusUnauthorized {
-			t.Fatalf("logins = %d, %d; want 401 for both", statusKnown, statusUnknown)
+		statusUnusable, x := login(t, base, "unusable@example.com", "Tr0ub4dor&3")
+		if statusKnown != http.StatusUnauthorized || statusUnknown != http.StatusUnauthorized || statusUnusable != http.StatusUnauthorized {
+			t.Fatalf("logins = %d, %d, %d; want 401 for all three", statusKnown, statusUnknown, statusUnusable)
 		}
-		known, unknown = append(known, k), append(unknown, u)
+		known, unknown, unusable = append(known, k), append(unknown, u), append(unusable, x)
 	}
 
-	mk, mu := median(known), median(unknown)
-	t.Logf("median login: known address %v, unknown address %v", mk, mu)
-	if diff := max(mk, mu) - min(mk, mu); diff*4 > max(mk, mu) {
-		t.Errorf("median login: known address %v, unknown address %v; want them within a quarter of each other", mk, mu)
+	mk, mu, mx := median(known), median(unknown), median(unusable)
+	t.Logf("median login: known address %v, unknown address %v, unusable password %v", mk, mu, mx)
+	if diff := max(mk, mu, mx) - min(mk, mu, mx); diff*4 > max(mk, mu, mx) {
+		t.Errorf("median login: known address %v, unknown address %v, unusable password %v; want them within a quarter of each other", mk, mu, mx)
 	}
 }
 
```

- [ ] **Step 3: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run TestLoginTakesAsLongForAnUnknownAddress -v ./internal/bootstrap/`
Expected: `ok`；日志行 `median login: known address …, unknown address …, unusable password …` 三个值相近（原型中约 14.5 毫秒）。

- [ ] **Step 4: 提交**

```bash
git add server/internal/modules/identity/adapter/argon2 server/internal/bootstrap/sessions_test.go
```
```bash
git commit -m "fix(M2/P3b): an unusable stored password verifies as long as any and matches nothing

A hash in another format was a quick 500 on login; now it costs one
argon2 verification at the current parameters and answers 401.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 三种登录的耗时测试通过；不可用的密码登录答 401，不是 500。

---

### Task 10: 账户行锁的交错测试 1–3（真实数据库，真的争锁）

**Files:**
- Modify: `server/internal/modules/identity/interleavings_test.go`
- Create: `server/internal/modules/identity/interleavings_reset_test.go`

**Interfaces:**
- Consumes: `app.Login`（P2）、`app.CreateAPIToken`（P3a）、`app.ResetPassword`（Task 7）、`postgresadapter.Store`、`postgres.TxManager`、`pgtest.WaitForLockWait`。
- `interleavings_test.go` 的 `login` 帮助函数多一个 `passwords` 参数（交错 2 要把闸门放在登录写新哈希之前），四处调用照改；`await` 的失败信息改为 "the use case did not finish"。
- **闸门在持锁的事务里**（P3a 评审第 6 节）：先走的一方锁住账户行之后、第一次写入之前停在闸门上（`gatedPasswords` 停在 `UPDATE users` 之前：这条 `UPDATE` 自己也会锁住账户行，闸门放在它之后就试不出显式的锁；`gatedSessions` 停在插入会话之前；`gatedTokens` 停在插入 PAT 之前）。`contend` 先让第一方停在闸门上，再启动第二方，用 `pgtest.WaitForLockWait` 等到 Postgres 显示有语句在等锁，然后才打开闸门。没有锁、或锁不在事务里时，没有语句等锁，测试在 10 秒后失败。
- 交错 3 的调用者有两种凭证：原来的会话和一个 PAT（重置同样撤销它）；账户行锁对两种凭证是同一把锁。

**Tests:**（`interleavings_reset_test.go`，包 `identity_test`；每个等待都有 10 秒的期限）
- 交错 1：`TestALoginWaitingForAResetFails`（重置持锁，登录等锁；之后登录看到重置写的哈希，再校验一次失败：401，没有新会话，登录校验过的正是旧哈希和新哈希）；`TestAResetWaitsForALoginThatHoldsTheLock`（登录持锁；重置等它提交，然后连同登录的新会话一起撤销，共 2 个）。
- 交错 2：`TestALoginWaitingForAResetDoesNotWriteItsNewHash`（旧参数的哈希；登录已算好新哈希，等锁；之后看到重置的哈希，失败，不写自己的哈希）；`TestAResetWaitsForALoginThatHashesThePasswordAgain`（登录在写新哈希之前持锁；重置等它，然后用自己的哈希覆盖、撤销登录的会话）。
- 交错 3：`TestATokenCreationWaitingForAResetFails`（两个子测试：用会话、用 PAT；重置持锁，创建等锁；之后凭证已撤销：401，没有新 PAT）；`TestAResetWaitsForATokenCreationThatHoldsTheLock`（两个子测试；创建持锁，重置等它，然后连同新 PAT 一起撤销）。
- 每个测试最后读库核对：存的哈希、未撤销的会话数、`password_reset` 撤销的会话数、未撤销和已撤销的 PAT 数，以及重置返回的两个数目。

- [ ] **Step 1: 交错测试**

`server/internal/modules/identity/interleavings_test.go`（对 `d6f313b` 的差异）：

```diff
--- a/server/internal/modules/identity/interleavings_test.go
+++ b/server/internal/modules/identity/interleavings_test.go
@@ -27,9 +27,10 @@
 // The interleavings of the account row lock protocol (M2 design 3.5) run
 // the use cases on a real database. A gated hasher stops one of them in
 // argon2, outside its transaction, while the test commits another; a gated
-// session insert stops a login inside its transaction, holding the lock,
-// until another waits for the lock. Every wait has a deadline, so a test
-// fails rather than hangs.
+// write stops one inside its transaction, holding the lock, until another
+// waits for the lock. Every wait has a deadline, so a test fails rather
+// than hangs. Interleavings 1-3, with the administrator's reset, are in
+// interleavings_reset_test.go.
 
 // waitLimit bounds every wait of these tests.
 const waitLimit = 10 * time.Second
@@ -139,10 +140,10 @@
 	return a
 }
 
-func (a *account) login(h app.PasswordHasher, sessions app.SessionCreator) *app.Login {
+func (a *account) login(h app.PasswordHasher, passwords app.PasswordHashWriter, sessions app.SessionCreator) *app.Login {
 	keys := signing.EphemeralKeys()
 	return app.NewLogin(app.LoginDeps{
-		Accounts: a.store, Locker: a.store, Passwords: a.store, Sessions: sessions, Hasher: h, Tx: a.tx,
+		Accounts: a.store, Locker: a.store, Passwords: passwords, Sessions: sessions, Hasher: h, Tx: a.tx,
 		Issuance: app.Issuance{Tokens: signing.NewAccessTokens(keys), MAC: signing.NewRefreshTokenMAC(keys), AccessTTL: time.Minute, SessionTTL: time.Hour},
 		Clock:    clock.System{}, Logger: slog.New(slog.DiscardHandler), DummyHash: "hashed:dummy:0",
 	})
@@ -173,7 +174,7 @@
 	case err := <-done:
 		return err
 	case <-time.After(waitLimit):
-		t.Fatal("the login did not finish")
+		t.Fatal("the use case did not finish")
 		return nil
 	}
 }
@@ -202,7 +203,7 @@
 	g := newGate()
 	loginHasher := &gatedHasher{salt: "login", gate: g}
 
-	done := loginAsync(a.login(loginHasher, a.store))
+	done := loginAsync(a.login(loginHasher, a.store, a.store))
 	g.await(t)
 	changed := a.changePassword(&gatedHasher{salt: "change"}).Execute(
 		shared.WithActor(context.Background(), shared.Actor{UserID: a.id, SessionID: a.session}),
@@ -231,9 +232,9 @@
 	g := newGate()
 	first := &gatedHasher{salt: "first", gate: g}
 
-	done := loginAsync(a.login(first, a.store))
+	done := loginAsync(a.login(first, a.store, a.store))
 	g.await(t)
-	_, second := a.login(&gatedHasher{salt: "second"}, a.store).Execute(context.Background(), app.LoginInput{Email: "alice@corp.com", Password: "Tr0ub4dor&3"})
+	_, second := a.login(&gatedHasher{salt: "second"}, a.store, a.store).Execute(context.Background(), app.LoginInput{Email: "alice@corp.com", Password: "Tr0ub4dor&3"})
 	close(g.opened)
 	err := await(t, done)
 
@@ -258,7 +259,7 @@
 	a := newAccount(t, "hashed:Tr0ub4dor&3:0")
 	g := newGate()
 
-	done := loginAsync(a.login(&gatedHasher{salt: "login"}, gatedSessions{a.store, g}))
+	done := loginAsync(a.login(&gatedHasher{salt: "login"}, a.store, gatedSessions{a.store, g}))
 	g.await(t)
 	changed := make(chan error, 1)
 	go func() {
```

`server/internal/modules/identity/interleavings_reset_test.go`（新文件）：

```go
package identity_test

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Interleavings 1-3 of M2 design 3.5: a login, a login that hashes the
// password again, and a token creation, each against the administrator's
// reset of the password. Each runs both ways round on a real database: the
// operation that goes first stops at a gate inside its transaction, after
// it locked the account row and before its first write; the test starts
// the other, waits until pg_stat_activity shows it waiting for a lock, and
// only then opens the gate. Without the lock, or with the lock taken
// outside the transaction, nothing waits and the test fails.

// gatedPasswords stops a hash write inside its transaction, before the
// UPDATE, which would lock the account row by itself.
type gatedPasswords struct {
	app.PasswordHashWriter
	gate *gate
}

func (p gatedPasswords) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string, now time.Time) error {
	if err := p.gate.stop(); err != nil {
		return err
	}
	return p.PasswordHashWriter.UpdatePasswordHash(ctx, id, hash, now)
}

// gatedTokens stops a token creation inside its transaction, after the
// credential check, before the INSERT.
type gatedTokens struct {
	app.APITokenCreator
	gate *gate
}

func (c gatedTokens) CreateAPIToken(ctx context.Context, t app.NewAPIToken) error {
	if err := c.gate.stop(); err != nil {
		return err
	}
	return c.APITokenCreator.CreateAPIToken(ctx, t)
}

// reset sets alice's password to N3w-Passw0rd!, hashed as
// "hashed:N3w-Passw0rd!:reset".
func (a *account) reset(passwords app.PasswordHashWriter) (app.ResetPasswordResult, error) {
	return app.NewResetPassword(app.ResetPasswordDeps{
		Accounts: a.store, Passwords: passwords, Sessions: a.store, APITokens: a.store,
		Hasher: &gatedHasher{salt: "reset"}, Rules: domain.NewPasswordRules(), Tx: a.tx,
		Clock: clock.System{}, Logger: slog.New(slog.DiscardHandler),
	}).Execute(context.Background(), "alice@corp.com", "N3w-Passw0rd!")
}

// createToken creates a token as actor.
func (a *account) createToken(tokens app.APITokenCreator, actor shared.Actor) error {
	_, err := app.NewCreateAPIToken(app.CreateAPITokenDeps{
		Lock:   app.CredentialLock{Locker: a.store, Sessions: a.store, APITokens: a.store},
		Tokens: tokens, Tx: a.tx, Clock: clock.System{}, Logger: slog.New(slog.DiscardHandler),
	}).Execute(shared.WithActor(context.Background(), actor), domain.APITokenSpec{})
	return err
}

// credentials of alice that create a token: her session of before, or a
// token of hers made before, which the reset revokes as well. The lock is
// the account's row, whichever authenticates.
var creators = []struct {
	name   string
	actor  func(t *testing.T, a *account) shared.Actor
	tokens int // alice's tokens before the creation
}{
	{"with a session", func(_ *testing.T, a *account) shared.Actor {
		return shared.Actor{UserID: a.id, SessionID: a.session}
	}, 0},
	{"with a token", func(t *testing.T, a *account) shared.Actor {
		id := uuid.NewV7()
		hash := make([]byte, 32)
		hash[0] = 1
		if err := a.store.CreateAPIToken(context.Background(), app.NewAPIToken{ID: id, UserID: a.id, TokenHash: hash, Label: "script", Now: time.Now()}); err != nil {
			t.Fatal(err)
		}
		return shared.Actor{UserID: a.id, APITokenID: id}
	}, 1},
}

// signIn logs alice in with Tr0ub4dor&3.
func signIn(uc *app.Login) error {
	_, err := uc.Execute(context.Background(), app.LoginInput{Email: "alice@corp.com", Password: "Tr0ub4dor&3"})
	return err
}

// contend runs first until it stops at g, then second until Postgres shows
// a statement waiting for a lock, then opens g; it returns both errors.
func contend(t *testing.T, a *account, g *gate, first, second func() error) (error, error) {
	t.Helper()
	firstDone := run(first)
	g.await(t)
	secondDone := run(second)
	pgtest.WaitForLockWait(t, a.pool, waitLimit)
	close(g.opened)
	return await(t, firstDone), await(t, secondDone)
}

func run(f func() error) <-chan error {
	done := make(chan error, 1)
	go func() { done <- f() }()
	return done
}

// credentials is what the account holds: its stored hash, its live
// sessions, its sessions revoked by a reset, its live and its revoked
// tokens.
type credentials struct {
	hash                            string
	liveSessions, resetSessions     int
	liveAPITokens, revokedAPITokens int
}

func (a *account) credentials(t *testing.T) credentials {
	t.Helper()
	var c credentials
	err := a.pool.QueryRow(context.Background(), `SELECT password,
		(SELECT count(*) FROM auth_sessions WHERE user_id = $1 AND revoked_at IS NULL),
		(SELECT count(*) FROM auth_sessions WHERE user_id = $1 AND revoke_reason = 'password_reset'),
		(SELECT count(*) FROM api_tokens WHERE user_id = $1 AND deleted_at IS NULL),
		(SELECT count(*) FROM api_tokens WHERE user_id = $1 AND deleted_at IS NOT NULL)
		FROM users WHERE id = $1`, a.id).Scan(&c.hash, &c.liveSessions, &c.resetSessions, &c.liveAPITokens, &c.revokedAPITokens)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// Interleaving 1: a login verified the old password; the reset holds the
// lock. The login's transaction waits for it, then finds the reset's hash,
// verifies the password against it and fails: no new session.
func TestALoginWaitingForAResetFails(t *testing.T) {
	a := newAccount(t, "hashed:Tr0ub4dor&3:0")
	g := newGate()
	hasher := &gatedHasher{salt: "login"}
	var reset app.ResetPasswordResult

	resetErr, loginErr := contend(t, a, g,
		func() (err error) { reset, err = a.reset(gatedPasswords{a.store, g}); return err },
		func() error { return signIn(a.login(hasher, a.store, a.store)) })

	got := a.credentials(t)
	if resetErr != nil || !errors.Is(loginErr, domain.ErrInvalidCredentials) || reset.Sessions != 1 ||
		got != (credentials{hash: "hashed:N3w-Passw0rd!:reset", resetSessions: 1}) {
		t.Errorf("reset %+v, %v; login %v; credentials %+v; want the reset, 401 and no new session", reset, resetErr, loginErr, got)
	}
	if want := []string{"hashed:Tr0ub4dor&3:0", "hashed:N3w-Passw0rd!:reset"}; !slices.Equal(hasher.verified, want) {
		t.Errorf("the login verified %q, want %q", hasher.verified, want)
	}
}

// Interleaving 1 the other way round: the login holds the lock, so the
// reset waits for it and then revokes the login's session with the rest.
func TestAResetWaitsForALoginThatHoldsTheLock(t *testing.T) {
	a := newAccount(t, "hashed:Tr0ub4dor&3:0")
	g := newGate()
	var reset app.ResetPasswordResult

	loginErr, resetErr := contend(t, a, g,
		func() error { return signIn(a.login(&gatedHasher{salt: "login"}, a.store, gatedSessions{a.store, g})) },
		func() (err error) { reset, err = a.reset(a.store); return err })

	got := a.credentials(t)
	if loginErr != nil || resetErr != nil || reset.Sessions != 2 ||
		got != (credentials{hash: "hashed:N3w-Passw0rd!:reset", resetSessions: 2}) {
		t.Errorf("login %v; reset %+v, %v; credentials %+v; want both, the login's session revoked by the reset", loginErr, reset, resetErr, got)
	}
}

// Interleaving 2: a login verified the password against a hash of old
// parameters and hashed it again; the reset holds the lock. The login's
// transaction waits for it, finds the reset's hash and fails without
// writing its own over it.
func TestALoginWaitingForAResetDoesNotWriteItsNewHash(t *testing.T) {
	a := newAccount(t, "old:Tr0ub4dor&3")
	g := newGate()
	hasher := &gatedHasher{salt: "login"}
	var reset app.ResetPasswordResult

	resetErr, loginErr := contend(t, a, g,
		func() (err error) { reset, err = a.reset(gatedPasswords{a.store, g}); return err },
		func() error { return signIn(a.login(hasher, a.store, a.store)) })

	got := a.credentials(t)
	if resetErr != nil || !errors.Is(loginErr, domain.ErrInvalidCredentials) || reset.Sessions != 1 ||
		got != (credentials{hash: "hashed:N3w-Passw0rd!:reset", resetSessions: 1}) {
		t.Errorf("reset %+v, %v; login %v; credentials %+v; want the reset's hash, 401 and no new session", reset, resetErr, loginErr, got)
	}
	if want := []string{"old:Tr0ub4dor&3", "hashed:N3w-Passw0rd!:reset"}; !slices.Equal(hasher.verified, want) {
		t.Errorf("the login verified %q, want %q", hasher.verified, want)
	}
}

// Interleaving 2 the other way round: the login holds the lock before it
// writes its new hash, so the reset waits, then writes its own hash over
// the login's and revokes the login's session.
func TestAResetWaitsForALoginThatHashesThePasswordAgain(t *testing.T) {
	a := newAccount(t, "old:Tr0ub4dor&3")
	g := newGate()
	var reset app.ResetPasswordResult

	loginErr, resetErr := contend(t, a, g,
		func() error { return signIn(a.login(&gatedHasher{salt: "login"}, gatedPasswords{a.store, g}, a.store)) },
		func() (err error) { reset, err = a.reset(a.store); return err })

	got := a.credentials(t)
	if loginErr != nil || resetErr != nil || reset.Sessions != 2 ||
		got != (credentials{hash: "hashed:N3w-Passw0rd!:reset", resetSessions: 2}) {
		t.Errorf("login %v; reset %+v, %v; credentials %+v; want both, the reset's hash, the login's session revoked", loginErr, reset, resetErr, got)
	}
}

// Interleaving 3: a token creation with a credential of before; the reset
// holds the lock. The creation's transaction waits for it, finds the
// credential revoked and fails with 401: no new token.
func TestATokenCreationWaitingForAResetFails(t *testing.T) {
	for _, c := range creators {
		t.Run(c.name, func(t *testing.T) {
			a := newAccount(t, "hashed:Tr0ub4dor&3:0")
			actor := c.actor(t, a)
			g := newGate()
			var reset app.ResetPasswordResult

			resetErr, createErr := contend(t, a, g,
				func() (err error) { reset, err = a.reset(gatedPasswords{a.store, g}); return err },
				func() error { return a.createToken(a.store, actor) })

			var se *shared.Error
			got := a.credentials(t)
			if resetErr != nil || !errors.As(createErr, &se) || se.Code != shared.CodeUnauthorized || reset.APITokens != c.tokens ||
				got != (credentials{hash: "hashed:N3w-Passw0rd!:reset", resetSessions: 1, revokedAPITokens: c.tokens}) {
				t.Errorf("reset %+v, %v; creation %v; credentials %+v; want the reset, 401 and no new token", reset, resetErr, createErr, got)
			}
		})
	}
}

// Interleaving 3 the other way round: the creation holds the lock, so the
// reset waits for it and then revokes the new token with the rest.
func TestAResetWaitsForATokenCreationThatHoldsTheLock(t *testing.T) {
	for _, c := range creators {
		t.Run(c.name, func(t *testing.T) {
			a := newAccount(t, "hashed:Tr0ub4dor&3:0")
			actor := c.actor(t, a)
			g := newGate()
			var reset app.ResetPasswordResult

			createErr, resetErr := contend(t, a, g,
				func() error { return a.createToken(gatedTokens{a.store, g}, actor) },
				func() (err error) { reset, err = a.reset(a.store); return err })

			got := a.credentials(t)
			if createErr != nil || resetErr != nil || reset.APITokens != c.tokens+1 ||
				got != (credentials{hash: "hashed:N3w-Passw0rd!:reset", resetSessions: 1, revokedAPITokens: c.tokens + 1}) {
				t.Errorf("creation %v; reset %+v, %v; credentials %+v; want both, the new token revoked by the reset", createErr, reset, resetErr, got)
			}
		})
	}
}
```

- [ ] **Step 2: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。

Run: `go -C server test -count=5 -run 'TestA|TestTwo' ./internal/modules/identity/`
Expected: `ok`（六个交错测试的 9 个测试函数、4 个子测试，各 5 次）。

Run: `go -C server test -race -count=5 -run 'TestA|TestTwo' ./internal/modules/identity/`
Expected: `ok`，没有 `DATA RACE`。

- [ ] **Step 3: 提交**

```bash
git add server/internal/modules/identity/interleavings_test.go server/internal/modules/identity/interleavings_reset_test.go
```
```bash
git commit -m "test(M2/P3b): interleavings 1 to 3 of the account row lock contend with the administrator's reset

The gate sits inside the transaction that holds the lock; the test waits
until Postgres shows the other statement waiting, both ways round.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 交错 1–3 的 6 个测试（交错 3 的两种凭证）通过，`-count=5` 和 `-race -count=5` 也通过；六个交错测试至此全部在真实数据库上。

---

### Task 11: 端到端：A12–A14、A16、A17 的接口版本

**Files:**
- Modify: `e2e/fixtures/server.ts`、`e2e/fixtures/assert/identity.ts`
- Create: `e2e/fixtures/users.ts`
- Create: `e2e/stories/identity/a12-deactivate.spec.ts`、`a13-reset-password.spec.ts`、`a14-session-cleanup.spec.ts`、`a16-set-email.spec.ts`、`a17-create-user.spec.ts`

**Interfaces:**
- Consumes: Task 4–8 的命令、清理任务和接口。
- Produces（spec 2.13，M2 设计 2、9.5；M0-P6 交接）：
  - `runNerve(args, databaseUrl, input = "", env = {})`：标准输入写完就关闭，读密码的命令拿到文件结尾，不会一直等；非零退出时拒绝，错误带着退出码（`code`）、`stdout`、`stderr`；
  - `users.ts`：`nerveUsers(db, args, input?)` 返回命令输出的一行，并核对输出和日志里都没有密码；`nerveUsersFails(db, args, message, input?)` 核对退出码 1、标准输出为空、标准错误以 `nerve: <message>` 结尾。两者都以 `NERVE_LOG__LEVEL=debug` 运行：test 配置的日志级别是 `warn`，命令的日志在那个级别一行都不出，"日志里没有密码"就不可能失败（原型中发现，spec 附录 A）；
  - `assert/identity.ts`：`expectNewAccount`（A1、A17 共用：账户和默认资料）、`expectCreated`、`expectAllSessionsRevoked`、`expectNewPassword`（A7、A13 共用）、`expectDeactivated`、`expectPasswordReset`、`expectEmailChanged`；`expectRegistered`、`expectPasswordChanged` 改为调用共用的部分，行为不变。
- 故事的接口版本只用 PAT 和命令；每个故事另有一个账户，核对命令没有碰到它。

**Tests:**
- `a12-deactivate.spec.ts`：先把新手引导做完（四个步骤、`is_onboarded`、`is_tour_completed`、`last_workspace_id`），好让重置看得出来；PAT 调用 `POST /me/deactivate` 204，`expectDeactivated`（`is_active = false`、密码不变；全部会话 `deactivated`；新手引导回到默认；PAT 不变）；同一个 PAT 读 `/me` 401；原密码登录 403 `identity.account_deactivated`；`nerve users activate` 输出 `activated <email>: 1 API tokens are usable again`，PAT 200，登录 200；再做完新手引导后 `nerve users deactivate` 输出 `deactivated <email>: revoked 1 sessions`，数据库结果与自助停用相同。
- `a13-reset-password.spec.ts`：两个会话、一个 PAT；`nerve users reset-password` 从标准输入读新密码，输出 `password reset for <email>: revoked 2 sessions, 1 API tokens`；`expectPasswordReset`；旧密码 401、新密码 200；旧会话的刷新令牌 401；旧 PAT 401；另一个账户的会话和 PAT 照常。
- `a14-session-cleanup.spec.ts`：把一个会话的 `expires_at` 改到过去，`expect.poll`（15 秒为限；test 配置的间隔 2 秒）等它被删掉；另一个会话仍在、未撤销。
- `a16-set-email.spec.ts`：改成另一个账户的地址（大写）：退出码 1，"An account with this e-mail address already exists."，账户和会话不变；改成新地址（大写）：输出 `email changed from <旧> to <新>: revoked 1 sessions`，`expectEmailChanged`（地址已规范化、密码不变、全部会话 `email_changed`、PAT 不变）；旧地址登录 401，新地址 200；旧会话的刷新令牌 401；PAT 读 `/me` 得到新地址。
- `a17-create-user.spec.ts`：被占用的地址（大写）和常见密码都是退出码 1，数据库不变；`nerve users create --email <大写>` 输出 `created user <小写>`，`expectCreated`（账户和默认资料，没有会话）；新账户能登录。

- [ ] **Step 1: fixture**

`e2e/fixtures/server.ts`（对 `d6f313b` 的差异）：

```diff
--- a/e2e/fixtures/server.ts
+++ b/e2e/fixtures/server.ts
@@ -32,17 +32,28 @@
 }
 
 /**
- * Runs a nerve command, such as migrate up, with the test configuration on
- * the database at databaseUrl. It rejects when the command exits non-zero,
- * and kills it after commandTimeoutMs: global setup runs it before any
- * Playwright timeout applies.
+ * Runs a nerve command, such as migrate up or users create, with the test
+ * configuration on the database at databaseUrl, plus the variables of env.
+ * input, a password for instance, is its standard input, which is closed
+ * after it, so a command that reads more gets end of file rather than
+ * waiting. It rejects when the command exits non-zero, with the exit code as
+ * the error's code and its stdout and stderr, and kills it after
+ * commandTimeoutMs: global setup runs it before any Playwright timeout
+ * applies.
  */
-export async function runNerve(args: string[], databaseUrl: string): Promise<{ stdout: string; stderr: string }> {
-  return promisify(execFile)(binary, args, {
-    env: nerveEnv(databaseUrl),
+export async function runNerve(
+  args: string[],
+  databaseUrl: string,
+  input = "",
+  env: Record<string, string> = {}
+): Promise<{ stdout: string; stderr: string }> {
+  const run = promisify(execFile)(binary, args, {
+    env: { ...nerveEnv(databaseUrl), ...env },
     timeout: commandTimeoutMs,
     killSignal: "SIGKILL",
   });
+  run.child.stdin?.end(input);
+  return run;
 }
 
 /**
```

`e2e/fixtures/users.ts`（新文件）：

```ts
import { expect } from "@playwright/test";

import type { Database } from "./db";
import { runNerve } from "./server";

// The server administrator's commands, nerve users (M2 design 3.17), on a
// worker's database: the worker's nerve sees what they change on its next
// request.

/** Every log line, so that a password in any of them shows. */
const allLogs = { NERVE_LOG__LEVEL: "debug" };

/**
 * Runs nerve users with args, and input as its standard input, and returns
 * its output: the one line the command prints. Neither the output nor the
 * logs, at every level, hold the input, a password.
 */
export async function nerveUsers(db: Database, args: string[], input?: string): Promise<string> {
  const { stdout, stderr } = await runNerve(["users", ...args], db.url, input, allLogs);
  const secret = input?.trim();
  if (secret) {
    expect(stdout, "the output").not.toContain(secret);
    expect(stderr, "the logs").not.toContain(secret);
  }
  return stdout;
}

/**
 * Runs nerve users with args, and input as its standard input, and expects
 * it to fail: exit code 1, no output, and "nerve: <message>" as the last
 * line of stderr.
 */
export async function nerveUsersFails(db: Database, args: string[], message: string, input?: string): Promise<void> {
  const failure = await runNerve(["users", ...args], db.url, input, allLogs).then(
    () => undefined,
    (err: { code?: unknown; stdout?: string; stderr?: string }) => err
  );
  const label = `nerve users ${args.join(" ")}`;
  expect(failure, `${label} fails`).toBeDefined();
  expect(failure?.code, label).toBe(1);
  expect(failure?.stdout, label).toBe("");
  expect(failure?.stderr?.endsWith(`nerve: ${message}\n`), `${label}: ${failure?.stderr}`).toBe(true);
}
```

`e2e/fixtures/assert/identity.ts`（对 `d6f313b` 的差异）：

```diff
--- a/e2e/fixtures/assert/identity.ts
+++ b/e2e/fixtures/assert/identity.ts
@@ -89,12 +89,12 @@
 }
 
 /**
- * A1: registration added one account with the address lowercased, an
- * argon2id hash and the display name from the address; its default profile;
- * and its one session, a new one.
+ * A1, A17: one account has the address typed, lowercased, with an argon2id
+ * hash, the display name from the address, active, and its default
+ * profile. Returns its id.
  */
-export async function expectRegistered(db: Database, r: SignIn): Promise<void> {
-  const email = r.email.toLowerCase();
+async function expectNewAccount(db: Database, typed: string): Promise<string | undefined> {
+  const email = typed.toLowerCase();
   const users = await db.query<{ id: string; password: string; display_name: string; is_active: boolean }>(
     "SELECT id, password, display_name, is_active FROM users WHERE email = $1",
     [email]
@@ -126,11 +126,22 @@
       start_of_the_week: 0,
     },
   ]);
+  return user?.id;
+}
 
-  expect(await db.query("SELECT id FROM auth_sessions WHERE user_id = $1", [user?.id])).toHaveLength(1);
-  await expectNewSession(db, user?.id, r);
+/** A1: registration added a new account and its one session, a new one. */
+export async function expectRegistered(db: Database, r: SignIn): Promise<void> {
+  const id = await expectNewAccount(db, r.email);
+  expect(await db.query("SELECT id FROM auth_sessions WHERE user_id = $1", [id])).toHaveLength(1);
+  await expectNewSession(db, id, r);
 }
 
+/** A17: nerve users create added a new account of the address typed, without a session. */
+export async function expectCreated(db: Database, typed: string): Promise<void> {
+  const id = await expectNewAccount(db, typed);
+  expect(await db.query("SELECT id FROM auth_sessions WHERE user_id = $1", [id])).toEqual([]);
+}
+
 /** A3: a login added a new session to the account of the address. */
 export async function expectSignedIn(db: Database, s: SignIn): Promise<void> {
   const users = await db.query<{ id: string }>("SELECT id FROM users WHERE email = $1", [s.email.toLowerCase()]);
@@ -159,6 +170,28 @@
   expect(session.revoked_at).toBeNull();
 }
 
+/**
+ * A12, A13, A16: every session of the account userId, and it has one at
+ * least, is revoked for reason.
+ */
+async function expectAllSessionsRevoked(
+  db: Database,
+  userId: string,
+  reason: "deactivated" | "password_reset" | "email_changed"
+): Promise<void> {
+  const sessions = await db.query<{ id: string; revoked: boolean; revoke_reason: string | null }>(
+    "SELECT id, revoked_at IS NOT NULL AS revoked, revoke_reason FROM auth_sessions WHERE user_id = $1",
+    [userId]
+  );
+  expect(sessions.length).toBeGreaterThan(0);
+  for (const s of sessions) {
+    expect({ revoked: s.revoked, revoke_reason: s.revoke_reason }, `session ${s.id}`).toEqual({
+      revoked: true,
+      revoke_reason: reason,
+    });
+  }
+}
+
 /** A5, A6: the session of refreshToken is revoked, for reason. */
 export async function expectRevoked(
   db: Database,
@@ -198,11 +231,18 @@
   return db.query("SELECT id, deleted_at FROM api_tokens WHERE user_id = $1 ORDER BY created_at, id", [userId]);
 }
 
+/** A7, A13: the password of the account `before` changed, to another argon2id hash. */
+async function expectNewPassword(db: Database, before: AccountRow): Promise<void> {
+  const [after] = await db.query<{ password: string }>("SELECT password FROM users WHERE id = $1", [before.id]);
+  expect(after?.password).toMatch(/^\$argon2id\$/);
+  expect(after?.password).not.toBe(before.password);
+}
+
 /**
- * A7: the password of the account `before` changed, to another argon2id
- * hash; every session of the account is revoked for password_changed but
- * survivingSession, the refresh token of the page's own session, which
- * stays live; the personal access tokens are as they were (M2 design 3.5).
+ * A7: the password of the account `before` changed; every session of the
+ * account is revoked for password_changed but survivingSession, the refresh
+ * token of the page's own session, which stays live; the personal access
+ * tokens are as they were (M2 design 3.5).
  */
 export async function expectPasswordChanged(
   db: Database,
@@ -210,9 +250,7 @@
   tokensBefore: { id: string; deleted_at: Date | null }[],
   survivingSession?: string
 ): Promise<void> {
-  const [after] = await db.query<{ password: string }>("SELECT password FROM users WHERE id = $1", [before.id]);
-  expect(after?.password).toMatch(/^\$argon2id\$/);
-  expect(after?.password).not.toBe(before.password);
+  await expectNewPassword(db, before);
   const surviving = survivingSession === undefined ? undefined : (await sessionOf(db, survivingSession)).id;
   const sessions = await db.query<{ id: string; revoke_reason: string | null }>(
     "SELECT id, revoke_reason FROM auth_sessions WHERE user_id = $1",
@@ -221,7 +259,75 @@
   expect(sessions.length).toBeGreaterThan(0);
   for (const s of sessions) {
     expect(s.revoke_reason, `session ${s.id}`).toBe(s.id === surviving ? null : "password_changed");
+  }
+  expect(await tokensOf(db, before.id)).toEqual(tokensBefore);
+}
+
+/**
+ * A12: the account `before` is deactivated and its password unchanged;
+ * every session is revoked for deactivated; its onboarding starts over;
+ * its personal access tokens are as they were (M2 design 3.5, decision 3).
+ */
+export async function expectDeactivated(
+  db: Database,
+  before: AccountRow,
+  tokensBefore: { id: string; deleted_at: Date | null }[]
+): Promise<void> {
+  expect(await db.query("SELECT is_active, password FROM users WHERE id = $1", [before.id])).toEqual([
+    { is_active: false, password: before.password },
+  ]);
+  await expectAllSessionsRevoked(db, before.id, "deactivated");
+  expect(
+    await db.query(
+      "SELECT onboarding_step, is_onboarded, is_tour_completed, last_workspace_id FROM profiles WHERE user_id = $1",
+      [before.id]
+    )
+  ).toEqual([
+    {
+      onboarding_step: {
+        profile_complete: false,
+        workspace_create: false,
+        workspace_invite: false,
+        workspace_join: false,
+      },
+      is_onboarded: false,
+      is_tour_completed: false,
+      last_workspace_id: null,
+    },
+  ]);
+  expect(await tokensOf(db, before.id)).toEqual(tokensBefore);
+}
+
+/**
+ * A13: the password of the account `before` changed; every session is
+ * revoked for password_reset and every personal access token deleted (M2
+ * design 3.5).
+ */
+export async function expectPasswordReset(db: Database, before: AccountRow): Promise<void> {
+  await expectNewPassword(db, before);
+  await expectAllSessionsRevoked(db, before.id, "password_reset");
+  const tokens = await tokensOf(db, before.id);
+  expect(tokens.length).toBeGreaterThan(0);
+  for (const t of tokens) {
+    expect(t.deleted_at, `token ${t.id}`).not.toBeNull();
   }
+}
+
+/**
+ * A16: the account `before` has the address email, and its password; every
+ * session is revoked for email_changed; its personal access tokens are as
+ * they were (M2 decision 1).
+ */
+export async function expectEmailChanged(
+  db: Database,
+  before: AccountRow,
+  email: string,
+  tokensBefore: { id: string; deleted_at: Date | null }[]
+): Promise<void> {
+  expect(await db.query("SELECT email, password FROM users WHERE id = $1", [before.id])).toEqual([
+    { email, password: before.password },
+  ]);
+  await expectAllSessionsRevoked(db, before.id, "email_changed");
   expect(await tokensOf(db, before.id)).toEqual(tokensBefore);
 }
 
```

- [ ] **Step 2: 五个故事**

`e2e/stories/identity/a12-deactivate.spec.ts`（新文件）：

```ts
import { randomUUID } from "node:crypto";

import type { Api } from "../../fixtures/api";
import { accountOf, expectDeactivated, sessionOf, tokensOf } from "../../fixtures/assert/identity";
import { bearer, createPAT, emailFor, login, password, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";
import { nerveUsers } from "../../fixtures/users";

// A12, deactivating an account (M2 design 2, decision 3): the API version,
// with the administrator's nerve users activate and deactivate. The page
// version joins in M2/P5.

/** Finishes onboarding, so that starting it over shows. */
async function onboard(api: Api, token: string): Promise<void> {
  const { response } = await api.PATCH("/api/v0/me/profile", {
    body: {
      onboarding_step: { profile_complete: true, workspace_create: true, workspace_invite: true, workspace_join: true },
      is_onboarded: true,
      is_tour_completed: true,
      last_workspace_id: randomUUID(),
    },
    headers: bearer(token),
  });
  expect(response.status).toBe(200);
}

test("A12 (API): a token deactivates the account; nerve users activate brings it and its tokens back, deactivate does as the API did", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const pat = await createPAT(api, (await register(api, email)).access_token);
  await onboard(api, pat.token);
  // Another account, which nothing here changes.
  const other = await register(api, emailFor(testInfo, "other"));
  const before = await accountOf(db, email);
  const tokensBefore = await tokensOf(db, before.id);

  const deactivated = await api.POST("/api/v0/me/deactivate", { headers: bearer(pat.token) });
  expect(deactivated.response.status).toBe(204);
  await expectDeactivated(db, before, tokensBefore);
  expect((await sessionOf(db, other.refresh_token)).revoked_at).toBeNull();

  // Nothing authenticates as the account: the token fails, the password is refused.
  expect((await api.GET("/api/v0/me", { headers: bearer(pat.token) })).response.status).toBe(401);
  const refused = await api.POST("/api/v0/auth/login", { body: { email, password } });
  expect(refused.response.status).toBe(403);
  expect(refused.error?.code).toBe("identity.account_deactivated");

  // The administrator activates it: the same token and password work again.
  expect(await nerveUsers(db, ["activate", "--email", email])).toBe(
    `activated ${email}: 1 API tokens are usable again\n`
  );
  expect((await api.GET("/api/v0/me", { headers: bearer(pat.token) })).response.status).toBe(200);
  await login(api, email);

  // The administrator's deactivate leaves the database as the API did.
  await onboard(api, pat.token);
  const again = await accountOf(db, email);
  expect(await nerveUsers(db, ["deactivate", "--email", email])).toBe(`deactivated ${email}: revoked 1 sessions\n`);
  await expectDeactivated(db, again, tokensBefore);
  expect((await sessionOf(db, other.refresh_token)).revoked_at).toBeNull();
});
```

`e2e/stories/identity/a13-reset-password.spec.ts`（新文件）：

```ts
import { accountOf, expectPasswordReset, sessionOf } from "../../fixtures/assert/identity";
import { bearer, createPAT, emailFor, login, password, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";
import { nerveUsers } from "../../fixtures/users";

// A13, the administrator resets a password (M2 design 2, 3.5): the way back
// into an account whose credentials leaked. There is no page version.

const newPassword = "N3w-Passw0rd!";

test("A13: nerve users reset-password sets the password from stdin and revokes every session and token", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const first = await register(api, email);
  const second = await login(api, email);
  const pat = await createPAT(api, first.access_token);
  // Another account, which nothing here changes.
  const other = await register(api, emailFor(testInfo, "other"));
  const otherPAT = await createPAT(api, other.access_token);
  const before = await accountOf(db, email);

  expect(await nerveUsers(db, ["reset-password", "--email", email], `${newPassword}\n`)).toBe(
    `password reset for ${email}: revoked 2 sessions, 1 API tokens\n`
  );
  await expectPasswordReset(db, before);
  expect((await sessionOf(db, other.refresh_token)).revoked_at).toBeNull();
  expect((await api.GET("/api/v0/me", { headers: bearer(otherPAT.token) })).response.status).toBe(200);

  // The new password signs in, the old one not; the old refresh token and
  // the old token fail.
  expect((await api.POST("/api/v0/auth/login", { body: { email, password } })).response.status).toBe(401);
  const renewed = await api.POST("/api/v0/auth/login", { body: { email, password: newPassword } });
  expect(renewed.response.status).toBe(200);
  const refreshed = await api.POST("/api/v0/auth/refresh", { body: { refresh_token: second.refresh_token } });
  expect(refreshed.response.status).toBe(401);
  expect((await api.GET("/api/v0/me", { headers: bearer(pat.token) })).response.status).toBe(401);
});
```

`e2e/stories/identity/a14-session-cleanup.spec.ts`（新文件）：

```ts
import { sessionOf } from "../../fixtures/assert/identity";
import { emailFor, login, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A14, expired sessions are cleaned up (M2 design 2, 3.15): River's periodic
// job runs every auth.session_cleanup_interval, 2 s in the test
// configuration. There is no page version.

test("A14: the cleanup job deletes an expired session and keeps the live one", async ({ api, db }, testInfo) => {
  const email = emailFor(testInfo);
  const expired = await register(api, email);
  const live = await login(api, email);
  const { id } = await sessionOf(db, expired.refresh_token);
  await db.query("UPDATE auth_sessions SET expires_at = now() - interval '1 minute' WHERE id = $1", [id]);

  await expect
    .poll(async () => (await db.query("SELECT id FROM auth_sessions WHERE id = $1", [id])).length, {
      message: "the expired session is deleted",
      timeout: 15_000,
    })
    .toBe(0);
  expect((await sessionOf(db, live.refresh_token)).revoked_at).toBeNull();
});
```

`e2e/stories/identity/a16-set-email.spec.ts`（新文件）：

```ts
import { accountOf, expectEmailChanged, sessionOf, tokensOf } from "../../fixtures/assert/identity";
import { bearer, createPAT, emailFor, login, password, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";
import { nerveUsers, nerveUsersFails } from "../../fixtures/users";

// A16, the administrator changes an address (M2 design 2, decision 1): the
// only way to change one. There is no page version.

test("A16: nerve users set-email changes the address and signs every session out; the tokens stay", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const first = await register(api, email);
  const pat = await createPAT(api, first.access_token);
  const taken = emailFor(testInfo, "taken");
  const other = await register(api, taken);
  const newEmail = emailFor(testInfo, "new");
  const before = await accountOf(db, email);
  const tokensBefore = await tokensOf(db, before.id);

  // Another account's address, whatever its case: exit code 1, and nothing changes.
  await nerveUsersFails(
    db,
    ["set-email", "--email", email, "--new-email", taken.toUpperCase()],
    "An account with this e-mail address already exists."
  );
  expect(await accountOf(db, email)).toEqual(before);
  expect((await sessionOf(db, first.refresh_token)).revoked_at).toBeNull();

  expect(await nerveUsers(db, ["set-email", "--email", email, "--new-email", newEmail.toUpperCase()])).toBe(
    `email changed from ${email} to ${newEmail}: revoked 1 sessions\n`
  );
  await expectEmailChanged(db, before, newEmail, tokensBefore);
  expect((await sessionOf(db, other.refresh_token)).revoked_at).toBeNull();

  // The old address no longer signs in, the new one does; the old refresh
  // token fails; the token goes on, and sees the new address.
  expect((await api.POST("/api/v0/auth/login", { body: { email, password } })).response.status).toBe(401);
  await login(api, newEmail);
  const refreshed = await api.POST("/api/v0/auth/refresh", { body: { refresh_token: first.refresh_token } });
  expect(refreshed.response.status).toBe(401);
  const me = await api.GET("/api/v0/me", { headers: bearer(pat.token) });
  expect(me.response.status).toBe(200);
  expect(me.data?.email).toBe(newEmail);
});
```

`e2e/stories/identity/a17-create-user.spec.ts`（新文件）：

```ts
import { countIdentity, expectCreated, expectNothingAdded } from "../../fixtures/assert/identity";
import { emailFor, login, password, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";
import { nerveUsers, nerveUsersFails } from "../../fixtures/users";

// A17, the administrator creates an account (M2 design 2, decision 2): how
// the first account comes to be while sign-up is closed. There is no page
// version.

test("A17: nerve users create makes an account with the password from stdin; a taken address or a weak password adds nothing", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const taken = emailFor(testInfo, "taken");
  await register(api, taken);
  const before = await countIdentity(db);

  await nerveUsersFails(
    db,
    ["create", "--email", taken.toUpperCase()],
    "An account with this e-mail address already exists.",
    `${password}\n`
  );
  await nerveUsersFails(db, ["create", "--email", email], "the password is too common", "Password1!\n");
  await expectNothingAdded(db, before);

  expect(await nerveUsers(db, ["create", "--email", email.toUpperCase()], `${password}\n`)).toBe(
    `created user ${email}\n`
  );
  await expectCreated(db, email);
  await login(api, email);
});
```

- [ ] **Step 3: 检查和端到端**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。

Run: `make lint-web`
Expected: 通过（e2e 的类型检查、oxlint、格式）。

Run: `make knip`
Expected: 通过（新的导出都有使用者）。

Run: `make e2e`
Expected: 全部通过：S1–S4，A1–A17 的接口版本（A12–A14、A16、A17 是新的）。

- [ ] **Step 4: 停机时间**

`make e2e` 通过本身说明每个 worker 的 nerve（River 在运行）都在 fixture 的 `stopTimeoutMs`（30 秒）之内以退出码 0 停下，否则 fixture 报 "nerve did not exit cleanly"。精确的停机时间按 spec 附录 A 的方法实测，写进 review（M0-P6 交接）。

- [ ] **Step 5: 提交**

```bash
git add e2e/fixtures e2e/stories/identity
```
```bash
git commit -m "test(M2/P3b): the API versions of stories A12, A13, A14, A16 and A17

runNerve feeds a command's standard input; the administrator's commands run
with every log level, so a password in any log line fails the story.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** `make e2e` 全部通过，含五个新故事；`make lint-web`、`make knip` 通过。

---

### Task 12: 上级文档、差异清单、README 与交接

**Files:**
- Modify: `docs/v0/v0-design.md`、`docs/v0/plane-diff.md`、`README.md`
- Modify: `docs/v0/M2-auth/handoffs/M0-P2-platform-notes.md`、`M0-P6-e2e-notes.md`、`M1-P3-trim-platform.md`

**Interfaces:**
- M2 设计 3.20 中 P3b 的各行、8.7 中 P3b 的三行（spec 2.14）；交接的处理结果（spec 第 7 节）。三份交接仍为 `open`（各自还有 P4 或 M8 的条目）。

- [ ] **Step 1: 总体设计**（4.2：管理员命令的撤销规则、恢复后 PAT 重新可用；账户行锁包括管理员的命令；`nerve users create` 不再标"P3b 加入"；忘记密码时的重置同时撤销全部会话和 PAT）

`docs/v0/v0-design.md`（对 `d6f313b` 的差异）：

```diff
--- a/docs/v0/v0-design.md
+++ b/docs/v0/v0-design.md
@@ -227,12 +227,12 @@
 
 ### 4.2 规则
 - **权限不放进令牌**：每个请求都从数据库读取成员关系和角色。所以移出项目、调整角色是立即生效的。
-- **会话撤销**：退出登录只结束当前这一处登录，同一账户的其他会话不受影响（M2 设计 11.1）；修改密码结束该账户的其他会话，用 PAT 修改时没有当前会话，全部结束，PAT 不受影响；停用账户结束该账户的全部会话，PAT 不删除，但停用期间认证失败（M2 设计 3.5）。
-- **账户行锁**：签发和变更凭证的事务（登录、创建 PAT、修改密码、停用）先锁账户行，再确认调用者的凭证仍然有效，所以并发的修改不会让已被撤销的凭证再签发或变更凭证；续期靠会话行上的条件更新（M2 设计 3.5）。
+- **会话撤销**：退出登录只结束当前这一处登录，同一账户的其他会话不受影响（M2 设计 11.1）；修改密码结束该账户的其他会话，用 PAT 修改时没有当前会话，全部结束，PAT 不受影响；停用账户结束该账户的全部会话，PAT 不删除，但停用期间认证失败（M2 设计 3.5）。服务器管理员的 `nerve users reset-password` 结束该账户的全部会话并撤销全部 PAT，`set-email` 结束全部会话，`deactivate` 与自助停用相同；`activate` 恢复账户后，没有过期的 PAT 重新可用（M2 设计 3.5、3.17）。
+- **账户行锁**：签发和变更凭证的事务（登录、创建 PAT、修改密码、停用，以及服务器管理员的命令）先锁账户行，再确认调用者的凭证仍然有效，所以并发的修改不会让已被撤销的凭证再签发或变更凭证；续期靠会话行上的条件更新（M2 设计 3.5）。
 - **重复使用检测**：一旦发现某个已经换过新的刷新令牌又被使用，就作废这次登录派生出的所有令牌。只有交出的旧令牌确是这个会话签发过的（它的 MAC 标签成立）才作废；会话 id 和代数对、密文和标签是伪造的旧令牌得到 401，会话不受影响（M2 设计 3.5）。
 - **密码哈希**：用 argon2id。
-- **登录方式**：v0 只有邮箱加密码。是否开放注册由配置 `auth.signup_enabled` 控制：prod 默认关闭，dev、test 默认开放；关闭时注册先答"注册已关闭"，不查邮箱。prod 的第一个账户由服务器管理员用 `nerve users create` 创建（M2/P3b 加入；在那之前用 `NERVE_AUTH__SIGNUP_ENABLED=true` 临时打开注册）（M2 设计决策点 2）。被邀请的邮箱始终可以注册。
-- **忘记密码**：没有邮件服务，由服务器管理员通过命令行重置：`nerve users reset-password --email <email>`。
+- **登录方式**：v0 只有邮箱加密码。是否开放注册由配置 `auth.signup_enabled` 控制：prod 默认关闭，dev、test 默认开放；关闭时注册先答"注册已关闭"，不查邮箱。prod 的第一个账户由服务器管理员用 `nerve users create` 创建（M2 设计决策点 2）。被邀请的邮箱始终可以注册。
+- **忘记密码**：没有邮件服务，由服务器管理员通过命令行重置：`nerve users reset-password --email <email>`，它同时结束该账户的全部会话、撤销全部 PAT。
 
 ### 4.3 浏览器端
 - **令牌存放**：访问令牌存在内存；刷新令牌存在 localStorage。
```

- [ ] **Step 2: 差异清单**（二·按表：River 的表；四：管理员重置密码、重置命令的邮箱、修改登录邮箱、创建账户的命令四行新增，密码规则、注册默认、停用账户三行改写）

`docs/v0/plane-diff.md`（对 `d6f313b` 的差异）：

```diff
--- a/docs/v0/plane-diff.md
+++ b/docs/v0/plane-diff.md
@@ -100,6 +100,7 @@
 | `api_tokens` | 删除 `user_type`、`workspace_id`、`is_active`、`is_service`、`allowed_rate_limit` | 不区分人和机器人；Plane 自己已把个人令牌的 `workspace_id` 置空；撤销用软删除，`is_active` 没有独立的写入方；社区版不创建服务令牌；没有限流读取 `allowed_rate_limit` |
 | `api_tokens` | 索引 `api_tokens_user_id_created_at_idx ON (user_id, created_at DESC, id DESC) WHERE deleted_at IS NULL` | 列表的游标分页（M2 设计 3.12） |
 | `auth_sessions` | **新增**（M2/P1，`00003_identity_auth_sessions.sql`，替代 `sessions`，见一 B）：一次登录一行，12 列：`id`（访问令牌中的 `sid`）、`user_id`（`ON DELETE CASCADE`）、`token_hash`（当前一代刷新令牌密文的 SHA-256，32 字节）、`generation`（代数）、`user_agent`、`ip`（`inet`）、`expires_at`（登录时刻加会话期限，之后不变）、`last_refreshed_at`、`revoked_at`、`revoke_reason`（六个取值）、`created_at`、`updated_at`；`auth_sessions_revoked_consistent_check` 要求 `revoked_at` 与 `revoke_reason` 同时为空或同时有值；索引 `auth_sessions_user_id_idx`、`auth_sessions_expires_at_idx`。旧代的刷新令牌不存，由令牌里的 HMAC 标签认出 | JWT 认证；刷新令牌的轮换和重复使用检测（M2 设计 3.4、3.5、4.5） |
+| River 的表 | **新增**的基础设施表（M2/P3b，`00005_river_main_v2_to_v7.sql`）：`river_job`、`river_leader`（`UNLOGGED`）、`river_queue`、`river_notification`，枚举 `river_job_state`，函数 `river_job_state_in_bitmask`。内容是 River v0.47.0 主线第 2–7 版迁移的原样导出（`river migrate-get --line main --all --exclude-version 1`），不建 `river_migration`，版本由 goose 管理；表、约束和索引的名字随 River，不按二·全局的约定改 | 后台任务和定时任务改用 River，与业务数据同库，替代 Plane 的 Celery 和 Celery Beat（v0 总体设计 5.2、6.7；M2 设计 3.15） |
 | `workspaces` | 删除旧的 `logo` URL 列 | 遗留列 |
 | `workspace_members` | 删除 `view_props`、`default_props` | 遗留列 |
 | `projects` | 删除 `emoji`、`icon_prop`、旧的 `cover_image`、`description_text`、`description_html`（旧的 json 列）、`page_view`、`is_time_tracking_enabled`、`is_issue_type_enabled`、`estimate_id`、`close_in` | 遗留列或对应功能已砍掉（归档保留，`archive_in` 和 `archived_at` 保留） |
@@ -156,9 +157,11 @@
 | 归档 | 工作项、迭代、模块、项目的归档与恢复，以及项目级自动归档 | 规则一致（见 v0-design 5.5）；归档和恢复改为 `POST .../archive` 和 `POST .../unarchive` 两个动作接口；列表通过 `?archived=true` 查询已归档的对象 |
 | 自动关闭 | 项目设置 `close_in` 后，长期未更新的未完成工作项会被自动关闭 | v0 不做 |
 | 忘记密码 | 发邮件重置 | 服务器管理员用命令行重置 |
-| 密码规则 | 服务端在注册、修改密码、重置命令三处用 zxcvbn 评分 ≥ 3；组合规则只在界面上 | 服务端执行组合规则（长度 8–128，至少一个大写字母、小写字母、数字和特殊字符）、NCSC 前 10 万常见密码名单和"主干不能是邮箱前缀的主干"（M2 设计 3.8）。M2/P1 在注册时执行，M2/P3a 在修改密码时执行；创建账户和重置密码两个命令随 M2/P3b 加入 |
+| 管理员重置密码 | 只改密码（会话随之失效），PAT 不动 | `nerve users reset-password` 结束全部会话、撤销全部 PAT，输出撤销的数量（M2 设计 3.5、3.17） |
+| 重置密码命令的邮箱 | `reset_password` 按原样匹配 | 按注册时的规则规范化（去掉首尾空白、转小写） |
+| 密码规则 | 服务端在注册、修改密码、重置命令三处用 zxcvbn 评分 ≥ 3；组合规则只在界面上 | 服务端执行组合规则（长度 8–128，至少一个大写字母、小写字母、数字和特殊字符）、NCSC 前 10 万常见密码名单和"主干不能是邮箱前缀的主干"（M2 设计 3.8）。M2/P1 在注册时执行，M2/P3a 在修改密码时执行，M2/P3b 在创建账户和重置密码两个命令中执行 |
 | 注册的前提 | 实例必须先由实例管理员完成设置 | 没有这一步 |
-| 注册默认是否开放 | `ENABLE_SIGNUP` 默认开放 | prod 默认关闭，dev、test 默认开放；关闭时先答"注册已关闭"，不查邮箱；第一个账户用 `nerve users create`（M2/P3b 加入）（M2 设计决策点 2） |
+| 注册默认是否开放 | `ENABLE_SIGNUP` 默认开放 | prod 默认关闭，dev、test 默认开放；关闭时先答"注册已关闭"，不查邮箱；第一个账户用 `nerve users create`（M2 设计决策点 2） |
 | 请求中的未知字段和不合法的 `null` | DRF 的序列化器忽略未知字段 | 按契约返回 400（M2 设计 3.11） |
 | 密码哈希过载 | 无并发上限 | 最多 4 个同时计算，等待 2 秒仍拿不到名额时 503 `server_busy`，带 `Retry-After: 1`（M2 设计 3.8） |
 | 登录时邮箱不存在 | 返回 `USER_DOES_NOT_EXIST` | 与密码错误相同的 401 `identity.invalid_credentials`，耗时也相同：对一个启动时生成的假哈希做一次同样参数的校验（M2 设计 3.9）；这只在存储的哈希都用当前参数时成立，调高 argon2 参数之后，休眠的账户再次登录之前能被耗时区分（M2 设计 §16） |
@@ -166,7 +169,9 @@
 | 限流 | 认证接口合计每 IP 10/min，匿名 30/min，API Key 60/min；`/api/v1` 的响应带 `X-RateLimit-Remaining`、`X-RateLimit-Reset`（`plane/apps/api/plane/api/views/base.py:120-126`） | 进程内的令牌桶，每个桶有速率和突发：匿名按 IP、已认证按凭证、登录按 IP 和"IP + 邮箱"、注册按 IP，认证之前另有按 IP 的失败闸门；超出时 429 带 `Retry-After`，不加 `X-RateLimit-*`（M2 设计 3.10）。修改密码另有按账户的桶 `password_user` |
 | 无效的个人访问令牌 | 403（`AuthenticationFailed` 没有 `authenticate_header`） | 401 `unauthorized` |
 | 修改密码、停用之后的旧凭证 | 其他会话在下一个请求时失效 | 相同，由每个请求的会话检查做到（M2 设计 3.5） |
-| 停用账户 | 自助停用：撤销会话、重置新手引导、把密码改成随机值、发邮件；"唯一管理员"的检查从不拒绝；只有命令 `activate_user` 能恢复 | 自助停用 `POST /api/v0/me/deactivate`：撤销全部会话，重置新手引导，不改密码，PAT 不删除但停用期间认证失败；管理员的 `deactivate`、`activate` 命令随 M2/P3b 加入；"唯一管理员"的检查由 M3 在同一个事务里实现（M2 设计决策点 3） |
+| 停用账户 | 自助停用：撤销会话、重置新手引导、把密码改成随机值、发邮件；"唯一管理员"的检查从不拒绝；只有命令 `activate_user` 能恢复 | 自助停用 `POST /api/v0/me/deactivate`：撤销全部会话，重置新手引导，不改密码，PAT 不删除但停用期间认证失败；服务器管理员的 `nerve users deactivate` 与自助停用相同，`nerve users activate` 恢复账户，恢复后没有过期的 PAT 重新可用；"唯一管理员"的检查由 M3 在同一个事务里实现（M2 设计决策点 3） |
+| 修改登录邮箱 | 用户在个人设置中向新邮箱索取验证码后修改 | 只能由服务器管理员用 `nerve users set-email` 修改：新邮箱按注册时的规则规范化，结束该账户的全部会话，PAT 不撤销（M2 设计决策点 1、3.17） |
+| 创建账户的命令 | 没有（第一个账户通过实例设置页创建） | `nerve users create`：建账户和资料，不建会话，注册关闭时也能用（M2 设计决策点 2） |
 | 个人访问令牌的管理 | 只能用 Cookie 会话管理 | 任何凭证都能管理，包括 PAT 本身（v0-design 0.2 原则 2） |
 | 个人访问令牌的 `last_used` | 每个请求都写 | 每分钟最多写一次 |
 | 个人访问令牌的名称和过期时间 | 不校验：名称过长时变成 500，过期时间可以是过去 | 名称 1–255 个字符；过期时间必须在未来 |
```

- [ ] **Step 3: README**（部署：第一个账户；管理命令；令牌泄露后的恢复）

`README.md`（对 `d6f313b` 的差异）：

```diff
--- a/README.md
+++ b/README.md
@@ -105,7 +105,16 @@
 
   访问令牌由它签名，刷新令牌的 MAC 密钥也从它派生。换密钥（替换文件后重启）的后果：已签发的访问令牌验签失败，客户端续期一次即可；换钥之前的旧代刷新令牌不再能被认出是重复使用，所以刷新令牌正被盗用时换钥，受害者只是被登出（恢复靠修改密码或 `nerve users reset-password`）；同一出口 IP 后面的大量标签页同时续期，会短暂得到 429。所以在低峰时换。
 - **反向代理**：nerve 前面有反向代理（例如 Caddy）时，把代理的地址写进 `server.trusted_proxies`（CIDR 列表；环境变量用逗号分隔，例如 `NERVE_SERVER__TRUSTED_PROXIES=10.0.0.0/8,fd00::/8`）。只有连接的对端在这个列表中时，nerve 才从 `X-Forwarded-For` 自右向左取第一个不可信的地址作为客户端 IP。不配置时，所有请求都算作代理的地址：按 IP 的限流（匿名请求、登录、注册、认证失败）让所有人共用一份额度；nerve 第一次收到不可信对端带来的 `X-Forwarded-For` 时记一条 WARN 提醒。代理必须往 `X-Forwarded-For` 里写不带端口的 IP 地址：某一项带端口或是主机名时，nerve 在转发它的那个代理处停下，这个代理后面的客户端都算作代理的地址（第一次遇到时同样记一条 WARN）。只信任你自己的代理的地址：`0.0.0.0/0`、`::/0` 这样信任所有地址的前缀让任何客户端都能自己选 IP，启动时被拒绝。IPv6 客户端按前缀计数（`ratelimit.ipv6_prefix_len`，默认 64）。
-- **注册**：prod 默认关闭注册（`auth.signup_enabled: false`）。第一个账户用 `nerve users create` 创建（M2/P3b 加入；在那之前临时设 `NERVE_AUTH__SIGNUP_ENABLED=true`）。开放注册时，注册接口会暴露一个邮箱是否已经注册：没有邮件通道，就无法让两种回答相同。注册按客户端 IP 限流（默认每分钟 10 次），只能压低探测的速度。登录不暴露账户是否存在；存储的哈希都用当前的 argon2 参数时，耗时也相同。调高 `auth.password.argon2_memory_kib`、`argon2_iterations` 之后，还没有重新登录过的账户能从耗时上与不存在的邮箱区分开，直到它们各登录一次，所以这两个参数少调（M2 设计 §16）。关闭注册时，已注册和未注册的邮箱得到同一个 403。
+- **注册**：prod 默认关闭注册（`auth.signup_enabled: false`）。第一个账户用 `nerve users create --email <邮箱>` 创建（见下一条），注册关闭时也能用。开放注册时，注册接口会暴露一个邮箱是否已经注册：没有邮件通道，就无法让两种回答相同。注册按客户端 IP 限流（默认每分钟 10 次），只能压低探测的速度。登录不暴露账户是否存在；存储的哈希都用当前的 argon2 参数时，耗时也相同。调高 `auth.password.argon2_memory_kib`、`argon2_iterations` 之后，还没有重新登录过的账户能从耗时上与不存在的邮箱区分开，直到它们各登录一次，所以这两个参数少调（M2 设计 §16）。关闭注册时，已注册和未注册的邮箱得到同一个 403。
+- **管理命令**：`nerve users` 下的五个命令直接连数据库执行，用与 `nerve serve` 相同的配置（`NERVE_ENV`、`NERVE_DATABASE__URL` 等），服务不用停。都用 `--email` 指定账户，邮箱按注册时的规则规范化（去掉首尾空白、转小写）。账户不存在、邮箱已被使用、密码不合规时，退出码为 1，打印一行说明，数据库不变。需要密码的 `create`、`reset-password` 在终端上不回显地提示输入两次；标准输入不是终端时读一行，供脚本使用（例如 `printf '%s\n' "$PASSWORD" | nerve users create --email ada@example.com`）。
+  - `create --email <邮箱>`：建账户，不建会话。
+  - `reset-password --email <邮箱>`：设新密码，结束该账户的全部会话，撤销全部个人访问令牌（PAT）。
+  - `set-email --email <旧邮箱> --new-email <新邮箱>`：修改登录邮箱（邮箱只能这样改），结束全部会话，**不撤销 PAT**。
+  - `deactivate --email <邮箱>`：停用账户，与用户自己停用相同：结束全部会话，重置新手引导，不改密码；PAT 保留，但停用期间认证失败。
+  - `activate --email <邮箱>`：恢复账户，没有过期的 PAT **重新可用**，输出它们的个数。
+
+  `set-email` 和 `activate` 都不是账户被盗后的恢复手段：怀疑账户被盗时，另外执行 `reset-password`。
+- **令牌泄露后的恢复**：刷新令牌存在浏览器的 localStorage 里，页面上的 XSS 能读出它，换来访问令牌后创建一个永不过期的 PAT（创建 PAT 不要求输入密码）。这个 PAT 不受退出、修改密码和会话 30 天期限的影响，还能再创建 PAT，所以泄露的影响不以 30 天为限（M2 设计 8.5）。怀疑泄露时：查看账户的 PAT 列表（`GET /api/v0/me/api-tokens`），按创建时间和最后使用时间认出不认识的令牌，逐个撤销；或者由服务器管理员执行 `nerve users reset-password --email <邮箱>`，它结束该账户的全部会话、撤销全部 PAT。
 - **令牌的密钥扫描**：个人访问令牌以 `nrv_pat_` 开头，刷新令牌以 `nrv_rt_` 开头，但前缀不会让代码托管平台自动识别它们。要让平台发现提交里泄露的令牌，在它的密钥扫描中加自定义规则，例如 GitHub 仓库或组织设置的 Secret scanning → Custom patterns（需要平台提供这项功能）：个人访问令牌 `nrv_pat_[A-Za-z0-9_-]{43}`，刷新令牌 `nrv_rt_[A-Za-z0-9_-]{91}`。
 - **迁移**：prod 默认不在启动时迁移（`database.auto_migrate: false`），先执行 `nerve migrate up`，再 `nerve serve`。用单独的数据库角色执行迁移时，运行服务的角色除了读写业务表，还要能读 `goose_db_version`（`GRANT SELECT ON goose_db_version TO <服务的角色>`）：`/readyz` 靠它判断迁移是否已完成。
 
```

- [ ] **Step 4: 交接**

`docs/v0/M2-auth/handoffs/M0-P2-platform-notes.md`（对 `d6f313b` 的差异）：

```diff
--- a/docs/v0/M2-auth/handoffs/M0-P2-platform-notes.md
+++ b/docs/v0/M2-auth/handoffs/M0-P2-platform-notes.md
@@ -61,3 +61,11 @@
 仍未处理，状态保持 `open`：第 2 条的接口调用日志（M8，挂在限流之后）；第 5 条的 River、停机顺序、连接池关闭的时限和 River 的迁移（M2/P3）。
 
 来源：[M2/P2 spec](../specs/P2-sessions.md) 第 7 节。
+
+## 处理结果（M2/P3b）
+
+5. **River 与停机顺序**（完成）：`platform/jobs` 建服务用的 River 客户端（v0.47.0，默认队列最多 2 个 worker），`bootstrap` 的 `run` 先启动任务、再运行 HTTP；停机时 HTTP 优雅停机 → River 停止（`jobs.shutdown_timeout`，默认 10 秒，到期后取消仍在运行的任务，再等 1 秒）→ 迁移执行器 → 连接池。`pool.Close()` 放在一个 5 秒的协程里，到期记 WARN，不挂住退出。数据库暂时不可达时，任务在后台重试启动（间隔从 1 秒加倍到 30 秒），服务照常运行，`/readyz` 答 503。River 的表用锁定版本的 `river migrate-get`（v0.47.0，`--line main --all --exclude-version 1`）导出第 2–7 版，写成 goose 迁移 `00005_river_main_v2_to_v7.sql`，Up、Down 各包在一对 `StatementBegin`/`StatementEnd` 里；5 个迁移 up、down、再 up 的测试通过。
+
+仍未处理，状态保持 `open`：第 2 条的接口调用日志（M8，挂在限流之后）。
+
+来源：[M2/P3b spec](../specs/P3b-jobs-and-admin.md) 第 7 节。
```

`docs/v0/M2-auth/handoffs/M0-P6-e2e-notes.md`（对 `d6f313b` 的差异）：

```diff
--- a/docs/v0/M2-auth/handoffs/M0-P6-e2e-notes.md
+++ b/docs/v0/M2-auth/handoffs/M0-P6-e2e-notes.md
@@ -78,3 +78,13 @@
 仍未处理，状态保持 `open`：页面的登录状态（M2/P4）；S2 的断言（M2/P4）；River 停机与 fixture 的预算（M2/P3b）；fixture 写法的延伸（M4、M5、M8）。
 
 来源：[M2/P3a spec](../specs/P3a-account-api.md) 第 7 节。
+
+## 处理结果（M2/P3b）
+
+- **River 停机与 fixture 的预算**（完成）：River 运行时实测 nerve 从收到 SIGTERM 到退出的时间：就绪后立即停机约 2 毫秒，运行 3 秒后停机 4–7 毫秒（各 10 次），远在 fixture 的 `stopTimeoutMs`（30 秒）和 worker 的预算（`readyTimeoutMs + stopTimeoutMs + 10`，70 秒）之内。按配置算的最坏情况（HTTP 20 秒、任务 10 秒加 1 秒、连接池 5 秒，共 36 秒）超过 `stopTimeoutMs`：那时 fixture 在 30 秒时 SIGKILL，报出带日志路径的错误，不会挂住；停机要这么久本身就是缺陷。
+- **命令的标准输入**：`runNerve` 接受标准输入和额外的环境变量，写完就关闭标准输入，读密码的命令拿到文件结尾，不会一直等；`e2e/fixtures/users.ts` 的 `nerveUsers`、`nerveUsersFails` 在 worker 的库上运行 `nerve users`（A12、A13、A16、A17），日志开到 DEBUG，核对输出和日志里都没有密码。
+- **新等待的期限**：A14 用 `expect.poll` 等清理任务删掉过期的会话，以 15 秒为限（test 配置的间隔是 2 秒）。
+
+仍未处理，状态保持 `open`：页面的登录状态（M2/P4）；S2 的断言（M2/P4）；fixture 写法的延伸（M4、M5、M8）。
+
+来源：[M2/P3b spec](../specs/P3b-jobs-and-admin.md) 第 7 节。
```

`docs/v0/M2-auth/handoffs/M1-P3-trim-platform.md`（对 `d6f313b` 的差异）：

```diff
--- a/docs/v0/M2-auth/handoffs/M1-P3-trim-platform.md
+++ b/docs/v0/M2-auth/handoffs/M1-P3-trim-platform.md
@@ -71,3 +71,11 @@
 仍未处理，状态保持 `open`：Cookie 会话和 CSRF、认证错误就地显示、前端改读新的实例字段（M2/P4）；`set-email` 命令（M2/P3b）。
 
 来源：[M2/P3a spec](../specs/P3a-account-api.md) 第 7 节。
+
+## 处理结果（M2/P3b）
+
+- **修改登录邮箱**（完成）：`nerve users set-email --email <旧邮箱> --new-email <新邮箱>`（M2 设计决策点 1、3.17）。新邮箱按注册时的规则规范化和校验，改写 `users.email`，结束该账户的全部会话（`email_changed`），PAT 不撤销；新邮箱已被别的账户使用、或与旧邮箱相同时，退出码为 1，数据库不变。端到端 A16 覆盖。接口和界面都没有修改邮箱的入口，`updateMe` 的说明写明这个命令。
+
+仍未处理，状态保持 `open`：Cookie 会话和 CSRF、认证错误就地显示、前端改读新的实例字段（M2/P4）。
+
+来源：[M2/P3b spec](../specs/P3b-jobs-and-admin.md) 第 7 节。
```

- [ ] **Step 5: 检查**

Run: `make lint-web`
Expected: 通过（关键词守卫扫描改过的文档）。

Run: `grep -rn "M2/P3b 加入\|随 M2/P3b" docs/v0/v0-design.md docs/v0/plane-diff.md README.md`
Expected: 没有输出（"随 M2/P3b 加入"的地方都已改为已加入）。

Run: `make test`
Expected: 全部 `ok`（文档不影响测试；确认工作区干净）。

- [ ] **Step 6: 提交**

```bash
git add docs/v0/v0-design.md docs/v0/plane-diff.md README.md docs/v0/M2-auth/handoffs
```
```bash
git commit -m "docs(M2/P3b): sync the design documents, plane-diff and README; record the handoff results

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** M2 设计 3.20 中 P3b 的各行和 8.7 中 P3b 的三行都已同步；三份交接追加了"处理结果（M2/P3b）"。
