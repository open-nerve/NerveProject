# M6 / P1 planning 模块与迭代的后端实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> 状态：预检通过（2026-11-07，[re-54](../../../reviews/re-54-M6设计与P1预检.md)），尚未实施。

**Goal：** 新建 `planning` 模块，交付迭代本身的接口、规则、权限与删除/清理，并删除到期的旧 `analytics` 前端调用路径。

**Architecture：** `planning` 拥有 `cycles`；项目事实与项目行锁经 `project` 的新端口；写入沿用 M4 的加锁之路（同一个项目写闸门）；`bootstrap` 唯一接线。

**Tech Stack：** 既有 Go/PostgreSQL/sqlc/oapi-codegen/River/Playwright，不加依赖。

**Spec：** [P1-cycles-backend](../specs/P1-cycles-backend.md)（§2.4 的守卫清单与 §3 的 R1–R12 是本计划的约束）；设计 [M6](../M6-design.md) §3.1、§3.6–§3.8、§4、§5、§6。

## Global Constraints

- worktree `/Users/xiaoruan/project/nerve-project/.codex/worktress/codex/m6-p1`，分支 `codex/m6-p1`，基线为设计合入后的 main；主目录永远 main。提交详细中文，前缀 `feat/fix/docs/test(M6/P1)`，推送用 `/usr/bin/git`；noff 已获连续授权。
- `plane/`、`refer/` 只读且不在其内跑 git；不动既有容器，不 stash/clean/reset --hard/force；容器套件一次一个。
- T1 改 phase 为 `M6/P1`，同一提交删除 `analytics` 的两条例外；不延长任何 `until`。
- 手写文件约 400 行内；SQL 只经本模块 sqlc；端口由消费方声明；模块不互相 import；不放宽 `TestSQLCSchemaScope`、`rawsql_test`、删除/清理/运行角色守卫。
- 新错误码在引入它的同一 Task 进 `web/apps/web/core/lib/error-messages.ts` 与 `web/packages/i18n/src/locales/{en,zh-CN}/errors.json`，跑 `make test-web`；每个声明的码在 planning 的 apitest 二进制里有处理器测试回答。
- 测试夹具：`cycle_view` 默认关，凡建迭代的 Go 夹具与 e2e 先打开它；删除守卫的父/邻种子以 SQL 写入迭代行（夹具允许），故事的前置数据经接口。
- 每个 Task：先写会失败的测试并看到它因所测性质失败，再实现到通过；安全、加锁、访客性质至少一个能让测试失败的变异。每 Task 跑聚焦回归；T2、T3 之后的目录守卫在本 Task 内转绿；T7 之前不做整段七项检查点，T7 起有产品变化的检查点推送前串行完整七项。
- 证据放 worktree 的 `.superpowers/sdd/P1-cycles-backend/`。

## Review Focus

1. 重叠检查与并发建立/修改：项目行锁下检查，交错测试证明两次重叠写入只成一次，无 `40P01`。
2. 状态的日界：非 UTC 项目时区、固定业务时钟，日界前后与跨日；状态从不入库。
3. 拒绝次序与访客：spec §3 R2 的分法（结构 400 → 先验 422 → 404 → 403 → 409 → 锁下 422 → 重叠），OwnOnly 的写 403。
4. 删除与清理：订阅者在发布者事务内投递、沿用原 At/By、不取项目锁；worker 有界可续跑、不误伤邻项目；清理语句合规模追踪器、运行角色可执行。
5. 前端删除面：只删 `cycleDistribution` 路径与因此不用的导入，`knip`、类型检查、测试零差错。

## Task 1（T1）：phase 与到期的旧调用路径

**Files：** `tools/keywords.json`；`web/apps/web/core/services/cycle.service.ts`、`cycle.service.test.ts`；`store/cycle.store.ts`；`components/cycles/active-cycle/use-cycles-details.ts`；`app/.../cycles/(detail)/[cycleId]/page.test.tsx`。

- [ ] Step 1：phase 改 `M6/P1`，跑 `node tools/keywords.mjs` 看到 `analytics` 例外到期失败（RED）。
- [ ] Step 2：删除 `cycleDistribution`、`fetchActiveCycleDistribution`、调用与测试行、两处 `TCycleDistribution` 导入及两条例外；进行中迭代卡片的分布由 P5 重建（台账）。
- [ ] Step 3：`make lint-web knip test-web` 通过；提交 `fix(M6/P1): 关键词阶段进入M6并删除指向不存在地址的旧分布调用`。

