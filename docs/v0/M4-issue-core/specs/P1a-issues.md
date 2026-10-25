# M4/P1a 工作项：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M4/P1a `issues` |
| 日期 | 2026-10-17 |
| 状态 | 已完成（[评审记录](../reviews/P1a-issues-review.md)）：第 3 节已由控制者裁定（2026-10-17：第 1–3、11 条是拆分裁定，其余照建议，`$M4TMP/p1a-architect-rulings.md`，标在各条之后）；预检的发现（High 0、Medium 0、Low 6）由修订一轮（`c37a3480`）改入；整分支评审（Critical 0、Important 1、Minor 8）之后的修复轮（`9e9db923`、`91e54133`）按评审记录第 4 节改入。P1 由控制者按决策点 2 的预先批准拆成 P1a、P1b（2026-10-17，`p1-split-rulings.md` 的八条裁定，设计第 12 节"拆分"一段）；P1a 的任务在约 16 个、每个约 1,500 行以内重新切分（裁定 3），没有用到退路（裁定 2） |
| 上级文档 | [M4 设计文档](../M4-design.md) 第 2（I1 与 M3 的 P4、W2、W3）、3.3、3.4（P1a 只有管理员和成员）、3.5（取锁的一步、预检、删除工作区的新顺序、交错 1–8、21、22 的创建一支）、3.6、3.7（P1a 的字段：不含描述、起止日期）、3.12（创建者、负责人的关注）、3.13（动态的第 1、2 行）、3.20（时钟）、3.23（P1a 的行）、4.1–4.5、4.11、4.16、4.17（P1a 的行）、4.18、5.1–5.3（`createIssue`）、6.1–6.7、9.1–9.6（P1a 的部分）、11.1、11.2、12（P1a 与约束 1–8）、13.1 节 |
| 前置交接 | [M3 收尾交接](../handoffs/M3-closeout.md)第 2、4 节；[M1/P2 交接](../handoffs/M1-P2-trim-content.md)、[M1/P3 交接](../handoffs/M1-P3-trim-platform.md)的接口字段一项。落点见第 3 节第 16 条、第 7 节 |
| 计划 | [P1a plan](../plans/P1a-issues.md) |

本 spec 只写 M4 设计交给 P1a 决定的东西：名字、签名、SQL、测试名，以及原型证明了什么。规则本身以 M4 设计为准，这里引用节号，不重述。P1a 从 `d693f539`（设计的拆分提交，`1eb9686a` 之后）开始。

P1a 是 M4 最评审敏感的一段（设计 12 节约束 3）：工作项的取锁约定第一次落地（读 → 不加锁的预检 → 项目 N → 重读 → 判定），删除工作区改为先锁它的项目、再读时刻，五张新表进入删除项目、删除工作区的连带。附录 A 的清扫逐个核对每个这样的测试在它所说的性质去掉之后失败，并记下失败在哪一层（单元、结构、存储、组合、端到端）；只在单元一层失败的安全或加锁的性质算缺口，按性质只能在单元一层的逐条写在第 3 节和附录 A。

## 1. 目标

按 M4 设计 12 节 P1a：工作项的五张表、项目行的锁、创建与连带。具体是：

- 迁移 `00015`–`00019`（`issues` 20 列、`issue_assignees`、`issue_labels`、`issue_subscribers` 各 10 列、`issue_activities` 17 列，4.2–4.5、4.11）、`sqlc.yaml` 的 `issue` 条目、运行时的授权；
- 新模块 `issue`（3.3、6.2）：`Provide` 交出 `IssueCascade`，`New` 接出 `createIssue`；`issue/domain` 创建一侧的规则和动态的第 1、2 行；`issue/app` 的端口（由使用方声明）、取锁的一步 `Locks`、`createIssue` 的用例；
- `project.Provide` 给 `issue` 的端口 `IssueProjects`（`ProjectGate` 的 `Find`、`Lock`，`Sequences`，`StateCatalog`，`LabelCatalog`，`ProjectMembers`），`identity` 的 `PublicProfiles` 经 `bootstrap/ports.go` 接给 `issue`；
- 每个工作项的写（P1a 只有创建）：读项目（不加锁）→ 预检（凡判定拒绝的 404、403 在这里答，不取任何锁）→ 事务 → 项目行 `FOR NO KEY UPDATE`（不锁工作区行）→ 锁下再判定（最终的）→ 依赖行的检查 → 编号 → 时刻 → 写（3.5，决策点 1 的 B）；
- 删除工作区的新顺序：工作区 N → 判定 → 邀请 N（`id` 升序）→ 项目 N（`id` 升序，`ProjectCascade.LockWorkspaceProjects`）→ 读时刻 → 写（3.5）；
- 连带：删除项目、删除工作区经过渡的 `IssueCascade.DeleteProjectIssues`、`DeleteWorkspaceIssues` 在请求的事务里分批软删除五张表的行（3.6，P9 换成任务）；删除状态在有工作项（含已归档的）时 409 `project.state_in_use`；删除标签经 `DetachLabel` 从工作项上摘下它和它的子标签；
- 接口 `createIssue`（`POST /api/v0/projects/{project_id}/issues`，`IssueCreate`、`IssueDetail`，5.1–5.3）；矩阵的行；M3 的最先锁工作区的测试收窄到项目管理类的写，新的取锁形状的测试（交错 21）；
- 交错 1、2、3、8、22 和交错 4–7 的创建一支，`-count=5 -race`，没有 40P01；
- 端到端：`createIssue` 的夹具、`assert/issue.ts`、`furnishIssueRows`（五张表），M3 的 P4、W2、W3 带着工作项一侧的行删除，`workspaceTables` 从目录读出，过渡结束；I1 的接口版本；3.23 中 P1a 的行。

不加 Go 模块、npm 包；跨模块的端口：`project` ↔ `issue` 两个（`IssueProjects`、`IssueCascade`），`workspace` → `project` 的 `ProjectCascade` 加一个方法；五张新表。

## 2. 交付物

### 2.1 文件总览

路径相对于仓库根目录（`server/internal/` 省略）。"Task"是 plan 中负责它的任务（plan 有完整的文件表）。本 Phase 手写的代码、测试和配置文件 140 个（新建 60 个，含 `issue` 模块的 `gen/oapi-codegen.yaml`；修改 80 个），生成物 14 个，文档 9 个（本 spec 和 plan 新建，7 个修改）；五个迁移、五张新表。

| 路径 | 内容 | Task |
|---|---|---|
| `tools/keywords.json`；`project/adapter/postgres/queries/cascade.sql`、`cascade.go`、`lock_workspace_projects_test.go`、`failures_test.go`；`project/app/ports.go`、`cascade.go`、`cascade_test.go`；`project/module.go`；`workspace/adapter/postgres/queries/invitations.sql`、`invitations.go` 和三个存储测试；`workspace/app/ports.go`、`delete_workspace.go` 及测试、两个假实现、`clock_test.go`；`bootstrap` 的四个交错测试 | 本 Phase 的第一个提交（关键词守卫）；删除工作区的新顺序 | 1 |
| `server/migrations/sql/00015_issue_issues.sql`、`server/sqlc.yaml`、`deploy/runtime-grants.sql`、`server/migrations/schema_test.go`、`issue_schema_test.go`；`issue/module.go`、`issue/app/cascade.go`、`delete_project_rows.go`、`issue/adapter/postgres/store.go`、`delete_rows.go`、`queries/delete_rows.sql`；`project/app/issue_ports.go`、`delete_project.go`、`cascade.go` 及测试、`fakes_issue_test.go`；`project/module.go`；`workspace/app/delete_workspace.go`；`bootstrap/app.go`、`users.go`；`archtest`；`bootstrap` 的删除和交错测试；`e2e/fixtures/assert/project.ts` | `issues` 表、`issue` 模块的骨架、两个连带的第一张表 | 2 |
| `00016`–`00019`；`sqlc.yaml`、授权；`delete_rows.sql`（整个文件）、`delete_rows.go`、`delete_project_rows.go`、`issue/app/cascade_test.go`；迁移的测试；`bootstrap` 的两个按目录的删除测试；`assert/project.ts` | 其余四张表进入两个连带 | 3 |
| `issue/adapter/postgres/store_test.go`、`delete_rows_test.go`；`issue_schema_test.go` | 键集的存储测试；CHECK 的反例、唯一键 | 4 |
| `issue/domain/issue.go`、`activity.go`、`errors.go` 及测试 | 创建一侧的规则；动态的第 1、2 行 | 5 |
| `project/adapter/postgres/queries/issue_ports.sql`、`issue_ports.go`、`issue_ports_test.go`、`failures_test.go`；`project/app/issue_ports.go`；`project/module.go` | `project.Provide` 的 `IssueProjects` | 6 |
| `issue/adapter/postgres/queries/issues.sql`、`issues.go`、`issues_test.go`；`issue/app/rows.go` | 创建的存储 | 7 |
| `issue/app/ports.go`、`lock.go`、`create_issue.go` 及测试、`fakes_test.go`；`issue/domain/actions.go`；`access/domain/rules.go` 及测试；`issue/module.go`；`bootstrap/actions_test.go` | 取锁的一步、`createIssue` 的用例、`issue.create` | 8 |
| `api/modules/issue.yaml`、`api/openapi.yaml`、`api/modules/project.yaml`；`issue/adapter/http/*`；`issue/module.go`；`bootstrap/app.go`、`ports.go` 及测试；`bootstrap/permission_matrix_issues_test.go`、`permission_matrix_test.go`、`project_write_locks_test.go` | `createIssue` 的接口、接线、矩阵；M3 的最先锁工作区的测试收窄 | 9 |
| `bootstrap/issue_world_test.go`、`issue_write_locks_test.go`、`issue_writes_test.go` | 取锁的形状（交错 21）；盖戳；事务的连接 | 10 |
| `issue/adapter/postgres/queries/cascade.sql`、`cascade.go`、`cascade_test.go`；`issue/app/cascade.go`（整个文件）；`project/app/delete_state.go`、`delete_label.go`（整个文件）及测试；`project` 的存储、HTTP、错误码；契约；前端的三个文案文件；`bootstrap/issue_cascade_test.go` 等 | `StateInUse`（409 `project.state_in_use`）、`DetachLabel` | 11 |
| `bootstrap/issue_interleaving_test.go`、`issue_interleaving_deletion_test.go`；`issue_writes_test.go`、`permission_matrix_issues_test.go`、`project_connection_test.go`、`ending_connection_test.go` | 交错 1、2、3、8、22，交错 4、5 的创建一支；分诊状态的矩阵行 | 12 |
| `bootstrap/issue_interleaving_catalog_test.go` | 交错 6、7 的创建一支 | 13 |
| `e2e/fixtures/issues.ts`、`states.ts`、`furnish.ts`、`assert/issue.ts`、`assert/catalog.ts`、`api.ts`、`assert/project.ts`、`assert/workspace.ts`；`e2e/stories/issue/i1-create-issue.spec.ts`；P4、W2、W3（拆出 `w3-workspace-settings-page.spec.ts`）、W11；P6 的 `p6-states.spec.ts`、`p6-states-page.spec.ts` | 夹具（`states.ts`：`createState` 从 `api.ts`、`listStates` 和 `stateIds` 从 P6 的两个文件移来，I1 和 P6 共用）；I1 的接口版本；M3 的删除故事带着工作项一侧的行；过渡结束 | 14 |
| `docs/v0/v0-design.md`、`plane-diff.md`、`M3-design.md`；三份交接 | 3.23 中 P1a 的行；交接的处理结果 | 15 |

生成物：`project/adapter/postgres/gen/cascade.sql.go`、`workspace/adapter/postgres/gen/invitations.sql.go`（Task 1）；`issue/adapter/postgres/gen/db.go`、`models.go`、`delete_rows.sql.go`（Task 2、3）；`project/adapter/postgres/gen/issue_ports.sql.go`（Task 6）；`issue/adapter/postgres/gen/issues.sql.go`（Task 7）；`api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`、`issue/adapter/http/gen/server.gen.go`、`bodyshape.gen.go`、`project/adapter/http/gen/server.gen.go`（Task 9）；`issue/adapter/postgres/gen/cascade.sql.go`、`project/adapter/postgres/gen/labels.sql.go`、`api/dist/openapi.yaml`、`schema.gen.ts`（Task 11）。

### 2.2 依赖

不加 Go 模块、npm 包。五个迁移（`00015`–`00019`）；`deploy/runtime-grants.sql` 给运行时的角色五张表的读写（Task 2、3）。`access` 的规则表加一行 `issue.create`（项目级，管理员、成员）。跨模块的端口都由使用方声明（M3 设计 6.6）：`issue/app` 声明 `ProjectGate`、`Sequences`、`StateCatalog`、`LabelCatalog`、`ProjectMembers`、`PublicProfiles`，`bootstrap/ports.go` 把 `project.Provide` 的 `IssueProjects`、`identity` 的 `PublicProfiles` 转换给它；`project/app` 声明 `IssueCascade`，`issue.Provide` 实现；`workspace/app` 的 `ProjectCascade` 加 `LockWorkspaceProjects`，`project` 的 `Cascade` 实现。没有模块之间的导入（`archtest`），`issue` 的 sqlc 只读写自己的五张表（`TestSQLCSchemaScope`）。

