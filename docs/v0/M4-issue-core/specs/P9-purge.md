# M4/P9 物理清理、描述版本保留与异步删除：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M4/P9 `purge` |
| 日期 | 2026-10-27 |
| 状态 | 准备与独立静态预检通过；负责人已授权从T1实施，以下验收按真实证据逐项关闭；阶段结束整分支deep-review，暂不合并；见[评审记录](../reviews/P9-purge-review.md) |
| 代码基线 | P8 的 main 合并 `6b9d08d4821f1423c7bb9795557727c206b7880f`；主控已核该准确 SHA 的 CI `40520609987` 三个 job 全绿，并完成 P8 证据归档、哈希复核及工作树/本远端分支清理 |
| 分支 | `codex/m4-p9`；worktree `.codex/worktress/codex/m4-p9`；主目录 HEAD 保持 main |
| 上级文档 | [M4 设计](../M4-design.md) §3.5–3.6、3.14、3.19–3.21、3.23、4.16、6.2–6.6、9.3、9.6、12/P9、13.1；[方法](../../claude_to_gpt/02-method.md)、[硬规则](../../claude_to_gpt/03-rules-and-pitfalls.md) |
| 计划 | [P9 plan](../plans/P9-purge.md) |

准备期沿 P7/P8 紧凑方法固定实施接口和验证方法，未运行完整原型、未逐 Task 重放，不把源码阅读当作运行证据。下文“新增”文件、符号、测试名是相对基线的实施目标。13个设计粗项落实为16个独立评审Task；负责人现已授权实施，合并仍待另行安排。

## 1. 目标

1. 四模块按固定顺序物理删除软删严格超过 60 天的行，23 张表各有模块内清理者、部分索引和真实目录覆盖；每批独立提交、按 id 键集推进。
2. 每个未删除项目在项目 N 锁的一次事务内，只为活 issue 的活描述版本保留 `(last_saved_at DESC, id DESC)` 最新 20 行；归档项目和归档 issue 仍处理。
3. 项目/工作区删除请求同步完成 M3 行并在原事务发布事件、投递 issue 侧任务；任务按项目或无项目草稿范围分批写回请求的时刻与删除者。普通请求、关系行、旧删除项目的归属不变。
4. 证明整个异步窗口中所有现有 issue 操作的回答与任务完成后相同；证明重做、真实 snooze、不耗尝试、失败耗尽、迟到任务与 purge 的边界。
5. 完成 I21、P4/W2/W3 的有限轮询，升级现有 scale fixture 为 SP2 的每项目 66 万行，五项目各任务首次运行完成，并记录请求、任务及其他存活项目读写延迟。

## 2. 交付物

### 2.1 文件职责、任务和尺寸

路径相对 P9 worktree。I/P/W/A 分别为 `server/internal/modules/issue`、`project`、`workspace`、`identity`；B=`server/internal/bootstrap`。同表中省略前缀的文件沿该模块目录。生成目录只由生成器修改。

| Task | 精确文件或同职责文件组 | 交付；预计手写差异行 |
|---|---|---|
| T1 | `server/migrations/sql/00030_identity_purge_indexes.sql`、`00031_workspace_purge_indexes.sql`、`00032_project_purge_indexes.sql`、`00033_issue_purge_indexes.sql`（新）；`server/migrations/schema_test.go`、`project_schema_test.go`、`issue_schema_test.go`、`draft_schema_test.go`、`trigram_schema_test.go`（新增迁移后的完整往返版本接续）；`server/sqlc.yaml`；`platform/config/{config,validate,load_test,validate_test}.go`、`purge_test.go`（新）；`server/configs/config.yaml`、`config.test.yaml`、`embed_test.go`（六字段的既有全量配置期望接续）；A/app/purge.go、adapter/postgres/purge.go、queries/purge.sql 及测试（新）、provide.go | 23 索引、既有精确全量名单同步、六配置、PAT 清理及独立批提交；500–900 |
| T2 | P/W 的 app/purge.go、adapter/postgres/purge.go、queries/purge.sql 及测试（新）；P/W/module.go | M3 两模块子先父后清理；450–850 |
| T3 | I/app/purge.go、adapter/postgres/purge.go、queries/purge.sql 及测试（新）、module.go；B/purge_catalog_test.go（新）；`server/migrations/purge_indexes_test.go`（新） | issue 13 表与完整入边目录、行为反例；700–1100 |
| T4 | B/purge.go、purge_test.go、purge_integration_test.go、purge_runtime_test.go（新，复用runtime_role_test.go的splitRoles）、app.go、app_test.go（手工testConfig补合法Cleanup，普通测试按小时调度）、all_periodic_jobs_test.go、business_clock_jobs_test.go | 单一聚合周期任务、真实运行角色与时钟；600–1000，检查点 F |
| T5 | I/app/trim_versions.go、trim_versions_test.go（新）、adapter/postgres/queries/trim_versions.sql、trim_versions.go、trim_versions_test.go（新）、adapter/river/trim_versions.go 及测试（新）、module.go、module_jobs.go；P/module.go、adapter/postgres/issue_ports.go、queries/issue_ports.sql；B/ports.go、app.go、issue_trim_jobs_test.go（新）、all_periodic_jobs_test.go | 完整未删项目目录、版本最新 20、周期接线；650–1100 |
| T6 | B/issue_retention_interleaving_test.go（请求交错与版本fixture）、issue_retention_sets_test.go（集合及归档子例）、purge_late_deletion_test.go（旧父FK链）（新）；必要的 T4/T5 生产根因修复 | trim/更新/删除互斥、归档仍保留、purge 集合分离、迟到任务前置存储边界；400–750 |
| T7 | I/app/relation_deletions.go 及测试（新）、delete_issue.go、delete_project_rows.go、ports.go、lock_either.go、对应 fakes；I/adapter/postgres/relation_deletions.go、queries/relation_deletions.sql 及测试（新）、delete_issue.go与对应 SQL；P/module.go、adapter/postgres/issue_ports.go、queries/issue_ports.sql；B/ports.go、issue_relation_ownership_test.go（新）、issue_relation_locks_test.go | DeletedAt、先锁关系后判归属、候选进度独立于写入数；集中补两端项目存在性预检；800–1200 |
| T8 | I/app/delete_project_rows.go、delete_rows.go、delete_rows_test.go、delete_rows_on_deletion.go 及测试（新）；I/adapter/postgres/deletion_remaining.go、queries/deletion_remaining.sql（新）；I/adapter/river/delete_rows.go、row_deletions.go 及测试（新）；I/module.go、module_jobs.go、module_events.go；P/W/module_events.go（新）；B/app.go、events.go、events_test.go、app_test.go、issue_writes_test.go、issue_link_jobs_test.go、issue_link_race_helpers_test.go、request_locks_test.go、request_lock_probe_test.go、request_jobs_test.go、issue_delete_jobs_test.go（新） | 预算编排、River 映射、订阅/声明/登记同时可启动；900–1300，F |
| T9 | P/app/delete_project.go 及测试、module.go；B/project_deletion_test.go、issue_project_deletion_events_test.go（新）、issue_interleaving_relations_test.go、issue_interleaving_deletion_test.go、issue_draft_interleaving_test.go；`api/modules/project.yaml`；`e2e/fixtures/assert/project.ts` | 单项目原事务事件与 204 后窗口；同步迁移旧交错11/23的关系戳断言；400–800 |
| T10 | W/app/delete_workspace.go 及测试、ports.go、module.go；P/app/cascade.go、issue_ports.go、module.go 及测试；I/app/cascade.go 及测试、module.go；B/users.go、workspace_deletion_test.go、workspace_deletion_catalog_test.go、runtime_role_test.go、issue_workspace_deletion_events_test.go（新）；`archtest/composition_test.go`；`api/modules/workspace.yaml`；`e2e/fixtures/assert/workspace.ts`、`stories/workspace/w3-workspace-settings.spec.ts` | 本次项目 Changes、删除两个过渡方法、CLI 去依赖、runtime角色草稿任务完成等待、W3 旧戳；750–1200 |
| T11 | B/issue_deletion_window_test.go、issue_deletion_window_cases_test.go、issue_null_creator_test.go（新）；既有 permission_matrix/issue_write_locks/draft_lock_shapes/relations/list/search tests | 37-operation 完整窗口、游标/动态/空创建者、现有守卫；700–1200 |
| T12 | B/issue_deletion_interleaving_test.go、issue_delete_job_lifecycle_test.go、job_state_observer_test.go（新）；既有 issue_interleaving_deletion/relations、issue_draft_interleaving tests；T6 late deletion test | 交错 24/25、既有 2/3/11/16/23、真实重试/snooze/耗尽/迟到；800–1300，F |
| T13 | `e2e/stories/issue/i21-purge.spec.ts`（新）、`e2e/fixtures/auto-archive.ts`、`clock.ts`、`assert/project.ts`、`assert/workspace.ts`；P4/W2/W3、I16 故事 | I21 的新轮次水位与所有删除故事；400–800 |
| T14 | B/issue_scale_test.go，拆出 issue_scale_fixture_test.go、issue_scale_evidence_test.go（新）；复用 T12 observer；本 spec 附录 | 原 fixture 升到 66 万/项目、直接运行次数与读写探针；650–1100 |
| T15 | 本 spec 附录、ignored 运行证据；按发现归属原生产/测试文件 | 最终性质变异、race 五遍、完整本地 F；100–350（发现修复另计） |
| T16 | 本 spec/plan/review（review 新）、`docs/reviews/re-09-M4P9整体审核.md`（目标，编号以实际占用核对）；M4/v0/M3/plane-diff/README；输入 handoffs、P1a–P8 spec§5/review§6 追加回执 | 整分支 deep-review、修复复审与后端逐项交接；350–750 |

