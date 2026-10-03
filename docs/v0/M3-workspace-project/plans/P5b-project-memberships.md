# M3/P5b 项目成员的角色、移出与离开 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 改项目成员的角色（`updateProjectMember`，3.5 的相对规则：按集合，工作区管理员的例外，工作区访客的上限）、移出项目成员（`removeProjectMember`）、离开项目（`leaveProject`，3.7 规则 1 的项目一侧：唯一的有效管理员不能离开，哪怕只有他一人）。三个写是第一批按资源寻址（`/project-members/{project_member_id}`）和项目级的写之一，经 P4b 的共用取锁路径 `Locks` 的新分支：不加锁地读成员关系 → 工作区 S → 改角色时它的成员在工作区的成员关系 S → 项目 N → 锁下重读、确认仍属这个项目 → 判定。三个操作各带操作名、规则行、矩阵行（108 格，矩阵共 523 格），各在每个项目级的写最先锁工作区的测试里有一行（含资源行的探测）；等锁期间的变化都答 404、一行不改；锁的强度在组合一层看得到；交错 1 的项目一侧、离开与 P5a 的结束两种顺序都串行；矩阵的 P-前和 `partingStates` 的已结束一行改由存储写出；故事 P5 的接口版本通过，W7 的"降级含已离开的项目"由它展示。

**Architecture:** 只动 `project` 模块、`access` 的规则表、`workspace` 的 HTTP 测试里 `project.sole_admin` 的文字、`bootstrap` 的测试和 e2e；不加迁移，不加跨模块的端口。`project`：存储（`MemberByID`、`UpdateMemberRole`、`EndMember`、`HasOtherAdmin`）；领域（`CheckMemberRole`、`CheckRoleChange`、`CheckRemoval`，三个码 `project.member_not_found`、`project.own_membership`、`project.role_too_high`；`project.sole_admin` 的说明为规则 1、2 两条措辞）；app（`Locks.lockMemberAndDecide`；按用例分的端口 `MemberFinder`、`MemberRoleChanger`、`MemberEnder`、`MemberLeaver`；三个用例）；HTTP（`api/modules/project.yaml` 的三个操作）。`access`：三条规则。`bootstrap`：组合出的相对规则、移出、离开，竞争、锁的强度、事务的连接、写入的时刻和写者，交错，矩阵。

**Tech Stack:** Go 1.27.1、pgx v5.11.0、sqlc v1.31.1（`CGO_ENABLED=0`）、oapi-codegen v2.8.0、goose v3.28.0、River v0.47.0、golangci-lint 2.13.2、PostgreSQL 18.6（testcontainers）；Node 24、pnpm 11.10.0、Playwright 1.63.0。

**Spec:** `docs/v0/M3-workspace-project/specs/P5b-project-memberships.md`（上级：`docs/v0/M3-workspace-project/M3-design.md`）

## Global Constraints

- **Go 版本和依赖**：本 plan 不执行 `go get`，不加任何 Go 模块或 npm 包。`server/go.mod` 和 `server/tools/go.mod` 保持 `go 1.27` 和 `toolchain go1.27.1`（golangci-lint 2.13.2 由 go1.27.0 构建，`go` 行更高就拒绝运行）。每个 Task 提交前执行 `grep -n "^go \|^toolchain" server/go.mod server/tools/go.mod`，四行必须是 `go 1.27` 和 `toolchain go1.27.1`；`git diff --stat ebfd2237 -- server/go.mod server/go.sum server/tools pnpm-lock.yaml` 在本 plan 的任何时刻都没有输出。
- **每个 Task 提交前**：`make lint-go` 两段都输出 `0 issues.`，`make test` 全部通过（没有 `FAIL`）。
- **生成的文件不手写、不从本 plan 复制**：有生成物的 Task（1、4、6、7）执行 Task 中的生成命令（只动了 sqlc 的 Task 1 执行 `make gen-go`，改了接口描述的 Task 4、6、7 执行 `make gen`），用 `shasum -a 256` 和 `wc -l` 核对表中的 SHA-256 和行数，对不上时停下来：说明某个输入与本 plan 不一致。这些 Task **提交之后**执行 `make gen-check`（它用 `git status` 判断生成物是否已提交，提交之前必然报差异）。
- **前端检查**：改了接口描述、前端文案或 `e2e/` 的 Task（4、6、7、11）另执行 `make lint-web`、`make knip`、`make test-web`；声明新错误码的 Task 4（`project.member_not_found`、`project.own_membership`、`project.role_too_high`）在同一个 Task 里把码加进 `PROBLEM_MESSAGES` 和两份 `auth.json`（M3 设计 12 节约束 4），Task 7 改 `project.sole_admin` 的两份文案；Task 11 执行 `make e2e`。
- **容器**：`make test` 和 `make e2e` 用自己的 testcontainers；机器忙时偶尔起不来，等 Docker 空闲之后重跑一次再当作失败。容器测试一次只跑一套。开发库 `nerve-dev-db-1` 可以用，但不要停止或重建它，不要执行 `make dev-db-down`、`make dev-db-reset`。不要碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`、`nervewiki-*`）。
- **git**：每次 Bash 调用只执行一个 git 命令，不用 `;`、`&&`、`|` 串联 git；不用 `git -C`、`stash`、`clean`、`reset --hard`。`cd` 不与别的命令组合，只读的命令也不行。不碰 `plane/`、`refer/`。
- **安装**：除了 Docker、Go、Node 不做任何全局安装；不执行 `corepack enable`（pnpm 已在 PATH 上）。
- **规则**：不写 `init()`，不用全局可变状态，构造函数显式传入依赖；不建 `utils`、`common`、`helpers` 包；一个文件只做一件事，不超过约 400 行；依赖只能向内；模块之间不互相导入，不跨模块的表 JOIN；模块的 SQL 只经 sqlc；角色只按集合判断（规则表；`roleOrder` 里的位置；`role = 20` 只出现在"管理员"这一个集合的查询里），不按大小比较；不留没有使用者的代码。**例外**：接口描述按模块一个文件（M0-P3 交接 5），`api/modules/project.yaml`（862 行）不受约 400 行的限制。本 plan 的其余文件都在约 400 行以内（最终原型上量的）：最长的是 `bootstrap/interleaving_growth_test.go`（404 行，P5a 留下 405 行，本 plan 只改 `growthRace` 的一句注释和结束的一步）、`bootstrap/permission_matrix_test.go`（389 行）、`project/app/ports.go`（352 行）、`bootstrap/project_write_locks_test.go`（347 行）、`bootstrap/permission_matrix_columns_test.go`（330 行）、`bootstrap/permission_matrix_seed_test.go`（321 行）、`bootstrap/project_writes_test.go`（302 行）。
- **注释**：Go、TS 代码、SQL 查询和接口描述用英文；中文文档照本 plan 原样。
- **代码块**：每个改动都写成四个反引号围起来的块，块的第一行写明种类和路径，照原样使用（原型中逐字节运行过）：
  - ````` ````file <路径> ````` 新文件，块的内容加一个结尾换行就是整个文件；
  - ````` ````whole <路径> ````` 已有文件的完整新内容（同样加结尾换行）；
  - ````` ````old <路径> ````` 与紧跟着的 ````` ````new <路径> `````：`old` 的文字在文件当前版本中恰好出现一次，把它换成 `new` 的文字；
  - ````` ````delete <路径> ````` 删除这个文件（块是空的）。

  一个文件的几个块按出现的顺序依次应用。拼 plan 的脚本已从 `ebfd2237` 起按顺序核对过全部块：每个 `old` 恰好出现一次（在它之前的块应用之后的文件中），每个新文件原来不存在，逐 Task 应用之后的文件与原型逐字节相同（spec 附录 A）。可以用 `node <planapply.mjs> <本 plan> apply <仓库根> <n>` 写入第 n 个 Task 的块，也可以手工照抄。
- **过渡版本**：一些文件先在较早的 Task 写成过渡版本，较晚的 Task 再修改（`api/modules/project.yaml`、`api/openapi.yaml`、`project/domain/actions.go`、`errors.go`、`project/app/ports.go`、`lock.go`、`fakes_member_test.go`、`clock_test.go`、`project/adapter/http/handler.go`、`handler_test.go`、`members.go`、`member_writes_test.go`、`project/module.go`、`access/domain/rules.go`、`rules_test.go`、前端文案、矩阵的文件、`bootstrap/project_write_locks_test.go`、`bootstrap/project_writes_test.go`、生成物）。每个过渡版本都在逐 Task 复现中运行过。
- **变异**：每个 Task 末尾的"变异"表列出：把代码改坏的方式、必须因此失败的测试和它所在的层（单元：假实现；存储：真实数据库；组合：`bootstrap` 组合出的 app 或模块；端到端：单独运行的故事 P5、W7）。它们在最终的原型上逐个跑过（`$M3TMP/p5btools/mutants_p5b.py`、`mutants_p5b_more.py`、`mutants_p5b_extra.py`，预检之后的 `mutants_p5b_amend.py`，由 `mutlevels.py` 在每一层各跑一次，spec 附录 A：113 个，112 个被发现，1 个等价）；实现者可以照表抽查，改坏之后必须恢复。表中"（Task n 起）"标出的测试在较晚的 Task 才有：这一行的变异从那个 Task 起才被发现。**安全或加锁的性质只由单元一层发现的，算缺口**（brief 的缺陷类别）；表中每一条这类性质都另有存储、组合或端到端一层的测试，例外写在 spec 第 3 节。
- **评审敏感**（M3 设计 12 节约束 3）：第一批按资源寻址的项目级的写（`Locks` 的新分支：重读确认项目、判定在锁之后、404 不是 403），3.5 的相对规则，项目一侧的规则 1 和交错 1，P5b 的写与 P5a 的结束不成环。谁能改角色、移出、离开是安全性质，唯一管理员的规则也是。改动这些测试、锁、规则之前，先照"变异"表确认它在所说的性质去掉之后失败。
- **提交**：提交信息用英文，末尾加一行：`Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明。

## 文件结构

路径相对于仓库根目录。"生成"表示由命令生成并提交；一个文件由多个 Task 修改时，"Task"一列都列出。

| 文件 | 职责 | Task |
|---|---|---|
| `server/internal/modules/project/adapter/postgres/queries/members.sql`（修改） | `MemberByID`、`UpdateMemberRole`、`EndMember`、`HasOtherAdmin` | 1 |
| `server/internal/modules/project/adapter/postgres/gen/members.sql.go`（生成） | | 1 |
| `server/internal/modules/project/adapter/postgres/membership.go`、`server/internal/modules/project/adapter/postgres/membership_test.go`；`server/internal/modules/project/adapter/postgres/failures_test.go`（修改） | 四个存储方法和它们的测试（另一个项目、另一个成员、已结束的、已删除的行各在两个行序下一列不动）；每个方法的失败原样返回 | 1 |
| `server/internal/modules/project/app/member_ports.go` | `ProjectMembership`；端口 `MemberFinder`、`MemberRoleChanger`、`MemberEnder`、`MemberLeaver` | 1 |
| `server/internal/modules/project/domain/membership.go`、`server/internal/modules/project/domain/membership_test.go` | `CheckMemberRole`、`CheckRoleChange`（3.5 的相对规则、工作区管理员的例外、工作区访客的上限）、`CheckRemoval`；按 `roleOrder` 的集合 | 2 |
| `server/internal/modules/project/domain/errors.go`（修改） | `ErrMemberNotFound`、`ErrOwnMembership`、`ErrRoleTooHigh`；`ErrSoleAdmin` 为规则 1、2 措辞（Task 7） | 2、7 |
| `server/internal/modules/project/domain/actions.go`（修改） | `project_member.update`、`project_member.remove`（Task 6）、`project.leave`（Task 7） | 3、4、6、7 |
| `server/internal/modules/project/app/lock.go`、`server/internal/modules/project/app/get_project.go`、`server/internal/modules/project/app/ports.go`（修改） | `Locks.lockMemberAndDecide`（按成员关系 id 寻址的写的取锁路径）；`decide` 带 404 的码；`ProjectLocks` 带 `MemberFinder` | 3 |
| `server/internal/modules/project/app/update_member.go`、`server/internal/modules/project/app/update_member_test.go` | `updateProjectMember` 的用例 | 3 |
| `server/internal/modules/project/app/fakes_test.go`、`server/internal/modules/project/app/fakes_write_test.go`、`server/internal/modules/project/app/fakes_create_test.go`（修改）；`server/internal/modules/project/app/fakes_member_test.go` | 假存储读、写一个成员关系；`outcome` 核对第一个问题和事务的回答；固定的成员关系 id | 3、6、7（`fakes_member_test.go`） |
| `server/internal/modules/project/app/clock_test.go`（修改） | 三个写在锁之后读一次时钟 | 3、6、7 |
| `api/modules/project.yaml`、`api/openapi.yaml`（修改） | 三个操作 | 4、6、7（`openapi.yaml`：4、7） |
| `api/dist/openapi.yaml`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`server/internal/modules/project/adapter/http/gen/bodyshape.gen.go`、`web/packages/api-client/src/schema.gen.ts`（生成） | | 4、6、7（`bodyshape.gen.go`：4） |
| `server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`（修改） | 三条规则和它们的格子 | 4、6、7 |
| `server/internal/modules/project/adapter/http/handler.go`、`server/internal/modules/project/adapter/http/members.go`、`server/internal/modules/project/adapter/http/handler_test.go`（修改）；`server/internal/modules/project/adapter/http/member_writes_test.go`；`server/internal/modules/project/module.go`（修改） | 三个 handler 和它们的测试；接线 | 4、6、7 |
| `web/apps/web/helpers/authentication.helper.ts`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json`（修改） | 三个新码的文案；`project.sole_admin` 的新说法（Task 7） | 4、7（`authentication.helper.ts`：4） |
| `server/internal/bootstrap/permission_matrix_seeded_test.go`、`server/internal/bootstrap/permission_matrix_seed_test.go`、`server/internal/bootstrap/permission_matrix_targets_test.go`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/bootstrap/permission_matrix_columns_test.go`（修改）；`server/internal/bootstrap/permission_matrix_membership_test.go` | 种下的项目成员关系按 id；`{project_member_id}` 的目标；改角色、移出、离开的矩阵行和前提；P-前和 WG- 的已结束改由存储写出（Task 10） | 4、6、7、10 |
| `server/internal/bootstrap/project_write_locks_test.go`（修改） | 每个项目级的写最先锁工作区：资源路径、写它的账户、资源行的探测；三个写各一行 | 4、6、7 |
| `server/internal/bootstrap/membership_world_test.go`、`server/internal/bootstrap/project_roles_test.go` | `memberWorld`（两个工作区、三个项目，经接口建）；组合出的相对规则 | 5 |
| `server/internal/bootstrap/project_removal_test.go` | `memberEnding`（一次结束或拒绝的核对）；组合出的移出和重新加入 | 6 |
| `server/internal/modules/project/app/remove_member.go`、`server/internal/modules/project/app/remove_member_test.go` | `removeProjectMember` 的用例 | 6 |
| `server/internal/modules/project/app/leave_project.go`、`server/internal/modules/project/app/leave_project_test.go` | `leaveProject` 的用例 | 7 |
| `server/internal/modules/workspace/adapter/http/members_test.go`（修改） | `project.sole_admin` 的新说法 | 7 |
| `server/internal/bootstrap/project_leaving_test.go` | 组合出的离开：规则 1，只有他一人也拒绝；已结束的管理员不算；规则看项目角色 | 7 |
| `server/internal/bootstrap/project_membership_races_test.go` | 等锁期间成员关系结束、删除、移到别的项目，调用者自己的结束，项目、工作区删除：404、一行不改；每把锁的顺序和强度 | 8 |
| `server/internal/bootstrap/project_membership_role_race_test.go` | 等锁期间成员关系被提拔为管理员、被删除：改角色、移出按锁下重读到的角色和成员关系判定（403 `project.role_too_high`；404 不是 403） | 8 |
| `server/internal/bootstrap/project_connection_test.go`、`server/internal/bootstrap/project_writes_test.go`（修改） | 三个写在事务的连接上；写入的时刻和写者；替身的注释（Task 10） | 8、10（`project_writes_test.go`） |
| `server/internal/bootstrap/interleaving_leaving_test.go` | 交错 1 的项目一侧；离开与 P5a 的移出两种顺序 | 9 |
| `server/internal/bootstrap/permission_matrix_members_test.go`、`server/internal/bootstrap/project_visibility_test.go`、`server/internal/bootstrap/project_members_test.go`、`server/internal/bootstrap/interleaving_growth_test.go`、`server/internal/bootstrap/interleaving_writes_test.go`（修改） | 替身换成 P5b 的写（接口或存储的语句）；只有写不出的状态留在 SQL，说明为什么 | 10 |
| `e2e/fixtures/assert/project.ts`（修改）；`e2e/stories/project/p5-project-members.spec.ts` | `expectMembers`；故事 P5 的接口版本（含 W7 的"降级含已离开的项目"） | 11 |

---

### Task 1: 项目成员关系的存储：按 id 读、改角色、结束、其余的管理员

**Files:**
- Create: `server/internal/modules/project/adapter/postgres/membership.go`、`server/internal/modules/project/adapter/postgres/membership_test.go`、`server/internal/modules/project/app/member_ports.go`
- Modify: `server/internal/modules/project/adapter/postgres/failures_test.go`、`server/internal/modules/project/adapter/postgres/queries/members.sql`
- Generate: `server/internal/modules/project/adapter/postgres/gen/members.sql.go`

**Interfaces:**
- Produces（spec 2.3，M3 设计 3.5、3.6 约定二、3.7 规则 1、4.3）：`project/app.ProjectMembership{ID, WorkspaceID, ProjectID, MemberID, Role, Active}`；按用例分的端口 `MemberFinder`（`MemberByID`）、`MemberRoleChanger`（`UpdateMemberRole`）、`MemberEnder`（`EndMember`）、`MemberLeaver`（`MemberEnder` 加 `HasOtherAdmin`）。`project/adapter/postgres.Store` 的四个方法，都在 `ctx` 带着的事务里执行：
  - `MemberByID(ctx, id) (ProjectMembership, found bool, error)`：未删除的成员关系，有效的或已结束的，带它的工作区、项目、成员、角色；已删除的、没有的，`found` 为假。写在锁之前不加锁地读它一次，在锁下再读一次（Task 3）。
  - `UpdateMemberRole(ctx, id, role, by, now) (domain.Member, error)`：有效、未删除的那一行改为 `role`，`updated_at = now`、`updated_by_id = by`，回答存下的行；已结束、已删除的是一个错误，不写。
  - `EndMember(ctx, projectID, userID, by, now) error`：这个账户在这个项目有效、未删除的成员关系 `is_active = false`，`updated_at = now`、`updated_by_id = by`，行留着、角色不变；不是恰好一行是一个错误（调用者在锁下读到它有效）。
  - `HasOtherAdmin(ctx, projectID, userID) (bool, error)`：这个项目除了他还有没有有效、未删除的管理员（`role = 20`，"管理员"这个集合）。
- 四条查询照原样写进 `queries/members.sql`（spec 2.3 有全文）。

**Tests:**（`adapter/postgres/membership_test.go`；存储测试的共同写法：被写的行除了写的列一列不动，其余每一行每一列不动，`membershipRows` 比较前后）
- `TestMemberByID`：alice 在 acme 的 Web 的有效的、bob 在 beta 的 Site 的已结束的成员关系各按 id 读出它自己（读第一行的实现对其中一个答错）；bob 在 Web 已删除的、没有的 id 都是 `found` 为假。
- `TestUpdateMemberRole`：bob 在 Web 的有效成员关系改为访客，由给的账户、在给的时刻，回答存下的行（id、项目、成员、新角色、建立时刻）；他在 Ops、beta 的 Site 的，他在 Web 已删除的（存在有效的之前或之后，两个行序），carol 在 Web 的，每一列不动；他在 Docs 已结束的、在 Web 已删除的各是一个错误，什么都不写。
- `TestEndMember`：bob 在 Web 的有效成员关系结束，由给的账户、在给的时刻，角色不变：他已删除的那一行先存时它是访客的，后存时是管理员的（把角色写成列的默认（访客的）、成员的、管理员的结束各在一种行序下失败，清扫 26）；旁边的行同上，每一列不动；再结束一次（已没有有效的）、结束他在 Docs（已结束）、Gone（只有一行已删除、仍是有效的）的，各是一个错误，什么都不写。
- `TestHasOtherAdmin`：每个情形一个 acme 的项目，另有一个项目有 bob 以外的有效管理员（只问别的项目的实现答错），bob 在问他时是那里的有效管理员（把他算进去的实现答错）：另一位有效管理员（bob 是管理员或成员）为真；只有他、另一个有效成员、另一个有效访客、另一位管理员已结束、已删除、没有成员为假。
- `adapter/postgres/failures_test.go`：`TestAFailedReadIsAnErrorNotAnAnswer` 加 `MemberByID`（不是"没有这个成员关系"）、`HasOtherAdmin`（不是"没有别的管理员"）；`TestAFailedWriteIsAnError` 加 `UpdateMemberRole`、`EndMember`（清扫 19）。

- [ ] **Step 1: 查询**

`server/internal/modules/project/adapter/postgres/queries/members.sql`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/queries/members.sql
WHERE workspace_id = sqlc.arg(workspace_id) AND member_id = sqlc.arg(member_id) AND NOT is_active AND deleted_at IS NULL;

````
````new server/internal/modules/project/adapter/postgres/queries/members.sql
WHERE workspace_id = sqlc.arg(workspace_id) AND member_id = sqlc.arg(member_id) AND NOT is_active AND deleted_at IS NULL;

-- name: MemberByID :one
-- A write on a project membership named by its id (M3 design 3.6 convention 2): the undeleted membership, active or
-- ended, read first without a lock for its project and the project's workspace, then again under their locks.
SELECT id, workspace_id, project_id, member_id, role, is_active
FROM project_members
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: UpdateMemberRole :one
-- updateProjectMember, under the project's FOR NO KEY UPDATE (M3 design 3.5, 3.6): the active membership's new role,
-- at the moment and by the account given. An ended or deleted one is not written.
UPDATE project_members
SET role = sqlc.arg(role), updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(updated_by)::uuid
WHERE id = sqlc.arg(id) AND is_active AND deleted_at IS NULL
RETURNING id, project_id, member_id, role, created_at;

-- name: EndMember :execrows
-- removeProjectMember and leaveProject, under the project's FOR NO KEY UPDATE (M3 design 3.5, 3.7): the account's
-- active membership of the project ends, at the moment and by the account given; the row stays, its role too.
UPDATE project_members
SET is_active = false, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(ended_by)::uuid
WHERE project_id = sqlc.arg(project_id) AND member_id = sqlc.arg(member_id) AND is_active AND deleted_at IS NULL;

-- name: HasOtherAdmin :one
-- leaveProject, under the project's FOR NO KEY UPDATE (M3 design 3.7 rule 1): whether the project has an active admin
-- other than the account.
SELECT EXISTS (SELECT 1 FROM project_members
               WHERE project_id = sqlc.arg(project_id) AND member_id <> sqlc.arg(member_id) AND role = 20 AND is_active
                 AND deleted_at IS NULL);

````

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `826d80358bd66c99a7763872affea86f7024651e5e4281744e8d603421cb797d` | 299 | `server/internal/modules/project/adapter/postgres/gen/members.sql.go` |

Run: `shasum -a 256 server/internal/modules/project/adapter/postgres/gen/members.sql.go`
Expected: 与上表相同。

- [ ] **Step 2: 端口、存储和测试**

`server/internal/modules/project/app/member_ports.go`（新文件，62 行）：

````file server/internal/modules/project/app/member_ports.go
package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The ports of the writes that change one project membership (M3 design
// 3.5, 3.7): updateProjectMember and removeProjectMember, which name it by
// its id, and leaveProject, which names the project and the caller.

// ProjectMembership is a project membership as MemberByID reads it: its id,
// its workspace and project, its member's account, his role, and whether
// it is active.
type ProjectMembership struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	MemberID    uuid.UUID
	Role        shared.Role
	Active      bool
}

// MemberFinder reads a project membership by its id: what a write on it
// reads first, for the workspace and the project it locks, and again under
// their locks (Locks).
type MemberFinder interface {
	// MemberByID is the undeleted membership id, active or ended; found is
	// false when there is none.
	MemberByID(ctx context.Context, id uuid.UUID) (m ProjectMembership, found bool, err error)
}

// MemberRoleChanger is updateProjectMember's repository. It runs in the
// transaction ctx carries, under the project's FOR NO KEY UPDATE.
type MemberRoleChanger interface {
	// UpdateMemberRole gives the active membership id role, by the account
	// by at now, and returns it as stored. An ended or deleted one is an
	// error, and is not written.
	UpdateMemberRole(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) (domain.Member, error)
}

// MemberEnder is removeProjectMember's repository. It runs in the
// transaction ctx carries, under the project's FOR NO KEY UPDATE.
type MemberEnder interface {
	// EndMember ends userID's active membership of projectID, by the account
	// by at now; the row stays. Anything but exactly one row ended is an
	// error.
	EndMember(ctx context.Context, projectID, userID, by uuid.UUID, now time.Time) error
}

// MemberLeaver is leaveProject's repository. It runs in the transaction ctx
// carries, under the project's FOR NO KEY UPDATE.
type MemberLeaver interface {
	MemberEnder
	// HasOtherAdmin reports whether projectID has an active admin other than
	// userID.
	HasOtherAdmin(ctx context.Context, projectID, userID uuid.UUID) (bool, error)
}
````

`server/internal/modules/project/adapter/postgres/membership.go`（新文件，65 行）：

````file server/internal/modules/project/adapter/postgres/membership.go
package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// MemberByID is the undeleted project membership id, active or ended;
// found is false when there is none (app.MemberFinder).
func (s *Store) MemberByID(ctx context.Context, id uuid.UUID) (m app.ProjectMembership, found bool, err error) {
	r, err := s.queries(ctx).MemberByID(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return app.ProjectMembership{}, false, nil
	case err != nil:
		return app.ProjectMembership{}, false, fmt.Errorf("read project membership %s: %w", id, err)
	}
	return app.ProjectMembership{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, MemberID: r.MemberID, Role: shared.Role(r.Role),
		Active: r.IsActive}, true, nil
}

// UpdateMemberRole gives the active membership id role, by the account by
// at now, and returns it as stored (app.MemberRoleChanger). An ended or
// deleted one is an error, not written.
func (s *Store) UpdateMemberRole(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) (domain.Member, error) {
	r, err := s.queries(ctx).UpdateMemberRole(ctx, gen.UpdateMemberRoleParams{ID: id, Role: int16(role), UpdatedBy: by, Now: now})
	if err != nil {
		return domain.Member{}, fmt.Errorf("change the role of project membership %s: %w", id, err)
	}
	return domain.Member{ID: r.ID, ProjectID: r.ProjectID, MemberID: r.MemberID, Role: shared.Role(r.Role), CreatedAt: r.CreatedAt}, nil
}

// EndMember ends userID's active membership of projectID, by the account
// by at now (app.MemberEnder): anything but exactly one row ended is an
// error.
func (s *Store) EndMember(ctx context.Context, projectID, userID, by uuid.UUID, now time.Time) error {
	n, err := s.queries(ctx).EndMember(ctx, gen.EndMemberParams{ProjectID: projectID, MemberID: userID, EndedBy: by, Now: now})
	switch {
	case err != nil:
		return fmt.Errorf("end the membership of %s of project %s: %w", userID, projectID, err)
	case n != 1:
		return fmt.Errorf("end the membership of %s of project %s: %d rows ended, want 1", userID, projectID, n)
	}
	return nil
}

// HasOtherAdmin reports whether projectID has an active admin other than
// userID (app.MemberLeaver).
func (s *Store) HasOtherAdmin(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	other, err := s.queries(ctx).HasOtherAdmin(ctx, gen.HasOtherAdminParams{ProjectID: projectID, MemberID: userID})
	if err != nil {
		return false, fmt.Errorf("look for another admin of project %s: %w", projectID, err)
	}
	return other, nil
}
````

`server/internal/modules/project/adapter/postgres/membership_test.go`（新文件，253 行）：

````file server/internal/modules/project/adapter/postgres/membership_test.go
package postgresadapter_test

import (
	"context"
	"fmt"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// membershipRows is every membership as text, by id: the one id without
// the columns cols, which a write of it writes.
func membershipRows(t *testing.T, pool *pgxpool.Pool, id uuid.UUID, cols ...string) string {
	t.Helper()
	var rows string
	if err := pool.QueryRow(context.Background(), `SELECT string_agg(CASE WHEN r.id = $1 THEN (to_jsonb(r) - $2::text[])::text
		ELSE r::text END, E'\n' ORDER BY r.id) FROM project_members r`, id, cols).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	return rows
}

// seedDeleted writes user's deleted membership of project, of role,
// active as a deletion leaves it, beside a live one or none, written by user
// at now, and returns its id.
func seedDeleted(t *testing.T, pool *pgxpool.Pool, workspace, project, user uuid.UUID, role int) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	exec(t, pool, `INSERT INTO project_members (id, workspace_id, project_id, member_id, role, created_by_id, updated_by_id, created_at,
		updated_at, deleted_at) VALUES ($1, $2, $3, $4, $5, $4, $4, $6, $6, $6)`, id, workspace, project, user, role, now)
	return id
}

// written reads the role, is_active, updated_at and updated_by_id of the
// membership id.
func written(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) string {
	t.Helper()
	var role int
	var active bool
	var at time.Time
	var by uuid.UUID
	if err := pool.QueryRow(context.Background(), "SELECT role, is_active, updated_at, updated_by_id FROM project_members WHERE id = $1",
		id).Scan(&role, &active, &at, &by); err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("role %d, active %v, at %s by %s", role, active, at.UTC().Format(time.RFC3339Nano), by)
}

// MemberByID reads the undeleted membership the id names, active or ended,
// with its workspace, project, member and role: not another membership,
// which a read of whichever row lies first would answer for one of the two
// asked about; not a deleted one, which is none, nor an id of none.
func TestMemberByID(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web, site := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, beta, "Site", "SITE", alice)
	alices := seedMember(t, pool, acme, web, alice, 20, true)
	bobs := seedMember(t, pool, beta, site, bob, 5, false)
	deleted := seedDeleted(t, pool, acme, web, bob, 15)
	for _, tt := range []struct {
		name  string
		id    uuid.UUID
		want  app.ProjectMembership
		found bool
	}{
		{"alice's of Web, active", alices, app.ProjectMembership{ID: alices, WorkspaceID: acme, ProjectID: web, MemberID: alice,
			Role: shared.RoleAdmin, Active: true}, true},
		{"bob's of beta's Site, ended", bobs, app.ProjectMembership{ID: bobs, WorkspaceID: beta, ProjectID: site, MemberID: bob,
			Role: shared.RoleGuest}, true},
		{"bob's of Web, deleted", deleted, app.ProjectMembership{}, false},
		{"none", uuid.NewV7(), app.ProjectMembership{}, false},
	} {
		if m, found, err := s.MemberByID(context.Background(), tt.id); err != nil || found != tt.found || m != tt.want {
			t.Errorf("%s: MemberByID() = %+v, %v, %v; want %+v, %v", tt.name, m, found, err, tt.want, tt.found)
		}
	}
}

// UpdateMemberRole gives bob's active membership of Web the role given, a
// guest's, at the moment and by the account given, and answers it as
// stored: its id, project, member, new role and the time it was made. The
// row keeps its other columns, and every other row every column: his
// memberships of Ops and of beta's Site, his deleted one of Web, stored
// before his live one or after it, carol's of Web. His ended membership of
// Docs, and his deleted one of Web, are each an error, and not written.
func TestUpdateMemberRole(t *testing.T) {
	for _, deletedFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("his deleted membership of Web stored first %v", deletedFirst), func(t *testing.T) {
			s, pool := newStore(t)
			alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
			acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
			web, ops, docs := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice),
				newProject(t, s, acme, "Docs", "DOCS", alice)
			site := newProject(t, s, beta, "Site", "SITE", alice)
			var deleted uuid.UUID
			deleteOne := func() { deleted = seedDeleted(t, pool, acme, web, bob, 15) }
			if deletedFirst {
				deleteOne()
			}
			bobs := seedMember(t, pool, acme, web, bob, 15, true)
			if !deletedFirst {
				deleteOne()
			}
			seedMember(t, pool, acme, ops, bob, 15, true)
			seedMember(t, pool, beta, site, bob, 15, true)
			seedMember(t, pool, acme, web, carol, 15, true)
			ended := seedMember(t, pool, acme, docs, bob, 15, false)
			before := membershipRows(t, pool, bobs, "role", "updated_at", "updated_by_id")
			later := now.Add(time.Hour)

			m, err := s.UpdateMemberRole(context.Background(), bobs, shared.RoleGuest, alice, later)
			if want := (domain.Member{ID: bobs, ProjectID: web, MemberID: bob, Role: shared.RoleGuest, CreatedAt: now}); err != nil || m != want {
				t.Errorf("UpdateMemberRole() = %+v, %v; want %+v", m, err, want)
			}
			if got, want := written(t, pool, bobs), "role 5, active true, at "+later.UTC().Format(time.RFC3339Nano)+" by "+alice.String(); got != want {
				t.Errorf("bob's membership of Web: %s; want %s", got, want)
			}
			if after := membershipRows(t, pool, bobs, "role", "updated_at", "updated_by_id"); after != before {
				t.Errorf("the memberships, bob's of Web without role, updated_at and updated_by_id:\n%s\nwant them as they were:\n%s", after, before)
			}
			for name, id := range map[string]uuid.UUID{"his ended one of Docs": ended, "his deleted one of Web": deleted} {
				before := membershipRows(t, pool, uuid.UUID{})
				if m, err := s.UpdateMemberRole(context.Background(), id, shared.RoleMember, alice, later.Add(time.Hour)); err == nil {
					t.Errorf("UpdateMemberRole() of %s = %+v, nil; want an error", name, m)
				}
				if after := membershipRows(t, pool, uuid.UUID{}); after != before {
					t.Errorf("changing %s wrote:\n%s\nwant the memberships as they were:\n%s", name, after, before)
				}
			}
		})
	}
}

// EndMember ends bob's active membership of Web, at the moment and by the
// account given; its role stays: a guest's when his deleted membership is
// stored first, which an ending that wrote a member's or an admin's would
// change, and an admin's when it is stored after, which an ending that
// wrote the column's default, a guest's, would. The row keeps its other
// columns, and every other row every column: his memberships of Ops and of
// beta's Site, his deleted one of Web, active as a deletion leaves it,
// stored before his live one or after it, and carol's of Web. Asked again,
// with nothing active left to end, it is an error and writes nothing; so
// is asking about Docs, where his membership ended, and about Gone, where
// his only membership is deleted, though active.
func TestEndMember(t *testing.T) {
	for _, deletedFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("his deleted membership of Web stored first %v", deletedFirst), func(t *testing.T) {
			s, pool := newStore(t)
			alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
			acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
			web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
			docs, gone := newProject(t, s, acme, "Docs", "DOCS", alice), newProject(t, s, acme, "Gone", "GONE", alice)
			site := newProject(t, s, beta, "Site", "SITE", alice)
			deleted := func(project uuid.UUID) { seedDeleted(t, pool, acme, project, bob, 15) }
			if deletedFirst {
				deleted(web)
			}
			role := 5
			if !deletedFirst {
				role = 20
			}
			bobs := seedMember(t, pool, acme, web, bob, role, true)
			if !deletedFirst {
				deleted(web)
			}
			seedMember(t, pool, acme, ops, bob, 15, true)
			seedMember(t, pool, beta, site, bob, 15, true)
			seedMember(t, pool, acme, web, carol, 15, true)
			seedMember(t, pool, acme, docs, bob, 15, false)
			deleted(gone)
			before := membershipRows(t, pool, bobs, "is_active", "updated_at", "updated_by_id")
			later := now.Add(time.Hour)

			if err := s.EndMember(context.Background(), web, bob, alice, later); err != nil {
				t.Fatalf("EndMember() = %v", err)
			}
			if got, want := written(t, pool, bobs), fmt.Sprintf("role %d, active false, at ", role)+later.UTC().Format(time.RFC3339Nano)+" by "+alice.String(); got != want {
				t.Errorf("bob's membership of Web: %s; want %s", got, want)
			}
			if after := membershipRows(t, pool, bobs, "is_active", "updated_at", "updated_by_id"); after != before {
				t.Errorf("the memberships, bob's of Web without is_active, updated_at and updated_by_id:\n%s\nwant them as they were:\n%s", after,
					before)
			}
			for name, project := range map[string]uuid.UUID{"Web again": web, "Docs": docs, "Gone": gone} {
				before := membershipRows(t, pool, uuid.UUID{})
				if err := s.EndMember(context.Background(), project, bob, alice, later.Add(time.Hour)); err == nil {
					t.Errorf("EndMember() of %s = nil; want an error", name)
				}
				if after := membershipRows(t, pool, uuid.UUID{}); after != before {
					t.Errorf("ending bob's membership of %s wrote:\n%s\nwant the memberships as they were:\n%s", name, after, before)
				}
			}
		})
	}
}

// HasOtherAdmin reports whether the project asked about has an active
// admin other than bob (M3 design 3.7 rule 1). Every project is acme's,
// and in each case one other project has an active admin other than bob,
// so that a check of another project answers otherwise; bob is an active
// admin of the projects where he is asked about as one, so that a check
// that counts him answers otherwise.
func TestHasOtherAdmin(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	acme := newWorkspace(t, pool, "acme")
	type member struct {
		user   uuid.UUID
		role   int
		active bool
	}
	n := 0
	// project stores a project of acme with members, the last of them
	// deleted when deleteLast is set, and returns its id.
	project := func(deleteLast bool, members ...member) uuid.UUID {
		t.Helper()
		n++
		p := newProject(t, s, acme, fmt.Sprintf("P%d", n), fmt.Sprintf("P%d", n), alice)
		for i, m := range members {
			id := seedMember(t, pool, acme, p, m.user, m.role, m.active)
			if deleteLast && i == len(members)-1 {
				exec(t, pool, "UPDATE project_members SET deleted_at = $2 WHERE id = $1", id, now)
			}
		}
		return p
	}
	project(false, member{alice, 20, true}, member{bob, 20, true})
	for _, tt := range []struct {
		name    string
		project uuid.UUID
		want    bool
	}{
		{"another active admin", project(false, member{bob, 20, true}, member{alice, 20, true}), true},
		{"another active admin, bob a member", project(false, member{bob, 15, true}, member{alice, 20, true}), true},
		{"bob alone", project(false, member{bob, 20, true}), false},
		{"another active member", project(false, member{bob, 20, true}, member{carol, 15, true}), false},
		{"another active guest", project(false, member{bob, 20, true}, member{carol, 5, true}), false},
		{"the other admin's membership ended", project(false, member{bob, 20, true}, member{alice, 20, false}), false},
		{"the other admin's membership deleted", project(true, member{bob, 20, true}, member{alice, 20, true}), false},
		{"no member", project(false), false},
	} {
		if got, err := s.HasOtherAdmin(context.Background(), tt.project, bob); err != nil || got != tt.want {
			t.Errorf("%s: HasOtherAdmin() = %v, %v; want %v", tt.name, got, err, tt.want)
		}
	}
}
````

`server/internal/modules/project/adapter/postgres/failures_test.go`（修改，4 处）：

````old server/internal/modules/project/adapter/postgres/failures_test.go
// member's project memberships still ended. Each read runs on a cancelled
// context against a project alice is the only admin of, beside bob, a
// member, and has display settings in, bob's membership of another project
// ended, so that the right answer is none of the zero values.
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
// member's project memberships still ended, nor "no such membership", which
// a write on it would answer as project.member_not_found, nor "no other
// admin", which leaveProject would answer as project.sole_admin. Each read
// runs on a cancelled context against a project alice is the only admin of,
// beside bob, a member, and has display settings in, bob's membership of
// another project ended, so that the right answer is none of the zero
// values.
````

````old server/internal/modules/project/adapter/postgres/failures_test.go
	seedMember(t, pool, acme, web, bob, 15, true)
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
	bobs := seedMember(t, pool, acme, web, bob, 15, true)
````

````old server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("CountInactive() = %d, %v; want context.Canceled, not a count", n, err)
	}
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("CountInactive() = %d, %v; want context.Canceled, not a count", n, err)
	}
	if m, found, err := s.MemberByID(cancelled, bobs); !failed(err) || found || m != (app.ProjectMembership{}) {
		t.Errorf("MemberByID() = %+v, %v, %v; want context.Canceled, not no membership", m, found, err)
	}
	if other, err := s.HasOtherAdmin(cancelled, web, bob); !failed(err) || other {
		t.Errorf("HasOtherAdmin() = %v, %v; want context.Canceled, not an answer", other, err)
	}
````

````old server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("EndMemberships() = %v; want context.Canceled", err)
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("EndMemberships() = %v; want context.Canceled", err)
	}
	if m, err := s.UpdateMemberRole(cancelled, uuid.NewV7(), shared.RoleMember, alice, now); !failed(err) || m != (domain.Member{}) {
		t.Errorf("UpdateMemberRole() = %+v, %v; want context.Canceled", m, err)
	}
	if err := s.EndMember(cancelled, web, alice, alice, now); !failed(err) {
		t.Errorf("EndMember() = %v; want context.Canceled", err)
````

- [ ] **Step 3: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/project/...`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`（`member_ports.go` 的四个端口这时只由存储实现，`app` 包里还没有使用者；它们是导出的接口，lint 不报未使用）

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/modules/project/adapter/postgres/failures_test.go server/internal/modules/project/adapter/postgres/membership.go server/internal/modules/project/adapter/postgres/membership_test.go server/internal/modules/project/adapter/postgres/queries/members.sql server/internal/modules/project/app/member_ports.go server/internal/modules/project/adapter/postgres/gen/members.sql.go
```
```bash
git commit -m "feat(M3/P5b): the project store reads a membership by its id, changes its role, ends it, and asks for another admin

MemberByID reads an undeleted project membership, active or ended,
with its workspace and project; UpdateMemberRole changes an active
one's role; EndMember ends a member's active membership of a project,
the row kept; HasOtherAdmin asks whether an active admin other than
the user is left. Each store test checks every column of every other
row, a deleted row before and after the live one, and each method's
failure comes back as an error, never a plausible answer.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A，清扫 1、3、8、19、20、26；故事一列是单独运行的故事 P5）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| `MemberByID` 去掉 id | `TestMemberByID`；`TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile`（Task 8 起）；P5（Task 11 起） | 存储；组合；端到端 |
| `MemberByID` 去掉 `deleted_at IS NULL` | `TestMemberByID`；`TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile`（成员关系删除的两个情形）、`TestPermissionMatrix`（gone 的成员关系）（故事看不到：项目成员关系只随项目删除，spec 第 3 节第 8 条） | 存储；组合 |
| `UpdateMemberRole` 去掉 id | `TestUpdateMemberRole`；`TestPermissionMatrix`、`TestEachLockOfAWriteOnAProjectMembershipIsItsStrength`；P5 | 存储；组合；端到端 |
| `UpdateMemberRole` 去掉 `is_active`、`deleted_at IS NULL` | `TestUpdateMemberRole`（组合一层等价：用例在锁下先拒绝已结束的；已删除的只随项目，spec 第 3 节第 8 条） | 存储 |
| `EndMember` 去掉项目；改成工作区的项目 | `TestEndMember`；`TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize`、`TestARestoredMembershipGivesNoMoreThanItHad`；P5（Task 11 起，Ops） | 存储；组合；端到端 |
| `EndMember` 去掉成员 | `TestEndMember`；`TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize`、`TestADemotionAndTheProjectSidesGrowthSerialize`；P5 | 存储；组合；端到端 |
| `EndMember` 去掉 `is_active`、`deleted_at IS NULL` | `TestEndMember`（组合一层等价，同 `UpdateMemberRole`） | 存储 |
| `HasOtherAdmin` 去掉项目；改成工作区的项目 | `TestHasOtherAdmin`；`TestLeavingAProject`（Task 7 起）；P5（Ops） | 存储；组合；端到端 |
| `HasOtherAdmin` 去掉角色；把他自己算进去 | `TestHasOtherAdmin`；`TestLeavingAProject`；P5 | 存储；组合；端到端 |
| `HasOtherAdmin` 去掉有效 | `TestHasOtherAdmin`；`TestLeavingAProject`（已结束的管理员一步） | 存储；组合 |
| `HasOtherAdmin` 去掉 `deleted_at IS NULL` | `TestHasOtherAdmin`（同 `MemberByID`） | 存储 |
| `HasOtherAdmin` 在项目没有别的有效成员时也答"有"（只有他一人照样离开） | `TestHasOtherAdmin`；`TestLeavingAProject`（只有他一人的一步）（故事看不到：故事里唯一管理员的项目有别的成员） | 存储；组合 |
| `EndMember` 把角色写成访客（列的默认）、成员的、20；`UpdateMemberRole` 写 `created_at`；把角色写成 20（只在管理员上看不出，清扫 26） | `TestEndMember`、`TestUpdateMemberRole`；`TestARestoredMembershipGivesNoMoreThanItHad`、`TestRemovingAProjectMember`、`TestLeavingAProject`、`TestTheRelativeRuleOnTheComposedApp`（Task 5–7 起）；P5（写角色的两个） | 存储；组合；端到端 |
| 不写写者（保留原来的 `updated_by_id`）；`EndMember` 保留原来的时刻 | `TestEndMember`、`TestUpdateMemberRole`；`TestTheWritesOnAProjectStampTheirRequest`（Task 8 起）、`TestRemovingAProjectMember`、`TestLeavingAProject`（`EndMember` 的两个）；P5（不写写者） | 存储；组合；端到端 |
| 一个方法的失败答成"没有"、被吞掉；`EndMember` 不核对行数 | `TestAFailedReadIsAnErrorNotAnAnswer`、`TestAFailedWriteIsAnError`；`TestEndMember`（行数：用例在锁下读到有效才结束，组合一层等价） | 存储 |
| `MemberByID`、`UpdateMemberRole`、`EndMember`、`HasOtherAdmin` 经连接池、在事务之外执行 | `TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 8 起） | 组合 |

**Done when:** 四条查询生成；四个存储测试通过，每个谓词都有一个只由它决定的行；每个新方法的失败原样返回。

---

### Task 2: 3.5 的相对规则（领域）

**Files:**
- Create: `server/internal/modules/project/domain/membership.go`、`server/internal/modules/project/domain/membership_test.go`
- Modify: `server/internal/modules/project/domain/errors.go`

**Interfaces:**
- Produces（spec 2.5，M3 设计 3.5、5.3）：`project/domain` 的三个码：`ErrMemberNotFound`（404 `project.member_not_found`）、`ErrOwnMembership`（409 `project.own_membership`）、`ErrRoleTooHigh`（403 `project.role_too_high`）。它们的文字照 spec 2.5（清扫 30：每个收到它的调用者都成立）。
- `CheckMemberRole(role) error`：三个角色之一，否则 422 `role` `invalid_format`（只看请求的值，在事务之前）。
- `RoleChange{Caller shared.Grant, Own bool, From, WorkspaceRole, To shared.Role}`；`CheckRoleChange(c) error`：不是工作区管理员的调用者：自己的 409；`From` 或 `To` 不在 `rolesBelow(调用者的项目角色)` 里 403 `project.role_too_high`；然后对每个调用者：`To` 不在 `assignable[WorkspaceRole]` 里 422 `role` `not_allowed`（工作区访客只能是访客）。
- `CheckRemoval(callerRole, own, role) error`：自己的 409（工作区管理员也是）；`role` 不在 `rolesUpTo(callerRole)` 里 403 `project.role_too_high`（工作区管理员同样没有例外）。
- "低于""不高于"按 `roleOrder` 里的位置（`rolesBelow`、`rolesUpTo`），不按数值：三者之外的角色下面什么都没有，也不在任何角色下面。

**Tests:**（`domain/membership_test.go`；`sameProblem` 比较同一个 `*shared.Error`，422 比较字段）
- `TestCheckMemberRole`：5、15、20 通过；0、10、25 各是 422，`role` `invalid_format`。
- `TestCheckRoleChange`：25 个情形，3.5 每一半一个成立的情形和它的反例：项目管理员（工作区成员）改成员为访客、访客为成员、保持成员；他自己的（改低、不变）409；另一位管理员、提拔为管理员 403；项目成员只能把访客保持为访客，别的 403；他自己的 409；三者之外的项目角色、从 10、改为 10 各 403；工作区管理员（项目成员、管理员、访客）改自己为管理员、改为访客、降另一位管理员、提拔成员、改另一位工作区管理员都通过；工作区访客改为成员、管理员 422（工作区管理员改也是），保持访客通过。
- `TestCheckRemoval`：12 个情形：管理员移出管理员、成员、访客通过；管理员、成员自己的 409；成员移出成员、访客通过，管理员 403；访客移出访客通过、成员 403；三者之外的调用者角色、10 这个目标角色各 403。

- [ ] **Step 1: 码和规则**

`server/internal/modules/project/domain/errors.go`（修改，1 处）：

````old server/internal/modules/project/domain/errors.go
	ErrArchived = shared.NewError(shared.KindConflict, "project.archived", "The project is archived; unarchive it to change it.")
````
````new server/internal/modules/project/domain/errors.go
	ErrArchived = shared.NewError(shared.KindConflict, "project.archived", "The project is archived; unarchive it to change it.")
	// ErrMemberNotFound answers a project membership that does not exist, is
	// deleted or has ended, or whose project the caller does not see: the
	// same 404 for all (M3 design 5.3, 8.2).
	ErrMemberNotFound = shared.NewError(shared.KindNotFound, "project.member_not_found",
		"The project member does not exist, or you cannot see the project.")
	// ErrOwnMembership answers a removal of the caller's own project
	// membership, and a change of his own project role by one who is not the
	// workspace's admin (M3 design 3.5, 5.3). It names no remedy: leaving,
	// which ends one's own membership, refuses a project's only admin too.
	ErrOwnMembership = shared.NewError(shared.KindConflict, "project.own_membership",
		"You cannot remove your own membership of the project, nor change your own role in it unless you are a workspace admin.")
	// ErrRoleTooHigh answers M3 design 3.5's relative rule: a change of the
	// role of a member whose role is not below the caller's, or to a role
	// not below his, by one who is not the workspace's admin; a removal of a
	// member whose role is above the caller's, by anyone.
	ErrRoleTooHigh = shared.NewError(shared.KindForbidden, "project.role_too_high",
		"The role is too high for you: unless you are a workspace admin, you change only a member whose project role is below yours, to a role "+
			"below yours; and you remove only a member whose project role is not above yours.")
````

`server/internal/modules/project/domain/membership.go`（新文件，102 行）：

````file server/internal/modules/project/domain/membership.go
package domain

import (
	"slices"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CheckMemberRole checks the role a change of a project membership gives,
// from the request alone: one of the three, else a 422 naming role.
func CheckMemberRole(role shared.Role) error {
	if !slices.Contains(roleOrder, role) {
		return shared.Invalid(shared.FieldError{Field: "role", Code: shared.FieldInvalidFormat, Message: "is not 5, 15 or 20"})
	}
	return nil
}

// assignable are the project roles a change of a membership can give its
// member, by his workspace role (M3 design 3.5; Plane
// views/project/member.py:257-261): a workspace guest a guest's alone, a
// workspace member or admin any of the three. The sets are named, not
// bounds.
var assignable = map[shared.Role][]shared.Role{
	shared.RoleAdmin:  roleOrder,
	shared.RoleMember: roleOrder,
	shared.RoleGuest:  {shared.RoleGuest},
}

// RoleChange is a change of a project membership's role as the use case
// reads it under its locks: the caller's grant, whether the membership is
// his own, its member's role in the project and in the workspace, and the
// role it is to have.
type RoleChange struct {
	Caller        shared.Grant
	Own           bool
	From          shared.Role
	WorkspaceRole shared.Role
	To            shared.Role
}

// CheckRoleChange checks c against M3 design 3.5, after the decision let
// the caller change roles (the project's admins, and its members who are
// the workspace's admins). One who is not the workspace's admin changes
// neither his own role (ErrOwnMembership), nor the role of a member whose
// role is not below his own, nor gives a role that is not below his own
// (ErrRoleTooHigh): a project admin promotes nobody to admin and changes no
// other admin. Then, for every caller, a workspace guest's role stays a
// guest's (422 role not_allowed). Below is roleOrder's, never the
// numbers': a role outside the three has no role below it and is below
// none.
func CheckRoleChange(c RoleChange) error {
	if c.Caller.WorkspaceRole != shared.RoleAdmin {
		below := rolesBelow(c.Caller.ProjectRole)
		switch {
		case c.Own:
			return ErrOwnMembership
		case !slices.Contains(below, c.From), !slices.Contains(below, c.To):
			return ErrRoleTooHigh
		}
	}
	if !slices.Contains(assignable[c.WorkspaceRole], c.To) {
		return shared.Invalid(shared.FieldError{Field: "role", Code: shared.FieldNotAllowed,
			Message: "must be 5: the member is a guest of the workspace"})
	}
	return nil
}

// CheckRemoval checks a removal of a project membership of role, the
// caller's own when own is set, after the decision let the caller remove
// members (M3 design 3.5): nobody removes his own membership
// (ErrOwnMembership: he leaves the project), nor one whose role is above
// his own project role, callerRole (ErrRoleTooHigh), the workspace's admins
// neither (Plane views/project/member.py:290-321 gives them no exception).
func CheckRemoval(callerRole shared.Role, own bool, role shared.Role) error {
	switch {
	case own:
		return ErrOwnMembership
	case !slices.Contains(rolesUpTo(callerRole), role):
		return ErrRoleTooHigh
	}
	return nil
}

// rolesBelow are the roles below role in roleOrder: none for a role outside
// it.
func rolesBelow(role shared.Role) []shared.Role {
	i := slices.Index(roleOrder, role)
	if i < 0 {
		return nil
	}
	return slices.Clone(roleOrder[:i])
}

// rolesUpTo are role and the roles below it in roleOrder: none for a role
// outside it.
func rolesUpTo(role shared.Role) []shared.Role {
	i := slices.Index(roleOrder, role)
	if i < 0 {
		return nil
	}
	return slices.Clone(roleOrder[:i+1])
}
````

- [ ] **Step 2: 测试**

`server/internal/modules/project/domain/membership_test.go`（新文件，138 行）：

````file server/internal/modules/project/domain/membership_test.go
package domain

import (
	"errors"
	"reflect"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The three roles are accepted; any other, between them or outside them,
// is a 422 naming role.
func TestCheckMemberRole(t *testing.T) {
	for _, role := range []shared.Role{shared.RoleGuest, shared.RoleMember, shared.RoleAdmin} {
		if err := CheckMemberRole(role); err != nil {
			t.Errorf("CheckMemberRole(%d) = %v, want nil", role, err)
		}
	}
	want := []shared.FieldError{{Field: "role", Code: shared.FieldInvalidFormat, Message: "is not 5, 15 or 20"}}
	for _, role := range []shared.Role{0, 10, 25} {
		var e *shared.Error
		if err := CheckMemberRole(role); !errors.As(err, &e) || e.Code != "validation_failed" || !reflect.DeepEqual(e.Fields, want) {
			t.Errorf("CheckMemberRole(%d) = %v, want validation_failed with %+v", role, err, want)
		}
	}
}

// The callers of the relative rule (M3 design 3.5): their workspace and
// project roles.
var (
	projectAdmin   = shared.Grant{WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleAdmin, ProjectAdmin: true}
	projectMember  = shared.Grant{WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleMember}
	bothAdmin      = shared.Grant{WorkspaceRole: shared.RoleAdmin, ProjectRole: shared.RoleAdmin, ProjectAdmin: true}
	adminAsMember  = shared.Grant{WorkspaceRole: shared.RoleAdmin, ProjectRole: shared.RoleMember, ProjectAdmin: true}
	adminAsGuest   = shared.Grant{WorkspaceRole: shared.RoleAdmin, ProjectRole: shared.RoleGuest, ProjectAdmin: true}
	unknownProject = shared.Grant{WorkspaceRole: shared.RoleMember, ProjectRole: 25}
)

// guestOnly is the 422 of a workspace guest given a role other than a
// guest's.
var guestOnly = shared.Invalid(shared.FieldError{Field: "role", Code: shared.FieldNotAllowed, Message: "must be 5: the member is a guest of the workspace"})

// sameProblem reports whether err is want: the same *shared.Error, or a 422
// with the same fields; nil for nil.
func sameProblem(err, want error) bool {
	var e, w *shared.Error
	switch {
	case want == nil || err == nil:
		return err == want
	case !errors.As(err, &e) || !errors.As(want, &w):
		return false
	case w.Code == "validation_failed":
		return e.Code == w.Code && reflect.DeepEqual(e.Fields, w.Fields)
	}
	return e == w
}

// Each half of M3 design 3.5's rule for a change of role, each with the
// case that holds it and its counterexample. One who is not the
// workspace's admin: his own role, never (even unchanged, even to a role
// below his); another's, only from a role below his own to a role below
// his own, so a project admin changes no admin and makes none, and a
// project member changes a guest to a guest at most. The workspace's
// admin, whatever his project role: his own, an admin's, to admin. For
// everyone: a workspace guest is a guest. Below is by roleOrder: a project
// role outside the three has nothing below it, and a role between them
// (10, below 20 by the numbers) is below nothing it is not in roleOrder
// before.
func TestCheckRoleChange(t *testing.T) {
	member, guest, admin := shared.RoleMember, shared.RoleGuest, shared.RoleAdmin
	for _, tt := range []struct {
		name string
		c    RoleChange
		want error
	}{
		{"a project admin makes a member a guest", RoleChange{projectAdmin, false, member, member, guest}, nil},
		{"a project admin makes a guest a member", RoleChange{projectAdmin, false, guest, member, member}, nil},
		{"a project admin keeps a member a member", RoleChange{projectAdmin, false, member, member, member}, nil},
		{"a project admin, his own role, to a member's", RoleChange{projectAdmin, true, admin, member, member}, ErrOwnMembership},
		{"a project admin, his own role, kept", RoleChange{projectAdmin, true, admin, member, admin}, ErrOwnMembership},
		{"a project admin changes another admin", RoleChange{projectAdmin, false, admin, member, member}, ErrRoleTooHigh},
		{"a project admin makes a member an admin", RoleChange{projectAdmin, false, member, member, admin}, ErrRoleTooHigh},
		{"a project admin makes a guest an admin", RoleChange{projectAdmin, false, guest, member, admin}, ErrRoleTooHigh},
		{"a project member keeps a guest a guest", RoleChange{projectMember, false, guest, member, guest}, nil},
		{"a project member makes a member a guest", RoleChange{projectMember, false, member, member, guest}, ErrRoleTooHigh},
		{"a project member makes a guest a member", RoleChange{projectMember, false, guest, member, member}, ErrRoleTooHigh},
		{"a project member, his own role", RoleChange{projectMember, true, member, member, guest}, ErrOwnMembership},
		{"a project role outside the three", RoleChange{unknownProject, false, guest, member, guest}, ErrRoleTooHigh},
		{"a project admin, from a role between the three", RoleChange{projectAdmin, false, 10, member, guest}, ErrRoleTooHigh},
		{"a project admin, to a role between the three", RoleChange{projectAdmin, false, guest, member, 10}, ErrRoleTooHigh},
		{"a workspace admin, his own role, to an admin's", RoleChange{adminAsMember, true, member, admin, admin}, nil},
		{"a workspace admin and project admin, his own role, to a guest's", RoleChange{bothAdmin, true, admin, admin, guest}, nil},
		{"a workspace admin makes the admin a member", RoleChange{adminAsMember, false, admin, member, member}, nil},
		{"a workspace admin makes a member an admin", RoleChange{adminAsMember, false, member, member, admin}, nil},
		{"a workspace admin who is a project guest changes an admin", RoleChange{adminAsGuest, false, admin, member, guest}, nil},
		{"a workspace admin makes another workspace admin a member", RoleChange{bothAdmin, false, admin, admin, member}, nil},
		{"a project admin makes a workspace guest a member", RoleChange{projectAdmin, false, guest, guest, member}, guestOnly},
		{"a workspace admin makes a workspace guest a member", RoleChange{adminAsMember, false, guest, guest, member}, guestOnly},
		{"a workspace admin makes a workspace guest an admin", RoleChange{bothAdmin, false, guest, guest, admin}, guestOnly},
		{"a workspace admin keeps a workspace guest a guest", RoleChange{bothAdmin, false, guest, guest, guest}, nil},
	} {
		if err := CheckRoleChange(tt.c); !sameProblem(err, tt.want) {
			t.Errorf("%s: CheckRoleChange(%+v) = %v, want %v", tt.name, tt.c, err, tt.want)
		}
	}
}

// M3 design 3.5's rule for a removal: nobody his own membership, whatever
// his roles; nobody a member whose project role is above his own, the
// workspace's admins neither; an equal or lower role, anyone the decision
// lets remove. Above is by roleOrder: a project role outside the three has
// nothing up to it, and a role between them is up to nothing.
func TestCheckRemoval(t *testing.T) {
	for _, tt := range []struct {
		name   string
		caller shared.Role
		own    bool
		role   shared.Role
		want   error
	}{
		{"an admin removes an admin", shared.RoleAdmin, false, shared.RoleAdmin, nil},
		{"an admin removes a member", shared.RoleAdmin, false, shared.RoleMember, nil},
		{"an admin removes a guest", shared.RoleAdmin, false, shared.RoleGuest, nil},
		{"an admin, his own", shared.RoleAdmin, true, shared.RoleAdmin, ErrOwnMembership},
		{"a member, his own", shared.RoleMember, true, shared.RoleMember, ErrOwnMembership},
		{"a member removes a member", shared.RoleMember, false, shared.RoleMember, nil},
		{"a member removes a guest", shared.RoleMember, false, shared.RoleGuest, nil},
		{"a member removes an admin", shared.RoleMember, false, shared.RoleAdmin, ErrRoleTooHigh},
		{"a guest removes a guest", shared.RoleGuest, false, shared.RoleGuest, nil},
		{"a guest removes a member", shared.RoleGuest, false, shared.RoleMember, ErrRoleTooHigh},
		{"a project role outside the three", 25, false, shared.RoleGuest, ErrRoleTooHigh},
		{"an admin removes a role between the three", shared.RoleAdmin, false, 10, ErrRoleTooHigh},
	} {
		if err := CheckRemoval(tt.caller, tt.own, tt.role); !sameProblem(err, tt.want) {
			t.Errorf("%s: CheckRemoval(%d, %v, %d) = %v, want %v", tt.name, tt.caller, tt.own, tt.role, err, tt.want)
		}
	}
}
````

- [ ] **Step 3: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/project/domain/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/modules/project/domain/errors.go server/internal/modules/project/domain/membership.go server/internal/modules/project/domain/membership_test.go
```
```bash
git commit -m "feat(M3/P5b): the relative rule of a project role change and of a removal

CheckRoleChange holds one who is no workspace admin to roles below his
own, from and to, and never his own; for everyone, a workspace guest
stays a guest. CheckRemoval refuses one's own membership and a role
above one's own, to workspace admins too. Below and above are by the
roles' order, never their numbers.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A，清扫 5、24、按集合；单元一层之外的测试从后面的 Task 起）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| 不是工作区管理员的人也不受相对规则约束 | `TestCheckRoleChange`；`TestTheRelativeRuleOnTheComposedApp`（Task 5 起）、`TestPermissionMatrix`；P5（Task 11 起） | 单元；组合；端到端 |
| 工作区管理员也受相对规则约束（没有例外） | `TestCheckRoleChange`；`TestTheRelativeRuleOnTheComposedApp`、`TestPermissionMatrix`（PM+WA 改自己为管理员） | 单元；组合 |
| 不看 `From`（管理员改另一位管理员）；不看 `To`（提拔为管理员） | `TestCheckRoleChange`；`TestTheRelativeRuleOnTheComposedApp`；P5（不看 `From`） | 单元；组合；端到端 |
| 不看自己的（改角色） | `TestCheckRoleChange`；`TestTheRelativeRuleOnTheComposedApp`、`TestPermissionMatrix` | 单元；组合 |
| 去掉工作区访客的上限 | `TestCheckRoleChange`；`TestTheRelativeRuleOnTheComposedApp`、`TestPermissionMatrix` | 单元；组合 |
| 移出：不看高于自己的；只许低于自己的；不看自己的 | `TestCheckRemoval`；`TestRemovingAProjectMember`（Task 6 起）、`TestPermissionMatrix`；P5（只许低于自己的；故事里没有人移出自己） | 单元；组合；端到端 |
| 按数值比较（`>=`、`>`） | `TestCheckRoleChange`、`TestCheckRemoval`（10、25 的情形） | 单元 |

**Done when:** 三个领域测试通过；3.5 的每一半在单元一层有成立的情形和反例。

---

### Task 3: 按成员关系 id 寻址的取锁路径与 `updateProjectMember` 的用例

**Files:**
- Create: `server/internal/modules/project/app/fakes_member_test.go`、`server/internal/modules/project/app/update_member.go`、`server/internal/modules/project/app/update_member_test.go`
- Modify: `server/internal/modules/project/app/clock_test.go`、`server/internal/modules/project/app/fakes_create_test.go`、`server/internal/modules/project/app/fakes_test.go`、`server/internal/modules/project/app/fakes_write_test.go`、`server/internal/modules/project/app/get_project.go`、`server/internal/modules/project/app/lock.go`、`server/internal/modules/project/app/ports.go`、`server/internal/modules/project/domain/actions.go`

**Interfaces:**
- Produces（spec 2.6，M3 设计 3.3、3.5、3.6 约定二、三和加锁表，6.7）：
  - `ProjectLocks` 多 `MemberFinder`。`Locks.lockMemberAndDecide(ctx, actor, id, action, target bool) (held, error)`：`MemberByID`（不加锁；没有 404 `project.member_not_found`；回答的 id 不是问的，是写自己的错误，清扫 22）→ `lock`：工作区 S（`ShareWorkspaceByID`，已删除 404）→ `target` 时成员在工作区的成员关系 S（`ShareMembers`，回答 `h.roles`）→ 项目 N（`LockProject`，已删除或不在这个工作区 404）→ 在锁下重读成员关系（没有 404，不再属于这个项目 404）→ 判定（看不到项目 404，角色不够 403）。`held.member` 是锁下读到的成员关系。
  - `lockAndDecide` 的锁部分提成 `lock(ctx, workspaceID, w, notFound)`，两条路径共用；`decide` 多一个参数 `notFound`：按项目 id 的写答 `project.not_found`，按成员关系 id 的写答 `project.member_not_found`。`get_project.go` 照旧传 `domain.ErrNotFound`。
  - `UpdateProjectMember`（`NewUpdateProjectMember(locks, members MemberRoleChanger, tx, clock)`），`Execute(ctx, id, role) (domain.Member, error)`：没有调用者 401 → `CheckMemberRole`（事务之前）→ 一个事务：`lockMemberAndDecide(…, project_member.update, target: true)` → 已结束 404 → 成员在工作区的角色（`h.roles` 里没有他是写自己的错误：约定六保证不会）→ `CheckRoleChange` → 读时钟 → `UpdateMemberRole`（回答的 id 不是这个成员关系是写自己的错误）。
  - `domain.ActionMemberUpdate`（`project_member.update`）；它的规则行和 `Actions()` 在 Task 4（与接口、矩阵一起）。
- 单元测试的假对象：`fakeStore.MemberByID`、`UpdateMemberRole`；`memberReads`（锁下重读失败、删除、移到别的项目）；`answersAs`、`changedAs`、`fakeMembers.answersFor`（回答别的 id、别的账户，清扫 22）；固定的成员关系 id（`bobInWeb`、`aliceInWeb`…）；`newMemberWrites`（gina：工作区管理员、项目成员；frank：另一位管理员；ivy：工作区访客）；`memberLocked`（取锁到判定的调用记录）；`outcome.check`（第一个 `*shared.Error` 是期望的那个，失败原样返回，事务的函数原样返回它：`fakeTx.answered`，清扫 21）。

**Tests:**
- `update_member_test.go`：`TestUpdateProjectMember`（5 个调用者和目标：项目管理员 bob 改成员为访客、保持工作区访客为访客；工作区管理员 gina 改自己为管理员、降另一位管理员、提拔成员；调用顺序是取锁、判定、读时钟、改，由调用者在那个时刻）；`TestUpdateProjectMemberRefuses`（20 个情形，每个在它的位置、目标不变：没有调用者、三者之外的角色在事务之前；没有成员关系、工作区或项目在等锁时删除、项目移到别的工作区、成员关系删除或移到 ops、看不到项目，各 404；项目成员 403；判定之后：已结束 404、自己的 409、另一位管理员、提拔为管理员、锁下重读到的管理员（等锁时被提拔）403 `project.role_too_high`、工作区访客改为成员 422（工作区管理员改也是）；他不是工作区的有效成员、角色回答成别的账户的、成员关系回答成别的 id 的，是写自己的错误）；`TestUpdateProjectMemberReturnsEachFailure`（9 个：读、工作区锁、成员在工作区的成员关系、项目锁、锁下重读、判定、改、改的回答是别的成员关系、提交；每个之后的调用都没有执行）。
- `clock_test.go`：`TestEachWriteReadsTheClockUnderItsLock` 加 `updateProjectMember` 一行。

- [ ] **Step 1: 操作名和取锁路径**

`server/internal/modules/project/domain/actions.go`（修改，1 处）：

````old server/internal/modules/project/domain/actions.go
	ActionJoin shared.Action = "project.join"
````
````new server/internal/modules/project/domain/actions.go
	ActionJoin shared.Action = "project.join"
	// ActionMemberUpdate is changing a project member's role:
	// updateProjectMember.
	ActionMemberUpdate shared.Action = "project_member.update"
````

`server/internal/modules/project/app/ports.go`（修改，2 处）：

````old server/internal/modules/project/app/ports.go
// project (Locks): the project's workspace, read first without a lock, and
// the project's own lock.
````
````new server/internal/modules/project/app/ports.go
// project (Locks): the project's workspace, read first without a lock, the
// project's own lock, and the membership a write on one names.
````

````old server/internal/modules/project/app/ports.go
	ProjectSharer
````
````new server/internal/modules/project/app/ports.go
	ProjectSharer
	MemberFinder
````

`server/internal/modules/project/app/lock.go`（修改，11 处）：

````old server/internal/modules/project/app/lock.go
	"errors"
````
````new server/internal/modules/project/app/lock.go
	"errors"
	"fmt"
````

````old server/internal/modules/project/app/lock.go
// Locks is the one way a write on a project named by its id takes its
// locks and its decision (M3 design 3.6 convention 2), in the transaction
// ctx carries: the project's workspace, read without a lock; the
// workspace's row FOR SHARE, the write's first lock; the memberships of the
// workspace of the accounts the write makes members of the project, FOR
// SHARE in id order (convention 3); the project's row, FOR NO KEY UPDATE,
// or FOR SHARE for a write under the project that leaves the row and its
// memberships as they are, still undeleted and of that workspace; then the
// decision, under them all. Every cascade over the workspace's projects
````
````new server/internal/modules/project/app/lock.go
// Locks is the one way a write on a project takes its locks and its
// decision (M3 design 3.6 convention 2), in the transaction ctx carries:
// the project's workspace, read without a lock (from the project, or from
// the membership a write on one names); the workspace's row FOR SHARE, the
// write's first lock; the memberships of the workspace of the accounts the
// write makes members of the project, or whose role it changes, FOR SHARE
// in id order (convention 3); the project's row, FOR NO KEY UPDATE, or FOR
// SHARE for a write under the project that leaves the row and its
// memberships as they are, still undeleted and of that workspace; the
// membership a write on one names, read again, still of that project; then
// the decision, under them all. Every cascade over the workspace's projects
````

````old server/internal/modules/project/app/lock.go
	// targets are the accounts the write makes members of the project.
````
````new server/internal/modules/project/app/lock.go
	// targets are the accounts the write makes members of the project, or
	// whose role in it it changes.
````

````old server/internal/modules/project/app/lock.go
// lock read it, the caller's grant, and the active roles in the workspace
// of the write's targets, by account.
````
````new server/internal/modules/project/app/lock.go
// lock read it, the caller's grant, the active roles in the workspace of
// the write's targets, by account, and, for a write on a membership, the
// membership as read under the locks.
````

````old server/internal/modules/project/app/lock.go
	roles   map[uuid.UUID]shared.Role
````
````new server/internal/modules/project/app/lock.go
	roles   map[uuid.UUID]shared.Role
	member  ProjectMembership
````

````old server/internal/modules/project/app/lock.go
	_, found, err = l.workspaces.ShareWorkspaceByID(ctx, workspaceID)
````
````new server/internal/modules/project/app/lock.go
	h, err := l.lock(ctx, workspaceID, w, domain.ErrNotFound)
	if err != nil {
		return held{}, err
	}
	if h.grant, err = decide(ctx, l.auth, actor, w.action, workspaceID, w.project, domain.ErrNotFound); err != nil {
		return held{}, err
	}
	return h, nil
}

// lockMemberAndDecide takes the locks of a write on the project membership
// id, a write addressed by its resource (M3 design 3.6 convention 2), and
// decides action on its project, in the order of Locks: the membership,
// read without a lock, names its project and the project's workspace; then
// their locks, with the membership's member as the write's target when
// target is set; then the membership read again, which must still be of
// that project; then the decision. A membership that is not there or is
// deleted, a workspace or project deleted while its lock waited, a
// membership no longer of the project, and a project not visible to actor
// are each domain.ErrMemberNotFound; a role the rule does not allow is the
// Authorizer's shared.Forbidden. An ended membership is the use case's to
// refuse, after the decision. A membership read for another id than asked
// is an error.
func (l Locks) lockMemberAndDecide(ctx context.Context, actor shared.Actor, id uuid.UUID, action shared.Action, target bool) (held, error) {
	m, err := l.member(ctx, id)
	if err != nil {
		return held{}, err
	}
	w := write{project: m.ProjectID, action: action}
	if target {
		w.targets = []uuid.UUID{m.MemberID}
	}
	h, err := l.lock(ctx, m.WorkspaceID, w, domain.ErrMemberNotFound)
	if err != nil {
		return held{}, err
	}
	if h.member, err = l.member(ctx, id); err != nil {
		return held{}, err
	}
	if h.member.ProjectID != m.ProjectID {
		return held{}, domain.ErrMemberNotFound
	}
	if h.grant, err = decide(ctx, l.auth, actor, action, m.WorkspaceID, m.ProjectID, domain.ErrMemberNotFound); err != nil {
		return held{}, err
	}
	return h, nil
}

// member reads the undeleted membership id: domain.ErrMemberNotFound when
// there is none, and an error when the store answers another one.
func (l Locks) member(ctx context.Context, id uuid.UUID) (ProjectMembership, error) {
	m, found, err := l.projects.MemberByID(ctx, id)
	switch {
	case err != nil:
		return ProjectMembership{}, err
	case !found:
		return ProjectMembership{}, domain.ErrMemberNotFound
	case m.ID != id:
		return ProjectMembership{}, fmt.Errorf("project membership %s read as %s", id, m.ID)
	}
	return m, nil
}

// lock takes w's locks after the read that named its project's workspace,
// workspaceID, in the order of Locks: the workspace FOR SHARE, the targets'
// memberships of it, the project. A workspace or project deleted while its
// lock waited, or a project not of workspaceID, is notFound, the 404 of
// what the caller named.
func (l Locks) lock(ctx context.Context, workspaceID uuid.UUID, w write, notFound error) (held, error) {
	_, found, err := l.workspaces.ShareWorkspaceByID(ctx, workspaceID)
````

````old server/internal/modules/project/app/lock.go
		return held{}, err
	case !found:
		return held{}, domain.ErrNotFound
	}
	var h held
````
````new server/internal/modules/project/app/lock.go
		return held{}, err
	case !found:
		return held{}, notFound
	}
	var h held
````

````old server/internal/modules/project/app/lock.go
		return held{}, domain.ErrNotFound
	}
	if h.grant, err = decide(ctx, l.auth, actor, w.action, workspaceID, w.project); err != nil {
		return held{}, err
````
````new server/internal/modules/project/app/lock.go
		return held{}, notFound
````

````old server/internal/modules/project/app/lock.go
	_, err = decide(ctx, auth, actor, action, workspaceID, id)
````
````new server/internal/modules/project/app/lock.go
	_, err = decide(ctx, auth, actor, action, workspaceID, id, domain.ErrNotFound)
````

````old server/internal/modules/project/app/lock.go
// project not visible to actor is domain.ErrNotFound, the 404 of what the
// caller named.
func decide(ctx context.Context, auth shared.Authorizer, actor shared.Actor, action shared.Action, workspaceID, id uuid.UUID) (shared.Grant, error) {
````
````new server/internal/modules/project/app/lock.go
// project not visible to actor is notFound, the 404 of what the caller
// named.
func decide(ctx context.Context, auth shared.Authorizer, actor shared.Actor, action shared.Action, workspaceID, id uuid.UUID,
	notFound error) (shared.Grant, error) {
````

````old server/internal/modules/project/app/lock.go
		return shared.Grant{}, domain.ErrNotFound
````
````new server/internal/modules/project/app/lock.go
		return shared.Grant{}, notFound
````

`server/internal/modules/project/app/get_project.go`（修改，1 处）：

````old server/internal/modules/project/app/get_project.go
	if _, err = decide(ctx, u.auth, actor, domain.ActionRead, p.WorkspaceID, p.ID); err != nil {
````
````new server/internal/modules/project/app/get_project.go
	if _, err = decide(ctx, u.auth, actor, domain.ActionRead, p.WorkspaceID, p.ID, domain.ErrNotFound); err != nil {
````

- [ ] **Step 2: 用例**

`server/internal/modules/project/app/update_member.go`（新文件，77 行）：

````file server/internal/modules/project/app/update_member.go
package app

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateProjectMember changes a project member's role: PATCH
// /api/v0/project-members/{project_member_id} (M3 design 3.5).
type UpdateProjectMember struct {
	locks   Locks
	members MemberRoleChanger
	tx      shared.TxManager
	clock   Clock
}

// NewUpdateProjectMember returns the use case.
func NewUpdateProjectMember(locks Locks, members MemberRoleChanger, tx shared.TxManager, clock Clock) *UpdateProjectMember {
	return &UpdateProjectMember{locks: locks, members: members, tx: tx, clock: clock}
}

// Execute checks role, then in one transaction, in the order of M3 design
// 3.6: the membership's locks (Locks.lockMemberAndDecide: the membership
// read for its project and workspace, the workspace FOR SHARE, its member's
// membership of the workspace FOR SHARE, the project FOR NO KEY UPDATE, the
// membership read again) and the decision on project_member.update; then
// the checks that only a caller allowed to change roles gets to see: an
// ended membership is project.member_not_found; then 3.5's rules
// (domain.CheckRoleChange) on the caller's grant, whether the membership is
// his own, the member's role, his workspace role as locked, and the role
// given; then the change, at the time the clock gives under the locks. The
// answer is the membership as stored.
func (u *UpdateProjectMember) Execute(ctx context.Context, id uuid.UUID, role shared.Role) (domain.Member, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Member{}, err
	}
	if err := domain.CheckMemberRole(role); err != nil {
		return domain.Member{}, err
	}
	var updated domain.Member
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		h, err := u.locks.lockMemberAndDecide(ctx, actor, id, domain.ActionMemberUpdate, true)
		if err != nil {
			return err
		}
		m := h.member
		if !m.Active {
			return domain.ErrMemberNotFound
		}
		workspaceRole, ok := h.roles[m.MemberID]
		if !ok {
			// Every shrinking of a workspace membership ends its project
			// memberships (3.6 convention 6): its absence is a bug.
			return fmt.Errorf("project membership %s: %s is no active member of the workspace", m.ID, m.MemberID)
		}
		if err := domain.CheckRoleChange(domain.RoleChange{Caller: h.grant, Own: m.MemberID == actor.UserID, From: m.Role,
			WorkspaceRole: workspaceRole, To: role}); err != nil {
			return err
		}
		if updated, err = u.members.UpdateMemberRole(ctx, m.ID, role, actor.UserID, u.clock.Now()); err != nil {
			return err
		}
		if updated.ID != m.ID {
			return fmt.Errorf("project membership %s changed as %s", m.ID, updated.ID)
		}
		return nil
	})
	if err != nil {
		return domain.Member{}, err
	}
	return updated, nil
}
````

- [ ] **Step 3: 假对象和测试**

`server/internal/modules/project/app/fakes_test.go`（修改，4 处）：

````old server/internal/modules/project/app/fakes_test.go
// commit failing after fn succeeded.
````
````new server/internal/modules/project/app/fakes_test.go
// commit failing after fn succeeded; returned is what fn returned.
````

````old server/internal/modules/project/app/fakes_test.go
	commitErr error
````
````new server/internal/modules/project/app/fakes_test.go
	commitErr error
	returned  error
````

````old server/internal/modules/project/app/fakes_test.go
	if err := fn(context.WithValue(ctx, inTxKey{}, true)); err != nil {
		return err
````
````new server/internal/modules/project/app/fakes_test.go
	if f.returned = fn(context.WithValue(ctx, inTxKey{}, true)); f.returned != nil {
		return f.returned
````

````old server/internal/modules/project/app/fakes_test.go
	return f.commitErr
````
````new server/internal/modules/project/app/fakes_test.go
	return f.commitErr
}

// answered reports whether err, a use case's answer, came out of the
// transaction as itself: what fn returned, which the transaction rolled
// back on; with commitErr set, the commit's failure, fn having returned
// nil.
func (f *fakeTx) answered(err error) bool {
	if f.commitErr != nil {
		return f.returned == nil && err == f.commitErr
	}
	return f.returned != nil && err == f.returned
````

`server/internal/modules/project/app/fakes_write_test.go`（修改，4 处）：

````old server/internal/modules/project/app/fakes_write_test.go
var webID, opsID = uuid.NewV7(), uuid.NewV7()

````
````new server/internal/modules/project/app/fakes_write_test.go
var webID, opsID = uuid.NewV7(), uuid.NewV7()

// The memberships of web and ops, by id, the same in every fixture.
var bobInWeb, aliceInWeb, carolInWeb, daveInWeb, bobInOps = uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()

````

````old server/internal/modules/project/app/fakes_write_test.go
				bob:   {ID: uuid.NewV7(), Role: shared.RoleAdmin, Active: true},
				alice: {ID: uuid.NewV7(), Role: shared.RoleMember, Active: true},
				carol: {ID: uuid.NewV7(), Role: shared.RoleGuest, Active: true},
				dave:  {ID: uuid.NewV7(), Role: shared.RoleMember},
````
````new server/internal/modules/project/app/fakes_write_test.go
				bob:   {ID: bobInWeb, Role: shared.RoleAdmin, Active: true},
				alice: {ID: aliceInWeb, Role: shared.RoleMember, Active: true},
				carol: {ID: carolInWeb, Role: shared.RoleGuest, Active: true},
				dave:  {ID: daveInWeb, Role: shared.RoleMember},
````

````old server/internal/modules/project/app/fakes_write_test.go
				bob: {ID: uuid.NewV7(), Role: shared.RoleAdmin, Active: true},
````
````new server/internal/modules/project/app/fakes_write_test.go
				bob: {ID: bobInOps, Role: shared.RoleAdmin, Active: true},
````

````old server/internal/modules/project/app/fakes_write_test.go
	deleted  bool // each project's lock finds nothing, as if it was deleted while the lock waited
````
````new server/internal/modules/project/app/fakes_write_test.go
	deleted  bool // each project's lock finds nothing, as if it was deleted while the lock waited
	// The reads of a membership by its id (fakes_member_test.go): how many
	// ran, how the second one answers, and answersAs, when set, the id each
	// answers for the one asked; changedAs, when set, is the id
	// UpdateMemberRole answers for the one it changed.
	memberReadCount int
	reread          memberReads
	answersAs       uuid.UUID
	changedAs       uuid.UUID
````

`server/internal/modules/project/app/fakes_create_test.go`（修改，3 处）：

````old server/internal/modules/project/app/fakes_create_test.go
// only of the accounts asked for; it logs each call and fails with err.
````
````new server/internal/modules/project/app/fakes_create_test.go
// only of the accounts asked for; it logs each call and fails with err.
// answersFor, when set, is the account it answers each role for.
````

````old server/internal/modules/project/app/fakes_create_test.go
	log   *callLog
	roles map[uuid.UUID]map[uuid.UUID]shared.Role // workspace → account → role
	err   error
````
````new server/internal/modules/project/app/fakes_create_test.go
	log        *callLog
	roles      map[uuid.UUID]map[uuid.UUID]shared.Role // workspace → account → role
	err        error
	answersFor uuid.UUID
````

````old server/internal/modules/project/app/fakes_create_test.go
			out[id] = role
````
````new server/internal/modules/project/app/fakes_create_test.go
			key := id
			if f.answersFor != (uuid.UUID{}) {
				key = f.answersFor
			}
			out[key] = role
````

`server/internal/modules/project/app/fakes_member_test.go`（新文件，171 行）：

````file server/internal/modules/project/app/fakes_member_test.go
package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeStore's reads and writes of one project membership: it logs each
// call, finds a membership among its projects' by id, and writes into it.

// memberSince is when the fakes' memberships were made, as stored: no
// clock's time.
var memberSince = time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)

// membership finds the membership id among the projects': its project, its
// member and itself.
func (f *fakeStore) membership(id uuid.UUID) (project, user uuid.UUID, m app.Membership, ok bool) {
	for pid, p := range f.projects {
		for uid, m := range p.members {
			if m.ID == id {
				return pid, uid, m, true
			}
		}
	}
	return uuid.UUID{}, uuid.UUID{}, app.Membership{}, false
}

// memberReads is how a membership read again under the locks answers: err
// fails it, gone finds none, as if it was deleted meanwhile, project,
// when set, is the project it is found in, as if it had moved there, and
// role, when set, its role, as if it had been changed meanwhile.
type memberReads struct {
	err     error
	gone    bool
	project uuid.UUID
	role    shared.Role
}

func (f *fakeStore) MemberByID(ctx context.Context, id uuid.UUID) (app.ProjectMembership, bool, error) {
	f.log.add(ctx, "MemberByID %s", id)
	f.memberReadCount++
	again := f.memberReadCount > 1
	if err := f.fail("MemberByID"); err != nil {
		return app.ProjectMembership{}, false, err
	}
	if again && f.reread.err != nil {
		return app.ProjectMembership{}, false, fmt.Errorf("MemberByID: %w", f.reread.err)
	}
	project, user, m, ok := f.membership(id)
	if !ok || (again && f.reread.gone) {
		return app.ProjectMembership{}, false, nil
	}
	out := app.ProjectMembership{ID: m.ID, WorkspaceID: f.projects[project].workspace, ProjectID: project, MemberID: user, Role: m.Role,
		Active: m.Active}
	if again && f.reread.project != (uuid.UUID{}) {
		out.ProjectID = f.reread.project
	}
	if again && f.reread.role != 0 {
		out.Role = f.reread.role
	}
	if f.answersAs != (uuid.UUID{}) {
		out.ID = f.answersAs
	}
	return out, true, nil
}

func (f *fakeStore) UpdateMemberRole(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) (domain.Member, error) {
	f.log.add(ctx, "UpdateMemberRole %s as %d by %s at %s", id, role, by, now.Format(timeFormat))
	if err := f.fail("UpdateMemberRole"); err != nil {
		return domain.Member{}, err
	}
	project, user, m, ok := f.membership(id)
	if !ok || !m.Active {
		return domain.Member{}, fmt.Errorf("UpdateMemberRole: no active membership %s", id)
	}
	m.Role = role
	f.projects[project].members[user] = m
	answer := domain.Member{ID: id, ProjectID: project, MemberID: user, Role: role, CreatedAt: memberSince}
	if f.changedAs != (uuid.UUID{}) {
		answer.ID = f.changedAs
	}
	return answer, nil
}

// frank is a project admin in newMemberWrites; the memberships it adds,
// by id, the same in every fixture.
var (
	frank                           = uuid.NewV7()
	ginaInWeb, frankInWeb, ivyInWeb = uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
)

// newMemberWrites is newGrowth with more members of web: gina, the
// workspace's admin, its member; frank, the workspace's member, a second
// admin; ivy, the workspace's guest, its guest. gina's grant in acme is a
// workspace admin's who is a project member.
func newMemberWrites() *writeFixture {
	f := newGrowth()
	web := f.store.projects[webID]
	web.members[gina] = app.Membership{ID: ginaInWeb, Role: shared.RoleMember, Active: true}
	web.members[frank] = app.Membership{ID: frankInWeb, Role: shared.RoleAdmin, Active: true}
	web.members[ivy] = app.Membership{ID: ivyInWeb, Role: shared.RoleGuest, Active: true}
	f.members.roles[acme.ID][frank] = shared.RoleMember
	f.auth.grants[grantKey{gina, acme.ID}] = shared.Grant{WorkspaceRole: shared.RoleAdmin, ProjectRole: shared.RoleMember, ProjectAdmin: true}
	return f
}

// memberOf is user's membership of project in f, by id.
func (f *writeFixture) memberOf(project, user uuid.UUID) uuid.UUID {
	return f.store.projects[project].members[user].ID
}

// memberLocked are the calls of a write by caller on the membership id,
// user's of project, up to its decision on action: the transaction, the
// membership read, acme's row FOR SHARE, user's membership of acme FOR
// SHARE when target is set, the project FOR NO KEY UPDATE, the membership
// read again, the decision.
func memberLocked(id, user, project, caller uuid.UUID, action shared.Action, target bool) []string {
	calls := []string{"Begin", "MemberByID " + id.String(), "ShareWorkspaceByID " + acme.ID.String()}
	if target {
		calls = append(calls, fmt.Sprintf("ShareMembers %s %v", acme.ID, []uuid.UUID{user}))
	}
	return append(calls, "LockProject "+project.String(), "MemberByID "+id.String(),
		fmt.Sprintf("Authorize %s %s on %s/%s", caller, action, acme.ID, project))
}

// outcome is how a write ends in a case: want is the refusal the API
// answers (a *shared.Error), a failure that comes back as itself (errDisk),
// or nil for the write's own error, neither of them: a 500. calls are the
// calls up to the end.
type outcome struct {
	name  string
	want  error
	calls []string
}

// check fails the test unless err, the write's answer, is tt's: a refusal
// is the first *shared.Error in err's chain, the one the API answers, of
// want's kind, code and fields (sameError); a failure is want, and the
// write's own error is an error, each with no *shared.Error in the chain.
// Once the write began its transaction, err is what came out of it
// (fakeTx.answered). The calls are tt.calls, each once, in order.
func (tt outcome) check(t *testing.T, err error, f *writeFixture) {
	t.Helper()
	var se, refusal *shared.Error
	switch {
	case errors.As(tt.want, &refusal):
		if !sameError(err, tt.want) {
			t.Errorf("%s: Execute() = %v, answered as another problem; want %v", tt.name, err, tt.want)
		}
	case errors.As(err, &se):
		t.Errorf("%s: Execute() = %v, answered as %s; want a 500", tt.name, err, se.Code)
	case err == nil, tt.want != nil && !errors.Is(err, tt.want):
		t.Errorf("%s: Execute() = %v; want %v, or an error of the write's own for nil", tt.name, err, tt.want)
	}
	if len(tt.calls) > 0 && !f.tx.answered(err) {
		t.Errorf("%s: Execute() = %v, the transaction's function returned %v, its commit failing with %v; want the answer to come out "+
			"of the transaction as itself", tt.name, err, f.tx.returned, f.tx.commitErr)
	}
	if !slices.Equal(f.log.calls, tt.calls) {
		t.Errorf("%s: calls\n%q\nwant\n%q", tt.name, f.log.calls, tt.calls)
	}
}
````

`server/internal/modules/project/app/update_member_test.go`（新文件，175 行）：

````file server/internal/modules/project/app/update_member_test.go
package app_test

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newUpdateMember is UpdateProjectMember over newMemberWrites' fakes, its
// clock logged.
func newUpdateMember() (*app.UpdateProjectMember, *writeFixture) {
	f := newMemberWrites()
	return app.NewUpdateProjectMember(f.locks(), f.store, f.tx, clockAt{clockNow, f.log}), f
}

// roleChanged are the calls of caller's change of user's membership of web to
// role: its locks and decision, the clock, the change at that time.
func roleChanged(f *writeFixture, caller, user uuid.UUID, role shared.Role) []string {
	id := f.memberOf(webID, user)
	return append(memberLocked(id, user, webID, caller, domain.ActionMemberUpdate, true), "Now",
		fmt.Sprintf("UpdateMemberRole %s as %d by %s at %s", id, role, caller, clockNow.Format(timeFormat)))
}

// UpdateProjectMember, in one transaction and in the order of M3 design
// 3.6, takes the membership's locks, decides, reads the clock, then changes
// the role, by the caller at that time, and answers the membership as
// stored. bob, a project admin who is a workspace member, makes alice, a
// member, a guest, and keeps carol, a workspace guest, a guest; gina, the
// workspace's admin and a project member, makes herself an admin, frank, an
// admin, a member, and alice an admin (3.5's exception).
func TestUpdateProjectMember(t *testing.T) {
	for _, tt := range []struct {
		name         string
		caller, user uuid.UUID
		role         shared.Role
	}{
		{"bob makes alice a guest", bob, alice, shared.RoleGuest},
		{"bob keeps carol a guest", bob, carol, shared.RoleGuest},
		{"gina makes herself an admin", gina, gina, shared.RoleAdmin},
		{"gina makes frank a member", gina, frank, shared.RoleMember},
		{"gina makes alice an admin", gina, alice, shared.RoleAdmin},
	} {
		uc, f := newUpdateMember()
		id, calls := f.memberOf(webID, tt.user), roleChanged(f, tt.caller, tt.user, tt.role)
		m, err := uc.Execute(as(tt.caller), id, tt.role)
		if want := (domain.Member{ID: id, ProjectID: webID, MemberID: tt.user, Role: tt.role, CreatedAt: memberSince}); err != nil || m != want {
			t.Errorf("%s: Execute() = %+v, %v; want %+v", tt.name, m, err, want)
		}
		if !slices.Equal(f.log.calls, calls) {
			t.Errorf("%s: calls\n%q\nwant\n%q", tt.name, f.log.calls, calls)
		}
	}
}

// Refusals, each in its place, and nothing changed: no caller, and a role
// outside the three, before the transaction; a membership that is not
// there, a workspace, project or membership deleted while a lock waited, a
// project or membership moved meanwhile, and a caller who does not see the
// project, each project.member_not_found, the last after the decision; a
// project member, the Authorizer's 403. Then, to a caller who may change
// roles, after the decision: an ended membership, 404; his own, 409; an
// admin's role and a role of admin, from a project admin who is no
// workspace admin, 403 project.role_too_high; a workspace guest made more
// than a guest, 422, from anyone. A member who is no active member of the
// workspace, or whose role is answered for another account, and a
// membership answered for another id, are the write's own error.
func TestUpdateProjectMemberRefuses(t *testing.T) {
	guestOnly := shared.Invalid(shared.FieldError{Field: "role", Code: shared.FieldNotAllowed, Message: "must be 5: the member is a guest of the workspace"})
	locked := func(f *writeFixture, caller, user uuid.UUID) []string {
		return memberLocked(f.memberOf(webID, user), user, webID, caller, domain.ActionMemberUpdate, true)
	}
	// upTo is the first n calls of caller's write on user's membership.
	upTo := func(n int) func(f *writeFixture, caller, user uuid.UUID) []string {
		return func(f *writeFixture, caller, user uuid.UUID) []string { return locked(f, caller, user)[:n] }
	}
	none := func(*writeFixture, uuid.UUID, uuid.UUID) []string { return nil }
	for _, tt := range []struct {
		name         string
		caller, user uuid.UUID // no caller when caller is zero; a membership not there when user is zero
		role         shared.Role
		set          func(f *writeFixture)
		want         error
		calls        func(f *writeFixture, caller, user uuid.UUID) []string
	}{
		{"no caller", uuid.UUID{}, alice, shared.RoleGuest, nil, shared.Unauthenticated(), none},
		{"a role outside the three", bob, alice, 10, nil,
			shared.Invalid(shared.FieldError{Field: "role", Code: shared.FieldInvalidFormat, Message: "is not 5, 15 or 20"}), none},
		{"no membership", bob, uuid.UUID{}, shared.RoleGuest, nil, domain.ErrMemberNotFound,
			func(*writeFixture, uuid.UUID, uuid.UUID) []string {
				return []string{"Begin", "MemberByID " + uuid.Nil().String()}
			}},
		{"acme deleted while its lock waited", bob, alice, shared.RoleGuest, func(f *writeFixture) { f.workspaces.gone = true },
			domain.ErrMemberNotFound, upTo(3)},
		{"web deleted while its lock waited", bob, alice, shared.RoleGuest, func(f *writeFixture) { f.store.deleted = true },
			domain.ErrMemberNotFound, upTo(5)},
		{"web moved to another workspace", bob, alice, shared.RoleGuest, func(f *writeFixture) { f.store.moved = uuid.NewV7() },
			domain.ErrMemberNotFound, upTo(5)},
		{"the membership deleted while the locks waited", bob, alice, shared.RoleGuest, func(f *writeFixture) { f.store.reread.gone = true },
			domain.ErrMemberNotFound, upTo(6)},
		{"the membership moved to ops", bob, alice, shared.RoleGuest, func(f *writeFixture) { f.store.reread.project = opsID },
			domain.ErrMemberNotFound, upTo(6)},
		{"a caller who does not see web", erin, alice, shared.RoleGuest, nil, domain.ErrMemberNotFound, locked},
		{"a project member", alice, carol, shared.RoleGuest, nil, shared.Forbidden(), locked},
		{"an ended membership", bob, dave, shared.RoleGuest, nil, domain.ErrMemberNotFound, locked},
		{"his own", bob, bob, shared.RoleMember, nil, domain.ErrOwnMembership, locked},
		{"another admin", bob, frank, shared.RoleMember, nil, domain.ErrRoleTooHigh, locked},
		{"a member made an admin while the locks waited", bob, alice, shared.RoleGuest,
			func(f *writeFixture) { f.store.reread.role = shared.RoleAdmin }, domain.ErrRoleTooHigh, locked},
		{"a member made an admin", bob, alice, shared.RoleAdmin, nil, domain.ErrRoleTooHigh, locked},
		{"a workspace guest made a member", bob, ivy, shared.RoleMember, nil, guestOnly, locked},
		{"a workspace guest made an admin by the workspace's admin", gina, ivy, shared.RoleAdmin, nil, guestOnly, locked},
		{"a member who is no active member of the workspace", bob, alice, shared.RoleGuest,
			func(f *writeFixture) { delete(f.members.roles[acme.ID], alice) }, nil, locked},
		{"his workspace role answered for another account", bob, alice, shared.RoleGuest,
			func(f *writeFixture) { f.members.answersFor = carol }, nil, locked},
		{"a membership answered for another id", bob, alice, shared.RoleGuest, func(f *writeFixture) { f.store.answersAs = uuid.NewV7() },
			nil, upTo(2)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newUpdateMember()
			if tt.set != nil {
				tt.set(f)
			}
			ctx, id := context.Background(), uuid.Nil()
			if tt.caller != (uuid.UUID{}) {
				ctx = as(tt.caller)
			}
			if tt.user != (uuid.UUID{}) {
				id = f.memberOf(webID, tt.user)
			}
			before := f.store.projects[webID].members[tt.user]
			_, err := uc.Execute(ctx, id, tt.role)
			outcome{tt.name, tt.want, tt.calls(f, tt.caller, tt.user)}.check(t, err, f)
			if after := f.store.projects[webID].members[tt.user]; after != before {
				t.Errorf("the membership after the refusal: %+v, want it as it was, %+v", after, before)
			}
		})
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after; a change answered for another membership
// is the write's own error.
func TestUpdateProjectMemberReturnsEachFailure(t *testing.T) {
	for _, tt := range []struct {
		name  string
		fail  func(f *writeFixture)
		want  error
		calls int // how many of the change's calls ran
	}{
		{"the read", func(f *writeFixture) { f.store.errs = map[string]error{"MemberByID": errDisk} }, errDisk, 2},
		{"the workspace's lock", func(f *writeFixture) { f.workspaces.err = errDisk }, errDisk, 3},
		{"the member's workspace membership", func(f *writeFixture) { f.members.err = errDisk }, errDisk, 4},
		{"the project's lock", func(f *writeFixture) { f.store.errs = map[string]error{"LockProject": errDisk} }, errDisk, 5},
		{"the read under the locks", func(f *writeFixture) { f.store.reread.err = errDisk }, errDisk, 6},
		{"the decision", func(f *writeFixture) { f.auth.errs = map[grantKey]error{{bob, acme.ID}: errDisk} }, errDisk, 7},
		{"the change", func(f *writeFixture) { f.store.errs = map[string]error{"UpdateMemberRole": errDisk} }, errDisk, 9},
		{"the change answered for another membership", func(f *writeFixture) { f.store.changedAs = uuid.NewV7() }, nil, 9},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, errDisk, 9},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newUpdateMember()
			tt.fail(f)
			all := roleChanged(f, bob, alice, shared.RoleGuest)
			_, err := uc.Execute(as(bob), f.memberOf(webID, alice), shared.RoleGuest)
			outcome{tt.name, tt.want, all[:tt.calls]}.check(t, err, f)
		})
	}
}
````

`server/internal/modules/project/app/clock_test.go`（修改，1 处）：

````old server/internal/modules/project/app/clock_test.go
			[]string{"GetProject " + webID.String() + " for " + hank.String()})},
````
````new server/internal/modules/project/app/clock_test.go
			[]string{"GetProject " + webID.String() + " for " + hank.String()})},
		{"updateProjectMember", func() ([]string, error) {
			uc, f := newUpdateMember()
			_, err := uc.Execute(as(bob), aliceInWeb, shared.RoleGuest)
			return f.log.calls, err
		}, roleChanged(newMemberWrites(), bob, alice, shared.RoleGuest)},
````

- [ ] **Step 4: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/project/...`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 5: 提交**

```bash
git add server/internal/modules/project/app/clock_test.go server/internal/modules/project/app/fakes_create_test.go server/internal/modules/project/app/fakes_member_test.go server/internal/modules/project/app/fakes_test.go server/internal/modules/project/app/fakes_write_test.go server/internal/modules/project/app/get_project.go server/internal/modules/project/app/lock.go server/internal/modules/project/app/ports.go server/internal/modules/project/app/update_member.go server/internal/modules/project/app/update_member_test.go server/internal/modules/project/domain/actions.go
```
```bash
git commit -m "feat(M3/P5b): Locks for a write named by a project membership's id, and updateProjectMember's use case

A write on a membership reads it unlocked for its project and
workspace, shares the workspace, and the member's membership of it for
a role change, locks the project, reads the membership again, still of
that project, then decides: each of the membership gone, its project
or workspace deleted, or a project not seen is project.member_not_found.
updateProjectMember then refuses an ended membership, holds the caller
to the relative rule, and changes the role at the clock's time under
the locks.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A，清扫 2、9、11、12、13、21、22、24）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| 在锁之前判定 | `TestUpdateProjectMemberRefuses`、`TestUpdateProjectMemberReturnsEachFailure`；`TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile`（Task 8 起，调用者的成员关系结束的情形） | 单元；组合 |
| 不在锁下重读；重读挪到锁之前；重读之后不核对项目 | `TestUpdateProjectMemberRefuses`；`TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile`（成员关系结束、删除、移到 Ops）（故事看不到：故事是顺序的，两次读相同） | 单元；组合 |
| 相对规则用锁之前读到的角色（`h.member.Role = m.Role`）；重读挪到判定之后 | `TestUpdateProjectMemberRefuses`；`TestAWriteOnAProjectMembershipDecidesOnWhatItReadUnderItsLocks`（Task 8 起） | 单元；组合 |
| 已结束的成员关系照样改 | `TestUpdateProjectMemberRefuses`；`TestPermissionMatrix`（Task 4 起，已结束的一行）、`TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile` | 单元；组合 |
| 等锁时删除的工作区不算 404 | `TestUpdateProjectMemberRefuses`、`TestLeaveProjectRefuses`（组合一层等价：删除工作区在同一个事务里删除它的项目，项目的锁随即读不到行，同样 404；spec 第 3 节第 8 条） | 单元 |
| 项目在工作区之前锁；成员关系的第一次读带 `FOR UPDATE`；成员在工作区的成员关系在项目之后锁 | `TestUpdateProjectMemberRefuses`（调用顺序）；`TestEachWriteOnAProjectSharesItsWorkspaceFirst`、`TestEachLockOfAWriteOnAProjectMembershipIsItsStrength`（Task 4、8 起） | 单元；组合 |
| 项目锁成 `FOR SHARE`；改角色不锁成员在工作区的成员关系 | `TestUpdateProjectMember`、`TestUpdateProjectMemberRefuses`（调用记录里的锁）；`TestEachLockOfAWriteOnAProjectMembershipIsItsStrength`（Task 8 起） | 单元；组合 |
| 已结束的在判定之前答 404 | `TestUpdateProjectMemberRefuses`；`TestPermissionMatrix`（Task 4 起，已结束的一行：PM 得到 403） | 单元；组合 |
| 判定的失败答成 404 | `TestUpdateProjectMemberReturnsEachFailure`；`TestPermissionMatrix`、`TestTheRelativeRuleOnTheComposedApp`（Task 4、5 起：拒绝也经这条路） | 单元；组合 |
| 读的失败答成 404；锁的失败答成 404；锁下重读失败时重试；改的失败被吞掉、被包进 404 | `TestUpdateProjectMemberReturnsEachFailure`（组合一层无从注入存储的失败） | 单元 |
| 回答别的 id 的成员关系照样往下；改的回答是别的成员关系照样回答 | `TestUpdateProjectMemberRefuses`、`TestUpdateProjectMemberReturnsEachFailure` | 单元 |
| 工作区访客的上限读调用者的工作区角色 | `TestUpdateProjectMemberRefuses`；`TestPermissionMatrix`（Task 4 起）、`TestTheRelativeRuleOnTheComposedApp`（Task 5 起） | 单元；组合 |
| 改角色看不到自己的成员关系 | `TestUpdateProjectMemberRefuses`；`TestPermissionMatrix`、`TestTheRelativeRuleOnTheComposedApp` | 单元；组合 |
| 写成成员自己改的 | `TestUpdateProjectMember`；`TestTheRelativeRuleOnTheComposedApp`；P5 | 单元；组合；端到端 |

**Done when:** 改角色的用例通过单元测试；按项目 id 的写照旧通过（`lock` 共用）。

---

### Task 4: `updateProjectMember` 的接口、规则、矩阵；每个项目级的写最先锁工作区的测试接资源路径

**Files:**
- Create: `server/internal/bootstrap/permission_matrix_membership_test.go`、`server/internal/modules/project/adapter/http/member_writes_test.go`
- Modify: `api/modules/project.yaml`、`api/openapi.yaml`、`server/internal/bootstrap/permission_matrix_columns_test.go`、`server/internal/bootstrap/permission_matrix_seed_test.go`、`server/internal/bootstrap/permission_matrix_seeded_test.go`、`server/internal/bootstrap/permission_matrix_targets_test.go`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/bootstrap/project_write_locks_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/project/adapter/http/handler.go`、`server/internal/modules/project/adapter/http/handler_test.go`、`server/internal/modules/project/adapter/http/members.go`、`server/internal/modules/project/domain/actions.go`、`server/internal/modules/project/module.go`、`web/apps/web/helpers/authentication.helper.ts`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/project/adapter/http/gen/bodyshape.gen.go`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.7，M3 设计 5.1、5.3、9.2、9.4）：`PATCH /api/v0/project-members/{project_member_id}`，正文 `ProjectMemberUpdate{role}`（`additionalProperties: false`），200 回答 `ProjectMember`；码 `[validation_failed, project.member_not_found, forbidden, project.own_membership, project.role_too_high]`。参数 `ProjectMemberID`。三个新码进 `PROBLEM_MESSAGES` 和两份 `auth.json`（约束 4）。
- `access/domain` 的规则 `"project_member.update": {Level: LevelProject, Roles: [admin]}`（工作区管理员是项目成员时由 `LevelProject` 的工作区管理员规则放行，同 `project_member.add`）；`Actions()` 加 `ActionMemberUpdate`。
- HTTP：`httpadapter.UpdateMemberUseCase`；`UseCases.UpdateMember`；`(handler).UpdateProjectMember`（角色原样交给用例，三者之外的也是，领域答 422）；`member(m)` 把 `domain.Member` 写成接口的形状，列表和改角色共用。`project.New` 的 `UpdateMember: app.NewUpdateProjectMember(locks, store, d.Tx, d.Clock)`。
- 矩阵（`permission_matrix_membership_test.go`）：`ofMembership(pa, pm, pg, pmwa, wa, wm)`（WM-私、WG-、P-前、X 是 404 `project.member_not_found`）；`aMembership`（`projectMemberOf`、`ownMembershipOf`、`endedMembershipOf`、`workspaceGuestOf`）；`toProjectMembership(method, body, which)`；四行改角色：PM 的成员关系改为访客 `ofMembership(200, 403, 403, 200, 403, 403)`；自己的改为管理员 `(409 own_membership, 403, 403, 200, 403, 403)`；已结束的 `(404, 403, 403, 404, 403, 403)`；工作区访客改为成员 `(422 role not_allowed, 403, 403, 422, 403, 403)`；各 12 格。`projectSeed.memberships` 是它们的前提（每个被指到的成员关系的状态、角色；gone 的已删除；PG 的账户是 acme 的有效访客）。`seeded.projectMember(key, c)`、`projectOfMember(id)`；`projectSeed.join` 取 id；gone 的项目多一个成员（gone 的成员），X 的写在那里有一个成员关系可指。
- 目标核对（`matrixViolations`）：`{project_member_id}` 必须是这一列的项目里种下的成员关系，且这一列属于项目的表（`ofAProjectTable`）。
- `TestEachWriteOnAProjectSharesItsWorkspaceFirst`：`projectWrite.param()`（资源路径是 `{project_member_id}`）；`by [2]string`（取代 `byTarget`）；`member [2]string`（写改的成员关系：资源路径指它，第一步用 `FOR UPDATE NOWAIT` 探测它的行，第二步核对它在项目之后才锁）；改角色一行：bob 在 Web、carol 在 Ops 改为访客。

**Tests:**
- `adapter/http/member_writes_test.go`：`TestUpdateProjectMember`（路径的成员关系、角色交给用例，三者之外的角色也是；200 回答用例的成员关系）；`TestUpdateProjectMemberHoldsTheBodyToItsStructure`（没有角色、`null`、字符串、多一个字段：400，不到用例）；`TestUpdateProjectMemberRefusals`（用例的每个拒绝照契约答，失败 500）。
- `access/domain/rules_test.go`：`TestEveryRuleDecidesItsCells` 的 `project_member.update` 一行（同 `project_member.add`）。
- 矩阵：四行 48 格；`TestMatrixViolationsCatchesEachColumnGap` 加四个情形（列的项目里的成员关系通过；别的列的项目的、没有种下的 id、工作区一级的行里的各报一条）；`seeded.projectMember` 没有种下时立即失败。
- `TestEachWriteOnAProjectSharesItsWorkspaceFirst` 的改角色一行；完整性核对少了它就失败（规则 (b)：路径里没有 `{project_id}`，它的矩阵行的列是项目一级的；spec 2.13）。

- [ ] **Step 1: 接口描述**

`api/modules/project.yaml`（修改，4 处）：

````old api/modules/project.yaml
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/me/projects/{project_id}/preferences:
````
````new api/modules/project.yaml
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/project-members/{project_member_id}:
    parameters:
      - $ref: '#/components/parameters/ProjectMemberID'
    patch:
      operationId: updateProjectMember
      tags: [project]
      summary: Change a project member's role
      description: >-
        For the project's admins, and its members who are the workspace's
        admins. The role is checked first (validation_failed). A membership
        that does not exist or is deleted, or whose project the caller does
        not see, answers project.member_not_found; a caller who sees the
        project but may not change roles in it, forbidden, whatever the
        membership. To a caller who may change roles, a membership that has
        ended answers project.member_not_found. One who is not a workspace
        admin cannot change his own role (project.own_membership), nor the
        role of a member whose role is not below his own, nor give a role
        that is not below his own (project.role_too_high): a project admin
        who is not a workspace admin makes nobody an admin and changes no
        admin's role. A workspace guest's role stays a guest's, whoever
        changes it (validation_failed, role not_allowed). The role is
        decided after the workspace and project rows are locked.
      security: [{bearer: []}]
      x-problem-codes: [validation_failed, project.member_not_found, forbidden, project.own_membership, project.role_too_high]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ProjectMemberUpdate'
      responses:
        '200':
          description: The membership with its new role.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ProjectMember'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/me/projects/{project_id}/preferences:
````

````old api/modules/project.yaml
      description: A project's id (Project.id).
````
````new api/modules/project.yaml
      description: A project's id (Project.id).
      schema:
        type: string
        format: uuid
    ProjectMemberID:
      name: project_member_id
      in: path
      required: true
      description: A project membership's id (ProjectMember.id), not the member's account id.
````

````old api/modules/project.yaml
          description: The membership's id.
````
````new api/modules/project.yaml
          description: The membership's id, which /project-members/{project_member_id} names.
````

````old api/modules/project.yaml
          description: An active member of the workspace.
          type: string
          format: uuid
        role:
          $ref: '#/components/schemas/ProjectRole'

````
````new api/modules/project.yaml
          description: An active member of the workspace.
          type: string
          format: uuid
        role:
          $ref: '#/components/schemas/ProjectRole'
    ProjectMemberUpdate:
      type: object
      additionalProperties: false
      required: [role]
      properties:
        role:
          $ref: '#/components/schemas/ProjectRole'

````

`api/openapi.yaml`（修改，1 处）：

````old api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1projects~1{project_id}~1members'
````
````new api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1projects~1{project_id}~1members'
  /api/v0/project-members/{project_member_id}:
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1project-members~1{project_member_id}'
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `8a9f44eaca9f2a8424c2834dcba1c60f1971f4dddb35e4c1f5db945ad1c7ab28` | 2460 | `api/dist/openapi.yaml` |
| `7431031da1b486c18763cd08aff64c372b7b425ef334f87240cbec6101d44774` | 62 | `server/internal/modules/project/adapter/http/gen/bodyshape.gen.go` |
| `d1938426e812eb98620399f2ade4cd864182eec5dd2cade685bc1928ae310ebd` | 2089 | `server/internal/modules/project/adapter/http/gen/server.gen.go` |
| `2b4c499dcf0fd45656b8dfbdec629327882122c0fb63c4e91cf06e54e69fecbf` | 2681 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/bodyshape.gen.go server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: 规则、handler、接线、文案**

`server/internal/modules/project/domain/actions.go`（修改，1 处）：

````old server/internal/modules/project/domain/actions.go
		ActionMemberList, ActionMemberAdd, ActionJoin}
````
````new server/internal/modules/project/domain/actions.go
		ActionMemberList, ActionMemberAdd, ActionJoin, ActionMemberUpdate}
````

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"project.join": {Level: LevelVisible},
````
````new server/internal/modules/access/domain/rules.go
	"project.join": {Level: LevelVisible},
	// The project's admins, and its members who are the workspace's admins
	// (M3 design 3.5, 9.2; Plane views/project/member.py:234-238); the
	// relative rules are the use case's (3.5).
	"project_member.update": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"project.join": {allowed, allowed, allowed, allowed, allowed, allowed, allowed, invisible, invisible, invisible, invisible, allowed,
		invisible, invisible, invisible, forbidden, forbidden},
````
````new server/internal/modules/access/domain/rules_test.go
	"project.join": {allowed, allowed, allowed, allowed, allowed, allowed, allowed, invisible, invisible, invisible, invisible, allowed,
		invisible, invisible, invisible, forbidden, forbidden},
	// As project_member.add: the relative rules are the use case's (M3
	// design 3.5).
	"project_member.update": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible,
		invisible, forbidden, invisible, invisible, invisible, forbidden, forbidden},
````

`server/internal/modules/project/adapter/http/handler.go`（修改，3 处）：

````old server/internal/modules/project/adapter/http/handler.go
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
````
````new server/internal/modules/project/adapter/http/handler.go
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/shared"
````

````old server/internal/modules/project/adapter/http/handler.go
}

// UseCases are the use cases behind the module's operations.
````
````new server/internal/modules/project/adapter/http/handler.go
}

// UpdateMemberUseCase is app.UpdateProjectMember.
type UpdateMemberUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, role shared.Role) (domain.Member, error)
}

// UseCases are the use cases behind the module's operations.
````

````old server/internal/modules/project/adapter/http/handler.go
	JoinProject       JoinProjectUseCase
````
````new server/internal/modules/project/adapter/http/handler.go
	JoinProject       JoinProjectUseCase
	UpdateMember      UpdateMemberUseCase
````

`server/internal/modules/project/adapter/http/members.go`（修改，3 处）：

````old server/internal/modules/project/adapter/http/members.go
}

// members is list as the API shows it: data an array, never null.
````
````new server/internal/modules/project/adapter/http/members.go
}

// UpdateProjectMember serves PATCH
// /api/v0/project-members/{project_member_id}: the role goes to the use
// case as given, one outside the three too, which the domain refuses (422).
func (h handler) UpdateProjectMember(ctx context.Context, req gen.UpdateProjectMemberRequestObject) (gen.UpdateProjectMemberResponseObject, error) {
	m, err := h.uc.UpdateMember.Execute(ctx, req.ProjectMemberID, shared.Role(req.Body.Role))
	if err != nil {
		return nil, err
	}
	return gen.UpdateProjectMember200JSONResponse(member(m)), nil
}

// members is list as the API shows it: data an array, never null.
````

````old server/internal/modules/project/adapter/http/members.go
		out.Data[i] = gen.ProjectMember{ID: m.ID, ProjectID: m.ProjectID, MemberID: m.MemberID, Role: gen.ProjectRole(m.Role), CreatedAt: m.CreatedAt}
````
````new server/internal/modules/project/adapter/http/members.go
		out.Data[i] = member(m)
````

````old server/internal/modules/project/adapter/http/members.go
	return out
}

````
````new server/internal/modules/project/adapter/http/members.go
	return out
}

// member is m as the API shows it.
func member(m domain.Member) gen.ProjectMember {
	return gen.ProjectMember{ID: m.ID, ProjectID: m.ProjectID, MemberID: m.MemberID, Role: gen.ProjectRole(m.Role), CreatedAt: m.CreatedAt}
}

````

`server/internal/modules/project/adapter/http/handler_test.go`（修改，3 处）：

````old server/internal/modules/project/adapter/http/handler_test.go
	list      *fakeList
	create    *fakeCreate
	get       *fakeGet
	check     *fakeCheck
	update    *fakeUpdate
	archive   *fakeOnProject
	unarchive *fakeOnProject
	delete    *fakeDelete
	prefs     *fakePreferences
	members   *fakeMembers
	add       *fakeAdd
	join      *fakeOnProject
````
````new server/internal/modules/project/adapter/http/handler_test.go
	list         *fakeList
	create       *fakeCreate
	get          *fakeGet
	check        *fakeCheck
	update       *fakeUpdate
	archive      *fakeOnProject
	unarchive    *fakeOnProject
	delete       *fakeDelete
	prefs        *fakePreferences
	members      *fakeMembers
	add          *fakeAdd
	join         *fakeOnProject
	updateMember *fakeUpdateMember
````

````old server/internal/modules/project/adapter/http/handler_test.go
		f.join = &fakeOnProject{}
	}
````
````new server/internal/modules/project/adapter/http/handler_test.go
		f.join = &fakeOnProject{}
	}
	if f.updateMember == nil {
		f.updateMember = &fakeUpdateMember{}
	}
````

````old server/internal/modules/project/adapter/http/handler_test.go
		JoinProject: f.join})
````
````new server/internal/modules/project/adapter/http/handler_test.go
		JoinProject: f.join, UpdateMember: f.updateMember})
````

`server/internal/modules/project/adapter/http/member_writes_test.go`（新文件，112 行）：

````file server/internal/modules/project/adapter/http/member_writes_test.go
package httpadapter_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeUpdateMember is updateProjectMember: each call is recorded as
// "caller id role"; it answers answer, or err.
type fakeUpdateMember struct {
	calls  []string
	answer domain.Member
	err    error
}

func (f *fakeUpdateMember) Execute(ctx context.Context, id uuid.UUID, role shared.Role) (domain.Member, error) {
	f.calls = append(f.calls, fmt.Sprintf("%s %s %d", caller(ctx), id, role))
	return f.answer, f.err
}

// The answers of the writes on a project membership, as the contract
// declares them: each refusal, and a failure's 500, never a 200, a 204 or
// another problem.
const (
	memberNotFoundJSON = `{"status":404,"code":"project.member_not_found","title":"Not Found",` +
		`"detail":"The project member does not exist, or you cannot see the project."}`
	forbiddenJSON     = `{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`
	ownMembershipJSON = `{"status":409,"code":"project.own_membership","title":"Conflict","detail":"You cannot remove your own membership ` +
		`of the project, nor change your own role in it unless you are a workspace admin."}`
	roleTooHighJSON = `{"status":403,"code":"project.role_too_high","title":"Forbidden","detail":"The role is too high for you: unless you ` +
		`are a workspace admin, you change only a member whose project role is below yours, to a role below yours; and you remove only a ` +
		`member whose project role is not above yours."}`
	internalErrorJSON = `{"status":500,"code":"internal_error","title":"Internal Server Error"}`
)

// errGone is the error of a use case that failed.
var errGone = errors.New("the database is gone")

// PATCH goes to the use case for the caller, the path's membership and the
// role given, one outside the three too; the answer is 200 with the
// membership the use case answers.
func TestUpdateProjectMember(t *testing.T) {
	for _, role := range []int{5, 10} {
		update := &fakeUpdateMember{answer: bobInWeb}
		h := newServer(t, fakes{updateMember: update})
		path := "/api/v0/project-members/" + bobInWeb.ID.String()
		want := `{"created_at":"2026-10-01T10:00:00.123456Z","id":"0199a2b4-0000-7000-8000-0000000000b2",` +
			`"member_id":"0199a2b4-0000-7000-8000-000000000002","project_id":"0199a2b4-0000-7000-8000-0000000000a1","role":5}`
		if res, body := do(t, h, request(http.MethodPatch, path, "alice", fmt.Sprintf(`{"role":%d}`, role))); res.StatusCode != http.StatusOK ||
			body != want+"\n" {
			t.Errorf("PATCH role %d = %d %s, want 200 %s", role, res.StatusCode, body, want)
		}
		if want := []string{fmt.Sprintf("alice %s %d", bobInWeb.ID, role)}; !slices.Equal(update.calls, want) {
			t.Errorf("calls = %q, want %q", update.calls, want)
		}
	}
}

// A body without its role, with a field it may not have, or a role of
// another type: refused as bad_request before the use case.
func TestUpdateProjectMemberHoldsTheBodyToItsStructure(t *testing.T) {
	update := &fakeUpdateMember{}
	h := newServer(t, fakes{updateMember: update})
	for _, body := range []string{`{}`, `{"role":null}`, `{"role":"5"}`, `{"role":5,"member_id":"0199a2b4-0000-7000-8000-000000000002"}`} {
		if res, got := do(t, h, request(http.MethodPatch, "/api/v0/project-members/"+bobInWeb.ID.String(), "alice", body)); res.StatusCode !=
			http.StatusBadRequest {
			t.Errorf("PATCH %s = %d %s, want 400", body, res.StatusCode, got)
		}
	}
	if len(update.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", update.calls)
	}
}

// The use case's refusals, as the contract declares them, and its failure.
func TestUpdateProjectMemberRefusals(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"a role outside the three", shared.Invalid(shared.FieldError{Field: "role", Code: shared.FieldInvalidFormat,
			Message: "is not 5, 15 or 20"}), http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"role","code":"invalid_format","message":"is not 5, 15 or 20"}]}`},
		{"no membership", domain.ErrMemberNotFound, http.StatusNotFound, memberNotFoundJSON},
		{"a project member", shared.Forbidden(), http.StatusForbidden, forbiddenJSON},
		{"his own", domain.ErrOwnMembership, http.StatusConflict, ownMembershipJSON},
		{"an admin's role", domain.ErrRoleTooHigh, http.StatusForbidden, roleTooHighJSON},
		{"a workspace guest made a member", shared.Invalid(shared.FieldError{Field: "role", Code: shared.FieldNotAllowed,
			Message: "must be 5: the member is a guest of the workspace"}), http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"role","code":"not_allowed","message":"must be 5: the member is a guest of the workspace"}]}`},
		{"a failure", errGone, http.StatusInternalServerError, internalErrorJSON},
	}
	for _, tt := range tests {
		h := newServer(t, fakes{updateMember: &fakeUpdateMember{err: tt.err}})
		res, body := do(t, h, request(http.MethodPatch, "/api/v0/project-members/"+bobInWeb.ID.String(), "alice", `{"role":5}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: PATCH = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
````

`server/internal/modules/project/module.go`（修改，2 处）：

````old server/internal/modules/project/module.go
// members, each member's display settings, carries out the workspace
// module's cascades on the projects (ProjectCascade), and offers the
// access module its reads of a project (ProjectAccess) and the workspace
// module its count of an account's ended project memberships
// (ProjectMembershipCounts).
````
````new server/internal/modules/project/module.go
// members, changing a member's role, each member's display settings,
// carries out the workspace module's cascades on the projects
// (ProjectCascade), and offers the access module its reads of a project
// (ProjectAccess) and the workspace module its count of an account's ended
// project memberships (ProjectMembershipCounts).
````

````old server/internal/modules/project/module.go
		JoinProject:       app.NewJoinProject(locks, store, d.Tx, d.Clock),
````
````new server/internal/modules/project/module.go
		JoinProject:       app.NewJoinProject(locks, store, d.Tx, d.Clock),
		UpdateMember:      app.NewUpdateProjectMember(locks, store, d.Tx, d.Clock),
````

`web/apps/web/helpers/authentication.helper.ts`（修改，1 处）：

````old web/apps/web/helpers/authentication.helper.ts
  "project.archived": "auth.errors.project_archived",
````
````new web/apps/web/helpers/authentication.helper.ts
  "project.archived": "auth.errors.project_archived",
  "project.member_not_found": "auth.errors.project_member_not_found",
  "project.own_membership": "auth.errors.project_own_membership",
  "project.role_too_high": "auth.errors.project_role_too_high",
````

`web/packages/i18n/src/locales/en/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/auth.json
      "project_archived": "The project is archived. Restore it to change it.",
````
````new web/packages/i18n/src/locales/en/auth.json
      "project_archived": "The project is archived. Restore it to change it.",
      "project_member_not_found": "The project member does not exist, or you cannot see the project.",
      "project_own_membership": "You cannot remove yourself from the project, nor change your own role in it unless you are a workspace admin.",
      "project_role_too_high": "The role is too high for you. Unless you are a workspace admin, you change only members whose project role is below yours, to roles below yours; and you remove only members whose project role is not above yours.",
````

`web/packages/i18n/src/locales/zh-CN/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/auth.json
      "project_archived": "项目已归档，恢复之后才能修改。",
````
````new web/packages/i18n/src/locales/zh-CN/auth.json
      "project_archived": "项目已归档，恢复之后才能修改。",
      "project_member_not_found": "项目成员不存在，或你看不到这个项目。",
      "project_own_membership": "不能把自己移出项目；不是工作区管理员时，也不能修改自己在项目里的角色。",
      "project_role_too_high": "角色太高：除非你是工作区管理员，你只能修改项目角色比你低的成员，并且只能改为比你低的角色；你只能移出项目角色不比你高的成员。",
````

- [ ] **Step 3: 矩阵**

`server/internal/bootstrap/permission_matrix_seeded_test.go`（修改，8 处）：

````old server/internal/bootstrap/permission_matrix_seeded_test.go
// each other workspace's admin in its project.
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
// each other workspace's admin in its project, and gone's member in gone's,
// a member of the project the X columns' writes on a membership name there.
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
	{"gone/project", callerDeleted, shared.RoleAdmin}, {"other/project", callerNever, shared.RoleAdmin},
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
	{"gone/project", callerDeleted, shared.RoleAdmin}, {"gone/project", callerMember, shared.RoleMember},
	{"other/project", callerNever, shared.RoleAdmin},
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
// slug and the address; each project, by its key; and each account, by
// its name in matrixAccounts, which prepareMatrix registers. t is the test
// that asks for them (in).
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
// slug and the address; each project, by its key; each project membership,
// by the project's key and the column; and each account, by its name in
// matrixAccounts, which prepareMatrix registers. t is the test that asks
// for them (in).
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
	t           testing.TB
	workspaces  map[string]uuid.UUID
	memberships map[string]uuid.UUID
	invitations map[string]uuid.UUID
	projects    map[string]uuid.UUID
	accounts    map[caller]uuid.UUID
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
	t              testing.TB
	workspaces     map[string]uuid.UUID
	memberships    map[string]uuid.UUID
	invitations    map[string]uuid.UUID
	projects       map[string]uuid.UUID
	projectMembers map[string]uuid.UUID
	accounts       map[caller]uuid.UUID
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
// matrixMemberships, each of matrixInvitations and each of matrixProjects
// before prepareMatrix writes them, so that matrixViolations, without a
// database, sees the keys and the targets the cells will.
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
// matrixMemberships, each of matrixInvitations, each of matrixProjects and
// each of matrixProjectMembers before prepareMatrix writes them, so that
// matrixViolations, without a database, sees the keys and the targets the
// cells will.
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
		projects: map[string]uuid.UUID{}, accounts: map[caller]uuid.UUID{}}
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
		projects: map[string]uuid.UUID{}, projectMembers: map[string]uuid.UUID{}, accounts: map[caller]uuid.UUID{}}
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
		s.projects[p.key] = uuid.NewV7()
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
		s.projects[p.key] = uuid.NewV7()
	}
	for _, pm := range matrixProjectMembers {
		s.projectMembers[pm.key+"|"+string(pm.c)] = uuid.NewV7()
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
		s.t.Fatalf("no project %s is seeded", key)
	}
	return id
}
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
		s.t.Fatalf("no project %s is seeded", key)
	}
	return id
}

// projectMember is the id of c's membership of the project key; one never
// seeded fails the test at once, as membership's does.
func (s seeded) projectMember(key string, c caller) uuid.UUID {
	id, ok := s.projectMembers[key+"|"+string(c)]
	if !ok {
		s.t.Helper()
		s.t.Fatalf("no membership of %s by %s is seeded", key, c)
	}
	return id
}

// projectOfMember is the key of the project of the seeded project
// membership id, false for an id no seeded project membership has.
func (s seeded) projectOfMember(id uuid.UUID) (string, bool) {
	for key, seededID := range s.projectMembers {
		if seededID == id {
			project, _, _ := strings.Cut(key, "|")
			return project, true
		}
	}
	return "", false
}
````

`server/internal/bootstrap/permission_matrix_seed_test.go`（修改，3 处）：

````old server/internal/bootstrap/permission_matrix_seed_test.go
// join makes c a member of the project key with role, and stores his
// display settings in it.
func (s projectSeed) join(key string, c caller, role shared.Role) {
````
````new server/internal/bootstrap/permission_matrix_seed_test.go
// join makes c a member of the project key with role, the membership id,
// and stores his display settings in it.
func (s projectSeed) join(id uuid.UUID, key string, c caller, role shared.Role) {
````

````old server/internal/bootstrap/permission_matrix_seed_test.go
		ID: uuid.NewV7(), WorkspaceID: s.workspaces[slug], ProjectID: s.projects[key], MemberID: s.ids[c], Role: role, CreatedBy: by, Now: s.now,
````
````new server/internal/bootstrap/permission_matrix_seed_test.go
		ID: id, WorkspaceID: s.workspaces[slug], ProjectID: s.projects[key], MemberID: s.ids[c], Role: role, CreatedBy: by, Now: s.now,
````

````old server/internal/bootstrap/permission_matrix_seed_test.go
	s.targets(sd)
````
````new server/internal/bootstrap/permission_matrix_seed_test.go
	s.targets(sd)
	s.memberships(sd)
````

`server/internal/bootstrap/permission_matrix_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_test.go
	return slices.Concat(workspaceMatrixRows(), projectMatrixRows(), memberMatrixRows())
````
````new server/internal/bootstrap/permission_matrix_test.go
	return slices.Concat(workspaceMatrixRows(), projectMatrixRows(), memberMatrixRows(), membershipMatrixRows())
````

````old server/internal/bootstrap/permission_matrix_test.go
			projects.join(pm.key, pm.c, pm.role)
````
````new server/internal/bootstrap/permission_matrix_test.go
			projects.join(s.projectMember(pm.key, pm.c), pm.key, pm.c, pm.role)
````

`server/internal/bootstrap/permission_matrix_targets_test.go`（修改，4 处）：

````old server/internal/bootstrap/permission_matrix_targets_test.go
}

// targetViolation is what is wrong with where path, a path of pattern that
````
````new server/internal/bootstrap/permission_matrix_targets_test.go
}

// ofAProjectTable reports whether c is a column of a project table
// (projectTables).
func ofAProjectTable(c caller) bool {
	return slices.ContainsFunc(projectTables, func(table []caller) bool { return slices.Contains(table, c) })
}

// targetViolation is what is wrong with where path, a path of pattern that
````

````old server/internal/bootstrap/permission_matrix_targets_test.go
// project named by its id ({project_id}) must be projectOf(c)'s, and c a
// column of a project table (projectTables): a project's operation in a
// workspace-level row, or the only admin's, would leave the project level's
// own columns unasked.
````
````new server/internal/bootstrap/permission_matrix_targets_test.go
// project named by its id ({project_id}) must be projectOf(c)'s, and a
// project membership ({project_member_id}) one seeded in projectOf(c),
// each from a column of a project table (projectTables): a project's
// operation in a workspace-level row, or the only admin's, would leave the
// project level's own columns unasked.
````

````old server/internal/bootstrap/permission_matrix_targets_test.go
			if !slices.ContainsFunc(projectTables, func(table []caller) bool { return slices.Contains(table, c) }) {
````
````new server/internal/bootstrap/permission_matrix_targets_test.go
			if !ofAProjectTable(c) {
````

````old server/internal/bootstrap/permission_matrix_targets_test.go
				return fmt.Sprintf("{project_id} %s is not its column's project %s, %s", got[i], projectOf(c), id)
````
````new server/internal/bootstrap/permission_matrix_targets_test.go
				return fmt.Sprintf("{project_id} %s is not its column's project %s, %s", got[i], projectOf(c), id)
			}
		case segment == "{project_member_id}":
			if !ofAProjectTable(c) {
				return "{project_member_id} from a column of no project table (projectTables): a project's row names its columns"
			}
			id, err := uuid.Parse(got[i])
			if key, isMember := s.projectOfMember(id); err != nil || !isMember || key != projectOf(c) {
				return fmt.Sprintf("{project_member_id} %s is no membership seeded in its column's project %s", got[i], projectOf(c))
````

`server/internal/bootstrap/permission_matrix_membership_test.go`（新文件，172 行）：

````file server/internal/bootstrap/permission_matrix_membership_test.go
package bootstrap

import (
	"context"
	"net/http"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The rows of the permission matrix of the writes on one project
// membership (M3 design 9.2): changing a member's role. Each names the
// membership by its id (/project-members/{project_member_id}), in its
// column's project; prepareMatrix's preconditions hold each membership a
// row names to the state the row says.

var (
	cellProjectMemberNotFound = cell{http.StatusNotFound, "project.member_not_found"}
	cellProjectOwnMembership  = cell{http.StatusConflict, "project.own_membership"}
)

// ofMembership are the cells of a row of a write on a project membership:
// the answers of PA, PM, PG, PM+WA, WA- and WM-公, and
// project.member_not_found for the columns that do not see their project:
// WM-私, WG-, P-前 and X.
func ofMembership(pa, pm, pg, pmwa, wa, wm cell) map[caller]cell {
	return map[caller]cell{callerProjectAdmin: pa, callerProjectMember: pm, callerProjectGuest: pg, callerMemberAndAdmin: pmwa,
		callerAdminOnly: wa, callerMemberPublic: wm, callerMemberPrivate: cellProjectMemberNotFound, callerGuestOnly: cellProjectMemberNotFound,
		callerBefore: cellProjectMemberNotFound, callerNever: cellProjectMemberNotFound, callerRemoved: cellProjectMemberNotFound,
		callerDeleted: cellProjectMemberNotFound}
}

// aMembership names, for a column, the membership of its project a row's
// cell names: the project's key, and the column whose membership it is.
type aMembership func(c caller) (key string, member caller)

// projectMemberOf: PM's membership of the column's project, active, 15;
// in gone's project, where PM has none, gone's member's, deleted with it.
func projectMemberOf(c caller) (string, caller) {
	if key := projectOf(c); key == "gone/project" {
		return key, callerMember
	}
	return projectOf(c), callerProjectMember
}

// ownMembershipOf: the column's own membership of its project, for the
// columns that have one (PA, PM, PG, the workspace's guest's account, and
// PM+WA); PM's for the others.
func ownMembershipOf(c caller) (string, caller) {
	switch c {
	case callerProjectAdmin, callerProjectMember, callerMemberAndAdmin:
		return projectOf(c), c
	case callerProjectGuest:
		return projectOf(c), callerGuest
	}
	return projectMemberOf(c)
}

// endedMembershipOf: an ended membership of the column's project: the
// removed member's of the public one, which his removal ended, the member
// before's of the private one; in gone's project, gone's member's.
func endedMembershipOf(c caller) (string, caller) {
	switch key := projectOf(c); key {
	case "acme/public":
		return key, callerRemoved
	case "acme/private":
		return key, callerBefore
	}
	return projectMemberOf(c)
}

// workspaceGuestOf: the membership of the column's project of the
// workspace's guest (PG's account), a guest's; in gone's project, gone's
// member's.
func workspaceGuestOf(c caller) (string, caller) {
	if key := projectOf(c); key != "gone/project" {
		return key, callerGuest
	}
	return projectMemberOf(c)
}

// toProjectMembership is the request of a row whose callers each send
// method, with body, to the membership which names for their column.
func toProjectMembership(method, body string, which aMembership) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		key, member := which(c)
		return method, "/api/v0/project-members/" + s.projectMember(key, member).String(), body
	}
}

func membershipMatrixRows() []matrixRow {
	return []matrixRow{
		// The project's admins, and its members who are the workspace's
		// admins (M3 design 3.5, 9.2): PM's membership, his own in his
		// column, made a guest's. PM gets the rule's 403, not the 409 of his
		// own: the decision comes first.
		{op: "updateProjectMember", write: true, columns: projectColumns, request: toProjectMembership(http.MethodPatch, `{"role":5}`, projectMemberOf),
			cells: ofMembership(cellOK, cellForbidden, cellForbidden, cellOK, cellForbidden, cellForbidden), check: changesTheRole(projectMemberOf, 5)},
		// One's own role, to an admin's: the project admin who is no
		// workspace admin may not (409); the workspace's admin who is a
		// project member may (M3 design 3.5's exception).
		{op: "updateProjectMember", variant: "one's own membership", write: true, columns: projectColumns,
			request: toProjectMembership(http.MethodPatch, `{"role":20}`, ownMembershipOf),
			cells:   ofMembership(cellProjectOwnMembership, cellForbidden, cellForbidden, cellOK, cellForbidden, cellForbidden),
			check:   changesTheRole(ownMembershipOf, 20)},
		// An ended membership is 404 to who may change roles, after the
		// decision: who may not gets his 403, and learns nothing of it.
		{op: "updateProjectMember", variant: "an ended membership", write: true, columns: projectColumns,
			request: toProjectMembership(http.MethodPatch, `{"role":5}`, endedMembershipOf),
			cells:   ofMembership(cellProjectMemberNotFound, cellForbidden, cellForbidden, cellProjectMemberNotFound, cellForbidden, cellForbidden)},
		// A workspace guest stays a guest, whoever changes his role (M3
		// design 3.5), after the decision.
		{op: "updateProjectMember", variant: "a workspace guest made a member", write: true, columns: projectColumns,
			request: toProjectMembership(http.MethodPatch, `{"role":15}`, workspaceGuestOf),
			cells:   ofMembership(cellValidationFailed, cellForbidden, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden),
			refusal: "role not_allowed"},
	}
}

// memberships checks the project memberships the rows of the writes on one
// name (membershipMatrixRows), each in the state its row says: PM's, PA's,
// PM+WA's and the workspace guest's (PG's account) active, of their roles;
// the removed member's of the public project and the member before's of
// the private one ended, not deleted; gone's member's of gone's project
// deleted with gone. The workspace guest is acme's active guest, so that a
// role's 422 is his workspace role's.
func (s projectSeed) memberships(sd seeded) {
	s.t.Helper()
	ctx := context.Background()
	for _, tt := range []struct {
		key           string
		c             caller
		role          shared.Role
		found, active bool
	}{
		{"acme/public", callerProjectMember, shared.RoleMember, true, true}, {"acme/private", callerProjectMember, shared.RoleMember, true, true},
		{"acme/public", callerProjectAdmin, shared.RoleAdmin, true, true}, {"acme/public", callerMemberAndAdmin, shared.RoleMember, true, true},
		{"acme/public", callerGuest, shared.RoleGuest, true, true}, {"acme/private", callerGuest, shared.RoleGuest, true, true},
		{"acme/public", callerRemoved, shared.RoleMember, true, false}, {"acme/private", callerBefore, shared.RoleMember, true, false},
		{"gone/project", callerMember, 0, false, false},
	} {
		m, found, err := s.store.MemberByID(ctx, sd.projectMember(tt.key, tt.c))
		if err != nil || found != tt.found || found && (m.ProjectID != sd.project(tt.key) || m.MemberID != s.ids[tt.c] || m.Role != tt.role ||
			m.Active != tt.active) {
			s.t.Fatalf("%s's membership of %s = %+v, %v, %v; want found %v, active %v, of role %d", tt.c, tt.key, m, found, err, tt.found,
				tt.active, tt.role)
		}
	}
	if role, active, err := s.matrixSeed.store.ActiveRole(ctx, sd.workspace("acme"), s.ids[callerGuest]); err != nil || !active ||
		role != shared.RoleGuest {
		s.t.Fatalf("%s's role in acme = %d, %v, %v; want its active guest", callerGuest, role, active, err)
	}
}

// changesTheRole: the membership which names for the column, of its
// project and member, with role.
func changesTheRole(which aMembership, role int) func(t *testing.T, c caller, s seeded, answer string) {
	return func(t *testing.T, c caller, s seeded, answer string) {
		var m struct {
			ID        uuid.UUID `json:"id"`
			ProjectID uuid.UUID `json:"project_id"`
			MemberID  uuid.UUID `json:"member_id"`
			Role      int       `json:"role"`
		}
		decodeAnswer(t, answer, &m)
		key, member := which(c)
		if m.ID != s.projectMember(key, member) || m.ProjectID != s.project(key) || m.MemberID != s.account(accountOf(member)) || m.Role != role {
			t.Errorf("%s changes %s; want %s's membership of %s as %d", c, answer, member, key, role)
		}
	}
}
````

`server/internal/bootstrap/permission_matrix_columns_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_columns_test.go
		}
	}
	// A parameter passed over before another leaves that one checked too.
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
		}
	}
	// A membership a path names by its id ({project_member_id}) is one
	// seeded in the column's project, from a column of a project table.
	membership := apitest.Operation{ID: "updateProjectMember", Tags: []string{"project"}, Method: http.MethodPatch,
		Path: "/api/v0/project-members/{project_member_id}"}
	changes := matrixRow{op: membership.ID, write: true, columns: projectColumns, cells: cells,
		request: toProjectMembership(http.MethodPatch, `{"role":5}`, projectMemberOf)}
	// privateNames is changes, but WM-私's cell names id.
	privateNames := func(id uuid.UUID) matrixRow {
		r := changes
		r.request = func(c caller, s seeded) (string, string, string) {
			if c == callerMemberPrivate {
				return http.MethodPatch, "/api/v0/project-members/" + id.String(), `{"role":5}`
			}
			return changes.request(c, s)
		}
		return r
	}
	inWorkspaceRow := changes
	inWorkspaceRow.columns, inWorkspaceRow.cells = nil, every(cellOK)
	var noTable []string
	for _, c := range []caller{callerAdmin, callerMember, callerGuest} {
		noTable = append(noTable, fmt.Sprintf("row updateProjectMember, %s: {project_member_id} from a column of no project table (projectTables): "+
			"a project's row names its columns", c))
	}
	publicPM := s.projectMember("acme/public", callerProjectMember)
	for _, tt := range []struct {
		name string
		row  matrixRow
		want []string
	}{
		{"memberships of their columns' projects", changes, nil},
		{"a membership of another column's project", privateNames(publicPM), []string{fmt.Sprintf(
			"row updateProjectMember, %s: {project_member_id} %s is no membership seeded in its column's project acme/private", callerMemberPrivate,
			publicPM)}},
		{"an id no seeded membership has", privateNames(uuid.Nil()), []string{fmt.Sprintf(
			"row updateProjectMember, %s: {project_member_id} %s is no membership seeded in its column's project acme/private", callerMemberPrivate,
			uuid.Nil())}},
		{"a membership in a workspace-level row", inWorkspaceRow, noTable},
	} {
		if got := matrixViolations(append(ops, membership), listed, []matrixRow{row, checks, tt.row}, s, nil); !slices.Equal(got, tt.want) {
			t.Errorf("%s: %q, want %q", tt.name, got, tt.want)
		}
	}
	// A parameter passed over before another leaves that one checked too.
````

````old server/internal/bootstrap/permission_matrix_columns_test.go
		t.Errorf("a project never seeded: failed with %q, want %q", failed, want)
	}
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
		t.Errorf("a project never seeded: failed with %q, want %q", failed, want)
	}
	// So does a project membership never seeded.
	failed = fatalOf(func(tb testing.TB) { newSeeded().in(tb).projectMember("acme/public", callerNever) })
	if want := "no membership of acme/public by never a member is seeded"; failed != want {
		t.Errorf("a project membership never seeded: failed with %q, want %q", failed, want)
	}
````

- [ ] **Step 4: 每个项目级的写最先锁工作区**

`server/internal/bootstrap/project_write_locks_test.go`（修改，16 处）：

````old server/internal/bootstrap/project_write_locks_test.go
// operationId: the request on the project, by alice unless byTarget.
````
````new server/internal/bootstrap/project_write_locks_test.go
// operationId: the request on the project, or on a membership of it, by
// alice unless by names another sender.
````

````old server/internal/bootstrap/project_write_locks_test.go
	op, method, path, body string // path: %s the project's id; body: %s the target's id
````
````new server/internal/bootstrap/project_write_locks_test.go
	// path: %s the project's id, or the membership's for a path of one
	// (/api/v0/project-members/); body: %s the target's id.
	op, method, path, body string
````

````old server/internal/bootstrap/project_write_locks_test.go
	// targets are the accounts the write makes members of the project, one
	// a phase: acme's members, none of the project's.
````
````new server/internal/bootstrap/project_write_locks_test.go
	// targets are the accounts, one a phase, whose membership of the
	// workspace the write locks: acme's members, whom it makes members of
	// the project, or whose role in it it changes.
````

````old server/internal/bootstrap/project_write_locks_test.go
	// byTarget is set when the target sends the write: a joining.
	byTarget bool
````
````new server/internal/bootstrap/project_write_locks_test.go
	// by are the accounts, one a phase, that send the write: alice when
	// empty.
	by [2]string
	// member are the accounts, one a phase, whose membership of the project
	// the write changes: a path of a membership names it, and its row is
	// probed as the project's is.
	member [2]string
}

// param is the parameter w's path names: a membership's id for a path of
// one, else the project's.
func (w projectWrite) param() string {
	if strings.HasPrefix(w.path, "/api/v0/project-members/") {
		return "{project_member_id}"
	}
	return "{project_id}"
````

````old server/internal/bootstrap/project_write_locks_test.go
		byTarget: true},
````
````new server/internal/bootstrap/project_write_locks_test.go
		by: [2]string{"dave", "erin"}},
	// The members added before, bob in Web and carol in Ops, made guests.
	{op: "updateProjectMember", method: http.MethodPatch, path: "/api/v0/project-members/%s", body: `{"role":5}`, want: http.StatusOK,
		targets: [2]string{"bob", "carol"}, member: [2]string{"bob", "carol"}},
````

````old server/internal/bootstrap/project_write_locks_test.go
// projects Web and Ops, each write once on each, and a write's targets are
// made members of them. Every write on a project of the contract has its
// row here: every operation but GET whose path names a project, or that a
// matrix row asks of a project-level column (writesOnAProject), is the
// list, which a write without a row here fails before any database.
````
````new server/internal/bootstrap/project_write_locks_test.go
// projects Web and Ops, each write once on each; a write's targets are made
// members of them or their roles changed, and a write on a membership names
// it by its id, or by its project and its sender. Every write on a project
// of the contract has its row here: every operation but GET whose path
// names a project, or that a matrix row asks of a project-level column
// (writesOnAProject), is the list, which a write without a row here fails
// before any database.
````

````old server/internal/bootstrap/project_write_locks_test.go
//     Web waits for that row, and meanwhile holds neither Web's row nor its
//     target's membership of acme: a FOR UPDATE NOWAIT of each succeeds.
````
````new server/internal/bootstrap/project_write_locks_test.go
//     Web waits for that row, and meanwhile holds neither Web's row, nor
//     its target's membership of acme, nor the membership of Web it
//     changes: a FOR UPDATE NOWAIT of each succeeds.
````

````old server/internal/bootstrap/project_write_locks_test.go
//     55P03).
````
````new server/internal/bootstrap/project_write_locks_test.go
//     55P03), but not the membership of Ops it changes, which comes after
//     the project.
````

````old server/internal/bootstrap/project_write_locks_test.go
			return o.ID == w.op && o.Method == w.method && o.Path == fmt.Sprintf(w.path, "{project_id}")
````
````new server/internal/bootstrap/project_write_locks_test.go
			return o.ID == w.op && o.Method == w.method && o.Path == fmt.Sprintf(w.path, w.param())
````

````old server/internal/bootstrap/project_write_locks_test.go
		for _, name := range w.targets {
			if name != "" {
````
````new server/internal/bootstrap/project_write_locks_test.go
		for _, name := range slices.Concat(w.targets[:], w.by[:], w.member[:]) {
			if _, registered := tokens[name]; name != "" && !registered {
````

````old server/internal/bootstrap/project_write_locks_test.go
	membership := "SELECT 1 FROM workspace_members WHERE workspace_id = $1 AND member_id = $2 AND deleted_at IS NULL FOR UPDATE"
````
````new server/internal/bootstrap/project_write_locks_test.go
	membership := "SELECT 1 FROM workspace_members WHERE workspace_id = $1 AND member_id = $2 AND deleted_at IS NULL FOR UPDATE"
	memberRow := "SELECT 1 FROM project_members WHERE id = $1 FOR UPDATE"
````

````old server/internal/bootstrap/project_write_locks_test.go
				target, token, body := w.targets[phase], alice, w.body
````
````new server/internal/bootstrap/project_write_locks_test.go
				target, member, token, body, named := w.targets[phase], w.member[phase], alice, w.body, project
````

````old server/internal/bootstrap/project_write_locks_test.go
				if w.byTarget {
					token = tokens[target]
````
````new server/internal/bootstrap/project_write_locks_test.go
				if by := w.by[phase]; by != "" {
					token = tokens[by]
````

````old server/internal/bootstrap/project_write_locks_test.go
				req := newRequest(t, w.method, base+fmt.Sprintf(w.path, project), token, []byte(body))
````
````new server/internal/bootstrap/project_write_locks_test.go
				// The membership the write changes, its row probed, and named
				// by a path of one.
				var row uuid.UUID
				if member != "" {
					if err := pool.QueryRow(soon(t), "SELECT id FROM project_members WHERE project_id = $1 AND member_id = $2 AND deleted_at IS NULL",
						project, ids[member]).Scan(&row); err != nil {
						t.Fatalf("%s's membership of %s: %v", member, on, err)
					}
					if w.param() == "{project_member_id}" {
						named = row
					}
				}
				req := newRequest(t, w.method, base+fmt.Sprintf(w.path, named), token, []byte(body))
````

````old server/internal/bootstrap/project_write_locks_test.go
						t.Errorf("%s holds %s's membership of acme while it waits for its workspace", w.op, target)
					}
````
````new server/internal/bootstrap/project_write_locks_test.go
						t.Errorf("%s holds %s's membership of acme while it waits for its workspace", w.op, target)
					}
					if member != "" && heldBy(t, pool, memberRow, row) {
						t.Errorf("%s holds %s's membership of %s while it waits for its workspace", w.op, member, on)
					}
````

````old server/internal/bootstrap/project_write_locks_test.go
						t.Errorf("%s does not hold %s's membership of acme while it waits for its project", w.op, target)
````
````new server/internal/bootstrap/project_write_locks_test.go
						t.Errorf("%s does not hold %s's membership of acme while it waits for its project", w.op, target)
					}
					if member != "" && heldBy(t, pool, memberRow, row) {
						t.Errorf("%s holds %s's membership of %s before its project", w.op, member, on)
````

- [ ] **Step 5: 测试、lint、前端检查**

Run: `go -C server test -count=1 ./internal/modules/project/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix$|TestMatrixViolationsCatchesEach|TestEachMatrixTableIsOfOneLevel|TestEachWriteOnAProjectSharesItsWorkspaceFirst|TestWritesOnAProjectAreEachShape|TestAPIRoutesAreTheContractsOperations' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

- [ ] **Step 6: 提交**

```bash
git add api/modules/project.yaml api/openapi.yaml server/internal/bootstrap/permission_matrix_columns_test.go server/internal/bootstrap/permission_matrix_membership_test.go server/internal/bootstrap/permission_matrix_seed_test.go server/internal/bootstrap/permission_matrix_seeded_test.go server/internal/bootstrap/permission_matrix_targets_test.go server/internal/bootstrap/permission_matrix_test.go server/internal/bootstrap/project_write_locks_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/project/adapter/http/handler.go server/internal/modules/project/adapter/http/handler_test.go server/internal/modules/project/adapter/http/member_writes_test.go server/internal/modules/project/adapter/http/members.go server/internal/modules/project/domain/actions.go server/internal/modules/project/module.go web/apps/web/helpers/authentication.helper.ts web/packages/i18n/src/locales/en/auth.json web/packages/i18n/src/locales/zh-CN/auth.json api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/bodyshape.gen.go server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P5b): updateProjectMember

PATCH /api/v0/project-members/{project_member_id}, for a project's
admins and its members who are the workspace's admins. The matrix
gains its four rows, aimed at memberships seeded in each column's
project, and the first-lock test takes a write named by a membership,
probing that row too.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A，清扫 4、5、10、18，P5b 自己的）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| 规则给项目成员 | `TestEveryRuleDecidesItsCells`；`TestPermissionMatrix`；P5（Task 11 起） | 单元；组合；端到端 |
| `project.New` 的改角色不开事务 | `TestEachWriteOnAProjectSharesItsWorkspaceFirst`（没有事务就不持锁）；`TestEachLockOfAWriteOnAProjectMembershipIsItsStrength`（Task 8 起） | 组合 |
| `project.New` 的改角色的判定换成放行一切的 `Authorizer` | `TestPermissionMatrix`；`TestTheRelativeRuleOnTheComposedApp`、`TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile`（Task 5、8 起） | 组合 |
| 写在锁资源行之后才锁工作区（成员关系的第一次读带 `FOR UPDATE`） | `TestEachWriteOnAProjectSharesItsWorkspaceFirst`（第一步的资源行探测） | 组合 |
| 去掉 `projectWrites` 的改角色一行 | `TestEachWriteOnAProjectSharesItsWorkspaceFirst`（完整性核对的第二种形状） | 组合 |
| PM 种成公开项目的管理员（种子行） | `TestPermissionMatrix/prepare`（`memberships` 的前提） | 组合 |
| 契约少声明一个用例会答的码；多声明一个从不回答的码 | `project/adapter/http` 的测试：少声明的由 `CheckResponse`（`TestUpdateProjectMemberRefusals`）发现，另由组合的 `TestTheRelativeRuleOnTheComposedApp`（Task 5 起，`project.role_too_high`）；多声明的由 `api/modules/project.yaml` 上的 `apitest.Main`（包的测试结束时）发现；同样的变异在 Task 6、7 的表里 | 单元；组合 |
| 矩阵格子指向别的列的项目的成员关系、没有种下的 id；成员关系的行放进工作区一级的表 | `TestMatrixViolationsCatchesEachColumnGap` | 组合（没有数据库） |

**Done when:** 改角色的四行矩阵（48 格）通过；`project` 的 `apitest.Main` 两个方向核对通过；完整性核对少了改角色一行就失败。

---

### Task 5: 组合出的相对规则

**Files:**
- Create: `server/internal/bootstrap/membership_world_test.go`、`server/internal/bootstrap/project_roles_test.go`

**Interfaces:** 没有新的产品代码（spec 2.8，brief 清扫 5、23、24、26）。`memberWorld`：组合出的 app，自己的数据库；acme（alice 管理员；bob、carol、dave、gina 成员；erin 访客，各经接受 alice 的邀请），beta（bob 的；carol 经他的邀请是成员）；Web（alice 的：bob、dave 管理员，carol、gina 成员，erin 访客，由她添加；之后 alice 把 gina 改为 acme 的管理员，gina 是 PM+WA）；Ops（alice 的：carol 管理员，bob 成员）；Lab（beta 的，bob 的：carol 成员）。bob、carol 各是一处的管理员、另一处的成员，bob 只在 beta 是工作区管理员（清扫 23）。`standing`（每个有效成员关系，按名字）、`membership`、`added`、`change`。

**Tests:**
- `project_roles_test.go`：`TestTheRelativeRuleOnTheComposedApp`：一步接一步，每一步的拒绝一行不改，每一次改只改目标的角色、由调用者在请求的时刻写（之前由 alice 写，清扫 14）、其余的列不动：carol 是 Web 的成员，连把 erin 保持为访客也 403（相对规则会放行，规则的角色在它之前）；她是 Ops 的管理员，把 bob 改为访客；bob 是 Web 的管理员（beta 的工作区管理员，acme 的成员），改自己 409，改 dave（另一位管理员）、提拔 carol 403 `project.role_too_high`，把 erin（acme 的访客）改为成员 422 `role` `not_allowed`，把 carol 改为访客；gina（acme 的管理员、Web 的成员）把 erin 改为成员同样 422，降 dave 为成员、提拔 carol、提拔自己都通过（3.5 的例外）。最后的 `standing` 核对每个工作区和项目的每个有效成员关系。

- [ ] **Step 1: 组合出的世界和测试**

`server/internal/bootstrap/membership_world_test.go`（新文件，136 行）：

````file server/internal/bootstrap/membership_world_test.go
package bootstrap

import (
	"fmt"
	"net/http"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// memberWorld is the wired app on a database of its own, for the writes on
// one project membership: acme, alice its admin; bob, carol, dave and gina
// its members, erin its guest, each by accepting alice's invitation; beta,
// bob's, carol its member by accepting his. Web, alice's: bob and dave its
// admins, carol and gina its members, erin its guest, each by her adding;
// gina then made acme's admin by alice, so that she is a member of Web who
// is a workspace admin (PM+WA). Ops, alice's: carol its admin, bob its
// member, by her adding. Lab, beta's, bob's: carol its member by his
// adding. bob and carol are each an admin somewhere and a member elsewhere,
// and bob is a workspace admin in beta alone.
type memberWorld struct {
	contract      *apitest.Contract
	base          string
	pool          *pgxpool.Pool
	tokens        map[string]string    // access tokens, by name
	ids           map[string]uuid.UUID // accounts, by name
	web, ops, lab uuid.UUID
}

func newMemberWorld(t *testing.T) memberWorld {
	t.Helper()
	contract := apitest.Load(t)
	dbURL := pgtest.NewDatabase(t)
	w := memberWorld{contract: contract, base: startApp(t, testConfig(t, dbURL, false), migrations.FS()), pool: openPool(t, dbURL),
		tokens: map[string]string{}, ids: map[string]uuid.UUID{}}
	for _, name := range []string{"alice", "bob", "carol", "dave", "erin", "gina"} {
		w.tokens[name] = registerAccount(t, contract, w.base, name+"@example.com").AccessToken
		w.ids[name] = accountID(t, contract, w.base, w.tokens[name])
	}
	for _, ws := range []struct{ slug, admin string }{{"acme", "alice"}, {"beta", "bob"}} {
		if status, body := call(t, contract, http.MethodPost, w.base+"/api/v0/workspaces", w.tokens[ws.admin],
			`{"name":"`+ws.slug+`","slug":"`+ws.slug+`"}`); status != http.StatusCreated {
			t.Fatalf("creating %s = %d %s", ws.slug, status, body)
		}
	}
	for _, m := range []struct {
		slug, admin, name string
		role              shared.Role
	}{{"acme", "alice", "bob", shared.RoleMember}, {"acme", "alice", "carol", shared.RoleMember}, {"acme", "alice", "dave", shared.RoleMember},
		{"acme", "alice", "gina", shared.RoleMember}, {"acme", "alice", "erin", shared.RoleGuest}, {"beta", "bob", "carol", shared.RoleMember}} {
		answerInvitation(t, contract, w.base, w.tokens[m.name], "accept",
			inviteAs(t, contract, w.base, w.tokens[m.admin], m.slug, m.name+"@example.com", m.role), http.StatusOK)
	}
	w.web = createdProject(t, contract, w.base, w.tokens["alice"], "acme", "Web", "WEB")
	w.ops = createdProject(t, contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	w.lab = createdProject(t, contract, w.base, w.tokens["bob"], "beta", "Lab", "LAB")
	for _, add := range []struct {
		by      string
		project uuid.UUID
		members string
	}{
		{"alice", w.web, w.added("bob", 20, "dave", 20, "carol", 15, "gina", 15, "erin", 5)},
		{"alice", w.ops, w.added("carol", 20, "bob", 15)},
		{"bob", w.lab, w.added("carol", 15)},
	} {
		if status, body := call(t, contract, http.MethodPost, w.base+"/api/v0/projects/"+add.project.String()+"/members", w.tokens[add.by],
			add.members); status != http.StatusCreated {
			t.Fatalf("%s's adding %s = %d %s", add.by, add.members, status, body)
		}
	}
	var gina uuid.UUID
	if err := w.pool.QueryRow(soon(t), `SELECT m.id FROM workspace_members m JOIN workspaces s ON s.id = m.workspace_id
		WHERE s.slug = 'acme' AND m.member_id = $1`, w.ids["gina"]).Scan(&gina); err != nil {
		t.Fatal(err)
	}
	if status, body := call(t, contract, http.MethodPatch, w.base+"/api/v0/workspace-members/"+gina.String(), w.tokens["alice"],
		`{"role":20}`); status != http.StatusOK {
		t.Fatalf("alice's making gina acme's admin = %d %s", status, body)
	}
	if got, want := w.standing(t), "acme: alice 20, bob 15, carol 15, dave 15, erin 5, gina 20; beta: bob 20, carol 15; "+
		"Lab: bob 20, carol 15; Ops: alice 20, bob 15, carol 20; Web: alice 20, bob 20, carol 15, dave 20, erin 5, gina 15"; got != want {
		t.Fatalf("the world: %s; want %s", got, want)
	}
	return w
}

// added is the body of an addition of the named accounts, each with the
// role after its name.
func (w memberWorld) added(namesAndRoles ...any) string {
	body := `{"members":[`
	for i := 0; i < len(namesAndRoles); i += 2 {
		if i > 0 {
			body += ","
		}
		body += fmt.Sprintf(`{"member_id":"%s","role":%d}`, w.ids[namesAndRoles[i].(string)], namesAndRoles[i+1].(int))
	}
	return body + "]}"
}

// standing is every active membership of the world, the workspaces' and
// the projects', each workspace and project by name with its members' roles
// in name order: "acme: alice 20, bob 15; Web: …".
func (w memberWorld) standing(t *testing.T) string {
	t.Helper()
	var got string
	if err := w.pool.QueryRow(soon(t), `SELECT string_agg(place || ': ' || members, '; ' ORDER BY place) FROM (
		SELECT place, string_agg(name || ' ' || role, ', ' ORDER BY name) AS members FROM (
			SELECT s.slug AS place, split_part(u.email, '@', 1) AS name, m.role FROM workspace_members m
				JOIN workspaces s ON s.id = m.workspace_id JOIN users u ON u.id = m.member_id WHERE m.is_active AND m.deleted_at IS NULL
			UNION ALL SELECT p.name, split_part(u.email, '@', 1), m.role FROM project_members m
				JOIN projects p ON p.id = m.project_id JOIN users u ON u.id = m.member_id WHERE m.is_active AND m.deleted_at IS NULL) a
		GROUP BY place) b`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	return got
}

// membership is the id of name's membership of project.
func (w memberWorld) membership(t *testing.T, project uuid.UUID, name string) uuid.UUID {
	t.Helper()
	return projectMemberships(t, w.pool, w.ids[name], project)[0]
}

// change is by's change of name's role in project to role: its status and
// body.
func (w memberWorld) change(t *testing.T, by string, project uuid.UUID, name string, role int) (int, string) {
	t.Helper()
	return call(t, w.contract, http.MethodPatch, w.base+"/api/v0/project-members/"+w.membership(t, project, name).String(), w.tokens[by],
		fmt.Sprintf(`{"role":%d}`, role))
}
````

`server/internal/bootstrap/project_roles_test.go`（新文件，83 行）：

````file server/internal/bootstrap/project_roles_test.go
package bootstrap

import (
	"maps"
	"net/http"
	"strings"
	"testing"
	"uuid"
)

// M3 design 3.5's rule for a change of a project role on the wired app,
// each half with its counterexample, one change after another on
// memberWorld; each refusal changes no row, and each change the target's
// role alone, by its caller:
//   - the rule's roles: carol, Web's member, may change no role there,
//     not even a guest's to a guest's, which the relative rule would let
//     her; as Ops's admin, she makes bob, its member, a guest;
//   - one who is not a workspace admin: bob, Web's admin, cannot change his
//     own role, though he is beta's admin; nor dave's, an admin's, nor make
//     carol an admin (project.role_too_high); he makes carol a guest;
//   - the workspace guest's cap: bob cannot make erin, acme's guest, a
//     member, and nor can gina (422 role not_allowed);
//   - the workspace admin's exception: gina, acme's admin and Web's member,
//     makes dave, an admin, a member, carol an admin, and herself an admin.
func TestTheRelativeRuleOnTheComposedApp(t *testing.T) {
	w := newMemberWorld(t)
	for _, step := range []struct {
		name    string
		by      string
		project uuid.UUID
		member  string
		role    int
		status  int
		code    string // of a refusal
	}{
		{"carol, Web's member, keeps erin a guest", "carol", w.web, "erin", 5, http.StatusForbidden, "forbidden"},
		{"carol, Ops's admin, makes bob, its member, a guest", "carol", w.ops, "bob", 5, http.StatusOK, ""},
		{"bob, Web's admin and beta's, changes his own role", "bob", w.web, "bob", 15, http.StatusConflict, "project.own_membership"},
		{"bob changes dave, another admin", "bob", w.web, "dave", 15, http.StatusForbidden, "project.role_too_high"},
		{"bob makes carol an admin", "bob", w.web, "carol", 20, http.StatusForbidden, "project.role_too_high"},
		{"bob makes erin, acme's guest, a member", "bob", w.web, "erin", 15, http.StatusUnprocessableEntity, "validation_failed"},
		{"bob makes carol a guest", "bob", w.web, "carol", 5, http.StatusOK, ""},
		{"gina, Web's member and acme's admin, makes erin a member", "gina", w.web, "erin", 15, http.StatusUnprocessableEntity,
			"validation_failed"},
		{"gina makes dave, an admin, a member", "gina", w.web, "dave", 15, http.StatusOK, ""},
		{"gina makes carol an admin", "gina", w.web, "carol", 20, http.StatusOK, ""},
		{"gina makes herself an admin", "gina", w.web, "gina", 20, http.StatusOK, ""},
	} {
		id := w.membership(t, step.project, step.member)
		target, others := rowJSON(t, w.pool, "project_members", id), rowsBut(t, w.pool, []uuid.UUID{id})
		status, body := w.change(t, step.by, step.project, step.member, step.role)
		if status != step.status || step.code != "" && problemCode(t, []byte(body)) != step.code {
			t.Fatalf("%s = %d %s, want %d %s", step.name, status, body, step.status, step.code)
		}
		if step.code == "validation_failed" && !strings.Contains(body, `"errors":[{"field":"role","code":"not_allowed"`) {
			t.Errorf("%s = %s, want its one error role not_allowed", step.name, body)
		}
		if after := rowsBut(t, w.pool, []uuid.UUID{id}); !maps.Equal(after, others) {
			t.Errorf("%s changed rows besides the target's:\n%v\nwant them as they were:\n%v", step.name, after, others)
		}
		after := rowJSON(t, w.pool, "project_members", id)
		if step.code != "" {
			if !maps.Equal(after, target) {
				t.Errorf("%s changed the target's membership:\n%v\nwant it as it was:\n%v", step.name, after, target)
			}
			continue
		}
		if after["role"] != float64(step.role) || after["updated_by_id"] != w.ids[step.by].String() || after["updated_at"] == target["updated_at"] {
			t.Errorf("%s: the target's membership %v; want role %d, written now by %s", step.name, after, step.role, step.by)
		}
		for _, column := range []string{"role", "updated_by_id", "updated_at"} {
			delete(after, column)
			delete(target, column)
		}
		if !maps.Equal(after, target) {
			t.Errorf("%s changed the target's other columns:\n%v\nwant them as they were:\n%v", step.name, after, target)
		}
	}
	if got, want := w.standing(t), "acme: alice 20, bob 15, carol 15, dave 15, erin 5, gina 20; beta: bob 20, carol 15; "+
		"Lab: bob 20, carol 15; Ops: alice 20, bob 5, carol 20; Web: alice 20, bob 20, carol 20, dave 15, erin 5, gina 20"; got != want {
		t.Errorf("the world after the changes: %s; want %s", got, want)
	}
}
````

- [ ] **Step 2: 测试和 lint**

Run: `go -C server test -count=1 -run 'TestTheRelativeRuleOnTheComposedApp$' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 3: 提交**

```bash
git add server/internal/bootstrap/membership_world_test.go server/internal/bootstrap/project_roles_test.go
```
```bash
git commit -m "test(M3/P5b): the relative rule on the wired app

memberWorld wires the app over two workspaces and three projects where
the same accounts are admins in one place and members in another. On
it, each half of the rule of a project role change has its case and
its counterexample: the rule's roles, one's own role, another admin's,
a promotion, a workspace guest's cap, and the workspace admin's
exception. A refusal changes no row; a change writes the target's role
alone, by its caller.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A，清扫 5、23、24）：Task 2、3 的变异表里标"Task 5 起"的每一行在这里被发现（相对规则不约束、工作区管理员也受约束、不看 `From`、`To`、自己的、访客的上限、上限读调用者的角色）。另：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| `memberWorld` 少了 gina 成为 acme 的管理员、bob 在 beta 的管理员身份、erin 是 acme 的访客 | `TestTheRelativeRuleOnTheComposedApp`（`newMemberWorld` 的 `standing` 前提） | 组合 |

**Done when:** 相对规则的每一半在组合出的 app 上有成立的情形和反例。

---

### Task 6: `removeProjectMember`

**Files:**
- Create: `server/internal/bootstrap/project_removal_test.go`、`server/internal/modules/project/app/remove_member.go`、`server/internal/modules/project/app/remove_member_test.go`
- Modify: `api/modules/project.yaml`、`server/internal/bootstrap/permission_matrix_membership_test.go`、`server/internal/bootstrap/project_write_locks_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/project/adapter/http/handler.go`、`server/internal/modules/project/adapter/http/handler_test.go`、`server/internal/modules/project/adapter/http/member_writes_test.go`、`server/internal/modules/project/adapter/http/members.go`、`server/internal/modules/project/app/clock_test.go`、`server/internal/modules/project/app/fakes_member_test.go`、`server/internal/modules/project/domain/actions.go`、`server/internal/modules/project/module.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.9，M3 设计 3.3、3.5、3.6、5.1、9.2）：`DELETE /api/v0/project-members/{project_member_id}`，204 无正文；码 `[project.member_not_found, forbidden, project.own_membership, project.role_too_high]`（Task 4 已进 `PROBLEM_MESSAGES` 和两份 `auth.json`）。
- `domain.ActionMemberRemove`（`project_member.remove`），规则 `{Level: LevelProject, Roles: [admin]}`，进 `Actions()`。
- `RemoveProjectMember`（`NewRemoveProjectMember(locks, members MemberEnder, tx, clock)`），`Execute(ctx, id) error`：没有调用者 401 → 一个事务：`lockMemberAndDecide(…, project_member.remove, target: false)`（不锁成员在工作区的成员关系：移出不读他的工作区角色）→ 已结束 404 → `CheckRemoval(调用者的项目角色, 是自己的, 成员的角色)` → 读时钟 → `EndMember(项目, 成员, 调用者, 时刻)`。
- HTTP：`RemoveMemberUseCase`；`UseCases.RemoveMember`；`(handler).RemoveProjectMember`。`project.New` 的 `RemoveMember`。单元的假存储多 `EndMember`。
- 矩阵：四行移出，各 12 格：PM 的成员关系 `ofMembership(204, 403, 403, 204, 403, 403)`；自己的 `(409, 403, 403, 409, 403, 403)`（工作区管理员也是）；已结束的 `(404, 403, 403, 404, 403, 403)`；管理员（PA）的 `(409 自己的, 403, 403, 403 project.role_too_high, 403, 403)`（PM+WA 是成员，不能移出比他高的角色：没有例外）。`adminOf`；`memberships` 的前提加 PA 在私有项目的、gone 的管理员的。
- `bootstrap/project_removal_test.go`：`memberEnding{step, name, by, project, status, code, send}` 和 `check`：结束或拒绝的一步的共同核对（之前他在项目里有显示设置；目标之外的每一行不变；拒绝时目标不变；结束时目标结束、未删除，由 `by`、在请求之内的时刻写，其余的列（角色在内）不变）。Task 7 的离开也用它。`memberWorld.remove`。
- `TestEachWriteOnAProjectSharesItsWorkspaceFirst` 加移出一行：之前加入的 dave（Web）、erin（Ops）被移出。

**Tests:**
- `remove_member_test.go`：`TestRemoveProjectMember`（项目管理员 bob 移出成员 alice、访客 carol、另一位管理员 frank；工作区管理员 gina（项目成员）移出成员 alice、访客 ivy；调用顺序，结束时角色和 id 不变）；`TestRemoveProjectMemberRefuses`（15 个情形，同改角色的取锁一半；判定之后：已结束 404，自己的 409（工作区管理员的也是），gina 移出管理员 bob、移出锁下重读到的管理员 alice（等锁时被提拔）各 403 `project.role_too_high`；回答别的 id 的成员关系是写自己的错误）；`TestRemoveProjectMemberReturnsEachFailure`（7 个）。
- `clock_test.go` 加 `removeProjectMember` 一行。
- `adapter/http/member_writes_test.go`：`TestRemoveProjectMember`（204 无正文；每个拒绝照契约答，失败 500）。
- `rules_test.go` 的 `project_member.remove` 一行。
- `project_removal_test.go`：`TestRemovingAProjectMember`：一步接一步，每步一个 `memberEnding`：carol（Web 的成员）移出 erin 403；bob（Web 的管理员）移出自己 409，gina（成员、acme 的管理员）移出自己 409；gina 移出管理员 bob 403 `project.role_too_high`；bob 移出另一位管理员 dave、访客 erin 204，gina 移出成员 carol 204（同她的角色）；bob 再移出 dave 404；dave（acme 的成员）加入 Web：原来那一行回来，15（原来的 20 以工作区角色为上限，3.5），建立时刻不变；最后的 `standing`。
- 矩阵：四行 48 格。

- [ ] **Step 1: 接口描述**

`api/modules/project.yaml`（修改，1 处）：

````old api/modules/project.yaml
                $ref: '#/components/schemas/ProjectMember'
````
````new api/modules/project.yaml
                $ref: '#/components/schemas/ProjectMember'
        default:
          $ref: '#/components/responses/Problem'
    delete:
      operationId: removeProjectMember
      tags: [project]
      summary: Remove a member from a project
      description: >-
        For the project's admins, and its members who are the workspace's
        admins. A membership that does not exist or is deleted, or whose
        project the caller does not see, answers project.member_not_found; a
        caller who sees the project but may not remove its members,
        forbidden, whatever the membership. To a caller who may remove
        members, a membership that has ended answers
        project.member_not_found; his own, project.own_membership; and one
        whose role in the
        project is above his own, project.role_too_high, the workspace's
        admins included. The membership ends, its row and its role kept, at
        the moment of the request, by the caller; the member's display
        settings in the project stay. The role is decided after the
        workspace and project rows are locked.
      security: [{bearer: []}]
      x-problem-codes: [project.member_not_found, forbidden, project.own_membership, project.role_too_high]
      responses:
        '204':
          description: The membership has ended.
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `64edf735e0cdb360e0eb2b5ea42d983fe3d2c26713dc94ec1c2fe3a0c5e7ed51` | 2478 | `api/dist/openapi.yaml` |
| `2c9a42319c3d54935f8f2e786d89a9a6a6050c0f948124ee96dbb074d0e4d699` | 2188 | `server/internal/modules/project/adapter/http/gen/server.gen.go` |
| `7c5284d237c60e40b76b924491728fb16b73b5b2178c2c2ec40f2aed0c6f5318` | 2707 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: 操作名、规则、用例**

`server/internal/modules/project/domain/actions.go`（修改，2 处）：

````old server/internal/modules/project/domain/actions.go
	ActionMemberUpdate shared.Action = "project_member.update"
````
````new server/internal/modules/project/domain/actions.go
	ActionMemberUpdate shared.Action = "project_member.update"
	// ActionMemberRemove is removing a member from a project:
	// removeProjectMember.
	ActionMemberRemove shared.Action = "project_member.remove"
````

````old server/internal/modules/project/domain/actions.go
		ActionMemberList, ActionMemberAdd, ActionJoin, ActionMemberUpdate}
````
````new server/internal/modules/project/domain/actions.go
		ActionMemberList, ActionMemberAdd, ActionJoin, ActionMemberUpdate, ActionMemberRemove}
````

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"project_member.update": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
````
````new server/internal/modules/access/domain/rules.go
	"project_member.update": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	// As project_member.update (M3 design 3.5, 9.2; Plane
	// views/project/member.py:290); one's own membership and a higher
	// role are the use case's (3.5).
	"project_member.remove": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"project_member.update": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible,
````
````new server/internal/modules/access/domain/rules_test.go
	"project_member.update": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible,
		invisible, forbidden, invisible, invisible, invisible, forbidden, forbidden},
	// As project_member.update.
	"project_member.remove": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible,
````

`server/internal/modules/project/app/remove_member.go`（新文件，56 行）：

````file server/internal/modules/project/app/remove_member.go
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// RemoveProjectMember ends another member's project membership: DELETE
// /api/v0/project-members/{project_member_id} (M3 design 3.5).
type RemoveProjectMember struct {
	locks   Locks
	members MemberEnder
	tx      shared.TxManager
	clock   Clock
}

// NewRemoveProjectMember returns the use case.
func NewRemoveProjectMember(locks Locks, members MemberEnder, tx shared.TxManager, clock Clock) *RemoveProjectMember {
	return &RemoveProjectMember{locks: locks, members: members, tx: tx, clock: clock}
}

// Execute, in one transaction, in the order of M3 design 3.6: the
// membership's locks (Locks.lockMemberAndDecide: the membership read for
// its project and workspace, the workspace FOR SHARE, the project FOR NO
// KEY UPDATE, the membership read again) and the decision on
// project_member.remove; then the checks that only a caller allowed to
// remove members gets to see: an ended membership is
// project.member_not_found; then 3.5's rule (domain.CheckRemoval): his own
// membership, and one whose role is above his project role, are refused;
// then the membership ended, its row and its role kept, by the caller at
// the time the clock gives under the locks. A caller who may remove members
// is a project admin, or a project member who is the workspace's admin and
// removes no one above his own role: the project keeps an admin.
func (u *RemoveProjectMember) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	return u.tx.WithinTx(ctx, func(ctx context.Context) error {
		h, err := u.locks.lockMemberAndDecide(ctx, actor, id, domain.ActionMemberRemove, false)
		if err != nil {
			return err
		}
		m := h.member
		if !m.Active {
			return domain.ErrMemberNotFound
		}
		if err := domain.CheckRemoval(h.grant.ProjectRole, m.MemberID == actor.UserID, m.Role); err != nil {
			return err
		}
		return u.members.EndMember(ctx, m.ProjectID, m.MemberID, actor.UserID, u.clock.Now())
	})
}
````

`server/internal/modules/project/app/fakes_member_test.go`（修改，1 处）：

````old server/internal/modules/project/app/fakes_member_test.go
}

// frank is a project admin in newMemberWrites; the memberships it adds,
````
````new server/internal/modules/project/app/fakes_member_test.go
}

func (f *fakeStore) EndMember(ctx context.Context, projectID, userID, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "EndMember %s of %s by %s at %s", userID, projectID, by, now.Format(timeFormat))
	if err := f.fail("EndMember"); err != nil {
		return err
	}
	if p, ok := f.projects[projectID]; ok {
		if m, member := p.members[userID]; member && m.Active {
			m.Active = false
			p.members[userID] = m
			return nil
		}
	}
	return fmt.Errorf("EndMember: %s has no active membership of %s", userID, projectID)
}

// frank is a project admin in newMemberWrites; the memberships it adds,
````

`server/internal/modules/project/app/remove_member_test.go`（新文件，138 行）：

````file server/internal/modules/project/app/remove_member_test.go
package app_test

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newRemoveMember is RemoveProjectMember over newMemberWrites' fakes, its
// clock logged.
func newRemoveMember() (*app.RemoveProjectMember, *writeFixture) {
	f := newMemberWrites()
	return app.NewRemoveProjectMember(f.locks(), f.store, f.tx, clockAt{clockNow, f.log}), f
}

// removed are the calls of caller's removal of user's membership of web:
// its locks and decision, no workspace membership among them, the clock,
// the ending at that time.
func removed(caller, user uuid.UUID) []string {
	id := newMemberWrites().memberOf(webID, user)
	return append(memberLocked(id, user, webID, caller, domain.ActionMemberRemove, false), "Now",
		fmt.Sprintf("EndMember %s of %s by %s at %s", user, webID, caller, clockNow.Format(timeFormat)))
}

// RemoveProjectMember, in one transaction and in the order of M3 design
// 3.6, takes the membership's locks, decides, reads the clock, then ends
// the membership, by the caller at that time; it keeps its role. bob, a
// project admin, removes alice, a member, carol, a guest, and frank,
// another admin (3.5 refuses only a higher role); gina, the workspace's
// admin and a project member, removes alice, a member, and ivy, a guest.
func TestRemoveProjectMember(t *testing.T) {
	for _, tt := range []struct{ caller, user uuid.UUID }{{bob, alice}, {bob, carol}, {bob, frank}, {gina, alice}, {gina, ivy}} {
		uc, f := newRemoveMember()
		before := f.store.projects[webID].members[tt.user]
		if err := uc.Execute(as(tt.caller), f.memberOf(webID, tt.user)); err != nil || !slices.Equal(f.log.calls, removed(tt.caller, tt.user)) {
			t.Errorf("Execute() by %s of %s = %v, calls\n%q\nwant\n%q", tt.caller, tt.user, err, f.log.calls, removed(tt.caller, tt.user))
		}
		if m := f.store.projects[webID].members[tt.user]; m.Active || m.Role != before.Role || m.ID != before.ID {
			t.Errorf("the membership after the removal: %+v; want %+v ended", m, before)
		}
	}
}

// Refusals, each in its place, and nothing ended: no caller, before the
// transaction; a membership that is not there, a workspace, project or
// membership deleted while a lock waited, a project or membership moved
// meanwhile, and a caller who does not see the project, each
// project.member_not_found; a project member, the Authorizer's 403. Then,
// to a caller who may remove members, after the decision: an ended
// membership, 404; his own, 409, the workspace's admin's too; a higher
// role, 403 project.role_too_high, from the workspace's admin too. A
// membership answered for another id is the write's own error.
func TestRemoveProjectMemberRefuses(t *testing.T) {
	locked := func(f *writeFixture, caller, user uuid.UUID) []string {
		return memberLocked(f.memberOf(webID, user), user, webID, caller, domain.ActionMemberRemove, false)
	}
	upTo := func(n int) func(f *writeFixture, caller, user uuid.UUID) []string {
		return func(f *writeFixture, caller, user uuid.UUID) []string { return locked(f, caller, user)[:n] }
	}
	for _, tt := range []struct {
		name         string
		caller, user uuid.UUID // no caller when caller is zero; a membership not there when user is zero
		set          func(f *writeFixture)
		want         error
		calls        func(f *writeFixture, caller, user uuid.UUID) []string
	}{
		{"no caller", uuid.UUID{}, alice, nil, shared.Unauthenticated(), func(*writeFixture, uuid.UUID, uuid.UUID) []string { return nil }},
		{"no membership", bob, uuid.UUID{}, nil, domain.ErrMemberNotFound,
			func(*writeFixture, uuid.UUID, uuid.UUID) []string {
				return []string{"Begin", "MemberByID " + uuid.Nil().String()}
			}},
		{"acme deleted while its lock waited", bob, alice, func(f *writeFixture) { f.workspaces.gone = true }, domain.ErrMemberNotFound, upTo(3)},
		{"web deleted while its lock waited", bob, alice, func(f *writeFixture) { f.store.deleted = true }, domain.ErrMemberNotFound, upTo(4)},
		{"web moved to another workspace", bob, alice, func(f *writeFixture) { f.store.moved = uuid.NewV7() }, domain.ErrMemberNotFound, upTo(4)},
		{"the membership deleted while the locks waited", bob, alice, func(f *writeFixture) { f.store.reread.gone = true },
			domain.ErrMemberNotFound, upTo(5)},
		{"the membership moved to ops", bob, alice, func(f *writeFixture) { f.store.reread.project = opsID }, domain.ErrMemberNotFound, upTo(5)},
		{"a caller who does not see web", erin, alice, nil, domain.ErrMemberNotFound, locked},
		{"a project member", alice, carol, nil, shared.Forbidden(), locked},
		{"an ended membership", bob, dave, nil, domain.ErrMemberNotFound, locked},
		{"his own", bob, bob, nil, domain.ErrOwnMembership, locked},
		{"the workspace's admin's own", gina, gina, nil, domain.ErrOwnMembership, locked},
		{"an admin, by the workspace's admin who is a project member", gina, bob, nil, domain.ErrRoleTooHigh, locked},
		{"a member made an admin while the locks waited, by the workspace's admin", gina, alice,
			func(f *writeFixture) { f.store.reread.role = shared.RoleAdmin }, domain.ErrRoleTooHigh, locked},
		{"a membership answered for another id", bob, alice, func(f *writeFixture) { f.store.answersAs = uuid.NewV7() }, nil, upTo(2)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newRemoveMember()
			if tt.set != nil {
				tt.set(f)
			}
			ctx, id := context.Background(), uuid.Nil()
			if tt.caller != (uuid.UUID{}) {
				ctx = as(tt.caller)
			}
			if tt.user != (uuid.UUID{}) {
				id = f.memberOf(webID, tt.user)
			}
			before := f.store.projects[webID].members[tt.user]
			outcome{tt.name, tt.want, tt.calls(f, tt.caller, tt.user)}.check(t, uc.Execute(ctx, id), f)
			if after := f.store.projects[webID].members[tt.user]; after != before {
				t.Errorf("the membership after the refusal: %+v, want it as it was, %+v", after, before)
			}
		})
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after.
func TestRemoveProjectMemberReturnsEachFailure(t *testing.T) {
	all := removed(bob, alice)
	for _, tt := range []struct {
		name  string
		fail  func(f *writeFixture)
		calls int // how many of the removal's calls ran
	}{
		{"the read", func(f *writeFixture) { f.store.errs = map[string]error{"MemberByID": errDisk} }, 2},
		{"the workspace's lock", func(f *writeFixture) { f.workspaces.err = errDisk }, 3},
		{"the project's lock", func(f *writeFixture) { f.store.errs = map[string]error{"LockProject": errDisk} }, 4},
		{"the read under the locks", func(f *writeFixture) { f.store.reread.err = errDisk }, 5},
		{"the decision", func(f *writeFixture) { f.auth.errs = map[grantKey]error{{bob, acme.ID}: errDisk} }, 6},
		{"the ending", func(f *writeFixture) { f.store.errs = map[string]error{"EndMember": errDisk} }, 8},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 8},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newRemoveMember()
			tt.fail(f)
			outcome{tt.name, errDisk, all[:tt.calls]}.check(t, uc.Execute(as(bob), aliceInWeb), f)
		})
	}
}
````

`server/internal/modules/project/app/clock_test.go`（修改，1 处）：

````old server/internal/modules/project/app/clock_test.go
		}, roleChanged(newMemberWrites(), bob, alice, shared.RoleGuest)},
````
````new server/internal/modules/project/app/clock_test.go
		}, roleChanged(newMemberWrites(), bob, alice, shared.RoleGuest)},
		{"removeProjectMember", func() ([]string, error) {
			uc, f := newRemoveMember()
			err := uc.Execute(as(bob), aliceInWeb)
			return f.log.calls, err
		}, removed(bob, alice)},
````

- [ ] **Step 3: handler、接线**

`server/internal/modules/project/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/project/adapter/http/handler.go
}

// UseCases are the use cases behind the module's operations.
````
````new server/internal/modules/project/adapter/http/handler.go
}

// RemoveMemberUseCase is app.RemoveProjectMember.
type RemoveMemberUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) error
}

// UseCases are the use cases behind the module's operations.
````

````old server/internal/modules/project/adapter/http/handler.go
	UpdateMember      UpdateMemberUseCase
````
````new server/internal/modules/project/adapter/http/handler.go
	UpdateMember      UpdateMemberUseCase
	RemoveMember      RemoveMemberUseCase
````

`server/internal/modules/project/adapter/http/members.go`（修改，1 处）：

````old server/internal/modules/project/adapter/http/members.go
}

// members is list as the API shows it: data an array, never null.
````
````new server/internal/modules/project/adapter/http/members.go
}

// RemoveProjectMember serves DELETE
// /api/v0/project-members/{project_member_id}.
func (h handler) RemoveProjectMember(ctx context.Context, req gen.RemoveProjectMemberRequestObject) (gen.RemoveProjectMemberResponseObject, error) {
	if err := h.uc.RemoveMember.Execute(ctx, req.ProjectMemberID); err != nil {
		return nil, err
	}
	return gen.RemoveProjectMember204Response{}, nil
}

// members is list as the API shows it: data an array, never null.
````

`server/internal/modules/project/adapter/http/handler_test.go`（修改，3 处）：

````old server/internal/modules/project/adapter/http/handler_test.go
	updateMember *fakeUpdateMember
````
````new server/internal/modules/project/adapter/http/handler_test.go
	updateMember *fakeUpdateMember
	removeMember *fakeDelete
````

````old server/internal/modules/project/adapter/http/handler_test.go
		f.updateMember = &fakeUpdateMember{}
	}
````
````new server/internal/modules/project/adapter/http/handler_test.go
		f.updateMember = &fakeUpdateMember{}
	}
	if f.removeMember == nil {
		f.removeMember = &fakeDelete{}
	}
````

````old server/internal/modules/project/adapter/http/handler_test.go
		JoinProject: f.join, UpdateMember: f.updateMember})
````
````new server/internal/modules/project/adapter/http/handler_test.go
		JoinProject: f.join, UpdateMember: f.updateMember, RemoveMember: f.removeMember})
````

`server/internal/modules/project/adapter/http/member_writes_test.go`（修改，1 处）：

````old server/internal/modules/project/adapter/http/member_writes_test.go
			t.Errorf("%s: PATCH = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

````
````new server/internal/modules/project/adapter/http/member_writes_test.go
			t.Errorf("%s: PATCH = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

// DELETE goes to the use case for the caller and the path's membership,
// and answers 204 with no body; the use case's refusals, as the contract
// declares them, and its failure.
func TestRemoveProjectMember(t *testing.T) {
	remove := &fakeDelete{}
	h := newServer(t, fakes{removeMember: remove})
	path := "/api/v0/project-members/" + bobInWeb.ID.String()
	if res, body := do(t, h, request(http.MethodDelete, path, "alice", "")); res.StatusCode != http.StatusNoContent || body != "" {
		t.Errorf("DELETE = %d %q, want 204 and no body", res.StatusCode, body)
	}
	if want := []string{"alice " + bobInWeb.ID.String()}; !slices.Equal(remove.calls, want) {
		t.Errorf("calls = %q, want %q", remove.calls, want)
	}
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"no membership", domain.ErrMemberNotFound, http.StatusNotFound, memberNotFoundJSON},
		{"a project member", shared.Forbidden(), http.StatusForbidden, forbiddenJSON},
		{"his own", domain.ErrOwnMembership, http.StatusConflict, ownMembershipJSON},
		{"a higher role", domain.ErrRoleTooHigh, http.StatusForbidden, roleTooHighJSON},
		{"a failure", errGone, http.StatusInternalServerError, internalErrorJSON},
	} {
		h := newServer(t, fakes{removeMember: &fakeDelete{err: tt.err}})
		if res, body := do(t, h, request(http.MethodDelete, path, "alice", "")); res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: DELETE = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

````

`server/internal/modules/project/module.go`（修改，2 处）：

````old server/internal/modules/project/module.go
// members, changing a member's role, each member's display settings,
// carries out the workspace module's cascades on the projects
// (ProjectCascade), and offers the access module its reads of a project
// (ProjectAccess) and the workspace module its count of an account's ended
// project memberships (ProjectMembershipCounts).
````
````new server/internal/modules/project/module.go
// members, changing a member's role and removing a member, each member's
// display settings, carries out the workspace module's cascades on the
// projects (ProjectCascade), and offers the access module its reads of a
// project (ProjectAccess) and the workspace module its count of an
// account's ended project memberships (ProjectMembershipCounts).
````

````old server/internal/modules/project/module.go
		UpdateMember:      app.NewUpdateProjectMember(locks, store, d.Tx, d.Clock),
````
````new server/internal/modules/project/module.go
		UpdateMember:      app.NewUpdateProjectMember(locks, store, d.Tx, d.Clock),
		RemoveMember:      app.NewRemoveProjectMember(locks, store, d.Tx, d.Clock),
````

- [ ] **Step 4: 矩阵、最先锁工作区、组合出的移出**

`server/internal/bootstrap/permission_matrix_membership_test.go`（修改，6 处）：

````old server/internal/bootstrap/permission_matrix_membership_test.go
// membership (M3 design 9.2): changing a member's role. Each names the
// membership by its id (/project-members/{project_member_id}), in its
// column's project; prepareMatrix's preconditions hold each membership a
// row names to the state the row says.
````
````new server/internal/bootstrap/permission_matrix_membership_test.go
// membership (M3 design 9.2): changing a member's role, removing a member.
// Each names the membership by its id
// (/project-members/{project_member_id}), in its column's project;
// prepareMatrix's preconditions hold each membership a row names to the
// state the row says.
````

````old server/internal/bootstrap/permission_matrix_membership_test.go
	cellProjectOwnMembership  = cell{http.StatusConflict, "project.own_membership"}
````
````new server/internal/bootstrap/permission_matrix_membership_test.go
	cellProjectOwnMembership  = cell{http.StatusConflict, "project.own_membership"}
	cellRoleTooHigh           = cell{http.StatusForbidden, "project.role_too_high"}
````

````old server/internal/bootstrap/permission_matrix_membership_test.go
			refusal: "role not_allowed"},
	}
````
````new server/internal/bootstrap/permission_matrix_membership_test.go
			refusal: "role not_allowed"},
		// As updateProjectMember (M3 design 3.5, 9.2): PM's membership ends;
		// PM+WA, a member, removes a member, of his own role.
		{op: "removeProjectMember", write: true, columns: projectColumns, request: toProjectMembership(http.MethodDelete, "", projectMemberOf),
			cells: ofMembership(cellNoContent, cellForbidden, cellForbidden, cellNoContent, cellForbidden, cellForbidden)},
		// Nobody removes his own membership, the workspace's admin neither:
		// he leaves the project (M3 design 3.5).
		{op: "removeProjectMember", variant: "one's own membership", write: true, columns: projectColumns,
			request: toProjectMembership(http.MethodDelete, "", ownMembershipOf),
			cells:   ofMembership(cellProjectOwnMembership, cellForbidden, cellForbidden, cellProjectOwnMembership, cellForbidden, cellForbidden)},
		{op: "removeProjectMember", variant: "an ended membership", write: true, columns: projectColumns,
			request: toProjectMembership(http.MethodDelete, "", endedMembershipOf),
			cells:   ofMembership(cellProjectMemberNotFound, cellForbidden, cellForbidden, cellProjectMemberNotFound, cellForbidden, cellForbidden)},
		// The admin's membership: PA's own (409); PM+WA, a member, may not
		// remove a role above his own, though he is the workspace's admin
		// (M3 design 3.5: no exception here).
		{op: "removeProjectMember", variant: "an admin's membership", write: true, columns: projectColumns,
			request: toProjectMembership(http.MethodDelete, "", adminOf),
			cells:   ofMembership(cellProjectOwnMembership, cellForbidden, cellForbidden, cellRoleTooHigh, cellForbidden, cellForbidden)},
	}
}

// adminOf: PA's membership of the column's project, an admin's; in gone's
// project, gone's admin's.
func adminOf(c caller) (string, caller) {
	if key := projectOf(c); key != "gone/project" {
		return key, callerProjectAdmin
	}
	return "gone/project", callerDeleted
````

````old server/internal/bootstrap/permission_matrix_membership_test.go
// the private one ended, not deleted; gone's member's of gone's project
// deleted with gone. The workspace guest is acme's active guest, so that a
````
````new server/internal/bootstrap/permission_matrix_membership_test.go
// the private one ended, not deleted; gone's member's and admin's of gone's
// project deleted with gone. The workspace guest is acme's active guest, so that a
````

````old server/internal/bootstrap/permission_matrix_membership_test.go
		{"acme/public", callerProjectAdmin, shared.RoleAdmin, true, true}, {"acme/public", callerMemberAndAdmin, shared.RoleMember, true, true},
````
````new server/internal/bootstrap/permission_matrix_membership_test.go
		{"acme/public", callerProjectAdmin, shared.RoleAdmin, true, true}, {"acme/private", callerProjectAdmin, shared.RoleAdmin, true, true},
		{"acme/public", callerMemberAndAdmin, shared.RoleMember, true, true},
````

````old server/internal/bootstrap/permission_matrix_membership_test.go
		{"gone/project", callerMember, 0, false, false},
````
````new server/internal/bootstrap/permission_matrix_membership_test.go
		{"gone/project", callerMember, 0, false, false}, {"gone/project", callerDeleted, 0, false, false},
````

`server/internal/bootstrap/project_write_locks_test.go`（修改，1 处）：

````old server/internal/bootstrap/project_write_locks_test.go
		targets: [2]string{"bob", "carol"}, member: [2]string{"bob", "carol"}},
````
````new server/internal/bootstrap/project_write_locks_test.go
		targets: [2]string{"bob", "carol"}, member: [2]string{"bob", "carol"}},
	// The joiners before, dave in Web and erin in Ops, removed.
	{op: "removeProjectMember", method: http.MethodDelete, path: "/api/v0/project-members/%s", want: http.StatusNoContent,
		member: [2]string{"dave", "erin"}},
````

`server/internal/bootstrap/project_removal_test.go`（新文件，120 行）：

````file server/internal/bootstrap/project_removal_test.go
package bootstrap

import (
	"maps"
	"net/http"
	"testing"
	"time"
	"uuid"
)

// remove is by's removal of name's membership of project: its status and
// body.
func (w memberWorld) remove(t *testing.T, by string, project uuid.UUID, name string) (int, string) {
	t.Helper()
	return call(t, w.contract, http.MethodDelete, w.base+"/api/v0/project-members/"+w.membership(t, project, name).String(), w.tokens[by], "")
}

// memberEnding is a step that ends name's membership of project, by by,
// or is refused with code: send sends its request, which answers status.
type memberEnding struct {
	step, name, by string
	project        uuid.UUID
	status         int
	code           string // of a refusal
	send           func() (int, string)
}

// check runs e and checks it: it changes no row besides the membership's;
// a refusal leaves the membership as it was; an ending ends it, not deleted,
// by e.by at a moment within its request, its other columns as they were,
// its role among them. The member has his display settings in the project
// before, which stay.
func (e memberEnding) check(t *testing.T, w memberWorld) {
	t.Helper()
	id := w.membership(t, e.project, e.name)
	target, others := rowJSON(t, w.pool, "project_members", id), rowsBut(t, w.pool, []uuid.UUID{id})
	var settings int
	if err := w.pool.QueryRow(soon(t), "SELECT count(*) FROM project_user_properties WHERE project_id = $1 AND user_id = $2 AND "+
		"deleted_at IS NULL", e.project, w.ids[e.name]).Scan(&settings); err != nil || settings != 1 {
		t.Fatalf("%s: %s's display settings in the project: %d, %v; want his one", e.step, e.name, settings, err)
	}
	before := time.Now().Truncate(time.Microsecond)
	status, body := e.send()
	after := time.Now()
	if status != e.status || e.code != "" && problemCode(t, []byte(body)) != e.code {
		t.Fatalf("%s = %d %s, want %d %s", e.step, status, body, e.status, e.code)
	}
	if got := rowsBut(t, w.pool, []uuid.UUID{id}); !maps.Equal(got, others) {
		t.Errorf("%s changed rows besides the target's:\n%v\nwant them as they were:\n%v", e.step, got, others)
	}
	ended := rowJSON(t, w.pool, "project_members", id)
	if e.code != "" {
		if !maps.Equal(ended, target) {
			t.Errorf("%s changed the target's membership:\n%v\nwant it as it was:\n%v", e.step, ended, target)
		}
		return
	}
	at, err := time.Parse(time.RFC3339Nano, ended["updated_at"].(string))
	if ended["is_active"] != false || ended["deleted_at"] != nil || ended["updated_by_id"] != w.ids[e.by].String() || err != nil ||
		at.Before(before) || at.After(after) {
		t.Errorf("%s: the target's membership %v; want it ended, not deleted, by %s within %v to %v", e.step, ended, e.by, before, after)
	}
	for _, column := range []string{"is_active", "updated_by_id", "updated_at"} {
		delete(ended, column)
		delete(target, column)
	}
	if !maps.Equal(ended, target) {
		t.Errorf("%s changed the target's other columns, its role among them:\n%v\nwant them as they were:\n%v", e.step, ended, target)
	}
}

// Removing a project member on the wired app (M3 design 3.5), one removal
// after another on memberWorld, each a memberEnding; the project keeps an
// admin.
//   - carol, Web's member, may remove nobody: erin, a guest, 403;
//   - bob, Web's admin, cannot remove himself (409); nor can gina, its
//     member and acme's admin (409);
//   - gina cannot remove bob, Web's admin, though she is acme's admin
//     (403 project.role_too_high: 3.5 gives no exception here);
//   - bob removes dave, another admin, and erin, a guest; gina removes
//     carol, a member of her own role;
//   - dave's ended membership is 404 to bob;
//   - dave, a member of acme, joins Web again: his row is back, a
//     member's, 15, the lesser of its 20 and his workspace role (3.5,
//     3.6 convention 6), made when it was.
func TestRemovingAProjectMember(t *testing.T) {
	w := newMemberWorld(t)
	daves := w.membership(t, w.web, "dave")
	for _, step := range []struct {
		name, by, member string
		status           int
		code             string // of a refusal
	}{
		{"carol, Web's member, removes erin", "carol", "erin", http.StatusForbidden, "forbidden"},
		{"bob, Web's admin, removes himself", "bob", "bob", http.StatusConflict, "project.own_membership"},
		{"gina, Web's member and acme's admin, removes herself", "gina", "gina", http.StatusConflict, "project.own_membership"},
		{"gina removes bob, Web's admin", "gina", "bob", http.StatusForbidden, "project.role_too_high"},
		{"bob removes dave, another admin", "bob", "dave", http.StatusNoContent, ""},
		{"bob removes erin, a guest", "bob", "erin", http.StatusNoContent, ""},
		{"gina removes carol, a member", "gina", "carol", http.StatusNoContent, ""},
		{"bob removes dave again", "bob", "dave", http.StatusNotFound, "project.member_not_found"},
	} {
		memberEnding{step: step.name, name: step.member, by: step.by, project: w.web, status: step.status, code: step.code,
			send: func() (int, string) { return w.remove(t, step.by, w.web, step.member) }}.check(t, w)
	}
	status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/projects/"+w.web.String()+"/join", w.tokens["dave"], "")
	var joined struct {
		MemberRole *int `json:"member_role"`
	}
	if status != http.StatusOK {
		t.Fatalf("dave's joining Web again = %d %s", status, body)
	}
	if decodeAnswer(t, body, &joined); joined.MemberRole == nil || *joined.MemberRole != 15 || w.membership(t, w.web, "dave") != daves {
		t.Errorf("dave joins Web again as %s, his membership %s; want his row back, %s, as a member", body, w.membership(t, w.web, "dave"), daves)
	}
	if got, want := w.standing(t), "acme: alice 20, bob 15, carol 15, dave 15, erin 5, gina 20; beta: bob 20, carol 15; "+
		"Lab: bob 20, carol 15; Ops: alice 20, bob 15, carol 20; Web: alice 20, bob 20, dave 15, gina 15"; got != want {
		t.Errorf("the world after the removals: %s; want %s", got, want)
	}
}
````

- [ ] **Step 5: 测试、lint、前端检查**

Run: `go -C server test -count=1 ./internal/modules/project/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix$|TestEachWriteOnAProjectSharesItsWorkspaceFirst|TestRemovingAProjectMember$|TestAPIRoutesAreTheContractsOperations' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

- [ ] **Step 6: 提交**

```bash
git add api/modules/project.yaml server/internal/bootstrap/permission_matrix_membership_test.go server/internal/bootstrap/project_removal_test.go server/internal/bootstrap/project_write_locks_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/project/adapter/http/handler.go server/internal/modules/project/adapter/http/handler_test.go server/internal/modules/project/adapter/http/member_writes_test.go server/internal/modules/project/adapter/http/members.go server/internal/modules/project/app/clock_test.go server/internal/modules/project/app/fakes_member_test.go server/internal/modules/project/app/remove_member.go server/internal/modules/project/app/remove_member_test.go server/internal/modules/project/domain/actions.go server/internal/modules/project/module.go api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P5b): removeProjectMember

DELETE /api/v0/project-members/{project_member_id}, for a project's
admins and its members who are the workspace's admins: one's own
membership is refused, and so is one whose role is above the caller's,
workspace admins included. The membership ends, its row and role kept,
by the caller under the locks. The matrix gains its four rows, the
first-lock test its row, and the wired app's removals end memberships
that a join later brings back no higher than the workspace role.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A，清扫 2、4、5、14、24）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| 规则给项目成员 | `TestEveryRuleDecidesItsCells`；`TestPermissionMatrix`、`TestRemovingAProjectMember`（故事看不到：故事里的项目成员不移出别人） | 单元；组合 |
| 已结束的成员关系又被移出 | `TestRemoveProjectMemberRefuses`；`TestRemovingAProjectMember`、`TestPermissionMatrix`、`TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile`（Task 8 起） | 单元；组合 |
| 结束的失败被吞掉 | `TestRemoveProjectMemberReturnsEachFailure` | 单元 |
| 移出以调用者是管理员来判（`CheckRemoval(RoleAdmin, …)`） | `TestRemoveProjectMemberRefuses`；`TestRemovingAProjectMember`、`TestPermissionMatrix`（PM+WA 移出 PA） | 单元；组合 |
| 移出看不到自己的成员关系 | `TestRemoveProjectMemberRefuses`；`TestRemovingAProjectMember`、`TestPermissionMatrix` | 单元；组合 |
| 相对规则用锁之前读到的角色（`h.member.Role = m.Role`）；重读挪到判定之后 | `TestRemoveProjectMemberRefuses`；`TestAWriteOnAProjectMembershipDecidesOnWhatItReadUnderItsLocks`（Task 8 起） | 单元；组合 |
| 写成被移出的人自己结束的 | `TestRemoveProjectMember`；`TestRemovingAProjectMember`；P5 | 单元；组合；端到端 |
| `project.New` 的移出不结束任何行；用停在 2001 年的时钟 | `TestRemovingAProjectMember`（`memberEnding` 的时刻）；`TestTheWritesOnAProjectStampTheirRequest`、`TestEachLockOfAWriteOnAProjectMembershipIsItsStrength`（Task 8 起） | 组合 |
| 去掉 `projectWrites` 的移出一行 | `TestEachWriteOnAProjectSharesItsWorkspaceFirst`（完整性核对的第二种形状） | 组合 |
| 契约（`api/modules/project.yaml`）多声明一个从不回答的码 | `project/adapter/http` 的 `apitest.Main`（包的测试结束时） | 单元 |

**Done when:** 移出的四行矩阵（48 格）通过；组合出的移出拒绝时一行不改，结束时只改目标的三列；重新加入回到 15。

---

### Task 7: `leaveProject`；`project.sole_admin` 为两条规则措辞

**Files:**
- Create: `server/internal/bootstrap/project_leaving_test.go`、`server/internal/modules/project/app/leave_project.go`、`server/internal/modules/project/app/leave_project_test.go`
- Modify: `api/modules/project.yaml`、`api/openapi.yaml`、`server/internal/bootstrap/permission_matrix_membership_test.go`、`server/internal/bootstrap/project_write_locks_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/project/adapter/http/handler.go`、`server/internal/modules/project/adapter/http/handler_test.go`、`server/internal/modules/project/adapter/http/member_writes_test.go`、`server/internal/modules/project/adapter/http/members.go`、`server/internal/modules/project/app/clock_test.go`、`server/internal/modules/project/app/fakes_member_test.go`、`server/internal/modules/project/domain/actions.go`、`server/internal/modules/project/domain/errors.go`、`server/internal/modules/project/module.go`、`server/internal/modules/workspace/adapter/http/members_test.go`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.10，M3 设计 3.7 规则 1、5.1、9.2，P5a review 第 6 节）：`POST /api/v0/projects/{project_id}/leave`，204 无正文；码 `[project.not_found, forbidden, project.sole_admin]`。
- `domain.ActionLeave`（`project.leave`），规则 `{Level: LevelProject, Roles: [admin, member, guest]}`（同 `project_member.list`：每个有效成员），进 `Actions()`。
- `LeaveProject`（`NewLeaveProject(locks, members MemberLeaver, tx, clock)`），`Execute(ctx, projectID) error`：没有调用者 401 → 一个事务：`lockAndDecide(write{project, project.leave})`（P4b 的按项目 id 的路径：工作区 S → 项目 N → 判定；看不到 404 `project.not_found`，不是成员 403）→ 调用者的项目角色是管理员时 `HasOtherAdmin`，没有另一位有效管理员 409 `project.sole_admin`（他是唯一的成员时也是）→ 读时钟 → `EndMember(项目, 他, 他, 时刻)`。规则看锁下判定得到的项目角色，不看工作区角色：PM+WA 照成员离开，降为访客的前管理员照访客离开。
- `ErrSoleAdmin` 的说明和文字为规则 1、2 两条措辞，补救是"先给项目另一位管理员，或者删除它"（spec 2.10：对每个收到它的调用者都成立，清扫 30）；`workspace` 的 HTTP 测试（它声明这个码，不导入 `project`）和两份 `auth.json` 同改。
- HTTP：`LeaveProjectUseCase`；`UseCases.LeaveProject`；`(handler).LeaveProject`。`project.New` 的 `LeaveProject`。单元的假存储多 `HasOtherAdmin`（记作 `HasOtherAdmin <项目> but <账户>`）。
- 矩阵：离开一行 `ofProject(409 project.sole_admin, 204, 204, 204, 403, 403)`（PA 是每个项目唯一的有效管理员；PM+WA 照成员离开）；前提：PA 在 acme 的公开、私有项目里没有另一位管理员（`HasOtherAdmin`）。这一行在项目表（`projectTables`）的列上，唯一管理员的项目一侧不另建表（spec 第 3 节第 5 条）。
- `TestEachWriteOnAProjectSharesItsWorkspaceFirst` 加离开一行：bob、carol（这时是访客）离开。
- `bootstrap/project_leaving_test.go`：`memberWorld.leave`。

**Tests:**
- `leave_project_test.go`：`TestLeaveProject`（成员 alice、访客 carol 和 ivy、PM+WA gina 离开，不问另一位管理员；管理员 bob、frank 各在另一位还在时离开，问过之后；只有他的那一行结束，角色不变）；`TestLeaveProjectRefuses`（10 个：没有调用者；没有项目、工作区或项目在等锁时删除、项目移到别的工作区、看不到项目，各 404；看得到、不是成员 403；另一位管理员已结束、另一位现在是成员、他是 ops 唯一的管理员和成员，各 409；每个情形两个项目的成员关系都不变）；`TestLeaveProjectReturnsEachFailure`（7 个）。
- `clock_test.go` 加 `leaveProject` 一行。
- `adapter/http/member_writes_test.go`：`TestLeaveProject`（204；每个拒绝照契约答，`project.sole_admin` 的新文字；失败 500）。
- `workspace/adapter/http/members_test.go`：`TestRemoveWorkspaceMemberRefusals`、`TestLeaveWorkspaceRefusals` 的 `project.sole_admin` 新文字。
- `rules_test.go` 的 `project.leave` 一行。
- `project_leaving_test.go`：`TestLeavingAProject`：一步接一步，每步一个 `memberEnding`，由他自己：bob 是 Lab 唯一的管理员，carol 还是成员时 409；她离开之后他仍 409（只有他一人也拒绝）；carol、erin 离开 Web；bob、dave（三位管理员中的两位）离开 Web；alice（现在唯一的管理员）409；carol（Ops 的管理员）离开；alice（Ops 的另一位）409：carol 已结束的管理员成员关系不算；gina 把 alice 降为 acme 的访客，她在 Web、Ops 也成为访客：两个项目都没有管理员了；gina（Web 的成员、acme 的管理员）照成员离开；bob 离开 Ops；alice（访客、Ops 唯一的成员）离开。最后的 `standing`。
- 矩阵：离开一行 12 格；`TestEachMatrixTableIsOfOneLevel` 照旧通过（没有新表）。

- [ ] **Step 1: 接口描述**

`api/modules/project.yaml`（修改，1 处）：

````old api/modules/project.yaml
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Project'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/projects/{project_id}/members:
````
````new api/modules/project.yaml
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Project'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/projects/{project_id}/leave:
    parameters:
      - $ref: '#/components/parameters/ProjectID'
    post:
      operationId: leaveProject
      tags: [project]
      summary: Leave a project
      description: >-
        Every active member of the project leaves it himself. A project that
        does not exist, is deleted, or that the caller does not see answers
        project.not_found; one he sees but is not a member of, forbidden.
        Its only active admin is refused project.sole_admin, also when he is
        its only member: the project gets another admin first, or is
        deleted. A member who is a workspace admin but no admin of the
        project leaves as any member. His membership ends, its row and its
        role kept, at the moment of the request; his display settings in
        the project stay. The role is decided after the workspace and
        project rows are locked.
      security: [{bearer: []}]
      x-problem-codes: [project.not_found, forbidden, project.sole_admin]
      responses:
        '204':
          description: The caller's membership has ended.
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/projects/{project_id}/members:
````

`api/openapi.yaml`（修改，1 处）：

````old api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1projects~1{project_id}~1join'
````
````new api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1projects~1{project_id}~1join'
  /api/v0/projects/{project_id}/leave:
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1projects~1{project_id}~1leave'
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `ab12a4bc5b39fbc1d2f6950fc7b1f9919f7a8e72a6bee8c51f3f1fb1b3b3eff5` | 2498 | `api/dist/openapi.yaml` |
| `8c5a9d64a449bc3d1e888e53b58f519739af4079bc5d6b2f79a1730a12bcd9a9` | 2287 | `server/internal/modules/project/adapter/http/gen/server.gen.go` |
| `dafea3376dfa1e2c87d2b688cfe6d79de610d9e687d5d962aa21e5640c00e006` | 2752 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: 领域、规则、用例**

`server/internal/modules/project/domain/errors.go`（修改，2 处）：

````old server/internal/modules/project/domain/errors.go
	// ErrSoleAdmin answers an ending of an account's project memberships
	// that would leave a project with other active members without an
	// active admin: he is its only one (M3 design 3.7 rule 2). The
	// workspace's removal and leaving declare it too (M3 design 5.1).
````
````new server/internal/modules/project/domain/errors.go
	// ErrSoleAdmin answers the leaving of a project's only active admin,
	// also when he is its only member (M3 design 3.7 rule 1), and an ending
	// of an account's project memberships that would leave a project with
	// other active members without an active admin: he is its only one
	// (rule 2). The workspace's removal and leaving declare it too (M3
	// design 5.1). Each who gets it can have the project given another
	// admin, through a workspace admin if no one else, or delete it.
````

````old server/internal/modules/project/domain/errors.go
		"Ending the membership would leave a project that has other members without an admin; make another of its members an admin first.")
````
````new server/internal/modules/project/domain/errors.go
		"The project would be left without an admin: its only active admin cannot leave it, nor can his membership end while it has "+
			"other active members. Give the project another admin first, or delete it.")
````

`server/internal/modules/project/domain/actions.go`（修改，2 处）：

````old server/internal/modules/project/domain/actions.go
	ActionMemberRemove shared.Action = "project_member.remove"
````
````new server/internal/modules/project/domain/actions.go
	ActionMemberRemove shared.Action = "project_member.remove"
	// ActionLeave is leaving a project: leaveProject.
	ActionLeave shared.Action = "project.leave"
````

````old server/internal/modules/project/domain/actions.go
		ActionMemberList, ActionMemberAdd, ActionJoin, ActionMemberUpdate, ActionMemberRemove}
````
````new server/internal/modules/project/domain/actions.go
		ActionMemberList, ActionMemberAdd, ActionJoin, ActionMemberUpdate, ActionMemberRemove, ActionLeave}
````

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"project_member.remove": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
````
````new server/internal/modules/access/domain/rules.go
	"project_member.remove": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	// One's own membership: every active member of the project; the only
	// admin's 409 is the use case's (M3 design 3.7 rule 1, 9.2).
	"project.leave": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"project_member.remove": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible,
		invisible, forbidden, invisible, invisible, invisible, forbidden, forbidden},
````
````new server/internal/modules/access/domain/rules_test.go
	"project_member.remove": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible,
		invisible, forbidden, invisible, invisible, invisible, forbidden, forbidden},
	// As project_member.list: every active member of the project.
	"project.leave": {allowed, allowed, allowed, allowed, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
````

`server/internal/modules/project/app/leave_project.go`（新文件，55 行）：

````file server/internal/modules/project/app/leave_project.go
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// LeaveProject ends the caller's own project membership: POST
// /api/v0/projects/{project_id}/leave (M3 design 3.7).
type LeaveProject struct {
	locks   Locks
	members MemberLeaver
	tx      shared.TxManager
	clock   Clock
}

// NewLeaveProject returns the use case.
func NewLeaveProject(locks Locks, members MemberLeaver, tx shared.TxManager, clock Clock) *LeaveProject {
	return &LeaveProject{locks: locks, members: members, tx: tx, clock: clock}
}

// Execute, in one transaction, in the order of M3 design 3.6: the project's
// locks (Locks.lockAndDecide: the workspace FOR SHARE, the project FOR NO
// KEY UPDATE) and the decision on project.leave, which only an active
// member of the project passes; then 3.7's rule 1: a project admin leaves
// only while the project has another active admin, also when he is its
// only member; then his membership ended, its row and its role kept, by
// himself at the time the clock gives under the locks. Every write of the
// project's memberships holds the project's row, so no other admin leaves
// or is removed between the read of the others and the ending.
func (u *LeaveProject) Execute(ctx context.Context, projectID uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	return u.tx.WithinTx(ctx, func(ctx context.Context) error {
		h, err := u.locks.lockAndDecide(ctx, actor, write{project: projectID, action: domain.ActionLeave})
		if err != nil {
			return err
		}
		if h.grant.ProjectRole == shared.RoleAdmin {
			other, err := u.members.HasOtherAdmin(ctx, projectID, actor.UserID)
			switch {
			case err != nil:
				return err
			case !other:
				return domain.ErrSoleAdmin
			}
		}
		return u.members.EndMember(ctx, projectID, actor.UserID, actor.UserID, u.clock.Now())
	})
}
````

`server/internal/modules/project/app/fakes_member_test.go`（修改，1 处）：

````old server/internal/modules/project/app/fakes_member_test.go
}

// frank is a project admin in newMemberWrites; the memberships it adds,
````
````new server/internal/modules/project/app/fakes_member_test.go
}

func (f *fakeStore) HasOtherAdmin(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	f.log.add(ctx, "HasOtherAdmin %s but %s", projectID, userID)
	if err := f.fail("HasOtherAdmin"); err != nil {
		return false, err
	}
	for user, m := range f.projects[projectID].members {
		if user != userID && m.Role == shared.RoleAdmin && m.Active {
			return true, nil
		}
	}
	return false, nil
}

// frank is a project admin in newMemberWrites; the memberships it adds,
````

`server/internal/modules/project/app/leave_project_test.go`（新文件，146 行）：

````file server/internal/modules/project/app/leave_project_test.go
package app_test

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newLeave is LeaveProject over newMemberWrites' fakes, its clock logged,
// each of web's members deciding as his roles in acme and web: alice its
// member, carol and ivy its guests, frank its second admin; gina, acme's
// admin, its member, as newMemberWrites has her; hank, acme's member, sees
// web and is not its member: the Authorizer's 403.
func newLeave() (*app.LeaveProject, *writeFixture) {
	f := newMemberWrites()
	delete(f.auth.errs, grantKey{alice, acme.ID})
	for user, g := range map[uuid.UUID]shared.Grant{alice: {WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleMember},
		carol: {WorkspaceRole: shared.RoleGuest, ProjectRole: shared.RoleGuest}, ivy: {WorkspaceRole: shared.RoleGuest, ProjectRole: shared.RoleGuest},
		frank: {WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleAdmin, ProjectAdmin: true}} {
		f.auth.grants[grantKey{user, acme.ID}] = g
	}
	f.auth.errs[grantKey{hank, acme.ID}] = shared.Forbidden()
	return app.NewLeaveProject(f.locks(), f.store, f.tx, clockAt{clockNow, f.log}), f
}

// left are the calls of user's leaving project: its locks and decision, the
// other admins read when admin is set, the clock, the ending at that time,
// by himself.
func left(user, project uuid.UUID, admin bool) []string {
	calls := lockedDecision(user, project, domain.ActionLeave)
	if admin {
		calls = append(calls, fmt.Sprintf("HasOtherAdmin %s but %s", project, user))
	}
	return append(calls, "Now", fmt.Sprintf("EndMember %s of %s by %s at %s", user, project, user, clockNow.Format(timeFormat)))
}

// LeaveProject, in one transaction and in the order of M3 design 3.6, locks
// the project, decides, reads the other admins when the caller is an admin,
// reads the clock, then ends the caller's membership, by himself at that
// time; it keeps its role, and no other membership changes. alice, a
// member, carol and ivy, guests, gina, a member who is acme's admin, leave
// web; bob and frank, its admins, each leave while the other stays.
func TestLeaveProject(t *testing.T) {
	for _, tt := range []struct {
		user  uuid.UUID
		admin bool
	}{{alice, false}, {carol, false}, {ivy, false}, {gina, false}, {bob, true}, {frank, true}} {
		uc, f := newLeave()
		before := maps.Clone(f.store.projects[webID].members)
		if err := uc.Execute(as(tt.user), webID); err != nil || !slices.Equal(f.log.calls, left(tt.user, webID, tt.admin)) {
			t.Errorf("Execute() by %s = %v, calls\n%q\nwant\n%q", tt.user, err, f.log.calls, left(tt.user, webID, tt.admin))
		}
		ended := before[tt.user]
		ended.Active = false
		before[tt.user] = ended
		if got := f.store.projects[webID].members; !maps.Equal(got, before) {
			t.Errorf("the memberships after %s left: %v; want %v, his ended alone", tt.user, got, before)
		}
	}
}

// Refusals, each in its place, and no membership changed: no caller, before
// the transaction; a project that is not there, a workspace or project
// deleted while its lock waited, a project moved meanwhile, and a caller
// who does not see the project, each project.not_found; one who sees it and
// is not its member, the Authorizer's 403. Then 3.7's rule 1: an admin
// whose project has no other active admin, the other's membership ended or
// the other a member now, and one who is the project's only member, is
// project.sole_admin.
func TestLeaveProjectRefuses(t *testing.T) {
	upTo := func(n int) []string { return lockedDecision(bob, webID, domain.ActionLeave)[:n] }
	soleAdmin := append(lockedDecision(bob, webID, domain.ActionLeave), fmt.Sprintf("HasOtherAdmin %s but %s", webID, bob))
	frankAs := func(role shared.Role, active bool) func(f *writeFixture) {
		return func(f *writeFixture) {
			f.store.projects[webID].members[frank] = app.Membership{ID: frankInWeb, Role: role, Active: active}
		}
	}
	for _, tt := range []struct {
		name    string
		ctx     context.Context
		project uuid.UUID
		set     func(f *writeFixture)
		want    error
		calls   []string
	}{
		{"no caller", context.Background(), webID, nil, shared.Unauthenticated(), nil},
		{"no project", as(bob), uuid.Nil(), nil, domain.ErrNotFound, noProject},
		{"acme deleted while its lock waited", as(bob), webID, func(f *writeFixture) { f.workspaces.gone = true }, domain.ErrNotFound, upTo(3)},
		{"web deleted while its lock waited", as(bob), webID, func(f *writeFixture) { f.store.deleted = true }, domain.ErrNotFound, upTo(4)},
		{"web moved to another workspace", as(bob), webID, func(f *writeFixture) { f.store.moved = uuid.NewV7() }, domain.ErrNotFound, upTo(4)},
		{"a caller who does not see web", as(erin), webID, nil, domain.ErrNotFound, lockedDecision(erin, webID, domain.ActionLeave)},
		{"a caller who sees web, not its member", as(hank), webID, nil, shared.Forbidden(), lockedDecision(hank, webID, domain.ActionLeave)},
		{"web's only admin, the other's membership ended", as(bob), webID, frankAs(shared.RoleAdmin, false), domain.ErrSoleAdmin, soleAdmin},
		{"web's only admin, the other a member now", as(bob), webID, frankAs(shared.RoleMember, true), domain.ErrSoleAdmin, soleAdmin},
		{"ops' only admin and member", as(bob), opsID, nil, domain.ErrSoleAdmin,
			append(lockedDecision(bob, opsID, domain.ActionLeave), fmt.Sprintf("HasOtherAdmin %s but %s", opsID, bob))},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newLeave()
			if tt.set != nil {
				tt.set(f)
			}
			web, ops := maps.Clone(f.store.projects[webID].members), maps.Clone(f.store.projects[opsID].members)
			outcome{tt.name, tt.want, tt.calls}.check(t, uc.Execute(tt.ctx, tt.project), f)
			if !maps.Equal(f.store.projects[webID].members, web) || !maps.Equal(f.store.projects[opsID].members, ops) {
				t.Errorf("the memberships after the refusal: %v, %v; want them as they were, %v, %v", f.store.projects[webID].members,
					f.store.projects[opsID].members, web, ops)
			}
		})
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after.
func TestLeaveProjectReturnsEachFailure(t *testing.T) {
	all := left(bob, webID, true)
	fail := func(method string) func(f *writeFixture) {
		return func(f *writeFixture) { f.store.errs = map[string]error{method: errDisk} }
	}
	for _, tt := range []struct {
		name  string
		fail  func(f *writeFixture)
		calls int // how many of the leaving's calls ran
	}{
		{"the project's workspace", fail("ProjectWorkspace"), 2},
		{"the workspace's lock", func(f *writeFixture) { f.workspaces.err = errDisk }, 3},
		{"the project's lock", fail("LockProject"), 4},
		{"the decision", func(f *writeFixture) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, 5},
		{"the other admins", fail("HasOtherAdmin"), 6},
		{"the ending", fail("EndMember"), 8},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 8},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newLeave()
			tt.fail(f)
			outcome{tt.name, errDisk, all[:tt.calls]}.check(t, uc.Execute(as(bob), webID), f)
		})
	}
}
````

`server/internal/modules/project/app/clock_test.go`（修改，1 处）：

````old server/internal/modules/project/app/clock_test.go
		}, removed(bob, alice)},
````
````new server/internal/modules/project/app/clock_test.go
		}, removed(bob, alice)},
		{"leaveProject", func() ([]string, error) {
			uc, f := newLeave()
			err := uc.Execute(as(bob), webID)
			return f.log.calls, err
		}, left(bob, webID, true)},
````

- [ ] **Step 3: handler、接线、文案**

`server/internal/modules/project/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/project/adapter/http/handler.go
}

// UseCases are the use cases behind the module's operations.
````
````new server/internal/modules/project/adapter/http/handler.go
}

// LeaveProjectUseCase is app.LeaveProject.
type LeaveProjectUseCase interface {
	Execute(ctx context.Context, projectID uuid.UUID) error
}

// UseCases are the use cases behind the module's operations.
````

````old server/internal/modules/project/adapter/http/handler.go
	RemoveMember      RemoveMemberUseCase
````
````new server/internal/modules/project/adapter/http/handler.go
	RemoveMember      RemoveMemberUseCase
	LeaveProject      LeaveProjectUseCase
````

`server/internal/modules/project/adapter/http/members.go`（修改，1 处）：

````old server/internal/modules/project/adapter/http/members.go
}

// members is list as the API shows it: data an array, never null.
````
````new server/internal/modules/project/adapter/http/members.go
}

// LeaveProject serves POST /api/v0/projects/{project_id}/leave.
func (h handler) LeaveProject(ctx context.Context, req gen.LeaveProjectRequestObject) (gen.LeaveProjectResponseObject, error) {
	if err := h.uc.LeaveProject.Execute(ctx, req.ProjectID); err != nil {
		return nil, err
	}
	return gen.LeaveProject204Response{}, nil
}

// members is list as the API shows it: data an array, never null.
````

`server/internal/modules/project/adapter/http/handler_test.go`（修改，3 处）：

````old server/internal/modules/project/adapter/http/handler_test.go
	removeMember *fakeDelete
````
````new server/internal/modules/project/adapter/http/handler_test.go
	removeMember *fakeDelete
	leave        *fakeDelete
````

````old server/internal/modules/project/adapter/http/handler_test.go
		f.removeMember = &fakeDelete{}
	}
````
````new server/internal/modules/project/adapter/http/handler_test.go
		f.removeMember = &fakeDelete{}
	}
	if f.leave == nil {
		f.leave = &fakeDelete{}
	}
````

````old server/internal/modules/project/adapter/http/handler_test.go
		JoinProject: f.join, UpdateMember: f.updateMember, RemoveMember: f.removeMember})
````
````new server/internal/modules/project/adapter/http/handler_test.go
		JoinProject: f.join, UpdateMember: f.updateMember, RemoveMember: f.removeMember, LeaveProject: f.leave})
````

`server/internal/modules/project/adapter/http/member_writes_test.go`（修改，1 处）：

````old server/internal/modules/project/adapter/http/member_writes_test.go
			t.Errorf("%s: DELETE = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

````
````new server/internal/modules/project/adapter/http/member_writes_test.go
			t.Errorf("%s: DELETE = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

// POST /projects/{project_id}/leave is the use case's 204, with no body, for
// the caller and the project of the path; each refusal the contract
// declares for it is its problem, a failure a 500.
func TestLeaveProject(t *testing.T) {
	leave := &fakeDelete{}
	h := newServer(t, fakes{leave: leave})
	path := "/api/v0/projects/" + webID.String() + "/leave"
	if res, body := do(t, h, request(http.MethodPost, path, "alice", "")); res.StatusCode != http.StatusNoContent || body != "" {
		t.Errorf("POST = %d %q, want 204 and no body", res.StatusCode, body)
	}
	if want := []string{"alice " + webID.String()}; !slices.Equal(leave.calls, want) {
		t.Errorf("calls = %q, want %q", leave.calls, want)
	}
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"no project", domain.ErrNotFound, http.StatusNotFound,
			`{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`},
		{"not its member", shared.Forbidden(), http.StatusForbidden, forbiddenJSON},
		{"its only admin", domain.ErrSoleAdmin, http.StatusConflict, `{"status":409,"code":"project.sole_admin","title":"Conflict",` +
			`"detail":"The project would be left without an admin: its only active admin cannot leave it, nor can his membership end while ` +
			`it has other active members. Give the project another admin first, or delete it."}`},
		{"a failure", errGone, http.StatusInternalServerError, internalErrorJSON},
	} {
		h := newServer(t, fakes{leave: &fakeDelete{err: tt.err}})
		if res, body := do(t, h, request(http.MethodPost, path, "alice", "")); res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: POST = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

````

`server/internal/modules/project/module.go`（修改，2 处）：

````old server/internal/modules/project/module.go
// members, changing a member's role and removing a member, each member's
// display settings, carries out the workspace module's cascades on the
// projects (ProjectCascade), and offers the access module its reads of a
// project (ProjectAccess) and the workspace module its count of an
````
````new server/internal/modules/project/module.go
// members, changing a member's role, removing a member and leaving, each
// member's display settings, carries out the workspace module's cascades
// on the projects (ProjectCascade), and offers the access module its reads
// of a project (ProjectAccess) and the workspace module its count of an
````

````old server/internal/modules/project/module.go
		RemoveMember:      app.NewRemoveProjectMember(locks, store, d.Tx, d.Clock),
````
````new server/internal/modules/project/module.go
		RemoveMember:      app.NewRemoveProjectMember(locks, store, d.Tx, d.Clock),
		LeaveProject:      app.NewLeaveProject(locks, store, d.Tx, d.Clock),
````

`server/internal/modules/workspace/adapter/http/members_test.go`（修改，4 处）：

````old server/internal/modules/workspace/adapter/http/members_test.go
func TestRemoveWorkspaceMemberRefusals(t *testing.T) {
	// The project module's ErrSoleAdmin, which this module does not import:
	// its kind, code and detail.
	soleAdmin := shared.NewError(shared.KindConflict, "project.sole_admin",
		"Ending the membership would leave a project that has other members without an admin; make another of its members an admin first.")
	tests := []struct {
		err    error
		status int
````
````new server/internal/modules/workspace/adapter/http/members_test.go
func TestRemoveWorkspaceMemberRefusals(t *testing.T) {
	// The project module's ErrSoleAdmin, which this module does not import:
	// its kind, code and detail.
	soleAdmin := shared.NewError(shared.KindConflict, "project.sole_admin",
		"The project would be left without an admin: its only active admin cannot leave it, nor can his membership end while it has "+
			"other active members. Give the project another admin first, or delete it.")
	tests := []struct {
		err    error
		status int
````

````old server/internal/modules/workspace/adapter/http/members_test.go
			`{"status":409,"code":"workspace.own_membership","title":"Conflict","detail":"You cannot change your own membership."}`},
		{fmt.Errorf("end the member's project memberships: %w", soleAdmin), http.StatusConflict,
			`{"status":409,"code":"project.sole_admin","title":"Conflict","detail":"Ending the membership would leave a project that has ` +
				`other members without an admin; make another of its members an admin first."}`},
		{errors.New("the database is gone"), http.StatusInternalServerError,
````
````new server/internal/modules/workspace/adapter/http/members_test.go
			`{"status":409,"code":"workspace.own_membership","title":"Conflict","detail":"You cannot change your own membership."}`},
		{fmt.Errorf("end the member's project memberships: %w", soleAdmin), http.StatusConflict,
			`{"status":409,"code":"project.sole_admin","title":"Conflict","detail":"The project would be left without an admin: its only ` +
				`active admin cannot leave it, nor can his membership end while it has other active members. Give the project another admin ` +
				`first, or delete it."}`},
		{errors.New("the database is gone"), http.StatusInternalServerError,
````

````old server/internal/modules/workspace/adapter/http/members_test.go
func TestLeaveWorkspaceRefusals(t *testing.T) {
	// The project module's ErrSoleAdmin, which this module does not import:
	// its kind, code and detail.
	soleAdmin := shared.NewError(shared.KindConflict, "project.sole_admin",
		"Ending the membership would leave a project that has other members without an admin; make another of its members an admin first.")
	tests := []struct {
		err    error
		status int
````
````new server/internal/modules/workspace/adapter/http/members_test.go
func TestLeaveWorkspaceRefusals(t *testing.T) {
	// The project module's ErrSoleAdmin, which this module does not import:
	// its kind, code and detail.
	soleAdmin := shared.NewError(shared.KindConflict, "project.sole_admin",
		"The project would be left without an admin: its only active admin cannot leave it, nor can his membership end while it has "+
			"other active members. Give the project another admin first, or delete it.")
	tests := []struct {
		err    error
		status int
````

````old server/internal/modules/workspace/adapter/http/members_test.go
			`"detail":"The workspace would be left without an admin; make another member an admin first."}`},
		{fmt.Errorf("end the member's project memberships: %w", soleAdmin), http.StatusConflict,
			`{"status":409,"code":"project.sole_admin","title":"Conflict","detail":"Ending the membership would leave a project that has ` +
				`other members without an admin; make another of its members an admin first."}`},
		{errors.New("the database is gone"), http.StatusInternalServerError,
````
````new server/internal/modules/workspace/adapter/http/members_test.go
			`"detail":"The workspace would be left without an admin; make another member an admin first."}`},
		{fmt.Errorf("end the member's project memberships: %w", soleAdmin), http.StatusConflict,
			`{"status":409,"code":"project.sole_admin","title":"Conflict","detail":"The project would be left without an admin: its only ` +
				`active admin cannot leave it, nor can his membership end while it has other active members. Give the project another admin ` +
				`first, or delete it."}`},
		{errors.New("the database is gone"), http.StatusInternalServerError,
````

`web/packages/i18n/src/locales/en/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/auth.json
      "project_sole_admin": "A project that has other members would be left without an admin. Make another of its members an admin first.",
````
````new web/packages/i18n/src/locales/en/auth.json
      "project_sole_admin": "The project would be left without an admin: its only admin cannot leave it, nor can his membership end while it has other members. Give the project another admin first, or delete it.",
````

`web/packages/i18n/src/locales/zh-CN/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/auth.json
      "project_sole_admin": "一个还有别的成员的项目会因此没有管理员。请先把那个项目的另一位成员设为管理员。",
````
````new web/packages/i18n/src/locales/zh-CN/auth.json
      "project_sole_admin": "项目会因此没有管理员：它唯一的管理员不能离开它；它还有别的成员时，他的成员关系也不能结束。请先给项目另一位管理员，或者删除这个项目。",
````

- [ ] **Step 4: 矩阵、最先锁工作区、组合出的离开**

`server/internal/bootstrap/permission_matrix_membership_test.go`（修改，4 处）：

````old server/internal/bootstrap/permission_matrix_membership_test.go
// membership (M3 design 9.2): changing a member's role, removing a member.
// Each names the membership by its id
// (/project-members/{project_member_id}), in its column's project;
// prepareMatrix's preconditions hold each membership a row names to the
// state the row says.
````
````new server/internal/bootstrap/permission_matrix_membership_test.go
// membership (M3 design 9.2): changing a member's role, removing a member,
// each naming the membership by its id
// (/project-members/{project_member_id}) in its column's project; and
// leaving, one's own. prepareMatrix's preconditions hold each membership a
// row names to the state the row says.
````

````old server/internal/bootstrap/permission_matrix_membership_test.go
	cellRoleTooHigh           = cell{http.StatusForbidden, "project.role_too_high"}
````
````new server/internal/bootstrap/permission_matrix_membership_test.go
	cellRoleTooHigh           = cell{http.StatusForbidden, "project.role_too_high"}
	cellProjectSoleAdmin      = cell{http.StatusConflict, "project.sole_admin"}
````

````old server/internal/bootstrap/permission_matrix_membership_test.go
			cells:   ofMembership(cellProjectOwnMembership, cellForbidden, cellForbidden, cellRoleTooHigh, cellForbidden, cellForbidden)},
````
````new server/internal/bootstrap/permission_matrix_membership_test.go
			cells:   ofMembership(cellProjectOwnMembership, cellForbidden, cellForbidden, cellRoleTooHigh, cellForbidden, cellForbidden)},
		// Every active member of the project, his own (M3 design 9.2): PA,
		// the only active admin of each project, is refused (3.7 rule 1);
		// PM+WA, a member who is the workspace's admin, leaves as a member.
		{op: "leaveProject", write: true, columns: projectColumns, request: toProject(http.MethodPost, "/leave", ""),
			cells: ofProject(cellProjectSoleAdmin, cellNoContent, cellNoContent, cellNoContent, cellForbidden, cellForbidden)},
````

````old server/internal/bootstrap/permission_matrix_membership_test.go
		s.t.Fatalf("%s's role in acme = %d, %v, %v; want its active guest", callerGuest, role, active, err)
	}
````
````new server/internal/bootstrap/permission_matrix_membership_test.go
		s.t.Fatalf("%s's role in acme = %d, %v, %v; want its active guest", callerGuest, role, active, err)
	}
	// PA is the only active admin of each project he is asked to leave,
	// beside its other members, so that his 409 is 3.7 rule 1's.
	for _, key := range []string{"acme/public", "acme/private"} {
		if other, err := s.store.HasOtherAdmin(ctx, sd.project(key), s.ids[callerProjectAdmin]); err != nil || other {
			s.t.Fatalf("another admin of %s than PA: %v, %v; want none", key, other, err)
		}
	}
````

`server/internal/bootstrap/project_write_locks_test.go`（修改，1 处）：

````old server/internal/bootstrap/project_write_locks_test.go
		member: [2]string{"dave", "erin"}},
````
````new server/internal/bootstrap/project_write_locks_test.go
		member: [2]string{"dave", "erin"}},
	// bob and carol, guests now, leave.
	{op: "leaveProject", method: http.MethodPost, path: "/api/v0/projects/%s/leave", want: http.StatusNoContent, by: [2]string{"bob", "carol"},
		member: [2]string{"bob", "carol"}},
````

`server/internal/bootstrap/project_leaving_test.go`（新文件，62 行）：

````file server/internal/bootstrap/project_leaving_test.go
package bootstrap

import (
	"net/http"
	"testing"
	"uuid"
)

// leave is name's leaving project: its status and body.
func (w memberWorld) leave(t *testing.T, name string, project uuid.UUID) (int, string) {
	t.Helper()
	return call(t, w.contract, http.MethodPost, w.base+"/api/v0/projects/"+project.String()+"/leave", w.tokens[name], "")
}

// Leaving a project on the wired app (M3 design 3.7 rule 1), one leaving
// after another on memberWorld, each a memberEnding by the member himself:
//   - bob, Lab's only admin, cannot leave it while carol is its member;
//     once she has left it, he cannot either: rule 1 holds when he is
//     alone too;
//   - carol and erin, Web's member and guest, leave it; bob and dave, two
//     of its three admins, leave it; alice, its only admin now, cannot,
//     gina its member;
//   - carol, an admin of Ops, leaves it; alice, its other admin, cannot:
//     carol's ended membership, an admin's, does not count;
//   - gina makes alice acme's guest, and so Web's and Ops' guest (3.3):
//     neither has an admin now. gina, Web's member and acme's admin, leaves
//     it as a member; bob, Ops' member, leaves it; alice, its guest now and
//     its only member, leaves it. The rule is the project role's, read
//     under the locks, not the workspace's.
func TestLeavingAProject(t *testing.T) {
	w := newMemberWorld(t)
	step := func(name, member string, project uuid.UUID, status int, code string) {
		memberEnding{step: name, name: member, by: member, project: project, status: status, code: code,
			send: func() (int, string) { return w.leave(t, member, project) }}.check(t, w)
	}
	step("bob leaves Lab, its only admin; carol its member", "bob", w.lab, http.StatusConflict, "project.sole_admin")
	step("carol leaves Lab, its member", "carol", w.lab, http.StatusNoContent, "")
	step("bob leaves Lab, its only admin and member", "bob", w.lab, http.StatusConflict, "project.sole_admin")
	step("carol leaves Web, its member", "carol", w.web, http.StatusNoContent, "")
	step("erin leaves Web, its guest", "erin", w.web, http.StatusNoContent, "")
	step("bob leaves Web, an admin; alice and dave its others", "bob", w.web, http.StatusNoContent, "")
	step("dave leaves Web, an admin; alice its other", "dave", w.web, http.StatusNoContent, "")
	step("alice leaves Web, its only admin; gina its member", "alice", w.web, http.StatusConflict, "project.sole_admin")
	step("carol leaves Ops, an admin; alice its other", "carol", w.ops, http.StatusNoContent, "")
	step("alice leaves Ops, its only active admin; bob its member", "alice", w.ops, http.StatusConflict, "project.sole_admin")
	var alices uuid.UUID
	if err := w.pool.QueryRow(soon(t), `SELECT m.id FROM workspace_members m JOIN workspaces s ON s.id = m.workspace_id
		WHERE s.slug = 'acme' AND m.member_id = $1`, w.ids["alice"]).Scan(&alices); err != nil {
		t.Fatal(err)
	}
	if status, body := call(t, w.contract, http.MethodPatch, w.base+"/api/v0/workspace-members/"+alices.String(), w.tokens["gina"],
		`{"role":5}`); status != http.StatusOK {
		t.Fatalf("gina's making alice acme's guest = %d %s", status, body)
	}
	step("gina leaves Web, its member and acme's admin; alice its guest now", "gina", w.web, http.StatusNoContent, "")
	step("bob leaves Ops, its member; alice its guest now", "bob", w.ops, http.StatusNoContent, "")
	step("alice leaves Ops, its guest now and only member", "alice", w.ops, http.StatusNoContent, "")
	if got, want := w.standing(t), "acme: alice 5, bob 15, carol 15, dave 15, erin 5, gina 20; beta: bob 20, carol 15; Lab: bob 20; "+
		"Web: alice 5"; got != want {
		t.Errorf("the world after the leavings: %s; want %s", got, want)
	}
}
````

- [ ] **Step 5: 测试、lint、前端检查**

Run: `go -C server test -count=1 ./internal/modules/project/... ./internal/modules/access/... ./internal/modules/workspace/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix$|TestEachMatrixTableIsOfOneLevel|TestEachWriteOnAProjectSharesItsWorkspaceFirst|TestLeavingAProject$|TestAPIRoutesAreTheContractsOperations' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

- [ ] **Step 6: 提交**

```bash
git add api/modules/project.yaml api/openapi.yaml server/internal/bootstrap/permission_matrix_membership_test.go server/internal/bootstrap/project_leaving_test.go server/internal/bootstrap/project_write_locks_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/project/adapter/http/handler.go server/internal/modules/project/adapter/http/handler_test.go server/internal/modules/project/adapter/http/member_writes_test.go server/internal/modules/project/adapter/http/members.go server/internal/modules/project/app/clock_test.go server/internal/modules/project/app/fakes_member_test.go server/internal/modules/project/app/leave_project.go server/internal/modules/project/app/leave_project_test.go server/internal/modules/project/domain/actions.go server/internal/modules/project/domain/errors.go server/internal/modules/project/module.go server/internal/modules/workspace/adapter/http/members_test.go web/packages/i18n/src/locales/en/auth.json web/packages/i18n/src/locales/zh-CN/auth.json api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P5b): leaveProject, and project.sole_admin worded for both rules

POST /api/v0/projects/{project_id}/leave, for each active member of
the project: its only active admin is refused project.sole_admin, also
when he is its only member (rule 1), decided on the project role read
under the locks. The code's detail now speaks of both rules and of a
remedy each who gets it has: give the project another admin first, or
delete it.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A，清扫 2、4、5、18、21、24、30）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| 唯一的管理员照样离开（不问另一位） | `TestLeaveProjectRefuses`；`TestLeavingAProject`、`TestPermissionMatrix`；P5（Task 11 起） | 单元；组合；端到端 |
| 每个离开的人都要另一位管理员 | `TestLeaveProject`；`TestLeavingAProject`（矩阵和故事看不到：离开的人的项目都有别的管理员） | 单元；组合 |
| 规则 1 看 `ProjectAdmin`（PM+WA 照管理员） | `TestLeaveProject`；`TestLeavingAProject`（gina 一步） | 单元；组合 |
| 规则不给项目访客 | `TestEveryRuleDecidesItsCells`；`TestPermissionMatrix`、`TestLeavingAProject`、`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（故事看不到：故事里离开的是成员） | 单元；组合 |
| 问另一位管理员的失败被忽略；结束失败之后重试 | `TestLeaveProjectReturnsEachFailure` | 单元 |
| 锁或判定的拒绝被吞掉，照样结束 | `TestLeaveProjectRefuses`；`TestPermissionMatrix`、`TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile`、`TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize`（Task 8、9 起） | 单元；组合 |
| 唯一管理员的 409 包在 403 后面（`errors.Join`） | `TestLeaveProjectRefuses`（第一个 `*shared.Error`，清扫 21）；`TestLeavingAProject` | 单元；组合 |
| 离开的时刻在锁之前读 | `TestEachWriteReadsTheClockUnderItsLock`；`TestEachLockOfAWriteOnAProjectMembershipIsItsStrength`（Task 8 起） | 单元；组合 |
| `project.New` 的离开不结束任何行；不开事务 | `TestLeavingAProject`（`memberEnding`，不结束）；`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（不开事务）；`TestEachLockOfAWriteOnAProjectMembershipIsItsStrength`、`TestTheWritesOnAProjectStampTheirRequest`（Task 8 起） | 组合 |
| `POST /leave` 交给移出的用例 | `TestLeaveProject`（HTTP）；`TestLeavingAProject`、`TestPermissionMatrix`；P5 | 单元；组合；端到端 |
| 去掉 `projectWrites` 的离开一行 | `TestEachWriteOnAProjectSharesItsWorkspaceFirst`（完整性核对的第一种形状：路径里的 `{project_id}`） | 组合 |
| 契约少声明 `project.sole_admin` | `TestLeaveProject`（HTTP，`CheckResponse`）；`TestLeavingAProject`、`TestPermissionMatrix` | 单元；组合 |

**Done when:** 离开的一行矩阵（12 格）通过；规则 1 在组合出的 app 上有正反例（只有他一人也拒绝，已结束的管理员不算，规则看项目角色）；`project.sole_admin` 的三处文字一致。

---

### Task 8: 三个写的竞争、锁的强度、事务的连接、时刻和写者（组合）

**Files:**
- Create: `server/internal/bootstrap/project_membership_races_test.go`、`server/internal/bootstrap/project_membership_role_race_test.go`
- Modify: `server/internal/bootstrap/project_connection_test.go`、`server/internal/bootstrap/project_writes_test.go`

**Interfaces:** 没有新的产品代码。组合一层的测试（spec 2.11，brief 清扫 8、9、12、13、14、29）：
- `membershipWrite{op, by, member, method, path, body, status, notFound, target}` 和 `membershipWrites`：bob（Web 的管理员、acme 的成员）把 gina（Web 的成员、acme 的管理员）改为访客、移出她；dave（Web 的管理员）离开；alice 是 Web 的另一位管理员，别的什么都不拒绝。`sent`（对照契约发出，答案经 `sendInBackground`）；`newPrivateWorld`（alice 把 Web 改为私有：不是 Web 成员的 acme 成员看不到它）。
- `project_connection_test.go`：连接测试加 alice 把 carol 改为 Web 的管理员、移出 bob、carol 离开。
- `project_writes_test.go`：`TestTheWritesOnAProjectStampTheirRequest` 的行多一列 `of`（路径按 id 指的成员关系的账户），加改角色（bob 改为管理员）、移出（carol）、离开（alice）三行；每行的目标之前由 bob 写（清扫 14）。

**Tests:**
- `TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile`：另一个事务持 Web 的 `FOR NO KEY UPDATE` 并改一行，或持 acme 的 `FOR NO KEY UPDATE` 并删除 acme；写已过认证、不加锁读过它指的成员关系，在 acme 或 Web 的行上等（`WaitForLockWaitOn`）；另一个提交之后，每个情形都是 404、从不是泄露存在的 403，另一个改的行是它留下的样子，其余每行不变：

  | 写 | 等待期间 | 回答 |
  |---|---|---|
  | 改角色、移出 | gina 的成员关系结束、删除、移到 Ops | 404 `project.member_not_found` |
  | 改角色、移出、离开 | 调用者自己的成员关系结束（Web 私有：他看不到它） | 404（离开：`project.not_found`） |
  | 改角色、移出、离开 | Web 删除；acme 删除 | 同上 |

- `TestAWriteOnAProjectMembershipDecidesOnWhatItReadUnderItsLocks`：另一个事务持 Web 的 `FOR NO KEY UPDATE`，把写指的成员关系改为管理员或删除；写在 Web 的行上等，另一个提交之后：bob（Web 的管理员，不是工作区管理员）改 carol 的角色、gina（Web 的成员、acme 的管理员）移出 carol，各 403 `project.role_too_high`（相对规则用锁下重读到的角色）；carol（Web 的成员，无权改、移出）改、移出 gina 已删除的成员关系，各 404 `project.member_not_found`（重读在判定之前，L-6）；那一行是另一个留下的样子，其余每行不变。
- `TestEachLockOfAWriteOnAProjectMembershipIsItsStrength`（三个写各一个子测试）：别的事务持 acme 的 `FOR NO KEY UPDATE`，Web 的行和写改的那一行成员关系的 `FOR SHARE`，逐个放开；每一步用 `lockOn` 读出各行最强的锁：

  | 写等的表 | 那时持有 |
  |---|---|
  | `workspaces` | 什么都没有（成员关系的第一次读不加锁） |
  | `projects` | acme `FOR SHARE`；改角色时 gina 在 acme 的成员关系 `FOR SHARE`（约定三），不更强；移出、离开时不锁它 |
  | `project_members` | 再加 Web `FOR NO KEY UPDATE`；Ops（alice 的另一个项目，gina、dave 都不是成员）没有锁 |

  然后它照单独时回答，写下的时刻不早于 Web 被放开（时钟在锁之后读，3.3）。
- `TestTheWritesOnAProjectRunOnTheirTransactionsConnection`：三个写在只有一个连接的池上各在期限内回答；任何一条语句经连接池就等一个不来的连接。
- `TestTheWritesOnAProjectStampTheirRequest`：三行，各写在请求之内的一个时刻、由 alice。

- [ ] **Step 1: 竞争和锁**

`server/internal/bootstrap/project_membership_races_test.go`（新文件，244 行）：

````file server/internal/bootstrap/project_membership_races_test.go
package bootstrap

import (
	"context"
	"maps"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// A write on a project membership that waits for a lock, against what
// another transaction changes or holds meanwhile, on the wired app (M3
// design 3.6 convention 2, the lock table): what it finds once it has its
// locks, and each lock it takes, at its strength.

// membershipWrite is one of the writes on a project membership as the
// races send it on memberWorld: bob, Web's admin and acme's member,
// changes gina's role in Web to a guest's, or removes her; dave, Web's
// admin, leaves it. gina is Web's member and acme's admin; alice, Web's
// other admin, stays; nothing else refuses each write.
type membershipWrite struct {
	op, by, member     string // the caller; whose membership of Web the write changes
	method, path, body string // path: %s the membership's id for a path of one, else Web's
	status             int    // its answer, alone
	notFound           string // the code of its 404
	target             bool   // it changes the member's role: it shares his membership of acme (convention 3)
}

var membershipWrites = []membershipWrite{
	{"updateProjectMember", "bob", "gina", http.MethodPatch, "/api/v0/project-members/%s", `{"role":5}`, http.StatusOK, "project.member_not_found",
		true},
	{"removeProjectMember", "bob", "gina", http.MethodDelete, "/api/v0/project-members/%s", "", http.StatusNoContent, "project.member_not_found",
		false},
	{"leaveProject", "dave", "dave", http.MethodPost, "/api/v0/projects/%s/leave", "", http.StatusNoContent, "project.not_found", false},
}

// sent sends m on w's app, checked against the contract, and hands over its
// answer.
func (m membershipWrite) sent(t *testing.T, w memberWorld) (*http.Request, <-chan answer) {
	t.Helper()
	named := w.web
	if strings.HasPrefix(m.path, "/api/v0/project-members/") {
		named = w.membership(t, w.web, m.member)
	}
	var body []byte
	if m.body != "" {
		body = []byte(m.body)
	}
	req := newRequest(t, m.method, w.base+strings.Replace(m.path, "%s", named.String(), 1), w.tokens[m.by], body)
	w.contract.CheckRequest(t, req)
	return req, sendInBackground(req)
}

// newPrivateWorld is memberWorld with Web private, made so by alice: a
// member of acme who is no member of Web does not see it.
func newPrivateWorld(t *testing.T) memberWorld {
	t.Helper()
	w := newMemberWorld(t)
	if status, body := call(t, w.contract, http.MethodPatch, w.base+"/api/v0/projects/"+w.web.String(), w.tokens["alice"], `{"network":0}`); status !=
		http.StatusOK {
		t.Fatalf("alice's making Web private = %d %s", status, body)
	}
	return w
}

// A write on a project membership that waits for a lock reads, once it has
// it, what another transaction committed meanwhile, and answers its 404, as
// for a membership or a project never there, never the 403 that would tell
// its caller that it was there; and it changes no row (M3 design 3.6
// convention 2, 8.2). The other transaction holds Web FOR NO KEY UPDATE, as
// a write on it does, and ends, deletes or moves to Ops the membership the
// write changes, or ends the caller's own membership, or deletes Web; or it
// holds acme FOR NO KEY UPDATE, as a cascade does, and deletes acme. The
// write has passed authentication, read the membership it names unlocked,
// and waits for that row. Once the other commits, the write is 404; the row
// the other changed is as it left it, and every other row as it was. Web is
// private: bob or dave, his membership ended, does not see it.
func TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile(t *testing.T) {
	type change struct {
		name, holds, table, sql string // holds: the table of the row the other transaction locks first, acme's or Web's
		member                  bool   // the change is of the write's member's membership, not its caller's own
		row                     func(w memberWorld, t *testing.T, m membershipWrite) uuid.UUID
	}
	membershipOf := func(member bool) func(w memberWorld, t *testing.T, m membershipWrite) uuid.UUID {
		return func(w memberWorld, t *testing.T, m membershipWrite) uuid.UUID {
			if member {
				return w.membership(t, w.web, m.member)
			}
			return w.membership(t, w.web, m.by)
		}
	}
	web := func(w memberWorld, _ *testing.T, _ membershipWrite) uuid.UUID { return w.web }
	acme := func(w memberWorld, t *testing.T, _ membershipWrite) uuid.UUID {
		var id uuid.UUID
		if err := w.pool.QueryRow(soon(t), "SELECT id FROM workspaces WHERE slug = 'acme'").Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	changes := []change{
		{"the member's membership ended", "projects", "project_members", "UPDATE project_members SET is_active = false WHERE id = $1", true,
			membershipOf(true)},
		{"the member's membership deleted", "projects", "project_members", "UPDATE project_members SET deleted_at = now() WHERE id = $1", true,
			membershipOf(true)},
		{"the member's membership moved to Ops", "projects", "project_members",
			"UPDATE project_members SET project_id = (SELECT id FROM projects WHERE name = 'Ops') WHERE id = $1", true, membershipOf(true)},
		{"the caller's membership ended", "projects", "project_members", "UPDATE project_members SET is_active = false WHERE id = $1", false,
			membershipOf(false)},
		{"Web deleted", "projects", "projects", "UPDATE projects SET deleted_at = now(), updated_at = now() WHERE id = $1", false, web},
		{"acme deleted", "workspaces", "workspaces", "UPDATE workspaces SET deleted_at = now(), updated_at = now() WHERE id = $1", false, acme},
	}
	for _, m := range membershipWrites {
		for _, c := range changes {
			if c.member && m.member == m.by {
				continue // leaving: the member's membership is the caller's
			}
			t.Run(m.op+", "+c.name, func(t *testing.T) {
				w := newPrivateWorld(t)
				id := c.row(w, t, m)
				others := rowsBut(t, w.pool, []uuid.UUID{id})
				first := "SELECT 1 FROM projects WHERE id = $1 FOR NO KEY UPDATE"
				firstID := w.web
				if c.holds == "workspaces" {
					first, firstID = "SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", acme(w, t, m)
				}
				other := holding(t, w.pool, first, firstID)
				if tag, err := other.Exec(soon(t), c.sql, id); err != nil || tag.RowsAffected() != 1 {
					t.Fatalf("%s: %v, %v; want one row changed", c.sql, tag, err)
				}
				changed := rowJSON(t, other, c.table, id)
				req, answered := m.sent(t, w)
				pgtest.WaitForLockWaitOn(t, w.pool, c.holds, 5*time.Second)
				if err := other.Commit(soon(t)); err != nil {
					t.Fatal(err)
				}

				a := receiveWithin(t, answered, 10*time.Second, "the answer to "+m.op)
				if a.err != nil {
					t.Fatal(a.err)
				}
				w.contract.CheckResponse(t, req, a.res)
				if a.res.StatusCode != http.StatusNotFound || problemCode(t, a.body) != m.notFound {
					t.Errorf("%s = %d %s, want 404 %s", m.op, a.res.StatusCode, a.body, m.notFound)
				}
				if after := rowJSON(t, w.pool, c.table, id); !reflect.DeepEqual(after, changed) {
					t.Errorf("%s %s after the write:\n%v\nwant it as the other transaction left it:\n%v", c.table, id, after, changed)
				}
				if after := rowsBut(t, w.pool, []uuid.UUID{id}); !maps.Equal(after, others) {
					t.Errorf("every other row after the write:\n%v\nwant them as they were:\n%v", after, others)
				}
			})
		}
	}
}

// Each lock of a write on a project membership is taken in the lock
// table's order, at its strength, as bootstrap wires it (M3 design 3.6's
// lock table, conventions 2 and 3). Other transactions hold acme's row FOR
// NO KEY UPDATE and, FOR SHARE, Web's row and the membership of Web the
// write changes, which the write waits for in turn; they let go one at a
// time, and lockOn reads each row's strongest lock then. The write waits
// for acme's row, holding nothing; then for Web's, holding acme's FOR
// SHARE and, for a change of a role, the member's membership of acme FOR
// SHARE (convention 3), no stronger; then for the membership of Web, holding
// Web FOR NO KEY UPDATE too; never the member's membership of acme for a
// removal or a leaving, nor Ops, alice's other project of acme, of which
// gina and dave are no members. Then it answers as alone, and the moment it
// wrote is no earlier than Web's release: it read the clock under Web's
// lock (3.3).
func TestEachLockOfAWriteOnAProjectMembershipIsItsStrength(t *testing.T) {
	for _, m := range membershipWrites {
		t.Run(m.op, func(t *testing.T) {
			w := newMemberWorld(t)
			var acme, inAcme uuid.UUID
			if err := w.pool.QueryRow(soon(t), `SELECT s.id, m.id FROM workspaces s JOIN workspace_members m ON m.workspace_id = s.id
				WHERE s.slug = 'acme' AND m.member_id = $1`, w.ids[m.member]).Scan(&acme, &inAcme); err != nil {
				t.Fatal(err)
			}
			inWeb := w.membership(t, w.web, m.member)
			locks := func() string {
				t.Helper()
				return "acme " + lockOn(t, w.pool, "workspaces WHERE id = $1", acme) +
					", in acme " + lockOn(t, w.pool, "workspace_members WHERE id = $1", inAcme) +
					", Web " + lockOn(t, w.pool, "projects WHERE id = $1", w.web) +
					", in Web " + lockOn(t, w.pool, "project_members WHERE id = $1", inWeb) +
					", Ops " + lockOn(t, w.pool, "projects WHERE id = $1", w.ops)
			}
			inAcmeShared := "no lock"
			if m.target {
				inAcmeShared = "FOR SHARE"
			}
			holdsInWeb := holding(t, w.pool, "SELECT 1 FROM project_members WHERE id = $1 FOR SHARE", inWeb)
			holdsWeb := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR SHARE", w.web)
			holdsAcme := holding(t, w.pool, "SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", acme)
			req, answered := m.sent(t, w)
			var released time.Time
			for _, step := range []struct {
				release pgx.Tx
				waitsOn string
				want    string
			}{
				{nil, "workspaces", "acme FOR NO KEY UPDATE, in acme no lock, Web FOR SHARE, in Web FOR SHARE, Ops no lock"},
				{holdsAcme, "projects", "acme FOR SHARE, in acme " + inAcmeShared + ", Web FOR SHARE, in Web FOR SHARE, Ops no lock"},
				{holdsWeb, "project_members", "acme FOR SHARE, in acme " + inAcmeShared + ", Web FOR NO KEY UPDATE, in Web FOR SHARE, Ops no lock"},
			} {
				if step.release != nil {
					if step.release == holdsWeb {
						released = time.Now()
					}
					if err := step.release.Rollback(context.Background()); err != nil {
						t.Fatal(err)
					}
				}
				pgtest.WaitForLockWaitOn(t, w.pool, step.waitsOn, 5*time.Second)
				if got := locks(); got != step.want {
					t.Errorf("%s waiting on %s: %s; want %s", m.op, step.waitsOn, got, step.want)
				}
			}
			if err := holdsInWeb.Rollback(context.Background()); err != nil {
				t.Fatal(err)
			}

			a := receiveWithin(t, answered, 10*time.Second, "the answer to "+m.op)
			if a.err != nil {
				t.Fatal(a.err)
			}
			w.contract.CheckResponse(t, req, a.res)
			if a.res.StatusCode != m.status {
				t.Errorf("%s = %d %s, want %d", m.op, a.res.StatusCode, a.body, m.status)
			}
			moment, _ := rowJSON(t, w.pool, "project_members", inWeb)["updated_at"].(string)
			if at, err := time.Parse(time.RFC3339Nano, moment); err != nil || at.Before(released.Truncate(time.Microsecond)) {
				t.Errorf("%s's moment %q (%v); want one no earlier than Web's release, %v", m.op, moment, err, released)
			}
		})
	}
}
````

`server/internal/bootstrap/project_membership_role_race_test.go`（新文件，81 行）：

````file server/internal/bootstrap/project_membership_role_race_test.go
package bootstrap

import (
	"maps"
	"net/http"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// A write on a project membership decides on the membership as it reads it
// under its locks (M3 design 3.5, 3.6 convention 2), on the wired app:
// another transaction holds Web FOR NO KEY UPDATE, as a write on it does,
// and changes the membership the write names, which the write has read
// before its locks and waits for Web's row. Once the other commits:
//   - carol, Web's member, made its admin: bob, Web's admin and no
//     workspace admin, may not change an admin's role, and gina, Web's
//     member and acme's admin, may not remove one above her own role: each
//     403 project.role_too_high, the relative rule reading the role under
//     the locks, not the one read before them;
//   - gina's membership deleted: carol, Web's member, who may change no
//     role and remove nobody, gets the membership's 404, not her 403: the
//     membership is read again before the decision.
//
// The membership is as the other left it, and every other row as it was.
func TestAWriteOnAProjectMembershipDecidesOnWhatItReadUnderItsLocks(t *testing.T) {
	const madeAdmin, deleted = "UPDATE project_members SET role = 20 WHERE id = $1", "UPDATE project_members SET deleted_at = now() WHERE id = $1"
	for _, tt := range []struct {
		name, by, member, method, body, sql string
		status                              int
		code                                string
	}{
		{"bob's change of carol's role, carol made an admin", "bob", "carol", http.MethodPatch, `{"role":5}`, madeAdmin, http.StatusForbidden,
			"project.role_too_high"},
		{"gina's removal of carol, carol made an admin", "gina", "carol", http.MethodDelete, "", madeAdmin, http.StatusForbidden,
			"project.role_too_high"},
		{"carol's change of gina's role, gina's membership deleted", "carol", "gina", http.MethodPatch, `{"role":5}`, deleted,
			http.StatusNotFound, "project.member_not_found"},
		{"carol's removal of gina, gina's membership deleted", "carol", "gina", http.MethodDelete, "", deleted, http.StatusNotFound,
			"project.member_not_found"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newMemberWorld(t)
			id := w.membership(t, w.web, tt.member)
			others := rowsBut(t, w.pool, []uuid.UUID{id})
			other := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR NO KEY UPDATE", w.web)
			if tag, err := other.Exec(soon(t), tt.sql, id); err != nil || tag.RowsAffected() != 1 {
				t.Fatalf("%s: %v, %v; want one row changed", tt.sql, tag, err)
			}
			changed := rowJSON(t, other, "project_members", id)
			var body []byte
			if tt.body != "" {
				body = []byte(tt.body)
			}
			req := newRequest(t, tt.method, w.base+"/api/v0/project-members/"+id.String(), w.tokens[tt.by], body)
			w.contract.CheckRequest(t, req)
			answered := sendInBackground(req)
			pgtest.WaitForLockWaitOn(t, w.pool, "projects", 5*time.Second)
			if err := other.Commit(soon(t)); err != nil {
				t.Fatal(err)
			}
			a := receiveWithin(t, answered, 10*time.Second, "the answer to "+tt.name)
			if a.err != nil {
				t.Fatal(a.err)
			}
			w.contract.CheckResponse(t, req, a.res)
			if a.res.StatusCode != tt.status || problemCode(t, a.body) != tt.code {
				t.Errorf("%s = %d %s, want %d %s", tt.name, a.res.StatusCode, a.body, tt.status, tt.code)
			}
			if after := rowJSON(t, w.pool, "project_members", id); !reflect.DeepEqual(after, changed) {
				t.Errorf("the membership after the write:\n%v\nwant it as the other left it:\n%v", after, changed)
			}
			if after := rowsBut(t, w.pool, []uuid.UUID{id}); !maps.Equal(after, others) {
				t.Errorf("every other row after the write:\n%v\nwant them as they were:\n%v", after, others)
			}
		})
	}
}
````

- [ ] **Step 2: 连接和时刻**

`server/internal/bootstrap/project_connection_test.go`（修改，2 处）：

````old server/internal/bootstrap/project_connection_test.go
// reads, the facts its decision reads (ProjectFacts and the workspace
// roles), its writes and its answer. The project module and the Authorizer
// are wired as bootstrap wires them, on a pool of one connection: a
// statement sent through the pool rather than the transaction would wait
// for a second connection that never comes, and its request fail at the
// request's deadline. alice, acme's admin, creates Ops, changes Web,
// archives and unarchives it, changes her display settings in it and adds
// bob, whose ended membership she restores; carol joins it anew; alice
// deletes both projects.
````
````new server/internal/bootstrap/project_connection_test.go
// reads, the membership it names among them, the facts its decision reads
// (ProjectFacts and the workspace roles), the other admins a leaving reads,
// its writes and its answer. The project module and the Authorizer are
// wired as bootstrap wires them, on a pool of one connection: a statement
// sent through the pool rather than the transaction would wait for a second
// connection that never comes, and its request fail at the request's
// deadline. alice, acme's admin, creates Ops, changes Web, archives and
// unarchives it, changes her display settings in it and adds bob, whose
// ended membership she restores; carol joins it anew; alice makes carol an
// admin of Web and removes bob; carol, an admin, leaves it; alice deletes
// both projects.
````

````old server/internal/bootstrap/project_connection_test.go
	send(http.MethodPost, web+"/join", carol, "", http.StatusOK)
````
````new server/internal/bootstrap/project_connection_test.go
	send(http.MethodPost, web+"/join", carol, "", http.StatusOK)
	send(http.MethodPatch, "/api/v0/project-members/"+projectMemberships(t, r.pool, carol, r.web)[0].String(), r.alice, `{"role":20}`,
		http.StatusOK)
	send(http.MethodDelete, "/api/v0/project-members/"+projectMemberships(t, r.pool, r.bob, r.web)[0].String(), r.alice, "", http.StatusNoContent)
	send(http.MethodPost, web+"/leave", carol, "", http.StatusNoContent)
````

`server/internal/bootstrap/project_writes_test.go`（修改，12 处）：

````old server/internal/bootstrap/project_writes_test.go
// writes. Bob and carol, whom she adds, are acme's members. Each row the
// write writes again is first made bob's, as last written by him, and
// checked so: a write that kept its row's writer would pass for alice's
// otherwise, she having made it.
````
````new server/internal/bootstrap/project_writes_test.go
// writes. Bob and carol, whom she adds, are acme's members; she makes bob
// an admin of Web, removes carol, and leaves Web. Each row the write writes
// again is first made bob's, as last written by him, and checked so: a
// write that kept its row's writer would pass for alice's otherwise, she
// having made it.
````

````old server/internal/bootstrap/project_writes_test.go
		name, method, path, body string
````
````new server/internal/bootstrap/project_writes_test.go
		name, method, path, body string
		of                       uuid.UUID // the account whose membership of Web the path names by its id, as %s
````

````old server/internal/bootstrap/project_writes_test.go
		{"updateProject", http.MethodPatch, "/api/v0/projects/" + web.String(), `{"name":"Site"}`, http.StatusOK, bobs("projects", "id = $1"),
````
````new server/internal/bootstrap/project_writes_test.go
		{"updateProject", http.MethodPatch, "/api/v0/projects/" + web.String(), `{"name":"Site"}`, uuid.UUID{}, http.StatusOK,
			bobs("projects", "id = $1"),
````

````old server/internal/bootstrap/project_writes_test.go
		{"archiveProject", http.MethodPost, "/api/v0/projects/" + web.String() + "/archive", "", http.StatusOK, bobs("projects", "id = $1"),
````
````new server/internal/bootstrap/project_writes_test.go
		{"archiveProject", http.MethodPost, "/api/v0/projects/" + web.String() + "/archive", "", uuid.UUID{}, http.StatusOK,
			bobs("projects", "id = $1"),
````

````old server/internal/bootstrap/project_writes_test.go
		{"archiveProject again", http.MethodPost, "/api/v0/projects/" + web.String() + "/archive", "", http.StatusOK, bobs("projects", "id = $1"),
````
````new server/internal/bootstrap/project_writes_test.go
		{"archiveProject again", http.MethodPost, "/api/v0/projects/" + web.String() + "/archive", "", uuid.UUID{}, http.StatusOK,
			bobs("projects", "id = $1"),
````

````old server/internal/bootstrap/project_writes_test.go
		{"unarchiveProject", http.MethodPost, "/api/v0/projects/" + web.String() + "/unarchive", "", http.StatusOK, bobs("projects", "id = $1"),
````
````new server/internal/bootstrap/project_writes_test.go
		{"unarchiveProject", http.MethodPost, "/api/v0/projects/" + web.String() + "/unarchive", "", uuid.UUID{}, http.StatusOK,
			bobs("projects", "id = $1"),
````

````old server/internal/bootstrap/project_writes_test.go
		{"updateProjectPreferences", http.MethodPatch, "/api/v0/me/projects/" + web.String() + "/preferences", `{"sort_order":5}`, http.StatusOK,
````
````new server/internal/bootstrap/project_writes_test.go
		{"updateProjectPreferences", http.MethodPatch, "/api/v0/me/projects/" + web.String() + "/preferences", `{"sort_order":5}`, uuid.UUID{}, http.StatusOK,
````

````old server/internal/bootstrap/project_writes_test.go
			`{"members":[{"member_id":"` + bobID.String() + `","role":15}]}`, http.StatusCreated, "",
````
````new server/internal/bootstrap/project_writes_test.go
			`{"members":[{"member_id":"` + bobID.String() + `","role":15}]}`, uuid.UUID{}, http.StatusCreated, "",
````

````old server/internal/bootstrap/project_writes_test.go
			`{"members":[{"member_id":"` + carol + `","role":15}]}`, http.StatusCreated,
````
````new server/internal/bootstrap/project_writes_test.go
			`{"members":[{"member_id":"` + carol + `","role":15}]}`, uuid.UUID{}, http.StatusCreated,
````

````old server/internal/bootstrap/project_writes_test.go
				"FROM project_user_properties WHERE project_id = $1 AND user_id = '" + carol + "'", 1, 2},
````
````new server/internal/bootstrap/project_writes_test.go
				"FROM project_user_properties WHERE project_id = $1 AND user_id = '" + carol + "'", 1, 2},
		// Bob's membership, a member's, made an admin's.
		{"updateProjectMember", http.MethodPatch, "/api/v0/project-members/%s", `{"role":20}`, bobID, http.StatusOK,
			bobs("project_members", "project_id = $1 AND member_id = '"+bobID.String()+"'"),
			"SELECT updated_at, updated_by_id = $2 AND role = 20 FROM project_members WHERE project_id = $1 AND member_id = '" + bobID.String() + "'",
			1, 1},
		// Carol's membership, ended.
		{"removeProjectMember", http.MethodDelete, "/api/v0/project-members/%s", "", carolID, http.StatusNoContent,
			bobs("project_members", "project_id = $1 AND member_id = '"+carol+"'"),
			"SELECT updated_at, updated_by_id = $2 AND NOT is_active FROM project_members WHERE project_id = $1 AND member_id = '" + carol + "'",
			1, 1},
		// Her own membership, ended: bob is Web's other admin.
		{"leaveProject", http.MethodPost, "/api/v0/projects/" + web.String() + "/leave", "", uuid.UUID{}, http.StatusNoContent,
			bobs("project_members", "project_id = $1 AND member_id = $2"),
			"SELECT updated_at, updated_by_id = $2 AND NOT is_active FROM project_members WHERE project_id = $1 AND member_id = $2", 1, 1},
````

````old server/internal/bootstrap/project_writes_test.go
			t.Fatalf("%s's rows before it: %v; want %d, none of alice's writing", w.name, seeded, w.seeded)
		}
````
````new server/internal/bootstrap/project_writes_test.go
			t.Fatalf("%s's rows before it: %v; want %d, none of alice's writing", w.name, seeded, w.seeded)
		}
		path := w.path
		if w.of != (uuid.UUID{}) {
			path = fmt.Sprintf(path, projectMemberships(t, pool, w.of, web)[0])
		}
````

````old server/internal/bootstrap/project_writes_test.go
		status, body := call(t, contract, w.method, base+w.path, alice, w.body)
````
````new server/internal/bootstrap/project_writes_test.go
		status, body := call(t, contract, w.method, base+path, alice, w.body)
````

- [ ] **Step 3: 测试和 lint**

Run: `go -C server test -count=1 -race -run 'TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile$|TestAWriteOnAProjectMembershipDecidesOnWhatItReadUnderItsLocks$|TestEachLockOfAWriteOnAProjectMembershipIsItsStrength$|TestTheWritesOnAProjectRunOnTheirTransactionsConnection$|TestTheWritesOnAProjectStampTheirRequest$' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/bootstrap/project_connection_test.go server/internal/bootstrap/project_membership_races_test.go server/internal/bootstrap/project_membership_role_race_test.go server/internal/bootstrap/project_writes_test.go
```
```bash
git commit -m "test(M3/P5b): the writes on a project membership against what changes while they wait, their locks, their connection and their stamps

On the wired app, a change of role, a removal and a leaving that wait
for a lock while the membership, the caller's own, the project or the
workspace ends, is deleted or moves, answer 404, never a revealing 403,
and change no row; a change of role and a removal decide on the
membership as read again under the locks, its role included. Each takes
the workspace FOR SHARE, the member's workspace membership FOR SHARE
for a role change alone, then the project FOR NO KEY UPDATE, holding
nothing before; runs on its transaction's connection; and stamps its
row with the caller and a moment of its request, read under the locks.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A，清扫 4、8、9、12、13、24、25）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| 在锁之前判定（改角色、移出；离开） | `TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile`（调用者的成员关系结束：403 而不是 404）；离开另由 `TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize`（Task 9 起） | 组合 |
| 不在锁下重读；重读挪到锁之前；重读之后不核对项目；已结束的照样改、照样移出 | 同上（gina 的成员关系结束、删除、移到 Ops） | 组合 |
| 相对规则用锁之前读到的角色；重读挪到判定之后 | `TestAWriteOnAProjectMembershipDecidesOnWhatItReadUnderItsLocks` | 组合 |
| 项目在工作区之前锁；成员关系的第一次读带 `FOR UPDATE`；成员在工作区的成员关系在项目之后锁 | `TestEachLockOfAWriteOnAProjectMembershipIsItsStrength`、`TestEachWriteOnAProjectSharesItsWorkspaceFirst` | 组合 |
| 写成员关系时项目锁成 `FOR SHARE`；离开时项目锁成 `FOR SHARE`；改角色不锁成员在工作区的成员关系 | `TestEachLockOfAWriteOnAProjectMembershipIsItsStrength` | 组合 |
| 写不取工作区的 S | `TestEachLockOfAWriteOnAProjectMembershipIsItsStrength`、`TestEachWriteOnAProjectSharesItsWorkspaceFirst`；交错（Task 9 起）；单元一层由 P4b 起每个用例的调用记录 | 单元；组合 |
| 四个存储方法之一经连接池 | `TestTheWritesOnAProjectRunOnTheirTransactionsConnection` | 组合 |
| `project.New` 的改角色、离开不开事务 | `TestEachLockOfAWriteOnAProjectMembershipIsItsStrength`（没有事务就不持锁） | 组合 |
| `project.New` 的改角色、移出用停在 2001 年的时钟；不写写者 | `TestTheWritesOnAProjectStampTheirRequest`；`TestEachLockOfAWriteOnAProjectMembershipIsItsStrength`（时刻） | 组合 |

**Done when:** 三个写的每个竞争情形 404、一行不改；改角色、移出按锁下重读到的成员关系判定（它的角色变了照新的拒绝，删除了 404 不是 403）；每把锁在组合一层钉住顺序和强度；每条语句在事务的连接上；每个写的时刻和写者。

---

### Task 9: 交错 1 的项目一侧；离开与 P5a 的结束

**Files:**
- Create: `server/internal/bootstrap/interleaving_leaving_test.go`

**Interfaces:** 没有新的产品代码（spec 2.12，M3 设计 3.6、3.7 规则 1、2，9.3，P5a review 第 6 节）。`interleaving_leaving_test.go`：
- `projectOtherFoundHolding`（离开问过另一位管理员之后停在 gate，持 acme 的 S 和 Ops 的 N，在结束之前）、`projectEndedHolding`（写完成员关系之后停在 gate）；`leavings`（"at the check"、"past the membership's end"）。
- `memberWorld.leaveProject(ctx, name, project, leavers)`：项目的 `LeaveProject` 用例，`Locks` 照 `project.New` 接（项目的存储、工作区的目录和成员锁、`Authorizer`），系统时钟；`removeFromAcme`：工作区的移出，带项目的连带，照 `bootstrap` 接；`opsEndedBy`。
- 每个等待都有期限（`soon(t)`、`receiveWithin`）；第二方在它说的那一行上等，由 `pgtest.WaitForLockWaitOn(…, "projects" | "workspaces", …)` 证明：两方都不写这张表，探针只由那一把锁的等待满足。

**Tests:**
- `TestTwoProjectAdminsLeavingLeaveAnAdmin`（交错 1 的项目一侧，两个 gate × 两种先后，4 个子测试）：Ops 的两位管理员 alice、carol 同时离开，bob 是成员。第一位持 acme 的 S 和 Ops 的 N 停在 gate；第二位共享 acme、等 Ops 的行；第一位提交之后第二位找不到另一位有效管理员：409 `project.sole_admin`；第一位结束、第二位有效：Ops 留一位管理员。停在问过之后的 gate 证明问和结束在同一把锁下：在两者之间放开 Ops 的实现让第二位看到第一位仍有效，两位都离开（T6-b 的写法）。
- `TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize`（离开与 P5a 的移出，两种顺序 × 两个 gate × 两个离开的人，6 个子测试；`serialized`、`sameOutcome` 比较第一个 `*shared.Error`）：
  - carol（同一个成员）离开 Ops、gina 把 carol 移出 acme：离开先，她在 Ops 的成员关系由她自己结束，移出再结束其余的（Web）；移出先，全部由 gina 结束，离开找到她不是 acme 的成员：404 `project.not_found`。
  - alice（Ops 的另一位管理员）离开 Ops、gina 移出 carol：离开先，carol 成为 Ops 唯一的管理员（bob 是成员），移出在 Ops 的锁下看到它：409 `project.sole_admin`（规则 2），什么都不结束；移出先，carol 在 Ops 的成员关系结束，离开在 Ops 的锁下找不到另一位有效管理员：409（规则 1）。两种都是 Ops 留一位管理员。
  - 两方都先取 acme 的行（离开 S、移出 N）再取 Ops 的，不成环（spec 2.12）。

- [ ] **Step 1: 交错测试**

`server/internal/bootstrap/interleaving_leaving_test.go`（新文件，258 行）：

````file server/internal/bootstrap/interleaving_leaving_test.go
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	projectpg "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	projectdomain "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Interleaving 1, the project's side, and a leaving of a project against
// P5a's ending of a workspace membership (M3 design 9.3, 3.7 rules 1 and
// 2), on memberWorld's real database, in both orders: project's
// LeaveProject over Locks, and workspace's removal with project's cascade,
// each with the Authorizer as bootstrap wires them, and gated as
// interleaving_endings_test.go gates. Every wait has a deadline.

// projectOtherFoundHolding stops a leaving of a project once it has asked
// whether the project has another admin, holding the workspace FOR SHARE
// and the project FOR NO KEY UPDATE, before it ends anything.
type projectOtherFoundHolding struct {
	*projectpg.Store
	gate *gate
}

func (m projectOtherFoundHolding) HasOtherAdmin(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	other, err := m.Store.HasOtherAdmin(ctx, projectID, userID)
	if err != nil {
		return false, err
	}
	return other, m.gate.wait(ctx)
}

// projectEndedHolding stops a leaving of a project after its write of the
// membership, holding the workspace, the project and the membership's row.
type projectEndedHolding struct {
	*projectpg.Store
	gate *gate
}

func (m projectEndedHolding) EndMember(ctx context.Context, projectID, userID, by uuid.UUID, now time.Time) error {
	if err := m.Store.EndMember(ctx, projectID, userID, by, now); err != nil {
		return err
	}
	return m.gate.wait(ctx)
}

// leaving is how a leaving of a project is held at its gate.
type leaving struct {
	name    string
	holding func(store *projectpg.Store, g *gate) projectapp.MemberLeaver
}

var leavings = []leaving{
	{"at the check", func(store *projectpg.Store, g *gate) projectapp.MemberLeaver {
		return projectOtherFoundHolding{store, g}
	}},
	{"past the membership's end", func(store *projectpg.Store, g *gate) projectapp.MemberLeaver {
		return projectEndedHolding{store, g}
	}},
}

// leaveProject is name's leaving of project, through project's use case
// over leavers, with Locks over project's store, workspace's directory and
// members' locks and the Authorizer, as project.New wires them.
func (w memberWorld) leaveProject(ctx context.Context, name string, project uuid.UUID, leavers projectapp.MemberLeaver) error {
	provided := workspace.Provide(w.pool)
	locks := projectapp.NewLocks(projectpg.New(w.pool), projectWorkspaces{directory: provided.WorkspaceDirectory}, provided.WorkspaceMembers,
		authorizerOn(w.pool))
	return projectapp.NewLeaveProject(locks, leavers, postgres.NewTxManager(w.pool, 2*time.Second), clock.System{}).
		Execute(shared.WithActor(ctx, shared.Actor{UserID: w.ids[name]}), project)
}

// removeFromAcme is by's removal of name from acme, over members, with
// project's cascade, identity's profiles and the Authorizer as bootstrap
// wires them.
func (w memberWorld) removeFromAcme(t *testing.T, ctx context.Context, by, name string, members workspaceapp.MemberRemover) error {
	t.Helper()
	var id uuid.UUID
	if err := w.pool.QueryRow(soon(t), `SELECT m.id FROM workspace_members m JOIN workspaces s ON s.id = m.workspace_id
		WHERE s.slug = 'acme' AND m.member_id = $1`, w.ids[name]).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return workspaceapp.NewRemoveWorkspaceMember(members, workspaceProfiles{profiles: identity.Provide(w.pool).PublicProfiles},
		project.New(project.Deps{Pool: w.pool}).Cascade(), authorizerOn(w.pool), postgres.NewTxManager(w.pool, 2*time.Second), clock.System{}).
		Execute(shared.WithActor(ctx, shared.Actor{UserID: w.ids[by]}), id)
}

// opsEndedBy is the name of who ended name's membership of Ops last, or
// "active" while it is.
func (w memberWorld) opsEndedBy(t *testing.T, name string) string {
	t.Helper()
	var by string
	if err := w.pool.QueryRow(soon(t), `SELECT CASE WHEN m.is_active THEN 'active' ELSE split_part(u.email, '@', 1) END
		FROM project_members m JOIN users u ON u.id = m.updated_by_id WHERE m.project_id = $1 AND m.member_id = $2`, w.ops, w.ids[name]).
		Scan(&by); err != nil {
		t.Fatal(err)
	}
	return by
}

// The world's standing, as memberWorld makes it, then with carol's
// memberships of acme and of its projects ended, and with alice's or
// carol's membership of Ops ended.
const (
	worldStanding = "acme: alice 20, bob 15, carol 15, dave 15, erin 5, gina 20; beta: bob 20, carol 15; Lab: bob 20, carol 15; " +
		"Ops: alice 20, bob 15, carol 20; Web: alice 20, bob 20, carol 15, dave 20, erin 5, gina 15"
	carolOutOfAcme = "acme: alice 20, bob 15, dave 15, erin 5, gina 20; beta: bob 20, carol 15; Lab: bob 20, carol 15; " +
		"Ops: alice 20, bob 15; Web: alice 20, bob 20, dave 20, erin 5, gina 15"
	aliceOutOfOps = "acme: alice 20, bob 15, carol 15, dave 15, erin 5, gina 20; beta: bob 20, carol 15; Lab: bob 20, carol 15; " +
		"Ops: bob 15, carol 20; Web: alice 20, bob 20, carol 15, dave 20, erin 5, gina 15"
	carolOutOfOps = "acme: alice 20, bob 15, carol 15, dave 15, erin 5, gina 20; beta: bob 20, carol 15; Lab: bob 20, carol 15; " +
		"Ops: alice 20, bob 15; Web: alice 20, bob 20, carol 15, dave 20, erin 5, gina 15"
)

// Interleaving 1, the project's side: Ops's two admins, alice and carol,
// leave it at once (M3 design 3.6, 3.7 rule 1); bob is its member. The
// first holds acme FOR SHARE and Ops FOR NO KEY UPDATE and waits at his
// gate: once he has found the other admin, before he ends anything; or once
// he has ended his membership. The second shares acme and waits for Ops's
// row: neither writes a row of projects, so only that lock's wait satisfies
// the probe. Once the first has committed, the second finds no other active
// admin: 409 project.sole_admin. The first one's membership is ended, the
// second's active: Ops keeps an admin. Held at his check, the first shows
// that he asks rule 1 under the lock his ending holds: a leaving that let
// Ops go between the two would let the second find him still active, and
// both would leave.
func TestTwoProjectAdminsLeavingLeaveAnAdmin(t *testing.T) {
	for _, l := range leavings {
		for _, aliceFirst := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s, alice first %v", l.name, aliceFirst), func(t *testing.T) {
				w := newMemberWorld(t)
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				first, second, want := "alice", "carol", aliceOutOfOps
				if !aliceFirst {
					first, second, want = second, first, carolOutOfOps
				}
				g := newGate()
				left := run(func() error { return w.leaveProject(ctx, first, w.ops, l.holding(projectpg.New(w.pool), g)) })
				held(t, ctx, g, left, "the first leaving")
				refused := run(func() error { return w.leaveProject(ctx, second, w.ops, projectpg.New(w.pool)) })
				pgtest.WaitForLockWaitOn(t, w.pool, "projects", 5*time.Second)
				if got := w.standing(t); got != worldStanding {
					t.Errorf("while the first holds Ops: %s; want %s", got, worldStanding)
				}
				close(g.open)

				if err := result(t, ctx, left, "the first leaving"); err != nil {
					t.Errorf("the first leaving = %v, want it done", err)
				}
				if err := result(t, ctx, refused, "the second leaving"); !errors.Is(err, projectdomain.ErrSoleAdmin) {
					t.Errorf("the second leaving = %v, want 409 project.sole_admin", err)
				}
				if got := w.standing(t); got != want {
					t.Errorf("after both: %s; want %s", got, want)
				}
			})
		}
	}
}

// A leaving of Ops and gina's removal of carol from acme, whose cascade
// ends carol's memberships of acme's projects (3.7 rule 2), serialize on
// acme's row in both orders, and never form a cycle: each takes acme's
// row first, the leaving FOR SHARE, the removal FOR NO KEY UPDATE, then
// Ops's (M3 design 3.6's lock table). The second waits for acme's row: no
// side writes a row of workspaces, so only that lock's wait satisfies the
// probe.
//   - carol leaves, the same member: the leaving first, held at its gate,
//     ends her membership of Ops, as hers; the removal then ends the others,
//     Web's. The removal first ends all of them, as gina's; the leaving then
//     finds carol no member of acme: 404 project.not_found.
//   - alice leaves, Ops's other admin: the leaving first leaves carol its
//     only admin, beside bob, its member; the removal, under Ops's lock,
//     sees it, and refuses: 409 project.sole_admin (rule 2). The removal
//     first ends carol's membership of Ops; the leaving, under Ops's lock,
//     finds no other active admin: 409 project.sole_admin (rule 1). Either
//     way Ops keeps an admin.
func TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize(t *testing.T) {
	for _, tt := range []struct {
		leaver                  string
		leaveFirst, removeFirst serialized // by which side came first
	}{
		{"carol", serialized{nil, nil, carolOutOfAcme, "carol"}, serialized{projectdomain.ErrNotFound, nil, carolOutOfAcme, "gina"}},
		{"alice", serialized{nil, projectdomain.ErrSoleAdmin, aliceOutOfOps, "active"},
			serialized{projectdomain.ErrSoleAdmin, nil, carolOutOfAcme, "gina"}},
	} {
		for _, at := range slices.Concat(leavings, []leaving{{name: "the removal first"}}) {
			t.Run(tt.leaver+" leaves, "+at.name, func(t *testing.T) {
				w := newMemberWorld(t)
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				g := newGate()
				var left, removed <-chan error
				want := tt.leaveFirst
				if at.holding != nil {
					left = run(func() error { return w.leaveProject(ctx, tt.leaver, w.ops, at.holding(projectpg.New(w.pool), g)) })
					held(t, ctx, g, left, "the leaving")
					removed = run(func() error { return w.removeFromAcme(t, ctx, "gina", "carol", workspacepg.New(w.pool)) })
				} else {
					want = tt.removeFirst
					removed = run(func() error {
						return w.removeFromAcme(t, ctx, "gina", "carol", endedHolding{workspacepg.New(w.pool), g})
					})
					held(t, ctx, g, removed, "the removal")
					left = run(func() error { return w.leaveProject(ctx, tt.leaver, w.ops, projectpg.New(w.pool)) })
				}
				pgtest.WaitForLockWaitOn(t, w.pool, "workspaces", 5*time.Second)
				if got := w.standing(t); got != worldStanding {
					t.Errorf("while the first holds acme: %s; want %s", got, worldStanding)
				}
				close(g.open)

				leave, removal := result(t, ctx, left, "the leaving"), result(t, ctx, removed, "the removal")
				if !sameOutcome(leave, want.leave) || !sameOutcome(removal, want.remove) {
					t.Errorf("the leaving = %v, the removal = %v; want %v, %v", leave, removal, want.leave, want.remove)
				}
				if got, ops := w.standing(t), w.opsEndedBy(t, "carol"); got != want.standing || ops != want.ops {
					t.Errorf("after both: %s, carol's membership of Ops %s; want %s, %s", got, ops, want.standing, want.ops)
				}
			})
		}
	}
}

// serialized is how a leaving of Ops and a removal from acme end: the
// leaving's and the removal's errors, the world after them, and who ended
// carol's membership of Ops ("active" while it is).
type serialized struct {
	leave, remove error
	standing, ops string
}

// sameOutcome is whether err is want: nil for nil, else an error whose
// first problem in its chain, the one the API would answer, is want.
func sameOutcome(err, want error) bool {
	if want == nil {
		return err == nil
	}
	var first *shared.Error
	return errors.As(err, &first) && error(first) == want
}
````

- [ ] **Step 2: 测试和 lint**

Run: `go -C server test -count=5 -race -run 'TestTwoProjectAdminsLeavingLeaveAnAdmin$|TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize$' ./internal/bootstrap/`
Expected: `ok`；输出里没有 `40P01`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 3: 提交**

```bash
git add server/internal/bootstrap/interleaving_leaving_test.go
```
```bash
git commit -m "test(M3/P5b): interleaving 1 on the project's side, and a leaving against a workspace removal

Two admins of a project leaving at once serialize on the project's row,
held at the other-admin check or past the ending: the second is refused
with project.sole_admin, and the project keeps an admin. A leaving of a
project and a removal from its workspace, whose cascade ends the
member's project memberships, serialize on the workspace's row in both
orders and never form a cycle: each order keeps the project an admin.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A 的探针反例和清扫 25）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| 离开先在一个事务里问另一位管理员，再在另一个事务里结束 | `TestTwoProjectAdminsLeavingLeaveAnAdmin`（问过之后的 gate：两位都离开） | 组合（`-race`） |
| 离开把项目锁成 `FOR SHARE` | `TestTwoProjectAdminsLeavingLeaveAnAdmin`（探针：第二位不在 Ops 的行上等） | 组合 |
| 项目级的写不取工作区的 S | `TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize`（探针：第二方不在 acme 的行上等） | 组合 |
| 离开在锁之前判定 | `TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize`（移出先：403 而不是 404） | 组合 |

**Done when:** 两个交错 `-count=5 -race` 通过，没有 40P01。

---

### Task 10: 替身换成 P5b 的写

**Files:**
- Modify: `server/internal/bootstrap/interleaving_growth_test.go`、`server/internal/bootstrap/interleaving_writes_test.go`、`server/internal/bootstrap/permission_matrix_members_test.go`、`server/internal/bootstrap/permission_matrix_seed_test.go`、`server/internal/bootstrap/permission_matrix_seeded_test.go`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/bootstrap/project_members_test.go`、`server/internal/bootstrap/project_visibility_test.go`、`server/internal/bootstrap/project_writes_test.go`

**Interfaces:** 没有新的产品代码（spec 2.14，M3 设计 9.2；P5a spec 第 5 节 P5b 一行）：
- 矩阵：`projectSeed.endings()`：经项目的存储 `EndMember` 结束 P-前在私有项目的成员关系（由 acme 的管理员，移出）和 WG- 在公开项目的（由他自己，离开），在种子的时刻。`standIns` 删除；`partingStates` 只留下没有写会留下的状态：两种已删除的成员关系（只随已删除的项目、工作区出现）和有效成员关系没有显示设置，并照旧先核对 `exec` 对不改一行的语句失败。说明（`prepareMatrix`、`matrixSeed`、文件头、`matrixProjectMembers`、`listsTheProjectMembers`、`project_visibility_test.go`）照改。
- `project_writes_test.go`：负责人、默认指派人的测试里 bob 的已结束成员关系由 alice 经接口移出写出；"恢复"一行的 SQL 留着（之前没有显示设置的已结束成员关系：移出留着显示设置，没有写会留下这个状态）。
- `project_members_test.go`：恢复的测试每次由 alice 经接口改角色、移出，结束时带着那个角色；增长在提交时被拒的测试的 SQL 留着（同上的理由）。
- `interleaving_growth_test.go`：`newGrowthRace(ended)` 用 `projects.EndMember(ctx, r.web, r.bob, r.alice, now)`；`interleaving_writes_test.go`：`bobAdministersWeb` 用 `store.UpdateMemberRole(…, RoleAdmin, r.alice, time.Now())`。

**Tests:** 没有新测试；矩阵（`endings` 之外不变的格子照旧）、前提、完整性核对、恢复、增长、交错的测试照旧通过。

- [ ] **Step 1: 矩阵的种子**

`server/internal/bootstrap/permission_matrix_seed_test.go`（修改，5 处）：

````old server/internal/bootstrap/permission_matrix_seed_test.go
// the workspace store's and the project store's, the SQL that stands in
// for the stores later phases add, and the checks that the rows the cells
// rest on are there.
````
````new server/internal/bootstrap/permission_matrix_seed_test.go
// the workspace store's and the project store's, the SQL that makes the
// states no store makes alone, and the checks that the rows the cells rest
// on are there.
````

````old server/internal/bootstrap/permission_matrix_seed_test.go
// runs the SQL that stands in for the stores P5b adds, and makes the
// states no store makes alone (partingStates).
````
````new server/internal/bootstrap/permission_matrix_seed_test.go
// runs the SQL that makes the states no store makes alone (partingStates).
````

````old server/internal/bootstrap/permission_matrix_seed_test.go
}

// partingStates puts memberships of matrixProjectMembers in the states in
// which a list and reading could part (TestListingProjectsIsReadingEach):
// WG-'s membership of acme's public project ended and of its private one
// deleted, the member's of the private one deleted, and PM's display
// settings in it deleted while his membership stays active. SQL stands in
// for the store that will end one project membership (P5b), and makes the two
// deleted states that only a deleted project or workspace makes today,
// which the list must still read as reading does. Each state is then read
// back: one missing would let a list that counts an ended or a deleted
````
````new server/internal/bootstrap/permission_matrix_seed_test.go
}

// endings ends, through the project store, the project memberships that
// P5b's writes end, as their statement ends one (EndMember), at the seed's
// moment: the member before's of the private project, removed by acme's
// admin (removeProjectMember), and WG-'s of the public one, which he left
// (leaveProject).
func (s projectSeed) endings() {
	s.t.Helper()
	for _, e := range []struct {
		key   string
		c, by caller
	}{{"acme/private", callerBefore, matrixAdmins["acme"]}, {"acme/public", callerGuestOnly, callerGuestOnly}} {
		if err := s.store.EndMember(context.Background(), s.projects[e.key], s.ids[e.c], s.ids[e.by], s.now); err != nil {
			s.t.Fatal(err)
		}
	}
}

// partingStates puts memberships of matrixProjectMembers in the states in
// which a list and reading could part (TestListingProjectsIsReadingEach),
// beside WG-'s membership of acme's public project, which he left
// (endings): his membership of its private one deleted, the member's
// deleted too, and PM's display settings in it deleted while his membership
// stays active. SQL makes these, which no store makes alone: the two
// deleted states only a deleted project or workspace makes, which the list
// must still read as reading does, and display settings gone from a live
// membership; exec fails a statement that changes no row, and names it,
// which it checks first. Each state is then read back, the ended one too:
// one missing would let a list that counts an ended or a deleted
````

````old server/internal/bootstrap/permission_matrix_seed_test.go
func (s projectSeed) partingStates(pool *pgxpool.Pool) {
	s.t.Helper()
	public, private := s.projects["acme/public"], s.projects["acme/private"]
	s.exec(s.t, pool, "UPDATE project_members SET is_active = false, updated_at = $3 WHERE project_id = $1 AND member_id = $2",
		public, s.ids[callerGuestOnly], s.now)
````
````new server/internal/bootstrap/permission_matrix_seed_test.go
func (s projectSeed) partingStates(pool *pgxpool.Pool) {
	s.t.Helper()
	const none = "UPDATE project_members SET is_active = false WHERE false"
	if failed, want := fatalOf(func(tb testing.TB) { s.exec(tb, pool, none) }), none+" changed 0 rows, want 1"; failed != want {
		s.t.Errorf("exec of a statement that changes no row: failed with %q, want %q", failed, want)
	}
	public, private := s.projects["acme/public"], s.projects["acme/private"]
````

````old server/internal/bootstrap/permission_matrix_seed_test.go
	}
}

// standIns writes, through SQL, the states no store writes yet, until the
// phase that adds the store replaces it: the member before's membership of
// the private project ended (P5b, whose removal of a project member ends
// one). exec fails a statement that changes no row, and names it, which it
// checks first.
func (s projectSeed) standIns(pool *pgxpool.Pool) {
	s.t.Helper()
	const none = "UPDATE project_members SET is_active = false WHERE false"
	if failed, want := fatalOf(func(tb testing.TB) { s.exec(tb, pool, none) }), none+" changed 0 rows, want 1"; failed != want {
		s.t.Errorf("exec of a statement that changes no row: failed with %q, want %q", failed, want)
	}
	s.exec(s.t, pool, "UPDATE project_members SET is_active = false WHERE project_id = $1 AND member_id = $2",
		s.projects["acme/private"], s.ids[callerBefore])
````
````new server/internal/bootstrap/permission_matrix_seed_test.go
	}
````

`server/internal/bootstrap/permission_matrix_seeded_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_seeded_test.go
// in the private one, ended; WG- in both, and the member in the private
// one, for partingStates to end or delete; the archived project's admin;
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
// in the private one, whose removal from it ends his (endings); WG- in
// both, who leaves the public one (endings), and the member in the
// private one, for partingStates to delete; the archived project's admin;
````

`server/internal/bootstrap/permission_matrix_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_test.go
// the invitations of matrixInvitations. Through SQL, the states no store
// writes yet (standIns, partingStates). Through the API, gone deleted by
````
````new server/internal/bootstrap/permission_matrix_test.go
// the invitations of matrixInvitations. Through the project store, the
// memberships P5b's writes end (endings); through SQL, the states no store
// makes alone (partingStates). Through the API, gone deleted by
````

````old server/internal/bootstrap/permission_matrix_test.go
		projects.standIns(pool)
````
````new server/internal/bootstrap/permission_matrix_test.go
		projects.endings()
````

`server/internal/bootstrap/permission_matrix_members_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_members_test.go
// WG-, whose membership partingStates ended.
````
````new server/internal/bootstrap/permission_matrix_members_test.go
// WG-, who left it (endings).
````

`server/internal/bootstrap/project_visibility_test.go`（修改，1 处）：

````old server/internal/bootstrap/project_visibility_test.go
// in the same state, partingStates' among them: a guest's ended and
// deleted memberships, a member's deleted one, an active one without
````
````new server/internal/bootstrap/project_visibility_test.go
// in the same state, partingStates' among them: a guest's ended (endings)
// and deleted memberships, a member's deleted one, an active one without
````

- [ ] **Step 2: 其余的替身**

`server/internal/bootstrap/project_writes_test.go`（修改，3 处）：

````old server/internal/bootstrap/project_writes_test.go
		// before (P5b's removal of a project member ends one; SQL stands
		// in), restored, and her display settings made: two rows.
````
````new server/internal/bootstrap/project_writes_test.go
		// before, without display settings, which no write leaves (a
		// removal keeps them; SQL makes it), restored, and her display
		// settings made: two rows.
````

````old server/internal/bootstrap/project_writes_test.go
// whom she adds as a guest, bob, whose membership ended (P5b's removal of a
// project member ends one; SQL stands in), and dave, acme's member and none
// of Web's, are each refused as lead and as default assignee: 422 naming
// the field not_allowed, and the project as it was. erin, its member, is
// taken as both.
````
````new server/internal/bootstrap/project_writes_test.go
// whom she adds as a guest, bob, whom she adds as a member and removes,
// and dave, acme's member and none of Web's, are each refused as lead and
// as default assignee: 422 naming the field not_allowed, and the project
// as it was. erin, its member, is taken as both.
````

````old server/internal/bootstrap/project_writes_test.go
	if tag, err := pool.Exec(context.Background(), "UPDATE project_members SET is_active = false WHERE project_id = $1 AND member_id = $2",
		web, ids["bob"]); err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("ending bob's membership: %v, %v", tag, err)
````
````new server/internal/bootstrap/project_writes_test.go
	if status, body := call(t, contract, http.MethodDelete, base+"/api/v0/project-members/"+projectMemberships(t, pool, ids["bob"], web)[0].String(),
		alice, ""); status != http.StatusNoContent {
		t.Fatalf("alice's removing bob = %d %s", status, body)
````

`server/internal/bootstrap/project_members_test.go`（修改，3 处）：

````old server/internal/bootstrap/project_members_test.go
// member, he is left as he is: nothing is written. Then, each time, his
// membership is ended with a role, as P5b's removal of a project member
// will end it (SQL stands in), and he joins again, or alice adds him with a
// role: the same row is active again with the role of 9.1's row, by the
// caller at the time of that request, still made when it was. alice's adds
// take the role she asks for, below the ended one and above it. His
// workspace role is changed through the API before the last row.
````
````new server/internal/bootstrap/project_members_test.go
// member, he is left as he is: nothing is written. Then, each time, alice
// changes his role and removes him, which ends his membership with that
// role, and he joins again, or alice adds him with a role: the same row is
// active again with the role of 9.1's row, by the caller at the time of
// that request, still made when it was. alice's adds take the role she
// asks for, below the ended one and above it. His workspace role is
// changed through the API before the last row.
````

````old server/internal/bootstrap/project_members_test.go
		if tag, err := pool.Exec(context.Background(), "UPDATE project_members SET role = $3, is_active = false WHERE project_id = $1 AND member_id = $2",
			web, bobID, tt.was); err != nil || tag.RowsAffected() != 1 {
			t.Fatalf("%s: ending bob's membership = %v, %v", tt.name, tag, err)
````
````new server/internal/bootstrap/project_members_test.go
		membership := base + "/api/v0/project-members/" + first.id.String()
		if status, body := call(t, contract, http.MethodPatch, membership, alice, fmt.Sprintf(`{"role":%d}`, tt.was)); status != http.StatusOK {
			t.Fatalf("%s: alice's making bob %d = %d %s", tt.name, tt.was, status, body)
		}
		if status, body := call(t, contract, http.MethodDelete, membership, alice, ""); status != http.StatusNoContent {
			t.Fatalf("%s: alice's removing bob = %d %s", tt.name, status, body)
````

````old server/internal/bootstrap/project_members_test.go
// there (P5b's removal of a project member ends it; SQL stands in): each
// time, with the commits of memberships refused, then those of display
// settings, which the growth writes last, alice's adding him and his
// joining answer 500 and change no membership and no display settings.
// Once commits are allowed again, he joins.
````
````new server/internal/bootstrap/project_members_test.go
// there, which no write leaves (a removal keeps them; SQL makes it), so
// that the growth writes both: each time, with the commits of memberships
// refused, then those of display settings, which the growth writes last,
// alice's adding him and his joining answer 500 and change no membership
// and no display settings. Once commits are allowed again, he joins.
````

`server/internal/bootstrap/interleaving_growth_test.go`（修改，2 处）：

````old server/internal/bootstrap/interleaving_growth_test.go
// membership as a member when ended is set: P5b's removal of a project
// member ends one, and SQL stands in for it.
````
````new server/internal/bootstrap/interleaving_growth_test.go
// membership as a member when ended is set: alice's removal of him ended
// it, through the project store's statement (EndMember).
````

````old server/internal/bootstrap/interleaving_growth_test.go
		if tag, err := r.pool.Exec(ctx, "UPDATE project_members SET is_active = false WHERE project_id = $1 AND member_id = $2", r.web, r.bob); err != nil ||
			tag.RowsAffected() != 1 {
			t.Fatalf("ending bob's membership of Web: %v, %v", tag, err)
````
````new server/internal/bootstrap/interleaving_growth_test.go
		if err := projects.EndMember(ctx, r.web, r.bob, r.alice, now); err != nil {
			t.Fatalf("ending bob's membership of Web: %v", err)
````

`server/internal/bootstrap/interleaving_writes_test.go`（修改，3 处）：

````old server/internal/bootstrap/interleaving_writes_test.go
// Web and is its admin (SQL stands in for P5b's role change), and Ops is
// another project of acme, of which alice is the admin.
````
````new server/internal/bootstrap/interleaving_writes_test.go
// Web and alice has made him its admin, through the project store's
// statement (UpdateMemberRole), and Ops is another project of acme, of
// which alice is the admin.
````

````old server/internal/bootstrap/interleaving_writes_test.go
	if _, err := r.pool.Exec(context.Background(), "UPDATE project_members SET role = 20 WHERE project_id = $1 AND member_id = $2", r.web,
		r.bob); err != nil {
````
````new server/internal/bootstrap/interleaving_writes_test.go
	store := projectpg.New(r.pool)
	if _, err := store.UpdateMemberRole(context.Background(), projectMemberships(t, r.pool, r.bob, r.web)[0], shared.RoleAdmin, r.alice,
		time.Now()); err != nil {
````

````old server/internal/bootstrap/interleaving_writes_test.go
	}
	store := projectpg.New(r.pool)
````
````new server/internal/bootstrap/interleaving_writes_test.go
	}
````

- [ ] **Step 3: 测试和 lint**

Run: `go -C server test -count=1 -run 'TestPermissionMatrix$|TestListingProjectsIsReadingEach|TestARestoredMembershipGivesNoMoreThanItHad$|TestAGrowthRefusedAtItsCommitLeavesNoRow$|TestTheLeadAndTheDefaultAssigneeAreActiveMembersWhoAreNoGuests$' ./internal/bootstrap/`
Expected: `ok`。

Run: `go -C server test -count=1 -race -run 'TestADemotionAndTheProjectSidesGrowthSerialize$|TestARemovalAndTheProjectSidesGrowthSerialize$|TestTwoWritesOnAProjectSerialize$|TestWritesOnTwoProjectsOfAWorkspaceDoNotWait$' ./internal/bootstrap/`
Expected: `ok`；输出里没有 `40P01`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/bootstrap/interleaving_growth_test.go server/internal/bootstrap/interleaving_writes_test.go server/internal/bootstrap/permission_matrix_members_test.go server/internal/bootstrap/permission_matrix_seed_test.go server/internal/bootstrap/permission_matrix_seeded_test.go server/internal/bootstrap/permission_matrix_test.go server/internal/bootstrap/project_members_test.go server/internal/bootstrap/project_visibility_test.go server/internal/bootstrap/project_writes_test.go
```
```bash
git commit -m "test(M3/P5b): the stand-ins become P5b's writes

The matrix's former member of the private project and the workspace
guest's left membership of the public one are ended by the project
store, as a removal and a leaving end them; the tests that ended,
changed or removed a project membership through SQL now do it through
the API or the store. SQL stays only for states no write leaves:
memberships deleted without their project, and display settings gone
from a membership.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A，清扫 10）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| `endings` 不结束 P-前的成员关系；不结束 WG- 的 | `TestPermissionMatrix/prepare`（`memberships`、`partingStates` 读回的前提） | 组合 |
| `EndMember` 去掉成员（结束整个项目的） | `TestPermissionMatrix/prepare`；`TestADemotionAndTheProjectSidesGrowthSerialize` | 组合 |

**Done when:** 矩阵里没有替身；剩下的 SQL 每一处都写明为什么没有写会留下那个状态。

---

### Task 11: 端到端：P5 的接口版本（含 W7 的"降级含已离开的项目"）

**Files:**
- Create: `e2e/stories/project/p5-project-members.spec.ts`
- Modify: `e2e/fixtures/assert/project.ts`

**Interfaces:**（spec 2.15，M3 设计第 2 节 P5、W7，9.6）
- 夹具：`MemberRow{email, role, is_active, by, sort_order, settings_by}`；`expectMembers(db, projectId, want)`：项目未删除的成员关系（已结束的也在），按地址排序，每行带他未删除的显示设置（已结束的留着），每行都属于项目的工作区。

**Tests:**
- `e2e/stories/project/p5-project-members.spec.ts`：P5（API）。acme：管理员；mia、pam、ray、tom 成员，gus 访客，wanda 另一位管理员（各经邀请）；olga 不是成员。Web（管理员建，`amidAnotherWorkspace`）；Ops（acme 的另一个项目：tom 成员，wanda 另一位管理员）。一步接一步，每一步之后 `expectMembers` 读 Web（离开和降级之后也读 Ops）：
  1. 添加 olga（不是成员）、gus 为成员（工作区访客）、wanda 为成员（工作区管理员）各 422，什么都不加；
  2. 管理员一次添加 mia（15）、gus（5）：两行成员关系、两行显示设置；
  3. 管理员（Web 唯一的管理员；Ops 另有管理员 wanda）离开 409 `project.sole_admin`；mia（成员）改 gus 403；
  4. 管理员添加 pam、ray（20）、tom（15）；pam（不是工作区管理员）改 ray（另一位管理员）403 `project.role_too_high`；
  5. pam 把 mia 改为访客、移出 gus：两行由 pam 写，gus 的结束、角色和显示设置留着；
  6. tom 离开 Web：由他自己结束；他在 Ops 的成员关系不动；管理员把他降为 acme 的访客：他已结束的 Web 成员关系、有效的 Ops 成员关系都成为访客的（W7 的一半）；
  7. pam 移出 ray（另一位管理员、acme 的成员）；ray 加入 Web：原来那一行回来，15（原来的 20 以工作区角色 15 为上限，3.5）；成员列表里是他原来的 id。

- [ ] **Step 1: 夹具**

`e2e/fixtures/assert/project.ts`（修改，1 处）：

````old e2e/fixtures/assert/project.ts
    tables.map(({ name }) => ({ table: name, deletedWithIt: true, other: 0 }))
  );
}

````
````new e2e/fixtures/assert/project.ts
    tables.map(({ name }) => ({ table: name, deletedWithIt: true, other: 0 }))
  );
}

/** A membership of a project as expectMembers reads it, its member by address. */
export interface MemberRow {
  email: string;
  role: number;
  is_active: boolean;
  /** The address of the account that wrote the membership last. */
  by: string;
  /** His place in his sidebar, and who wrote his display settings in the project last. */
  sort_order: number;
  settings_by: string;
}

/**
 * P5: the undeleted memberships of the project of projectId, ended ones too, are exactly want, in their members'
 * addresses' order, each beside his undeleted display settings in the project: an ended membership keeps them. Every
 * row is of the project's workspace.
 */
export async function expectMembers(db: Database, projectId: string, want: MemberRow[]): Promise<void> {
  expect(
    await db.query(
      `SELECT u.email, m.role, m.is_active, b.email AS by, s.sort_order, sb.email AS settings_by,
              m.workspace_id = p.workspace_id AND s.workspace_id = p.workspace_id AS in_its_workspace
         FROM project_members m
         JOIN projects p ON p.id = m.project_id
         JOIN users u ON u.id = m.member_id
         JOIN users b ON b.id = m.updated_by_id
         LEFT JOIN project_user_properties s ON s.project_id = m.project_id AND s.user_id = m.member_id AND s.deleted_at IS NULL
         LEFT JOIN users sb ON sb.id = s.updated_by_id
        WHERE m.project_id = $1 AND m.deleted_at IS NULL ORDER BY u.email COLLATE "C"`,
      [projectId]
    ),
    `the memberships of ${projectId}`
  ).toEqual(
    want
      .toSorted((a, b) => (a.email < b.email ? -1 : 1))
      .map(({ email, role, is_active, by, sort_order, settings_by }) => ({
        email,
        role,
        is_active,
        by,
        sort_order,
        settings_by,
        in_its_workspace: true,
      }))
  );
}

````

- [ ] **Step 2: 故事**

`e2e/stories/project/p5-project-members.spec.ts`（新文件，254 行）：

````file e2e/stories/project/p5-project-members.spec.ts
import {
  addProjectMembers,
  amidAnotherWorkspace,
  createProject,
  createWorkspace,
  inviteAndAccept,
  membershipOf,
  slugFor,
  type Api,
  type ProjectMemberNew,
} from "../../fixtures/api";
import { expectMembers, type MemberRow } from "../../fixtures/assert/project";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// P5, a project's members (M3 design 2, 3.5, 3.7): adding them, changing a
// role, removing a member, leaving. The page version comes with the
// project's members page (P10).

/** A write's answer: its status, and the problem's code and fields for a refusal. */
interface Answer {
  status: number;
  code?: string;
  errors?: { field: string; code: string }[];
}

function answer(
  response: Response,
  error?: { code: string; errors?: { field: string; code: string }[] | null } | null
): Answer {
  return error
    ? {
        status: response.status,
        code: error.code,
        errors: error.errors?.map((e) => ({ field: e.field, code: e.code })),
      }
    : { status: response.status };
}

/** The writes on a project's members, each by the caller of token. */
function writes(api: Api, projectId: string) {
  return {
    add: async (token: string, members: ProjectMemberNew[]) => {
      const { error, response } = await api.POST("/api/v0/projects/{project_id}/members", {
        params: { path: { project_id: projectId } },
        body: { members },
        headers: bearer(token),
      });
      return answer(response, error);
    },
    change: async (token: string, membership: string, role: 5 | 15 | 20) => {
      const { error, response } = await api.PATCH("/api/v0/project-members/{project_member_id}", {
        params: { path: { project_member_id: membership } },
        body: { role },
        headers: bearer(token),
      });
      return answer(response, error);
    },
    remove: async (token: string, membership: string) => {
      const { error, response } = await api.DELETE("/api/v0/project-members/{project_member_id}", {
        params: { path: { project_member_id: membership } },
        headers: bearer(token),
      });
      return answer(response, error);
    },
    leave: async (token: string) => {
      const { error, response } = await api.POST("/api/v0/projects/{project_id}/leave", {
        params: { path: { project_id: projectId } },
        headers: bearer(token),
      });
      return answer(response, error);
    },
  };
}

test("P5 (API): the admin adds a member and a guest at once, and cannot leave, the only admin; another admin makes the member a guest, removes the guest, and removes a third admin, who joins again as a member, his row back; a member leaves, his membership of the workspace's other project kept, and the workspace's making him a guest makes his ended membership a guest's; an add of one who is no workspace member, of a workspace guest or admin as a member, a member's change of a role and an admin's change of another admin's change nothing", async ({
  api,
  db,
}, testInfo) => {
  const account = async (label: string) => {
    const email = emailFor(testInfo, label);
    const token = (await createPAT(api, (await register(api, email)).access_token)).token;
    return { email, token, id: await accountId(api, token) };
  };
  const admin = await account("admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  // acme's members mia, pam, ray and tom, its guest gus, its other admin wanda; olga is no member of it.
  const [mia, gus, pam, ray, tom, wanda, olga] = await Promise.all(
    ["mia", "gus", "pam", "ray", "tom", "wanda", "olga"].map(account)
  );
  if (!mia || !gus || !pam || !ray || !tom || !wanda || !olga) {
    throw new Error("the accounts were not registered");
  }
  await Promise.all(
    (
      [
        [mia, 15],
        [gus, 5],
        [pam, 15],
        [ray, 15],
        [tom, 15],
        [wanda, 20],
      ] as const
    ).map(([who, role]) => inviteAndAccept(api, admin.token, slug, who, role))
  );
  const web = await amidAnotherWorkspace(api, admin.token, testInfo, () =>
    createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" })
  );
  const { add, change, remove, leave } = writes(api, web.id);
  // Ops, acme's other project: wanda its other admin, none of Web's.
  const ops = await createProject(api, admin.token, slug, { name: "Ops", identifier: "OPS" });
  await addProjectMembers(api, admin.token, ops.id, [{ member_id: wanda.id, role: 20 }]);
  // Each membership of Web or Ops, as expectMembers reads it, with its display settings, the admin's writing, as the
  // adds and the creations made them: at 65535, or at 55535 in Ops for the admin and tom, whose second project of acme
  // it is (10000 before the first, M3 design 3.18).
  const row = (
    who: { email: string },
    role: number,
    is_active: boolean,
    by: { email: string },
    sort_order = 65535
  ): MemberRow => ({
    email: who.email,
    role,
    is_active,
    by: by.email,
    sort_order,
    settings_by: admin.email,
  });
  let members = [row(admin, 20, true, admin)];
  let opsMembers = [row(admin, 20, true, admin, 55535), row(wanda, 20, true, admin)];
  await expectMembers(db, ops.id, opsMembers);

  // Refused, each adding nothing: olga, no member of acme; gus, its guest, as a member; wanda, its admin, as a member.
  expect(
    [
      await add(admin.token, [{ member_id: olga.id, role: 15 }]),
      await add(admin.token, [{ member_id: gus.id, role: 15 }]),
      await add(admin.token, [{ member_id: wanda.id, role: 15 }]),
    ],
    "the adds of olga, of gus and of wanda as members"
  ).toEqual([
    { status: 422, code: "validation_failed", errors: [{ field: "members[0].member_id", code: "not_allowed" }] },
    { status: 422, code: "validation_failed", errors: [{ field: "members[0].role", code: "not_allowed" }] },
    { status: 422, code: "validation_failed", errors: [{ field: "members[0].role", code: "not_allowed" }] },
  ]);
  await expectMembers(db, web.id, members);

  // The admin adds mia as a member and gus as a guest, at once: two memberships and two display settings, his.
  const added = await addProjectMembers(api, admin.token, web.id, [
    { member_id: mia.id, role: 15 },
    { member_id: gus.id, role: 5 },
  ]);
  expect(added.map((m) => [m.member_id, m.role])).toEqual([
    [mia.id, 15],
    [gus.id, 5],
  ]);
  const [mias, guss] = added.map((m) => m.id);
  if (!mias || !guss) {
    throw new Error("the add answered no memberships");
  }
  members = [...members, row(mia, 15, true, admin), row(gus, 5, true, admin)];
  await expectMembers(db, web.id, members);

  // Web's only admin cannot leave it; mia, its member, cannot change gus's role.
  expect(
    [await leave(admin.token), await change(mia.token, guss, 5)],
    "the admin's leaving, mia's change of gus"
  ).toEqual([
    { status: 409, code: "project.sole_admin" },
    { status: 403, code: "forbidden" },
  ]);
  await expectMembers(db, web.id, members);

  // The admin adds pam and ray as admins, tom as a member: pam, who is no workspace admin, cannot change ray's role,
  // another admin's.
  const more = await addProjectMembers(api, admin.token, web.id, [
    { member_id: pam.id, role: 20 },
    { member_id: ray.id, role: 20 },
    { member_id: tom.id, role: 15 },
  ]);
  const rays = more[1]?.id;
  if (!rays) {
    throw new Error("the add answered no membership of ray");
  }
  members = [...members, row(pam, 20, true, admin), row(ray, 20, true, admin), row(tom, 15, true, admin)];
  // tom is Ops's member too.
  await addProjectMembers(api, admin.token, ops.id, [{ member_id: tom.id, role: 15 }]);
  opsMembers = [...opsMembers, row(tom, 15, true, admin, 55535)];
  expect(await change(pam.token, rays, 15), "pam's change of ray, an admin").toEqual({
    status: 403,
    code: "project.role_too_high",
  });
  await expectMembers(db, web.id, members);

  // pam makes mia a guest and removes gus: each row written by pam, gus's ended with its role and his display
  // settings kept.
  expect(
    [await change(pam.token, mias, 5), await remove(pam.token, guss)],
    "pam's change of mia, her removal of gus"
  ).toEqual([{ status: 200 }, { status: 204 }]);
  members = members.map((m) =>
    m.email === mia.email ? row(mia, 5, true, pam) : m.email === gus.email ? row(gus, 5, false, pam) : m
  );
  await expectMembers(db, web.id, members);

  // tom leaves Web: his membership ends, by him; his membership of Ops stays. The admin then makes him acme's guest:
  // his ended membership of Web becomes a guest's too, and so does his membership of Ops (M3 design 2, W7).
  expect(await leave(tom.token), "tom's leaving").toEqual({ status: 204 });
  members = members.map((m) => (m.email === tom.email ? row(tom, 15, false, tom) : m));
  await expectMembers(db, web.id, members);
  await expectMembers(db, ops.id, opsMembers);
  const demoted = await api.PATCH("/api/v0/workspace-members/{workspace_member_id}", {
    params: { path: { workspace_member_id: await membershipOf(api, admin.token, slug, tom.id) } },
    body: { role: 5 },
    headers: bearer(admin.token),
  });
  expect(demoted.response.status, "the admin's making tom acme's guest").toBe(200);
  members = members.map((m) => (m.email === tom.email ? row(tom, 5, false, admin) : m));
  await expectMembers(db, web.id, members);
  opsMembers = opsMembers.map((m) => (m.email === tom.email ? row(tom, 5, true, admin, 55535) : m));
  await expectMembers(db, ops.id, opsMembers);

  // pam removes ray, another admin and a member of acme; he joins Web again: his row is back, a member's, the 20 it
  // kept no more than his workspace role, 15 (M3 design 3.5).
  expect(await remove(pam.token, rays), "pam's removal of ray").toEqual({ status: 204 });
  members = members.map((m) => (m.email === ray.email ? row(ray, 20, false, pam) : m));
  await expectMembers(db, web.id, members);
  const joined = await api.POST("/api/v0/projects/{project_id}/join", {
    params: { path: { project_id: web.id } },
    headers: bearer(ray.token),
  });
  expect([joined.response.status, joined.data?.member_role], "ray's joining Web again").toEqual([200, 15]);
  members = members.map((m) => (m.email === ray.email ? row(ray, 15, true, ray) : m));
  await expectMembers(db, web.id, members);
  const listed = await api.GET("/api/v0/projects/{project_id}/members", {
    params: { path: { project_id: web.id } },
    headers: bearer(admin.token),
  });
  expect(
    listed.data?.data.map((m) => [m.member_id, m.role, m.id === rays]).toSorted(),
    "Web's members, ray's membership his row of before"
  ).toEqual(
    (
      [
        [admin.id, 20, false],
        [mia.id, 5, false],
        [pam.id, 20, false],
        [ray.id, 15, true],
      ] as const
    ).toSorted()
  );
});
````

- [ ] **Step 3: 检查**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 66 个故事全部通过（此前的 65 个，加 P5）。

- [ ] **Step 4: 提交**

```bash
git add e2e/fixtures/assert/project.ts e2e/stories/project/p5-project-members.spec.ts
```
```bash
git commit -m "test(M3/P5b): the API version of story P5

An admin adds a member and a guest at once and cannot leave, the only
admin; another admin makes the member a guest, removes the guest, and
removes a third admin, who joins again as a member, his row back. A
member leaves, his membership of the workspace's other project kept,
and his demotion to the workspace's guest makes his ended membership a
guest's too. Each refusal design 2 lists changes nothing.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A 清扫 1、3、5、14、15、20、24 的故事一半；每个都只运行 P5）：

| 改坏 | 必须失败的故事 |
|---|---|
| `MemberByID` 去掉 id；`UpdateMemberRole` 去掉 id；`EndMember` 去掉成员 | P5 |
| `EndMember` 去掉项目、改成工作区的项目（tom 在 Ops 的成员关系） | P5 |
| `HasOtherAdmin` 去掉项目、改成工作区的项目（Ops 的 wanda）；去掉角色；把他自己算进去 | P5 |
| `EndMember` 也写角色；`UpdateMemberRole` 把角色写成 20 | P5 |
| 不写写者：移出、改角色写成成员自己的；改角色、结束保留原来的写者 | P5 |
| 改角色的规则给项目成员；离开的规则 1 不问 | P5 |
| 不受相对规则约束；不看 `From`（pam 改 ray）；移出只许低于自己的（pam 移出 ray） | P5 |
| `POST /leave` 交给移出的用例 | P5 |
| 降为访客跳过已结束的项目成员关系（W7 的一半） | P5、W7 |

故事看不到、单独运行 P5 照样通过的（每个由矩阵或组合测试发现，spec 第 3 节第 8 条）：移出的规则给项目成员、规则不给项目访客（故事里的项目成员不移出别人，离开的是成员）；移出自己的、改自己的角色（故事里没有人这样做）；每个离开的人都问另一位管理员、只有他一人照样离开（故事里离开的人的项目有别的管理员，唯一管理员的项目有别的成员）；不在锁下重读（故事是顺序的，两次读相同）；`Memberships` 的账户（P4b，每个调用者都按问到的账户取结果）。

**Done when:** 66 个故事中除 S3 的 F4（只在不是 git 仓库的副本里）外全部通过；P5 单独运行时上表每个变异都让它失败。
