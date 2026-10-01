# M3/P4b 项目的管理、显示设置与成员的加入：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M3/P4b `project-members` |
| 日期 | 2026-10-01 |
| 状态 | 进行中：第 3 节各条待控制者裁定 |
| 上级文档 | [M3 设计文档](../M3-design.md) 第 2（P2、P3、P4、P8）、3.3、3.4、3.5、3.6（约定二、三、五、六，加锁表）、3.12、3.18、3.19、3.20（P4b 一行）、4.11（P4b 各行）、5.1–5.3、6.4、6.7、9.1（恢复时的角色）、9.2、9.3（交错 17）、9.6、12（P4b 与约束 1–4）、13.1 节 |
| 前置交接 | [P4a spec](P4a-projects.md) 第 5 节 P4b 一行，[P4a review](../reviews/P4a-projects-review.md) 第 6 节（落点见第 3 节第 2 条）；[M1-P3](../handoffs/M1-P3-trim-platform.md) 的项目成员 |
| 裁定 | 负责人（2026-10-01）：保持设计 3.3 的"一个连带一个时刻"（P4a 的 F-M3），本 spec 在第 7 节写明它和窗口，不改测试；第 3 节第 13 条报告窗口比"一次锁等待"宽的一面，请确认 |
| 计划 | [P4b plan](../plans/P4b-project-members.md) |

本 spec 只写 M3 设计交给 P4b 决定的东西：名字、签名、SQL、测试名，以及原型证明了什么。规则本身以 M3 设计为准，这里引用节号，不重述。P4b 依赖 P4a（`e22e5080` 的 `main`）。

P4b 是评审敏感的一段（M3 设计 12 节约束 3）：项目一侧的成员关系增长（约定三、六的添加和加入，恢复时的角色，目标的 422 在判定之后）与降为访客的连带在交错 17 相遇；谁能修改、归档、删除、看成员、添加、加入项目是安全性质。附录 A 的十一类清扫逐个核对每个这样的测试在它所说的性质去掉之后失败，并记下失败在哪一层（单元、存储、组合、端到端）。

## 1. 目标

按 M3 设计 12 节 P4b：项目的修改、删除、归档、恢复，每个成员自己的项目显示设置，项目成员的列出、添加、加入；添加和加入这两条项目一侧的增长与降为访客串行（交错 17）。具体是：

- 九个操作：`updateProject`、`archiveProject`、`unarchiveProject`、`deleteProject`、`getProjectPreferences`、`updateProjectPreferences`、`listProjectMembers`、`addProjectMembers`、`joinProject`，各带操作名、规则行、矩阵行（共 175 格，含添加无效目标的 PM、X 格）；
- 写的共同头两步：锁项目（`FOR NO KEY UPDATE`，改显示设置是 `FOR SHARE`），然后在锁下、在事务的连接上判定；锁之后读时钟；
- 删除项目与删除工作区共用一处删除步骤（`deleteProjects`），由目录驱动的组合测试核对每张项目之下的表；
- 增长的一步（`growth`）由添加、加入共用：恢复以前的成员行时添加取请求的角色，加入取原来那一行与工作区角色中较低的一个（9.1 的表）；
- 交错 17（添加、加入各两个顺序），降级与删除项目，项目级的写与降级，全部在真实数据库上、经 `bootstrap` 组合出的模块；每个写的每条语句在事务的连接上（只有一个连接的池）；
- 矩阵准备的拆分、账户的登记；故事 P2、P3、P4、P8 的接口版本，W3 先删除一个项目；
- 3.20 中 P4b 的一行（差异清单），M1-P3 交接的"处理结果（M3/P4b）"。

## 2. 交付物

### 2.1 文件总览

路径相对于仓库根目录。"生成"表示由 `make gen`（或 `make gen-go`）生成并提交，不手改。"Task"是 plan 中负责它的任务（plan 有完整的文件表）。本 Phase 93 个手写的文件、8 个生成物；没有迁移，不加跨模块端口。

| 路径 | 内容 | Task |
|---|---|---|
| `bootstrap/permission_matrix_seed_test.go`、`permission_matrix_seeded_test.go`、`permission_matrix_test.go` | 矩阵准备的写入从表和登记中拆出（只移动） | 1 |
| `project/domain/patch.go`、`errors.go`；`adapter/postgres/update.go`、`members.go`、`projects.go`、`queries/projects.sql`、`members.sql` 及测试；`app/ports.go` | 修改的领域；项目的锁；成员关系的读 | 2 |
| `api/modules/project.yaml`；`project/app/lock.go`、`update_project.go`、`fakes_write_test.go`、`clock_test.go`；`adapter/http/*`；`access/domain/rules.go`；`bootstrap/permission_matrix_project_test.go`、`project_writes_test.go`；前端文案 | `updateProject`；一个新码 | 3 |
| `project/app/archive_project.go` 及测试；存储的 `SetArchived`；契约、handler、规则、矩阵 | `archiveProject`、`unarchiveProject` | 4 |
| `project/app/deletion.go`、`cascade.go`；`adapter/postgres/cascade.go`、`queries/cascade.sql` 及测试；`bootstrap/workspace_deletion_catalog_test.go`、`workspace_deletion_test.go` | 删除项目的一处步骤；目录读取按父表 | 5 |
| `project/app/delete_project.go` 及测试；契约、handler、规则、矩阵；`bootstrap/project_deletion_test.go` | `deleteProject` | 6 |
| `project/domain/preferences.go`；`app/get_preferences.go`、`update_preferences.go`、`fakes_preferences_test.go`；`adapter/postgres/preferences.go`、`queries/preferences.sql` 及测试 | 显示设置的领域、存储、用例 | 7 |
| `project/adapter/http/preferences.go` 及测试；契约、矩阵；`bootstrap/project_preferences_test.go` | 显示设置的接口和组合 | 8 |
| `project/domain/member.go`；`app/list_members.go`；`adapter/http/members.go`；存储的 `ListMembers`；契约、规则、矩阵；`bootstrap/permission_matrix_seeded_test.go` 的账户登记 | `listProjectMembers` | 9 |
| `project/domain/member.go`、`member_test.go`；存储的 `RestoreMember`、`EnsurePreferences` | 增长与恢复的领域、存储 | 10 |
| `project/app/growth.go`、`add_members.go`、`fakes_growth_test.go` 及测试；规则 | `addProjectMembers` 的用例 | 11 |
| 契约；`project/adapter/http/add_test.go`；`bootstrap/permission_matrix_members_test.go`、`project_writes_test.go` | 添加的接口；成员的矩阵行 | 12 |
| `project/app/join_project.go` 及测试；契约、handler、规则；`bootstrap/project_members_test.go` | `joinProject`；恢复时的角色 | 13 |
| `bootstrap/interleaving_growth_test.go`、`interleaving_writes_test.go`、`project_connection_test.go`、`demotion_test.go`；`project/adapter/postgres/demote_test.go` | 交错 17；降级与删除、与项目级的写；事务的连接 | 14 |
| `e2e/fixtures/api.ts`、`assert/project.ts`、`assert/workspace.ts`；`e2e/stories/project/p2-…`、`p3-…`、`p4-…`、`p8-….spec.ts`；`workspace/w3-workspace-settings.spec.ts` | 四个故事的接口版本；W3 先删除一个项目 | 15 |
| `docs/v0/plane-diff.md`、`handoffs/M1-P3-trim-platform.md` | 3.20 中 P4b 的一行；M1-P3 的项目成员 | 16 |
| `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`、`project/adapter/http/gen/server.gen.go`、`bodyshape.gen.go`、`project/adapter/postgres/gen/projects.sql.go`、`members.sql.go`、`cascade.sql.go`、`preferences.sql.go`（生成） | | 2–10、12、13 |

`project/…`、`access/…` 指 `server/internal/modules/` 下的模块；`bootstrap/…` 指 `server/internal/bootstrap/`。

### 2.2 依赖

没有新依赖（6.1）。`server/go.mod`、`server/tools/go.mod` 和两个 `go.sum` 不变，仍是 `go 1.27` / `toolchain go1.27.1`；不加 npm 包。

