# M6 / P1：planning 模块与迭代的后端

> 状态：预检通过（2026-11-07，[re-54](../../../reviews/re-54-M6设计与P1预检.md)），尚未实施。基线为设计合入后的 main。设计见 [M6 设计](../M6-design.md)，负责人 2026-11-07 的决定 A/B/C 见设计 §15。

## 1. 目标与边界

新建后端模块 `planning`，交付迭代本身：表、建/读/改/删、归档与恢复、归档目录（`?archived=true`）、日期与状态、重叠、项目开关、权限矩阵、删除图（项目/工作区删除时的清理任务与 60 天物理清理，含运行角色）。覆盖设计 §2 的 G1–G3、G8、G18 的接口与数据库证据。

不做：工作项的归属、进度、冻结、转移（P3、P4）；模块（P2）；`cycle_user_properties`（P3）；`progress_snapshot` 列（P4，“字段随数据进入接口”）；`planning.cycle_deleted` 事件（P3 随第一个订阅者加入）；前端页面（P5）。

前端只做两件必须的事：phase 改为 `M6/P1` 后 `analytics` 例外到期，删除旧的 `cycleDistribution` 调用路径；新错误码的文案。

## 2. 交付合同

### 2.1 文件与职责

| 路径 | 职责 |
|---|---|
| `server/migrations/sql/00039_planning_cycles.sql` | `cycles` 表、约束、索引，含 Down 段（§2.2） |
| `server/sqlc.yaml` | 新条目 `planning`，schema 只列 `00039`，带 `issue` 条目同样的 `date` 覆盖 |
| `deploy/runtime-grants.sql` | 运行角色对 `cycles` 的四种 DML |
| `server/internal/modules/planning/domain/` | `cycle.go`、`dates.go`、`status.go`、`actions.go`、`errors.go` |
| `server/internal/modules/planning/app/` | `ports.go`、`lock.go`、`create_cycle.go`、`update_cycle.go`、`archive_cycle.go`、`delete_cycle.go`、`list_cycles.go`、`get_cycle.go`、`delete_rows.go`、`delete_rows_on_deletion.go`、`purge.go` |
| `server/internal/modules/planning/adapter/postgres/` | sqlc 查询与 `Store` |
| `server/internal/modules/planning/adapter/http/` | `gen/oapi-codegen.yaml`（照 `issue`）、处理器、`TestMain(m){apitest.Main(m, "planning")}` 与每个错误码的处理器测试 |
| `server/internal/modules/planning/adapter/river/` | `planning.delete_rows` 的 worker 与投递器 |
| `server/internal/modules/planning/{module.go,module_events.go,module_jobs.go,module_purge.go}` | 照 `issue` 的模板 |
| `server/internal/modules/project/` | 新端口两法：`PlanningFacts`（不加锁，`workspace_id, archived_at, timezone, cycle_view, module_view`；不加锁的查找即读它的工作区）、`LockPlanning`（事务内 `FOR NO KEY UPDATE`，返回同样的事实；项目已删除时 found=false，同 `issue_ports.sql`） |
| `server/internal/modules/access/domain/rules.go` | `cycle.read/write/delete`（`LevelProject`；delete 带 `AllowCreator`） |
| `server/internal/platform/config/` 与四份配置文件 | `planning.delete_batch`、`planning.delete_budget`（默认同 `issue` 的对应值）与校验 |
| `server/internal/bootstrap/` | 接线；清理顺序 asset → issue → planning → project → workspace → identity；投递种类；`actions_test`；权限矩阵的迭代表（§3 R10）；删除守卫、清理守卫、运行角色与规模测试的夹具（§2.4） |
| `server/migrations/` 测试 | `planning_schema_test.go` 的约束与索引名；`purge_indexes_test.go` 的表数 |
| `api/modules/planning.yaml`、`api/openapi.yaml` | 新标签与路径（§2.3） |
| `web/apps/web/core/services/cycle.service.ts`（及测试）、`store/cycle.store.ts`、`components/cycles/active-cycle/use-cycles-details.ts`、`[cycleId]/page.test.tsx` | 删除 `cycleDistribution`/`fetchActiveCycleDistribution`、它们的调用与因此不再使用的 `TCycleDistribution` 导入（`progress-stats.tsx` 仍用该类型） |
| `tools/keywords.json` | `phase` 为 `M6/P1`；删除 `analytics` 对这两个文件的例外 |
| `web/apps/web/core/lib/error-messages.ts`、`web/packages/i18n/src/locales/{en,zh-CN}/errors.json` | 新错误码的文案（M4 约束 4） |
| `e2e/` | 新的 `furnishPlanningRows` 以项目管理员经接口打开 `cycle_view` 并建一个迭代，`furnishWorkspace` 与 P4、W3 的删除故事在 `furnishIssueRows` 旁调用它（`furnishIssueRows` 的调用者多是项目成员，不能改项目设置）；`catalog.ts` 的 `asynchronousDeletionTable` 加 `cycles`；`stories/planning/` 的 G1–G3、G8、G18 接口故事 |
| `docs/v0/plane-diff.md`、各输入交接 | `cycles` 的差异；“处理结果（M6/P1）” |

