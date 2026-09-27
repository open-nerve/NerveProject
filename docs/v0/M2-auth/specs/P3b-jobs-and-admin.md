# M2/P3b River 与管理命令：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M2/P3b `jobs-and-admin` |
| 日期 | 2026-09-26 |
| 状态 | 进行中 |
| 上级文档 | [M2 设计文档](../M2-design.md) 第 2（A12–A14、A16、A17）、3.5、3.8–3.10、3.15、3.17、3.20（P3b 各行）、4.1、4.6、5、6.4–6.6、8.5、8.7、9、12（P3b）、13.1 节，决策点 1–3；[v0 总体设计](../../v0-design.md) 4.2、5.2、6.7 节 |
| 前置交接 | [M0-P2-platform-notes](../handoffs/M0-P2-platform-notes.md) 第 5 条、[M0-P6-e2e-notes](../handoffs/M0-P6-e2e-notes.md) 的 River 停机一项、[M1-P3-trim-platform](../handoffs/M1-P3-trim-platform.md) 的修改登录邮箱；[P2 评审记录](../reviews/P2-sessions-review.md) 第 6 节和 [P3a 评审记录](../reviews/P3a-account-api-review.md) 第 6 节交给 P3b 的事项（本 Phase 的处理见第 7 节） |
| 计划 | [P3b plan](../plans/P3b-jobs-and-admin.md) |

本 spec 只写 M2 设计交给 P3b 决定的东西：名字、签名、SQL、配置项、测试名，以及原型证明了什么。规则本身以 M2 设计为准，这里引用节号，不重述。P3b 依赖 P3a（M2 设计 12 节）。

## 1. 目标

按 M2 设计 12 节 P3b 的目标：管理命令可用；River 的第一个定时任务运行；账户行锁的六个交错测试全部通过。具体是：

- 平台：`platform/jobs`（服务用的 River 客户端）；配置 `auth.session_cleanup_interval`、`jobs.shutdown_timeout`；River 的迁移 `00005`；
- `identity`：清理过期会话的用例、查询和 River worker；管理员的五个用例（创建、重置密码、改邮箱、按邮箱停用、恢复）和它们的查询；只用连接池的组合 `NewAdmin`；
- `bootstrap`：任务与 HTTP 一起运行，停机顺序 HTTP → 任务 → 迁移执行器 → 连接池，连接池关闭有时限；命令行的最小组合 `Users`；
- `cmd/nerve`：`nerve users create | reset-password | set-email | deactivate | activate`；
- 存的哈希不可用时登录的耗时不变（P2 评审第 6 节）；
- 测试：交错测试 1–3 在真实数据库上真的争锁；A12（含 `deactivate`、`activate`）、A13、A14、A16、A17 的接口版本；实测停机时间；
- 3.20 中 P3b 的各行、8.7 中 P3b 的三行、交接的处理记录。

## 2. 交付物

### 2.1 文件总览

路径相对于仓库根目录。"生成"表示由命令生成并提交，不手改。"Task"是 plan 中负责它的任务；几个 Task 号表示先写过渡版本、最后一个 Task 写成最终版本。

| 路径 | 内容 | Task |
|---|---|---|
| `server/internal/platform/config/`、`server/configs/` | 两个新键；`config.yaml` 中注册一条的注释 | 1、8 |
| `server/migrations/sql/00005_river_main_v2_to_v7.sql`（生成）、`server/migrations/schema_test.go` | River 的表 | 2 |
| `server/internal/archtest/sqlc_test.go`、`sqlc_cases_test.go` | `CREATE UNLOGGED TABLE` | 2 |
| `server/go.mod`、`server/go.sum` | River v0.47.0；`golang.org/x/term` 进 `require` | 3、8 |
| `server/internal/platform/jobs/` | 服务用的 River 客户端 | 3 |
| `server/internal/modules/identity/adapter/postgres/`（含 `queries/`） | 清理和管理员命令的查询、存储 | 4、6 |
| `server/internal/modules/identity/adapter/postgres/gen/`（生成） | | 4、6 |
| `server/internal/modules/identity/adapter/river/` | 清理任务的 worker 和定时任务 | 4 |
| `server/internal/modules/identity/app/` | 端口；清理用例；管理员的五个用例 | 4、6、7 |
| `server/internal/modules/identity/domain/` | `NewEmail`；两个撤销原因；两个模块错误 | 7 |
| `server/internal/modules/identity/module.go`、`admin.go` | `Jobs()`；`NewAdmin` | 4、8 |
| `server/internal/bootstrap/` | 运行和停机顺序；命令行的最小组合 | 5、8、9 |
| `server/cmd/nerve/` | `nerve users` | 5、8 |
| `api/modules/identity.yaml`；`api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`（生成） | 两个操作的说明写上命令 | 8 |
| `server/internal/modules/identity/adapter/argon2/` | 不可用的密码 | 9 |
| `server/internal/modules/identity/interleavings_test.go`、`interleavings_reset_test.go` | 交错测试 1–3 | 10 |
| `e2e/fixtures/`、`e2e/stories/identity/a{12,13,14,16,17}-*.spec.ts` | 命令的标准输入；五个故事 | 11 |
| `docs/…`、`README.md` | 3.20 的 P3b 各行、8.7 的 P3b 三行、交接的处理记录 | 12 |

### 2.2 依赖

- `github.com/riverqueue/river` v0.47.0 和 `github.com/riverqueue/river/riverdriver/riverpgxv5` v0.47.0（M2 设计 3.15、6.6）。`go -C server get` 两个模块再 `go -C server mod tidy`，另带进间接依赖 `riverdriver`、`rivershared`、`rivertype`（v0.47.0）和 `github.com/tidwall/{gjson v1.19.0, match v1.2.0, pretty v1.2.1, sjson v1.2.5}`。
- `golang.org/x/term` v0.46.0（6.6：x/crypto v0.57.0 所需的版本）本来就在 `go.sum` 里，`cmd/nerve/users.go` 导入它之后写进 `require`；`go.sum` 不变。
- River 的命令行 `github.com/riverqueue/river/cmd/river` v0.47.0 只在写迁移时用 `go run …@v0.47.0` 导出 SQL，不进任何 `go.mod`（6.6）。
- `server/go.mod` 仍是 `go 1.27` / `toolchain go1.27.1`；`server/tools/go.mod` 不变。不加 npm 包。

### 2.3 配置（M2 设计 3.15、6.5）

| 键 | 默认值 | test | 校验 |
|---|---|---|---|
| `auth.session_cleanup_interval` | `1h` | `2s` | 至少 1 秒：River 的定时任务"never less than one second"（`river@v0.47.0/periodic_job.go:12-14`） |
| `jobs.shutdown_timeout` | `10s` | — | 为正 |

- `Config.LogValue` 加上两项。`validConfig` 的两个值（`3h`、`12s`）不同于默认值、也不同于 `server.shutdown_timeout`，日志从错的字段取值时测试失败（附录 A）。
- 测试：`TestLoadAppliesLayersInOrder`（YAML 和两个环境变量）；`TestValidateReportsEveryInvalidKey`、`TestValidateCrossKeyRules`（500 毫秒的间隔、负的时限）；`TestLogValueMasksDatabaseURL`；`TestBuiltInProfiles`（三个环境）。

### 2.4 River 的迁移 `00005`（M2 设计 3.15、4.1；M0-P2 交接 5）

- 由锁定版本的命令行导出：`go run github.com/riverqueue/river/cmd/river@v0.47.0 migrate-get --line main --all --exclude-version 1 --up`（`--down`），得到第 2–7 版，up 之后留下 `river_job`、`river_leader`（`UNLOGGED`）、`river_queue`、`river_notification`，枚举 `river_job_state`，函数 `river_job_state_in_bitmask`（第 2 版建的 `river_job_notify` 由第 4 版删除，第 5 版建的 `river_client`、`river_client_queue` 由第 7 版删除；第 1 版只建 `river_migration`，客户端不读它，不导出）。导出与 M2 设计 spike 的文件逐字节相同（附录 A 的 E2）。
- 文件：6 行中文注释（来源、导出命令、为什么包起来、以后怎么升级），`-- +goose Up` 和 `-- +goose StatementBegin`，Up 段原样，`-- +goose StatementEnd`；Down 段同样包起来。521 行，SHA-256 `03e6c6a07d5a8a03addf7482d381f1106dcb53cb0f2164a84abd78e0ec7a9699`。它不在 `server/sqlc.yaml` 的任何 `schema` 中。
- `TestSQLCSchemaScope` 的建表正则加上可选的 `UNLOGGED`（第 3 节第 2 条）；`sqlc_cases_test.go` 的基准布局加上一个 River 式的迁移。
- `TestMigrationsGoUpDownAndUpAgain`：五个迁移 up 之后 River 的表、枚举和函数都在，down 之后一个都不剩，再 up 成功。

