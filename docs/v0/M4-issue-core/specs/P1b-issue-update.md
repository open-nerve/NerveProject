# M4/P1b 工作项的取、改与写闸门：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M4/P1b `issue-update` |
| 日期 | 2026-10-18 |
| 状态 | 已完成（[评审记录](../reviews/P1b-issue-update-review.md)）：第 3 节已由控制者裁定（2026-10-18，`$M4TMP/p1b-architect-rulings.md`：各条照建议，标在各条之后）；预检的发现（High 0、Medium 4、Low 6，外观 1）由修订一轮（`f094ced6`，`$M4TMP/p1b-preflight-rulings.md`，附录 A.2；设计 3.5、8.8 的一句在 `e5df62f6`）改入；15 个 Task 逐个实现、逐个评审（2026-10-18 至 10-19）；整分支评审（Critical 0、Important 0、Minor 5）之后的修复轮（`203f1f94`、`f18de99d`）和范围复审（Critical 0、Important 0、Minor 3、Nit 3，另有可选的两条）之后的遗留（评审记录的提交）按评审记录第 4 节改入本 spec、plan 和设计。P1 由控制者按决策点 2 的预先批准拆成 P1a、P1b（2026-10-17，设计第 12 节"拆分"一段）；P1a 已合并（`5ce30a1a`）。P1b 的十二个草稿任务在约 16 个、每个约 1,500 行以内重新切成 15 个 Task（裁定 3），切分时没有超出，没有停下（执行中的修复轮让 Task 5 长过了 1,500 行，附录 A.1） |
| 上级文档 | [M4 设计文档](../M4-design.md) 第 2（I2 的"P1b"一段）、3.4（P1b 只有管理员和成员）、3.5（取锁的一步、写闸门、`lock_timeout`、交错 3–7 的修改一支、交错 20、约定五）、3.6（负责人的读取）、3.7（P1b 的字段：名称、状态、优先级、负责人、标签）、3.12、3.13（动态的第 3、6、7、10–13 行）、3.20、3.23 与 8.8（P1b 的行）、4.17（P1b 的行）、5.1–5.4（`getIssue`、`updateIssue`）、6.1–6.7、9.1–9.6（P1b 的部分）、10 决策点 7（规模的测量）、11.11、12（P1b 与约束 1–8）、13.1 节 |
| 前置交接 | [P1a spec](P1a-issues.md) 第 5 节"交给 P1b"；[P1a 评审](../reviews/P1a-issues-review.md)第 6 节；[M2 收尾交接](../handoffs/M2-closeout.md)第 3 节（修改的部分）；[M3 收尾交接](../handoffs/M3-closeout.md)第 2 节。落点见第 3 节第 17 条、第 7 节 |
| 计划 | [P1b plan](../plans/P1b-issue-update.md) |

本 spec 只写 M4 设计交给 P1b 决定的东西：名字、签名、SQL、测试名，以及原型证明了什么。规则本身以 M4 设计为准，这里引用节号，不重述。P1b 从 `5ce30a1a`（P1a 的合并提交）开始。

P1b 是 M4 加锁约定的后一半（设计 12 节约束 3：写闸门和连接池是 P1b 的评审重点）：按资源寻址的写第一次落在 P1a 的取锁路径上（读工作项 → 预检 → 闸门 → 项目 N → 锁下重读 → 判定），写闸门插在预检和事务之间，`lock_timeout` 兜底，两者的 503 只在 `bootstrap/gate.go` 一处换码。附录 A 的清扫逐个核对每个这样的测试在它所说的性质去掉之后失败，并记下失败在哪一层（单元、存储、组合、端到端）；只在单元一层失败的安全或加锁的性质算缺口，按性质只能在单元一层的逐条写在附录 A.3。

## 1. 目标

按 M4 设计 12 节 P1b：工作项的取、改与写闸门。具体是：

- `getIssue`（`GET /api/v0/issues/{issue_id}`）：不加锁、不开事务，读工作项和它全部未删除的负责人、标签（3.6），判定 `issue.read`（P1b 是项目的管理员、成员，3.4）；没有、已删除、看不到项目的 404 `issue.not_found`（新码，带文案），看得到而不能读的 403。
- `updateIssue`（`PATCH /api/v0/issues/{issue_id}`）：名称、状态、优先级、负责人、标签（3.7；描述在 P4、起止日期在 P2、排序和父工作项在 P5、P3）。取锁的一步照 3.5 的"修改工作项"一行：读工作项（不加锁）→ 预检（凡判定拒绝的 404、403 在这里答）→ 写闸门 → 事务 → 项目行 `FOR NO KEY UPDATE` → 锁下重读工作项，确认仍在、仍属于这个项目 → 锁下判定（最终的）→ 项目已归档 409 → 什么都不改时不写 → 依赖行的检查 → 时刻 → 写工作项、负责人和标签的差集、新负责人的关注、动态的第 3、6、7、10–13 行（3.13），同一时刻、同一个写者。
- 写闸门（3.5、11.11）：`platform/gate` 的 `Keyed[K]`（只用标准库）；`issue/app` 声明的端口 `WriteGate`；`bootstrap/gate.go` 用四个配置接成它：`issue.write_gate.per_project`（2）、`max_wait`（2 秒）、`max_queue`（64），`issue.write_lock_timeout`（5 秒）；`platform/postgres` 的 `TxManager` 选项 `LockTimeout(d)` 和 `LockNotAvailable(err)`；排满、等到 `max_wait`、在闸门上请求到期、等锁到期都答 503 `server_busy`，带 `Retry-After: 1`；`createIssue`、`updateIssue` 在契约里声明它。
- 交错 3–7 的修改一支（3u–7u）、本 spec 补的同一个工作项的两个修改（1u）和等锁期间工作项被删除或移走的修改、交错 20（经真实的连接池和闸门），`-race -count=5`，没有 40P01。
- 在规模目标上量 P1a 的同步连带（决策点 7）：请求的耗时和工作项一侧的软删除的耗时分开记下（附录 A.4、第 3 节第 14 条）。
- 端到端：I2 的接口版本（`expectIssueUpdated`，照 `expectIssueCreated` 的写法）；3.23、8.8 中 P1b 的行。

不加 Go 模块、npm 包；跨模块的端口不变（`project` 给 `issue` 的 `IssueProjects` 的两个目录多回答名称）；没有新表、新迁移。

## 2. 交付物

### 2.1 文件总览

路径相对于仓库根目录（`server/internal/` 省略）。"Task"是 plan 中负责它的任务（plan 有完整的文件表）。本 Phase 手写的代码、测试和配置文件 81 个（新建 24 个，修改 57 个），生成物 6 个，文档 8 个（本 spec 和 plan 新建；修改的是总体设计、差异清单、README、两份交接和 P1a 的 spec，最后一个只把第 3 节的"待裁定"标成已裁定、改了它的状态一行，整分支评审的修复轮和评审记录的提交；另有 M4 设计：第 15 节的一行，3.5、8.8、11.11、13.2 的连接池的总代价（预检的 M4：`f094ced6` 改 11.11、13.2，`e5df62f6` 改 3.5、8.8），整分支评审的修复轮改的 3.5、6.5、9.1、11.11、13.2 的几句）。

| 路径 | 内容 | Task |
|---|---|---|
| `issue/domain/patch.go` 及测试；`activity.go` 及测试 | 修改一侧的规则：`IssuePatch`、`CheckIssuePatch`、`Diff`、`Changes`、`ChangesOf`、`None`；动态的第 3、6、7、10–13 行（`Update`、`UpdateActivity`） | 1 |
| `project/adapter/postgres/queries/issue_ports.sql`、`issue_ports.go` 及测试；`project/app/issue_ports.go`；`project/module.go`；`issue/app/ports.go`、`create_issue.go` 及测试、`fakes_test.go`；`issue/module.go`；`bootstrap/ports.go` 及测试 | 状态、标签的目录回答名称（动态写名称）；`LabelRef` | 2 |
| `issue/adapter/postgres/queries/update.sql`、`update.go`、`update_test.go`；`issue/app/rows.go` | 修改的存储：`UpdateIssue`、`RemoveAssignees`、`RemoveLabels` | 3 |
| `issue/domain/errors.go`、`actions.go`；`access/domain/rules.go` 及测试；`issue/app/lock.go`、`ports.go`、`get_issue.go` 及测试、`create_issue_test.go`；`issue/module.go`；契约；`issue/adapter/http/*`；矩阵的六个文件（`permission_matrix_seed_issues_test.go`（新）、`permission_matrix_seeded_test.go`、`permission_matrix_targets_test.go`、`permission_matrix_project_test.go`、`permission_matrix_issues_test.go`、`permission_matrix_test.go`）；前端的三个文案文件 | `getIssue`：`issue.not_found`、`issue.read`、`Locks.see`、`IssueReader`、契约、HTTP、矩阵的种子和行 | 4 |
| `issue/app/lock.go`（整个文件）、`writes.go`、`update_issue.go` 及测试、`create_issue.go` 及测试、`fakes_test.go`、`ports.go`、`get_issue_test.go`；`issue/domain/patch.go` 及测试、`issue.go`、`actions.go`；`access/domain/rules.go` 及测试；`issue/module.go` | `Locks.run` 和按资源寻址的写；`updateIssue` 的用例；`issue.update` | 5 |
| 契约；`issue/adapter/http/handler.go`、`issues.go`、`update_test.go`、`handler_test.go`；`issue/module.go`；`bootstrap/permission_matrix_issue_updates_test.go`、`permission_matrix_issues_test.go`、`permission_matrix_test.go`、`issue_world_test.go`、`issue_write_locks_test.go`、`issue_writes_test.go` | `updateIssue` 的接口、接线、矩阵、取锁形状的一行（每行发的路径是它的操作的，`pathOf`）；创建的盖戳测试用 `newWorkItem` | 6 |
| `bootstrap/issue_updates_test.go`、`issue_writes_test.go`、`issue_world_test.go` | 盖戳、回答如存、事务的连接；`workItemStamps` | 7 |
| `platform/gate/gate.go`、`gate_test.go` | `Keyed[K]` | 8 |
| `platform/postgres/tx.go`、`lock_timeout_test.go`；`platform/config/config.go`、`validate.go` 及三个测试；`server/configs/config.yaml`、`embed_test.go` | `LockTimeout`、`LockNotAvailable`；四个配置 | 9 |
| `issue/app/ports.go`、`lock.go`、`lock_test.go`、`fakes_test.go`、`create_issue.go` 及测试、`update_issue.go`（两个 `Execute` 的说明写上闸门）、`get_issue_test.go`；`issue/module.go`；`bootstrap/gate.go`、`gate_test.go`、`app.go`、`app_test.go`、`issue_world_test.go`、`issue_writes_test.go`；契约；HTTP 的两个测试；`archtest/composition_test.go` | 写闸门接入；`server_busy` | 10 |
| `bootstrap/issue_interleaving_test.go`、`issue_interleaving_deletion_test.go`、`issue_interleaving_catalog_test.go`（整个文件）、`issue_interleaving_change_test.go`（新）、`interleaving_writes_test.go`、`interleaving_test.go`、`issue_world_test.go`、`issue_updates_test.go` | 交错 1u、3u–7u；`serializeOn` 收进 M3 的手写门，它的说明随之；`newWorkItemIn`、`activityBy` 共用 | 11 |
| `bootstrap/issue_pool_test.go` | 交错 20 | 12 |
| `bootstrap/issue_scale_test.go` | 规模的测量（只在 `NERVE_MEASURE_SCALE` 设置时运行） | 13 |
| `e2e/fixtures/issues.ts`、`assert/issue.ts`；`e2e/stories/issue/i1-create-issue.spec.ts`、`i2-update-issue.spec.ts` | I2 的接口版本 | 14 |
| `docs/v0/v0-design.md`、`plane-diff.md`、`README.md`；两份交接 | 3.23、8.8 中 P1b 的行；交接的处理结果 | 15 |

生成物：`project/adapter/postgres/gen/issue_ports.sql.go`（Task 2）；`issue/adapter/postgres/gen/update.sql.go`（Task 3）；`api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`、`issue/adapter/http/gen/server.gen.go`（Task 4、6、10），`issue/adapter/http/gen/bodyshape.gen.go`（Task 6）。

### 2.2 依赖

不加 Go 模块、npm 包；`platform/gate` 只用标准库（`context`、`errors`、`slices`、`sync`、`time`）。`access` 的规则表加两行：`issue.read`、`issue.update`（项目级，管理员、成员）。跨模块的端口不变，只多回答名称：`issue/app` 的 `StateRef` 加 `Name`，`LabelCatalog.Labels` 回答 `[]LabelRef`（`id`、名称），`project` 的 `IssueState` 加 `Name`、`IssueLabels` 回答 `[]IssueLabel`，`bootstrap/ports.go` 转换。`issue/app` 新声明 `IssueReader`（`GetIssue`，`IssueCreator` 嵌入它）、`IssueUpdater`、`WriteGate`。没有模块之间的导入（`archtest`），`issue` 的 sqlc 只读写自己的表（`TestSQLCSchemaScope`）。命令行的组合（`bootstrap.Users`、`Workspaces`）不建写闸门（`archtest` 的 `composesMore` 加 `platform/gate`）。

### 2.3 修改一侧的规则（Task 1；3.7、3.13）

`issue/domain/patch.go`：
- `IssuePatch{Name, StateID, Priority, AssigneeIDs, LabelIDs}`，各为指针：没给的不改；给出的负责人、标签是新的集合，空的就是没有。
- `CheckIssuePatch(p) (IssuePatch, error)`：与 `CheckNewIssue` 同一组规则（名称去掉首尾空白之后 1–255 个字符、没有 NUL；优先级五个值之一，给空串也拒绝；负责人、标签各至多 100 个、不重复），全部问题一次报出（422）；回答去掉空白之后的补丁。
- `Diff{Added, Removed}`、`diffOf(from, to)`：新集合中工作项没有的（按 `id` 排序，约定五）、工作项有而新集合没有的（按工作项的次序，即 `id` 的次序）。
- `Changes`、`ChangesOf(i, p)`：名称、优先级、状态只在与工作项的不同时算改动；负责人、标签给出时算差集。`None()`：什么都不改。
- `Unassignable(asked, added, roles)`（Task 5 加）：只查新加的负责人（`CanAssign`），留下的不查（第 3 节第 3 条），问题落在它在请求里的下标。

`activity.go`：`Named`、`Changed[T]`、`Update`、`UpdateActivity(u)`：按 3.13 的表的次序：第 3 行名称、第 6 行优先级、第 7 行状态（名称和 `id`）、每个加的标签（第 10 行）、去的标签（第 11 行）、每个加的负责人（第 12 行）、去的负责人（第 13 行），各列表按给出的次序；Plane 的 `comment` 文字（"updated the name to"、"added label " 等）。没改的字段不写行。

### 2.4 目录回答名称（Task 2；3.13，6.5）

动态的第 7、10、11 行写状态、标签的名称（3.13 的 `old_value`、`new_value`）。`IssueStates` 多读 `name`，`IssueLabels` 回答 `id` 和 `name`（P1a 的第 5 节把"状态的名称"放在 P5，P1b 的动态先要它，第 3 节第 9 条）。`createIssue` 按 `id` 找标签。存储测试 `TestIssueCatalogs` 读出名称（标签的回答按集合比较：写的一侧按 `id` 查找，不依赖次序）；转换测试 `TestIssueProjectsConvertsProjectsAnswers` 加标签的一行和它的失败。

### 2.5 修改的存储（Task 3；3.7）

`queries/update.sql`：
- `UpdateIssue`：写名称、状态、优先级、`completed_at`（给什么写什么，`NULL` 也写）、`updated_by_id`、`updated_at`，`WHERE id = $id`。
- `RemoveAssignees`、`RemoveLabels`：这个工作项未删除的、给出的账户或标签的行，写 `deleted_at`、`updated_at`、`updated_by_id`，一条语句（`= ANY`，部分唯一键 `(issue_id, assignee_id)`、`(issue_id, label_id)` 服务它）。

三条都在项目 N 之下，只有持项目 N 的写会写到这些行，不按 `id` 先锁（约定五，第 3 节第 12 条）。加的负责人、标签、关注沿用 P1a 的 `InsertAssignees`、`InsertLabels`、`InsertSubscribers`。`app/rows.go` 的 `IssueFields`、`UnlinkRows`。存储测试 `TestUpdatingAWorkItemWritesItsRows`：另一个工作项的同一个账户、标签的行，这个工作项上没给的账户、标签的行，一小时之前已删除的同一个账户、标签的行，都不动、保留它们的戳；改回不是完成组的状态、`completed_at` 给空，就不再完成。`TestAFailedUpdateStatementIsAnError`。

### 2.6 `getIssue`（Task 4；3.4、3.6、5.1–5.3）