### 2.3 本 Phase 的第一个提交和删除工作区的新顺序（Task 1；3.5，7.9）

- `tools/keywords.json`：`phase` 改为 `M4/P1`（两段都写 `M4/P1`，裁定 5），`/user-properties/` 的一条例外（命中两处）的 `until` 改为 `M4/P11`（约束 6）。
- `project`：`queries/cascade.sql` 的 `LockWorkspaceProjects`（这个工作区未删除的项目，`ORDER BY id FOR NO KEY UPDATE`，回答 `id`）；`Store.LockWorkspaceProjects`；端口 `WorkspaceProjectsLocker`（`app/ports.go`）；`Cascade.LockWorkspaceProjects`（`app/cascade.go`，`NewCascade` 的第二个参数是 locker）；`module.go` 的 `Cascade` 接口加这个方法。
- `workspace`：邀请的锁与删除分开：`WorkspaceDeleter.LockWorkspaceInvitations`（这个工作区未删除的邀请，`id` 升序 `FOR NO KEY UPDATE`，回答 `id`）和 `DeleteInvitations(ids, by, now)`；存储的 `DeleteWorkspaceInvitations` 删去。`ProjectCascade` 加 `LockWorkspaceProjects`。`DeleteWorkspace.Execute`：工作区 N → 判定 → 邀请 N → 项目 N → `u.clock.Now()` → `cascade`（工作区、锁住的邀请、成员、显示设置、`DeleteWorkspaceProjects(ws, projects, by, now)`）。
- 测试：单元的 `deletionLocks`、`cascadeCalls`（次序）、`clock_test.go`（时钟在两步锁之后）；存储的 `TestLockWorkspaceProjects`、`TestLockWorkspaceProjectsLocksInIDOrder`（两个连接，按 `id` 等）、`TestLockWorkspaceInvitationsLocksInIDOrder`、失败测试；组合的 `TestAWorkspacesDeletionWaitsForAWriteHoldingOneOfItsProjects`（持项目 N 的写提交之后，删除才读时刻）。M3 的交错测试照旧通过（`NewCascade` 多一个参数）。
- 为什么在 Task 1（设计的任务 10）：`DeleteWorkspaceIssues` 要这次锁住的项目（第 3 节第 2 条）。

### 2.4 表、`issue` 模块的骨架和两个连带（Task 2、3、4；3.3、3.6、4.1–4.5、4.11、4.16）

- 迁移：`00015_issue_issues.sql`（Task 2）、`00016`–`00019`（Task 3），列、约束和索引照 4.2–4.5、4.11（差异清单二·按表逐列）。`issues_project_id_sequence_id_key` 全表唯一；`issues_not_own_parent_check`、`issues_dates_check`；外键列只留不带条件的索引，加 `issues_project_id_state_id_idx ON (project_id, state_id) WHERE deleted_at IS NULL`；三张关联表各一个部分唯一索引；动态的 `issue_activities_issue_id_created_at_idx ON (issue_id, created_at, id)`。每张表带 `workspace_id`、`project_id`（按目录的删除守卫和 `expectProjectDeleted` 要求）。迁移的测试：`TestMigrationsGoUpDownAndUpAgain`（19 个）、`TestConstraintAndIndexNames`（`issueNames`，`issue_schema_test.go`）、`TestIssueChecksRejectCounterexamples`、`TestIssueUniqueKeys`（Task 4）。
- `issue` 模块（Task 2）：`module.go`（`Provide(pool) Provided{IssueCascade}`，`Cascade` 接口，`deleteBatch = 1000`）；`app/cascade.go`（`NewCascade(deleter, …, limit)`：`DeleteProjectIssues`、`DeleteWorkspaceIssues`，Task 11 加 `StateInUse`、`DetachLabel`）；`app/delete_project_rows.go`（`RowBatch`、`ProjectRowsDeleter`、`rowSteps`：全局顺序 `issues` → `issue_assignees` → `issue_labels` → `issue_subscribers` → `issue_activities`；`deleteProjectRows`：每张表从 `after` 起每批 `limit` 行，下一批从这一批的最大 `id` 之后，直到一批为空）；`adapter/postgres/store.go`、`delete_rows.go`、`queries/delete_rows.sql`（每张表一条：`UPDATE … WHERE t.id IN (SELECT r.id … WHERE r.project_id = $p AND r.deleted_at IS NULL AND r.id > $after ORDER BY r.id LIMIT $batch FOR NO KEY UPDATE) RETURNING t.id`，写删除的时刻和删除者）。这是 P9 的任务将来调的同一份编排和语句（设计 6.2，决策点 7）。
- `project`（Task 2）：`app/issue_ports.go` 的 `IssueCascade`（使用方声明）；`DeleteProject` 的最后一步 `DeleteProjectIssues`；`Cascade.DeleteWorkspaceProjects(ctx, ws, projectIDs, by, now)` 的最后一步 `DeleteWorkspaceIssues(projectIDs, …)`；`CascadeDeps.Issues`、`Deps.Issues`。`bootstrap/app.go`、`users.go`（命令行的组合）先 `issue.Provide`，再把 `IssueCascade` 交给 `project`；`archtest` 的 `Users` 也建它。
- 按目录的删除测试（Task 2、3）：`TestDeletingAProject…`、`TestDeletingAWorkspace…` 照 M3 读出指向 `projects`、`workspaces` 的每个外键，五张表各种下一行（`seedIssues`：一个工作项、它的负责人、标签、关注、动态，由不是删除者的账户写；工作区种两个项目 Web、Ops），不开豁免；删除工作区最后一步失败的测试改为让工作项一侧失败（请求期间把 `issue_activities` 改名），之前的步骤都回滚。
- 单元（Task 3）：`TestTheCascadeDeletesEachProjectsRowsInBatches`（`limit` 2，五张表的次序、`after` 的推进、空批次结束）、`TestTheCascadeStopsAtAFailingBatch`。存储（Task 4）：`TestTheBatchStatementsDeleteAProjectsRowsInIDOrder`（每条语句：`after` 之后的、`id` 升序、至多 `limit` 行、只这个项目的、跳过已删除的、写下时刻和删除者；读失败时回答失败，不当作没有行）。
- 端到端的过渡（Task 2 到 Task 13，第 3 节第 11 条，裁定 7）：`expectProjectDeleted` 对五张表要求"没有随项目删除的行"（`withoutRows`），说明写明到哪里；Task 14 删去。

### 2.5 创建一侧的规则（Task 5；3.7、3.12、3.13）

`issue/domain`：
- `issue.go`：`Priority`（五个值）、`StateGroup`、`Issue`、`NewIssue`、`CheckNewIssue`（名称去掉首尾空白之后 1–255 个字符、没有 NUL；优先级；负责人、标签各至多 100 个、不重复，`duplicate`；全部问题一次报出）、`CanAssign`（项目的管理员、成员）、`Assignees`（请求的负责人按 `id` 排序、各须能指派，否则 `assignee_ids[i]` `not_allowed`；不带时项目的默认负责人在他能指派时，否则没有）、`UnknownLabel`、`StateNotAllowed`、`SortOrderAfter`（最大值加 10000，没有时 65535）、`CompletedAt`（"已完成"组为当前时刻）。
- `activity.go`：`Verb`、`Activity`、`CreationActivity`：第 1 行（`created`，"created the work item"），第 2 行每个负责人一行（`field` 为 `assignees`、`old_value` 为空串、`new_value` 是他的显示名、`new_identifier` 是他的 id、"added assignee "）。
- `errors.go`：`ErrProjectNotFound`、`ErrProjectArchived`（`project.not_found`、`project.archived`：P1a 没有新的 `issue.*` 码）。

### 2.6 `project.Provide` 的端口（Task 6；6.5）

`queries/issue_ports.sql`：`LockIssueProject`（`FOR NO KEY UPDATE`，回答工作区、是否归档、默认负责人）、`NextIssueSequence`（`UPDATE … SET last_issue_sequence = last_issue_sequence + 1 … RETURNING`）、`IssueStates`（未删除的状态，含分诊，`id` 升序）、`IssueLabels`（给出的 id 中这个项目未删除的标签）、`ActiveRoles`（给出的账户中有效成员的角色）。`Store` 的同名方法；`project/app/issue_ports.go` 的 `IssueProject`、`IssueState`；`module.go` 的 `IssueProjects` 接口（不加锁的 `ProjectWorkspace` 用 M3 已有的读）和 `Provided.IssueProjects`。存储测试：`TestLockIssueProject`（锁的强度：外键的 `FOR KEY SHARE` 不等它、`FOR SHARE` 等它，工作区行上没有锁；已删除的不回答）、`TestNextIssueSequence`、`TestIssueCatalogs`，失败测试各一行。

### 2.7 创建的存储（Task 7；3.7、3.12、3.13）

`queries/issues.sql`：`GreatestSortOrder`（这个项目、这个状态未删除的工作项（含已归档的）的最大值）、`InsertIssue`、`InsertAssignees`、`InsertLabels`、`InsertSubscribers`（`ON CONFLICT … DO NOTHING`：已关注的不再关注）、`InsertActivity`、`GetIssue`（未删除的工作项和它未删除的负责人、标签，各按 `id` 升序；没有时数组为 NULL，"列表从不为 null"由 HTTP 的 `orEmpty` 一处负责，第 3 节第 9 条）。`app/rows.go` 的 `IssueRow`、`LinkRows`、`ActivityRow`。存储测试 `TestCreatingAWorkItemWritesItsRows`、`TestGreatestSortOrder`、`TestInsertSubscribersSubscribesEachAccountOnce`，失败测试。

### 2.8 取锁的一步和 `createIssue` 的用例（Task 8；3.5、3.7）

- `issue/app/ports.go`：上面的六个端口和 `Clock`、`IssueCreator`（端口由使用方声明）。
- `issue/app/lock.go`：`Locks`（`NewLocks(projects, auth)`，每个写共用一个）、`write{project, action, notFound}`；`precheck`（`Find` → `decide`：看不到 `notFound`、角色不允许 `shared.Forbidden`，不取锁、不开事务）、`lockAndDecide`（`Lock` → `decide`，最终的；回答锁住的项目）、`decide`。判定的 `Grant` 在 P1a 没有读者（P1 只有管理员和成员），不带出来；访客规则（P2）要它时再加。P1b 的按资源寻址的写在这条路径上加"读资源、锁下重读"一步，不另写一份（第 5 节）。
- `create_issue.go`：`Execute`：`CheckNewIssue` → `precheck` → 事务：`lockAndDecide` → 已归档 409 → `check`（状态：给出的须是项目的、不是分诊的，否则 `state_id` `not_allowed`，不给时默认状态；负责人经 `Assignees`，角色经 `ProjectMembers`；标签经 `LabelCatalog`；问题一次报出）→ `Sequences.Next` → `Clock.Now()` → `GreatestSortOrder` → 写：工作项、负责人、标签、关注（创建者和负责人，`id` 升序）、动态（显示名经 `PublicProfiles`，在事务里）→ 重读回答。
- `issue/domain/actions.go` 的 `issue.create`，`access` 的规则行（项目级，管理员、成员）和 `rules_test.go` 的一行；`issue.Actions()`；`bootstrap/actions_test.go` 的模块清单（设计 12 节约束 4：动作随它的操作）。
- 单元测试（假实现 `fakes_test.go`，调用按次序记下）：`TestCreateIssueWritesTheWorkItemAndItsRows`（次序、写下的行）、`TestCreateIssueDefaults`、`TestCreateIssueRefusesBeforeAnyLock`（预检拒绝的不取锁、不开事务）、`TestCreateIssueRefusesUnderTheLock`（预检放行、锁下拒绝；依赖行的问题一次报出）、`TestCreateIssueStopsAtAFailingPort`（每个端口的失败原样返回，之后不再调用；不可能的情况也是错误：没有显示名的负责人、没有默认状态的项目、写入之后读不回来的工作项）；依赖行的两个问题由 Task 5 的 `TestDependentRowProblems` 钉住。

### 2.9 `createIssue` 的接口和组合（Task 9；5.1–5.3、6.6、9.2、9.4）

- 契约 `api/modules/issue.yaml`：`createIssue`（`x-problem-codes`：`validation_failed`、`project.not_found`、`forbidden`、`project.archived`；描述写明默认状态、默认负责人"仍是项目中不是访客的有效成员时"、编号、排序值、完成时间、关注、动态）、`IssuePriority`、`IssueDetail`、`IssueCreate`；`api/openapi.yaml` 的标签。`project.yaml` 的 `Project.identifier`、`default_assignee_id`、`State.default` 的说明改成它们的作用（M3 交接第 4 节）。
- `issue/adapter/http`：`gen/oapi-codegen.yaml`、`handler.go`（`UseCases`、`Register`）、`issues.go`（`createIssue`、`issueDetail`、`orEmpty`）；`apitest.Main("issue")` 两个方向核对码。
- 接线：`issue.New(Deps{Pool, Tx, Clock, Authorizer, Projects, Profiles})`；`bootstrap/app.go` 的组合顺序 `access` → `issue` → `project` → `workspace` → `identity`（6.6）；`ports.go` 的 `issueProjects`、`issueProfiles` 和它们的转换测试。
- 矩阵（`permission_matrix_issues_test.go`）：`createIssue` 六行（不带字段、每个字段都给、已归档、另一个项目的状态、另一个项目的标签、指派访客），Task 12 加"分诊状态"；`prepareMatrix` 不改。
- `project_write_locks_test.go`：`writesOnAProject` 去掉标签为 `issue` 的写（它们不锁工作区行），M3 的 `TestEachWriteOnAProjectSharesItsWorkspaceFirst` 收窄到项目管理类的写（设计 9.3）；形状测试的操作带上真实的标签（`issue` 的写被留在外面时失败）。

