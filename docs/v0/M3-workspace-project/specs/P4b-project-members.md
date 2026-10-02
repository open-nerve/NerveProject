# M3/P4b 项目的管理、显示设置与成员的加入：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M3/P4b `project-members` |
| 日期 | 2026-10-01；修订 2026-10-02（方案 E 和预检） |
| 状态 | 已完成（[评审记录](../reviews/P4b-project-members-review.md)）：第 3 节第 1–12 条已裁定（2026-10-01）；第 13 条由负责人定为方案 E（2026-10-02）；第 14–17 条由控制者裁定接受（2026-10-02） |
| 上级文档 | [M3 设计文档](../M3-design.md) 第 2（P2、P3、P4、P8）、3.3、3.4、3.5、3.6（约定二、三、五、六，加锁表）、3.12、3.18、3.19、3.20（P4b 一行）、4.11（P4b 各行）、5.1–5.3、6.4、6.7、9.1（恢复时的角色）、9.2、9.3（交错 17）、9.6、12（P4b 与约束 1–4）、13.1 节 |
| 前置交接 | [P4a spec](P4a-projects.md) 第 5 节 P4b 一行，[P4a review](../reviews/P4a-projects-review.md) 第 6 节（落点见第 3 节第 2 条）；[M1-P3](../handoffs/M1-P3-trim-platform.md) 的项目成员 |
| 裁定 | 负责人（2026-10-01）：第 3 节第 1–12 条照建议。负责人（2026-10-02，M3 设计 17.4）：方案 E，每个项目级的写最先取它的工作区行的 `FOR SHARE`（第 3 节第 13 条）；P4a 的 F-M3、预检的 M1 和 L1 由此关闭，设计 3.3 的"一个改变一个时刻"对连带改写的每一列都成立（第 7 节） |
| 计划 | [P4b plan](../plans/P4b-project-members.md) |

本 spec 只写 M3 设计交给 P4b 决定的东西：名字、签名、SQL、测试名，以及原型证明了什么。规则本身以 M3 设计为准，这里引用节号，不重述。P4b 依赖 P4a（`e22e5080` 的 `main`）。

P4b 是评审敏感的一段（M3 设计 12 节约束 3）：项目一侧的成员关系增长（约定三、六的添加和加入，恢复时的角色，目标的 422 在判定之后）与降为访客的连带在交错 17 相遇；每个项目级的写最先锁工作区行（约定二，方案 E），工作区一级的连带因此在工作区行上等它们、在它们之后读时刻；谁能修改、归档、删除、看成员、添加、加入项目是安全性质。附录 A 的十一类清扫逐个核对每个这样的测试在它所说的性质去掉之后失败，并记下失败在哪一层（单元、存储、组合、端到端）。

## 1. 目标

按 M3 设计 12 节 P4b：项目的修改、删除、归档、恢复，每个成员自己的项目显示设置，项目成员的列出、添加、加入；添加和加入这两条项目一侧的增长与降为访客串行（交错 17）。具体是：

- 九个操作：`updateProject`、`archiveProject`、`unarchiveProject`、`deleteProject`、`getProjectPreferences`、`updateProjectPreferences`、`listProjectMembers`、`addProjectMembers`、`joinProject`，各带操作名、规则行、矩阵行（共 175 格，含添加无效目标的 PM、X 格；执行时的修复轮另加已归档项目的两列和"成员关系已结束"的两个变体，矩阵共 390 格）；
- 写的唯一一条加锁路径（`Locks`，第 3 节第 14 条）：不加锁读项目的工作区；工作区行 `FOR SHARE`，写的第一把锁（方案 E）；添加、加入再取目标的工作区成员行；然后锁项目（`FOR NO KEY UPDATE`，改显示设置是 `FOR SHARE`）并确认它还在那个工作区；在全部锁之下、在事务的连接上判定；锁之后读时钟；
- 删除项目与删除工作区共用一处删除步骤（`deleteProjects`），由目录驱动的组合测试核对每张项目之下的表；
- 增长的一步（`growth`）由添加、加入共用：恢复以前的成员行时添加取请求的角色，加入取原来那一行与工作区角色中较低的一个（9.1 的表）；
- 交错 17（添加、加入各两个顺序），降级与删除项目，项目级的写与降级，同一个项目上的两个写，删除工作区与项目级的写，添加多个成员与删除工作区，全部在真实数据库上、经 `bootstrap` 组合出的模块；每个写的每条语句在事务的连接上（只有一个连接的池）；每个项目级的写先锁工作区，由一个组合测试逐个核对，它的行与矩阵中写在项目一级的行对上；
- 矩阵准备的拆分、账户的登记；故事 P2、P3、P4、P8 的接口版本，W3 先删除一个项目；
- 3.20 中 P4b 的一行（差异清单），M1-P3 交接的"处理结果（M3/P4b）"。

## 2. 交付物

### 2.1 文件总览

路径相对于仓库根目录。"生成"表示由 `make gen`（或 `make gen-go`）生成并提交，不手改。"Task"是 plan 中负责它的任务（plan 有完整的文件表）。本 Phase 102 个手写的文件、9 个生成物；没有迁移。跨模块端口只多一个方法：`WorkspaceDirectory.ShareWorkspaceByID`，与 P4a 的 `ShareWorkspaceBySlug` 同一个端口、同一种写法，`bootstrap` 照旧转换（第 3 节第 13 条）；`workspace` 模块只多这条查询和它的存储方法。

| 路径 | 内容 | Task |
|---|---|---|
| `bootstrap/permission_matrix_seed_test.go`、`permission_matrix_seeded_test.go`、`permission_matrix_test.go` | 矩阵准备的写入从表和登记中拆出（只移动） | 1 |
| `project/domain/patch.go`、`errors.go`；`adapter/postgres/update.go`、`members.go`、`projects.go`、`queries/projects.sql`、`members.sql` 及测试；`app/ports.go`、`fakes_create_test.go`；`workspace/adapter/postgres/queries/directory.sql`、`directory.go`、`directory_share_test.go`、`workspace/module.go`；`bootstrap/ports.go`、`ports_test.go` | 修改的领域；项目的锁；不加锁读项目的工作区；成员关系的读；工作区按 id 的锁 | 2 |
| `api/modules/project.yaml`；`project/app/ports.go`、`lock.go`、`update_project.go`、`fakes_write_test.go`、`clock_test.go`；`adapter/http/*`；`access/domain/rules.go`；`bootstrap/permission_matrix_project_test.go`、`project_writes_test.go`；前端文案 | `updateProject`；`Locks`；一个新码 | 3 |
| `project/app/archive_project.go` 及测试；存储的 `SetArchived`；契约、handler、规则、矩阵；`bootstrap/project_write_locks_test.go`、`project_writes_test.go` 的 `createdProject` | `archiveProject`、`unarchiveProject`；每个写先锁工作区的组合测试 | 4 |
| `project/app/deletion.go`、`cascade.go`；`adapter/postgres/cascade.go`、`queries/cascade.sql` 及测试；`bootstrap/workspace_deletion_catalog_test.go`、`workspace_deletion_test.go` | 删除项目的一处步骤；目录读取按父表 | 5 |
| `project/app/delete_project.go` 及测试；契约、handler、规则、矩阵；`bootstrap/project_deletion_test.go`、`project_write_locks_test.go` 的一行 | `deleteProject` | 6 |
| `project/domain/preferences.go`；`app/get_preferences.go`、`update_preferences.go`、`fakes_preferences_test.go`；`adapter/postgres/preferences.go`、`queries/preferences.sql` 及测试 | 显示设置的领域、存储、用例 | 7 |
| `project/adapter/http/preferences.go` 及测试；契约、矩阵；`bootstrap/project_preferences_test.go`、`project_write_locks_test.go` 的一行 | 显示设置的接口和组合 | 8 |
| `project/domain/member.go`；`app/list_members.go`；`adapter/http/members.go`；存储的 `ListMembers`；契约、规则、矩阵；`bootstrap/permission_matrix_seeded_test.go` 的账户登记 | `listProjectMembers` | 9 |
| `project/domain/member.go`、`member_test.go`；存储的 `RestoreMember`、`EnsurePreferences` | 增长与恢复的领域、存储 | 10 |
| `project/app/growth.go`、`add_members.go`、`fakes_growth_test.go` 及测试；`lock.go` 的目标、`module.go`；规则 | `addProjectMembers` 的用例 | 11 |
| 契约；`project/adapter/http/add_test.go`；`bootstrap/permission_matrix_members_test.go`、`project_writes_test.go`、`project_write_locks_test.go` 的一行 | 添加的接口；成员的矩阵行 | 12 |
| `project/app/join_project.go` 及测试；契约、handler、规则；`bootstrap/project_members_test.go`、`project_write_locks_test.go` 的一行 | `joinProject`；恢复时的角色 | 13 |
| `bootstrap/interleaving_growth_test.go`、`interleaving_writes_test.go`、`interleaving_deletion_test.go`、`project_connection_test.go`、`demotion_test.go`；`project/adapter/postgres/demote_test.go` | 交错 17；降级与删除、与项目级的写；两个写；删除工作区与项目级的写（L1、M1）；事务的连接 | 14 |
| `e2e/fixtures/api.ts`、`assert/project.ts`、`assert/workspace.ts`；`e2e/stories/project/p2-…`、`p3-…`、`p4-…`、`p8-….spec.ts`；`workspace/w3-workspace-settings.spec.ts` | 四个故事的接口版本；W3 先删除一个项目 | 15 |
| `docs/v0/plane-diff.md`、`handoffs/M1-P3-trim-platform.md` | 3.20 中 P4b 的一行；M1-P3 的项目成员 | 16 |
| `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`、`project/adapter/http/gen/server.gen.go`、`bodyshape.gen.go`、`project/adapter/postgres/gen/projects.sql.go`、`members.sql.go`、`cascade.sql.go`、`preferences.sql.go`、`workspace/adapter/postgres/gen/directory.sql.go`（生成） | | 2–10、12、13 |

`project/…`、`access/…`、`workspace/…` 指 `server/internal/modules/` 下的模块；`bootstrap/…` 指 `server/internal/bootstrap/`。

### 2.2 依赖

没有新依赖（6.1）。`server/go.mod`、`server/tools/go.mod` 和两个 `go.sum` 不变，仍是 `go 1.27` / `toolchain go1.27.1`；不加 npm 包。

### 2.3 修改的领域；项目的锁；成员关系的读；工作区按 id 的锁（Task 2；3.6 约定二、3.19、5.2、6.5）

- `domain.ProjectPatch`：每个可写的字段一个指针（名称、说明、标识、网络、四个功能开关、`guest_view_all_features`、`archive_in`、`logo_props`、时区），负责人、默认负责人各带一个 `SetLead`、`SetDefaultAssignee` 标志（标志在、id 为 `nil` 是清空）。`CheckProjectPatch` 对给出的字段照 `CheckNewProject` 的规则逐字段核对（3.19："修改时同一规则"），`archive_in` 0–`MaxArchiveIn`（12，4.6），全部问题一个 422；`TestThePatchChecksAsCreateDoes` 用 33 个值要求两者给出同样的字段错误。`CanAssign(role)`：项目的管理员、成员，按集合。`ErrArchived`（409 `project.archived`）、`Unassignable(field)`（`not_allowed`）。
- 项目的锁（父锁，约定二）：

```sql
SELECT workspace_id, (archived_at IS NOT NULL)::boolean AS archived
FROM projects
WHERE id = $id AND deleted_at IS NULL
FOR NO KEY UPDATE;
```

  等锁之后 Postgres 在行的最新版本上重新求值 `deleted_at IS NULL`，等待期间被删除的项目读不到（`TestTheProjectLockSeesADeletionItWaitedFor`）。`FOR NO KEY UPDATE` 与 `FOR SHARE`、另一个 `FOR NO KEY UPDATE` 冲突，与外键检查的 `FOR KEY SHARE` 不冲突（`TestLockProject` 核对前两者，和别的项目不被锁）。修改、归档、恢复、删除、添加、加入都取它（3.6 的加锁表）。
- `UpdateProject`：没给的字段 `coalesce` 保留，负责人、默认负责人只在标志在时写；`updated_by_id`、`updated_at`。建项目和修改共用 `taken(write, err)`：`projects_workspace_id_identifier_key`、`projects_workspace_id_name_key` 的 23505 是 `ErrIdentifierTaken`、`ErrNameTaken`，别的错误原样包装（`archive_in` 13 是 `projects_archive_in_check` 的违反，500：领域先查过，到这里是缺陷）。
- `Memberships(projectID, userIDs)`：问到的账户在这个项目里未删除的成员关系（有效的、已结束的），按账户。在项目的 `FOR NO KEY UPDATE` 之下读到的就是写时的：改项目成员关系的每个写都先取这把锁（3.6 的加锁表）。
- `ProjectWorkspace(id)`（`app.ProjectFinder`）：未删除的项目的工作区，已归档的也读，不加锁（`TestProjectWorkspace` 核对读的事务开着时项目的 `FOR UPDATE` 不等）。每个写在加锁之前先读它（第 3 节第 10 条），读（Task 7、9）在它上面判定。
- 工作区按 id 的锁（方案 E，第 3 节第 13 条）：

```sql
-- name: ShareDirectoryWorkspaceByID :one
SELECT id, timezone
FROM workspaces
WHERE id = $id AND deleted_at IS NULL
FOR SHARE;
```

  `workspace` 的 `Directory.ShareWorkspaceByID` 与 P4a 的 `ShareWorkspaceBySlug` 并列；`project` 的 `WorkspaceDirectory` 嵌入 `WorkspaceSharer.ShareWorkspaceByID`，`bootstrap` 的 `projectWorkspaces` 转换回答（`TestProjectWorkspacesConvertsWorkspacesAnswer`）。存储测试在新文件 `directory_share_test.go`（`directory_test.go` 已 396 行）：`TestShareWorkspaceByIDFindsTheUndeletedWorkspace`（两个工作区各得自己的，已删除的、不存在的找不到，失败是错误）、`TestTheDirectorysLockIsForShare/byID`（持锁时工作区的 `FOR NO KEY UPDATE` 等待、另一个 `FOR SHARE` 不等，别的工作区不被锁）、`TestTheDirectorysLockSeesADeletionItWaitedFor/byID`。执行时（T2-c）按 slug、按 id 的两份锁测试合成一份：P4a 按 slug 的两个测试从 `directory_test.go` 挪进 `directory_share_test.go`，每个测试跑 `bySlug`、`byID` 两个子测试，`directory_test.go` 余 296 行。

### 2.4 `updateProject`（Task 3；3.4、3.6、3.19、5.2）