### 2.3 修改的领域；项目的锁；成员关系的读（Task 2；3.6 约定二、3.19、5.2）

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

### 2.4 `updateProject`（Task 3；3.4、3.6、3.19、5.2）

- 接口：`PATCH /api/v0/projects/{project_id}`，`ProjectUpdate`（`additionalProperties: false`，只有 `project_lead_id`、`default_assignee_id` 可为 `null`），200 `Project`；码 `[validation_failed, project.not_found, forbidden, project.archived, project.identifier_taken, project.name_taken]`。新码 `project.archived` 加进 `PROBLEM_MESSAGES` 和两份 `auth.json`（约束 4）。
- 规则 `project.update`：`{Level: LevelProject, Roles: [admin]}`，项目管理员和是工作区管理员的项目成员（3.4）；不是成员的工作区管理员 403。
- `app/lock.go`：`lockAndDecide(ctx, lock, auth, actor, id, action) (LockedProject, Grant, error)`（锁 → 没有是 404 → `decide`），`decide`（看不到换成 `project.not_found`，别的失败原样返回），`answer`（在事务里经 `GetProject` 读回存下的行，写完读不到是内部错误）。修改、归档、恢复、删除、添加、加入共用。
- 用例的顺序：`RequireActor` → `CheckProjectPatch`（事务之前）→ 事务：`lockAndDecide(project.update)` → 已归档 409 → 负责人、默认负责人（只在给了时读 `Memberships`；不是有效成员或是访客：一个 422 列出每个字段，判定之后，约定三）→ 锁下读时钟 → `UpdateProject` → `answer`。不能改项目的人得到 403、404，得不到负责人的任何信息（矩阵的"负责人不是成员"一行）。
- `TestEachWriteReadsTheClockUnderItsLock`（`project/app/clock_test.go`）：每个改已有行的写在锁、判定、检查之后读一次时钟（P2 spec 2.6）；`TestTheWritesOnAProjectStampTheirRequest`（组合出的 app）：每个写由调用者在请求之内盖上 `project.New` 的时钟。

### 2.5 `archiveProject`、`unarchiveProject`（Task 4；3.4、3.19）

- `POST /api/v0/projects/{project_id}/archive`、`/unarchive`，200 `Project`；码 `[project.not_found, forbidden]`；规则同 `project.update`。
- `NewArchiveProject`、`NewUnarchiveProject` 建同一个用例（`archive` 标志）：`lockAndDecide` → 锁下读时钟 → `SetArchived(id, archive, by, now)`（`archived_at` 是 `now` 或 `null`，`updated_by_id`、`updated_at`，别的列不动）→ `answer`。已归档的再归档取新的时刻，未归档的恢复照样成功（Plane `views/project/base.py:427-441`；第 3 节第 8 条）。
- 矩阵的已归档项目从 Task 4 起经存储归档，`standIns` 不再用 SQL 写它。

### 2.6 删除项目；一处删除步骤（Task 5、6；3.3、3.6、3.19；移交第 7 条）

- `app.Deletion{WorkspaceID, ProjectID *uuid.UUID, By, Now}`；`app.ProjectsDeleter` 的四个方法 `DeleteProjects`、`DeleteProjectMembers`、`DeleteProjectPreferences`、`DeleteStates`（取代 P4a 的 `WorkspaceProjectsDeleter` 的四个 `DeleteWorkspace…`，第 3 节第 3 条）。`app/deletion.go` 的 `deleteProjects(ctx, p, d)` 是项目之下各表的唯一一处清单，按 3.6 的全局顺序：项目 → 项目成员 → 显示设置 → 状态，P7 在最后加标签。`Cascade.DeleteWorkspaceProjects`（`ProjectID` 为 `nil`）和 `deleteProject` 都调它，谁都不能漏掉对方删除的表。四条语句形同：

```sql
UPDATE projects
SET deleted_at = $now, updated_at = $now, updated_by_id = $deleted_by
WHERE workspace_id = $workspace_id AND ($project_id::uuid IS NULL OR id = $project_id)
  AND deleted_at IS NULL;
```

- `DELETE /api/v0/projects/{project_id}`，204；码 `[project.not_found, forbidden]`；规则同 `project.update`；已归档的照样删除。用例：`lockAndDecide` → 锁下读时钟 → `deleteProjects(Deletion{锁读到的工作区, &id, 调用者, now})`。
- 目录驱动的核对（移交第 7、8 条，P4a 的 F-M1）：`bootstrap/workspace_deletion_catalog_test.go` 的 `keysTo(t, pool, parent)` 是父表本身（`<父表>.id`）和目录中指向它的每个外键；只经别的表挂在父表下、自己没有指向父表的外键的表让测试失败，并说明它经哪张表挂在下面（`TestKeysRefuseATableWithoutItsParent`，对 `workspaces`、`projects` 两个父表）。`TestDeletingAProjectLeavesNoUndeletedRowUnderIt` 经 `keysTo("projects")` 读出每张表，经接口删除 Web：每个外键下删除之前未删除的行都在项目的时刻、由删除者删除，Ops 的每一行不变；`TestAProjectDeletionRefusedAtItsCommitChangesNoRow` 让 `states`（最后一步的表）的延迟约束触发器拒绝提交，两个项目下的每一行都不变。工作区删除的三个组合测试经 `keysTo("workspaces")` 照旧通过，没有加豁免。
- 存储：`TestDeletingAProjectSoftDeletesItsRowsAlone`（Web、Ops 各一次：已归档、在工作区第一个或最后一个；只动它和它下面的行，同一个时刻、同一个账户，除了三列什么都不写；同工作区的另一个项目、此前删除的项目、别的工作区的项目一行不动；再跑一次什么都不变）。
- 剩下的手写清单：存储测试的 `projectTables`（`adapter/postgres/cascade_test.go`）和 e2e 的 `workspaceTables`；新表漏在它们里面时，组合的目录测试和 e2e 的 `expectProjectDeleted`（也读目录）先失败（第 5 节 P7 一行）。

### 2.7 项目的显示设置（Task 7、8；3.18、4.8、5.2）

- 领域：`Navigation{DefaultTab, HideInMoreMenu}`、`Preferences{Navigation, SortOrder}`、`PreferencesPatch`（没给的不变，导航整个替换）、`DefaultPreferences()`（列的默认值：`work_items`、什么都不藏、65535）、`(Preferences).Apply`、`CheckPreferencesPatch`：默认标签页是 `work_items, cycles, modules, views, intake` 之一（网页的五个，3.18；`pages` 随文档页砍掉），藏起的是 `work_items` 之外的四个之一、只出现一次，大小写不同的算未知；全部问题一个 422，按位置命名字段。
- 读：`GET /api/v0/me/projects/{project_id}/preferences`，`findAndDecide(project_preferences.read)`（`ProjectWorkspace` 不加锁 → 判定，不开事务，6.7）→ `Preferences`，没有行时答 `DefaultPreferences()`，不写（3.18）。
- 改：`PATCH` 同一路径，`CheckPreferencesPatch`（事务之前）→ 事务：`ShareProject`（`FOR SHARE`，3.6 加锁表的"修改项目的显示设置"）→ 判定 → 锁下读时钟 → `UpsertPreferences`：

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

