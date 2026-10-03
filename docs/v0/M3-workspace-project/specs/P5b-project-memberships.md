# M3/P5b 项目成员的角色、移出与离开：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M3/P5b `project-memberships` |
| 日期 | 2026-10-03 |
| 状态 | 第 3 节已由控制者裁定（2026-10-03）；裁定和预检的发现（M-1、L-2–L-6）已改入（2026-10-03） |
| 上级文档 | [M3 设计文档](../M3-design.md) 第 2（P5、W7）、3.3、3.4、3.5、3.6（约定二、三、六，加锁表，方案 E 的代价）、3.7、4.3、5.1、5.3、6.7、8.2、9.2、9.3（交错 1）、9.4、9.6、12（P5b 与约束 1–4）、13.1、17.4 节 |
| 前置交接 | [P5a spec](P5a-memberships.md) 第 5 节 P5b 一行，[P5a review](../reviews/P5a-memberships-review.md) 第 6 节，[P4b review](../reviews/P4b-project-members-review.md) 第 6 节中由 P5a 转给 P5b 的条目（P5a spec 第 3 节第 2 条；落点见第 3 节第 2 条） |
| 计划 | [P5b plan](../plans/P5b-project-memberships.md) |

本 spec 只写 M3 设计交给 P5b 决定的东西：名字、签名、SQL、测试名，以及原型证明了什么。规则本身以 M3 设计为准，这里引用节号，不重述。P5b 依赖 P5a（`ebfd2237` 的 `main`）。

P5b 是评审敏感的一段（M3 设计 12 节约束 3）：它加了第一批按资源寻址的项目级的写，经共用取锁路径 `Locks` 的一个新分支（不加锁地读成员关系 → 工作区 S →（改角色）成员的工作区成员行 S → 项目 N → 锁下重读、确认仍属这个项目 → 判定）；它写 3.5 的相对规则（按集合，工作区管理员的例外、工作区访客的上限）和 3.7 规则 1 的项目一侧。谁能改项目角色、移出项目成员、离开项目是安全性质，唯一管理员的规则也是。附录 A 的三十类清扫逐个核对每个这样的测试在它所说的性质去掉之后失败，并记下失败在哪一层（单元、存储、组合、端到端）；只在单元一层失败的安全或加锁的性质算缺口，例外逐条写在第 3 节。

## 1. 目标

按 M3 设计 12 节 P5b：改项目成员的角色、移出项目成员、离开项目，第一批按资源寻址的项目级的写。具体是：

- 三个操作：`updateProjectMember`（`PATCH /api/v0/project-members/{project_member_id}`）、`removeProjectMember`（`DELETE` 同一路径）、`leaveProject`（`POST /api/v0/projects/{project_id}/leave`），各带操作名、规则行、矩阵行（共 108 格，矩阵共 523 格），各在 `TestEachWriteOnAProjectSharesItsWorkspaceFirst` 有一行；三个新码 `project.member_not_found`、`project.own_membership`、`project.role_too_high`；
- `Locks.lockMemberAndDecide`：按成员关系 id 寻址的写的取锁路径（3.6 约定二、加锁表），与按项目 id 的路径共用锁的部分；等锁期间成员关系结束、删除、不再属于这个项目，调用者自己的成员关系结束，项目或工作区删除，都答 404、从不是 403，一行不改；
- 3.5 的相对规则（`CheckRoleChange`、`CheckRemoval`，按 `roleOrder` 的集合）；
- 3.7 规则 1 的项目一侧：项目唯一的有效管理员不能离开，哪怕只有他一人；`project.sole_admin` 的说明为规则 1、2 两条措辞，补救对每个收到它的调用者都成立；
- 交错 1 的项目一侧（两位项目管理员同时离开，问过另一位管理员之后另有一个 gate），离开与 P5a 的移出（连带结束他的项目成员关系）两种顺序都串行、不成环；
- 矩阵的 P-前和 `partingStates` 的已结束一行改由存储的写写出，替身换成 P5b 的写；
- 故事 P5 的接口版本，含被移出的项目管理员（工作区成员）重新加入得到 15，和 W7 的"降级含已离开的项目"。

## 2. 交付物

### 2.1 文件总览

路径相对于仓库根目录（`server/internal/` 省略）。"Task"是 plan 中负责它的任务（plan 有完整的文件表）。本 Phase 59 个手写的文件（新建 22 个、修改 37 个）、5 个生成物；没有迁移，没有跨模块的端口，不改 `identity`、`workspace` 的产品代码（`workspace` 只改一个 HTTP 测试里 `project.sole_admin` 的文字）。

| 路径 | 内容 | Task |
|---|---|---|
| `project/adapter/postgres/queries/members.sql`、`membership.go`、`membership_test.go`、`failures_test.go`；`project/app/member_ports.go` | 按 id 读、改角色、结束、其余的管理员；按用例分的端口 | 1 |
| `project/domain/membership.go`、`membership_test.go`、`errors.go` | 相对规则；三个码 | 2 |
| `project/app/lock.go`、`get_project.go`、`ports.go`、`update_member.go` 及测试、假实现、`clock_test.go`；`project/domain/actions.go` | 取锁路径的新分支；改角色的用例 | 3 |
| `api/modules/project.yaml`、`api/openapi.yaml`；`project/adapter/http/*`；`project/module.go`；`access/domain/rules.go` 及测试；前端文案；矩阵的六个文件；`bootstrap/project_write_locks_test.go` | 改角色的接口、规则、矩阵；资源路径的最先锁工作区 | 4 |
| `bootstrap/membership_world_test.go`、`project_roles_test.go` | 组合出的相对规则 | 5 |
| 同 Task 4 的接口、HTTP、规则、矩阵；`project/app/remove_member.go` 及测试；`bootstrap/project_removal_test.go` | 移出 | 6 |
| 同上；`project/app/leave_project.go` 及测试；`project/domain/errors.go`；`workspace/adapter/http/members_test.go`；`bootstrap/project_leaving_test.go` | 离开；`project.sole_admin` 的措辞 | 7 |
| `bootstrap/project_membership_races_test.go`、`project_membership_role_race_test.go`、`project_connection_test.go`、`project_writes_test.go` | 竞争、锁下重读到的角色、锁的强度、事务的连接、时刻和写者 | 8 |
| `bootstrap/interleaving_leaving_test.go` | 交错 1 的项目一侧；离开与 P5a 的移出 | 9 |
| 矩阵的种子；`bootstrap/project_writes_test.go`、`project_members_test.go`、`interleaving_growth_test.go`、`interleaving_writes_test.go`、`project_visibility_test.go`、`permission_matrix_members_test.go` | 替身换成 P5b 的写 | 10 |
| `e2e/fixtures/assert/project.ts`；`e2e/stories/project/p5-project-members.spec.ts` | 故事 P5 | 11 |

### 2.2 依赖

不加 Go 模块、npm 包、迁移。P4b 的 `Locks` 加一个分支、把锁的部分提出来共用（2.6）；P4b 的按项目 id 的写、P5a 的连带的代码不动。

### 2.3 存储（Task 1；3.5、3.6、3.7 规则 1、4.3）

四条查询，写进 `project/adapter/postgres/queries/members.sql`，都在 `ctx` 带着的事务里执行：

```sql
-- name: MemberByID :one
SELECT id, workspace_id, project_id, member_id, role, is_active
FROM project_members
WHERE id = $id AND deleted_at IS NULL;

-- name: UpdateMemberRole :one
UPDATE project_members
SET role = $role, updated_at = $now, updated_by_id = $updated_by::uuid
WHERE id = $id AND is_active AND deleted_at IS NULL
RETURNING id, project_id, member_id, role, created_at;

-- name: EndMember :execrows
UPDATE project_members
SET is_active = false, updated_at = $now, updated_by_id = $ended_by::uuid
WHERE project_id = $project_id AND member_id = $member_id AND is_active AND deleted_at IS NULL;

-- name: HasOtherAdmin :one
SELECT EXISTS (SELECT 1 FROM project_members
               WHERE project_id = $project_id AND member_id <> $member_id AND role = 20 AND is_active
                 AND deleted_at IS NULL);
```

- `Store.MemberByID(ctx, id) (app.ProjectMembership, found bool, error)`：有效的和已结束的都读出（已结束的由用例在判定之后答 404，3.6 约定二"依赖行的检查在判定之后"）；已删除的、没有的 `found` 为假。它不加锁：第一次读在任何锁之前（取它的项目和工作区），第二次在锁下（项目 N 之下，同一项目的写串行，读到的是最新的已提交版本）。
- `Store.UpdateMemberRole(ctx, id, role, by, now) (domain.Member, error)`：只写有效、未删除的那一行；不是恰好一行（`pgx.ErrNoRows`）是一个错误，不是 `ErrNotFound`（调用者在锁下读到它有效）。`created_*` 不动。
- `Store.EndMember(ctx, projectID, userID, by, now) error`：部分唯一索引保证一对最多一行未删除；不是恰好一行是一个错误。角色、`created_*` 不动（4.3：结束不删除）。移出按锁下读到的成员关系的项目和成员调用，离开按路径的项目和调用者调用；两者都在项目 N 之下。
- `Store.HasOtherAdmin(ctx, projectID, userID) (bool, error)`：`role = 20` 是"管理员"这个集合，不按大小比较。
- 测试：`TestMemberByID`、`TestUpdateMemberRole`、`TestEndMember`、`TestHasOtherAdmin`；`TestAFailedReadIsAnErrorNotAnAnswer`、`TestAFailedWriteIsAnError` 各加两个方法（清扫 19）。每个谓词都有一个只由它决定的行（另一个项目、另一个成员、另一个工作区、已结束的、已删除的行，已删除的行存在有效的之前或之后两个行序）；被写的行除了写的列一列不动，其余每行每列不动；被写的行之前由另一个账户写过（清扫 14）；`HasOtherAdmin` 的每个情形是自己的一个项目，另有一个项目有别的有效管理员（清扫 20：只问别的项目、把他自己算进去的实现各答错一个情形）。

### 2.4 端口（Task 1；6.7）

`project/app/member_ports.go`，按用例分：

```go
type ProjectMembership struct {
	ID, WorkspaceID, ProjectID, MemberID uuid.UUID
	Role                                 shared.Role
	Active                               bool
}
type MemberFinder interface {
	MemberByID(ctx context.Context, id uuid.UUID) (m ProjectMembership, found bool, err error)
}
type MemberRoleChanger interface {
	UpdateMemberRole(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) (domain.Member, error)
}
type MemberEnder interface {
	EndMember(ctx context.Context, projectID, userID, by uuid.UUID, now time.Time) error
}
type MemberLeaver interface {
	MemberEnder
	HasOtherAdmin(ctx context.Context, projectID, userID uuid.UUID) (bool, error)
}
```

`ProjectLocks`（`Locks` 的项目一侧）多 `MemberFinder`；`*postgres.Store` 实现全部，`project.New` 直接传 `store`。

### 2.5 相对规则与三个码（Task 2；3.5、5.3）

`project/domain/membership.go`：