### 2.2 表与 Plane 差异

`cycles`：

| 列 | 定义 | 与 Plane |
|---|---|---|
| `id` | `uuid PRIMARY KEY` | 同 |
| `name` | `varchar(255) NOT NULL` | 同；去掉首尾空白后不能为空 |
| `description` | `text NOT NULL DEFAULT ''` | Plane 可空 |
| `start_date`、`end_date` | `date NULL` | Plane `timestamptz`；CHECK 两者同空同有且开始 ≤ 结束 |
| `owned_by_id` | `uuid NULL REFERENCES users ON DELETE SET NULL` | Plane 非空 `CASCADE`（同 M3 对项目负责人的改法） |
| `project_id`、`workspace_id` | `uuid NOT NULL`，`CASCADE` | 同 |
| `sort_order` | `double precision NOT NULL DEFAULT 65535` | 同；新建取本项目未删除迭代（含已归档）的最小值 − 10000，没有时 65535（Plane `cycle.py:88-95`） |
| `archived_at`、`deleted_at` | `timestamptz NULL` | 同 |
| 审计四列 | 照 M2 设计 3.13 | 同 |
| 删除 | `view_props`、`logo_props`、`timezone`、`version`、`external_source`、`external_id`；`progress_snapshot` 在 P4 加回 | 没有读者，或用项目的时区 |

索引：每条 `CASCADE` 外键（`workspace_id`、`project_id`）无条件索引，指向 `users` 的 `SET NULL` 外键照 `projects.project_lead_id` 不建（`TestConstraintAndIndexNames` 的约定）；活行列表 `(project_id, sort_order, created_at, id) WHERE deleted_at IS NULL`；清理用 `cycles_deleted_at_idx (deleted_at) WHERE deleted_at IS NOT NULL`（照 `purge_indexes_test`）。约束与索引按既有命名，列入 `planning_schema_test.go`。

### 2.3 接口与错误目录

`Cycle`：`id, project_id, workspace_id, name, description, start_date, end_date, status, owned_by_id, sort_order, archived_at, created_at, updated_at, created_by_id, updated_by_id`。`status` ∈ `draft|upcoming|current|completed`，读取时按项目时区与业务时钟推算。