### 2.5 `platform/jobs`（M2 设计 3.15）

```go
type Job struct {
	Add      func(*river.Workers) error // registers the job's worker
	Periodic *river.PeriodicJob         // nil unless periodic
}
type Config struct {
	ShutdownTimeout time.Duration // jobs.shutdown_timeout
	Logger          *slog.Logger
}
func New(pool *pgxpool.Pool, cfg Config, jobs []Job) (*Runner, error)
func (r *Runner) Start(ctx context.Context)
func (r *Runner) Stop(ctx context.Context) error
```

- `New`：默认队列最多 2 个 worker（M2 只有一个定时任务）；定时任务交给 River，由 leader 投递，多台服务也只投一次；`SoftStopTimeout = ShutdownTimeout`；同一种任务的两个 worker 是错误。
- `Start` 在后台启动，立即返回。River 的 `Start` 先对数据库做一次 `SELECT 1`，数据库不可达时返回错误（`river@v0.47.0/client.go:1106-1115`；第 3 节第 1 条），runner 就间隔 1 秒、每次加倍、最多 30 秒地重试，每次记 WARN "jobs did not start; trying again"（`error`、`retry_in`）；启动后记 INFO "jobs started"（`shutdown_timeout`）。每次尝试把 `context.WithoutCancel(ctx)` 派生的一个新 ctx 交给 River 的 `Start`：传给 River `Start` 的 ctx 一取消，River 就开始停止（附录 A 的 E1），调用者的 ctx 结束不应让任务在 `Stop` 之前停下。`Stop` 只在这次 `Start` 还在运行时取消这个 ctx（`SELECT 1` 挂住也不挡停机）；`Start` 返回 nil 之后 runner 不再取消它，停止交给 River 的 `Stop`（第 3 节第 17 条）。`Stop` 恰好在 `Start` 返回时到达，River 收到的仍是普通的取消：打断进行中的 `Start` 避不开这个窗口。
- `Stop`：结束重试，取消还在进行的启动尝试的 ctx；客户端启动过时只调用 River 的 `Stop`，不先取消启动时的 ctx，River 用自己的停止原因（`startstop.ErrStop`）取消它的各个 ctx；期限是 `ShutdownTimeout` 加 1 秒（`cancelGrace`）：River 不再取任务，正在运行的任务有 `ShutdownTimeout` 结束，然后它们的 ctx 被取消；1 秒后还没返回就报错 `jobs still running 1s after jobs.shutdown_timeout (…)`，停机不被挂住。River 的 `Stop` 返回之后（成功或报错）释放启动时的 ctx。成功时记 INFO "jobs stopped"。
- 只导入 River 和 pgx，不导入别的平台包（archtest 规则 7；M2 设计 3.15）。
- 测试（真实数据库和假客户端）：`TestRunnerWorksAPeriodicJobUntilStopped`；`TestStopCancelsARunningJobAfterTheShutdownTimeout`（1 秒、3 秒两个设置）；`TestStopGivesUpOnAJobThatIgnoresCancellation`；`TestStartWithoutADatabase`；`TestNewRejectsTwoWorkersOfOneKind`；`TestStartTriesAgainUntilTheClientStarts`（等待 10、20、25 毫秒；调用者的 ctx 取消后客户端的 ctx 仍有效；`Stop` 的期限是时限加 1 秒）；`TestStopEndsTheAttemptsToStart`；`TestStopCancelsAnAttemptUnderWay`（挂住的 `Start` 被取消，`Stop` 不等它）；`TestStopLeavesTheStartedClientToItsOwnStop`（调用客户端的 `Stop` 时启动的 ctx 仍有效，`Stop` 之后被释放）。

### 2.6 清理过期会话（M2 设计 3.5、3.15）

```sql
-- name: DeleteExpiredSessions :execrows
DELETE FROM auth_sessions
WHERE id IN (
    SELECT s.id FROM auth_sessions s
    WHERE s.expires_at < sqlc.arg(now)
    LIMIT sqlc.arg(batch)
    FOR UPDATE SKIP LOCKED
);
```

- 子查询要用别名，不然 sqlc 认为 `expires_at` 有歧义。别的事务锁着的会话（续期、撤销）跳过，下一轮再删，清理不进入 3.5 的加锁顺序。
- 端口 `app.ExpiredSessionDeleter`；`Store.DeleteExpiredSessions(ctx, now, limit int) (int, error)`。
- `app.NewCleanupSessions(sessions, clock, logger)`：取一次时钟的 `now`，每批 1000 行，直到某一批不满；删掉了就记 INFO "expired sessions deleted"（`sessions`）；某一批失败时返回错误和之前各批的数目。
- `riveradapter`（`identity/adapter/river`）：`CleanupKind = "identity.cleanup_expired_sessions"`，也是定时任务的 id；`CleanupWorker.Work` 只调用用例，失败时由 River 重试；`CleanupJob(uc, interval)`：`river.AddWorkerSafely` 加 `NewPeriodicJob(PeriodicInterval(interval), …, &PeriodicJobOpts{ID: CleanupKind, RunOnStart: true})`。
- 模块入口：`Deps.SessionCleanupInterval`，`(*Module).Jobs() []jobs.Job`。
- 测试：真实数据库的 `TestDeleteExpiredSessions`（两个账户；三个过期的会话，其中一个已撤销、只早 1 微秒；四个留下的，其中一个恰好在 `now` 到期、一个已撤销；每批 2 行，三次调用得 2、1、0）、`TestDeleteExpiredSessionsSkipsLockedRows`（另一个事务持有一个过期会话的行锁时，清理在 `lock_timeout` 500 毫秒下不等待、跳过它；那个事务提交后下一次删掉它）；`TestCleanupSessionsDeletesBatchAfterBatch`（4 个）、`TestCleanupSessionsKeepsWhatExpiresFromNowOn`、`TestCleanupSessionsWhenABatchFails`；`TestCleanupKind`、`TestCleanupWorkerDeletesTheExpiredSessions`（真实数据库，调用 `Work()`：两个账户的过期会话删掉，未过期的留下）、`TestCleanupWorkerFailsWithTheCleanup`。

### 2.7 `bootstrap` 的运行和停机顺序（M2 设计 3.15；M0-P2 交接 5）

- `newApp` 把清理间隔交给 identity，`jobs.New(pool, jobs.Config{ShutdownTimeout: cfg.Jobs.ShutdownTimeout, Logger: logger}, ident.Jobs())`。
- `run`：需要时自动迁移 → `jobs.Start` → HTTP 直到 ctx 结束（HTTP 自己的优雅停机，M0）→ `jobs.Stop`，两个错误 `errors.Join`。HTTP 先停：没有请求还在投递任务时再停任务。
- `close`：迁移执行器 → 连接池；`pool.Close()` 在协程里，最多等 `poolCloseTimeout`（5 秒），等到记 INFO "database pool closed"，超时记 WARN "database pool not closed: connections still in use"（`waited`）。
- 数据库不可达时：任务在后台重试，HTTP 照常服务，`/readyz` 答 503（M0 的行为，`TestNotReadyWhenDatabaseIsUnavailable`、`TestNotReadyWithPendingMigrations` 不变而通过）。
- 测试：`TestTheSessionCleanupRunsAsConfigured`（两组设置：间隔 1 秒、时限 2 秒；间隔 1 小时、时限 3 秒。启动时运行一次；1 秒的在之后 3 秒内再运行，1 小时的不再运行；"jobs started" 带着各自的时限）；`TestCloseDoesNotWaitForAConnectionInUse`（时限 200 毫秒）；`TestRunStopsTheJobsAfterHTTP`（一个登录在账户行锁上等着时取消 `run`：HTTP 等它，"jobs stopped" 还没出现；放开锁，登录 200，任务随后停止）；`TestServeUntilCancelled`（`nerve serve` 等到 `river_job` 中有一个完成的任务；日志依次是 "http server stopped"、"jobs stopped"、"database pool closed"）。

