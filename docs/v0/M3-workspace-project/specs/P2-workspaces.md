# M3/P2 工作区的管理和加锁约定：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M3/P2 `workspaces` |
| 日期 | 2026-09-30 |
| 状态 | 进行中 |
| 上级文档 | [M3 设计文档](../M3-design.md) 第 2（W8）、3.4、3.6（约定一、二、五）、3.12、3.14、3.18、3.20（P2 各行）、4.1、4.5、4.11（P2 各行）、4.12、5.1–5.3、6.5、6.6、8.2–8.4、9.1–9.4、9.6、11.2、11.3、12（P2 与约束 4）节；[v0 总体设计](../../v0-design.md) 3.6、4.2 节；[M2 设计](../../M2-auth/M2-design.md) 3.11、3.13、3.14 节 |
| 前置交接 | [P1 review](../reviews/P1-platform-review.md) 第 6 节交给 P2 的 7 件（落点见第 3 节第 2 条）；[M1-P2](../handoffs/M1-P2-trim-content.md) 的侧边栏偏好；[M2 收尾交接](../handoffs/M2-closeout.md) 第 7 节的 `MemberUser.avatar_url` |
| 计划 | [P2 plan](../plans/P2-workspaces.md) |
| 裁定 | 控制者对第 3 节各条的裁定（2026-09-30）和 pre-flight 的七条（M1、M2、L1–L5）全部接受，已落实到本 spec 和 plan：第 3 节各条后的标注、第 15 条；附录 A 的"修订轮" |

本 spec 只写 M3 设计交给 P2 决定的东西：名字、签名、SQL、测试名，以及原型证明了什么。规则本身以 M3 设计为准，这里引用节号，不重述。P2 依赖 P1（`d247b554` 之前的 `main`）。

## 1. 目标

按 M3 设计 12 节 P2：修改、删除工作区，列出成员、改角色，工作区的显示设置；"先锁父行，再判定"第一次落地。具体是：

- 迁移 `00008`（`workspace_user_properties`）和它的运行时权限（4.5）；
- 父行的锁：工作区行的 `FOR NO KEY UPDATE`（按 slug、按 id）和 `FOR SHARE`（按 slug），都带 `deleted_at IS NULL`；写用例的固定步骤"锁 → 判定 → 写"写在一处（`app/lock.go`）（3.6 约定二）；
- 六个用例：`updateWorkspace`、`deleteWorkspace`（连带成员和显示设置）、`listWorkspaceMembers`（`MemberProfiles`，邮箱按角色）、`updateWorkspaceMember`（改自己 409）、`getWorkspacePreferences`、`updateWorkspacePreferences`（`ON CONFLICT`）；
- `identity.Provide` 交出 `PublicProfiles`，在 `bootstrap/ports.go` 转成 `workspace` 的 `MemberProfiles`（6.5、6.6）；
- 交错 2（两位管理员互相降级，两个顺序）；矩阵的 7 行（42 格），矩阵的形状按 P1 review 的移交修改；整程序测试覆盖新操作；
- 测试工具：`archtest` 的"模块只经 sqlc 执行 SQL"、`pgtest.WaitForLockWaitOn`；
- 端到端：W8 的接口版本；3.20 中 P2 的各行（总体设计 3.6、4.2，差异清单）。

## 2. 交付物

### 2.1 文件总览

路径相对于仓库根目录。"生成"表示由 `make gen`（或 `make gen-go`）生成并提交，不手改。"Task"是 plan 中负责它的任务（plan 有完整的文件表）。

| 路径 | 内容 | Task |
|---|---|---|
| `server/migrations/sql/00008_workspace_workspace_user_properties.sql`；`schema_test.go`；`deploy/runtime-grants.sql`；`server/sqlc.yaml` | 显示设置的表；名字、种类、CHECK 的反例、部分唯一键；运行时权限 | 1 |
| `server/internal/archtest/rawsql_test.go`、`rawsql_cases_test.go` | 模块只经 sqlc 执行 SQL | 2 |
| `server/internal/platform/postgres/pgtest/lockwait.go` 及测试；`bootstrap/workspace_test.go` | `WaitForLockWaitOn`；P1 的整程序测试改用它 | 3 |
| `workspace/adapter/postgres/locks.go`、`locks_test.go`；`queries/workspaces.sql` | 按 slug 的两把父行锁；外键检查不等它们 | 4 |
| `server/internal/bootstrap/permission_matrix_test.go`、`permission_matrix_workspace_test.go` | 矩阵的形状 | 5 |
| `workspace/domain/workspace.go`、`app/lock.go`、`app/update_workspace.go`、存储、HTTP、`api/modules/workspace.yaml`；`access/domain/rules.go`；`bootstrap/workspace_test.go`；前端文案 | `updateWorkspace`；`forbidden` 的文案 | 6 |
| `workspace/domain/preferences.go`、`app/ports.go`、`adapter/postgres/preferences.go`、`queries/preferences.sql` | 显示设置的领域、端口、存储 | 7 |
| `workspace/app/get_preferences.go`、`app/update_preferences.go`、`adapter/http/preferences.go`、接口描述 | 读、改显示设置 | 8 |
| `workspace/app/delete_workspace.go`、存储的三步、接口描述 | `deleteWorkspace` | 9 |
| `identity/adapter/postgres/queries/users.sql`、`accounts.go`、`identity/app/accounts.go`、`identity/provide.go`；`bootstrap/ports.go`、`app.go`；`workspace/domain/member.go`、`app/list_members.go`、存储、HTTP、接口描述 | `PublicProfiles`、`MemberProfiles`；`listWorkspaceMembers` | 10 |
| `workspace/adapter/postgres/locks.go`、`workspaces.go`、`queries/`、`update_member_test.go` | 按 id 的锁、`MemberByID`、`UpdateMemberRole` | 11 |
| `workspace/app/update_member.go`、`app/lock.go`、`domain/errors.go`、`domain/member.go`、HTTP、接口描述；矩阵的 `seeded`；`bootstrap/workspace_test.go`；前端文案 | `updateWorkspaceMember`；两个新码 | 12 |
| `server/internal/bootstrap/interleaving_roles_test.go` | 交错 2 | 13 |
| `e2e/fixtures/api.ts`、`assert/workspace.ts`；`e2e/stories/workspace/w8-navigation-preferences.spec.ts` | W8 的接口版本 | 14 |
| `docs/v0/v0-design.md`、`plane-diff.md`；`M3-design.md`；`handoffs/M1-P2-trim-content.md`、`handoffs/M2-closeout.md` | 3.20 的 P2 各行；M3 设计按第 3 节的裁定；交接的处理结果 | 15 |
| `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`、`workspace/adapter/http/gen/*.gen.go`、`workspace/adapter/postgres/gen/*.go`、`identity/adapter/postgres/gen/users.sql.go`（生成） | | 1、4、6–12 |

### 2.2 依赖

没有新依赖（6.1）。`server/go.mod`、`server/tools/go.mod` 和两个 `go.sum` 不变，仍是 `go 1.27` / `toolchain go1.27.1`；不加 npm 包。

### 2.3 迁移 `00008`（4.1、4.5）

`workspace_user_properties` 的 10 列照 4.5；约束和索引的名字：`_pkey`、`_workspace_id_fkey`（`ON DELETE CASCADE`）、`_user_id_fkey`（`CASCADE`）、`_created_by_id_fkey`、`_updated_by_id_fkey`（`SET NULL`）、`_navigation_project_limit_check`（`>= 0`）、`_navigation_control_preference_check`（`ACCORDION`、`TABBED`）、`_workspace_id_user_id_key`（`UNIQUE … WHERE deleted_at IS NULL`，显示设置 `ON CONFLICT` 的目标）、`_workspace_id_idx`（物理级联按它找子行）。默认值 `10`、`'ACCORDION'` 是模型的，`DefaultPreferences()` 由存储测试钉住等于它们。`schema_test.go` 按 P1 修复轮的写法同时钉住名字和种类（`iuw`：带条件的唯一索引），并有部分唯一键的行为测试（P1 review T1-a）。

### 2.4 模块只经 sqlc 执行 SQL（P1 review 第 6 节，整分支评审 M7）