### 2.10 取锁的形状、盖戳和事务的连接（Task 10；3.5 交错 21，9.3）

- `issue_world_test.go`：`newIssueWorld`（acme、Web、Todo、Bug；bob 是 Web 的成员；walt 是工作区成员而不是 Web 的；wanda 是工作区管理员而不是 Web 的；olga 是外人）、`send`、`fullIssue`。
- `issue_write_locks_test.go`：`issueWrites`（契约中工作项一侧的每个写各一行，完整性核对）、`TestEachIssueWriteLocksItsProjectFirst`：写的第一把锁是项目的 N（它先等 `projects`）；持写在锁下最先读、预检没有读的表（`states`）的 `ACCESS EXCLUSIVE` 锁时，写在它自己的任何写之前（编号的 `UPDATE` 之前）停下，持项目行 `FOR NO KEY UPDATE`（由它的锁，不更弱、不更强）、工作区行上没有锁（只有外键的 `FOR KEY SHARE`）；看不到项目（olga）、看得到而不能写（walt 在公开的 Web 上、wanda）的写回答 404、403，不取行锁、不写（另一个事务以 `EXCLUSIVE` 持着它读写的每张表（`users` 除外）时照样回答）；工作区行被持 N 时写照样完成。
- `issue_writes_test.go`：`TestCreatingAWorkItemStampsItsRows`（七行同一时刻、同一个写者，在请求之内；动态的显示名来自 `users.display_name`）、`TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`（一个连接的池：每个决定所依赖的读都在写的事务里）。

### 2.11 删除状态和删除标签（Task 11；3.6、3.7）

- `issue`：`queries/cascade.sql` 的 `StateInUse`（未删除的工作项，含已归档的）、`DetachLabels`（这些标签未删除的 `issue_labels` 行，写删除的时刻和删除者）；`Store`；`app/cascade.go`（整个文件）加两个方法。
- `project`：`domain.ErrStateInUse`（409 `project.state_in_use`）；`DeleteState`：`lockRowAndDecide` 之后、守卫的删除之前问 `StateInUse`（所以默认状态有工作项时答 `state_in_use`）；`DeleteLabel`：存储的 `DeleteLabel` 回答删除的 `id`（这个标签和它下面的标签），用例在同一时刻 `DetachLabel(ids, by, now)`。`IssueCascade` 的说明。
- 契约：`deleteState` 声明 `project.state_in_use`；`deleteProject`、`deleteWorkspace` 的描述提到工作项；`deleteLabel` 的描述写明同一时刻、由同一个调用者从工作项上摘下。前端：`web/apps/web/core/lib/error-messages.ts` 和两份 `errors.json` 的 `project_state_in_use`（约束 4）。
- 测试：单元的 `delete_state_test.go`、`delete_label_test.go`、`clock_test.go`；存储的 `TestStateInUse`、`TestDetachLabels`、`label_writes_test.go`（回答的 `id`）；HTTP 的 `state_writes_test.go`；组合的 `TestDeletingAStateAWorkItemIsInIsRefused`（Review、默认的 Backlog 各有工作项时 409，什么都不改）、`TestDeletingALabelTakesItOffTheWorkItems`；M3 的状态、标签的交错改用 `issue.Provide(pool).IssueCascade`。

### 2.12 交错（Task 12、13；3.5，9.3）

`issue_interleaving_test.go` 的工具：`heldTx`（写的事务在它的 `fn` 之后、提交之前停在门上）、`exchange{req, rec, released}`、`racingWrite`、`serialize`（第一个写停在门上时，第二个写在 `projects` 上等锁（`pgtest.WaitForLockWaitOn`），再开门）、`inBothOrders`；写经各模块的路由（`newIssueRoute`、`newProjectRouteOn`、`newWorkspaceRoute`，各带自己的 `TxManager`）。

| 交错 | 测试 | 断言 |
|---|---|---|
| 1 | `TestTwoCreationsOnAProjectSerialize` | 编号 1、2，第二个的 `sort_order` 在第一个之后，第二个的时刻不早于门打开 |
| 2、3 | `TestCreatingAWorkItemAndDeletingItsProjectOrWorkspaceSerialize`（Web、acme，各两种顺序） | 先建：七行随项目或工作区删除，时刻不早于门打开；先删：404，没有行 |
| 4、5（创建一支） | `TestACallerRemovedOrDemotedWhileHisCreationWaitsIsRefused` | 预检放行之后被移出：404 `project.not_found`；被降为访客：403 |
| 6（创建一支） | `TestDeletingAStateAndCreatingAWorkItemInItSerialize` | 先删：422 `state_id`；先建：409 `project.state_in_use`，Review 保留 |
| 7（创建一支） | `TestDeletingALabelAndCreatingAWorkItemWithItSerialize` | 先删：422 `label_ids[0]`；先建：Bug 在删除的时刻由删除者摘下 |
| 8 | `TestArchivingAProjectAndCreatingAWorkItemSerialize` | 先归档：409；先建：建成，之后归档 |
| 22 | `TestRemovingTheDefaultAssigneeAndCreatingAWorkItemSerialize` | 先移出：没有负责人；先建：指派给他，移出不改这一行 |

分诊状态 422 经接口造不出（M3 不许建分诊状态），矩阵的"the triage state"一行用矩阵经存储种下的 Triage（照 `createProject` 的写法；接口的状态列表不含它）：能建工作项的格子答 422 `state_id`，其余的格子在判定时答 403（Task 12）。全部 `-count=5 -race`，没有 40P01（附录 A）。

### 2.13 端到端（Task 14；第 2 节 I1、P4、W2、W3，3.6）

- 夹具：`e2e/fixtures/issues.ts`（`Issue`、`IssueCreate` 来自 `schema.gen.ts`，`issueCreated` 回答状态和工作项或问题，`createIssue` 要求 201）；`e2e/fixtures/furnish.ts`（`furnishIssueRows(api, token, projectId, assigneeId, labelId)`：一个工作项，指派一人、带一个标签，五张表各至少一行；`furnishWorkspace` 从 `api.ts` 移来，给 Web 一个标签和这样一个工作项）；`assert/issue.ts`（`expectIssueCreated`：工作项的行、编号等于项目的 `last_issue_sequence`、负责人、标签、关注、动态，以及每一行的工作区、项目、写者、执行者和时刻都是创建的）。
- `assert/catalog.ts`（新）：`tablesUnder(db, parent, column)` 从目录读出外键指向 `parent` 的表，并核对它就是有 `column` 列的每张表；`projectTables`、`workspaceTables` 是它的两个用法。`expectProjectDeleted` 读 `projectTables`，过渡的 `withoutRows` 删去；`assert/workspace.ts` 的手写清单 `workspaceTables`、`deletedAloneTables` 删去，`expectWorkspaceDeleted` 读 `workspaceTables`，它的 `deletedAlone` 由故事给出。
- 故事：P4（接口版本：删除之前暂时恢复、成员建一个工作项、再归档；页面版本：wes 被指派）、W2（`furnishWorkspace`）、W3（Acme 的 Web、Old 各一个工作项；Other 的 Web 没有；拆成接口版本和页面版本两个文件，第 3 节第 12 条）、W11（表的清单从目录读）带着工作项一侧的行；I1 的接口版本（`i1-create-issue.spec.ts`）：ann 在 In Progress 建一个工作项，优先级 high，指派 bob 和管理员，标签 UI、Bug：编号 1、排在它的状态第一、负责人和标签按 `id`、创建者和负责人关注、创建和每个负责人的动态、同一时刻；默认负责人 ann 时 bob 建的下一个在 Backlog、指派给 ann；ann 离开之后再建：没有负责人，排在前一个之后；在 Cancelled 建的不算完成；同一个账户、标签给两次和大小写不对的优先级 422，Web 没有的状态、标签和指派访客 422，各自的问题一次报出；访客 403、已归档的项目 409。页面版本在 P14。

### 2.14 文档（Task 15；3.23）

总体设计 4.2（加锁顺序延伸到 M4 的表；工作项的写最先锁项目行；预检；删除工作区的新顺序；B 的两个代价）、6.2（`issue` 和它的 13 张表）；M3 设计 3.6 的取锁表（删除工作区、删除项目、删除状态、删除标签四行）、3.17（`project.state_in_use`）、17.4（M4 的选择）；差异清单二·按表的五张表、第四节中 4.17 标 P1a 的行；三份交接写下"处理结果（M4/P1a）"（第 7 节）。

## 3. 与设计的差异和补充

第 1、2、3、11 条已由控制者的拆分裁定覆盖（`p1-split-rulings.md`）；第 4–10、12–15 条是本 spec 对设计交给 P1a 的事的决定，控制者都照建议裁定（2026-10-17，`$M4TMP/p1a-architect-rulings.md`，标在各条之后）；第 16 条是承接条目的落点。