- 领域：`NewMember{MemberID, Role}`；`CheckNewMembers`（1–`MaxNewMembers`（100）个，角色是三种之一，同一个账户只出现一次：事务之前的 422，第 3 节第 9 条）；`CanAdd(workspaceRole, role)`：`addable` 表（工作区管理员只能作管理员、工作区访客只能作访客、工作区成员三种都可以，Plane `views/project/member.py:69-83`，3.5），按集合；`Target{NewMember, WorkspaceRole *Role, Member bool}`、`CheckTargets`：不是工作区的有效成员 `members[i].member_id` `not_allowed`，已是项目的有效成员 `duplicate`，角色不合规 `members[i].role` `not_allowed`，一个 422 按位置。
- 存储：`RestoreMember(id, role, by, now)`（已结束的恢复为有效、取给的角色，保留 id 和 `created_at`）；`EnsurePreferences(row)`（`ON CONFLICT (project_id, user_id) WHERE deleted_at IS NULL DO NOTHING`：已有的设置不动，恢复的成员关系保留它们）。
- `app/growth.go`：`growth{workspaceID, projectID, user, ended *Membership, role, sortOrder, by, now}.apply(ctx, MemberGrower)`：有已结束的成员关系就以 `role` 恢复，否则 `CreateMember`；然后 `EnsurePreferences`。添加、加入共用。
- 用例（`NewAddProjectMembers(AddMembersDeps{Members, Projects, Auth, Tx, Clock})`）：`RequireActor` → `CheckNewMembers` → 事务，按 3.6 的加锁表：`ProjectWorkspace`（不加锁，第 3 节第 10 条）→ `ShareMembers`（目标的工作区成员行 `FOR SHARE`，id 升序，约定三）→ `lockAndDecide(project_member.add)` → `Memberships` → `CheckTargets`（判定之后）→ 锁下读时钟 → 每个目标按请求的顺序：`LowestSortOrder`、`growth.apply`（恢复时取请求的角色，9.1；显示设置在 `SortOrderFirst`，3.18）→ 回答经 `ListMembers` 读回、按请求的顺序（201，第 3 节第 9 条）。整批在一个事务里：一个目标被拒，什么都不写。
- 接口：`POST /api/v0/projects/{project_id}/members`，`ProjectMembersAdd{members: ProjectMemberNew[]}`（`ProjectMemberNew{member_id, role}`），201 `ProjectMemberList`；码 `[validation_failed, project.not_found, forbidden]`；规则 `project_member.add` 同 `project.update`。契约的说明照 P4a review 第 6 节的写法避开守卫 `project-invitations` 的词序（移交第 6 条）：关键词守卫在 P4b 前后都是 5 个命中、都有例外。
- 矩阵（9.2 的 PM、X 两格，约定三）：添加一行把工作区成员 WM-公 加为成员；四个无效目标的变体行（X 的账户不是工作区成员；WG- 是工作区访客，作成员；WA- 是工作区管理员，作成员；PM 已是有效成员），能添加的列得到 422，别的列得到与有效目标相同的 403、404；已归档项目照样添加。`projectSeed.targets` 核对每个目标的前提（清扫 10）。
- `TestTheLeadAndTheDefaultAssigneeAreActiveMembersWhoAreNoGuests`（组合出的 app，移交 1a）：alice 把 carol 加为访客，bob 的成员关系已结束（SQL 代替 P5 的移出），dave 是 acme 的成员、不是 Web 的；三人作负责人、作默认负责人各被拒（422 指名字段），项目不变；成员 erin 两者都可以。

### 2.10 `joinProject`（Task 13；3.5、3.6 约定三和六、3.18、9.1）

- `POST /api/v0/projects/{project_id}/join`，200 `Project`；码 `[project.not_found, forbidden]`；规则 `project.join`：`{Level: LevelVisible}`。
- `CanJoin(workspaceRole)`：`joiners` 表（工作区的管理员、成员，Plane `views/project/invite.py:131-189`），按集合；`JoinRole(ended, workspaceRole)`：新的成员关系取工作区角色，已结束的取它原来的角色与工作区角色中较低的一个，顺序按 `roleOrder`（访客、成员、管理员），不按数字（约定六）。
- 用例（`NewJoinProject(members, projects MemberJoiner, auth, tx, clock)`）：`RequireActor` → 事务：`ProjectWorkspace` → `ShareMembers`（他自己的工作区成员行）→ `lockAndDecide(project.join)` → `CanJoin`，否则 403（在读他的项目成员关系之前，G4：是项目成员的工作区访客也被拒，9.2 的 PG 一格）→ `Memberships`：已是有效成员的什么都不写 → 锁下读时钟 → `growth.apply`（`JoinRole`；显示设置在 65535，3.18）→ `answer`。
- 恢复时的角色（9.1 的表）：`TestJoinRole` 是四行和两个相等的情形；`TestARestoredMembershipGivesNoMoreThanItHad`（组合出的 app）经接口跑其中三行（5/15 → 5、20/15 → 15、15/20 → 15，第四行经接口是 404）和添加的一行（以前是 20，添加为 5 → 5）：同一行恢复为有效，由调用者在那次请求的时刻，建立的时刻不变；新加入的位置 65535。

### 2.11 端到端（Task 15；2 的 P2、P3、P4、P8，9.6；移交第 5 条）

- fixture：`api.ts` 的 `addProjectMembers(api, token, projectId, members)`、`amidAnotherWorkspace`（故事的项目建在调用者另一个工作区的两个项目之间，一前一后：丢掉项目 id 的查询无论先读哪一行都读到别的工作区的项目）；`assert/project.ts` 的 `expectMember`、`expectProjectDeleted`（从目录读出指向 `projects` 的每张表，每张都有这个项目的行，全部在项目的时刻、由删除者删除）；`expectProjectCreated` 只读未删除的项目（删除之后标识可以再用）。
- `assert/workspace.ts`：项目的四张表 `deletedAlone: true`。W3 现在在删除工作区之前删除 Old（成员作负责人，四张表各有它的行），这些行保留自己的时刻，其余的行带工作区的时刻（移交第 5 条：有意设置，原因在注释里）。
- **P2**：管理员列出每个项目，成员列出公开的和自己的，访客只有自己的，已归档的只在要求时列出；成员以成员加入公开项目、位置 65535，再加入什么都不变；成员加入私密项目、访客加入公开项目都是 404，访客加入他是成员的项目是 403。
- **P3**：项目管理员添加一个成员和一个访客，成员列出他们和自己；管理员改每个设置（`site` 存成 `SITE`），以成员为负责人和默认负责人；项目成员不能改（403）；负责人或默认负责人是访客、不是成员，`archive_in` 13：都是 422、不改任何东西。
- **P4**：归档：离开列表、进入已归档的列表，修改 409；恢复、再归档；删除：成员、显示设置、状态在同一时刻删除，之后每个操作都是 404，标识可以再用（移交 1d：删除项目之后"已删除"的谓词可以由故事准备）。
- **P8**：默认标签页设为 modules，把 views 藏进"更多"，把项目拖到侧边栏第一位，设置保留；未知的标签页、藏起 work_items 都是 422、不改任何东西；他的成员的设置是成员自己的。

### 2.12 交错与事务的连接（Task 14；3.6、6.7、9.3 的 17；移交 1e–1g、第 2 条）

全部在真实数据库上，项目一侧经 `bootstrap` 那样组合出的 `project.New`（经接口），降级经工作区的用例。gate 在第一方的事务里、它持有第二方需要的锁时；`pgtest.WaitForLockWaitOn(table)` 证明第二方在那张表的行上等待，之后才放开 gate；每次等待都有期限。

| 测试 | 顺序 | 第二方等在 | 探测 | 反例（附录 A 的清扫 9） |
|---|---|---|---|---|
| `TestADemotionAndTheProjectSidesGrowthSerialize`（交错 17；加入、添加 × 新的、已结束的成员关系 × 两个顺序，8 个子测试） | 增长先：持有 bob 在 acme 的成员行 `FOR SHARE`，在锁 Web 之前停在 gate | 降级等 `workspace_members` | 等待期间 `SELECT … FROM projects WHERE id = Web FOR UPDATE NOWAIT` 成功：两方都还没有锁 Web（P4a 的 F-M2 顺序） | `lo-add-project-first`、`lo-join-project-first`（增长先锁项目）、`lo-demote-projects-first`（降级先锁项目，40P01 或探测失败）、`lo-join-no-share`、`w-add-no-shares`、`w-join-no-shares` |
| 同上 | 降级先：写了成员行之后停住（`demotedHolding`） | 增长等 `workspace_members` | 同上 | 同上；之后增长读到他是访客：加入 404，添加作成员 422 `members[0].role` |
| `TestADemotionAndAProjectsDeletionSerialize`（两个顺序） | 删除先：持有 Web，判定之后停在 gate | 降级等 `projects` | — | 降级把等待期间删除的 Web 留在外面（`LockMemberProjects` 在等待之后重新求值 `deleted_at`，移交 1g） |
| `TestAProjectWriteAndADemotionSerialize`（修改、归档、恢复、改设置 × 两个顺序） | 写先：持有 Web（`FOR NO KEY UPDATE`，改设置 `FOR SHARE`），判定之后停在 gate | 降级等 `projects` | — | `lo-*-decide-first`（先判定后锁）、`lo-lock-none`、`lo-share-*`、`lo-member-projects-share`、`lo-lock-outside-tx`、四个 `w-*-no-tx`；降级先时访客的写 403、改自己的设置 200 |

