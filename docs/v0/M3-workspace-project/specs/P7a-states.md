# M3/P7a 状态与按资源寻址的共用取锁路径：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M3/P7a `states` |
| 日期 | 2026-10-05 |
| 状态 | 草稿，待预检。拆分和第二步照负责人 2026-10-05 的裁定与控制者的裁定 S1–S8（`p7-split-rulings.md`）；第 3 节第 1、4、5 条落实裁定 S6、S7；第 3 节其余各条是本 spec 的决定，待控制者确认 |
| 上级文档 | [M3 设计文档](../M3-design.md) 第 2（P6、W11）、3.4、3.6（加锁表，约定一、二、三、五及其例外）、3.12、3.17、3.19、3.20（P7a 三行）、4.11、5.1–5.3、6.4、6.7、9.2、9.3（交错 10）、9.4、12（P7a 与约束 1–4）、13.1 节 |
| 前置交接 | [P5b spec](P5b-project-memberships.md) 第 5 节 P7 一行、[P5b review](../reviews/P5b-project-memberships-review.md) 第 6 节（共用路径、工作区锁的键）；[P4b spec](P4b-project-members.md) 第 5 节、[P4b review](../reviews/P4b-project-members-review.md) 第 6 节（每个写经 `Locks`、最先锁工作区的一行）；[P6 spec](P6-deactivation.md) 第 5 节 P7 一行、[P6 review](../reviews/P6-deactivation-review.md) 第 6 节（与停用没有新的交错，清扫 41）；[P4a review](../reviews/P4a-projects-review.md) 第 6 节（标签的两条，P7b）；[P2 review](../reviews/P2-workspaces-review.md) 第 6 节（`pgtest` 的探测有期限）。落点见第 3 节第 12 条 |
| 计划 | [P7a plan](../plans/P7a-states.md) |

本 spec 只写 M3 设计交给 P7a 决定的东西：名字、签名、SQL、测试名，以及原型证明了什么。规则本身以 M3 设计为准，这里引用节号，不重述。P7a 依赖 P6（`46abb231` 的 `main`，设计随 `3f87fcd3` 拆出 P7a、P7b）。

P7a 是评审敏感的一段（M3 设计 12 节约束 3）：按资源寻址的共用取锁路径从 P5b 的项目成员的写里提出来，它的第一批新用户是带守卫的写。删除默认状态由语句的守卫拒绝，设为默认的第二条语句写 0 行时让事务失败，一组最后一个状态在并发下由项目行的 `FOR NO KEY UPDATE` 保证，等锁期间被删除、移走或看不到的状态答它自己的 404。附录 A 的五十类清扫逐个核对每个这样的测试在它所说的性质去掉之后失败，并记下失败在哪一层（单元、存储、组合、端到端）；只在单元一层失败的安全或加锁的性质算缺口，按性质只能在单元一层的逐条写在第 3 节和附录 A。

P7a 的原型从 P7 的架构子任务第一次运行的 T1–T7（`$M3TMP/p7snap`）接过来，照裁定 S4 重新切成十个 Task，逐条复审、清扫之后改在根上；改了什么、为什么，见附录 A 的"对第一次运行的复审"。

## 1. 目标

按 M3 设计 12 节 P7a 和 3.17：状态的全部操作和 3.17 的规则，按资源寻址的共用取锁路径。具体是：

- 共用路径：`project/app/lock.go` 的 `placed`、`rowWrite[R]`、`lockRowAndDecide[R]`、`rowWrite.read`；P5b 按行寻址的两个写（改角色、移出）改走它，`lockMemberAndDecide` 删除；离开按项目寻址，照旧经 `lockAndDecide`（第 3 节第 2 条）。`Locks.lock` 核对 `ShareWorkspaceByID` 回答的工作区的 id（P5b review 第 6 节）。路径上的三个核对测一次（裁定 S6）；
- 状态的规则（`project/domain/state.go`）：组和分诊（`group = triage` 422 `not_allowed`）、名称和颜色、`SequenceAfter`（非分诊状态的最大值加 15000，没有时 65535）、`CheckGroupKept`；四个状态码；
- 状态的存储：十条语句（`queries/states.sql`），每条的存储测试每个谓词各由一行决定，每个方法有失败测试；
- 六个操作：`listStates`（已归档的项目为空）、`createState`、`updateState`、`deleteState`（守卫的删除，删了 0 行时在锁下重读答 409 `project.state_default` 或 404）、`markDefaultState`（两条语句，第二条写 0 行时 404、事务回滚第一条）、`listWorkspaceStates`（他是有效成员的未归档项目，不含分诊状态）；分诊状态对状态的操作不可见（列表不含它，按 id 操作 404 `project.state_not_found`）；
- 每个状态的写：工作区 `FOR SHARE` → 项目 `FOR NO KEY UPDATE` → 时钟 → 它的行；锁下重读确认仍属那个项目，否则答 `project.state_not_found`；最先锁工作区的测试里一行，第一步探测它的状态行；
- 矩阵：每个项目种下状态（`matrixStates`），六个操作 19 行，已归档项目的小表随每个操作；
- 并发：交错 10 和一组最后两个状态的删除、改组，两种顺序；两个创建；P5b 的竞争和锁强度推广到状态的行；每个状态的写只写它的行；`-count=5 -race`，没有 40P01；
- 故事 P6、W11 的接口版本；`plane-diff.md` 4.11 中 P7a 的三行（3.20）。

不加迁移、表、Go 模块、npm 包；跨模块的端口不变。

## 2. 交付物

### 2.1 文件总览

路径相对于仓库根目录（`server/internal/` 省略）。"Task"是 plan 中负责它的任务（plan 有完整的文件表）。本 Phase 77 个手写的文件（新建 27 个、修改 49 个、删除 1 个）、5 个生成物；没有迁移，没有新表，跨模块的端口不变。

| 路径 | 内容 | Task |
|---|---|---|
| `project/app/lock.go`、`lock_test.go`、`member_ports.go`、`ports.go`、`update_member.go`、`remove_member.go` 及两者的测试、`fakes_write_test.go` | 共用路径；P5b 按行寻址的两个写改走它；路径的三个核对测一次 | 1 |
| `project/domain/state.go`、`state_test.go`、`errors.go` | 状态的规则，四个状态码 | 2 |
| `project/adapter/postgres/queries/states.sql`、`states.go`、`rows.go`、`states_test.go`、`state_writes_test.go`、`failures_test.go`、`store_test.go`、`membership_test.go`、`update_test.go` | 十条语句、存储方法、存储测试、失败测试；测试的共用移到 `store_test.go` | 3 |
| `api/modules/project.yaml`、`api/openapi.yaml`；`access/domain/rules.go`、`rules_test.go`；`project/domain/actions.go`；`project/app/state_ports.go`、`create_state.go` 及测试、`fakes_state_test.go`、`clock_test.go`；`project/adapter/http/handler.go`、`states.go` 及测试；`project/module.go`；前端的三个文案文件；`bootstrap/permission_matrix_*_test.go`、`project_write_locks_test.go`、`project_writes_test.go`、`project_connection_test.go` | `createState`；矩阵的状态种子和行 | 4 |
| `project/app/update_state.go` 及测试、`state_ports.go`、`fakes_state_test.go`、`fakes_write_test.go`、`fakes_member_test.go`、`clock_test.go`；`rules.go`、`actions.go` | `updateState` 的用例 | 5 |
| 契约；`project/adapter/http/*`；`module.go`；文案；`bootstrap/permission_matrix_{seeded,columns,targets,states}_test.go`、`project_write_locks_test.go`、`project_writes_test.go`、`project_connection_test.go` | `updateState` 的接口和组合；以状态行为目标的矩阵列；最先锁工作区的测试推广到按行寻址的写 | 6 |
| 契约；`rules.go`、`actions.go`、`state_ports.go`、`delete_state.go`、`mark_default_state.go` 及测试；HTTP；`module.go`；文案；矩阵、最先锁工作区、盖戳、连接 | `deleteState`、`markDefaultState` | 7 |
| 契约；`rules.go`、`actions.go`、`state_ports.go`、`lock.go`、`list_projects.go`、`check_identifier.go`、`list_states.go`、`list_workspace_states.go` 及测试；HTTP；`module.go`；`bootstrap/permission_matrix_states_test.go`、`project_visibility_test.go` | 两个读；`findWorkspaceAndDecide` | 8 |
| `bootstrap/project_row_races_test.go`（由 `project_membership_races_test.go` 改名）、`interleaving_states_test.go`、`state_rows_test.go`、`membership_world_test.go`、`interleaving_leaving_test.go` | 并发 | 9 |
| `e2e/stories/project/p6-states.spec.ts`、`e2e/stories/workspace/w11-guest-bounds.spec.ts`、`e2e/fixtures/api.ts`、`assert/project.ts`、`assert/workspace.ts`、`e2e/stories/project/p5-project-members.spec.ts`；`project/app/deletion.go`；`docs/v0/plane-diff.md` | 故事 P6、W11；文档 | 10 |

生成物：`project/adapter/postgres/gen/states.sql.go`（Task 3）；`api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`、`project/adapter/http/gen/server.gen.go`（Task 4、6、7、8）、`project/adapter/http/gen/bodyshape.gen.go`（Task 4、6）。

### 2.2 依赖

不加 Go 模块、npm 包、迁移、表。状态的表、三个部分唯一索引（`states_project_id_name_key`、`states_project_id_default_key`、`states_project_id_triage_key`）和建项目时的六个默认状态都在 P4a（`00013_project_states.sql`）；删除项目、删除工作区的连带里状态一步也在 P4a（`deleteProjects`），P7a 不改。`access` 的规则表加六行；`workspace` 模块不改。

### 2.3 共用取锁路径（Task 1；3.6 约定二、三，P5b review 第 6 节）

```go
type placed interface {
	Place() (id, workspaceID, projectID uuid.UUID)
}

type rowWrite[R placed] struct {
	id       uuid.UUID
	action   shared.Action
	find     func(ctx context.Context, id uuid.UUID) (r R, found bool, err error)
	targets  func(r R) []uuid.UUID
	notFound error
}

func lockRowAndDecide[R placed](ctx context.Context, l Locks, actor shared.Actor, rw rowWrite[R]) (held, R, error)
func (rw rowWrite[R]) read(ctx context.Context) (R, error)
```

- 顺序：`rw.read`（不加锁，未删除的这一行，经写自己的端口；它说出项目和项目的工作区）→ `l.lock`（工作区 `FOR SHARE` → `targets` 的工作区成员关系 `FOR SHARE`（约定三，只有改角色给）→ 项目 `FOR NO KEY UPDATE`）→ `rw.read` 再读，必须仍属那个项目 → `decide`。回答锁下读到的行。没有这一行、已删除、等锁时删除的工作区或项目、不再属那个项目的行、调用者看不到的项目，都是 `rw.notFound`；规则不允许的是 Authorizer 的 403；这一行的状态拒绝什么（组里最后一个、默认状态），是用例在判定之后的事。
- `rw.read`：`rw.find` 没有时 `rw.notFound`；回答的行的 id 不是问的那个时是错误（不是 404）。`Locks.lock` 核对 `ShareWorkspaceByID` 回答的工作区：不是 `workspaceID` 时是错误（P5b review 第 6 节 P2：共用路径上唯一没核对键的回答）。
- `app.ProjectMembership.Place`（`member_ports.go`）；`domain.State.Place`（Task 2）。`held` 不再有 `member`；`lockMemberAndDecide`、`Locks.member` 删除（P5b review 第 6 节 m1：没有近似的副本）。`ProjectLocks` 不再带 `MemberFinder`：改角色、移出的端口（`MemberRoleChanger`、新的 `MemberRemover`）各自带它，按行寻址的写经自己的端口读自己的行；`MemberEnder` 是移出和离开共用的一步。
- 行为与 P5b 相同：改角色、移出的调用记录、顺序、每个拒绝的码都不变（`TestUpdateProjectMemberRefuses`、`TestRemoveProjectMemberRefuses` 的其余各行照旧通过；P5b 的竞争、锁强度和交错在 Task 1 之后照旧通过，Task 9 把前两者推广到状态的行）。
- `TestLocksCheckEachAnswerAgainstItsKey`（新，`app/lock_test.go`）：三个核对各一行，经一个三者都走到的写（bob 把 alice 在 web 的角色改为访客）：acme 的锁回答另一个工作区（第 3 个调用之后停下）、这一行第一次读回答另一个 id（第 2 个之后）、锁下重读回答另一个 id（第 6 个之后），各是写自己的错误，之后什么都不运行，alice 的成员关系不变。P5b 的两个测试里"回答另一个 id"的 3 行删去（第 3 节第 1 条）。

### 2.4 状态的规则（Task 2；3.17、5.2、5.3）