T4 完整 F 发现的锁探针修复同属 bootstrap 测试组合职责：新增 B/request_locks_test.go（真实请求 PID、precheck QueryTracer、表/行等待与真实发布路由）和 request_lock_probe_test.go（真实 purge 干扰、连接复用身份）；普通写矩阵及既有 draft PID 调用者共用该 helper，gate 超时 fixture 同步接入。B/issue_writes_test.go、project_connection_test.go 只扩测试配置/期限接续。保留 18+3+1 目录、完整 app 的准备/拒绝/workspace 兼容/恢复请求；锁观测请求仍经真实 issue.New、事务管理、订阅与 InsertTx。该修复不修改生产锁图，其他行宽探针在 T5 引入 trim 前按静态盘点另行核验。

尺寸是计划估算，不是已生成代码的行数；Task 留 5–10% 修复余量，超过约 1500 手写行先按职责重切，不能省略锁/权限反例。新逻辑文件约 400 行以内，静态 SQL、生成物、合理测试表按现有例外处理。

### 2.2 23 表清理目录、SQL 与模块边界

以下是完整所有权集合及**实际清理顺序**；每张表索引固定为 `<table>_deleted_at_idx ON <table>(deleted_at) WHERE deleted_at IS NOT NULL`。

| 模块/迁移 | 从子到父的表序 |
|---|---|
| issue / 00033 | comment_reactions → issue_reactions → issue_mentions → issue_description_versions → issue_activities → issue_comments → issue_links → issue_relations → issue_assignees → issue_labels → issue_subscribers → draft_issues → issues |
| project / 00032 | project_user_properties → project_members → labels → states → projects |
| workspace / 00031 | workspace_user_properties → workspace_member_invites → workspace_members → workspaces |
| identity / 00030 | api_tokens |

T1随四迁移同步既有精确全量约束/索引期望：`schema_test.go` 的主体名单、`project_schema_test.go` 的 projectNames、`issue_schema_test.go` 的 issueNames加入23个部分索引（`iw`），`draft_schema_test.go` 的 wantIndexes加入 `CREATE INDEX draft_issues_deleted_at_idx ON public.draft_issues USING btree (deleted_at) WHERE (deleted_at IS NOT NULL)`。`TestConstraintAndIndexNames`、`TestDraftSchema`在T1先因新迁移与旧名单不符而红，同Task补齐精确期望后绿；不能改成contains、排除新增索引或放宽全量相等。T3新增索引catalog测试仍保留，不代替这两项既有守卫。

同一表自引用（issues.parent_id、labels.parent_id）仍由已有 CASCADE 处理；返回计数只统计本条 `DELETE RETURNING id` 的目标行，不把 FK 级联行重复计入。近期子行若所属父链已到物理清理期限，可被父 CASCADE 带走；“59 天留”对照的父链必须不具备清理资格。

```go
// 新：B/purge.go，接口只声明在消费方。
type Purger interface { Purge(context.Context, time.Time, int) (int, error) }

// 新：四模块各自 app/purge.go；四份局部类型，不互相 import。
type PurgeBatch struct { Cutoff time.Time; After uuid.UUID; Limit int }
type PurgeStep struct {
    Table string
    Delete func(context.Context, PurgeBatch) ([]uuid.UUID, error)
}
type Purger struct { /* 私有步骤 */ }
func NewPurger([]PurgeStep) *Purger
func (*Purger) Purge(context.Context, time.Time, int) (int, error)
func (*Purger) Tables() []string // 返回步骤名称副本，供目录核对；不驱动动态 SQL。
```

各 `Provided` 新增 `Purger *app.Purger`；`Provide(pool *pgxpool.Pool)` 签名不变。模块构造器将表序绑定到本模块 adapter 的 `Purge<Table>` 方法，精确后缀按表转 PascalCase，例如 `PurgeAPITokens`、`PurgeWorkspaceUserProperties`、`PurgeCommentReactions`；参数统一为本模块 `app.PurgeBatch`，结果 `([]uuid.UUID,error)`。这些是新方法，查询名与方法名相同，全部位于各模块 `queries/purge.sql`。