`TestModulesRunSQLOnlyThroughSQLC`：`internal/modules` 下不是测试、不在 `gen` 包里的 Go 文件，不调用两个参数以上的 `Exec`、`Query`、`QueryRow`、`SendBatch`、`CopyFrom`、`Prepare`（pgx 的 `Conn`、`Tx`、`Batch`、`pgxpool` 和 `platform/postgres` 的 `Querier` 都有这些方法）和 `ExecParams`、`ExecPrepared`、`CopyTo`（pgconn 的 `PgConn`，经 `PgConn()` 可达），不含匹配 `\b(?:SELECT\b[\s\S]*\bFROM|INSERT\s+INTO|UPDATE\s+\S+\s+SET|DELETE\s+FROM|TRUNCATE|MERGE\s+INTO)\b` 的字符串字面量（仓库的查询都是大写，小写的文案不算）；解析失败的文件也报；一个存储都找不到时失败。`TestSQLCSchemaScope` 只看得到 sqlc 的查询，直接经 pgx 执行的语句可以读别的模块的表（最容易出现在成员列表：`JOIN users`，6.5 要求经 `MemberProfiles`）。反例在 `TestRawSQLViolationsAreReported` 的表里，每个方法至少一处；另有 `TestRawSQLInTheAppLayerIsReported`：`app` 层（不在 `adapter/postgres` 下）一个文件的字符串和调用都被报告，只查存储的规则过不了它（pre-flight M2）。规则找的是无意写下的 SQL，不是有意的规避：方法值（`run := tx.Query`）、由小写片段拼出的 SQL 不在它的范围内（pre-flight L1）。

### 2.5 `pgtest.WaitForLockWaitOn`（P1 review 第 6 节）

`WaitForLockWaitOn(t, pool, table, limit)`：只在 `pool` 的数据库里有连接在等 `table` 某一行时返回。等一行的连接持有或等待这一行在表上的 tuple 锁，所以查询是 `pg_stat_activity`（`wait_event_type = 'Lock'`）与 `pg_locks`（`locktype = 'tuple' AND relation = to_regclass(table)`）相连。组合出的 app 在同一个数据库上跑 River 的任务（咨询锁），P1 的探测不分表，可能被它们提前满足：P1 的 `TestAnAccountDeactivatedMeanwhileCannotCreateAWorkspace` 改等 `users`，P2 的整程序测试和交错等 `workspaces`。

### 2.6 父行的锁（3.6 约定二）

```sql
-- LockWorkspaceBySlug / LockWorkspace (Task 11) / ShareWorkspaceBySlug
SELECT id FROM workspaces WHERE slug = $1 AND deleted_at IS NULL FOR NO KEY UPDATE;
SELECT id FROM workspaces WHERE id = $1 AND deleted_at IS NULL FOR NO KEY UPDATE;
SELECT id FROM workspaces WHERE slug = $1 AND deleted_at IS NULL FOR SHARE;
```

- 在 ctx 带的事务里执行，锁到事务结束；等锁期间工作区被删除时，Postgres 在提交的新版本上重新求值 `deleted_at IS NULL`，读到 0 行：`app.ErrNotFound`（原型的锁测试）。别的失败是错误，不是 `ErrNotFound`。
- 选哪一把：改工作区行本身、改成员关系的写用 `FOR NO KEY UPDATE`（`updateWorkspace`、`deleteWorkspace`、`updateWorkspaceMember`）；在工作区下加、改行的写用 `FOR SHARE`（`updateWorkspacePreferences`：互不等待，但挡住删除，外键检查的 `FOR KEY SHARE` 挡不住）。
- 强度不超过约定二要的：在工作区下插入一行时，外键检查取工作区行的 `FOR KEY SHARE`，`FOR NO KEY UPDATE`、`FOR SHARE` 都不挡它，`FOR UPDATE` 挡。`TestAForeignKeyCheckDoesNotWaitForTheWorkspaceLocks` 对每把锁核对一次（pre-flight L2）。
- 用例的固定步骤（`app/lock.go`）：`lockAndDecide(ctx, lock, auth, actor, slug, action)` 先锁再判定；`decide(ctx, auth, actor, action, workspaceID, notFound)` 把 `ErrNotVisible` 换成调用方的 404（`workspace.not_found` 或 `workspace.member_not_found`），别的拒绝和失败原样。按资源寻址的写（`updateWorkspaceMember`）：读资源行得到工作区 → 锁工作区 → 重读资源行 → 判定。
- 值的校验（名字、角色的值、上限）在事务之前：它只看请求，不透露工作区的任何事；目标的检查（成员关系已结束、是自己的）在判定之后：没有权限的人得不到目标的任何信息（第 3 节第 3 条；裁定 (d) 接受，M3 设计 6.7、3.6 约定二随 Task 15 写成同样的规则）。

### 2.7 矩阵的形状（9.2；P1 review 第 6 节，整分支评审 M8）

- 行按模块分文件：`permission_matrix_<module>_test.go`，`matrixRows()` 接起来。
- 写格子（和带 `config` 的格子）同时最多 `matrixApps = 8` 个 app（每个 app 的连接池最多 4 个，读格子共用的 app 和 `pgtest` 的管理连接各 4 个，容器的 `max_connections` 是 100）。
- `matrixRow.check(t, c, answer)`：格子的状态码和码都对、而且不是 problem 时核对答案的内容；每个格子在自己里面数应核对的和核对了的答案，`TestPermissionMatrix` 在两者不等时失败；`-run` 只选部分格子时只数被选中的，按格子运行照样通过（pre-flight M1）。P2 的核对：`listWorkspaces`（各列列出的 slug）、`getWorkspace`（调用者自己的角色）、`updateWorkspace`（改名、角色 20）、`listWorkspaceMembers`（四个成员关系，被移出的 `is_active: false`；访客看到的邮箱都是 `null`）、`updateWorkspaceMember`（成员降为访客，带邮箱）、两行显示设置（每列读到、改到自己的）。
- 请求可以指名准备好的行：`request func(c caller, s seeded)`，`seeded.membership(slug, caller)` 是准备数据经存储写入成员关系时记下的 id。
- "工作区已删除"一列：`gone` 由它的管理员经 `DELETE /api/v0/workspaces/gone` 删除（连带成员关系），代替 P1 的 SQL；"已被移出"一列仍是 SQL，由 P5 换成存储。
- 每行带自己的列（项目级十列）、答案按身份的名字取：P4（第 3 节第 2 条）。
- 本 Phase 的行：`updateWorkspace`、`deleteWorkspace`（管理员 200/204，成员、访客 403 `forbidden`，另三列 404 `workspace.not_found`）；`listWorkspaceMembers`、两行显示设置（管理员、成员、访客 200，另三列 404）；`updateWorkspaceMember` 两行：改别人的（管理员 200，成员、访客 403，另三列 404 `workspace.member_not_found`）、改自己的（管理员 409 `workspace.own_membership`，成员、访客 403，另三列 404）。"从来不是成员"一列没有 `acme` 的成员关系，改自己的一格指名成员的成员关系。共 12 行、72 格（P1 的 30 格加 42 格），写的 42 格。

### 2.8 `updateWorkspace`（3.4、3.6、5.1、5.2）

- 接口：`PATCH /api/v0/workspaces/{slug}`，`WorkspaceUpdate{name?, organization_size?, timezone?}`（`additionalProperties: false`：带 `slug` 是 400 `bad_request`，slug 不改），200 答 `Workspace`（角色取自 `Grant`）；码 `[validation_failed, workspace.not_found, forbidden]`。`organization_size` 在修改中不可为 `null`（Plane 的设置表单只能选一个值；建工作区时可以不给）；`WorkspaceUpdate` 的描述写明这一点，`null` 由请求体的结构检查答 400（第 3 节第 9 条，裁定 (e)）。
- `domain.WorkspacePatch{Name, OrganizationSize, Timezone *string}`，`CheckWorkspacePatch` 按建工作区的规则逐个检查设了的字段（与 `CheckNewWorkspace` 共用 `checkOrganizationSize`、`checkTimezone`、`invalid`）。空补丁通过，照样写 `updated_at`、`updated_by_id`（第 3 节第 8 条）。
- 查询：`UPDATE workspaces w SET name = CASE WHEN $set_name THEN $name ELSE w.name END, …, updated_by_id, updated_at WHERE w.id = $id RETURNING …, (有效成员数)`。存储答存下的值；没有这一行是错误（调用者持着它的锁）。