- `TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（P4a 的 L3，移交 1e 的 `az-project-outside-tx`）：`project.New` 和 `Authorizer` 照 `bootstrap` 接在只有一个连接的池上（请求 3 秒截止，每次等待 5 秒看门狗），alice 建 Ops、修改、归档、恢复 Web、改设置、添加 bob（恢复他已结束的成员关系），carol 新加入，alice 删除两个项目：每个写的锁、读、判定读的事实（`ProjectFacts`、工作区角色）、写和回答都必须在事务的连接上，经池发出的语句等第二个连接、请求在截止时失败（清扫 8 的 20 个变异）。
- `TestAGrowthRefusedAtItsCommitLeavesNoRow`：增长在一个事务里（`refusingCommits(t, pool, table)` 先拒绝成员关系、再拒绝显示设置的提交：添加和加入答 500，什么都不留）。
- `TestLockMemberProjectsLeavesOutAProjectDeletedWhileItWaited`（存储）：另一个事务持有 Web 并软删除它，`LockMemberProjects` 等它，提交之后只返回 Ops。
- 原型：三个交错测试 `-count=5 -race`，15 个顶层 PASS（90 个子测试），没有 40P01，约 17 秒（附录 A）。

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

共 175 格，加 P1–P4a 的 181 格是 356 格；整个矩阵 1.38 秒，准备 0.10 秒（9.2 的预算是 20–30 秒）。

### 2.14 文档（Task 16；3.20 的 P4b 一行）

| 文档 | 位置 | 内容 |
|---|---|---|
| 差异清单 | 四 | 项目负责人、默认负责人一行补上修改时的规则；项目标识一行写"修改项目时同一规则"；新加修改和删除、归档和恢复、添加已是有效成员的人、加入时恢复以前的成员行四行（4.11 中标 P4b 的行） |
| M1-P3 交接 | 文末 | "处理结果（M3/P4b）"：项目成员只经 `addProjectMembers` 和 `joinProject`（接口一侧完成）；页面改调新接口、守卫例外的删除、`RESTRICTED_URLS` 留给 P8 |

## 3. 与设计的差异和补充（待控制者裁定）

以下都没有改变 M3 设计的架构，除第 13 条请负责人确认外，每条给出建议。

1. **任务的划分：16 个，不是 9 个**。设计的任务与 plan 的对应：1 → 2（领域、锁、存储）、3（用例）；2 → 4（归档、恢复）、5（一处删除步骤）、6（删除）；3 → 7（领域、存储、用例）、8（接口）；4 → 9（列表）、10（恢复、增长的存储）；5 → 11（用例）、12（接口、矩阵）；6 → 13；7 → 14；8 → 15；9 → 16（review 是控制者的）；Task 1 是 P4a review 第 6 节要的矩阵拆分。拆分的理由是每个 Task 在约 1,500 行以内并各自是绿的：设计的任务 1、2、5 连同各自的存储、矩阵行和组合测试都超过上限。最大的是 Task 7（1,243 行）、Task 3（1,213 行）和 Task 5（1,106 行）；plan 共 13,363 行，16 个在 brief 的上限之内。**建议接受。**
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
   | 3 一个连带一个时刻（F-M3） | 第 7 节；第 13 条 |
   | 4 矩阵：PA 与 WA- 分开；项目行的 `_id`；拆分；接近 400 行的文件 | Task 1（拆分）、9（账户登记）；P4b 的路径参数只有 `{project_id}`，`_id` 规则不需要改；`schema_test.go`、`directory_test.go`、`invitations_test.go` 不改，`permission_matrix_test.go` 361 行 |
   | 5 e2e：`expectProjectCreated` 读未删除的行；`deletedAlone` | Task 15（2.11） |
   | 6 关键词守卫 | Task 12、13 的契约措辞；守卫前后都是 5 个命中、都有例外，没有新的例外 |
   | 7 删除项目的步骤与 P7 的标签：一处清单或读目录 | Task 5（`deleteProjects` 一处，`keysTo` 读目录）；第 3 条 |
   | 8 锁之后读时钟；锁下的重读确认父行；目录驱动的核对；`apitest.Main`；端口的错误原样返回；按集合 | 时钟：Task 3、4、6、7、11、13 的 `TestEachWriteReadsTheClockUnderItsLock`；重读：项目的锁和 `ShareProject` 在等待之后重新求值 `deleted_at`，添加、加入在锁下用锁读到的工作区；目录：Task 5、6（没有新表，没有豁免）；`apitest.Main`：每个新码有 HTTP 测试回答；端口的错误：清扫 2；按集合：`CanAssign`、`CanAdd`、`CanJoin`、`roleOrder` |

3. **删除的端口改形**（移交第 7 条）：P4a 的 `WorkspaceProjectsDeleter` 的四个 `DeleteWorkspace…` 方法改为 `ProjectsDeleter` 的四个方法，各收一个 `Deletion`（项目可选）；`Cascade.DeleteWorkspaceProjects` 的签名和行为不变（P4a 的组合测试照旧通过）。`deleteProjects` 是项目之下各表的唯一一处清单，P7 在这里加标签；组合的目录测试（`keysTo`）对两个父表都读目录。这是 `project` 模块内部的端口，不跨模块。**建议接受。**
4. **成员列表只读 `project_members`**：不与工作区成员关系做连接（模块之间不 JOIN，6.5）。一个有效的项目成员一定是工作区的有效成员：每条增长锁住他的工作区成员行（约定三），每条收缩结束他的项目成员关系（约定六，`EndMemberships` 在 P5）。矩阵里被移出的成员（P5 之前由 SQL 替身结束了工作区成员关系）因此仍在列表里；P5 的移出写好之后替身换成存储，他就不在了（第 5 节 P5 一行）。**建议接受。**
5. **未知的标签页答 422，不是 400**：契约的 `ProjectTab` 列出五个，生成的请求体结构检查只核对结构，不核对枚举的值；未知的值传到领域，`CheckPreferencesPatch` 答 422（设计第 2 节的 P8"未知的标签页 422"）。HTTP 测试照此传未知的值（`TestUpdateProjectPreferencesPassesTheChange`）。**建议接受。**
6. **故事看不到的谓词，和互相遮住的"已删除"谓词**（清扫 1 的故事一半，移交 1d）：P2、P3、P4、P8、W3 单独运行时，41 个查询谓词变异和 `ProjectFacts` 的两个"已删除"谓词中 29 个被故事发现，14 个看不到：
   - `LockProject`、`ShareProject`、`ProjectWorkspace` 的 `deleted_at IS NULL` 各自单独去掉时由 `ProjectFacts` 的 `p.deleted_at` 遮住（判定答 404），反过来也一样；两两一起去掉时 P4 发现（`pair-lp`、`pair-sp`、`pair-pw`）。
   - `Memberships` 的账户（两序）、已删除（两序），`ListMembers` 的已结束、已删除，`RestoreMember` 的 id，`Preferences` 的已删除（两序），`ProjectFacts` 的 `m.deleted_at`：挡的是已结束或已删除的成员关系、设置，故事里只有 P5 的移出、离开（已结束）写得出；项目还在时已删除的成员关系、设置，P4b 的接口写不出（只有删除项目连同项目一起删除），以后的 Phase 若有写得出它们的接口，由它的故事清扫接过。`Memberships` 的账户谓词要一次添加多个账户、其中一个在项目里有别的行。
   这 14 个都由存储测试在两个行序下发现（附录 A）。**建议接受；P5 的故事清扫接过"已结束"的那些（第 5 节）。**
7. **只在单元或存储一层发现的变异**（brief：安全或加锁的性质只由单元一层发现的算缺口）：
   - `CanJoin` 按大小比较（`d-join-magnitude`）只由 `TestCanJoin` 发现：数据库的 CHECK 只容许 5、15、20，经接口按大小与按集合分不开。`CanAssign`、`CanAdd` 的同类变异由组合出的测试发现（取三种之内的角色就能区分）。
   - 没有调用者时不读（清扫 11 的 8 个 `nocaller-*`）只由单元测试发现：经接口的每个请求都已认证，用例收不到没有调用者的 `ctx`。
   - `archive_in` −1、标识和名称不按规则查（`u-archive-in-low`、`u-identifier-unchecked`、`u-name-unchecked`）只由领域测试发现：数据库的 CHECK 和列的类型拒绝这些值（答 500 而不是 422），不是安全性质。
   - 加锁的性质只在存储一层发现的：`LockProject` 取 `FOR SHARE`、`FOR UPDATE`（`TestLockProject`；没有两个项目级的写互相等待的组合测试，`FOR SHARE` 下两个修改会死锁、`FOR UPDATE` 多挡外键检查）；`LockMemberProjects` 留下等锁时删除的项目（`TestLockMemberProjectsLeavesOutAProjectDeletedWhileItWaited`）。都在真实数据库上，不是缺口。
   **建议接受。**
8. **再归档取新的时刻，恢复未归档的照样成功**：3.19 没有写。照 Plane（`views/project/base.py:427-441`：归档就是写当前时刻）；同一个请求重复发出时答复一致。**建议接受。**
9. **添加的回答和请求里的重复**：添加答 201 `ProjectMemberList`，按请求的顺序，经 `ListMembers` 在事务里读回（恢复的成员关系保留 id 和建立的时刻）。请求里同一个账户出现两次是事务之前的 422（`members[i].member_id` `duplicate`，"is listed before"），与"已是有效成员"同一个码、不同的说明。**建议接受。**
10. **添加、加入先不加锁读项目的工作区**：3.6 加锁表从目标的工作区成员行开始，而请求只给项目的 id；`ProjectWorkspace` 不加锁读出工作区，再按表的顺序加锁。项目的工作区没有写能改，锁之后用锁读到的那个；没有的项目在这一步答 404，与看不到的 404 相同，不泄露什么。**建议接受。**
11. **小的结构调整**（说明）：`SortOrderReader`（`LowestSortOrder`）从 `ProjectCreator` 拆出，`MemberAdder` 只要它；`refusingCommits` 按表（P4a 只拒绝 `workspace_members` 的修改），两个 P4a 的测试照旧用 `workspace_members`；`TestTheWritesOnAProjectStampTheirRequest` 的写一个接一个跑在同一个项目上。
12. **原型中加强的测试，写 spec 时补的变异**（说明）：清扫中发现变异存活的地方，测试在原型中加强，并从带来它的 Task 起改进每一份快照（`amend.py`）：`amidAnotherWorkspace`（故事的谓词在两个行序下都看得到）、一个连接的池的测试、负责人和默认负责人的组合测试、矩阵添加的变体行和 `targets` 的前提、`update_test.go` 的 `cycle_view` 夹具、显示设置读到别的账户的测试、删除的 `unwritten`。写本 spec 的附录时发现修改和显示设置的领域规则、两个唯一键的 409、已归档项目被锁找到、删除的同一时刻没有自己的变异，补了 11 个（`mutants_s12.py`），全部被原有的测试发现，其中 6 个另由故事发现（附录 A）。
13. **一个连带一个时刻的窗口比"一次锁等待"宽**（负责人的裁定，brief 移交第 3 条要求报告）：连带（降为访客的 `LockMemberProjects`、删除工作区的 `DeleteProjects`）在读了时刻之后，按 id 顺序逐行等它要锁的项目行。P4b 起持有项目行的是项目级的写（修改、归档、恢复、删除、添加、加入持 `FOR NO KEY UPDATE`，改设置持 `FOR SHARE`），它们不持工作区锁，所以连带可能依次等几个项目，每等一个，那个项目上的写就以更晚的时刻提交，连带再以它较早的时刻改写那个项目的行（第 7 节）。倒退的长度是连带读时刻之后等锁的总时间：次数可以多于一次，但总时间受连带自己请求的期限限制（`server.request_timeout`，默认 15 秒），这个期限也是一次等待的上限。没有测试这件事（裁定是不改测试）。**请确认"一次锁等待"的说法改为"不超过连带请求的期限"，或改为连带在自己的锁之后读时钟（W3 的"同一时刻"要改写）。**

## 4. 验收标准（完成线，M3 设计 12 节 P4b）

- [ ] P2、P3、P4、P8 的接口版本通过，此前的每个故事仍然通过（`make e2e` 共 62 个：此前的 58 个，加上四个）。
- [ ] 恢复时角色的表（`TestJoinRole` 的四行、`TestARestoredMembershipGivesNoMoreThanItHad` 经接口的三行和添加的一行）通过。
- [ ] 交错 17（加入、添加各两种顺序，新的和已结束的成员关系）、降级与删除项目、项目级的写与降级，`-count=5 -race` 通过，没有 40P01。
- [ ] 本 Phase 的矩阵格子（175 个）通过，含添加无效目标的 PM、X 格；共 356 格，耗时记下；完整性核对通过。
- [ ] 每个项目级的写在事务的连接上（`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`）；增长和删除被拒于提交时一行不留（`TestAGrowthRefusedAtItsCommitLeavesNoRow`、`TestAProjectDeletionRefusedAtItsCommitChangesNoRow`）。
- [ ] 删除项目由目录驱动的组合测试核对（`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`），删除工作区的三个组合测试不加豁免照旧通过。
- [ ] `project` 的 `apitest.Main` 两个方向通过（新码 `project.archived`）。
- [ ] `make lint`、`make test`、`make gen-check`、`make knip`、`make test-web`、`make e2e` 通过。
- [ ] 3.20 中 P4b 的一行写好；M1-P3 交接有"处理结果（M3/P4b）"。

## 5. 不在 P4b 范围内

- `EndMemberships`、移出、离开，交错 1、4、5、6：P5。停用：P6。标签、默认之外的状态：P7。`PROBLEM_MESSAGES` 的搬迁：P8。页面：P9–P11。`project.NewCascade`：P6（G2）。

**留给后面的 Phase**（各 Phase 的 spec 接过去；P4b 的 review 第 6 节照录）：

| Phase | 条目 |
|---|---|
| P5 | 移出、离开写好之后，矩阵的 `standIns` 中"以前的成员""被移出的成员"两条替身换成存储，`listsTheProjectMembers` 不再有被移出的成员（第 3 节第 4 条）；`EndMemberships` 改项目成员关系之前取那些项目的 `FOR NO KEY UPDATE`（3.6 的加锁表；`Memberships` 在项目锁下读到的才是写时的）；改项目成员的角色、移出、离开项目照 3.6 的加锁表先锁项目；故事清扫接过"已结束"的谓词：`ListMembers` 的 `is_active`、`RestoreMember` 的 id（重新加入恢复以前的成员行）、`ProjectFacts` 的 `m.is_active`，`Memberships` 的账户（一次添加多个账户）；交错 1、4、5、6 用 P4b 的添加、加入（`growth`）；`refusingCommits(t, pool, table)` 可以按表拒绝提交；连带的时刻（第 7 节）对 `EndMemberships` 同样成立 |
| P6 | 交错 13、14 另跑恢复以前的项目成员行（`growth` 的 `RestoreMember`）；`project.NewCascade` 随 `nerve users deactivate` 加入（G2） |
| P7 | 标签加进 `deleteProjects`（`app/deletion.go`，最后一步）和 `ProjectsDeleter`；组合的目录测试（`keysTo`）和 e2e 的 `expectProjectDeleted` 读目录，标签表没有准备行、没有被删除时它们失败；两处手写清单要加上标签：存储测试的 `projectTables`、e2e 的 `workspaceTables`；P4、W3 的故事建一个标签 |

## 6. 风险

| 风险 | 应对 |
|---|---|
| 项目级的写不持工作区锁，与工作区一侧的连带只在项目行上相遇 | 每个写先锁项目、在锁下判定（`lockAndDecide`）；三个交错测试在真实数据库上两个顺序都跑，先判定后锁、锁在事务之外、不开事务的变异都被发现 |
| 增长与降级的加锁顺序（P4a 的 F-M2） | 交错 17 的探测证明第二方等待时两方都还没有锁项目；反过来的三种顺序（`lo-*-project-first`、`lo-demote-projects-first`）都被发现 |
| 删除项目与删除工作区漏掉对方的表 | 一处 `deleteProjects`；两个父表的组合测试都读目录；`keysTo` 拒绝只经别的表挂在下面的表 |
| 故事看不到的谓词（第 3 节第 6 条） | 存储测试在两个行序下都看得到；互相遮住的三对由 P4 一起发现；P5 的故事清扫接过"已结束"的那些 |
| plan 的最大 Task 接近上限 | 最大的 Task 7 是 1,243 行，每个都在约 1,500 行以内 |
| 持续集成（ubuntu）上的运行 | 原型和复现都在 macOS 上；合并之后看持续集成，P1–P4a 都是这样合并的 |

## 7. 已知的限制、交接和关闭条件

- **连带的时刻**（P4a 的 F-M3，负责人 2026-10-01 裁定保持设计 3.3）：连带把调用者在加锁之前读的一个时刻写进它之后才锁的行。P4b 起项目级的写只持项目锁：连带（降为访客、删除工作区）读了时刻之后等某个项目的行时，那个项目上的写以更晚的时刻提交，连带随后以较早的时刻改写同一个项目的行（降级改他的项目成员关系；删除工作区改项目、成员关系、显示设置、状态），这些行的 `updated_at` 可能比之前的值早。**窗口**：连带读时刻之后等锁的总时间。连带逐个等它要锁的项目行，P4b 的写可能持有其中几个，所以可以是几次等待；总时间受连带自己请求的期限限制（`server.request_timeout`，默认 15 秒），这也是一次等待的上限。`deleted_at`、`role` 等值本身不受影响，只有时刻的先后。没有测试这件事（第 3 节第 13 条）。
- **矩阵里被移出的成员仍在成员列表里**：他的工作区成员关系由 SQL 替身结束，项目成员关系留着有效；P5 的移出结束两者（第 3 节第 4 条）。
- **`getProjectPreferences` 的读和判定是两次查询**：不开事务（6.7 的读），答案是一时的。

| 交接 | P4b 处理的条目 | 留下的条目 |
|---|---|---|
| M1-P3 项目成员 | 只有"从工作区成员中添加"（`addProjectMembers`）和自己加入（`joinProject`），没有按邮件加人的接口（Task 12、13；"处理结果"在 Task 16） | `joinProject` 的页面改调新接口、守卫的 `project-invitations` 例外删除、`RESTRICTED_URLS` 与后端同源（前端一侧，P8）；本节保持 `open` |
| P4a spec 第 5 节、review 第 6 节的 P4b 一行 | 落点见第 3 节第 2 条 | 第 5 节 |

**M3 设计 13.1 的关闭条件**（P4b 的一行）：M1-P3 项目成员：接口一侧只有"从工作区成员中添加"（`addProjectMembers` 的契约只收工作区成员的 `member_id`），`joinProject` 的前端一侧在 P8。P4b 的条件都有落点，没有放不下的。

## 附录 A：原型验证记录（2026-10-01）

原型在 `$M3TMP/p4bproto`（`e22e5080` 的副本，Go 1.27.1、Node 24、pnpm 11.10.0、Docker；`bin/golangci-lint` 2.13.2）。做法照 P4a：每个 Task 做完时存一份源文件的快照（`$M3TMP/p4bsnap/T1`…`T16`），plan 的代码块由脚本从相邻两份快照的差异生成（`p4btools/mkblocks.py`），生成的文件不进块，按 SHA-256 核对（`gensha.py`）。清扫之后加强的测试由 `amend.py` 从带来它的 Task 起改进每一份快照（第 3 节第 12 条），之后重新生成全部块。

**逐 Task 复现**（`$M3TMP/p4btools/replay.py`、`replay_all.sh`，日志在 `replay-logs`）：在 `$M3TMP/p4breplay`（`e22e5080` 的另一个副本，`pnpm install --frozen-lockfile`）按 plan 的 16 个 Task 依次执行：`planapply.mjs` 从 plan 的文本中取出这个 Task 的块写入，然后按顺序执行这个 Task 的每一条 `Run:` 命令，原样照 plan。例外只有：`make gen`、`make gen-go` 之后核对全部生成物与这个 Task 的快照逐字节相同；`shasum -a 256` 的输出与 plan 表中的 SHA-256 和行数核对；副本不是 git 仓库时 `make lint-web` 的关键词守卫由 `kwcheck.mjs` 代替（同样的规则、同样的文件）；`make e2e` 之前把副本初始化为 git 仓库并提交（F4）；提交之后的 `make gen-check` 由生成物的核对代替。每个 Task 另核对 `go.mod`、`go.sum`、`server/tools` 和 `pnpm-lock.yaml` 不变。整个复现 739 秒，全部通过。第一次复现在 Task 13 的 `make test` 中断：另一个会话的容器正在启动时，`platform/postgres` 的 testcontainers 起不来（Docker socket 上 `context deadline exceeded`，`p4breplay-v1`、`replay-logs-v1`），与 plan 无关；等 Docker 空闲之后在新的副本上从头重跑。

| Task | 复现的结果（每个 Task 都有 `make lint-go` 两个 `0 issues.`、`make test` 42 个 `ok`） |
|---|---|
| 1 | 3 个文件；矩阵、完整性、账户的核对 ok |
| 2 | 12 个文件；`make gen-go`：32 个生成物相同，2 个 SHA-256 相同；`project/...` ok |
| 3 | 19 个文件；`make gen`：32 个相同，4 个 SHA-256 相同；`project/...`、`access/...` ok；矩阵、完整性、规则表、路由、401、请求体、写的时刻 ok；前端检查通过 |
| 4 | 23 个文件；`make gen`：4 个 SHA-256 相同；同上 |
| 5 | 11 个文件；`make gen-go`：1 个 SHA-256 相同；`project/...` ok；删除工作区的三个组合测试、`keysTo` 的反例 ok |
| 6 | 16 个文件；`make gen`：3 个 SHA-256 相同；矩阵、两个删除项目的组合测试、写的时刻 ok；前端检查通过 |
| 7 | 20 个文件；`make gen-go`：2 个 SHA-256 相同；`project/...`、`access/...`、规则表 ok |
| 8 | 10 个文件；`make gen`：4 个 SHA-256 相同；矩阵、`TestEachMemberHasHisOwnDisplaySettings`、写的时刻 ok；前端检查通过 |
| 9 | 21 个文件；`make gen`：4 个 SHA-256 相同；矩阵、账户、规则表 ok；前端检查通过 |
| 10 | 10 个文件；`make gen-go`：2 个 SHA-256 相同；`project/...` ok |
| 11 | 13 个文件；`project/...`、`access/...`、规则表 ok |
| 12 | 12 个文件；`make gen`：4 个 SHA-256 相同；矩阵、负责人的组合测试、显示设置、写的时刻 ok；前端检查通过 |
| 13 | 19 个文件；`make gen`：3 个 SHA-256 相同；矩阵、`TestARestoredMembershipGivesNoMoreThanItHad` ok；前端检查通过 |
| 14 | 7 个文件；`LockMemberProjects` 的存储测试、一个连接的池、两个提交被拒的测试、P4a 的两个降级测试 ok；三个交错测试 `-count=5 -race` ok（17 秒，0 个 40P01） |
| 15 | 8 个文件；前端检查通过；`make e2e` 62 个全部通过 |
| 16 | 2 个文件；`make lint-web`（副本此时已是仓库，照原样执行）通过 |
| 结束 | 复现的树与原型逐文件相同（2,990 个文件，0 个差异）；副本上 `make gen-check` 通过（生成物在 Task 15 的提交里，Task 16 只改文档）；`planapply.mjs check` 从 `e22e5080` 起 342 个块全部通过 |

| 核对 | 命令 | 结果 |
|---|---|---|
| 生成物 | 原型上 `make gen`，生成物与运行之前逐字节比较；复现的副本上 `make gen-check` | 没有差异 |
| Go 静态检查 | `make lint-go` | server 0 issues；server/tools 0 issues |
| Go 测试 | `make test`（testcontainers，`postgres:18.6`） | 全部通过（41 个包加 `server/tools` 的 1 个，与 P4a 相同；P4b 不加包） |
| 前端检查 | `make lint-web`（关键词守卫：3,020 个文件、5 个命中、都有例外，P4a 结束时同样 5 个；turbo 54 个任务）；`make knip`；`make test-web`（16 个任务） | 全部通过 |
| 端到端 | `make e2e` | 62 个：原型不是 git 仓库时 S3 失败（F4），其余 61 个通过；复现的副本 62 个全部通过 |
| 矩阵 | `go test -count=1 -v -run 'TestPermissionMatrix$'` | 356 格（本 Phase 175 格），整个矩阵 1.38 秒，准备 0.10 秒（原型）；9.2 的预算是 20–30 秒 |
| 交错 | `go test -count=5 -race -run '…GrowthSerialize$|…DeletionSerialize$|…DemotionSerialize$' ./internal/bootstrap/` | 15 个顶层 PASS（90 个子测试），0 个 FAIL，输出中 0 个 40P01，17 秒 |
| 变异 | `mutants_s1`–`s6`、`s8`、`s9`、`s10_s11`、`s12.py`（Go，270 个）、`e2e_sweep1.py`（43 个，每个只运行用到它的故事，另 3 对）、`e2e_mutants.py`（15 个，单独运行一个故事） | 见下 |

**原型中定下的事实**：

- **F1** 等锁之后 Postgres 在行的最新版本上重新求值 `WHERE` 的条件：`LockProject`、`ShareProject`、`LockMemberProjects` 等待期间被删除的项目读不到（`TestTheProjectLockSeesADeletionItWaitedFor`、`TestLockMemberProjectsLeavesOutAProjectDeletedWhileItWaited`）。
- **F2** `FOR NO KEY UPDATE` 与 `FOR SHARE`、另一个 `FOR NO KEY UPDATE` 冲突，与外键检查的 `FOR KEY SHARE` 不冲突；两个 `FOR SHARE` 不冲突（`TestLockProject`、`TestShareProjectAndProjectWorkspace` 用 `NOWAIT` 核对）：改设置之间不互等，修改与改设置互等。
- **F3** `ON CONFLICT (project_id, user_id)` 不带部分唯一键的条件时，Postgres 推断不出那个部分唯一索引，每次插入都报错：`ups-target`、`ens-target` 因此连故事都发现（`EnsurePreferences` 在每次添加、加入里）。
- **F4** 原型不是 git 仓库时 S3（实例信息里的 `commit`）失败（P3 附录 A 的 F8）；复现在 `make e2e` 之前把副本初始化为仓库并提交。
- **F5** 一条语句锁几行时逐行等待（`LockMemberProjects` 按 id 顺序，`DeleteProjects` 按扫描的顺序）：连带的等待可以是几次（第 7 节）。
- **F6** 只有一个连接的池上，经池而不是经事务发出的语句等第二个连接，直到请求的期限：这让"在事务之外"的每个变异都表现为写失败，不依赖时序（`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`）。

**清扫**（brief 的十一类，穷尽地跑在 P4b 新加或修改的代码上；每个变异的类别和必须失败的测试写在 `mutants_*.py` 的每一条上；"层"是必须失败的测试所在的层）：

| 清扫 | 规模 | 结果和层 |
|---|---|---|
| 1 每个 SQL 谓词 | 新加或修改的 15 条查询（`LockProject`、`ShareProject`、`ProjectWorkspace`、`UpdateProject`、`SetArchived`、`Memberships`、`ListMembers`、`RestoreMember`、`Preferences`、`UpsertPreferences` 和 `EnsurePreferences` 的冲突目标、删除的四条）的每个 `WHERE` 条件各去掉一个：41 个 Go 变异；第一行决定答案的（`:one`、`Memberships` 的映射）按 id 升序、降序两个行序各跑一次 | 41 个全部被存储测试发现，两个行序都是。故事一半（P2、P3、P4、P8、W3 各自单独运行，另加 `ProjectFacts` 的两个"已删除"谓词）：43 个中 29 个被故事发现，14 个看不到（第 3 节第 6 条），互相遮住的三对一起去掉时 P4 发现 |
| 2 每个端口调用的错误 | 用例对端口（存储的读和写、项目的锁、跨模块的 `ShareMembers`、`Authorizer`、事务的提交）的每个调用失败被吞掉、被重试、被答成别的问题；每个 handler 吞掉用例的失败：63 个 | 全部被发现：54 个在用例一层（调用记录比较到失败的那一次为止，之后的调用不能有），9 个在 HTTP 一层 |
| 3 每个写不动的行和列 | 每个存储的写（`UpdateProject`、`SetArchived`、四条删除、`UpsertPreferences`、`RestoreMember`、`EnsurePreferences`）多写一列、少写一列、不看标志、保留不该保留的：30 个 | 全部被存储测试发现；每个测试都有别的项目（同工作区、已归档、别的工作区）、别的成员、别的账户的行，核对每一行的每一列 |
| 4 组合根的每个接线 | `project.New` 给七个写的事务管理器换成不开事务的、时钟换成固定在 2000 年的（各 7 个），给九个用例的 `Authorizer` 换成谁都当作管理员放行的（9 个），给添加、加入的成员端口换成什么都不锁、答谁都是工作区成员的（2 个），归档和恢复接反（1 个）：26 个 | 全部被组合出的测试发现（`TestPermissionMatrix`、`TestTheWritesOnAProjectStampTheirRequest`、`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`、`TestARestoredMembershipGivesNoMoreThanItHad`、三个交错测试、两个提交被拒的测试） |
| 5 每个安全性质在真实的系统上 | 八条规则各放宽或收紧、看不到的项目答 403（17 个）；P4b 自己的类别（加入的两条规则在其中，下表）36 个 | 17 个全部被矩阵发现；P4b 自己的类别见下表，只在单元一层的见第 3 节第 7 条 |
| 6 每一句说明 | 契约九个操作的每段描述、P4b 的代码注释和查询注释、差异清单和交接的每一行，逐句对照代码或测试（下面的对照）；3 个契约变异（`updateProject` 不声明 `project.archived`、`addProjectMembers` 声明 200、`joinProject` 不声明 `forbidden`） | 3 个变异被矩阵（答案按契约核对）和 HTTP 测试发现；逐句的对照没有发现不符 |
| 7 反例里没有随机 | P4b 新加、修改的测试和故事里没有 `math/rand`、`crypto/rand`、`Math.random`；id 由 `uuid.NewV7()` 生成，行序决定结果的存储测试按固定的顺序写入（例如 `TestMemberships`：carol 已删除的成员关系先于她已结束的存入），清扫 1 另按两个行序各跑一次；`time.Now()` 只给准备的行盖时刻，断言比较的是写入的时刻本身或请求前后的区间 | 没有发现 |
| 8 写的判定读在调用者的事务里 | 写的事务里的每个读和写（项目的锁、`ShareProject`、`ProjectWorkspace`、`Memberships`、`ProjectFacts`、`ActiveRole`、`ShareMembers`、`LowestSortOrder`、`ListMembers`、`GetProject`、每个写）各改为经连接池：20 个 | 全部被 `TestTheWritesOnAProjectRunOnTheirTransactionsConnection` 发现（`ProjectFacts` 另有 P4a 的存储测试） |
| 9 每个加锁顺序在真实的系统上 | 锁的强度（`LockProject`、`ShareProject`、`LockMemberProjects`）、先判定后锁（修改、归档）、项目锁在事务之外、增长先锁项目（添加、加入）、加入不锁他的工作区成员行、降级先锁项目（F-M2）、`LockMemberProjects` 留下等锁时删除的项目：14 个 | 全部被发现：6 个只在组合一层（交错测试），4 个在组合和存储两层，3 个只在存储一层（`LockProject` 的两种强度、`LockMemberProjects` 的删除，第 3 节第 7 条），1 个在组合和单元两层 |
| 10 承重的准备行都有前提 | 矩阵的准备（`targets`、`preconditions`）和交错测试的准备中，每一行改成会让变异通过的状态：11 个 | 全部被前提或测试发现（组合） |
| 11 每条拒绝的路径 | 没有调用者时不读（每个用例一个，8 个）；负责人在判定之前检查（1 个）：9 个；添加的目标在判定之前检查（`a-targets-first`）在 P4b 自己的类别里，矩阵的四个变体行给 PM、X 等格 | 全部被发现：8 个在用例一层（第 3 节第 7 条），1 个在矩阵 |
| 写 spec 时补的 | 修改的领域规则（`archive_in` 两端、标识不转大写、标识和名称不按规则查）、两个唯一键的 409 互换、显示设置的两条规则、项目的锁和 `ShareProject` 找不到已归档的项目、状态在另一个时刻删除：11 个（`mutants_s12.py`，第 3 节第 12 条） | 全部被发现；其中 6 个另各让它的故事单独运行（`e2e_mutants.py`）失败 |

**按缺陷类别**（brief 的 P4b 类别；"层"是变异被发现的层）：

| 类别 | 变异 | 被哪些测试发现 | 层 |
|---|---|---|---|
| 恢复时的角色（9.1，I1） | `JoinRole` 取较高的、取工作区角色、保留已结束的角色；添加不取请求的角色 | `TestJoinRole`、`TestARestoredMembershipGivesNoMoreThanItHad`；后者另有 P3 | 组合；端到端 |
| 加入 | 工作区访客加入公开项目；成员加入私密项目；`CanJoin` 在"已是有效成员"之后（G4）；位置不是 65535；不锁他的工作区成员行；规则只给项目成员；`CanJoin` 按大小 | `TestPermissionMatrix`（PG、WM-私 等格）、P2；`TestARestoredMembershipGivesNoMoreThanItHad`；交错 17、`TestJoinProject`；`TestCanJoin`（按大小，第 3 节第 7 条） | 组合；端到端；单元 |
| 添加 | 不是工作区有效成员的目标；访客作成员、管理员作成员；目标在判定之前；已是有效成员不答 `duplicate`；部分写入；位置不按 3.18 | `TestPermissionMatrix`（四个变体行）；`TestAddProjectMembers`、P3 | 组合；单元；端到端 |
| 修改 | 负责人、默认负责人是项目访客、不是成员、已结束的成员；不检查；`archive_in` 越界；已归档的照改；标识的规则与建项目不同；两个唯一键的 409 互换 | `TestPermissionMatrix`、`TestTheLeadAndTheDefaultAssigneeAreActiveMembersWhoAreNoGuests`、P3、P4；`TestCheckProjectPatchReportsEveryField`、`TestThePatchChecksAsCreateDoes`；`TestUpdateProjectIdentifierOrNameTaken` | 组合；端到端；单元；存储 |
| 归档、恢复、删除 | 项目成员、不是成员的工作区管理员能做（六个规则变异）；删除别的工作区；一张表的行留着、在另一个时刻、由另一个账户删除；已归档的项目找不到 | `TestPermissionMatrix`；`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`；`TestDeletingAProjectSoftDeletesItsRowsAlone`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`、P4；`TestLockProject`、`TestShareProjectAndProjectWorkspace`、矩阵的已归档列 | 组合；存储；端到端 |
| 显示设置 | 读、改别人的；未知的标签页、藏起 work_items；看不到的项目（规则给看得到的每个人、不给访客） | `TestEachMemberHasHisOwnDisplaySettings`、P8；`TestCheckPreferencesPatchReportsEveryTab`、P8；`TestPermissionMatrix` | 组合；端到端；单元 |
| 成员列表 | 列已结束、已删除的；别的项目的；看不到的人得到 404 以外的 | `TestListMembers`；P3；`TestPermissionMatrix` | 存储；端到端；组合 |
| 交错 17 | 添加、加入各对降级，两个顺序；探测（`webFree` 的 `NOWAIT`，`WaitForLockWaitOn` 指名表）；没有挂起：每次等待有期限 | 2.12 的表 | 组合 |
| 父锁、强度、事务 | `Authorize` 在父锁之前；父锁没有 `deleted_at IS NULL`；强度错；锁在事务之外 | 清扫 8、9；`TestLockProject`、`TestTheProjectLockSeesADeletionItWaitedFor`、`TestShareProjectAndProjectWorkspace` | 组合；存储 |
| 规则表加行而没有格子、放宽 | 九条规则各放宽、收紧；每条新规则带它的格子 | `TestPermissionMatrix`；`TestEveryRuleDecidesItsCells` 要求每条规则的格子 | 组合；单元 |
| 精确的错误 | 看不到答 403；`Authorizer` 的失败答 403；加入时找工作区的失败答 404 | `TestPermissionMatrix`；各用例的 `ReturnsEachFailure`（`errors.Is` 比较具体的错误；写完读不到是内部错误，不是 404） | 组合；单元 |
| 码只在别的模块的测试包里返回（9.4） | 没有新变异：`project` 的 HTTP 测试回答 `project.archived` 和每个新操作声明的码，`apitest.Main` 两个方向通过；契约少声明一个码的 3 个变异见清扫 6 | `apitest.Main` | 单元 |

