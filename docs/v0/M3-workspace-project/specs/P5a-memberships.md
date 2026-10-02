# M3/P5a 结束与恢复工作区的成员关系：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M3/P5a `memberships` |
| 日期 | 2026-10-02 |
| 状态 | 第 3 节已由控制者裁定；裁定和预检的发现（L1–L7）已改入（2026-10-02） |
| 上级文档 | [M3 设计文档](../M3-design.md) 第 2（W2、W7、W12）、3.3、3.6（加锁表，约定一、二、五、六，方案 E 的代价）、3.7、3.8、3.11、3.20（P5a 一行）、4.11（P5a 各行）、5.1、5.3、6.5、6.6、8.7、9.2、9.3（交错 1、4、5、6，S2，9d）、9.4、12（P5a 与约束 1–4）、13.1、17.2–17.4 节 |
| 前置交接 | [P4b spec](P4b-project-members.md) 第 5 节 P5 一行，[P4b review](../reviews/P4b-project-members-review.md) 第 6、7 节，P4a、P3、P2、P1 的 spec 第 5 节与 review 第 6 节中 P5 的条目（落点见第 3 节第 2 条） |
| 裁定 | 负责人（2026-10-02）：设计中的 P5 拆成 P5a、P5b，依次合并（第 12 节）；退路 A'（恢复的一半移到 P5b）预先批准，本 spec 没有用到（第 3 节第 1 条）；G1 取 (a)：管理命令没有请求期限，等到操作者中断（17.4，README 照 8.7）；G2：移出的检查顺序是重读 → 判定 → 已结束 404 → 自己 409 → 时钟 → 结束一步 |
| 计划 | [P5a plan](../plans/P5a-memberships.md) |

本 spec 只写 M3 设计交给 P5a 决定的东西：名字、签名、SQL、测试名，以及原型证明了什么。规则本身以 M3 设计为准，这里引用节号，不重述。P5a 依赖 P4b（`b5e826b3` 的 `main`，含拆分的设计修改）。

P5a 是评审敏感的一段（M3 设计 12 节约束 3）：结束的连带跨两个模块（`EndMemberships` 在调用时列举、按 id 锁项目、在锁下查规则 2 的项目集合），移出、离开、恢复都是工作区一级的写（取工作区行的 `FOR NO KEY UPDATE`，在它之后读时刻，方案 E 的代价落在它们身上），恢复在账户行的 `FOR SHARE` 之下进行、停用的账户照样恢复（约定六唯一的例外）。谁能移出、离开、恢复是安全性质，唯一管理员的两条规则也是。附录 A 的十八类清扫逐个核对每个这样的测试在它所说的性质去掉之后失败，并记下失败在哪一层（单元、存储、组合、端到端）。

## 1. 目标

按 M3 设计 12 节 P5a：移出工作区成员、离开工作区，两者共用的结束一步，3.7 两条规则在工作区一侧的部分，恢复成员的命令。具体是：

- 两个操作：`removeWorkspaceMember`（`DELETE /api/v0/workspace-members/{workspace_member_id}`）、`leaveWorkspace`（`POST /api/v0/workspaces/{slug}/leave`），各带操作名、规则行、矩阵行（共 25 格，矩阵共 415 格）；它们为 `project` 模块的 `project.sole_admin` 答 409（9.4）；
- 结束一步（`membershipEnd`）：经 `MemberProfiles` 不加锁读他的地址，软删除这个工作区发给它的待接受邀请（3.8，已拒绝的不动），结束他的成员关系（行留着），再经 `ProjectCascade.EndMemberships` 结束他在这个工作区各项目的成员关系；
- `ProjectCascade.EndMemberships`（`project` 的连带的第三个方法）：在调用时找出项目，按 id 升序 `FOR NO KEY UPDATE` 锁住，在锁下查规则 2（他是某个"还有别的有效成员"的项目唯一的有效管理员时 409 `project.sole_admin`，什么都不改），然后一条语句结束；
- 离开的规则 1（工作区唯一的有效管理员不能离开，哪怕只有他一人，409 `workspace.sole_admin`）；
- `nerve workspaces reactivate-member`：账户行 `FOR SHARE` → 工作区 `FOR NO KEY UPDATE` → 读成员关系 → 恢复，角色不变；停用的账户照样恢复并提示下一步；已结束的项目成员关系的个数经 `ProjectMembershipCounts`（`project.Provide`）读出；
- 交错 1 的工作区一侧、4、5、6，两种顺序，`-count=5 -race`，没有 40P01；S2 顺序、9d 顺序（移出、离开各一次）的集成测试；
- 矩阵的"被移出的成员"由存储写出；唯一管理员的一张表；
- 故事 W2、W7、W12 的接口版本；README 部署一节的"恢复被移出的成员"（8.7）、差异清单中 P5a 的三行（3.20、4.11）。

## 2. 交付物

### 2.1 文件总览

路径相对于仓库根目录（`server/internal/` 省略）。"Task"是 plan 中负责它的任务（plan 有完整的文件表）。本 Phase 85 个手写的文件、7 个生成物；没有迁移。跨模块端口多两个方法：`ProjectCascade.EndMemberships` 和 `ProjectMembershipCounts.CountInactive`，参数和回答只有 `uuid`、`time.Time`、`int`，`bootstrap` 直接接上（第 3 节第 5 条）。

| 路径 | 内容 | Task |
|---|---|---|
| `workspace/adapter/postgres/queries/members.sql`、`invitations.sql`、`endings.go`、`endings_test.go` | 结束成员关系、删除待接受的邀请、其余的管理员 | 1 |
| `project/adapter/postgres/queries/cascade.sql`、`cascade.go`、`end_test.go`；`project/app/cascade.go`、`cascade_test.go`、`ports.go`；`project/domain/errors.go`；`project/module.go`；两个交错测试的构造 | `EndMemberships` | 2 |
| `workspace/app/end_membership.go`、`lock.go`、`update_member.go`、`remove_member.go` 及测试、`ports.go`、假实现、`clock_test.go`；`workspace/domain/actions.go`；`access/domain/rules.go` 及测试 | 结束一步；移出的用例 | 3 |
| `api/modules/workspace.yaml`；`workspace/adapter/http/*`；`workspace/module.go`；前端文案；`bootstrap/permission_matrix_workspace_test.go`、`permission_matrix_coverage_test.go`；`bootstrap/ending_test.go`（最终的表形，只有移出一行） | 移出的接口、矩阵、组合 | 4 |
| `api/modules/workspace.yaml`、`api/openapi.yaml`；`workspace/app/leave_workspace.go` 及测试；`workspace/domain/errors.go`；规则；HTTP；前端文案；矩阵的五个文件；`bootstrap/project_write_locks_test.go`；`bootstrap/ending_test.go`（加离开） | 离开；唯一管理员的表；组合出的结束加离开 | 5 |
| `bootstrap/ending_races_test.go`、`ending_connection_test.go`、`ending_test.go`、`project_connection_test.go`、`interleaving_not_found_test.go` | 竞争、锁的强度、事务的连接 | 6 |
| `project/adapter/postgres/queries/members.sql`、`members.go` 及测试；`project/module.go`；`workspace/adapter/postgres/reactivation.go` 及测试；`workspace/app/reactivate_member.go` 及测试；`workspace/domain/errors.go` | `ProjectMembershipCounts`；恢复的用例 | 7 |
| `workspace/admin.go`；`bootstrap/workspaces.go`、`reactivation_test.go`、`reactivation_races_test.go`；`cmd/nerve/workspaces.go` 及测试 | `nerve workspaces reactivate-member` | 8 |
| `bootstrap/interleaving_endings_test.go`、`interleaving_growth_test.go` | 交错 1 的工作区一侧、4 | 9 |
| `bootstrap/interleaving_removal_test.go` | 交错 5、6 | 10 |
| 矩阵的种子、成员行；`bootstrap/demotion_test.go`；替身的注释 | "被移出的成员"由存储写出 | 11 |
| `bootstrap/invitation_rules_test.go` | S2、9d | 12 |
| `e2e/fixtures/api.ts`、`assert/workspace.ts`；`e2e/stories/workspace/w2-landing.spec.ts`、`w7-member-management.spec.ts`、`w3-workspace-settings.spec.ts` | W2、W7 | 13 |
| `e2e/stories/workspace/w12-reactivate-member.spec.ts`；`README.md`；`docs/v0/plane-diff.md` | W12；文档 | 14 |

### 2.2 依赖

不加 Go 模块、npm 包、迁移。P4b 的 `Locks` 不动：P5a 没有项目级的写（第 5 节）。

### 2.3 工作区的存储（Task 1；3.6、3.7 规则 1、3.8、4.3）

三条查询，都在调用者持工作区行的 `FOR NO KEY UPDATE` 时执行，在 `ctx` 带着的事务里：

```sql
-- name: EndMember :execrows
UPDATE workspace_members
SET is_active = false, updated_at = $now, updated_by_id = $ended_by::uuid
WHERE workspace_id = $workspace_id AND member_id = $member_id AND deleted_at IS NULL;

-- name: HasOtherAdmin :one
SELECT EXISTS (SELECT 1 FROM workspace_members
               WHERE workspace_id = $workspace_id AND member_id <> $member_id AND role = 20 AND is_active
                 AND deleted_at IS NULL);

-- name: DeletePendingInvitations :exec
UPDATE workspace_member_invites
SET deleted_at = $now::timestamptz, updated_at = $now, updated_by_id = $deleted_by::uuid
WHERE workspace_id = $workspace_id AND email = $email AND responded_at IS NULL AND deleted_at IS NULL;
```