### 2.8 管理员命令的存储（M2 设计 3.5、3.17）

```sql
-- name: LockAccountByEmail :one
SELECT id FROM users WHERE email = sqlc.arg(email) FOR NO KEY UPDATE;

-- name: RevokeAllAPITokens :execrows
UPDATE api_tokens
SET updated_at = sqlc.arg(now), deleted_at = sqlc.arg(now), updated_by_id = NULL
WHERE user_id = sqlc.arg(user_id) AND deleted_at IS NULL;

-- name: CountUsableAPITokens :one
SELECT count(*) FROM api_tokens
WHERE user_id = sqlc.arg(user_id) AND deleted_at IS NULL
  AND (expired_at IS NULL OR expired_at > sqlc.arg(now)::timestamptz);
```

- `LockAccountByEmail`：一条语句找到并锁住账户行，命令的事务里账户在两步之间不会变；端口 `app.AccountLocker{LockAccount}`，没有这个地址是 `app.ErrNotFound`。
- `ChangeEmail`（`email`、`updated_at`；`users_email_key` 冲突译为 `domain.ErrEmailTaken`，端口 `app.EmailChanger`）；`ActivateUser`（`is_active = true`、`updated_at`，端口 `app.UserActivator`）。
- `RevokeAllAPITokens`：过期的也撤销；已撤销的保留原来的撤销时刻和撤销者；**`updated_by_id` 为 NULL**（第 3 节第 5 条）。端口 `app.AllAPITokensRevoker`。
- `CountUsableAPITokens`：认证的判断，到期那一刻就不能用；`::timestamptz` 让 sqlc 生成 `time.Time` 而不是指针。端口 `app.UsableAPITokenCounter`。
- 测试（`admin_test.go`，真实数据库，每个测试另有一个账户证明 `WHERE` 只碰一个账户）：`TestLockAccountFindsTheAccountByAddress`；`TestLockAccountIsTheAccountRowLock`（持有它时，同一账户的 `LockForCredentials` 和 `LockAccount` 都在 `lock_timeout` 500 毫秒后失败；别的账户不等，插入这个账户的会话也不等）；`TestChangeEmail`；`TestActivateUser`；`TestRevokeAllAPITokens`；`TestCountUsableAPITokens`。

### 2.9 管理员的用例（M2 设计 3.5、3.17，决策点 1–3）

| 用例 | 依赖 | 做什么 | 结果 |
|---|---|---|---|
| `NewCreateUser` | `Rules, Hasher, Tx, Users, Profiles, Clock, Logger` | "建账户"一步：一次报出地址和密码的全部问题（422），在事务外哈希；事务里插入账户和默认资料；不经过 `SignupPolicy`，不建会话 | 规范化的地址 |
| `NewResetPassword` | `Accounts, Passwords, Sessions, APITokens, Hasher, Rules, Tx, Clock, Logger` | 按密码规则检查（带地址，查词干），事务外哈希；事务：按地址锁账户行 → 写哈希 → 撤销全部会话（`password_reset`）→ 撤销全部 PAT | `ResetPasswordResult{Email, Sessions, APITokens}` |
| `NewSetEmail` | `Accounts, Users, Sessions, Tx, Clock, Logger` | 新地址 `domain.NewEmail("new_email", …)`，规范化后与旧的相同是 `ErrEmailUnchanged`；事务：按旧地址锁账户行 → 改地址 → 撤销全部会话（`email_changed`）；PAT 不撤销 | `SetEmailResult{From, To, Sessions}` |
| `(*Deactivate).ExecuteByEmail` | `DeactivateDeps` 加上 `Accounts` | 事务：按地址锁账户行 → 与自助停用相同的三步（私有方法 `deactivate` 共用） | `DeactivateResult{Email, Sessions}` |
| `NewActivate` | `Accounts, Users, APITokens, Tx, Clock, Logger` | 事务：按地址锁账户行 → 恢复 → 数可用的 PAT | `ActivateResult{Email, APITokens}` |

- 地址一律按注册的规则规范化（`domain.NormalizeEmail`；Plane 的 `reset_password` 按原样匹配，4.6 的差异）。`lockAccount` 把 `app.ErrNotFound` 译为 `domain.ErrAccountNotFound`。
- "建账户"一步（`app.accounts{rules, hasher, clock, users, profiles}` 的 `prepare`、`create`）由注册和创建共用（3.17）；注册的测试不变而通过。
- 日志：INFO "account created"、"password reset"（两个数目）、"e-mail address changed"（不记两个地址，8.4）、"account deactivated"、"account activated"（`usable_api_tokens`），都带 `user_id` 和 `by: cli`（自助停用是 `by: self`，P3a）。
- `domain.NewEmail(field, email)`（与 `NewAccount` 共用 `checkEmail`）；`domain.RevokePasswordReset`、`RevokeEmailChanged`；`domain.ErrAccountNotFound`（`identity.account_not_found`）、`ErrEmailUnchanged`（`identity.email_unchanged`），只有命令行用（第 3 节第 4 条）。
- 测试：`TestNewEmail`；`create_user_test.go` 3 个；`reset_password_test.go` 4 个（调用顺序、一个事务、原因、两个数目；弱密码、常见密码、含词干的密码都不哈希不写；不认识的账户；写入失败）；`set_email_test.go` 3 个；`activate_test.go` 2 个；`TestDeactivateByEmail`、`TestDeactivateByEmailOfAnUnknownAccount`。假仓储 `fakeAdmin` 包着 P3a 的 `fakeCredentials`，只认 `alice@corp.com`，`bob@corp.com` 是被占用的地址，每次调用把参数记进调用记录。

### 2.10 命令行（M2 设计 3.17）

- **`identity.NewAdmin(AdminDeps{Pool, Tx, Clock, Logger, Password}) *Admin`**，`Admin{CreateUser, ResetPassword, SetEmail, Deactivate, Activate}`：只用连接池和密码哈希组合，不要签名密钥、限流和注册策略（第 3 节第 3 条）。
- **`bootstrap.Users(ctx, cfg, logOut, out, cmd UserCommand) error`**：日志、连接池、`TxManager`、`NewAdmin`；没有 HTTP，没有 River 客户端（M4 之前命令行不投递任务，13.2）。`UserCommand func(ctx, *identity.Admin) (string, error)` 返回输出的一行：

  | 命令 | 输出 |
  |---|---|
  | `CreateUser` | `created user <email>` |
  | `ResetPassword` | `password reset for <email>: revoked <n> sessions, <m> API tokens` |
  | `SetEmail` | `email changed from <旧> to <新>: revoked <n> sessions` |
  | `DeactivateUser` | `deactivated <email>: revoked <n> sessions` |
  | `ActivateUser` | `activated <email>: <m> API tokens are usable again` |

- **错误是一行**：领域错误的字段按命令行的名字写成 `<名字> <问题>`，多个用 `; ` 连接（`email` → `--email`，`new_email` → `--new-email`，`password` → `the password`）；没有字段时是错误的说明（例如 "An account with this e-mail address already exists."）；别的错误原样。`main` 打印 `nerve: <这一行>`，退出码 1，没有输出行，数据库不变。
- **`passwordHashing(config.PasswordConfig)`**：`newApp` 和 `Users` 共用，命令按 `auth.password` 的参数哈希。
- **`cmd/nerve`**：`nerve users` 下五个子命令，都要 `--email`，`set-email` 另要 `--new-email`；只输入 `nerve users` 打印帮助。`create`、`reset-password` 读密码：标准输入是终端时用 `golang.org/x/term` 不回显地提示两次，不同则失败；否则读一行，只去掉行尾的 `\n` 或 `\r\n`（最后一行没有换行也算），前后的空格保留；什么都读不到是 `read the password from standard input: EOF`；空行由密码规则报 `the password is required`。不设密码的三个命令不读标准输入。`run` 和 `newRootCommand` 多一个 `stdin` 参数。
- **契约说明**：`updateMe` 写明邮箱由 `nerve users set-email` 修改并结束全部会话；`deactivateMe` 写明 `nerve users activate` 恢复、之后未过期的 PAT 重新可用（P3a 评审第 6 节的复核）。只有说明文字变化，生成的 `api/dist/openapi.yaml`、`schema.gen.ts` 随之更新。
- `config.yaml` 注册一条的注释改为 `nerve users create --email <邮箱>`。
- 测试：`bootstrap/users_test.go` 的 `TestUsersCommands`（五个命令在同一个账户上依次运行，2 个会话、3 个 PAT，每步的地址、状态、会话原因、PAT 都核对；bob 始终不变；`river_job` 是空的）、`TestActivateCountsTheUsableTokens`、`TestUsersCommandErrors`（9 个）、`TestCommandErrorKeepsOtherErrors`；`cmd/nerve/users_test.go` 的 `TestUsersReadThePasswordFromStandardInput`（存的哈希是 test 配置的 `m=64,t=1,p=1`）、`TestUsersCommandsWithoutAPassword`、`TestUsersCommandsFail`（6 个）、`TestBareUsersPrintsHelp`。终端上的两次提示没有自动化测试（第 6 节）。

