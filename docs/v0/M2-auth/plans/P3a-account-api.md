# M2/P3a 账户接口 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 账户的其余接口都可用，而且都能用 PAT 完成：PAT 的创建、分页列表、撤销和认证；修改资料、偏好和新手引导的步骤；修改密码；停用账户；实例配置的三个字段和时区列表。A7–A11 的接口版本和 S3 通过；3.5 交错测试中的第 4、5 条在真实数据库上通过。

**Architecture:** 平台加三样东西：`shared` 的页游标封套（`EncodeCursor`、`DecodeCursor`），`APIErrors.BadRequest` 在 400 里写出绑定失败的参数名，`apitest` 从接口描述推出参数用例和"每一种问题一起"的请求体用例。`identity` 按端口与适配器分层：`domain` 加上 PAT、令牌规格、页大小、令牌列表的游标、资料和偏好的补丁检查（含 Plane 的 `contains_url`）；`app` 加上共用的 `CredentialLock`（锁账户行、在锁下复核调用者的凭证）和八个用例（创建、列出、撤销令牌，修改资料、读写偏好，修改密码，停用），`Authenticate` 认出 `nrv_pat_` 令牌；`adapter/postgres` 加上 `api_tokens` 的迁移、查询和存储方法；`adapter/http` 挂八个新操作和 `password_user` 桶。`instance` 加上三个配置字段和 `listTimezones`，`cmd/nerve` 嵌入 `time/tzdata`。`bootstrap` 多一个桶，instance 模块改为按 `Deps` 构造。

**Tech Stack:** Go 1.27.1、pgx v5.11.0、sqlc v1.31.1（`CGO_ENABLED=0`）、oapi-codegen v2.8.0、oapi-codegen/runtime v1.7.0（新加）、golang-jwt/jwt/v5 v5.3.1、golang.org/x/crypto v0.57.0、golangci-lint 2.13.2、PostgreSQL 18.6；Node 24、pnpm 11.10.0、Playwright 1.63.0。

**Spec:** `docs/v0/M2-auth/specs/P3a-account-api.md`（上级：`docs/v0/M2-auth/M2-design.md`）

## Global Constraints

- **Go 版本**：`server/go.mod` 和 `server/tools/go.mod` 保持 `go 1.27` 和 `toolchain go1.27.1`。只有 Task 7 执行 `go get`（`go -C server get github.com/oapi-codegen/runtime@v1.7.0`，再 `go -C server mod tidy`），之后用 `grep -n "^go \|^toolchain" server/go.mod server/tools/go.mod` 核对（golangci-lint 2.13.2 由 go1.27.0 构建，`go` 行更高就拒绝运行）。本 plan 结束时，`server/go.mod`、`go.sum` 对 `605f367` 的差异只有 runtime v1.7.0 和它带来的间接依赖 `github.com/apapsch/go-jsonmerge/v2` v2.0.0（Task 7 的差异块）；`server/tools/go.mod`、`go.sum` 不变。
- **依赖**：只加 `github.com/oapi-codegen/runtime` v1.7.0（spec 2.2）；不加 npm 包。
- **每个 Task 提交前**：`make lint-go` 两段都输出 `0 issues.`，`make test` 全部通过。有生成物的 Task（5、7、8、9、10、11、12）先执行 `make gen`，核对生成物的 SHA-256（`shasum -a 256`）和行数，**提交之后**执行 `make gen-check`（它用 `git status` 判断生成物是否已提交，提交之前必然报差异）。改了 `schema.gen.ts` 或 `e2e/` 的 Task（7、9、10、11、12、14）另执行 `make lint-web` 和 `make e2e`；Task 12、14 另执行 `make knip`。
- **生成的文件不手写、不从本 plan 复制**：执行生成命令，提交它的输出。表中是生成物的 SHA-256 和行数；对不上时停下来，说明某个输入与本 plan 不一致。
- **容器**：`make test` 和 `make e2e` 用自己的 testcontainers。开发库 `nerve-dev-db-1` 可以用，但不要停止或重建它，不要执行 `make dev-db-down`、`make dev-db-reset`。不要碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`）。
- **git**：每次 Bash 调用只执行一个 git 命令，不用 `;`、`&&`、`|` 串联 git；不用 `git -C`、`stash`、`clean`、`reset --hard`。`cd` 不与别的命令组合。
- **安装**：除了 Docker、Go、Node 不做任何全局安装；不执行 `corepack enable`（pnpm 已在 PATH 上）。
- **规则**：不写 `init()`，不用全局可变状态，构造函数显式传入依赖；不建 `utils`、`common`、`helpers` 包；一个文件只做一件事，不超过约 400 行；依赖只能向内；平台包之间不互相导入（规则 7）；不留没有使用者的代码。
- **注释**：Go、TS 代码、SQL 查询和接口描述用英文；迁移文件、`server/configs/*.yaml` 的中文注释和中文文档照本 plan 原样。
- **代码块**：标为"新文件"或"完整内容"的块是**完整的文件内容**，照原样写入，不要改动（原型中逐字节运行过）；标为"差异"的块是对这个文件当前版本（`605f367` 或前一个 Task 写的版本）的统一差异，照差异修改，改完的文件与原型逐字节相同。差异块可以存成文件后在仓库根目录用 `patch -p1 < <文件>` 应用（块里的路径是 `a/<路径>`、`b/<路径>`；拼 plan 的脚本已用 `patch -p1` 逐个核对过，108 个差异块都得出原型中的文件）。
- **过渡版本**：少数文件先在较早的 Task 写成过渡版本，较晚的 Task 再写成最终版本；每个过渡版本都在原型的逐 Task 复现中运行过（spec 附录 A）。
- **提交**：提交信息用英文，末尾加一行：`Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明。

## 文件结构

路径相对于仓库根目录。"生成"表示由命令生成并提交；"过渡"表示这个 Task 写过渡版本，后面的 Task 写最终版本。

| 文件 | 职责 | Task |
|---|---|---|
| `server/internal/platform/config/config.go`、`validate.go` 及三个测试（修改） | `password_user` 桶；`workspace.creation_enabled`、`files.size_limit` | 1 |
| `server/configs/config.yaml`、`config.test.yaml`、`embed_test.go`（修改） | 默认值；test 的桶调高 | 1 |
| `server/internal/platform/httpserver/apierrors.go`、`apierrors_test.go`、`contract_test.go`（修改） | 参数绑定的 400 写出参数名 | 2 |
| `server/internal/platform/httpserver/apitest/operations.go`、`operations_test.go`（修改） | 请求体用例"每一种问题一起"；参数用例 | 2（过渡）、7 |
| `server/internal/platform/httpserver/apitest/rules_test.go`、`rules_cases_test.go`（修改） | 参数写 `schema`；JSON 请求体是对象 | 2 |
| `server/tools/bodyshapegen/generate.go`、`generate_test.go`（修改） | OpenAPI 3.1 的 `contentMediaType`、`contentEncoding` | 2 |
| `server/internal/archtest/deps_test.go`、`purity_test.go`（修改）、`generated_test.go` | 按导入的边判断禁用；生成代码的 uuid 规则 | 3 |
| `server/internal/archtest/binary_test.go`（修改） | google/uuid 只许 runtime 导入；嵌入时区数据库 | 3（过渡）、12 |
| `server/internal/shared/cursor.go`、`cursor_test.go`、`error.go`（修改） | 页游标的封套；`CodeBadRequest` | 4 |
| `server/internal/modules/identity/domain/api_token.go`、`api_token_test.go` | PAT、令牌规格、页大小、令牌列表的游标 | 4 |
| `server/migrations/sql/00004_identity_api_tokens.sql`、`schema_test.go`（修改）、`server/sqlc.yaml`（修改） | `api_tokens` 表 | 5 |
| `server/internal/modules/identity/adapter/postgres/queries/api_tokens.sql`、`api_tokens.go`、`api_tokens_test.go` | 令牌的查询和存储 | 5 |
| `server/internal/modules/identity/adapter/postgres/gen/api_tokens.sql.go`、`models.go`（生成） | | 5 |
| `server/internal/modules/identity/app/ports.go`（修改） | 令牌、账户、偏好、密码、停用的端口 | 5（过渡）、8（过渡）、10（过渡）、11 |
| `server/internal/shared/actor.go`（修改） | `Actor.APITokenID` | 6 |
| `server/internal/modules/identity/app/authenticate.go`、`authenticate_test.go`（修改）、`authenticate_pat_test.go` | PAT 认证和 `last_used` | 6 |
| `server/internal/modules/identity/app/credential_lock.go` | 账户行锁下复核调用者的凭证 | 6 |
| `server/internal/modules/identity/app/{create,list,revoke}_api_token.go` 及三个测试 | 令牌的三个用例 | 6 |
| `server/internal/modules/identity/app/register_test.go`（修改）、`fakes_test.go`（修改） | 新的 `Authenticate` 构造；假仓储 | 6、`fakes_test.go` 6（过渡）、9（过渡）、10（过渡）、11 |
| `server/internal/modules/identity/adapter/authn/authenticator.go`、`authenticator_test.go`（修改） | PAT 的限流键 `pat:<id>` | 6 |
| `server/internal/modules/identity/domain/errors.go`（修改） | `api_token_not_found`；`current_password_incorrect` | 6（过渡）、10 |
| `server/internal/modules/identity/module.go`（修改） | 接线 | 6（过渡）、7（过渡）、9（过渡）、10（过渡）、11 |
| `api/common.yaml`（修改） | `Limit`、`Cursor`、`NextCursor` | 7 |
| `api/openapi.yaml`、`api/modules/identity.yaml`（修改） | identity 的八个新操作；`openapi.yaml` 另加 `listTimezones` | 7（过渡）、9（过渡）、10（过渡）、11；`openapi.yaml` 12 |
| `server/go.mod`、`server/go.sum`（修改） | `oapi-codegen/runtime` v1.7.0 | 7 |
| `server/internal/modules/identity/adapter/http/api_tokens.go`、`api_tokens_test.go` | 令牌的三个操作 | 7 |
| `server/internal/modules/identity/adapter/http/handler.go`、`handler_test.go`（修改） | 用例的小接口；测试服务 | 7（过渡）、9（过渡）、10（过渡）、11 |
| `server/internal/bootstrap/auth_test.go`、`contract_test.go`（修改） | PAT 的整程序认证；第五个整程序测试 | 7 |
| `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`、`*/adapter/http/gen/*.gen.go`、`apigen/components.gen.go`（生成） | | 7、9、10、11、12 |
| `server/internal/modules/identity/domain/user.go`、`profile.go`、`url.go` 及三个测试 | 资料、偏好的补丁检查；`contains_url` | 8 |
| `server/internal/modules/identity/adapter/postgres/queries/users.sql`、`profiles.sql`、`users.go`、`profiles.go`（修改）、`account_test.go` | 资料、偏好、密码、停用的查询和存储 | 8（过渡）、10（过渡：`users`）、11 |
| `server/internal/modules/identity/adapter/postgres/gen/users.sql.go`、`profiles.sql.go`、`sessions.sql.go`（生成） | | 8、10、11 |
| `server/internal/modules/identity/app/update_me.go`、`get_profile.go`、`update_profile.go`、`account_test.go` | 资料和偏好的三个用例 | 9 |
| `server/internal/modules/identity/adapter/http/me.go`、`me_test.go`（修改）、`profile.go`、`profile_test.go` | 资料、偏好、修改密码、停用的操作 | 9（过渡）、10（过渡）、11；`profile*` 9 |
| `server/internal/bootstrap/account_test.go` | PAT 的整程序测试 | 9（过渡）、10（过渡）、11 |
| `server/internal/modules/identity/app/change_password.go`、`change_password_test.go` | 修改密码 | 10 |
| `server/internal/modules/identity/adapter/postgres/queries/sessions.sql`、`sessions.go`、`credentials_test.go`（修改） | 按账户撤销会话；修改密码的账户读取 | 10 |
| `server/internal/modules/identity/domain/session.go`（修改） | 撤销的原因 | 10（过渡）、11 |
| `server/internal/modules/identity/adapter/http/limits.go`、`limits_test.go`（修改） | `password_user` 桶 | 10 |
| `server/internal/bootstrap/app.go`、`app_test.go`（修改） | `password_user` 桶的接线；instance 模块的 `Deps` | 10（过渡）、12 |
| `server/internal/bootstrap/limits_test.go` | 每个配置的桶都限住它的操作 | 10 |
| `server/internal/modules/identity/app/deactivate.go`、`deactivate_test.go` | 停用 | 11 |
| `api/modules/instance.yaml`（修改） | 实例配置的三个字段；`listTimezones` | 12 |
| `server/internal/modules/instance/`（`domain/info.go`、`timezones.go`，`app/ports.go`、`get_info.go`、`list_timezones.go`，`adapter/http/handler.go`，`module.go` 及测试） | 实例配置；时区列表 | 12 |
| `server/cmd/nerve/main.go`（修改） | 嵌入 `time/tzdata` | 12 |
| `server/internal/bootstrap/api_test.go`、`server/internal/platform/httpserver/apitest/apitest_test.go`、`e2e/stories/smoke/s3-instance-info.spec.ts`（修改） | 新字段；时区的整程序测试 | 12 |
| `server/internal/modules/identity/interleavings_test.go` | 3.5 的交错测试 4、5 | 13 |
| `e2e/fixtures/auth.ts`、`e2e/fixtures/assert/identity.ts`（修改） | `createPAT`、`bearer`；账户和令牌的断言 | 14 |
| `e2e/stories/identity/a7-change-password.spec.ts`、`a8-update-me.spec.ts`、`a9-preferences.spec.ts`、`a10-onboarding-profile.spec.ts`、`a11-api-tokens.spec.ts` | 五个故事的接口版本 | 14 |
| `docs/v0/v0-design.md`、`docs/v0/M0-foundation/M0-design.md`、`docs/v0/M0-foundation/specs/P3-api-contract.md`、`docs/v0/plane-diff.md`、`README.md`、`docs/v0/M2-auth/handoffs/{M0-P3,M0-P6,M1-P2,M1-P3}-*.md`（修改） | 文档同步、交接 | 15 |

---

### Task 1: 配置：`password_user` 桶、工作区和上传的实例配置

**Files:**
- Modify: `server/internal/platform/config/config.go`、`validate.go`、`config_test.go`、`load_test.go`、`validate_test.go`
- Modify: `server/configs/config.yaml`、`config.test.yaml`、`embed_test.go`

**Interfaces:**
- Produces（spec 2.3）：
  - `config.RateLimitConfig.PasswordUser BucketConfig`（`ratelimit.password_user`，默认 `{per_minute: 5, burst: 5}`）；
  - `config.WorkspaceConfig{CreationEnabled bool}`（`workspace.creation_enabled`，默认 `true`）、`config.FilesConfig{SizeLimit int64}`（`files.size_limit`，字节，默认 5242880），`Config.Workspace`、`Config.Files`；
  - 校验：`password_user` 两项至少 1；`files.size_limit` 至少 1；
  - `Config.LogValue` 加上 `ratelimit.password_user`、`workspace.creation_enabled`、`files.size_limit`。
- 使用者：`password_user` 在 Task 10 接到修改密码；两个实例字段在 Task 12 由实例配置接口报告。

**Tests:**
- `load_test.go`：`TestLoadAppliesLayersInOrder` 的 YAML 写上三组新键，三个环境变量（`NERVE_RATELIMIT__PASSWORD_USER__PER_MINUTE`、`NERVE_WORKSPACE__CREATION_ENABLED`、`NERVE_FILES__SIZE_LIMIT`）覆盖它们。
- `validate_test.go`：`TestValidateReportsEveryInvalidKey` 加上 `password_user` 的两项和 `files.size_limit`；`TestValidateCrossKeyRules` 加上负的上传上限。
- `config_test.go`：`TestLogValueMasksDatabaseURL` 核对四个新的日志项。`validConfig` 中三组新值（`password_user` 7、3，`creation_enabled: false`，7340032）都不同于默认值，也不同于别的键（`auth.signup_enabled` 是 `true`），日志从错的字段取值时测试失败（原型中第一次用了默认值，这样的变异没被发现，spec 附录 A）。
- `server/configs/embed_test.go`：`TestBuiltInProfiles` 核对三个环境的 `password_user`（dev、prod 是默认值，test 调高）和两个实例字段。

- [ ] **Step 1: 配置类型和校验**

`server/internal/platform/config/config.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/platform/config/config.go
+++ b/server/internal/platform/config/config.go
@@ -24,6 +24,8 @@
 	Database  DatabaseConfig  `koanf:"database"`
 	Auth      AuthConfig      `koanf:"auth"`
 	RateLimit RateLimitConfig `koanf:"ratelimit"`
+	Workspace WorkspaceConfig `koanf:"workspace"`
+	Files     FilesConfig     `koanf:"files"`
 	Log       LogConfig       `koanf:"log"`
 }
 
@@ -104,6 +106,9 @@
 	LoginIP      BucketConfig `koanf:"login_ip"`
 	LoginIPEmail BucketConfig `koanf:"login_ip_email"`
 	RegisterIP   BucketConfig `koanf:"register_ip"`
+	// PasswordUser limits the authenticated operations that verify a
+	// password, by account: in M2, changing the password.
+	PasswordUser BucketConfig `koanf:"password_user"`
 }
 
 // BucketConfig is a token bucket: it holds at most Burst units and gains
@@ -118,6 +123,18 @@
 	return slog.GroupValue(slog.Int("per_minute", b.PerMinute), slog.Int("burst", b.Burst))
 }
 
+// WorkspaceConfig configures workspaces. The instance API reports it to
+// clients; creating workspaces arrives, and honours it, in M3 (M2 design 5.3).
+type WorkspaceConfig struct {
+	CreationEnabled bool `koanf:"creation_enabled"`
+}
+
+// FilesConfig configures uploads. The instance API reports it to clients;
+// uploads arrive, and honour it, in M5 (M2 design 5.3).
+type FilesConfig struct {
+	SizeLimit int64 `koanf:"size_limit"` // bytes
+}
+
 // LogConfig configures the process logger.
 type LogConfig struct {
 	Level  string `koanf:"level"`  // debug, info, warn or error
@@ -170,7 +187,14 @@
 			slog.Any("login_ip", c.RateLimit.LoginIP),
 			slog.Any("login_ip_email", c.RateLimit.LoginIPEmail),
 			slog.Any("register_ip", c.RateLimit.RegisterIP),
+			slog.Any("password_user", c.RateLimit.PasswordUser),
 		),
+		slog.Group("workspace",
+			slog.Bool("creation_enabled", c.Workspace.CreationEnabled),
+		),
+		slog.Group("files",
+			slog.Int64("size_limit", c.Files.SizeLimit),
+		),
 		slog.Group("log",
 			slog.String("level", c.Log.Level),
 			slog.String("format", c.Log.Format),
```

`server/internal/platform/config/validate.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/platform/config/validate.go
+++ b/server/internal/platform/config/validate.go
@@ -81,6 +81,9 @@
 			c.Database.CommitTimeout, webRefreshTimeout, c.Auth.RefreshDeadline)
 	}
 	c.RateLimit.validate(fail)
+	if c.Files.SizeLimit < 1 {
+		fail("files.size_limit", "must be at least 1, got %d", c.Files.SizeLimit)
+	}
 	var level slog.Level
 	if err := level.UnmarshalText([]byte(c.Log.Level)); err != nil {
 		fail("log.level", "must be one of debug, info, warn, error, got %q", c.Log.Level)
@@ -139,6 +142,7 @@
 		{"login_ip", r.LoginIP},
 		{"login_ip_email", r.LoginIPEmail},
 		{"register_ip", r.RegisterIP},
+		{"password_user", r.PasswordUser},
 	} {
 		if b.bucket.PerMinute < 1 {
 			fail("ratelimit."+b.name+".per_minute", "must be at least 1, got %d", b.bucket.PerMinute)
```

- [ ] **Step 2: 测试**

`server/internal/platform/config/config_test.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/platform/config/config_test.go
+++ b/server/internal/platform/config/config_test.go
@@ -48,6 +48,10 @@
 		"config.ratelimit.login_ip.per_minute=30",
 		"config.ratelimit.login_ip_email.burst=5",
 		"config.ratelimit.register_ip.per_minute=10",
+		"config.ratelimit.password_user.per_minute=7",
+		"config.ratelimit.password_user.burst=3",
+		"config.workspace.creation_enabled=false",
+		"config.files.size_limit=7340032",
 		"config.log.format=json",
 	} {
 		if !strings.Contains(out, want) {
```

`server/internal/platform/config/load_test.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/platform/config/load_test.go
+++ b/server/internal/platform/config/load_test.go
@@ -48,6 +48,11 @@
   login_ip: {per_minute: 30, burst: 10}
   login_ip_email: {per_minute: 10, burst: 5}
   register_ip: {per_minute: 10, burst: 5}
+  password_user: {per_minute: 5, burst: 5}
+workspace:
+  creation_enabled: true
+files:
+  size_limit: 5242880
 log:
   level: info
   format: json
@@ -86,6 +91,9 @@
 			"NERVE_AUTH__PASSWORD__ARGON2_MEMORY_KIB=64",
 			"NERVE_SERVER__TRUSTED_PROXIES=10.0.0.0/8,fd00::/8",
 			"NERVE_RATELIMIT__LOGIN_IP__BURST=3",
+			"NERVE_RATELIMIT__PASSWORD_USER__PER_MINUTE=7",
+			"NERVE_WORKSPACE__CREATION_ENABLED=false",
+			"NERVE_FILES__SIZE_LIMIT=1024",
 		},
 		LocalFile: local,
 	})
@@ -132,8 +140,11 @@
 			LoginIP:       BucketConfig{PerMinute: 30, Burst: 3}, // environment
 			LoginIPEmail:  BucketConfig{PerMinute: 10, Burst: 5},
 			RegisterIP:    BucketConfig{PerMinute: 10, Burst: 5},
+			PasswordUser:  BucketConfig{PerMinute: 7, Burst: 5}, // environment
 		},
-		Log: LogConfig{Level: "debug", Format: "text"},
+		Workspace: WorkspaceConfig{CreationEnabled: false}, // environment
+		Files:     FilesConfig{SizeLimit: 1024},            // environment
+		Log:       LogConfig{Level: "debug", Format: "text"},
 	}
 	if !reflect.DeepEqual(cfg, want) {
 		t.Errorf("Load() =\n%+v\nwant\n%+v", cfg, want)
```

`server/internal/platform/config/validate_test.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/platform/config/validate_test.go
+++ b/server/internal/platform/config/validate_test.go
@@ -42,8 +42,13 @@
 			LoginIP:       BucketConfig{PerMinute: 30, Burst: 10},
 			LoginIPEmail:  BucketConfig{PerMinute: 10, Burst: 5},
 			RegisterIP:    BucketConfig{PerMinute: 10, Burst: 5},
+			PasswordUser:  BucketConfig{PerMinute: 7, Burst: 3},
 		},
-		Log: LogConfig{Level: "info", Format: "json"},
+		// Unlike the defaults and unlike auth.signup_enabled, so that a key
+		// logged from the wrong field shows.
+		Workspace: WorkspaceConfig{CreationEnabled: false},
+		Files:     FilesConfig{SizeLimit: 7340032},
+		Log:       LogConfig{Level: "info", Format: "json"},
 	}
 }
 
@@ -97,6 +102,9 @@
 		"ratelimit.login_ip_email.burst: must be at least 1, got 0",
 		"ratelimit.register_ip.per_minute: must be at least 1, got 0",
 		"ratelimit.register_ip.burst: must be at least 1, got 0",
+		"ratelimit.password_user.per_minute: must be at least 1, got 0",
+		"ratelimit.password_user.burst: must be at least 1, got 0",
+		"files.size_limit: must be at least 1, got 0",
 		`log.level: must be one of debug, info, warn, error, got "verbose"`,
 		`log.format: must be text or json, got "xml"`,
 	}
@@ -172,6 +180,11 @@
 			want: "server.trusted_proxies: ::/0 trusts every address, so any client could choose its own IP; list only your proxies' addresses",
 		},
 		{
+			name:   "negative file size limit",
+			change: func(c *Config) { c.Files.SizeLimit = -1 },
+			want:   "files.size_limit: must be at least 1, got -1",
+		},
+		{
 			name:   "IPv6 prefix longer than an address",
 			change: func(c *Config) { c.RateLimit.IPv6PrefixLen = 129 },
 			want:   "ratelimit.ipv6_prefix_len: must be from 1 to 128, got 129",
```

- [ ] **Step 3: 内嵌的配置文件**

`server/configs/config.yaml`（对 `605f367` 的差异）：

```diff
--- a/server/configs/config.yaml
+++ b/server/configs/config.yaml
@@ -68,7 +68,17 @@
   login_ip_email: {per_minute: 10, burst: 5}
   # 注册：按客户端 IP
   register_ip: {per_minute: 10, burst: 5}
+  # 需要校验密码的已认证操作（M2 中是修改密码），按账户
+  password_user: {per_minute: 5, burst: 5}
 
+workspace:
+  # 是否允许创建工作区。实例配置接口报告它；创建工作区在 M3 加入时照它执行
+  creation_enabled: true
+
+files:
+  # 单个上传文件的大小上限（字节，默认 5 MiB）。实例配置接口报告它；上传在 M5 加入时照它执行
+  size_limit: 5242880
+
 log:
   level: info    # debug | info | warn | error
   format: json   # text | json
```

`server/configs/config.test.yaml`（对 `605f367` 的差异）：

```diff
--- a/server/configs/config.test.yaml
+++ b/server/configs/config.test.yaml
@@ -15,6 +15,7 @@
   login_ip: {per_minute: 600000, burst: 100000}
   login_ip_email: {per_minute: 600000, burst: 100000}
   register_ip: {per_minute: 600000, burst: 100000}
+  password_user: {per_minute: 600000, burst: 100000}
 
 log:
   level: warn
```

`server/configs/embed_test.go`（对 `605f367` 的差异）：

```diff
--- a/server/configs/embed_test.go
+++ b/server/configs/embed_test.go
@@ -22,10 +22,12 @@
 		LoginIP:       config.BucketConfig{PerMinute: 30, Burst: 10},
 		LoginIPEmail:  config.BucketConfig{PerMinute: 10, Burst: 5},
 		RegisterIP:    config.BucketConfig{PerMinute: 10, Burst: 5},
+		PasswordUser:  config.BucketConfig{PerMinute: 5, Burst: 5},
 	}
 	high       = config.BucketConfig{PerMinute: 600000, Burst: 100000}
 	testLimits = config.RateLimitConfig{
 		IPv6PrefixLen: 64, Anonymous: high, AuthFailure: high, Authenticated: high, LoginIP: high, LoginIPEmail: high, RegisterIP: high,
+		PasswordUser: high,
 	}
 )
 
@@ -89,6 +91,8 @@
 					},
 				},
 				RateLimit: tt.limits,
+				Workspace: config.WorkspaceConfig{CreationEnabled: true},
+				Files:     config.FilesConfig{SizeLimit: 5 << 20},
 				Log:       config.LogConfig{Level: tt.level, Format: tt.format},
 			}
 			if !reflect.DeepEqual(cfg, want) {
```

- [ ] **Step 4: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。`cmd/nerve` 的测试用内嵌的 test 配置启动 `nerve serve`，证明新键都有默认值、校验通过。

- [ ] **Step 5: 提交**

```bash
git add server/internal/platform/config server/configs
```
```bash
git commit -m "feat(M2/P3a): configuration for the password_user bucket, workspace creation and the upload size limit

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** `TestValidateReportsEveryInvalidKey` 报出 `password_user` 的两项和 `files.size_limit`；`TestBuiltInProfiles` 核对三个环境的新键；`make test` 通过。

---

### Task 2: 平台：参数绑定的 400 写出参数名；请求体用例合并每一种问题；两条接口描述规则；3.1 的字符串写法

**Files:**
- Modify: `server/internal/platform/httpserver/apierrors.go`、`apierrors_test.go`、`contract_test.go`
- Modify: `server/internal/platform/httpserver/apitest/operations.go`、`operations_test.go`（过渡：Task 7 加上参数用例）
- Modify: `server/internal/platform/httpserver/apitest/rules_test.go`、`rules_cases_test.go`
- Modify: `server/tools/bodyshapegen/generate.go`、`generate_test.go`

**Interfaces:**
- Produces（spec 2.4）：
  - `APIErrors.BadRequest(w, r, err)`：400 `bad_request`，`detail` 固定为 "The request parameters do not match the API description."，`errors` 里是参数：`parameterOf(err)` 用反射读绑定错误的字符串字段 `ParamName`，类型名以 `Required` 开头时 `code` 为 `required`，否则 `invalid_format`；读不出参数时没有 `errors`。绑定错误的原文只进 DEBUG "request parameters not bound"（带 `request_id`）。
  - `Operation.BodyCases()` 的第 7 种"every problem at once"：schema 有的每一种问题（未声明的属性、嵌套对象里未声明的属性、非空可选属性为 `null`、缺少必填属性、格式错误）各取一个别的种类没用过的属性，合在一个请求体里；只有未声明属性一种时不生成（spec 第 3 节第 1 条）。
  - `apitest` 的规则：参数写 `schema`，不写 `content`；`application/json` 的请求体 schema 是对象。
  - `bodyshapegen`：没有 `format` 时，带 `contentMediaType` 的字符串按 `binary`、`contentEncoding: base64` 的按 `byte` 查格式（oapi-codegen v2.8.0 的做法），生成为 bodyshape 没有检查器的类型时报错。
- 参数用例（`ParamCases`）和第五个整程序测试在 Task 7 随第一批带参数的操作加入。

**Tests:**
- `apierrors_test.go`：`TestAPIErrorsBadRequestNamesTheParameter`（7 个：格式不对、缺少查询参数、缺少响应头参数、值太多、不是绑定错误、`ParamName` 不是字符串、不是结构），替代原来的 `TestAPIErrorsBadRequest`；每个都核对完整的响应体和 DEBUG 日志。测试里的四个错误类型是 identity `server.gen.go` 中绑定错误的仿制品。
- `contract_test.go`：`TestPlatformProblemsMatchTheContract` 的 400 用 `InvalidParamFormatError`。
- 两个文件里读响应头的测试都改读 `rec.Result().Header`（状态写出时的快照），不读记录器上活的头部映射：写出之后才设的响应头到不了客户端，活的映射却照样能读到（P2 缺陷类别"读活的头部映射"，spec 附录 A）。
- `operations_test.go`：`TestBodyCases` 的最后一个用例带上五种问题；`TestBodyCasesCombineEveryKindTheSchemaHas`（4 个：没有必填属性、只有嵌套对象、必填属性是唯一带格式的属性、只有可为 `null` 的属性）。
- `rules_cases_test.go`：`TestAuthoringRulesReportViolations` 加上两个违规例子（参数写 `content`、数组请求体）。
- `generate_test.go`：`TestGenerateRejectsWhatItCannotCheck` 加上 `contentMediaType` 和 `contentEncoding: base64`；`TestGenerateAcceptsTheStringsItCanCheck`（`base64url`、带 `format` 时不看 `contentEncoding` 和 `contentMediaType`）。

- [ ] **Step 1: 参数绑定的 400**

`server/internal/platform/httpserver/apierrors.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/platform/httpserver/apierrors.go
+++ b/server/internal/platform/httpserver/apierrors.go
@@ -7,7 +7,9 @@
 	"log/slog"
 	"math"
 	"net/http"
+	"reflect"
 	"strconv"
+	"strings"
 	"time"
 )
 
@@ -63,18 +65,57 @@
 	return APIErrors{logger: logger}
 }
 
-// BadRequest answers 400 bad_request for a request whose parameters the
-// generated code could not bind; err says what was wrong and becomes the
-// detail. (M2/P3 derives the field from binding errors instead.)
-func (APIErrors) BadRequest(w http.ResponseWriter, _ *http.Request, err error) {
-	WriteProblem(w, Problem{
+// BadRequest answers a request whose path, query or header parameters the
+// generated code could not bind: 400 bad_request, with the parameter in
+// errors (M2 design 3.11). The binding errors' messages name Go functions and
+// types, so the detail is generic and err is logged at debug level only.
+func (e APIErrors) BadRequest(w http.ResponseWriter, r *http.Request, err error) {
+	e.logger.LogAttrs(r.Context(), slog.LevelDebug, "request parameters not bound",
+		slog.String("request_id", RequestID(r.Context())), slog.Any("error", err))
+	p := Problem{
 		Status: http.StatusBadRequest,
 		Code:   CodeBadRequest,
 		Title:  http.StatusText(http.StatusBadRequest),
-		Detail: err.Error(),
-	})
+		Detail: "The request parameters do not match the API description.",
+	}
+	if f, ok := parameterOf(err); ok {
+		p.Errors = []FieldError{f}
+	}
+	WriteProblem(w, p)
 }
 
+// The field codes of a parameter that did not bind (api/common.yaml).
+const (
+	fieldRequired      = "required"
+	fieldInvalidFormat = "invalid_format"
+)
+
+// parameterOf reads the parameter that a binding error names. oapi-codegen
+// declares the binding errors (InvalidParamFormatError, RequiredParamError
+// and four more) in each module's gen package, which the platform does not
+// import, and gives them no method that names the parameter. Each is a
+// pointer to a struct with the name in the string field ParamName, so the
+// name is read by reflection; a type named Required... reports a missing
+// parameter. The whole-program test of bootstrap binds every operation's
+// parameters with a wrong value through the generated code.
+func parameterOf(err error) (FieldError, bool) {
+	v := reflect.ValueOf(err)
+	if v.Kind() == reflect.Pointer {
+		v = v.Elem()
+	}
+	if v.Kind() != reflect.Struct {
+		return FieldError{}, false
+	}
+	name := v.FieldByName("ParamName")
+	if name.Kind() != reflect.String {
+		return FieldError{}, false
+	}
+	if strings.HasPrefix(v.Type().Name(), "Required") {
+		return FieldError{Field: name.String(), Code: fieldRequired, Message: "is required"}, true
+	}
+	return FieldError{Field: name.String(), Code: fieldInvalidFormat, Message: "has the wrong type or format"}, true
+}
+
 // BodyError answers a request body that could not be read, parsed or
 // decoded, from the bodyshape middleware or the generated strict handler. A
 // ProblemError (bodyshape's structural 400 with its fields) and
```

`server/internal/platform/httpserver/apierrors_test.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/platform/httpserver/apierrors_test.go
+++ b/server/internal/platform/httpserver/apierrors_test.go
@@ -42,20 +42,82 @@
 func (minimalErr) ProblemStatus() int  { return http.StatusNotFound }
 func (minimalErr) ProblemCode() string { return "things.not_found" }
 
-func TestAPIErrorsBadRequest(t *testing.T) {
-	errs := NewAPIErrors(slog.New(slog.DiscardHandler))
-	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
-		errs.BadRequest(w, r, errors.New("Invalid format for parameter limit: not a number"))
-	})
+// Lookalikes of the binding errors that oapi-codegen declares in each
+// module's gen package (identity's server.gen.go): the platform sees only
+// their shape.
+type (
+	InvalidParamFormatError struct {
+		ParamName string
+		Err       error
+	}
+	RequiredParamError  struct{ ParamName string }
+	RequiredHeaderError struct {
+		ParamName string
+		Err       error
+	}
+	TooManyValuesForParamError struct {
+		ParamName string
+		Count     int
+	}
+	numberedParamError struct{ ParamName int }
+	codeError          string
+)
 
-	rec := serve(h, httptest.NewRequest(http.MethodGet, "/api/v0/things?limit=x", nil))
+func (e *InvalidParamFormatError) Error() string {
+	return "Invalid format for parameter " + e.ParamName + ": " + e.Err.Error()
+}
+func (e *RequiredParamError) Error() string {
+	return "Query argument " + e.ParamName + " is required, but not found"
+}
+func (e *RequiredHeaderError) Error() string {
+	return "Header parameter " + e.ParamName + " is required, but not found"
+}
+func (e *TooManyValuesForParamError) Error() string {
+	return fmt.Sprintf("Expected one value for %s, got %d", e.ParamName, e.Count)
+}
+func (e *numberedParamError) Error() string { return fmt.Sprint("parameter number ", e.ParamName) }
+func (e codeError) Error() string           { return string(e) }
 
-	if rec.Code != http.StatusBadRequest || rec.Header().Get("Content-Type") != ContentTypeProblem {
-		t.Errorf("response = %d %s, want 400 problem+json", rec.Code, rec.Header().Get("Content-Type"))
+// The parameter comes from the binding error; its message names Go
+// functions, so the detail is generic and the message goes to the debug log.
+func TestAPIErrorsBadRequestNamesTheParameter(t *testing.T) {
+	const detail = `"detail":"The request parameters do not match the API description."`
+	invalid := &InvalidParamFormatError{ParamName: "limit", Err: errors.New(`error binding string parameter: strconv.ParseInt: parsing "abc": invalid syntax`)}
+	tests := []struct {
+		name string
+		err  error
+		want string
+	}{
+		{"invalid format", invalid, `,"errors":[{"field":"limit","code":"invalid_format","message":"has the wrong type or format"}]`},
+		{"required", &RequiredParamError{ParamName: "view"}, `,"errors":[{"field":"view","code":"required","message":"is required"}]`},
+		{"required header", &RequiredHeaderError{ParamName: "X-Thing", Err: errors.New("missing")},
+			`,"errors":[{"field":"X-Thing","code":"required","message":"is required"}]`},
+		{"too many values", &TooManyValuesForParamError{ParamName: "X-Thing", Count: 2},
+			`,"errors":[{"field":"X-Thing","code":"invalid_format","message":"has the wrong type or format"}]`},
+		{"not a binding error", errors.New("Invalid format for parameter limit"), ""},
+		{"ParamName not a string", &numberedParamError{ParamName: 7}, ""},
+		{"not a struct", codeError("limit"), ""},
 	}
-	want := `{"status":400,"code":"bad_request","title":"Bad Request","detail":"Invalid format for parameter limit: not a number"}` + "\n"
-	if rec.Body.String() != want {
-		t.Errorf("body = %s, want %s", rec.Body, want)
+	for _, tt := range tests {
+		t.Run(tt.name, func(t *testing.T) {
+			logger, logs := captureLogs(t)
+			errs := NewAPIErrors(logger)
+			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { errs.BadRequest(w, r, tt.err) })
+			req := httptest.NewRequest(http.MethodGet, "/api/v0/things?limit=abc", nil)
+			req.Header.Set(HeaderRequestID, "req-9")
+
+			rec := serve(middleware(h, logger), req)
+
+			sent := rec.Result().Header // as the status went out, not as set after it
+			want := `{"status":400,"code":"bad_request","title":"Bad Request",` + detail + tt.want + "}\n"
+			if rec.Code != http.StatusBadRequest || sent.Get("Content-Type") != ContentTypeProblem || rec.Body.String() != want {
+				t.Errorf("response = %d %s %s, want 400 problem+json %s", rec.Code, sent.Get("Content-Type"), rec.Body, want)
+			}
+			entry := findLog(logs(), "request parameters not bound")
+			if entry == nil || entry["level"] != "DEBUG" || entry["error"] != tt.err.Error() || entry["request_id"] != "req-9" {
+				t.Errorf("log = %v, want the binding error with request_id at debug level", entry)
+			}
+		})
 	}
 }
 
@@ -132,10 +194,11 @@
 
 			rec := serve(h, httptest.NewRequest(http.MethodPost, "/api/v0/things", nil))
 
-			if rec.Code != tt.status || rec.Header().Get("Content-Type") != ContentTypeProblem || rec.Body.String() != tt.body+"\n" {
-				t.Errorf("response = %d %s %s, want %d problem+json %s", rec.Code, rec.Header().Get("Content-Type"), rec.Body, tt.status, tt.body)
+			sent := rec.Result().Header
+			if rec.Code != tt.status || sent.Get("Content-Type") != ContentTypeProblem || rec.Body.String() != tt.body+"\n" {
+				t.Errorf("response = %d %s %s, want %d problem+json %s", rec.Code, sent.Get("Content-Type"), rec.Body, tt.status, tt.body)
 			}
-			if got := rec.Header().Get("Retry-After"); got != tt.retryAfter {
+			if got := sent.Get("Retry-After"); got != tt.retryAfter {
 				t.Errorf("Retry-After = %q, want %q", got, tt.retryAfter)
 			}
 		})
@@ -170,7 +233,7 @@
 
 			rec := serve(h, httptest.NewRequest(http.MethodPost, "/api/v0/auth/login", nil))
 
-			if got := rec.Header().Values("WWW-Authenticate"); rec.Code != tt.status || !slices.Equal(got, tt.want) {
+			if got := rec.Result().Header.Values("WWW-Authenticate"); rec.Code != tt.status || !slices.Equal(got, tt.want) {
 				t.Errorf("response = %d WWW-Authenticate %q, want %d %q", rec.Code, got, tt.status, tt.want)
 			}
 		})
@@ -190,8 +253,8 @@
 
 	rec := serve(h, req)
 
-	if rec.Code != http.StatusInternalServerError || rec.Header().Get("Content-Type") != ContentTypeProblem {
-		t.Errorf("response = %d %s, want 500 problem+json", rec.Code, rec.Header().Get("Content-Type"))
+	if ct := rec.Result().Header.Get("Content-Type"); rec.Code != http.StatusInternalServerError || ct != ContentTypeProblem {
+		t.Errorf("response = %d %s, want 500 problem+json", rec.Code, ct)
 	}
 	want := `{"status":500,"code":"internal_error","title":"Internal Server Error"}` + "\n"
 	if rec.Body.String() != want {
```

`server/internal/platform/httpserver/contract_test.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/platform/httpserver/contract_test.go
+++ b/server/internal/platform/httpserver/contract_test.go
@@ -30,7 +30,7 @@
 		{"not ready", NewRouter(discard, notReady), "/readyz", http.StatusServiceUnavailable},
 		{"panic", middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }), discard), "/api/v0/boom", http.StatusInternalServerError},
 		{"bad request", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
-			errs.BadRequest(w, r, errors.New("Invalid format for parameter limit"))
+			errs.BadRequest(w, r, &InvalidParamFormatError{ParamName: "limit", Err: errors.New("not a number")})
 		}), "/api/v0/things?limit=x", http.StatusBadRequest},
 		{"internal error", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
 			errs.Write(w, r, errors.New("boom"))
@@ -54,8 +54,8 @@
 		t.Run(tt.name, func(t *testing.T) {
 			rec := serve(tt.h, httptest.NewRequest(http.MethodGet, tt.target, nil))
 
-			if rec.Code != tt.status || rec.Header().Get("Content-Type") != ContentTypeProblem {
-				t.Fatalf("response = %d %s, want %d problem+json", rec.Code, rec.Header().Get("Content-Type"), tt.status)
+			if ct := rec.Result().Header.Get("Content-Type"); rec.Code != tt.status || ct != ContentTypeProblem {
+				t.Fatalf("response = %d %s, want %d problem+json", rec.Code, ct, tt.status)
 			}
 			contract.CheckSchema(t, "Problem", rec.Body.Bytes())
 		})
```

- [ ] **Step 2: 请求体用例**

`server/internal/platform/httpserver/apitest/operations.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/platform/httpserver/apitest/operations.go
+++ b/server/internal/platform/httpserver/apitest/operations.go
@@ -104,8 +104,9 @@
 //  4. each required property missing;
 //  5. null for a nullable property: not 400;
 //  6. each format property with a wrong string;
-//  7. an undeclared property, a missing required property and a wrong
-//     format together: every problem in one answer.
+//  7. every kind of 1–4 and 6 that the schema has, each on a property of its
+//     own, together: every problem in one answer. Left out when the schema
+//     has only the first kind.
 func (o Operation) BodyCases() []BodyCase {
 	s := o.body
 	valid := validValue(s).(map[string]any)
@@ -115,55 +116,87 @@
 		out, _ := json.Marshal(body)
 		return out
 	}
-	var cases []BodyCase
-	cases = append(cases, BodyCase{Name: "undeclared property", Body: with(func(b map[string]any) { b[unknownField] = 1 }),
-		Fields: []FieldProblem{{unknownField, "not_allowed"}}})
 	names := slices.Sorted(maps.Keys(s.Properties))
+	var nested string
+	var optional, required, formatted []string
 	for _, name := range names {
 		p := s.Properties[name].Value
-		if isObject(p) {
-			nested := validValue(p).(map[string]any)
-			nested[unknownField] = 1
-			cases = append(cases, BodyCase{Name: "undeclared property in " + name, Body: with(func(b map[string]any) { b[name] = nested }),
-				Fields: []FieldProblem{{name + "." + unknownField, "not_allowed"}}})
-			break
+		if nested == "" && isObject(p) {
+			nested = name
 		}
+		if !isNullable(p) && !slices.Contains(s.Required, name) {
+			optional = append(optional, name)
+		}
+		if checkedFormat(p) {
+			formatted = append(formatted, name)
+		}
 	}
+	required = slices.Sorted(slices.Values(s.Required))
+	undeclaredIn := func(name string) map[string]any {
+		v := validValue(s.Properties[name].Value).(map[string]any)
+		v[unknownField] = 1
+		return v
+	}
+	wrong := func(name string) string { return "not-a-" + s.Properties[name].Value.Format }
+
+	cases := []BodyCase{{Name: "undeclared property", Body: with(func(b map[string]any) { b[unknownField] = 1 }),
+		Fields: []FieldProblem{{unknownField, "not_allowed"}}}}
+	if nested != "" {
+		cases = append(cases, BodyCase{Name: "undeclared property in " + nested, Body: with(func(b map[string]any) { b[nested] = undeclaredIn(nested) }),
+			Fields: []FieldProblem{{nested + "." + unknownField, "not_allowed"}}})
+	}
 	for _, name := range names {
-		switch p := s.Properties[name].Value; {
-		case isNullable(p):
+		switch {
+		case isNullable(s.Properties[name].Value):
 			cases = append(cases, BodyCase{Name: "null for nullable " + name, Body: with(func(b map[string]any) { b[name] = nil }), Accepted: true})
-		case !slices.Contains(s.Required, name):
+		case slices.Contains(optional, name):
 			cases = append(cases, BodyCase{Name: "null for optional " + name, Body: with(func(b map[string]any) { b[name] = nil }),
 				Fields: []FieldProblem{{name, "invalid_format"}}})
 		}
 	}
-	for _, name := range slices.Sorted(slices.Values(s.Required)) {
+	for _, name := range required {
 		cases = append(cases, BodyCase{Name: "missing " + name, Body: with(func(b map[string]any) { delete(b, name) }),
 			Fields: []FieldProblem{{name, "required"}}})
 	}
-	var formatted string
-	for _, name := range names {
-		if p := s.Properties[name].Value; checkedFormat(p) {
-			formatted = name
-			cases = append(cases, BodyCase{Name: "wrong " + p.Format + " in " + name, Body: with(func(b map[string]any) { b[name] = "not-a-" + p.Format }),
-				Fields: []FieldProblem{{name, "invalid_format"}}})
-		}
+	for _, name := range formatted {
+		cases = append(cases, BodyCase{Name: "wrong " + s.Properties[name].Value.Format + " in " + name,
+			Body: with(func(b map[string]any) { b[name] = wrong(name) }), Fields: []FieldProblem{{name, "invalid_format"}}})
 	}
-	if len(s.Required) > 0 {
-		missing := slices.Sorted(slices.Values(s.Required))[0]
-		all := []FieldProblem{{unknownField, "not_allowed"}, {missing, "required"}}
-		if formatted != "" && formatted != missing {
-			all = append(all, FieldProblem{formatted, "invalid_format"})
+
+	// Case 7: each kind takes the first property no earlier kind took.
+	body := maps.Clone(valid)
+	body[unknownField] = 1
+	all := []FieldProblem{{unknownField, "not_allowed"}}
+	used := map[string]bool{}
+	first := func(names []string) string {
+		for _, name := range names {
+			if !used[name] {
+				used[name] = true
+				return name
+			}
 		}
+		return ""
+	}
+	if name := first([]string{nested}); name != "" {
+		body[name] = undeclaredIn(name)
+		all = append(all, FieldProblem{name + "." + unknownField, "not_allowed"})
+	}
+	if name := first(optional); name != "" {
+		body[name] = nil
+		all = append(all, FieldProblem{name, "invalid_format"})
+	}
+	if name := first(required); name != "" {
+		delete(body, name)
+		all = append(all, FieldProblem{name, "required"})
+	}
+	if name := first(formatted); name != "" {
+		body[name] = wrong(name)
+		all = append(all, FieldProblem{name, "invalid_format"})
+	}
+	if len(all) >= 2 {
 		slices.SortFunc(all, func(a, b FieldProblem) int { return strings.Compare(a.Field, b.Field) })
-		cases = append(cases, BodyCase{Name: "every problem at once", Body: with(func(b map[string]any) {
-			b[unknownField] = 1
-			delete(b, missing)
-			if formatted != "" && formatted != missing {
-				b[formatted] = "not-a-format"
-			}
-		}), Fields: all})
+		out, _ := json.Marshal(body)
+		cases = append(cases, BodyCase{Name: "every problem at once", Body: out, Fields: all})
 	}
 	return cases
 }
```

`server/internal/platform/httpserver/apitest/operations_test.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/platform/httpserver/apitest/operations_test.go
+++ b/server/internal/platform/httpserver/apitest/operations_test.go
@@ -113,13 +113,63 @@
 		`missing name {"owner_id":"00000000-0000-0000-0000-000000000000"}`,
 		`missing owner_id {"name":"x"}`,
 		`wrong uuid in owner_id {"name":"x","owner_id":"not-a-uuid"}`,
-		`every problem at once {"nerve_undeclared":1,"owner_id":"not-a-format"}`,
+		`every problem at once {"count":null,"nerve_undeclared":1,"owner_id":"not-a-uuid","settings":{"nerve_undeclared":1}}`,
 	}
 	if !slices.Equal(got, want) {
 		t.Errorf("BodyCases() =\n%q\nwant\n%q", got, want)
 	}
 	last := post.BodyCases()[len(want)-1]
-	if wantFields := []FieldProblem{{"name", "required"}, {"nerve_undeclared", "not_allowed"}, {"owner_id", "invalid_format"}}; !slices.Equal(last.Fields, wantFields) {
+	wantFields := []FieldProblem{{"count", "invalid_format"}, {"name", "required"}, {"nerve_undeclared", "not_allowed"},
+		{"owner_id", "invalid_format"}, {"settings.nerve_undeclared", "not_allowed"}}
+	if !slices.Equal(last.Fields, wantFields) {
 		t.Errorf("every problem at once expects %v, want %v", last.Fields, wantFields)
 	}
 }
+
+// The case with every problem at once combines whichever kinds the schema
+// has, each on a property no other kind took, as soon as there are two; a
+// schema with only undeclared properties to offer has no such case.
+func TestBodyCasesCombineEveryKindTheSchemaHas(t *testing.T) {
+	tests := []struct {
+		name, schema string
+		want         string // the body of "every problem at once", or "" for none
+		fields       []FieldProblem
+	}{
+		{"no required property", "{type: object, properties: {label: {type: string}, due: {type: [string, 'null'], format: date-time}}}",
+			`{"due":"not-a-date-time","label":null,"nerve_undeclared":1}`,
+			[]FieldProblem{{"due", "invalid_format"}, {"label", "invalid_format"}, {"nerve_undeclared", "not_allowed"}}},
+		{"only a nested object", "{type: object, properties: {step: {type: object, properties: {a: {type: boolean}}}}}",
+			`{"nerve_undeclared":1,"step":{"nerve_undeclared":1}}`,
+			[]FieldProblem{{"nerve_undeclared", "not_allowed"}, {"step.nerve_undeclared", "not_allowed"}}},
+		{"the required property is the only formatted one", "{type: object, required: [id], properties: {id: {type: string, format: uuid}}}",
+			`{"nerve_undeclared":1}`, []FieldProblem{{"id", "required"}, {"nerve_undeclared", "not_allowed"}}},
+		{"only nullable properties", "{type: object, properties: {note: {type: [string, 'null']}}}", "", nil},
+	}
+	for _, tt := range tests {
+		t.Run(tt.name, func(t *testing.T) {
+			cases := bodyOperation(t, tt.schema).BodyCases()
+			var got *BodyCase
+			for i := range cases {
+				if cases[i].Name == "every problem at once" {
+					got = &cases[i]
+				}
+			}
+			switch {
+			case tt.want == "" && got != nil:
+				t.Errorf("every problem at once = %s, want no such case", got.Body)
+			case tt.want != "" && (got == nil || string(got.Body) != tt.want || !slices.Equal(got.Fields, tt.fields)):
+				t.Errorf("every problem at once = %+v, want %s with %v", got, tt.want, tt.fields)
+			}
+		})
+	}
+}
+
+// bodyOperation returns the one operation of a contract whose JSON body has
+// the given schema.
+func bodyOperation(t *testing.T, schema string) Operation {
+	t.Helper()
+	src := "openapi: 3.1.0\ninfo: {title: body, version: v0}\npaths:\n  /api/v0/x:\n    post:\n" +
+		"      requestBody:\n        content:\n          application/json:\n            schema: " + schema + "\n" +
+		"      responses: {'204': {description: none}}\n"
+	return contractFrom(t, src).Operations()[0]
+}
```

- [ ] **Step 3: 接口描述的两条规则**

`server/internal/platform/httpserver/apitest/rules_test.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/platform/httpserver/apitest/rules_test.go
+++ b/server/internal/platform/httpserver/apitest/rules_test.go
@@ -76,7 +76,8 @@
 
 // ruleCheck walks one document and collects its violations: path checks the
 // /api/v0/ prefix, operation the operationId, tags, default problem,
-// security and problem codes, closedObject additionalProperties, and schema
+// security and problem codes, parameter and requestBody the shapes the
+// whole-program tests build, closedObject additionalProperties, and schema
 // the keywords that oapi-codegen mistranslates.
 type ruleCheck struct {
 	doc        *openapi3.T
@@ -183,7 +184,7 @@
 		r.parameter(where+" parameters/"+p.Value.Name, p)
 	}
 	if body := op.RequestBody; body != nil && body.Ref == "" {
-		r.content(where+" requestBody", body.Value.Content)
+		r.requestBody(where+" requestBody", body.Value.Content)
 	}
 	responses := op.Responses.Map()
 	for _, status := range slices.Sorted(maps.Keys(responses)) {
@@ -219,7 +220,7 @@
 	}
 	for _, name := range slices.Sorted(maps.Keys(c.RequestBodies)) {
 		if body := c.RequestBodies[name]; body.Ref == "" {
-			r.content("components/requestBodies/"+name, body.Value.Content)
+			r.requestBody("components/requestBodies/"+name, body.Value.Content)
 		}
 	}
 	for _, name := range slices.Sorted(maps.Keys(c.Responses)) {
@@ -246,14 +247,29 @@
 	}
 }
 
+// parameter: a parameter declares schema, not content. The whole-program
+// tests fill each parameter from its schema (Operation.Target).
 func (r *ruleCheck) parameter(where string, p *openapi3.ParameterRef) {
 	if p.Ref != "" {
 		return
+	}
+	if len(p.Value.Content) > 0 {
+		r.report(where, "declares content; write schema")
 	}
 	r.schema(where, p.Value.Schema)
 	r.content(where, p.Value.Content)
 }
 
+// requestBody: a JSON request body is an object, which can take new
+// properties without breaking clients. The whole-program tests build every
+// body as one (Operation.BodyCases).
+func (r *ruleCheck) requestBody(where string, content openapi3.Content) {
+	if media := content.Get("application/json"); media != nil && media.Schema != nil && !isObject(media.Schema.Value) {
+		r.report(where, "application/json schema is not an object")
+	}
+	r.content(where, content)
+}
+
 func (r *ruleCheck) response(where string, res *openapi3.ResponseRef) {
 	if res.Ref != "" {
 		return
```

`server/internal/platform/httpserver/apitest/rules_cases_test.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/platform/httpserver/apitest/rules_cases_test.go
+++ b/server/internal/platform/httpserver/apitest/rules_cases_test.go
@@ -117,6 +117,12 @@
 		{"additionalProperties: true", thing, "    Thing:\n      type: object\n      additionalProperties: true\n", "components/schemas/Thing" + open},
 		{"composition with properties of its own", "    Named:\n      type: object\n",
 			"    Named:\n      type: object\n      properties: {id: {type: string}}\n", "components/schemas/Named" + open},
+		{"parameter with content", "- {name: kind, in: query, schema: {type: string, enum: [a, b]}}",
+			"- {name: kind, in: query, content: {application/json: {schema: {type: string}}}}",
+			"GET /api/v0/things parameters/kind: declares content; write schema"},
+		{"request body that is not an object", "      parameters:\n",
+			"      requestBody:\n        content:\n          application/json:\n            schema: {type: array, items: {type: string}}\n      parameters:\n",
+			"GET /api/v0/things requestBody: application/json schema is not an object"},
 		{"no top-level codes", "x-problem-codes: [bad_request, internal_error]\n", "", "top level: has no x-problem-codes"},
 		{"module code at the top level", "[bad_request, internal_error]", "[bad_request, things.taken]",
 			`top level: x-problem-codes holds "things.taken", which is not a platform code`},
```

- [ ] **Step 4: `bodyshapegen` 认出 3.1 的字符串写法**

`server/tools/bodyshapegen/generate.go`（对 `605f367` 的差异）：

```diff
--- a/server/tools/bodyshapegen/generate.go
+++ b/server/tools/bodyshapegen/generate.go
@@ -142,13 +142,13 @@
 		}
 		n.types = s.Type.Slice()
 	}
-	if s.Format != "" && s.Type.Includes("string") {
-		spec := g.types.String.Resolve(s.Format)
+	if format, spelled := stringFormat(s); format != "" && s.Type.Includes("string") {
+		spec := g.types.String.Resolve(format)
 		switch checker, ok := checkers[spec]; {
 		case ok:
 			n.format = checker
 		case spec.Type != "string":
-			return fail("format %q is generated as %s, which has no bodyshape checker", s.Format, spec.Type)
+			return fail("%s is generated as %s, which has no bodyshape checker", spelled, spec.Type)
 		}
 	}
 	if s.Type.Includes("integer") {
@@ -221,6 +221,23 @@
 	return len(g.nodes) - 1, nil
 }
 
+// stringFormat returns the format by which oapi-codegen picks a string's Go
+// type, and how the schema spelled it. Without format, an OpenAPI 3.1 string
+// with contentMediaType is generated as format "binary", one with
+// contentEncoding base64 as format "byte" (oapi-codegen v2.8.0,
+// pkg/codegen/schema.go:1294-1330); nerve's descriptions are all 3.1.
+func stringFormat(s *openapi3.Schema) (format, spelled string) {
+	switch {
+	case s.Format != "":
+		return s.Format, fmt.Sprintf("format %q", s.Format)
+	case s.ContentMediaType != "":
+		return "binary", `contentMediaType, read as format "binary",`
+	case s.ContentEncoding == "base64":
+		return "byte", `contentEncoding base64, read as format "byte",`
+	}
+	return "", ""
+}
+
 // uncheckedNumber says why a number of JSON type t with format is generated as
 // a Go type whose range bodyshape does not check, or returns "" when it is one
 // of covered. bodyshape's Integer is a literal that an int64 holds, its Number
```

`server/tools/bodyshapegen/generate_test.go`（对 `605f367` 的差异）：

```diff
--- a/server/tools/bodyshapegen/generate_test.go
+++ b/server/tools/bodyshapegen/generate_test.go
@@ -90,6 +90,10 @@
 		{"not", "{not: {type: string}}", "due: allOf, oneOf and not are not supported"},
 		{"anyOf of two types", "{anyOf: [{type: string}, {type: integer}]}", "due: anyOf is supported only as [X, {type: 'null'}]"},
 		{"nullable", "{type: string, nullable: true}", "due: nullable is OpenAPI 3.0"},
+		{"contentMediaType", "{type: string, contentMediaType: image/png}",
+			`due: contentMediaType, read as format "binary", is generated as openapi_types.File`},
+		{"contentEncoding base64", "{type: string, contentEncoding: base64}",
+			`due: contentEncoding base64, read as format "byte", is generated as []byte`},
 		{"nested", "{type: object, properties: {at: {type: array, items: {type: string, format: date}}}}", `due.at[]: format "date"`},
 		{"nested number", "{type: array, items: {anyOf: [{type: integer, format: uint64}, {type: 'null'}]}}", `due[]: integer format "uint64" is generated as uint64`},
 	}
@@ -104,6 +108,20 @@
 	}
 }
 
+// oapi-codegen reads contentMediaType and contentEncoding only when format is
+// absent, and maps no contentEncoding but base64: these stay strings.
+func TestGenerateAcceptsTheStringsItCanCheck(t *testing.T) {
+	for _, property := range []string{
+		"{type: string, contentEncoding: base64url}",
+		"{type: string, format: uuid, contentEncoding: base64}",
+		"{type: string, format: date-time, contentMediaType: text/plain}",
+	} {
+		if _, err := generate(bodySpec(t, property), thingsConf); err != nil {
+			t.Errorf("generate() of %s = %v, want nil", property, err)
+		}
+	}
+}
+
 // A number passes when it is generated as a Go type whose whole range
 // bodyshape checks: int or int64 for an integer, float64 for a number.
 func TestGenerateAcceptsTheNumbersItCanCheck(t *testing.T) {
```

- [ ] **Step 5: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。`bootstrap` 的 `TestBodiesThatBreakTheStructureAnswer400` 对现有的四个请求体跑新的第 7 种用例。

- [ ] **Step 6: 提交**

```bash
git add server/internal/platform/httpserver server/tools/bodyshapegen
```
```bash
git commit -m "feat(M2/P3a): a parameter that does not bind is named in the 400; body cases combine every kind of problem

APIErrors.BadRequest reads the parameter from the binding error and logs
the error's text at debug level only. The contract rules require schema
parameters and object bodies; bodyshapegen reads contentMediaType and
contentEncoding as oapi-codegen does.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** `TestAPIErrorsBadRequestNamesTheParameter` 的 7 个子测试和 `TestBodyCasesCombineEveryKindTheSchemaHas` 通过；`make test` 通过。

---

### Task 3: archtest：google/uuid 只许 `oapi-codegen/runtime` 导入；生成代码的 uuid 规则

**Files:**
- Create: `server/internal/archtest/generated_test.go`
- Modify: `server/internal/archtest/deps_test.go`、`purity_test.go`
- Modify: `server/internal/archtest/binary_test.go`（过渡：Task 12 加上时区数据库的测试）

**Interfaces:**
- Produces（spec 2.5，M2 设计 3.12）：
  - `bannedImports(g, root, isBanned func(importer, path string) bool)`：按导入的边判断，同一个包可以只对某些导入者禁用；报出的链以被禁的包结尾；
  - `isBannedFromBinary(importer, path)`：`github.com/google/uuid` 由 `github.com/oapi-codegen/runtime` 模块的包导入时放行，别的导入者照旧禁止；
  - `TestGeneratedCodeUsesTheStandardUUID`：`apigen/*.gen.go` 和 `modules/*/adapter/http/gen/*.gen.go` 不得引用 `github.com/oapi-codegen/runtime/types` 的 `UUID`（不论导入时用什么名字）；两个 glob 都必须有匹配。
- runtime 本身在 Task 7 加入；这个 Task 先把守卫改写好，Task 7 的 `go get` 之后传递依赖测试照样通过。

**Tests:**
- `binary_test.go`：`TestBannedImports` 的图加上 runtime、runtime/types（放行）、名字相近的 `runtime-extra`（仍报出）和平台包导入 google/uuid（仍报出），期望的两条链逐字核对。
- `generated_test.go`：`TestRuntimeUUIDUses`（6 个：生成代码的名字、别的名字、包自己的名字、runtime 的别的类型、标准库的 uuid、别的名为 `types` 的包）；`TestGeneratedCodeUsesTheStandardUUID` 扫描现有的生成文件。
- 原型中的接线核对（spec 附录 A）：删掉 identity 的 `oapi-codegen.yaml` 中 uuid 的映射再生成，`TestGeneratedCodeUsesTheStandardUUID` 报出 `server.gen.go` 中的位置。

- [ ] **Step 1: 按边判断的禁用**

`server/internal/archtest/deps_test.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/archtest/deps_test.go
+++ b/server/internal/archtest/deps_test.go
@@ -54,31 +54,33 @@
 	return strings.Join(hops, " → ")
 }
 
-// bannedImports walks g from root and returns every banned package that an
-// allowed package imports, each with the shortest chain to it. The imports
-// of a banned package are not followed: it has to go anyway.
-func bannedImports(g graph, root string, isBanned func(path string) bool) []bannedImport {
+// bannedImports walks g from root and returns every import of a banned
+// package by a package that root reaches, each with the shortest chain to the
+// importer. isBanned judges one import edge, so a package can be banned for
+// some importers only. The imports of a banned package are not followed: it
+// has to go anyway.
+func bannedImports(g graph, root string, isBanned func(importer, path string) bool) []bannedImport {
 	importer := map[string]string{root: ""}
 	var found []bannedImport
 	for queue := []string{root}; len(queue) > 0; queue = queue[1:] {
 		pkg := queue[0]
-		if isBanned(pkg) {
-			var chain []string
-			for p := pkg; p != ""; p = importer[p] {
-				chain = append(chain, p)
-			}
-			slices.Reverse(chain)
-			found = append(found, bannedImport{chain})
-			continue
-		}
 		for _, dep := range g[pkg] {
+			if isBanned(pkg, dep) {
+				var chain []string
+				for p := pkg; p != ""; p = importer[p] {
+					chain = append(chain, p)
+				}
+				slices.Reverse(chain)
+				found = append(found, bannedImport{append(chain, dep)})
+				continue
+			}
 			if _, seen := importer[dep]; !seen {
 				importer[dep] = pkg
 				queue = append(queue, dep)
 			}
 		}
 	}
-	slices.SortFunc(found, func(a, b bannedImport) int {
+	slices.SortStableFunc(found, func(a, b bannedImport) int {
 		return strings.Compare(a.banned(), b.banned())
 	})
 	return found
```

`server/internal/archtest/purity_test.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/archtest/purity_test.go
+++ b/server/internal/archtest/purity_test.go
@@ -10,7 +10,7 @@
 // reachesInfrastructure marks what the pure layers (domain, app,
 // internal/shared) must not depend on, even indirectly: net/http,
 // database/sql and any module other than this one.
-func reachesInfrastructure(path string) bool {
+func reachesInfrastructure(_, path string) bool {
 	_, inModule := local(path)
 	return !inModule && isInfrastructure(path)
 }
```

`server/internal/archtest/binary_test.go`（完整内容）：

```go
package archtest

import (
	"slices"
	"strings"
	"testing"
)

// bannedFromBinary lists the import path prefixes that must not reach the
// nerve binary: the test-only kin-openapi (apitest), testcontainers and
// docker (pgtest), and google/uuid, which the standard library's uuid
// replaces. Rule 8 keeps the test helpers themselves out, but generated code
// (an embedded spec) or any other import could still pull these in, and
// depguard does not look at generated files.
func bannedFromBinary() []string {
	return []string{
		"github.com/getkin/kin-openapi",
		"github.com/testcontainers/",
		"github.com/google/uuid",
		"github.com/docker/",
	}
}

// oapiRuntime is the module that generated code imports to bind parameters.
// It imports google/uuid itself (runtime/types/uuid.go and the runtime
// package's styleparam.go, v1.7.0), so its packages, and only they, may
// (M2 design 3.12). That generated code uses the standard library's uuid is
// checked directly, by TestGeneratedCodeUsesTheStandardUUID.
const oapiRuntime = "github.com/oapi-codegen/runtime"

// isBannedFromBinary judges the import of path by importer.
func isBannedFromBinary(importer, path string) bool {
	if strings.HasPrefix(path, "github.com/google/uuid") && (importer == oapiRuntime || strings.HasPrefix(importer, oapiRuntime+"/")) {
		return false
	}
	return slices.ContainsFunc(bannedFromBinary(), func(prefix string) bool { return strings.HasPrefix(path, prefix) })
}

func TestNerveBinaryLinksNoBannedModule(t *testing.T) {
	root := m("cmd/nerve")
	g := loadDeps(t, "./cmd/nerve")
	// A loader problem must not pass as "nothing banned".
	for _, want := range []string{root, m("internal/bootstrap"), "net/http"} {
		if _, ok := g[want]; !ok {
			t.Fatalf("dependency graph lacks %s; loaded %d packages", want, len(g))
		}
	}
	for _, b := range bannedImports(g, root, isBannedFromBinary) {
		t.Errorf("the nerve binary must not link %s, imported via %s", b.banned(), b.via())
	}
}

// google/uuid is reached first through oapi-codegen/runtime, which may
// import it; every other import of it is still reported, each on its own.
func TestBannedImports(t *testing.T) {
	root := m("cmd/nerve")
	gen := m("internal/modules/issue/adapter/http/gen")
	g := graph{
		root:                                        {"github.com/spf13/cobra", m("internal/bootstrap")},
		"github.com/spf13/cobra":                    {"github.com/spf13/pflag"},
		m("internal/bootstrap"):                     {gen, m("internal/platform/httpserver")},
		gen:                                         {"github.com/oapi-codegen/runtime", "github.com/oapi-codegen/runtime-extra", "net/http"},
		"github.com/oapi-codegen/runtime":           {"github.com/google/uuid", "github.com/oapi-codegen/runtime/types"},
		"github.com/oapi-codegen/runtime/types":     {"github.com/google/uuid"},
		"github.com/oapi-codegen/runtime-extra":     {"github.com/google/uuid"},
		m("internal/platform/httpserver"):           {"github.com/getkin/kin-openapi/openapi3", m("internal/platform/httpserver/bodyshape"), "net/http"},
		"github.com/getkin/kin-openapi/openapi3":    {"github.com/google/uuid"},
		m("internal/platform/httpserver/bodyshape"): {"github.com/google/uuid"},
		// Not reachable from the root: test helpers stay out of the walk.
		m("internal/platform/postgres/pgtest"): {"github.com/testcontainers/testcontainers-go"},
	}
	var got []string
	for _, b := range bannedImports(g, root, isBannedFromBinary) {
		got = append(got, b.via())
	}
	want := []string{
		"cmd/nerve → internal/bootstrap → internal/platform/httpserver → github.com/getkin/kin-openapi/openapi3",
		"cmd/nerve → internal/bootstrap → internal/modules/issue/adapter/http/gen → github.com/oapi-codegen/runtime-extra → github.com/google/uuid",
		"cmd/nerve → internal/bootstrap → internal/platform/httpserver → internal/platform/httpserver/bodyshape → github.com/google/uuid",
	}
	if !slices.Equal(got, want) {
		t.Errorf("bannedImports() =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
```

- [ ] **Step 2: 生成代码的 uuid 规则**

`server/internal/archtest/generated_test.go`（新文件）：

```go
package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
)

// runtimeTypes is the package whose UUID type oapi-codegen uses for format:
// uuid when a module's type-mapping lacks the standard library's uuid.
const runtimeTypes = "github.com/oapi-codegen/runtime/types"

// The binary may link google/uuid through oapi-codegen/runtime (see
// isBannedFromBinary), so what M0 guarded by banning it is checked here
// directly: no file that oapi-codegen generates, for the shared components or
// for a module's HTTP adapter, uses the runtime's UUID type. The program has one
// uuid type, and a module whose type-mapping lacks it fails here (M2 design
// 3.12).
func TestGeneratedCodeUsesTheStandardUUID(t *testing.T) {
	registerSources(t)
	var files []string
	for _, pattern := range []string{"internal/platform/httpserver/apigen/*.gen.go", "internal/modules/*/adapter/http/gen/*.gen.go"} {
		matches, err := filepath.Glob(filepath.Join(moduleRoot, pattern))
		// A wrong pattern must not pass as "nothing uses it".
		if err != nil || len(matches) == 0 {
			t.Fatalf("glob %s = %q, %v; want generated files", pattern, matches, err)
		}
		files = append(files, matches...)
	}
	fset := token.NewFileSet()
	for _, path := range files {
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		for _, at := range runtimeUUIDUses(fset, f) {
			t.Errorf("%s uses %s.UUID; map format uuid to the standard library's uuid.UUID in the oapi-codegen config", at, runtimeTypes)
		}
	}
}

// runtimeUUIDUses returns the positions where f refers to UUID of
// runtimeTypes, under whatever name f imports it.
func runtimeUUIDUses(fset *token.FileSet, f *ast.File) []string {
	var names []string
	for _, imp := range f.Imports {
		if path, err := strconv.Unquote(imp.Path.Value); err != nil || path != runtimeTypes {
			continue
		}
		name := "types"
		if imp.Name != nil {
			name = imp.Name.Name
		}
		names = append(names, name)
	}
	var found []string
	ast.Inspect(f, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == "UUID" {
			if id, ok := sel.X.(*ast.Ident); ok && slices.Contains(names, id.Name) {
				found = append(found, fset.Position(sel.Pos()).String())
			}
		}
		return true
	})
	return found
}

func TestRuntimeUUIDUses(t *testing.T) {
	tests := []struct {
		name, src string
		want      []string
	}{
		{"the generated name", `package gen
import openapi_types "github.com/oapi-codegen/runtime/types"
type Thing struct{ ID openapi_types.UUID }`, []string{"gen.go:3:23"}},
		{"another name", `package gen
import rt "github.com/oapi-codegen/runtime/types"
func f(id *rt.UUID) {}`, []string{"gen.go:3:12"}},
		{"the package's own name", `package gen
import "github.com/oapi-codegen/runtime/types"
var ids []types.UUID`, []string{"gen.go:3:11"}},
		{"another type of the runtime", `package gen
import openapi_types "github.com/oapi-codegen/runtime/types"
var day openapi_types.Date`, nil},
		{"the standard library's uuid", `package gen
import "uuid"
var id uuid.UUID`, nil},
		{"another package named types", `package gen
import "example.com/types"
var id types.UUID`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "gen.go", tt.src, parser.SkipObjectResolution)
			if err != nil {
				t.Fatal(err)
			}
			if got := runtimeUUIDUses(fset, f); !slices.Equal(got, tt.want) {
				t.Errorf("runtimeUUIDUses() = %q, want %q", got, tt.want)
			}
		})
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
git add server/internal/archtest
```
```bash
git commit -m "test(M2/P3a): google/uuid only through oapi-codegen/runtime; generated code uses the standard uuid

The binary test judges each import edge, so the runtime's own packages
may import google/uuid. What banning it guarded is now checked directly:
no generated file refers to the runtime's UUID type (M2 design 3.12).

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** `TestBannedImports`、`TestRuntimeUUIDUses`、`TestGeneratedCodeUsesTheStandardUUID` 通过；`make test` 通过。

---

### Task 4: 页游标的封套；PAT、令牌规格、页大小和令牌列表的游标

**Files:**
- Create: `server/internal/shared/cursor.go`、`cursor_test.go`
- Modify: `server/internal/shared/error.go`
- Create: `server/internal/modules/identity/domain/api_token.go`、`api_token_test.go`

**Interfaces:**
- Produces（spec 2.6、2.7，M2 设计 3.4、3.12、4.4）：
  - `shared.EncodeCursor(payload any) (string, error)`：`{"v":1,"p":…}` 的无补位 base64url；`shared.DecodeCursor(cursor string, payload any) error`：只认 `EncodeCursor` 写出的形式（严格的 JSON 封套、没有别的成员、后面没有第二个值、版本 1、有载荷且不是 `null`），其余都是 `shared.InvalidCursor()`：400 `bad_request`，`errors[{field: cursor, code: invalid_format}]`；
  - `shared.CodeBadRequest`；
  - `domain.PATPrefix = "nrv_pat_"`、`domain.PAT [32]byte`（`String()`、`Hash()` = 整个令牌的 SHA-256）、`domain.ParsePAT(s) (PAT, bool)`（51 个字符，`Strict()` 解码）；
  - `domain.APIToken`、`domain.APITokenSpec{Label *string; Description string; ExpiredAt *time.Time}`、`domain.CheckAPIToken(spec, now) error`（标签 1–255 个字符；标签和说明不含 NUL；`expired_at` 晚于 `now`；一次报出全部问题）；
  - `domain.DefaultPageSize = 50`、`MaxPageSize = 100`、`domain.PageSize(limit *int) (int, error)`（1–100 之外 422 `limit` `out_of_range`）；
  - `domain.APITokenCursor{CreatedAt; ID}`：JSON 是 `[created_at, id]`。

**Tests:**
- `cursor_test.go`：`TestCursorRoundTrip`（编码结果逐字核对：封套的 JSON 加无补位 base64url）；`TestDecodeCursorRejects`（13 个：空串、不是 base64、带补位、标准字母表、不是 JSON、不是对象、多余的成员、后面还有值、别的版本、没有版本、没有载荷、`null` 载荷、别的列表的载荷），每个都核对 400 和 `cursor` 字段。
- `api_token_test.go`：`TestPATRoundTrip`；`TestParsePATRejects`（11 个：空串、没有前缀、刷新令牌的前缀、大写前缀、少一个字符、多一个字符、`=` 补位、没用到的位不为 0、中间换行、标准字母表、前面的空格）；`TestCheckAPITokenAcceptsAValidSpec`；`TestCheckAPITokenReportsEveryField`（6 个，含一次报出三个）；`TestPageSize`（`nil`、1、100；0、101、-1）；`TestAPITokenCursorRoundTrip`（微秒精度往返）；`TestAPITokenCursorRejects`（8 种载荷）。

- [ ] **Step 1: 页游标**

`server/internal/shared/error.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/shared/error.go
+++ b/server/internal/shared/error.go
@@ -27,6 +27,7 @@
 // Codes of the platform problems that domain errors carry. Module codes are
 // prefixed with the module, e.g. "identity.email_taken".
 const (
+	CodeBadRequest       = "bad_request"
 	CodeValidationFailed = "validation_failed"
 	CodeUnauthorized     = "unauthorized"
 	CodeRateLimited      = "rate_limited"
```

`server/internal/shared/cursor.go`（新文件）：

```go
package shared

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
)

// cursorVersion is the version of the cursor envelope, v.
const cursorVersion = 1

// cursorEnvelope is what a page cursor encodes (M2 design 3.12): the
// envelope's version and the list's own payload.
type cursorEnvelope struct {
	V int             `json:"v"`
	P json.RawMessage `json:"p"`
}

// EncodeCursor returns the page cursor of payload: the JSON {"v":1,"p":…}
// in unpadded base64url. Each list defines its payload by its own sort key,
// ending in the id, so pages neither repeat nor skip a row (M2 design 3.12).
func EncodeCursor(payload any) (string, error) {
	p, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	out, err := json.Marshal(cursorEnvelope{V: cursorVersion, P: p})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(out), nil
}

// DecodeCursor reads a cursor that EncodeCursor wrote into payload, whose
// JSON decoding judges the payload. Anything else is InvalidCursor: not
// unpadded base64url, not the envelope and nothing more, another version, no
// payload, or a payload that does not decode.
func DecodeCursor(cursor string, payload any) error {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return InvalidCursor()
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var env cursorEnvelope
	if err := dec.Decode(&env); err != nil || !errors.Is(dec.Decode(new(json.RawMessage)), io.EOF) {
		return InvalidCursor()
	}
	if env.V != cursorVersion || len(env.P) == 0 || string(env.P) == "null" {
		return InvalidCursor()
	}
	if err := json.Unmarshal(env.P, payload); err != nil {
		return InvalidCursor()
	}
	return nil
}

// InvalidCursor reports a cursor that is not one the list issued: 400
// bad_request on the cursor parameter (M2 design 3.11, 3.12).
func InvalidCursor() *Error {
	return &Error{
		Kind: KindBadRequest, Code: CodeBadRequest, Detail: "The cursor is not one this list issued.",
		Fields: []FieldError{{Field: "cursor", Code: FieldInvalidFormat, Message: "is not a cursor of this list"}},
	}
}
```

`server/internal/shared/cursor_test.go`（新文件）：

```go
package shared_test

import (
	"encoding/base64"
	"errors"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// page stands for a list's payload.
type page struct {
	After string `json:"after"`
}

func b64(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }

func TestCursorRoundTrip(t *testing.T) {
	cursor, err := shared.EncodeCursor(page{After: "x?y"})
	if err != nil {
		t.Fatal(err)
	}
	// The envelope is the JSON {"v":1,"p":…} in unpadded base64url.
	if want := b64(`{"v":1,"p":{"after":"x?y"}}`); cursor != want {
		t.Errorf("EncodeCursor() = %q, want %q", cursor, want)
	}

	var got page
	if err := shared.DecodeCursor(cursor, &got); err != nil || got.After != "x?y" {
		t.Errorf("DecodeCursor() = %+v, %v; want the payload back", got, err)
	}
}

// Anything but a cursor that EncodeCursor wrote is 400 bad_request on the
// cursor parameter.
func TestDecodeCursorRejects(t *testing.T) {
	valid := `{"v":1,"p":{"after":"x"}}`
	tests := []struct{ name, cursor string }{
		{"empty", ""},
		{"not base64", "not a cursor!"},
		{"padded", base64.URLEncoding.EncodeToString([]byte(valid + " "))},
		{"standard alphabet", base64.RawStdEncoding.EncodeToString([]byte(`{"v":1,"p":{"after":"??>"}}`))},
		{"not JSON", b64(`{"v":1,`)},
		{"not an object", b64(`[1,{"after":"x"}]`)},
		{"an unknown member", b64(`{"v":1,"p":{"after":"x"},"x":0}`)},
		{"a second value after it", b64(valid + `{}`)},
		{"another version", b64(`{"v":2,"p":{"after":"x"}}`)},
		{"no version", b64(`{"p":{"after":"x"}}`)},
		{"no payload", b64(`{"v":1}`)},
		{"a null payload", b64(`{"v":1,"p":null}`)},
		{"a payload of another list", b64(`{"v":1,"p":{"after":5}}`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got page
			err := shared.DecodeCursor(tt.cursor, &got)

			var se *shared.Error
			want := []shared.FieldError{{Field: "cursor", Code: shared.FieldInvalidFormat, Message: "is not a cursor of this list"}}
			if !errors.As(err, &se) || se.ProblemStatus() != 400 || se.Code != shared.CodeBadRequest || !slices.Equal(se.Fields, want) {
				t.Errorf("DecodeCursor(%q) = %v, want 400 bad_request on cursor", tt.cursor, err)
			}
		})
	}
	// The same bytes with the version it expects decode: each case above
	// breaks one thing.
	var got page
	if err := shared.DecodeCursor(b64(valid), &got); err != nil {
		t.Errorf("DecodeCursor(valid) = %v", err)
	}
}
```

- [ ] **Step 2: PAT 与令牌的领域规则**

`server/internal/modules/identity/domain/api_token.go`（新文件）：

```go
package domain

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// PATPrefix starts every personal access token (M2 design 3.4).
const PATPrefix = "nrv_pat_"

// PAT is the random part of a personal access token: 32 bytes. The server
// stores only Hash.
type PAT [32]byte

// patTextLen is the length of a token as the client holds it: the prefix
// and 43 characters of unpadded base64url, 51.
var patTextLen = len(PATPrefix) + base64.RawURLEncoding.EncodedLen(len(PAT{}))

// String is the token the client holds.
func (p PAT) String() string {
	return PATPrefix + base64.RawURLEncoding.EncodeToString(p[:])
}

// Hash is what api_tokens.token_hash holds: the SHA-256 of the whole token
// (M2 design 3.4).
func (p PAT) Hash() []byte {
	sum := sha256.Sum256([]byte(p.String()))
	return sum[:]
}

// ParsePAT reads a token that String wrote, without looking anything up. It
// accepts only that spelling: the prefix, then 43 characters of unpadded
// base64url whose unused last bits are zero, nothing around them.
func ParsePAT(s string) (PAT, bool) {
	if len(s) != patTextLen || !strings.HasPrefix(s, PATPrefix) {
		return PAT{}, false
	}
	// A decoder skips \r and \n: with them inside, fewer than 32 bytes come out.
	raw, err := base64.RawURLEncoding.Strict().DecodeString(s[len(PATPrefix):])
	if err != nil || len(raw) != len(PAT{}) {
		return PAT{}, false
	}
	return PAT(raw), true
}

// APIToken is a personal access token as the API lists it: never the token
// itself.
type APIToken struct {
	ID          uuid.UUID
	Label       string
	Description string
	ExpiredAt   *time.Time // nil: never expires
	LastUsed    *time.Time // nil: never used
	CreatedAt   time.Time
}

// APITokenSpec is what the caller asks for when creating a token. A nil
// Label asks for a generated one.
type APITokenSpec struct {
	Label       *string
	Description string
	ExpiredAt   *time.Time
}

// maxLabelLength is api_tokens.label's varchar(255), in characters.
const maxLabelLength = 255

// CheckAPIToken checks spec at now (M2 design 4.4, 4.6): a label of 1–255
// characters, a label and a description without NUL, which the database
// cannot store, and an expiry after now. Every problem is reported at once,
// as one 422 validation_failed.
func CheckAPIToken(spec APITokenSpec, now time.Time) error {
	var fields []shared.FieldError
	if spec.Label != nil {
		if f := checkText("label", *spec.Label, true, maxLabelLength); f != nil {
			fields = append(fields, *f)
		}
	}
	if f := checkText("description", spec.Description, false, 0); f != nil {
		fields = append(fields, *f)
	}
	if spec.ExpiredAt != nil && !spec.ExpiredAt.After(now) {
		fields = append(fields, shared.FieldError{Field: "expired_at", Code: shared.FieldMustBeFuture, Message: "must be in the future"})
	}
	if len(fields) > 0 {
		return shared.Invalid(fields...)
	}
	return nil
}

// checkText checks a text field: empty when it must not be, longer than
// maxLen characters (no bound when 0), or holding a NUL, which Postgres
// text cannot store.
func checkText(field, s string, nonEmpty bool, maxLen int) *shared.FieldError {
	switch {
	case nonEmpty && s == "":
		return &shared.FieldError{Field: field, Code: shared.FieldTooShort, Message: "must not be empty"}
	case maxLen > 0 && utf8.RuneCountInString(s) > maxLen:
		return &shared.FieldError{Field: field, Code: shared.FieldTooLong, Message: fmt.Sprintf("must be at most %d characters", maxLen)}
	case strings.ContainsRune(s, 0):
		return &shared.FieldError{Field: field, Code: shared.FieldInvalidFormat, Message: "must not contain a NUL character"}
	}
	return nil
}

// Page sizes of a list (api/common.yaml Limit).
const (
	DefaultPageSize = 50
	MaxPageSize     = 100
)

// PageSize returns the page size that limit asks for: DefaultPageSize when
// it is nil. Outside 1–MaxPageSize it is 422 validation_failed on limit
// (M2 design 3.11); a limit that is no integer never gets here, the
// parameter binding answers 400.
func PageSize(limit *int) (int, error) {
	switch {
	case limit == nil:
		return DefaultPageSize, nil
	case *limit < 1 || *limit > MaxPageSize:
		return 0, shared.Invalid(shared.FieldError{Field: "limit", Code: shared.FieldOutOfRange, Message: "must be between 1 and 100"})
	}
	return *limit, nil
}

// errCursorShape is a token list cursor that is not [created_at, id].
var errCursorShape = errors.New("the cursor is not [created_at, id]")

// APITokenCursor is the payload of the token list's cursor (M2 design
// 3.12): the last row's created_at and id, the list's order.
type APITokenCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

// MarshalJSON writes the cursor as [created_at, id].
func (c APITokenCursor) MarshalJSON() ([]byte, error) {
	return json.Marshal([2]string{c.CreatedAt.Format(time.RFC3339Nano), c.ID.String()})
}

// UnmarshalJSON reads what MarshalJSON wrote: an array of exactly an RFC
// 3339 time and a uuid.
func (c *APITokenCursor) UnmarshalJSON(b []byte) error {
	var parts []string
	if err := json.Unmarshal(b, &parts); err != nil {
		return err
	}
	if len(parts) != 2 {
		return errCursorShape
	}
	at, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return err
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return err
	}
	*c = APITokenCursor{CreatedAt: at, ID: id}
	return nil
}
```

`server/internal/modules/identity/domain/api_token_test.go`（新文件）：

```go
package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// now is the time the rules are checked at.
var now = time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)

// samplePAT has every byte different, so a shifted or truncated copy differs.
func samplePAT() PAT {
	var p PAT
	for i := range p {
		p[i] = byte(i*7 + 3)
	}
	return p
}

func TestPATRoundTrip(t *testing.T) {
	p := samplePAT()
	s := p.String()

	// The secret-scanning pattern of M2 design 8.6 matches it.
	if !regexp.MustCompile(`^nrv_pat_[A-Za-z0-9_-]{43}$`).MatchString(s) {
		t.Errorf("String() = %q, want nrv_pat_ and 43 base64url characters", s)
	}
	if got, ok := ParsePAT(s); !ok || got != p {
		t.Errorf("ParsePAT(String()) = %x, %v; want the token back", got, ok)
	}
	// The whole token is hashed, prefix included, not only the secret.
	if want := sha256.Sum256([]byte(s)); !bytes.Equal(p.Hash(), want[:]) {
		t.Errorf("Hash() = %x, want SHA-256 of %q", p.Hash(), s)
	}
}

// Only the spelling String writes is a token: nothing is looked up for
// anything else.
func TestParsePATRejects(t *testing.T) {
	s := samplePAT().String()
	last := s[len(s)-1]
	// The last character carries 4 bits of the secret and 2 unused ones:
	// flipping its lowest bit keeps the secret and breaks the strict form.
	alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	unusedBit := string(alphabet[strings.IndexByte(alphabet, last)^1])
	tests := []struct{ name, token string }{
		{"empty", ""},
		{"no prefix", s[len(PATPrefix):]},
		{"a refresh token's prefix", RefreshTokenPrefix + s[len(PATPrefix):]},
		{"upper-case prefix", strings.ToUpper(PATPrefix) + s[len(PATPrefix):]},
		{"one character short", s[:len(s)-1]},
		{"one character more", s + "A"},
		{"padded", s[:len(s)-1] + "="},
		{"unused bits set", s[:len(s)-1] + unusedBit},
		{"a newline inside", s[:20] + "\n" + s[21:]},
		{"the standard alphabet", s[:20] + "+" + s[21:]},
		{"blank around it", " " + s[:len(s)-1]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, ok := ParsePAT(tt.token); ok {
				t.Errorf("ParsePAT(%q) = %x, true; want false", tt.token, got)
			}
		})
	}
}

func TestCheckAPITokenAcceptsAValidSpec(t *testing.T) {
	label, max := "deploy bot", strings.Repeat("界", 255)
	later := now.Add(time.Nanosecond)
	for _, spec := range []APITokenSpec{
		{},
		{Label: &label, Description: "ci", ExpiredAt: &later},
		{Label: &max},
	} {
		if err := CheckAPIToken(spec, now); err != nil {
			t.Errorf("CheckAPIToken(%+v) = %v, want nil", spec, err)
		}
	}
}

// Every problem is reported at once, as one 422.
func TestCheckAPITokenReportsEveryField(t *testing.T) {
	empty, long, nul := "", strings.Repeat("界", 256), "a\x00b"
	past := now.Add(-time.Second)
	tests := []struct {
		name string
		spec APITokenSpec
		want []shared.FieldError
	}{
		{"empty label", APITokenSpec{Label: &empty}, []shared.FieldError{{Field: "label", Code: "too_short", Message: "must not be empty"}}},
		{"long label", APITokenSpec{Label: &long}, []shared.FieldError{{Field: "label", Code: "too_long", Message: "must be at most 255 characters"}}},
		{"NUL in the label", APITokenSpec{Label: &nul}, []shared.FieldError{{Field: "label", Code: "invalid_format", Message: "must not contain a NUL character"}}},
		{"NUL in the description", APITokenSpec{Description: nul}, []shared.FieldError{{Field: "description", Code: "invalid_format", Message: "must not contain a NUL character"}}},
		{"expiry now", APITokenSpec{ExpiredAt: &now}, []shared.FieldError{{Field: "expired_at", Code: "must_be_future", Message: "must be in the future"}}},
		{"all at once", APITokenSpec{Label: &empty, Description: nul, ExpiredAt: &past}, []shared.FieldError{
			{Field: "label", Code: "too_short", Message: "must not be empty"},
			{Field: "description", Code: "invalid_format", Message: "must not contain a NUL character"},
			{Field: "expired_at", Code: "must_be_future", Message: "must be in the future"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckAPIToken(tt.spec, now)

			var se *shared.Error
			if !errors.As(err, &se) || se.Code != shared.CodeValidationFailed || !slices.Equal(se.Fields, tt.want) {
				t.Errorf("CheckAPIToken() = %+v, want validation_failed with %+v", err, tt.want)
			}
		})
	}
}

func TestPageSize(t *testing.T) {
	for _, tt := range []struct {
		limit *int
		want  int
	}{{nil, 50}, {ptr(1), 1}, {ptr(100), 100}} {
		if got, err := PageSize(tt.limit); err != nil || got != tt.want {
			t.Errorf("PageSize(%v) = %d, %v; want %d", tt.limit, got, err, tt.want)
		}
	}
	for _, limit := range []int{0, 101, -1} {
		_, err := PageSize(&limit)

		var se *shared.Error
		want := []shared.FieldError{{Field: "limit", Code: "out_of_range", Message: "must be between 1 and 100"}}
		if !errors.As(err, &se) || se.Code != shared.CodeValidationFailed || !slices.Equal(se.Fields, want) {
			t.Errorf("PageSize(%d) = %v, want 422 limit out_of_range", limit, err)
		}
	}
}

func ptr[T any](v T) *T { return &v }

func TestAPITokenCursorRoundTrip(t *testing.T) {
	c := APITokenCursor{CreatedAt: time.Date(2026, 9, 25, 10, 0, 0, 123456000, time.UTC), ID: uuid.MustParse("0199a2b4-0000-7000-8000-000000000001")}

	b, err := json.Marshal(c)
	if want := `["2026-09-25T10:00:00.123456Z","0199a2b4-0000-7000-8000-000000000001"]`; err != nil || string(b) != want {
		t.Errorf("json.Marshal() = %s, %v; want %s", b, err, want)
	}
	cursor, err := shared.EncodeCursor(c)
	if err != nil {
		t.Fatal(err)
	}
	var got APITokenCursor
	if err := shared.DecodeCursor(cursor, &got); err != nil || !got.CreatedAt.Equal(c.CreatedAt) || got.ID != c.ID {
		t.Errorf("DecodeCursor() = %+v, %v; want %+v", got, err, c)
	}
}

func TestAPITokenCursorRejects(t *testing.T) {
	for _, payload := range []string{
		`null`,
		`{"created_at":"2026-09-25T10:00:00Z","id":"0199a2b4-0000-7000-8000-000000000001"}`,
		`[]`,
		`["2026-09-25T10:00:00Z"]`,
		`["2026-09-25T10:00:00Z","0199a2b4-0000-7000-8000-000000000001","x"]`,
		`[1,"0199a2b4-0000-7000-8000-000000000001"]`,
		`["2026-09-25 10:00:00","0199a2b4-0000-7000-8000-000000000001"]`,
		`["2026-09-25T10:00:00Z","not-a-uuid"]`,
	} {
		var c APITokenCursor
		if err := json.Unmarshal([]byte(payload), &c); err == nil {
			t.Errorf("json.Unmarshal(%s) = %+v, want an error", payload, c)
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
git add server/internal/shared server/internal/modules/identity/domain
```
```bash
git commit -m "feat(M2/P3a): the page cursor envelope; personal access tokens, token specs and page sizes

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** `TestDecodeCursorRejects`、`TestParsePATRejects`、`TestCheckAPITokenReportsEveryField`、`TestPageSize` 通过；`make test` 通过。

---

### Task 5: `api_tokens` 表、查询与存储

**Files:**
- Create: `server/migrations/sql/00004_identity_api_tokens.sql`
- Modify: `server/migrations/schema_test.go`、`server/sqlc.yaml`
- Create: `server/internal/modules/identity/adapter/postgres/queries/api_tokens.sql`、`api_tokens.go`、`api_tokens_test.go`
- Modify: `server/internal/modules/identity/app/ports.go`（过渡：Task 8、10、11 加上别的端口）
- Generate: `server/internal/modules/identity/adapter/postgres/gen/api_tokens.sql.go`、`models.go`

**Interfaces:**
- Consumes: `domain.APIToken`、`domain.APITokenCursor`（Task 4）。
- Produces（spec 2.8，M2 设计 4.4、3.5、3.13、3.14）：
  - 表 `api_tokens`：Plane 17 列留 12 列；`token_hash bytea UNIQUE`（32 字节）；`label` 非空；`description` 默认 `''`；`created_by_id`、`updated_by_id` `ON DELETE SET NULL`；`user_id` `ON DELETE CASCADE`；列表的部分索引 `(user_id, created_at DESC, id DESC) WHERE deleted_at IS NULL`；
  - 端口（`app/ports.go`）：`NewAPIToken`、`APITokenCreator`、`APITokenLister`、`APITokenRevoker`、`APITokenCredential{ID, UserID, ExpiredAt, LastUsed, Revoked, UserActive}`、`APITokenReader{APITokenByHash; APITokenByID}`、`APITokenToucher`；
  - `Store` 的方法：`CreateAPIToken`、`ListAPITokens`（`after` 为 `nil` 时从头开始）、`RevokeAPIToken`（命中一行时 true）、`APITokenByHash`、`APITokenByID`（`ErrNoRows` → `app.ErrNotFound`）、`TouchAPIToken`；
  - 查询：列表的行比较 `(created_at, id) < (…::timestamptz, …::uuid)` 两个参数都显式转换（M2 设计 3.14）；撤销是软删除，只命中本账户未撤销的令牌；`TouchAPIToken` 只在 `last_used` 为空或早于 `stale_before` 时写，**不改** `updated_at`（Plane 的 `save(update_fields=["last_used"])`）。

**Tests:**
- `schema_test.go`：四个迁移都能 up、down、再 up；约束和索引的清单加上 `api_tokens` 的 10 项；合法的写入加上一行令牌；两个 CHECK 的反例（`token_hash` 不是 32 字节、空标签）。
- `api_tokens_test.go`（真实数据库）：`TestCreateAPIToken`（全部列、审计列等于固定时钟、创建者和更新者是账户本身）；`TestListAPITokensPageByPage`（5 个令牌，其中 3 个同一时刻，每页 2 个：翻完不重不漏，顺序与独立算出的 `created_at DESC, id DESC` 相同；已撤销和别的账户的令牌不列出）；`TestListAPITokensReadsEveryField`；`TestRevokeAPIToken`（别的账户、不存在、第一次、第二次，只有第一次命中；`deleted_at`、`updated_at`、`updated_by_id`）；`TestAPITokenCredentials`（按哈希、按 id；不存在是 `app.ErrNotFound`；撤销和停用之后的读数）；`TestTouchAPIToken`（第一次写、一分钟内不写、过了一分钟再写；`updated_at` 不变）。

- [ ] **Step 1: 迁移**

`server/migrations/sql/00004_identity_api_tokens.sql`（新文件）：

```sql
-- api_tokens：个人访问令牌（PAT）。Plane 的 api_tokens 表（17 列）按 M2 设计 4.4 保留 12 列；
-- 原来存明文的 token 列改为只存整个令牌的 SHA-256（3.4）。

-- +goose Up
CREATE TABLE api_tokens (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    -- 整个令牌（nrv_pat_ 加 43 个字符）的 SHA-256；令牌原文只在创建时返回一次
    token_hash bytea NOT NULL UNIQUE CHECK (octet_length(token_hash) = 32),
    -- 不传时由应用生成 32 位十六进制（Plane 的 uuid4().hex）
    label varchar(255) NOT NULL CHECK (label <> ''),
    description text NOT NULL DEFAULT '',
    -- 为空表示永不过期；创建时必须晚于当前时刻（领域层校验）
    expired_at timestamptz,
    -- 最多每分钟写一次（3.5）
    last_used timestamptz,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    -- 撤销就是软删除
    deleted_at timestamptz
);
-- 列表：未撤销的令牌按创建时间倒序，同一时刻按 id 倒序（5.2）
CREATE INDEX api_tokens_user_id_created_at_idx ON api_tokens (user_id, created_at DESC, id DESC) WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE api_tokens;
```

`server/migrations/schema_test.go`（对 `605f367` 的差异）：

```diff
--- a/server/migrations/schema_test.go
+++ b/server/migrations/schema_test.go
@@ -57,10 +57,10 @@
 	t.Cleanup(func() { _ = m.Close() })
 
 	up, err := m.Up(ctx)
-	if err != nil || len(up) != 3 {
-		t.Fatalf("Up() = %d migrations, %v; want 3", len(up), err)
+	if err != nil || len(up) != 4 {
+		t.Fatalf("Up() = %d migrations, %v; want 4", len(up), err)
 	}
-	if got, want := tables(t, pool), []string{"auth_sessions", "profiles", "users"}; !slices.Equal(got, want) {
+	if got, want := tables(t, pool), []string{"api_tokens", "auth_sessions", "profiles", "users"}; !slices.Equal(got, want) {
 		t.Errorf("tables after Up = %q, want %q", got, want)
 	}
 	for range up {
@@ -71,8 +71,8 @@
 	if got := tables(t, pool); len(got) != 0 {
 		t.Errorf("tables after every Down = %q, want none", got)
 	}
-	if again, err := m.Up(ctx); err != nil || len(again) != 3 {
-		t.Errorf("Up() again = %d migrations, %v; want 3", len(again), err)
+	if again, err := m.Up(ctx); err != nil || len(again) != 4 {
+		t.Errorf("Up() again = %d migrations, %v; want 4", len(again), err)
 	}
 }
 
@@ -82,10 +82,10 @@
 	pool := newPool(t, pgtest.NewDatabase(t))
 	rows, err := pool.Query(context.Background(), `
 		SELECT conname || ' ' || contype::text || CASE WHEN contype = 'f' THEN ' ' || confdeltype::text ELSE '' END FROM pg_constraint
-		WHERE conrelid IN ('users'::regclass, 'profiles'::regclass, 'auth_sessions'::regclass)
+		WHERE conrelid IN ('users'::regclass, 'profiles'::regclass, 'auth_sessions'::regclass, 'api_tokens'::regclass)
 			AND contype <> 'n' -- PG 18 lists NOT NULL as constraints too
 		UNION ALL
-		SELECT indexname || ' i' FROM pg_indexes WHERE tablename IN ('users', 'profiles', 'auth_sessions')
+		SELECT indexname || ' i' FROM pg_indexes WHERE tablename IN ('users', 'profiles', 'auth_sessions', 'api_tokens')
 		ORDER BY 1`)
 	if err != nil {
 		t.Fatal(err)
@@ -98,8 +98,19 @@
 		}
 		got = append(got, s)
 	}
-	// contype: p primary key, u unique, f foreign key (confdeltype c: ON DELETE CASCADE), c check.
+	// contype: p primary key, u unique, f foreign key (confdeltype c: ON
+	// DELETE CASCADE, n: ON DELETE SET NULL), c check.
 	want := []string{
+		"api_tokens_created_by_id_fkey f n",
+		"api_tokens_label_check c",
+		"api_tokens_pkey i",
+		"api_tokens_pkey p",
+		"api_tokens_token_hash_check c",
+		"api_tokens_token_hash_key i",
+		"api_tokens_token_hash_key u",
+		"api_tokens_updated_by_id_fkey f n",
+		"api_tokens_user_id_created_at_idx i",
+		"api_tokens_user_id_fkey f c",
 		"auth_sessions_expires_at_idx i",
 		"auth_sessions_generation_check c",
 		"auth_sessions_pkey i",
@@ -143,6 +154,7 @@
 		`UPDATE profiles SET onboarding_step = onboarding_step || '{"profile_complete": true}'`,
 		"UPDATE profiles SET onboarding_step = onboarding_step || '{}'",
 		"UPDATE auth_sessions SET revoked_at = now(), revoke_reason = 'logout'",
+		"INSERT INTO api_tokens (id, user_id, token_hash, label) VALUES ('0199a2b4-0000-7000-8000-000000000004', " + user + ", sha256('t'), 'x')",
 	} {
 		if _, err := pool.Exec(ctx, stmt); err != nil {
 			t.Fatalf("%s: %v", stmt, err)
@@ -177,6 +189,8 @@
 		{"unknown revoke reason", "UPDATE auth_sessions SET revoke_reason = 'expired'", "auth_sessions_revoke_reason_check"},
 		{"revoked without a reason", "UPDATE auth_sessions SET revoke_reason = NULL", "auth_sessions_revoked_consistent_check"},
 		{"a reason without revoked_at", "UPDATE auth_sessions SET revoked_at = NULL", "auth_sessions_revoked_consistent_check"},
+		{"token hash of a PAT not 32 bytes", "UPDATE api_tokens SET token_hash = '\\x00'", "api_tokens_token_hash_check"},
+		{"empty label", "UPDATE api_tokens SET label = ''", "api_tokens_label_check"},
 	}
 	for _, tt := range tests {
 		t.Run(tt.name, func(t *testing.T) {
```

`server/sqlc.yaml`（对 `605f367` 的差异）：

```diff
--- a/server/sqlc.yaml
+++ b/server/sqlc.yaml
@@ -9,6 +9,7 @@
       - migrations/sql/00001_identity_users.sql
       - migrations/sql/00002_identity_profiles.sql
       - migrations/sql/00003_identity_auth_sessions.sql
+      - migrations/sql/00004_identity_api_tokens.sql
     queries: internal/modules/identity/adapter/postgres/queries
     gen:
       go:
```

- [ ] **Step 2: 查询**

`server/internal/modules/identity/adapter/postgres/queries/api_tokens.sql`（新文件）：

```sql
-- name: CreateAPIToken :exec
-- An account creates its own tokens: it is their creator and last updater (M2 design 4.4).
INSERT INTO api_tokens (id, user_id, token_hash, label, description, expired_at, created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(token_hash), sqlc.arg(label), sqlc.arg(description), sqlc.arg(expired_at),
        sqlc.arg(user_id), sqlc.arg(user_id), sqlc.arg(now), sqlc.arg(now));

-- name: ListAPITokens :many
-- One page of an account's unrevoked tokens, newest first and then by id (M2 design 5.2): from the start, or after
-- the cursor's row. The row comparison casts both parameters, or sqlc types the id as a time (M2 design 3.14).
SELECT id, label, description, expired_at, last_used, created_at
FROM api_tokens
WHERE user_id = sqlc.arg(user_id) AND deleted_at IS NULL
  AND (sqlc.narg(cursor_created_at)::timestamptz IS NULL
       OR (created_at, id) < (sqlc.narg(cursor_created_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(row_limit);

-- name: RevokeAPIToken :execrows
-- Revoking is a soft delete. Another account's token, or one revoked already, is not hit (M2 design 5.4).
UPDATE api_tokens
SET updated_at = sqlc.arg(now), deleted_at = sqlc.arg(now), updated_by_id = sqlc.arg(user_id)::uuid
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id)::uuid AND deleted_at IS NULL;

-- name: GetAPITokenByHash :one
-- What authentication checks of a personal access token (M2 design 3.5); the use case judges it against its clock.
SELECT t.id, t.user_id, t.expired_at, t.last_used, t.deleted_at, u.is_active AS user_active
FROM api_tokens t
JOIN users u ON u.id = t.user_id
WHERE t.token_hash = sqlc.arg(token_hash);

-- name: GetAPITokenByID :one
-- The same, of the token a request authenticated with: the credential lock checks it again (M2 design 3.5).
SELECT t.id, t.user_id, t.expired_at, t.last_used, t.deleted_at, u.is_active AS user_active
FROM api_tokens t
JOIN users u ON u.id = t.user_id
WHERE t.id = sqlc.arg(id);

-- name: TouchAPIToken :exec
-- last_used, written at most once a minute (M2 design 3.5): only when it is older than stale_before. Using a token
-- changes nothing of it, so updated_at stays, as in Plane (save(update_fields=["last_used"])).
UPDATE api_tokens
SET last_used = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id) AND (last_used IS NULL OR last_used < sqlc.arg(stale_before)::timestamptz);
```

- [ ] **Step 3: 生成**

Run: `make gen-go`
Expected: 新增 `api_tokens.sql.go`，`models.go` 多出 `ApiToken`，其余生成物不变：

| 生成的文件 | SHA-256 | 行数 |
|---|---|---|
| `server/internal/modules/identity/adapter/postgres/gen/api_tokens.sql.go` | `c486b4bd1449273fea8a5a8058e6ff888b408697aaef85a2035b62219bd4fd8a` | 205 |
| `server/internal/modules/identity/adapter/postgres/gen/models.go` | `31c80a41503ce5ab8725391c4a46cfd028a75254afed674325162186414fa02b` | 69 |

- [ ] **Step 4: 端口和存储**

`server/internal/modules/identity/app/ports.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/modules/identity/app/ports.go
+++ b/server/internal/modules/identity/app/ports.go
@@ -148,6 +148,64 @@
 	EndSession(ctx context.Context, g SessionGeneration) (bool, error)
 }
 
+// NewAPIToken is a personal access token to insert. The account creates it
+// for itself, at Now.
+type NewAPIToken struct {
+	ID          uuid.UUID
+	UserID      uuid.UUID
+	TokenHash   []byte // domain.PAT.Hash
+	Label       string
+	Description string
+	ExpiredAt   *time.Time
+	Now         time.Time
+}
+
+// APITokenCreator inserts personal access tokens.
+type APITokenCreator interface {
+	CreateAPIToken(ctx context.Context, t NewAPIToken) error
+}
+
+// APITokenLister reads an account's tokens, a page at a time.
+type APITokenLister interface {
+	// ListAPITokens returns up to limit of userID's unrevoked tokens,
+	// newest first and then by id: from the start when after is nil,
+	// otherwise those after its row.
+	ListAPITokens(ctx context.Context, userID uuid.UUID, after *domain.APITokenCursor, limit int) ([]domain.APIToken, error)
+}
+
+// APITokenRevoker revokes personal access tokens.
+type APITokenRevoker interface {
+	// RevokeAPIToken revokes token id of userID at now; false when userID
+	// has no such unrevoked token.
+	RevokeAPIToken(ctx context.Context, id, userID uuid.UUID, now time.Time) (bool, error)
+}
+
+// APITokenCredential is what authentication and the credential lock check
+// of a personal access token.
+type APITokenCredential struct {
+	ID         uuid.UUID
+	UserID     uuid.UUID
+	ExpiredAt  *time.Time // nil: never expires
+	LastUsed   *time.Time
+	Revoked    bool
+	UserActive bool
+}
+
+// APITokenReader reads what authentication and the credential lock need.
+type APITokenReader interface {
+	// APITokenByHash returns ErrNotFound when no token has hash.
+	APITokenByHash(ctx context.Context, hash []byte) (APITokenCredential, error)
+	// APITokenByID returns ErrNotFound when there is no token id.
+	APITokenByID(ctx context.Context, id uuid.UUID) (APITokenCredential, error)
+}
+
+// APITokenToucher records that a token was used.
+type APITokenToucher interface {
+	// TouchAPIToken sets token id's last_used to now when it is unset or
+	// older than staleBefore.
+	TouchAPIToken(ctx context.Context, id uuid.UUID, now, staleBefore time.Time) error
+}
+
 // PasswordHasher hashes and verifies passwords with argon2id. Both return a
 // *shared.Error of 503 server_busy when no slot frees up within the wait
 // limit (M2 design 3.8).
```

`server/internal/modules/identity/adapter/postgres/api_tokens.go`（新文件）：

```go
package postgresadapter

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// CreateAPIToken inserts t. The domain validated every value, so a CHECK
// violation is a bug: an internal error (500).
func (s *Store) CreateAPIToken(ctx context.Context, t app.NewAPIToken) error {
	err := s.queries(ctx).CreateAPIToken(ctx, gen.CreateAPITokenParams{
		ID: t.ID, UserID: t.UserID, TokenHash: t.TokenHash, Label: t.Label, Description: t.Description, ExpiredAt: t.ExpiredAt, Now: t.Now,
	})
	if err != nil {
		return fmt.Errorf("create API token: %w", err)
	}
	return nil
}

// ListAPITokens returns up to limit of userID's unrevoked tokens, newest
// first and then by id, after the row of after when it is not nil.
func (s *Store) ListAPITokens(ctx context.Context, userID uuid.UUID, after *domain.APITokenCursor, limit int) ([]domain.APIToken, error) {
	p := gen.ListAPITokensParams{UserID: userID, RowLimit: int32(limit)} // at most MaxPageSize + 1
	if after != nil {
		p.CursorCreatedAt, p.CursorID = &after.CreatedAt, &after.ID
	}
	rows, err := s.queries(ctx).ListAPITokens(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("list API tokens: %w", err)
	}
	tokens := make([]domain.APIToken, len(rows))
	for i, r := range rows {
		tokens[i] = domain.APIToken{
			ID: r.ID, Label: r.Label, Description: r.Description, ExpiredAt: r.ExpiredAt, LastUsed: r.LastUsed, CreatedAt: r.CreatedAt,
		}
	}
	return tokens, nil
}

// RevokeAPIToken soft-deletes token id of userID; false when userID has no
// such unrevoked token.
func (s *Store) RevokeAPIToken(ctx context.Context, id, userID uuid.UUID, now time.Time) (bool, error) {
	n, err := s.queries(ctx).RevokeAPIToken(ctx, gen.RevokeAPITokenParams{Now: now, UserID: userID, ID: id})
	if err != nil {
		return false, fmt.Errorf("revoke API token: %w", err)
	}
	return n == 1, nil
}

// APITokenByHash reads what authentication checks of the token with hash;
// app.ErrNotFound when there is none.
func (s *Store) APITokenByHash(ctx context.Context, hash []byte) (app.APITokenCredential, error) {
	r, err := s.queries(ctx).GetAPITokenByHash(ctx, hash)
	if err != nil {
		return app.APITokenCredential{}, notFound(err)
	}
	return app.APITokenCredential{
		ID: r.ID, UserID: r.UserID, ExpiredAt: r.ExpiredAt, LastUsed: r.LastUsed, Revoked: r.DeletedAt != nil, UserActive: r.UserActive,
	}, nil
}

// APITokenByID reads the same of token id; app.ErrNotFound when there is
// none.
func (s *Store) APITokenByID(ctx context.Context, id uuid.UUID) (app.APITokenCredential, error) {
	r, err := s.queries(ctx).GetAPITokenByID(ctx, id)
	if err != nil {
		return app.APITokenCredential{}, notFound(err)
	}
	return app.APITokenCredential{
		ID: r.ID, UserID: r.UserID, ExpiredAt: r.ExpiredAt, LastUsed: r.LastUsed, Revoked: r.DeletedAt != nil, UserActive: r.UserActive,
	}, nil
}

// TouchAPIToken sets token id's last_used to now when it is unset or older
// than staleBefore.
func (s *Store) TouchAPIToken(ctx context.Context, id uuid.UUID, now, staleBefore time.Time) error {
	if err := s.queries(ctx).TouchAPIToken(ctx, gen.TouchAPITokenParams{Now: now, ID: id, StaleBefore: staleBefore}); err != nil {
		return fmt.Errorf("touch API token: %w", err)
	}
	return nil
}
```

`server/internal/modules/identity/adapter/postgres/api_tokens_test.go`（新文件）：

```go
package postgresadapter_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// newToken is a token of userID made at when, whose hash comes from name.
func newToken(userID uuid.UUID, name string, when time.Time) app.NewAPIToken {
	hash := sha256.Sum256([]byte(name))
	return app.NewAPIToken{ID: uuid.NewV7(), UserID: userID, TokenHash: hash[:], Label: name, Now: when}
}

func mustCreateToken(t *testing.T, s *postgresadapter.Store, n app.NewAPIToken) {
	t.Helper()
	if err := s.CreateAPIToken(context.Background(), n); err != nil {
		t.Fatal(err)
	}
}

func TestCreateAPIToken(t *testing.T) {
	s, pool := newStore(t)
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)
	expires := now.Add(7 * 24 * time.Hour)
	n := newToken(u.ID, "deploy", now)
	n.Description, n.ExpiredAt = "ci", &expires

	mustCreateToken(t, s, n)

	var hash []byte
	var label, description string
	var expired, lastUsed, deleted *time.Time
	var createdBy, updatedBy *uuid.UUID
	var created, updated time.Time
	err := pool.QueryRow(context.Background(), `SELECT token_hash, label, description, expired_at, last_used, deleted_at,
		created_by_id, updated_by_id, created_at, updated_at FROM api_tokens WHERE id = $1 AND user_id = $2`, n.ID, u.ID).
		Scan(&hash, &label, &description, &expired, &lastUsed, &deleted, &createdBy, &updatedBy, &created, &updated)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(hash, n.TokenHash) || label != "deploy" || description != "ci" || expired == nil || !expired.Equal(expires) ||
		lastUsed != nil || deleted != nil {
		t.Errorf("row = % x %q %q expired %v last used %v deleted %v", hash, label, description, expired, lastUsed, deleted)
	}
	// The account created the token for itself; the audit columns are the
	// use case's time (M2 design 3.13).
	if createdBy == nil || *createdBy != u.ID || updatedBy == nil || *updatedBy != u.ID || !created.Equal(now) || !updated.Equal(now) {
		t.Errorf("audit = created by %v at %v, updated by %v at %v; want the account at %v", createdBy, created, updatedBy, updated, now)
	}
}

// Pages follow created_at, newest first, then id: rows made at the same
// instant are neither repeated nor skipped across a page boundary. Revoked
// tokens and other accounts' tokens are not listed.
func TestListAPITokensPageByPage(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newUser("alice@corp.com"), newUser("bob@corp.com")
	mustCreate(t, s, alice)
	mustCreate(t, s, bob)
	// Three tokens share an instant, so a page of two ends inside them.
	for i, at := range []time.Time{now, now.Add(time.Second), now.Add(time.Second), now.Add(time.Second), now.Add(2 * time.Second)} {
		mustCreateToken(t, s, newToken(alice.ID, "t"+string(rune('a'+i)), at))
	}
	mustCreateToken(t, s, newToken(bob.ID, "bob's", now.Add(time.Hour)))
	revoked := newToken(alice.ID, "revoked", now.Add(time.Minute))
	mustCreateToken(t, s, revoked)
	if _, err := s.RevokeAPIToken(context.Background(), revoked.ID, alice.ID, now); err != nil {
		t.Fatal(err)
	}

	var got []uuid.UUID
	var after *domain.APITokenCursor
	for range 4 {
		page, err := s.ListAPITokens(context.Background(), alice.ID, after, 2)
		if err != nil {
			t.Fatal(err)
		}
		if len(page) == 0 {
			break
		}
		for _, tok := range page {
			got = append(got, tok.ID)
		}
		last := page[len(page)-1]
		after = &domain.APITokenCursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}
	if want := expectedOrder(t, pool, alice.ID); !slices.Equal(got, want) || len(got) != 5 {
		t.Errorf("pages = %v, want %v", got, want)
	}
}

// expectedOrder is the order the list promises, computed apart from the
// query under test: created_at descending, then id descending.
func expectedOrder(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID) []uuid.UUID {
	t.Helper()
	rows, err := pool.Query(context.Background(), "SELECT id, created_at FROM api_tokens WHERE user_id = $1 AND deleted_at IS NULL", userID)
	if err != nil {
		t.Fatal(err)
	}
	type row struct {
		id uuid.UUID
		at time.Time
	}
	var all []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.at); err != nil {
			t.Fatal(err)
		}
		all = append(all, r)
	}
	slices.SortFunc(all, func(a, b row) int {
		if c := b.at.Compare(a.at); c != 0 {
			return c
		}
		return bytes.Compare(b.id[:], a.id[:])
	})
	ids := make([]uuid.UUID, len(all))
	for i, r := range all {
		ids[i] = r.id
	}
	return ids
}

func TestListAPITokensReadsEveryField(t *testing.T) {
	s, _ := newStore(t)
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)
	expires := now.Add(time.Hour)
	n := newToken(u.ID, "deploy", now)
	n.Description, n.ExpiredAt = "ci", &expires
	mustCreateToken(t, s, n)
	used := now.Add(time.Minute)
	if err := s.TouchAPIToken(context.Background(), n.ID, used, used.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}

	got, err := s.ListAPITokens(context.Background(), u.ID, nil, 10)

	if err != nil || len(got) != 1 {
		t.Fatalf("ListAPITokens() = %+v, %v; want one token", got, err)
	}
	tok := got[0]
	if tok.ID != n.ID || tok.Label != "deploy" || tok.Description != "ci" || tok.ExpiredAt == nil || !tok.ExpiredAt.Equal(expires) ||
		tok.LastUsed == nil || !tok.LastUsed.Equal(used) || !tok.CreatedAt.Equal(now) {
		t.Errorf("ListAPITokens() = %+v", tok)
	}
}

func TestRevokeAPIToken(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newUser("alice@corp.com"), newUser("bob@corp.com")
	mustCreate(t, s, alice)
	mustCreate(t, s, bob)
	n := newToken(alice.ID, "deploy", now)
	mustCreateToken(t, s, n)
	later := now.Add(time.Hour)

	byBob, err1 := s.RevokeAPIToken(context.Background(), n.ID, bob.ID, later)
	unknown, err2 := s.RevokeAPIToken(context.Background(), uuid.NewV7(), alice.ID, later)
	first, err3 := s.RevokeAPIToken(context.Background(), n.ID, alice.ID, later)
	again, err4 := s.RevokeAPIToken(context.Background(), n.ID, alice.ID, later.Add(time.Hour))

	if err := errors.Join(err1, err2, err3, err4); err != nil || byBob || unknown || !first || again {
		t.Errorf("revoke by another account %v, unknown %v, first %v, again %v (%v); want only the first", byBob, unknown, first, again, err)
	}
	var deleted, updated time.Time
	var updatedBy uuid.UUID
	if err := pool.QueryRow(context.Background(), "SELECT deleted_at, updated_at, updated_by_id FROM api_tokens WHERE id = $1", n.ID).
		Scan(&deleted, &updated, &updatedBy); err != nil {
		t.Fatal(err)
	}
	if !deleted.Equal(later) || !updated.Equal(later) || updatedBy != alice.ID {
		t.Errorf("row = deleted %v updated %v by %v; want both at %v by the account", deleted, updated, updatedBy, later)
	}
}

func TestAPITokenCredentials(t *testing.T) {
	s, pool := newStore(t)
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)
	expires := now.Add(time.Hour)
	n := newToken(u.ID, "deploy", now)
	n.ExpiredAt = &expires
	mustCreateToken(t, s, n)

	byHash, err1 := s.APITokenByHash(context.Background(), n.TokenHash)
	byID, err2 := s.APITokenByID(context.Background(), n.ID)

	want := app.APITokenCredential{ID: n.ID, UserID: u.ID, ExpiredAt: &expires, UserActive: true}
	for _, got := range []app.APITokenCredential{byHash, byID} {
		if got.ID != want.ID || got.UserID != want.UserID || got.ExpiredAt == nil || !got.ExpiredAt.Equal(expires) ||
			got.LastUsed != nil || got.Revoked || !got.UserActive {
			t.Errorf("credential = %+v, want %+v", got, want)
		}
	}
	if err := errors.Join(err1, err2); err != nil {
		t.Fatal(err)
	}
	other := sha256.Sum256([]byte("other"))
	if _, err := s.APITokenByHash(context.Background(), other[:]); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("APITokenByHash(unknown) = %v, want app.ErrNotFound", err)
	}
	if _, err := s.APITokenByID(context.Background(), uuid.NewV7()); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("APITokenByID(unknown) = %v, want app.ErrNotFound", err)
	}

	if _, err := s.RevokeAPIToken(context.Background(), n.ID, u.ID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), "UPDATE users SET is_active = false"); err != nil {
		t.Fatal(err)
	}
	byHash, err1 = s.APITokenByHash(context.Background(), n.TokenHash)
	byID, err2 = s.APITokenByID(context.Background(), n.ID)
	if err := errors.Join(err1, err2); err != nil || !byHash.Revoked || byHash.UserActive || !byID.Revoked || byID.UserActive {
		t.Errorf("after revoke and deactivate: by hash %+v, by id %+v, %v", byHash, byID, err)
	}
}

// last_used is written at most once a minute, and using a token changes
// nothing else of it (M2 design 3.5).
func TestTouchAPIToken(t *testing.T) {
	s, pool := newStore(t)
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)
	n := newToken(u.ID, "deploy", now)
	mustCreateToken(t, s, n)
	lastUsed := func() time.Time {
		t.Helper()
		var used *time.Time
		var updated time.Time
		if err := pool.QueryRow(context.Background(), "SELECT last_used, updated_at FROM api_tokens WHERE id = $1", n.ID).Scan(&used, &updated); err != nil {
			t.Fatal(err)
		}
		if !updated.Equal(now) {
			t.Errorf("updated_at = %v, want it unchanged at %v", updated, now)
		}
		if used == nil {
			return time.Time{}
		}
		return *used
	}
	first := now.Add(time.Minute)
	touch := func(at time.Time) {
		t.Helper()
		if err := s.TouchAPIToken(context.Background(), n.ID, at, at.Add(-time.Minute)); err != nil {
			t.Fatal(err)
		}
	}

	touch(first)
	if got := lastUsed(); !got.Equal(first) {
		t.Errorf("last_used after the first use = %v, want %v", got, first)
	}
	touch(first.Add(59 * time.Second))
	if got := lastUsed(); !got.Equal(first) {
		t.Errorf("last_used after a use within the minute = %v, want it still %v", got, first)
	}
	touch(first.Add(61 * time.Second))
	if got := lastUsed(); !got.Equal(first.Add(61 * time.Second)) {
		t.Errorf("last_used after the minute = %v, want %v", got, first.Add(61*time.Second))
	}
}
```

- [ ] **Step 5: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。archtest 的 `TestSQLCSchemaScope` 认可新的迁移归 identity。

- [ ] **Step 6: 提交**

```bash
git add server/migrations server/sqlc.yaml server/internal/modules/identity/adapter/postgres server/internal/modules/identity/app/ports.go
```
```bash
git commit -m "feat(M2/P3a): the api_tokens table, its queries and the store

A token is stored as the SHA-256 of the whole token; revoking is a soft
delete; last_used is written at most once a minute and leaves updated_at
as it is.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run: `make gen-check`
Expected: 没有差异。

**Done when:** 四个迁移 up、down、再 up；`TestListAPITokensPageByPage` 翻页不重不漏；`TestTouchAPIToken` 证明每分钟最多写一次；两个生成物与上表相同；`make gen-check` 干净。

---

### Task 6: PAT 认证、账户行锁的复核、令牌的三个用例

**Files:**
- Modify: `server/internal/shared/actor.go`
- Modify: `server/internal/modules/identity/app/authenticate.go`、`authenticate_test.go`、`register_test.go`
- Create: `server/internal/modules/identity/app/authenticate_pat_test.go`、`credential_lock.go`
- Create: `server/internal/modules/identity/app/create_api_token.go`、`list_api_tokens.go`、`revoke_api_token.go` 及三个测试
- Modify: `server/internal/modules/identity/app/fakes_test.go`（过渡：Task 9、10、11 加上别的假仓储）
- Modify: `server/internal/modules/identity/domain/errors.go`（过渡：Task 10 加上 `current_password_incorrect`）
- Modify: `server/internal/modules/identity/adapter/authn/authenticator.go`、`authenticator_test.go`
- Modify: `server/internal/modules/identity/module.go`（过渡：Task 7 接上令牌的三个用例）

**Interfaces:**
- Consumes: 端口（Task 5）；`domain.ParsePAT`、`CheckAPIToken`、`PageSize`、`APITokenCursor`（Task 4）；`shared.EncodeCursor`、`DecodeCursor`（Task 4）。
- Produces（spec 2.9、2.10，M2 设计 3.5、3.10、4.6、5.4）：
  - `shared.Actor.APITokenID`：`SessionID` 和 `APITokenID` 恰好一个有值；
  - `app.NewAuthenticate(AuthenticateDeps{AccessTokens, Sessions, APITokens, Touch, Clock})`：`nrv_pat_` 开头的令牌按哈希查一次（`ParsePAT` 失败不查库）；撤销、到期（`now >= expired_at`）、账户停用都是 401；`last_used` 为空或早于一分钟前才写（`Touch`）；读或写失败是内部错误，不是 401；
  - `sessionInvalid`、`tokenInvalid`：认证和锁共用的判定；
  - `app.CredentialLock{Locker, Sessions, APITokens}.Lock(ctx, actor, now) (LockedAccount, error)`：在事务里先 `LockForCredentials`，再在锁下复核调用者的凭证（会话或 PAT），不成立时 401（spec 2.9）；
  - `app.NewCreateAPIToken(CreateAPITokenDeps{Lock, Tokens, Tx, Clock, Logger})`：先 `CheckAPIToken`，再一个事务：`Lock` → 插入；没有标签时生成 32 位十六进制（Plane 的 `uuid4().hex`）；INFO "API token created"（`user_id`、`token_id`）；返回 `CreatedAPIToken{APIToken; Token}`；
  - `app.NewListAPITokens(tokens)`：先判游标（400），再判 `limit`（422），多读一行判断有没有下一页，`NextCursor` 是本页最后一行；
  - `app.NewRevokeAPIToken(tokens, clock, logger)`：一条语句；不存在、已撤销、别的账户的令牌都是 404 `identity.api_token_not_found`；请求所用的令牌可以撤销自己；INFO "API token revoked"；
  - `domain.ErrAPITokenNotFound`；
  - `authn`：PAT 的限流键是 `pat:<id>`，访问令牌仍是 `session:<id>`。

**Tests:**
- `authenticate_pat_test.go`：`TestAuthenticateAPAT`（不碰访问令牌和会话；actor 带 `APITokenID`）；`TestAuthenticateAPATTouchesAtMostOnceAMinute`（从未用过、59 秒前、恰好一分钟前、61 秒前）；`TestAuthenticateRejectsAPAT`（6 个：畸形（不查库）、不存在、已撤销、恰好到期、账户停用，都是 401 且不写 `last_used`；令牌属于别的账户时认证为那个账户）；`TestAuthenticateAPATDatabaseFailureIsNot401`（读、写各一次）。
- `create_api_token_test.go`：`TestCreateAPIToken`（调用顺序 `lock → session → insert`；插入的行逐字段核对；令牌只出现在返回值里，日志中没有令牌和哈希的任何写法）；`TestCreateAPITokenGeneratesALabel`；`TestCreateAPITokenWithAToken`（锁下复核的是令牌，不是会话）；`TestCreateAPITokenRechecksTheCredentialUnderTheLock`（9 个：账户不存在、停用；会话撤销、到期、不存在、属于别的账户；令牌撤销、到期、属于别的账户；都是 401，没有插入）；`TestCreateAPITokenLockFailureIsNot401`；`TestCreateAPITokenChecksTheSpecFirst`（不开事务）。
- `list_api_tokens_test.go`：`TestListAPITokensFirstPage`（默认 50，读 51 行）；`TestListAPITokensNextPage`（游标交给存储的是上一页最后一行）；`TestListAPITokensLastFullPage`（满页但没有下一页时没有游标）；`TestListAPITokensRejects`（4 个：不是本列表的游标 400、`limit` 0 和 101 是 422、两者都错时 400；都不问存储）。
- `revoke_api_token_test.go`：`TestRevokeAPIToken`、`TestRevokeAnAPITokenThatIsNotTheCallers`、`TestRevokeAPITokenDatabaseFailure`。
- `authenticator_test.go`：`TestAuthenticateKeysAPATByItsID`。
- `register_test.go`：`assertNoSecret` 另查大写的十六进制（P2 缺陷类别"日志里的密文漏掉某种写法"）。
- 假仓储（`fakes_test.go`）都核对参数（P2 缺陷类别"忽略参数的假实现"）：`fakeAPITokens` 只按它持有的哈希和 id 找到令牌，别的键都是 `app.ErrNotFound`；`fakeCredentials` 把账户 id 和会话 id 写进调用记录，测试逐条核对。

- [ ] **Step 1: 调用者的凭证种类**

`server/internal/shared/actor.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/shared/actor.go
+++ b/server/internal/shared/actor.go
@@ -10,8 +10,11 @@
 // tells which credential authenticated the request, never what kind of
 // account it is (v0 design 0.2, principle 1).
 type Actor struct {
-	UserID    uuid.UUID
-	SessionID uuid.UUID // the login session of the access token
+	UserID uuid.UUID
+	// The credential: the login session of an access token, or a personal
+	// access token. Exactly one of them is set.
+	SessionID  uuid.UUID
+	APITokenID uuid.UUID
 }
 
 type actorKey struct{}
```

- [ ] **Step 2: PAT 认证**

`server/internal/modules/identity/app/authenticate.go`（完整内容）：

```go
package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// lastUsedInterval is how often a personal access token's last_used is
// written at most (M2 design 3.5).
const lastUsedInterval = time.Minute

// AuthenticateDeps are Authenticate's collaborators.
type AuthenticateDeps struct {
	AccessTokens AccessTokens
	Sessions     SessionReader
	APITokens    APITokenReader
	Touch        APITokenToucher
	Clock        Clock
}

// Authenticate turns a bearer token into the request's actor (M2 design
// 3.4–3.6). A token that starts with nrv_pat_ is a personal access token:
// one query by its hash. Any other is an access token: its signature and
// expiry, then one primary-key query for the session and its account.
type Authenticate struct {
	d AuthenticateDeps
}

// NewAuthenticate returns the use case.
func NewAuthenticate(d AuthenticateDeps) *Authenticate {
	return &Authenticate{d: d}
}

// The reasons a credential fails. They go to the debug log only; the
// caller always sees 401 unauthorized.
var (
	errSessionUnknown  = errors.New("session does not exist")
	errSessionMismatch = errors.New("session belongs to another account")
	errSessionRevoked  = errors.New("session is revoked")
	errSessionExpired  = errors.New("session has expired")
	errPATMalformed    = errors.New("personal access token is malformed")
	errPATUnknown      = errors.New("personal access token does not exist")
	errPATMismatch     = errors.New("personal access token belongs to another account")
	errPATRevoked      = errors.New("personal access token is revoked")
	errPATExpired      = errors.New("personal access token has expired")
	errUserUnknown     = errors.New("account does not exist")
	errUserDeactivated = errors.New("account is deactivated")
)

// Execute returns the actor of token. An invalid credential is a
// *shared.Error of 401 that wraps the reason; an expired access token also
// matches ErrAccessTokenExpired. Any other error is an internal fault.
func (a *Authenticate) Execute(ctx context.Context, token string) (shared.Actor, error) {
	now := a.d.Clock.Now()
	if strings.HasPrefix(token, domain.PATPrefix) {
		return a.personal(ctx, token, now)
	}
	claims, err := a.d.AccessTokens.Verify(token, now)
	if err != nil {
		return shared.Actor{}, unauthenticated(err)
	}
	cred, err := a.d.Sessions.SessionCredential(ctx, claims.SessionID)
	if err := sessionInvalid(cred, err, claims.UserID, now); err != nil {
		return shared.Actor{}, err
	}
	return shared.Actor{UserID: claims.UserID, SessionID: claims.SessionID}, nil
}

// personal authenticates a personal access token, and records its use at
// most once a minute (M2 design 3.5).
func (a *Authenticate) personal(ctx context.Context, token string, now time.Time) (shared.Actor, error) {
	pat, ok := domain.ParsePAT(token)
	if !ok {
		return shared.Actor{}, unauthenticated(errPATMalformed)
	}
	cred, err := a.d.APITokens.APITokenByHash(ctx, pat.Hash())
	if err := tokenInvalid(cred, err, cred.UserID, now); err != nil {
		return shared.Actor{}, err
	}
	staleBefore := now.Add(-lastUsedInterval)
	if cred.LastUsed == nil || cred.LastUsed.Before(staleBefore) {
		if err := a.d.Touch.TouchAPIToken(ctx, cred.ID, now, staleBefore); err != nil {
			return shared.Actor{}, err
		}
	}
	return shared.Actor{UserID: cred.UserID, APITokenID: cred.ID}, nil
}

// sessionInvalid judges a session that the credential of userID names, as
// SessionReader read it with err, at now: nil when it is valid, a 401 with
// the reason when it is not, err itself when the read failed.
func sessionInvalid(cred SessionCredential, err error, userID uuid.UUID, now time.Time) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return unauthenticated(errSessionUnknown)
	case err != nil:
		return err
	case cred.UserID != userID:
		return unauthenticated(errSessionMismatch)
	case cred.Revoked:
		return unauthenticated(errSessionRevoked)
	case !now.Before(cred.ExpiresAt):
		return unauthenticated(errSessionExpired)
	case !cred.UserActive:
		return unauthenticated(errUserDeactivated)
	}
	return nil
}

// tokenInvalid judges a personal access token of userID the same way. A
// token expires at its expired_at, like a session.
func tokenInvalid(cred APITokenCredential, err error, userID uuid.UUID, now time.Time) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return unauthenticated(errPATUnknown)
	case err != nil:
		return err
	case cred.UserID != userID:
		return unauthenticated(errPATMismatch)
	case cred.Revoked:
		return unauthenticated(errPATRevoked)
	case cred.ExpiredAt != nil && !now.Before(*cred.ExpiredAt):
		return unauthenticated(errPATExpired)
	case !cred.UserActive:
		return unauthenticated(errUserDeactivated)
	}
	return nil
}

func unauthenticated(reason error) error {
	return fmt.Errorf("%w: %w", shared.Unauthenticated(), reason)
}
```

`server/internal/modules/identity/app/authenticate_test.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/modules/identity/app/authenticate_test.go
+++ b/server/internal/modules/identity/app/authenticate_test.go
@@ -21,7 +21,10 @@
 func newAuthenticate(cred app.SessionCredential, credErr error) (*app.Authenticate, *fakeTokens, *fakeStore) {
 	tokens := newFakeTokens()
 	store := &fakeStore{credential: cred, credErr: credErr}
-	return app.NewAuthenticate(tokens, store, clocktest.At(now)), tokens, store
+	uc := app.NewAuthenticate(app.AuthenticateDeps{
+		AccessTokens: tokens, Sessions: store, APITokens: &fakeAPITokens{log: &callLog{}}, Touch: &fakeAPITokens{}, Clock: clocktest.At(now),
+	})
+	return uc, tokens, store
 }
 
 func validCredential() app.SessionCredential {
```

`server/internal/modules/identity/app/authenticate_pat_test.go`（新文件）：

```go
package app_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var tokenID = uuid.MustParse("0199a2b4-0000-7000-8000-000000000003")

// samplePAT is a token whose bytes all differ.
func samplePAT() domain.PAT {
	var p domain.PAT
	for i := range p {
		p[i] = byte(i*5 + 1)
	}
	return p
}

// newPATAuthenticate authenticates samplePAT as cred; the access tokens and
// sessions it holds are there to show that a PAT never reaches them.
func newPATAuthenticate(cred app.APITokenCredential) (*app.Authenticate, *fakeAPITokens, *fakeTokens, *fakeStore) {
	tokens := &fakeAPITokens{log: &callLog{}, credential: cred, hash: samplePAT().Hash()}
	access, sessions := newFakeTokens(), &fakeStore{}
	uc := app.NewAuthenticate(app.AuthenticateDeps{AccessTokens: access, Sessions: sessions, APITokens: tokens, Touch: tokens, Clock: clocktest.At(now)})
	return uc, tokens, access, sessions
}

func validToken() app.APITokenCredential {
	expires := now.Add(time.Hour)
	return app.APITokenCredential{ID: tokenID, UserID: userID, ExpiredAt: &expires, UserActive: true}
}

func TestAuthenticateAPAT(t *testing.T) {
	uc, tokens, access, sessions := newPATAuthenticate(validToken())

	actor, err := uc.Execute(context.Background(), samplePAT().String())

	if err != nil || actor != (shared.Actor{UserID: userID, APITokenID: tokenID}) {
		t.Errorf("Execute() = %+v, %v; want the token's account and id", actor, err)
	}
	if len(tokens.log.calls) != 1 || len(access.verifiedAt) != 0 || len(sessions.credentials) != 0 {
		t.Errorf("lookups = %q, access tokens verified %d, sessions read %d; want one lookup by hash, no JWT, no session",
			tokens.log.calls, len(access.verifiedAt), len(sessions.credentials))
	}
	// Never used before: last_used is written, at the clock's time.
	if want := []touch{{tokenID, now, now.Add(-time.Minute)}}; len(tokens.touches) != 1 || tokens.touches[0] != want[0] {
		t.Errorf("touches = %+v, want %+v", tokens.touches, want)
	}
}

// last_used is written at most once a minute (M2 design 3.5): not when it
// is less than a minute old.
func TestAuthenticateAPATTouchesAtMostOnceAMinute(t *testing.T) {
	for _, tt := range []struct {
		name    string
		used    time.Duration // before now
		touched bool
	}{
		{"used 59 seconds ago", 59 * time.Second, false},
		{"used a minute ago", time.Minute, false},
		{"used 61 seconds ago", 61 * time.Second, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cred := validToken()
			used := now.Add(-tt.used)
			cred.LastUsed = &used
			uc, tokens, _, _ := newPATAuthenticate(cred)

			if _, err := uc.Execute(context.Background(), samplePAT().String()); err != nil {
				t.Fatal(err)
			}
			if got := len(tokens.touches) == 1; got != tt.touched {
				t.Errorf("touched = %v, want %v", got, tt.touched)
			}
		})
	}
}

func TestAuthenticateRejectsAPAT(t *testing.T) {
	other := uuid.MustParse("0199a2b4-0000-7000-8000-000000000009")
	tests := []struct {
		name   string
		token  string
		cred   func(*app.APITokenCredential)
		reason string
		looked bool // whether the token was looked up
	}{
		{"malformed", domain.PATPrefix + "AAAA", nil, "personal access token is malformed", false},
		{"unknown", func() string { p := samplePAT(); p[0]++; return p.String() }(), nil, "personal access token does not exist", true},
		{"revoked", "", func(c *app.APITokenCredential) { c.Revoked = true }, "personal access token is revoked", true},
		{"expiring now", "", func(c *app.APITokenCredential) { c.ExpiredAt = &now }, "personal access token has expired", true},
		{"of a deactivated account", "", func(c *app.APITokenCredential) { c.UserActive = false }, "account is deactivated", true},
		{"of another account", "", func(c *app.APITokenCredential) { c.UserID = other }, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cred := validToken()
			if tt.cred != nil {
				tt.cred(&cred)
			}
			uc, tokens, _, _ := newPATAuthenticate(cred)
			token := tt.token
			if token == "" {
				token = samplePAT().String()
			}

			actor, err := uc.Execute(context.Background(), token)

			if tt.reason == "" {
				// A token authenticates its own account, whichever that is.
				if err != nil || actor.UserID != other {
					t.Errorf("Execute() = %+v, %v; want the token's own account", actor, err)
				}
				return
			}
			var se *shared.Error
			if !errors.As(err, &se) || se.ProblemStatus() != 401 || err.Error() != "Authentication is required.: "+tt.reason {
				t.Errorf("Execute() = %v, want 401 because %s", err, tt.reason)
			}
			if looked := len(tokens.log.calls) > 0; looked != tt.looked || len(tokens.touches) != 0 {
				t.Errorf("looked up %v, touched %d; want looked up %v, never touched", looked, len(tokens.touches), tt.looked)
			}
		})
	}
}

// A database failure is an internal fault, not a 401: whether reading the
// token or writing last_used.
func TestAuthenticateAPATDatabaseFailureIsNot401(t *testing.T) {
	boom := errors.New("connection refused")
	for _, failing := range []string{"read", "touch"} {
		uc, tokens, _, _ := newPATAuthenticate(validToken())
		if failing == "read" {
			tokens.readErr = boom
		} else {
			tokens.touchErr = boom
		}

		_, err := uc.Execute(context.Background(), samplePAT().String())

		var se *shared.Error
		if !errors.Is(err, boom) || errors.As(err, &se) {
			t.Errorf("the %s fails: Execute() = %v, want the database error, not a problem", failing, err)
		}
	}
}
```

- [ ] **Step 3: 账户行锁的复核**

`server/internal/modules/identity/app/credential_lock.go`（新文件）：

```go
package app

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CredentialLock is the step that the account row lock protocol shares
// (M2 design 3.5): a transaction that issues or changes a credential with
// the caller's credential (creating a token, changing the password,
// deactivating) locks the account row first, then checks again, under the
// lock, that the caller's credential still holds. A concurrent reset that
// committed in between is seen here.
type CredentialLock struct {
	Locker    CredentialLocker
	Sessions  SessionReader
	APITokens APITokenReader
}

// Lock locks actor's account row until the transaction ends and returns
// it, once the account is active and actor's credential valid at now: its
// session unrevoked and unexpired, or its personal access token unrevoked
// and unexpired. Otherwise it is 401 unauthorized. Call it first inside the
// transaction.
func (c CredentialLock) Lock(ctx context.Context, actor shared.Actor, now time.Time) (LockedAccount, error) {
	account, err := c.Locker.LockForCredentials(ctx, actor.UserID)
	switch {
	case errors.Is(err, ErrNotFound):
		return LockedAccount{}, unauthenticated(errUserUnknown)
	case err != nil:
		return LockedAccount{}, err
	case !account.Active:
		return LockedAccount{}, unauthenticated(errUserDeactivated)
	}
	if actor.APITokenID != uuid.Nil() {
		cred, err := c.APITokens.APITokenByID(ctx, actor.APITokenID)
		if err := tokenInvalid(cred, err, actor.UserID, now); err != nil {
			return LockedAccount{}, err
		}
		return account, nil
	}
	cred, err := c.Sessions.SessionCredential(ctx, actor.SessionID)
	if err := sessionInvalid(cred, err, actor.UserID, now); err != nil {
		return LockedAccount{}, err
	}
	return account, nil
}
```

- [ ] **Step 4: 令牌的三个用例**

`server/internal/modules/identity/domain/errors.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/modules/identity/domain/errors.go
+++ b/server/internal/modules/identity/domain/errors.go
@@ -20,4 +20,8 @@
 	// ErrRefreshTokenInvalid answers every refresh that does not rotate:
 	// unknown, expired, revoked, reused or forged (M2 design 3.5).
 	ErrRefreshTokenInvalid = shared.NewError(shared.KindUnauthenticated, "identity.refresh_token_invalid", "The refresh token is not valid; sign in again.")
+	// ErrAPITokenNotFound answers a revocation of a token that does not
+	// exist, is revoked already or belongs to another account: what the
+	// caller cannot see is not found (v0 design 3.5).
+	ErrAPITokenNotFound = shared.NewError(shared.KindNotFound, "identity.api_token_not_found", "The API token does not exist.")
 )
```

`server/internal/modules/identity/app/create_api_token.go`（新文件）：

```go
package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CreateAPITokenDeps are CreateAPIToken's collaborators.
type CreateAPITokenDeps struct {
	Lock   CredentialLock
	Tokens APITokenCreator
	Tx     shared.TxManager
	Clock  Clock
	Logger *slog.Logger
}

// CreateAPIToken creates a personal access token for the caller:
// POST /api/v0/me/api-tokens. Any credential may, a token too (M2 design
// 4.6); no password is asked for, as in Plane (8.5).
type CreateAPIToken struct {
	d CreateAPITokenDeps
}

// NewCreateAPIToken returns the use case.
func NewCreateAPIToken(d CreateAPITokenDeps) *CreateAPIToken {
	return &CreateAPIToken{d: d}
}

// CreatedAPIToken is a new token and, this once, the token itself.
type CreatedAPIToken struct {
	domain.APIToken
	Token string
}

// Execute checks spec, then, in one transaction, locks the caller's account
// row, checks the caller's credential again and inserts the token (M2
// design 3.5): a token made with a credential that a concurrent reset
// revoked is never inserted. A spec without a label gets 32 hexadecimal
// digits, as Plane's uuid4().hex.
func (c *CreateAPIToken) Execute(ctx context.Context, spec domain.APITokenSpec) (CreatedAPIToken, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return CreatedAPIToken{}, err
	}
	now := c.d.Clock.Now()
	if err := domain.CheckAPIToken(spec, now); err != nil {
		return CreatedAPIToken{}, err
	}
	var label string
	if spec.Label != nil {
		label = *spec.Label
	} else {
		id := uuid.NewV4()
		label = hex.EncodeToString(id[:])
	}
	var pat domain.PAT
	_, _ = rand.Read(pat[:]) // never fails since Go 1.24
	n := NewAPIToken{
		ID: uuid.NewV7(), UserID: actor.UserID, TokenHash: pat.Hash(),
		Label: label, Description: spec.Description, ExpiredAt: spec.ExpiredAt, Now: now,
	}
	err = c.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if _, err := c.d.Lock.Lock(ctx, actor, now); err != nil {
			return err
		}
		return c.d.Tokens.CreateAPIToken(ctx, n)
	})
	if err != nil {
		return CreatedAPIToken{}, err
	}
	c.d.Logger.InfoContext(ctx, "API token created", slog.String("user_id", actor.UserID.String()), slog.String("token_id", n.ID.String()))
	return CreatedAPIToken{
		APIToken: domain.APIToken{ID: n.ID, Label: n.Label, Description: n.Description, ExpiredAt: n.ExpiredAt, CreatedAt: now},
		Token:    pat.String(),
	}, nil
}
```

`server/internal/modules/identity/app/list_api_tokens.go`（新文件）：

```go
package app

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListAPITokens lists the caller's tokens a page at a time:
// GET /api/v0/me/api-tokens.
type ListAPITokens struct {
	tokens APITokenLister
}

// NewListAPITokens returns the use case.
func NewListAPITokens(tokens APITokenLister) *ListAPITokens {
	return &ListAPITokens{tokens: tokens}
}

// APITokenPage is one page of tokens and the cursor of the next, "" after
// the last page.
type APITokenPage struct {
	Tokens     []domain.APIToken
	NextCursor string
}

// Execute returns the page that limit and cursor ask for, from the start
// when cursor is nil. A cursor this list did not issue is 400 bad_request
// and a limit outside 1–100 is 422 validation_failed; the cursor, being
// the request's structure, is judged first. One row more than the page is
// read to tell whether another page follows.
func (l *ListAPITokens) Execute(ctx context.Context, limit *int, cursor *string) (APITokenPage, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return APITokenPage{}, err
	}
	var after *domain.APITokenCursor
	if cursor != nil {
		after = new(domain.APITokenCursor)
		if err := shared.DecodeCursor(*cursor, after); err != nil {
			return APITokenPage{}, err
		}
	}
	size, err := domain.PageSize(limit)
	if err != nil {
		return APITokenPage{}, err
	}
	tokens, err := l.tokens.ListAPITokens(ctx, actor.UserID, after, size+1)
	if err != nil || len(tokens) <= size {
		return APITokenPage{Tokens: tokens}, err
	}
	last := tokens[size-1]
	next, err := shared.EncodeCursor(domain.APITokenCursor{CreatedAt: last.CreatedAt, ID: last.ID})
	if err != nil {
		return APITokenPage{}, err
	}
	return APITokenPage{Tokens: tokens[:size], NextCursor: next}, nil
}
```

`server/internal/modules/identity/app/revoke_api_token.go`（新文件）：

```go
package app

import (
	"context"
	"log/slog"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// RevokeAPIToken revokes one of the caller's tokens:
// DELETE /api/v0/api-tokens/{token_id}. It is one statement, no transaction
// (M2 design 6.4).
type RevokeAPIToken struct {
	tokens APITokenRevoker
	clock  Clock
	logger *slog.Logger
}

// NewRevokeAPIToken returns the use case.
func NewRevokeAPIToken(tokens APITokenRevoker, clock Clock, logger *slog.Logger) *RevokeAPIToken {
	return &RevokeAPIToken{tokens: tokens, clock: clock, logger: logger}
}

// Execute revokes token id of the caller. A token that does not exist, is
// revoked already or is another account's is identity.api_token_not_found.
// The token the request came with may revoke itself.
func (r *RevokeAPIToken) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	revoked, err := r.tokens.RevokeAPIToken(ctx, id, actor.UserID, r.clock.Now())
	switch {
	case err != nil:
		return err
	case !revoked:
		return domain.ErrAPITokenNotFound
	}
	r.logger.InfoContext(ctx, "API token revoked", slog.String("user_id", actor.UserID.String()), slog.String("token_id", id.String()))
	return nil
}
```

- [ ] **Step 5: 用例的测试**

`server/internal/modules/identity/app/fakes_test.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/modules/identity/app/fakes_test.go
+++ b/server/internal/modules/identity/app/fakes_test.go
@@ -279,7 +279,119 @@
 }
 
 func (m fakeMAC) Verify(message []byte, tag [16]byte) bool { return m.Tag(message) == tag }
+
+// callLog records the calls of the fakes that share it, in order, each
+// with the id it was given and " outside tx" when it ran outside a
+// transaction.
+type callLog struct{ calls []string }
+
+func (l *callLog) add(ctx context.Context, call string) {
+	if !inTx(ctx) {
+		call += " outside tx"
+	}
+	l.calls = append(l.calls, call)
+}
+
+// fakeCredentials is the account row and the session that the credential
+// lock reads.
+type fakeCredentials struct {
+	log        *callLog
+	account    app.LockedAccount
+	accountErr error
+	session    app.SessionCredential
+	sessionErr error
+}
+
+func (f *fakeCredentials) LockForCredentials(ctx context.Context, id uuid.UUID) (app.LockedAccount, error) {
+	f.log.add(ctx, "lock "+id.String())
+	return f.account, f.accountErr
+}
+
+func (f *fakeCredentials) SessionCredential(ctx context.Context, id uuid.UUID) (app.SessionCredential, error) {
+	f.log.add(ctx, "session "+id.String())
+	return f.session, f.sessionErr
+}
+
+// touch is one TouchAPIToken call.
+type touch struct {
+	id               uuid.UUID
+	now, staleBefore time.Time
+}
+
+// revocation is one RevokeAPIToken call.
+type revocation struct {
+	id, userID uuid.UUID
+	now        time.Time
+}
+
+// fakeAPITokens is the token ports, in memory. It finds a credential by the
+// hash and by the id it holds, so a lookup by any other key fails; it lists
+// at most limit of its rows.
+type fakeAPITokens struct {
+	log        *callLog
+	credential app.APITokenCredential
+	hash       []byte // the hash that finds credential
+	readErr    error
+	touches    []touch
+	touchErr   error
+	created    []app.NewAPIToken
+	rows       []domain.APIToken
+	listed     []listCall
+	revoked    []revocation
+	revokeOK   bool
+	revokeErr  error
+}
 
+// listCall is one ListAPITokens call.
+type listCall struct {
+	userID uuid.UUID
+	after  *domain.APITokenCursor
+	limit  int
+}
+
+func (f *fakeAPITokens) APITokenByHash(ctx context.Context, hash []byte) (app.APITokenCredential, error) {
+	f.log.add(ctx, "token by hash")
+	if f.readErr != nil {
+		return app.APITokenCredential{}, f.readErr
+	}
+	if !bytes.Equal(hash, f.hash) {
+		return app.APITokenCredential{}, app.ErrNotFound
+	}
+	return f.credential, nil
+}
+
+func (f *fakeAPITokens) APITokenByID(ctx context.Context, id uuid.UUID) (app.APITokenCredential, error) {
+	f.log.add(ctx, "token "+id.String())
+	if f.readErr != nil {
+		return app.APITokenCredential{}, f.readErr
+	}
+	if id != f.credential.ID {
+		return app.APITokenCredential{}, app.ErrNotFound
+	}
+	return f.credential, nil
+}
+
+func (f *fakeAPITokens) TouchAPIToken(_ context.Context, id uuid.UUID, now, staleBefore time.Time) error {
+	f.touches = append(f.touches, touch{id, now, staleBefore})
+	return f.touchErr
+}
+
+func (f *fakeAPITokens) CreateAPIToken(ctx context.Context, n app.NewAPIToken) error {
+	f.log.add(ctx, "insert "+n.ID.String())
+	f.created = append(f.created, n)
+	return nil
+}
+
+func (f *fakeAPITokens) ListAPITokens(_ context.Context, userID uuid.UUID, after *domain.APITokenCursor, limit int) ([]domain.APIToken, error) {
+	f.listed = append(f.listed, listCall{userID, after, limit})
+	return f.rows[:min(limit, len(f.rows))], nil
+}
+
+func (f *fakeAPITokens) RevokeAPIToken(_ context.Context, id, userID uuid.UUID, now time.Time) (bool, error) {
+	f.revoked = append(f.revoked, revocation{id, userID, now})
+	return f.revokeOK, f.revokeErr
+}
+
 type fixedPolicy struct {
 	allow bool
 	err   error
```

`server/internal/modules/identity/app/register_test.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/modules/identity/app/register_test.go
+++ b/server/internal/modules/identity/app/register_test.go
@@ -58,7 +58,7 @@
 func isV7(id uuid.UUID) bool { return id[6]>>4 == 7 }
 
 // assertNoSecret fails when logs hold secret in a spelling a log would
-// give it: as is (a string), in lower-case hex, or in either base64
+// give it: as is (a string), in hex of either case, or in either base64
 // alphabet (slog's JSON handler writes a []byte as standard base64). The
 // unpadded spellings also find the padded ones, which contain them.
 func assertNoSecret(t *testing.T, logs, name string, secret []byte) {
@@ -66,6 +66,7 @@
 	for _, spelling := range []string{
 		string(secret),
 		hex.EncodeToString(secret),
+		strings.ToUpper(hex.EncodeToString(secret)),
 		base64.RawStdEncoding.EncodeToString(secret),
 		base64.RawURLEncoding.EncodeToString(secret),
 	} {
```

`server/internal/modules/identity/app/create_api_token_test.go`（新文件）：

```go
package app_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"regexp"
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

// credentialFixture is the credential lock over fakes that share one call
// log: a live session and a live token of userID, an active account.
type credentialFixture struct {
	log    *callLog
	creds  *fakeCredentials
	tokens *fakeAPITokens
	tx     *fakeTx
	logs   *bytes.Buffer
}

func newCredentialFixture() *credentialFixture {
	log := &callLog{}
	return &credentialFixture{
		log:    log,
		creds:  &fakeCredentials{log: log, account: app.LockedAccount{PasswordHash: "hashed:Tr0ub4dor&3", Active: true}, session: validCredential()},
		tokens: &fakeAPITokens{log: log, credential: validToken()},
		tx:     &fakeTx{},
		logs:   &bytes.Buffer{},
	}
}

func (f *credentialFixture) lock() app.CredentialLock {
	return app.CredentialLock{Locker: f.creds, Sessions: f.creds, APITokens: f.tokens}
}

func (f *credentialFixture) logger() *slog.Logger { return slog.New(slog.NewJSONHandler(f.logs, nil)) }

func (f *credentialFixture) createAPIToken() *app.CreateAPIToken {
	return app.NewCreateAPIToken(app.CreateAPITokenDeps{Lock: f.lock(), Tokens: f.tokens, Tx: f.tx, Clock: clocktest.At(now), Logger: f.logger()})
}

var (
	sessionActor = shared.Actor{UserID: userID, SessionID: sessionID}
	tokenActor   = shared.Actor{UserID: userID, APITokenID: tokenID}
)

func TestCreateAPIToken(t *testing.T) {
	f := newCredentialFixture()
	label, expires := "deploy", now.Add(7*24*time.Hour)

	got, err := f.createAPIToken().Execute(shared.WithActor(context.Background(), sessionActor),
		domain.APITokenSpec{Label: &label, Description: "ci", ExpiredAt: &expires})

	if err != nil {
		t.Fatal(err)
	}
	pat, ok := domain.ParsePAT(got.Token)
	if !ok || len(f.tokens.created) != 1 {
		t.Fatalf("Execute() token %q (parses %v), %d rows; want a token and one row", got.Token, ok, len(f.tokens.created))
	}
	n := f.tokens.created[0]
	if n.UserID != userID || !bytes.Equal(n.TokenHash, pat.Hash()) || n.Label != "deploy" || n.Description != "ci" ||
		n.ExpiredAt == nil || !n.ExpiredAt.Equal(expires) || !n.Now.Equal(now) || !isV7(n.ID) {
		t.Errorf("row = %+v, want the caller's token with the hash of %q", n, got.Token)
	}
	want := domain.APIToken{ID: n.ID, Label: "deploy", Description: "ci", ExpiredAt: &expires, CreatedAt: now}
	if got.ID != want.ID || got.Label != want.Label || got.Description != want.Description || !got.ExpiredAt.Equal(expires) ||
		got.LastUsed != nil || !got.CreatedAt.Equal(now) {
		t.Errorf("Execute() = %+v, want %+v", got.APIToken, want)
	}
	// Locked, the credential checked again under the lock, then inserted:
	// one transaction (M2 design 3.5).
	wantCalls := []string{"lock " + userID.String(), "session " + sessionID.String(), "insert " + n.ID.String()}
	if !slices.Equal(f.log.calls, wantCalls) || f.tx.calls != 1 {
		t.Errorf("calls = %q in %d transactions, want %q in one", f.log.calls, f.tx.calls, wantCalls)
	}
	logs := f.logs.String()
	if !strings.Contains(logs, `"msg":"API token created"`) || !strings.Contains(logs, `"user_id":"`+userID.String()) ||
		!strings.Contains(logs, `"token_id":"`+n.ID.String()) {
		t.Errorf("logs = %s, want the creation with user_id and token_id", logs)
	}
	assertNoSecret(t, logs, "token", []byte(got.Token))
	assertNoSecret(t, logs, "token hash", n.TokenHash)
}

// Without a label the token gets 32 hexadecimal digits, as Plane's
// uuid4().hex, different each time.
func TestCreateAPITokenGeneratesALabel(t *testing.T) {
	f := newCredentialFixture()
	ctx := shared.WithActor(context.Background(), sessionActor)

	first, err1 := f.createAPIToken().Execute(ctx, domain.APITokenSpec{})
	second, err2 := f.createAPIToken().Execute(ctx, domain.APITokenSpec{})

	hex32 := regexp.MustCompile(`^[0-9a-f]{32}$`)
	if err := errors.Join(err1, err2); err != nil || !hex32.MatchString(first.Label) || first.Label == second.Label ||
		f.tokens.created[0].Label != first.Label {
		t.Errorf("labels %q and %q (%v), want two different ones of 32 hexadecimal digits", first.Label, second.Label, err)
	}
	if first.Token == second.Token {
		t.Errorf("two tokens are both %q", first.Token)
	}
}

// A token may create tokens (M2 design 4.6): the lock checks that token
// again, not a session.
func TestCreateAPITokenWithAToken(t *testing.T) {
	f := newCredentialFixture()

	got, err := f.createAPIToken().Execute(shared.WithActor(context.Background(), tokenActor), domain.APITokenSpec{})

	wantCalls := []string{"lock " + userID.String(), "token " + tokenID.String(), "insert " + got.ID.String()}
	if err != nil || !slices.Equal(f.log.calls, wantCalls) {
		t.Errorf("calls = %q, %v; want %q", f.log.calls, err, wantCalls)
	}
}

// Under the lock the caller's credential is checked again: one that a
// concurrent reset or logout ended since the request was authenticated
// creates nothing (M2 design 3.5, interleaving 3).
func TestCreateAPITokenRechecksTheCredentialUnderTheLock(t *testing.T) {
	other := uuid.MustParse("0199a2b4-0000-7000-8000-000000000009")
	tests := []struct {
		name   string
		actor  shared.Actor
		change func(*credentialFixture)
		reason string
	}{
		{"account gone", sessionActor, func(f *credentialFixture) { f.creds.accountErr = app.ErrNotFound }, "account does not exist"},
		{"account deactivated", sessionActor, func(f *credentialFixture) { f.creds.account.Active = false }, "account is deactivated"},
		{"session revoked", sessionActor, func(f *credentialFixture) { f.creds.session.Revoked = true }, "session is revoked"},
		{"session expired", sessionActor, func(f *credentialFixture) { f.creds.session.ExpiresAt = now }, "session has expired"},
		{"session gone", sessionActor, func(f *credentialFixture) { f.creds.sessionErr = app.ErrNotFound }, "session does not exist"},
		{"session of another account", sessionActor, func(f *credentialFixture) { f.creds.session.UserID = other }, "session belongs to another account"},
		{"token revoked", tokenActor, func(f *credentialFixture) { f.tokens.credential.Revoked = true }, "personal access token is revoked"},
		{"token expired", tokenActor, func(f *credentialFixture) { f.tokens.credential.ExpiredAt = &now }, "personal access token has expired"},
		{"token of another account", tokenActor, func(f *credentialFixture) { f.tokens.credential.UserID = other }, "personal access token belongs to another account"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newCredentialFixture()
			tt.change(f)

			_, err := f.createAPIToken().Execute(shared.WithActor(context.Background(), tt.actor), domain.APITokenSpec{})

			var se *shared.Error
			if !errors.As(err, &se) || se.ProblemStatus() != 401 || err.Error() != "Authentication is required.: "+tt.reason {
				t.Errorf("Execute() = %v, want 401 because %s", err, tt.reason)
			}
			if len(f.tokens.created) != 0 {
				t.Errorf("inserted %+v, want nothing", f.tokens.created)
			}
		})
	}
}

func TestCreateAPITokenLockFailureIsNot401(t *testing.T) {
	f := newCredentialFixture()
	boom := errors.New("connection refused")
	f.creds.accountErr = boom

	_, err := f.createAPIToken().Execute(shared.WithActor(context.Background(), sessionActor), domain.APITokenSpec{})

	var se *shared.Error
	if !errors.Is(err, boom) || errors.As(err, &se) || len(f.tokens.created) != 0 {
		t.Errorf("Execute() = %v, inserted %d; want the database error and nothing inserted", err, len(f.tokens.created))
	}
}

// The spec is checked before anything is locked.
func TestCreateAPITokenChecksTheSpecFirst(t *testing.T) {
	f := newCredentialFixture()
	empty := ""

	_, err := f.createAPIToken().Execute(shared.WithActor(context.Background(), sessionActor), domain.APITokenSpec{Label: &empty})

	var se *shared.Error
	if !errors.As(err, &se) || se.Code != shared.CodeValidationFailed || len(f.log.calls) != 0 {
		t.Errorf("Execute() = %v after calls %q, want 422 before any", err, f.log.calls)
	}
}
```

`server/internal/modules/identity/app/list_api_tokens_test.go`（新文件）：

```go
package app_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fiveTokens are rows in the list's order, one second apart.
func fiveTokens() []domain.APIToken {
	var rows []domain.APIToken
	for i := range 5 {
		rows = append(rows, domain.APIToken{ID: uuid.NewV7(), Label: "t", CreatedAt: now.Add(-time.Duration(i) * time.Second)})
	}
	return rows
}

func listPage(t *testing.T, tokens *fakeAPITokens, limit *int, cursor *string) (app.APITokenPage, error) {
	t.Helper()
	ctx := shared.WithActor(context.Background(), sessionActor)
	return app.NewListAPITokens(tokens).Execute(ctx, limit, cursor)
}

func ptr[T any](v T) *T { return &v }

// Without a limit the page holds 50; one row more is read to tell whether
// another page follows.
func TestListAPITokensFirstPage(t *testing.T) {
	tokens := &fakeAPITokens{rows: fiveTokens()}

	page, err := listPage(t, tokens, nil, nil)

	if err != nil || len(page.Tokens) != 5 || page.NextCursor != "" {
		t.Errorf("Execute() = %d tokens, next %q, %v; want all 5 and no next page", len(page.Tokens), page.NextCursor, err)
	}
	if len(tokens.listed) != 1 || tokens.listed[0] != (listCall{userID, nil, 51}) {
		t.Errorf("store asked for %+v, want the caller's first 51 rows", tokens.listed)
	}
}

// A full page with a row after it names the page's last row in its
// cursor; the next call passes that row on to the store.
func TestListAPITokensNextPage(t *testing.T) {
	rows := fiveTokens()
	tokens := &fakeAPITokens{rows: rows}

	page, err := listPage(t, tokens, ptr(2), nil)
	if err != nil || len(page.Tokens) != 2 || page.Tokens[1].ID != rows[1].ID || page.NextCursor == "" {
		t.Fatalf("Execute() = %+v, %v; want the first 2 rows and a cursor", page, err)
	}
	var c domain.APITokenCursor
	if err := shared.DecodeCursor(page.NextCursor, &c); err != nil || c.ID != rows[1].ID || !c.CreatedAt.Equal(rows[1].CreatedAt) {
		t.Errorf("cursor = %+v, %v; want the second row", c, err)
	}

	if _, err := listPage(t, tokens, ptr(2), &page.NextCursor); err != nil {
		t.Fatal(err)
	}
	if next := tokens.listed[1]; next.limit != 3 || next.after == nil || next.after.ID != rows[1].ID {
		t.Errorf("second call asked for %+v, want 3 rows after the second row", next)
	}
}

// A page that ends the list has no cursor, even when it is full.
func TestListAPITokensLastFullPage(t *testing.T) {
	page, err := listPage(t, &fakeAPITokens{rows: fiveTokens()[:2]}, ptr(2), nil)

	if err != nil || len(page.Tokens) != 2 || page.NextCursor != "" {
		t.Errorf("Execute() = %d tokens, next %q, %v; want 2 and no next page", len(page.Tokens), page.NextCursor, err)
	}
}

// A bad cursor is 400 and a bad limit 422; with both, the cursor's 400.
// The store is not asked.
func TestListAPITokensRejects(t *testing.T) {
	bad := "not a cursor"
	tests := []struct {
		name   string
		limit  *int
		cursor *string
		status int
		field  string
	}{
		{"a cursor the list did not issue", nil, &bad, 400, "cursor"},
		{"limit 0", ptr(0), nil, 422, "limit"},
		{"limit 101", ptr(101), nil, 422, "limit"},
		{"both", ptr(0), &bad, 400, "cursor"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := &fakeAPITokens{rows: fiveTokens()}

			_, err := listPage(t, tokens, tt.limit, tt.cursor)

			var se *shared.Error
			if !errors.As(err, &se) || se.ProblemStatus() != tt.status || len(se.Fields) != 1 || se.Fields[0].Field != tt.field {
				t.Errorf("Execute() = %v, want %d on %s", err, tt.status, tt.field)
			}
			if len(tokens.listed) != 0 {
				t.Errorf("store asked for %+v, want nothing", tokens.listed)
			}
		})
	}
}
```

`server/internal/modules/identity/app/revoke_api_token_test.go`（新文件）：

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

func revoke(tokens *fakeAPITokens, logs *bytes.Buffer) error {
	uc := app.NewRevokeAPIToken(tokens, clocktest.At(now), slog.New(slog.NewJSONHandler(logs, nil)))
	return uc.Execute(shared.WithActor(context.Background(), sessionActor), tokenID)
}

func TestRevokeAPIToken(t *testing.T) {
	tokens, logs := &fakeAPITokens{revokeOK: true}, &bytes.Buffer{}

	err := revoke(tokens, logs)

	if want := (revocation{tokenID, userID, now}); err != nil || len(tokens.revoked) != 1 || tokens.revoked[0] != want {
		t.Errorf("Execute() = %v, revoked %+v; want %+v", err, tokens.revoked, want)
	}
	if s := logs.String(); !strings.Contains(s, `"msg":"API token revoked"`) || !strings.Contains(s, `"token_id":"`+tokenID.String()) ||
		!strings.Contains(s, `"user_id":"`+userID.String()) {
		t.Errorf("logs = %s, want the revocation with user_id and token_id", s)
	}
}

// A token that does not exist, is revoked already or is another account's
// is not found (M2 design 5.4).
func TestRevokeAnAPITokenThatIsNotTheCallers(t *testing.T) {
	err := revoke(&fakeAPITokens{revokeOK: false}, &bytes.Buffer{})

	if !errors.Is(err, domain.ErrAPITokenNotFound) {
		t.Errorf("Execute() = %v, want identity.api_token_not_found", err)
	}
}

func TestRevokeAPITokenDatabaseFailure(t *testing.T) {
	boom := errors.New("connection refused")

	if err := revoke(&fakeAPITokens{revokeErr: boom}, &bytes.Buffer{}); !errors.Is(err, boom) {
		t.Errorf("Execute() = %v, want the database error", err)
	}
}
```

- [ ] **Step 6: 限流键和接线**

`server/internal/modules/identity/adapter/authn/authenticator.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/modules/identity/adapter/authn/authenticator.go
+++ b/server/internal/modules/identity/adapter/authn/authenticator.go
@@ -5,6 +5,7 @@
 import (
 	"context"
 	"errors"
+	"uuid"
 
 	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
 	"github.com/open-nerve/NerveProject/server/internal/shared"
@@ -26,7 +27,8 @@
 }
 
 // Authenticate returns a context carrying the actor of token and the
-// caller's rate-limit key, session:<id>. An invalid token is the use case's
+// caller's rate-limit key: session:<id> for an access token, pat:<id> for a
+// personal access token (M2 design 3.10). An invalid token is the use case's
 // 401 *shared.Error; for an access token that is valid but for its expiry,
 // that error also reports ExpiredCredential() true, so the platform's
 // failure gate does not count it. Any other error passes through as an
@@ -39,7 +41,11 @@
 	if err != nil {
 		return nil, "", err
 	}
-	return shared.WithActor(ctx, actor), "session:" + actor.SessionID.String(), nil
+	key := "session:" + actor.SessionID.String()
+	if actor.APITokenID != uuid.Nil() {
+		key = "pat:" + actor.APITokenID.String()
+	}
+	return shared.WithActor(ctx, actor), key, nil
 }
 
 // expired is the 401 of an expired access token: the client's cue to
```

`server/internal/modules/identity/adapter/authn/authenticator_test.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/modules/identity/adapter/authn/authenticator_test.go
+++ b/server/internal/modules/identity/adapter/authn/authenticator_test.go
@@ -41,6 +41,18 @@
 	}
 }
 
+// A personal access token is limited by its own id (M2 design 3.10).
+func TestAuthenticateKeysAPATByItsID(t *testing.T) {
+	actor := shared.Actor{UserID: uuid.MustParse("0199a2b4-0000-7000-8000-000000000001"), APITokenID: uuid.MustParse("0199a2b4-0000-7000-8000-000000000003")}
+
+	ctx, key, err := authn.New(fakeUseCase{token: "nrv_pat_x", actor: actor}).Authenticate(context.Background(), "nrv_pat_x")
+
+	got, actorErr := shared.RequireActor(ctx)
+	if err != nil || actorErr != nil || got != actor || key != "pat:0199a2b4-0000-7000-8000-000000000003" {
+		t.Errorf("Authenticate() = actor %+v (%v), key %q, %v", got, actorErr, key, err)
+	}
+}
+
 // expiredCredential is the platform's optional interface on a 401.
 type expiredCredential interface{ ExpiredCredential() bool }
 
```

`server/internal/modules/identity/module.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/modules/identity/module.go
+++ b/server/internal/modules/identity/module.go
@@ -110,7 +110,9 @@
 			RefreshDeadline: d.RefreshDeadline,
 			Logger:          d.Logger,
 		},
-		authenticator: authn.New(app.NewAuthenticate(tokens, store, d.Clock)),
+		authenticator: authn.New(app.NewAuthenticate(app.AuthenticateDeps{
+			AccessTokens: tokens, Sessions: store, APITokens: store, Touch: store, Clock: d.Clock,
+		})),
 	}, nil
 }
 
```

- [ ] **Step 7: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。令牌的三个用例此时还没有接到接口上（Task 7）。

- [ ] **Step 8: 提交**

```bash
git add server/internal/shared/actor.go server/internal/modules/identity/app server/internal/modules/identity/domain/errors.go server/internal/modules/identity/adapter/authn server/internal/modules/identity/module.go
```
```bash
git commit -m "feat(M2/P3a): personal access token authentication, the credential lock and the token use cases

A nrv_pat_ token is looked up by its hash; last_used is written at most
once a minute. CredentialLock locks the account row and checks the
caller's credential again under it, so a token is never created with a
credential that a concurrent change ended (M2 design 3.5).

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** `TestAuthenticateRejectsAPAT`、`TestCreateAPITokenRechecksTheCredentialUnderTheLock`、`TestListAPITokensRejects` 通过；`make test` 通过。

---

### Task 7: 令牌的三个操作；`oapi-codegen/runtime`；参数绑定的整程序测试

**Files:**
- Modify: `api/common.yaml`；`api/openapi.yaml`、`api/modules/identity.yaml`（过渡：Task 9、10、11、12 加上别的操作）
- Modify: `server/go.mod`、`server/go.sum`
- Modify: `server/internal/platform/httpserver/apitest/operations.go`、`operations_test.go`
- Create: `server/internal/modules/identity/adapter/http/api_tokens.go`、`api_tokens_test.go`
- Modify: `server/internal/modules/identity/adapter/http/handler.go`、`handler_test.go`（过渡）；`server/internal/modules/identity/module.go`（过渡）
- Modify: `server/internal/bootstrap/auth_test.go`、`contract_test.go`
- Generate: `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`、`server/internal/platform/httpserver/apigen/components.gen.go`、`server/internal/modules/identity/adapter/http/gen/server.gen.go`、`bodyshape.gen.go`

**Interfaces:**
- Consumes: 令牌的三个用例（Task 6）；`APIErrors.BadRequest`（Task 2）；uuid 守卫（Task 3）。
- Produces（spec 2.11，M2 设计 3.11、3.12、5.1、5.2、5.4）：
  - `common.yaml`：参数 `Limit`（`integer`，1–100，默认 50）、`Cursor`（`string`）；schema `NextCursor`（`[string, 'null']`）；
  - 操作：`listApiTokens`（`GET /api/v0/me/api-tokens`，200 `ApiTokenPage{data, next_cursor}`，`x-problem-codes: [validation_failed]`）、`createApiToken`（`POST /api/v0/me/api-tokens`，201 `ApiTokenCreated`，`[validation_failed]`）、`revokeApiToken`（`DELETE /api/v0/api-tokens/{token_id}`，204，`[identity.api_token_not_found]`）；
  - `ApiToken{id, label, description, expired_at, last_used, created_at}`（`expired_at`、`last_used` 必有、可为 `null`）；`ApiTokenCreate{label, description, expired_at}` 全部可选，`expired_at` 可为 `null`；`ApiTokenCreated` 加上 `token`；
  - `github.com/oapi-codegen/runtime` v1.7.0（第一批带参数的操作）；
  - `apitest.Operation.ParamCases()`：路径和查询参数中 Go 类型会拒绝某些字符串的（整数、数、布尔、生成为 Go 类型的格式）各一个写错的目标；`Target` 只填必填的查询参数，用例里写错的那个总会填上；
  - 整程序测试 5：`TestParametersThatDoNotBindAnswer400`（spec 2.17）。

**Tests:**
- `api_tokens_test.go`：`TestCreateAPITokenAnswers201WithTheToken`（响应体逐字核对，微秒精度的时间）；`TestCreateAPITokenWithoutFields`（`{}` 和 `{"expired_at":null}` 到用例都是"没有"）；`TestCreateAPITokenInvalid`；`TestListAPITokens`；`TestListAPITokensFirstAndLastPage`（`next_cursor` 是 `null`，不是空串）；`TestListAPITokensProblems`（422 `limit`、400 `cursor`）；`TestRevokeAPIToken`；`TestRevokeAnAPITokenThatIsNotTheCallers`（404）；`TestParametersThatDoNotBind`（`limit=abc`、`token_id=not-a-uuid`：400 响应体逐字核对，用例没有被调用）。
- `operations_test.go`：`TestParamCases`（uuid 路径参数和整数得到用例，枚举和自由字符串不得）；`TestParamCasesOfAnOptionalParameter`（布尔、`date-time`；没有用例时目标不带查询串）。
- `bootstrap/contract_test.go`：`TestParametersThatDoNotBindAnswer400`（不带令牌，数据库不可达：参数在认证之前绑定；从契约推不出用例时失败）。
- `bootstrap/auth_test.go`：`TestAPersonalAccessTokenAuthenticates`（真实数据库：用访问令牌建第一个 PAT，用它建第二个；第一个读 `/me` 200、撤销自己 204、再读 401；第二个仍然 200）。
- `apitest.Main` 核对 `identity.api_token_not_found` 由 handler 测试答过。

- [ ] **Step 1: 契约**

`api/common.yaml`（对 `605f367` 的差异）：

```diff
--- a/api/common.yaml
+++ b/api/common.yaml
@@ -7,7 +7,31 @@
   title: Nerve common components
   version: v0
 components:
+  # 分页（M2 设计 3.12）：每个列表接口引用 Limit、Cursor，响应的 next_cursor 是 NextCursor
+  parameters:
+    Limit:
+      name: limit
+      in: query
+      description: >-
+        The page size, 1–100; 50 when absent. Outside that range the answer is
+        422 validation_failed on limit.
+      schema:
+        type: integer
+        minimum: 1
+        maximum: 100
+        default: 50
+    Cursor:
+      name: cursor
+      in: query
+      description: >-
+        The next_cursor of the page before; absent for the first page. A
+        cursor that this list did not issue is 400 bad_request on cursor.
+      schema:
+        type: string
   schemas:
+    NextCursor:
+      description: The cursor of the next page; null on the last page.
+      type: [string, 'null']
     Problem:
       description: >-
         RFC 9457 problem details (v0 design 3.5). `title` is the HTTP status
```

`api/openapi.yaml`（对 `605f367` 的差异）：

```diff
--- a/api/openapi.yaml
+++ b/api/openapi.yaml
@@ -31,6 +31,10 @@
     $ref: 'modules/identity.yaml#/paths/~1api~1v0~1auth~1logout'
   /api/v0/me:
     $ref: 'modules/identity.yaml#/paths/~1api~1v0~1me'
+  /api/v0/me/api-tokens:
+    $ref: 'modules/identity.yaml#/paths/~1api~1v0~1me~1api-tokens'
+  /api/v0/api-tokens/{token_id}:
+    $ref: 'modules/identity.yaml#/paths/~1api~1v0~1api-tokens~1{token_id}'
   /api/v0/instance:
     $ref: 'modules/instance.yaml#/paths/~1api~1v0~1instance'
 components:
```

`api/modules/identity.yaml`（对 `605f367` 的差异）：

```diff
--- a/api/modules/identity.yaml
+++ b/api/modules/identity.yaml
@@ -130,8 +130,81 @@
             application/json:
               schema:
                 $ref: '#/components/schemas/User'
+        default:
+          $ref: '#/components/responses/Problem'
+  /api/v0/me/api-tokens:
+    get:
+      operationId: listApiTokens
+      tags: [identity]
+      summary: List the caller's personal access tokens
+      description: >-
+        The caller's personal access tokens that are not revoked, newest first,
+        a page at a time. The tokens themselves are never listed: only
+        createApiToken answers one, once.
+      security: [{bearer: []}]
+      x-problem-codes: [validation_failed]
+      parameters:
+        - $ref: '../common.yaml#/components/parameters/Limit'
+        - $ref: '../common.yaml#/components/parameters/Cursor'
+      responses:
+        '200':
+          description: One page of tokens.
+          content:
+            application/json:
+              schema:
+                $ref: '#/components/schemas/ApiTokenPage'
+        default:
+          $ref: '#/components/responses/Problem'
+    post:
+      operationId: createApiToken
+      tags: [identity]
+      summary: Create a personal access token
+      description: >-
+        Creates a personal access token of the caller and answers the token
+        itself, this once. Sent as "Authorization: Bearer", it acts as the
+        account in every operation until it expires or is revoked, and it can
+        create tokens itself. Any credential may create one; no password is
+        asked for.
+      security: [{bearer: []}]
+      x-problem-codes: [validation_failed]
+      requestBody:
+        required: true
+        content:
+          application/json:
+            schema:
+              $ref: '#/components/schemas/ApiTokenCreate'
+      responses:
+        '201':
+          description: The new token, with the token itself.
+          content:
+            application/json:
+              schema:
+                $ref: '#/components/schemas/ApiTokenCreated'
         default:
           $ref: '#/components/responses/Problem'
+  /api/v0/api-tokens/{token_id}:
+    parameters:
+      - name: token_id
+        in: path
+        required: true
+        schema:
+          type: string
+          format: uuid
+    delete:
+      operationId: revokeApiToken
+      tags: [identity]
+      summary: Revoke a personal access token
+      description: >-
+        Revokes one of the caller's tokens, which stops working at once; a
+        token may revoke itself. A token that does not exist, is revoked
+        already or belongs to another account is identity.api_token_not_found.
+      security: [{bearer: []}]
+      x-problem-codes: [identity.api_token_not_found]
+      responses:
+        '204':
+          description: Revoked.
+        default:
+          $ref: '#/components/responses/Problem'
 components:
   # 与 api/openapi.yaml 的相同：打包时丢掉模块文件的 securitySchemes，oapi-codegen 读这一份（M0-P3 交接 3）
   securitySchemes:
@@ -248,7 +321,86 @@
           type: [string, 'null']
         cover_image_url:
           description: Null until uploads arrive (M5).
+          type: [string, 'null']
+        created_at:
+          type: string
+          format: date-time
+    ApiToken:
+      description: >-
+        A personal access token as lists show it. The token itself appears
+        only in ApiTokenCreated.
+      type: object
+      additionalProperties: false
+      required: [id, label, description, expired_at, last_used, created_at]
+      properties:
+        id:
+          type: string
+          format: uuid
+        label:
+          type: string
+        description:
+          type: string
+        expired_at:
+          description: When the token stops working; null when it never does.
           type: [string, 'null']
+          format: date-time
+        last_used:
+          description: When the token last authenticated a request, to the minute; null when it never has.
+          type: [string, 'null']
+          format: date-time
         created_at:
           type: string
           format: date-time
+    ApiTokenCreate:
+      type: object
+      additionalProperties: false
+      properties:
+        label:
+          description: 1–255 characters; 32 hexadecimal digits are made up when it is absent.
+          type: string
+        description:
+          type: string
+        expired_at:
+          description: A time in the future; absent or null for a token that never expires.
+          type: [string, 'null']
+          format: date-time
+    ApiTokenCreated:
+      description: >-
+        A new personal access token: the fields of ApiToken, and the token
+        itself, which is shown this once and cannot be read again.
+      type: object
+      additionalProperties: false
+      required: [id, label, description, expired_at, last_used, created_at, token]
+      properties:
+        id:
+          type: string
+          format: uuid
+        label:
+          type: string
+        description:
+          type: string
+        expired_at:
+          description: When the token stops working; null when it never does.
+          type: [string, 'null']
+          format: date-time
+        last_used:
+          description: Null; the token has not been used yet.
+          type: [string, 'null']
+          format: date-time
+        created_at:
+          type: string
+          format: date-time
+        token:
+          description: The token, nrv_pat_ and 43 more characters.
+          type: string
+    ApiTokenPage:
+      type: object
+      additionalProperties: false
+      required: [data, next_cursor]
+      properties:
+        data:
+          type: array
+          items:
+            $ref: '#/components/schemas/ApiToken'
+        next_cursor:
+          $ref: '../common.yaml#/components/schemas/NextCursor'
```

- [ ] **Step 2: 生成**

Run: `make gen`
Expected: 以下五个文件改变，其余生成物不变：

| 生成的文件 | SHA-256 | 行数 |
|---|---|---|
| `api/dist/openapi.yaml` | `20439ddfda9fd5007f33fb8a84bbfa89ef6a7bd92d769747961a04cd80d26323` | 565 |
| `server/internal/modules/identity/adapter/http/gen/bodyshape.gen.go` | `113b28c4a162fc1a8436c3557bbe9dd00c338ccf594fe62d5626b7338f7a4974` | 36 |
| `server/internal/modules/identity/adapter/http/gen/server.gen.go` | `967d8dbb2496efe4c91007865bd5a6bcce1c99ce9b67d10576fddb8e81b708d3` | 1163 |
| `server/internal/platform/httpserver/apigen/components.gen.go` | `ccac4fe0c4162baa0867cd916d94d30bef5a417cb254680dc4015bfc66c3c2b3` | 86 |
| `web/packages/api-client/src/schema.gen.ts` | `6ea759946a287187158e0d14eaca8d45665033af43b4e682801ee35d30d2816b` | 569 |

生成的 `server.gen.go` 导入 `github.com/oapi-codegen/runtime`，下一步把它加进 `go.mod`。`api/dist/openapi.yaml`、`schema.gen.ts`、`server.gen.go`、`bodyshape.gen.go` 在 Task 9–12 再次生成；`components.gen.go` 是最终版本。

- [ ] **Step 3: `oapi-codegen/runtime`**

Run: `go -C server get github.com/oapi-codegen/runtime@v1.7.0`
Run: `go -C server mod tidy`
Run: `grep -n "^go \|^toolchain" server/go.mod server/tools/go.mod`
Expected: 两个文件都是 `go 1.27`、`toolchain go1.27.1`；`server/go.mod`、`go.sum` 与下面的差异相同：

`server/go.mod`（对 `605f367` 的差异）：

```diff
--- a/server/go.mod
+++ b/server/go.mod
@@ -14,6 +14,7 @@
 	github.com/knadh/koanf/providers/rawbytes v1.0.1
 	github.com/knadh/koanf/v2 v2.3.6
 	github.com/oapi-codegen/nullable v1.2.0
+	github.com/oapi-codegen/runtime v1.7.0
 	github.com/pressly/goose/v3 v3.28.0
 	github.com/spf13/cobra v1.10.2
 	github.com/testcontainers/testcontainers-go/modules/postgres v0.44.0
@@ -25,6 +26,7 @@
 	dario.cat/mergo v1.0.2 // indirect
 	github.com/Azure/go-ansiterm v0.0.0-20250102033503-faa5f7b0171c // indirect
 	github.com/Microsoft/go-winio v0.6.2 // indirect
+	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
 	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
 	github.com/cespare/xxhash/v2 v2.3.0 // indirect
 	github.com/containerd/errdefs v1.0.0 // indirect
```

`server/go.sum`（对 `605f367` 的差异）：

```diff
--- a/server/go.sum
+++ b/server/go.sum
@@ -6,6 +6,10 @@
 github.com/Azure/go-ansiterm v0.0.0-20250102033503-faa5f7b0171c/go.mod h1:xomTg63KZ2rFqZQzSB4Vz2SUXa1BpHTVz9L5PTmPC4E=
 github.com/Microsoft/go-winio v0.6.2 h1:F2VQgta7ecxGYO8k3ZZz3RS8fVIXVxONVUPlNERoyfY=
 github.com/Microsoft/go-winio v0.6.2/go.mod h1:yd8OoFMLzJbo9gZq8j5qaps8bJ9aShtEA8Ipt1oGCvU=
+github.com/RaveNoX/go-jsoncommentstrip v1.0.0/go.mod h1:78ihd09MekBnJnxpICcwzCMzGrKSKYe4AqU6PDYYpjk=
+github.com/apapsch/go-jsonmerge/v2 v2.0.0 h1:axGnT1gRIfimI7gJifB699GoE/oq+F2MU7Dml6nw9rQ=
+github.com/apapsch/go-jsonmerge/v2 v2.0.0/go.mod h1:lvDnEdqiQrp0O42VQGgmlKpxL1AP2+08jFMw88y4klk=
+github.com/bmatcuk/doublestar v1.1.1/go.mod h1:UD6OnuiIn0yFxxA2le/rnRU1G4RaI4UvFv1sNto9p6w=
 github.com/cenkalti/backoff/v4 v4.3.0 h1:MyRJ/UdXutAwSAT+s3wNd7MfTIcy71VQueUuFK343L8=
 github.com/cenkalti/backoff/v4 v4.3.0/go.mod h1:Y3VNntkOUPxTVeUxJ/G5vcM//AlwfmyYozVcomhLiZE=
 github.com/cespare/xxhash/v2 v2.3.0 h1:UL815xU9SqsFlibzuggzjXhog7bL6oX9BbNZnL2UFvs=
@@ -74,6 +78,7 @@
 github.com/jackc/pgx/v5 v5.11.0/go.mod h1:mal1tBGAFfLHvZzaYh77YS/eC6IX9OWbRV1QIIM0Jn4=
 github.com/jackc/puddle/v2 v2.2.2 h1:PR8nw+E/1w0GLuRFSmiioY6UooMp6KJv0/61nB7icHo=
 github.com/jackc/puddle/v2 v2.2.2/go.mod h1:vriiEXHvEE654aYKXXjOvZM39qJ0q+azkZFrfEOc3H4=
+github.com/juju/gnuflag v0.0.0-20171113085948-2ce1bb71843d/go.mod h1:2PavIy+JPciBPrBUjwbNvtwB6RQlve+hkpll6QSNmOE=
 github.com/klauspost/compress v1.19.2 h1:hMRETovs/pu/dVWN7zIT1PGG8t509MwT6bO7XSi26R8=
 github.com/klauspost/compress v1.19.2/go.mod h1:cwPg85FWrGar70rWktvGQj8/hthj3wpl0PGDogxkrSQ=
 github.com/knadh/koanf/maps v0.1.2 h1:RBfmAW5CnZT+PJ1CVc1QSJKf4Xu9kxfQgYVQSu8hpbo=
@@ -128,6 +133,8 @@
 github.com/ncruces/go-strftime v1.0.0/go.mod h1:Fwc5htZGVVkseilnfgOVb9mKy6w1naJmn9CehxcKcls=
 github.com/oapi-codegen/nullable v1.2.0 h1:VflFkDW980KhBPiFF7nWSyjg+r4Obqj8lXipV0UkP5w=
 github.com/oapi-codegen/nullable v1.2.0/go.mod h1:KUZ3vUzkmEKY90ksAmit2+5juDIhIZhfDl+0PwOQlFY=
+github.com/oapi-codegen/runtime v1.7.0 h1:t7358VYPvNbWJ9gdAkIK/smVeHpBf6yp8VTsaZsb/7k=
+github.com/oapi-codegen/runtime v1.7.0/go.mod h1:GwV7hC2hviaMzj+ITfHVRESK5J2W/GefVwIND/bMGvU=
 github.com/oasdiff/yaml v0.1.1 h1:6nHx+pn9gBRM6YpBlFZFQGCCd1nuvqOBtTD3KKTgGxY=
 github.com/oasdiff/yaml v0.1.1/go.mod h1:EYJNoyktvWMJ0Hmhx+6qTaqMOsalUaRGT8Sj1hNcegU=
 github.com/oasdiff/yaml3 v0.0.14 h1:aLJee3hxBK2H5wdXd9iPcIXb93Nty1Ge0pT171eHtkw=
@@ -158,6 +165,7 @@
 github.com/spf13/cobra v1.10.2/go.mod h1:7C1pvHqHw5A4vrJfjNwvOdzYu0Gml16OCs2GRiTUUS4=
 github.com/spf13/pflag v1.0.9 h1:9exaQaMOCwffKiiiYk6/BndUBv+iRViNW+4lEMi0PvY=
 github.com/spf13/pflag v1.0.9/go.mod h1:McXfInJRrz4CZXVZOBLb0bTZqETkiAhM9Iw0y3An2Bg=
+github.com/spkg/bom v0.0.0-20160624110644-59b7046e48ad/go.mod h1:qLr4V1qq6nMqFKkMo8ZTx3f+BZEkzsRUY10Xsm2mwU0=
 github.com/stretchr/objx v0.1.0/go.mod h1:HFkY916IF+rwdDfMAkV7OtwuqBVzrE8GR6GFx+wExME=
 github.com/stretchr/objx v0.5.3 h1:jmXUvGomnU1o3W/V5h2VEradbpJDwGrzugQQvL0POH4=
 github.com/stretchr/objx v0.5.3/go.mod h1:rDQraq+vQZU7Fde9LOZLr8Tax6zZvy4kuNKF+QYS+U0=
```

- [ ] **Step 4: 参数用例**

`server/internal/platform/httpserver/apitest/operations.go`（对 Task 2 版本的差异）：

```diff
--- a/server/internal/platform/httpserver/apitest/operations.go
+++ b/server/internal/platform/httpserver/apitest/operations.go
@@ -36,21 +36,65 @@
 // Parameters bind before the middlewares, so a wrong one would answer 400
 // before anything else runs (M2 design 3.6).
 func (o Operation) Target() string {
+	return o.target("", "")
+}
+
+// target is Target with parameter name set to value, when name is not "".
+func (o Operation) target(name, value string) string {
 	path, query := o.Path, url.Values{}
 	for _, ref := range o.params {
 		p := ref.Value
-		value := fmt.Sprint(validValue(p.Schema.Value))
+		v, set := fmt.Sprint(validValue(p.Schema.Value)), p.Required
+		if p.Name == name {
+			v, set = value, true
+		}
 		switch {
 		case p.In == openapi3.ParameterInPath:
-			path = strings.ReplaceAll(path, "{"+p.Name+"}", url.PathEscape(value))
-		case p.In == openapi3.ParameterInQuery && p.Required:
-			query.Set(p.Name, value)
+			path = strings.ReplaceAll(path, "{"+p.Name+"}", url.PathEscape(v))
+		case p.In == openapi3.ParameterInQuery && set:
+			query.Set(p.Name, v)
 		}
 	}
 	if len(query) == 0 {
 		return path
 	}
 	return path + "?" + query.Encode()
+}
+
+// ParamCase is a request target with one parameter that cannot bind, and
+// the parameter the 400 must name.
+type ParamCase struct {
+	Name   string
+	Target string
+	Field  string
+}
+
+// ParamCases derives the cases of the parameter binding whole-program test
+// (M2 design 3.11): for each path or query parameter whose Go type rejects
+// some strings, a number, a boolean, or a string whose format is generated
+// as a Go type, the example target with that parameter wrong. Other strings,
+// enums too, bind whatever they are.
+func (o Operation) ParamCases() []ParamCase {
+	var cases []ParamCase
+	for _, ref := range o.params {
+		p := ref.Value
+		if p.In != openapi3.ParameterInPath && p.In != openapi3.ParameterInQuery {
+			continue
+		}
+		var wrong string
+		switch s := p.Schema.Value; {
+		case s.Type.Includes("integer"), s.Type.Includes("number"):
+			wrong = "not-a-number"
+		case s.Type.Includes("boolean"):
+			wrong = "not-a-boolean"
+		case checkedFormat(s):
+			wrong = "not-a-" + s.Format
+		default:
+			continue
+		}
+		cases = append(cases, ParamCase{Name: "wrong " + p.Name, Target: o.target(p.Name, wrong), Field: p.Name})
+	}
+	return cases
 }
 
 // Operations lists every operation of the contract, sorted by pattern.
```

`server/internal/platform/httpserver/apitest/operations_test.go`（对 Task 2 版本的差异）：

```diff
--- a/server/internal/platform/httpserver/apitest/operations_test.go
+++ b/server/internal/platform/httpserver/apitest/operations_test.go
@@ -91,6 +91,49 @@
 	}
 }
 
+// Only the parameters whose Go type rejects some strings get a case: the
+// uuid path parameter and the integer, not the enum or the free string. The
+// others keep their example values.
+func TestParamCases(t *testing.T) {
+	ops := contractFrom(t, bodiesContract).Operations()
+
+	got := ops[1].ParamCases()
+
+	want := []ParamCase{
+		{"wrong thing_id", "/api/v0/things/not-a-uuid?limit=1&view=full", "thing_id"},
+		{"wrong limit", "/api/v0/things/00000000-0000-0000-0000-000000000000?limit=not-a-number&view=full", "limit"},
+	}
+	if !slices.Equal(got, want) {
+		t.Errorf("ParamCases() =\n%q\nwant\n%q", got, want)
+	}
+	if got := ops[0].ParamCases(); got != nil {
+		t.Errorf("ParamCases() without parameters = %q, want none", got)
+	}
+}
+
+// An optional parameter gets its case too, set only there.
+func TestParamCasesOfAnOptionalParameter(t *testing.T) {
+	op := contractFrom(t, `
+openapi: 3.1.0
+info: {title: params, version: v0}
+paths:
+  /api/v0/x:
+    get:
+      parameters:
+        - {name: flag, in: query, schema: {type: boolean}}
+        - {name: at, in: query, schema: {type: string, format: date-time}}
+      responses: {'204': {description: none}}
+`).Operations()[0]
+
+	want := []ParamCase{
+		{"wrong flag", "/api/v0/x?flag=not-a-boolean", "flag"},
+		{"wrong at", "/api/v0/x?at=not-a-date-time", "at"},
+	}
+	if got := op.ParamCases(); !slices.Equal(got, want) || op.Target() != "/api/v0/x" {
+		t.Errorf("ParamCases() = %q, Target() = %q; want %q and no query", got, op.Target(), want)
+	}
+}
+
 func TestBodyCases(t *testing.T) {
 	post := contractFrom(t, bodiesContract).Operations()[2]
 
```

- [ ] **Step 5: HTTP 适配器和接线**

`server/internal/modules/identity/adapter/http/handler.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/modules/identity/adapter/http/handler.go
+++ b/server/internal/modules/identity/adapter/http/handler.go
@@ -9,6 +9,7 @@
 	"log/slog"
 	"net/netip"
 	"time"
+	"uuid"
 
 	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/http/gen"
 	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
@@ -41,13 +42,31 @@
 	Execute(ctx context.Context) (domain.User, error)
 }
 
+// ListAPITokensUseCase is app.ListAPITokens.
+type ListAPITokensUseCase interface {
+	Execute(ctx context.Context, limit *int, cursor *string) (app.APITokenPage, error)
+}
+
+// CreateAPITokenUseCase is app.CreateAPIToken.
+type CreateAPITokenUseCase interface {
+	Execute(ctx context.Context, spec domain.APITokenSpec) (app.CreatedAPIToken, error)
+}
+
+// RevokeAPITokenUseCase is app.RevokeAPIToken.
+type RevokeAPITokenUseCase interface {
+	Execute(ctx context.Context, id uuid.UUID) error
+}
+
 // UseCases are the use cases behind the module's operations.
 type UseCases struct {
-	Register RegisterUseCase
-	Login    LoginUseCase
-	Refresh  RefreshUseCase
-	Logout   LogoutUseCase
-	GetMe    GetMeUseCase
+	Register       RegisterUseCase
+	Login          LoginUseCase
+	Refresh        RefreshUseCase
+	Logout         LogoutUseCase
+	GetMe          GetMeUseCase
+	ListAPITokens  ListAPITokensUseCase
+	CreateAPIToken CreateAPITokenUseCase
+	RevokeAPIToken RevokeAPITokenUseCase
 }
 
 // Settings are what the handler applies around the use cases.
```

`server/internal/modules/identity/adapter/http/api_tokens.go`（新文件）：

```go
package httpadapter

import (
	"context"
	"time"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// ListAPITokens serves GET /api/v0/me/api-tokens.
func (h handler) ListAPITokens(ctx context.Context, req gen.ListAPITokensRequestObject) (gen.ListAPITokensResponseObject, error) {
	page, err := h.uc.ListAPITokens.Execute(ctx, req.Params.Limit, req.Params.Cursor)
	if err != nil {
		return nil, err
	}
	out := gen.ListAPITokens200JSONResponse{Data: make([]gen.APIToken, len(page.Tokens)), NextCursor: nullable.NewNullNullable[string]()}
	for i, t := range page.Tokens {
		out.Data[i] = gen.APIToken{
			ID: t.ID, Label: t.Label, Description: t.Description,
			ExpiredAt: nullableTime(t.ExpiredAt), LastUsed: nullableTime(t.LastUsed), CreatedAt: t.CreatedAt,
		}
	}
	if page.NextCursor != "" {
		out.NextCursor = nullable.NewNullableWithValue(page.NextCursor)
	}
	return out, nil
}

// CreateAPIToken serves POST /api/v0/me/api-tokens.
func (h handler) CreateAPIToken(ctx context.Context, req gen.CreateAPITokenRequestObject) (gen.CreateAPITokenResponseObject, error) {
	spec := domain.APITokenSpec{Label: req.Body.Label}
	if req.Body.Description != nil {
		spec.Description = *req.Body.Description
	}
	if expires, err := req.Body.ExpiredAt.Get(); err == nil {
		spec.ExpiredAt = &expires
	}
	t, err := h.uc.CreateAPIToken.Execute(ctx, spec)
	if err != nil {
		return nil, err
	}
	return gen.CreateAPIToken201JSONResponse{
		ID: t.ID, Label: t.Label, Description: t.Description,
		ExpiredAt: nullableTime(t.ExpiredAt), LastUsed: nullableTime(t.LastUsed), CreatedAt: t.CreatedAt,
		Token: t.Token,
	}, nil
}

// RevokeAPIToken serves DELETE /api/v0/api-tokens/{token_id}.
func (h handler) RevokeAPIToken(ctx context.Context, req gen.RevokeAPITokenRequestObject) (gen.RevokeAPITokenResponseObject, error) {
	if err := h.uc.RevokeAPIToken.Execute(ctx, req.TokenID); err != nil {
		return nil, err
	}
	return gen.RevokeAPIToken204Response{}, nil
}

// nullableTime is t as a required field that may be null: the zero
// Nullable is "unspecified" and would marshal as "", so nil is set to null
// explicitly.
func nullableTime(t *time.Time) nullable.Nullable[time.Time] {
	if t == nil {
		return nullable.NewNullNullable[time.Time]()
	}
	return nullable.NewNullableWithValue(*t)
}
```

`server/internal/modules/identity/adapter/http/handler_test.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/modules/identity/adapter/http/handler_test.go
+++ b/server/internal/modules/identity/adapter/http/handler_test.go
@@ -65,10 +65,13 @@
 // fakes are the use cases behind a test server; newServer puts an idle fake
 // in place of each one left nil.
 type fakes struct {
-	register *fakeRegister
-	login    *fakeLogin
-	refresh  *fakeRefresh
-	logout   *fakeLogout
+	register    *fakeRegister
+	login       *fakeLogin
+	refresh     *fakeRefresh
+	logout      *fakeLogout
+	listTokens  *fakeListTokens
+	createToken *fakeCreateToken
+	revokeToken *fakeRevokeToken
 }
 
 // newServer serves the module with limits no test here reaches.
@@ -119,8 +122,18 @@
 	if f.logout == nil {
 		f.logout = &fakeLogout{}
 	}
+	if f.listTokens == nil {
+		f.listTokens = &fakeListTokens{}
+	}
+	if f.createToken == nil {
+		f.createToken = &fakeCreateToken{}
+	}
+	if f.revokeToken == nil {
+		f.revokeToken = &fakeRevokeToken{}
+	}
 	httpadapter.Register(router, api, httpadapter.UseCases{
 		Register: f.register, Login: f.login, Refresh: f.refresh, Logout: f.logout, GetMe: fakeGetMe{},
+		ListAPITokens: f.listTokens, CreateAPIToken: f.createToken, RevokeAPIToken: f.revokeToken,
 	}, s)
 	return router
 }
```

`server/internal/modules/identity/adapter/http/api_tokens_test.go`（新文件）：

```go
package httpadapter_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var tokenID = uuid.MustParse("0199a2b4-0000-7000-8000-000000000003")

type fakeListTokens struct {
	calls  int
	limit  *int
	cursor *string
	page   app.APITokenPage
	err    error
}

func (f *fakeListTokens) Execute(_ context.Context, limit *int, cursor *string) (app.APITokenPage, error) {
	f.calls++
	f.limit, f.cursor = limit, cursor
	return f.page, f.err
}

type fakeCreateToken struct {
	calls   int
	spec    domain.APITokenSpec
	created app.CreatedAPIToken
	err     error
}

func (f *fakeCreateToken) Execute(_ context.Context, spec domain.APITokenSpec) (app.CreatedAPIToken, error) {
	f.calls++
	f.spec = spec
	return f.created, f.err
}

type fakeRevokeToken struct {
	calls int
	id    uuid.UUID
	err   error
}

func (f *fakeRevokeToken) Execute(_ context.Context, id uuid.UUID) error {
	f.calls++
	f.id = id
	return f.err
}

// withToken is req with the bearer token fakeAuth accepts.
func withToken(req *http.Request) *http.Request {
	req.Header.Set("Authorization", "Bearer valid")
	return req
}

func TestCreateAPITokenAnswers201WithTheToken(t *testing.T) {
	expires := created.Add(7 * 24 * time.Hour)
	create := &fakeCreateToken{created: app.CreatedAPIToken{
		APIToken: domain.APIToken{ID: tokenID, Label: "deploy", Description: "ci", ExpiredAt: &expires, CreatedAt: created},
		Token:    "nrv_pat_x",
	}}
	req := withToken(postJSON("/api/v0/me/api-tokens", `{"label":"deploy","description":"ci","expired_at":"2026-10-02T10:00:00.123456Z"}`))
	apitest.Load(t).CheckRequest(t, req)

	res, body := do(t, newServer(t, fakes{createToken: create}), req)

	want := `{"created_at":"2026-09-25T10:00:00.123456Z","description":"ci","expired_at":"2026-10-02T10:00:00.123456Z",` +
		`"id":"` + tokenID.String() + `","label":"deploy","last_used":null,"token":"nrv_pat_x"}` + "\n"
	if res.StatusCode != http.StatusCreated || body != want {
		t.Errorf("POST /me/api-tokens = %d %s, want 201 %s", res.StatusCode, body, want)
	}
	if s := create.spec; s.Label == nil || *s.Label != "deploy" || s.Description != "ci" || s.ExpiredAt == nil || !s.ExpiredAt.Equal(expires) {
		t.Errorf("use case got %+v, want the label, description and expiry sent", s)
	}
}

// Absent fields and a null expiry reach the use case as absent: a
// generated label, no description, no expiry.
func TestCreateAPITokenWithoutFields(t *testing.T) {
	for _, body := range []string{`{}`, `{"expired_at":null}`} {
		create := &fakeCreateToken{created: app.CreatedAPIToken{APIToken: domain.APIToken{ID: tokenID, Label: "l", CreatedAt: created}, Token: "nrv_pat_x"}}

		res, out := do(t, newServer(t, fakes{createToken: create}), withToken(postJSON("/api/v0/me/api-tokens", body)))

		if res.StatusCode != http.StatusCreated || !strings.Contains(out, `"expired_at":null`) {
			t.Errorf("POST %s = %d %s, want 201 with a null expiry", body, res.StatusCode, out)
		}
		if s := create.spec; s.Label != nil || s.Description != "" || s.ExpiredAt != nil {
			t.Errorf("POST %s: use case got %+v, want nothing set", body, s)
		}
	}
}

func TestCreateAPITokenInvalid(t *testing.T) {
	create := &fakeCreateToken{err: shared.Invalid(shared.FieldError{Field: "expired_at", Code: shared.FieldMustBeFuture, Message: "must be in the future"})}

	res, body := do(t, newServer(t, fakes{createToken: create}), withToken(postJSON("/api/v0/me/api-tokens", `{"expired_at":"2020-01-01T00:00:00Z"}`)))

	if res.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(body, `"field":"expired_at","code":"must_be_future"`) {
		t.Errorf("response = %d %s, want 422 on expired_at", res.StatusCode, body)
	}
}

func TestListAPITokens(t *testing.T) {
	used := created.Add(time.Hour)
	list := &fakeListTokens{page: app.APITokenPage{
		Tokens:     []domain.APIToken{{ID: tokenID, Label: "deploy", LastUsed: &used, CreatedAt: created}},
		NextCursor: "next",
	}}
	req := withToken(httptest.NewRequest(http.MethodGet, "/api/v0/me/api-tokens?limit=2&cursor=abc", nil))
	apitest.Load(t).CheckRequest(t, req)

	res, body := do(t, newServer(t, fakes{listTokens: list}), req)

	want := `{"data":[{"created_at":"2026-09-25T10:00:00.123456Z","description":"","expired_at":null,"id":"` + tokenID.String() +
		`","label":"deploy","last_used":"2026-09-25T11:00:00.123456Z"}],"next_cursor":"next"}` + "\n"
	if res.StatusCode != http.StatusOK || body != want {
		t.Errorf("GET /me/api-tokens = %d %s, want 200 %s", res.StatusCode, body, want)
	}
	if list.limit == nil || *list.limit != 2 || list.cursor == nil || *list.cursor != "abc" {
		t.Errorf("use case got limit %v cursor %v, want 2 and abc", list.limit, list.cursor)
	}
}

// Without parameters the use case gets neither; the last page's cursor is
// null.
func TestListAPITokensFirstAndLastPage(t *testing.T) {
	list := &fakeListTokens{page: app.APITokenPage{Tokens: []domain.APIToken{}}}

	res, body := do(t, newServer(t, fakes{listTokens: list}), withToken(httptest.NewRequest(http.MethodGet, "/api/v0/me/api-tokens", nil)))

	if want := `{"data":[],"next_cursor":null}` + "\n"; res.StatusCode != http.StatusOK || body != want {
		t.Errorf("GET /me/api-tokens = %d %s, want 200 %s", res.StatusCode, body, want)
	}
	if list.limit != nil || list.cursor != nil {
		t.Errorf("use case got limit %v cursor %v, want neither", list.limit, list.cursor)
	}
}

// The handler exit: limit out of range is 422, a foreign cursor 400.
func TestListAPITokensProblems(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"limit out of range", shared.Invalid(shared.FieldError{Field: "limit", Code: shared.FieldOutOfRange, Message: "must be between 1 and 100"}),
			422, `"code":"validation_failed"`},
		{"foreign cursor", shared.InvalidCursor(), 400, `"errors":[{"field":"cursor","code":"invalid_format"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, body := do(t, newServer(t, fakes{listTokens: &fakeListTokens{err: tt.err}}),
				withToken(httptest.NewRequest(http.MethodGet, "/api/v0/me/api-tokens?limit=0", nil)))

			if res.StatusCode != tt.status || !strings.Contains(body, tt.want) {
				t.Errorf("response = %d %s, want %d with %s", res.StatusCode, body, tt.status, tt.want)
			}
		})
	}
}

func TestRevokeAPIToken(t *testing.T) {
	revoke := &fakeRevokeToken{}
	req := withToken(httptest.NewRequest(http.MethodDelete, "/api/v0/api-tokens/"+tokenID.String(), nil))
	apitest.Load(t).CheckRequest(t, req)

	res, body := do(t, newServer(t, fakes{revokeToken: revoke}), req)

	if res.StatusCode != http.StatusNoContent || body != "" || revoke.id != tokenID {
		t.Errorf("DELETE /api-tokens/{id} = %d %q, use case got %v; want 204 for %v", res.StatusCode, body, revoke.id, tokenID)
	}
}

func TestRevokeAnAPITokenThatIsNotTheCallers(t *testing.T) {
	revoke := &fakeRevokeToken{err: domain.ErrAPITokenNotFound}

	res, body := do(t, newServer(t, fakes{revokeToken: revoke}), withToken(httptest.NewRequest(http.MethodDelete, "/api/v0/api-tokens/"+tokenID.String(), nil)))

	if res.StatusCode != http.StatusNotFound || !strings.Contains(body, `"code":"identity.api_token_not_found"`) {
		t.Errorf("response = %d %s, want 404 identity.api_token_not_found", res.StatusCode, body)
	}
}

// The parameter binding exit (M0-P3 handoff 2): a parameter that does not
// bind is 400 with the parameter as the field, before authentication (M2
// design 3.6) and without Go's words; the use case never runs.
func TestParametersThatDoNotBind(t *testing.T) {
	tests := []struct {
		name, method, target, field string
	}{
		{"limit not an integer", http.MethodGet, "/api/v0/me/api-tokens?limit=abc", "limit"},
		{"token_id not a uuid", http.MethodDelete, "/api/v0/api-tokens/not-a-uuid", "token_id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list, revoke := &fakeListTokens{}, &fakeRevokeToken{}

			res, body := do(t, newServer(t, fakes{listTokens: list, revokeToken: revoke}), httptest.NewRequest(tt.method, tt.target, nil))

			want := `{"status":400,"code":"bad_request","title":"Bad Request","detail":"The request parameters do not match the API description.",` +
				`"errors":[{"field":"` + tt.field + `","code":"invalid_format","message":"has the wrong type or format"}]}` + "\n"
			if res.StatusCode != http.StatusBadRequest || body != want {
				t.Errorf("response = %d %s, want 400 %s", res.StatusCode, body, want)
			}
			if list.calls+revoke.calls != 0 {
				t.Errorf("the use case ran %d times, want never", list.calls+revoke.calls)
			}
		})
	}
}
```

`server/internal/modules/identity/module.go`（对 Task 6 版本的差异）：

```diff
--- a/server/internal/modules/identity/module.go
+++ b/server/internal/modules/identity/module.go
@@ -1,6 +1,6 @@
 // Package identity is the accounts module (M2 design 3.3, 6.2): accounts,
-// profiles, sessions and, from later phases, personal access tokens. It
-// brings registration, login, refresh, logout, GET /me and the
+// profiles, sessions and personal access tokens. It brings registration,
+// login, refresh, logout, the caller's account and tokens, and the
 // authentication every other operation goes through.
 package identity
 
@@ -84,6 +84,7 @@
 		return nil, fmt.Errorf("hash the dummy password: %w", err)
 	}
 	store := postgresadapter.New(d.Pool)
+	lock := app.CredentialLock{Locker: store, Sessions: store, APITokens: store}
 	tokens := signing.NewAccessTokens(keys)
 	issuance := app.Issuance{
 		Tokens:     tokens,
@@ -101,9 +102,14 @@
 				Accounts: store, Locker: store, Passwords: store, Sessions: store, Hasher: hasher, Tx: d.Tx,
 				Issuance: issuance, Clock: d.Clock, Logger: d.Logger, DummyHash: dummy,
 			}),
-			Refresh: app.NewRefresh(app.RefreshDeps{Sessions: store, Tx: d.Tx, Issuance: issuance, Clock: d.Clock, Logger: d.Logger}),
-			Logout:  app.NewLogout(store, d.Clock, d.Logger),
-			GetMe:   app.NewGetMe(store),
+			Refresh:       app.NewRefresh(app.RefreshDeps{Sessions: store, Tx: d.Tx, Issuance: issuance, Clock: d.Clock, Logger: d.Logger}),
+			Logout:        app.NewLogout(store, d.Clock, d.Logger),
+			GetMe:         app.NewGetMe(store),
+			ListAPITokens: app.NewListAPITokens(store),
+			CreateAPIToken: app.NewCreateAPIToken(app.CreateAPITokenDeps{
+				Lock: lock, Tokens: store, Tx: d.Tx, Clock: d.Clock, Logger: d.Logger,
+			}),
+			RevokeAPIToken: app.NewRevokeAPIToken(store, d.Clock, d.Logger),
 		},
 		settings: httpadapter.Settings{
 			Limits:          httpadapter.Limits(d.RateLimits),
```

- [ ] **Step 6: 整程序测试**

`server/internal/bootstrap/contract_test.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/bootstrap/contract_test.go
+++ b/server/internal/bootstrap/contract_test.go
@@ -130,6 +130,44 @@
 	}
 }
 
+// 5. Every path or query parameter whose Go type rejects some strings
+// answers a wrong value with 400 that names it (M2 design 3.11, M0-P3
+// handoff 2), before authentication: parameters bind first (3.6), so no
+// token is sent.
+func TestParametersThatDoNotBindAnswer400(t *testing.T) {
+	contract := apitest.Load(t)
+	base := startApp(t, testConfig(t, unreachableDB, false), fstest.MapFS{})
+
+	cases := 0
+	for _, op := range contract.Operations() {
+		for _, c := range op.ParamCases() {
+			cases++
+			t.Run(op.Pattern()+"/"+c.Name, func(t *testing.T) {
+				req := newRequest(t, op.Method, base+c.Target, "", nil)
+				res, body := send(t, req)
+
+				contract.CheckResponse(t, req, res)
+				var p struct {
+					Code   string                 `json:"code"`
+					Errors []apitest.FieldProblem `json:"errors"`
+				}
+				if err := json.Unmarshal(body, &p); err != nil {
+					t.Fatalf("decode %s: %v", body, err)
+				}
+				want := []apitest.FieldProblem{{Field: c.Field, Code: "invalid_format"}}
+				if res.StatusCode != http.StatusBadRequest || p.Code != "bad_request" || !slices.Equal(p.Errors, want) {
+					t.Errorf("%s = %d %s, want 400 bad_request with %v", c.Target, res.StatusCode, body, want)
+				}
+			})
+		}
+	}
+	// The contract has such parameters (limit, token_id): none found means
+	// the derivation broke, not that there is nothing to test.
+	if cases == 0 {
+		t.Fatal("no parameter case derived from the contract")
+	}
+}
+
 // Every operation's default response, the problem, declares the two headers
 // a problem may carry: Retry-After and WWW-Authenticate. Each module declares
 // its own Problem response (spec P2 3 item 13), so a module that leaves one
```

`server/internal/bootstrap/auth_test.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/bootstrap/auth_test.go
+++ b/server/internal/bootstrap/auth_test.go
@@ -66,6 +66,53 @@
 	}
 }
 
+// A personal access token made through the app authenticates as its
+// account until it is revoked (M2 design 3.4, 3.5). It can make another
+// token, which the credential lock checks it for, and revoke itself.
+func TestAPersonalAccessTokenAuthenticates(t *testing.T) {
+	contract := apitest.Load(t)
+	base := startApp(t, testConfig(t, pgtest.NewDatabase(t), false), migrations.FS())
+	access := registerAccount(t, contract, base, "pat@example.com").AccessToken
+	status := func(method, path, token string) int {
+		t.Helper()
+		req := newRequest(t, method, base+path, token, nil)
+		res, _ := send(t, req)
+		contract.CheckResponse(t, req, res)
+		return res.StatusCode
+	}
+
+	first := createPAT(t, contract, base, access)
+	second := createPAT(t, contract, base, first.Token)
+	me, revoked, after, other := status(http.MethodGet, "/api/v0/me", first.Token),
+		status(http.MethodDelete, "/api/v0/api-tokens/"+first.ID, first.Token),
+		status(http.MethodGet, "/api/v0/me", first.Token),
+		status(http.MethodGet, "/api/v0/me", second.Token)
+
+	if me != http.StatusOK || revoked != http.StatusNoContent || after != http.StatusUnauthorized || other != http.StatusOK {
+		t.Errorf("GET /me %d, revoke itself %d, GET /me after %d, the other token %d; want 200, 204, 401, 200", me, revoked, after, other)
+	}
+}
+
+// pat is the ApiTokenCreated answer.
+type pat struct {
+	ID    string `json:"id"`
+	Token string `json:"token"`
+}
+
+// createPAT makes a personal access token with the bearer token given.
+func createPAT(t *testing.T, contract *apitest.Contract, base, bearer string) pat {
+	t.Helper()
+	req := newRequest(t, http.MethodPost, base+"/api/v0/me/api-tokens", bearer, []byte(`{"label":"ci"}`))
+	contract.CheckRequest(t, req)
+	res, body := send(t, req)
+	contract.CheckResponse(t, req, res)
+	var created pat
+	if err := json.Unmarshal(body, &created); err != nil || res.StatusCode != http.StatusCreated || !strings.HasPrefix(created.Token, "nrv_pat_") {
+		t.Fatalf("POST /me/api-tokens = %d %s, want 201 with a token", res.StatusCode, body)
+	}
+	return created
+}
+
 // signedBy reports whether the JWT's EdDSA signature verifies with the
 // public half of the PKCS#8 key.
 func signedBy(t *testing.T, token, keyPEM string) bool {
```

- [ ] **Step 7: lint、测试和端到端**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。archtest 的 `TestNerveBinaryLinksNoBannedModule` 在 runtime 进入程序之后仍然通过（google/uuid 只经 runtime 导入），`TestGeneratedCodeUsesTheStandardUUID` 通过（identity 的 `type-mapping` 把 `uuid` 映射到标准库）。

Run: `make lint-web`
Expected: 通过（`schema.gen.ts` 多出三个操作）。

Run: `make e2e`
Expected: P1、P2 的故事全部通过。

- [ ] **Step 8: 提交**

```bash
git add api web/packages/api-client/src/schema.gen.ts server/go.mod server/go.sum server/internal/platform/httpserver server/internal/modules/identity/adapter/http server/internal/modules/identity/module.go server/internal/bootstrap/contract_test.go server/internal/bootstrap/auth_test.go
```
```bash
git commit -m "feat(M2/P3a): create, list and revoke personal access tokens; the parameter binding whole-program test

The first operations with parameters bring oapi-codegen/runtime v1.7.0.
Lists page with the shared cursor envelope; a parameter that does not
bind is 400 naming it, before authentication, which the fifth
whole-program test derives from the contract.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run: `make gen-check`
Expected: 没有差异。

**Done when:** 令牌的 handler 测试、`TestParametersThatDoNotBindAnswer400`、`TestAPersonalAccessTokenAuthenticates` 通过；传递依赖测试和生成代码的 uuid 规则在 runtime 进入之后通过；五个生成物与上表相同；`make e2e` 中 P1、P2 的故事通过；`make gen-check` 干净。

---

### Task 8: 资料和偏好的领域规则与存储

**Files:**
- Modify: `server/internal/modules/identity/domain/user.go`、`user_test.go`
- Create: `server/internal/modules/identity/domain/profile.go`、`profile_test.go`、`url.go`、`url_test.go`
- Modify（过渡：Task 10、11 加上修改密码和停用的查询）: `server/internal/modules/identity/adapter/postgres/queries/users.sql`、`queries/profiles.sql`、`users.go`、`profiles.go`
- Create（过渡：Task 11 加上停用的测试）: `server/internal/modules/identity/adapter/postgres/account_test.go`
- Modify: `server/internal/modules/identity/app/ports.go`（过渡）
- Generate: `server/internal/modules/identity/adapter/postgres/gen/users.sql.go`、`profiles.sql.go`

**Interfaces:**
- Produces（spec 2.12，M2 设计 3.11、3.14、4.2、4.3）：
  - `domain.UserPatch{FirstName, LastName, DisplayName, Timezone *string}`、`domain.CheckUserPatch(p) error`：名、姓至多 255 个字符、不含网址（Plane 的 `contains_url`，`contains_url` 码）；显示名 1–255 个字符；都不含 NUL；时区是 Go 的 `time` 包能加载的 IANA 名字，不是 `Local` 或空串；一次报出全部问题；
  - `domain.containsURL(s)`：Plane `url.py:12-53` 的移植，按 Python 的 `\s` 和 `re.IGNORECASE` 的字母范围写成 RE2，长度按字符计；
  - `domain.Profile`、`domain.OnboardingSteps`、`domain.OnboardingStepsPatch`（`JSON()` 只含设了的步骤）、`domain.ProfilePatch`（`LastWorkspaceSet` 区分"不改"和"清空"）、`domain.CheckProfilePatch(p) error`：主题是五个之一、语言是 `en` 或 `zh-CN`（`invalid_format`），每周第一天 0–6（`out_of_range`）；结构（步骤的键和类型）在边界上由 bodyshape 检查；
  - 端口：`UserUpdater`、`ProfileReader`、`ProfileUpdater`；
  - 查询：`UpdateUser`、`UpdateProfile` 按 `set_*` 标志只改传入的列（M2 设计 3.14 的 `CASE WHEN`），`updated_at` 在 `SET` 的第一位（P2 spec 2.10 的 sqlc 推断规则）；`onboarding_step = onboarding_step || patch::jsonb` 在语句里合并；`GetProfile`；
  - `Store.UpdateUser`、`GetProfile`、`UpdateProfile`（没有这一行时 `app.ErrNotFound`）。

**Tests:**
- `user_test.go`：`TestCheckUserPatchAcceptsAValidPatch`；`TestCheckUserPatchReportsEveryField`（10 个：名里的网址、姓里的网址、长名、姓里的 NUL、空显示名、长显示名、未知时区、`Local`、空时区，以及一次报出四个）。
- `url_test.go`：`TestContainsURLAsPlane`（41 个，每个的期望值都是 Plane 的 `contains_url` 在 Python 3 中对同一字符串的答案，spec 附录 A）。
- `profile_test.go`：`TestCheckProfilePatchAcceptsAValidPatch`（五个主题、两种语言、0 和 6）；`TestCheckProfilePatchReportsEveryField`；`TestOnboardingStepsPatchJSON`。
- `account_test.go`（真实数据库）：`TestUpdateUser`（只改传入的列；`updated_at` 是用例的时间）；`TestUpdateUnknownUser`；`TestGetProfileReadsTheDefaults`；`TestUpdateProfile`（全部字段；空补丁不改任何偏好；只清空工作区）；`TestUpdateUnknownProfile`；`TestUpdateProfileMergesConcurrentSteps`（M2 设计 3.14 的并发合并：第一个事务改 `profile_complete` 后持锁，等到第二条语句确实在等锁（`pg_stat_activity`）才放开，两个键都保留；每个等待都有 10 秒的期限，超时即失败）。

- [ ] **Step 1: 资料的规则**

`server/internal/modules/identity/domain/url.go`（新文件）：

```go
package domain

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// pySpace is what Python's \s matches in a str pattern (str.isspace()):
// Go's \s is ASCII only and lacks \v.
const pySpace = `\t-\r\x1c-\x20\x85\xa0\x{1680}\x{2000}-\x{200a}\x{2028}\x{2029}\x{202f}\x{205f}\x{3000}`

// pyLetters is [a-zA-Z] under Python's re.IGNORECASE, which also matches
// İ, ı, ſ and K (Python's re documentation). Go's (?i) adds ſ and K by case
// folding; İ and ı are listed.
const pyLetters = `a-zA-Z\x{0130}\x{0131}`

// urlPattern is Plane's URL_PATTERN (plane/apps/api/plane/utils/url.py:12-23)
// for RE2: an http(s) address, a www. host, a dotted host name with a TLD
// of 2–6 letters, or an IPv4 address, anywhere in the text.
var urlPattern = regexp.MustCompile(`(?i)(?:` +
	`https?://[^` + pySpace + `]+` +
	`|www\.[` + pyLetters + `0-9](?:[` + pyLetters + `0-9-]{0,61}[` + pyLetters + `0-9])?(?:\.[` + pyLetters + `0-9](?:[` + pyLetters + `0-9-]{0,61}[` + pyLetters + `0-9])?)*` +
	`|(?:[` + pyLetters + `0-9](?:[` + pyLetters + `0-9-]{0,61}[` + pyLetters + `0-9])?\.)+[` + pyLetters + `]{2,6}` +
	`|(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)` +
	`)`)

// containsURL is Plane's contains_url (plane/apps/api/plane/utils/url.py:
// 26-53), which it applies to first and last names: text of more than 1000
// characters is not looked at, and each line only up to its 500th
// character.
func containsURL(s string) bool {
	if utf8.RuneCountInString(s) > 1000 {
		return false
	}
	for line := range strings.SplitSeq(s, "\n") {
		if runes := []rune(line); len(runes) > 500 {
			line = string(runes[:500])
		}
		if urlPattern.MatchString(line) {
			return true
		}
	}
	return false
}
```

`server/internal/modules/identity/domain/url_test.go`（新文件）：

```go
package domain

import (
	"strings"
	"testing"
)

// The expectations are what Plane's contains_url answers for the same
// strings (Python 3, plane/apps/api/plane/utils/url.py).
func TestContainsURLAsPlane(t *testing.T) {
	a := strings.Repeat
	tests := []struct {
		name, s string
		want    bool
	}{
		{"https address", "https://x", true},
		{"scheme only", "http://", false},
		{"scheme then a no-break space", "http://\u00a0x", false},
		{"scheme then a vertical tab", "http://\vx", false},
		{"scheme then an information separator", "http://\x1cx", false},
		{"upper case", "HTTPS://X", true},
		{"ftp", "ftp://x", false},
		{"dotted name", "John.Smith", true},
		{"plain name", "Ann", false},
		{"hyphen", "Mary-Ann", false},
		{"apostrophe", "O'Brien", false},
		{"www host", "www.x", true},
		{"upper-case www", "WWW.Example", true},
		{"one-letter TLD", "a.b", false},
		{"two-letter TLD", "a.bc", true},
		{"label ending in a hyphen", "a-.bc", false},
		{"TLD with a hyphen", "a.b-c", false},
		{"leading hyphen", "-a.bc", true},
		{"IPv4", "1.2.3.4", true},
		{"IPv4 inside a longer number", "999.1.1.1", true},
		{"decimal", "3.14", false},
		{"three numbers", "1.2.3", false},
		{"dotted capital I", "\u0130.ab", true},
		{"dotless i", "\u0131.ab", true},
		{"long s", "\u017f.ab", true},
		{"Kelvin sign", "\u212a.ab", true},
		{"e acute", "\u00e9.ab", false},
		{"letters before an address", "\u00e9l\u00e8ve.com", true},
		{"after an Ogham space", "\u1680http://x", true},
		{"second line", "x\nhttps://y", true},
		{"after a line separator", "x\u2028https://y", true},
		{"no-break space after it", "x.co\u00a0", true},
		{"address ending at character 500", a("a", 490) + " https://x", true},
		{"address cut at character 500", a("a", 491) + " https://x", false},
		{"1000 characters", "x.io\n" + a("a", 995), true},
		{"1001 characters", "x.io\n" + a("a", 996), false},
		{"1000 characters, not bytes", "x.io\n" + a("\u00e9", 995), true},
		{"address ending at character 500, not byte 500", a("\u00e9", 490) + " https://x", true},
		{"address cut at character 500, not byte 500", a("\u00e9", 491) + " https://x", false},
		{"address within character 500, beyond byte 500", a("\u00e9", 300) + " https://x" + a("a", 300), true},
		{"a long first line, an address on the second", a("a", 450) + "\n" + a("b", 60) + " x.io", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsURL(tt.s); got != tt.want {
				t.Errorf("containsURL(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}
```

`server/internal/modules/identity/domain/user.go`（完整内容）：

```go
// Package domain holds the identity module's rules (M2 design 6.2): pure
// functions and values, no I/O.
package domain

import (
	"time"
	"unicode/utf8"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// User is an account as the API shows it.
type User struct {
	ID          uuid.UUID
	Email       string
	FirstName   string
	LastName    string
	DisplayName string
	Timezone    string
	CreatedAt   time.Time
}

// UserPatch is a partial update of an account: a nil field stays as it is.
// The e-mail address is not in it: only the administrator's command changes
// it (M2 decision 1).
type UserPatch struct {
	FirstName   *string
	LastName    *string
	DisplayName *string
	Timezone    *string
}

// maxNameLength is the length of users' name columns, varchar(255), in
// characters.
const maxNameLength = 255

// CheckUserPatch checks p (M2 design 4.2): first and last names of at most
// 255 characters without an address in them, as Plane checks them
// (contains_url); a display name of 1–255 characters; no NUL, which the
// database cannot store; and a time zone that Go's time package knows, not
// "Local". Every problem is reported at once, as one 422
// validation_failed.
func CheckUserPatch(p UserPatch) error {
	var fields []shared.FieldError
	for _, name := range []struct {
		field string
		value *string
	}{{"first_name", p.FirstName}, {"last_name", p.LastName}} {
		if name.value == nil {
			continue
		}
		if f := checkText(name.field, *name.value, false, maxNameLength); f != nil {
			fields = append(fields, *f)
		} else if containsURL(*name.value) {
			fields = append(fields, shared.FieldError{Field: name.field, Code: shared.FieldContainsURL, Message: "must not contain a web address"})
		}
	}
	if p.DisplayName != nil {
		if f := checkText("display_name", *p.DisplayName, true, maxNameLength); f != nil {
			fields = append(fields, *f)
		}
	}
	if p.Timezone != nil && !validTimezone(*p.Timezone) {
		fields = append(fields, shared.FieldError{Field: "user_timezone", Code: shared.FieldInvalidFormat, Message: "is not a known time zone"})
	}
	if len(fields) > 0 {
		return shared.Invalid(fields...)
	}
	return nil
}

// validTimezone reports whether name is an IANA time zone that Go's time
// package loads (M2 design 4.2): "UTC" and every zone of the embedded
// tzdata, not "Local" or "", which LoadLocation takes for the host's zone
// and for UTC.
func validTimezone(name string) bool {
	if name == "" || name == "Local" {
		return false
	}
	_, err := time.LoadLocation(name)
	return err == nil
}

// NewAccount checks the e-mail address and the password of a new account
// and returns the normalized address. Every problem is reported at once, as
// one 422 validation_failed.
func NewAccount(rules *PasswordRules, email, password string) (string, error) {
	email = NormalizeEmail(email)
	var fields []shared.FieldError
	switch {
	case email == "":
		fields = append(fields, shared.FieldError{Field: "email", Code: shared.FieldRequired, Message: "is required"})
	case utf8.RuneCountInString(email) > MaxEmailLength:
		fields = append(fields, shared.FieldError{Field: "email", Code: shared.FieldTooLong, Message: "must be at most 255 characters"})
	case !ValidEmail(email):
		fields = append(fields, shared.FieldError{Field: "email", Code: shared.FieldInvalidFormat, Message: "is not a valid e-mail address"})
	}
	if f := rules.Check("password", password, email); f != nil {
		fields = append(fields, *f)
	}
	if len(fields) > 0 {
		return "", shared.Invalid(fields...)
	}
	return email, nil
}
```

`server/internal/modules/identity/domain/user_test.go`（完整内容）：

```go
package domain

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestNewAccountNormalizesTheEmail(t *testing.T) {
	email, err := NewAccount(rules, "  Alice@Corp.COM ", "Tr0ub4dor&3")
	if err != nil || email != "alice@corp.com" {
		t.Errorf("NewAccount() = %q, %v; want alice@corp.com", email, err)
	}
}

// Every problem is reported at once, as one 422.
func TestNewAccountReportsEveryField(t *testing.T) {
	tests := []struct {
		name, email, password string
		want                  []shared.FieldError
	}{
		{"both empty", "", "", []shared.FieldError{
			{Field: "email", Code: "required", Message: "is required"},
			{Field: "password", Code: "required", Message: "is required"},
		}},
		{"bad address, weak password", "not-an-address", "short", []shared.FieldError{
			{Field: "email", Code: "invalid_format", Message: "is not a valid e-mail address"},
			{Field: "password", Code: "weak_password", Message: "must be 8-128 characters with an upper-case letter, a lower-case letter, a digit and one of " + passwordSpecials},
		}},
		{"too long", strings.Repeat("a", 250) + "@x.com", "Tr0ub4dor&3", []shared.FieldError{
			{Field: "email", Code: "too_long", Message: "must be at most 255 characters"},
		}},
		{"common password", "bob@corp.com", "Password1!", []shared.FieldError{
			{Field: "password", Code: "common_password", Message: "is too common"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAccount(rules, tt.email, tt.password)
			var se *shared.Error
			if !errors.As(err, &se) || se.Kind != shared.KindInvalid || se.Code != shared.CodeValidationFailed || !slices.Equal(se.Fields, tt.want) {
				t.Errorf("NewAccount() = %+v, want validation_failed with %+v", err, tt.want)
			}
		})
	}
}

func TestCheckUserPatchAcceptsAValidPatch(t *testing.T) {
	name, long, tz := "Ada", strings.Repeat("界", 255), "America/St_Johns"
	for _, p := range []UserPatch{
		{},
		{FirstName: &name, LastName: &long, DisplayName: &long, Timezone: &tz},
		{FirstName: ptr(""), LastName: ptr("")},
		{Timezone: ptr("UTC")},
	} {
		if err := CheckUserPatch(p); err != nil {
			t.Errorf("CheckUserPatch(%+v) = %v, want nil", p, err)
		}
	}
}

// Every problem is reported at once, as one 422.
func TestCheckUserPatchReportsEveryField(t *testing.T) {
	long := strings.Repeat("界", 256)
	tests := []struct {
		name  string
		patch UserPatch
		want  []shared.FieldError
	}{
		{"a web address in the first name", UserPatch{FirstName: ptr("see x.io")},
			[]shared.FieldError{{Field: "first_name", Code: "contains_url", Message: "must not contain a web address"}}},
		{"a web address in the last name", UserPatch{LastName: ptr("https://x")},
			[]shared.FieldError{{Field: "last_name", Code: "contains_url", Message: "must not contain a web address"}}},
		{"a long first name", UserPatch{FirstName: &long},
			[]shared.FieldError{{Field: "first_name", Code: "too_long", Message: "must be at most 255 characters"}}},
		{"NUL in the last name", UserPatch{LastName: ptr("a\x00")},
			[]shared.FieldError{{Field: "last_name", Code: "invalid_format", Message: "must not contain a NUL character"}}},
		{"an empty display name", UserPatch{DisplayName: ptr("")},
			[]shared.FieldError{{Field: "display_name", Code: "too_short", Message: "must not be empty"}}},
		{"a long display name", UserPatch{DisplayName: &long},
			[]shared.FieldError{{Field: "display_name", Code: "too_long", Message: "must be at most 255 characters"}}},
		{"an unknown time zone", UserPatch{Timezone: ptr("Mars/Olympus")},
			[]shared.FieldError{{Field: "user_timezone", Code: "invalid_format", Message: "is not a known time zone"}}},
		{"the host's zone", UserPatch{Timezone: ptr("Local")},
			[]shared.FieldError{{Field: "user_timezone", Code: "invalid_format", Message: "is not a known time zone"}}},
		{"no time zone", UserPatch{Timezone: ptr("")},
			[]shared.FieldError{{Field: "user_timezone", Code: "invalid_format", Message: "is not a known time zone"}}},
		{"all at once", UserPatch{FirstName: ptr("x.io"), LastName: &long, DisplayName: ptr(""), Timezone: ptr("Nowhere")}, []shared.FieldError{
			{Field: "first_name", Code: "contains_url", Message: "must not contain a web address"},
			{Field: "last_name", Code: "too_long", Message: "must be at most 255 characters"},
			{Field: "display_name", Code: "too_short", Message: "must not be empty"},
			{Field: "user_timezone", Code: "invalid_format", Message: "is not a known time zone"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckUserPatch(tt.patch)

			var se *shared.Error
			if !errors.As(err, &se) || se.Code != shared.CodeValidationFailed || !slices.Equal(se.Fields, tt.want) {
				t.Errorf("CheckUserPatch() = %+v, want validation_failed with %+v", err, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: 偏好的规则**

`server/internal/modules/identity/domain/profile.go`（新文件）：

```go
package domain

import (
	"encoding/json"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// themes are the themes a profile can hold (M1-P2 handoff): web's
// THEME_OPTIONS, without custom.
var themes = []string{"system", "light", "dark", "light-contrast", "dark-contrast"}

// languages are the languages of the web UI (M1 design 6).
var languages = []string{"en", "zh-CN"}

// OnboardingSteps are the four steps of onboarding (M2 design 4.3).
type OnboardingSteps struct {
	ProfileComplete bool `json:"profile_complete"`
	WorkspaceCreate bool `json:"workspace_create"`
	WorkspaceInvite bool `json:"workspace_invite"`
	WorkspaceJoin   bool `json:"workspace_join"`
}

// Profile is an account's preferences as the API shows them.
type Profile struct {
	Theme           string
	Language        string
	StartOfTheWeek  int // 0 Sunday … 6 Saturday
	OnboardingStep  OnboardingSteps
	IsOnboarded     bool
	IsTourCompleted bool
	LastWorkspaceID *uuid.UUID // a workspace the client names; no foreign key (M2 design 3.2)
	UpdatedAt       time.Time
}

// OnboardingStepsPatch sets the steps that are not nil and leaves the
// others: the database merges it into the stored object (M2 design 3.14).
type OnboardingStepsPatch struct {
	ProfileComplete *bool `json:"profile_complete,omitempty"`
	WorkspaceCreate *bool `json:"workspace_create,omitempty"`
	WorkspaceInvite *bool `json:"workspace_invite,omitempty"`
	WorkspaceJoin   *bool `json:"workspace_join,omitempty"`
}

// JSON is the patch as the object the database merges in: only the steps
// set, {} for none.
func (p OnboardingStepsPatch) JSON() []byte {
	out, _ := json.Marshal(p) // four *bool fields: cannot fail
	return out
}

// ProfilePatch is a partial update of a profile: a nil field stays as it
// is. LastWorkspaceSet tells whether LastWorkspaceID is written; nil then
// clears it.
type ProfilePatch struct {
	Theme            *string
	Language         *string
	StartOfTheWeek   *int
	OnboardingStep   OnboardingStepsPatch
	IsOnboarded      *bool
	IsTourCompleted  *bool
	LastWorkspaceSet bool
	LastWorkspaceID  *uuid.UUID
}

// CheckProfilePatch checks p (M2 design 4.3): a theme and a language of
// their lists, a first day of the week from 0 to 6. The structure, down to
// the steps' keys and types, was checked at the boundary (M2 design 3.11).
// Every problem is reported at once, as one 422 validation_failed.
func CheckProfilePatch(p ProfilePatch) error {
	var fields []shared.FieldError
	if p.Theme != nil && !slices.Contains(themes, *p.Theme) {
		fields = append(fields, shared.FieldError{Field: "theme", Code: shared.FieldInvalidFormat, Message: "is not a known theme"})
	}
	if p.Language != nil && !slices.Contains(languages, *p.Language) {
		fields = append(fields, shared.FieldError{Field: "language", Code: shared.FieldInvalidFormat, Message: "is not a known language"})
	}
	if p.StartOfTheWeek != nil && (*p.StartOfTheWeek < 0 || *p.StartOfTheWeek > 6) {
		fields = append(fields, shared.FieldError{Field: "start_of_the_week", Code: shared.FieldOutOfRange, Message: "must be between 0 and 6"})
	}
	if len(fields) > 0 {
		return shared.Invalid(fields...)
	}
	return nil
}
```

`server/internal/modules/identity/domain/profile_test.go`（新文件）：

```go
package domain

import (
	"errors"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestCheckProfilePatchAcceptsAValidPatch(t *testing.T) {
	// Web's THEME_OPTIONS (web/packages/constants/src/themes.ts), and the
	// CHECK of profiles.theme.
	for _, theme := range []string{"system", "light", "dark", "light-contrast", "dark-contrast"} {
		if err := CheckProfilePatch(ProfilePatch{Theme: &theme}); err != nil {
			t.Errorf("theme %q: %v", theme, err)
		}
	}
	for _, week := range []int{0, 6} {
		if err := CheckProfilePatch(ProfilePatch{Language: ptr("zh-CN"), StartOfTheWeek: &week}); err != nil {
			t.Errorf("week %d: %v", week, err)
		}
	}
}

func TestCheckProfilePatchReportsEveryField(t *testing.T) {
	err := CheckProfilePatch(ProfilePatch{Theme: ptr("custom"), Language: ptr("fr"), StartOfTheWeek: ptr(7)})

	want := []shared.FieldError{
		{Field: "theme", Code: "invalid_format", Message: "is not a known theme"},
		{Field: "language", Code: "invalid_format", Message: "is not a known language"},
		{Field: "start_of_the_week", Code: "out_of_range", Message: "must be between 0 and 6"},
	}
	var se *shared.Error
	if !errors.As(err, &se) || !slices.Equal(se.Fields, want) {
		t.Errorf("CheckProfilePatch() = %+v, want %+v", err, want)
	}
	if err := CheckProfilePatch(ProfilePatch{StartOfTheWeek: ptr(-1)}); err == nil {
		t.Error("CheckProfilePatch(week -1) = nil, want out_of_range")
	}
}

// The patch the database merges holds exactly the steps set.
func TestOnboardingStepsPatchJSON(t *testing.T) {
	tests := []struct {
		patch OnboardingStepsPatch
		want  string
	}{
		{OnboardingStepsPatch{}, `{}`},
		{OnboardingStepsPatch{ProfileComplete: ptr(true)}, `{"profile_complete":true}`},
		{OnboardingStepsPatch{WorkspaceCreate: ptr(false), WorkspaceJoin: ptr(true)}, `{"workspace_create":false,"workspace_join":true}`},
		{OnboardingStepsPatch{ProfileComplete: ptr(false), WorkspaceCreate: ptr(false), WorkspaceInvite: ptr(false), WorkspaceJoin: ptr(false)},
			`{"profile_complete":false,"workspace_create":false,"workspace_invite":false,"workspace_join":false}`},
	}
	for _, tt := range tests {
		if got := string(tt.patch.JSON()); got != tt.want {
			t.Errorf("JSON() = %s, want %s", got, tt.want)
		}
	}
}
```

- [ ] **Step 3: 查询**

`server/internal/modules/identity/adapter/postgres/queries/users.sql`（对 `605f367` 的差异）：

```diff
--- a/server/internal/modules/identity/adapter/postgres/queries/users.sql
+++ b/server/internal/modules/identity/adapter/postgres/queries/users.sql
@@ -24,6 +24,17 @@
 WHERE id = sqlc.arg(id)
 FOR NO KEY UPDATE;
 
+-- name: UpdateUser :one
+-- PATCH /me: only the fields that are set change (M2 design 3.14); the rest keep what a concurrent write left.
+UPDATE users
+SET updated_at    = sqlc.arg(now),
+    first_name    = CASE WHEN sqlc.arg(set_first_name)::boolean THEN sqlc.arg(first_name)::text ELSE first_name END,
+    last_name     = CASE WHEN sqlc.arg(set_last_name)::boolean THEN sqlc.arg(last_name)::text ELSE last_name END,
+    display_name  = CASE WHEN sqlc.arg(set_display_name)::boolean THEN sqlc.arg(display_name)::text ELSE display_name END,
+    user_timezone = CASE WHEN sqlc.arg(set_user_timezone)::boolean THEN sqlc.arg(user_timezone)::text ELSE user_timezone END
+WHERE id = sqlc.arg(id)
+RETURNING id, email, first_name, last_name, display_name, user_timezone, created_at;
+
 -- name: UpdatePasswordHash :exec
 UPDATE users
 SET password = sqlc.arg(password), updated_at = sqlc.arg(now)
```

`server/internal/modules/identity/adapter/postgres/queries/profiles.sql`（完整内容）：

```sql
-- name: CreateProfile :exec
-- Every preference takes its default (Plane's model defaults, M2 design 4.3).
INSERT INTO profiles (id, user_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(now), sqlc.arg(now));

-- name: GetProfile :one
SELECT theme, language, start_of_the_week, onboarding_step, is_onboarded, is_tour_completed, last_workspace_id, updated_at
FROM profiles
WHERE user_id = sqlc.arg(user_id);

-- name: UpdateProfile :one
-- PATCH /me/profile: only the fields that are set change, and the steps are merged into the stored object in the
-- statement (M2 design 3.14): a concurrent update of other steps waits for the row lock and is merged into the new
-- row, not lost. An empty patch, {}, merges nothing.
UPDATE profiles
SET updated_at        = sqlc.arg(now),
    theme             = CASE WHEN sqlc.arg(set_theme)::boolean THEN sqlc.arg(theme)::text ELSE theme END,
    language          = CASE WHEN sqlc.arg(set_language)::boolean THEN sqlc.arg(language)::text ELSE language END,
    start_of_the_week = CASE WHEN sqlc.arg(set_start_of_the_week)::boolean THEN sqlc.arg(start_of_the_week)::smallint ELSE start_of_the_week END,
    onboarding_step   = onboarding_step || sqlc.arg(onboarding_step_patch)::jsonb,
    is_onboarded      = CASE WHEN sqlc.arg(set_is_onboarded)::boolean THEN sqlc.arg(is_onboarded)::boolean ELSE is_onboarded END,
    is_tour_completed = CASE WHEN sqlc.arg(set_is_tour_completed)::boolean THEN sqlc.arg(is_tour_completed)::boolean ELSE is_tour_completed END,
    last_workspace_id = CASE WHEN sqlc.arg(set_last_workspace_id)::boolean THEN sqlc.narg(last_workspace_id)::uuid ELSE last_workspace_id END
WHERE user_id = sqlc.arg(user_id)
RETURNING theme, language, start_of_the_week, onboarding_step, is_onboarded, is_tour_completed, last_workspace_id, updated_at;
```

- [ ] **Step 4: 生成**

Run: `make gen-go`
Expected: 以下两个文件改变，其余生成物不变：

| 生成的文件 | SHA-256 | 行数 |
|---|---|---|
| `server/internal/modules/identity/adapter/postgres/gen/profiles.sql.go` | `c98fb7dd96fef4ff8065a42e5ef51fd5e5e314ececf008779af993ca9a57c862` | 141 |
| `server/internal/modules/identity/adapter/postgres/gen/users.sql.go` | `3b23734c0ba1dade31cc7c809e7ba074ea06e1b6ba5ae50b4864691ea1cf506b` | 189 |

- [ ] **Step 5: 端口和存储**

`server/internal/modules/identity/app/ports.go`（对 Task 5 版本的差异）：

```diff
--- a/server/internal/modules/identity/app/ports.go
+++ b/server/internal/modules/identity/app/ports.go
@@ -41,8 +41,28 @@
 type UserReader interface {
 	// GetUser returns ErrNotFound when there is no such account.
 	GetUser(ctx context.Context, id uuid.UUID) (domain.User, error)
+}
+
+// UserUpdater applies partial updates to accounts.
+type UserUpdater interface {
+	// UpdateUser applies p to account id at now and returns the account;
+	// ErrNotFound when there is none.
+	UpdateUser(ctx context.Context, id uuid.UUID, p domain.UserPatch, now time.Time) (domain.User, error)
+}
+
+// ProfileReader reads preferences.
+type ProfileReader interface {
+	// GetProfile returns ErrNotFound when userID has no profile.
+	GetProfile(ctx context.Context, userID uuid.UUID) (domain.Profile, error)
 }
 
+// ProfileUpdater applies partial updates to preferences.
+type ProfileUpdater interface {
+	// UpdateProfile applies p to userID's profile at now and returns it;
+	// ErrNotFound when there is none.
+	UpdateProfile(ctx context.Context, userID uuid.UUID, p domain.ProfilePatch, now time.Time) (domain.Profile, error)
+}
+
 // LoginAccount is what login reads of an account before its transaction:
 // the hash is the snapshot it verifies the password against (M2 design 3.5).
 type LoginAccount struct {
```

`server/internal/modules/identity/adapter/postgres/users.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/modules/identity/adapter/postgres/users.go
+++ b/server/internal/modules/identity/adapter/postgres/users.go
@@ -44,6 +44,38 @@
 	}, nil
 }
 
+// UpdateUser applies p to account id at now, in one statement, and returns
+// the account; app.ErrNotFound when there is none.
+func (s *Store) UpdateUser(ctx context.Context, id uuid.UUID, p domain.UserPatch, now time.Time) (domain.User, error) {
+	arg := gen.UpdateUserParams{Now: now, ID: id}
+	arg.SetFirstName, arg.FirstName = set(p.FirstName)
+	arg.SetLastName, arg.LastName = set(p.LastName)
+	arg.SetDisplayName, arg.DisplayName = set(p.DisplayName)
+	arg.SetUserTimezone, arg.UserTimezone = set(p.Timezone)
+	row, err := s.queries(ctx).UpdateUser(ctx, arg)
+	if err != nil {
+		return domain.User{}, notFound(err)
+	}
+	return domain.User{
+		ID:          row.ID,
+		Email:       row.Email,
+		FirstName:   row.FirstName,
+		LastName:    row.LastName,
+		DisplayName: row.DisplayName,
+		Timezone:    row.UserTimezone,
+		CreatedAt:   row.CreatedAt,
+	}, nil
+}
+
+// set is an optional value as a query's set flag and value.
+func set[T any](v *T) (bool, T) {
+	if v == nil {
+		var zero T
+		return false, zero
+	}
+	return true, *v
+}
+
 // FindLoginAccount reads the account of email, a normalized address;
 // app.ErrNotFound when there is none.
 func (s *Store) FindLoginAccount(ctx context.Context, email string) (app.LoginAccount, error) {
```

`server/internal/modules/identity/adapter/postgres/profiles.go`（新文件）：

```go
package postgresadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// GetProfile reads userID's preferences; app.ErrNotFound when there are
// none.
func (s *Store) GetProfile(ctx context.Context, userID uuid.UUID) (domain.Profile, error) {
	row, err := s.queries(ctx).GetProfile(ctx, userID)
	if err != nil {
		return domain.Profile{}, notFound(err)
	}
	return profileOf(row)
}

// UpdateProfile applies p to userID's preferences at now, in one statement
// that merges the steps (M2 design 3.14), and returns them;
// app.ErrNotFound when there are none.
func (s *Store) UpdateProfile(ctx context.Context, userID uuid.UUID, p domain.ProfilePatch, now time.Time) (domain.Profile, error) {
	arg := gen.UpdateProfileParams{Now: now, UserID: userID, OnboardingStepPatch: p.OnboardingStep.JSON()}
	arg.SetTheme, arg.Theme = set(p.Theme)
	arg.SetLanguage, arg.Language = set(p.Language)
	if p.StartOfTheWeek != nil {
		arg.SetStartOfTheWeek, arg.StartOfTheWeek = true, int16(*p.StartOfTheWeek) // 0–6, checked by the domain
	}
	arg.SetIsOnboarded, arg.IsOnboarded = set(p.IsOnboarded)
	arg.SetIsTourCompleted, arg.IsTourCompleted = set(p.IsTourCompleted)
	arg.SetLastWorkspaceID, arg.LastWorkspaceID = p.LastWorkspaceSet, p.LastWorkspaceID
	row, err := s.queries(ctx).UpdateProfile(ctx, arg)
	if err != nil {
		return domain.Profile{}, notFound(err)
	}
	return profileOf(gen.GetProfileRow(row))
}

// profileOf reads a profile row. The steps are the object that the
// column's CHECK guarantees: four booleans.
func profileOf(row gen.GetProfileRow) (domain.Profile, error) {
	var steps domain.OnboardingSteps
	if err := json.Unmarshal(row.OnboardingStep, &steps); err != nil {
		return domain.Profile{}, fmt.Errorf("read onboarding_step: %w", err)
	}
	return domain.Profile{
		Theme:           row.Theme,
		Language:        row.Language,
		StartOfTheWeek:  int(row.StartOfTheWeek),
		OnboardingStep:  steps,
		IsOnboarded:     row.IsOnboarded,
		IsTourCompleted: row.IsTourCompleted,
		LastWorkspaceID: row.LastWorkspaceID,
		UpdatedAt:       row.UpdatedAt,
	}, nil
}
```

`server/internal/modules/identity/adapter/postgres/account_test.go`（新文件）：

```go
package postgresadapter_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func ptr[T any](v T) *T { return &v }

// accountWithProfile is a new account and its default profile.
func accountWithProfile(t *testing.T, s *postgresadapter.Store) app.NewUser {
	t.Helper()
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)
	if err := s.CreateDefaultProfile(context.Background(), uuid.NewV7(), u.ID, now); err != nil {
		t.Fatal(err)
	}
	return u
}

// Only the fields set change; updated_at is the use case's time.
func TestUpdateUser(t *testing.T) {
	s, pool := newStore(t)
	u := accountWithProfile(t, s)
	later := now.Add(time.Hour)

	got, err := s.UpdateUser(context.Background(), u.ID, domain.UserPatch{FirstName: ptr("Ada"), Timezone: ptr("Asia/Shanghai")}, later)

	want := domain.User{ID: u.ID, Email: u.Email, FirstName: "Ada", DisplayName: "alice", Timezone: "Asia/Shanghai", CreatedAt: now}
	if err != nil || got != want {
		t.Errorf("UpdateUser() = %+v, %v; want %+v", got, err, want)
	}
	got, err = s.UpdateUser(context.Background(), u.ID, domain.UserPatch{LastName: ptr("Lovelace"), DisplayName: ptr("ada")}, later)
	want.LastName, want.DisplayName = "Lovelace", "ada"
	if err != nil || got != want {
		t.Errorf("second UpdateUser() = %+v, %v; want %+v", got, err, want)
	}
	if read, err := s.GetUser(context.Background(), u.ID); err != nil || read != want {
		t.Errorf("GetUser() = %+v, %v; want %+v", read, err, want)
	}
	assertUpdatedAt(t, pool, "users", "id", u.ID, later)
}

func TestUpdateUnknownUser(t *testing.T) {
	s, _ := newStore(t)
	if _, err := s.UpdateUser(context.Background(), uuid.NewV7(), domain.UserPatch{FirstName: ptr("x")}, now); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("UpdateUser() = %v, want app.ErrNotFound", err)
	}
}

// assertUpdatedAt checks the updated_at of the row of table whose column
// is id.
func assertUpdatedAt(t *testing.T, pool *pgxpool.Pool, table, column string, id uuid.UUID, want time.Time) {
	t.Helper()
	var got time.Time
	if err := pool.QueryRow(context.Background(), "SELECT updated_at FROM "+table+" WHERE "+column+" = $1", id).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if !got.Equal(want) {
		t.Errorf("%s.updated_at = %v, want %v", table, got, want)
	}
}

func TestGetProfileReadsTheDefaults(t *testing.T) {
	s, _ := newStore(t)
	u := accountWithProfile(t, s)

	got, err := s.GetProfile(context.Background(), u.ID)

	want := domain.Profile{Theme: "system", Language: "en", UpdatedAt: now}
	if err != nil || got != want {
		t.Errorf("GetProfile() = %+v, %v; want %+v", got, err, want)
	}
	if _, err := s.GetProfile(context.Background(), uuid.NewV7()); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("GetProfile(unknown) = %v, want app.ErrNotFound", err)
	}
}

func TestUpdateProfile(t *testing.T) {
	s, _ := newStore(t)
	u := accountWithProfile(t, s)
	workspace := uuid.NewV7()
	later := now.Add(time.Hour)

	got, err := s.UpdateProfile(context.Background(), u.ID, domain.ProfilePatch{
		Theme: ptr("dark-contrast"), Language: ptr("zh-CN"), StartOfTheWeek: ptr(6),
		OnboardingStep: domain.OnboardingStepsPatch{ProfileComplete: ptr(true), WorkspaceJoin: ptr(true)},
		IsOnboarded:    ptr(true), IsTourCompleted: ptr(true), LastWorkspaceSet: true, LastWorkspaceID: &workspace,
	}, later)

	want := domain.Profile{
		Theme: "dark-contrast", Language: "zh-CN", StartOfTheWeek: 6,
		OnboardingStep: domain.OnboardingSteps{ProfileComplete: true, WorkspaceJoin: true},
		IsOnboarded:    true, IsTourCompleted: true, LastWorkspaceID: &workspace, UpdatedAt: later,
	}
	if err != nil || !sameProfile(got, want) {
		t.Errorf("UpdateProfile() = %+v, %v; want %+v", got, err, want)
	}

	// An empty patch changes nothing; clearing the workspace leaves the rest.
	got, err = s.UpdateProfile(context.Background(), u.ID, domain.ProfilePatch{}, later)
	if err != nil || !sameProfile(got, want) {
		t.Errorf("UpdateProfile(empty) = %+v, %v; want %+v", got, err, want)
	}
	got, err = s.UpdateProfile(context.Background(), u.ID, domain.ProfilePatch{LastWorkspaceSet: true}, later)
	want.LastWorkspaceID = nil
	if err != nil || !sameProfile(got, want) {
		t.Errorf("UpdateProfile(clear the workspace) = %+v, %v; want %+v", got, err, want)
	}
	if read, err := s.GetProfile(context.Background(), u.ID); err != nil || !sameProfile(read, want) {
		t.Errorf("GetProfile() = %+v, %v; want %+v", read, err, want)
	}
}

func TestUpdateUnknownProfile(t *testing.T) {
	s, _ := newStore(t)
	if _, err := s.UpdateProfile(context.Background(), uuid.NewV7(), domain.ProfilePatch{Theme: ptr("dark")}, now); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("UpdateProfile() = %v, want app.ErrNotFound", err)
	}
}

func sameProfile(a, b domain.Profile) bool {
	sameWorkspace := (a.LastWorkspaceID == nil) == (b.LastWorkspaceID == nil) &&
		(a.LastWorkspaceID == nil || *a.LastWorkspaceID == *b.LastWorkspaceID)
	a.LastWorkspaceID, b.LastWorkspaceID = nil, nil
	return sameWorkspace && a.UpdatedAt.Equal(b.UpdatedAt) && a == b
}

// Two updates of different steps at once both stay (M2 design 3.14): the
// second waits for the first's row lock, then merges into the row the first
// committed. The test holds the first transaction open until the second
// statement is seen waiting for the lock, so the order is not left to
// chance; every wait has a deadline and fails rather than hangs.
func TestUpdateProfileMergesConcurrentSteps(t *testing.T) {
	s, pool := newStore(t)
	u := accountWithProfile(t, s)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	ctx := context.Background()
	updated, release := make(chan struct{}), make(chan struct{})
	first, second := make(chan error, 1), make(chan error, 1)
	go func() {
		first <- tx.WithinTx(ctx, func(ctx context.Context) error {
			if _, err := s.UpdateProfile(ctx, u.ID, domain.ProfilePatch{OnboardingStep: domain.OnboardingStepsPatch{ProfileComplete: ptr(true)}}, now); err != nil {
				return err
			}
			close(updated)
			<-release
			return nil
		})
	}()
	select {
	case <-updated:
	case err := <-first:
		t.Fatalf("first update: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("the first update did not happen within 10s")
	}
	go func() {
		_, err := s.UpdateProfile(ctx, u.ID, domain.ProfilePatch{OnboardingStep: domain.OnboardingStepsPatch{WorkspaceCreate: ptr(true)}}, now)
		second <- err
	}()
	waitForLockWait(t, pool)
	close(release)
	for name, done := range map[string]chan error{"first": first, "second": second} {
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("%s update: %v", name, err)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("the %s update did not finish within 10s", name)
		}
	}

	got, err := s.GetProfile(ctx, u.ID)
	want := domain.OnboardingSteps{ProfileComplete: true, WorkspaceCreate: true}
	if err != nil || got.OnboardingStep != want {
		t.Errorf("steps = %+v, %v; want both updates kept: %+v", got.OnboardingStep, err, want)
	}
}

// waitForLockWait returns once a backend of this test's database waits for
// a lock, and fails the test after 10s.
func waitForLockWait(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		var waiting int
		if err := pool.QueryRow(context.Background(),
			"SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock'").Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting > 0 {
			return
		}
	}
	t.Fatal("no statement waited for a lock within 10s")
}
```

- [ ] **Step 6: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。

- [ ] **Step 7: 提交**

```bash
git add server/internal/modules/identity/domain server/internal/modules/identity/adapter/postgres server/internal/modules/identity/app/ports.go
```
```bash
git commit -m "feat(M2/P3a): the rules and the store of the account and its preferences

Names are checked as Plane's contains_url checks them; a partial update
writes only the fields set, and the onboarding steps merge into the
stored object in the statement, so concurrent updates of different
steps both stay.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run: `make gen-check`
Expected: 没有差异。

**Done when:** `TestContainsURLAsPlane`、`TestCheckUserPatchReportsEveryField`、`TestUpdateProfileMergesConcurrentSteps` 通过；两个生成物与上表相同；`make gen-check` 干净。

---

### Task 9: `PATCH /me`、`GET`/`PATCH /me/profile`

**Files:**
- Create: `server/internal/modules/identity/app/update_me.go`、`get_profile.go`、`update_profile.go`、`account_test.go`
- Modify: `server/internal/modules/identity/app/fakes_test.go`（过渡）
- Modify: `api/openapi.yaml`、`api/modules/identity.yaml`（过渡）
- Create: `server/internal/modules/identity/adapter/http/profile.go`、`profile_test.go`
- Modify（过渡）: `server/internal/modules/identity/adapter/http/me.go`、`me_test.go`、`handler.go`、`handler_test.go`；`server/internal/modules/identity/module.go`
- Create（过渡：Task 10、11 加上修改密码和停用）: `server/internal/bootstrap/account_test.go`
- Generate: `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`、`server/internal/modules/identity/adapter/http/gen/server.gen.go`、`bodyshape.gen.go`

**Interfaces:**
- Consumes: `CheckUserPatch`、`CheckProfilePatch` 和三个端口（Task 8）。
- Produces（spec 2.13，M2 设计 4.2、4.3、5.1、5.2）：
  - `app.NewUpdateMe(users UserUpdater, clock Clock)`、`app.NewGetProfile(profiles ProfileReader)`、`app.NewUpdateProfile(profiles ProfileUpdater, clock Clock)`：先检查补丁（422），再一条语句（不开事务，M2 设计 6.4"单条语句的写入"）；账户或资料已不存在（认证之后被删）是 401；
  - 操作：`updateMe`（`PATCH /api/v0/me`，200 `User`，`[validation_failed]`）、`getProfile`（`GET /api/v0/me/profile`，200 `Profile`，`[]`）、`updateProfile`（`PATCH /api/v0/me/profile`，200 `Profile`，`[validation_failed]`）；
  - `UserUpdate`：`first_name`、`last_name`、`display_name`、`user_timezone` 都可选、不可为 `null`，没有 `email`（只有管理员能改，决策点 1）；
  - `Theme`（五个值）、`Language`（`en`、`zh-CN`）、`StartOfTheWeek`（0–6 的整数枚举）；`OnboardingSteps`（四个必有的布尔值）、`OnboardingStepsUpdate`（四个可选的布尔值，不认识的键是 400）；`Profile`；`ProfileUpdate`（`last_workspace_id` 可为 `null`，`null` 清空）；
  - 适配器：`me.go` 的 `GetMe`、`UpdateMe` 共用 `user(domain.User) gen.User`；`profile.go` 的 `GetProfile`、`UpdateProfile`。

**Tests:**
- `app/account_test.go`：`TestUpdateMe`、`TestGetProfile`、`TestUpdateProfile`（假仓储按账户 id 找行，记下补丁和时刻）；`TestAccountPatchesAreChecked`（不合规的补丁是一个 422，什么都不写）；`TestAccountUseCasesErrors`（没有 actor、账户已不存在是 401；数据库错误原样返回）。
- `me_test.go`：`TestUpdateMe`（传入的字段逐个到达用例，没传的是 `nil`）；`TestUpdateMeProblems`（`{"first_name":null}` 400 `invalid_format`、`email` 400 `not_allowed`，用例都没有运行；用例的 422 原样答出）。
- `profile_test.go`：`TestGetProfile`；`TestUpdateProfile`（`last_workspace_id` 只在传入时写，`null` 清空）；`TestUpdateProfileProblems`（不认识的步骤 `onboarding_step.profile_completed` 400 `not_allowed`、`{"theme":null}` 400、`last_workspace_id` 不是 uuid 400，用例都没有运行；未知主题的 422 原样答出）。
- `bootstrap/account_test.go`：`TestTheAccountAndItsPreferencesWithAPersonalAccessToken`（真实数据库，只用 PAT：改名和时区；改主题和一个步骤；读回的步骤已合并）。
- 整程序测试 1–4 自动覆盖三个新操作；第 4 个对 `updateProfile` 跑"每一种问题一起"（嵌套对象里未声明的属性、非空可选属性为 `null`、`last_workspace_id` 的格式错误）。

- [ ] **Step 1: 用例**

`server/internal/modules/identity/app/update_me.go`（新文件）：

```go
package app

import (
	"context"
	"errors"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateMe changes the caller's names and time zone: PATCH /api/v0/me. It is
// one statement, no transaction (M2 design 6.4).
type UpdateMe struct {
	users UserUpdater
	clock Clock
}

// NewUpdateMe returns the use case.
func NewUpdateMe(users UserUpdater, clock Clock) *UpdateMe {
	return &UpdateMe{users: users, clock: clock}
}

// Execute checks p and applies it to the caller's account at the clock's
// now; the fields p leaves nil stay as they are.
func (u *UpdateMe) Execute(ctx context.Context, p domain.UserPatch) (domain.User, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.User{}, err
	}
	if err := domain.CheckUserPatch(p); err != nil {
		return domain.User{}, err
	}
	user, err := u.users.UpdateUser(ctx, actor.UserID, p, u.clock.Now())
	if errors.Is(err, ErrNotFound) {
		// Authentication found the account a moment ago; it is gone now.
		return domain.User{}, shared.Unauthenticated()
	}
	return user, err
}
```

`server/internal/modules/identity/app/get_profile.go`（新文件）：

```go
package app

import (
	"context"
	"errors"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// GetProfile reads the caller's preferences: GET /api/v0/me/profile.
type GetProfile struct {
	profiles ProfileReader
}

// NewGetProfile returns the use case.
func NewGetProfile(profiles ProfileReader) *GetProfile {
	return &GetProfile{profiles: profiles}
}

// Execute returns the preferences of the request's actor.
func (g *GetProfile) Execute(ctx context.Context) (domain.Profile, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Profile{}, err
	}
	p, err := g.profiles.GetProfile(ctx, actor.UserID)
	if errors.Is(err, ErrNotFound) {
		// Every account has its profile from registration on (M2 design
		// 4.3), and it goes only with the account: the account is gone.
		return domain.Profile{}, shared.Unauthenticated()
	}
	return p, err
}
```

`server/internal/modules/identity/app/update_profile.go`（新文件）：

```go
package app

import (
	"context"
	"errors"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateProfile changes the caller's preferences: PATCH /api/v0/me/profile.
// It is one statement, which merges the onboarding steps (M2 design 3.14,
// 6.4).
type UpdateProfile struct {
	profiles ProfileUpdater
	clock    Clock
}

// NewUpdateProfile returns the use case.
func NewUpdateProfile(profiles ProfileUpdater, clock Clock) *UpdateProfile {
	return &UpdateProfile{profiles: profiles, clock: clock}
}

// Execute checks p and applies it to the caller's preferences at the
// clock's now; the fields and steps p leaves nil stay as they are.
func (u *UpdateProfile) Execute(ctx context.Context, p domain.ProfilePatch) (domain.Profile, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Profile{}, err
	}
	if err := domain.CheckProfilePatch(p); err != nil {
		return domain.Profile{}, err
	}
	profile, err := u.profiles.UpdateProfile(ctx, actor.UserID, p, u.clock.Now())
	if errors.Is(err, ErrNotFound) {
		// The profile goes only with the account (GetProfile).
		return domain.Profile{}, shared.Unauthenticated()
	}
	return profile, err
}
```

`server/internal/modules/identity/app/fakes_test.go`（对 Task 6 版本的差异）：

```diff
--- a/server/internal/modules/identity/app/fakes_test.go
+++ b/server/internal/modules/identity/app/fakes_test.go
@@ -390,8 +390,67 @@
 func (f *fakeAPITokens) RevokeAPIToken(_ context.Context, id, userID uuid.UUID, now time.Time) (bool, error) {
 	f.revoked = append(f.revoked, revocation{id, userID, now})
 	return f.revokeOK, f.revokeErr
+}
+
+// userUpdate is one UpdateUser call.
+type userUpdate struct {
+	id    uuid.UUID
+	patch domain.UserPatch
+	now   time.Time
+}
+
+// profileUpdate is one UpdateProfile call.
+type profileUpdate struct {
+	userID uuid.UUID
+	patch  domain.ProfilePatch
+	now    time.Time
 }
 
+// fakeAccount is one account and its profile, in memory: it finds them by
+// the account's id only. err, when set, is every call's error.
+type fakeAccount struct {
+	user           domain.User
+	profile        domain.Profile
+	err            error
+	userUpdates    []userUpdate
+	profileReads   []uuid.UUID
+	profileUpdates []profileUpdate
+}
+
+func (f *fakeAccount) find(id uuid.UUID) error {
+	switch {
+	case f.err != nil:
+		return f.err
+	case id != f.user.ID:
+		return app.ErrNotFound
+	}
+	return nil
+}
+
+func (f *fakeAccount) UpdateUser(_ context.Context, id uuid.UUID, p domain.UserPatch, now time.Time) (domain.User, error) {
+	f.userUpdates = append(f.userUpdates, userUpdate{id, p, now})
+	if err := f.find(id); err != nil {
+		return domain.User{}, err
+	}
+	return f.user, nil
+}
+
+func (f *fakeAccount) GetProfile(_ context.Context, userID uuid.UUID) (domain.Profile, error) {
+	f.profileReads = append(f.profileReads, userID)
+	if err := f.find(userID); err != nil {
+		return domain.Profile{}, err
+	}
+	return f.profile, nil
+}
+
+func (f *fakeAccount) UpdateProfile(_ context.Context, userID uuid.UUID, p domain.ProfilePatch, now time.Time) (domain.Profile, error) {
+	f.profileUpdates = append(f.profileUpdates, profileUpdate{userID, p, now})
+	if err := f.find(userID); err != nil {
+		return domain.Profile{}, err
+	}
+	return f.profile, nil
+}
+
 type fixedPolicy struct {
 	allow bool
 	err   error
```

`server/internal/modules/identity/app/account_test.go`（新文件）：

```go
package app_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The use cases of the caller's account and preferences: UpdateMe,
// GetProfile and UpdateProfile.

func aliceAccount() *fakeAccount {
	workspace := uuid.MustParse("0199a2b4-0000-7000-8000-000000000007")
	return &fakeAccount{
		user: domain.User{ID: userID, Email: "alice@corp.com", FirstName: "Ann", DisplayName: "alice", Timezone: "Asia/Shanghai", CreatedAt: now},
		profile: domain.Profile{
			Theme: "dark", Language: "zh-CN", StartOfTheWeek: 1, OnboardingStep: domain.OnboardingSteps{ProfileComplete: true},
			LastWorkspaceID: &workspace, UpdatedAt: now,
		},
	}
}

func TestUpdateMe(t *testing.T) {
	account := aliceAccount()
	patch := domain.UserPatch{FirstName: ptr("Ann"), Timezone: ptr("Asia/Shanghai")}

	got, err := app.NewUpdateMe(account, clocktest.At(now)).Execute(shared.WithActor(context.Background(), tokenActor), patch)

	if err != nil || got != account.user {
		t.Errorf("Execute() = %+v, %v; want %+v", got, err, account.user)
	}
	if want := []userUpdate{{userID, patch, now}}; !slices.Equal(account.userUpdates, want) {
		t.Errorf("updates = %+v, want %+v", account.userUpdates, want)
	}
}

func TestGetProfile(t *testing.T) {
	account := aliceAccount()

	got, err := app.NewGetProfile(account).Execute(shared.WithActor(context.Background(), sessionActor))

	if err != nil || got != account.profile {
		t.Errorf("Execute() = %+v, %v; want %+v", got, err, account.profile)
	}
	if want := []uuid.UUID{userID}; !slices.Equal(account.profileReads, want) {
		t.Errorf("profiles read = %v, want %v", account.profileReads, want)
	}
}

func TestUpdateProfile(t *testing.T) {
	account := aliceAccount()
	patch := domain.ProfilePatch{
		Theme: ptr("dark"), OnboardingStep: domain.OnboardingStepsPatch{ProfileComplete: ptr(true)}, LastWorkspaceSet: true,
	}

	got, err := app.NewUpdateProfile(account, clocktest.At(now)).Execute(shared.WithActor(context.Background(), tokenActor), patch)

	if err != nil || got != account.profile {
		t.Errorf("Execute() = %+v, %v; want %+v", got, err, account.profile)
	}
	if want := []profileUpdate{{userID, patch, now}}; !slices.Equal(account.profileUpdates, want) {
		t.Errorf("updates = %+v, want %+v", account.profileUpdates, want)
	}
}

// An invalid patch is one 422 with every field, and writes nothing (M2
// design 4.2, 4.3).
func TestAccountPatchesAreChecked(t *testing.T) {
	ctx := shared.WithActor(context.Background(), sessionActor)
	tests := []struct {
		name   string
		run    func(*fakeAccount) error
		fields []string
	}{
		{"UpdateMe", func(f *fakeAccount) error {
			_, err := app.NewUpdateMe(f, clocktest.At(now)).Execute(ctx, domain.UserPatch{DisplayName: ptr(""), Timezone: ptr("Mars/Olympus")})
			return err
		}, []string{"display_name", "user_timezone"}},
		{"UpdateProfile", func(f *fakeAccount) error {
			_, err := app.NewUpdateProfile(f, clocktest.At(now)).Execute(ctx, domain.ProfilePatch{Theme: ptr("neon"), StartOfTheWeek: ptr(7)})
			return err
		}, []string{"theme", "start_of_the_week"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := aliceAccount()

			err := tt.run(account)

			var se *shared.Error
			var fields []string
			if errors.As(err, &se) {
				for _, f := range se.Fields {
					fields = append(fields, f.Field)
				}
			}
			if se == nil || se.Code != shared.CodeValidationFailed || !slices.Equal(fields, tt.fields) {
				t.Errorf("Execute() = %v, want validation_failed on %v", err, tt.fields)
			}
			if len(account.userUpdates)+len(account.profileUpdates) != 0 {
				t.Errorf("wrote %+v %+v, want nothing", account.userUpdates, account.profileUpdates)
			}
		})
	}
}

// Without an actor, and when the account is gone since authentication, the
// caller is unauthenticated; a database error stays one (500).
func TestAccountUseCasesErrors(t *testing.T) {
	boom := errors.New("connection refused")
	gone := uuid.MustParse("0199a2b4-0000-7000-8000-000000000009")
	authed := shared.WithActor(context.Background(), sessionActor)
	useCases := []struct {
		name string
		run  func(context.Context, *fakeAccount) error
	}{
		{"UpdateMe", func(ctx context.Context, f *fakeAccount) error {
			_, err := app.NewUpdateMe(f, clocktest.At(now)).Execute(ctx, domain.UserPatch{})
			return err
		}},
		{"GetProfile", func(ctx context.Context, f *fakeAccount) error {
			_, err := app.NewGetProfile(f).Execute(ctx)
			return err
		}},
		{"UpdateProfile", func(ctx context.Context, f *fakeAccount) error {
			_, err := app.NewUpdateProfile(f, clocktest.At(now)).Execute(ctx, domain.ProfilePatch{})
			return err
		}},
	}
	tests := []struct {
		name    string
		ctx     context.Context
		account uuid.UUID // the id of the fake's account
		err     error     // the fake's error
		want    error
	}{
		{"without an actor", context.Background(), userID, nil, shared.Unauthenticated()},
		{"when the account is gone", authed, gone, nil, shared.Unauthenticated()},
		{"when the database fails", authed, userID, boom, boom},
	}
	for _, uc := range useCases {
		for _, tt := range tests {
			t.Run(uc.name+" "+tt.name, func(t *testing.T) {
				account := &fakeAccount{user: domain.User{ID: tt.account}, err: tt.err}

				if err := uc.run(tt.ctx, account); !errors.Is(err, tt.want) {
					t.Errorf("Execute() = %v, want %v", err, tt.want)
				}
			})
		}
	}
}
```

- [ ] **Step 2: 契约**

`api/openapi.yaml`（对 Task 7 版本的差异）：

```diff
--- a/api/openapi.yaml
+++ b/api/openapi.yaml
@@ -31,6 +31,8 @@
     $ref: 'modules/identity.yaml#/paths/~1api~1v0~1auth~1logout'
   /api/v0/me:
     $ref: 'modules/identity.yaml#/paths/~1api~1v0~1me'
+  /api/v0/me/profile:
+    $ref: 'modules/identity.yaml#/paths/~1api~1v0~1me~1profile'
   /api/v0/me/api-tokens:
     $ref: 'modules/identity.yaml#/paths/~1api~1v0~1me~1api-tokens'
   /api/v0/api-tokens/{token_id}:
```

`api/modules/identity.yaml`（对 Task 7 版本的差异）：

```diff
--- a/api/modules/identity.yaml
+++ b/api/modules/identity.yaml
@@ -132,6 +132,71 @@
                 $ref: '#/components/schemas/User'
         default:
           $ref: '#/components/responses/Problem'
+    patch:
+      operationId: updateMe
+      tags: [identity]
+      summary: Change the caller's names and time zone
+      description: >-
+        Changes the fields sent and leaves the others. The e-mail address
+        cannot change here: the server's administrator changes it.
+      security: [{bearer: []}]
+      x-problem-codes: [validation_failed]
+      requestBody:
+        required: true
+        content:
+          application/json:
+            schema:
+              $ref: '#/components/schemas/UserUpdate'
+      responses:
+        '200':
+          description: The caller's account, changed.
+          content:
+            application/json:
+              schema:
+                $ref: '#/components/schemas/User'
+        default:
+          $ref: '#/components/responses/Problem'
+  /api/v0/me/profile:
+    get:
+      operationId: getProfile
+      tags: [identity]
+      summary: Read the caller's preferences
+      security: [{bearer: []}]
+      x-problem-codes: []
+      responses:
+        '200':
+          description: The caller's preferences.
+          content:
+            application/json:
+              schema:
+                $ref: '#/components/schemas/Profile'
+        default:
+          $ref: '#/components/responses/Problem'
+    patch:
+      operationId: updateProfile
+      tags: [identity]
+      summary: Change the caller's preferences
+      description: >-
+        Changes the fields sent and leaves the others. The onboarding steps
+        sent are merged into the stored ones: a step not sent keeps its value,
+        also when another request changes it at the same time.
+      security: [{bearer: []}]
+      x-problem-codes: [validation_failed]
+      requestBody:
+        required: true
+        content:
+          application/json:
+            schema:
+              $ref: '#/components/schemas/ProfileUpdate'
+      responses:
+        '200':
+          description: The caller's preferences, changed.
+          content:
+            application/json:
+              schema:
+                $ref: '#/components/schemas/Profile'
+        default:
+          $ref: '#/components/responses/Problem'
   /api/v0/me/api-tokens:
     get:
       operationId: listApiTokens
@@ -323,8 +388,105 @@
           description: Null until uploads arrive (M5).
           type: [string, 'null']
         created_at:
+          type: string
+          format: date-time
+    UserUpdate:
+      type: object
+      additionalProperties: false
+      properties:
+        first_name:
+          description: At most 255 characters, without a web address.
+          type: string
+        last_name:
+          description: At most 255 characters, without a web address.
+          type: string
+        display_name:
+          description: 1–255 characters.
+          type: string
+        user_timezone:
+          description: An IANA time zone name, e.g. from GET /api/v0/timezones.
           type: string
+    Theme:
+      type: string
+      enum: [system, light, dark, light-contrast, dark-contrast]
+    Language:
+      description: The language of the web UI.
+      type: string
+      enum: [en, zh-CN]
+    StartOfTheWeek:
+      description: The first day of the week, 0 Sunday to 6 Saturday.
+      type: integer
+      enum: [0, 1, 2, 3, 4, 5, 6]
+    OnboardingSteps:
+      type: object
+      additionalProperties: false
+      required: [profile_complete, workspace_create, workspace_invite, workspace_join]
+      properties:
+        profile_complete:
+          type: boolean
+        workspace_create:
+          type: boolean
+        workspace_invite:
+          type: boolean
+        workspace_join:
+          type: boolean
+    OnboardingStepsUpdate:
+      description: The steps to set; the others keep their values.
+      type: object
+      additionalProperties: false
+      properties:
+        profile_complete:
+          type: boolean
+        workspace_create:
+          type: boolean
+        workspace_invite:
+          type: boolean
+        workspace_join:
+          type: boolean
+    Profile:
+      type: object
+      additionalProperties: false
+      required: [theme, language, start_of_the_week, onboarding_step, is_onboarded, is_tour_completed, last_workspace_id, updated_at]
+      properties:
+        theme:
+          $ref: '#/components/schemas/Theme'
+        language:
+          $ref: '#/components/schemas/Language'
+        start_of_the_week:
+          $ref: '#/components/schemas/StartOfTheWeek'
+        onboarding_step:
+          $ref: '#/components/schemas/OnboardingSteps'
+        is_onboarded:
+          type: boolean
+        is_tour_completed:
+          type: boolean
+        last_workspace_id:
+          description: The workspace the web app opens last; null for none. It is not checked to exist.
+          type: [string, 'null']
+          format: uuid
+        updated_at:
+          type: string
           format: date-time
+    ProfileUpdate:
+      type: object
+      additionalProperties: false
+      properties:
+        theme:
+          $ref: '#/components/schemas/Theme'
+        language:
+          $ref: '#/components/schemas/Language'
+        start_of_the_week:
+          $ref: '#/components/schemas/StartOfTheWeek'
+        onboarding_step:
+          $ref: '#/components/schemas/OnboardingStepsUpdate'
+        is_onboarded:
+          type: boolean
+        is_tour_completed:
+          type: boolean
+        last_workspace_id:
+          description: Null clears it.
+          type: [string, 'null']
+          format: uuid
     ApiToken:
       description: >-
         A personal access token as lists show it. The token itself appears
```

- [ ] **Step 3: 生成**

Run: `make gen`
Expected: 以下四个文件改变，其余生成物不变：

| 生成的文件 | SHA-256 | 行数 |
|---|---|---|
| `api/dist/openapi.yaml` | `889d61a7186232209ebac251cec4ea3eb439c19068a3b1b0f95b01194aab1ffd` | 760 |
| `server/internal/modules/identity/adapter/http/gen/bodyshape.gen.go` | `faaf53d21c1b85069c20698164dd068ac3ff7ab03abcdb0f7608d72f3dde1a46` | 55 |
| `server/internal/modules/identity/adapter/http/gen/server.gen.go` | `0d36fbe6d094b2ae1c0502fbe1be09c3bbdb432da300a2178a8def09cca1b879` | 1610 |
| `web/packages/api-client/src/schema.gen.ts` | `f4448e258dd6718800003a0b83cfdeaac30da5bd6c3cb1283544a3640732bc6e` | 728 |

- [ ] **Step 4: HTTP 适配器和接线**

`server/internal/modules/identity/adapter/http/handler.go`（对 Task 7 版本的差异）：

```diff
--- a/server/internal/modules/identity/adapter/http/handler.go
+++ b/server/internal/modules/identity/adapter/http/handler.go
@@ -42,6 +42,21 @@
 	Execute(ctx context.Context) (domain.User, error)
 }
 
+// UpdateMeUseCase is app.UpdateMe.
+type UpdateMeUseCase interface {
+	Execute(ctx context.Context, p domain.UserPatch) (domain.User, error)
+}
+
+// GetProfileUseCase is app.GetProfile.
+type GetProfileUseCase interface {
+	Execute(ctx context.Context) (domain.Profile, error)
+}
+
+// UpdateProfileUseCase is app.UpdateProfile.
+type UpdateProfileUseCase interface {
+	Execute(ctx context.Context, p domain.ProfilePatch) (domain.Profile, error)
+}
+
 // ListAPITokensUseCase is app.ListAPITokens.
 type ListAPITokensUseCase interface {
 	Execute(ctx context.Context, limit *int, cursor *string) (app.APITokenPage, error)
@@ -64,6 +79,9 @@
 	Refresh        RefreshUseCase
 	Logout         LogoutUseCase
 	GetMe          GetMeUseCase
+	UpdateMe       UpdateMeUseCase
+	GetProfile     GetProfileUseCase
+	UpdateProfile  UpdateProfileUseCase
 	ListAPITokens  ListAPITokensUseCase
 	CreateAPIToken CreateAPITokenUseCase
 	RevokeAPIToken RevokeAPITokenUseCase
```

`server/internal/modules/identity/adapter/http/me.go`（完整内容）：

```go
package httpadapter

import (
	"context"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// GetMe serves GET /api/v0/me.
func (h handler) GetMe(ctx context.Context, _ gen.GetMeRequestObject) (gen.GetMeResponseObject, error) {
	u, err := h.uc.GetMe.Execute(ctx)
	if err != nil {
		return nil, err
	}
	return gen.GetMe200JSONResponse(user(u)), nil
}

// UpdateMe serves PATCH /api/v0/me.
func (h handler) UpdateMe(ctx context.Context, req gen.UpdateMeRequestObject) (gen.UpdateMeResponseObject, error) {
	u, err := h.uc.UpdateMe.Execute(ctx, domain.UserPatch{
		FirstName:   req.Body.FirstName,
		LastName:    req.Body.LastName,
		DisplayName: req.Body.DisplayName,
		Timezone:    req.Body.UserTimezone,
	})
	if err != nil {
		return nil, err
	}
	return gen.UpdateMe200JSONResponse(user(u)), nil
}

// user is u as the API shows it.
func user(u domain.User) gen.User {
	return gen.User{
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
	}
}
```

`server/internal/modules/identity/adapter/http/profile.go`（新文件）：

```go
package httpadapter

import (
	"context"
	"uuid"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// GetProfile serves GET /api/v0/me/profile.
func (h handler) GetProfile(ctx context.Context, _ gen.GetProfileRequestObject) (gen.GetProfileResponseObject, error) {
	p, err := h.uc.GetProfile.Execute(ctx)
	if err != nil {
		return nil, err
	}
	return gen.GetProfile200JSONResponse(profile(p)), nil
}

// UpdateProfile serves PATCH /api/v0/me/profile.
func (h handler) UpdateProfile(ctx context.Context, req gen.UpdateProfileRequestObject) (gen.UpdateProfileResponseObject, error) {
	b := req.Body
	patch := domain.ProfilePatch{
		Theme:           (*string)(b.Theme),
		Language:        (*string)(b.Language),
		StartOfTheWeek:  (*int)(b.StartOfTheWeek),
		IsOnboarded:     b.IsOnboarded,
		IsTourCompleted: b.IsTourCompleted,
	}
	if b.OnboardingStep != nil {
		patch.OnboardingStep = domain.OnboardingStepsPatch(*b.OnboardingStep)
	}
	// Absent keeps the workspace, null clears it.
	if b.LastWorkspaceID.IsSpecified() {
		patch.LastWorkspaceSet = true
		if id, err := b.LastWorkspaceID.Get(); err == nil {
			patch.LastWorkspaceID = &id
		}
	}
	p, err := h.uc.UpdateProfile.Execute(ctx, patch)
	if err != nil {
		return nil, err
	}
	return gen.UpdateProfile200JSONResponse(profile(p)), nil
}

// profile is p as the API shows it.
func profile(p domain.Profile) gen.Profile {
	workspace := nullable.NewNullNullable[uuid.UUID]()
	if p.LastWorkspaceID != nil {
		workspace = nullable.NewNullableWithValue(*p.LastWorkspaceID)
	}
	return gen.Profile{
		Theme:           gen.Theme(p.Theme),
		Language:        gen.Language(p.Language),
		StartOfTheWeek:  gen.StartOfTheWeek(p.StartOfTheWeek),
		OnboardingStep:  gen.OnboardingSteps(p.OnboardingStep),
		IsOnboarded:     p.IsOnboarded,
		IsTourCompleted: p.IsTourCompleted,
		LastWorkspaceID: workspace,
		UpdatedAt:       p.UpdatedAt,
	}
}
```

`server/internal/modules/identity/adapter/http/handler_test.go`（对 Task 7 版本的差异）：

```diff
--- a/server/internal/modules/identity/adapter/http/handler_test.go
+++ b/server/internal/modules/identity/adapter/http/handler_test.go
@@ -65,13 +65,16 @@
 // fakes are the use cases behind a test server; newServer puts an idle fake
 // in place of each one left nil.
 type fakes struct {
-	register    *fakeRegister
-	login       *fakeLogin
-	refresh     *fakeRefresh
-	logout      *fakeLogout
-	listTokens  *fakeListTokens
-	createToken *fakeCreateToken
-	revokeToken *fakeRevokeToken
+	register      *fakeRegister
+	login         *fakeLogin
+	refresh       *fakeRefresh
+	logout        *fakeLogout
+	updateMe      *fakeUpdateMe
+	getProfile    *fakeGetProfile
+	updateProfile *fakeUpdateProfile
+	listTokens    *fakeListTokens
+	createToken   *fakeCreateToken
+	revokeToken   *fakeRevokeToken
 }
 
 // newServer serves the module with limits no test here reaches.
@@ -122,6 +125,15 @@
 	if f.logout == nil {
 		f.logout = &fakeLogout{}
 	}
+	if f.updateMe == nil {
+		f.updateMe = &fakeUpdateMe{}
+	}
+	if f.getProfile == nil {
+		f.getProfile = &fakeGetProfile{}
+	}
+	if f.updateProfile == nil {
+		f.updateProfile = &fakeUpdateProfile{}
+	}
 	if f.listTokens == nil {
 		f.listTokens = &fakeListTokens{}
 	}
@@ -132,7 +144,8 @@
 		f.revokeToken = &fakeRevokeToken{}
 	}
 	httpadapter.Register(router, api, httpadapter.UseCases{
-		Register: f.register, Login: f.login, Refresh: f.refresh, Logout: f.logout, GetMe: fakeGetMe{},
+		Register: f.register, Login: f.login, Refresh: f.refresh, Logout: f.logout,
+		GetMe: fakeGetMe{}, UpdateMe: f.updateMe, GetProfile: f.getProfile, UpdateProfile: f.updateProfile,
 		ListAPITokens: f.listTokens, CreateAPIToken: f.createToken, RevokeAPIToken: f.revokeToken,
 	}, s)
 	return router
```

`server/internal/modules/identity/adapter/http/me_test.go`（新文件）：

```go
package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type fakeUpdateMe struct {
	calls int
	patch domain.UserPatch
	user  domain.User
	err   error
}

func (f *fakeUpdateMe) Execute(_ context.Context, p domain.UserPatch) (domain.User, error) {
	f.calls++
	f.patch = p
	return f.user, f.err
}

// patchJSON is a PATCH with the bearer token fakeAuth accepts.
func patchJSON(path, body string) *http.Request {
	req := withToken(httptest.NewRequest(http.MethodPatch, path, strings.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// asJSON shows a patch's pointers by what they point to: null for nil.
func asJSON(v any) string {
	out, _ := json.Marshal(v)
	return string(out)
}

func TestUpdateMe(t *testing.T) {
	update := &fakeUpdateMe{user: domain.User{
		ID: userID, Email: "alice@corp.com", FirstName: "Ann", DisplayName: "alice", Timezone: "Asia/Shanghai", CreatedAt: created,
	}}
	req := patchJSON("/api/v0/me", `{"first_name":"Ann","user_timezone":"Asia/Shanghai"}`)
	apitest.Load(t).CheckRequest(t, req)

	res, body := do(t, newServer(t, fakes{updateMe: update}), req)

	want := `{"avatar_url":null,"cover_image_url":null,"created_at":"2026-09-25T10:00:00.123456Z","display_name":"alice",` +
		`"email":"alice@corp.com","first_name":"Ann","id":"` + userID.String() + `","last_name":"","user_timezone":"Asia/Shanghai"}` + "\n"
	if res.StatusCode != http.StatusOK || body != want {
		t.Errorf("PATCH /me = %d %s, want 200 %s", res.StatusCode, body, want)
	}
	if got, want := asJSON(update.patch), `{"FirstName":"Ann","LastName":null,"DisplayName":null,"Timezone":"Asia/Shanghai"}`; got != want {
		t.Errorf("use case got %s, want %s", got, want)
	}
}

// The structure is checked before the handler (M2 design 3.11): a null name
// and the e-mail address, which only the administrator changes, are 400 and
// the use case never runs. The use case's problems pass through.
func TestUpdateMeProblems(t *testing.T) {
	tests := []struct {
		name, body string
		err        error
		status     int
		want       string
		ran        bool
	}{
		{"null name", `{"first_name":null}`, nil, 400, `"errors":[{"field":"first_name","code":"invalid_format"`, false},
		{"e-mail address", `{"email":"a@b.co"}`, nil, 400, `"errors":[{"field":"email","code":"not_allowed"`, false},
		{"invalid values", `{"display_name":""}`, shared.Invalid(shared.FieldError{Field: "display_name", Code: shared.FieldTooShort, Message: "must not be empty"}),
			422, `"errors":[{"field":"display_name","code":"too_short"`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update := &fakeUpdateMe{err: tt.err}

			res, body := do(t, newServer(t, fakes{updateMe: update}), patchJSON("/api/v0/me", tt.body))

			if res.StatusCode != tt.status || !strings.Contains(body, tt.want) {
				t.Errorf("response = %d %s, want %d with %s", res.StatusCode, body, tt.status, tt.want)
			}
			if ran := update.calls == 1; ran != tt.ran {
				t.Errorf("use case ran: %v, want %v", ran, tt.ran)
			}
		})
	}
}
```

`server/internal/modules/identity/adapter/http/profile_test.go`（新文件）：

```go
package httpadapter_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type fakeGetProfile struct{ profile domain.Profile }

func (f *fakeGetProfile) Execute(context.Context) (domain.Profile, error) { return f.profile, nil }

type fakeUpdateProfile struct {
	calls   int
	patch   domain.ProfilePatch
	profile domain.Profile
	err     error
}

func (f *fakeUpdateProfile) Execute(_ context.Context, p domain.ProfilePatch) (domain.Profile, error) {
	f.calls++
	f.patch = p
	return f.profile, f.err
}

var workspaceID = uuid.MustParse("0199a2b4-0000-7000-8000-000000000007")

func TestGetProfile(t *testing.T) {
	get := &fakeGetProfile{profile: domain.Profile{
		Theme: "system", Language: "en", OnboardingStep: domain.OnboardingSteps{ProfileComplete: true}, UpdatedAt: created,
	}}
	req := withToken(httptest.NewRequest(http.MethodGet, "/api/v0/me/profile", nil))
	apitest.Load(t).CheckRequest(t, req)

	res, body := do(t, newServer(t, fakes{getProfile: get}), req)

	want := `{"is_onboarded":false,"is_tour_completed":false,"language":"en","last_workspace_id":null,` +
		`"onboarding_step":{"profile_complete":true,"workspace_create":false,"workspace_invite":false,"workspace_join":false},` +
		`"start_of_the_week":0,"theme":"system","updated_at":"2026-09-25T10:00:00.123456Z"}` + "\n"
	if res.StatusCode != http.StatusOK || body != want {
		t.Errorf("GET /me/profile = %d %s, want 200 %s", res.StatusCode, body, want)
	}
}

// Each field sent reaches the use case; an absent one stays nil, and the
// workspace is written only when sent, null clearing it.
func TestUpdateProfile(t *testing.T) {
	tests := []struct {
		name, body, patch string
	}{
		{"every field",
			`{"theme":"dark","language":"zh-CN","start_of_the_week":1,"onboarding_step":{"workspace_join":true},` +
				`"is_onboarded":true,"is_tour_completed":false,"last_workspace_id":"` + workspaceID.String() + `"}`,
			`{"Theme":"dark","Language":"zh-CN","StartOfTheWeek":1,"OnboardingStep":{"workspace_join":true},` +
				`"IsOnboarded":true,"IsTourCompleted":false,"LastWorkspaceSet":true,"LastWorkspaceID":"` + workspaceID.String() + `"}`},
		{"one step, the workspace cleared", `{"onboarding_step":{"profile_complete":true},"last_workspace_id":null}`,
			`{"Theme":null,"Language":null,"StartOfTheWeek":null,"OnboardingStep":{"profile_complete":true},` +
				`"IsOnboarded":null,"IsTourCompleted":null,"LastWorkspaceSet":true,"LastWorkspaceID":null}`},
		{"nothing", `{}`,
			`{"Theme":null,"Language":null,"StartOfTheWeek":null,"OnboardingStep":{},` +
				`"IsOnboarded":null,"IsTourCompleted":null,"LastWorkspaceSet":false,"LastWorkspaceID":null}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update := &fakeUpdateProfile{profile: domain.Profile{
				Theme: "dark", Language: "zh-CN", StartOfTheWeek: 1, OnboardingStep: domain.OnboardingSteps{WorkspaceJoin: true},
				IsOnboarded: true, LastWorkspaceID: &workspaceID, UpdatedAt: created,
			}}
			req := patchJSON("/api/v0/me/profile", tt.body)
			apitest.Load(t).CheckRequest(t, req)

			res, body := do(t, newServer(t, fakes{updateProfile: update}), req)

			want := `{"is_onboarded":true,"is_tour_completed":false,"language":"zh-CN","last_workspace_id":"` + workspaceID.String() + `",` +
				`"onboarding_step":{"profile_complete":false,"workspace_create":false,"workspace_invite":false,"workspace_join":true},` +
				`"start_of_the_week":1,"theme":"dark","updated_at":"2026-09-25T10:00:00.123456Z"}` + "\n"
			if res.StatusCode != http.StatusOK || body != want {
				t.Errorf("PATCH /me/profile = %d %s, want 200 %s", res.StatusCode, body, want)
			}
			if got := asJSON(update.patch); got != tt.patch {
				t.Errorf("use case got %s, want %s", got, tt.patch)
			}
		})
	}
}

// The structure, down to the steps' keys, is checked before the handler
// (M2 design 3.11): the use case never runs. The use case's problems pass
// through.
func TestUpdateProfileProblems(t *testing.T) {
	tests := []struct {
		name, body string
		err        error
		status     int
		want       string
		ran        bool
	}{
		{"unknown step", `{"onboarding_step":{"profile_completed":true}}`, nil, 400,
			`"errors":[{"field":"onboarding_step.profile_completed","code":"not_allowed"`, false},
		{"null theme", `{"theme":null}`, nil, 400, `"errors":[{"field":"theme","code":"invalid_format"`, false},
		{"workspace not a uuid", `{"last_workspace_id":"x"}`, nil, 400, `"errors":[{"field":"last_workspace_id","code":"invalid_format"`, false},
		{"invalid values", `{"theme":"neon"}`, shared.Invalid(shared.FieldError{Field: "theme", Code: shared.FieldInvalidFormat, Message: "is not a known theme"}),
			422, `"errors":[{"field":"theme","code":"invalid_format"`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update := &fakeUpdateProfile{err: tt.err}

			res, body := do(t, newServer(t, fakes{updateProfile: update}), patchJSON("/api/v0/me/profile", tt.body))

			if res.StatusCode != tt.status || !strings.Contains(body, tt.want) {
				t.Errorf("response = %d %s, want %d with %s", res.StatusCode, body, tt.status, tt.want)
			}
			if ran := update.calls == 1; ran != tt.ran {
				t.Errorf("use case ran: %v, want %v", ran, tt.ran)
			}
		})
	}
}
```

`server/internal/modules/identity/module.go`（对 Task 7 版本的差异）：

```diff
--- a/server/internal/modules/identity/module.go
+++ b/server/internal/modules/identity/module.go
@@ -1,7 +1,7 @@
 // Package identity is the accounts module (M2 design 3.3, 6.2): accounts,
 // profiles, sessions and personal access tokens. It brings registration,
-// login, refresh, logout, the caller's account and tokens, and the
-// authentication every other operation goes through.
+// login, refresh, logout, the caller's account, preferences and tokens, and
+// the authentication every other operation goes through.
 package identity
 
 import (
@@ -105,6 +105,9 @@
 			Refresh:       app.NewRefresh(app.RefreshDeps{Sessions: store, Tx: d.Tx, Issuance: issuance, Clock: d.Clock, Logger: d.Logger}),
 			Logout:        app.NewLogout(store, d.Clock, d.Logger),
 			GetMe:         app.NewGetMe(store),
+			UpdateMe:      app.NewUpdateMe(store, d.Clock),
+			GetProfile:    app.NewGetProfile(store),
+			UpdateProfile: app.NewUpdateProfile(store, d.Clock),
 			ListAPITokens: app.NewListAPITokens(store),
 			CreateAPIToken: app.NewCreateAPIToken(app.CreateAPITokenDeps{
 				Lock: lock, Tokens: store, Tx: d.Tx, Clock: d.Clock, Logger: d.Logger,
```

- [ ] **Step 5: 整程序测试**

`server/internal/bootstrap/account_test.go`（新文件）：

```go
package bootstrap

import (
	"net/http"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// accountApp is the wired app on a new database, with a personal access
// token of a new account: every operation that needs a token works with one
// (M2 design 12, P3).
func accountApp(t *testing.T, email string) (contract *apitest.Contract, base, token string) {
	t.Helper()
	contract = apitest.Load(t)
	base = startApp(t, testConfig(t, pgtest.NewDatabase(t), false), migrations.FS())
	return contract, base, createPAT(t, contract, base, registerAccount(t, contract, base, email).AccessToken).Token
}

// call sends a request that the contract allows and returns its status and
// body.
func call(t *testing.T, contract *apitest.Contract, method, url, token, body string) (int, string) {
	t.Helper()
	var b []byte
	if body != "" {
		b = []byte(body)
	}
	req := newRequest(t, method, url, token, b)
	contract.CheckRequest(t, req)
	res, out := send(t, req)
	contract.CheckResponse(t, req, res)
	return res.StatusCode, string(out)
}

func TestTheAccountAndItsPreferencesWithAPersonalAccessToken(t *testing.T) {
	contract, base, token := accountApp(t, "account@example.com")

	meStatus, me := call(t, contract, http.MethodPatch, base+"/api/v0/me", token, `{"first_name":"Ann","user_timezone":"Asia/Shanghai"}`)
	patchStatus, _ := call(t, contract, http.MethodPatch, base+"/api/v0/me/profile", token, `{"theme":"dark","onboarding_step":{"profile_complete":true}}`)
	getStatus, profile := call(t, contract, http.MethodGet, base+"/api/v0/me/profile", token, "")

	if meStatus != http.StatusOK || !strings.Contains(me, `"first_name":"Ann"`) || !strings.Contains(me, `"user_timezone":"Asia/Shanghai"`) {
		t.Errorf("PATCH /me = %d %s, want 200 with the new name and time zone", meStatus, me)
	}
	if patchStatus != http.StatusOK || getStatus != http.StatusOK || !strings.Contains(profile, `"theme":"dark"`) ||
		!strings.Contains(profile, `"onboarding_step":{"profile_complete":true,"workspace_create":false,`) {
		t.Errorf("PATCH /me/profile %d, then GET = %d %s; want 200 and the new theme with the step merged", patchStatus, getStatus, profile)
	}
}
```

- [ ] **Step 6: lint、测试和端到端**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。

Run: `make lint-web`
Expected: 通过。

Run: `make e2e`
Expected: P1、P2 的故事全部通过。

- [ ] **Step 7: 提交**

```bash
git add api web/packages/api-client/src/schema.gen.ts server/internal/modules/identity server/internal/bootstrap/account_test.go
```
```bash
git commit -m "feat(M2/P3a): update the account; read and update the preferences

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run: `make gen-check`
Expected: 没有差异。

**Done when:** 三个用例、两个适配器和 PAT 的整程序测试通过；四个生成物与上表相同；`make e2e` 中 P1、P2 的故事通过；`make gen-check` 干净。

---

### Task 10: 修改密码与 `password_user` 桶；每个配置的桶都有整程序测试

**Files:**
- Create: `server/internal/modules/identity/app/change_password.go`、`change_password_test.go`
- Modify: `server/internal/modules/identity/domain/errors.go`；`domain/session.go`（过渡：Task 11 加上 `deactivated`）
- Modify: `server/internal/modules/identity/adapter/postgres/queries/sessions.sql`、`sessions.go`、`credentials_test.go`；`queries/users.sql`、`users.go`（过渡）
- Modify: `server/internal/modules/identity/app/ports.go`、`fakes_test.go`（过渡）
- Modify: `api/openapi.yaml`、`api/modules/identity.yaml`（过渡）
- Modify: `server/internal/modules/identity/adapter/http/limits.go`、`limits_test.go`；`me.go`、`me_test.go`、`handler.go`、`handler_test.go`（过渡）；`server/internal/modules/identity/module.go`（过渡）
- Modify（过渡：Task 12 改 instance 的接线）: `server/internal/bootstrap/app.go`、`app_test.go`
- Create: `server/internal/bootstrap/limits_test.go`；Modify: `server/internal/bootstrap/account_test.go`（过渡）
- Generate: `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`、`server/internal/modules/identity/adapter/http/gen/server.gen.go`、`bodyshape.gen.go`、`server/internal/modules/identity/adapter/postgres/gen/users.sql.go`、`sessions.sql.go`

**Interfaces:**
- Consumes: `CredentialLock`（Task 6）；`password_user` 的配置（Task 1）；`domain.PasswordRules`、`PasswordHasher`（P1、P2）。
- Produces（spec 2.14，M2 设计 3.5、3.8、3.10、5.4）：
  - `domain.ErrCurrentPasswordIncorrect`（422 `identity.current_password_incorrect`，"The current password is incorrect."，没有 `errors`，spec 第 3 节第 2 条）；`domain.RevokeReason`、`RevokePasswordChanged`；
  - 端口：`PasswordAccount{Email, PasswordHash}`、`PasswordAccountReader`、`SessionRevoker.RevokeSessions(ctx, userID, keep, reason, now) (int, error)`；
  - 查询：`GetPasswordAccount`；`RevokeSessions` 撤销账户中除 `keep` 外**未撤销、未到期**的会话（已撤销或到期的保留原样）；
  - `app.NewChangePassword(ChangePasswordDeps{Accounts, Lock, Passwords, Sessions, Hasher, Rules, Tx, Clock, Logger})`，`Execute(ctx, ChangePasswordInput{Current, New})`：
    1. 读账户（已不存在是 401）；新密码按密码规则检查（含邮箱词干），不合规 422 `new_password`，此时不做任何 argon2；
    2. 事务外对快照校验当前密码，新密码只哈希一次；
    3. 事务：`Lock`（复核凭证）→ 哈希不等于快照时不写 → `UpdatePasswordHash` → `RevokeSessions(userID, actor.SessionID, password_changed)`：PAT 没有会话，`SessionID` 是零值，全部会话都撤销；PAT 不变；
    4. 哈希变了：对新哈希再校验一次、再做一次第 3 步（`for range 2`）；第二次仍变是 `current_password_incorrect`；
    5. 成功记 INFO "password changed"（`user_id`）；
  - 操作 `changePassword`（`POST /api/v0/me/change-password`，204，`[validation_failed, identity.current_password_incorrect, server_busy]`），`ChangePasswordRequest{current_password, new_password}` 都必填；
  - `password_user` 桶：`Limits.PasswordUser`，`limitPassword` 在调用用例之前按账户 id 扣一个单位；模块入口 `RateLimits.PasswordUser`；`bootstrap` 建第七个桶；
  - 注册与修改密码共用一个 `domain.NewPasswordRules()`。

**Tests:**
- `change_password_test.go`：`TestChangePassword`（读账户在事务外；锁、复核会话、写哈希、撤销除当前会话外的会话在一个事务里；快照只校验一次、新密码只哈希一次；两次写入的时刻都是用例的时钟；日志中没有两个密码的任何写法）；`TestChangePasswordWithATokenRevokesEverySession`；`TestChangePasswordChecksTheNewPasswordFirst`（3 个：空、弱、邮箱词干；不做 argon2）；`TestChangePasswordWithAWrongCurrentPassword`（不哈希、不开事务）；`TestChangePasswordAfterAConcurrentChange`（3 个：并发登录重新哈希了 → 成功；密码被改了 → 422；变了两次 → 422；新密码都只哈希一次）；`TestChangePasswordRechecksTheCredentialUnderTheLock`；`TestChangePasswordErrors`（5 个：没有 actor、账户已不存在、数据库错误、校验时没有 argon2 名额、哈希时没有名额）。
- `credentials_test.go`：`TestPasswordAccount`；`TestRevokeSessions`（保留的会话、已撤销的、已到期的、别的账户的都不变，只撤销一个；`keep` 为零值时连保留的也撤销）。
- `me_test.go`：`TestChangePassword`（请求体原样到达用例）；`TestChangePasswordProblems`（用例的每种错误都答成它的 problem）。
- `limits_test.go`：`TestChangePasswordLimitsByAccount`（`password_user` 为 2：同一账户第三次被拒，即使换了 IP；别的账户仍有自己的额度）。
- `bootstrap/limits_test.go`：`TestEachConfiguredBucketLimitsItsOperations`（七个桶的突发各不相同：9、8、3、5、2、1、4；可信代理后每个用例一个客户端 IP；每个操作恰好在自己桶的突发之后得到 429：没有哪个桶接在别的桶的位置上）。这关闭 P2 评审移交的"桶与配置的接线没有测试"（spec 第 7 节）。
- `bootstrap/account_test.go`：`TestChangingThePasswordWithAPersonalAccessToken`（真实数据库：PAT 修改密码 204；注册时的会话 `password_changed`，它的刷新令牌 401；PAT 仍能读 `/me`；旧密码登录 401，新密码 200）。

- [ ] **Step 1: 领域和端口**

`server/internal/modules/identity/domain/errors.go`（对 Task 6 版本的差异）：

```diff
--- a/server/internal/modules/identity/domain/errors.go
+++ b/server/internal/modules/identity/domain/errors.go
@@ -20,6 +20,10 @@
 	// ErrRefreshTokenInvalid answers every refresh that does not rotate:
 	// unknown, expired, revoked, reused or forged (M2 design 3.5).
 	ErrRefreshTokenInvalid = shared.NewError(shared.KindUnauthenticated, "identity.refresh_token_invalid", "The refresh token is not valid; sign in again.")
+	// ErrCurrentPasswordIncorrect answers a change of password whose current
+	// password is wrong, or was changed concurrently since it was verified
+	// (M2 design 3.5).
+	ErrCurrentPasswordIncorrect = shared.NewError(shared.KindInvalid, "identity.current_password_incorrect", "The current password is incorrect.")
 	// ErrAPITokenNotFound answers a revocation of a token that does not
 	// exist, is revoked already or belongs to another account: what the
 	// caller cannot see is not found (v0 design 3.5).
```

`server/internal/modules/identity/domain/session.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/modules/identity/domain/session.go
+++ b/server/internal/modules/identity/domain/session.go
@@ -12,6 +12,15 @@
 	"uuid"
 )
 
+// RevokeReason is why a session was revoked: auth_sessions.revoke_reason
+// (M2 design 3.5). Logout and reuse detection write theirs in their own
+// statements.
+type RevokeReason string
+
+// RevokePasswordChanged revokes the other sessions of an account whose
+// password its owner changed.
+const RevokePasswordChanged RevokeReason = "password_changed"
+
 // RefreshTokenPrefix starts every refresh token (M2 design 3.4).
 const RefreshTokenPrefix = "nrv_rt_"
 
```

`server/internal/modules/identity/app/ports.go`（对 Task 8 版本的差异）：

```diff
--- a/server/internal/modules/identity/app/ports.go
+++ b/server/internal/modules/identity/app/ports.go
@@ -75,6 +75,20 @@
 	// FindLoginAccount returns ErrNotFound when no account has email, a
 	// normalized address.
 	FindLoginAccount(ctx context.Context, email string) (LoginAccount, error)
+}
+
+// PasswordAccount is what changing the password reads of the account
+// before its transaction: the address, for the password rules, and the
+// hash, as the snapshot (M2 design 3.5).
+type PasswordAccount struct {
+	Email        string // normalized
+	PasswordHash string
+}
+
+// PasswordAccountReader reads an account whose password is to change.
+type PasswordAccountReader interface {
+	// PasswordAccount returns ErrNotFound when there is no account id.
+	PasswordAccount(ctx context.Context, id uuid.UUID) (PasswordAccount, error)
 }
 
 // LockedAccount is an account's row under the credential lock.
@@ -161,6 +175,14 @@
 	RevokeForReuse(ctx context.Context, id uuid.UUID, now time.Time) error
 }
 
+// SessionRevoker revokes an account's sessions (M2 design 3.5).
+type SessionRevoker interface {
+	// RevokeSessions revokes at now, with reason, every session of userID
+	// that is neither revoked nor expired, except keep (uuid.Nil keeps
+	// none), and returns how many it revoked.
+	RevokeSessions(ctx context.Context, userID, keep uuid.UUID, reason domain.RevokeReason, now time.Time) (int, error)
+}
+
 // SessionEnder ends sessions at logout.
 type SessionEnder interface {
 	// EndSession revokes the session with reason logout while it is at g;
```

- [ ] **Step 2: 查询**

`server/internal/modules/identity/adapter/postgres/queries/users.sql`（对 Task 8 版本的差异）：

```diff
--- a/server/internal/modules/identity/adapter/postgres/queries/users.sql
+++ b/server/internal/modules/identity/adapter/postgres/queries/users.sql
@@ -14,6 +14,13 @@
 FROM users
 WHERE email = sqlc.arg(email);
 
+-- name: GetPasswordAccount :one
+-- What changing the password reads before its transaction: the address for the password rules,
+-- the hash as the snapshot (M2 design 3.5).
+SELECT email, password
+FROM users
+WHERE id = sqlc.arg(id);
+
 -- name: LockUserForCredentials :one
 -- The account row lock of M2 design 3.5. FOR NO KEY UPDATE conflicts with itself and with
 -- FOR UPDATE, so the credential transactions of one account run one after another; it does not
```

`server/internal/modules/identity/adapter/postgres/queries/sessions.sql`（对 `605f367` 的差异）：

```diff
--- a/server/internal/modules/identity/adapter/postgres/queries/sessions.sql
+++ b/server/internal/modules/identity/adapter/postgres/queries/sessions.sql
@@ -36,6 +36,14 @@
 SET updated_at = sqlc.arg(now), revoked_at = sqlc.arg(now), revoke_reason = 'reuse_detected'
 WHERE id = sqlc.arg(id) AND revoked_at IS NULL;
 
+-- name: RevokeSessions :execrows
+-- Every live session of the account but keep, the nil uuid to keep none, with reason (M2 design
+-- 3.5). A session revoked or expired already keeps what it has.
+UPDATE auth_sessions
+SET updated_at = sqlc.arg(now), revoked_at = sqlc.arg(now), revoke_reason = sqlc.arg(reason)::text
+WHERE user_id = sqlc.arg(user_id) AND id <> sqlc.arg(keep)
+  AND revoked_at IS NULL AND expires_at > sqlc.arg(now);
+
 -- name: EndSession :execrows
 -- Logout: the same conditions as the rotation (M2 design 3.5).
 UPDATE auth_sessions
```

- [ ] **Step 3: 契约**

`api/openapi.yaml`（对 Task 9 版本的差异）：

```diff
--- a/api/openapi.yaml
+++ b/api/openapi.yaml
@@ -31,6 +31,8 @@
     $ref: 'modules/identity.yaml#/paths/~1api~1v0~1auth~1logout'
   /api/v0/me:
     $ref: 'modules/identity.yaml#/paths/~1api~1v0~1me'
+  /api/v0/me/change-password:
+    $ref: 'modules/identity.yaml#/paths/~1api~1v0~1me~1change-password'
   /api/v0/me/profile:
     $ref: 'modules/identity.yaml#/paths/~1api~1v0~1me~1profile'
   /api/v0/me/api-tokens:
```

`api/modules/identity.yaml`（对 Task 9 版本的差异）：

```diff
--- a/api/modules/identity.yaml
+++ b/api/modules/identity.yaml
@@ -156,6 +156,30 @@
                 $ref: '#/components/schemas/User'
         default:
           $ref: '#/components/responses/Problem'
+  /api/v0/me/change-password:
+    post:
+      operationId: changePassword
+      tags: [identity]
+      summary: Change the caller's password
+      description: >-
+        Needs the current password. The new one follows the rules of
+        registration. Every other session of the account is signed out: all
+        of them when the caller is a personal access token. Personal access
+        tokens keep working. Password changes have a rate limit of their own
+        per account.
+      security: [{bearer: []}]
+      x-problem-codes: [validation_failed, identity.current_password_incorrect, server_busy]
+      requestBody:
+        required: true
+        content:
+          application/json:
+            schema:
+              $ref: '#/components/schemas/ChangePasswordRequest'
+      responses:
+        '204':
+          description: Changed.
+        default:
+          $ref: '#/components/responses/Problem'
   /api/v0/me/profile:
     get:
       operationId: getProfile
@@ -390,6 +414,15 @@
         created_at:
           type: string
           format: date-time
+    ChangePasswordRequest:
+      type: object
+      additionalProperties: false
+      required: [current_password, new_password]
+      properties:
+        current_password:
+          type: string
+        new_password:
+          type: string
     UserUpdate:
       type: object
       additionalProperties: false
```

- [ ] **Step 4: 生成**

Run: `make gen`
Expected: 以下六个文件改变，其余生成物不变：

| 生成的文件 | SHA-256 | 行数 |
|---|---|---|
| `api/dist/openapi.yaml` | `1724e39fce69eead4a3db95c20cc1462a4d5137e6b4d3db89e46892a579c1270` | 795 |
| `server/internal/modules/identity/adapter/http/gen/bodyshape.gen.go` | `764e8d142ca73dc182612f5f08b2d0dd167caf4c83e353556ced66f09bcb6ef7` | 59 |
| `server/internal/modules/identity/adapter/http/gen/server.gen.go` | `16048e2e593bfa8e6c82796d9d84772ad8af82dbb83fe9689ba8851f2f81a1bf` | 1711 |
| `server/internal/modules/identity/adapter/postgres/gen/sessions.sql.go` | `f4a4867da5e61f09c9f55dce12abf7fd80a24d20393fc411f6d7fb0c4966b033` | 209 |
| `server/internal/modules/identity/adapter/postgres/gen/users.sql.go` | `4bc7f318e9aa5c7e4b6e8cdecb32be2ec7c44f66202d1625248ff32445ae232e` | 209 |
| `web/packages/api-client/src/schema.gen.ts` | `c9f4f4271e1f9b80f7c7b1ace04e4e8302ddf88f8743882111787b0107e80c29` | 775 |

- [ ] **Step 5: 存储**

`server/internal/modules/identity/adapter/postgres/users.go`（对 Task 8 版本的差异）：

```diff
--- a/server/internal/modules/identity/adapter/postgres/users.go
+++ b/server/internal/modules/identity/adapter/postgres/users.go
@@ -86,6 +86,16 @@
 	return app.LoginAccount{ID: row.ID, PasswordHash: row.Password}, nil
 }
 
+// PasswordAccount reads account id's address and hash; app.ErrNotFound when
+// there is none.
+func (s *Store) PasswordAccount(ctx context.Context, id uuid.UUID) (app.PasswordAccount, error) {
+	row, err := s.queries(ctx).GetPasswordAccount(ctx, id)
+	if err != nil {
+		return app.PasswordAccount{}, notFound(err)
+	}
+	return app.PasswordAccount{Email: row.Email, PasswordHash: row.Password}, nil
+}
+
 // LockForCredentials locks account id's row until the transaction ends and
 // reads it; app.ErrNotFound when there is none. Outside a transaction the
 // lock would end with the statement: call it inside one.
```

`server/internal/modules/identity/adapter/postgres/sessions.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/modules/identity/adapter/postgres/sessions.go
+++ b/server/internal/modules/identity/adapter/postgres/sessions.go
@@ -81,6 +81,16 @@
 	return nil
 }
 
+// RevokeSessions revokes userID's live sessions but keep at now, with
+// reason, and returns how many.
+func (s *Store) RevokeSessions(ctx context.Context, userID, keep uuid.UUID, reason domain.RevokeReason, now time.Time) (int, error) {
+	n, err := s.queries(ctx).RevokeSessions(ctx, gen.RevokeSessionsParams{Now: now, Reason: string(reason), UserID: userID, Keep: keep})
+	if err != nil {
+		return 0, fmt.Errorf("revoke sessions: %w", err)
+	}
+	return int(n), nil
+}
+
 // EndSession revokes the session with reason logout while it is at g;
 // false when it is not.
 func (s *Store) EndSession(ctx context.Context, g app.SessionGeneration) (bool, error) {
```

`server/internal/modules/identity/adapter/postgres/credentials_test.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/modules/identity/adapter/postgres/credentials_test.go
+++ b/server/internal/modules/identity/adapter/postgres/credentials_test.go
@@ -11,6 +11,7 @@
 	"github.com/jackc/pgx/v5/pgconn"
 
 	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
+	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
 	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
 )
 
@@ -32,6 +33,75 @@
 	}
 }
 
+func TestPasswordAccount(t *testing.T) {
+	s, _ := newStore(t)
+	u := newUser("alice@corp.com")
+	mustCreate(t, s, u)
+
+	got, err := s.PasswordAccount(context.Background(), u.ID)
+
+	if want := (app.PasswordAccount{Email: "alice@corp.com", PasswordHash: u.PasswordHash}); err != nil || got != want {
+		t.Errorf("PasswordAccount() = %+v, %v; want %+v", got, err, want)
+	}
+	if _, err := s.PasswordAccount(context.Background(), uuid.NewV7()); !errors.Is(err, app.ErrNotFound) {
+		t.Errorf("PasswordAccount(unknown) = %v, want app.ErrNotFound", err)
+	}
+}
+
+// RevokeSessions revokes the account's live sessions but the one kept, and
+// only them: a session revoked or expired already keeps what it has, and
+// another account's is untouched (M2 design 3.5).
+func TestRevokeSessions(t *testing.T) {
+	s, pool := newStore(t)
+	alice, bob := newUser("alice@corp.com"), newUser("bob@corp.com")
+	mustCreate(t, s, alice)
+	mustCreate(t, s, bob)
+	session := func(u app.NewUser, expires time.Time) uuid.UUID {
+		t.Helper()
+		n := app.NewSession{ID: uuid.NewV7(), UserID: u.ID, TokenHash: secretHash[:], ExpiresAt: expires, Now: now}
+		if err := s.CreateSession(context.Background(), n); err != nil {
+			t.Fatal(err)
+		}
+		return n.ID
+	}
+	kept, live, expired, loggedOut, bobs := session(alice, sessionEnd), session(alice, sessionEnd),
+		session(alice, now.Add(time.Second)), session(alice, sessionEnd), session(bob, sessionEnd)
+	exec(t, pool, `UPDATE auth_sessions SET revoked_at = $2, revoke_reason = 'logout' WHERE id = $1`, loggedOut, now)
+
+	n, err := s.RevokeSessions(context.Background(), alice.ID, kept, domain.RevokePasswordChanged, later)
+
+	if err != nil || n != 1 {
+		t.Fatalf("RevokeSessions() = %d, %v; want 1", n, err)
+	}
+	tests := []struct {
+		name            string
+		id              uuid.UUID
+		reason          string // "" for none
+		revoked, update time.Time
+	}{
+		{"the kept session", kept, "", time.Time{}, now},
+		{"a live session", live, "password_changed", later, later},
+		{"an expired session", expired, "", time.Time{}, now},
+		{"a session logged out", loggedOut, "logout", now, now},
+		{"another account's session", bobs, "", time.Time{}, now},
+	}
+	for _, tt := range tests {
+		r := readSession(t, pool, tt.id)
+		var reason string
+		var revoked time.Time
+		if r.reason != nil {
+			reason, revoked = *r.reason, *r.revoked
+		}
+		if reason != tt.reason || !revoked.Equal(tt.revoked) || !r.updated.Equal(tt.update) {
+			t.Errorf("%s: reason %q revoked %v updated %v; want %q, %v, %v", tt.name, reason, revoked, r.updated, tt.reason, tt.revoked, tt.update)
+		}
+	}
+	// uuid.Nil keeps none.
+	if n, err := s.RevokeSessions(context.Background(), alice.ID, uuid.Nil(), domain.RevokePasswordChanged, later); err != nil || n != 1 {
+		t.Errorf("RevokeSessions(keep none) = %d, %v; want the kept one revoked", n, err)
+	}
+}
+
 func TestLockForCredentialsReadsTheRow(t *testing.T) {
 	s, pool := newStore(t)
 	u := newUser("alice@corp.com")
```

- [ ] **Step 6: 用例**

`server/internal/modules/identity/app/change_password.go`（新文件）：

```go
package app

import (
	"context"
	"errors"
	"log/slog"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ChangePasswordDeps are ChangePassword's collaborators.
type ChangePasswordDeps struct {
	Accounts  PasswordAccountReader
	Lock      CredentialLock
	Passwords PasswordHashWriter
	Sessions  SessionRevoker
	Hasher    PasswordHasher
	Rules     *domain.PasswordRules
	Tx        shared.TxManager
	Clock     Clock
	Logger    *slog.Logger
}

// ChangePassword changes the caller's password:
// POST /api/v0/me/change-password.
type ChangePassword struct {
	d ChangePasswordDeps
}

// NewChangePassword returns the use case.
func NewChangePassword(d ChangePasswordDeps) *ChangePassword {
	return &ChangePassword{d: d}
}

// ChangePasswordInput is the current password and the new one.
type ChangePasswordInput struct {
	Current string
	New     string
}

// Execute changes the caller's password (M2 design 3.5, 3.8):
//
//  1. the new password is checked by the password rules, with the
//     account's address for the stem rule: 422 validation_failed on
//     new_password;
//  2. outside the transaction, the current password is verified against
//     the account's hash, the snapshot, and the new one is hashed; a wrong
//     current password is 422 identity.current_password_incorrect;
//  3. one transaction takes the credential lock, checks that the hash still
//     equals the snapshot, writes the new hash and revokes the account's
//     other sessions with reason password_changed: all of them when the
//     caller is a personal access token, which has no session. Tokens stay.
//
// When the hash changed in between, a concurrent login hashed the password
// again, or the password changed: the current password is verified against
// the new hash, and step 3 is done once more. A second change fails.
func (c *ChangePassword) Execute(ctx context.Context, in ChangePasswordInput) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	account, err := c.d.Accounts.PasswordAccount(ctx, actor.UserID)
	switch {
	case errors.Is(err, ErrNotFound):
		// Authentication found the account a moment ago; it is gone now.
		return shared.Unauthenticated()
	case err != nil:
		return err
	}
	if f := c.d.Rules.Check("new_password", in.New, account.Email); f != nil {
		return shared.Invalid(*f)
	}

	snapshot, hash := account.PasswordHash, ""
	for range 2 {
		ok, _, err := c.d.Hasher.Verify(ctx, in.Current, snapshot)
		if err != nil {
			return err
		}
		if !ok {
			break
		}
		if hash == "" {
			if hash, err = c.d.Hasher.Hash(ctx, in.New); err != nil {
				return err
			}
		}
		found, err := c.change(ctx, actor, snapshot, hash)
		if err != nil {
			return err
		}
		if found == snapshot {
			c.d.Logger.InfoContext(ctx, "password changed", slog.String("user_id", actor.UserID.String()))
			return nil
		}
		snapshot = found
	}
	return domain.ErrCurrentPasswordIncorrect
}

// change is step 3. It returns the hash it found under the lock: when that
// is not snapshot, it wrote nothing.
func (c *ChangePassword) change(ctx context.Context, actor shared.Actor, snapshot, hash string) (string, error) {
	now := c.d.Clock.Now()
	found := snapshot
	err := c.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		locked, err := c.d.Lock.Lock(ctx, actor, now)
		if err != nil {
			return err
		}
		if found = locked.PasswordHash; found != snapshot {
			return nil
		}
		if err := c.d.Passwords.UpdatePasswordHash(ctx, actor.UserID, hash, now); err != nil {
			return err
		}
		_, err = c.d.Sessions.RevokeSessions(ctx, actor.UserID, actor.SessionID, domain.RevokePasswordChanged, now)
		return err
	})
	return found, err
}
```

`server/internal/modules/identity/app/fakes_test.go`（对 Task 9 版本的差异）：

```diff
--- a/server/internal/modules/identity/app/fakes_test.go
+++ b/server/internal/modules/identity/app/fakes_test.go
@@ -293,15 +293,36 @@
 }
 
 // fakeCredentials is the account row and the session that the credential
-// lock reads.
+// lock reads. A change of password reads the row and writes it and the
+// sessions: it logs them too.
 type fakeCredentials struct {
 	log        *callLog
+	email      string
 	account    app.LockedAccount
 	accountErr error
 	session    app.SessionCredential
 	sessionErr error
+	writtenAt  []time.Time // the times the writes were given
 }
 
+func (f *fakeCredentials) PasswordAccount(ctx context.Context, id uuid.UUID) (app.PasswordAccount, error) {
+	f.log.add(ctx, "read "+id.String())
+	return app.PasswordAccount{Email: f.email, PasswordHash: f.account.PasswordHash}, f.accountErr
+}
+
+func (f *fakeCredentials) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string, now time.Time) error {
+	f.log.add(ctx, "password "+id.String()+" "+hash)
+	f.writtenAt = append(f.writtenAt, now)
+	f.account.PasswordHash = hash
+	return nil
+}
+
+func (f *fakeCredentials) RevokeSessions(ctx context.Context, userID, keep uuid.UUID, reason domain.RevokeReason, now time.Time) (int, error) {
+	f.log.add(ctx, "revoke "+string(reason)+" sessions of "+userID.String()+" but "+keep.String())
+	f.writtenAt = append(f.writtenAt, now)
+	return 1, nil
+}
+
 func (f *fakeCredentials) LockForCredentials(ctx context.Context, id uuid.UUID) (app.LockedAccount, error) {
 	f.log.add(ctx, "lock "+id.String())
 	return f.account, f.accountErr
```

`server/internal/modules/identity/app/change_password_test.go`（新文件）：

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

// newPasswordChange is ChangePassword over the credential fixture, whose
// account zqxwv@corp.com has the password Tr0ub4dor&3.
func newPasswordChange() (*credentialFixture, *fakeHasher, *app.ChangePassword) {
	f, h := newCredentialFixture(), &fakeHasher{}
	f.creds.email = "zqxwv@corp.com"
	return f, h, app.NewChangePassword(app.ChangePasswordDeps{
		Accounts: f.creds, Lock: f.lock(), Passwords: f.creds, Sessions: f.creds, Hasher: h,
		Rules: domain.NewPasswordRules(), Tx: f.tx, Clock: clocktest.At(now), Logger: f.logger(),
	})
}

var change = app.ChangePasswordInput{Current: "Tr0ub4dor&3", New: "N3w-Passw0rd!"}

func TestChangePassword(t *testing.T) {
	f, h, uc := newPasswordChange()

	err := uc.Execute(shared.WithActor(context.Background(), sessionActor), change)

	// The snapshot is read and argon2 runs outside the transaction; the
	// lock, the recheck of the session and the writes are one transaction
	// (M2 design 3.5). The session the request came with stays.
	want := []string{
		"read " + userID.String() + " outside tx", "lock " + userID.String(), "session " + sessionID.String(),
		"password " + userID.String() + " hashed:N3w-Passw0rd!",
		"revoke password_changed sessions of " + userID.String() + " but " + sessionID.String(),
	}
	if err != nil || !slices.Equal(f.log.calls, want) || f.tx.calls != 1 {
		t.Errorf("Execute() = %v, calls %q in %d transactions; want %q in one", err, f.log.calls, f.tx.calls, want)
	}
	if !slices.Equal(h.verified, []string{"hashed:Tr0ub4dor&3"}) || h.calls != 1 || !slices.Equal(f.creds.writtenAt, []time.Time{now, now}) {
		t.Errorf("verified %q, %d hashes, writes at %v; want the snapshot verified, one hash, both writes at %v", h.verified, h.calls, f.creds.writtenAt, now)
	}
	logs := f.logs.String()
	if !strings.Contains(logs, `"msg":"password changed"`) || !strings.Contains(logs, `"user_id":"`+userID.String()) {
		t.Errorf("logs = %s, want the change with user_id", logs)
	}
	assertNoSecret(t, logs, "current password", []byte(change.Current))
	assertNoSecret(t, logs, "new password", []byte(change.New))
}

// A personal access token has no session: every session is revoked, and
// the lock checks the token again (M2 design 3.5).
func TestChangePasswordWithATokenRevokesEverySession(t *testing.T) {
	f, _, uc := newPasswordChange()

	err := uc.Execute(shared.WithActor(context.Background(), tokenActor), change)

	want := []string{
		"lock " + userID.String(), "token " + tokenID.String(), "password " + userID.String() + " hashed:N3w-Passw0rd!",
		"revoke password_changed sessions of " + userID.String() + " but " + uuid.Nil().String(),
	}
	if err != nil || !slices.Equal(f.log.calls[1:], want) {
		t.Errorf("Execute() = %v, calls %q; want %q after the read", err, f.log.calls, want)
	}
}

// The new password is checked, with the account's address, before any
// argon2 (M2 design 3.8).
func TestChangePasswordChecksTheNewPasswordFirst(t *testing.T) {
	tests := []struct{ name, password, code string }{
		{"empty", "", shared.FieldRequired},
		{"weak", "password", shared.FieldWeakPassword},
		{"the stem of the address", "Zqxwv123!", shared.FieldCommonPassword},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, h, uc := newPasswordChange()

			err := uc.Execute(shared.WithActor(context.Background(), sessionActor), app.ChangePasswordInput{Current: change.Current, New: tt.password})

			var se *shared.Error
			if !errors.As(err, &se) || se.Code != shared.CodeValidationFailed || len(se.Fields) != 1 ||
				se.Fields[0].Field != "new_password" || se.Fields[0].Code != tt.code {
				t.Errorf("Execute() = %v, want validation_failed with new_password %s", err, tt.code)
			}
			if len(h.verified)+h.calls != 0 || len(f.log.calls) != 1 {
				t.Errorf("argon2 ran %d times, calls %q; want only the read", len(h.verified)+h.calls, f.log.calls)
			}
		})
	}
}

func TestChangePasswordWithAWrongCurrentPassword(t *testing.T) {
	f, h, uc := newPasswordChange()

	err := uc.Execute(shared.WithActor(context.Background(), sessionActor), app.ChangePasswordInput{Current: "Tr0ub4dor&4", New: change.New})

	if !errors.Is(err, domain.ErrCurrentPasswordIncorrect) || h.calls != 0 || f.tx.calls != 0 {
		t.Errorf("Execute() = %v after %d hashes and %d transactions, want identity.current_password_incorrect before either", err, h.calls, f.tx.calls)
	}
}

// When the hash under the lock is not the snapshot, the current password
// is verified against it and the transaction runs once more, with the new
// password hashed once (M2 design 3.5).
func TestChangePasswordAfterAConcurrentChange(t *testing.T) {
	tests := []struct {
		name     string
		hashes   []string // what the row holds after the first, second, … verification
		err      error
		verified []string
		changed  bool
	}{
		{"a login hashed the password again", []string{"hashed:Tr0ub4dor&3"}, nil,
			[]string{"old:Tr0ub4dor&3", "hashed:Tr0ub4dor&3"}, true},
		{"the password changed", []string{"hashed:Other-Passw0rd!"}, domain.ErrCurrentPasswordIncorrect,
			[]string{"old:Tr0ub4dor&3", "hashed:Other-Passw0rd!"}, false},
		{"it changed twice", []string{"hashed:Tr0ub4dor&3", "old:Tr0ub4dor&3"}, domain.ErrCurrentPasswordIncorrect,
			[]string{"old:Tr0ub4dor&3", "hashed:Tr0ub4dor&3"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, h, uc := newPasswordChange()
			f.creds.account.PasswordHash = "old:Tr0ub4dor&3"
			h.onVerify = func() {
				if i := len(h.verified) - 1; i < len(tt.hashes) {
					f.creds.account.PasswordHash = tt.hashes[i] // a transaction that commits meanwhile
				}
			}

			err := uc.Execute(shared.WithActor(context.Background(), sessionActor), change)

			if !errors.Is(err, tt.err) || !slices.Equal(h.verified, tt.verified) || h.calls != 1 {
				t.Errorf("Execute() = %v, verified %q, %d hashes; want %v, %q, one hash", err, h.verified, h.calls, tt.err, tt.verified)
			}
			if changed := f.creds.account.PasswordHash == "hashed:N3w-Passw0rd!"; changed != tt.changed {
				t.Errorf("the row holds %q, want the new password: %v", f.creds.account.PasswordHash, tt.changed)
			}
		})
	}
}

// Under the lock the caller's credential is checked again: a session that
// a concurrent reset revoked changes nothing (M2 design 3.5).
func TestChangePasswordRechecksTheCredentialUnderTheLock(t *testing.T) {
	f, _, uc := newPasswordChange()
	f.creds.session.Revoked = true

	err := uc.Execute(shared.WithActor(context.Background(), sessionActor), change)

	if !errors.Is(err, shared.Unauthenticated()) || len(f.creds.writtenAt) != 0 {
		t.Errorf("Execute() = %v, %d writes; want 401 and none", err, len(f.creds.writtenAt))
	}
}

func TestChangePasswordErrors(t *testing.T) {
	boom, busy := errors.New("connection refused"), shared.ServerBusy(time.Second)
	authed := shared.WithActor(context.Background(), sessionActor)
	tests := []struct {
		name   string
		ctx    context.Context
		change func(*credentialFixture, *fakeHasher)
		want   error
	}{
		{"without an actor", context.Background(), func(*credentialFixture, *fakeHasher) {}, shared.Unauthenticated()},
		{"the account gone", authed, func(f *credentialFixture, _ *fakeHasher) { f.creds.accountErr = app.ErrNotFound }, shared.Unauthenticated()},
		{"the database down", authed, func(f *credentialFixture, _ *fakeHasher) { f.creds.accountErr = boom }, boom},
		{"no argon2 slot to verify", authed, func(_ *credentialFixture, h *fakeHasher) { h.verifyErr = busy }, busy},
		{"no argon2 slot to hash", authed, func(_ *credentialFixture, h *fakeHasher) { h.err = busy }, busy},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, h, uc := newPasswordChange()
			tt.change(f, h)

			if err := uc.Execute(tt.ctx, change); !errors.Is(err, tt.want) || len(f.creds.writtenAt) != 0 {
				t.Errorf("Execute() = %v, %d writes; want %v and none", err, len(f.creds.writtenAt), tt.want)
			}
		})
	}
}
```

- [ ] **Step 7: HTTP 适配器、桶和接线**

`server/internal/modules/identity/adapter/http/limits.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/modules/identity/adapter/http/limits.go
+++ b/server/internal/modules/identity/adapter/http/limits.go
@@ -27,6 +27,7 @@
 	LoginIP      *ratelimit.Bucket // ratelimit.login_ip, by client IP key
 	LoginIPEmail *ratelimit.Bucket // ratelimit.login_ip_email, by client IP key and address
 	RegisterIP   *ratelimit.Bucket // ratelimit.register_ip, by client IP key
+	PasswordUser *ratelimit.Bucket // ratelimit.password_user, by account
 }
 
 // limitLogin takes a unit of login_ip and one of login_ip_email, or
@@ -44,6 +45,17 @@
 	return h.allow(ctx, ratelimit.Check{Bucket: h.s.Limits.RegisterIP, Key: httpserver.RequestMetaFrom(ctx).IPKey})
 }
 
+// limitPassword takes a unit of password_user. Its key is the caller's
+// account, which authentication verified: no one else can use up its units
+// (M2 design 3.10).
+func (h handler) limitPassword(ctx context.Context) error {
+	actor, err := shared.RequireActor(ctx)
+	if err != nil {
+		return err
+	}
+	return h.allow(ctx, ratelimit.Check{Bucket: h.s.Limits.PasswordUser, Key: actor.UserID.String()})
+}
+
 // allow answers 429 rate_limited with Retry-After when a bucket refuses,
 // and logs which bucket turned the client away (M2 design 8.4), as the
 // platform does for its own.
```

`server/internal/modules/identity/adapter/http/handler.go`（对 Task 9 版本的差异）：

```diff
--- a/server/internal/modules/identity/adapter/http/handler.go
+++ b/server/internal/modules/identity/adapter/http/handler.go
@@ -47,6 +47,11 @@
 	Execute(ctx context.Context, p domain.UserPatch) (domain.User, error)
 }
 
+// ChangePasswordUseCase is app.ChangePassword.
+type ChangePasswordUseCase interface {
+	Execute(ctx context.Context, in app.ChangePasswordInput) error
+}
+
 // GetProfileUseCase is app.GetProfile.
 type GetProfileUseCase interface {
 	Execute(ctx context.Context) (domain.Profile, error)
@@ -80,6 +85,7 @@
 	Logout         LogoutUseCase
 	GetMe          GetMeUseCase
 	UpdateMe       UpdateMeUseCase
+	ChangePassword ChangePasswordUseCase
 	GetProfile     GetProfileUseCase
 	UpdateProfile  UpdateProfileUseCase
 	ListAPITokens  ListAPITokensUseCase
```

`server/internal/modules/identity/adapter/http/me.go`（对 Task 9 版本的差异）：

```diff
--- a/server/internal/modules/identity/adapter/http/me.go
+++ b/server/internal/modules/identity/adapter/http/me.go
@@ -6,6 +6,7 @@
 	"github.com/oapi-codegen/nullable"
 
 	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/http/gen"
+	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
 	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
 )
 
@@ -32,6 +33,19 @@
 	return gen.UpdateMe200JSONResponse(user(u)), nil
 }
 
+// ChangePassword serves POST /api/v0/me/change-password. password_user
+// limits it per account (M2 design 3.10): each attempt costs argon2.
+func (h handler) ChangePassword(ctx context.Context, req gen.ChangePasswordRequestObject) (gen.ChangePasswordResponseObject, error) {
+	if err := h.limitPassword(ctx); err != nil {
+		return nil, err
+	}
+	err := h.uc.ChangePassword.Execute(ctx, app.ChangePasswordInput{Current: req.Body.CurrentPassword, New: req.Body.NewPassword})
+	if err != nil {
+		return nil, err
+	}
+	return gen.ChangePassword204Response{}, nil
+}
+
 // user is u as the API shows it.
 func user(u domain.User) gen.User {
 	return gen.User{
```

`server/internal/modules/identity/adapter/http/handler_test.go`（对 Task 9 版本的差异）：

```diff
--- a/server/internal/modules/identity/adapter/http/handler_test.go
+++ b/server/internal/modules/identity/adapter/http/handler_test.go
@@ -52,14 +52,20 @@
 	return domain.User{ID: actor.UserID, Email: "alice@corp.com", DisplayName: "alice", Timezone: "UTC", CreatedAt: created}, nil
 }
 
-// fakeAuth accepts the token "valid" as the account userID.
+// fakeAuth accepts the token "valid" as a session of the account userID,
+// and "other" as a personal access token of otherUserID.
 type fakeAuth struct{}
 
+var otherUserID = uuid.MustParse("0199a2b4-0000-7000-8000-000000000009")
+
 func (fakeAuth) Authenticate(ctx context.Context, token string) (context.Context, string, error) {
-	if token != "valid" {
-		return nil, "", shared.Unauthenticated()
+	switch token {
+	case "valid":
+		return shared.WithActor(ctx, shared.Actor{UserID: userID, SessionID: sessionID}), "session:" + sessionID.String(), nil
+	case "other":
+		return shared.WithActor(ctx, shared.Actor{UserID: otherUserID, APITokenID: tokenID}), "pat:" + tokenID.String(), nil
 	}
-	return shared.WithActor(ctx, shared.Actor{UserID: userID, SessionID: sessionID}), "session:" + sessionID.String(), nil
+	return nil, "", shared.Unauthenticated()
 }
 
 // fakes are the use cases behind a test server; newServer puts an idle fake
@@ -70,6 +76,7 @@
 	refresh       *fakeRefresh
 	logout        *fakeLogout
 	updateMe      *fakeUpdateMe
+	change        *fakeChangePassword
 	getProfile    *fakeGetProfile
 	updateProfile *fakeUpdateProfile
 	listTokens    *fakeListTokens
@@ -85,7 +92,10 @@
 		return limiter.Bucket(name, ratelimit.Rate{PerMinute: 600, Burst: 100})
 	}
 	return serverWith(t, f, httpadapter.Settings{
-		Limits:          httpadapter.Limits{Limiter: limiter, LoginIP: roomy("login_ip"), LoginIPEmail: roomy("login_ip_email"), RegisterIP: roomy("register_ip")},
+		Limits: httpadapter.Limits{
+			Limiter: limiter, LoginIP: roomy("login_ip"), LoginIPEmail: roomy("login_ip_email"),
+			RegisterIP: roomy("register_ip"), PasswordUser: roomy("password_user"),
+		},
 		RefreshDeadline: refreshDeadline,
 		Logger:          slog.New(slog.DiscardHandler),
 	})
@@ -128,6 +138,9 @@
 	if f.updateMe == nil {
 		f.updateMe = &fakeUpdateMe{}
 	}
+	if f.change == nil {
+		f.change = &fakeChangePassword{}
+	}
 	if f.getProfile == nil {
 		f.getProfile = &fakeGetProfile{}
 	}
@@ -145,7 +158,7 @@
 	}
 	httpadapter.Register(router, api, httpadapter.UseCases{
 		Register: f.register, Login: f.login, Refresh: f.refresh, Logout: f.logout,
-		GetMe: fakeGetMe{}, UpdateMe: f.updateMe, GetProfile: f.getProfile, UpdateProfile: f.updateProfile,
+		GetMe: fakeGetMe{}, UpdateMe: f.updateMe, ChangePassword: f.change, GetProfile: f.getProfile, UpdateProfile: f.updateProfile,
 		ListAPITokens: f.listTokens, CreateAPIToken: f.createToken, RevokeAPIToken: f.revokeToken,
 	}, s)
 	return router
```

`server/internal/modules/identity/adapter/http/limits_test.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/modules/identity/adapter/http/limits_test.go
+++ b/server/internal/modules/identity/adapter/http/limits_test.go
@@ -17,7 +17,7 @@
 
 // tightLimits are small buckets on a limiter whose clock stands still, so
 // nothing refills during a test: login_ip 3, login_ip_email 2,
-// register_ip 1, each regaining a unit a minute.
+// register_ip 1, password_user 2, each regaining a unit a minute.
 func tightLimits() httpadapter.Limits {
 	limiter := ratelimit.New(func() time.Time { return created })
 	return httpadapter.Limits{
@@ -25,6 +25,7 @@
 		LoginIP:      limiter.Bucket("login_ip", ratelimit.Rate{PerMinute: 1, Burst: 3}),
 		LoginIPEmail: limiter.Bucket("login_ip_email", ratelimit.Rate{PerMinute: 1, Burst: 2}),
 		RegisterIP:   limiter.Bucket("register_ip", ratelimit.Rate{PerMinute: 1, Burst: 1}),
+		PasswordUser: limiter.Bucket("password_user", ratelimit.Rate{PerMinute: 1, Burst: 2}),
 	}
 }
 
@@ -100,6 +101,34 @@
 	}
 }
 
+// password_user counts the account, whatever the client (M2 design 3.10):
+// the account's third change is refused although it comes from another IP,
+// and another account still has its own units.
+func TestChangePasswordLimitsByAccount(t *testing.T) {
+	var logs bytes.Buffer
+	change := &fakeChangePassword{err: domain.ErrCurrentPasswordIncorrect}
+	h := limitedServer(t, fakes{change: change}, tightLimits(), &logs)
+	attempt := func(token, peer string) int {
+		req := postJSON("/api/v0/me/change-password", `{"current_password":"x","new_password":"y"}`)
+		req.Header.Set("Authorization", "Bearer "+token)
+		req.RemoteAddr = peer
+		res, _ := do(t, h, req)
+		return res.StatusCode
+	}
+
+	statuses := []int{
+		attempt("valid", "203.0.113.7:5555"), attempt("valid", "198.51.100.9:5555"),
+		attempt("valid", "192.0.2.1:5555"), attempt("other", "203.0.113.7:5555"),
+	}
+
+	if want := []int{422, 422, 429, 422}; !slices.Equal(statuses, want) || change.calls != 3 {
+		t.Errorf("statuses = %v after %d changes, want %v after 3", statuses, change.calls, want)
+	}
+	if entries := rateLimitLogs(t, &logs); len(entries) != 1 || entries[0]["bucket"] != "password_user" {
+		t.Errorf("rate limited logs = %v, want password_user", entries)
+	}
+}
+
 // The module's buckets count a client by its IP key, so an IPv6 /64 is one
 // client; login_ip_email counts that key and the address together, so
 // another client trying the same address has a bucket of its own.
```

`server/internal/modules/identity/adapter/http/me_test.go`（对 Task 9 版本的差异）：

```diff
--- a/server/internal/modules/identity/adapter/http/me_test.go
+++ b/server/internal/modules/identity/adapter/http/me_test.go
@@ -7,7 +7,9 @@
 	"net/http/httptest"
 	"strings"
 	"testing"
+	"time"
 
+	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
 	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
 	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
 	"github.com/open-nerve/NerveProject/server/internal/shared"
@@ -26,6 +28,18 @@
 	return f.user, f.err
 }
 
+type fakeChangePassword struct {
+	calls int
+	in    app.ChangePasswordInput
+	err   error
+}
+
+func (f *fakeChangePassword) Execute(_ context.Context, in app.ChangePasswordInput) error {
+	f.calls++
+	f.in = in
+	return f.err
+}
+
 // patchJSON is a PATCH with the bearer token fakeAuth accepts.
 func patchJSON(path, body string) *http.Request {
 	req := withToken(httptest.NewRequest(http.MethodPatch, path, strings.NewReader(body)))
@@ -89,3 +103,42 @@
 		})
 	}
 }
+
+func TestChangePassword(t *testing.T) {
+	change := &fakeChangePassword{}
+	req := withToken(postJSON("/api/v0/me/change-password", `{"current_password":"Tr0ub4dor&3","new_password":"N3w-Passw0rd!"}`))
+	apitest.Load(t).CheckRequest(t, req)
+
+	res, body := do(t, newServer(t, fakes{change: change}), req)
+
+	want := app.ChangePasswordInput{Current: "Tr0ub4dor&3", New: "N3w-Passw0rd!"}
+	if res.StatusCode != http.StatusNoContent || body != "" || change.in != want {
+		t.Errorf("POST /me/change-password = %d %q, use case got %+v; want 204 for %+v", res.StatusCode, body, change.in, want)
+	}
+}
+
+// The handler exit: every error the use case returns becomes its problem.
+func TestChangePasswordProblems(t *testing.T) {
+	tests := []struct {
+		name       string
+		err        error
+		status     int
+		code       string
+		retryAfter string
+	}{
+		{"weak new password", shared.Invalid(shared.FieldError{Field: "new_password", Code: shared.FieldWeakPassword, Message: "is weak"}),
+			422, "validation_failed", ""},
+		{"wrong current password", domain.ErrCurrentPasswordIncorrect, 422, "identity.current_password_incorrect", ""},
+		{"hashing saturated", shared.ServerBusy(time.Second), 503, "server_busy", "1"},
+	}
+	for _, tt := range tests {
+		t.Run(tt.name, func(t *testing.T) {
+			res, body := do(t, newServer(t, fakes{change: &fakeChangePassword{err: tt.err}}),
+				withToken(postJSON("/api/v0/me/change-password", `{"current_password":"x","new_password":"y"}`)))
+
+			if res.StatusCode != tt.status || !strings.Contains(body, `"code":"`+tt.code+`"`) || res.Header.Get("Retry-After") != tt.retryAfter {
+				t.Errorf("response = %d %s Retry-After %q, want %d %s", res.StatusCode, body, res.Header.Get("Retry-After"), tt.status, tt.code)
+			}
+		})
+	}
+}
```

`server/internal/modules/identity/module.go`（对 Task 9 版本的差异）：

```diff
--- a/server/internal/modules/identity/module.go
+++ b/server/internal/modules/identity/module.go
@@ -60,6 +60,7 @@
 	LoginIP      *ratelimit.Bucket
 	LoginIPEmail *ratelimit.Bucket
 	RegisterIP   *ratelimit.Bucket
+	PasswordUser *ratelimit.Bucket
 }
 
 // Module is the wired identity module.
@@ -83,6 +84,7 @@
 	if err != nil {
 		return nil, fmt.Errorf("hash the dummy password: %w", err)
 	}
+	rules := domain.NewPasswordRules()
 	store := postgresadapter.New(d.Pool)
 	lock := app.CredentialLock{Locker: store, Sessions: store, APITokens: store}
 	tokens := signing.NewAccessTokens(keys)
@@ -95,17 +97,21 @@
 	return &Module{
 		uc: httpadapter.UseCases{
 			Register: app.NewRegister(app.RegisterDeps{
-				Policy: d.SignupPolicy, Rules: domain.NewPasswordRules(), Hasher: hasher, Tx: d.Tx,
+				Policy: d.SignupPolicy, Rules: rules, Hasher: hasher, Tx: d.Tx,
 				Users: store, Profiles: store, Sessions: store, Issuance: issuance, Clock: d.Clock, Logger: d.Logger,
 			}),
 			Login: app.NewLogin(app.LoginDeps{
 				Accounts: store, Locker: store, Passwords: store, Sessions: store, Hasher: hasher, Tx: d.Tx,
 				Issuance: issuance, Clock: d.Clock, Logger: d.Logger, DummyHash: dummy,
 			}),
-			Refresh:       app.NewRefresh(app.RefreshDeps{Sessions: store, Tx: d.Tx, Issuance: issuance, Clock: d.Clock, Logger: d.Logger}),
-			Logout:        app.NewLogout(store, d.Clock, d.Logger),
-			GetMe:         app.NewGetMe(store),
-			UpdateMe:      app.NewUpdateMe(store, d.Clock),
+			Refresh:  app.NewRefresh(app.RefreshDeps{Sessions: store, Tx: d.Tx, Issuance: issuance, Clock: d.Clock, Logger: d.Logger}),
+			Logout:   app.NewLogout(store, d.Clock, d.Logger),
+			GetMe:    app.NewGetMe(store),
+			UpdateMe: app.NewUpdateMe(store, d.Clock),
+			ChangePassword: app.NewChangePassword(app.ChangePasswordDeps{
+				Accounts: store, Lock: lock, Passwords: store, Sessions: store, Hasher: hasher,
+				Rules: rules, Tx: d.Tx, Clock: d.Clock, Logger: d.Logger,
+			}),
 			GetProfile:    app.NewGetProfile(store),
 			UpdateProfile: app.NewUpdateProfile(store, d.Clock),
 			ListAPITokens: app.NewListAPITokens(store),
```

`server/internal/bootstrap/app.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/bootstrap/app.go
+++ b/server/internal/bootstrap/app.go
@@ -96,6 +96,7 @@
 			LoginIP:      bucket(limiter, "login_ip", cfg.RateLimit.LoginIP),
 			LoginIPEmail: bucket(limiter, "login_ip_email", cfg.RateLimit.LoginIPEmail),
 			RegisterIP:   bucket(limiter, "register_ip", cfg.RateLimit.RegisterIP),
+			PasswordUser: bucket(limiter, "password_user", cfg.RateLimit.PasswordUser),
 		},
 	})
 	if err != nil {
```

`server/internal/bootstrap/app_test.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/bootstrap/app_test.go
+++ b/server/internal/bootstrap/app_test.go
@@ -71,7 +71,7 @@
 		RateLimit: config.RateLimitConfig{
 			IPv6PrefixLen: 64,
 			Anonymous:     roomy, AuthFailure: roomy, Authenticated: roomy,
-			LoginIP: roomy, LoginIPEmail: roomy, RegisterIP: roomy,
+			LoginIP: roomy, LoginIPEmail: roomy, RegisterIP: roomy, PasswordUser: roomy,
 		},
 		Log: config.LogConfig{Level: "error", Format: "text"},
 	}
```

- [ ] **Step 8: 整程序测试**

`server/internal/bootstrap/limits_test.go`（新文件）：

```go
package bootstrap

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// Each bucket of the configuration limits the operations it belongs to (M2
// design 3.10). Every bucket has a burst of its own, and each operation is
// refused after exactly its bucket's burst: no bucket is wired in another's
// place. Behind the trusted proxy each case comes from a client of its own,
// so the buckets by client IP share no units across cases.
func TestEachConfiguredBucketLimitsItsOperations(t *testing.T) {
	contract := apitest.Load(t)
	cfg := testConfig(t, pgtest.NewDatabase(t), false)
	cfg.Server.TrustedProxies = []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8")} // the test's own peer
	burst := func(n int) config.BucketConfig { return config.BucketConfig{PerMinute: 1, Burst: n} }
	cfg.RateLimit.Anonymous, cfg.RateLimit.Authenticated, cfg.RateLimit.AuthFailure = burst(9), burst(8), burst(3)
	cfg.RateLimit.LoginIP, cfg.RateLimit.LoginIPEmail = burst(5), burst(2)
	cfg.RateLimit.RegisterIP, cfg.RateLimit.PasswordUser = burst(1), burst(4)
	base := startApp(t, cfg, migrations.FS())
	from := func(client, method, path, token, body string) *http.Request {
		var b []byte
		if body != "" {
			b = []byte(body)
		}
		req := newRequest(t, method, base+path, token, b)
		req.Header.Set("X-Forwarded-For", client)
		return req
	}
	register := func(client, email string) *http.Request {
		return from(client, http.MethodPost, "/api/v0/auth/register", "", `{"email":"`+email+`","password":"Tr0ub4dor&3"}`)
	}
	accessToken := func(client, email string) string {
		res, body := send(t, register(client, email))
		var tokens authTokens
		if res.StatusCode != http.StatusCreated || json.Unmarshal(body, &tokens) != nil {
			t.Fatalf("register %s = %d %s, want 201 with tokens", email, res.StatusCode, body)
		}
		return tokens.AccessToken
	}
	reader, changer := accessToken("198.51.100.101", "reader@example.com"), accessToken("198.51.100.102", "changer@example.com")

	tests := []struct {
		bucket string
		burst  int
		next   func(i int) *http.Request
	}{
		{"anonymous", 9, func(int) *http.Request { return from("198.51.100.1", http.MethodGet, "/api/v0/instance", "", "") }},
		{"authenticated", 8, func(int) *http.Request { return from("198.51.100.2", http.MethodGet, "/api/v0/me", reader, "") }},
		{"auth_failure", 3, func(int) *http.Request { return from("198.51.100.3", http.MethodGet, "/api/v0/me", "forged", "") }},
		{"login_ip", 5, func(i int) *http.Request {
			return from("198.51.100.4", http.MethodPost, "/api/v0/auth/login", "", fmt.Sprintf(`{"email":"nobody%d@example.com","password":"x"}`, i))
		}},
		{"login_ip_email", 2, func(int) *http.Request {
			return from("198.51.100.5", http.MethodPost, "/api/v0/auth/login", "", `{"email":"nobody@example.com","password":"x"}`)
		}},
		{"register_ip", 1, func(i int) *http.Request { return register("198.51.100.6", fmt.Sprintf("new%d@example.com", i)) }},
		{"password_user", 4, func(int) *http.Request {
			return from("198.51.100.7", http.MethodPost, "/api/v0/me/change-password", changer, `{"current_password":"x","new_password":"y"}`)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.bucket, func(t *testing.T) {
			admitted := 0
			for i := range tt.burst + 1 {
				req := tt.next(i)
				res, _ := send(t, req)
				contract.CheckResponse(t, req, res)
				if res.StatusCode == http.StatusTooManyRequests {
					break
				}
				admitted++
			}
			if admitted != tt.burst {
				t.Errorf("%d requests admitted before 429, want the burst %d", admitted, tt.burst)
			}
		})
	}
}
```

`server/internal/bootstrap/account_test.go`（对 Task 9 版本的差异）：

```diff
--- a/server/internal/bootstrap/account_test.go
+++ b/server/internal/bootstrap/account_test.go
@@ -35,6 +35,32 @@
 	return res.StatusCode, string(out)
 }
 
+// Changing the password with a personal access token revokes every
+// session, the token keeps working, and only the new password signs in (M2
+// design 3.5, story A7).
+func TestChangingThePasswordWithAPersonalAccessToken(t *testing.T) {
+	base, pool := sessionApp(t)
+	contract := apitest.Load(t)
+	session := registerAccount(t, contract, base, "change@example.com")
+	token := createPAT(t, contract, base, session.AccessToken).Token
+
+	changed, body := call(t, contract, http.MethodPost, base+"/api/v0/me/change-password", token,
+		`{"current_password":"Tr0ub4dor&3","new_password":"N3w-Passw0rd!"}`)
+	reason := revocation(t, pool, "change@example.com")
+	refreshed, _ := postTokens(t, contract, base, "/api/v0/auth/refresh", session.RefreshToken)
+	me, _ := call(t, contract, http.MethodGet, base+"/api/v0/me", token, "")
+	oldLogin, _ := login(t, base, "change@example.com", "Tr0ub4dor&3")
+	newLogin, _ := login(t, base, "change@example.com", "N3w-Passw0rd!")
+
+	if changed != http.StatusNoContent || reason != "password_changed" || refreshed != http.StatusUnauthorized || me != http.StatusOK {
+		t.Errorf("change = %d %s; the session revoked for %q, its refresh %d; the token's GET /me %d; want 204, password_changed, 401, 200",
+			changed, body, reason, refreshed, me)
+	}
+	if oldLogin != http.StatusUnauthorized || newLogin != http.StatusOK {
+		t.Errorf("login with the old password %d, with the new one %d; want 401, 200", oldLogin, newLogin)
+	}
+}
+
 func TestTheAccountAndItsPreferencesWithAPersonalAccessToken(t *testing.T) {
 	contract, base, token := accountApp(t, "account@example.com")
 
```

- [ ] **Step 9: lint、测试和端到端**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。

Run: `make lint-web`
Expected: 通过。

Run: `make e2e`
Expected: P1、P2 的故事全部通过。

- [ ] **Step 10: 提交**

```bash
git add api web/packages/api-client/src/schema.gen.ts server/internal/modules/identity server/internal/bootstrap
```
```bash
git commit -m "feat(M2/P3a): change the password; the password_user bucket

Under the credential lock the new hash is written and the account's other
sessions are revoked, every session when the caller is a personal access
token; tokens stay. A whole-program test drives each configured bucket to
its own burst, which closes the bucket wiring carried from the P2 review.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run: `make gen-check`
Expected: 没有差异。

**Done when:** 修改密码的用例、存储、适配器和整程序测试通过；`TestEachConfiguredBucketLimitsItsOperations` 的七个子测试通过；六个生成物与上表相同；`make gen-check` 干净。

---

### Task 11: 停用账户

**Files:**
- Create: `server/internal/modules/identity/app/deactivate.go`、`deactivate_test.go`
- Modify: `server/internal/modules/identity/domain/session.go`、`app/ports.go`、`app/fakes_test.go`
- Modify: `server/internal/modules/identity/adapter/postgres/queries/users.sql`、`queries/profiles.sql`、`users.go`、`profiles.go`、`account_test.go`
- Modify: `api/modules/identity.yaml`；`api/openapi.yaml`（过渡：Task 12 加上时区）
- Modify: `server/internal/modules/identity/adapter/http/handler.go`、`handler_test.go`、`me.go`、`me_test.go`；`server/internal/modules/identity/module.go`
- Modify: `server/internal/bootstrap/account_test.go`
- Generate: `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`、`server/internal/modules/identity/adapter/http/gen/server.gen.go`、`server/internal/modules/identity/adapter/postgres/gen/users.sql.go`、`profiles.sql.go`

**Interfaces:**
- Consumes: `CredentialLock`（Task 6）；`RevokeSessions`（Task 10）。
- Produces（spec 2.15，M2 设计 3.5、4.2、4.3、6.4，决策点 3）：
  - `domain.RevokeDeactivated`；
  - 端口：`UserDeactivator.DeactivateUser`、`OnboardingResetter.ResetOnboarding`；
  - 查询：`DeactivateUser`（`is_active = false`，密码不变）；`ResetOnboarding`（`onboarding_step`、`is_onboarded`、`is_tour_completed`、`last_workspace_id` 回到列的默认值，即注册时的值；别的偏好不变）；
  - `app.NewDeactivate(DeactivateDeps{Lock, Users, Profiles, Sessions, Tx, Clock, Logger})`：一个事务，按全局加锁顺序：`Lock`（复核凭证）→ 账户 → 资料 → 全部会话 `deactivated`（`keep` 为零值）；PAT 不撤销，账户停用期间认证会拒绝它们；INFO "account deactivated"（`user_id`、`revoked_sessions`、`by: self`）；
  - 操作 `deactivateMe`（`POST /api/v0/me/deactivate`，没有请求体，204，`x-problem-codes: []`）。

**Tests:**
- `deactivate_test.go`：`TestDeactivate`（会话、PAT 各一次：锁、复核调用者自己的凭证、账户、资料、会话依次在一个事务里；三次写入都在用例的时刻；日志带撤销的个数和 `by`）；`TestDeactivateRechecksTheCredentialUnderTheLock`（什么都不写）；`TestDeactivateWithoutAnActor`。
- `account_test.go`（真实数据库）：`TestDeactivateUserAndResetOnboarding`（两个账户：停用的一个 `is_active = false`、密码不变、引导的四项回到默认、主题等不变；另一个账户完全不变）。
- `me_test.go`：`TestDeactivateMe`。
- `bootstrap/account_test.go`：`TestDeactivatingWithAPersonalAccessToken`（真实数据库：先完成一个引导步骤，再用 PAT 停用：204；会话 `deactivated`；PAT 读 `/me` 401；原密码登录 403 `identity.account_deactivated`；账户停用、引导重新开始、没有 PAT 被撤销）。A12 的端到端要用 `nerve users activate`，随管理命令在 P3b 加入。

- [ ] **Step 1: 领域、端口和查询**

`server/internal/modules/identity/domain/session.go`（对 Task 10 版本的差异）：

```diff
--- a/server/internal/modules/identity/domain/session.go
+++ b/server/internal/modules/identity/domain/session.go
@@ -17,9 +17,14 @@
 // statements.
 type RevokeReason string
 
-// RevokePasswordChanged revokes the other sessions of an account whose
-// password its owner changed.
-const RevokePasswordChanged RevokeReason = "password_changed"
+// The reasons of the use cases that revoke an account's sessions.
+const (
+	// RevokePasswordChanged revokes the other sessions of an account whose
+	// password its owner changed.
+	RevokePasswordChanged RevokeReason = "password_changed"
+	// RevokeDeactivated revokes every session of a deactivated account.
+	RevokeDeactivated RevokeReason = "deactivated"
+)
 
 // RefreshTokenPrefix starts every refresh token (M2 design 3.4).
 const RefreshTokenPrefix = "nrv_rt_"
```

`server/internal/modules/identity/app/ports.go`（对 Task 10 版本的差异）：

```diff
--- a/server/internal/modules/identity/app/ports.go
+++ b/server/internal/modules/identity/app/ports.go
@@ -106,6 +106,19 @@
 	LockForCredentials(ctx context.Context, id uuid.UUID) (LockedAccount, error)
 }
 
+// UserDeactivator deactivates accounts.
+type UserDeactivator interface {
+	// DeactivateUser sets account id inactive at now.
+	DeactivateUser(ctx context.Context, id uuid.UUID, now time.Time) error
+}
+
+// OnboardingResetter starts an account's onboarding over.
+type OnboardingResetter interface {
+	// ResetOnboarding puts userID's onboarding steps, is_onboarded,
+	// is_tour_completed and last workspace back to their defaults at now.
+	ResetOnboarding(ctx context.Context, userID uuid.UUID, now time.Time) error
+}
+
 // PasswordHashWriter stores a new hash of an account's password.
 type PasswordHashWriter interface {
 	UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string, now time.Time) error
```

`server/internal/modules/identity/adapter/postgres/queries/users.sql`（对 Task 10 版本的差异）：

```diff
--- a/server/internal/modules/identity/adapter/postgres/queries/users.sql
+++ b/server/internal/modules/identity/adapter/postgres/queries/users.sql
@@ -42,6 +42,11 @@
 WHERE id = sqlc.arg(id)
 RETURNING id, email, first_name, last_name, display_name, user_timezone, created_at;
 
+-- name: DeactivateUser :exec
+UPDATE users
+SET is_active = false, updated_at = sqlc.arg(now)
+WHERE id = sqlc.arg(id);
+
 -- name: UpdatePasswordHash :exec
 UPDATE users
 SET password = sqlc.arg(password), updated_at = sqlc.arg(now)
```

`server/internal/modules/identity/adapter/postgres/queries/profiles.sql`（对 Task 8 版本的差异）：

```diff
--- a/server/internal/modules/identity/adapter/postgres/queries/profiles.sql
+++ b/server/internal/modules/identity/adapter/postgres/queries/profiles.sql
@@ -23,3 +23,10 @@
     last_workspace_id = CASE WHEN sqlc.arg(set_last_workspace_id)::boolean THEN sqlc.narg(last_workspace_id)::uuid ELSE last_workspace_id END
 WHERE user_id = sqlc.arg(user_id)
 RETURNING theme, language, start_of_the_week, onboarding_step, is_onboarded, is_tour_completed, last_workspace_id, updated_at;
+
+-- name: ResetOnboarding :exec
+-- Deactivation: onboarding starts over, from the defaults of registration (M2 design 3.5, story A12).
+UPDATE profiles
+SET updated_at = sqlc.arg(now), onboarding_step = DEFAULT, is_onboarded = DEFAULT, is_tour_completed = DEFAULT,
+    last_workspace_id = DEFAULT
+WHERE user_id = sqlc.arg(user_id);
```

- [ ] **Step 2: 契约**

`api/openapi.yaml`（对 Task 10 版本的差异）：

```diff
--- a/api/openapi.yaml
+++ b/api/openapi.yaml
@@ -31,6 +31,8 @@
     $ref: 'modules/identity.yaml#/paths/~1api~1v0~1auth~1logout'
   /api/v0/me:
     $ref: 'modules/identity.yaml#/paths/~1api~1v0~1me'
+  /api/v0/me/deactivate:
+    $ref: 'modules/identity.yaml#/paths/~1api~1v0~1me~1deactivate'
   /api/v0/me/change-password:
     $ref: 'modules/identity.yaml#/paths/~1api~1v0~1me~1change-password'
   /api/v0/me/profile:
```

`api/modules/identity.yaml`（对 Task 10 版本的差异）：

```diff
--- a/api/modules/identity.yaml
+++ b/api/modules/identity.yaml
@@ -154,6 +154,24 @@
             application/json:
               schema:
                 $ref: '#/components/schemas/User'
+        default:
+          $ref: '#/components/responses/Problem'
+  /api/v0/me/deactivate:
+    post:
+      operationId: deactivateMe
+      tags: [identity]
+      summary: Deactivate the caller's account
+      description: >-
+        Any credential may, a personal access token too; no password is
+        asked for. Every session is signed out and onboarding starts over.
+        The password and the personal access tokens stay, but nothing
+        authenticates as the account until the server's administrator
+        activates it again.
+      security: [{bearer: []}]
+      x-problem-codes: []
+      responses:
+        '204':
+          description: Deactivated.
         default:
           $ref: '#/components/responses/Problem'
   /api/v0/me/change-password:
```

- [ ] **Step 3: 生成**

Run: `make gen`
Expected: 以下五个文件改变，其余生成物不变：

| 生成的文件 | SHA-256 | 行数 |
|---|---|---|
| `api/dist/openapi.yaml` | `4f735ffcfd7d3e071211d3754f6d57612a1917e1dfc04514acdc5ea0494075b9` | 810 |
| `server/internal/modules/identity/adapter/http/gen/server.gen.go` | `e4f5f7cb8187fde1be00f508d9d1490f5bf2edd0bf1bb3c27689efbab3a68cb2` | 1795 |
| `server/internal/modules/identity/adapter/postgres/gen/profiles.sql.go` | `17c3dda9dcfbee398698d553ed97e7a0f84ceee1d0d7cd2270193f698916c251` | 159 |
| `server/internal/modules/identity/adapter/postgres/gen/users.sql.go` | `e6971fa59826778eda92586523b24eb524ec7982c97675d2828f28a990d0007a` | 225 |
| `web/packages/api-client/src/schema.gen.ts` | `93186899e4ac3db3b8c9157f3567f7f6c00c6a126e7a35841a219a12b497a425` | 814 |

- [ ] **Step 4: 存储**

`server/internal/modules/identity/adapter/postgres/users.go`（对 Task 10 版本的差异）：

```diff
--- a/server/internal/modules/identity/adapter/postgres/users.go
+++ b/server/internal/modules/identity/adapter/postgres/users.go
@@ -107,6 +107,14 @@
 	return app.LockedAccount{PasswordHash: row.Password, Active: row.IsActive}, nil
 }
 
+// DeactivateUser sets account id inactive at now.
+func (s *Store) DeactivateUser(ctx context.Context, id uuid.UUID, now time.Time) error {
+	if err := s.queries(ctx).DeactivateUser(ctx, gen.DeactivateUserParams{Now: now, ID: id}); err != nil {
+		return fmt.Errorf("deactivate user: %w", err)
+	}
+	return nil
+}
+
 // UpdatePasswordHash stores hash as account id's password.
 func (s *Store) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string, now time.Time) error {
 	if err := s.queries(ctx).UpdatePasswordHash(ctx, gen.UpdatePasswordHashParams{Password: hash, Now: now, ID: id}); err != nil {
```

`server/internal/modules/identity/adapter/postgres/profiles.go`（对 Task 8 版本的差异）：

```diff
--- a/server/internal/modules/identity/adapter/postgres/profiles.go
+++ b/server/internal/modules/identity/adapter/postgres/profiles.go
@@ -41,6 +41,15 @@
 	return profileOf(gen.GetProfileRow(row))
 }
 
+// ResetOnboarding puts userID's onboarding back to the defaults it had at
+// registration, at now.
+func (s *Store) ResetOnboarding(ctx context.Context, userID uuid.UUID, now time.Time) error {
+	if err := s.queries(ctx).ResetOnboarding(ctx, gen.ResetOnboardingParams{Now: now, UserID: userID}); err != nil {
+		return fmt.Errorf("reset onboarding: %w", err)
+	}
+	return nil
+}
+
 // profileOf reads a profile row. The steps are the object that the
 // column's CHECK guarantees: four booleans.
 func profileOf(row gen.GetProfileRow) (domain.Profile, error) {
```

`server/internal/modules/identity/adapter/postgres/account_test.go`（对 Task 8 版本的差异）：

```diff
--- a/server/internal/modules/identity/adapter/postgres/account_test.go
+++ b/server/internal/modules/identity/adapter/postgres/account_test.go
@@ -129,6 +129,60 @@
 	}
 }
 
+// Deactivation sets the account inactive and starts its onboarding over,
+// from the defaults of registration; the password, the other preferences
+// and other accounts stay (M2 design 3.5, story A12).
+func TestDeactivateUserAndResetOnboarding(t *testing.T) {
+	s, pool := newStore(t)
+	alice, bob := accountWithProfile(t, s), newUser("bob@corp.com")
+	mustCreate(t, s, bob)
+	if err := s.CreateDefaultProfile(context.Background(), uuid.NewV7(), bob.ID, now); err != nil {
+		t.Fatal(err)
+	}
+	workspace := uuid.NewV7()
+	onboarded := domain.ProfilePatch{
+		Theme: ptr("dark"), OnboardingStep: domain.OnboardingStepsPatch{ProfileComplete: ptr(true), WorkspaceJoin: ptr(true)},
+		IsOnboarded: ptr(true), IsTourCompleted: ptr(true), LastWorkspaceSet: true, LastWorkspaceID: &workspace,
+	}
+	var bobs domain.Profile
+	for _, u := range []app.NewUser{alice, bob} {
+		p, err := s.UpdateProfile(context.Background(), u.ID, onboarded, now)
+		if err != nil {
+			t.Fatal(err)
+		}
+		bobs = p
+	}
+	deactivated := now.Add(time.Hour)
+
+	if err := errors.Join(s.DeactivateUser(context.Background(), alice.ID, deactivated),
+		s.ResetOnboarding(context.Background(), alice.ID, deactivated)); err != nil {
+		t.Fatal(err)
+	}
+
+	active := func(u app.NewUser) (bool, string) {
+		var active bool
+		var password string
+		if err := pool.QueryRow(context.Background(), "SELECT is_active, password FROM users WHERE id = $1", u.ID).Scan(&active, &password); err != nil {
+			t.Fatal(err)
+		}
+		return active, password
+	}
+	if a, password := active(alice); a || password != alice.PasswordHash {
+		t.Errorf("alice: active %v, password %q; want inactive with the password kept", a, password)
+	}
+	if b, _ := active(bob); !b {
+		t.Error("bob is inactive, want him untouched")
+	}
+	assertUpdatedAt(t, pool, "users", "id", alice.ID, deactivated)
+	want := domain.Profile{Theme: "dark", Language: "en", UpdatedAt: deactivated}
+	if got, err := s.GetProfile(context.Background(), alice.ID); err != nil || !sameProfile(got, want) {
+		t.Errorf("alice's profile = %+v, %v; want %+v", got, err, want)
+	}
+	if got, err := s.GetProfile(context.Background(), bob.ID); err != nil || !sameProfile(got, bobs) {
+		t.Errorf("bob's profile = %+v, %v; want it untouched, %+v", got, err, bobs)
+	}
+}
+
 func sameProfile(a, b domain.Profile) bool {
 	sameWorkspace := (a.LastWorkspaceID == nil) == (b.LastWorkspaceID == nil) &&
 		(a.LastWorkspaceID == nil || *a.LastWorkspaceID == *b.LastWorkspaceID)
```

- [ ] **Step 5: 用例**

`server/internal/modules/identity/app/deactivate.go`（新文件）：

```go
package app

import (
	"context"
	"log/slog"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DeactivateDeps are Deactivate's collaborators.
type DeactivateDeps struct {
	Lock     CredentialLock
	Users    UserDeactivator
	Profiles OnboardingResetter
	Sessions SessionRevoker
	Tx       shared.TxManager
	Clock    Clock
	Logger   *slog.Logger
}

// Deactivate deactivates the caller's account: POST /api/v0/me/deactivate.
// Any credential may, a token too, and no password is asked for, as in
// Plane (M2 decision 3, design 8.6).
type Deactivate struct {
	d DeactivateDeps
}

// NewDeactivate returns the use case.
func NewDeactivate(d DeactivateDeps) *Deactivate {
	return &Deactivate{d: d}
}

// Execute takes the credential lock, then writes in the global lock order
// (M2 design 3.5, 6.4): the account inactive, its onboarding started over,
// every session revoked with reason deactivated. The password and the
// personal access tokens stay; authentication refuses the tokens while the
// account is inactive.
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
		if err := u.d.Users.DeactivateUser(ctx, actor.UserID, now); err != nil {
			return err
		}
		if err := u.d.Profiles.ResetOnboarding(ctx, actor.UserID, now); err != nil {
			return err
		}
		var err error
		revoked, err = u.d.Sessions.RevokeSessions(ctx, actor.UserID, uuid.Nil(), domain.RevokeDeactivated, now)
		return err
	})
	if err != nil {
		return err
	}
	u.d.Logger.InfoContext(ctx, "account deactivated", slog.String("user_id", actor.UserID.String()),
		slog.Int("revoked_sessions", revoked), slog.String("by", "self"))
	return nil
}
```

`server/internal/modules/identity/app/fakes_test.go`（对 Task 10 版本的差异）：

```diff
--- a/server/internal/modules/identity/app/fakes_test.go
+++ b/server/internal/modules/identity/app/fakes_test.go
@@ -293,8 +293,8 @@
 }
 
 // fakeCredentials is the account row and the session that the credential
-// lock reads. A change of password reads the row and writes it and the
-// sessions: it logs them too.
+// lock reads. A change of password and a deactivation read and write the
+// account's rows: it logs them too.
 type fakeCredentials struct {
 	log        *callLog
 	email      string
@@ -314,9 +314,21 @@
 	f.log.add(ctx, "password "+id.String()+" "+hash)
 	f.writtenAt = append(f.writtenAt, now)
 	f.account.PasswordHash = hash
+	return nil
+}
+
+func (f *fakeCredentials) DeactivateUser(ctx context.Context, id uuid.UUID, now time.Time) error {
+	f.log.add(ctx, "deactivate "+id.String())
+	f.writtenAt = append(f.writtenAt, now)
 	return nil
 }
 
+func (f *fakeCredentials) ResetOnboarding(ctx context.Context, userID uuid.UUID, now time.Time) error {
+	f.log.add(ctx, "reset onboarding of "+userID.String())
+	f.writtenAt = append(f.writtenAt, now)
+	return nil
+}
+
 func (f *fakeCredentials) RevokeSessions(ctx context.Context, userID, keep uuid.UUID, reason domain.RevokeReason, now time.Time) (int, error) {
 	f.log.add(ctx, "revoke "+string(reason)+" sessions of "+userID.String()+" but "+keep.String())
 	f.writtenAt = append(f.writtenAt, now)
```

`server/internal/modules/identity/app/deactivate_test.go`（新文件）：

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
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func (f *credentialFixture) deactivate() *app.Deactivate {
	return app.NewDeactivate(app.DeactivateDeps{
		Lock: f.lock(), Users: f.creds, Profiles: f.creds, Sessions: f.creds, Tx: f.tx, Clock: clocktest.At(now), Logger: f.logger(),
	})
}

// One transaction takes the lock and checks the caller's credential again,
// then writes in the global lock order: the account, the profile, the
// sessions, all of them, whatever the credential (M2 design 3.5, 6.4).
func TestDeactivate(t *testing.T) {
	tests := []struct {
		name       string
		actor      shared.Actor
		credential string
	}{
		{"with a session", sessionActor, "session " + sessionID.String()},
		{"with a token", tokenActor, "token " + tokenID.String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newCredentialFixture()

			err := f.deactivate().Execute(shared.WithActor(context.Background(), tt.actor))

			want := []string{
				"lock " + userID.String(), tt.credential, "deactivate " + userID.String(), "reset onboarding of " + userID.String(),
				"revoke deactivated sessions of " + userID.String() + " but " + uuid.Nil().String(),
			}
			if err != nil || !slices.Equal(f.log.calls, want) || f.tx.calls != 1 || !slices.Equal(f.creds.writtenAt, []time.Time{now, now, now}) {
				t.Errorf("Execute() = %v, calls %q in %d transactions at %v; want %q in one at %v", err, f.log.calls, f.tx.calls, f.creds.writtenAt, want, now)
			}
			logs := f.logs.String()
			if !strings.Contains(logs, `"msg":"account deactivated","user_id":"`+userID.String()+`","revoked_sessions":1,"by":"self"`) {
				t.Errorf("logs = %s, want the deactivation with user_id, the sessions revoked and by self", logs)
			}
		})
	}
}

// A credential that a concurrent reset revoked since authentication
// deactivates nothing (M2 design 3.5).
func TestDeactivateRechecksTheCredentialUnderTheLock(t *testing.T) {
	f := newCredentialFixture()
	f.tokens.credential.Revoked = true

	err := f.deactivate().Execute(shared.WithActor(context.Background(), tokenActor))

	if !errors.Is(err, shared.Unauthenticated()) || len(f.creds.writtenAt) != 0 {
		t.Errorf("Execute() = %v, %d writes; want 401 and none", err, len(f.creds.writtenAt))
	}
}

func TestDeactivateWithoutAnActor(t *testing.T) {
	f := newCredentialFixture()

	if err := f.deactivate().Execute(context.Background()); !errors.Is(err, shared.Unauthenticated()) || len(f.log.calls) != 0 {
		t.Errorf("Execute() = %v after calls %q, want 401 before any", err, f.log.calls)
	}
}
```

- [ ] **Step 6: HTTP 适配器和接线**

`server/internal/modules/identity/adapter/http/handler.go`（对 Task 10 版本的差异）：

```diff
--- a/server/internal/modules/identity/adapter/http/handler.go
+++ b/server/internal/modules/identity/adapter/http/handler.go
@@ -52,6 +52,11 @@
 	Execute(ctx context.Context, in app.ChangePasswordInput) error
 }
 
+// DeactivateUseCase is app.Deactivate.
+type DeactivateUseCase interface {
+	Execute(ctx context.Context) error
+}
+
 // GetProfileUseCase is app.GetProfile.
 type GetProfileUseCase interface {
 	Execute(ctx context.Context) (domain.Profile, error)
@@ -86,6 +91,7 @@
 	GetMe          GetMeUseCase
 	UpdateMe       UpdateMeUseCase
 	ChangePassword ChangePasswordUseCase
+	Deactivate     DeactivateUseCase
 	GetProfile     GetProfileUseCase
 	UpdateProfile  UpdateProfileUseCase
 	ListAPITokens  ListAPITokensUseCase
```

`server/internal/modules/identity/adapter/http/me.go`（对 Task 10 版本的差异）：

```diff
--- a/server/internal/modules/identity/adapter/http/me.go
+++ b/server/internal/modules/identity/adapter/http/me.go
@@ -46,6 +46,14 @@
 	return gen.ChangePassword204Response{}, nil
 }
 
+// DeactivateMe serves POST /api/v0/me/deactivate.
+func (h handler) DeactivateMe(ctx context.Context, _ gen.DeactivateMeRequestObject) (gen.DeactivateMeResponseObject, error) {
+	if err := h.uc.Deactivate.Execute(ctx); err != nil {
+		return nil, err
+	}
+	return gen.DeactivateMe204Response{}, nil
+}
+
 // user is u as the API shows it.
 func user(u domain.User) gen.User {
 	return gen.User{
```

`server/internal/modules/identity/adapter/http/handler_test.go`（对 Task 10 版本的差异）：

```diff
--- a/server/internal/modules/identity/adapter/http/handler_test.go
+++ b/server/internal/modules/identity/adapter/http/handler_test.go
@@ -77,6 +77,7 @@
 	logout        *fakeLogout
 	updateMe      *fakeUpdateMe
 	change        *fakeChangePassword
+	deactivate    *fakeDeactivate
 	getProfile    *fakeGetProfile
 	updateProfile *fakeUpdateProfile
 	listTokens    *fakeListTokens
@@ -141,6 +142,9 @@
 	if f.change == nil {
 		f.change = &fakeChangePassword{}
 	}
+	if f.deactivate == nil {
+		f.deactivate = &fakeDeactivate{}
+	}
 	if f.getProfile == nil {
 		f.getProfile = &fakeGetProfile{}
 	}
@@ -158,7 +162,8 @@
 	}
 	httpadapter.Register(router, api, httpadapter.UseCases{
 		Register: f.register, Login: f.login, Refresh: f.refresh, Logout: f.logout,
-		GetMe: fakeGetMe{}, UpdateMe: f.updateMe, ChangePassword: f.change, GetProfile: f.getProfile, UpdateProfile: f.updateProfile,
+		GetMe: fakeGetMe{}, UpdateMe: f.updateMe, ChangePassword: f.change, Deactivate: f.deactivate,
+		GetProfile: f.getProfile, UpdateProfile: f.updateProfile,
 		ListAPITokens: f.listTokens, CreateAPIToken: f.createToken, RevokeAPIToken: f.revokeToken,
 	}, s)
 	return router
```

`server/internal/modules/identity/adapter/http/me_test.go`（对 Task 10 版本的差异）：

```diff
--- a/server/internal/modules/identity/adapter/http/me_test.go
+++ b/server/internal/modules/identity/adapter/http/me_test.go
@@ -40,6 +40,13 @@
 	return f.err
 }
 
+type fakeDeactivate struct{ calls int }
+
+func (f *fakeDeactivate) Execute(context.Context) error {
+	f.calls++
+	return nil
+}
+
 // patchJSON is a PATCH with the bearer token fakeAuth accepts.
 func patchJSON(path, body string) *http.Request {
 	req := withToken(httptest.NewRequest(http.MethodPatch, path, strings.NewReader(body)))
@@ -117,6 +124,18 @@
 	}
 }
 
+func TestDeactivateMe(t *testing.T) {
+	deactivate := &fakeDeactivate{}
+	req := withToken(httptest.NewRequest(http.MethodPost, "/api/v0/me/deactivate", nil))
+	apitest.Load(t).CheckRequest(t, req)
+
+	res, body := do(t, newServer(t, fakes{deactivate: deactivate}), req)
+
+	if res.StatusCode != http.StatusNoContent || body != "" || deactivate.calls != 1 {
+		t.Errorf("POST /me/deactivate = %d %q after %d calls, want 204 after one", res.StatusCode, body, deactivate.calls)
+	}
+}
+
 // The handler exit: every error the use case returns becomes its problem.
 func TestChangePasswordProblems(t *testing.T) {
 	tests := []struct {
```

`server/internal/modules/identity/module.go`（对 Task 10 版本的差异）：

```diff
--- a/server/internal/modules/identity/module.go
+++ b/server/internal/modules/identity/module.go
@@ -112,6 +112,9 @@
 				Accounts: store, Lock: lock, Passwords: store, Sessions: store, Hasher: hasher,
 				Rules: rules, Tx: d.Tx, Clock: d.Clock, Logger: d.Logger,
 			}),
+			Deactivate: app.NewDeactivate(app.DeactivateDeps{
+				Lock: lock, Users: store, Profiles: store, Sessions: store, Tx: d.Tx, Clock: d.Clock, Logger: d.Logger,
+			}),
 			GetProfile:    app.NewGetProfile(store),
 			UpdateProfile: app.NewUpdateProfile(store, d.Clock),
 			ListAPITokens: app.NewListAPITokens(store),
```

- [ ] **Step 7: 整程序测试**

`server/internal/bootstrap/account_test.go`（对 Task 10 版本的差异）：

```diff
--- a/server/internal/bootstrap/account_test.go
+++ b/server/internal/bootstrap/account_test.go
@@ -1,6 +1,7 @@
 package bootstrap
 
 import (
+	"context"
 	"net/http"
 	"strings"
 	"testing"
@@ -61,6 +62,40 @@
 	}
 }
 
+// Deactivating with a personal access token: the account is inactive with
+// its password, every session is revoked, onboarding starts over, and the
+// token stays but authenticates no more; the right password answers
+// identity.account_deactivated (M2 design 3.5, story A12).
+func TestDeactivatingWithAPersonalAccessToken(t *testing.T) {
+	base, pool := sessionApp(t)
+	contract := apitest.Load(t)
+	token := createPAT(t, contract, base, registerAccount(t, contract, base, "leaving@example.com").AccessToken).Token
+	if status, body := call(t, contract, http.MethodPatch, base+"/api/v0/me/profile", token,
+		`{"onboarding_step":{"profile_complete":true},"is_onboarded":true}`); status != http.StatusOK {
+		t.Fatalf("PATCH /me/profile = %d %s", status, body)
+	}
+
+	deactivated, body := call(t, contract, http.MethodPost, base+"/api/v0/me/deactivate", token, "")
+	reason := revocation(t, pool, "leaving@example.com")
+	me, _ := call(t, contract, http.MethodGet, base+"/api/v0/me", token, "")
+	signIn, _ := login(t, base, "leaving@example.com", "Tr0ub4dor&3")
+
+	if deactivated != http.StatusNoContent || reason != "deactivated" || me != http.StatusUnauthorized || signIn != http.StatusForbidden {
+		t.Errorf("deactivate = %d %s; the session revoked for %q; the token's GET /me %d; login %d; want 204, deactivated, 401, 403",
+			deactivated, body, reason, me, signIn)
+	}
+	var active, startsOver bool
+	var revokedTokens int
+	err := pool.QueryRow(context.Background(), `SELECT u.is_active,
+		p.onboarding_step = '{"profile_complete": false, "workspace_create": false, "workspace_invite": false, "workspace_join": false}'
+			AND NOT p.is_onboarded,
+		(SELECT count(*) FROM api_tokens t WHERE t.user_id = u.id AND t.deleted_at IS NOT NULL)
+		FROM users u JOIN profiles p ON p.user_id = u.id WHERE u.email = 'leaving@example.com'`).Scan(&active, &startsOver, &revokedTokens)
+	if err != nil || active || !startsOver || revokedTokens != 0 {
+		t.Errorf("active %v, onboarding started over %v, tokens revoked %d (%v); want false, true, 0", active, startsOver, revokedTokens, err)
+	}
+}
+
 func TestTheAccountAndItsPreferencesWithAPersonalAccessToken(t *testing.T) {
 	contract, base, token := accountApp(t, "account@example.com")
 
```

- [ ] **Step 8: lint、测试和端到端**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。

Run: `make lint-web`
Expected: 通过。

Run: `make e2e`
Expected: P1、P2 的故事全部通过。

- [ ] **Step 9: 提交**

```bash
git add api web/packages/api-client/src/schema.gen.ts server/internal/modules/identity server/internal/bootstrap/account_test.go
```
```bash
git commit -m "feat(M2/P3a): deactivate the account

Under the credential lock, in the global lock order: the account becomes
inactive, onboarding starts over and every session is revoked. The
password and the tokens stay; authentication refuses the tokens while the
account is inactive.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run: `make gen-check`
Expected: 没有差异。

**Done when:** 停用的用例、存储、适配器和整程序测试通过；五个生成物与上表相同；`make gen-check` 干净。

---

### Task 12: 实例配置的三个字段；时区列表；嵌入时区数据库

**Files:**
- Modify: `api/modules/instance.yaml`、`api/openapi.yaml`
- Modify: `server/internal/modules/instance/domain/info.go`；Create: `domain/timezones.go`、`timezones_test.go`
- Modify: `server/internal/modules/instance/app/ports.go`、`get_info.go`、`get_info_test.go`；Create: `app/list_timezones.go`、`list_timezones_test.go`
- Modify: `server/internal/modules/instance/adapter/http/handler.go`、`handler_test.go`；`server/internal/modules/instance/module.go`
- Modify: `server/cmd/nerve/main.go`、`server/internal/archtest/binary_test.go`
- Modify: `server/internal/bootstrap/app.go`、`app_test.go`、`api_test.go`
- Modify: `server/internal/platform/httpserver/apitest/apitest_test.go`、`e2e/stories/smoke/s3-instance-info.spec.ts`
- Generate: `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`、`server/internal/modules/instance/adapter/http/gen/server.gen.go`、`bodyshape.gen.go`

**Interfaces:**
- Consumes: `workspace.creation_enabled`、`files.size_limit`（Task 1）。
- Produces（spec 2.16，M2 设计 3.19、4.2、5.3）：
  - `InstanceInfo` 加上必有的 `signup_enabled`、`workspace_creation_enabled`（布尔）、`file_size_limit`（整数，至少 1）；`is_self_managed` 不定义；
  - 操作 `listTimezones`（`GET /api/v0/timezones`，公开，`x-problem-codes: []`），`TimezoneList{data: Timezone[]}`，`Timezone{label, value, utc_offset, gmt_offset}`；
  - `domain.Settings{SignupEnabled, WorkspaceCreationEnabled, FileSizeLimit}`（嵌在 `Info` 里）；`domain.Timezone{Label, Name, Offset}`；`domain.Timezones(t)`：Plane 的 120 个地点（`base.py:30-178`），偏移量按 `t` 时的时区数据计算，按偏移、再按标签排序（`base.py:209`）；偏移写成 `±hh:mm`，负的非整点偏移写对（Plane 在 `base.py:192` 把 -09:30 写成 -10:30，spec 第 3 节第 11 条）；
  - `app.Clock`；`app.NewGetInfo(source, settings)`；`app.NewListTimezones(clock)`；
  - `instance.New(instance.Deps{SignupEnabled, WorkspaceCreationEnabled, FileSizeLimit, Clock})`；
  - `cmd/nerve` 导入 `time/tzdata`：接受哪些 `user_timezone`、时区列表的偏移不取决于宿主（容器可能没有时区文件，M2 设计 4.2）。

**Tests:**
- `timezones_test.go`：`TestTimezonesAreSorted`（120 个都能加载；按偏移、再按标签；第一个是 American Samoa -11:00，最后一个是 Kiritimati +14:00）；`TestTimezoneOffsets`（9 个地点在一月和七月的偏移：Marquesas -09:30、Newfoundland、太平洋时间、Reykjavik、Dublin、Kolkata、Kathmandu +05:45、北京、Chatham +13:45 / +12:45）。
- `get_info_test.go`：`TestGetInfoDescribesTheBuildAndTheSettings`（替代原来只核对构建信息的测试）；`list_timezones_test.go`：`TestListTimezonesAtTheClocksNow`（Dublin 在七月是 +01:00：偏移按用例的时钟算）。
- `handler_test.go`：`TestGetInstanceMatchesTheContract`（三个新字段）；`TestListTimezones`（`UTC`、`GMT` 前缀；`CheckResponse`）。
- `bootstrap/api_test.go`：`TestServesTheInstanceAPI` 用不同于默认的设置（`true`、`false`、7340032）核对三个字段来自配置；`TestServesTheTimezones`。
- `archtest/binary_test.go`：`TestNerveBinaryEmbedsTheTimeZoneDatabase`。
- `apitest_test.go` 的实例响应、`s3-instance-info.spec.ts` 的 `toEqual` 加上三个字段。

- [ ] **Step 1: 领域和用例**

`server/internal/modules/instance/domain/info.go`（完整内容）：

```go
// Package domain holds the instance module's model.
package domain

// Product is the product name every instance reports.
const Product = "Nerve"

// APIVersion is the version of the HTTP API this build serves. It follows
// the product's major version (v0 design 3.1) and prefixes every API path.
const APIVersion = "v0"

// Build identifies the binary an instance runs.
type Build struct {
	Version string // product version, e.g. "0.1.0-dev"
	Commit  string // git revision, "unknown" without a VCS stamp
}

// Settings are the configuration that an instance reports to clients, so
// that they adapt to it (M2 design 5.3). The modules that act on them
// enforce them.
type Settings struct {
	SignupEnabled            bool  // auth.signup_enabled
	WorkspaceCreationEnabled bool  // workspace.creation_enabled
	FileSizeLimit            int64 // files.size_limit, in bytes
}

// Info is what an instance tells API clients about itself.
type Info struct {
	Product    string
	Version    string
	Commit     string
	APIVersion string
	Settings
}
```

`server/internal/modules/instance/domain/timezones.go`（新文件）：

```go
package domain

import (
	"cmp"
	"fmt"
	"slices"
	"time"
)

// Timezone is one of the time zones the web app offers (M2 design 5.3).
type Timezone struct {
	Label  string // a place in the zone, e.g. "Beijing"
	Name   string // its IANA name, e.g. "Asia/Shanghai"
	Offset string // from UTC at the time asked, e.g. "+08:00" or "-09:30"
}

// timezoneLocations are Plane's places and their zones
// (plane/apps/api/plane/app/views/timezone/base.py:30-178), in its order.
var timezoneLocations = [][2]string{
	{"Midway Island", "Pacific/Midway"},
	{"American Samoa", "Pacific/Pago_Pago"},
	{"Hawaii", "Pacific/Honolulu"},
	{"Aleutian Islands", "America/Adak"},
	{"Marquesas Islands", "Pacific/Marquesas"},
	{"Alaska", "America/Anchorage"},
	{"Gambier Islands", "Pacific/Gambier"},
	{"Pacific Time (US and Canada)", "America/Los_Angeles"},
	{"Baja California", "America/Tijuana"},
	{"Mountain Time (US and Canada)", "America/Denver"},
	{"Arizona", "America/Phoenix"},
	{"Chihuahua, Mazatlan", "America/Chihuahua"},
	{"Central Time (US and Canada)", "America/Chicago"},
	{"Saskatchewan", "America/Regina"},
	{"Guadalajara, Mexico City, Monterrey", "America/Mexico_City"},
	{"Tegucigalpa, Honduras", "America/Tegucigalpa"},
	{"Costa Rica", "America/Costa_Rica"},
	{"Eastern Time (US and Canada)", "America/New_York"},
	{"Lima", "America/Lima"},
	{"Bogota", "America/Bogota"},
	{"Quito", "America/Guayaquil"},
	{"Chetumal", "America/Cancun"},
	{"Caracas (Old Venezuela Time)", "America/Caracas"},
	{"Atlantic Time (Canada)", "America/Halifax"},
	{"Caracas", "America/Caracas"},
	{"Santiago", "America/Santiago"},
	{"La Paz", "America/La_Paz"},
	{"Manaus", "America/Manaus"},
	{"Georgetown", "America/Guyana"},
	{"Bermuda", "Atlantic/Bermuda"},
	{"Newfoundland Time (Canada)", "America/St_Johns"},
	{"Buenos Aires", "America/Argentina/Buenos_Aires"},
	{"Brasilia", "America/Sao_Paulo"},
	{"Greenland", "America/Godthab"},
	{"Montevideo", "America/Montevideo"},
	{"Falkland Islands", "Atlantic/Stanley"},
	{"South Georgia and the South Sandwich Islands", "Atlantic/South_Georgia"},
	{"Azores", "Atlantic/Azores"},
	{"Cape Verde Islands", "Atlantic/Cape_Verde"},
	{"Dublin", "Europe/Dublin"},
	{"Reykjavik", "Atlantic/Reykjavik"},
	{"Lisbon", "Europe/Lisbon"},
	{"Monrovia", "Africa/Monrovia"},
	{"Casablanca", "Africa/Casablanca"},
	{"Central European Time (Berlin, Rome, Paris)", "Europe/Paris"},
	{"West Central Africa", "Africa/Lagos"},
	{"Algiers", "Africa/Algiers"},
	{"Lagos", "Africa/Lagos"},
	{"Tunis", "Africa/Tunis"},
	{"Eastern European Time (Cairo, Helsinki, Kyiv)", "Europe/Kyiv"},
	{"Athens", "Europe/Athens"},
	{"Jerusalem", "Asia/Jerusalem"},
	{"Johannesburg", "Africa/Johannesburg"},
	{"Harare, Pretoria", "Africa/Harare"},
	{"Moscow Time", "Europe/Moscow"},
	{"Baghdad", "Asia/Baghdad"},
	{"Nairobi", "Africa/Nairobi"},
	{"Kuwait, Riyadh", "Asia/Riyadh"},
	{"Tehran", "Asia/Tehran"},
	{"Abu Dhabi", "Asia/Dubai"},
	{"Baku", "Asia/Baku"},
	{"Yerevan", "Asia/Yerevan"},
	{"Astrakhan", "Europe/Astrakhan"},
	{"Tbilisi", "Asia/Tbilisi"},
	{"Mauritius", "Indian/Mauritius"},
	{"Kabul", "Asia/Kabul"},
	{"Islamabad", "Asia/Karachi"},
	{"Karachi", "Asia/Karachi"},
	{"Tashkent", "Asia/Tashkent"},
	{"Yekaterinburg", "Asia/Yekaterinburg"},
	{"Maldives", "Indian/Maldives"},
	{"Chagos", "Indian/Chagos"},
	{"Chennai", "Asia/Kolkata"},
	{"Kolkata", "Asia/Kolkata"},
	{"Mumbai", "Asia/Kolkata"},
	{"New Delhi", "Asia/Kolkata"},
	{"Sri Jayawardenepura", "Asia/Colombo"},
	{"Kathmandu", "Asia/Kathmandu"},
	{"Dhaka", "Asia/Dhaka"},
	{"Almaty", "Asia/Almaty"},
	{"Bishkek", "Asia/Bishkek"},
	{"Thimphu", "Asia/Thimphu"},
	{"Yangon (Rangoon)", "Asia/Yangon"},
	{"Cocos Islands", "Indian/Cocos"},
	{"Bangkok", "Asia/Bangkok"},
	{"Hanoi", "Asia/Ho_Chi_Minh"},
	{"Jakarta", "Asia/Jakarta"},
	{"Novosibirsk", "Asia/Novosibirsk"},
	{"Krasnoyarsk", "Asia/Krasnoyarsk"},
	{"Beijing", "Asia/Shanghai"},
	{"Singapore", "Asia/Singapore"},
	{"Perth", "Australia/Perth"},
	{"Hong Kong", "Asia/Hong_Kong"},
	{"Ulaanbaatar", "Asia/Ulaanbaatar"},
	{"Palau", "Pacific/Palau"},
	{"Eucla", "Australia/Eucla"},
	{"Tokyo", "Asia/Tokyo"},
	{"Seoul", "Asia/Seoul"},
	{"Yakutsk", "Asia/Yakutsk"},
	{"Adelaide", "Australia/Adelaide"},
	{"Darwin", "Australia/Darwin"},
	{"Sydney", "Australia/Sydney"},
	{"Brisbane", "Australia/Brisbane"},
	{"Guam", "Pacific/Guam"},
	{"Vladivostok", "Asia/Vladivostok"},
	{"Tahiti", "Pacific/Tahiti"},
	{"Lord Howe Island", "Australia/Lord_Howe"},
	{"Solomon Islands", "Pacific/Guadalcanal"},
	{"Magadan", "Asia/Magadan"},
	{"Norfolk Island", "Pacific/Norfolk"},
	{"Bougainville Island", "Pacific/Bougainville"},
	{"Chokurdakh", "Asia/Srednekolymsk"},
	{"Auckland", "Pacific/Auckland"},
	{"Wellington", "Pacific/Auckland"},
	{"Fiji Islands", "Pacific/Fiji"},
	{"Anadyr", "Asia/Anadyr"},
	{"Chatham Islands", "Pacific/Chatham"},
	{"Nuku'alofa", "Pacific/Tongatapu"},
	{"Samoa", "Pacific/Apia"},
	{"Kiritimati Island", "Pacific/Kiritimati"},
}

// Timezones returns the time zones with their offsets at t, sorted by
// offset and then by label, as Plane sorts them (base.py:209). Plane writes
// a negative offset that is not a whole hour an hour too far west, -09:30
// as -10:30 (base.py:192); here it is -09:30.
func Timezones(t time.Time) ([]Timezone, error) {
	type zone struct {
		Timezone
		seconds int
	}
	zones := make([]zone, 0, len(timezoneLocations))
	for _, l := range timezoneLocations {
		loc, err := time.LoadLocation(l[1])
		if err != nil {
			return nil, fmt.Errorf("time zone %s: %w", l[1], err)
		}
		_, seconds := t.In(loc).Zone()
		zones = append(zones, zone{Timezone{Label: l[0], Name: l[1], Offset: offset(seconds)}, seconds})
	}
	slices.SortFunc(zones, func(a, b zone) int {
		return cmp.Or(cmp.Compare(a.seconds, b.seconds), cmp.Compare(a.Label, b.Label))
	})
	out := make([]Timezone, len(zones))
	for i, z := range zones {
		out[i] = z.Timezone
	}
	return out, nil
}

// offset writes seconds east of UTC as ±hh:mm.
func offset(seconds int) string {
	sign := '+'
	if seconds < 0 {
		sign, seconds = '-', -seconds
	}
	return fmt.Sprintf("%c%02d:%02d", sign, seconds/3600, seconds%3600/60)
}
```

`server/internal/modules/instance/domain/timezones_test.go`（新文件）：

```go
package domain_test

import (
	"cmp"
	"testing"
	"time"
	_ "time/tzdata" // as in the binary: a zone the host lacks loads from the embedded database

	"github.com/open-nerve/NerveProject/server/internal/modules/instance/domain"
)

// minutes reads an offset ±hh:mm.
func minutes(t *testing.T, offset string) int {
	t.Helper()
	at, err := time.Parse("-07:00", offset)
	if err != nil {
		t.Fatalf("offset %q is not ±hh:mm: %v", offset, err)
	}
	_, seconds := at.Zone()
	return seconds / 60
}

// Every zone loads; the list is sorted by offset, then by label.
func TestTimezonesAreSorted(t *testing.T) {
	got, err := domain.Timezones(time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC))

	if err != nil || len(got) != 120 {
		t.Fatalf("Timezones() = %d zones, %v; want Plane's 120", len(got), err)
	}
	for i := 1; i < len(got); i++ {
		a, b := got[i-1], got[i]
		if cmp.Or(cmp.Compare(minutes(t, a.Offset), minutes(t, b.Offset)), cmp.Compare(a.Label, b.Label)) >= 0 {
			t.Errorf("%+v comes before %+v", a, b)
		}
	}
	first, last := domain.Timezone{Label: "American Samoa", Name: "Pacific/Pago_Pago", Offset: "-11:00"},
		domain.Timezone{Label: "Kiritimati Island", Name: "Pacific/Kiritimati", Offset: "+14:00"}
	if got[0] != first || got[len(got)-1] != last {
		t.Errorf("first %+v, last %+v; want %+v, %+v", got[0], got[len(got)-1], first, last)
	}
}

// The offsets are the zones' at the time asked, daylight saving time
// included; a negative offset that is not a whole hour keeps its hour.
func TestTimezoneOffsets(t *testing.T) {
	january, july := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC), time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		label, january, july string
	}{
		{"Marquesas Islands", "-09:30", "-09:30"},
		{"Newfoundland Time (Canada)", "-03:30", "-02:30"},
		{"Pacific Time (US and Canada)", "-08:00", "-07:00"},
		{"Reykjavik", "+00:00", "+00:00"},
		{"Dublin", "+00:00", "+01:00"},
		{"Kolkata", "+05:30", "+05:30"},
		{"Kathmandu", "+05:45", "+05:45"},
		{"Beijing", "+08:00", "+08:00"},
		{"Chatham Islands", "+13:45", "+12:45"},
	}
	offsets := func(at time.Time) map[string]string {
		zones, err := domain.Timezones(at)
		if err != nil {
			t.Fatal(err)
		}
		m := map[string]string{}
		for _, z := range zones {
			m[z.Label] = z.Offset
		}
		return m
	}
	inJanuary, inJuly := offsets(january), offsets(july)
	for _, tt := range tests {
		if inJanuary[tt.label] != tt.january || inJuly[tt.label] != tt.july {
			t.Errorf("%s: %q in January, %q in July; want %q, %q", tt.label, inJanuary[tt.label], inJuly[tt.label], tt.january, tt.july)
		}
	}
}
```

`server/internal/modules/instance/app/ports.go`（完整内容）：

```go
// Package app holds the instance module's use cases and the ports they need.
package app

import (
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/instance/domain"
)

// Clock tells the time. platform/clock implements it.
type Clock interface {
	Now() time.Time
}

// InfoSource reports the build this instance runs. GetInfo declares it;
// adapter/buildinfo implements it.
type InfoSource interface {
	Build() domain.Build
}
```

`server/internal/modules/instance/app/get_info.go`（完整内容）：

```go
package app

import "github.com/open-nerve/NerveProject/server/internal/modules/instance/domain"

// GetInfo tells API clients what this instance runs and how it is set up.
type GetInfo struct {
	source   InfoSource
	settings domain.Settings
}

// NewGetInfo returns the use case, reading the build from source.
func NewGetInfo(source InfoSource, settings domain.Settings) *GetInfo {
	return &GetInfo{source: source, settings: settings}
}

// Execute describes the instance.
func (uc *GetInfo) Execute() domain.Info {
	build := uc.source.Build()
	return domain.Info{
		Product:    domain.Product,
		Version:    build.Version,
		Commit:     build.Commit,
		APIVersion: domain.APIVersion,
		Settings:   uc.settings,
	}
}
```

`server/internal/modules/instance/app/get_info_test.go`（完整内容）：

```go
package app_test

import (
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/instance/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/domain"
)

type fixedSource domain.Build

func (s fixedSource) Build() domain.Build { return domain.Build(s) }

func TestGetInfoDescribesTheBuildAndTheSettings(t *testing.T) {
	settings := domain.Settings{SignupEnabled: true, FileSizeLimit: 5242880}
	uc := app.NewGetInfo(fixedSource{Version: "1.2.3", Commit: "4f2a9c1"}, settings)

	got := uc.Execute()

	want := domain.Info{Product: "Nerve", Version: "1.2.3", Commit: "4f2a9c1", APIVersion: "v0", Settings: settings}
	if got != want {
		t.Errorf("Execute() = %+v, want %+v", got, want)
	}
}
```

`server/internal/modules/instance/app/list_timezones.go`（新文件）：

```go
package app

import "github.com/open-nerve/NerveProject/server/internal/modules/instance/domain"

// ListTimezones lists the time zones the web app offers:
// GET /api/v0/timezones.
type ListTimezones struct {
	clock Clock
}

// NewListTimezones returns the use case.
func NewListTimezones(clock Clock) *ListTimezones {
	return &ListTimezones{clock: clock}
}

// Execute returns the time zones with their offsets at the clock's now.
func (uc *ListTimezones) Execute() ([]domain.Timezone, error) {
	return domain.Timezones(uc.clock.Now())
}
```

`server/internal/modules/instance/app/list_timezones_test.go`（新文件）：

```go
package app_test

import (
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/instance/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
)

// The offsets are the clock's: Dublin is an hour east of UTC in summer.
func TestListTimezonesAtTheClocksNow(t *testing.T) {
	for _, tt := range []struct {
		at   time.Time
		want string
	}{
		{time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC), "+00:00"},
		{time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC), "+01:00"},
	} {
		zones, err := app.NewListTimezones(clocktest.At(tt.at)).Execute()
		if err != nil {
			t.Fatal(err)
		}
		var dublin string
		for _, z := range zones {
			if z.Name == "Europe/Dublin" {
				dublin = z.Offset
			}
		}
		if dublin != tt.want {
			t.Errorf("Dublin at %v: %q, want %s", tt.at, dublin, tt.want)
		}
	}
}
```

- [ ] **Step 2: 契约**

`api/openapi.yaml`（对 Task 11 版本的差异）：

```diff
--- a/api/openapi.yaml
+++ b/api/openapi.yaml
@@ -43,6 +43,8 @@
     $ref: 'modules/identity.yaml#/paths/~1api~1v0~1api-tokens~1{token_id}'
   /api/v0/instance:
     $ref: 'modules/instance.yaml#/paths/~1api~1v0~1instance'
+  /api/v0/timezones:
+    $ref: 'modules/instance.yaml#/paths/~1api~1v0~1timezones'
 components:
   # 模块文件的 securitySchemes 在打包时被丢掉，操作引用的 scheme 写在这里（M0-P3 交接 3）
   securitySchemes:
```

`api/modules/instance.yaml`（完整内容）：

```yaml
# instance 模块的接口。生成的 Go 代码在
# server/internal/modules/instance/adapter/http/gen（make gen-go）。
openapi: 3.1.0
info:
  title: Nerve instance API
  version: v0
paths:
  /api/v0/instance:
    get:
      operationId: getInstance
      tags: [instance]
      summary: Describe this instance
      description: >-
        Reports the product, the build and the API version this instance
        runs, and the settings clients adapt to. Public: needs no
        authentication.
      security: []
      x-problem-codes: []
      responses:
        '200':
          description: What this instance runs.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/InstanceInfo'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/timezones:
    get:
      operationId: listTimezones
      tags: [instance]
      summary: List the time zones to choose from
      description: >-
        The time zones the web app offers, each with its offset at the time
        of the request, sorted by offset and then by label. user_timezone
        takes any IANA name, not only these. Public: needs no
        authentication.
      security: []
      x-problem-codes: []
      responses:
        '200':
          description: The time zones.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/TimezoneList'
        default:
          $ref: '#/components/responses/Problem'
components:
  # 与 api/openapi.yaml 的相同：打包时丢掉模块文件的 securitySchemes，oapi-codegen 读这一份（M0-P3 交接 3）
  securitySchemes:
    bearer:
      type: http
      scheme: bearer
  responses:
    Problem:
      description: Error (RFC 9457 problem details).
      headers:
        Retry-After:
          description: >-
            Whole seconds to wait before trying again, rounded up; sent with
            rate_limited and server_busy.
          schema:
            type: integer
            minimum: 1
        WWW-Authenticate:
          description: >-
            Sent with every 401 (RFC 9110 15.5.2): Bearer, or
            Bearer error="invalid_token" when the bearer token sent is invalid
            or has expired (RFC 6750 3).
          schema:
            type: string
      content:
        application/problem+json:
          schema:
            $ref: '../common.yaml#/components/schemas/Problem'
  schemas:
    InstanceInfo:
      type: object
      additionalProperties: false
      required: [product, version, commit, api_version, signup_enabled, workspace_creation_enabled, file_size_limit]
      properties:
        product:
          description: Product name.
          type: string
          examples: [Nerve]
        version:
          description: Product version of the running build.
          type: string
          examples: [0.1.0-dev]
        commit:
          description: Git revision of the running build; "unknown" when the build carries no VCS stamp.
          type: string
        api_version:
          description: Version of this HTTP API; every path starts with /api/{api_version}.
          type: string
          enum: [v0]
        signup_enabled:
          description: Whether anyone may register (auth.signup_enabled).
          type: boolean
        workspace_creation_enabled:
          description: Whether workspaces can be created (workspace.creation_enabled).
          type: boolean
        file_size_limit:
          description: The largest file an upload may have, in bytes (files.size_limit).
          type: integer
          minimum: 1
    Timezone:
      type: object
      additionalProperties: false
      required: [label, value, utc_offset, gmt_offset]
      properties:
        label:
          description: A place in the time zone, e.g. Beijing.
          type: string
        value:
          description: The IANA name, what user_timezone takes.
          type: string
          examples: [Asia/Shanghai]
        utc_offset:
          description: The offset at the time of the request.
          type: string
          examples: [UTC+08:00]
        gmt_offset:
          description: The same offset, written from GMT.
          type: string
          examples: [GMT+08:00]
    TimezoneList:
      type: object
      additionalProperties: false
      required: [data]
      properties:
        data:
          type: array
          items:
            $ref: '#/components/schemas/Timezone'
```

- [ ] **Step 3: 生成**

Run: `make gen`
Expected: 以下四个文件改变，其余生成物不变；四个都是最终版本：

| 生成的文件 | SHA-256 | 行数 |
|---|---|---|
| `api/dist/openapi.yaml` | `836664152d95b06a5deeb3fd2e689da6e15e261ac270d19391983da6e216c697` | 878 |
| `server/internal/modules/instance/adapter/http/gen/bodyshape.gen.go` | `58984645db9d05e171021d200a9959593a0f73b86769b9c9f44e49c909399629` | 15 |
| `server/internal/modules/instance/adapter/http/gen/server.gen.go` | `3978a7437d261b5e11d700e15742171766e388bdec8e071a5e948010173989cd` | 461 |
| `web/packages/api-client/src/schema.gen.ts` | `86f897bef4954e3a0df71af5c9a75a357780069c8a515017948779132b58d39a` | 883 |

- [ ] **Step 4: HTTP 适配器、模块和接线**

`server/internal/modules/instance/adapter/http/handler.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/modules/instance/adapter/http/handler.go
+++ b/server/internal/modules/instance/adapter/http/handler.go
@@ -13,13 +13,14 @@
 
 // UseCases are the use cases behind the module's operations.
 type UseCases struct {
-	GetInfo *app.GetInfo
+	GetInfo       *app.GetInfo
+	ListTimezones *app.ListTimezones
 }
 
 // PublicOperations are the module's routes that need no token (M2 design
 // 3.6), as the generated code registers them.
 func PublicOperations() []string {
-	return []string{"GET /api/v0/instance"}
+	return []string{"GET /api/v0/instance", "GET /api/v0/timezones"}
 }
 
 // Register mounts the module's routes on router, the root router from
@@ -57,5 +58,22 @@
 		Version:    info.Version,
 		Commit:     info.Commit,
 		APIVersion: gen.InstanceInfoAPIVersion(info.APIVersion),
+
+		SignupEnabled:            info.SignupEnabled,
+		WorkspaceCreationEnabled: info.WorkspaceCreationEnabled,
+		FileSizeLimit:            int(info.FileSizeLimit),
 	}, nil
 }
+
+// ListTimezones serves GET /api/v0/timezones.
+func (h handler) ListTimezones(context.Context, gen.ListTimezonesRequestObject) (gen.ListTimezonesResponseObject, error) {
+	zones, err := h.uc.ListTimezones.Execute()
+	if err != nil {
+		return nil, err
+	}
+	out := gen.ListTimezones200JSONResponse{Data: make([]gen.Timezone, len(zones))}
+	for i, z := range zones {
+		out.Data[i] = gen.Timezone{Label: z.Label, Value: z.Name, UtcOffset: "UTC" + z.Offset, GmtOffset: "GMT" + z.Offset}
+	}
+	return out, nil
+}
```

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
	"strings"
	"testing"
	"time"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/instance/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
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

// get serves the module with uc and answers a GET of path, checked against
// the contract.
func get(t *testing.T, uc httpadapter.UseCases, path string) (*http.Response, string) {
	t.Helper()
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
	httpadapter.Register(router, api, uc)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	res := rec.Result()

	apitest.Load(t).CheckResponse(t, req, res)
	body, _ := io.ReadAll(res.Body)
	return res, string(body)
}

func TestGetInstanceMatchesTheContract(t *testing.T) {
	getInfo := app.NewGetInfo(fixedSource{Version: "1.2.3", Commit: "4f2a9c1"},
		domain.Settings{SignupEnabled: true, WorkspaceCreationEnabled: false, FileSizeLimit: 5242880})

	res, body := get(t, httpadapter.UseCases{GetInfo: getInfo}, "/api/v0/instance")

	want := `{"api_version":"v0","commit":"4f2a9c1","file_size_limit":5242880,"product":"Nerve","signup_enabled":true,` +
		`"version":"1.2.3","workspace_creation_enabled":false}` + "\n"
	if res.StatusCode != http.StatusOK || body != want {
		t.Errorf("GET /api/v0/instance = %d %s, want 200 %s", res.StatusCode, body, want)
	}
}

func TestListTimezones(t *testing.T) {
	list := app.NewListTimezones(clocktest.At(time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)))

	res, body := get(t, httpadapter.UseCases{ListTimezones: list}, "/api/v0/timezones")

	first := `{"data":[{"gmt_offset":"GMT-11:00","label":"American Samoa","utc_offset":"UTC-11:00","value":"Pacific/Pago_Pago"},`
	marquesas := `{"gmt_offset":"GMT-09:30","label":"Marquesas Islands","utc_offset":"UTC-09:30","value":"Pacific/Marquesas"}`
	if res.StatusCode != http.StatusOK || !strings.HasPrefix(body, first) || !strings.Contains(body, marquesas) {
		t.Errorf("GET /api/v0/timezones = %d %s, want 200 starting %s and holding %s", res.StatusCode, body, first, marquesas)
	}
}
```

`server/internal/modules/instance/module.go`（完整内容）：

```go
// Package instance is the pilot module and the template for every module
// (M0 design 3.2): GET /api/v0/instance tells API clients what this nerve
// instance runs and how it is set up, and GET /api/v0/timezones lists the
// time zones to choose from (M2 design 5.3).
package instance

import (
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/adapter/buildinfo"
	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/instance/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

// Deps are what bootstrap gives the module: the settings the instance
// reports, from the configuration, and the clock.
type Deps struct {
	SignupEnabled            bool  // auth.signup_enabled
	WorkspaceCreationEnabled bool  // workspace.creation_enabled
	FileSizeLimit            int64 // files.size_limit, in bytes
	Clock                    app.Clock
}

// Module is the wired instance module.
type Module struct {
	uc httpadapter.UseCases
}

// New wires the module: GetInfo reads the build of the running binary.
func New(d Deps) *Module {
	settings := domain.Settings{
		SignupEnabled:            d.SignupEnabled,
		WorkspaceCreationEnabled: d.WorkspaceCreationEnabled,
		FileSizeLimit:            d.FileSizeLimit,
	}
	return &Module{uc: httpadapter.UseCases{
		GetInfo:       app.NewGetInfo(buildinfo.Source{}, settings),
		ListTimezones: app.NewListTimezones(d.Clock),
	}}
}

// PublicOperations are the module's routes that need no token.
func (m *Module) PublicOperations() []string {
	return httpadapter.PublicOperations()
}

// Register mounts the module's API on router, the root router from
// httpserver.NewRouter, behind api's per-route middlewares.
func (m *Module) Register(router *httpserver.Router, api *httpserver.API) {
	httpadapter.Register(router, api, m.uc)
}
```

`server/internal/bootstrap/app.go`（对 Task 10 版本的差异）：

```diff
--- a/server/internal/bootstrap/app.go
+++ b/server/internal/bootstrap/app.go
@@ -103,7 +103,12 @@
 		a.close()
 		return nil, err
 	}
-	inst := instance.New()
+	inst := instance.New(instance.Deps{
+		SignupEnabled:            cfg.Auth.SignupEnabled,
+		WorkspaceCreationEnabled: cfg.Workspace.CreationEnabled,
+		FileSizeLimit:            cfg.Files.SizeLimit,
+		Clock:                    clock.System{},
+	})
 
 	a.router = httpserver.NewRouter(logger,
 		httpserver.Check{Name: "database", Run: pool.Ping},
```

`server/internal/bootstrap/app_test.go`（对 Task 10 版本的差异）：

```diff
--- a/server/internal/bootstrap/app_test.go
+++ b/server/internal/bootstrap/app_test.go
@@ -73,7 +73,9 @@
 			Anonymous:     roomy, AuthFailure: roomy, Authenticated: roomy,
 			LoginIP: roomy, LoginIPEmail: roomy, RegisterIP: roomy, PasswordUser: roomy,
 		},
-		Log: config.LogConfig{Level: "error", Format: "text"},
+		Workspace: config.WorkspaceConfig{CreationEnabled: true},
+		Files:     config.FilesConfig{SizeLimit: 5242880},
+		Log:       config.LogConfig{Level: "error", Format: "text"},
 	}
 }
 
```

`server/internal/bootstrap/api_test.go`（完整内容）：

```go
package bootstrap

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"testing/fstest"

	"github.com/open-nerve/NerveProject/server/internal/platform/buildinfo"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
)

// The API needs no database: the app runs against an unreachable one. The
// instance reports the settings of the configuration (M2 design 5.3).
func TestServesTheInstanceAPI(t *testing.T) {
	contract := apitest.Load(t)
	cfg := testConfig(t, unreachableDB, false)
	cfg.Auth.SignupEnabled, cfg.Workspace.CreationEnabled, cfg.Files.SizeLimit = true, false, 7340032
	base := startApp(t, cfg, fstest.MapFS{})
	req, err := http.NewRequest(http.MethodGet, base+"/api/v0/instance", nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()

	contract.CheckResponse(t, req, res)
	var got struct {
		Product                  string `json:"product"`
		Version                  string `json:"version"`
		APIVersion               string `json:"api_version"`
		SignupEnabled            bool   `json:"signup_enabled"`
		WorkspaceCreationEnabled bool   `json:"workspace_creation_enabled"`
		FileSizeLimit            int64  `json:"file_size_limit"`
	}
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if res.StatusCode != http.StatusOK || got.Product != "Nerve" || got.Version != buildinfo.Get().Version || got.APIVersion != "v0" {
		t.Errorf("GET /api/v0/instance = %d %+v, want 200 Nerve %s v0", res.StatusCode, got, buildinfo.Get().Version)
	}
	if !got.SignupEnabled || got.WorkspaceCreationEnabled || got.FileSizeLimit != 7340032 {
		t.Errorf("settings %+v, want sign-up on, workspace creation off, 7340032 bytes", got)
	}
	if res.Header.Get(httpserver.HeaderRequestID) == "" {
		t.Error("response has no X-Request-Id: the platform middleware did not run")
	}
}

// The time zones need no token and no database either.
func TestServesTheTimezones(t *testing.T) {
	contract := apitest.Load(t)
	base := startApp(t, testConfig(t, unreachableDB, false), fstest.MapFS{})
	req := newRequest(t, http.MethodGet, base+"/api/v0/timezones", "", nil)

	res, body := send(t, req)

	contract.CheckResponse(t, req, res)
	var got struct {
		Data []struct {
			Value string `json:"value"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &got); err != nil || res.StatusCode != http.StatusOK || len(got.Data) != 120 {
		t.Errorf("GET /api/v0/timezones = %d, %d zones (%v); want 200 with 120", res.StatusCode, len(got.Data), err)
	}
}

// Mounting a module keeps the platform's /api/ fallback: an unknown path and
// a known path with the wrong method both answer 404 problem+json.
func TestUnknownAPIRequestsStillAnswerProblem404(t *testing.T) {
	contract := apitest.Load(t)
	base := startApp(t, testConfig(t, unreachableDB, false), fstest.MapFS{})
	for _, tt := range []struct{ method, path string }{
		{http.MethodGet, "/api/v0/nope"},
		{http.MethodPost, "/api/v0/instance"},
	} {
		req, err := http.NewRequest(tt.method, base+tt.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(res.Body)
		_ = res.Body.Close()
		if err != nil {
			t.Fatal(err)
		}

		if res.StatusCode != http.StatusNotFound || res.Header.Get("Content-Type") != httpserver.ContentTypeProblem {
			t.Errorf("%s %s = %d %s, want 404 problem+json", tt.method, tt.path, res.StatusCode, res.Header.Get("Content-Type"))
		}
		contract.CheckSchema(t, "Problem", body)
		var p httpserver.Problem
		if err := json.Unmarshal(body, &p); err != nil || p.Detail != "no API endpoint for "+tt.method+" "+tt.path {
			t.Errorf("%s %s detail = %q (%v), want no API endpoint for %s %s", tt.method, tt.path, p.Detail, err, tt.method, tt.path)
		}
	}
}
```

`server/internal/platform/httpserver/apitest/apitest_test.go`（对 `605f367` 的差异）：

```diff
--- a/server/internal/platform/httpserver/apitest/apitest_test.go
+++ b/server/internal/platform/httpserver/apitest/apitest_test.go
@@ -16,7 +16,8 @@
 	"github.com/getkin/kin-openapi/openapi3"
 )
 
-const instanceJSON = `{"product":"Nerve","version":"0.1.0-dev","commit":"unknown","api_version":"v0"}`
+const instanceJSON = `{"product":"Nerve","version":"0.1.0-dev","commit":"unknown","api_version":"v0",` +
+	`"signup_enabled":true,"workspace_creation_enabled":true,"file_size_limit":5242880}`
 
 // Component names become Go and TypeScript type names. Redocly's bundler
 // renames a clash between module files to "Name-2" and only warns, so a
```

- [ ] **Step 5: 时区数据库**

`server/cmd/nerve/main.go`（对 `605f367` 的差异）：

```diff
--- a/server/cmd/nerve/main.go
+++ b/server/cmd/nerve/main.go
@@ -8,6 +8,7 @@
 	"os"
 	"os/signal"
 	"syscall"
+	_ "time/tzdata" // every zone is known, whatever the host has (M2 design 4.2)
 )
 
 func main() {
```

`server/internal/archtest/binary_test.go`（对 Task 3 版本的差异）：

```diff
--- a/server/internal/archtest/binary_test.go
+++ b/server/internal/archtest/binary_test.go
@@ -50,6 +50,16 @@
 	}
 }
 
+// The binary embeds the time zone database (M2 design 4.2): which
+// user_timezone is accepted, and the offsets of GET /api/v0/timezones, must
+// not depend on the zones of the host, which a container may lack.
+func TestNerveBinaryEmbedsTheTimeZoneDatabase(t *testing.T) {
+	g := loadDeps(t, "./cmd/nerve")
+	if !slices.Contains(g[m("cmd/nerve")], "time/tzdata") {
+		t.Errorf("cmd/nerve imports %q, not time/tzdata", g[m("cmd/nerve")])
+	}
+}
+
 // google/uuid is reached first through oapi-codegen/runtime, which may
 // import it; every other import of it is still reported, each on its own.
 func TestBannedImports(t *testing.T) {
```

- [ ] **Step 6: S3**

`e2e/stories/smoke/s3-instance-info.spec.ts`（对 `605f367` 的差异）：

```diff
--- a/e2e/stories/smoke/s3-instance-info.spec.ts
+++ b/e2e/stories/smoke/s3-instance-info.spec.ts
@@ -13,5 +13,8 @@
     version,
     commit: expect.stringMatching(/^[0-9a-f]{40}$/),
     api_version: "v0",
+    signup_enabled: true,
+    workspace_creation_enabled: true,
+    file_size_limit: 5242880,
   });
 });
```

- [ ] **Step 7: lint、测试和端到端**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。

Run: `make lint-web`
Expected: 通过。

Run: `make knip`
Expected: 通过。

Run: `make e2e`
Expected: 全部通过；S3 核对三个新字段。

- [ ] **Step 8: 提交**

```bash
git add api web/packages/api-client/src/schema.gen.ts server/cmd/nerve/main.go server/internal/modules/instance server/internal/archtest/binary_test.go server/internal/bootstrap server/internal/platform/httpserver/apitest/apitest_test.go e2e/stories/smoke/s3-instance-info.spec.ts
```
```bash
git commit -m "feat(M2/P3a): the instance reports its settings; the time zone list; the binary embeds tzdata

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run: `make gen-check`
Expected: 没有差异。

**Done when:** 时区和实例配置的测试、`TestNerveBinaryEmbedsTheTimeZoneDatabase` 通过；四个生成物与上表相同；`make e2e` 全部通过（含 S3）；`make gen-check` 干净。

---

### Task 13: 账户行锁的交错测试 4、5（真实数据库）

**Files:**
- Create: `server/internal/modules/identity/interleavings_test.go`

**Interfaces:**
- Consumes: `app.Login`（P2）、`app.ChangePassword`（Task 10）、`postgresadapter.Store`、`postgres.TxManager`、`signing`（真实的）。
- 放在模块根目录的测试包 `identity_test`：它把真实的存储、事务和签名接到用例上，这是模块入口的组合；`app` 的测试包不导入适配器（spec 第 3 节第 16 条）。
- 交错 1–3 要管理员重置密码，随 P3b 加入；交错 6 在 P2（`TestTheCredentialLockBlocksLocksNotInserts`）。

**Tests:**
- `gatedHasher`：`Hash` 给出 `hashed:<密码>:<盐>`；`Verify` 认任何盐的 `hashed:`，把 `old:<密码>` 当作旧参数的哈希并要求重新哈希；带闸门时第一次 `Verify` 停在闸门上，直到测试打开；记下校验过的每个哈希。每个等待都有 10 秒的期限，超时即失败，不会挂住。
- `TestALoginWithTheOldPasswordFailsWhenThePasswordChangesMeanwhile`（交错 4）：登录校验了旧密码、停在闸门上；修改密码在这时提交；登录在锁下看到另一个哈希，对它再校验一次，失败：401 `identity.invalid_credentials`，没有新会话，存的是新哈希；登录校验过的正是旧哈希和新哈希。
- `TestTwoLoginsWhileOneHashesThePasswordAgain`（交错 5）：两个登录都读到旧参数的哈希；第二个重新哈希并先提交；第一个在锁下看到新哈希，再校验一次后成功，不用自己的哈希覆盖新哈希：三个会话，存的是第二个登录的哈希。

- [ ] **Step 1: 交错测试**

`server/internal/modules/identity/interleavings_test.go`（新文件）：

```go
package identity_test

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/signing"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The interleavings of the account row lock protocol (M2 design 3.5) run
// the use cases on a real database. A gated hasher stops one of them in
// argon2, outside its transaction, while the test commits another; every
// wait has a deadline, so a test fails rather than hangs.

// waitLimit bounds every wait of these tests.
const waitLimit = 10 * time.Second

// gatedHasher stands in for argon2: Hash gives "hashed:<password>:<salt>",
// and Verify takes that for any salt, and "old:<password>" as a hash of
// other parameters, which it asks to rehash. With a gate, its first Verify
// stops until the test opens the gate. It records the hashes it verified.
type gatedHasher struct {
	salt     string
	gate     *gate // nil: never stops
	mu       sync.Mutex
	verified []string
}

func (h *gatedHasher) Hash(_ context.Context, password string) (string, error) {
	return "hashed:" + password + ":" + h.salt, nil
}

func (h *gatedHasher) Verify(_ context.Context, password, hash string) (bool, bool, error) {
	h.mu.Lock()
	h.verified = append(h.verified, hash)
	first := len(h.verified) == 1
	h.mu.Unlock()
	if first && h.gate != nil {
		if err := h.gate.stop(); err != nil {
			return false, false, err
		}
	}
	if hash == "old:"+password {
		return true, true, nil
	}
	return strings.HasPrefix(hash, "hashed:"+password+":"), false, nil
}

// gate stops a Verify until the test opens it.
type gate struct {
	reached, opened chan struct{}
}

func newGate() *gate { return &gate{reached: make(chan struct{}), opened: make(chan struct{})} }

// stop tells the test that a Verify reached the gate and waits until it is
// opened.
func (g *gate) stop() error {
	close(g.reached)
	select {
	case <-g.opened:
		return nil
	case <-time.After(waitLimit):
		return errors.New("the gate was never opened")
	}
}

// await waits until a Verify stops at the gate.
func (g *gate) await(t *testing.T) {
	t.Helper()
	select {
	case <-g.reached:
	case <-time.After(waitLimit):
		t.Fatal("nothing reached the gate")
	}
}

// account is alice@corp.com, whose password is Tr0ub4dor&3 and whose
// stored hash is given, with a live session, on a database of its own.
type account struct {
	store   *postgresadapter.Store
	pool    *pgxpool.Pool
	tx      shared.TxManager
	id      uuid.UUID
	session uuid.UUID
}

func newAccount(t *testing.T, hash string) *account {
	t.Helper()
	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, config.DatabaseConfig{URL: pgtest.NewDatabase(t), MaxConns: 8})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	a := &account{store: postgresadapter.New(pool), pool: pool, tx: postgres.NewTxManager(pool, 2*time.Second), id: uuid.NewV7(), session: uuid.NewV7()}
	now := time.Now()
	err = errors.Join(
		a.store.CreateUser(ctx, app.NewUser{ID: a.id, Email: "alice@corp.com", PasswordHash: hash, DisplayName: "alice", Now: now}),
		a.store.CreateDefaultProfile(ctx, uuid.NewV7(), a.id, now),
		a.store.CreateSession(ctx, app.NewSession{ID: a.session, UserID: a.id, TokenHash: make([]byte, 32), ExpiresAt: now.Add(time.Hour), Now: now}),
	)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func (a *account) login(h app.PasswordHasher) *app.Login {
	keys := signing.EphemeralKeys()
	return app.NewLogin(app.LoginDeps{
		Accounts: a.store, Locker: a.store, Passwords: a.store, Sessions: a.store, Hasher: h, Tx: a.tx,
		Issuance: app.Issuance{Tokens: signing.NewAccessTokens(keys), MAC: signing.NewRefreshTokenMAC(keys), AccessTTL: time.Minute, SessionTTL: time.Hour},
		Clock:    clock.System{}, Logger: slog.New(slog.DiscardHandler), DummyHash: "hashed:dummy:0",
	})
}

func (a *account) changePassword(h app.PasswordHasher) *app.ChangePassword {
	return app.NewChangePassword(app.ChangePasswordDeps{
		Accounts: a.store, Lock: app.CredentialLock{Locker: a.store, Sessions: a.store, APITokens: a.store},
		Passwords: a.store, Sessions: a.store, Hasher: h, Rules: domain.NewPasswordRules(), Tx: a.tx,
		Clock: clock.System{}, Logger: slog.New(slog.DiscardHandler),
	})
}

// loginAsync starts a login with Tr0ub4dor&3 and returns where its error
// arrives.
func loginAsync(uc *app.Login) <-chan error {
	done := make(chan error, 1)
	go func() {
		_, err := uc.Execute(context.Background(), app.LoginInput{Email: "alice@corp.com", Password: "Tr0ub4dor&3"})
		done <- err
	}()
	return done
}

func await(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(waitLimit):
		t.Fatal("the login did not finish")
		return nil
	}
}

// sessionsAndHash are the account's session count and its stored hash.
func (a *account) sessionsAndHash(t *testing.T) (int, string) {
	t.Helper()
	var n int
	var hash string
	err := a.pool.QueryRow(context.Background(),
		`SELECT (SELECT count(*) FROM auth_sessions WHERE user_id = $1), password FROM users WHERE id = $1`, a.id).Scan(&n, &hash)
	if err != nil {
		t.Fatal(err)
	}
	return n, hash
}

// Interleaving 4: a login verified the old password; the password changes
// before the login's transaction. Under the lock the login finds another
// hash, verifies the password again, against it, and fails: no session.
func TestALoginWithTheOldPasswordFailsWhenThePasswordChangesMeanwhile(t *testing.T) {
	a := newAccount(t, "hashed:Tr0ub4dor&3:0")
	g := newGate()
	loginHasher := &gatedHasher{salt: "login", gate: g}

	done := loginAsync(a.login(loginHasher))
	g.await(t)
	changed := a.changePassword(&gatedHasher{salt: "change"}).Execute(
		shared.WithActor(context.Background(), shared.Actor{UserID: a.id, SessionID: a.session}),
		app.ChangePasswordInput{Current: "Tr0ub4dor&3", New: "N3w-Passw0rd!"})
	close(g.opened)
	err := await(t, done)

	sessions, hash := a.sessionsAndHash(t)
	if changed != nil || !errors.Is(err, domain.ErrInvalidCredentials) || sessions != 1 || hash != "hashed:N3w-Passw0rd!:change" {
		t.Errorf("change %v; login %v; %d sessions, hash %q; want the change, 401, only the session of before, the new hash", changed, err, sessions, hash)
	}
	if want := []string{"hashed:Tr0ub4dor&3:0", "hashed:N3w-Passw0rd!:change"}; !slices.Equal(loginHasher.verified, want) {
		t.Errorf("the login verified %q, want %q", loginHasher.verified, want)
	}
}

// Interleaving 5: two logins read the hash of old parameters; the second
// hashes the password again and commits first. The first finds the new
// hash under the lock, verifies the password against it once more, and
// signs in, without writing its own hash over the new one.
func TestTwoLoginsWhileOneHashesThePasswordAgain(t *testing.T) {
	a := newAccount(t, "old:Tr0ub4dor&3")
	g := newGate()
	first := &gatedHasher{salt: "first", gate: g}

	done := loginAsync(a.login(first))
	g.await(t)
	_, second := a.login(&gatedHasher{salt: "second"}).Execute(context.Background(), app.LoginInput{Email: "alice@corp.com", Password: "Tr0ub4dor&3"})
	close(g.opened)
	err := await(t, done)

	sessions, hash := a.sessionsAndHash(t)
	if second != nil || err != nil || sessions != 3 || hash != "hashed:Tr0ub4dor&3:second" {
		t.Errorf("logins %v, %v; %d sessions, hash %q; want both signed in beside the session of before, the second's hash", err, second, sessions, hash)
	}
	if want := []string{"old:Tr0ub4dor&3", "hashed:Tr0ub4dor&3:second"}; !slices.Equal(first.verified, want) {
		t.Errorf("the first login verified %q, want %q", first.verified, want)
	}
}
```

- [ ] **Step 2: lint 和测试**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`。

Run: `go -C server test -count=5 -run 'TestALoginWithTheOldPassword|TestTwoLoginsWhile' ./internal/modules/identity/`
Expected: `ok`（交错由闸门决定，不靠运气）。

- [ ] **Step 3: 提交**

```bash
git add server/internal/modules/identity/interleavings_test.go
```
```bash
git commit -m "test(M2/P3a): interleavings 4 and 5 of the account row lock on a real database

A gated hasher stops a login in argon2 while the test commits a password
change or another login that rehashes; every wait has a deadline.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 两个交错测试通过，`-count=5` 也通过；`make test` 通过。

---

### Task 14: 端到端：A7–A11 的接口版本

**Files:**
- Modify: `e2e/fixtures/auth.ts`、`e2e/fixtures/assert/identity.ts`
- Create: `e2e/stories/identity/a7-change-password.spec.ts`、`a8-update-me.spec.ts`、`a9-preferences.spec.ts`、`a10-onboarding-profile.spec.ts`、`a11-api-tokens.spec.ts`

**Interfaces:**
- Consumes: Task 7–11 的操作。
- Produces（spec 2.19，M2 设计 2、9.5；M0-P6 交接）：
  - `auth.ts`：`ApiTokenCreated` 类型；`bearer(token)`（`Authorization` 请求头）；`createPAT(api, token, body = {})`（断言 201，返回带令牌的结果）；
  - `assert/identity.ts`：`AccountRow`、`accountOf(db, email)`、`tokensOf(db, userId)`、`expectPasswordChanged(db, before, tokensBefore, survivingSession?)`（哈希换成另一个 argon2id；除 `survivingSession` 外的会话都是 `password_changed`；PAT 不变。页面版本（P5）传入页面自己的会话）、`expectTokenStored(db, id, token, expiredAt)`（`token_hash` 是令牌的 SHA-256；整行的任何一列都不含令牌；`expired_at`；未撤销）。
- 这些故事的接口版本只用 PAT；页面版本（P4、P5）调用同一组断言函数。

**Tests:**
- `a7-change-password.spec.ts`：当前密码错误 422 `identity.current_password_incorrect`，账户不变；正确时 204，`expectPasswordChanged`（PAT 没有自己的会话，全部会话结束）；PAT 仍能读 `/me`；旧密码登录 401，新密码 200。
- `a8-update-me.spec.ts`：改名、姓、显示名和时区，响应和数据库都是新值，`updated_at` 变大；`{"first_name": null}` 是 400 `bad_request`（`first_name` `invalid_format`），数据库不变。
- `a9-preferences.spec.ts`：主题、语言、每周第一天；数据库和 `GET /me/profile` 读回的相同。
- `a10-onboarding-profile.spec.ts`：`PATCH /me` 设名；只带 `profile_complete` 的 `PATCH /me/profile` 合并进去，其余三个键不变；拼错的键 `profile_completed` 是 400 `onboarding_step.profile_completed` `not_allowed`，数据库不变。
- `a11-api-tokens.spec.ts`：用一个 PAT 创建另一个（名称、说明、一周后到期）：`expectTokenStored`；每页 1 个翻完列表，新的在前，列表里没有令牌原文；新令牌读 `/me` 200，`last_used` 已写；撤销后 `deleted_at` 已填、读 `/me` 401、从列表消失；把第三个令牌的 `expired_at` 改到过去，读 `/me` 401。

- [ ] **Step 1: fixture**

`e2e/fixtures/auth.ts`（对 `605f367` 的差异）：

```diff
--- a/e2e/fixtures/auth.ts
+++ b/e2e/fixtures/auth.ts
@@ -4,6 +4,7 @@
 import type { Api } from "./api";
 
 export type AuthTokens = components["schemas"]["AuthTokens"];
+export type ApiTokenCreated = components["schemas"]["ApiTokenCreated"];
 
 /** A password that meets the rules and is not common. */
 export const password = "Tr0ub4dor&3";
@@ -42,6 +43,25 @@
   return data;
 }
 
+/** The Authorization header of a bearer token: an access token or a personal access token. */
+export function bearer(token: string): Record<string, string> {
+  return { Authorization: `Bearer ${token}` };
+}
+
+/** Creates a personal access token with the bearer token given, and returns it with its token. */
+export async function createPAT(
+  api: Api,
+  token: string,
+  body: components["schemas"]["ApiTokenCreate"] = {}
+): Promise<ApiTokenCreated> {
+  const { data, error, response } = await api.POST("/api/v0/me/api-tokens", { body, headers: bearer(token) });
+  expect(response.status, `create a personal access token: ${JSON.stringify(error)}`).toBe(201);
+  if (!data) {
+    throw new Error("createApiToken answered 201 without the token");
+  }
+  return data;
+}
+
 /** Exchanges refreshToken for the session's next tokens. */
 export async function refresh(api: Api, refreshToken: string): Promise<AuthTokens> {
   const { data, error, response } = await api.POST("/api/v0/auth/refresh", { body: { refresh_token: refreshToken } });
```

`e2e/fixtures/assert/identity.ts`（对 `605f367` 的差异）：

```diff
--- a/e2e/fixtures/assert/identity.ts
+++ b/e2e/fixtures/assert/identity.ts
@@ -170,6 +170,78 @@
   expect(session.revoke_reason).toBe(reason);
 }
 
+/** An account's row, as the account stories read it. */
+export interface AccountRow {
+  id: string;
+  password: string;
+  first_name: string;
+  last_name: string;
+  display_name: string;
+  user_timezone: string;
+  is_active: boolean;
+  updated_at: Date;
+}
+
+/** The account of email, a lowercased address. */
+export async function accountOf(db: Database, email: string): Promise<AccountRow> {
+  const rows = await db.query<AccountRow>(
+    `SELECT id, password, first_name, last_name, display_name, user_timezone, is_active, updated_at
+       FROM users WHERE email = $1`,
+    [email]
+  );
+  expect(rows, `the account of ${email}`).toHaveLength(1);
+  return rows[0] as AccountRow;
+}
+
+/** The personal access tokens of the account userId, oldest first: id and deleted_at. */
+export async function tokensOf(db: Database, userId: string): Promise<{ id: string; deleted_at: Date | null }[]> {
+  return db.query("SELECT id, deleted_at FROM api_tokens WHERE user_id = $1 ORDER BY created_at, id", [userId]);
+}
+
+/**
+ * A7: the password of the account `before` changed, to another argon2id
+ * hash; every session of the account is revoked for password_changed but
+ * survivingSession, the refresh token of the page's own session, which
+ * stays live; the personal access tokens are as they were (M2 design 3.5).
+ */
+export async function expectPasswordChanged(
+  db: Database,
+  before: AccountRow,
+  tokensBefore: { id: string; deleted_at: Date | null }[],
+  survivingSession?: string
+): Promise<void> {
+  const [after] = await db.query<{ password: string }>("SELECT password FROM users WHERE id = $1", [before.id]);
+  expect(after?.password).toMatch(/^\$argon2id\$/);
+  expect(after?.password).not.toBe(before.password);
+  const surviving = survivingSession === undefined ? undefined : (await sessionOf(db, survivingSession)).id;
+  const sessions = await db.query<{ id: string; revoke_reason: string | null }>(
+    "SELECT id, revoke_reason FROM auth_sessions WHERE user_id = $1",
+    [before.id]
+  );
+  expect(sessions.length).toBeGreaterThan(0);
+  for (const s of sessions) {
+    expect(s.revoke_reason, `session ${s.id}`).toBe(s.id === surviving ? null : "password_changed");
+  }
+  expect(await tokensOf(db, before.id)).toEqual(tokensBefore);
+}
+
+/**
+ * A11: the new token's row holds the SHA-256 of the token, and no column
+ * holds the token itself; it expires at expiredAt and is live.
+ */
+export async function expectTokenStored(db: Database, id: string, token: string, expiredAt: string): Promise<void> {
+  const rows = await db.query<{ token_hash: Buffer; expired_at: Date; deleted_at: Date | null; row: string }>(
+    "SELECT token_hash, expired_at, deleted_at, row_to_json(t)::text AS row FROM api_tokens t WHERE id = $1",
+    [id]
+  );
+  expect(rows).toHaveLength(1);
+  const [row] = rows;
+  expect(row?.token_hash.equals(createHash("sha256").update(token).digest())).toBe(true);
+  expect(row?.row).not.toContain(token.slice("nrv_pat_".length));
+  expect(row?.expired_at.getTime()).toBe(new Date(expiredAt).getTime());
+  expect(row?.deleted_at).toBeNull();
+}
+
 /** How many accounts, profiles and sessions there are. */
 export interface IdentityCounts {
   users: number;
```

- [ ] **Step 2: 五个故事**

`e2e/stories/identity/a7-change-password.spec.ts`（新文件）：

```ts
import { accountOf, expectPasswordChanged, tokensOf } from "../../fixtures/assert/identity";
import { bearer, createPAT, emailFor, password, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A7, changing the password (M2 design 2). The page version, which keeps
// its own session, joins in M2/P5.

const newPassword = "N3w-Passw0rd!";

test("A7 (API): a personal access token changes the password; every session ends, the token goes on", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const pat = await createPAT(api, (await register(api, email)).access_token);
  const change = (current: string) =>
    api.POST("/api/v0/me/change-password", {
      body: { current_password: current, new_password: newPassword },
      headers: bearer(pat.token),
    });
  const before = await accountOf(db, email);
  const tokensBefore = await tokensOf(db, before.id);

  // A wrong current password: its problem, and nothing changes.
  const wrong = await change("Wr0ng-password");
  expect(wrong.response.status).toBe(422);
  expect(wrong.error?.code).toBe("identity.current_password_incorrect");
  expect(await accountOf(db, email)).toEqual(before);

  const changed = await change(password);
  expect(changed.response.status).toBe(204);
  // A token has no session of its own: every session ends (M2 design 3.5).
  await expectPasswordChanged(db, before, tokensBefore);

  // The token goes on; the old password no longer signs in, the new one does.
  expect((await api.GET("/api/v0/me", { headers: bearer(pat.token) })).response.status).toBe(200);
  const old = await api.POST("/api/v0/auth/login", { body: { email, password } });
  expect(old.response.status).toBe(401);
  const renewed = await api.POST("/api/v0/auth/login", { body: { email, password: newPassword } });
  expect(renewed.response.status).toBe(200);
});
```

`e2e/stories/identity/a8-update-me.spec.ts`（新文件）：

```ts
import { accountOf } from "../../fixtures/assert/identity";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A8, changing the names and the time zone (M2 design 2). The page version
// joins in M2/P5.

test("A8 (API): a personal access token changes the names and the time zone; null is a 400", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const pat = await createPAT(api, (await register(api, email)).access_token);
  const before = await accountOf(db, email);
  const change = { first_name: "Ada", last_name: "Lovelace", display_name: "ada", user_timezone: "Asia/Shanghai" };

  const { data, response } = await api.PATCH("/api/v0/me", { body: change, headers: bearer(pat.token) });

  expect(response.status).toBe(200);
  expect(data).toMatchObject(change);
  const after = await accountOf(db, email);
  expect(after).toMatchObject(change);
  expect(after.updated_at.getTime()).toBeGreaterThan(before.updated_at.getTime());

  // null breaks the contract: the platform's 400, and nothing changes.
  const nulled = await api.PATCH("/api/v0/me", {
    // @ts-expect-error -- null for a name breaks the contract on purpose
    body: { first_name: null },
    headers: bearer(pat.token),
  });
  expect(nulled.response.status).toBe(400);
  expect(nulled.error?.code).toBe("bad_request");
  expect(nulled.error?.errors?.map((e) => ({ field: e.field, code: e.code }))).toEqual([
    { field: "first_name", code: "invalid_format" },
  ]);
  expect(await accountOf(db, email)).toEqual(after);
});
```

`e2e/stories/identity/a9-preferences.spec.ts`（新文件）：

```ts
import { accountOf } from "../../fixtures/assert/identity";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A9, changing the preferences (M2 design 2). The page version, in both
// languages, joins in M2/P5.

test("A9 (API): a personal access token changes the theme, the language and the first day of the week", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const pat = await createPAT(api, (await register(api, email)).access_token);
  const change = { theme: "dark", language: "zh-CN", start_of_the_week: 1 } as const;

  const { data, response } = await api.PATCH("/api/v0/me/profile", { body: change, headers: bearer(pat.token) });

  expect(response.status).toBe(200);
  expect(data).toMatchObject(change);
  const { id } = await accountOf(db, email);
  expect(await db.query("SELECT theme, language, start_of_the_week FROM profiles WHERE user_id = $1", [id])).toEqual([
    change,
  ]);
  const read = await api.GET("/api/v0/me/profile", { headers: bearer(pat.token) });
  expect(read.data).toEqual(data);
});
```

`e2e/stories/identity/a10-onboarding-profile.spec.ts`（新文件）：

```ts
import { accountOf } from "../../fixtures/assert/identity";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A10, the profile step of onboarding (M2 design 2). The page version, with
// the first visit of /onboarding, joins in M2/P4.

test("A10 (API): the profile step sets the name and one step, which the others keep beside", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const pat = await createPAT(api, (await register(api, email)).access_token);
  const { id } = await accountOf(db, email);
  const steps = async () =>
    (await db.query<{ onboarding_step: unknown }>("SELECT onboarding_step FROM profiles WHERE user_id = $1", [id]))[0]
      ?.onboarding_step;

  const named = await api.PATCH("/api/v0/me", { body: { first_name: "Ada" }, headers: bearer(pat.token) });
  expect(named.response.status).toBe(200);
  expect((await accountOf(db, email)).first_name).toBe("Ada");

  // One key: it is merged in, the other three keep their values (M2 design 3.14).
  const stepped = await api.PATCH("/api/v0/me/profile", {
    body: { onboarding_step: { profile_complete: true } },
    headers: bearer(pat.token),
  });
  expect(stepped.response.status).toBe(200);
  const merged = { profile_complete: true, workspace_create: false, workspace_invite: false, workspace_join: false };
  expect(await steps()).toEqual(merged);

  // An unknown key, here misspelt, breaks the contract: the platform's 400,
  // and nothing changes.
  const unknown = await api.PATCH("/api/v0/me/profile", {
    body: { onboarding_step: { profile_completed: true } },
    headers: bearer(pat.token),
  });
  expect(unknown.response.status).toBe(400);
  expect(unknown.error?.errors?.map((e) => ({ field: e.field, code: e.code }))).toEqual([
    { field: "onboarding_step.profile_completed", code: "not_allowed" },
  ]);
  expect(await steps()).toEqual(merged);
});
```

`e2e/stories/identity/a11-api-tokens.spec.ts`（新文件）：

```ts
import type { components } from "@nerve/api-client";

import { expectTokenStored } from "../../fixtures/assert/identity";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A11, personal access tokens (M2 design 2). The page version joins in
// M2/P5.

const weekMs = 7 * 24 * 60 * 60 * 1000;

test("A11 (API): a token creates, lists page by page and revokes another; a revoked or expired token fails", async ({
  api,
  db,
}, testInfo) => {
  const admin = await createPAT(api, (await register(api, emailFor(testInfo))).access_token, { label: "admin" });
  const expiredAt = new Date(Date.now() + weekMs).toISOString();

  const created = await createPAT(api, admin.token, { label: "deploy", description: "ci", expired_at: expiredAt });

  expect(created).toMatchObject({ label: "deploy", description: "ci", last_used: null });
  expect(created.token).toMatch(/^nrv_pat_[A-Za-z0-9_-]{43}$/);
  await expectTokenStored(db, created.id, created.token, expiredAt);

  // The list, a token a page, newest first, never shows a token itself.
  const list = async (cursor?: string): Promise<components["schemas"]["ApiToken"][]> => {
    const { data, response } = await api.GET("/api/v0/me/api-tokens", {
      params: { query: { limit: 1, cursor } },
      headers: bearer(admin.token),
    });
    expect(response.status).toBe(200);
    const page = data?.data ?? [];
    return data?.next_cursor ? [...page, ...(await list(data.next_cursor))] : page;
  };
  const listed = await list();
  expect(listed.map((t) => t.id)).toEqual([created.id, admin.id]);
  expect(JSON.stringify(listed)).not.toContain(created.token);

  // The token authenticates, and its use is recorded.
  expect((await api.GET("/api/v0/me", { headers: bearer(created.token) })).response.status).toBe(200);
  const [used] = await db.query<{ last_used: Date | null }>("SELECT last_used FROM api_tokens WHERE id = $1", [
    created.id,
  ]);
  expect(used?.last_used).not.toBeNull();

  // Revoked, it stops at once and leaves the list.
  const revoked = await api.DELETE("/api/v0/api-tokens/{token_id}", {
    params: { path: { token_id: created.id } },
    headers: bearer(admin.token),
  });
  expect(revoked.response.status).toBe(204);
  const [row] = await db.query<{ deleted_at: Date | null }>("SELECT deleted_at FROM api_tokens WHERE id = $1", [
    created.id,
  ]);
  expect(row?.deleted_at).not.toBeNull();
  expect((await api.GET("/api/v0/me", { headers: bearer(created.token) })).response.status).toBe(401);
  expect((await list()).map((t) => t.id)).toEqual([admin.id]);

  // A token past its expiry fails too.
  const old = await createPAT(api, admin.token, { label: "old" });
  await db.query("UPDATE api_tokens SET expired_at = now() - interval '1 minute' WHERE id = $1", [old.id]);
  expect((await api.GET("/api/v0/me", { headers: bearer(old.token) })).response.status).toBe(401);
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
Expected: 全部通过：S1–S4，A1–A11（接口版本），A15。

- [ ] **Step 4: 提交**

```bash
git add e2e/fixtures e2e/stories/identity
```
```bash
git commit -m "test(M2/P3a): the API versions of stories A7 to A11, with personal access tokens

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** `make e2e` 全部通过，含 A7–A11；`make lint-web`、`make knip` 通过。

---

### Task 15: 上级文档、差异清单、README 与交接

**Files:**
- Modify: `docs/v0/v0-design.md`、`docs/v0/M0-foundation/M0-design.md`、`docs/v0/M0-foundation/specs/P3-api-contract.md`、`docs/v0/plane-diff.md`、`README.md`
- Modify: `docs/v0/M2-auth/handoffs/M0-P3-api-codegen-notes.md`、`M0-P6-e2e-notes.md`、`M1-P2-trim-content.md`、`M1-P3-trim-platform.md`

**Interfaces:**
- M2 设计 3.20 中 P3a 的各行、8.7 中 P3a 的两行（spec 2.20）；交接的处理结果（spec 第 7 节）。M0-P3 的状态改为 `done`，另外三份仍为 `open`。

- [ ] **Step 1: 总体设计**（3.1 两个动作接口答 204；3.4 游标的封套、载荷和两种错误；3.6 `password_user` 的默认值；4.2 修改密码和停用时的会话撤销、账户行锁；`nerve users create` 在 P3b；6.2 `shared` 有了游标的封套）

`docs/v0/v0-design.md`（对 `605f367` 的差异）：

````diff
--- a/docs/v0/v0-design.md
+++ b/docs/v0/v0-design.md
@@ -146,7 +146,7 @@
 - **HTTP 方法**：
   - GET 查询，POST 创建，DELETE 删除（软删除）。
   - PATCH 部分更新：只改传入的字段，传 `null` 表示清空。
-  - **POST 和 PATCH 都返回改完之后的完整资源。** 例外：注册（`POST /api/v0/auth/register`）返回令牌，不返回新建的账户，账户由 `GET /api/v0/me` 读取（M2 设计 5.1）。
+  - **POST 和 PATCH 都返回改完之后的完整资源。** 例外：注册（`POST /api/v0/auth/register`）返回令牌，不返回新建的账户，账户由 `GET /api/v0/me` 读取；修改密码（`POST /api/v0/me/change-password`）和停用账户（`POST /api/v0/me/deactivate`）没有要返回的资源，返回 204（M2 设计 5.1）。
 - **接口描述**：放在 `api/`，按模块拆分（`api/modules/<模块>.yaml`，公共组件放在 `api/common.yaml`），打包为 `api/dist/openapi.yaml`。先改接口描述，再写实现。
 - **OpenAPI 版本：已定为 3.1**（`openapi: 3.1.0`）。M0/P3 验证了整条工具链，结论和写法约定见 [P3 spec](M0-foundation/specs/P3-api-contract.md) 2.2、2.4。
 
@@ -178,7 +178,7 @@
 - 依据：Plane 社区版前端的筛选器只会生成"多个条件全部满足"的组合，每个条件只有"等于 / 属于 / 范围"三种判断，上面的格式能完整表达。
 
 ### 3.4 分页与分组
-- **分页**：用游标。请求带 `?limit=50&cursor=...`，响应格式为 `{ "data": [...], "next_cursor": "..." }`。游标是不透明的字符串。
+- **分页**：用游标。请求带 `?limit=50&cursor=...`，响应格式为 `{ "data": [...], "next_cursor": "..." }`，最后一页的 `next_cursor` 是 `null`。游标是不透明的字符串：封套（版本号加载荷，base64url 编码）由 `internal/shared` 定义，载荷由各个列表按自己的排序定义，例如 PAT 列表的载荷是这一页最后一行的 `(created_at, id)`（M2 设计 3.12）。别的列表发的游标、篡改过的游标都是 400 `bad_request`；`limit` 超出 1–100 是 422 `validation_failed`。
 - **分组**：作为列表接口的可选参数，由服务端完成：
   ```
   GET /api/v0/projects/{id}/issues?group_by=state_id&sub_group_by=priority&limit=50
@@ -206,7 +206,7 @@
 ### 3.6 其他
 - **限流**：在进程内实现，按键的令牌桶，每个桶有速率和突发两个配置项（`ratelimit.<桶>.per_minute`、`burst`）。超出时 429 `rate_limited`，带 `Retry-After`（秒，向上取整）（M2 设计 3.10）：
   - 公开操作按客户端 IP 计数（`anonymous`，每分钟 600、突发 100），其余操作按凭证计数（`authenticated`，每分钟 1200、突发 200）；
-  - 登录另按 IP（`login_ip`，30、10）和"IP + 邮箱"（`login_ip_email`，10、5）计数，两个桶全扣或全不扣；注册另按 IP 计数（`register_ip`，10、5）；修改密码按账户计数（`password_user`，M2/P3）；
+  - 登录另按 IP（`login_ip`，30、10）和"IP + 邮箱"（`login_ip_email`，10、5）计数，两个桶全扣或全不扣；注册另按 IP 计数（`register_ip`，10、5）；修改密码按账户计数（`password_user`，5、5）；
   - **认证之前的失败闸门**：非公开操作带了令牌时，先预留本 IP 的一个单位（`auth_failure`，60、60），闸门已空就直接 429，不再验证令牌；认证失败时单位留下，成功、只是过期的访问令牌、内部错误时退回（M2 设计 3.6）；
   - 客户端 IP 取连接的对端，前面有反向代理时配置 `server.trusted_proxies`；IPv6 按前缀计数（`ratelimit.ipv6_prefix_len`，默认 64）。
 - **关联对象只返回 ID**：以后按需增加 `?expand=`。
@@ -227,10 +227,11 @@
 
 ### 4.2 规则
 - **权限不放进令牌**：每个请求都从数据库读取成员关系和角色。所以移出项目、调整角色是立即生效的。
-- **会话撤销**：退出登录只结束当前这一处登录，同一账户的其他会话不受影响（M2 设计 11.1）；修改密码、账户停用时，吊销该账户的刷新令牌；账户停用时，同时让该账户的所有会话失效。
+- **会话撤销**：退出登录只结束当前这一处登录，同一账户的其他会话不受影响（M2 设计 11.1）；修改密码结束该账户的其他会话，用 PAT 修改时没有当前会话，全部结束，PAT 不受影响；停用账户结束该账户的全部会话，PAT 不删除，但停用期间认证失败（M2 设计 3.5）。
+- **账户行锁**：签发和变更凭证的事务（登录、创建 PAT、修改密码、停用）先锁账户行，再确认调用者的凭证仍然有效，所以并发的修改不会让已被撤销的凭证再签发或变更凭证（M2 设计 3.5）。
 - **重复使用检测**：一旦发现某个已经换过新的刷新令牌又被使用，就作废这次登录派生出的所有令牌。只有交出的旧令牌确是这个会话签发过的（它的 MAC 标签成立）才作废；会话 id 和代数对、密文和标签是伪造的旧令牌得到 401，会话不受影响（M2 设计 3.5）。
 - **密码哈希**：用 argon2id。
-- **登录方式**：v0 只有邮箱加密码。是否开放注册由配置 `auth.signup_enabled` 控制：prod 默认关闭，dev、test 默认开放；关闭时注册先答"注册已关闭"，不查邮箱。prod 的第一个账户由服务器管理员用 `nerve users create` 创建（M2/P3 加入；在那之前用 `NERVE_AUTH__SIGNUP_ENABLED=true` 临时打开注册）（M2 设计决策点 2）。被邀请的邮箱始终可以注册。
+- **登录方式**：v0 只有邮箱加密码。是否开放注册由配置 `auth.signup_enabled` 控制：prod 默认关闭，dev、test 默认开放；关闭时注册先答"注册已关闭"，不查邮箱。prod 的第一个账户由服务器管理员用 `nerve users create` 创建（M2/P3b 加入；在那之前用 `NERVE_AUTH__SIGNUP_ENABLED=true` 临时打开注册）（M2 设计决策点 2）。被邀请的邮箱始终可以注册。
 - **忘记密码**：没有邮件服务，由服务器管理员通过命令行重置：`nerve users reset-password --email <email>`。
 
 ### 4.3 浏览器端
@@ -355,7 +356,7 @@
     platform/               与业务无关的技术基础件：config、postgres（连接池、事务管理器）、
                             logging、httpserver（中间件、problem+json）、ratelimit、clock、idgen
     shared/                 共享内核，尽量小，只放值会跨越模块边界的东西（M2 设计 3.3）：Actor（当前账户）、
-                            领域错误与错误码、TxManager 端口；以后加入分页游标的封套、领域事件接口、
+                            领域错误与错误码、TxManager 端口、分页游标的封套；以后加入领域事件接口、
                             Authorizer 端口。平台不导入它；时钟等其余端口由使用方的 app 层声明
     modules/
       identity/             账户、会话、PAT、密码
````

- [ ] **Step 2: M0 设计**（3.2 实例配置的三个字段；3.7 传递依赖测试的改写、生成代码的 uuid 规则）

`docs/v0/M0-foundation/M0-design.md`（对 `605f367` 的差异）：

````diff
--- a/docs/v0/M0-foundation/M0-design.md
+++ b/docs/v0/M0-foundation/M0-design.md
@@ -149,7 +149,7 @@
 
 ### 3.2 试点模块 `instance`
 - **接口**：`GET /api/v0/instance`，不需要登录。返回 `{ "product": "Nerve", "version": "0.1.0-dev", "commit": "…", "api_version": "v0" }`。
-- **后续扩展**：M2 加入 `signup_enabled`，M5 加入文件大小上限等。前端启动时用它读取公开配置，Agent 可以用它确认服务端的版本。
+- **后续扩展**：M2 加入 `signup_enabled`、`workspace_creation_enabled`、`file_size_limit`（M2 设计 5.3）。前端启动时用它读取公开配置，Agent 可以用它确认服务端的版本。
 - **目录结构**（这就是以后所有模块的模板）：
   ```
   modules/instance/
@@ -247,7 +247,8 @@
   8. 测试工具（`pgtest`、`apitest`）只能被测试代码导入。
   9. 模块内的包只能放在 `domain`、`app`、`adapter`（含子目录）或模块根目录（`module.go`）。放在别处的包，第 1 条排不出它的层次，第 2 条又把模块内的导入交给第 1 条判断，`domain → 模块内其他目录 → net/http` 就能两条都绕过（M0 对抗性评审 Important 1）。
   10. `internal/shared` 只能依赖标准库（不含 `net/http`、`database/sql`）和它自己：`domain`、`app` 可以导入它，它不干净，技术依赖就会经它带进这两层。M0 还没有 `internal/shared`，这条规则先用合成的导入关系测试。
-- **传递依赖测试**（`TestNerveBinaryLinksNoBannedModule`，M0/P3）：规则 8 只挡住测试工具包本身，挡不住生成的代码或其他途径间接引入的依赖（例如内嵌的接口描述、未映射的 `format: uuid`），depguard 也不检查生成的文件。这个测试用 `golang.org/x/tools/go/packages` 读取 `./cmd/nerve` 的全部传递依赖（不含测试），出现 `github.com/getkin/kin-openapi`、`github.com/testcontainers/`、`github.com/google/uuid`、`github.com/docker/` 开头的包就失败，并打印导入链。
+- **传递依赖测试**（`TestNerveBinaryLinksNoBannedModule`，M0/P3；M2/P3a 改写）：规则 8 只挡住测试工具包本身，挡不住生成的代码或其他途径间接引入的依赖（例如内嵌的接口描述），depguard 也不检查生成的文件。这个测试用 `golang.org/x/tools/go/packages` 读取 `./cmd/nerve` 的全部传递依赖（不含测试），出现 `github.com/getkin/kin-openapi`、`github.com/testcontainers/`、`github.com/docker/` 开头的包就失败，并打印导入链。`github.com/google/uuid` 只允许由 `github.com/oapi-codegen/runtime` 模块的包导入（它的参数绑定自己用），别的包导入它同样失败。
+- **生成代码的 uuid 规则**（`TestGeneratedCodeUsesTheStandardUUID`，M2/P3a）：禁止 google/uuid 的真实意图是"生成代码不漏 uuid 的映射"，所以直接检查：生成文件不得引用 `oapi-codegen/runtime/types` 的 `UUID`（不论导入时用什么名字；生成代码默认叫它 `openapi_types`）（M2 设计 3.12）。
 - **纯净性的传递检查**（`TestPureLayersReachNoInfrastructure`，M0 加固）：第 2、10 条只看直接导入，而标准库里的 `expvar`、`net/rpc` 自己就导入 `net/http`，`domain → expvar` 能过这两条，却把 `net/http` 链接了进来。这个测试沿全部传递依赖检查每个 `domain`、`app`、`internal/shared` 包：不能碰到 `net/http`、`database/sql` 或本模块以外的库。发现时打印完整的导入链。
 - **depguard**：只管"整个项目都禁止使用的库"，例如：
   - 第三方 uuid 库（`github.com/google/uuid`、`github.com/gofrs/uuid`、`github.com/satori/go.uuid`）：用标准库。
````

- [ ] **Step 3: M0/P3 spec**（2.8 规则 8 一段的补注；第 7 节 uuid 和分页组件两行）

`docs/v0/M0-foundation/specs/P3-api-contract.md`（对 `605f367` 的差异）：

```diff
--- a/docs/v0/M0-foundation/specs/P3-api-contract.md
+++ b/docs/v0/M0-foundation/specs/P3-api-contract.md
@@ -286,7 +286,7 @@
 | `modules/instance/adapter/http/handler_test.go` | `GET /api/v0/instance` 的响应符合契约，内容正确（M0 设计 3.8） |
 | `bootstrap/api_test.go` | 整个程序（含三个中间件）：instance 返回 200、`version` 等于 `buildinfo.Get().Version`、带 `X-Request-Id`；`GET /api/v0/nope`、`POST /api/v0/instance` 返回 404 problem+json。不需要数据库 |
 
-**archtest 规则 8** 改为"测试工具（`pgtest`、`apitest`）只能被测试导入"；规则表格补上相应的违规例子，以及 `gen` → `apigen` 这条合法的边。规则 8 只保证测试工具包本身不进生产程序，管不到别的途径：生成的代码（`embedded-spec: true`，或没有映射的 `format: uuid` 经 `oapi-codegen/runtime/types` 引入 `github.com/google/uuid`）或其他导入，而 depguard 不检查生成的文件。所以 archtest 另有 **`TestNerveBinaryLinksNoBannedModule`**：用 `packages.Load`（`NeedName | NeedImports | NeedDeps`，不含测试）读取 `./cmd/nerve` 的全部传递依赖，出现以 `github.com/getkin/kin-openapi`、`github.com/testcontainers/`、`github.com/google/uuid`、`github.com/docker/` 开头的包就失败，并给出一条导入链，例如 `cmd/nerve → internal/bootstrap → internal/platform/httpserver → github.com/getkin/kin-openapi/openapi3`。它与规则测试共用遍历源码目录的函数，新的导入会让缓存的结果失效。两者合起来保证：测试工具只在测试中使用，测试专用和禁用的模块不进 `nerve` 程序。
+**archtest 规则 8** 改为"测试工具（`pgtest`、`apitest`）只能被测试导入"；规则表格补上相应的违规例子，以及 `gen` → `apigen` 这条合法的边。规则 8 只保证测试工具包本身不进生产程序，管不到别的途径：生成的代码（`embedded-spec: true`，或没有映射的 `format: uuid` 经 `oapi-codegen/runtime/types` 引入 `github.com/google/uuid`）或其他导入，而 depguard 不检查生成的文件。所以 archtest 另有 **`TestNerveBinaryLinksNoBannedModule`**：用 `packages.Load`（`NeedName | NeedImports | NeedDeps`，不含测试）读取 `./cmd/nerve` 的全部传递依赖，出现以 `github.com/getkin/kin-openapi`、`github.com/testcontainers/`、`github.com/google/uuid`、`github.com/docker/` 开头的包就失败，并给出一条导入链，例如 `cmd/nerve → internal/bootstrap → internal/platform/httpserver → github.com/getkin/kin-openapi/openapi3`。它与规则测试共用遍历源码目录的函数，新的导入会让缓存的结果失效。两者合起来保证：测试工具只在测试中使用，测试专用和禁用的模块不进 `nerve` 程序。（M2/P3a 改写：`github.com/google/uuid` 只允许由 `oapi-codegen/runtime` 模块的包导入；没有映射的 `format: uuid` 由生成文件的规则 `TestGeneratedCodeUsesTheStandardUUID` 拦下：生成文件不得引用 `oapi-codegen/runtime/types` 的 `UUID`（不论导入时用什么名字；生成代码默认叫它 `openapi_types`）。见 M2 设计 3.12。）
 
 ### 2.9 TS 客户端：`web/packages/api-client`
 
@@ -408,7 +408,7 @@
 | M2 | 请求取值的校验放在哪一层（生成的代码不校验枚举、长度等），以及"参数校验错误"的错误码和 `errors` 字段；领域错误在 `ResponseErrorHandlerFunc` 中映射为 problem |
 | M2 | 错误出口的细节：请求体 JSON 解码失败时，`BadRequest` 的 `detail` 会带出 Go 的类型名（`json: cannot unmarshal … Go struct field IssueCreate.name …`），改为 `errors[]` 或通用的说明；`http.MaxBytesError` → 413；`context.Canceled` 不产生 500，也不记 ERROR 日志 |
 | M2 | problem 只走一条路：用 `apigen.Problem` 的 typed `default` 响应，或者 handler 返回错误再统一映射，二选一 |
-| M2 | `format: uuid`：用 `output-options.type-mapping` 映射到标准库 `uuid.UUID`，否则生成代码会用 `github.com/google/uuid`（2.2 已验证可行；漏掉时 archtest 的传递依赖测试失败） |
+| M2 | `format: uuid`：用 `output-options.type-mapping` 映射到标准库 `uuid.UUID`，否则生成代码会用 `github.com/google/uuid`（2.2 已验证可行；漏掉时生成代码引用 `openapi_types.UUID`，archtest 的生成文件规则失败，M2 设计 3.12） |
 | M2 | PATCH 的"传 `null` 清空"：`output-options.nullable-type: true`，生成 `nullable.Nullable[T]`（引入 `github.com/oapi-codegen/nullable`） |
 | M2 | 模块配置的模板选项一次定下、每个模块照抄：`nullable-type`、`prefer-skip-optional-pointer`、uuid 的 `type-mapping`（与上两行一起决定） |
 | M2 | 第一个带参数的接口会让生成代码导入 `github.com/oapi-codegen/runtime`（最新版 v1.7.0），写死版本 |
@@ -416,7 +416,7 @@
 | M2 | 模块入口的扩展：第二个模块出现前，把平台的 HTTP 依赖合成一个值传入；`httpadapter.Register` 接收一个用例结构体；生成的 `Middlewares` 按相反的顺序包装（最后一个在最外层）；其他模块要用的能力由模块导出访问方法 |
 | M2 | 接口描述的布局：多个模块共用的接口类型放在 `common.yaml`（模块文件之间互相 `$ref` 会违反 archtest 规则 3 和 6）；一个路径只属于一个模块文件；模块文件名与 Go 的模块目录名相同 |
 | M2 | 第一个带参数或请求体的模块，为每个错误出口写测试；可选：`apitest` 增加 `CheckRequest` |
-| M3 | 第一个列表接口把 `Limit`、`Cursor`（parameters）和 `NextCursor`（schema，`type: [string, 'null']`）加入 `api/common.yaml` |
+| M2 | 第一个列表接口把 `Limit`、`Cursor`（parameters）和 `NextCursor`（schema，`type: [string, 'null']`）加入 `api/common.yaml`（原定 M3；M2/P3a 随 `listApiTokens` 加入，M2 设计 3.12） |
 | P5 | oxfmt、oxlint 排除 `web/packages/api-client/src/schema.gen.ts` 和 `api/dist/`；`make lint-web` 改为 turbo 驱动，保留 api-client 的类型检查（脚本已按 Plane 的习惯叫 `check:types`）；`typescript` 改用 `catalog:`；合并 Plane 根目录的 `package.json` 时保留 `@redocly/cli` |
 | P6 | 端到端测试通过 `@nerve/api-client` 的 `createClient({ baseUrl })` 调用接口 |
 | P6 | `e2e` 任务的 `if` 和 `needs`：同仓 PR 跳过的任务也报告为成功，`e2e` 经 `needs` 继承这一点；将来把检查设为必需之前，去掉跳过条件或加一个 `if: always()` 的汇总任务 |
```

- [ ] **Step 4: 差异清单**（二：`api_tokens` 逐列；三：令牌的路径；四：密码规则、注册默认、限流三行改写，无效的 PAT、修改密码和停用之后的旧凭证、停用账户、PAT 的管理、`last_used`、名称和过期时间、编辑、时区八行新增）

`docs/v0/plane-diff.md`（对 `605f367` 的差异）：

```diff
--- a/docs/v0/plane-diff.md
+++ b/docs/v0/plane-diff.md
@@ -73,7 +73,7 @@
 | `users` | `password`：列名和类型照搬，内容改为 argon2id 的 PHC 字符串 | 密码哈希改用 argon2id（M2 设计 3.8） |
 | `users` | `first_name`、`last_name`：新加 `DEFAULT ''` | 模型的默认值 |
 | `users` | `display_name`：新加 `CHECK (display_name <> '')`；注册时取邮箱 @ 之前的部分 | Plane 由 `User.save()` 填上，数据库不约束 |
-| `users` | `user_timezone`：新加 `DEFAULT 'UTC'` | 模型的默认值（取值范围的差异由 M2/P3 登记在第四节） |
+| `users` | `user_timezone`：新加 `DEFAULT 'UTC'` | 模型的默认值（取值范围的差异见第四节） |
 | `users` | `is_active`：新加 `DEFAULT true`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 模型的默认值 |
 | `users` | 删除 Django 与管理后台的列 `last_login`、`is_superuser`、`is_staff`、`username`，以及与 `created_at` 重复的 `date_joined` | Nerve 没有 Django 的管理后台；`username` 前端从不读取 |
 | `users` | 删除登录记录 `last_login_time`、`last_logout_time`、`last_login_ip`、`last_logout_ip`、`last_login_medium`、`last_login_uagent`、`last_active`、`token`、`token_updated_at` | 改由 `auth_sessions` 承担 |
@@ -91,7 +91,14 @@
 | `profiles` | 删除 `role`、`use_case` | 新手引导的"角色""用途"两步已删除（M2 设计 3.19） |
 | `profiles` | 删除账单和移动端字段 `billing_address_country`、`billing_address`、`has_billing_address`、`company_name`、`is_mobile_onboarded`、`mobile_onboarding_step`、`mobile_timezone_auto_set` | Plane 云服务专用或已废弃 |
 | `profiles` | 删除 `is_smooth_cursor_enabled`、`is_app_rail_docked`、`background_color`、`goals`、`is_navigation_tour_completed`、`product_tour`、`notification_view_mode`、`has_marketing_email_consent`、`is_subscribed_to_changelog` | 前端不用的 Plane 新功能，以及营销邮件、更新日志 |
-| `api_tokens` | `token` 改为 `token_hash`；删除 `user_type` | 不存令牌原文；不区分人和机器人 |
+| `api_tokens` | 17 列保留 12 列（M2/P3a，`00004_identity_api_tokens.sql`） | M2 设计 4.4 |
+| `api_tokens` | `token`（令牌原文）改为 `token_hash bytea NOT NULL UNIQUE`：整个令牌（`nrv_pat_` 加 43 个字符）的 SHA-256，新加 `CHECK (octet_length(token_hash) = 32)`；令牌原文只在创建时返回一次 | 不存令牌原文 |
+| `api_tokens` | `label`：新加 `CHECK (label <> '')`；不传时由应用生成 32 位十六进制（Plane 的 `uuid4().hex`） | Plane 只在 Python 中生成 |
+| `api_tokens` | `description`：新加 `DEFAULT ''`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 模型的默认值 |
+| `api_tokens` | `user_id`：加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL` | 二·全局（模型的 `on_delete`） |
+| `api_tokens` | `expired_at`（空表示永不过期）、`last_used`、`deleted_at` 照搬；撤销就是软删除 | — |
+| `api_tokens` | 删除 `user_type`、`workspace_id`、`is_active`、`is_service`、`allowed_rate_limit` | 不区分人和机器人；Plane 自己已把个人令牌的 `workspace_id` 置空；撤销用软删除，`is_active` 没有独立的写入方；社区版不创建服务令牌；没有限流读取 `allowed_rate_limit` |
+| `api_tokens` | 索引 `api_tokens_user_id_created_at_idx ON (user_id, created_at DESC, id DESC) WHERE deleted_at IS NULL` | 列表的游标分页（M2 设计 3.12） |
 | `auth_sessions` | **新增**（M2/P1，`00003_identity_auth_sessions.sql`，替代 `sessions`，见一 B）：一次登录一行，12 列：`id`（访问令牌中的 `sid`）、`user_id`（`ON DELETE CASCADE`）、`token_hash`（当前一代刷新令牌密文的 SHA-256，32 字节）、`generation`（代数）、`user_agent`、`ip`（`inet`）、`expires_at`（登录时刻加会话期限，之后不变）、`last_refreshed_at`、`revoked_at`、`revoke_reason`（六个取值）、`created_at`、`updated_at`；`auth_sessions_revoked_consistent_check` 要求 `revoked_at` 与 `revoke_reason` 同时为空或同时有值；索引 `auth_sessions_user_id_idx`、`auth_sessions_expires_at_idx`。旧代的刷新令牌不存，由令牌里的 HMAC 标签认出 | JWT 认证；刷新令牌的轮换和重复使用检测（M2 设计 3.4、3.5、4.5） |
 | `workspaces` | 删除旧的 `logo` URL 列 | 遗留列 |
 | `workspace_members` | 删除 `view_props`、`default_props` | 遗留列 |
@@ -130,6 +137,7 @@
 | 无权限 | 403 | 看不到的资源返回 404，看得到但没权限返回 403 |
 | 错误码 | 接口描述中没有 | 每个操作在接口描述中用 `x-problem-codes` 声明它可能返回的错误码；字段错误带 `code`（M2 设计 3.11） |
 | 请求体 | DRF 的序列化器忽略未知字段 | 按契约拒绝未知字段、不合法的 `null` 和缺少的必填字段：400 `bad_request`，一次列出全部问题（M2 设计 3.11） |
+| 个人访问令牌的路径 | `/api/users/api-tokens/`、`/api/users/api-tokens/{id}/`（查看、修改、撤销） | 列出和创建是 `/api/v0/me/api-tokens`，撤销是 `DELETE /api/v0/api-tokens/{token_id}`：单个资源用短路径（v0-design 3.2）；不能查看单个令牌，也不能修改 |
 
 ---
 
@@ -148,11 +156,19 @@
 | 归档 | 工作项、迭代、模块、项目的归档与恢复，以及项目级自动归档 | 规则一致（见 v0-design 5.5）；归档和恢复改为 `POST .../archive` 和 `POST .../unarchive` 两个动作接口；列表通过 `?archived=true` 查询已归档的对象 |
 | 自动关闭 | 项目设置 `close_in` 后，长期未更新的未完成工作项会被自动关闭 | v0 不做 |
 | 忘记密码 | 发邮件重置 | 服务器管理员用命令行重置 |
-| 密码规则 | 服务端在注册、修改密码、重置命令三处用 zxcvbn 评分 ≥ 3；组合规则只在界面上 | 服务端执行组合规则（长度 8–128，至少一个大写字母、小写字母、数字和特殊字符）、NCSC 前 10 万常见密码名单和"主干不能是邮箱前缀的主干"（M2 设计 3.8）。M2/P1 在注册时执行；修改密码、创建账户和重置密码两个命令随 M2/P3 加入 |
+| 密码规则 | 服务端在注册、修改密码、重置命令三处用 zxcvbn 评分 ≥ 3；组合规则只在界面上 | 服务端执行组合规则（长度 8–128，至少一个大写字母、小写字母、数字和特殊字符）、NCSC 前 10 万常见密码名单和"主干不能是邮箱前缀的主干"（M2 设计 3.8）。M2/P1 在注册时执行，M2/P3a 在修改密码时执行；创建账户和重置密码两个命令随 M2/P3b 加入 |
 | 注册的前提 | 实例必须先由实例管理员完成设置 | 没有这一步 |
-| 注册默认是否开放 | `ENABLE_SIGNUP` 默认开放 | prod 默认关闭，dev、test 默认开放；关闭时先答"注册已关闭"，不查邮箱；第一个账户用 `nerve users create`（M2/P3 加入）（M2 设计决策点 2） |
+| 注册默认是否开放 | `ENABLE_SIGNUP` 默认开放 | prod 默认关闭，dev、test 默认开放；关闭时先答"注册已关闭"，不查邮箱；第一个账户用 `nerve users create`（M2/P3b 加入）（M2 设计决策点 2） |
 | 请求中的未知字段和不合法的 `null` | DRF 的序列化器忽略未知字段 | 按契约返回 400（M2 设计 3.11） |
 | 密码哈希过载 | 无并发上限 | 最多 4 个同时计算，等待 2 秒仍拿不到名额时 503 `server_busy`，带 `Retry-After: 1`（M2 设计 3.8） |
 | 登录时邮箱不存在 | 返回 `USER_DOES_NOT_EXIST` | 与密码错误相同的 401 `identity.invalid_credentials`，耗时也相同：对一个启动时生成的假哈希做一次同样参数的校验（M2 设计 3.9）；这只在存储的哈希都用当前参数时成立，调高 argon2 参数之后，休眠的账户再次登录之前能被耗时区分（M2 设计 §16） |
 | 会话的期限 | Django 会话，从登录起固定 7 天（`SESSION_COOKIE_AGE`），请求不延长 | 访问令牌 15 分钟；会话从登录起 30 天，续期不延长；刷新令牌每次使用后换新，并检测重复使用；退出只结束当前这一处登录（M2 设计 3.5） |
-| 限流 | 认证接口合计每 IP 10/min，匿名 30/min，API Key 60/min；`/api/v1` 的响应带 `X-RateLimit-Remaining`、`X-RateLimit-Reset`（`plane/apps/api/plane/api/views/base.py:120-126`） | 进程内的令牌桶，每个桶有速率和突发：匿名按 IP、已认证按凭证、登录按 IP 和"IP + 邮箱"、注册按 IP，认证之前另有按 IP 的失败闸门；超出时 429 带 `Retry-After`，不加 `X-RateLimit-*`（M2 设计 3.10）。修改密码按账户的桶随 M2/P3 加入 |
+| 限流 | 认证接口合计每 IP 10/min，匿名 30/min，API Key 60/min；`/api/v1` 的响应带 `X-RateLimit-Remaining`、`X-RateLimit-Reset`（`plane/apps/api/plane/api/views/base.py:120-126`） | 进程内的令牌桶，每个桶有速率和突发：匿名按 IP、已认证按凭证、登录按 IP 和"IP + 邮箱"、注册按 IP，认证之前另有按 IP 的失败闸门；超出时 429 带 `Retry-After`，不加 `X-RateLimit-*`（M2 设计 3.10）。修改密码另有按账户的桶 `password_user` |
+| 无效的个人访问令牌 | 403（`AuthenticationFailed` 没有 `authenticate_header`） | 401 `unauthorized` |
+| 修改密码、停用之后的旧凭证 | 其他会话在下一个请求时失效 | 相同，由每个请求的会话检查做到（M2 设计 3.5） |
+| 停用账户 | 自助停用：撤销会话、重置新手引导、把密码改成随机值、发邮件；"唯一管理员"的检查从不拒绝；只有命令 `activate_user` 能恢复 | 自助停用 `POST /api/v0/me/deactivate`：撤销全部会话，重置新手引导，不改密码，PAT 不删除但停用期间认证失败；管理员的 `deactivate`、`activate` 命令随 M2/P3b 加入；"唯一管理员"的检查由 M3 在同一个事务里实现（M2 设计决策点 3） |
+| 个人访问令牌的管理 | 只能用 Cookie 会话管理 | 任何凭证都能管理，包括 PAT 本身（v0-design 0.2 原则 2） |
+| 个人访问令牌的 `last_used` | 每个请求都写 | 每分钟最多写一次 |
+| 个人访问令牌的名称和过期时间 | 不校验：名称过长时变成 500，过期时间可以是过去 | 名称 1–255 个字符；过期时间必须在未来 |
+| 个人访问令牌的编辑 | `PATCH` 可以改名称和说明，响应中带令牌原文 | 不提供；令牌原文只在创建时返回一次 |
+| 时区 | 只接受 `pytz.common_timezones`；时区列表中负的非整点偏移多算一小时（例如马克萨斯群岛的 −09:30 写成 −10:30） | 接受 Go 的时区数据认得的任何 IANA 名称（`Local` 除外），程序内嵌时区数据；时区列表接口给的仍是同一份常用列表，偏移按请求时刻计算，写法正确（M2 设计 5.3） |
```

- [ ] **Step 5: README**（部署：注册一条补上账户枚举的说明；令牌的密钥扫描）

`README.md`（对 `605f367` 的差异）：

```diff
--- a/README.md
+++ b/README.md
@@ -105,7 +105,8 @@
 
   访问令牌由它签名，刷新令牌的 MAC 密钥也从它派生。换密钥（替换文件后重启）的后果：已签发的访问令牌验签失败，客户端续期一次即可；换钥之前的旧代刷新令牌不再能被认出是重复使用，所以刷新令牌正被盗用时换钥，受害者只是被登出（恢复靠修改密码或 `nerve users reset-password`）；同一出口 IP 后面的大量标签页同时续期，会短暂得到 429。所以在低峰时换。
 - **反向代理**：nerve 前面有反向代理（例如 Caddy）时，把代理的地址写进 `server.trusted_proxies`（CIDR 列表；环境变量用逗号分隔，例如 `NERVE_SERVER__TRUSTED_PROXIES=10.0.0.0/8,fd00::/8`）。只有连接的对端在这个列表中时，nerve 才从 `X-Forwarded-For` 自右向左取第一个不可信的地址作为客户端 IP。不配置时，所有请求都算作代理的地址：按 IP 的限流（匿名请求、登录、注册、认证失败）让所有人共用一份额度；nerve 第一次收到不可信对端带来的 `X-Forwarded-For` 时记一条 WARN 提醒。代理必须往 `X-Forwarded-For` 里写不带端口的 IP 地址：某一项带端口或是主机名时，nerve 在转发它的那个代理处停下，这个代理后面的客户端都算作代理的地址（第一次遇到时同样记一条 WARN）。只信任你自己的代理的地址：`0.0.0.0/0`、`::/0` 这样信任所有地址的前缀让任何客户端都能自己选 IP，启动时被拒绝。IPv6 客户端按前缀计数（`ratelimit.ipv6_prefix_len`，默认 64）。
-- **注册**：prod 默认关闭注册（`auth.signup_enabled: false`）。第一个账户用 `nerve users create` 创建（M2/P3 加入；在那之前临时设 `NERVE_AUTH__SIGNUP_ENABLED=true`）。
+- **注册**：prod 默认关闭注册（`auth.signup_enabled: false`）。第一个账户用 `nerve users create` 创建（M2/P3b 加入；在那之前临时设 `NERVE_AUTH__SIGNUP_ENABLED=true`）。开放注册时，注册接口会暴露一个邮箱是否已经注册：没有邮件通道，就无法让两种回答相同。注册按客户端 IP 限流（默认每分钟 10 次），只能压低探测的速度。登录不暴露账户是否存在，耗时也相同；关闭注册时，已注册和未注册的邮箱得到同一个 403。
+- **令牌的密钥扫描**：个人访问令牌以 `nrv_pat_` 开头，刷新令牌以 `nrv_rt_` 开头，但前缀不会让代码托管平台自动识别它们。要让平台发现提交里泄露的令牌，在它的密钥扫描中加自定义规则，例如 GitHub 仓库或组织设置的 Secret scanning → Custom patterns（需要平台提供这项功能）：个人访问令牌 `nrv_pat_[A-Za-z0-9_-]{43}`，刷新令牌 `nrv_rt_[A-Za-z0-9_-]{91}`。
 - **迁移**：prod 默认不在启动时迁移（`database.auto_migrate: false`），先执行 `nerve migrate up`，再 `nerve serve`。用单独的数据库角色执行迁移时，运行服务的角色除了读写业务表，还要能读 `goose_db_version`（`GRANT SELECT ON goose_db_version TO <服务的角色>`）：`/readyz` 靠它判断迁移是否已完成。
 
 ## Plane 表结构快照
```

- [ ] **Step 6: 交接**

`docs/v0/M2-auth/handoffs/M0-P3-api-codegen-notes.md`（对 `605f367` 的差异）：

```diff
--- a/docs/v0/M2-auth/handoffs/M0-P3-api-codegen-notes.md
+++ b/docs/v0/M2-auth/handoffs/M0-P3-api-codegen-notes.md
@@ -1,5 +1,5 @@
 ---
-status: open
+status: done
 from: M0/P3
 to: M2
 created: 2026-09-22
@@ -74,3 +74,12 @@
 仍未处理，状态保持 `open`：第 1 条中 google/uuid 的守卫，第 2 条中参数绑定的出口（`listApiTokens`、`revokeApiToken`），都在 M2/P3。
 
 来源：[M2/P1 spec](../specs/P1-platform-core.md) 第 7 节。
+
+## 处理结果（M2/P3a）
+
+1. **google/uuid 的守卫**（完成，M2 设计 3.12）：`github.com/oapi-codegen/runtime` v1.7.0 随第一批带参数的操作（`listApiTokens`、`revokeApiToken`）加入，`server/go.mod` 仍是 `go 1.27`、`toolchain go1.27.1`。它自己的参数绑定导入 google/uuid，所以传递依赖测试 `TestNerveBinaryLinksNoBannedModule` 只允许 `oapi-codegen/runtime` 模块的包导入 google/uuid，别的包导入仍然失败并打印导入链；"生成代码不漏 uuid 的映射"由新规则 `TestGeneratedCodeUsesTheStandardUUID` 直接检查：生成文件不得引用 `oapi-codegen/runtime/types` 的 `UUID`（不论导入时用什么名字；生成代码默认叫它 `openapi_types`）。
+2. **参数绑定的出口**（完成）：参数绑定失败由 `APIErrors.BadRequest` 答 400 `bad_request`，`errors` 里写出参数名（`code` 为 `invalid_format`，缺少必填参数时为 `required`），`detail` 不带出 Go 的类型名。identity 的 handler 测试（`limit=abc`、`token_id=not-a-uuid`）和整程序测试 5（从接口描述推出每个会拒绝某些字符串的参数）覆盖这个出口，并对 problem 做 `CheckResponse`。
+
+全部处理完，状态改为 `done`。
+
+来源：[M2/P3a spec](../specs/P3a-account-api.md) 第 7 节。
```

`docs/v0/M2-auth/handoffs/M0-P6-e2e-notes.md`（对 `605f367` 的差异）：

```diff
--- a/docs/v0/M2-auth/handoffs/M0-P6-e2e-notes.md
+++ b/docs/v0/M2-auth/handoffs/M0-P6-e2e-notes.md
@@ -69,3 +69,12 @@
 仍未处理，状态保持 `open`：PAT 对等验收和认证 fixture 的 PAT（M2/P3）；页面的登录状态（M2/P4）；S3 的 `signup_enabled`（M2/P3）；S2 的断言（M2/P4）；River 停机与 fixture 的预算（M2/P3）；fixture 写法的延伸（M4、M5、M8）。
 
 来源：[M2/P2 spec](../specs/P2-sessions.md) 第 7 节。
+
+## 处理结果（M2/P3a）
+
+- **PAT 对等验收和认证 fixture 的 PAT**（完成）：`e2e/fixtures/auth.ts` 加上 `createPAT` 和 `bearer`；A7–A11 的接口版本只用 PAT 调接口，调用 `e2e/fixtures/assert/identity.ts` 的断言函数（`accountOf`、`tokensOf`、`expectPasswordChanged`、`expectTokenStored`），页面版本以后调用同一组函数。
+- **S3**（完成）：`toEqual` 加上 `signup_enabled`、`workspace_creation_enabled`、`file_size_limit` 的期望值。
+
+仍未处理，状态保持 `open`：页面的登录状态（M2/P4）；S2 的断言（M2/P4）；River 停机与 fixture 的预算（M2/P3b）；fixture 写法的延伸（M4、M5、M8）。
+
+来源：[M2/P3a spec](../specs/P3a-account-api.md) 第 7 节。
```

`docs/v0/M2-auth/handoffs/M1-P2-trim-content.md`（对 `605f367` 的差异）：

```diff
--- a/docs/v0/M2-auth/handoffs/M1-P2-trim-content.md
+++ b/docs/v0/M2-auth/handoffs/M1-P2-trim-content.md
@@ -25,3 +25,11 @@
 逐项的结论写进该 M 的 review，然后 `status` 改为 `closed`。
 
 来源：[M1/P2 评审记录](../../M1-frontend-trim/reviews/P2-trim-content-review.md)第 7 节。
+
+## 处理结果（M2/P3a）
+
+- **接口描述**（完成）：`Profile` 和 `ProfileUpdate` 的 `theme` 是枚举 `Theme`，只有 `system`、`light`、`dark`、`light-contrast`、`dark-contrast` 五个值，没有 `custom`，也没有调色板字段；`profiles.theme` 的 CHECK 是同样五个值。
+
+仍未处理，状态保持 `open`：前端的 `IUserTheme` 改用生成的类型（M2/P4、P5）。
+
+来源：[M2/P3a spec](../specs/P3a-account-api.md) 第 7 节。
```

`docs/v0/M2-auth/handoffs/M1-P3-trim-platform.md`（对 `605f367` 的差异）：

```diff
--- a/docs/v0/M2-auth/handoffs/M1-P3-trim-platform.md
+++ b/docs/v0/M2-auth/handoffs/M1-P3-trim-platform.md
@@ -60,3 +60,14 @@
 逐项的结论写进该 M 的 review，然后 `status` 改为 `closed`。
 
 来源：[M1/P3 评审记录](../../M1-frontend-trim/reviews/P3-trim-platform-review.md)第 7 节。
+
+## 处理结果（M2/P3a）
+
+- **实例配置**（接口完成，M2 设计 5.3）：`InstanceInfo` 加上 `signup_enabled`（替代 `enable_signup`）、`workspace_creation_enabled`（`is_workspace_creation_disabled` 取反）、`file_size_limit`，取自配置文件；`is_self_managed` 不定义，新手引导的"角色""用途"两步随之删除（M2 设计 3.19，前端在 M2/P4）。
+- **不再读的用户字段、不再调用的地址**（接口完成）：两节列出的字段和地址都不出现在 `api/` 的接口描述中。
+- **令牌的地址**（完成）：`/api/v0/me/api-tokens`、`/api/v0/api-tokens/{token_id}`，结尾都不带 `/`。
+- **修改登录邮箱**：负责人已裁定只能由管理员用命令行修改（M2 设计决策点 1），命令 `nerve users set-email` 在 M2/P3b。
+
+仍未处理，状态保持 `open`：Cookie 会话和 CSRF、认证错误就地显示、前端改读新的实例字段（M2/P4）；`set-email` 命令（M2/P3b）。
+
+来源：[M2/P3a spec](../specs/P3a-account-api.md) 第 7 节。
```

- [ ] **Step 7: 检查**

Run: `make lint-web`
Expected: 通过（关键词守卫扫描改过的文档）。

Run: `grep -rn "M2/P3[^ab]" docs/v0/v0-design.md docs/v0/plane-diff.md README.md`
Expected: 没有输出（指向 P3 的地方都已写明 P3a 或 P3b）。

Run: `make test`
Expected: 全部 `ok`（文档不影响测试；确认工作区干净）。

- [ ] **Step 8: 提交**

```bash
git add docs/v0/v0-design.md docs/v0/M0-foundation docs/v0/plane-diff.md README.md docs/v0/M2-auth/handoffs
```
```bash
git commit -m "docs(M2/P3a): sync the design documents, plane-diff and README; record the handoff results

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** M2 设计 3.20 中 P3a 的各行和 8.7 中 P3a 的两行都已同步；四份交接追加了"处理结果（M2/P3a）"，M0-P3 为 `done`。