- `Store.EndMember(ctx, workspaceID, userID, by, now) error`：部分唯一索引保证一对最多一行未删除；不是恰好一行时是一个错误，不是 `app.ErrNotFound`（调用者在锁下读到它有效）。角色、`created_*` 不动（4.3：结束不删除）。
- `Store.DeletePendingInvitations(ctx, workspaceID, email, by, now) error`：已拒绝的（`responded_at` 不为空）不动，已删除的留着自己的时刻。
- `Store.HasOtherAdmin(ctx, workspaceID, userID) (bool, error)`：`role = 20` 是"管理员"这个集合，不按大小比较。
- 测试：`TestEndMember`、`TestDeletePendingInvitations`、`TestHasOtherAdmin`（plan Task 1）。每个谓词都有一个只由它决定的行（另一个工作区、另一个成员、已删除的行，已删除的行存在有效的之前或之后两个行序）；被写的行除了写的列一列不动，其余每行每列不动；被写的行之前由另一个账户写过（清扫 14）。

### 2.4 `ProjectCascade.EndMemberships`（Task 2；3.6 约定五、六，3.7 规则 2，6.5）

```sql
-- name: LockActiveMemberProjects :many
SELECT p.id
FROM projects p
WHERE p.workspace_id = ANY ($workspace_ids::uuid[]) AND p.deleted_at IS NULL
  AND EXISTS (SELECT 1 FROM project_members m
              WHERE m.project_id = p.id AND m.member_id = $member_id AND m.is_active AND m.deleted_at IS NULL)
ORDER BY p.id
FOR NO KEY UPDATE;

-- name: SoleAdmin :one
SELECT EXISTS (
    SELECT 1 FROM project_members m
    WHERE m.project_id = ANY ($project_ids::uuid[]) AND m.member_id = $member_id AND m.role = 20
      AND m.is_active AND m.deleted_at IS NULL
      AND NOT EXISTS (SELECT 1 FROM project_members a
                      WHERE a.project_id = m.project_id AND a.member_id <> m.member_id AND a.role = 20 AND a.is_active
                        AND a.deleted_at IS NULL)
      AND EXISTS (SELECT 1 FROM project_members o
                  WHERE o.project_id = m.project_id AND o.member_id <> m.member_id AND o.is_active AND o.deleted_at IS NULL));

-- name: EndMemberships :exec
UPDATE project_members
SET is_active = false, updated_at = $now::timestamptz, updated_by_id = $ended_by::uuid
WHERE project_id = ANY ($project_ids::uuid[]) AND member_id = $member_id AND is_active AND deleted_at IS NULL;
```

- `project/app.MembershipEnder`（上面三个方法）；`NewCascade(projects ProjectsDeleter, members MemberDemoter, enders MembershipEnder)`。
- `(*Cascade).EndMemberships(ctx, workspaceIDs, userID, by, now) error`：`LockActiveMemberProjects` → 一个都没有时返回 → `SoleAdmin` → 是则 `domain.ErrSoleAdmin`（409 `project.sole_admin`）→ `EndMemberships`。项目在调用时找出，不由调用者给（调用者锁工作区之前提交的增长在其中，约定六）；锁按排好序的行取，顺序就是 id 的顺序；等锁之后 Postgres 在最新版本上重新求值 `deleted_at IS NULL`，等待期间删除的项目不在结果里。已归档的项目照样结束。规则 2 只在锁下的项目集合上查：他是唯一的有效管理员、而那里还有别的有效成员；只有他一人、或另有管理员的项目不算。Plane 查错过的三种集合（工作区成员关系的 id 比项目成员的账户、"只有一个成员"的项目、"只有他一人"的项目）各是 `TestSoleAdmin` 的一个情形。
- `project.Cascade` 接口和 `workspace/app.ProjectCascade` 各多这个方法；`time` 由调用者给（3.3：移出、离开在工作区 N 之后读，第 3 节第 11 条），`by` 是结束者（G1）。
- 测试：`TestEndingAMembersProjectMemberships`、`TestSoleAdmin`、`TestLockActiveMemberProjectsLocksInIDOrder`、`TestLockActiveMemberProjectsLeavesOutAProjectDeletedWhileItWaited`（存储）；`TestEndMemberships`（单元：调用顺序、参数、锁返回的项目原样往下传，顺序不是任何一种排序；每个失败原样返回）。

### 2.5 结束一步（Task 3；3.6 的加锁表、约定一、3.7 规则 2、3.8）

`workspace/app/end_membership.go`：

- `membershipEnd{members MembershipEnder, profiles MemberProfiles, projects ProjectCascade}`；`run(ctx, workspaceID, userID, by, now) error`：`PublicProfiles([userID])`（不加锁，约定一；不是恰好一个是一个错误：外键保证成员有账户）→ `DeletePendingInvitations(workspaceID, 地址, by, now)` → `EndMember` → `projects.EndMemberships([workspaceID], userID, by, now)`。失败原样返回，之后的不执行，调用者的事务回滚，邀请的删除一起回滚（`project.sole_admin` 时邀请仍待接受：`TestAnEndingEndsTheMembershipsAndLeavesNoInvitation`、9d）。
- 顺序照全局的锁顺序：邀请在成员之前，项目在项目成员之前（3.6）。`TestEachLockOfAnEndingIsItsStrength` 在组合一层核对每一步等的表和那时持有的锁。
- 端口：`MembershipEnder`（`DeletePendingInvitations`、`EndMember`）；`ProjectCascade.EndMemberships`。

### 2.6 `removeWorkspaceMember`（Task 3、4；3.3、3.4、3.6、5.1、9.2、9.4）

- 接口：`DELETE /api/v0/workspace-members/{workspace_member_id}`，204 无正文；码 `[workspace.member_not_found, forbidden, workspace.own_membership, project.sole_admin]`。`project.sole_admin` 进 `PROBLEM_MESSAGES` 和两份 `auth.json`（约束 4），`workspace` 的 HTTP 测试为它答 409（9.4、M-2）。
- 操作名 `workspace_member.remove`，规则 `{Level: LevelWorkspace, Roles: [admin]}`。
- 用例（`NewRemoveWorkspaceMember(members MemberRemover, profiles, projects, auth, tx, clock)`），一个事务：`lockedMember`（读成员行 → `LockWorkspace` 的 `FOR NO KEY UPDATE` → 在锁下重读，须仍在那个工作区；从 `updateWorkspaceMember` 提到 `lock.go`，两个用例共用）→ 判定（看不到时 `workspace.member_not_found`）→ 已结束 404 `workspace.member_not_found` → 自己的 409 `workspace.own_membership` → 读时钟 → 结束一步，由调用者。这是 G2 裁定的顺序，与 `updateWorkspaceMember` 相同；两个检查只对通过判定的调用者（有效的管理员）可见，顺序在组合一层看不到（第 3 节第 10 条）。
- 组合：`workspace.New` 的 `RemoveMember`。组合出的 `TestAnEndingEndsTheMembershipsAndLeavesNoInvitation` 从 Task 4 起（`ending_test.go` 的最终表形，只有移出一行；第 3 节第 7 条，内容见 2.7）。

### 2.7 `leaveWorkspace`（Task 5；3.7 规则 1、5.1、9.2）

- 接口：`POST /api/v0/workspaces/{slug}/leave`，204 无正文；码 `[workspace.not_found, workspace.sole_admin, project.sole_admin]`。`workspace.sole_admin`（`workspace/domain.ErrSoleAdmin`）进 `PROBLEM_MESSAGES` 和两份 `auth.json`。
- 操作名 `workspace.leave`，规则 `{Level: LevelWorkspace, Roles: [admin, member, guest]}`。
- 用例（`NewLeaveWorkspace(workspaces WorkspaceLeaver, profiles, projects, auth, tx, clock)`），一个事务：`LockWorkspaceBySlug` 的 `FOR NO KEY UPDATE` → 判定（不是有效成员 404 `workspace.not_found`）→ 他是管理员时 `HasOtherAdmin`，没有则 `workspace.sole_admin`（他是唯一的成员时也是）→ 读时钟 → 结束一步，由他自己。
- 矩阵：唯一管理员的一张表（第 3 节第 6 条）。
- 组合出的测试：`TestAnEndingEndsTheMembershipsAndLeavesNoInvitation`（Task 4 建，Task 5 加离开的一行；移出和离开各一次：bob 是 Ops 唯一的有效管理员时 409 且任何行不变，erin 已结束的管理员成员关系不算另一位；提交时被拒，对它写的每张表各一次，500 且任何行不变；然后 204，只有他一个成员的 Solo 不拒绝，结束的五行由结束者在一个时刻写、各留角色，之前由 dave 写；其余每行不变；已拒绝的邀请不变；`endingWorld.bystanders`、`soleAdmins` 是前提）。这样规则 2 的三半在组合出的 app 上各有反例：已结束的管理员不算（409 一步），只有他一人的项目不拒绝、"另一个成员"不是他自己（204 一步；预检 L6）。`TestTheOnlyAdminCannotLeave`（只有一人的 solo、有成员的 acme 都是 409，任何行不变；有了第二位管理员之后离开）。

### 2.8 竞争、锁的强度、事务的连接（Task 6；brief 清扫 8、9、12、13）

全部在组合出的 app 上（`endingWorld`，或经 `newWorkspaceRoute` 在只有一个连接的池上）：

- `TestAnEndingFindsWhatEndedMeanwhile`：另一个事务持 acme 的 `FOR NO KEY UPDATE` 并改一行；结束已过认证、在 acme 的行上等（`WaitForLockWaitOn`）；另一个提交之后，五个情形都是 404、从不是泄露存在的 403，一行不改：

  | 结束 | 等待期间 | 回答 |
  |---|---|---|
  | alice 移出 bob | bob 的成员关系结束 | 404 `workspace.member_not_found`（已结束） |
  | alice 移出 bob | alice 的成员关系结束 | 404 `workspace.member_not_found`（看不到） |
  | alice 移出 bob | acme 删除 | 404 `workspace.member_not_found`（锁读不到行） |
  | bob 离开 | bob 的成员关系结束 | 404 `workspace.not_found` |
  | bob 离开 | acme 删除 | 404 `workspace.not_found` |