### 2.11 不可用的密码保持登录的耗时（M2 设计 3.9；P2 评审第 6 节）

- v0 不写"不可用的密码"：`create` 必须设密码（3.17 的表），没有别的形式。但存的哈希不是本 hasher 写的 argon2id PHC 字符串时（例如手工改库），原来 `Verify` 立即返回 `errNotOurHash`，登录变成很快的 500，能从耗时和状态码看出这个账户。
- 现在 `argon2adapter.Hasher` 在 `New` 时生成一个替身（当前参数、随机的盐和密钥）；`Verify` 遇到别的格式时对替身做一次同样的 argon2 计算，占用同样的名额（同样的等待、同样的 503），然后答"不匹配"、没有错误，并记 WARN "a stored password hash is not an argon2id PHC string: no password matches it"（不带哈希）。登录答 401，耗时与密码错误相同，只做一次校验（与假哈希的做法相当）。
- 测试：`TestVerifyMatchesNothingAgainstAnotherFormat`、`TestVerifyOfAnotherFormatIsBusyWhenNoSlotFreesUp`、`TestTheUnusableStandInHasTheCurrentParameters`；`TestLoginTakesAsLongForAnUnknownAddress` 加上第三种登录（密码列是 `!` 的账户），三种的中位数相差不到四分之一（原型：14.48、14.50、14.45 毫秒）。

### 2.12 账户行锁的交错测试 1–3（M2 设计 3.5、9.2；P3a 评审第 6 节）

`server/internal/modules/identity/interleavings_reset_test.go`（包 `identity_test`，真实数据库、真实的存储、事务和签名）：

- **真的争锁**：先走的一方停在**持锁的事务里**的闸门上，位置在锁住账户行之后、第一次写入之前。重置用 `gatedPasswords` 停在 `UPDATE users` 之前（这条 `UPDATE` 自己也会锁住账户行，闸门放在它之后就试不出显式的锁）；登录用 `gatedSessions` 停在插入会话之前，重新哈希时用 `gatedPasswords` 停在写新哈希之前；创建 PAT 用 `gatedTokens` 停在插入之前。`contend` 启动第二方，用 `pgtest.WaitForLockWait` 等到 Postgres 显示有语句在等锁，然后才打开闸门。每个等待都有 10 秒的期限，失败不挂住。
- **两个方向**各一个测试：

  | 交错 | 重置先持锁 | 另一方先持锁 |
  |---|---|---|
  | 1 登录 | `TestALoginWaitingForAResetFails`：登录等锁，之后看到重置的哈希，再校验一次失败（401），没有新会话；登录校验过的正是旧哈希和新哈希 | `TestAResetWaitsForALoginThatHoldsTheLock`：重置等登录提交，然后连同登录的新会话一起撤销（2 个） |
  | 2 重新哈希 | `TestALoginWaitingForAResetDoesNotWriteItsNewHash`：登录已算好新哈希，等锁；之后看到重置的哈希，失败，不写自己的哈希 | `TestAResetWaitsForALoginThatHashesThePasswordAgain`：重置等登录写完新哈希，然后覆盖它、撤销登录的会话 |
  | 3 创建 PAT | `TestATokenCreationWaitingForAResetFails`（用会话、用 PAT 两个子测试）：创建等锁，之后凭证已撤销，401，没有新 PAT | `TestAResetWaitsForATokenCreationThatHoldsTheLock`（两个子测试）：重置等创建提交，然后连同新 PAT 一起撤销 |

- 每个测试最后读库：存的哈希、未撤销的会话数、`password_reset` 的会话数、未撤销和已撤销的 PAT 数，以及重置返回的数目。
- `interleavings_test.go` 的 `login` 多一个 `passwords` 参数；`await` 的失败信息改为 "the use case did not finish"。
- 原型的核对（附录 A）：`-count=5` 和 `-race -count=5` 都通过；两个锁的变异（去掉 `FOR NO KEY UPDATE`、把锁挪到事务外）在两边（签发方和重置）都让 1–3 失败。

### 2.13 端到端（M2 设计 2、9.5；M0-P6 交接）

- `server.ts`：`runNerve(args, databaseUrl, input = "", env = {})`，标准输入写完就关闭（读密码的命令拿到文件结尾，不会一直等）；非零退出时拒绝，错误带退出码 `code`、`stdout`、`stderr`。9.5 的"`runNerve` 支持标准输入"。
- `users.ts`：`nerveUsers(db, args, input?)` 返回输出的一行，核对输出和日志里都没有密码；`nerveUsersFails(db, args, message, input?)` 核对退出码 1、没有输出、标准错误以 `nerve: <message>` 结尾。两者都带 `NERVE_LOG__LEVEL=debug`：test 配置的日志级别是 `warn`，命令的日志一行都不出，"日志里没有密码"就不可能失败（附录 A）。
- `assert/identity.ts`：`expectNewAccount`（A1、A17 共用）、`expectCreated`、`expectAllSessionsRevoked`、`expectNewPassword`（A7、A13 共用）、`expectDeactivated`、`expectPasswordReset`、`expectEmailChanged`。
- 故事（接口版本；A12、A13、A16 另有一个账户，核对命令没碰到它；A17 核对失败的命令什么都没加；A14 核对未过期的会话留下）：A12（先做完新手引导好让重置看得出来；PAT 自助停用 204 与数据库断言；PAT 401、登录 403；`activate` 的输出有 1 个 PAT；PAT 200、登录 200；再做完新手引导后 `deactivate` 的数据库结果与自助停用相同）；A13（两个会话、一个 PAT；输出 `revoked 2 sessions, 1 API tokens`；新密码能登录，旧密码、旧刷新令牌、旧 PAT 401）；A14（`expires_at` 改到过去，`expect.poll` 15 秒内被删掉，另一个会话仍在）；A16（改成别人的地址：退出码 1、数据库不变；改成新地址（大写）：已规范化、全部会话 `email_changed`、PAT 不变；旧地址 401、新地址 200、旧刷新令牌 401、PAT 读到新地址）；A17（被占用的地址、常见密码都是退出码 1，数据库不变；`create --email <大写>` 输出小写的地址；新账户能登录）。
- 新增的等待：A14 的轮询（15 秒为限）；`runNerve` 沿用 60 秒的上限。

### 2.14 文档、README 与交接（M2 设计 3.20、8.7、13.1）

plan 的 Task 12 逐行给出文字。要点：

- **总体设计 4.2**：会话撤销加上管理员命令的规则（重置撤销全部会话和 PAT，改邮箱撤销会话，`deactivate` 与自助停用相同，`activate` 后 PAT 重新可用）；账户行锁包括管理员的命令；`nerve users create` 去掉"M2/P3b 加入"；忘记密码的重置同时撤销全部会话和 PAT。
- **差异清单**：二·按表 River 的表；四 新增"管理员重置密码""重置密码命令的邮箱""修改登录邮箱""创建账户的命令"四行，改写"密码规则""注册默认""停用账户"三行。
- **README** 部署：第一个账户（`nerve users create --email`）；管理命令（五个命令、规范化、错误、密码的输入、`set-email` 和 `activate` 不是恢复手段）；令牌泄露后的恢复（8.5：逐个撤销 PAT，或 `nerve users reset-password`）。
- **交接**：M0-P2、M0-P6、M1-P3 追加"处理结果（M2/P3b）"，仍为 `open`（第 7 节）。

## 3. 与设计的差异和补充（请控制者裁定）

以下都没有改变 M2 设计的架构。