- `CheckMemberRole(role) error`：三个角色之一，否则 422 `role` `invalid_format`（"is not 5, 15 or 20"）。只看请求的值，在事务之前（3.6 约定二"只看请求的值的校验在事务之前"）。
- `RoleChange{Caller shared.Grant, Own bool, From, WorkspaceRole, To shared.Role}`；`CheckRoleChange(c)`：
  1. 调用者不是工作区管理员时：自己的 409 `project.own_membership`；`From` 不在 `rolesBelow(调用者的项目角色)` 里、或 `To` 不在其中，403 `project.role_too_high`。所以项目管理员（不是工作区管理员）不能改另一位管理员、不能提拔任何人为管理员；项目成员（只有同时是工作区管理员时才过得了判定，见 2.7）在这里不会出现，单元测试照样核对他只能把访客保持为访客。
  2. 对每个调用者：`To` 不在 `assignable[WorkspaceRole]` 里 422 `role` `not_allowed`（"must be 5: the member is a guest of the workspace"）：工作区访客只能是访客（Plane `views/project/member.py:257-261`）。
- `CheckRemoval(callerRole, own, role)`：自己的 409 `project.own_membership`（工作区管理员也是："请用离开"）；`role` 不在 `rolesUpTo(callerRole)` 里 403 `project.role_too_high`（工作区管理员同样没有例外，3.5；Plane `:313`）。同级的可以移出：项目管理员移出另一位管理员（故事 P5）。
- `rolesBelow`、`rolesUpTo` 按 `roleOrder` 里的位置取集合：三者之外的角色下面什么都没有，也不在任何角色下面。按数值比较的实现在 10、25 这两个情形失败（`TestCheckRoleChange`、`TestCheckRemoval`）。
- 三个码（`project/domain/errors.go`）：

  | 码 | 状态 | 文字 |
  |---|---|---|
  | `project.member_not_found` | 404 | The project member does not exist, or you cannot see the project. |
  | `project.own_membership` | 409 | You cannot remove your own membership of the project, nor change your own role in it unless you are a workspace admin. |
  | `project.role_too_high` | 403 | The role is too high for you: unless you are a workspace admin, you change only a member whose project role is below yours, to a role below yours; and you remove only a member whose project role is not above yours. |

  每句对每个收到它的调用者都成立（清扫 30）：404 给看不到项目的人、成员关系不存在或已删除或已结束（已结束只对过了判定的人）；409 给移出自己的任何人和改自己角色的非工作区管理员（改自己的工作区管理员从不收到它）；403 的两半各说一种写，"unless you are a workspace admin"只在改角色一半。`project.own_membership` 不给补救：离开是结束自己的成员关系的路，但它同样拒绝唯一的管理员，"请用离开"对唯一的管理员不成立（第 3 节第 7 条）。

### 2.6 按成员关系 id 寻址的取锁路径；`updateProjectMember` 的用例（Task 3；3.3、3.5、3.6 约定二、三和加锁表、6.7）

`project/app/lock.go`：

```go
func (l Locks) lockMemberAndDecide(ctx context.Context, actor shared.Actor, id uuid.UUID, action shared.Action, target bool) (held, error)
func (l Locks) member(ctx context.Context, id uuid.UUID) (ProjectMembership, error)
func (l Locks) lock(ctx context.Context, workspaceID uuid.UUID, w write, notFound error) (held, error)
func decide(ctx context.Context, auth shared.Authorizer, actor shared.Actor, action shared.Action, workspaceID, id uuid.UUID, notFound error) (shared.Grant, error)
```

- `lockMemberAndDecide`：`member(id)`（不加锁；没有 404 `project.member_not_found`；回答的 id 不是问的，是写自己的错误，清扫 22）→ `lock(m.WorkspaceID, write{project: m.ProjectID, action, targets: [m.MemberID] 当 target}, ErrMemberNotFound)`：工作区 S（已删除 404）→ `target` 时成员在工作区的成员关系 S（`ShareMembers`，回答 `h.roles`；第 3 节第 3 条）→ 项目 N（已删除、不在这个工作区 404）→ `member(id)` 在锁下再读（没有 404，`ProjectID` 不是第一次读到的 404）→ `decide(…, ErrMemberNotFound)`（看不到项目 404，角色不够是 `Authorizer` 的 403）。`held.member` 是锁下读到的成员关系，用例只用它。
- `lockAndDecide`（P4b，按项目 id）改为经 `lock`，行为不变；`decide` 多一个 `notFound` 参数，各路径答各自的 404（第 3 节第 4 条）。
- 已结束的成员关系不在这里拒绝：它是"依赖行的检查"，在判定之后（3.6 约定二），看不到判定通过的人只得到判定的 403，不知道它的状态（矩阵的已结束一行：PM 得到 403）。
- `UpdateProjectMember`（`NewUpdateProjectMember(locks Locks, members MemberRoleChanger, tx shared.TxManager, clock Clock)`），`Execute(ctx, id, role) (domain.Member, error)`：`RequireActor` → `CheckMemberRole` → 一个事务：`lockMemberAndDecide(…, ActionMemberUpdate, true)` → 已结束 404 → `h.roles[m.MemberID]`（没有他是写自己的错误：项目成员一定是工作区的有效成员，约定六的每条收缩都同时结束他的项目成员关系）→ `CheckRoleChange{h.grant, m.MemberID == actor.UserID, m.Role, 他的工作区角色, role}` → `clock.Now()` → `UpdateMemberRole(m.ID, role, actor.UserID, now)`（回答的 id 不是这个成员关系，是写自己的错误）。检查的顺序与 P5a 的 G2 相同：重读 → 判定 → 已结束 404 → 自己 409 → 相对 403 → 访客的上限 422 → 时钟 → 写。
- 单元测试（`update_member_test.go`）的调用记录核对整个顺序；`outcome.check` 要求回答的第一个 `*shared.Error` 就是期望的那个（不是 `errors.Is`，清扫 21），失败原样返回，事务的函数返回的正是用例的回答（`fakeTx.answered`）。

### 2.7 `updateProjectMember` 的接口（Task 4；3.4、3.5、5.1、5.3、9.2、9.4）

- `PATCH /api/v0/project-members/{project_member_id}`，正文 `ProjectMemberUpdate{role}`（`additionalProperties: false`，只有 `role`），200 回答 `ProjectMember`；码 `[validation_failed, project.member_not_found, forbidden, project.own_membership, project.role_too_high]`。描述照 2.5、2.6 的顺序写，每句由测试持住（清扫 18）。参数 `ProjectMemberID` 的描述说明它是成员关系的 id，不是成员的账户 id；`ProjectMember.id` 的描述说明 `/project-members/{project_member_id}` 指它。
- 操作名 `project_member.update`，规则 `{Level: LevelProject, Roles: [admin]}`：项目管理员；同时是工作区管理员的项目成员由 `LevelProject` 的工作区管理员规则放行（3.4，同 `project_member.add`）；不是成员的工作区管理员 403（先加入）。
- 三个新码进 `PROBLEM_MESSAGES` 和两份 `auth.json`（约束 4）。
- `project.New` 的 `UpdateMember: app.NewUpdateProjectMember(locks, store, d.Tx, d.Clock)`；`member(m)` 把 `domain.Member` 写成接口的形状，列表和改角色共用。

### 2.8 组合出的相对规则（Task 5；brief 清扫 5、23、24、26）

`memberWorld`（`bootstrap/membership_world_test.go`）：组合出的 app 和自己的数据库，经接口建出两个工作区、三个项目，同一批账户在一处是管理员、在另一处是成员（清扫 23）：

| 地方 | 管理员 | 成员 | 访客 |
|---|---|---|---|
| acme | alice、gina（alice 改的） | bob、carol、dave | erin |
| beta | bob | carol | |
| Web（acme，alice 的） | alice、bob、dave | carol、gina | erin |
| Ops（acme，alice 的） | alice、carol | bob | |
| Lab（beta，bob 的） | bob | carol | |

`TestTheRelativeRuleOnTheComposedApp` 一步接一步，3.5 的每一半各有成立的情形和反例（清扫 24），每一步的拒绝一行不改，每一次改只写目标的角色、由调用者、在请求的时刻（目标之前由 alice 写），其余的列不动：规则的角色（carol 是 Web 的成员：连把访客保持为访客也 403；她是 Ops 的管理员：改 bob）；自己的（bob 是 beta 的工作区管理员，在 acme 不是：409）；另一位管理员、提拔（403 `project.role_too_high`）；工作区访客的上限（bob、gina 各 422）；工作区管理员的例外（gina 降 dave、提拔 carol、提拔自己）。"角色不变"在非管理员的行上核对（清扫 26）。

### 2.9 `removeProjectMember`（Task 6；3.3、3.5、5.1、9.2）

- 接口：`DELETE /api/v0/project-members/{project_member_id}`，204 无正文；码 `[project.member_not_found, forbidden, project.own_membership, project.role_too_high]`。
- 操作名 `project_member.remove`，规则同 `project_member.update`。
- 用例（`NewRemoveProjectMember(locks, members MemberEnder, tx, clock)`），一个事务：`lockMemberAndDecide(…, ActionMemberRemove, false)`（移出不读成员的工作区角色，不锁他在工作区的成员关系）→ 已结束 404 → `CheckRemoval(h.grant.ProjectRole, 是自己的, m.Role)` → 读时钟 → `EndMember(m.ProjectID, m.MemberID, actor.UserID, now)`。行和角色留着，他在项目里的显示设置留着（与 Plane 相同，P5a 第 7 节）。
- 过了判定的调用者是项目管理员，或同时是工作区管理员的项目成员；后者不能移出比他高的角色：移出之后项目照旧有管理员（项目管理员移出另一位管理员时，他自己还在）。
- 重新加入：被移出的项目管理员（工作区成员）加入公开项目，原来那一行回来，角色 `min(20, 15) = 15`（3.5、约定六；P4b 的 `RestoreMember`）：`TestRemovingAProjectMember` 和故事 P5 各一次。
- `bootstrap/project_removal_test.go`：`memberEnding` 和 `check`（结束或拒绝的一步的共同核对，移出和离开共用，清扫 3、14、27）；`TestRemovingAProjectMember`（plan Task 6）。

### 2.10 `leaveProject`；`project.sole_admin`（Task 7；3.7 规则 1、5.1、9.2；P5a review 第 6 节）