每批静态 sqlc SQL：严格 `deleted_at < Cutoff AND id > After`，`ORDER BY id LIMIT Limit`，单条 `DELETE ... RETURNING id`。adapter **直接使用 `gen.New(s.pool)`** 执行此单语句，依赖 PostgreSQL 隐式独立事务，不使用 `s.queries(ctx)`，不承接 ctx 内的请求事务；app 不 import pgx/platform。读取完生成查询返回的结果、确认无错误后，才累加直接删除数并把 After 设为本批最大 id；空结果才结束该表。发生错误返回此前成功批次的计数和原错误链，后续表不执行。取消不继续新批；batch 非法拒绝且不访问数据库。

这保留设计“Provide 只需 pool”和“每批独立事务”，不回填 Tx、不在 bootstrap 写 SQL、不建跨模块通用清理框架。目录测试把实际 `Tables()`、数据库所有含 deleted_at 的业务表及所有**指向这些表的 FK 入边**逐一核对；包括 comment_reactions 二级边和非清理表指向清理表的边。没有外键豁免，动作只允许 CASCADE/SET NULL，既有无条件 FK 索引守卫仍在。清理者标签与实际 SQL 绑定也通过每表行为 fixture 证明，不能只比两张手写名单。

不删除 users/profiles，不增加账户清理功能；auth_sessions 仍由原 cleanup_expired_sessions 处理。

### 2.3 配置与一个 cleanup 任务

| 配置 | 默认 | 校验 | 测试配置 |
|---|---|---|---|
| cleanup.purge_after | 1440h | >0 | 1440h |
| cleanup.purge_interval | 24h | ≥1s | 2s |
| cleanup.purge_batch | 1000 | 1..2147483647，匹配 sqlc int32 LIMIT | 1000 |
| issue.trim_versions_interval | 24h | ≥1s | 2s |
| issue.delete_batch | 1000 | 1..2147483647 | 1000 |
| issue.delete_budget | 45s | >0，且加 5s 不溢出 time.Duration | 45s |

`Config` 新增 `Cleanup CleanupConfig`，字段 `PurgeAfter/PurgeInterval time.Duration; PurgeBatch int`；`IssueConfig` 增 `TrimVersionsInterval time.Duration; DeleteBatch int; DeleteBudget time.Duration`。默认、环境变量覆盖、日志展示与拒绝均在 config 测试钉住；不引入面向用户的新错误码。

```go
// 新：B/purge.go；这里是组合任务，不是模块 SQL 仓库。
type purgeClock interface { Now() time.Time }
type purgeDeleted struct { /* 四个有序 Purger、Clock、after、batch */ }
func newPurgeDeleted([]Purger, purgeClock, time.Duration, int) *purgeDeleted
func (*purgeDeleted) Execute(context.Context) (int, error)
func purgeJob(*purgeDeleted, time.Duration) jobs.Job
```

执行时只读一次业务 Clock，固定 `cutoff=Now()-PurgeAfter`，顺序 issue→project→workspace→identity；失败停止，保留已提交计数。`cleanup.purge_deleted` 无参数、periodic、RunOnStart=true、默认24h；只登记这一个清理 worker，不把四个模块分成四任务。仅此任务的 `PeriodicSpec.InsertOpts` 设置 `Queue:"periodic"` 和 `UniqueOpts{ByState: []rivertype.JobState{JobStateAvailable,JobStatePending,JobStateRunning,JobStateRetryable,JobStateScheduled}}`：kind自动进入唯一键，不设ByPeriod/ByArgs，不包含Completed。一个周期ID本身不能防止不同轮次重叠；该活动状态唯一性同时防止旧轮重试与新轮并跑，完成后允许下一轮。T4真实River覆盖跨至少两个周期仍只有一个active/max并发1、首个job按默认策略真实失败两次（attempt 1为available、attempt 2为retryable），retryable跨周期不增active、attempt 3按原id重试、completed后新job出现，去UniqueOpts变异必须红。

它按默认 River 超时/重试运行，不套 issue.delete_budget。任务不会发布事件。runtime 角色已有相关 DELETE 权限，仍须真实使用 runtime 连接验证，不能以 grants 文本代替；不改平台登记结构或其他任务的唯一性配置。

### 2.4 描述版本保留

```go
// 新：I/app/trim_versions.go；ProjectGate 仍复用原 N 锁。
type RetentionProjects interface { UndeletedProjectIDs(context.Context) ([]uuid.UUID, error) }
type DescriptionVersionTrimmer interface { TrimDescriptionVersions(context.Context, uuid.UUID) (int, error) }
type TrimVersionsDeps struct {
    Projects RetentionProjects
    Gate ProjectGate
    Rows DescriptionVersionTrimmer
    Tx shared.TxManager
}
func NewTrimVersions(TrimVersionsDeps) *TrimVersions
func (*TrimVersions) Execute(context.Context) (int, error)
```

P 的 `IssueProjects` 增 `AllUndeletedIssueProjectIDs(context.Context)([]uuid.UUID,error)`，仅本模块 SQL；B/issueProjects 的 `UndeletedProjectIDs` 桥接它，I/ProjectPorts 加入 RetentionProjects。返回全部未删项目，含归档、没有 archive_in、没有当前成员的项目；不可复用只含归档策略项目的 ArchivePolicies。

每项目一个 `JobTx.WithinTx`，`ProjectGate.Lock` 获取 N，等待后未找到则跳过；不因 Archived 跳过。一条静态 issue SQL 用 `row_number() OVER(PARTITION BY issue_id ORDER BY last_saved_at DESC,id DESC)` 删除序号 >20 的活版本，只选择本 project 的活 issue 和活 version，不额外过滤 archived_at。保留任务不写审计/活动/事件，失败只回滚当前项目并返回此前已提交数量。

I/adapter/river 新 `TrimVersionsKind="issue.trim_description_versions"`、`TrimVersionsArgs{}`、`TrimVersionsUseCase.Execute(context.Context)(int,error)`、`NewTrimVersionsWorker(TrimVersionsUseCase)`、`TrimVersionsJob(TrimVersionsUseCase,time.Duration) jobs.Job`；结构沿 AutoArchive，periodic/RunOnStart=true。I/Deps 增 TrimVersionsInterval；Module 在构造时建出用例，Jobs 在 T5 即登记。四种 periodic 的实际运行/间隔/队列守卫同期扩展。

