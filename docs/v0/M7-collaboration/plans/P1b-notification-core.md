# M7 / P1b plan：通知模块的核心（生成与删除）

> spec：[P1b spec](../specs/P1b-notification-core.md)（预检 [re-83](../../../reviews/re-83-M7P1b预检.md) 修正后通过）。基线 main `9a3a388a`。worktree `.codex/worktress/codex/m7-p1b`，分支 `codex/m7-p1b`。台账 `.superpowers/sdd/P1b-notifications/progress.md`（ignored），现状调研 `research/p1b-facts.md`。

## Global Constraints

- spec §2 是合同，§3 的 R1–R11 是裁定；超出 spec 的契约或架构改动先停下报告。
- 模块之间只经端口：notification 不 import 别的模块；issue、project、workspace、access 交出的端口由 bootstrap 转换（`notification_ports.go`）。SQL 只经各模块自己的 sqlc；夹具不进产品包；手写文件约 400 行以内。
- P1b 不改 `api/`（没有接口）。
- 允许改的已有测试（改之前在台账逐条写明，每处只照这里写的改法）：
  - 迁移的计数与清单（`purge_indexes_test`、`trigram_schema_test`、`schema_test` 的拼接）。
  - `actions_test` 的 `actionlessModules`（加 `notification` 与一行理由）。
  - 清理：目录（`purge_catalog_test` 的计数与 `providedPurgeTables`）、运行角色（`purge_runtime_test` 的分组、表与模块次序）、规模夹具（`purge_scale_fixture_test` 的表、种行与等待；“待处理任务为 0”现在包含分发，只加等待，不放宽）、迟到删除、`issue_retention_sets_test` 自带的模块清单。
  - 项目、工作区删除守卫的夹具与等待（`twoProjects`、`seedProject`、`parentAsyncTable`、`waitProjectDeletion`、`waitWorkspaceDeletion`）。
  - 用 `IssueReader` 假件的单元测试（内嵌的新接口）。
  - 关联事件的形状断言（追加 `requested_on`，只加不改）。
  - 整表的 `river_job` 快照（`issue_draft_snapshot_test.go`、`issue_draft_publish_test.go`）：按 `kind` 过滤或多认 `notification.fanout`，原有的行照旧断言。
  - 自带任务种类清单的运行器（`app_test.go`、`issue_delete_job_fixture_test.go`、`request_jobs_test.go`）：加通知的两个种类。
  - 锁探针：只限 T8 开头按 `grep -lE "pg_blocking_pids|wait_event_type *= *'Lock'|WaitForLockWait"` 生成、记入台账的文件；只改认等锁者的方式，不改断言。
  - e2e 的 `asynchronousDeletionTable` 与删除故事的夹具。
  - 起应用的测试在持锁（`FOR UPDATE`、逐表 `LOCK TABLE`）或整库快照之前等分发跑完（T9 主控裁定：R10 的同一类，T8 的 grep 只认得探针写法）：`gate_test`、`issue_draft_publish_locks_test`、`issue_draft_write_locks_test`、`issue_collaboration_security_test`、`issue_link_security_test`、`p5_read_boundary_test`、`invitations_test` 的 `registrationRows`；只加等待，不改断言。
  - 自动归档的事件发布者假件（T14b，R12）：`issue/app/auto_archive_test.go`、`issue_auto_archive_test.go`、`issue_auto_archive_race_helpers_test.go` 的发布者实现 `PublishAll`，断言不改；事件总线自己的测试加批量的情形。
  
  其余 M4–M6 与 P1a 的断言是回归，不改。
- 证据：每次检查经 `run-check.py`，变异经 `mutate.py`，失败的运行也留着；每个加锁、可见、幂等性质至少一个能让测试失败的变异。
- 测试时间：P1b 新增的 bootstrap 测试每路增量预算 ≤ 150 秒（含删除测试多出的等待）；两个检查点各记一次两路的实测；接近单次 `go test` 20 分钟或作业 25 分钟的八成先报告（设计 §9）。规模、基准类一律选择性运行。

## Tasks

### T1 基线与设计（主控）
设计 §12 把 P1b 拆为 P1b、P1c 两行（R1）；§6.1 加关联事件的 `requested_on`（R11）；§14 回填 P1a 完成、加 P1b 一行；P1a 阶段记录 §7 回填集成。台账写基线。