- `TestEachLockOfAnEndingIsItsStrength`：别的事务持 acme（`FOR NO KEY UPDATE`）、发给 bob 的待接受邀请、他在 acme 的成员关系、他在 Ops 的成员关系（各 `FOR SHARE`），逐个放开；每一步用 `lockOn`（P4b 的 `NOWAIT` 阶梯）读出各行最强的锁：

  | 结束等的表 | 那时持有 |
  |---|---|
  | `workspaces` | 什么都没有 |
  | `workspace_member_invites` | acme `FOR NO KEY UPDATE` |
  | `workspace_members` | acme、邀请 `FOR NO KEY UPDATE` |
  | `project_members` | acme、邀请、成员关系、Web、Ops 各 `FOR NO KEY UPDATE`；Site（alice 的项目，他不是成员）没有锁 |

  他的账户从不被锁到 `FOR SHARE` 或更强（停用的 `FOR NO KEY UPDATE` 会等它；地址不加锁读，约定一）；204 之后写下的时刻不早于 acme 被放开（时钟在锁之后读，3.3）。
- `TestTheEndingsRunOnTheirTransactionsConnection`：工作区模块和项目的连带在一个连接的池上，移出和离开各在 5 秒内 204；任何一条语句经连接池就等一个不来的连接，请求在期限时失败。
- `projectRoute` 改成 `moduleRoute`（第 3 节第 8 条）。

### 2.9 `ProjectMembershipCounts`；恢复的用例（Task 7；3.6 的加锁表、约定六，3.11，6.5）

```sql
-- name: CountInactiveMemberships :one
SELECT count(*)
FROM project_members
WHERE workspace_id = $workspace_id AND member_id = $member_id AND NOT is_active AND deleted_at IS NULL;

-- name: ReactivateMember :execrows
UPDATE workspace_members
SET is_active = true, updated_at = $now
WHERE workspace_id = $workspace_id AND member_id = $member_id AND deleted_at IS NULL;
```

- `project.ProjectMembershipCounts`（`CountInactive(ctx, workspaceID, userID) (int, error)`，不加锁）进 `project.Provided`；`workspace/app.ProjectMembershipCounts` 同一个方法。
- `Store.ReactivateMember(ctx, workspaceID, userID, now) error`：角色不变，`updated_by_id` 不动（第 3 节第 4 条）；不是恰好一行是一个错误。
- `ReactivateMember`（`NewReactivateMember(accounts, members MemberReactivator, counts, tx, clock, logger)`），`Execute(ctx, slug, email) (Reactivation, error)`：地址照注册的规则规范化；一个事务：`ShareAccountByEmail`（账户行 `FOR SHARE`，没有时 `workspace.account_not_found`；停用的照样往下）→ `LockWorkspaceBySlug`（没有时 `workspace.slug_not_found`）→ `MemberOf`（没有 `workspace.never_a_member`；有效的：报告，不读时钟、不写、不数）→ 读时钟 → `ReactivateMember` → `CountInactive`；提交之后记一行日志。`Reactivation{Email, Role, AccountActive, AlreadyActive, EndedProjectMemberships}`。
- 两个新码只在命令行（第 3 节第 3 条）。

### 2.10 `nerve workspaces reactivate-member`（Task 8；3.11、6.6、17.4）

- `workspace.AdminDeps.Counts`；`(*Admin).ReactivateMember`。`NewAdmin` 不建项目的连带（恢复不调用它；`project.NewCascade` 随 P6 的 `nerve users deactivate`，6.3、G2）。
- `bootstrap.Workspaces` 的组合多 `project.Provide(pool).ProjectMembershipCounts`；`bootstrap.ReactivateMember(slug, email)` 的输出：

  ```
  reactivated <email> in <slug> as <admin|member|guest>; project memberships still ended: <n>, each restored when the member joins its project
  <email> is an active member of <slug> already; nothing changed
  ```

  账户已停用时行末加 `; the account is deactivated: run nerve users activate --email <email> next`。退出码 1：stderr 一行（码的说明），stdout 为空。
- 命令没有请求期限（G1 (a)，17.4）：`TestAnInterruptedReactivationChangesNothing` 证明等锁时中断回滚、什么都不改，之后重跑成功；README 照 8.7 写明。
- 真实数据库上：`TestWorkspacesReactivateMember`、`TestWorkspacesReactivateMemberErrors`（每个退出码 1 的情形和提交时被拒：每张表不变）、`TestAReactivationFindsWhatChangedMeanwhile`、`TestEachLockOfAReactivationIsItsStrength`、`TestTheReactivationRunsOnItsTransactionsConnection`；命令行：`TestWorkspacesReactivateMemberCommand`。

### 2.11 交错 1、4、5、6（Task 9、10；3.6 约定二、三、六，3.7 规则 1，9.3）

每个交错两种顺序；第一方停在一个 gate（`endedHolding`：结束一步写完成员关系之后、项目一步之前；P4b 的 `gatedShares`：增长或建项目取完他的成员行之后；P4b 的 `gatedDeleter`：删除工作区判定之后），第二方在工作区行上等，由 `pgtest.WaitForLockWaitOn(…, "workspaces", 5*time.Second)` 证明，然后 gate 放开。每个等待都有期限。

| 交错 | 测试 | 第一方 → 第二方 | 结果 |
|---|---|---|---|
| 1（工作区一侧） | `TestTwoAdminsLeavingLeaveAnAdmin` | 两位管理员离开（两种顺序） | 第二位 409 `workspace.sole_admin`，acme 留一位管理员 |
| 4 | `TestARemovalAndTheProjectSidesGrowthSerialize` | 增长（添加、加入；新的、已结束的成员关系）↔ 移出 | 增长先：移出结束他在 Web 的成员关系，时刻不早于 gate 放开；移出先：添加 422 `members[0].member_id` `not_allowed`，加入 404 |
| 5 | `TestARemovalAndTheRemovedMembersProjectSerialize` | 他建 Ops ↔ 移出 | 建先：201，移出结束他在 Ops 的成员关系；移出先：404 `workspace.not_found`，没有 Ops |
| 6 | `TestARemovalAndTheRemovedAdminsDeletionSerialize` | 她删除 acme ↔ 移出她 | 删除先：移出 404 `workspace.member_not_found`；移出先：删除 404 `workspace.not_found` |

**探针的反例**（附录 A；`mutants_probes.py`）：把第二方的那一把锁拿掉，它不再等工作区行：

| 拿掉的锁 | 失败的顺序 | 怎样失败 |
|---|---|---|
| 离开的 `LockWorkspaceBySlug` 锁成 `FOR SHARE`、或不加锁 | 交错 1 的两种顺序 | 探针在期限时失败（第二位不等，两位都离开） |
| 移出的 `LockWorkspace` 不加锁 | 交错 4 的 8 个、5 的 2 个、6 的 2 个 | 探针失败（移出先时第一方也不再持工作区行） |
| 增长的 `ShareDirectoryWorkspaceByID` 不加锁 | 交错 4 的 8 个 | 增长先：`lockOn` 看不到 acme 的 `FOR SHARE`；移出先：探针失败 |
| 建项目的 `ShareDirectoryWorkspace` 不加锁 | 交错 5 的 2 个 | 建先：`lockOn`；移出先：探针失败 |
| 删除工作区的 `LockWorkspaceBySlug` 不加锁 | 交错 6 的 2 个 | 删除先：探针失败；移出先：删除改写 acme 的行同样等在 acme 上，探针得到满足，由结果发现（acme 被删除） |

最后一行是唯一一个能被另一个等待满足的探针：那个等待在同一行上（删除的 `UPDATE`），结果仍然看得到缺锁。

### 2.12 矩阵（Task 4、5、11；9.2）

- 移出三个变体（各 6 格）：另一个成员 `ofMember(204, 403, 403)`、自己的 `ofMember(409 own_membership, 403, 403)`、已结束的 `ofMember(404 member_not_found, 403, 403)`；`toMembership(method, target, body)` 改角色、移出共用；`endedMembership(c)`。
- 离开：`inWorkspace(204, 204, 204)`（6 格）；唯一管理员的一张表一格（409 `workspace.sole_admin`）。
- "被移出的成员"由存储写出（`projectSeed.removal`，Task 11）：`EndMember`，然后 `LockActiveMemberProjects` 在这时找出的项目、`EndMemberships`，由 acme 的管理员在一个时刻。前提：他在 acme 不是有效成员（`targets`）、他在公开项目的成员关系已结束未删除（`preconditions`）、other 的管理员是它唯一的有效管理员、acme 的管理员另有一位、被移出的成员是 other 的有效成员。成员列表不再有他。P-前（以前的成员在私有项目的成员关系）和 `partingStates` 仍是 SQL，留给 P5b（第 5 节）。
- 本 Phase 25 格，矩阵共 415 格，两轮原型分别 1.91、1.39 秒（9.2 的预算之内；P4b 结束时 390 格、约 1.4 秒）。

### 2.13 S2、9d（Task 12；3.8、9.3）

`adminsWorld`：acme 的管理员 alice、bob，成员 carol，Web；alice 移出 bob、以访客再邀请他，`reactivate-member` 恢复他。这是"有效成员带着发给他的待接受邀请"唯一经接口和命令得到的状态（3.8 拒绝邀请有效成员）。`TestAnInvitationNeverChangesAnActiveMembership`（S2）、`TestAnEndedMembershipLeavesNoInvitation`（9d，移出、离开各一次）的内容见 plan Task 12。

### 2.14 端到端（Task 13、14；第 2 节 W2、W7、W12，9.6）

- 夹具：`listMembers`、`membershipOf`；`expectMembershipEnded`（工作区成员关系和列出的项目成员关系由同一个账户在同一个时刻结束、行留着、角色不变；没有别的有效项目成员关系；没有待接受的邀请）；`expectWrittenLastBy`（结束之前各行由谁最后写，清扫 14）；`expectWorkspaceDeleted` 的 `deletedAlone` 成为参数（第 3 节第 12 条）。
- W2、W7、W12 照设计第 2 节的接口一列，为清扫加了几步（第 3 节第 13 条）。W12 从故事里运行命令（P1 的 `nerveWorkspaces`、`nerveWorkspacesFails`）。