- 接口：`POST /api/v0/projects/{project_id}/leave`，204 无正文；码 `[project.not_found, forbidden, project.sole_admin]`。
- 操作名 `project.leave`，规则 `{Level: LevelProject, Roles: [admin, member, guest]}`：每个有效成员（同 `project_member.list`）；看得到而不是成员的 403，看不到的 404。
- 用例（`NewLeaveProject(locks, members MemberLeaver, tx, clock)`），一个事务：P4b 的 `lockAndDecide(write{project, ActionLeave})`（工作区 S → 项目 N → 判定；加锁表"离开项目"一行）→ `h.grant.ProjectRole == RoleAdmin` 时 `HasOtherAdmin(project, 他)`，没有 409 `project.sole_admin`（他是唯一的成员时也是：规则 1"哪怕只有他一人"）→ 读时钟 → `EndMember(project, 他, 他, now)`。
- 规则 1 看锁下判定得到的项目角色（`ProjectRole`），不看 `ProjectAdmin`：同时是工作区管理员的项目成员照成员离开，不问另一位管理员；被降为访客的前管理员照访客离开（`TestLeavingAProject` 的最后三步）。
- `project.sole_admin`（`domain.ErrSoleAdmin`）的文字为两条规则措辞：

  > The project would be left without an admin: its only active admin cannot leave it, nor can his membership end while it has other active members. Give the project another admin first, or delete it.

  收到它的有三种调用者，补救对每一个都成立（清扫 30）：离开项目的唯一管理员（他可以添加一位管理员，或请工作区管理员加入：工作区管理员加入任何项目都以管理员加入，3.5；或删除项目）；移出工作区成员的工作区管理员（他自己可以加入那个项目成为管理员，或作为项目成员提拔别人，3.5 的例外）；离开工作区的唯一项目管理员（同第一种）。`workspace` 的 HTTP 测试（它声明这个码而不导入 `project`，9.4）和两份 `auth.json` 同改。原来的"make another of its members an admin first"对不是工作区管理员的项目管理员不成立：相对规则不许他提拔任何人为管理员。

### 2.11 竞争、锁的强度、事务的连接、时刻和写者（Task 8；brief 清扫 8、9、12、13、14、24、29）

全部在组合出的 app 上（`memberWorld`，`membershipWrites`：bob 改 gina 的角色、移出她，dave 离开 Web）：

- `TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile`：另一个事务持 Web 的 `FOR NO KEY UPDATE`（或 acme 的，删除 acme 时）并改一行；写已过认证、不加锁读过它指的成员关系，等在 Web（或 acme）的行上（`WaitForLockWaitOn`）；另一个提交之后都是 404、从不是泄露存在的 403，另一个改的行是它留下的样子，其余每行不变。Web 是私有的（`newPrivateWorld`），调用者的成员关系结束之后他看不到它：

  | 写 | 等待期间 | 回答 |
  |---|---|---|
  | 改角色、移出 | gina 的成员关系结束 | 404 `project.member_not_found`（已结束，在判定之后） |
  | 改角色、移出 | gina 的成员关系删除 | 404 `project.member_not_found`（锁下重读没有） |
  | 改角色、移出 | gina 的成员关系移到 Ops | 404 `project.member_not_found`（不再属于这个项目） |
  | 改角色、移出、离开 | 调用者自己的成员关系结束 | 404 `project.member_not_found`、离开 `project.not_found`（看不到） |
  | 改角色、移出、离开 | Web 删除 | 同上（项目锁读不到行） |
  | 改角色、移出、离开 | acme 删除 | 同上（工作区锁读不到行） |

- `TestAWriteOnAProjectMembershipDecidesOnWhatItReadUnderItsLocks`（`bootstrap/project_membership_role_race_test.go`，预检 M-1、L-6）：另一个事务持 Web 的 `FOR NO KEY UPDATE`，把写指的成员关系改为管理员或删除；写在 Web 的行上等，另一个提交之后：bob（Web 的管理员，不是工作区管理员）改 carol 的角色、gina（Web 的成员、acme 的管理员）移出 carol，各 403 `project.role_too_high`（相对规则用锁下重读到的角色，不用锁之前读到的）；carol（Web 的成员，无权改、移出）改、移出 gina 已删除的成员关系，各 404 `project.member_not_found`（重读在判定之前：删除的答 404，不是她的 403）；那一行是另一个留下的样子，其余每行不变。等锁时被提拔的目标不会被一个按旧角色判定的写降级或移出。
- `TestEachLockOfAWriteOnAProjectMembershipIsItsStrength`：别的事务持 acme（`FOR NO KEY UPDATE`）、Web 和写改的那一行（`FOR SHARE`），逐个放开；每一步用 `lockOn`（P4b 的 `NOWAIT` 阶梯）读出各行最强的锁：

  | 写等的表 | 那时持有 |
  |---|---|
  | `workspaces` | 什么都没有（成员关系的第一次读不加锁） |
  | `projects` | acme `FOR SHARE`；改角色时 gina 在 acme 的成员关系 `FOR SHARE`，不更强；移出、离开不锁它 |
  | `project_members` | 再加 Web `FOR NO KEY UPDATE`；Ops（gina、dave 都不是成员）没有锁 |

  然后它照单独时回答；写下的时刻不早于 Web 被放开（时钟在锁之后读，3.3）。成员关系的行本身只由写的 `UPDATE` 锁（`FOR NO KEY UPDATE` 的行锁），不在判定之前。