- `issue/domain`：`ErrIssueNotFound`（404 `issue.not_found`）、`ActionRead`（`issue.read`）；`access` 的规则行（项目级，管理员、成员：P1b 的访客一律 403，访客规则在 P2）。前端：`error-messages.ts` 和两份 `errors.json` 的 `issue_not_found`（约束 4）。
- `issue/app`：`IssueReader`（`GetIssue`）；`Locks` 带上它（`NewLocks(projects, issues, auth, …)`）；`Locks.see(ctx, actor, id, action)`：不加锁地读工作项，判定它的项目：没有或看不到 `issue.not_found`，角色不允许 403；`decide` 带上 `notFound`。`GetIssue.Execute(ctx, id)`：`see` 一步，不取锁、不开事务、不进闸门。
- 契约：`GET /api/v0/issues/{issue_id}`（`IssueID` 参数，`x-problem-codes: [issue.not_found, forbidden]`），回答 `IssueDetail`；HTTP 的 `GetIssue`，`newServer(t, UseCases)`。
- 矩阵：种子由工作区的管理员经存储给每个项目写一个工作项 `matrixIssue`（项目的第一个，Todo、medium，指派 PM 和一个已离开工作区和它的公开项目的账户，标签 Bug，`permission_matrix_seed_issues_test.go`），`{issue_id}` 进 `underProject`；`getIssue` 两行（项目的各列；已归档的项目的工作项照样读）；`createIssue` 的格子的编号从 2 起。

### 2.7 `updateIssue` 的用例（Task 5；3.5、3.7、6.7）

`issue/app/lock.go`（整个文件）：
- `Locks{projects, issues, auth, tx}`（Task 10 加 `gate`），`write{project, issue *uuid.UUID, action, notFound}`，`held{project, issue}`。
- `run(ctx, actor, w, fn)`：`precheck` → 事务：`lockAndDecide` → `fn(ctx, held)`。每个工作项一侧的写都经它（`createIssue` 改到它上面，`Tx` 移进 `Locks`）。
- `precheck`：写指名工作项时 `see`（读工作项、判定它的项目），`w.project` 取工作项的项目；否则照 P1a。
- `lockAndDecide`：项目 N → 指名工作项的写在锁下重读它，没有或已不在这个项目时 `w.notFound` → 判定（最终的）。回答锁住的项目和锁下读到的工作项。

`writes.go`：每个写共用的 `stamp`（`links`、`unlinks`、`activity`）、`assignees`（显示名）、`readBack`。`update_issue.go`：`UpdateIssue.Execute(ctx, id, p)`：`CheckIssuePatch` → `run`：已归档 409 → `ChangesOf`，`None` 时不写、回答锁下读到的工作项 → `check`（状态改时：是项目的、不是分诊的，否则 `state_id` `not_allowed`；新加的负责人经 `Members.Roles` 和 `Unassignable`；加、去的标签经 `Labels`，加的不是项目的 `label_ids[i]` `not_allowed`；全部一次报出；顺便读出动态要的名称）→ `Clock.Now()` → 写：工作项（状态改时 `completed_at` 按新状态的组：完成组为这一刻，否则空）→ 负责人先去后加 → 标签先去后加 → 新负责人的关注 → 动态 → 在事务里读回。`IssueUpdater` 端口（嵌入 `IssueReader`；`UpdateIssue`、`RemoveAssignees`、`RemoveLabels` 和负责人、标签、关注、动态的插入）。`issue.update` 的规则行（管理员、成员）。

单元测试：`TestUpdateIssueWritesTheChangesAndTheirRows`（调用的次序、写下的行）、`TestUpdateIssueThatChangesNothingWritesNothing`、`TestUpdateIssueCompletesTheWorkItemByItsState`（已取消组的也不完成，从完成组移进去时不再完成）、`TestUpdateIssueRefusesBeforeAnyLock`、`TestUpdateIssueRefusesUnderTheLock`（锁下判定、已归档、锁下读不到工作项或它换了项目、依赖行的问题一次报出）、`TestUpdateIssueStopsAtAFailingPort`（每个端口的失败原样返回；不可能的情况也是错误）；领域的 `TestUnassignable`。

### 2.8 `updateIssue` 的接口（Task 6；5.1–5.3、6.6、9.2）

- 契约：`PATCH /api/v0/issues/{issue_id}`（`x-problem-codes: [validation_failed, issue.not_found, forbidden, project.archived]`，Task 10 加 `server_busy`），描述写明锁之前答的、锁下再判定的、什么都不改时不写、完成时间的转换、新负责人的关注；`IssueUpdate`（`additionalProperties: false`：没有 P1b 之外的字段）。
- HTTP：`UpdateIssue`（每个字段照原样交给用例，用例检查）；`update_test.go`（转换、体的形状、拒绝）。接线：`issue.New` 的 `UpdateIssue`。
- 矩阵（`permission_matrix_issue_updates_test.go`）：`updateIssue` 九行（只改名称、每个字段都给、什么都不改、已归档（什么都不改也 409：已归档的检查在算改动之前，第 3 节第 4 条）、另一个项目的状态、另一个项目的标签、分诊状态、指派访客、留下已离开的负责人）。
- `issueWrites` 加 `updateIssue` 的一行：`reads` 是 `states`（它在锁下最先读、在锁之前不读的表；`issues` 在锁之前读过，P1a 评审第 6 节）；每行发的路径是它的操作的（`pathOf`：一行发的路径不是它的契约路径时在发之前失败，不在一个操作的名下测另一个操作）；`issueWorld` 加 Done、`newWorkItem`，P1a 的 `TestCreatingAWorkItemStampsItsRows` 用它建工作项（`issue_writes_test.go`）。

### 2.9 盖戳、回答如存、事务的连接（Task 7；3.5，P1a 评审第 6 节）

- `TestUpdatingAWorkItemStampsItsRows`：alice 把 walt 加进 Web、建工作项（Todo，指派她自己，标签 Bug，`newWorkItemIn`），由 bob 改名称、优先级、状态（到 Done）、把负责人换成 walt、去掉 Bug：工作项和 bob 写的每一行（负责人的增删、标签的去、walt 的关注、六行动态）是 bob 在请求之内的同一时刻，bob 改了它也不关注它（3.12，第 3 节第 7 条）；alice 写的行保留她的戳；动态的值经 `users.display_name`；回答的创建者、最后修改者、两个时刻。工作项五张表的行由 `issueWorld.workItemStamps` 一次读出（创建的盖戳测试共用），修改者的动态由 `issueWorld.activityBy` 读（交错 1u 共用）。
- `TestAWorkItemReadsAsStored`：Web 的第二个工作项，bob 先把它移到 Done（完成）、再改一次，之后在数据库里给它父工作项（第一个）和归档时刻（P1b 没有归档的操作）：每个可为空的字段各有自己的值，创建、完成、最后修改、归档是依次的四个时刻；工作项的每个字段经 `getIssue` 照存储回答，组合一层看得到 `issueDetail` 的每个字段（P1a 只在 HTTP 一层）。
- `TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection` 加修改（完整的修改、拒绝、什么都不改）和读回。

### 2.10 `platform/gate`（Task 8；3.5、11.11，9.1）

`Keyed[K comparable]`、`Limits{PerKey, MaxQueue, MaxWait}`、`New[K](l)`、`Enter(ctx, key) (leave func(), err error)`、`ErrFull`、`ErrTimeout`。一把互斥锁下的 `map[K]*queue{holders, waiting []*waiter}`：有空位、没有人在等时立即进入；否则排在队尾（先来先得），队列已有 `MaxQueue` 个时立即 `ErrFull`；等到 `MaxWait` 时 `ErrTimeout`，`ctx` 结束时 `ctx.Err()`（进入之前已结束的也不进）；放弃的离开队列，正好在放弃时被交到位置的把位置交给下一个；`leave` 只生效一次（`sync.Once`）；没人持有、没人等的键不占内存。一个键的位置由 `release` 直接交给队首，所以有人等时没有空位（第 3 节第 11 条的等价变异）。测试（表和 `-race`；该进入的每个调用者都由 `held` 核对它拿到了位置）：`TestAKeyAdmitsPerKeyCallersAtOnce`、`TestWaitersAreServedInTheOrderTheyCame`（队列中间的人放弃之后，后面的照原来的次序：T8.13）、`TestAFullQueueRefusesAtOnce`（`MaxQueue` 0 和 2；`MaxQueue` 0 时第一个调用者照样拿到空着的键：T8.14）、`TestAWaiterGivesUp`（`MaxWait`、取消、期限、进入之前已取消；每种放弃之前先看到它在排队）、`TestLeavingTwiceLeavesOnce`、`TestUnderLoadAKeyIsNeverHeldByMoreThanPerKey`（32 个 goroutine 各 1000 次、三个键、随机的期限：不带 `-race` 每次也走到放弃时被交到位置的一支；每次 `Enter` 之后在闸门的锁下核对有人等的键没有空位，`idle`：T8.12）。

### 2.11 锁的期限和配置（Task 9；3.5、11.11，8.8）

- `platform/postgres/tx.go`：`Option`、`LockTimeout(d)`（`TxOptions.BeginQuery` 为 `BEGIN; SET LOCAL lock_timeout = N`，`N` 是向上取整的毫秒数：同一个往返，`fn` 的第一条语句之前）、`NewTxManager(pool, commitTimeout, opts...)`、`LockNotAvailable(err)`（`55P03`，与 `UniqueViolation` 同一种写法）。模块的 SQL 里没有 `SET`。测试 `TestLockTimeoutBoundsEachWaitOfTheTransaction`（`SHOW lock_timeout` 是 `80ms`，等锁约 80 毫秒之后 `55P03`、回滚；之后的连接回到默认，提交之后也是；1.2 毫秒取整为 `2ms`：向上取整，不是取最近的；不带选项的是 `0`）、`TestLockNotAvailable`。
- `platform/config`：`IssueConfig{WriteGate WriteGateConfig{PerProject, MaxWait, MaxQueue}, WriteLockTimeout}`（`koanf:"issue"`），校验：`per_project` 至少 1、`max_wait` 为正、`max_queue` 不为负、`write_lock_timeout` 为正，加上 `max_wait` 短于 `server.request_timeout`（在闸门上等到 `max_wait`、再等一次锁到期的写，仍要在请求到期之前答 503：Task 9 的修复轮把只看锁的期限的规则改成两者之和；规则只盖住闸门和一次等锁）；两个下限本身是允许的（`per_project` 1、`max_queue` 0，`TestValidateAcceptsTheWriteGatesMinimums`）；`LogValue` 记下四个键（测试的值各不相同，也不是别的键的值的前缀）；`configs/config.yaml` 的默认值和中文说明。

### 2.12 写闸门接入（Task 10；3.5、6.1、11.11）

- `issue/app`：`WriteGate` 端口（`Enter(ctx, projectID) (leave func(), error)`，第 3 节第 10 条）；`Locks.run`：预检 → `gate.Enter(ctx, w.project)` → `defer leave()` → 事务。预检拒绝的不进闸门；闸门拒绝的不开事务；离开在事务结束之后，每条路上都离开（拒绝、失败、panic）。`issue.Deps.WriteGate`。
- `bootstrap/gate.go`（新）：`writeGate`（`platform/gate` 按项目，`ErrFull`、`ErrTimeout`、`context.DeadlineExceeded` 换成 `shared.ServerBusy(1s)`，`context.Canceled` 照原样：没有人等它的回答）；`busyTx`（`issue` 的事务：`postgres.NewTxManager` 带 `LockTimeout(issue.write_lock_timeout)`，`LockNotAvailable` 换成同一个 503）；`newIssueTx`；`busyRetry`（1 秒）。这是 `55P03` 换码的唯一一处（设计 3.5）。`app.go` 把它们交给 `issue.New`。
- 契约：`createIssue`、`updateIssue` 声明 `server_busy`，描述写明"太多写时 503，按 `Retry-After` 重试"；`server_busy` 是 M2 的平台码，文案已在 `error-messages.ts` 和两份 `errors.json`（约束 4 不另加）。
- 测试：单元的 `TestAWriteEntersItsProjectsGateBetweenItsPrecheckAndItsTransaction`、`TestAWriteLeavesTheGateOnEveryPath`（锁下的拒绝、端口的失败、`fn` 的 panic，每条路上都在事务结束之后离开，panic 照样传给调用者）、`TestTheGateTakesOnlyWritesThePrecheckLetThrough`（`fakeGate` 记下进出，事务还没有结束时的离开记作 `in tx`：离开在提交、回滚之后），`TestGetIssueReadsTheWorkItem`（`newGet` 也用记下进出的闸门：读不进闸门）；`bootstrap` 的 `TestTheWriteGateAnswersItsRefusalsAsServerBusy`（一个位置被持时，队列满、等到 `max_wait`、等着时请求到期都是 `server_busy`，请求取消的照原样；这时另一个项目的写立即进入：每个项目有自己的位置）、`TestABusyTransactionIsServerBusy`（`55P03` 包了几层也换，`40P01` 不换）；组合的 `TestAProjectTooBusyForAWriteAnswersServerBusy`（一个项目一个位置、排队一个、`max_wait` 1 秒、锁的期限 3 秒：别的事务持 Web 的行时，alice 的修改进了闸门、等行；随后的两个创建一个排队、到 `max_wait` 时 503，另一个在 `max_wait` 之前 503（队列满）；olga、walt 的修改在位置被占时由预检答 404、403，都在 `max_wait` 之前：预检拒绝的不占闸门；alice 在 Ops（acme 的另一个项目）的创建这时也在 `max_wait` 之前答 201：每个项目有自己的位置；alice 的修改在约 3 秒（等锁到期）503，远早于请求的期限；行放开之后修改和创建都通过）；组合的 `TestAWriteWhoseRequestEndsWhileItWaitsAtTheGateIsServerBusy`（请求的期限 3 秒、锁的期限 2 秒、一个项目一个位置、`max_wait` 10 秒：别的事务以 `ACCESS EXCLUSIVE` 持 `issue_labels`，bob 的修改在预检里等它（预检读工作项的标签）；2 秒之后 alice 在 Web 的创建（预检不读 `issue_labels`）占了位置、等 Web 的行；`issue_labels` 放开之后 bob 的修改在闸门上排在她后面，在他的请求的期限（3 秒）503，她的创建在等锁到期时（bob 的修改发出之后约 4 秒，比她自己的请求的期限早约 1 秒）503：等闸门的写随它的请求结束；bob 的修改在 3 到 4 秒之间：她最早在 4 秒离开闸门，T8.7、T10.16 的 bob 只在那之后得到回答）；HTTP 的两行（503、`Retry-After: 1`，`apitest.Main` 两个方向）；`archtest`。`issueWorld` 由 `newIssueWorldWith(t, configure)` 改配置；`testIssueConfig` 是闸门的默认值和 4 秒的 `write_lock_timeout`（短于测试的请求期限 5 秒）：只给测试用的值，配置的加载会拒绝它（`max_wait` 加锁的期限不短于请求的期限，2.11），测试的世界不校验配置（第 3 节第 15 条）。

### 2.13 交错 1u、3u–7u（Task 11；3.5，9.3）

`issue_interleaving_test.go` 的 `itemWrite`、`itemWrites`（"created"：Web 里的创建；"changed"：改 alice 的 Fix，`aWorkItem`、`workItem`、`changesIssue`）、`after`、`wroteNothing`、`workItemRows`：P1a 的交错测试各加修改的一支，不另写一份。同一个工作项的两个修改（1u）和等锁期间工作项被删除或移走的修改在 `issue_interleaving_change_test.go`（第 3 节第 16 条）。`interleaving_writes_test.go` 的 `TestTwoWritesOnAProjectSerialize` 改用 `serializeOn`（P1a 交给 P1b 的一条）。

| 交错 | 测试 | 修改一支的断言 |
|---|---|---|
| 1u | `TestTwoChangesOfAWorkItemSerialize` | alice 的修改（改名、加 Bug）先持 Web 的行；bob 的修改按改之前的工作项过了预检、等行，她提交之后在锁下重读：动态写"Mend → Repair"、去掉她加的 Bug，时刻不早于门打开 |
| 等锁期间工作项不在了 | `TestAWorkItemGoneWhileItsChangeWaitsIsNotFound`（删除、移到 Ops，由持 Web 的行的事务在数据库里做：P1b 没有这两个操作） | 修改过了预检、等行；提交之后锁下重读找不到 Web 里的工作项：404 `issue.not_found`，没有写 |
| 3u（和 2 的修改一支） | `TestAWorkItemsWriteAndDeletingItsProjectOrWorkspaceSerialize`（Web、acme，各两种顺序；写的是 bob，删除的是 alice） | 先改：Web 的工作项一侧的九行（alice 的 Fix 的三行，bob 的修改写下的六行）随项目或工作区删除，九行都由删除记在 alice 名下（bob 写过的行原来记在他名下），删除的时刻不早于门打开；先删：404 `issue.not_found`，没有写 |
| 4u、5u | `TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused` | 预检放行之后被移出：404 `issue.not_found`；被降为访客：403；都没有写 |
| 6u | `TestDeletingAStateAndPuttingAWorkItemInItSerialize` | 状态先删：422 `state_id`；先改到它：409 `project.state_in_use`，状态保留 |
| 7u | `TestDeletingALabelAndLabellingAWorkItemWithItSerialize` | 标签先删：422 `label_ids[0]`；先加：Bug 在删除的时刻由删除者摘下 |

### 2.14 交错 20：饥饿与连接池（Task 12；3.5，9.3，11.11）