- 接口：`PATCH /api/v0/projects/{project_id}`，`ProjectUpdate`（`additionalProperties: false`，只有 `project_lead_id`、`default_assignee_id` 可为 `null`），200 `Project`；码 `[validation_failed, project.not_found, forbidden, project.archived, project.identifier_taken, project.name_taken]`。新码 `project.archived` 加进 `PROBLEM_MESSAGES` 和两份 `auth.json`（约束 4）。
- 规则 `project.update`：`{Level: LevelProject, Roles: [admin]}`，项目管理员和是工作区管理员的项目成员（3.4）；不是成员的工作区管理员 403。
- `app/lock.go`：`Locks{projects ProjectLocks, workspaces WorkspaceSharer, members WorkspaceMembers, auth}`，项目级的写取锁和判定的唯一一条路径（第 3 节第 14 条）。`(Locks).lockAndDecide(ctx, actor, write{project, action, share, targets}) (held{project, grant, roles}, error)`，在 `ctx` 带的事务里：`ProjectWorkspace`（不加锁；没有是 404）→ `ShareWorkspaceByID`（工作区行 `FOR SHARE`；等待期间被删除是 404）→ 有 `targets` 时 `ShareMembers`（目标的工作区成员行 `FOR SHARE`，id 升序，约定三；他们的有效角色进 `roles`）→ `LockProject`，`share` 时 `ShareProject`（没有，或锁读到的工作区不是先读的那个，是 404）→ `decide`（看不到换成 `project.not_found`，别的失败原样返回）。`answer`（在事务里经 `GetProject` 读回存下的行，写完读不到是内部错误）。写不持自己的 `Authorizer`：`project.New` 建一个 `Locks` 给七个写。`share`、`findAndDecide` 在 Task 7，`members`、`targets`、`roles` 在 Task 11。
- 用例的顺序：`RequireActor` → `CheckProjectPatch`（事务之前）→ 事务：`Locks`（`project.update`）→ 已归档 409 → 负责人、默认负责人（只在给了时读 `Memberships`；不是有效成员或是访客：一个 422 列出每个字段，判定之后，约定三）→ 锁下读时钟 → `UpdateProject` → `answer`。不能改项目的人得到 403、404，得不到负责人的任何信息（矩阵的"负责人不是成员"一行）。
- `TestEachWriteReadsTheClockUnderItsLock`（`project/app/clock_test.go`）：每个改已有行的写在它的锁（工作区的 `FOR SHARE` 在先，然后项目的）、判定、检查之后读一次时钟（P2 spec 2.6、M3 设计 3.3），排在别的写之后的写不会盖上更早的时刻；`TestTheWritesOnAProjectStampTheirRequest`（组合出的 app）：每个写由调用者在请求之内盖上 `project.New` 的时钟。

### 2.5 `archiveProject`、`unarchiveProject`（Task 4；3.4、3.19）

- `POST /api/v0/projects/{project_id}/archive`、`/unarchive`，200 `Project`；码 `[project.not_found, forbidden]`；规则同 `project.update`。
- `NewArchiveProject`、`NewUnarchiveProject` 建同一个用例（`archive` 标志）：`Locks` → 锁下读时钟 → `SetArchived(id, archive, by, now)`（`archived_at` 是 `now` 或 `null`，`updated_by_id`、`updated_at`，别的列不动）→ `answer`。已归档的再归档取新的时刻，未归档的恢复照样成功（Plane `views/project/base.py:427-441`；第 3 节第 8 条）。
- 矩阵的已归档项目从 Task 4 起经存储归档，`standIns` 不再用 SQL 写它。
- `TestEachWriteOnAProjectSharesItsWorkspaceFirst`（`bootstrap/project_write_locks_test.go`，Task 4 起，每个写的 Task 加一行；第 3 节第 14 条）：组合出的 app 上，alice 在 acme 的 Web、Ops 上把每个项目级的写各执行一次，添加、加入的目标是 acme 的成员。Web：另一个事务持有 acme 行的 `FOR NO KEY UPDATE`（每个连带都这样持有），写在 `workspaces` 上等，等待期间既不持有 Web 也不持有目标在 acme 的成员关系（`FOR UPDATE NOWAIT` 都成功）。Ops：另一个事务持有 Ops 行的 `FOR NO KEY UPDATE`，写在 `projects` 上等，等待期间在自己的事务里持有 acme 的 `FOR SHARE`、不更强（`FOR NO KEY UPDATE NOWAIT` 失败、`FOR SHARE NOWAIT` 成功），也持有目标的成员关系。放开之后各自照常回答。写的行与矩阵中写在项目一级的行（`r.write` 且列是 `projectColumns`）逐个对上：一个新的项目级的写没有行，测试在连数据库之前失败。

### 2.6 删除项目；一处删除步骤（Task 5、6；3.3、3.6、3.19；移交第 7 条）

- `app.Deletion{WorkspaceID, ProjectID *uuid.UUID, By, Now}`；`app.ProjectsDeleter` 的四个方法 `DeleteProjects`、`DeleteProjectMembers`、`DeleteProjectPreferences`、`DeleteStates`（取代 P4a 的 `WorkspaceProjectsDeleter` 的四个 `DeleteWorkspace…`，第 3 节第 3 条）。`app/deletion.go` 的 `deleteProjects(ctx, p, d)` 是项目之下各表的唯一一处清单，按 3.6 的全局顺序：项目 → 项目成员 → 显示设置 → 状态，P7 在最后加标签。`Cascade.DeleteWorkspaceProjects`（`ProjectID` 为 `nil`）和 `deleteProject` 都调它，谁都不能漏掉对方删除的表。四条语句形同：

```sql
UPDATE projects
SET deleted_at = $now, updated_at = $now, updated_by_id = $deleted_by
WHERE workspace_id = $workspace_id AND ($project_id::uuid IS NULL OR id = $project_id)
  AND deleted_at IS NULL;
```

- `DELETE /api/v0/projects/{project_id}`，204；码 `[project.not_found, forbidden]`；规则同 `project.update`；已归档的照样删除。用例（`NewDeleteProject(ProjectsDeleter, Locks, tx, clock)`，不另设删除项目的端口）：`Locks` → 锁下读时钟 → `deleteProjects(Deletion{锁读到的工作区, &id, 调用者, now})`。
- 目录驱动的核对（移交第 7、8 条，P4a 的 F-M1）：`bootstrap/workspace_deletion_catalog_test.go` 的 `keysTo(t, pool, parent)` 是父表本身（`<父表>.id`）和目录中指向它的每个外键；只经别的表挂在父表下、自己没有指向父表的外键的表让测试失败，并说明它经哪张表挂在下面（`TestKeysRefuseATableWithoutItsParent`，对 `workspaces`、`projects` 两个父表）。`TestDeletingAProjectLeavesNoUndeletedRowUnderIt` 经 `keysTo("projects")` 读出每张表，经接口删除 Web：每个外键下删除之前未删除的行都在项目的时刻、由删除者删除，Ops 的每一行不变；`TestAProjectDeletionRefusedAtItsCommitChangesNoRow` 对 `keysTo("projects")` 的每张表各跑一次，让那张表的延迟约束触发器拒绝提交，两个项目下的每一行都不变（执行时 T6-a、T14：只拒绝最后一步的 `states` 时，在事务之外执行的最后一步拒绝的是它自己的提交，测试照样通过）。工作区删除的三个组合测试经 `keysTo("workspaces")` 照旧通过，没有加豁免。
- 存储：`TestDeletingAProjectSoftDeletesItsRowsAlone`（Web、Ops 各一次：已归档、在工作区第一个或最后一个；只动它和它下面的行，同一个时刻、同一个账户，除了三列什么都不写；同工作区的另一个项目、此前删除的项目、别的工作区的项目一行不动；再跑一次什么都不变）。
- 剩下的手写清单：存储测试的 `projectTables`（`adapter/postgres/cascade_test.go`，执行时并进了每张表的行数，没有准备行的表失败）、`seedProject`、`deleteAll`，`failures_test.go` 的取消上下文的步骤，`delete_project_test.go` 的每步失败行，e2e 的 `workspaceTables`；单元测试 `app/cascade_test.go` 的失败行由 `deletionSteps` 推出。新表漏在它们里面时，组合的目录测试和 e2e 的 `expectProjectDeleted`（也读目录）先失败（第 5 节 P7 一行）。

### 2.7 项目的显示设置（Task 7、8；3.18、4.8、5.2）

- 领域：`Navigation{DefaultTab, HideInMoreMenu}`、`Preferences{Navigation, SortOrder}`、`PreferencesPatch`（没给的不变，导航整个替换）、`DefaultPreferences()`（列的默认值：`work_items`、什么都不藏、65535）、`(Preferences).Apply`、`CheckPreferencesPatch`：默认标签页是 `work_items, cycles, modules, views, intake` 之一（网页的五个，3.18；`pages` 随文档页砍掉），藏起的是 `work_items` 之外的四个之一、只出现一次，大小写不同的算未知；全部问题一个 422，按位置命名字段。
- 读：`GET /api/v0/me/projects/{project_id}/preferences`，`findAndDecide(project_preferences.read)`（`ProjectWorkspace` 不加锁 → 判定，不开事务，6.7）→ `Preferences`，没有行时答 `DefaultPreferences()`，不写（3.18）。
- 改：`PATCH` 同一路径，`CheckPreferencesPatch`（事务之前）→ 事务：`Locks`（`share`：工作区 `FOR SHARE` → `ShareProject`，项目的 `FOR SHARE`，3.6 加锁表的"修改项目的显示设置"→ 判定）→ 锁下读时钟 → `UpsertPreferences`：

```sql
INSERT INTO project_user_properties AS p (…) VALUES (…)
ON CONFLICT (project_id, user_id) WHERE deleted_at IS NULL DO UPDATE
SET preferences = CASE WHEN $set_navigation THEN EXCLUDED.preferences ELSE p.preferences END,
    sort_order  = CASE WHEN $set_sort_order THEN EXCLUDED.sort_order ELSE p.sort_order END,
    updated_by_id = EXCLUDED.updated_by_id, updated_at = EXCLUDED.updated_at
RETURNING preferences, sort_order;
```

  冲突目标带部分唯一键的条件：不带时 Postgres 推断不出部分唯一索引，每次都报错（F3）。已归档项目的设置照改（3.19）。
- 规则 `project_preferences.read`、`project_preferences.update`：`{Level: LevelProject, Roles: [admin, member, guest]}`，项目的有效成员，各自的设置；不是成员的工作区管理员 403。
- 契约的 `ProjectTab` 列出五个标签页；请求体的结构检查不核对枚举的值，未知的标签页传到领域，答 422（第 3 节第 5 条）；缺字段、`null`、别的类型、多出的字段答 400。
- `TestEachMemberHasHisOwnDisplaySettings`（组合出的 app）：alice 改她在 Web 的设置，bob 在 Web 的、她在 Ops 的照旧，每人读回自己的，没有设置的读到默认值。

### 2.8 `listProjectMembers`；矩阵的账户登记（Task 9；3.12、5.2）

- `GET /api/v0/projects/{project_id}/members`，200 `ProjectMemberList{data: ProjectMember[]}`（`ProjectMember{id, project_id, member_id, role, created_at}`）；码 `[project.not_found, forbidden]`；规则 `project_member.list`：项目的有效成员（三种角色）。
- `findAndDecide` → `ListMembers`，不开事务：

```sql
SELECT id, project_id, member_id, role, created_at
FROM project_members
WHERE project_id = $project_id AND is_active AND deleted_at IS NULL
ORDER BY created_at, id;
```

  只读 `project_members`（第 3 节第 4 条）。
- 矩阵：`seeded.account(c)` 按列取账户的 id（`prepareMatrix` 登记；不是列的账户让测试立刻失败）；P4b 的路径参数只有 `{project_id}`，`targetViolation` 的 `_id` 规则不需要改（移交第 4 条），正文里的账户经 `s.account` 取，前提由 `projectSeed.targets` 核对（2.9）。

### 2.9 `addProjectMembers`（Task 10–12；3.5、3.6 约定三和六、3.18、9.2）

- 领域：`NewMember{MemberID, Role}`；`CheckNewMembers`（1–`MaxNewMembers`（100）个，角色是三种之一，同一个账户只出现一次：事务之前的 422，第 3 节第 9 条；个数不在范围内时只报个数、立即返回，重复按集合找，执行时由整分支评审的 M2 改定）；`CanAdd(workspaceRole, role)`：`addable` 表（工作区管理员只能作管理员、工作区访客只能作访客、工作区成员三种都可以，Plane `views/project/member.py:69-83`，3.5），按集合；`Target{NewMember, WorkspaceRole *Role, Member bool}`、`CheckTargets`：不是工作区的有效成员 `members[i].member_id` `not_allowed`，已是项目的有效成员 `duplicate`，角色不合规 `members[i].role` `not_allowed`，一个 422 按位置。
- 存储：`RestoreMember(id, role, by, now)`（已结束的恢复为有效、取给的角色，保留 id 和 `created_at`）；`EnsurePreferences(row)`（`ON CONFLICT (project_id, user_id) WHERE deleted_at IS NULL DO NOTHING`：已有的设置不动，恢复的成员关系保留它们）。
- `app/growth.go`：`growth{workspaceID, projectID, user, ended *Membership, role, sortOrder, by, now}.apply(ctx, MemberGrower)`：有已结束的成员关系就以 `role` 恢复，否则 `CreateMember`；然后 `EnsurePreferences`。添加、加入共用。
- 用例（`NewAddProjectMembers(AddMembersDeps{Locks, Projects, Tx, Clock})`）：`RequireActor` → `CheckNewMembers` → 事务，按 3.6 的加锁表经 `Locks`（`targets`）：`ProjectWorkspace`（不加锁，第 3 节第 10 条）→ 工作区 `FOR SHARE` → `ShareMembers`（目标的工作区成员行 `FOR SHARE`，id 升序，约定三）→ 项目 `FOR NO KEY UPDATE` → 判定（`project_member.add`）→ `Memberships` → `CheckTargets`（判定之后）→ 锁下读时钟 → 每个目标按请求的顺序：`LowestSortOrder`、`growth.apply`（恢复时取请求的角色，9.1；显示设置在 `SortOrderFirst`，3.18）→ 回答经 `ListMembers` 读回、按请求的顺序（201，第 3 节第 9 条）。整批在一个事务里：一个目标被拒，什么都不写。
- 接口：`POST /api/v0/projects/{project_id}/members`，`ProjectMembersAdd{members: ProjectMemberNew[]}`（`ProjectMemberNew{member_id, role}`），201 `ProjectMemberList`；码 `[validation_failed, project.not_found, forbidden]`；规则 `project_member.add` 同 `project.update`。契约的说明照 P4a review 第 6 节的写法避开守卫 `project-invitations` 的词序（移交第 6 条）：关键词守卫在 P4b 前后都是 5 个命中、都有例外。
- 矩阵（9.2 的 PM、X 两格，约定三）：添加一行把工作区成员 WM-公 加为成员；四个无效目标的变体行（X 的账户不是工作区成员；WG- 是工作区访客，作成员；WA- 是工作区管理员，作成员；PM 已是有效成员），能添加的列得到 422，别的列得到与有效目标相同的 403、404；已归档项目照样添加。`projectSeed.targets` 核对每个目标的前提（清扫 10）。
- `TestTheLeadAndTheDefaultAssigneeAreActiveMembersWhoAreNoGuests`（组合出的 app，移交 1a）：alice 把 carol 加为访客，bob 的成员关系已结束（SQL 代替 P5 的移出），dave 是 acme 的成员、不是 Web 的；三人作负责人、作默认负责人各被拒（422 指名字段），项目不变；成员 erin 两者都可以。

