# M3/P4a 项目的建立、可见性与两个连带：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M3/P4a `projects` |
| 日期 | 2026-10-01 |
| 状态 | 待裁定（第 3 节） |
| 上级文档 | [M3 设计文档](../M3-design.md) 第 2（P1、W3）、3.3、3.4、3.6（加锁顺序、加锁表、约定一至五）、3.8（接受恢复为访客）、3.12、3.15、3.17、3.18、3.19、3.20（P4a 一行）、4.1、4.6–4.9、4.11（P4a 各行）、4.12、5.1–5.3、6.3、6.5、6.6、6.7、8.2、9.1（最后一条中接受的一半）、9.2、9.3、9.4、9.6、12（P4a 与约束 1–4）、13.1 节；[M2 设计](../../M2-auth/M2-design.md) 3.11–3.14 节 |
| 前置交接 | [P3 review](../reviews/P3-invitations-review.md)、[P2 review](../reviews/P2-workspaces-review.md)、[P1 review](../reviews/P1-platform-review.md) 第 6 节交给 P4 的各件（落点见第 3 节第 2 条）；[M2 收尾交接](../handoffs/M2-closeout.md) 第 7 节的接口一侧、第 9 节；[M1-P2](../handoffs/M1-P2-trim-content.md) 的项目字段；[M1-P3](../handoffs/M1-P3-trim-platform.md) 不再读的字段、地址 |
| 拆分与裁定 | 负责人裁定设计中的 P4 拆成 P4a、P4b（2026-10-01，M3 设计第 12 节约束 2）；控制者的裁定 S1–S5、G1–G4 已落实：S1、G1、G2、G4 在设计（`d0796853`）；G3 在 2.7；S2 在 2.9 和 M3 设计 6.3 的 `module.go` 一行（本 spec 的提交）；S3 在 2.3；S4 在 2.15；S5 在第 5 节 |
| 计划 | [P4a plan](../plans/P4a-projects.md) |

本 spec 只写 M3 设计交给 P4a 决定的东西：名字、签名、SQL、测试名，以及原型证明了什么。规则本身以 M3 设计为准，这里引用节号，不重述。P4a 依赖 P3（`25d1c331` 的 `main`）和拆分的设计修改（`d0796853`）。

P4a 是评审敏感的一段（M3 设计 12 节约束 3）：两个连带第一次跨越模块边界，"谁看得到项目"第一次落地。连带、锁和可见性的每个测试都按安全测试对待：附录 A 的七个清扫逐个核对它们在所说的性质去掉之后失败。

## 1. 目标

按 M3 设计 12 节 P4a：`project` 模块和项目的四张表出现；工作区的管理员和成员能建项目，调用者按可见性列出、查看项目，检查标识；删除工作区和降为访客的连带随项目一起成立。具体是：

- 迁移 `00010`–`00013`（`projects`、`project_members`、`project_user_properties`、`states`），它们的运行时权限和 `sqlc.yaml` 的 `project` 条目；四张表的名字、种类、CHECK 的反例（含 `projects_logo_props_check` 的十个反例、四个合法值）和部分唯一键的行为（4.6–4.9）；
- 新模块 `project`：domain（名称、标识、网络、图标、负责人、默认的 6 个状态、侧边栏的位置、可见性、操作名）；四个操作 `createProject`、`getProject`、`checkProjectIdentifier`、`listProjects`；`Cascade` 的 `DeleteWorkspaceProjects`、`DemoteToGuest`；
- 跨模块的端口：`project.Provide` 的 `ProjectAccess`（只有它，裁定 S2），`access` 的项目级端口；`workspace.Provide` 的 `WorkspaceDirectory`、`WorkspaceMembers`；`workspace` 的 `ProjectCascade`；`project.New` 在 `workspace.New` 之前（6.6）；
- 两个连带：删除工作区的 `cascade()` 的最后一步；`updateWorkspaceMember` 改为访客、接受邀请恢复为访客时 `DemoteToGuest`（3.3、3.8），两处都有组合出的 app 上的回滚测试；
- 矩阵：每行带自己的列（项目级 12 列），`notTargets` 按路径参数列出（裁定 G3），`moduleActions` 加 `project`（裁定 S3）；本 Phase 的 49 格（共 181 格）；列表与逐个判定一致的测试（9.3）；
- 每个模块的 HTTP 测试都经 `apitest.Main` 的核对；`workspace/app/fakes_test.go` 按用例拆开；
- 端到端：`api.ts` 的建项目、`assert/project.ts`；P1 的接口版本；W3 加上项目的连带（不含标签，裁定 S4）；
- 3.20 中 P4a 的一行（差异清单），M2 收尾、M1-P2、M1-P3 交接的"处理结果（M3/P4a）"。

## 2. 交付物

### 2.1 文件总览

路径相对于仓库根目录。"生成"表示由 `make gen`（或 `make gen-go`）生成并提交，不手改。"Task"是 plan 中负责它的任务（plan 有完整的文件表）。

| 路径 | 内容 | Task |
|---|---|---|
| `workspace/app/fakes_test.go` 和四个 `fakes_*_test.go` | P2、P3 的共用假实现按用例拆开（只移动） | 1 |
| `server/migrations/sql/00010`–`00013`、`schema_test.go`、`project_schema_test.go`；`deploy/runtime-grants.sql`；`server/sqlc.yaml` | 四张表；名字和种类；运行时权限 | 2 |
| `project/domain/actions.go`、`app/ports.go`、`app/cascade.go`、`adapter/postgres/store.go`、`cascade.go`、`queries/cascade.sql` 及测试；`project/module.go` | `project` 的骨架和删除的连带 | 2 |
| `workspace/app/ports.go`、`delete_workspace.go`、`fakes_projects_test.go`、`module.go` 及测试；`api/modules/workspace.yaml`；`bootstrap/app.go`、`actions_test.go`、`workspace_deletion_test.go`、`interleaving_answers_test.go` | `ProjectCascade`；`cascade()` 的最后一步；组合的顺序；组合的删除测试 | 2 |
| `project/domain/project.go`、`logo.go`、`state.go`、`preferences.go` 及测试；`project_schema_test.go` 的反例 | 领域；CHECK 的反例、部分唯一键 | 3 |
| `project/adapter/postgres/projects.go`、`rows.go`、`queries/projects.sql`、`members.sql`、`preferences.sql`、`states.sql` 及测试；`domain/errors.go` | 插入、`GetProject`、`LowestSortOrder` | 4 |
| `workspace/app/directory.go`、`adapter/postgres/directory.go`、`queries/directory.sql` 及测试 | `WorkspaceDirectory`、`WorkspaceMembers` | 5 |
| `bootstrap/permission_matrix_columns_test.go`、`permission_matrix_targets_test.go` 和矩阵的其余文件 | 每行带自己的列；项目级的列、账户、项目；按参数的"不是目标" | 6 |
| `project/app/create_project.go`、假实现和测试；`access/domain/rules.go` | `createProject` 的用例；`project.create` | 7 |
| `api/modules/project.yaml`、`api/openapi.yaml`；`project/adapter/http/*`；`bootstrap/ports.go`、`project_test.go`、`permission_matrix_project_test.go`；`apitest/main_callers_test.go`；前端文案 | `createProject` 的接口；接线；`apitest.Main` 的核对；两个新码 | 8 |
| `access/app/ports.go`、`authorizer.go`、`module.go`；`project/app/access.go`、`get_project.go`、`adapter/postgres/access.go`、`queries/access.sql` 及测试；前端文案 | `ProjectAccess`；`getProject`；一个新码 | 9 |
| `project/app/check_identifier.go` 及测试；`project/domain/project.go` 的 `ValidIdentifier`；存储的 `IdentifierTaken`；契约、handler、规则、矩阵 | `checkProjectIdentifier` | 10 |
| `project/domain/visibility.go`、`app/list_projects.go`、`adapter/postgres/list_test.go`、`projects_test.go`（夹具移进 `newWebFixture`）；`bootstrap/project_visibility_test.go` | `listProjects`；列表与逐个判定一致 | 11 |
| `project/adapter/postgres/demote_test.go`；`workspace/app/update_member.go`；`bootstrap/demotion_test.go` | `DemoteToGuest`；改为访客时调用 | 12 |
| `workspace/app/accept_invitation.go` 及测试；`bootstrap/demotion_test.go` | 接受恢复为访客时调用 | 13 |
| `e2e/fixtures/api.ts`、`auth.ts`、`assert/project.ts`、`assert/workspace.ts`；`e2e/stories/project/p1-create-project.spec.ts`、`workspace/w3-workspace-settings.spec.ts` | P1 的接口版本；W3 的项目连带 | 14 |
| `docs/v0/plane-diff.md`、`handoffs/M2-closeout.md`、`M1-P2-trim-content.md`、`M1-P3-trim-platform.md` | 3.20 中 P4a 的一行；交接的处理结果 | 15 |
| `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`、`project/adapter/http/gen/*.gen.go`、`project/adapter/postgres/gen/*.go`、`workspace/adapter/postgres/gen/directory.sql.go`（生成） | | 2、4、5、8–12 |

`project/…`、`workspace/…`、`access/…` 指 `server/internal/modules/` 下的模块；`bootstrap/…`、`apitest/…` 指 `server/internal/` 下（`apitest` 在 `platform/httpserver/` 下）。

### 2.2 依赖

没有新依赖（6.1）。`server/go.mod`、`server/tools/go.mod` 和两个 `go.sum` 不变，仍是 `go 1.27` / `toolchain go1.27.1`；不加 npm 包。

### 2.3 迁移、`project` 的骨架与删除工作区的连带（Task 2；M3 设计 3.3、3.6、4.6–4.9、4.12、6.6）