`project/domain/state.go`（完整内容）：

- `type StateGroup string`，`GroupBacklog`、`GroupUnstarted`、`GroupStarted`、`GroupCompleted`、`GroupCancelled`、`GroupTriage`；`State{ID, WorkspaceID, ProjectID, Name, Description, Color, Group, Default, Sequence, CreatedAt, UpdatedAt}`，`State.Place()`；`NewState` 加 `Description`；`StateCreate{Name, Color, Group, Description}`；`StatePatch`（每个字段都是指针，给了的才改）。
- `CheckNewState(StateCreate) error`：名称和颜色 1–255 个字符、不空白、不含 NUL；组是五个之一，`triage` 是 `not_allowed`（收集箱的状态不在这里建，M7），别的值是 `invalid_format`；说明不含 NUL。有问题的字段一次全部报告（一个 422 `validation_failed`）。名称是否已被占用由数据库判断（第 3 节第 6 条）。`CheckStatePatch(StatePatch) error`：只查给了的字段，规则同上；`sequence` 可以是任何数。
- `SequenceAfter(greatest *float64) float64`：最大值加 15000；`nil` 时 65535（Plane `State.save()`，`db/models/state.py:117-128`，列的默认值）。`CheckGroupKept(left int) error`：一个写把状态从它的组拿走（删除或改到别的组）之后组里还剩 `left` 个，0 时 `ErrStateLastInGroup`。
- `project/domain/errors.go`：`ErrStateNotFound`（404 `project.state_not_found`）、`ErrStateNameTaken`（409 `project.state_name_taken`）、`ErrStateLastInGroup`（409 `project.state_last_in_group`）、`ErrStateDefault`（409 `project.state_default`）。它们的说明对每个收到它的调用者都成立（清扫 30）；它们随第一个回答它们的操作进入契约（Task 4、6、7）。
- 测试：`TestCheckNewStateReportsEveryField`（14 行）、`TestCheckNewStateAcceptsValidStates`、`TestCheckStatePatch`、`TestSequenceAfter`、`TestCheckGroupKept`（按切片的顺序，清扫 7）、`TestStatePlace`、`TestDefaultStates`。

### 2.5 存储（Task 3；3.6、3.17、6.7）

十条语句，写进 `project/adapter/postgres/queries/states.sql`（`CreateState` 由 P4a 的 `:exec` 改为 `:one`，回答存下的行，加 `description`；其余九条新加），分诊状态只由 `"group" = 'triage'` 识别（3.17）：

```sql
-- name: CreateState :one
-- createProject's six states and createState's one (M3 design 3.17): the row as stored.
INSERT INTO states (id, workspace_id, project_id, name, description, color, sequence, "group", "default", created_by_id,
                    updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(name), sqlc.arg(description), sqlc.arg(color),
        sqlc.arg(sequence), sqlc.arg(state_group), sqlc.arg(is_default), sqlc.arg(created_by), sqlc.arg(created_by), sqlc.arg(now),
        sqlc.arg(now))
RETURNING id, workspace_id, project_id, name, description, color, "group", "default", sequence, created_at, updated_at;

-- name: StateByID :one
-- A write on a state named by its id (M3 design 3.6 convention 2, 6.7): the undeleted state, read first without a lock
-- for its project and the project's workspace, then again under their locks. The triage state is none (3.17).
SELECT id, workspace_id, project_id, name, description, color, "group", "default", sequence, created_at, updated_at
FROM states
WHERE id = sqlc.arg(id) AND "group" <> 'triage' AND deleted_at IS NULL;

-- name: ListStates :many
-- listStates (M3 design 3.12, 3.17): the project's undeleted states but its triage state, by sequence, then id; none
-- while the project is archived (Plane's views/state/base.py:37).
SELECT s.id, s.workspace_id, s.project_id, s.name, s.description, s.color, s."group", s."default", s.sequence, s.created_at,
       s.updated_at
FROM states s
         JOIN projects p ON p.id = s.project_id
WHERE s.project_id = sqlc.arg(project_id) AND s."group" <> 'triage' AND s.deleted_at IS NULL AND p.archived_at IS NULL
ORDER BY s.sequence, s.id;

-- name: GreatestSequence :one
-- createState, under the project's FOR NO KEY UPDATE (M3 design 3.17): the greatest sequence of the project's undeleted
-- states but its triage state, which Plane's State.objects leaves out (db/models/state.py:65-68); no row when it has
-- none.
SELECT sequence
FROM states
WHERE project_id = sqlc.arg(project_id) AND "group" <> 'triage' AND deleted_at IS NULL
ORDER BY sequence DESC
LIMIT 1;

-- name: UpdateState :one
-- updateState, under the project's FOR NO KEY UPDATE (M3 design 3.17, 6.7): a field left out, null here, keeps its
-- value. A deleted state and the triage state are not written.
UPDATE states s
SET name          = coalesce(sqlc.narg(name)::text, s.name),
    description   = coalesce(sqlc.narg(description)::text, s.description),
    color         = coalesce(sqlc.narg(color)::text, s.color),
    "group"       = coalesce(sqlc.narg(state_group)::text, s."group"),
    sequence      = coalesce(sqlc.narg(sequence)::double precision, s.sequence),
    updated_by_id = sqlc.arg(updated_by)::uuid,
    updated_at    = sqlc.arg(now)
WHERE s.id = sqlc.arg(id) AND s."group" <> 'triage' AND s.deleted_at IS NULL
RETURNING s.id, s.workspace_id, s.project_id, s.name, s.description, s.color, s."group", s."default", s.sequence, s.created_at,
    s.updated_at;

-- name: CountGroupStates :one
-- updateState and deleteState, under the project's FOR NO KEY UPDATE (M3 design 3.17): how many undeleted states the
-- project's group has.
SELECT count(*)
FROM states
WHERE project_id = sqlc.arg(project_id) AND "group" = sqlc.arg(state_group) AND deleted_at IS NULL;

-- name: DeleteState :execrows
-- deleteState, under the project's FOR NO KEY UPDATE (M3 design 3.17): a guarded write. The undeleted state, unless it
-- is the project's default or its triage state, deleted at the moment and by the account given; no row otherwise.
UPDATE states
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE id = sqlc.arg(id) AND NOT "default" AND "group" <> 'triage' AND deleted_at IS NULL;

-- name: ClearDefaultState :exec
-- markDefaultState's first statement, under the project's FOR NO KEY UPDATE (M3 design 3.17): the project's undeleted
-- default state the default no longer, at the moment and by the account given. states_project_id_default_key holds at
-- most one.
UPDATE states
SET "default" = false, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(updated_by)::uuid
WHERE project_id = sqlc.arg(project_id) AND "default" AND deleted_at IS NULL;

-- name: SetDefaultState :execrows
-- markDefaultState's second statement: the project's undeleted state the default, unless it is the triage state, at the
-- moment and by the account given; no row otherwise, and the caller rolls the first statement back.
UPDATE states
SET "default" = true, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(updated_by)::uuid
WHERE id = sqlc.arg(id) AND project_id = sqlc.arg(project_id) AND "group" <> 'triage' AND deleted_at IS NULL;

-- name: ListWorkspaceStates :many
-- listWorkspaceStates (M3 design 3.12, 3.17, 5.1): the undeleted states but the triage states of the workspace's
-- undeleted, unarchived projects the account is an active member of, by project, then sequence, then id.
SELECT s.id, s.workspace_id, s.project_id, s.name, s.description, s.color, s."group", s."default", s.sequence, s.created_at,
       s.updated_at
FROM states s
         JOIN projects p ON p.id = s.project_id
         JOIN project_members m ON m.project_id = p.id
WHERE p.workspace_id = sqlc.arg(workspace_id) AND p.deleted_at IS NULL AND p.archived_at IS NULL
  AND m.member_id = sqlc.arg(user_id) AND m.is_active AND m.deleted_at IS NULL
  AND s."group" <> 'triage' AND s.deleted_at IS NULL
ORDER BY s.project_id, s.sequence, s.id;
```

`postgresadapter.Store` 的方法（`states.go`），都在 `ctx` 带着的事务里执行（写的都在项目的 `FOR NO KEY UPDATE` 之下，Task 4–7）：

- `CreateState(ctx, app.StateRow) (domain.State, error)`：同名是 `domain.ErrStateNameTaken`，只认 `states_project_id_name_key`；别的唯一键、CHECK 是内部错误。`CreateStates`（建项目的六个，`rows.go`）经它逐行插入。
- `StateByID(ctx, id) (domain.State, bool, error)`、`ListStates(ctx, projectID) ([]domain.State, error)`（项目已归档时空列表，不是 `nil`）、`GreatestSequence(ctx, projectID) (*float64, error)`（没有时 `nil`）、`CountGroupStates(ctx, projectID, group) (int, error)`、`ListWorkspaceStates(ctx, workspaceID, userID) ([]domain.State, error)`。
- `UpdateState(ctx, id, domain.StatePatch, by, now) (domain.State, error)`：只改给了的字段（`coalesce`），审计列改为给的账户和时刻；同名是 `ErrStateNameTaken`；已删除的、分诊状态不写，是错误（用例在锁下重读过，到不了）。
- `DeleteState(ctx, id, by, now) (bool, error)`：守卫的写；`false` 时什么都不写（3.17）。
- `MarkDefaultState(ctx, projectID, id, by, now) (bool, error)`：`ClearDefaultState`、`SetDefaultState` 两条语句；第二条写 0 行时 `false`，由调用者回滚第一条；第二条失败时回答这个失败，不是 `false`。
- 测试（`states_test.go`、`state_writes_test.go`；`stateWorld`：acme 的 Web、Ops 和 beta 的 Site 各有六个默认状态，由 maker 在 `earlier` 建，测试的写都由 alice（清扫 14）；`tableRows`、`columns` 比较被写的行之外每一行每一列（清扫 3））：`TestCreateState`、`TestCreateStateNameTaken`（Web 里再建 In Progress、Triage 各 409、什么都不存；小写的 in progress、已删除状态的名称、别的项目的名称可以）、`TestCreateStateBreakingAnotherConstraintIsInternal`、`TestStateByID`（问两个时不回答另一个；分诊、已删除、不存在的都没有）、`TestListStates`（Review 最后加入、与 In Progress 同 `sequence`、id 更小，排在它之前：行在表里的顺序给不出这个次序，前提由 `ctid` 核对（清扫 48）；不含已删除的、别的项目的；归档的 Ops 空列表）、`TestGreatestSequence`（55000：不是 Triage 的 65000、已删除的 99999、Ops 的 80000；没有状态的、只有分诊状态的项目没有）、`TestCountGroupStates`、`TestListWorkspaceStates`（alice 的 Web、Ops；成员关系已结束的 Docs、已归档的 Arch、已删除的 Gone、她唯一的成员关系已删除的 Lab、只有 bob 的 Team、beta 的 Site 都不算；同样的 `ctid` 前提）、`TestUpdateState`（五行）、`TestUpdateStateNameTaken`（Done、Triage 各 409）、`TestUpdateStateWritesNoDeletedOrTriageState`、`TestDeleteState`（默认、分诊、已删除（保留时刻）、不存在的都答 `false`、什么都不写）、`TestMarkDefaultState`、`TestMarkDefaultStateRefuses`（已删除的、分诊的、别的项目的（两个方向）、不存在的：答 `false`，在调用者的事务里第一条语句已运行：这个项目此时没有默认，另一个项目的照旧；回滚之后每一行不变）、`TestMarkDefaultStateAnswersTheFailureOfItsSecondStatement`（第二条等另一个事务 `FOR UPDATE` 持有的 Todo，到调用者的 `lock_timeout`：答 55P03，不答"没设"；清扫 19，附录 A）；`failures_test.go` 加九个方法（清扫 19）。
- 每条语句都没有"取第一行"之外的行序依赖：`GreatestSequence` 带 `ORDER BY sequence DESC LIMIT 1`，两个列表带完整的 `ORDER BY`，`ListStates`、`ListWorkspaceStates` 的次序由与表里的顺序相反的一行核对（清扫 1 的"两种行序"、清扫 48）。

### 2.6 `createState`（Task 4；3.4、3.6、3.17、3.19、5.1–5.3、9.2）