1. **拆分与重新切分（裁定 1、3）**。设计 12 节 P1a 的十五个任务草稿与 plan 的十五个 Task 的对应：草稿 1（第一个提交、五张表、骨架、两个连带、过渡）拆成 Task 2（`issues` 一张表和骨架：原型量到草稿 1 一个任务 1,815 行代码块，超过上限）和 Task 3（其余四张表）；草稿 10（删除工作区的新顺序）提到 Task 1（下一条）；草稿 2、3、4 是 Task 4、5、6，草稿 6 是 Task 7，草稿 5、7 合为 Task 8（取锁的一步没有使用者时过不了 lint，附录 A.1），草稿 8、9 是 Task 9、10，其中动作 `issue.create` 和它的规则行从草稿 1 移到 Task 8（用例：设计 12 节约束 4 要求动作随它的操作；`TestEveryModuleDeclaresItsActionsOrHasNone` 允许 Task 2–7 的 `issue` 没有动作）；草稿 11–15 是 Task 11–15。每个 Task 在 plan 里的行数见附录 A.1，最大的是 Task 2（1,435 行）。
2. **删除工作区的新顺序在 Task 1**。`IssueCascade.DeleteWorkspaceIssues` 要这次锁住的项目（`DeleteWorkspaceProjects(ctx, ws, projectIDs, by, now)`），而这些项目由 `LockWorkspaceProjects` 回答；把新顺序放在连带之前，Task 2 的连带直接接在它上面，不必先写一个按工作区读项目的过渡再删去。
3. **规模的测量在 P1b**（裁定 4）。设计的决策点 7 和 9.3 说 P1 量 2 万个工作项的删除；P1a 写下同步的过渡写法，P1b 不改它，测量放在 P1b 的 spec（P1a 没有用退路，没有腾出地方）。
4. **`DeleteWorkspaceIssues` 的签名不带工作区**（裁定：照本条，P8 的 spec 加这个参数）。设计 3.3 写 `DeleteWorkspaceIssues(ctx, workspaceID, projectIDs, by, now)`，工作区的 id 是为没有项目的草稿（P8）。P1a 的签名是 `(ctx, projectIDs, by, now)`：P1a 没有草稿，带一个不读的参数是没有使用者的代码（约束：不留没有使用者的代码）。P8 加草稿时加这个参数，接口一处、实现一处、两个调用方（`project` 的 `Cascade`）。建议照本条。
5. **端口只带 P1a 读的东西**（裁定：照本条）。设计 3.3 的 `ProjectGate.Lock` 回答 `archived`、`identifier`、`default_assignee_id`、`guest_view_all_features`、`archive_in`，另有 `LockMany`、按工作区列出项目、`DeletedAt`；`StateCatalog` 回答名称、组、`default`、是否分诊，另有按 id 取一个。P1a 的 `Lock` 回答工作区、`archived`、`default_assignee_id`，`StateCatalog.States` 回答 `id`、组、`default`（分诊由组认出，3.17），`ProjectGate` 只有 `Find`、`Lock`。其余的随它们的读者加入：`identifier`（按编号取，P2）、`guest_view_all_features`（访客规则，P2）、`archive_in`（自动归档，P6）、`LockMany`（关联，P3）、`DeletedAt`（P3、P9）、状态的名称（列表的分组，P5）。设计 3.3 的列表是 M4 全程的端口，P1a 取其中用到的部分。建议照本条。
6. **`PublicProfiles` 在 P1a**（裁定：照本条）。设计 3.3 说这个端口"由 `identity` 的读接成（P5）"，用于按负责人名排序。动态的第 2 行写负责人的显示名（3.13 的 `new_value`），创建就要它，所以 P1a 接上（`issue/app` 声明，`bootstrap/ports.go` 的 `issueProfiles` 转换 `identity.Provide` 已有的 `PublicProfiles`），只是 Phase 提前，端口不变。P5 照用。
7. **契约只有 `IssueDetail`**（裁定：照本条）。设计 5.2 有 `Issue`（列表的行）和 `IssueDetail`，P1–P3 两者同形。P1a 只有 `createIssue`，它回答 `IssueDetail`；`Issue` 随它的第一个读者（P5 的列表）加入，不加没有使用者的结构。P1b 的 `getIssue`、`updateIssue` 也回答 `IssueDetail`。
8. **`createIssue` 不声明 `server_busy`**。写闸门和 `lock_timeout` 在 P1b；P1a 的 `createIssue` 答不出 503，`apitest.Main` 的两个方向要求声明的码都被返回过，所以 P1b 随闸门在每个工作项的写上声明它。
9. **"列表从不为 null"由 HTTP 一处负责**。`GetIssue` 对没有负责人、标签的工作项回答 NULL（领域的 `nil`），`issues.go` 的 `orEmpty` 把它们答成 `[]`；存储不另 `COALESCE`。M3 的处理函数也是这样做列表的。变异 T9.2 在 HTTP 和矩阵两层失败。
10. **分诊状态的格子经仓储准备**（设计对抗性评审 M17）。`createProject` 建出的 Triage 不在状态列表里；矩阵的种子经存储种下它（M3 的矩阵已有），能建工作项的格子答 422 `state_id`（Task 12 的"the triage state"一行）。
11. **端到端的过渡（裁定 7）**。Task 2 建表之后，`expectProjectDeleted` 从目录读出 `issues`（Task 3 起五张表），要求每张表有随项目删除的行；故事要到 Task 14（`createIssue` 的接口在 Task 9，夹具和 M3 的故事在 Task 14）才写得出工作项。Task 2 到 13 在 `expectProjectDeleted` 里把这五张表写成"没有随项目删除的行、也没有没删的行"（`withoutRows`），注释写明为什么、到哪里；Task 14 删去，同时 `projectTables` 核对目录的表就是有 `project_id` 列的每张表。Task 1、2、3、11、14 跑 `make e2e`（Task 1 改了删除工作区，Task 11 改了删除状态、标签，其余的 Task 不碰故事读到的东西）。Task 14 的变异 T14.3（留着这个排除）在 P4、W3 失败，T14.1（一张表的 `project_id` 没有外键）在自查上失败。
12. **夹具的落点和签名**（裁定：照本条，设计第 2 节的写法由本条取代）。设计第 2 节原写 `furnishIssueRows(api, token, projectId)`，在 `e2e/fixtures/api.ts`（P1a 的最终评审之后照本条改了）。P1a 的是 `furnishIssueRows(api, token, projectId, assigneeId, labelId)`，在新的 `e2e/fixtures/furnish.ts`：负责人和标签是故事自己的账户和标签（W3 里是成员和他建的 Bug），删除要把它们写成删除者的，故事读得出；一个夹具自己去找"项目里的某个成员、某个标签"会让故事看不出它造了什么。`api.ts` 有 423 行（基点），`createIssue` 的夹具放在 `issues.ts`；`furnishIssueRows` 和从 `api.ts` 移来的 `furnishWorkspace` 放在 `furnish.ts`（它要 `api.ts` 和 `issues.ts`，放在 `api.ts` 里会让两者互相导入）。`projectTables`、`workspaceTables` 是同一个查询（目录里指向父表的外键表，核对有那一列的每张表），放在新的 `assert/catalog.ts` 的 `tablesUnder` 一处，不写两份。W3 在基点有 399 行，P1a 再加 16 行；照 M3 的写法拆成接口版本（`w3-workspace-settings.spec.ts`）和页面版本（`w3-workspace-settings-page.spec.ts`，两个页面的测试原样移过去）。
13. **交错 3 在 P1a 是创建的一支**。设计的交错 3 是"修改工作项与删除工作区"。P1a 没有修改，用创建做持项目 N 的写，验证同一件事（删除先拿到工作区 N 而工作项的写已持项目 N 时，删除等它提交，时刻晚于它的）；修改的一支在 P1b（交错 3u）。
14. **交错 1 看不出缺少显式的锁**。两次创建都要 `NextIssueSequence` 的 `UPDATE`，它本身行锁项目，第二个在这里等第一个提交。所以把项目的锁换成普通的读（T12.1）交错 1 照样通过：显式的锁保证的是编号之前的判定和检查也在锁下（交错 4、5、8 看得出，T12.1 在它们上失败；取锁的形状测试让写停在编号之前，读出它的锁的强度，T12.1 和 `FOR SHARE` 的 T6.1 也在那里失败）。交错 1 留着它的断言（编号、`sort_order`、时刻）。
15. **I1 不断言"拒绝不耗号"**。拒绝都发生在取号之前（Task 8 的次序），取号又在事务里，拒绝时回滚：两者中去掉任何一个都看不出来（T8.8 把取号移到检查之前，事务回滚，故事照样通过）。没有单个的缺陷能让这条断言失败，留着它就是一条没有变异的检查，所以删去；次序由单元测试钉住，按性质只在单元一层（附录 A.3）。
16. **承接的条目**：

    | 来处 | 条目 | 落点 |
    |---|---|---|
    | M3 收尾交接第 2 节 | 先锁后判定的副作用：凡判定拒绝的写在锁之前答，不取锁 | 2.8；`TestEachIssueWriteLocksItsProjectFirst`（2.10） |
    | 同上 | 方案 E 的选择（B：不取工作区 S）；新的批量写照约定五 | 2.3、2.4（键集按 `id`、`FOR NO KEY UPDATE`）、第 5 节 |
    | 同上 | 新的成员关系增长 | 没有（指派、关注不让谁成为成员，3.5） |
    | M3 收尾交接第 4 节 | 删除状态、标签时工作项的去向；默认负责人的契约；`identifier`、`default_assignee_id`、`State.default` 的说明；`workspaceTables` 从目录读出并自查；`furnishIssueRows` | 2.11、2.9、2.13 |
    | M1/P2、M1/P3 交接 | 工作项的接口没有 `estimate_point`、`description_binary`、`is_epic`、`type_id`，没有 `/issue-dates/`、批量的接口 | 2.9 |
    | M3 的常设规则 | 时钟在最后一把锁之后；锁下重读确认父行；`apitest.Main` 两个方向；端口的错误原样返回；角色按集合；目录驱动的检查不加豁免；一条共用的取锁路径 | 各 Task |

## 4. 验收标准（完成线，M4 设计 12 节 P1a）

- [ ] I1 的接口版本（工作项的行、编号、默认状态和默认负责人、负责人、标签、关注、创建和负责人的动态、各种拒绝）通过；M3 的 P4、W2、W3 带着工作项一侧的行通过，没有留下端到端的过渡（`make e2e` 121 个：此前的 120 个，加 I1）。
- [ ] 交错 1、2、3、8、21、22 和交错 4–7 的创建一支 `-count=5 -race` 通过，没有 40P01（附录 A.4）。
- [ ] 看不到项目、看得到而不能写的写不取锁（`TestEachIssueWriteLocksItsProjectFirst`）；每个工作项的写在 `issueWrites` 里有一行（完整性核对，T10.8）。
- [ ] 本 Phase 的矩阵格子通过（`createIssue` 七行）。
- [ ] 删除在用的状态 409 `project.state_in_use`；删除标签从工作项上摘下它和它的子标签。
- [ ] 迁移 `00015`–`00019` 升、降、再升通过；`issue` 的 `apitest.Main` 两个方向通过。
- [ ] `make gen-check`、`make lint-go`、`make test`、`make lint-web`、`make knip`、`make test-web`、`make e2e` 通过。

## 5. 不在 P1a 范围内

- 取、改工作项和写闸门（`getIssue`、`updateIssue`、`platform/gate`、`WriteGate`、`lock_timeout` → 503）：P1b。描述、起止日期、访客规则、删除工作项：P2–P4。删除项目、工作区之后的任务：P9。

**交给 P1b**（P1b 的 spec 接过去）：

| 条目 | P1a 留下的 |
|---|---|
| 取锁的一步 | `issue/app/lock.go` 的 `Locks`、`write`、`precheck`、`lockAndDecide`：P1b 要锁下读到的东西（工作项）时让 `lockAndDecide` 带上它；P1b 的按资源寻址的写在这条路径上加"读工作项（不加锁）得到项目、锁下重读确认仍在这个项目"，`write` 带上它的读；不另写一份路径。预检之后、事务之前插入写闸门（3.5） |
| `server_busy` | 写闸门和 `lock_timeout` 的 503 在 P1b，随它在每个工作项的写的 `x-problem-codes` 上声明（第 3 节第 8 条） |
| 端口 | `ProjectGate.Find`、`Lock` 照用；`IssueCreator.GetIssue` 已回答带负责人、标签的工作项（`getIssue` 用它）；修改要的存储方法、差集写法（4.3）在 P1b 加 |
| 领域 | `NewIssue`、`CheckNewIssue`、`Assignees`、`UnknownLabel`、`StateNotAllowed`、`SortOrderAfter`、`CompletedAt`、`CreationActivity`；P1b 加修改的规则和动态的第 3、6、7、10–13 行 |
| 契约 | `IssueDetail`（`getIssue`、`updateIssue` 的回答）；`IssueUpdate` 在 P1b |
| 组合 | `issueWrites`（每个工作项的写一行，完整性核对：P1b 的写不加行时 `TestEachIssueWriteLocksItsProjectFirst` 失败；每行的 `reads` 是写在锁下最先读、在锁之前不读的表：`updateIssue` 在锁之前读 `issues`，它的 `reads` 不能是 `issues`）；`writesOnAProject` 按标签 `issue` 去掉工作项的写；P8 的草稿的写和发布草稿先取工作区 S（设计 3.5 代价二），不是这个形状：P8 把它们从 `writesOnIssues` 分出（标签或清单），另写它们的形状测试；矩阵 `permission_matrix_issues_test.go` |
| 交错 | `heldTx`、`exchange`、`racingWrite`、`serialize`、`inBothOrders`、`createsIssue`、`deletedWith`：交错 3u–7u 照它们写（修改代替创建），交错 20 另经真实的连接池和闸门 |
| 交错的门（M3 的旧写法） | `interleaving_writes_test.go` 的 `TestTwoWritesOnAProjectSerialize`（:199-211）仍手写门的几步（第一个写停在门上，第二个写在 `projects` 上等，开门）；P1b 的交错 3u–7u 改到这个文件，在那里把这几步收进 `interleaving_test.go` 的 `serializeOn` |
| 规模 | 同步的过渡写法（`deleteProjectRows`，每批 1,000 行）：P1b 在 2 万个工作项（每个 20 条动态）的项目和五个这样的项目的工作区上量请求的耗时，记进 P1b 的 spec（裁定 4） |
| 端到端 | `issues.ts`、`assert/issue.ts`：I2 的接口版本照 `expectIssueCreated` 的写法加 `expectIssueUpdated` |
| 关键词守卫 | `phase` 照旧是 `M4/P1`（裁定 5） |

**留给后面的 Phase**：