`issue_pool_test.go`：`newPoolWorld`（池 10 个连接，`database.max_conns` 的默认值；闸门是 `testIssueConfig` 的默认值，`configure` 再改它）、`reply`、`replyTo`。
- `TestChainedWorkItemWritesHoldOffNoWorkspaceWrite`（S5 的回归）：acme 的四个项目各一个工作项、20 个标签，16 串首尾相连的修改（每次换上或去掉 20 个标签）；每串都有一次修改得到回答之后（5 秒为限），alice 改 acme 的名称、把 walt 改为管理员、删除 acme，各在 1 秒之内；每串的修改都答 200，直到 acme 删除之后答 404 `issue.not_found`。
- `TestAHotProjectHoldsUpNoReadOfAnother`（SP1 的回归）：这个测试的 `max_wait` 是一分钟（它断言闸门的位置和队列的容量，不断言等多久）。别的事务持 Web 的行至少一秒、直到 134 个 503 都答了（3 秒为限），期间 200 个修改同时打到 Web：闸门放两个进数据库（等行），64 个在进程里等，其余 134 个立即 503 `server_busy`（`Retry-After: 1`）；Ops 的工作项和 Web 的这个工作项每 20 毫秒各读一次，至少 20 次，每次 200、不超过 `poolReadBound`（500 毫秒，第 3 节第 13 条；读不进闸门）；行放开之后等着的 66 个都答 200。

### 2.15 规模的测量（Task 13；决策点 7，9.3）

`issue_scale_test.go` 的 `TestTheDeletionsAtTheScaleTarget`，只在设了 `NERVE_MEASURE_SCALE` 时运行（`make test` 和持续集成跳过它，第 3 节第 14 条）：Web 和另一个工作区 beta 的五个项目，每个项目 2 万个工作项，每个工作项一个负责人、一个标签、两个关注、20 行动态（每个项目 50 万行，共 300 万行，`id` 按随机的次序，一个事务种下，`VACUUM ANALYZE`；种下之后核对未删除的行数是 6 × 50 万，不对时失败）；先在回滚的事务里量工作项一侧的软删除（`DeleteProjectIssues`、`DeleteWorkspaceIssues`），再经接口删除 Web、删除 beta，量请求；最后核对六个项目没有未删除的行。这个 app 的请求、写和提交的期限和访问令牌的有效期都是一小时；单独运行，从 `server/`、`go test -count=1`、`-timeout 60m`。数字在附录 A.4。

### 2.16 端到端（Task 14；第 2 节 I2，3.7、3.13）

- 夹具：`e2e/fixtures/issues.ts` 的 `IssueUpdate`、`issueUpdated`（状态和工作项或问题）、`updateIssue`（要求 200）；`assert/issue.ts`：动态的行带被改的名称或账户（`named`、`was`），`activityOf`、`rowsOf`（未删除的）、`rowStamps`（共用的戳的读）；`UpdatedIssue` 和 `expectIssueUpdated`：工作项的名称、状态、优先级、完成时间、最后修改者，负责人、标签、关注的集合，动态（创建的行和每次修改的行），以及最后一次修改写下的每一行（`updated_at` 等于工作项的那些）恰是 `wrote` 行、都是修改者在那一刻写的。I1、I2 共用期望的动态行：`byId`（按 `id` 排序）、`createdFor`（创建的行）、`assigned`、`changed`（修改的一行），`was: null` 在它们里面。
- I2 的接口版本（`i2-update-issue.spec.ts`）：改名称；优先级改为 high 并移到 Done（完成）；换一个负责人和一个标签、另一个负责人和标签留下（由不是这些行最后写入者的账户改：去掉的行的戳是他的；一次写九行）；什么都不改（集合按回答的反序给出；不写、`updated_at` 不变）；再指派、再加标签（两个标签按 `id` 的反序给出，动态按 `id` 的次序）；只去掉一个标签、只去掉一个负责人（各是一次修改，先前去掉的行保留那次的戳）；移回 In Progress（不再完成）；值的问题（大小写不对的优先级、给两次的账户和标签）在读工作项之前一次报出 422，发给一个不在的工作项也一样（不是 404），项目的问题（没有的状态和标签）在锁下一次报出 422，都什么都不写；与它共有负责人和标签的另一个工作项始终如创建时。页面版本在 P14。

### 2.17 文档（Task 15；3.23、8.8）

总体设计 4.2（写闸门、`lock_timeout`、503、连接池的代价）、6.2（`platform/gate`）；差异清单第四节 4.17 中 P1b 的三行（一个项目上的并发写、修改的响应、负责人的读取）；README 的部署一条（四个配置，8.8）；两份交接写下"处理结果（M4/P1b）"（第 7 节）。

## 3. 与设计的差异和补充

第 1 条是控制者的拆分裁定允许的重新切分；第 2–16 条是本 spec 对设计交给 P1b 的事的决定，控制者都照建议裁定（2026-10-18，`$M4TMP/p1b-architect-rulings.md`；第 13 条的计时在执行之前加固，A.2）；第 17 条是承接条目的落点。

1. **重新切分（裁定 3）**。设计 12 节 P1b 的十二个草稿任务与 plan 的十五个 Task：草稿 1 是 Task 1；草稿 2（修改的存储）是 Task 3，它要的目录的名称先在 Task 2（动态写名称，第 9 条）；草稿 3 是 Task 4；草稿 4 是 Task 5；草稿 5 拆成 Task 6（接口、矩阵、取锁形状的一行）和 Task 7（盖戳、回答如存、事务的连接：组合一层的三个测试）；草稿 6 是 Task 8；草稿 7 拆成 Task 9（`platform/postgres` 的选项和配置，没有使用者也独立可测）和 Task 10（接入）；草稿 8–12 是 Task 11–15。每个 Task 在 plan 里的行数见附录 A.1，最大的是 Task 5。
2. **`IssueUpdate` 的字段**（裁定：照本条）。设计 5.2 的 `IssueUpdate` 有名称、状态、优先级、负责人、标签、父工作项、排序值、描述、起止日期。P1b 只有前五个：描述和 `skip_activity` 在 P4、起止日期在 P2（约束 5）；父工作项要不成环的检查（3.9），在 P3；`sort_order` 的修改是列表里的拖动（I6，P5），没有 P1b 的读者。`additionalProperties: false`：带别的字段 400（`TestUpdateIssueHoldsTheBodyToItsStructure`）。建议照本条。
3. **负责人的检查只看新加的**（裁定：照本条）。设计 3.6 说离开了项目、成为访客的人"不能被新指派"，负责人的读返回全部未删除的行。修改给出的新集合里，工作项已有的负责人照旧保留，不再查他现在的角色（`Unassignable` 只查 `Added`；矩阵"an assignee who left kept"一行）；否则一个人离开项目之后，改这个工作项的任何负责人都要先把他去掉。标签不同：工作项上的标签都是项目未删除的标签（删除标签时摘下，`DetachLabel`），所以只查新加的。建议照本条。
4. **什么都不改的修改不写**（裁定：照本条）。给出的字段都与工作项的相同（含给出相同的集合）时，不写任何行：不改 `updated_at`、不写动态，回答锁下读到的工作项；检查和拒绝照旧（已归档 409 在这之前）。与 3.5 表中"归档：已是归档：不写，200"同一种写法，也与 I2 的"不做改动：没有新的动态"同一个方向（M1-P2 交接，那里说的是描述的块 id）。建议照本条。
5. **完成时间只在状态改变时转换**（裁定：照本条，这就是设计 3.7）。状态改到完成组时 `completed_at` 是这一刻，改到别的组时为空（3.7；Plane 同样只在状态改变时同步它，`plane/apps/api/plane/db/models/issue.py:240-250`）；从完成组改到另一个完成组的状态也写新的时刻；状态不变（含给出相同的状态）时不动。建议照本条。
6. **动态的行和次序**（裁定：照本条）。3.13 的表的次序：名称（3）、优先级（6）、状态（7，`old_value`、`new_value` 是名称，`old_identifier`、`new_identifier` 是 `id`）、加的标签（10）、去的标签（11）、加的负责人（12）、去的负责人（13），每个集合按 `id`；`comment` 照 Plane 的文字。一次修改的全部行同一时刻、同一个执行者，`id` 递增（`stamp.activity`）。建议照本条。
7. **关注只加新的负责人**（裁定：照本条）。3.12：实际被指派的人关注工作项；修改者不因修改而关注（3.12 的"不是每个改过工作项的人"：Plane 让动态的执行者关注，已登记为差异）。`InsertSubscribers` 的 `ON CONFLICT DO NOTHING`：已关注的不再加。建议照本条。
8. **`issue.read` 与 `issue.update` 的规则**（裁定：照本条）。两行都是项目的管理员、成员（3.4）；P1b 的访客一律 403（矩阵的访客格子），访客规则在 P2（设计 12 节 P2 的第 10 个任务"交错 5 加上访客规则下 404 的一支"）。`getIssue` 对已归档的工作项照样回答（P1b 没有归档，P2 加）。建议照本条。
9. **状态、标签的名称在 P1b 进目录**（裁定：照本条；P1a 第 5 节 P5 一行的"状态的名称"由此做完）。P1a 的第 5 节把"状态的名称进 `StateCatalog`"放在 P5（列表的分组）；P1b 的动态第 7、10、11 行先要名称，所以 Task 2 把两个目录改为回答名称，端口的形状不变，只多一列。P5 照用。建议照本条。
10. **`WriteGate` 只取一个键**（裁定：照本条，第 5 节 P3 的一行是要接的交接；偏离设计 3.5、6.5、6.7、11.11 的 `Enter(ctx, projectIDs)`，设计在 3.5、6.5、9.1、11.11 写明 P1b 一次一个项目）。P1b 的每个写只在一个项目上；按 `id` 升序一次取齐几个闸门、任一个失败时放回已取的，是 P3 的关联（约定七）才有的写法。P1b 的端口是 `Enter(ctx, projectID) (leave func(), error)`，`Keyed.Enter(ctx, key)` 一次一个键；P3 加多个键的取法（在 `Keyed` 上，或在 `bootstrap/gate.go` 的 `writeGate` 里按 `id` 升序逐个进入、失败时倒序离开），端口随它改成多个项目。理由同 P1a 第 3 节第 5 条（端口只带它的读者用的东西）。建议照本条，P3 的 spec 接过去（第 5 节）。
11. **`platform/gate` 的一处冗余条件**。`Enter` 的"有空位、**且没有人在等**才立即进入"里的后半句由不变式保证：`release` 把位置直接交给队首、不腾出空位，调用者只在位置全被持有时排队，所以有人等时一定没有空位。去掉它的变异 T8.11 是等价的（附录 A.3），留着它是为了把"先来先得"写在进入的地方，不依赖读者推出不变式。
12. **约定五**（M3 交接第 2 节，逐个核对）。P1b 的批量写：`RemoveAssignees`、`RemoveLabels`（按谓词软删除一个工作项的若干行）、`InsertAssignees`、`InsertLabels`、`InsertSubscribers`（按 `id` 的顺序插入，`diffOf` 排序）。每个都在项目 N 之下，这些行的每个写者（工作项的写、删除标签的 `DetachLabel`、删除项目和工作区的连带）都先持这个项目的 N（3.5"删除标签摘下 `issue_labels` 不需要按 `id` 先锁"一条的论证），所以不落在约定五的例外里，不先按 `id` 锁；插入的部分唯一键冲突在 N 之下不会互相等。没有新的成员关系增长（指派、关注不让谁成为成员）。
13. **交错 20 的上限：500 毫秒**（裁定：上限照本条；计数在执行之前改成按条件，见本条的"持续集成"）。设计 9.3 说上限"在 P1b 的 spec 按实测定"。本机（macOS、Docker Desktop、8 GB）上，有闸门时另一个项目的读最慢 2–49 毫秒（45 次运行，带和不带 `-race`）；把闸门去掉（每个写不进闸门）时最慢 1.12–1.26 秒（一版测试的三次），另一版的三次都是约 4.97 秒（等到请求的期限）。500 毫秒比有闸门的最坏约宽 10 倍，比没有闸门的最好约窄 2 倍。改工作区、改角色在链上 3–9 毫秒，删除工作区 17–54 毫秒，上限 1 秒（附录 A.4）。预检之后加固的测试（附录 A.2）在本机带和不带 `-race` 各跑 5 次：读最慢 3.0–41.4 毫秒，删除工作区至多 74 毫秒（附录 A.4）。**持续集成**：计数不靠固定的时间（裁定 13）：Web 的行持到至少一秒、并且 134 个 503 都答了（3 秒为限，到了算失败），这个测试的 `max_wait` 是一分钟，所以 134 由构造成立；66 个在行放开之后都答 200，它们真正的界是请求和客户端的 5 秒期限（Task 12 的评审：强制持到 2.9 秒时，最慢的一个在 3.13 秒答）。固定的至少一秒的持有让在被持的行上等的读超过上限（读也取项目行的锁的变异 T4.8、T4.9 因此失败），也让读至少有 20 次；没有闸门的变异的读因池满而超过上限，那时 134 个 503 不会来，持有无论如何持到 3 秒。执行中的运行见附录 A.4。链在每串都有一次修改得到回答之后才开始测工作区的写；两个忙的项目的测试的窗口以期限为界（忙的项目：`max_wait` 1 秒、锁的期限 3 秒；请求在闸门上到期：bob 的请求的期限 3 秒，alice 的等锁最早在 bob 的修改发出之后 4 秒到期），而不是几百毫秒的余量。
14. **规模的测量的落点**（裁定：照本条）。D7 要求 P9 在同样的数据上再量。建议：测量提交进仓库，作为 `bootstrap` 的一个测试，只在设了 `NERVE_MEASURE_SCALE` 时运行（`make test`、持续集成跳过它，但它随代码编译、过 lint，接口变了会被发现）：P9 改成任务之后，在同一个测试里加任务的测量，种子和数字可比。另一种是放在 scratch（不进仓库）：P9 要重写种子，两次的数据可能不同。数字见附录 A.4。
15. **测试的 `write_lock_timeout` 是 4 秒**（裁定：照本条）。测试的配置（`testConfig`）的请求期限是 5 秒，锁的期限要短于它，默认的 5 秒放不进去。`testIssueConfig` 用 4 秒，其余三个是默认值。这些是只给测试用的值：Task 9 的修复轮把规则改成 `max_wait` 加锁的期限短于请求的期限（2.11）之后，配置的加载会拒绝它（2 秒加 4 秒不短于 5 秒），而测试的世界从不校验配置（校验只在 `config.Load` 里）；交错 20 的两个测试和取锁的测试照默认值的闸门跑。
16. **按资源寻址的写在 P1a 的路径上**（P1a 第 5 节）。`write` 带 `issue`，`precheck` 读它（`see`）、得到项目，`lockAndDecide` 在锁下重读它并确认仍在这个项目；`getIssue` 用同一个 `see`（读和预检同一个判定）。只有一条取锁路径；`createIssue` 也改到 `run` 上（事务移进 `Locks`），它的测试的调用次序不变。`updateIssue` 不嵌套 `WithinTx`，交错的 `heldTx` 每个写只开一个事务（P1a 评审第 6 节）。锁下重读决定修改从什么改起：设计 3.5 的交错表里同一个工作项的两个修改没有一条，本 spec 补上交错 1 的修改一支（1u，`TestTwoChangesOfAWorkItemSerialize`）：第二个修改等过锁之后按第一个留下的样子算差异，时刻在它之后；等锁期间工作项被删除或移到别的项目（P1b 没有这两个操作，测试由持项目行的事务在数据库里做）时锁下重读答 404（`TestAWorkItemGoneWhileItsChangeWaitsIsNotFound`）。"锁下重读"的三种性质因此在组合一层都有反例（附录 A.2）。
17. **承接的条目**：

    | 来处 | 条目 | 落点 |
    |---|---|---|
    | P1a spec 第 5 节、评审第 6 节 | 取锁的一步：锁下带上工作项；按资源寻址的读和锁下重读在同一条路径上，`write` 带上它的读 | 2.7、第 16 条 |
    | 同上 | 写闸门插在预检之后、事务之前 | 2.12 |
    | 同上 | `server_busy` 在每个工作项的写上声明 | 2.12 |
    | 同上 | `issueWrites` 每个写一行，`reads` 不是 `issues`；`writtenTables` 照样约束 | 2.8（`updateIssue` 的 `reads` 是 `states`） |
    | 同上 | `heldTx` 每个写一个事务；嵌套时跳过门 | 第 16 条（不嵌套，不必改 `heldTx`） |
    | 同上 | `TestTwoWritesOnAProjectSerialize` 手写的门收进 `serializeOn` | 2.13 |
    | 同上 | `issueDetail` 的最后修改者、时刻、父工作项、归档时刻在组合一层看得到 | 2.9（`TestAWorkItemReadsAsStored`，变异 T7.2、T7.3） |
    | 同上 | 端口、领域、契约的延续；`expectIssueUpdated` 照 `expectIssueCreated` 写 | 2.3–2.8、2.16 |
    | 同上 | 规模：在目标规模上量请求的耗时 | 2.15、附录 A.4 |
    | 同上 | 关键词守卫的 `phase` 照旧 `M4/P1` | `tools/keywords.json` 不改 |
    | M2 收尾交接第 3 节 | 修改用时钟写 `updated_at` | 2.7、2.9 |
    | M3 收尾交接第 2 节 | 连接池的代价和写闸门；P1b 的批量写按约定五；没有新的增长 | 2.12、2.14、第 12 条 |
    | P1a 的教训（brief） | 数据分不出性质、复制的测试帮手、任务结束时不成立的注释、给出没人能做的补救的文案、没钉住的 SQL 谓词、后面的 Task 才发现的变异、容器起不来不算发现 | 附录 A.2、A.6 |

## 4. 验收标准（完成线，M4 设计 12 节 P1b）