### 2.15 文档（Task 14；3.20、4.11、8.7）

- `README.md` 部署一节加"恢复被移出的成员"一条（8.7，G1 (a)）。
- `docs/v0/plane-diff.md` 第四节加三行：移出、离开时的唯一管理员检查；恢复被移出的成员；移出、离开时发给他的待接受邀请。

## 3. 与设计的差异和补充（待控制者裁定）

以下都没有改变 M3 设计的架构。每条给出建议。

1. **任务的划分：14 个，不是 15 个**。设计的任务与 plan 的对应：1 → 1；2 → 2；3、4 → 3（结束一步和移出的用例合在一起：结束一步没有自己的使用者就测不到调用顺序，合起来 724 行）；5 → 4；6 → 5；7 → 6；8 → 7；9 → 8；10 → 9；11 → 10；12 → 11（移出、离开的矩阵行随各自的操作在 Task 4、5，约束 1）；13 → 12；14 → 13；15 → 14（review 是控制者的）。最大的是 Task 5（1,040 行：离开的用例、接口、矩阵、组合出的结束加离开）、Task 7（830 行）、Task 4（783 行）；plan 共 8,206 行。退路 A' 没有用到：14 个任务在约 16 个以内，每个在约 1,500 行以内。**建议接受。**
2. **移交的落点**（说明）：

   | 移交 | 落点 |
   |---|---|
   | P4b spec 第 5 节：`standIns` 的"被移出的成员"换成存储；`listsTheProjectMembers` 不再有他 | Task 11（2.12） |
   | 同上："以前的成员"（P-前）换成存储 | P5b（它的移出项目成员写得出那一行；第 5 节） |
   | 同上：`EndMemberships` 先锁项目 `FOR NO KEY UPDATE`、在工作区 N 之后读时刻 | Task 2（2.4）；时刻由移出、离开在锁之后读并传入（Task 3、5；`TestEachLockOfAnEndingIsItsStrength` 的时刻） |
   | 同上：改项目成员的角色、移出项目成员、离开项目经 `Locks`，各在 `TestEachWriteOnAProjectSharesItsWorkspaceFirst` 加一行 | P5b |
   | 同上：交错 4、5 照方案 E 的写法，第二方等在工作区行上 | Task 9、10（2.11，探针和反例） |
   | 同上：故事清扫接过"已结束"的谓词：`ListMembers` 的 `is_active`、`ProjectFacts` 的 `m.is_active`、`RestoreMember` 的 id、`Memberships` 的账户 | 前三个由 W7、W7、W12 发现（附录 A 清扫 1）；`Memberships` 的账户故事看不到（第 9 条），由 P4b 的存储测试发现；P5b 的故事 P5 一次添加几个账户 |
   | 同上：交错 1、4、5、6 用 P4b 的添加、加入（`growth`） | 交错 4 的增长是 P4b 的添加和加入，经项目模块的接口；交错 5 是 P4a 的建项目 |
   | 同上：`refusingCommits(t, pool, table)` 按表 | 已在 P4b；`TestAnEndingEndsTheMembershipsAndLeavesNoInvitation`、降级测试照表用它 |
   | 同上：一个改变一个时刻对 `EndMemberships` 成立 | `TestAnEndingEndsTheMembershipsAndLeavesNoInvitation`（四行一个时刻）、交错 4、5（项目成员关系的时刻不早于 gate 放开） |
   | P4b review 第 6 节：按资源寻址的项目级的写、P8 的完整性规则的第二种形状 | P5b；P5a 只把完整性核对限定为项目表的列，离开工作区的唯一管理员一行不算项目级的写（第 6 条） |
   | 同上：e2e 的 `deletedAlone`、`expectProjectDeleted` 只数删除之前未删除的行（M5） | `deletedAlone` 成为参数（第 12 条）；P5a 的故事不删除有单独删除的行的项目，`expectProjectDeleted` 不动（W12 删除 Old 之前不读它，第 5 节） |
   | 同上：添加的"成员关系已结束"的变体、建项目的负责人变体各只答一个错误（M4） | Task 11：两者照旧以被移出的成员为目标，矩阵的格子核对整个错误列表 |
   | P4a spec 第 5 节：`EndMemberships` 带 `by`；`ProjectMembershipCounts` 进 `project.Provide`；`nerve workspaces` 是否建 `NewCascade` | Task 2；Task 7；不建（Task 8：恢复不调用连带，2.10） |
   | P4a review 第 6 节：`createProject` 的锁顺序由交错 5 在真实数据库上核对（F-M2）；acme 有两位管理员，唯一管理员的 409 要另一个工作区；`deletedAlone` | Task 10（建先时 `lockOn` 核对它持 acme 和他的成员行的 `FOR SHARE`）；第 6 条；第 12 条 |
   | P1–P3 review 第 6 节："被移出的成员"一列的 SQL 换成存储；交错 1 照交错 2 的写法；移出、离开先锁工作区（P2 T11 C2）；邀请的一步；S2、9d、W12 | Task 11；Task 9（`adminRace`、`endedHolding`）；Task 3、5；Task 3（2.5）；Task 12、14 |
   | 方案 E 的代价（17.4） | 移出、离开、恢复列在第 6 节的风险里；三个写都不延长项目级的写持 `FOR SHARE` 的时间（它们不是项目级的写） |
   | 一直有效的规则：锁之后读时钟；锁下重读确认父行；目录驱动的检查；`apitest.Main` 两个方向；端口的错误原样返回；角色按集合；关键词守卫 | Task 3、5、7（`TestEachWriteReadsTheClockUnderItsLock` 三行）；Task 3（`lockedMember`）；`table_rows_test.go` 和组合出的删除测试不加豁免照旧通过（P5a 没有新表）；Task 4、5；清扫 2；`role = 20`；契约措辞没有新的命中 |

3. **命令行的两个新码**：`workspace.slug_not_found`（"No workspace has this slug."）、`workspace.never_a_member`（"The account has never been a member of this workspace."）。3.11 说恢复在工作区不存在、从来不是成员时退出码 1，没有给码。它们与 P1 的 `workspace.account_not_found` 一样只给命令行，不进任何操作的 `x-problem-codes`，也不进 `PROBLEM_MESSAGES`（约束 4 说的是契约声明的码）。不用 `workspace.not_found`：它在接口上的意思是"不存在或你不是成员"，命令行没有调用者。**建议接受。**
4. **恢复不写 `updated_by_id`**：照 Plane 的 `reactivate_workspace_member`（它只写 `is_active` 和自动的 `updated_at`）。命令没有发起它的账户；写成成员自己会让"由谁结束"的记录丢失：恢复之后 `updated_at` 是恢复的时刻，`updated_by_id` 仍是结束这一成员关系的账户（移出他的管理员、离开的本人；P6 起也可以是被停用的账户本人）。W12 核对移出的情形（仍是 A），`TestReactivateMember` 核对 `updated_at`。**已接受**（第 7 节）。
5. **`ProjectMembershipCounts` 不经 `bootstrap/ports.go` 转换**（设计 12 节 P5a 任务 8 写了"`bootstrap/ports.go` 的转换"）：方法的参数和回答只有 `uuid` 和 `int`，`project.ProjectMembershipCounts` 与 `workspace/app.ProjectMembershipCounts` 方法相同，`bootstrap` 直接接上（6.5"跨边界的值"只要求有模块类型时转换）。同理 `workspace.New` 不收它：P5a 的接口操作没有用它的（6.6 第 5 步列了它），只有命令的组合用（第 2.10 节）。**建议接受。**
6. **唯一管理员的一张表**（P4a review 第 6 节：acme 有两位管理员）：一列 `callerSoleAdmin`，是 other 的管理员（从来不是 acme 成员的那个账户），other 里有被移出的成员作它的有效成员，前提核对这些。完整性核对（`writesOnAProject`）原来把"列不全是工作区一级的"行都算作项目级的写，会把这一行算进去；改为只数项目表（`projectTables`）的列，并加两个反例（离开工作区不是项目级的写；按资源寻址、列组不在 `projectTables` 里的也不是）。`projectTables` 由此是"项目一级"的定义：规则 (b) 的说明照它写（预检 L1），项目一级的新表要列进 `projectTables`，不能放在 `matrixTables` 里唯一管理员的表旁边，否则规则 (b) 看不到它的写；这交给 P5b（第 5 节）。规则 (a)（路径里有 `{project_id}`）照旧发现按路径寻址的写；P4b 的七行各去掉一行都被完整性核对发现（附录 A，`pf-r6-*`）。目标核对（`matrixViolations`）同样拒绝在这张表里放项目的操作。**已接受。**
7. **`ending_test.go` 从 Task 4 起就是最终的表形**（裁定：不要过渡文件）：Task 4 建它，`ending{name, request, by}`、`endings` 只有移出一行，`TestAnEndingEndsTheMembershipsAndLeavesNoInvitation` 已是最终的测试（"之前由 dave 写"的前提、全由 alice 写的邀请），`endingWorld`、`bystanders`、`soleAdmins`、`workspace`、`membership`、`rowJSON`、`rowsBut`、`uuidTexts`、`projectMemberships` 也都是最终的；它只缺四样：`leave`（Task 4 没有使用者）、离开的一行、`TestTheOnlyAdminCannotLeave` 和说明里关于离开的话。Task 5 用 old/new 块加上这四样，得到与原来 Task 5 相同的文件（加上预检 L6 的几行）；Task 6 的块一字不改照样放得上。原来 Task 4 的 `removal_test.go` 独有的三样都被取代，没有丢掉什么：发给 bob 的邀请记作 carol 发的（现在全由 alice 发）、bob 的三行之前由 bob、carol、bob 写的前提（现在 dave 先写四行，Web 也在内，原来的前提没有它）、`remove` 帮手（现在是 `endings` 的一行）。plan 少了约 290 行。
8. **`moduleRoute`**：P4b 的 `projectRoute`（项目模块在只有一个连接的池上）改成任何模块的路由：`newRoute(t, register)`，`newProjectRoute` 经它，`newWorkspaceRoute` 新加；`answerWithin` 把"5 秒内拿到回答"的核对提出来。只是测试代码的移动，P4b 的连接测试内容不变。**建议接受。**
9. **故事看不到的谓词**（清扫 1 的故事一半，附录 A）：
   - 工作区成员关系的 `deleted_at IS NULL`（`EndMember`、`HasOtherAdmin`、`ReactivateMember`）：工作区成员关系不单独删除，只随工作区删除（4.3），故事里没有一个已删除而工作区未删除的成员行。
   - 项目成员关系的 `deleted_at IS NULL`（`LockActiveMemberProjects` 的 `m.deleted_at`、`SoleAdmin` 的三处、`EndMemberships`）和 `p.deleted_at IS NULL`：已删除的项目成员关系只随已删除的项目出现，已删除的项目不在锁的结果里（`p.deleted_at`），而锁住的项目里同一对最多一行未删除。
   - `LockActiveMemberProjects` 的 `EXISTS` 的相关、成员、有效，整个 `EXISTS`：只多锁项目，写由 `EndMemberships` 自己的谓词再筛一遍，规则 2 由 `SoleAdmin` 自己的谓词决定；存储测试发现每一个，相关、成员、整个 `EXISTS` 另由组合的 `TestEachLockOfAnEndingIsItsStrength`（Site 被锁）发现。
   - `SoleAdmin` 的 `m.is_active`、`EndMemberships` 的 `is_active`：在连带之下等价：锁住的项目里他有一行有效的成员关系，同一对最多一行未删除。
   - `CountInactive` 的 `NOT is_active`：结束工作区成员关系的每条路都同时结束他的项目成员关系，恢复之前他没有有效的项目成员关系可数。
   - P4b 留下的 `Memberships` 的账户：每个调用者都按问到的账户取结果（`add_members.go`、`growth.go`、`join_project.go`、`update_project.go`），多出来的行不被读到；P5b 的故事 P5 一次添加几个账户。
   
   每一个都由存储测试在两个行序下发现。安全性质的变异里现有的故事都没有展示的只有一个，结束一步跳过邀请（`s5-end-invitations`）：每个故事结束成员关系时，那个工作区都没有发给他的待接受邀请（W7 那一封已被撤回，W12 的旧邀请在移出之后才发）。能展示它的状态只有 9d 那一种：他有效，带着一份在他被移出期间发的待接受邀请；W12 的 B 用这份邀请展示 S2，9d 由组合的测试展示。预检设想的那一步（carol 拒绝之后，B 再给她一份新的待接受邀请，然后恢复、移出她）不成立：3.8 说已拒绝的邀请仍占着这个邮箱，部分唯一索引 `workspace_member_invites (workspace_id, email) WHERE deleted_at IS NULL` 让再邀请得到 422 `duplicate`，已拒绝的与新的待接受邀请不能并存。它由组合的 `TestAnEndingEndsTheMembershipsAndLeavesNoInvitation`、`TestAnEndedMembershipLeavesNoInvitation` 发现（`pf-end-noinv`）。**已接受。**