- **四张表**照 4.6–4.9：`projects` 23 列（Plane 的 22 列加计数列 `last_issue_sequence`），`project_members`、`project_user_properties` 各 11 列，`states` 14 列。负责人、默认负责人 `ON DELETE SET NULL`（3.15、4.12）；名称、标识、网络、`archive_in`、`last_issue_sequence`、项目角色、显示设置、状态的名称和组各有 CHECK；`projects_logo_props_check` 按 M2 设计 3.13 查到底：是对象、键的集合、出现的每个键的值类型，嵌套的 `emoji`、`icon` 同样。五个部分唯一键（`projects` 的标识、名称，`project_members`、`project_user_properties` 的项目加账户，`states` 的项目加名称）和两个"每个项目至多一个"（`states_project_id_default_key`、`states_project_id_triage_key`）。
- **名字和种类**：`server/migrations/project_schema_test.go` 的 `projectNames`（52 个）并进 `TestConstraintAndIndexNames` 的期望（`schema_test.go` 因此不超过 400 行）；迁移数 13。
- **运行时权限**：`GRANT SELECT, INSERT, UPDATE, DELETE ON projects, project_members, project_user_properties, states TO nerve_runtime`。
- **连带**（第 1 条移交）：`workspace/app.ProjectCascade` 是 `WorkspaceDeleter` 之外的端口；`cascade()` 的最后一步是 `u.projects.DeleteWorkspaceProjects`，同一个 `now`、同一个删除者。`project/app.Cascade.deletion()` 依次是 `DeleteWorkspaceProjects`、`DeleteWorkspaceProjectMembers`、`DeleteWorkspaceProjectPreferences`、`DeleteWorkspaceStates`（3.6 的全局顺序），P7 在最后加标签，不是 `cascade()` 的一步（3.6 的加锁表、11.5）。四条语句形同：

```sql
UPDATE projects
SET deleted_at = $now, updated_at = $now, updated_by_id = $deleted_by
WHERE workspace_id = $workspace_id AND deleted_at IS NULL;
```

  已归档的项目、已结束的成员关系、分诊状态一起删除；此前删除的行保持原来的时间。存储在调用者的事务里执行（`postgres.DB(ctx, pool)`），失败原样返回，删除整个回滚。
- **骨架**：`project.New(Deps)`、`(*Module).Cascade()`、`project.Actions()`；`bootstrap` 在 `workspace.New` 之前建 `project.New`，把它的 `Cascade()` 交给 `workspace.Deps.Projects`（6.6）。`project.NewCascade` 不建（裁定 G2）；`project/app.NewCascade` 是 `New` 用的应用层构造（第 3 节第 21 条）。
- **裁定 S3**：`project` 在建出它的目录的这个 Task 进 `moduleActions`，`Actions()` 先是空的：`TestEveryModuleDeclaresItsActionsOrHasNone` 和 `TestEveryActionHasARuleAndEveryRuleAnAction` 照原样通过，没有豁免；从 Task 7 起每个操作带上它的操作名。"不列 `project`"的变异由这两个测试发现。
- **组合的删除测试**（第 9 条移交）：`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt` 从目录读出四张新表，两个工作区都准备了它们的行，没有加豁免；`TestAFailedProjectsStepRollsTheDeletionBack` 在组合出的 app 上让 `states` 表在请求中改名，最后一条语句失败：500，两个工作区的每一行都不变（完成线"删除工作区的连带回滚测试"）。

### 2.4 领域与 CHECK 的反例（Task 3；3.17–3.19、4.6）

- `CheckNewProject(NewProject) (NewProject, error)`：名称 1–255 个字符、不全是空白、没有 NUL、没有 `& + , : ; $ ^ } { * = ? @ # | ' < > . ( ) % ! -`（`not_allowed`）；标识转成大写后 1–10 个 `A-Z0-9ÇŞĞİÖÜ`；说明没有 NUL；网络 0 或 2，不给时 2；时区经 `shared.ValidTimezone`；`logo_props` 的 `in_use` 是 `emoji` 或 `icon`、五个文本没有 NUL（键和类型由契约守住，第 3 节第 12 条）。全部问题一个 422 `validation_failed`。名称、标识是否被占由数据库说，负责人由用例在锁下判断。
- `CanLead(role)`：管理员或成员，按集合；`DefaultStates()`：Plane 的六个（Backlog 是默认，Triage 是分诊）；`SortOrderFirst(lowest)`：`lowest - 10000`，没有时 `65535`（3.18，Plane `ProjectMember.save`）。
- `TestProjectChecksRejectCounterexamples`：`logo_props` 的四个合法值（`{}`、只有表情、只有图标、三个键都有）通过，4.6 的十个反例（含 Codex S5 的 `{"unexpected": true}`、`{"in_use": 17, "emoji": []}`）都得到 `check_violation`；另有名称（空串、每个禁用字符；反斜杠和中文可以）、标识（小写、空串、11 个、`-`、空格、别的字母；`ÇŞĞİÖÜ0129` 和一个字母可以）、网络、`archive_in`、`last_issue_sequence`、项目角色、显示设置、状态的名称和组的反例和边界。`TestProjectUniqueKeysHoldAmongUndeletedRowsOnly`：每个部分唯一键下第二个未删除的行被拒绝，软删除第一行之后可以再用；别的工作区、项目、账户有自己的键。

### 2.5 存储：插入与读（Task 4）

- `CreateProject`、`CreateMember`、`CreatePreferences`（导航取列的默认值，位置取用例给的）、`CreateStates`（逐行，第一个失败就停并指名）；审计列是用例的时刻和创建者。`projects_workspace_id_identifier_key`、`projects_workspace_id_name_key` 的 23505 翻译为 `project.identifier_taken`、`project.name_taken`（409），别的约束冲突是错误。
- `GetProject(ctx, id, userID)`：未删除的项目（已归档的也读），调用者的角色和侧边栏位置只在他的成员关系有效时，有效成员按成为成员的时刻、再按成员关系的 id：

```sql
SELECT p.…, m.role AS member_role, u.sort_order,
       ARRAY(SELECT a.member_id FROM project_members a
             WHERE a.project_id = p.id AND a.is_active AND a.deleted_at IS NULL
             ORDER BY a.created_at, a.id)::uuid[] AS member_ids
FROM projects p
LEFT JOIN project_members m ON m.project_id = p.id AND m.member_id = $user_id AND m.is_active AND m.deleted_at IS NULL
LEFT JOIN project_user_properties u ON u.project_id = m.project_id AND u.user_id = m.member_id AND u.deleted_at IS NULL
WHERE p.id = $id AND p.deleted_at IS NULL;
```

- `LowestSortOrder(ctx, workspaceID, userID)`：他在这个工作区未删除的显示设置中最小的位置（离开的项目的也算），没有时 `nil`。
- 测试的夹具让丢了谓词的连接先读到别的行（`TestGetProject` 的 `ops` 先存；Task 11 把夹具移进 `newWebFixture`，与列表的测试共用），附录 A 另按 id 升序、降序两个行序各跑一次关键的五个谓词，都被发现。

### 2.6 `WorkspaceDirectory`、`WorkspaceMembers`（Task 5；3.6 约定二、三，6.5）

- `workspace.Provide` 加两个字段，都由 `adapter/postgres/directory.go` 的 `Directory` 实现（第 3 节第 8 条）：
  - `WorkspaceBySlug`：`SELECT id, timezone FROM workspaces WHERE slug = $slug AND deleted_at IS NULL`，不加锁，给读；
  - `ShareWorkspaceBySlug`：同一句加 `FOR SHARE`，建项目的父锁；等锁期间删除的工作区读不到（`TestTheDirectorysLockSeesADeletionItWaitedFor`）；
  - `ShareMembers(ctx, workspaceID, userIDs)`：`WHERE workspace_id = $1 AND member_id = ANY ($2) AND deleted_at IS NULL ORDER BY id FOR SHARE`，已结束的也锁（第 9 条），只答有效成员的角色。
- 跨边界的值 `workspace.DirectoryEntry{ID, Timezone}`，`bootstrap/ports.go` 的 `projectWorkspaces` 把它转成 `project.Workspace`（Task 8、10），`WorkspaceMembers` 直接接上。`workspace` 不导入 `project`，反之亦然。

### 2.7 矩阵的形状（Task 6；9.2；P1 review M8，P2 re-review Minor 2，裁定 G3）

- `matrixRow.columns`：`nil` 是工作区级的六列；项目级的行用 `projectColumns`：PA、PM、PG、PM+WA、WA-、WM-公、WM-私、WG-、P-前 和 X 的三个账户（从来不是成员、已被移出、工作区已删除），共 12 列；已归档项目的一列 `archivedColumns`。一行的每个格子都运行；完整性核对要求一行恰好有它每一列的格子、没有别的列的格子。
- 账户：`matrixAccounts` 11 个（9.2），`accountOf(c)` 让同一个账户在两个级别下各有自己的名字（PG 是工作区的访客、WA- 是它的管理员、WM-公/私 是它的成员、已归档项目的列是 PA）；`TestEveryColumnCallsAsARegisteredAccount` 要求每一列的账户都已注册、每个账户都是某一列的。
- 项目：`matrixProjects`（`acme` 的公开、私密、已归档，`gone` 的，`other` 的）和 `matrixProjectMembers` 在准备之前定名，经项目的存储写入；`projectOf(c)` 是一列指向的项目，`targetViolation` 要求 `{project_id}` 是它。
- **`notTargets` 按参数列出**（裁定 G3）：键是 `notTarget{path, param}`；`targetViolation(pattern, path, c, s, passOver)` 只跳过列出的参数，同一路径上别的参数照常核对。反例（`TestMatrixViolationsCatchesEachColumnGap`）：`checkProjectIdentifier` 列出 `{identifier}` 之后，已删除工作区那一列的 `{slug}` 指向 `acme` 仍然报告；"列出的参数放过整条路径"的变异由它发现。P3 的三个列出（`workspace-slugs/{slug}`、`accept`、`decline` 的 `{invitation_id}`）照新写法。

### 2.8 `createProject`（Task 7、8；3.6、3.17–3.19、5.1）