T5 测试观测接续：bootstrap 的 `newApp` 以私有 `newAppWithPoolFactory` 委托单一组合体，默认传 `postgres.NewPool`，签名加载/MAC→pool 顺序及失败/正常关闭所有权不变；仅测试工厂在新建 pool 前装 RequestID QueryTracer，不增公开接口或平台配置。`request_app_test.go` 与既有 requestProbe 共用 PID/查询序号观测，保留 full-app 网络、JWT、同池 gate/workers 及200并发原契约。`deactivation_probe_test.go` 以 users 独占起点固定事务；每行短 probe 从 KEY SHARE 逐级升级，只认目标/fixture 的实际 transactionid 冲突，fixture SHARE 遮挡不冒称排除目标弱锁，保留全表状态、逐阶段释放和原期限。`row_lock_probe_test.go` 保存 tuple 排队/实际 XID 的反例证明；未知冲突不能填默认结果。

### 2.5 关联归属与候选进度

P 的 `IssueProjects` 新 `DeletedIssueProjects(context.Context,[]uuid.UUID)(map[uuid.UUID]time.Time,error)`；I/app.ProjectGate 新 `DeletedAt(context.Context,[]uuid.UUID)(map[uuid.UUID]time.Time,error)`；B/issueProjects 显式转换。不加项目锁，只返回已软删项目及真实 deleted_at；空输入返回空 map，错误原样返回。缺项不是“物理删除时间”，不能合成时间。已锁住的有效 relation 的父链受 NOT NULL CASCADE 保护；父清理已完成则候选不存在，并发父清理按§2.9处理。

```go
// 新：I/app/relation_deletions.go。
type RelationDeletionRow struct { ID, ProjectID, RelatedProjectID uuid.UUID }
type RelationDeletionRows interface {
    LockProjectRelationRows(context.Context, RowBatch) ([]RelationDeletionRow, error)
    LockIssueRelationRows(context.Context, IssueDeletion) ([]RelationDeletionRow, error)
    DeleteLockedRelationRows(context.Context, []uuid.UUID, uuid.UUID, time.Time) (int, error)
}
type BatchProgress struct { After uuid.UUID; Candidates, Written int }
func NewRelationDeletions(RelationDeletionRows, ProjectGate) *RelationDeletions
func (*RelationDeletions) ForProject(context.Context, RowBatch) (BatchProgress, error)
func (*RelationDeletions) ForIssues(context.Context, uuid.UUID, IssueDeletion) error
```

同一原事务/短事务里先按 relation.id 升序取 N 锁，再用 DeletedAt 读另一端项目，最后只更新已锁且归本次的 ids。SQL 只 JOIN issue 自己的表，不 JOIN projects，不锁另一个项目。关系查询覆盖两存储方向，读取另一端 issues 时不要求其活着（当前步骤之前已软删本项目 issues）。

- 活 `deleteIssue`：另一端项目已删除就跳过。`DeleteIssueDeps` 增 `Relations *RelationDeletions`，在现有全局顺序的关系位置调用 `ForIssues(ctx,h.issue.ProjectID,d)`，其余步骤/单时刻/事件保持；去掉 IssueRowsDeleter 中旧直接 DeleteIssueRelationRows 方法及 SQL，避免第二条写路径。
- 项目任务：另一端 `deleted_at < b.At` 就跳过；同次 workspace 删除的相同 At/By 由先拿到关系锁者写，相同审计；本项目内部关系照常处理。
- `BatchProgress.Candidates` 是实际锁定候选数，After 是这些候选的最大 id；Written 仅是本次更新数。**Candidates=0 才结束关系扫描；Written=0 仍按 After 前进。** 低 id 整批归更早另一端、后面高 id 归本端的反例必须真实执行。最后一批全跳过、混合归属、重做、两任务同时处理相同 At 都要验证。

`rowSteps` 在原关系位置接受一个 `(context.Context,RowBatch)(BatchProgress,error)` 步骤函数，其余12个批方法映射返回的ids为候选/写入相同的BatchProgress，单一表序不变。T7先接live deleteIssue和目标ForProject；T8异步编排传ForProject。T7–T9的同步Cascade仍传旧 `DeleteProjectRelationRows` 的直接批函数（它处于原请求项目锁下），保留到T10同两个过渡入口一起删；不让只依赖pool的Provide提前依赖ProjectGate，不回填端口。最终只剩同一份关系归属编排和批次扫描。

T7 实施细化：`rowProgress` 将普通批次的 ids 转为 `BatchProgress`；`deleteRowBatches` 承担实际候选扫描并由同步 `deleteProjectRows` 调用，以 `Candidates==0` 结束、按 `After` 推进。`TestRelationDeletionCursorUsesCandidates` 将真实 `RelationDeletions.ForProject` 交给此生产扫描器，M17 直接变异其终止判断；不以测试内复制循环替代生产终止证明。同步 Cascade 仍为 `rowSteps` 注入旧项目关系批语句，T8 再接异步预算编排。

T7 回归接续：`B/issue_description_versions_test.go` 的 101 版本分页夹具会被正常 retention 修剪。保持原 1/50/100 容量、排序、跨资源游标与 detail 断言，改为项目 SHARE 下直接播种并通过真实 HTTP 读取；实际 trim worker 的 PID 等待证明此锁保护。新增 `TestDescriptionVersionsPagesRetentionInterference` 释放保护后验证真实 worker 保留精确最新20条以及下一页变空；不关闭 worker、不调整周期。原100次 PATCH 构造不再是分页场景职责，HTTP 版本创建与合并仍由 `TestIssueDescriptionVersionsSlideFromLastSave` 覆盖。

同时在既有 `Locks.eitherOf` 的 `both` 之后、`decideEither` 之前，用 `ProjectGate.Find` 对去重的两端项目作无锁存在性检查；任一缺失立即返回本操作的notFound，端口错误原样返回，尚未进入gate/事务。`eitherUnder` 现有两项目map存在性检查继续先于授权。否则P已删但job挂起、Q可读而不可update时会答403，job完成却答404；Q可update时还会错误进入gate/锁。T7用两存储方向、两种Q权限、已占gate/项目锁的真实反例钉住逐值404和预检零资源，不复制第二条锁路。

### 2.6 分批 app 与预算

```go
// 新：I/app/delete_rows.go；原 RowBatch/WorkspaceDraftBatch 字段不变。
type DeleteRowsResult struct { Deleted int; Complete bool }
type RowStep string // 精确取现有13表名，顺序见下文。
type ProjectRemainder struct { ProjectID, After uuid.UUID; Step RowStep }
type RemainingRows interface {
    ProjectDeletionRemaining(context.Context, ProjectRemainder) (bool, error)
    WorkspaceDraftDeletionRemaining(context.Context, WorkspaceDraftBatch) (bool, error)
}
type DeleteRowsDeps struct {
    Rows ProjectRowsDeleter
    Relations *RelationDeletions
    Remaining RemainingRows
    Tx shared.TxManager
    RuntimeNow func() time.Time
    Batch int
    Budget time.Duration
}
func NewDeleteRows(DeleteRowsDeps) *DeleteRows
func (*DeleteRows) Execute(context.Context, string, uuid.UUID, time.Time, uuid.UUID) (DeleteRowsResult, error)
```