- [ ] I2 的接口版本（名称、状态、优先级、负责人、标签的修改和动态、`completed_at`、`updated_at`、什么都不改、只去掉一个、两种 422、不被碰的另一个工作项）通过；`make e2e` 122 个：此前的 121 个，加 I2。
- [ ] `getIssue`、`updateIssue` 的矩阵格子通过。
- [ ] 交错 1u、3–7 的修改一支、等锁期间工作项不在了的修改和交错 20 `-count=5 -race` 通过，没有 40P01；P1a、M3 的交错照旧通过（附录 A.4）。
- [ ] 闸门满时、等到 `max_wait` 时、请求在闸门上到期时 503 带 `Retry-After: 1`，`lock_timeout` 到时 503（`TestAProjectTooBusyForAWriteAnswersServerBusy`、`TestAWriteWhoseRequestEndsWhileItWaitsAtTheGateIsServerBusy`、交错 20）。
- [ ] 规模目标上的删除耗时写进本 spec（附录 A.4）。
- [ ] `make gen-check`、`make lint-go`、`make test`、`make lint-web`、`make knip`、`make test-web`、`make e2e` 通过。

## 5. 不在 P1b 范围内

- 描述、`skip_activity`（P4），起止日期、访客规则、删除和归档工作项、按编号取（P2），父工作项和关联（P3），列表和 `sort_order` 的修改（P5），删除项目和工作区的任务（P9），I2 的页面版本（P14）。

**交给后面的 Phase**：

| Phase | 条目 |
|---|---|
| P2 | 访客规则：`issue.read`、`issue.update` 的访客（P1b 一律 403）；`see` 和预检照样在锁之前答访客规则的 404；已归档的工作项的修改 409 `issue.archived`（3.5 的"修改工作项"一行的"工作项已归档 409"，P1b 没有归档）；起止日期进 `IssuePatch`（合并之后的值，3.7）；`project.state_in_use` 的补救（P1a 交给 P2 的一条照旧：修改在 P1b 有了，恢复在 P2） |
| P3 | `WriteGate` 取多个项目（按 `id` 升序一次取齐、失败时放回，第 3 节第 10 条；进第二个闸门时不持连接、不持锁（设计 11.11），只持着前一个闸门的位置）；`parent_id` 进 `IssuePatch` |
| P4 | 描述、`skip_activity` 进 `IssuePatch`；动态的第 4 行（描述，合并）；I2 的描述的断言 |
| P5 | `sort_order` 的修改；状态的名称已在目录里；工作项一侧的快照事务把隔离级别和锁的期限放在同一个 `BeginQuery` 里（`BEGIN ISOLATION LEVEL REPEATABLE READ; SET LOCAL lock_timeout = …`）：pgx 的 `BeginQuery` 盖过 `TxOptions` 的 `IsoLevel`、`AccessMode`，只加 `postgres.LockTimeout` 会悄悄丢掉隔离级别 |
| P9 | 规模的测量在同一个测试里加任务的一支（第 3 节第 14 条）；同步连带的两个方法删去之后这个测试量任务；在规模的数字旁再量第 7 节连接池的情形（删除一个大的工作区、写在它的项目上等行时，另一个项目的读）；连带进了 River 的任务之后，已删除的项目上的写应在预检答出、而不是在它的行上等，由 P9 核对；规模的测量的 `workItemRows` 每次计数都在 `pgtest.Soon`（5 秒）之下（最大的一次数 240 万行动态），再量时给计数一个自己的 `context` |
| P14 | I2 的页面版本 |
| 收尾 | 3.23、8.8 中 P1b 的行由 Task 15 写好，收尾逐行核对；M8 的部署交接写明连接池的大小与多进程时的 K、同时热的项目数，以及一个工作项的写的等待：闸门上至多 `issue.write_gate.max_wait`（或到请求的期限），数据库里每一次等锁各至多 `issue.write_lock_timeout`，同时热的项目达到 ⌈池 / K⌉ 个时另有池里的等待，只以请求的期限为界；配置的规则（`max_wait` 加 `write_lock_timeout` 短于 `server.request_timeout`）只盖住闸门和一次等锁，等到请求的期限的写答 500（设计 13.2；第 7 节）；P1b 之前的 14 个测试文件里手写的 17 处 `pgErr.Code == "55P03"` 改用 `postgres.LockNotAvailable`（identity 的 `provide`、`admin`、`accounts`、`credentials`，workspace 的 `directory`、`directory_share`、`update_invitation`、`locks`、`deactivation`，project 的 `state_writes`、`demote`，bootstrap 的 `project_write_locks`、`demotion`、`interleaving_growth` 的测试），P2 起新的测试用平台的帮手（`postgres.LockNotAvailable`、`postgres.LockTimeout`），不再手写 |

## 6. 风险

| 风险 | 应对 |
|---|---|
| 闸门在持连接或持锁时等 | 进入在预检之后、`WithinTx` 之前（单元的调用次序，T10.1、T10.3 在单元和交错 20 失败）；离开是 `defer`（T10.2）；多个闸门在 P3（第 3 节第 10 条） |
| 闸门的位置泄漏 | 每条路上离开（拒绝、失败、panic，`TestAWriteLeavesTheGateOnEveryPath`）；放弃时正好被交到位置的交给下一个（负载测试，带和不带 `-race`，T8.5）；`leave` 只一次（T8.3） |
| 闸门的界：提交之前让出位置、读占位置 | 离开在事务结束之后（单元的 `fakeTx` 记下事务里的离开，T10.17，按性质只在单元一层，A.3）；读不进闸门（`newGet` 的闸门记下进出，交错 20 读热的工作项，T10.18） |
| 503 的换码散在各处 | 只在 `bootstrap/gate.go`：闸门的拒绝和 `55P03`；单元的两张表、组合的 `TestAProjectTooBusyForAWriteAnswersServerBusy`、`TestAWriteWhoseRequestEndsWhileItWaitsAtTheGateIsServerBusy` |
| 等闸门的写不随它的请求结束 | 进入闸门用请求的 `context`（T10.16），请求在闸门上到期答 503（T10.5、T8.7），都在组合一层的 `TestAWriteWhoseRequestEndsWhileItWaitsAtTheGateIsServerBusy` 失败 |
| 预检与锁下的判定、锁下的重读不一致 | 锁下的判定和重读是最终的，修改从锁下读到的样子算起（交错 1u、3u、4u、5u；T5.15–T5.18、T5.20） |
| 交错 20 的时间在持续集成上 | 条件式的持有和窗口（第 3 节第 13 条，裁定 13） |
| 同步的连带在规模目标上超过期限 | 已知（决策点 7，附录 A.4）：P9 换成任务 |
| 持续集成（ubuntu）上的运行 | 原型和复现都在 macOS 上；合并之后看持续集成 |

## 7. 已知的限制、交接和关闭条件

- **规模目标上删不掉一个工作区**（决策点 7 的已知，附录 A.4）：五个 2 万个工作项的项目的工作区，请求约 23–27 秒，超过生产的 15 秒期限，到期时回滚、什么都没删；一个这样的项目约 4.4–4.8 秒。P9 换成任务。
- **K 是每个进程的**（11.11）：多个进程时每个进程各 K 个；连接池的大小与 K、进程数怎样配合交给 M8 的部署说明（设计 13.2，M4 收尾写进 M8 的交接）。闸门限的是每个项目的份额：一个进程里 ⌈池 / K⌉ 个项目同时热（默认 5 个；规模目标上删除 beta 的约 25 秒里它的五个项目上都有写）就能占满池，M8 按同时热的项目数定池的大小。
- **热的项目上超出的写得到 503**（11.11 的代价）：客户端按 `Retry-After` 重试；P1b 之前它们会一直等到请求的期限。

| 交接 | P1b 处理的条目 | 留下的条目 |
|---|---|---|
| M2 收尾交接第 3 节 | 修改用时钟写 `updated_at`（创建在 P1a） | P2–P4、P7 的其余写工作项的用例；自动归档推进固定时钟的测试（P6） |
| M3 收尾交接第 2 节 | 连接池的代价和写闸门；P1b 的批量写按约定五核对；没有新的增长 | 加关联的另一端的预检（P3）；之后各 Phase 的批量写（到 P9）；物理删除账户（P9） |

两份交接的"处理结果（M4/P1b）"由 Task 15 写下，`status` 照旧 `open`，在 M4 收尾随 review 关闭（P1a 的做法）。

**M4 设计 13.1 的关闭条件**：落在 P1b 的两项（M2 交接第 3 节修改的部分，M3 交接第 2 节的连接池、写闸门和 P1b 的批量写）都在上表。没有放不下的条件。

## 附录 A：原型验证记录（2026-10-18）

### A.1 做法和 Phase 的大小

原型在 `$M4TMP/p1bproto`：`5ce30a1a`（P1a 的合并提交）的一份 `git archive`（`p1btools/base-5ce30a1a.tar`，快照 `p1bsnap/T0`）之上逐 Task 做，每个 Task 做完存一份源文件的快照（`p1bsnap/T1`…`T15`）；`node_modules` 由 `pnpm install --frozen-lockfile` 装好（不用符号链接），`bin/golangci-lint` 2.13.2；Go 1.27.1（`toolchain`）、Node 24.15.0、pnpm 11.10.0、Docker 29.7.2。plan 的代码块由脚本从相邻两份快照的差异生成（`p1btools/mkblocks.py`），每个块从基点起按顺序核对过（`problems: 0`）；生成的文件不进块，按 SHA-256 和行数钉住（`gensha.py`），plan 在每次生成之后有一步 `shasum -a 256` 与表核对；`compose_plan.py` 组装 plan，核对每个块放了一次、每个文件在文件表里（`filetablecheck.py`）、没有遗留的标记，并从变异的结果写出每个 Task 的变异表（被发现的层、"（Task n 起）"）。之后的修改（A.2）都写回拥有那个文件的 Task 的快照，从那个 Task 起每一份快照都改（`p1btools/edits/` 的编辑说明），`mkblocks.py` 重新生成全部块。预检之后的修改（A.2）同样写回快照；随后的逐 Task 复现（A.5）在每个 Task 之后把复现的树存成快照（`replay.py`：它取源文件的列表 `mkblocks_lib.listing`，列表不收的两项基点的条目，`web/packages/i18n/locales` 的目录链接和 `server/internal/platform/webui/dist/.gitkeep`，没有块碰它们，由 `snapcomplete.sh` 从复现的树补上），十六份与直接改过的快照逐个文件相同（`snapcmp.py`；`snapcomplete.sh` 按 rsync 比较内容、链接和两边各有的文件），`p1bsnap/T0`…`T15` 换成它们（直接改过的一份在 `p1bsnap-amend-direct`，预检之前的在 `p1bsnap-a04d479d`）。

**切分**（裁定 3 允许在上限之内重新切分；第 3 节第 1 条）：设计草稿的十二个任务成为 15 个 Task：目录回答名称提成 Task 2（修改的动态写状态、标签的名称，第 3 节第 9 条）；草稿 5 拆成接口（Task 6）和组合一层的三个测试（Task 7）；草稿 7 拆成 `platform/postgres` 的选项和配置（Task 9）与接入（Task 10）。没有用到退路。

**Phase 的大小**（裁定 3：约 16 个任务、每个任务约 1,500 行以内）：15 个 Task。执行之前 plan 8,885 行（`wc -l`；预检之前 8,769 行），每个 Task 都在约 1,500 行以内，最大的是 Task 5（1,484 行：`Locks.run` 的整个文件和 `updateIssue` 的用例、单元测试），其次 Task 10（1,007 行）、Task 4（1,005 行）。各 Task 从它的标题到下一个标题（Task 15 到文件末尾）：1 495、2 438、3 333、4 1,005、5 1,484、6 748、7 353、8 495、9 472、10 1,007、11 605、12 327、13 229、14 623、15 126，Task 1 之前（开头的约束、脚本和文件表）145 行。执行完的 plan 9,757 行（块写的是提交的代码，各修复轮的测试和变异表执行中加的行都在里面，评审记录的提交加的 T9.17 一行也在）：同样量，1 549、2 438、3 337、4 1,091、5 1,619、6 794、7 521、8 537、9 523、10 1,109、11 705、12 327、13 233、14 702、15 126，Task 1 之前 146 行；Task 5 长过了约 1,500 行（它的两个修复轮的单元测试和代码，变异表执行中加的十行），其余都在以内。P1 的规模报告估 P1b 约 12 个任务、9,500–11,500 行。手写的文件 81 个（新建 24 个、修改 57 个：原型的 79 个，加上 Task 4 的修复轮改的 `permission_matrix_project_test.go` 和 Task 11 改的 `interleaving_test.go`），生成物 6 个，文档 8 个（2.1），另有设计第 15 节的一行、3.5、8.8、11.11 和 13.2 的连接池的总代价（预检的 M4）和整分支评审的修复轮改的几句。切分时没有超出，没有停下。

### A.2 清扫之后的修改和加强，预检之后的修改

**清扫之后的修改**（变异按 Task 跑完之后复审时发现，改在拥有那个文件的 Task 的快照上，从那个 Task 起每一份快照都改，`p1btools/edits/` 的编辑说明逐条记下，受影响的变异重跑）：

- 往后看、说得比代码宽的注释和描述（清扫 6、53，`prophecy.py` 列出每个 Task 加的这类行）：`patch.go` 的"给出空的优先级"说成"被拒绝，不当作不改"（`t1a`）；`issue.read` 的规则和矩阵的说明从"直到 P2 加上访客"改成"访客是 P2 的"（`t4f`）；`updateIssue` 的契约描述从"在完成组的状态里就完成"改成"移到完成组的状态时完成、移到别的组时不再完成"（`t6d`，完成时间只随状态的改变而变，第 3 节第 5 条）；HTTP 测试的"后面的 Phase 带来的字段"改成"P1b 的 `IssueUpdate` 没有的字段"（`t6e`）；盖戳测试的"接口还做不到"改成"P1b 没有归档工作项的操作"（`t7b`）；规模测试的"P9 在同样的数据上量它的任务"改成设计决策点 7 的要求（`t13c`）。
- 数据分不出的性质（清扫 51）：标签目录的存储测试原来按一个次序比较回答，而查询不保证次序；改为当作集合比较，说明写明修改按 `id` 找每个标签（`t2b`）。修改的存储测试加一段：移回不在完成组的状态、完成时间给 `nil` 时不再完成（`t3b`，T3.3）。
- 锁的期限的测试（Task 9）：表的两行换了次序，提交过的事务之后再核对默认值，"会话级的 `SET`"（T9.1）因此被发现（`t9c`）；第一个事务放在 3 秒的期限上，没有锁的期限的变异（T9.3）在期限上失败而不是一直等（`t9d`）。
- 复制的辅助（清扫 52）：`aWorkItem` 改用 `newWorkItemIn`（`t11d`，`newWorkItemIn` 因此从 Task 12 提到 Task 11），`workItemRows` 收进按项目、是否只数未删除的一个（`t11d`、`t13b`：规模的测量原来另写了一个数未删除的行的）。`archtest` 的说明重新折行（`t10i`）。
- 变异本身：三个变异原来编译不过，是坏的变异而不是发现（T1.4、T3.9、T4.6），改写成编译得过的同一个缺陷；Task 11 的树上第一次跑"（Task 11 起）"的四个变异时构建失败（那时它的 `aWorkItem` 已用 `newWorkItemIn`，而 `newWorkItemIn` 还在 Task 12），`t11d` 之后重跑，构建失败不算发现（清扫 56）；T9.3 原来让测试一直等，`t9d` 之后在期限上失败；契约的变异（T10.13、T10.14）原来只改 `api/modules`，而测试读 `api/dist` 的打包结果，改为两处一起改。

**清扫之后的加强**（变异按层跑完之后，只在单元一层或存储一层被发现、而组合一层或故事本可以展示的，改测试或补上更高一层的检查，不改产品代码；`layercounts.py` 列出它们）：