- 接口：`POST /api/v0/workspaces/{slug}/projects`，`ProjectCreate{name, identifier, description?, network?, project_lead_id?, logo_props?, timezone?}`（`additionalProperties: false`，`LogoProps`、`LogoEmoji`、`LogoIcon` 都是封闭的结构），201 `Project`；码 `[validation_failed, workspace.not_found, forbidden, project.identifier_taken, project.name_taken]`；规则 `project.create`：工作区的管理员和成员。新码 `project.identifier_taken`、`project.name_taken`（409）加进 `PROBLEM_MESSAGES` 和两份 `auth.json`（约束 4）。
- `Project` 的 23 个字段都必有：`project_lead_id`、`default_assignee_id`、`archived_at`、`member_role`、`sort_order` 可为 `null`，`cover_image_url` 在 M5 之前总是 `null`（M2 收尾交接第 7 节），`member_ids` 是数组。
- 用例（`app.NewCreateProject(CreateProjectDeps{…})`）：`CheckNewProject` → 读时钟、取新 id（事务之前，第 10 条）→ 一个事务：`ShareWorkspaceBySlug`（没有是 `workspace.not_found`）→ 判定（看不到换成 `workspace.not_found`，访客 `forbidden`）→ `ShareMembers(创建者[, 负责人])` → 负责人不是有效的管理员或成员：422 `project_lead_id` `not_allowed`（判定之后，约定三：没有权限的人得不到负责人的任何信息）→ `CreateProject` → 每个管理员（创建者、与他不同的负责人）`CreateMember`（20）、`LowestSortOrder`、`CreatePreferences(SortOrderFirst(…))` → 六个状态 → 在事务里 `GetProject(id, 调用者)` 作回答（存下的行，不是用例拼的）。
- 组合出的 app：`TestCreatingAProject`（alice 建上海时区的 `acme`，邀请 bob 加入，以他为负责人建项目：答案、四张表的行；同一个标识的别的大小写、同一个名称各答自己的 409，什么都不写）。矩阵两行（12 格）：`inWorkspace(201, 201, 403)`；负责人不是成员时 `inWorkspace(422, 422, 403)`。

### 2.9 `ProjectAccess` 与 `getProject`（Task 9；3.4、3.19、6.5、8.2；裁定 S2）

- `access.ProjectAccess.ProjectFacts(ctx, projectID, userID) (ProjectFacts, found bool, err)`，`ProjectFacts{WorkspaceID, Public, Member, Role}`；`Authorize` 在 `Target.ProjectID` 不为零时，在调用者的 ctx 里读它：找不到、或它的工作区不是目标的工作区，项目级的规则看不到它（第 8 条移交：读到的行确认属于目标的工作区）。
- `project.Provide(pool)` 只交出 `ProjectAccess`；`ProjectMembershipCounts` 随它的第一个使用者 P5 的 `reactivate-member` 加入（裁定 S2，M3 设计 6.3 的 `module.go` 一行在本 spec 的提交中照改）。`bootstrap/ports.go` 的 `accessProjects` 转换事实。
- 查询：

```sql
SELECT p.workspace_id, p.network, m.role AS member_role
FROM projects p
LEFT JOIN project_members m ON m.project_id = p.id AND m.member_id = $user_id AND m.is_active AND m.deleted_at IS NULL
WHERE p.id = $project_id AND p.deleted_at IS NULL;
```

- 接口：`GET /api/v0/projects/{project_id}`，200 `Project`；码 `[project.not_found]`（新码，加文案）；规则 `project.read`：`LevelVisible`。用例先按 id 读（带调用者的视角），再对它的工作区和项目判定；没有和看不到是同一个 404；不开事务（6.7）。
- 矩阵：项目级 12 格（`ofProject`：PA、PM、PG、PM+WA、WA-、WM-公 200；WM-私、WG-、P-前、X 404 `project.not_found`）和已归档项目 1 格（200，`archived_at` 已填）；答案核对项目、调用者的角色（不是成员时 `null`）和归档。

### 2.10 `checkProjectIdentifier`（Task 10；3.19、5.1；M1-P3 的地址）

- `GET /api/v0/workspaces/{slug}/project-identifiers/{identifier}`（不带结尾 `/`），200 `IdentifierAvailability{available}`；码 `[workspace.not_found, forbidden]`；规则 `project_identifier.check`：与 `project.create` 相同（第 13 条）。
- 用例：`WorkspaceBySlug`（不加锁、不开事务，第 15 条）→ 判定 → `ValidIdentifier` 不通过就答 `available: false`、不问存储 → 否则 `IdentifierTaken(工作区, 大写)`：

```sql
SELECT EXISTS (SELECT 1 FROM projects WHERE workspace_id = $1 AND identifier = $2 AND deleted_at IS NULL);
```

- 矩阵两行（被占的 `web`、空着的 `NEW`，12 格）；`{identifier}` 按参数列为"不是目标"（问的是一个标识），同一路径的 `{slug}` 照常核对。

### 2.11 `listProjects` 与可见性一致（Task 11；3.4、3.12、3.19、9.3）

- `GET /api/v0/workspaces/{slug}/projects?archived=`，200 `ProjectList{data}`；码 `[workspace.not_found]`；规则 `project.list`：工作区的每个有效成员。用例：`WorkspaceBySlug` → 判定 → `ListProjects(工作区, 调用者, VisibilityOf(判定读到的工作区角色), archived)`；不加锁、不开事务。
- `domain.Visibility{All, Public}`：管理员全部，成员另看公开的，访客只看自己加入的（按集合）。查询与 `GetProject` 读同样的列，过滤条件：

```sql
WHERE p.workspace_id = $workspace_id AND p.deleted_at IS NULL
  AND (p.archived_at IS NOT NULL) = $archived
  AND ($sees_all OR m.id IS NOT NULL OR ($sees_public AND p.network = 2))
ORDER BY u.sort_order NULLS LAST, p.name;
```

- 存储：`TestListProjects`（可见性的三种、已归档的和别的、已删除的、别的工作区的、已结束的成员关系，排序）；`TestListProjectsAnswersEachAsGetProjectDoes`：在 `TestGetProject` 的夹具上，每个账户列出的每一行等于 `GetProject` 给他的那一行（成员列表、角色、侧边栏位置），`GetProject` 的每个谓词都由 `TestGetProject` 钉住，列表的同样的列由此钉住（第 3 节第 28 条）。
- `TestListingProjectsIsReadingEach`（完成线的"可见性一致测试"）：矩阵的 11 个账户，未归档、已归档两种，`acme` 的项目中列表列出的恰好是 `getProject` 让他读到的；有的账户两边都是空的、有的两边都是全部（测试要求两种都出现，不会因两边都空而通过）。
- 矩阵两行（12 格，答案核对每一列看到的项目名）。

### 2.12 `DemoteToGuest` 的两处调用（Task 12、13；3.3、3.6 约定五、3.8；裁定 G1）

- `ProjectCascade.DemoteToGuest(ctx, workspaceID, userID, by uuid.UUID, now time.Time) error`（G1）。两步都在调用者持有工作区 `FOR NO KEY UPDATE` 的事务里：

```sql
-- LockMemberProjects：他有未删除成员关系（有效、已结束、已是访客的都算）的未删除项目，已归档的也算
SELECT p.id FROM projects p
WHERE p.workspace_id = $workspace_id AND p.deleted_at IS NULL
  AND EXISTS (SELECT 1 FROM project_members m
              WHERE m.project_id = p.id AND m.member_id = $member_id AND m.deleted_at IS NULL)
ORDER BY p.id
FOR NO KEY UPDATE;
-- DemoteMemberships：一条语句（约定五）；访客的行不动，保留它的审计列
UPDATE project_members
SET role = 5, updated_at = $now, updated_by_id = $updated_by
WHERE project_id = ANY ($project_ids) AND member_id = $member_id AND deleted_at IS NULL AND role <> 5;
```

  没有锁到项目时不写（第 20 条）。`FOR NO KEY UPDATE` 让 P4b 的加入、添加（`FOR SHARE` 项目）等它，外键检查的 `FOR KEY SHARE` 不等（`TestDemotingAMemberToGuest`）；按 id 的顺序由 `TestLockMemberProjectsLocksInIDOrder` 证明（行在表里、在名称和标识的索引里的顺序都与 id 相反）。
- **`updateWorkspaceMember`**：锁之后读一次时钟 → `UpdateMemberRole` → 新角色是访客时 `DemoteToGuest(工作区, 成员, 管理员, 同一个时刻)` → 读资料（P2 review 第 6 节的位置）。
- **接受邀请**：`switch` 中恢复已结束的成员关系改由 `restore(ctx, m, role, now)`：`RestoreMember` 之后、`AcceptInvitation` 之前、同一个事务里，邀请的角色是访客时 `DemoteToGuest(工作区, 他, 他自己, 同一个时刻)`（第 22 条）；有效成员（不改成员关系）和新成员（没有项目成员关系）不调。它让 P4b 加入时 `min(原角色, 现在的工作区角色)` 的上限成立（3.8）。
- 组合出的 app（第 2 条移交、完成线）：`TestDemotingToGuestDemotesInTheWorkspacesProjects`（bob 是 `acme`、`beta` 的成员，各领导一个项目；给项目成员表加一个 `CHECK (role <> 5) NOT VALID` 让项目一步失败，alice 改他的角色答 500、什么都不变；去掉之后成功：他是 `acme` 和它的项目的访客，由 alice 在改角色的时刻写入，`beta` 不变）；`TestAcceptingAsAGuestAgainDemotesInTheWorkspacesProjects`（bob 领导过 Web，照 P5 的离开结束了两个成员关系；以访客再被邀请；失败时接受答 500、邀请仍待接受；成功时他是访客、Web 的成员关系成为访客且仍结束，由他自己写入，邀请已消费）。

### 2.13 每个模块的 HTTP 测试经 `apitest.Main`（Task 8；P1 review 第 6 节 M6，9.4）

- `apitest/main_callers_test.go` 的 `TestEveryModuleRunsItsHTTPTestsThroughMain`：对 `api/modules/` 下有文件的每个模块，用 `go/parser` 读 `server/internal/modules/<模块>/adapter/http/*_test.go`，要求有 `TestMain`，函数体恰好是一句 `apitest.Main(m, "<模块>")`，传它自己的 `*testing.M`。`TestMainViolationsCatchesEachGap` 的反例：没有 `TestMain`、普通的 `TestMain`（`os.Exit(m.Run())`）、别的模块名、函数体多做别的事、传别的 `*testing.M`；照做的包通过。它不能经 `go test -overlay` 变异（读磁盘上的文件），在原型上就地改 `project` 的 `handler_test.go` 验证（附录 A）。
- `project` 的 HTTP 测试回答它声明的每个码，含 `workspace.not_found`（`listProjects`、`createProject`、`checkProjectIdentifier`）和 `forbidden`：`apitest.Main` 两个方向通过（完成线）。