1. **River 的 `Start` 在数据库不可达时失败，任务在后台重试**：3.15 写"客户端的 `Start` 只做 `SELECT 1`，不检查迁移版本"。这没错，但 `SELECT 1` 失败时 `Start` 返回错误（`client.go:1106-1115`）。照直接接法，数据库暂时不可达时 `nerve serve` 会退出，M0 的行为（服务照常运行、`/readyz` 答 503，`TestNotReadyWhenDatabaseIsUnavailable` 守着）就变了。`platform/jobs` 的 `Start` 改为在后台重试（1 秒起、加倍、最多 30 秒），`Stop` 随时结束重试。River 自己在失败的 `Start` 之后允许再次 `Start`（附录 A 的 E1）。
2. **`TestSQLCSchemaScope` 认出 `CREATE UNLOGGED TABLE`**：River 第 5 版建的 `river_leader` 是 `UNLOGGED`，原来的正则不认，`ALTER TABLE river_leader` 就被当作"改别的文件建的表"。正则加上可选的 `UNLOGGED`，基准用例加上 River 式的迁移。
3. **`Admin` 由包级函数 `identity.NewAdmin(AdminDeps)` 组合，不是 `(*Module).Admin()`**：3.3 写模块入口导出 `Admin()`，3.17 写"连接池和 `identity` 的 `Admin()` 用例"。`Admin()` 若是 `Module` 的方法，命令行就要先 `New` 出整个模块：`New` 组合的是 HTTP 接口，要签名密钥（没有时生成临时密钥并警告）、访问令牌和会话的期限、限流的桶和注册策略，命令行一样都不用，却要为它们读配置、生成密钥。所以管理员的组合是模块入口的另一个构造函数（`identity/admin.go`），只要连接池、事务、时钟、日志和密码参数；名字仍是 `Admin`，仍在模块入口，`bootstrap` 组合、`cmd/nerve` 只解析参数，与 3.17 的分工相同。用例本身与接口共用（停用是同一个用例的两个入口）。M2 设计 3.3 的导出列表（`M2-design.md:164-165`、`:1105`）要随之改为 `NewAdmin(AdminDeps)`：这是设计文本的改动，请控制者裁定。
4. **两个只给命令行的模块错误**：`identity.account_not_found`（地址不对应任何账户）和 `identity.email_unchanged`（`set-email` 新旧相同，3.17 要求退出码 1）。5.4 的码表只列接口的码；这两个不进 `x-problem-codes`，命令行打印它们的说明。
5. **`reset-password` 撤销 PAT 时 `updated_by_id` 为 NULL**（P3a 评审第 6 节）：这次撤销不是任何账户做的。Plane 的 `BaseModel.save` 在没有登录用户时（例如管理命令）写 `updated_by = None`（`plane/apps/api/plane/db/models/base.py:31-33`），照此办理；用户自己撤销时仍写自己的 id（P3a）。`TestRevokeAllAPITokens` 核对；先撤销过的 PAT 保留原来的撤销者。
6. **不可用的密码由 hasher 处理**（P2 评审第 6 节）：v0 没有不设密码的建账户方式，所以没有"不可用的密码"这种写入；可能出现的只有库里被改成别的格式的哈希。hasher 对它做一次同样的工作、答"不匹配"并记 WARN，不再返回 `errNotOurHash`（2.11）。`errNotOurHash` 仍是 `parsePHC` 的错误。
7. **停止任务的期限多 1 秒**：3.15 写"River 停止（`jobs.shutdown_timeout`，默认 10 秒）"。River 在时限到期时取消任务的 ctx，任务还要一点时间返回；`Stop` 给它 1 秒（`cancelGrace`），过了就报错返回，不挂住停机。默认配置下停机最多约 HTTP 20 秒、任务 11 秒、连接池 5 秒（第 6 节）。
8. **"jobs started" 的日志带 `shutdown_timeout`**：`bootstrap` 把 `jobs.shutdown_timeout` 交给 River 这一步没有别的可观察的结果（River 的 `SoftStopTimeout` 只在停机时体现）；日志带上它，`TestTheSessionCleanupRunsAsConfigured` 用两个不同的设置核对接线（缺陷类别"接线没人看"）。
9. **默认队列最多 2 个 worker**：3.15 没有写。M2 只有一个每小时一次的任务；2 个给以后的第二种任务留一个位置，不占连接池（`database.max_conns`）太多。
10. **日志的 `by: cli`**：8.4 要区分"自助还是命令行"。P3a 的自助停用写 `by: self`；五个命令都写 `by: cli`。
11. **`DeactivateDeps` 多一个 `Accounts`**：同一个用例的两个入口各用一种锁：接口用 `Lock`（按调用者的凭证），命令行用 `Accounts`（按地址）。接口的组合不设 `Accounts`，命令行的组合不设 `Lock`；写入部分（`deactivate`）两者共用，保证"与自助停用相同的数据库结果"（A12）。
12. **交错 3 覆盖两种凭证**：9.2 写"用被并发重置撤销的凭证创建 PAT"。凭证是会话或 PAT 时账户行锁是同一把锁，但 `CredentialLock` 在锁下按凭证种类分两条路复核；只测会话时，"PAT 凭证不锁账户行"的变异不会被发现（附录 A，缺陷类别 E）。所以交错 3 的两个测试各有两个子测试。
13. **交错测试 1–3 放在新文件**：`interleavings_test.go` 已有 291 行，加上 1–3 会超过约 400 行的规则。新文件复用它的闸门、账户和登录帮助函数；`login` 帮助函数多一个 `passwords` 参数（交错 2 的闸门）。
14. **命令的密码只去掉行尾**：3.17 写"标准输入不是终端时读一行"。这一行只去掉 `\n` 或 `\r\n`，前后的空格保留（空格可以是密码的一部分）；空行交给密码规则（`the password is required`）；什么都读不到是错误，不当作空密码。
15. **e2e 的命令日志开到 DEBUG**：test 配置的日志级别是 `warn`，五个命令的日志都是 INFO，照原级别运行时"日志里没有密码"的断言不可能失败。`nerveUsers`、`nerveUsersFails` 设 `NERVE_LOG__LEVEL=debug`（附录 A）。
16. **`config/load_test.go` 到 403 行**：分层用例的表加两个键，超过约 400 行 3 行；拆开一张表反而难读，不拆（plan 的 Global Constraints 写明）。接口描述按模块一个文件，`identity.yaml` 超过 400 行，照 M0-P3 交接 5 不拆。
17. **Runner 在 River 启动后只用 `client.Stop` 停止它**：只在 River 的 `Start` 还在运行时取消它的 ctx（`SELECT 1` 挂住也不挡停机）；启动之后取消原因由 River 自己写为 `startstop.ErrStop`，River 的重建索引在停机时据此删掉没建完的 `_ccnew` 索引。代价：删除被长事务挡住时最多 15 秒，超过任务的停机时限，`Stop` 报错。附录 A 的 C1；控制者裁定 C1-a。

## 4. 验收标准（完成线，M2 设计 12 节 P3b）

- [ ] A12–A14、A16、A17 的接口版本通过，此前的故事仍然通过。
- [ ] 交错测试 1–3 在真实数据库上通过，`-count=5`、`-race -count=5` 也通过；六个交错测试至此全部通过。
- [ ] 清理任务的测试通过：只删除过期的会话，跳过被锁住的会话；5 个迁移都能 up、down、再 up。
- [ ] 实测的停机时间在端到端 fixture 的预算内，写进 review。
- [ ] 命令行的组合只有连接池和 `Admin`（`bootstrap.Users`；`TestUsersCommands` 核对 `river_job` 是空的）。
- [ ] 存的哈希不可用时登录答 401，耗时与密码错误相同（`TestLoginTakesAsLongForAnUnknownAddress`）。
- [ ] `make lint`、`make test`、`make gen-check`、`make knip`、`make e2e` 通过。
- [ ] 3.20 中 P3b 的各行、8.7 中 P3b 的三行在同一次合并中写好；交接按第 7 节处理。

## 5. 不在 P3b 范围内

- 命令行的"只投递"River 客户端：M4，随第一个投递任务的命令加入（负责人 2026-09-26 批准，M2 设计 13.2）。
- A12 的页面版本（P5）；安全页的 PAT 列表（P5）。
- `nerve users` 的其余命令（列出账户、删除账户）：3.17 只有五个。