- I2（Task 14）原来分不出三个变异（T3.6、T3.8、T3.10）：工作项只有一个负责人、一个标签，去掉的就是全部；去掉它们的人又是写那些行的人。改为指派 bob 和 ann、标签 Bug 和 Docs，由 bob 去掉 ann 写的行，ann 和 Docs 留下（`t14b`；创建的动态按 `id` 的次序，`t14c`）。再加四步和一个旁观的工作项：ann 再指派 bob、再加 Bug 和 Login（两个按 `id` 的反序给出：T1.5）；bob 只去掉 Bug、ann 只去掉 bob（各是一次修改：T1.9、T1.15；先前去掉的行保留那次的戳：T3.5、T3.7）；两个 422（值的问题：T1.2、T1.3；项目的问题）；与它共有 bob 和 Bug 的 Fix the logout 始终如创建时（T3.1、T3.4、T3.11）。标题随之（`strengthen.sh` 把原型的故事放进 Task 14、15 的快照）。
- 忙的项目的测试（Task 10）原来只看排满的拒绝和等锁的期限：改为排队一个、`max_wait` 500 毫秒，两个创建一个立即被拒、一个到 `max_wait` 被拒（T8.2、T8.4、T8.6、T10.6），位置被占时 olga、walt 的修改立即由预检答 404、403（T10.1：预检拒绝的不占闸门）（`t10j`；预检之后改为 `max_wait` 1 秒、锁的期限 3 秒，见下）。它本来就发现、却没有记进变异的层的 T9.3（`TxManager` 开始事务时不带它的选项：alice 的修改等不到锁的期限）补上这一层。
- 请求在闸门上到期（Task 10）原来只在单元一层（`writeGate` 的表）：组合出的 app 里，写按先来先得进闸门，排在前面的写通常比它先开始、期限先到，等待者的期限要在闸门上先到，得让它进闸门之前用掉的时间比前面的写多。加 `TestAWriteWhoseRequestEndsWhileItWaitsAtTheGateIsServerBusy`（2.12）：bob 的修改在预检里等别的事务持的 `issue_labels`，1.5 秒之后 alice 的创建占了位置、等 Web 的行，bob 的修改在闸门上等到他的请求的期限，503（`t10k`；Task 10 的修复轮改为 2 秒之后、她的锁的期限 2 秒：她最早仍在 bob 的修改发出之后 4 秒离开闸门，她的等锁比她自己的请求的期限早约 1 秒到期，2.12）。T10.5（请求到期答 500）、T8.7（等待者不看 `ctx`）在它上失败；新加的变异 T10.16（写用自己的 `context` 进闸门：brief 的"忽略 ctx"）只在它上失败。
- 交错（Task 11）加 `issue_interleaving_change_test.go`：同一个工作项的两个修改（1u：T5.3 时刻在锁之前、T5.20 重读在锁之前，后者是新加的变异），等锁期间工作项被删除、被移到别的项目（T5.16、T5.17；P1b 没有这两个操作，由持项目行的事务在数据库里做）；`strengthen.sh`、`strengthen2.sh` 把它放进 Task 11–15 的快照。设计 3.5 的交错表没有这两条，第 3 节第 16 条记下。
- 加强之后，有新的一层的变异在那一层的 Task 的树和最终的树上重跑（`passD.sh`、`passE.sh`），Task 10 的全部变异在 Task 10 的树上重跑（它的组合测试改了），每个有 I2 一层的变异在 Task 14 的树上重跑。

**预检之后的修改**（控制者在执行之前的预检：`$M4TMP/p1b-preflight/preflight.md` 的发现 M1–M4、L1–L6、c1，裁定在 `$M4TMP/p1b-preflight-rulings.md`。代码照预检的加固原样写进（它的 `harden.py`，`p1btools/harden_pf.py`：H1–H4、FA、FB、CANCELLED、STAMPS、G1000），文字的修改在 `p1btools/edits/a-*.json`（`mk_amend.py`）；每处改在拥有那个文件的 Task 的快照上，从那个 Task 起每一份快照都改（`amend-snaps.sh`），`mkblocks.py` 重新生成全部块；Task 6、9、10、15 的快照上 `make gen` 前后 43 个生成物逐字节相同（`gencheck.sh`），plan 钉住的摘要不变）：

- 离开闸门在事务结束之后（M1，Task 10）：`fakeTx.WithinTx` 在事务里设 `callLog.inTx`、`defer` 复位，`fakeGate` 把事务还没有结束时的离开记作 `Leave <项目> in tx`；离开放进事务的函数里（提交、回滚之前就离开）的变异 T10.17 在单元一层被发现，按性质只在单元一层（A.3）。
- 读不进闸门（M2，Task 10、12）：`newGet` 用记下进出的闸门，`TestGetIssueReadsTheWorkItem` 的调用记录里没有进出；交错 20 也读热的工作项。`getIssue` 进闸门的变异 T10.18 在单元一层和（Task 12 起）组合一层被发现（热的工作项的读答 503）。
- 计时（M3，裁定 13）：交错 20 的持有到至少一秒、并且 134 个 503 都答了（3 秒为限），这个测试的 `max_wait` 一分钟，等着的 66 个都答 200；链在每串都有一次修改得到回答之后才测工作区的写（5 秒为限）；忙的项目的测试 `max_wait` 1 秒、锁的期限 3 秒，窗口以它们为界；请求在闸门上到期的测试里 bob 的回答在 3 到 4 秒之间（第 3 节第 13 条，2.12、2.14）。由这四个测试发现的变异都重跑，仍在同一个测试被发现。
- 连接池的总代价（M4）：README、总体设计 4.2、M3 交接的"别的项目的读写不排在它后面"按同时热的项目数改写（Task 15）；本 spec 第 5、7 节和设计 11.11、13.2 写明同时热的项目达到池的大小除以 K（默认 5 个）时池仍会被等锁的写占满（预检在它的树上量的，`pf_hot_projects_test.go`，只作证据、不进仓库：五个项目同时热时第六个项目的读从 1.0 毫秒变为 3.99 秒）；第 5 节 P9 一行加上再量这种情形的交接。
- 负载测试（L1，Task 8）每个 goroutine 1000 次：不带 `-race` 也走到放弃时被交到位置的一支，T8.5 在 `make test` 也被发现。
- 数据（L5、L6）：修改的单元测试加已取消组的状态 Dropped（移进去不完成，从完成组移进去不再完成；变异 T5.21，数据的规则，A.3）；盖戳的测试由 alice 把 walt 加进 Web、bob 指派 walt（原来 bob 指派自己，他的关注本来就在），修改者因修改而关注的变异 T5.22 在（Task 7 起）组合一层被发现。完成组之间的移动写新的时刻（预检的 PF.C1）只有单元一层分得出：数据的规则，按 plan 的规则不算缺口（A.6 的完成时间一行）。
- 文字（L2、L3、L4、c1，裁定 10 的 P3 一句）：Task 12 的 Expected 写 2–49 毫秒；`IssueUpdater` 的说明照代码（嵌入 `IssueReader`，2.7）；`issue/module.go` 的包说明写到取、改工作项（Task 6 多一处）；`updateIssue` 的描述把"at one moment."写在一行（Task 6 写整行，Task 10 的锚点是整行；折叠之后的字符串不变，生成物不变）；第 5 节 P3 一行加上进第二个闸门时只持着前一个闸门的位置。
- 变异：新加 4 个（T5.21、T5.22、T10.17、T10.18），T8.5 加不带 `-race` 的一层；它们和每个以改过的测试为一层的变异（共 27 个）在改过的快照上按它们的层重跑（`passAM.sh`：各自的 Task 的快照、"Task n 起"的、最终的树），98 层全部发现，没有构建失败和崩溃，结果在新的文件里（A.6）。

### A.3 只在单元、存储一层被发现的变异

**只在单元一层被发现的（按性质，35 个：原型的 21 个和执行中加的 14 个）**：

- 端口的失败被吞掉（清扫 2）3 个：T5.13（工作项的写失败之后照样往下走）、T5.14（读成员的角色失败之后照样往下走）、T5.23（锁下重读工作项的失败答成 404，执行中加的）。同 P1a：真实的存储在这些步骤不失败；而一条失败的语句让 Postgres 中止事务，之后的语句和提交都失败，吞掉它的用例也写不进任何东西（T5.14 把没读到的角色当作没有角色，答 422 而不是 500，同样什么都没写；T5.23 把失败的重读答成 404 `issue.not_found` 而不是 500，在写之前就停下）。存储方法自己的失败由清扫 19 在存储一层核对。
- 次序的约定（清扫 9）1 个：T5.11（先写标签的行、再写负责人的）。工作项一侧五张表的每个写者都先持项目 N，同一个项目的行在它之下没有第二个写者，表的次序等不起来；全局顺序是给以后的写（P9 的任务）的约定，由单元的调用次序钉住（同 P1a 的 T3.2、T8.12）。
- 关注（3.12）2 个：T5.10（修改去掉的负责人也加关注）、T5.26（只给新指派的第一个账户加关注，执行中加的）。经接口到得了的状态里，去掉的负责人在被指派时已经关注（创建给创建者和负责人、修改给新的负责人加关注），P1b 没有取消关注（M7），关注的部分唯一键让再插的一行什么都不做（`ON CONFLICT DO NOTHING`，P1a 的 `TestInsertSubscribersSubscribesEachAccountOnce`）：组合出的 app 里等价，只有假实现记下了这次调用。T5.26 是数据的规则（每个新负责人都关注）：组合一层和 I2 没有一次加两个负责人的修改（整分支评审：它在每个 `bootstrap` 测试和 I2 上都通过），单元的表一次加两个。
- 判定的动作 1 个：T4.7（`getIssue` 判定 `issue.create` 而不是 `issue.read`，执行中加的）。P1b 里两条规则的角色是同一个集合（项目的管理员、成员，3.4），组合出的 app 里两个判定的回答处处相同；P2 加上访客规则之后才分得出（第 5 节 P2 一行）。单元测试由假的判定记下所问的动作。
- 完成时间 1 个：T5.21（移进已取消组的状态当作完成，预检 L5）。状态的组决定完成时间是数据的规则，不是安全或加锁的性质：`domain.CompletedAt` 由它的单元测试钉住，修改的单元测试的表有已取消组的两行；组合的测试和 I2 没有移进已取消组的一步（I1 在已取消的状态里创建，走的是创建）。完成组之间的移动写新的时刻（预检的 PF.C1）同样只有单元一层分得出（A.6 的完成时间一行）。
- 契约和文案 3 个。T6.2（`IssueUpdate` 收下后面的 Phase 的字段）：体的形状由生成的 `bodyshape` 在用例之前检查，HTTP 的测试驱动的就是生成的路由和真实的处理函数，这条路上没有替身（同 P1a 的 T9.11）。T4.4（页面没有 `issue.not_found` 的文案）：`error-messages.test.ts` 读的是真实的文案表，显示它的工作项的页面在 P14（同 P1a 的 T11.17）；中文的一份缺这个键（T4.5）由 `check:sync` 发现。T5.28（不是项目的标签的问题落在它在加的标签里的下标，执行中加的）：问题的位置（`label_ids[i]`）是契约，单元的表有一个不是第一个的被拒绝的标签；组合一层（7u）和 I2 的被拒绝的标签都是请求里的第一个，两种下标相同。
- `platform/gate` 的内部 8 个：T8.1（交给最后来的）、T8.3（离开两次让出两次）、T8.5（放弃时正好被交到位置的留着它）、T8.8（`ctx` 已结束的照样拿到空位）、T8.9（没人持有、没人等的键不被忘掉），执行中加的 T8.12（放弃时正好被交到位置的把位置空出来、不交给下一个）、T8.13（放弃的等待者由队尾的顶替）、T8.14（排满在拿空位之前就拒绝：`MaxQueue` 0 时谁也进不了）。闸门的测试驱动的就是真实的组件（真实的 goroutine、`context` 和计时，负载测试带和不带 `-race`：T8.5 在两者都被发现，预检 L1），没有替身；组合出的 app 只加它的接线和换码，由两个忙的项目的测试在组合一层核对（T10.9–T10.12），排满、等到 `max_wait`、放弃的离开队列、请求到期时放弃也在那里（T8.2、T8.4、T8.6、T8.7）。原来的五个是等待者之间的次序、交接与放弃的竞争和内存：组合一层要让几个 HTTP 请求按指定的次序排进队列、在交接的那一刻放弃，造不确定；T8.8 要请求在预检之后、进闸门之前结束，它随后的事务在结束的 `context` 上开不了，同样什么都不写。T8.12、T8.13 同样是交接时的放弃和等待者之间的次序；T8.14 要 `max_queue` 0（配置允许的下限），组合的测试的 `max_queue` 是 1 和 64。
- 配置 12 个：T9.5–T9.10（`per_project` 可以是 0、`max_wait` 加锁的期限可以等于请求的期限、`issue` 不校验、`max_queue` 可以是负数、随附的配置让一个项目进三个写、日志写错键）和执行中加的 T9.11–T9.15、T9.17（拒绝两个下限 `max_queue` 0、`per_project` 1；日志的三个值取自别的键；和的规则丢掉 `max_wait`）。`platform/config` 的测试驱动的是真实的加载、校验和随附的 `configs/config.yaml`（`configs` 的测试读嵌入的那一份），没有替身；组合的测试各自给出配置（`testIssueConfig`），不读随附的文件，组合出的 app 拿到的是已经通过校验的配置。
- 写闸门接入 4 个。T10.2（不用 `defer` 离开：panic 留着位置）、T10.20（`Locks.run` 把写的 panic 换成错误，执行中加的）：组合出的 app 里没有输入能让写在事务里 panic，单元测试由写函数自己 panic（`TestAWriteLeavesTheGateOnEveryPath` 核对 panic 照样传给调用者）。T10.4（等闸门时被取消的请求答 `server_busy`）：被取消的请求的回答没有人收到（客户端已经走了），答 503 还是 499（`httpserver` 对 `context.Canceled` 的回答），只在服务端的日志里不同；取消与到期走的是同一条路（`Enter` 收到请求的 `context`），到期的一半在组合一层（T10.5、T10.16）。T10.17（离开闸门放进事务的函数里：提交、回滚之前就离开，预检 M1）：位置在提交之前交出，下一个写在前一个提交的同时取连接、开始事务，一个热的项目在每次交接时至多占 2K 个连接（提交慢时更久）；两个写的回答和写下的行都一样，提交的时刻与交接的先后经 HTTP 不靠计时看不到。单元的 `fakeTx` 记下事务还没有结束时的离开（`in tx`）。

**只在存储一层被发现的（按性质，6 个：原型的 4 个和执行中加的 2 个；都在真实的数据库上）**：

- T3.9（`RemoveAssignees` 把语句的失败答成没事）：存储方法自己的失败（清扫 19），取消的 `context`；组合出的 app 里真实的存储不失败。
- T3.12（`RemoveLabels` 只去掉给出的第一个标签，执行中加的）：存储方法交给语句的集合（`= ANY`），存储测试一次去掉两个标签，在真实的数据库上；组合一层和 I2 的修改每次至多去掉一个标签（矩阵的"每个字段都给"一行去掉 Bug，I2 每次去掉一个），同一种缩小的 `RemoveAssignees`（T3.13）在矩阵被发现（那一行一次去掉两个负责人）。
- 锁的期限 4 个（`platform/postgres` 的 `TestLockTimeoutBoundsEachWaitOfTheTransaction`、`TestLockNotAvailable`，真实的连接和锁）。T9.1（`SET` 在会话上，不是 `SET LOCAL`）：存储的测试在同一个连接上、事务提交之后直接核对期限回到默认值（A.2 的 `t9c`）；组合出的 app 里要看到它，得让工作项一侧的写之后、同一个连接上的别的模块的写等锁超过期限（一个连接的池才定得下是同一个连接），看到的仍是这一条语句的性质。T9.2（向下取整到毫秒）、T9.16（取最近的毫秒，执行中加的）：组合的测试和随附的配置都用整秒、整毫秒的期限；存储测试的 1.2 毫秒分得出（向上取整是 2 毫秒）。T9.4（把死锁 `40P01` 当作等锁到期）：组合出的 app 里没有死锁（交错和竞争的 `-race` 跑核对没有 `40P01`），`bootstrap` 的 `TestABusyTransactionIsServerBusy` 另核对 `40P01` 不换成 503。T9.3（`TxManager` 开始事务时不带它的选项）在组合一层的忙的项目的测试失败（alice 的修改等不到锁的期限，A.2）。

**等价的 1 个**：T8.11（有人等时新来的拿走空位）。`release` 把位置直接交给队首、不腾出空位，调用者只在位置全被持有时排队，所以有空位时一定没有人在等，去掉的条件不改变行为（第 3 节第 11 条）；它在闸门的单元测试和 `-race` 的负载测试上都通过。

**组合一层没有、只有端到端发现的**：没有。I2 发现的 40 个变异每个另在单元、存储或组合一层被发现（原型的 35 个由 `e2ealso.py` 核对；执行中加的 5 个：T5.25、T5.27、T5.31、T5.32 在单元一层，T5.30 在组合一层）。

### A.4 最终的原型、矩阵、交错、连接池和规模

**最终的原型**（`gates.sh`，2026-10-18：七个门禁在最终的树上，`logs/final4-*.log`、`final4.summary`；此后原型的代码只被变异改动（每个在自己的副本里），文档换成复制进来的 spec、plan 和设计；逐 Task 复现的终态与它逐个文件相同，A.5；预检之后的修改（A.2）只改测试和文档，改过的原型由新的逐 Task 复现在每个 Task 之后跑过它的门禁，A.5）：`make gen` 之后生成物没有差异；`make lint-go` 两段 `0 issues.`；`make test` 47 个 `ok`；`make lint-web`（关键词守卫 72 条规则、23 个例外、没有命中，54 个任务）；`make knip`；`make test-web`（17 个任务）；`make e2e` 122 个中 119 个通过，I1、I2 在其中；失败的三个与 P1b 无关：S3 因原型不是仓库的检出、构建没有提交号而失败（M3/P4b 的 F4），A4 的两个 firefox 测试因为机器上没有 Firefox。

**矩阵**：`TestPermissionMatrix` 1,055 格全部通过（`getIssue` 15 格、`createIssue` 75 格、`updateIssue` 九行 99 格，其余 866 格是 M3 的），顶层 1.75 秒（`matrixcount.sh`，`logs/matrix.log`）。

**交错和竞争**（`raceall.sh`，`logs/race-final.log`；预检之后在改过的原型上再跑一次，`raceall_amend.sh`，`logs/race-amend.log`）：`bootstrap` 的 45 个交错、竞争、取锁形状和忙的项目的测试各 `-race -count=5`，225 次顶层、930 个子测试全部通过，257 秒（改过之后 265 秒），两次都没有 40P01、没有死锁、没有数据竞争。其中 M3 的 30 个；P1a 的 9 个（4 个加了修改的一支，换成两支共用的名字；取锁形状的测试多了 `updateIssue` 的一行）；P1b 的 6 个：1u、等锁期间工作项不在了、两个忙的项目的测试、交错 20 的两个。