10. **G2 的顺序在组合一层看不到**：已结束 404 和自己 409 只对通过判定的调用者可见，而通过 `workspace_member.remove` 判定的调用者是有效的管理员，"自己的"就是有效的：两个检查不相交。单元测试（假的 `Authorizer` 放行）核对 G2 的顺序；矩阵核对每个检查本身。**说明。**
11. **结束的时刻由调用者读**：3.3 要移出、离开在工作区 N 之后读时刻，`EndMemberships` 在 `now` 上写；一个改变一个时刻（P4b 第 7 节）因此对连带成立：工作区一级的写持 N，项目级的写都先取工作区的 S（方案 E），连带写下的时刻不早于任何进行中的项目级的写。**说明。**
12. **`deletedAlone` 成为参数**（P4b review 第 6 节 P26）：移出、离开软删除邀请，故事可以先单独删除一个工作区里的邀请；`expectWorkspaceDeleted(db, slug, adminEmail, deletedAlone)` 由故事说它单独删除过哪些表的行（W3 传 `deletedAloneTables`，W2 删除 First 时传空的）。**建议接受。**
13. **故事为清扫多加的步骤**（设计第 2 节的 W2、W7、W12 照旧都在）：W2 里 Ops 是 bob 的、alice 由他添加为管理员，bob 成为管理员之后把她的角色原样设一次（清扫 14：她离开之前她的行由别人写过）；W7 加了 Other（bob 是别处的管理员，`HasOtherAdmin` 的工作区）、Docs（carol 的项目，dave 是成员；erin 是另一位管理员、离开了 acme：规则 2 的另一个管理员须有效）、发给 eve 的邀请、Other 发给 carol 的邀请和被撤回的邀请（不动的邀请），bob、carol 自己加入 Web；W12 加了 Other、Lab、Spare（他在别处的行和邀请不动）、Old（结束之后删除的项目不计数）、carol 离开 acme（只恢复 B 的行）和 carol 被拒绝的邀请（3.8 已拒绝的不动）。设计第 2 节 W7 的降级"含已离开的项目"：工作区成员关系有效而项目成员关系已结束，只能由 P5b 的离开项目写出，P5a 里没有这样的行；W7 经 carol 以访客身份接受邀请展示已结束的项目成员关系也降为访客。PATCH 降级走同一个 P4a 的 `DemoteMemberships`（有效的和已结束的都改），它对已离开的项目的展示留给 P5b（第 5 节）。**建议接受。**
14. **P5a 改了 P4b 的降级测试的来历**（`TestAcceptingAsAGuestAgainDemotesInTheWorkspacesProjects`）：已结束的成员关系原来由 SQL 替身写，现在经接口由 alice 移出 bob：替身换成真实的写，"由他自己写"的接受之前那两行由 alice 写（清扫 14）。**说明。**

## 4. 验收标准（完成线，M3 设计 12 节 P5a）

- [ ] W2、W7、W12 的接口版本通过，此前的每个故事仍然通过（`make e2e` 共 65 个：此前的 62 个，加三个）。
- [ ] 交错 1 的工作区一侧、4、5、6，两种顺序，`-count=5 -race` 通过，没有 40P01（原型：70 个子测试，12–14 秒）。
- [ ] S2 顺序和 9d 顺序（移出、离开各一次）的集成测试通过。
- [ ] 离开的规则 1（`TestTheOnlyAdminCannotLeave`：solo、acme 的 409，第二位管理员之后 204）和连带结束的规则 2（`TestAnEndingEndsTheMembershipsAndLeavesNoInvitation`：唯一管理员的 409，另有管理员之后 204）在组合出的 app 上各有正反例。
- [ ] `reactivate-member` 在真实数据库上通过；退出码 1 的每种情形（没有账户、没有工作区、已删除的工作区、从来不是成员）和提交时被拒，每张表不变。
- [ ] `workspace` 的 `apitest.Main` 两个方向核对通过（含 `project.sole_admin`）。
- [ ] 本 Phase 的矩阵格子（25 个）通过；共 415 格，耗时记下；完整性核对通过。
- [ ] `make lint`、`make test`、`make gen-check`、`make knip`、`make test-web`、`make e2e` 通过。
- [ ] 3.20、8.7 中 P5a 的行写好。

## 5. 不在 P5a 范围内

- 改项目成员的角色、移出项目成员、离开项目、交错 1 的项目一侧、故事 P5：P5b。停用、`project.NewCascade`、交错 7、13、14、15、16、19：P6。标签、默认之外的状态：P7。`PROBLEM_MESSAGES` 的搬迁：P8。页面：P9–P11。M4 的工作项写入是否取工作区的 S：M4（负责人 2026-10-02）。

**留给后面的 Phase**（各 Phase 的 spec 接过去；P5a 的 review 第 6 节照录）：

| Phase | 条目 |
|---|---|
| P5b | 矩阵的 P-前替身（`standIns` 剩下的一条：以前的成员在私有项目的成员关系结束）和 `partingStates` 换成 P5b 的移出项目成员、离开项目写出；唯一管理员的表若要项目一侧的格子（`leaveProject` 的唯一项目管理员），另建项目的一张表，或在这张表里给出项目一级的列并让完整性核对（只数 `projectTables` 的列，第 3 节第 6 条）照旧成立；改项目成员的角色、移出项目成员、离开项目是第一批按资源寻址的项目级的写：`TestEachWriteOnAProjectSharesItsWorkspaceFirst` 要能接资源路径、第一步探测资源行（P4b review 第 6 节）；交错 1 的项目一侧照 `TestTwoAdminsLeavingLeaveAnAdmin` 的写法（`endedHolding` 停在结束之后）；`Memberships` 的账户由故事 P5 一次添加几个账户发现（第 3 节第 9 条）；P5b 的写持项目的 N 期间 P5a 的连带等在项目行上（`LockActiveMemberProjects`），反过来项目级的写先取工作区的 S、等工作区一级的 N：两者不成环，P5b 的交错要两种顺序都证明；每一张项目一级的新矩阵表都要列进 `projectTables`，否则 `TestEachWriteOnAProjectSharesItsWorkspaceFirst` 的规则 (b) 看不到它的写（规则 (a) 只认路径里的 `{project_id}`；第 3 节第 6 条）；设计第 2 节 W7 的"降级含已离开的项目"：成员先离开一个项目（`leaveProject`），再被 PATCH 降为访客，那一行也成为访客的（第 3 节第 13 条），由 P5b 的故事展示 |
| P6 | `EndMemberships` 照约定六跨几个工作区调用一次（`workspaceIDs` 已是切片）：停用按 id 顺序锁住他所在的每个工作区的 N 之后调用，时刻在这些锁之后读；`TestLockActiveMemberProjectsLocksInIDOrder` 的顺序对跨工作区的结果同样成立（查询按项目 id 排序，不按工作区）；规则 2 跨工作区（他在任何一个工作区是"还有别的有效成员"的项目唯一的管理员都拒绝）还要核对工作区一级的唯一管理员（3.7 规则 2 的工作区部分：P5a 的移出、离开不需要它，规则 1 只管离开）；`project.NewCascade` 随 `nerve users deactivate` 加入；停用之后 `reactivate-member` 照样恢复（交错 16）；`nerve users deactivate` 同样没有请求期限（17.4 G1 (a)），README 照 8.7 写明 |
| P7 | 状态、标签的写是项目级的写，经 P4b 的 `Locks`；连带不改它们（结束成员关系不碰状态和标签） |
| 后来 | 结束的成员关系的显示设置（`workspace_user_properties`、`project_user_properties`）留着，恢复之后照旧（Plane 相同，第 7 节） |