### 2.10 `joinProject`（Task 13；3.5、3.6 约定三和六、3.18、9.1）

- `POST /api/v0/projects/{project_id}/join`，200 `Project`；码 `[project.not_found, forbidden]`；规则 `project.join`：`{Level: LevelVisible}`。
- `CanJoin(workspaceRole)`：`joiners` 表（工作区的管理员、成员，Plane `views/project/invite.py:131-189`），按集合；`JoinRole(ended, workspaceRole)`：新的成员关系取工作区角色，已结束的取它原来的角色与工作区角色中较低的一个，顺序按 `roleOrder`（访客、成员、管理员），不按数字（约定六）。
- 用例（`NewJoinProject(locks, projects MemberJoiner, tx, clock)`）：`RequireActor` → 事务：`Locks`（`targets` 是他自己）：`ProjectWorkspace` → 工作区 `FOR SHARE` → `ShareMembers`（他自己的工作区成员行）→ 项目 `FOR NO KEY UPDATE` → 判定（`project.join`）→ `CanJoin`，否则 403（在读他的项目成员关系之前，G4：是项目成员的工作区访客也被拒，9.2 的 PG 一格）→ `Memberships`：已是有效成员的什么都不写 → 锁下读时钟 → `growth.apply`（`JoinRole`；显示设置在 65535，3.18）→ `answer`。
- 恢复时的角色（9.1 的表）：`TestJoinRole` 是四行和两个相等的情形；`TestARestoredMembershipGivesNoMoreThanItHad`（组合出的 app）经接口跑其中三行（5/15 → 5、20/15 → 15、15/20 → 15，第四行经接口是 404）和添加的一行（以前是 20，添加为 5 → 5）：同一行恢复为有效，由调用者在那次请求的时刻，建立的时刻不变；新加入的位置 65535。

### 2.11 端到端（Task 15；2 的 P2、P3、P4、P8，9.6；移交第 5 条）

- fixture：`api.ts` 的 `addProjectMembers(api, token, projectId, members)`、`amidAnotherWorkspace`（故事的项目建在调用者另一个工作区的两个项目之间，一前一后：丢掉项目 id 的查询无论先读哪一行都读到别的工作区的项目）；`assert/project.ts` 的 `expectMember`、`expectProjectDeleted`（从目录读出指向 `projects` 的每张表，每张都有这个项目的行，全部在项目的时刻、由删除者删除）；`expectProjectCreated` 只读未删除的项目（删除之后标识可以再用）。
- `assert/workspace.ts`：项目的四张表 `deletedAlone: true`。W3 现在在删除工作区之前删除 Old（成员作负责人，四张表各有它的行），这些行保留自己的时刻，其余的行带工作区的时刻（移交第 5 条：有意设置，原因在注释里）。
- **P2**：管理员列出每个项目，成员列出公开的和自己的，访客只有自己的，已归档的只在要求时列出；成员以成员加入公开项目、位置 65535，再加入什么都不变；成员加入私密项目、访客加入公开项目都是 404，访客加入他是成员的项目是 403。
- **P3**：项目管理员添加一个成员和一个访客，成员列出他们和自己；管理员改每个设置（`site` 存成 `SITE`），以成员为负责人和默认负责人；项目成员不能改（403）；负责人或默认负责人是访客、不是成员，`archive_in` 13：都是 422、不改任何东西。
- **P4**：归档：离开列表、进入已归档的列表，修改 409；恢复、再归档；删除：成员、显示设置、状态在同一时刻删除，之后每个操作都是 404，标识可以再用（移交 1d：删除项目之后"已删除"的谓词可以由故事准备）。
- **P8**：默认标签页设为 modules，把 views 藏进"更多"，把项目拖到侧边栏第一位，设置保留；未知的标签页、藏起 work_items 都是 422、不改任何东西；他的成员的设置是成员自己的。

### 2.12 交错与事务的连接（Task 14；3.3、3.6、6.7、9.3 的 4、13、17；移交 1e–1g、第 2 条；预检的 M1、L1、L2）

全部在真实数据库上，项目一侧经 `bootstrap` 那样组合出的 `project.New`（经接口），降级、删除工作区经工作区的用例（连带照 `bootstrap` 接上）。gate 在第一方的事务里、它持有第二方需要的锁时；`pgtest.WaitForLockWaitOn(table)` 证明第二方在那张表的行上等待，之后才放开 gate；每次等待都有期限。方案 E 之下工作区一级的连带（持工作区行的 `FOR NO KEY UPDATE`）与项目级的写（持它的 `FOR SHARE`）都在工作区行上相遇，所以除了同一个项目上的两个写，每个探测都指名 `workspaces`；每个探测都有一个测试变异，把它改成方案 E 之前的表，证明它不会被别的等待满足（附录 A）。

| 测试 | 顺序 | 第二方等在 | 探测 | 反例（附录 A） |
|---|---|---|---|---|
| `TestADemotionAndTheProjectSidesGrowthSerialize`（交错 17；加入、添加 × 新的、已结束的成员关系 × 两个顺序，8 个子测试） | 增长先：持有 acme 和 bob 在 acme 的成员行 `FOR SHARE`，在锁 Web 之前停在 gate | 降级等 `workspaces` | `workspaces`；等待期间 `webFree`（Web 的 `FOR UPDATE NOWAIT` 成功：两方都还没有锁 Web，P4a 的 F-M2 顺序） | 增长先锁项目、工作区在目标之后（`E-targets-after-project`、`E-share-after-targets`）、只有添加或加入不锁工作区、`w-add-no-shares`、`w-join-no-shares`、`E-demote-clock-early`、`t-17-probe-members` |
| 同上 | 降级先：持有 acme `FOR NO KEY UPDATE`，写了成员行之后停住（`demotedHolding`） | 增长等 `workspaces` | 同上 | 同上；`lo-demote-projects-first`（降级先锁项目：等待期间 Web 被持有）；之后增长读到他是访客：加入 404，添加作成员 422 `members[0].role` |
| `TestADemotionAndAProjectsDeletionSerialize`（两个顺序） | 删除先：持有 acme `FOR SHARE` 和 Web，判定之后停在 gate | 降级等 `workspaces` | `workspaces` | 只有删除不锁工作区、`t-delproj-probe-projects`、`tm-deletion-probe-wrong-table`；降级的项目一步在删除提交之后才锁项目，找到 Web 已删除、把它留在外面（`LockMemberProjects` 在等锁之后重新求值 `deleted_at` 由下面的存储测试核对，移交 1g） |
| `TestAProjectWriteAndADemotionSerialize`（修改、归档、恢复、改设置 × 两个顺序） | 写先：持有 acme `FOR SHARE` 和 Web（`FOR NO KEY UPDATE`，改设置 `FOR SHARE`），判定之后停在 gate | 降级等 `workspaces` | `workspaces` | `lo-*-decide-first`（先判定后锁）、四个写各自不锁工作区、四个 `w-*-no-tx`、`t-writes-probe-projects`；降级先时访客的写 403、改自己的设置 200 |
| `TestTwoWritesOnAProjectSerialize`（预检的 L2） | 归档先：持有 acme `FOR SHARE` 和 Web `FOR NO KEY UPDATE`，判定之后停在 gate | 同一个项目上的修改等 `projects` | `projects` | `lo-lock-share`（`FOR SHARE` 下两者都持有 Web，各自的修改等对方，40P01）、`lo-lock-none`、`lo-lock-outside-tx`、`t-two-probe-workspaces`；修改按已归档判定，409 `project.archived` |
| `TestWritesOnTwoProjectsOfAWorkspaceDoNotWait` | 归档 Web 先：持有 acme 和 Web，判定之后停在 gate | 修改 Ops 不等 | 5 秒之内答 200 | 工作区按 id 的锁取 `FOR NO KEY UPDATE`、`FOR UPDATE`（同一个工作区的写互等） |
| `TestAWorkspacesDeletionWaitsForTheWritesOnItsProjects`（预检的 L1，F-M3 的两个窗口；修改、加入两个子测试） | bob 的写先：持有 acme `FOR SHARE`，判定之后停在 gate | 删除工作区等 `workspaces` | `workspaces` | `Locks` 不锁工作区、修改或加入各自不锁工作区、`E-key-share`、`E-no-lock`、`E-wire-noop`、`E-module-no-share`、`E-wsdelete-clock-early`、`t-fm3-probe-projects`；写写下的每一行（Web；他的成员关系和显示设置）在删除的一个时刻、由 alice 删除和最后写入，不早于建立，Web 的删除时刻不早于修改答复的 `updated_at` |
| `TestAddingSeveralMembersAndDeletingTheWorkspaceSerialize`（预检的 M1） | carol 加入 Ops 先：持有 acme 和她的成员行 `FOR SHARE`，判定之后停在 gate；alice 删除 acme；alice 添加 carol、dave | 删除工作区等 `workspaces`；添加与加入共享，不等 | `WaitForLockWait`；添加答 201（两个后端都在等时先等 1.2 秒，让添加那一次死锁检查过去） | `Locks` 不锁工作区（`E-share-none-M1`：删除工作区 40P01）、只有加入不锁工作区、`E-module-no-share`；三条新的成员关系在删除的一个时刻删除，没有 40P01 |

- `TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 4 起，2.5）：每个写自己的第一把锁；它与上表一起核对每一种相遇。
- `TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（P4a 的 L3，移交 1e 的 `az-project-outside-tx`）：`project.New` 和 `Authorizer` 照 `bootstrap` 接在只有一个连接的池上（请求 3 秒截止，每次等待 5 秒看门狗），alice 建 Ops、修改、归档、恢复 Web、改设置、添加 bob（恢复他已结束的成员关系），carol 新加入，alice 删除两个项目：每个写的锁（工作区按 id 的锁在内）、读、判定读的事实（`ProjectFacts`、工作区角色）、写和回答都必须在事务的连接上，经池发出的语句等第二个连接、请求在截止时失败（清扫 8 的 20 个变异，加上 `E-on-pool`、`E-wire-outside-tx`）。
- `TestAGrowthRefusedAtItsCommitLeavesNoRow`：增长在一个事务里（`refusingCommits(t, pool, table)` 先拒绝成员关系、再拒绝显示设置的提交：添加和加入答 500，什么都不留）。
- `TestLockMemberProjectsLeavesOutAProjectDeletedWhileItWaited`（存储）：另一个事务持有 Web 并软删除它，`LockMemberProjects` 等它，提交之后只返回 Ops。
- 原型：七个交错测试和 `TestEachWriteOnAProjectSharesItsWorkspaceFirst` `-count=5 -race`：25 秒，40 个顶层 PASS（100 个子测试），没有 FAIL，输出中没有 40P01、没有 DATA RACE（附录 A）。

### 2.13 矩阵（Task 1、3、4、6、8、9、12、13；9.2；移交第 4 条）

- **拆分**（Task 1）：`permission_matrix_seeded_test.go`（P4a 留下 400 行）沿接缝拆开：表、`seeded` 的登记和查找留下；两个存储的写入（`matrixSeed`、`projectSeed`）、SQL 替身（`standIns`）、`partingStates`、前提检查（`preconditions`）移进 `permission_matrix_seed_test.go`。本 plan 结束时两者 255 行、260 行，`permission_matrix_test.go` 361 行。
- **PM+WA 与 WA-**（移交 1b）：修改、归档、恢复、删除、添加的行要求 PA 和 PM+WA 得到允许、WA-（不是项目成员的工作区管理员）得到 403；显示设置、成员列表的行要求 PM+WA（项目成员）得到允许、WA- 得到 403。规则从 `LevelProject` 改为 `LevelWorkspace` 的变异（`r-*-workspace-admin`）都被矩阵发现。

| 行 | 列 | 格子 | 答案的核对 |
|---|---|---|---|
| `updateProject` | 项目级 12 列 | 200、403、403、200、403、403、404 × 6 | `renamesItsProject`：这一列的项目改了名，调用者的角色 |
| `updateProject`，负责人不是成员 | 同上 | 422、403、403、422、403、403、404 × 6 | |
| `updateProject`、`archiveProject`、`unarchiveProject`、`deleteProject`、`updateProjectPreferences`、`addProjectMembers`、`joinProject`，已归档 | 已归档项目 1 列 | 409；200；200；204；200；201；200 | 同各自的行 |
| `archiveProject`、`unarchiveProject` | 项目级 12 列 | 200、403、403、200、403、403、404 × 6 | `archivesItsProject`：归档的时刻等于最后修改的时刻，或为空 |
| `deleteProject` | 同上 | 204、403、403、204、403、403、404 × 6 | |
| `getProjectPreferences`、`updateProjectPreferences` | 同上 | 200 × 4、403、403、404 × 6 | `readsPreferences`：标签页、藏起的、65535 |
| `listProjectMembers` | 同上 | 200 × 4、403、403、404 × 6 | `listsTheProjectMembers`：`acme/public` 的五个有效成员按建立的顺序 |
| `addProjectMembers` | 同上 | 201、403、403、201、403、403、404 × 6 | `addsTheMember`：WM-公 在这一列的项目里作成员 |
| `addProjectMembers`，四个无效目标 | 同上 | 422、403、403、422、403、403、404 × 6 | |
| `joinProject` | 同上 | 200、200、403、200、200、200、404 × 6 | `joinsAs`：角色（已是成员的照旧，WA- 20、WM-公 15）、65535 |

共 175 格，加 P1–P4a 的 181 格是 356 格；整个矩阵 1.42 秒，准备 0.11 秒（9.2 的预算是 20–30 秒）。

### 2.14 文档（Task 16；3.20 的 P4b 一行）

| 文档 | 位置 | 内容 |
|---|---|---|
| 差异清单 | 四 | 项目负责人、默认负责人一行补上修改时的规则；项目标识一行写"修改项目时同一规则"；新加修改和删除、归档和恢复、添加已是有效成员的人、加入时恢复以前的成员行四行（4.11 中标 P4b 的行） |
| M1-P3 交接 | 文末 | "处理结果（M3/P4b）"：项目成员只经 `addProjectMembers` 和 `joinProject`（接口一侧完成）；页面改调新接口、守卫例外的删除、`RESTRICTED_URLS` 留给 P8 |