**交错 20 的上限**（加固之后的测试（A.2）在改过的原型上，`poolmeasure.sh` 带和不带 `-race` 各 5 次，`logs/hardpool-race.log`、`hardpool.log`，`poolamend.py`；池 10 个连接，闸门是默认值，这个测试的 `max_wait` 一分钟）：别的事务持 Web 的行至少一秒、200 个修改打到 Web 时，每次恰好 134 个 503、行放开之后 66 个 200；Ops 和 Web 的工作项的读每次 74–78 次（`-race`）、86–88 次（不带 `-race`），最慢的一次 11.4–41.4 毫秒（`-race`）、3.0–11.7 毫秒（不带），上限 500 毫秒（第 3 节第 13 条）。16 串修改之中，改 acme 的名称 2.5–4.9 毫秒、把 walt 改为管理员 4.0–6.7 毫秒、删除 acme 17.7–74.4 毫秒，上限 1 秒。预检在它的加固的树上（`$M4TMP/p1b-preflight/preflight.md` 的计时一节）：`-race` 20 次，每次读 74 次、最慢 39–44 毫秒；不带 `-race` 40 次，每次读 84–90 次、最慢 1.2–46 毫秒；没有闸门的变异读最慢 3.42 秒（加固之前的测试上 1.25 秒）。逐 Task 复现的 Task 12 的一次（A.5，`replay-logs/T12-1.log`，不带 `-race`）：读 86 次、最慢 8.7 毫秒，改名称 4.5 毫秒、改为管理员 4.1 毫秒、删除 26.5 毫秒。加固之前的测试（只读 Ops，`logs/pool-final-race.log`、`pool-final.log`，各 5 次）：读每次 41–45 次，最慢 11.7–38.8 毫秒（`-race`）、10.5–28.0 毫秒（不带），改名称 2.7–4.9 毫秒、改为管理员 3.6–6.6 毫秒、删除 30.7–45.7 毫秒。原型上全部有闸门的运行（`logs/pool-*.log`，连同先前几版测试共 45 次，带和不带 `-race`；`poolranges.py`）：读最慢 2–49 毫秒，改名称 2.7–7.7 毫秒、改为管理员 3.6–8.9 毫秒、删除 17.2–50.2 毫秒；先前的逐 Task 复现的 Task 12 的一次（`replay-logs-a04d479d/T12-1.log`，不带 `-race`）：读 46 次、最慢 5.3 毫秒，改名称 3.1 毫秒、改为管理员 4.9 毫秒、删除 54.0 毫秒。把闸门去掉（每个写不进闸门，`pool-nogate2.log`、`pool-nogate3.log`）：一版测试的三次，读最慢 1.12–1.26 秒；另一版的三次，都是约 4.97 秒（等到请求的期限）。

**规模**（`TestTheDeletionsAtTheScaleTarget`，`NERVE_MEASURE_SCALE=1`，四次：`logs/scale-1.log` 一次、`scale-2.log` 两次、`scale-final.log` 一次，最后一次在最终的原型上；本机 macOS、Docker Desktop、PostgreSQL 18.6 的 testcontainer，按 ±2–3 倍读）：六个项目各 2 万个工作项、每个项目 50 万行，共 300 万行，种下 38.6–44.3 秒。

| | 工作项一侧的软删除（回滚的事务） | 经接口的请求 |
|---|---|---|
| 删除 Web（一个项目，50 万行） | 4.83–6.32 秒 | 4.40–4.71 秒 |
| 删除 beta（五个项目的工作区，250 万行） | 25.1–29.7 秒 | 23.0–24.6 秒 |

逐 Task 复现的 Task 13 在复现的副本上又量了两次（A.5；`replay-logs-a04d479d/T13-2.log`：种下 45.3 秒，删除 Web 一侧 5.25 秒、请求 4.79 秒，删除 beta 一侧 28.5 秒、请求 26.7 秒；`replay-logs/T13-2.log`：种下 40.8 秒，删除 Web 一侧 4.95 秒、请求 4.62 秒，删除 beta 一侧 25.5 秒、请求 23.4 秒）。请求在一侧的测量之后量（同样的页已经读过一遍），比一侧的单独测量略快。生产的请求期限是 15 秒：删除 beta 的请求在那里到期、回滚，什么都没删（测试把期限放长到一小时才量得到它）；删除一个这样的项目在期限之内。决策点 7（负责人裁定 A）把工作项一侧交给 P9 的任务，并要 P9 在同样的数据上再量（第 3 节第 14 条、第 7 节）。

**执行中的运行**（各 Task 的实现者和整分支评审在执行的树上；`progress.md`、`$M4TMP/p1bfinal/review.md`）：矩阵在执行完的分支上同样：`-v` 的 1,056 个 PASS 是顶层一个、`prepare` 一个和格子 1,054 个（`getIssue` 15、`createIssue` 75、`updateIssue` 99；上面的 1,055 格把 `prepare` 算在里面）。交错 20：Task 12 的实现者带 `-race` 5 次、不带 10 次、`GOMAXPROCS=2 -race` 3 次都通过，读每次 74–88 次、最慢 2.1–45 毫秒，链上工作区的写 2.5–44.9 毫秒，没有闸门的变异读最慢 3.19 秒；整分支评审在 `3693abc7` 的副本上带 `-race` 5 次：读每次 76–80 次、最慢 29.8–48.0 毫秒，改名称 3.9–5.7 毫秒、改为管理员 5.7–8.6 毫秒、删除 34.0–53.7 毫秒；`GOMAXPROCS=2 -race` 3 次：读每次 78–80 次、最慢 10.8–27.2 毫秒，改名称 3.1–5.8 毫秒、改为管理员 5.5–12.8 毫秒、删除 31.3–43.8 毫秒。交错和竞争：整分支评审的 `bootstrap` 48 个测试（`raceall.sh` 的 45 个，加事务的连接和两个换码的测试）各 `-race -count=5`，290 秒，240 次顶层、930 个子测试全部通过，没有 40P01、死锁、数据竞争。规模：Task 13 的实现者量了三次，第一次种下 300 万行 42.2 秒，删除 Web 的请求 4.95 秒、删除 beta 的请求 24.99 秒，另两次在别的会话的 Docker 负载下（Web 5.04、6.19 秒，beta 36.74、29.42 秒）；修复轮（种子的核对）之后 Docker 空闲时一次：种下 43.0 秒，Web 5.08 秒，beta 27.45 秒。都在 ±2–3 倍之内，结论不变：一个这样的项目在 15 秒之内，五个的工作区不在。门禁：整分支评审在副本上 `make gen`（代替 `make gen-check`：前后 43 个生成物逐字节相同）、`make lint-go`、`make test`、`make lint-web`、`make knip`、`make test-web` 通过，`make e2e` 122 个中 119 个（失败的同上：S3、A4 的两个 firefox 测试）；worktree 里 Task 14 的 `make e2e` 122 个中 120 个（只有 A4 的两个 firefox 测试失败）。

### A.5 逐 Task 复现

**逐 Task 复现**（预检之后的 plan；`replay-all-amend.sh`、`replay.py`，日志在 `p1btools/replay-logs`、`logs/replay-all-amend.log`，`replaysum_amend.py`）：`5ce30a1a` 的一份新的副本（`base-5ce30a1a.tar`），本提交的 spec、plan 和设计照它们在 worktree 里的样子放进去，`pnpm install --frozen-lockfile`，`bin/golangci-lint` 取原型的（2.13.2；`make tools` 会下载它）。先在基点的另一份新的副本上 `planapply.mjs check`：273 个块按顺序都对。按 plan 的顺序应用 15 个 Task 的 273 个块，照原样运行各 Task 的 84 个 `Run:` 命令（其中 5 个是提交之后的 `make gen-check`），一共 1,595 秒，没有重跑。Task 1–13 之后 `make lint-go` 两段 `0 issues.`；除 Task 8 之外 `make test` 全部 `ok`（Task 1–7 是 46 个，Task 8 加了 `platform/gate` 之后 47 个），Task 8 跑 `platform/gate` 的测试（含 `-race -count=5`）和 `archtest`；`make gen-go`（Task 2、3）和 `make gen`（Task 4、6、10）之后，生成物的每个 SHA-256 和行数与 plan 的表相同，提交之后的 gen-same 没有差异；`make lint-web`、`make knip`、`make test-web` 在 Task 4、6、10、14 通过；Task 8、11、12 的 `-race -count=5` 没有 40P01、死锁和数据竞争；Task 13 的规模测量不设 `NERVE_MEASURE_SCALE` 时跳过，设了时 105 秒（数字在 A.4）；`make e2e` 在 Task 5、10 是 121 个中 118 个通过、在 Task 14 是 122 个中 119 个通过（I1、I2 在其中），失败的只有 S3（F4）和 A4 的两个 firefox 测试（机器上没有 Firefox）；Task 15 的 `make lint-web` 通过，`doclinks.py` 没有断的链接，`make test` 47 个 `ok`。每个 Task 之后的树存成快照，换下 `p1bsnap`（A.1）。最后 `treediff` 比较复现的树与改过的原型的 3,372 个文件，没有差异；与预检的加固的树（`$M4TMP/p1b-preflight/hard`）比较，差的只有：`issue/module.go` 的包说明（L4）、`api/modules/issue.yaml` 的折行（c1）、README、总体设计和 M3 交接的连接池的总代价（M4）这五个文件的那几行（`harddiff.py`），本提交的 spec、plan 和设计，和只在加固的树里的预检的证据测试 `bootstrap/pf_hot_projects_test.go`。复现的副本不是 git 检出：`make gen-check` 由 gen-same 代替（`make gen` 前后生成物逐字节相同），`make lint-web` 的关键词守卫经 `kwreal.py` 列出文件。

**复现之后**：plan 和设计与复现读的逐字节相同。spec 的正文在复现读过之后改了三行（`specbody.py`）：2.1 的文档数写上设计 11.11、13.2，第 3 节第 13 条加上加固之后的运行，第 6 节 T8.5 一行写明负载测试带和不带 `-race`；附录 A 里预检之后的部分（A.1 的快照和大小、A.2、A.3、A.4 的运行、本节、A.6）也在复现之后写入。

**先前的复现**（`a04d479d` 的 plan；`replay-logs-a04d479d`、`p1breplay-a04d479d`）：267 个块、84 个 `Run:` 命令，1,823 秒，没有重跑，门禁的结果同上，`treediff` 与当时的原型的 3,372 个文件没有差异。那次之后改过三份文档的几行文字，块、`Run:` 和生成物的摘要都没有变：Global Constraints 的门禁一句（Task 8 不跑 `make test`）、`doclinks.py` 不把代码当作链接、Task 10 的 alice 的 503 的时刻；spec 的第 1 节和第 4 节第 3 条、2.12、第 3 节第 13 条和第 7 节的数字；设计第 15 节 P1b 一行的"（15 个 Task）"。

**执行中的复现**（`progress.md`）：每个改了 plan 的块的修复轮之后，实现者在基点的一份新副本上照 plan 应用到那个 Task（`planapply.mjs`），`treediff` 与 worktree 比较；块数随修复轮从 273 到 308。整分支评审在 `3693abc7` 上：308 个块（新文件 24、替换 282、整个文件 2）从基点起按顺序都对，15 个 Task 应用之后与 `3693abc7` 比较（3,369 个文件对 3,372 个），差的只有 6 个生成物（plan 钉住它们，由 `Run:` 的步骤生成）、plan、spec 和设计（`$M4TMP/p1bfinal/logs/replay.log`）。之后的修复轮（`203f1f94` 和 spec、plan 的这次修改）每次之后同样：308 个块都对，差的只有生成物、plan、spec 和设计，第二次还有这次修改的 P1a 的 spec（`$M4TMP/p1bsdd/ffw/replay-c1.log`、`replay-c2.log`）。

### A.6 清扫和变异表

**五十六类清扫**（P1a 的五十类（P1a spec 附录 A.6）和 brief 加的六类（51–56）；每个变异在一份新的副本里（`p1bmut/<id>`），Go 以外的文件（契约、故事、文案）也在副本里改，`mut.py` 在变异写的每一层各跑一次：单元（各模块的 `app`、`domain`、`adapter/http`，`access/domain`，`platform/gate`、`platform/config`、`configs`，`bootstrap` 的转换和换码的测试，Web 的 `error-messages.test.ts`）、存储（各模块的 `adapter/postgres`，`platform/postgres` 的锁的期限）、组合（`bootstrap`、`archtest`）、端到端（单独运行的故事 I2）、守卫（`check:sync`）；每个变异在它的 Task 的快照上跑，后面的 Task 才有的检查在那个 Task 的快照上跑（"（Task n 起）"），全部的 Go 一层在最终的树上再跑一次；"层"是它在最终的树上被发现的每一层（端到端在 Task 14 的树上：Task 15 只改文档）；端到端的"发现"由 `e2ekills.py` 核对是断言失败，不是崩溃或超时；预检之后的修改（A.2）之后，受影响的变异在改过的快照上照同样的做法重跑，`passAM.sh`）：