## 6. 风险

| 风险 | 应对 |
|---|---|
| 同一个工作区里连续重叠的项目级的写让移出、离开等到请求期限、让恢复等到操作者中断（方案 E 的代价，17.4） | 移出、离开、恢复都是短事务；到期或中断时回滚、可以重试（`TestAnInterruptedReactivationChangesNothing`）；README 照 8.7 写明命令没有期限 |
| 规则 2 的项目集合查错（Plane 查错过三种） | 只在锁下的项目集合上查；`TestSoleAdmin` 的每个情形是自己的一个项目、别的情形的项目各有管理员和成员；W7、W12 单独运行看得到每个决定结果的谓词（附录 A 清扫 1） |
| 连带和项目一侧的增长、建项目、删除工作区交错 | 交错 4、5、6 两种顺序，探针各有反例；连带在调用时列举项目（交错 4、5 的增长先） |
| 故事看不到的谓词 | 存储测试在两个行序下都看得到；理由逐条在第 3 节第 9 条 |
| plan 的最大 Task 接近上限 | 最大的 Task 5 是 1,040 行，每个都在约 1,500 行以内 |
| 持续集成（ubuntu）上的运行 | 原型和复现都在 macOS 上；合并之后看持续集成，P1–P4b 都是这样合并的 |

## 7. 已知的限制、交接和关闭条件

- **结束的成员关系的显示设置留着**：移出、离开不删除他在工作区、项目的显示设置；恢复、重新加入之后照旧（与 Plane 相同）。
- **恢复只恢复工作区的成员关系**：项目成员关系仍结束，他经接口加入项目时恢复（P4b 的 `RestoreMember`，角色取原来那一行与工作区角色中较低的，3.5）；命令输出还有几个。恢复出来的工作区成员关系计入管理员的人数，账户停用时他还不能登录（3.11）。
- **恢复不写 `updated_by_id`**：恢复之后 `updated_at` 是恢复的时刻，`updated_by_id` 仍是结束这一成员关系的账户（移出他的管理员、离开的本人；P6 起也可以是被停用的账户本人）：恢复不写 `updated_by_id`（第 3 节第 4 条）。
- **一个旧邀请在恢复之后可能还在**：移出之后才发的邀请不随恢复删除；他有效时接受它只消费邀请（S2），他再被移出时它被删除（3.11，`TestAnEndedMembershipLeavesNoInvitation`）。
- **调用者的账户在等锁期间被停用**：停用的连带在 P6；P5a 的竞争测试覆盖等锁期间成员关系结束、工作区删除。
- **两个管理员同时移出对方**：两者都持同一个工作区的 N，后到的在锁下重读之后判定，它的调用者已被移出：404（与 `TestAnEndingFindsWhatEndedMeanwhile` 的"alice 的成员关系结束"同一条路）。
- **方案 E 的代价**（17.4）：见第 6 节。

| 交接 | P5a 处理的条目 | 留下的条目 |
|---|---|---|
| P4b spec 第 5 节、review 第 6 节的 P5 一行 | 第 3 节第 2 条 | P5b 的条目（第 5 节） |
| P4a、P3、P2、P1 的 P5 条目 | 第 3 节第 2 条 | 无 |

**M3 设计 13.1 的关闭条件**：13.1 没有落在 P5（P5a、P5b）的一项："M2-closeout §13 P5 改到"是 M2 的 P5 页面，在 P9；"M1-P4 离开项目的顺序"是页面的行为，在 P10。`docs/v0/M3-workspace-project/handoffs/` 里没有交给 P5a 的条目。P5a 的条件都有落点，没有放不下的。

## 附录 A：原型验证记录（2026-10-02）

原型在 `$M3TMP/p5aproto`（`b5e826b3` 的副本，Go 1.27.1、Node 24、pnpm 11.10.0、Docker；`bin/golangci-lint` 2.13.2）。做法照 P4b：每个 Task 做完时存一份源文件的快照（`$M3TMP/p5asnap/T1`…`T14`），plan 的代码块由脚本从相邻两份快照的差异生成（`p5tools/mkblocks.py`），生成的文件不进块，按 SHA-256 核对（`gensha.py`）。清扫之后加强的测试由 `propagate.py`、`carry_edit.py` 从带来它的 Task 起改进每一份快照和原型，之后重新生成全部块；`assemble.py` 组装 plan 并核对每个块放了一次、每个文件在文件表里、每个 Task 在约 1,500 行以内。

**修订一轮**（控制者的裁定和预检之后，2026-10-02）：原型、快照的旧版本留在 `$M3TMP/p5aproto-before-amend`、`p5asnap-before-amend`。裁定 7 和预检的 L1、L3、L4、L6 由 `p5tools/amend_*.py` 以逐字的替换改进原型和快照（每个替换在每一份里恰好出现一次）；Task 4 的 `ending_test.go` 由 Task 5 的去掉离开的四样得出；契约的生成物在原型（Task 5 起）和 Task 4 的快照副本里各 `make gen` 一次（`amend_gen.py`）；之后重新生成全部块和生成物的表。Task 6 的块一字未变。下面的数字都是修订之后在原型上重跑的。

**逐 Task 复现**（`$M3TMP/p5tools/replay.py`、`replay_all.sh`，日志在 `replay-logs`）：从 `b5e826b3` 的一份新副本开始，照 plan 的顺序应用 14 个 Task 的块并运行每个 Task 写明的命令。每个 Task 之后 `make lint-go` 两段 `0 issues.`、`make test` 42 个 `ok`；Task 1、2、4、5、7 的 `make gen`/`make gen-go` 之后生成物与快照没有差异，SHA-256 和行数与 plan 的表相同；Task 4、5、13、14 的 `make lint-web`（关键词守卫 5 个命中都有例外）、`make knip`、`make test-web` 通过；Task 13 的 `make e2e` 64 个中 63 个通过、Task 14 的 65 个中 64 个通过，失败的都只是 S3 的 F4（副本不是 git 仓库，构建没有提交号）。隔离规则不允许在副本里 `git init`，`replay.py` 只接受这一个失败：失败的故事只有 S3、它的断言只是提交号，其余任何失败都算复现失败（`check_f4.py`）。最后的树与原型逐个文件相同（`treediff.mjs`：3,020 个文件，0 处差异）；`planapply.mjs` 从基线核对 plan 的 225 个块（202 处替换、23 个新文件）都放得上；最后一次 `make gen` 之后与原型逐个文件相同（`treediff.mjs` 0 处差异；副本不是 git 仓库，`make gen-check` 的 `git status` 在这里不起作用）。共 751 秒（修订之后的复现；第一轮的 756 秒、222 个块的记录在 `replay-logs-v1`）。

**最终的原型**（`gates.sh`）：`make gen` 之后生成物没有差异；`make lint-go` 两段 `0 issues.`；`make test` 42 个 `ok`；`make lint-web`（关键词守卫 5 个命中都有例外，54 个任务）；`make knip`；`make test-web`（16 个任务）；`make e2e` 65 个故事中 64 个通过，S3 因原型不是 git 仓库、构建没有提交号而失败（P4b 的 F4；它读 `commit`，与 P5a 无关）。

**矩阵**：`TestPermissionMatrix -v` 415 格（P4b 结束时 390 格，P5a 加 25 格：移出三个变体各 6 格、离开 6 格、唯一管理员 1 格），1.39 秒（第一轮 1.91 秒，基线 1.42 秒），全部通过。

**交错**：交错 1、4、5、6 `-count=5 -race -v`：70 个子测试全部通过，13.5 秒（第一轮 12.2 秒）；输出里没有 40P01，没有数据竞争。S2、9d 的集成测试在 `make test` 里通过。

**`reactivate-member` 在真实数据库上**：`TestWorkspacesReactivateMember`、`TestWorkspacesReactivateMemberErrors`（没有账户、没有工作区、已删除的工作区、从来不是成员的账户、提交时被拒：每张表不变）、`TestAReactivationFindsWhatChangedMeanwhile`、`TestEachLockOfAReactivationIsItsStrength`、`TestAnInterruptedReactivationChangesNothing`、`TestTheReactivationRunsOnItsTransactionsConnection`、`TestWorkspacesReactivateMemberCommand` 通过；W12 从故事里运行命令。

**十八类清扫**（brief 的缺陷类别；每个变异一个 `go test -overlay` 或 `go build -overlay`，树不动；"层"是它被发现的最低一层之外的每一层）：