### 2.9 显示设置（3.18、4.5、5.1）

- 接口：`GET`、`PATCH /api/v0/me/workspaces/{slug}/preferences`，`WorkspacePreferences{navigation_control_preference, navigation_project_limit}`，`WorkspacePreferencesUpdate` 两项都可选；码：`GET` `[workspace.not_found]`，`PATCH` `[validation_failed, workspace.not_found]`。规则表两行 `workspace_preferences.read`、`workspace_preferences.update`：任何有效成员，只读写自己的。
- 两个用例各一个文件：`app/get_preferences.go`、`app/update_preferences.go`（6.2、6.3：一个用例一个文件；pre-flight L5）；它们的测试在 `app/preferences_test.go`。
- `GET`：不开事务：`WorkspaceBySlug` → 判定 → `Preferences`；没有行时答 `DefaultPreferences()`（`ACCORDION`、10），不写库（`PreferencesReader` 没有写的方法）。
- `PATCH`：校验 → 一个事务：`ShareWorkspaceBySlug` → 判定 → `UpsertPreferences`：

```sql
INSERT INTO workspace_user_properties AS p (id, workspace_id, user_id, navigation_control_preference,
       navigation_project_limit, created_by_id, updated_by_id, created_at, updated_at)
VALUES ($id, $workspace_id, $user_id, $control, $limit, $user_id, $user_id, $now, $now)
ON CONFLICT (workspace_id, user_id) WHERE deleted_at IS NULL DO UPDATE
SET navigation_control_preference = CASE WHEN $set_control THEN EXCLUDED.navigation_control_preference
                                         ELSE p.navigation_control_preference END,
    navigation_project_limit      = CASE WHEN $set_limit THEN EXCLUDED.navigation_project_limit
                                         ELSE p.navigation_project_limit END,
    updated_by_id = EXCLUDED.updated_by_id, updated_at = EXCLUDED.updated_at
RETURNING navigation_control_preference, navigation_project_limit;
```

  插入的值是默认值加上补丁（存储算出）；两个第一次修改同时发生时，第二个等第一个的索引项，之后改那一行（`TestConcurrentFirstChangesLeaveOneRow`）。已删除的行不算冲突（部分唯一索引）。

### 2.10 `deleteWorkspace`（3.6、3.14、4.12；总体设计 5.5）

- 接口：`DELETE /api/v0/workspaces/{slug}`，204；码 `[workspace.not_found, forbidden]`；规则 `workspace.delete`：管理员。
- 一个事务：`lockAndDecide(LockWorkspaceBySlug)` → `cascade()` 的每一步，同一个 `now`（用例测试的时钟每读一次走一微秒，每一步各读时钟的实现让三步的时间不同；pre-flight L3）、同一个删除者：`DeleteWorkspace`（`UPDATE workspaces SET deleted_at, updated_at, updated_by_id WHERE id = $id AND deleted_at IS NULL`）、`DeleteWorkspaceMembers`、`DeleteWorkspacePreferences`（`… WHERE workspace_id = $id AND deleted_at IS NULL`，已删除的行保持原来的时间；一条语句按扫描顺序加锁，都在工作区的 `FOR NO KEY UPDATE` 之下，约定五）。提交之后记 INFO `workspace deleted`（`workspace_id`、`user_id`）。不清任何人的 `last_workspace_id`（3.14）。
- `cascade()` 是连带的唯一列表，后面的 Phase 各加一步：P3 在工作区行之后加邀请，P4 在最后加项目（经 `ProjectCascade`，由 `WorkspaceDeleter` 以外的端口提供，3.3），P7 加标签。P4 的 `DeleteWorkspaceProjects` 同样带删除者（`by`）。

### 2.11 `listWorkspaceMembers` 与 `MemberProfiles`（3.4、5.1、5.2、6.5、6.6、8.4）

- `identity`：`PublicProfiles` 查询 `SELECT id, email, first_name, last_name, display_name FROM users WHERE id = ANY($1::uuid[]) ORDER BY id`：不加锁（约定一），不看 `is_active`；`identityapp.PublicProfile`；`identity.Provided.PublicProfiles`。
- `bootstrap/ports.go` 的 `workspaceProfiles` 把它转成 `workspace.MemberProfiles`（6.6 第 2 步之后、`workspace.New` 之前），`workspace.Deps.Profiles`。
- `workspace`：`domain.Membership`、`MemberUser{ID, DisplayName, FirstName, LastName, Email *string}`、`Member`；`SeesEmails(role)`：管理员、成员（按集合比较）。
- 用例：不开事务：`WorkspaceBySlug` → 判定（`workspace_member.list`：任何有效成员）→ `ListMembers`（未删除的，含已结束的，`ORDER BY created_at, id`）→ 一次 `PublicProfiles`（列出的成员）；邮箱按调用者的角色，访客看到的全部是 `null`，他自己的也是（第 3 节第 6 条；裁定 (b) 接受，M3 设计 9.2 的一格随 Task 15 改为"`email` 都为 `null`"）；成员没有账户是错误（外键保证它不发生），不是部分列表。
- 接口：`GET /api/v0/workspaces/{slug}/members`，`WorkspaceMemberList{data: [WorkspaceMember]}`；`WorkspaceMember{id, workspace_id, role, is_active, created_at, member: MemberUser}`；`MemberUser{id, display_name, first_name, last_name, avatar_url, email}`，`avatar_url`、`email` 必有、可为 `null`（`avatar_url` 在 M5 之前总是 `null`）。

### 2.12 `updateWorkspaceMember`（3.4、3.6、5.1–5.3）

- 接口：`PATCH /api/v0/workspace-members/{workspace_member_id}`（成员关系的 id），`WorkspaceMemberUpdate{role}`，200 答 `WorkspaceMember`；码 `[validation_failed, workspace.member_not_found, forbidden, workspace.own_membership]`；规则 `workspace_member.update`：管理员。
- 新码：`workspace.member_not_found`（404，"The member does not exist, or you cannot see the workspace."）：不存在、已删除、已结束的成员关系，看不到的工作区的，同一个 404（8.2）；`workspace.own_membership`（409，"You cannot change your own membership."）。前端文案表在同一个 Task 加这两个码（设计第 12 节约束 4）。
- 步骤：`CheckMemberRole`（5、15、20 之一；事务之前）→ 一个事务：`MemberByID` → `LockWorkspace(它的工作区)` → `MemberByID` 再读 → `decide(workspace_member.update, member_not_found)` → 已结束：`member_not_found`；是自己的：`own_membership` → `UpdateMemberRole` → 读他的公开资料（经 `MemberProfiles`，提交之前：读失败时事务回滚，改动不留下），邮箱按调用者的角色。
- 降为访客时项目一侧的连带（`DemoteToGuest`）在 P4 随项目加在"写"之后（12 节）。唯一管理员的规则：改别人的角色不会让工作区没有管理员（调用者自己仍是管理员，且不能改自己的）；两位管理员互相降级由锁保证（2.13）。

### 2.13 交错 2（9.3）

`TestTwoAdminsDemotingEachOtherLeaveAnAdmin`，两个顺序。真实的用例，`identity` 的资料、`access` 的 `Authorizer` 照 `bootstrap` 的接法；先的一方的存储在 `UpdateMemberRole` 之前停在闸门上（判定之后、写入之前，持着工作区的 `FOR NO KEY UPDATE`）；后的一方开始后，`WaitForLockWaitOn(…, "workspaces", 5s)` 确认它等在工作区行上，这时两人都还是管理员；闸门打开：先的一方成功，后的一方 403 `forbidden`（锁下判定时已是成员）。每个等待 10 秒为限，闸门不开时在期限失败而不挂住。

### 2.14 端到端（2 的 W8、9.6）