## 6. 风险

| 风险 | 应对 |
|---|---|
| 终端上的密码提示（两次、不回显）没有自动化测试：需要伪终端 | 代码短、只用 `term.IsTerminal` 和 `term.ReadPassword`；两次不同时失败。人工核对一次；标准输入的路径由单元测试和 A13、A17 覆盖 |
| 按配置的最坏停机时间约 36 秒（HTTP 20 秒、任务 10 秒加 1 秒、连接池 5 秒），超过 fixture 的 `stopTimeoutMs`（30 秒） | 实测只有几毫秒（附录 A）。真到最坏情况时 fixture 在 30 秒 SIGKILL 并报出日志路径，不会挂住；停机要这么久本身就是缺陷，应当失败 |
| 停机时"迁移执行器先于连接池关闭"没有测试能看出来 | 两者都只在进程退出前运行一次；顺序写在 `close` 的注释和代码里。把两步对调的变异没有被发现（附录 A），记在这里 |
| River 的重试把数据库故障藏在 WARN 里 | `/readyz` 照常报告数据库；每次重试都记 WARN，间隔最多 30 秒 |
| A14 依赖真实的时间（2 秒的间隔） | 轮询以 15 秒为限；原型中 A14 的耗时 0.9–4.9 秒（附录 A），重复 5 次都通过 |
| 管理员命令与在线的服务并发 | 命令按 3.5 的账户行锁执行，交错测试 1–3 证明它与登录、创建 PAT 正确地互相等待；清理任务用 `SKIP LOCKED`，不参与加锁顺序 |
| 停机落在 River 启动后的最初几秒内时，River 记一两条 ERROR | River 对周期任务投递的 `Begin` 失败、通知连接上被打断的语句不看取消的原因（`periodic_job_enqueuer.go:529-532`、`notifier.go:146-151`），runner 无从避免；实测就绪后立即停机 10/10、3 秒后 7/40，运行 10 秒、30 秒后 0/40；没有任务停在 running、没有连接泄漏、没有写一半的数据（附录 A 的 C1）；README 部署一节说明（Task 12） |
| 停机恰好落在 River 的重建索引中（默认每天 00:00 UTC），又有访问过 `river_job` 的长事务挡住没建完的 `_ccnew` 索引的删除时：任务的停止在 `jobs.shutdown_timeout` 加 1 秒（默认 11 秒）时放弃，nerve 记 ERROR（`jobs still running …`）并以退出码 1 退出；River 自己的这次删除另有 15 秒的时限（`reindexer.go:263-266`），连接池的关闭等它占用的连接（默认配置下 11 秒加连接池的 5 秒，长于这 15 秒）；所以只有删除被挡满 River 的 15 秒时，索引才留下 | 恢复方法写进 README 部署一节（Task 12）：日志出现 WARN `maintenance.Reindexer: Found reindex artifact … skipping reindex` 时，用 `DROP INDEX CONCURRENTLY` 删掉 `artifact_names` 中列出的索引。改之前的 runner 在这样的停机中无论有没有长事务都留下索引，从此不再重建它（附录 A 的 C1） |

## 7. 交接的处理

| 交接 | P3b 处理的条目 | 留下的条目 | 状态 |
|---|---|---|---|
| M0-P2-platform-notes | 5：River、停机顺序、连接池关闭的时限、River 的迁移（Task 2、3、5） | 2 的接口调用日志（M8） | open |
| M0-P6-e2e-notes | River 停机与 fixture 的预算（Task 11，附录 A 实测）；`runNerve` 的标准输入 | 页面的登录状态、S2（P4）；fixture 写法的延伸（M4、M5、M8） | open |
| M1-P3-trim-platform | 修改登录邮箱：`nerve users set-email`（Task 7、8，A16） | Cookie 会话和 CSRF、认证错误、前端改读实例字段（P4） | open |

评审交给 P3b 的事项：

| 事项 | 落在 |
|---|---|
| 交错 1–3 真的在真实数据库上争锁；闸门在持锁的事务里；`WaitForLockWait`；去掉锁的子句、把锁挪到事务外时都失败（P3a 评审第 6 节） | Task 10（2.12；附录 A 的变异表） |
| 管理员撤销全部 PAT 时 `updated_by_id` 写什么（P3a 评审第 6 节） | Task 6：NULL（第 3 节第 5 条） |
| `nerve users activate` 有了之后复核 `deactivateMe` 的说明（P3a 评审第 6 节） | Task 8：写上命令和"之后未过期的 PAT 重新可用" |
| README 8.5 的恢复一条；A12 的端到端（P3a 评审第 6 节） | Task 12、Task 11 |
| 任何"不可用的密码"都保持登录的耗时（P2 评审第 6 节） | Task 9（2.11；第 3 节第 6 条） |
| 接口描述按模块一个文件，不受约 400 行的限制（P3a 评审第 6 节） | plan 的 Global Constraints |

## 附录 A：原型验证记录（2026-09-26）

原型在 `$M2TMP/p3bproto`（`d6f313b` 的副本，Go 1.27.1、Node 24.15.0、pnpm 11.10.0、Docker 29.7.2）。plan 中的代码就是原型中运行过的代码，由脚本从原型文件原样拼入 plan（新文件和改动大的文件给完整内容，改动小的给对前一个版本的统一差异；拼接脚本用 `patch -p1` 把 48 个差异块逐个应用到前一个版本上，结果都与原型的文件逐字节相同；plan 提交前重新拼了一次，与提交的 plan 逐字节相同）。

**逐 Task 复现**：在 `$M2TMP/p3bstage`（`d6f313b` 的另一个副本）按 plan 的 12 个 Task 依次执行：放入这个 Task 写的文件（最终版本或过渡版本，即 plan 中的内容）；执行 plan 中的命令（Task 2 的导出命令从 plan 的文本中原样取出执行；Task 3、8 的 `go get` 和 `go mod tidy`，之后核对 `go`、`toolchain` 四行；Task 4、6、8 的 `make gen`，生成物与 plan 表中的 SHA-256 所对应的文件逐字节相同）；然后 `make lint-go`（每个 Task 两段都是 `0 issues.`）和 `make test`（Task 1、2 各 30 个包，Task 3 31 个，Task 4 起 32 个，都没有失败）。Task 8、11 另跑前端检查、`make knip` 和 `make e2e`，Task 12 跑关键词守卫。每个 Task 13–64 秒。12 个 Task 之后，复现的目录与原型逐文件相同（2608 个文件，不含构建产物）。

| 核对 | 命令 | 结果 |
|---|---|---|
| 生成物 | Task 4、6、8 的 `make gen`（副本不是 git 仓库，`make gen-check` 用不了，改为与 plan 表中的 SHA-256 对应的文件比较） | 5 个生成的文件逐字节相同 |
| Go 静态检查 | `make lint-go`（每个 Task） | server 0 issues；server/tools 0 issues |
| Go 测试 | `make test`（每个 Task；testcontainers，`postgres:18.6`） | 全部通过（最终 32 个包） |
| 前端检查 | `turbo run check:types check:lint check:format check:sync`（54 个任务）；`make knip`；关键词守卫（遍历目录的等价脚本，2657 个文件，5 处命中都有例外） | 全部通过 |
| 端到端 | `make e2e`（Task 8、11） | Task 11 之后 22 个测试中 21 个通过（S1、S2 两个、S4、A1–A17）。S3 失败的原因与 P1–P3a 相同：副本不是 git 仓库，`commit` 是 `unknown` |
| P3b 的故事重复 | `playwright test` 五个新故事 `--repeat-each=5` | 25 个全部通过（9 个 worker，6.2 秒） |
| 交错测试重复 | `go test -count=5 -run 'TestA\|TestTwo' ./internal/modules/identity/`；同样加 `-race` | 都通过：9 个测试函数、4 个子测试各 5 次，没有 `DATA RACE`（5.2 秒；`-race` 7.2 秒） |
| 迁移 | `TestMigrationsGoUpDownAndUpAgain`（五个迁移） | up、down、再 up 通过；down 之后 River 的表、枚举、函数一个不剩 |
| 依赖 | 从 `d6f313b` 的 `go.mod`、`go.sum` 出发执行 plan 的 `go get` 和 `go mod tidy`（Task 3、8） | 结果与 plan 的差异逐字节相同 |
| 迁移的导出 | plan 中 Task 2 的命令块 | 文件与原型逐字节相同，SHA-256 与 plan 的表相同 |