- `TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（P4b）加三个写：alice 把 carol 改为 Web 的管理员、移出 bob，carol 离开；池只有一个连接，任何一条语句经连接池就等一个不来的连接。
- `TestTheWritesOnAProjectStampTheirRequest`（P4b）加三行，各由 alice、在请求之内的一个时刻写；每行之前由 bob 写（清扫 14）。
- 每个等待都有期限（`soon(t)`、`receiveWithin`、`WaitForLockWaitOn` 的 5 秒；清扫 29）；池的大小照 `openPool`。

### 2.12 交错（Task 9；3.6、3.7 规则 1、2，9.3）

`bootstrap/interleaving_leaving_test.go`。离开经项目的 `LeaveProject` 用例，`Locks` 照 `project.New` 接；移出经工作区的用例，带项目的连带，照 `bootstrap` 接；第一方停在一个 gate（`projectOtherFoundHolding`：问过另一位管理员之后、结束之前；`projectEndedHolding`：写完成员关系之后），第二方在它说的那一行上等，由 `pgtest.WaitForLockWaitOn(…, "projects" | "workspaces", 5*time.Second)` 证明：两方都不写那张表，探针只由那一把锁的等待满足（P5a T10-a 的那一类）。每个等待都有期限。

| 交错 | 测试 | 第一方 → 第二方 | 结果 |
|---|---|---|---|
| 1（项目一侧） | `TestTwoProjectAdminsLeavingLeaveAnAdmin` | Ops 的两位管理员 alice、carol 同时离开（两个 gate × 两种先后，4 个） | 第二位在 Ops 的行上等，之后 409 `project.sole_admin`；Ops 留一位管理员 |
| 离开 ↔ P5a 的移出（同一个成员） | `TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize` | carol 离开 Ops ↔ gina 把 carol 移出 acme | 离开先：她在 Ops 的由她自己结束，移出结束其余的；移出先：全部由 gina 结束，离开 404 `project.not_found` |
| 离开 ↔ P5a 的移出（另一位管理员） | 同上 | alice 离开 Ops ↔ gina 把 carol 移出 acme | 离开先：移出在 Ops 的锁下看到 carol 是唯一的管理员（bob 是成员），409（规则 2）；移出先：离开在 Ops 的锁下找不到另一位有效管理员，409（规则 1）；Ops 留一位管理员 |

停在问过之后的 gate 证明规则 1 在结束它的同一把锁下问：先在一个事务里问、再在另一个事务里结束的离开在这里失败（两位都离开，P5a T6-b 的写法）。

**P5b 的写与 P5a 的结束不成环**（P5a spec 第 5 节）：P5b 的三个写最先取工作区的 S，P5a 的移出、离开最先取工作区的 N（停用在 P6，同样先取工作区的 N）；两者在工作区行上串行，谁先拿到它，谁就在另一方取任何别的锁之前做完。P5a 的连带在工作区 N 之下按 id 锁项目，这时没有 P5b 的写持有或等待这个工作区里的项目行（约定五）。反过来，P5b 的写持项目 N 时，它已持工作区 S，P5a 的写等在工作区行上，不会先持项目再等工作区。上表后两行在真实数据库上两种顺序都证明它，`-count=5 -race` 没有 40P01。规则 2 在项目 N 之下查（P5a 的 `SoleAdmin`），所以它看得到先提交的 P5b 的写改了谁是管理员（上表最后一行的离开先）。

### 2.13 矩阵与完整性核对（Task 4、6、7；9.2）

- 改角色四行、移出四行（各 12 格）、离开一行（12 格），共 108 格；矩阵共 523 格。每一格指向它这一列的项目里种下的成员关系（`toProjectMembership`、`aMembership`），目标核对拒绝别的列的项目的、没有种下的、工作区一级的表里的（`TestMatrixViolationsCatchesEachColumnGap` 的四个情形）。
- 前提（`projectSeed.memberships`）：每个被指到的成员关系的状态和角色（PM、PA、PM+WA、工作区访客的有效，被移出的成员、P-前的已结束未删除，gone 的成员、管理员的随 gone 删除）；工作区访客是 acme 的有效访客；PA 在 acme 的公开、私有项目里没有另一位管理员（离开的 409 是规则 1 的）。
- 唯一管理员的项目一侧是离开一行的 PA 格子（409 `project.sole_admin`），在项目表的列上；不另建表（第 3 节第 5 条）。`projectTables`、`workspaceLevelTables` 不变，`TestEachMatrixTableIsOfOneLevel` 照旧通过。
- 完整性核对（`writesOnAProject`）：离开的路径有 `{project_id}`（规则 (a)）；改角色、移出的路径没有，它们的矩阵行的列是项目表的（规则 (b)）：P5b 的这两个写是第二种形状的第一批。三行各去掉一行都被发现（附录 A，`pf-row-*`）。
- `TestEachWriteOnAProjectSharesItsWorkspaceFirst`：`projectWrite.param()` 接资源路径；`by`、`member` 两列；第一步除了项目的行、目标在工作区的成员关系，还用 `FOR UPDATE NOWAIT` 探测写改的成员关系的行（P4b review 第 6 节：先锁资源行再锁工作区的写在这里失败，`s9-member-row-first`）；第二步核对它在项目之后才锁。

### 2.14 替身换成 P5b 的写（Task 10；9.2；P5a spec 第 5 节）

- 矩阵：`projectSeed.endings()` 经项目的存储 `EndMember` 结束 P-前在私有项目的成员关系（由 acme 的管理员：移出）和 WG- 在公开项目的（由他自己：离开），在种子的时刻。`standIns` 删除。
- 留下的 SQL，每一处都写明为什么没有写会留下那个状态：
  - `partingStates`：两种已删除的成员关系（WG- 在私有项目的、成员在私有项目的：已删除的成员关系只随已删除的项目、工作区出现，列表须照样跟读取一致）、PM 在私有项目的显示设置删除而成员关系有效；
  - `project_writes_test.go` 的"恢复"一行和 `project_members_test.go` 的提交时被拒：之前没有显示设置的已结束成员关系（移出、离开都留着显示设置）。
- 改由 P5b 的写：负责人、默认指派人的测试里 bob 由 alice 经接口移出；恢复的测试由 alice 经接口改角色、移出；`newGrowthRace(ended)` 用 `EndMember`；`bobAdministersWeb` 用 `UpdateMemberRole`。
- 添加的"成员关系已结束"的变体、建项目的负责人变体（P4b M4）照旧以被移出的成员为目标，各只答一个错误（矩阵的格子核对整个错误列表）。

### 2.15 端到端（Task 11；第 2 节 P5、W7，9.6）

- 夹具：`MemberRow`、`expectMembers(db, projectId, want)`：项目未删除的成员关系（已结束的也在），按地址排序，每行带最后写它的账户、他未删除的显示设置和写它的账户，每行都属于项目的工作区。
- 故事 P5（API）照设计第 2 节的接口一列，为清扫加了几步（第 3 节第 10 条）：添加的三个 422；一次添加两人（成员、访客）；唯一管理员离开 409，项目成员改别人的角色 403；项目管理员（不是工作区管理员）改另一位管理员 403 `project.role_too_high`；把成员改为访客、移出访客；成员离开，他在另一个项目的成员关系不动，降为工作区访客之后已结束的那一行也成为访客的（W7 的一半）；项目管理员移出另一位项目管理员（工作区成员），他加入公开项目：原来那一行回来，15。设计说"管理员……把成员改为访客，移出访客"：故事里是另一位项目管理员 pam 做这两步，这样行的写者（之前由管理员写）可以失败（清扫 14）。
- 页面版本（成员点"离开项目"先等接口成功再回到项目列表，M1-P4 交接）在 P10。

### 2.16 文档

- 3.20、8.7 没有 P5b 的行（设计 12 节 P5b 任务 9）。
- `docs/v0/plane-diff.md` 不加行：改角色、移出、离开的规则与 Plane 相同（3.5"照搬"）；Nerve 的状态和码（403、409、422，各有 `x-problem-codes` 声明的码）与 Plane 的 400 加一句话不同，属于第三节"错误""错误码"两行的总条目；Plane 的 `PATCH` 经序列化器还能改 `is_active`，Nerve 的 `ProjectMemberUpdate` 只有 `role`、结束只经移出和离开，这与 P2 的工作区成员的 `PATCH` 同一种情形，P2 没有为它加行（Plane 的工作区成员的 `partial_update` 同样经序列化器），这里照旧。

## 3. 与设计的差异和补充（控制者已裁定，2026-10-03）

以下都没有改变 M3 设计的架构。每条后面是控制者的裁定（`$M3TMP/p5b-architect-rulings.md`）。

1. **任务的划分：11 个，不是 9 个**。设计的任务与 plan 的对应：1（取锁路径、存储、最先锁工作区的测试、完整性核对的第二种形状）→ 1、3、4（存储一个 Task，取锁路径和第一个使用它的用例一个 Task：路径没有使用者就测不到调用顺序；最先锁工作区的测试随第一个资源路径的接口，约束 1）；2（相对规则与改角色的用例）→ 2、3；3（改角色的接口、规则、矩阵、码）→ 4，组合出的相对规则另成 Task 5（清扫 5、24 要每一半在组合一层有反例，放进 Task 4 会让它超过约 1,500 行）；4 → 6；5 → 7；6（竞争、锁的强度、交错 1）→ 8、9；7（矩阵的剩余行、P-前）→ 4、6、7（每个操作的矩阵行随它的接口，约束 1）和 10（替身）；8 → 11；9（review）是控制者的。最大的是 Task 4（1,195 行）、Task 3（884 行）、Task 7（867 行）；plan 共 6,688 行。11 个任务在约 16 个以内，每个在约 1,500 行以内。**已接受。**
2. **移交的落点**（说明）：

   | 移交 | 落点 |
   |---|---|
   | P5a spec 第 5 节：P-前和 `partingStates` 换成 P5b 的写 | Task 10（2.14）：两个已结束的成员关系由 `EndMember` 写出；`partingStates` 只留没有写会留下的已删除状态和显示设置 |
   | 同上：唯一管理员的项目一侧，完整性核对照旧成立 | Task 7：离开一行的 PA 格子，在项目表的列上（第 5 条） |
   | 同上：最先锁工作区的测试接资源路径、第一步探测资源行 | Task 4（2.13）；资源行的探测有反例 `s9-member-row-first` |
   | 同上：交错 1 的项目一侧 | Task 9（2.12） |
   | 同上：`Memberships` 的账户由故事 P5 一次添加几个账户 | Task 11：P5 一次添加两人、再一次添加三人；这个谓词在每个调用者之下等价（P5a 第 3 节第 9 条），故事照样看不到，附录 A 的 `x-memberships-account` 记下 |
   | 同上：P5b 的写与 P5a 的连带不成环，两种顺序 | Task 9（2.12 的论证和后两行） |
   | 同上：新矩阵表放进 `projectTables` 或 `workspaceLevelTables` | P5b 没有新表：九行都在 `projectColumns` 上，它已在 `projectTables` 里（2.13） |
   | 同上：`project.ErrSoleAdmin` 的说明和补救 | Task 7（2.10；第 7 条） |
   | 同上、P5a review 第 6 节 m5：W7 的"降级含已离开的项目" | Task 11：tom 离开 Web 之后被降为访客，已结束的那一行也成为访客的；`s15-demote-skips-ended` 让 P5 和 W7 都失败 |
   | P5a review 第 6 节：交错 1 的项目一侧在问过之后另有一个 gate、探针只认锁的等待、每个等待有期限 | Task 9：`projectOtherFoundHolding`；两方都不写被探的表；`soon(t)`、`receiveWithin` |
   | P4b review 第 6 节（经 P5a）：设计 `Locks` 的新分支、在组合一层证明 | Task 3（2.6）、Task 8（2.11 的两个测试） |
   | 同上：等锁期间成员关系结束、删除，项目、工作区删除，组合一层 404 不是 403 | Task 8（`TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile` 的 15 个情形） |
   | 同上：P8 的完整性规则，P5b 的写是第二种形状的第一批 | Task 4、6（2.13）；附录 A 的 `pf-row-*` |
   | 同上：e2e 的 `expectProjectDeleted` 只数删除之前未删除的行（M5） | 不适用：故事 P5 不删除项目 |
   | 同上：添加、建项目负责人的"已结束"变体各只答一个错误（M4） | Task 10（2.14）：照旧以被移出的成员为目标，格子核对整个错误列表 |
   | brief 第 4 条：规则 2 看得到 P5b 的写改了谁是管理员 | Task 9（2.12 最后一行的离开先：移出在 Ops 的锁下看到 carol 是唯一的管理员） |
   | 一直有效的规则：锁之后读时钟；锁下重读确认父行；目录驱动的检查；`apitest.Main` 两个方向；端口的错误原样返回；角色按集合；关键词守卫 | `TestEachWriteReadsTheClockUnderItsLock` 三行（Task 3、6、7）；`lockMemberAndDecide` 的重读（Task 3）；`table_rows_test.go` 和组合出的删除测试不加豁免照旧通过（P5b 没有新表）；Task 4、6、7；清扫 2；`roleOrder` 的集合；契约措辞没有新的命中（`make lint-web` 5 个命中照旧都有例外） |

3. **改角色锁住成员在工作区的成员关系（`FOR SHARE`）**：3.6 的加锁表"改项目成员的角色、移出项目成员"一行是"工作区 S → 项目 N → 重读项目成员 → 判定 → 改"，没有这一把锁。改角色要读成员的工作区角色（访客的上限，3.5）。方案 E 之下，工作区 S 已经挡住每一个改 `workspace_members` 的写（改角色、移出、离开、接受、恢复、停用都先取工作区的 N），所以不加锁读也读得到一致的值；P5b 仍经 `Locks` 的 `targets` 取它：这是添加、加入已用的路径（约定三的 `ShareMembers`，回答 `h.roles`），不需要新的端口，锁在全局顺序里（`workspace_members` 在 `workspaces` 之后、`projects` 之前），`TestEachLockOfAWriteOnAProjectMembershipIsItsStrength` 钉住它只在改角色时取、不更强。代价是一行共享锁。另一种写法：给 `Locks` 加一个不加锁读工作区角色的端口（`WorkspaceMembers` 多一个方法），加锁表照原样。**已接受。** 设计 3.6 的加锁表这一行已在修订一轮改为"工作区 S →（改角色）成员的工作区成员行 S → 项目 N → 重读项目成员 → 判定 → 改"，并写明理由：工作区访客的上限（3.5）读成员的工作区角色，移出不读它、不锁（设计提交 `0818c806`，P5b 一节的目标和第一个任务同改）。
4. **按成员关系 id 的写，看不到项目时答 `project.member_not_found`**：3.6 说按资源 id 的写"没有时答资源的 404"；看不到项目也是资源的 404（8.2：不泄露存在），所以 `decide` 多一个 `notFound` 参数，按项目 id 的写照旧 `project.not_found`。P4b 的 `lockAndDecide` 的锁部分提成 `lock`，两条路径共用；它的行为和调用顺序不变（P4b 的单元测试和组合测试照旧通过）。**说明。**
5. **唯一管理员的项目一侧不另建表**：P5a spec 第 3 节第 6 条留给 P5b 两种做法：另建项目的一张表，或在项目表里给出格子。离开一行在 `projectColumns` 上，PA 一格是 409 `project.sole_admin`：矩阵的 PA 是每个项目唯一的有效管理员（前提核对），所以这一格就是规则 1；规则 1 的其余情形（只有他一人、另一位已结束、规则看项目角色）在组合的 `TestLeavingAProject`。不建新表，`projectTables` 和完整性核对照旧成立。**已接受。**
6. **离开的规则 1 看项目角色，不看 `ProjectAdmin`**：同时是工作区管理员的项目成员在判定里是"项目管理员"（`ProjectAdmin` 为真，3.4），但他不是项目的管理员，离开时不问另一位管理员；他离开之后项目仍有它的管理员。反过来，被降为访客的前管理员不再算管理员。`TestLeavingAProject` 的 gina 一步和最后两步各是一个反例（`s24-leave-rule1-workspace-admin`）。**说明。**
7. **两个码的说法**：
   - `project.sole_admin` 照 P5a review 第 6 节为两条规则措辞，不给离开自己的码：两条规则说的是同一件事（项目会没有管理员），补救相同（先给项目另一位管理员，或删除它），对三种收到它的调用者都成立（2.10）。原来的"先让另一位成员成为管理员"对不是工作区管理员的项目管理员不成立：相对规则不许他提拔任何人为管理员。补救里没有"请工作区管理员"：给项目另一位管理员的路有几条（添加一位管理员、工作区管理员加入或提拔），文字不替调用者选。
   - `project.own_membership` 不给补救：移出自己的人该去离开，但离开同样拒绝唯一的管理员，"请用离开"对他不成立；改自己角色的非工作区管理员没有别的路。`removeProjectMember` 的描述同样不给它补救（预检 L-4：原来的"one leaves a project through leaveProject"已删去）。**已接受。**
8. **故事看不到的谓词**（清扫 1 的故事一半，附录 A）：
   - 四条查询的 `deleted_at IS NULL`：项目成员关系只随项目、工作区删除（4.3），已删除的项目在锁里读不到，故事里没有一个已删除而项目未删除的成员行。`MemberByID` 的这一条另由组合的竞争测试（等锁期间删除成员关系）和矩阵（gone 的成员关系）发现。
   - `UpdateMemberRole`、`EndMember` 的 `is_active`：用例在锁下先读到有效才写（改角色、移出的已结束 404；离开的判定要求有效成员），这个谓词在组合一层等价。
   - `HasOtherAdmin` 的 `is_active`：组合的 `TestLeavingAProject` 发现（carol 已结束的管理员成员关系），故事里没有已结束的管理员而项目只剩一位的时刻。
   - `Memberships`（P4b）的账户：每个调用者都按问到的账户取结果（第 2 条）。

   每一个都由存储测试在两个行序下发现。安全性质的变异里故事看不到的，每个都由矩阵或组合测试发现：
   - 规则不给项目访客、移出的规则给项目成员：故事里离开的是成员，项目成员不移出别人（矩阵、`TestLeavingAProject`、`TestRemovingAProjectMember`）。
   - 移出自己的、改自己的角色：故事里没有人这样做（矩阵、`TestRemovingAProjectMember`、`TestTheRelativeRuleOnTheComposedApp`）。
   - 每个离开的人都问另一位管理员、只有他一人照样离开：故事里离开的人的项目有别的管理员，唯一管理员的项目有别的成员（`TestLeavingAProject`；后者另由 `TestHasOtherAdmin`）。
   - 不在锁下重读成员关系：故事是顺序的，锁前锁后两次读相同（`TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile`）。

   预检的 M-1、L-6 原来不在这张表里，修订之后都在组合一层被发现：相对规则的目标角色取锁下重读到的那个（`pf-stale-target-role`：用锁之前读到的角色，等锁时被提拔为管理员的人会被不是工作区管理员的项目管理员降级、被工作区管理员移出），判定在重读之后（`pf-decide-before-reread`：删除的成员关系对无权写的调用者答 403 而不是 404），都由 `TestAWriteOnAProjectMembershipDecidesOnWhatItReadUnderItsLocks`（Task 8）发现；单元一层另由 Task 3、6 的单元测试发现（目标的角色：拒绝表各加的一行；判定的次序：调用记录）。

   **只在单元一层被发现的变异**（附录 A），每个的理由：
   - 等锁时删除的工作区不算 404（`s12-workspace-gone-ignored`，P4b 起的 `lock` 的一步）：删除工作区在同一个事务里软删除它的项目（加锁表"删除工作区"一行），等过工作区之后项目的锁读不到行，同样答 404、一行不改；组合一层没有"工作区已删除而项目未删除"的状态，这一步是冗余的防线。`TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile` 的 acme 删除情形核对结果。
   - 端口的失败被吞掉、换成 404、重试、包进别的问题（清扫 2、21 的 `s2-*`、`s21-upd-failure-as-404`）和回答别的 id（清扫 22 的 `s22-*`）：组合一层的真实存储不失败、不答错 id，无从注入；这些不是授权或加锁的性质，单元测试比较到失败那一步为止的整个调用记录。
   - 相对规则、移出的规则按数值比较（`set-change-magnitude`、`set-removal-magnitude`）：只在三个角色之外的值上与按集合不同；组合一层到不了规则的角色只有三个（请求的角色先由 `CheckMemberRole` 拒绝，422；存储的角色由 `project_members.role` 的 `CHECK (role IN (5, 15, 20))` 限定），所以等价。领域测试的 10、25 的情形守住"按集合"这条写法。离开的规则 1 改成 `>=`（`set-leave-magnitude`）连单元一层也等价：它比较的只是存储的角色。
   - 契约多声明一个从不回答的码（`s18-remove-code-unused-module`）：由 `project/adapter/http` 包的 `apitest.Main` 在包的测试结束时发现，这一核对本来就在单元一层。**说明。**
9. **共用的测试逻辑写一次**（brief"不重复测试逻辑"）：`memberWorld`（Task 5）是改角色、移出、离开、竞争、交错的同一个世界；`memberEnding.check`（Task 6）是移出和离开的每一步的同一个核对；`outcome.check`、`memberLocked`（Task 3）是三个用例的单元测试共用的；`membershipWrites`（Task 8）是两个组合测试共用的三个写。**说明。**
10. **故事为清扫多加的步骤**（设计第 2 节的 P5 照旧都在）：改角色、移出由另一位项目管理员 pam 做（行之前由管理员写，清扫 14）；Ops 是 acme 的另一个项目，tom 是成员、wanda 是另一位管理员（`EndMember`、`HasOtherAdmin` 的项目谓词在故事里看得到，清扫 1、20）；添加的三个 422 各一次，wanda（工作区管理员）以成员添加是设计列的"工作区管理员加为管理员以外的角色"；离开和降级之间、移出和重新加入之间各读一次成员关系（清扫 27）；重新加入之后成员列表里是他原来的 id。设计第 2 节 P5 的"成员点'离开项目'"在接口一列是 tom 的离开。**已接受。**
11. **一个项目可以没有管理员**：3.7 说同时是工作区管理员的项目成员可以把唯一的项目管理员改成成员，"这时项目由工作区管理员管理"；P5b 照做（`TestTheRelativeRuleOnTheComposedApp` 的 gina 降 dave 时 Web 还有 bob，`TestLeavingAProject` 的降级让两个项目都没有管理员）。工作区访客的降级同样可以（P5a 第 7 节）。这样的项目里，工作区管理员加入就是管理员（3.5：加入的角色是工作区角色）。**说明。**
12. **plane-diff 不加行**（2.16）。**说明。**

## 4. 验收标准（完成线，M3 设计 12 节 P5b）

- [ ] P5 的接口版本通过，此前的每个故事仍然通过（`make e2e` 共 66 个：此前的 65 个，加 P5）。
- [ ] 交错 1 的项目一侧两种顺序通过，离开与 P5a 的移出两种顺序通过，`-count=5 -race`，没有 40P01。
- [ ] `TestEachWriteOnAProjectSharesItsWorkspaceFirst` 对三个写各有一行（含资源行的探测）；少一行时完整性核对失败。
- [ ] 相对规则的表（`TestCheckRoleChange`、`TestCheckRemoval`）和它在组合出的 app 上的情形（`TestTheRelativeRuleOnTheComposedApp`、`TestRemovingAProjectMember`）通过。
- [ ] 规则 1 的项目一侧在组合出的 app 上有正反例（`TestLeavingAProject`）。
- [ ] 三个写的竞争（404 不是 403，一行不改）、改角色和移出按锁下重读到的角色判定、锁的强度在组合一层通过。
- [ ] `project` 的 `apitest.Main` 两个方向核对通过。
- [ ] 本 Phase 的矩阵格子（108 个）通过；共 523 格，耗时记下。
- [ ] `make lint`、`make test`、`make gen-check`、`make knip`、`make test-web`、`make e2e` 通过。

## 5. 不在 P5b 范围内

- 停用、`project.NewCascade`、交错 7、13、14、15、16、19：P6。标签、默认之外的状态：P7。`PROBLEM_MESSAGES` 的搬迁：P8。页面（P5 的页面版本、M1-P4 交接的离开项目的顺序）：P9–P11。M4 的工作项写入是否取工作区的 S：M4（负责人 2026-10-02）。

**留给后面的 Phase**（各 Phase 的 spec 接过去；P5b 的 review 第 6 节照录）：

| Phase | 条目 |
|---|---|
| P6 | 停用结束他的项目成员关系经 P5a 的 `EndMemberships`（规则 2），不经 P5b 的 `EndMember`；P5b 的三个写最先取工作区的 S，停用按 id 顺序取他所在的每个工作区的 N，两者在工作区行上串行（2.12 的论证照样成立），交错 13–15 的写法照 `TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize`（第二方等工作区行，两方都不写 `workspaces`，探针只由那一把锁的等待满足）；`project.sole_admin` 的文字（2.10）要对停用的调用者（服务器管理员的命令）同样成立："给项目另一位管理员，或删除它"，命令的输出照 8.7 写下一步，P6 核对（清扫 30） |
| P7 | 状态、标签的写是项目级的写，经 `Locks`；按资源寻址的（`PATCH /states/{id}`、标签）照 `lockMemberAndDecide` 的写法另加一个分支：不加锁读资源行 → 工作区 S → 项目 N → 锁下重读、确认仍属这个项目 → 判定，`decide` 的 `notFound` 是资源自己的 404；每个写在 `TestEachWriteOnAProjectSharesItsWorkspaceFirst` 有一行，资源路径照 `param()` 加一种，第一步探测资源行；竞争和锁的强度照 `TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile`、`TestEachLockOfAWriteOnAProjectMembershipIsItsStrength` |
| P8 | 三个新码在 `PROBLEM_MESSAGES` 里（Task 4），随它搬迁 |
| P10 | P5 的页面版本：成员页的改角色、移出，"离开项目"先等接口成功再回到项目列表（M1-P4 交接）；页面只给调用者能做的操作（相对规则：不是工作区管理员的项目管理员看不到"设为管理员"，也看不到改另一位管理员的入口），接口的 403、409、422 照 `PROBLEM_MESSAGES` 提示 |
| 后来 | 结束的成员关系的显示设置留着，重新加入之后照旧（与 Plane 相同，第 7 节） |

## 6. 风险

| 风险 | 应对 |
|---|---|
| 同一个工作区里连续重叠的项目级的写让工作区一级的写等到请求期限（方案 E 的代价，17.4）：P5b 加了三个项目级的写 | 三个都是短事务，每个只锁一个项目和至多一行工作区成员；`TestEachLockOfAWriteOnAProjectMembershipIsItsStrength` 钉住它们不持更多 |
| 先锁后判定：任何已认证的账户知道一个成员关系的 id，就能在得到 404 之前短暂取一个工作区的 S、一个项目的 N（P5a review 第 6 节 M4 的同一类） | 与按项目 id 的写相同；S 不挡别的项目级的写，项目 N 只挡同一个项目；限速约束频率（3.13） |
| 相对规则按数值比较、或某一半漏掉（Plane 的写法是 `>=`） | 按 `roleOrder` 的集合（10、25 的情形）；每一半在单元、组合、矩阵各有反例（附录 A 清扫 5、24） |
| 故事看不到的谓词 | 存储测试在两个行序下都看得到；理由逐条在第 3 节第 8 条 |
| plan 的最大 Task 接近上限 | 最大的 Task 4 是 1,195 行，每个都在约 1,500 行以内 |
| 持续集成（ubuntu）上的运行 | 原型和复现都在 macOS 上；合并之后看持续集成，P1–P5a 都是这样合并的 |

## 7. 已知的限制、交接和关闭条件

- **结束的成员关系的显示设置留着**：移出、离开不删除他在项目里的显示设置；重新加入之后照旧（与 Plane 相同）。
- **一个项目可以没有管理员**（第 3 节第 11 条）：同时是工作区管理员的项目成员降唯一的项目管理员、降他自己，工作区一侧的降级；工作区管理员加入就是管理员。
- **被移出、离开的人可以自己回来**：工作区成员可以加入公开项目，原来那一行以 `min(原来的角色, 工作区角色)` 回来（3.5）；私有项目只有工作区管理员能加入。移出不是封禁（与 Plane 相同）。
- **两位项目管理员同时移出对方**：两者在同一个项目的 N 上串行，后到的在锁下重读之后判定，它的调用者已被移出：私有项目答 404 `project.member_not_found`（他看不到项目，与 `TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile` 的"调用者自己的成员关系结束"同一条路），公开项目答 403 `forbidden`（他看得到项目，不再能移出成员）；两种都一行不改。
- **停用的账户仍算另一位管理员**（P5a review 第 7 节同一条）：P6 之前停用不结束他的项目成员关系，`HasOtherAdmin` 把停用账户的有效管理员成员关系算作"另一位有效管理员"，唯一还能登录的管理员可以离开；P6 的停用经 `EndMemberships` 结束它们之后，不再有这种状态。
- **方案 E 的代价**（17.4）：见第 6 节。

| 交接 | P5b 处理的条目 | 留下的条目 |
|---|---|---|
| P5a spec 第 5 节、review 第 6 节的 P5b 一行 | 第 3 节第 2 条 | 无 |
| P4b review 第 6 节中经 P5a 转来的条目 | 第 3 节第 2 条 | 无 |

**M3 设计 13.1 的关闭条件**：13.1 没有落在 P5b 的一项（"M2-closeout §13 P5 改到"是 M2 的 P5 页面，在 P9；"M1-P4 离开项目的顺序"是页面的行为，在 P10）。`docs/v0/M3-workspace-project/handoffs/` 里没有交给 P5b 的条目。P5b 的条件都有落点，没有放不下的。

## 附录 A：原型验证记录（2026-10-03）

原型在 `$M3TMP/p5bproto`（`ebfd2237` 的副本，Go 1.27.1、Node 24、pnpm 11.10.0、Docker；`bin/golangci-lint` 2.13.2）。做法照 P5a：每个 Task 做完时存一份源文件的快照（`$M3TMP/p5bsnap/T1`…`T11`），plan 的代码块由脚本从相邻两份快照的差异生成（`p5btools/mkblocks.py`），生成的文件不进块，按 SHA-256 核对（`gensha.py`）。清扫之后加强的测试由 `propagate.py` 从带来它的 Task 起改进每一份快照和原型（`fix_endmember.py`、`story_ops.py` 是清扫之后的两处），之后重新生成全部块；`assemble.py` 组装 plan 并核对每个块放了一次、每个文件在文件表里、每个 Task 在约 1,500 行以内、没有 HTML 实体。

**修订一轮**（控制者的裁定和预检之后，2026-10-03）：修订之前的快照留在 `$M3TMP/p5bsnap-before-amend`。预检的 M-1、L-3、L-4 由 `p5btools/amend.py` 以逐字的替换改进原型和快照：每个替换在每一份里恰好出现一次，有它的快照从它的 Task 起连续到最后一个。M-1 的组合测试是预检提出的文件（`$M3TMP/p5b-preflight/proposed/project_membership_role_race_test.go`，81 行），从 Task 8 起放进快照。L-4 的生成物在原型上 `make gen`：描述里不再有 `: `，打包器不再给它加引号，快照里的那一行由 `amend_quote.py` 写成同样的样子。之后重新生成全部块和生成物的表（`mkblocks.py`、`gensha.py`）；spec 和 plan 的文字由 `amend_spec.py`、`amend_appx.py` 和逐处的修改改入。下面的数字都是修订之后重跑的。

**逐 Task 复现**（`$M3TMP/p5btools/replay.py`、`replay_all.sh`，日志在 `replay-logs`）：从 `ebfd2237` 的一份新副本开始，照 plan 的顺序应用 11 个 Task 的块并运行每个 Task 写明的命令。每个 Task 之后 `make lint-go` 两段 `0 issues.`、`make test` 42 个 `ok`；Task 1、4、6、7 的 `make gen`/`make gen-go` 之后生成物与快照没有差异，SHA-256 和行数与 plan 的表相同；Task 4、6、7、11 的 `make lint-web`（关键词守卫 5 个命中都有例外）、`make knip`、`make test-web` 通过；Task 8、9、10 的 `-race` 运行（Task 8 含预检之后的新测试，Task 9 是 `-count=5`）通过；Task 11 的 `make e2e` 66 个故事中 65 个通过，失败的只是 S3 的 F4（副本不是 git 仓库，构建没有提交号；`replay.py` 只接受这一个失败，其余任何失败都算复现失败）。最后的树与修订之后的原型逐个文件相同（`treediff.mjs`：3,046 个文件，0 处差异）；`planapply.mjs` 从基线核对 plan 的 201 个块（179 处替换、22 个新文件）都放得上。共 908 秒，其中两次 Docker 没有起来测试容器（Task 1、3 的 `make test`；同一台机器上别的项目在跑测试容器），`replay.py` 等过之后重跑那一条命令，重跑通过；第一次修订之后的复现在 Task 8 的 `make test` 因同一原因停下（`replay-logs-amend-docker`），之后加了这条重跑，从头再复现一次。

**最终的原型**（`gates.sh`）：修订之后由复现的最后一份代替（它与原型逐个文件相同）：`make gen` 之后生成物没有差异；`make lint-go` 两段 `0 issues.`；`make test` 42 个 `ok`；`make lint-web`（关键词守卫 5 个命中都有例外，54 个任务）；`make knip`；`make test-web`（16 个任务）；`make e2e` 66 个故事中 65 个通过，S3 因原型不是 git 仓库、构建没有提交号而失败（P4b 的 F4；它读 `commit`，与 P5b 无关）。

**矩阵**：`TestPermissionMatrix -v` 523 格（P5a 结束时 415 格，P5b 加 108 格：改角色四行 48 格、移出四行 48 格、离开一行 12 格），1.47 秒（修订之前 1.51 秒，P5a 结束时 1.39 秒；修订不改矩阵），全部通过。

**交错和竞争**（Task 8、9 的全部测试，含修订加的一个）：`go test -count=5 -race -v -run 'TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile$|TestAWriteOnAProjectMembershipDecidesOnWhatItReadUnderItsLocks$|TestEachLockOfAWriteOnAProjectMembershipIsItsStrength$|TestTheWritesOnAProjectRunOnTheirTransactionsConnection$|TestTheWritesOnAProjectStampTheirRequest$|TestTwoProjectAdminsLeavingLeaveAnAdmin$|TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize$' ./internal/bootstrap/`（修订之后的原型上）：`ok`，69.4 秒，7 个测试各 5 次全部通过，160 个子测试（15 个竞争情形、4 个锁下重读的情形、3 个锁的强度、10 个交错子测试，各 5 次）；输出里没有 40P01，没有数据竞争。修订之前同样的四个测试是 140 个子测试、60.4 秒。

**三十类清扫**（brief 的缺陷类别；每个变异一个 `go test -overlay` 或 `go build -overlay`，树不动；`mutlevels.py` 在变异写的每一层各跑一次：单元（`project`、`access`、`workspace` 的单元包）、存储（`project/adapter/postgres`）、组合（`bootstrap`）、端到端（单独运行的故事 P5，`s15-*` 另跑 W7）；"层"是它被发现的每一层）：

| 清扫 | 大小 | 结果 | 层 |
|---|---|---|---|
| 1 每个 SQL 谓词 | 存储一半：P5b 的 4 条新查询的 14 个谓词（`MemberByID` 2、`UpdateMemberRole` 3、`EndMember` 4、`HasOtherAdmin` 5，其中"别的成员"是清扫 20 的 `s20-hoa-actor`），每个去掉，参数照旧绑定；加 P4b 交来的 `Memberships` 的账户（`x-memberships-account`）。故事一半：这 15 个和清扫 20 的两个集合形式，每个只运行 P5 | 存储一半 15/15，每个存储测试都有只由那个谓词决定的行，已删除的行在有效的之前、之后两种行序下各一次。组合一层 10/15，另 5 个（`UpdateMemberRole`、`EndMember` 的 `is_active` 和 `deleted_at`，`HasOtherAdmin` 的 `deleted_at`）在组合一层等价。故事一半 9/17：P5 发现每条查询的 id、项目、成员、角色和两个集合形式；看不到的 8 个（四个 `deleted_at`、三个 `is_active`、`Memberships` 的账户）理由逐条在第 3 节第 8 条，每个都由存储测试发现 | 存储；组合；端到端 |
| 2 端口调用的错误 | 三个用例的端口调用（读成员关系、锁、锁下重读、改、结束、问另一位管理员）：失败被吞掉、换成 404、重试一次、离开的拒绝被吞掉照样结束；8 个 | 8/8；每个单元测试比较到失败那一步为止的整个调用记录 | 单元；组合（`s2-leave-refusal-swallowed`） |
| 3 写不动的行 | 2 个写（`UpdateMemberRole`、`EndMember`），存储测试有另一个项目、另一个成员、另一个工作区，状态各不相同，核对每行每列；5 个变异（结束也写角色、改写 `created_at`、两个写保留原来的写者、结束保留原来的时刻） | 5/5。清扫中把 `TestEndMember` 被结束的行改成访客的：结束写成员的角色原来在存储一层看不出；预检 L-3 又发现访客的正是列的默认值（`DEFAULT 5`），写 `role = DEFAULT` 的结束照样通过，修订之后它随行序是访客的或管理员的（清扫 26） | 存储；组合；端到端（写角色、写者的 3 个） |
| 4 组合根的接线 | `project.New` 的三个用例：不开事务、放行一切的 `Authorizer`、不结束任何行、停在 2001 年的时钟、`POST /leave` 交给移出；8 个 | 8/8 | 组合；`s4-leave-served-by-removal` 另在单元（HTTP）和 P5 |
| 5 安全性质在真实环境上 | 规则表的三行（改角色、移出给项目成员；离开不给项目访客）；规则 1 和相对规则的每一半在第 24 条 | 3/3，每个都在组合出的 app 上被发现（矩阵）；故事一半 1/3，另两个故事看不到（第 3 节第 8 条） | 单元；组合；端到端（1 个） |
| 6 每句文档 | P5b 加或改的注释 247 块（59 个文件，约 700 句）；契约三个操作的描述和摘要、新的参数和字段（13 处）；四个码的文字；故事 P5 的标题；一个变异（"时刻在锁之后读"的离开） | 清扫中改了 2 处：`TestEndMember` 的说明（"角色不变"在访客的行上）、故事 P5 的标题（加"他在工作区另一个项目的成员关系不动"，由 Ops 一步展示）；`project.sole_admin` 的改写见第 3 节第 7 条。预检又发现两处：`removeProjectMember` 的描述给 `project.own_membership` 补救（L-4，已删去），spec 第 7 节说互相移出的后到者答 404，公开项目上是 403（L-2，已改）。其余每句都有代码或测试持住；`s6-leave-before-lock-clock` 1/1 | 单元；组合 |
| 7 反例里没有随机 | P5b 的全部测试 | 决定变异是否被发现的输入都固定：存储测试的行序由插入顺序决定，两种顺序各跑一次；单元假对象的回答固定；`time.Now` 只用来框住请求的时刻；P5b 的测试不用 `math/rand` | — |
| 8 决定所依赖的读在调用者的事务里 | 四个存储方法各改成走池（锁和 `ShareMembers` 是 P4b、P5a 的，照旧由它们的连接测试）；4 个 | 4/4 | 组合（`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`：池只有一个连接，走池的读写等到期限） |
| 9 锁顺序在真实环境上 | 工作区 → 目标在工作区的成员关系 → 项目 → 锁下重读：项目在工作区之前、成员关系的第一次读带 `FOR UPDATE`、目标在项目之后、重读挪到锁之前；4 个 | 4/4 | 组合（前一个锁被别的事务持有时，后一个还没有取，NOWAIT）；单元（调用记录，3 个） |
| 10 承重的种子行有前提 | 矩阵种子 3 行（PM 在公开项目是成员、P-前的成员关系已结束、WG- 的已结束）；组合夹具 `memberWorld` 2 行（gina 是 acme 的管理员、erin 是 acme 的访客）；每行一个变异 | 5/5（`TestPermissionMatrix/prepare`；`newMemberWorld` 的 `standing`，每个用它的测试） | 组合 |
| 11 每条拒绝路径 | 没有调用者时什么都不读（单元的调用记录为空）；判定的失败原样返回；已结束的不在判定之前答 404；2 个变异 | 2/2 | 单元；组合 |
| 12 与结束、删除的竞争（组合） | `TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile` 15 个情形（目标的成员关系结束、删除、移到 Ops，调用者的结束，项目、工作区删除），`TestAWriteOnAProjectMembershipDecidesOnWhatItReadUnderItsLocks` 4 个（目标等锁时被提拔、被删除），探针证明在等锁；6 个变异（在锁之前判定、离开在锁之前判定、不在锁下重读、重读之后不核对项目、等锁时删除的工作区不算、重读挪到判定之后） | 6/6；重读挪到判定之后（预检 L-6）原来只在单元一层，修订之后由新的组合测试发现（删除的成员关系对无权写的调用者答 403 而不是 404） | 组合（5 个）；单元（`s12-workspace-gone-ignored` 在组合一层等价，第 3 节第 8 条） |
| 13 锁强度在组合一层看得到 | 写成员关系时项目锁成 `FOR SHARE`、离开时项目锁成 `FOR SHARE`、改角色不锁目标在工作区的成员关系、写不取工作区的 S；4 个 | 4/4 | 组合（`TestEachLockOfAWriteOnAProjectMembershipIsItsStrength` 的 NOWAIT 阶梯，交错的探针）；单元 |
| 14 每个"由谁"可以失败 | 存储测试里被写的行之前由另一个账户写；组合的三个写和 `TestTheWritesOnAProjectStampTheirRequest` 的三行之前由 bob 写；P5 里 pam 改、移出的行之前由管理员写；2 个变异（写成成员自己的；保留写者的两个在清扫 3） | 2/2 | 单元；组合；端到端 |
| 15 标题的每个说法都有展示 | P5 的标题 12 个分句；P5b 的 Go 测试名和说明 | 每个分句由故事自己的一步展示；"他在另一个项目的成员关系不动"是清扫中加的 Ops 一步；`s15-demote-skips-ended`（降为访客跳过已结束的，W7 的一半）1/1 | 存储；组合；端到端（P5、W7） |
| 16 单元假对象的回答顺序 | P5b 的四个假方法（`MemberByID`、`UpdateMemberRole`、`EndMember`、`HasOtherAdmin`）都只回答一个值；改角色的 `ShareMembers` 只问一个账户 | 没有回答列表的新假对象，不适用 | — |
| 17 判定之前的代价有界 | 三个操作的输入：路径里的一个 id（改角色另有一个角色）；判定之前只读一行成员关系、锁一行工作区、至多一行目标在工作区的成员关系、一行项目，再读一行成员关系 | 没有无界的输入，不需要上限 | — |
| 18 契约描述只说代码做的 | 三个操作的描述（2.5、2.6、2.9、2.10 的顺序）；3 个变异（少声明 `project.sole_admin`、少声明 `project.role_too_high`、多声明 `project.archived`） | 3/3；每句都有按所说的顺序和结果的测试：矩阵（谁可以、谁 404、谁 403）、`TestTheRelativeRuleOnTheComposedApp`、`TestRemovingAProjectMember`、`TestLeavingAProject`、`TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile`（锁之后判定） | 单元（`CheckResponse`、`apitest.Main`）；组合（2 个） |
| 19 每个新的存储方法有失败测试 | 4 个新方法在 `failures_test.go` 里；5 个变异（失败被吞掉或读成"没有" 4、`EndMember` 不核对行数） | 5/5 | 存储（`s19-em-count-unchecked` 在组合一层等价：用例在锁下读到有效才结束） |
| 20 相关谓词的集合形式、"别的"算进自己 | `EndMember`、`HasOtherAdmin` 的项目改成"这个工作区的任何项目"；`HasOtherAdmin` 把他自己算进去；3 个 | 3/3 | 存储；组合；端到端（Ops 的 tom、wanda） |
| 21 拒绝和失败钉住第一个 `*shared.Error` | 单元的 `outcome.check`；2 个变异（改的失败包进 404、离开的两个拒绝 `errors.Join`） | 2/2 | 单元；组合（离开） |
| 22 端口的回答对得上所问 | `MemberByID` 回答别的 id、`UpdateMemberRole` 回答别的成员关系；2 个 | 2/2 | 单元（组合一层的真实存储不答错 id，第 3 节第 8 条） |
| 23 组合的夹具跨第二个工作区、第二个项目 | `memberWorld`：acme、beta 两个工作区，Web、Ops、Lab 三个项目，bob、carol 在一处是管理员、在另一处是成员；矩阵照 P4b 的 acme、other；故事 P5 的 Web、Ops | 清扫 1、20 的跨范围变异（项目、工作区的项目）都在组合一层被发现（`s1-em-project`、`s1-hoa-project`、`s20-*`） | 组合 |
| 24 每条规则的每一半在组合一层有反例 | 相对规则（不约束、工作区管理员的例外、`From`、`To`、自己的、访客的上限和它读的角色）、移出（高于自己的、只许低于自己的、自己的、以管理员来判、看不到自己的、已结束的）、改角色（看不到自己的、已结束的）、目标的角色取锁之前的读（预检 M-1）、规则 1（不问、每个人都问、看 `ProjectAdmin`、只有他一人）；20 个 | 20/20，每个都在组合出的 app 上被发现（目标的角色原来哪一层都没有发现：没有一个竞争改目标的角色，单元的假实现也答不出别的角色；修订之后由 Task 3、6 的拒绝表各一行和新的组合测试发现），`s24-hoa-alone-allowed` 另在存储一层；故事一半 4/8，另 4 个故事看不到（第 3 节第 8 条） | 单元；存储（1 个）；组合；端到端（4 个） |
| 25 判定和执行在同一把锁下 | 离开先在一个事务里问另一位管理员，再在另一个事务里结束；1 个 | 1/1 | 组合（`TestTwoProjectAdminsLeavingLeaveAnAdmin` 问过之后的 gate：两位都离开）；单元 |
| 26 "不变"用一个变异不会碰巧得出的值 | 改角色写 20（目标是成员、改成访客）；结束写成员的角色（见清扫 3）；结束写列的默认（访客的）、15、20（预检 L-3，`TestEndMember` 被结束的行随行序是访客的或管理员的）；4 个加清扫 3 的 1 个 | 5/5 | 存储；组合；端到端 |
| 27 故事里的"不动"、"留着"在动作之前读、之后核对 | P5 的 4 处：被移出的 gus 的行和显示设置、tom 在 Ops 的成员关系、被拒绝的添加和改角色、ray 回来的那一行 | 每处之前、之后各一次 `expectMembers`（Ops 的两次读是清扫中加的） | 端到端 |
| 28 只在端到端被发现的性质另有 Go 的测试 | 全部 113 个变异 | 没有一个只在端到端被发现 | — |
| 29 每个等待都有期限 | P5b 新的组合测试里 22 处等待（`soon(t)`、`receiveWithin`、`WaitForLockWaitOn` 的 5 秒）；池照 `openPool` | 每处都有期限 | — |
| 30 每个输出、问题的细节和补救对每个收到它的调用者都成立 | 四个码（`project.member_not_found`、`project.own_membership`、`project.role_too_high`、`project.sole_admin`）的文字和补救；三个操作的描述 | `project.sole_admin` 的补救改写，对三种调用者都成立；`project.own_membership` 不给补救；`project.role_too_high` 的"unless you are a workspace admin"只在改角色一半（第 3 节第 7 条、2.6、2.10） | — |
| 按集合（P5b 自己的） | 相对规则、移出的规则、离开的规则 1 按数值比较；3 个 | 2/3；`set-leave-magnitude` 等价（它比较的只是存储的角色，`CHECK (role IN (5, 15, 20))`）；另两个只在单元一层，理由在第 3 节第 8 条 | 单元 |
| 完整性核对、探针（P5b 自己的） | `projectWrites` 的三行各去掉一行（`pf-row-*`）；写不取工作区的 S（`p-no-workspace-share`，交错的探针的反例，记在清扫 13） | 3/3 | 组合 |

113 个变异，112 个被发现，1 个等价（`set-leave-magnitude`）。只在单元一层被发现的是清扫 2、21、22 的错误传递（组合一层无从注入）、`s12-workspace-gone-ignored`、按数值比较的两个（组合一层等价）和契约多声明的码（`apitest.Main` 本来在单元一层），理由逐条在第 3 节第 8 条；没有只在单元一层被发现、在组合一层可以展示的安全性质或锁的性质。

**变异表（按缺陷类别）**：

变异的定义在 `$M3TMP/p5btools/mutants_p5b.py`（第一套，81 个）、`mutants_p5b_more.py`（第二套，42 个：新的 23 个，第一套没有构建起来的 3 个和改错了文件的 1 个的重做，故事一半的 14 个重跑，`TestEndMember` 改成访客之后存储一层的 1 个重跑）、`mutants_p5b_extra.py`（第三套，4 个）；预检之后的 `mutants_p5b_amend.py`（92 个：预检的 5 个新变异，和此前在单元或存储一层被发现的 87 个在那些层重跑，因为修订改了单元的假实现、改角色和移出的拒绝表、`TestEndMember`）；结果在 `p5btools/logs`（`sweeps-*.out`、`mutants_p5b*-all.json`、每个变异的 `mutant-*.log`），`merge.py` 把三套按变异合成一行（`logs/merged.json`）。第一套里"活下来"的 4 个都不是真的：3 个没有构建起来（变异让一个变量不再被使用），1 个改的是打包的契约，而 `apitest.Main` 读模块自己的文件；重做之后都被发现。

| 缺陷类别 | 变异 | 结果 | 失败的测试 | 层 |
|---|---|---|---|---|
| 清扫 1：谓词（存储） | `s1-mbi-{id,deleted}`、`s1-umr-{id,active,deleted}`、`s1-em-{project,member,active,deleted}`、`s1-hoa-{project,role,active,deleted}`、`x-memberships-account` | 14/14 | `TestMemberByID`、`TestUpdateMemberRole`、`TestEndMember`、`TestHasOtherAdmin`、`TestMemberships` | 存储 |
| 清扫 1：谓词（组合） | 同上 | 9/14 | `TestPermissionMatrix`、`TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile`、`TestLeavingAProject`、`TestRemovingAProjectMember`、`TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize`、`TestEachLockOfAWriteOnAProjectMembershipIsItsStrength`；另 5 个组合一层等价（第 3 节第 8 条） | 组合 |
| 清扫 1、20：谓词（故事，单独运行） | 上面 14 个和 `s20-*` 3 | 9/17 | P5；看不到的 8 个见第 3 节第 8 条 | 端到端 |
| 清扫 2：端口的错误 | `s2-{upd-write,rem-end,leave-other,leave-refusal}-swallowed`、`s2-leave-end-retried`、`s2-reread-retried`、`s2-member-read-as-404`、`s2-lock-failure-as-404` | 8/8 | `Test{Update,Remove}ProjectMember{Refuses,ReturnsEachFailure}`、`TestLeaveProject{Refuses,ReturnsEachFailure}`；`s2-leave-refusal-swallowed` 另由 `TestPermissionMatrix`、`TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile`、`TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize` | 单元；组合（1 个） |
| 清扫 3、26：写不动的行、不会碰巧的值 | `s3-em-writes-role`、`s3-umr-touches-created`、`s3-{umr,em}-keeps-writer`、`s3-em-keeps-time`、`s26-umr-writes-admin`、`pf-em-role-{default,15,20}`（预检 L-3） | 9/9 | `TestEndMember`、`TestUpdateMemberRole`；`TestARestoredMembershipGivesNoMoreThanItHad`、`TestTheRelativeRuleOnTheComposedApp`、`TestRemovingAProjectMember`、`TestLeavingAProject`、`TestTheWritesOnAProjectStampTheirRequest`；P5（写角色、写者的 4 个） | 存储；组合；端到端 |
| 清扫 4：接线 | `s4-{update,leave}-no-tx`、`s4-update-allow-all`、`s4-{remove,leave}-ends-nothing`、`s4-remove-frozen-clock`、`s4-update-old-clock`、`s4-leave-served-by-removal` | 8/8 | `TestEachLockOfAWriteOnAProjectMembershipIsItsStrength`、`TestEachWriteOnAProjectSharesItsWorkspaceFirst`、`TestRemovingAProjectMember`、`TestLeavingAProject`、`TestTheWritesOnAProjectStampTheirRequest`、`TestPermissionMatrix`、`TestTheRelativeRuleOnTheComposedApp`；`TestLeaveProject`（HTTP）和 P5（`/leave`） | 组合；单元、端到端（1 个） |
| 清扫 5：规则表 | `s5-rule-{update,remove}-member`、`s5-rule-leave-no-guest` | 3/3 | `TestEveryRuleDecidesItsCells`；`TestPermissionMatrix`、`TestTheRelativeRuleOnTheComposedApp`、`TestRemovingAProjectMember`、`TestLeavingAProject`；P5（改角色的一行） | 单元；组合；端到端（1 个） |
| 清扫 6：说明里的"锁之后读时刻" | `s6-leave-before-lock-clock` | 1/1 | `TestEachWriteReadsTheClockUnderItsLock`；`TestEachLockOfAWriteOnAProjectMembershipIsItsStrength` | 单元；组合 |
| 清扫 8：事务的连接 | `s8-{mbi,umr,em,hoa}-pool` | 4/4 | `TestTheWritesOnAProjectRunOnTheirTransactionsConnection` | 组合 |
| 清扫 9：锁顺序 | `s9-project-before-workspace`、`s9-member-row-first`、`s9-targets-after-project`、`s9-reread-before-locks` | 4/4 | `TestEachLockOfAWriteOnAProjectMembershipIsItsStrength`、`TestEachWriteOnAProjectSharesItsWorkspaceFirst`、`TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile`；单元的调用记录 | 单元（3 个）；组合 |
| 清扫 10：种子和夹具的行 | `s10-{pm-an-admin,before-active,wg-active}`、`w-world-{gina,erin}-member` | 5/5 | `TestPermissionMatrix/prepare`；`newMemberWorld` 的 `standing` | 组合 |
| 清扫 11：拒绝路径 | `s11-decision-as-404`、`s11-ended-before-decision` | 2/2 | `Test{Update,Remove}ProjectMemberRefuses`、`…ReturnsEachFailure`；`TestPermissionMatrix` | 单元；组合 |
| 清扫 12：竞争 | `s12-{decide-before-locks,leave-decide-before-locks,no-reread,project-unconfirmed,workspace-gone-ignored}`、`pf-decide-before-reread`（预检 L-6） | 6/6 | `TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile`、`TestAWriteOnAProjectMembershipDecidesOnWhatItReadUnderItsLocks`（`pf-decide-before-reread`）、`TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize`；单元的 `…Refuses` | 单元；组合（5 个） |
| 清扫 13：锁强度 | `s13-{member-path-share,leave-share,no-target-share}`、`p-no-workspace-share` | 4/4 | `TestEachLockOfAWriteOnAProjectMembershipIsItsStrength`；`TestTwoProjectAdminsLeavingLeaveAnAdmin`（`s13-leave-share`）；交错和竞争的探针（`p-no-workspace-share`） | 单元；组合 |
| 清扫 14：由谁 | `s14-{upd,rem}-by-member` | 2/2 | `Test{Update,Remove}ProjectMember`；`TestTheRelativeRuleOnTheComposedApp`、`TestRemovingAProjectMember`、`TestTheWritesOnAProjectStampTheirRequest`；P5 | 单元；组合；端到端 |
| 清扫 15：标题 | `s15-demote-skips-ended` | 1/1 | `TestDemotingAMemberToGuest`；`TestAcceptingAsAGuestAgainDemotesInTheWorkspacesProjects`、`TestADemotionAndTheProjectSidesGrowthSerialize`；P5、W7 | 存储；组合；端到端 |
| 清扫 18：契约 | `s18-leave-code-undeclared`、`s18-update-code-undeclared`、`s18-remove-code-unused-module` | 3/3 | `TestLeaveProject`、`TestUpdateProjectMemberRefusals`（`CheckResponse`）、`apitest.Main`；`TestLeavingAProject`、`TestTheRelativeRuleOnTheComposedApp`、`TestPermissionMatrix` | 单元；组合（2 个） |
| 清扫 19：存储方法的失败 | `s19-{mbi,umr,em,hoa}-err-swallowed`、`s19-em-count-unchecked` | 5/5 | `TestAFailedReadIsAnErrorNotAnAnswer`、`TestAFailedWriteIsAnError`、`TestEndMember`、`TestUpdateMemberRole` | 存储 |
| 清扫 20：集合形式、算进自己 | `s20-hoa-actor`、`s20-{hoa,em}-workspace` | 3/3 | `TestHasOtherAdmin`、`TestEndMember`；`TestLeavingAProject`、`TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize`、`TestTwoProjectAdminsLeavingLeaveAnAdmin`；P5 | 存储；组合；端到端 |
| 清扫 21：第一个 `*shared.Error` | `s21-upd-failure-as-404`、`s21-leave-two-refusals` | 2/2 | `TestUpdateProjectMemberReturnsEachFailure`、`TestLeaveProjectRefuses`；`TestLeavingAProject`、`TestPermissionMatrix` | 单元；组合（1 个） |
| 清扫 22：回答对得上所问 | `s22-member-id-unchecked`、`s22-update-id-unchecked` | 2/2 | `Test{Update,Remove}ProjectMemberRefuses`、`TestUpdateProjectMemberReturnsEachFailure` | 单元 |
| 清扫 24：每一半 | `s24-*` 19（相对规则 7、移出 6、改角色 2、规则 1 4）；`pf-stale-target-role`（预检 M-1，目标的角色取锁之前的读） | 20/20 | `TestCheckRoleChange`、`TestCheckRemoval`、三个用例的单元测试；`TestPermissionMatrix`、`TestTheRelativeRuleOnTheComposedApp`、`TestRemovingAProjectMember`、`TestLeavingAProject`、`TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile`、`TestAWriteOnAProjectMembershipDecidesOnWhatItReadUnderItsLocks`（`pf-stale-target-role`）；`TestHasOtherAdmin`；P5（4 个） | 单元；存储（1 个）；组合（每个）；端到端（4 个） |
| 清扫 25：判定和执行在一把锁下 | `s25-leave-check-then-end` | 1/1 | `TestTwoProjectAdminsLeavingLeaveAnAdmin`、`TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize`、`TestEachLockOfAWriteOnAProjectMembershipIsItsStrength`；`TestLeaveProject` | 单元；组合 |
| 按集合 | `set-{change,removal,leave}-magnitude` | 2/3（`set-leave-magnitude` 等价） | `TestCheckRoleChange`、`TestCheckRemoval` | 单元 |
| 完整性核对 | `pf-row-{updateProjectMember,removeProjectMember,leaveProject}` | 3/3 | `TestEachWriteOnAProjectSharesItsWorkspaceFirst` | 组合 |

P5b 自己的条目在上表中的位置：取锁路径的顺序和强度是清扫 9、13，资源行的探测和完整性核对的第二种形状是 `s9-member-row-first`、`pf-row-*`，每个竞争答 404 不答 403 是清扫 12、`s24-{upd,rem}-ended-taken`、`s2-leave-refusal-swallowed`；相对规则按集合是 `set-*`，每一半是 `s24-{no-relative-rule,no-wa-exception,from-unchecked,to-unchecked,own-unchecked,guest-cap-gone,guest-cap-callers-built}`，它用的是锁下重读到的目标角色是 `pf-stale-target-role`，判定在重读之后是 `pf-decide-before-reread`；移出不删除、一个时刻、由调用者是 `s3-em-*`、`s4-remove-*`、`s14-rem-by-member`，重新加入回到 15 是 `TestRemovingAProjectMember` 的最后一步和 P5 里 ray 的重新加入；离开的规则 1 看项目角色是 `s24-leave-rule1-workspace-admin`，唯一管理员是 `s24-leave-no-rule1-built`、`s24-hoa-alone-allowed`、`s1-hoa-*`、`s20-hoa-*`，判定和结束在一把锁下是 `s25-*`、`s13-leave-share`；W7 的降级是 `s15-*`。

P1–P5a 交来的类别：在父行的锁之前判定是 `s12-*-decide-before-locks`；锁在事务之外是 `s4-*-no-tx`、`s8-*`；规则表的行放宽是 `s5-*`；"其余的行不变"不在空集上是清扫 3 的另一个项目、成员、工作区；`errors.Is` 对准确切的问题是清扫 21；按资源寻址的项目级的写被完整性核对看到是 `pf-row-*`；一行一个账户看不到少了的 `WHERE` 是清扫 1；P5b 没有新的表和索引；挂住的测试都有期限（清扫 29）；`project.sole_admin` 在工作区的两个操作上仍由 `workspace` 的 `apitest.Main` 两个方向核对。