## Task 2（T2）：首张表、模块骨架、动作与表级守卫

**Files：** `server/migrations/sql/00039_planning_cycles.sql`；`server/sqlc.yaml`；`deploy/runtime-grants.sql`；`server/migrations/planning_schema_test.go`、`purge_indexes_test.go`；`server/internal/modules/planning/{module.go,domain/actions.go}`；`access/domain/rules.go` 与 `rules_test.go`；`bootstrap/{app.go,actions_test.go}`；`purge_catalog_test.go`。

- [ ] Step 1：先改守卫（RED）：`planning_schema_test` 的约束/索引名、两处硬编码表数、`rules_test` 的三个动作、运行角色授权覆盖。
- [ ] Step 2：迁移（含 Down 与 spec §2.2 的全部约束、索引）、sqlc 条目（`date` 覆盖）、运行角色授权、模块骨架与 `Actions()`、`cycle.read/write/delete` 规则；`TestMigrationsGoUpDownAndUpAgain`、`TestConstraintAndIndexNames`、`TestTheGrantsFileCoversEveryRelationAndFunction`、`TestEveryActionHasARuleAndEveryRuleAnAction` 通过。
- [ ] Step 3：约束测试（日期 CHECK、外键去向、索引）；变异：删日期 CHECK、把 `owned_by_id` 改回 `CASCADE` 时测试失败。提交。

## Task 3（T3）：删除任务、物理清理与删除守卫

**Files：** `planning/app/{delete_rows.go,delete_rows_on_deletion.go,purge.go}`、`planning/adapter/river/`、`planning/{module_events.go,module_jobs.go,module_purge.go}`；`platform/config/`（`planning.delete_batch`/`delete_budget` 与校验）与四份配置文件；`bootstrap/{app.go,purge_ports.go}`；`project_deletion_test.go`、`workspace_deletion_test.go`、`purge_late_deletion_test.go`、`issue_project_deletion_jobs_test.go`（`parentAsyncTable`、两个 wait 助手）、`purge_runtime_test.go`、`purge_scale_fixture_test.go`。

- [ ] Step 1：守卫与夹具（RED）：父/邻项目各种一行迭代并核删除后软删、wait 助手认得 `planning.delete_rows`、运行角色清理组 `{"planning", {"cycles"}}`、规模夹具的迭代。
- [ ] Step 2：订阅者 `planning_delete_rows_on_deletion`（发布者事务内投递，原 At/By）、worker（有界续跑、不取项目锁、可重试）、清理步骤（单语句分页）、清理顺序、`EnqueuedKinds`；单元测试：回滚不投递、原 At/By、续跑、邻项目不受影响、重试幂等。
- [ ] Step 3：本地跑一次 `NERVE_PURGE_SCALE=1` 的 `TestPurgeScale`（收据）；变异：用 worker 时钟代替原 At、漏掉工作区事件里的项目、清理顺序把 planning 放到 project 之后时测试失败。提交。

## Task 4（T4）：日期、状态与重叠的纯规则

**Files：** `planning/domain/{cycle.go,dates.go,status.go,errors.go}` 与测试（导入 `_ "time/tzdata"`）。

- [ ] Step 1：表驱动测试（RED）：同空同有与顺序、名字规整、四种状态在 `Asia/Shanghai` 与 `America/Los_Angeles` 的日界前后、闭区间相交（边界同日、包含、相离、排除自身）。
- [ ] Step 2：实现；变异：`<=` 改 `<`、用 UTC 代替项目时区时测试失败。提交。

## Task 5（T5）：项目端口

**Files：** `project/adapter/postgres/queries/planning_ports.sql`、`planning_ports.go` 与测试；`project/module.go` 导出；`bootstrap/ports.go` 适配器；`planning/app/ports.go`。

- [ ] Step 1：测试（RED）：`Find`、`PlanningFacts` 不加锁且已删除项目 found=false；`LockPlanning` 的锁模式为 `FOR NO KEY UPDATE`（照 `lock_issue_projects_test.go`）。
- [ ] Step 2：实现三法与消费方端口；变异：锁模式改 `FOR KEY SHARE` 时失败。提交。

## Task 6（T6）：加锁之路与拒绝次序

**Files：** `planning/app/lock.go` 与测试。

- [ ] Step 1：单元测试（RED）：spec §3 R2 的每个分支次序；OwnOnly 的写 403；锁下重读后再判定（等待期间被删、被归档、开关被关）；锁超时经 `busyTx` 为 503。
- [ ] Step 2：实现；变异：调换 404 与 403、去掉锁下再判定时失败。提交。