| 操作 | 成功 | 声明的错误 |
|---|---|---|
| `GET /projects/{project_id}/cycles?archived=` | 200 `{data}`；默认未归档按 `sort_order, created_at, id`；`archived=true` 只返回已归档，按 `archived_at DESC, id` | 400（参数不是布尔）、404、403（看得到项目但不是它的成员） |
| `POST /projects/{project_id}/cycles` `{name, description?, start_date?, end_date?}` | 201 | 400、422、404、403、409 `project.archived`/`planning.cycles_disabled`/`planning.cycle_dates_overlap`、503 |
| `GET /cycles/{cycle_id}` | 200 | 404 `planning.cycle_not_found`、403（看得到项目但不是它的成员） |
| `PATCH /cycles/{cycle_id}` `{name?, description?, start_date?, end_date?, sort_order?}` | 200 | 400、422、404、403、409 `project.archived`/`planning.cycles_disabled`/`planning.archived`/`planning.cycle_completed`/`planning.cycle_dates_overlap`、503 |
| `DELETE /cycles/{cycle_id}` | 204 | 404、403、409 `project.archived`/`planning.cycles_disabled`、503 |
| `POST /cycles/{cycle_id}/archive`、`/unarchive` | 200 `Cycle` | 404、403、409 `project.archived`/`planning.cycles_disabled`、`planning.cycle_not_completed`（归档）、503 |

看不到的项目回答沿用既有的项目 404；看不到或不存在的迭代 404 `planning.cycle_not_found`（先例 `issue.comment_not_found`）。

### 2.4 删除与清理

- 软删：`deleted_at` 与 `updated_*` 同一时刻、同一写者；P1 不发事件。
- 项目、工作区删除：订阅者 `planning_delete_rows_on_deletion` 在发布者事务内投递 `planning.delete_rows`（参数：项目 id、原 At/By）；worker 按 `planning.delete_batch` 有界软删该项目的迭代，沿用原 At/By，在 `planning.delete_budget` 内续跑，可重试。不取项目锁（项目已删除，同 `issue/app/delete_rows.go`）。父范围立即隐藏：项目已删除即 404。
- 60 天物理清理：单语句分页（`WITH candidates AS MATERIALIZED … DELETE FROM cycles`）有界删除；运行角色执行（`TestPurgeDeletedRunsWithRuntimeRole`）。
- 守卫与夹具同步扩展：`project_deletion_test`、`workspace_deletion_test`、`purge_late_deletion_test` 以 SQL 种入 `cycles` 行并核软删（夹具允许 SQL）；`parentAsyncTable` 与 `waitProjectDeletion`/`waitWorkspaceDeletion` 认得 `planning.delete_rows`；`purge_runtime_test` 的模块组加 `{"planning", {"cycles"}}` 与夹具中的一行迭代；`purge_catalog_test` 与 `purge_indexes_test` 的表数；`purgeScaleTables` 加 `cycles`，规模夹具以 SQL 种迭代（建立接口在 T7 才有）并等待 `planning.delete_rows`。P1 以 `NERVE_PURGE_SCALE=1` 跑一次 `TestPurgeScale` 证明追踪器认得新语句，P8 在最终代码上重跑。

## 3. 局部裁定