### 2.14 矩阵的格子与耗时（9.2）

| 行 | 列 | 格子 | 答案的核对 |
|---|---|---|---|
| `createProject` | 工作区级六列 | 201、201、403、404 × 3 | `createsItsProject`：`NEW`，调用者是唯一的管理员 |
| `createProject`，负责人不是成员 | 同上 | 422、422、403、404 × 3 | |
| `getProject` | 项目级 12 列 | 200 × 6，`project.not_found` × 6 | `readsItsProject`：这一列的项目、角色、未归档 |
| `getProject`，已归档 | 已归档项目 1 列 | 200 | 同上，`archived_at` 已填 |
| `checkProjectIdentifier`，被占、空着 | 工作区级六列 | 200、200、403、404 × 3 | `identifierAvailable` |
| `listProjects`，未归档、已归档 | 工作区级六列 | 200 × 3、404 × 3 | `listsTheProjects`：每一列看到的项目名 |

共 49 格，加 P1–P3 的 132 格是 181 格；耗时见附录 A。

### 2.15 端到端（Task 14；2 的 P1、W3，9.6；裁定 S4）

- fixture：`api.ts` 的 `Project`、`ProjectCreate`（取自生成的客户端）、`createProject`；`auth.ts` 的 `accountId`；`assert/project.ts` 的 `expectProjectCreated(db, slug, p, creatorEmail, leadEmail, members)`（项目、成员关系和各自的显示设置位置、六个状态，每一行都由创建者在创建的时刻写入、之后没有变）、`countProjects`；`assert/workspace.ts` 的 `workspaceTables` 加上项目的四张表（标签由 P7 加入）。
- **P1 (API)**：成员以管理员为负责人建 Web：答案、四张表的行；标识的检查（之前可用、`WE-B` 不可用；之后 `Web` 不可用、`ops` 可用，陌生人自己工作区的 `web` 可用）；七种拒绝一起发出、每种都不写（标识、名称被占，名称含 `-`、`.`，访客，负责人是访客、是陌生人：他是自己工作区的管理员也不行）；之后管理员建 Ops（他的 55535），成员以管理员为负责人建 Docs（成员 55535、管理员 45535），各按自己的位置（3.18）。
- **W3 (API)**：两个工作区各由管理员以成员为负责人建 Web，每个答案是这个工作区自己的项目；删除 `acme` 之后项目、项目成员、显示设置、状态与工作区在同一时刻删除，`other` 的项目照 `expectProjectCreated` 不变。故事只断言本 Phase 已有的表：W3 和故事 P4 在 P7 加上标签（第 5 节的移交）。
- 故事看得到的谓词见附录 A 的清扫 1；为此 P1 加了陌生人的工作区、两个标识检查和 Ops、Docs，W3 加了每个答案的核对（原型中的修订）。

### 2.16 文档（3.20 的 P4a 一行）

| 文档 | 位置 | 内容 |
|---|---|---|
| 差异清单 | 二·按表 | `projects`（共 11 行）、`project_members`（共 6 行）、`project_user_properties`（5 行）、`states`（5 行）逐列：保留的列数、外键、默认值、CHECK、部分唯一键、删除的列；原来的"新增计数列（列名在 M3 建表时确定）"一行定名为 `last_issue_sequence`，原来的两行删除的列保留 |
| 差异清单 | 四 | "删除工作区"一行加上项目；"接受邀请时已有成员行"写上访客的连带；4.11 中标 P4a 的行（负责人、默认负责人的 `SET NULL` 和创建时的规则，看得到而不是成员时取项目，工作区访客取没加入的公开项目，已归档的项目，项目标识，默认状态、分诊状态） |
| M2 收尾交接 | 文末 | "处理结果（M3/P4a）"：第 7 节的 `cover_image_url`（部分），第 9 节（完成） |
| M1-P2 交接 | 文末 | "处理结果（M3/P4a）"：项目字段（完成） |
| M1-P3 交接 | 文末 | "处理结果（M3/P4a）"：不再读的字段（项目一侧完成）、地址（`project-identifiers` 一条完成） |

## 3. 与设计的差异和补充（待控制者裁定）

以下都没有改变 M3 设计的架构。每条后面写明建议；第 25 条要裁定。

1. **任务的划分：15 个，不是 12 个**。设计的任务与 plan 的对应：1 → 2；2 → 3；3 → 5（目录）、9（`ProjectAccess`，第 4 条）；4 → 6；5 → 4（存储）、7（用例）；6 → 8；7 → 9（`getProject`）、10（`checkProjectIdentifier`）；8 → 11；9 → 1；10 → 12（`updateWorkspaceMember`）、13（接受）；11 → 14；12 → 15（review 是控制者的）。拆分的理由是每个 Task 在约 1,500 行以内并各自是绿的：设计的任务 1 连同存储的插入超过上限，任务 7 两个操作合在一起、任务 10 两处调用合在一起都接近上限。拆 `fakes_test.go` 放在最前面，因为 Task 2 在它旁边加项目的假连带。最大的是 Task 2（1,465 行）、Task 8（1,402 行）和 Task 9（1,300 行）；plan 共 13,016 行。**建议接受。**
2. **移交和裁定的落点**（说明）：

   | 移交 | 落点 |
   |---|---|
   | 1. `cascade()` 最后加项目，经 `WorkspaceDeleter` 以外的端口；组合的删除测试不加豁免；P7 的标签在 `DeleteWorkspaceProjects` 之内 | Task 2（2.3） |
   | 2. `DemoteToGuest` 的两处调用，各有组合出的 app 上的测试 | Task 12、13（2.12） |
   | 3. 每行带自己的列、答案按身份的名字取 | Task 6（2.7） |
   | 3. `notTargets` 按参数，`{identifier}` 列出，反例（G3） | Task 6（规则、反例）、Task 10（列出） |
   | 3. `moduleActions` 加 `project`（S3） | Task 2（空的 `Actions()`，2.3） |
   | 4. `apitest.Main` 两个方向，每个模块 | Task 8（2.13） |
   | 5. `joinProject` 按集合要求工作区角色 | P4b |
   | 6. `fakes_test.go` 按用例拆开 | Task 1 |
   | 7. 锁之后读时钟；建项目说明前后；连带的时刻由调用者给 | Task 7（建项目在事务之前读，第 10 条）；Task 2、12、13（连带取调用者的 `now`，`TestEachWriteReadsTheClockUnderItsLock` 核对） |
   | 8. 锁下的重读确认父行 | Task 5（目录的锁在等待之后重新求值 `deleted_at`）；Task 9（判定只认目标工作区的项目）；Task 12（`LockMemberProjects` 在工作区的锁下按 `workspace_id` 和 `deleted_at` 取项目） |
   | 9. 目录驱动的核对覆盖新表 | Task 2（组合的删除测试准备四张表的行）；`expectNoTokenStored`、`registrationRows` 读每张表，邀请和注册的写不碰项目的表，空表不会藏住它们要找的东西，不需要准备行，没有加豁免 |
   | G1 `DemoteToGuest` 的 `by` 和错误 | Task 12（签名、测试） |
   | G2 不建 `project.NewCascade` | Task 2、12（只有 `New` 的 `Cascade()`；第 21 条） |
   | S2 `Provide` 只交出 `ProjectAccess` | Task 9；M3 设计 6.3（本 spec 的提交） |
   | S4 W3 不含标签 | Task 14；第 5 节的 P7 移交 |
   | S5 `nerve workspaces` 是否建 `project.NewCascade` | 第 5 节的 P5、P6 移交 |