`W8 (API)`：一个账户（PAT）建两个工作区；读到默认值、库里没有行；改成 `TABBED`、3：答案、库里的一行（由这个账户写、未删除）；再读相同（刷新）；只改上限为 0：同一行，另一个字段不变；另一个工作区仍是默认值、没有行；不是成员的账户 404 `workspace.not_found`。`expectPreferences(db, slug, email, want | null)` 在 `e2e/fixtures/assert/workspace.ts`。页面版本在 P9。

### 2.15 文档（3.20 的 P2 各行）

| 文档 | 位置 | 内容 |
|---|---|---|
| 总体设计 | 3.6 | 关联字段的名字带 `_id`；例外：工作区成员内嵌成员的公开资料（`MemberUser`），邮箱按角色（11.3） |
| 总体设计 | 4.2 | 加锁的全局顺序和六条约定（3.6、11.2）；"先锁父行，再判定"从 P2 落地；约定六的接受邀请一段在 P3、停用一段在 P6 核对 |
| 差异清单 | 二·按表 | `workspace_user_properties` 的五行（保留的列、外键的删除行为、默认值和 CHECK、部分唯一索引、删除的四列） |
| 差异清单 | 三 | 关联字段带 `_id`（P1 移交）；显示设置的路径 |
| 差异清单 | 四 | 显示设置的 `GET` 不写库；删除工作区的连带（4.11 中标 P2 的两行） |
| M3 设计 | 3.6 约定二、6.7、9.2 | 第 3 节的裁定：每个写用例只看请求的值的校验在事务之前、依赖行的检查在判定之后（约定二加一句；6.7 的第 5 步分成事务之前的第 0 步和判定之后的第 5 步）；9.2 `listWorkspaceMembers` 的访客一格"`email` 都为 `null`" |
| M1-P2 交接 | 文末 | "处理结果（M3/P2）"：侧边栏偏好的接口一侧、个人主页的数据来源 |
| M2 收尾交接 | 文末 | "处理结果（M3/P2）"：第 7 节中 `MemberUser.avatar_url` 的部分（pre-flight L4） |

## 3. 与设计的差异和补充（控制者已裁定，2026-09-30）

以下都没有改变 M3 设计的架构。每条后面标出控制者的裁定；pre-flight 的七条和它们的落点在第 15 条。

1. **任务的划分：15 个，不是 12 个**。设计的任务与 plan 的对应：1 → 1；2 → 3（探测）、4（按 slug 的两把锁）、6（`lockAndDecide`）、11（按 id 的锁）；3 → 6；4 → 9；5 → 10；6 → 11（存储）、12（用例）；7 → 7（领域和存储）、8（用例和接口）；8 → 13；9 → 每个用例的 Task 加自己的行，5 改矩阵的形状；10 → 没有单独的 Task（下面说明）；11 → 14；12 → 15（review 是控制者的）。另加 2（P1 移交的架构守卫）。显示设置和改角色各拆成两个 Task，因为合在一起时 plan 超过每个 Task 约 1,500 行（原型中 1,401 + 说明、1,604 行）；拆开之后最大的是 Task 12（1,489 行）和 Task 10（1,468 行）。设计的任务 10（整程序测试覆盖新操作）不需要新代码：M2、P1 的整程序测试按契约逐个操作生成用例（路由等于契约的操作、没有令牌答 401、破坏结构的请求体和不能绑定的参数答 400、每个 problem 响应声明它的头），新操作自动在其中；P2 另加两个点名的整程序测试（`TestAnAdminDemotedMeanwhileCannotUpdateTheWorkspace`、`TestAMembershipEndedMeanwhileIsNotFound`）。**裁定 (a)：接受。**
2. **P1 移交的 7 件的落点**（P1 review 第 6 节；说明，没有单独的裁定）：

   | 移交 | 落点 |
   |---|---|
   | 矩阵"工作区已删除"一列换成存储 | Task 9：`gone` 经 `DELETE` 删除（经接口而不是直接调用存储：同一个事务的连带才是"删除的唯一方式"） |
   | "在锁父行之前判定"的变异 | Task 6：`TestAnAdminDemotedMeanwhileCannotUpdateTheWorkspace`（真实数据库、组合出的 app）；变异答 200 而不是 403（附录 A）。改角色一侧：Task 12 的用例测试和 Task 13 的交错 |
   | 差异清单"关联字段带 `_id`" | Task 15：差异清单第三节一行，总体设计 3.6 的说法 |
   | 前端文案表加 `forbidden`，以后每个码同一个 Task 加文案 | Task 6（`forbidden`）、Task 12（`workspace.member_not_found`、`workspace.own_membership`） |
   | 按关系过滤的锁等待探针 | Task 3：`WaitForLockWaitOn`；用在 Task 3（`users`）、4、6、12、13（`workspaces`） |
   | 矩阵：行按模块分文件、写格子的并行上限、可选的答案核对、`request` 拿到准备好的 id | Task 5（前三件）、Task 12（`seeded`）；每行带自己的列留给 P4（P2 的行都是工作区级的六列） |
   | 手写 SQL 的架构守卫 | Task 2 |