- **R1 加锁之路**：照设计 §6：不加锁预检 → 项目写闸门（同一个 `projectWrites`）→ 事务（`busyTx{issueTx}`，锁超时 503 `server_busy`）、项目行 `FOR NO KEY UPDATE` → 锁下重读迭代与再判定 → 最后一把锁之后读时钟 → 写。重叠检查在项目行锁下读本项目所有未删除迭代。
- **R2 拒绝的次序**（同 M4 的分法）：结构问题 400 → 不读数据就能判定的字段问题 422（名字为空或超长、NUL、`sort_order` 非有限数、POST 的日期同空同有与顺序）→ 404 → 403（角色、OwnOnly 的写）→ 409 `project.archived` → 409 `planning.cycles_disabled` → 409 `planning.archived` → 409 `planning.cycle_completed` → 锁下才能判定的 422（PATCH 合并后的日期，R4）→ 409 `planning.cycle_dates_overlap`。因此已结束的迭代收到空名字时回答 422（先于 R5 的 409）。
- **R3 重复归档与恢复**：照 M4 工作项归档：已是目标状态时 200、不写入（`archive_issue.go:62-70`）；`planning.archived` 不适用于归档与恢复本身。
- **R4 PATCH 的日期**：`start_date`、`end_date` 各自可给、可为 `null`；与现值合并后须同空同有且开始 ≤ 结束（422 指向给出的字段）。空 PATCH 或没有实际改动：200、不写入（同 M4）。
- **R5 已结束的迭代**：只接受 `sort_order`；其他字段出现即 409 `planning.cycle_completed`（哪怕值相同）。删除、归档不受此限。
- **R6 OwnOnly**：Grant 为 OwnOnly 时一切写 403；读照常。
- **R7 开关**：`cycle_view=false` 时一切写（含删除、归档、恢复）409；读照常。`cycle_view` 默认 `false`（`00010`，同 Plane），测试与故事先经接口打开（Go 的 `issueWorld` 类夹具、清理运行角色夹具、e2e 的 `changeProject`）。
- **R8 时区**：项目的 `timezone` 由 M3 校验为合法 IANA 名；状态以 `time.LoadLocation` 换算业务时钟的“今天”。`LoadLocation` 失败视为内部错误（500，不应出现）。domain 测试导入 `_ "time/tzdata"`。
- **R9 归档参数**：照 M3 `listProjects` 的布尔参数：`true` 只返回已归档，`false` 或不给返回未归档；非布尔在绑定时 400。
- **R10 权限矩阵**：在 `projectTables` 下加迭代表（`underProject` 的 `{cycle_id}`、`seeded.cycles` 经 planning 的 store 种入、`prepareMatrix` 打开 `cycle_view`），列含建立者与工作区管理员兼访客（OwnOnly），每列一个 `cycleOf(c)`。
- **R11 配置**：`planning.delete_batch`/`delete_budget` 独立于 `issue` 的键（同默认值），四份配置文件与校验一起加。
- **R12 动作与路由的次序**：`actions.go` 与 `rules.go` 在建表的同一 Task 加入（`TestEveryActionHasARuleAndEveryRuleAnAction`），`Register` 在第一个接口 Task 加入。

## 4. 验收标准

- [ ] A1 G1：只有名字成草稿；日期同空同有、顺序；重叠（含已归档、边界同日即相交）409；两次并发建立重叠的迭代只有一次成功（交错测试，无 `40P01`）。
- [ ] A2 G2：四种状态在非 UTC 项目时区的日界前后推算正确（固定业务时钟，含跨日）。
- [ ] A3 G3：改名、改说明、改日期（含重叠、排除自身）、改排序；空改动不写入；已结束只能改排序；已归档 409；删除只限管理员或建立者。
- [ ] A4 G8：只有已结束的能归档；重复归档与恢复 200 不写入；归档目录与默认列表互斥且顺序正确。
- [ ] A5 G18 与权限：开关关闭时写 409、读 200；矩阵（管理员、成员、建立者、访客、OwnOnly、非成员、私有项目）与 `permission_matrix_coverage_test` 一致；看不到的一律 404；每个声明的错误码在 planning 的 apitest 二进制里有回答。
- [ ] A6 删除：项目、工作区删除后迭代立即不可读，任务以原 At/By 软删并续跑；60 天清理有界、运行角色可执行、`TestPurgeScale` 认得新语句；删除与清理的全部守卫覆盖 `cycles`。
- [ ] A7 phase 为 `M6/P1`，`analytics` 例外与旧调用路径删除；新错误码有中英文案；输入交接写“处理结果（M6/P1）”。
- [ ] A8 七项本地门禁、deep-review 修复闭环、准确分支 CI 与 noff 后 main CI 全绿、归档清理。

## 5. 后续责任

P2 模块（共用项目端口与加锁之路，`module_view` 已在端口里）；P3 归属、`planning.cycle_deleted`、`cycle_user_properties`；P4 进度、冻结（`progress_snapshot`）、转移；P5 前端迭代页面重写（含进行中迭代卡片的分布与燃尽）；P8 规模测试重跑。

## 6. 风险

- 项目端口加在 `project` 模块：只加查询与端口，不改工作项路径的行为；以现有锁测试为回归。
- 建表即触发多道按目录驱动的守卫（§2.4）：建表与删除任务放在相邻的两个 Task，第一个接口 Task（T6）之前不做整段七项检查点。
- 启动时运行的任务与持锁测试（M5 P8 的教训）：本阶段不加周期任务。