| 清扫 | 大小 | 结果 | 层 |
|---|---|---|---|
| 1 每个 SQL 谓词 | 存储一半：P5a 的 8 个新查询的 46 个谓词（`EndMember` 3、`HasOtherAdmin` 5、`ReactivateMember` 3、`DeletePendingInvitations` 4、`LockActiveMemberProjects` 7、`SoleAdmin` 16、`EndMemberships` 4、`CountInactive` 4），每个去掉或改成 true，参数照旧绑定；加项目成员列表的 `is_active`（`x-pl-active`）。故事一半：这 46 个加 P4b 交来的 4 个（`pl-active`、`pf-active`、`rm-id`、`ms-member`），每个只运行它的故事 | 存储一半 47/47，每个存储测试都有只由那个谓词决定的行，已删除的行在有效的之前、之后两种行序下各一次。故事一半 33/50：W12 发现 22 个、W7 17 个、W2 4 个，只由一个故事发现的 W12 16 个、W7 9 个；17 个故事看不到，理由逐条在第 3 节第 9 条，每个都由存储测试发现。规则 2 的三个谓词（另一个管理员的有效 `sa-a-active`、另一个成员的成员 `sa-o-member`、整个 `EXISTS` `sa-o-exists`）另在组合一层被发现（清扫 5 的 `pf-sa-*`，预检 L6） | 存储；组合（`x-pl-active`：矩阵；规则 2 的三个）；端到端 |
| 2 端口调用的错误 | 移出、离开、恢复、结束一步、`EndMemberships` 的每个端口调用：失败被忽略、被换成别的问题、被吞掉、重试一次、拿到别的账户的地址；18 个 | 18/18；每个单元测试比较到失败那一步为止的整个调用记录 | 单元；组合（`s2-rm-end`、`s2-lv-end`：连带的失败回滚整个结束） |
| 3 写不动的行 | 4 个写（`EndMember`、`DeletePendingInvitations`、`ReactivateMember`、`EndMemberships`），每个的存储测试有另一个工作区、另一个成员、另一个项目，状态各不相同，核对每行每列；7 个变异（多写角色、不写结束者、恢复写成成员自己、也删已接受的邀请） | 7/7 | 存储；"不写结束者"另由故事（清扫 14） |
| 4 组合根的接线 | `workspace.New` 的移出、离开（连带、事务、时钟），`project.New` 的连带，`project.Provide` 的计数，`workspace.NewAdmin`（计数、事务、时钟），`nerve workspaces` 的组合（计数、账户）；13 个 | 13/13 | 组合（`TestAnEndingEndsTheMembershipsAndLeavesNoInvitation`、`TestWorkspacesReactivateMember`、`TestEachLockOfAReactivationIsItsStrength`） |
| 5 安全性质在真实环境上 | 规则 1（唯一的管理员照样离开、不问另一个管理员）、规则 2（`EndMemberships` 照样结束）、结束一步跳过项目或邀请、移出不判定或判定成离开、移出自己的、移出已结束的、离开判定成移出、两条规则放宽或收窄、恢复已有效的、恢复拒绝停用的账户；S2（`x-accept-active`）；修订加了预检的四个：规则 2 的三半（`pf-sa-admin-inactive`：已结束的管理员算另一位；`pf-sa-himself`："另一个成员"可以是他自己；`pf-sa-nootherreq`：只有他一人的项目也拒绝）和跳过邀请一步（`pf-end-noinv`）；19 个 | 19/19，每个都在组合出的 app 或真实数据库上被发现；规则 2 的三半在 `TestAnEndingEndsTheMembershipsAndLeavesNoInvitation` 里（已结束的管理员在 409 一步，另外两个在 204 一步的 Solo），第一轮它们只在存储一层。故事一半 7 个：5 个由故事发现，`s5-cas-sole` 只由 W7 发现（W12 里他是唯一管理员的项目没有别的成员），`s5-end-invitations` 现有的故事都没有展示（第 3 节第 9 条） | 组合；单元（其中 7 个另有）；端到端 |
| 6 每句文档 | P5a 加或改的注释 261 块（66 个文件，约 690 句）；契约两个操作的描述（12 句，加两个 204 的描述）；README 一行；plane-diff 三行 | 3 句原来没有被持住，已改：`workspaces_test.go` 的说明（"拒绝不改数据库"由 bootstrap 的 `TestWorkspacesReactivateMemberErrors` 持住，命令的测试不持住）；降级测试的说明（"由他自己写"之前那两行由 alice 写，清扫 14）；`expectMembershipEnded` 的使用者（W2、W7、W12）。预检又发现两处说得比代码多，已改：规则 (b) 的说明（L1，照 `projectTables` 的定义写）、两个操作的描述里规则 2 少了"有效"（L3）。其余每句都有代码或测试持住 | — |
| 7 反例里没有随机 | P5a 的全部测试 | 决定变异是否被发现的输入都固定：行序由插入顺序决定，两种顺序各跑一次；按 id 排序的测试的 id 由 `uuid.NewV7` 按写明的顺序取；单元假对象的回答是固定的排列；P5a 的测试不用 `math/rand` | — |
| 8 决定所依赖的读在调用者的事务里 | 移出、离开的 10 个读和锁，恢复的 5 个，各改成走池；15 个 | 15/15 | 组合（`TestTheEndingsRunOnTheirTransactionsConnection`、`TestTheReactivationRunsOnItsTransactionsConnection`：池只有一个连接，走池的读等到期限） |
| 9 锁顺序在真实环境上 | 结束一步（邀请 → 成员 → 项目）、恢复（账户 → 工作区）、`LockActiveMemberProjects` 的 id 顺序和强度；7 个 | 7/7 | 组合（顺序：前一个锁被别的事务持有时，后一个还没有取，NOWAIT）；存储（`lamp-*`；强度另由清扫 13 在组合一层） |
| 10 承重的种子行有前提 | 矩阵种子 5 行（被移出的成员的结束、他的项目成员关系、other 的两种管理员、acme 的两位管理员）；组合夹具 8 行（`endingWorld` 的旁观者 4 行；修订加的 Solo 由 bob 建、Solo 没有别的成员、erin 是 Ops 的管理员、erin 的已结束，由 `soleAdmins` 核对）；每行一个去掉它的变异；故事里的前提 | 13/13（`prepare`、`bystanders`、`soleAdmins` 失败）；清扫中补了 `bystanders` 和 W7 的待接受邀请的前提，补上之后 `dpi-email` 由 W7 发现 | 组合；端到端 |
| 11 每条拒绝路径 | 移出、离开：没有调用者时什么都不读（单元的调用记录为空）；判定失败原样返回；两者没有请求体，没有 422；2 个变异 | 2/2 | 单元、组合（已结束的 404 在判定之前）；单元（G2 的顺序反过来：组合一层两个检查不相交，第 3 节第 10 条） |
| 12 与结束、删除的竞争（组合） | 移出：目标的成员关系结束、调用者的结束、工作区删除；离开：调用者的结束、工作区删除；恢复：工作区删除、成员关系又有效、账户停用；8 个情形，探针证明在等锁；5 个变异（不在锁下重读、三处时钟在锁之前读、在锁之前判定） | 5/5 | 组合；单元 |
| 13 锁强度在组合一层看得到 | 移出、离开的工作区 N 和项目 N，恢复的账户 S 和工作区 N；在 gate 停住时用 `lockOn` 的 NOWAIT 阶梯；10 个 | 10/10 | 组合（`TestEachLockOfAnEndingIsItsStrength`、`TestEachLockOfAReactivationIsItsStrength`） |
| 14 每个"由谁"可以失败 | 存储测试里被写的行之前由另一个账户写；组合的结束测试和降级测试；W2、W7、W12 结束、降级之前各行的写者（`expectWrittenLastBy`）；7 个变异（存储一层是清扫 3 的三个 `*-by`，组合一层 `x-demote-by`，故事三个 `a-*-by`） | 7/7；清扫中补了 W2、W7、W12 的前提，W2 改为 bob 建 Ops、添加 alice，降级测试的已结束的成员关系改由 alice 移出 bob 写出 | 存储；组合；端到端 |
| 15 标题的每个说法都有展示 | W2、W7、W12 的标题共 14 个分句，P5a 的测试标题和说明 | 每个分句由故事自己的一步展示（W7 的"每个项目"是两个，其中 Ops 他是唯一管理员；"没有别的邀请"有待接受的前提；W12 的"什么都不改"核对它可能写的三张表，预检 L4） | 端到端 |
| 16 单元假对象的回答顺序 | 1 处回答列表的假对象（连带的 `LockActiveMemberProjects`：second、first、third）；`MemberProfiles` 只回答一个 | `r16-resort` 1/1 | 单元 |
| 17 判定之前的代价有界 | 3 个操作的输入：路径里的一个 id、一个 slug、命令的两个参数；判定之前只读一行成员关系、锁一行工作区（恢复：一行账户） | 没有无界的输入，不需要上限 | — |
| 18 契约描述只说代码做的 | 两个操作的描述 12 句、两个 204 的描述 | 规则 2 的一句在两个描述里都补上"有效"（预检 L3）。每句都有一个按所说的顺序和结果的测试：矩阵（谁可以、谁 404、谁 403）、`TestTheOnlyAdminCannotLeave`（规则 1 和"先让另一位成为管理员"）、`TestAnEndingEndsTheMembershipsAndLeavesNoInvitation`（邀请、一个时刻、行留着、规则 2 的"有效"两处和之后什么都不变）、`TestAnEndingFindsWhatEndedMeanwhile`（已删除的工作区）、`TestRemoveWorkspaceMemberRefusals`（G2 的顺序） | 组合；单元（G2） |

清扫 2 在单元一层被发现的不是安全性质或锁的性质：它们是错误的传递，每个端口的失败在组合一层没有可以注入的地方（池的失败由清扫 8 的连接测试覆盖）。清扫 11 的 G2 顺序只在单元一层，理由见第 3 节第 10 条。没有只在单元一层被发现的安全性质或锁的性质。

**按缺陷类别的变异表**：

变异的定义在 `$M3TMP/p5tools/mutants_{s1,p5a,probes,seed,fixture,extra,review,amend}.py`，故事的在 `e2e_sweep1.py`、`e2e_code.py`、`e2e_actor.py`；修订之后的结果在 `p5tools/logs`（`amend-mut-*.out`、`mutants_*-all.json`、`amend-e2e-*.out`；第一轮的是 `final-*`）。每个变异都在修订之后的原型上重新运行过（`amend_sweeps.sh`，一次一套）：Go 的 171 个全部被发现，故事的 60 个里 42 个；除了下面写明的新增，被发现的层与第一轮相同（`levelchange.py`）。