参数顺序为 ctx/scope/id/at/by。scope 只接受 `project`、`workspace_drafts`；无效范围、零 id/by/at 拒绝且不写，不给它们造公开 HTTP 码。RuntimeNow 由 bootstrap 传 `time.Now`，用带 monotonic 的差值计算运行耗时；**不能读偏移业务 Clock 来扣预算**，也不把请求事件 At 当起点。

项目全局表序固定为 drafts→issues→assignees→labels→relations→links→subscribers→mentions→comments→issue_reactions→comment_reactions→activities→description_versions；RowStep 的字符串对应实际表名（drafts=`draft_issues` 等）。每批 `JobTx.WithinTx`，普通批仍调用现有静态 keyset SQL，关系批包含锁→跨端口读→受限更新，事务提交后才累计/推进。已删项目任务不取项目 N、不入 WriteGate；无项目草稿任务只写 `workspace_id=id AND project_id IS NULL`。只写 `deleted_at IS NULL`，三个审计列始终用原 At/By。

每批前检查取消和运行预算。预算耗尽时，**只在此边界**调用 RemainingRows 的静态候选 EXISTS：当前步骤只看 id>After，后续步骤从头；不重复已完成步骤，不把关系 Written=0 当无候选。它只查 issue 自己的表，不锁/不改行。仍有候选则 `Complete=false`；已无候选则完成，避免最后一批刚好删完后空 snooze。pending 查询不造动态表名/通用引擎，其13段顺序由单一 RowStep 表和完整性测试守住。查询错误仍是原错误，不能伪装预算耗尽。

重试/snooze 每次从各表第一条未删候选重新扫描，不持久化跨运行 cursor。已提交批次保留，失败批次回滚；预算结束不取消半批，worker 的额外 5s 只提供收尾余量，不承诺数据库阻塞一定能在余量内结束。

### 2.7 事件、投递和 River adapter

```go
// 新：I/app/delete_rows_on_deletion.go。
type RowDeletions interface {
    Enqueue(context.Context, string, uuid.UUID, time.Time, uuid.UUID) error
}
func NewDeleteRowsOnDeletion(RowDeletions) shared.EventSubscriber

// 新：I/adapter/river/delete_rows.go、row_deletions.go。
const DeleteRowsKind = "issue.delete_rows"
type DeleteRowsArgs struct {
    Scope string `json:"scope"`
    ID uuid.UUID `json:"id"`
    At time.Time `json:"at"`
    By uuid.UUID `json:"by"`
}
type DeleteRowsUseCase interface {
    Execute(context.Context, string, uuid.UUID, time.Time, uuid.UUID) (app.DeleteRowsResult, error)
}
func NewDeleteRowsWorker(DeleteRowsUseCase, time.Duration, *slog.Logger) *DeleteRowsWorker
func (*DeleteRowsWorker) Timeout(*river.Job[DeleteRowsArgs]) time.Duration
func DeleteRowsJob(DeleteRowsUseCase, time.Duration, *slog.Logger) jobs.Job
func NewRowDeletions(Inserter) *RowDeletions
```

Args.Kind 返回 DeleteRowsKind，InsertOpts 只固定 default 队列，保留 River 默认25次（不能抄 crawl 的3次）。Work 原样传 ctx/args；error 原样返回；无 error 且 Complete=false 唯一映射 `river.JobSnooze(0)`，Complete=true 返回 nil。Timeout 为 Budget+5s（默认50s）。最后一次真实失败记录 error 日志 `issue row deletion exhausted` 及 job_id/scope/id/error，不把 snooze 当失败/耗尽。

订阅者名 `delete_rows_on_deletion`，订阅 project.deleted/workspace.deleted；调用现有事务 Inserter。project 事件投一个 project；workspace Changes 每个本次 project 投一个，再投一个 workspace_drafts（零项目也投）。At/By 从 OccurredAt/ActorID；这两个请求事件 ActorID 必须非空，不用 worker 当前时钟/系统 actor 替代。事件形状非法返回错误使发布事务回滚，不静默丢项目；不扩大为事件验证框架。

T8 **同时**声明 P/W.EventTypes()、更新 B/events_test.go 的旧“future event”断言、登记 I.Subscribers/Jobs/EnqueuedKinds 及 bootstrap 类型并集；此时尚未切换请求发布点也必须完整可启动。P9 增加的 EnqueuedKinds 是 issue.delete_rows，trim/purge 属周期声明。T9/T10 再分别切换请求；任一中间提交不留未知订阅或未注册 worker。

T8 接线补充：bootstrap 的 `eventTypes()` 汇总 I/P/W 目录，用于既有 5 处完整 Subscribers 构造（app、link jobs 的两处、link race helper、request locks）；仅用自定义 issue observer 的构造保留 issue 目录。3 处 `issue.Deps` 构造显式补 DeleteBatch/DeleteBudget/RuntimeNow，手造 testIssueConfig 同步 1000/45s；生产 RuntimeNow 为 time.Now、删除 Tx 为 JobTx。`DeletionRowSteps()` 返回独立表序切片，共用于批方法映射和静态 EXISTS 候选查询，完整性测试逐表覆盖当前游标、后续表归零、已完成表不重扫。真实 Inserter 测试直接读取 default/25，fake opts 零仅证明未覆盖默认。请求快照保留 kind/args 及同 id 替换检测，排除周期与 worker 进度噪声。

T8 影响回归的局部测试修正：真实请求可能先合法等待其他事务持有的 P(N)，再到下游表夹具。请求锁探针采用“首次由本夹具阻塞的稳定观测”，同时核 backend/xact/query_start、实际 SQL、等待状态及 query tracer 序号；不按 expected SQL 过滤。全表夹具仍拒绝错误首表，下游夹具仍核 P(N) 强度及未提前写表。受控上游事务反例、错误首写/弱锁/强锁及旧 SQL 快照变异验证此观测修正；不改变生产锁协议，不归因历史瞬态 peer 身份。

T8 checkpoint fix round1：`B/project_write_locks_test.go` 的 holding 与 `issue_description_version_page_lock_test.go` 的 lockVersionPage 共用新 `row_lock_fixture_acquisition_test.go` 的 test-only 单行获取：最多5秒且尊重更短 context，成功事务交回调用者持锁，失败独立有界回滚。setup 允许合法先占者；heldBy、lockOn、rowprobe 的 NOWAIT 性质探测不变。受控已知持有者复现原 NOWAIT 的55P03，并验证真实 removeProjectMember 等待后404及精确快照、取消/deadline/连接归还、零行/多行失败与持锁生存期；原顺序/强度矩阵及生产变异保留。未知历史持有者不归因于具体 worker。