### T2 表、骨架与清理
- `00049_notification_notifications.sql`（spec §2.2，含 Down）；`notification_schema_test.go`（约束、索引名，`data` 与 `change_seq` 的 CHECK、部分唯一键的反例）；`purge_indexes_test` 32 → 33，`trigram_schema_test` 48 → 49。
- `server/sqlc.yaml` 条目、`deploy/runtime-grants.sql`。
- 模块骨架：`module.go`（`Provide` 交出清理器）、`module_purge.go`、清理步骤与查询；`actionlessModules` 加 `notification`；bootstrap 的清理接线，模块次序 asset → notification → issue → …；清理目录 32 → 33、运行角色与规模夹具、`issue_retention_sets_test` 的清单。
- 新表出现后，Go 的项目、工作区删除守卫在 T3 之前会红；e2e 的删除故事到 T10 之前会红。台账记下。

### T3 删除
- issue 交出第一个端口 `ProjectIssuesRemain`（语句在 issue 的 sqlc）。
- 订阅者 `forget_deleted_issue_notifications`（`issue.deleted`，原事务，按 id 升序加锁再更新，spec §2.8）。
- `project.deleted`、`workspace.deleted` → `notification.delete_rows`：等待（500 毫秒的固定 snooze、以 `CreatedAt` 计 1 小时的上限后返回错误）、分批、预算、次数用尽记日志；配置 `notification.delete_batch`、`notification.delete_budget`（config、校验、yaml、README）。
- 守卫：`twoProjects`、`seedProject` 用 SQL 种通知行；`parentAsyncTable` 与两个等待认得新表与新任务；同事务软删的断言（工作项删除、子树删除各一，时刻与删除者取事件的；自动的删除执行者为空）；`issue.delete_rows` 被扣住时通知的删除任务只 snooze、`attempt` 不超过 1、不写。e2e 的 `asynchronousDeletionTable` 加 `notifications`。
- 变异：软删漏条件；不等工作项删完就删；等待用 `JobSnooze(0)`（应被“attempt/snoozes”的断言或时间界抓到，抓不到则记为不可区分并说明）。

### T4 issue 的端口与关联事件
- `IssueRowLock`：`ShareIssue` 移到 `IssueReader` 内嵌的小接口（R8）。
- `SubscriberIDs`、`NotificationIssue`、`CommentText`；交出方式照 `module_planning_stats.go`。
- 关联的两个事件追加 `requested_on`（spec §2.3、R11）：`relationEvent` 接收请求端（`h.own()`）；形状测试含以 `blocking` 发起与经关联 id 移除。
- 单元与 bootstrap 测试；`SubscriberIDs` 不含已软删的关注者。

### T5 收件人的批量判定
- workspace：一组账户在一个工作区的有效角色（一次查询）；project：一个项目的事实与一组账户的项目角色（一次查询）；access：`DecideMany`（spec §2.7）。
- 等价测试：权限矩阵的种子上逐格比对 `DecideMany` 与 `Authorize`（`issue.read`），加“别的工作区的项目”“已删除的工作区”。变异：漏工作区角色、漏 `guest_view_all_features`、漏项目已删除、漏工作区比对。

### T6 领域规则（纯函数）
spec §2.3–§2.5：可通知的变化与动词、变化序号（原始位置，创建 `-1`，提及取各自 `mention` 的位置）、收件人（提及去执行者、关注去执行者与提及者）、类型、快照的形状、参数的变化列表（保留位置、HTML 置空）。逐事件的表驱动单元测试；每条规则一个变异（含“删掉变化使位置前移”）。

### T7 分发用例、存储与快照的补齐
- 用例：事务里 `ShareIssue`（已删除 → 什么都不写）→ issue 的列、关注者、评论正文、状态名与项目标识 → `DecideMany` → `HiddenFrom` → 算行（工作区、项目取锁住的行）→ 一条 `INSERT … ON CONFLICT DO NOTHING`。端口在 `app/ports.go`。
- bootstrap 的 `notification_ports.go`：快照由 issue 的列加 project 不加锁的读补齐。
- 单元测试钉住调用次序（先锁后读、锁后没有取项目锁的调用）；存储的 bootstrap 测试：同一批行写两次只一份。

### T8 锁探针（上线之前，R10）
按 Global Constraints 的 grep 生成名单记入台账；逐个文件把认等锁者改为按自己语句的文本（或探针前等 `notification.fanout` 跑完）。这一步不依赖分发存在：整包两路各跑一遍为绿。

### T9 订阅者、任务与接线（上线）
- 订阅者 `notify_on_issue_events`（spec §2.9）。
- River：`notification.fanout` 参数、`Kind`、`InsertOpts`（`default`、`MaxAttempts` 5）、worker（`Timeout` 30 秒、次数用尽记错误日志）。
- bootstrap：`Provide`、订阅者登记次序、`New`、任务与种类、`job_inserter.go` 的断言；Global Constraints 列出的 `river_job` 快照与种类清单。
- 变异：参数里留 HTML；没有可通知变化也投递。整包两路各一遍。