3. **值的校验在锁之前，目标的检查在判定之后**：3.6 的"锁 → 判定 → 写"管的是依赖数据库状态的判断。请求本身的值（名字、角色的取值、上限）不依赖任何行，先校验、不开事务（与 P1 的 `createWorkspace` 相同）；成员关系已结束、是自己的，这两个关于目标的检查在判定之后：没有权限的人对任何目标都得到 403（`TestUpdateWorkspaceMemberRefusals` 的三种"a member"）。**裁定 (d)：接受，并照此修改 M3 设计：6.7 的第 5 步分成事务之前的值的校验和判定之后的依赖行的检查，3.6 约定二加一句，管 P2–P7 的每个写用例（Task 15）。**
4. **改角色的答案在事务里读资料**：约定一允许事务中途经 `MemberProfiles` 读账户（不加锁）。放在提交之前，读失败时整个修改回滚，调用者不会收到错误而库里已经改了（`TestUpdateWorkspaceMemberFailsWithinTheTransaction`）。**接受。**
5. **"先判定"的变异由整程序测试证明**（P1 spec 第 3 节第 13 条的后续）：`updateWorkspace` 改为先读工作区、判定、再锁，`TestAnAdminDemotedMeanwhileCannotUpdateTheWorkspace` 得到 200、名字被改；改角色的同类变异由交错 2 两个顺序都发现。说明，没有单独的裁定；附录 A 的 F1（把锁错成 `FOR SHARE` 整程序测试看不出）按"已覆盖"接受：用例测试的调用记录写着锁的名字，存储的冲突测试钉住每把锁；pre-flight 在库上复现了代价：两个并发的修改在 `FOR SHARE` 下死锁，一个答 500，没有错的数据。
6. **访客看到的邮箱都是 `null`，他自己的也是**：5.2、8.4 和 3.4 的"与 Plane 相同"（Plane 给访客的成员列表用不带邮箱的序列化器）。9.2 的表写"别人的 `email` 为 `null`"，字面上允许访客看到自己的；本 Phase 按 5.2 实现（访客从 `GET /me` 读自己的邮箱）。若要按 9.2 的字面，只改 `list_members.go` 一处和两处测试。**裁定 (b)：接受；M3 设计 9.2 的这一格改为"`email` 都为 `null`"（Task 15）。**
7. **成员列表含已结束的成员关系**：5.1 的表写"含已不是有效成员的，带 `is_active`"，M1-P2 交接的个人主页卡片按 `is_active: false` 显示"不是成员"。P2 的 brief 在缺陷类别中写"a removed member present"是缺陷；本 Phase 按设计实现：已删除的行不在，已结束的（被移出的）在、`is_active: false`（`TestListMembers`、矩阵的 `listsTheMembers`）。请裁定 brief 的这一句指的是已删除的行。**裁定 (c)：接受；brief 的那一句指已删除的行，措辞有误。**
8. **空补丁照样写 `updated_at`、`updated_by_id`**：`PATCH` 没有字段时仍是一次修改，不特殊处理（`TestUpdateWorkspace` 的"nothing"一例）。**接受。**
9. **`organization_size` 在修改中不可为 `null`**（2.8）：设计 5.1 只写 `organization_size?`。Plane 的设置表单总是提交一个值；允许 `null` 要用 `nullable.Nullable` 区分"没传"和"清空"，没有使用者。**裁定 (e)：接受；`WorkspaceUpdate` 的描述写明它不能设为 `null`（Task 6）。**
10. **"工作区已删除"一列不再测"已删除的工作区里仍有效的成员关系"**：经 `DELETE` 删除时成员关系一起软删除，这一列的调用者在 `gone` 里已没有成员关系。"成员关系有效而工作区已删除"的判定由 P1 的 `TestActiveRole` 在存储一侧覆盖（它的一对就是这种状态）。**接受。**
11. **矩阵"从来不是成员"一列改自己的一格**：他在 `acme` 没有成员关系，这一格指名成员的成员关系（答 404 `workspace.member_not_found`，与改别人的相同）。**接受。**
12. **显示设置的一格"答案"核对的是调用者自己的**：矩阵的准备数据只给管理员写了设置；成员、访客读到默认值、改到自己的新行。另一个账户的行读不到由存储测试和这两格一起看到。**接受。**
13. **测试补上的缺口**（原型的第一轮变异发现，附录 A）：P2 的 9 个存储读写和 `PublicProfiles` 原来没有"失败是错误、不是回答"的测试（其中 8 个的变异留下，另两个碰巧被"没有这一行"的测试发现）：加 `TestAFailedWriteIsAnError`、`TestAFailedProfilesReadIsAnError`，P1 的读的测试加三个读（它移到 `failures_test.go`，`store_test.go` 留在 400 行以内）；矩阵从不运行核对也能通过：加计数；"在此之前已删除的成员关系保持原来的时间"没有测试：删除的测试加一行；`CheckMemberRole` 只有用例测试：加 `TestCheckMemberRole`（按集合，含 10、16、25）。**接受。**
14. **`platform` 的测试工具**：`pgtest.WaitForLockWaitOn` 只给测试用（`testHelpersOnlyInTests`），`platform` 的生产代码不变（6.1）。**接受。**
15. **pre-flight 的七条**（高 0、中 2、低 5；控制者全部接受，修正的写法照 pre-flight，另有说明的除外）：

   | 条目 | 改动 | Task | 核对（附录 A 的修订轮） |
   |---|---|---|---|
   | M1 矩阵的计数让部分运行失败 | 每个格子在自己里面数；plan 加部分运行的命令 | 5（12 也跑一次） | `-run 'TestPermissionMatrix/(prepare\|getWorkspace)$'` 通过；"从不运行核对"在部分运行上仍失败（0 对 3）；"在父循环里数"被部分运行发现（3 对 20） |
   | M2 "应用层"的反例在存储的路径上 | `TestRawSQLInTheAppLayerIsReported`（`app/` 路径，一个字符串、一次 `Exec`） | 2 | "只查 `adapter/postgres/`"被发现 |
   | L1 pgconn 的方法 | `statementMethods` 加 `ExecParams`、`ExecPrepared`、`CopyTo`；规则的注释写明它不管有意的规避 | 2 | 三个方法各去掉一个，都被发现 |
   | L2 `FOR NO KEY UPDATE` 与 `FOR UPDATE` 分不开 | `TestAForeignKeyCheckDoesNotWaitForTheWorkspaceLocks` | 4（11 加按 id 的锁） | 两把 N 锁各换成 `FOR UPDATE`，都被发现 |
   | L3 "同一个 `now`"不会失败 | 删除的用例测试用每读一次走一微秒的时钟 | 9 | "每一步各读一次时钟"被发现 |
   | L4 M2 收尾交接第 7 节 | "处理结果（M3/P2）"：`MemberUser.avatar_url` | 15 | — |
   | L5 一个文件两个用例 | `app/get_preferences.go`、`app/update_preferences.go` | 8 | — |

   L1 与 pre-flight 的写法有一处不同：pre-flight 的反例只有一处 `ExecParams`，这样去掉 `ExecPrepared` 或 `CopyTo` 的规则仍能通过；本 spec 让同一个反例（表中仍是一条）调用三个方法各一次，三个变异都被发现。

## 4. 验收标准（完成线，M3 设计 12 节 P2）

- [ ] W8 的接口版本通过，此前的 52 个故事仍然通过（共 53 个）。
- [ ] 交错 2 在真实数据库上两个顺序都通过，`-count=5`、`-race` 也通过；四个变异（不锁、先判定、锁在事务之外、共享锁）都让它失败。
- [ ] 本 Phase 的矩阵格子（42 个）通过，共 72 格，耗时记下；矩阵的完整性核对和答案的计数通过，只跑部分格子也通过。
- [ ] 两个点名的整程序测试（先判定、锁下重读）在真实数据库上通过。
- [ ] 8 个迁移 up、down、再 up 通过；新表的名字、种类、CHECK、部分唯一键由测试核对。
- [ ] 架构测试通过（含"模块只经 sqlc 执行 SQL"）。
- [ ] `make lint`、`make test`、`make gen-check`、`make knip`、`make test-web`、`make e2e` 通过。
- [ ] 3.20 中 P2 的各行（总体设计 3.6、4.2，差异清单）写好；M3 设计 3.6、6.7、9.2 按第 3 节的裁定改好；M1-P2、M2 收尾交接有"处理结果（M3/P2）"。

## 5. 不在 P2 范围内

- 邀请（P3）；项目、`ProjectAccess`、降为访客的项目连带（P4）；移出、离开、`reactivate-member`、唯一管理员的离开（P5）；停用（P6）；状态、标签（P7）；前端（P8–P11）。
- 显示设置的页面（`ProjectNavigationDialog` 改调新接口）、成员页、个人主页的页面：P9。

**留给后面的 Phase**（各 Phase 的 spec 接过去）：

| Phase | 条目 |
|---|---|
| P3 | `cascade()` 在工作区行之后加邀请的一步；交错 3（接受邀请与删除工作区）用 `WaitForLockWaitOn(…, "workspaces", …)`；有了邀请，W8 的接口版本可以加第二个成员，单独运行也看得到 `Preferences` 不看 `user_id`（第 6 节） |
| P4 | `cascade()` 在最后加项目（`ProjectCascade`，带 `by`）；`updateWorkspaceMember` 降为访客时的项目连带（在写之后、读资料之前）；矩阵每行带自己的列、答案按身份的名字取（P1 review 第 6 节） |
| P5 | 矩阵"已被移出"一列的 SQL 换成存储；交错 1 照交错 2 的写法 |
| P7 | `cascade()` 加标签 |

## 6. 风险

| 风险 | 应对 |
|---|---|
| 矩阵长到约 400 格，时间超过 9.2 的预算（20–30 秒） | P2 的 72 格 0.2–1.6 秒（写的 42 格各一个副本和一个 app，最多 8 个同时）。P1 的风险"连接数超过 `max_connections`"由 `matrixApps` 处理；去掉它的变异在默认并行度下仍通过（它是资源的上限，不是正确性的性质） |
| W8 单独运行看不到 `Preferences` 不看 `user_id`：P2 没有让第二个账户加入工作区的接口 | 存储测试（`TestPreferencesReadsTheAccountsOwnRow`）和矩阵的两格（成员、访客读到管理员的设置就失败）看得到；P3 之后可以在 W8 加第二个成员 |
| `cascade()` 少一步只有用例测试（假实现记下每一步）看得到：少了成员一步，被删工作区的成员关系在今天的任何读里都看不到（`ActiveRole`、`ListWorkspaces` 都连着工作区） | 用例测试逐步核对参数和顺序；存储测试核对三步的效果；P3–P7 每加一步就在同一个测试里加一行 |
| `lockAndDecide` 的参数是锁的函数：选错锁（`FOR SHARE` 代替 `FOR NO KEY UPDATE`）编译通过；错了的代价是两个并发的修改死锁，一个答 500 | 用例测试的调用记录写着锁的名字；存储的锁测试钉住三把锁的冲突，外键检查的测试钉住它们不强于约定二 |
| 访客自己的邮箱（第 3 节第 6 条）按 5.2 为 `null`，9.2 的字面原来不同 | 已裁定 (b)：接受；9.2 随 Task 15 改为同一个说法 |