3. **存储的插入移到 Task 4**（设计的任务 1 列了它）。Task 2 连同插入超过约 1,500 行；Task 2 的组合删除测试照表用 SQL 准备项目的行（测试可以直接写 SQL，模块的代码不可以），不经存储。**建议接受。**
4. **`ProjectAccess` 在 Task 9，随它的第一个使用者 `getProject`**（设计的任务 3）。Task 7、8 的规则都是工作区级的，不读项目；在 Task 5 加它会是没有使用者的代码。**建议接受。**
5. **`archive_in` 和默认负责人的规则不在 P4a 的领域**（设计的任务 2 列了 `archive_in`）。P4a 没有写入它们的操作，列有默认值和 CHECK；规则随它们的第一个使用者 P4b 的 `updateProject` 加入。**建议接受；P4b 的 spec 接过去（第 5 节）。**
6. **名称不能全是空白，文本不能含 NUL**：3.19 没有写。Plane 的表单去掉首尾空白之后要求非空；Postgres 的 `text` 和 `jsonb` 存不下 NUL，不检查会是 500。两者都是 422 `validation_failed`（`too_short`、`invalid_format`）。**建议接受。**
7. **`member_ids` 按成为成员的时刻、再按成员关系的 id**：3.12 只说"有序"。同一个事务里建的两个成员关系时刻相同，id（v7）决定顺序，答案稳定。**建议接受。**
8. **`workspace` 的目录是单独的适配器类型 `Directory`**：`Store` 已有 `WorkspaceBySlug`、`ShareWorkspaceBySlug`（答 `workspace` 自己的 `domain.Workspace`），同名的方法答另一种值会冲突；`Directory` 包着同一个 `Store`，查询在 `queries/directory.sql`，名字各自的。**建议接受。**
9. **`ShareMembers` 锁已结束的成员行，只答有效的角色**：约定三锁"这些账户的成员行"；已结束的也锁住，并发的恢复要等它。答案只有有效成员（负责人要求有效）。**建议接受。**
10. **`createProject` 在事务之前读时钟，回答在事务里经 `GetProject` 读回**（第 7 条移交）：插入的都是新行，锁下读到的行没有它必须晚于的时刻，与 P3 的裁定 (c) 相同（`TestCreatingAProjectReadsTheClockBeforeItsTransaction`）。回答读存下的行（带调用者的视角），不是用例拼的值（"答用例拼的值"的变异被发现）。**建议接受。**
11. **矩阵的账户与列的名字、已归档项目的列、每个格子都运行**：9.2 的列名（PA、PM+WA、WM-公 等）在代码里是英文的 `caller` 常量；已归档项目的小表是单独的一列 `archivedColumns`（以 PA 的账户调用）；P1–P3 的运行按工作区级的列逐列，现在运行一行声明的每个格子（按列运行会悄悄跳过别的级别的格子）。**建议接受。**
12. **`logo_props` 的结构由契约守住**：键和类型由 `additionalProperties: false` 和生成的 `bodyshape` 在用例之前答 400（`TestCreateProjectHoldsTheLogoToItsStructure`，四个变异）；领域只查 `in_use` 的取值和 NUL；数据库的 CHECK 兜底。**建议接受。**
13. **操作名 `project_identifier.check`，规则与 `project.create` 相同；不合规的标识答 `available: false`**：9.2 把两者放在同一行；一个 422 会让表单的即时检查与提交的答复不一致，"不可用"是它要的回答，不问存储。**建议接受。**
14. **列表的排序没有 id 作末位**：名称在工作区的未删除项目中唯一（部分唯一键），`(sort_order NULLS LAST, name)` 已经是全序。**建议接受。**
15. **`checkProjectIdentifier`、`listProjects` 不加锁、不开事务**（6.7 的读）：答案是一时的，锁不改变它。**建议接受。**
16. **矩阵为 `other` 准备一个项目**：`acme` 的列表不能列出别的工作区的项目，没有它这个谓词在矩阵里看不到。**建议接受。**
17. **`apitest.Main` 的核对在 `apitest` 自己的测试里**，不在 `bootstrap`：它读 `api/modules/` 和各模块的测试文件，与组合无关。**建议接受。**
18. **`ProjectCreate.project_lead_id` 不可为 `null`**：不传是没有负责人；5.2 的可空只针对 `ProjectUpdate`（区分"没传"和"清空"，P4b）。**建议接受。**
19. **PM+WA 一列的工作区管理员身份在 P4a 看不出来**：P4a 的项目行（`getProject`）对 PM+WA 的回答与 PM 相同（看得到、角色 15），把他准备成工作区成员的变异没有格子能发现；P4b 的修改、添加、归档是第一批区分两者的行。**建议接受；P4b 的矩阵接过去（第 5 节）。**
20. **`DemoteToGuest` 锁的是他有成员关系的全部项目，写只改不是访客的行；没有锁到就不写**：锁不按角色过滤，访客的项目也锁，与 P4b 的加入、添加串行的范围不因他此刻的角色而变；写跳过访客的行，保留它们的审计列；没有项目时不发空的写。**建议接受。**
21. **`project/app.NewCascade(projects, members)`**：这是 `project.New` 用来建 `Cascade` 的应用层构造（两个端口，同一个存储），不是 G2 说的命令行组合用的 `project.NewCascade(CascadeDeps)`，后者不建。**说明。**
22. **接受恢复时 `by` 是接受者自己**：G1 写"接受的账户"；恢复与降级都由他的接受引起，`updated_by_id` 记他。**说明。**
23. **存储把失败吞掉的变异是等价的**（降为访客的锁和写、删除的四步在存储一层）：失败的语句让 Postgres 中止事务，之后的读和提交都失败，写不可能提交。错误原样返回由用例的测试（每个端口的失败）和组合出的 app 的 500 核对；这些变异不列进变异表。**建议接受。**
24. **W3、P1 只断言本 Phase 已有的表**（裁定 S4）：标签在 P7 加入 W3 和故事 P4 的断言。**说明。**
25. **清扫 1 的故事一半：P4a 的故事看不到的谓词**（附录 A 的清扫 1）。P1、W3 运行的新查询共有 33 个谓词。原型加强了两个故事之后：12 个由单独运行的故事在两个行序下都发现；5 个（`GetProject` 的调用者成员关系和显示设置的四个连接谓词、项目的 id）只在一个行序下由故事发现，另一个行序由存储测试 `TestGetProject` 发现（两个行序都证明过，`mutants_rev.py`），要故事在两个行序下都看到，P1 要在建了新项目之后读一个旧项目（`getProject`），那会把 `ProjectFacts` 的谓词也带进 P1 的清扫；15 个是"已删除"、"已结束"的谓词，它们挡的行只有以后的 Phase 的接口能产生（删除项目 P4b、离开和移出 P5；已删除工作区的行同时被判定挡住），由存储测试核对，移交给 P4b、P5 的故事清扫；1 个（`ShareMembers` 的 `member_id`）只影响锁住的行，答案不变，由锁的存储测试核对。**待裁定：接受这个范围，或要求 P4a 的 P1 读一个旧项目（加 `getProject`，连同 `ProjectFacts` 的谓词进 P1 的清扫）。**
26. **清扫 6 的发现，已在原型中改正**：迁移的注释和差异清单原来写 `projects`"保留 23 列"，Plane 的列只保留了 22 个，第 23 个是新加的 `last_issue_sequence`（设计 4.6 写的"36 列 → 23 列"是结果的列数）。两处改为"保留 22 列，另加 `last_issue_sequence`，共 23 列"。**说明。**
27. **最终一轮的变异**：各 Task 的变异写于它的快照；较晚的 Task 改了 4 个变异锚定的代码（`app.go` 的构造、`actions.go` 的列表、`projects.go` 的 handler、`ports.go` 的转换）。最终的原型上这 4 个按同一缺陷重新锚定（`mutants_final.py`），其余照原样再跑一次。**说明。**
28. **写 spec 时补全了清扫 1，Task 11 因此多一个测试**：逐条对照查询写附录 A 时发现 `mutants_a`–`h` 漏了 15 个谓词（`ListProjects` 与 `GetProject` 相同的 11 个：成员列表、调用者的成员关系、显示设置的连接和成员列表的排序；`ProjectFacts` 的项目 id 和成员关系的项目；`IdentifierTaken` 的标识；`LockMemberProjects` 的成员关系的项目）。补成 17 个变异（`mutants_i.py`，`ProjectFacts` 的两个各跑两个行序）之后，10 个被原有的测试发现，`ListProjects` 的 7 个留下：`TestListProjects` 的夹具里没有它们挡住的行（已结束、已删除的成员关系，已删除的显示设置，别人的显示设置，成为成员的时刻与 id 顺序不同的成员）。根因是列表与 `GetProject` 读同样的列而只有后者的测试覆盖这些行；Task 11 加 `TestListProjectsAnswersEachAsGetProjectDoes`，在 `TestGetProject` 的夹具上要求列表的每一行等于 `GetProject` 的那一行，夹具移进 `newWebFixture`（`TestGetProject` 的断言不变）。之后 17 个全部被发现，`TestGetProject` 的 14 个、两个行序的 10 个、列表的 22 个在改过的原型上再跑一次，全部被发现；plan 重新生成、从头复现。**说明。**

## 4. 验收标准（完成线，M3 设计 12 节 P4a）

- [ ] P1 的接口版本、加了项目连带的 W3 通过，此前的每个故事仍然通过（`make e2e` 共 58 个：此前的 57 个，加上 P1）。
- [ ] 可见性一致的测试（`TestListingProjectsIsReadingEach`）通过；`logo_props` 的十个反例、四个合法值和别的 CHECK 反例通过。
- [ ] 删除工作区（`TestAFailedProjectsStepRollsTheDeletionBack`）和降为访客的两处调用（`TestDemotingToGuestDemotesInTheWorkspacesProjects`、`TestAcceptingAsAGuestAgainDemotesInTheWorkspacesProjects`）在组合出的 app 上连带失败时整个写回滚。
- [ ] `project` 的 `apitest.Main` 两个方向通过；每个模块的 HTTP 测试都经 `apitest.Main`（`TestEveryModuleRunsItsHTTPTestsThroughMain`）。
- [ ] 本 Phase 的矩阵格子（49 个）通过，共 181 格，耗时记下；完整性核对（每行的列、`{project_id}` 的目标、按参数的"不是目标"）和它们的反例通过。
- [ ] 13 个迁移 up、down、再 up 通过；四张表的名字、种类、CHECK、部分唯一键由测试核对。
- [ ] `make lint`、`make test`、`make gen-check`、`make knip`、`make test-web`、`make e2e` 通过。
- [ ] 3.20 中 P4a 的一行写好；三份交接有"处理结果（M3/P4a）"。

## 5. 不在 P4a 范围内

- `updateProject`、`deleteProject`、`archiveProject`、`unarchiveProject`、项目的显示设置、项目成员的列出、添加、加入，交错 17，故事 P2、P3、P4、P8：P4b。
- `EndMemberships`、移出、离开，交错 1、4、5、6：P5。停用：P6。标签、默认之外的状态：P7。页面：P9–P11。

**留给后面的 Phase**（各 Phase 的 spec 接过去；P4a 的 review 第 6 节照录）：

| Phase | 条目 |
|---|---|
| P4b | `archive_in`、默认负责人、修改时负责人的规则（第 5 条）；PM+WA 一列在修改、添加、归档行里区分工作区管理员（第 19 条）；`joinProject` 按集合要求工作区角色，在"已是有效成员"之前（第 2 条移交 5，G4）；故事清扫：删除项目让"已删除"的谓词可以准备（第 25 条），P2–P4、P8 单独运行时看得到各自查询的谓词；交错 17 用 `LockMemberProjects` 的锁（`FOR NO KEY UPDATE`，与加入、添加的 `FOR SHARE` 互等）；降为访客与删除项目的交错（`LockMemberProjects` 在等待之后重新求值 `deleted_at`，P4a 只有 Postgres 的语义和目录一侧的测试） |
| P5 | `ProjectCascade.EndMemberships` 是第三个方法（`by` 按 G1）；`ProjectMembershipCounts` 随 `reactivate-member` 加入 `project.Provide`（S2）；离开、移出让"已结束"的谓词可以由故事准备（第 25 条）；S5：`nerve workspaces` 是否建 `project.NewCascade` 由 `workspace.NewAdmin` 实际建什么决定，要连带而不调用时按用途拆开构造 |
| P6 | `project.NewCascade(CascadeDeps)` 随 `nerve users deactivate` 加入（G2），`New` 改用它建同一个实现；S5 同上 |
| P7 | 标签加进 `DeleteWorkspaceProjects`（`deletion()` 的最后一步）；W3 和故事 P4 的断言加上标签（S4） |