## 3. 与设计的差异和补充（待控制者裁定）

以下都没有改变 M3 设计的架构：第 13 条是负责人对设计的决定（M3 设计 17.4），已写进设计；其余每条给出建议。

1. **任务的划分：16 个，不是 9 个**。设计的任务与 plan 的对应：1 → 2（领域、锁、存储）、3（用例）；2 → 4（归档、恢复）、5（一处删除步骤）、6（删除）；3 → 7（领域、存储、用例）、8（接口）；4 → 9（列表）、10（恢复、增长的存储）；5 → 11（用例）、12（接口、矩阵）；6 → 13；7 → 14；8 → 15；9 → 16（review 是控制者的）；Task 1 是 P4a review 第 6 节要的矩阵拆分。拆分的理由是每个 Task 在约 1,500 行以内并各自是绿的：设计的任务 1、2、5 连同各自的存储、矩阵行和组合测试都超过上限。方案 E 加进 Task 2（工作区按 id 的锁、`ProjectWorkspace` 从 Task 7 移来）、3（`Locks`）、4（每个写先锁工作区的组合测试，Task 6、8、12、13 各加一行）、11（`Locks` 的目标）、14（删除工作区的两个交错、两个写的交错），任务的数目和划分不变。最大的是 Task 14（1,426 行）、Task 2（1,404 行）和 Task 3（1,373 行）；plan 共 14,862 行，16 个在 brief 的上限之内。**建议接受。**
2. **移交的落点**（说明）：

   | 移交 | 落点 |
   |---|---|
   | 1a `archive_in`、默认负责人、修改时负责人的规则 | Task 2（领域、存储）、3（用例、矩阵）、12（组合出的 `TestTheLeadAndTheDefaultAssigneeAreActiveMembersWhoAreNoGuests`） |
   | 1b PM+WA 一列在修改、添加、归档行里区分工作区管理员 | Task 3、4、6、8、9、12 的矩阵行（2.13）；添加的"工作区管理员作成员"变体在 Task 12 |
   | 1c `joinProject` 按集合、在"已是有效成员"之前（G4） | Task 13（`CanJoin`、`TestJoinProjectRefuses`、矩阵 PG 格） |
   | 1d 故事清扫：删除项目让"已删除"的谓词可以准备；P2–P4、P8 单独运行看得到各自的谓词 | Task 15（P4 删除之后每个操作 404）；附录 A 清扫 1 的故事一半；看不到的和互相遮住的见第 6 条 |
   | 1e 项目级的写在事务里判定（L3，`az-project-outside-tx`） | Task 14（`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`；`TestAProjectWriteAndADemotionSerialize` 的降级先的一半） |
   | 1f 交错 17 的锁强度 | Task 2（`LockProject` 的 `FOR NO KEY UPDATE`，`TestLockProject`）；Task 14（交错 17） |
   | 1g 降级与删除项目 | Task 14（`TestADemotionAndAProjectsDeletionSerialize`、`TestLockMemberProjectsLeavesOutAProjectDeletedWhileItWaited`） |
   | 2 交错 17 含 P4a 的 F-M2 顺序 | Task 14（2.12；`lo-demote-projects-first`） |
   | 3 一个连带一个时刻（F-M3） | 第 13 条（方案 E 关闭它）；第 7 节 |
   | 4 矩阵：PA 与 WA- 分开；项目行的 `_id`；拆分；接近 400 行的文件 | Task 1（拆分）、9（账户登记）；P4b 的路径参数只有 `{project_id}`，`_id` 规则不需要改；`schema_test.go`、`directory_test.go`、`invitations_test.go` 不改，`permission_matrix_test.go` 361 行 |
   | 5 e2e：`expectProjectCreated` 读未删除的行；`deletedAlone` | Task 15（2.11） |
   | 6 关键词守卫 | Task 12、13 的契约措辞；守卫前后都是 5 个命中、都有例外，没有新的例外 |
   | 7 删除项目的步骤与 P7 的标签：一处清单或读目录 | Task 5（`deleteProjects` 一处，`keysTo` 读目录）；第 3 条 |
   | 8 锁之后读时钟；锁下的重读确认父行；目录驱动的核对；`apitest.Main`；端口的错误原样返回；按集合 | 时钟：Task 3、4、6、7、11、13 的 `TestEachWriteReadsTheClockUnderItsLock`；重读：工作区按 id 的锁、项目的锁和 `ShareProject` 在等待之后重新求值 `deleted_at`，`Locks` 在项目的锁下确认项目还在先读到的工作区，之后的写用锁读到的工作区；目录：Task 5、6（没有新表，没有豁免）；`apitest.Main`：每个新码有 HTTP 测试回答；端口的错误：清扫 2；按集合：`CanAssign`、`CanAdd`、`CanJoin`、`roleOrder` |

3. **删除的端口改形**（移交第 7 条）：P4a 的 `WorkspaceProjectsDeleter` 的四个 `DeleteWorkspace…` 方法改为 `ProjectsDeleter` 的四个方法，各收一个 `Deletion`（项目可选）；`Cascade.DeleteWorkspaceProjects` 的签名和行为不变（P4a 的组合测试照旧通过）。`deleteProjects` 是项目之下各表的唯一一处清单，P7 在这里加标签；组合的目录测试（`keysTo`）对两个父表都读目录。这是 `project` 模块内部的端口，不跨模块。**建议接受。**
4. **成员列表只读 `project_members`**：不与工作区成员关系做连接（模块之间不 JOIN，6.5）。一个有效的项目成员一定是工作区的有效成员：每条增长锁住他的工作区成员行（约定三），每条收缩结束他的项目成员关系（约定六，`EndMemberships` 在 P5）。矩阵里被移出的成员（P5 之前由 SQL 替身结束了工作区成员关系）因此仍在列表里；P5 的移出写好之后替身换成存储，他就不在了（第 5 节 P5 一行）。**建议接受。**
5. **未知的标签页答 422，不是 400**：契约的 `ProjectTab` 列出五个，生成的请求体结构检查只核对结构，不核对枚举的值；未知的值传到领域，`CheckPreferencesPatch` 答 422（设计第 2 节的 P8"未知的标签页 422"）。HTTP 测试照此传未知的值（`TestUpdateProjectPreferencesPassesTheChange`）。**建议接受。**
6. **故事看不到的谓词，和互相遮住的"已删除"谓词**（清扫 1 的故事一半，移交 1d）：P2、P3、P4、P8、W3 单独运行时，41 个查询谓词变异和 `ProjectFacts` 的两个"已删除"谓词中 29 个被故事发现，14 个看不到：
   - `LockProject`、`ShareProject`、`ProjectWorkspace` 的 `deleted_at IS NULL` 各自单独去掉时由别的遮住：读先经 `ProjectWorkspace`、再经 `ProjectFacts` 的 `p.deleted_at`（判定答 404）；方案 E 之下写也先经 `ProjectWorkspace`（第 10 条），再经它的锁和 `ProjectFacts`。`ProjectWorkspace` 的与 `ProjectFacts` 的一起去掉时 P4 发现（`pair-pw`）；锁的与 `ProjectFacts` 的一起去掉在方案 E 之下由 `ProjectWorkspace` 遮住（`pair-lp`、`pair-sp` 存活，方案 E 之前 P4 发现它们），三者一起去掉时 P4 发现（`triple-lp`、`triple-sp`）。工作区按 id 的锁的 id、`deleted_at IS NULL` 见第 15 条。
   - `Memberships` 的账户（两序）、已删除（两序），`ListMembers` 的已结束、已删除，`RestoreMember` 的 id，`Preferences` 的已删除（两序），`ProjectFacts` 的 `m.deleted_at`：挡的是已结束或已删除的成员关系、设置，故事里只有 P5 的移出、离开（已结束）写得出；项目还在时已删除的成员关系、设置，P4b 的接口写不出（只有删除项目连同项目一起删除），以后的 Phase 若有写得出它们的接口，由它的故事清扫接过。`Memberships` 的账户谓词要一次添加多个账户、其中一个在项目里有别的行。
   这 14 个都由存储测试在两个行序下发现（附录 A）。**建议接受；P5 的故事清扫接过"已结束"的那些（第 5 节）。**
7. **只在单元或存储一层发现的变异**（brief：安全或加锁的性质只由单元一层发现的算缺口）：
   - `CanJoin` 按大小比较（`d-join-magnitude`）只由 `TestCanJoin` 发现：数据库的 CHECK 只容许 5、15、20，经接口按大小与按集合分不开。`CanAssign`、`CanAdd` 的同类变异由组合出的测试发现（取三种之内的角色就能区分）。
   - 没有调用者时不读（清扫 11 的 8 个 `nocaller-*`）只由单元测试发现：经接口的每个请求都已认证，用例收不到没有调用者的 `ctx`。
   - `archive_in` −1、标识和名称不按规则查（`u-archive-in-low`、`u-identifier-unchecked`、`u-name-unchecked`）只由领域测试发现：数据库的 CHECK 和列的类型拒绝这些值（答 500 而不是 422），不是安全性质。
   - 加锁的性质只在存储一层发现的：`LockProject` 取 `FOR UPDATE`（`TestLockProject`：多挡外键检查的 `FOR KEY SHARE`，经接口分不出）；`LockMemberProjects` 留下等锁时删除的项目（`TestLockMemberProjectsLeavesOutAProjectDeletedWhileItWaited`）；方案 E 之下另有三个，见第 15 条。都在真实数据库上，不是缺口。`LockProject` 取 `FOR SHARE` 现在也由组合一层的 `TestTwoWritesOnAProjectSerialize` 发现（预检的 L2：两个写都持有 Web，各自的修改等对方，40P01）。
   **建议接受。**