| 清扫 | 大小 | 结果 | 层 |
|---|---|---|---|
| 1 每个 SQL 谓词 | `update.sql` 的三条语句：`UpdateIssue` 的 `id`（T3.1）、写者和完成时间（T3.2、T3.3）；`RemoveAssignees`、`RemoveLabels` 的工作项、`= ANY`、已删除（T3.4–T3.7、T3.10、T3.11），`= ANY` 只收到给出的第一个（T3.12、T3.13，执行中加的：Task 3 的修复轮让存储测试一次去掉两个）；`issue_ports.sql` 的状态、标签的名称（T2.1、T2.2） | 全部被发现；T3.1、T3.4–T3.7、T3.10、T3.11 原来只在存储一层，I2 的加强（A.2）之后也在端到端；T3.12 只在存储一层（A.3），T3.13 也在矩阵 | 存储；组合；端到端 |
| 2 端口调用的错误 | 用例吞掉端口的失败：工作项的写、读角色（T5.13、T5.14）；存储的失败（T3.9） | T5.13、T5.14 只在单元一层（按性质，A.3）；T3.9 在存储一层 | 单元；存储 |
| 3 写不动的行 | 修改的存储测试比较被写的工作项之外的每一行每一列（`TestUpdatingAWorkItemWritesItsRows`）；I2 的另一个工作项（与它共有 bob 和 Bug）从头到尾如创建时 | T3.1、T3.4、T3.11 在存储一层和 I2 | 存储；端到端 |
| 4 组合根的接线 | `bootstrap` 不给 `issue` 闸门、带锁的期限的事务（T10.9–T10.12），命令行不建闸门（T10.15）；状态、标签名称的转换（T2.3）；HTTP 的映射（T4.6、T6.1、T7.2、T7.3）；读不进写闸门（T10.18：`newGet` 的闸门记下进出，交错 20 读热的工作项，预检 M2） | 都在组合一层或 I2 | 单元；组合；端到端 |
| 5 安全性质在真实环境上 | `issue.read`、`issue.update` 的规则（T4.2）；预检、锁下的判定（T4.1、T5.1、T5.15）；预检拒绝的不进闸门（T10.1）；闸门在事务之外（T10.3）、用请求的 `context`（T10.16）、离开在事务结束之后（T10.17，预检 M1） | 都在组合一层（矩阵、交错 4u、5u、两个忙的项目的测试、交错 20）；T10.17 只在单元一层（按性质，A.3） | 单元；组合 |
| 6 每句文档 | 每句说明、契约描述、故事标题对照代码；`prophecy.py` 列出每个 Task 加的往后看、计数的注释 | 改了五处（A.2：`patch.go`、规则、HTTP 测试、盖戳测试、契约的完成时间一句）和规模测试的一句；预检之后又改了 `issue/module.go` 的包说明（L4）、`IssueUpdater` 的说明（L3）、连接池的总代价（M4：README、总体设计 4.2、M3 交接）、交错 20 的 Expected（L2） | 审阅 |
| 7 反例里没有随机 | P1b 的测试 | 表都是切片；I2 和矩阵的"没有的状态、标签"用随机的 id，是不存在的 id，不是取样；交错 20 的突发是 200 个固定的请求 | 审阅 |
| 8 决定所依赖的读在调用者的事务里 | 修改经池读回答（T7.1）；锁下的重读、判定、目录、角色、显示名都在事务的连接上 | T7.1 在组合一层（一个连接的池：`TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`） | 组合 |
| 9 锁顺序在真实环境上 | 修改最先锁项目、工作区上没有锁（T6.3、T12.1）；写表的次序（T5.11）；重读在锁之后（T5.20）；闸门的位置持到提交、回滚之后（T10.17） | T6.3、T12.1、T5.20 在组合一层；T5.11、T10.17 只在单元一层（按性质，A.3） | 单元；组合 |
| 10 承重的种子行有前提 | 矩阵的 `issueSeed`（每个项目一个工作项，由工作区的管理员写，指派 PM 和一个已离开工作区和它的公开项目的账户，标签 Bug）；`aWorkItem` 的三行；I2 的两个工作项；规模的测量的种子（执行中加的核对） | 矩阵的种子由 `seededIssues` 读回核对；`wroteNothing` 先核对准备的行数；I2 每一步之后 `expectIssueUpdated`；规模的测量种下之后核对未删除的行数是 6 × 50 万（T13.1） | 审阅；组合；端到端 |
| 11 每条拒绝路径 | 预检拒绝的（404、403）、锁下的（404、403、409、422）、规则的 422 | 单元的两个拒绝表；矩阵的拒绝五行（已归档（什么都不改也 409）、另一个项目的状态、另一个项目的标签、分诊状态、指派访客）；交错 3u–7u 和"工作项不在了"；I2 的两个 422 | 审阅 |
| 12 与结束、删除的竞争（组合） | 修改与删除项目、工作区（3u），与移出、降级（4u、5u），与删除状态、标签（6u、7u），与另一个修改（1u），与工作项的删除、移走（P1b 没有这两个操作，由事务在数据库里做） | 3u、6u、7u 两种顺序都答对；4u、5u、1u 和工作项不在了只跑一种：写先过了预检、在锁上等，另一个先提交；等待由探测确认；T5.3、T5.6、T5.9、T5.15–T5.18、T5.20 在这些交错里被发现 | 组合 |
| 13 锁强度在组合一层看得到 | 修改停在它锁下最先读、预检没有读的表（`states`）上，`lockOn` 读出项目的 N、工作区上没有锁 | T6.3（预检以 `FOR SHARE` 读工作项）、T12.1 在组合一层 | 组合 |
| 14 每个"由谁"可以失败 | 盖戳的测试：alice 建、bob 改；I2：bob 去掉 ann 写的行（换掉负责人和标签的一步、只去掉 Bug 的一步），去掉的行的戳是他的；修改的写者（T3.2）和去掉的行的写者（T3.8）；3u：bob 写的行由删除记在 alice 名下（T11.1，执行中加的：Task 11 的修复轮改由 bob 写） | T3.2、T3.8 在存储一层和 I2；T11.1 在存储一层（P1a 的批量删除的测试）和组合一层（3u、P1a 的按目录的删除测试） | 存储；组合；端到端 |
| 15 标题的每个说法都有展示 | I2、交错和组合测试的说明逐句 | I2 的标题随加的步骤改写（A.2），每个说法一步 | 审阅 |
| 16 单元假对象的回答顺序 | 假实现的标签目录按目录自己的次序回答，存储的按 `id` 排序；端口不承诺次序：写按 `id` 找每个标签（Task 2 起），存储测试把回答当作集合比较 | T2.1、T2.2 在存储一层 | 审阅；存储 |
| 17 判定之前的代价有界 | 预检：一次不加锁的读、一次判定；锁下：至多读状态、角色、标签各一次，写每张表各一条语句 | 有界 | 审阅 |
| 18 契约描述只说代码做的 | `updateIssue` 的描述（完成时间只在状态改变时变，A.2）；`createIssue`、`updateIssue` 声明 `server_busy`（T10.13、T10.14）；`IssueUpdate` 不收别的字段（T6.2） | T10.13、T10.14 在单元和组合一层；T6.2 只在单元一层（按性质，A.3） | 单元；组合 |
| 19 每个新的存储方法有失败测试 | `UpdateIssue`、`RemoveAssignees`、`RemoveLabels`（`TestAFailedUpdateStatementIsAnError`，T3.9）；`LockTimeout` 的等锁失败（`TestLockTimeoutBoundsEachWaitOfTheTransaction`） | 都在存储一层 | 存储 |
| 20 相关谓词的集合形式、"别的"算进自己 | `RemoveAssignees`、`RemoveLabels`、目录用 `= ANY`；留下的负责人和标签（ann、Docs）与去掉的在同一个集合里 | T3.6、T3.10 在存储一层和 I2 | 存储；端到端 |
| 21 拒绝和失败钉住第一个 `*shared.Error` | 单元的拒绝表用 `errors.Is` 比较码；HTTP 的拒绝表按码；矩阵按格子的码和问题 | 审阅 | 审阅 |
| 22 端口的回答对得上所问 | 工作项的状态不是项目的、去掉的标签不是项目的、没有显示名的负责人、写之后读不回来都是错误 | `TestUpdateIssueStopsAtAFailingPort` 的四段 | 单元 |
| 23 组合的夹具跨第二个工作区、第二个项目 | 矩阵的各项目；"工作项不在了"移到 Ops；交错 20 的四个项目；规模的测量的 beta | 修改不碰别的项目的行（I2 的另一个工作项、存储测试） | 组合；端到端 |
| 24 每条规则的每一半在组合一层有反例 | 3.7 的规则：值的检查（T1.1–T1.3）、留下的负责人不查（T5.7）、新负责人的角色（T5.8）、别的项目的状态和标签、分诊（T5.5、T5.6、T5.9）；什么都不改（T1.7–T1.9、T1.14、T1.15、T5.2）；完成时间：移进已取消组不完成（T5.21，预检 L5） | 都在矩阵或 I2；T1.2、T1.3、T1.9、T1.15 原来只在单元一层，I2 的 422 和"只去掉一个"两步之后在端到端；T5.21 只在单元一层（数据的规则，A.3） | 单元；组合；端到端 |
| 25 判定和执行在同一把锁下 | 锁之前读时刻（T5.3）；锁之前重读（T5.20）；锁下不判定（T5.15）；闸门的位置持到事务结束（T10.17） | T5.3、T5.20、T5.15 在组合一层（1u、4u、5u）；T10.17 只在单元一层（按性质，A.3） | 单元；组合 |
| 26 "不变"用不会碰巧得出的值 | 存储测试的工作项每一列都不是默认值，去掉的行一小时前由别人写；盖戳的测试由 bob 在 alice 之后改 | 逐列比较 | 存储；组合 |
| 27 故事里的"不动"在动作之前读、之后核对 | I2 的什么都不改和两个 422：之前一步的 `expectIssueUpdated`、之后再核对同一个；另一个工作项在最后核对如创建时 | 都有 | 审阅 |
| 28 只在端到端被发现的性质另有 Go 的测试 | 变异表里只在端到端失败的 | 没有：每个有 I2 一层的变异另在单元、存储或组合一层被发现（执行中加的 T5.30 在组合一层，A.3） | 审阅 |
| 29 每个能等锁的等待都有期限 | 写的等待：闸门上到 `max_wait` 或请求的期限（T8.6、T8.7、T10.5、T10.6、T10.16），事务里的每次等锁到 `issue.write_lock_timeout`（T9.3、T10.9、T10.10）；测试一侧：交错的 context 10 秒、探测 5 秒、锁的期限的测试 3 秒（A.2）、忙的项目的测试的每个回答，交错 20 的持有（134 个 503 都答了，3 秒为限）和链的开始（每串一次回答，5 秒为限），两个忙的项目的测试的窗口以期限为界（`max_wait` 1 秒、锁的期限 3 秒；bob 的请求的期限 3 秒、alice 的等锁最早在 4 秒到期；预检 M3） | 写的等待都在组合一层（两个忙的项目的测试）；测试一侧都有 | 组合；审阅 |
| 30 每个输出、问题的细节和补救对每个收到它的调用者都成立 | `issue.not_found` 的文案；`server_busy`（M2 的文案：稍后重试）只在排满、等不到、在闸门上请求到期、等锁到期时答，都是暂时的 | 成立 | 审阅 |
| 31 检查和执行之间的 gate 在检查的事务之外 | 交错的门在第一个写的写之后、提交之前（`heldTx`）；1u 用同一个 | P1a 的 `serialize`、`serializeOn`，没有新写一个 | 组合 |
| 32 每个探针有反例 | 两个忙的项目的测试、交错 20、"工作项不在了"用已有的 `pgtest.WaitForLockWaitBehind`（等的是持有者的那个连接后面的等待，不是表名，等不到就在 5 秒的期限上失败）；交错照用 P1a 的 `serialize`（它的反例是 P1a 的 T12.3） | 没有新写的探针 | 审阅 |
| 33 goroutine 上不调用 `t.Fatal` | `replyTo`、`sendInBackground`、交错 20 的两组 goroutine | 闭包里只发请求、只送回回答 | 审阅 |
| 34 "没有替身"、"完成"的说法全包 grep | `issueWrites` 与契约中工作项一侧的写一一对应；矩阵覆盖每个操作；`apitest.Main` 两个方向；`archtest` 禁止命令行建闸门（T10.15）；"读不进闸门"在读的单元测试（`newGet` 用记下进出的闸门）和交错 20（读热的工作项）都核对（T10.18，预检 M2） | 都被发现 | 单元；组合 |
| 35 "由 X 写"的种子是规则允许的 | 矩阵的种子由工作区的管理员写（他建了这些项目，按 `createProject` 是它们的管理员；状态、标签也是他写的）；I2 经接口 | 规则到得了的状态；"工作项不在了"的删除和移走由事务写，测试写明 P1b 没有这两个操作 | 审阅 |
| 36 别的操作或两步到达同样的结果 | 再指派一个去掉过的账户、再加一个去掉过的标签：新的一行，旧的保留它的戳；已关注的人再被指派不再写关注 | I2 再加的一步 | 端到端 |
| 37 归档项目、删除的工作区、停用的调用者 | 已归档的项目 409（矩阵）；已删除的项目和工作区在锁下 404（3u）；停用的调用者由认证在请求之前拒绝 | T5.1 在组合一层 | 组合 |
| 38 故事的每个拒绝之前先读 | I2 的两个 422 | 之前一步的 `expectIssueUpdated`，之后再核对 | 审阅 |
| 39 测试里的排序与排序规则无关 | `rowsOf` 用 `COLLATE "C"`；`toSorted()`、`byId` 按 UTF-16 的码元比较 `id` | 与排序规则无关 | 审阅 |
| 40 每个交错的结果写明每个结束由谁 | 3u：删除由 alice；7u：Bug 由 alice 摘下；1u：bob 的动态 | 结果读出写者 | 审阅 |
| 41 每个写多行的语句与同一些行的其他写者 | `RemoveAssignees`、`RemoveLabels` 和三个插入（第 3 节第 12 条） | 都在项目 N 之下 | 审阅 |
| 42 锁语句之后的写只写锁住的行 | 修改写的是它在锁下重读、确认仍在锁住的项目里的工作项 | T5.16、T5.17 在组合一层（"工作项不在了"） | 组合 |
| 43 每个"由 X"可以失败 | 同清扫 14 | 同上 | 存储；端到端 |
| 44 "保留"、"还有"读状态 | 已删除的负责人、标签的行保留它的删除（T3.5、T3.7）；留下的已离开的负责人（矩阵） | T3.5、T3.7 在存储一层和 I2（"只去掉一个"的两步） | 存储；端到端 |
| 45 没有不会失败的断言 | P1b 的断言 | 原型的每个新的 Go 测试至少被一个变异弄失败（`checkkills.py`，规模的测量除外：它量耗时）；执行中规模的测量加了种子的核对（T13.1），新加的两个配置测试各有弄失败它的变异（本节末尾的合计） | 审阅 |
| 46 测试一侧的等待都有期限 | 种子、读回、持有事务时的语句；交错 20 等 134 个 503、链等每串的第一次回答，两个忙的项目的测试等每个回答（预检 M3：计数、开始和窗口不靠固定的时间） | 都在 `pgtest.Soon`、context 或测试自己的期限上（3 秒、4 秒、5 秒，到了算失败） | 审阅 |
| 47 连续的探测在同一张表上要指名 | 每个测试一次探测 | 没有连续的同表探测 | 审阅 |
| 48 堆序或索引序的夹具有前提 | 规模的测量按随机的次序写每张表；I2 按 `id` 的反序给出集合和再加的两个标签 | T1.5 在 I2 被发现 | 端到端 |
| 49 组合的夹具看得到过宽的写 | I2 的另一个工作项；存储测试的别的工作项、别的项目的行 | T3.1、T3.4、T3.11 在 I2 | 存储；端到端 |
| 50 每句注释和文档对照它说的代码 | 每个 Task 末尾 | 见清扫 6、15 | 审阅 |
| 51 数据分不出性质（brief） | 每个谓词、次序、过滤、去重的测试数据里有变异会错拿的行或错排的次序 | 原来的 I2 分不出三个（T3.6、T3.8、T3.10：没有留下的负责人和标签，去掉的人就是写那些行的人），改了数据（A.2）；再加的两个标签按 `id` 的反序给出（T1.5）；标签目录的回答当作集合（T2.1、T2.2）；盖戳的测试里 bob 原来指派自己，他的关注本来就在，改为指派 walt（T5.22，Task 7 起在组合一层，预检 L6）；完成组之间的移动写新的时刻只有单元的表分得出（预检的 PF.C1：数据的规则） | 单元；存储；组合；端到端 |
| 52 复制的测试辅助（brief） | 扩展已有的：`issueWorld`（`newIssueWorldWith`、`newWorkItemIn`）、`serializeOn`、`heldTx`、`holding`、`sendInBackground`、`workItemRows` | `TestTwoWritesOnAProjectSerialize` 改用 `serializeOn`；`aWorkItem` 用 `newWorkItemIn`；规模的测量用 `workItemRows`（A.2）；没有重写的辅助 | 审阅 |
| 53 任务结束时为假的注释和文档（brief） | `prophecy.py` 列出的每一行 | 五处改写（A.2），规模测试的"P9 量"一句改成设计的要求；预检之后 `issue/module.go` 的包说明写到取、改工作项（L4） | 审阅 |
| 54 文案给出没人能做的补救（brief） | `issue.not_found` 的文案；`project.state_in_use` 不改（P2 的事） | 没有补救 | 审阅 |
| 55 新的存储查询的谓词没有钉住（brief） | `update.sql`、`issue_ports.sql` 的每个 `WHERE`、`ANY` | 每个有一个存储测试的变异（清扫 1） | 存储 |
| 56 只从后面的 Task 才被发现的变异；容器起不来不算（brief） | "（Task n 起）"的行在那个 Task 的树上跑；构建失败、崩溃不算发现（`mut.py`），端到端的发现核对是断言失败（`e2ekills.py`） | 见变异表和各 Task 的表；T8.5 原来只在 `-race` 的负载测试被发现，负载测试每个 goroutine 1000 次之后 `make test`（不带 `-race`）也发现它（预检 L1） | 审阅 |

**变异表（按缺陷类别）**：变异的定义在 `$M4TMP/p1btools/mutants_p1b.py`（104 个，按 Task 分在 `mutants_b1.py`、`mutants_b2_4.py`、`mutants_b5.py`、`mutants_b6_7.py`、`mutants_b8_10.py`、`mutants_b12.py`；中文描述在 `mutants_zh_p1b.py`），结果在 `mut-results-pertask.json`（各自的 Task）、`mut-results-from.json`（"Task n 起"）、`mut-results.json`（最终的树），每次运行的日志在 `mut-logs-*`；预检之后（A.2）重跑的 27 个变异的结果在 `mut-results-amend-pertask.json`、`mut-results-amend-from.json`、`mut-results-amend.json`（`passAM.sh`，日志在 `mut-logs-amend*`），`mutresults.py` 把它们按层叠在先前的结果上，表从叠好的结果写出。原型的 104 个变异，103 个被发现，T8.11 是等价的（A.3）。在最终的原型上（端到端一层在 Task 14 的树上），单元一层的检查发现 77 个，组合 46 个，端到端（I2）35 个，存储 17 个，守卫 1 个（一个变异可以在几层被发现）；跑过而没有发现的只有 T8.11 的两个单元一层（闸门的测试和 `-race` 的负载测试）。只有一层发现的：单元 21 个、存储 4 个（都在 A.3），组合 8 个（T6.3、T7.1、T10.9–T10.11、T10.15、T10.16、T12.1），守卫 1 个（T4.5），端到端没有。"（Task n 起）"的：Task 6 起 6 个、Task 7 起 3 个、Task 10 起 5 个、Task 11 起 8 个、Task 12 起 4 个、Task 14 起 35 个（I2）。原型上 P1b 新加的 46 个 Go 测试中 45 个至少被一个变异弄失败（`checkkills.py`），没有的是规模的测量（它量耗时，第 3 节第 14 条）。下表是原型的 104 个，执行中记下的层标明；执行中加的 33 个和执行完的分支的合计在表后。