## 7. 交接和关闭条件

| 交接 | P2 处理的条目 | 留下的条目 |
|---|---|---|
| M1-P2 侧边栏偏好 | 接口一侧完成：没有 `/sidebar-preferences/`；项目导航偏好经 `GET`、`PATCH /api/v0/me/workspaces/{slug}/preferences`（Task 8、14） | `ProjectNavigationDialog` 改调它：P9 |
| M1-P2 个人主页 | 数据来源：`listWorkspaceMembers` 在矩阵中有行，含已结束的成员关系，访客的邮箱都是 `null`（Task 10） | 页面：P9 |
| P1 review 第 6 节（P2 的 7 件） | 全部，落点见第 3 节第 2 条 | 每行带自己的列：P4 |
| M2 收尾交接第 7 节 可空的引用字段 | `MemberUser.avatar_url` 必有、可为 `null`，M5 之前总是 `null`（Task 10；`updateWorkspaceMember` 的答复同样，Task 12）；"处理结果（M3/P2）"（Task 15） | `cover_image_url`（P4）、`IUserLite`（P8）；本节保持 `open` |

M1-P2 交接追加"处理结果（M3/P2）"（Task 15），状态保持 `open`：保留名单的前端一侧（P8）、项目字段（P4）、个人主页的页面（P9）仍未处理。

## 附录 A：原型验证记录（2026-09-30）

原型在 `$M3TMP/p2proto`（`d247b554` 的副本，Go 1.27.1、Node 24.15.0、pnpm 11.10.0、Docker；`bin/golangci-lint` 2.13.2）。每个 Task 做完时把源文件存一份快照（`$M3TMP/p2snap/T1`…`T15`），plan 的代码块由脚本从相邻两份快照的差异生成（新文件 ````file`，改动的文件最少上下文的 ````old`/````new` 对，每个 `old` 在它之前的块应用之后的文件中恰好出现一次；改动大于文件本身时 ````whole`），脚本把每个 Task 的块应用到前一份快照上，结果与这一份逐字节相同；生成的文件不进块，按 SHA-256 核对。原来的 13 个 Task 中两个超过约 1,500 行，拆开时在原型上重建了两份中间状态（先放存储、后放用例），各自 `make gen-go`、`make lint-go`、`make test` 通过之后才存快照。

**逐 Task 复现**（`$M3TMP/p2tools/replay.py`；修订轮在一个新的副本上从头重跑，下表是修订轮的结果，原来那一轮的副本和日志在 `p2replay-v1`、`replay-logs-v1`）：在 `$M3TMP/p2replay`（`d247b554` 的另一个副本，`pnpm install --frozen-lockfile`）按 plan 的 15 个 Task 依次执行：`planapply.mjs` 从 plan 的文本中取出这个 Task 的块写入，然后按顺序执行这个 Task 的每一条 `Run:` 命令，原样照 plan。例外只有：`make gen`、`make gen-go` 之后核对全部生成物与这个 Task 的快照逐字节相同；`shasum -a 256` 的输出与 plan 表中的 SHA-256 和行数核对；副本不是 git 仓库时 `make lint-web` 的关键词守卫由 `kwcheck.mjs` 代替（同样的规则、同样的文件）；`make e2e` 之前把副本初始化为 git 仓库并提交（F2）；提交之后的 `make gen-check` 由生成物的核对代替。每个 Task 另核对 `go.mod`、`go.sum`、`server/tools` 和 `pnpm-lock.yaml` 不变。

| Task | 复现的结果 |
|---|---|
| 1 | `make gen-go`：19 个生成物与快照相同，`models.go` 的 SHA-256 与表相同；`./migrations/`、`TestTheGrantsFileCoversEveryRelationAndFunction` ok；lint 2 × `0 issues.`；`make test` 38 个 `ok` |
| 2 | `TestRawSQL*` ok（含 `TestRawSQLInTheAppLayerIsReported`）；lint、`make test` 同上 |
| 3 | `pgtest` ok；`TestAnAccountDeactivatedMeanwhileCannotCreateAWorkspace` `-count=3` ok；lint；`make test` 38 `ok` |
| 4 | `make gen-go`：19 个生成物相同，1 个 SHA-256 相同；存储 ok（含 `TestAForeignKeyCheckDoesNotWaitForTheWorkspaceLocks`）；lint；`make test` 38 `ok` |
| 5 | 矩阵三个测试 ok；只跑 `TestPermissionMatrix/(prepare\|getWorkspace)$` 的部分运行 ok；lint；`make test` 38 `ok` |
| 6 | `make gen`：19 个生成物相同，5 个 SHA-256 相同（`WorkspaceUpdate` 的描述改了其中 `api/dist/openapi.yaml`、`server.gen.go`、`schema.gen.ts`）；`workspace/...`、`access/...` ok；两个整程序测试和矩阵 ok；lint；`make test` 38 `ok`；`make lint-web`（关键词守卫、turbo 54 个任务）、`make knip`、`make test-web`（16 个任务）通过 |
| 7 | `make gen-go`：1 个 SHA-256 相同；领域、存储 ok；lint；`make test` 38 `ok` |
| 8、9 | `make gen`：4 个、6 个 SHA-256 相同；各自的包和矩阵 ok；lint；`make test` 38 `ok`；前端检查通过 |
| 10 | `make gen`：5 个 SHA-256 相同；`workspace/...`、`identity` 的存储、`access/...` ok；转换和矩阵 ok；lint；`make test` 38 `ok`；前端检查通过 |
| 11 | `make gen-go`：2 个 SHA-256 相同；存储 ok；lint；`make test` 38 `ok` |
| 12 | `make gen`：4 个 SHA-256 相同；各包、两个整程序测试、矩阵和完整性 ok；部分运行 ok；`-v`：`TestPermissionMatrix` 1.30 秒、`prepare` 0.07 秒；lint；`make test` 38 `ok`；前端检查通过 |
| 13 | 交错 `-count=5 -race` ok；lint；`make test` 38 `ok` |
| 14 | lint；`make test` 38 `ok`；前端检查通过；`make e2e` 53 个通过 |
| 15 | lint；`make test` 38 `ok`；`make lint-web`（副本此时已是仓库，照原样执行）通过 |
| 结束 | 复现的树与原型、与 T15 快照逐文件相同（2,779 个文件，0 个差异）；提交之后的副本上 `make gen-check`、`make lint-web` 通过；`planapply.mjs check` 从 `d247b554` 起 306 个块全部通过 |

| 核对 | 命令 | 结果 |
|---|---|---|
| 生成物 | 原型上 `make gen` 之后与之前逐字节比较（副本不是 git 仓库，`make gen-check` 由它代替）；复现中每个 Task 与快照比较 | 没有差异 |
| Go 静态检查 | `make lint-go` | server 0 issues；server/tools 0 issues |
| Go 测试 | `make test`（testcontainers，`postgres:18.6`） | 全部通过（37 个包加 `server/tools` 的 1 个） |
| 前端检查 | `make lint-web`（关键词守卫、turbo 54 个任务）；`make knip`；`make test-web`（16 个任务） | 全部通过 |
| 端到端 | `make e2e` | 复现（初始化为仓库之后）53 个全部通过（此前的 52 个、W8）；原型不是仓库，52 个通过、只有 S3 失败（F2） |
| 迁移 | `TestMigrationsGoUpDownAndUpAgain`（8 个迁移） | up、down、再 up 通过 |
| 矩阵 | `go test -count=3 -v -run 'TestPermissionMatrix$'`，另单独跑一次 | 72 格；整个矩阵 0.21–1.60 秒（每次运行的第一次最慢；修订轮 0.23–1.31 秒），准备 0.05–0.07 秒 |
| 交错和整程序测试 | `go test -count=5 -race -run 'TestTwoAdminsDemotingEachOtherLeaveAnAdmin\|TestAnAdminDemotedMeanwhileCannotUpdateTheWorkspace\|TestAMembershipEndedMeanwhileIsNotFound'` | 通过（5.8 秒；修订轮 5.2 秒） |
| 变异 | `$M3TMP/p2tools/mutants.py`，84 个（修订轮） | 全部被点名的测试发现（下表） |