## Task 7（T7）：建立迭代（第一个接口）

**Files：** `api/modules/planning.yaml`、`api/openapi.yaml`、生成物；`planning/app/create_cycle.go`、`adapter/http/`（`TestMain` 与处理器测试）、`adapter/postgres/`；`planning/module.go` 的 `Register`；权限矩阵（`permission_matrix_targets_test.go`、`seeded_test.go`、`permission_matrix_test.go` 的 `prepareMatrix`、新的迭代表与列）；`e2e/fixtures/{furnish.ts,catalog.ts}`；错误文案。

- [ ] Step 1：契约与生成；矩阵的迭代表（spec §3 R10）与种子；e2e 的 furnish 打开 `cycle_view` 并经接口建迭代、`asynchronousDeletionTable` 加 `cycles`。
- [ ] Step 2：组合与处理器测试（RED）：草稿、带日期、先验 422、重叠 409、开关 409、已归档项目 409、访客/OwnOnly 403、看不到的项目 404、`sort_order` 规则、审计同一时刻；交错测试：两个事务同时建立重叠迭代只有一个成功、无 `40P01`。
- [ ] Step 3：实现；变异：去掉重叠检查、在锁外检查时失败；串行完整七项（第一个检查点）后推送并核分支 CI。提交。

## Task 8（T8）：读取与归档目录

**Files：** `planning.yaml`；`planning/app/{list_cycles.go,get_cycle.go}`；处理器与组合测试；`planning.cycle_not_found` 的文案。

- [ ] Step 1：测试（RED）：两种顺序、互斥、`archived` 非布尔 400、已删除迭代与已删除项目 404、状态按项目时区。
- [ ] Step 2：实现；提交。

## Task 9（T9）：修改迭代

**Files：** `planning.yaml`；`planning/app/update_cycle.go`；测试；`planning.cycle_completed`、`planning.archived` 的文案。

- [ ] Step 1：测试（RED）：各字段、日期合并（R4）、空改动不写入、重叠排除自身、已结束只能改排序（R5，空名字先 422）、已归档 409。
- [ ] Step 2：实现；变异：重叠不排除自身、已结束按 UTC 判断时失败。提交。

## Task 10（T10）：归档与恢复

**Files：** `planning.yaml`；`planning/app/archive_cycle.go`；测试；`planning.cycle_not_completed` 的文案。

- [ ] Step 1：测试（RED）：只有已结束能归档、恢复、重复归档与恢复 200 不写入（R3）、`archived_at` 与审计同一时刻、开关与已归档项目 409。
- [ ] Step 2：实现；提交。

## Task 11（T11）：删除迭代

**Files：** `planning.yaml`；`planning/app/delete_cycle.go`；测试。

- [ ] Step 1：测试（RED）：管理员删任何、成员只删自己建立的、访客与 OwnOnly 403、开关与已归档项目 409、删后 404、`deleted_at` 与 `updated_*` 同一时刻。
- [ ] Step 2：实现；变异：去掉建立者判断时失败。提交。

## Task 12（T12）：接口故事

**Files：** `e2e/stories/planning/`。

- [ ] Step 1：G1–G3、G8、G18 的接口故事（PAT 与浏览器令牌对等、数据库断言）；先跑一个期望错误的版本看到断言失败，再修正。
- [ ] Step 2：`make build` 后聚焦本目录与 P4/W2/W3 删除故事通过；提交。

## Task 13（T13）：文档与交接回执

**Files：** `docs/v0/plane-diff.md`；输入交接 M1-P2（`analytics` 例外）、M2 §2、M3 §2、M4 M6-03、M5-M6-03 的“处理结果（M6/P1）”；设计 §14 的 P1 行。

- [ ] Step 1：差异（spec §2.2、`date-check` 删除、状态推算、重叠在服务端、重复归档 200）与回执，status 保持 open。
- [ ] Step 2：文档链接检查；提交。

## Task 14（T14）：门禁、整审与集成

- [ ] Step 1：串行完整七项本地门禁（收据），推送准确 HEAD，核分支 CI。
- [ ] Step 2：一次整分支 deep-review（新的只读审者），Critical/Important 一次修复轮各有 RED/GREEN，Minor 入台账；阶段记录 `reviews/P1-cycles-backend-review.md`；spec §4 勾选。
- [ ] Step 3：noff 合入、准确 main CI、归档与正常清理，设计 §14 的 P1 行标为已完成。