## 6. 风险

| 风险 | 应对 |
|---|---|
| 两个连带在调用者的事务里跨模块执行，存储绕开事务时连带与写分开提交 | 存储只经 `postgres.DB(ctx, pool)` 取连接；"在事务之外执行"的变异由组合出的 app 上的回滚测试发现（Task 2）；两处降级也各有组合出的回滚测试 |
| 列表与判定是两处写的同一条可见性规则 | `TestListingProjectsIsReadingEach` 对矩阵的每个账户比较两者，两个方向的改动（查询、`VisibilityOf`、规则）都被发现 |
| `DemoteToGuest` 锁住他所有的项目，项目很多时锁很多行 | 只锁他有成员关系的项目，一条语句按 id 取锁；P4b 的交错 17 在真实数据库上两个顺序都跑 |
| 故事看不到的谓词（第 25 条） | 存储测试在两个行序下都看得到；P4b、P5 的故事清扫接过"已删除"、"已结束"的谓词 |
| plan 的最大 Task 接近上限 | Task 2 是 1,465 行：迁移与删除的连带必须在同一个 Task（组合的删除测试从目录读外键），插入已移到 Task 4 |
| 持续集成（ubuntu）上的运行 | 原型和复现都在 macOS 上；合并之后看持续集成，P1–P3 都是这样合并的 |

## 7. 交接和关闭条件

| 交接 | P4a 处理的条目 | 留下的条目 |
|---|---|---|
| M2 收尾交接第 7 节 可空的引用字段 | `Project.cover_image_url` 必有、可为 `null`（Task 8；"处理结果"在 Task 15） | `IUserLite`：P8；本节保持 `open` |
| M2 收尾交接第 9 节 删除关系图 | 负责人、默认负责人 `SET NULL`（Task 2），登记在差异清单（Task 15） | — |
| M1-P2 项目字段 | 接口和表里没有 `close_in`、`default_state`、`page_view`、`estimate_id`（Task 2、8） | 保留名单的前端一侧（P8）；个人主页的页面（P9） |
| M1-P3 不再读的字段、地址 | 项目接口没有 `anchor`、发布设置；`project-identifiers` 不带结尾 `/`（Task 8、10） | 项目成员、`RESTRICTED_URLS` 与后端同源（前端一侧，P8）和其余各条 |
| P1、P2、P3 review 第 6 节（P4 的各件） | 落点见第 3 节第 2 条 | `joinProject` 按集合：P4b |

**M3 设计 13.1 的关闭条件**（P4a 的各行），逐条核对：

| 条件 | 落点 |
|---|---|
| M2-closeout §7 可空的引用字段（P4a 的一份） | `Project.cover_image_url`：`type: [string, 'null']`、必有；`TestCreateProjectAnswers201` 核对答成 `null` |
| M2-closeout §9 删除关系图 | 4.12 的图不变；`projects_project_lead_id_fkey`、`projects_default_assignee_id_fkey` 的种类 `f n` 由 `TestConstraintAndIndexNames` 钉住；差异清单的 `projects` 一行 |
| M1-P2 项目字段 | `Project`、`ProjectCreate` 没有这四个字段，表里没有这四列 |
| M1-P3 不再读的字段 | 项目接口没有 `anchor` 和发布设置；Nerve 没有项目动态 |
| M1-P3 地址 | `project-identifiers` 不带结尾 `/`（契约的路径） |

P4a 的条件都有落点，没有放不下的。

## 附录 A：原型验证记录（2026-10-01）

原型在 `$M3TMP/p4proto`（`d0796853` 的副本，Go 1.27.1、Node 24、pnpm 11.10.0、Docker；`bin/golangci-lint` 2.13.2）。做法照 P3：每个 Task 做完时存一份源文件的快照（`$M3TMP/p4snap/T1`…`T15`），plan 的代码块由脚本从相邻两份快照的差异生成（`p4tools/mkblocks.py`），脚本把每个 Task 的块应用到前一份快照上，结果与这一份逐字节相同；生成的文件不进块，按 SHA-256 核对（`gensha.py`）。原型的修改都写成编辑脚本（`p4tools/edits/`），清扫之后的改动（故事的加强、列数的更正）从带来它的 Task 起应用到每一份快照和原型。

**逐 Task 复现**（`$M3TMP/p4tools/replay.py`、`replay_all.sh`，日志在 `replay-logs`）：在 `$M3TMP/p4replay`（`d0796853` 的另一个副本，`pnpm install --frozen-lockfile`）按 plan 的 15 个 Task 依次执行：`planapply.mjs` 从 plan 的文本中取出这个 Task 的块写入，然后按顺序执行这个 Task 的每一条 `Run:` 命令，原样照 plan。例外只有：`make gen`、`make gen-go` 之后核对全部生成物与这个 Task 的快照逐字节相同；`shasum -a 256` 的输出与 plan 表中的 SHA-256 和行数核对；副本不是 git 仓库时 `make lint-web` 的关键词守卫由 `kwcheck.mjs` 代替（同样的规则、同样的文件）；`make e2e` 之前把副本初始化为 git 仓库并提交（F4）；提交之后的 `make gen-check` 由生成物的核对代替。每个 Task 另核对 `go.mod`、`go.sum`、`server/tools` 和 `pnpm-lock.yaml` 不变。整个复现 673 秒。下表是第 3 节第 28 条改过 Task 11 之后在新的副本上从头重跑的结果；之前那一轮（`p4replay-v1`、`replay-logs-v1`，672 秒）同样全部通过，与原型 0 个差异。

| Task | 复现的结果 |
|---|---|
| 1 | 写入 5 个文件；`workspace/...` ok；lint 2 × `0 issues.`；`make test` 38 个 `ok` |
| 2 | 30 个文件；`make gen`：24 个生成物与快照相同，5 个 SHA-256 与表相同；`./migrations/`、`project/...`、`workspace/...` ok；组合的删除、回滚、权限文件、`moduleActions`、规则表的测试 ok；lint；`make test` 40 `ok`；`make lint-web`（关键词守卫、turbo 54 个任务）、`make knip`、`make test-web`（16 个任务）通过 |
| 3 | 8 个文件；`project/domain`、`./migrations/` ok；lint；`make test` 41 `ok` |
| 4 | 13 个文件；`make gen-go`：28 个生成物相同，4 个 SHA-256 相同；`project/...` ok；lint；`make test` 41 `ok` |
| 5 | 6 个文件；`make gen-go`：29 个生成物相同，1 个 SHA-256 相同；`workspace/...` ok；lint；`make test` 41 `ok` |
| 6 | 7 个文件；矩阵、完整性、两个反例的测试和账户的核对 ok；lint；`make test` 41 `ok` |
| 7 | 10 个文件；`project/...`、`access/...` ok；规则表、`moduleActions` ok；lint；`make test` 41 `ok` |
| 8 | 18 个文件；`make gen`：31 个生成物相同，4 个 SHA-256 相同；`project/...`、`apitest` ok；`TestCreatingAProject`、目录的转换、矩阵和完整性、路由、请求体 ok；lint；`make test` 42 `ok`；前端检查通过 |
| 9 | 30 个文件；`make gen`：32 个生成物相同，4 个 SHA-256 相同；`access/...`、`project/...` ok；事实的转换、矩阵和完整性、账户、规则表 ok；lint；`make test` 42 `ok`；前端检查通过 |
| 10 | 23 个文件；`make gen`：32 个生成物相同，4 个 SHA-256 相同；`project/...`、`access/...` ok；目录的转换、矩阵和完整性、规则表 ok；lint；`make test` 42 `ok`；前端检查通过 |
| 11 | 21 个文件（含 `projects_test.go` 的 `newWebFixture`）；`make gen`：32 个生成物相同，4 个 SHA-256 相同；`project/...`、`access/...` ok；`-v`：`TestPermissionMatrix` 1.48 秒、`prepare` 0.10 秒；`TestListingProjectsIsReadingEach`、完整性、规则表 ok；lint；`make test` 42 `ok`；前端检查通过 |
| 12 | 17 个文件；`make gen-go`：32 个生成物相同，1 个 SHA-256 相同；`project/...`、`workspace/...` ok；`TestDemotingToGuestDemotesInTheWorkspacesProjects`、P2 的交错 `TestTwoAdminsDemotingEachOtherLeaveAnAdmin` ok；lint；`make test` 42 `ok` |
| 13 | 9 个文件；`workspace/...` ok；两个组合出的降级测试、P3 的交错 `TestAcceptingAndDeletingTheWorkspace`、`TestAcceptingAndChangingTheAddress` ok；lint；`make test` 42 `ok` |
| 14 | 6 个文件；lint；`make test` 42 `ok`；前端检查通过；`make e2e` 58 个通过 |
| 15 | 4 个文件；lint；`make test` 42 `ok`；`make lint-web`（副本此时已是仓库，照原样执行）通过 |
| 结束 | 复现的树与原型逐文件相同（2,927 个文件，0 个差异）；副本上 `make gen-check` 通过（生成物在 Task 14 的提交里，Task 15 只改文档）；`planapply.mjs check` 从 `d0796853` 起 416 个块全部通过 |