**原型中定下的事实**：

- **E1** River v0.47.0 的客户端：`Start` 先执行 `SELECT 1`，数据库不可达时返回 `error making initial connection to database`（`client.go:1106-1115`，`TestStartWithoutADatabase` 实测）；失败的路径 `defer stopped()` 复位启停状态，所以可以再次 `Start`（`client.go:1180-1185`，读代码得出；runner 的重试循环由假客户端测试）；传给 `Start` 的 ctx 被取消时客户端开始停止，所以 runner 给它 `context.WithoutCancel` 派生的 ctx，只由 `Stop` 停止；`SoftStopTimeout` 到期时取消正在运行的任务的 ctx；River 自己记 "River client started"、"River client stopped"。
- **E2** 迁移的导出：`go run github.com/riverqueue/river/cmd/river@v0.47.0 migrate-get --line main --all --exclude-version 1 --up`（`--down`）的输出与 M2 设计 spike 的 `$M2TMP/spikes/river/migrations/river_all.up.sql`、`river_all.down.sql` 逐字节相同（318 行、189 行，SHA-256 `d77988487ddf…a3f316`、`6fc8d83c43c6…0772f1b`）；加上 14 行注释和 goose 标记，共 521 行。
- **E3** `http.Server.Shutdown` 会立即关掉还没读出请求的新连接：停机顺序的测试若只发半个请求，连接在停机时被关掉，看不出 HTTP 是否等请求。`TestRunStopsTheJobsAfterHTTP` 改用一个真正在处理中的登录（另一个事务锁着账户行，`WaitForLockWait` 确认登录在等锁）。
- **E4** 登录的耗时（`argon2_memory_kib` 19456、`argon2_iterations` 2，三种登录交替各 15 次）：中位数 已知地址错误密码 14.48 毫秒、未知地址 14.50 毫秒、不可用的密码 14.45 毫秒。
- **E5** 停机时间（`bin/nerve serve`，test 配置加 `NERVE_LOG__LEVEL=info`，开发库上的一个独立的库，各 10 次）：就绪后立即 SIGTERM，退出用时最小 2、中位 2、最大 2 毫秒；就绪 3 秒后（清理任务已运行）最小 4、中位 6、最大 7 毫秒；每次日志都有 "jobs started" 和 "jobs stopped"。fixture 的 `stopTimeoutMs` 是 30 秒，worker 的预算 `nerveFixtureTimeoutMs = readyTimeoutMs + stopTimeoutMs + 10_000` 是 70 秒。
- **E6** A14 的耗时：第一次全量运行 4.9 秒（worker 的 nerve 刚启动，River 选出 leader 之后才投递定时任务），重复 5 次时 0.9–2.9 秒。
- **C1**（Task 11 之后的调查，2026-09-27；控制者裁定 C1-a）停机时 River 记的 ERROR。`bin/nerve serve`，test 配置加 `NERVE_LOG__LEVEL=info`，开发库上每组一个独立的库，组内顺序运行；表中是至少有一条 River ERROR 的运行数（没有一次出现 WARN）。"改后"是第 3 节第 17 条的 runner。

  | SIGTERM | 清理间隔 | 原 runner | 改后 |
  |---|---|---|---|
  | 就绪后立即 | 2 秒（test） | 10/10 | 10/10 |
  | 就绪 3 秒后 | 2 秒 | 7/40 | 4/40 |
  | 就绪 10 秒、30 秒后 | 2 秒 | 0/20 | — |
  | 就绪 10 秒、30 秒后 | 1h（默认） | 0/20 | — |
  | 就绪 10 秒后，`river_queue` 从 7 秒起被锁 | 2 秒 | 1/10 | 0/10 |

  210 次运行（原 runner 150 次，其中 50 次是 debug 日志级别的诊断运行、不在表中；改后 60 次）都以 0 退出，都有 "jobs stopped" 和 "database pool closed"，没有任务停在 `running`，`river_leader` 为空，PostgreSQL 日志中没有断开的连接或被取消的语句。出现的只有两种 ERROR：`maintenance.PeriodicJobEnqueuer: Error starting transaction`（`periodic_job_enqueuer.go:529-532`：leader 的维护服务逐个启动，每个先随机等 0–1 秒（`queue_maintainer.go:50-55`、`river_shared_maintenance.go:133`），停机落在周期任务的第一次投递之前时，这次投递在已取消的 ctx 上执行；运行中只在停机恰好落在一次投递的几毫秒内时出现）；`notifier.Notifier: Error running listener … conn closed`（`notifier.go:146-151`：停机落在 River 自己的 `Start` 之中，打断了通知连接上的 `LISTEN`，pgx 关掉这个连接）。两处都不看取消的原因，与 runner 怎样停止无关，3 秒一行的差别是噪声（Fisher 检验 p = 0.52）。

  取消原因有影响的是 River 的重建索引（默认每天 00:00 UTC，`river_job` 的 7 个索引）：停机中断 `REINDEX INDEX CONCURRENTLY` 时，只有取消原因是 `startstop.ErrStop`，它才删掉没建完的 `_ccnew` 索引（`internal/maintenance/reindexer.go:263`；删除用 `context.WithoutCancel` 加 15 秒的时限，`:264-266`）。原 runner 先取消 River 启动时的 ctx，原因成了 `context.Canceled`。实验：诊断用的二进制把 `ReindexerSchedule` 设为 3 秒；另一个会话开着读 `river_job` 的事务，让 REINDEX 停在建好 `_ccnew` 之后；这时 SIGTERM，0.5 秒后结束那个事务。原 runner 5/5 留下 INVALID 的 `river_job_args_index_ccnew`，再启动后每次重建都记 WARN "Found reindex artifact … skipping reindex"（`reindexer.go:244`），这个索引不再重建；改后 5/5 删掉它，SIGTERM 后约 0.6 秒以 0 退出，再启动没有 WARN。代价：挡住删除的事务一直不结束时，River 等满 15 秒；`Stop` 在 `jobs.shutdown_timeout` 加 1 秒（默认 11 秒）时报错，连接池等删除占用的连接释放后关闭，nerve 以退出码 1 退出，索引留下（观察到一次，15.0 秒）。

**变异核对**（`$M2TMP/p3btools/muts.py`：改一处代码，跑相关的包，恢复；端到端的变异由 `muts_e2e.py` 重新构建 `bin/nerve` 后跑一个故事）。共 123 个变异，按 P3a 评审列出的缺陷类别归类；每个变异按最后一次运行计，122 个被发现，1 个没有（见下）：

| Task | A | C | D | E | F | H | I | K | 合计 |
|---|---|---|---|---|---|---|---|---|---|
| 1 配置 | 4 | | | | | | | 5 | 9 |
| 2 迁移 | 5 | | | | | | | | 5 |
| 3 `platform/jobs` | 8 | | | | | 2 | | 2 | 12 |
| 4 清理 | 11 | 1 | 1 | | | | 2 | | 15 |
| 5 运行和停机 | 5 | | | | | 1 | | 6 | 12 |
| 6 管理员的存储 | 7 | | 5 | | | | 2 | | 14 |
| 7 管理员的用例 | 21 | | | | | | | | 21 |
| 8 命令行 | 12 | | | | | | | 1 | 13 |
| 9 不可用的密码 | 5 | | | | | | | | 5 |
| 10 交错测试 | 5 | | | 1 | | | | | 6 |
| 11 端到端 | 7 | | 1 | | 2 | | | 1 | 11 |
| 合计 | 90 | 1 | 7 | 1 | 2 | 3 | 4 | 15 | 123 |

类别：A 属性去掉了测试仍通过；C 假实现忽略参数；D 只有一行或一个账户，看不出缺了 `WHERE`；E 对一种凭证对、对另一种错的键或锁；F 日志里的密文（这里是密码）；H 测试挂住而不是失败；I 并发测试的闸门在争用的区段之外；K 接线没人看。B（断言不可能失败）、G（读记录器上活的头部映射）、J（说明与代码不符）、L（时间按请求原样回显）见下面的类别表。

有代表性的几个：