| 缺陷类别 | 变异 | 结果 | 失败的检查 | 层 |
|---|---|---|---|---|
| 清扫 24：修改的规则（3.7） | T1.1、T1.2、T1.3、T4.2、T5.5、T5.6、T5.7、T5.8、T5.9 | 9/9 | `TestCheckIssuePatchAccepts`、I2、`TestCheckIssuePatchReportsEveryProblem`、`TestEveryRuleDecidesItsCells`、`TestPermissionMatrix`、`TestUpdateIssueRefusesUnderTheLock`、`TestDeletingAStateAndPuttingAWorkItemInItSerialize`、等 4 个 | 单元；组合；端到端 |
| 集合的差（负责人、标签） | T1.4、T1.6 | 2/2 | `TestChangesOf`、I2 | 单元；端到端 |
| 约定五：按 `id` 的次序加 | T1.5 | 1/1 | `TestChangesOf`、I2 | 单元；端到端 |
| 什么都不改时不写（第 3 节第 4 条） | T1.7、T1.8、T1.9、T1.14、T1.15、T5.2 | 6/6 | `TestChangesOf`、I2、`TestChangesNone`、`TestUpdateIssueThatChangesNothingWritesNothing`、`TestPermissionMatrix`（T1.15、T5.2，Task 6 起，执行中记下） | 单元；组合；端到端 |
| 动态的第 3、6、7、10–13 行（3.13） | T1.10、T1.11、T1.12、T1.13、T5.12、T5.19 | 6/6 | `TestUpdateActivity`、I2、`TestUpdateIssueWritesTheChangesAndTheirRows`、`TestUpdateIssueStopsAtAFailingPort` | 单元；端到端 |
| 目录回答名称（第 3 节第 9 条） | T2.1、T2.2、T2.3 | 3/3 | `TestIssueCatalogs`、`TestUpdatingAWorkItemStampsItsRows`、I2、`TestIssueProjectsConvertsProjectsAnswers` | 单元；存储；组合；端到端 |
| 清扫 1、44、49：修改的存储的谓词 | T3.1、T3.4、T3.5、T3.6、T3.7、T3.10、T3.11 | 7/7 | `TestUpdatingAWorkItemWritesItsRows`、I2 | 存储；端到端 |
| 清扫 14：由谁 | T3.2、T3.8、T7.4 | 3/3 | `TestUpdatingAWorkItemWritesItsRows`、I2、`TestAWriteEntersItsProjectsGateBetweenItsPrecheckAndItsTransaction`、`TestUpdateIssueWritesTheChangesAndTheirRows`、`TestUpdateIssueCompletesTheWorkItemByItsState`、`TestUpdatingAWorkItemStampsItsRows` | 单元；存储；组合；端到端 |
| 完成时间（第 3 节第 5 条；完成组之间的移动写新的时刻只有单元一层分得出，是数据的规则：预检的 PF.C1） | T3.3、T5.4、T5.21 | 3/3 | `TestUpdatingAWorkItemWritesItsRows`、I2、`TestUpdateIssueWritesTheChangesAndTheirRows`、`TestUpdateIssueCompletesTheWorkItemByItsState` | 单元；存储；端到端 |
| 关注只给新负责人（3.12，第 3 节第 7 条） | T5.10、T5.22 | 2/2 | `TestUpdateIssueWritesTheChangesAndTheirRows`、`TestAWriteEntersItsProjectsGateBetweenItsPrecheckAndItsTransaction`、`TestUpdatingAWorkItemStampsItsRows` | 单元；组合 |
| 清扫 25：时刻在锁下读 | T5.3 | 1/1 | `TestAWriteEntersItsProjectsGateBetweenItsPrecheckAndItsTransaction`、`TestTheGateTakesOnlyWritesThePrecheckLetThrough`、`TestUpdateIssueWritesTheChangesAndTheirRows`、`TestUpdateIssueThatChangesNothingWritesNothing`、`TestUpdateIssueRefusesBeforeAnyLock`、`TestUpdateIssueRefusesUnderTheLock`、`TestUpdateIssueStopsAtAFailingPort`、等 1 个 | 单元；组合 |
| 清扫 5：预检和锁下的判定 | T4.1、T5.1、T5.15 | 3/3 | `TestGetIssueReadsTheWorkItem`、`TestGetIssueRefusals`、`TestAWriteEntersItsProjectsGateBetweenItsPrecheckAndItsTransaction`、`TestTheGateTakesOnlyWritesThePrecheckLetThrough`、`TestUpdateIssueWritesTheChangesAndTheirRows`、`TestUpdateIssueThatChangesNothingWritesNothing`、`TestUpdateIssueRefusesBeforeAnyLock`、等 5 个 | 单元；组合 |
| 按资源寻址的写：锁下重读 | T5.16、T5.17、T5.20 | 3/3 | `TestUpdateIssueRefusesUnderTheLock`、`TestAWorkItemGoneWhileItsChangeWaitsIsNotFound`、`TestAWriteEntersItsProjectsGateBetweenItsPrecheckAndItsTransaction`、`TestUpdateIssueWritesTheChangesAndTheirRows`、`TestUpdateIssueThatChangesNothingWritesNothing`、`TestTwoChangesOfAWorkItemSerialize` | 单元；组合 |
| 拒绝的码：`issue.not_found` | T4.3、T5.18 | 2/2 | `TestGetIssueRefusals`、`TestUpdateIssueRefusesBeforeAnyLock`、`TestPermissionMatrix`、`TestUpdateIssueRefusesUnderTheLock`、`TestAWorkItemsWriteAndDeletingItsProjectOrWorkspaceSerialize` | 单元；组合 |
| 清扫 9：写表的次序 | T5.11、T12.1 | 2/2 | `TestUpdateIssueWritesTheChangesAndTheirRows`、`TestChainedWorkItemWritesHoldOffNoWorkspaceWrite`、`TestEachIssueWriteLocksItsProjectFirst` | 单元；组合 |
| 清扫 13：取锁的形状 | T6.3 | 1/1 | `TestEachIssueWriteLocksItsProjectFirst`、`TestAWorkItemGoneWhileItsChangeWaitsIsNotFound`（Task 11 起，执行中记下） | 组合 |
| 清扫 8：事务的连接 | T7.1 | 1/1 | `TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection` | 组合 |
| 写闸门（`platform/gate`、取锁的一步） | T8.1、T8.2、T8.3、T8.4、T8.5、T8.6、T8.7、T8.8、T8.9、T8.10、T8.11、T10.1、T10.2、T10.3、T10.16、T10.17、T10.18 | 16/17（T8.11 等价，第 3 节第 11 条） | `TestWaitersAreServedInTheOrderTheyCame`、`TestAFullQueueRefusesAtOnce`、`TestAProjectTooBusyForAWriteAnswersServerBusy`、`TestLeavingTwiceLeavesOnce`、`TestAWaiterGivesUp`、`TestUnderLoadAKeyIsNeverHeldByMoreThanPerKey`、`TestAWriteWhoseRequestEndsWhileItWaitsAtTheGateIsServerBusy`、等 6 个 | 单元；组合 |
| 503 `server_busy` 的换码 | T10.4、T10.5、T10.6、T10.7、T10.8 | 5/5 | `TestTheWriteGateAnswersItsRefusalsAsServerBusy`、`TestAWriteWhoseRequestEndsWhileItWaitsAtTheGateIsServerBusy`、`TestAProjectTooBusyForAWriteAnswersServerBusy`、`TestABusyTransactionIsServerBusy` | 单元；组合 |
| 锁的期限（`LockTimeout`、`55P03`） | T9.1、T9.2、T9.3、T9.4 | 4/4 | `TestLockTimeoutBoundsEachWaitOfTheTransaction`、`TestAProjectTooBusyForAWriteAnswersServerBusy`、`TestLockNotAvailable` | 存储；组合 |
| `issue` 的配置 | T9.5、T9.6、T9.7、T9.8、T9.9、T9.10 | 6/6 | `TestValidateReportsEveryInvalidKey`、`TestValidateCrossKeyRules`、`TestBuiltInProfiles`、`TestLogValueMasksDatabaseURL` | 单元 |
| 清扫 4：接线 | T10.9、T10.10、T10.11、T10.12 | 4/4 | `TestAProjectTooBusyForAWriteAnswersServerBusy`、`TestTheWriteGateAnswersItsRefusalsAsServerBusy`、`TestAHotProjectHoldsUpNoReadOfAnother` | 单元；组合 |
| 命令行的组合 | T10.15 | 1/1 | `TestCommandsComposeNoServerAndNoJobs` | 组合 |
| HTTP 的映射和回答 | T4.6、T6.1、T7.2、T7.3 | 4/4 | `TestCreateIssue`、`TestGetIssue`、`TestUpdateIssue`、`TestPermissionMatrix`、I2、`TestAWorkItemReadsAsStored` | 单元；组合；端到端 |
| 清扫 18：契约 | T6.2、T10.13、T10.14 | 3/3 | `TestUpdateIssueHoldsTheBodyToItsStructure`、`TestUpdateIssueRefusals`、`TestAProjectTooBusyForAWriteAnswersServerBusy`、`TestCreateIssueRefusals` | 单元；组合 |
| 清扫 2、19：失败的传递 | T3.9、T5.13、T5.14 | 3/3 | `TestAFailedUpdateStatementIsAnError`、`TestAWriteLeavesTheGateOnEveryPath`、`TestUpdateIssueStopsAtAFailingPort` | 单元；存储 |
| 错误码的文案（约束 4） | T4.4、T4.5 | 2/2 | `error-messages.test.ts`、`check:sync` | 单元；守卫 |

**执行中加的变异**（33 个，编号接在各 Task 原来的之后；各 Task 的实现者、评审和整分支评审的探针，控制者把它们排进各 Task 的表，plan 的表里每行写明改法、必须失败的检查和层）。改法在 `$M4TMP/p1bsdd/`：`probes_t3_fix1.py`（T3.12、T3.13 是 PT3.6、PT3.7）、`probes_t4.py`（T4.7 是 PT4.3）、`t12/probes_t12.py`（T4.8、T4.9 是 PT12.2、PT12.3）、`probes_t5.py`（T5.23–T5.30 是 PT5.1–PT5.8）、`$M4TMP/p1bfinal/tools/mutants_final.py`（T5.31 是 RV5.1）、`probes_t14b.py`（T5.32 是 PT14.1）、`probes_t6_fix1.py`（T6.4 是 PT6.F2a、PT6.F2b 两种改法）、`probes_t7_fix2.py`（T7.5、T7.6 是 PT7.1、PT7.4）、`mutants_t8_review1.py`（T8.12–T8.14 是 PT8.1、RV8.1、RV8.2）、`probes_t9.py`（T9.11–T9.16 是 PT9.1–PT9.6）、`mutants_t9_fix2.py`（T9.17 是 PT9.7，Task 9 的修复轮写的）、`probes_t10.py`（T10.19、T10.20 是 PT10.1、PT10.2）、`ffw/mutants_ffw2.py`（T10.21，整分支评审之后的修复轮写的）、`t11/probes_t11.py`（T11.1 是 PT11.1；它的 PT11.2 与 T6.3 是同一个改法）、`probes_t13.py`（T13.1 是 PT13.1）。T10.21 之外的每个都在加它的 Task 的树上跑过，全部又在执行完的树上按 plan 的表的层跑过：整分支评审在 `3693abc7` 上跑了 T5.23、T5.26、T5.28、T5.31（单元一层、每个 `bootstrap` 测试和 I2）和 T5.32（单元一层和 I2）；T9.17 由范围复审在 `f18de99d` 上跑（`$M4TMP/p1bfinal/rr/`）；其余的由之后的修复轮在 `203f1f94` 上跑（T8.12 的 `-race` 一层除外：它在 Task 8 的树上，闸门的包此后只改了注释），连同较早的行在执行中新记的层：T1.15、T5.2 在矩阵，T2.2、T2.3 在盖戳的测试，T6.3 在"工作项不在了"（`$M4TMP/p1bsdd/ffw/` 的 `mutants_ffw.py`、`mutants_ffw2.py`–`mutants_ffw4.py`、`mutants_ffw6.py`、`mutants_ffw7.py`，结果在 `mut-results-*.json`、日志在 `mut-logs-*`；T13.1 设了 `NERVE_MEASURE_SCALE`）。每个都被发现，端到端的发现由 `e2ekills.py` 核对是断言失败，没有容器起不来的运行。

| 缺陷类别 | 变异 | 结果 | 失败的检查 | 层 |
|---|---|---|---|---|
| 清扫 1：`= ANY` 只收到给出的第一个 | T3.12、T3.13 | 2/2 | `TestUpdatingAWorkItemWritesItsRows`、`TestPermissionMatrix` | 存储；组合 |
| 清扫 5：判定的动作 | T4.7 | 1/1 | `TestGetIssueReadsTheWorkItem`、`TestGetIssueRefusals` | 单元 |
| 读不取锁（3.6，第 3 节第 13 条） | T4.8、T4.9 | 2/2 | `TestGetIssueReadsTheWorkItem`、`TestAHotProjectHoldsUpNoReadOfAnother`、`TestAProjectTooBusyForAWriteAnswersServerBusy`、`TestAWriteWhoseRequestEndsWhileItWaitsAtTheGateIsServerBusy`、等 7 个 | 单元；组合 |
| 清扫 2：失败的传递（锁下的重读） | T5.23 | 1/1 | `TestUpdateIssueStopsAtAFailingPort` | 单元 |
| 集合的差（负责人、标签） | T5.24、T5.25、T5.31 | 3/3 | `TestPermissionMatrix`、`TestUpdateIssueWritesTheChangesAndTheirRows`、I2 | 单元；组合；端到端 |
| 关注只给新负责人（3.12） | T5.26 | 1/1 | `TestUpdateIssueWritesTheChangesAndTheirRows` | 单元 |
| 动态的第 12、13 行（3.13） | T5.27 | 1/1 | `TestUpdateIssueWritesTheChangesAndTheirRows`、I2 | 单元；端到端 |
| 清扫 18：契约（问题的下标） | T5.28 | 1/1 | `TestUpdateIssueRefusesUnderTheLock` | 单元 |
| 清扫 5：已归档在什么都不改之前（第 3 节第 4 条） | T5.29 | 1/1 | `TestUpdateIssueRefusesUnderTheLock`、`TestPermissionMatrix` | 单元；组合 |
| 清扫 24：没给的字段不改（2.3） | T5.30 | 1/1 | `TestPermissionMatrix`、`TestEachIssueWriteLocksItsProjectFirst`、`TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`、`TestAWorkItemReadsAsStored`、`TestAWorkItemsWriteAndDeletingItsProjectOrWorkspaceSerialize`、`TestDeletingAStateAndPuttingAWorkItemInItSerialize`、`TestDeletingALabelAndLabellingAWorkItemWithItSerialize`、`TestChainedWorkItemWritesHoldOffNoWorkspaceWrite`、I2 | 组合；端到端 |
| 清扫 24：值在任何读之前检查 | T5.32 | 1/1 | `TestUpdateIssueRefusesBeforeAnyLock`、I2 | 单元；端到端 |
| 清扫 13：取锁的形状（每行发它的操作的路径） | T6.4 | 1/1（两种改法都被发现） | `TestEachIssueWriteLocksItsProjectFirst` | 组合 |
| HTTP 的映射和回答（`getIssue` 的完成时间） | T7.5、T7.6 | 2/2 | `TestAWorkItemReadsAsStored`、`TestGetIssueReadsTheWorkItem` | 单元；组合 |
| 写闸门（`platform/gate`、取锁的一步） | T8.12、T8.13、T8.14、T10.20 | 4/4 | `TestUnderLoadAKeyIsNeverHeldByMoreThanPerKey`、`TestWaitersAreServedInTheOrderTheyCame`、`TestAFullQueueRefusesAtOnce`、`TestAWriteLeavesTheGateOnEveryPath` | 单元 |
| `issue` 的配置 | T9.11、T9.12、T9.13、T9.14、T9.15、T9.17 | 6/6 | `TestValidateAcceptsTheWriteGatesMinimums`、`TestLogValueMasksDatabaseURL`、`TestValidateCrossKeyRules` | 单元 |
| 锁的期限（`LockTimeout`） | T9.16 | 1/1 | `TestLockTimeoutBoundsEachWaitOfTheTransaction` | 存储 |
| 清扫 4：接线（每个项目一个键） | T10.19 | 1/1 | `TestTheWriteGateAnswersItsRefusalsAsServerBusy`、`TestAProjectTooBusyForAWriteAnswersServerBusy` | 单元；组合 |
| 503 `server_busy` 的换码（排满） | T10.21 | 1/1 | `TestTheWriteGateAnswersItsRefusalsAsServerBusy`、`TestAProjectTooBusyForAWriteAnswersServerBusy`、`TestAHotProjectHoldsUpNoReadOfAnother` | 单元；组合 |
| 清扫 14：由谁（删除的写者） | T11.1 | 1/1 | `TestTheBatchStatementsDeleteAProjectsRowsInIDOrder`、`TestABatchPassesOverARowDeletedWhileItWaits`、`TestAWorkItemsWriteAndDeletingItsProjectOrWorkspaceSerialize`、`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt` | 存储；组合 |
| 清扫 10：规模的测量的种子 | T13.1 | 1/1 | `TestTheDeletionsAtTheScaleTarget`（设了 `NERVE_MEASURE_SCALE`） | 组合 |

**合计**（执行完的分支，变异表 137 行）：136 个被发现，T8.11 是等价的；单元一层的检查发现 101 个，组合 63 个，端到端（I2）40 个，存储 21 个，守卫 1 个（一个变异可以在几层被发现）。只有一层发现的：单元 35 个、存储 6 个（都在 A.3），组合 12 个（原来的 8 个和 T5.24、T6.4、T7.5、T13.1），守卫 1 个（T4.5），端到端没有。"（Task n 起）"的：Task 6 起 12 个、Task 7 起 6 个、Task 10 起 6 个、Task 11 起 10 个、Task 12 起 8 个、Task 14 起 40 个（I2）。执行中新加的 Go 测试两个（Task 9 的修复轮）：`TestValidateAcceptsTheWriteGatesMinimums` 被 T9.11、T9.12 弄失败；`TestValidateAcceptsWriteWaitsJustUnderTheRequestTimeout`（和的规则的另一侧：刚好短于请求的期限的照样接受）被一个只为核对它而跑、不进表的变异弄失败（规则多留一毫秒，`$M4TMP/p1bsdd/ffw/mutants_ffw5.py`）；规模的测量加了种子的核对之后被 T13.1 弄失败。