| 核对 | 命令 | 结果 |
|---|---|---|
| 生成物 | 原型上 `make gen`，生成物与快照比较；复现的副本上 `make gen-check` | 没有差异 |
| Go 静态检查 | `make lint-go` | server 0 issues；server/tools 0 issues |
| Go 测试 | `make test`（testcontainers，`postgres:18.6`） | 全部通过（41 个包加 `server/tools` 的 1 个；P3 是 37 加 1，新的是 `project` 的四个包） |
| 前端检查 | `make lint-web`（关键词守卫、turbo 54 个任务）；`make knip`；`make test-web`（16 个任务） | 全部通过 |
| 端到端 | `make e2e` | 58 个全部通过（此前的 57 个，W3 加了项目的连带；P1 是新的）；原型不是 git 仓库时只有 S3 失败（F4），复现的副本全部通过 |
| 迁移 | `TestMigrationsGoUpDownAndUpAgain`（13 个迁移）；另在 `nerve-dev-db-1` 的临时库 `p4a_migcheck` 上用原型的 `bin/nerve`（`p4tools/migcheck.sh`）：up、down 四次（00013 → 00010）、再 up，两次 up 之后 `pg_dump --schema-only` 相同，临时库已删除 | 通过 |
| 矩阵 | `go test -count=1 -v -run 'TestPermissionMatrix$'` | 181 格（本 Phase 49 格）；整个矩阵 1.45 秒（原型）、1.47 和 1.48 秒（两轮复现的 Task 11），准备 0.09–0.10 秒；9.2 的预算是 20–30 秒 |
| 变异 | `mutants_a`–`i.py`（311 个，Go 测试；`mutants_final.py` 是其中 4 个重新锚定的）、`mutants_rev.py`（10 个，两个行序）、`guard_inplace.py`（1 个，就地改文件）、`e2e_mutants.py`（3 个，单独运行一个故事）、`e2e_sweep1.py`（33 个谓词，41 次运行） | 见下 |

**原型中定下的事实**：

- **F1** sqlc 把两条查询之间的注释并进后一条查询的说明，并在它的 SQL 里留一个空行：`queries/*.sql` 的分节注释写进每条查询自己的注释（`cascade.sql` 的文件头仍并进第一条，生成物照录）。
- **F2** 一个失败的语句让 Postgres 中止事务：之后的语句（例如读资料）都失败。存储一层"吞掉失败"的变异因此等价（第 3 节第 23 条），错误原样返回在用例一层核对。
- **F3** `ORDER BY … FOR NO KEY UPDATE` 按排序后的行取锁：`LockMemberProjects`、`ShareMembers` 的锁的顺序是 id 的顺序，与行在表里、在索引里的顺序无关；两个测试让行的物理顺序与 id 相反来证明。
- **F4** 原型不是 git 仓库时 S3（实例信息里的 `commit`）失败（P3 附录 A 的 F8）；复现在 `make e2e` 之前把副本初始化为仓库并提交。
- **F5** `FOR NO KEY UPDATE` 与 `FOR SHARE` 冲突、与外键检查的 `FOR KEY SHARE` 不冲突：降为访客锁住项目时，P4b 的加入、添加要等，别的表插入指向这个项目的行不等（`TestDemotingAMemberToGuest` 用 `NOWAIT` 得到 55P03 或不等）。
- **F6** 部分唯一键只管未删除的行，一个账户在同一个项目里可以有已删除和未删除的两个成员关系：`TestDemotingAMemberToGuest` 先建再软删除 bob 在 Web 的一个成员关系，再建他现在的那个，丢了 `deleted_at IS NULL` 的降级写由此被发现（HR 里他的已删除行不在锁住的项目中，写不到它）。
- **F7** 同一次建项目的两个管理员：第二个的 `LowestSortOrder` 读得到第一个刚在同一个事务里插入的显示设置。丢了 `user_id` 的变异因此不只由 P1 发现，W3 单独运行也发现（清扫 1 的故事一半）。

**清扫**（brief 的七类，穷尽地跑在 P4a 新加或修改的代码上；类别写在 `mutants_*.py` 的每一条上）：

| 清扫 | 规模 | 结果 |
|---|---|---|
| 1 每个 SQL 谓词 | P4a 新加的 14 条带条件的查询（删除的四条、`LockMemberProjects`、`DemoteMemberships`、`GetProject`、`ListProjects`、`IdentifierTaken`、`ProjectFacts`、`LowestSortOrder`、目录的三条）的每个 `WHERE`、`JOIN … ON` 的条件和影响答案的 `ORDER BY` 各去掉一个，另有用例丢掉工作区、删除少一步：74 个 Go 变异（其中 17 个是写 spec 时补的，`mutants_i.py`，第 3 节第 28 条）；`GetProject` 的五个关键谓词另按 id 升序、降序两个行序各跑一次（`mutants_rev.py`，10 个）；故事一半：P1、W3 运行的 33 个谓词，各让运行它的故事单独运行，取一行的查询两个行序各一次（41 次运行） | Go 84 个全部被发现（补的 17 个中 7 个要 Task 11 的新测试）；故事：12 个在两个行序下都被发现，5 个只在一个行序下，16 个故事看不到（15 个挡的是以后的 Phase 才能产生的行，1 个只影响锁的范围）；第 3 节第 25 条 |
| 2 每个端口调用的错误 | 用例对端口（存储、锁、目录、跨模块端口、`Authorizer`、事务的提交）的调用失败被吞掉、被答成别的问题：22 个 | 全部被发现：每个失败原样返回，不是别的问题、不被吞掉；存储一层的等价变异见第 3 节第 23 条 |
| 3 每个写不动的行和列 | 每个写的测试都有别的项目、别的成员、别的工作区，处在不同的状态，核对恰好改了哪些行和列：59 个变异（少写、多写一列，改错行，HTTP 的每个字段） | 全部被发现 |
| 4 组合根的每个接线 | 每个接进组合根的端口换成空的或错的：10 个（删除的连带、降级的存储、两处降级的接线、目录的转换、事实的转换、`Authorizer` 找不到项目、路由的注册、操作名、`moduleActions`） | 全部被组合出的 app 上的测试发现 |
| 5 每个安全性质在真实的系统上 | 谁看得到、能做什么的每条规则，每个连带：29 个变异（规则表、判定、用例跳过判定、看不到泄露、可见性）；另有矩阵（真实数据库、组合出的 app）、可见性一致、两处降级和删除的回滚测试 | 全部被发现；没有一个只由假实现守着 |
| 6 每一句说明 | 契约的每段描述、P4a 的代码注释和查询注释、迁移注释、差异清单和交接的每一行，逐句对照代码或测试；3 个契约变异 | 一句不符（`projects`"保留 23 列"，第 26 条），已改正；3 个变异被发现 |
| 7 反例里没有随机 | P4a 新加、修改的测试和故事里没有 `math/rand`、`Math.random`；id 由 `uuid.NewV7()` 按生成的顺序递增，顺序相关的测试按固定的顺序生成；`time.Now()` 只用来给准备的行盖时刻，没有断言依赖它的值；`gen_random_uuid()` 只给没有断言比较的准备行作 id | 没有发现 |

七类之外，brief 的 P4 缺陷类别（连带、可见性、标识、`logo_props`、负责人；加入和添加在 P4b）和 P1–P3 的类别（锁、目录只钉名字、测试工具被削弱、按大小比较角色、精确的错误……）按下面的类别表列出。`mutants_*.py` 每一条的类别标记（Go 311 个）：`sweep1` 74、`sweep2` 22、`sweep3` 59、`sweep4` 10、`sweep5` 29、`sweep6` 3；`p4`（P4 自己的顺序、连带、回答）29、`dom`（领域的规则）27、`check`（CHECK、部分唯一键、生成的请求体结构）15、`lock`（锁的强度、顺序、只读不锁）16、`harness`（矩阵、`apitest` 的核对被削弱）27；另有 `mutants_rev.py` 的 10 个（`sweep1`）和就地改文件的 1 个（`harness`）。

**变异核对**（`mutrun.py`：Go 文件经 `go test -overlay` 改，别的文件就地改并恢复；生成的查询常量代表它的查询；期望点名的测试失败并且输出含它的失败行；编译不过不算）。各 Task 的变异在它的快照上跑过。第 3 节第 28 条改过 Task 11 之后，最终的原型上全部一次跑完（`mutall2.sh`，700 秒）：`mutants_a`–`i` 共 311 个全部被发现（`mutants_a`–`c` 中锚点被较晚的 Task 改了的 4 个跳过，由 `mutants_final.py` 按同一缺陷重新锚定，第 27 条），`mutants_rev` 10 个、`guard_inplace` 1 个全部被发现；之后 `e2e_mutants` 3 个在同一个原型上各让它的故事单独运行，全部被发现，原型与复现的树仍然逐文件相同。下表按缺陷类别：