| Phase | 条目 |
|---|---|
| P2 | 访客规则：判定的 `Grant` 重新由 `decide`、`precheck`、`lockAndDecide` 带出（附录 A.2 删去的 `held`；`OwnOnly` 的拒绝也在预检里答，设计 3.4、3.5）；`OwnOnly` 由 `access` 的 `ProjectFacts` 算出，`ProjectGate.Lock` 要不要 `guest_view_all_features` 由 P2 定；起止日期进接口；按编号取（`identifier`）；`project.state_in_use` 的说明和文案要不要加补救由 P2 的文案定：P1b 的修改和 P2 的恢复都有了才做得到（例如"先把它们移到别的状态，已归档的先恢复"） |
| P3 | `LockMany`（约定七）、`DeletedAt`；关联进删除的语句；加关联与删除工作区的交错（两个项目同属一个工作区，两种顺序）：`LockWorkspaceProjects` 的 `id` 升序从这时起承重（P1a 只由存储测试钉住） |
| P5 | `Issue`（列表的行）；状态的名称进 `StateCatalog`；**P5接收结果（2026-10-26）**：已有Issue/Detail已在P4分离，状态名已在P1b、PublicProfiles已在P1a交付；P5 T4/T9/T10实际消费而非重复交付，T8及T15 I6沿原锁/审计路径，T12精度/快照与T14回归通过。P6事件、P8草稿、P9删除和页面范围仍开放，实际证据见 [P5 review §2](../reviews/P5-issue-list-review.md#2-验收标准逐条核对) |
| P6 | `archive_in` |
| P8 | `DeleteWorkspaceIssues` 加工作区的 id（没有项目的草稿，第 3 节第 4 条） |
| P9 | `DeleteProjectIssues`、`DeleteWorkspaceIssues` 换成事件和任务，删去这两个方法 |
| P14 | I1 的页面版本 |
| 收尾 | 3.23 中 P1a 的行由 Task 15 写好，收尾逐行核对 |

## 6. 风险

| 风险 | 应对 |
|---|---|
| 预检与锁下的判定不一致 | 锁下的判定是最终的（`lockAndDecide`）；交错 4、5 的创建一支在真实数据库上展示等锁期间被移出、降级的人得到 404、403；T8.2（锁下不再判定）在单元和组合一层失败 |
| 删除工作区的新顺序与 M3 的写成环 | 新的两步锁都在工作区 N 之下，邀请先于项目（全局顺序）；M3 的每个写在工作区 S 或 N 之下才碰项目；M3 的交错照旧通过，交错 3 两种顺序；`TestAWorkspacesDeletionWaitsForAWriteHoldingOneOfItsProjects` |
| 新表进入两个连带 | 按目录的删除测试不开豁免，五张表各种下一行；端到端从目录读出、自查 |
| 同步的连带在规模目标上超过期限 | 已知（决策点 7）：P9 换成任务；P1b 量出数字 |
| 持续集成（ubuntu）上的运行 | 原型和复现都在 macOS 上；合并之后看持续集成 |

## 7. 已知的限制、交接和关闭条件

- **授权与时刻的先后可以不一致**（B 的代价一，设计 3.5）：只改工作区角色的写不锁项目，降为访客、结束成员关系的连带在锁项目之前读时刻。写进总体设计 4.2（Task 15）。
- **拒绝的回答在预检时定下**：预检拒绝的请求在那一刻排定先后（可线性化），锁下不再看（3.5）。

| 交接 | P1a 处理的条目 | 留下的条目 |
|---|---|---|
| M3 收尾交接第 2 节 | 预检（不取锁）；方案 E 的选择写进总体设计 4.2、M3 设计 17.4 | 新的批量写逐个照约定五（以后各 Phase）；物理删除账户（P9） |
| M3 收尾交接第 4 节 | 删除状态、标签的去向；默认负责人的契约；`identifier`、`default_assignee_id`、`State.default` 的说明；`workspaceTables`、`furnishIssueRows` | `guest_view_all_features`（P2）、`archive_in`（P6）的说明 |
| M1/P2 交接 | 接口没有 `estimate_point`、`description_binary`、`/issue-dates/` | `skip_activity`（P4）；"撤销再重做"（P14） |
| M1/P3 交接 | 接口没有 `is_epic`、`type_id`、批量的接口 | 动态、评论、筛选、收集箱的各项 |

各交接的"处理结果（M4/P1a）"由 Task 15 写下，`status` 照旧 `open`，在 M4 收尾随 review 关闭（M3 的做法）。

**M4 设计 13.1 的关闭条件**：落在 P1a 的四项（M1-P2、M1-P3 的接口字段，M3 交接第 2 节的预检，第 4 节）都在上表。没有放不下的条件。

## 附录 A：原型验证记录（2026-10-17）

### A.1 做法和 Phase 的大小

原型在 `$M4TMP/p1proto`：`d693f539`（设计的拆分提交）的一份 `git archive`（`p1tools/base-d693f539.tar`，快照 `p1snap/T0`）之上逐 Task 做，每个 Task 做完存一份源文件的快照（`p1snap/T1`…`T15`）；`node_modules` 由 `pnpm install --frozen-lockfile` 装好（不用符号链接），`bin/golangci-lint` 2.13.2；Go 1.27.1（`toolchain`）、Node 24、pnpm 11.10.0、Docker。plan 的代码块由脚本从相邻两份快照的差异生成（`p1tools/mkblocks.py`），每个块从基点起按顺序核对过（`problems: 0`）；生成的文件不进块，按 SHA-256 和行数钉住（`gensha.py`）；`compose_plan.py` 组装 plan，核对每个块放了一次、每个文件在文件表里（`filetablecheck.py`）、没有遗留的标记，并从变异的结果写出每个 Task 的变异表（被发现的层、"（Task n 起）"）。之后的修改（A.2）都写回拥有那个文件的 Task 的快照，从那个 Task 起每一份快照都改，`mkblocks.py` 重新生成全部块。

**切分**（裁定 3 允许在上限之内重新切分）：设计草稿的任务 1 量到 1,815 行代码块，拆成 Task 2（`issues` 一张表和骨架）和 Task 3（其余四张表）；取锁的一步不能单独成为一个 Task（没有使用者的未导出函数过不了 lint，M3 也是经用例测它），与 `createIssue` 的用例同在 Task 8；删除工作区的新顺序提到 Task 1（第 3 节第 1、2 条）。没有用到退路。

**Phase 的大小**（裁定 3：约 16 个任务、每个任务约 1,500 行以内）：15 个 Task；plan 12,388 行（`wc -l`）；每个 Task 都在约 1,500 行以内，最大的是 Task 2（1,435 行），其次 Task 11（1,312 行）、Task 9（1,307 行）。各 Task：1 1,115、2 1,435、3 803、4 554、5 622、6 526、7 623、8 1,246、9 1,307、10 503、11 1,312、12 575、13 135、14 1,221、15 235（`compose_plan.py` 量的，从 Task 的标题到下一个标题）。手写的文件 141 个（新建 59 个、修改 82 个），生成物 14 个（原型上的数；分支的终态见 2.1）。没有超出，没有停下。

### A.2 清扫之后的修改和加强

**清扫之后的修改**（变异按 Task 跑完之后，复审时发现，改在根上，受影响的变异重跑）：

- `issue/app/lock.go`：判定的 `Grant` 原来放在 `held` 里带出 `lockAndDecide`，P1a 没有一处读它（P1 只有管理员和成员，访客规则在 P2）。删去 `held`，`lockAndDecide` 回答锁住的项目，`decide` 只回答错误（Task 8；第 2.8 节）。锚在旧形状上的 T8.2、T8.3、T8.8、T8.9、T10.3、T10.4 改锚，Task 8、10 的全部变异在各自的树、`from` 的树和最终的树上重跑。
- `createIssue` 的三个不可能的情况原来没有测试：没有默认状态的项目、写入之后读不回来的工作项（显示名缺一个的那个有）。`TestCreateIssueStopsAtAFailingPort` 加两段，假实现加 `lost`；变异 T8.15–T8.17。
- 失败的传递：原来只有存储一层的三个失败变异（T1.10、T1.11、T6.8）。按清扫 2、19 补上：用例吞掉端口的失败（T1.13、T2.9、T2.10、T3.7、T8.14、T11.19、T11.20），存储把读失败答成"没有"（T4.10、T7.10、T11.21）。
- `checkkills.py`（P1a 新加的每个 Go 测试至少被一个变异弄失败）找出三个没有变异的检查：`TestLockWorkspaceInvitationsLocksInIDOrder`（加 T1.14、T1.15，并补 `LockWorkspaceProjects` 的范围 T1.16）、`TestDependentRowProblems`（加 T5.10）、`TestCreateIssueHoldsTheBodyToItsStructure`（加 T9.11）。
- 三个变异的旧文字在后面的 Task 里变了（T1.2、T1.3 的项目锁在 Task 2 起回答锁住的项目，T2.5 的授权在 Task 3 起是一行五张表）：`mut.py` 的一处编辑可以给出两个 Task 的两段文字，树上恰好有其中一段（`mutcheck.py` 核对每个变异在它的 Task、`from` 的 Task 和最终的树上都放得下）。
- 注释：七处编辑留下的过长的注释行重新折行（`longcomments.py`，只改注释）。

**清扫之后的加强**（变异按层跑完之后，只在单元一层被发现、而组合一层或故事本可以展示的，改测试或补上更高一层的检查，不改产品代码）：

- I1（Task 14）加三步：bob 在 Cancelled 建第 4 个工作项，不算完成（T5.7 原来只在单元一层）；同一个账户、同一个标签各给两次，422，两个问题一次报出（T5.3）；Web 没有的状态、Web 没有的标签和指派访客，422，三个问题一次报出（T8.11 原来只在单元一层；T8.7、T5.10 也在这里失败）。原来的"指派访客 422"并进第二个。标题随之。
- 已有的更高一层的检查补进变异的层：T5.5（默认负责人不论角色都被指派）在交错 22 和 I1（ann 离开之后）失败；T5.6（负责人按请求的次序）在 I1（动态的次序）；T5.8（第一个排序值）在矩阵和 I1；T5.9（第 2 行的旧值）在 I1；T8.7（别的项目的标签）在交错 7 和 I1；T8.9（看不到的项目答 403）在矩阵（外人的格子）和取锁的形状测试（olga）。这些都是从后面的 Task 起才有的检查，标"（Task n 起）"。
- 第二轮（同样只改测试）：I1 的第一个 422 加大小写不对的优先级 `High`（契约的值是小写的；体的形状不核对枚举的值，接口到得了领域的检查，T5.2 原来只在单元一层）；矩阵加"another project's label"一行（Web 以外的项目的标签，能建工作项的格子 422 `label_ids[0]`，T6.6、T8.7 在它上失败：清扫 24 的"别的项目的标签"一半原来在组合一层没有反例）；只在存储一层跑过、组合一层或故事看得到的变异补上那些层：T4.3、T4.4（按目录的删除测试）、T6.4（交错 1、I1）、T7.4、T7.9（盖戳的测试，T7.9 也在 I1）。I1 改了，所以每个有 I1 一层的变异在 Task 14 的快照和最终的树上重跑那一层（`rerun6.sh`）。
- 取锁的形状测试（Task 10）原来让写停在它写的第一张表（`issues`）上，那时编号的 `UPDATE` 已经以 `FOR NO KEY UPDATE` 的强度锁了项目，`lockOn` 读到的 N 不一定来自它的锁（T6.1 的 `FOR SHARE` 照样读成 N）。改为让写停在它锁下最先读、预检没有读的表（`reads`：`states`，`ACCESS EXCLUSIVE`，普通的读也等它）上，在它自己的任何写之前：读出的是锁本身的强度，T6.1、T12.1（普通的读）在组合一层失败。有这一层的变异（Task 10 的、T8.1、T8.9、T6.1、T12.1）在各自的树、`from` 的树和最终的树上重跑（`rerun7.sh`）。

### A.3 只在单元、结构、存储一层被发现的变异

**只在单元一层被发现的（按性质，20 个；修复轮另加 6 个，列在本段末）**：

- 端口的失败被吞掉（清扫 2）7 个：T1.13、T2.9、T2.10、T3.7、T8.14、T11.19、T11.20。真实的存储在这些步骤不失败；而一条失败的语句让 Postgres 中止事务，之后的语句和提交都失败，用例吞掉它也得到同一个失败的请求、同样全部回滚。组合一层的失败测试看得到这一点：`TestAFailedProjectsStepRollsTheDeletionBack` 在请求期间把 `issue_activities` 改名，T2.10 在它上照样通过（请求 500，全部回滚）。吞掉与否只在假实现上看得出；存储方法自己的失败由清扫 19 在存储一层核对（M3 P7b 的 `s2-*` 同类）。
- 端口的回答对不上所问（清扫 22）3 个：T8.15（显示名的端口少回答一个负责人）、T8.16（项目没有默认状态）、T8.17（写入之后读不回来）。真实的端口在同一个事务里回答：负责人刚由 `ProjectMembers` 确认是有效成员，`identity` 的显示名对每个有效的账户都有；每个项目恰有一个默认状态（M3：建项目时建它，部分唯一索引，默认状态删不掉）；`GetIssue` 读的是同一个事务刚写的行。组合出的 app 到不了这些情况（M3 P7b 的 `s22-*` 同类）。
- 次序的约定（清扫 9）3 个。T1.3（删除工作区先锁项目、再锁邀请）：唯一不经工作区行就锁邀请的写是停用（M3 3.9：他所在的工作区 N → 寄给他的邀请 → 他的项目），它之后要锁的项目都在它已先锁 N 的工作区里；所以持着这个工作区的一个邀请、再等它的项目的写，一定先等过删除持有的工作区 N，反过来的次序在 P1a 构不成环。T3.2（连带先删动态、再删工作项）、T8.12（创建先写动态、再写关注）：五张表的每个写者都先持项目 N，同一个项目的行在它之下不会有两个写者，表的次序等不起来。全局顺序是给以后的写（P9 的任务）的约定，由单元的调用次序钉住。
- 键集（3.6）2 个。T3.3（一张表的分批在第一个不满的批次就停）：持着项目 N，一个不满的批次之后不会再有行，所以与原来的写法等价；等到空的批次是为 P9 的任务（它不在请求的锁下）。T3.4（下一批从上一批最小的 `id` 之后）：语句自己跳过已删除的行，下一批照样只取没删的，只是扫得多：一个性能的性质。两者在真实环境上都看不到；P1b 的规模测量（2 万个工作项）跨很多批。
- T8.8（在检查依赖的行之前取编号）：拒绝都在事务里，回滚连编号一起回滚（第 3 节第 15 条）。
- 契约和文案 3 个。T9.11（`IssueCreate` 收下多余的字段）：体的形状由生成的 `bodyshape` 在用例之前检查，HTTP 的测试驱动的就是生成的路由和真实的处理函数，这条路上没有替身；T11.16（`deleteState` 的契约不声明 `project.state_in_use`）：只有 `apitest.Main` 的覆盖核对看得出（M3 P7b 的 `s18-update-extra-default` 同类）；T11.17（`project.state_in_use` 没有文案）：`error-messages.test.ts` 读的是真实的文案表，页面在删除状态时显示它，工作项的页面在 P14。
- T9.6（`writesOnAProject` 去掉每个模块带标签的写）：改的是测试的辅助函数，由它自己的形状测试 `TestWritesOnAProjectAreEachShape` 发现。
- 修复轮加的 6 个（定义在 `$M4TMP/p1sdd/` 的探针文件里，不在 `mutants_p1a.py` 的 136 个之中），都在单元一层被发现。T3F.1、T3F.2（下一批从上一批回答的第一个、最后一个 `id` 之后开始，不是最大的）：同 T3.4，语句自己跳过已删除的行，真实环境上只是扫得多；单元的假实现轮流按两种次序回答，两者都被发现。T8P.4（关注不先去重）：存储的 `ON CONFLICT DO NOTHING` 跳过第二行（`TestInsertSubscribersSubscribesEachAccountOnce`），组合出的 app 里等价，同 T7.3；由 `TestCreateIssueDefaults` 里调用者也是负责人的一行发现。T8P.5（关注按调用者、负责人的次序写，不按 `id`）：同 T8.12，同一个项目的行在项目 N 之下没有第二个写者。T9Q.1、T9Q.2（回答把创建者、创建的时刻当作最后修改的）：P1a 只有创建，经接口造出的工作项的这两列与创建的相同，只有 HTTP 的测试（`webSeven` 一分钟之后由 bob 改过）分得出；修改在 P1b。

**只在结构、存储一层被发现的（按性质或按设计，28 个；都在真实的数据库上）**：

- 存储方法自己的失败（清扫 19）6 个：T1.10、T1.11、T4.10、T6.8、T7.10、T11.21（取消的 context；组合出的 app 里真实的存储不失败）。
- 迁移的约束 5 个：T2.8、T4.6–T4.9（结构一层的 `TestConstraintAndIndexNames`、`TestIssueChecksRejectCounterexamples`、`TestIssueUniqueKeys`）。领域先拒绝这些值，或 P1a 造不出它们（编号在已删除的工作项中也唯一：P1a 的工作项只随项目删除；日期在 P2 才进接口；同一个标签再加到工作项上在 P1b 的修改），CHECK 和唯一键是纵深防御。
- 删除工作区锁项目的强度 2 个：T1.6（取 `FOR SHARE`）：`FOR SHARE` 与工作项的写的 `FOR NO KEY UPDATE` 照样互斥，只放过别的 `FOR SHARE`，而取项目 `FOR SHARE` 的是项目管理类的写，它们先共享工作区，被删除持有的工作区 N 挡在外面。T1.7（取 `FOR UPDATE`）：多挡的是外键的 `FOR KEY SHARE`，往这个工作区的项目下插行的写或者先持项目 N（工作项的写，删除等它），或者先共享工作区（项目管理类的写，被挡在外面）。两者由存储的 `TestLockWorkspaceProjects` 用两个连接核对（外键的 `FOR KEY SHARE` 不等它，`FOR NO KEY UPDATE` 等它）。
- 约定五（多行的锁按 `id`）3 个：T1.4、T1.14、T4.1。工作区 N 之下没有别的写锁这个工作区的几个项目或几个邀请，分批的语句在项目 N 之下；按 `id` 的次序是给以后的写的约定，由存储测试钉住（清扫 48）：T1.4、T1.14 由两个连接的存储测试，T4.1 由核对每一批按 `id` 升序回答的存储测试（`TestTheBatchStatementsDeleteAProjectsRowsInIDOrder`）。
- 已删除的行 8 个，P1a 造不出能区分它们的状态，或之后的语句自己跳过已删除的行：T1.5、T1.15（锁也锁已删除的项目、邀请：之后的删除语句自己跳过它们）；T6.3（`issue` 的锁回答已删除的项目：锁下的判定读项目的事实，看不到已删除的项目，同样答 404）；T4.2（负责人的批次再盖已删除的行）、T7.5（读到已删除的负责人行）：去掉负责人的写在 P1b；T11.13（`DetachLabels` 再盖已删除的行）：P1a 里工作项上的标签只由这次摘下或项目的连带删除，先删掉一行的修改在 P1b；T7.1（已删除的工作项的排序值也算）、T7.8（读到已删除的工作项）：P1a 的工作项只随它的项目删除，读的是同一个事务刚写的行。
- 等价的写法 2 个：T4.5（动态的批次不看下界）：语句自己跳过已删除的行，结果相同，只是扫得多；T7.6（标签按行的次序读）：创建按标签的 `id` 写行，行的次序就是标签的次序，修改时后加的标签在 P1b。
- 用例已经做了的 1 个：T7.3（关注的 `ON CONFLICT`）：创建先去重（`slices.Compact`）；已关注的人被指派在 P1b 的修改。
- 生成 1 个：T6.2（`issue` 的锁也锁工作区行：sqlc 拒绝，`project` 的 schema 没有 `workspaces`）。

**组合一层没有、端到端有反例的**：T14.3（留着过渡的排除）改的是故事的夹具（清扫 28），它守的性质由组合一层的按目录的删除测试核对。

### A.4 最终的原型、迁移、矩阵、交错和竞争

**最终的原型**（`gates.sh`，2026-10-17：七个门禁在最终的树上（`logs/final2-*.log`、`final2.summary`）；之后只改了取锁的形状测试（A.2），Go 的三个门禁重跑（`final3.summary`）；此后原型只被变异（每个在自己的副本里）和复现结束时复制进来的 spec、plan、设计改动，逐 Task 复现的终态与它逐个文件相同）：`make gen` 之后生成物没有差异；`make lint-go` 两段 `0 issues.`；`make test` 46 个 `ok`；`make lint-web`（关键词守卫 72 条规则、23 个例外、没有命中，54 个任务）；`make knip`；`make test-web`（17 个任务）；`make e2e` 121 个中 118 个通过，I1、P4、W2、W3、W11 在其中；失败的三个与 P1a 无关：S3 因原型不是仓库的检出、构建没有提交号而失败（M3/P4b 的 F4），A4 的两个 firefox 测试因为机器上没有 Firefox。

**迁移**：`TestMigrationsGoUpDownAndUpAgain`（19 个迁移，降到零、再升）通过（`make test` 的 `server/migrations`）；`00015`–`00019` 的 Down 各删它的表。

**矩阵**：`TestPermissionMatrix` 941 格全部通过（`createIssue` 的七行 75 格：六行各 12 格，已归档的一行 3 格；其余 866 格是 M3 的），8.7 秒（`matrixcount.sh`，`logs/matrix.log`）。

**交错和竞争**（`raceall.sh`，`logs/race-final.log`）：`bootstrap` 的 39 个交错、竞争和取锁形状的测试（M3 的 30 个、P1a 的 9 个）各 `-race -count=5`，195 次顶层、835 个子测试全部通过，183 秒，没有 40P01、没有死锁、没有数据竞争。P1a 的 9 个：交错 1、8、22；交错 2、3（`TestCreatingAWorkItemAndDeletingItsProjectOrWorkspaceSerialize`，删除 Web、删除 acme 各两种顺序）；交错 4、5 的创建一支；交错 6、7 的创建一支（两个测试）；交错 21（`TestEachIssueWriteLocksItsProjectFirst`）；删除工作区等持项目 N 的写（`TestAWorkspacesDeletionWaitsForAWriteHoldingOneOfItsProjects`）。M3 的 30 个在删除工作区的新顺序之下照旧通过，其中有收窄之后的 `TestEachWriteOnAProjectSharesItsWorkspaceFirst` 和 `TestWritesOnAProjectAreEachShape`。

### A.5 逐 Task 复现

**逐 Task 复现**（`replay-all.sh`、`replay.py`，日志在 `p1tools/replay-logs-5396a085`；预检之后的第二次见本节末）：`d693f539` 的一份新的副本（`base-d693f539.tar`），`pnpm install --frozen-lockfile`，`bin/golangci-lint` 取原型的（2.13.2；`make tools` 会下载它）；按 plan 的顺序应用 15 个 Task 的 408 个块，照原样运行每个 Task 的 `Run:` 命令，一共 1,554 秒，没有重跑。Task 1–13 之后 `make lint-go` 两段 `0 issues.`、`make test` 全部 `ok`（42 个到 46 个）；`make gen-go`（Task 1、2、3、6、7）和 `make gen`（Task 9、11）之后，生成物的每个 SHA-256 和行数与 plan 的表相同，之后的 gen-same 没有差异；`make lint-web`、`make knip`、`make test-web` 在 Task 1、2、3、9、11、14 通过；Task 12、13 的 `-count=5 -race` 没有 40P01、死锁和数据竞争；`make e2e` 在 Task 1、2、3、11 是 120 个中 117 个通过、在 Task 14 是 121 个中 118 个通过，失败的只有 S3（F4）和 A4 的两个 firefox 测试（机器上没有 Firefox）；Task 15 的 `make lint-web` 通过，`doclinks.py` 没有断的链接，`make test` 46 个 `ok`。最后 `treediff` 比较复现的树与原型的 3,343 个文件，没有差异。复现的副本不是 git 检出：`make gen-check` 由 gen-same 代替（`make gen` 前后生成物逐字节相同），`make lint-web` 的关键词守卫经 `kwreal.py` 列出文件。

复现之后 plan 只改了 Global Constraints 的一句：原来写"每个 Task 提交前"跑 `make lint-go` 和 `make test`，改为照各 Task 实际的门禁写（Task 14 只改 `e2e/`，Task 15 只改文档，两者不跑 `make lint-go`）。块和 `Run:` 都没有变：与复现用的 plan 逐行比较，只差这一行。

**预检之后的修订**（2026-10-17：预检 0 High、0 Medium、6 Low，控制者照原文全部接受）：L1 改 `delete_rows.sql` 的说明（Task 2、3：等锁期间被删除的行被跳过、取下一行，每一批读新的快照；原来的"短的一批不说明是最后一批"不是 Postgres 的行为），快照 T2–T15 和原型照改，Task 2、3 的树和最终的树重跑 `make gen-go`，`gen/delete_rows.sql.go` 重新钉住（Task 2 `3dc81983…`、Task 3 `f0baebcc…`，行数不变，以后的 Task 不钉它）；L2 给 Task 15 加 M3 设计 3.3 的一对（工作项一侧的写不锁工作区行，同一个保证经项目行成立；`（修改，6 处）`）；L3、L4、L6 是第 5 节的 P2、组合、P3 三行，L5 是第 3 节第 1 条。没有变异的锚点变（`mutcheck.py`：136 个变异在各自的树、`from` 的树和最终的树上都放得下；改的只是注释和文档），变异不重跑。修订之后整个复现在一份新的副本上重跑（`replay-logs`，第一次的留在 `replay-logs-5396a085`）：409 个块，1,694 秒，没有失败，每个生成物的 SHA-256 和行数与 plan 的表相同，gen-same 每次 0 个差异，`make e2e` 同上；`treediff` 比较 3,343 个文件，没有差异；最终的原型 gen-same 0 个差异。

### A.6 五十类清扫和变异表

**五十类清扫**（brief 的缺陷类别；每个变异在一份新的副本里（`p1mut/<id>`），Go 以外的文件（迁移、契约、故事）也在副本里改，`mut.py` 在变异写的每一层各跑一次：单元（各模块的 `app`、`domain`、`adapter/http`，`access/domain`，`bootstrap` 的转换测试）、结构（`server/migrations`）、存储（各模块的 `adapter/postgres`）、组合（`bootstrap`）、端到端（单独运行的故事 I1、P4、W3、W11）、守卫（关键词、`check:sync`）；每个变异在它的 Task 的快照上跑，后面的 Task 才有的检查在那个 Task 的快照上跑（"（Task n 起）"），全部在最终的树上再跑一次；"层"是它在最终的树上被发现的每一层；端到端的"发现"由 `e2ekills.py` 核对是断言失败，不是崩溃或超时）：

| 清扫 | 大小 | 结果 | 层 |
|---|---|---|---|
| 1 每个 SQL 谓词 | 新的语句的谓词和次序：两把锁（`LockWorkspaceProjects` 的范围、已删除、次序，`LockWorkspaceInvitations` 的已删除、次序：T1.4、T1.5、T1.14–T1.16）；分批的五条语句，每种谓词落在一条上（次序、已删除、项目、下界：T4.1、T4.2、T4.3、T4.5）；`project` 给 `issue` 的五条（已删除的项目、状态、标签，标签的项目，有效的成员关系：T6.3、T6.5、T6.6、T6.7、T13.1）；创建的读（排序值的已删除、状态，负责人行的已删除，标签的次序，已删除的工作项：T7.1、T7.2、T7.5、T7.6、T7.8）；`StateInUse` 的已归档、已删除，`DetachLabels` 的已删除（T11.2、T11.3、T11.13）；迁移的约束和外键 7 个（T2.8、T4.6–T4.9、T14.1、T14.2） | 29/29。组合一层或故事 11 个；其余 18 个只在结构、存储一层（A.3） | 结构；存储；组合；端到端 |
| 2 端口调用的错误 | 用例吞掉端口的失败：删除工作区锁邀请、删除项目和删除工作区的工作项一侧、分批的一批、创建写关注、删除状态问工作项、删除标签摘下标签（T1.13、T2.9、T2.10、T3.7、T8.14、T11.19、T11.20） | 7/7，都只在单元一层（按性质，A.3） | 单元 |
| 3 写不动的行 | 存储的写比较被写的行之外的每一行每一列（分批的语句的 `rowsAsText`，`DetachLabels`、创建的存储测试）；关注的批次取每个项目的行（T4.3）、排序值算每个状态的（T7.2） | 都被发现：T4.3 在存储一层和按目录的删除测试，T7.2 在存储一层和 I1 | 存储；组合；端到端 |
| 4 组合根的接线 | 服务端、命令行的组合不给 `project` 工作项的连带（T2.6、T2.7），`bootstrap` 不给 `issue` 显示名、不挂它的接口（T9.7、T9.10），`issue` 的连带不问存储（T11.14、T11.15），`project` 的 `Cascade` 不加锁（T1.8）；HTTP 的映射 3 个（T9.1–T9.3），端口的转换 2 个（T9.8、T9.9） | 12/12，都在组合一层（T9.8、T9.9 也在 `bootstrap` 的转换测试） | 单元；组合 |
| 5 安全性质在真实环境上 | `issue.create` 放进访客（T8.13）；没有预检、预检不判定（T8.1、T10.3）；锁下不再判定（T8.2）；看不到的项目答 403（T8.9） | 5/5，都在组合一层（矩阵、取锁的形状、交错 4、5），T8.13 也在 I1 | 单元；组合；端到端 |
| 6 每句文档 | 每句说明、契约描述、故事标题、差异清单的行对照代码 | 七处过长的注释重新折行；`held` 删去之后 `lockAndDecide` 的说明随之 | 审阅 |
| 7 反例里没有随机 | P1a 的测试 | 表都是切片；I1 的"Web 没有的状态、标签"用随机的 id，是不存在的 id，不是取样 | 审阅 |
| 8 决定所依赖的读在调用者的事务里 | 锁下的判定、显示名、删除状态问工作项、删除标签摘下标签各改为走池（T10.4、T10.5、T11.5、T11.10） | 4/4，都在组合一层（一个连接的池：`TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`） | 组合 |
| 9 锁顺序在真实环境上 | 删除工作区：邀请 N → 项目 N → 时刻（T1.1–T1.3）；工作项的写：项目 N 最先、工作区上没有锁（T10.2、T10.7）；连带和创建写表的次序（T3.2、T8.12） | T1.1、T1.2、T10.2、T10.7 在组合一层；T1.3、T3.2、T8.12 只在单元一层（按性质，A.3） | 单元；组合 |
| 10 承重的种子行有前提 | 删除测试的 `seedIssues`（五张表各一行，由不是删除者的账户）、工作区的两个项目；`newIssueWorld` 的人和角色；矩阵的 Triage | 按目录的删除测试先核对每张表有种下的行（M3 的 `DeletionViolations`）；`newIssueWorld` 经接口建出，回答由 `send` 核对；矩阵的种子由 `seededStates` 读回核对 | 审阅；组合 |
| 11 每条拒绝路径 | 预检拒绝的不取锁（404、403），锁下拒绝的（404、403、409），依赖行的 422，规则的 422 | 单元的两个拒绝表；矩阵的拒绝五行（已归档、另一个项目的状态和标签、指派访客、分诊）；取锁的形状测试的拒绝一段；I1 的四种拒绝 | 审阅 |
| 12 与结束、删除的竞争（组合） | 创建与删除项目、工作区（交错 2、3），与移出、降级（交错 4、5），与归档（交错 8），与删除状态、标签（交错 6、7），与移出默认负责人（交错 22） | 两种顺序都答对，等待由探测确认（`serialize`）；T12.4、T6.5、T13.1、T6.7、T8.10 在交错里被发现 | 组合 |
| 13 锁强度在组合一层看得到 | 写停在它锁下最先读、预检没有读的表（`states`）上（编号的 `UPDATE` 之前），`lockOn` 读出它的锁是项目的 N、工作区上没有锁；删除工作区等持项目 N 的写 | T10.1（`FOR UPDATE`）、T6.1（`FOR SHARE`）、T12.1（普通的读）在组合一层；T1.6（`FOR SHARE`）、T1.7 在存储一层（A.3） | 存储；组合 |
| 14 每个"由谁"可以失败 | 删除的种子由不是删除者的账户写；盖戳的测试由 alice 建、种子由别人；分批语句、`DetachLabels` 保留写者（T2.4、T4.4、T11.11） | 3/3，都在组合一层（按目录的删除测试、`TestDeletingALabelTakesItOffTheWorkItems`），T4.4、T11.11 也在存储一层 | 存储；组合 |
| 15 标题的每个说法都有展示 | I1、P4、W2、W3、W11 的标题逐句 | I1 的"一次报出"和"已取消不算完成"原来没有步骤，加上（A.2 的加强） | 审阅 |
| 16 单元假对象的回答顺序 | `fakeRows` 回答的 `id` 最大的在前（取最后一个做下一批的起点会重读）；假实现的标签目录按 `id` | T3.4（下一批从最小的 `id` 之后）在单元一层被发现 | 审阅；单元 |
| 17 判定之前的代价有界 | 预检：领域的检查（不读库）、一次不加锁的读、一次判定；创建在锁下：常数次读和写（关联表各一条语句） | 有界 | 审阅 |
| 18 契约描述只说代码做的 | `createIssue` 的描述、`IssueDetail` 的字段、`deleteState` 声明 `project.state_in_use`、`IssueCreate` 不收多余的字段（T9.4、T11.16、T9.11） | 3/3：T9.4 也在组合一层（矩阵读问题码）；T11.16、T9.11 只在单元一层（按性质，A.3） | 单元；组合 |
| 19 每个新的存储方法有失败测试 | `project` 的五个、两把锁，`issue` 的分批、创建、连带的方法；把读失败答成"没有"（T1.10、T1.11、T4.10、T6.8、T7.10、T11.21） | 6/6，都在存储一层（按性质，A.3） | 存储 |
| 20 相关谓词的集合形式、"别的"算进自己 | `ActiveRoles`、`IssueLabels` 用 `= ANY`，只回答请求中的；关注的部分唯一键：已关注的保留自己的行 | `TestInsertSubscribersSubscribesEachAccountOnce`（T7.3）在存储一层；创建先去重，已关注的人被指派在 P1b（A.3） | 存储 |
| 21 拒绝和失败钉住第一个 `*shared.Error` | 单元的拒绝表用 `errors.Is` 比较码，HTTP 的拒绝表按码；矩阵按格子的码和问题 | 审阅 | 审阅 |
| 22 端口的回答对得上所问 | `named` 按请求的 id 取显示名，缺一个是错误（T8.15）；标签目录的回答只当作集合；状态按 id 取；没有默认状态的项目是错误（T8.16）；`GetIssue` 读回的工作项不在时是错误（T8.17） | 3/3，只在单元一层（按性质，A.3） | 单元 |
| 23 组合的夹具跨第二个工作区、第二个项目 | 删除工作区的种子是两个项目；分批语句的存储测试有 ops 的行；矩阵另有已归档的项目和工作区；W3 的 Other | 删除工作区只删它的第一个项目的（T3.5）在组合一层；T4.3（关注的批次取每个项目的行）在存储一层和组合一层 | 存储；组合 |
| 24 每条规则的每一半在组合一层有反例 | 3.7 的规则：名称、优先级、给两次、访客、默认负责人、次序、完成、排序值、动态的第 2 行、分诊、别的项目的状态和标签、一次报出（T5.1–T5.10、T8.3、T8.5–T8.7、T8.10、T8.11） | 16/16，都在组合一层或故事（矩阵、交错、I1）；T5.2（大小写不对的优先级）、T5.3（给两次）由 I1 的第一个 422，"别的项目的标签"由矩阵的"another project's label"一行（A.2 的加强） | 单元；组合；端到端 |
| 25 判定和执行在同一把锁下 | 锁之前读时刻（T8.4、T1.1）；编号在检查之前（T8.8）；目录在锁下读（交错 6、7） | T8.4、T1.1 在组合一层；T8.8 按性质只在单元一层（第 3 节第 15 条） | 单元；组合 |
| 26 "不变"用不会碰巧得出的值 | 创建的存储测试的每一列都不是默认值（`createWebIssue`：7 号、high、75535、一小时前完成） | 存储一层逐列比较；T7.4（负责人的行没有写者）也在盖戳的测试 | 存储 |
| 27 故事里的"不动"在动作之前读、之后核对 | P4、W3 删除之前读 `expectProjectCreated`、之后 `expectProjectDeleted`；I1 每个创建之后 `expectIssueCreated` | 都有 | 审阅 |
| 28 只在端到端被发现的性质另有 Go 的测试 | 变异表里只在端到端失败的 | T14.3（留着过渡的排除）改的是故事的夹具，只有故事看得到；它守的性质由组合一层的按目录的删除测试核对 | 审阅 |
| 29 每个能等锁的等待都有期限 | 交错的 context 10 秒；探测 `WaitForLockWaitOn` 5 秒；种子和读回在 `pgtest.Soon` | 都有 | 审阅 |
| 30 每个输出、问题的细节和补救对每个收到它的调用者都成立 | `project.state_in_use` 的说明和两份文案；`createIssue` 的四个码 | 收到 409 的是判定通过的项目管理员；说明和文案只说状态里有工作项（已归档的也算）、所以不能删除，不给补救：P1a 没有移动工作项的写，已归档的工作项在恢复之前是只读的（设计第 2 节 I15：改字段 409 `issue.archived`），"先把它们移到别的状态"对谁都做不到；补救要等 P1b 的修改和 P2 的恢复都有了，交给 P2（第 5 节） | 审阅 |
| 31 检查和执行之间的 gate 在检查的事务之外 | 交错的门在第一个写的写之后、提交之前（`heldTx`） | T12.2（门在写之前扣住）被发现 | 组合 |
| 32 每个探针有反例 | 交错的探测等 `projects` 上的等待 | T12.3（等 `workspaces`）被发现 | 组合 |
| 33 goroutine 上不调用 `t.Fatal` | `racingWrite`、`run` | 闭包里只发请求、只送回回答 | 审阅 |
| 34 "没有替身"、"完成"的说法全包 grep | `issueWrites` 与契约中工作项一侧的写一一对应（T10.8）；`writesOnAProject` 按标签去掉工作项的写（T9.5、T9.6）；`projectTables`、`workspaceTables` 核对目录的表就是有那一列的每张表（T14.1、T14.2）；矩阵覆盖每个操作；`apitest.Main` 两个方向 | 都被发现：T10.8、T9.5 在组合一层，T9.6 在单元一层（按性质），T14.1、T14.2 在结构一层和故事 | 单元；组合；结构；端到端 |
| 35 "由 X 写"的种子是规则允许的 | `seedIssues` 由项目的管理员写；矩阵的种子由工作区管理员 | 规则到得了的状态 | 审阅 |
| 36 别的操作或两步到达同样的结果 | 关注：创建者和负责人各一次，部分唯一键挡住重复 | 没有绕过 | 审阅 |
| 37 归档项目、删除的工作区、停用的调用者 | 已归档的项目 409（T8.3）；已删除的项目和工作区在锁下 404（交错 2、3）；停用的调用者由认证在请求之前拒绝 | T8.3 在组合一层（矩阵的已归档小表、交错 8） | 组合 |
| 38 故事的每个拒绝之前先读 | I1 的四组拒绝 | 每组之前是上一个创建的 `expectIssueCreated`；之后不再读：拒绝都在写之前，或在事务里随回滚撤销，没有单个的缺陷能让一个拒绝留下行而仍答拒绝（同第 3 节第 15 条）；单元的拒绝表核对拒绝不调用写 | 审阅 |
| 39 测试里的排序与排序规则无关 | `tablesUnder` 用 `COLLATE "C"`；`expectIssueCreated` 的地址、标签名按 `toSorted()` | 与排序规则无关 | 审阅 |
| 40 每个交错的结果写明每个结束由谁 | 交错 4、5：移出、降级由 alice；交错 2、3：删除由 alice | 结果读出写者 | 审阅 |
| 41 每个写多行的语句与同一些行的其他写者 | 分批的五条（每批锁住它取的行，约定五）；`DetachLabels`（项目 N 之下，没有别的写者）；`DeleteLabel` 回答 `id` | 第 3 节、2.4、2.11 | 审阅 |
| 42 锁语句之后的写只写锁住的行 | 删除工作区锁住的邀请、项目：写的正是这些 `id`（`DeleteInvitations(ids)`、`DeleteWorkspaceIssues(projectIDs)`） | T1.9、T2.3 在组合一层 | 组合 |
| 43 每个"由 X"可以失败 | 同清扫 14 | 同上 | 存储；组合 |
| 44 "保留"、"还有"读状态 | 已删除的行保留它的删除（分批、`DetachLabels`、锁）；"在用"读未删除的 | T4.2、T11.13、T1.5、T1.15 只在存储一层（A.3），T11.2、T11.3 也在组合一层 | 存储；组合 |
| 45 没有不会失败的断言 | I1 原来的"拒绝不耗号"删去（第 3 节第 15 条） | 删去 | 审阅 |
| 46 测试一侧的等待都有期限 | 种子、读回、持有事务时的语句 | 都在 `pgtest.Soon` 或 context 的期限上 | 审阅 |
| 47 连续的探测在同一张表上要指名 | 交错每个子测试一次探测（`projects`） | 没有连续的同表探测 | 审阅 |
| 48 堆序或索引序的夹具有前提 | 分批语句的存储测试（web 的行在表里不按 `id`）；`TestLockWorkspaceProjectsLocksInIDOrder`、`TestLockWorkspaceInvitationsLocksInIDOrder` | 有前提；T1.4、T1.14、T4.1 被发现 | 存储 |
| 49 组合的夹具看得到过宽的写 | 删除只删它的项目和工作区的行（按目录的删除测试读别的工作区）；`DetachLabels` 只摘这些标签 | T1.16、T4.3 在存储和组合一层；T11.7 在组合一层 | 存储；组合 |
| 50 每句注释和文档对照它说的代码 | 每个 Task 末尾 | 见清扫 6、15 | 审阅 |

**变异表（按缺陷类别）**：变异的定义在 `$M4TMP/p1tools/mutants_p1a.py`（136 个，中文描述在 `mutants_zh_p1a.py`），结果在 `mut-results-pertask.json`（各自的 Task）、`mut-results-from.json`（"Task n 起"）、`mut-results.json`（最终的树），每次运行的日志在 `mut-logs-*`。136 个变异，136 个被发现。在最终的树上，组合一层的检查发现 72 个，端到端 23 个（I1 18 个、W3 5 个、P4 3 个、W11 1 个），存储 41 个，结构 8 个，单元 67 个，守卫 2 个（一个变异可以在几层被发现）；跑过而没有发现的只有组合一层的 7 次，都是 A.3 的按性质的（T1.3、T1.6、T2.10、T3.2、T3.3、T3.4、T3.7）。只在单元一层被发现的 20 个、只在结构和存储一层的 28 个，A.3 逐条写明。端到端和 vitest 的每次发现都是断言失败（`e2ekills.py`：最终的树 24 次、"Task n 起" 20 次、各自的 Task 4 次，没有可疑的）。P1a 新加的 45 个 Go 测试每个至少被一个变异弄失败（`checkkills.py`）。

| 缺陷类别 | 变异 | 结果 | 失败的检查 | 层 |
|---|---|---|---|---|
| 清扫 9：锁的顺序（删除工作区、工作项的写、连带和创建写表） | T1.2、T1.3、T3.2、T8.12、T10.2、T10.7 | 6/6 | `TestEachWriteReadsTheClockUnderItsLock`、`TestDeleteWorkspaceLocksThenDecidesThenCascades`、`TestDeleteWorkspaceRefusals`、`TestAWorkspacesDeletionWaitsForAWriteHoldingOneOfItsProjects`、`TestTheCascadeDeletesEachProjectsRowsInBatches`、`TestTheCascadeStopsAtAFailingBatch`、`TestCreateIssueWritesTheWorkItemAndItsRows`、等 3 个 | 单元；组合 |
| 清扫 13：锁的强度 | T1.6、T1.7、T10.1、T12.1 | 4/4 | `TestLockWorkspaceProjects`、`TestLockWorkspaceProjectsLocksInIDOrder`、`TestLockIssueProject`、`TestEachIssueWriteLocksItsProjectFirst`、`TestArchivingAProjectAndCreatingAWorkItemSerialize`、`TestACallerRemovedOrDemotedWhileHisCreationWaitsIsRefused` | 存储；组合 |
| 项目的锁和编号的语句（6.5） | T6.1、T6.2 | 2/2 | `TestLockIssueProject`、`TestEachIssueWriteLocksItsProjectFirst` | 存储；组合；生成 |
| 约定五：多行的锁按 `id` | T1.4、T1.14 | 2/2 | `TestLockWorkspaceProjects`、`TestLockWorkspaceProjectsLocksInIDOrder`、`TestLockWorkspaceInvitationsLocksInIDOrder`、`TestAWorkspaceDeletionAndTwoDeactivationsAllGoThrough` | 存储 |
| 键集的分批（3.6） | T3.3、T3.4、T4.1、T4.5 | 4/4 | `TestTheCascadeDeletesEachProjectsRowsInBatches`、`TestTheCascadeStopsAtAFailingBatch`、`TestTheBatchStatementsDeleteAProjectsRowsInIDOrder` | 单元；存储 |
| 清扫 5：预检 | T8.1、T10.3 | 2/2 | `TestCreateIssueWritesTheWorkItemAndItsRows`、`TestCreateIssueDefaults`、`TestCreateIssueRefusesBeforeAnyLock`、`TestCreateIssueRefusesUnderTheLock`、`TestCreateIssueStopsAtAFailingPort`、`TestEachIssueWriteLocksItsProjectFirst` | 单元；组合 |
| 清扫 5：锁下的判定 | T8.2、T8.9 | 2/2 | `TestCreateIssueWritesTheWorkItemAndItsRows`、`TestCreateIssueDefaults`、`TestCreateIssueRefusesUnderTheLock`、`TestACallerRemovedOrDemotedWhileHisCreationWaitsIsRefused`、`TestCreateIssueRefusesBeforeAnyLock`、`TestPermissionMatrix`、`TestEachIssueWriteLocksItsProjectFirst` | 单元；组合 |
| 清扫 24：创建的规则（3.7、3.12） | T5.1、T5.2、T5.3、T5.4、T5.5、T5.6、T5.7、T5.8、T6.4、T7.3、T8.3、T8.5、T8.6、T8.7、T8.8、T8.10、T8.11、T8.13 | 18/18 | `TestCheckNewIssueAccepts`、I1、`TestCheckNewIssueReportsEveryProblem`、`TestCanAssign`、`TestAssignees`、`TestRemovingTheDefaultAssigneeAndCreatingAWorkItemSerialize`、`TestCompletedAt`、等 12 个 | 单元；存储；组合；端到端 |
| 清扫 11：拒绝 | T5.10、T11.1、T11.4 | 3/3 | `TestDependentRowProblems`、`TestCreateIssueRefusesUnderTheLock`、`TestDeletingALabelAndCreatingAWorkItemWithItSerialize`、I1、`TestEachWriteReadsTheClockUnderItsLock`、`TestDeleteState`、`TestDeleteStateRefuses`、等 4 个 | 单元；组合；端到端 |
| 清扫 25、3：同一时刻 | T1.1、T2.4、T4.2、T4.4、T7.4、T8.4、T10.6、T11.9、T11.12、T12.4 | 10/10 | `TestEachWriteReadsTheClockUnderItsLock`、`TestAWorkspacesDeletionWaitsForAWriteHoldingOneOfItsProjects`、`TestCreatingAWorkItemAndDeletingItsProjectOrWorkspaceSerialize`、`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`、`TestTheBatchStatementsDeleteAProjectsRowsInIDOrder`、`TestCreatingAWorkItemWritesItsRows`、等 11 个 | 单元；存储；组合 |
| 清扫 14：由谁 | T11.11 | 1/1 | `TestDetachLabels`、`TestDeletingALabelTakesItOffTheWorkItems` | 存储；组合 |
| 连带：删除项目、工作区（3.6） | T1.9、T2.1、T2.2、T2.3、T3.1、T3.5、T4.3、T11.6、T11.7、T11.8 | 10/10 | `TestEachWriteReadsTheClockUnderItsLock`、`TestDeleteWorkspaceLocksThenDecidesThenCascades`、`TestDeleteWorkspaceRefusals`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`、`TestDeleteProject`、`TestDeleteProjectReturnsEachFailure`、`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`、等 12 个 | 单元；存储；组合；端到端 |
| 清扫 1、44：已删除的行（端口的读） | T6.3、T6.5、T7.1、T7.5、T7.8、T13.1 | 6/6 | `TestLockIssueProject`、`TestIssueCatalogs`、`TestDeletingAStateAndCreatingAWorkItemInItSerialize`、`TestGreatestSortOrder`、`TestCreatingAWorkItemWritesItsRows`、`TestAFailedIssueStatementIsAnError`、`TestDeletingALabelAndCreatingAWorkItemWithItSerialize` | 存储；组合 |
| 清扫 1、44：已删除的行（锁和连带） | T1.5、T1.15、T11.2、T11.3、T11.13 | 5/5 | `TestLockWorkspaceProjects`、`TestLockWorkspaceInvitationsLocksInIDOrder`、`TestStateInUse`、`TestDeletingAStateAWorkItemIsInIsRefused`、`TestDetachLabels` | 存储；组合 |
| 清扫 1、49：范围 | T1.16、T6.6、T7.2 | 3/3 | `TestLockWorkspaceProjects`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`、`TestIssueCatalogs`、`TestPermissionMatrix`、`TestGreatestSortOrder`、I1 | 存储；组合；端到端 |
| 清扫 1：有效的成员关系 | T6.7 | 1/1 | `TestIssueCatalogs`、`TestRemovingTheDefaultAssigneeAndCreatingAWorkItemSerialize` | 存储；组合 |
| 清扫 1：次序 | T7.6 | 1/1 | `TestCreatingAWorkItemWritesItsRows` | 存储 |
| 动态的第 1、2 行（3.13） | T5.9、T7.7、T7.9 | 3/3 | `TestCreationActivity`、I1、`TestCreatingAWorkItemWritesItsRows`、`TestCreatingAWorkItemStampsItsRows` | 单元；存储；组合；端到端 |
| 清扫 1：迁移的约束 | T2.8、T4.6、T4.7、T4.8、T4.9 | 5/5 | `TestConstraintAndIndexNames`、`TestIssueChecksRejectCounterexamples`、`TestIssueUniqueKeys` | 结构 |
| 清扫 34：目录的外键 | T14.1、T14.2 | 2/2 | `TestConstraintAndIndexNames`、P4、W3、W11 | 结构；端到端 |
| 运行时的授权 | T2.5、T3.6 | 2/2 | `TestTheGrantsFileCoversEveryRelationAndFunction` | 组合 |
| 清扫 8：事务的连接 | T10.4、T10.5、T11.5、T11.10 | 4/4 | `TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`、`TestEachWriteReadsTheClockUnderItsLock`、`TestDeleteState`、`TestDeleteStateRefuses`、`TestDeleteStateReturnsEachFailure`、`TestDeleteStateChecksTheReadAfterIt`、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`、等 2 个 | 单元；组合 |
| 清扫 4：接线 | T1.8、T2.6、T2.7、T9.7、T9.10、T11.14、T11.15 | 7/7 | `TestLockWorkspaceProjects`、`TestAWorkspacesDeletionWaitsForAWriteHoldingOneOfItsProjects`、`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`、`TestCommandsComposeNoServerAndNoJobs`、`TestPermissionMatrix`、`TestDeletingAStateAWorkItemIsInIsRefused`、等 1 个 | 单元；组合 |
| 清扫 4：端口的转换 | T9.8、T9.9 | 2/2 | `TestIssueProjectsConvertsProjectsAnswers`、`TestPermissionMatrix`、`TestIssueProfilesConvertsIdentitysAnswer`、`TestCreatingAWorkItemStampsItsRows` | 单元；组合 |
| 清扫 4：HTTP 的映射 | T9.1、T9.2、T9.3 | 3/3 | `TestCreateIssue`、`TestPermissionMatrix` | 单元；组合 |
| 清扫 18：契约 | T9.4、T9.11、T11.16 | 3/3 | `TestCreateIssueRefusals`、`TestPermissionMatrix`、`TestCreateIssueHoldsTheBodyToItsStructure`、`TestDeleteState` | 单元；组合 |
| 清扫 34：最先锁工作区的测试收窄 | T9.5、T9.6 | 2/2 | `TestWritesOnAProjectAreEachShape`、`TestEachWriteOnAProjectSharesItsWorkspaceFirst` | 单元；组合 |
| 清扫 34：`issueWrites` 的完整性 | T10.8 | 1/1 | `TestEachIssueWriteLocksItsProjectFirst` | 组合 |
| 清扫 2、19：失败的传递 | T1.10、T1.11、T1.13、T2.9、T2.10、T3.7、T4.10、T6.8、T7.10、T8.14、T11.19、T11.20、T11.21 | 13/13 | `TestAFailedReadIsAnErrorNotAnAnswer`、`TestDeleteWorkspaceRefusals`、`TestDeleteProjectReturnsEachFailure`、`TestDeleteWorkspaceProjects`、`TestTheCascadeStopsAtAFailingBatch`、`TestTheBatchStatementsDeleteAProjectsRowsInIDOrder`、`TestAFailedIssueStatementIsAnError`、等 4 个 | 单元；存储 |
| 清扫 22：回答对得上所问 | T8.15、T8.16、T8.17 | 3/3 | `TestCreateIssueStopsAtAFailingPort` | 单元 |
| 错误码的文案（约束 4） | T11.17、T11.18 | 2/2 | `error-messages.test.ts`、`check:sync` | 单元；守卫 |
| 关键词守卫（7.9） | T1.12 | 1/1 | 关键词守卫 | 守卫 |
| 清扫 31、32：交错的门和探针 | T12.2、T12.3 | 2/2 | `TestTwoCreationsOnAProjectSerialize` | 组合 |
| 端到端的过渡（裁定 7） | T14.3 | 1/1 | P4、W3 | 端到端 |