**原型中定下的事实**：

- **F1** `FOR SHARE` 也等 `FOR NO KEY UPDATE`：把 `updateWorkspace` 的锁换成 `FOR SHARE`，整程序测试照样等在降级的事务后面、照样 403，看不出来；只有存储的锁测试（`TestTheWorkspaceLocksConflictAsConvention2Says`）和用例测试的调用记录看得到。所以锁的种类在两处钉住（第 6 节）。控制者按"已覆盖"接受；pre-flight 在库上复现：两个并发的修改在 `FOR SHARE` 下，一个以 `40P01`（死锁）失败、答 500，没有错的数据。
- **F2** S3（实例信息里的 `commit`）要求副本是 git 仓库：不是时 `commit` 是 `unknown`，S3 失败（P1 附录 A 的 F3）。复现在 `make e2e` 之前 `git init` 并提交（只在副本里，用 `--git-dir`、`--work-tree`）。
- **F3** 两个第一次修改同时 `INSERT … ON CONFLICT` 时，第二个等第一个的索引项（`pg_locks` 中是 `transactionid` 的等待，不是行锁：`TestConcurrentFirstChangesLeaveOneRow` 用不分表的 `WaitForLockWait`），第一个提交之后它走 `DO UPDATE`，没有唯一冲突的错误。
- **F4** 组合出的 app 在测试库上跑 River：P1 的整程序测试用不分表的探测时，可能被 River 的等待提前满足（没有观察到失败，但它让测试的断言依赖时序）；P2 的整程序测试都用 `WaitForLockWaitOn`。
- **F5** 容器启动偶尔超时（`pgtest: start postgres:18.6 container … context deadline exceeded`）：第二轮变异中一次出现，结果被误算为"发现"。变异脚本改为容器没有起来就重跑，最后一轮 75 个变异没有一个因此失败。

**变异核对**（`$M3TMP/p2tools/mutants.py`：改一处代码，跑点名的测试，恢复；期望这些测试失败，并且输出含点名的失败行；生成的查询常量代表它的查询，改常量等于改 `.sql` 再生成）。第一轮 69 个中 49 个按期望被发现。其余 20 个：10 个是测试的缺口，补上测试（第 3 节第 13 条）之后都被发现：8 个存储的读写失败时被吞掉或答成"没有"（删除的三步、`ListMembers`、`MemberByID`、`Preferences`、`UpsertPreferences`、`PublicProfiles`），矩阵从不运行核对，删除时"已删除的成员关系保持原来的时间"；9 个其实被别的测试发现，是期望写错（子测试名带逗号的 3 个；F1 的 2 个；`UpdateWorkspace`、`UpdateMemberRole` 吞掉错误由它们自己的存储测试发现；`CheckMemberRole` 按范围、连带少成员一步只有用例测试发现，领域包补上了 `TestCheckMemberRole`）；1 个是等价变异（答时钟的时间，见下面的缺陷类别）。加上迁移、码的变异之后，最后一轮 75 个全部被发现（在最终的原型上一次跑完，263 秒）。

**修订轮**（控制者的裁定和 pre-flight 之后，2026-09-30）：pre-flight 在复现的副本上跑了 17 个变异，3 个留下（M2 的"只查 `adapter/postgres/`"、L2 的 `FOR UPDATE`、L3 的"每一步各读一次时钟"）。按第 3 节第 15 条补上测试之后，原型从 T2 起逐个快照重建（`$M3TMP/p2tools/amend/amend.py`、`rebuild.py`：每个快照照原来的改，T6 起重新生成），变异加上这三类、L1 的三个、M1 的两个（部分运行），共 84 个，在修订后的原型上一次跑完，全部被发现（285 秒；`logs/mutants-run4.txt`）。M1 的部分运行在原型上通过。下表是修订轮的：

