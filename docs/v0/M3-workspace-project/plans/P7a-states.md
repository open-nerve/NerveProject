# M3/P7a 状态与按资源寻址的共用取锁路径 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 状态的全部操作（`listStates`、`createState`、`updateState`、`deleteState`、`markDefaultState`、`listWorkspaceStates`）和 3.17 的规则：新状态的 `sequence` 是非分诊状态的最大值加 15000；`group = triage` 是 422；每一组（分诊组除外）至少留一个状态，删除或改走组里唯一的状态 409 `project.state_last_in_group`；项目恰好一个默认状态，删除默认状态 409 `project.state_default`，设为默认在一个事务里两条语句；分诊状态对状态的操作不可见；已归档的项目 `listStates` 返回空列表，`listWorkspaceStates` 不含它。按资源寻址的写（P5b 的改角色、移出两个写和状态的三个写）共用一条取锁路径：不加锁地读这一行 → 工作区 `FOR SHARE` → 项目 `FOR NO KEY UPDATE` → 锁下重读、核对仍属那个项目 → 判定，各写答自己的 404；`lock` 核对工作区的锁回答的 id。交错测试 10 和一组最后两个状态的并发写两种顺序串行；故事 P6 和 W11 的接口版本通过。

**Architecture:** `project/app`：`lock.go` 的 `placed`、`rowWrite[R]`、`lockRowAndDecide[R]`、`rowWrite.read`、`findWorkspaceAndDecide`（从 `listProjects`、`checkProjectIdentifier` 提出来，`listWorkspaceStates` 也用）；`state_ports.go` 的八个端口；六个用例。`project/domain`：`state.go` 的规则（`CheckNewState`、`CheckStatePatch`、`SequenceAfter`、`CheckGroupKept`、`State.Place`），四个状态码。`project/adapter/postgres`：`queries/states.sql` 的十条语句和存储方法。`project/adapter/http`：六个处理函数。`access`：六行规则。`bootstrap`：矩阵的状态种子和行（已归档项目的小表随每个操作），最先锁工作区的测试、锁的阶梯、盖戳、连接各加状态的写，按资源寻址的竞争和锁强度的测试推广到状态的行，交错 10。不加迁移、表、Go 模块、npm 包；跨模块的端口不变。

**Tech Stack:** Go 1.27.1、pgx v5.11.0、sqlc v1.31.1（`CGO_ENABLED=0`）、oapi-codegen v2.8.0、goose v3.28.0、River v0.47.0、golangci-lint 2.13.2、PostgreSQL 18.6（testcontainers）；Node 24、pnpm 11.10.0、Playwright 1.63.0。

**Spec:** `docs/v0/M3-workspace-project/specs/P7a-states.md`（上级：`docs/v0/M3-workspace-project/M3-design.md`）

## Global Constraints

- **Go 版本和依赖**：本 plan 不执行 `go get`，不加任何 Go 模块或 npm 包。`server/go.mod` 和 `server/tools/go.mod` 保持 `go 1.27` 和 `toolchain go1.27.1`（golangci-lint 2.13.2 由 go1.27.0 构建，`go` 行更高就拒绝运行）。每个 Task 提交前执行 `grep -n "^go \|^toolchain" server/go.mod server/tools/go.mod`，四行必须是 `go 1.27` 和 `toolchain go1.27.1`；`git diff --stat 3f87fcd3 -- server/go.mod server/go.sum server/tools pnpm-lock.yaml` 在本 plan 的任何时刻都没有输出。
- **每个 Task 提交前**：`make lint-go` 两段都输出 `0 issues.`，`make test` 全部通过（没有 `FAIL`）。
- **生成的文件不手写、不从本 plan 复制**：有生成物的 Task（3、4、6、7、8）执行 Task 中的生成命令（只动了 sqlc 的 Task 3 执行 `make gen-go`，改了接口描述的 Task 4、6、7、8 执行 `make gen`），用 `shasum -a 256` 和 `wc -l` 核对表中的 SHA-256 和行数，对不上时停下来：说明某个输入与本 plan 不一致。这些 Task **提交之后**执行 `make gen-check`（它用 `git status` 判断生成物是否已提交，提交之前必然报差异）。
- **前端检查和新的错误码**（M3 设计 12 节约束 4）：声明新码的 Task 在同一个 Task 里把它加进 `PROBLEM_MESSAGES`（`web/apps/web/helpers/authentication.helper.ts`）和两份 `auth.json`：Task 4 `project.state_name_taken`，Task 6 `project.state_not_found`、`project.state_last_in_group`，Task 7 `project.state_default`。改了接口描述的 Task 4、6、7、8 和改了 `e2e/` 的 Task 10 另执行 `make lint-web`、`make knip`、`make test-web`；Task 10 执行 `make e2e`。
- **容器**：`make test` 和 `make e2e` 用自己的 testcontainers；机器忙时偶尔起不来，等 Docker 空闲之后重跑一次再当作失败。容器测试一次只跑一套。开发库 `nerve-dev-db-1` 可以用，但不要停止或重建它，不要执行 `make dev-db-down`、`make dev-db-reset`。不要碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`、`nervewiki-*`）。
- **git**：每次 Bash 调用只执行一个 git 命令，不用 `;`、`&&`、`|` 串联 git；不用 `git -C`、`stash`、`clean`、`reset --hard`。`cd` 不与别的命令组合，只读的命令也不行。不碰 `plane/`、`refer/`。
- **安装**：除了 Docker、Go、Node 不做任何全局安装；不执行 `corepack enable`（pnpm 已在 PATH 上）。
- **规则**：不写 `init()`，不用全局可变状态，构造函数显式传入依赖；不建 `utils`、`common`、`helpers` 包；一个文件只做一件事，不超过约 400 行；依赖只能向内；模块之间不互相导入，不跨模块的表 JOIN；模块的 SQL 只经 sqlc；角色只按集合判断，不按大小比较；不留没有使用者的代码。**例外**：接口描述按模块一个文件（M0-P3 交接 5），`api/modules/project.yaml`（1,139 行）不受约 400 行的限制。本 plan 的其余文件都在约 400 行以内（最终原型上量的）：最长的是 `bootstrap/permission_matrix_seed_test.go`（389 行）、`bootstrap/permission_matrix_test.go`（386 行）、`e2e/fixtures/assert/workspace.ts`（378 行，本 plan 只改一句注释）、`bootstrap/project_write_locks_test.go`（373 行）、`bootstrap/permission_matrix_columns_test.go`（353 行）、`project/adapter/postgres/update_test.go`（352 行）、`project/app/ports.go`（351 行）、`bootstrap/project_writes_test.go`（339 行）、`e2e/fixtures/assert/project.ts`（323 行）、`project/adapter/postgres/states_test.go`（321 行）、`bootstrap/permission_matrix_seeded_test.go`（314 行）。
- **注释**：Go、TS 代码、SQL 查询和接口描述用英文；中文文档照本 plan 原样。
- **代码块**：每个改动都写成四个反引号围起来的块，块的第一行写明种类和路径，照原样使用（原型中逐字节运行过）：
  - ````` ````file <路径> ````` 新文件，块的内容加一个结尾换行就是整个文件；
  - ````` ````whole <路径> ````` 已有文件的完整新内容（同样加结尾换行）；
  - ````` ````old <路径> ````` 与紧跟着的 ````` ````new <路径> `````：`old` 的文字在文件当前版本中恰好出现一次，把它换成 `new` 的文字；
  - ````` ````delete <路径> ````` 删除这个文件（块是空的）。

  一个文件的几个块按出现的顺序依次应用。拼 plan 的脚本已从 `3f87fcd3` 起按顺序核对过全部块：每个 `old` 恰好出现一次（在它之前的块应用之后的文件中），每个新文件原来不存在，逐 Task 应用之后的文件与原型逐字节相同（spec 附录 A）。可以用 `node <planapply.mjs> <本 plan> apply <仓库根> <n>` 写入第 n 个 Task 的块，也可以手工照抄。
- **过渡版本**：一些文件先在较早的 Task 写成过渡版本，较晚的 Task 再修改：契约 `api/modules/project.yaml`、`api/openapi.yaml`，`project/app/state_ports.go`、`fakes_state_test.go`、`clock_test.go`，`project/adapter/http/handler.go`、`handler_test.go`、`states.go`、`state_writes_test.go`，`project/module.go`、`project/domain/actions.go`，`access/domain/rules.go`、`rules_test.go`，`bootstrap/permission_matrix_states_test.go`、`permission_matrix_seeded_test.go`、`project_write_locks_test.go`、`project_writes_test.go`、`project_connection_test.go`，前端的三个文案文件。每个过渡版本都在逐 Task 复现中运行过。
- **变异**：每个 Task 末尾的"变异"表列出：把代码改坏的方式、必须因此失败的测试和它所在的层（单元：假实现；存储：真实数据库；组合：`bootstrap` 组合出的 app；端到端：单独运行的故事 P6、W11）。它们在最终的原型上逐个跑过（`$M3TMP/p7atools/mutants_p7a.py`，由 `mutlevels.py` 在它写的每一层各跑一次；spec 附录 A：137 个变异，137 个被发现；只在单元一层被发现的 20 个，都是按性质的（第 3 节））；实现者可以照表抽查，改坏之后必须恢复。表中"（Task n 起）"标出的测试在较晚的 Task 才有，或才改成最终的样子：这一行的变异最迟从那个 Task 起被它发现。**安全或加锁的性质只由单元一层发现的，算缺口**（brief 的缺陷类别）；表中每一条这类性质都另有存储、组合或端到端一层的测试，例外（"按性质只在单元一层"）写在 spec 第 3 节和附录 A。
- **评审敏感**（M3 设计 12 节约束 3）：按资源寻址的共用取锁路径从 P5b 的项目成员的写里提出来（改角色、移出两个写改走它，行为不变；离开按项目寻址，照旧经 `lockAndDecide`，spec 第 3 节第 2 条），它的第一批新用户是带守卫的写：删除默认状态由语句的守卫拒绝，设为默认的第二条语句写 0 行时让事务失败，一组最后一个状态在并发下由项目行的 `FOR NO KEY UPDATE` 保证。改动这些锁、守卫、规则或测试之前，先照"变异"表确认它在所说的性质去掉之后失败。
- **提交**：提交信息用英文，末尾加一行：`Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明。

## 文件结构

路径相对于仓库根目录。"生成"表示由命令生成并提交；一个文件由多个 Task 修改时，"Task"一列都列出。

| 文件 | 职责 | Task |
|---|---|---|
| `server/internal/modules/project/app/lock.go`（修改） | `placed`、`rowWrite[R]`、`lockRowAndDecide`、`rowWrite.read`；`lock` 核对工作区的锁回答的 id；`findWorkspaceAndDecide`（Task 8） | 1、8 |
| `server/internal/modules/project/app/lock_test.go` | `TestLocksCheckEachAnswerAgainstItsKey`：共用路径对三个回答的核对，测一次（裁定 S6） | 1 |
| `server/internal/modules/project/app/member_ports.go`、`server/internal/modules/project/app/ports.go`、`server/internal/modules/project/app/update_member.go`、`server/internal/modules/project/app/remove_member.go`（修改） | `ProjectMembership.Place`；改角色、移出的端口带 `MemberFinder`；`ProjectLocks` 不再读成员关系；这两个写经 `lockRowAndDecide` | 1 |
| `server/internal/modules/project/app/update_member_test.go`、`server/internal/modules/project/app/remove_member_test.go`（修改） | 删去各自的"回答另一个 id"的行，改由共用路径的测试核对 | 1 |
| `server/internal/modules/project/app/fakes_write_test.go`（修改） | `fakeWorkspaces.answersAs`；状态的假实现接进 `writeFixture`（Task 4、5、7） | 1、4、5、7 |
| `server/internal/modules/project/domain/state.go`、`server/internal/modules/project/domain/state_test.go`（完整内容） | 状态、组、`CheckNewState`、`CheckStatePatch`、`SequenceAfter`、`CheckGroupKept`、`State.Place` | 2 |
| `server/internal/modules/project/domain/errors.go`（修改） | 四个状态码：`project.state_not_found`、`project.state_name_taken`、`project.state_last_in_group`、`project.state_default` | 2 |
| `server/internal/modules/project/adapter/postgres/queries/states.sql`（完整内容） | `CreateState`（改为 `:one`）和九条新语句 | 3 |
| `server/internal/modules/project/adapter/postgres/gen/states.sql.go`（生成） | | 3 |
| `server/internal/modules/project/adapter/postgres/states.go`；`server/internal/modules/project/adapter/postgres/rows.go`（修改） | 状态的存储方法；`CreateStates` 经 `CreateState` | 3 |
| `server/internal/modules/project/adapter/postgres/states_test.go`、`server/internal/modules/project/adapter/postgres/state_writes_test.go` | 每条语句的存储测试，每个谓词各由一行决定 | 3 |
| `server/internal/modules/project/adapter/postgres/failures_test.go`、`server/internal/modules/project/adapter/postgres/store_test.go`、`server/internal/modules/project/adapter/postgres/membership_test.go`、`server/internal/modules/project/adapter/postgres/update_test.go`（修改） | 新方法的失败测试；测试的共用（`columns`、`audit`、`changed`、`earlier`） | 3 |
| `api/modules/project.yaml`、`api/openapi.yaml`（修改） | 六个操作、`StateID`、`State`、`StateCreate`、`StateUpdate`、`StateList`、`StateGroup` | 4、6、7、8 |
| `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`server/internal/modules/project/adapter/http/gen/bodyshape.gen.go`（生成） | | 4、6、7、8（`bodyshape.gen.go`：4、6） |
| `server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`（修改） | `state.create`（Task 4）、`state.update`（Task 5）、`state.delete`、`state.mark_default`（Task 7）、`state.list`、`workspace_state.list`（Task 8） | 4、5、7、8 |
| `server/internal/modules/project/domain/actions.go`（修改） | 六个操作名 | 4、5、7、8 |
| `server/internal/modules/project/app/state_ports.go` | `StateCreator`（Task 4），`StateFinder`、`GroupCounter`、`StateUpdater`（Task 5），`StateDeleter`、`DefaultMarker`（Task 7），`StateLister`、`WorkspaceStateLister`（Task 8） | 4、5、7、8 |
| `server/internal/modules/project/app/fakes_state_test.go`；`server/internal/modules/project/app/fakes_member_test.go`（修改） | 状态的假实现（`fakeStates`）；`rowReads` 的说明 | 4、5、7、8（`fakes_member_test.go`：5） |
| `server/internal/modules/project/app/clock_test.go`（修改） | 每个状态的写在锁之后读时钟的一行 | 4、5、7 |
| `server/internal/modules/project/app/create_state.go`、`server/internal/modules/project/app/create_state_test.go` | `createState` | 4 |
| `server/internal/modules/project/app/update_state.go`、`server/internal/modules/project/app/update_state_test.go` | `updateState` | 5 |
| `server/internal/modules/project/app/delete_state.go`、`server/internal/modules/project/app/delete_state_test.go`、`server/internal/modules/project/app/mark_default_state.go`、`server/internal/modules/project/app/mark_default_state_test.go` | `deleteState`（`deletedNothing`）、`markDefaultState` | 7 |
| `server/internal/modules/project/app/list_states.go`、`server/internal/modules/project/app/list_states_test.go`、`server/internal/modules/project/app/list_workspace_states.go`、`server/internal/modules/project/app/list_workspace_states_test.go`；`server/internal/modules/project/app/list_projects.go`、`server/internal/modules/project/app/check_identifier.go`（修改） | 两个读；`listProjects`、`checkProjectIdentifier` 改用 `findWorkspaceAndDecide` | 8 |
| `server/internal/modules/project/adapter/http/states.go`、`server/internal/modules/project/adapter/http/states_test.go`、`server/internal/modules/project/adapter/http/state_writes_test.go`；`server/internal/modules/project/adapter/http/handler.go`、`server/internal/modules/project/adapter/http/handler_test.go`（修改） | 六个处理函数和它们的测试；`UseCases` 的六个字段 | 4、6、7、8 |
| `server/internal/modules/project/module.go`（修改） | 六个用例的接线；包说明 | 4、6、7、8 |
| `web/apps/web/helpers/authentication.helper.ts`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json`（修改） | 四个状态码的文案 | 4、6、7 |
| `server/internal/bootstrap/permission_matrix_states_test.go` | 状态的矩阵行（已归档项目的小表随每个操作）和它们的核对 | 4、6、7、8 |
| `server/internal/bootstrap/permission_matrix_seed_test.go`、`server/internal/bootstrap/permission_matrix_seeded_test.go`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/bootstrap/permission_matrix_columns_test.go`、`server/internal/bootstrap/permission_matrix_targets_test.go`（修改） | 每个矩阵项目的状态种子（`matrixStates`）和前提；状态的行瞄准它的列的项目下的状态 | 4、6 |
| `server/internal/bootstrap/project_write_locks_test.go`、`server/internal/bootstrap/project_writes_test.go`、`server/internal/bootstrap/project_connection_test.go`（修改） | 最先锁工作区的测试、盖戳、连接各加状态的写（`underRow`、`rowPaths`、`stateNamed`） | 4、6、7 |
| `server/internal/bootstrap/project_visibility_test.go`（修改） | `TestListingWorkspaceStatesIsListingEachProjects` | 8 |
| `server/internal/bootstrap/project_row_races_test.go`；`server/internal/bootstrap/project_membership_races_test.go`（删除） | P5b 的竞争和锁强度推广到状态的行（`rowWrites`） | 9 |
| `server/internal/bootstrap/interleaving_states_test.go`、`server/internal/bootstrap/state_rows_test.go`；`server/internal/bootstrap/membership_world_test.go`、`server/internal/bootstrap/interleaving_leaving_test.go`（修改） | 交错 10、一组最后两个状态、两个创建；每个状态的写只写它的行；`memberWorld` 的 QA、`projectLocks`、`tx` | 9 |
| `e2e/stories/project/p6-states.spec.ts`、`e2e/stories/workspace/w11-guest-bounds.spec.ts` | 故事 P6、W11 的接口版本 | 10 |
| `e2e/fixtures/api.ts`、`e2e/fixtures/assert/project.ts`、`e2e/stories/project/p5-project-members.spec.ts`、`e2e/fixtures/assert/workspace.ts`（修改） | `createState`、`answer`（P5 的提到夹具）、`State*` 类型；`expectStates`、`statesOfANewProject`；标签的 Phase 改称 P7b | 10 |
| `server/internal/modules/project/app/deletion.go`（修改） | 标签的 Phase 改称 P7b | 10 |
| `docs/v0/plane-diff.md`（修改） | 4.11 的三行（3.20 中 P7a 的行） | 10 |

---

### Task 1: 按资源寻址的共用取锁路径：`placed`、`rowWrite[R]`、`lockRowAndDecide`；P5b 按行寻址的两个写改走它；`lock` 核对工作区的回答

**Files:**
- Create: `server/internal/modules/project/app/lock_test.go`
- Modify: `server/internal/modules/project/app/fakes_write_test.go`、`server/internal/modules/project/app/lock.go`、`server/internal/modules/project/app/member_ports.go`、`server/internal/modules/project/app/ports.go`、`server/internal/modules/project/app/remove_member.go`、`server/internal/modules/project/app/remove_member_test.go`、`server/internal/modules/project/app/update_member.go`、`server/internal/modules/project/app/update_member_test.go`

**Interfaces:**
- Produces（spec 2.3；M3 设计 3.6 约定二，P5b review 第 6 节 m1、P2）：`project/app/lock.go`：
  - `type placed interface { Place() (id, workspaceID, projectID uuid.UUID) }`：按 id 寻址的项目下的一行，经它自己的端口读出时回答自己的 id、项目和项目的工作区。`app.ProjectMembership.Place`（`member_ports.go`）；状态的 `domain.State.Place` 在 Task 2。
  - `type rowWrite[R placed] struct { id uuid.UUID; action shared.Action; find func(ctx, id) (R, bool, error); targets func(R) []uuid.UUID; notFound error }`：一个写的这一行、它的操作名、读它的端口方法（不加锁、未删除的）、它改角色的账户（约定三，只有改项目成员的角色有）、这一行的 404。
  - `func lockRowAndDecide[R placed](ctx, l Locks, actor shared.Actor, rw rowWrite[R]) (held, R, error)`：`rw.read` 读这一行（它说出项目和工作区）→ `l.lock`（工作区 `FOR SHARE` → `targets` 的工作区成员关系 `FOR SHARE` → 项目 `FOR NO KEY UPDATE`）→ 锁下 `rw.read` 再读，必须仍属那个项目 → `decide`。回答锁下读到的行。没有这一行、已删除、等锁时删除的工作区或项目、不再属那个项目的行、调用者看不到的项目，都是 `rw.notFound`；规则不允许的是 Authorizer 的 403；这一行的状态拒绝什么，是用例在判定之后的事。
  - `func (rw rowWrite[R]) read(ctx) (R, error)`：`rw.find` 的回答：没有时 `rw.notFound`；端口回答的行的 id 不是问的那个时是错误（不是 404）。
  - `Locks.lock` 核对 `ShareWorkspaceByID` 回答的工作区的 id，不是 `workspaceID` 时是错误（P5b review 第 6 节 P2：共用路径上唯一没核对键的回答）。
  - `held` 不再有 `member` 字段；`lockMemberAndDecide`、`Locks.member` 删除（没有近似的副本：P5b review 第 6 节 m1）。`ProjectLocks` 不再带 `MemberFinder`；改角色、移出的端口（`MemberRoleChanger`，新的 `MemberRemover`）各自带它（按行寻址的写经自己的端口读自己的行）；`MemberEnder` 是移出和离开共用的一步，离开照旧按项目寻址（`lockAndDecide`）。
- `updateProjectMember`、`removeProjectMember`：`lockRowAndDecide(ctx, u.locks, actor, rowWrite[ProjectMembership]{… find: u.members.MemberByID, notFound: domain.ErrMemberNotFound …})`，改角色的另给 `targets`（他的工作区成员关系）。行为与 P5b 相同：调用记录、顺序、每个拒绝的码都不变。

**Tests:**
- `TestLocksCheckEachAnswerAgainstItsKey`（新，`lock_test.go`）：共用路径上的三个核对各测一次（裁定 S6），经一个三者都走到的写：bob 把 alice 在 web 的角色改为访客。acme 的锁回答另一个工作区（第 3 个调用之后停下）、这一行第一次读回答另一个 id（第 2 个之后）、锁下重读回答另一个 id（第 6 个之后）：各是写自己的错误（不是 404、不是 403），之后什么都不运行，alice 的成员关系不变。每个写都经 `lockAndDecide` 或 `lockRowAndDecide` 到 `Locks.lock`，所以这三个核对是路径的，在这里钉一次；真实的存储回答不了别的键，它们的变异按性质只在单元一层被发现（spec 第 3 节第 1 条）。
- `TestUpdateProjectMemberRefuses`、`TestRemoveProjectMemberRefuses`：删去"回答另一个 id"的行（共 3 行），说明改指向上一个测试；其余各行照旧通过：调用记录、顺序、码与 P5b 相同。

- [ ] **Step 1: 共用路径和成员的写**

`server/internal/modules/project/app/lock.go`（修改，11 处）：

````old server/internal/modules/project/app/lock.go
// the membership a write on one names); the workspace's row FOR SHARE, the
// write's first lock; the memberships of the workspace of the accounts the
// write makes members of the project, or whose role it changes, FOR SHARE
// in id order (convention 3); the project's row, FOR NO KEY UPDATE, or FOR
// SHARE for a write under the project that leaves the row and its
// memberships as they are, still undeleted and of that workspace; the
// membership a write on one names, read again, still of that project; then
// the decision, under them all. Every cascade over the workspace's projects
````
````new server/internal/modules/project/app/lock.go
// the row under it a write on one names); the workspace's row FOR SHARE,
// the write's first lock; the memberships of the workspace of the accounts
// the write makes members of the project, or whose role it changes, FOR
// SHARE in id order (convention 3); the project's row, FOR NO KEY UPDATE,
// or FOR SHARE for a write under the project that leaves the row and its
// memberships as they are, still undeleted and of that workspace; the row
// a write on one names, read again, still of that project; then the
// decision, under them all. Every cascade over the workspace's projects
````

````old server/internal/modules/project/app/lock.go
// lock read it, the caller's grant, the active roles in the workspace of
// the write's targets, by account, and, for a write on a membership, the
// membership as read under the locks.
````
````new server/internal/modules/project/app/lock.go
// lock read it, the caller's grant, and the active roles in the workspace
// of the write's targets, by account.
````

````old server/internal/modules/project/app/lock.go
	roles   map[uuid.UUID]shared.Role
	member  ProjectMembership
````
````new server/internal/modules/project/app/lock.go
	roles   map[uuid.UUID]shared.Role
````

````old server/internal/modules/project/app/lock.go
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
````
````new server/internal/modules/project/app/lock.go
// placed is a row under a project that a write names by its id, as the
// write's port reads it: Place answers the row's id, its project and the
// project's workspace.
type placed interface {
	Place() (id, workspaceID, projectID uuid.UUID)
````

````old server/internal/modules/project/app/lock.go
// member reads the undeleted membership id: domain.ErrMemberNotFound when
// there is none, and an error when the store answers another one.
func (l Locks) member(ctx context.Context, id uuid.UUID) (ProjectMembership, error) {
	m, found, err := l.projects.MemberByID(ctx, id)
````
````new server/internal/modules/project/app/lock.go
// rowWrite is a write on the row id under a project, addressed by its
// resource (M3 design 3.6 convention 2): its action, the read of its row,
// the accounts it changes the roles of, and its row's 404.
type rowWrite[R placed] struct {
	id     uuid.UUID
	action shared.Action
	// find reads the undeleted row id, without a lock, through the write's
	// own port; found is false when there is none.
	find func(ctx context.Context, id uuid.UUID) (r R, found bool, err error)
	// targets, when set, are the accounts of the row whose memberships of
	// the workspace the write locks (Locks): those whose role it changes.
	targets func(r R) []uuid.UUID
	// notFound is the 404 of the row the caller named.
	notFound error
}

// lockRowAndDecide takes the locks of rw, a write on a row under a project
// addressed by its resource, and decides its action on the row's project,
// in the order of Locks: the row, read without a lock, names its project
// and the project's workspace; then their locks, with rw's targets of the
// row; then the row read again, which must still be of that project; then
// the decision. It answers the row as read under the locks. A row that is
// not there or is deleted, a workspace or project deleted while its lock
// waited, a row no longer of the project, and a project not visible to
// actor are each rw.notFound; a role the rule does not allow is the
// Authorizer's shared.Forbidden. What the row's state refuses is the use
// case's, after the decision. A row read for another id than asked is an
// error.
func lockRowAndDecide[R placed](ctx context.Context, l Locks, actor shared.Actor, rw rowWrite[R]) (held, R, error) {
	var none R
	first, err := rw.read(ctx)
	if err != nil {
		return held{}, none, err
	}
	_, workspaceID, projectID := first.Place()
	w := write{project: projectID, action: rw.action}
	if rw.targets != nil {
		w.targets = rw.targets(first)
	}
	h, err := l.lock(ctx, workspaceID, w, rw.notFound)
	if err != nil {
		return held{}, none, err
	}
	again, err := rw.read(ctx)
	if err != nil {
		return held{}, none, err
	}
	if _, _, project := again.Place(); project != projectID {
		return held{}, none, rw.notFound
	}
	if h.grant, err = decide(ctx, l.auth, actor, rw.action, workspaceID, projectID, rw.notFound); err != nil {
		return held{}, none, err
	}
	return h, again, nil
}

// read reads rw's row: rw.notFound when there is none, and an error when
// the port answers another one.
func (rw rowWrite[R]) read(ctx context.Context) (R, error) {
	var none R
	r, found, err := rw.find(ctx, rw.id)
````

````old server/internal/modules/project/app/lock.go
		return ProjectMembership{}, err
````
````new server/internal/modules/project/app/lock.go
		return none, err
````

````old server/internal/modules/project/app/lock.go
		return ProjectMembership{}, domain.ErrMemberNotFound
	case m.ID != id:
		return ProjectMembership{}, fmt.Errorf("project membership %s read as %s", id, m.ID)
````
````new server/internal/modules/project/app/lock.go
		return none, rw.notFound
````

````old server/internal/modules/project/app/lock.go
	return m, nil
````
````new server/internal/modules/project/app/lock.go
	if id, _, _ := r.Place(); id != rw.id {
		return none, fmt.Errorf("row %s read as %s", rw.id, id)
	}
	return r, nil
````

````old server/internal/modules/project/app/lock.go
// what the caller named.
````
````new server/internal/modules/project/app/lock.go
// what the caller named. A workspace's lock answered for another workspace
// is an error.
````

````old server/internal/modules/project/app/lock.go
	_, found, err := l.workspaces.ShareWorkspaceByID(ctx, workspaceID)
````
````new server/internal/modules/project/app/lock.go
	ws, found, err := l.workspaces.ShareWorkspaceByID(ctx, workspaceID)
````

````old server/internal/modules/project/app/lock.go
	case !found:
		return held{}, notFound
````
````new server/internal/modules/project/app/lock.go
	case !found:
		return held{}, notFound
	case ws.ID != workspaceID:
		return held{}, fmt.Errorf("workspace %s locked as %s", workspaceID, ws.ID)
````

`server/internal/modules/project/app/member_ports.go`（修改，5 处）：

````old server/internal/modules/project/app/member_ports.go
}

// MemberFinder reads a project membership by its id: what a write on it
````
````new server/internal/modules/project/app/member_ports.go
}

// Place is the membership's id, its project and the project's workspace:
// where a write on it takes its locks (lockRowAndDecide).
func (m ProjectMembership) Place() (id, workspaceID, projectID uuid.UUID) {
	return m.ID, m.WorkspaceID, m.ProjectID
}

// MemberFinder reads a project membership by its id: what a write on it
````

````old server/internal/modules/project/app/member_ports.go
// their locks (Locks).
````
````new server/internal/modules/project/app/member_ports.go
// their locks (lockRowAndDecide).
````

````old server/internal/modules/project/app/member_ports.go
type MemberRoleChanger interface {
````
````new server/internal/modules/project/app/member_ports.go
type MemberRoleChanger interface {
	MemberFinder
````

````old server/internal/modules/project/app/member_ports.go
// MemberEnder is removeProjectMember's repository. It runs in the
// transaction ctx carries, under the project's FOR NO KEY UPDATE.
````
````new server/internal/modules/project/app/member_ports.go
// MemberEnder ends one project membership, for removeProjectMember and
// leaveProject. It runs in the transaction ctx carries, under the
// project's FOR NO KEY UPDATE.
````

````old server/internal/modules/project/app/member_ports.go
	EndMember(ctx context.Context, projectID, userID, by uuid.UUID, now time.Time) error
````
````new server/internal/modules/project/app/member_ports.go
	EndMember(ctx context.Context, projectID, userID, by uuid.UUID, now time.Time) error
}

// MemberRemover is removeProjectMember's repository. It runs in the
// transaction ctx carries.
type MemberRemover interface {
	MemberFinder
	MemberEnder
````

`server/internal/modules/project/app/ports.go`（修改，2 处）：

````old server/internal/modules/project/app/ports.go
// project (Locks): the project's workspace, read first without a lock, the
// project's own lock, and the membership a write on one names.
````
````new server/internal/modules/project/app/ports.go
// project (Locks): the project's workspace, read first without a lock, and
// the project's own lock.
````

````old server/internal/modules/project/app/ports.go
	ProjectSharer
	MemberFinder
````
````new server/internal/modules/project/app/ports.go
	ProjectSharer
````

`server/internal/modules/project/app/update_member.go`（修改，4 处）：

````old server/internal/modules/project/app/update_member.go
// 3.6: the membership's locks (Locks.lockMemberAndDecide: the membership
// read for its project and workspace, the workspace FOR SHARE, its member's
````
````new server/internal/modules/project/app/update_member.go
// 3.6: the membership's locks (lockRowAndDecide: the membership read for
// its project and workspace, the workspace FOR SHARE, its member's
````

````old server/internal/modules/project/app/update_member.go
		h, err := u.locks.lockMemberAndDecide(ctx, actor, id, domain.ActionMemberUpdate, true)
````
````new server/internal/modules/project/app/update_member.go
		h, m, err := lockRowAndDecide(ctx, u.locks, actor, rowWrite[ProjectMembership]{id: id, action: domain.ActionMemberUpdate,
			find: u.members.MemberByID, targets: memberOf, notFound: domain.ErrMemberNotFound})
````

````old server/internal/modules/project/app/update_member.go
		}
		m := h.member
````
````new server/internal/modules/project/app/update_member.go
		}
````

````old server/internal/modules/project/app/update_member.go
	return updated, nil
}

````
````new server/internal/modules/project/app/update_member.go
	return updated, nil
}

// memberOf is the account of m: the target of a change of his role, whose
// membership of the workspace the change locks (convention 3).
func memberOf(m ProjectMembership) []uuid.UUID {
	return []uuid.UUID{m.MemberID}
}

````

`server/internal/modules/project/app/remove_member.go`（修改，5 处）：

````old server/internal/modules/project/app/remove_member.go
	members MemberEnder
````
````new server/internal/modules/project/app/remove_member.go
	members MemberRemover
````

````old server/internal/modules/project/app/remove_member.go
func NewRemoveProjectMember(locks Locks, members MemberEnder, tx shared.TxManager, clock Clock) *RemoveProjectMember {
````
````new server/internal/modules/project/app/remove_member.go
func NewRemoveProjectMember(locks Locks, members MemberRemover, tx shared.TxManager, clock Clock) *RemoveProjectMember {
````

````old server/internal/modules/project/app/remove_member.go
// membership's locks (Locks.lockMemberAndDecide: the membership read for
// its project and workspace, the workspace FOR SHARE, the project FOR NO
// KEY UPDATE, the membership read again) and the decision on
````
````new server/internal/modules/project/app/remove_member.go
// membership's locks (lockRowAndDecide: the membership read for its
// project and workspace, the workspace FOR SHARE, the project FOR NO KEY
// UPDATE, the membership read again) and the decision on
````

````old server/internal/modules/project/app/remove_member.go
		h, err := u.locks.lockMemberAndDecide(ctx, actor, id, domain.ActionMemberRemove, false)
````
````new server/internal/modules/project/app/remove_member.go
		h, m, err := lockRowAndDecide(ctx, u.locks, actor, rowWrite[ProjectMembership]{id: id, action: domain.ActionMemberRemove,
			find: u.members.MemberByID, notFound: domain.ErrMemberNotFound})
````

````old server/internal/modules/project/app/remove_member.go
		}
		m := h.member
````
````new server/internal/modules/project/app/remove_member.go
		}
````

- [ ] **Step 2: 测试**

`server/internal/modules/project/app/fakes_write_test.go`（修改，3 处）：

````old server/internal/modules/project/app/fakes_write_test.go
// id: it logs each call, finds acme unless gone, and fails with err.
````
````new server/internal/modules/project/app/fakes_write_test.go
// id: it logs each call, finds acme unless gone, and fails with err;
// answersAs, when set, is the workspace it answers for acme.
````

````old server/internal/modules/project/app/fakes_write_test.go
	log  *callLog
	gone bool
	err  error
````
````new server/internal/modules/project/app/fakes_write_test.go
	log       *callLog
	gone      bool
	err       error
	answersAs uuid.UUID
````

````old server/internal/modules/project/app/fakes_write_test.go
		return app.Workspace{}, false, nil
````
````new server/internal/modules/project/app/fakes_write_test.go
		return app.Workspace{}, false, nil
	}
	if f.answersAs != (uuid.UUID{}) {
		return app.Workspace{ID: f.answersAs, Timezone: acme.Timezone}, true, nil
````

`server/internal/modules/project/app/lock_test.go`（新文件，44 行）：

````file server/internal/modules/project/app/lock_test.go
package app_test

import (
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The lock path every write on a project takes (lock.go) checks each answer
// it locks or reads by against what it asked for (M3 design 3.6 convention
// 2; P5b review §6): the workspace its lock answers, and the row under the
// project a write names, read before the locks and again under them. A
// mismatch is the write's own error, never a 404 or a 403, and nothing runs
// after it. Every write on a project reaches its locks through
// lockAndDecide or lockRowAndDecide, and both through Locks.lock, so the
// checks are the path's and are pinned here once, through one write that
// takes all three: bob's change of alice's role in web. A real store cannot
// answer another key: each check keeps the path consistent with its ports,
// and its mutant dies here, at the unit level, by nature.
func TestLocksCheckEachAnswerAgainstItsKey(t *testing.T) {
	for _, tt := range []struct {
		name  string
		set   func(f *writeFixture)
		calls int // how many of the write's calls ran
	}{
		{"acme's lock answered for another workspace", func(f *writeFixture) { f.workspaces.answersAs = uuid.NewV7() }, 3},
		{"the row answered for another id", func(f *writeFixture) { f.store.answersAs = uuid.NewV7() }, 2},
		{"the row read again for another id", func(f *writeFixture) { f.store.reread.id = uuid.NewV7() }, 6},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newUpdateMember()
			tt.set(f)
			id := f.memberOf(webID, alice)
			before := f.store.projects[webID].members[alice]
			_, err := uc.Execute(as(bob), id, shared.RoleGuest)
			outcome{tt.name, nil, memberLocked(id, alice, webID, bob, domain.ActionMemberUpdate, true)[:tt.calls]}.check(t, err, f)
			if after := f.store.projects[webID].members[alice]; after != before {
				t.Errorf("alice's membership after the refusal: %+v, want it as it was, %+v", after, before)
			}
		})
	}
}
````

`server/internal/modules/project/app/update_member_test.go`（修改，2 处）：

````old server/internal/modules/project/app/update_member_test.go
// workspace, or whose role is answered for another account, and a
// membership answered for another id, by either read, are the write's
// own error.
````
````new server/internal/modules/project/app/update_member_test.go
// workspace, or whose role is answered for another account, is the
// write's own error. An answer of the lock path for another key than it
// asked for is the path's to refuse (TestLocksCheckEachAnswerAgainstItsKey).
````

````old server/internal/modules/project/app/update_member_test.go
			func(f *writeFixture) { f.members.answersFor = carol }, nil, locked},
		{"a membership answered for another id", bob, alice, shared.RoleGuest, func(f *writeFixture) { f.store.answersAs = uuid.NewV7() },
			nil, upTo(2)},
		{"the membership read again for another id", bob, alice, shared.RoleGuest, func(f *writeFixture) { f.store.reread.id = uuid.NewV7() },
			nil, upTo(6)},
````
````new server/internal/modules/project/app/update_member_test.go
			func(f *writeFixture) { f.members.answersFor = carol }, nil, locked},
````

`server/internal/modules/project/app/remove_member_test.go`（修改，3 处）：

````old server/internal/modules/project/app/remove_member_test.go
// the workspace's admin too. A membership answered for another id is the
// write's own error. The row of his own membership ended while the locks
// waited pins only the order of the two checks, ended before own: on the
// wired app it cannot occur, as a caller who passes the decision has an
````
````new server/internal/modules/project/app/remove_member_test.go
// the workspace's admin too. The row of his own membership ended while the
// locks waited pins only the order of the two checks, ended before own: on
// the wired app it cannot occur, as a caller who passes the decision has an
````

````old server/internal/modules/project/app/remove_member_test.go
// same locks.
````
````new server/internal/modules/project/app/remove_member_test.go
// same locks. An answer of the lock path for another key than it asked for
// is the path's to refuse (TestLocksCheckEachAnswerAgainstItsKey).
````

````old server/internal/modules/project/app/remove_member_test.go
			func(f *writeFixture) { f.store.reread.role = shared.RoleAdmin }, domain.ErrRoleTooHigh, locked},
		{"a membership answered for another id", bob, alice, func(f *writeFixture) { f.store.answersAs = uuid.NewV7() }, nil, upTo(2)},
````
````new server/internal/modules/project/app/remove_member_test.go
			func(f *writeFixture) { f.store.reread.role = shared.RoleAdmin }, domain.ErrRoleTooHigh, locked},
````

- [ ] **Step 3: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/project/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=3 -race -run 'TestAWriteOnAProjectMembershipFindsWhatChangedMeanwhile$|TestEachLockOfAWriteOnAProjectMembershipIsItsStrength$|TestTwoProjectAdminsLeavingLeaveAnAdmin$|TestTwoAdminsDemotingEachOtherLeaveAnAdmin$' ./internal/bootstrap/`
Expected: `ok`：P5b 的竞争和锁强度在共用路径上照旧通过，输出里没有 `40P01`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/modules/project/app/fakes_write_test.go server/internal/modules/project/app/lock.go server/internal/modules/project/app/lock_test.go server/internal/modules/project/app/member_ports.go server/internal/modules/project/app/ports.go server/internal/modules/project/app/remove_member.go server/internal/modules/project/app/remove_member_test.go server/internal/modules/project/app/update_member.go server/internal/modules/project/app/update_member_test.go
```
```bash
git commit -m "refactor(M3/P7a): the writes on a row under a project share one lock path, which checks every answer against its key

lockRowAndDecide takes the locks of a write addressed by its resource
in the order of M3 design 3.6: the row read without a lock names its
project and workspace, then the workspace FOR SHARE, the targets'
memberships of it, the project FOR NO KEY UPDATE, the row read again,
still of that project, and the decision; the row's own 404 for a row
gone, moved or hidden. P5b's change of a role and removal take it, and
lockMemberAndDecide goes. The lock path now refuses a workspace its
lock answered for another id, as it refuses a row read for another;
the three checks are pinned once, at the path.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p7a.py` 的编号；"层"是它被发现的每一层，端到端是单独运行的故事）：

| 变异 | 改坏 | 必须失败的测试 | 层 |
|---|---|---|---|
| `s2-path-reread-404` | 共用路径把锁下重读的失败答成这一行的 404 | `TestDeleteStateReturnsEachFailure`（Task 7 起）、`TestLocksCheckEachAnswerAgainstItsKey`、`TestMarkDefaultStateReturnsEachFailure`（Task 7 起）、`TestRemoveProjectMemberReturnsEachFailure` 等 6 个 | 单元（按性质只在单元一层：真实的存储在这一步不失败） |
| `s2-path-lock-404` | 共用路径把工作区锁的失败答成 404 | `TestAddProjectMembersReturnsEachFailure`、`TestCreateStateReturnsEachFailure`（Task 4 起）、`TestDeleteStateReturnsEachFailure`（Task 7 起）、`TestJoinProjectReturnsEachFailure` 等 10 个 | 单元（按性质只在单元一层：真实的存储在这一步不失败） |
| `s22-workspace-key` | 共用路径不核对工作区的锁回答的 id（裁定 S6） | `TestLocksCheckEachAnswerAgainstItsKey` | 单元（按性质只在单元一层：真实的存储回答不了别的键（裁定 S6）） |
| `s22-row-id` | 共用路径不核对读到的行的 id | `TestDeleteStateChecksTheReadAfterIt`（Task 7 起）、`TestLocksCheckEachAnswerAgainstItsKey` | 单元（按性质只在单元一层：真实的存储回答不了别的键（裁定 S6）） |
| `p-no-reread` | 共用路径不在锁下重读，按第一次读到的判定 | `TestDeleteState`（Task 7 起）、`TestDeleteStateChecksTheReadAfterIt`（Task 7 起）、`TestDeleteStateRefuses`（Task 7 起）、`TestDeleteStateReturnsEachFailure`（Task 7 起） 等 18 个、`TestAWriteOnAProjectMembershipDecidesOnWhatItReadUnderItsLocks`、`TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`（Task 9 起） | 单元；组合 |
| `p-project-unchecked` | 共用路径不核对锁下重读的行仍属那个项目 | `TestDeleteStateRefuses`（Task 7 起）、`TestMarkDefaultStateRefuses`（Task 7 起）、`TestRemoveProjectMemberRefuses`、`TestUpdateProjectMemberRefuses` 等 5 个、`TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`（Task 9 起） | 单元；组合 |
| `p-decide-first` | 共用路径在取锁之前判定 | `TestDeleteState`（Task 7 起）、`TestDeleteStateChecksTheReadAfterIt`（Task 7 起）、`TestDeleteStateRefuses`（Task 7 起）、`TestDeleteStateReturnsEachFailure`（Task 7 起） 等 18 个、`TestAWriteOnAProjectMembershipDecidesOnWhatItReadUnderItsLocks`、`TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`（Task 9 起） | 单元；组合 |
| `p-reread-before-lock` | 共用路径在取锁之前重读 | `TestDeleteState`（Task 7 起）、`TestDeleteStateChecksTheReadAfterIt`（Task 7 起）、`TestDeleteStateRefuses`（Task 7 起）、`TestDeleteStateReturnsEachFailure`（Task 7 起） 等 18 个、`TestAWriteOnAProjectMembershipDecidesOnWhatItReadUnderItsLocks`、`TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`（Task 9 起） | 单元；组合 |
| `p-project-before-workspace` | 共用的锁先锁项目、再锁工作区 | `TestADeactivationAndTheProjectSidesGrowthSerialize`、`TestADeactivationFindsWhatChangedMeanwhile`、`TestADemotionAndAProjectsDeletionSerialize`、`TestADemotionAndTheProjectSidesGrowthSerialize` 等 11 个 | 组合 |
| `p-rows-share-project` | 按行寻址的写以 `FOR SHARE` 锁项目 | `TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`（Task 9 起）、`TestStateWritesOnOneProjectSerialize`（Task 9 起） | 组合 |
| `p-row-404-forbidden` | 等锁时没有了的行答 403 | `TestDeleteStateRefuses`（Task 7 起）、`TestMarkDefaultStateRefuses`（Task 7 起）、`TestRemoveProjectMemberRefuses`、`TestUpdateProjectMemberRefuses` 等 5 个、`TestAWriteOnAProjectMembershipDecidesOnWhatItReadUnderItsLocks`、`TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`（Task 9 起）、`TestPermissionMatrix`、`TestStateWritesOnOneProjectSerialize`（Task 9 起） | 单元；组合 |

---

### Task 2: 状态的规则：组和分诊、名称和颜色、`SequenceAfter`、`CheckGroupKept`、四个状态码

**Files:**
- Modify: `server/internal/modules/project/domain/errors.go`
- Modify（完整内容）: `server/internal/modules/project/domain/state.go`、`server/internal/modules/project/domain/state_test.go`

**Interfaces:**
- Produces（spec 2.4；M3 设计 3.17、5.2、5.3）：`project/domain/state.go`：
  - `type StateGroup string`；`GroupBacklog`、`GroupUnstarted`、`GroupStarted`、`GroupCompleted`、`GroupCancelled`、`GroupTriage`；`State{ID, WorkspaceID, ProjectID, Name, Description, Color, Group, Default, Sequence, CreatedAt, UpdatedAt}` 和 `State.Place()`；`NewState`（P4a 的默认状态加 `Description`）；`StateCreate{Name, Color, Group, Description}`；`StatePatch{Name, Color, Group, Description *string…; Sequence *float64}`。
  - `CheckNewState(StateCreate) error`：名称和颜色 1–255 个字符、不空白、不含 NUL；组是五个之一，`triage` 是 `not_allowed`（收集箱的状态不在这里建），别的是 `invalid_format`；说明不含 NUL。有问题的字段一次全部报告（一个 422 `validation_failed`）。名称是否已被占用由数据库判断。
  - `CheckStatePatch(StatePatch) error`：只查给了的字段，规则同上；`sequence` 可以是任何数。
  - `SequenceAfter(greatest *float64) float64`：最大值加 15000；`nil`（项目除分诊状态外没有状态）时是列的默认值 65535（Plane `State.save()`，`db/models/state.py:117-128`）。
  - `CheckGroupKept(left int) error`：一个写把状态从它的组拿走（删除或改到别的组）之后，组里还剩 `left` 个；0 时 `ErrStateLastInGroup`。
- `project/domain/errors.go`：`ErrStateNotFound`（404 `project.state_not_found`，说明"The state does not exist, cannot be changed through this API, or you cannot see its project."：按 id 指向分诊状态的项目管理员看得到项目，也成立）、`ErrStateNameTaken`（409 `project.state_name_taken`）、`ErrStateLastInGroup`（409 `project.state_last_in_group`）、`ErrStateDefault`（409 `project.state_default`）。它们的说明在 Task 4、6、7 随声明它们的操作进入契约。

**Tests:**（`state_test.go`，完整内容）
- `TestCheckNewStateReportsEveryField`（14 行：名称、颜色各为空、空白、256 个字符、含 NUL；分诊组 `not_allowed`；大写的组、空的组、别的组 `invalid_format`；说明含 NUL；全部一起，一个 422 里每个字段一个问题）、`TestCheckNewStateAcceptsValidStates`（五个组各一个；名称和颜色 1 个、255 个字符；多行的说明）、`TestCheckStatePatch`（空的、只有 `sequence`、每个字段都合法的通过；8 行各报给了的那个字段）。
- `TestSequenceAfter`（55000 之后是 70000；`nil` 是 65535）、`TestCheckGroupKept`（0 是 `ErrStateLastInGroup`，1 和更多不拒绝；逐个情形按切片的顺序）、`TestStatePlace`（三个 id 各在自己的位置）、`TestDefaultStates`（加上 `Description` 为空）。

- [ ] **Step 1: 规则和码**

`server/internal/modules/project/domain/state.go`（完整内容，188 行）：

````whole server/internal/modules/project/domain/state.go
package domain

import (
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// StateGroup is a state's group, a value of states."group" (M3 design 4.9,
// Plane's StateGroup, db/models/state.py:14-20).
type StateGroup string

// The groups. The triage group holds the intake's state only (M3 design
// 3.17).
const (
	GroupBacklog   StateGroup = "backlog"
	GroupUnstarted StateGroup = "unstarted"
	GroupStarted   StateGroup = "started"
	GroupCompleted StateGroup = "completed"
	GroupCancelled StateGroup = "cancelled"
	GroupTriage    StateGroup = "triage"
)

// stateGroups are the groups of the states the API shows and writes: every
// group but the triage group (M3 design 3.17, 5.2).
var stateGroups = []StateGroup{GroupBacklog, GroupUnstarted, GroupStarted, GroupCompleted, GroupCancelled}

// NewState is a state to create: its values (M3 design 3.17).
type NewState struct {
	Name        string
	Color       string
	Sequence    float64
	Group       StateGroup
	Default     bool
	Description string
}

// DefaultStates are the six states a new project has, in their order:
// Plane's DEFAULT_STATES (db/models/state.py:24-62), Backlog the default,
// Triage the triage state (M3 design 3.17).
func DefaultStates() []NewState {
	return []NewState{
		{Name: "Backlog", Color: "#60646C", Sequence: 15000, Group: GroupBacklog, Default: true},
		{Name: "Todo", Color: "#60646C", Sequence: 25000, Group: GroupUnstarted},
		{Name: "In Progress", Color: "#F59E0B", Sequence: 35000, Group: GroupStarted},
		{Name: "Done", Color: "#46A758", Sequence: 45000, Group: GroupCompleted},
		{Name: "Cancelled", Color: "#9AA4BC", Sequence: 55000, Group: GroupCancelled},
		{Name: "Triage", Color: "#4E5355", Sequence: 65000, Group: GroupTriage},
	}
}

// State is a state of a project as stored and as the API shows it (M3
// design 5.2). The triage state is never one: the state operations do not
// see it (3.17).
type State struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	Name        string
	Description string
	Color       string
	Group       StateGroup
	Default     bool
	Sequence    float64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Place is the state's id, its project and the project's workspace: where
// a write on it takes its locks.
func (s State) Place() (id, workspaceID, projectID uuid.UUID) {
	return s.ID, s.WorkspaceID, s.ProjectID
}

// StateCreate is what the caller asks for when creating a state (M3 design
// 5.1). Its sequence is the project's to give (SequenceAfter); a new state
// is never the default.
type StateCreate struct {
	Name        string
	Color       string
	Group       StateGroup
	Description string
}

// StatePatch is what updateState changes (M3 design 5.1): each field nil
// when it is not given, and kept. Which state is the default changes
// through markDefaultState alone.
type StatePatch struct {
	Name        *string
	Color       *string
	Group       *StateGroup
	Description *string
	Sequence    *float64
}

// maxStateText is the length of states.name and states.color,
// varchar(255), in characters.
const maxStateText = 255

// sequenceStep is the gap between a project's last state and a new one,
// and firstSequence the sequence of a state of a project that has none
// besides its triage state: Plane's State.save() (db/models/state.py:
// 117-128) and the column's default (M3 design 3.17, 4.9).
const (
	sequenceStep  = 15000
	firstSequence = 65535
)

// CheckNewState checks s (M3 design 3.17):
//   - a name and a color of 1–255 characters, not blank, without NUL,
//     which the database cannot store;
//   - a group of the five; the triage group is not_allowed: the intake's
//     state is not made here;
//   - a description without NUL.
//
// Every field with a problem is reported at once, in one 422
// validation_failed, with one problem per field. Whether the name is
// taken is the database's to say.
func CheckNewState(s StateCreate) error {
	return invalid(checkStateText("name", s.Name), checkStateText("color", s.Color), checkGroup(s.Group),
		checkText("description", s.Description))
}

// CheckStatePatch checks the fields p gives by CheckNewState's rules; a
// sequence is any number. Every field with a problem is reported at once.
func CheckStatePatch(p StatePatch) error {
	var found []*shared.FieldError
	if p.Name != nil {
		found = append(found, checkStateText("name", *p.Name))
	}
	if p.Color != nil {
		found = append(found, checkStateText("color", *p.Color))
	}
	if p.Group != nil {
		found = append(found, checkGroup(*p.Group))
	}
	if p.Description != nil {
		found = append(found, checkText("description", *p.Description))
	}
	return invalid(found...)
}

// SequenceAfter is the sequence of a new state of a project whose states
// but its triage state have greatest as their greatest sequence, nil when
// it has none: one step after it, or the column's default (Plane's
// State.save(), whose manager leaves the triage state out; M3 design
// 3.17).
func SequenceAfter(greatest *float64) float64 {
	if greatest == nil {
		return firstSequence
	}
	return *greatest + sequenceStep
}

// CheckGroupKept refuses a write that took a state from its group, by
// deleting it or moving it to another, and left the group with left
// states: every group keeps one (M3 design 3.17).
func CheckGroupKept(left int) error {
	if left == 0 {
		return ErrStateLastInGroup
	}
	return nil
}

func checkStateText(field, s string) *shared.FieldError {
	switch {
	case strings.TrimSpace(s) == "":
		return &shared.FieldError{Field: field, Code: shared.FieldTooShort, Message: "must not be empty"}
	case utf8.RuneCountInString(s) > maxStateText:
		return &shared.FieldError{Field: field, Code: shared.FieldTooLong, Message: fmt.Sprintf("must be at most %d characters", maxStateText)}
	}
	return checkText(field, s)
}

func checkGroup(g StateGroup) *shared.FieldError {
	switch {
	case g == GroupTriage:
		return &shared.FieldError{Field: "group", Code: shared.FieldNotAllowed, Message: "must not be triage: the intake's state is not made here"}
	case !slices.Contains(stateGroups, g):
		return &shared.FieldError{Field: "group", Code: shared.FieldInvalidFormat, Message: "must be backlog, unstarted, started, completed or cancelled"}
	}
	return nil
}
````

`server/internal/modules/project/domain/errors.go`（修改，1 处）：

````old server/internal/modules/project/domain/errors.go
			"other active members. It must first be given another admin, or be deleted.")
````
````new server/internal/modules/project/domain/errors.go
			"other active members. It must first be given another admin, or be deleted.")
	// ErrStateNotFound answers a state that does not exist, is deleted, is
	// the triage state, or whose project the caller does not see: the same
	// 404 for all (M3 design 3.17, 5.3, 8.2).
	ErrStateNotFound = shared.NewError(shared.KindNotFound, "project.state_not_found",
		"The state does not exist, cannot be changed through this API, or you cannot see its project.")
	// ErrStateNameTaken answers a name another undeleted state of the
	// project has, compared as written.
	ErrStateNameTaken = shared.NewError(shared.KindConflict, "project.state_name_taken", "A state of the project has this name.")
	// ErrStateDefault answers the deletion of the project's default state
	// (M3 design 3.17).
	ErrStateDefault = shared.NewError(shared.KindConflict, "project.state_default",
		"The default state cannot be deleted; make another state the default first.")
	// ErrStateLastInGroup answers the deletion of a group's only state, and
	// its move to another group: every group keeps a state (M3 design 3.17).
	ErrStateLastInGroup = shared.NewError(shared.KindConflict, "project.state_last_in_group",
		"The state is the only one of its group, and every group keeps one; add another to the group first.")
````

- [ ] **Step 2: 测试**

`server/internal/modules/project/domain/state_test.go`（完整内容，162 行）：

````whole server/internal/modules/project/domain/state_test.go
package domain

import (
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The six default states are Plane's, in its order: one per group, Backlog
// the only default, Triage the only triage state, none with a description
// (M3 design 3.17).
func TestDefaultStates(t *testing.T) {
	want := []NewState{
		{"Backlog", "#60646C", 15000, "backlog", true, ""},
		{"Todo", "#60646C", 25000, "unstarted", false, ""},
		{"In Progress", "#F59E0B", 35000, "started", false, ""},
		{"Done", "#46A758", 45000, "completed", false, ""},
		{"Cancelled", "#9AA4BC", 55000, "cancelled", false, ""},
		{"Triage", "#4E5355", 65000, "triage", false, ""},
	}
	if got := DefaultStates(); !slices.Equal(got, want) {
		t.Errorf("DefaultStates() =\n%+v\nwant\n%+v", got, want)
	}
}

// Place answers the state's own id, workspace and project, each in its
// place.
func TestStatePlace(t *testing.T) {
	s := State{ID: uuid.NewV7(), WorkspaceID: uuid.NewV7(), ProjectID: uuid.NewV7()}
	if id, workspace, project := s.Place(); id != s.ID || workspace != s.WorkspaceID || project != s.ProjectID {
		t.Errorf("Place() = %s, %s, %s; want %s, %s, %s", id, workspace, project, s.ID, s.WorkspaceID, s.ProjectID)
	}
}

// CheckNewState accepts a state of each of the five groups, a name and a
// color of one and of 255 characters, and a description of several lines.
func TestCheckNewStateAcceptsValidStates(t *testing.T) {
	for _, s := range []StateCreate{
		{Name: "Review", Color: "#F59E0B", Group: GroupStarted},
		{Name: "B", Color: "r", Group: GroupBacklog, Description: "多行\n说明"},
		{Name: strings.Repeat("状", 255), Color: strings.Repeat("c", 255), Group: GroupUnstarted},
		{Name: "Shipped", Color: "#46A758", Group: GroupCompleted},
		{Name: "Won't do", Color: "#9AA4BC", Group: GroupCancelled},
	} {
		if err := CheckNewState(s); err != nil {
			t.Errorf("CheckNewState(%+v) = %v, want nil", s, err)
		}
	}
}

// The problems CheckNewState and CheckStatePatch report: one per field,
// each field's first.
var (
	nameEmpty      = shared.FieldError{Field: "name", Code: "too_short", Message: "must not be empty"}
	nameLong       = shared.FieldError{Field: "name", Code: "too_long", Message: "must be at most 255 characters"}
	nameNUL        = shared.FieldError{Field: "name", Code: "invalid_format", Message: "must not contain a NUL character"}
	colorEmpty     = shared.FieldError{Field: "color", Code: "too_short", Message: "must not be empty"}
	colorLong      = shared.FieldError{Field: "color", Code: "too_long", Message: "must be at most 255 characters"}
	colorNUL       = shared.FieldError{Field: "color", Code: "invalid_format", Message: "must not contain a NUL character"}
	groupTriage    = shared.FieldError{Field: "group", Code: "not_allowed", Message: "must not be triage: the intake's state is not made here"}
	groupUnknown   = shared.FieldError{Field: "group", Code: "invalid_format", Message: "must be backlog, unstarted, started, completed or cancelled"}
	descriptionNUL = shared.FieldError{Field: "description", Code: "invalid_format", Message: "must not contain a NUL character"}
)

// CheckNewState reports every field with a problem at once: a name or a
// color empty, blank, of 256 characters or with NUL; the triage group
// (not_allowed) and a group of none of the six, its case another or empty
// (invalid_format); a description with NUL.
func TestCheckNewStateReportsEveryField(t *testing.T) {
	valid := StateCreate{Name: "Review", Color: "#F59E0B", Group: GroupStarted}
	with := func(change func(*StateCreate)) StateCreate {
		s := valid
		change(&s)
		return s
	}
	for _, tt := range []struct {
		name string
		s    StateCreate
		want []shared.FieldError
	}{
		{"empty name", with(func(s *StateCreate) { s.Name = "" }), []shared.FieldError{nameEmpty}},
		{"blank name", with(func(s *StateCreate) { s.Name = " \t\n" }), []shared.FieldError{nameEmpty}},
		{"name of 256 characters", with(func(s *StateCreate) { s.Name = strings.Repeat("状", 256) }), []shared.FieldError{nameLong}},
		{"name with NUL", with(func(s *StateCreate) { s.Name = "Re\x00view" }), []shared.FieldError{nameNUL}},
		{"empty color", with(func(s *StateCreate) { s.Color = "" }), []shared.FieldError{colorEmpty}},
		{"blank color", with(func(s *StateCreate) { s.Color = "  " }), []shared.FieldError{colorEmpty}},
		{"color of 256 characters", with(func(s *StateCreate) { s.Color = strings.Repeat("c", 256) }), []shared.FieldError{colorLong}},
		{"color with NUL", with(func(s *StateCreate) { s.Color = "#\x00" }), []shared.FieldError{colorNUL}},
		{"the triage group", with(func(s *StateCreate) { s.Group = GroupTriage }), []shared.FieldError{groupTriage}},
		{"a group in upper case", with(func(s *StateCreate) { s.Group = "Started" }), []shared.FieldError{groupUnknown}},
		{"an empty group", with(func(s *StateCreate) { s.Group = "" }), []shared.FieldError{groupUnknown}},
		{"another group", with(func(s *StateCreate) { s.Group = "review" }), []shared.FieldError{groupUnknown}},
		{"description with NUL", with(func(s *StateCreate) { s.Description = "a\x00" }), []shared.FieldError{descriptionNUL}},
		{"all at once", StateCreate{Name: "", Color: "\x00", Group: GroupTriage, Description: "\x00"},
			[]shared.FieldError{nameEmpty, colorNUL, groupTriage, descriptionNUL}},
	} {
		if got := fieldsOf(t, CheckNewState(tt.s)); !slices.Equal(got, tt.want) {
			t.Errorf("%s: CheckNewState() fields %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

// CheckStatePatch checks only the fields given, by CheckNewState's rules:
// an empty patch, a sequence alone (any number), and every field valid
// pass; each field given with a problem is reported, all at once.
func TestCheckStatePatch(t *testing.T) {
	for _, p := range []StatePatch{{}, {Sequence: ptr(-1.5)}, {Sequence: ptr(0.0)},
		{Name: ptr("Review"), Color: ptr("#000"), Group: ptr(GroupCancelled), Description: ptr(""), Sequence: ptr(70000.0)}} {
		if err := CheckStatePatch(p); err != nil {
			t.Errorf("CheckStatePatch(%+v) = %v, want nil", p, err)
		}
	}
	for _, tt := range []struct {
		name string
		p    StatePatch
		want []shared.FieldError
	}{
		{"empty name", StatePatch{Name: ptr("")}, []shared.FieldError{nameEmpty}},
		{"name of 256 characters", StatePatch{Name: ptr(strings.Repeat("a", 256))}, []shared.FieldError{nameLong}},
		{"blank color", StatePatch{Color: ptr(" ")}, []shared.FieldError{colorEmpty}},
		{"color of 256 characters", StatePatch{Color: ptr(strings.Repeat("c", 256))}, []shared.FieldError{colorLong}},
		{"the triage group", StatePatch{Group: ptr(GroupTriage)}, []shared.FieldError{groupTriage}},
		{"another group", StatePatch{Group: ptr(StateGroup("done"))}, []shared.FieldError{groupUnknown}},
		{"description with NUL", StatePatch{Description: ptr("\x00")}, []shared.FieldError{descriptionNUL}},
		{"all at once", StatePatch{Name: ptr("a\x00"), Color: ptr(""), Group: ptr(StateGroup("")), Description: ptr("\x00"), Sequence: ptr(1.0)},
			[]shared.FieldError{nameNUL, colorEmpty, groupUnknown, descriptionNUL}},
	} {
		if got := fieldsOf(t, CheckStatePatch(tt.p)); !slices.Equal(got, tt.want) {
			t.Errorf("%s: CheckStatePatch() fields %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

// A new state comes 15000 after the greatest sequence of its project's
// states but the triage state, 70000 after the six default ones, and at
// the column's default, 65535, when the project has none.
func TestSequenceAfter(t *testing.T) {
	for _, tt := range []struct {
		greatest *float64
		want     float64
	}{{nil, 65535}, {ptr(55000.0), 70000}, {ptr(-1.5), 14998.5}, {ptr(0.0), 15000}} {
		if got := SequenceAfter(tt.greatest); got != tt.want {
			t.Errorf("SequenceAfter(%v) = %v, want %v", tt.greatest, got, tt.want)
		}
	}
}

// A write that left a group without states is project.state_last_in_group;
// one that left it one or more is not refused.
func TestCheckGroupKept(t *testing.T) {
	for _, tt := range []struct {
		left int
		want error
	}{{0, ErrStateLastInGroup}, {1, nil}, {2, nil}} {
		if err := CheckGroupKept(tt.left); err != tt.want {
			t.Errorf("CheckGroupKept(%d) = %v, want %v", tt.left, err, tt.want)
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
git add server/internal/modules/project/domain/errors.go server/internal/modules/project/domain/state.go server/internal/modules/project/domain/state_test.go
```
```bash
git commit -m "feat(M3/P7a): the rules of a project's states

A state's name and color have 1-255 characters, not blank, without
NUL; its group is one of the five, and the triage group is not_allowed,
the intake's state being made elsewhere; every field with a problem is
reported at once. A new state comes 15000 after the greatest sequence
of its project's states but the triage state, or at 65535 when there is
none; a write that leaves a group without states is
project.state_last_in_group. The four state codes are declared.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p7a.py` 的编号；"层"是它被发现的每一层，端到端是单独运行的故事）：

| 变异 | 改坏 | 必须失败的测试 | 层 |
|---|---|---|---|
| `g-group-kept-off` | `CheckGroupKept` 让组变空 | `TestCheckGroupKept`、`TestDeleteStateRefuses`（Task 7 起）、`TestUpdateStateRefuses`（Task 5 起）、`TestPermissionMatrix`、`TestStateWritesOnOneProjectSerialize`（Task 9 起）、P6（Task 10 起） | 单元；组合；端到端 |
| `t-triage-allowed` | `group = triage` 不再是 422 | `TestCheckNewStateReportsEveryField`、`TestCheckStatePatch`、`TestCreateStateRefuses`（Task 4 起）、`TestUpdateStateRefuses`（Task 5 起）、`TestEachStateWriteChangesItsRowsAlone`（Task 9 起）、P6（Task 10 起） | 单元；组合；端到端 |
| `q-step` | 新状态在最后一个之后 10000 | `TestCreateState`（Task 4 起）、`TestCreateStateRefuses`（Task 4 起）、`TestCreateStateReturnsEachFailure`（Task 4 起）、`TestEachWriteReadsTheClockUnderItsLock` 等 5 个、`TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize`、`TestARoleChangeCapsItsMemberByHisWorkspaceRoleUnderItsLocks`、`TestAWriteOnAProjectMembershipDecidesOnWhatItReadUnderItsLocks`、`TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`（Task 9 起） 等 14 个、P6（Task 10 起） | 单元；组合；端到端 |
| `q-first` | 项目没有非分诊状态时新状态在 0 | `TestCreateState`（Task 4 起）、`TestSequenceAfter` | 单元（按性质只在单元一层：每一组至少留一个状态，项目总有非分诊状态） |
| `s21-last-wrapped` | 组的拒绝包在 403 之后 | `TestCheckGroupKept`、`TestDeleteStateRefuses`（Task 7 起）、`TestUpdateStateRefuses`（Task 5 起）、`TestPermissionMatrix`、`TestStateWritesOnOneProjectSerialize`（Task 9 起）、P6（Task 10 起） | 单元；组合；端到端 |

---

### Task 3: 状态的存储：十条语句、存储方法、存储测试和失败测试

**Files:**
- Create: `server/internal/modules/project/adapter/postgres/state_writes_test.go`、`server/internal/modules/project/adapter/postgres/states.go`、`server/internal/modules/project/adapter/postgres/states_test.go`
- Modify: `server/internal/modules/project/adapter/postgres/failures_test.go`、`server/internal/modules/project/adapter/postgres/membership_test.go`、`server/internal/modules/project/adapter/postgres/rows.go`、`server/internal/modules/project/adapter/postgres/store_test.go`、`server/internal/modules/project/adapter/postgres/update_test.go`
- Modify（完整内容）: `server/internal/modules/project/adapter/postgres/queries/states.sql`
- Generate: `server/internal/modules/project/adapter/postgres/gen/states.sql.go`

**Interfaces:**
- Produces（spec 2.5；M3 设计 3.6、3.17、6.7）：`queries/states.sql`（完整内容，spec 2.5 有全文）：`CreateState` 改为 `:one`（加 `description`，回答存下的行；`CreateStates` 经它逐行插入）；新加 `StateByID`、`ListStates`、`GreatestSequence`、`UpdateState`、`CountGroupStates`、`DeleteState`（`:execrows`，守卫 `NOT "default"`）、`ClearDefaultState`、`SetDefaultState`（`:execrows`）、`ListWorkspaceStates`。分诊状态只由 `"group" = 'triage'` 识别（3.17）。
- `postgresadapter.Store` 的方法（`states.go`），都在 `ctx` 带着的事务里执行（写的都在项目的 `FOR NO KEY UPDATE` 之下，Task 4–7）：
  - `CreateState(ctx, app.StateRow) (domain.State, error)`：同名是 `domain.ErrStateNameTaken`（只认 `states_project_id_name_key`）；别的唯一键、CHECK 是内部错误。`StateRow` 是 P4a 的（`app/ports.go`），`rows.go` 的 `CreateStates` 经它逐行插入。
  - `StateByID(ctx, id) (domain.State, bool, error)`：未删除、不是分诊状态。
  - `ListStates(ctx, projectID) ([]domain.State, error)`：未删除、不是分诊状态，按 `sequence`、再按 `id`；项目已归档时空列表（不是 `nil`）。
  - `GreatestSequence(ctx, projectID) (*float64, error)`：未删除、非分诊状态的最大值；没有时 `nil`。
  - `UpdateState(ctx, id, domain.StatePatch, by, now) (domain.State, error)`：只改给了的字段（`coalesce`），审计列改为给的账户和时刻；同名是 `ErrStateNameTaken`；已删除的、分诊状态不写，是错误（用例在锁下重读过，到不了）。
  - `CountGroupStates(ctx, projectID, group) (int, error)`：未删除的。
  - `DeleteState(ctx, id, by, now) (bool, error)`：守卫的写：未删除、不是默认、不是分诊的这一行，`deleted_at = updated_at = now`、`updated_by_id = by`；否则 `false`、什么都不写（3.17）。
  - `MarkDefaultState(ctx, projectID, id, by, now) (bool, error)`：两条语句：`ClearDefaultState`（项目未删除的默认状态不再是默认，至多一行，部分唯一索引）、`SetDefaultState`（这个项目未删除、非分诊的这一行设为默认）；第二条写 0 行时 `false`，由调用者回滚第一条。
  - `ListWorkspaceStates(ctx, workspaceID, userID) ([]domain.State, error)`：工作区未删除、未归档、他是有效成员的项目的未删除、非分诊状态，按项目、`sequence`、`id`。

**Tests:**（`states_test.go`、`state_writes_test.go`；`stateWorld`：acme 的 Web、Ops 和 beta 的 Site 各有六个默认状态，由 maker 在 `earlier` 建，测试的写都由 alice；`tableRows`、`columns` 比较被写的行之外的每一行每一列；持有另一个事务时运行的语句、`Begin` 和连接池的等待在 `pgtest.Soon(t)` 的期限上（清扫 46））
- `TestCreateState`、`TestCreateStateNameTaken`（Web 里再建 In Progress、Triage（分诊状态的名称也占用）各 409、什么都不存；小写的 in progress、已删除状态的名称、别的项目的名称可以）、`TestCreateStateBreakingAnotherConstraintIsInternal`（重复的 id、第二个默认、未知的组）。
- `TestStateByID`（每列与 `CreateState` 的回答相同；问两个时不回答另一个：按表里排在前面的一行回答的实现答错；分诊、已删除、不存在的都没有）、`TestListStates`（Review 最后加入、与 In Progress 同 `sequence`、id 更小：排在它之前、在 Done 之前，行在表里的顺序给不出这个次序，前提由 `ctid` 核对（清扫 48）；不含已删除的 Todo、别的项目的；归档的 Ops 空列表）、`TestGreatestSequence`（55000：不是 Triage 的 65000、已删除的 99999、Ops 的 80000；没有状态的、只有分诊状态的项目没有）、`TestCountGroupStates`、`TestListWorkspaceStates`（alice 的 Web、Ops；Docs 她的成员关系已结束、Arch 已归档、Gone 已删除、Lab 她唯一的成员关系已删除、Team 只有 bob、beta 的 Site 都不算；bob 只得 Team 的；同样的 `ctid` 前提）。
- `TestUpdateState`（五行：什么都不给、全部给、只给名称、只给组、只给 `sequence`；只改给了的列）、`TestUpdateStateNameTaken`（Done、Triage 各 409）、`TestUpdateStateWritesNoDeletedOrTriageState`、`TestDeleteState`（Todo 由 alice 在给的时刻删除；默认、分诊、已删除（保留时刻）、不存在的都答 `false`、什么都不写）、`TestMarkDefaultState`（Todo 成为默认、Backlog 不再是，各由 alice 在给的时刻；Ops 的默认、Web 以前是默认的已删除状态不动；Backlog 再要回来；它是默认时再设一次仍只有它）、`TestMarkDefaultStateRefuses`（已删除的、分诊的、别的项目的（两个方向）、不存在的：答 `false`，在调用者的事务里第一条语句已运行（Web 没有默认），另一个项目的照旧；回滚之后每一行不变：两条语句都在调用者的事务里）、`TestMarkDefaultStateAnswersTheFailureOfItsSecondStatement`（第一条语句之后第二条等 Todo 的行（另一个事务 `FOR UPDATE` 持有），到调用者的 `lock_timeout`：答这个失败（55P03），不答"没设"）。
- `failures_test.go`：`TestAFailedReadIsAnErrorNotAnAnswer` 加 `StateByID`、`ListStates`、`GreatestSequence`、`CountGroupStates`、`ListWorkspaceStates`；`TestAFailedWriteIsAnError` 加 `CreateState`、`UpdateState`、`DeleteState`、`MarkDefaultState`（清扫 19）。
- `membership_test.go`、`update_test.go`、`store_test.go`：P5b 的测试改用移到 `store_test.go` 的共用（`columns`、`audit`、`changed`）。

- [ ] **Step 1: 查询**

`server/internal/modules/project/adapter/postgres/queries/states.sql`（完整内容，92 行）：

````whole server/internal/modules/project/adapter/postgres/queries/states.sql
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
````

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `cff9f6556440890b99875afb8a25d070f108759cfff6524db321abe23fb6e41c` | 422 | `server/internal/modules/project/adapter/postgres/gen/states.sql.go` |

Run: `shasum -a 256 server/internal/modules/project/adapter/postgres/gen/states.sql.go`
Expected: 与上表相同。

- [ ] **Step 2: 存储方法和测试**

`server/internal/modules/project/adapter/postgres/rows.go`（修改，2 处）：

````old server/internal/modules/project/adapter/postgres/rows.go
// CreateStates inserts rows in the order given, and stops at the first
// that fails.
````
````new server/internal/modules/project/adapter/postgres/rows.go
// CreateStates inserts rows in the order given, each by CreateState, and
// stops at the first that fails.
````

````old server/internal/modules/project/adapter/postgres/rows.go
		err := s.queries(ctx).CreateState(ctx, gen.CreateStateParams{
			ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, Name: r.State.Name, Color: r.State.Color,
			Sequence: r.State.Sequence, StateGroup: string(r.State.Group), IsDefault: r.State.Default, CreatedBy: &r.CreatedBy, Now: r.Now,
		})
		if err != nil {
			return fmt.Errorf("create state %q: %w", r.State.Name, err)
````
````new server/internal/modules/project/adapter/postgres/rows.go
		if _, err := s.CreateState(ctx, r); err != nil {
			return err
````

`server/internal/modules/project/adapter/postgres/states.go`（新文件，161 行）：

````file server/internal/modules/project/adapter/postgres/states.go
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
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// stateOf is a state row as the domain has it. The queries that read a
// state each read the same columns, so their rows convert to
// gen.StateByIDRow.
func stateOf(r gen.StateByIDRow) domain.State {
	return domain.State{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, Name: r.Name, Description: r.Description, Color: r.Color,
		Group: domain.StateGroup(r.Group), Default: r.Default, Sequence: r.Sequence, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

// stateTaken is err, a write's of a state, as a name another undeleted
// state of the project has: domain.ErrStateNameTaken; any other error is
// the write's.
func stateTaken(write string, err error) error {
	if postgres.UniqueViolation(err, "states_project_id_name_key") {
		return domain.ErrStateNameTaken
	}
	return fmt.Errorf("%s: %w", write, err)
}

// CreateState inserts r and returns it as stored (app.StateCreator). A
// name another undeleted state of the project has is
// domain.ErrStateNameTaken. The domain checked every value, so a CHECK
// violation is a bug: an internal error, not a domain error.
func (s *Store) CreateState(ctx context.Context, r app.StateRow) (domain.State, error) {
	row, err := s.queries(ctx).CreateState(ctx, gen.CreateStateParams{
		ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, Name: r.State.Name, Description: r.State.Description, Color: r.State.Color,
		Sequence: r.State.Sequence, StateGroup: string(r.State.Group), IsDefault: r.State.Default, CreatedBy: &r.CreatedBy, Now: r.Now,
	})
	if err != nil {
		return domain.State{}, stateTaken(fmt.Sprintf("create state %q", r.State.Name), err)
	}
	return stateOf(gen.StateByIDRow(row)), nil
}

// StateByID is the undeleted state id, unless it is the triage state;
// found is false when there is none (app.StateFinder).
func (s *Store) StateByID(ctx context.Context, id uuid.UUID) (st domain.State, found bool, err error) {
	r, err := s.queries(ctx).StateByID(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.State{}, false, nil
	case err != nil:
		return domain.State{}, false, fmt.Errorf("read state %s: %w", id, err)
	}
	return stateOf(r), true, nil
}

// ListStates lists projectID's undeleted states but its triage state, by
// sequence, then id; none while the project is archived (app.StateLister).
func (s *Store) ListStates(ctx context.Context, projectID uuid.UUID) ([]domain.State, error) {
	rows, err := s.queries(ctx).ListStates(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list states of project %s: %w", projectID, err)
	}
	out := make([]domain.State, len(rows))
	for i, r := range rows {
		out[i] = stateOf(gen.StateByIDRow(r))
	}
	return out, nil
}

// GreatestSequence is the greatest sequence of projectID's undeleted
// states but its triage state, nil when it has none (app.StateCreator).
func (s *Store) GreatestSequence(ctx context.Context, projectID uuid.UUID) (*float64, error) {
	greatest, err := s.queries(ctx).GreatestSequence(ctx, projectID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("read the greatest sequence of project %s: %w", projectID, err)
	}
	return &greatest, nil
}

// UpdateState changes the fields p gives of the undeleted state id, by the
// account by at now, and returns it as stored (app.StateUpdater). A name
// another undeleted state of the project has is domain.ErrStateNameTaken; a
// deleted state and the triage state are an error, not written.
func (s *Store) UpdateState(ctx context.Context, id uuid.UUID, p domain.StatePatch, by uuid.UUID, now time.Time) (domain.State, error) {
	arg := gen.UpdateStateParams{ID: id, Name: p.Name, Description: p.Description, Color: p.Color, Sequence: p.Sequence, UpdatedBy: by, Now: now}
	if p.Group != nil {
		group := string(*p.Group)
		arg.StateGroup = &group
	}
	r, err := s.queries(ctx).UpdateState(ctx, arg)
	if err != nil {
		return domain.State{}, stateTaken(fmt.Sprintf("update state %s", id), err)
	}
	return stateOf(gen.StateByIDRow(r)), nil
}

// CountGroupStates is the number of projectID's undeleted states in group
// (app.GroupCounter).
func (s *Store) CountGroupStates(ctx context.Context, projectID uuid.UUID, group domain.StateGroup) (int, error) {
	n, err := s.queries(ctx).CountGroupStates(ctx, gen.CountGroupStatesParams{ProjectID: projectID, StateGroup: string(group)})
	if err != nil {
		return 0, fmt.Errorf("count the %s states of project %s: %w", group, projectID, err)
	}
	return int(n), nil
}

// DeleteState deletes the undeleted state id, unless it is its project's
// default or its triage state, by the account by at now; deleted is false
// when it did not (app.StateDeleter).
func (s *Store) DeleteState(ctx context.Context, id, by uuid.UUID, now time.Time) (deleted bool, err error) {
	n, err := s.queries(ctx).DeleteState(ctx, gen.DeleteStateParams{ID: id, DeletedBy: by, Now: now})
	if err != nil {
		return false, fmt.Errorf("delete state %s: %w", id, err)
	}
	return n == 1, nil
}

// MarkDefaultState makes the undeleted state id of projectID the project's
// default, by the account by at now, in two statements: the project's
// default state the default no longer, then this one the default
// (app.DefaultMarker). marked is false when the second wrote no row, the
// state deleted, of another project or the triage state, and the caller
// rolls the first back.
func (s *Store) MarkDefaultState(ctx context.Context, projectID, id, by uuid.UUID, now time.Time) (marked bool, err error) {
	q := s.queries(ctx)
	if err := q.ClearDefaultState(ctx, gen.ClearDefaultStateParams{ProjectID: projectID, UpdatedBy: by, Now: now}); err != nil {
		return false, fmt.Errorf("clear the default state of project %s: %w", projectID, err)
	}
	n, err := q.SetDefaultState(ctx, gen.SetDefaultStateParams{ProjectID: projectID, ID: id, UpdatedBy: by, Now: now})
	if err != nil {
		return false, fmt.Errorf("make state %s the default: %w", id, err)
	}
	return n == 1, nil
}

// ListWorkspaceStates lists the undeleted states but the triage states of
// workspaceID's undeleted, unarchived projects that userID is an active
// member of, by project, then sequence, then id
// (app.WorkspaceStateLister).
func (s *Store) ListWorkspaceStates(ctx context.Context, workspaceID, userID uuid.UUID) ([]domain.State, error) {
	rows, err := s.queries(ctx).ListWorkspaceStates(ctx, gen.ListWorkspaceStatesParams{WorkspaceID: workspaceID, UserID: userID})
	if err != nil {
		return nil, fmt.Errorf("list the states of workspace %s: %w", workspaceID, err)
	}
	out := make([]domain.State, len(rows))
	for i, r := range rows {
		out[i] = stateOf(gen.StateByIDRow(r))
	}
	return out, nil
}
````

`server/internal/modules/project/adapter/postgres/store_test.go`（修改，3 处）：

````old server/internal/modules/project/adapter/postgres/store_test.go
// tableRows is every row of table as text but the row id, in order: what
// an insert of the row id must leave as it was.
func tableRows(t *testing.T, pool *pgxpool.Pool, table string, id uuid.UUID) string {
````
````new server/internal/modules/project/adapter/postgres/store_test.go
// tableRows is every row of table as text but the rows ids, in order:
// what a write of the rows ids must leave as it was.
func tableRows(t *testing.T, pool *pgxpool.Pool, table string, ids ...uuid.UUID) string {
````

````old server/internal/modules/project/adapter/postgres/store_test.go
		" r WHERE r.id <> $1", id).Scan(&s); err != nil {
````
````new server/internal/modules/project/adapter/postgres/store_test.go
		" r WHERE r.id <> ALL (coalesce($1::uuid[], '{}'))", ids).Scan(&s); err != nil {
````

````old server/internal/modules/project/adapter/postgres/store_test.go
	return s
}

````
````new server/internal/modules/project/adapter/postgres/store_test.go
	return s
}

// rowsBut is every row of table as text, by id: the one id without the
// columns cols, which a write of it writes.
func rowsBut(t *testing.T, pool *pgxpool.Pool, table string, id uuid.UUID, cols ...string) string {
	t.Helper()
	var rows string
	if err := pool.QueryRow(context.Background(), `SELECT coalesce(string_agg(CASE WHEN r.id = $1 THEN (to_jsonb(r) - $2::text[])::text
		ELSE r::text END, E'\n' ORDER BY r.id), '') FROM `+table+` r`, id, cols).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	return rows
}

````

`server/internal/modules/project/adapter/postgres/membership_test.go`（修改，9 处）：

````old server/internal/modules/project/adapter/postgres/membership_test.go
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
````
````new server/internal/modules/project/adapter/postgres/membership_test.go
)
````

````old server/internal/modules/project/adapter/postgres/membership_test.go
			before := membershipRows(t, pool, bobs, "role", "updated_at", "updated_by_id")
````
````new server/internal/modules/project/adapter/postgres/membership_test.go
			before := rowsBut(t, pool, "project_members", bobs, "role", "updated_at", "updated_by_id")
````

````old server/internal/modules/project/adapter/postgres/membership_test.go
			if after := membershipRows(t, pool, bobs, "role", "updated_at", "updated_by_id"); after != before {
````
````new server/internal/modules/project/adapter/postgres/membership_test.go
			if after := rowsBut(t, pool, "project_members", bobs, "role", "updated_at", "updated_by_id"); after != before {
````

````old server/internal/modules/project/adapter/postgres/membership_test.go
			for name, id := range map[string]uuid.UUID{"his ended one of Docs": ended, "his deleted one of Web": deleted} {
				before := membershipRows(t, pool, uuid.UUID{})
````
````new server/internal/modules/project/adapter/postgres/membership_test.go
			for name, id := range map[string]uuid.UUID{"his ended one of Docs": ended, "his deleted one of Web": deleted} {
				before := rowsBut(t, pool, "project_members", uuid.UUID{})
````

````old server/internal/modules/project/adapter/postgres/membership_test.go
				}
				if after := membershipRows(t, pool, uuid.UUID{}); after != before {
					t.Errorf("changing %s wrote:\n%s\nwant the memberships as they were:\n%s", name, after, before)
````
````new server/internal/modules/project/adapter/postgres/membership_test.go
				}
				if after := rowsBut(t, pool, "project_members", uuid.UUID{}); after != before {
					t.Errorf("changing %s wrote:\n%s\nwant the memberships as they were:\n%s", name, after, before)
````

````old server/internal/modules/project/adapter/postgres/membership_test.go
			before := membershipRows(t, pool, bobs, "is_active", "updated_at", "updated_by_id")
````
````new server/internal/modules/project/adapter/postgres/membership_test.go
			before := rowsBut(t, pool, "project_members", bobs, "is_active", "updated_at", "updated_by_id")
````

````old server/internal/modules/project/adapter/postgres/membership_test.go
			if after := membershipRows(t, pool, bobs, "is_active", "updated_at", "updated_by_id"); after != before {
````
````new server/internal/modules/project/adapter/postgres/membership_test.go
			if after := rowsBut(t, pool, "project_members", bobs, "is_active", "updated_at", "updated_by_id"); after != before {
````

````old server/internal/modules/project/adapter/postgres/membership_test.go
			for name, project := range map[string]uuid.UUID{"Web again": web, "Docs": docs, "Gone": gone} {
				before := membershipRows(t, pool, uuid.UUID{})
````
````new server/internal/modules/project/adapter/postgres/membership_test.go
			for name, project := range map[string]uuid.UUID{"Web again": web, "Docs": docs, "Gone": gone} {
				before := rowsBut(t, pool, "project_members", uuid.UUID{})
````

````old server/internal/modules/project/adapter/postgres/membership_test.go
				}
				if after := membershipRows(t, pool, uuid.UUID{}); after != before {
					t.Errorf("ending bob's membership of %s wrote:\n%s\nwant the memberships as they were:\n%s", name, after, before)
````
````new server/internal/modules/project/adapter/postgres/membership_test.go
				}
				if after := rowsBut(t, pool, "project_members", uuid.UUID{}); after != before {
					t.Errorf("ending bob's membership of %s wrote:\n%s\nwant the memberships as they were:\n%s", name, after, before)
````

`server/internal/modules/project/adapter/postgres/update_test.go`（修改，5 处）：

````old server/internal/modules/project/adapter/postgres/update_test.go
	maps.Copy(out, change)
	return out
````
````new server/internal/modules/project/adapter/postgres/update_test.go
	maps.Copy(out, change)
	return out
}

// jsonTime is at as columns reads a timestamptz: JSON text, in UTC.
func jsonTime(at time.Time) string {
	return `"` + at.UTC().Format("2006-01-02T15:04:05.999999") + `+00:00"`
}

// audit is the audit columns a write by by at at leaves, as columns reads
// them.
func audit(by uuid.UUID, at time.Time) map[string]string {
	return map[string]string{"updated_by_id": `"` + by.String() + `"`, "updated_at": jsonTime(at)}
````

````old server/internal/modules/project/adapter/postgres/update_test.go
	others := tableRows(t, pool, "projects", web)
	audit := func(by uuid.UUID, at time.Time) map[string]string {
		return map[string]string{"updated_by_id": `"` + by.String() + `"`, "updated_at": `"` + at.Format("2006-01-02T15:04:05.999999") + `+00:00"`}
	}
````
````new server/internal/modules/project/adapter/postgres/update_test.go
	others := tableRows(t, pool, "projects", web)
````

````old server/internal/modules/project/adapter/postgres/update_test.go
	others := tableRows(t, pool, "projects", web)
	stamp := func(at time.Time) string { return `"` + at.Format("2006-01-02T15:04:05.999999") + `+00:00"` }
````
````new server/internal/modules/project/adapter/postgres/update_test.go
	others := tableRows(t, pool, "projects", web)
````

````old server/internal/modules/project/adapter/postgres/update_test.go
			archivedAt = stamp(step.at)
````
````new server/internal/modules/project/adapter/postgres/update_test.go
			archivedAt = jsonTime(step.at)
````

````old server/internal/modules/project/adapter/postgres/update_test.go
		want := changed(before, map[string]string{"archived_at": archivedAt, "updated_at": stamp(step.at), "updated_by_id": `"` + step.by.String() + `"`})
````
````new server/internal/modules/project/adapter/postgres/update_test.go
		want := changed(before, changed(audit(step.by, step.at), map[string]string{"archived_at": archivedAt}))
````

`server/internal/modules/project/adapter/postgres/states_test.go`（新文件，321 行）：

````file server/internal/modules/project/adapter/postgres/states_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// earlier is when the fixtures' states were made: no write of the tests
// is at that moment.
var earlier = now.Add(-time.Hour)

// stateWorld is acme's Web and Ops and beta's Site, each with the six
// default states, made by maker at earlier: no write of the tests is by
// maker. alice writes.
type stateWorld struct {
	s              *postgresadapter.Store
	pool           *pgxpool.Pool
	alice, maker   uuid.UUID
	acme, beta     uuid.UUID
	web, ops, site uuid.UUID
	states         map[uuid.UUID]map[string]uuid.UUID // by project, by name
	// early is an id made before Web's states, for a state added after them.
	early uuid.UUID
}

func newStateWorld(t *testing.T) stateWorld {
	t.Helper()
	s, pool := newStore(t)
	w := stateWorld{s: s, pool: pool, alice: newAccount(t, pool, "alice@corp.com"), maker: newAccount(t, pool, "maker@corp.com"),
		acme: newWorkspace(t, pool, "acme"), beta: newWorkspace(t, pool, "beta"), states: map[uuid.UUID]map[string]uuid.UUID{}}
	w.early = uuid.NewV7()
	w.web, w.ops = newProject(t, s, w.acme, "Web", "WEB", w.maker), newProject(t, s, w.acme, "Ops", "OPS", w.maker)
	w.site = newProject(t, s, w.beta, "Site", "SITE", w.maker)
	for _, p := range []struct{ workspace, project uuid.UUID }{{w.acme, w.web}, {w.acme, w.ops}, {w.beta, w.site}} {
		w.states[p.project] = seedDefaultStates(t, s, p.workspace, p.project, w.maker)
	}
	return w
}

// seedDefaultStates stores project's six default states, made by by at
// earlier, and returns their ids by name.
func seedDefaultStates(t *testing.T, s *postgresadapter.Store, workspace, project, by uuid.UUID) map[string]uuid.UUID {
	t.Helper()
	ids := map[string]uuid.UUID{}
	var rows []app.StateRow
	for _, st := range domain.DefaultStates() {
		ids[st.Name] = uuid.NewV7()
		rows = append(rows, app.StateRow{ID: ids[st.Name], WorkspaceID: workspace, ProjectID: project, State: st, CreatedBy: by, Now: earlier})
	}
	if err := s.CreateStates(context.Background(), rows); err != nil {
		t.Fatal(err)
	}
	return ids
}

// addState stores a state of w's project, made by maker at earlier, with
// the id given, and returns it as CreateState answers it.
func (w stateWorld) addState(t *testing.T, project, id uuid.UUID, st domain.NewState) domain.State {
	t.Helper()
	workspace := w.acme
	if project == w.site {
		workspace = w.beta
	}
	got, err := w.s.CreateState(context.Background(), app.StateRow{ID: id, WorkspaceID: workspace, ProjectID: project, State: st,
		CreatedBy: w.maker, Now: earlier})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// names are the names of states, in order.
func names(states []domain.State) []string {
	out := make([]string, len(states))
	for i, s := range states {
		out[i] = s.Name
	}
	return out
}

// CreateState stores the row with its values, its description too, by the
// account and at the moment given, never deleted, and answers it as
// stored; every other state keeps every column.
func TestCreateState(t *testing.T) {
	w := newStateWorld(t)
	id := uuid.NewV7()
	others := tableRows(t, w.pool, "states", id)

	got, err := w.s.CreateState(context.Background(), app.StateRow{ID: id, WorkspaceID: w.acme, ProjectID: w.web, CreatedBy: w.alice, Now: now,
		State: domain.NewState{Name: "Review", Description: "Waiting for a review", Color: "#F59E0B", Sequence: 70000, Group: domain.GroupStarted}})

	want := domain.State{ID: id, WorkspaceID: w.acme, ProjectID: w.web, Name: "Review", Description: "Waiting for a review", Color: "#F59E0B",
		Group: domain.GroupStarted, Sequence: 70000, CreatedAt: now, UpdatedAt: now}
	if err != nil || got != want {
		t.Errorf("CreateState() = %+v, %v; want %+v", got, err, want)
	}
	cols := columns(t, w.pool, "states", id)
	if a := `"` + w.alice.String() + `"`; cols["created_by_id"] != a || cols["updated_by_id"] != a || cols["deleted_at"] != "null" ||
		cols["created_at"] != jsonTime(now) || cols["updated_at"] != jsonTime(now) {
		t.Errorf("the stored row: %v; want it made and written by alice at now, undeleted", cols)
	}
	if after := tableRows(t, w.pool, "states", id); after != others {
		t.Errorf("the other states:\n%s\nwant\n%s", after, others)
	}
}

// A name is taken by another undeleted state of the same project only,
// written as it is: In Progress again in Web, and Triage, its triage
// state's, are each project.state_name_taken and store nothing; in
// progress, in another case, is free, as are a deleted state's name and a
// name of another project's state.
func TestCreateStateNameTaken(t *testing.T) {
	w := newStateWorld(t)
	exec(t, w.pool, "UPDATE states SET deleted_at = $2 WHERE id = $1", w.states[w.web]["Todo"], earlier)
	w.addState(t, w.ops, uuid.NewV7(), domain.NewState{Name: "Review", Color: "#000", Sequence: 1, Group: domain.GroupStarted})
	create := func(name string) error {
		_, err := w.s.CreateState(context.Background(), app.StateRow{ID: uuid.NewV7(), WorkspaceID: w.acme, ProjectID: w.web, CreatedBy: w.alice,
			Now: now, State: domain.NewState{Name: name, Color: "#000", Sequence: 1, Group: domain.GroupStarted}})
		return err
	}
	before := tableRows(t, w.pool, "states")
	for _, taken := range []string{"In Progress", "Triage"} {
		if err := create(taken); !errors.Is(err, domain.ErrStateNameTaken) {
			t.Errorf("%s again: %v, want project.state_name_taken", taken, err)
		}
	}
	if after := tableRows(t, w.pool, "states"); after != before {
		t.Errorf("the states after the refusal:\n%s\nwant\n%s", after, before)
	}
	for _, free := range []string{"in progress", "Todo", "Review"} {
		if err := create(free); err != nil {
			t.Errorf("%s: %v, want it free", free, err)
		}
	}
}

// Only the name's unique key is a 409: another unique key (a duplicate id,
// a second default state) and a CHECK the domain should have kept (an
// unknown group) are each an internal error, never a domain error.
func TestCreateStateBreakingAnotherConstraintIsInternal(t *testing.T) {
	w := newStateWorld(t)
	for _, tt := range []struct {
		name       string
		id         uuid.UUID
		state      domain.NewState
		constraint string
	}{
		{"a duplicate id", w.states[w.web]["Todo"], domain.NewState{Name: "Review", Color: "#000", Group: domain.GroupStarted}, "states_pkey"},
		{"a second default", uuid.NewV7(), domain.NewState{Name: "Review", Color: "#000", Group: domain.GroupStarted, Default: true},
			"states_project_id_default_key"},
		{"an unknown group", uuid.NewV7(), domain.NewState{Name: "Review", Color: "#000", Group: "review"}, "states_group_check"},
	} {
		_, err := w.s.CreateState(context.Background(), app.StateRow{ID: tt.id, WorkspaceID: w.acme, ProjectID: w.web, State: tt.state,
			CreatedBy: w.alice, Now: now})
		var se *shared.Error
		var pgErr *pgconn.PgError
		if errors.As(err, &se) || !errors.As(err, &pgErr) || pgErr.ConstraintName != tt.constraint {
			t.Errorf("%s: CreateState() = %v; want the violation of %s, not a domain error", tt.name, err, tt.constraint)
		}
	}
}

// StateByID reads the undeleted state the id names, every column as
// CreateState answered it: not another state, which a read of whichever
// row lies first would answer for one of the two asked about; not the
// triage state, nor a deleted state, nor an id of none.
func TestStateByID(t *testing.T) {
	w := newStateWorld(t)
	review := w.addState(t, w.web, uuid.NewV7(), domain.NewState{Name: "Review", Description: "d", Color: "#111", Sequence: 70000.5,
		Group: domain.GroupStarted})
	exec(t, w.pool, "UPDATE states SET deleted_at = $2 WHERE id = $1", w.states[w.web]["Todo"], earlier)
	backlog := domain.State{ID: w.states[w.ops]["Backlog"], WorkspaceID: w.acme, ProjectID: w.ops, Name: "Backlog", Color: "#60646C",
		Group: domain.GroupBacklog, Default: true, Sequence: 15000, CreatedAt: earlier, UpdatedAt: earlier}
	for _, want := range []domain.State{review, backlog} {
		if got, found, err := w.s.StateByID(context.Background(), want.ID); err != nil || !found || got != want {
			t.Errorf("StateByID(%s) = %+v, %v, %v; want %+v", want.Name, got, found, err, want)
		}
	}
	for name, id := range map[string]uuid.UUID{"Web's triage state": w.states[w.web]["Triage"], "Web's deleted Todo": w.states[w.web]["Todo"],
		"no state": uuid.NewV7()} {
		if got, found, err := w.s.StateByID(context.Background(), id); err != nil || found || got != (domain.State{}) {
			t.Errorf("StateByID() of %s = %+v, %v, %v; want none", name, got, found, err)
		}
	}
}

// ListStates lists the project's undeleted states but its triage state, by
// sequence, then id: Review, added last at In Progress's sequence with an
// id below its, comes before it and before Done, which the order the rows
// lie in does not give; not Web's deleted Todo, nor another project's
// states. An archived project has none: an empty list.
func TestListStates(t *testing.T) {
	w := newStateWorld(t)
	w.addState(t, w.web, w.early, domain.NewState{Name: "Review", Color: "#111", Sequence: 35000, Group: domain.GroupStarted})
	exec(t, w.pool, "UPDATE states SET deleted_at = $2 WHERE id = $1", w.states[w.web]["Todo"], earlier)
	var heap []string
	if err := w.pool.QueryRow(context.Background(), "SELECT array_agg(name ORDER BY ctid) FROM states WHERE project_id = $1 AND sequence = 35000",
		w.web).Scan(&heap); err != nil || !slices.Equal(heap, []string{"In Progress", "Review"}) {
		t.Fatalf("the rows of sequence 35000 lie as %q, %v; the test needs In Progress first", heap, err)
	}

	got, err := w.s.ListStates(context.Background(), w.web)
	if want := []string{"Backlog", "Review", "In Progress", "Done", "Cancelled"}; err != nil || !slices.Equal(names(got), want) {
		t.Errorf("ListStates(Web) = %q, %v; want %q", names(got), err, want)
	}
	if len(got) > 0 && got[0] != (domain.State{ID: w.states[w.web]["Backlog"], WorkspaceID: w.acme, ProjectID: w.web, Name: "Backlog",
		Color: "#60646C", Group: domain.GroupBacklog, Default: true, Sequence: 15000, CreatedAt: earlier, UpdatedAt: earlier}) {
		t.Errorf("Web's Backlog listed as %+v", got[0])
	}
	exec(t, w.pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", w.ops, now)
	if got, err := w.s.ListStates(context.Background(), w.ops); err != nil || got == nil || len(got) != 0 {
		t.Errorf("ListStates(Ops, archived) = %+v, %v; want an empty list", got, err)
	}
}

// GreatestSequence is the greatest sequence of the project's undeleted
// states but its triage state: Web's Cancelled, 55000, not its Triage,
// 65000, nor a deleted state's 99999, nor Ops's 80000. A project without
// states, and one with its triage state alone, have none.
func TestGreatestSequence(t *testing.T) {
	w := newStateWorld(t)
	w.addState(t, w.web, uuid.NewV7(), domain.NewState{Name: "Gone", Color: "#111", Sequence: 99999, Group: domain.GroupStarted})
	exec(t, w.pool, "UPDATE states SET deleted_at = $1 WHERE name = 'Gone'", earlier)
	w.addState(t, w.ops, uuid.NewV7(), domain.NewState{Name: "Late", Color: "#111", Sequence: 80000, Group: domain.GroupStarted})
	empty := newProject(t, w.s, w.acme, "Empty", "EMPTY", w.maker)
	triage := newProject(t, w.s, w.acme, "Intake", "INTAKE", w.maker)
	w.addState(t, triage, uuid.NewV7(), domain.NewState{Name: "Triage", Color: "#111", Sequence: 65000, Group: domain.GroupTriage})

	if got, err := w.s.GreatestSequence(context.Background(), w.web); err != nil || got == nil || *got != 55000 {
		t.Errorf("GreatestSequence(Web) = %s, %v; want 55000", jsonOf(t, got), err)
	}
	for name, project := range map[string]uuid.UUID{"a project without states": empty, "a project with its triage state alone": triage} {
		if got, err := w.s.GreatestSequence(context.Background(), project); err != nil || got != nil {
			t.Errorf("GreatestSequence() of %s = %s, %v; want none", name, jsonOf(t, got), err)
		}
	}
}

// CountGroupStates counts the project's undeleted states of the group:
// Web's started group holds In Progress and Review, not Web's deleted
// started state, nor Ops's In Progress, nor a state of another group.
func TestCountGroupStates(t *testing.T) {
	w := newStateWorld(t)
	w.addState(t, w.web, uuid.NewV7(), domain.NewState{Name: "Review", Color: "#111", Sequence: 1, Group: domain.GroupStarted})
	w.addState(t, w.web, uuid.NewV7(), domain.NewState{Name: "Gone", Color: "#111", Sequence: 2, Group: domain.GroupStarted})
	exec(t, w.pool, "UPDATE states SET deleted_at = $1 WHERE name = 'Gone'", earlier)
	for _, tt := range []struct {
		project uuid.UUID
		group   domain.StateGroup
		want    int
	}{{w.web, domain.GroupStarted, 2}, {w.web, domain.GroupBacklog, 1}, {w.ops, domain.GroupStarted, 1}} {
		if got, err := w.s.CountGroupStates(context.Background(), tt.project, tt.group); err != nil || got != tt.want {
			t.Errorf("CountGroupStates(%s, %s) = %d, %v; want %d", tt.project, tt.group, got, err, tt.want)
		}
	}
}

// ListWorkspaceStates lists the states but the triage states of the
// workspace's undeleted, unarchived projects alice is an active member of,
// by project, then sequence, then id: Web's and Ops's, Review, added last
// at In Progress's sequence with an id below its, before In Progress in
// Web, which the order the rows lie in does not give; not the deleted
// Todo; not the states of Docs, where her membership ended, of Arch,
// archived, of Gone, deleted though its rows were left as no deletion
// leaves them, of Lab, where her only membership is deleted, nor of Team,
// where bob is a member and she is not; nor of beta's Site. bob, whose
// only project is Team, lists Team's.
func TestListWorkspaceStates(t *testing.T) {
	w := newStateWorld(t)
	bob := newAccount(t, w.pool, "bob@corp.com")
	w.addState(t, w.web, w.early, domain.NewState{Name: "Review", Color: "#111", Sequence: 35000, Group: domain.GroupStarted})
	exec(t, w.pool, "UPDATE states SET deleted_at = $2 WHERE id = $1", w.states[w.web]["Todo"], earlier)
	var heap []string
	if err := w.pool.QueryRow(context.Background(), "SELECT array_agg(name ORDER BY ctid) FROM states WHERE project_id = $1 AND sequence = 35000",
		w.web).Scan(&heap); err != nil || !slices.Equal(heap, []string{"In Progress", "Review"}) {
		t.Fatalf("the rows of sequence 35000 lie as %q, %v; the test needs In Progress first", heap, err)
	}
	more := map[string]uuid.UUID{}
	for _, name := range []string{"Docs", "Arch", "Gone", "Lab", "Team"} {
		more[name] = newProject(t, w.s, w.acme, name, strings.ToUpper(name), w.maker)
		w.states[more[name]] = seedDefaultStates(t, w.s, w.acme, more[name], w.maker)
	}
	for _, p := range []uuid.UUID{w.web, w.ops, more["Arch"], more["Gone"]} {
		seedMember(t, w.pool, w.acme, p, w.alice, 15, true)
	}
	seedMember(t, w.pool, w.beta, w.site, w.alice, 15, true)
	seedMember(t, w.pool, w.acme, more["Docs"], w.alice, 20, false)
	seedDeleted(t, w.pool, w.acme, more["Lab"], w.alice, 20)
	seedMember(t, w.pool, w.acme, more["Team"], bob, 15, true)
	exec(t, w.pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", more["Arch"], now)
	exec(t, w.pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", more["Gone"], now)

	got, err := w.s.ListWorkspaceStates(context.Background(), w.acme, w.alice)
	want := []string{"Backlog", "Review", "In Progress", "Done", "Cancelled", "Backlog", "Todo", "In Progress", "Done", "Cancelled"}
	if err != nil || !slices.Equal(names(got), want) || len(got) != len(want) || got[0].ProjectID != w.web || got[5].ProjectID != w.ops {
		t.Errorf("ListWorkspaceStates(acme, alice) = %+v, %v; want Web's then Ops's, %q", got, err, want)
	}
	got, err = w.s.ListWorkspaceStates(context.Background(), w.acme, bob)
	if want := []string{"Backlog", "Todo", "In Progress", "Done", "Cancelled"}; err != nil || !slices.Equal(names(got), want) ||
		got[0].ProjectID != more["Team"] || got[0] != (domain.State{ID: w.states[more["Team"]]["Backlog"], WorkspaceID: w.acme,
		ProjectID: more["Team"], Name: "Backlog", Color: "#60646C", Group: domain.GroupBacklog, Default: true, Sequence: 15000,
		CreatedAt: earlier, UpdatedAt: earlier}) {
		t.Errorf("ListWorkspaceStates(acme, bob) = %+v, %v; want Team's, %q", got, err, want)
	}
	if got, err := w.s.ListWorkspaceStates(context.Background(), w.beta, bob); err != nil || got == nil || len(got) != 0 {
		t.Errorf("ListWorkspaceStates(beta, bob) = %+v, %v; want an empty list", got, err)
	}
}
````

`server/internal/modules/project/adapter/postgres/state_writes_test.go`（新文件，244 行）：

````file server/internal/modules/project/adapter/postgres/state_writes_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// UpdateState changes exactly the fields the patch gives, and the audit
// columns to the moment and the account given, and answers the state as
// stored; every other column keeps its value, and every other state every
// column. A patch that gives nothing changes the audit columns alone.
func TestUpdateState(t *testing.T) {
	w := newStateWorld(t)
	todo := w.states[w.web]["Todo"]
	others := tableRows(t, w.pool, "states", todo)
	for _, tt := range []struct {
		name   string
		patch  domain.StatePatch
		change map[string]string
	}{
		{"nothing", domain.StatePatch{}, nil},
		{"every field", domain.StatePatch{Name: ptr("Next"), Description: ptr("Up next"), Color: ptr("#123456"), Group: ptr(domain.GroupStarted),
			Sequence: ptr(-2.5)}, map[string]string{"name": `"Next"`, "description": `"Up next"`, "color": `"#123456"`, "group": `"started"`,
			"sequence": "-2.5"}},
		{"the name alone", domain.StatePatch{Name: ptr("Later")}, map[string]string{"name": `"Later"`}},
		{"the group alone", domain.StatePatch{Group: ptr(domain.GroupCancelled)}, map[string]string{"group": `"cancelled"`}},
		{"the sequence alone", domain.StatePatch{Sequence: ptr(25000.0)}, map[string]string{"sequence": "25000"}},
	} {
		before := columns(t, w.pool, "states", todo)
		got, err := w.s.UpdateState(context.Background(), todo, tt.patch, w.alice, now)
		want := changed(before, changed(audit(w.alice, now), tt.change))
		if after := columns(t, w.pool, "states", todo); err != nil || !maps.Equal(after, want) {
			t.Errorf("%s: UpdateState() = %v, the row %v\nwant %v", tt.name, err, after, want)
		}
		if stored, _, _ := w.s.StateByID(context.Background(), todo); got != stored || got.UpdatedAt != now {
			t.Errorf("%s: UpdateState() answered %+v; want the row as stored, %+v", tt.name, got, stored)
		}
	}
	if after := tableRows(t, w.pool, "states", todo); after != others {
		t.Errorf("the other states:\n%s\nwant\n%s", after, others)
	}
}

// A name another undeleted state of the project has, its triage state's
// too, is project.state_name_taken and changes nothing; the same name in
// another case, a deleted state's and another project's are free.
func TestUpdateStateNameTaken(t *testing.T) {
	w := newStateWorld(t)
	todo := w.states[w.web]["Todo"]
	exec(t, w.pool, "UPDATE states SET deleted_at = $2 WHERE id = $1", w.states[w.web]["Cancelled"], earlier)
	w.addState(t, w.ops, uuid.NewV7(), domain.NewState{Name: "Review", Color: "#000", Sequence: 1, Group: domain.GroupStarted})
	before := tableRows(t, w.pool, "states")
	for _, taken := range []string{"Done", "Triage"} {
		if _, err := w.s.UpdateState(context.Background(), todo, domain.StatePatch{Name: ptr(taken)}, w.alice, now); !errors.Is(err,
			domain.ErrStateNameTaken) {
			t.Errorf("%s: %v, want project.state_name_taken", taken, err)
		}
	}
	if after := tableRows(t, w.pool, "states"); after != before {
		t.Errorf("the states after the refusal:\n%s\nwant\n%s", after, before)
	}
	for _, free := range []string{"done", "Cancelled", "Review"} {
		if _, err := w.s.UpdateState(context.Background(), todo, domain.StatePatch{Name: ptr(free)}, w.alice, now); err != nil {
			t.Errorf("%s: %v, want it free", free, err)
		}
	}
}

// UpdateState writes neither a deleted state nor the triage state: each is
// an error, not project.state_name_taken, and every state keeps every
// column.
func TestUpdateStateWritesNoDeletedOrTriageState(t *testing.T) {
	w := newStateWorld(t)
	exec(t, w.pool, "UPDATE states SET deleted_at = $2 WHERE id = $1", w.states[w.web]["Todo"], earlier)
	before := tableRows(t, w.pool, "states")
	for name, id := range map[string]uuid.UUID{"the deleted Todo": w.states[w.web]["Todo"], "the triage state": w.states[w.web]["Triage"]} {
		if got, err := w.s.UpdateState(context.Background(), id, domain.StatePatch{Name: ptr("Other")}, w.alice, now); err == nil ||
			errors.Is(err, domain.ErrStateNameTaken) || got != (domain.State{}) {
			t.Errorf("UpdateState() of %s = %+v, %v; want an error", name, got, err)
		}
	}
	if after := tableRows(t, w.pool, "states"); after != before {
		t.Errorf("the states after the refusals:\n%s\nwant\n%s", after, before)
	}
}

// DeleteState deletes Web's Todo at the moment and by the account given,
// deleted_at and updated_at alike, and touches no other column, nor any
// other state. It deletes neither the default state, nor the triage state,
// nor a state deleted before, which keeps its moment, nor an id of none:
// each answers false and writes nothing.
func TestDeleteState(t *testing.T) {
	w := newStateWorld(t)
	todo := w.states[w.web]["Todo"]
	exec(t, w.pool, "UPDATE states SET deleted_at = $2 WHERE id = $1", w.states[w.web]["Done"], earlier)
	others := tableRows(t, w.pool, "states", todo)
	before := columns(t, w.pool, "states", todo)
	if deleted, err := w.s.DeleteState(context.Background(), todo, w.alice, now); err != nil || !deleted {
		t.Errorf("DeleteState(Todo) = %v, %v; want true", deleted, err)
	}
	if got, want := columns(t, w.pool, "states", todo), changed(before, changed(audit(w.alice, now),
		map[string]string{"deleted_at": jsonTime(now)})); !maps.Equal(got, want) {
		t.Errorf("Todo: %v\nwant %v", got, want)
	}
	if after := tableRows(t, w.pool, "states", todo); after != others {
		t.Errorf("the other states:\n%s\nwant\n%s", after, others)
	}
	all := tableRows(t, w.pool, "states")
	for name, id := range map[string]uuid.UUID{"the default Backlog": w.states[w.web]["Backlog"], "the triage state": w.states[w.web]["Triage"],
		"Done, deleted before": w.states[w.web]["Done"], "no state": uuid.NewV7()} {
		if deleted, err := w.s.DeleteState(context.Background(), id, w.alice, now.Add(1)); err != nil || deleted {
			t.Errorf("DeleteState() of %s = %v, %v; want false", name, deleted, err)
		}
	}
	if after := tableRows(t, w.pool, "states"); after != all {
		t.Errorf("the states after the refusals:\n%s\nwant\n%s", after, all)
	}
}

// MarkDefaultState makes Web's Todo its default and Backlog the default no
// longer, each stamped with the moment and the account given; no other
// state changes: Ops's default stays, as does a deleted state of Web that
// was its default. Then Backlog takes it back, and Backlog made the
// default while it is the default stays the only one.
func TestMarkDefaultState(t *testing.T) {
	w := newStateWorld(t)
	todo, backlog := w.states[w.web]["Todo"], w.states[w.web]["Backlog"]
	old := w.addState(t, w.web, uuid.NewV7(), domain.NewState{Name: "Old", Color: "#000", Sequence: 1, Group: domain.GroupStarted})
	exec(t, w.pool, `UPDATE states SET deleted_at = $2, "default" = true WHERE id = $1`, old.ID, earlier)
	others := tableRows(t, w.pool, "states", todo, backlog)
	before := map[uuid.UUID]map[string]string{todo: columns(t, w.pool, "states", todo), backlog: columns(t, w.pool, "states", backlog)}

	if marked, err := w.s.MarkDefaultState(context.Background(), w.web, todo, w.alice, now); err != nil || !marked {
		t.Errorf("MarkDefaultState(Todo) = %v, %v; want true", marked, err)
	}
	for id, isDefault := range map[uuid.UUID]string{todo: "true", backlog: "false"} {
		if got, want := columns(t, w.pool, "states", id), changed(before[id], changed(audit(w.alice, now),
			map[string]string{"default": isDefault})); !maps.Equal(got, want) {
			t.Errorf("%s: %v\nwant %v", id, got, want)
		}
	}
	if after := tableRows(t, w.pool, "states", todo, backlog); after != others {
		t.Errorf("the other states:\n%s\nwant\n%s", after, others)
	}
	for range 2 {
		if marked, err := w.s.MarkDefaultState(context.Background(), w.web, backlog, w.alice, now); err != nil || !marked {
			t.Errorf("MarkDefaultState(Backlog) = %v, %v; want true", marked, err)
		}
		var defaults []uuid.UUID
		if err := w.pool.QueryRow(context.Background(), `SELECT array_agg(id) FROM states WHERE project_id = $1 AND "default"
			AND deleted_at IS NULL`, w.web).Scan(&defaults); err != nil || len(defaults) != 1 || defaults[0] != backlog {
			t.Errorf("Web's defaults after marking Backlog: %v, %v; want Backlog alone", defaults, err)
		}
	}
}

// MarkDefaultState makes no state the default that is deleted, the triage
// state, of another project than the one given, or none: each answers
// false, after its first statement. In the caller's transaction the
// project given has no default then, its Backlog the default no longer and
// no state made it, and the other project keeps its own; the caller rolls
// back, and the states are as they were: both statements ran in its
// transaction.
func TestMarkDefaultStateRefuses(t *testing.T) {
	w := newStateWorld(t)
	exec(t, w.pool, "UPDATE states SET deleted_at = $2 WHERE id = $1", w.states[w.web]["Todo"], earlier)
	before := tableRows(t, w.pool, "states")
	rollBack := errors.New("roll back")
	for _, tt := range []struct {
		name        string
		project, id uuid.UUID
		kept        uuid.UUID // the other project's default
	}{
		{"the deleted Todo", w.web, w.states[w.web]["Todo"], w.states[w.ops]["Backlog"]},
		{"the triage state", w.web, w.states[w.web]["Triage"], w.states[w.ops]["Backlog"]},
		{"Ops's Done, for Web", w.web, w.states[w.ops]["Done"], w.states[w.ops]["Backlog"]},
		{"Web's Done, for Ops", w.ops, w.states[w.web]["Done"], w.states[w.web]["Backlog"]},
		{"no state", w.web, uuid.NewV7(), w.states[w.ops]["Backlog"]},
	} {
		var marked bool
		var defaults []uuid.UUID
		err := postgres.NewTxManager(w.pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
			var err error
			if marked, err = w.s.MarkDefaultState(ctx, tt.project, tt.id, w.alice, now); err != nil {
				return err
			}
			if err := postgres.DB(ctx, w.pool).QueryRow(ctx, `SELECT coalesce(array_agg(id), '{}') FROM states WHERE "default"
				AND deleted_at IS NULL AND project_id = ANY($1)`, []uuid.UUID{w.web, w.ops}).Scan(&defaults); err != nil {
				return err
			}
			return rollBack
		})
		if !errors.Is(err, rollBack) || marked || !slices.Equal(defaults, []uuid.UUID{tt.kept}) {
			t.Errorf("MarkDefaultState() of %s = %v, %v, the defaults of Web and Ops then %v; want false, %v alone", tt.name, marked, err,
				defaults, tt.kept)
		}
		if after := tableRows(t, w.pool, "states"); after != before {
			t.Errorf("the states after %s:\n%s\nwant\n%s", tt.name, after, before)
		}
	}
}

// MarkDefaultState answers the failure of its second statement as itself,
// never "not marked", which markDefaultState would answer as
// project.state_not_found: its first statement makes Web's Backlog the
// default no longer, and its second waits for Todo, which another
// transaction holds FOR UPDATE, until the caller's lock_timeout ends it. A
// cancelled context (TestAFailedWriteIsAnError) fails the first.
func TestMarkDefaultStateAnswersTheFailureOfItsSecondStatement(t *testing.T) {
	w := newStateWorld(t)
	todo := w.states[w.web]["Todo"]
	other, err := w.pool.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = other.Rollback(context.Background()) }()
	if _, err := other.Exec(pgtest.Soon(t), "SELECT 1 FROM states WHERE id = $1 FOR UPDATE", todo); err != nil {
		t.Fatal(err)
	}
	var marked bool
	err = postgres.NewTxManager(w.pool, 2*time.Second).WithinTx(pgtest.Soon(t), func(ctx context.Context) error {
		if _, err := postgres.DB(ctx, w.pool).Exec(ctx, "SET LOCAL lock_timeout = '200ms'"); err != nil {
			return err
		}
		var err error
		marked, err = w.s.MarkDefaultState(ctx, w.web, todo, w.alice, now)
		return err
	})
	var pgErr *pgconn.PgError
	if marked || !errors.As(err, &pgErr) || pgErr.Code != "55P03" {
		t.Errorf("MarkDefaultState(Todo) with Todo held = %v, %v; want lock_not_available (55P03)", marked, err)
	}
}
````

`server/internal/modules/project/adapter/postgres/failures_test.go`（修改，5 处）：

````old server/internal/modules/project/adapter/postgres/failures_test.go
// admin", which leaveProject would answer as project.sole_admin. Each read
// runs on a cancelled context against a project alice is the only admin of,
// beside bob, a member, and has display settings in, bob's membership of
// another project ended, so that the right answer is none of the zero
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
// admin", which leaveProject would answer as project.sole_admin; not "no
// such state", which a write on it would answer as project.state_not_found,
// nor no states, which listStates and listWorkspaceStates would answer as
// none, nor "no sequence", which createState would take for the first
// state's, nor an empty group, which a deletion or a move would answer as
// project.state_last_in_group. Each read runs on a cancelled context
// against a project alice is the only admin of, beside bob, a member, and
// has display settings in, with its six default states, bob's membership
// of another project ended, so that the right answer is none of the zero
````

````old server/internal/modules/project/adapter/postgres/failures_test.go
		t.Fatalf("CountInactive() of bob = %d, %v; want his ended membership of Ops, 1", n, err)
	}
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
		t.Fatalf("CountInactive() of bob = %d, %v; want his ended membership of Ops, 1", n, err)
	}
	todo := seedDefaultStates(t, s, acme, web, alice)["Todo"]
````

````old server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("HasOtherAdmin() = %v, %v; want context.Canceled, not an answer", other, err)
	}
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("HasOtherAdmin() = %v, %v; want context.Canceled, not an answer", other, err)
	}
	if st, found, err := s.StateByID(cancelled, todo); !failed(err) || found || st != (domain.State{}) {
		t.Errorf("StateByID() = %+v, %v, %v; want context.Canceled, not no state", st, found, err)
	}
	if list, err := s.ListStates(cancelled, web); !failed(err) || list != nil {
		t.Errorf("ListStates() = %+v, %v; want context.Canceled, not none", list, err)
	}
	if greatest, err := s.GreatestSequence(cancelled, web); !failed(err) || greatest != nil {
		t.Errorf("GreatestSequence() = %s, %v; want context.Canceled, not none", jsonOf(t, greatest), err)
	}
	if n, err := s.CountGroupStates(cancelled, web, domain.GroupUnstarted); !failed(err) || n != 0 {
		t.Errorf("CountGroupStates() = %d, %v; want context.Canceled, not a count", n, err)
	}
	if list, err := s.ListWorkspaceStates(cancelled, acme, alice); !failed(err) || list != nil {
		t.Errorf("ListWorkspaceStates() = %+v, %v; want context.Canceled, not none", list, err)
	}
````

````old server/internal/modules/project/adapter/postgres/failures_test.go
// take for done; a project's, never a taken identifier or name. Each write
// runs on a cancelled context, with values the database would take.
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
// take for done; a project's or a state's, never a taken name or
// identifier; a deletion or a marking of a state, never "not written",
// which the use case would answer as project.state_default or
// project.state_not_found. Each write runs on a cancelled context, with
// values the database would take.
````

````old server/internal/modules/project/adapter/postgres/failures_test.go
			t.Errorf("%s() = %v; want context.Canceled", step.name, err)
		}
	}
}
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
			t.Errorf("%s() = %v; want context.Canceled", step.name, err)
		}
	}
	todo := seedDefaultStates(t, s, acme, web, alice)["Todo"]
	if st, err := s.CreateState(cancelled, app.StateRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, CreatedBy: alice, Now: now,
		State: domain.NewState{Name: "Review", Color: "#000", Group: domain.GroupStarted}}); !failed(err) || errors.Is(err, domain.ErrStateNameTaken) ||
		st != (domain.State{}) {
		t.Errorf("CreateState() = %+v, %v; want context.Canceled", st, err)
	}
	if st, err := s.UpdateState(cancelled, todo, domain.StatePatch{Name: ptr("Next")}, alice, now); !failed(err) ||
		errors.Is(err, domain.ErrStateNameTaken) || st != (domain.State{}) {
		t.Errorf("UpdateState() = %+v, %v; want context.Canceled", st, err)
	}
	if deleted, err := s.DeleteState(cancelled, todo, alice, now); !failed(err) || deleted {
		t.Errorf("DeleteState() = %v, %v; want context.Canceled", deleted, err)
	}
	if marked, err := s.MarkDefaultState(cancelled, web, todo, alice, now); !failed(err) || marked {
		t.Errorf("MarkDefaultState() = %v, %v; want context.Canceled", marked, err)
	}
}
````

- [ ] **Step 3: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/project/...`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/modules/project/adapter/postgres/failures_test.go server/internal/modules/project/adapter/postgres/membership_test.go server/internal/modules/project/adapter/postgres/queries/states.sql server/internal/modules/project/adapter/postgres/rows.go server/internal/modules/project/adapter/postgres/state_writes_test.go server/internal/modules/project/adapter/postgres/states.go server/internal/modules/project/adapter/postgres/states_test.go server/internal/modules/project/adapter/postgres/store_test.go server/internal/modules/project/adapter/postgres/update_test.go server/internal/modules/project/adapter/postgres/gen/states.sql.go
```
```bash
git commit -m "feat(M3/P7a): the project store reads, creates, changes, deletes and marks a project's states

Ten statements on states: a state created and answered as stored; one
read by its id, the triage state none; a project's states by sequence,
none while it is archived; the greatest sequence but the triage
state's; a change of the fields given; the states of a group; a
guarded deletion that passes over the default; the default cleared,
then set, the second answering whether it wrote; and the states of a
workspace's unarchived projects its member is in. Each store test
decides each predicate with a row of its own, and every method's
failure comes back as an error, never a plausible answer.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p7a.py` 的编号；"层"是它被发现的每一层，端到端是单独运行的故事）：

| 变异 | 改坏 | 必须失败的测试 | 层 |
|---|---|---|---|
| `s1-cd-project` | `ClearDefaultState` 去掉 `project_id`（每个项目的默认状态都清掉） | `TestMarkDefaultState`（Task 7 起）、`TestMarkDefaultStateRefuses`（Task 7 起）、`TestEachStateWriteChangesItsRowsAlone`（Task 9 起）、`TestStateWritesOnOneProjectSerialize`（Task 9 起）、P6（Task 10 起） | 存储；组合；端到端 |
| `s1-cd-default` | `ClearDefaultState` 去掉 `"default"`（项目的每个状态都写） | `TestMarkDefaultState`（Task 7 起）、`TestEachStateWriteChangesItsRowsAlone`（Task 9 起）、P6（Task 10 起） | 存储；组合；端到端 |
| `s1-cd-deleted` | `ClearDefaultState` 去掉 `deleted_at IS NULL` | `TestMarkDefaultState`（Task 7 起） | 存储 |
| `s1-cg-project` | `CountGroupStates` 去掉 `project_id` | `TestCountGroupStates`、`TestPermissionMatrix`、`TestStateWritesOnOneProjectSerialize`（Task 9 起）、P6（Task 10 起） | 存储；组合；端到端 |
| `s1-cg-group` | `CountGroupStates` 去掉组 | `TestCountGroupStates`、`TestPermissionMatrix`、`TestStateWritesOnOneProjectSerialize`（Task 9 起）、P6（Task 10 起） | 存储；组合；端到端 |
| `s1-cg-deleted` | `CountGroupStates` 去掉 `deleted_at IS NULL` | `TestCountGroupStates`、`TestPermissionMatrix`、`TestStateWritesOnOneProjectSerialize`（Task 9 起）、P6（Task 10 起） | 存储；组合；端到端 |
| `s1-ds-id` | `DeleteState` 去掉 `id` | `TestDeleteState`（Task 7 起）、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`（Task 9 起）、`TestEachStateWriteChangesItsRowsAlone`（Task 9 起）、`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 6 起）、`TestPermissionMatrix` 等 7 个、P6（Task 10 起） | 存储；组合；端到端 |
| `s1-ds-default` | `DeleteState` 去掉 `NOT "default"`（守卫） | `TestDeleteState`（Task 7 起）、`TestPermissionMatrix`、`TestStateWritesOnOneProjectSerialize`（Task 9 起）、P6（Task 10 起） | 存储；组合；端到端 |
| `s1-ds-triage` | `DeleteState` 去掉分诊的排除 | `TestDeleteState`（Task 7 起） | 存储 |
| `s1-ds-deleted` | `DeleteState` 去掉 `deleted_at IS NULL` | `TestDeleteState`（Task 7 起） | 存储 |
| `s1-gs-project` | `GreatestSequence` 去掉 `project_id` | `TestGreatestSequence`、`TestStateWritesOnOneProjectSerialize`（Task 9 起） | 存储；组合 |
| `s1-gs-triage` | `GreatestSequence` 去掉分诊的排除 | `TestGreatestSequence`、`TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize`、`TestARoleChangeCapsItsMemberByHisWorkspaceRoleUnderItsLocks`、`TestAWriteOnAProjectMembershipDecidesOnWhatItReadUnderItsLocks`、`TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`（Task 9 起） 等 14 个、P6（Task 10 起） | 存储；组合；端到端 |
| `s1-gs-deleted` | `GreatestSequence` 去掉 `deleted_at IS NULL` | `TestGreatestSequence`、`TestStateWritesOnOneProjectSerialize`（Task 9 起） | 存储；组合 |
| `s1-gs-asc` | `GreatestSequence` 取最小的 | `TestGreatestSequence`、`TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize`、`TestARoleChangeCapsItsMemberByHisWorkspaceRoleUnderItsLocks`、`TestAWriteOnAProjectMembershipDecidesOnWhatItReadUnderItsLocks`、`TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`（Task 9 起） 等 14 个、P6（Task 10 起） | 存储；组合；端到端 |
| `s1-ls-join` | `ListStates` 的 `JOIN projects` 改成 `ON true` | `TestListStates`（Task 8 起）、`TestListingWorkspaceStatesIsListingEachProjects`（Task 8 起）、`TestPermissionMatrix`、P6（Task 10 起） | 存储；组合；端到端 |
| `s1-ls-project` | `ListStates` 去掉 `s.project_id` | `TestListStates`（Task 8 起）、`TestListingWorkspaceStatesIsListingEachProjects`（Task 8 起）、`TestPermissionMatrix`、P6（Task 10 起） | 存储；组合；端到端 |
| `s1-ls-triage` | `ListStates` 去掉分诊的排除 | `TestListStates`（Task 8 起）、`TestListingWorkspaceStatesIsListingEachProjects`（Task 8 起）、`TestPermissionMatrix`、P6（Task 10 起） | 存储；组合；端到端 |
| `s1-ls-deleted` | `ListStates` 去掉 `s.deleted_at IS NULL` | `TestListStates`（Task 8 起）、`TestListingWorkspaceStatesKeepsToItsWorkspace`（Task 8 起）、P6（Task 10 起） | 存储；组合；端到端 |
| `s1-ls-archived` | `ListStates` 去掉 `p.archived_at IS NULL` | `TestListStates`（Task 8 起）、`TestListingWorkspaceStatesIsListingEachProjects`（Task 8 起）、`TestPermissionMatrix`、P6（Task 10 起） | 存储；组合；端到端 |
| `s1-ls-order` | `ListStates` 按 id 排 | `TestListStates`（Task 8 起）、`TestListingWorkspaceStatesIsListingEachProjects`（Task 8 起）、`TestPermissionMatrix`、P6（Task 10 起） | 存储；组合；端到端 |
| `s1-lw-join-p` | `ListWorkspaceStates` 的 `JOIN projects` 不连到状态 | `TestListWorkspaceStates`（Task 8 起）、`TestListingWorkspaceStatesIsListingEachProjects`（Task 8 起）、`TestPermissionMatrix`、P6（Task 10 起） | 存储；组合；端到端 |
| `s1-lw-join-m` | `ListWorkspaceStates` 的 `JOIN project_members` 改成 `ON true` | `TestListWorkspaceStates`（Task 8 起）、`TestListingWorkspaceStatesIsListingEachProjects`（Task 8 起）、`TestPermissionMatrix`、P6（Task 10 起） | 存储；组合；端到端 |
| `s1-lw-workspace` | `ListWorkspaceStates` 去掉 `p.workspace_id` | `TestListWorkspaceStates`（Task 8 起）、`TestListingWorkspaceStatesKeepsToItsWorkspace`（Task 8 起） | 存储；组合 |
| `s1-lw-pdeleted` | `ListWorkspaceStates` 去掉 `p.deleted_at IS NULL` | `TestListWorkspaceStates`（Task 8 起） | 存储 |
| `s1-lw-archived` | `ListWorkspaceStates` 去掉 `p.archived_at IS NULL` | `TestListWorkspaceStates`（Task 8 起）、`TestListingWorkspaceStatesIsListingEachProjects`（Task 8 起）、P6（Task 10 起） | 存储；组合；端到端 |
| `s1-lw-member` | `ListWorkspaceStates` 去掉 `m.member_id` | `TestListWorkspaceStates`（Task 8 起）、`TestListingWorkspaceStatesIsListingEachProjects`（Task 8 起）、`TestPermissionMatrix`、P6（Task 10 起） | 存储；组合；端到端 |
| `s1-lw-active` | `ListWorkspaceStates` 去掉 `m.is_active` | `TestListWorkspaceStates`（Task 8 起）、`TestListingWorkspaceStatesIsListingEachProjects`（Task 8 起） | 存储；组合 |
| `s1-lw-mdeleted` | `ListWorkspaceStates` 去掉 `m.deleted_at IS NULL` | `TestListWorkspaceStates`（Task 8 起）、`TestListingWorkspaceStatesIsListingEachProjects`（Task 8 起）、`TestPermissionMatrix` | 存储；组合 |
| `s1-lw-triage` | `ListWorkspaceStates` 去掉分诊的排除 | `TestListWorkspaceStates`（Task 8 起）、`TestListingWorkspaceStatesIsListingEachProjects`（Task 8 起）、`TestPermissionMatrix`、P6（Task 10 起） | 存储；组合；端到端 |
| `s1-lw-sdeleted` | `ListWorkspaceStates` 去掉 `s.deleted_at IS NULL` | `TestListWorkspaceStates`（Task 8 起）、`TestListingWorkspaceStatesKeepsToItsWorkspace`（Task 8 起）、P6（Task 10 起） | 存储；组合；端到端 |
| `s1-lw-order` | `ListWorkspaceStates` 先按 `sequence` 排 | `TestListWorkspaceStates`（Task 8 起）、`TestListingWorkspaceStatesIsListingEachProjects`（Task 8 起）、`TestPermissionMatrix`、P6（Task 10 起） | 存储；组合；端到端 |
| `s1-sd-id` | `SetDefaultState` 去掉 `id`（项目的每个状态都设为默认） | `TestMarkDefaultState`（Task 7 起）、`TestMarkDefaultStateRefuses`（Task 7 起）、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`（Task 9 起）、`TestEachStateWriteChangesItsRowsAlone`（Task 9 起）、`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 6 起）、`TestPermissionMatrix` 等 6 个、P6（Task 10 起） | 存储；组合；端到端 |
| `s1-sd-project` | `SetDefaultState` 去掉 `project_id` | `TestMarkDefaultStateRefuses`（Task 7 起） | 存储 |
| `s1-sd-triage` | `SetDefaultState` 去掉分诊的排除 | `TestMarkDefaultStateRefuses`（Task 7 起） | 存储 |
| `s1-sd-deleted` | `SetDefaultState` 去掉 `deleted_at IS NULL` | `TestMarkDefaultStateRefuses`（Task 7 起） | 存储 |
| `s1-sb-id` | `StateByID` 去掉 `id` | `TestStateByID`、`TestUpdateState`（Task 5 起）、`TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`（Task 9 起）、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`（Task 9 起）、`TestEachStateWriteChangesItsRowsAlone`（Task 9 起）、`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 6 起） 等 7 个、P6（Task 10 起） | 存储；组合；端到端 |
| `s1-sb-triage` | `StateByID` 去掉分诊的排除 | `TestStateByID`、`TestPermissionMatrix`、P6（Task 10 起） | 存储；组合；端到端 |
| `s1-sb-deleted` | `StateByID` 去掉 `deleted_at IS NULL` | `TestStateByID`、`TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`（Task 9 起）、P6（Task 10 起） | 存储；组合；端到端 |
| `s1-us-id` | `UpdateState` 去掉 `s.id` | `TestUpdateState`（Task 5 起）、`TestUpdateStateNameTaken`、`TestUpdateStateWritesNoDeletedOrTriageState`、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`（Task 9 起）、`TestEachStateWriteChangesItsRowsAlone`（Task 9 起）、`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 6 起）、`TestPermissionMatrix` 等 7 个、P6（Task 10 起） | 存储；组合；端到端 |
| `s1-us-triage` | `UpdateState` 去掉分诊的排除 | `TestUpdateStateWritesNoDeletedOrTriageState` | 存储 |
| `s1-us-deleted` | `UpdateState` 去掉 `s.deleted_at IS NULL` | `TestUpdateStateWritesNoDeletedOrTriageState` | 存储 |
| `s3-us-created` | `UpdateState` 也写 `created_at` | `TestUpdateState`（Task 5 起） | 存储 |
| `s14-us-keeps-writer` | `UpdateState` 保留原来的写者 | `TestUpdateState`（Task 5 起）、`TestTheWritesOnAProjectStampTheirRequest`（Task 7 起）、P6（Task 10 起） | 存储；组合；端到端 |
| `s3-us-keeps-moment` | `UpdateState` 保留原来的时刻 | `TestUpdateState`（Task 5 起）、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`（Task 9 起）、`TestTheWritesOnAProjectStampTheirRequest`（Task 7 起） | 存储；组合 |
| `s14-ds-keeps-writer` | `DeleteState` 保留原来的写者 | `TestDeleteState`（Task 7 起）、`TestTheWritesOnAProjectStampTheirRequest`（Task 7 起）、P6（Task 10 起） | 存储；组合；端到端 |
| `s3-ds-keeps-moment` | `DeleteState` 保留原来的 `updated_at` | `TestDeleteState`（Task 7 起）、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`（Task 9 起）、`TestTheWritesOnAProjectStampTheirRequest`（Task 7 起）、P6（Task 10 起） | 存储；组合；端到端 |
| `s14-cd-keeps-writer` | `ClearDefaultState` 保留原来的写者 | `TestMarkDefaultState`（Task 7 起）、`TestTheWritesOnAProjectStampTheirRequest`（Task 7 起）、P6（Task 10 起） | 存储；组合；端到端 |
| `s14-sd-keeps-writer` | `SetDefaultState` 保留原来的写者 | `TestMarkDefaultState`（Task 7 起）、`TestTheWritesOnAProjectStampTheirRequest`（Task 7 起）、P6（Task 10 起） | 存储；组合；端到端 |
| `s26-us-sequence-zero` | `UpdateState` 没给的 `sequence` 写成 0 | `TestUpdateState`（Task 5 起）、`TestStateWritesOnOneProjectSerialize`（Task 9 起）、P6（Task 10 起） | 存储；组合；端到端 |
| `s26-us-group-backlog` | `UpdateState` 没给的组写成 backlog | `TestUpdateState`（Task 5 起）、`TestPermissionMatrix`、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 7 起）、P6（Task 10 起） | 存储；组合；端到端 |
| `s8-create` | `CreateState` 走池 | `TestADeactivationAndACreationHeLeadsSerialize`、`TestADeactivationDeletesTheInvitationsOfAWorkspaceItEmpties`、`TestADeactivationEndsEveryMembership`、`TestADeactivationFindsWhatChangedMeanwhile` 等 47 个 | 组合 |
| `s8-byid` | `StateByID` 走池 | `TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 7 起） | 组合 |
| `s8-greatest` | `GreatestSequence` 走池 | `TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 7 起） | 组合 |
| `s8-update` | `UpdateState` 走池 | `TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 7 起） | 组合 |
| `s8-count` | `CountGroupStates` 走池 | `TestPermissionMatrix`、`TestStateWritesOnOneProjectSerialize`（Task 9 起）、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 7 起） | 组合 |
| `s8-delete` | `DeleteState` 走池 | `TestStateWritesOnOneProjectSerialize`（Task 9 起）、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 7 起） | 组合 |
| `s8-mark` | `MarkDefaultState` 的两条语句走池 | `TestMarkDefaultStateRefuses`（Task 7 起）、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`（Task 9 起）、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 7 起） | 存储；组合 |
| `s19-create` | `CreateState` 的失败被读成回答 | `TestAFailedWriteIsAnError`、`TestCreateStateBreakingAnotherConstraintIsInternal`、`TestCreateStateNameTaken`、`TestCreateStatesStopsAtTheFirstFailure` | 存储 |
| `s19-byid` | `StateByID` 的失败被读成"没有" | `TestAFailedReadIsAnErrorNotAnAnswer` | 存储 |
| `s19-list` | `ListStates` 的失败被读成空列表 | `TestAFailedReadIsAnErrorNotAnAnswer` | 存储 |
| `s19-greatest` | `GreatestSequence` 的失败被读成"没有" | `TestAFailedReadIsAnErrorNotAnAnswer` | 存储 |
| `s19-update` | `UpdateState` 的失败被读成回答 | `TestAFailedWriteIsAnError`、`TestUpdateStateNameTaken`、`TestUpdateStateWritesNoDeletedOrTriageState` | 存储 |
| `s19-count` | `CountGroupStates` 的失败被读成 0 | `TestAFailedReadIsAnErrorNotAnAnswer` | 存储 |
| `s19-delete` | `DeleteState` 的失败被读成"没删" | `TestAFailedWriteIsAnError` | 存储 |
| `s19-clear` | `MarkDefaultState` 第一条语句的失败被读成"没设" | `TestAFailedWriteIsAnError` | 存储 |
| `s19-set` | `MarkDefaultState` 第二条语句的失败被读成"没设" | `TestMarkDefaultStateAnswersTheFailureOfItsSecondStatement` | 存储 |
| `s19-lws` | `ListWorkspaceStates` 的失败被读成空列表 | `TestAFailedReadIsAnErrorNotAnAnswer` | 存储 |
| `s19-name-taken-internal` | 同名的唯一冲突答成内部错误 | `TestCreateStateNameTaken`、`TestUpdateStateNameTaken`、`TestPermissionMatrix` | 存储；组合 |
| `g-mark-no-clear` | 设为默认不清掉原来的默认 | `TestMarkDefaultState`（Task 7 起）、`TestMarkDefaultStateRefuses`（Task 7 起）、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`（Task 9 起）、`TestEachStateWriteChangesItsRowsAlone`（Task 9 起）、`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 6 起）、`TestPermissionMatrix` 等 6 个、P6（Task 10 起） | 存储；组合；端到端 |
| `g-mark-set-first` | 设为默认先设新的、再清原来的 | `TestMarkDefaultState`（Task 7 起）、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`（Task 9 起）、`TestEachStateWriteChangesItsRowsAlone`（Task 9 起）、`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 6 起）、`TestPermissionMatrix` 等 6 个、P6（Task 10 起） | 存储；组合；端到端 |
| `g-mark-n` | `MarkDefaultState` 不看第二条语句写了几行 | `TestMarkDefaultStateRefuses`（Task 7 起） | 存储（按性质只在存储一层：组合一层到不了（O4）） |
| `g-delete-n` | `DeleteState` 不看写了几行 | `TestDeleteState`（Task 7 起）、`TestPermissionMatrix`、`TestStateWritesOnOneProjectSerialize`（Task 9 起）、P6（Task 10 起） | 存储；组合；端到端 |
| `s32-wait-on-state` | 写在第一次读时就锁住状态行（等待从项目行挪到状态行，结果不变） | `TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`（Task 9 起）、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`（Task 9 起）、`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 6 起）、`TestStateWritesOnOneProjectSerialize`（Task 9 起） | 组合 |

---

### Task 4: `createState`：契约、规则、用例、HTTP、矩阵的状态种子和行；`project.state_name_taken`

**Files:**
- Create: `server/internal/bootstrap/permission_matrix_states_test.go`、`server/internal/modules/project/adapter/http/states.go`、`server/internal/modules/project/adapter/http/states_test.go`、`server/internal/modules/project/app/create_state.go`、`server/internal/modules/project/app/create_state_test.go`、`server/internal/modules/project/app/fakes_state_test.go`、`server/internal/modules/project/app/state_ports.go`
- Modify: `api/modules/project.yaml`、`api/openapi.yaml`、`server/internal/bootstrap/permission_matrix_columns_test.go`、`server/internal/bootstrap/permission_matrix_seed_test.go`、`server/internal/bootstrap/permission_matrix_seeded_test.go`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/bootstrap/project_connection_test.go`、`server/internal/bootstrap/project_write_locks_test.go`、`server/internal/bootstrap/project_writes_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/project/adapter/http/handler.go`、`server/internal/modules/project/adapter/http/handler_test.go`、`server/internal/modules/project/app/clock_test.go`、`server/internal/modules/project/app/fakes_write_test.go`、`server/internal/modules/project/domain/actions.go`、`server/internal/modules/project/module.go`、`web/apps/web/helpers/authentication.helper.ts`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/project/adapter/http/gen/bodyshape.gen.go`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.6；M3 设计 3.4、3.6、3.17、3.19、5.1–5.3、9.2）：
  - 契约（`api/modules/project.yaml`）：`POST /api/v0/projects/{project_id}/states`（`createState`，201 `State`；`x-problem-codes: [validation_failed, project.not_found, forbidden, project.state_name_taken]`）；`State`、`StateCreate`、`StateGroup`（五个组，不含 `triage`：`group = triage` 由领域答 422 `not_allowed`，处理函数原样传给用例）。`api/openapi.yaml` 加路径的引用。
  - `access`：`state.create`（项目级，项目管理员；同时是工作区管理员的项目成员由 Authorizer 的通则得到，3.4；不给项目访客：Plane 让访客改状态，9.2）。`project/domain/actions.go`：`ActionStateCreate`。
  - `project/app/state_ports.go`：`StateCreator`（`GreatestSequence`、`CreateState`；锁经用例的 `Locks`）。它插入的 `StateRow{ID, WorkspaceID, ProjectID, CreatedBy, Now, State domain.NewState}` 是 P4a 的（`ports.go`）。
  - `CreateState`（`create_state.go`）：`NewCreateState(locks Locks, states StateCreator, tx shared.TxManager, clock Clock)`；`Execute(ctx, projectID, domain.StateCreate) (domain.State, error)`：没有调用者、领域拒绝的值，在事务之前；事务里 `lockAndDecide`（工作区 S → 项目 N → 判定 `state.create`）→ `GreatestSequence` → 时钟 → `CreateState`（`SequenceAfter`，不是默认）；回答的 id 不是插入的 id 时是错误。已归档的项目照常建（3.19）。
  - HTTP：`UseCases.CreateState`（`CreateStateUseCase`）、`CreateState` 处理函数、`state(domain.State) gen.State`（`states.go`）。
  - `project.New` 接上 `app.NewCreateState(locks, store, d.Tx, d.Clock)`；包说明照 `moddoc` 改。
  - 前端文案：`project.state_name_taken`（`PROBLEM_MESSAGES`、两份 `auth.json`）。
- 矩阵：`matrixStates`（`permission_matrix_seeded_test.go`）是每个矩阵项目的六个默认状态加 Review（started，40000）；`prepareMatrix` 经存储建出，`seeded.state(key, name)` 回答它们的 id；`seededStates`（`permission_matrix_seed_test.go`，`prepareMatrix` 在 `preconditions` 之后调用）按名称读回每个矩阵项目的状态，前提核对恰好是这些（组、`sequence`、默认、删除：gone 的随它删除），默认的只有 Backlog、分诊的只有 Triage。契约的 `StateGroup` 只说组的取值和"每一组保留一个状态"。`permission_matrix_states_test.go`：`createState` 的三行（成功、名称已占用、已归档项目的小表）。

**Tests:**
- `TestCreateState`（`create_state_test.go`，三行：web 在 Cancelled 之后 70000；归档的 ops 照常，在它的 Backlog 之后 30000；ops 的状态都没有了时 65535；调用记录、由调用者、时刻到微秒）、`TestCreateStateRefuses`（10 行：没有调用者、分诊组，在事务之前；没有项目、等锁时 acme 或 web 被删除、web 移到别的工作区、看不到 web 的调用者，各 `project.not_found`；成员 403；与 web 的 Review 同名，存储的 409，在插入之后；回答另一个 id 的状态，写自己的错误）、`TestCreateStateReturnsEachFailure`（七个端口调用的失败各原样返回，之前的调用照旧、之后什么都不运行）；`clock_test.go` 的 `TestEachWriteReadsTheClockUnderItsLock` 加 `createState`（在锁下读了最大的 `sequence` 之后读时钟）。
- HTTP：`TestCreateState`（`states_test.go`：用例收到调用者、路径的项目、请求体的字段，没给的说明为空、组原样（`triage` 也是），201 回答用例的状态）、`TestCreateStateHoldsTheBodyToItsStructure`（缺名称、颜色、组，多余的字段，类型不对：400 `bad_request`，不到用例）、`TestCreateStateRefusals`（分诊组 422、没有项目 404、项目成员 403、名称已占用 409、失败 500，各照契约声明的经 `CheckResponse`）。
- 组合：`TestPermissionMatrix` 的 `createState` 三行（`createsTheState`：QA 在列的项目里、completed、70000、不是默认；名称已占用的 409 在判定之后：不能建状态的人不知道项目有哪些状态；已归档项目的小表 PA 201、WM 403）；`TestMatrixViolationsCatchesEachColumnGap` 加一行（一个从没有种下的状态）；`TestEachWriteOnAProjectSharesItsWorkspaceFirst` 加 `createState` 的一行；`TestTheWritesOnAProjectStampTheirRequest` 加 `createState`（建出的行由 alice、在请求的时刻）；`TestTheWritesOnAProjectRunOnTheirTransactionsConnection` 加一个状态的创建。

- [ ] **Step 1: 契约**

`api/modules/project.yaml`（修改，2 处）：

````old api/modules/project.yaml
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/me/projects/{project_id}/preferences:
````
````new api/modules/project.yaml
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/projects/{project_id}/states:
    parameters:
      - $ref: '#/components/parameters/ProjectID'
    post:
      operationId: createState
      tags: [project]
      summary: Create a state in a project
      description: >-
        For the project's admins, and its members who are the workspace's
        admins; an archived project's states are created as any other's.
        The name and the color have 1–255 characters, not blank, without
        NUL; the group is one of the five, and triage is refused
        (group not_allowed): the intake's state is not made here; the
        description has no NUL (validation_failed). The values are checked
        before the project is looked at. The new state comes after the
        project's others: its sequence is the greatest of theirs, the
        intake's triage state's left out, plus 15000, or 65535 when the
        project has none. It is not the default. Its name may not be another
        undeleted state's of the project, the intake's triage state's too,
        compared as written (project.state_name_taken). A project that does
        not exist, is deleted, or that the caller does not see answers
        project.not_found; one he sees but may not change, forbidden. The
        role is decided after the workspace and project rows are locked.
      security: [{bearer: []}]
      x-problem-codes: [validation_failed, project.not_found, forbidden, project.state_name_taken]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/StateCreate'
      responses:
        '201':
          description: The new state.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/State'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/me/projects/{project_id}/preferences:
````

````old api/modules/project.yaml
      properties:
        role:
          $ref: '#/components/schemas/ProjectRole'

````
````new api/modules/project.yaml
      properties:
        role:
          $ref: '#/components/schemas/ProjectRole'
    StateGroup:
      description: >-
        The group a state is in: backlog, unstarted, started, completed or
        cancelled. Every group of a project keeps a state.
      type: string
      enum: [backlog, unstarted, started, completed, cancelled]
    State:
      description: A state of a project, in one of the groups of StateGroup.
      type: object
      additionalProperties: false
      required: [id, workspace_id, project_id, name, description, color, group, default, sequence, created_at, updated_at]
      properties:
        id:
          type: string
          format: uuid
        workspace_id:
          type: string
          format: uuid
        project_id:
          type: string
          format: uuid
        name:
          type: string
        description:
          type: string
        color:
          description: As the web app's color picker gives it, e.g. "#F59E0B".
          type: string
        group:
          $ref: '#/components/schemas/StateGroup'
        default:
          description: Whether the state is its project's default state; a project has exactly one.
          type: boolean
        sequence:
          description: The state's place among the project's states, the lowest first.
          type: number
        created_at:
          type: string
          format: date-time
        updated_at:
          type: string
          format: date-time
    StateCreate:
      type: object
      additionalProperties: false
      required: [name, color, group]
      properties:
        name:
          description: 1–255 characters, not blank; another undeleted state of the project may not have it.
          type: string
        color:
          description: 1–255 characters, not blank.
          type: string
        group:
          $ref: '#/components/schemas/StateGroup'
        description:
          type: string

````

`api/openapi.yaml`（修改，1 处）：

````old api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1project-members~1{project_member_id}'
````
````new api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1project-members~1{project_member_id}'
  /api/v0/projects/{project_id}/states:
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1projects~1{project_id}~1states'
````

Run: `make gen`
Expected: 成功（`server` 的编译要到 Step 4 处理函数写好之后）：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `1e982b1cb46816ebe228c8c22c90d22b4d9a789581e4359ea271710c40a57a2a` | 2605 | `api/dist/openapi.yaml` |
| `02c1e07e74a9da6d1692f5fdfec8c99aafaad1f75d93aa189ed27cfe62889d17` | 68 | `server/internal/modules/project/adapter/http/gen/bodyshape.gen.go` |
| `94f9c6db250279184e1c4ff32e488edfb949348e60bd5450e2c94bde77000656` | 2468 | `server/internal/modules/project/adapter/http/gen/server.gen.go` |
| `d214da7071f47867c4217386b0b8e03611420e7c1ef5a26ea9566159f5b627a8` | 2841 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml web/packages/api-client/src/schema.gen.ts server/internal/modules/project/adapter/http/gen/server.gen.go server/internal/modules/project/adapter/http/gen/bodyshape.gen.go`
Expected: 与上表相同。

- [ ] **Step 2: 规则和操作名**

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"project.leave": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
````
````new server/internal/modules/access/domain/rules.go
	"project.leave": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// The project's admins, and its members who are the workspace's admins
	// (M3 design 3.4: not its guests, whom Plane lets change states; 9.2).
	"state.create": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"project.leave": {allowed, allowed, allowed, allowed, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
````
````new server/internal/modules/access/domain/rules_test.go
	"project.leave": {allowed, allowed, allowed, allowed, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
	// As project.update: not the project's guests.
	"state.create": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
````

`server/internal/modules/project/domain/actions.go`（修改，2 处）：

````old server/internal/modules/project/domain/actions.go
	ActionLeave shared.Action = "project.leave"
````
````new server/internal/modules/project/domain/actions.go
	ActionLeave shared.Action = "project.leave"
	// ActionStateCreate is creating a state in a project: createState.
	ActionStateCreate shared.Action = "state.create"
````

````old server/internal/modules/project/domain/actions.go
		ActionMemberList, ActionMemberAdd, ActionJoin, ActionMemberUpdate, ActionMemberRemove, ActionLeave}
````
````new server/internal/modules/project/domain/actions.go
		ActionMemberList, ActionMemberAdd, ActionJoin, ActionMemberUpdate, ActionMemberRemove, ActionLeave,
		ActionStateCreate}
````

- [ ] **Step 3: 用例和它的测试**

`server/internal/modules/project/app/state_ports.go`（新文件，22 行）：

````file server/internal/modules/project/app/state_ports.go
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// The project store's states, as the state operations read and write them
// (M3 design 3.17). The triage state is never one of them.

// StateCreator is createState's repository. Each method runs in the
// transaction ctx carries, under the project's FOR NO KEY UPDATE.
type StateCreator interface {
	// GreatestSequence is the greatest sequence of projectID's undeleted
	// states but its triage state, nil when it has none.
	GreatestSequence(ctx context.Context, projectID uuid.UUID) (*float64, error)
	// CreateState inserts r and answers it as stored. A name another
	// undeleted state of the project has is domain.ErrStateNameTaken.
	CreateState(ctx context.Context, r StateRow) (domain.State, error)
}
````

`server/internal/modules/project/app/create_state.go`（新文件，67 行）：

````file server/internal/modules/project/app/create_state.go
package app

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CreateState creates a state in a project: POST
// /api/v0/projects/{project_id}/states (M3 design 3.17).
type CreateState struct {
	locks  Locks
	states StateCreator
	tx     shared.TxManager
	clock  Clock
}

// NewCreateState returns the use case.
func NewCreateState(locks Locks, states StateCreator, tx shared.TxManager, clock Clock) *CreateState {
	return &CreateState{locks: locks, states: states, tx: tx, clock: clock}
}

// Execute checks in (domain.CheckNewState), then in one transaction, in the
// order of M3 design 3.6: the project's locks (its workspace FOR SHARE, the
// project FOR NO KEY UPDATE, which every write of its states takes) and the
// decision on state.create; an archived project's states are created as
// any other's (3.19). Then the greatest sequence of the project's states,
// the clock under the locks, and the state, after them all
// (domain.SequenceAfter), never the default, by the caller at that time.
// The answer is the state as stored.
func (u *CreateState) Execute(ctx context.Context, projectID uuid.UUID, in domain.StateCreate) (domain.State, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.State{}, err
	}
	if err := domain.CheckNewState(in); err != nil {
		return domain.State{}, err
	}
	var created domain.State
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		h, err := u.locks.lockAndDecide(ctx, actor, write{project: projectID, action: domain.ActionStateCreate})
		if err != nil {
			return err
		}
		greatest, err := u.states.GreatestSequence(ctx, projectID)
		if err != nil {
			return err
		}
		row := StateRow{ID: uuid.NewV7(), WorkspaceID: h.project.WorkspaceID, ProjectID: projectID, CreatedBy: actor.UserID, Now: u.clock.Now(),
			State: domain.NewState{Name: in.Name, Color: in.Color, Sequence: domain.SequenceAfter(greatest), Group: in.Group,
				Description: in.Description}}
		if created, err = u.states.CreateState(ctx, row); err != nil {
			return err
		}
		if created.ID != row.ID {
			return fmt.Errorf("state %s created as %s", row.ID, created.ID)
		}
		return nil
	})
	if err != nil {
		return domain.State{}, err
	}
	return created, nil
}
````

`server/internal/modules/project/app/fakes_state_test.go`（新文件，109 行）：

````file server/internal/modules/project/app/fakes_state_test.go
package app_test

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// stateSince is when the fakes' states were made, as stored: no clock's
// time.
var stateSince = time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)

// The states of web and ops, by id, the same in every fixture: made in
// this order, so their ids are too.
var webBacklog, webTodo, webStarted, webReview, webDone, webCancelled, opsBacklog = uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(),
	uuid.NewV7(), uuid.NewV7(), uuid.NewV7()

// fakeStates is the project store's states as the state operations' tests
// hold them, by id, beside fakeStore's projects, whose log, failures and
// changedAs it shares; the triage states are none of them, as the store
// reads none.
type fakeStates struct {
	*fakeStore
	states map[uuid.UUID]domain.State
}

// newStates is newWrites with web's states, Backlog its default, In
// Progress and Review the started group's, each other group's one, and
// ops's Backlog, its default.
func newStates() (*writeFixture, *fakeStates) {
	f := newWrites()
	s := &fakeStates{fakeStore: f.store, states: map[uuid.UUID]domain.State{}}
	for _, st := range []domain.State{
		{ID: webBacklog, ProjectID: webID, Name: "Backlog", Group: domain.GroupBacklog, Default: true, Sequence: 15000},
		{ID: webTodo, ProjectID: webID, Name: "Todo", Group: domain.GroupUnstarted, Sequence: 25000},
		{ID: webStarted, ProjectID: webID, Name: "In Progress", Group: domain.GroupStarted, Sequence: 35000},
		{ID: webReview, ProjectID: webID, Name: "Review", Group: domain.GroupStarted, Sequence: 40000},
		{ID: webDone, ProjectID: webID, Name: "Done", Group: domain.GroupCompleted, Sequence: 45000},
		{ID: webCancelled, ProjectID: webID, Name: "Cancelled", Group: domain.GroupCancelled, Sequence: 55000},
		{ID: opsBacklog, ProjectID: opsID, Name: "Backlog", Group: domain.GroupBacklog, Default: true, Sequence: 15000},
	} {
		st.WorkspaceID, st.Color, st.CreatedAt, st.UpdatedAt = acme.ID, "#60646C", stateSince, stateSince
		s.states[st.ID] = st
	}
	return f, s
}

// of are the states of project, in the store's order: by name, which
// neither a sort by sequence, either way, nor one by id gives, so that a
// use case that sorted them would answer another.
func (f *fakeStates) of(project uuid.UUID) []domain.State {
	var out []domain.State
	for _, s := range f.states {
		if s.ProjectID == project {
			out = append(out, s)
		}
	}
	slices.SortFunc(out, func(a, b domain.State) int { return cmp.Compare(a.Name, b.Name) })
	return out
}

func (f *fakeStates) GreatestSequence(ctx context.Context, projectID uuid.UUID) (*float64, error) {
	f.log.add(ctx, "GreatestSequence %s", projectID)
	if err := f.fail("GreatestSequence"); err != nil {
		return nil, err
	}
	var greatest *float64
	for _, s := range f.of(projectID) {
		if greatest == nil || s.Sequence > *greatest {
			greatest = &s.Sequence
		}
	}
	return greatest, nil
}

// CreateState stores r's state, as stored, unless its project has a state
// of its name: domain.ErrStateNameTaken.
func (f *fakeStates) CreateState(ctx context.Context, r app.StateRow) (domain.State, error) {
	f.log.add(ctx, "CreateState %s", stateRow(r))
	if err := f.fail("CreateState"); err != nil {
		return domain.State{}, err
	}
	for _, s := range f.of(r.ProjectID) {
		if s.Name == r.State.Name {
			return domain.State{}, domain.ErrStateNameTaken
		}
	}
	at := r.Now.Truncate(time.Microsecond)
	s := domain.State{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, Name: r.State.Name, Description: r.State.Description,
		Color: r.State.Color, Group: r.State.Group, Default: r.State.Default, Sequence: r.State.Sequence, CreatedAt: at, UpdatedAt: at}
	f.states[r.ID] = s
	if f.changedAs != (uuid.UUID{}) {
		s.ID = f.changedAs
	}
	return s, nil
}

// stateRow is r as CreateState logs it: every field but its id, which the
// use case makes.
func stateRow(r app.StateRow) string {
	return fmt.Sprintf("%s/%s %q %s %s at %v default %v %q by %s at %s", r.WorkspaceID, r.ProjectID, r.State.Name, r.State.Color, r.State.Group,
		r.State.Sequence, r.State.Default, r.State.Description, r.CreatedBy, r.Now.Format(timeFormat))
}
````

`server/internal/modules/project/app/fakes_write_test.go`（修改，1 处）：

````old server/internal/modules/project/app/fakes_write_test.go
	// answers for the one asked; changedAs, when set, is the id
	// UpdateMemberRole answers for the one it changed.
````
````new server/internal/modules/project/app/fakes_write_test.go
	// answers for the one asked; changedAs, when set, is the id the write of
	// the row answers for the one it wrote (UpdateMemberRole, CreateState).
````

`server/internal/modules/project/app/create_state_test.go`（新文件，149 行）：

````file server/internal/modules/project/app/create_state_test.go
package app_test

import (
	"context"
	"maps"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newCreateState is CreateState over newStates' fakes, its clock logged.
func newCreateState() (*app.CreateState, *writeFixture, *fakeStates) {
	f, s := newStates()
	return app.NewCreateState(f.locks(), s, f.tx, clockAt{clockNow, f.log}), f, s
}

// review is the state the tests create: QA in the completed group, with a
// description, a name no state of web or ops has.
var review = domain.StateCreate{Name: "QA", Color: "#0EA5E9", Group: domain.GroupCompleted, Description: "Checked by QA"}

// stateCreated are the calls of user's creation of in in project after
// greatest: its locks and decision, the greatest sequence, the clock, the
// insert of the state after it, never the default, by user at that time.
func stateCreated(user, project uuid.UUID, in domain.StateCreate, sequence float64) []string {
	return append(lockedDecision(user, project, domain.ActionStateCreate), "GreatestSequence "+project.String(), "Now",
		"CreateState "+stateRow(app.StateRow{WorkspaceID: acme.ID, ProjectID: project, CreatedBy: user, Now: clockNow,
			State: domain.NewState{Name: in.Name, Color: in.Color, Group: in.Group, Description: in.Description, Sequence: sequence}}))
}

// CreateState, in one transaction and in the order of M3 design 3.6, locks
// the project, decides, reads the greatest sequence of its states and the
// clock, then inserts the state 15000 after it, by the caller at that time;
// it answers the state as stored, the time to the microsecond: web's after
// its Cancelled, 70000; archived ops's, as any other's (3.19), after its
// Backlog, 30000; and ops's, its states gone, at 65535.
func TestCreateState(t *testing.T) {
	for _, tt := range []struct {
		name     string
		project  uuid.UUID
		set      func(s *fakeStates)
		sequence float64
	}{
		{"web", webID, nil, 70000},
		{"archived ops", opsID, nil, 30000},
		{"ops without states", opsID, func(s *fakeStates) { delete(s.states, opsBacklog) }, 65535},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, s := newCreateState()
			if tt.set != nil {
				tt.set(s)
			}
			got, err := uc.Execute(as(bob), tt.project, review)
			stored := s.states[got.ID]
			want := domain.State{ID: got.ID, WorkspaceID: acme.ID, ProjectID: tt.project, Name: "QA", Description: "Checked by QA", Color: "#0EA5E9",
				Group: domain.GroupCompleted, Sequence: tt.sequence, CreatedAt: now, UpdatedAt: now}
			if err != nil || got != want || stored != want {
				t.Errorf("Execute() = %+v, %v, stored %+v; want %+v", got, err, stored, want)
			}
			if want := stateCreated(bob, tt.project, review, tt.sequence); !slices.Equal(f.log.calls, want) {
				t.Errorf("calls\n%q\nwant\n%q", f.log.calls, want)
			}
		})
	}
}

// Refusals, each in its place, and no state stored: no caller, and values
// the domain refuses, before the transaction; a project that is not there,
// a workspace or project deleted while its lock waited, a project moved
// meanwhile, and a caller who does not see the project, each
// project.not_found; a member, the Authorizer's 403. A name another state
// of the project has is the store's project.state_name_taken, after the
// insert. The state answered for another id is the write's own error.
func TestCreateStateRefuses(t *testing.T) {
	upTo := func(n int) []string { return lockedDecision(bob, webID, domain.ActionStateCreate)[:n] }
	taken := review
	taken.Name = "Review"
	for _, tt := range []struct {
		name    string
		ctx     context.Context
		project uuid.UUID
		in      domain.StateCreate
		set     func(f *writeFixture, s *fakeStates)
		want    error
		calls   []string
	}{
		{"no caller", context.Background(), webID, review, nil, shared.Unauthenticated(), nil},
		{"the triage group", as(bob), webID, domain.StateCreate{Name: "Intake", Color: "#000", Group: domain.GroupTriage}, nil,
			shared.Invalid(shared.FieldError{Field: "group", Code: shared.FieldNotAllowed}), nil},
		{"no project", as(bob), uuid.Nil(), review, nil, domain.ErrNotFound, noProject},
		{"acme deleted while its lock waited", as(bob), webID, review, func(f *writeFixture, _ *fakeStates) { f.workspaces.gone = true },
			domain.ErrNotFound, upTo(3)},
		{"web deleted while its lock waited", as(bob), webID, review, func(f *writeFixture, _ *fakeStates) { f.store.deleted = true },
			domain.ErrNotFound, upTo(4)},
		{"web moved to another workspace", as(bob), webID, review, func(f *writeFixture, _ *fakeStates) { f.store.moved = uuid.NewV7() },
			domain.ErrNotFound, upTo(4)},
		{"a caller who does not see web", as(erin), webID, review, nil, domain.ErrNotFound,
			lockedDecision(erin, webID, domain.ActionStateCreate)},
		{"a member", as(alice), webID, review, nil, shared.Forbidden(), lockedDecision(alice, webID, domain.ActionStateCreate)},
		{"a name web's Review has", as(bob), webID, taken, nil, domain.ErrStateNameTaken, stateCreated(bob, webID, taken, 70000)},
		{"the state answered for another id", as(bob), webID, review, func(_ *writeFixture, s *fakeStates) { s.changedAs = uuid.NewV7() },
			nil, stateCreated(bob, webID, review, 70000)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, s := newCreateState()
			if tt.set != nil {
				tt.set(f, s)
			}
			before := maps.Clone(s.states)
			_, err := uc.Execute(tt.ctx, tt.project, tt.in)
			outcome{tt.name, tt.want, tt.calls}.check(t, err, f)
			if tt.name != "the state answered for another id" && !maps.Equal(s.states, before) {
				t.Errorf("the states after the refusal: %v; want them as they were, %v", s.states, before)
			}
		})
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after.
func TestCreateStateReturnsEachFailure(t *testing.T) {
	all := stateCreated(bob, webID, review, 70000)
	fail := func(method string) func(f *writeFixture) {
		return func(f *writeFixture) { f.store.errs = map[string]error{method: errDisk} }
	}
	for _, tt := range []struct {
		name  string
		fail  func(f *writeFixture)
		calls int // how many of the creation's calls ran
	}{
		{"the project's workspace", fail("ProjectWorkspace"), 2},
		{"the workspace's lock", func(f *writeFixture) { f.workspaces.err = errDisk }, 3},
		{"the project's lock", fail("LockProject"), 4},
		{"the decision", func(f *writeFixture) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, 5},
		{"the greatest sequence", fail("GreatestSequence"), 6},
		{"the insert", fail("CreateState"), 8},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 8},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, _ := newCreateState()
			tt.fail(f)
			_, err := uc.Execute(as(bob), webID, review)
			outcome{tt.name, errDisk, all[:tt.calls]}.check(t, err, f)
		})
	}
}
````

`server/internal/modules/project/app/clock_test.go`（修改，2 处）：

````old server/internal/modules/project/app/clock_test.go
// never stamps an earlier time than the one it waited for. The clock logs
// its read among the fakes' calls.
````
````new server/internal/modules/project/app/clock_test.go
// never stamps an earlier time than the one it waited for. So does
// createState, which only inserts, after it reads the project's states
// under its lock. The clock logs its read among the fakes' calls.
````

````old server/internal/modules/project/app/clock_test.go
		}, left(bob, webID, true)},
````
````new server/internal/modules/project/app/clock_test.go
		}, left(bob, webID, true)},
		{"createState", func() ([]string, error) {
			uc, f, _ := newCreateState()
			_, err := uc.Execute(as(bob), webID, review)
			return f.log.calls, err
		}, stateCreated(bob, webID, review, 70000)},
````

- [ ] **Step 4: HTTP、接线、文案**

`server/internal/modules/project/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/project/adapter/http/handler.go
}

// UseCases are the use cases behind the module's operations.
````
````new server/internal/modules/project/adapter/http/handler.go
}

// CreateStateUseCase is app.CreateState.
type CreateStateUseCase interface {
	Execute(ctx context.Context, projectID uuid.UUID, in domain.StateCreate) (domain.State, error)
}

// UseCases are the use cases behind the module's operations.
````

````old server/internal/modules/project/adapter/http/handler.go
	LeaveProject      LeaveProjectUseCase
````
````new server/internal/modules/project/adapter/http/handler.go
	LeaveProject      LeaveProjectUseCase
	CreateState       CreateStateUseCase
````

`server/internal/modules/project/adapter/http/states.go`（新文件，29 行）：

````file server/internal/modules/project/adapter/http/states.go
package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// CreateState serves POST /api/v0/projects/{project_id}/states: the group
// goes to the use case as given, triage and one outside the five too, which
// the domain refuses (422).
func (h handler) CreateState(ctx context.Context, req gen.CreateStateRequestObject) (gen.CreateStateResponseObject, error) {
	in := domain.StateCreate{Name: req.Body.Name, Color: req.Body.Color, Group: domain.StateGroup(req.Body.Group)}
	if req.Body.Description != nil {
		in.Description = *req.Body.Description
	}
	s, err := h.uc.CreateState.Execute(ctx, req.ProjectID, in)
	if err != nil {
		return nil, err
	}
	return gen.CreateState201JSONResponse(state(s)), nil
}

// state is s as the API shows it.
func state(s domain.State) gen.State {
	return gen.State{ID: s.ID, WorkspaceID: s.WorkspaceID, ProjectID: s.ProjectID, Name: s.Name, Description: s.Description, Color: s.Color,
		Group: gen.StateGroup(s.Group), Default: s.Default, Sequence: s.Sequence, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt}
}
````

`server/internal/modules/project/adapter/http/handler_test.go`（修改，3 处）：

````old server/internal/modules/project/adapter/http/handler_test.go
	leave        *fakeDelete
````
````new server/internal/modules/project/adapter/http/handler_test.go
	leave        *fakeDelete
	createState  *fakeCreateState
````

````old server/internal/modules/project/adapter/http/handler_test.go
		f.leave = &fakeDelete{}
	}
````
````new server/internal/modules/project/adapter/http/handler_test.go
		f.leave = &fakeDelete{}
	}
	if f.createState == nil {
		f.createState = &fakeCreateState{}
	}
````

````old server/internal/modules/project/adapter/http/handler_test.go
		JoinProject: f.join, UpdateMember: f.updateMember, RemoveMember: f.removeMember, LeaveProject: f.leave})
````
````new server/internal/modules/project/adapter/http/handler_test.go
		JoinProject: f.join, UpdateMember: f.updateMember, RemoveMember: f.removeMember, LeaveProject: f.leave,
		CreateState: f.createState})
````

`server/internal/modules/project/adapter/http/states_test.go`（新文件，113 行）：

````file server/internal/modules/project/adapter/http/states_test.go
package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeCreateState is createState: each call is recorded as "caller id",
// with what it got; it answers answer, or err.
type fakeCreateState struct {
	calls  []string
	got    []domain.StateCreate
	answer domain.State
	err    error
}

func (f *fakeCreateState) Execute(ctx context.Context, projectID uuid.UUID, in domain.StateCreate) (domain.State, error) {
	f.calls = append(f.calls, caller(ctx)+" "+projectID.String())
	f.got = append(f.got, in)
	return f.answer, f.err
}

var (
	webReview = domain.State{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000c2"), WorkspaceID: acmeID, ProjectID: webID, Name: "Review",
		Description: "Waiting for a review", Color: "#F59E0B", Group: domain.GroupStarted, Sequence: 70000.5, CreatedAt: created,
		UpdatedAt: created.Add(1)}
	webReviewJSON = `{"color":"#F59E0B","created_at":"2026-10-01T10:00:00.123456Z","default":false,"description":"Waiting for a review",` +
		`"group":"started","id":"0199a2b4-0000-7000-8000-0000000000c2","name":"Review","project_id":"0199a2b4-0000-7000-8000-0000000000a1",` +
		`"sequence":70000.5,"updated_at":"2026-10-01T10:00:00.123456001Z","workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
	projectNotFoundJSON = `{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`
)

// POST goes to the use case for the caller and the path's project, with
// the body's fields, the description empty when not given and the group as
// given, triage too, which the domain refuses; the answer is 201 with the
// state the use case answers.
func TestCreateState(t *testing.T) {
	path := "/api/v0/projects/" + webID.String() + "/states"
	for _, tt := range []struct {
		body string
		want domain.StateCreate
	}{
		{`{"name":"Review","color":"#F59E0B","group":"started","description":"Waiting for a review"}`,
			domain.StateCreate{Name: "Review", Color: "#F59E0B", Group: domain.GroupStarted, Description: "Waiting for a review"}},
		{`{"name":"Intake","color":"#000","group":"triage"}`, domain.StateCreate{Name: "Intake", Color: "#000", Group: domain.GroupTriage}},
	} {
		create := &fakeCreateState{answer: webReview}
		h := newServer(t, fakes{createState: create})
		if res, body := do(t, h, request(http.MethodPost, path, "alice", tt.body)); res.StatusCode != http.StatusCreated ||
			body != webReviewJSON+"\n" {
			t.Errorf("POST %s = %d %s, want 201 %s", tt.body, res.StatusCode, body, webReviewJSON)
		}
		if want := []string{"alice " + webID.String()}; !slices.Equal(create.calls, want) || !reflect.DeepEqual(create.got,
			[]domain.StateCreate{tt.want}) {
			t.Errorf("POST %s: calls %q with %+v; want %q with %+v", tt.body, create.calls, create.got, want, tt.want)
		}
	}
}

// A body without its name, its color or its group, with a field it may not
// have, or a field of another type: refused as bad_request before the use
// case.
func TestCreateStateHoldsTheBodyToItsStructure(t *testing.T) {
	create := &fakeCreateState{}
	h := newServer(t, fakes{createState: create})
	for _, body := range []string{`{"color":"#000","group":"started"}`, `{"name":"A","group":"started"}`, `{"name":"A","color":"#000"}`,
		`{"name":"A","color":"#000","group":"started","default":true}`, `{"name":"A","color":"#000","group":"started","sequence":1}`,
		`{"name":1,"color":"#000","group":"started"}`, `{"name":"A","color":"#000","group":1}`} {
		res, got := do(t, h, request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/states", "alice", body))
		var problem struct{ Code string }
		if err := json.Unmarshal([]byte(got), &problem); err != nil || res.StatusCode != http.StatusBadRequest || problem.Code != "bad_request" {
			t.Errorf("POST %s = %d %s, want 400 bad_request", body, res.StatusCode, got)
		}
	}
	if len(create.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", create.calls)
	}
}

// The use case's refusals, as the contract declares them, and its failure.
func TestCreateStateRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"the triage group", shared.Invalid(shared.FieldError{Field: "group", Code: shared.FieldNotAllowed,
			Message: "must not be triage: the intake's state is not made here"}), http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"group","code":"not_allowed","message":"must not be triage: the intake's state is not made here"}]}`},
		{"no project", domain.ErrNotFound, http.StatusNotFound, projectNotFoundJSON},
		{"a project member", shared.Forbidden(), http.StatusForbidden, forbiddenJSON},
		{"a name taken", domain.ErrStateNameTaken, http.StatusConflict,
			`{"status":409,"code":"project.state_name_taken","title":"Conflict","detail":"A state of the project has this name."}`},
		{"a failure", errGone, http.StatusInternalServerError, internalErrorJSON},
	} {
		h := newServer(t, fakes{createState: &fakeCreateState{err: tt.err}})
		res, body := do(t, h, request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/states", "alice",
			`{"name":"Review","color":"#F59E0B","group":"started"}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: POST = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
````

`server/internal/modules/project/module.go`（修改，2 处）：

````old server/internal/modules/project/module.go
// member's display settings, carries out the workspace module's cascades
// on the projects (ProjectCascade), and offers the access module its reads
// of a project (ProjectAccess) and the workspace module its count of an
// account's ended project memberships (ProjectMembershipCounts).
````
````new server/internal/modules/project/module.go
// member's display settings, creating a project's states, carries out the
// workspace module's cascades on the projects (ProjectCascade), and offers
// the access module its reads of a project (ProjectAccess) and the
// workspace module its count of an account's ended project memberships
// (ProjectMembershipCounts).
````

````old server/internal/modules/project/module.go
		LeaveProject:      app.NewLeaveProject(locks, store, d.Tx, d.Clock),
````
````new server/internal/modules/project/module.go
		LeaveProject:      app.NewLeaveProject(locks, store, d.Tx, d.Clock),
		CreateState:       app.NewCreateState(locks, store, d.Tx, d.Clock),
````

`web/apps/web/helpers/authentication.helper.ts`（修改，1 处）：

````old web/apps/web/helpers/authentication.helper.ts
  "project.sole_admin": "auth.errors.project_sole_admin",
````
````new web/apps/web/helpers/authentication.helper.ts
  "project.sole_admin": "auth.errors.project_sole_admin",
  "project.state_name_taken": "auth.errors.project_state_name_taken",
````

`web/packages/i18n/src/locales/en/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/auth.json
      "project_sole_admin": "The project would be left without an admin: its only admin cannot leave it, nor can his membership end while it has other members. Give the project another admin first, or delete it.",
````
````new web/packages/i18n/src/locales/en/auth.json
      "project_sole_admin": "The project would be left without an admin: its only admin cannot leave it, nor can his membership end while it has other members. Give the project another admin first, or delete it.",
      "project_state_name_taken": "A state of this project already has this name.",
````

`web/packages/i18n/src/locales/zh-CN/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/auth.json
      "project_sole_admin": "项目会因此没有管理员：它唯一的管理员不能离开它；它还有别的成员时，他的成员关系也不能结束。请先给项目另一位管理员，或者删除这个项目。",
````
````new web/packages/i18n/src/locales/zh-CN/auth.json
      "project_sole_admin": "项目会因此没有管理员：它唯一的管理员不能离开它；它还有别的成员时，他的成员关系也不能结束。请先给项目另一位管理员，或者删除这个项目。",
      "project_state_name_taken": "这个项目里已有同名的状态。",
````

- [ ] **Step 5: 矩阵和组合测试**

`server/internal/bootstrap/permission_matrix_seeded_test.go`（修改，7 处）：

````old server/internal/bootstrap/permission_matrix_seeded_test.go
}

// ownInvitation is the workspace of the invitation to c's own address: one
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
}

// matrixStates are the states prepareMatrix seeds in each project of
// matrixProjects through the project store, by the workspace's admin, as
// createProject would make them: the six of domain.DefaultStates, Backlog
// the default and Triage the triage state, and Review, a second state of
// the started group, after In Progress. gone's are deleted with it.
var matrixStates = append(projectdomain.DefaultStates(),
	projectdomain.NewState{Name: "Review", Color: "#F59E0B", Sequence: 40000, Group: projectdomain.GroupStarted})

// ownInvitation is the workspace of the invitation to c's own address: one
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
// by the project's key and the column; and each account, by its name in
// matrixAccounts, which prepareMatrix registers. t is the test that asks
// for them (in).
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
// by the project's key and the column; each state, by the project's key
// and its name; and each account, by its name in matrixAccounts, which
// prepareMatrix registers. t is the test that asks for them (in).
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
	projectMembers map[string]uuid.UUID
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
	projectMembers map[string]uuid.UUID
	states         map[string]uuid.UUID
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
// matrixMemberships, each of matrixInvitations, each of matrixProjects and
// each of matrixProjectMembers before prepareMatrix writes them, so that
// matrixViolations, without a database, sees the keys and the targets the
// cells will.
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
// matrixMemberships, each of matrixInvitations, each of matrixProjects, each
// of matrixProjectMembers and each of matrixStates in each project before
// prepareMatrix writes them, so that matrixViolations, without a database,
// sees the keys and the targets the cells will.
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
		projects: map[string]uuid.UUID{}, projectMembers: map[string]uuid.UUID{}, accounts: map[caller]uuid.UUID{}}
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
		projects: map[string]uuid.UUID{}, projectMembers: map[string]uuid.UUID{}, states: map[string]uuid.UUID{}, accounts: map[caller]uuid.UUID{}}
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
		s.projects[p.key] = uuid.NewV7()
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
		s.projects[p.key] = uuid.NewV7()
		for _, st := range matrixStates {
			s.states[p.key+"|"+st.Name] = uuid.NewV7()
		}
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
		s.t.Fatalf("no membership of %s by %s is seeded", key, c)
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
		s.t.Fatalf("no membership of %s by %s is seeded", key, c)
	}
	return id
}

// state is the id of the state name of the project key; one never seeded
// fails the test at once, as membership's does.
func (s seeded) state(key, name string) uuid.UUID {
	id, ok := s.states[key+"|"+name]
	if !ok {
		s.t.Helper()
		s.t.Fatalf("no state %s of %s is seeded", name, key)
````

`server/internal/bootstrap/permission_matrix_seed_test.go`（修改，3 处）：

````old server/internal/bootstrap/permission_matrix_seed_test.go
	"context"
````
````new server/internal/bootstrap/permission_matrix_seed_test.go
	"context"
	"slices"
````

````old server/internal/bootstrap/permission_matrix_seed_test.go
	"uuid"

````
````new server/internal/bootstrap/permission_matrix_seed_test.go
	"uuid"

	"github.com/jackc/pgx/v5"
````

````old server/internal/bootstrap/permission_matrix_seed_test.go
		Now: s.now,
	}); err != nil {
		s.t.Fatal(err)
	}
````
````new server/internal/bootstrap/permission_matrix_seed_test.go
		Now: s.now,
	}); err != nil {
		s.t.Fatal(err)
	}
}

// states stores each project's matrixStates, as createProject makes its
// own, with the ids sd names, by its workspace's admin.
func (s projectSeed) states(sd seeded) {
	s.t.Helper()
	for _, p := range matrixProjects {
		slug, _, _ := strings.Cut(p.key, "/")
		var rows []projectapp.StateRow
		for _, st := range matrixStates {
			rows = append(rows, projectapp.StateRow{ID: sd.state(p.key, st.Name), WorkspaceID: s.workspaces[slug], ProjectID: s.projects[p.key],
				State: st, CreatedBy: s.ids[matrixAdmins[slug]], Now: s.now})
		}
		if err := s.store.CreateStates(context.Background(), rows); err != nil {
			s.t.Fatal(err)
		}
	}
}

// seededState is a state of a matrix project as seededStates reads it.
type seededState struct {
	Name, Group string
	Default     bool
	Sequence    float64
	Deleted     bool
}

// seededStates checks the states the cells of each matrix project rest on,
// read back by name: those of matrixStates exactly, each in its group, at
// its sequence, Backlog the only default and Triage the only state of the
// triage group; undeleted, but gone's, deleted with gone. A state missing
// or seeded otherwise would let a cell answer as it wants for another
// reason.
func (s projectSeed) seededStates(pool *pgxpool.Pool) {
	s.t.Helper()
	for _, p := range matrixProjects {
		gone := strings.HasPrefix(p.key, "gone/")
		var want []seededState
		for _, st := range matrixStates {
			want = append(want, seededState{Name: st.Name, Group: string(st.Group), Default: st.Default, Sequence: st.Sequence, Deleted: gone})
		}
		slices.SortFunc(want, func(a, b seededState) int { return strings.Compare(a.Name, b.Name) })
		rows, err := pool.Query(context.Background(), `SELECT name, "group", "default", sequence, deleted_at IS NOT NULL FROM states
			WHERE project_id = $1 ORDER BY name COLLATE "C"`, s.projects[p.key])
		if err != nil {
			s.t.Fatal(err)
		}
		got, err := pgx.CollectRows(rows, pgx.RowToStructByPos[seededState])
		if err != nil {
			s.t.Fatal(err)
		}
		var defaults, triage []string
		for _, st := range got {
			if st.Default {
				defaults = append(defaults, st.Name)
			}
			if st.Group == string(projectdomain.GroupTriage) {
				triage = append(triage, st.Name)
			}
		}
		if !slices.Equal(got, want) || !slices.Equal(defaults, []string{"Backlog"}) || !slices.Equal(triage, []string{"Triage"}) {
			s.t.Fatalf("%s's states by name = %+v, default %q, triage %q; want %+v, Backlog the default, Triage the triage state",
				p.key, got, defaults, triage, want)
		}
	}
````

`server/internal/bootstrap/permission_matrix_test.go`（修改，4 处）：

````old server/internal/bootstrap/permission_matrix_test.go
	return slices.Concat(workspaceMatrixRows(), projectMatrixRows(), memberMatrixRows(), membershipMatrixRows())
````
````new server/internal/bootstrap/permission_matrix_test.go
	return slices.Concat(workspaceMatrixRows(), projectMatrixRows(), memberMatrixRows(), membershipMatrixRows(), stateMatrixRows())
````

````old server/internal/bootstrap/permission_matrix_test.go
// projects and project memberships of matrixProjects and
// matrixProjectMembers, and acme's archived project archived. Through both
// stores, the removed member's removal; then, through the workspace store,
// the invitations of matrixInvitations. Through the project store, the
// memberships P5b's writes end (endings); through SQL, the states no store
// makes alone (partingStates). Through the API, gone deleted by
// its admin, which soft-deletes its memberships and its project with it;
// then the checks that the rows the cells rest on are there
// (preconditions). Everything that connected to the database is closed
// when it returns, so that it can be copied. A -run that leaves out
// prepare fails here, not with a 401 in every cell.
````
````new server/internal/bootstrap/permission_matrix_test.go
// projects, project memberships and states of matrixProjects,
// matrixProjectMembers and matrixStates, and acme's archived project
// archived. Through both stores, the removed member's removal; then,
// through the workspace store, the invitations of matrixInvitations.
// Through the project store, the memberships P5b's writes end (endings);
// through SQL, the states no store makes alone (partingStates). Through
// the API, gone deleted by its admin, which soft-deletes its memberships
// and its project, with its states, with it; then the checks that the rows
// the cells rest on are there (preconditions), the states among them
// (seededStates). Everything that connected to the database is closed when
// it returns, so that it can be copied. A -run that leaves out prepare
// fails here, not with a 401 in every cell.
````

````old server/internal/bootstrap/permission_matrix_test.go
			projects.join(s.projectMember(pm.key, pm.c), pm.key, pm.c, pm.role)
		}
````
````new server/internal/bootstrap/permission_matrix_test.go
			projects.join(s.projectMember(pm.key, pm.c), pm.key, pm.c, pm.role)
		}
		projects.states(s)
````

````old server/internal/bootstrap/permission_matrix_test.go
		projects.preconditions(s)
````
````new server/internal/bootstrap/permission_matrix_test.go
		projects.preconditions(s)
		projects.seededStates(pool)
````

`server/internal/bootstrap/permission_matrix_columns_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_columns_test.go
		t.Errorf("a project membership never seeded: failed with %q, want %q", failed, want)
	}
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
		t.Errorf("a project membership never seeded: failed with %q, want %q", failed, want)
	}
	// And a state never seeded.
	failed = fatalOf(func(tb testing.TB) { newSeeded().in(tb).state("acme/public", "Triaged") })
	if want := "no state Triaged of acme/public is seeded"; failed != want {
		t.Errorf("a state never seeded: failed with %q, want %q", failed, want)
	}
````

`server/internal/bootstrap/permission_matrix_states_test.go`（新文件，53 行）：

````file server/internal/bootstrap/permission_matrix_states_test.go
package bootstrap

import (
	"net/http"
	"testing"
	"uuid"
)

// The rows of the permission matrix of a project's states (M3 design 9.2):
// creating them, under the project of each column, its seeded states those
// of matrixStates.

var cellStateNameTaken = cell{http.StatusConflict, "project.state_name_taken"}

// newState is the body of the state the rows create: QA, of the completed
// group, a name no seeded state has.
const newState = `{"name":"QA","color":"#0EA5E9","group":"completed"}`

func stateMatrixRows() []matrixRow {
	return []matrixRow{
		// The project's admins, and its members who are the workspace's
		// admins (M3 design 3.4): not its guests, whom Plane lets change
		// states.
		{op: "createState", write: true, columns: projectColumns, request: toProject(http.MethodPost, "/states", newState),
			cells: ofProject(cellCreated, cellForbidden, cellForbidden, cellCreated, cellForbidden, cellForbidden), check: createsTheState},
		// The name's 409 comes after the decision: who may not create states
		// learns nothing of the project's.
		{op: "createState", variant: "a name taken", write: true, columns: projectColumns,
			request: toProject(http.MethodPost, "/states", `{"name":"Todo","color":"#0EA5E9","group":"completed"}`),
			cells:   ofProject(cellStateNameTaken, cellForbidden, cellForbidden, cellStateNameTaken, cellForbidden, cellForbidden)},
		// An archived project's states are created as any other's (M3 design
		// 3.19).
		{op: "createState", variant: "archived", write: true, columns: archivedColumns, request: toProject(http.MethodPost, "/states", newState),
			cells: ofArchived(cellCreated, cellForbidden), check: createsTheState},
	}
}

// createsTheState: QA in the column's project, after its Cancelled, the
// greatest sequence of its states but its triage state, and not its
// default.
func createsTheState(t *testing.T, c caller, s seeded, answer string) {
	var st struct {
		ProjectID uuid.UUID `json:"project_id"`
		Name      string    `json:"name"`
		Group     string    `json:"group"`
		Default   bool      `json:"default"`
		Sequence  float64   `json:"sequence"`
	}
	decodeAnswer(t, answer, &st)
	if st.ProjectID != s.project(projectOf(c)) || st.Name != "QA" || st.Group != "completed" || st.Default || st.Sequence != 70000 {
		t.Errorf("%s creates %s; want QA in %s, completed, at 70000, not the default", c, answer, projectOf(c))
	}
}
````

`server/internal/bootstrap/project_write_locks_test.go`（修改，1 处）：

````old server/internal/bootstrap/project_write_locks_test.go
		member: [2]string{"bob", "carol"}},
````
````new server/internal/bootstrap/project_write_locks_test.go
		member: [2]string{"bob", "carol"}},
	{op: "createState", method: http.MethodPost, path: "/api/v0/projects/%s/states", body: `{"name":"QA","color":"#0EA5E9","group":"completed"}`,
		want: http.StatusCreated},
````

`server/internal/bootstrap/project_writes_test.go`（修改，2 处）：

````old server/internal/bootstrap/project_writes_test.go
// an admin of Web, removes carol, and leaves Web. Each row the write writes
// again is first made bob's, as last written by him, and checked so: a
// write that kept its row's writer would pass for alice's otherwise, she
// having made it.
````
````new server/internal/bootstrap/project_writes_test.go
// an admin of Web, removes carol, creates a state, and leaves Web. Each row
// the write writes again is first made bob's, as last written by him, and
// checked so: a write that kept its row's writer would pass for alice's
// otherwise, she having made it.
````

````old server/internal/bootstrap/project_writes_test.go
			"SELECT updated_at, updated_by_id = $2 AND NOT is_active FROM project_members WHERE project_id = $1 AND member_id = '" + carol + "'",
			1, 1},
````
````new server/internal/bootstrap/project_writes_test.go
			"SELECT updated_at, updated_by_id = $2 AND NOT is_active FROM project_members WHERE project_id = $1 AND member_id = '" + carol + "'",
			1, 1},
		// QA, made.
		{"createState", http.MethodPost, "/api/v0/projects/" + web.String() + "/states", `{"name":"QA","color":"#0EA5E9","group":"completed"}`,
			uuid.UUID{}, http.StatusCreated, "",
			"SELECT created_at, created_by_id = $2 AND updated_by_id = $2 AND updated_at = created_at FROM states WHERE project_id = $1 AND name = 'QA'",
			0, 1},
````

`server/internal/bootstrap/project_connection_test.go`（修改，2 处）：

````old server/internal/bootstrap/project_connection_test.go
// admin of Web and removes bob; carol, an admin, leaves it; alice deletes
// both projects.
````
````new server/internal/bootstrap/project_connection_test.go
// admin of Web and removes bob, and creates a state in it; carol, an
// admin, leaves it; alice deletes both projects.
````

````old server/internal/bootstrap/project_connection_test.go
	send(http.MethodDelete, "/api/v0/project-members/"+projectMemberships(t, r.pool, r.bob, r.web)[0].String(), r.alice, "", http.StatusNoContent)
````
````new server/internal/bootstrap/project_connection_test.go
	send(http.MethodDelete, "/api/v0/project-members/"+projectMemberships(t, r.pool, r.bob, r.web)[0].String(), r.alice, "", http.StatusNoContent)
	send(http.MethodPost, web+"/states", r.alice, `{"name":"QA","color":"#0EA5E9","group":"completed"}`, http.StatusCreated)
````

- [ ] **Step 6: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/project/... ./internal/modules/access/...`
Expected: 全部 `ok`（`project` 的 `apitest.Main` 两个方向：`createState` 的四个码各由 HTTP 测试返回过）。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix$|TestMatrixViolationsCatchesEachColumnGap$|TestEachWriteOnAProjectSharesItsWorkspaceFirst$|TestTheWritesOnAProjectStampTheirRequest$|TestTheWritesOnAProjectRunOnTheirTransactionsConnection$' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过（关键词守卫 5 个命中都有例外）。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

- [ ] **Step 7: 提交**

```bash
git add api/modules/project.yaml api/openapi.yaml server/internal/bootstrap/permission_matrix_columns_test.go server/internal/bootstrap/permission_matrix_seed_test.go server/internal/bootstrap/permission_matrix_seeded_test.go server/internal/bootstrap/permission_matrix_states_test.go server/internal/bootstrap/permission_matrix_test.go server/internal/bootstrap/project_connection_test.go server/internal/bootstrap/project_write_locks_test.go server/internal/bootstrap/project_writes_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/project/adapter/http/handler.go server/internal/modules/project/adapter/http/handler_test.go server/internal/modules/project/adapter/http/states.go server/internal/modules/project/adapter/http/states_test.go server/internal/modules/project/app/clock_test.go server/internal/modules/project/app/create_state.go server/internal/modules/project/app/create_state_test.go server/internal/modules/project/app/fakes_state_test.go server/internal/modules/project/app/fakes_write_test.go server/internal/modules/project/app/state_ports.go server/internal/modules/project/domain/actions.go server/internal/modules/project/module.go web/apps/web/helpers/authentication.helper.ts web/packages/i18n/src/locales/en/auth.json web/packages/i18n/src/locales/zh-CN/auth.json api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/bodyshape.gen.go server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P7a): createState

POST /api/v0/projects/{project_id}/states creates a state for the
project's admins, and its members who are the workspace's admins,
after the workspace's and the project's locks and the decision: 15000
after the greatest sequence of the project's states but the triage
state, not the default, by the caller at the time read under the
locks; an archived project's states are created as any other's. A
taken name, the triage state's too, is project.state_name_taken, after
the decision. The matrix seeds each project's states, reads them back
as a precondition, and holds the operation's cells, the archived
project's among them.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p7a.py` 的编号；"层"是它被发现的每一层，端到端是单独运行的故事）：

| 变异 | 改坏 | 必须失败的测试 | 层 |
|---|---|---|---|
| `s2-create-greatest` | `createState` 不管 `GreatestSequence` 的失败 | `TestCreateStateReturnsEachFailure` | 单元（按性质只在单元一层：真实的存储在这一步不失败） |
| `s2-create-create` | `createState` 吞掉 `CreateState` 的失败 | `TestCreateStateRefuses`、`TestCreateStateReturnsEachFailure` | 单元（按性质只在单元一层：真实的存储在这一步不失败） |
| `s22-created-id` | `createState` 不核对建出的状态的 id | `TestCreateStateRefuses` | 单元（按性质只在单元一层：真实的存储回答不了别的键） |
| `p-create-share-project` | `createState` 以 `FOR SHARE` 锁项目 | `TestStateWritesOnOneProjectSerialize`（Task 9 起） | 组合 |
| `c-create-early` | `createState` 在取锁之前读时钟 | `TestCreateState`、`TestCreateStateRefuses`、`TestCreateStateReturnsEachFailure`、`TestEachWriteReadsTheClockUnderItsLock`、`TestStateWritesOnOneProjectSerialize`（Task 9 起） | 单元；组合 |
| `q-greatest-ignored` | `createState` 不看最大的 `sequence` | `TestCreateState`、`TestCreateStateRefuses`、`TestCreateStateReturnsEachFailure`、`TestEachWriteReadsTheClockUnderItsLock`、`TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize`、`TestARoleChangeCapsItsMemberByHisWorkspaceRoleUnderItsLocks`、`TestAWriteOnAProjectMembershipDecidesOnWhatItReadUnderItsLocks`、`TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`（Task 9 起） 等 14 个、P6（Task 10 起） | 单元；组合；端到端 |
| `s4-create-frozen-clock` | `CreateState` 接上停在 2001 年的时钟 | `TestTheWritesOnAProjectStampTheirRequest`（Task 7 起） | 组合 |
| `s5-create-guests` | 规则表：`state.create` 也给项目访客 | `TestEveryRuleDecidesItsCells`、`TestPermissionMatrix`、W11（Task 10 起） | 单元；组合；端到端 |
| `pf-create-decide-unlocked` | `createState` 在取锁之前判定，取锁时不再判定（预检的变异，裁定 M1） | `TestCreateState`、`TestCreateStateRefuses`、`TestCreateStateReturnsEachFailure`、`TestEachWriteReadsTheClockUnderItsLock`、`TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`（Task 9 起） | 单元；组合 |
| `s10-seed-no-default` | 矩阵的状态种子不把 Backlog 设为默认（裁定 L1） | `TestListingProjectsIsReadingEach`、`TestListingWorkspaceStatesIsListingEachProjects`（Task 8 起）、`TestPermissionMatrix`、`TestTheInvitationLinkAnswersEveryCallerAlike` | 组合 |

---

### Task 5: `updateState` 的用例：端口、假实现、规则、操作名、时钟

**Files:**
- Create: `server/internal/modules/project/app/update_state.go`、`server/internal/modules/project/app/update_state_test.go`
- Modify: `server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/project/app/clock_test.go`、`server/internal/modules/project/app/fakes_member_test.go`、`server/internal/modules/project/app/fakes_state_test.go`、`server/internal/modules/project/app/fakes_write_test.go`、`server/internal/modules/project/app/state_ports.go`、`server/internal/modules/project/domain/actions.go`

**Interfaces:**
- Produces（spec 2.7；M3 设计 3.6、3.17、6.7）：
  - `project/app/state_ports.go`：`StateFinder`（`StateByID`：按行寻址的写的 `rowWrite.find`）、`GroupCounter`（`CountGroupStates`）、`StateUpdater`（`StateFinder`、`GroupCounter`、`UpdateState`；锁经用例的 `Locks`）。
  - `UpdateState`（`update_state.go`）：`NewUpdateState(locks Locks, states StateUpdater, tx shared.TxManager, clock Clock)`；`Execute(ctx, id, domain.StatePatch) (domain.State, error)`：没有调用者、领域拒绝的值，在事务之前；事务里 `lockRowAndDecide(… rowWrite[domain.State]{id, ActionStateUpdate, find: StateByID, notFound: domain.ErrStateNotFound})`；给了组、且与它现在的组不同时，数它现在的组（`CountGroupStates`），`CheckGroupKept(n - 1)`（判定之后、写之前，6.7）；时钟；`UpdateState`；回答的 id 不是这一行的时是错误。已归档项目的状态照常改（3.19）。
  - `access`：`state.update`（同 `state.create`）；`ActionStateUpdate`。接口和组合在 Task 6。
- 假实现（`fakes_state_test.go`）：`fakeStates.StateByID`（按 id 回答，分诊、已删除的不回答；`answersAs` 回答另一个 id）、`CountGroupStates`、`UpdateState`（`statePatch` 照存储的 `coalesce`）；`fakes_member_test.go` 的 `rowReads` 说明成员关系多三列。

**Tests:**
- `TestUpdateState`（`update_state_test.go`，四行：Review 改到 completed 组，started 组还有 In Progress；In Progress 留在它的组，不数；Todo 改名、往下移；归档的 ops 的 Backlog 照常；调用记录、由调用者、时刻到微秒）、`TestUpdateStateRefuses`（12 行：没有调用者、分诊组在事务之前；没有这一行、等锁时 acme 或 web 被删除、这一行被删除或移到 ops、看不到 web 的调用者，各 `project.state_not_found`；成员 403；Todo（组里唯一的状态）改到别的组 409，在写之前；Done 改成 Review 的名称，存储的 409；回答另一个状态的改动，写自己的错误）、`TestUpdateStateReturnsEachFailure`（Review 改到 backlog 组，数组的一步也运行；八个调用的失败各原样返回）；`TestEachWriteReadsTheClockUnderItsLock` 加 `updateState`。

- [ ] **Step 1: 规则和操作名**

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"state.create": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
````
````new server/internal/modules/access/domain/rules.go
	"state.create": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	// As state.create.
	"state.update": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"state.create": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
````
````new server/internal/modules/access/domain/rules_test.go
	"state.create": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
	"state.update": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
````

`server/internal/modules/project/domain/actions.go`（修改，2 处）：

````old server/internal/modules/project/domain/actions.go
	ActionStateCreate shared.Action = "state.create"
````
````new server/internal/modules/project/domain/actions.go
	ActionStateCreate shared.Action = "state.create"
	// ActionStateUpdate is changing a state: updateState.
	ActionStateUpdate shared.Action = "state.update"
````

````old server/internal/modules/project/domain/actions.go
		ActionStateCreate}
````
````new server/internal/modules/project/domain/actions.go
		ActionStateCreate, ActionStateUpdate}
````

- [ ] **Step 2: 端口、用例和测试**

`server/internal/modules/project/app/state_ports.go`（修改，2 处）：

````old server/internal/modules/project/app/state_ports.go
	"context"
````
````new server/internal/modules/project/app/state_ports.go
	"context"
	"time"
````

````old server/internal/modules/project/app/state_ports.go
}

````
````new server/internal/modules/project/app/state_ports.go
}

// StateFinder reads a state a write names by its id: first without a lock,
// for its project and the project's workspace, then again under their
// locks (lockRowAndDecide).
type StateFinder interface {
	// StateByID is the undeleted state id, unless it is the triage state;
	// found is false when there is none.
	StateByID(ctx context.Context, id uuid.UUID) (s domain.State, found bool, err error)
}

// GroupCounter counts the states of a group of a project: a write that
// takes a state from its group refuses to leave the group without one
// (domain.CheckGroupKept). It runs in the transaction ctx carries, under
// the project's FOR NO KEY UPDATE, which every write of its states takes,
// so the count stays as read.
type GroupCounter interface {
	// CountGroupStates is the number of projectID's undeleted states in
	// group.
	CountGroupStates(ctx context.Context, projectID uuid.UUID, group domain.StateGroup) (int, error)
}

// StateUpdater is updateState's repository. Its writes run in the
// transaction ctx carries, under the project's FOR NO KEY UPDATE.
type StateUpdater interface {
	StateFinder
	GroupCounter
	// UpdateState changes the fields p gives of the undeleted state id, by
	// the account by at now, and answers it as stored. A name another
	// undeleted state of the project has is domain.ErrStateNameTaken.
	UpdateState(ctx context.Context, id uuid.UUID, p domain.StatePatch, by uuid.UUID, now time.Time) (domain.State, error)
}

````

`server/internal/modules/project/app/update_state.go`（新文件，70 行）：

````file server/internal/modules/project/app/update_state.go
package app

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateState changes a state: PATCH /api/v0/states/{state_id} (M3 design
// 3.17, 6.7).
type UpdateState struct {
	locks  Locks
	states StateUpdater
	tx     shared.TxManager
	clock  Clock
}

// NewUpdateState returns the use case.
func NewUpdateState(locks Locks, states StateUpdater, tx shared.TxManager, clock Clock) *UpdateState {
	return &UpdateState{locks: locks, states: states, tx: tx, clock: clock}
}

// Execute checks p (domain.CheckStatePatch), then in one transaction, in
// the order of M3 design 3.6 and 6.7: the state's locks (lockRowAndDecide:
// the state read for its project and workspace, the workspace FOR SHARE,
// the project FOR NO KEY UPDATE, the state read again) and the decision on
// state.update; an archived project's states change as any other's
// (3.19). Then, when p moves the state to another group, the states its
// group keeps without it (domain.CheckGroupKept); then the change, at the
// time the clock gives under the locks. The answer is the state as stored.
func (u *UpdateState) Execute(ctx context.Context, id uuid.UUID, p domain.StatePatch) (domain.State, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.State{}, err
	}
	if err := domain.CheckStatePatch(p); err != nil {
		return domain.State{}, err
	}
	var updated domain.State
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		_, s, err := lockRowAndDecide(ctx, u.locks, actor, rowWrite[domain.State]{id: id, action: domain.ActionStateUpdate,
			find: u.states.StateByID, notFound: domain.ErrStateNotFound})
		if err != nil {
			return err
		}
		if p.Group != nil && *p.Group != s.Group {
			n, err := u.states.CountGroupStates(ctx, s.ProjectID, s.Group)
			if err != nil {
				return err
			}
			if err := domain.CheckGroupKept(n - 1); err != nil {
				return err
			}
		}
		if updated, err = u.states.UpdateState(ctx, s.ID, p, actor.UserID, u.clock.Now()); err != nil {
			return err
		}
		if updated.ID != s.ID {
			return fmt.Errorf("state %s changed as %s", s.ID, updated.ID)
		}
		return nil
	})
	if err != nil {
		return domain.State{}, err
	}
	return updated, nil
}
````

`server/internal/modules/project/app/fakes_state_test.go`（修改，2 处）：

````old server/internal/modules/project/app/fakes_state_test.go
// hold them, by id, beside fakeStore's projects, whose log, failures and
// changedAs it shares; the triage states are none of them, as the store
// reads none.
````
````new server/internal/modules/project/app/fakes_state_test.go
// hold them, by id, beside fakeStore's projects, whose log, failures,
// reads of a row by its id and changedAs it shares; the triage states are
// none of them, as the store reads none.
````

````old server/internal/modules/project/app/fakes_state_test.go
}

// stateRow is r as CreateState logs it: every field but its id, which the
````
````new server/internal/modules/project/app/fakes_state_test.go
}

// StateByID is the state id, read again under the locks as f.reread says.
func (f *fakeStates) StateByID(ctx context.Context, id uuid.UUID) (domain.State, bool, error) {
	f.log.add(ctx, "StateByID %s", id)
	f.rowReadCount++
	again := f.rowReadCount > 1
	if err := f.fail("StateByID"); err != nil {
		return domain.State{}, false, err
	}
	if again && f.reread.err != nil {
		return domain.State{}, false, fmt.Errorf("StateByID: %w", f.reread.err)
	}
	s, ok := f.states[id]
	if !ok || (again && f.reread.gone) {
		return domain.State{}, false, nil
	}
	if again && f.reread.project != (uuid.UUID{}) {
		s.ProjectID = f.reread.project
	}
	return s, true, nil
}

func (f *fakeStates) CountGroupStates(ctx context.Context, projectID uuid.UUID, group domain.StateGroup) (int, error) {
	f.log.add(ctx, "CountGroupStates %s %s", projectID, group)
	if err := f.fail("CountGroupStates"); err != nil {
		return 0, err
	}
	n := 0
	for _, s := range f.of(projectID) {
		if s.Group == group {
			n++
		}
	}
	return n, nil
}

// UpdateState changes the state id as p gives, stored, unless another state
// of its project has the name p gives: domain.ErrStateNameTaken.
func (f *fakeStates) UpdateState(ctx context.Context, id uuid.UUID, p domain.StatePatch, by uuid.UUID, now time.Time) (domain.State, error) {
	f.log.add(ctx, "UpdateState %s %s by %s at %s", id, statePatch(p), by, now.Format(timeFormat))
	if err := f.fail("UpdateState"); err != nil {
		return domain.State{}, err
	}
	s, ok := f.states[id]
	if !ok {
		return domain.State{}, fmt.Errorf("UpdateState: no state %s", id)
	}
	if p.Name != nil {
		for _, other := range f.of(s.ProjectID) {
			if other.ID != id && other.Name == *p.Name {
				return domain.State{}, domain.ErrStateNameTaken
			}
		}
		s.Name = *p.Name
	}
	if p.Color != nil {
		s.Color = *p.Color
	}
	if p.Group != nil {
		s.Group = *p.Group
	}
	if p.Description != nil {
		s.Description = *p.Description
	}
	if p.Sequence != nil {
		s.Sequence = *p.Sequence
	}
	s.UpdatedAt = now.Truncate(time.Microsecond)
	f.states[id] = s
	if f.changedAs != (uuid.UUID{}) {
		s.ID = f.changedAs
	}
	return s, nil
}

// statePatch is p as UpdateState logs it: each field it gives.
func statePatch(p domain.StatePatch) string {
	out := "{"
	for _, field := range []struct {
		name  string
		value any
	}{{"name", p.Name}, {"color", p.Color}, {"group", p.Group}, {"description", p.Description}, {"sequence", p.Sequence}} {
		switch v := field.value.(type) {
		case *string:
			if v != nil {
				out += fmt.Sprintf(" %s %q", field.name, *v)
			}
		case *domain.StateGroup:
			if v != nil {
				out += fmt.Sprintf(" %s %s", field.name, *v)
			}
		case *float64:
			if v != nil {
				out += fmt.Sprintf(" %s %v", field.name, *v)
			}
		}
	}
	return out + " }"
}

// stateRow is r as CreateState logs it: every field but its id, which the
````

`server/internal/modules/project/app/fakes_write_test.go`（修改，1 处）：

````old server/internal/modules/project/app/fakes_write_test.go
	// The reads of a membership by its id (fakes_member_test.go): how many
	// ran, how the second one answers, and answersAs, when set, the id each
	// answers for the one asked; changedAs, when set, is the id the write of
	// the row answers for the one it wrote (UpdateMemberRole, CreateState).
	memberReadCount int
	reread          memberReads
	answersAs       uuid.UUID
	changedAs       uuid.UUID
````
````new server/internal/modules/project/app/fakes_write_test.go
	// The reads of a row under a project by its id, a membership's
	// (fakes_member_test.go) or a state's (fakes_state_test.go): how many
	// ran, and how the second one answers; answersAs, when set, is the id
	// each read of a membership answers for the one asked. changedAs, when
	// set, is the id the write of the row answers for the one it wrote
	// (UpdateMemberRole, CreateState, UpdateState).
	rowReadCount int
	reread       rowReads
	answersAs    uuid.UUID
	changedAs    uuid.UUID
}

// rowReads is how a row under a project read again under the locks
// answers: err fails it, gone finds none, as if it was deleted meanwhile,
// and project, when set, is the project it is found in, as if it had moved
// there. A membership's has three more: id, when set, is the id it answers
// for the one asked; role, when set, its role, as if it had been changed
// meanwhile; and ended finds it ended, as if it had been ended meanwhile.
type rowReads struct {
	err     error
	gone    bool
	project uuid.UUID
	id      uuid.UUID
	role    shared.Role
	ended   bool
````

`server/internal/modules/project/app/fakes_member_test.go`（修改，2 处）：

````old server/internal/modules/project/app/fakes_member_test.go

// memberReads is how a membership read again under the locks answers: err
// fails it, gone finds none, as if it was deleted meanwhile, project,
// when set, is the project it is found in, as if it had moved there,
// role, when set, its role, as if it had been changed meanwhile, ended
// finds it ended, as if it had been ended meanwhile, and id, when set, is
// the id it answers for the one asked.
type memberReads struct {
	err     error
	gone    bool
	project uuid.UUID
	role    shared.Role
	ended   bool
	id      uuid.UUID
}

````
````new server/internal/modules/project/app/fakes_member_test.go

````

````old server/internal/modules/project/app/fakes_member_test.go
	f.memberReadCount++
	again := f.memberReadCount > 1
````
````new server/internal/modules/project/app/fakes_member_test.go
	f.rowReadCount++
	again := f.rowReadCount > 1
````

`server/internal/modules/project/app/update_state_test.go`（新文件，185 行）：

````file server/internal/modules/project/app/update_state_test.go
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

// newUpdateState is UpdateState over newStates' fakes, its clock logged.
func newUpdateState() (*app.UpdateState, *writeFixture, *fakeStates) {
	f, s := newStates()
	return app.NewUpdateState(f.locks(), s, f.tx, clockAt{clockNow, f.log}), f, s
}

// stateLocked are the calls of a write by caller on the state id of
// project up to its decision on action: the transaction, the state read,
// acme's row FOR SHARE, the project FOR NO KEY UPDATE, the state read
// again, the decision.
func stateLocked(id, project, caller uuid.UUID, action shared.Action) []string {
	return []string{"Begin", "StateByID " + id.String(), "ShareWorkspaceByID " + acme.ID.String(), "LockProject " + project.String(),
		"StateByID " + id.String(), fmt.Sprintf("Authorize %s %s on %s/%s", caller, action, acme.ID, project)}
}

// stateUpdated are the calls of user's change p of the state id of web:
// its locks and decision, the count of its group when p moves it from
// group, the clock, the change by user at that time.
func stateUpdated(user, id uuid.UUID, p domain.StatePatch, from domain.StateGroup) []string {
	calls := stateLocked(id, webID, user, domain.ActionStateUpdate)
	if p.Group != nil && *p.Group != from {
		calls = append(calls, fmt.Sprintf("CountGroupStates %s %s", webID, from))
	}
	return append(calls, "Now", fmt.Sprintf("UpdateState %s %s by %s at %s", id, statePatch(p), user, clockNow.Format(timeFormat)))
}

// UpdateState, in one transaction and in the order of M3 design 3.6 and
// 6.7, locks the state's project, decides, counts the group a change of
// group takes the state from, reads the clock, then changes the fields
// given, by the caller at that time; it answers the state as stored, the
// time to the microsecond. Review leaves the started group, which keeps In
// Progress; In Progress stays in it, which counts nothing; Todo is renamed
// and moved down; and ops's Backlog, of an archived project, changes as any
// other (3.19).
func TestUpdateState(t *testing.T) {
	for _, tt := range []struct {
		name string
		id   uuid.UUID
		p    domain.StatePatch
	}{
		{"Review to the completed group", webReview, domain.StatePatch{Group: ptr(domain.GroupCompleted), Color: ptr("#46A758")}},
		{"In Progress kept in its group", webStarted, domain.StatePatch{Group: ptr(domain.GroupStarted), Description: ptr("Being done")}},
		{"Todo renamed and moved down", webTodo, domain.StatePatch{Name: ptr("Next"), Sequence: ptr(60000.0)}},
		{"archived ops's Backlog", opsBacklog, domain.StatePatch{Name: ptr("Inbox")}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, s := newUpdateState()
			before := s.states[tt.id]
			got, err := uc.Execute(as(bob), tt.id, tt.p)
			want := before
			if tt.p.Name != nil {
				want.Name = *tt.p.Name
			}
			if tt.p.Color != nil {
				want.Color = *tt.p.Color
			}
			if tt.p.Group != nil {
				want.Group = *tt.p.Group
			}
			if tt.p.Description != nil {
				want.Description = *tt.p.Description
			}
			if tt.p.Sequence != nil {
				want.Sequence = *tt.p.Sequence
			}
			want.UpdatedAt = now
			if err != nil || got != want || s.states[tt.id] != want {
				t.Errorf("Execute() = %+v, %v, stored %+v; want %+v", got, err, s.states[tt.id], want)
			}
			calls := stateUpdated(bob, tt.id, tt.p, before.Group)
			if before.ProjectID == opsID {
				calls = append(stateLocked(tt.id, opsID, bob, domain.ActionStateUpdate), calls[6:]...)
			}
			if !slices.Equal(f.log.calls, calls) {
				t.Errorf("calls\n%q\nwant\n%q", f.log.calls, calls)
			}
		})
	}
}

// Refusals, each in its place, and no state changed: no caller, and values
// the domain refuses, before the transaction; a state that is not there, a
// workspace or project deleted while its lock waited, a state deleted or
// moved to ops meanwhile, and a caller who does not see web, each
// project.state_not_found; a member, the Authorizer's 403. Then Todo, the
// only state of its group, moved to another (project.state_last_in_group,
// before the change), and Done renamed to Review's name, the store's
// project.state_name_taken. A state changed as another is the write's own
// error.
func TestUpdateStateRefuses(t *testing.T) {
	upTo := func(n int) []string { return stateLocked(webTodo, webID, bob, domain.ActionStateUpdate)[:n] }
	rename := domain.StatePatch{Name: ptr("Later")}
	moveTodo := domain.StatePatch{Group: ptr(domain.GroupBacklog)}
	for _, tt := range []struct {
		name  string
		ctx   context.Context
		id    uuid.UUID
		p     domain.StatePatch
		set   func(f *writeFixture)
		want  error
		calls []string
	}{
		{"no caller", context.Background(), webTodo, rename, nil, shared.Unauthenticated(), nil},
		{"the triage group", as(bob), webTodo, domain.StatePatch{Group: ptr(domain.GroupTriage)}, nil,
			shared.Invalid(shared.FieldError{Field: "group", Code: shared.FieldNotAllowed}), nil},
		{"no state", as(bob), uuid.Nil(), rename, nil, domain.ErrStateNotFound, []string{"Begin", "StateByID " + uuid.Nil().String()}},
		{"acme deleted while its lock waited", as(bob), webTodo, rename, func(f *writeFixture) { f.workspaces.gone = true },
			domain.ErrStateNotFound, upTo(3)},
		{"web deleted while its lock waited", as(bob), webTodo, rename, func(f *writeFixture) { f.store.deleted = true },
			domain.ErrStateNotFound, upTo(4)},
		{"the state deleted while the locks waited", as(bob), webTodo, rename, func(f *writeFixture) { f.store.reread.gone = true },
			domain.ErrStateNotFound, upTo(5)},
		{"the state moved to ops", as(bob), webTodo, rename, func(f *writeFixture) { f.store.reread.project = opsID },
			domain.ErrStateNotFound, upTo(5)},
		{"a caller who does not see web", as(erin), webTodo, rename, nil, domain.ErrStateNotFound,
			stateLocked(webTodo, webID, erin, domain.ActionStateUpdate)},
		{"a member", as(alice), webTodo, rename, nil, shared.Forbidden(), stateLocked(webTodo, webID, alice, domain.ActionStateUpdate)},
		{"Todo, its group's only state, moved", as(bob), webTodo, moveTodo, nil, domain.ErrStateLastInGroup,
			append(upTo(6), fmt.Sprintf("CountGroupStates %s %s", webID, domain.GroupUnstarted))},
		{"Done renamed to Review's name", as(bob), webDone, domain.StatePatch{Name: ptr("Review")}, nil, domain.ErrStateNameTaken,
			stateUpdated(bob, webDone, domain.StatePatch{Name: ptr("Review")}, domain.GroupCompleted)},
		{"the change answered for another state", as(bob), webTodo, rename, func(f *writeFixture) { f.store.changedAs = uuid.NewV7() }, nil,
			stateUpdated(bob, webTodo, rename, domain.GroupUnstarted)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, s := newUpdateState()
			if tt.set != nil {
				tt.set(f)
			}
			before := maps.Clone(s.states)
			_, err := uc.Execute(tt.ctx, tt.id, tt.p)
			outcome{tt.name, tt.want, tt.calls}.check(t, err, f)
			if tt.name != "the change answered for another state" && !maps.Equal(s.states, before) {
				t.Errorf("the states after the refusal: %v; want them as they were, %v", s.states, before)
			}
		})
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after: Review moved to the backlog group, so
// that the count runs.
func TestUpdateStateReturnsEachFailure(t *testing.T) {
	move := domain.StatePatch{Group: ptr(domain.GroupBacklog)}
	all := stateUpdated(bob, webReview, move, domain.GroupStarted)
	fail := func(method string) func(f *writeFixture) {
		return func(f *writeFixture) { f.store.errs = map[string]error{method: errDisk} }
	}
	for _, tt := range []struct {
		name  string
		fail  func(f *writeFixture)
		calls int // how many of the change's calls ran
	}{
		{"the state's read", fail("StateByID"), 2},
		{"the workspace's lock", func(f *writeFixture) { f.workspaces.err = errDisk }, 3},
		{"the project's lock", fail("LockProject"), 4},
		{"the read under the locks", func(f *writeFixture) { f.store.reread.err = errDisk }, 5},
		{"the decision", func(f *writeFixture) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, 6},
		{"the count of the group", fail("CountGroupStates"), 7},
		{"the change", fail("UpdateState"), 9},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 9},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, _ := newUpdateState()
			tt.fail(f)
			_, err := uc.Execute(as(bob), webReview, move)
			outcome{tt.name, errDisk, all[:tt.calls]}.check(t, err, f)
		})
	}
}
````

`server/internal/modules/project/app/clock_test.go`（修改，1 处）：

````old server/internal/modules/project/app/clock_test.go
		}, stateCreated(bob, webID, review, 70000)},
````
````new server/internal/modules/project/app/clock_test.go
		}, stateCreated(bob, webID, review, 70000)},
		{"updateState", func() ([]string, error) {
			uc, f, _ := newUpdateState()
			_, err := uc.Execute(as(bob), webReview, domain.StatePatch{Group: ptr(domain.GroupBacklog)})
			return f.log.calls, err
		}, stateUpdated(bob, webReview, domain.StatePatch{Group: ptr(domain.GroupBacklog)}, domain.GroupStarted)},
````

- [ ] **Step 3: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/project/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`（`NewUpdateState` 在 Task 6 接线；它是导出的，lint 不报未使用）

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`（bootstrap 的操作名与规则表的核对：`state.update` 两边都有）。

- [ ] **Step 4: 提交**

```bash
git add server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/project/app/clock_test.go server/internal/modules/project/app/fakes_member_test.go server/internal/modules/project/app/fakes_state_test.go server/internal/modules/project/app/fakes_write_test.go server/internal/modules/project/app/state_ports.go server/internal/modules/project/app/update_state.go server/internal/modules/project/app/update_state_test.go server/internal/modules/project/domain/actions.go
```
```bash
git commit -m "feat(M3/P7a): updateState's use case

A change of a state takes the shared lock path: the state read for its
project and workspace, their locks, the state read again under them,
still of that project, and the decision, the state's own 404 for one
gone, moved or hidden. Moving the only state of its group to another
is project.state_last_in_group, counted under the project's lock; the
clock is read after the locks and the check. Its rule is the project's
admins', as createState's.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p7a.py` 的编号；"层"是它被发现的每一层，端到端是单独运行的故事）：

| 变异 | 改坏 | 必须失败的测试 | 层 |
|---|---|---|---|
| `s2-update-count` | `updateState` 不管 `CountGroupStates` 的失败 | `TestUpdateStateReturnsEachFailure` | 单元（按性质只在单元一层：真实的存储在这一步不失败） |
| `s2-update-update` | `updateState` 吞掉 `UpdateState` 的失败 | `TestUpdateStateRefuses`、`TestUpdateStateReturnsEachFailure` | 单元（按性质只在单元一层：真实的存储在这一步不失败） |
| `s22-updated-id` | `updateState` 不核对改过的状态的 id | `TestUpdateStateRefuses` | 单元（按性质只在单元一层：真实的存储回答不了别的键） |
| `c-update-early` | `updateState` 在取锁之前读时钟 | `TestEachWriteReadsTheClockUnderItsLock`、`TestUpdateState`、`TestUpdateStateRefuses`、`TestUpdateStateReturnsEachFailure`、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`（Task 9 起） | 单元；组合 |
| `g-update-unchecked` | `updateState` 不查组里还剩几个 | `TestEachWriteReadsTheClockUnderItsLock`、`TestUpdateState`、`TestUpdateStateRefuses`、`TestUpdateStateReturnsEachFailure`、`TestPermissionMatrix`、`TestStateWritesOnOneProjectSerialize`（Task 9 起）、P6（Task 10 起） | 单元；组合；端到端 |
| `g-update-same-group` | `updateState` 改到原来的组也查 | `TestUpdateState`、`TestEachStateWriteChangesItsRowsAlone`（Task 9 起） | 单元；组合 |
| `g-update-new-group` | `updateState` 数要去的组 | `TestEachWriteReadsTheClockUnderItsLock`、`TestUpdateState`、`TestUpdateStateRefuses`、`TestUpdateStateReturnsEachFailure`、`TestStateWritesOnOneProjectSerialize`（Task 9 起）、P6（Task 10 起） | 单元；组合；端到端 |
| `g-update-n` | `updateState` 把自己也算作留下的 | `TestUpdateStateRefuses`、`TestPermissionMatrix`、`TestStateWritesOnOneProjectSerialize`（Task 9 起）、P6（Task 10 起） | 单元；组合；端到端 |
| `s25-move-counted-unlocked` | `updateState` 在取锁之前、在另一个事务里数组 | `TestStateWritesOnOneProjectSerialize`（Task 9 起）、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 7 起） | 组合 |
| `s5-update-members` | 规则表：`state.update` 也给项目成员 | `TestEveryRuleDecidesItsCells`、`TestPermissionMatrix`、P6（Task 10 起） | 单元；组合；端到端 |

---

### Task 6: `updateState` 的接口和组合：契约、HTTP、接线、矩阵、锁的阶梯、盖戳、连接；`project.state_not_found`、`project.state_last_in_group`

**Files:**
- Create: `server/internal/modules/project/adapter/http/state_writes_test.go`
- Modify: `api/modules/project.yaml`、`api/openapi.yaml`、`server/internal/bootstrap/permission_matrix_columns_test.go`、`server/internal/bootstrap/permission_matrix_seeded_test.go`、`server/internal/bootstrap/permission_matrix_states_test.go`、`server/internal/bootstrap/permission_matrix_targets_test.go`、`server/internal/bootstrap/project_connection_test.go`、`server/internal/bootstrap/project_write_locks_test.go`、`server/internal/bootstrap/project_writes_test.go`、`server/internal/modules/project/adapter/http/handler.go`、`server/internal/modules/project/adapter/http/handler_test.go`、`server/internal/modules/project/adapter/http/states.go`、`server/internal/modules/project/module.go`、`web/apps/web/helpers/authentication.helper.ts`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/project/adapter/http/gen/bodyshape.gen.go`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.7；M3 设计 3.4、3.6、3.17、3.19、9.2）：
  - 契约：`PATCH /api/v0/states/{state_id}`（`updateState`，200 `State`；`x-problem-codes: [validation_failed, project.state_not_found, forbidden, project.state_name_taken, project.state_last_in_group]`）；参数 `StateID`；`StateUpdate`。
  - HTTP：`UseCases.UpdateState`、`UpdateState` 处理函数（给了的字段，组原样，没给的为 `nil`）。`project.New` 接上 `app.NewUpdateState`。前端文案：`project.state_not_found`、`project.state_last_in_group`。
  - 矩阵：`ofState`、`ofArchivedState`、`toState`（瞄准列的项目下的这个名称的状态）；`updateState` 的五行（改名、组里最后一个改走 409、名称已占用 409、分诊状态每一列 404、已归档项目的小表），`renamesTheState`。列的核对（`permission_matrix_columns_test.go`、`permission_matrix_targets_test.go`）认得瞄准状态的格子：状态的行的目标是它的列的项目下的状态。
  - 最先锁工作区的测试推广到按行寻址的写（`project_write_locks_test.go`）：`underRow`、`rowPaths`（`/project-members/`、`/states/`）、`stateNamed`；第一步探测它的这一行（状态行）没有被持有。盖戳、连接加 `updateState`。

**Tests:**
- HTTP：`TestUpdateState`（`state_writes_test.go`：给了的字段、组原样（`triage` 也是）、没给的 `nil`；200）、`TestUpdateStateHoldsTheBodyToItsStructure`（多余的字段、类型不对、`null`：400）、`TestUpdateStateRefusals`（空的名称 422、没有这一行 404、项目成员 403、名称已占用 409、组里唯一的状态改走 409、失败 500）。
- 组合：`TestPermissionMatrix` 的 `updateState` 五行；`TestMatrixViolationsCatchesEachColumnGap` 的说明加"或它下面种下的一行"；`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（`updateState` 一行：等 acme 时 Web、它的状态行都能 `FOR UPDATE NOWAIT`；等 Ops 时 acme 只是 `FOR SHARE`，Ops 的状态行还没有锁）；`TestTheWritesOnAProjectStampTheirRequest`（改名的行先盖上 bob 的戳，之后由 alice）；`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（建出的状态改名）。

- [ ] **Step 1: 契约**

`api/modules/project.yaml`（修改，3 处）：

````old api/modules/project.yaml
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/me/projects/{project_id}/preferences:
````
````new api/modules/project.yaml
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/states/{state_id}:
    parameters:
      - $ref: '#/components/parameters/StateID'
    patch:
      operationId: updateState
      tags: [project]
      summary: Change a state
      description: >-
        For the project's admins, and its members who are the workspace's
        admins; an archived project's states change as any other's. The
        fields given change and the others stay; the values follow
        createState's rules, and the sequence is any number
        (validation_failed), checked before the state is looked at. A state
        that does not exist or is deleted, the intake's triage state, and a
        state whose project the caller does not see answer
        project.state_not_found; a caller who sees the project but may not
        change its states, forbidden, whether or not the state is the
        default or the last of its group. Moving the only state of its group
        to another group is refused (project.state_last_in_group): every
        group keeps a state. Its name may not be another undeleted state's
        of the project, the intake's triage state's too
        (project.state_name_taken). The role is decided after the workspace
        and project rows are locked.
      security: [{bearer: []}]
      x-problem-codes: [validation_failed, project.state_not_found, forbidden, project.state_name_taken, project.state_last_in_group]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/StateUpdate'
      responses:
        '200':
          description: The state as changed.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/State'
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
    StateID:
      name: state_id
      in: path
      required: true
      description: A state's id (State.id).
````

````old api/modules/project.yaml
          $ref: '#/components/schemas/StateGroup'
        description:
          type: string

````
````new api/modules/project.yaml
          $ref: '#/components/schemas/StateGroup'
        description:
          type: string
    StateUpdate:
      description: Changes the fields it names; a field left out keeps its value.
      type: object
      additionalProperties: false
      properties:
        name:
          description: 1–255 characters, not blank; another undeleted state of the project may not have it.
          type: string
        color:
          description: 1–255 characters, not blank.
          type: string
        group:
          $ref: '#/components/schemas/StateGroup'
        description:
          type: string
        sequence:
          description: The state's place among the project's states, the lowest first.
          type: number

````

`api/openapi.yaml`（修改，1 处）：

````old api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1projects~1{project_id}~1states'
````
````new api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1projects~1{project_id}~1states'
  /api/v0/states/{state_id}:
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1states~1{state_id}'
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `ed202e706c7aadd11c9ddb77002705cf042dcc22fe64bb768d7e56def2ad3290` | 2663 | `api/dist/openapi.yaml` |
| `c4236d98c3293a387f6e14f6f77eddd33ad8c7da850bc28afbfaf4b12c07e776` | 74 | `server/internal/modules/project/adapter/http/gen/bodyshape.gen.go` |
| `63af8a46230f273cc825623f764b13d26b2a39d14b391a1f146ad6000b548ae6` | 2603 | `server/internal/modules/project/adapter/http/gen/server.gen.go` |
| `7b3721d19175c8621f7a055f43a3b8ff00820c553e55ce68f801bf22f20d5fd3` | 2907 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml web/packages/api-client/src/schema.gen.ts server/internal/modules/project/adapter/http/gen/server.gen.go server/internal/modules/project/adapter/http/gen/bodyshape.gen.go`
Expected: 与上表相同。

- [ ] **Step 2: HTTP、接线、文案**

`server/internal/modules/project/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/project/adapter/http/handler.go
}

// UseCases are the use cases behind the module's operations.
````
````new server/internal/modules/project/adapter/http/handler.go
}

// UpdateStateUseCase is app.UpdateState.
type UpdateStateUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, p domain.StatePatch) (domain.State, error)
}

// UseCases are the use cases behind the module's operations.
````

````old server/internal/modules/project/adapter/http/handler.go
	CreateState       CreateStateUseCase
````
````new server/internal/modules/project/adapter/http/handler.go
	CreateState       CreateStateUseCase
	UpdateState       UpdateStateUseCase
````

`server/internal/modules/project/adapter/http/states.go`（修改，1 处）：

````old server/internal/modules/project/adapter/http/states.go
}

// state is s as the API shows it.
````
````new server/internal/modules/project/adapter/http/states.go
}

// UpdateState serves PATCH /api/v0/states/{state_id}: the fields given go
// to the use case, a group as given, which the domain checks (422).
func (h handler) UpdateState(ctx context.Context, req gen.UpdateStateRequestObject) (gen.UpdateStateResponseObject, error) {
	p := domain.StatePatch{Name: req.Body.Name, Color: req.Body.Color, Description: req.Body.Description, Sequence: req.Body.Sequence}
	if req.Body.Group != nil {
		group := domain.StateGroup(*req.Body.Group)
		p.Group = &group
	}
	s, err := h.uc.UpdateState.Execute(ctx, req.StateID, p)
	if err != nil {
		return nil, err
	}
	return gen.UpdateState200JSONResponse(state(s)), nil
}

// state is s as the API shows it.
````

`server/internal/modules/project/adapter/http/handler_test.go`（修改，3 处）：

````old server/internal/modules/project/adapter/http/handler_test.go
	createState  *fakeCreateState
````
````new server/internal/modules/project/adapter/http/handler_test.go
	createState  *fakeCreateState
	updateState  *fakeUpdateState
````

````old server/internal/modules/project/adapter/http/handler_test.go
		f.createState = &fakeCreateState{}
	}
````
````new server/internal/modules/project/adapter/http/handler_test.go
		f.createState = &fakeCreateState{}
	}
	if f.updateState == nil {
		f.updateState = &fakeUpdateState{}
	}
````

````old server/internal/modules/project/adapter/http/handler_test.go
		CreateState: f.createState})
````
````new server/internal/modules/project/adapter/http/handler_test.go
		CreateState: f.createState, UpdateState: f.updateState})
````

`server/internal/modules/project/adapter/http/state_writes_test.go`（新文件，109 行）：

````file server/internal/modules/project/adapter/http/state_writes_test.go
package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeUpdateState is updateState: each call is recorded as "caller id",
// with what it got; it answers answer, or err.
type fakeUpdateState struct {
	calls  []string
	got    []domain.StatePatch
	answer domain.State
	err    error
}

func (f *fakeUpdateState) Execute(ctx context.Context, id uuid.UUID, p domain.StatePatch) (domain.State, error) {
	f.calls = append(f.calls, caller(ctx)+" "+id.String())
	f.got = append(f.got, p)
	return f.answer, f.err
}

// The answers of the writes on a state, as the contract declares them.
const (
	stateNotFoundJSON = `{"status":404,"code":"project.state_not_found","title":"Not Found",` +
		`"detail":"The state does not exist, cannot be changed through this API, or you cannot see its project."}`
	stateLastInGroupJSON = `{"status":409,"code":"project.state_last_in_group","title":"Conflict",` +
		`"detail":"The state is the only one of its group, and every group keeps one; add another to the group first."}`
)

// PATCH goes to the use case for the caller and the path's state, with the
// fields the body gives, a group as given, and nil for each it does not;
// the answer is 200 with the state the use case answers.
func TestUpdateState(t *testing.T) {
	path := "/api/v0/states/" + webReview.ID.String()
	for _, tt := range []struct {
		body string
		want domain.StatePatch
	}{
		{`{}`, domain.StatePatch{}},
		{`{"name":"Review","color":"#F59E0B","group":"started","description":"Waiting for a review","sequence":70000.5}`,
			domain.StatePatch{Name: ptr("Review"), Color: ptr("#F59E0B"), Group: ptr(domain.GroupStarted), Description: ptr("Waiting for a review"),
				Sequence: ptr(70000.5)}},
		{`{"group":"triage"}`, domain.StatePatch{Group: ptr(domain.GroupTriage)}},
		{`{"sequence":-1}`, domain.StatePatch{Sequence: ptr(-1.0)}},
	} {
		update := &fakeUpdateState{answer: webReview}
		h := newServer(t, fakes{updateState: update})
		if res, body := do(t, h, request(http.MethodPatch, path, "alice", tt.body)); res.StatusCode != http.StatusOK || body != webReviewJSON+"\n" {
			t.Errorf("PATCH %s = %d %s, want 200 %s", tt.body, res.StatusCode, body, webReviewJSON)
		}
		if want := []string{"alice " + webReview.ID.String()}; !slices.Equal(update.calls, want) || !reflect.DeepEqual(update.got,
			[]domain.StatePatch{tt.want}) {
			t.Errorf("PATCH %s: calls %q with %+v; want %q with %+v", tt.body, update.calls, update.got, want, tt.want)
		}
	}
}

// A body with a field it may not have, a field of another type, or null:
// refused as bad_request before the use case.
func TestUpdateStateHoldsTheBodyToItsStructure(t *testing.T) {
	update := &fakeUpdateState{}
	h := newServer(t, fakes{updateState: update})
	for _, body := range []string{`{"default":true}`, `{"project_id":"0199a2b4-0000-7000-8000-0000000000a1"}`, `{"name":1}`, `{"sequence":"1"}`,
		`{"name":null}`, `{"group":null}`} {
		res, got := do(t, h, request(http.MethodPatch, "/api/v0/states/"+webReview.ID.String(), "alice", body))
		var problem struct{ Code string }
		if err := json.Unmarshal([]byte(got), &problem); err != nil || res.StatusCode != http.StatusBadRequest || problem.Code != "bad_request" {
			t.Errorf("PATCH %s = %d %s, want 400 bad_request", body, res.StatusCode, got)
		}
	}
	if len(update.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", update.calls)
	}
}

// The use case's refusals, as the contract declares them, and its failure.
func TestUpdateStateRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"an empty name", shared.Invalid(shared.FieldError{Field: "name", Code: shared.FieldTooShort, Message: "must not be empty"}),
			http.StatusUnprocessableEntity, `{"status":422,"code":"validation_failed","title":"Unprocessable Entity",` +
				`"detail":"The request has invalid values.","errors":[{"field":"name","code":"too_short","message":"must not be empty"}]}`},
		{"no state", domain.ErrStateNotFound, http.StatusNotFound, stateNotFoundJSON},
		{"a project member", shared.Forbidden(), http.StatusForbidden, forbiddenJSON},
		{"a name taken", domain.ErrStateNameTaken, http.StatusConflict,
			`{"status":409,"code":"project.state_name_taken","title":"Conflict","detail":"A state of the project has this name."}`},
		{"its group's only state moved", domain.ErrStateLastInGroup, http.StatusConflict, stateLastInGroupJSON},
		{"a failure", errGone, http.StatusInternalServerError, internalErrorJSON},
	} {
		h := newServer(t, fakes{updateState: &fakeUpdateState{err: tt.err}})
		res, body := do(t, h, request(http.MethodPatch, "/api/v0/states/"+webReview.ID.String(), "alice", `{"name":""}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: PATCH = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
````

`server/internal/modules/project/module.go`（修改，2 处）：

````old server/internal/modules/project/module.go
// member's display settings, creating a project's states, carries out the
// workspace module's cascades on the projects (ProjectCascade), and offers
// the access module its reads of a project (ProjectAccess) and the
// workspace module its count of an account's ended project memberships
// (ProjectMembershipCounts).
````
````new server/internal/modules/project/module.go
// member's display settings, creating and changing a project's states,
// carries out the workspace module's cascades on the projects
// (ProjectCascade), and offers the access module its reads of a project
// (ProjectAccess) and the workspace module its count of an account's ended
// project memberships (ProjectMembershipCounts).
````

````old server/internal/modules/project/module.go
		CreateState:       app.NewCreateState(locks, store, d.Tx, d.Clock),
````
````new server/internal/modules/project/module.go
		CreateState:       app.NewCreateState(locks, store, d.Tx, d.Clock),
		UpdateState:       app.NewUpdateState(locks, store, d.Tx, d.Clock),
````

`web/apps/web/helpers/authentication.helper.ts`（修改，1 处）：

````old web/apps/web/helpers/authentication.helper.ts
  "project.state_name_taken": "auth.errors.project_state_name_taken",
````
````new web/apps/web/helpers/authentication.helper.ts
  "project.state_name_taken": "auth.errors.project_state_name_taken",
  "project.state_not_found": "auth.errors.project_state_not_found",
  "project.state_last_in_group": "auth.errors.project_state_last_in_group",
````

`web/packages/i18n/src/locales/en/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/auth.json
      "project_state_name_taken": "A state of this project already has this name.",
````
````new web/packages/i18n/src/locales/en/auth.json
      "project_state_name_taken": "A state of this project already has this name.",
      "project_state_not_found": "The state does not exist, cannot be changed through this API, or you cannot see its project.",
      "project_state_last_in_group": "This is the only state of its group, and every group keeps one. Add another state to the group first.",
````

`web/packages/i18n/src/locales/zh-CN/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/auth.json
      "project_state_name_taken": "这个项目里已有同名的状态。",
````
````new web/packages/i18n/src/locales/zh-CN/auth.json
      "project_state_name_taken": "这个项目里已有同名的状态。",
      "project_state_not_found": "状态不存在、不能经这个接口修改，或者你看不到它所在的项目。",
      "project_state_last_in_group": "这是它所在分组里唯一的状态，每个分组都要保留一个状态。请先给这个分组添加另一个状态。",
````

- [ ] **Step 3: 矩阵和组合测试**

`server/internal/bootstrap/permission_matrix_seeded_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_seeded_test.go
// projectOfMember is the key of the project of the seeded project
// membership id, false for an id no seeded project membership has.
func (s seeded) projectOfMember(id uuid.UUID) (string, bool) {
	for key, seededID := range s.projectMembers {
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
// projectOfRow is the key of the project of id, a row of rows, which are
// keyed by the project's key, then "|": false for an id none of rows has.
func projectOfRow(rows map[string]uuid.UUID, id uuid.UUID) (string, bool) {
	for key, seededID := range rows {
````

`server/internal/bootstrap/permission_matrix_columns_test.go`（修改，4 处）：

````old server/internal/bootstrap/permission_matrix_columns_test.go
	"net/http"
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
	"net/http"
	"path"
````

````old server/internal/bootstrap/permission_matrix_columns_test.go
// projectColumns, each cell aims at its column's project, and a parameter
// listed as not a target leaves the path's others checked.
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
// projectColumns, each cell aims at its column's project, or at a row
// seeded under it (a membership, a state), and a parameter listed as not a
// target leaves the path's others checked.
````

````old server/internal/bootstrap/permission_matrix_columns_test.go
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
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
	// A row under a project that a path names by its id (underProject: a
	// membership, a state) is one seeded in the column's project, from a
	// column of a project table.
	for _, under := range []struct {
		op      apitest.Operation
		what    string
		named   func(c caller, s seeded) uuid.UUID // the row each column's cell names
		another uuid.UUID                          // a row of acme's public project, which WM-私's is not
	}{
		{apitest.Operation{ID: "updateProjectMember", Tags: []string{"project"}, Method: http.MethodPatch,
			Path: "/api/v0/project-members/{project_member_id}"}, "membership", func(c caller, s seeded) uuid.UUID {
			key, member := projectMemberOf(c)
			return s.projectMember(key, member)
		}, s.projectMember("acme/public", callerProjectMember)},
		{apitest.Operation{ID: "updateState", Tags: []string{"project"}, Method: http.MethodPatch, Path: "/api/v0/states/{state_id}"}, "state",
			func(c caller, s seeded) uuid.UUID { return s.state(projectOf(c), "Todo") }, s.state("acme/public", "Todo")},
	} {
		prefix, param := path.Split(under.op.Path)
		changes := matrixRow{op: under.op.ID, write: true, columns: projectColumns, cells: cells,
			request: func(c caller, s seeded) (string, string, string) {
				return under.op.Method, prefix + under.named(c, s).String(), ""
			}}
		// privateNames is changes, but WM-私's cell names id.
		privateNames := func(id uuid.UUID) matrixRow {
			r := changes
			r.request = func(c caller, s seeded) (string, string, string) {
				if c == callerMemberPrivate {
					return under.op.Method, prefix + id.String(), ""
				}
				return changes.request(c, s)
````

````old server/internal/bootstrap/permission_matrix_columns_test.go
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
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
			return r
		}
		inWorkspaceRow := changes
		inWorkspaceRow.columns, inWorkspaceRow.cells = nil, every(cellOK)
		var noTable []string
		for _, c := range []caller{callerAdmin, callerMember, callerGuest} {
			noTable = append(noTable, fmt.Sprintf("row %s, %s: %s from a column of no project table (projectTables): "+
				"a project's row names its columns", under.op.ID, c, param))
		}
		elsewhere := func(id uuid.UUID) []string {
			return []string{fmt.Sprintf("row %s, %s: %s %s is no %s seeded in its column's project acme/private", under.op.ID, callerMemberPrivate,
				param, id, under.what)}
		}
		for _, tt := range []struct {
			name string
			row  matrixRow
			want []string
		}{
			{"rows of their columns' projects", changes, nil},
			{"a row of another column's project", privateNames(under.another), elsewhere(under.another)},
			{"an id no seeded row has", privateNames(uuid.Nil()), elsewhere(uuid.Nil())},
			{"a row in a workspace-level row", inWorkspaceRow, noTable},
		} {
			if got := matrixViolations(append(ops, under.op), listed, []matrixRow{row, checks, tt.row}, s, nil); !slices.Equal(got, tt.want) {
				t.Errorf("%s, %s: %q, want %q", under.what, tt.name, got, tt.want)
			}
````

`server/internal/bootstrap/permission_matrix_targets_test.go`（修改，5 处）：

````old server/internal/bootstrap/permission_matrix_targets_test.go
}

// targetViolation is what is wrong with where path, a path of pattern that
````
````new server/internal/bootstrap/permission_matrix_targets_test.go
}

// underProject are the rows under a project that a path names by their
// id, by their parameter: what each is called, and the seeded rows among
// which it must be, keyed by their project's key.
var underProject = map[string]struct {
	what string
	rows func(s seeded) map[string]uuid.UUID
}{
	"{project_member_id}": {"membership", func(s seeded) map[string]uuid.UUID { return s.projectMembers }},
	"{state_id}":          {"state", func(s seeded) map[string]uuid.UUID { return s.states }},
}

// targetViolation is what is wrong with where path, a path of pattern that
````

````old server/internal/bootstrap/permission_matrix_targets_test.go
// project named by its id ({project_id}) must be projectOf(c)'s, and a
// project membership ({project_member_id}) one seeded in projectOf(c),
// each from a column of a project table (projectTables): a project's
// operation in a workspace-level row, or the only admin's, would leave the
// project level's own columns unasked.
````
````new server/internal/bootstrap/permission_matrix_targets_test.go
// project named by its id ({project_id}) must be projectOf(c)'s, and a row
// under a project (underProject: a project membership, a state) one seeded
// in projectOf(c), each from a column of a project table (projectTables):
// a project's operation in a workspace-level row, or the only admin's,
// would leave the project level's own columns unasked.
````

````old server/internal/bootstrap/permission_matrix_targets_test.go
		case segment == "{project_member_id}":
````
````new server/internal/bootstrap/permission_matrix_targets_test.go
		case underProject[segment].rows != nil:
````

````old server/internal/bootstrap/permission_matrix_targets_test.go
				return "{project_member_id} from a column of no project table (projectTables): a project's row names its columns"
````
````new server/internal/bootstrap/permission_matrix_targets_test.go
				return segment + " from a column of no project table (projectTables): a project's row names its columns"
````

````old server/internal/bootstrap/permission_matrix_targets_test.go
			if key, isMember := s.projectOfMember(id); err != nil || !isMember || key != projectOf(c) {
				return fmt.Sprintf("{project_member_id} %s is no membership seeded in its column's project %s", got[i], projectOf(c))
````
````new server/internal/bootstrap/permission_matrix_targets_test.go
			if key, isRow := projectOfRow(underProject[segment].rows(s), id); err != nil || !isRow || key != projectOf(c) {
				return fmt.Sprintf("%s %s is no %s seeded in its column's project %s", segment, got[i], underProject[segment].what, projectOf(c))
````

`server/internal/bootstrap/permission_matrix_states_test.go`（修改，4 处）：

````old server/internal/bootstrap/permission_matrix_states_test.go
// creating them, under the project of each column, its seeded states those
// of matrixStates.
````
````new server/internal/bootstrap/permission_matrix_states_test.go
// creating them, under the project of each column, and the writes on one,
// naming it by its id (/states/{state_id}) among its column's project's
// seeded states, those of matrixStates.
````

````old server/internal/bootstrap/permission_matrix_states_test.go
var cellStateNameTaken = cell{http.StatusConflict, "project.state_name_taken"}
````
````new server/internal/bootstrap/permission_matrix_states_test.go
var (
	cellStateNameTaken   = cell{http.StatusConflict, "project.state_name_taken"}
	cellStateNotFound    = cell{http.StatusNotFound, "project.state_not_found"}
	cellStateLastInGroup = cell{http.StatusConflict, "project.state_last_in_group"}
)

// ofState are the cells of a row of a write on a state: the answers of PA,
// PM, PG, PM+WA, WA- and WM-公, and project.state_not_found for the
// columns that do not see their project: WM-私, WG-, P-前 and X, whose
// state in gone's project is deleted with it.
func ofState(pa, pm, pg, pmwa, wa, wm cell) map[caller]cell {
	return map[caller]cell{callerProjectAdmin: pa, callerProjectMember: pm, callerProjectGuest: pg, callerMemberAndAdmin: pmwa,
		callerAdminOnly: wa, callerMemberPublic: wm, callerMemberPrivate: cellStateNotFound, callerGuestOnly: cellStateNotFound,
		callerBefore: cellStateNotFound, callerNever: cellStateNotFound, callerRemoved: cellStateNotFound, callerDeleted: cellStateNotFound}
}

// ofArchivedState are the cells of a row of a write on a state of the
// archived project: the answers of PA and of the workspace's member, who
// sees the project, and project.state_not_found for X, who does not.
func ofArchivedState(pa, wm cell) map[caller]cell {
	return map[caller]cell{callerArchivedAdmin: pa, callerArchivedMember: wm, callerArchivedNever: cellStateNotFound}
}

// toState is the request of a row whose callers each send method, with
// body, to the state name of their column's project, after its id.
func toState(method, after, name, body string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return method, "/api/v0/states/" + s.state(projectOf(c), name).String() + after, body
	}
}
````

````old server/internal/bootstrap/permission_matrix_states_test.go
			cells: ofArchived(cellCreated, cellForbidden), check: createsTheState},
````
````new server/internal/bootstrap/permission_matrix_states_test.go
			cells: ofArchived(cellCreated, cellForbidden), check: createsTheState},
		// As createState: Todo renamed.
		{op: "updateState", write: true, columns: projectColumns, request: toState(http.MethodPatch, "", "Todo", `{"name":"Next"}`),
			cells: ofState(cellOK, cellForbidden, cellForbidden, cellOK, cellForbidden, cellForbidden), check: renamesTheState},
		// Todo, the only state of its group, moved to another: the 409 comes
		// after the decision (M3 design 6.7).
		{op: "updateState", variant: "the last of its group moved", write: true, columns: projectColumns,
			request: toState(http.MethodPatch, "", "Todo", `{"group":"backlog"}`),
			cells:   ofState(cellStateLastInGroup, cellForbidden, cellForbidden, cellStateLastInGroup, cellForbidden, cellForbidden)},
		{op: "updateState", variant: "a name taken", write: true, columns: projectColumns, request: toState(http.MethodPatch, "", "Todo", `{"name":"Done"}`),
			cells: ofState(cellStateNameTaken, cellForbidden, cellForbidden, cellStateNameTaken, cellForbidden, cellForbidden)},
		// The intake's triage state is none of the states (M3 design 3.17):
		// every column's 404, its project's admins' too.
		{op: "updateState", variant: "the triage state", write: true, columns: projectColumns,
			request: toState(http.MethodPatch, "", "Triage", `{"name":"Next"}`), cells: ofState(cellStateNotFound, cellStateNotFound,
				cellStateNotFound, cellStateNotFound, cellStateNotFound, cellStateNotFound)},
		{op: "updateState", variant: "archived", write: true, columns: archivedColumns, request: toState(http.MethodPatch, "", "Todo", `{"name":"Next"}`),
			cells: ofArchivedState(cellOK, cellForbidden), check: renamesTheState},
````

````old server/internal/bootstrap/permission_matrix_states_test.go
		t.Errorf("%s creates %s; want QA in %s, completed, at 70000, not the default", c, answer, projectOf(c))
	}
}

````
````new server/internal/bootstrap/permission_matrix_states_test.go
		t.Errorf("%s creates %s; want QA in %s, completed, at 70000, not the default", c, answer, projectOf(c))
	}
}

// renamesTheState: the column's project's Todo, renamed Next, as stored.
func renamesTheState(t *testing.T, c caller, s seeded, answer string) {
	var st struct {
		ID        uuid.UUID `json:"id"`
		ProjectID uuid.UUID `json:"project_id"`
		Name      string    `json:"name"`
		Group     string    `json:"group"`
	}
	decodeAnswer(t, answer, &st)
	if st.ID != s.state(projectOf(c), "Todo") || st.ProjectID != s.project(projectOf(c)) || st.Name != "Next" || st.Group != "unstarted" {
		t.Errorf("%s renames %s; want %s's Todo, unstarted, named Next", c, answer, projectOf(c))
	}
}

````

`server/internal/bootstrap/project_write_locks_test.go`（修改，20 处）：

````old server/internal/bootstrap/project_write_locks_test.go
// operationId: the request on the project, or on a membership of it, by
// alice unless by names another sender.
````
````new server/internal/bootstrap/project_write_locks_test.go
// operationId: the request on the project, or on a row under it, by alice
// unless by names another sender.
````

````old server/internal/bootstrap/project_write_locks_test.go
	// path: %s the project's id, or the membership's for a path of one
	// (/api/v0/project-members/); body: %s the target's id.
````
````new server/internal/bootstrap/project_write_locks_test.go
	// path: %s the project's id, or the row's for a path of one
	// (rowPaths); body: %s the target's id.
````

````old server/internal/bootstrap/project_write_locks_test.go
	// member are the accounts, one a phase, whose membership of the project
	// the write changes: a path of a membership names it, and its row is
	// probed as the project's is.
	member [2]string
````
````new server/internal/bootstrap/project_write_locks_test.go
	// row is the row under the project the write changes, one a phase: a
	// path of one names it, and it is probed as the project's row is.
	row underRow
````

````old server/internal/bootstrap/project_write_locks_test.go
// param is the parameter w's path names: a membership's id for a path of
// one, else the project's.
````
````new server/internal/bootstrap/project_write_locks_test.go
// underRow is a row under a project, one a phase: in table, the row whose
// column by is the phase's key, an account's name for a membership
// (member_id), a state's name for a state (name).
type underRow struct {
	table, by string
	keys      [2]string
}

// membershipsOf is the membership of a's account in the first phase's
// project and b's in the second's.
func membershipsOf(a, b string) underRow {
	return underRow{"project_members", "member_id", [2]string{a, b}}
}

// stateNamed is the state name of each phase's project.
func stateNamed(name string) underRow { return underRow{"states", "name", [2]string{name, name}} }

// rowPaths are the paths that name a row under a project by its id, by
// their beginning, and the parameter each names it by.
var rowPaths = map[string]string{"/api/v0/project-members/": "{project_member_id}", "/api/v0/states/": "{state_id}"}

// param is the parameter w's path names: a row's id for a path of one
// (rowPaths), else the project's.
````

````old server/internal/bootstrap/project_write_locks_test.go
	if strings.HasPrefix(w.path, "/api/v0/project-members/") {
		return "{project_member_id}"
````
````new server/internal/bootstrap/project_write_locks_test.go
	for prefix, param := range rowPaths {
		if strings.HasPrefix(w.path, prefix) {
			return param
		}
````

````old server/internal/bootstrap/project_write_locks_test.go
		targets: [2]string{"bob", "carol"}, member: [2]string{"bob", "carol"}},
````
````new server/internal/bootstrap/project_write_locks_test.go
		targets: [2]string{"bob", "carol"}, row: membershipsOf("bob", "carol")},
````

````old server/internal/bootstrap/project_write_locks_test.go
		member: [2]string{"dave", "erin"}},
````
````new server/internal/bootstrap/project_write_locks_test.go
		row: membershipsOf("dave", "erin")},
````

````old server/internal/bootstrap/project_write_locks_test.go
		member: [2]string{"bob", "carol"}},
````
````new server/internal/bootstrap/project_write_locks_test.go
		row: membershipsOf("bob", "carol")},
````

````old server/internal/bootstrap/project_write_locks_test.go
		want: http.StatusCreated},
````
````new server/internal/bootstrap/project_write_locks_test.go
		want: http.StatusCreated},
	// The state made before, renamed.
	{op: "updateState", method: http.MethodPatch, path: "/api/v0/states/%s", body: `{"name":"Checked"}`, want: http.StatusOK, row: stateNamed("QA")},
````

````old server/internal/bootstrap/project_write_locks_test.go
// by a row under it (P5b's /project-members/{project_member_id}, P7's
````
````new server/internal/bootstrap/project_write_locks_test.go
// by a row under it (P5b's /project-members/{project_member_id}, P7a's
````

````old server/internal/bootstrap/project_write_locks_test.go
//     its target's membership of acme, nor the membership of Web it
//     changes: a FOR UPDATE NOWAIT of each succeeds.
````
````new server/internal/bootstrap/project_write_locks_test.go
//     its target's membership of acme, nor the row under Web it changes, a
//     membership or a state: a FOR UPDATE NOWAIT of each succeeds.
````

````old server/internal/bootstrap/project_write_locks_test.go
//     55P03), but not the membership of Ops it changes, which comes after
//     the project.
````
````new server/internal/bootstrap/project_write_locks_test.go
//     55P03), but not the row under Ops it changes, which comes after the
//     project.
````

````old server/internal/bootstrap/project_write_locks_test.go
		for _, name := range slices.Concat(w.targets[:], w.by[:], w.member[:]) {
````
````new server/internal/bootstrap/project_write_locks_test.go
		accounts := slices.Concat(w.targets[:], w.by[:])
		if w.row.table == "project_members" {
			accounts = append(accounts, w.row.keys[:]...)
		}
		for _, name := range accounts {
````

````old server/internal/bootstrap/project_write_locks_test.go
	membership := "SELECT 1 FROM workspace_members WHERE workspace_id = $1 AND member_id = $2 AND deleted_at IS NULL FOR UPDATE"
	memberRow := "SELECT 1 FROM project_members WHERE id = $1 FOR UPDATE"
````
````new server/internal/bootstrap/project_write_locks_test.go
	membership := "SELECT 1 FROM workspace_members WHERE workspace_id = $1 AND member_id = $2 AND deleted_at IS NULL FOR UPDATE"
````

````old server/internal/bootstrap/project_write_locks_test.go
				target, member, token, body, named := w.targets[phase], w.member[phase], alice, w.body, project
````
````new server/internal/bootstrap/project_write_locks_test.go
				target, key, token, body, named := w.targets[phase], w.row.keys[phase], alice, w.body, project
````

````old server/internal/bootstrap/project_write_locks_test.go
				// The membership the write changes, its row probed, and named
				// by a path of one.
````
````new server/internal/bootstrap/project_write_locks_test.go
				// The row under the project the write changes, probed, and
				// named by a path of one.
````

````old server/internal/bootstrap/project_write_locks_test.go
				if member != "" {
					if err := pool.QueryRow(pgtest.Soon(t), "SELECT id FROM project_members WHERE project_id = $1 AND member_id = $2 AND deleted_at IS NULL",
						project, ids[member]).Scan(&row); err != nil {
						t.Fatalf("%s's membership of %s: %v", member, on, err)
````
````new server/internal/bootstrap/project_write_locks_test.go
				rowLock := "SELECT 1 FROM " + w.row.table + " WHERE id = $1 FOR UPDATE"
				if key != "" {
					var by any = key
					if w.row.by == "member_id" {
						by = ids[key]
````

````old server/internal/bootstrap/project_write_locks_test.go
					if w.param() == "{project_member_id}" {
````
````new server/internal/bootstrap/project_write_locks_test.go
					if err := pool.QueryRow(pgtest.Soon(t), "SELECT id FROM "+w.row.table+" WHERE project_id = $1 AND "+w.row.by+
						" = $2 AND deleted_at IS NULL", project, by).Scan(&row); err != nil {
						t.Fatalf("the row of %s %s in %s: %v", w.row.table, key, on, err)
					}
					if w.param() != "{project_id}" {
````

````old server/internal/bootstrap/project_write_locks_test.go
					if member != "" && heldBy(t, pool, memberRow, row) {
						t.Errorf("%s holds %s's membership of %s while it waits for its workspace", w.op, member, on)
````
````new server/internal/bootstrap/project_write_locks_test.go
					if key != "" && heldBy(t, pool, rowLock, row) {
						t.Errorf("%s holds the row of %s %s in %s while it waits for its workspace", w.op, w.row.table, key, on)
````

````old server/internal/bootstrap/project_write_locks_test.go
					if member != "" && heldBy(t, pool, memberRow, row) {
						t.Errorf("%s holds %s's membership of %s before its project", w.op, member, on)
````
````new server/internal/bootstrap/project_write_locks_test.go
					if key != "" && heldBy(t, pool, rowLock, row) {
						t.Errorf("%s holds the row of %s %s in %s before its project", w.op, w.row.table, key, on)
````

`server/internal/bootstrap/project_writes_test.go`（修改，17 处）：

````old server/internal/bootstrap/project_writes_test.go
// an admin of Web, removes carol, creates a state, and leaves Web. Each row
// the write writes again is first made bob's, as last written by him, and
// checked so: a write that kept its row's writer would pass for alice's
// otherwise, she having made it.
````
````new server/internal/bootstrap/project_writes_test.go
// an admin of Web, removes carol, creates a state and renames it, and
// leaves Web. Each row the write writes again is first made bob's, as last
// written by him, and checked so: a write that kept its row's writer would
// pass for alice's otherwise, she having made it.
````

````old server/internal/bootstrap/project_writes_test.go
		return "UPDATE " + table + " SET updated_by_id = $3 WHERE " + where + " AND updated_by_id = $2"
	}
````
````new server/internal/bootstrap/project_writes_test.go
		return "UPDATE " + table + " SET updated_by_id = $3 WHERE " + where + " AND updated_by_id = $2"
	}
	// membership is the id of user's membership of Web; state, of Web's
	// state name.
	membership := func(user uuid.UUID) func() uuid.UUID {
		return func() uuid.UUID { return projectMemberships(t, pool, user, web)[0] }
	}
	state := func(name string) func() uuid.UUID { return func() uuid.UUID { return stateID(t, pool, web, name) } }
````

````old server/internal/bootstrap/project_writes_test.go
		of                       uuid.UUID // the account whose membership of Web the path names by its id, as %s
````
````new server/internal/bootstrap/project_writes_test.go
		named                    func() uuid.UUID // the row under Web the path names by its id, as %s; nil when it names Web
````

````old server/internal/bootstrap/project_writes_test.go
		{"updateProject", http.MethodPatch, "/api/v0/projects/" + web.String(), `{"name":"Site"}`, uuid.UUID{}, http.StatusOK,
````
````new server/internal/bootstrap/project_writes_test.go
		{"updateProject", http.MethodPatch, "/api/v0/projects/" + web.String(), `{"name":"Site"}`, nil, http.StatusOK,
````

````old server/internal/bootstrap/project_writes_test.go
		{"archiveProject", http.MethodPost, "/api/v0/projects/" + web.String() + "/archive", "", uuid.UUID{}, http.StatusOK,
````
````new server/internal/bootstrap/project_writes_test.go
		{"archiveProject", http.MethodPost, "/api/v0/projects/" + web.String() + "/archive", "", nil, http.StatusOK,
````

````old server/internal/bootstrap/project_writes_test.go
		{"archiveProject again", http.MethodPost, "/api/v0/projects/" + web.String() + "/archive", "", uuid.UUID{}, http.StatusOK,
````
````new server/internal/bootstrap/project_writes_test.go
		{"archiveProject again", http.MethodPost, "/api/v0/projects/" + web.String() + "/archive", "", nil, http.StatusOK,
````

````old server/internal/bootstrap/project_writes_test.go
		{"unarchiveProject", http.MethodPost, "/api/v0/projects/" + web.String() + "/unarchive", "", uuid.UUID{}, http.StatusOK,
````
````new server/internal/bootstrap/project_writes_test.go
		{"unarchiveProject", http.MethodPost, "/api/v0/projects/" + web.String() + "/unarchive", "", nil, http.StatusOK,
````

````old server/internal/bootstrap/project_writes_test.go
		{"updateProjectPreferences", http.MethodPatch, "/api/v0/me/projects/" + web.String() + "/preferences", `{"sort_order":5}`, uuid.UUID{}, http.StatusOK,
````
````new server/internal/bootstrap/project_writes_test.go
		{"updateProjectPreferences", http.MethodPatch, "/api/v0/me/projects/" + web.String() + "/preferences", `{"sort_order":5}`, nil, http.StatusOK,
````

````old server/internal/bootstrap/project_writes_test.go
			`{"members":[{"member_id":"` + bobID.String() + `","role":15}]}`, uuid.UUID{}, http.StatusCreated, "",
````
````new server/internal/bootstrap/project_writes_test.go
			`{"members":[{"member_id":"` + bobID.String() + `","role":15}]}`, nil, http.StatusCreated, "",
````

````old server/internal/bootstrap/project_writes_test.go
			`{"members":[{"member_id":"` + carol + `","role":15}]}`, uuid.UUID{}, http.StatusCreated,
````
````new server/internal/bootstrap/project_writes_test.go
			`{"members":[{"member_id":"` + carol + `","role":15}]}`, nil, http.StatusCreated,
````

````old server/internal/bootstrap/project_writes_test.go
		{"updateProjectMember", http.MethodPatch, "/api/v0/project-members/%s", `{"role":20}`, bobID, http.StatusOK,
````
````new server/internal/bootstrap/project_writes_test.go
		{"updateProjectMember", http.MethodPatch, "/api/v0/project-members/%s", `{"role":20}`, membership(bobID), http.StatusOK,
````

````old server/internal/bootstrap/project_writes_test.go
		{"removeProjectMember", http.MethodDelete, "/api/v0/project-members/%s", "", carolID, http.StatusNoContent,
````
````new server/internal/bootstrap/project_writes_test.go
		{"removeProjectMember", http.MethodDelete, "/api/v0/project-members/%s", "", membership(carolID), http.StatusNoContent,
````

````old server/internal/bootstrap/project_writes_test.go
			uuid.UUID{}, http.StatusCreated, "",
````
````new server/internal/bootstrap/project_writes_test.go
			nil, http.StatusCreated, "",
````

````old server/internal/bootstrap/project_writes_test.go
			0, 1},
````
````new server/internal/bootstrap/project_writes_test.go
			0, 1},
		// QA, renamed Checked.
		{"updateState", http.MethodPatch, "/api/v0/states/%s", `{"name":"Checked"}`, state("QA"), http.StatusOK,
			bobs("states", "project_id = $1 AND name = 'QA'"),
			"SELECT updated_at, updated_by_id = $2 AND name = 'Checked' FROM states WHERE project_id = $1 AND name IN ('QA', 'Checked')", 1, 1},
````

````old server/internal/bootstrap/project_writes_test.go
		{"leaveProject", http.MethodPost, "/api/v0/projects/" + web.String() + "/leave", "", uuid.UUID{}, http.StatusNoContent,
````
````new server/internal/bootstrap/project_writes_test.go
		{"leaveProject", http.MethodPost, "/api/v0/projects/" + web.String() + "/leave", "", nil, http.StatusNoContent,
````

````old server/internal/bootstrap/project_writes_test.go
		if w.of != (uuid.UUID{}) {
			path = fmt.Sprintf(path, projectMemberships(t, pool, w.of, web)[0])
````
````new server/internal/bootstrap/project_writes_test.go
		if w.named != nil {
			path = fmt.Sprintf(path, w.named())
````

````old server/internal/bootstrap/project_writes_test.go
	return p.ID
````
````new server/internal/bootstrap/project_writes_test.go
	return p.ID
}

// stateID is the id of project's undeleted state name.
func stateID(t *testing.T, pool *pgxpool.Pool, project uuid.UUID, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT id FROM states WHERE project_id = $1 AND name = $2 AND deleted_at IS NULL", project,
		name).Scan(&id); err != nil {
		t.Fatalf("the state %s of %s: %v", name, project, err)
	}
	return id
````

`server/internal/bootstrap/project_connection_test.go`（修改，2 处）：

````old server/internal/bootstrap/project_connection_test.go
// admin of Web and removes bob, and creates a state in it; carol, an
// admin, leaves it; alice deletes both projects.
````
````new server/internal/bootstrap/project_connection_test.go
// admin of Web and removes bob, and creates a state in it and renames it;
// carol, an admin, leaves it; alice deletes both projects.
````

````old server/internal/bootstrap/project_connection_test.go
	send(http.MethodPost, web+"/states", r.alice, `{"name":"QA","color":"#0EA5E9","group":"completed"}`, http.StatusCreated)
````
````new server/internal/bootstrap/project_connection_test.go
	var qa struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, send(http.MethodPost, web+"/states", r.alice, `{"name":"QA","color":"#0EA5E9","group":"completed"}`, http.StatusCreated), &qa)
	send(http.MethodPatch, "/api/v0/states/"+qa.ID.String(), r.alice, `{"name":"Checked"}`, http.StatusOK)
````

- [ ] **Step 4: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/project/...`
Expected: 全部 `ok`（`apitest.Main` 两个方向：`updateState` 的五个码）。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix$|TestMatrixViolationsCatchesEachColumnGap$|TestEachWriteOnAProjectSharesItsWorkspaceFirst$|TestTheWritesOnAProjectStampTheirRequest$|TestTheWritesOnAProjectRunOnTheirTransactionsConnection$|TestEachLockOfAWriteOnAProjectMembershipIsItsStrength$' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过（关键词守卫 5 个命中都有例外）。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

- [ ] **Step 5: 提交**

```bash
git add api/modules/project.yaml api/openapi.yaml server/internal/bootstrap/permission_matrix_columns_test.go server/internal/bootstrap/permission_matrix_seeded_test.go server/internal/bootstrap/permission_matrix_states_test.go server/internal/bootstrap/permission_matrix_targets_test.go server/internal/bootstrap/project_connection_test.go server/internal/bootstrap/project_write_locks_test.go server/internal/bootstrap/project_writes_test.go server/internal/modules/project/adapter/http/handler.go server/internal/modules/project/adapter/http/handler_test.go server/internal/modules/project/adapter/http/state_writes_test.go server/internal/modules/project/adapter/http/states.go server/internal/modules/project/module.go web/apps/web/helpers/authentication.helper.ts web/packages/i18n/src/locales/en/auth.json web/packages/i18n/src/locales/zh-CN/auth.json api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/bodyshape.gen.go server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P7a): PATCH /api/v0/states/{state_id}, in the matrix and on the lock ladder

updateState serves the fields a body gives and declares its five
codes, project.state_not_found and project.state_last_in_group with
their messages. The matrix's state rows aim at the state of the
column's project, the triage state 404 to every column, the archived
project's changed as any other's. The first-lock test probes the row a
write names under its project, a membership or a state, and the stamps
and the connection test take the change.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p7a.py` 的编号；"层"是它被发现的每一层，端到端是单独运行的故事）：

| 变异 | 改坏 | 必须失败的测试 | 层 |
|---|---|---|---|
| `s4-update-frozen-clock` | `UpdateState` 接上停在 2001 年的时钟 | `TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`（Task 9 起）、`TestTheWritesOnAProjectStampTheirRequest`（Task 7 起） | 组合 |
| `s4-update-drops-group` | `PATCH /states/{id}` 丢掉组 | `TestUpdateState`、`TestPermissionMatrix`、P6（Task 10 起） | 单元；组合；端到端 |

---

### Task 7: `deleteState`、`markDefaultState`：守卫的删除、两条语句的设为默认；`project.state_default`

**Files:**
- Create: `server/internal/modules/project/app/delete_state.go`、`server/internal/modules/project/app/delete_state_test.go`、`server/internal/modules/project/app/mark_default_state.go`、`server/internal/modules/project/app/mark_default_state_test.go`
- Modify: `api/modules/project.yaml`、`api/openapi.yaml`、`server/internal/bootstrap/permission_matrix_states_test.go`、`server/internal/bootstrap/project_connection_test.go`、`server/internal/bootstrap/project_write_locks_test.go`、`server/internal/bootstrap/project_writes_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/project/adapter/http/handler.go`、`server/internal/modules/project/adapter/http/handler_test.go`、`server/internal/modules/project/adapter/http/state_writes_test.go`、`server/internal/modules/project/adapter/http/states.go`、`server/internal/modules/project/app/clock_test.go`、`server/internal/modules/project/app/fakes_state_test.go`、`server/internal/modules/project/app/fakes_write_test.go`、`server/internal/modules/project/app/state_ports.go`、`server/internal/modules/project/domain/actions.go`、`server/internal/modules/project/module.go`、`web/apps/web/helpers/authentication.helper.ts`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.8；M3 设计 3.6、3.17、3.19、9.3 交错 10）：
  - 契约：`DELETE /api/v0/states/{state_id}`（`deleteState`，204；`[project.state_not_found, forbidden, project.state_default, project.state_last_in_group]`）、`POST /api/v0/states/{state_id}/mark-default`（`markDefaultState`，204；`[project.state_not_found, forbidden]`）。设为默认的描述不说工作项（M4 的事）。
  - `access`：`state.delete`、`state.mark_default`（同 `state.create`）；`ActionStateDelete`、`ActionStateMarkDefault`。
  - 端口：`StateDeleter`（`StateFinder`、`GroupCounter`、`DeleteState`）、`DefaultMarker`（`StateFinder`、`MarkDefaultState`）；锁经用例的 `Locks`。
  - `DeleteState`（`delete_state.go`）：`lockRowAndDecide` → 时钟 → `DeleteState`（守卫的写，不先查默认）；删了 0 行时 `deletedNothing(ctx, rw)`：经 `rw.read` 重读这一行（还在锁下），默认的答 `ErrStateDefault`，没有了的答这一行的 404，既在又不是默认的是错误；删了之后 `CountGroupStates` 数它的组，`CheckGroupKept(left)`：拒绝时事务回滚删除。既是默认、又是组里唯一的状态答 `project.state_default`：守卫的写在数组之前（spec 第 3 节第 13 条，设计 3.6 的锁表照此改）；契约写明。
  - `MarkDefaultState`（`mark_default_state.go`）：`lockRowAndDecide` → 时钟 → `MarkDefaultState`；第二条语句写 0 行时 `ErrStateNotFound`，事务回滚第一条：项目不会没有默认状态。哪一种交错到得了哪一个守卫，见 spec 第 3 节第 4 条（O4）：`DeleteState` 的守卫在组合一层由"先设为默认、再删除"到达；`SetDefaultState` 写 0 行在组合一层到不了（先删除时重读在锁下先答 404），是纵深防御，钉在单元和存储两层。
  - HTTP、接线；前端文案：`project.state_default`。
  - 矩阵：`deleteState` 五行（删除 Review、默认的 Backlog 409、组里唯一的 Done 409、分诊状态每一列 404、已归档项目的小表）、`markDefaultState` 三行（Todo、分诊状态、已归档项目的小表）；最先锁工作区的测试、盖戳、连接各加两个写；`bobs` 的说明。

**Tests:**
- `TestDeleteState`（两行：Review，started 组还有 In Progress；归档的 ops 的 Icebox，backlog 组还有 Backlog）、`TestDeleteStateRefuses`（12 行：在判定之前的同 Task 5；判定之后：默认的 Backlog，删除跳过它，重读答 409；跳过之后重读时没有了，404；Todo 组里唯一的，删除之后 409、事务回滚；跳过之后重读时既在又不是默认，写自己的错误；判定之前的拒绝每一行不变）、`TestDeleteStateReturnsEachFailure`（八个调用）、`TestDeleteStateChecksTheReadAfterIt`（`afterDeletion`：重读失败原样返回；重读回答另一个状态，写自己的错误：`rowWrite.read` 拒绝它，不是 `project.state_default`）、`TestMarkDefaultState`（三行：Todo 成为默认、Backlog 不再是；Backlog 是默认时再设一次；归档的 ops）、`TestMarkDefaultStateRefuses`（9 行：同上，加第二条语句跳过的状态，404，事务回滚第一条）、`TestMarkDefaultStateReturnsEachFailure`（七个调用）；`TestEachWriteReadsTheClockUnderItsLock` 加两行（`deleteState` 在删除之后数组，时钟在删除之前读）。
- HTTP：`TestDeleteState`（`state_writes_test.go`：204、没有正文；没有这一行、项目成员、默认、组里唯一、失败）、`TestMarkDefaultState`（204；没有这一行、项目成员、失败）。
- 组合：`TestPermissionMatrix` 的八行；`TestEachWriteOnAProjectSharesItsWorkspaceFirst` 两行；`TestTheWritesOnAProjectStampTheirRequest`（删除的行由 alice、`deleted_at` 是请求的时刻；设为默认写 Done 和原来的默认 Backlog，两行都先盖上 bob 的戳）；`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（建两个状态，改名、删除第一个，第二个设为默认）。

- [ ] **Step 1: 契约**

`api/modules/project.yaml`（修改，1 处）：

````old api/modules/project.yaml
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/State'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/me/projects/{project_id}/preferences:
````
````new api/modules/project.yaml
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/State'
        default:
          $ref: '#/components/responses/Problem'
    delete:
      operationId: deleteState
      tags: [project]
      summary: Delete a state
      description: >-
        For the project's admins, and its members who are the workspace's
        admins; an archived project's states are deleted as any other's. A
        state that does not exist or is deleted, the intake's triage state,
        and a state whose project the caller does not see answer
        project.state_not_found; a caller who sees the project but may not
        change its states, forbidden, whether or not the state is the
        default or the last of its group. The project's default state is not
        deleted (project.state_default): make another state the default
        first. Nor is the only state of its group
        (project.state_last_in_group): every group keeps a state. A state
        that is both is project.state_default. The state is deleted at the
        moment of the request, by the caller, and its name is free again in
        the project. The role is decided after the workspace and project
        rows are locked.
      security: [{bearer: []}]
      x-problem-codes: [project.state_not_found, forbidden, project.state_default, project.state_last_in_group]
      responses:
        '204':
          description: The state is deleted.
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/states/{state_id}/mark-default:
    parameters:
      - $ref: '#/components/parameters/StateID'
    post:
      operationId: markDefaultState
      tags: [project]
      summary: Make a state its project's default
      description: >-
        For the project's admins, and its members who are the workspace's
        admins; an archived project's default changes as any other's. The
        state becomes the project's default, and the state that was the
        default is no longer: a project has exactly one. Making the default
        state the default again leaves it so. A state that does not exist
        or is deleted, the intake's triage state, and a state whose project
        the caller does not see answer project.state_not_found; a caller
        who sees the project but may not change its states, forbidden,
        whether or not the state is the default or the last of its group.
        The role is decided after the workspace and project rows are locked.
      security: [{bearer: []}]
      x-problem-codes: [project.state_not_found, forbidden]
      responses:
        '204':
          description: The state is the project's default.
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/me/projects/{project_id}/preferences:
````

`api/openapi.yaml`（修改，1 处）：

````old api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1states~1{state_id}'
````
````new api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1states~1{state_id}'
  /api/v0/states/{state_id}/mark-default:
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1states~1{state_id}~1mark-default'
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `b316b55bd2e170ae16702ba430aa13058d4d586d6a5376ee0ec9504f974bfcc3` | 2700 | `api/dist/openapi.yaml` |
| `36630ec12ac2d76df68a3094412e854654e9ea8cca39c5f509f982486b020fc7` | 2801 | `server/internal/modules/project/adapter/http/gen/server.gen.go` |
| `c215f77fc52eff227f86dbe5eb81a5f1155c4cd3873f4a2599c555e44f451589` | 2978 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml web/packages/api-client/src/schema.gen.ts server/internal/modules/project/adapter/http/gen/server.gen.go`
Expected: 与上表相同。

- [ ] **Step 2: 规则、操作名、端口和用例**

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"state.update": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
````
````new server/internal/modules/access/domain/rules.go
	"state.update":       {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	"state.delete":       {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	"state.mark_default": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"state.update": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
````
````new server/internal/modules/access/domain/rules_test.go
	"state.update": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
	"state.delete": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
	"state.mark_default": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible,
		invisible, forbidden, invisible, invisible, invisible, forbidden, forbidden},
````

`server/internal/modules/project/domain/actions.go`（修改，2 处）：

````old server/internal/modules/project/domain/actions.go
	ActionStateUpdate shared.Action = "state.update"
````
````new server/internal/modules/project/domain/actions.go
	ActionStateUpdate shared.Action = "state.update"
	// ActionStateDelete is deleting a state: deleteState.
	ActionStateDelete shared.Action = "state.delete"
	// ActionStateMarkDefault is making a state its project's default:
	// markDefaultState.
	ActionStateMarkDefault shared.Action = "state.mark_default"
````

````old server/internal/modules/project/domain/actions.go
		ActionStateCreate, ActionStateUpdate}
````
````new server/internal/modules/project/domain/actions.go
		ActionStateCreate, ActionStateUpdate, ActionStateDelete,
		ActionStateMarkDefault}
````

`server/internal/modules/project/app/state_ports.go`（修改，1 处）：

````old server/internal/modules/project/app/state_ports.go
	UpdateState(ctx context.Context, id uuid.UUID, p domain.StatePatch, by uuid.UUID, now time.Time) (domain.State, error)
}

````
````new server/internal/modules/project/app/state_ports.go
	UpdateState(ctx context.Context, id uuid.UUID, p domain.StatePatch, by uuid.UUID, now time.Time) (domain.State, error)
}

// StateDeleter is deleteState's repository. Its writes run in the
// transaction ctx carries, under the project's FOR NO KEY UPDATE.
type StateDeleter interface {
	StateFinder
	GroupCounter
	// DeleteState deletes the undeleted state id, unless it is its project's
	// default or its triage state, by the account by at now: a guarded
	// write (M3 design 3.17). deleted is false when it did not.
	DeleteState(ctx context.Context, id, by uuid.UUID, now time.Time) (deleted bool, err error)
}

// DefaultMarker is markDefaultState's repository. It runs in the
// transaction ctx carries, under the project's FOR NO KEY UPDATE.
type DefaultMarker interface {
	StateFinder
	// MarkDefaultState makes the undeleted state id of projectID its
	// default, by the account by at now, in two statements: the project's
	// default state the default no longer, then this one the default (M3
	// design 3.17). marked is false when the second wrote no row, and the
	// caller rolls the first back.
	MarkDefaultState(ctx context.Context, projectID, id, by uuid.UUID, now time.Time) (marked bool, err error)
}

````

`server/internal/modules/project/app/delete_state.go`（新文件，75 行）：

````file server/internal/modules/project/app/delete_state.go
package app

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DeleteState deletes a state: DELETE /api/v0/states/{state_id} (M3
// design 3.17).
type DeleteState struct {
	locks  Locks
	states StateDeleter
	tx     shared.TxManager
	clock  Clock
}

// NewDeleteState returns the use case.
func NewDeleteState(locks Locks, states StateDeleter, tx shared.TxManager, clock Clock) *DeleteState {
	return &DeleteState{locks: locks, states: states, tx: tx, clock: clock}
}

// Execute, in one transaction and in the order of M3 design 3.6: the
// state's locks (lockRowAndDecide: the state read for its project and
// workspace, the workspace FOR SHARE, the project FOR NO KEY UPDATE, the
// state read again) and the decision on state.delete; an archived
// project's states are deleted as any other's (3.19). Then, at the time the
// clock gives under the locks, the guarded deletion (3.17), which passes
// over the project's default state: the state read again answers why
// (deletedNothing). Then the states its group keeps without it
// (domain.CheckGroupKept): a refusal rolls the deletion back.
func (u *DeleteState) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	return u.tx.WithinTx(ctx, func(ctx context.Context) error {
		rw := rowWrite[domain.State]{id: id, action: domain.ActionStateDelete, find: u.states.StateByID, notFound: domain.ErrStateNotFound}
		_, s, err := lockRowAndDecide(ctx, u.locks, actor, rw)
		if err != nil {
			return err
		}
		deleted, err := u.states.DeleteState(ctx, s.ID, actor.UserID, u.clock.Now())
		switch {
		case err != nil:
			return err
		case !deleted:
			return deletedNothing(ctx, rw)
		}
		left, err := u.states.CountGroupStates(ctx, s.ProjectID, s.Group)
		if err != nil {
			return err
		}
		return domain.CheckGroupKept(left)
	})
}

// deletedNothing answers a guarded deletion of the state rw names that
// deleted nothing, from the state read again as the lock path reads it
// (rowWrite.read): project.state_default for the default,
// project.state_not_found for a state no longer there. A state there that
// is not the default is an error: the guard had no reason to pass it over.
func deletedNothing(ctx context.Context, rw rowWrite[domain.State]) error {
	s, err := rw.read(ctx)
	switch {
	case err != nil:
		return err
	case s.Default:
		return domain.ErrStateDefault
	}
	return fmt.Errorf("state %s neither deleted nor the default", s.ID)
}
````

`server/internal/modules/project/app/mark_default_state.go`（新文件，53 行）：

````file server/internal/modules/project/app/mark_default_state.go
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// MarkDefaultState makes a state its project's default: POST
// /api/v0/states/{state_id}/mark-default (M3 design 3.17).
type MarkDefaultState struct {
	locks  Locks
	states DefaultMarker
	tx     shared.TxManager
	clock  Clock
}

// NewMarkDefaultState returns the use case.
func NewMarkDefaultState(locks Locks, states DefaultMarker, tx shared.TxManager, clock Clock) *MarkDefaultState {
	return &MarkDefaultState{locks: locks, states: states, tx: tx, clock: clock}
}

// Execute, in one transaction and in the order of M3 design 3.6: the
// state's locks (lockRowAndDecide) and the decision on state.mark_default;
// an archived project's default changes as any other's (3.19). Then, at
// the time the clock gives under the locks, the two statements of 3.17:
// the project's default the default no longer, then the state the default.
// When the second writes no row the state is project.state_not_found, and
// the transaction rolls the first back: no project is left without a
// default.
func (u *MarkDefaultState) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	return u.tx.WithinTx(ctx, func(ctx context.Context) error {
		_, s, err := lockRowAndDecide(ctx, u.locks, actor, rowWrite[domain.State]{id: id, action: domain.ActionStateMarkDefault,
			find: u.states.StateByID, notFound: domain.ErrStateNotFound})
		if err != nil {
			return err
		}
		marked, err := u.states.MarkDefaultState(ctx, s.ProjectID, s.ID, actor.UserID, u.clock.Now())
		switch {
		case err != nil:
			return err
		case !marked:
			return domain.ErrStateNotFound
		}
		return nil
	})
}
````

- [ ] **Step 3: 用例的测试**

`server/internal/modules/project/app/fakes_state_test.go`（修改，4 处）：

````old server/internal/modules/project/app/fakes_state_test.go
// none of them, as the store reads none.
````
````new server/internal/modules/project/app/fakes_state_test.go
// none of them, as the store reads none. passOver makes the guarded
// writes, DeleteState and MarkDefaultState's second statement, write no
// row: "kept" leaves the state as it is, "gone" takes it away, as a
// deletion meanwhile would.
````

````old server/internal/modules/project/app/fakes_state_test.go
	states map[uuid.UUID]domain.State
````
````new server/internal/modules/project/app/fakes_state_test.go
	states   map[uuid.UUID]domain.State
	passOver string
````

````old server/internal/modules/project/app/fakes_state_test.go
		s.ProjectID = f.reread.project
	}
````
````new server/internal/modules/project/app/fakes_state_test.go
		s.ProjectID = f.reread.project
	}
	if f.answersAs != (uuid.UUID{}) {
		s.ID = f.answersAs
	}
````

````old server/internal/modules/project/app/fakes_state_test.go
}

// statePatch is p as UpdateState logs it: each field it gives.
````
````new server/internal/modules/project/app/fakes_state_test.go
}

// DeleteState deletes the state id, unless it is its project's default or
// f.passOver passes over it.
func (f *fakeStates) DeleteState(ctx context.Context, id, by uuid.UUID, now time.Time) (bool, error) {
	f.log.add(ctx, "DeleteState %s by %s at %s", id, by, now.Format(timeFormat))
	if err := f.fail("DeleteState"); err != nil {
		return false, err
	}
	s, ok := f.states[id]
	if !ok || s.Default || f.passedOver(id) {
		return false, nil
	}
	delete(f.states, id)
	return true, nil
}

// MarkDefaultState makes the state id of projectID its default, after the
// project's default the default no longer, unless f.passOver passes over
// the second statement: the first stays written, as the store leaves it for
// the caller to roll back.
func (f *fakeStates) MarkDefaultState(ctx context.Context, projectID, id, by uuid.UUID, now time.Time) (bool, error) {
	f.log.add(ctx, "MarkDefaultState %s %s by %s at %s", projectID, id, by, now.Format(timeFormat))
	if err := f.fail("MarkDefaultState"); err != nil {
		return false, err
	}
	at := now.Truncate(time.Microsecond)
	for _, s := range f.of(projectID) {
		if s.Default {
			s.Default, s.UpdatedAt = false, at
			f.states[s.ID] = s
		}
	}
	s, ok := f.states[id]
	if !ok || s.ProjectID != projectID || f.passedOver(id) {
		return false, nil
	}
	s.Default, s.UpdatedAt = true, at
	f.states[id] = s
	return true, nil
}

// passedOver reports whether f.passOver passes over the state id, taking it
// away for "gone".
func (f *fakeStates) passedOver(id uuid.UUID) bool {
	if f.passOver == "gone" {
		delete(f.states, id)
	}
	return f.passOver != ""
}

// statePatch is p as UpdateState logs it: each field it gives.
````

`server/internal/modules/project/app/fakes_write_test.go`（修改，1 处）：

````old server/internal/modules/project/app/fakes_write_test.go
	// ran, and how the second one answers; answersAs, when set, is the id
	// each read of a membership answers for the one asked. changedAs, when
	// set, is the id the write of the row answers for the one it wrote
	// (UpdateMemberRole, CreateState, UpdateState).
````
````new server/internal/modules/project/app/fakes_write_test.go
	// ran, how the second one answers, and answersAs, when set, the id each
	// answers for the one asked; changedAs, when set, is the id the write of
	// the row answers for the one it wrote (UpdateMemberRole, CreateState,
	// UpdateState).
````

`server/internal/modules/project/app/delete_state_test.go`（新文件，187 行）：

````file server/internal/modules/project/app/delete_state_test.go
package app_test

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newDeleteState is DeleteState over newStates' fakes, its clock logged.
func newDeleteState() (*app.DeleteState, *writeFixture, *fakeStates) {
	f, s := newStates()
	return app.NewDeleteState(f.locks(), s, f.tx, clockAt{clockNow, f.log}), f, s
}

// stateDeleted are the calls of user's deletion of the state id of project
// up to the deletion: its locks and decision, the clock, the deletion by
// user at that time.
func stateDeleted(user, id, project uuid.UUID) []string {
	return append(stateLocked(id, project, user, domain.ActionStateDelete), "Now",
		fmt.Sprintf("DeleteState %s by %s at %s", id, user, clockNow.Format(timeFormat)))
}

// DeleteState, in one transaction and in the order of M3 design 3.6 and
// 3.17, locks the state's project, decides, reads the clock, deletes the
// state by the caller at that time, then counts the states its group keeps:
// Review leaves In Progress in the started group. ops, archived, has its
// states deleted as any other's (3.19): Icebox leaves Backlog in the
// backlog group.
func TestDeleteState(t *testing.T) {
	icebox := uuid.NewV7()
	for _, tt := range []struct {
		name    string
		id      uuid.UUID
		project uuid.UUID
		group   domain.StateGroup
	}{
		{"Review", webReview, webID, domain.GroupStarted},
		{"archived ops's Icebox", icebox, opsID, domain.GroupBacklog},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, s := newDeleteState()
			s.states[icebox] = domain.State{ID: icebox, WorkspaceID: acme.ID, ProjectID: opsID, Name: "Icebox", Group: domain.GroupBacklog,
				Sequence: 20000}
			want := maps.Clone(s.states)
			delete(want, tt.id)
			err := uc.Execute(as(bob), tt.id)
			calls := append(stateDeleted(bob, tt.id, tt.project), fmt.Sprintf("CountGroupStates %s %s", tt.project, tt.group))
			if err != nil || !maps.Equal(s.states, want) {
				t.Errorf("Execute() = %v, the states after it %v; want nil, %v", err, s.states, want)
			}
			if !slices.Equal(f.log.calls, calls) {
				t.Errorf("calls\n%q\nwant\n%q", f.log.calls, calls)
			}
		})
	}
}

// Refusals, each in its place: no caller, before the transaction; a state
// that is not there, a workspace or project deleted while its lock waited,
// a state deleted or moved to ops meanwhile, and a caller who does not see
// web, each project.state_not_found; a member, the Authorizer's 403. Then,
// past the decision: Backlog, the default, which the deletion passes over
// and the state read again answers project.state_default; a state the
// deletion passed over that is gone when read again,
// project.state_not_found; Todo, the only state of its group, deleted and
// project.state_last_in_group, which the transaction rolls back. A
// deletion that passed over a state neither gone nor the default is the
// write's own error. A refusal before the deletion leaves the states as
// they were.
func TestDeleteStateRefuses(t *testing.T) {
	upTo := func(n int) []string { return stateLocked(webReview, webID, bob, domain.ActionStateDelete)[:n] }
	readAgain := func(id uuid.UUID) []string { return append(stateDeleted(bob, id, webID), "StateByID "+id.String()) }
	for _, tt := range []struct {
		name  string
		ctx   context.Context
		id    uuid.UUID
		set   func(f *writeFixture, s *fakeStates)
		want  error
		calls []string
	}{
		{"no caller", context.Background(), webReview, nil, shared.Unauthenticated(), nil},
		{"no state", as(bob), uuid.Nil(), nil, domain.ErrStateNotFound, []string{"Begin", "StateByID " + uuid.Nil().String()}},
		{"acme deleted while its lock waited", as(bob), webReview, func(f *writeFixture, _ *fakeStates) { f.workspaces.gone = true },
			domain.ErrStateNotFound, upTo(3)},
		{"web deleted while its lock waited", as(bob), webReview, func(f *writeFixture, _ *fakeStates) { f.store.deleted = true },
			domain.ErrStateNotFound, upTo(4)},
		{"the state deleted while the locks waited", as(bob), webReview, func(f *writeFixture, _ *fakeStates) { f.store.reread.gone = true },
			domain.ErrStateNotFound, upTo(5)},
		{"the state moved to ops", as(bob), webReview, func(f *writeFixture, _ *fakeStates) { f.store.reread.project = opsID },
			domain.ErrStateNotFound, upTo(5)},
		{"a caller who does not see web", as(erin), webReview, nil, domain.ErrStateNotFound,
			stateLocked(webReview, webID, erin, domain.ActionStateDelete)},
		{"a member", as(alice), webReview, nil, shared.Forbidden(), stateLocked(webReview, webID, alice, domain.ActionStateDelete)},
		{"Backlog, the default", as(bob), webBacklog, nil, domain.ErrStateDefault, readAgain(webBacklog)},
		{"a state passed over and gone", as(bob), webReview, func(_ *writeFixture, s *fakeStates) { s.passOver = "gone" },
			domain.ErrStateNotFound, readAgain(webReview)},
		{"Todo, its group's only state", as(bob), webTodo, nil, domain.ErrStateLastInGroup,
			append(stateDeleted(bob, webTodo, webID), fmt.Sprintf("CountGroupStates %s %s", webID, domain.GroupUnstarted))},
		{"a state passed over and kept", as(bob), webReview, func(_ *writeFixture, s *fakeStates) { s.passOver = "kept" }, nil,
			readAgain(webReview)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, s := newDeleteState()
			if tt.set != nil {
				tt.set(f, s)
			}
			before := maps.Clone(s.states)
			err := uc.Execute(tt.ctx, tt.id)
			outcome{tt.name, tt.want, tt.calls}.check(t, err, f)
			if beforeTheDeletion := len(tt.calls) <= 6; beforeTheDeletion && !maps.Equal(s.states, before) {
				t.Errorf("the states after the refusal: %v; want them as they were, %v", s.states, before)
			}
		})
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after: Review deleted, so that the count runs.
func TestDeleteStateReturnsEachFailure(t *testing.T) {
	all := append(stateDeleted(bob, webReview, webID), fmt.Sprintf("CountGroupStates %s %s", webID, domain.GroupStarted))
	fail := func(method string) func(f *writeFixture) {
		return func(f *writeFixture) { f.store.errs = map[string]error{method: errDisk} }
	}
	for _, tt := range []struct {
		name  string
		fail  func(f *writeFixture)
		calls int // how many of the deletion's calls ran
	}{
		{"the state's read", fail("StateByID"), 2},
		{"the workspace's lock", func(f *writeFixture) { f.workspaces.err = errDisk }, 3},
		{"the project's lock", fail("LockProject"), 4},
		{"the read under the locks", func(f *writeFixture) { f.store.reread.err = errDisk }, 5},
		{"the decision", func(f *writeFixture) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, 6},
		{"the deletion", fail("DeleteState"), 8},
		{"the count of the group", fail("CountGroupStates"), 9},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 9},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, _ := newDeleteState()
			tt.fail(f)
			err := uc.Execute(as(bob), webReview)
			outcome{tt.name, errDisk, all[:tt.calls]}.check(t, err, f)
		})
	}
}

// afterDeletion is s, which set changes once a deletion ran: how the read
// after the deletion answers.
type afterDeletion struct {
	*fakeStates
	set func(s *fakeStates)
}

func (s afterDeletion) DeleteState(ctx context.Context, id, by uuid.UUID, now time.Time) (bool, error) {
	deleted, err := s.fakeStates.DeleteState(ctx, id, by, now)
	s.set(s.fakeStates)
	return deleted, err
}

// The read that answers why the deletion passed over Backlog, the last
// call: its failure comes back as itself, and Backlog answered for another
// id is the write's own error, never project.state_default, as the lock
// path's read refuses it (rowWrite.read).
func TestDeleteStateChecksTheReadAfterIt(t *testing.T) {
	for _, tt := range []struct {
		name string
		set  func(s *fakeStates)
		want error
	}{
		{"the read failing", func(s *fakeStates) { s.errs = map[string]error{"StateByID": errDisk} }, errDisk},
		{"the read answering another state", func(s *fakeStates) { s.answersAs = uuid.NewV7() }, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f, s := newStates()
			err := app.NewDeleteState(f.locks(), afterDeletion{s, tt.set}, f.tx, clockAt{clockNow, f.log}).Execute(as(bob), webBacklog)
			outcome{tt.name, tt.want, append(stateDeleted(bob, webBacklog, webID), "StateByID "+webBacklog.String())}.check(t, err, f)
		})
	}
}
````

`server/internal/modules/project/app/mark_default_state_test.go`（新文件，142 行）：

````file server/internal/modules/project/app/mark_default_state_test.go
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

// newMarkDefaultState is MarkDefaultState over newStates' fakes, its clock
// logged.
func newMarkDefaultState() (*app.MarkDefaultState, *writeFixture, *fakeStates) {
	f, s := newStates()
	return app.NewMarkDefaultState(f.locks(), s, f.tx, clockAt{clockNow, f.log}), f, s
}

// defaultMarked are the calls of user's making the state id of project its
// default: its locks and decision, the clock, the two statements by user at
// that time.
func defaultMarked(user, id, project uuid.UUID) []string {
	return append(stateLocked(id, project, user, domain.ActionStateMarkDefault), "Now",
		fmt.Sprintf("MarkDefaultState %s %s by %s at %s", project, id, user, clockNow.Format(timeFormat)))
}

// MarkDefaultState, in one transaction and in the order of M3 design 3.6
// and 3.17, locks the state's project, decides, reads the clock, then makes
// the state the default by the caller at that time, the project's default
// before it the default no longer: Todo becomes web's, and Backlog is no
// longer. Backlog made web's default again stays it; ops, archived, has
// its default made as any other's (3.19).
func TestMarkDefaultState(t *testing.T) {
	for _, tt := range []struct {
		name    string
		id      uuid.UUID
		project uuid.UUID
		was     uuid.UUID // the project's default before
	}{
		{"Todo", webTodo, webID, webBacklog},
		{"Backlog again", webBacklog, webID, webBacklog},
		{"archived ops's Backlog", opsBacklog, opsID, opsBacklog},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, s := newMarkDefaultState()
			want := maps.Clone(s.states)
			was, is := want[tt.was], want[tt.id]
			was.Default, was.UpdatedAt = false, now
			want[tt.was] = was
			is.Default, is.UpdatedAt = true, now
			want[tt.id] = is
			err := uc.Execute(as(bob), tt.id)
			if err != nil || !maps.Equal(s.states, want) {
				t.Errorf("Execute() = %v, the states after it %v; want nil, %v", err, s.states, want)
			}
			if calls := defaultMarked(bob, tt.id, tt.project); !slices.Equal(f.log.calls, calls) {
				t.Errorf("calls\n%q\nwant\n%q", f.log.calls, calls)
			}
		})
	}
}

// Refusals, each in its place: no caller, before the transaction; a state
// that is not there, a workspace or project deleted while its lock waited,
// a state deleted or moved to ops meanwhile, and a caller who does not see
// web, each project.state_not_found; a member, the Authorizer's 403. Then a
// state the second statement passed over, project.state_not_found, which
// the transaction rolls the first back from. A refusal before the
// statements leaves the states as they were.
func TestMarkDefaultStateRefuses(t *testing.T) {
	upTo := func(n int) []string { return stateLocked(webTodo, webID, bob, domain.ActionStateMarkDefault)[:n] }
	for _, tt := range []struct {
		name  string
		ctx   context.Context
		id    uuid.UUID
		set   func(f *writeFixture, s *fakeStates)
		want  error
		calls []string
	}{
		{"no caller", context.Background(), webTodo, nil, shared.Unauthenticated(), nil},
		{"no state", as(bob), uuid.Nil(), nil, domain.ErrStateNotFound, []string{"Begin", "StateByID " + uuid.Nil().String()}},
		{"acme deleted while its lock waited", as(bob), webTodo, func(f *writeFixture, _ *fakeStates) { f.workspaces.gone = true },
			domain.ErrStateNotFound, upTo(3)},
		{"web deleted while its lock waited", as(bob), webTodo, func(f *writeFixture, _ *fakeStates) { f.store.deleted = true },
			domain.ErrStateNotFound, upTo(4)},
		{"the state deleted while the locks waited", as(bob), webTodo, func(f *writeFixture, _ *fakeStates) { f.store.reread.gone = true },
			domain.ErrStateNotFound, upTo(5)},
		{"the state moved to ops", as(bob), webTodo, func(f *writeFixture, _ *fakeStates) { f.store.reread.project = opsID },
			domain.ErrStateNotFound, upTo(5)},
		{"a caller who does not see web", as(erin), webTodo, nil, domain.ErrStateNotFound,
			stateLocked(webTodo, webID, erin, domain.ActionStateMarkDefault)},
		{"a member", as(alice), webTodo, nil, shared.Forbidden(), stateLocked(webTodo, webID, alice, domain.ActionStateMarkDefault)},
		{"a state the second statement passed over", as(bob), webTodo, func(_ *writeFixture, s *fakeStates) { s.passOver = "gone" },
			domain.ErrStateNotFound, defaultMarked(bob, webTodo, webID)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, s := newMarkDefaultState()
			if tt.set != nil {
				tt.set(f, s)
			}
			before := maps.Clone(s.states)
			err := uc.Execute(tt.ctx, tt.id)
			outcome{tt.name, tt.want, tt.calls}.check(t, err, f)
			if beforeTheStatements := len(tt.calls) <= 6; beforeTheStatements && !maps.Equal(s.states, before) {
				t.Errorf("the states after the refusal: %v; want them as they were, %v", s.states, before)
			}
		})
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after.
func TestMarkDefaultStateReturnsEachFailure(t *testing.T) {
	all := defaultMarked(bob, webTodo, webID)
	fail := func(method string) func(f *writeFixture) {
		return func(f *writeFixture) { f.store.errs = map[string]error{method: errDisk} }
	}
	for _, tt := range []struct {
		name  string
		fail  func(f *writeFixture)
		calls int // how many of the write's calls ran
	}{
		{"the state's read", fail("StateByID"), 2},
		{"the workspace's lock", func(f *writeFixture) { f.workspaces.err = errDisk }, 3},
		{"the project's lock", fail("LockProject"), 4},
		{"the read under the locks", func(f *writeFixture) { f.store.reread.err = errDisk }, 5},
		{"the decision", func(f *writeFixture) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, 6},
		{"the statements", fail("MarkDefaultState"), 8},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 8},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, _ := newMarkDefaultState()
			tt.fail(f)
			err := uc.Execute(as(bob), webTodo)
			outcome{tt.name, errDisk, all[:tt.calls]}.check(t, err, f)
		})
	}
}
````

`server/internal/modules/project/app/clock_test.go`（修改，3 处）：

````old server/internal/modules/project/app/clock_test.go
import (
````
````new server/internal/modules/project/app/clock_test.go
import (
	"fmt"
````

````old server/internal/modules/project/app/clock_test.go
// under its lock. The clock logs its read among the fakes' calls.
````
````new server/internal/modules/project/app/clock_test.go
// under its lock. deleteState counts the states its group keeps after it
// deletes the state, the clock read before. The clock logs its read among
// the fakes' calls.
````

````old server/internal/modules/project/app/clock_test.go
		}, stateUpdated(bob, webReview, domain.StatePatch{Group: ptr(domain.GroupBacklog)}, domain.GroupStarted)},
````
````new server/internal/modules/project/app/clock_test.go
		}, stateUpdated(bob, webReview, domain.StatePatch{Group: ptr(domain.GroupBacklog)}, domain.GroupStarted)},
		{"deleteState", func() ([]string, error) {
			uc, f, _ := newDeleteState()
			err := uc.Execute(as(bob), webReview)
			return f.log.calls, err
		}, append(stateDeleted(bob, webReview, webID), fmt.Sprintf("CountGroupStates %s %s", webID, domain.GroupStarted))},
		{"markDefaultState", func() ([]string, error) {
			uc, f, _ := newMarkDefaultState()
			err := uc.Execute(as(bob), webTodo)
			return f.log.calls, err
		}, defaultMarked(bob, webTodo, webID)},
````

- [ ] **Step 4: HTTP、接线、文案**

`server/internal/modules/project/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/project/adapter/http/handler.go
}

// UseCases are the use cases behind the module's operations.
````
````new server/internal/modules/project/adapter/http/handler.go
}

// DeleteStateUseCase is app.DeleteState.
type DeleteStateUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) error
}

// MarkDefaultStateUseCase is app.MarkDefaultState.
type MarkDefaultStateUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) error
}

// UseCases are the use cases behind the module's operations.
````

````old server/internal/modules/project/adapter/http/handler.go
	UpdateState       UpdateStateUseCase
````
````new server/internal/modules/project/adapter/http/handler.go
	UpdateState       UpdateStateUseCase
	DeleteState       DeleteStateUseCase
	MarkDefaultState  MarkDefaultStateUseCase
````

`server/internal/modules/project/adapter/http/states.go`（修改，1 处）：

````old server/internal/modules/project/adapter/http/states.go
}

// state is s as the API shows it.
````
````new server/internal/modules/project/adapter/http/states.go
}

// DeleteState serves DELETE /api/v0/states/{state_id}.
func (h handler) DeleteState(ctx context.Context, req gen.DeleteStateRequestObject) (gen.DeleteStateResponseObject, error) {
	if err := h.uc.DeleteState.Execute(ctx, req.StateID); err != nil {
		return nil, err
	}
	return gen.DeleteState204Response{}, nil
}

// MarkDefaultState serves POST /api/v0/states/{state_id}/mark-default.
func (h handler) MarkDefaultState(ctx context.Context, req gen.MarkDefaultStateRequestObject) (gen.MarkDefaultStateResponseObject, error) {
	if err := h.uc.MarkDefaultState.Execute(ctx, req.StateID); err != nil {
		return nil, err
	}
	return gen.MarkDefaultState204Response{}, nil
}

// state is s as the API shows it.
````

`server/internal/modules/project/adapter/http/handler_test.go`（修改，3 处）：

````old server/internal/modules/project/adapter/http/handler_test.go
	updateState  *fakeUpdateState
````
````new server/internal/modules/project/adapter/http/handler_test.go
	updateState  *fakeUpdateState
	deleteState  *fakeDelete
	markDefault  *fakeDelete
````

````old server/internal/modules/project/adapter/http/handler_test.go
		f.updateState = &fakeUpdateState{}
	}
````
````new server/internal/modules/project/adapter/http/handler_test.go
		f.updateState = &fakeUpdateState{}
	}
	if f.deleteState == nil {
		f.deleteState = &fakeDelete{}
	}
	if f.markDefault == nil {
		f.markDefault = &fakeDelete{}
	}
````

````old server/internal/modules/project/adapter/http/handler_test.go
		CreateState: f.createState, UpdateState: f.updateState})
````
````new server/internal/modules/project/adapter/http/handler_test.go
		CreateState: f.createState, UpdateState: f.updateState, DeleteState: f.deleteState,
		MarkDefaultState: f.markDefault})
````

`server/internal/modules/project/adapter/http/state_writes_test.go`（修改，2 处）：

````old server/internal/modules/project/adapter/http/state_writes_test.go
		`"detail":"The state is the only one of its group, and every group keeps one; add another to the group first."}`
````
````new server/internal/modules/project/adapter/http/state_writes_test.go
		`"detail":"The state is the only one of its group, and every group keeps one; add another to the group first."}`
	stateDefaultJSON = `{"status":409,"code":"project.state_default","title":"Conflict",` +
		`"detail":"The default state cannot be deleted; make another state the default first."}`
````

````old server/internal/modules/project/adapter/http/state_writes_test.go
			t.Errorf("%s: PATCH = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

````
````new server/internal/modules/project/adapter/http/state_writes_test.go
			t.Errorf("%s: PATCH = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

// DELETE goes to the use case for the caller and the path's state, and
// answers 204 with no body; the use case's refusals, as the contract
// declares them, and its failure.
func TestDeleteState(t *testing.T) {
	remove := &fakeDelete{}
	h := newServer(t, fakes{deleteState: remove})
	path := "/api/v0/states/" + webReview.ID.String()
	if res, body := do(t, h, request(http.MethodDelete, path, "alice", "")); res.StatusCode != http.StatusNoContent || body != "" {
		t.Errorf("DELETE = %d %q, want 204 and no body", res.StatusCode, body)
	}
	if want := []string{"alice " + webReview.ID.String()}; !slices.Equal(remove.calls, want) {
		t.Errorf("calls = %q, want %q", remove.calls, want)
	}
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"no state", domain.ErrStateNotFound, http.StatusNotFound, stateNotFoundJSON},
		{"a project member", shared.Forbidden(), http.StatusForbidden, forbiddenJSON},
		{"the default", domain.ErrStateDefault, http.StatusConflict, stateDefaultJSON},
		{"its group's only state", domain.ErrStateLastInGroup, http.StatusConflict, stateLastInGroupJSON},
		{"a failure", errGone, http.StatusInternalServerError, internalErrorJSON},
	} {
		h := newServer(t, fakes{deleteState: &fakeDelete{err: tt.err}})
		if res, body := do(t, h, request(http.MethodDelete, path, "alice", "")); res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: DELETE = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

// POST mark-default goes to the use case for the caller and the path's
// state, and answers 204 with no body; the use case's refusals, as the
// contract declares them, and its failure.
func TestMarkDefaultState(t *testing.T) {
	mark := &fakeDelete{}
	h := newServer(t, fakes{markDefault: mark})
	path := "/api/v0/states/" + webReview.ID.String() + "/mark-default"
	if res, body := do(t, h, request(http.MethodPost, path, "alice", "")); res.StatusCode != http.StatusNoContent || body != "" {
		t.Errorf("POST = %d %q, want 204 and no body", res.StatusCode, body)
	}
	if want := []string{"alice " + webReview.ID.String()}; !slices.Equal(mark.calls, want) {
		t.Errorf("calls = %q, want %q", mark.calls, want)
	}
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"no state", domain.ErrStateNotFound, http.StatusNotFound, stateNotFoundJSON},
		{"a project member", shared.Forbidden(), http.StatusForbidden, forbiddenJSON},
		{"a failure", errGone, http.StatusInternalServerError, internalErrorJSON},
	} {
		h := newServer(t, fakes{markDefault: &fakeDelete{err: tt.err}})
		if res, body := do(t, h, request(http.MethodPost, path, "alice", "")); res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: POST = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

````

`server/internal/modules/project/module.go`（修改，2 处）：

````old server/internal/modules/project/module.go
// member's display settings, creating and changing a project's states,
// carries out the workspace module's cascades on the projects
// (ProjectCascade), and offers the access module its reads of a project
// (ProjectAccess) and the workspace module its count of an account's ended
// project memberships (ProjectMembershipCounts).
````
````new server/internal/modules/project/module.go
// member's display settings, creating, changing and deleting a project's
// states and making one its default, carries out the workspace module's
// cascades on the projects (ProjectCascade), and offers the access module
// its reads of a project (ProjectAccess) and the workspace module its
// count of an account's ended project memberships
// (ProjectMembershipCounts).
````

````old server/internal/modules/project/module.go
		UpdateState:       app.NewUpdateState(locks, store, d.Tx, d.Clock),
````
````new server/internal/modules/project/module.go
		UpdateState:       app.NewUpdateState(locks, store, d.Tx, d.Clock),
		DeleteState:       app.NewDeleteState(locks, store, d.Tx, d.Clock),
		MarkDefaultState:  app.NewMarkDefaultState(locks, store, d.Tx, d.Clock),
````

`web/apps/web/helpers/authentication.helper.ts`（修改，1 处）：

````old web/apps/web/helpers/authentication.helper.ts
  "project.state_last_in_group": "auth.errors.project_state_last_in_group",
````
````new web/apps/web/helpers/authentication.helper.ts
  "project.state_last_in_group": "auth.errors.project_state_last_in_group",
  "project.state_default": "auth.errors.project_state_default",
````

`web/packages/i18n/src/locales/en/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/auth.json
      "project_state_last_in_group": "This is the only state of its group, and every group keeps one. Add another state to the group first.",
````
````new web/packages/i18n/src/locales/en/auth.json
      "project_state_last_in_group": "This is the only state of its group, and every group keeps one. Add another state to the group first.",
      "project_state_default": "The default state cannot be deleted. Make another state the default first.",
````

`web/packages/i18n/src/locales/zh-CN/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/auth.json
      "project_state_last_in_group": "这是它所在分组里唯一的状态，每个分组都要保留一个状态。请先给这个分组添加另一个状态。",
````
````new web/packages/i18n/src/locales/zh-CN/auth.json
      "project_state_last_in_group": "这是它所在分组里唯一的状态，每个分组都要保留一个状态。请先给这个分组添加另一个状态。",
      "project_state_default": "默认状态不能删除。请先把另一个状态设为默认。",
````

- [ ] **Step 5: 矩阵和组合测试**

`server/internal/bootstrap/permission_matrix_states_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_states_test.go
	cellStateLastInGroup = cell{http.StatusConflict, "project.state_last_in_group"}
````
````new server/internal/bootstrap/permission_matrix_states_test.go
	cellStateLastInGroup = cell{http.StatusConflict, "project.state_last_in_group"}
	cellStateDefault     = cell{http.StatusConflict, "project.state_default"}
````

````old server/internal/bootstrap/permission_matrix_states_test.go
			cells: ofArchivedState(cellOK, cellForbidden), check: renamesTheState},
````
````new server/internal/bootstrap/permission_matrix_states_test.go
			cells: ofArchivedState(cellOK, cellForbidden), check: renamesTheState},
		// As createState: Review deleted, In Progress kept in its group.
		{op: "deleteState", write: true, columns: projectColumns, request: toState(http.MethodDelete, "", "Review", ""),
			cells: ofState(cellNoContent, cellForbidden, cellForbidden, cellNoContent, cellForbidden, cellForbidden)},
		// The default, and Done, the only state of its group: each 409 comes
		// after the decision (M3 design 3.17).
		{op: "deleteState", variant: "the default", write: true, columns: projectColumns, request: toState(http.MethodDelete, "", "Backlog", ""),
			cells: ofState(cellStateDefault, cellForbidden, cellForbidden, cellStateDefault, cellForbidden, cellForbidden)},
		{op: "deleteState", variant: "the last of its group", write: true, columns: projectColumns, request: toState(http.MethodDelete, "", "Done", ""),
			cells: ofState(cellStateLastInGroup, cellForbidden, cellForbidden, cellStateLastInGroup, cellForbidden, cellForbidden)},
		{op: "deleteState", variant: "the triage state", write: true, columns: projectColumns, request: toState(http.MethodDelete, "", "Triage", ""),
			cells: ofState(cellStateNotFound, cellStateNotFound, cellStateNotFound, cellStateNotFound, cellStateNotFound, cellStateNotFound)},
		{op: "deleteState", variant: "archived", write: true, columns: archivedColumns, request: toState(http.MethodDelete, "", "Review", ""),
			cells: ofArchivedState(cellNoContent, cellForbidden)},
		// As createState: Todo made the default.
		{op: "markDefaultState", write: true, columns: projectColumns, request: toState(http.MethodPost, "/mark-default", "Todo", ""),
			cells: ofState(cellNoContent, cellForbidden, cellForbidden, cellNoContent, cellForbidden, cellForbidden)},
		{op: "markDefaultState", variant: "the triage state", write: true, columns: projectColumns,
			request: toState(http.MethodPost, "/mark-default", "Triage", ""), cells: ofState(cellStateNotFound, cellStateNotFound, cellStateNotFound,
				cellStateNotFound, cellStateNotFound, cellStateNotFound)},
		{op: "markDefaultState", variant: "archived", write: true, columns: archivedColumns,
			request: toState(http.MethodPost, "/mark-default", "Todo", ""), cells: ofArchivedState(cellNoContent, cellForbidden)},
````

`server/internal/bootstrap/project_write_locks_test.go`（修改，1 处）：

````old server/internal/bootstrap/project_write_locks_test.go
	{op: "updateState", method: http.MethodPatch, path: "/api/v0/states/%s", body: `{"name":"Checked"}`, want: http.StatusOK, row: stateNamed("QA")},
````
````new server/internal/bootstrap/project_write_locks_test.go
	{op: "updateState", method: http.MethodPatch, path: "/api/v0/states/%s", body: `{"name":"Checked"}`, want: http.StatusOK, row: stateNamed("QA")},
	// The state renamed before, deleted.
	{op: "deleteState", method: http.MethodDelete, path: "/api/v0/states/%s", want: http.StatusNoContent, row: stateNamed("Checked")},
	{op: "markDefaultState", method: http.MethodPost, path: "/api/v0/states/%s/mark-default", want: http.StatusNoContent, row: stateNamed("Done")},
````

`server/internal/bootstrap/project_writes_test.go`（修改，5 处）：

````old server/internal/bootstrap/project_writes_test.go
// an admin of Web, removes carol, creates a state and renames it, and
// leaves Web. Each row the write writes again is first made bob's, as last
// written by him, and checked so: a write that kept its row's writer would
// pass for alice's otherwise, she having made it.
````
````new server/internal/bootstrap/project_writes_test.go
// an admin of Web, removes carol, creates a state, renames it and deletes
// it, makes Done the default, and leaves Web. Each row the write writes
// again is first made bob's, as last written by him, and checked so: a
// write that kept its row's writer would pass for alice's otherwise, she
// having made it.
````

````old server/internal/bootstrap/project_writes_test.go
	// bobs makes bob, $3, the last writer of the one row of table that where
````
````new server/internal/bootstrap/project_writes_test.go
	// bobs makes bob, $3, the last writer of the rows of table that where
````

````old server/internal/bootstrap/project_writes_test.go
		// id, $2 alice's and $3 bob's: one row, none of alice's writing.
````
````new server/internal/bootstrap/project_writes_test.go
		// id, $2 alice's and $3 bob's: the seeded rows, none of alice's
		// writing.
````

````old server/internal/bootstrap/project_writes_test.go
			"SELECT updated_at, updated_by_id = $2 AND name = 'Checked' FROM states WHERE project_id = $1 AND name IN ('QA', 'Checked')", 1, 1},
````
````new server/internal/bootstrap/project_writes_test.go
			"SELECT updated_at, updated_by_id = $2 AND name = 'Checked' FROM states WHERE project_id = $1 AND name IN ('QA', 'Checked')", 1, 1},
		// Checked, deleted.
		{"deleteState", http.MethodDelete, "/api/v0/states/%s", "", state("Checked"), http.StatusNoContent,
			bobs("states", "project_id = $1 AND name = 'Checked'"),
			"SELECT updated_at, updated_by_id = $2 AND deleted_at = updated_at FROM states WHERE project_id = $1 AND name = 'Checked'", 1, 1},
		// Done made the default, and Backlog the default no longer: two rows.
		{"markDefaultState", http.MethodPost, "/api/v0/states/%s/mark-default", "", state("Done"), http.StatusNoContent,
			bobs("states", "project_id = $1 AND name IN ('Backlog', 'Done')"),
			`SELECT updated_at, updated_by_id = $2 AND "default" = (name = 'Done') FROM states WHERE project_id = $1 AND name IN ('Backlog', 'Done')`,
			2, 2},
````

````old server/internal/bootstrap/project_writes_test.go
			if tag, err := pool.Exec(context.Background(), w.seed, web, aliceID, bobID); err != nil || tag.RowsAffected() != 1 {
				t.Fatalf("%s's seed: %v, %v; want one row written", w.name, tag, err)
````
````new server/internal/bootstrap/project_writes_test.go
			if tag, err := pool.Exec(context.Background(), w.seed, web, aliceID, bobID); err != nil || tag.RowsAffected() != int64(w.seeded) {
				t.Fatalf("%s's seed: %v, %v; want %d rows written", w.name, tag, err, w.seeded)
````

`server/internal/bootstrap/project_connection_test.go`（修改，3 处）：

````old server/internal/bootstrap/project_connection_test.go
// admin of Web and removes bob, and creates a state in it and renames it;
// carol, an admin, leaves it; alice deletes both projects.
````
````new server/internal/bootstrap/project_connection_test.go
// admin of Web and removes bob, creates two states in it, renames the
// first and deletes it, and makes the second Web's default; carol, an
// admin, leaves it; alice deletes both projects.
````

````old server/internal/bootstrap/project_connection_test.go
	var qa struct {
````
````new server/internal/bootstrap/project_connection_test.go
	var qa, done struct {
````

````old server/internal/bootstrap/project_connection_test.go
	send(http.MethodPatch, "/api/v0/states/"+qa.ID.String(), r.alice, `{"name":"Checked"}`, http.StatusOK)
````
````new server/internal/bootstrap/project_connection_test.go
	send(http.MethodPatch, "/api/v0/states/"+qa.ID.String(), r.alice, `{"name":"Checked"}`, http.StatusOK)
	decodeAnswer(t, send(http.MethodPost, web+"/states", r.alice, `{"name":"Done","color":"#46A758","group":"completed"}`, http.StatusCreated),
		&done)
	send(http.MethodDelete, "/api/v0/states/"+qa.ID.String(), r.alice, "", http.StatusNoContent)
	send(http.MethodPost, "/api/v0/states/"+done.ID.String()+"/mark-default", r.alice, "", http.StatusNoContent)
````

- [ ] **Step 6: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/project/... ./internal/modules/access/...`
Expected: 全部 `ok`（`apitest.Main` 两个方向：两个操作的码）。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix$|TestEachWriteOnAProjectSharesItsWorkspaceFirst$|TestTheWritesOnAProjectStampTheirRequest$|TestTheWritesOnAProjectRunOnTheirTransactionsConnection$' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过（关键词守卫 5 个命中都有例外）。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

- [ ] **Step 7: 提交**

```bash
git add api/modules/project.yaml api/openapi.yaml server/internal/bootstrap/permission_matrix_states_test.go server/internal/bootstrap/project_connection_test.go server/internal/bootstrap/project_write_locks_test.go server/internal/bootstrap/project_writes_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/project/adapter/http/handler.go server/internal/modules/project/adapter/http/handler_test.go server/internal/modules/project/adapter/http/state_writes_test.go server/internal/modules/project/adapter/http/states.go server/internal/modules/project/app/clock_test.go server/internal/modules/project/app/delete_state.go server/internal/modules/project/app/delete_state_test.go server/internal/modules/project/app/fakes_state_test.go server/internal/modules/project/app/fakes_write_test.go server/internal/modules/project/app/mark_default_state.go server/internal/modules/project/app/mark_default_state_test.go server/internal/modules/project/app/state_ports.go server/internal/modules/project/domain/actions.go server/internal/modules/project/module.go web/apps/web/helpers/authentication.helper.ts web/packages/i18n/src/locales/en/auth.json web/packages/i18n/src/locales/zh-CN/auth.json api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P7a): deleteState and markDefaultState

A deletion is a guarded write that passes over the project's default
state: the state read again under the locks then answers
project.state_default, or its own 404 when it is gone; the states its
group keeps are counted after it, and none left is
project.state_last_in_group, rolled back; a state both the default and
its group's only one is project.state_default. Making a state the default
clears the old default, then sets the new one; a second statement that
wrote nothing fails the transaction, so no project is left without a
default. project.state_default is declared, with its messages.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p7a.py` 的编号；"层"是它被发现的每一层，端到端是单独运行的故事）：

| 变异 | 改坏 | 必须失败的测试 | 层 |
|---|---|---|---|
| `s2-delete-delete` | `deleteState` 吞掉 `DeleteState` 的失败 | `TestDeleteStateReturnsEachFailure` | 单元（按性质只在单元一层：真实的存储在这一步不失败） |
| `s2-delete-count` | `deleteState` 不管 `CountGroupStates` 的失败 | `TestDeleteStateReturnsEachFailure` | 单元（按性质只在单元一层：真实的存储在这一步不失败） |
| `s2-delete-reread` | `deletedNothing` 的重读失败答成默认状态 | `TestDeleteStateChecksTheReadAfterIt`、`TestDeleteStateRefuses` | 单元（按性质只在单元一层：真实的存储在这一步不失败） |
| `s2-mark` | `markDefaultState` 吞掉 `MarkDefaultState` 的失败 | `TestMarkDefaultStateReturnsEachFailure` | 单元（按性质只在单元一层：真实的存储在这一步不失败） |
| `c-delete-early` | `deleteState` 在取锁之前读时钟 | `TestDeleteState`、`TestDeleteStateChecksTheReadAfterIt`、`TestDeleteStateRefuses`、`TestDeleteStateReturnsEachFailure` 等 5 个、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`（Task 9 起） | 单元；组合 |
| `c-mark-early` | `markDefaultState` 在取锁之前读时钟 | `TestEachWriteReadsTheClockUnderItsLock`、`TestMarkDefaultState`、`TestMarkDefaultStateRefuses`、`TestMarkDefaultStateReturnsEachFailure`、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`（Task 9 起） | 单元；组合 |
| `g-delete-unchecked` | `deleteState` 不查组里还剩几个 | `TestDeleteStateRefuses`、`TestPermissionMatrix`、`TestStateWritesOnOneProjectSerialize`（Task 9 起）、P6（Task 10 起） | 单元；组合；端到端 |
| `g-delete-default-answers-404` | 守卫跳过默认状态时答 404 | `TestDeleteStateRefuses`、`TestPermissionMatrix`、`TestStateWritesOnOneProjectSerialize`（Task 9 起）、P6（Task 10 起） | 单元；组合；端到端 |
| `g-delete-nothing-default` | 守卫删了 0 行时不重读、一律答默认状态 | `TestDeleteStateChecksTheReadAfterIt`、`TestDeleteStateRefuses` | 单元（按性质只在单元一层：锁下的重读） |
| `g-mark-nothing-ok` | 第二条语句写了 0 行时照样提交（O4） | `TestMarkDefaultStateRefuses` | 单元（按性质只在单元一层：组合一层到不了（O4）） |
| `s21-default-wrapped` | 默认状态的拒绝包在 404 之后 | `TestDeleteStateRefuses`、`TestPermissionMatrix`、`TestStateWritesOnOneProjectSerialize`（Task 9 起）、P6（Task 10 起） | 单元；组合；端到端 |
| `s4-delete-frozen-clock` | `DeleteState` 接上停在 2001 年的时钟 | `TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`（Task 9 起）、`TestTheWritesOnAProjectStampTheirRequest` | 组合 |
| `s4-mark-frozen-clock` | `MarkDefaultState` 接上停在 2001 年的时钟 | `TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`（Task 9 起）、`TestTheWritesOnAProjectStampTheirRequest` | 组合 |
| `s4-mark-serves-delete` | `POST /mark-default` 调用 `deleteState` | `TestMarkDefaultState`、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`（Task 9 起）、`TestEachWriteOnAProjectSharesItsWorkspaceFirst`、`TestPermissionMatrix`、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection` 等 5 个、P6（Task 10 起） | 单元；组合；端到端 |
| `s5-delete-members` | 规则表：`state.delete` 也给项目成员 | `TestEveryRuleDecidesItsCells`、`TestPermissionMatrix` | 单元；组合 |
| `s5-mark-members` | 规则表：`state.mark_default` 也给项目成员 | `TestEveryRuleDecidesItsCells`、`TestPermissionMatrix` | 单元；组合 |
| `s18-delete-no-default` | `deleteState` 不声明 `project.state_default` | `TestDeleteState`、`TestPermissionMatrix` | 单元；组合 |
| `s18-update-extra-default` | `updateState` 多声明 `project.state_default` | `apitest.Main` | 单元（按性质只在单元一层：多声明、从不回答的码，只有 `apitest.Main` 的覆盖核对看得出） |
| `s34-first-lock-row` | 最先锁工作区的测试少了 `deleteState` 一行 | `TestEachWriteOnAProjectSharesItsWorkspaceFirst` | 组合 |

---

### Task 8: 两个读：`listStates`、`listWorkspaceStates`；`findWorkspaceAndDecide`

**Files:**
- Create: `server/internal/modules/project/app/list_states.go`、`server/internal/modules/project/app/list_states_test.go`、`server/internal/modules/project/app/list_workspace_states.go`、`server/internal/modules/project/app/list_workspace_states_test.go`
- Modify: `api/modules/project.yaml`、`api/openapi.yaml`、`server/internal/bootstrap/permission_matrix_states_test.go`、`server/internal/bootstrap/project_visibility_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/project/adapter/http/handler.go`、`server/internal/modules/project/adapter/http/handler_test.go`、`server/internal/modules/project/adapter/http/states.go`、`server/internal/modules/project/adapter/http/states_test.go`、`server/internal/modules/project/app/check_identifier.go`、`server/internal/modules/project/app/fakes_state_test.go`、`server/internal/modules/project/app/list_projects.go`、`server/internal/modules/project/app/lock.go`、`server/internal/modules/project/app/state_ports.go`、`server/internal/modules/project/domain/actions.go`、`server/internal/modules/project/module.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.9；M3 设计 3.4、3.12、3.17、6.4、9.2）：
  - 契约：`GET /api/v0/projects/{project_id}/states`（`listStates`，200 `StateList`；`x-problem-codes: [project.not_found, forbidden]`）、`GET /api/v0/workspaces/{slug}/states`（`listWorkspaceStates`，200 `StateList`；`[workspace.not_found]`）；`StateList{data: [State]}`。
  - `access`：`state.list`（项目级，项目的每个有效成员）、`workspace_state.list`（工作区级，每个有效成员；列表只含他是有效成员的项目的状态，可见性不放宽：Plane `views/workspace/state.py:20-26`）。`ActionStateList`、`ActionWorkspaceStateList`。
  - 端口：`StateLister`（`ProjectFinder`、`ListStates`）、`WorkspaceStateLister`（`ListWorkspaceStates`）。
  - `ListStates`（`list_states.go`）：`NewListStates(states StateLister, auth shared.Authorizer)`；`Execute(ctx, projectID) ([]domain.State, error)`：`findAndDecide`（`state.list`）→ `ListStates`，存储的顺序原样；不开事务。
  - `findWorkspaceAndDecide`（`lock.go`）：按 slug 找到未删除的工作区、判定工作区级的操作，回答工作区和授权；没有、看不到的是 `ErrWorkspaceNotFound`。`listProjects`、`checkProjectIdentifier` 原来各写一遍的这两步改用它（行为不变）。
  - `ListWorkspaceStates`（`list_workspace_states.go`）：`NewListWorkspaceStates(workspaces WorkspaceDirectory, states WorkspaceStateLister, auth shared.Authorizer)`；`Execute(ctx, slug)`：`findWorkspaceAndDecide`（`workspace_state.list`）→ `ListWorkspaceStates(ws.ID, actor.UserID)`，不论角色。
  - HTTP：`UseCases.ListStates`、`UseCases.ListWorkspaceStates`，两个处理函数；`project.New` 接上两个用例；包说明。
  - 矩阵：`listStates` 两行（每个有效成员 200，`PM+WA` 作为成员、`WA-` 不是成员 403；已归档项目的小表 PA 200 空列表、WM 403）、`listWorkspaceStates` 一行（每个有效成员 200；`listsTheWorkspaceStates`：管理员和成员不是任何矩阵项目的成员，列表为空；访客是公开和私有项目的成员，按项目 id 的顺序得两个项目的状态）。

**Tests:**
- `TestListStates`（`list_states_test.go`，三行：存储的顺序按 `sequence`、反过来、按 id，用例原样回答：排序的用例答错；不开事务）、`TestListStatesRefuses`（7 行：没有调用者；没有项目、看不到 404；看得到不是成员 403；三个端口调用的失败各原样返回，之后什么都不运行）。
- `TestListWorkspaceStates`（`list_workspace_states_test.go`，四行：存储的两种次序各原样回答，不论调用者的角色）、`TestListWorkspaceStatesRefuses`（6 行：没有调用者什么都不读；没有工作区、不是它的成员 `workspace.not_found`；三个调用的失败各原样返回）。
- HTTP：`TestListStates`、`TestListWorkspaceStates`（`states_test.go`：用例收到调用者和路径的项目或工作区；200，列表的次序、每个字段，`[]`；拒绝和失败照契约声明的经 `CheckResponse`）。
- 组合：`TestPermissionMatrix` 的三行；`TestListingWorkspaceStatesIsListingEachProjects`（新，`project_visibility_test.go`：矩阵的每个账户，`listWorkspaceStates` 列出的恰好是 `listStates` 对 acme 每个项目列出的，按项目 id 的顺序：工作区的列表在它的查询里读成员关系，项目的列表在 `access` 的规则里读，这里两者不会分开）；`TestListingWorkspaceStatesKeepsToItsWorkspace`（新，同一文件：在 `memberWorld` 上，bob、carol 是 acme 的 Web、Ops 和 beta 的 Lab 的成员，Web 有一个 bob 建了又删除的 Dropped；每个账户每个工作区的列表恰好是它自己的项目逐个 `listStates` 的并：别的工作区的项目、已删除的状态都不在里面（清扫 23））。

- [ ] **Step 1: 契约**

`api/modules/project.yaml`（修改，3 处）：

````old api/modules/project.yaml
  /api/v0/projects/{project_id}/states:
    parameters:
      - $ref: '#/components/parameters/ProjectID'
    post:
````
````new api/modules/project.yaml
  /api/v0/projects/{project_id}/states:
    parameters:
      - $ref: '#/components/parameters/ProjectID'
    get:
      operationId: listStates
      tags: [project]
      summary: List a project's states
      description: >-
        For the project's active members: its states, by sequence, the
        lowest first, then by id. The intake's triage state is none of
        them. An archived project lists none. A project that does not exist,
        is deleted, or that the caller does not see answers
        project.not_found; one he sees but is not a member of, forbidden.
        The whole collection at once: collections are not paginated.
      security: [{bearer: []}]
      x-problem-codes: [project.not_found, forbidden]
      responses:
        '200':
          description: The project's states.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/StateList'
        default:
          $ref: '#/components/responses/Problem'
    post:
````

````old api/modules/project.yaml
          description: The state is the project's default.
````
````new api/modules/project.yaml
          description: The state is the project's default.
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/workspaces/{slug}/states:
    parameters:
      - $ref: '#/components/parameters/Slug'
    get:
      operationId: listWorkspaceStates
      tags: [project]
      summary: List the states of the projects the caller is a member of
      description: >-
        For the workspace's active members: the states of its projects that
        the caller is an active member of, the archived ones left out, by
        project, then by sequence, the lowest first, then by id; each
        project's as listStates lists them, the intake's triage state none of
        them. A project the caller sees but is not a member of adds none. A
        workspace that does not exist, is deleted, or of which the caller is
        not an active member answers workspace.not_found. The whole
        collection at once: collections are not paginated.
      security: [{bearer: []}]
      x-problem-codes: [workspace.not_found]
      responses:
        '200':
          description: The states.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/StateList'
````

````old api/modules/project.yaml
          type: string
          format: date-time
    StateCreate:
````
````new api/modules/project.yaml
          type: string
          format: date-time
    StateList:
      type: object
      additionalProperties: false
      required: [data]
      properties:
        data:
          type: array
          items:
            $ref: '#/components/schemas/State'
    StateCreate:
````

`api/openapi.yaml`（修改，1 处）：

````old api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1states~1{state_id}~1mark-default'
````
````new api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1states~1{state_id}~1mark-default'
  /api/v0/workspaces/{slug}/states:
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1workspaces~1{slug}~1states'
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `36793961b3418a40d650993813bb0db5fea430e82ba3e236de0b3a2043c6c499` | 2752 | `api/dist/openapi.yaml` |
| `d7c981b40489088756fc1fe8d8b661dfa1cef627092a0cbbd6dbdc625145807e` | 3016 | `server/internal/modules/project/adapter/http/gen/server.gen.go` |
| `63f6ef50c96cf8b3212b9a7a7fad47be633fbe15dc82a260718b4635b603e54d` | 3057 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml web/packages/api-client/src/schema.gen.ts server/internal/modules/project/adapter/http/gen/server.gen.go`
Expected: 与上表相同。

- [ ] **Step 2: 规则、操作名、端口、用例和测试**

`server/internal/modules/access/domain/rules.go`（修改，2 处）：

````old server/internal/modules/access/domain/rules.go
	"project_identifier.check": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
````
````new server/internal/modules/access/domain/rules.go
	"project_identifier.check": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
	// Every active member; the list holds the states of the projects he is
	// an active member of, those state.list lets him list (M3 design 3.4).
	"workspace_state.list": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
````

````old server/internal/modules/access/domain/rules.go
	"project.leave": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
````
````new server/internal/modules/access/domain/rules.go
	"project.leave": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// Every active member of the project (M3 design 9.2).
	"state.list": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，2 处）：

````old server/internal/modules/access/domain/rules_test.go
	"project_identifier.check":     {allowed, allowed, forbidden, invisible, invisible, invisible, forbidden},
````
````new server/internal/modules/access/domain/rules_test.go
	"project_identifier.check":     {allowed, allowed, forbidden, invisible, invisible, invisible, forbidden},
	"workspace_state.list":         {allowed, allowed, allowed, invisible, invisible, invisible, forbidden},
````

````old server/internal/modules/access/domain/rules_test.go
	"project.leave": {allowed, allowed, allowed, allowed, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
````
````new server/internal/modules/access/domain/rules_test.go
	"project.leave": {allowed, allowed, allowed, allowed, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
	// As project_member.list.
	"state.list": {allowed, allowed, allowed, allowed, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
````

`server/internal/modules/project/domain/actions.go`（修改，3 处）：

````old server/internal/modules/project/domain/actions.go
	ActionLeave shared.Action = "project.leave"
````
````new server/internal/modules/project/domain/actions.go
	ActionLeave shared.Action = "project.leave"
	// ActionStateList is listing a project's states: listStates.
	ActionStateList shared.Action = "state.list"
````

````old server/internal/modules/project/domain/actions.go
	ActionStateMarkDefault shared.Action = "state.mark_default"
````
````new server/internal/modules/project/domain/actions.go
	ActionStateMarkDefault shared.Action = "state.mark_default"
	// ActionWorkspaceStateList is listing the states of a workspace's
	// projects that the caller is a member of: listWorkspaceStates.
	ActionWorkspaceStateList shared.Action = "workspace_state.list"
````

````old server/internal/modules/project/domain/actions.go
		ActionStateCreate, ActionStateUpdate, ActionStateDelete,
		ActionStateMarkDefault}
````
````new server/internal/modules/project/domain/actions.go
		ActionStateList, ActionStateCreate, ActionStateUpdate, ActionStateDelete, ActionStateMarkDefault, ActionWorkspaceStateList}
````

`server/internal/modules/project/app/state_ports.go`（修改，1 处）：

````old server/internal/modules/project/app/state_ports.go
// (M3 design 3.17). The triage state is never one of them.
````
````new server/internal/modules/project/app/state_ports.go
// (M3 design 3.17). The triage state is never one of them.

// StateLister is listStates' repository.
type StateLister interface {
	ProjectFinder
	// ListStates lists projectID's undeleted states but its triage state, by
	// sequence, then id; none while the project is archived.
	ListStates(ctx context.Context, projectID uuid.UUID) ([]domain.State, error)
}

// WorkspaceStateLister is listWorkspaceStates' repository.
type WorkspaceStateLister interface {
	// ListWorkspaceStates lists the undeleted states but the triage states
	// of workspaceID's undeleted, unarchived projects that userID is an
	// active member of, by project, then sequence, then id.
	ListWorkspaceStates(ctx context.Context, workspaceID, userID uuid.UUID) ([]domain.State, error)
}
````

`server/internal/modules/project/app/lock.go`（修改，1 处）：

````old server/internal/modules/project/app/lock.go
}

// decide asks the Authorizer for action on the project id of the workspace
// for actor. A write calls it under its locks, so the facts it reads are
// the ones committed after the locks were granted (M3 design 6.7): a
// demotion or a removal that committed while the write waited is seen; a
// read calls it after the read that names the project's workspace. A
// project not visible to actor is notFound, the 404 of what the caller
// named.
````
````new server/internal/modules/project/app/lock.go
}

// findWorkspaceAndDecide is the first two steps of a read in a workspace
// named by its slug (M3 design 6.4), without a transaction: workspaces
// finds the undeleted workspace, then decide decides action in it, and
// answers the grant. A workspace that is not there, or not visible to
// actor, is domain.ErrWorkspaceNotFound.
func findWorkspaceAndDecide(ctx context.Context, workspaces WorkspaceDirectory, auth shared.Authorizer, actor shared.Actor, slug string,
	action shared.Action) (Workspace, shared.Grant, error) {
	ws, found, err := workspaces.WorkspaceBySlug(ctx, slug)
	switch {
	case err != nil:
		return Workspace{}, shared.Grant{}, err
	case !found:
		return Workspace{}, shared.Grant{}, domain.ErrWorkspaceNotFound
	}
	grant, err := decide(ctx, auth, actor, action, ws.ID, uuid.UUID{}, domain.ErrWorkspaceNotFound)
	if err != nil {
		return Workspace{}, shared.Grant{}, err
	}
	return ws, grant, nil
}

// decide asks the Authorizer for action on the project id of the workspace
// for actor, or in the workspace for a zero id. A write calls it under its
// locks, so the facts it reads are the ones committed after the locks were
// granted (M3 design 6.7): a demotion or a removal that committed while the
// write waited is seen; a read calls it after the read that names the
// project's workspace. A project, or a workspace, not visible to actor is
// notFound, the 404 of what the caller named.
````

`server/internal/modules/project/app/list_projects.go`（修改，2 处）：

````old server/internal/modules/project/app/list_projects.go
	"context"
	"errors"
````
````new server/internal/modules/project/app/list_projects.go
	"context"
````

````old server/internal/modules/project/app/list_projects.go
	ws, found, err := u.workspaces.WorkspaceBySlug(ctx, slug)
	switch {
	case err != nil:
		return nil, err
	case !found:
		return nil, domain.ErrWorkspaceNotFound
	}
	grant, err := u.auth.Authorize(ctx, actor, domain.ActionList, shared.Target{WorkspaceID: ws.ID})
	switch {
	case errors.Is(err, shared.ErrNotVisible):
		return nil, domain.ErrWorkspaceNotFound
	case err != nil:
````
````new server/internal/modules/project/app/list_projects.go
	ws, grant, err := findWorkspaceAndDecide(ctx, u.workspaces, u.auth, actor, slug, domain.ActionList)
	if err != nil {
````

`server/internal/modules/project/app/check_identifier.go`（修改，2 处）：

````old server/internal/modules/project/app/check_identifier.go
	"context"
	"errors"
````
````new server/internal/modules/project/app/check_identifier.go
	"context"
````

````old server/internal/modules/project/app/check_identifier.go
	ws, found, err := u.workspaces.WorkspaceBySlug(ctx, slug)
	switch {
	case err != nil:
		return false, err
	case !found:
		return false, domain.ErrWorkspaceNotFound
	}
	_, err = u.auth.Authorize(ctx, actor, domain.ActionCheckIdentifier, shared.Target{WorkspaceID: ws.ID})
	switch {
	case errors.Is(err, shared.ErrNotVisible):
		return false, domain.ErrWorkspaceNotFound
	case err != nil:
````
````new server/internal/modules/project/app/check_identifier.go
	ws, _, err := findWorkspaceAndDecide(ctx, u.workspaces, u.auth, actor, slug, domain.ActionCheckIdentifier)
	if err != nil {
````

`server/internal/modules/project/app/list_states.go`（新文件，35 行）：

````file server/internal/modules/project/app/list_states.go
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListStates lists a project's states: GET
// /api/v0/projects/{project_id}/states.
type ListStates struct {
	states StateLister
	auth   shared.Authorizer
}

// NewListStates returns the use case.
func NewListStates(states StateLister, auth shared.Authorizer) *ListStates {
	return &ListStates{states: states, auth: auth}
}

// Execute lists the project's states but its triage state, by sequence,
// none while it is archived (M3 design 3.12, 3.17), after the decision on
// state.list. A read opens no transaction.
func (u *ListStates) Execute(ctx context.Context, projectID uuid.UUID) ([]domain.State, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	if err := findAndDecide(ctx, u.states, u.auth, actor, projectID, domain.ActionStateList); err != nil {
		return nil, err
	}
	return u.states.ListStates(ctx, projectID)
}
````

`server/internal/modules/project/app/list_workspace_states.go`（新文件，40 行）：

````file server/internal/modules/project/app/list_workspace_states.go
package app

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListWorkspaceStates lists the states of a workspace's projects that the
// caller is a member of: GET /api/v0/workspaces/{slug}/states (M3 design
// 3.4, 3.17).
type ListWorkspaceStates struct {
	workspaces WorkspaceDirectory
	states     WorkspaceStateLister
	auth       shared.Authorizer
}

// NewListWorkspaceStates returns the use case.
func NewListWorkspaceStates(workspaces WorkspaceDirectory, states WorkspaceStateLister, auth shared.Authorizer) *ListWorkspaceStates {
	return &ListWorkspaceStates{workspaces: workspaces, states: states, auth: auth}
}

// Execute finds the workspace slug names, decides workspace_state.list in
// it, then lists the states but the triage states of its unarchived
// projects that the caller is an active member of: those whose states
// state.list lets him list, which no visibility widens (Plane
// views/workspace/state.py:20-26). A workspace not there, or not visible to
// the caller, is domain.ErrWorkspaceNotFound. A read opens no transaction.
func (u *ListWorkspaceStates) Execute(ctx context.Context, slug string) ([]domain.State, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	ws, _, err := findWorkspaceAndDecide(ctx, u.workspaces, u.auth, actor, slug, domain.ActionWorkspaceStateList)
	if err != nil {
		return nil, err
	}
	return u.states.ListWorkspaceStates(ctx, ws.ID, actor.UserID)
}
````

`server/internal/modules/project/app/fakes_state_test.go`（修改，1 处）：

````old server/internal/modules/project/app/fakes_state_test.go
	slices.SortFunc(out, func(a, b domain.State) int { return cmp.Compare(a.Name, b.Name) })
	return out
````
````new server/internal/modules/project/app/fakes_state_test.go
	slices.SortFunc(out, func(a, b domain.State) int { return cmp.Compare(a.Name, b.Name) })
	return out
}

func (f *fakeStates) ListStates(ctx context.Context, projectID uuid.UUID) ([]domain.State, error) {
	f.log.add(ctx, "ListStates %s", projectID)
	if err := f.fail("ListStates"); err != nil {
		return nil, err
	}
	return f.of(projectID), nil
````

`server/internal/modules/project/app/list_states_test.go`（新文件，86 行）：

````file server/internal/modules/project/app/list_states_test.go
package app_test

import (
	"cmp"
	"context"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// statesListed are the calls of user's list of project's states: the
// project's workspace, the decision, the list; none in a transaction.
func statesListed(user, project uuid.UUID) []string {
	return []string{"ProjectWorkspace " + project.String() + " outside tx",
		fmt.Sprintf("Authorize %s %s on %s/%s outside tx", user, domain.ActionStateList, acme.ID, project),
		fmt.Sprintf("ListStates %s outside tx", project)}
}

// ListStates decides state.list on the project, then answers the store's
// list as it is, in the store's order, which no sort by sequence, either
// way, nor by id gives: a use case that sorted the states would answer
// another. A read opens no transaction.
func TestListStates(t *testing.T) {
	f, s := newStates()
	want := s.of(webID)
	for _, by := range []struct {
		name  string
		order func(a, b domain.State) int
	}{
		{"the sequence's", func(a, b domain.State) int { return cmp.Compare(a.Sequence, b.Sequence) }},
		{"the sequence's, reversed", func(a, b domain.State) int { return cmp.Compare(b.Sequence, a.Sequence) }},
		{"the id's", func(a, b domain.State) int { return slices.Compare(a.ID[:], b.ID[:]) }},
	} {
		if slices.IsSortedFunc(want, by.order) {
			t.Fatalf("the store's order %v is %s: a use case that sorted so would pass", want, by.name)
		}
	}
	got, err := app.NewListStates(s, f.auth).Execute(as(bob), webID)
	if err != nil || !reflect.DeepEqual(got, want) || !slices.Equal(f.log.calls, statesListed(bob, webID)) {
		t.Errorf("Execute() = %+v, %v, calls %q; want %+v, calls %q", got, err, f.log.calls, want, statesListed(bob, webID))
	}
}

// Refusals, each in its place: no caller; a project not there, or not
// visible: 404; the Authorizer's 403 for one who sees it and is not its
// member. Every port's failure comes back as itself, after the calls
// before it and none after.
func TestListStatesRefuses(t *testing.T) {
	tests := []struct {
		name  string
		ctx   context.Context
		id    uuid.UUID
		fail  func(f *writeFixture)
		want  error
		calls []string
	}{
		{"no caller", context.Background(), webID, nil, shared.Unauthenticated(), nil},
		{"no project", as(bob), uuid.Nil(), nil, domain.ErrNotFound, []string{"ProjectWorkspace " + uuid.Nil().String() + " outside tx"}},
		{"not seen", as(erin), webID, nil, domain.ErrNotFound, statesListed(erin, webID)[:2]},
		{"forbidden", as(alice), webID, nil, shared.Forbidden(), statesListed(alice, webID)[:2]},
		{"the project failing", as(bob), webID, func(f *writeFixture) { f.store.errs = map[string]error{"ProjectWorkspace": errDisk} }, errDisk,
			statesListed(bob, webID)[:1]},
		{"the decision failing", as(bob), webID, func(f *writeFixture) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, errDisk,
			statesListed(bob, webID)[:2]},
		{"the list failing", as(bob), webID, func(f *writeFixture) { f.store.errs = map[string]error{"ListStates": errDisk} }, errDisk,
			statesListed(bob, webID)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, s := newStates()
			if tt.fail != nil {
				tt.fail(f)
			}
			got, err := app.NewListStates(s, f.auth).Execute(tt.ctx, tt.id)
			if !refusedAs(err, tt.want) || got != nil || !slices.Equal(f.log.calls, tt.calls) {
				t.Errorf("Execute() = %+v, %v, calls %q; want %v, calls %q", got, err, f.log.calls, tt.want, tt.calls)
			}
		})
	}
}
````

`server/internal/modules/project/app/list_workspace_states_test.go`（新文件，114 行）：

````file server/internal/modules/project/app/list_workspace_states_test.go
package app_test

import (
	"cmp"
	"context"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeWorkspaceStates answers its list, logs each call, and fails with err,
// answering nothing then, as the store does.
type fakeWorkspaceStates struct {
	log  *callLog
	list []domain.State
	err  error
}

func (f *fakeWorkspaceStates) ListWorkspaceStates(ctx context.Context, workspaceID, userID uuid.UUID) ([]domain.State, error) {
	f.log.add(ctx, "ListWorkspaceStates %s for %s", workspaceID, userID)
	if f.err != nil {
		return nil, f.err
	}
	return f.list, nil
}

// newListWorkspaceStates is ListWorkspaceStates over fakes sharing one log:
// in acme, alice is a member, bob an admin, carol a guest. The store's list
// is web's Review, ops's Backlog, then web's Todo: in no order a sort by
// project or by sequence, either way, gives (TestListWorkspaceStates), so a
// use case that sorted would answer another.
func newListWorkspaceStates() (*app.ListWorkspaceStates, *fakeDirectory, *fakeWorkspaceStates, *fakeAuthorizer) {
	log := &callLog{}
	workspaces := &fakeDirectory{log: log, workspaces: map[string]app.Workspace{"acme": acme}}
	states := &fakeWorkspaceStates{log: log, list: []domain.State{{ID: webReview, ProjectID: webID, Name: "Review", Sequence: 40000},
		{ID: opsBacklog, ProjectID: opsID, Name: "Backlog", Sequence: 15000}, {ID: webTodo, ProjectID: webID, Name: "Todo", Sequence: 25000}}}
	auth := &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{{alice, acme.ID}: {WorkspaceRole: shared.RoleMember},
		{bob, acme.ID}: {WorkspaceRole: shared.RoleAdmin}, {carol, acme.ID}: {WorkspaceRole: shared.RoleGuest}}}
	return app.NewListWorkspaceStates(workspaces, states, auth), workspaces, states, auth
}

// workspaceStatesListed are the calls of user's list of acme's states: the
// workspace, without a lock, the decision in it, the list of the states of
// the projects he is a member of; none in a transaction.
func workspaceStatesListed(user uuid.UUID) []string {
	return []string{"WorkspaceBySlug acme outside tx",
		fmt.Sprintf("Authorize %s %s on %s/%s outside tx", user, domain.ActionWorkspaceStateList, acme.ID, uuid.UUID{}),
		fmt.Sprintf("ListWorkspaceStates %s for %s outside tx", acme.ID, user)}
}

// The use case finds the workspace, decides workspace_state.list in it, then
// answers the store's list for the caller as it is, whatever his role.
func TestListWorkspaceStates(t *testing.T) {
	_, _, fake, _ := newListWorkspaceStates()
	for _, by := range []struct {
		name  string
		order func(a, b domain.State) int
	}{
		{"the project's", func(a, b domain.State) int { return slices.Compare(a.ProjectID[:], b.ProjectID[:]) }},
		{"the project's, reversed", func(a, b domain.State) int { return slices.Compare(b.ProjectID[:], a.ProjectID[:]) }},
		{"the sequence's", func(a, b domain.State) int { return cmp.Compare(a.Sequence, b.Sequence) }},
		{"the sequence's, reversed", func(a, b domain.State) int { return cmp.Compare(b.Sequence, a.Sequence) }},
	} {
		if slices.IsSortedFunc(fake.list, by.order) {
			t.Fatalf("the store's list %v is in %s order: a use case that sorted so would pass", fake.list, by.name)
		}
	}
	for _, user := range []uuid.UUID{alice, bob, carol} {
		uc, _, states, _ := newListWorkspaceStates()
		got, err := uc.Execute(as(user), "acme")
		if want := workspaceStatesListed(user); err != nil || !reflect.DeepEqual(got, states.list) || !slices.Equal(states.log.calls, want) {
			t.Errorf("Execute() as %s = %+v, %v, calls %q; want %+v, calls %q", user, got, err, states.log.calls, states.list, want)
		}
	}
}

// A workspace not there, or not visible, is workspace.not_found and lists
// nothing; every failure comes back as itself, after the calls before it
// and none after; without a caller nothing is read.
func TestListWorkspaceStatesRefuses(t *testing.T) {
	for _, tt := range []struct {
		name                     string
		ctx                      context.Context
		slug                     string
		dirErr, authErr, listErr error
		want                     error
		calls                    []string
	}{
		{"no caller", context.Background(), "acme", nil, nil, nil, shared.Unauthenticated(), nil},
		{"no workspace", as(alice), "gone", nil, nil, nil, domain.ErrWorkspaceNotFound, []string{"WorkspaceBySlug gone outside tx"}},
		{"a workspace he is not in", as(dave), "acme", nil, nil, nil, domain.ErrWorkspaceNotFound, workspaceStatesListed(dave)[:2]},
		{"the directory failing", as(alice), "acme", errDisk, nil, nil, errDisk, workspaceStatesListed(alice)[:1]},
		{"the decision failing", as(alice), "acme", nil, errDisk, nil, errDisk, workspaceStatesListed(alice)[:2]},
		{"the list failing", as(alice), "acme", nil, nil, errDisk, errDisk, workspaceStatesListed(alice)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, workspaces, states, auth := newListWorkspaceStates()
			workspaces.err, states.err = tt.dirErr, tt.listErr
			if tt.authErr != nil {
				auth.errs = map[grantKey]error{{alice, acme.ID}: tt.authErr}
			}
			got, err := uc.Execute(tt.ctx, tt.slug)
			if !refusedAs(err, tt.want) || got != nil || !slices.Equal(states.log.calls, tt.calls) {
				t.Errorf("Execute() = %+v, %v, calls %q; want %v, calls %q", got, err, states.log.calls, tt.want, tt.calls)
			}
		})
	}
}
````

- [ ] **Step 3: HTTP、接线**

`server/internal/modules/project/adapter/http/handler.go`（修改，3 处）：

````old server/internal/modules/project/adapter/http/handler.go
}

// CreateStateUseCase is app.CreateState.
````
````new server/internal/modules/project/adapter/http/handler.go
}

// ListStatesUseCase is app.ListStates.
type ListStatesUseCase interface {
	Execute(ctx context.Context, projectID uuid.UUID) ([]domain.State, error)
}

// CreateStateUseCase is app.CreateState.
````

````old server/internal/modules/project/adapter/http/handler.go
}

// UseCases are the use cases behind the module's operations.
````
````new server/internal/modules/project/adapter/http/handler.go
}

// ListWorkspaceStatesUseCase is app.ListWorkspaceStates.
type ListWorkspaceStatesUseCase interface {
	Execute(ctx context.Context, slug string) ([]domain.State, error)
}

// UseCases are the use cases behind the module's operations.
````

````old server/internal/modules/project/adapter/http/handler.go
	ListProjects      ListProjectsUseCase
	CreateProject     CreateProjectUseCase
	GetProject        GetProjectUseCase
	CheckIdentifier   CheckIdentifierUseCase
	UpdateProject     UpdateProjectUseCase
	ArchiveProject    ArchiveProjectUseCase
	UnarchiveProject  ArchiveProjectUseCase
	DeleteProject     DeleteProjectUseCase
	GetPreferences    GetPreferencesUseCase
	UpdatePreferences UpdatePreferencesUseCase
	ListMembers       ListMembersUseCase
	AddMembers        AddMembersUseCase
	JoinProject       JoinProjectUseCase
	UpdateMember      UpdateMemberUseCase
	RemoveMember      RemoveMemberUseCase
	LeaveProject      LeaveProjectUseCase
	CreateState       CreateStateUseCase
	UpdateState       UpdateStateUseCase
	DeleteState       DeleteStateUseCase
	MarkDefaultState  MarkDefaultStateUseCase
````
````new server/internal/modules/project/adapter/http/handler.go
	ListProjects        ListProjectsUseCase
	CreateProject       CreateProjectUseCase
	GetProject          GetProjectUseCase
	CheckIdentifier     CheckIdentifierUseCase
	UpdateProject       UpdateProjectUseCase
	ArchiveProject      ArchiveProjectUseCase
	UnarchiveProject    ArchiveProjectUseCase
	DeleteProject       DeleteProjectUseCase
	GetPreferences      GetPreferencesUseCase
	UpdatePreferences   UpdatePreferencesUseCase
	ListMembers         ListMembersUseCase
	AddMembers          AddMembersUseCase
	JoinProject         JoinProjectUseCase
	UpdateMember        UpdateMemberUseCase
	RemoveMember        RemoveMemberUseCase
	LeaveProject        LeaveProjectUseCase
	ListStates          ListStatesUseCase
	CreateState         CreateStateUseCase
	UpdateState         UpdateStateUseCase
	DeleteState         DeleteStateUseCase
	MarkDefaultState    MarkDefaultStateUseCase
	ListWorkspaceStates ListWorkspaceStatesUseCase
````

`server/internal/modules/project/adapter/http/states.go`（修改，2 处）：

````old server/internal/modules/project/adapter/http/states.go
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)
````
````new server/internal/modules/project/adapter/http/states.go
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// ListStates serves GET /api/v0/projects/{project_id}/states.
func (h handler) ListStates(ctx context.Context, req gen.ListStatesRequestObject) (gen.ListStatesResponseObject, error) {
	list, err := h.uc.ListStates.Execute(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}
	return gen.ListStates200JSONResponse(states(list)), nil
}
````

````old server/internal/modules/project/adapter/http/states.go
}

// state is s as the API shows it.
````
````new server/internal/modules/project/adapter/http/states.go
}

// ListWorkspaceStates serves GET /api/v0/workspaces/{slug}/states.
func (h handler) ListWorkspaceStates(ctx context.Context, req gen.ListWorkspaceStatesRequestObject) (gen.ListWorkspaceStatesResponseObject, error) {
	list, err := h.uc.ListWorkspaceStates.Execute(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	return gen.ListWorkspaceStates200JSONResponse(states(list)), nil
}

// states is list as the API shows it: data an array, never null.
func states(list []domain.State) gen.StateList {
	out := gen.StateList{Data: make([]gen.State, len(list))}
	for i, s := range list {
		out.Data[i] = state(s)
	}
	return out
}

// state is s as the API shows it.
````

`server/internal/modules/project/adapter/http/handler_test.go`（修改，5 处）：

````old server/internal/modules/project/adapter/http/handler_test.go
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
	removeMember *fakeDelete
	leave        *fakeDelete
	createState  *fakeCreateState
	updateState  *fakeUpdateState
	deleteState  *fakeDelete
	markDefault  *fakeDelete
````
````new server/internal/modules/project/adapter/http/handler_test.go
	list                *fakeList
	create              *fakeCreate
	get                 *fakeGet
	check               *fakeCheck
	update              *fakeUpdate
	archive             *fakeOnProject
	unarchive           *fakeOnProject
	delete              *fakeDelete
	prefs               *fakePreferences
	members             *fakeMembers
	add                 *fakeAdd
	join                *fakeOnProject
	updateMember        *fakeUpdateMember
	removeMember        *fakeDelete
	leave               *fakeDelete
	listStates          *fakeListStates
	createState         *fakeCreateState
	updateState         *fakeUpdateState
	deleteState         *fakeDelete
	markDefault         *fakeDelete
	listWorkspaceStates *fakeListWorkspaceStates
````

````old server/internal/modules/project/adapter/http/handler_test.go
		f.leave = &fakeDelete{}
	}
````
````new server/internal/modules/project/adapter/http/handler_test.go
		f.leave = &fakeDelete{}
	}
	if f.listStates == nil {
		f.listStates = &fakeListStates{}
	}
````

````old server/internal/modules/project/adapter/http/handler_test.go
		f.markDefault = &fakeDelete{}
	}
````
````new server/internal/modules/project/adapter/http/handler_test.go
		f.markDefault = &fakeDelete{}
	}
	if f.listWorkspaceStates == nil {
		f.listWorkspaceStates = &fakeListWorkspaceStates{}
	}
````

````old server/internal/modules/project/adapter/http/handler_test.go
		JoinProject: f.join, UpdateMember: f.updateMember, RemoveMember: f.removeMember, LeaveProject: f.leave,
````
````new server/internal/modules/project/adapter/http/handler_test.go
		JoinProject: f.join, UpdateMember: f.updateMember, RemoveMember: f.removeMember, LeaveProject: f.leave, ListStates: f.listStates,
````

````old server/internal/modules/project/adapter/http/handler_test.go
		MarkDefaultState: f.markDefault})
````
````new server/internal/modules/project/adapter/http/handler_test.go
		MarkDefaultState: f.markDefault, ListWorkspaceStates: f.listWorkspaceStates})
````

`server/internal/modules/project/adapter/http/states_test.go`（修改，5 处）：

````old server/internal/modules/project/adapter/http/states_test.go
	"github.com/open-nerve/NerveProject/server/internal/shared"
)
````
````new server/internal/modules/project/adapter/http/states_test.go
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeListStates is listStates: each call is recorded as "caller id"; it
// answers list, or err.
type fakeListStates struct {
	calls []string
	list  []domain.State
	err   error
}

func (f *fakeListStates) Execute(ctx context.Context, projectID uuid.UUID) ([]domain.State, error) {
	f.calls = append(f.calls, caller(ctx)+" "+projectID.String())
	return f.list, f.err
}
````

````old server/internal/modules/project/adapter/http/states_test.go
var (
````
````new server/internal/modules/project/adapter/http/states_test.go
var (
	webBacklog = domain.State{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000c1"), WorkspaceID: acmeID, ProjectID: webID, Name: "Backlog",
		Color: "#60646C", Group: domain.GroupBacklog, Default: true, Sequence: 15000, CreatedAt: created, UpdatedAt: created}
````

````old server/internal/modules/project/adapter/http/states_test.go
		UpdatedAt: created.Add(1)}
````
````new server/internal/modules/project/adapter/http/states_test.go
		UpdatedAt: created.Add(1)}
	webBacklogJSON = `{"color":"#60646C","created_at":"2026-10-01T10:00:00.123456Z","default":true,"description":"","group":"backlog",` +
		`"id":"0199a2b4-0000-7000-8000-0000000000c1","name":"Backlog","project_id":"0199a2b4-0000-7000-8000-0000000000a1","sequence":15000,` +
		`"updated_at":"2026-10-01T10:00:00.123456Z","workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
````

````old server/internal/modules/project/adapter/http/states_test.go
	projectNotFoundJSON = `{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`
)
````
````new server/internal/modules/project/adapter/http/states_test.go
	projectNotFoundJSON = `{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`
)

// GET goes to the use case for the caller and the path's project; the
// answer is 200 with its list, in its order, every field of each state,
// and [] for none. Its refusals and its failure, as the contract declares
// them.
func TestListStates(t *testing.T) {
	path := "/api/v0/projects/" + webID.String() + "/states"
	for _, tt := range []struct {
		list []domain.State
		want string
	}{{[]domain.State{webReview, webBacklog}, `{"data":[` + webReviewJSON + `,` + webBacklogJSON + `]}`}, {nil, `{"data":[]}`}} {
		list := &fakeListStates{list: tt.list}
		h := newServer(t, fakes{listStates: list})
		if res, body := do(t, h, request(http.MethodGet, path, "bob", "")); res.StatusCode != http.StatusOK || body != tt.want+"\n" {
			t.Errorf("GET = %d %s, want 200 %s", res.StatusCode, body, tt.want)
		}
		if want := []string{"bob " + webID.String()}; !slices.Equal(list.calls, want) {
			t.Errorf("calls = %q, want %q", list.calls, want)
		}
	}
	for _, tt := range []struct {
		err    error
		status int
		want   string
	}{
		{domain.ErrNotFound, http.StatusNotFound, projectNotFoundJSON},
		{shared.Forbidden(), http.StatusForbidden, forbiddenJSON},
		{errGone, http.StatusInternalServerError, internalErrorJSON},
	} {
		h := newServer(t, fakes{listStates: &fakeListStates{err: tt.err}})
		if res, body := do(t, h, request(http.MethodGet, path, "alice", "")); res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("GET = %d %s, want %d %s", res.StatusCode, body, tt.status, tt.want)
		}
	}
}
````

````old server/internal/modules/project/adapter/http/states_test.go
			t.Errorf("%s: POST = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

````
````new server/internal/modules/project/adapter/http/states_test.go
			t.Errorf("%s: POST = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

// fakeListWorkspaceStates is listWorkspaceStates: each call is recorded
// as "caller slug"; it answers list, or err.
type fakeListWorkspaceStates struct {
	calls []string
	list  []domain.State
	err   error
}

func (f *fakeListWorkspaceStates) Execute(ctx context.Context, slug string) ([]domain.State, error) {
	f.calls = append(f.calls, caller(ctx)+" "+slug)
	return f.list, f.err
}

// GET goes to the use case for the caller and the path's workspace; the
// answer is 200 with its list, in its order, every field of each state,
// and [] for none. Its refusal and its failure, as the contract declares
// them.
func TestListWorkspaceStates(t *testing.T) {
	for _, tt := range []struct {
		list []domain.State
		want string
	}{{[]domain.State{webReview, webBacklog}, `{"data":[` + webReviewJSON + `,` + webBacklogJSON + `]}`}, {nil, `{"data":[]}`}} {
		list := &fakeListWorkspaceStates{list: tt.list}
		h := newServer(t, fakes{listWorkspaceStates: list})
		if res, body := do(t, h, request(http.MethodGet, "/api/v0/workspaces/acme/states", "bob", "")); res.StatusCode != http.StatusOK ||
			body != tt.want+"\n" {
			t.Errorf("GET = %d %s, want 200 %s", res.StatusCode, body, tt.want)
		}
		if want := []string{"bob acme"}; !slices.Equal(list.calls, want) {
			t.Errorf("calls = %q, want %q", list.calls, want)
		}
	}
	for _, tt := range []struct {
		err    error
		status int
		want   string
	}{
		{domain.ErrWorkspaceNotFound, http.StatusNotFound,
			`{"status":404,"code":"workspace.not_found","title":"Not Found","detail":"The workspace does not exist, or you are not a member of it."}`},
		{errGone, http.StatusInternalServerError, internalErrorJSON},
	} {
		h := newServer(t, fakes{listWorkspaceStates: &fakeListWorkspaceStates{err: tt.err}})
		if res, body := do(t, h, request(http.MethodGet, "/api/v0/workspaces/acme/states", "bob", "")); res.StatusCode != tt.status ||
			body != tt.want+"\n" {
			t.Errorf("GET answering %v = %d %s, want %d %s", tt.err, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

````

`server/internal/modules/project/module.go`（修改，2 处）：

````old server/internal/modules/project/module.go
// member's display settings, creating, changing and deleting a project's
// states and making one its default, carries out the workspace module's
// cascades on the projects (ProjectCascade), and offers the access module
// its reads of a project (ProjectAccess) and the workspace module its
// count of an account's ended project memberships
````
````new server/internal/modules/project/module.go
// member's display settings, listing, creating, changing and deleting a
// project's states and making one its default, listing the states of a
// workspace's projects one is a member of, carries out the workspace
// module's cascades on the projects (ProjectCascade), and offers the
// access module its reads of a project (ProjectAccess) and the workspace
// module its count of an account's ended project memberships
````

````old server/internal/modules/project/module.go
		ListProjects:      app.NewListProjects(d.Workspaces, store, d.Authorizer),
		GetProject:        app.NewGetProject(store, d.Authorizer),
		CheckIdentifier:   app.NewCheckProjectIdentifier(d.Workspaces, store, d.Authorizer),
		UpdateProject:     app.NewUpdateProject(store, locks, d.Tx, d.Clock),
		ArchiveProject:    app.NewArchiveProject(store, locks, d.Tx, d.Clock),
		UnarchiveProject:  app.NewUnarchiveProject(store, locks, d.Tx, d.Clock),
		DeleteProject:     app.NewDeleteProject(store, locks, d.Tx, d.Clock),
		GetPreferences:    app.NewGetProjectPreferences(store, d.Authorizer),
		UpdatePreferences: app.NewUpdateProjectPreferences(store, locks, d.Tx, d.Clock),
		ListMembers:       app.NewListProjectMembers(store, d.Authorizer),
		AddMembers:        app.NewAddProjectMembers(app.AddMembersDeps{Locks: locks, Projects: store, Tx: d.Tx, Clock: d.Clock}),
		JoinProject:       app.NewJoinProject(locks, store, d.Tx, d.Clock),
		UpdateMember:      app.NewUpdateProjectMember(locks, store, d.Tx, d.Clock),
		RemoveMember:      app.NewRemoveProjectMember(locks, store, d.Tx, d.Clock),
		LeaveProject:      app.NewLeaveProject(locks, store, d.Tx, d.Clock),
		CreateState:       app.NewCreateState(locks, store, d.Tx, d.Clock),
		UpdateState:       app.NewUpdateState(locks, store, d.Tx, d.Clock),
		DeleteState:       app.NewDeleteState(locks, store, d.Tx, d.Clock),
		MarkDefaultState:  app.NewMarkDefaultState(locks, store, d.Tx, d.Clock),
````
````new server/internal/modules/project/module.go
		ListProjects:        app.NewListProjects(d.Workspaces, store, d.Authorizer),
		GetProject:          app.NewGetProject(store, d.Authorizer),
		CheckIdentifier:     app.NewCheckProjectIdentifier(d.Workspaces, store, d.Authorizer),
		UpdateProject:       app.NewUpdateProject(store, locks, d.Tx, d.Clock),
		ArchiveProject:      app.NewArchiveProject(store, locks, d.Tx, d.Clock),
		UnarchiveProject:    app.NewUnarchiveProject(store, locks, d.Tx, d.Clock),
		DeleteProject:       app.NewDeleteProject(store, locks, d.Tx, d.Clock),
		GetPreferences:      app.NewGetProjectPreferences(store, d.Authorizer),
		UpdatePreferences:   app.NewUpdateProjectPreferences(store, locks, d.Tx, d.Clock),
		ListMembers:         app.NewListProjectMembers(store, d.Authorizer),
		AddMembers:          app.NewAddProjectMembers(app.AddMembersDeps{Locks: locks, Projects: store, Tx: d.Tx, Clock: d.Clock}),
		JoinProject:         app.NewJoinProject(locks, store, d.Tx, d.Clock),
		UpdateMember:        app.NewUpdateProjectMember(locks, store, d.Tx, d.Clock),
		RemoveMember:        app.NewRemoveProjectMember(locks, store, d.Tx, d.Clock),
		LeaveProject:        app.NewLeaveProject(locks, store, d.Tx, d.Clock),
		ListStates:          app.NewListStates(store, d.Authorizer),
		CreateState:         app.NewCreateState(locks, store, d.Tx, d.Clock),
		UpdateState:         app.NewUpdateState(locks, store, d.Tx, d.Clock),
		DeleteState:         app.NewDeleteState(locks, store, d.Tx, d.Clock),
		MarkDefaultState:    app.NewMarkDefaultState(locks, store, d.Tx, d.Clock),
		ListWorkspaceStates: app.NewListWorkspaceStates(d.Workspaces, store, d.Authorizer),
````

- [ ] **Step 4: 矩阵和组合测试**

`server/internal/bootstrap/permission_matrix_states_test.go`（修改，6 处）：

````old server/internal/bootstrap/permission_matrix_states_test.go
	"net/http"
````
````new server/internal/bootstrap/permission_matrix_states_test.go
	"net/http"
	"slices"
````

````old server/internal/bootstrap/permission_matrix_states_test.go
// creating them, under the project of each column, and the writes on one,
// naming it by its id (/states/{state_id}) among its column's project's
// seeded states, those of matrixStates.
````
````new server/internal/bootstrap/permission_matrix_states_test.go
// listing and creating them, under the project of each column, the writes
// on one, naming it by its id (/states/{state_id}) among its column's
// project's seeded states, those of matrixStates, and listing a
// workspace's, under the workspace of each column.
````

````old server/internal/bootstrap/permission_matrix_states_test.go
const newState = `{"name":"QA","color":"#0EA5E9","group":"completed"}`

````
````new server/internal/bootstrap/permission_matrix_states_test.go
const newState = `{"name":"QA","color":"#0EA5E9","group":"completed"}`

// listedStates are the names of each matrix project's states as a list
// answers them: matrixStates by sequence, the triage state left out.
var listedStates = []string{"Backlog", "Todo", "In Progress", "Review", "Done", "Cancelled"}

````

````old server/internal/bootstrap/permission_matrix_states_test.go
	return []matrixRow{
````
````new server/internal/bootstrap/permission_matrix_states_test.go
	return []matrixRow{
		// Every active member of the project (M3 design 9.2): PM+WA as its
		// member, and not WA-, who is none.
		{op: "listStates", columns: projectColumns, request: toProject(http.MethodGet, "/states", ""),
			cells: ofProject(cellOK, cellOK, cellOK, cellOK, cellForbidden, cellForbidden), check: listsTheStates},
		// An archived project lists none (M3 design 3.17), to who may list.
		{op: "listStates", variant: "archived", columns: archivedColumns, request: toProject(http.MethodGet, "/states", ""),
			cells: ofArchived(cellOK, cellForbidden), check: listsTheStates},
````

````old server/internal/bootstrap/permission_matrix_states_test.go
			request: toState(http.MethodPost, "/mark-default", "Todo", ""), cells: ofArchivedState(cellNoContent, cellForbidden)},
````
````new server/internal/bootstrap/permission_matrix_states_test.go
			request: toState(http.MethodPost, "/mark-default", "Todo", ""), cells: ofArchivedState(cellNoContent, cellForbidden)},
		// Every active member of the workspace (M3 design 9.2), each the
		// states of the projects he is a member of.
		{op: "listWorkspaceStates", request: toWorkspace(http.MethodGet, "/states", ""), cells: inWorkspace(cellOK, cellOK, cellOK),
			check: listsTheWorkspaceStates},
	}
}

// listsTheStates: the column's project's states but its triage state, by
// sequence: Review after In Progress; none of the archived project's.
func listsTheStates(t *testing.T, c caller, s seeded, answer string) {
	var list struct {
		Data []struct {
			ID        uuid.UUID `json:"id"`
			ProjectID uuid.UUID `json:"project_id"`
			Name      string    `json:"name"`
		} `json:"data"`
	}
	decodeAnswer(t, answer, &list)
	var got []string
	for _, st := range list.Data {
		if st.ProjectID != s.project(projectOf(c)) || st.ID != s.state(projectOf(c), st.Name) {
			t.Errorf("%s lists %s, not its project's", c, answer)
		}
		got = append(got, st.Name)
	}
	want := listedStates
	if projectOf(c) == "acme/archived" {
		want = nil
	}
	if !slices.Equal(got, want) {
		t.Errorf("%s lists %q, want %q", c, got, want)
````

````old server/internal/bootstrap/permission_matrix_states_test.go
		t.Errorf("%s renames %s; want %s's Todo, unstarted, named Next", c, answer, projectOf(c))
	}
}

````
````new server/internal/bootstrap/permission_matrix_states_test.go
		t.Errorf("%s renames %s; want %s's Todo, unstarted, named Next", c, answer, projectOf(c))
	}
}

// listsTheWorkspaceStates: the states of acme's unarchived projects that
// the column's account is an active member of, by project, then sequence:
// none for the admin and the member, members of none of them, and the
// public and the private project's for the guest, PG's account.
func listsTheWorkspaceStates(t *testing.T, c caller, s seeded, answer string) {
	var list struct {
		Data []struct {
			ID uuid.UUID `json:"id"`
		} `json:"data"`
	}
	decodeAnswer(t, answer, &list)
	var got, want []uuid.UUID
	for _, st := range list.Data {
		got = append(got, st.ID)
	}
	if c == callerGuest {
		for _, key := range byProjectID(s, "acme/public", "acme/private") {
			for _, name := range listedStates {
				want = append(want, s.state(key, name))
			}
		}
	}
	if !slices.Equal(got, want) {
		t.Errorf("%s lists %v, want %v", c, got, want)
	}
}

// byProjectID are the projects keys names, in the order of their ids.
func byProjectID(s seeded, keys ...string) []string {
	return slices.SortedFunc(slices.Values(keys), func(a, b string) int {
		pa, pb := s.project(a), s.project(b)
		return slices.Compare(pa[:], pb[:])
	})
}

````

`server/internal/bootstrap/project_visibility_test.go`（修改，1 处）：

````old server/internal/bootstrap/project_visibility_test.go
	}
}

````
````new server/internal/bootstrap/project_visibility_test.go
	}
}

// The workspace's list of states and each project's agree (M3 design 3.4,
// 9.3): for every account of the matrix, the states listWorkspaceStates
// lists of acme are exactly those listStates lists of each of acme's
// projects, by project in the order of their ids: the projects whose states
// state.list lets him list, partingStates' memberships among them, the
// archived one's none. The workspace's list reads the membership in its
// query, each project's list in access's rule: here the two cannot part.
// Some accounts list no state, some both unarchived projects'.
func TestListingWorkspaceStatesIsListingEachProjects(t *testing.T) {
	d := prepareMatrix(t)
	contract := apitest.Load(t)
	base := startApp(t, d.config(t, pgtest.NewDatabaseFrom(t, d.url), nil), migrations.FS())
	s := d.seeded.in(t)
	var sizes []int
	for _, account := range matrixAccounts {
		token := d.tokens[account]
		listed := stateIDs(t, contract, base+"/api/v0/workspaces/acme/states", token)
		var each []uuid.UUID
		for _, key := range byProjectID(s, "acme/public", "acme/private", "acme/archived") {
			each = append(each, stateIDs(t, contract, base+"/api/v0/projects/"+s.project(key).String()+"/states", token)...)
		}
		if !slices.Equal(listed, each) {
			t.Errorf("%s: acme's list %v, each project's %v", account, listed, each)
		}
		sizes = append(sizes, len(each))
	}
	if slices.Min(sizes) != 0 || slices.Max(sizes) != 2*len(listedStates) {
		t.Errorf("the accounts list %v states; want some none and some both unarchived projects' %d", sizes, 2*len(listedStates))
	}
}

// Each workspace's list of states holds its own projects' alone (M3 design
// 3.17, 9.3), on memberWorld, whose bob and carol are members of acme's Web
// and Ops and of beta's Lab: for each of its accounts, acme's list is
// exactly what listStates lists of Web and of Ops, by project id, and
// beta's what it lists of Lab. A list that took in the projects of another
// workspace would hold Lab's in acme's, Web's and Ops's in beta's; Web has
// a deleted state, Dropped, which bob created and deleted, and which
// neither list holds. The matrix's accounts are members of one live
// workspace's projects alone, and no live project of it has a deleted
// state.
func TestListingWorkspaceStatesKeepsToItsWorkspace(t *testing.T) {
	w := newMemberWorld(t)
	status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/projects/"+w.web.String()+"/states", w.tokens["bob"],
		`{"name":"Dropped","color":"#000000","group":"backlog"}`)
	if status != http.StatusCreated {
		t.Fatalf("bob's creating Dropped = %d %s", status, body)
	}
	var dropped struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, body, &dropped)
	if status, body := call(t, w.contract, http.MethodDelete, w.base+"/api/v0/states/"+dropped.ID.String(), w.tokens["bob"],
		""); status != http.StatusNoContent {
		t.Fatalf("bob's deleting Dropped = %d %s", status, body)
	}
	acme := slices.SortedFunc(slices.Values([]uuid.UUID{w.web, w.ops}), func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) })
	for _, p := range []uuid.UUID{w.web, w.ops, w.lab} {
		if len(stateIDs(t, w.contract, w.base+"/api/v0/projects/"+p.String()+"/states", w.tokens["bob"])) == 0 {
			t.Fatalf("bob lists no state of %s; the test needs him to list each project's", p)
		}
	}
	for _, name := range []string{"alice", "bob", "carol", "dave", "erin", "gina"} {
		for _, ws := range []struct {
			slug     string
			projects []uuid.UUID
		}{{"acme", acme}, {"beta", []uuid.UUID{w.lab}}} {
			listed := stateIDs(t, w.contract, w.base+"/api/v0/workspaces/"+ws.slug+"/states", w.tokens[name])
			var each []uuid.UUID
			for _, p := range ws.projects {
				each = append(each, stateIDs(t, w.contract, w.base+"/api/v0/projects/"+p.String()+"/states", w.tokens[name])...)
			}
			if !slices.Equal(listed, each) {
				t.Errorf("%s: %s's list %v, each project's %v", name, ws.slug, listed, each)
			}
		}
	}
}

// stateIDs are the ids of the states GET url answers token, in its order;
// none when it answers other than 200.
func stateIDs(t *testing.T, contract *apitest.Contract, url, token string) []uuid.UUID {
	t.Helper()
	status, body := call(t, contract, http.MethodGet, url, token, "")
	if status != http.StatusOK {
		return nil
	}
	var list struct {
		Data []struct {
			ID uuid.UUID `json:"id"`
		} `json:"data"`
	}
	decodeAnswer(t, body, &list)
	ids := make([]uuid.UUID, len(list.Data))
	for i, st := range list.Data {
		ids[i] = st.ID
	}
	return ids
}

````

- [ ] **Step 5: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/project/... ./internal/modules/access/...`
Expected: 全部 `ok`（`apitest.Main` 两个方向：两个读的码；`listProjects`、`checkProjectIdentifier` 的测试照旧通过）。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix$|TestListingWorkspaceStatesIsListingEachProjects$|TestListingWorkspaceStatesKeepsToItsWorkspace$' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过（关键词守卫 5 个命中都有例外）。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

- [ ] **Step 6: 提交**

```bash
git add api/modules/project.yaml api/openapi.yaml server/internal/bootstrap/permission_matrix_states_test.go server/internal/bootstrap/project_visibility_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/project/adapter/http/handler.go server/internal/modules/project/adapter/http/handler_test.go server/internal/modules/project/adapter/http/states.go server/internal/modules/project/adapter/http/states_test.go server/internal/modules/project/app/check_identifier.go server/internal/modules/project/app/fakes_state_test.go server/internal/modules/project/app/list_projects.go server/internal/modules/project/app/list_states.go server/internal/modules/project/app/list_states_test.go server/internal/modules/project/app/list_workspace_states.go server/internal/modules/project/app/list_workspace_states_test.go server/internal/modules/project/app/lock.go server/internal/modules/project/app/state_ports.go server/internal/modules/project/domain/actions.go server/internal/modules/project/module.go api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P7a): listStates and listWorkspaceStates

A project's active members list its states but the triage state, by
sequence, then id, none while it is archived; a workspace's active
members list the states of its unarchived projects they are active
members of, by project, which no visibility widens. The workspace's
two first steps of a read, its lookup by slug and the decision in it,
are one function, which listProjects and checkProjectIdentifier take
too. The matrix holds the reads' cells, and a test holds the
workspace's list to each project's.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p7a.py` 的编号；"层"是它被发现的每一层，端到端是单独运行的故事）：

| 变异 | 改坏 | 必须失败的测试 | 层 |
|---|---|---|---|
| `s2-list` | `listStates` 把失败答成空列表 | `TestListStatesRefuses` | 单元（按性质只在单元一层：真实的存储在这一步不失败） |
| `s2-lws` | `listWorkspaceStates` 把失败答成空列表 | `TestListWorkspaceStatesRefuses` | 单元（按性质只在单元一层：真实的存储在这一步不失败） |
| `s5-list-no-guests` | 规则表：`state.list` 不给项目访客 | `TestEveryRuleDecidesItsCells`、`TestListingWorkspaceStatesIsListingEachProjects`、`TestPermissionMatrix` | 单元；组合 |
| `s5-lws-no-guests` | 规则表：`workspace_state.list` 不给工作区访客 | `TestEveryRuleDecidesItsCells`、`TestListingWorkspaceStatesIsListingEachProjects`、`TestPermissionMatrix` | 单元；组合 |
| `s5-list-undecided` | `listStates` 不判定就列出 | `TestListStates`、`TestListStatesRefuses`、`TestListingWorkspaceStatesIsListingEachProjects`、`TestPermissionMatrix` | 单元；组合 |
| `s5-lws-undecided` | `listWorkspaceStates` 不判定就列出 | `TestListWorkspaceStates`、`TestListWorkspaceStatesRefuses`、`TestPermissionMatrix` | 单元；组合 |
| `s18-lws-no-not-found` | `listWorkspaceStates` 不声明 `workspace.not_found` | `TestListWorkspaceStates`、`TestListingWorkspaceStatesIsListingEachProjects`、`TestListingWorkspaceStatesKeepsToItsWorkspace`、`TestPermissionMatrix` | 单元；组合 |

---

### Task 9: 并发：按资源寻址的竞争和锁强度推广到状态的行；交错 10 和一组最后两个状态；每个状态的写只写它的行

**Files:**
- Create: `server/internal/bootstrap/interleaving_states_test.go`、`server/internal/bootstrap/project_row_races_test.go`、`server/internal/bootstrap/state_rows_test.go`
- Modify: `server/internal/bootstrap/interleaving_leaving_test.go`、`server/internal/bootstrap/membership_world_test.go`
- Delete: `server/internal/bootstrap/project_membership_races_test.go`

**Interfaces:**
- 只有测试（spec 2.10；M3 设计 3.6 约定二、三，3.17，9.3 交错 10）：
  - `project_membership_races_test.go` 改名为 `project_row_races_test.go`，`membershipWrite` 推广为 `rowWrite`（`member`、`state`、`clears`、`creates`；`row` 回答它改的表和行；`param`）：P5b 的两个测试各加状态的四个写：`updateState`、`deleteState`、`markDefaultState`（bob 改 Web 的 QA），和按项目寻址的 `createState`（bob 在 Web 建 Checked，同离开；裁定 M1）。竞争的六行：这一行结束（只有成员关系有）、删除、移到 Ops，调用者的成员关系结束，Web 删除，acme 删除，写答 404、什么都不改（离开和建状态只跑后三行）；锁强度：等 acme 时什么都没持有，等 Web 时持有 acme 的 `FOR SHARE`，等这一行时持有 Web 的 `FOR NO KEY UPDATE`，只有设为默认持有 Web 的默认 Backlog（`FOR NO KEY UPDATE`，它先写了它）；建状态没有自己的行可等，由 `creates` 跳过，它的锁的顺序和强度由最先锁工作区的测试和交错的两个创建钉住。
  - `memberWorld`（`membership_world_test.go`）：Web 多一个 completed 组的 QA，由 alice 经接口建（`newState`）；alice 在 QA 之后建、随即删除的 Retired（85000）；Ops 的 Cancelled 由 alice 经接口移到 100000，在 Web 的每个状态之后（读了别的项目的、已删除的状态的最大 `sequence` 的创建在交错里看得出，清扫 23、24、26）；`interleaving_leaving_test.go` 的 `projectLocks`、`tx` 由离开和状态的写共用。
  - `interleaving_states_test.go`：两个写在 Web 上串行，两种顺序；第一个持有锁、停在它写之后的门，第二个由探测确认在等 Web 的行（第二个的锁的顺序是最先锁工作区的测试的）。
  - `state_rows_test.go`：每个状态的写只写它的行（`tableRows` 比较其余每张表每一行）。

**Tests:**
- `TestStateWritesOnOneProjectSerialize`（新）：completed 组的 Done 和 QA 各删除或改走，第二个 409 `project.state_last_in_group`；QA 先设为默认再删除，删除 409 `project.state_default`；先删除再设为默认，404 `project.state_not_found`（交错 10，Web 两种顺序都恰好一个默认）；两个创建，第二个在第一个之后的 `sequence`，时刻不早于门打开的时刻（它在锁下读时钟，3.3）。之后 Web 的状态是第一个留下的样子（第二个成功时加上它的），Ops、beta 的 Lab 不变。
- `TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`（改名、推广；P5b 的两个测试删除）：前者加 `createState` 的三行（调用者的成员关系结束、Web 删除、acme 删除），各 404 `project.not_found`、不建状态。
- `TestEachStateWriteChangesItsRowsAlone`（新，七行：bob 建 Shipped、改 QA 的名称、删除它、把 Backlog（backlog 组唯一的状态）改名并给它自己的组（不是改组，不数）、在分诊组建状态和把 Done 改到分诊组（各 422 `group not_allowed`，在写的事务之前，什么都不写；这两个请求体在契约的枚举之外，测试不核对请求、照旧核对回答）、把 Done 设为默认；每个写之后，它写的行之外每张表每一行不变）。

- [ ] **Step 1: 测试的世界**

`server/internal/bootstrap/membership_world_test.go`（修改，2 处）：

````old server/internal/bootstrap/membership_world_test.go
// and bob is a workspace admin in beta alone.
````
````new server/internal/bootstrap/membership_world_test.go
// and bob is a workspace admin in beta alone. Each project has the states
// it is made with; Web has QA too, of the completed group beside Done,
// alice's (newState), and Retired, deleted, which alice created after QA,
// at 85000; Ops's Cancelled is at 100000, after every state of Web, moved
// there by alice.
````

````old server/internal/bootstrap/membership_world_test.go
			t.Fatalf("%s's adding %s = %d %s", add.by, add.members, status, body)
		}
````
````new server/internal/bootstrap/membership_world_test.go
			t.Fatalf("%s's adding %s = %d %s", add.by, add.members, status, body)
		}
	}
	if status, body := call(t, contract, http.MethodPost, w.base+"/api/v0/projects/"+w.web.String()+"/states", w.tokens["alice"],
		newState); status != http.StatusCreated {
		t.Fatalf("alice's creating Web's QA = %d %s", status, body)
	}
	status, body := call(t, contract, http.MethodPost, w.base+"/api/v0/projects/"+w.web.String()+"/states", w.tokens["alice"],
		`{"name":"Retired","color":"#000000","group":"completed"}`)
	var retired struct {
		ID       uuid.UUID `json:"id"`
		Sequence float64   `json:"sequence"`
	}
	if status != http.StatusCreated {
		t.Fatalf("alice's creating Web's Retired = %d %s", status, body)
	}
	if decodeAnswer(t, body, &retired); retired.Sequence != 85000 {
		t.Fatalf("Web's Retired at %v; want 85000, after QA", retired.Sequence)
	}
	if status, body := call(t, contract, http.MethodDelete, w.base+"/api/v0/states/"+retired.ID.String(), w.tokens["alice"],
		""); status != http.StatusNoContent {
		t.Fatalf("alice's deleting Web's Retired = %d %s", status, body)
	}
	if status, body := call(t, contract, http.MethodPatch, w.base+"/api/v0/states/"+stateID(t, w.pool, w.ops, "Cancelled").String(),
		w.tokens["alice"], `{"sequence":100000}`); status != http.StatusOK {
		t.Fatalf("alice's moving Ops's Cancelled = %d %s", status, body)
````

`server/internal/bootstrap/interleaving_leaving_test.go`（修改，3 处）：

````old server/internal/bootstrap/interleaving_leaving_test.go
}

// leaveProject is name's leaving of project, through project's use case
// over leavers, with Locks over project's store, workspace's directory and
// members' locks and the Authorizer, as project.New wires them.
````
````new server/internal/bootstrap/interleaving_leaving_test.go
}

// projectLocks is project's Locks over its store, workspace's directory
// and members' locks and the Authorizer, as project.New wires them.
func (w memberWorld) projectLocks() projectapp.Locks {
	provided := workspace.Provide(w.pool)
	return projectapp.NewLocks(projectpg.New(w.pool), projectWorkspaces{directory: provided.WorkspaceDirectory}, provided.WorkspaceMembers,
		authorizerOn(w.pool))
}

// tx is a transaction manager on w's pool, as bootstrap gives each module
// one, with a commit timeout of 2s.
func (w memberWorld) tx() shared.TxManager {
	return postgres.NewTxManager(w.pool, 2*time.Second)
}

// leaveProject is name's leaving of project, through project's use case
// over leavers, with projectLocks and tx.
````

````old server/internal/bootstrap/interleaving_leaving_test.go
	provided := workspace.Provide(w.pool)
	locks := projectapp.NewLocks(projectpg.New(w.pool), projectWorkspaces{directory: provided.WorkspaceDirectory}, provided.WorkspaceMembers,
		authorizerOn(w.pool))
	return projectapp.NewLeaveProject(locks, leavers, postgres.NewTxManager(w.pool, 2*time.Second), clock.System{}).
````
````new server/internal/bootstrap/interleaving_leaving_test.go
	return projectapp.NewLeaveProject(w.projectLocks(), leavers, w.tx(), clock.System{}).
````

````old server/internal/bootstrap/interleaving_leaving_test.go
		project.NewCascade(project.CascadeDeps{Pool: w.pool}), authorizerOn(w.pool), postgres.NewTxManager(w.pool, 2*time.Second), clock.System{}).
````
````new server/internal/bootstrap/interleaving_leaving_test.go
		project.NewCascade(project.CascadeDeps{Pool: w.pool}), authorizerOn(w.pool), w.tx(), clock.System{}).
````

- [ ] **Step 2: 竞争和锁强度推广到状态的行**

`server/internal/bootstrap/project_membership_races_test.go`（删除）：

````delete server/internal/bootstrap/project_membership_races_test.go
````

`server/internal/bootstrap/project_row_races_test.go`（新文件，309 行）：

````file server/internal/bootstrap/project_row_races_test.go
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

// A write on a row under a project that waits for a lock, against what
// another transaction changes or holds meanwhile, on the wired app (M3
// design 3.6 convention 2, the lock table): what it finds once it has its
// locks, and each lock it takes, at its strength.

// rowWrite is one of the writes on a row under a project as the races send
// it on memberWorld: bob, Web's admin and acme's member, changes gina's
// role in Web to a guest's, or removes her; dave, Web's admin, leaves it;
// bob renames Web's QA, deletes it, or makes it Web's default; or bob
// creates a state in Web, a write addressed by its project, as leaving is.
// gina is Web's member and acme's admin; alice, Web's other admin, stays;
// QA is of the completed group, beside Done; nothing else refuses each
// write.
type rowWrite struct {
	op, by string
	// member is whose membership of Web the write changes, the caller's for
	// a write on a state or a creation; state, when set, is the name of
	// Web's state the write changes instead.
	member, state      string
	method, path, body string // path: %s the row's id for a path of one (rowPaths), else Web's
	status             int    // its answer, alone
	notFound           string // the code of its 404
	target             bool   // it changes the member's role: it shares his membership of acme (convention 3)
	clears             bool   // it makes its state Web's default: it writes Web's default, Backlog, before its state
	creates            bool   // it creates a state of Web: it has no row of its own to wait on
}

var rowWrites = []rowWrite{
	{op: "updateProjectMember", by: "bob", member: "gina", method: http.MethodPatch, path: "/api/v0/project-members/%s", body: `{"role":5}`,
		status: http.StatusOK, notFound: "project.member_not_found", target: true},
	{op: "removeProjectMember", by: "bob", member: "gina", method: http.MethodDelete, path: "/api/v0/project-members/%s",
		status: http.StatusNoContent, notFound: "project.member_not_found"},
	{op: "leaveProject", by: "dave", member: "dave", method: http.MethodPost, path: "/api/v0/projects/%s/leave", status: http.StatusNoContent,
		notFound: "project.not_found"},
	{op: "updateState", by: "bob", member: "bob", state: "QA", method: http.MethodPatch, path: "/api/v0/states/%s", body: `{"name":"Checked"}`,
		status: http.StatusOK, notFound: "project.state_not_found"},
	{op: "deleteState", by: "bob", member: "bob", state: "QA", method: http.MethodDelete, path: "/api/v0/states/%s",
		status: http.StatusNoContent, notFound: "project.state_not_found"},
	{op: "markDefaultState", by: "bob", member: "bob", state: "QA", method: http.MethodPost, path: "/api/v0/states/%s/mark-default",
		status: http.StatusNoContent, notFound: "project.state_not_found", clears: true},
	{op: "createState", by: "bob", member: "bob", method: http.MethodPost, path: "/api/v0/projects/%s/states",
		body: `{"name":"Checked","color":"#0EA5E9","group":"completed"}`, status: http.StatusCreated, notFound: "project.not_found",
		creates: true},
}

// row is the row of Web that m changes: its table and its id.
func (m rowWrite) row(t *testing.T, w memberWorld) (string, uuid.UUID) {
	t.Helper()
	if m.state != "" {
		return "states", stateID(t, w.pool, w.web, m.state)
	}
	return "project_members", w.membership(t, w.web, m.member)
}

// sent sends m on w's app, checked against the contract, and hands over its
// answer.
func (m rowWrite) sent(t *testing.T, w memberWorld) (*http.Request, <-chan answer) {
	t.Helper()
	named := w.web
	if m.param() != "{project_id}" {
		_, named = m.row(t, w)
	}
	var body []byte
	if m.body != "" {
		body = []byte(m.body)
	}
	req := newRequest(t, m.method, w.base+strings.Replace(m.path, "%s", named.String(), 1), w.tokens[m.by], body)
	w.contract.CheckRequest(t, req)
	return req, sendInBackground(req)
}

// param is the parameter m's path names, as projectWrite.param.
func (m rowWrite) param() string {
	return projectWrite{path: m.path}.param()
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

// A write on a row under a project that waits for a lock reads, once it has
// it, what another transaction committed meanwhile, and answers its 404, as
// for a row or a project never there, never the 403 that would tell its
// caller that it was there; and it changes no row (M3 design 3.6
// convention 2, 8.2). The other transaction holds Web FOR NO KEY UPDATE, as
// a write on it does, and ends (a membership), deletes or moves to Ops the
// row the write changes, or ends the caller's own membership, or deletes
// Web; or it holds acme FOR NO KEY UPDATE, as a cascade does, and deletes
// acme. SQL makes each change inside the other transaction, which must hold
// its lock open across the probe: a real write cannot without a hook in
// product code. The write has passed authentication and its read without a
// lock, of the row it names or, leaving or creating a state, of Web's
// workspace, and waits for the row the other holds. Once the other commits,
// the write is 404; the row the other changed is as it left it, and every
// other row as it was, no state created among them. Web is private: bob or
// dave, his membership ended, does not see it.
func TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile(t *testing.T) {
	type change struct {
		name, holds, sql string // holds: the table of the row the other transaction locks first, acme's or Web's; sql: %s the table
		table            string // the table sql changes; empty for the write's row's
		ofRow            bool   // the change is of the write's row, not its caller's own membership
		membership       bool   // it ends its row: a membership's change alone
		row              func(w memberWorld, t *testing.T, m rowWrite) uuid.UUID
	}
	ofRow := func(w memberWorld, t *testing.T, m rowWrite) uuid.UUID {
		_, id := m.row(t, w)
		return id
	}
	callers := func(w memberWorld, t *testing.T, m rowWrite) uuid.UUID { return w.membership(t, w.web, m.by) }
	web := func(w memberWorld, _ *testing.T, _ rowWrite) uuid.UUID { return w.web }
	acme := func(w memberWorld, t *testing.T, _ rowWrite) uuid.UUID {
		var id uuid.UUID
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT id FROM workspaces WHERE slug = 'acme'").Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	changes := []change{
		{"the row ended", "projects", "UPDATE %s SET is_active = false WHERE id = $1", "", true, true, ofRow},
		{"the row deleted", "projects", "UPDATE %s SET deleted_at = now() WHERE id = $1", "", true, false, ofRow},
		{"the row moved to Ops", "projects", "UPDATE %s SET project_id = (SELECT id FROM projects WHERE name = 'Ops') WHERE id = $1", "", true,
			false, ofRow},
		{"the caller's membership ended", "projects", "UPDATE %s SET is_active = false WHERE id = $1", "project_members", false, false, callers},
		{"Web deleted", "projects", "UPDATE %s SET deleted_at = now(), updated_at = now() WHERE id = $1", "projects", false, false, web},
		{"acme deleted", "workspaces", "UPDATE %s SET deleted_at = now(), updated_at = now() WHERE id = $1", "workspaces", false, false, acme},
	}
	for _, m := range rowWrites {
		for _, c := range changes {
			if c.ofRow && m.state == "" && m.member == m.by || c.membership && m.state != "" {
				continue // leaving, creating: no row but the caller's own membership; a state does not end
			}
			t.Run(m.op+", "+c.name, func(t *testing.T) {
				w := newPrivateWorld(t)
				table := c.table
				if table == "" {
					table, _ = m.row(t, w)
				}
				id := c.row(w, t, m)
				others := rowsBut(t, w.pool, []uuid.UUID{id})
				first := "SELECT 1 FROM projects WHERE id = $1 FOR NO KEY UPDATE"
				firstID := w.web
				if c.holds == "workspaces" {
					first, firstID = "SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", acme(w, t, m)
				}
				other := holding(t, w.pool, first, firstID)
				sql := strings.Replace(c.sql, "%s", table, 1)
				if tag, err := other.Exec(pgtest.Soon(t), sql, id); err != nil || tag.RowsAffected() != 1 {
					t.Fatalf("%s: %v, %v; want one row changed", sql, tag, err)
				}
				changed := rowJSON(t, other, table, id)
				req, answered := m.sent(t, w)
				pgtest.WaitForLockWaitOn(t, w.pool, c.holds, 5*time.Second)
				if err := other.Commit(pgtest.Soon(t)); err != nil {
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
				if after := rowJSON(t, w.pool, table, id); !reflect.DeepEqual(after, changed) {
					t.Errorf("%s %s after the write:\n%v\nwant it as the other transaction left it:\n%v", table, id, after, changed)
				}
				if after := rowsBut(t, w.pool, []uuid.UUID{id}); !maps.Equal(after, others) {
					t.Errorf("every other row after the write:\n%v\nwant them as they were:\n%v", after, others)
				}
			})
		}
	}
}

// Each lock of a write on a row under a project is taken in the lock
// table's order, at its strength, as bootstrap wires it (M3 design 3.6's
// lock table, conventions 2 and 3). Other transactions hold acme's row FOR
// NO KEY UPDATE and, FOR SHARE, Web's row and the row of Web the write
// changes, which the write waits for in turn; they let go one at a time,
// and lockOn reads each row's strongest lock then:
//   - waiting for acme's row, the write holds neither the member's
//     membership of acme, nor Web's default state, nor Ops, and nothing of
//     Web or of its row that conflicts with the others' shares, or it would
//     wait there; a share of either, hidden under the others', is
//     TestEachWriteOnAProjectSharesItsWorkspaceFirst's to see;
//   - waiting for Web's row, it holds acme's FOR SHARE and, for a change of
//     a role, the member's membership of acme FOR SHARE (convention 3), no
//     stronger; that membership not at all for any other write;
//   - waiting for its row, it holds Web FOR NO KEY UPDATE too, and still not
//     Ops, alice's other project of acme, of which gina and dave are no
//     members; Web's default, Backlog, only a write that makes its state
//     the default holds, FOR NO KEY UPDATE, having written it first (M3
//     design 3.17).
//
// Then it answers as alone. The moment it wrote is no earlier than Web's
// release, as it read the clock under Web's lock (3.3), and earlier than
// its row's release: it read the clock before it waited for its row, which
// only its write's UPDATE locks; nothing before the decision and the clock
// does, not the read of it under the locks either.
func TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength(t *testing.T) {
	for _, m := range rowWrites {
		if m.creates {
			// A creation has no row of its own to wait on: the order and the
			// strength of its locks are TestEachWriteOnAProjectSharesItsWorkspaceFirst's
			// and the two creations' of TestStateWritesOnOneProjectSerialize.
			continue
		}
		t.Run(m.op, func(t *testing.T) {
			w := newMemberWorld(t)
			var acme, inAcme uuid.UUID
			if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT s.id, m.id FROM workspaces s JOIN workspace_members m ON m.workspace_id = s.id
				WHERE s.slug = 'acme' AND m.member_id = $1`, w.ids[m.member]).Scan(&acme, &inAcme); err != nil {
				t.Fatal(err)
			}
			table, row := m.row(t, w)
			backlog := stateID(t, w.pool, w.web, "Backlog")
			locks := func() string {
				t.Helper()
				return "acme " + lockOn(t, w.pool, "workspaces WHERE id = $1", acme) +
					", in acme " + lockOn(t, w.pool, "workspace_members WHERE id = $1", inAcme) +
					", Web " + lockOn(t, w.pool, "projects WHERE id = $1", w.web) +
					", its row " + lockOn(t, w.pool, table+" WHERE id = $1", row) +
					", Web's default " + lockOn(t, w.pool, "states WHERE id = $1", backlog) +
					", Ops " + lockOn(t, w.pool, "projects WHERE id = $1", w.ops)
			}
			inAcmeShared, defaultHeld := "no lock", "no lock"
			if m.target {
				inAcmeShared = "FOR SHARE"
			}
			if m.clears {
				defaultHeld = "FOR NO KEY UPDATE"
			}
			holdsRow := holding(t, w.pool, "SELECT 1 FROM "+table+" WHERE id = $1 FOR SHARE", row)
			holdsWeb := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR SHARE", w.web)
			holdsAcme := holding(t, w.pool, "SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", acme)
			req, answered := m.sent(t, w)
			var releasedWeb time.Time
			for _, step := range []struct {
				release pgx.Tx
				waitsOn string
				want    string
			}{
				{nil, "workspaces", "acme FOR NO KEY UPDATE, in acme no lock, Web FOR SHARE, its row FOR SHARE, Web's default no lock, Ops no lock"},
				{holdsAcme, "projects", "acme FOR SHARE, in acme " + inAcmeShared + ", Web FOR SHARE, its row FOR SHARE, Web's default no lock, " +
					"Ops no lock"},
				{holdsWeb, table, "acme FOR SHARE, in acme " + inAcmeShared + ", Web FOR NO KEY UPDATE, its row FOR SHARE, Web's default " +
					defaultHeld + ", Ops no lock"},
			} {
				if step.release != nil {
					if step.release == holdsWeb {
						releasedWeb = time.Now()
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
			releasedRow := time.Now()
			if err := holdsRow.Rollback(context.Background()); err != nil {
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
			moment, _ := rowJSON(t, w.pool, table, row)["updated_at"].(string)
			if at, err := time.Parse(time.RFC3339Nano, moment); err != nil || at.Before(releasedWeb.Truncate(time.Microsecond)) ||
				!at.Before(releasedRow) {
				t.Errorf("%s's moment %q (%v); want one no earlier than Web's release, %v, and earlier than its row's, %v", m.op, moment,
					err, releasedWeb, releasedRow)
			}
		})
	}
}
````

- [ ] **Step 3: 交错 10、一组最后两个状态、每个写只写它的行**

`server/internal/bootstrap/interleaving_states_test.go`（新文件，249 行）：

````file server/internal/bootstrap/interleaving_states_test.go
package bootstrap

import (
	"context"
	"maps"
	"testing"
	"time"
	"uuid"

	projectpg "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	projectdomain "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Interleaving 10, the last state of a group and a state's sequence (M3
// design 9.3, 3.17), on memberWorld's real database, in both orders:
// project's CreateState, UpdateState, DeleteState and MarkDefaultState over
// projectLocks, with the Authorizer as bootstrap wires them, the first held
// at a gate past its write. Every wait has a deadline.

// stateStore is what a write on a state reads and writes through: project's
// store, or one held at a gate.
type stateStore interface {
	projectapp.StateCreator
	projectapp.StateUpdater
	projectapp.StateDeleter
	projectapp.DefaultMarker
}

// stateWrittenHolding stops a write on a state past its write: once it
// holds the workspace FOR SHARE, the project FOR NO KEY UPDATE and the rows
// it wrote, before its commit; a deletion before it counts the group.
type stateWrittenHolding struct {
	*projectpg.Store
	gate *gate
}

func (s stateWrittenHolding) CreateState(ctx context.Context, r projectapp.StateRow) (projectdomain.State, error) {
	created, err := s.Store.CreateState(ctx, r)
	if err != nil {
		return projectdomain.State{}, err
	}
	return created, s.gate.wait(ctx)
}

func (s stateWrittenHolding) UpdateState(ctx context.Context, id uuid.UUID, p projectdomain.StatePatch, by uuid.UUID,
	now time.Time) (projectdomain.State, error) {
	changed, err := s.Store.UpdateState(ctx, id, p, by, now)
	if err != nil {
		return projectdomain.State{}, err
	}
	return changed, s.gate.wait(ctx)
}

func (s stateWrittenHolding) DeleteState(ctx context.Context, id, by uuid.UUID, now time.Time) (bool, error) {
	deleted, err := s.Store.DeleteState(ctx, id, by, now)
	if err != nil {
		return false, err
	}
	return deleted, s.gate.wait(ctx)
}

func (s stateWrittenHolding) MarkDefaultState(ctx context.Context, projectID, id, by uuid.UUID, now time.Time) (bool, error) {
	marked, err := s.Store.MarkDefaultState(ctx, projectID, id, by, now)
	if err != nil {
		return false, err
	}
	return marked, s.gate.wait(ctx)
}

// projectState is a state of a project as statesOf reads it.
type projectState struct {
	Group    projectdomain.StateGroup
	Sequence float64
	Default  bool
}

// statesOf is project's undeleted states, its triage state too, by name.
func (w memberWorld) statesOf(t *testing.T, project uuid.UUID) map[string]projectState {
	t.Helper()
	rows, err := w.pool.Query(pgtest.Soon(t), `SELECT name, "group", sequence, "default" FROM states WHERE project_id = $1 AND deleted_at IS NULL`,
		project)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	states := map[string]projectState{}
	for rows.Next() {
		var name string
		var s projectState
		if err := rows.Scan(&name, &s.Group, &s.Sequence, &s.Default); err != nil {
			t.Fatal(err)
		}
		states[name] = s
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return states
}

// stateWrite is bob's write on Web's states, through project's use case
// over a store, with projectLocks, as project.New wires it, given the ids
// of Web's states by name; apply is what it does to Web's statesOf.
type stateWrite struct {
	name  string
	run   func(ctx context.Context, w memberWorld, states stateStore, ids map[string]uuid.UUID) error
	apply func(states map[string]projectState)
}

// asBob is ctx with bob, Web's admin, its caller.
func (w memberWorld) asBob(ctx context.Context) context.Context {
	return shared.WithActor(ctx, shared.Actor{UserID: w.ids["bob"]})
}

// creates is the creation of the state name in group: after the greatest
// sequence of Web's undeleted states but its triage state: not Ops's,
// whose Cancelled comes after them all, nor Web's deleted Retired.
func creates(name string, group projectdomain.StateGroup) stateWrite {
	return stateWrite{"creates " + name, func(ctx context.Context, w memberWorld, states stateStore, _ map[string]uuid.UUID) error {
		_, err := projectapp.NewCreateState(w.projectLocks(), states, w.tx(), clock.System{}).Execute(w.asBob(ctx), w.web, projectdomain.StateCreate{Name: name, Color: "#0EA5E9", Group: group})
		return err
	}, func(states map[string]projectState) {
		greatest := 0.0
		for _, s := range states {
			if s.Group != projectdomain.GroupTriage {
				greatest = max(greatest, s.Sequence)
			}
		}
		states[name] = projectState{Group: group, Sequence: greatest + 15000}
	}}
}

func deletes(name string) stateWrite {
	return stateWrite{"deletes " + name, func(ctx context.Context, w memberWorld, states stateStore, ids map[string]uuid.UUID) error {
		return projectapp.NewDeleteState(w.projectLocks(), states, w.tx(), clock.System{}).Execute(w.asBob(ctx), ids[name])
	}, func(states map[string]projectState) { delete(states, name) }}
}

func moves(name string, group projectdomain.StateGroup) stateWrite {
	return stateWrite{"moves " + name + " to " + string(group), func(ctx context.Context, w memberWorld, states stateStore,
		ids map[string]uuid.UUID) error {
		_, err := projectapp.NewUpdateState(w.projectLocks(), states, w.tx(), clock.System{}).Execute(w.asBob(ctx), ids[name], projectdomain.StatePatch{Group: &group})
		return err
	}, func(states map[string]projectState) {
		s := states[name]
		s.Group = group
		states[name] = s
	}}
}

func marksDefault(name string) stateWrite {
	return stateWrite{"marks " + name + " the default", func(ctx context.Context, w memberWorld, states stateStore, ids map[string]uuid.UUID) error {
		return projectapp.NewMarkDefaultState(w.projectLocks(), states, w.tx(), clock.System{}).Execute(w.asBob(ctx), ids[name])
	}, func(states map[string]projectState) {
		for n, s := range states {
			s.Default = n == name
			states[n] = s
		}
	}}
}

// Two writes on Web's states at once serialize on Web's row (M3 design 3.6,
// 3.17; 9.3, interleaving 10), in both orders. The first holds acme FOR
// SHARE, Web FOR NO KEY UPDATE and the rows it wrote, and waits at its gate
// past its write: a deletion before it counts the group, a move after its
// count, a creation after its read of the greatest sequence. The second
// shares acme and waits for Web's row: neither writes a row of projects, so
// only that lock's wait satisfies the probe (the order of the second's
// locks is TestEachWriteOnAProjectSharesItsWorkspaceFirst's). Once the
// first has committed, the second decides on what it committed:
//   - Done and QA, the completed group's two states: one deleted or moved to
//     another group, the other's deletion or move is 409
//     project.state_last_in_group: the group keeps one, whichever goes
//     first;
//   - QA made the default and deleted (interleaving 10): made the default
//     first, its deletion is 409 project.state_default; deleted first,
//     making it the default is 404 project.state_not_found; Web keeps one
//     default either way;
//   - two states created: each after the greatest sequence it reads, the
//     second's after the first's, at a moment no earlier than the gate's
//     opening: it read the clock under the locks it took once the first
//     had committed (M3 design 3.3).
//
// Web's states are then as the first left them, and the second's too when
// it succeeded, each read with its group, sequence and default; Ops's, in
// acme too, and Lab's, in beta, as they were.
func TestStateWritesOnOneProjectSerialize(t *testing.T) {
	last := projectdomain.ErrStateLastInGroup
	for _, tt := range []struct {
		a, b               stateWrite
		ifAFirst, ifBFirst error // the second's answer
	}{
		{deletes("Done"), deletes("QA"), last, last},
		{deletes("Done"), moves("QA", projectdomain.GroupStarted), last, last},
		{moves("Done", projectdomain.GroupStarted), moves("QA", projectdomain.GroupBacklog), last, last},
		{marksDefault("QA"), deletes("QA"), projectdomain.ErrStateDefault, projectdomain.ErrStateNotFound},
		{creates("Checked", projectdomain.GroupCompleted), creates("Shipped", projectdomain.GroupCompleted), nil, nil},
	} {
		for _, aFirst := range []bool{true, false} {
			first, second, want := tt.a, tt.b, tt.ifAFirst
			if !aFirst {
				first, second, want = tt.b, tt.a, tt.ifBFirst
			}
			t.Run(first.name+", then "+second.name, func(t *testing.T) {
				w := newMemberWorld(t)
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				states, ops, lab := w.statesOf(t, w.web), w.statesOf(t, w.ops), w.statesOf(t, w.lab)
				ids := map[string]uuid.UUID{}
				for name := range states {
					ids[name] = stateID(t, w.pool, w.web, name)
				}
				g := newGate()
				done := run(func() error { return first.run(ctx, w, stateWrittenHolding{projectpg.New(w.pool), g}, ids) })
				held(t, ctx, g, done, "the first write")
				answered := run(func() error { return second.run(ctx, w, projectpg.New(w.pool), ids) })
				pgtest.WaitForLockWaitOn(t, w.pool, "projects", 5*time.Second)
				opened := time.Now().Truncate(time.Microsecond)
				close(g.open)

				if err := result(t, ctx, done, "the first write"); err != nil {
					t.Errorf("%s = %v, want it done", first.name, err)
				}
				if err := result(t, ctx, answered, "the second write"); !sameOutcome(err, want) {
					t.Errorf("%s = %v, want %v as its first problem", second.name, err, want)
				}
				first.apply(states)
				if want == nil {
					second.apply(states)
					var latest time.Time
					err := w.pool.QueryRow(pgtest.Soon(t), "SELECT max(updated_at) FROM states WHERE project_id = $1", w.web).Scan(&latest)
					if err != nil || latest.Before(opened) {
						t.Errorf("Web's last write at %v (%v); want one no earlier than the gate's opening, %v", latest, err, opened)
					}
				}
				if got := w.statesOf(t, w.web); !maps.Equal(got, states) {
					t.Errorf("Web's states after both: %v; want %v", got, states)
				}
				if got, gotLab := w.statesOf(t, w.ops), w.statesOf(t, w.lab); !maps.Equal(got, ops) || !maps.Equal(gotLab, lab) {
					t.Errorf("Ops's and Lab's states after both: %v, %v; want them as they were, %v, %v", got, gotLab, ops, lab)
				}
			})
		}
	}
}
````

`server/internal/bootstrap/state_rows_test.go`（新文件，71 行）：

````file server/internal/bootstrap/state_rows_test.go
package bootstrap

import (
	"maps"
	"net/http"
	"testing"
	"uuid"
)

// Each write on a state, as bootstrap wires it, changes the rows it writes
// and no other (M3 design 3.17): on memberWorld, whose acme has Web and Ops
// and whose beta has Lab, each with its states, bob, Web's admin, creates
// Shipped in Web, renames QA, deletes it, renames Backlog, the only state
// of its group, giving its own group again, which moves it nowhere, and
// makes Done Web's default, one write after another; between them, his
// creation of a state in the triage group and his move of Done to it,
// bodies the contract's enum leaves out and the server must still answer
// as the contract says, are each 422 group not_allowed, refused before
// the write's transaction begins. After each, every row of every table but
// the rows it writes is as it was before it: Web's other states, its
// triage state among them, Ops's, in acme too, and Lab's, in beta. The
// rows each writes are its new state, the state it names, and, for the
// default, Web's default before it, Backlog; a refused one writes none.
func TestEachStateWriteChangesItsRowsAlone(t *testing.T) {
	w := newMemberWorld(t)
	qa, done, backlog := stateID(t, w.pool, w.web, "QA"), stateID(t, w.pool, w.web, "Done"), stateID(t, w.pool, w.web, "Backlog")
	for _, s := range []struct {
		name, method, path, body string
		status                   int
		writes                   []uuid.UUID // the rows it writes that are there before it
		creates                  string      // the name of the state it creates, a row it writes too
		// refusal, when set, is the one problem (field and code) a refused
		// write answers, for a body the contract's enum leaves out.
		refusal string
	}{
		{"createState", http.MethodPost, "/api/v0/projects/" + w.web.String() + "/states", `{"name":"Shipped","color":"#46A758","group":"completed"}`,
			http.StatusCreated, nil, "Shipped", ""},
		{"updateState", http.MethodPatch, "/api/v0/states/" + qa.String(), `{"name":"Checked"}`, http.StatusOK, []uuid.UUID{qa}, "", ""},
		{"deleteState", http.MethodDelete, "/api/v0/states/" + qa.String(), "", http.StatusNoContent, []uuid.UUID{qa}, "", ""},
		{"updateState, its own group", http.MethodPatch, "/api/v0/states/" + backlog.String(), `{"name":"Later","group":"backlog"}`, http.StatusOK,
			[]uuid.UUID{backlog}, "", ""},
		{"createState, the triage group", http.MethodPost, "/api/v0/projects/" + w.web.String() + "/states",
			`{"name":"Intake","color":"#4E5355","group":"triage"}`, http.StatusUnprocessableEntity, nil, "", "group not_allowed"},
		{"updateState, to the triage group", http.MethodPatch, "/api/v0/states/" + done.String(), `{"group":"triage"}`,
			http.StatusUnprocessableEntity, nil, "", "group not_allowed"},
		{"markDefaultState", http.MethodPost, "/api/v0/states/" + done.String() + "/mark-default", "", http.StatusNoContent,
			[]uuid.UUID{done, backlog}, "", ""},
	} {
		before := rowsBut(t, w.pool, s.writes)
		var body []byte
		if s.body != "" {
			body = []byte(s.body)
		}
		req := newRequest(t, s.method, w.base+s.path, w.tokens["bob"], body)
		if s.refusal == "" {
			w.contract.CheckRequest(t, req)
		}
		res, answer := send(t, req)
		w.contract.CheckResponse(t, req, res)
		if res.StatusCode != s.status || s.refusal != "" && oneError(t, answer) != s.refusal {
			t.Fatalf("%s = %d %s, want %d %s", s.name, res.StatusCode, answer, s.status, s.refusal)
		}
		written := s.writes
		if s.creates != "" {
			written = append(written, stateID(t, w.pool, w.web, s.creates))
		}
		if after := rowsBut(t, w.pool, written); !maps.Equal(after, before) {
			t.Errorf("%s: every other row after it:\n%v\nwant them as they were:\n%v", s.name, after, before)
		}
	}
}
````

- [ ] **Step 4: 测试和 lint**

Run: `go -C server test -count=5 -race -run 'TestStateWritesOnOneProjectSerialize$|TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile$|TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength$|TestTwoProjectAdminsLeavingLeaveAnAdmin$|TestTwoAdminsDemotingEachOtherLeaveAnAdmin$' ./internal/bootstrap/`
Expected: `ok`，输出里没有 `40P01`（原型上 `race5.sh` 跑这五个和另外七个，各 5 次全部通过，155.1 秒，没有数据竞争）。

Run: `go -C server test -count=1 -run 'TestEachStateWriteChangesItsRowsAlone$' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 5: 提交**

```bash
git add server/internal/bootstrap/interleaving_leaving_test.go server/internal/bootstrap/interleaving_states_test.go server/internal/bootstrap/membership_world_test.go server/internal/bootstrap/project_membership_races_test.go server/internal/bootstrap/project_row_races_test.go server/internal/bootstrap/state_rows_test.go
```
```bash
git commit -m "test(M3/P7a): the writes on a project's states race as one ladder

Two writes on one project's states serialize on the project's row in
either order: the completed group's two states each deleted or moved
keep one, the second write's 409; a state made the default and deleted
(interleaving 10) is the deletion's 409 one way and the marking's 404
the other, the project keeping one default; two states created take
their sequences in turn. P5b's races and lock strengths become those of
any write on a row under a project, the state writes among them, and
createState, addressed by its project as leaving is, races the same
endings and deletions; each state write changes its rows alone.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**：本 Task 没有产品代码；它的测试在别的 Task 的变异表里（"（Task 9 起）"）。

---

### Task 10: 故事 P6、W11 的接口版本；`plane-diff.md`；标签的 Phase 改称 P7b

**Files:**
- Create: `e2e/stories/project/p6-states.spec.ts`、`e2e/stories/workspace/w11-guest-bounds.spec.ts`
- Modify: `docs/v0/plane-diff.md`、`e2e/fixtures/api.ts`、`e2e/fixtures/assert/project.ts`、`e2e/fixtures/assert/workspace.ts`、`e2e/stories/project/p5-project-members.spec.ts`、`server/internal/modules/project/app/deletion.go`

**Interfaces:**
- 故事（M3 设计 10 节 P6、W11；接口版本；页面的版本在 P11）：
  - `p6-states.spec.ts`：Web 的管理员 ann 建 Review（started，70000）；工作区管理员 admin（他建了 Web，是它的另一位管理员）改它的颜色（`sequence` 不变），再把它拖到 In Progress 之前（30000，颜色不变）；ann 把它设为默认，Backlog 不再是；之后五个拒绝：删除默认的 Review 409 `project.state_default`、删除组里唯一的 Backlog、把它改到有两个状态（Review、In Progress）的 started 组，各 409 `project.state_last_in_group`（数的是它自己的组）、建分诊组的状态 422 `{field: group, code: not_allowed}`、项目成员 mem 改状态 403；ann 删除 In Progress（最后由 admin 写）；之后再删除它、改 Web 的分诊状态的名称，各 404 `project.state_not_found`；mem 的列表；ann 建 Ops，工作区的列表按项目 id；ann 归档 Ops，它列出 `[]`，工作区的列表只有 Web 的；最后 admin 在 Elsewhere 的两个项目（`amidAnotherWorkspace` 建在 Web 前后）的状态照建出时的样子：Web 上的写没有写别的项目。每一步之后、每组拒绝之前之后用 SQL 读出状态的行（`expectStates`：名称、颜色、组、`sequence`、默认、删除、最后写它的人、行在它的工作区、删除的时刻就是它最后一次写的时刻；清扫 27、38）。
  - `w11-guest-bounds.spec.ts`：工作区和 Web 的访客 gus：读邀请 403 `forbidden`、建状态 403、读他不是成员的私有项目 Secret 404 `project.not_found`、成员的邮箱对他为 `null`（管理员看得到）；之后五张表不变。
- 夹具：`api.ts` 的 `State`、`StateCreate`、`StateUpdate`，`answer`（从 P5 的故事提到夹具，P6 也用），`createState`；`assert/project.ts` 的 `expectStates`、`statesOfANewProject`。
- `docs/v0/plane-diff.md` 4.11：修改状态的人、分诊状态对状态操作不可见和设为默认的事务、一组中唯一的状态三行（3.20 中 P7a 的行）。
- `e2e/fixtures/assert/workspace.ts`、`project/app/deletion.go`：标签的说明改称 P7b。

**Tests:**
- e2e：P6（API）、W11（API）新；P5（API）改用夹具的 `answer`，步骤不变。

- [ ] **Step 1: 夹具和断言**

`e2e/fixtures/api.ts`（修改，3 处）：

````old e2e/fixtures/api.ts
export type ProjectPreferencesUpdate = components["schemas"]["ProjectPreferencesUpdate"];
````
````new e2e/fixtures/api.ts
export type ProjectPreferencesUpdate = components["schemas"]["ProjectPreferencesUpdate"];
export type State = components["schemas"]["State"];
export type StateCreate = components["schemas"]["StateCreate"];
export type StateUpdate = components["schemas"]["StateUpdate"];
````

````old e2e/fixtures/api.ts
  return createClient({ baseUrl: baseURL });
````
````new e2e/fixtures/api.ts
  return createClient({ baseUrl: baseURL });
}

/** A call's answer: its status, and the problem's code and fields for a refusal. */
interface Answer {
  status: number;
  code?: string;
  errors?: { field: string; code: string }[];
}

/** The answer of a call whose response is response, and whose problem is error when it is refused. */
export function answer(
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
````

````old e2e/fixtures/api.ts
  return data.data;
}

/**
 * Runs make, which makes a story's projects, between two projects of another workspace of the caller of token, one
````
````new e2e/fixtures/api.ts
  return data.data;
}

/** Creates a state in the project of projectId with the bearer token given, an admin's of the project, and returns it. */
export async function createState(api: Api, token: string, projectId: string, body: StateCreate): Promise<State> {
  const { data, error, response } = await api.POST("/api/v0/projects/{project_id}/states", {
    params: { path: { project_id: projectId } },
    body,
    headers: bearer(token),
  });
  expect(response.status, `create the state ${body.name} in ${projectId}: ${JSON.stringify(error)}`).toBe(201);
  if (!data) {
    throw new Error(`createState ${body.name} answered 201 without the state`);
  }
  return data;
}

/**
 * Runs make, which makes a story's projects, between two projects of another workspace of the caller of token, one
````

`e2e/fixtures/assert/project.ts`（修改，1 处）：

````old e2e/fixtures/assert/project.ts
  expect(settings?.count, `the display settings in ${projectId}`).toBe(want.length);
}

````
````new e2e/fixtures/assert/project.ts
  expect(settings?.count, `the display settings in ${projectId}`).toBe(want.length);
}

/** A state of a project as expectStates reads it. */
export interface StateRow {
  name: string;
  color: string;
  group: string;
  sequence: number;
  default: boolean;
  deleted: boolean;
  /** The address of the account that wrote the state last: created, changed, deleted it, or made it the default or not. */
  by: string;
}

/** The states of a new project that the account of creatorEmail created, as its creation made them (newStates). */
export function statesOfANewProject(creatorEmail: string): StateRow[] {
  return newStates.map(({ name, color, sequence, group, default: isDefault }) => ({
    name,
    color,
    group,
    sequence,
    default: isDefault,
    deleted: false,
    by: creatorEmail,
  }));
}

/**
 * P6: the states of the project of projectId, deleted ones too, its triage state among them, are exactly want, by
 * sequence, then name. Each is a row of the project's workspace, and a deleted one was deleted at the moment of its
 * last write, by its writer (M3 design 3.17).
 */
export async function expectStates(db: Database, projectId: string, want: StateRow[]): Promise<void> {
  expect(
    await db.query(
      `SELECT s.name, s.color, s."group" AS group, s.sequence, s."default" AS default, s.deleted_at IS NOT NULL AS deleted,
              b.email AS by, s.workspace_id = p.workspace_id AS in_its_workspace,
              coalesce(s.deleted_at = s.updated_at, true) AS deleted_with_its_last_write
         FROM states s
         JOIN projects p ON p.id = s.project_id
         JOIN users b ON b.id = s.updated_by_id
        WHERE s.project_id = $1 ORDER BY s.sequence, s.name COLLATE "C"`,
      [projectId]
    ),
    `the states of ${projectId}`
  ).toEqual(
    want
      .toSorted((a, b) => a.sequence - b.sequence || (a.name < b.name ? -1 : 1))
      .map(({ name, color, group, sequence, default: isDefault, deleted, by }) => ({
        name,
        color,
        group,
        sequence,
        default: isDefault,
        deleted,
        by,
        in_its_workspace: true,
        deleted_with_its_last_write: true,
      }))
  );
}

````

`e2e/fixtures/assert/workspace.ts`（修改，1 处）：

````old e2e/fixtures/assert/workspace.ts
/** The tables whose rows belong to a workspace and are deleted with it (M3 design 3.6, 4.12); P7 adds the labels. */
````
````new e2e/fixtures/assert/workspace.ts
/** The tables whose rows belong to a workspace and are deleted with it (M3 design 3.6, 4.12); P7b adds the labels. */
````

`e2e/stories/project/p5-project-members.spec.ts`（修改，2 处）：

````old e2e/stories/project/p5-project-members.spec.ts
  amidAnotherWorkspace,
````
````new e2e/stories/project/p5-project-members.spec.ts
  amidAnotherWorkspace,
  answer,
````

````old e2e/stories/project/p5-project-members.spec.ts
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
````
````new e2e/stories/project/p5-project-members.spec.ts
// project's members page (P10).
````

`server/internal/modules/project/app/deletion.go`（修改，1 处）：

````old server/internal/modules/project/app/deletion.go
// display settings, the states; P7 adds the labels at the end. Deleting a
````
````new server/internal/modules/project/app/deletion.go
// display settings, the states; P7b adds the labels at the end. Deleting a
````

- [ ] **Step 2: 两个故事**

`e2e/stories/project/p6-states.spec.ts`（新文件，245 行）：

````file e2e/stories/project/p6-states.spec.ts
import {
  addProjectMembers,
  amidAnotherWorkspace,
  answer,
  createProject,
  createState,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
  type State,
  type StateCreate,
  type StateUpdate,
} from "../../fixtures/api";
import { expectStates, statesOfANewProject, type StateRow } from "../../fixtures/assert/project";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// P6, a project's states (M3 design 2, 3.17): creating one, changing it,
// making it the default and deleting one, with what the default and every
// group keep. The page version comes with the states' settings page (P11).

/** The states of the project of projectId that the caller of token lists, in their order. */
async function listStates(api: Api, token: string, projectId: string): Promise<State[]> {
  const { data, error, response } = await api.GET("/api/v0/projects/{project_id}/states", {
    params: { path: { project_id: projectId } },
    headers: bearer(token),
  });
  expect(response.status, `list ${projectId}'s states: ${JSON.stringify(error)}`).toBe(200);
  return data?.data ?? [];
}

/** The states of the workspace of slug that the caller of token lists, each as its project's id and its name, in their order. */
async function listWorkspaceStates(api: Api, token: string, slug: string): Promise<[string, string][]> {
  const { data, error, response } = await api.GET("/api/v0/workspaces/{slug}/states", {
    params: { path: { slug } },
    headers: bearer(token),
  });
  expect(response.status, `list ${slug}'s states: ${JSON.stringify(error)}`).toBe(200);
  return (data?.data ?? []).map((s) => [s.project_id, s.name]);
}

/** The writes on a state named by its id, each by the caller of token. */
function writes(api: Api) {
  return {
    update: async (token: string, stateId: string, body: StateUpdate) => {
      const { error, response } = await api.PATCH("/api/v0/states/{state_id}", {
        params: { path: { state_id: stateId } },
        body,
        headers: bearer(token),
      });
      return answer(response, error);
    },
    remove: async (token: string, stateId: string) => {
      const { error, response } = await api.DELETE("/api/v0/states/{state_id}", {
        params: { path: { state_id: stateId } },
        headers: bearer(token),
      });
      return answer(response, error);
    },
    markDefault: async (token: string, stateId: string) => {
      const { error, response } = await api.POST("/api/v0/states/{state_id}/mark-default", {
        params: { path: { state_id: stateId } },
        headers: bearer(token),
      });
      return answer(response, error);
    },
  };
}

/** rows with the state of name changed as to says. */
function changed(rows: StateRow[], name: string, to: Partial<StateRow>): StateRow[] {
  return rows.map((s) => (s.name === name ? { ...s, ...to } : s));
}

test("P6 (API): an admin of a project creates Review in the started group at 70000, after its states but the triage state; another admin changes its color and moves it before In Progress, and the first makes it the default, Backlog the default no longer; deleting the default, deleting the only state of a group or moving it to another, a triage state and a member's change are refused, each changing nothing; In Progress is deleted, Review left in its group, and neither it nor the triage state is found any more; a member lists the states but the triage state, an archived project lists none, and the workspace's list leaves out the triage states and the archived project's", async ({
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
  // ann and mem, acme's members: ann Web's other admin, mem its member.
  const [ann, mem] = await Promise.all(["ann", "mem"].map(account));
  if (!ann || !mem) {
    throw new Error("the accounts were not registered");
  }
  await Promise.all([ann, mem].map((who) => inviteAndAccept(api, admin.token, slug, who, 15)));
  const web = await amidAnotherWorkspace(api, admin.token, testInfo, () =>
    createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" })
  );
  await addProjectMembers(api, admin.token, web.id, [
    { member_id: ann.id, role: 20 },
    { member_id: mem.id, role: 15 },
  ]);
  const { update, remove, markDefault } = writes(api);
  const made = await listStates(api, ann.token, web.id);
  const [backlog, inProgress] = ["Backlog", "In Progress"].map((name) => made.find((s) => s.name === name)?.id);
  if (!backlog || !inProgress) {
    throw new Error("Web was made without Backlog or In Progress");
  }
  // Web's states as expectStates reads them, each as Web's creation made it, the admin's.
  let states = statesOfANewProject(admin.email);
  await expectStates(db, web.id, states);

  // ann creates Review in the started group: after Web's states, the triage state's 65000 left out, at 55000 + 15000;
  // not the default.
  const review = await createState(api, ann.token, web.id, { name: "Review", color: "#8B5CF6", group: "started" });
  expect([review.name, review.group, review.sequence, review.default], "Review as created").toEqual([
    "Review",
    "started",
    70000,
    false,
  ]);
  states = [
    ...states,
    {
      name: "Review",
      color: "#8B5CF6",
      group: "started",
      sequence: 70000,
      default: false,
      deleted: false,
      by: ann.email,
    },
  ];
  await expectStates(db, web.id, states);

  // The admin changes its color, its sequence kept, then moves it between Todo (25000) and In Progress (35000), as a
  // drag does, its color kept.
  expect(await update(admin.token, review.id, { color: "#3E63DD" }), "the admin's change of Review's color").toEqual({
    status: 200,
  });
  states = changed(states, "Review", { color: "#3E63DD", by: admin.email });
  await expectStates(db, web.id, states);
  expect(await update(admin.token, review.id, { sequence: 30000 }), "the admin's move of Review").toEqual({
    status: 200,
  });
  states = changed(states, "Review", { sequence: 30000 });
  await expectStates(db, web.id, states);

  // ann makes Review the default: Backlog is the default no longer, both written by her; Web has one default.
  expect(await markDefault(ann.token, review.id), "ann makes Review the default").toEqual({ status: 204 });
  states = changed(changed(states, "Backlog", { default: false, by: ann.email }), "Review", {
    default: true,
    by: ann.email,
  });
  await expectStates(db, web.id, states);

  // Refused, each changing nothing of what was just read: ann's deletion of Review, the default; her deletion of
  // Backlog, its group's only state, and her move of it to the started group, which has two, Review and In Progress,
  // its own group counted; her creation of a state in the triage group; mem's change of Review, a member's.
  const triage = await api.POST("/api/v0/projects/{project_id}/states", {
    params: { path: { project_id: web.id } },
    body: { name: "Intake", color: "#4E5355", group: "triage" } as unknown as StateCreate,
    headers: bearer(ann.token),
  });
  expect(
    [
      await remove(ann.token, review.id),
      await remove(ann.token, backlog),
      await update(ann.token, backlog, { group: "started" }),
      answer(triage.response, triage.error),
      await update(mem.token, review.id, { name: "Checked" }),
    ],
    "ann's deletions of Review and of Backlog, her move of Backlog, her triage state, mem's change of Review"
  ).toEqual([
    { status: 409, code: "project.state_default" },
    { status: 409, code: "project.state_last_in_group" },
    { status: 409, code: "project.state_last_in_group" },
    { status: 422, code: "validation_failed", errors: [{ field: "group", code: "not_allowed" }] },
    { status: 403, code: "forbidden" },
  ]);
  await expectStates(db, web.id, states);

  // ann deletes In Progress, which the admin wrote last: the started group keeps Review.
  expect(await remove(ann.token, inProgress), "ann deletes In Progress").toEqual({ status: 204 });
  states = changed(states, "In Progress", { deleted: true, by: ann.email });
  await expectStates(db, web.id, states);

  // Not found, changing nothing: ann's deletion of In Progress again, and her renaming of Web's triage state, which no
  // state operation sees.
  const [triageState] = await db.query<{ id: string }>(
    `SELECT id FROM states WHERE project_id = $1 AND "group" = 'triage' AND deleted_at IS NULL`,
    [web.id]
  );
  if (!triageState) {
    throw new Error("Web was made without its triage state");
  }
  expect(
    [await remove(ann.token, inProgress), await update(ann.token, triageState.id, { name: "Intake" })],
    "ann's deletion of In Progress again, her renaming of the triage state"
  ).toEqual([
    { status: 404, code: "project.state_not_found" },
    { status: 404, code: "project.state_not_found" },
  ]);
  await expectStates(db, web.id, states);

  // mem lists Web's states: by sequence, the triage state and the deleted In Progress left out.
  expect(
    (await listStates(api, mem.token, web.id)).map((s) => [s.name, s.default]),
    "Web's states as mem lists them"
  ).toEqual([
    ["Backlog", false],
    ["Todo", false],
    ["Review", true],
    ["Done", false],
    ["Cancelled", false],
  ]);

  // Ops, ann's project of acme: acme's states as she lists them are Web's and Ops's, by project id, no triage state
  // among them, until she archives Ops; then Ops lists none, and acme's are Web's.
  const ops = await createProject(api, ann.token, slug, { name: "Ops", identifier: "OPS" });
  const webStates = ["Backlog", "Todo", "Review", "Done", "Cancelled"].map((name): [string, string] => [web.id, name]);
  const opsStates = ["Backlog", "Todo", "In Progress", "Done", "Cancelled"].map((name): [string, string] => [
    ops.id,
    name,
  ]);
  expect(await listWorkspaceStates(api, ann.token, slug), "acme's states as ann lists them").toEqual(
    web.id < ops.id ? [...webStates, ...opsStates] : [...opsStates, ...webStates]
  );
  const archived = await api.POST("/api/v0/projects/{project_id}/archive", {
    params: { path: { project_id: ops.id } },
    headers: bearer(ann.token),
  });
  expect(archived.response.status, `ann archives Ops: ${JSON.stringify(archived.error)}`).toBe(200);
  expect(await listStates(api, ann.token, ops.id), "the states of Ops, archived").toEqual([]);
  expect(await listWorkspaceStates(api, ann.token, slug), "acme's states, Ops archived").toEqual(webStates);

  // The admin's projects of Elsewhere, made around Web, have their states as they were made: no write on Web's
  // states wrote another project's.
  const elsewhere = await api.GET("/api/v0/workspaces/{slug}/projects", {
    params: { path: { slug: slugFor(testInfo, "elsewhere") } },
    headers: bearer(admin.token),
  });
  expect(elsewhere.data?.data.length, `Elsewhere's projects: ${JSON.stringify(elsewhere.error)}`).toBe(2);
  await Promise.all(
    (elsewhere.data?.data ?? []).map((project) => expectStates(db, project.id, statesOfANewProject(admin.email)))
  );
});
````

`e2e/stories/workspace/w11-guest-bounds.spec.ts`（新文件，89 行）：

````file e2e/stories/workspace/w11-guest-bounds.spec.ts
import {
  addProjectMembers,
  answer,
  createProject,
  createWorkspace,
  invite,
  inviteAndAccept,
  listMembers,
  slugFor,
} from "../../fixtures/api";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// W11, a guest's bounds (M3 design 2, 9.2): the permission matrix's backend
// test holds every cell; this story samples four of them through the API.
// The page version is P11's.

test("W11 (API): a guest of a workspace may not list its invitations nor create a state in the project he is a guest of, does not see a private project he is not a member of, and reads no member's address, his own neither; none of it changes a row of the workspace's", async ({
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
  const gus = await account("gus");
  await inviteAndAccept(api, admin.token, slug, gus, 5);
  // Web, public, gus its guest; Secret, private, the admin's alone; an invitation to olga, pending.
  const web = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, admin.token, web.id, [{ member_id: gus.id, role: 5 }]);
  const secret = await createProject(api, admin.token, slug, { name: "Secret", identifier: "SECRET", network: 0 });
  await invite(api, admin.token, slug, [{ email: emailFor(testInfo, "olga"), role: 15 }]);
  // The tables of the workspace's rows, whole: none of gus's calls writes one.
  const tables = async () => ({
    members: await db.query("SELECT * FROM workspace_members ORDER BY id"),
    invitations: await db.query("SELECT * FROM workspace_member_invites ORDER BY id"),
    projects: await db.query("SELECT * FROM projects ORDER BY id"),
    projectMembers: await db.query("SELECT * FROM project_members ORDER BY id"),
    states: await db.query("SELECT * FROM states ORDER BY id"),
  });
  const before = await tables();

  const invitations = await api.GET("/api/v0/workspaces/{slug}/invitations", {
    params: { path: { slug } },
    headers: bearer(gus.token),
  });
  const created = await api.POST("/api/v0/projects/{project_id}/states", {
    params: { path: { project_id: web.id } },
    body: { name: "Review", color: "#8B5CF6", group: "started" },
    headers: bearer(gus.token),
  });
  const read = await api.GET("/api/v0/projects/{project_id}", {
    params: { path: { project_id: secret.id } },
    headers: bearer(gus.token),
  });
  expect(
    [
      answer(invitations.response, invitations.error),
      answer(created.response, created.error),
      answer(read.response, read.error),
    ],
    "gus's listing of acme's invitations, his creation of a state in Web, his reading of Secret"
  ).toEqual([
    { status: 403, code: "forbidden" },
    { status: 403, code: "forbidden" },
    { status: 404, code: "project.not_found" },
  ]);

  // acme's members as gus lists them: no address, his own neither; the admin reads both.
  const addresses = async (token: string) =>
    (await listMembers(api, token, slug)).map((m) => [m.member.id, m.member.email]).toSorted();
  expect(await addresses(gus.token), "acme's members as gus lists them").toEqual(
    [
      [admin.id, null],
      [gus.id, null],
    ].toSorted()
  );
  expect(await addresses(admin.token), "acme's members as the admin lists them").toEqual(
    [
      [admin.id, admin.email],
      [gus.id, gus.email],
    ].toSorted()
  );
  expect(await tables(), "the tables after gus's calls").toEqual(before);
});
````

- [ ] **Step 3: 与 Plane 的差异**

`docs/v0/plane-diff.md`（修改，1 处）：

````old docs/v0/plane-diff.md
| 默认状态、分诊状态 | 在代码里维持唯一；`is_triage` 可以与 `group` 不一致 | 数据库保证每个项目各至多一个（部分唯一索引）；分诊状态只看 `group`（M3 设计 3.17） |
````
````new docs/v0/plane-diff.md
| 修改状态 | 项目的访客也能 | 新建、修改、删除状态和设为默认：项目管理员，或同时是工作区管理员的项目成员（M3 设计 3.4） |
| 默认状态、分诊状态 | 在代码里维持唯一；`is_triage` 可以与 `group` 不一致 | 数据库保证每个项目各至多一个（部分唯一索引）；分诊状态只看 `group`；状态的操作同样按 `group` 认出分诊状态（列表不含它，按 id 修改、删除、设为默认答 404 `project.state_not_found`），设为默认在同一个事务里先清掉原来的默认，项目恰好一个默认状态（M3 设计 3.17） |
| 一组中唯一的状态 | 服务端能删除、能改到别的组（页面不让） | 删除它、把它改到别的组：409 `project.state_last_in_group`，每一组（分诊组除外）至少一个状态（M3 设计 3.17） |
````

- [ ] **Step 4: 测试和 lint**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过（关键词守卫 5 个命中都有例外）。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 在仓库的检出里 69 个故事全部通过，P6、W11 在其中（S3 只在不是检出的副本里失败：它读构建的提交号，P4b 的 F4）。

- [ ] **Step 5: 提交**

```bash
git add docs/v0/plane-diff.md e2e/fixtures/api.ts e2e/fixtures/assert/project.ts e2e/fixtures/assert/workspace.ts e2e/stories/project/p5-project-members.spec.ts e2e/stories/project/p6-states.spec.ts e2e/stories/workspace/w11-guest-bounds.spec.ts server/internal/modules/project/app/deletion.go
```
```bash
git commit -m "test(M3/P7a): stories P6 and W11, through the API

P6: a project's admin creates a state, another of its admins, the
workspace's, changes it, the state is made the default, another is
deleted, and the default, the only state of a group, the triage group
and a project member's change are refused, each refusal leaving the
rows as they were; a deleted state and the triage state are not found;
the project's members list its states, the workspace's list is by
project, an archived project lists none, and another workspace's
projects keep their states.
W11: a workspace's guest reads no invitation, creates no state, sees
no private project he is not in, nor another member's email, and
changes nothing. plane-diff.md takes P7a's rows.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**：本 Task 没有产品代码；它的测试在别的 Task 的变异表里（"（Task 10 起）"）。