| 缺陷类别 | 变异 | 结果 | 失败的测试 | 层 |
|---|---|---|---|---|
| 清扫 1：谓词（存储） | `em-*` 3、`hoa-*` 5、`rea-*` 3、`dpi-*` 4、`lamp-*` 7、`sa-*` 16、`ems-*` 4、`ci-*` 4 | 46/46 | `TestEndMember`、`TestHasOtherAdmin`、`TestReactivateMember`、`TestDeletePendingInvitations`、`TestEndingAMembersProjectMemberships`、`TestSoleAdmin`、`TestCountInactive` | 存储 |
| 清扫 1：谓词（组合） | `x-pl-active` | 1/1 | `TestPermissionMatrix`（`listProjectMembers` 四格） | 组合 |
| 清扫 1：谓词（故事，单独运行） | 上面 46 个和 `pl-active`、`pf-active`、`rm-id`、`ms-member` | 33/50 | W12 22、W7 17、W2 4；看不到的 17 个见第 3 节第 9 条 | 端到端 |
| 清扫 2：端口的错误 | `s2-rm-*` 2、`s2-lv-*` 2、`s2-end-*` 5、`s2-re-*` 5、`s2-cas-*` 4 | 18/18 | `Test{Remove,Leave}Workspace…Refusals`、`…FailsWithinTheTransaction`、`…LocksThenDecidesThenEnds`、`TestReactivateMemberRefusals`、`TestEndMemberships`、`TestAnEndingEndsTheMembershipsAndLeavesNoInvitation` | 单元；组合 |
| 清扫 3：写不动的行 | `s3-em-role`、`s3-em-by`、`s3-dpi-accepted`、`s3-rea-by`、`s3-rea-role`、`s3-ems-role`、`s3-ems-by` | 7/7 | 各写的存储测试 | 存储 |
| 清扫 4：接线 | `s4-{leave,remove}-{projects,tx,clock}`、`s4-cascade-enders`、`s4-provide-counts`、`s4-admin-{counts,tx,clock}`、`s4-command-{counts,accounts}` | 13/13 | `TestAnEndingEndsTheMembershipsAndLeavesNoInvitation`、`TestWorkspacesReactivateMember`、`TestEachLockOfAReactivationIsItsStrength` | 组合 |
| 清扫 5：安全性质 | `s5-leave-rule1`、`s5-leave-ask-none`、`s5-cas-sole`、`s5-end-{projects,invitations}`、`s5-rm-{decide,action,own,ended}`、`s5-lv-action`、`s5-rule-{remove,leave}`、`s5-re-{active,deactivated}`；`x-accept-active`；`pf-sa-{admin-inactive,himself,nootherreq}`、`pf-end-noinv`（修订） | 19/19 | `TestPermissionMatrix`、`TestTheOnlyAdminCannotLeave`、`TestAnEndingEndsTheMembershipsAndLeavesNoInvitation`、`TestAnEndedMembershipLeavesNoInvitation`、`TestAnEndingFindsWhatEndedMeanwhile`、`TestEveryRuleDecidesItsCells`、`TestWorkspacesReactivateMember`、`TestAnInvitationNeverChangesAnActiveMembership` | 组合（每个）；单元 |
| 清扫 5：安全性质（故事） | `s5-leave-rule1`、`s5-rm-own`、`s5-cas-sole`、`s5-end-{projects,invitations}`、`s5-re-{deactivated,active}` | 6/7 | W2、W7、W12（规则 1、项目一步）；W7（自己的、规则 2）；W12（恢复两个）；`s5-end-invitations` 现有的故事都没有展示 | 端到端 |
| 清扫 8：事务的连接 | `s8-*` 15 | 15/15 | `TestTheEndingsRunOnTheirTransactionsConnection`、`TestTheReactivationRunsOnItsTransactionsConnection` | 组合 |
| 清扫 9：锁顺序 | `s9-end-member-first`、`s9-end-projects-first`、`s9-re-workspace-first`；`lamp-order`、`lamp-{share,update,nolock}` | 7/7 | `TestEachLockOfAnEndingIsItsStrength`、`TestEachLockOfAReactivationIsItsStrength`；`TestLockActiveMemberProjectsLocksInIDOrder`、`TestEndingAMembersProjectMemberships` | 组合；存储 |
| 清扫 10：种子和夹具的行 | `s10-*` 5；`f-*` 8（修订加了 `f-solo-alice`、`f-solo-carol`、`f-erin-member`、`f-erin-active`） | 13/13 | `TestPermissionMatrix/prepare`；`TestAnEndingEndsTheMembershipsAndLeavesNoInvitation`（`bystanders`、`soleAdmins`） | 组合 |
| 清扫 11：拒绝路径 | `s11-rm-ended-first`、`s11-rm-own-first` | 2/2 | `TestRemoveWorkspaceMemberRefusals`；`TestPermissionMatrix`（前者） | 单元；组合 |
| 清扫 12：竞争 | `s12-rm-noreread`、`s12-{rm,lv,re}-clock`；`r-rm-decide-first`（P1–P4b 的"在父行的锁之前判定"） | 5/5 | `TestAnEndingFindsWhatEndedMeanwhile`、`TestEachWriteReadsTheClockUnderItsLock`、`TestEachLockOfAnEndingIsItsStrength`、`TestEachLockOfAReactivationIsItsStrength`、`TestRemoveWorkspaceMember…` | 组合；单元 |
| 清扫 13：锁强度 | `s13-lockws-share`、`s13-lockslug-{share,nolock}`、`s13-account-{keyshare,nokeyupdate}`、`s13-lamp-{share,update,nolock,correl,member}` | 10/10 | `TestEachLockOfAnEndingIsItsStrength`、`TestEachLockOfAReactivationIsItsStrength`、`TestAReactivationFindsWhatChangedMeanwhile` | 组合 |
| 清扫 14：由谁 | `x-demote-by`；`a-em-by`、`a-ems-by`、`a-demote-by`（存储一层的三个是清扫 3 的 `*-by`） | 4/4 | `TestAcceptingAsAGuestAgainDemotesInTheWorkspacesProjects`；W2、W7、W12（结束者），W7（降级者） | 组合；端到端 |
| 清扫 16：回答顺序 | `r16-resort` | 1/1 | `TestEndMemberships` | 单元 |
| 裁定 6：项目级的写的完整性核对 | `pf-r6-*`：P4b 的 `projectWrites` 七行各去掉一行；`pf-r6-p4b-heuristic`：规则 (b) 改回"任何不是工作区一级的列" | 8/8 | `TestEachWriteOnAProjectSharesItsWorkspaceFirst`（完整性核对）；`TestWritesOnAProjectAreEachShape`（离开工作区、`projectTables` 之外的列组） | 组合 |
| P5a 自己的：交错的探针 | `p-removal-nolock`、`p-growth-noshare`、`p-creation-noshare`、`p-slug-share`、`p-slug-nolock`（每个探针的反例：去掉被探的锁，或换成别的等待） | 5/5 | `TestARemovalAndTheProjectSidesGrowthSerialize`、`TestARemovalAndTheRemovedMembersProjectSerialize`、`TestARemovalAndTheRemovedAdminsDeletionSerialize`、`TestTwoAdminsLeavingLeaveAnAdmin` | 组合（`-race`） |

P5a 自己的条目在上表中的位置：规则 1 是 `s5-leave-rule1`、`s5-leave-ask-none`、矩阵的唯一管理员一格和交错 1（`p-slug-*`）；规则 2 的项目集合是 `sa-*` 16 个（Plane 查错的三种：项目 id 是 `sa-project`，"只有一个成员"是 `sa-o-exists`，"只有他"是 `sa-a-member`、`sa-o-member`）、`s5-cas-sole` 和在组合一层的 `pf-sa-*`；`EndMemberships` 在调用时列举是交错 4 的增长先（`p-growth-noshare`），按 id 锁 N 是 `lamp-order`、`lamp-*`、`s13-lamp-*`，结束不删除是 `ems-*`、`s3-ems-*`，一个时刻、由调用者是 `s3-ems-by`、`s4-*-clock`、`s12-*-clock`，邀请一步是 `dpi-*`、`s9-end-member-first`、`s5-end-invitations`，唯一管理员的拒绝连邀请一起回滚是 `s4-*-tx`；恢复的账户 S 是 `s13-account-*`，停用的账户、已有效的是 `s5-re-*`，计数是 `s4-*-counts`、`ci-*`；S2 是 `x-accept-active`；矩阵的"被移出的成员"是 `s10-*`。相对角色的规则、`leaveProject`、项目一侧的交错 1 是 P5b 的。

P1–P4b 交来的类别：在父行的锁之前判定是 `r-rm-decide-first`（离开经 P2 的 `lockAndDecide`，顺序由 P2 证明）；锁的强度、锁在事务之外是清扫 13、`s4-*-tx`、`s8-lockws`；规则表的行放宽是 `s5-rule-*`；"其余的行不变"不在空集上是清扫 3 的第二个工作区、成员、项目和清扫 10 的 `f-*`；`errors.Is` 对准确切的问题、不能失败的断言是清扫 2、14；按资源寻址的项目级的写被完整性核对看到是裁定 6 的 `pf-r6-*`；一行一个账户看不到少了的 `WHERE` 是清扫 1；矩阵的格子对准它那一列的目标由 P4b 的 `matrixViolations` 核对；P5a 没有新的表和索引；挂住的测试都有期限（`answerWithin` 5 秒，`WaitForLockWaitOn` 5 秒）；`project.sole_admin` 声明在工作区的两个操作上，由 `workspace` 的 `apitest.Main` 两个方向核对。