- 契约：`POST /api/v0/projects/{project_id}/states`，201 `State`；`x-problem-codes: [validation_failed, project.not_found, forbidden, project.state_name_taken]`。`State`、`StateCreate`、`StateGroup`（五个组；`triage` 不在枚举里。请求体的结构检查（`bodyshape.gen.go`）只看字段和类型，不看枚举，`group = triage` 由领域答 422 `not_allowed`，处理函数原样传给用例）。`State` 和 `State.default` 的说明不说工作项（第 3 节第 7 条）。
- `access`：`state.create`（项目级，项目管理员；同时是工作区管理员的项目成员由 Authorizer 的通则得到，3.4；不给项目访客，9.2）。`ActionStateCreate`。
- `project/app/state_ports.go`：`StateRow{ID, WorkspaceID, ProjectID, CreatedBy, Now, State domain.NewState}`；`StateCreator`（`ProjectLocks`、`GreatestSequence`、`CreateState`）。
- `CreateState`：`NewCreateState(locks Locks, states StateCreator, tx shared.TxManager, clock Clock)`；`Execute(ctx, projectID, domain.StateCreate) (domain.State, error)`：没有调用者、领域拒绝的值，在事务之前；事务里 `lockAndDecide`（工作区 S → 项目 N → 判定）→ `GreatestSequence` → 时钟 → `CreateState`（`SequenceAfter`，不是默认）；回答的 id 不是插入的 id 时是错误（清扫 22）。已归档的项目照常建（3.19）。
- HTTP：`UseCases.CreateState`，处理函数，`state(domain.State) gen.State`（`states.go`）。前端文案：`project.state_name_taken`。
- 测试：`TestCreateState`、`TestCreateStateRefuses`（10 行）、`TestCreateStateReturnsEachFailure`（七个调用）；`TestEachWriteReadsTheClockUnderItsLock` 加一行；HTTP 的 `TestCreateState`、`TestCreateStateHoldsTheBodyToItsStructure`、`TestCreateStateRefusals`；组合：矩阵三行，`TestEachWriteOnAProjectSharesItsWorkspaceFirst`、`TestTheWritesOnAProjectStampTheirRequest`、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection` 各加一行，`TestMatrixViolationsCatchesEachColumnGap` 加一个从没有种下的状态。
- 矩阵的状态种子（`permission_matrix_seed_test.go`、`permission_matrix_seeded_test.go`）：`matrixStates` 是每个矩阵项目的六个默认状态加 Review（started，40000）；`prepareMatrix` 经存储（`CreateStates`）建出，`seeded.state(key, name)` 回答它们的 id（没有种下的名称立即失败）；前提核对每个项目恰好这些状态、默认的是 Backlog、分诊的是 Triage。

### 2.7 `updateState`（Task 5、6；3.4、3.6、3.17、3.19、6.7、9.2）

- Task 5（用例）：`StateFinder`（`StateByID`，`rowWrite.find`）、`GroupCounter`（`CountGroupStates`）、`StateUpdater`；`NewUpdateState(locks, states StateUpdater, tx, clock)`；`Execute(ctx, id, domain.StatePatch)`：没有调用者、领域拒绝的值，在事务之前；事务里 `lockRowAndDecide`；给了组、且与现在的组不同时，数现在的组，`CheckGroupKept(n - 1)`（判定之后、写之前，6.7）；时钟；`UpdateState`；回答的 id 不是这一行的时是错误。`state.update`、`ActionStateUpdate`。测试：`TestUpdateState`（四行）、`TestUpdateStateRefuses`（12 行）、`TestUpdateStateReturnsEachFailure`（八个调用）、时钟一行。
- Task 6（接口和组合）：`PATCH /api/v0/states/{state_id}`，200 `State`；`[validation_failed, project.state_not_found, forbidden, project.state_name_taken, project.state_last_in_group]`；`StateID`、`StateUpdate`。HTTP、接线；文案 `project.state_not_found`、`project.state_last_in_group`。矩阵：`ofState`、`ofArchivedState`、`toState`，`updateState` 五行；列的核对（`permission_matrix_columns_test.go`、`permission_matrix_targets_test.go`）认得瞄准状态的格子。最先锁工作区的测试推广到按行寻址的写（`underRow`、`rowPaths`、`stateNamed`）：第一步探测它的这一行没有被持有；盖戳、连接加 `updateState`。

### 2.8 `deleteState`、`markDefaultState`（Task 7；3.6、3.17、3.19、9.3 交错 10）

- 契约：`DELETE /api/v0/states/{state_id}`（204；`[project.state_not_found, forbidden, project.state_default, project.state_last_in_group]`）、`POST /api/v0/states/{state_id}/mark-default`（204；`[project.state_not_found, forbidden]`）。`state.delete`、`state.mark_default`（同 `state.create`）。端口 `StateDeleter`、`DefaultMarker`。文案 `project.state_default`。
- `DeleteState`：`lockRowAndDecide` → 时钟 → `DeleteState`（守卫的写，不先查默认）；删了 0 行时 `deletedNothing(ctx, rw)` 经 `rw.read` 重读这一行（还在锁下；它核对回答的 id）：默认的答 `ErrStateDefault`，没有了的答这一行的 404，既在又不是默认的是错误；删了之后 `CountGroupStates` 数它的组，`CheckGroupKept(left)`：拒绝时事务回滚删除。
- `MarkDefaultState`：`lockRowAndDecide` → 时钟 → `MarkDefaultState`；第二条语句写 0 行时 `ErrStateNotFound`，事务回滚第一条：项目不会没有默认状态。哪一种交错到得了哪一个守卫，见第 3 节第 4 条。
- 测试：`TestDeleteState`、`TestDeleteStateRefuses`（12 行）、`TestDeleteStateReturnsEachFailure`、`TestDeleteStateChecksTheReadAfterIt`（重读失败原样返回；重读回答另一个状态，写自己的错误，不是 `project.state_default`）、`TestMarkDefaultState`、`TestMarkDefaultStateRefuses`（9 行，含第二条语句跳过的状态）、`TestMarkDefaultStateReturnsEachFailure`；时钟两行；HTTP 两组；矩阵八行；最先锁工作区、盖戳（设为默认写的两行都先盖上 bob 的戳）、连接各加两个写。

### 2.9 两个读（Task 8；3.4、3.12、3.17、6.4、9.2、9.3、9.4）

- 契约：`GET /api/v0/projects/{project_id}/states`（`listStates`，200 `StateList`；`[project.not_found, forbidden]`）、`GET /api/v0/workspaces/{slug}/states`（`listWorkspaceStates`，200 `StateList`；`[workspace.not_found]`）；`StateList{data: [State]}`。
- `access`：`state.list`（项目级，项目的每个有效成员）、`workspace_state.list`（工作区级，每个有效成员；列表只含他是有效成员的未归档项目，可见性不放宽：Plane `views/workspace/state.py:20-26`）。
- `ListStates`：`findAndDecide`（`state.list`）→ `ListStates`，存储的顺序原样；不开事务。`ListWorkspaceStates`：`findWorkspaceAndDecide`（`workspace_state.list`）→ `ListWorkspaceStates(ws.ID, actor.UserID)`，不论角色。
- `findWorkspaceAndDecide`（`lock.go`）：按 slug 找到未删除的工作区、判定工作区级的操作，回答工作区和授权；没有、看不到的是 `ErrWorkspaceNotFound`。`listProjects`、`checkProjectIdentifier` 原来各写一遍这两步，改用它（行为不变，它们的测试照旧通过；不留重复的逻辑）。
- 测试：`TestListStates`（三行：存储的顺序既不是按 `sequence` 的、反过来的，也不是按 id 的，用例原样回答，清扫 16）、`TestListStatesRefuses`（7 行）、`TestListWorkspaceStates`（四种排序都给不出存储的顺序）、`TestListWorkspaceStatesRefuses`（6 行）；HTTP 两组（`workspace.not_found` 由 `project` 自己的 HTTP 测试返回，9.4）；矩阵三行；`TestListingWorkspaceStatesIsListingEachProjects`（`project_visibility_test.go`，9.3 的可见性一致：矩阵的每个账户，工作区的列表恰好是对 acme 每个项目逐个 `listStates` 的并，按项目 id 的顺序）；`TestListingWorkspaceStatesKeepsToItsWorkspace`（同一文件，在 `memberWorld` 上：bob、carol 是两个工作区的项目的成员，Web 有一个已删除的状态；每个工作区的列表只有它自己的项目的、未删除的状态，清扫 23；矩阵的账户只是一个活的工作区的项目的成员）。

### 2.10 并发（Task 9；3.6 约定二、三，3.17，9.3 交错 10）

- `project_membership_races_test.go` 改名为 `project_row_races_test.go`，`membershipWrite` 推广为 `rowWrite`（`member`、`state`、`clears`；`row` 回答它改的表和行；`param`），`rowWrites` 有 P5b 的三个写和状态的三个写（bob 改 Web 的 QA：改名、删除、设为默认）：
  - `TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`：另一个事务持有 Web 的 `FOR NO KEY UPDATE`（或 acme 的，如连带），其间这一行结束（只有成员关系有）、删除、移到 Ops，调用者的成员关系结束，Web 删除，acme 删除；写在通过认证和不加锁的读之后等它，提交之后答 404，什么都不改（3.6 约定二、8.2）。
  - `TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`：另外的事务持有 acme 的 `FOR NO KEY UPDATE`、Web 和这一行的 `FOR SHARE`，逐个放开，`lockOn` 读出每一行此时最强的锁：等 acme 时什么都没持有；等 Web 时持有 acme 的 `FOR SHARE`（改角色另有成员的工作区成员关系 `FOR SHARE`）；等这一行时持有 Web 的 `FOR NO KEY UPDATE`，只有设为默认持有 Web 原来的默认 Backlog（`FOR NO KEY UPDATE`，它先写了它，3.17）。写的时刻不早于 Web 的放开、早于这一行的放开（3.3）。
- `memberWorld`（`membership_world_test.go`）：Web 多一个 completed 组的 QA，由 alice 经接口建；alice 在 QA 之后建、随即删除的 Retired（85000）；Ops 的 Cancelled 由 alice 移到 100000，在 Web 的每个状态之后（清扫 23、24、26：读了别的项目的、已删除的状态的最大 `sequence` 的创建在组合一层看得出）；`projectLocks`、`tx`（`interleaving_leaving_test.go`）由离开和状态的写共用。
- `TestStateWritesOnOneProjectSerialize`（`interleaving_states_test.go`）：第一个写持有 acme 的 `FOR SHARE`、Web 的 `FOR NO KEY UPDATE` 和它写的行，停在它写之后的门；第二个共享 acme、等 Web 的行（两者都不写 `projects` 的行，只有这把锁的等待满足探测）。第一个提交之后，第二个照第一个提交的判定：completed 组的 Done 和 QA 各删除或改走，第二个 409 `project.state_last_in_group`；QA 先设为默认再删除，删除 409 `project.state_default`；先删除再设为默认，404 `project.state_not_found`（交错 10，两种顺序 Web 都恰好一个默认）；两个创建，第二个在第一个之后的 `sequence`，时刻不早于门打开的时刻（它在第一个提交之后取得的锁下读时钟，3.3）。之后 Web 的状态是第一个留下的样子（第二个成功时加上它的），各带组、`sequence`、默认；Ops、beta 的 Lab 不变（清扫 49）。
- `TestEachStateWriteChangesItsRowsAlone`（`state_rows_test.go`）：bob 建 Shipped、改 QA 的名称、删除它、把 Backlog（backlog 组唯一的状态）改名并给它自己的组（不是改组：不数，清扫 24）、把 Done 设为默认；其间他在分诊组建状态、把 Done 改到分诊组，各 422 `group not_allowed`（在写的事务之前），什么都不写；两个请求体在契约的枚举之外，测试只核对回答。每个写之后，它写的行之外每张表每一行不变（清扫 49）。

### 2.11 端到端与文档（Task 10；第 2 节 P6、W11，3.20、4.11）

- `e2e/stories/project/p6-states.spec.ts`（P6 的接口版本）：Web 的管理员 ann 建 Review（started，70000）；工作区管理员 admin（他建了 Web，是它的另一位管理员）改它的颜色（`sequence` 不变），再把它拖到 In Progress 之前（30000，颜色不变）；ann 把它设为默认，Backlog 不再是；之后五个拒绝：删除默认的 Review 409 `project.state_default`、删除组里唯一的 Backlog、把它改到有两个状态（Review、In Progress）的 started 组，各 409 `project.state_last_in_group`（数的是它自己的组）、建分诊组的状态 422 `{field: group, code: not_allowed}`、项目成员 mem 改状态 403；ann 删除 In Progress（最后由 admin 写）；之后再删除它、改 Web 的分诊状态的名称，各 404 `project.state_not_found`；mem 的列表；ann 建 Ops，工作区的列表按项目 id；ann 归档 Ops，它列出 `[]`，工作区的列表只有 Web 的；最后 admin 在 Elsewhere 的两个项目（`amidAnotherWorkspace` 建在 Web 前后）的状态照建出时的样子：Web 上的写没有写别的项目。每一步之后、每组拒绝之前之后用 SQL 读出状态的行（`expectStates`：名称、颜色、组、`sequence`、默认、删除、最后写它的人、行在它的工作区、删除的时刻就是它最后一次写的时刻；清扫 27、38）。
- `e2e/stories/workspace/w11-guest-bounds.spec.ts`（W11 的四格抽样）：acme 和 Web 的访客 gus：读邀请 403 `forbidden`、建状态 403、读他不是成员的私有项目 Secret 404 `project.not_found`、成员的邮箱对他为 `null`（管理员看得到）；之后五张表（`workspace_members`、`workspace_member_invites`、`projects`、`project_members`、`states`）不变。
- 夹具：`api.ts` 的 `State`、`StateCreate`、`StateUpdate`，`answer`（从 P5 的故事提到夹具，P5、P6 共用），`createState`；`assert/project.ts` 的 `expectStates`、`statesOfANewProject`（P4a 的六个默认状态，由建项目的人）。
- `docs/v0/plane-diff.md` 4.11：修改状态的人（项目管理员，或同时是工作区管理员的项目成员）、默认状态和分诊状态一行补上状态的操作按 `group` 认出分诊状态和设为默认的事务、一组中唯一的状态 409。`e2e/fixtures/assert/workspace.ts`、`project/app/deletion.go` 的说明里标签的 Phase 改称 P7b（拆分报告 2.4）。

### 2.12 矩阵

状态的 19 行（`permission_matrix_states_test.go`）：`listStates` 2 行（每个有效成员 200，`PM+WA` 作为成员、`WA-` 不是成员 403；已归档项目的小表）、`createState` 3 行（成功、名称已占用：409 在判定之后，不能建状态的人不知道项目有哪些状态；已归档）、`updateState` 5 行（改名、组里最后一个改走、名称已占用、分诊状态每一列 404、已归档）、`deleteState` 5 行（Review、默认的 Backlog、组里唯一的 Done、分诊状态、已归档）、`markDefaultState` 3 行（Todo、分诊状态、已归档）、`listWorkspaceStates` 1 行（每个有效成员 200：管理员和成员不是任何矩阵项目的成员，列表为空；访客是公开和私有项目的成员，按项目 id 得两个项目的状态）。写的格子各在自己的数据库副本上运行。已归档项目的小表（`archivedColumns`：PA、WM、X）随每个操作，在加这个操作的 Task（第 3 节第 3 条）。矩阵一共 709 格（P6 结束时 532 格，P7a 加 177 格：13 行各 12 格、5 个已归档的小表各 3 格、`listWorkspaceStates` 6 格）。

## 3. 与设计的差异和补充

第 1、4、5 条落实控制者的裁定 S6、S7；其余是本 spec 的决定，待控制者确认。

1. **工作区锁的键测一次，在共用路径上（裁定 S6，O1）**。`Locks.lock` 核对 `ShareWorkspaceByID` 回答的工作区的 id，`rowWrite.read` 核对这一行的回答的 id（锁前、锁下各一次）。三个核对都是共用路径的，由 `TestLocksCheckEachAnswerAgainstItsKey`（`app/lock_test.go`）各一行钉一次（2.3）；P5b 的 `TestUpdateProjectMemberRefuses`、`TestRemoveProjectMemberRefuses` 里"回答另一个 id"的 3 行删去，说明改指向它；第一次运行在 `createState`、`updateState`、`deleteState`、`markDefaultState`、`leaveProject`、`updateProjectMember` 的拒绝表里加的同类 12 行不带过来（附录 A）。
   - **没有用例绕过这条路径**：`ShareWorkspaceByID` 只在 `Locks.lock` 里调用（`grep` 全包）；项目级的每个写（`updateProject`、`archiveProject`、`unarchiveProject`、`deleteProject`、`updateProjectPreferences`、`addProjectMembers`、`joinProject`、`leaveProject`、`createState` 经 `lockAndDecide`，`updateProjectMember`、`removeProjectMember`、`updateState`、`deleteState`、`markDefaultState` 经 `lockRowAndDecide`）都经 `Locks.lock` 取工作区的锁。这是结构上的：`project.New` 给这些写的构造函数只传 `Locks`，不传 Authorizer，一个写不经 `Locks` 就做不了判定（`Locks` 的说明）。`createProject` 按 slug 锁工作区（`ShareWorkspaceBySlug`）：它回答的 `app.Workspace` 没有 slug，键无从核对（P4a 的形状，第 7 节）；读不取锁。以后的写（P7b 的标签）照样经这两个入口，路径的核对不再在各用例的表里重复。
   - **按性质只在单元一层**：真实的存储回答不了别的键（按主键的一行），这三个核对是路径与它的端口之间的一致性检查，不是只在单元一层被发现的安全性质。它们的变异（`s22-workspace-key`、`s22-row-id`）只在单元一层失败，附录 A 记为"按性质"。`createState`、`updateState` 各自核对写回答的 id（`s22-created-id`、`s22-updated-id`），不是路径的，留在各自的表里，同样按性质只在单元一层。
2. **P5b 按行寻址的写是两个，不是三个**。设计 12 节 P7a 任务 1 说"项目成员的三个写改走它"；离开（`POST /projects/{project_id}/leave`）按项目寻址，没有一行可以先读，照旧经 `lockAndDecide`，与按行寻址的两个写共用 `Locks.lock`（工作区锁的键的核对在那里，第 1 条）。完成线的"项目成员的三个写的竞争和锁的强度测试在共用路径上照旧通过"照旧成立：`rowWrites` 有离开（2.10）。建议设计的这一句在收尾时改为"按行寻址的两个写改走它，离开经同一个 `Locks.lock`"（第 5 节"收尾"）。
3. **任务的切分**（裁定 S3、S4；设计的十个任务仍是十个）：
   - `project.state_not_found` 在 Task 6，不在 Task 4：`createState` 按项目寻址，回答不了它；码随第一个回答它的操作（`updateState`）进入契约，`apitest.Main` 的两个方向从那时起核对它，文案随之（约束 4）。
   - 设计的任务 5（`updateState` 的接口）和 6（`updateState` 在组合出的 app 上）改为 Task 5 只有用例（端口、规则行、操作名、假实现、单元测试），Task 6 是契约、HTTP、接线、矩阵、最先锁工作区、盖戳和连接：契约的码要在同一个 Task 由 HTTP 测试返回（`apitest.Main`），矩阵的行要接好的 app；第一次运行的 T4（块 1,518 行，说明之前已超出约 1,500 行）由此分成两个（附录 A"Phase 的大小"）。
   - 已归档项目的小表中状态的行随每个操作（Task 4、6、7、8），不集中在 Task 10：设计 12 节 P7a 的"每个加操作的任务同时加它的……矩阵行"，完整性核对要求契约的每个操作从它出现起就有这些。设计的任务 10 由此只剩端到端和文档；"review"是 Phase 的评审，不是 plan 的 Task。
4. **哪一种交错到得了哪一个守卫（裁定 S7，O4；清扫 50）**。3.17 的两个守卫：
   - `DeleteState` 的守卫（`NOT "default"`）在组合一层到得了：交错 10 的"先设为默认、再删除"，删除在 Web 的锁上等设为默认提交，锁下重读的 QA 已是默认，`deleteState` 不先查默认，守卫的 `UPDATE` 删 0 行，`deletedNothing` 重读答 409 `project.state_default`（`TestStateWritesOnOneProjectSerialize`）；矩阵的 `deleteState` 默认的一行（没有并发）也经它。
   - `SetDefaultState` 写 0 行时让事务失败的守卫，在组合一层到不了：交错 10 的"先删除、再设为默认"，设为默认在 Web 的锁上等删除提交，锁下重读（`lockRowAndDecide`）已读不到 QA，先答 404，到不了第二条语句。项目的 `FOR NO KEY UPDATE` 之下，第一次读和锁下重读之间没有别的写能删除或移走这一行，第二条语句写 0 行只在路径的重读与语句不一致时出现：它是纵深防御，钉在单元一层（`TestMarkDefaultStateRefuses` 的"第二条语句跳过的状态"：404、事务回滚第一条）和存储一层（`TestMarkDefaultStateRefuses` 的五种跳过：第一条已运行、回滚之后每一行不变）。它的两个变异按性质在组合一层之下失败（附录 A）：`g-mark-nothing-ok`（用例在写 0 行时照样提交）在单元一层，`g-mark-n`（存储方法不看第二条语句写了几行）在存储一层。本 spec、plan、代码的说明里没有一句说它在组合一层被交错走到。
5. **清扫 41：每个写多行的语句与同一些行的其他写者（裁定 S7，O2；P6 review 第 6 节）**。P7a 的语句里，按谓词写的只有 `ClearDefaultState`（`WHERE project_id = $1 AND "default" AND deleted_at IS NULL`；部分唯一索引 `states_project_id_default_key` 让它至多写一行）；`UpdateState`、`DeleteState`、`SetDefaultState` 按主键写至多一行；`CreateState` 插入新行；没有别的批量写。这些行（项目 N 的状态）的全部写者和它们持有的锁：
   - `createState`、`updateState`、`deleteState`、`markDefaultState`：工作区 `FOR SHARE` → 项目 `FOR NO KEY UPDATE`，之后才碰状态的行（共用路径，`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength` 读出每一步的锁）；
   - 建项目（P4a，`CreateStates`）：只插入这个新项目的行，别人还看不到它；
   - 删除项目（P4b，`deleteProjects` 的状态一步）：工作区 `FOR SHARE` → 项目 `FOR NO KEY UPDATE`；
   - 删除工作区的连带（P4a，`DeleteWorkspaceProjects` 的同一组步骤）：工作区 `FOR NO KEY UPDATE`，与任何状态的写持有的工作区 `FOR SHARE` 冲突；
   - 停用（P6）、结束和恢复成员关系（P5a）、项目成员的写（P4b、P5b）不写状态。
   每个能写到这些行的写都先持有项目的 N 或工作区的 N，状态的写持有工作区的 S 和项目的 N：父行的锁让它们先后进行，约定五的例外（按 id 先锁、只写锁住的）不适用。最坏的交错由真实数据库上的探测展示：交错 10 两种顺序（设为默认与删除同一个状态）、一组最后两个状态的删除和改组（`TestStateWritesOnOneProjectSerialize`，第二个写的等待由探测确认在 Web 的行上），另一个事务像删除项目那样持有 Web 的 N、像删除工作区的连带那样持有 acme 的 N，在其中删除它们，状态的写等它提交之后答 404、什么都不改（`TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile` 的"Web deleted"、"acme deleted"两行，状态的三个写各一遍）。
6. **分诊状态的名称也占用名称**。3.17 的"名称在项目内唯一"由 P4a 的部分唯一索引 `states_project_id_name_key`（`(project_id, name) WHERE deleted_at IS NULL`）保证，它不看组：建一个名为 Triage 的状态、把一个状态改名为 Triage，各 409 `project.state_name_taken`，虽然分诊状态对状态的操作不可见。契约的 `createState`、`updateState` 写明（"the intake's triage state's too"）；存储测试（`TestCreateStateNameTaken`、`TestUpdateStateNameTaken`）各有一行。这条码让调用者知道项目有一个他列不出的名称；那只是 Plane 默认建出的、每个项目都有的 Triage，不泄露项目的数据。
7. **契约不说工作项**（清扫 6、18）。第一次运行把 `State.default` 写成"没有指定状态的新工作项取它"、`State` 写成"工作项所在的状态"；工作项在 M4，P7a 的代码没有这样的行为。现在只说它是不是项目的默认状态（`a project has exactly one`）、状态属于哪一组。P4a 的 `Project` 里几个字段（标识、默认负责人、自动归档）照旧说工作项，不在 P7a 改（待控制者确认是否在收尾统一）。"状态下还有工作项时不能删"由 M4 加入（3.17、13.2）。
8. **`findWorkspaceAndDecide`**：按 slug 找到工作区、判定工作区级的操作，这两步原来在 `listProjects`、`checkProjectIdentifier` 里各写一遍，`listWorkspaceStates` 是第三个使用者；提成 `lock.go` 的一个函数（2.9），三者共用，P4a 的两个用例行为不变（它们的单元、HTTP、组合测试照旧通过）。
9. **清扫 36：别的操作或两步到达同样的结果**（brief：默认状态、一组至少一个、分诊的排除）。
   - 恰好一个默认：至多一个由索引保证；至少一个：删除默认 409（守卫），设为默认的第二条写 0 行时回滚第一条，`updateState` 改不了 `default`（`StateUpdate` 没有这个字段，多余的字段 400），新建的不是默认，改组不碰默认；两步：设为默认再删除，删除 409；删除再设为默认，404。删除项目、删除工作区连同全部状态一起删除。没有绕过。
   - 一组至少一个：删除、改走组里唯一的状态各 409；改进一个组不会让别的组变空（数的是它原来的组）；两个写一前一后由项目的 N 串行（2.10）；建项目总是建出全部五组的默认状态（P4a）。分诊组不在这条规则里，它的状态不可寻址。没有绕过。
   - 分诊的排除：建、改到 `triage` 组 422；分诊状态不在列表里、`GreatestSequence` 不看它，按 id 的读、改、删、设为默认都答 404（`StateByID` 不回答它），`UpdateState`、`DeleteState`、`SetDefaultState` 的语句另带 `"group" <> 'triage'`（存储一层核对）。改名为 Triage 是第 6 条，不让它成为分诊状态。没有绕过。
10. **与停用、连带没有新的交错**（P6 spec 第 5 节 P7 一行、review 第 6 节）。状态的写是项目级的写（工作区 S → 项目 N），不改成员关系，不是约定六的增长或收缩；停用（工作区 N → 项目 N）、删除工作区的连带（工作区 N）与它们在工作区行上串行，不成环；连带的状态一步是 P4a 的，P7a 不改。
11. **清扫 37：已归档的项目、已删除的工作区、停用的调用者**。已归档项目的状态照常建、改、删、设为默认（3.19），契约逐个写明，矩阵的已归档小表每个操作一行；`listStates` 对它答空列表，`listWorkspaceStates` 不含它（3.17）。工作区在等锁时被删除：写答它自己的 404（`TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile` 的"acme deleted"）；已删除的工作区下的状态在矩阵的 X 列（gone 的项目随它删除）。停用的调用者在认证一步被拒绝（M2、P6），到不了状态的操作。
12. **承接的条目**（拆分报告第 4 节 P7a 一列）：

    | 来处 | 条目 | 落点 |
    |---|---|---|
    | P5b review 第 6 节、spec 第 5 节 | 共用路径提出来，没有近似的副本（m1）；工作区锁的回答核对它的键（P2）；竞争和锁强度照 P5b | 2.3、第 1 条；2.10 |
    | P4b review 第 6 节、spec 第 5 节 | 每个状态的写经 `Locks`，在最先锁工作区的测试里有一行，第一步探测它的状态行；故事清扫里"先删除的行"的计数只在故事先删除一个状态、再删除它的项目时相关：P6 不删除项目（它删除的 In Progress 由 `expectStates` 读出 `deleted`），不涉及 | Task 4、6、7；2.11 |
    | P6 review 第 6 节、spec 第 5 节 | 与停用没有新的交错，工作区行上 S 对 N；`ClearDefaultState` 进清扫 41 的表 | 第 10 条、第 5 条 |
    | P2 review 第 6 节 | `pgtest` 的探测有期限，探测不存在的表立即失败（P6 加了 `WaitForLockWaitBehind` 之后照旧）：P7a 的五处探测都经 `pgtest.WaitForLockWaitOn`（有期限，按表名），没有新的探测函数；连续的探测各在不同的表上（清扫 47） | Task 6、9 |
    | P1–P6 的常设规则 | 时钟在最后一把锁之后（`TestEachWriteReadsTheClockUnderItsLock` 四行）；锁下重读确认父行；`apitest.Main` 两个方向；端口的错误原样返回；角色按集合；关键词守卫（5 个命中都有例外，P7a 不加）；矩阵不加新的列表（`archivedColumns` 已在 `projectTables`）；已归档的小表 | 各 Task |
    | P4a review 第 6 节 | 标签的两条 | P7b（第 5 节） |

## 4. 验收标准（完成线，M3 设计 12 节 P7a）

- [ ] P6、W11 的接口版本通过，此前的每个故事仍然通过（`make e2e` 共 69 个：此前的 67 个，加 P6、W11；原型和复现的副本不是仓库的检出，S3 读不到构建的提交号，单独失败，与 P7a 无关）。
- [ ] 交错 10 和一组最后两个状态的删除、改组，两种顺序，`-count=5 -race` 通过，没有 40P01（`race5.sh`：12 个测试各 5 次，470 个子测试，147.8 秒）。
- [ ] 项目成员的三个写的竞争和锁强度测试在共用路径上照旧通过（`TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength` 的成员关系三个写，P5b 的交错）。
- [ ] 每个状态的写在 `TestEachWriteOnAProjectSharesItsWorkspaceFirst` 里有一行，第一步探测它的状态行；少一行时完整性核对失败（`s34-first-lock-row`）。
- [ ] `project` 的 `apitest.Main` 两个方向核对通过：四个状态码和 `listWorkspaceStates` 的 `workspace.not_found` 各由 `project` 自己的 HTTP 测试返回。
- [ ] 整程序测试覆盖六个操作（矩阵、最先锁工作区、盖戳、连接、每个写只写它的行）。
- [ ] 状态的矩阵行和已归档项目的小表中状态的行通过，格子数和耗时记下。
- [ ] `make gen-check`、`make lint-go`、`make test`、`make lint-web`、`make knip`、`make test-web`、`make e2e` 通过。

## 5. 不在 P7a 范围内

- 标签、`labels` 表和它的两个连带、故事 P7、W3 和 P4 的标签断言：P7b。`PROBLEM_MESSAGES` 的搬迁：P8。P6、W11 的页面版本（状态设置、访客看到"没有权限"）：P11（C9）。"状态下还有工作项时不能删"：M4（3.17、13.2）。收集箱取分诊状态的接口：M7。

**留给后面的 Phase**（各 Phase 的 spec 接过去；P7a 的 review 第 6 节照录）：

| Phase | 条目 |
|---|---|
| P7b | 共用路径：`lockRowAndDecide[R placed](ctx, l Locks, actor shared.Actor, rw rowWrite[R]) (held, R, error)`，`rowWrite[R]{id, action, find, targets, notFound}`；标签实现 `Place()`，`updateLabel`、`deleteLabel` 经它（`find` 是标签的按 id 读，`notFound` 是标签的 404），`createLabel` 经 `lockAndDecide`、`listLabels` 经 `findAndDecide`；路径的键的核对已由 `TestLocksCheckEachAnswerAgainstItsKey` 钉住，标签的用例不再加这类行（裁定 S6）。竞争和阶梯：`project_row_races_test.go` 的 `rowWrite`（`row()` 加标签的一支）和 `rowWrites`；`project_write_locks_test.go` 的 `underRow`、`rowPaths`（加 `/api/v0/labels/`）、`stateNamed` 的同类；`memberWorld` 的 `projectLocks()`、`tx()`；交错 11 照 `interleaving_states_test.go` 的 `stateWrittenHolding`、门和探测。矩阵：`matrixStates` 由 `prepareMatrix` 经存储的 `CreateStates` 种下，`seeded.state(key, name)` 找它们；标签照它加种子和查找，行照 `permission_matrix_states_test.go`（`ofState`、`ofArchivedState`、`toState`，每个操作一行 `variant: "archived"`），列的核对（`permission_matrix_columns_test.go`、`permission_matrix_targets_test.go`）认得瞄准标签的格子。假实现：`fakeStates`（`fakes_state_test.go`）嵌入 `*fakeStore`，共用它的调用记录、失败（`errs`）、锁下的重读（`reread`）、`answersAs`、`changedAs`，列表的次序不是任何排序给得出的；标签的假实现照它。已归档的小表：`archivedColumns`（PA、WM、X），按项目寻址的行用 `ofArchived`，按行寻址的用 `ofArchivedState`，在加那个操作的 Task 里。第一次运行的 T8–T10 的差异（`p7snap/T7`→`T8`→`T9`→`p7proto`）在 P7a 合并之上重放时，与 P7a 改过的文件相交的是 `api/modules/project.yaml`（P7a 改了状态操作的描述和 `State` 的两句，标签的路径照旧接在状态的之后）、`project/app/deletion.go`（P7a 的说明说"P7b adds the labels at the end"，P7b 加上那一步、删去这半句）、`project/app/fakes_write_test.go`（P7a 改了 `rowReads` 的说明，P7b 加标签的假实现）：都是相邻的文字，手工合并；其余文件的差异照原样应用（第一次运行的 T9 把 `checkShortText` 移出 `state.go`，P7a 没有改 `state.go`）。生成物重新生成。P7b 另要扩展、第一次运行还没有改到的共用测试文件：`project_writes_test.go`（盖戳）、`project_write_locks_test.go`、`project_connection_test.go`、矩阵的文件、`project_row_races_test.go`、`fakes_write_test.go`。承接的条目：P4a review 第 6 节（标签经 `deletion()` 的最后一步进 `DeleteWorkspaceProjects`；W3、P4 断言标签，裁定 S4）；P4b（每个标签的写经 `Locks`，最先锁工作区的测试一行探测标签行；`deleteProjects`、`ProjectsDeleter`、手工维护的表；`keysTo` 的说明：`labels.parent_id` 指向 `labels` 自己；`expectProjectDeleted` 只数之前未删除的行）；P5b（标签是共用路径的第三种行，竞争和锁强度的标签行）；P6（`deleteLabel` 连带子标签与约定五的例外，带最坏交错的探测，裁定 S7 的 O3） |
| P8 | 四个状态码的前端文案（`PROBLEM_MESSAGES`、两份 `auth.json`）随搬迁照旧；状态的类型（`State`、`StateList`）来自 `schema.gen.ts`；`order` 不进接口，由 state store 按 `sortStates` 算（3.17、7.3） |
| P11 | P6 的页面版本：状态设置里新建、改颜色、拖动改 `sequence`、设为默认、删除；默认状态和组里唯一的状态删除按钮不可用，服务端并发下仍拒绝时显示 409 的说明（7.6）；W11 的页面版本：访客打开项目的状态设置看到"没有权限"。接口和它们的码已由 P7a 钉住 |
| M4 | "状态下还有工作项时不能删"；新工作项取默认状态（`State.default` 的说明那时补上，第 3 节第 7 条）；工作项的写是否取工作区的 S 由 M4 决定（负责人 2026-10-02） |
| M7 | 收集箱取分诊状态的接口（3.17）；分诊状态的名称占用（第 3 节第 6 条）照旧 |
| 收尾 | 3.20 的 P7a 三行（`plane-diff.md` 4.11）由 Task 10 写好，收尾逐行核对；设计 12 节 P7a 任务 1 的"三个写"一句（第 3 节第 2 条） |

## 6. 风险

| 风险 | 应对 |
|---|---|
| 共用路径的提取改了 P5b 的两个写 | 它们的单元测试只删去"回答另一个 id"的 3 行（改由路径的测试核对），其余各行的调用记录、顺序、码不变；P5b 的竞争、锁强度、交错在 Task 1 之后照旧通过（plan Task 1 的 `-count=3 -race`），Task 9 推广之后在 `race5.sh` 里各 5 次 |
| 设为默认的第二条语句写 0 行的守卫在组合一层到不了 | 第 3 节第 4 条；单元、存储两层钉住；说明不说更多 |
| 状态的写与删除项目、删除工作区的连带、停用 | 都在父行（工作区、项目）上串行，第 3 节第 5、10 条；竞争的测试读出等待在哪一行 |
| 一个大的种子（矩阵每个项目七个状态） | `prepareMatrix` 经存储建出，前提核对每个项目恰好这些；写的格子各在自己的数据库副本上 |
| 第一次运行的代码没有清扫 | 全部复审、五十类清扫、变异逐个跑过（附录 A）；改了的列在附录 A |
| 持续集成（ubuntu）上的运行 | 原型和复现都在 macOS 上；合并之后看持续集成，P1–P6 都是这样合并的 |

## 7. 已知的限制、交接和关闭条件

- **`createProject` 锁工作区的回答没有键可核对**（第 3 节第 1 条）：它按 slug 锁，回答的 `app.Workspace` 没有 slug；P4a 的形状，P7a 不改。真实的存储按 slug 的唯一索引回答，不会答另一个。
- **分诊状态的名称占用名称**（第 3 节第 6 条）：与唯一索引一致，调用者看到 409 而列不出那个状态。
- **`SetDefaultState` 的守卫在组合一层到不了**（第 3 节第 4 条）。
- **状态的说明不说工作项**，P4a 的几个字段照旧（第 3 节第 7 条）。

| 交接 | P7a 处理的条目 | 留下的条目 |
|---|---|---|
| P5b spec 第 5 节、review 第 6 节的 P7 一行 | 第 3 节第 1、12 条 | 标签是共用路径的第三种行（P7b） |
| P4b spec 第 5 节、review 第 6 节的 P7 一行 | 第 3 节第 12 条 | 标签的写和它们的行（P7b） |
| P6 spec 第 5 节、review 第 6 节的 P7 一行 | 第 3 节第 5、10 条 | `deleteLabel` 连带子标签与约定五的例外（P7b，O3） |
| P4a review 第 6 节 | 无 | 标签进 `DeleteWorkspaceProjects`、W3 和 P4 的标签断言（P7b） |
| P2 review 第 6 节 | 第 3 节第 12 条 | 无 |

**M3 设计 13.1 的关闭条件**：13.1 没有落在 P7（P7a、P7b）的一项（设计 12 节 P7a"关闭：没有"）。没有放不下的条件。

## 附录 A：原型验证记录（2026-10-05）

原型在 `$M3TMP/p7aproto`：照裁定 S5 从第一次运行的 `p7snap/T7` 复制，换上 `3f87fcd3` 的两份设计文档，`node_modules` 由仓库的安装目标装好（不用符号链接）；Go 1.27.1（`toolchain`）、Node 24、pnpm 11.10.0、Docker；`bin/golangci-lint` 2.13.2。复现的基础是 `3f87fcd3` 的一份新的副本（`p7abase-3f87fcd3.tar`，快照 T0）。做法照 P6：每个 Task 做完时存一份源文件的快照（`$M3TMP/p7asnap/T1`…`T10`），plan 的代码块由脚本从相邻两份快照的差异生成（`p7atools/mkblocks.py`），生成的文件不进块，按 SHA-256 核对（`gensha.py`）。Task 1–3 照第一次运行的 T1、T2 重新切出、手工改好；Task 4–10 由 `build.py` 按记下的步骤从前一份快照建出（从第一次运行的快照取的文件、`edits_t*.py` 的逐处替换、整份写的文件、`make gen`、包说明），之后的修改写进它所在 Task 的步骤，从那个 Task 起重建。`assemble.py` 组装 plan 并核对每个块放了一次、每个文件在文件表里、每个 Task 在约 1,500 行以内、没有 HTML 实体。

**对第一次运行的复审**（第一次运行的 T1–T7 没有清扫；这里逐条复审之后在根上改了的，按缺陷类别）：

- **裁定 S6**：路径的三个核对由 `TestLocksCheckEachAnswerAgainstItsKey` 测一次；P5b 的 3 行和第一次运行在六个用例里加的 12 行删去（第 3 节第 1 条）。
- **清扫 22**：`deletedNothing` 原来直接调用 `StateByID`，不核对回答的 id，回答另一个状态时会答 `project.state_default`；现在经 `rowWrite.read` 重读（2.8），新的 `TestDeleteStateChecksTheReadAfterIt` 两行。
- **清扫 44、45**：存储的 `TestMarkDefaultStateRefuses` 说"第一条语句已运行"，原来只看回滚之后不变，那一句不会失败；现在在调用者的事务里、回滚之前读出两个项目的默认状态：这个项目没有，另一个项目的照旧。
- **清扫 16**：单元假实现的列表原来按 `sequence` 倒序，按 id 排序的用例也会通过；`fakeStates.of` 改按名称（既不是 `sequence` 的两种次序，也不是 id 的），`GreatestSequence` 的假实现自己找最大的（不靠次序）；两个列表的单元测试先核对存储的次序不是任何一种排序给得出的。
- **清扫 7**：`TestCheckGroupKept` 原来遍历一个 map，次序随机；改为切片。
- **清扫 48**：`TestListStates`、`TestListWorkspaceStates` 的"Review 在 In Progress 之前"依赖行在表里的次序给不出这个次序，原来没有前提；现在由 `ctid` 核对。
- **清扫 6、18、50**：契约的 `State`、`State.default` 不说工作项（第 3 节第 7 条）；`createState`、`updateState` 的描述写明分诊状态的名称也占用（第 3 节第 6 条）；`bobs` 的说明（设为默认重写两行）、`rowReads` 的说明（`id` 只剩成员关系的）、两处"P7"改为 P7a、P7b，`TestCheckGroupKept` 之外几处说明照代码改写。
- **命名**：`UseCases.WorkspaceStates`、HTTP 测试的 `fakeWorkspaceStates` 改为 `ListWorkspaceStates`、`fakeListWorkspaceStates`：与字段的类型 `ListWorkspaceStatesUseCase` 同名，与其余的字段一致。
- **清扫 23、49**：新的 `TestEachStateWriteChangesItsRowsAlone`（`state_rows_test.go`）：每个状态的写之后，它写的行之外每张表每一行不变。
- **矩阵**：`seeded.state` 对没有种下的名称立即失败，`TestMatrixViolationsCatchesEachColumnGap` 加一行（第一次运行的这一检查在它之后的 Task，现在随它检查的查找一起在 Task 4）。
- **重新切分**（裁定 S4）：第一次运行 T2 的领域、存储分成 Task 2、3；T3 的 `listStates` 移到 Task 8，与 `listWorkspaceStates` 一起；T4 分成用例（Task 5）和接口与组合（Task 6），`seeded` 的按行定位的查找随第一个按 id 寻址状态的行移到 Task 6；`project.state_not_found` 随 `updateState` 的接口（第 3 节第 3 条）。端到端、文档是新的（第一次运行没有）。

**清扫之后的加强**（变异分层跑完之后）：变异按层跑完之后（单元、存储、组合、端到端，`stage-*.json`），只在单元或存储一层被发现、而组合一层本可以展示的，改测试，不改产品代码（`edits_t3b.py`、`edits_t8b.py`、`edits_t9.py`、`src/state_rows_test.go.txt`、`src/p6-states.spec.ts.txt`）：

- `TestMarkDefaultStateAnswersTheFailureOfItsSecondStatement`（存储，Task 3）：`s19-set`（第二条语句的失败读成"没设"，`markDefaultState` 会答 404）原来活下来：失败测试用取消的 context，第一条语句就失败；现在第二条语句在另一个事务持有的行上等到调用者的 `lock_timeout`。
- `TestListingWorkspaceStatesKeepsToItsWorkspace`（组合，Task 8）：`s1-lw-workspace`、`s1-lw-sdeleted`、`s1-ls-deleted` 原来只在存储一层（后两个另在 P6）：矩阵的账户只是一个活的工作区的项目的成员，活的项目没有已删除的状态（清扫 23）。现在在 `memberWorld` 上，两个工作区，Web 有一个 bob 建了又删除的状态。
- `memberWorld`（Task 9）：Ops 的 Cancelled 在 100000，Web 有已删除的 Retired（85000）：`s1-gs-project`、`s1-gs-deleted` 原来只在存储一层（每个项目的状态的 `sequence` 都一样，没有已删除的，清扫 23、26）；交错的两个创建现在看得出。
- 交错的两个创建（Task 9）：第二个的时刻不早于门打开的时刻：`c-create-early`（`createState` 在锁之前读时钟）原来只在单元一层：组合一层读时刻的只有按行寻址的写的锁强度测试，按项目寻址的 `createState` 不在其中。加锁的性质只在单元一层，是缺口，现在补上。
- `TestEachStateWriteChangesItsRowsAlone`（Task 9）加三行：Backlog 给它自己的组（`g-update-same-group` 原来只在单元一层，清扫 24 的例外一半）、在分诊组建状态、把 Done 改到分诊组（`t-triage-allowed` 原来只在单元一层和 P6，清扫 28）：两个请求体在契约的枚举之外，服务端照契约回答 422 `group not_allowed`，测试不核对请求、照旧核对回答和它的问题（没有这条规则时组是 `invalid_format`，同样 422，所以问题也要核对）。
- 故事 P6（Task 10）：改颜色、拖动各读一遍（`s26-us-sequence-zero`：没给的 `sequence` 写成 0，原来被之后的拖动盖住）；In Progress 由 ann 删除，它最后由 admin 写（`s14-ds-keeps-writer`，清扫 43）；Backlog 改到有两个状态的 started 组（`g-update-new-group`：数要去的组，原来改到只有一个状态的组时答案碰巧相同）；删除了的 In Progress、Web 的分诊状态再写各 404（`s1-sb-deleted`、`s1-sb-triage`）；最后读 Elsewhere 的两个项目的状态（`s1-cd-project`：清掉每个项目的默认，清扫 49）。
- 契约（清扫 18）：少声明码的两个变异（`s18-delete-no-default`、`s18-lws-no-not-found`）第一次只改了模块的契约，`CheckResponse` 读的打包的契约没改，单元一层没有失败；改成两个都改，并加上组合一层（P5b 的同类在单元、组合两层）。

之后快照由 `build.py` 从 Task 3 起重建，块和生成物的表重新生成，最终原型的门禁重跑；这些变异在它们写的那一层重跑。

**逐 Task 复现**（`$M3TMP/p7atools/replay.py`、`replay_all.sh`，日志在 `replay-logs`）：从 `3f87fcd3` 的一份新副本开始，照 plan 的顺序应用 10 个 Task 的块并运行每个 Task 写明的命令（每个 Task 的定向测试、`make lint-go`、`make test`，有的 Task 另有生成、`-race`、前端检查和端到端）。每个 Task 之后 `make lint-go` 两段 `0 issues.`、`make test` 42 个 `ok`；Task 3 的 `make gen-go`、Task 4、6、7、8 的 `make gen` 之后生成物与快照没有差异，SHA-256 和行数与 plan 的表相同；Task 4、6、7、8、10 的 `make lint-web`（关键词守卫 5 个命中都有例外）、`make knip`、`make test-web` 通过；Task 1 的 `-count=3 -race`（P5b 的竞争和锁强度在共用路径上，32 秒）、Task 9 的 `-count=5 -race`（119 秒）通过，输出里没有 40P01；Task 10 的 `make e2e` 69 个故事中 68 个通过，失败的只是 S3 的 F4（`replay.py` 只接受这一个失败）。最后的树与原型逐个文件相同（`treediff.mjs`：3,102 个文件，0 处差异）；`planapply.mjs` 从基线核对 plan 的 303 个块（272 处替换、27 个新文件、3 个整文件、1 个删除）都放得上；复现的副本最后 `make gen-check` 通过。共 1,219 秒，没有重跑。

前一次复现停在 Task 7 的 `make test`：`platform/jobs` 的 `TestRunnerWorksAPeriodicJobUntilStopped` 失败一次（周期任务的两次运行相隔 434 毫秒，测试要约 1 秒；P7a 不碰 `platform/jobs`，同一份代码在别的每次 `make test` 里都通过）。它还暴露了 plan 把三个前端检查写在一行 `Run:` 里，`replay.py` 不认这样的行，前几个 Task 的前端检查没有在复现里运行。plan 改成每个检查一行，`replay.py` 照简报的规则在 `make test` 失败时重跑一次并记下第一次的失败，之后从新的副本整个重做，就是上一段的结果。

**最终的原型**（`gates.sh`）：`make gen` 之后生成物没有差异；`make lint-go` 两段 `0 issues.`；`make test` 42 个 `ok`；`make lint-web`（关键词守卫 5 个命中都有例外，54 个任务）；`make knip`；`make test-web`（16 个任务）；`make e2e` 69 个故事中 68 个通过，P6、W11 在其中；S3 因原型不是仓库的检出、构建没有提交号而失败（P4b 的 F4，读 `commit`，与 P7a 无关，P6 的原型同样）。P7a 没有迁移。

**矩阵**：`TestPermissionMatrix -v` 709 格（P6 结束时 532 格，P7a 的 19 行加 177 格），1.51 秒，全部通过。

**交错和竞争**（`race5.sh`）：`go test -count=5 -race -v -run '…' ./internal/bootstrap/` 跑 12 个测试：交错 10 和一组最后两个状态的删除、改组（`TestStateWritesOnOneProjectSerialize`，两种顺序），按行寻址的写在等锁期间的变化和锁的强度（`TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`，项目成员关系的三个写和状态的写都在共用路径上），第一把锁（`TestEachWriteOnAProjectSharesItsWorkspaceFirst`），每个状态的写只写它的行（`TestEachStateWriteChangesItsRowsAlone`），P5b 的七个交错（两位项目管理员同时离开、互相降级，离开与工作区成员关系的结束，移出与被移出的成员的项目、与被移出的管理员的删除，锁下的判定，角色按工作区角色封顶）：`ok`，147.8 秒，12 个测试各 5 次全部通过（470 个子测试）；输出里没有 40P01，没有数据竞争。

**五十类清扫**（brief 的缺陷类别；每个变异一个 `go test -overlay` 或 `go build -overlay`，树不动，Go 以外的文件（契约和打包的契约）原地改、跑完复原；`mutlevels.py` 在变异写的每一层各跑一次：单元（`project` 的 `app`、`domain`、`adapter/http`，`access/domain`）、存储（`project/adapter/postgres`）、组合（`bootstrap`）、端到端（单独运行的故事 P6、W11）；"层"是它被发现的每一层）：

| 清扫 | 大小 | 结果 | 层 |
|---|---|---|---|
| 1 每个 SQL 谓词 | 十条语句的 41 个谓词和次序（`ClearDefaultState` 3、`CountGroupStates` 3、`DeleteState` 4、`GreatestSequence` 4（含取最小的）、`ListStates` 6、`ListWorkspaceStates` 11、`SetDefaultState` 4、`StateByID` 3、`UpdateState` 3），每个去掉，参数照旧绑定；P6 的故事跑到的那些另在 P6 单独运行时跑 | 41/41 个变异被发现；存储一层每个都失败（每个谓词各由一行决定；两种行序见 2.5）；只在存储一层的见下面"组合一层等价" | 存储；组合；端到端 |
| 2 端口调用的错误 | 六个用例的每个端口调用注入失败（`Test…ReturnsEachFailure`、`TestDeleteStateChecksTheReadAfterIt`），12 个吞掉或改答的变异 | 12/12 个变异被发现；调用记录到失败的那一步为止、之后不运行；真实的存储在这些步骤不失败，按性质只在单元一层 | 单元 |
| 3 写不动的行 | 四个写的存储测试比较被写的行之外每一行每一列（`tableRows`、`columns`）；3 个变异 | 3/3 个变异被发现 | 存储；组合；端到端 |
| 4 组合根的接线 | `project.New` 接上的六个用例：停住的时钟 4 个、`mark-default` 调用删除、`PATCH` 丢掉组 | 6/6 个变异被发现 | 单元；组合；端到端 |
| 5 安全性质在真实环境上 | 六行规则各放宽或收紧一格（6 个），两个读不判定（2 个） | 8/8 个变异被发现；矩阵的格子 | 单元；组合；端到端 |
| 6 每句文档 | 每句说明、契约描述、故事标题、差异清单的行对照代码 | 改了的见"对第一次运行的复审"的清扫 6、18、50 一条 | 审阅 |
| 7 反例里没有随机 | P7a 的测试 | `TestCheckGroupKept` 的 map 改为切片；别处的表都是切片 | 审阅 |
| 8 决定所依赖的读在调用者的事务里 | 存储的七个方法各改为走池 | 7/7 个变异被发现 | 存储；组合 |
| 9 锁顺序在真实环境上 | 共用路径的先后（先工作区、后项目；锁前读、锁下重读、判定） | 3/3 个变异被发现；最先锁工作区的测试和锁强度的测试，`NOWAIT` 和 `lockOn` | 单元；组合 |
| 10 承重的种子行有前提 | 矩阵的状态种子（每个项目恰好七个状态，默认 Backlog、分诊 Triage）、`stateWorld`、`memberWorld` 的 QA 和 Ops 的 Cancelled | 各有前提：矩阵的 `preconditions`、`worldStanding`、存储测试的 `ctid` 核对 | 审阅 |
| 11 每条拒绝路径 | 没有调用者什么都不读；判定失败原样返回；不能写的人对无效的目标得到 403（矩阵"名称已占用"、"组里最后一个"、"默认"各行的非管理员列） | 每个用例的拒绝表第一行；矩阵 | 审阅 |
| 12 与结束、删除的竞争（组合） | 状态的三个写各五种：这一行删除、移到 Ops，调用者的成员关系结束，Web 删除，acme 删除 | 3/3 个变异被发现；答 404，不答 403；什么都不改；等待由探测确认（2.10） | 单元；组合 |
| 13 锁强度在组合一层看得到 | `lockOn` 读出每一步的锁：项目 N、工作区 S、按行寻址的写不锁项目以外的行 | 2/2 个变异被发现 | 组合 |
| 14 每个"由谁"可以失败 | 存储的写之前由 maker 写过；组合的盖戳先由 bob 写过（`bobs`）；4 个保留写者的变异 | 4/4 个变异被发现 | 存储；组合；端到端 |
| 15 标题的每个说法都有展示 | P6、W11 的标题逐句 | 每一句都有步骤：P6 的"each changing nothing"由拒绝之后的 `expectStates` | 审阅 |
| 16 单元假对象的回答顺序 | 假实现的列表、`GreatestSequence` | 改按名称，测试先核对不是任何排序给得出的（"对第一次运行的复审"） | 审阅 |
| 17 判定之前的代价有界 | 每个写在判定之前：领域的检查（不读库）、一行的读和三把锁；读：一次查找 | 有界 | 审阅 |
| 18 契约描述只说代码做的 | 六个操作的描述和 `State` 的字段；3 个变异（少声明、多声明） | 3/3 个变异被发现；`State`、`State.default` 不说工作项（第 3 节第 7 条） | 单元；组合 |
| 19 每个新的存储方法有失败测试 | 九个方法，11 个变异（`MarkDefaultState` 两条语句各一个，同名的唯一冲突答成内部错误） | 11/11 个变异被发现；`s19-set` 原来活下来：取消的 context 让第一条语句失败；加 `TestMarkDefaultStateAnswersTheFailureOfItsSecondStatement` | 存储；组合 |
| 20 相关谓词的集合形式、"别的"算进自己 | P7a 的语句没有相关子查询，也没有"别的" | 不适用 | 审阅 |
| 21 拒绝和失败钉住第一个 `*shared.Error` | 每个拒绝表用 `outcome.check`（`sameOutcome`）；默认、组里最后一个的 409 各包在 404、403 之后（2 个） | 2/2 个变异被发现 | 单元；组合；端到端 |
| 22 端口的回答对得上所问 | 路径的三个核对（测一次，裁定 S6）、两个写的回答 | 4/4 个变异被发现；按性质只在单元一层（第 3 节第 1 条） | 单元 |
| 23 组合的夹具跨第二个工作区、第二个项目 | 矩阵（acme、gone、other）、`memberWorld`（acme 的 Web、Ops，beta 的 Lab） | 2/2 个变异被发现；加强：`TestListingWorkspaceStatesKeepsToItsWorkspace`，Ops 的 Cancelled 在 100000（"清扫之后的加强"） | 存储；组合 |
| 24 每条规则的每一半在组合一层有反例 | 一组至少一个（删除、改组；改到同一组、改进别的组不数）、默认（删除、设为默认）、分诊（每个操作）、`sequence` | 17/17 个变异被发现 | 单元；存储；组合；端到端 |
| 25 判定和执行在同一把锁下 | 改组在锁前、在另一个事务里数组 | 1/1 个变异被发现；交错的门在检查和写之间 | 组合 |
| 26 "不变"用不会碰巧得出的值 | `UpdateState` 没给的 `sequence` 写成 0、组写成 backlog；Ops 的 Cancelled 在 100000 | 2/2 个变异被发现 | 存储；组合；端到端 |
| 27 故事里的"不动"在动作之前读、之后核对 | P6 每一步之前之后 `expectStates`；W11 的五张表 |  | 审阅 |
| 28 只在端到端被发现的性质另有 Go 的测试 | 变异表里只在端到端失败的 | 没有：在故事里被发现的 52 个（P6 51 个、W11 1 个）都另在 Go 的某一层被发现（`mut_tables.py` 合并的结果） | 审阅 |
| 29 每个能等锁的等待都有期限 | 交错的 context 10 秒；竞争的 `receiveWithin`；探测 5、10 秒；`pgtest.Soon` |  | 审阅 |
| 30 每个输出、问题的细节和补救对每个收到它的调用者都成立 | 四个状态码的说明和补救 | 收到 409 的都是判定通过的项目管理员；"make another state the default first"对他成立 | 审阅 |
| 31 检查和执行之间的 gate 在检查的事务之外 | 交错的门在第一个写的写之后、提交之前 | 1/1 个变异被发现；`s25-move-counted-unlocked` 在门上失败 | 组合 |
| 32 每个探针有反例 | 等待从项目行挪到状态行（结果不变） | 1/1 个变异被发现；探测按表名，挪走的等待在探测上失败 | 组合 |
| 33 goroutine 上不调用 `t.Fatal` | P7a 的 `run(...)`、`sendInBackground` | 闭包里只调用用例、只送回答案 | 审阅 |
| 34 "没有替身"、"完成"的说法全包 grep | 完整性核对：契约的每个项目级的写在最先锁工作区的测试里有一行；`ShareWorkspaceByID` 只在 `Locks.lock`（第 3 节第 1 条） | 1/1 个变异被发现 | 组合 |
| 35 "由 X 写"的种子是规则允许的 | `memberWorld` 的 QA 和 Ops 的 Cancelled 由 alice（两个项目的管理员）；盖戳由 bob（Web 的管理员） | 都是规则允许的 | 审阅 |
| 36 别的操作或两步到达同样的结果 | 默认、一组至少一个、分诊的排除 | 没有绕过（第 3 节第 9 条） | 审阅 |
| 37 归档项目、删除的工作区、停用的调用者 | 每个写 | 第 3 节第 11 条 | 审阅 |
| 38 故事的每个拒绝之前先读 | P6 的五个拒绝；W11 的四格 | 之前 `expectStates`、`tables()` | 审阅 |
| 39 测试里的排序与排序规则无关 | `expectStates`（`sequence`，再按名称 `COLLATE "C"`）；故事按 id 比较 |  | 审阅 |
| 40 每个交错的结果写明每个结束由谁 | 交错 10 和一组最后两个：两个写都由 bob；结果读出每个状态的组、`sequence`、默认 | 没有成员关系的结束；写者由盖戳的测试钉住 | 审阅 |
| 41 每个写多行的语句与同一些行的其他写者 | `ClearDefaultState` 和状态的其余写者 | 父行的锁排除它们，约定五的例外不适用（第 3 节第 5 条） | 审阅 |
| 42 锁语句之后的写只写锁住的行 | P7a 没有先锁一批行再写的语句 | 不适用 | 审阅 |
| 43 每个"由 X"可以失败 | 同清扫 14 | 4/4 个变异被发现 | 存储；组合；端到端 |
| 44 "保留"、"还有"读状态 | "一组还有"读未删除的；"恰好一个默认"读 `default` 和未删除；`TestMarkDefaultStateRefuses` 在回滚之前读默认 |  | 审阅 |
| 45 没有不会失败的断言 | 路径的核对在各用例里的重复（裁定 S6）；`TestMarkDefaultStateRefuses` 的"第一条已运行" | 删去重复，改为读得出的核对 | 审阅 |
| 46 测试一侧的等待都有期限 | 持有事务时的语句、`Begin`、连接池的等待 | 持有另一个事务时的新语句都在 `pgtest.Soon` 的期限上：存储一层的 `TestMarkDefaultStateAnswersTheFailureOfItsSecondStatement`，组合一层的交错、竞争和锁的测试；不持有事务的测试照 `Soon` 的说明不用它 | 审阅 |
| 47 连续的探测在同一张表上要指名 | P7a 的五处探测 | 连续的探测各在不同的表上 | 审阅 |
| 48 堆序或索引序的夹具有前提 | 两个列表的存储测试 | 2/2 个变异被发现；`ctid` 前提（"对第一次运行的复审"） | 存储；组合；端到端 |
| 49 组合的夹具看得到过宽的写 | 状态的四个写 | 4/4 个变异被发现；`TestEachStateWriteChangesItsRowsAlone`；交错核对 Ops、Lab | 存储；组合；端到端 |
| 50 每句注释和文档对照它说的代码 | 每个 Task 末尾 | 见清扫 6；本 spec 第 3 节第 4 条 | 审阅 |

**只在单元一层被发现的变异（按性质）**：路径和两个写的键的核对（`s22-*` 4 个，第 3 节第 1 条）；端口失败的 12 个（`s2-*`：真实的存储在这些步骤不失败，存储方法自己的失败由清扫 19 在存储一层核对）；`q-first`（项目没有非分诊状态时的 65535：每一组至少留一个状态、建项目建出全部五组，组合出的 app 里到不了）；`g-delete-nothing-default`（删了 0 行时不重读、一律答默认：项目的 N 之下守卫只会跳过默认状态，重读与否答案相同；重读要的是锁下的一致，单元一层钉住）；`g-mark-nothing-ok`（用例在第二条语句写 0 行时照样提交：组合一层到不了，第 3 节第 4 条）；`s18-update-extra-default`（多声明、从不回答的码：只有 `apitest.Main` 的覆盖核对看得出，P5b 的同类也只在单元一层）。

**只在存储一层（按性质）**：`g-mark-n`（存储方法不看第二条语句写了几行：同上，组合一层到不了）；`s19-*` 10 个（存储方法自己的失败，清扫 19）。

**组合一层等价、只在存储一层的谓词**（依据组合出的 app 的不变式）：(a) 状态的写的 id 来自项目的 N 之下的 `StateByID`（不回答分诊的、已删除的），之后没有别的写能改它：`s1-ds-triage`、`s1-ds-deleted`、`s1-sd-triage`、`s1-sd-deleted`、`s1-us-triage`、`s1-us-deleted`（语句自己的守卫是纵深防御，3.17）；(b) 用例给 `SetDefaultState` 的项目就是它锁下读到的这一行的：`s1-sd-project`；(c) 未删除的项目没有已删除的默认状态（默认不能删除，项目连同全部状态一起删除）：`s1-cd-deleted`；(d) 删除项目在同一个事务里删除它的成员关系和状态（P4a 的 `deleteProjects`）：`s1-lw-pdeleted`。`s3-us-created`（`UpdateState` 改写 `created_at`）在存储一层（每列比较）：审计列，组合一层读写者和时刻（盖戳），不读 `created_at`，与 P4b、P5b 的写同一类。

**故事 P6 看不到的**：带 P6 一层的 68 个变异，故事发现 51 个；其余 17 个都在 Go 的某一层被发现：上面组合一层等价的 10 个；`s1-gs-project`、`s1-gs-deleted`（Review 建在 Web 时，别的项目的状态的 `sequence` 不比 Web 的大，Web 没有已删除的状态；In Progress 删除之后 Web 不再建状态）；`s1-lw-workspace`、`s1-lw-active`、`s1-lw-mdeleted`（ann 没有别的工作区的项目的、已结束的、已删除的项目成员关系）；`s3-us-keeps-moment`（故事只核对删除的时刻等于最后一次写的时刻，不读改的时刻）；`p-no-reread`（故事里没有并发）。后七个都在组合一层被发现。

**变异表（按缺陷类别）**：变异的定义在 `$M3TMP/p7atools/mutants_p7a.py`（135 个），分层的结果在 `p7atools/logs`（`stage-unit.json`、`stage-store.json`、`stage-composed.json`、`stage-e2e-P6.json`、`stage-e2e-W11.json`，加强之后的重跑 `rerun-unit.json`（两个契约变异）、`rerun-store.json`（`s19-set`）、`rerun-composed.json`（Go 测试加强的八个；`t-triage-allowed` 那时活下来，核对问题之后在 `rerun-composed-b.json`）、`rerun-composed-c.json`（两个契约变异）、`rerun-e2e-P6.json`（`s1-lw-order`：故事 P6 加强之后整层重跑时，测试容器两次没起来，`mutlevels.py` 把"既没过也没失败"记成了活下来；改成重试、六次都起不来就停下整个运行，之后重跑在故事里被发现），每个变异的 `mutant-*.log`），`mut_tables.py` 把重跑按层合进去，写出 plan 每个 Task 的变异表和下表。135 个变异，135 个被发现；只在单元一层被发现的 20 个，都是按性质的（第 3 节）。

| 缺陷类别 | 变异 | 结果 | 失败的测试 | 层 |
|---|---|---|---|---|
| 清扫 1：SQL 谓词 | `s1-cd-project`、`s1-cd-default`、`s1-cd-deleted`、`s1-cg-project`、`s1-cg-group`、`s1-cg-deleted`、`s1-ds-id`、`s1-ds-default`、`s1-ds-triage`、`s1-ds-deleted`、`s1-gs-project`、`s1-gs-triage`、`s1-gs-deleted`、`s1-gs-asc`、`s1-ls-join`、`s1-ls-project`、`s1-ls-triage`、`s1-ls-deleted`、`s1-ls-archived`、`s1-ls-order`、`s1-lw-join-p`、`s1-lw-join-m`、`s1-lw-workspace`、`s1-lw-pdeleted`、`s1-lw-archived`、`s1-lw-member`、`s1-lw-active`、`s1-lw-mdeleted`、`s1-lw-triage`、`s1-lw-sdeleted`、`s1-lw-order`、`s1-sd-id`、`s1-sd-project`、`s1-sd-triage`、`s1-sd-deleted`、`s1-sb-id`、`s1-sb-triage`、`s1-sb-deleted`、`s1-us-id`、`s1-us-triage`、`s1-us-deleted` | 41/41 | `TestMarkDefaultState`、`TestMarkDefaultStateRefuses`、`TestEachStateWriteChangesItsRowsAlone`、`TestStateWritesOnOneProjectSerialize`、P6、`TestCountGroupStates`、`TestPermissionMatrix`、等 15 个 | 存储；组合；端到端 |
| 清扫 3：写不动的列 | `s3-us-created`、`s3-us-keeps-moment`、`s3-ds-keeps-moment` | 3/3 | `TestUpdateState`、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`、`TestTheWritesOnAProjectStampTheirRequest`、`TestDeleteState`、P6 | 存储；组合；端到端 |
| 清扫 14：由谁 | `s14-us-keeps-writer`、`s14-ds-keeps-writer`、`s14-cd-keeps-writer`、`s14-sd-keeps-writer` | 4/4 | `TestUpdateState`、`TestTheWritesOnAProjectStampTheirRequest`、P6、`TestDeleteState`、`TestMarkDefaultState` | 存储；组合；端到端 |
| 清扫 26：不会碰巧的值 | `s26-us-sequence-zero`、`s26-us-group-backlog` | 2/2 | `TestUpdateState`、`TestStateWritesOnOneProjectSerialize`、P6、`TestPermissionMatrix`、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection` | 存储；组合；端到端 |
| 清扫 8：事务的连接 | `s8-create`、`s8-byid`、`s8-greatest`、`s8-update`、`s8-count`、`s8-delete`、`s8-mark` | 7/7 | `TestADeactivationAndACreationHeLeadsSerialize`、`TestADeactivationDeletesTheInvitationsOfAWorkspaceItEmpties`、`TestADeactivationEndsEveryMembership`、`TestADeactivationFindsWhatChangedMeanwhile`、`TestADeactivationReadsTheAddressUnderItsLock`、`TestADeactivationRefusedAtItsCommitChangesNothing`、`TestAGrowthRefusedAtItsCommitLeavesNoRow`、等 40 个 | 存储；组合 |
| 清扫 19：存储方法的失败 | `s19-create`、`s19-byid`、`s19-list`、`s19-greatest`、`s19-update`、`s19-count`、`s19-delete`、`s19-clear`、`s19-set`、`s19-lws`、`s19-name-taken-internal` | 11/11 | `TestAFailedWriteIsAnError`、`TestCreateStateBreakingAnotherConstraintIsInternal`、`TestCreateStateNameTaken`、`TestCreateStatesStopsAtTheFirstFailure`、`TestAFailedReadIsAnErrorNotAnAnswer`、`TestUpdateStateNameTaken`、`TestUpdateStateWritesNoDeletedOrTriageState`、等 2 个 | 存储；组合 |
| 清扫 2：端口的错误 | `s2-create-greatest`、`s2-create-create`、`s2-update-count`、`s2-update-update`、`s2-delete-delete`、`s2-delete-count`、`s2-delete-reread`、`s2-mark`、`s2-list`、`s2-lws`、`s2-path-reread-404`、`s2-path-lock-404` | 12/12 | `TestCreateStateReturnsEachFailure`、`TestCreateStateRefuses`、`TestUpdateStateReturnsEachFailure`、`TestUpdateStateRefuses`、`TestDeleteStateReturnsEachFailure`、`TestDeleteStateChecksTheReadAfterIt`、`TestDeleteStateRefuses`、等 10 个 | 单元 |
| 清扫 22：回答对得上所问（裁定 S6） | `s22-workspace-key`、`s22-row-id`、`s22-created-id`、`s22-updated-id` | 4/4 | `TestLocksCheckEachAnswerAgainstItsKey`、`TestDeleteStateChecksTheReadAfterIt`、`TestCreateStateRefuses`、`TestUpdateStateRefuses` | 单元 |
| P7a：共用取锁路径 | `p-no-reread`、`p-project-unchecked`、`p-decide-first`、`p-reread-before-lock`、`p-project-before-workspace`、`p-rows-share-project`、`p-create-share-project`、`p-row-404-forbidden` | 8/8 | `TestDeleteState`、`TestDeleteStateChecksTheReadAfterIt`、`TestDeleteStateRefuses`、`TestDeleteStateReturnsEachFailure`、`TestEachWriteReadsTheClockUnderItsLock`、`TestLocksCheckEachAnswerAgainstItsKey`、`TestMarkDefaultState`、等 26 个 | 单元；组合 |
| P7a：时钟在锁下 | `c-create-early`、`c-update-early`、`c-delete-early`、`c-mark-early` | 4/4 | `TestCreateState`、`TestCreateStateRefuses`、`TestCreateStateReturnsEachFailure`、`TestEachWriteReadsTheClockUnderItsLock`、`TestStateWritesOnOneProjectSerialize`、`TestUpdateState`、`TestUpdateStateRefuses`、等 9 个 | 单元；组合 |
| P7a：组和默认状态的规则、守卫（清扫 36） | `g-group-kept-off`、`g-update-unchecked`、`g-update-same-group`、`g-update-new-group`、`g-update-n`、`g-delete-unchecked`、`g-delete-default-answers-404`、`g-delete-nothing-default`、`g-mark-nothing-ok`、`g-mark-no-clear`、`g-mark-set-first`、`g-mark-n`、`g-delete-n` | 13/13 | `TestCheckGroupKept`、`TestDeleteStateRefuses`、`TestUpdateStateRefuses`、`TestPermissionMatrix`、`TestStateWritesOnOneProjectSerialize`、P6、`TestEachWriteReadsTheClockUnderItsLock`、等 10 个 | 单元；存储；组合；端到端 |
| P7a：分诊组 | `t-triage-allowed` | 1/1 | `TestCheckNewStateReportsEveryField`、`TestCheckStatePatch`、`TestCreateStateRefuses`、`TestUpdateStateRefuses`、`TestEachStateWriteChangesItsRowsAlone`、P6 | 单元；组合；端到端 |
| P7a：`sequence` | `q-step`、`q-first`、`q-greatest-ignored` | 3/3 | `TestCreateState`、`TestCreateStateRefuses`、`TestCreateStateReturnsEachFailure`、`TestEachWriteReadsTheClockUnderItsLock`、`TestSequenceAfter`、`TestPermissionMatrix`、`TestStateWritesOnOneProjectSerialize`、等 1 个 | 单元；组合；端到端 |
| 清扫 21：第一个问题 | `s21-default-wrapped`、`s21-last-wrapped` | 2/2 | `TestDeleteStateRefuses`、`TestPermissionMatrix`、`TestStateWritesOnOneProjectSerialize`、P6、`TestCheckGroupKept`、`TestUpdateStateRefuses` | 单元；组合；端到端 |
| 清扫 25：判定和执行在同一把锁下 | `s25-move-counted-unlocked` | 1/1 | `TestStateWritesOnOneProjectSerialize`、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection` | 组合 |
| 清扫 32：探针的反例 | `s32-wait-on-state` | 1/1 | `TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`、`TestEachWriteOnAProjectSharesItsWorkspaceFirst`、`TestStateWritesOnOneProjectSerialize` | 组合 |
| 清扫 4：接线 | `s4-create-frozen-clock`、`s4-update-frozen-clock`、`s4-delete-frozen-clock`、`s4-mark-frozen-clock`、`s4-mark-serves-delete`、`s4-update-drops-group` | 6/6 | `TestTheWritesOnAProjectStampTheirRequest`、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`、`TestMarkDefaultState`、`TestEachWriteOnAProjectSharesItsWorkspaceFirst`、`TestPermissionMatrix`、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`、P6、等 1 个 | 单元；组合；端到端 |
| 清扫 5：规则表和判定 | `s5-update-members`、`s5-create-guests`、`s5-delete-members`、`s5-mark-members`、`s5-list-no-guests`、`s5-lws-no-guests`、`s5-list-undecided`、`s5-lws-undecided` | 8/8 | `TestEveryRuleDecidesItsCells`、`TestPermissionMatrix`、P6、W11、`TestListingWorkspaceStatesIsListingEachProjects`、`TestListStates`、`TestListStatesRefuses`、等 2 个 | 单元；组合；端到端 |
| 清扫 18：契约 | `s18-delete-no-default`、`s18-update-extra-default`、`s18-lws-no-not-found` | 3/3 | `TestDeleteState`、`TestPermissionMatrix`、`apitest.Main`、`TestListWorkspaceStates`、`TestListingWorkspaceStatesIsListingEachProjects`、`TestListingWorkspaceStatesKeepsToItsWorkspace` | 单元；组合 |
| 清扫 34：完整性的核对 | `s34-first-lock-row` | 1/1 | `TestEachWriteOnAProjectSharesItsWorkspaceFirst` | 组合 |

**Phase 的大小**（裁定 S3：约 16 个任务、每个任务约 1,500 行以内）：10 个任务，在约 16 个之内。plan 里每个 Task 一节的行数（它的块、生成物的表、变异表和命令，`assemble.py` 量的）：Task 1 585、Task 2 450、Task 3 1,300、Task 4 1,246、Task 5 632、Task 6 1,137、Task 7 1,277、Task 8 1,396、Task 9 791、Task 10 619，最大的 Task 8 在约 1,500 行之内；全文 9,514 行（`wc -l`）。其中块（`mkblocks.py`）最大的是 Task 8 的 1,246 行；第一次运行的 T4 单是块就有 1,518 行，这次分成 Task 5、6。