同轮 scoped 回归另暴露 draft CRUD 无项目分支的宽 owner 归因：新 `B/request_row_locks_test.go` 为既有真实 XID 升级探针加请求等待身份前后校验，`issue_draft_write_locks_test.go` 的 W/P/D 三处断言共用；新 `request_row_lock_probe_test.go` 以真实无项目 PATCH、draft 表夹具和已知第三方项目 N 复现旧宽探针红。仅目标 XID 的直接 transactionid 冲突计为持锁，tuple 排队及链可达性不作持锁证据；无项目错误锁、附项目弱/强锁生产变异继续拒绝。历史 peer 身份仍未知。

盘点接续同根因 callers：`B/issue_draft_publish_locks_test.go`、`issue_write_locks_test.go`、`request_lock_probe_test.go`、`issue_draft_interleaving_test.go` 复用真实 request XID；`membership_world_test.go`、`ending_world_test.go` 的可选 RequestID tracer 构造供 `project_row_races_test.go`、`ending_races_test.go` 使用，`project_write_locks_test.go` 保持完整网络 app/JWT/workers。row-races 显式合并尚存项目 SHARE 夹具，释放后去掉；其余项目只认请求。`interleaving_writes_test.go`、`interleaving_not_found_test.go` 与 `issue_link_transactions_test.go` 的 gate 仅在已观测 held 后允许 idle-in-transaction，通过既有 TxManager 暴露首操作身份，分别接续 `interleaving_deletion_test.go`、`issue_interleaving_test.go`；`interleaving_test.go` 共用释放 gate 后有界 join。无 backend_xid 的首锁阶段不映射空 owner；请求状态、SQL 与事务身份前后稳定检查保留；deactivation/retention 原正确路径与任意owner性质探针本体不变。



### 2.8 请求发布点、过渡删除与 CLI

P/app.NewDeleteProject 的目标签名为 `(ProjectsDeleter, shared.EventPublisher, Locks, shared.TxManager, Clock) *DeleteProject`，以 EventPublisher 取代原第二参数 IssueCascade。M3 deleteProjects 成功后在同事务发布：Type=project.deleted、Entity={project,id}、WorkspaceID/ProjectID 正确、ActorID=请求人、OccurredAt=锁后唯一时刻、Changes空。

W/app.NewDeleteWorkspace 目标签名为 `(WorkspaceDeleter, ProjectCascade, shared.Authorizer, shared.TxManager, Clock, shared.EventPublisher, *slog.Logger) *DeleteWorkspace`。原 workspace N→邀请升序→未删项目升序 N→读时钟不变；同步完成 workspace/邀请/成员/偏好及 M3 项目行后发布 workspace.deleted。Entity={workspace,id}，ProjectID零值；Changes 按此次锁住项目的 id 顺序，且每项仅 `Field:"project",OldID:&id`，不含更早删掉的 Old，也不另发 project.deleted。任意投递/订阅错误使全部 M3 改动与任务回滚；提交后原成功日志保持。

P/W.Deps 增 Events。T10 删除 I.Cascade、P/app.IssueCascade 的 `DeleteProjectIssues`/`DeleteWorkspaceIssues`，保留 StateInUse/DetachLabel；I/app.NewCascade 变为 `NewCascade(StateLabelUses) *Cascade`。P/app.NewCascade 去最后的 issues 参数，目标 `(ProjectsDeleter, WorkspaceProjectsLocker, MemberDemoter, MembershipEnder) *Cascade`；`CascadeDeps` 只留 Pool。P/W 的 `DeleteWorkspaceProjects` 收窄为 `(ctx,workspaceID,by,now) error`，workspace 用例自己保留锁住的 projects 供事件，不能因收窄丢集合。

`bootstrap/users.go` 删除过渡 issue import/Provide 参数，继续只构造 project.NewCascade；CLI 不新增 issue.New、分发器、InsertTx/jobs。组合架构测试从“Users 到达 issue.Provide”改为不得到达这些依赖；停用账户成员关系流程保持。公开契约 project/workspace DELETE 的文字同期改为 M3 同步、issue 侧任务及窗口不可见，生成物按原流程更新。

### 2.9 完整窗口、交错与故障

基线 `api/modules/issue.yaml` 实际为 **37 operation=15读+22写**。新增窗口 case 目录与契约中的 issue-tag operation 双向相等，不能靠历史“26种”或仅测 getIssue 代替。case 可以共享 fixture，不能豁免 operation；列表的分页/分组/排序变体另列子用例。

| operation 集合 | project 删除但任务挂起时与完成后必须逐值一致 |
|---|---|
| getIssue、getIssueByIdentifier；updateIssue、deleteIssue、archiveIssue、unarchiveIssue | 指名隐藏资源按现有404，与随机不存在同码/体/稳定头；编号只匹配未删项目 |
| listIssues、createIssue | 已删项目404；列表平铺/分组/子组与游标每次重新检查项目 |
| listSubIssues、addSubIssues | 隐藏目标404；从活目标引用隐藏子项按现有422路径；不泄漏计数/parent_id |
| listIssueRelations、createIssueRelations、deleteIssueRelation | 活端列表不含隐藏端；添加隐藏端422（相同字段/位置），删除指向隐藏端的关联404 |
| listIssueLinks、createIssueLink、updateIssueLink、deleteIssueLink | 隐藏父项目下所有入口同资源404，抓取晚写不改变删除审计 |
| listIssueComments、createIssueComment、updateIssueComment、deleteIssueComment | 隐藏父项目下同资源404；创建者与删除者不混淆 |
| listIssueReactions、addIssueReaction、removeIssueReaction、addCommentReaction、removeCommentReaction | 隐藏父项目下同资源404，评论反应沿真实二级父链 |
| listIssueActivities、listDescriptionVersions、getDescriptionVersion | 隐藏源404；活源动态引用已删端仍沿历史 membership 规则，不错误套 deleted 过滤 |
| searchIssues、listMemberIssues | VisibleProjects 目录排已删项目；total、顺序、OwnOnly、游标无泄漏；其他存活项目结果不变 |
| listDrafts、createDraft、getDraft、updateDraft、deleteDraft、publishDraft | 带旧项目草稿从list消失、按id404；显式把活草稿/新草稿指向删项目422；无项目草稿在活workspace仍正常，workspace删除后404；publish先守旧项目404 |

以上所有操作另覆盖 workspace 删除窗口；对 workspace slug 的请求为既有 workspace404，不能误把活workspace的无项目草稿清掉。稳定响应头比较保留请求ID/Date等非确定值的既有规范化规则，不弱化 status/problem字段/结果顺序/total/422位置。

保留已有写守卫：普通18写 P(N)、草稿3个 CRUD 写 W(S)→P(S)→D(U)、publish W(S)→P(N)→D(U) 的 union 恰为22写；预检拒绝零锁/零gate、项目锁下重读和时刻、跨项目升序、不追新项目都继续证明。