| 类别（brief） | 变异 | 被哪些测试发现 |
|---|---|---|
| 先判定、再锁 | `updateWorkspace` 先读、`Authorize`，再锁 | `TestUpdateWorkspaceLocksThenDecidesThenWrites`、`TestAnAdminDemotedMeanwhileCannotUpdateTheWorkspace`、`TestPermissionMatrix/updateWorkspace/never_a_member`、`…/removed` |
| | `updateWorkspaceMember` 按第一次读到的判定 | `TestUpdateWorkspaceMemberLocksThenDecidesThenWrites`、`TestTwoAdminsDemotingEachOtherLeaveAnAdmin`（两个顺序） |
| | 锁下不再读成员关系 | `TestUpdateWorkspaceMemberLocksThenDecidesThenWrites`、`TestAMembershipEndedMeanwhileIsNotFound` |
| 交错 2 没有锁 | 不锁工作区 | `TestTwoAdminsDemotingEachOtherLeaveAnAdmin`（两个顺序）、`TestAMembershipEndedMeanwhileIsNotFound`、用例测试 |
| 锁在事务之外 | `updateWorkspace`、`updateWorkspacePreferences` 的锁在事务之前 | 用例测试（假事务记下 `outside tx`） |
| | `updateWorkspaceMember` 的读、锁、再读在事务之前 | 用例测试、`TestTwoAdminsDemotingEachOtherLeaveAnAdmin` |
| | 删除的连带在事务之后、各自提交 | `TestDeleteWorkspaceLocksThenDecidesThenCascades`、`TestAFailedDeletionLeavesTheWorkspace` |
| 锁没有 `deleted_at IS NULL` | 三把锁各一个 | `TestTheWorkspaceLocksFindOnlyAnUndeletedWorkspace`、`TestTheWorkspaceLocksSkipAWorkspaceDeletedWhileTheyWait/<锁>` |
| `FOR SHARE` 与 `FOR NO KEY UPDATE` 互换 | 按 slug 的 N 换 S；S 换 N | `TestTheWorkspaceLocksConflictAsConvention2Says` |
| | 按 id 的 N 换 S | `TestTheWorkspaceLocksConflictAsConvention2Says`、`TestTwoAdminsDemotingEachOtherLeaveAnAdmin` |
| | 按 slug、按 id 的 N 各换成 `FOR UPDATE`（修订轮） | `TestAForeignKeyCheckDoesNotWaitForTheWorkspaceLocks/<锁>` |
| | `updateWorkspace` 用 S；`updateWorkspacePreferences` 用 N | 两个用例测试 |
| 改自己的角色 | 允许改自己的 | `TestUpdateWorkspaceMemberRefusals`、`TestPermissionMatrix/updateWorkspaceMember,_one's_own/admin` |
| | 允许改已结束的 | `TestUpdateWorkspaceMemberRefusals`、`TestAMembershipEndedMeanwhileIsNotFound` |
| | 目标的检查在判定之前；只把"已结束"挪到之前 | `TestUpdateWorkspaceMemberRefusals`、`TestPermissionMatrix/updateWorkspaceMember,_one's_own/member`、`…/guest` |
| 删除的连带 | 少成员一步；少显示设置一步 | `TestDeleteWorkspaceLocksThenDecidesThenCascades`（后者另有 `TestAFailedDeletionLeavesTheWorkspace`） |
| | 每一步各读一次时钟（修订轮） | `TestDeleteWorkspaceLocksThenDecidesThenCascades`（时钟每读一次走一微秒） |
| | 成员、显示设置的一步不看 `workspace_id` | `TestDeletingAWorkspaceSoftDeletesItsRows` |
| | 成员的一步不看 `deleted_at IS NULL` | `TestDeletingAWorkspaceSoftDeletesItsRows` |
| | 矩阵的 `gone` 不删除 | `TestPermissionMatrix/*/workspace_deleted`（问到 `gone` 的 9 格）、`TestPermissionMatrix/listWorkspaces/member` |
| 邮箱与成员列表 | 访客看得到邮箱 | `TestSeesEmails`、`TestListWorkspaceMembersShowsAddressesByRole`、`TestPermissionMatrix/listWorkspaceMembers/guest` |
| | 用例按管理员的角色给邮箱 | `TestListWorkspaceMembersShowsAddressesByRole`、`TestPermissionMatrix/listWorkspaceMembers/guest` |
| | `ListMembers` 列出已删除的；只列有效的；不看工作区；只按 id 倒序 | `TestListMembers`（后两个另有矩阵的三格） |
| | `PublicProfiles` 只读有效的账户 | `TestPublicProfilesReadsTheAccountsAskedFor` |
| | `PublicProfiles` 加 `FOR SHARE` | `TestPublicProfilesDoesNotWaitForTheRowsLock` |
| | 转换丢掉邮箱 | `TestWorkspaceProfilesConvertsIdentitysAnswer`、矩阵的三格 |
| | HTTP 总答 `null` | `TestListWorkspaceMembers`、`TestUpdateWorkspaceMember` |
| `ON CONFLICT`、别人的设置 | 冲突的目标换成 `(id)` | `TestUpsertPreferences`、`TestConcurrentFirstChangesLeaveOneRow`、`TestPermissionMatrix/updateWorkspacePreferences/admin` |
| | 冲突时总是覆盖方式 | `TestUpsertPreferences`、`TestPermissionMatrix/updateWorkspacePreferences/admin` |
| | `Preferences` 不看账户 | `TestPreferencesReadsTheAccountsOwnRow`、`TestPermissionMatrix/getWorkspacePreferences/member`、`…/guest` |
| | `Preferences` 不看工作区；读已删除的行 | `TestPreferencesReadsTheAccountsOwnRow`；`TestUpsertPreferencesAfterTheRowIsDeleted` |
| | 写入时的账户不是调用者 | `TestUpdateWorkspacePreferencesLocksThenDecidesThenWrites` |
| | 默认值与列的默认值不一致（两边各一个） | `TestDefaultPreferencesAreTheColumnDefaults` |
| 规则表 | `workspace.update` 给成员（放宽） | `TestEveryRuleDecidesItsCells`、`TestPermissionMatrix/updateWorkspace/member`、`TestAnAdminDemotedMeanwhileCannotUpdateTheWorkspace` |
| | `workspace_member.list` 去掉访客；`workspace_preferences.update` 只给管理员（收窄） | `TestEveryRuleDecidesItsCells`、矩阵的格子 |
| | 加一行而 `tableCells` 没有 | `TestEveryRuleDecidesItsCells`、`TestEveryActionHasARuleAndEveryRuleAnAction` |
| 按大小比较角色 | `SeesEmails` 用 `>= 15`；`CheckMemberRole` 用 5 到 20 | `TestSeesEmails`（25）；`TestCheckMemberRole`（10、16）、`TestUpdateWorkspaceMemberRefusals` |
| 目录只钉名字 | 唯一索引去掉 `WHERE`；去掉两个 CHECK；外键去掉 `CASCADE` | `TestConstraintAndIndexNames`（前三个另有 `TestUniqueKeysHoldAmongUndeletedRowsOnly`、`TestChecksRejectCounterexamples`，第一个另有 `TestUpsertPreferencesAfterTheRowIsDeleted`） |
| 存储的失败分支 | 9 个读写吞掉错误或答成"没有" | `TestAFailedWriteIsAnError`（6 个写）、`TestAFailedReadIsAnErrorNotAnAnswer`（3 个读） |
| | `PublicProfiles` 吞掉错误 | `TestAFailedProfilesReadIsAnError` |
| 断言不可能失败 | 矩阵从不运行核对；只对 problem 运行核对 | `TestPermissionMatrix`（"answers checked, want …"） |
| | 只跑部分格子时从不运行核对；应核对的格子在父循环里数（修订轮） | 部分运行的 `TestPermissionMatrix`（"0 answers checked, want 3"；"3 answers checked, want 20"） |
| 码只在别的包里返回（9.4） | HTTP 测试不再答 `workspace.own_membership` | `apitest.Main`：`declares problem codes that no test answered through CheckResponse` |
| | `workspace.own_membership` 答 403 | `TestUpdateWorkspaceMemberRefusals`（HTTP）、矩阵的一格 |
| 接线没人看 | 删除的日志在提交之前多记一行 | `TestDeleteWorkspaceLocksThenDecidesThenCascades` |
| 架构守卫被削弱 | 不查 `Exec`；不认 `SELECT`；跳过 `adapter/` | `TestRawSQLViolationsAreReported` |
| | 只查 `adapter/postgres/`（修订轮） | `TestRawSQLInTheAppLayerIsReported` |
| | 不查 `ExecParams`、`ExecPrepared`、`CopyTo`，各一个（修订轮） | `TestRawSQLViolationsAreReported` |
| 探针被削弱 | 不看表；不看锁的种类 | `TestWaitForLockWaitOnSeesOnlyWaitsForItsTablesRows` |
| 校验 | 上限放行 -1 | `TestCheckPreferencesPatch` |

**缺陷类别**（brief 列出的，逐类核对）：

| 类别 | 核对了什么 | 结果 |
|---|---|---|
| 测试在它说的性质去掉之后仍通过 | 上表的每一类 | 84 个变异全部被发现（修订轮） |
| 目录只钉名字不钉种类 | 新表的每个索引标 `u`、`w`，外键标删除行为；部分唯一键有行为测试 | 四个变异都被发现 |
| 按大小比较角色 | `SeesEmails` 的表含 0、10、25；`CheckMemberRole` 含 0、4、10、16、21、25、-5 | 两个变异都被发现 |
| 存储的失败分支 | 每个新的读写在已取消的 ctx 上答 `context.Canceled`，读不答"没有"、不答 `ErrNotFound` | 第一轮留下 8 个，补测试之后 10 个全部被发现 |
| 精确的错误 | 用例、HTTP、存储测试用 `errors.Is` 比较具体的错误；失败不是 404 另外断言 | — |
| 断言不可能失败 | 矩阵每格断言状态码和码，答案的核对在格子里计数；令牌在每个副本上有效 | 四个变异都被发现（两个在部分运行上） |
| 锁的强度（修订轮） | 持有每把锁时，外键检查的 `FOR KEY SHARE` 立即得到 | 两个 `FOR UPDATE` 变异都被发现 |
| 一次删除一个时刻（修订轮） | 删除的用例测试的时钟每读一次走一微秒 | "每一步各读一次时钟"被发现 |
| 假实现忽略参数 | 假实现按参数回答并记下参数（锁的名字、工作区、账户、补丁、时间）；两个用户 × 两个工作区 | 参数换掉的变异（写入的账户）被发现 |
| 只有一行 | 存储测试都有第二个账户、第二个工作区和一行已删除的；W8 有第二个工作区 | `WHERE` 的 9 个变异都被发现；W8 看不到的一个见第 6 节 |
| 测试挂住 | 交错、锁测试每个等待 10 秒为限，`lock_timeout` 之下确定地得到 `55P03` | 不锁的变异在探测的期限（5 秒）失败，不挂住 |
| 闸门在争用区段之外 | 交错的闸门在持锁的事务里（判定之后、写入之前）；`WaitForLockWaitOn` 在放开闸门之前确认对方等在工作区行上 | 去掉锁、锁挪出事务、先判定、共享锁都让交错失败 |
| 说明与代码不符 | 接口描述（400、401 在各码之前另有说明）、代码注释中的设计节号、文档 | 原型中改正：矩阵准备数据的注释（换行）、`organization_size` 的说法 |
| 接线没人看 | `Profiles` 的转换和接线；删除的日志；`Logger` 传给删除 | 转换、日志的变异被发现 |
| 时间不是存下的值 | 存储按 `RETURNING` 回答；P2 没有客户端送来的时间 | 时钟给 UTC 微秒，"答时钟的时间"与存下的值相同，这个变异是等价的，不计 |
| 码只在别的模块的测试包里返回（9.4） | `workspace` 的 HTTP 测试返回它声明的每个码 | 删掉一个的变异被 `apitest.Main` 发现 |

**没有证明的**：

- 持续集成（ubuntu）上的运行：原型和复现都在 macOS 上。
- 矩阵到约 400 格时的耗时和连接数（第 6 节）。
- W8 的页面版本、成员页、个人主页（P9）。
- `cascade()` 加上邀请、项目、标签之后的一个事务（P3、P4、P7）。