8. **再归档取新的时刻，恢复未归档的照样成功**：3.19 没有写。照 Plane（`views/project/base.py:427-441`：归档就是写当前时刻）；同一个请求重复发出时答复一致。**建议接受。**
9. **添加的回答和请求里的重复**：添加答 201 `ProjectMemberList`，按请求的顺序，经 `ListMembers` 在事务里读回（恢复的成员关系保留 id 和建立的时刻）。请求里同一个账户出现两次是事务之前的 422（`members[i].member_id` `duplicate`，"is listed before"），与"已是有效成员"同一个码、不同的说明。**建议接受。**执行时（T11-b）发现它不只是领域的事：漏掉这个检查时，恢复的路径按 id 恢复同一行两次、答一个错的 201，所以它由用例的拒绝行和组合测试 `TestAnAddThatNamesAnAccountTwiceWritesNothing` 守着。
10. **每个项目级的写先不加锁读项目的工作区**（方案 E 之后本条覆盖七个写，`Locks` 的第一步）：3.6 加锁表的第一步是工作区行（第 13 条），而请求只给项目的 id；`ProjectWorkspace` 不加锁读出工作区，`Locks` 再按表的顺序加锁：工作区 `FOR SHARE`，添加、加入的目标的工作区成员行，项目。项目的锁读到的工作区必须是先读到的那个，否则 404（`TestUpdateProjectRefuses` 的"moved to another workspace"）；锁之后各处用锁读到的那个（删除的 `Deletion`、显示设置的行、增长的成员关系和 `LowestSortOrder`）。没有写能把项目移到别的工作区，所以这个确认在组合一层不会失败：把增长换成不加锁读到的工作区是**等价变异**（`pf-join-unlocked-workspace-2`，组合一层存活，单元一层由"moved"发现）。"锁之后用锁读到的那个"因此是评审的事实，不是测试的事实：预检逐处核对过代码，本轮在原型上照样核对（`join_project.go`、`add_members.go` 用 `h.project.WorkspaceID`，`delete_project.go` 用它作 `Deletion` 的工作区，`update_preferences.go` 用它作显示设置的行；只有 `ShareWorkspaceByID` 和 `ShareMembers` 用先读的 id，不可避免）。没有的项目在第一步答 404，与看不到的 404 相同，不泄露什么。**建议接受。**
11. **小的结构调整**（说明）：`SortOrderReader`（`LowestSortOrder`）从 `ProjectCreator` 拆出，`MemberAdder` 只要它；`refusingCommits` 按表（P4a 只拒绝 `workspace_members` 的修改），两个 P4a 的测试照旧用 `workspace_members`；`TestTheWritesOnAProjectStampTheirRequest` 的写一个接一个跑在同一个项目上。
12. **原型中加强的测试，写 spec 时补的变异**（说明）：清扫中发现变异存活的地方，测试在原型中加强，并从带来它的 Task 起改进每一份快照（`amend.py`）：`amidAnotherWorkspace`（故事的谓词在两个行序下都看得到）、一个连接的池的测试、负责人和默认负责人的组合测试、矩阵添加的变体行和 `targets` 的前提、`update_test.go` 的 `cycle_view` 夹具、显示设置读到别的账户的测试、删除的 `unwritten`。写本 spec 的附录时发现修改和显示设置的领域规则、两个唯一键的 409、已归档项目被锁找到、删除的同一时刻没有自己的变异，补了 11 个（`mutants_s12.py`），全部被原有的测试发现，其中 6 个另由故事发现（附录 A）。
13. **方案 E：每个项目级的写最先取它的工作区行的 `FOR SHARE`**（负责人 2026-10-02 决定，M3 设计 3.3、3.6 约定二和五、加锁表、17.4）。
    - **起因**：本条原来报告的 F-M3（P4a review 第 7 节）：连带（降为访客的 `LockMemberProjects`、删除工作区的 `DeleteProjects`）在工作区的锁之后读时刻，之后才逐行等项目级的写持有的行；写以更晚的时刻提交，连带随后以较早的时刻改写同一行，`updated_at` 倒退。预检（plan `5740002c`）又找到同一窗口的两面和一个死锁：**L1**，交错 17 中增长在先时降级写下的 `updated_at` 早于新成员关系的 `created_at`；删除工作区经过一个进行中的加入时，`deleted_at` 早于加入新建的行的 `created_at`。**M1**，添加两个以上的成员与删除工作区成环（第 16 条）。三者同出一处：项目级的写与工作区一级的连带没有共同的父行锁。
    - **考虑过的**：A 不改，把窗口写成连带请求的期限；B 连带写 `GREATEST(updated_at, $now)`（L1 的 `deleted_at < created_at` 和 M1 另要修）；C 连带在自己的全部锁之后读时刻（一个连带的各行不再是一个时刻，W3 要改写）；E。负责人选 E；M1 的另一种修法（工作区的存储按 id 顺序批量锁成员行的 `LockWorkspaceMembers`）和 B 的 `GREATEST` 都不做。
    - **做法**：一条路径（第 14 条）。工作区按 id 的锁是 `WorkspaceDirectory` 上与 `ShareWorkspaceBySlug` 并列的 `ShareWorkspaceByID`（`deleted_at IS NULL`，`FOR SHARE`），`bootstrap` 照旧转换；`workspace` 模块只多这条查询和它的方法。每个写的顺序：不加锁读项目的工作区（第 10 条）→ 工作区 S → 添加、加入的目标的工作区成员行 S（id 升序，约定三、六照旧）→ 项目 N（改设置 S），确认工作区 → 判定 → 检查 → 时钟 → 写。没有 S 到 N 的升级：同一个写里工作区行只取一次 S。
    - **它让什么成立，由哪个测试守着**（层见附录 A）：
      - 每个写先锁工作区，在自己的事务里，在目标和项目之前，强度是 `FOR SHARE`：`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（组合，Task 4 起每个写一行）；存储一层 `TestTheDirectorysLockIsForShare/byID`。
      - 不比 `FOR SHARE` 弱（`FOR KEY SHARE` 时连带的 N 不等它）：上一条两个测试和 `TestAWorkspacesDeletionWaitsForTheWritesOnItsProjects`；不比它强（同一个工作区的写互等）：`TestWritesOnTwoProjectsOfAWorkspaceDoNotWait`。
      - 在调用者的事务里：`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（锁在写等项目时仍被持有）、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`；接线不能是空的：`TestProjectWorkspacesConvertsWorkspacesAnswer` 和上面的组合测试。
      - M1 关闭：`TestAddingSeveralMembersAndDeletingTheWorkspaceSerialize`（第 16 条）。
      - F-M3 的两个窗口关闭（L1 在内）：交错 17 的每个子测试核对他在 Web 的成员关系最后写入不早于建立（增长先时也不早于 gate 放开：降级的时刻在它持有 acme 之后读）；`TestAWorkspacesDeletionWaitsForTheWritesOnItsProjects` 的修改一路核对 Web 的 `updated_at`、`deleted_at` 是删除的一个时刻、不早于修改答复的 `updated_at`、`updated_by_id` 是 alice，加入一路核对他新建的成员关系和显示设置 `deleted_at = updated_at` 是那个时刻、不早于 `created_at`。去掉相应的写的工作区锁，这些测试各自失败（`E-skip-updateProject`、`E-skip-joinProject`、`E-share-none`，附录 A）。
      - 连带在工作区的锁之后读时刻：`E-wsdelete-clock-early`、`E-demote-clock-early`（把时刻挪到锁之前）由上面两个测试发现。
      - 时钟在全部锁之后：`TestEachWriteReadsTheClockUnderItsLock`（调用记录里工作区的锁在时钟之前）。
    - **代价**（执行时整分支评审的 I2 改正，负责人 2026-10-02 确认，M3 设计 17.4）：两边不对称。工作区一级的写（修改、删除工作区，改成员的角色，接受邀请；以后移出、离开、恢复成员、停用）取工作区行的 `FOR NO KEY UPDATE`；只有共享锁持有这一行时，PostgreSQL 让新的 `FOR SHARE` 立即取得，不排在等待中的它之后，所以连续重叠的项目级的写可以让它一直等到请求期限（默认 15 秒），到期失败、回滚、可以重试；先锁后判定，被拒绝的请求也先取这把 S。按每个账户每分钟 1,200 次的限速和每个写几毫秒的持锁，风险低，代码不改。项目级的写之间 S 与 S 不冲突。约定二覆盖项目管理类的写（P4b、P5、P7）；M4 的工作项写入不默认沿用，由 M4 的设计决定。预检的反例复现在 E 之下：原型的 `deadlock-demo-3E.sh`（预检的 `deadlock-demo-3.sh` 换成 E 的语句）在 E 下没有 40P01，去掉添加的工作区锁时 40P01。
14. **一条加锁路径，和它的完整性核对**（brief："一处，不是七个调用点"）：`app.Locks`（`lock.go`）是项目级的写取锁和判定的唯一一条路径，写的用例只持 `Locks`、不持 `Authorizer`，所以一个写不经它就无法判定；`project.New` 只建一个 `Locks`。写用 `write{project, action, share, targets}` 说明自己要什么：改项目之下的行、不改项目行和成员关系的写 `share`，让账户成为项目成员的写给 `targets`。`TestEachWriteOnAProjectSharesItsWorkspaceFirst` 的写的列表与矩阵中写在项目一级的行逐个对上（不是手写的清单，P4a 的 F-M1 的教训）：契约里一个新的项目级的写有矩阵行（`TestThePermissionMatrixCoversEveryOperation` 要求）而没有这里的行，测试在连数据库之前失败。执行时的修复轮（P8，整分支评审改定范围）把"项目级的写"认作两者之一：路径带 `{project_id}` 的非 GET 操作，或矩阵里有某行的列含项目一级调用者的非 GET 操作；原来只认列恰好是 `projectColumns` 的行，会漏掉列不同的写（例如只对本人的离开）和 P5、P7 按资源寻址的写。**P5、P7 怎样继承**：3.6 的加锁表上，P5 的改项目成员的角色、移出项目成员、离开项目和 P7 的状态、标签的写都是"工作区 S → 项目 N → …"：它们的用例只拿到 `Locks`，以 `write{project, action}` 取锁（不给 `targets`：它们不让人成为项目成员；不给 `share`：它们改项目之下的成员关系或项目自己的行），重读、检查在 `lockAndDecide` 之后；它们的操作进矩阵时，完整性核对要求它们在 `projectWrites` 各有一行，否则失败。**建议接受。**
15. **方案 E 改变了发现的层**（brief 的层的规则；附录 A 逐条）：
    - 升到组合一层：`LockProject` 取 `FOR SHARE`（`lo-lock-share`、`pf-lockproject-share`，`TestTwoWritesOnAProjectSerialize`）。
    - 从交错 17、写的交错移到 `TestEachWriteOnAProjectSharesItsWorkspaceFirst`（仍在组合一层）：增长先锁项目、加入不锁他的成员关系（`lo-add-project-first`、`lo-join-project-first`、`lo-join-no-share`、`pf-add-project-first`、`pf-join-no-share`），`ShareProject` 取 `FOR KEY SHARE`、不加锁（`lo-share-*`），`LockProject` 不加锁（`lo-lock-none`，另有 `TestTwoWritesOnAProjectSerialize`）。方案 E 之下第二方都等在工作区行上，交错 17 和写的交错看不到项目行上的顺序；它们仍由交错 17 发现的是增长与降级之间的事（`w-*-no-shares`、降级先锁项目，`webFree`）。
    - 降到存储一层：降级的 `LockMemberProjects` 取 `FOR SHARE`（`lo-member-projects-share`，P4a 的 `TestLockMemberProjectsLocksInIDOrder`、`TestDemotingAMemberToGuest`）：每个项目级的写先等降级持有的工作区行，降级的项目锁不再与任何写相遇；它挡的只剩将来不经工作区锁的写，P5、P7 的写都经 `Locks`。工作区按 id 的锁去掉 `deleted_at IS NULL`（`E-deleted-found`，存储的两个测试）：组合一层由项目锁的 `deleted_at` 和连带遮住（删除工作区在同一个事务里删除它的项目）。去掉工作区的 id（`E-id-fwd`、`E-id-rev`，两个行序）：`Locks` 只看找到与否、不用它答的工作区，组合一层和故事看到的只有多锁了别的工作区，由 `TestShareWorkspaceByIDFindsTheUndeletedWorkspace`、`TestTheDirectorysLockIsForShare/byID`（别的工作区被锁）发现。
    - 只在单元一层：等待期间被删除的工作区答成找到、项目的锁读到的工作区不核对（`E-share-gone-dropped`、`E-confirm-dropped`，`TestUpdateProjectRefuses`）：同上的遮挡，和没有写能移动项目（第 10 条）。
    - 等价：交错 17 的 gate 移到 `ShareMembers` 之前（`tm-gate-before-share`）：增长在 gate 之前已持有 acme，第二方照样等在 `workspaces`；交错 17 去掉 `webFree` 同时加入先锁项目（`tm-webfree-dropped`）：两种顺序都不成环，先锁项目由 `TestJoinProject`、`TestEachWriteOnAProjectSharesItsWorkspaceFirst` 发现，`webFree` 是交错 17 里唯一核对它的地方。连接池上的接线（`postgres.DB` 从 `ctx` 取事务）与正确的接线等价；接在写的事务之外（`E-wire-outside-tx`）被发现。
    - 只有添加不锁工作区（`E-skip-addProjectMembers`）不能单独复现 M1：M1 的排程里加入自己的工作区锁让删除等在 acme，添加照样走完；由 `TestEachWriteOnAProjectSharesItsWorkspaceFirst` 和交错 17 发现。方案 E 之下没有哪个真实的写只持工作区成员行的 S 而不持工作区的 S，M1 只剩纯时序才能碰到。
    - 故事一层（清扫 1 的故事一半在最终的原型上重跑）：`ProjectWorkspace` 的 id 降序多被 P2、P4 发现（每个写先读它）；`LockProject`、`ShareProject` 的 `deleted_at IS NULL` 与 `ProjectFacts` 的一起去掉不再被 P4 发现（`pair-lp`、`pair-sp`：写先经 `ProjectWorkspace` 的 `deleted_at`），与它也一起去掉时 P4 发现（`triple-lp`、`triple-sp`，第 6 条）；存储一层照旧发现每一个。
    **建议接受。**
16. **预检的 M1：添加两个以上的成员与删除工作区死锁**（Medium，方案 E 关闭）：添加按 id 升序共享锁住目标的工作区成员行，删除工作区的成员关系一步按扫描顺序批量改这些行；一个第三方（另一个项目上的加入）先持有其中一行时，删除持有 acme、改了一行再等第三方的那一行，添加持有它、等删除改过的那一行，第三方结束之后成环，40P01，违反约定五"只会一方等另一方"。方案 E 之下添加先取 acme 的 S，删除工作区的 N 与它互斥，批量不再与它交错。测试：`TestAddingSeveralMembersAndDeletingTheWorkspaceSerialize`（预检的三方排程，改成 E：第三方的加入也持有 acme 的 S；删除工作区完成，三条新的成员关系在它的一个时刻删除，没有 40P01）；`Locks` 不锁工作区时它以 40P01 失败（`E-share-none-M1`）。**说明。**
17. **小的结构调整**（方案 E 带来的，说明）：`ProjectWorkspace` 的查询、存储方法和存储测试从 Task 7 移到 Task 2（它是每个写的第一步），`TestShareProjectAndProjectWorkspace` 拆成 `TestShareProject`（Task 7）和 `TestProjectWorkspace`（Task 2）；旧 plan 在 Task 6 加的 `ProjectDeleter`（只为删除项目的锁）不再需要，删除项目只要 `ProjectsDeleter`；`ProjectUpdater`、`ProjectArchiver`、`MemberGrower`、`PreferencesWriter` 不再嵌入锁的端口，锁在 `ProjectLocks`（`ProjectFinder`、`ProjectLocker`、`ProjectSharer`）；`growthFixture` 并进 `writeFixture`（`members`）；`createdProject` 从 Task 6 提前到 Task 4（第一个用两个项目的组合测试）；工作区目录按 id 的锁的存储测试另起 `directory_share_test.go`（`directory_test.go` 已 396 行）。

## 4. 验收标准（完成线，M3 设计 12 节 P4b）

- [x] P2、P3、P4、P8 的接口版本通过，此前的每个故事仍然通过（`make e2e` 共 62 个：此前的 58 个，加上四个）。
- [x] 恢复时角色的表（`TestJoinRole` 的四行、`TestARestoredMembershipGivesNoMoreThanItHad` 经接口的三行和添加的一行）通过。
- [x] 交错 17（加入、添加各两种顺序，新的和已结束的成员关系）、降级与删除项目、项目级的写与降级、同一个项目上的两个写、同一个工作区里两个项目上的写、删除工作区与项目级的写、添加多个成员与删除工作区，连同每个写先锁工作区的测试，`-count=5 -race` 通过，没有 40P01。
- [x] 每个项目级的写最先取工作区行的 `FOR SHARE`，在自己的事务里、在目标和项目之前（`TestEachWriteOnAProjectSharesItsWorkspaceFirst`，写的行与矩阵中在项目一级写的行对上）；删除工作区以它的一个时刻删除进行中的写写下的行，交错 17 的成员关系最后写入不早于建立（M3 设计 3.3 的一个改变一个时刻）。
- [x] 本 Phase 的矩阵格子（175 个）通过，含添加无效目标的 PM、X 格；共 356 格，耗时记下；完整性核对通过。（执行之后共 390 格，约 1.3 秒，见评审记录第 2 节。）
- [x] 每个项目级的写在事务的连接上（`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`）；增长和删除被拒于提交时一行不留（`TestAGrowthRefusedAtItsCommitLeavesNoRow`、`TestAProjectDeletionRefusedAtItsCommitChangesNoRow`）。
- [x] 删除项目由目录驱动的组合测试核对（`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`），删除工作区的三个组合测试不加豁免照旧通过。
- [x] `project` 的 `apitest.Main` 两个方向通过（新码 `project.archived`）。
- [x] `make lint`、`make test`、`make gen-check`、`make knip`、`make test-web`、`make e2e` 通过。
- [x] 3.20 中 P4b 的一行写好；M1-P3 交接有"处理结果（M3/P4b）"。

## 5. 不在 P4b 范围内

- `EndMemberships`、移出、离开，交错 1、4、5、6：P5。停用：P6。标签、默认之外的状态：P7。`PROBLEM_MESSAGES` 的搬迁：P8。页面：P9–P11。`project.NewCascade`：P6（G2）。

**留给后面的 Phase**（各 Phase 的 spec 接过去；P4b 的 review 第 6 节照录）：

| Phase | 条目 |
|---|---|
| P5 | 移出、离开写好之后，矩阵的 `standIns` 中"以前的成员""被移出的成员"两条替身换成存储，`listsTheProjectMembers` 不再有被移出的成员（第 3 节第 4 条）；`EndMemberships` 改项目成员关系之前取那些项目的 `FOR NO KEY UPDATE`（3.6 的加锁表；`Memberships` 在项目锁下读到的才是写时的），在移出、离开工作区的工作区 N 之后读时刻（3.3）；改项目成员的角色、移出项目成员、离开项目经 P4b 的 `Locks`（工作区 S → 项目 N → 判定，3.6 的加锁表，方案 E，第 3 节第 14 条），它们的操作进矩阵时在 `TestEachWriteOnAProjectSharesItsWorkspaceFirst` 各加一行（完整性核对要求：路径带 `{project_id}`，或矩阵里有某行的列含项目一级的调用者，执行时的修复轮改定）；它们按资源寻址（`/project-members/{id}`），测试的 `projectWrite.path` 要能接资源路径，第一步还要探测写自己的资源行（`FOR UPDATE NOWAIT`），否则先锁资源行的写会通过；交错 4、5 照设计 9.3 的方案 E 写法，第二方等在工作区行上；故事清扫接过"已结束"的谓词：`ListMembers` 的 `is_active`、`RestoreMember` 的 id（重新加入恢复以前的成员行）、`ProjectFacts` 的 `m.is_active`，`Memberships` 的账户（一次添加多个账户）；交错 1、4、5、6 用 P4b 的添加、加入（`growth`）；`refusingCommits(t, pool, table)` 可以按表拒绝提交；一个改变一个时刻（第 7 节）对 `EndMemberships` 同样成立，前提是它的时刻在工作区的锁之后读 |
| P6 | 交错 13、14 另跑恢复以前的项目成员行（`growth` 的 `RestoreMember`）；`project.NewCascade` 随 `nerve users deactivate` 加入（G2）；停用按 id 顺序锁住几个工作区的 N 之后才调 `EndMemberships`，它的时刻在这些锁之后读（3.3；交错 13 照设计 9.3 的方案 E 写法：增长、项目级的写等在工作区行上） |
| P7 | 状态和标签的写经 P4b 的 `Locks`（工作区 S → 项目 N，3.6 的加锁表，方案 E），各在 `TestEachWriteOnAProjectSharesItsWorkspaceFirst` 加一行；标签加进 `deleteProjects`（`app/deletion.go`，最后一步）和 `ProjectsDeleter`；组合的目录测试（`keysTo`）和 e2e 的 `expectProjectDeleted` 读目录，标签表没有准备行、没有被删除时它们失败；手写清单要加上标签：存储测试的 `projectTables`、`seedProject`、`deleteAll`，`failures_test.go` 的取消上下文的步骤，`delete_project_test.go` 的每步失败行，e2e 的 `workspaceTables`（第 2.6 节）；`keysTo` 不排除父表自己，父表引用它下面的表（Plane 的 `projects.default_state_id → states`）时会把父表报成没有键的表，需要时加 `AND tab <> $1::text::regclass` 和一个反例；按资源寻址的写（`/states/{id}`、`/labels/{id}`）在每个写先锁工作区的测试里要能接资源路径，第一步还要探测写自己的资源行；先删除状态或标签再删除项目的故事要让 `expectProjectDeleted` 只数删除之前未删除的行；P4、W3 的故事建一个标签（详见 review 第 6 节） |
| M4 | 方案 E 的范围（执行时整分支评审的 I2，负责人 2026-10-02）：工作项的写是否最先取工作区行的 `FOR SHARE` 由 M4 的设计决定，不默认沿用约定二；取的话要说明工作区一级的写在高频的工作项写入下怎样不被饿死（第 3 节第 13 条的代价）。P2 留下的（P2 review 第 6 节，T11 C2）：物理删除账户时 `workspace_members.member_id` 的 `ON DELETE CASCADE` 不经工作区的锁删除成员关系。方案 E 不改变这个旁路：级联不取工作区锁。它改变的是修法的范围：M4 的清理若先按 id 顺序取账户所在工作区的 `FOR NO KEY UPDATE`，这把锁也挡住这些工作区里每个项目级的写（它们先取工作区的 S），与它们不再交错。不取时，删除账户与进行中的添加、加入可能成环（删除持有账户行、级联等目标的工作区成员行；增长持有那一行的 S，插入成员关系时外键要账户行的 `FOR KEY SHARE`）：M4 要先取工作区锁，或用测试证明不成环 |

## 6. 风险

| 风险 | 应对 |
|---|---|
| 项目级的写与工作区一级的连带在工作区行上相遇（方案 E）；一个新的写绕过它 | 一条路径 `Locks`，写不持自己的 `Authorizer`；每个写先锁工作区由组合测试逐个核对，写的列表与矩阵对上（第 3 节第 14 条）；七个交错测试在真实数据库上两个顺序都跑，锁的强度、顺序、在事务里、接线、先判定后锁的变异都被发现 |
| 同一个工作区里，连续重叠的项目级的写让工作区一级的写（修改、删除工作区，改成员的角色，接受邀请；以后移出、离开、恢复成员、停用）一直等到请求期限（方案 E 的代价，不对称：新的 `FOR SHARE` 越过等待中的 `FOR NO KEY UPDATE`） | 两边都是短事务，每个账户限速，按目前的负载占用远不到一半；到期失败、回滚、可以重试；项目级的写之间 S 与 S 不冲突（`TestWritesOnTwoProjectsOfAWorkspaceDoNotWait`）；M4 的工作项写入不默认沿用约定二（第 3 节第 13 条） |
| 增长与降级的加锁顺序（P4a 的 F-M2） | 交错 17 的探测证明第二方等待时两方都还没有锁项目；增长先锁项目由 `TestEachWriteOnAProjectSharesItsWorkspaceFirst` 发现，降级先锁项目由交错 17 发现（第 3 节第 15 条） |
| 删除项目与删除工作区漏掉对方的表 | 一处 `deleteProjects`；两个父表的组合测试都读目录；`keysTo` 拒绝只经别的表挂在下面的表 |
| 故事看不到的谓词（第 3 节第 6 条） | 存储测试在两个行序下都看得到；互相遮住的由 P4 一起发现（`ProjectWorkspace` 与 `ProjectFacts` 的一对；方案 E 之下锁的两个要与它们三个一起去掉）；P5 的故事清扫接过"已结束"的那些 |
| plan 的最大 Task 接近上限 | 最大的 Task 14 是 1,426 行（Task 2 1,404 行、Task 3 1,373 行），每个都在约 1,500 行以内 |
| 持续集成（ubuntu）上的运行 | 原型和复现都在 macOS 上；合并之后看持续集成，P1–P4a 都是这样合并的 |

## 7. 已知的限制、交接和关闭条件

- **一个改变一个时刻，对连带改写的每一列都成立**（M3 设计 3.3，方案 E，第 3 节第 13 条）：每个项目级的写在工作区行的 `FOR SHARE` 之下写，工作区一级的每个连带（降为访客、删除工作区）持工作区行的 `FOR NO KEY UPDATE`、在它之后读时刻；连带读时刻时，它要改的项目一侧的行上没有进行中的写，所以 `updated_at` 不倒退，`deleted_at`、`updated_at` 不早于 `created_at`。测试：交错 17 的每个子测试（成员关系最后写入不早于建立，增长先时也不早于 gate 放开）、`TestAWorkspacesDeletionWaitsForTheWritesOnItsProjects`（修改：Web 的 `updated_at`、`deleted_at` 是删除的一个时刻，不早于修改的，`updated_by_id` 是删除者；加入：新建的成员关系和显示设置在那个时刻删除，不早于建立）、`TestAddingSeveralMembersAndDeletingTheWorkspaceSerialize`（新的成员关系在删除的一个时刻删除）；去掉写的工作区锁、或把连带的时刻挪到锁之前，它们失败（附录 A）。**P4a review 第 7 节的 F-M3 由 P4b 关闭**（P4b 的 review 照录）；本 spec 原来写在这里的窗口（连带读时刻之后等锁的总时间）和预检的 L1 不再存在。以后的连带照 3.3 在它的工作区锁之后读时刻（P5 的 `EndMemberships`、P6 的停用，第 5 节）。
- **两个只持共享锁的写之间不保证时刻的先后**（执行时整分支评审的 M1）：同一个账户并发改自己在同一个项目的显示设置时，两个写都持项目的 `FOR SHARE`、各自在锁之后读时刻，到 `UPSERT` 里才相遇：后提交的可以带较早的时刻，新建的行可以 `updated_at` 早于 `created_at`。只涉及调用者自己那一行的审计列，留下的值是后提交的；P2 的工作区显示设置同样。要严格的先后时，在读时刻之前锁住调用者自己的成员关系行。
- **矩阵里被移出的成员仍在成员列表里**：他的工作区成员关系由 SQL 替身结束，项目成员关系留着有效；P5 的移出结束两者（第 3 节第 4 条）。
- **`getProjectPreferences` 的读和判定是两次查询**：不开事务（6.7 的读），答案是一时的。

| 交接 | P4b 处理的条目 | 留下的条目 |
|---|---|---|
| M1-P3 项目成员 | 只有"从工作区成员中添加"（`addProjectMembers`）和自己加入（`joinProject`），没有按邮件加人的接口（Task 12、13；"处理结果"在 Task 16） | `joinProject` 的页面改调新接口、守卫的 `project-invitations` 例外删除、`RESTRICTED_URLS` 与后端同源（前端一侧，P8）；本节保持 `open` |
| P4a spec 第 5 节、review 第 6 节的 P4b 一行 | 落点见第 3 节第 2 条 | 第 5 节 |

**M3 设计 13.1 的关闭条件**（P4b 的一行）：M1-P3 项目成员：接口一侧只有"从工作区成员中添加"（`addProjectMembers` 的契约只收工作区成员的 `member_id`），`joinProject` 的前端一侧在 P8。P4b 的条件都有落点，没有放不下的。

## 附录 A：原型验证记录（2026-10-01；方案 E，2026-10-02）

原型在 `$M3TMP/p4bproto`（`e22e5080` 的副本，Go 1.27.1、Node 24、pnpm 11.10.0、Docker；`bin/golangci-lint` 2.13.2）。做法照 P4a：每个 Task 做完时存一份源文件的快照（`$M3TMP/p4bsnap/T1`…`T16`），plan 的代码块由脚本从相邻两份快照的差异生成（`p4btools/mkblocks.py`），生成的文件不进块，按 SHA-256 核对（`gensha.py`）。清扫之后加强的测试由 `amend.py` 从带来它的 Task 起改进每一份快照（第 3 节第 12 条），之后重新生成全部块。

**方案 E 的修订**（2026-10-02）：方案 E 之前的原型和快照留在 `p4bproto-before-E`、`p4bsnap-before-E`（plan `5740002c`）。方案 E 先在原型上做完，再由 `E/rebuild.py` 重建 Task 2–16 的快照：方案 E 改的、只由一个 Task 带来的文件从那个 Task 起取原型的最终版本；几个 Task 都改的文件（`app/ports.go`、`lock.go`、`module.go`、`fakes_write_test.go`、`clock_test.go`、`projects.sql` 和存储测试里 `ProjectWorkspace` 的几处、`project_writes_test.go`、每个写先锁工作区的测试）由 `E/stages.py` 的变换从旧快照的版本做出这个 Task 时的样子；之后 gofmt，在副本里 `make gen-go`。每份重建的快照 `go vet ./...`（编译全部测试）和 `gofmt -l` 干净，`project` 的用例、领域和 `access` 的测试通过（`E/vetall.py`、`E/unitall.py`）。plan 的块、生成物的 SHA-256 和行数从重建的快照重新生成，`assemble.py` 组装 plan 并核对每个 Task 在约 1,500 行以内、文件表与块一致。下面的数字都是方案 E 之后的原型和 plan 的；方案 E 之前的 plan 同样逐 Task 复现过（739 秒）。

**逐 Task 复现**（`$M3TMP/p4btools/replay.py`、`replay_all.sh`，日志在 `replay-logs`）：在 `$M3TMP/p4breplay`（`e22e5080` 的另一个副本，`pnpm install --frozen-lockfile`）按 plan 的 16 个 Task 依次执行：`planapply.mjs` 从 plan 的文本中取出这个 Task 的块写入，然后按顺序执行这个 Task 的每一条 `Run:` 命令，原样照 plan。例外只有：`make gen`、`make gen-go` 之后核对全部生成物与这个 Task 的快照逐字节相同；`shasum -a 256` 的输出与 plan 表中的 SHA-256 和行数核对；副本不是 git 仓库时 `make lint-web` 的关键词守卫由 `kwcheck.mjs` 代替（同样的规则、同样的文件）；`make e2e` 之前把副本初始化为 git 仓库并提交（F4）；提交之后的 `make gen-check` 由生成物的核对代替。每个 Task 另核对 `go.mod`、`go.sum`、`server/tools` 和 `pnpm-lock.yaml` 不变。整个复现 758 秒，全部通过。

| Task | 复现的结果（Task 1–15 每个都有 `make lint-go` 两个 `0 issues.`、`make test` 42 个 `ok`；Task 16 只改文档） |
|---|---|
| 1 | 3 个文件；`bootstrap` 的 4 个测试 ok |
| 2 | 19 个文件；`make gen-go`：32 个生成物与快照相同；3 个 SHA-256 和行数相同；`project/...`、`workspace/...` ok；`bootstrap` 的 1 个测试 ok |
| 3 | 20 个文件；`make gen`：32 个生成物与快照相同；4 个 SHA-256 和行数相同；`project/...`、`access/...` ok；`bootstrap` 的 9 个测试 ok；前端检查通过 |
| 4 | 24 个文件；`make gen`：32 个生成物与快照相同；4 个 SHA-256 和行数相同；`project/...`、`access/...` ok；`bootstrap` 的 9 个测试 ok；前端检查通过 |
| 5 | 11 个文件；`make gen-go`：32 个生成物与快照相同；1 个 SHA-256 和行数相同；`project/...` ok；`bootstrap` 的 5 个测试 ok |
| 6 | 15 个文件；`make gen`：32 个生成物与快照相同；3 个 SHA-256 和行数相同；`project/...`、`access/...` ok；`bootstrap` 的 10 个测试 ok；前端检查通过 |
| 7 | 19 个文件；`make gen-go`：32 个生成物与快照相同；2 个 SHA-256 和行数相同；`project/...`、`access/...` ok；`bootstrap` 的 2 个测试 ok |
| 8 | 11 个文件；`make gen`：32 个生成物与快照相同；4 个 SHA-256 和行数相同；`project/...` ok；`bootstrap` 的 9 个测试 ok；前端检查通过 |
| 9 | 21 个文件；`make gen`：32 个生成物与快照相同；4 个 SHA-256 和行数相同；`project/...`、`access/...` ok；`bootstrap` 的 7 个测试 ok；前端检查通过 |
| 10 | 10 个文件；`make gen-go`：32 个生成物与快照相同；2 个 SHA-256 和行数相同；`project/...` ok |
| 11 | 15 个文件；`project/...`、`access/...` ok；`bootstrap` 的 2 个测试 ok |
| 12 | 13 个文件；`make gen`：32 个生成物与快照相同；4 个 SHA-256 和行数相同；`project/...` ok；`bootstrap` 的 11 个测试 ok；前端检查通过 |
| 13 | 20 个文件；`make gen`：32 个生成物与快照相同；3 个 SHA-256 和行数相同；`project/...`、`access/...` ok；`bootstrap` 的 9 个测试 ok；前端检查通过 |
| 14 | 8 个文件；`project/adapter/postgres` 的 1 个测试 ok；`bootstrap` 的 5 个测试 ok；`bootstrap` 的 8 个测试 `-count=5 -race` ok（25 秒，0 个 40P01） |
| 15 | 8 个文件；前端检查通过；`make e2e` 62 个全部通过 |
| 16 | 2 个文件；前端检查通过 |
| 结束 | 复现的树与原型逐文件相同（2,993 个文件，0 个差异）；副本上 `make gen-check` 通过（生成物在 Task 15 的提交里，Task 16 只改文档）；`planapply.mjs check` 从 `e22e5080` 起 393 个块全部通过 |

| 核对 | 命令 | 结果 |
|---|---|---|
| 生成物 | 原型上 `make gen`，生成物与运行之前逐字节比较；复现的副本上 `make gen-check` | 没有差异 |
| Go 静态检查 | `make lint-go` | server 0 issues；server/tools 0 issues |
| Go 测试 | `make test`（testcontainers，`postgres:18.6`） | 全部通过（41 个包加 `server/tools` 的 1 个，与 P4a 相同；P4b 不加包） |
| 前端检查 | `make lint-web`（关键词守卫：3,023 个文件、5 个命中、都有例外，P4a 结束时同样 5 个；turbo 54 个任务）；`make knip`；`make test-web`（16 个任务） | 全部通过 |
| 端到端 | `make e2e` | 62 个：原型不是 git 仓库时 S3 失败（F4），其余 61 个通过；复现的副本 62 个全部通过 |
| 矩阵 | `go test -count=1 -v -run 'TestPermissionMatrix$'` | 356 格（本 Phase 175 格），整个矩阵 1.42 秒，准备 0.11 秒（原型）；9.2 的预算是 20–30 秒 |
| 交错 | `go test -count=5 -race -run` 2.12 的七个交错测试和 `TestEachWriteOnAProjectSharesItsWorkspaceFirst` `./internal/bootstrap/` | 40 个顶层 PASS（100 个子测试），0 个 FAIL，输出中 0 个 40P01、0 个 DATA RACE，25 秒 |
| 预检的死锁复现 | 预检的 `deadlock-demo-3.sh` 的排程换成方案 E 的语句（`deadlock-demo-3E.sh`，在 `nerve-dev-db-1` 的临时库上，用完删除） | 方案 E：三方都提交，没有 40P01，删除工作区在添加和第三方之后；`MODE=noshare`（预检原来的语句）：添加报 40P01 |
| 变异 | `mutants_s1`–`s6`、`s8`、`s9`、`s10_s11`、`s12.py`（Go，270 个）、预检的 `mutants_pf.py`、`mutants_pf2.py`（24 个）、`mutants_E.py`（36 个）、`mutants_split.py`（10 个，其中 1 个是新的）、`mutants_reanchored.py`（74 个，都是上面的，照最终的代码重新锚定）；`E/e2e_sweep1_E.py`（43 个、3 对、方案 E 的 3 个，每个只运行用到它的故事）、`E/e2e_mutants_E.py`（15 个，单独运行一个故事） | 见下 |

**原型中定下的事实**：

- **F1** 等锁之后 Postgres 在行的最新版本上重新求值 `WHERE` 的条件：`LockProject`、`ShareProject`、`LockMemberProjects` 等待期间被删除的项目读不到（`TestTheProjectLockSeesADeletionItWaitedFor`、`TestLockMemberProjectsLeavesOutAProjectDeletedWhileItWaited`）。
- **F2** `FOR NO KEY UPDATE` 与 `FOR SHARE`、另一个 `FOR NO KEY UPDATE` 冲突，与外键检查的 `FOR KEY SHARE` 不冲突；两个 `FOR SHARE` 不冲突（`TestLockProject`、`TestShareProject` 用 `NOWAIT` 核对，工作区按 id 的锁由 `TestTheDirectorysLockIsForShare/byID` 核对）：改设置之间不互等，修改与改设置互等；同一个工作区里项目级的写之间不互等，它们与工作区一级的写互等，但不对称：新的 `FOR SHARE` 越过等待中的 `FOR NO KEY UPDATE`（第 3 节第 13 条）。
- **F3** `ON CONFLICT (project_id, user_id)` 不带部分唯一键的条件时，Postgres 推断不出那个部分唯一索引，每次插入都报错：`ups-target`、`ens-target` 因此连故事都发现（`EnsurePreferences` 在每次添加、加入里）。
- **F4** 原型不是 git 仓库时 S3（实例信息里的 `commit`）失败（P3 附录 A 的 F8）；复现在 `make e2e` 之前把副本初始化为仓库并提交。
- **F5** 一条语句锁几行时逐行等待（`LockMemberProjects` 按 id 顺序，`DeleteProjects` 按扫描的顺序）：方案 E 之前，连带读了时刻之后可以逐个等几个项目级的写（P4a 的 F-M3 的窗口），删除工作区逐行改成员行时与添加逐行锁的目标成环（预检的 M1）；方案 E 之下这些写都先等在工作区行上（第 7 节、第 3 节第 16 条）。
- **F6** 只有一个连接的池上，经池而不是经事务发出的语句等第二个连接，直到请求的期限：这让"在事务之外"的每个变异都表现为写失败，不依赖时序（`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`）。
- **F7** 一行已被 `FOR SHARE` 持有、另一个事务的 `FOR NO KEY UPDATE` 正在等它时，新来的 `FOR SHARE` 立即被授予，不排在等待者之后（与持有者相容）：M1 的排程里，删除工作区等 acme 时添加照样取得 acme 的 S 并答 201（`TestAddingSeveralMembersAndDeletingTheWorkspaceSerialize`）。所以方案 E 之下等待中的连带可以被后来的项目级的写推迟，但不成环：后来者都只取 S，最后一个持有者结束时连带得到 N。
- **F8** 死锁检查在一次等待开始 `deadlock_timeout`（1 秒）之后由等待的后端做一次：M1 的环在第三方结束、删除再等添加时才合上，添加那一次检查已经过去，所以去掉工作区的锁时是删除工作区报 40P01（`E-share-none-M1`）；测试在两个后端都等时先等 1.2 秒，让报错的一方确定。

**清扫**（brief 的十一类，穷尽地跑在 P4b 新加或修改的代码上；每个变异的类别和必须失败的测试写在 `mutants_*.py` 的每一条上；"层"是必须失败的测试所在的层）：

| 清扫 | 规模 | 结果和层 |
|---|---|---|
| 1 每个 SQL 谓词 | 新加或修改的 15 条查询（`LockProject`、`ShareProject`、`ProjectWorkspace`、`UpdateProject`、`SetArchived`、`Memberships`、`ListMembers`、`RestoreMember`、`Preferences`、`UpsertPreferences` 和 `EnsurePreferences` 的冲突目标、删除的四条）的每个 `WHERE` 条件各去掉一个：41 个 Go 变异；第一行决定答案的（`:one`、`Memberships` 的映射）按 id 升序、降序两个行序各跑一次 | 41 个全部被存储测试发现，两个行序都是。故事一半（P2、P3、P4、P8、W3 各自单独运行，另加 `ProjectFacts` 的两个"已删除"谓词）：43 个中 29 个被故事发现，14 个看不到（第 3 节第 6 条）；`ProjectWorkspace` 与 `ProjectFacts` 的一对由 P4 发现，锁的两对在方案 E 之下要连同 `ProjectWorkspace` 三个一起去掉 P4 才发现（`triple-lp`、`triple-sp`）。方案 E 加的工作区按 id 的锁：id（两序）、`deleted_at IS NULL` 3 个（`mutants_E.py`），被存储测试发现，故事看不到（第 3 节第 15 条） |
| 2 每个端口调用的错误 | 用例对端口（存储的读和写、项目的锁、跨模块的 `ShareMembers`、`Authorizer`、事务的提交）的每个调用失败被吞掉、被重试、被答成别的问题；每个 handler 吞掉用例的失败：63 个 | 全部被发现：54 个在用例一层（调用记录比较到失败的那一次为止，之后的调用不能有），9 个在 HTTP 一层 |
| 3 每个写不动的行和列 | 每个存储的写（`UpdateProject`、`SetArchived`、四条删除、`UpsertPreferences`、`RestoreMember`、`EnsurePreferences`）多写一列、少写一列、不看标志、保留不该保留的：30 个 | 全部被存储测试发现；每个测试都有别的项目（同工作区、已归档、别的工作区）、别的成员、别的账户的行，核对每一行的每一列 |
| 4 组合根的每个接线 | `project.New` 给七个写的事务管理器换成不开事务的、时钟换成固定在 2000 年的（各 7 个），给九个用例的 `Authorizer`（七个写经 `Locks`）换成谁都当作管理员放行的（9 个），给添加、加入的 `Locks` 的成员端口换成什么都不锁、答谁都是工作区成员的（2 个），归档和恢复接反（1 个）：26 个；方案 E 的接线 3 个见"方案 E"一行 | 全部被组合出的测试发现（`TestPermissionMatrix`、`TestTheWritesOnAProjectStampTheirRequest`、`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`、`TestARestoredMembershipGivesNoMoreThanItHad`、三个交错测试、两个提交被拒的测试） |
| 5 每个安全性质在真实的系统上 | 八条规则各放宽或收紧、看不到的项目答 403（17 个）；P4b 自己的类别（加入的两条规则在其中，下表）36 个 | 17 个全部被矩阵发现；P4b 自己的类别见下表，只在单元一层的见第 3 节第 7 条 |
| 6 每一句说明 | 契约九个操作的每段描述、P4b 的代码注释和查询注释、差异清单和交接的每一行，逐句对照代码或测试（下面的对照）；3 个契约变异（`updateProject` 不声明 `project.archived`、`addProjectMembers` 声明 200、`joinProject` 不声明 `forbidden`） | 3 个变异被矩阵（答案按契约核对）和 HTTP 测试发现；逐句的对照没有发现不符 |
| 7 反例里没有随机 | P4b 新加、修改的测试和故事里没有 `math/rand`、`crypto/rand`、`Math.random`；id 由 `uuid.NewV7()` 生成，行序决定结果的存储测试按固定的顺序写入（例如 `TestMemberships`：carol 已删除的成员关系先于她已结束的存入），清扫 1 另按两个行序各跑一次；`time.Now()` 只给准备的行盖时刻，断言比较的是写入的时刻本身或请求前后的区间 | 没有发现 |
| 8 写的判定读在调用者的事务里 | 写的事务里的每个读和写（项目的锁、`ShareProject`、`ProjectWorkspace`、`Memberships`、`ProjectFacts`、`ActiveRole`、`ShareMembers`、`LowestSortOrder`、`ListMembers`、`GetProject`、每个写）各改为经连接池：20 个；工作区按 id 的锁经池、`bootstrap` 在事务之外问目录（`E-on-pool`、`E-wire-outside-tx`）2 个见"方案 E"一行 | 全部被 `TestTheWritesOnAProjectRunOnTheirTransactionsConnection` 发现（`ProjectFacts` 另有 P4a 的存储测试） |
| 9 每个加锁顺序在真实的系统上 | 锁的强度（`LockProject`、`ShareProject`、`LockMemberProjects`）、先判定后锁（修改、归档）、项目锁在事务之外、增长先锁项目（添加、加入）、加入不锁他的工作区成员行、降级先锁项目（F-M2）、`LockMemberProjects` 留下等锁时删除的项目：14 个 | 全部被发现（方案 E 之后的层，第 3 节第 15 条）：2 个只在组合一层（降级先锁项目、项目锁在事务之外），5 个在组合和单元两层（增长先锁项目、加入不锁他的成员关系、先判定后锁），4 个在组合和存储两层（`ShareProject` 取 `FOR KEY SHARE`、不加锁，`LockProject` 取 `FOR SHARE`、不加锁），3 个只在存储一层（`LockProject` 取 `FOR UPDATE`；`LockMemberProjects` 取 `FOR SHARE`、留下等锁时删除的项目，第 3 节第 7、15 条） |
| 10 承重的准备行都有前提 | 矩阵的准备（`targets`、`preconditions`）和交错测试的准备中，每一行改成会让变异通过的状态：11 个 | 全部被前提或测试发现（组合） |
| 11 每条拒绝的路径 | 没有调用者时不读（每个用例一个，8 个）；负责人在判定之前检查（1 个）：9 个；添加的目标在判定之前检查（`a-targets-first`）在 P4b 自己的类别里，矩阵的四个变体行给 PM、X 等格 | 全部被发现：8 个在用例一层（第 3 节第 7 条），1 个在矩阵 |
| 方案 E（第 3 节第 13–16 条） | `mutants_E.py` 36 个：一条路径（不锁工作区，另在 M1 的排程上；工作区在目标之后、在项目之后；目标在项目之后；锁的错误被吞；等待期间删除的工作区答成找到；项目锁读到的工作区不核对）8 个，七个写各自不锁工作区 7 个，查询（`FOR KEY SHARE`、`FOR NO KEY UPDATE`、`FOR UPDATE`、不加锁、去掉 `deleted_at IS NULL`、去掉 id 的两个行序、经池）8 个，接线（`bootstrap` 不问目录、在事务之外问；`project.New` 的 `Locks` 接在不加锁的工作区端口上）3 个，连带在锁之前读时刻 2 个，测试变异（每个探测改指方案 E 之前或之外的表、完整性核对少一行）8 个；`mutants_split.py` 10 个（`ProjectWorkspace` 移到 Task 2、`TestShareProjectAndProjectWorkspace` 拆开之后的清扫 1 的 6 个和 `ShareProject` 的 3 个、`ProjectWorkspace` 加锁 1 个）；预检的 `mutants_pf.py`、`mutants_pf2.py` 24 个 | `mutants_E.py` 全部被发现：30 个在组合一层（其中 4 个另在单元一层、5 个另在存储一层），3 个只在存储一层（`deleted_at`、id 的两个行序），3 个只在单元一层（错误被吞、等待期间删除、工作区不核对）；`mutants_split.py` 全部被发现，6 个只在存储一层，4 个另在组合一层；预检的 17 个在每一层被发现，3 个只在低一层（`pf-list-order-id` 存储、`pf-update-clock-before-lock` 单元、`pf-add-duplicate-unchecked` 领域，预检已按它们的类别接受；执行时 T11-b、T12 把后者升到用例和组合一层，见第 3 节第 9 条），`tm-gate-before-share`、`tm-webfree-dropped` 和 `pf-join-unlocked-workspace-2` 是等价的（第 3 节第 10、15 条），`pf-join-unlocked-workspace` 由 `pf2` 的写法代替 |
| 写 spec 时补的 | 修改的领域规则（`archive_in` 两端、标识不转大写、标识和名称不按规则查）、两个唯一键的 409 互换、显示设置的两条规则、项目的锁和 `ShareProject` 找不到已归档的项目、状态在另一个时刻删除：11 个（`mutants_s12.py`，第 3 节第 12 条） | 全部被发现；其中 6 个另各让它的故事单独运行（`e2e_mutants.py`）失败 |

**按缺陷类别**（brief 的 P4b 类别；"层"是变异被发现的层）：

| 类别 | 变异 | 被哪些测试发现 | 层 |
|---|---|---|---|
| 恢复时的角色（9.1，I1） | `JoinRole` 取较高的、取工作区角色、保留已结束的角色；添加不取请求的角色 | `TestJoinRole`、`TestARestoredMembershipGivesNoMoreThanItHad`；后者另有 P3 | 组合；端到端 |
| 加入 | 工作区访客加入公开项目；成员加入私密项目；`CanJoin` 在"已是有效成员"之后（G4）；位置不是 65535；不锁他的工作区成员行；规则只给项目成员；`CanJoin` 按大小 | `TestPermissionMatrix`（PG、WM-私 等格）、P2；`TestARestoredMembershipGivesNoMoreThanItHad`；交错 17、`TestJoinProject`；`TestCanJoin`（按大小，第 3 节第 7 条） | 组合；端到端；单元 |
| 添加 | 不是工作区有效成员的目标；访客作成员、管理员作成员；目标在判定之前；已是有效成员不答 `duplicate`；部分写入；位置不按 3.18 | `TestPermissionMatrix`（四个变体行）；`TestAddProjectMembers`、P3 | 组合；单元；端到端 |
| 修改 | 负责人、默认负责人是项目访客、不是成员、已结束的成员；不检查；`archive_in` 越界；已归档的照改；标识的规则与建项目不同；两个唯一键的 409 互换 | `TestPermissionMatrix`、`TestTheLeadAndTheDefaultAssigneeAreActiveMembersWhoAreNoGuests`、P3、P4；`TestCheckProjectPatchReportsEveryField`、`TestThePatchChecksAsCreateDoes`；`TestUpdateProjectIdentifierOrNameTaken` | 组合；端到端；单元；存储 |
| 归档、恢复、删除 | 项目成员、不是成员的工作区管理员能做（六个规则变异）；删除别的工作区；一张表的行留着、在另一个时刻、由另一个账户删除；已归档的项目找不到 | `TestPermissionMatrix`；`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`；`TestDeletingAProjectSoftDeletesItsRowsAlone`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`、P4；`TestLockProject`、`TestShareProject`、矩阵的已归档列 | 组合；存储；端到端 |
| 显示设置 | 读、改别人的；未知的标签页、藏起 work_items；看不到的项目（规则给看得到的每个人、不给访客） | `TestEachMemberHasHisOwnDisplaySettings`、P8；`TestCheckPreferencesPatchReportsEveryTab`、P8；`TestPermissionMatrix` | 组合；端到端；单元 |
| 成员列表 | 列已结束、已删除的；别的项目的；看不到的人得到 404 以外的 | `TestListMembers`；P3；`TestPermissionMatrix` | 存储；端到端；组合 |
| 交错 17 | 添加、加入各对降级，两个顺序；探测（`WaitForLockWaitOn` 指名 `workspaces`，`webFree` 的 `NOWAIT`）；他的成员关系最后写入不早于建立（L1）；没有挂起：每次等待有期限 | 2.12 的表 | 组合 |
| 父锁、强度、事务 | `Authorize` 在父锁之前；父锁没有 `deleted_at IS NULL`；强度错；锁在事务之外 | 清扫 8、9；`TestLockProject`、`TestTheProjectLockSeesADeletionItWaitedFor`、`TestShareProject`、`TestProjectWorkspace`；`TestTwoWritesOnAProjectSerialize` | 组合；存储 |
| 工作区的锁（方案 E） | 一个写或全部写不锁工作区；锁在目标或项目之后；`FOR KEY SHARE`、`FOR NO KEY UPDATE`、`FOR UPDATE`、不加锁；经池；没有 `deleted_at IS NULL`；接线是空的、在事务之外；连带在锁之前读时刻；探测指名方案 E 之前的表 | `TestEachWriteOnAProjectSharesItsWorkspaceFirst`、`TestWritesOnTwoProjectsOfAWorkspaceDoNotWait`、`TestAWorkspacesDeletionWaitsForTheWritesOnItsProjects`、`TestAddingSeveralMembersAndDeletingTheWorkspaceSerialize`、交错 17、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`、`TestProjectWorkspacesConvertsWorkspacesAnswer`；`TestTheDirectorysLockIsForShare/byID`、`TestShareWorkspaceByIDFindsTheUndeletedWorkspace`、`TestTheDirectorysLockSeesADeletionItWaitedFor/byID`；`TestUpdateProjectRefuses` 等用例测试 | 组合；存储；单元（第 3 节第 15 条） |
| 规则表加行而没有格子、放宽 | 九条规则各放宽、收紧；每条新规则带它的格子 | `TestPermissionMatrix`；`TestEveryRuleDecidesItsCells` 要求每条规则的格子 | 组合；单元 |
| 精确的错误 | 看不到答 403；`Authorizer` 的失败答 403；加入时找工作区的失败答 404 | `TestPermissionMatrix`；各用例的 `ReturnsEachFailure`（`errors.Is` 比较具体的错误；写完读不到是内部错误，不是 404） | 组合；单元 |
| 码只在别的模块的测试包里返回（9.4） | 没有新变异：`project` 的 HTTP 测试回答 `project.archived` 和每个新操作声明的码，`apitest.Main` 两个方向通过；契约少声明一个码的 3 个变异见清扫 6 | `apitest.Main` | 单元 |

**故事看得到的谓词**（`E/e2e_sweep1_E.py`、`E/e2e_triples_E.py`，在最终的原型上；每次只运行用到这条查询的故事；"两序"是按 id 升序、降序各跑一次）：

| 查询 | 谓词 | 发现它的故事 |
|---|---|---|
| `LockProject` | `id`（两序） | 升序 P2、P3、P4、P8、W3；降序 P2、P3、P4 |
| | `deleted_at IS NULL` | 看不到（与 `ProjectWorkspace`、`ProjectFacts` 的一起去掉时 P4；只与 `ProjectFacts` 的一起去掉时由 `ProjectWorkspace` 遮住） |
| `ShareProject` | `id`（两序） | P8 |
| | `deleted_at IS NULL` | 看不到（同上） |
| `ProjectWorkspace` | `id`（两序） | 两序都是 P2、P3、P4、P8（方案 E 之前降序只有 P3、P8：每个写现在先读它） |
| | `deleted_at IS NULL` | 看不到（与 `ProjectFacts.p.deleted_at` 一起去掉时 P4） |
| `ShareWorkspaceByID`（方案 E） | `id`（两序）；`deleted_at IS NULL` | 看不到（第 3 节第 15 条） |
| `UpdateProject`、`SetArchived` | `id` | P3；P4 |
| `Memberships` | `project_id`（两序） | P3 |
| | `member_id`（两序）；`deleted_at IS NULL`（两序） | 看不到 |
| `ListMembers` | `project_id` | P3 |
| | `is_active`；`deleted_at IS NULL` | 看不到 |
| `RestoreMember` | `id` | 看不到 |
| `Preferences` | `project_id`、`user_id`（两序） | P8 |
| | `deleted_at IS NULL`（两序） | 看不到 |
| `UpsertPreferences`、`EnsurePreferences` | 冲突目标的条件 | P8；P2、P3、P4、P8 |
| 删除的四条语句 | 项目 × 4 | P4、W3 |
| | 工作区 × 4；`deleted_at IS NULL` × 4 | W3 |
| `ProjectFacts`（P4a 的） | `m.deleted_at`；`p.deleted_at` | 看不到（后者与 `ProjectWorkspace` 的一起去掉时 P4） |

**清扫 6 的对照**（契约的每句说明对照的测试；代码注释和查询注释照同样的做法逐句对照）：

| 说明 | 守着它的测试 |
|---|---|
| 修改：给出的字段改、别的不变；值照建项目的规则；`archive_in` 0–12；在看项目之前检查 | `TestUpdateProject`（存储、用例）、`TestThePatchChecksAsCreateDoes`、`TestCheckProjectPatchReportsEveryField`、`TestUpdateProjectRefuses` |
| 已归档的不能改（`project.archived`） | 矩阵的已归档列、`TestUpdateProjectRefuses`、P4 |
| 负责人、默认负责人可为 `null`，须是不是访客的有效成员，在调用者的角色之后检查 | `TestUpdateProjectPassesThePatch`、`TestTheLeadAndTheDefaultAssigneeAreActiveMembersWhoAreNoGuests`、矩阵的"负责人不是成员"一行 |
| 名称、标识不能是工作区另一个未删除项目的 | `TestUpdateProjectIdentifierOrNameTaken` |
| 角色在锁住项目行之后判定，期间被降级的人被拒 | `TestAProjectWriteAndADemotionSerialize` |
| 归档：从请求的时刻起归档，`listProjects` 列在已归档的里面，恢复之前不能改；再归档取新的时刻 | `TestSetArchived`、`TestArchiveProject`、矩阵的 `archivesItsProject`、`TestTheWritesOnAProjectStampTheirRequest`、P4 |
| 恢复：`archived_at` 为空，可以再改；未归档的照旧 | `TestSetArchived`、`TestArchiveProject`、P4 |
| 删除：已归档的照样删除；成员关系、显示设置、状态在同一时刻；不再被读、列出、修改；名称和标识在工作区里可以再用 | 矩阵的已归档列、`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`、`TestDeletingAProjectSoftDeletesItsRowsAlone`、P4、`TestUpdateProjectIdentifierOrNameTaken` |
| 显示设置：只是自己的；没有时是默认值，读不写；导航整个替换，第一次修改以默认值加上修改存下；未知的标签页、藏起 work_items、藏两次在看项目之前拒绝；已归档的照改 | `TestEachMemberHasHisOwnDisplaySettings`、`TestGetProjectPreferences`、`TestUpsertPreferences`、`TestCheckPreferencesPatchReportsEveryTab`、`TestUpdateProjectPreferencesRefuses`、矩阵的已归档列 |
| 成员列表：有效成员，带角色，按成为成员的顺序 | `TestListMembers`、矩阵的 `listsTheProjectMembers` |
| 添加：1–100 个工作区有效成员，角色按他的工作区角色；以前的成员恢复，取给的角色；显示设置把项目放在第一位，已有的不动；按位置拒绝；在调用者的角色之后检查；已归档的照样添加 | `TestCheckNewMembersReportsEveryProblem`、`TestCanAdd`、`TestCheckTargets`、`TestAddProjectMembers`、`TestARestoredMembershipGivesNoMoreThanItHad`、`TestEnsurePreferences`、矩阵的四个变体行和已归档列 |
| 加入：工作区的管理员、成员，看得到的；工作区角色；以前的成员取较低的；已是有效成员不变；已归档的照样加入；工作区访客 `forbidden` | `TestJoinProject`（用例）、`TestJoinRole`、`TestJoinProjectLeavesAnActiveMemberAsHeIs`、`TestJoinProjectJoinsAnArchivedProject`、`TestJoinProjectRefuses`、矩阵的 `joinsAs` 和 PG 格、P2 |

**变异核对**（`mutrun.py`：Go 文件经 `go test -overlay` 改，契约的打包文件就地改并恢复；生成的查询常量代表它的查询；期望点名的测试失败并且输出含它的失败行；编译不过不算）。方案 E 之后全部 Go 变异在最终的原型上再跑（`E/final_mutants.sh`，一组接一组）；代码被方案 E 移动、替换的文本对不上的记为 STALE、跳过，它们和发现它的测试改变了的在 `mutants_reanchored.py` 中照最终的代码重新锚定、以同样的 id 再跑；`TestShareProjectAndProjectWorkspace` 拆开之后，点名它的在 `mutants_split.py` 中以同样的 id 再跑，`s1`、`s3`、`s12`、`E` 和重新锚定的也再跑一次（`E/after_split_mutants.sh`）。每个变异的结论取它在最终原型上最后一次不是 STALE 的运行（`E/tally.py`）：

| 集合 | 个数 | 结论 |
|---|---|---|
| `s1` | 41 | 全部被发现（35 个第二次运行，6 个经 `mutants_split.py`） |
| `s2` | 63 | 全部（18 个重新锚定） |
| `s3` | 30 | 全部 |
| `s4` | 26 | 全部（24 个重新锚定） |
| `s5` | 42 | 全部（5 个重新锚定） |
| `s6` | 5 | 全部 |
| `s8` | 20 | 全部 |
| `s9` | 14 | 全部（9 个重新锚定，2 个经 `mutants_split.py`） |
| `s10_s11` | 18 | 全部（7 个重新锚定） |
| `s12` | 11 | 全部（1 个经 `mutants_split.py`） |
| 预检的 `pf`、`pf2` | 24 | 17 个在点名的每一层被发现；3 个只在低一层（预检已接受的类别）；3 个等价；1 个由 `pf2` 的写法代替（清扫表"方案 E"一行） |
| `E` | 36 | 全部 |
| `mutants_split.py` 自己的 | 1 | 被发现（`E-finder-locks`） |
| 共 | 331 | 324 个被发现，3 个只在低一层，3 个等价，1 个代替 |

故事一层（在最终的原型上重跑，`E/e2e_final.sh`）：`E/e2e_sweep1_E.py` 43 个中 29 个被故事发现（与方案 E 之前相同；`ProjectWorkspace` 的 id 降序现在由 P2、P3、P4、P8 发现），三对中 `pair-pw` 被发现、`pair-lp`、`pair-sp` 存活，`E/e2e_triples_E.py` 的两个三者一起去掉的被 P4 发现，方案 E 的 3 个看不到；`E/e2e_mutants_E.py` 15 个（`s5` 的 9 个，其中 2 个用重新锚定的写法，`s12` 的 6 个）全部让它的故事失败。每个 Go 变异按带来它所改代码的 Task 列进 plan 的变异表（`E/mutmap_E.py`，在重建的快照上；改 P4a 代码的按它的测试所在的 Task）：Task 2 50 个、3 40 个、4 19 个、5 31 个、6 9 个、7 40 个、8 19 个、9 13 个、10 17 个、11 32 个、12 14 个、13 21 个、14 21 个，P4a 的代码而测试也在 P4a 的 4 个。

**没有证明的**：

- 持续集成（ubuntu）上的运行：原型和复现都在 macOS 上。
- 第 3 节第 6 条的故事看不到的谓词：由存储测试证明，"已结束"的故事一侧留给 P5。
- 删除账户（M4）与进行中的添加、加入：级联不取工作区的锁，可能成环（第 5 节 M4 一行），这里没有测试。
- 等待中的连带被源源不断的项目级的写推迟（F7）：只会推迟、不成环，两边都是短事务，没有长时间的测试。
- 页面版本（P10）；交错 1、4、5、6（P5）。