交错 2/3/11/16/23 用现有测试升级到事件提交后的窗口；新24覆盖 P先删任务挂起与活Q删除Y两顺序、两端UUID次序、worker/活删除双方关系锁竞争；新25覆盖 Old项目先删且任务未完，再删workspace，Old与本次项目/无项目草稿逐行保持各自请求的 deleted_at/updated_at/updated_by_id，Old删除者与workspace删除者不同。同次workspace两个项目任务竞争同一关系也要核审计不变。使用真实 PID、pgtest.RowWaits/条件期限，不用 sleep。

普通窗口/24/25锁路径要求无40P01。**例外**：任务挂起超过60天，issue Purger跳过活行，父project/workspace Purger CASCADE清掉它们；迟到任务零候选完成。两者恰好交叠允许原始40P01由River重试；测试既覆盖先清理后运行，也覆盖真实并发的一方回滚、重试收敛，不能把这条设计接受的边界写成“绝无死锁”。

真实 River 生命周期测试须观察任务id/args、每次进入running及其终态：部分批完成→故障→重试、重复执行零改旧戳、预算多次snooze后完成且attempt不增、最后一次错误→discarded及耗尽日志、已清理后执行。锁定River0.47.0在每次snooze累加并保留 `metadata.snoozes`，所以真实预算测试还断言同job最终snoozes>0、attempt=1；没有发生snooze则该字段缺省0。失败耗尽可在隔离库把已投递任务预置attempt=24来执行真实第25次，不等待完整退避；不得改生产默认25。观测只放 `_test.go` 的隔离测试schema：请求之前给 river_job 安装状态迁移记录，记录job_id/old_state/new_state/attempt/clock_timestamp，测试清理；不添加生产探针/API/队列。

### 2.10 E2E、规模与交接数据

P4/W2/W3 继续经既有 `furnishIssueRows(api,token,workspaceSlug,projectId,assigneeId,labelId):Promise<void>` 和 furnishWorkspace API 造齐当前13表；目录读取不改为手写豁免。expectProjectDeleted/expectWorkspaceDeleted 仅 issue 侧有限轮询，M3 同步行仍立即断言；每张表非空及整行审计、旧行/邻居不变都保留。

T10同步迁移 B/runtime_role_test.go 的既有 `TestRuntimeRoleDeletesDraftsInWorkspaceWithoutProjects`：DELETE 204后先有限等待真实 `issue.delete_rows` 的 `workspace_drafts` 任务completed，准确核对args的workspace id、原workspace请求At/By，再保留原两个草稿的deleted_at/updated_at/updated_by_id精确断言和邻workspace草稿整行不变。App与worker继续使用实际runtime角色，owner连接只做既有fixture/只读断言，不改为管理员直调清理，也不把窗口中的NULL即时Scan为bool。T10红绿选择器明确命中该测试。

W3 必须另改 Old DELETE 后立即读取 `oldDraftAudit()` 的点：从同步 project 行取得 Old请求的 At/By，记录草稿id并在最终有限轮询后逐行对照该请求戳；不得先等待 Old任务全完成才删除workspace。真实“任务未完成”由交错25的barrier钉住，页面故事不以调度恰好慢为证明。

W3零项目workspace故事还有直接 `Promise.all(tables.map(db.query))` 核live=0/审计的目录断言，不经过两个expect helper；T10同样将其中issue侧改为有限轮询，M3侧保留即时断言，原count非空及精确At/By继续成立。

I21 复用 ClockWorld，自己的独立数据库任一时刻只有一个nerve。正常进程通过API准备并删除旧issue（带活动、评论、二级反应）、PAT、label、project、workspace；偏移+61d后通过偏移进程再造/删除recent对照并留活对照。最近一次API写后读取自己的 `clock_timestamp()::text` 微秒水位；等待 kind=cleanup.purge_deleted、attempted_at严格晚于水位、completed且finalized_at晚于本代启动门，才按真实目录断言旧行及子链物理消失、recent/活对照在。抽取既有auto-archive helper为接受kind的局部参数，I16的crossing-attempt反例仍通过；不能仅看finalized_at。

升级而非替换 `TestTheDeletionsAtTheScaleTarget`：6个大项目用于单项目场景1个、workspace场景5个；另备探针项目。每项目严格20,000 issues、400,000 activities、40,000 assignees、40,000 labels、60,000 subscribers、40,000 comments、60,000 description_versions，共660,000；动态的1/5带约1KB描述。SP2不含relation/link/mention/reaction/draft与M3行，它们保留functional/catalog证明，若额外加规模行须单列，不能冒充基准。随机/交错id、真实payload长度和各表计数写证据。

不再把request/write/commit timeout全放宽为1h；HTTP请求保留正常15s deadline，token有效期可为长seed延长，计数用独立有界context。各project任务及workspace_drafts任务核唯一id、准确args、completed、attempt=1、errors空、`coalesce((metadata->>'snoozes')::int,0)=0`，并以状态记录证明**进入running恰1次且无snooze/失败**；attempt=1单独不构成证明。五项目任务每个都满足，不能只报workspace总时间。

删单项目时，读写探针在**同workspace存活邻项目**；删workspace时在**另一个存活workspace的项目**。先取无删除基线，再在任务确实running且至少一个批次提交后采样，并证明采样期间任务仍在处理；用状态记录/有限条件barrier，不长时间持锁歪曲耗时。保存全部样本、p50/p95/max、request/每job耗时、参数/机器/数据库/准确源码身份；记录测得延迟，不套历史SP2/SP4数字或凭空承诺毫秒上限。

## 3. 与设计的局部补充和裁定

1. **裁定：Purger每批用pool上的单条sqlc隐式事务。** 保留Provide(pool)与独立批提交，不回填Tx；测试外层ctx回滚不撤销已清批、失败批不计数。错了会把清理变成长事务或丢计数，T1–T4真实PG守住。
2. **裁定：关系返回候选进度与实际写入数。** 这是设计键集与先删端归属同时成立所需的局部细化；Written=0不表示耗尽。错了会漏清高id或原地循环，T7/T12反例覆盖。
3. **裁定：预算用RuntimeNow，耗尽边界才做同模块静态pending探测。** app不依赖River，adapter映射Complete；不以业务Clock偏移影响运行预算。错了会空snooze或超时，T8/T12真实调度验证；预检仍可审此实现细化，不重开事件架构。
4. **裁定：版本同刻以id降序打破平局，归档仍处理。** 与现有读取全序及“仅排已删除”正文一致；不增加历史设计没有的归档例外。T5/T6覆盖。
5. **裁定：P2“P9物理删除账户之后”的历史措辞按M4§3.21/§12/P9不删账户关闭。** 保留created_by_id=NULL的OwnOnly隐藏与AllowCreator不认人的组合测试；仅在隔离测试库将API创建行的nullable creator设NULL，不新增账户删除功能/API，生产FK仍照现有SET NULL。T11验证，T16给P2追加准确回执，不倒改历史证据。
6. **裁定：37操作完整窗口与66万规模。** 前者以当前契约目录为准，后者补足P1b现50万/项目缺的协作行与payload；旧测量只为历史，不算P9通过线。
7. **裁定：仅purge任务使用River既有活动状态唯一性。** 单一种类不等于实例串行，长轮/重试必须禁止下周期生成并行清理者；ByState包含Retryable且不含Completed，不改平台或别的任务。T4真实跨周期反例守住，错了会让父CASCADE与另一轮子表清理相遇。