### T10 e2e 的删除夹具 → 检查点 1
删除故事的夹具经接口让第二个账户被指派，轮询到通知行写出再删除（前提只经接口）。**检查点 1**：七项本地门禁、分支 CI、两路时间写进台账。

### T11 组合测试：工作项的写
经真实的接口写，等任务跑完读 `notifications`：`issue.created`（创建本身与负责人、序号）、`issue.updated` 的每个字段、归档与恢复、自动归档（执行者为空）；不通知的写（描述、迭代、模块、`sort_order`、转移、删除）；执行者不收；类型。

### T12 组合测试：协作与提及
评论的增改删（摘录、评论 id）、链接（含只改标题不通知）、关联（以 `blocking` 发起、经关联 id 移除，只在请求端）、附件、草稿发布带附件（R5）；提及（描述、评论），提及与关注去重，一次写里不重复。

### T13 复核、幂等与交错
- 写入之后、任务运行之前：撤权、降为只看自己（看不到他人建的）、移出私有项目、工作区管理员兼访客。
- 同一任务跑两次只一份。
- 交错：分发持有工作项行时删除等待，分发先提交、删除事务里软删；删除先提交、分发什么都不写；项目删除时分发持锁，`notification.delete_rows` 等工作项删完再删、最终没有未删的行；锁超时让任务重试。
- 变异：不锁行；锁后不复核已删除；`ON CONFLICT` 键少一列。**检查点 2**：七项本地门禁、两路时间。

### T14 规模与写入基准（R9）
- 一个工作项 1000 个关注者的一次分发（选择性运行）：行锁持有 ≤ 1 秒、持锁期间同一工作项一次 PATCH 的等待、行数、幂等重跑。
- 轻量的写入基准（R9）：分支上两组（等队列空与不等）；把测试文件复制到基线 `9a3a388a` 的临时 worktree 里再跑一次作对照（只用基线已有的辅助函数），跑完删除那个 worktree。自动归档一轮，同一次运行里带与不带通知订阅者。数字写进阶段记录。

### T14b 批量发布与批量投递（R12，负责人 2026-11-18 决定）
- `shared`：`EventSubscriber.HandleAll`（与 `Handle` 恰有其一）、`EventBatchPublisher`（`PublishAll`）；bootstrap 的分发器实现它并校验登记；`Publish(e)` 即 `PublishAll([e])`。
- `platform/jobs` 的 `Inserter.InsertManyTx`（River 的 `InsertManyFastTx`）、bootstrap 的 `jobInserter`、notification 的 river 适配器 `Fanouts.Enqueue` 收一批。
- notification：`notify_on_issue_events` 为批量订阅者，逐个事件的 `AnyBut`、形状检查与 `WithoutHTML` 不变。
- issue：自动归档的依赖改为 `EventBatchPublisher`，每个项目的事务里一次 `PublishAll`。
- 测试：分发器的批量语义（类型先检、登记次序、逐个与批量、拷贝、错误即停）；订阅者批量与逐个投出相同的参数；自动归档的事件与行不变（已有测试只换假件）；T14 的自动归档 A/B 重跑达到 R12 的验收。变异：批量订阅者漏事件、分发器对批量订阅者逐个调用或跳过、批量插入不在写的事务里。

### T15 文档与交接（主控）
`plane-diff.md`（通知的生成、复核、类型、关联的请求端、草稿附件 R5、评论摘录 R6、`issue.updated` 的动词）；README 的任务与配置；M4 交接 M7-02、M5 交接 M5-M7-02 的“处理结果（M7/P1b）”；设计 §14；阶段记录。

之后：整分支审查（fable，只读）→ 修复轮 → 限定复审 → 最终七项 → 分支 CI → `--no-ff` 合入 main → main CI → 归档、清理、记忆。

## Review Focus

- 分发与工作项删除、项目删除的串行性（R3）；锁序：分发只取工作项行的 `FOR SHARE`，之后只有不加锁的读与一条 INSERT。
- `notification.delete_rows` 的等待有界（500 毫秒、1 小时上限后报错）。
- 复核与 `Authorize` 的等价；`HiddenFrom` 用的是快照里的创建者。
- 幂等键与变化序号的稳定（参数保留原始位置）；`river_job` 里没有 HTML；没有可通知变化的写不投递。
- 关联的请求端（`requested_on`）与形状测试只加不改。
- 后台分发对已有锁探针的影响（R10）。