| 类别 | 变异 | 被哪些测试发现 |
|---|---|---|
| 连带：别的工作区的项目被删除或降级 | 四条删除语句不看 `workspace_id`；降级的锁不看工作区、不看是谁的、哪个项目的成员关系；写不看项目、不看账户 | `TestDeletingAWorkspaceSoftDeletesItsProjects`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`；`TestDemotingAMemberToGuest`、`TestDemotingToGuestDemotesInTheWorkspacesProjects` |
| 连带：一步在事务之外 | 项目的存储在调用者的事务之外 | `TestAFailedProjectsStepRollsTheDeletionBack` |
| 连带：降级到了别人、没有降低每个项目角色 | 锁的参数把工作区的 id 当成员；以成员关系的 id 当账户；写成员的角色；只锁不写；已结束的恢复有效；重写访客的行 | `TestDemotingAMemberToGuest`、`TestDemoteToGuest`、`TestUpdateWorkspaceMemberLocksThenDecidesThenWrites`、`TestAcceptWorkspaceInvitation`、两个组合出的降级测试 |
| 连带：接受恢复为访客而不降级 | 不调；每次恢复都调；新成员、有效成员也调；在恢复之前、在接受之后调 | `TestAcceptWorkspaceInvitation`、`TestAcceptingAsAGuestAgainDemotesInTheWorkspacesProjects`、`TestEachWriteReadsTheClockUnderItsLock` |
| 连带：失败不让整个写回滚 | 删除一步、降级的锁、两处调用的失败被吞掉 | `TestDeleteWorkspaceProjects`、`TestDemoteToGuest`、`TestUpdateWorkspaceMemberFailsWithinTheTransaction`、`TestAcceptWorkspaceInvitationRefusals`；组合出的三个回滚测试 |
| 连带：顺序、时刻、账户 | 删除的步骤顺序；项目一步在工作区行之前；各步另一个时刻；为项目再读时钟；由成员写 | `TestDeleteWorkspaceProjects`、`TestDeleteWorkspace`、`TestEachWriteReadsTheClockUnderItsLock`、`TestDemotingAMemberToGuest`、`TestDemoteToGuest` |
| 可见性：私密项目被不是成员的工作区成员列出、读到 | 列表忽略网络、忽略管理员的可见性、算已结束的成员关系；`VisibilityOf` 让成员看全部、让访客看公开的；`project.read` 只给成员；判定不看有效的工作区成员关系；`ProjectFacts` 算已结束、已删除、别人的、别的项目的成员关系，不看项目的 id（两个行序） | `TestListProjects`、`TestVisibilityOf`、`TestListingProjectsIsReadingEach`、`TestProjectFacts`、`TestDecideAtTheProjectLevels`、`TestPermissionMatrix` |
| 可见性：列表与读取不一致 | 列表按工作区管理员列出；不传 `archived`；查询看反 `archived` | `TestListingProjectsIsReadingEach`（另有矩阵） |
| 可见性：列表的一行与读到的不同 | 列表的成员列表不看项目、有效、未删除，只按 id、只按时刻排；调用者的成员关系不看项目、账户、未删除；显示设置不看项目、账户、未删除 | `TestListProjectsAnswersEachAsGetProjectDoes`（第 3 节第 28 条；其中 7 个 `TestListProjects` 发现不了） |
| 可见性：别的工作区的项目 | 判定认别的工作区的项目；列表列出每个工作区的 | `TestAuthorizeReadsTheTargetsProject`；`TestListProjects`、`TestListingProjectsIsReadingEach` |
| 标识 | 不转大写（领域、检查）；接受 `-`、11 个、空串；检查不合规的也问存储；`IdentifierTaken` 算已删除的、别的工作区的，不看标识；唯一键也管已删除的 | `TestCheckNewProjectAcceptsValidProjects`、`TestCheckNewProjectReportsEveryField`、`TestCheckProjectIdentifier`、`TestIdentifierTaken`、`TestValidIdentifier`、`TestProjectUniqueKeysHoldAmongUndeletedRowsOnly` |
| `logo_props` | CHECK 放宽 `in_use`、`icon.color`、外层的键、非对象；生成的 `bodyshape` 让三层接受任何键、`color` 接受 `null`；领域接受任何 `in_use` | `TestProjectChecksRejectCounterexamples`、`TestCreateProjectHoldsTheLogoToItsStructure`、`TestCheckNewProjectReportsEveryField` |
| 负责人 | 访客可以领导、`CanLead` 按大小；不检查负责人；负责人的成员行不锁；检查在判定之前；负责人不成为成员 | `TestCanLead`、`TestCreateProjectRefuses`、`TestCreateProject`；P1（`e2e_mutants.py`） |
| `Authorize` 在父锁之前、锁在事务之外、锁的强度 | 成员行的锁在判定之前；目录的锁用 `FOR KEY SHARE`、`FOR NO KEY UPDATE`、不加锁；降级的锁用 `FOR SHARE`、`FOR UPDATE`、不加锁、不排序；`ShareMembers` 不排序、倒序、`FOR KEY SHARE`；只读的用例加锁 | `TestCreateProject`、`TestTheDirectorysLockIsForShare`、`TestDemotingAMemberToGuest`、`TestLockMemberProjectsLocksInIDOrder`、`TestShareMembersLocksInIDOrder`、`TestShareMembersLocksTheRowsAskedFor`、`TestCheckProjectIdentifier`、`TestListProjects` |
| 父锁没有 `deleted_at IS NULL` | 目录的两种读、`ShareMembers`、降级的锁去掉它 | `TestWorkspaceDirectoryFindsTheUndeletedWorkspace`、`TestTheDirectorysLockSeesADeletionItWaitedFor`、`TestShareMembersAnswersTheActiveMembersRoles`、`TestDemotingAMemberToGuest` |
| 规则表加行而没有格子、放宽 | `project.create` 给访客、只给管理员；`project.list` 只给管理员和成员；`project_identifier.check` 给访客；操作名不列出；`moduleActions` 不列 `project` | `TestEveryRuleDecidesItsCells`、`TestEveryActionHasARuleAndEveryRuleAnAction`、`TestEveryModuleDeclaresItsActionsOrHasNone`、`TestPermissionMatrix` |
| 存储少写、写错；回答不是存下的行 | 不存负责人、`logo_props`；成员关系存成别的角色；每个状态都是默认；一个状态失败之后接着插入；回答用例拼的值、以"谁都不是"的视角读、在事务之外读 | `TestCreateProjectStoresTheRow`、`TestCreateProjectKeepsTheLogo`、`TestCreateTheRowsUnderAProject`、`TestCreateStatesStopsAtTheFirstFailure`、`TestCreateProject` |
| 精确的错误 | 看不到不换成本资源的 404；读的失败答成 404、答成不可用；拒绝答成 201；标识被占答成名称被占 | 各用例的拒绝测试（`errors.Is` 比较具体的错误）、`TestCreateProjectRefusals`、`TestCreateProjectIdentifierOrNameTaken` |
| 断言不可能失败 | 列以没有注册的账户调用（每格都是 401）；准备只注册工作区级的账户；可见性一致另要求有的账户一个都读不到、有的读到两个，不会因两边都空而通过 | `TestEveryColumnCallsAsARegisteredAccount`、`TestPermissionMatrix`、`TestListingProjectsIsReadingEach` |
| 只有一行 | 存储测试都有第二个项目、账户、工作区；`GetProject` 的五个关键谓词两个行序 | 清扫 1、`mutants_rev.py` |
| 目录只钉名字 | 部分唯一键改成全表的、少一列；CHECK 放宽 | `TestConstraintAndIndexNames`、`TestProjectUniqueKeysHoldAmongUndeletedRowsOnly`、`TestProjectChecksRejectCounterexamples` |
| 按大小比较角色 | `CanLead` 按大小 | `TestCanLead` |
| 矩阵的格子指向别的列的目标 | `{project_id}` 指向别的列的项目也通过、按任何准备过的行核对；列出的参数放过整条路径（G3）；列出的参数照样报告；不在路径里的参数也能列出；每行的列都是工作区级的；放过别的列的格子；WM-私 指向公开项目；已归档的列以别的账户调用；准备数据的角色、网络、归档、以前的成员改错 | `TestMatrixViolationsCatchesEachColumnGap`、`TestMatrixViolationsCatchesEachGap`、`TestEveryColumnCallsAsARegisteredAccount`、`TestPermissionMatrix` |
| 码只在别的模块的测试包里返回（9.4） | 模块文件声明一个没有测试回答的码；`project` 的 HTTP 测试不经 `apitest.Main`；核对不看模块名、函数体、`*testing.M`，没有 `TestMain` 的包通过 | `apitest.Main`；`TestEveryModuleRunsItsHTTPTestsThroughMain`（就地改文件）、`TestMainViolationsCatchesEachGap` |
| 接线没人看 | 删除的连带、降级的存储、两处降级、目录和事实的转换、`Authorizer` 找不到项目、路由不注册、`createProject` 不经 `Authorizer` | `TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`、两个组合出的降级测试、`TestProjectWorkspacesConvertsWorkspacesAnswer`、`TestAccessProjectsConvertsProjectsAnswer`、`TestCreatingAProject`、`TestAPIRoutesAreTheContractsOperations`、`TestPermissionMatrix` |
| 端到端单独运行看得到 | 删除少显示设置一步；Backlog 不是默认；负责人不成为成员；清扫 1 的故事一半 | W3；P1；P1（`e2e_mutants.py`、`e2e_sweep1.py`） |

**故事看得到的谓词**（`e2e_sweep1.py`，每次只运行一个故事；"两序"是按 id 升序、降序各跑一次）：

| 查询 | 谓词 | P1 | W3 |
|---|---|---|---|
| `DirectoryWorkspace` | `slug`（两序） | 两序发现 | — |
| | `deleted_at IS NULL` | 看不到 | — |
| `ShareDirectoryWorkspace` | `slug`（两序） | 降序发现 | 两序发现 |
| | `deleted_at IS NULL` | 看不到 | 看不到 |
| `ShareMembers` | `workspace_id`（两序） | 两序发现 | 看不到 |
| | `member_id`；`deleted_at IS NULL` | 看不到 | 看不到 |
| `LowestSortOrder` | `workspace_id` | 看不到 | 发现 |
| | `user_id` | 发现 | 发现 |
| | `deleted_at IS NULL` | 看不到 | 看不到 |
| `IdentifierTaken` | `workspace_id`；`identifier` | 发现 | — |
| | `deleted_at IS NULL` | 看不到 | — |
| `GetProject` | 成员列表的 `a.project_id` | 发现 | 发现 |
| | `p.id`（两序） | 升序发现 | 升序发现 |
| | `m.project_id`、`u.project_id`（两序） | 升序发现 | 看不到 |
| | `m.member_id`、`u.user_id`（两序） | 降序发现 | 看不到 |
| | `a.is_active`、`a.deleted_at`、`m.is_active`、`m.deleted_at`、`u.deleted_at`、`p.deleted_at` | 看不到 | 看不到 |
| 删除的四条语句 | `workspace_id` × 4 | — | 发现 |
| | `deleted_at IS NULL` × 4 | — | 看不到 |

"看不到"的谓词除 `ShareMembers.member_id`（只影响锁的范围）外，挡的都是已删除、已结束的行，或已删除工作区的行（同时被判定挡住）：P4a 的接口不能产生它们（删除项目在 P4b，离开和移出在 P5）。每一个都由它的存储测试发现（上表的 Go 变异）。

**没有证明的**：

- 持续集成（ubuntu）上的运行：原型和复现都在 macOS 上。
- 第 3 节第 25 条的故事看不到的谓词：由存储测试证明，故事一侧留给 P4b、P5。
- `LockMemberProjects` 在等待之后重新求值 `deleted_at`：P4a 没有删除项目的写，只有 Postgres 的语义和目录一侧的同类测试（`TestTheDirectorysLockSeesADeletionItWaitedFor`）；P4b 的交错接过去。
- 交错 17（P4b）；页面版本（P10）。