这些是已批准协议的局部细化，没有新增跨模块协同方案。若实施发现架构回归、必须改变既定锁/事件/模块边界或意外高风险，按负责人纪律提交已完成工作后上报；不拿本节当越界授权。

## 4. 验收标准（全部待实施）

- [ ] A1：00030–00033的23索引、sqlc模块schema、真实目录清理者/FK入边覆盖；漏清理者/漏入边/错误动作反例实际失败。
- [ ] A2：四模块61天删、59天及精确cutoff留、活行留；子先父、id键集、独立批提交/错误计数、账户/会话范围及runtime权限通过。
- [ ] A3：单个cleanup.purge_deleted按固定模块序和一次业务时钟运行；四周期种类真实completed，漏worker两类启动守卫有效。
- [ ] A4：活项目（含归档）、活issue（含归档）的活版本仅最新20，last_saved_at/id全序；已删三层不碰；项目N和每项目事务通过。
- [ ] A5：project/workspace原事务事件/任务提交或一起回滚，准确本次项目集合、零项目草稿任务、原At/By；CLI过渡依赖和两个同步方法删除，状态/标签能力仍在。
- [ ] A6：13表软删顺序/短事务/幂等；关系双方向先锁再判、零Written候选继续、旧归属/同At竞争正确。
- [ ] A7：15读22写完整窗口、目录双向守卫、动态历史引用/游标/OwnOnly/NULL creator组合证明；既有锁/gate/权限守卫未放宽。
- [ ] A8：交错2/3/11/16/23/24/25含两项目id次序、两个删除者、真实PID等待，正常路径-race五遍无死锁。
- [ ] A9：真实River重做/局部失败/预算多轮snooze且attempt不增/耗尽日志；>60天清理后无害，允许40P01的交叠原错重试收敛。
- [ ] A10：I21 fresh-attempt水位、P4/W2/W3限时轮询及W3 Old审计确定通过；API造前置，目录无新增豁免。
- [ ] A11：SP2每项目660000行、五项目各任务首次运行完成无snooze/重试；正常请求deadline、两种存活探针与全部耗时原件写附录。
- [ ] A12：T4/T8/T12/T15及最终产品修复后F、性质变异、整分支deep-review/修复复审完成；准确分支HEAD CI与后续获授权mainCI分别核对。
- [ ] A13：§3.23、README、契约及后端交接逐条准确同步；review记录逐批约定五和未关闭责任，不提前关闭M4/后续阶段。

## 5. 不在范围内与后续责任

- 不做账户物理删除、outbox/event持久化、通用清理平台、新队列、新依赖、前端功能或M5–M8功能；M2命令行只投递客户端的首个真实使用方仍交M5。
- 不把自动归档/metadata抓取/草稿锁/列表权限重写一遍；只消费既有端口并补P9窗口证明。超过目标规模的性能不承诺，实测数字交M8。
- M4页面、后续故事、Webhook/附件/迭代订阅和最终交接status关闭仍由所属阶段承担。P9自己的实施授权、整审、集成不是本准备稿的既成事实。

## 6. 风险

关联候选被早删另一端占满时可能重复扫描，测试必须把零更新与高id推进分开；低预算的重启仍从头开始，不能偷偷持久化cursor。SQL生成类型、runtime权限和目录FK行为须真实PG验证。预算+5s无法保证任意数据库阻塞及时结束，真实数据库/context错误归River重试。大表seed/计数耗时与产品任务耗时必须分开。静态预检不能证明性能、死锁或E2E水位，所有运行通过线保持待办。

## 7. 交接与关闭条件

| 输入 | P9关闭的后端子项 | 证据责任 |
|---|---|---|
| M2-closeout §5 | PAT、带动态/评论issue的物理清理和删除关系图 | T1–4/T13；不再沿用DO_NOTHING阻挡 |
| M3-closeout §1 | issue侧改挂project/workspace事件；邀请仍原事务；registrationRows识别新增请求kind | T8–12；不新增CLI任务 |
| M3-closeout §2 | 不物理删除账户；每个批写按约定五复核 | T7/T12/T16；NULL creator测试独立保留 |
| M3-closeout §4 | M3表纳入清理、删除故事有限轮询、目录/FK索引 | T2–4/T10/T13 |
| P1a/P1b | 删除过渡方法、异步请求与升级规模/邻项目读写 | T9/T10/T14 |
| P2/P3 | NULL creator；DeletedAt；LockMany等待时项目被删、eitherUnder两项目均已锁的真实证明 | T7/T11/T12 |
| P4/P5 | 五协作表和20版本、完整窗口/游标与物理清理 | T3/T5/T11/T13 |
| P6/P7/P8 | 新事件/任务/保留/清理、旧登记守卫；links和drafts完整目录、W3、搜索窗口 | T4–13 |

T16同步总体设计§5.5/6.7清理、§5.5/6.3异步边界、M3设计§3.6/13.2删除步骤、plane-diff第四节，README列六配置、窗口不可见、重试耗尽与60天后父清理。输入handoff仅追加“处理结果（M4/P9）”，status仍open至M4收尾；不把局部关闭写成整份handoff关闭。

## 附录 A：准备与运行证据边界

2026-10-27作者准备仅静态阅读设计、交接、P8结构与准确main源码；作者未跑产品/迁移/并发/性能测试、生成/容器/构建/安装，未做原型。主控已单独运行准备关键词与架构守卫（`node tools/keywords.mjs`为72规则/23例外零命中，`go -C server test -count=1 ./internal/archtest`通过；原件在P9 ignored `checks/preparation-{keywords,architecture}/`），这不是P9产品运行证据。独立静态预检及2项局部修订复审已通过，准确冻结指纹、原始发现与证据见[阶段review](../reviews/P9-purge-review.md)及[deep-review报告](../../../reviews/re-08-M4P9预检审核.md)。当前没有P9性能数字、红绿或变异输出；全部产品验收仍待实施。

实施期每Task保存准确SHA/dirty字节hash、argv/cwd、时间、原始输出、RUN/skip/exit、变异恢复hash和独评记录；F按plan列出的七项串行取得。T14附录需填写每表seed计数、每job完整args/运行次数/attempt/终态/耗时、两组读写探针全部样本与环境；T15汇总真实范围，不能只贴PASS摘要。