| 变异 | 结果 |
|---|---|
| 签发方的锁去掉 `FOR NO KEY UPDATE`（生成的 `LockUserForCredentials`） | 交错 1–3 的 6 个测试全部失败：重置先持锁时，交错 2 的登录把自己的哈希写在重置的哈希上面、留下一个未撤销的会话（登录只校验了一次）；另外几个在 10 秒内没有语句等锁 |
| 重置的锁去掉 `FOR NO KEY UPDATE`（`LockAccountByEmail`） | 重置先持锁的三个测试失败（含交错 3 的两个子测试） |
| 登录、创建 PAT、重置的锁挪到事务之外（先锁、再开事务） | 登录：另一方先持锁的交错 1、2 失败；创建 PAT：交错 3 的另一方向两个子测试失败；重置：重置先持锁的三个测试失败 |
| PAT 凭证不锁账户行（`CredentialLock` 对 PAT 直接复核） | 交错 3 的"用 PAT"两个子测试失败（E 类；只测会话时不会被发现） |
| 清理不用 `SKIP LOCKED` / 不加行锁 | `TestDeleteExpiredSessionsSkipsLockedRows` 失败 |
| 清理只删一批 / 删到 `now` 为止（`<=`） / 每批 100 行 | `TestCleanupSessionsDeletesBatchAfterBatch`、`TestDeleteExpiredSessions` 失败 |
| `Stop` 不给 1 秒 / 不限时 / 客户端随调用者的 ctx 停止 / 重试不加倍、不封顶 | `platform/jobs` 的对应测试失败；不限时的变异在期限后失败，不挂住 |
| `bootstrap` 不接清理间隔 / 不接停止时限 / 用 `server.shutdown_timeout` / 模块不导出任务 / 不 `RunOnStart` | `TestTheSessionCleanupRunsAsConfigured` 的对应设置失败 |
| 任务与 HTTP 一起停（不等 HTTP 停完） | `TestRunStopsTheJobsAfterHTTP` 失败 |
| 连接池关闭不限时 / 时限写死 | `TestCloseDoesNotWaitForAConnectionInUse` 失败 |
| 重置不撤销 PAT / 留一个会话 / 原因写错 / 地址不规范化 / 密码规则不带地址 | `TestResetPassword`、`TestResetPasswordChecksTheNewPassword` 失败；不撤销 PAT 的变异端到端 A13 也失败 |
| 撤销全部 PAT 时保留 `updated_by_id` / 写成账户自己 / 连别的账户的也撤销 / 重复撤销已撤销的 / 只撤销未过期的 | `TestRevokeAllAPITokens` 失败；"别的账户"的变异端到端 A13 也失败 |
| 可用 PAT 数算上恰好在 `now` 到期的 / 已撤销的 / 别的账户的 | `TestCountUsableAPITokens` 失败 |
| 读密码时去掉空格 / 保留 `\r` / 不接受没有换行的最后一行 / 空输入当作空密码 | `TestUsersReadThePasswordFromStandardInput`、`TestUsersCommandsFail` 失败；保留换行的变异 A17 也失败 |
| 命令按默认参数哈希（不接 `auth.password`） | `TestUsersReadThePasswordFromStandardInput` 失败（哈希不是 `m=64,t=1,p=1`） |
| 不可用的密码仍是很快的 500 / 很快的 401 / 替身用便宜的参数 / 替身不占名额 | `TestVerify…`、`TestTheUnusableStandInHasTheCurrentParameters`、`TestLoginTakesAsLongForAnUnknownAddress` 失败 |
| 恢复不改 `is_active` / 命令行停用不重置新手引导 / 改邮箱不撤销会话、不规范化 / 创建不建资料 / 清理任务没有注册 | A12、A16、A17、A14 失败（重新构建 `bin/nerve` 后跑） |
| 重置、创建在 DEBUG 日志里写出密码 | A13、A17 失败 |

**没被发现的**：把 `close` 中"迁移执行器 → 连接池"的顺序对调。两步都在进程退出前各运行一次，没有可观察的差别；记为风险（第 6 节），不为它造测试专用的钩子。

**原型中最初没被发现、改了测试之后才被发现的**：

- **e2e 的"日志里没有密码"不可能失败**（B 类）：test 配置的日志级别是 `warn`，命令的日志都是 INFO。把密码写进 DEBUG 日志的变异，照 test 的级别运行时 A13 通过；`nerveUsers` 加上 `NERVE_LOG__LEVEL=debug` 之后 A13、A17 都失败（第 3 节第 15 条）。
- **交错 3 只测会话**（E 类）：原来的交错 3 只用会话创建 PAT，"PAT 凭证不锁账户行"的变异通过；加上用 PAT 的子测试后被发现（第 3 节第 12 条）。
- **停机顺序的测试看不出 HTTP 是否在等请求**：第一稿用半个请求，停机时连接被直接关掉（E3），"任务与 HTTP 一起停"的变异通过；改用等锁的登录后被发现。
- **重置的输出行交换两个数目**：`TestUsersCommands` 第一稿的会话和 PAT 都是 2 个，交换看不出来；改为 2 个会话、3 个 PAT 后被发现。

**缺陷类别**（P3a 评审列出的类别，对本 plan 逐类核对）：

| 类别 | 核对了什么 | 结果 |
|---|---|---|
| 测试不失败（A） | 每个 Task 的规则、边界（恰好在 `now` 到期、1 微秒、1 秒的间隔、批的大小、时限加 1 秒）、SQL 的条件、锁的子句 | 发现并修正两处：停机顺序（E3）、重置输出的两个数目 |
| 断言不可能失败（B） | 被挡住的一方（交错 1–3 的失败方向、命令的错误）除了错误，还读库断言没有新会话、新 PAT，数据库不变；交错测试以 `WaitForLockWait` 为前提，没有语句等锁时测试失败而不是空转；e2e 的密码断言 | 发现并修正一处：e2e 的日志级别 |
| 假实现忽略参数（C） | `fakeAdmin` 只认自己的地址，调用记录带账户 id、地址和原因；清理的假存储记下 `now` 和批大小 | "存储忽略批大小"的变异被发现 |
| 一行或一个账户（D） | 每个存储测试、`TestUsersCommands`、A12、A13、A16 都有第二个账户；清理的测试有已撤销的过期会话 | 7 个 D 类变异都被发现 |
| 跨凭证的键和锁（E） | 交错 3 的会话和 PAT；P3b 没有新的限流键 | 发现并修正一处：交错 3 加上 PAT |
| 日志里的密文（F） | 新增日志不带密码、地址（改邮箱）、哈希（不可用的密码）；e2e 在 DEBUG 下查密码原文 | 两个 F 类变异被发现；Go 侧 `TestSetEmail` 核对日志里没有两个地址，`TestVerifyMatchesNothingAgainstAnotherFormat` 核对 WARN 里没有哈希 |
| 读活的头部映射（G） | P3b 没有新的 HTTP handler 测试 | 不适用 |
| 测试挂住（H） | 交错测试、`platform/jobs`、停机、连接池关闭、e2e 的命令（标准输入写完即关闭） | 每个等待都有期限；3 个 H 类变异在期限后失败 |
| 闸门在争用区段之外（I） | 交错 1–3 的闸门都在持锁的事务里、第一次写入之前（2.12）；清理跳过锁住的行、按地址锁行的测试都在另一个事务持锁时断言 | 4 个 I 类变异被发现 |
| 说明与代码不符（J） | 两个操作的说明、命令的 `Short`、README 的三条、交接的处理结果、配置的注释 | 逐句与代码核对；`activate` 之后 PAT 重新可用、`set-email` 结束会话由 A12、A16 验证 |
| 接线没人看（K） | 两个配置键、清理任务、`auth.password` 进命令行、`jobs.shutdown_timeout` 进 River | 两组设置（2.7）；15 个 K 类变异都被发现 |
| 时间按原样回显（L） | P3b 的输出没有时间；写入的时刻都来自用例的时钟，存储测试按存下的值（微秒）比较 | 不适用于输出；存储测试覆盖 |

**没有证明的**：

- 持续集成（ubuntu）上的运行；S3 在 git 仓库中的结果。P3b 合并后看持续集成。
- 终端上的密码提示（第 6 节）。
- 停机时迁移执行器先于连接池关闭（第 6 节）。
- 数据库恢复之后真实的 River 客户端重新启动成功：只有读代码（E1）和假客户端的重试测试；真实客户端只测了"不可达时失败、`Stop` 立即结束重试"。
- A12 的页面版本（P5）。