**故事看得到的谓词**（`e2e_sweep1.py`；每次只运行用到这条查询的故事；"两序"是按 id 升序、降序各跑一次）：

| 查询 | 谓词 | 发现它的故事 |
|---|---|---|
| `LockProject` | `id`（两序） | 升序 P2、P3、P4、P8、W3；降序 P2、P3、P4 |
| | `deleted_at IS NULL` | 看不到（与 `ProjectFacts.p.deleted_at` 一起去掉时 P4） |
| `ShareProject` | `id`（两序） | P8 |
| | `deleted_at IS NULL` | 看不到（同上） |
| `ProjectWorkspace` | `id`（两序） | 升序 P2、P3、P4、P8；降序 P3、P8 |
| | `deleted_at IS NULL` | 看不到（同上） |
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
| `ProjectFacts`（P4a 的） | `m.deleted_at`；`p.deleted_at` | 看不到（后者与三个锁的一起去掉时 P4） |

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

**变异核对**（`mutrun.py`：Go 文件经 `go test -overlay` 改，契约的打包文件就地改并恢复；生成的查询常量代表它的查询；期望点名的测试失败并且输出含它的失败行；编译不过不算）。各 Task 的变异在它的快照上写成，最终的原型上全部再跑一次（`final_runs.sh`，与关卡、矩阵、交错依次运行）：`s1` 41/41（90 秒）、`s2` 63/63（39 秒）、`s3` 30/30（70 秒）、`s4` 26/26（177 秒）、`s5` 42/42（235 秒）、`s6` 5/5（19 秒）、`s8` 20/20（116 秒）、`s9` 14/14（222 秒）、`s10_s11` 18/18（32 秒）；`s12` 11/11 之后单独跑。故事一层：`e2e_sweep1.py` 43 个、三对；`e2e_mutants.py` 15 个（`s5` 的 9 个、`s12` 的 6 个）全部让它的故事失败。每个 Go 变异按带来它所改代码的 Task 列进 plan 的变异表（`mutmap.py`）：Task 2 33 个、3 23 个、4 18 个、5 28 个、6 8 个、7 40 个、8 19 个、9 11 个、10 15 个、11 23 个、12 14 个、13 24 个、14 12 个，P4a 的代码 2 个。

**没有证明的**：

- 持续集成（ubuntu）上的运行：原型和复现都在 macOS 上。
- 两个项目级的写在项目锁上互相等待：只有存储一层的 `NOWAIT` 核对（第 3 节第 7 条）。
- 第 3 节第 6 条的故事看不到的谓词：由存储测试证明，"已结束"的故事一侧留给 P5。
- 连带时刻的窗口（第 7 节）：按裁定没有测试。
- 页面版本（P10）；交错 1、4、5、6（P5）。
