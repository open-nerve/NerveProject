# M3/P4b 项目的管理、显示设置与成员的加入 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 项目的修改、归档、恢复、删除，每个成员自己的项目显示设置，项目成员的列出、添加、加入；项目一侧的成员关系增长（添加、加入，含恢复以前的成员行）与降为访客的连带在真实数据库上串行（交错 17）。九个操作各带操作名、规则行和矩阵行（175 格，含添加无效目标的 PM、X 格）；每个项目级的写先取工作区行的 `FOR SHARE`、再锁项目（负责人 2026-10-02 的方案 E），在锁之后、在事务的连接上判定，删除工作区等它们提交、以自己的一个时刻删除它们写的行；删除项目与删除工作区共用一处删除步骤，由目录驱动的组合测试核对；故事 P2、P3、P4、P8 的接口版本通过；差异清单中 P4b 的各行、M1-P3 交接的项目成员写好。

**Architecture:** 只动 `project` 模块、`access` 的规则表、`bootstrap` 和 e2e，`workspace` 模块只加一条查询和它的存储方法；不加迁移。跨模块端口只多一个方法：`WorkspaceDirectory` 的 `ShareWorkspaceByID`（按 id 取未删除的工作区行 `FOR SHARE`，与 P4a 的 `ShareWorkspaceBySlug` 同一个端口、同一种写法，`bootstrap` 照旧转换，不 JOIN、模块之间不导入）。`project`：domain（`ProjectPatch`、`CheckProjectPatch`、`CanAssign`、`Preferences`、`CheckPreferencesPatch`、`Member`、`CheckNewMembers`、`CanAdd`、`CheckTargets`、`CanJoin`、`JoinRole`、九个操作名）；app（九个用例；写的唯一一条加锁路径 `Locks.lockAndDecide`：不加锁读项目的工作区 → 工作区 `FOR SHARE` → 添加、加入的目标的工作区成员行 `FOR SHARE` → 项目 `FOR NO KEY UPDATE`（改设置 `FOR SHARE`），确认它还在那个工作区 → 判定，`project.New` 只建一个 `Locks`，写不持自己的 `Authorizer`；读的 `findAndDecide`；增长的一步 `growth`；删除的一处步骤 `deleteProjects`，`Cascade.DeleteWorkspaceProjects` 和 `deleteProject` 共用；按用例分的存储端口）；存储（`LockProject` `FOR NO KEY UPDATE`、`ShareProject` `FOR SHARE`、`ProjectWorkspace`、`UpdateProject`、`SetArchived`、`Memberships`、`ListMembers`、`RestoreMember`、`Preferences`、`UpsertPreferences`、`EnsurePreferences`；删除的四条语句带可选的项目）；HTTP（`api/modules/project.yaml` 的九个操作）。`access`：九条规则（`LevelProject` 的管理员、成员、访客集合，`project.join` 是 `LevelVisible`，工作区角色由用例按集合要求）。`bootstrap`：`ShareWorkspaceByID` 的转换；矩阵准备拆开、账户登记、成员的行；每个项目级的写先锁工作区（按矩阵中在项目一级写的行核对完整，一个新的写没有自己的行时失败）；组合出的写、删除、显示设置、恢复角色的测试；交错 17、降级与删除项目、项目级的写与降级、同一个项目上的两个写、同一个工作区里两个项目上的写；删除工作区与项目级的写、添加多个成员与删除工作区（预检的 M1）；写在事务的连接上。

**Tech Stack:** Go 1.27.1、pgx v5.11.0、sqlc v1.31.1（`CGO_ENABLED=0`）、oapi-codegen v2.8.0、goose v3.28.0、River v0.47.0、golangci-lint 2.13.2、PostgreSQL 18.6（testcontainers）；Node 24、pnpm 11.10.0、Playwright 1.63.0。

**Spec:** `docs/v0/M3-workspace-project/specs/P4b-project-members.md`（上级：`docs/v0/M3-workspace-project/M3-design.md`）

## Global Constraints

- **Go 版本和依赖**：本 plan 不执行 `go get`，不加任何 Go 模块或 npm 包。`server/go.mod` 和 `server/tools/go.mod` 保持 `go 1.27` 和 `toolchain go1.27.1`（golangci-lint 2.13.2 由 go1.27.0 构建，`go` 行更高就拒绝运行）。每个 Task 提交前执行 `grep -n "^go \|^toolchain" server/go.mod server/tools/go.mod`，四行必须是 `go 1.27` 和 `toolchain go1.27.1`；`git diff --stat e22e5080 -- server/go.mod server/go.sum server/tools pnpm-lock.yaml` 在本 plan 的任何时刻都没有输出。
- **每个 Task 提交前**：`make lint-go` 两段都输出 `0 issues.`，`make test` 全部通过（没有 `FAIL`）。
- **生成的文件不手写、不从本 plan 复制**：有生成物的 Task（2–10、12、13）执行 Task 中的生成命令（改了接口描述的 Task 3、4、6、8、9、12、13 执行 `make gen`，只动了 sqlc 的 Task 2、5、7、10 执行 `make gen-go`），用 `shasum -a 256` 和 `wc -l` 核对表中的 SHA-256 和行数，对不上时停下来：说明某个输入与本 plan 不一致。这些 Task **提交之后**执行 `make gen-check`（它用 `git status` 判断生成物是否已提交，提交之前必然报差异）。
- **前端检查**：改了接口描述、前端或 `e2e/` 的 Task（3、4、6、8、9、12、13、15）另执行 `make lint-web`、`make knip`、`make test-web`；声明新错误码的 Task 3（`project.archived`）在同一个 Task 里把码加进 `PROBLEM_MESSAGES` 和两份 `auth.json`（M3 设计 12 节约束 4），其余 Task 只用已有的码；Task 15 执行 `make e2e`；Task 16 只改文档，执行 `make lint-web`（关键词守卫也查文档）。
- **容器**：`make test` 和 `make e2e` 用自己的 testcontainers；机器忙时偶尔起不来，等 Docker 空闲之后重跑一次再当作失败。容器测试一次只跑一套。开发库 `nerve-dev-db-1` 可以用，但不要停止或重建它，不要执行 `make dev-db-down`、`make dev-db-reset`。不要碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`）。
- **git**：每次 Bash 调用只执行一个 git 命令，不用 `;`、`&&`、`|` 串联 git；不用 `git -C`、`stash`、`clean`、`reset --hard`。`cd` 不与别的命令组合，只读的命令也不行。不碰 `plane/`、`refer/`。
- **安装**：除了 Docker、Go、Node 不做任何全局安装；不执行 `corepack enable`（pnpm 已在 PATH 上）。
- **规则**：不写 `init()`，不用全局可变状态，构造函数显式传入依赖；不建 `utils`、`common`、`helpers` 包；一个文件只做一件事，不超过约 400 行；依赖只能向内；模块之间不互相导入，值在 `bootstrap` 中转换或直接接上（M3 设计 6.5、6.6），不跨模块的表 JOIN；模块的 SQL 只经 sqlc；角色只按集合判断（`CanAssign`、`CanAdd`、`CanJoin`、`JoinRole` 的顺序表），不按大小比较；不留没有使用者的代码（`project.NewCascade` 不建，裁定 G2）。**例外**：接口描述按模块一个文件（M0-P3 交接 5），`api/modules/project.yaml`（751 行）不受约 400 行的限制。本 plan 的其余文件都在约 400 行以内：最长的是 `bootstrap/interleaving_growth_test.go`（398 行）、`bootstrap/permission_matrix_test.go`（361 行）、`project/app/ports.go`（334 行）和 `bootstrap/workspace_deletion_test.go`（323 行）；P4a review 第 6 节点名的 `permission_matrix_seeded_test.go` 在 Task 1 拆成登记和写入（`permission_matrix_seed_test.go`），本 plan 结束时分别是 255 行和 260 行，`schema_test.go`、`directory_test.go`（396 行）、`invitations_test.go` 本 plan 不改：工作区目录按 id 的锁的存储测试另起 `directory_share_test.go`。
- **注释**：Go、TS 代码、SQL 查询和接口描述用英文；中文文档照本 plan 原样。
- **代码块**：每个改动都写成四个反引号围起来的块，块的第一行写明种类和路径，照原样使用（原型中逐字节运行过）：
  - ````` ````file <路径> ````` 新文件，块的内容加一个结尾换行就是整个文件；
  - ````` ````whole <路径> ````` 已有文件的完整新内容（同样加结尾换行）；
  - ````` ````old <路径> ````` 与紧跟着的 ````` ````new <路径> `````：`old` 的文字在文件当前版本中恰好出现一次，把它换成 `new` 的文字；
  - ````` ````delete <路径> ````` 删除这个文件（块是空的）。

  一个文件的几个块按出现的顺序依次应用。拼 plan 的脚本已从 `e22e5080` 起按顺序核对过全部块：每个 `old` 恰好出现一次（在它之前的块应用之后的文件中），每个新文件原来不存在，逐 Task 应用之后的文件与原型逐字节相同（spec 附录 A）。可以用 `node <planapply.mjs> <本 plan> apply <仓库根> <n>` 写入第 n 个 Task 的块，也可以手工照抄。
- **过渡版本**：一些文件先在较早的 Task 写成过渡版本，较晚的 Task 再修改（`api/modules/project.yaml`、`api/openapi.yaml`、`project/module.go`、`app/ports.go`、`app/lock.go`、`app/fakes_write_test.go`、`app/clock_test.go`、`domain/actions.go`、`domain/member.go`、`adapter/http/handler.go`、`handler_test.go`、`projects.go`、`members.go`、`adapter/postgres/update.go`、`members.go`、`preferences.go`、`queries/*.sql`、`failures_test.go`、`access/domain/rules.go`、矩阵的文件、`bootstrap/project_writes_test.go`、`bootstrap/project_write_locks_test.go`（每个写的 Task 加一行）、生成物）；每个过渡版本都在逐 Task 复现中运行过。
- **变异**：每个 Task 末尾的"变异"表列出：把代码改坏的方式、必须因此失败的测试和它所在的层（单元：假实现；存储：真实数据库；组合：`bootstrap` 组合出的 app 或模块；端到端：单独运行的故事）。它们在最终的原型上逐个跑过（`$M3TMP/p4btools/mutants_s*.py`、方案 E 的 `mutants_E.py`、`mutants_reanchored.py`、`mutants_split.py`，预检的 `mutants_pf*.py`，`e2e_sweep1.py`、`e2e_mutants.py`，spec 附录 A）；实现者可以照表抽查，改坏之后必须恢复。表中"（Task n 起）"标出的测试在较晚的 Task 才有：这一行的变异从那个 Task 起才被发现。**安全或加锁的性质只由单元一层发现的，算缺口**（brief 的缺陷类别）；表中每一条这类性质都另有存储、组合或端到端一层的测试，例外写在 spec 第 3 节。
- **评审敏感**（M3 设计 12 节约束 3）：项目一侧的成员关系增长（约定三、六的添加和加入）与降为访客的连带在交错 17 相遇；每个项目级的写先取工作区行的 `FOR SHARE`（约定二，方案 E），工作区一侧的每个连带因此等在工作区行上；谁能修改、归档、删除、看成员、加入项目是安全性质。改动这些测试、锁、恢复时角色的测试之前，先照"变异"表确认它在所说的性质去掉之后失败。
- **提交**：提交信息用英文，末尾加一行：`Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明。

## 文件结构

路径相对于仓库根目录。"生成"表示由命令生成并提交；一个文件由多个 Task 修改时，"Task"一列都列出。

| 文件 | 职责 | Task |
|---|---|---|
| `server/internal/bootstrap/permission_matrix_seed_test.go`；`server/internal/bootstrap/permission_matrix_seeded_test.go`（修改） | 矩阵准备的写入（两个存储的写、SQL 替身 `standIns`、`partingStates`、前提检查 `preconditions`）从表和登记中拆出；`archive` 经存储（Task 4）；账户按列登记（Task 9）；添加的目标、加入者的前提（Task 12、13） | 1、4、9、12、13 |
| `server/internal/bootstrap/permission_matrix_test.go`（修改） | `prepareMatrix` 调用拆出的写入；归档经存储；账户的 id 登记；成员的行接进矩阵 | 1、4、9、12 |
| `server/internal/modules/project/domain/patch.go`、`server/internal/modules/project/domain/patch_test.go` | `ProjectPatch`、`CheckProjectPatch`（与建项目同一规则，`archive_in` 0–12）、`CanAssign` | 2 |
| `server/internal/modules/project/domain/errors.go`（修改） | `ErrArchived`（409 `project.archived`）、`Unassignable` | 2 |
| `server/internal/modules/project/adapter/postgres/update.go`、`server/internal/modules/project/adapter/postgres/update_test.go` | `LockProject`、`UpdateProject`，`ProjectWorkspace` 的测试；`SetArchived`（Task 4）；`ShareProject`（Task 7） | 2、4、7 |
| `server/internal/modules/project/adapter/postgres/members.go`、`server/internal/modules/project/adapter/postgres/members_test.go`、`server/internal/modules/project/adapter/postgres/queries/members.sql` | `Memberships`；`ListMembers`（Task 9）；`RestoreMember`（Task 10） | 2、9、10 |
| `server/internal/modules/project/adapter/postgres/projects.go`、`server/internal/modules/project/adapter/postgres/queries/projects.sql`（修改） | 唯一键冲突的翻译 `taken` 由建项目和修改共用；`ProjectWorkspace`（不加锁读项目的工作区：每个写在加锁之前先读它）；`LockProject`、`ProjectWorkspace`、`UpdateProject`、`SetArchived`、`ShareProject` 的查询 | 2、4、7 |
| `server/internal/modules/project/adapter/postgres/failures_test.go`（修改） | 每个新方法的失败原样返回，不答成"没有" | 2、4、5、7、9、10 |
| `server/internal/modules/project/adapter/postgres/gen/projects.sql.go`、`server/internal/modules/project/adapter/postgres/gen/members.sql.go`、`server/internal/modules/project/adapter/postgres/gen/cascade.sql.go`、`server/internal/modules/project/adapter/postgres/gen/preferences.sql.go`、`server/internal/modules/workspace/adapter/postgres/gen/directory.sql.go`（生成） | | 2、4、5、7、9、10 |
| `server/internal/modules/workspace/adapter/postgres/queries/directory.sql`、`server/internal/modules/workspace/adapter/postgres/directory.go`、`server/internal/modules/workspace/module.go`（修改）；`server/internal/modules/workspace/adapter/postgres/directory_share_test.go` | 工作区目录按 id 取未删除的工作区行 `FOR SHARE`（`ShareWorkspaceByID`）：项目级的写的第一把锁 | 2 |
| `server/internal/bootstrap/ports.go`、`server/internal/bootstrap/ports_test.go`（修改） | `projectWorkspaces` 转换 `ShareWorkspaceByID` 的回答 | 2 |
| `server/internal/modules/project/app/fakes_create_test.go`（修改） | 假目录也按 id 找工作区 | 2 |
| `server/internal/modules/project/app/ports.go`（修改） | `WorkspaceDirectory` 的 `WorkspaceSharer`（按 id 取工作区行的 `FOR SHARE`）；每个用例的存储端口：`ProjectLocker`、`ProjectFinder`、`ProjectLocks`（Task 3）、`MembershipReader`、`ProjectUpdater`、`ProjectArchiver`、`Deletion`、`ProjectsDeleter`、`ProjectSharer`、`PreferencesReader`、`PreferencesWriter`、`MemberLister`、`MemberGrower`、`SortOrderReader`、`MemberAdder`、`MemberJoiner` | 2、3、4、5、7、9、10、11、13 |
| `api/modules/project.yaml`；`api/openapi.yaml`（修改） | 九个操作和它们的结构 | 3、4、6、8、9、12、13（`openapi.yaml`：4、8、9、13） |
| `api/dist/openapi.yaml`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`server/internal/modules/project/adapter/http/gen/bodyshape.gen.go`、`web/packages/api-client/src/schema.gen.ts`（生成） | | 3、4、6、8、9、12、13（`bodyshape.gen.go`：3、8、12） |
| `server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`（修改） | 规则表的九行和它们的格子 | 3、4、6、7、9、11、13 |
| `server/internal/modules/project/domain/actions.go`（修改） | 九个操作名 | 3、4、6、7、9、11、13 |
| `server/internal/modules/project/app/lock.go` | 写的唯一一条加锁路径 `Locks`（`lockAndDecide`：不加锁读项目的工作区、工作区 `FOR SHARE`、项目的锁并确认它的工作区、判定）、判定 `decide`、写的回答 `answer`；改设置的 `FOR SHARE` 和读的头两步 `findAndDecide`（Task 7）；添加、加入的目标的工作区成员行（Task 11） | 3、7、11 |
| `server/internal/modules/project/app/update_project.go`、`server/internal/modules/project/app/update_project_test.go` | `updateProject` | 3 |
| `server/internal/modules/project/app/fakes_write_test.go` | 写的假存储（按方法名失败、记下事务之外的调用）和 web、ops 两个项目；工作区的锁（`fakeWorkspaces`） | 3、4、5、7、11 |
| `server/internal/modules/project/app/clock_test.go`（修改） | 每个改已有行的写在它的锁（工作区的先）、判定、检查之后读一次时钟 | 3、4、6、7、11、13 |
| `server/internal/modules/project/adapter/http/handler.go`、`server/internal/modules/project/adapter/http/handler_test.go`、`server/internal/modules/project/adapter/http/projects.go`（修改）；`server/internal/modules/project/adapter/http/update_test.go`、`server/internal/modules/project/adapter/http/archive_test.go`、`server/internal/modules/project/adapter/http/delete_test.go`、`server/internal/modules/project/adapter/http/join_test.go` | 用例的接口；项目的 handler 和它们的测试 | 3、4、6、8、9、12、13 |
| `server/internal/modules/project/module.go`（修改） | 九个用例的接线；七个写共用的一个 `Locks` | 3、4、6、8、9、11、12、13 |
| `server/internal/bootstrap/permission_matrix_project_test.go`（修改） | 修改、归档、恢复、删除、显示设置的矩阵行 | 3、4、6、8、9、12 |
| `server/internal/bootstrap/project_writes_test.go` | 组合出的每个写盖上请求的时刻和调用者；`createdProject`（Task 4）；负责人、默认负责人是项目中不是访客的有效成员（Task 12） | 3、4、8、12 |
| `server/internal/bootstrap/project_write_locks_test.go` | 每个项目级的写先取工作区行的 `FOR SHARE`，在它的事务里、在目标和项目之前；写的行与矩阵中在项目一级写的行逐个对上（每个写的 Task 加一行） | 4、6、8、12、13 |
| `web/apps/web/helpers/authentication.helper.ts`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json`（修改） | `project.archived` 的文案 | 3 |
| `server/internal/modules/project/app/archive_project.go`、`server/internal/modules/project/app/archive_project_test.go` | `archiveProject`、`unarchiveProject` | 4 |
| `server/internal/modules/project/app/deletion.go`；`server/internal/modules/project/app/cascade.go`（完整内容）、`server/internal/modules/project/app/cascade_test.go`（修改） | 删除项目的一处步骤 `deleteProjects`，删除工作区的连带和删除项目共用（P7 在这里加标签） | 5 |
| `server/internal/modules/project/adapter/postgres/cascade.go`、`server/internal/modules/project/adapter/postgres/cascade_test.go`、`server/internal/modules/project/adapter/postgres/queries/cascade.sql`（完整内容） | 四条删除语句带可选的项目；只删一个项目时别的行一列不动 | 5 |
| `server/internal/bootstrap/workspace_deletion_catalog_test.go`（完整内容）、`server/internal/bootstrap/workspace_deletion_test.go`（修改） | 目录读取按父表（`workspaces`、`projects`）：`keysTo`，及它的反例 | 5 |
| `server/internal/modules/project/app/delete_project.go`、`server/internal/modules/project/app/delete_project_test.go` | `deleteProject` | 6 |
| `server/internal/bootstrap/project_deletion_test.go` | 组合出的删除项目对照目录；提交被拒时一行不变 | 6、14 |
| `server/internal/modules/project/domain/preferences.go`、`server/internal/modules/project/domain/preferences_test.go`（修改） | `Navigation`、`Preferences`、`PreferencesPatch`、`DefaultPreferences`、`CheckPreferencesPatch` | 7 |
| `server/internal/modules/project/adapter/postgres/preferences.go`、`server/internal/modules/project/adapter/postgres/preferences_test.go`、`server/internal/modules/project/adapter/postgres/queries/preferences.sql`（修改） | `Preferences`、`UpsertPreferences`；`EnsurePreferences`（Task 10） | 7、10 |
| `server/internal/modules/project/app/get_preferences.go`、`server/internal/modules/project/app/update_preferences.go`、`server/internal/modules/project/app/preferences_test.go`、`server/internal/modules/project/app/fakes_preferences_test.go` | 显示设置的两个用例 | 7 |
| `server/internal/modules/project/adapter/http/preferences.go`、`server/internal/modules/project/adapter/http/preferences_test.go` | 显示设置的 handler | 8 |
| `server/internal/bootstrap/project_preferences_test.go` | 组合出的显示设置各归各 | 8、12 |
| `server/internal/modules/project/domain/member.go`、`server/internal/modules/project/domain/member_test.go` | `Member`；`NewMember`、`CheckNewMembers`、`CanAdd`、`JoinRole`（Task 10）；`Target`、`CheckTargets`（Task 11）；`CanJoin`（Task 13） | 9、10、11、13 |
| `server/internal/modules/project/app/list_members.go`、`server/internal/modules/project/app/list_members_test.go` | `listProjectMembers` | 9、11 |
| `server/internal/modules/project/adapter/http/members.go`、`server/internal/modules/project/adapter/http/members_test.go`、`server/internal/modules/project/adapter/http/add_test.go` | 成员的 handler | 9、12 |
| `server/internal/modules/project/app/growth.go`、`server/internal/modules/project/app/add_members.go`、`server/internal/modules/project/app/add_members_test.go`、`server/internal/modules/project/app/fakes_growth_test.go` | 项目一侧的增长一步；`addProjectMembers` | 11 |
| `server/internal/bootstrap/permission_matrix_members_test.go` | 成员的矩阵行：列出、添加（四个无效目标）、加入 | 12、13 |
| `server/internal/modules/project/app/join_project.go`、`server/internal/modules/project/app/join_project_test.go` | `joinProject` | 13 |
| `server/internal/bootstrap/project_members_test.go` | 恢复时的角色（9.1）；增长在一个事务里（Task 14） | 13、14 |
| `server/internal/bootstrap/interleaving_growth_test.go`、`server/internal/bootstrap/interleaving_writes_test.go`、`server/internal/bootstrap/project_connection_test.go`；`server/internal/bootstrap/demotion_test.go`（修改） | 交错 17；降级与删除项目；项目级的写与降级；同一个项目上的两个写；同一个工作区里两个项目上的写不互等；每个写在事务的连接上；`refusingCommits` 按表 | 14 |
| `server/internal/bootstrap/interleaving_deletion_test.go` | 删除工作区等项目级的写提交，以自己的一个时刻删除它们写的行（预检的 L1）；添加多个成员与删除工作区串行，没有 40P01（预检的 M1） | 14 |
| `server/internal/modules/project/adapter/postgres/demote_test.go`（修改） | `LockMemberProjects` 不锁等锁期间删除的项目 | 14 |
| `e2e/fixtures/api.ts`、`e2e/fixtures/assert/project.ts`、`e2e/fixtures/assert/workspace.ts`（修改） | 加项目成员、`amidAnotherWorkspace`；`expectMember`、`expectProjectDeleted`，`expectProjectCreated` 读未删除的行；`deletedAlone` | 15 |
| `e2e/stories/project/p2-visibility.spec.ts`、`e2e/stories/project/p3-project-settings.spec.ts`、`e2e/stories/project/p4-archive.spec.ts`、`e2e/stories/project/p8-project-preferences.spec.ts`；`e2e/stories/workspace/w3-workspace-settings.spec.ts`（修改） | P2、P3、P4、P8 的接口版本；W3 删除一个项目在工作区之前 | 15 |
| `docs/v0/plane-diff.md`、`docs/v0/M3-workspace-project/handoffs/M1-P3-trim-platform.md`（修改） | 3.20 中 P4b 的一行；M1-P3 的项目成员 | 16 |

---

### Task 1: 矩阵准备的写入从登记中拆开

**Files:**
- Create: `server/internal/bootstrap/permission_matrix_seed_test.go`
- Modify: `server/internal/bootstrap/permission_matrix_seeded_test.go`、`server/internal/bootstrap/permission_matrix_test.go`

**Interfaces:** 没有新接口。`permission_matrix_seeded_test.go`（P4a 留下约 400 行）沿它的接缝拆开（P4a review 第 6 节）：表、`seeded` 的登记和 `in` 留在原文件；两个存储的写入（`matrixSeed`、`projectSeed`）、全部 SQL 替身和前提检查移进 `permission_matrix_seed_test.go`。`prepareMatrix` 里的三段内联代码各成一个方法：`standIns`（还没有存储写的状态：已归档项目，Task 4 起改经存储；以前的成员、被移出的成员，P5）、`partingStates`（原样）、`preconditions`（被移出的成员仍是项目的有效成员、`other` 的项目在自己的工作区）。`exec` 的"一行都没改就失败"的自检随 `standIns` 移动。

**Tests:** 没有新测试；矩阵、它的完整性核对和账户的核对照旧通过。

- [ ] **Step 1: 拆开**

`server/internal/bootstrap/permission_matrix_seed_test.go`（新文件，215 行）：

````file server/internal/bootstrap/permission_matrix_seed_test.go
package bootstrap

import (
	"context"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	projectpg "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	projectdomain "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The writers of prepareMatrix's rows (permission_matrix_seeded_test.go):
// the workspace store's and the project store's, the SQL that stands in
// for the stores later phases add, and the checks that the rows the cells
// rest on are there.

// matrixSeed writes the prepared workspaces, memberships and settings
// through the workspace store, and keeps the workspaces' ids by slug; exec
// runs the SQL that stands in for the stores P4b and P5 add.
type matrixSeed struct {
	t          *testing.T
	store      *workspacepg.Store
	ids        map[caller]uuid.UUID
	now        time.Time
	workspaces map[string]uuid.UUID // by slug
}

func (s matrixSeed) workspace(id uuid.UUID, slug string, admin caller) {
	s.t.Helper()
	w, err := s.store.CreateWorkspace(context.Background(), workspaceapp.WorkspaceRow{
		ID: id, Name: slug, Slug: slug, Timezone: "UTC", CreatedBy: s.ids[admin], Now: s.now,
	})
	if err != nil {
		s.t.Fatal(err)
	}
	s.workspaces[slug] = w.ID
}

func (s matrixSeed) join(id uuid.UUID, slug string, c caller, role shared.Role) {
	s.t.Helper()
	if err := s.store.CreateMember(context.Background(), workspaceapp.MemberRow{
		ID: id, WorkspaceID: s.workspaces[slug], MemberID: s.ids[c], Role: role, CreatedBy: s.ids[c], Now: s.now,
	}); err != nil {
		s.t.Fatal(err)
	}
}

func (s matrixSeed) preferences(slug string, c caller, p workspacedomain.PreferencesPatch) {
	s.t.Helper()
	if _, err := s.store.UpsertPreferences(context.Background(), workspaceapp.PreferencesRow{
		ID: uuid.NewV7(), WorkspaceID: s.workspaces[slug], UserID: s.ids[c], Patch: p, Now: s.now,
	}); err != nil {
		s.t.Fatal(err)
	}
}

// invite stores the invitation id of email to the workspace slug, by its
// admin.
func (s matrixSeed) invite(id uuid.UUID, slug, email string, role shared.Role) {
	s.t.Helper()
	if _, err := s.store.CreateInvitations(context.Background(), []workspaceapp.InvitationRow{
		{ID: id, WorkspaceID: s.workspaces[slug], Email: email, Role: role, CreatedBy: s.ids[matrixAdmins[slug]], Now: s.now},
	}); err != nil {
		s.t.Fatal(err)
	}
}

// exec runs sql for the test tb, which sql must change one row of: a
// statement that matched none would leave the seed as it was, and the
// cells that need the change would test another case.
func (s matrixSeed) exec(tb testing.TB, pool *pgxpool.Pool, sql string, args ...any) {
	tb.Helper()
	tag, err := pool.Exec(context.Background(), sql, args...)
	if err != nil {
		tb.Fatal(err)
	}
	if tag.RowsAffected() != 1 {
		tb.Fatalf("%s changed %d rows, want 1", sql, tag.RowsAffected())
	}
}

// projectSeed writes the prepared projects and their memberships through
// the project store, each by its workspace's admin, and keeps the
// projects' ids by key.
type projectSeed struct {
	matrixSeed
	store    *projectpg.Store
	projects map[string]uuid.UUID
}

// project stores the project id with key.
func (s projectSeed) project(id uuid.UUID, key, name, identifier string, network projectdomain.Network) {
	s.t.Helper()
	slug, _, _ := strings.Cut(key, "/")
	if err := s.store.CreateProject(context.Background(), projectapp.ProjectRow{
		ID: id, WorkspaceID: s.workspaces[slug], Name: name, Identifier: identifier, Network: network, Timezone: "UTC",
		CreatedBy: s.ids[matrixAdmins[slug]], Now: s.now,
	}); err != nil {
		s.t.Fatal(err)
	}
	s.projects[key] = id
}

// join makes c a member of the project key with role, and stores his
// display settings in it.
func (s projectSeed) join(key string, c caller, role shared.Role) {
	s.t.Helper()
	slug, _, _ := strings.Cut(key, "/")
	ctx, by := context.Background(), s.ids[matrixAdmins[slug]]
	if err := s.store.CreateMember(ctx, projectapp.MemberRow{
		ID: uuid.NewV7(), WorkspaceID: s.workspaces[slug], ProjectID: s.projects[key], MemberID: s.ids[c], Role: role, CreatedBy: by, Now: s.now,
	}); err != nil {
		s.t.Fatal(err)
	}
	if err := s.store.CreatePreferences(ctx, projectapp.PreferencesRow{
		ID: uuid.NewV7(), WorkspaceID: s.workspaces[slug], ProjectID: s.projects[key], UserID: s.ids[c], SortOrder: 65535, CreatedBy: by,
		Now: s.now,
	}); err != nil {
		s.t.Fatal(err)
	}
}

// partingStates puts memberships of matrixProjectMembers in the states in
// which a list and reading could part (TestListingProjectsIsReadingEach):
// WG-'s membership of acme's public project ended and of its private one
// deleted, the member's of the private one deleted, and PM's display
// settings in it deleted while his membership stays active. SQL stands in
// for the store that will end a membership (P5), and makes the two
// deleted states that only a deleted project or workspace makes today,
// which the list must still read as reading does. Each state is then read
// back: one missing would let a list that counts an ended or a deleted
// membership, or takes display settings for a membership, agree with
// reading for every account.
func (s projectSeed) partingStates(pool *pgxpool.Pool) {
	s.t.Helper()
	public, private := s.projects["acme/public"], s.projects["acme/private"]
	s.exec(s.t, pool, "UPDATE project_members SET is_active = false, updated_at = $3 WHERE project_id = $1 AND member_id = $2",
		public, s.ids[callerGuestOnly], s.now)
	for _, c := range []caller{callerGuestOnly, callerMember} {
		s.exec(s.t, pool, "UPDATE project_members SET deleted_at = $3 WHERE project_id = $1 AND member_id = $2", private, s.ids[c], s.now)
	}
	s.exec(s.t, pool, "UPDATE project_user_properties SET deleted_at = $3 WHERE project_id = $1 AND user_id = $2", private,
		s.ids[callerProjectMember], s.now)
	// A deleted membership with a live one beside it would be no deleted
	// state at all: the live one is what the list and reading would see. The
	// row stays active, as cascade.sql leaves it, so only its deleted_at
	// keeps it out.
	deleted := "m.is_active AND m.deleted_at IS NOT NULL AND NOT EXISTS (SELECT 1 FROM project_members o " +
		"WHERE o.project_id = m.project_id AND o.member_id = m.member_id AND o.deleted_at IS NULL)"
	for _, st := range []struct {
		project uuid.UUID
		c       caller
		holds   string // of m, his membership of the project
	}{
		{public, callerGuestOnly, "NOT m.is_active AND m.deleted_at IS NULL"},
		{private, callerGuestOnly, deleted},
		{private, callerMember, deleted},
		{private, callerProjectMember, "m.is_active AND m.deleted_at IS NULL AND NOT EXISTS (SELECT 1 FROM project_user_properties u " +
			"WHERE u.project_id = m.project_id AND u.user_id = m.member_id AND u.deleted_at IS NULL)"},
	} {
		var holds bool
		if err := pool.QueryRow(context.Background(), "SELECT EXISTS (SELECT 1 FROM project_members m WHERE m.project_id = $1 AND "+
			"m.member_id = $2 AND "+st.holds+")", st.project, s.ids[st.c]).Scan(&holds); err != nil || !holds {
			s.t.Fatalf("%s's membership of %s: %v, %v; want %s", st.c, st.project, holds, err, st.holds)
		}
	}
}

// standIns writes, through SQL, the states no store writes yet, until the
// phase that adds the store replaces it: acme's archived project archived
// (P4b), the member before's membership of the private project ended (P5)
// and the removed member's membership of acme ended (P5). exec fails a
// statement that changes no row, and names it, which it checks first.
func (s projectSeed) standIns(pool *pgxpool.Pool, sd seeded) {
	s.t.Helper()
	const none = "UPDATE projects SET archived_at = now() WHERE false"
	if failed, want := fatalOf(func(tb testing.TB) { s.exec(tb, pool, none) }), none+" changed 0 rows, want 1"; failed != want {
		s.t.Errorf("exec of a statement that changes no row: failed with %q, want %q", failed, want)
	}
	s.exec(s.t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", s.projects["acme/archived"], s.now)
	s.exec(s.t, pool, "UPDATE project_members SET is_active = false WHERE project_id = $1 AND member_id = $2",
		s.projects["acme/private"], s.ids[callerBefore])
	s.exec(s.t, pool, "UPDATE workspace_members SET is_active = false WHERE id = $1", sd.membership("acme", callerRemoved))
}

// preconditions checks the seeded rows that a cell's answer rests on and
// that no answer shows: one missing would let a mutant pass the cells.
func (s projectSeed) preconditions(sd seeded) {
	s.t.Helper()
	ctx := context.Background()
	// The removed member is still an active member of the project his
	// column aims at, so that only his ended membership of acme keeps him
	// out of it: his cell's 404 would not show which, were he none.
	if f, found, err := s.store.ProjectFacts(ctx, s.projects[projectOf(callerRemoved)], s.ids[callerRemoved]); err != nil || !found || !f.Member {
		s.t.Fatalf("the removed member's facts of %s = %+v, %v, %v; want him its active member", projectOf(callerRemoved), f, found, err)
	}
	// other's project is the one no list of acme's may show: were it not
	// there, undeleted in a workspace of its own, a list of every
	// workspace's projects would pass the matrix and
	// TestListingProjectsIsReadingEach alike.
	if f, found, err := s.store.ProjectFacts(ctx, s.projects["other/project"], s.ids[callerNever]); err != nil || !found ||
		f.WorkspaceID != sd.workspace("other") {
		s.t.Fatalf("other's project's facts = %+v, %v, %v; want it undeleted in other (%s), not acme (%s)", f, found, err,
			sd.workspace("other"), sd.workspace("acme"))
	}
}
````

`server/internal/bootstrap/permission_matrix_seeded_test.go`（修改，6 处）：

````old server/internal/bootstrap/permission_matrix_seeded_test.go
import (
	"context"
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
import (
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
	"testing"
	"time"
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
	"testing"
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go

	"github.com/jackc/pgx/v5/pgxpool"

	projectpg "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	projectdomain "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go

	projectdomain "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
// The rows prepareMatrix seeds, and how a row's request names them.
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
// The rows prepareMatrix seeds, and how a row's request names them; the
// writers that seed them are in permission_matrix_seed_test.go.
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go

// matrixSeed writes the prepared workspaces, memberships and settings
// through the workspace store, and keeps the workspaces' ids by slug; exec
// runs the SQL that stands in for the stores P4b and P5 add.
type matrixSeed struct {
	t          *testing.T
	store      *workspacepg.Store
	ids        map[caller]uuid.UUID
	now        time.Time
	workspaces map[string]uuid.UUID // by slug
}

func (s matrixSeed) workspace(id uuid.UUID, slug string, admin caller) {
	s.t.Helper()
	w, err := s.store.CreateWorkspace(context.Background(), workspaceapp.WorkspaceRow{
		ID: id, Name: slug, Slug: slug, Timezone: "UTC", CreatedBy: s.ids[admin], Now: s.now,
	})
	if err != nil {
		s.t.Fatal(err)
	}
	s.workspaces[slug] = w.ID
}

func (s matrixSeed) join(id uuid.UUID, slug string, c caller, role shared.Role) {
	s.t.Helper()
	if err := s.store.CreateMember(context.Background(), workspaceapp.MemberRow{
		ID: id, WorkspaceID: s.workspaces[slug], MemberID: s.ids[c], Role: role, CreatedBy: s.ids[c], Now: s.now,
	}); err != nil {
		s.t.Fatal(err)
	}
}

func (s matrixSeed) preferences(slug string, c caller, p workspacedomain.PreferencesPatch) {
	s.t.Helper()
	if _, err := s.store.UpsertPreferences(context.Background(), workspaceapp.PreferencesRow{
		ID: uuid.NewV7(), WorkspaceID: s.workspaces[slug], UserID: s.ids[c], Patch: p, Now: s.now,
	}); err != nil {
		s.t.Fatal(err)
	}
}

````
````new server/internal/bootstrap/permission_matrix_seeded_test.go

````

````old server/internal/bootstrap/permission_matrix_seeded_test.go

// invite stores the invitation id of email to the workspace slug, by its
// admin.
func (s matrixSeed) invite(id uuid.UUID, slug, email string, role shared.Role) {
	s.t.Helper()
	if _, err := s.store.CreateInvitations(context.Background(), []workspaceapp.InvitationRow{
		{ID: id, WorkspaceID: s.workspaces[slug], Email: email, Role: role, CreatedBy: s.ids[matrixAdmins[slug]], Now: s.now},
	}); err != nil {
		s.t.Fatal(err)
	}
}

// exec runs sql for the test tb, which sql must change one row of: a
// statement that matched none would leave the seed as it was, and the
// cells that need the change would test another case.
func (s matrixSeed) exec(tb testing.TB, pool *pgxpool.Pool, sql string, args ...any) {
	tb.Helper()
	tag, err := pool.Exec(context.Background(), sql, args...)
	if err != nil {
		tb.Fatal(err)
	}
	if tag.RowsAffected() != 1 {
		tb.Fatalf("%s changed %d rows, want 1", sql, tag.RowsAffected())
	}
}

// projectSeed writes the prepared projects and their memberships through
// the project store, each by its workspace's admin, and keeps the
// projects' ids by key.
type projectSeed struct {
	matrixSeed
	store    *projectpg.Store
	projects map[string]uuid.UUID
}

// project stores the project id with key.
func (s projectSeed) project(id uuid.UUID, key, name, identifier string, network projectdomain.Network) {
	s.t.Helper()
	slug, _, _ := strings.Cut(key, "/")
	if err := s.store.CreateProject(context.Background(), projectapp.ProjectRow{
		ID: id, WorkspaceID: s.workspaces[slug], Name: name, Identifier: identifier, Network: network, Timezone: "UTC",
		CreatedBy: s.ids[matrixAdmins[slug]], Now: s.now,
	}); err != nil {
		s.t.Fatal(err)
	}
	s.projects[key] = id
}

// join makes c a member of the project key with role, and stores his
// display settings in it.
func (s projectSeed) join(key string, c caller, role shared.Role) {
	s.t.Helper()
	slug, _, _ := strings.Cut(key, "/")
	ctx, by := context.Background(), s.ids[matrixAdmins[slug]]
	if err := s.store.CreateMember(ctx, projectapp.MemberRow{
		ID: uuid.NewV7(), WorkspaceID: s.workspaces[slug], ProjectID: s.projects[key], MemberID: s.ids[c], Role: role, CreatedBy: by, Now: s.now,
	}); err != nil {
		s.t.Fatal(err)
	}
	if err := s.store.CreatePreferences(ctx, projectapp.PreferencesRow{
		ID: uuid.NewV7(), WorkspaceID: s.workspaces[slug], ProjectID: s.projects[key], UserID: s.ids[c], SortOrder: 65535, CreatedBy: by,
		Now: s.now,
	}); err != nil {
		s.t.Fatal(err)
	}
}

// partingStates puts memberships of matrixProjectMembers in the states in
// which a list and reading could part (TestListingProjectsIsReadingEach):
// WG-'s membership of acme's public project ended and of its private one
// deleted, the member's of the private one deleted, and PM's display
// settings in it deleted while his membership stays active. SQL stands in
// for the store that will end a membership (P5), and makes the two
// deleted states that only a deleted project or workspace makes today,
// which the list must still read as reading does. Each state is then read
// back: one missing would let a list that counts an ended or a deleted
// membership, or takes display settings for a membership, agree with
// reading for every account.
func (s projectSeed) partingStates(pool *pgxpool.Pool) {
	s.t.Helper()
	public, private := s.projects["acme/public"], s.projects["acme/private"]
	s.exec(s.t, pool, "UPDATE project_members SET is_active = false, updated_at = $3 WHERE project_id = $1 AND member_id = $2",
		public, s.ids[callerGuestOnly], s.now)
	for _, c := range []caller{callerGuestOnly, callerMember} {
		s.exec(s.t, pool, "UPDATE project_members SET deleted_at = $3 WHERE project_id = $1 AND member_id = $2", private, s.ids[c], s.now)
	}
	s.exec(s.t, pool, "UPDATE project_user_properties SET deleted_at = $3 WHERE project_id = $1 AND user_id = $2", private,
		s.ids[callerProjectMember], s.now)
	// A deleted membership with a live one beside it would be no deleted
	// state at all: the live one is what the list and reading would see. The
	// row stays active, as cascade.sql leaves it, so only its deleted_at
	// keeps it out.
	deleted := "m.is_active AND m.deleted_at IS NOT NULL AND NOT EXISTS (SELECT 1 FROM project_members o " +
		"WHERE o.project_id = m.project_id AND o.member_id = m.member_id AND o.deleted_at IS NULL)"
	for _, st := range []struct {
		project uuid.UUID
		c       caller
		holds   string // of m, his membership of the project
	}{
		{public, callerGuestOnly, "NOT m.is_active AND m.deleted_at IS NULL"},
		{private, callerGuestOnly, deleted},
		{private, callerMember, deleted},
		{private, callerProjectMember, "m.is_active AND m.deleted_at IS NULL AND NOT EXISTS (SELECT 1 FROM project_user_properties u " +
			"WHERE u.project_id = m.project_id AND u.user_id = m.member_id AND u.deleted_at IS NULL)"},
	} {
		var holds bool
		if err := pool.QueryRow(context.Background(), "SELECT EXISTS (SELECT 1 FROM project_members m WHERE m.project_id = $1 AND "+
			"m.member_id = $2 AND "+st.holds+")", st.project, s.ids[st.c]).Scan(&holds); err != nil || !holds {
			s.t.Fatalf("%s's membership of %s: %v, %v; want %s", st.c, st.project, holds, err, st.holds)
		}
	}
}

````
````new server/internal/bootstrap/permission_matrix_seeded_test.go

````

`server/internal/bootstrap/permission_matrix_test.go`（修改，4 处）：

````old server/internal/bootstrap/permission_matrix_test.go
// and matrixProjectMembers. Through the API, gone deleted by its admin,
// which soft-deletes its memberships and its project with it. Through SQL,
// until the stores of P4b and P5 replace it, acme's archived project
// archived, the member before's membership of the private project ended
// and the removed member's membership of acme ended; and partingStates'
// ended and deleted project memberships and deleted display settings.
````
````new server/internal/bootstrap/permission_matrix_test.go
// and matrixProjectMembers. Through SQL, the states no store writes yet
// (standIns, partingStates); then the checks that the rows the cells rest
// on are there (preconditions). Through the API, gone deleted by its admin,
// which soft-deletes its memberships and its project with it.
````

````old server/internal/bootstrap/permission_matrix_test.go
		// No store archives a project (P4b), ends a project membership (P5)
		// or removes a member (P5) yet, so SQL stands in until those phases
		// replace it.
		seed.exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", s.project("acme/archived"), seed.now)
		seed.exec(t, pool, "UPDATE project_members SET is_active = false WHERE project_id = $1 AND member_id = $2",
			s.project("acme/private"), ids[callerBefore])
		seed.exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE id = $1", s.membership("acme", callerRemoved))
		// exec fails a statement that changes no row, and names it.
		const none = "UPDATE projects SET archived_at = now() WHERE false"
		if failed, want := fatalOf(func(tb testing.TB) { seed.exec(tb, pool, none) }), none+" changed 0 rows, want 1"; failed != want {
			t.Errorf("exec of a statement that changes no row: failed with %q, want %q", failed, want)
		}
````
````new server/internal/bootstrap/permission_matrix_test.go
		projects.standIns(pool, s)
````

````old server/internal/bootstrap/permission_matrix_test.go
		// The removed member is still an active member of the project his
		// column aims at, so that only his ended membership of acme keeps him
		// out of it: his cell's 404 would not show which, were he none.
		if f, found, err := projects.store.ProjectFacts(context.Background(), s.project(projectOf(callerRemoved)), ids[callerRemoved]); err != nil ||
			!found || !f.Member {
			t.Fatalf("the removed member's facts of %s = %+v, %v, %v; want him its active member", projectOf(callerRemoved), f, found, err)
		}
````
````new server/internal/bootstrap/permission_matrix_test.go
		projects.preconditions(s)
````

````old server/internal/bootstrap/permission_matrix_test.go
			t.Fatalf("deleting gone = %d %s", status, body)
		}
		// other's project is the one no list of acme's may show: were it not
		// there, undeleted in a workspace of its own, a list of every
		// workspace's projects would pass the matrix and
		// TestListingProjectsIsReadingEach alike.
		if f, found, err := projects.store.ProjectFacts(context.Background(), s.project("other/project"), ids[callerNever]); err != nil ||
			!found || f.WorkspaceID != s.workspace("other") {
			t.Fatalf("other's project's facts = %+v, %v, %v; want it undeleted in other (%s), not acme (%s)", f, found, err,
				s.workspace("other"), s.workspace("acme"))
````
````new server/internal/bootstrap/permission_matrix_test.go
			t.Fatalf("deleting gone = %d %s", status, body)
````

- [ ] **Step 2: 测试、lint**

Run: `go -C server test -count=1 -run 'TestPermissionMatrix$|TestThePermissionMatrixCoversEveryOperation|TestMatrixViolationsCatchesEach|TestEveryColumnCallsAsARegisteredAccount' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 3: 提交**

```bash
git add server/internal/bootstrap/permission_matrix_seed_test.go server/internal/bootstrap/permission_matrix_seeded_test.go server/internal/bootstrap/permission_matrix_test.go
```
```bash
git commit -m "refactor(M3/P4b): split the matrix's seed writers from its registry

permission_matrix_seeded_test.go (about 400 lines, P4a review 6) keeps the
tables, the seeded registry and its lookups; the workspace and project
store writers, the SQL that stands in for later stores and the checks of
the rows the cells rest on move to permission_matrix_seed_test.go, each
inline block of prepareMatrix a method of its own. Only moved.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**变异**：只移动代码，没有要变异的性质；`make test` 通过即可。前提检查的变异在 Task 12、14（清扫 10）。

**Done when:** 两个文件都在约 400 行以内（240 行、215 行），矩阵的每一格照旧通过。

---

### Task 2: 修改项目的领域与存储；项目锁；成员关系的读；工作区按 id 的锁

**Files:**
- Create: `server/internal/modules/project/adapter/postgres/members.go`、`server/internal/modules/project/adapter/postgres/members_test.go`、`server/internal/modules/project/adapter/postgres/update.go`、`server/internal/modules/project/adapter/postgres/update_test.go`、`server/internal/modules/project/domain/patch.go`、`server/internal/modules/project/domain/patch_test.go`、`server/internal/modules/workspace/adapter/postgres/directory_share_test.go`
- Modify: `server/internal/bootstrap/ports.go`、`server/internal/bootstrap/ports_test.go`、`server/internal/modules/project/adapter/postgres/failures_test.go`、`server/internal/modules/project/adapter/postgres/projects.go`、`server/internal/modules/project/adapter/postgres/queries/projects.sql`、`server/internal/modules/project/app/fakes_create_test.go`、`server/internal/modules/project/app/ports.go`、`server/internal/modules/project/domain/errors.go`、`server/internal/modules/workspace/adapter/postgres/directory.go`、`server/internal/modules/workspace/adapter/postgres/queries/directory.sql`、`server/internal/modules/workspace/module.go`
- Modify（完整内容）: `server/internal/modules/project/adapter/postgres/queries/members.sql`
- Generate: `server/internal/modules/project/adapter/postgres/gen/members.sql.go`、`server/internal/modules/project/adapter/postgres/gen/projects.sql.go`、`server/internal/modules/workspace/adapter/postgres/gen/directory.sql.go`

**Interfaces:**
- Produces（spec 2.3，M3 设计 3.6 约定二、3.19、5.2）：`domain.ProjectPatch`（每个可写的字段一个指针，负责人、默认负责人各带一个 `Set…` 标志，`null` 清空）；`domain.CheckProjectPatch(ProjectPatch) (ProjectPatch, error)`：给出的字段照 `CheckNewProject` 的规则逐字段核对（名称、标识转大写、说明、网络、时区、图标），`archive_in` 0–`MaxArchiveIn`（12），全部问题一个 422；`domain.CanAssign(role)`：项目的管理员、成员，按集合；`domain.ErrArchived`（409 `project.archived`）；`domain.Unassignable(field)`。
- `app.LockedProject{WorkspaceID, Archived}`；`app.ProjectLocker.LockProject(ctx, id) (LockedProject, found, error)`：`FOR NO KEY UPDATE`，带 `deleted_at IS NULL`，等锁之后重新求值；`app.ProjectFinder.ProjectWorkspace(ctx, id) (workspaceID, found, error)`：未删除的项目的工作区，不加锁（每个写在加锁之前先读它，M3 设计 3.6 约定二；Task 7、9 的读也用它）；`app.Membership{ID, Role, Active}`、`app.MembershipReader.Memberships(ctx, projectID, userIDs)`：未删除的成员关系（有效、已结束），按账户；`app.ProjectUpdater`。
- 存储：`LockProject`、`UpdateProject`（没给的字段 `coalesce` 保留，负责人、默认负责人按标志写，`updated_by_id`、`updated_at`）、`Memberships`；建项目和修改共用 `taken`：两个部分唯一键的冲突是 `ErrIdentifierTaken`、`ErrNameTaken`，别的错误原样包装。
- 工作区按 id 的锁（方案 E，M3 设计 3.6 约定二、6.5）：`workspace` 的查询 `ShareDirectoryWorkspaceByID`（`WHERE id = $1 AND deleted_at IS NULL FOR SHARE`，等锁之后重新求值）、`Directory.ShareWorkspaceByID(ctx, id) (DirectoryEntry, found, error)`，与 P4a 的 `ShareWorkspaceBySlug` 并列；`app.WorkspaceDirectory` 嵌入 `app.WorkspaceSharer.ShareWorkspaceByID(ctx, id) (Workspace, found, error)`；`bootstrap` 的 `projectWorkspaces` 照旧转换回答。`workspace` 模块只多这条查询和它的方法。
- 使用者：Task 3 的 `Locks`（`ProjectWorkspace` → `ShareWorkspaceByID` → `LockProject`）；`LockProject` 由 Task 4、6、11、13 的写共用，`Memberships` 由 Task 11、13 共用。

**Tests:**
- `domain/patch_test.go`：`TestCheckProjectPatchAcceptsValidPatches`（空的；`çay1` 存成 `ÇAY1`、`archive_in` 0；每个字段都给、设负责人、清空默认负责人、`archive_in` 12 的：标识转大写，别的照原样）；`TestCheckProjectPatchReportsEveryField`（`archive_in` −1、13、120 各一个 `out_of_range`；七个字段同时出错按字段顺序一个 422）；`TestThePatchChecksAsCreateDoes`（名称、标识、时区、网络、说明、图标的 33 个值，交给 `CheckNewProject` 和 `CheckProjectPatch` 得到同样的字段错误，或都通过：修改与建项目同一规则，M3 设计 3.19）；`TestCanAssign`（20、15 可以；5、0、10、16、25 不可以）。
- `adapter/postgres/update_test.go`：`TestLockProject`（未删除的项目读到工作区和是否归档，已删除的、不存在的找不到；持锁时 Web 的 `FOR SHARE` 等待、`FOR KEY SHARE` 不等，别的项目不被锁；事务结束后放开）；`TestTheProjectLockSeesADeletionItWaitedFor`（另一个事务软删除 Web 未提交，`LockProject` 等它（`pgtest.WaitForLockWaitOn`），提交之后找不到）；`TestUpdateProject`（每个字段都有一次没给而保留与默认、与别的项目都不同的值；空的修改只改审计列；清空负责人、默认负责人；别的项目每一列不变）；`TestUpdateProjectIdentifierOrNameTaken`（同工作区未删除的项目（已归档的也算）的标识、名称各答自己的 409，什么都不变；自己的、别的工作区的、已删除的可用）；`TestUpdateProjectBreakingAnotherConstraintIsInternal`（`archive_in` 13 是 `projects_archive_in_check` 的违反，不是领域错误）。
- `adapter/postgres/members_test.go`：`TestMemberships`（问到的账户各得他在这个项目未删除的成员关系，有效的、已结束的；不是别的项目的、不是已删除的、不是没问到的账户的；行序让两种物理顺序都不能互相顶替）。
- `adapter/postgres/update_test.go` 另有 `TestProjectWorkspace`（未删除的项目的工作区，已归档的也找到；已删除的、不存在的找不到；读的事务开着时 Web 的 `FOR UPDATE` 不等：不加锁）。
- `failures_test.go`：`LockProject`、`ProjectWorkspace`、`Memberships`、`UpdateProject` 的失败是 `context.Canceled`，不答成"没有"、不答成名称或标识被占。
- `workspace/adapter/postgres/directory_share_test.go`（新文件：`directory_test.go` 已 396 行）：`TestShareWorkspaceByIDFindsTheUndeletedWorkspace`（两个工作区各得自己的 id 和时区，已删除的、不存在的找不到；失败是错误，不是"没有"）；`TestTheDirectorysLockByIDIsForShare`（持锁时工作区的 `FOR NO KEY UPDATE` 等待、`FOR SHARE` 不等，别的工作区不被锁；它自己等 `FOR NO KEY UPDATE`）；`TestTheDirectorysLockByIDSeesADeletionItWaitedFor`。
- `bootstrap/ports_test.go`：`TestProjectWorkspacesConvertsWorkspacesAnswer` 加"按 id 取锁"一路：转换后的工作区、`found` 和错误照原样。

- [ ] **Step 1: 领域**

`server/internal/modules/project/domain/patch.go`（新文件，85 行）：

````file server/internal/modules/project/domain/patch.go
package domain

import (
	"fmt"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ProjectPatch is what updateProject changes (M3 design 5.2): every field
// of the project the caller may write, each nil when it is not given and
// keeps its value. The lead and the default assignee can be cleared, so
// each has a flag: set, it changes to the id, or to none when the id is
// nil.
type ProjectPatch struct {
	Name                 *string
	Description          *string
	Identifier           *string
	Network              *Network
	SetLead              bool
	LeadID               *uuid.UUID
	SetDefaultAssignee   bool
	DefaultAssigneeID    *uuid.UUID
	CycleView            *bool
	ModuleView           *bool
	IssueViewsView       *bool
	IntakeView           *bool
	GuestViewAllFeatures *bool
	ArchiveIn            *int
	LogoProps            *LogoProps
	Timezone             *string
}

// MaxArchiveIn is the most months archive_in takes: Plane's validators,
// which the column's CHECK holds too (M3 design 4.6).
const MaxArchiveIn = 12

// CheckProjectPatch checks the fields p gives, by createProject's rules
// (CheckNewProject), and archive_in from 0 to MaxArchiveIn; it returns p as
// it is stored, the identifier in upper case. Every field with a problem is
// reported at once, in one 422 validation_failed. Whether the lead and the
// default assignee may be them is the use case's, under its lock (M3 design
// 3.19).
func CheckProjectPatch(p ProjectPatch) (ProjectPatch, error) {
	var found []*shared.FieldError
	if p.Name != nil {
		found = append(found, checkName(*p.Name))
	}
	if p.Identifier != nil {
		identifier := Identifier(*p.Identifier)
		p.Identifier = &identifier
		found = append(found, checkIdentifier(identifier))
	}
	if p.Description != nil {
		found = append(found, checkText("description", *p.Description))
	}
	if p.Network != nil {
		found = append(found, checkNetwork(*p.Network))
	}
	if p.ArchiveIn != nil && (*p.ArchiveIn < 0 || *p.ArchiveIn > MaxArchiveIn) {
		found = append(found, &shared.FieldError{Field: "archive_in", Code: shared.FieldOutOfRange,
			Message: fmt.Sprintf("must be between 0 and %d", MaxArchiveIn)})
	}
	found = append(found, checkTimezone(p.Timezone))
	if p.LogoProps != nil {
		found = append(found, checkLogoProps(*p.LogoProps)...)
	}
	if err := invalid(found...); err != nil {
		return ProjectPatch{}, err
	}
	return p, nil
}

// assignableRoles are the project roles of the members a project's lead and
// default assignee are chosen from on update: not its guests (Plane
// core/components/project/member-select.tsx:32-42; M3 design 3.19).
var assignableRoles = []shared.Role{shared.RoleAdmin, shared.RoleMember}

// CanAssign reports whether an active member of a project, of project role
// role, may be its lead or its default assignee. The set is named, not a
// bound.
func CanAssign(role shared.Role) bool {
	return slices.Contains(assignableRoles, role)
}
````

`server/internal/modules/project/domain/patch_test.go`（新文件，143 行）：

````file server/internal/modules/project/domain/patch_test.go
package domain

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fieldsOf are the field errors of err, a 422 validation_failed, or nil.
func fieldsOf(t *testing.T, err error) []shared.FieldError {
	t.Helper()
	if err == nil {
		return nil
	}
	var e *shared.Error
	if !errors.As(err, &e) || e.Code != shared.CodeValidationFailed {
		t.Fatalf("%v is not a validation_failed", err)
	}
	return e.Fields
}

// CheckProjectPatch accepts these and returns each as it is stored: the
// identifier in upper case, every other field as given, those not given
// nil; a lead or a default assignee cleared stays cleared.
func TestCheckProjectPatchAcceptsValidPatches(t *testing.T) {
	lead := uuid.MustParse("0199a2b4-0000-7000-8000-000000000001")
	tests := []struct {
		p          ProjectPatch
		identifier *string
	}{
		{ProjectPatch{}, nil},
		{ProjectPatch{Identifier: ptr("çay1"), ArchiveIn: ptr(0)}, ptr("ÇAY1")},
		{ProjectPatch{Name: ptr("研发 Web"), Description: ptr("多行\n说明"), Network: ptr(NetworkPrivate), SetLead: true, LeadID: &lead,
			SetDefaultAssignee: true, CycleView: ptr(true), ModuleView: ptr(false), IssueViewsView: ptr(true), IntakeView: ptr(true),
			GuestViewAllFeatures: ptr(true), ArchiveIn: ptr(12), LogoProps: &LogoProps{InUse: ptr(LogoEmoji)}, Timezone: ptr("Asia/Shanghai")}, nil},
	}
	for _, tt := range tests {
		got, err := CheckProjectPatch(tt.p)
		want := tt.p
		if tt.identifier != nil {
			want.Identifier = tt.identifier
		}
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("CheckProjectPatch(%+v) = %+v, %v; want %+v", tt.p, got, err, want)
		}
	}
}

// archive_in takes 0 to 12 months, Plane's validators (M3 design 4.6); and
// every field with a problem is reported at once, in the order of the
// fields.
func TestCheckProjectPatchReportsEveryField(t *testing.T) {
	outOfRange := shared.FieldError{Field: "archive_in", Code: "out_of_range", Message: "must be between 0 and 12"}
	for _, months := range []int{-1, 13, 120} {
		if got := fieldsOf(t, func() error { _, err := CheckProjectPatch(ProjectPatch{ArchiveIn: &months}); return err }()); !slices.Equal(got,
			[]shared.FieldError{outOfRange}) {
			t.Errorf("archive_in %d: %+v, want %+v", months, got, outOfRange)
		}
	}
	const nul = "must not contain a NUL character"
	all := ProjectPatch{Name: ptr(""), Identifier: ptr("web-2"), Description: ptr("\x00"), Network: ptr(Network(1)), ArchiveIn: ptr(13),
		Timezone: ptr("Mars/Olympus"), LogoProps: &LogoProps{InUse: ptr("x")}}
	want := []shared.FieldError{
		{Field: "name", Code: "too_short", Message: "must not be empty"},
		{Field: "identifier", Code: "invalid_format", Message: "may hold only A-Z, 0-9 and ÇŞĞİÖÜ"},
		{Field: "description", Code: "invalid_format", Message: nul},
		{Field: "network", Code: "invalid_format", Message: "must be 0 (private) or 2 (public)"},
		outOfRange,
		{Field: "timezone", Code: "invalid_format", Message: "is not a known time zone"},
		{Field: "logo_props.in_use", Code: "invalid_format", Message: "must be emoji or icon"},
	}
	_, err := CheckProjectPatch(all)
	if got := fieldsOf(t, err); !slices.Equal(got, want) {
		t.Errorf("all at once: %+v, want %+v", got, want)
	}
}

// A field a patch gives is held to createProject's rule, problem for
// problem (M3 design 3.19): each value below, given to CheckNewProject in
// an otherwise valid project and to CheckProjectPatch alone, is refused
// with the same field errors, or accepted by both.
func TestThePatchChecksAsCreateDoes(t *testing.T) {
	valid := NewProject{Name: "Web", Identifier: "WEB"}
	type pair struct {
		create NewProject
		patch  ProjectPatch
	}
	var pairs []pair
	for _, name := range []string{"", " ", strings.Repeat("项", 255), strings.Repeat("项", 256), "W\x00", "Web-2", "Web.2", "研发 Web"} {
		create := valid
		create.Name = name
		pairs = append(pairs, pair{create, ProjectPatch{Name: &name}})
	}
	for _, id := range []string{"", "web", "çay1", "ABCDEFGHIJ", "ABCDEFGHIJK", "WEB-2", "WEB 2", "WEB\n", "ÉQUIPE"} {
		create := valid
		create.Identifier = id
		pairs = append(pairs, pair{create, ProjectPatch{Identifier: &id}})
	}
	for _, zone := range []string{"", "Local", "UTC", "Asia/Shanghai", "Mars/Olympus"} {
		create := valid
		create.Timezone = &zone
		pairs = append(pairs, pair{create, ProjectPatch{Timezone: &zone}})
	}
	for _, n := range []Network{0, 1, 2, 3} {
		create := valid
		create.Network = &n
		pairs = append(pairs, pair{create, ProjectPatch{Network: &n}})
	}
	for _, d := range []string{"", "a\x00", "多行\n说明"} {
		create := valid
		create.Description = d
		pairs = append(pairs, pair{create, ProjectPatch{Description: &d}})
	}
	for _, l := range []LogoProps{{}, {InUse: ptr("other")}, {Emoji: &Emoji{URL: ptr("\x00")}}, {Icon: &Icon{Name: ptr("home")}}} {
		create := valid
		create.LogoProps = l
		pairs = append(pairs, pair{create, ProjectPatch{LogoProps: &l}})
	}
	for _, p := range pairs {
		_, createErr := CheckNewProject(p.create)
		_, patchErr := CheckProjectPatch(p.patch)
		if c, u := fieldsOf(t, createErr), fieldsOf(t, patchErr); !slices.Equal(c, u) {
			t.Errorf("%+v: create %+v, update %+v; want the same", p.patch, c, u)
		}
	}
}

// Only a project's admins and members may be its lead or its default
// assignee on update, not its guests nor a role outside the three (M3
// design 3.19).
func TestCanAssign(t *testing.T) {
	for role, want := range map[shared.Role]bool{shared.RoleAdmin: true, shared.RoleMember: true, shared.RoleGuest: false, 0: false, 10: false,
		16: false, 25: false} {
		if got := CanAssign(role); got != want {
			t.Errorf("CanAssign(%d) = %v, want %v", role, got, want)
		}
	}
}
````

`server/internal/modules/project/domain/errors.go`（修改，2 处）：

````old server/internal/modules/project/domain/errors.go
	ErrNotFound = shared.NewError(shared.KindNotFound, "project.not_found", "The project does not exist, or you cannot see it.")
````
````new server/internal/modules/project/domain/errors.go
	ErrNotFound = shared.NewError(shared.KindNotFound, "project.not_found", "The project does not exist, or you cannot see it.")
	// ErrArchived answers a change of an archived project (M3 design 3.19).
	ErrArchived = shared.NewError(shared.KindConflict, "project.archived", "The project is archived; unarchive it to change it.")
````

````old server/internal/modules/project/domain/errors.go
}

````
````new server/internal/modules/project/domain/errors.go
}

// Unassignable is the problem of field, the lead or the default assignee
// of an update, naming an account that is not an active member of the
// project, or is its guest (M3 design 3.19).
func Unassignable(field string) shared.FieldError {
	return shared.FieldError{Field: field, Code: shared.FieldNotAllowed, Message: "must be an active member of the project who is not its guest"}
}

````

- [ ] **Step 2: 查询**

`server/internal/modules/project/adapter/postgres/queries/members.sql`（完整内容，11 行）：

````whole server/internal/modules/project/adapter/postgres/queries/members.sql
-- name: CreateMember :exec
INSERT INTO project_members (id, workspace_id, project_id, member_id, role, created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(member_id), sqlc.arg(role), sqlc.arg(created_by),
        sqlc.arg(created_by), sqlc.arg(now), sqlc.arg(now));

-- name: Memberships :many
-- The accounts' undeleted memberships of the project, active and ended (M3 design 3.5, 3.19): under the project's
-- FOR NO KEY UPDATE, which every change of its memberships takes, they stay as read until the transaction ends.
SELECT id, member_id, role, is_active
FROM project_members
WHERE project_id = sqlc.arg(project_id) AND member_id = ANY (sqlc.arg(member_ids)::uuid[]) AND deleted_at IS NULL;
````

`server/internal/modules/project/adapter/postgres/queries/projects.sql`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/queries/projects.sql
ORDER BY u.sort_order NULLS LAST, p.name;

````
````new server/internal/modules/project/adapter/postgres/queries/projects.sql
ORDER BY u.sort_order NULLS LAST, p.name;

-- name: LockProject :one
-- The parent lock of a write that changes the project row or its memberships (M3 design 3.6 convention 2): FOR NO KEY
-- UPDATE waits for another FOR NO KEY UPDATE and for FOR SHARE, not for a foreign key's FOR KEY SHARE. After a wait,
-- Postgres evaluates deleted_at IS NULL again on the row's newest version, so a project deleted meanwhile reads no row.
SELECT workspace_id, (archived_at IS NOT NULL)::boolean AS archived
FROM projects
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
FOR NO KEY UPDATE;

-- name: ProjectWorkspace :one
-- The workspace of the undeleted project, archived or not, without a lock: what a read decides on (M3 design 6.4), and
-- what a write on the project reads first, to lock the workspace before the project (3.6 convention 2).
SELECT workspace_id
FROM projects
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: UpdateProject :exec
-- updateProject, under the project's FOR NO KEY UPDATE (M3 design 5.2): a field left out, null here, keeps its value;
-- the lead and the default assignee change when their flags are set, to null too.
UPDATE projects p
SET name                    = coalesce(sqlc.narg(name)::text, p.name),
    description             = coalesce(sqlc.narg(description)::text, p.description),
    identifier              = coalesce(sqlc.narg(identifier)::text, p.identifier),
    network                 = coalesce(sqlc.narg(network)::smallint, p.network),
    project_lead_id         = CASE WHEN sqlc.arg(set_lead)::boolean THEN sqlc.narg(project_lead_id)::uuid
                                   ELSE p.project_lead_id END,
    default_assignee_id     = CASE WHEN sqlc.arg(set_default_assignee)::boolean THEN sqlc.narg(default_assignee_id)::uuid
                                   ELSE p.default_assignee_id END,
    cycle_view              = coalesce(sqlc.narg(cycle_view)::boolean, p.cycle_view),
    module_view             = coalesce(sqlc.narg(module_view)::boolean, p.module_view),
    issue_views_view        = coalesce(sqlc.narg(issue_views_view)::boolean, p.issue_views_view),
    intake_view             = coalesce(sqlc.narg(intake_view)::boolean, p.intake_view),
    guest_view_all_features = coalesce(sqlc.narg(guest_view_all_features)::boolean, p.guest_view_all_features),
    archive_in              = coalesce(sqlc.narg(archive_in)::integer, p.archive_in),
    logo_props              = coalesce(sqlc.narg(logo_props)::jsonb, p.logo_props),
    timezone                = coalesce(sqlc.narg(timezone)::text, p.timezone),
    updated_by_id           = sqlc.arg(updated_by)::uuid,
    updated_at              = sqlc.arg(now)
WHERE p.id = sqlc.arg(id);

````

`server/internal/modules/workspace/adapter/postgres/queries/directory.sql`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/queries/directory.sql
FOR SHARE;

-- name: ShareMembers :many
````
````new server/internal/modules/workspace/adapter/postgres/queries/directory.sql
FOR SHARE;

-- name: ShareDirectoryWorkspaceByID :one
-- WorkspaceDirectory's lock by id: the first lock of every write on a project of the workspace (M3 design 3.6
-- convention 2), as ShareWorkspaceByID takes it. FOR SHARE waits for the workspace's FOR NO KEY UPDATE, under which
-- every cascade over its projects runs, and makes it wait; it does not wait for another write on a project. After a
-- wait, Postgres evaluates deleted_at IS NULL again on the row's newest version, so a workspace deleted meanwhile
-- reads no row.
SELECT id, timezone
FROM workspaces
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
FOR SHARE;

-- name: ShareMembers :many
````

- [ ] **Step 3: 生成**

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `647dff7a51d6e3bc1bb864cccb913010b9beb7102b53bae27eb5bd3a44a6545d` | 87 | `server/internal/modules/project/adapter/postgres/gen/members.sql.go` |
| `00d39cf9fa264e66922356d252d4069de91a91e5e4a312687e01e983229417bc` | 363 | `server/internal/modules/project/adapter/postgres/gen/projects.sql.go` |
| `bb1fe74df1c3ea4454eced2ec7d1218c596dc8a4e8f2cdb11b92ab1c919c13ff` | 119 | `server/internal/modules/workspace/adapter/postgres/gen/directory.sql.go` |

Run: `shasum -a 256 server/internal/modules/project/adapter/postgres/gen/members.sql.go server/internal/modules/project/adapter/postgres/gen/projects.sql.go server/internal/modules/workspace/adapter/postgres/gen/directory.sql.go`
Expected: 与上表相同。

- [ ] **Step 4: 端口和存储**

`server/internal/modules/project/app/ports.go`（修改，3 处）：

````old server/internal/modules/project/app/ports.go
// (workspace.Provide): the undeleted workspace a slug names; found is false
// when there is none (M3 design 6.5).
````
````new server/internal/modules/project/app/ports.go
// (workspace.Provide): the undeleted workspace a slug names, or an id
// names; found is false when there is none (M3 design 6.5).
````

````old server/internal/modules/project/app/ports.go
	ShareWorkspaceBySlug(ctx context.Context, slug string) (w Workspace, found bool, err error)
````
````new server/internal/modules/project/app/ports.go
	ShareWorkspaceBySlug(ctx context.Context, slug string) (w Workspace, found bool, err error)
	WorkspaceSharer
}

// WorkspaceSharer takes the first lock of a write on a project (M3 design
// 3.6 convention 2).
type WorkspaceSharer interface {
	// ShareWorkspaceByID locks the undeleted workspace id's row FOR SHARE
	// until the transaction ctx carries ends. Every cascade over the
	// workspace's projects runs under the row's FOR NO KEY UPDATE, which
	// this waits for and holds off; another write on a project of the
	// workspace does not wait for it. A workspace deleted while the lock
	// waited is not found.
	ShareWorkspaceByID(ctx context.Context, id uuid.UUID) (w Workspace, found bool, err error)
````

````old server/internal/modules/project/app/ports.go
	CreateStates(ctx context.Context, rows []StateRow) error
````
````new server/internal/modules/project/app/ports.go
	CreateStates(ctx context.Context, rows []StateRow) error
}

// LockedProject is a project as its lock reads it.
type LockedProject struct {
	WorkspaceID uuid.UUID
	Archived    bool
}

// ProjectLocker takes the parent lock of a write on a project (M3 design
// 3.6 convention 2).
type ProjectLocker interface {
	// LockProject locks the undeleted project id FOR NO KEY UPDATE until the
	// transaction ctx carries ends; found is false when there is none, a
	// project deleted while the lock waited too.
	LockProject(ctx context.Context, id uuid.UUID) (p LockedProject, found bool, err error)
}

// ProjectFinder finds the workspace of a project: what a read decides on,
// and what a write on the project reads first, to lock the workspace
// before the project (M3 design 3.6 convention 2).
type ProjectFinder interface {
	// ProjectWorkspace is the workspace of the undeleted project id, read
	// without a lock; found is false when there is none.
	ProjectWorkspace(ctx context.Context, id uuid.UUID) (workspaceID uuid.UUID, found bool, err error)
}

// Membership is an account's undeleted membership of a project, active or
// ended.
type Membership struct {
	ID     uuid.UUID
	Role   shared.Role
	Active bool
}

// MembershipReader reads an account's membership of a project.
type MembershipReader interface {
	// Memberships are userIDs' undeleted memberships of projectID, active or
	// ended, by account; an account without one is not in it. Under the
	// project's FOR NO KEY UPDATE they stay as read.
	Memberships(ctx context.Context, projectID uuid.UUID, userIDs []uuid.UUID) (map[uuid.UUID]Membership, error)
}

// ProjectUpdater is updateProject's repository. Each method runs in the
// transaction ctx carries.
type ProjectUpdater interface {
	ProjectReader
	ProjectLocker
	MembershipReader
	// UpdateProject changes the fields p gives of the project id, by the
	// account by at now. An identifier or a name another undeleted project of
	// the workspace has is domain.ErrIdentifierTaken or domain.ErrNameTaken.
	UpdateProject(ctx context.Context, id uuid.UUID, p domain.ProjectPatch, by uuid.UUID, now time.Time) error
````

`server/internal/modules/project/adapter/postgres/projects.go`（修改，3 处）：

````old server/internal/modules/project/adapter/postgres/projects.go
	})
````
````new server/internal/modules/project/adapter/postgres/projects.go
	})
	return taken("create project", err)
}

// taken is err, a write's of a project, as an identifier or a name another
// undeleted project of the workspace has: domain.ErrIdentifierTaken or
// domain.ErrNameTaken; any other error is the write's, nil is nil.
func taken(write string, err error) error {
````

````old server/internal/modules/project/adapter/postgres/projects.go
	case err != nil:
		return fmt.Errorf("create project: %w", err)
````
````new server/internal/modules/project/adapter/postgres/projects.go
	case err != nil:
		return fmt.Errorf("%s: %w", write, err)
````

````old server/internal/modules/project/adapter/postgres/projects.go
	return taken, nil
````
````new server/internal/modules/project/adapter/postgres/projects.go
	return taken, nil
}

// ProjectWorkspace is the workspace of the undeleted project id, archived
// or not, read without a lock; found is false when there is none
// (app.ProjectFinder).
func (s *Store) ProjectWorkspace(ctx context.Context, id uuid.UUID) (workspaceID uuid.UUID, found bool, err error) {
	workspaceID, err = s.queries(ctx).ProjectWorkspace(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return uuid.UUID{}, false, nil
	case err != nil:
		return uuid.UUID{}, false, fmt.Errorf("find project %s: %w", id, err)
	}
	return workspaceID, true, nil
````

`server/internal/modules/project/adapter/postgres/update.go`（新文件，60 行）：

````file server/internal/modules/project/adapter/postgres/update.go
package postgresadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// LockProject locks the undeleted project id FOR NO KEY UPDATE until the
// transaction ctx carries ends, and reads its workspace and whether it is
// archived; found is false when there is none, a project deleted while the
// lock waited too (app.ProjectLocker).
func (s *Store) LockProject(ctx context.Context, id uuid.UUID) (p app.LockedProject, found bool, err error) {
	r, err := s.queries(ctx).LockProject(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return app.LockedProject{}, false, nil
	case err != nil:
		return app.LockedProject{}, false, fmt.Errorf("lock project %s: %w", id, err)
	}
	return app.LockedProject{WorkspaceID: r.WorkspaceID, Archived: r.Archived}, true, nil
}

// UpdateProject changes the fields p gives of the project id, by the
// account by at now; the others keep their values. An identifier or a name
// another undeleted project of the workspace has is domain.ErrIdentifierTaken
// or domain.ErrNameTaken.
func (s *Store) UpdateProject(ctx context.Context, id uuid.UUID, p domain.ProjectPatch, by uuid.UUID, now time.Time) error {
	arg := gen.UpdateProjectParams{
		ID: id, Name: p.Name, Description: p.Description, Identifier: p.Identifier, SetLead: p.SetLead, ProjectLeadID: p.LeadID,
		SetDefaultAssignee: p.SetDefaultAssignee, DefaultAssigneeID: p.DefaultAssigneeID, CycleView: p.CycleView, ModuleView: p.ModuleView,
		IssueViewsView: p.IssueViewsView, IntakeView: p.IntakeView, GuestViewAllFeatures: p.GuestViewAllFeatures, Timezone: p.Timezone,
		UpdatedBy: by, Now: now,
	}
	if p.Network != nil {
		network := int16(*p.Network)
		arg.Network = &network
	}
	if p.ArchiveIn != nil {
		months := int32(*p.ArchiveIn)
		arg.ArchiveIn = &months
	}
	if p.LogoProps != nil {
		logo, err := json.Marshal(*p.LogoProps)
		if err != nil {
			return fmt.Errorf("update project %s: %w", id, err)
		}
		arg.LogoProps = logo
	}
	return taken(fmt.Sprintf("update project %s", id), s.queries(ctx).UpdateProject(ctx, arg))
}
````

`server/internal/modules/project/adapter/postgres/members.go`（新文件，25 行）：

````file server/internal/modules/project/adapter/postgres/members.go
package postgresadapter

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Memberships are userIDs' undeleted memberships of projectID, active or
// ended, by account (app.MembershipReader).
func (s *Store) Memberships(ctx context.Context, projectID uuid.UUID, userIDs []uuid.UUID) (map[uuid.UUID]app.Membership, error) {
	rows, err := s.queries(ctx).Memberships(ctx, gen.MembershipsParams{ProjectID: projectID, MemberIds: userIDs})
	if err != nil {
		return nil, fmt.Errorf("read memberships of project %s: %w", projectID, err)
	}
	out := make(map[uuid.UUID]app.Membership, len(rows))
	for _, r := range rows {
		out[r.MemberID] = app.Membership{ID: r.ID, Role: shared.Role(r.Role), Active: r.IsActive}
	}
	return out, nil
}
````

`server/internal/modules/project/adapter/postgres/update_test.go`（新文件，275 行）：

````file server/internal/modules/project/adapter/postgres/update_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// columns are the row id of table, each column's value as JSON text.
func columns(t *testing.T, pool *pgxpool.Pool, table string, id uuid.UUID) map[string]string {
	t.Helper()
	rows, err := pool.Query(context.Background(), "SELECT key, value::text FROM "+table+" r, jsonb_each(to_jsonb(r)) WHERE r.id = $1", id)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			t.Fatal(err)
		}
		out[k] = v
	}
	if err := rows.Err(); err != nil || len(out) == 0 {
		t.Fatalf("the row %s of %s: %v, %d columns", id, table, err, len(out))
	}
	return out
}

// changed is before with the columns of change changed, as JSON text.
func changed(before map[string]string, change map[string]string) map[string]string {
	out := maps.Clone(before)
	maps.Copy(out, change)
	return out
}

// LockProject reads the undeleted project's workspace and whether it is
// archived, and holds it FOR NO KEY UPDATE until the transaction ends: a
// FOR SHARE of it waits, a foreign key's FOR KEY SHARE does not, and no
// other project is held. A deleted project, and no project, are not found.
func TestLockProject(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web, ops, old := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, beta, "Ops", "OPS", alice), newProject(t, s, acme, "Old", "OLD", alice)
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", ops, now)
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", old, now)
	for _, tt := range []struct {
		id    uuid.UUID
		want  app.LockedProject
		found bool
	}{{web, app.LockedProject{WorkspaceID: acme}, true}, {ops, app.LockedProject{WorkspaceID: beta, Archived: true}, true},
		{old, app.LockedProject{}, false}, {uuid.NewV7(), app.LockedProject{}, false}} {
		if got, found, err := s.LockProject(context.Background(), tt.id); err != nil || found != tt.found || got != tt.want {
			t.Errorf("LockProject(%s) = %+v, %v, %v; want %+v, %v", tt.id, got, found, err, tt.want, tt.found)
		}
	}
	err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		if _, _, err := s.LockProject(ctx, web); err != nil {
			return err
		}
		if !waits(t, pool, web, "FOR SHARE") || waits(t, pool, web, "FOR KEY SHARE") || waits(t, pool, ops, "FOR SHARE") {
			t.Error("while the lock is held: want a FOR SHARE of web to wait, and neither its FOR KEY SHARE nor a FOR SHARE of ops")
		}
		return nil
	})
	if err != nil || waits(t, pool, web, "FOR SHARE") {
		t.Errorf("after the transaction: %v; want web free", err)
	}
}

// A project deleted while LockProject waited for its lock is not found:
// after the wait, Postgres evaluates deleted_at IS NULL again on the row's
// newest version (M3 design 3.6 convention 2).
func TestTheProjectLockSeesADeletionItWaitedFor(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	web := newProject(t, s, newWorkspace(t, pool, "acme"), "Web", "WEB", alice)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "UPDATE projects SET deleted_at = $2 WHERE id = $1", web, now); err != nil {
		t.Fatal(err)
	}
	type answer struct {
		found bool
		err   error
	}
	done := make(chan answer, 1)
	go func() {
		var a answer
		a.err = postgres.NewTxManager(pool, 5*time.Second).WithinTx(ctx, func(ctx context.Context) error {
			var err error
			_, a.found, err = s.LockProject(ctx, web)
			return err
		})
		done <- a
	}()
	pgtest.WaitForLockWaitOn(t, pool, "projects", 5*time.Second)
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case a := <-done:
		if a.err != nil || a.found {
			t.Errorf("LockProject after the deletion committed = found %v, %v; want not found", a.found, a.err)
		}
	case <-ctx.Done():
		t.Fatal("LockProject did not end within 10s")
	}
}

// ProjectWorkspace reads the undeleted project's workspace, archived or
// not, and takes no lock: while the read's transaction is open, a FOR
// UPDATE of the project does not wait. A deleted project, and no project,
// are not found. Every write on a project reads it first, to lock the
// workspace before the project (M3 design 3.6 convention 2).
func TestProjectWorkspace(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web, ops, old := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, beta, "Ops", "OPS", alice), newProject(t, s, acme, "Old", "OLD", alice)
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", ops, now)
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", old, now)
	for _, tt := range []struct {
		id, workspace uuid.UUID
		found         bool
	}{{web, acme, true}, {ops, beta, true}, {old, uuid.UUID{}, false}, {uuid.NewV7(), uuid.UUID{}, false}} {
		if got, found, err := s.ProjectWorkspace(context.Background(), tt.id); err != nil || found != tt.found || got != tt.workspace {
			t.Errorf("ProjectWorkspace(%s) = %s, %v, %v; want %s, %v", tt.id, got, found, err, tt.workspace, tt.found)
		}
	}
	err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		if _, _, err := s.ProjectWorkspace(ctx, web); err != nil {
			return err
		}
		if waits(t, pool, web, "FOR UPDATE") {
			t.Error("while the read's transaction is open, a FOR UPDATE of web waits; want no lock taken")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// UpdateProject changes exactly the fields the patch gives, and the audit
// columns to the moment and the account given; every other column of the
// project keeps its value, and every other project keeps every column: one
// of the same workspace, archived, one of another. A patch that gives
// nothing changes the audit columns alone; the lead and the default
// assignee change only when their flags are set, to none too.
func TestUpdateProject(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web := newProject(t, s, acme, "Web", "WEB", alice)
	// Every field is left out once while it holds another value than its default and than the other projects': the
	// cycles switch, which the third change sets, starts on.
	exec(t, pool, "UPDATE projects SET project_lead_id = $2, default_assignee_id = $3, cycle_view = true WHERE id = $1", web, alice, bob)
	ops := newProject(t, s, acme, "Ops", "OPS", alice)
	exec(t, pool, "UPDATE projects SET archived_at = $2, project_lead_id = $3 WHERE id = $1", ops, now, bob)
	newProject(t, s, beta, "Web", "WEB", alice)
	later := now.Add(time.Hour)
	update := func(p domain.ProjectPatch, by uuid.UUID, at time.Time) {
		t.Helper()
		if err := s.UpdateProject(context.Background(), web, p, by, at); err != nil {
			t.Fatal(err)
		}
	}
	others := tableRows(t, pool, "projects", web)
	audit := func(by uuid.UUID, at time.Time) map[string]string {
		return map[string]string{"updated_by_id": `"` + by.String() + `"`, "updated_at": `"` + at.Format("2006-01-02T15:04:05.999999") + `+00:00"`}
	}

	before := columns(t, pool, "projects", web)
	update(domain.ProjectPatch{}, carol, later)
	if got, want := columns(t, pool, "projects", web), changed(before, audit(carol, later)); !maps.Equal(got, want) {
		t.Errorf("an empty patch: %v\nwant %v", got, want)
	}

	before = columns(t, pool, "projects", web)
	update(domain.ProjectPatch{Name: ptr("研发 Web"), Description: ptr("The app"), Identifier: ptr("WEBÇ"), Network: ptr(domain.NetworkPrivate),
		SetLead: true, LeadID: &carol, CycleView: ptr(true), ModuleView: ptr(true), IssueViewsView: ptr(true), IntakeView: ptr(true),
		GuestViewAllFeatures: ptr(true), ArchiveIn: ptr(12), LogoProps: &domain.LogoProps{InUse: ptr("icon"), Icon: &domain.Icon{Name: ptr("home")}},
		Timezone: ptr("Asia/Shanghai")}, bob, later.Add(time.Hour))
	want := changed(before, audit(bob, later.Add(time.Hour)))
	maps.Copy(want, map[string]string{"name": `"研发 Web"`, "description": `"The app"`, "identifier": `"WEBÇ"`, "network": "0",
		"project_lead_id": `"` + carol.String() + `"`, "cycle_view": "true", "module_view": "true", "issue_views_view": "true",
		"intake_view": "true", "guest_view_all_features": "true", "archive_in": "12", "logo_props": `{"icon": {"name": "home"}, "in_use": "icon"}`,
		"timezone": `"Asia/Shanghai"`})
	if got := columns(t, pool, "projects", web); !maps.Equal(got, want) {
		t.Errorf("every field: %v\nwant %v", got, want)
	}

	before = columns(t, pool, "projects", web)
	update(domain.ProjectPatch{SetDefaultAssignee: true, CycleView: ptr(false)}, alice, later)
	if got, want := columns(t, pool, "projects", web), changed(before, changed(audit(alice, later),
		map[string]string{"default_assignee_id": "null", "cycle_view": "false"})); !maps.Equal(got, want) {
		t.Errorf("the default assignee cleared: %v\nwant %v", got, want)
	}
	before = columns(t, pool, "projects", web)
	update(domain.ProjectPatch{SetLead: true}, alice, later)
	if got, want := columns(t, pool, "projects", web), changed(before, changed(audit(alice, later),
		map[string]string{"project_lead_id": "null"})); !maps.Equal(got, want) {
		t.Errorf("the lead cleared: %v\nwant %v", got, want)
	}
	if after := tableRows(t, pool, "projects", web); after != others {
		t.Errorf("the other projects:\n%s\nwant\n%s", after, others)
	}
}

// An identifier or a name is taken by another undeleted project of the
// workspace only, archived or not: the project's own, another
// workspace's and a deleted project's are free (M3 design 3.19). A
// refused change changes nothing.
func TestUpdateProjectIdentifierOrNameTaken(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", ops, now)
	newProject(t, s, beta, "Docs", "DOCS", alice)
	old := newProject(t, s, acme, "Old", "OLD", alice)
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", old, now)
	update := func(name, identifier string) error {
		return s.UpdateProject(context.Background(), web, domain.ProjectPatch{Name: &name, Identifier: &identifier}, alice, now)
	}
	before := columns(t, pool, "projects", web)
	if err := update("Web", "OPS"); !errors.Is(err, domain.ErrIdentifierTaken) {
		t.Errorf("ops's identifier: %v, want project.identifier_taken", err)
	}
	if err := update("Ops", "WEB"); !errors.Is(err, domain.ErrNameTaken) {
		t.Errorf("ops's name: %v, want project.name_taken", err)
	}
	if got := columns(t, pool, "projects", web); !maps.Equal(got, before) {
		t.Errorf("after the refusals: %v, want %v", got, before)
	}
	for _, free := range [][2]string{{"Web", "WEB"}, {"Docs", "DOCS"}, {"Old", "OLD"}} {
		if err := update(free[0], free[1]); err != nil {
			t.Errorf("%s %s: %v, want it free", free[0], free[1], err)
		}
	}
}

// A change that breaks another constraint than the two unique keys is the
// store's error, not a taken identifier or name: the domain checks every
// value, so it is a bug (500).
func TestUpdateProjectBreakingAnotherConstraintIsInternal(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	web := newProject(t, s, newWorkspace(t, pool, "acme"), "Web", "WEB", alice)
	err := s.UpdateProject(context.Background(), web, domain.ProjectPatch{ArchiveIn: ptr(13)}, alice, now)
	var se *shared.Error
	var pgErr *pgconn.PgError
	if errors.As(err, &se) || !errors.As(err, &pgErr) || pgErr.ConstraintName != "projects_archive_in_check" {
		t.Errorf("archive_in 13: UpdateProject() = %v; want the violation of projects_archive_in_check, not a domain error", err)
	}
}
````

`server/internal/modules/project/adapter/postgres/members_test.go`（新文件，50 行）：

````file server/internal/modules/project/adapter/postgres/members_test.go
package postgresadapter_test

import (
	"context"
	"maps"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Memberships answers the accounts asked about, each by his undeleted
// membership of the project, active or ended, with its id and role: not
// another project's, not a deleted one, not an account not asked about.
// carol's deleted membership was stored before her ended one, and dave's
// one of ops before his none in web, so that neither row order answers
// the other's.
func TestMemberships(t *testing.T) {
	s, pool := newStore(t)
	var ids []uuid.UUID
	for _, email := range []string{"alice@corp.com", "bob@corp.com", "carol@corp.com", "dave@corp.com", "erin@corp.com", "frank@corp.com"} {
		ids = append(ids, newAccount(t, pool, email))
	}
	alice, bob, carol, dave, erin, frank := ids[0], ids[1], ids[2], ids[3], ids[4], ids[5]
	acme := newWorkspace(t, pool, "acme")
	ops, web := newProject(t, s, acme, "Ops", "OPS", alice), newProject(t, s, acme, "Web", "WEB", alice)
	seedMember(t, pool, acme, ops, dave, 20, true)
	deleted := seedMember(t, pool, acme, web, carol, 20, true)
	exec(t, pool, "UPDATE project_members SET deleted_at = $2 WHERE id = $1", deleted, now)
	carols := seedMember(t, pool, acme, web, carol, 5, false)
	alices := seedMember(t, pool, acme, web, alice, 20, true)
	bobs := seedMember(t, pool, acme, web, bob, 15, false)
	seedMember(t, pool, acme, web, erin, 15, true)
	frank2 := seedMember(t, pool, acme, web, frank, 20, true)
	exec(t, pool, "UPDATE project_members SET deleted_at = $2 WHERE id = $1", frank2, now)

	got, err := s.Memberships(context.Background(), web, []uuid.UUID{alice, bob, carol, dave, frank})
	want := map[uuid.UUID]app.Membership{
		alice: {ID: alices, Role: shared.RoleAdmin, Active: true},
		bob:   {ID: bobs, Role: shared.RoleMember},
		carol: {ID: carols, Role: shared.RoleGuest},
	}
	if err != nil || !maps.Equal(got, want) {
		t.Errorf("Memberships() = %v, %v; want %v", got, err, want)
	}
	if got, err := s.Memberships(context.Background(), web, nil); err != nil || len(got) != 0 {
		t.Errorf("Memberships() of nobody = %v, %v; want none", got, err)
	}
}
````

`server/internal/modules/project/adapter/postgres/failures_test.go`（修改，2 处）：

````old server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("ListProjects() = %+v, %v; want context.Canceled, not an empty list", list, err)
	}
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("ListProjects() = %+v, %v; want context.Canceled, not an empty list", list, err)
	}
	if p, found, err := s.LockProject(cancelled, web); !failed(err) || found || p != (app.LockedProject{}) {
		t.Errorf("LockProject() = %+v, %v, %v; want context.Canceled, not no project", p, found, err)
	}
	if w, found, err := s.ProjectWorkspace(cancelled, web); !failed(err) || found || w != (uuid.UUID{}) {
		t.Errorf("ProjectWorkspace() = %s, %v, %v; want context.Canceled, not no project", w, found, err)
	}
	if m, err := s.Memberships(cancelled, web, []uuid.UUID{alice}); !failed(err) || m != nil {
		t.Errorf("Memberships() = %v, %v; want context.Canceled, not none", m, err)
	}
````

````old server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("CreateStates() = %v; want context.Canceled", err)
	}
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("CreateStates() = %v; want context.Canceled", err)
	}
	if err := s.UpdateProject(cancelled, web, domain.ProjectPatch{Identifier: ptr("WEB")}, alice, now); !failed(err) ||
		errors.Is(err, domain.ErrIdentifierTaken) || errors.Is(err, domain.ErrNameTaken) {
		t.Errorf("UpdateProject() = %v; want context.Canceled", err)
	}
````

- [ ] **Step 5: 工作区按 id 的锁**

`server/internal/modules/workspace/adapter/postgres/directory.go`（修改，2 处）：

````old server/internal/modules/workspace/adapter/postgres/directory.go
// 6.5): WorkspaceDirectory, the undeleted workspace a slug names, and
````
````new server/internal/modules/workspace/adapter/postgres/directory.go
// 6.5): WorkspaceDirectory, the undeleted workspace a slug names, or its
// row locked by its id for a write on a project of it, and
````

````old server/internal/modules/workspace/adapter/postgres/directory.go
	r, err := d.store.queries(ctx).ShareDirectoryWorkspace(ctx, slug)
````
````new server/internal/modules/workspace/adapter/postgres/directory.go
	r, err := d.store.queries(ctx).ShareDirectoryWorkspace(ctx, slug)
	return directoryEntry(r.ID, r.Timezone, err)
}

// ShareWorkspaceByID returns the undeleted workspace id and locks its row
// FOR SHARE until the transaction ctx carries ends: the first lock of every
// write on a project of it (M3 design 3.6 convention 2). found is false
// when there is none, also when it was deleted while the lock waited.
func (d *Directory) ShareWorkspaceByID(ctx context.Context, id uuid.UUID) (w app.DirectoryEntry, found bool, err error) {
	r, err := d.store.queries(ctx).ShareDirectoryWorkspaceByID(ctx, id)
````

`server/internal/modules/workspace/adapter/postgres/directory_share_test.go`（新文件，157 行）：

````file server/internal/modules/workspace/adapter/postgres/directory_share_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The directory's lock by id, the first lock of every write on a project
// (M3 design 3.6 convention 2), finds the undeleted workspace with the id,
// with its id and its time zone: each of two workspaces its own, so a lock
// that read the first row whatever its id would answer one for the other;
// not a deleted one, not an id no workspace has. A failing call answers its
// error, not "no such workspace", which the write would turn into a 404.
func TestShareWorkspaceByIDFindsTheUndeletedWorkspace(t *testing.T) {
	s, pool := newStore(t)
	d := postgresadapter.NewDirectory(pool)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	beta := newWorkspace(t, s, "Beta", "beta", alice)
	gone := newWorkspace(t, s, "Gone", "gone", alice)
	exec(t, pool, "UPDATE workspaces SET timezone = 'Asia/Shanghai' WHERE id = $1", beta.ID)
	exec(t, pool, "UPDATE workspaces SET deleted_at = $2 WHERE id = $1", gone.ID, now)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
		for id, want := range map[uuid.UUID]app.DirectoryEntry{acme.ID: {ID: acme.ID, Timezone: "UTC"}, beta.ID: {ID: beta.ID, Timezone: "Asia/Shanghai"}} {
			if got, found, err := d.ShareWorkspaceByID(ctx, id); err != nil || !found || got != want {
				t.Errorf("ShareWorkspaceByID(%s) = %+v, %v, %v; want %+v", id, got, found, err, want)
			}
		}
		for _, id := range []uuid.UUID{gone.ID, uuid.NewV7()} {
			if got, found, err := d.ShareWorkspaceByID(ctx, id); err != nil || found || got != (app.DirectoryEntry{}) {
				t.Errorf("ShareWorkspaceByID(%s) = %+v, %v, %v; want not found", id, got, found, err)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if got, found, err := d.ShareWorkspaceByID(cancelled, acme.ID); !errors.Is(err, context.Canceled) || found || got != (app.DirectoryEntry{}) {
		t.Errorf("ShareWorkspaceByID() = %+v, %v, %v; want context.Canceled, not no workspace", got, found, err)
	}
}

// The lock by id is convention 2's FOR SHARE: it waits for the workspace's
// FOR NO KEY UPDATE, which every cascade over the workspace's projects runs
// under, and makes it wait; it does not wait for another FOR SHARE, which
// another write on a project of the workspace holds; and it locks no other
// workspace. A lock that waits ends with lock_not_available under a
// lock_timeout.
func TestTheDirectorysLockByIDIsForShare(t *testing.T) {
	const directory = "Directory.ShareWorkspaceByID"
	tests := []struct {
		held, then string
		slug       string // then's
		waits      bool
	}{
		{directory, noKeyUpdateByID.name, "acme", true},
		{noKeyUpdateByID.name, directory, "acme", true},
		{directory, forShareByID.name, "acme", false},
		{forShareByID.name, directory, "acme", false},
		{directory, noKeyUpdateByID.name, "beta", false},
		{noKeyUpdateByID.name, directory, "beta", false},
	}
	for _, tt := range tests {
		t.Run(tt.held+" held, "+tt.then+" of "+tt.slug, func(t *testing.T) {
			s, pool := newStore(t)
			d := postgresadapter.NewDirectory(pool)
			locks := map[string]lock{noKeyUpdateByID.name: noKeyUpdateByID, forShareByID.name: forShareByID, directory: {directory,
				func(ctx context.Context, _ *postgresadapter.Store, w named) (uuid.UUID, error) {
					e, found, err := d.ShareWorkspaceByID(ctx, w.id)
					if err == nil && !found {
						err = app.ErrNotFound
					}
					return e.ID, err
				}}}
			alice := newAccount(t, pool, "alice@corp.com")
			ids := map[string]uuid.UUID{"acme": newWorkspace(t, s, "Acme", "acme", alice).ID, "beta": newWorkspace(t, s, "Beta", "beta", alice).ID}
			tx := postgres.NewTxManager(pool, 2*time.Second)
			hold(t, tx, func(ctx context.Context) error {
				_, err := locks[tt.held].take(ctx, s, named{"acme", ids["acme"]})
				return err
			})

			var got uuid.UUID
			err := withLockTimeout(tx, pool, func(ctx context.Context) error {
				var err error
				got, err = locks[tt.then].take(ctx, s, named{tt.slug, ids[tt.slug]})
				return err
			})

			var pgErr *pgconn.PgError
			switch {
			case tt.waits && (!errors.As(err, &pgErr) || pgErr.Code != "55P03"):
				t.Errorf("%s() = %s, %v; want lock_not_available after waiting", tt.then, got, err)
			case !tt.waits && (err != nil || got != ids[tt.slug]):
				t.Errorf("%s() = %s, %v; want %s's id %s without waiting", tt.then, got, err, tt.slug, ids[tt.slug])
			}
		})
	}
}

// A workspace deleted while the lock by id waits is not found: the
// statement has deleted_at IS NULL, which Postgres evaluates again on the
// row's newest version after the wait.
func TestTheDirectorysLockByIDSeesADeletionItWaitedFor(t *testing.T) {
	s, pool := newStore(t)
	d := postgresadapter.NewDirectory(pool)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	end := hold(t, tx, func(ctx context.Context) error {
		if err := s.LockWorkspace(ctx, acme.ID); err != nil {
			return err
		}
		return s.DeleteWorkspace(ctx, acme.ID, alice, now)
	})
	type answer struct {
		w     app.DirectoryEntry
		found bool
		err   error
	}
	done := make(chan answer, 1)
	go func() {
		var a answer
		a.err = tx.WithinTx(context.Background(), func(ctx context.Context) error {
			var err error
			a.w, a.found, err = d.ShareWorkspaceByID(ctx, acme.ID)
			return err
		})
		done <- a
	}()
	pgtest.WaitForLockWaitOn(t, pool, "workspaces", 10*time.Second)
	if err := end(); err != nil {
		t.Fatal(err)
	}
	select {
	case a := <-done:
		if a.err != nil || a.found {
			t.Errorf("ShareWorkspaceByID() after the deletion = %+v, %v, %v; want not found", a.w, a.found, a.err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("ShareWorkspaceByID() did not end within 10s")
	}
}
````

`server/internal/modules/workspace/module.go`（修改，2 处）：

````old server/internal/modules/workspace/module.go
// project module (M3 design 6.5); found is false when there is none.
````
````new server/internal/modules/workspace/module.go
// project module, or locks one by its id (M3 design 6.5); found is false
// when there is none.
````

````old server/internal/modules/workspace/module.go
	ShareWorkspaceBySlug(ctx context.Context, slug string) (w DirectoryEntry, found bool, err error)
````
````new server/internal/modules/workspace/module.go
	ShareWorkspaceBySlug(ctx context.Context, slug string) (w DirectoryEntry, found bool, err error)
	// ShareWorkspaceByID locks the undeleted workspace id's row FOR SHARE
	// until the transaction ctx carries ends: the first lock of every write
	// on a project of it (M3 design 3.6 convention 2). A workspace deleted
	// while the lock waited is not found.
	ShareWorkspaceByID(ctx context.Context, id uuid.UUID) (w DirectoryEntry, found bool, err error)
````

`server/internal/bootstrap/ports.go`（修改，1 处）：

````old server/internal/bootstrap/ports.go
}

// accessProjects is project's ProjectAccess as access's port: the same read,
````
````new server/internal/bootstrap/ports.go
}

func (d projectWorkspaces) ShareWorkspaceByID(ctx context.Context, id uuid.UUID) (project.Workspace, bool, error) {
	w, found, err := d.directory.ShareWorkspaceByID(ctx, id)
	return project.Workspace(w), found, err
}

// accessProjects is project's ProjectAccess as access's port: the same read,
````

`server/internal/bootstrap/ports_test.go`（修改，5 处）：

````old server/internal/bootstrap/ports_test.go
// fakeWorkspaceDirectory answers the workspaces it holds by slug, and
// records what it was asked: "read" without a lock, "share" with one.
````
````new server/internal/bootstrap/ports_test.go
// fakeWorkspaceDirectory answers the workspaces it holds by slug, or by the
// id names gives a slug, and records what it was asked: "read" without a
// lock, "share" with one, "share by id" by the id's name.
````

````old server/internal/bootstrap/ports_test.go
	workspaces map[string]workspace.DirectoryEntry
````
````new server/internal/bootstrap/ports_test.go
	workspaces map[string]workspace.DirectoryEntry
	names      map[uuid.UUID]string
````

````old server/internal/bootstrap/ports_test.go
}

// projectWorkspaces hands project workspace's answer to the same question,
// through the same lock or without one: the workspace converted, found and
// the error as they came.
````
````new server/internal/bootstrap/ports_test.go
}

func (f *fakeWorkspaceDirectory) ShareWorkspaceByID(_ context.Context, id uuid.UUID) (workspace.DirectoryEntry, bool, error) {
	f.asked = append(f.asked, "share by id "+f.names[id])
	w, found := f.workspaces[f.names[id]]
	return w, found, f.err
}

// projectWorkspaces hands project workspace's answer to the same question,
// through the same lock or without one, by slug or by id: the workspace
// converted, found and the error as they came.
````

````old server/internal/bootstrap/ports_test.go
	fake := &fakeWorkspaceDirectory{workspaces: map[string]workspace.DirectoryEntry{"acme": acme}}
````
````new server/internal/bootstrap/ports_test.go
	ids := map[string]uuid.UUID{"acme": acme.ID, "gone": uuid.NewV7()}
	fake := &fakeWorkspaceDirectory{workspaces: map[string]workspace.DirectoryEntry{"acme": acme},
		names: map[uuid.UUID]string{ids["acme"]: "acme", ids["gone"]: "gone"}}
````

````old server/internal/bootstrap/ports_test.go
		"share": d.ShareWorkspaceBySlug, "read": d.WorkspaceBySlug,
````
````new server/internal/bootstrap/ports_test.go
		"share": d.ShareWorkspaceBySlug, "read": d.WorkspaceBySlug,
		"share by id": func(ctx context.Context, slug string) (project.Workspace, bool, error) {
			return d.ShareWorkspaceByID(ctx, ids[slug])
		},
````

`server/internal/modules/project/app/fakes_create_test.go`（修改，2 处）：

````old server/internal/modules/project/app/fakes_create_test.go
// fakeDirectory finds the workspaces it holds by slug, logs each call and
// fails with err.
````
````new server/internal/modules/project/app/fakes_create_test.go
// fakeDirectory finds the workspaces it holds by slug, or by id, logs each
// call and fails with err.
````

````old server/internal/modules/project/app/fakes_create_test.go
		return app.Workspace{}, false, f.err
	}
	w, ok := f.workspaces[slug]
	return w, ok, nil
}

// fakeMembers answers the active members' roles it holds by workspace,
````
````new server/internal/modules/project/app/fakes_create_test.go
		return app.Workspace{}, false, f.err
	}
	w, ok := f.workspaces[slug]
	return w, ok, nil
}

func (f *fakeDirectory) ShareWorkspaceByID(ctx context.Context, id uuid.UUID) (app.Workspace, bool, error) {
	f.log.add(ctx, "ShareWorkspaceByID %s", id)
	if f.err != nil {
		return app.Workspace{}, false, f.err
	}
	for _, w := range f.workspaces {
		if w.ID == id {
			return w, true, nil
		}
	}
	return app.Workspace{}, false, nil
}

// fakeMembers answers the active members' roles it holds by workspace,
````

- [ ] **Step 6: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/project/... ./internal/modules/workspace/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestProjectWorkspacesConvertsWorkspacesAnswer' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 7: 提交**

```bash
git add server/internal/bootstrap/ports.go server/internal/bootstrap/ports_test.go server/internal/modules/project/adapter/postgres/failures_test.go server/internal/modules/project/adapter/postgres/members.go server/internal/modules/project/adapter/postgres/members_test.go server/internal/modules/project/adapter/postgres/projects.go server/internal/modules/project/adapter/postgres/queries/members.sql server/internal/modules/project/adapter/postgres/queries/projects.sql server/internal/modules/project/adapter/postgres/update.go server/internal/modules/project/adapter/postgres/update_test.go server/internal/modules/project/app/fakes_create_test.go server/internal/modules/project/app/ports.go server/internal/modules/project/domain/errors.go server/internal/modules/project/domain/patch.go server/internal/modules/project/domain/patch_test.go server/internal/modules/workspace/adapter/postgres/directory.go server/internal/modules/workspace/adapter/postgres/directory_share_test.go server/internal/modules/workspace/adapter/postgres/queries/directory.sql server/internal/modules/workspace/module.go server/internal/modules/project/adapter/postgres/gen/members.sql.go server/internal/modules/project/adapter/postgres/gen/projects.sql.go server/internal/modules/workspace/adapter/postgres/gen/directory.sql.go
```
```bash
git commit -m "feat(M3/P4b): the project patch, the project lock and the memberships read

CheckProjectPatch holds each field an update gives to createProject's
rule, and archive_in to 0-12; CanAssign names the roles a lead or a
default assignee may have. The store locks an undeleted project FOR NO
KEY UPDATE, evaluating deleted_at again after a wait, changes the fields
a patch gives, and reads accounts' undeleted memberships of a project
and, without a lock, a project's workspace. createProject and the update
share the unique keys' translation. The workspace directory locks an
undeleted workspace by its id FOR SHARE: the first lock of every write
on a project (M3 design 3.6 convention 2), converted by bootstrap.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| `LockProject` 去掉项目的 id（按 id 升序、降序两个行序） | `TestLockProject`；Task 15 起单独运行的故事：升序 P2、P3、P4、P8、W3，降序 P2、P3、P4 | 存储；端到端 |
| `LockProject` 去掉 `deleted_at IS NULL` | `TestLockProject`；与 `ProjectWorkspace` 的 `deleted_at`、`ProjectFacts` 的 `p.deleted_at` 一起去掉时 P4（Task 15 起；方案 E 之下写先经 `ProjectWorkspace`，spec 第 3 节第 6 条） | 存储；端到端 |
| `LockProject` 取 `FOR SHARE` | `TestLockProject`；`TestTwoWritesOnAProjectSerialize`（Task 14 起：两个写都持有 Web，各自的修改等对方，40P01） | 存储；组合 |
| `LockProject` 取 `FOR UPDATE` | `TestLockProject`（多挡外键检查的 `FOR KEY SHARE`；组合一层看不到，spec 第 3 节第 7 条） | 存储 |
| `LockProject` 不加锁 | `TestLockProject`；`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 4 起：等 Ops 的写不在 `projects` 上等）、`TestTwoWritesOnAProjectSerialize`（Task 14 起） | 存储；组合 |
| `LockProject` 不读已归档的项目 | `TestLockProject`；`TestPermissionMatrix`（Task 3 起，已归档项目的列）；P4（Task 15 起） | 存储；组合；端到端 |
| `UpdateProject` 改每个项目（去掉 id） | `TestUpdateProject`；P3（Task 15 起） | 存储；端到端 |
| `UpdateProject` 也写 `created_at`、`created_by_id`；没给的名称、`archive_in`、`logo_props`、`cycle_view`、时区不保留原值；不看标志就写负责人、默认负责人 | `TestUpdateProject` | 存储 |
| `Memberships` 读别的项目的（两个行序） | `TestMemberships`；P3（Task 15 起） | 存储；端到端 |
| `Memberships` 读没问到的账户的、已删除的（两个行序） | `TestMemberships`（故事看不到：spec 第 3 节第 6 条） | 存储 |
| 两个唯一键的 409 互换 | `TestUpdateProjectIdentifierOrNameTaken` | 存储 |
| `archive_in` 收 13、收 −1；标识不按规则查；名称不按规则查 | `TestCheckProjectPatchReportsEveryField`（后两个另有 `TestThePatchChecksAsCreateDoes`）；收 13 另有 P3（Task 15 起，答 500 而不是 422）；其余只在单元一层：数据库的 CHECK 和列的类型拒绝这些值，答 500 | 单元；端到端 |
| 标识不转大写 | `TestCheckProjectPatchAcceptsValidPatches`、`TestThePatchChecksAsCreateDoes`；P3（Task 15 起） | 单元；端到端 |
| `CanAssign` 收项目的访客 | `TestTheLeadAndTheDefaultAssigneeAreActiveMembersWhoAreNoGuests`（Task 12 起）；P3（Task 15 起） | 组合；端到端 |
| `ProjectWorkspace` 去掉项目的 id（两个行序） | `TestProjectWorkspace`；Task 15 起单独运行的故事：两个行序都是 P2、P3、P4、P8 | 存储；端到端 |
| `ProjectWorkspace` 去掉 `deleted_at IS NULL` | `TestProjectWorkspace`；与 `ProjectFacts` 的 `p.deleted_at` 一起去掉时 P4（Task 15 起，spec 第 3 节第 6 条） | 存储；端到端 |
| `ProjectWorkspace` 取项目的 `FOR SHARE`（在工作区之前锁了项目） | `TestProjectWorkspace`；`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 4 起：等工作区时已持有项目） | 存储；组合 |
| `ShareWorkspaceByID` 取 `FOR KEY SHARE`（连带的 `FOR NO KEY UPDATE` 不等它）、不加锁 | `TestTheDirectorysLockByIDIsForShare`；`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 4 起）、`TestAWorkspacesDeletionWaitsForTheWritesOnItsProjects`（Task 14 起） | 存储；组合 |
| `ShareWorkspaceByID` 取 `FOR NO KEY UPDATE`、`FOR UPDATE`（同一个工作区的写互等） | `TestTheDirectorysLockByIDIsForShare`；`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 4 起）、`TestWritesOnTwoProjectsOfAWorkspaceDoNotWait`（Task 14 起） | 存储；组合 |
| `ShareWorkspaceByID` 去掉工作区的 id（两个行序：锁住每个未删除的工作区，答第一行的） | `TestShareWorkspaceByIDFindsTheUndeletedWorkspace`、`TestTheDirectorysLockByIDIsForShare`（别的工作区被锁；`Locks` 只看找到与否，组合一层和故事看不到，spec 第 3 节第 15 条） | 存储 |
| `ShareWorkspaceByID` 去掉 `deleted_at IS NULL` | `TestShareWorkspaceByIDFindsTheUndeletedWorkspace`、`TestTheDirectorysLockByIDSeesADeletionItWaitedFor`（组合一层由项目锁的 `deleted_at` 和连带遮住，spec 第 3 节第 15 条） | 存储 |
| `ShareWorkspaceByID` 经连接池、在写的事务之外 | `TestTheDirectorysLockByIDIsForShare`；`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 4 起）、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 14 起） | 存储；组合 |
| `bootstrap` 的 `ShareWorkspaceByID` 不问目录就答找到 | `TestProjectWorkspacesConvertsWorkspacesAnswer`；`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 4 起）、`TestAWorkspacesDeletionWaitsForTheWritesOnItsProjects`（Task 14 起） | 单元；组合 |
| `bootstrap` 的 `ShareWorkspaceByID` 在写的事务之外问目录 | `TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 4 起）、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 14 起） | 组合 |
| `LockProject`、`ProjectWorkspace`、`UpdateProject`、`Memberships` 经连接池、在事务之外执行 | `TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 14 起） | 组合 |

**Done when:** 领域的三个测试、存储的七个测试、工作区目录的三个测试通过；修改的每个字段与建项目同一规则；项目锁和工作区按 id 的锁在等锁之后看得到删除；`ProjectWorkspace` 不加锁。

---

### Task 3: `updateProject`

**Files:**
- Create: `server/internal/bootstrap/project_writes_test.go`、`server/internal/modules/project/adapter/http/update_test.go`、`server/internal/modules/project/app/fakes_write_test.go`、`server/internal/modules/project/app/lock.go`、`server/internal/modules/project/app/update_project.go`、`server/internal/modules/project/app/update_project_test.go`
- Modify: `api/modules/project.yaml`、`server/internal/bootstrap/permission_matrix_project_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/project/adapter/http/handler.go`、`server/internal/modules/project/adapter/http/handler_test.go`、`server/internal/modules/project/adapter/http/projects.go`、`server/internal/modules/project/app/clock_test.go`、`server/internal/modules/project/app/ports.go`、`server/internal/modules/project/domain/actions.go`、`server/internal/modules/project/module.go`、`web/apps/web/helpers/authentication.helper.ts`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/project/adapter/http/gen/bodyshape.gen.go`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.4，M3 设计 3.4、3.6、3.19、5.2）：`PATCH /api/v0/projects/{project_id}`，`ProjectUpdate`（`additionalProperties: false`，只有 `project_lead_id`、`default_assignee_id` 可为 `null`），200 `Project`；码 `[validation_failed, project.not_found, forbidden, project.archived, project.identifier_taken, project.name_taken]`；新码 `project.archived`（409）进 `PROBLEM_MESSAGES` 和两份 `auth.json`（约束 4）。
- 操作名 `project.update`，规则 `{Level: LevelProject, Roles: [RoleAdmin]}`：项目管理员，和是工作区管理员的项目成员（`ProjectAdmin`），不是成员的工作区管理员不行（M3 设计 3.4）。
- `app.NewUpdateProject(projects ProjectUpdater, locks Locks, tx, clock)`：`CheckProjectPatch`（事务之前）→ 一个事务：`locks.lockAndDecide(actor, write{project: id, action: project.update})`（下一条）→ 已归档 409 → `checkAssignees`（判定之后：给出的负责人、默认负责人各在 `Memberships` 里是有效的、`CanAssign` 的成员，否则一个 422 列出每个字段）→ 锁下读时钟 → `UpdateProject` → `answer`（在事务里经 `GetProject` 读回存下的行）。
- `app/lock.go`（M3 设计 3.6 约定二，方案 E）：`Locks{projects ProjectLocks, workspaces WorkspaceSharer, auth}`、`NewLocks(projects, workspaces, auth)`：项目级的写取锁和判定的唯一一条路径。`(Locks).lockAndDecide(ctx, actor, write{project, action}) (held{project LockedProject, grant}, error)`，在 `ctx` 带的事务里：`ProjectWorkspace`（不加锁；没有是 404）→ `ShareWorkspaceByID`（工作区行 `FOR SHARE`，写的第一把锁；等待期间被删除是 404）→ `LockProject`（没有，或锁读到的工作区不是先读的那个，是 404）→ `decide`（在全部锁之下；看不到换成 `project.not_found`）。`decide`、`answer`（在事务里经 `GetProject` 读回存下的行）。写不持自己的 `Authorizer`，只经 `Locks` 判定；`project.New` 只建一个 `Locks` 给每个写。Task 4、6、7（`share`：改设置的 `FOR SHARE`）、11（`targets`：目标的工作区成员行）、13 共用。`app.ProjectLocks`（`ProjectFinder` + `ProjectLocker`）；`ProjectUpdater` 不再嵌入 `ProjectLocker`。
- 使用者：Task 15 的 P3、P4。

**Tests:**
- `app/update_project_test.go`（假实现 `fakes_write_test.go`：web 的管理员 bob、成员 alice、访客 carol、已结束的 dave；ops 已归档；`fakeWorkspaces` 锁 acme 的行，`gone` 时找不到；`fakeStore.moved` 让项目的锁读到另一个工作区；`lockedTo`、`lockedDecision`、`noProject` 是调用记录的开头）：`TestUpdateProject`（三种修改的完整调用记录：项目的工作区、工作区的锁、项目的锁、判定、负责人的成员关系（只在给了负责人时）、时钟、改、读回；标识转大写；回答是存下的行）；`TestUpdateProjectRefuses`（`archive_in` 13、没有调用者在事务之前；没有的项目在读它的工作区时、工作区等待期间被删除在工作区的锁、项目已不在那个工作区在项目的锁、看不到在判定、项目成员的 403 都在检查负责人之前，带一个无效的负责人也一样；已归档 409 在判定之后；负责人是访客、已结束、不是成员，默认负责人是访客，两者同时：一个 422 列出每个字段）；`TestUpdateProjectReturnsEachFailure`（项目的工作区、工作区的锁、项目的锁、判定、成员关系、改、读回、提交各失败：原样返回，之前的调用都在、之后的都没有；读回找不到是内部错误，不是 404）；`TestUpdateProjectAnswersTheStoresConflicts`。
- `app/clock_test.go`：`TestEachWriteReadsTheClockUnderItsLock`（新：每个改已有行的写在它的锁（工作区的 `FOR SHARE` 在先，然后项目的）、判定、检查之后读一次时钟，排在别的写之后的写不会盖上更早的时刻；此后每个写的 Task 加一行）。
- `adapter/http/update_test.go`：`TestUpdateProjectPassesThePatch`（每个字段照写传给用例，负责人给出、`null`、没给三种）；`TestUpdateProjectHoldsTheBodyToItsStructure`（不能写的字段、别的类型、只有负责人可为 `null`：400，用例没被调用）；`TestUpdateProjectRefusals`（六种拒绝照契约）。
- `access/domain/rules_test.go`：`project.update` 的 17 格。
- `bootstrap`：矩阵三行（`ofProject(200, 403, 403, 200, 403, 403)`，答案核对改名、调用者的角色；负责人不是成员时 `ofProject(422, 403, 403, 422, 403, 403)`：不能改的人得不到负责人的任何信息；已归档项目 1 格 409）；`TestTheWritesOnAProjectStampTheirRequest`（新：经 API 的写在请求之内盖上 `project.New` 的时钟，由调用者写入；此后每个写的 Task 加一行）。

- [ ] **Step 1: 接口描述**

`api/modules/project.yaml`（修改，2 处）：

````old api/modules/project.yaml
          description: The project, as the caller sees it.
````
````new api/modules/project.yaml
          description: The project, as the caller sees it.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Project'
        default:
          $ref: '#/components/responses/Problem'
    patch:
      operationId: updateProject
      tags: [project]
      summary: Change a project
      description: >-
        For the project's admins, and its members who are the workspace's
        admins. The fields given change and the others stay; the values
        follow createProject's rules, and archive_in is 0–12
        (validation_failed), checked before the project is looked at. An
        archived project cannot be changed (project.archived): unarchive it
        first. The lead and the default assignee, null to clear either, must
        be active members of the project who are not its guests
        (project_lead_id, default_assignee_id not_allowed), which is checked
        after the caller's role. Neither the name nor the identifier may be
        another undeleted project's of the workspace (project.name_taken,
        project.identifier_taken). A project that does not exist, is deleted,
        or that the caller does not see answers project.not_found; one he
        sees but may not change, forbidden. The role is decided after the
        project row is locked, so a caller demoted meanwhile is refused.
      security: [{bearer: []}]
      x-problem-codes: [validation_failed, project.not_found, forbidden, project.archived, project.identifier_taken, project.name_taken]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ProjectUpdate'
      responses:
        '200':
          description: The project as changed, as the caller sees it.
````

````old api/modules/project.yaml
          description: An IANA time zone name, e.g. from GET /api/v0/timezones; the workspace's when not given.
          type: string
````
````new api/modules/project.yaml
          description: An IANA time zone name, e.g. from GET /api/v0/timezones; the workspace's when not given.
          type: string
    ProjectUpdate:
      description: >-
        Changes the fields it names; a field left out keeps its value. Only
        project_lead_id and default_assignee_id can be null, which clears
        them.
      type: object
      additionalProperties: false
      properties:
        name:
          description: >-
            1–255 characters, not blank, without any of
            & + , : ; $ ^ } { * = ? @ # | ' < > . ( ) % ! -
          type: string
        identifier:
          description: 1–10 of A-Z, 0-9 and ÇŞĞİÖÜ, once upper-cased.
          type: string
        description:
          type: string
        network:
          $ref: '#/components/schemas/ProjectNetwork'
        project_lead_id:
          description: An active member of the project who is not its guest; null for none.
          type: [string, 'null']
          format: uuid
        default_assignee_id:
          description: An active member of the project who is not its guest; null for none.
          type: [string, 'null']
          format: uuid
        cycle_view:
          type: boolean
        module_view:
          type: boolean
        issue_views_view:
          type: boolean
        intake_view:
          type: boolean
        guest_view_all_features:
          type: boolean
        archive_in:
          description: After how many months a closed work item is archived, 0–12; 0 never.
          type: integer
        logo_props:
          $ref: '#/components/schemas/LogoProps'
        timezone:
          description: An IANA time zone name.
          type: string
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `75a7a09ce8be6c7e0bbe8607154e818f7b35d83ea7b06fa38adbffdb6e9840e5` | 2081 | `api/dist/openapi.yaml` |
| `86d7abb16a62cf346541201fb421c1048e3a34bb605eed3b6b6e93b955edff4d` | 48 | `server/internal/modules/project/adapter/http/gen/bodyshape.gen.go` |
| `32edef5124c248b823a3dad3b6bf9189669300c6b65bdb2b0ec0f93564d4d915` | 1017 | `server/internal/modules/project/adapter/http/gen/server.gen.go` |
| `04b20ead6fe93becce51b20825a7d26a20314dd1b40ac00093857804a18748cf` | 2161 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/bodyshape.gen.go server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: 操作名和规则**

`server/internal/modules/project/domain/actions.go`（修改，2 处）：

````old server/internal/modules/project/domain/actions.go
	ActionCheckIdentifier shared.Action = "project_identifier.check"
````
````new server/internal/modules/project/domain/actions.go
	ActionCheckIdentifier shared.Action = "project_identifier.check"
	// ActionUpdate is changing a project: updateProject.
	ActionUpdate shared.Action = "project.update"
````

````old server/internal/modules/project/domain/actions.go
	return []shared.Action{ActionList, ActionCreate, ActionRead, ActionCheckIdentifier}
````
````new server/internal/modules/project/domain/actions.go
	return []shared.Action{ActionList, ActionCreate, ActionRead, ActionCheckIdentifier, ActionUpdate}
````

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"project_identifier.check": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
````
````new server/internal/modules/access/domain/rules.go
	"project_identifier.check": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
	// The project's admins, and its members who are the workspace's admins
	// (M3 design 3.4: a project-level rule, which Plane's pages hold).
	"project.update": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
		invisible, invisible, invisible, forbidden, forbidden},
````
````new server/internal/modules/access/domain/rules_test.go
		invisible, invisible, invisible, forbidden, forbidden},
	"project.update": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
````

- [ ] **Step 3: 用例**

`server/internal/modules/project/app/ports.go`（修改，2 处）：

````old server/internal/modules/project/app/ports.go
}

// Membership is an account's undeleted membership of a project, active or
````
````new server/internal/modules/project/app/ports.go
}

// ProjectLocks is the project store's side of the locks of a write on a
// project (Locks): the project's workspace, read first without a lock, and
// the project's own lock.
type ProjectLocks interface {
	ProjectFinder
	ProjectLocker
}

// Membership is an account's undeleted membership of a project, active or
````

````old server/internal/modules/project/app/ports.go
	ProjectReader
	ProjectLocker
````
````new server/internal/modules/project/app/ports.go
	ProjectReader
````

`server/internal/modules/project/app/lock.go`（新文件，110 行）：

````file server/internal/modules/project/app/lock.go
package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Locks is the one way a write on a project named by its id takes its
// locks and its decision (M3 design 3.6 convention 2), in the transaction
// ctx carries: the project's workspace, read without a lock; the
// workspace's row FOR SHARE, the write's first lock; the project's row FOR
// NO KEY UPDATE, still undeleted and of that workspace; then the decision,
// under them all. Every cascade over the workspace's projects runs under
// the workspace's FOR NO KEY UPDATE and reads its time after it (3.3):
// while a write holds the workspace FOR SHARE, no cascade touches the rows
// it writes. project.New builds one Locks for every write: a write holds
// no Authorizer of its own, so it decides only under these locks.
type Locks struct {
	projects   ProjectLocks
	workspaces WorkspaceSharer
	auth       shared.Authorizer
}

// NewLocks returns the locks over the project store, the workspace
// module's directory and the Authorizer.
func NewLocks(projects ProjectLocks, workspaces WorkspaceSharer, auth shared.Authorizer) Locks {
	return Locks{projects: projects, workspaces: workspaces, auth: auth}
}

// write is a write on a project, as Locks takes its locks.
type write struct {
	project uuid.UUID
	action  shared.Action
}

// held is a write's locks taken and its decision made: the project as its
// lock read it and the caller's grant.
type held struct {
	project LockedProject
	grant   shared.Grant
}

// lockAndDecide takes w's locks and decides w's action on the project, in
// the order of Locks. A project that is not there, deleted while a lock
// waited, or not visible to actor is domain.ErrNotFound, and so is one
// whose workspace was deleted while its lock waited; a role the rule does
// not allow is the Authorizer's shared.Forbidden.
func (l Locks) lockAndDecide(ctx context.Context, actor shared.Actor, w write) (held, error) {
	workspaceID, found, err := l.projects.ProjectWorkspace(ctx, w.project)
	switch {
	case err != nil:
		return held{}, err
	case !found:
		return held{}, domain.ErrNotFound
	}
	_, found, err = l.workspaces.ShareWorkspaceByID(ctx, workspaceID)
	switch {
	case err != nil:
		return held{}, err
	case !found:
		return held{}, domain.ErrNotFound
	}
	var h held
	h.project, found, err = l.projects.LockProject(ctx, w.project)
	switch {
	case err != nil:
		return held{}, err
	case !found, h.project.WorkspaceID != workspaceID:
		return held{}, domain.ErrNotFound
	}
	if h.grant, err = decide(ctx, l.auth, actor, w.action, workspaceID, w.project); err != nil {
		return held{}, err
	}
	return h, nil
}

// decide asks the Authorizer for action on the project id of the workspace
// for actor. A write calls it under its locks, so the facts it reads are
// the ones committed after the locks were granted (M3 design 6.7): a
// demotion or a removal that committed while the write waited is seen. A
// project not visible to actor is domain.ErrNotFound, the 404 of what the
// caller named.
func decide(ctx context.Context, auth shared.Authorizer, actor shared.Actor, action shared.Action, workspaceID, id uuid.UUID) (shared.Grant, error) {
	grant, err := auth.Authorize(ctx, actor, action, shared.Target{WorkspaceID: workspaceID, ProjectID: id})
	switch {
	case errors.Is(err, shared.ErrNotVisible):
		return shared.Grant{}, domain.ErrNotFound
	case err != nil:
		return shared.Grant{}, err
	}
	return grant, nil
}

// answer reads the project id back as the caller sees it, in the transaction
// ctx carries: a write's answer. A project not there after the write is an
// internal error, not a 404.
func answer(ctx context.Context, projects ProjectReader, id, userID uuid.UUID) (domain.Project, error) {
	p, found, err := projects.GetProject(ctx, id, userID)
	switch {
	case err != nil:
		return domain.Project{}, err
	case !found:
		return domain.Project{}, errors.New("project " + id.String() + " is not there after its write")
	}
	return p, nil
}
````

`server/internal/modules/project/app/update_project.go`（新文件，102 行）：

````file server/internal/modules/project/app/update_project.go
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateProject changes a project: PATCH /api/v0/projects/{project_id} (M3
// design 3.19).
type UpdateProject struct {
	projects ProjectUpdater
	locks    Locks
	tx       shared.TxManager
	clock    Clock
}

// NewUpdateProject returns the use case.
func NewUpdateProject(projects ProjectUpdater, locks Locks, tx shared.TxManager, clock Clock) *UpdateProject {
	return &UpdateProject{projects: projects, locks: locks, tx: tx, clock: clock}
}

// Execute checks p (domain.CheckProjectPatch), then, in one transaction, in
// the order of M3 design 3.6: the project's locks (Locks: its workspace FOR
// SHARE, then the project FOR NO KEY UPDATE) and the decision on
// project.update; an archived project refused (409 project.archived,
// 3.19); the lead and the default assignee p names checked, after the
// decision, so that a caller who may not change the project learns nothing
// of them (422 not_allowed unless an active member of the project who is
// not its guest, 3.19); the change, at the clock read under the locks, so
// a change that waited for another is not stamped earlier than it. The
// answer is the project as stored, as the caller sees it.
func (u *UpdateProject) Execute(ctx context.Context, id uuid.UUID, p domain.ProjectPatch) (domain.Project, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Project{}, err
	}
	if p, err = domain.CheckProjectPatch(p); err != nil {
		return domain.Project{}, err
	}
	var updated domain.Project
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		h, err := u.locks.lockAndDecide(ctx, actor, write{project: id, action: domain.ActionUpdate})
		switch {
		case err != nil:
			return err
		case h.project.Archived:
			return domain.ErrArchived
		}
		if err := u.checkAssignees(ctx, id, p); err != nil {
			return err
		}
		if err := u.projects.UpdateProject(ctx, id, p, actor.UserID, u.clock.Now()); err != nil {
			return err
		}
		updated, err = answer(ctx, u.projects, id, actor.UserID)
		return err
	})
	if err != nil {
		return domain.Project{}, err
	}
	return updated, nil
}

// checkAssignees refuses a lead or a default assignee that p sets to an
// account who is not an active member of the project, or is its guest:
// one 422 naming each such field.
func (u *UpdateProject) checkAssignees(ctx context.Context, id uuid.UUID, p domain.ProjectPatch) error {
	fields := []struct {
		name    string
		set     bool
		account *uuid.UUID
	}{{"project_lead_id", p.SetLead, p.LeadID}, {"default_assignee_id", p.SetDefaultAssignee, p.DefaultAssigneeID}}
	var accounts []uuid.UUID
	for _, f := range fields {
		if f.set && f.account != nil {
			accounts = append(accounts, *f.account)
		}
	}
	if len(accounts) == 0 {
		return nil
	}
	members, err := u.projects.Memberships(ctx, id, accounts)
	if err != nil {
		return err
	}
	var problems []shared.FieldError
	for _, f := range fields {
		if !f.set || f.account == nil {
			continue
		}
		if m, ok := members[*f.account]; !ok || !m.Active || !domain.CanAssign(m.Role) {
			problems = append(problems, domain.Unassignable(f.name))
		}
	}
	if len(problems) > 0 {
		return shared.Invalid(problems...)
	}
	return nil
}
````

`server/internal/modules/project/app/fakes_write_test.go`（新文件，196 行）：

````file server/internal/modules/project/app/fakes_write_test.go
package app_test

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeProject is a project as fakeStore holds it: its workspace, whether it
// is archived, its memberships by account, and when it was last written,
// as stored.
type fakeProject struct {
	workspace uuid.UUID
	archived  bool
	members   map[uuid.UUID]app.Membership
	updated   time.Time
}

// The projects of the writes' tests: acme's web, whose admin is bob, whose
// member is alice and whose guest is carol, dave's membership ended; acme's
// ops, archived, whose admin is bob.
var webID, opsID = uuid.NewV7(), uuid.NewV7()

// writeFixture is a write use case's fakes, sharing one log.
type writeFixture struct {
	log        *callLog
	tx         *fakeTx
	store      *fakeStore
	workspaces *fakeWorkspaces
	auth       *fakeAuthorizer
}

// newWrites is writeFixture with web and ops as stored at now, acme's row
// to lock, and the Authorizer's answers: bob's grant in acme, the project's
// admin; alice's 403, a project member; any other caller sees nothing.
func newWrites() *writeFixture {
	log := &callLog{}
	return &writeFixture{log: log, tx: &fakeTx{log: log}, workspaces: &fakeWorkspaces{log: log},
		store: &fakeStore{log: log, projects: map[uuid.UUID]*fakeProject{
			webID: {workspace: acme.ID, updated: now, members: map[uuid.UUID]app.Membership{
				bob:   {ID: uuid.NewV7(), Role: shared.RoleAdmin, Active: true},
				alice: {ID: uuid.NewV7(), Role: shared.RoleMember, Active: true},
				carol: {ID: uuid.NewV7(), Role: shared.RoleGuest, Active: true},
				dave:  {ID: uuid.NewV7(), Role: shared.RoleMember},
			}},
			opsID: {workspace: acme.ID, archived: true, updated: now, members: map[uuid.UUID]app.Membership{
				bob: {ID: uuid.NewV7(), Role: shared.RoleAdmin, Active: true},
			}},
		}},
		auth: &fakeAuthorizer{log: log,
			grants: map[grantKey]shared.Grant{{bob, acme.ID}: {WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleAdmin, ProjectAdmin: true}},
			errs:   map[grantKey]error{{alice, acme.ID}: shared.Forbidden()}},
	}
}

// locks is Locks over the fixture's fakes.
func (f *writeFixture) locks() app.Locks {
	return app.NewLocks(f.store, f.workspaces, f.auth)
}

// fakeWorkspaces is the workspace module's lock of a workspace's row by its
// id: it logs each call, finds acme unless gone, and fails with err.
type fakeWorkspaces struct {
	log  *callLog
	gone bool
	err  error
}

func (f *fakeWorkspaces) ShareWorkspaceByID(ctx context.Context, id uuid.UUID) (app.Workspace, bool, error) {
	f.log.add(ctx, "ShareWorkspaceByID %s", id)
	if f.err != nil {
		return app.Workspace{}, false, f.err
	}
	if f.gone || id != acme.ID {
		return app.Workspace{}, false, nil
	}
	return acme, true, nil
}

// lockedTo are the calls of Locks on project before its own lock: the
// transaction, the project's workspace, acme's row FOR SHARE.
func lockedTo(project uuid.UUID) []string {
	return []string{"Begin", "ProjectWorkspace " + project.String(), "ShareWorkspaceByID " + acme.ID.String()}
}

// lockedDecision are the calls of a write on project by user before its
// checks: its locks (the workspace's, the project's FOR NO KEY UPDATE) and
// the decision on action.
func lockedDecision(user, project uuid.UUID, action shared.Action) []string {
	return append(lockedTo(project), "LockProject "+project.String(), fmt.Sprintf("Authorize %s %s on %s/%s", user, action, acme.ID, project))
}

// noProject are the calls of a write on a project that is not there: the
// transaction, the project's workspace, which finds none.
var noProject = []string{"Begin", "ProjectWorkspace " + uuid.Nil().String()}

// fakeStore is the project store of the writes on a project: it logs each
// call with its arguments, holds the projects by id, writes what it is
// given into them and answers GetProject from them, the time as stored (to
// the microsecond). errs fails a method by its name, after logging the
// call; missing makes GetProject find nothing; moved, when set, is the
// workspace each project's lock reads, as if the project had moved there
// since ProjectWorkspace read it.
type fakeStore struct {
	log      *callLog
	projects map[uuid.UUID]*fakeProject
	errs     map[string]error
	missing  bool
	moved    uuid.UUID
}

func (f *fakeStore) fail(name string) error {
	if err := f.errs[name]; err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func (f *fakeStore) LockProject(ctx context.Context, id uuid.UUID) (app.LockedProject, bool, error) {
	f.log.add(ctx, "LockProject %s", id)
	if err := f.fail("LockProject"); err != nil {
		return app.LockedProject{}, false, err
	}
	p, ok := f.projects[id]
	if !ok {
		return app.LockedProject{}, false, nil
	}
	if f.moved != (uuid.UUID{}) {
		return app.LockedProject{WorkspaceID: f.moved, Archived: p.archived}, true, nil
	}
	return app.LockedProject{WorkspaceID: p.workspace, Archived: p.archived}, true, nil
}

func (f *fakeStore) ProjectWorkspace(ctx context.Context, id uuid.UUID) (uuid.UUID, bool, error) {
	f.log.add(ctx, "ProjectWorkspace %s", id)
	if err := f.fail("ProjectWorkspace"); err != nil {
		return uuid.UUID{}, false, err
	}
	p, ok := f.projects[id]
	if !ok {
		return uuid.UUID{}, false, nil
	}
	return p.workspace, true, nil
}

func (f *fakeStore) Memberships(ctx context.Context, projectID uuid.UUID, userIDs []uuid.UUID) (map[uuid.UUID]app.Membership, error) {
	f.log.add(ctx, "Memberships %s %v", projectID, userIDs)
	if err := f.fail("Memberships"); err != nil {
		return nil, err
	}
	out := map[uuid.UUID]app.Membership{}
	if p, ok := f.projects[projectID]; ok {
		for _, user := range userIDs {
			if m, ok := p.members[user]; ok {
				out[user] = m
			}
		}
	}
	return out, nil
}

func (f *fakeStore) UpdateProject(ctx context.Context, id uuid.UUID, p domain.ProjectPatch, by uuid.UUID, now time.Time) error {
	patch, _ := json.Marshal(p)
	f.log.add(ctx, "UpdateProject %s %s by %s at %s", id, patch, by, now.Format(timeFormat))
	if err := f.fail("UpdateProject"); err != nil {
		return err
	}
	f.projects[id].updated = now.Truncate(time.Microsecond)
	return nil
}

func (f *fakeStore) GetProject(ctx context.Context, id, userID uuid.UUID) (domain.Project, bool, error) {
	f.log.add(ctx, "GetProject %s for %s", id, userID)
	if err := f.fail("GetProject"); err != nil {
		return domain.Project{}, false, err
	}
	p, ok := f.projects[id]
	if !ok || f.missing {
		return domain.Project{}, false, nil
	}
	out := domain.Project{ID: id, WorkspaceID: p.workspace, UpdatedAt: p.updated}
	if p.archived {
		out.ArchivedAt = &p.updated
	}
	if m, ok := p.members[userID]; ok && m.Active {
		role := m.Role
		out.MemberRole = &role
	}
	return out, true, nil
}
````

`server/internal/modules/project/app/update_project_test.go`（新文件，193 行）：

````file server/internal/modules/project/app/update_project_test.go
package app_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newUpdate is UpdateProject over newWrites' fakes, its clock logged.
func newUpdate() (*app.UpdateProject, *writeFixture) {
	f := newWrites()
	return app.NewUpdateProject(f.store, f.locks(), f.tx, clockAt{clockNow, f.log}), f
}

// updated are the calls of bob's update of web with p, which names the
// accounts assignees as its lead and default assignee: the locks, the
// decision, the assignees' memberships when it names any, the clock, the
// change, the answer.
func updated(p domain.ProjectPatch, assignees ...uuid.UUID) []string {
	patch, _ := json.Marshal(p)
	calls := lockedDecision(bob, webID, domain.ActionUpdate)
	if len(assignees) > 0 {
		calls = append(calls, fmt.Sprintf("Memberships %s %v", webID, assignees))
	}
	return append(calls, "Now", fmt.Sprintf("UpdateProject %s %s by %s at %s", webID, patch, bob, clockNow.Format(timeFormat)),
		fmt.Sprintf("GetProject %s for %s", webID, bob))
}

// UpdateProject checks the patch, then, in one transaction and in the
// order of M3 design 3.6, locks the project, decides, checks the lead and
// the default assignee it names against their memberships of the project,
// reads the clock and changes the project, the identifier upper-cased, by
// the caller; the answer is the project read back as stored, as the caller
// sees it. A patch that names no assignee, or clears them, reads no
// membership.
func TestUpdateProject(t *testing.T) {
	tests := []struct {
		name      string
		in        domain.ProjectPatch
		stored    domain.ProjectPatch
		assignees []uuid.UUID
	}{
		{"the lead a member, the default assignee the admin", domain.ProjectPatch{Name: ptr("Site"), Identifier: ptr("site"), SetLead: true,
			LeadID: &alice, SetDefaultAssignee: true, DefaultAssigneeID: &bob},
			domain.ProjectPatch{Name: ptr("Site"), Identifier: ptr("SITE"), SetLead: true, LeadID: &alice, SetDefaultAssignee: true,
				DefaultAssigneeID: &bob}, []uuid.UUID{alice, bob}},
		{"both cleared", domain.ProjectPatch{SetLead: true, SetDefaultAssignee: true, ArchiveIn: ptr(3)},
			domain.ProjectPatch{SetLead: true, SetDefaultAssignee: true, ArchiveIn: ptr(3)}, nil},
		{"nothing", domain.ProjectPatch{}, domain.ProjectPatch{}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newUpdate()

			got, err := uc.Execute(as(bob), webID, tt.in)

			if want := updated(tt.stored, tt.assignees...); err != nil || !slices.Equal(f.log.calls, want) {
				t.Errorf("Execute() = %v, calls\n%q\nwant\n%q", err, f.log.calls, want)
			}
			if got.ID != webID || got.UpdatedAt != now || got.MemberRole == nil || *got.MemberRole != shared.RoleAdmin {
				t.Errorf("Execute() = %+v; want web as stored at %v, the caller its admin", got, now)
			}
		})
	}
}

// Refusals, each in its place, and nothing changed:
//   - the request's values, and no caller, before the transaction;
//   - a project not there, its workspace deleted while its lock waited, the
//     project of another workspace by the time it is locked, or not
//     visible: 404, before the decision or after it, whatever the patch
//     names;
//   - a project member: the Authorizer's 403, an invalid lead too;
//   - an archived project: 409, after the decision, before the assignees;
//   - a lead or a default assignee who is not an active member of the
//     project, or is its guest: one 422 naming each, after the decision.
func TestUpdateProjectRefuses(t *testing.T) {
	leads := func(lead *uuid.UUID, assignee *uuid.UUID) domain.ProjectPatch {
		return domain.ProjectPatch{SetLead: lead != nil, LeadID: lead, SetDefaultAssignee: assignee != nil, DefaultAssigneeID: assignee}
	}
	decided := func(user, project uuid.UUID) []string { return lockedDecision(user, project, domain.ActionUpdate) }
	memberships := func(accounts ...uuid.UUID) string { return fmt.Sprintf("Memberships %s %v", webID, accounts) }
	unassignable := func(fields ...string) error {
		var problems []shared.FieldError
		for _, f := range fields {
			problems = append(problems, domain.Unassignable(f))
		}
		return shared.Invalid(problems...)
	}
	tests := []struct {
		name    string
		ctx     context.Context
		project uuid.UUID
		in      domain.ProjectPatch
		want    error
		calls   []string
	}{
		{"archive_in 13", as(bob), webID, domain.ProjectPatch{ArchiveIn: ptr(13)},
			shared.Invalid(shared.FieldError{Field: "archive_in", Code: "out_of_range"}), nil},
		{"no caller", context.Background(), webID, domain.ProjectPatch{}, shared.Unauthenticated(), nil},
		{"no project", as(bob), uuid.Nil(), domain.ProjectPatch{}, domain.ErrNotFound, noProject},
		{"the workspace gone", as(bob), webID, leads(&dave, nil), domain.ErrNotFound, lockedTo(webID)},
		{"moved to another workspace", as(bob), webID, leads(&dave, nil), domain.ErrNotFound,
			append(lockedTo(webID), "LockProject "+webID.String())},
		{"not seen", as(erin), webID, leads(&dave, nil), domain.ErrNotFound, decided(erin, webID)},
		{"a project member", as(alice), webID, leads(&dave, nil), shared.Forbidden(), decided(alice, webID)},
		{"archived", as(bob), opsID, leads(&dave, nil), domain.ErrArchived, decided(bob, opsID)},
		{"a guest as the lead", as(bob), webID, leads(&carol, nil), unassignable("project_lead_id"), append(decided(bob, webID), memberships(carol))},
		{"an ended member as the lead", as(bob), webID, leads(&dave, nil), unassignable("project_lead_id"),
			append(decided(bob, webID), memberships(dave))},
		{"no member as the lead", as(bob), webID, leads(&erin, nil), unassignable("project_lead_id"), append(decided(bob, webID), memberships(erin))},
		{"a guest as the default assignee", as(bob), webID, leads(nil, &carol), unassignable("default_assignee_id"),
			append(decided(bob, webID), memberships(carol))},
		{"both", as(bob), webID, leads(&erin, &dave), unassignable("project_lead_id", "default_assignee_id"),
			append(decided(bob, webID), memberships(erin, dave))},
		{"a member lead, a guest default assignee", as(bob), webID, leads(&alice, &carol), unassignable("default_assignee_id"),
			append(decided(bob, webID), memberships(alice, carol))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newUpdate()
			f.workspaces.gone = tt.name == "the workspace gone"
			if tt.name == "moved to another workspace" {
				f.store.moved = uuid.NewV7()
			}

			got, err := uc.Execute(tt.ctx, tt.project, tt.in)

			if !sameError(err, tt.want) || got.ID != (uuid.UUID{}) || !slices.Equal(f.log.calls, tt.calls) {
				t.Errorf("Execute() = %+v, %v, calls %q; want %v, calls %q", got, err, f.log.calls, tt.want, tt.calls)
			}
		})
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after; a project the store cannot read back
// after the change is an internal error, not a 404.
func TestUpdateProjectReturnsEachFailure(t *testing.T) {
	failure := errors.New("disk full")
	in := domain.ProjectPatch{SetLead: true, LeadID: &alice}
	all := updated(in, alice)
	tests := []struct {
		name  string
		fail  func(f *writeFixture)
		calls int // how many of all's ran
	}{
		{"the project's workspace", func(f *writeFixture) { f.store.errs = map[string]error{"ProjectWorkspace": failure} }, 2},
		{"the workspace's lock", func(f *writeFixture) { f.workspaces.err = failure }, 3},
		{"the lock", func(f *writeFixture) { f.store.errs = map[string]error{"LockProject": failure} }, 4},
		{"the decision", func(f *writeFixture) { f.auth.errs = map[grantKey]error{{bob, acme.ID}: failure} }, 5},
		{"the memberships", func(f *writeFixture) { f.store.errs = map[string]error{"Memberships": failure} }, 6},
		{"the change", func(f *writeFixture) { f.store.errs = map[string]error{"UpdateProject": failure} }, 8},
		{"the answer", func(f *writeFixture) { f.store.errs = map[string]error{"GetProject": failure} }, 9},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = failure }, 9},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newUpdate()
			tt.fail(f)
			got, err := uc.Execute(as(bob), webID, in)
			if !errors.Is(err, failure) || got.ID != (uuid.UUID{}) || !slices.Equal(f.log.calls, all[:tt.calls]) {
				t.Errorf("Execute() = %+v, %v, calls %q; want %v after %q", got, err, f.log.calls, failure, all[:tt.calls])
			}
		})
	}
	uc, f := newUpdate()
	f.store.missing = true
	var e *shared.Error
	if got, err := uc.Execute(as(bob), webID, in); err == nil || errors.As(err, &e) || got.ID != (uuid.UUID{}) {
		t.Errorf("Execute() with the project gone = %+v, %v; want an internal error", got, err)
	}
}

// The store's conflicts come back as themselves: 409 of the identifier or
// the name.
func TestUpdateProjectAnswersTheStoresConflicts(t *testing.T) {
	for _, conflict := range []error{domain.ErrIdentifierTaken, domain.ErrNameTaken} {
		uc, f := newUpdate()
		f.store.errs = map[string]error{"UpdateProject": conflict}
		if _, err := uc.Execute(as(bob), webID, domain.ProjectPatch{Identifier: ptr("OPS")}); !errors.Is(err, conflict) {
			t.Errorf("Execute() = %v, want %v", err, conflict)
		}
	}
}
````

`server/internal/modules/project/app/clock_test.go`（修改，1 处）：

````old server/internal/modules/project/app/clock_test.go
	}
}

````
````new server/internal/modules/project/app/clock_test.go
	}
}

// Each write that changes an existing row reads the clock once, in its
// transaction, after its locks (its workspace's FOR SHARE first, then its
// project's), its decision and its checks, just before it writes (P2 spec
// 2.6, M3 design 3.3): a write that queued behind another on either lock
// never stamps an earlier time than the one it waited for. The clock logs
// its read among the fakes' calls.
func TestEachWriteReadsTheClockUnderItsLock(t *testing.T) {
	tests := []struct {
		name string
		run  func() (calls []string, err error)
		want []string
	}{
		{"updateProject", func() ([]string, error) {
			uc, f := newUpdate()
			_, err := uc.Execute(as(bob), webID, domain.ProjectPatch{SetLead: true, LeadID: &alice})
			return f.log.calls, err
		}, updated(domain.ProjectPatch{SetLead: true, LeadID: &alice}, alice)},
	}
	for _, tt := range tests {
		if calls, err := tt.run(); err != nil || !slices.Equal(calls, tt.want) {
			t.Errorf("%s: calls = %q, %v; want %q", tt.name, calls, err, tt.want)
		}
	}
}

````

- [ ] **Step 4: HTTP 和接线**

`server/internal/modules/project/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/project/adapter/http/handler.go
}

// UseCases are the use cases behind the module's operations.
````
````new server/internal/modules/project/adapter/http/handler.go
}

// UpdateProjectUseCase is app.UpdateProject.
type UpdateProjectUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, p domain.ProjectPatch) (domain.Project, error)
}

// UseCases are the use cases behind the module's operations.
````

````old server/internal/modules/project/adapter/http/handler.go
	CheckIdentifier CheckIdentifierUseCase
````
````new server/internal/modules/project/adapter/http/handler.go
	CheckIdentifier CheckIdentifierUseCase
	UpdateProject   UpdateProjectUseCase
````

`server/internal/modules/project/adapter/http/projects.go`（修改，1 处）：

````old server/internal/modules/project/adapter/http/projects.go
	return gen.GetProject200JSONResponse(project(p)), nil
````
````new server/internal/modules/project/adapter/http/projects.go
	return gen.GetProject200JSONResponse(project(p)), nil
}

// UpdateProject serves PATCH /api/v0/projects/{project_id}: the fields the
// body names go to the use case, project_lead_id and default_assignee_id
// set when named, to none when null.
func (h handler) UpdateProject(ctx context.Context, req gen.UpdateProjectRequestObject) (gen.UpdateProjectResponseObject, error) {
	b := req.Body
	in := domain.ProjectPatch{Name: b.Name, Description: b.Description, Identifier: b.Identifier, CycleView: b.CycleView,
		ModuleView: b.ModuleView, IssueViewsView: b.IssueViewsView, IntakeView: b.IntakeView, GuestViewAllFeatures: b.GuestViewAllFeatures,
		ArchiveIn: b.ArchiveIn, Timezone: b.Timezone}
	if b.Network != nil {
		n := domain.Network(*b.Network)
		in.Network = &n
	}
	if b.LogoProps != nil {
		logo := logoIn(*b.LogoProps)
		in.LogoProps = &logo
	}
	in.SetLead, in.LeadID = named(b.ProjectLeadID)
	in.SetDefaultAssignee, in.DefaultAssigneeID = named(b.DefaultAssigneeID)
	p, err := h.uc.UpdateProject.Execute(ctx, req.ProjectID, in)
	if err != nil {
		return nil, err
	}
	return gen.UpdateProject200JSONResponse(project(p)), nil
}

// named reports whether the body names v, and its id: nil when it is null.
func named(v nullable.Nullable[uuid.UUID]) (bool, *uuid.UUID) {
	if !v.IsSpecified() {
		return false, nil
	}
	if id, err := v.Get(); err == nil {
		return true, &id
	}
	return true, nil
````

`server/internal/modules/project/adapter/http/handler_test.go`（修改，4 处）：

````old server/internal/modules/project/adapter/http/handler_test.go
	check  *fakeCheck
````
````new server/internal/modules/project/adapter/http/handler_test.go
	check  *fakeCheck
	update *fakeUpdate
````

````old server/internal/modules/project/adapter/http/handler_test.go
}

// newServer serves the module with f; a fake left nil is an idle one.
````
````new server/internal/modules/project/adapter/http/handler_test.go
}

type fakeUpdate struct {
	calls  []string // "caller id"
	got    []domain.ProjectPatch
	answer domain.Project
	err    error
}

func (f *fakeUpdate) Execute(ctx context.Context, id uuid.UUID, p domain.ProjectPatch) (domain.Project, error) {
	f.calls = append(f.calls, caller(ctx)+" "+id.String())
	f.got = append(f.got, p)
	return f.answer, f.err
}

// newServer serves the module with f; a fake left nil is an idle one.
````

````old server/internal/modules/project/adapter/http/handler_test.go
		f.check = &fakeCheck{}
	}
````
````new server/internal/modules/project/adapter/http/handler_test.go
		f.check = &fakeCheck{}
	}
	if f.update == nil {
		f.update = &fakeUpdate{}
	}
````

````old server/internal/modules/project/adapter/http/handler_test.go
		CheckIdentifier: f.check})
````
````new server/internal/modules/project/adapter/http/handler_test.go
		CheckIdentifier: f.check, UpdateProject: f.update})
````

`server/internal/modules/project/adapter/http/update_test.go`（新文件，99 行）：

````file server/internal/modules/project/adapter/http/update_test.go
package httpadapter_test

import (
	"net/http"
	"reflect"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The fields the body names go to the use case for the caller and the
// path's project, as written: a field left out nil; the lead and the
// default assignee set when named, to none when null, unset when left out.
// The answer is 200 with the project the use case answers.
func TestUpdateProjectPassesThePatch(t *testing.T) {
	update := &fakeUpdate{answer: web}
	h := newServer(t, fakes{update: update})
	path := "/api/v0/projects/" + webID.String()
	for _, tt := range []struct {
		token, body string
	}{
		{"alice", `{"name":"Site","identifier":"site","description":"The site","network":0,"project_lead_id":"0199a2b4-0000-7000-8000-000000000002",` +
			`"default_assignee_id":null,"cycle_view":true,"module_view":false,"issue_views_view":true,"intake_view":false,` +
			`"guest_view_all_features":true,"archive_in":13,"logo_props":{"in_use":"emoji"},"timezone":"Asia/Shanghai"}`},
		{"bob", `{}`},
		{"bob", `{"project_lead_id":null,"default_assignee_id":"0199a2b4-0000-7000-8000-000000000001"}`},
	} {
		if res, body := do(t, h, request(http.MethodPatch, path, tt.token, tt.body)); res.StatusCode != http.StatusOK || body != webJSON+"\n" {
			t.Errorf("PATCH %s = %d %s, want 200 %s", tt.body, res.StatusCode, body, webJSON)
		}
	}
	private := domain.NetworkPrivate
	want := []domain.ProjectPatch{
		{Name: ptr("Site"), Identifier: ptr("site"), Description: ptr("The site"), Network: &private, SetLead: true, LeadID: &bobID,
			SetDefaultAssignee: true, CycleView: ptr(true), ModuleView: ptr(false), IssueViewsView: ptr(true), IntakeView: ptr(false),
			GuestViewAllFeatures: ptr(true), ArchiveIn: ptr(13), LogoProps: &domain.LogoProps{InUse: ptr("emoji")}, Timezone: ptr("Asia/Shanghai")},
		{},
		{SetLead: true, SetDefaultAssignee: true, DefaultAssigneeID: &aliceID},
	}
	if !reflect.DeepEqual(update.got, want) {
		t.Errorf("inputs = %+v, want %+v", update.got, want)
	}
	if want := []string{"alice " + webID.String(), "bob " + webID.String(), "bob " + webID.String()}; !slices.Equal(update.calls, want) {
		t.Errorf("calls = %q, want %q", update.calls, want)
	}
}

// A field the caller may not write, a value of another type, a null where
// only the lead and the default assignee take one: refused as bad_request
// before the use case.
func TestUpdateProjectHoldsTheBodyToItsStructure(t *testing.T) {
	update := &fakeUpdate{answer: web}
	h := newServer(t, fakes{update: update})
	for _, body := range []string{`{"archived_at":null}`, `{"workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`, `{"member_role":20}`,
		`{"id":"0199a2b4-0000-7000-8000-0000000000a1"}`, `{"name":null}`, `{"archive_in":"3"}`, `{"project_lead_id":"bob"}`,
		`{"logo_props":{"shape":"round"}}`, `[]`} {
		if res, got := do(t, h, request(http.MethodPatch, "/api/v0/projects/"+webID.String(), "alice", body)); res.StatusCode != http.StatusBadRequest {
			t.Errorf("PATCH %s = %d %s, want 400", body, res.StatusCode, got)
		}
	}
	if len(update.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", update.calls)
	}
}

// The use case's refusals, as the contract declares them.
func TestUpdateProjectRefusals(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"no project", domain.ErrNotFound, http.StatusNotFound,
			`{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`},
		{"a project member", shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"archived", domain.ErrArchived, http.StatusConflict,
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
		{"a lead and a default assignee who may not be", shared.Invalid(domain.Unassignable("project_lead_id"),
			domain.Unassignable("default_assignee_id")), http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"project_lead_id","code":"not_allowed","message":"must be an active member of the project who is not its guest"},` +
				`{"field":"default_assignee_id","code":"not_allowed","message":"must be an active member of the project who is not its guest"}]}`},
		{"a taken identifier", domain.ErrIdentifierTaken, http.StatusConflict,
			`{"status":409,"code":"project.identifier_taken","title":"Conflict","detail":"A project of the workspace has this identifier."}`},
		{"a taken name", domain.ErrNameTaken, http.StatusConflict,
			`{"status":409,"code":"project.name_taken","title":"Conflict","detail":"A project of the workspace has this name."}`},
	}
	for _, tt := range tests {
		h := newServer(t, fakes{update: &fakeUpdate{err: tt.err}})
		res, body := do(t, h, request(http.MethodPatch, "/api/v0/projects/"+webID.String(), "alice", `{"name":"Site"}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: PATCH = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
````

`server/internal/modules/project/module.go`（修改，3 处）：

````old server/internal/modules/project/module.go
// brings listing, creating and reading projects and checking an identifier,
// carries out the workspace module's cascades on the projects
````
````new server/internal/modules/project/module.go
// brings listing, creating, reading and changing projects and checking an
// identifier, carries out the workspace module's cascades on the projects
````

````old server/internal/modules/project/module.go
	store := postgresadapter.New(d.Pool)
````
````new server/internal/modules/project/module.go
	store := postgresadapter.New(d.Pool)
	locks := app.NewLocks(store, d.Workspaces, d.Authorizer)
````

````old server/internal/modules/project/module.go
		CheckIdentifier: app.NewCheckProjectIdentifier(d.Workspaces, store, d.Authorizer),
````
````new server/internal/modules/project/module.go
		CheckIdentifier: app.NewCheckProjectIdentifier(d.Workspaces, store, d.Authorizer),
		UpdateProject:   app.NewUpdateProject(store, locks, d.Tx, d.Clock),
````

- [ ] **Step 5: 新码的文案**

`web/apps/web/helpers/authentication.helper.ts`（修改，1 处）：

````old web/apps/web/helpers/authentication.helper.ts
  "project.not_found": "auth.errors.project_not_found",
````
````new web/apps/web/helpers/authentication.helper.ts
  "project.not_found": "auth.errors.project_not_found",
  "project.archived": "auth.errors.project_archived",
````

`web/packages/i18n/src/locales/en/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/auth.json
      "project_not_found": "The project does not exist, or you cannot see it.",
````
````new web/packages/i18n/src/locales/en/auth.json
      "project_not_found": "The project does not exist, or you cannot see it.",
      "project_archived": "The project is archived. Restore it to change it.",
````

`web/packages/i18n/src/locales/zh-CN/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/auth.json
      "project_not_found": "项目不存在，或你看不到它。",
````
````new web/packages/i18n/src/locales/zh-CN/auth.json
      "project_not_found": "项目不存在，或你看不到它。",
      "project_archived": "项目已归档，恢复之后才能修改。",
````

- [ ] **Step 6: 矩阵和组合出的测试**

`server/internal/bootstrap/permission_matrix_project_test.go`（修改，3 处）：

````old server/internal/bootstrap/permission_matrix_project_test.go
var cellProjectNotFound = cell{http.StatusNotFound, "project.not_found"}
````
````new server/internal/bootstrap/permission_matrix_project_test.go
var (
	cellProjectNotFound = cell{http.StatusNotFound, "project.not_found"}
	cellProjectArchived = cell{http.StatusConflict, "project.archived"}
)
````

````old server/internal/bootstrap/permission_matrix_project_test.go
			cells: map[caller]cell{callerArchivedAdmin: cellOK}, check: readsItsProject},
````
````new server/internal/bootstrap/permission_matrix_project_test.go
			cells: map[caller]cell{callerArchivedAdmin: cellOK}, check: readsItsProject},
		// The project's admins, and its members who are the workspace's
		// admins (M3 design 3.4): PM+WA is a member of the project and WA- is
		// not, so the row parts the workspace's admin who joined from the one
		// who did not.
		{op: "updateProject", write: true, columns: projectColumns, request: toProject(http.MethodPatch, "", `{"name":"Renamed"}`),
			cells: ofProject(cellOK, cellForbidden, cellForbidden, cellOK, cellForbidden, cellForbidden), check: renamesItsProject},
		// A lead who is no member of the project is refused after the
		// decision: who may not change the project learns nothing of the lead.
		{op: "updateProject", variant: "a lead who is no member", write: true, columns: projectColumns,
			request: toProject(http.MethodPatch, "", `{"project_lead_id":"`+uuid.Nil().String()+`"}`),
			cells:   ofProject(cellValidationFailed, cellForbidden, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden)},
		{op: "updateProject", variant: "archived", write: true, columns: archivedColumns, request: toProject(http.MethodPatch, "", `{"name":"Renamed"}`),
			cells: map[caller]cell{callerArchivedAdmin: cellProjectArchived}},
	}
}

// memberRoles are the project roles of the columns that are their
// project's members.
var memberRoles = map[caller]int{callerProjectAdmin: 20, callerProjectMember: 15, callerProjectGuest: 5, callerMemberAndAdmin: 15,
	callerArchivedAdmin: 20}

// renamesItsProject: the column's project, renamed, with the caller's role
// in it.
func renamesItsProject(t *testing.T, c caller, s seeded, answer string) {
	var p struct {
		ID         uuid.UUID `json:"id"`
		Name       string    `json:"name"`
		MemberRole *int      `json:"member_role"`
	}
	decodeAnswer(t, answer, &p)
	if p.ID != s.project(projectOf(c)) || p.Name != "Renamed" || p.MemberRole == nil || *p.MemberRole != memberRoles[c] {
		t.Errorf("%s renames %s; want %s renamed, his role %d", c, answer, projectOf(c), memberRoles[c])
````

````old server/internal/bootstrap/permission_matrix_project_test.go
	roles := map[caller]int{callerProjectAdmin: 20, callerProjectMember: 15, callerProjectGuest: 5, callerMemberAndAdmin: 15,
		callerArchivedAdmin: 20}
	role, member := roles[c]
````
````new server/internal/bootstrap/permission_matrix_project_test.go
	role, member := memberRoles[c]
````

`server/internal/bootstrap/project_writes_test.go`（新文件，71 行）：

````file server/internal/bootstrap/project_writes_test.go
package bootstrap

import (
	"context"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// Each write on a project as bootstrap wires it stamps the rows it writes
// with the time of its request, from the clock project.New takes, and with
// its caller (M3 design 3.6): alice, acme's admin, writes on her project
// Web, one write after another; the statement of each reads the rows it
// wrote, by $1 Web's id and $2 alice's.
func TestTheWritesOnAProjectStampTheirRequest(t *testing.T) {
	contract := apitest.Load(t)
	dbURL := pgtest.NewDatabase(t)
	base := startApp(t, testConfig(t, dbURL, false), migrations.FS())
	pool := openPool(t, dbURL)
	alice := registerAccount(t, contract, base, "alice@example.com").AccessToken
	aliceID := accountID(t, contract, base, alice)
	if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", alice, `{"name":"Acme","slug":"acme"}`); status != http.StatusCreated {
		t.Fatalf("creating acme = %d %s", status, body)
	}
	status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces/acme/projects", alice, `{"name":"Web","identifier":"WEB"}`)
	if status != http.StatusCreated {
		t.Fatalf("creating Web = %d %s", status, body)
	}
	var web struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, body, &web)
	for _, w := range []struct {
		name, method, path, body string
		status                   int
		stamps                   string // the rows written: their updated_at, and whether alice wrote them
	}{
		{"updateProject", http.MethodPatch, "/api/v0/projects/" + web.ID.String(), `{"name":"Site"}`, http.StatusOK,
			"SELECT updated_at, updated_by_id = $2 FROM projects WHERE id = $1"},
	} {
		before := time.Now().Truncate(time.Microsecond)
		status, body := call(t, contract, w.method, base+w.path, alice, w.body)
		after := time.Now()
		if status != w.status {
			t.Fatalf("%s = %d %s, want %d", w.name, status, body, w.status)
		}
		rows, err := pool.Query(context.Background(), w.stamps, web.ID, aliceID)
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for rows.Next() {
			var at time.Time
			var hers bool
			if err := rows.Scan(&at, &hers); err != nil {
				t.Fatal(err)
			}
			if n++; at.Before(before) || at.After(after) || !hers {
				t.Errorf("%s wrote a row at %v, by alice %v; want within the request, %v to %v, by alice", w.name, at, hers, before, after)
			}
		}
		if err := rows.Err(); err != nil || n == 0 {
			t.Errorf("%s: %d rows written, %v; want one at least", w.name, n, err)
		}
	}
}
````

- [ ] **Step 7: 测试、lint、前端检查**

Run: `go -C server test -count=1 ./internal/modules/project/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix$|TestThePermissionMatrixCoversEveryOperation|TestMatrixViolationsCatchesEach|TestEveryColumnCallsAsARegisteredAccount|TestEveryActionHasARuleAndEveryRuleAnAction|TestAPIRoutesAreTheContractsOperations|TestOperationsThatNeedATokenAnswer401WithoutOne|TestBodiesThatBreakTheStructureAnswer400|TestTheWritesOnAProjectStampTheirRequest' ./internal/bootstrap/`
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

- [ ] **Step 8: 提交**

```bash
git add api/modules/project.yaml server/internal/bootstrap/permission_matrix_project_test.go server/internal/bootstrap/project_writes_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/project/adapter/http/handler.go server/internal/modules/project/adapter/http/handler_test.go server/internal/modules/project/adapter/http/projects.go server/internal/modules/project/adapter/http/update_test.go server/internal/modules/project/app/clock_test.go server/internal/modules/project/app/fakes_write_test.go server/internal/modules/project/app/lock.go server/internal/modules/project/app/ports.go server/internal/modules/project/app/update_project.go server/internal/modules/project/app/update_project_test.go server/internal/modules/project/domain/actions.go server/internal/modules/project/module.go web/apps/web/helpers/authentication.helper.ts web/packages/i18n/src/locales/en/auth.json web/packages/i18n/src/locales/zh-CN/auth.json api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/bodyshape.gen.go server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P4b): updateProject

PATCH /api/v0/projects/{project_id}, for the project's admins and its
members who are the workspace's admins. The patch is checked before the
transaction; then Locks, the one path every write on a project takes,
reads the project's workspace without a lock, locks the workspace's row
FOR SHARE and the project FOR NO KEY UPDATE, and decides under both
(M3 design 3.6 convention 2); an archived project is refused (409
project.archived), and a lead or default assignee who is not an active
member of the project, or is its guest, refused after the decision
(422), so who may not change the project learns nothing of them. The
clock is read under the locks.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| 规则给项目成员；给不是成员的工作区管理员（`LevelWorkspace`） | `TestPermissionMatrix`（PM、WA- 两格） | 组合 |
| 看不到的项目答 403 而不是 404 | `TestPermissionMatrix` | 组合 |
| 已归档的项目照改 | `TestPermissionMatrix`（已归档的一格）；P4（Task 15 起） | 组合；端到端 |
| 不检查负责人、默认负责人 | `TestPermissionMatrix`（负责人不是成员的行） | 组合 |
| 已结束的成员可以作负责人 | `TestTheLeadAndTheDefaultAssigneeAreActiveMembersWhoAreNoGuests`（Task 12 起） | 组合 |
| 负责人在判定之前检查 | `TestPermissionMatrix`（负责人不是成员的行：不能改项目的列答 422 而不是 403、404） | 组合 |
| 没有调用者时也读 | `TestUpdateProjectRefuses` | 单元 |
| 项目的工作区、工作区的锁、项目的锁、判定、成员关系、改、读回、提交的失败被吞掉；改重试一次；`Authorizer` 的失败被吞掉、答成 403 | `TestUpdateProjectReturnsEachFailure`（`Locks` 的另有 `TestArchiveProjectReturnsEachFailure`、`TestDeleteProjectReturnsEachFailure`，Task 4、6 起；`decide` 的两个另有 `TestListProjectMembersRefuses`，Task 9 起） | 单元 |
| handler 吞掉用例的失败 | `TestUpdateProjectRefusals` | 单元 |
| 契约不声明 `project.archived` | `TestPermissionMatrix`、`TestUpdateProjectRefusals`（两者都按契约核对答案） | 组合；单元 |
| 每个项目锁在写的事务之外取 | `TestTwoWritesOnAProjectSerialize`、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 14 起） | 组合 |
| 修改先判定后锁 | `TestUpdateProject`；`TestAProjectWriteAndADemotionSerialize`（Task 14 起） | 单元；组合 |
| `Locks` 不锁工作区 | `TestUpdateProject`、`TestEachWriteReadsTheClockUnderItsLock`；`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 4 起）；三个交错测试、`TestAWorkspacesDeletionWaitsForTheWritesOnItsProjects`、`TestAddingSeveralMembersAndDeletingTheWorkspaceSerialize`（Task 14 起：删除工作区 40P01） | 单元；组合 |
| 工作区在项目之后锁 | `TestUpdateProject`、`TestEachWriteReadsTheClockUnderItsLock`；`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 4 起：等工作区时已持有项目） | 单元；组合 |
| 只有修改不锁工作区 | `TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 4 起）；`TestAProjectWriteAndADemotionSerialize`、`TestAWorkspacesDeletionWaitsForTheWritesOnItsProjects`（Task 14 起） | 组合 |
| 工作区的锁的失败答成找到 | `TestUpdateProjectReturnsEachFailure`（添加、加入的另有，Task 11、13 起） | 单元 |
| 等待期间被删除的工作区答成找到；项目的锁读到的工作区不与先读的核对 | `TestUpdateProjectRefuses`（组合一层看不到：删除工作区在同一个事务里删除它的项目，项目的锁随之找不到；没有写能把项目移到别的工作区，spec 第 3 节第 10 条） | 单元 |
| 修改在锁之前读时钟 | `TestEachWriteReadsTheClockUnderItsLock`、`TestUpdateProject`（组合一层只核对时刻在请求之内，预检认可的类别） | 单元 |
| `project.New` 的 `Locks` 接一个不加锁的工作区端口 | `TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 4 起）；`TestAWorkspacesDeletionWaitsForTheWritesOnItsProjects`、`TestAddingSeveralMembersAndDeletingTheWorkspaceSerialize`（Task 14 起） | 组合 |
| `project.New` 的接线 | 见 Task 8 的表 | 组合 |

**Done when:** 矩阵三行（25 格）通过；修改经 `Locks` 先锁工作区、再锁项目，在锁之后判定、在判定之后检查负责人；`project.archived` 的文案在前端；`apitest.Main` 两个方向通过。

---

### Task 4: `archiveProject`、`unarchiveProject`

**Files:**
- Create: `server/internal/bootstrap/project_write_locks_test.go`、`server/internal/modules/project/adapter/http/archive_test.go`、`server/internal/modules/project/app/archive_project.go`、`server/internal/modules/project/app/archive_project_test.go`
- Modify: `api/modules/project.yaml`、`api/openapi.yaml`、`server/internal/bootstrap/permission_matrix_project_test.go`、`server/internal/bootstrap/permission_matrix_seed_test.go`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/bootstrap/project_writes_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/project/adapter/http/handler.go`、`server/internal/modules/project/adapter/http/handler_test.go`、`server/internal/modules/project/adapter/http/projects.go`、`server/internal/modules/project/adapter/postgres/failures_test.go`、`server/internal/modules/project/adapter/postgres/queries/projects.sql`、`server/internal/modules/project/adapter/postgres/update.go`、`server/internal/modules/project/adapter/postgres/update_test.go`、`server/internal/modules/project/app/clock_test.go`、`server/internal/modules/project/app/fakes_write_test.go`、`server/internal/modules/project/app/ports.go`、`server/internal/modules/project/domain/actions.go`、`server/internal/modules/project/module.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`server/internal/modules/project/adapter/postgres/gen/projects.sql.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.5，M3 设计 3.4、3.19）：`POST /api/v0/projects/{project_id}/archive`、`/unarchive`，200 `Project`；码 `[project.not_found, forbidden]`；操作名 `project.archive`、`project.unarchive`，规则同 `project.update`。
- `app.NewArchiveProject`、`app.NewUnarchiveProject(projects ProjectArchiver, locks Locks, tx, clock)`：一个事务：`locks.lockAndDecide`（工作区 `FOR SHARE` → 项目 `FOR NO KEY UPDATE` → 判定，Task 3）→ 锁下读时钟 → `SetArchived(id, archive, by, now)` → `answer`。已归档的再归档取新的时刻，未归档的恢复照样成功（Plane `views/project/base.py:427-441`）。
- 存储：`SetArchived`：`archived_at` 是给的时刻或 `null`，`updated_by_id`、`updated_at`；别的列不动。
- `ProjectArchiver` 不嵌入 `ProjectLocker`（锁在 `Locks` 里）。
- 矩阵：`projectSeed.archive` 经存储归档 `acme/archived`，`standIns` 不再用 SQL 归档（Task 1 留下的替身由第一个存储替换）。
- `bootstrap/project_writes_test.go` 的 `createdProject(t, contract, base, token, slug, name, identifier)`：经接口建项目，取回它的 id（`TestTheWritesOnAProjectStampTheirRequest` 改用它）。

**Tests:**
- `app/archive_project_test.go`：`TestArchiveProject`（归档 web、再归档 ops、恢复 ops、恢复未归档的 web：完整的调用记录和回答）；`TestArchiveProjectRefuses`（两个动作各四种：没有调用者、没有的项目、看不到、项目成员）；`TestArchiveProjectReturnsEachFailure`（锁、判定、写、读回、提交；读回找不到是内部错误）。
- `adapter/postgres/update_test.go`：`TestSetArchived`（归档、再归档取新时刻、恢复为 `null`，每次恰好改三列，别的项目每一列不变）。
- `adapter/http/archive_test.go`：`TestArchiveAndUnarchiveProject`（各自的路由只调各自的用例；两种拒绝）。
- `bootstrap`：矩阵四行（两个动作各 12 格加已归档 1 格，答案核对归档的时刻等于最后修改的时刻）；`TestTheWritesOnAProjectStampTheirRequest` 加归档、再归档、恢复三行。
- `bootstrap/project_write_locks_test.go`（新，M3 设计 3.6 约定二，方案 E）：`TestEachWriteOnAProjectSharesItsWorkspaceFirst`：经组合出的 app，alice 在 acme 的 Web、Ops 上把每个项目级的写各执行一次（修改、归档、恢复三行；此后每个写的 Task 加一行）。Web：另一个事务持有 acme 行的 `FOR NO KEY UPDATE`（每个连带都这样持有），写在 `workspaces` 上等（`pgtest.WaitForLockWaitOn`），等待期间不持有 Web（`FOR UPDATE NOWAIT` 成功）。Ops：另一个事务持有 Ops 行的 `FOR NO KEY UPDATE`，写在 `projects` 上等，等待期间在自己的事务里持有 acme 的 `FOR SHARE`、不更强（`FOR NO KEY UPDATE NOWAIT` 失败、`FOR SHARE NOWAIT` 成功：同一个工作区的别的写照样能取）。放开之后各自照常回答（按契约核对）。`projectWrites` 的操作与矩阵中写在项目一级（`projectColumns`）的行逐个对上：契约里一个新的项目级的写没有自己的行，测试在连数据库之前失败（P5、P7 的写由此必须加行）。

- [ ] **Step 1: 接口描述**

`api/modules/project.yaml`（修改，1 处）：

````old api/modules/project.yaml
          description: The project as changed, as the caller sees it.
````
````new api/modules/project.yaml
          description: The project as changed, as the caller sees it.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Project'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/projects/{project_id}/archive:
    parameters:
      - $ref: '#/components/parameters/ProjectID'
    post:
      operationId: archiveProject
      tags: [project]
      summary: Archive a project
      description: >-
        For the project's admins, and its members who are the workspace's
        admins. The project is archived as of the request: listProjects lists
        it among the archived ones, and it cannot be changed until it is
        unarchived. An archived project archived again takes the new time. A
        project that does not exist, is deleted, or that the caller does not
        see answers project.not_found; one he sees but may not archive,
        forbidden. The role is decided after the project row is locked.
      security: [{bearer: []}]
      x-problem-codes: [project.not_found, forbidden]
      responses:
        '200':
          description: The project as archived, as the caller sees it.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Project'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/projects/{project_id}/unarchive:
    parameters:
      - $ref: '#/components/parameters/ProjectID'
    post:
      operationId: unarchiveProject
      tags: [project]
      summary: Unarchive a project
      description: >-
        For the project's admins, and its members who are the workspace's
        admins. The project is no longer archived (archived_at null), and
        can be changed again; one that is not archived stays so. A project
        that does not exist, is deleted, or that the caller does not see
        answers project.not_found; one he sees but may not unarchive,
        forbidden. The role is decided after the project row is locked.
      security: [{bearer: []}]
      x-problem-codes: [project.not_found, forbidden]
      responses:
        '200':
          description: The project as unarchived, as the caller sees it.
````

`api/openapi.yaml`（修改，1 处）：

````old api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1projects~1{project_id}'
````
````new api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1projects~1{project_id}'
  /api/v0/projects/{project_id}/archive:
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1projects~1{project_id}~1archive'
  /api/v0/projects/{project_id}/unarchive:
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1projects~1{project_id}~1unarchive'
````

- [ ] **Step 2: 查询**

`server/internal/modules/project/adapter/postgres/queries/projects.sql`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/queries/projects.sql
WHERE p.id = sqlc.arg(id);

````
````new server/internal/modules/project/adapter/postgres/queries/projects.sql
WHERE p.id = sqlc.arg(id);

-- name: SetArchived :exec
-- archiveProject and unarchiveProject, under the project's FOR NO KEY UPDATE (M3 design 3.19): archived_at becomes the
-- moment given, or null. Archiving an archived project stamps it again, as Plane's does (views/project/base.py:427-441).
UPDATE projects
SET archived_at   = CASE WHEN sqlc.arg(archived)::boolean THEN sqlc.arg(now)::timestamptz END,
    updated_by_id = sqlc.arg(updated_by)::uuid,
    updated_at    = sqlc.arg(now)
WHERE id = sqlc.arg(id);

````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `97546a6ad9a74bfaf1fd5e7030bab297ec9897b0826764272227208be28e3357` | 2127 | `api/dist/openapi.yaml` |
| `ed988d7ba1e3a08b32d674a5b3b0279e2861d47f297de58834d66b1562cc283a` | 1227 | `server/internal/modules/project/adapter/http/gen/server.gen.go` |
| `a985f81e44c6f48f9b1ab351ea9eb0ffee529cf9f62808138c75398ce2433e8f` | 390 | `server/internal/modules/project/adapter/postgres/gen/projects.sql.go` |
| `2eccb5e7299e4b96f3759083958565cf9b4819d6c505570ef56338c8df5d3590` | 2255 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/server.gen.go server/internal/modules/project/adapter/postgres/gen/projects.sql.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 3: 存储**

`server/internal/modules/project/app/ports.go`（修改，1 处）：

````old server/internal/modules/project/app/ports.go
}

// ProjectRow is a project to insert: checked values, its id, its creator
````
````new server/internal/modules/project/app/ports.go
}

// ProjectArchiver is archiveProject's and unarchiveProject's repository.
// Each method runs in the transaction ctx carries.
type ProjectArchiver interface {
	ProjectReader
	// SetArchived archives the project id at now, or unarchives it, by the
	// account by.
	SetArchived(ctx context.Context, id uuid.UUID, archived bool, by uuid.UUID, now time.Time) error
}

// ProjectRow is a project to insert: checked values, its id, its creator
````

`server/internal/modules/project/adapter/postgres/update.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/update.go
	return taken(fmt.Sprintf("update project %s", id), s.queries(ctx).UpdateProject(ctx, arg))
}

````
````new server/internal/modules/project/adapter/postgres/update.go
	return taken(fmt.Sprintf("update project %s", id), s.queries(ctx).UpdateProject(ctx, arg))
}

// SetArchived archives the project id at now, or unarchives it, by the
// account by (app.ProjectArchiver).
func (s *Store) SetArchived(ctx context.Context, id uuid.UUID, archived bool, by uuid.UUID, now time.Time) error {
	if err := s.queries(ctx).SetArchived(ctx, gen.SetArchivedParams{ID: id, Archived: archived, UpdatedBy: by, Now: now}); err != nil {
		return fmt.Errorf("set project %s archived %v: %w", id, archived, err)
	}
	return nil
}

````

`server/internal/modules/project/adapter/postgres/update_test.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/update_test.go
		t.Errorf("archive_in 13: UpdateProject() = %v; want the violation of projects_archive_in_check, not a domain error", err)
	}
}

````
````new server/internal/modules/project/adapter/postgres/update_test.go
		t.Errorf("archive_in 13: UpdateProject() = %v; want the violation of projects_archive_in_check, not a domain error", err)
	}
}

// SetArchived archives the project at the moment given, by the account
// given, and touches no other column: archived again, it takes the new
// moment; unarchived, archived_at is null. Every other project keeps every
// column: one of the same workspace, archived, one of another.
func TestSetArchived(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", ops, now)
	newProject(t, s, beta, "Web", "WEB", alice)
	others := tableRows(t, pool, "projects", web)
	stamp := func(at time.Time) string { return `"` + at.Format("2006-01-02T15:04:05.999999") + `+00:00"` }
	for _, step := range []struct {
		archived bool
		by       uuid.UUID
		at       time.Time
	}{{true, bob, now.Add(time.Hour)}, {true, alice, now.Add(2 * time.Hour)}, {false, bob, now.Add(3 * time.Hour)}} {
		before := columns(t, pool, "projects", web)
		if err := s.SetArchived(context.Background(), web, step.archived, step.by, step.at); err != nil {
			t.Fatal(err)
		}
		archivedAt := "null"
		if step.archived {
			archivedAt = stamp(step.at)
		}
		want := changed(before, map[string]string{"archived_at": archivedAt, "updated_at": stamp(step.at), "updated_by_id": `"` + step.by.String() + `"`})
		if got := columns(t, pool, "projects", web); !maps.Equal(got, want) {
			t.Errorf("archived %v by %s at %v: %v\nwant %v", step.archived, step.by, step.at, got, want)
		}
	}
	if after := tableRows(t, pool, "projects", web); after != others {
		t.Errorf("the other projects:\n%s\nwant\n%s", after, others)
	}
}

````

`server/internal/modules/project/adapter/postgres/failures_test.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("UpdateProject() = %v; want context.Canceled", err)
	}
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("UpdateProject() = %v; want context.Canceled", err)
	}
	if err := s.SetArchived(cancelled, web, true, alice, now); !failed(err) {
		t.Errorf("SetArchived() = %v; want context.Canceled", err)
	}
````

- [ ] **Step 4: 操作名、规则、用例**

`server/internal/modules/project/domain/actions.go`（修改，2 处）：

````old server/internal/modules/project/domain/actions.go
	ActionUpdate shared.Action = "project.update"
````
````new server/internal/modules/project/domain/actions.go
	ActionUpdate shared.Action = "project.update"
	// ActionArchive is archiving a project: archiveProject.
	ActionArchive shared.Action = "project.archive"
	// ActionUnarchive is unarchiving a project: unarchiveProject.
	ActionUnarchive shared.Action = "project.unarchive"
````

````old server/internal/modules/project/domain/actions.go
	return []shared.Action{ActionList, ActionCreate, ActionRead, ActionCheckIdentifier, ActionUpdate}
````
````new server/internal/modules/project/domain/actions.go
	return []shared.Action{ActionList, ActionCreate, ActionRead, ActionCheckIdentifier, ActionUpdate, ActionArchive,
		ActionUnarchive}
````

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"project.update": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
````
````new server/internal/modules/access/domain/rules.go
	"project.update": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	// As project.update (M3 design 3.4, §12 P4b).
	"project.archive":   {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	"project.unarchive": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
````
````new server/internal/modules/access/domain/rules_test.go
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
	"project.archive": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
	"project.unarchive": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible,
		invisible, forbidden, invisible, invisible, invisible, forbidden, forbidden},
````

`server/internal/modules/project/app/archive_project.go`（新文件，62 行）：

````file server/internal/modules/project/app/archive_project.go
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ArchiveProject archives a project, or unarchives it: POST
// /api/v0/projects/{project_id}/archive and /unarchive (M3 design 3.4,
// 3.19).
type ArchiveProject struct {
	projects ProjectArchiver
	locks    Locks
	tx       shared.TxManager
	clock    Clock
	archive  bool
}

// NewArchiveProject returns the use case that archives.
func NewArchiveProject(projects ProjectArchiver, locks Locks, tx shared.TxManager, clock Clock) *ArchiveProject {
	return &ArchiveProject{projects: projects, locks: locks, tx: tx, clock: clock, archive: true}
}

// NewUnarchiveProject returns the use case that unarchives.
func NewUnarchiveProject(projects ProjectArchiver, locks Locks, tx shared.TxManager, clock Clock) *ArchiveProject {
	return &ArchiveProject{projects: projects, locks: locks, tx: tx, clock: clock}
}

// Execute, in one transaction (M3 design 3.6): the project's locks (Locks:
// its workspace FOR SHARE, then the project FOR NO KEY UPDATE) and the
// decision on project.archive or project.unarchive; archived_at set to the
// clock's time read under the locks, or cleared. An archived project
// archived again takes the new time, as Plane's does. The answer is the
// project as stored, as the caller sees it.
func (u *ArchiveProject) Execute(ctx context.Context, id uuid.UUID) (domain.Project, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Project{}, err
	}
	action := domain.ActionUnarchive
	if u.archive {
		action = domain.ActionArchive
	}
	var stored domain.Project
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		if _, err := u.locks.lockAndDecide(ctx, actor, write{project: id, action: action}); err != nil {
			return err
		}
		if err := u.projects.SetArchived(ctx, id, u.archive, actor.UserID, u.clock.Now()); err != nil {
			return err
		}
		stored, err = answer(ctx, u.projects, id, actor.UserID)
		return err
	})
	if err != nil {
		return domain.Project{}, err
	}
	return stored, nil
}
````

`server/internal/modules/project/app/fakes_write_test.go`（修改，1 处）：

````old server/internal/modules/project/app/fakes_write_test.go
}

func (f *fakeStore) GetProject(ctx context.Context, id, userID uuid.UUID) (domain.Project, bool, error) {
````
````new server/internal/modules/project/app/fakes_write_test.go
}

func (f *fakeStore) SetArchived(ctx context.Context, id uuid.UUID, archived bool, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "SetArchived %s %v by %s at %s", id, archived, by, now.Format(timeFormat))
	if err := f.fail("SetArchived"); err != nil {
		return err
	}
	f.projects[id].archived, f.projects[id].updated = archived, now.Truncate(time.Microsecond)
	return nil
}

func (f *fakeStore) GetProject(ctx context.Context, id, userID uuid.UUID) (domain.Project, bool, error) {
````

`server/internal/modules/project/app/archive_project_test.go`（新文件，141 行）：

````file server/internal/modules/project/app/archive_project_test.go
package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newArchive is ArchiveProject, or UnarchiveProject when archive is false,
// over newWrites' fakes, its clock logged.
func newArchive(archive bool) (*app.ArchiveProject, *writeFixture) {
	f := newWrites()
	if archive {
		return app.NewArchiveProject(f.store, f.locks(), f.tx, clockAt{clockNow, f.log}), f
	}
	return app.NewUnarchiveProject(f.store, f.locks(), f.tx, clockAt{clockNow, f.log}), f
}

// archived are the calls of bob's archive of project, or unarchive: the
// locks, the decision, the clock, the change, the answer.
func archived(project uuid.UUID, archive bool) []string {
	action := domain.ActionUnarchive
	if archive {
		action = domain.ActionArchive
	}
	return append(lockedDecision(bob, project, action), "Now",
		fmt.Sprintf("SetArchived %s %v by %s at %s", project, archive, bob, clockNow.Format(timeFormat)),
		fmt.Sprintf("GetProject %s for %s", project, bob))
}

// ArchiveProject, in one transaction and in the order of M3 design 3.6,
// takes the project's locks, decides, reads the clock and archives it, by the
// caller; UnarchiveProject the same, and clears the time. An archived
// project archives again, and an unarchived one unarchives: each is the
// same change, as Plane's. The answer is the project read back as stored.
func TestArchiveProject(t *testing.T) {
	tests := []struct {
		name     string
		project  uuid.UUID
		archive  bool
		archived bool
	}{
		{"archive web", webID, true, true},
		{"archive ops, archived already", opsID, true, true},
		{"unarchive ops", opsID, false, false},
		{"unarchive web, not archived", webID, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newArchive(tt.archive)

			got, err := uc.Execute(as(bob), tt.project)

			if want := archived(tt.project, tt.archive); err != nil || !slices.Equal(f.log.calls, want) {
				t.Errorf("Execute() = %v, calls\n%q\nwant\n%q", err, f.log.calls, want)
			}
			if got.ID != tt.project || (got.ArchivedAt != nil) != tt.archived || (tt.archived && !got.ArchivedAt.Equal(now)) {
				t.Errorf("Execute() = %+v; want the project, archived %v at %v", got, tt.archived, now)
			}
		})
	}
}

// Refusals, each in its place, and nothing changed: no caller, before the
// transaction; a project not there, or not visible: 404; a project member:
// the Authorizer's 403.
func TestArchiveProjectRefuses(t *testing.T) {
	for _, archive := range []bool{true, false} {
		action := domain.ActionUnarchive
		if archive {
			action = domain.ActionArchive
		}
		tests := []struct {
			name    string
			ctx     context.Context
			project uuid.UUID
			want    error
			calls   []string
		}{
			{"no caller", context.Background(), webID, shared.Unauthenticated(), nil},
			{"no project", as(bob), uuid.Nil(), domain.ErrNotFound, noProject},
			{"not seen", as(erin), webID, domain.ErrNotFound, lockedDecision(erin, webID, action)},
			{"a project member", as(alice), webID, shared.Forbidden(), lockedDecision(alice, webID, action)},
		}
		for _, tt := range tests {
			t.Run(fmt.Sprintf("%s %s", action, tt.name), func(t *testing.T) {
				uc, f := newArchive(archive)

				got, err := uc.Execute(tt.ctx, tt.project)

				if !sameError(err, tt.want) || got.ID != (uuid.UUID{}) || !slices.Equal(f.log.calls, tt.calls) {
					t.Errorf("Execute() = %+v, %v, calls %q; want %v, calls %q", got, err, f.log.calls, tt.want, tt.calls)
				}
			})
		}
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after; a project the store cannot read back
// after the change is an internal error, not a 404.
func TestArchiveProjectReturnsEachFailure(t *testing.T) {
	failure := errors.New("disk full")
	for _, archive := range []bool{true, false} {
		all := archived(webID, archive)
		tests := []struct {
			name  string
			fail  func(f *writeFixture)
			calls int // how many of all's ran
		}{
			{"the lock", func(f *writeFixture) { f.store.errs = map[string]error{"LockProject": failure} }, 4},
			{"the decision", func(f *writeFixture) { f.auth.errs = map[grantKey]error{{bob, acme.ID}: failure} }, 5},
			{"the change", func(f *writeFixture) { f.store.errs = map[string]error{"SetArchived": failure} }, 7},
			{"the answer", func(f *writeFixture) { f.store.errs = map[string]error{"GetProject": failure} }, 8},
			{"the commit", func(f *writeFixture) { f.tx.commitErr = failure }, 8},
		}
		for _, tt := range tests {
			t.Run(fmt.Sprintf("archive %v, %s", archive, tt.name), func(t *testing.T) {
				uc, f := newArchive(archive)
				tt.fail(f)
				got, err := uc.Execute(as(bob), webID)
				if !errors.Is(err, failure) || got.ID != (uuid.UUID{}) || !slices.Equal(f.log.calls, all[:tt.calls]) {
					t.Errorf("Execute() = %+v, %v, calls %q; want %v after %q", got, err, f.log.calls, failure, all[:tt.calls])
				}
			})
		}
		uc, f := newArchive(archive)
		f.store.missing = true
		var e *shared.Error
		if got, err := uc.Execute(as(bob), webID); err == nil || errors.As(err, &e) || got.ID != (uuid.UUID{}) {
			t.Errorf("Execute() with the project gone = %+v, %v; want an internal error", got, err)
		}
	}
}
````

`server/internal/modules/project/app/clock_test.go`（修改，1 处）：

````old server/internal/modules/project/app/clock_test.go
		}, updated(domain.ProjectPatch{SetLead: true, LeadID: &alice}, alice)},
````
````new server/internal/modules/project/app/clock_test.go
		}, updated(domain.ProjectPatch{SetLead: true, LeadID: &alice}, alice)},
		{"archiveProject", func() ([]string, error) {
			uc, f := newArchive(true)
			_, err := uc.Execute(as(bob), webID)
			return f.log.calls, err
		}, archived(webID, true)},
		{"unarchiveProject", func() ([]string, error) {
			uc, f := newArchive(false)
			_, err := uc.Execute(as(bob), opsID)
			return f.log.calls, err
		}, archived(opsID, false)},
````

- [ ] **Step 5: HTTP 和接线**

`server/internal/modules/project/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/project/adapter/http/handler.go
}

// UseCases are the use cases behind the module's operations.
````
````new server/internal/modules/project/adapter/http/handler.go
}

// ArchiveProjectUseCase is app.ArchiveProject, which archives or
// unarchives.
type ArchiveProjectUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) (domain.Project, error)
}

// UseCases are the use cases behind the module's operations.
````

````old server/internal/modules/project/adapter/http/handler.go
	ListProjects    ListProjectsUseCase
	CreateProject   CreateProjectUseCase
	GetProject      GetProjectUseCase
	CheckIdentifier CheckIdentifierUseCase
	UpdateProject   UpdateProjectUseCase
````
````new server/internal/modules/project/adapter/http/handler.go
	ListProjects     ListProjectsUseCase
	CreateProject    CreateProjectUseCase
	GetProject       GetProjectUseCase
	CheckIdentifier  CheckIdentifierUseCase
	UpdateProject    UpdateProjectUseCase
	ArchiveProject   ArchiveProjectUseCase
	UnarchiveProject ArchiveProjectUseCase
````

`server/internal/modules/project/adapter/http/projects.go`（修改，1 处）：

````old server/internal/modules/project/adapter/http/projects.go
	return gen.UpdateProject200JSONResponse(project(p)), nil
````
````new server/internal/modules/project/adapter/http/projects.go
	return gen.UpdateProject200JSONResponse(project(p)), nil
}

// ArchiveProject serves POST /api/v0/projects/{project_id}/archive.
func (h handler) ArchiveProject(ctx context.Context, req gen.ArchiveProjectRequestObject) (gen.ArchiveProjectResponseObject, error) {
	p, err := h.uc.ArchiveProject.Execute(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}
	return gen.ArchiveProject200JSONResponse(project(p)), nil
}

// UnarchiveProject serves POST /api/v0/projects/{project_id}/unarchive.
func (h handler) UnarchiveProject(ctx context.Context, req gen.UnarchiveProjectRequestObject) (gen.UnarchiveProjectResponseObject, error) {
	p, err := h.uc.UnarchiveProject.Execute(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}
	return gen.UnarchiveProject200JSONResponse(project(p)), nil
````

`server/internal/modules/project/adapter/http/handler_test.go`（修改，4 处）：

````old server/internal/modules/project/adapter/http/handler_test.go
	list   *fakeList
	create *fakeCreate
	get    *fakeGet
	check  *fakeCheck
	update *fakeUpdate
````
````new server/internal/modules/project/adapter/http/handler_test.go
	list      *fakeList
	create    *fakeCreate
	get       *fakeGet
	check     *fakeCheck
	update    *fakeUpdate
	archive   *fakeOnProject
	unarchive *fakeOnProject
````

````old server/internal/modules/project/adapter/http/handler_test.go
}

// newServer serves the module with f; a fake left nil is an idle one.
````
````new server/internal/modules/project/adapter/http/handler_test.go
}

// fakeOnProject is a use case on a project that takes nothing more.
type fakeOnProject struct {
	calls  []string // "caller id"
	answer domain.Project
	err    error
}

func (f *fakeOnProject) Execute(ctx context.Context, id uuid.UUID) (domain.Project, error) {
	f.calls = append(f.calls, caller(ctx)+" "+id.String())
	return f.answer, f.err
}

// newServer serves the module with f; a fake left nil is an idle one.
````

````old server/internal/modules/project/adapter/http/handler_test.go
		f.update = &fakeUpdate{}
	}
````
````new server/internal/modules/project/adapter/http/handler_test.go
		f.update = &fakeUpdate{}
	}
	if f.archive == nil {
		f.archive = &fakeOnProject{}
	}
	if f.unarchive == nil {
		f.unarchive = &fakeOnProject{}
	}
````

````old server/internal/modules/project/adapter/http/handler_test.go
		CheckIdentifier: f.check, UpdateProject: f.update})
````
````new server/internal/modules/project/adapter/http/handler_test.go
		CheckIdentifier: f.check, UpdateProject: f.update, ArchiveProject: f.archive, UnarchiveProject: f.unarchive})
````

`server/internal/modules/project/adapter/http/archive_test.go`（新文件，47 行）：

````file server/internal/modules/project/adapter/http/archive_test.go
package httpadapter_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// POST .../archive goes to the archiving use case and .../unarchive to the
// other, each for the caller and the path's project, and nothing else; the
// answer is 200 with the project the use case answers. Each has its
// refusals, as the contract declares them.
func TestArchiveAndUnarchiveProject(t *testing.T) {
	for _, op := range []string{"archive", "unarchive"} {
		archive, unarchive := &fakeOnProject{answer: web}, &fakeOnProject{answer: web}
		h := newServer(t, fakes{archive: archive, unarchive: unarchive})
		path := "/api/v0/projects/" + webID.String() + "/" + op
		if res, body := do(t, h, request(http.MethodPost, path, "bob", "")); res.StatusCode != http.StatusOK || body != webJSON+"\n" {
			t.Errorf("POST %s = %d %s, want 200 %s", path, res.StatusCode, body, webJSON)
		}
		called, idle := archive, unarchive
		if op == "unarchive" {
			called, idle = unarchive, archive
		}
		if want := []string{"bob " + webID.String()}; !slices.Equal(called.calls, want) || len(idle.calls) != 0 {
			t.Errorf("%s: calls %q and %q; want %q, and none to the other", op, called.calls, idle.calls, want)
		}
		for _, tt := range []struct {
			err    error
			status int
			want   string
		}{
			{domain.ErrNotFound, http.StatusNotFound,
				`{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`},
			{shared.Forbidden(), http.StatusForbidden, `{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		} {
			refusing := &fakeOnProject{err: tt.err}
			h := newServer(t, fakes{archive: refusing, unarchive: refusing})
			if res, body := do(t, h, request(http.MethodPost, path, "alice", "")); res.StatusCode != tt.status || body != tt.want+"\n" {
				t.Errorf("%s: POST = %d %s, want %d %s", op, res.StatusCode, body, tt.status, tt.want)
			}
		}
	}
}
````

`server/internal/modules/project/module.go`（修改，2 处）：

````old server/internal/modules/project/module.go
// brings listing, creating, reading and changing projects and checking an
// identifier, carries out the workspace module's cascades on the projects
````
````new server/internal/modules/project/module.go
// brings listing, creating, reading, changing and archiving projects and
// checking an identifier, carries out the workspace module's cascades on the projects
````

````old server/internal/modules/project/module.go
		ListProjects:    app.NewListProjects(d.Workspaces, store, d.Authorizer),
		GetProject:      app.NewGetProject(store, d.Authorizer),
		CheckIdentifier: app.NewCheckProjectIdentifier(d.Workspaces, store, d.Authorizer),
		UpdateProject:   app.NewUpdateProject(store, locks, d.Tx, d.Clock),
````
````new server/internal/modules/project/module.go
		ListProjects:     app.NewListProjects(d.Workspaces, store, d.Authorizer),
		GetProject:       app.NewGetProject(store, d.Authorizer),
		CheckIdentifier:  app.NewCheckProjectIdentifier(d.Workspaces, store, d.Authorizer),
		UpdateProject:    app.NewUpdateProject(store, locks, d.Tx, d.Clock),
		ArchiveProject:   app.NewArchiveProject(store, locks, d.Tx, d.Clock),
		UnarchiveProject: app.NewUnarchiveProject(store, locks, d.Tx, d.Clock),
````

- [ ] **Step 6: 矩阵和组合出的测试**

`server/internal/bootstrap/permission_matrix_seed_test.go`（修改，4 处）：

````old server/internal/bootstrap/permission_matrix_seed_test.go
}

// partingStates puts memberships of matrixProjectMembers in the states in
````
````new server/internal/bootstrap/permission_matrix_seed_test.go
}

// archive archives the project key, by its workspace's admin.
func (s projectSeed) archive(key string) {
	s.t.Helper()
	slug, _, _ := strings.Cut(key, "/")
	if err := s.store.SetArchived(context.Background(), s.projects[key], true, s.ids[matrixAdmins[slug]], s.now); err != nil {
		s.t.Fatal(err)
	}
}

// partingStates puts memberships of matrixProjectMembers in the states in
````

````old server/internal/bootstrap/permission_matrix_seed_test.go
// phase that adds the store replaces it: acme's archived project archived
// (P4b), the member before's membership of the private project ended (P5)
// and the removed member's membership of acme ended (P5). exec fails a
// statement that changes no row, and names it, which it checks first.
````
````new server/internal/bootstrap/permission_matrix_seed_test.go
// phase that adds the store replaces it: the member before's membership of
// the private project ended (P5) and the removed member's membership of
// acme ended (P5). exec fails a statement that changes no row, and names
// it, which it checks first.
````

````old server/internal/bootstrap/permission_matrix_seed_test.go
	const none = "UPDATE projects SET archived_at = now() WHERE false"
````
````new server/internal/bootstrap/permission_matrix_seed_test.go
	const none = "UPDATE project_members SET is_active = false WHERE false"
````

````old server/internal/bootstrap/permission_matrix_seed_test.go
	}
	s.exec(s.t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", s.projects["acme/archived"], s.now)
````
````new server/internal/bootstrap/permission_matrix_seed_test.go
	}
````

`server/internal/bootstrap/permission_matrix_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_test.go
// and matrixProjectMembers. Through SQL, the states no store writes yet
````
````new server/internal/bootstrap/permission_matrix_test.go
// and matrixProjectMembers, and acme's archived project archived. Through SQL, the states no store writes yet
````

````old server/internal/bootstrap/permission_matrix_test.go
			projects.join(pm.key, pm.c, pm.role)
		}
````
````new server/internal/bootstrap/permission_matrix_test.go
			projects.join(pm.key, pm.c, pm.role)
		}
		projects.archive("acme/archived")
````

`server/internal/bootstrap/permission_matrix_project_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_project_test.go
			cells: map[caller]cell{callerArchivedAdmin: cellProjectArchived}},
````
````new server/internal/bootstrap/permission_matrix_project_test.go
			cells: map[caller]cell{callerArchivedAdmin: cellProjectArchived}},
		// As updateProject (M3 design 3.4); an archived project archives
		// again, and an unarchived one unarchives.
		{op: "archiveProject", write: true, columns: projectColumns, request: toProject(http.MethodPost, "/archive", ""),
			cells: ofProject(cellOK, cellForbidden, cellForbidden, cellOK, cellForbidden, cellForbidden), check: archivesItsProject(true)},
		{op: "archiveProject", variant: "archived", write: true, columns: archivedColumns, request: toProject(http.MethodPost, "/archive", ""),
			cells: map[caller]cell{callerArchivedAdmin: cellOK}, check: archivesItsProject(true)},
		{op: "unarchiveProject", write: true, columns: projectColumns, request: toProject(http.MethodPost, "/unarchive", ""),
			cells: ofProject(cellOK, cellForbidden, cellForbidden, cellOK, cellForbidden, cellForbidden), check: archivesItsProject(false)},
		{op: "unarchiveProject", variant: "archived", write: true, columns: archivedColumns, request: toProject(http.MethodPost, "/unarchive", ""),
			cells: map[caller]cell{callerArchivedAdmin: cellOK}, check: archivesItsProject(false)},
	}
}

// archivesItsProject: the column's project, with the caller's role in it,
// archived when archived is true, as of its last change, and not archived
// otherwise.
func archivesItsProject(archived bool) func(t *testing.T, c caller, s seeded, answer string) {
	return func(t *testing.T, c caller, s seeded, answer string) {
		var p struct {
			ID         uuid.UUID  `json:"id"`
			MemberRole *int       `json:"member_role"`
			ArchivedAt *time.Time `json:"archived_at"`
			UpdatedAt  time.Time  `json:"updated_at"`
		}
		decodeAnswer(t, answer, &p)
		if p.ID != s.project(projectOf(c)) || p.MemberRole == nil || *p.MemberRole != memberRoles[c] || (p.ArchivedAt != nil) != archived ||
			(archived && !p.ArchivedAt.Equal(p.UpdatedAt)) {
			t.Errorf("%s archives (%v) %s; want %s, his role %d, archived %v as of its change", c, archived, answer, projectOf(c), memberRoles[c],
				archived)
		}
````

`server/internal/bootstrap/project_writes_test.go`（修改，5 处）：

````old server/internal/bootstrap/project_writes_test.go
	status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces/acme/projects", alice, `{"name":"Web","identifier":"WEB"}`)
	if status != http.StatusCreated {
		t.Fatalf("creating Web = %d %s", status, body)
	}
	var web struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, body, &web)
````
````new server/internal/bootstrap/project_writes_test.go
	web := createdProject(t, contract, base, alice, "acme", "Web", "WEB")
````

````old server/internal/bootstrap/project_writes_test.go
		stamps                   string // the rows written: their updated_at, and whether alice wrote them
````
````new server/internal/bootstrap/project_writes_test.go
		stamps                   string // the rows written: the time each took, and whether alice wrote it as the write does
````

````old server/internal/bootstrap/project_writes_test.go
		{"updateProject", http.MethodPatch, "/api/v0/projects/" + web.ID.String(), `{"name":"Site"}`, http.StatusOK,
			"SELECT updated_at, updated_by_id = $2 FROM projects WHERE id = $1"},
````
````new server/internal/bootstrap/project_writes_test.go
		{"updateProject", http.MethodPatch, "/api/v0/projects/" + web.String(), `{"name":"Site"}`, http.StatusOK,
			"SELECT updated_at, updated_by_id = $2 FROM projects WHERE id = $1"},
		{"archiveProject", http.MethodPost, "/api/v0/projects/" + web.String() + "/archive", "", http.StatusOK,
			"SELECT archived_at, updated_by_id = $2 AND updated_at = archived_at FROM projects WHERE id = $1"},
		// Archived again, it takes the new time.
		{"archiveProject again", http.MethodPost, "/api/v0/projects/" + web.String() + "/archive", "", http.StatusOK,
			"SELECT archived_at, updated_by_id = $2 AND updated_at = archived_at FROM projects WHERE id = $1"},
		{"unarchiveProject", http.MethodPost, "/api/v0/projects/" + web.String() + "/unarchive", "", http.StatusOK,
			"SELECT updated_at, updated_by_id = $2 AND archived_at IS NULL FROM projects WHERE id = $1"},
````

````old server/internal/bootstrap/project_writes_test.go
		rows, err := pool.Query(context.Background(), w.stamps, web.ID, aliceID)
````
````new server/internal/bootstrap/project_writes_test.go
		rows, err := pool.Query(context.Background(), w.stamps, web, aliceID)
````

````old server/internal/bootstrap/project_writes_test.go
	}
}

````
````new server/internal/bootstrap/project_writes_test.go
	}
}

// createdProject creates the project name with identifier in the workspace
// slug through the API, by the caller of token, and returns its id.
func createdProject(t *testing.T, contract *apitest.Contract, base, token, slug, name, identifier string) uuid.UUID {
	t.Helper()
	status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces/"+slug+"/projects", token,
		`{"name":"`+name+`","identifier":"`+identifier+`"}`)
	if status != http.StatusCreated {
		t.Fatalf("creating %s in %s = %d %s", name, slug, status, body)
	}
	var p struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, body, &p)
	return p.ID
}

````

`server/internal/bootstrap/project_write_locks_test.go`（新文件，157 行）：

````file server/internal/bootstrap/project_write_locks_test.go
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// projectWrite is a write on a project as
// TestEachWriteOnAProjectSharesItsWorkspaceFirst sends it, by its
// operationId: the request on the project, by alice.
type projectWrite struct {
	op, method, path, body string // path: %s the project's id
	want                   int
}

// projectWrites are the writes on a project, in the order they run on
// Web, then on Ops.
var projectWrites = []projectWrite{
	{op: "updateProject", method: http.MethodPatch, path: "/api/v0/projects/%s", body: `{"description":"Changed"}`, want: http.StatusOK},
	{op: "archiveProject", method: http.MethodPost, path: "/api/v0/projects/%s/archive", want: http.StatusOK},
	{op: "unarchiveProject", method: http.MethodPost, path: "/api/v0/projects/%s/unarchive", want: http.StatusOK},
}

// writesOnAProject are the operations whose matrix rows write and have the
// project level's columns: every write on a project of the contract.
func writesOnAProject() []string {
	var ops []string
	for _, r := range matrixRows() {
		if r.write && slices.Equal(r.columns, projectColumns) && !slices.Contains(ops, r.op) {
			ops = append(ops, r.op)
		}
	}
	return slices.Sorted(slices.Values(ops))
}

// holding begins a transaction on pool that runs sql with args, and keeps
// it open until the test rolls it back, or ends.
func holding(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) pgx.Tx {
	t.Helper()
	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	if tag, err := tx.Exec(context.Background(), sql, args...); err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("%s: %v, %v; want one row held", sql, tag, err)
	}
	return tx
}

// heldBy reports whether a transaction holds the one row sql locks NOWAIT:
// it answers lock_not_available (55P03) at once while one does.
func heldBy(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) bool {
	t.Helper()
	tag, err := pool.Exec(context.Background(), sql, args...)
	var pgErr *pgconn.PgError
	switch {
	case errors.As(err, &pgErr) && pgErr.Code == "55P03":
		return true
	case err != nil || tag.RowsAffected() != 1:
		t.Fatalf("%s: %v, %v; want one row", sql, tag, err)
	}
	return false
}

// Every write on a project takes its workspace's row FOR SHARE first, in
// its transaction, before any other lock (M3 design 3.6 convention 2, the
// lock table), as bootstrap wires it: alice, acme's admin, writes on her
// projects Web and Ops, each write once on each. Every write on a project
// of the contract has its row here: the matrix's rows that write at the
// project level are the list.
//   - The workspace first: another transaction holds acme's row FOR NO KEY
//     UPDATE, as every cascade over its projects does (3.3). The write on
//     Web waits for that row, and meanwhile does not hold Web's row: a FOR
//     UPDATE NOWAIT of it succeeds.
//   - In its transaction, FOR SHARE, before its project: another
//     transaction holds Ops's row FOR NO KEY UPDATE. The write on Ops waits
//     for it, and meanwhile holds acme's row at FOR SHARE, no stronger,
//     which another write on a project of acme shares (a FOR NO KEY UPDATE
//     NOWAIT of it fails, a FOR SHARE NOWAIT succeeds).
//
// Once the other transaction ends, each write answers as it would alone.
func TestEachWriteOnAProjectSharesItsWorkspaceFirst(t *testing.T) {
	ops := make([]string, len(projectWrites))
	for i, w := range projectWrites {
		ops[i] = w.op
	}
	if want := writesOnAProject(); !slices.Equal(slices.Sorted(slices.Values(ops)), want) {
		t.Fatalf("the writes here are %q; the writes on a project are %q: each has its row, in projectWrites", slices.Sorted(slices.Values(ops)),
			want)
	}
	contract := apitest.Load(t)
	dbURL := pgtest.NewDatabase(t)
	base := startApp(t, testConfig(t, dbURL, false), migrations.FS())
	pool := openPool(t, dbURL)
	alice := registerAccount(t, contract, base, "alice@example.com").AccessToken
	status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", alice, `{"name":"Acme","slug":"acme"}`)
	if status != http.StatusCreated {
		t.Fatalf("creating acme = %d %s", status, body)
	}
	var acme struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, body, &acme)
	projects := [2]uuid.UUID{createdProject(t, contract, base, alice, "acme", "Web", "WEB"),
		createdProject(t, contract, base, alice, "acme", "Ops", "OPS")}
	for _, w := range projectWrites {
		for phase, project := range projects {
			req := newRequest(t, w.method, base+fmt.Sprintf(w.path, project), alice, []byte(w.body))
			contract.CheckRequest(t, req)
			var other pgx.Tx
			if phase == 0 {
				other = holding(t, pool, "SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", acme.ID)
			} else {
				other = holding(t, pool, "SELECT 1 FROM projects WHERE id = $1 FOR NO KEY UPDATE", project)
			}
			answered := sendInBackground(req)
			if phase == 0 {
				pgtest.WaitForLockWaitOn(t, pool, "workspaces", 10*time.Second)
				if heldBy(t, pool, "SELECT 1 FROM projects WHERE id = $1 FOR UPDATE NOWAIT", project) {
					t.Errorf("%s holds its project while it waits for its workspace", w.op)
				}
			} else {
				pgtest.WaitForLockWaitOn(t, pool, "projects", 10*time.Second)
				if !heldBy(t, pool, "SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE NOWAIT", acme.ID) ||
					heldBy(t, pool, "SELECT 1 FROM workspaces WHERE id = $1 FOR SHARE NOWAIT", acme.ID) {
					t.Errorf("%s does not hold its workspace FOR SHARE in its transaction while it waits for its project", w.op)
				}
			}
			if err := other.Rollback(context.Background()); err != nil {
				t.Fatal(err)
			}
			a := receiveWithin(t, answered, 10*time.Second, "the answer to "+w.op)
			if a.err != nil {
				t.Fatal(a.err)
			}
			contract.CheckResponse(t, req, a.res)
			if a.res.StatusCode != w.want {
				t.Errorf("%s on %s = %d %s, want %d", w.op, []string{"Web", "Ops"}[phase], a.res.StatusCode, a.body, w.want)
			}
		}
	}
}
````

- [ ] **Step 7: 测试、lint、前端检查**

Run: `go -C server test -count=1 ./internal/modules/project/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix$|TestThePermissionMatrixCoversEveryOperation|TestMatrixViolationsCatchesEach|TestEveryColumnCallsAsARegisteredAccount|TestEveryActionHasARuleAndEveryRuleAnAction|TestAPIRoutesAreTheContractsOperations|TestOperationsThatNeedATokenAnswer401WithoutOne|TestTheWritesOnAProjectStampTheirRequest|TestEachWriteOnAProjectSharesItsWorkspaceFirst' ./internal/bootstrap/`
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

- [ ] **Step 8: 提交**

```bash
git add api/modules/project.yaml api/openapi.yaml server/internal/bootstrap/permission_matrix_project_test.go server/internal/bootstrap/permission_matrix_seed_test.go server/internal/bootstrap/permission_matrix_test.go server/internal/bootstrap/project_write_locks_test.go server/internal/bootstrap/project_writes_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/project/adapter/http/archive_test.go server/internal/modules/project/adapter/http/handler.go server/internal/modules/project/adapter/http/handler_test.go server/internal/modules/project/adapter/http/projects.go server/internal/modules/project/adapter/postgres/failures_test.go server/internal/modules/project/adapter/postgres/queries/projects.sql server/internal/modules/project/adapter/postgres/update.go server/internal/modules/project/adapter/postgres/update_test.go server/internal/modules/project/app/archive_project.go server/internal/modules/project/app/archive_project_test.go server/internal/modules/project/app/clock_test.go server/internal/modules/project/app/fakes_write_test.go server/internal/modules/project/app/ports.go server/internal/modules/project/domain/actions.go server/internal/modules/project/module.go api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/server.gen.go server/internal/modules/project/adapter/postgres/gen/projects.sql.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P4b): archiveProject and unarchiveProject

POST .../archive and .../unarchive, for whoever may change the project.
Each takes Locks' path (the workspace FOR SHARE, then the project FOR NO
KEY UPDATE), decides, reads the clock under the locks and sets
archived_at to it, or clears it; archiving an archived project stamps it
again, as Plane's does. The matrix archives its archived project through
the store, no longer through SQL. A composed test holds the workspace,
then the project, from another transaction, and finds each write on a
project waiting on the workspace's row first, holding nothing, then
holding the workspace FOR SHARE in its transaction; its writes are the
matrix's writes at the project level, one row each.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| `SetArchived` 改每个项目（去掉 id） | `TestSetArchived`；P4（Task 15 起） | 存储；端到端 |
| 恢复时保留 `archived_at`；不写 `updated_at`；也写 `created_at` | `TestSetArchived` | 存储 |
| 两条规则各给项目成员、给不是成员的工作区管理员 | `TestPermissionMatrix` | 组合 |
| 没有调用者时也读 | `TestArchiveProjectRefuses` | 单元 |
| 锁和判定、写、读回、提交的失败被吞掉；写重试一次 | `TestArchiveProjectReturnsEachFailure` | 单元 |
| handler 吞掉两个用例的失败 | `TestArchiveAndUnarchiveProject` | 单元 |
| 归档先判定后锁 | `TestArchiveProject`；`TestAProjectWriteAndADemotionSerialize`（Task 14 起） | 单元；组合 |
| 只有归档、只有恢复不锁工作区（各一个） | `TestEachWriteOnAProjectSharesItsWorkspaceFirst`；`TestAProjectWriteAndADemotionSerialize`（Task 14 起） | 组合 |
| 测试改坏：第一段的探测等在 `projects`、第二段的等在 `workspaces` | `TestEachWriteOnAProjectSharesItsWorkspaceFirst` 自己失败（等待在指名的表上，探测超时）：探测不会被别的等待满足 | 组合 |
| `SetArchived` 在事务之外执行 | `TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 14 起） | 组合 |
| `project.New` 的接线（归档和恢复接反等） | 见 Task 8 的表 | 组合 |

**Done when:** 矩阵四行（26 格）通过；归档、再归档、恢复由调用者在请求之内写入；矩阵不再用 SQL 归档；修改、归档、恢复在组合出的 app 上先锁工作区（`TestEachWriteOnAProjectSharesItsWorkspaceFirst`）。

---

### Task 5: 删除项目的一处步骤；目录读取按父表

**Files:**
- Create: `server/internal/modules/project/app/deletion.go`
- Modify: `server/internal/bootstrap/workspace_deletion_test.go`、`server/internal/modules/project/adapter/postgres/failures_test.go`、`server/internal/modules/project/app/cascade_test.go`、`server/internal/modules/project/app/fakes_write_test.go`、`server/internal/modules/project/app/ports.go`
- Modify（完整内容）: `server/internal/bootstrap/workspace_deletion_catalog_test.go`、`server/internal/modules/project/adapter/postgres/cascade.go`、`server/internal/modules/project/adapter/postgres/cascade_test.go`、`server/internal/modules/project/adapter/postgres/queries/cascade.sql`、`server/internal/modules/project/app/cascade.go`
- Generate: `server/internal/modules/project/adapter/postgres/gen/cascade.sql.go`

**Interfaces:**
- Produces（spec 2.6，M3 设计 3.3、3.6，brief 移交第 7 条）：`app.Deletion{WorkspaceID, ProjectID *uuid.UUID, By, Now}`；`app.ProjectsDeleter`（`DeleteProjects`、`DeleteProjectMembers`、`DeleteProjectPreferences`、`DeleteStates`，各一条语句）取代 P4a 的 `WorkspaceProjectsDeleter` 的四个方法；`app/deletion.go` 的 `deleteProjects(ctx, p, d)` 是删除项目的步骤唯一的一处：全局顺序项目 → 项目成员 → 显示设置 → 状态，P7 在最后加标签；`Cascade.DeleteWorkspaceProjects` 和 Task 6 的 `deleteProject` 都调它，谁都不能漏掉对方删除的表。
- 四条语句加 `(sqlc.narg(project_id)::uuid IS NULL OR <列> = sqlc.narg(project_id))`：没给项目时是工作区的全部（P4a 的行为不变），给了时只有它。
- `bootstrap/workspace_deletion_catalog_test.go`：`workspaceKeys` 改为 `keysTo(t, pool, parent)`：父表本身（`<父表>.id`）和目录中指向它的每个外键；只经别的表挂在父表下、自己没有指向父表的外键的表让测试失败（P4a 的 F-M1，现在对 `workspaces`、`projects` 两个父表都成立）。`deletionViolations(parent, …)` 的说法按父表。

**Tests:**
- `adapter/postgres/cascade_test.go`：`TestDeletingAWorkspaceSoftDeletesItsProjects`（原样，经新的步骤）；`TestDeletingAProjectSoftDeletesItsRowsAlone`（新，Web、Ops（已归档、bob 的成员关系已结束）各一次：它和它下面的行在同一时刻、由同一个账户删除；同工作区的另一个项目、此前删除的项目、别的工作区的项目一行不动；`unwritten` 要求删除除了三列什么都不写；再跑一次什么都不变）。
- `app/cascade_test.go`：`TestDeleteWorkspaceProjects` 的调用记录改为 `deletionSteps(workspace, "*", …)`（假存储由 `fakes_write_test.go` 的 `deleting` 记下工作区、项目或 `*`）。
- `bootstrap`：`TestKeysRefuseATableWithoutItsParent`（新：建 `widgets`（指向 `projects`）和 `gadgets`（指向 `widgets`），`keysTo("workspaces")` 让测试失败并说明两者各经哪张表挂在下面，`keysTo("projects")` 失败并说明 `gadgets` 经 `widgets`）；工作区删除的三个组合测试经 `keysTo("workspaces")` 照旧通过。

- [ ] **Step 1: 查询**

`server/internal/modules/project/adapter/postgres/queries/cascade.sql`（完整内容，50 行）：

````whole server/internal/modules/project/adapter/postgres/queries/cascade.sql
-- name: DeleteProjects :exec
-- The first step of deleting projects (M3 design 3.3, 3.6), one statement (convention 5) under the parent's FOR NO KEY
-- UPDATE, which the caller took: the workspace's, when it deletes the workspace, and the project's, when it deletes
-- the project. The workspace's undeleted projects, archived ones too, or only the one project_id names when it is
-- given, at the moment and by the account of the deletion. Projects deleted before keep their moment.
UPDATE projects
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND (sqlc.narg(project_id)::uuid IS NULL OR id = sqlc.narg(project_id))
  AND deleted_at IS NULL;

-- name: DeleteProjectMembers :exec
-- The memberships of those projects, active and ended ones alike.
UPDATE project_members
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND (sqlc.narg(project_id)::uuid IS NULL OR project_id = sqlc.narg(project_id))
  AND deleted_at IS NULL;

-- name: DeleteProjectPreferences :exec
UPDATE project_user_properties
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND (sqlc.narg(project_id)::uuid IS NULL OR project_id = sqlc.narg(project_id))
  AND deleted_at IS NULL;

-- name: DeleteStates :exec
-- The triage states too.
UPDATE states
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND (sqlc.narg(project_id)::uuid IS NULL OR project_id = sqlc.narg(project_id))
  AND deleted_at IS NULL;

-- name: LockMemberProjects :many
-- The first step of making an account a guest in the workspace's projects, DemoteToGuest's: the workspace's undeleted
-- projects, archived ones too, in which he has an undeleted membership, active or not, FOR NO KEY UPDATE in id order.
-- The lock is taken as the sorted rows come, so the order is the ids'. After a wait, Postgres evaluates deleted_at IS
-- NULL again on the row's newest version: a project deleted meanwhile is left out.
SELECT p.id
FROM projects p
WHERE p.workspace_id = sqlc.arg(workspace_id) AND p.deleted_at IS NULL
  AND EXISTS (SELECT 1 FROM project_members m
              WHERE m.project_id = p.id AND m.member_id = sqlc.arg(member_id) AND m.deleted_at IS NULL)
ORDER BY p.id
FOR NO KEY UPDATE;

-- name: DemoteMemberships :exec
-- The second step, one statement under the projects' locks (convention 5): the account's undeleted memberships of the
-- projects, active or not, a guest's now, at the moment and by the account given; a guest's keeps its audit columns.
UPDATE project_members
SET role = 5, updated_at = sqlc.arg(now)::timestamptz, updated_by_id = sqlc.arg(updated_by)::uuid
WHERE project_id = ANY (sqlc.arg(project_ids)::uuid[]) AND member_id = sqlc.arg(member_id) AND deleted_at IS NULL
  AND role <> 5;
````

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `563eccbc2b658b96b1ab5d62f2d0d28d4ac085846ad786285f0d15ffaa267982` | 180 | `server/internal/modules/project/adapter/postgres/gen/cascade.sql.go` |

Run: `shasum -a 256 server/internal/modules/project/adapter/postgres/gen/cascade.sql.go`
Expected: 与上表相同。

- [ ] **Step 2: 端口、步骤、存储**

`server/internal/modules/project/app/ports.go`（修改，1 处）：

````old server/internal/modules/project/app/ports.go
// WorkspaceProjectsDeleter soft-deletes a workspace's projects and the rows
// under them, one statement a table, in the order of M3 design 3.6. Each
// sets deleted_at and updated_at to now and updated_by_id to by, on the
// workspace's undeleted rows only; it runs in the transaction ctx carries.
type WorkspaceProjectsDeleter interface {
	DeleteWorkspaceProjects(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error
	DeleteWorkspaceProjectMembers(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error
	DeleteWorkspaceProjectPreferences(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error
	DeleteWorkspaceStates(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error
}

````
````new server/internal/modules/project/app/ports.go
// Deletion is what a deletion of projects deletes, and when and by whom:
// the workspace's projects, or only the one ProjectID names, a project of
// the workspace, when it is set.
type Deletion struct {
	WorkspaceID uuid.UUID
	ProjectID   *uuid.UUID
	By          uuid.UUID
	Now         time.Time
}

// ProjectsDeleter soft-deletes the projects of a Deletion and the rows
// under them, one statement a table. Each sets deleted_at and updated_at to
// the deletion's moment and updated_by_id to its account, on undeleted rows
// only; it runs in the transaction ctx carries.
type ProjectsDeleter interface {
	DeleteProjects(ctx context.Context, d Deletion) error
	DeleteProjectMembers(ctx context.Context, d Deletion) error
	DeleteProjectPreferences(ctx context.Context, d Deletion) error
	DeleteStates(ctx context.Context, d Deletion) error
}

````

`server/internal/modules/project/app/deletion.go`（新文件，20 行）：

````file server/internal/modules/project/app/deletion.go
package app

import "context"

// deleteProjects runs the steps of a deletion of projects, in the global
// order of M3 design 3.6: the projects, their memberships, the members'
// display settings, the states; P7 adds the labels at the end. Deleting a
// workspace (Cascade) and deleting a project both run it, so neither can
// leave out a table the other deletes. A failing step comes back as itself
// and the steps after it do not run.
func deleteProjects(ctx context.Context, p ProjectsDeleter, d Deletion) error {
	for _, step := range []func(context.Context, Deletion) error{
		p.DeleteProjects, p.DeleteProjectMembers, p.DeleteProjectPreferences, p.DeleteStates,
	} {
		if err := step(ctx, d); err != nil {
			return err
		}
	}
	return nil
}
````

`server/internal/modules/project/app/cascade.go`（完整内容，47 行）：

````whole server/internal/modules/project/app/cascade.go
package app

import (
	"context"
	"time"
	"uuid"
)

// Cascade is what the workspace module's writes ask of the projects,
// workspace's ProjectCascade port (M3 design 3.3). Each method uses
// project's own repositories only, in the transaction ctx carries, which
// the caller began and holds its workspace's lock in: a failure comes back
// as itself and the caller's whole transaction rolls back. The caller gives
// who acts and the moment, which each method writes into the rows it
// changes.
type Cascade struct {
	projects ProjectsDeleter
	members  MemberDemoter
}

// NewCascade returns the cascade over projects and members.
func NewCascade(projects ProjectsDeleter, members MemberDemoter) *Cascade {
	return &Cascade{projects: projects, members: members}
}

// DeleteWorkspaceProjects soft-deletes the workspace's projects and the
// rows under them (M3 design 3.6): the last step of deleteWorkspace's
// cascade, under its FOR NO KEY UPDATE of the workspace, each step one
// statement (convention 5), the steps deleteProject runs too
// (deleteProjects), at by and now.
func (c *Cascade) DeleteWorkspaceProjects(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return deleteProjects(ctx, c.projects, Deletion{WorkspaceID: workspaceID, By: by, Now: now})
}

// DemoteToGuest makes userID a guest in each of the workspace's projects he
// has a membership of, ended ones too (M3 design 3.3; Plane
// views/workspace/member.py:87-89): under the caller's FOR NO KEY UPDATE of
// the workspace, those projects FOR NO KEY UPDATE in id order, then his
// memberships of them in one statement (convention 5), at now, by by. With
// no such project there is nothing to write.
func (c *Cascade) DemoteToGuest(ctx context.Context, workspaceID, userID, by uuid.UUID, now time.Time) error {
	projects, err := c.members.LockMemberProjects(ctx, workspaceID, userID)
	if err != nil || len(projects) == 0 {
		return err
	}
	return c.members.DemoteMemberships(ctx, projects, userID, by, now)
}
````

`server/internal/modules/project/app/fakes_write_test.go`（修改，1 处）：

````old server/internal/modules/project/app/fakes_write_test.go
}

func (f *fakeStore) GetProject(ctx context.Context, id, userID uuid.UUID) (domain.Project, bool, error) {
````
````new server/internal/modules/project/app/fakes_write_test.go
}

// deleting logs a step of a deletion: the workspace, the project, or every
// project of the workspace ("*"), the account and the moment.
func (f *fakeStore) deleting(ctx context.Context, step string, d app.Deletion) error {
	project := "*"
	if d.ProjectID != nil {
		project = d.ProjectID.String()
	}
	f.log.add(ctx, "%s %s/%s by %s at %s", step, d.WorkspaceID, project, d.By, d.Now.Format(timeFormat))
	return f.fail(step)
}

func (f *fakeStore) DeleteProjects(ctx context.Context, d app.Deletion) error {
	return f.deleting(ctx, "DeleteProjects", d)
}

func (f *fakeStore) DeleteProjectMembers(ctx context.Context, d app.Deletion) error {
	return f.deleting(ctx, "DeleteProjectMembers", d)
}

func (f *fakeStore) DeleteProjectPreferences(ctx context.Context, d app.Deletion) error {
	return f.deleting(ctx, "DeleteProjectPreferences", d)
}

func (f *fakeStore) DeleteStates(ctx context.Context, d app.Deletion) error {
	return f.deleting(ctx, "DeleteStates", d)
}

func (f *fakeStore) GetProject(ctx context.Context, id, userID uuid.UUID) (domain.Project, bool, error) {
````

`server/internal/modules/project/app/cascade_test.go`（修改，9 处）：

````old server/internal/modules/project/app/cascade_test.go

// fakeDeleter is the projects' repository: it records each step with its
// arguments, and " outside tx" when it ran outside the caller's
// transaction (callLog), and fails the step named in fail.
type fakeDeleter struct {
	log  callLog
	fail string
}

````
````new server/internal/modules/project/app/cascade_test.go

````

````old server/internal/modules/project/app/cascade_test.go
func (f *fakeDeleter) step(ctx context.Context, name string, workspaceID, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "%s %s by %s at %s", name, workspaceID, by, now.Format(time.RFC3339Nano))
	if name == f.fail {
		return errDisk
````
````new server/internal/modules/project/app/cascade_test.go
// deletionSteps are the steps of a deletion of project of workspace, "*"
// for every project of it, by by at at, in the order of M3 design 3.6.
func deletionSteps(workspace uuid.UUID, project string, by uuid.UUID, at time.Time) []string {
	var steps []string
	for _, name := range []string{"DeleteProjects", "DeleteProjectMembers", "DeleteProjectPreferences", "DeleteStates"} {
		steps = append(steps, fmt.Sprintf("%s %s/%s by %s at %s", name, workspace, project, by, at.Format(timeFormat)))
````

````old server/internal/modules/project/app/cascade_test.go
	return nil
}

func (f *fakeDeleter) DeleteWorkspaceProjects(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return f.step(ctx, "DeleteWorkspaceProjects", workspaceID, by, now)
}

func (f *fakeDeleter) DeleteWorkspaceProjectMembers(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return f.step(ctx, "DeleteWorkspaceProjectMembers", workspaceID, by, now)
}

func (f *fakeDeleter) DeleteWorkspaceProjectPreferences(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return f.step(ctx, "DeleteWorkspaceProjectPreferences", workspaceID, by, now)
}

func (f *fakeDeleter) DeleteWorkspaceStates(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return f.step(ctx, "DeleteWorkspaceStates", workspaceID, by, now)
````
````new server/internal/modules/project/app/cascade_test.go
	return steps
````

````old server/internal/modules/project/app/cascade_test.go
// 3.6, each with the caller's workspace, account and moment, in the
// caller's transaction; a failing step comes back as itself and the steps
// after it do not run.
````
````new server/internal/modules/project/app/cascade_test.go
// 3.6, each on every project of the caller's workspace, with the caller's
// account and moment, in the caller's transaction; a failing step comes
// back as itself and the steps after it do not run.
````

````old server/internal/modules/project/app/cascade_test.go
	now := time.Date(2026, 10, 1, 10, 0, 0, 123456000, time.UTC)
	var steps []string
	for _, name := range []string{"DeleteWorkspaceProjects", "DeleteWorkspaceProjectMembers", "DeleteWorkspaceProjectPreferences",
		"DeleteWorkspaceStates"} {
		steps = append(steps, fmt.Sprintf("%s %s by %s at %s", name, workspace, by, now.Format(time.RFC3339Nano)))
	}
````
````new server/internal/modules/project/app/cascade_test.go
	steps := deletionSteps(workspace, "*", by, clockNow)
````

````old server/internal/modules/project/app/cascade_test.go
		{"DeleteWorkspaceProjects", errDisk, steps[:1]},
		{"DeleteWorkspaceProjectMembers", errDisk, steps[:2]},
		{"DeleteWorkspaceProjectPreferences", errDisk, steps[:3]},
		{"DeleteWorkspaceStates", errDisk, steps},
````
````new server/internal/modules/project/app/cascade_test.go
		{"DeleteProjects", errDisk, steps[:1]},
		{"DeleteProjectMembers", errDisk, steps[:2]},
		{"DeleteProjectPreferences", errDisk, steps[:3]},
		{"DeleteStates", errDisk, steps},
````

````old server/internal/modules/project/app/cascade_test.go
		f := &fakeDeleter{fail: tt.fail}
````
````new server/internal/modules/project/app/cascade_test.go
		f := newWrites()
		f.store.errs = map[string]error{tt.fail: errDisk}
````

````old server/internal/modules/project/app/cascade_test.go
		err := app.NewCascade(f, &fakeDemoter{}).DeleteWorkspaceProjects(inTx, workspace, by, now)
````
````new server/internal/modules/project/app/cascade_test.go
		err := app.NewCascade(f.store, &fakeDemoter{}).DeleteWorkspaceProjects(inTx, workspace, by, clockNow)
````

````old server/internal/modules/project/app/cascade_test.go
		err := app.NewCascade(&fakeDeleter{}, f).DemoteToGuest(inTx, workspace, user, by, now)
````
````new server/internal/modules/project/app/cascade_test.go
		err := app.NewCascade(newWrites().store, f).DemoteToGuest(inTx, workspace, user, by, now)
````

`server/internal/modules/project/adapter/postgres/cascade.go`（完整内容，81 行）：

````whole server/internal/modules/project/adapter/postgres/cascade.go
package postgresadapter

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
)

// The steps of deleting projects (app.ProjectsDeleter): each soft-deletes
// the undeleted rows of one table under the workspace's projects, or under
// the one project the deletion names, at its moment, by its account, in one
// statement.

// DeleteProjects soft-deletes the projects, archived ones too.
func (s *Store) DeleteProjects(ctx context.Context, d app.Deletion) error {
	err := s.queries(ctx).DeleteProjects(ctx, gen.DeleteProjectsParams{WorkspaceID: d.WorkspaceID, ProjectID: d.ProjectID, DeletedBy: d.By, Now: d.Now})
	if err != nil {
		return fmt.Errorf("delete the projects: %w", err)
	}
	return nil
}

// DeleteProjectMembers soft-deletes the projects' memberships, active or
// not.
func (s *Store) DeleteProjectMembers(ctx context.Context, d app.Deletion) error {
	err := s.queries(ctx).DeleteProjectMembers(ctx, gen.DeleteProjectMembersParams{
		WorkspaceID: d.WorkspaceID, ProjectID: d.ProjectID, DeletedBy: d.By, Now: d.Now,
	})
	if err != nil {
		return fmt.Errorf("delete the project members: %w", err)
	}
	return nil
}

// DeleteProjectPreferences soft-deletes the display settings in the
// projects.
func (s *Store) DeleteProjectPreferences(ctx context.Context, d app.Deletion) error {
	err := s.queries(ctx).DeleteProjectPreferences(ctx, gen.DeleteProjectPreferencesParams{
		WorkspaceID: d.WorkspaceID, ProjectID: d.ProjectID, DeletedBy: d.By, Now: d.Now,
	})
	if err != nil {
		return fmt.Errorf("delete the project preferences: %w", err)
	}
	return nil
}

// DeleteStates soft-deletes the projects' states, the triage states too.
func (s *Store) DeleteStates(ctx context.Context, d app.Deletion) error {
	err := s.queries(ctx).DeleteStates(ctx, gen.DeleteStatesParams{WorkspaceID: d.WorkspaceID, ProjectID: d.ProjectID, DeletedBy: d.By, Now: d.Now})
	if err != nil {
		return fmt.Errorf("delete the states: %w", err)
	}
	return nil
}

// LockMemberProjects locks FOR NO KEY UPDATE, in id order, the workspace's
// undeleted projects in which userID has an undeleted membership, active or
// not, and returns their ids (app.MemberDemoter).
func (s *Store) LockMemberProjects(ctx context.Context, workspaceID, userID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).LockMemberProjects(ctx, gen.LockMemberProjectsParams{WorkspaceID: workspaceID, MemberID: userID})
	if err != nil {
		return nil, fmt.Errorf("lock the member's projects: %w", err)
	}
	return ids, nil
}

// DemoteMemberships makes userID's undeleted memberships of projectIDs,
// active or not, a guest's, at now, by the account by (app.MemberDemoter).
func (s *Store) DemoteMemberships(ctx context.Context, projectIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error {
	err := s.queries(ctx).DemoteMemberships(ctx, gen.DemoteMembershipsParams{
		ProjectIds: projectIDs, MemberID: userID, UpdatedBy: by, Now: now,
	})
	if err != nil {
		return fmt.Errorf("demote the member's project memberships: %w", err)
	}
	return nil
}
````

`server/internal/modules/project/adapter/postgres/cascade_test.go`（完整内容，256 行）：

````whole server/internal/modules/project/adapter/postgres/cascade_test.go
package postgresadapter_test

import (
	"context"
	"fmt"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// deletion is a row's audit columns, as the tests read them, and the
// project the row is under.
type deletion struct {
	project   uuid.UUID
	deletedAt *time.Time
	updatedAt time.Time
	updatedBy *uuid.UUID
}

// deletedAtBy reports whether d was deleted at when by by.
func (d deletion) deletedAtBy(when time.Time, by uuid.UUID) bool {
	return d.deletedAt != nil && d.deletedAt.Equal(when) && d.updatedAt.Equal(when) && d.updatedBy != nil && *d.updatedBy == by
}

// String is d as a failure prints it: the times in RFC 3339 to the
// microsecond and the account's id, each null when the column is.
func (d deletion) String() string {
	by := "null"
	if d.updatedBy != nil {
		by = d.updatedBy.String()
	}
	return fmt.Sprintf("deleted_at %s, updated_at %s, updated_by %s", instant(d.deletedAt), instant(&d.updatedAt), by)
}

// instant is *t in RFC 3339 to the microsecond, as timestamptz keeps it, or
// null.
func instant(t *time.Time) string {
	if t == nil {
		return "null"
	}
	return t.UTC().Format("2006-01-02T15:04:05.000000Z07:00")
}

// projectTables are the tables a deletion of projects deletes the rows of,
// in its order, each with its column of the project.
var projectTables = []struct{ name, project string }{
	{"projects", "id"}, {"project_members", "project_id"}, {"project_user_properties", "project_id"}, {"states", "project_id"},
}

// perProject is how many rows of each table seedProject writes.
var perProject = map[string]int{"projects": 1, "project_members": 2, "project_user_properties": 2, "states": 2}

// deletions reads the audit columns of the workspace's rows of each project
// table, keyed by table then id.
func deletions(t *testing.T, pool *pgxpool.Pool, workspace uuid.UUID) map[string]map[uuid.UUID]deletion {
	t.Helper()
	out := map[string]map[uuid.UUID]deletion{}
	for _, table := range projectTables {
		rows, err := pool.Query(context.Background(),
			"SELECT id, "+table.project+", deleted_at, updated_at, updated_by_id FROM "+table.name+" WHERE workspace_id = $1", workspace)
		if err != nil {
			t.Fatal(err)
		}
		out[table.name] = map[uuid.UUID]deletion{}
		for rows.Next() {
			var id uuid.UUID
			var d deletion
			if err := rows.Scan(&id, &d.project, &d.deletedAt, &d.updatedAt, &d.updatedBy); err != nil {
				t.Fatal(err)
			}
			out[table.name][id] = d
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
	}
	return out
}

// seedProject writes a project of workspace named name, by alice at now,
// with alice's and bob's memberships and display settings in it, and a
// backlog state and the triage state, and returns its id. The rows are
// written directly: the delete statements read no column the inserts of
// the use cases would set otherwise.
func seedProject(t *testing.T, pool *pgxpool.Pool, workspace uuid.UUID, name string, alice, bob uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	exec(t, pool, `INSERT INTO projects (id, workspace_id, name, identifier, created_by_id, updated_by_id, created_at, updated_at)
		VALUES ($1, $2, $3::text, upper($3::text), $4, $4, $5, $5)`, id, workspace, name, alice, now)
	for _, user := range []uuid.UUID{alice, bob} {
		exec(t, pool, `INSERT INTO project_members (id, workspace_id, project_id, member_id, role, created_by_id, updated_by_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, 20, $5, $5, $6, $6)`, uuid.NewV7(), workspace, id, user, alice, now)
		exec(t, pool, `INSERT INTO project_user_properties (id, workspace_id, project_id, user_id, created_by_id, updated_by_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $5, $6, $6)`, uuid.NewV7(), workspace, id, user, alice, now)
	}
	for _, group := range []string{"backlog", "triage"} {
		exec(t, pool, `INSERT INTO states (id, workspace_id, project_id, name, color, "group", created_by_id, updated_by_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, '#60646C', $4, $5, $5, $6, $6)`, uuid.NewV7(), workspace, id, group, alice, now)
	}
	return id
}

// seedProjects writes three projects of workspace (seedProject) and
// returns their ids: web; ops, archived, bob's membership of it inactive;
// old, deleted at earlier with the rows under it.
func seedProjects(t *testing.T, pool *pgxpool.Pool, workspace, alice, bob uuid.UUID, earlier time.Time) (web, ops, old uuid.UUID) {
	t.Helper()
	web = seedProject(t, pool, workspace, "Web", alice, bob)
	ops = seedProject(t, pool, workspace, "Ops", alice, bob)
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", ops, now)
	exec(t, pool, "UPDATE project_members SET is_active = false WHERE project_id = $1 AND member_id = $2", ops, bob)
	old = seedProject(t, pool, workspace, "Old", alice, bob)
	for _, table := range projectTables {
		exec(t, pool, "UPDATE "+table.name+" SET deleted_at = $2 WHERE "+table.project+" = $1", old, earlier)
	}
	return web, ops, old
}

// deleteAll runs the steps of d, in their order, in the transaction ctx
// carries if any.
func deleteAll(ctx context.Context, s *postgresadapter.Store, d app.Deletion) error {
	for _, step := range []func(context.Context, app.Deletion) error{
		s.DeleteProjects, s.DeleteProjectMembers, s.DeleteProjectPreferences, s.DeleteStates,
	} {
		if err := step(ctx, d); err != nil {
			return err
		}
	}
	return nil
}

// checkDeletions holds the workspace's rows of each project table to want,
// each row's state by its project: "deleted" at later by bob; "before",
// deleted at earlier and not changed since; "kept", neither deleted nor
// changed. Each project of want has perProject's rows of each table, and
// no row is under another.
func checkDeletions(t *testing.T, pool *pgxpool.Pool, workspace uuid.UUID, want map[uuid.UUID]string, later, earlier time.Time, bob uuid.UUID) {
	t.Helper()
	for table, rows := range deletions(t, pool, workspace) {
		under := map[uuid.UUID]int{}
		for id, d := range rows {
			under[d.project]++
			var ok bool
			switch want[d.project] {
			case "deleted":
				ok = d.deletedAtBy(later, bob)
			case "before":
				ok = d.deletedAt != nil && d.deletedAt.Equal(earlier) && d.updatedAt.Equal(now)
			case "kept":
				ok = d.deletedAt == nil && d.updatedAt.Equal(now)
			}
			if !ok {
				t.Errorf("%s %s of project %s: %s; want it %q (deleted: at %s by bob %s; before: deleted_at %s, updated_at %s; kept: "+
					"deleted_at null, updated_at %s)", table, id, d.project, d, want[d.project], instant(&later), bob, instant(&earlier),
					instant(&now), instant(&now))
			}
		}
		for project, state := range want {
			if under[project] != perProject[table] {
				t.Errorf("%s: %d rows under the %s project %s, want %d", table, under[project], state, project, perProject[table])
			}
		}
	}
}

// The four steps, on every project of a workspace, in one transaction,
// soft-delete the workspace's projects, archived ones too, every membership
// of them, active or not, every member's display settings in them and
// every state, the triage ones too, at the same moment and by the same
// account; a row deleted before keeps its time, and another workspace
// keeps everything. Running the steps again changes nothing.
func TestDeletingAWorkspaceSoftDeletesItsProjects(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	earlier, later := now.Add(-time.Hour), now.Add(time.Hour)
	betaWeb, betaOps, betaOld := seedProjects(t, pool, beta, alice, bob, earlier)
	web, ops, old := seedProjects(t, pool, acme, alice, bob, earlier)

	if err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		return deleteAll(ctx, s, app.Deletion{WorkspaceID: acme, By: bob, Now: later})
	}); err != nil {
		t.Fatal(err)
	}
	// Once more, by alice an hour later: nothing is left undeleted.
	if err := deleteAll(context.Background(), s, app.Deletion{WorkspaceID: acme, By: alice, Now: later.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}

	checkDeletions(t, pool, acme, map[uuid.UUID]string{web: "deleted", ops: "deleted", old: "before"}, later, earlier, bob)
	var active []bool
	if err := pool.QueryRow(context.Background(), `SELECT array_agg(is_active ORDER BY is_active) FROM project_members
		WHERE workspace_id = $1 AND deleted_at = $2`, acme, later).Scan(&active); err != nil || len(active) != 4 || active[0] || !active[1] {
		t.Errorf("is_active of acme's deleted memberships: %v, %v; want one false, three true, as they were", active, err)
	}
	checkDeletions(t, pool, beta, map[uuid.UUID]string{betaWeb: "kept", betaOps: "kept", betaOld: "before"}, later, earlier, bob)
}

// unwritten is every row of each project table under workspace, as text,
// without the three columns a deletion writes: what a deletion must leave.
func unwritten(t *testing.T, pool *pgxpool.Pool, workspace uuid.UUID) string {
	t.Helper()
	var all string
	for _, table := range projectTables {
		var rows string
		if err := pool.QueryRow(context.Background(), `SELECT coalesce(string_agg((to_jsonb(r) - 'deleted_at' - 'updated_at' - 'updated_by_id')::text,
			E'\n' ORDER BY r.id), '') FROM `+table.name+` r WHERE r.workspace_id = $1`, workspace).Scan(&rows); err != nil {
			t.Fatal(err)
		}
		all += table.name + ":\n" + rows + "\n"
	}
	return all
}

// The four steps, on one project, soft-delete it and the rows under it
// alone, archived or not, first or last in its workspace, at the same
// moment and by the same account, writing no other column: the workspace's
// other project, its project deleted before and another workspace's
// projects keep everything. Running the steps again changes nothing.
func TestDeletingAProjectSoftDeletesItsRowsAlone(t *testing.T) {
	for _, target := range []string{"Web", "Ops"} {
		t.Run(target, func(t *testing.T) {
			s, pool := newStore(t)
			alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
			acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
			earlier, later := now.Add(-time.Hour), now.Add(time.Hour)
			web, ops, old := seedProjects(t, pool, acme, alice, bob, earlier)
			betaWeb, betaOps, betaOld := seedProjects(t, pool, beta, alice, bob, earlier)
			want := map[uuid.UUID]string{web: "kept", ops: "kept", old: "before"}
			id := map[string]uuid.UUID{"Web": web, "Ops": ops}[target]
			want[id] = "deleted"
			before := unwritten(t, pool, acme)

			if err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
				return deleteAll(ctx, s, app.Deletion{WorkspaceID: acme, ProjectID: &id, By: bob, Now: later})
			}); err != nil {
				t.Fatal(err)
			}
			if err := deleteAll(context.Background(), s, app.Deletion{WorkspaceID: acme, ProjectID: &id, By: alice, Now: later.Add(time.Hour)}); err != nil {
				t.Fatal(err)
			}

			checkDeletions(t, pool, acme, want, later, earlier, bob)
			checkDeletions(t, pool, beta, map[uuid.UUID]string{betaWeb: "kept", betaOps: "kept", betaOld: "before"}, later, earlier, bob)
			if after := unwritten(t, pool, acme); after != before {
				t.Errorf("acme's rows but the deletion's columns:\n%s\nwant\n%s", after, before)
			}
		})
	}
}
````

`server/internal/modules/project/adapter/postgres/failures_test.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("SetArchived() = %v; want context.Canceled", err)
	}
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("SetArchived() = %v; want context.Canceled", err)
	}
	for i, step := range []func(context.Context, app.Deletion) error{s.DeleteProjects, s.DeleteProjectMembers, s.DeleteProjectPreferences,
		s.DeleteStates} {
		if err := step(cancelled, app.Deletion{WorkspaceID: acme, ProjectID: &web, By: alice, Now: now}); !failed(err) {
			t.Errorf("deletion step %d = %v; want context.Canceled", i, err)
		}
	}
````

- [ ] **Step 3: 目录读取按父表**

`server/internal/bootstrap/workspace_deletion_catalog_test.go`（完整内容，150 行）：

````whole server/internal/bootstrap/workspace_deletion_catalog_test.go
package bootstrap

import (
	"context"
	"strings"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The catalog reading of the deletions' guards (workspace_deletion_test.go,
// project_deletion_test.go): the foreign keys to the deleted row's table,
// workspaces or projects, which fail on a table under it without one of its
// own (keysTo, with its counterexample), and each key's rows under a
// deleted row: undeleted, unstamped by the deletion, or all of them as
// text.

// foreignKey is a column that references parent, table as SQL names it.
type foreignKey struct{ parent, table, column string }

func (k foreignKey) String() string { return k.table + "." + k.column }

// keysTo are the parent row itself, as <parent>.id, then every foreign key
// to parent in the catalog. A table under parent only through others'
// foreign keys, followed from parent however far, fails the test, named
// with the tables it hangs from: no check here would see its rows, nor
// would a cascade that deletes by the parent's key column.
func keysTo(t testing.TB, pool *pgxpool.Pool, parent string) []foreignKey {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT c.conrelid::regclass::text, a.attname, cardinality(c.conkey)
		FROM pg_constraint c JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = c.conkey[1]
		WHERE c.contype = 'f' AND c.confrelid = $1::text::regclass
		ORDER BY 1, 2`, parent)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	keys := []foreignKey{{parent, parent, "id"}}
	for rows.Next() {
		k := foreignKey{parent: parent}
		var columns int
		if err := rows.Scan(&k.table, &k.column, &columns); err != nil {
			t.Fatal(err)
		}
		if columns != 1 {
			t.Fatalf("%s references %s with %d columns: find its rows another way", k, parent, columns)
		}
		keys = append(keys, k)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(keys) == 1 {
		t.Fatalf("no foreign key to %s in the catalog: the test would check the %s row only", parent, parent)
	}
	rows, err = pool.Query(context.Background(), `
		WITH RECURSIVE under (tab, via) AS (
			SELECT conrelid, confrelid FROM pg_constraint WHERE contype = 'f' AND confrelid = $1::text::regclass
			UNION
			SELECT c.conrelid, c.confrelid FROM pg_constraint c JOIN under u ON c.confrelid = u.tab WHERE c.contype = 'f')
		SELECT tab::regclass::text || ' (through ' || string_agg(DISTINCT via::regclass::text, ', ' ORDER BY via::regclass::text) || ')'
		FROM under
		WHERE NOT EXISTS (SELECT 1 FROM pg_constraint w WHERE w.contype = 'f' AND w.conrelid = under.tab AND w.confrelid = $1::text::regclass)
		GROUP BY tab ORDER BY 1`, parent)
	if err != nil {
		t.Fatal(err)
	}
	unkeyed, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	if len(unkeyed) > 0 {
		t.Fatalf("%s: under %s with no foreign key to %s of its own, so no check here sees its rows: add %s_id, or extend the guard",
			strings.Join(unkeyed, "; "), parent, parent, strings.TrimSuffix(parent, "s"))
	}
	return keys
}

// undeleted are the ids of k's undeleted rows under the parent row id.
func (k foreignKey) undeleted(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) []uuid.UUID {
	t.Helper()
	rows, err := pool.Query(context.Background(), "SELECT id FROM "+k.table+" WHERE "+pgx.Identifier{k.column}.Sanitize()+
		" = $1 AND deleted_at IS NULL ORDER BY id", id)
	if err != nil {
		t.Fatalf("%s: %v", k, err)
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		t.Fatalf("%s: %v", k, err)
	}
	return ids
}

// unstamped is those of k's rows ids that the deletion of the parent row id
// did not stamp, each as text: a deleted_at other than the parent row's, or
// an updated_by_id other than by. Every table under a workspace or a
// project has both columns.
func (k foreignKey) unstamped(t *testing.T, pool *pgxpool.Pool, ids []uuid.UUID, id, by uuid.UUID) string {
	t.Helper()
	var s string
	if err := pool.QueryRow(context.Background(), "SELECT coalesce(string_agg(r::text, E'\\n' ORDER BY r::text), '') FROM "+k.table+
		" r WHERE r.id = ANY ($1) AND (r.deleted_at IS DISTINCT FROM (SELECT deleted_at FROM "+k.parent+" WHERE id = $2)"+
		" OR r.updated_by_id IS DISTINCT FROM $3)", ids, id, by).Scan(&s); err != nil {
		t.Fatalf("%s: %v", k, err)
	}
	return s
}

// rows is k's rows under the parent row id, deleted ones too, each as text,
// in order; "" when there is none.
func (k foreignKey) rows(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) string {
	t.Helper()
	var s string
	if err := pool.QueryRow(context.Background(), "SELECT coalesce(string_agg(r::text, E'\\n' ORDER BY r::text), '') FROM "+k.table+
		" r WHERE r."+pgx.Identifier{k.column}.Sanitize()+" = $1", id).Scan(&s); err != nil {
		t.Fatalf("%s: %v", k, err)
	}
	return s
}

// keysTo fails on a table under workspaces through projects only, and on
// one under that one, and names both with the tables they hang from; and,
// for projects, on the one under projects only through the other: the
// deletions' checks would see neither's rows.
func TestKeysRefuseATableWithoutItsParent(t *testing.T) {
	pool := openPool(t, pgtest.NewDatabase(t))
	for _, sql := range []string{
		"CREATE TABLE widgets (id uuid PRIMARY KEY, project_id uuid NOT NULL REFERENCES projects, deleted_at timestamptz)",
		"CREATE TABLE gadgets (id uuid PRIMARY KEY, widget_id uuid NOT NULL REFERENCES widgets, deleted_at timestamptz)",
	} {
		if _, err := pool.Exec(context.Background(), sql); err != nil {
			t.Fatal(err)
		}
	}
	for parent, want := range map[string]string{
		"workspaces": "gadgets (through widgets); widgets (through projects): under workspaces with no foreign key to workspaces of its own, " +
			"so no check here sees its rows: add workspace_id, or extend the guard",
		"projects": "gadgets (through widgets): under projects with no foreign key to projects of its own, " +
			"so no check here sees its rows: add project_id, or extend the guard",
	} {
		if failed := fatalOf(func(tb testing.TB) { keysTo(tb, pool, parent) }); failed != want {
			t.Errorf("keysTo(%s) with widgets and gadgets: failed with %q, want %q", parent, failed, want)
		}
	}
}
````

`server/internal/bootstrap/workspace_deletion_test.go`（修改，13 处）：

````old server/internal/bootstrap/workspace_deletion_test.go
// rowsUnder is what one foreign key to workspaces (the workspace row itself
// as workspaces.id) held under the test's two workspaces, before and after
// the deletion: the deleted workspace's undeleted rows, counted, and the
// kept workspace's rows, as text.
````
````new server/internal/bootstrap/workspace_deletion_test.go
// rowsUnder is what one foreign key to the parent's table (the parent row
// itself as its id) held under the test's two parent rows, a workspace or
// a project each, before and after the deletion: the deleted one's
// undeleted rows, counted, and the kept one's rows, as text.
````

````old server/internal/bootstrap/workspace_deletion_test.go
// deletionViolations reports, for each foreign key: no row under either
// workspace before the deletion, which would leave its checks nothing to
// see; an undeleted row left under the deleted workspace, or, for a key on
// exempt, a row deleted that must survive; a row of the kept workspace
// changed. An exempt key that no foreign key matches is reported too.
func deletionViolations(under []rowsUnder, exempt map[string]string) []string {
````
````new server/internal/bootstrap/workspace_deletion_test.go
// deletionViolations reports, for each foreign key to the parent's table
// (parent: "workspace" or "project"): no row under either parent row before
// the deletion, which would leave its checks nothing to see; an undeleted
// row left under the deleted one, or, for a key on exempt, a row deleted
// that must survive; a row of the kept one changed. An exempt key that no
// foreign key matches is reported too.
func deletionViolations(parent string, under []rowsUnder, exempt map[string]string) []string {
````

````old server/internal/bootstrap/workspace_deletion_test.go
			found = append(found, fmt.Sprintf("%s: seed a row under the deleted workspace", r.key))
````
````new server/internal/bootstrap/workspace_deletion_test.go
			found = append(found, fmt.Sprintf("%s: seed a row under the deleted %s", r.key, parent))
````

````old server/internal/bootstrap/workspace_deletion_test.go
			found = append(found, fmt.Sprintf("%s: %d of %d rows deleted with the workspace, want them kept: %s",
				r.key, r.deletedBefore-r.deletedAfter, r.deletedBefore, reason))
````
````new server/internal/bootstrap/workspace_deletion_test.go
			found = append(found, fmt.Sprintf("%s: %d of %d rows deleted with the %s, want them kept: %s",
				r.key, r.deletedBefore-r.deletedAfter, r.deletedBefore, parent, reason))
````

````old server/internal/bootstrap/workspace_deletion_test.go
			found = append(found, fmt.Sprintf("%s: %d rows left undeleted under the deleted workspace: the cascade misses the table",
				r.key, r.deletedAfter))
````
````new server/internal/bootstrap/workspace_deletion_test.go
			found = append(found, fmt.Sprintf("%s: %d rows left undeleted under the deleted %s: the cascade misses the table",
				r.key, r.deletedAfter, parent))
````

````old server/internal/bootstrap/workspace_deletion_test.go
			found = append(found, fmt.Sprintf("%s: seed a row under the kept workspace", r.key))
````
````new server/internal/bootstrap/workspace_deletion_test.go
			found = append(found, fmt.Sprintf("%s: seed a row under the kept %s", r.key, parent))
````

````old server/internal/bootstrap/workspace_deletion_test.go
			found = append(found, fmt.Sprintf("%s: the kept workspace's rows changed:\n%s\nwant\n%s", r.key, r.keptAfter, r.keptBefore))
````
````new server/internal/bootstrap/workspace_deletion_test.go
			found = append(found, fmt.Sprintf("%s: the kept %s's rows changed:\n%s\nwant\n%s", r.key, parent, r.keptAfter, r.keptBefore))
````

````old server/internal/bootstrap/workspace_deletion_test.go
			found = append(found, fmt.Sprintf("the exempt %s is no foreign key to workspaces", key))
````
````new server/internal/bootstrap/workspace_deletion_test.go
			found = append(found, fmt.Sprintf("the exempt %s is no foreign key to %ss", key, parent))
````

````old server/internal/bootstrap/workspace_deletion_test.go
	keys := workspaceKeys(t, pool)
````
````new server/internal/bootstrap/workspace_deletion_test.go
	keys := keysTo(t, pool, "workspaces")
````

````old server/internal/bootstrap/workspace_deletion_test.go
	for _, v := range deletionViolations(under, survivesItsWorkspace) {
````
````new server/internal/bootstrap/workspace_deletion_test.go
	for _, v := range deletionViolations("workspace", under, survivesItsWorkspace) {
````

````old server/internal/bootstrap/workspace_deletion_test.go
	url := pgtest.NewDatabase(t)
	base := startApp(t, testConfig(t, url, false), migrations.FS())
	pool := openPool(t, url)
	admin := registerAccount(t, contract, base, "admin@example.com").AccessToken
	registerAccount(t, contract, base, "member@example.com")
	ids := []uuid.UUID{seedWorkspace(t, contract, base, pool, admin, "deleted"), seedWorkspace(t, contract, base, pool, admin, "kept")}
	rowsOf := func() []string {
		var all []string
		for _, k := range workspaceKeys(t, pool) {
			for _, id := range ids {
				all = append(all, k.String()+":\n"+k.rows(t, pool, id))
			}
		}
		return all
	}
	before := rowsOf()
	exec := func(sql string) {
````
````new server/internal/bootstrap/workspace_deletion_test.go
	url := pgtest.NewDatabase(t)
	base := startApp(t, testConfig(t, url, false), migrations.FS())
	pool := openPool(t, url)
	admin := registerAccount(t, contract, base, "admin@example.com").AccessToken
	registerAccount(t, contract, base, "member@example.com")
	ids := []uuid.UUID{seedWorkspace(t, contract, base, pool, admin, "deleted"), seedWorkspace(t, contract, base, pool, admin, "kept")}
	rowsOf := func() []string {
		var all []string
		for _, k := range keysTo(t, pool, "workspaces") {
			for _, id := range ids {
				all = append(all, k.String()+":\n"+k.rows(t, pool, id))
			}
		}
		return all
	}
	before := rowsOf()
	exec := func(sql string) {
````

````old server/internal/bootstrap/workspace_deletion_test.go
	url := pgtest.NewDatabase(t)
	base := startApp(t, testConfig(t, url, false), migrations.FS())
	pool := openPool(t, url)
	admin := registerAccount(t, contract, base, "admin@example.com").AccessToken
	registerAccount(t, contract, base, "member@example.com")
	ids := []uuid.UUID{seedWorkspace(t, contract, base, pool, admin, "deleted"), seedWorkspace(t, contract, base, pool, admin, "kept")}
	rowsOf := func() []string {
		var all []string
		for _, k := range workspaceKeys(t, pool) {
			for _, id := range ids {
				all = append(all, k.String()+":\n"+k.rows(t, pool, id))
			}
		}
		return all
	}
	before := rowsOf()
	for _, sql := range []string{
````
````new server/internal/bootstrap/workspace_deletion_test.go
	url := pgtest.NewDatabase(t)
	base := startApp(t, testConfig(t, url, false), migrations.FS())
	pool := openPool(t, url)
	admin := registerAccount(t, contract, base, "admin@example.com").AccessToken
	registerAccount(t, contract, base, "member@example.com")
	ids := []uuid.UUID{seedWorkspace(t, contract, base, pool, admin, "deleted"), seedWorkspace(t, contract, base, pool, admin, "kept")}
	rowsOf := func() []string {
		var all []string
		for _, k := range keysTo(t, pool, "workspaces") {
			for _, id := range ids {
				all = append(all, k.String()+":\n"+k.rows(t, pool, id))
			}
		}
		return all
	}
	before := rowsOf()
	for _, sql := range []string{
````

````old server/internal/bootstrap/workspace_deletion_test.go
		if got := deletionViolations(tt.under, tt.exempt); !slices.Equal(got, tt.want) {
````
````new server/internal/bootstrap/workspace_deletion_test.go
		if got := deletionViolations("workspace", tt.under, tt.exempt); !slices.Equal(got, tt.want) {
````

- [ ] **Step 4: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/project/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt|TestAFailedProjectsStepRollsTheDeletionBack|TestADeletionRefusedAtItsCommitChangesNoRow|TestKeysRefuseATableWithoutItsParent|TestDeletionViolationsCatchesEachGap' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 5: 提交**

```bash
git add server/internal/bootstrap/workspace_deletion_catalog_test.go server/internal/bootstrap/workspace_deletion_test.go server/internal/modules/project/adapter/postgres/cascade.go server/internal/modules/project/adapter/postgres/cascade_test.go server/internal/modules/project/adapter/postgres/failures_test.go server/internal/modules/project/adapter/postgres/queries/cascade.sql server/internal/modules/project/app/cascade.go server/internal/modules/project/app/cascade_test.go server/internal/modules/project/app/deletion.go server/internal/modules/project/app/fakes_write_test.go server/internal/modules/project/app/ports.go server/internal/modules/project/adapter/postgres/gen/cascade.sql.go
```
```bash
git commit -m "refactor(M3/P4b): one list of a project deletion's steps, by workspace or project

The four statements that delete projects and the rows under them take an
optional project, so deleting a workspace and deleting one project run
the same steps, listed once in app/deletion.go, where P7 adds the labels.
The catalog reading of the deletion guards takes the parent table, and
refuses a table under projects with no project_id of its own.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| 四条语句各去掉项目（删除工作区的每个项目） | `TestDeletingAProjectSoftDeletesItsRowsAlone`；P4、W3（Task 15 起） | 存储；端到端 |
| 四条语句各去掉工作区 | `TestDeletingAWorkspaceSoftDeletesItsProjects`；W3 | 存储；端到端 |
| 四条语句各去掉 `deleted_at IS NULL`（再删此前删除的） | `TestDeletingAWorkspaceSoftDeletesItsProjects`、`TestDeletingAProjectSoftDeletesItsRowsAlone`；W3（Task 15 起，W3 先删除 Old） | 存储；端到端 |
| 只删一个项目时四张表各也写 `created_at`、不写 `updated_by_id` | `TestDeletingAProjectSoftDeletesItsRowsAlone` | 存储 |
| 状态在另一个时刻删除 | `TestDeletingAWorkspaceSoftDeletesItsProjects`、`TestDeletingAProjectSoftDeletesItsRowsAlone`；`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`（Task 6 起）、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`；P4（Task 15 起） | 存储；组合；端到端 |
| `deleteProjects` 少状态一步 | `TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`；`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`（Task 6 起） | 组合 |
| 一步的失败被吞掉、之后的照跑；一步重试 | `TestDeleteWorkspaceProjects`；`TestDeleteProjectReturnsEachFailure`（Task 6 起） | 单元 |
| 四步各经连接池、在事务之外执行 | `TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 14 起） | 组合 |

**Done when:** 删除工作区照旧（三个组合测试不加豁免）；删除一个项目只动它和它下面的行；`keysTo` 的反例对两个父表都成立。

---

### Task 6: `deleteProject`

**Files:**
- Create: `server/internal/bootstrap/project_deletion_test.go`、`server/internal/modules/project/adapter/http/delete_test.go`、`server/internal/modules/project/app/delete_project.go`、`server/internal/modules/project/app/delete_project_test.go`
- Modify: `api/modules/project.yaml`、`server/internal/bootstrap/permission_matrix_project_test.go`、`server/internal/bootstrap/project_write_locks_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/project/adapter/http/handler.go`、`server/internal/modules/project/adapter/http/handler_test.go`、`server/internal/modules/project/adapter/http/projects.go`、`server/internal/modules/project/app/clock_test.go`、`server/internal/modules/project/domain/actions.go`、`server/internal/modules/project/module.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.6，M3 设计 3.3、3.6、3.19）：`DELETE /api/v0/projects/{project_id}`，204；码 `[project.not_found, forbidden]`；操作名 `project.delete`，规则同 `project.update`；已归档的项目照样删除。
- `app.NewDeleteProject(projects ProjectsDeleter, locks Locks, tx, clock)`：一个事务：`locks.lockAndDecide`（工作区 `FOR SHARE` → 项目 `FOR NO KEY UPDATE` → 判定）→ 锁下读时钟 → `deleteProjects(Deletion{WorkspaceID: 锁读到的工作区, ProjectID: &id, By: 调用者, Now})`。删除用锁读到的工作区（M3 设计 3.6 约定二：锁下的重读确认父行）。用例只要 `ProjectsDeleter`：不另设删除项目的端口。

**Tests:**
- `app/delete_project_test.go`：`TestDeleteProject`（web、已归档的 ops：锁、判定、时钟、四步，只有这个项目）；`TestDeleteProjectRefuses`（没有调用者、没有、看不到、项目成员）；`TestDeleteProjectReturnsEachFailure`（锁、判定、四步各一、提交）。
- `adapter/http/delete_test.go`：`TestDeleteProject`（204、没有正文；两种拒绝）。
- `bootstrap/project_deletion_test.go`：`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`（`keysTo("projects")` 读出每张表；经 API 删除 Web：每个外键下删除之前未删除的行都删除了，`deleted_at` 是项目的、`updated_by_id` 是删除者；Ops 的行一行不变；Web 的 `deleted_at` 在请求之内；读 Web 得 404）；`TestAProjectDeletionRefusedAtItsCommitChangesNoRow`（`states` 上的延迟约束触发器拒绝提交：500，两个项目下的每一行不变；`states` 是最后一步的表，一步单独提交就会留下删除的行）。
- 矩阵两行（12 格 `ofProject(204, 403, 403, 204, 403, 403)`；已归档 1 格 204）。
- `TestEachWriteOnAProjectSharesItsWorkspaceFirst` 加删除一行（最后一行：两个项目都删除之后不再有别的写）。

- [ ] **Step 1: 接口描述**

`api/modules/project.yaml`（修改，1 处）：

````old api/modules/project.yaml
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Project'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/projects/{project_id}/archive:
````
````new api/modules/project.yaml
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Project'
        default:
          $ref: '#/components/responses/Problem'
    delete:
      operationId: deleteProject
      tags: [project]
      summary: Delete a project
      description: >-
        For the project's admins, and its members who are the workspace's
        admins; an archived project is deleted as any other. The project is
        deleted with its memberships, its members' display settings and its
        states, all at one moment: it is no longer read, listed or
        changed, and its name and identifier are free again in the
        workspace. A project that does not exist, is deleted, or that the
        caller does not see answers project.not_found; one he sees but may
        not delete, forbidden. The role is decided after the project row is
        locked.
      security: [{bearer: []}]
      x-problem-codes: [project.not_found, forbidden]
      responses:
        '204':
          description: The project is deleted.
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/projects/{project_id}/archive:
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `49d9688e05f09c02820819a044c4797afedd6215b252ef7d9c9b782d5650c129` | 2143 | `api/dist/openapi.yaml` |
| `66c37c8db6b47c2317eb39e71428c19c52be58c2972bc74ac909b5fa69a8d752` | 1326 | `server/internal/modules/project/adapter/http/gen/server.gen.go` |
| `815ef07f9177d1948373499d297d66cee457ca5438abc379b558dd075cd08fdf` | 2281 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: 操作名、规则、用例**

`server/internal/modules/project/domain/actions.go`（修改，2 处）：

````old server/internal/modules/project/domain/actions.go
	ActionUnarchive shared.Action = "project.unarchive"
````
````new server/internal/modules/project/domain/actions.go
	ActionUnarchive shared.Action = "project.unarchive"
	// ActionDelete is deleting a project: deleteProject.
	ActionDelete shared.Action = "project.delete"
````

````old server/internal/modules/project/domain/actions.go
		ActionUnarchive}
````
````new server/internal/modules/project/domain/actions.go
		ActionUnarchive, ActionDelete}
````

`server/internal/modules/access/domain/rules.go`（修改，2 处）：

````old server/internal/modules/access/domain/rules.go
	// As project.update (M3 design 3.4, §12 P4b).
````
````new server/internal/modules/access/domain/rules.go
	// As project.update (M3 design 3.4, 9.2).
````

````old server/internal/modules/access/domain/rules.go
	"project.unarchive": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
````
````new server/internal/modules/access/domain/rules.go
	"project.unarchive": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	"project.delete":    {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
		invisible, forbidden, invisible, invisible, invisible, forbidden, forbidden},
````
````new server/internal/modules/access/domain/rules_test.go
		invisible, forbidden, invisible, invisible, invisible, forbidden, forbidden},
	"project.delete": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
````

`server/internal/modules/project/app/delete_project.go`（新文件，43 行）：

````file server/internal/modules/project/app/delete_project.go
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DeleteProject deletes a project: DELETE /api/v0/projects/{project_id}
// (M3 design 3.3, 3.6).
type DeleteProject struct {
	projects ProjectsDeleter
	locks    Locks
	tx       shared.TxManager
	clock    Clock
}

// NewDeleteProject returns the use case.
func NewDeleteProject(projects ProjectsDeleter, locks Locks, tx shared.TxManager, clock Clock) *DeleteProject {
	return &DeleteProject{projects: projects, locks: locks, tx: tx, clock: clock}
}

// Execute, in one transaction (M3 design 3.6): the project's locks (Locks:
// its workspace FOR SHARE, then the project FOR NO KEY UPDATE) and the
// decision on project.delete; the clock read under the locks; then the
// project and the rows under it soft-deleted, each at that one moment, by
// the caller (deleteProjects). An archived project is deleted as any
// other.
func (u *DeleteProject) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	return u.tx.WithinTx(ctx, func(ctx context.Context) error {
		h, err := u.locks.lockAndDecide(ctx, actor, write{project: id, action: domain.ActionDelete})
		if err != nil {
			return err
		}
		return deleteProjects(ctx, u.projects, Deletion{WorkspaceID: h.project.WorkspaceID, ProjectID: &id, By: actor.UserID, Now: u.clock.Now()})
	})
}
````

`server/internal/modules/project/app/delete_project_test.go`（新文件，94 行）：

````file server/internal/modules/project/app/delete_project_test.go
package app_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newDelete is DeleteProject over newWrites' fakes, its clock logged.
func newDelete() (*app.DeleteProject, *writeFixture) {
	f := newWrites()
	return app.NewDeleteProject(f.store, f.locks(), f.tx, clockAt{clockNow, f.log}), f
}

// deleted are the calls of bob's deletion of project: the locks, the
// decision, the clock, the four steps on that project alone.
func deleted(project uuid.UUID) []string {
	return slices.Concat(lockedDecision(bob, project, domain.ActionDelete), []string{"Now"},
		deletionSteps(acme.ID, project.String(), bob, clockNow))
}

// DeleteProject, in one transaction and in the order of M3 design 3.6,
// takes the project's locks, decides, reads the clock, then runs the steps a
// workspace's deletion runs, on the project alone, of the workspace its
// lock read, by the caller at that one moment. An archived project is
// deleted as any other.
func TestDeleteProject(t *testing.T) {
	for _, project := range []uuid.UUID{webID, opsID} {
		uc, f := newDelete()
		if err := uc.Execute(as(bob), project); err != nil || !slices.Equal(f.log.calls, deleted(project)) {
			t.Errorf("Execute(%s) = %v, calls\n%q\nwant\n%q", project, err, f.log.calls, deleted(project))
		}
	}
}

// Refusals, each in its place, and nothing deleted: no caller, before the
// transaction; a project not there, or not visible: 404; a project member:
// the Authorizer's 403.
func TestDeleteProjectRefuses(t *testing.T) {
	tests := []struct {
		name  string
		ctx   context.Context
		id    uuid.UUID
		want  error
		calls []string
	}{
		{"no caller", context.Background(), webID, shared.Unauthenticated(), nil},
		{"no project", as(bob), uuid.Nil(), domain.ErrNotFound, noProject},
		{"not seen", as(erin), webID, domain.ErrNotFound, lockedDecision(erin, webID, domain.ActionDelete)},
		{"a project member", as(alice), webID, shared.Forbidden(), lockedDecision(alice, webID, domain.ActionDelete)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newDelete()
			if err := uc.Execute(tt.ctx, tt.id); !sameError(err, tt.want) || !slices.Equal(f.log.calls, tt.calls) {
				t.Errorf("Execute() = %v, calls %q; want %v, calls %q", err, f.log.calls, tt.want, tt.calls)
			}
		})
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after.
func TestDeleteProjectReturnsEachFailure(t *testing.T) {
	all := deleted(webID)
	tests := []struct {
		name  string
		fail  func(f *writeFixture)
		calls int // how many of all's ran
	}{
		{"the lock", func(f *writeFixture) { f.store.errs = map[string]error{"LockProject": errDisk} }, 4},
		{"the decision", func(f *writeFixture) { f.auth.errs = map[grantKey]error{{bob, acme.ID}: errDisk} }, 5},
		{"the projects", func(f *writeFixture) { f.store.errs = map[string]error{"DeleteProjects": errDisk} }, 7},
		{"the members", func(f *writeFixture) { f.store.errs = map[string]error{"DeleteProjectMembers": errDisk} }, 8},
		{"the display settings", func(f *writeFixture) { f.store.errs = map[string]error{"DeleteProjectPreferences": errDisk} }, 9},
		{"the states", func(f *writeFixture) { f.store.errs = map[string]error{"DeleteStates": errDisk} }, 10},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newDelete()
			tt.fail(f)
			if err := uc.Execute(as(bob), webID); !errors.Is(err, errDisk) || !slices.Equal(f.log.calls, all[:tt.calls]) {
				t.Errorf("Execute() = %v, calls %q; want %v after %q", err, f.log.calls, errDisk, all[:tt.calls])
			}
		})
	}
}
````

`server/internal/modules/project/app/clock_test.go`（修改，1 处）：

````old server/internal/modules/project/app/clock_test.go
		}, archived(opsID, false)},
````
````new server/internal/modules/project/app/clock_test.go
		}, archived(opsID, false)},
		{"deleteProject", func() ([]string, error) {
			uc, f := newDelete()
			return f.log.calls, uc.Execute(as(bob), webID)
		}, deleted(webID)},
````

- [ ] **Step 3: HTTP 和接线**

`server/internal/modules/project/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/project/adapter/http/handler.go
}

// UseCases are the use cases behind the module's operations.
````
````new server/internal/modules/project/adapter/http/handler.go
}

// DeleteProjectUseCase is app.DeleteProject.
type DeleteProjectUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) error
}

// UseCases are the use cases behind the module's operations.
````

````old server/internal/modules/project/adapter/http/handler.go
	UnarchiveProject ArchiveProjectUseCase
````
````new server/internal/modules/project/adapter/http/handler.go
	UnarchiveProject ArchiveProjectUseCase
	DeleteProject    DeleteProjectUseCase
````

`server/internal/modules/project/adapter/http/projects.go`（修改，1 处）：

````old server/internal/modules/project/adapter/http/projects.go
}

// named reports whether the body names v, and its id: nil when it is null.
````
````new server/internal/modules/project/adapter/http/projects.go
}

// DeleteProject serves DELETE /api/v0/projects/{project_id}.
func (h handler) DeleteProject(ctx context.Context, req gen.DeleteProjectRequestObject) (gen.DeleteProjectResponseObject, error) {
	if err := h.uc.DeleteProject.Execute(ctx, req.ProjectID); err != nil {
		return nil, err
	}
	return gen.DeleteProject204Response{}, nil
}

// named reports whether the body names v, and its id: nil when it is null.
````

`server/internal/modules/project/adapter/http/handler_test.go`（修改，4 处）：

````old server/internal/modules/project/adapter/http/handler_test.go
	unarchive *fakeOnProject
````
````new server/internal/modules/project/adapter/http/handler_test.go
	unarchive *fakeOnProject
	delete    *fakeDelete
````

````old server/internal/modules/project/adapter/http/handler_test.go
}

// newServer serves the module with f; a fake left nil is an idle one.
````
````new server/internal/modules/project/adapter/http/handler_test.go
}

type fakeDelete struct {
	calls []string // "caller id"
	err   error
}

func (f *fakeDelete) Execute(ctx context.Context, id uuid.UUID) error {
	f.calls = append(f.calls, caller(ctx)+" "+id.String())
	return f.err
}

// newServer serves the module with f; a fake left nil is an idle one.
````

````old server/internal/modules/project/adapter/http/handler_test.go
		f.unarchive = &fakeOnProject{}
	}
````
````new server/internal/modules/project/adapter/http/handler_test.go
		f.unarchive = &fakeOnProject{}
	}
	if f.delete == nil {
		f.delete = &fakeDelete{}
	}
````

````old server/internal/modules/project/adapter/http/handler_test.go
		CheckIdentifier: f.check, UpdateProject: f.update, ArchiveProject: f.archive, UnarchiveProject: f.unarchive})
````
````new server/internal/modules/project/adapter/http/handler_test.go
		CheckIdentifier: f.check, UpdateProject: f.update, ArchiveProject: f.archive, UnarchiveProject: f.unarchive, DeleteProject: f.delete})
````

`server/internal/modules/project/adapter/http/delete_test.go`（新文件，39 行）：

````file server/internal/modules/project/adapter/http/delete_test.go
package httpadapter_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DELETE goes to the use case for the caller and the path's project, and
// answers 204 with no body; the use case's refusals, as the contract
// declares them.
func TestDeleteProject(t *testing.T) {
	del := &fakeDelete{}
	h := newServer(t, fakes{delete: del})
	path := "/api/v0/projects/" + webID.String()
	if res, body := do(t, h, request(http.MethodDelete, path, "bob", "")); res.StatusCode != http.StatusNoContent || body != "" {
		t.Errorf("DELETE = %d %q, want 204 and no body", res.StatusCode, body)
	}
	if want := []string{"bob " + webID.String()}; !slices.Equal(del.calls, want) {
		t.Errorf("calls = %q, want %q", del.calls, want)
	}
	for _, tt := range []struct {
		err    error
		status int
		want   string
	}{
		{domain.ErrNotFound, http.StatusNotFound,
			`{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`},
		{shared.Forbidden(), http.StatusForbidden, `{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
	} {
		h := newServer(t, fakes{delete: &fakeDelete{err: tt.err}})
		if res, body := do(t, h, request(http.MethodDelete, path, "alice", "")); res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("DELETE = %d %s, want %d %s", res.StatusCode, body, tt.status, tt.want)
		}
	}
}
````

`server/internal/modules/project/module.go`（修改，2 处）：

````old server/internal/modules/project/module.go
// brings listing, creating, reading, changing and archiving projects and
// checking an identifier, carries out the workspace module's cascades on the projects
````
````new server/internal/modules/project/module.go
// brings listing, creating, reading, changing, archiving and deleting
// projects and checking an identifier, carries out the workspace module's cascades on the projects
````

````old server/internal/modules/project/module.go
		UnarchiveProject: app.NewUnarchiveProject(store, locks, d.Tx, d.Clock),
````
````new server/internal/modules/project/module.go
		UnarchiveProject: app.NewUnarchiveProject(store, locks, d.Tx, d.Clock),
		DeleteProject:    app.NewDeleteProject(store, locks, d.Tx, d.Clock),
````

- [ ] **Step 4: 矩阵和组合出的测试**

`server/internal/bootstrap/permission_matrix_project_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_project_test.go
			cells: map[caller]cell{callerArchivedAdmin: cellOK}, check: archivesItsProject(false)},
````
````new server/internal/bootstrap/permission_matrix_project_test.go
			cells: map[caller]cell{callerArchivedAdmin: cellOK}, check: archivesItsProject(false)},
		// As updateProject; an archived project is deleted as any other.
		{op: "deleteProject", write: true, columns: projectColumns, request: toProject(http.MethodDelete, "", ""),
			cells: ofProject(cellNoContent, cellForbidden, cellForbidden, cellNoContent, cellForbidden, cellForbidden)},
		{op: "deleteProject", variant: "archived", write: true, columns: archivedColumns, request: toProject(http.MethodDelete, "", ""),
			cells: map[caller]cell{callerArchivedAdmin: cellNoContent}},
````

`server/internal/bootstrap/project_deletion_test.go`（新文件，118 行）：

````file server/internal/bootstrap/project_deletion_test.go
package bootstrap

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// The cascade of deleteProject against the catalog (M3 design 3.3, 3.6,
// 4), as deleteWorkspace's (workspace_deletion_test.go): a project deleted
// through the wired app leaves no undeleted row under it, in any table with
// a foreign key to projects; deletes each at the project's instant, by its
// deleter; and changes no row of the workspace's other project. The tables
// come from pg_constraint (keysTo): a phase that adds a table under
// projects fails this test until createProject or the seed here writes a
// row of it and the deletion deletes it.

// twoProjects is the wired app on a database of its own, and acme with two
// projects alice created, Web and Ops: each has her membership, her display
// settings in it and its states, through createProject.
func twoProjects(t *testing.T) (contract *apitest.Contract, base string, pool *pgxpool.Pool, alice string, aliceID, web, ops uuid.UUID) {
	t.Helper()
	contract = apitest.Load(t)
	url := pgtest.NewDatabase(t)
	base = startApp(t, testConfig(t, url, false), migrations.FS())
	pool = openPool(t, url)
	alice = registerAccount(t, contract, base, "alice@example.com").AccessToken
	aliceID = accountID(t, contract, base, alice)
	if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", alice, `{"name":"Acme","slug":"acme"}`); status != http.StatusCreated {
		t.Fatalf("creating acme = %d %s", status, body)
	}
	return contract, base, pool, alice, aliceID, createdProject(t, contract, base, alice, "acme", "Web", "WEB"),
		createdProject(t, contract, base, alice, "acme", "Ops", "OPS")
}

// Deleting a project through the API soft-deletes every row under it in
// every table the catalog ties to projects, and the project row, each at
// the project's deleted_at, a time within the request, and by the account
// that deleted it, and changes nothing under the workspace's other project.
// The deleted project is not found any more.
func TestDeletingAProjectLeavesNoUndeletedRowUnderIt(t *testing.T) {
	contract, base, pool, alice, aliceID, web, ops := twoProjects(t)
	keys := keysTo(t, pool, "projects")
	under, recorded := make([]rowsUnder, len(keys)), make([][]uuid.UUID, len(keys))
	for i, k := range keys {
		recorded[i] = k.undeleted(t, pool, web)
		under[i] = rowsUnder{key: k.String(), deletedBefore: len(recorded[i]), keptBefore: k.rows(t, pool, ops)}
	}

	before := time.Now().Truncate(time.Microsecond)
	if status, body := call(t, contract, http.MethodDelete, base+"/api/v0/projects/"+web.String(), alice, ""); status != http.StatusNoContent {
		t.Fatalf("deleting Web = %d %s, want 204", status, body)
	}
	after := time.Now()

	for i, k := range keys {
		under[i].deletedAfter, under[i].keptAfter = len(k.undeleted(t, pool, web)), k.rows(t, pool, ops)
	}
	for _, v := range deletionViolations("project", under, nil) {
		t.Error(v)
	}
	for i, k := range keys {
		if rows := k.unstamped(t, pool, recorded[i], web, aliceID); rows != "" {
			t.Errorf("%s: rows not deleted at the project's deleted_at by its deleter %s:\n%s", k, aliceID, rows)
		}
	}
	var at time.Time
	if err := pool.QueryRow(context.Background(), "SELECT deleted_at FROM projects WHERE id = $1", web).Scan(&at); err != nil ||
		at.Before(before) || at.After(after) {
		t.Errorf("Web's deleted_at = %v, %v; want within the request, %v to %v", at, err, before, after)
	}
	if status, body := call(t, contract, http.MethodGet, base+"/api/v0/projects/"+web.String(), alice, ""); status != http.StatusNotFound {
		t.Errorf("reading Web deleted = %d %s, want 404", status, body)
	}
}

// Every statement of the deletion runs in its one transaction (M3 design
// 3.3, 9.3): a deletion refused at its commit, after every statement ran,
// changes no row under either project. A deferred constraint trigger on
// states refuses it: the last step's table, so a step that committed on
// its own, before it, would leave its rows deleted.
func TestAProjectDeletionRefusedAtItsCommitChangesNoRow(t *testing.T) {
	contract, base, pool, alice, _, web, ops := twoProjects(t)
	rowsOf := func() []string {
		var all []string
		for _, k := range keysTo(t, pool, "projects") {
			for _, id := range []uuid.UUID{web, ops} {
				all = append(all, k.String()+":\n"+k.rows(t, pool, id))
			}
		}
		return all
	}
	before := rowsOf()
	for _, sql := range []string{
		`CREATE FUNCTION refuse_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'the commit is refused'; END $$`,
		`CREATE CONSTRAINT TRIGGER refuse_commit AFTER UPDATE ON states DEFERRABLE INITIALLY DEFERRED
			FOR EACH ROW EXECUTE FUNCTION refuse_commit()`,
	} {
		if _, err := pool.Exec(context.Background(), sql); err != nil {
			t.Fatal(err)
		}
	}
	if status, body := call(t, contract, http.MethodDelete, base+"/api/v0/projects/"+web.String(), alice, ""); status != http.StatusInternalServerError {
		t.Fatalf("deleting Web with its commit refused = %d %s, want 500", status, body)
	}
	if after := rowsOf(); !slices.Equal(after, before) {
		t.Errorf("the rows after the refused deletion:\n%q\nwant\n%q", after, before)
	}
}
````

`server/internal/bootstrap/project_write_locks_test.go`（修改，1 处）：

````old server/internal/bootstrap/project_write_locks_test.go
	{op: "unarchiveProject", method: http.MethodPost, path: "/api/v0/projects/%s/unarchive", want: http.StatusOK},
````
````new server/internal/bootstrap/project_write_locks_test.go
	{op: "unarchiveProject", method: http.MethodPost, path: "/api/v0/projects/%s/unarchive", want: http.StatusOK},
	{op: "deleteProject", method: http.MethodDelete, path: "/api/v0/projects/%s", want: http.StatusNoContent},
````

- [ ] **Step 5: 测试、lint、前端检查**

Run: `go -C server test -count=1 ./internal/modules/project/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix$|TestThePermissionMatrixCoversEveryOperation|TestMatrixViolationsCatchesEach|TestEveryActionHasARuleAndEveryRuleAnAction|TestAPIRoutesAreTheContractsOperations|TestOperationsThatNeedATokenAnswer401WithoutOne|TestDeletingAProjectLeavesNoUndeletedRowUnderIt|TestAProjectDeletionRefusedAtItsCommitChangesNoRow|TestTheWritesOnAProjectStampTheirRequest|TestEachWriteOnAProjectSharesItsWorkspaceFirst' ./internal/bootstrap/`
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
git add api/modules/project.yaml server/internal/bootstrap/permission_matrix_project_test.go server/internal/bootstrap/project_deletion_test.go server/internal/bootstrap/project_write_locks_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/project/adapter/http/delete_test.go server/internal/modules/project/adapter/http/handler.go server/internal/modules/project/adapter/http/handler_test.go server/internal/modules/project/adapter/http/projects.go server/internal/modules/project/app/clock_test.go server/internal/modules/project/app/delete_project.go server/internal/modules/project/app/delete_project_test.go server/internal/modules/project/domain/actions.go server/internal/modules/project/module.go api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P4b): deleteProject

DELETE /api/v0/projects/{project_id}, for whoever may change the project,
archived or not. It takes Locks' path (the workspace FOR SHARE, then the
project FOR NO KEY UPDATE), decides, reads the clock under the locks and
runs the steps a workspace's deletion runs, on the project alone, in the
workspace its lock read. The composed test
reads the tables under projects from the catalog, as the workspace's
does, and a commit refused after the last step changes no row.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| 规则给项目成员、给不是成员的工作区管理员 | `TestPermissionMatrix` | 组合 |
| 删除用别的工作区（调用者的 id）而不是锁读到的 | `TestDeletingAProjectLeavesNoUndeletedRowUnderIt` | 组合 |
| 没有调用者时也读 | `TestDeleteProjectRefuses` | 单元 |
| 锁和判定、四步、提交的失败被吞掉 | `TestDeleteProjectReturnsEachFailure` | 单元 |
| 只有删除不锁工作区 | `TestEachWriteOnAProjectSharesItsWorkspaceFirst`；`TestADemotionAndAProjectsDeletionSerialize`（Task 14 起） | 组合 |
| 测试改坏：`projectWrites` 少了删除的一行 | `TestEachWriteOnAProjectSharesItsWorkspaceFirst` 自己失败（与矩阵的写对不上，"each has its row, in projectWrites"） | 组合 |
| handler 吞掉用例的失败 | `TestDeleteProject`（HTTP） | 单元 |
| `project.New` 的接线 | 见 Task 8 的表 | 组合 |

**Done when:** 矩阵两行（13 格）通过；删除项目由目录驱动的组合测试核对每张表的每一行，提交被拒时一行不变。

---

### Task 7: 项目的显示设置：领域、存储、两个用例

**Files:**
- Create: `server/internal/modules/project/adapter/postgres/preferences.go`、`server/internal/modules/project/adapter/postgres/preferences_test.go`、`server/internal/modules/project/app/fakes_preferences_test.go`、`server/internal/modules/project/app/get_preferences.go`、`server/internal/modules/project/app/preferences_test.go`、`server/internal/modules/project/app/update_preferences.go`
- Modify: `server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/project/adapter/postgres/failures_test.go`、`server/internal/modules/project/adapter/postgres/queries/preferences.sql`、`server/internal/modules/project/adapter/postgres/queries/projects.sql`、`server/internal/modules/project/adapter/postgres/update.go`、`server/internal/modules/project/app/clock_test.go`、`server/internal/modules/project/app/fakes_write_test.go`、`server/internal/modules/project/app/lock.go`、`server/internal/modules/project/app/ports.go`、`server/internal/modules/project/domain/actions.go`、`server/internal/modules/project/domain/preferences.go`、`server/internal/modules/project/domain/preferences_test.go`
- Generate: `server/internal/modules/project/adapter/postgres/gen/preferences.sql.go`、`server/internal/modules/project/adapter/postgres/gen/projects.sql.go`

**Interfaces:**
- Produces（spec 2.7，M3 设计 3.18、4.8、5.2）：`domain.Navigation{DefaultTab, HideInMoreMenu}`、`Preferences{Navigation, SortOrder}`、`PreferencesPatch`（`nil` 不变，导航整个替换）、`DefaultPreferences()`（列的默认值：`work_items`、什么都不藏、65535）、`(Preferences).Apply`、`CheckPreferencesPatch`：默认标签页是五个之一，藏起的是 `work_items` 之外的四个之一且只出现一次，全部问题一个 422，按位置命名字段。
- 端口：`ProjectSharer.ShareProject`（`FOR SHARE`，带 `deleted_at IS NULL`），进 `ProjectLocks`；`PreferencesReader`（嵌入 Task 2 的 `ProjectFinder`）、`PreferencesChange`、`PreferencesWriter`（只有 `UpsertPreferences`：锁在 `Locks` 里）。`app/lock.go`：`write` 加 `share`（为真时项目取 `FOR SHARE`：改项目之下的行、不改项目行和成员关系的写），加 `findAndDecide`（读：找到工作区 → 判定，不开事务）。
- `app.NewGetProjectPreferences(preferences, auth)`：`findAndDecide(project_preferences.read)` → `Preferences`，没有行时答默认值，不写（M3 设计 3.18）。`app.NewUpdateProjectPreferences(preferences PreferencesWriter, locks Locks, tx, clock)`：`CheckPreferencesPatch`（事务之前）→ 一个事务：`locks.lockAndDecide(write{project, project_preferences.update, share: true})`（工作区 `FOR SHARE` → 项目 `FOR SHARE` → 判定）→ 锁下读时钟 → `UpsertPreferences`（有未删除的行就改给出的字段和审计列，没有就以默认值加上修改插入，冲突的目标是部分唯一键）。已归档项目的设置照改（3.19）。
- 规则：`project_preferences.read`、`project_preferences.update`：项目的有效成员（三种角色），各自的设置。

**Tests:**
- `domain/preferences_test.go`：`TestPreferencesApply`；`TestCheckPreferencesPatchAcceptsEveryTab`；`TestCheckPreferencesPatchReportsEveryTab`（未知的默认标签页、藏起 `work_items`、未知的、重复的、大小写不同的：一个 422 按位置）。
- `adapter/postgres/preferences_test.go`：`TestShareProject`（`ShareProject` 持锁时 `FOR NO KEY UPDATE` 等待、另一个 `FOR SHARE` 不等、别的项目不被锁；已删除、不存在的找不到；已归档的也读）；`TestPreferences`（全默认值的行读成 `DefaultPreferences`；别的账户、别的项目、已删除的行都不读）；`TestUpsertPreferences`（第一次插入：默认值加修改，由他在那一刻建；行由别人建、最后由别人改时，下一次修改的审计列是他和那一刻；导航整个替换；空的修改不变；别的行（别人的、别的项目的、他已删除的）每一列不变）。
- `app/preferences_test.go`：`TestGetProjectPreferences`（bob 的设置、carol 没有时的默认值；不在事务里）；`TestGetProjectPreferencesRefuses`（没有调用者、没有、看不到、不是成员的 403；三个端口的失败各自原样返回，之后的不调用）；`TestUpdateProjectPreferences`（改 bob 的导航、从默认值建 carol 的位置、已归档项目里的）；`TestUpdateProjectPreferencesRefuses`（未知的标签页、没有调用者在事务之前；没有、看不到、403；锁、判定、写、提交的失败；被拒时设置不变）。

- [ ] **Step 1: 领域**

`server/internal/modules/project/domain/preferences.go`（修改，2 处）：

````old server/internal/modules/project/domain/preferences.go
package domain
````
````new server/internal/modules/project/domain/preferences.go
package domain

import (
	"fmt"
	"slices"
	"strings"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)
````

````old server/internal/modules/project/domain/preferences.go
	return *lowest - 10000
}

````
````new server/internal/modules/project/domain/preferences.go
	return *lowest - 10000
}

// tabs are a project's tabs, in the web app's order (M3 design 5.2;
// core/components/navigation/use-navigation-items.ts:38-78). The first,
// the work items, is the default tab and never hidden.
var tabs = []string{"work_items", "cycles", "modules", "views", "intake"}

// Navigation is the tab bar of a project's header, as one account has it:
// the tab the project opens on, and the tabs moved under "more".
type Navigation struct {
	DefaultTab     string
	HideInMoreMenu []string
}

// Preferences are an account's display settings in a project (M3 design
// 3.18, 5.2): its tab bar, and the project's place in his sidebar.
type Preferences struct {
	Navigation Navigation
	SortOrder  float64
}

// PreferencesPatch is a change of Preferences: a nil field stays as it
// is; a navigation given replaces the navigation whole (M3 design 5.2).
type PreferencesPatch struct {
	Navigation *Navigation
	SortOrder  *float64
}

// DefaultPreferences are an account's settings in a project while he has
// no row of them: the columns' defaults (M3 design 4.8), which the store's
// test holds equal.
func DefaultPreferences() Preferences {
	return Preferences{Navigation: Navigation{DefaultTab: tabs[0], HideInMoreMenu: []string{}}, SortOrder: DefaultSortOrder}
}

// Apply returns p with the fields patch gives changed.
func (p Preferences) Apply(patch PreferencesPatch) Preferences {
	if patch.Navigation != nil {
		p.Navigation = *patch.Navigation
	}
	if patch.SortOrder != nil {
		p.SortOrder = *patch.SortOrder
	}
	return p
}

// CheckPreferencesPatch checks the navigation p gives: its default tab one
// of the tabs; each tab it hides one of the tabs but the work items, and
// hidden once. Every problem is reported at once, as one 422
// validation_failed; a tab the web app does not have is refused here, not
// stored (M3 design 12, P4b).
func CheckPreferencesPatch(p PreferencesPatch) error {
	if p.Navigation == nil {
		return nil
	}
	var found []*shared.FieldError
	if !slices.Contains(tabs, p.Navigation.DefaultTab) {
		found = append(found, &shared.FieldError{Field: "navigation.default_tab", Code: shared.FieldInvalidFormat,
			Message: "is not one of " + strings.Join(tabs, ", ")})
	}
	hideable := tabs[1:]
	for i, tab := range p.Navigation.HideInMoreMenu {
		field := fmt.Sprintf("navigation.hide_in_more_menu[%d]", i)
		switch {
		case !slices.Contains(hideable, tab):
			found = append(found, &shared.FieldError{Field: field, Code: shared.FieldInvalidFormat, Message: "is not one of " + strings.Join(hideable, ", ")})
		case slices.Contains(p.Navigation.HideInMoreMenu[:i], tab):
			found = append(found, &shared.FieldError{Field: field, Code: shared.FieldDuplicate, Message: "is listed before"})
		}
	}
	return invalid(found...)
}

````

`server/internal/modules/project/domain/preferences_test.go`（修改，2 处）：

````old server/internal/modules/project/domain/preferences_test.go
import "testing"
````
````new server/internal/modules/project/domain/preferences_test.go
import (
	"errors"
	"reflect"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)
````

````old server/internal/modules/project/domain/preferences_test.go
	}
}

````
````new server/internal/modules/project/domain/preferences_test.go
	}
}

// The defaults are the work items' tab, nothing hidden, the default place;
// a patch changes what it gives, the navigation whole, and nothing else.
func TestPreferencesApply(t *testing.T) {
	d := DefaultPreferences()
	if want := (Preferences{Navigation: Navigation{DefaultTab: "work_items", HideInMoreMenu: []string{}}, SortOrder: 65535}); !reflect.DeepEqual(d, want) {
		t.Errorf("DefaultPreferences() = %+v, want %+v", d, want)
	}
	tabbed := Navigation{DefaultTab: "modules", HideInMoreMenu: []string{"views"}}
	for _, tt := range []struct {
		patch PreferencesPatch
		want  Preferences
	}{
		{PreferencesPatch{}, d},
		{PreferencesPatch{Navigation: &tabbed}, Preferences{Navigation: tabbed, SortOrder: 65535}},
		{PreferencesPatch{SortOrder: ptr(-1.5)}, Preferences{Navigation: d.Navigation, SortOrder: -1.5}},
	} {
		if got := d.Apply(tt.patch); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Apply(%+v) = %+v, want %+v", tt.patch, got, tt.want)
		}
	}
}

// Every tab is a default tab, and every one but the work items can be
// hidden, each once, in any order; no navigation is checked when none is
// given, whatever the place.
func TestCheckPreferencesPatchAcceptsEveryTab(t *testing.T) {
	for _, p := range []PreferencesPatch{
		{},
		{SortOrder: ptr(-1e9)},
		{Navigation: &Navigation{DefaultTab: "work_items", HideInMoreMenu: []string{}}},
		{Navigation: &Navigation{DefaultTab: "cycles", HideInMoreMenu: []string{"intake", "views", "modules", "cycles"}}},
		{Navigation: &Navigation{DefaultTab: "modules"}},
		{Navigation: &Navigation{DefaultTab: "views"}},
		{Navigation: &Navigation{DefaultTab: "intake"}},
	} {
		if err := CheckPreferencesPatch(p); err != nil {
			t.Errorf("CheckPreferencesPatch(%+v) = %v, want nil", p, err)
		}
	}
}

// An unknown tab as the default, or hidden, the work items hidden, a tab
// hidden twice: each refused, all of them in one 422 that names each by its
// place in the request.
func TestCheckPreferencesPatchReportsEveryTab(t *testing.T) {
	err := CheckPreferencesPatch(PreferencesPatch{Navigation: &Navigation{DefaultTab: "pages",
		HideInMoreMenu: []string{"views", "work_items", "pages", "views", "Views"}}})
	tabs, hideable := "is not one of work_items, cycles, modules, views, intake", "is not one of cycles, modules, views, intake"
	want := []shared.FieldError{
		{Field: "navigation.default_tab", Code: "invalid_format", Message: tabs},
		{Field: "navigation.hide_in_more_menu[1]", Code: "invalid_format", Message: hideable},
		{Field: "navigation.hide_in_more_menu[2]", Code: "invalid_format", Message: hideable},
		{Field: "navigation.hide_in_more_menu[3]", Code: "duplicate", Message: "is listed before"},
		{Field: "navigation.hide_in_more_menu[4]", Code: "invalid_format", Message: hideable},
	}
	var e *shared.Error
	if !errors.As(err, &e) || e.Code != "validation_failed" || !reflect.DeepEqual(e.Fields, want) {
		t.Errorf("CheckPreferencesPatch() = %v; want validation_failed with\n%+v", err, want)
	}
}

````

- [ ] **Step 2: 查询**

`server/internal/modules/project/adapter/postgres/queries/preferences.sql`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/queries/preferences.sql
LIMIT 1;

````
````new server/internal/modules/project/adapter/postgres/queries/preferences.sql
LIMIT 1;

-- name: Preferences :one
-- getProjectPreferences: the account's undeleted display settings in the project, if any (M3 design 3.18).
SELECT preferences, sort_order
FROM project_user_properties
WHERE project_id = sqlc.arg(project_id) AND user_id = sqlc.arg(user_id) AND deleted_at IS NULL;

-- name: UpsertPreferences :one
-- updateProjectPreferences, under the project's FOR SHARE (M3 design 3.6, 3.18): an account without an undeleted row
-- gets one, with the values given, the defaults with the change applied; one with a row has the fields that are set
-- changed, the navigation whole. The conflict target is the partial unique index, so a deleted row does not count.
INSERT INTO project_user_properties AS p (id, workspace_id, project_id, user_id, preferences, sort_order, created_by_id,
                                           updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(user_id), sqlc.arg(preferences), sqlc.arg(sort_order),
        sqlc.arg(user_id), sqlc.arg(user_id), sqlc.arg(now), sqlc.arg(now))
ON CONFLICT (project_id, user_id) WHERE deleted_at IS NULL DO UPDATE
SET preferences   = CASE WHEN sqlc.arg(set_navigation)::boolean THEN EXCLUDED.preferences ELSE p.preferences END,
    sort_order    = CASE WHEN sqlc.arg(set_sort_order)::boolean THEN EXCLUDED.sort_order ELSE p.sort_order END,
    updated_by_id = EXCLUDED.updated_by_id,
    updated_at    = EXCLUDED.updated_at
RETURNING preferences, sort_order;

````

`server/internal/modules/project/adapter/postgres/queries/projects.sql`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/queries/projects.sql
WHERE id = sqlc.arg(id);

````
````new server/internal/modules/project/adapter/postgres/queries/projects.sql
WHERE id = sqlc.arg(id);

-- name: ShareProject :one
-- The parent lock of a write under the project that leaves the project row and its memberships as they are (M3 design
-- 3.6): FOR SHARE waits for FOR NO KEY UPDATE, not for another FOR SHARE. After a wait, Postgres evaluates deleted_at
-- IS NULL again on the row's newest version, so a project deleted meanwhile reads no row.
SELECT workspace_id, (archived_at IS NOT NULL)::boolean AS archived
FROM projects
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
FOR SHARE;

````

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `71e551a9b14f2ec62f3d4ca19a786a749899da0cc23a955062858bc0271ac0bd` | 140 | `server/internal/modules/project/adapter/postgres/gen/preferences.sql.go` |
| `aa8b87285bcc5a4e9ebf09b16b79741cb17662a22ee81e19478ab7f93f50cb60` | 412 | `server/internal/modules/project/adapter/postgres/gen/projects.sql.go` |

Run: `shasum -a 256 server/internal/modules/project/adapter/postgres/gen/preferences.sql.go server/internal/modules/project/adapter/postgres/gen/projects.sql.go`
Expected: 与上表相同。

- [ ] **Step 3: 端口和存储**

`server/internal/modules/project/app/ports.go`（修改，2 处）：

````old server/internal/modules/project/app/ports.go
}

// ProjectFinder finds the workspace of a project: what a read decides on,
````
````new server/internal/modules/project/app/ports.go
}

// ProjectSharer takes the parent lock of a write under a project that
// leaves the project row and its memberships as they are (M3 design 3.6).
type ProjectSharer interface {
	// ShareProject locks the undeleted project id FOR SHARE until the
	// transaction ctx carries ends; found is false when there is none, a
	// project deleted while the lock waited too.
	ShareProject(ctx context.Context, id uuid.UUID) (p LockedProject, found bool, err error)
}

// ProjectFinder finds the workspace of a project: what a read decides on,
````

````old server/internal/modules/project/app/ports.go
	ProjectLocker
````
````new server/internal/modules/project/app/ports.go
	ProjectLocker
	ProjectSharer
}

// PreferencesReader is getProjectPreferences' repository.
type PreferencesReader interface {
	ProjectFinder
	// Preferences are userID's display settings in projectID; found is false
	// while he has no undeleted row of them.
	Preferences(ctx context.Context, projectID, userID uuid.UUID) (p domain.Preferences, found bool, err error)
}

// PreferencesChange is a change of an account's display settings in a
// project, and the id of the row if the change inserts one.
type PreferencesChange struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	UserID      uuid.UUID
	Patch       domain.PreferencesPatch
	Now         time.Time
}

// PreferencesWriter is updateProjectPreferences' repository. It runs in the
// transaction ctx carries.
type PreferencesWriter interface {
	// UpsertPreferences applies c.Patch to the account's undeleted row, or
	// inserts one with domain.DefaultPreferences and the patch applied, by
	// the account at c.Now, and returns the settings as stored.
	UpsertPreferences(ctx context.Context, c PreferencesChange) (domain.Preferences, error)
````

`server/internal/modules/project/adapter/postgres/update.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/update.go
		return app.LockedProject{}, false, fmt.Errorf("lock project %s: %w", id, err)
````
````new server/internal/modules/project/adapter/postgres/update.go
		return app.LockedProject{}, false, fmt.Errorf("lock project %s: %w", id, err)
	}
	return app.LockedProject{WorkspaceID: r.WorkspaceID, Archived: r.Archived}, true, nil
}

// ShareProject locks the undeleted project id FOR SHARE until the
// transaction ctx carries ends, and reads its workspace and whether it is
// archived; found is false when there is none, a project deleted while the
// lock waited too (app.ProjectSharer).
func (s *Store) ShareProject(ctx context.Context, id uuid.UUID) (p app.LockedProject, found bool, err error) {
	r, err := s.queries(ctx).ShareProject(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return app.LockedProject{}, false, nil
	case err != nil:
		return app.LockedProject{}, false, fmt.Errorf("share project %s: %w", id, err)
````

`server/internal/modules/project/adapter/postgres/preferences.go`（新文件，78 行）：

````file server/internal/modules/project/adapter/postgres/preferences.go
package postgresadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// navigationJSON is project_user_properties.preferences as stored: one key,
// navigation, with exactly its two (M3 design 4.8).
type navigationJSON struct {
	Navigation struct {
		DefaultTab     string   `json:"default_tab"`
		HideInMoreMenu []string `json:"hide_in_more_menu"`
	} `json:"navigation"`
}

// Preferences returns userID's display settings in projectID; found is
// false while he has no undeleted row of them (app.PreferencesReader).
func (s *Store) Preferences(ctx context.Context, projectID, userID uuid.UUID) (domain.Preferences, bool, error) {
	r, err := s.queries(ctx).Preferences(ctx, gen.PreferencesParams{ProjectID: projectID, UserID: userID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Preferences{}, false, nil
	case err != nil:
		return domain.Preferences{}, false, fmt.Errorf("read project preferences: %w", err)
	}
	p, err := preferences(r.Preferences, r.SortOrder)
	if err != nil {
		return domain.Preferences{}, false, err
	}
	return p, true, nil
}

// UpsertPreferences applies c.Patch to the account's undeleted row, or
// inserts one with domain.DefaultPreferences and the patch applied, by the
// account at c.Now, and returns the settings as stored
// (app.PreferencesWriter).
func (s *Store) UpsertPreferences(ctx context.Context, c app.PreferencesChange) (domain.Preferences, error) {
	inserted := domain.DefaultPreferences().Apply(c.Patch)
	var n navigationJSON
	n.Navigation.DefaultTab, n.Navigation.HideInMoreMenu = inserted.Navigation.DefaultTab, inserted.Navigation.HideInMoreMenu
	if n.Navigation.HideInMoreMenu == nil {
		n.Navigation.HideInMoreMenu = []string{}
	}
	stored, err := json.Marshal(n)
	if err != nil {
		return domain.Preferences{}, fmt.Errorf("write project preferences: %w", err)
	}
	r, err := s.queries(ctx).UpsertPreferences(ctx, gen.UpsertPreferencesParams{
		ID: c.ID, WorkspaceID: c.WorkspaceID, ProjectID: c.ProjectID, UserID: c.UserID, Preferences: stored, SortOrder: inserted.SortOrder,
		SetNavigation: c.Patch.Navigation != nil, SetSortOrder: c.Patch.SortOrder != nil, Now: c.Now,
	})
	if err != nil {
		return domain.Preferences{}, fmt.Errorf("write project preferences: %w", err)
	}
	return preferences(r.Preferences, r.SortOrder)
}

// preferences are the settings a row holds.
func preferences(stored []byte, sortOrder float64) (domain.Preferences, error) {
	var n navigationJSON
	if err := json.Unmarshal(stored, &n); err != nil {
		return domain.Preferences{}, fmt.Errorf("read project preferences %s: %w", stored, err)
	}
	return domain.Preferences{
		Navigation: domain.Navigation{DefaultTab: n.Navigation.DefaultTab, HideInMoreMenu: n.Navigation.HideInMoreMenu},
		SortOrder:  sortOrder,
	}, nil
}
````

`server/internal/modules/project/adapter/postgres/preferences_test.go`（新文件，162 行）：

````file server/internal/modules/project/adapter/postgres/preferences_test.go
package postgresadapter_test

import (
	"context"
	"maps"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// ShareProject reads the undeleted project's workspace and whether it is
// archived, and holds it FOR SHARE until the transaction ends: a FOR NO KEY
// UPDATE of it waits, another FOR SHARE does not, and no other project is
// held. A deleted project, and no project, are not found.
func TestShareProject(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web, ops, old := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, beta, "Ops", "OPS", alice), newProject(t, s, acme, "Old", "OLD", alice)
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", ops, now)
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", old, now)
	for _, tt := range []struct {
		id    uuid.UUID
		want  app.LockedProject
		found bool
	}{{web, app.LockedProject{WorkspaceID: acme}, true}, {ops, app.LockedProject{WorkspaceID: beta, Archived: true}, true},
		{old, app.LockedProject{}, false}, {uuid.NewV7(), app.LockedProject{}, false}} {
		if got, found, err := s.ShareProject(context.Background(), tt.id); err != nil || found != tt.found || got != tt.want {
			t.Errorf("ShareProject(%s) = %+v, %v, %v; want %+v, %v", tt.id, got, found, err, tt.want, tt.found)
		}
	}
	err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		if _, _, err := s.ShareProject(ctx, web); err != nil {
			return err
		}
		if !waits(t, pool, web, "FOR NO KEY UPDATE") || waits(t, pool, web, "FOR SHARE") || waits(t, pool, ops, "FOR NO KEY UPDATE") {
			t.Error("while the lock is held: want a FOR NO KEY UPDATE of web to wait, and neither its FOR SHARE nor ops' FOR NO KEY UPDATE")
		}
		return nil
	})
	if err != nil || waits(t, pool, web, "FOR NO KEY UPDATE") {
		t.Errorf("after the transaction: %v; want web free", err)
	}
}

// preferencesRow inserts user's display settings in project with no value
// but the keys, so that every other column takes its default, and returns
// the row's id.
func preferencesRow(t *testing.T, pool *pgxpool.Pool, workspace, project, user uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	exec(t, pool, `INSERT INTO project_user_properties (id, workspace_id, project_id, user_id) VALUES ($1, $2, $3, $4)`, id, workspace, project, user)
	return id
}

// A row with every column at its default reads as domain.DefaultPreferences:
// the defaults the use case answers while there is no row are the columns'.
// Preferences reads the account's undeleted row in the project alone: none
// for another account, another project or a deleted row.
func TestPreferences(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	preferencesRow(t, pool, acme, web, bob)
	views := preferencesRow(t, pool, acme, ops, bob)
	exec(t, pool, `UPDATE project_user_properties SET preferences = '{"navigation": {"default_tab": "views", "hide_in_more_menu": ["cycles"]}}',
		sort_order = 7 WHERE id = $1`, views)
	exec(t, pool, "UPDATE project_user_properties SET deleted_at = $2 WHERE id = $1", preferencesRow(t, pool, acme, web, carol), now)
	for _, tt := range []struct {
		project, user uuid.UUID
		want          domain.Preferences
		found         bool
	}{
		{web, bob, domain.DefaultPreferences(), true},
		{ops, bob, domain.Preferences{Navigation: domain.Navigation{DefaultTab: "views", HideInMoreMenu: []string{"cycles"}}, SortOrder: 7}, true},
		{web, carol, domain.Preferences{}, false},
		{web, alice, domain.Preferences{}, false},
		{uuid.NewV7(), bob, domain.Preferences{}, false},
	} {
		if got, found, err := s.Preferences(context.Background(), tt.project, tt.user); err != nil || found != tt.found || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Preferences(%s, %s) = %+v, %v, %v; want %+v, %v", tt.project, tt.user, got, found, err, tt.want, tt.found)
		}
	}
}

// UpsertPreferences inserts the account's row while he has no undeleted
// one, the defaults with the change applied, made and last changed by him
// at the moment given; with a row, it changes the fields the change gives,
// the navigation whole, and the audit columns, to him and that moment even
// when another account made the row, and leaves the rest. The
// answer is the row as stored. Every other row keeps every column: another
// account's, another project's, his deleted one.
func TestUpsertPreferences(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	deleted := preferencesRow(t, pool, acme, web, bob)
	exec(t, pool, "UPDATE project_user_properties SET deleted_at = $2 WHERE id = $1", deleted, now)
	preferencesRow(t, pool, acme, ops, bob)
	others := tableRows(t, pool, "project_user_properties", uuid.Nil())
	upsert := func(p domain.PreferencesPatch, at time.Time) domain.Preferences {
		t.Helper()
		got, err := s.UpsertPreferences(context.Background(), app.PreferencesChange{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, UserID: bob,
			Patch: p, Now: at})
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	var id uuid.UUID // bob's row in web, once inserted
	row := func() map[string]string {
		if err := pool.QueryRow(context.Background(), "SELECT id FROM project_user_properties WHERE project_id = $1 AND user_id = $2 AND deleted_at IS NULL",
			web, bob).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return columns(t, pool, "project_user_properties", id)
	}
	stamp := func(at time.Time) string { return `"` + at.Format("2006-01-02T15:04:05.999999") + `+00:00"` }
	views := domain.Navigation{DefaultTab: "views", HideInMoreMenu: []string{"intake", "cycles"}}

	if got := upsert(domain.PreferencesPatch{SortOrder: ptr(-5.5)}, now); !reflect.DeepEqual(got,
		domain.Preferences{Navigation: domain.DefaultPreferences().Navigation, SortOrder: -5.5}) {
		t.Errorf("the first change = %+v; want the defaults, at -5.5", got)
	}
	first := row()
	for k, want := range map[string]string{"sort_order": "-5.5", "created_by_id": `"` + bob.String() + `"`, "updated_by_id": `"` + bob.String() + `"`,
		"created_at": stamp(now), "updated_at": stamp(now), "deleted_at": "null", "workspace_id": `"` + acme.String() + `"`,
		"preferences": `{"navigation": {"default_tab": "work_items", "hide_in_more_menu": []}}`} {
		if first[k] != want {
			t.Errorf("the inserted row's %s = %s, want %s", k, first[k], want)
		}
	}

	// A row another account made and changed last, as alice's adding him makes it: the next change is his.
	exec(t, pool, "UPDATE project_user_properties SET created_by_id = $2, updated_by_id = $2 WHERE id = $1", id, alice)
	first = row()
	later := now.Add(time.Hour)
	if got := upsert(domain.PreferencesPatch{Navigation: &views}, later); !reflect.DeepEqual(got, domain.Preferences{Navigation: views, SortOrder: -5.5}) {
		t.Errorf("the second change = %+v; want views, still at -5.5", got)
	}
	want := maps.Clone(first)
	maps.Copy(want, map[string]string{"preferences": `{"navigation": {"default_tab": "views", "hide_in_more_menu": ["intake", "cycles"]}}`,
		"updated_at": stamp(later), "updated_by_id": `"` + bob.String() + `"`})
	if got := row(); !maps.Equal(got, want) {
		t.Errorf("after the second change: %v\nwant %v", got, want)
	}
	if got := upsert(domain.PreferencesPatch{}, later.Add(time.Hour)); !reflect.DeepEqual(got, domain.Preferences{Navigation: views, SortOrder: -5.5}) {
		t.Errorf("an empty change = %+v; want the settings as they were", got)
	}
	if after := tableRows(t, pool, "project_user_properties", id); after != others {
		t.Errorf("the other rows:\n%s\nwant\n%s", after, others)
	}
}
````

`server/internal/modules/project/adapter/postgres/failures_test.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("SetArchived() = %v; want context.Canceled", err)
	}
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("SetArchived() = %v; want context.Canceled", err)
	}
	if p, found, err := s.ShareProject(cancelled, web); !failed(err) || found || p != (app.LockedProject{}) {
		t.Errorf("ShareProject() = %+v, %v, %v; want context.Canceled, not no project", p, found, err)
	}
	if p, found, err := s.Preferences(cancelled, web, alice); !failed(err) || found {
		t.Errorf("Preferences() = %+v, %v, %v; want context.Canceled, not none", p, found, err)
	}
	if _, err := s.UpsertPreferences(cancelled, app.PreferencesChange{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, UserID: alice,
		Now: now}); !failed(err) {
		t.Errorf("UpsertPreferences() = %v; want context.Canceled", err)
	}
````

- [ ] **Step 4: 操作名、规则、用例**

`server/internal/modules/project/domain/actions.go`（修改，2 处）：

````old server/internal/modules/project/domain/actions.go
	ActionDelete shared.Action = "project.delete"
````
````new server/internal/modules/project/domain/actions.go
	ActionDelete shared.Action = "project.delete"
	// ActionPreferencesRead is reading one's display settings in a project:
	// getProjectPreferences.
	ActionPreferencesRead shared.Action = "project_preferences.read"
	// ActionPreferencesUpdate is changing them: updateProjectPreferences.
	ActionPreferencesUpdate shared.Action = "project_preferences.update"
````

````old server/internal/modules/project/domain/actions.go
		ActionUnarchive, ActionDelete}
````
````new server/internal/modules/project/domain/actions.go
		ActionUnarchive, ActionDelete, ActionPreferencesRead, ActionPreferencesUpdate}
````

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"project.delete":    {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
````
````new server/internal/modules/access/domain/rules.go
	"project.delete":    {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	// Every active member of the project, his own settings (M3 design 9.2).
	"project_preferences.read":   {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	"project_preferences.update": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"project.delete": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
````
````new server/internal/modules/access/domain/rules_test.go
	"project.delete": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
	"project_preferences.read": {allowed, allowed, allowed, allowed, allowed, forbidden, forbidden, invisible, invisible, invisible,
		invisible, forbidden, invisible, invisible, invisible, forbidden, forbidden},
	"project_preferences.update": {allowed, allowed, allowed, allowed, allowed, forbidden, forbidden, invisible, invisible, invisible,
		invisible, forbidden, invisible, invisible, invisible, forbidden, forbidden},
````

`server/internal/modules/project/app/lock.go`（修改，4 处）：

````old server/internal/modules/project/app/lock.go
// workspace's row FOR SHARE, the write's first lock; the project's row FOR
// NO KEY UPDATE, still undeleted and of that workspace; then the decision,
// under them all. Every cascade over the workspace's projects runs under
// the workspace's FOR NO KEY UPDATE and reads its time after it (3.3):
// while a write holds the workspace FOR SHARE, no cascade touches the rows
// it writes. project.New builds one Locks for every write: a write holds
// no Authorizer of its own, so it decides only under these locks.
````
````new server/internal/modules/project/app/lock.go
// workspace's row FOR SHARE, the write's first lock; the project's row, FOR
// NO KEY UPDATE, or FOR SHARE for a write under the project that leaves
// the row and its memberships as they are, still undeleted and of that
// workspace; then the decision, under them all. Every cascade over the
// workspace's projects runs under the workspace's FOR NO KEY UPDATE and
// reads its time after it (3.3): while a write holds the workspace FOR
// SHARE, no cascade touches the rows it writes. project.New builds one
// Locks for every write: a write holds no Authorizer of its own, so it
// decides only under these locks.
````

````old server/internal/modules/project/app/lock.go
	action  shared.Action
````
````new server/internal/modules/project/app/lock.go
	action  shared.Action
	// share locks the project FOR SHARE, for a write under the project that
	// leaves the project row and its memberships as they are; otherwise it
	// is locked FOR NO KEY UPDATE.
	share bool
````

````old server/internal/modules/project/app/lock.go
	h.project, found, err = l.projects.LockProject(ctx, w.project)
````
````new server/internal/modules/project/app/lock.go
	lock := l.projects.LockProject
	if w.share {
		lock = l.projects.ShareProject
	}
	h.project, found, err = lock(ctx, w.project)
````

````old server/internal/modules/project/app/lock.go
	return h, nil
````
````new server/internal/modules/project/app/lock.go
	return h, nil
}

// findAndDecide is the first two steps of a read under a project named by
// its id (M3 design 6.4), without a transaction: projects finds the
// undeleted project's workspace, then decide decides action on it. A
// project that is not there, or not visible to actor, is domain.ErrNotFound;
// a role the rule does not allow is the Authorizer's shared.Forbidden.
func findAndDecide(ctx context.Context, projects ProjectFinder, auth shared.Authorizer, actor shared.Actor, id uuid.UUID,
	action shared.Action) error {
	workspaceID, found, err := projects.ProjectWorkspace(ctx, id)
	switch {
	case err != nil:
		return err
	case !found:
		return domain.ErrNotFound
	}
	_, err = decide(ctx, auth, actor, action, workspaceID, id)
	return err
````

`server/internal/modules/project/app/get_preferences.go`（新文件，44 行）：

````file server/internal/modules/project/app/get_preferences.go
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// GetProjectPreferences reads the caller's display settings in a project:
// GET /api/v0/me/projects/{project_id}/preferences.
type GetProjectPreferences struct {
	preferences PreferencesReader
	auth        shared.Authorizer
}

// NewGetProjectPreferences returns the use case.
func NewGetProjectPreferences(preferences PreferencesReader, auth shared.Authorizer) *GetProjectPreferences {
	return &GetProjectPreferences{preferences: preferences, auth: auth}
}

// Execute returns the caller's settings in the project, or
// domain.DefaultPreferences while he has no row of them, after the
// decision on project_preferences.read. A read opens no transaction and
// writes nothing (M3 design 3.18): a row missing is created by the first
// change.
func (u *GetProjectPreferences) Execute(ctx context.Context, projectID uuid.UUID) (domain.Preferences, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Preferences{}, err
	}
	if err := findAndDecide(ctx, u.preferences, u.auth, actor, projectID, domain.ActionPreferencesRead); err != nil {
		return domain.Preferences{}, err
	}
	p, found, err := u.preferences.Preferences(ctx, projectID, actor.UserID)
	switch {
	case err != nil:
		return domain.Preferences{}, err
	case !found:
		return domain.DefaultPreferences(), nil
	}
	return p, nil
}
````

`server/internal/modules/project/app/update_preferences.go`（新文件，54 行）：

````file server/internal/modules/project/app/update_preferences.go
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateProjectPreferences changes the caller's display settings in a
// project: PATCH /api/v0/me/projects/{project_id}/preferences.
type UpdateProjectPreferences struct {
	preferences PreferencesWriter
	locks       Locks
	tx          shared.TxManager
	clock       Clock
}

// NewUpdateProjectPreferences returns the use case.
func NewUpdateProjectPreferences(preferences PreferencesWriter, locks Locks, tx shared.TxManager, clock Clock) *UpdateProjectPreferences {
	return &UpdateProjectPreferences{preferences: preferences, locks: locks, tx: tx, clock: clock}
}

// Execute checks p, then in one transaction (M3 design 3.6): the project's
// locks (Locks: its workspace FOR SHARE, then the project FOR SHARE) and
// the decision on project_preferences.update, the clock read under the
// locks, the caller's row changed or inserted (3.18). An archived
// project's settings change as any other's (3.19). The answer is the
// settings as stored.
func (u *UpdateProjectPreferences) Execute(ctx context.Context, projectID uuid.UUID, p domain.PreferencesPatch) (domain.Preferences, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Preferences{}, err
	}
	if err := domain.CheckPreferencesPatch(p); err != nil {
		return domain.Preferences{}, err
	}
	var stored domain.Preferences
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		h, err := u.locks.lockAndDecide(ctx, actor, write{project: projectID, action: domain.ActionPreferencesUpdate, share: true})
		if err != nil {
			return err
		}
		stored, err = u.preferences.UpsertPreferences(ctx, PreferencesChange{
			ID: uuid.NewV7(), WorkspaceID: h.project.WorkspaceID, ProjectID: projectID, UserID: actor.UserID, Patch: p, Now: u.clock.Now(),
		})
		return err
	})
	if err != nil {
		return domain.Preferences{}, err
	}
	return stored, nil
}
````

`server/internal/modules/project/app/fakes_write_test.go`（修改，6 处）：

````old server/internal/modules/project/app/fakes_write_test.go
// is archived, its memberships by account, and when it was last written,
// as stored.
````
````new server/internal/modules/project/app/fakes_write_test.go
// is archived, its memberships and its members' display settings by
// account, and when it was last written, as stored.
````

````old server/internal/modules/project/app/fakes_write_test.go
	members   map[uuid.UUID]app.Membership
````
````new server/internal/modules/project/app/fakes_write_test.go
	members   map[uuid.UUID]app.Membership
	prefs     map[uuid.UUID]domain.Preferences
````

````old server/internal/modules/project/app/fakes_write_test.go
// ops, archived, whose admin is bob.
var webID, opsID = uuid.NewV7(), uuid.NewV7()
````
````new server/internal/modules/project/app/fakes_write_test.go
// ops, archived, whose admin is bob. Bob has display settings in web
// (bobsTabs), the others none.
var webID, opsID = uuid.NewV7(), uuid.NewV7()

var bobsTabs = domain.Preferences{Navigation: domain.Navigation{DefaultTab: "modules", HideInMoreMenu: []string{"views"}}, SortOrder: 10}
````

````old server/internal/modules/project/app/fakes_write_test.go
				dave:  {ID: uuid.NewV7(), Role: shared.RoleMember},
			}},
````
````new server/internal/modules/project/app/fakes_write_test.go
				dave:  {ID: uuid.NewV7(), Role: shared.RoleMember},
			}, prefs: map[uuid.UUID]domain.Preferences{bob: bobsTabs}},
````

````old server/internal/modules/project/app/fakes_write_test.go
func (f *fakeStore) LockProject(ctx context.Context, id uuid.UUID) (app.LockedProject, bool, error) {
	f.log.add(ctx, "LockProject %s", id)
	if err := f.fail("LockProject"); err != nil {
````
````new server/internal/modules/project/app/fakes_write_test.go
// lock is LockProject and ShareProject, named by method.
func (f *fakeStore) lock(ctx context.Context, method string, id uuid.UUID) (app.LockedProject, bool, error) {
	f.log.add(ctx, "%s %s", method, id)
	if err := f.fail(method); err != nil {
````

````old server/internal/modules/project/app/fakes_write_test.go
	return app.LockedProject{WorkspaceID: p.workspace, Archived: p.archived}, true, nil
````
````new server/internal/modules/project/app/fakes_write_test.go
	return app.LockedProject{WorkspaceID: p.workspace, Archived: p.archived}, true, nil
}

func (f *fakeStore) LockProject(ctx context.Context, id uuid.UUID) (app.LockedProject, bool, error) {
	return f.lock(ctx, "LockProject", id)
}

func (f *fakeStore) ShareProject(ctx context.Context, id uuid.UUID) (app.LockedProject, bool, error) {
	return f.lock(ctx, "ShareProject", id)
````

`server/internal/modules/project/app/fakes_preferences_test.go`（新文件，41 行）：

````file server/internal/modules/project/app/fakes_preferences_test.go
package app_test

import (
	"context"
	"encoding/json"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// fakeStore's display settings: Preferences reads a member's from his
// project, UpsertPreferences applies the patch to them, or to the defaults
// while he has none, and answers them as stored.

func (f *fakeStore) Preferences(ctx context.Context, projectID, userID uuid.UUID) (domain.Preferences, bool, error) {
	f.log.add(ctx, "Preferences %s for %s", projectID, userID)
	if err := f.fail("Preferences"); err != nil {
		return domain.Preferences{}, false, err
	}
	p, ok := f.projects[projectID].prefs[userID]
	return p, ok, nil
}

func (f *fakeStore) UpsertPreferences(ctx context.Context, c app.PreferencesChange) (domain.Preferences, error) {
	patch, _ := json.Marshal(c.Patch)
	f.log.add(ctx, "UpsertPreferences %s/%s for %s %s at %s", c.WorkspaceID, c.ProjectID, c.UserID, patch, c.Now.Format(timeFormat))
	if err := f.fail("UpsertPreferences"); err != nil {
		return domain.Preferences{}, err
	}
	project := f.projects[c.ProjectID]
	p, ok := project.prefs[c.UserID]
	if !ok {
		p = domain.DefaultPreferences()
	}
	if project.prefs == nil {
		project.prefs = map[uuid.UUID]domain.Preferences{}
	}
	project.prefs[c.UserID] = p.Apply(c.Patch)
	return project.prefs[c.UserID], nil
}
````

`server/internal/modules/project/app/preferences_test.go`（新文件，189 行）：

````file server/internal/modules/project/app/preferences_test.go
package app_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newPreferences is newWrites with carol's grant in acme too, a project
// guest's, so that a member without display settings reads and writes
// hers.
func newPreferences() *writeFixture {
	f := newWrites()
	f.auth.grants[grantKey{carol, acme.ID}] = shared.Grant{WorkspaceRole: shared.RoleGuest, ProjectRole: shared.RoleGuest}
	return f
}

// refusedAs reports whether err is want: errDisk itself, a port's failure
// come back, or the same refusal (sameError).
func refusedAs(err, want error) bool {
	if want == errDisk {
		return errors.Is(err, errDisk)
	}
	return sameError(err, want)
}

// read are the calls of user's read of his display settings in project: the
// project's workspace, the decision, his settings; none in a transaction.
func read(user, project uuid.UUID) []string {
	return []string{"ProjectWorkspace " + project.String() + " outside tx",
		fmt.Sprintf("Authorize %s %s on %s/%s outside tx", user, domain.ActionPreferencesRead, acme.ID, project),
		fmt.Sprintf("Preferences %s for %s outside tx", project, user)}
}

// GetProjectPreferences answers the caller's display settings in the
// project after the decision on project_preferences.read, or the defaults
// while he has none; it opens no transaction and writes nothing.
func TestGetProjectPreferences(t *testing.T) {
	for _, tt := range []struct {
		user uuid.UUID
		want domain.Preferences
	}{{bob, bobsTabs}, {carol, domain.DefaultPreferences()}} {
		f := newPreferences()
		got, err := app.NewGetProjectPreferences(f.store, f.auth).Execute(as(tt.user), webID)
		if err != nil || !reflect.DeepEqual(got, tt.want) || !slices.Equal(f.log.calls, read(tt.user, webID)) {
			t.Errorf("Execute() for %s = %+v, %v, calls %q; want %+v, calls %q", tt.user, got, err, f.log.calls, tt.want, read(tt.user, webID))
		}
	}
}

// Refusals, each in its place: no caller; a project not there, or not
// visible: 404; a workspace admin who is not its member: the Authorizer's
// 403. Every port's failure comes back as itself, after the calls before it
// and none after.
func TestGetProjectPreferencesRefuses(t *testing.T) {
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
		{"not seen", as(erin), webID, nil, domain.ErrNotFound, read(erin, webID)[:2]},
		{"forbidden", as(alice), webID, nil, shared.Forbidden(), read(alice, webID)[:2]},
		{"the project failing", as(bob), webID, func(f *writeFixture) { f.store.errs = map[string]error{"ProjectWorkspace": errDisk} }, errDisk,
			read(bob, webID)[:1]},
		{"the decision failing", as(bob), webID, func(f *writeFixture) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, errDisk,
			read(bob, webID)[:2]},
		{"the settings failing", as(bob), webID, func(f *writeFixture) { f.store.errs = map[string]error{"Preferences": errDisk} }, errDisk,
			read(bob, webID)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newPreferences()
			if tt.fail != nil {
				tt.fail(f)
			}
			got, err := app.NewGetProjectPreferences(f.store, f.auth).Execute(tt.ctx, tt.id)
			if !refusedAs(err, tt.want) || !reflect.DeepEqual(got, domain.Preferences{}) || !slices.Equal(f.log.calls, tt.calls) {
				t.Errorf("Execute() = %+v, %v, calls %q; want %v, calls %q", got, err, f.log.calls, tt.want, tt.calls)
			}
		})
	}
}

// newUpdatePreferences is UpdateProjectPreferences over newPreferences'
// fakes, its clock logged.
func newUpdatePreferences() (*app.UpdateProjectPreferences, *writeFixture) {
	f := newPreferences()
	return app.NewUpdateProjectPreferences(f.store, f.locks(), f.tx, clockAt{clockNow, f.log}), f
}

// changed are the calls of user's change p of his display settings in
// project: the transaction, the project's workspace, its lock, the project
// FOR SHARE, the decision, the clock, the change.
func changed(user, project uuid.UUID, p domain.PreferencesPatch) []string {
	patch, _ := json.Marshal(p)
	return append(lockedTo(project), "ShareProject "+project.String(),
		fmt.Sprintf("Authorize %s %s on %s/%s", user, domain.ActionPreferencesUpdate, acme.ID, project), "Now",
		fmt.Sprintf("UpsertPreferences %s/%s for %s %s at %s", acme.ID, project, user, patch, clockNow.Format(timeFormat)))
}

// UpdateProjectPreferences checks the change, then, in one transaction,
// takes the project's locks, the project FOR SHARE, decides, reads the clock and changes the
// caller's settings, or makes them from the defaults while he has none; an
// archived project's change too. The answer is the settings as stored.
func TestUpdateProjectPreferences(t *testing.T) {
	cycles := domain.Navigation{DefaultTab: "cycles", HideInMoreMenu: []string{"intake"}}
	tests := []struct {
		name    string
		user    uuid.UUID
		project uuid.UUID
		in      domain.PreferencesPatch
		want    domain.Preferences
	}{
		{"bob's tabs", bob, webID, domain.PreferencesPatch{Navigation: &cycles}, domain.Preferences{Navigation: cycles, SortOrder: 10}},
		{"carol's place, from the defaults", carol, webID, domain.PreferencesPatch{SortOrder: ptr(-2.5)},
			domain.Preferences{Navigation: domain.DefaultPreferences().Navigation, SortOrder: -2.5}},
		{"in the archived project", bob, opsID, domain.PreferencesPatch{SortOrder: ptr(3.0)},
			domain.Preferences{Navigation: domain.DefaultPreferences().Navigation, SortOrder: 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newUpdatePreferences()
			got, err := uc.Execute(as(tt.user), tt.project, tt.in)
			if want := changed(tt.user, tt.project, tt.in); err != nil || !reflect.DeepEqual(got, tt.want) || !slices.Equal(f.log.calls, want) {
				t.Errorf("Execute() = %+v, %v, calls\n%q\nwant %+v, calls\n%q", got, err, f.log.calls, tt.want, want)
			}
		})
	}
}

// Refusals, each in its place, and nothing changed: an unknown tab, and no
// caller, before the transaction; a project not there, or not visible: 404;
// a workspace admin who is not its member: 403. Every port's failure comes
// back as itself, the commit's too, after the calls before it and none
// after.
func TestUpdateProjectPreferencesRefuses(t *testing.T) {
	in := domain.PreferencesPatch{SortOrder: ptr(1.0)}
	all := changed(bob, webID, in)
	tests := []struct {
		name  string
		ctx   context.Context
		id    uuid.UUID
		in    domain.PreferencesPatch
		fail  func(f *writeFixture)
		want  error
		calls []string
	}{
		{"an unknown tab", as(bob), webID, domain.PreferencesPatch{Navigation: &domain.Navigation{DefaultTab: "pages"}}, nil,
			shared.Invalid(shared.FieldError{Field: "navigation.default_tab", Code: "invalid_format"}), nil},
		{"no caller", context.Background(), webID, in, nil, shared.Unauthenticated(), nil},
		{"no project", as(bob), uuid.Nil(), in, nil, domain.ErrNotFound, noProject},
		{"not seen", as(erin), webID, in, nil, domain.ErrNotFound, changed(erin, webID, in)[:5]},
		{"forbidden", as(alice), webID, in, nil, shared.Forbidden(), changed(alice, webID, in)[:5]},
		{"the lock failing", as(bob), webID, in, func(f *writeFixture) { f.store.errs = map[string]error{"ShareProject": errDisk} }, errDisk, all[:4]},
		{"the decision failing", as(bob), webID, in, func(f *writeFixture) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, errDisk, all[:5]},
		{"the change failing", as(bob), webID, in, func(f *writeFixture) { f.store.errs = map[string]error{"UpsertPreferences": errDisk} }, errDisk,
			all},
		{"the commit failing", as(bob), webID, in, func(f *writeFixture) { f.tx.commitErr = errDisk }, errDisk, all},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newUpdatePreferences()
			if tt.fail != nil {
				tt.fail(f)
			}
			got, err := uc.Execute(tt.ctx, tt.id, tt.in)
			if !refusedAs(err, tt.want) || !reflect.DeepEqual(got, domain.Preferences{}) || !slices.Equal(f.log.calls, tt.calls) {
				t.Errorf("Execute() = %+v, %v, calls %q; want %v, calls %q", got, err, f.log.calls, tt.want, tt.calls)
			}
			if tt.fail == nil && !reflect.DeepEqual(f.store.projects[webID].prefs, map[uuid.UUID]domain.Preferences{bob: bobsTabs}) {
				t.Errorf("the settings after the refusal: %+v, want bob's alone, as they were", f.store.projects[webID].prefs)
			}
		})
	}
}
````

`server/internal/modules/project/app/clock_test.go`（修改，1 处）：

````old server/internal/modules/project/app/clock_test.go
		}, deleted(webID)},
````
````new server/internal/modules/project/app/clock_test.go
		}, deleted(webID)},
		{"updateProjectPreferences", func() ([]string, error) {
			uc, f := newUpdatePreferences()
			_, err := uc.Execute(as(bob), webID, domain.PreferencesPatch{SortOrder: ptr(1.0)})
			return f.log.calls, err
		}, changed(bob, webID, domain.PreferencesPatch{SortOrder: ptr(1.0)})},
````

- [ ] **Step 5: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/project/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestEveryActionHasARuleAndEveryRuleAnAction|TestEveryModuleDeclaresItsActionsOrHasNone' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 6: 提交**

```bash
git add server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/project/adapter/postgres/failures_test.go server/internal/modules/project/adapter/postgres/preferences.go server/internal/modules/project/adapter/postgres/preferences_test.go server/internal/modules/project/adapter/postgres/queries/preferences.sql server/internal/modules/project/adapter/postgres/queries/projects.sql server/internal/modules/project/adapter/postgres/update.go server/internal/modules/project/app/clock_test.go server/internal/modules/project/app/fakes_preferences_test.go server/internal/modules/project/app/fakes_write_test.go server/internal/modules/project/app/get_preferences.go server/internal/modules/project/app/lock.go server/internal/modules/project/app/ports.go server/internal/modules/project/app/preferences_test.go server/internal/modules/project/app/update_preferences.go server/internal/modules/project/domain/actions.go server/internal/modules/project/domain/preferences.go server/internal/modules/project/domain/preferences_test.go server/internal/modules/project/adapter/postgres/gen/preferences.sql.go server/internal/modules/project/adapter/postgres/gen/projects.sql.go
```
```bash
git commit -m "feat(M3/P4b): a member's display settings in a project: domain, store, use cases

The tab bar's tabs are the web app's five, the work items never hidden;
a patch replaces the navigation whole and is checked before the
transaction. Reading the settings decides on the project without a
transaction and answers the columns' defaults while the member has none;
changing them takes Locks' path with the project FOR SHARE (the
workspace FOR SHARE first), decides, and inserts or updates the member's
own row at the clock read under the locks.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| `ShareProject` 去掉项目的 id（两个行序） | `TestShareProject`；Task 15 起单独运行的故事：两个行序都是 P8 | 存储；端到端 |
| `ShareProject` 去掉 `deleted_at IS NULL` | `TestShareProject`；与 `ProjectWorkspace` 的 `deleted_at`、`ProjectFacts` 的 `p.deleted_at` 一起去掉时 P4（Task 15 起；方案 E 之下写先经 `ProjectWorkspace`，spec 第 3 节第 6 条） | 存储；端到端 |
| `ShareProject` 取 `FOR KEY SHARE`、不加锁 | `TestShareProject`；`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 8 起：等 Ops 的改设置不在 `projects` 上等） | 存储；组合 |
| `ShareProject` 不读已归档的项目 | `TestShareProject`；`TestPermissionMatrix`（Task 8 起） | 存储；组合 |
| `Preferences` 读别的项目的、别的账户的（两个行序） | `TestPreferences`；P8（Task 15 起） | 存储；端到端 |
| `Preferences` 读已删除的（两个行序） | `TestPreferences`（故事看不到：spec 第 3 节第 6 条） | 存储 |
| `UpsertPreferences` 的冲突目标没有 `WHERE deleted_at IS NULL`；不看标志就写导航、位置；修改时写 `created_at`；不改 `updated_by_id` | `TestUpsertPreferences`；第一个另有 P8（Task 15 起） | 存储；端到端 |
| 未知的默认标签页、藏起 `work_items` 被接受 | `TestCheckPreferencesPatchReportsEveryTab`；P8（Task 15 起） | 单元；端到端 |
| 两条规则给看得到项目的每个人；不给项目的访客 | `TestPermissionMatrix`（Task 8 起） | 组合 |
| 读的是别人（谁都不是）的设置；改的是别人的设置 | `TestEachMemberHasHisOwnDisplaySettings`（Task 8 起）；前者另有 P8 | 组合；端到端 |
| 没有调用者时也读（两个用例） | `TestGetProjectPreferencesRefuses`、`TestUpdateProjectPreferencesRefuses` | 单元 |
| `findAndDecide` 吞掉工作区的失败；读的判定、读的失败被吞掉、读的失败答成默认值；改的锁和判定（`Locks`）、写、提交的失败被吞掉 | `TestGetProjectPreferencesRefuses`、`TestUpdateProjectPreferencesRefuses`（`findAndDecide` 另有 `TestListProjectMembersRefuses`，Task 9 起） | 单元 |
| `ShareProject`、`UpsertPreferences` 在事务之外执行 | `TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 14 起） | 组合 |

**Done when:** 领域、存储、用例的测试通过；读不写、不开事务；改经 `Locks` 先取工作区、再取项目的 `FOR SHARE`，锁之后读时钟。

---

### Task 8: 显示设置的接口和组合

**Files:**
- Create: `server/internal/bootstrap/project_preferences_test.go`、`server/internal/modules/project/adapter/http/preferences.go`、`server/internal/modules/project/adapter/http/preferences_test.go`
- Modify: `api/modules/project.yaml`、`api/openapi.yaml`、`server/internal/bootstrap/permission_matrix_project_test.go`、`server/internal/bootstrap/project_write_locks_test.go`、`server/internal/bootstrap/project_writes_test.go`、`server/internal/modules/project/adapter/http/handler.go`、`server/internal/modules/project/adapter/http/handler_test.go`、`server/internal/modules/project/module.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/project/adapter/http/gen/bodyshape.gen.go`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.7，M3 设计 5.2）：`GET`、`PATCH /api/v0/me/projects/{project_id}/preferences`，200 `ProjectPreferences{navigation, sort_order}`；码 `[project.not_found, forbidden]`、`[validation_failed, project.not_found, forbidden]`；`ProjectTab` 枚举五个标签页、`ProjectNavigation` 两个字段都必有、`ProjectPreferencesUpdate` 两个字段都可省。契约的枚举不由生成的代码在请求时核对：未知的标签页照样传到用例，由领域答 422（M3 设计 12 节"未知的标签页 422"）；结构不对（缺字段、`null`、别的类型、多出的字段）答 400。
- 使用者：Task 15 的 P8。

**Tests:**
- `adapter/http/preferences_test.go`：`TestGetProjectPreferences`（设置照写，藏起的为空时是 `[]`）；`TestUpdateProjectPreferencesPassesTheChange`（导航整个、未知的标签页也传、没给的 `nil`）；`TestUpdateProjectPreferencesHoldsTheBodyToItsStructure`（九种结构错误 400）；`TestProjectPreferencesRefusals`（两个动作的 404、403，未知标签页的 422）。
- `bootstrap/project_preferences_test.go`：`TestEachMemberHasHisOwnDisplaySettings`（alice 改她在 Web 的设置，bob 在 Web 的（3）和她在 Ops 的照旧；每人读回自己的）。
- 矩阵三行（读、改各 12 格 `ofProject(200, 200, 200, 200, 403, 403)`，改的已归档 1 格 200；答案核对标签页和位置）；`TestTheWritesOnAProjectStampTheirRequest` 加改设置一行；`TestEachWriteOnAProjectSharesItsWorkspaceFirst` 加改设置一行（在删除之前：工作区 `FOR SHARE` 在先，项目的 `FOR SHARE` 也让第二段的写在 `projects` 上等）。

- [ ] **Step 1: 接口描述**

`api/modules/project.yaml`（修改，2 处）：

````old api/modules/project.yaml
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Project'
        default:
          $ref: '#/components/responses/Problem'
components:
````
````new api/modules/project.yaml
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Project'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/me/projects/{project_id}/preferences:
    parameters:
      - $ref: '#/components/parameters/ProjectID'
    get:
      operationId: getProjectPreferences
      tags: [project]
      summary: Read the caller's display settings in a project
      description: >-
        The caller's own settings in the project, for its active members: the
        tab bar of its header and its place in his sidebar. While he has none
        stored they are the defaults, work_items with nothing hidden and
        65535, and reading them writes nothing. A project that does not
        exist, is deleted, or that the caller does not see answers
        project.not_found; one he sees but is not a member of, forbidden.
      security: [{bearer: []}]
      x-problem-codes: [project.not_found, forbidden]
      responses:
        '200':
          description: The caller's settings in the project.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ProjectPreferences'
        default:
          $ref: '#/components/responses/Problem'
    patch:
      operationId: updateProjectPreferences
      tags: [project]
      summary: Change the caller's display settings in a project
      description: >-
        For the project's active members, their own settings. The fields
        given change and the others stay; navigation is replaced whole, and
        the first change stores the caller's settings, the defaults with the
        change applied. A tab the web app does not have, work_items hidden,
        and a tab hidden twice are refused (validation_failed), before the
        project is looked at. An archived project's settings change as any
        other's. A project that does not exist, is deleted, or that the
        caller does not see answers project.not_found; one he sees but is
        not a member of, forbidden.
      security: [{bearer: []}]
      x-problem-codes: [validation_failed, project.not_found, forbidden]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ProjectPreferencesUpdate'
      responses:
        '200':
          description: The caller's settings as changed.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ProjectPreferences'
        default:
          $ref: '#/components/responses/Problem'
components:
````

````old api/modules/project.yaml
          type: integer
        logo_props:
          $ref: '#/components/schemas/LogoProps'
        timezone:
          description: An IANA time zone name.
          type: string

````
````new api/modules/project.yaml
          type: integer
        logo_props:
          $ref: '#/components/schemas/LogoProps'
        timezone:
          description: An IANA time zone name.
          type: string
    ProjectTab:
      description: A tab of a project's header.
      type: string
      enum: [work_items, cycles, modules, views, intake]
    ProjectNavigation:
      description: >-
        The tab bar of a project's header, as the caller has it: the tab the
        project opens on, and the tabs moved under "more", each once and
        never work_items.
      type: object
      additionalProperties: false
      required: [default_tab, hide_in_more_menu]
      properties:
        default_tab:
          $ref: '#/components/schemas/ProjectTab'
        hide_in_more_menu:
          type: array
          items:
            $ref: '#/components/schemas/ProjectTab'
    ProjectPreferences:
      description: The caller's display settings in a project.
      type: object
      additionalProperties: false
      required: [navigation, sort_order]
      properties:
        navigation:
          $ref: '#/components/schemas/ProjectNavigation'
        sort_order:
          description: The project's place in the caller's sidebar, lowest first.
          type: number
    ProjectPreferencesUpdate:
      description: >-
        Changes the fields it names; a field left out keeps its value, and
        navigation replaces the tab bar whole.
      type: object
      additionalProperties: false
      properties:
        navigation:
          $ref: '#/components/schemas/ProjectNavigation'
        sort_order:
          description: The project's place in the caller's sidebar, lowest first.
          type: number

````

`api/openapi.yaml`（修改，1 处）：

````old api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1projects~1{project_id}~1unarchive'
````
````new api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1projects~1{project_id}~1unarchive'
  /api/v0/me/projects/{project_id}/preferences:
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1me~1projects~1{project_id}~1preferences'
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `a982907ae600a79312d8d954769cc63926ac98d00c543e53793532fa78c00eee` | 2239 | `api/dist/openapi.yaml` |
| `309f32276b08fee9f78d1e3f761a9ea62b7ce4e94475c2ee8a0d37cef05a3d2d` | 54 | `server/internal/modules/project/adapter/http/gen/bodyshape.gen.go` |
| `720f0b475c6fd25e1977d4efd634e99f6c2c9b152b2fd3335330c20830a7f797` | 1602 | `server/internal/modules/project/adapter/http/gen/server.gen.go` |
| `d6bb854e74ac4cd7ebb278f7f97adea9ee326002e84004efd61a66cef0cab49c` | 2386 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/bodyshape.gen.go server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: HTTP 和接线**

`server/internal/modules/project/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/project/adapter/http/handler.go
}

// UseCases are the use cases behind the module's operations.
````
````new server/internal/modules/project/adapter/http/handler.go
}

// GetPreferencesUseCase is app.GetProjectPreferences.
type GetPreferencesUseCase interface {
	Execute(ctx context.Context, projectID uuid.UUID) (domain.Preferences, error)
}

// UpdatePreferencesUseCase is app.UpdateProjectPreferences.
type UpdatePreferencesUseCase interface {
	Execute(ctx context.Context, projectID uuid.UUID, p domain.PreferencesPatch) (domain.Preferences, error)
}

// UseCases are the use cases behind the module's operations.
````

````old server/internal/modules/project/adapter/http/handler.go
	ListProjects     ListProjectsUseCase
	CreateProject    CreateProjectUseCase
	GetProject       GetProjectUseCase
	CheckIdentifier  CheckIdentifierUseCase
	UpdateProject    UpdateProjectUseCase
	ArchiveProject   ArchiveProjectUseCase
	UnarchiveProject ArchiveProjectUseCase
	DeleteProject    DeleteProjectUseCase
````
````new server/internal/modules/project/adapter/http/handler.go
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
````

`server/internal/modules/project/adapter/http/preferences.go`（新文件，49 行）：

````file server/internal/modules/project/adapter/http/preferences.go
package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// GetProjectPreferences serves GET /api/v0/me/projects/{project_id}/preferences.
func (h handler) GetProjectPreferences(ctx context.Context, req gen.GetProjectPreferencesRequestObject) (gen.GetProjectPreferencesResponseObject, error) {
	p, err := h.uc.GetPreferences.Execute(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}
	return gen.GetProjectPreferences200JSONResponse(preferencesOut(p)), nil
}

// UpdateProjectPreferences serves PATCH
// /api/v0/me/projects/{project_id}/preferences: the fields the body names go
// to the use case, the navigation whole; a tab the contract does not list
// goes too, and the domain refuses it (422).
func (h handler) UpdateProjectPreferences(ctx context.Context, req gen.UpdateProjectPreferencesRequestObject) (gen.UpdateProjectPreferencesResponseObject, error) {
	b := req.Body
	in := domain.PreferencesPatch{SortOrder: b.SortOrder}
	if n := b.Navigation; n != nil {
		in.Navigation = &domain.Navigation{DefaultTab: string(n.DefaultTab), HideInMoreMenu: make([]string, len(n.HideInMoreMenu))}
		for i, tab := range n.HideInMoreMenu {
			in.Navigation.HideInMoreMenu[i] = string(tab)
		}
	}
	p, err := h.uc.UpdatePreferences.Execute(ctx, req.ProjectID, in)
	if err != nil {
		return nil, err
	}
	return gen.UpdateProjectPreferences200JSONResponse(preferencesOut(p)), nil
}

// preferencesOut is p as the API shows it: the hidden tabs an array, never
// null.
func preferencesOut(p domain.Preferences) gen.ProjectPreferences {
	out := gen.ProjectPreferences{SortOrder: p.SortOrder, Navigation: gen.ProjectNavigation{
		DefaultTab: gen.ProjectTab(p.Navigation.DefaultTab), HideInMoreMenu: make([]gen.ProjectTab, len(p.Navigation.HideInMoreMenu)),
	}}
	for i, tab := range p.Navigation.HideInMoreMenu {
		out.Navigation.HideInMoreMenu[i] = gen.ProjectTab(tab)
	}
	return out
}
````

`server/internal/modules/project/adapter/http/handler_test.go`（修改，3 处）：

````old server/internal/modules/project/adapter/http/handler_test.go
	delete    *fakeDelete
````
````new server/internal/modules/project/adapter/http/handler_test.go
	delete    *fakeDelete
	prefs     *fakePreferences
````

````old server/internal/modules/project/adapter/http/handler_test.go
		f.delete = &fakeDelete{}
	}
````
````new server/internal/modules/project/adapter/http/handler_test.go
		f.delete = &fakeDelete{}
	}
	if f.prefs == nil {
		f.prefs = &fakePreferences{}
	}
````

````old server/internal/modules/project/adapter/http/handler_test.go
		CheckIdentifier: f.check, UpdateProject: f.update, ArchiveProject: f.archive, UnarchiveProject: f.unarchive, DeleteProject: f.delete})
````
````new server/internal/modules/project/adapter/http/handler_test.go
		CheckIdentifier: f.check, UpdateProject: f.update, ArchiveProject: f.archive, UnarchiveProject: f.unarchive, DeleteProject: f.delete,
		GetPreferences: f.prefs, UpdatePreferences: fakeUpdatePreferences{f.prefs}})
````

`server/internal/modules/project/adapter/http/preferences_test.go`（新文件，149 行）：

````file server/internal/modules/project/adapter/http/preferences_test.go
package httpadapter_test

import (
	"context"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakePreferences is both use cases of the display settings: each call is
// recorded as "caller id", a change with what it got; both answer answer,
// or err.
type fakePreferences struct {
	calls  []string
	got    []domain.PreferencesPatch
	answer domain.Preferences
	err    error
}

func (f *fakePreferences) Execute(ctx context.Context, projectID uuid.UUID) (domain.Preferences, error) {
	f.calls = append(f.calls, "get "+caller(ctx)+" "+projectID.String())
	return f.answer, f.err
}

// fakeUpdatePreferences is fakePreferences' change.
type fakeUpdatePreferences struct{ *fakePreferences }

func (f fakeUpdatePreferences) Execute(ctx context.Context, projectID uuid.UUID, p domain.PreferencesPatch) (domain.Preferences, error) {
	f.calls = append(f.calls, "update "+caller(ctx)+" "+projectID.String())
	f.got = append(f.got, p)
	return f.answer, f.err
}

var (
	bobsTabs = domain.Preferences{Navigation: domain.Navigation{DefaultTab: "modules", HideInMoreMenu: []string{"views", "intake"}}, SortOrder: -2.5}
	tabsJSON = `{"navigation":{"default_tab":"modules","hide_in_more_menu":["views","intake"]},"sort_order":-2.5}`
)

// GET goes to the reading use case for the caller and the path's project;
// the answer is 200 with the settings it answers, an empty hidden list as
// [] too.
func TestGetProjectPreferences(t *testing.T) {
	path := "/api/v0/me/projects/" + webID.String() + "/preferences"
	for _, tt := range []struct {
		answer domain.Preferences
		want   string
	}{
		{bobsTabs, tabsJSON},
		{domain.DefaultPreferences(), `{"navigation":{"default_tab":"work_items","hide_in_more_menu":[]},"sort_order":65535}`},
	} {
		prefs := &fakePreferences{answer: tt.answer}
		h := newServer(t, fakes{prefs: prefs})
		if res, body := do(t, h, request(http.MethodGet, path, "bob", "")); res.StatusCode != http.StatusOK || body != tt.want+"\n" {
			t.Errorf("GET = %d %s, want 200 %s", res.StatusCode, body, tt.want)
		}
		if want := []string{"get bob " + webID.String()}; !slices.Equal(prefs.calls, want) {
			t.Errorf("calls = %q, want %q", prefs.calls, want)
		}
	}
}

// The fields the body names go to the changing use case, the navigation
// whole, a tab the contract does not list too (the domain refuses it); a
// field left out nil. The answer is 200 with the settings it answers.
func TestUpdateProjectPreferencesPassesTheChange(t *testing.T) {
	prefs := &fakePreferences{answer: bobsTabs}
	h := newServer(t, fakes{prefs: prefs})
	path := "/api/v0/me/projects/" + webID.String() + "/preferences"
	for _, body := range []string{
		`{"navigation":{"default_tab":"cycles","hide_in_more_menu":["intake","views"]},"sort_order":-2.5}`,
		`{}`,
		`{"navigation":{"default_tab":"pages","hide_in_more_menu":[]}}`,
		`{"sort_order":0}`,
	} {
		if res, got := do(t, h, request(http.MethodPatch, path, "alice", body)); res.StatusCode != http.StatusOK || got != tabsJSON+"\n" {
			t.Errorf("PATCH %s = %d %s, want 200 %s", body, res.StatusCode, got, tabsJSON)
		}
	}
	want := []domain.PreferencesPatch{
		{Navigation: &domain.Navigation{DefaultTab: "cycles", HideInMoreMenu: []string{"intake", "views"}}, SortOrder: ptr(-2.5)},
		{},
		{Navigation: &domain.Navigation{DefaultTab: "pages", HideInMoreMenu: []string{}}},
		{SortOrder: ptr(0.0)},
	}
	if !reflect.DeepEqual(prefs.got, want) {
		t.Errorf("inputs = %+v, want %+v", prefs.got, want)
	}
	if len(prefs.calls) != 4 || prefs.calls[0] != "update alice "+webID.String() {
		t.Errorf("calls = %q, want four updates by alice of web", prefs.calls)
	}
}

// A field the body may not have, a navigation without one of its two
// fields, a null, a value of another type: refused as bad_request before
// the use case.
func TestUpdateProjectPreferencesHoldsTheBodyToItsStructure(t *testing.T) {
	prefs := &fakePreferences{answer: bobsTabs}
	h := newServer(t, fakes{prefs: prefs})
	for _, body := range []string{`{"pages":{}}`, `{"navigation":{"default_tab":"views"}}`, `{"navigation":{"hide_in_more_menu":[]}}`,
		`{"navigation":null}`, `{"sort_order":null}`, `{"sort_order":"1"}`, `{"navigation":{"default_tab":"views","hide_in_more_menu":"views"}}`,
		`{"navigation":{"default_tab":"views","hide_in_more_menu":[],"pages":[]}}`, `[]`} {
		if res, got := do(t, h, request(http.MethodPatch, "/api/v0/me/projects/"+webID.String()+"/preferences", "alice", body)); res.StatusCode != http.StatusBadRequest {
			t.Errorf("PATCH %s = %d %s, want 400", body, res.StatusCode, got)
		}
	}
	if len(prefs.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", prefs.calls)
	}
}

// The use cases' refusals, as the contract declares them.
func TestProjectPreferencesRefusals(t *testing.T) {
	path := "/api/v0/me/projects/" + webID.String() + "/preferences"
	tests := []struct {
		name   string
		err    error
		method string
		status int
		want   string
	}{
		{"no project", domain.ErrNotFound, http.MethodGet, http.StatusNotFound,
			`{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`},
		{"no member", shared.Forbidden(), http.MethodGet, http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"no project", domain.ErrNotFound, http.MethodPatch, http.StatusNotFound,
			`{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`},
		{"no member", shared.Forbidden(), http.MethodPatch, http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"an unknown tab", shared.Invalid(shared.FieldError{Field: "navigation.default_tab", Code: "invalid_format",
			Message: "is not one of work_items, cycles, modules, views, intake"}), http.MethodPatch, http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"navigation.default_tab","code":"invalid_format","message":"is not one of work_items, cycles, modules, views, intake"}]}`},
	}
	for _, tt := range tests {
		h := newServer(t, fakes{prefs: &fakePreferences{err: tt.err}})
		body := ""
		if tt.method == http.MethodPatch {
			body = `{"sort_order":1}`
		}
		if res, got := do(t, h, request(tt.method, path, "alice", body)); res.StatusCode != tt.status || got != tt.want+"\n" {
			t.Errorf("%s %s = %d %s, want %d %s", tt.name, tt.method, res.StatusCode, got, tt.status, tt.want)
		}
	}
}
````

`server/internal/modules/project/module.go`（修改，2 处）：

````old server/internal/modules/project/module.go
// projects and checking an identifier, carries out the workspace module's cascades on the projects
````
````new server/internal/modules/project/module.go
// projects, checking an identifier and each member's display settings,
// carries out the workspace module's cascades on the projects
````

````old server/internal/modules/project/module.go
		ListProjects:     app.NewListProjects(d.Workspaces, store, d.Authorizer),
		GetProject:       app.NewGetProject(store, d.Authorizer),
		CheckIdentifier:  app.NewCheckProjectIdentifier(d.Workspaces, store, d.Authorizer),
		UpdateProject:    app.NewUpdateProject(store, locks, d.Tx, d.Clock),
		ArchiveProject:   app.NewArchiveProject(store, locks, d.Tx, d.Clock),
		UnarchiveProject: app.NewUnarchiveProject(store, locks, d.Tx, d.Clock),
		DeleteProject:    app.NewDeleteProject(store, locks, d.Tx, d.Clock),
````
````new server/internal/modules/project/module.go
		ListProjects:      app.NewListProjects(d.Workspaces, store, d.Authorizer),
		GetProject:        app.NewGetProject(store, d.Authorizer),
		CheckIdentifier:   app.NewCheckProjectIdentifier(d.Workspaces, store, d.Authorizer),
		UpdateProject:     app.NewUpdateProject(store, locks, d.Tx, d.Clock),
		ArchiveProject:    app.NewArchiveProject(store, locks, d.Tx, d.Clock),
		UnarchiveProject:  app.NewUnarchiveProject(store, locks, d.Tx, d.Clock),
		DeleteProject:     app.NewDeleteProject(store, locks, d.Tx, d.Clock),
		GetPreferences:    app.NewGetProjectPreferences(store, d.Authorizer),
		UpdatePreferences: app.NewUpdateProjectPreferences(store, locks, d.Tx, d.Clock),
````

- [ ] **Step 3: 矩阵和组合出的测试**

`server/internal/bootstrap/permission_matrix_project_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_project_test.go
import (
````
````new server/internal/bootstrap/permission_matrix_project_test.go
import (
	"encoding/json"
````

````old server/internal/bootstrap/permission_matrix_project_test.go
			cells: map[caller]cell{callerArchivedAdmin: cellNoContent}},
````
````new server/internal/bootstrap/permission_matrix_project_test.go
			cells: map[caller]cell{callerArchivedAdmin: cellNoContent}},
		// Every active member of the project, his own settings (M3 design
		// 9.2): PM+WA as its member, and not WA-, who is none.
		{op: "getProjectPreferences", columns: projectColumns, request: toProjectPreferences(http.MethodGet, ""),
			cells: ofProject(cellOK, cellOK, cellOK, cellOK, cellForbidden, cellForbidden), check: readsPreferences("work_items", "[]")},
		{op: "updateProjectPreferences", write: true, columns: projectColumns,
			request: toProjectPreferences(http.MethodPatch, `{"navigation":{"default_tab":"cycles","hide_in_more_menu":["views"]}}`),
			cells:   ofProject(cellOK, cellOK, cellOK, cellOK, cellForbidden, cellForbidden), check: readsPreferences("cycles", `["views"]`)},
		// An archived project's settings change as any other's (M3 design
		// 3.19).
		{op: "updateProjectPreferences", variant: "archived", write: true, columns: archivedColumns,
			request: toProjectPreferences(http.MethodPatch, `{"navigation":{"default_tab":"cycles","hide_in_more_menu":["views"]}}`),
			cells:   map[caller]cell{callerArchivedAdmin: cellOK}, check: readsPreferences("cycles", `["views"]`)},
	}
}

// toProjectPreferences is the request of a row whose callers each send method to
// their display settings in the project their column targets.
func toProjectPreferences(method, body string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return method, "/api/v0/me/projects/" + s.project(projectOf(c)).String() + "/preferences", body
	}
}

// readsPreferences: the settings with the default tab tab and the hidden
// tabs hidden, as JSON, at the place every seeded member has, the default.
func readsPreferences(tab, hidden string) func(t *testing.T, c caller, _ seeded, answer string) {
	return func(t *testing.T, c caller, _ seeded, answer string) {
		var p struct {
			Navigation struct {
				DefaultTab     string          `json:"default_tab"`
				HideInMoreMenu json.RawMessage `json:"hide_in_more_menu"`
			} `json:"navigation"`
			SortOrder float64 `json:"sort_order"`
		}
		decodeAnswer(t, answer, &p)
		if p.Navigation.DefaultTab != tab || string(p.Navigation.HideInMoreMenu) != hidden || p.SortOrder != 65535 {
			t.Errorf("%s reads %s; want %s, %s hidden, at 65535", c, answer, tab, hidden)
		}
````

`server/internal/bootstrap/project_writes_test.go`（修改，1 处）：

````old server/internal/bootstrap/project_writes_test.go
			"SELECT updated_at, updated_by_id = $2 AND archived_at IS NULL FROM projects WHERE id = $1"},
````
````new server/internal/bootstrap/project_writes_test.go
			"SELECT updated_at, updated_by_id = $2 AND archived_at IS NULL FROM projects WHERE id = $1"},
		{"updateProjectPreferences", http.MethodPatch, "/api/v0/me/projects/" + web.String() + "/preferences", `{"sort_order":5}`, http.StatusOK,
			"SELECT updated_at, updated_by_id = $2 FROM project_user_properties WHERE project_id = $1 AND user_id = $2 AND deleted_at IS NULL"},
````

`server/internal/bootstrap/project_preferences_test.go`（新文件，74 行）：

````file server/internal/bootstrap/project_preferences_test.go
package bootstrap

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
	"uuid"

	projectpg "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Each member reads and changes his own display settings in a project, on
// the wired app (M3 design 3.18): alice's change of hers in Web leaves
// bob's there, and hers in Ops, as they were, and each reads back his own,
// a member without any the defaults. Bob is made a member of acme and Web
// through the stores, his settings in Web at 3.
func TestEachMemberHasHisOwnDisplaySettings(t *testing.T) {
	contract, base, pool, alice, aliceID, web, ops := twoProjects(t)
	bob := registerAccount(t, contract, base, "bob@example.com").AccessToken
	bobID := accountID(t, contract, base, bob)
	var acme uuid.UUID
	if err := pool.QueryRow(context.Background(), "SELECT workspace_id FROM projects WHERE id = $1", web).Scan(&acme); err != nil {
		t.Fatal(err)
	}
	ctx, now := context.Background(), time.Now()
	if err := workspacepg.New(pool).CreateMember(ctx, workspaceapp.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, MemberID: bobID,
		Role: shared.RoleMember, CreatedBy: aliceID, Now: now}); err != nil {
		t.Fatal(err)
	}
	projects := projectpg.New(pool)
	if err := projects.CreateMember(ctx, projectapp.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, MemberID: bobID,
		Role: shared.RoleMember, CreatedBy: aliceID, Now: now}); err != nil {
		t.Fatal(err)
	}
	if err := projects.CreatePreferences(ctx, projectapp.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, UserID: bobID,
		SortOrder: 3, CreatedBy: aliceID, Now: now}); err != nil {
		t.Fatal(err)
	}
	settings := func(token string, project uuid.UUID) string {
		t.Helper()
		status, body := call(t, contract, http.MethodGet, base+"/api/v0/me/projects/"+project.String()+"/preferences", token, "")
		if status != http.StatusOK {
			t.Fatalf("reading the settings in %s = %d %s", project, status, body)
		}
		return strings.TrimSuffix(body, "\n")
	}
	alicesOps := settings(alice, ops)

	changed := `{"navigation":{"default_tab":"cycles","hide_in_more_menu":["intake"]},"sort_order":1}`
	if status, body := call(t, contract, http.MethodPatch, base+"/api/v0/me/projects/"+web.String()+"/preferences", alice, changed); status !=
		http.StatusOK || body != changed+"\n" {
		t.Fatalf("alice's change in Web = %d %s, want 200 %s", status, body, changed)
	}

	for _, tt := range []struct {
		who, token string
		project    uuid.UUID
		want       string
	}{
		{"alice in Web", alice, web, changed},
		{"bob in Web", bob, web, `{"navigation":{"default_tab":"work_items","hide_in_more_menu":[]},"sort_order":3}`},
		{"alice in Ops", alice, ops, alicesOps},
	} {
		if got := settings(tt.token, tt.project); got != tt.want {
			t.Errorf("%s: %s, want %s", tt.who, got, tt.want)
		}
	}
}
````

`server/internal/bootstrap/project_write_locks_test.go`（修改，1 处）：

````old server/internal/bootstrap/project_write_locks_test.go
	{op: "unarchiveProject", method: http.MethodPost, path: "/api/v0/projects/%s/unarchive", want: http.StatusOK},
````
````new server/internal/bootstrap/project_write_locks_test.go
	{op: "unarchiveProject", method: http.MethodPost, path: "/api/v0/projects/%s/unarchive", want: http.StatusOK},
	{op: "updateProjectPreferences", method: http.MethodPatch, path: "/api/v0/me/projects/%s/preferences", body: `{"sort_order":1}`,
		want: http.StatusOK},
````

- [ ] **Step 4: 测试、lint、前端检查**

Run: `go -C server test -count=1 ./internal/modules/project/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix$|TestThePermissionMatrixCoversEveryOperation|TestMatrixViolationsCatchesEach|TestAPIRoutesAreTheContractsOperations|TestOperationsThatNeedATokenAnswer401WithoutOne|TestBodiesThatBreakTheStructureAnswer400|TestEachMemberHasHisOwnDisplaySettings|TestTheWritesOnAProjectStampTheirRequest|TestEachWriteOnAProjectSharesItsWorkspaceFirst' ./internal/bootstrap/`
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

- [ ] **Step 5: 提交**

```bash
git add api/modules/project.yaml api/openapi.yaml server/internal/bootstrap/permission_matrix_project_test.go server/internal/bootstrap/project_preferences_test.go server/internal/bootstrap/project_write_locks_test.go server/internal/bootstrap/project_writes_test.go server/internal/modules/project/adapter/http/handler.go server/internal/modules/project/adapter/http/handler_test.go server/internal/modules/project/adapter/http/preferences.go server/internal/modules/project/adapter/http/preferences_test.go server/internal/modules/project/module.go api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/bodyshape.gen.go server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P4b): the display settings' operations

GET and PATCH /api/v0/me/projects/{project_id}/preferences, for the
project's active members, each his own. The contract names the five
tabs; a tab the web app does not have reaches the domain, which refuses
it (422), and a body of another shape is a 400.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| handler 吞掉两个用例的失败 | `TestProjectPreferencesRefusals` | 单元 |
| `project.New` 给修改、归档、恢复、改设置不开事务的事务管理器 | `TestAProjectWriteAndADemotionSerialize`（Task 14 起） | 组合 |
| `project.New` 给删除不开事务的事务管理器 | `TestAProjectDeletionRefusedAtItsCommitChangesNoRow`；`TestADemotionAndAProjectsDeletionSerialize`（Task 14 起） | 组合 |
| `project.New` 给修改、归档、恢复、改设置固定在 2000 年的时钟 | `TestTheWritesOnAProjectStampTheirRequest` | 组合 |
| `project.New` 给删除固定在 2000 年的时钟 | `TestDeletingAProjectLeavesNoUndeletedRowUnderIt` | 组合 |
| `project.New` 给六个用例（修改、归档、恢复、删除、读设置、改设置）各一个谁都当作管理员放行的 `Authorizer`（写的经它们自己的 `Locks`） | `TestPermissionMatrix` | 组合 |
| 只有改设置不锁工作区 | `TestEachWriteOnAProjectSharesItsWorkspaceFirst`；`TestAProjectWriteAndADemotionSerialize`（Task 14 起） | 组合 |
| `project.New` 把归档和恢复接反 | `TestPermissionMatrix`、`TestTheWritesOnAProjectStampTheirRequest` | 组合 |

清扫 4 的接线变异按本 Task 的 `project.New` 写成（Task 3–8 的六个用例都已接上），所以列在这里；Task 3、4、6 的接线在各自的 Task 里同样由这些测试核对。

**Done when:** 矩阵三行（25 格）通过；每个成员读写自己的设置；接线的变异由组合出的测试发现。

---

### Task 9: `listProjectMembers`；矩阵的账户登记

**Files:**
- Create: `server/internal/modules/project/adapter/http/members.go`、`server/internal/modules/project/adapter/http/members_test.go`、`server/internal/modules/project/app/list_members.go`、`server/internal/modules/project/app/list_members_test.go`、`server/internal/modules/project/domain/member.go`
- Modify: `api/modules/project.yaml`、`api/openapi.yaml`、`server/internal/bootstrap/permission_matrix_project_test.go`、`server/internal/bootstrap/permission_matrix_seeded_test.go`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/project/adapter/http/handler.go`、`server/internal/modules/project/adapter/http/handler_test.go`、`server/internal/modules/project/adapter/postgres/failures_test.go`、`server/internal/modules/project/adapter/postgres/members.go`、`server/internal/modules/project/adapter/postgres/members_test.go`、`server/internal/modules/project/adapter/postgres/queries/members.sql`、`server/internal/modules/project/app/ports.go`、`server/internal/modules/project/domain/actions.go`、`server/internal/modules/project/module.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`server/internal/modules/project/adapter/postgres/gen/members.sql.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.8，M3 设计 3.12、5.2）：`GET /api/v0/projects/{project_id}/members`，200 `ProjectMemberList{data: ProjectMember[]}`（`ProjectMember{id, project_id, member_id, role, created_at}`）；码 `[project.not_found, forbidden]`；操作名 `project_member.list`，规则 `{Level: LevelProject, Roles: [admin, member, guest]}`：项目的有效成员，不是成员的工作区管理员 403。
- `domain.Member{ID, ProjectID, MemberID, Role, CreatedAt}`；`app.MemberLister`（`ProjectFinder` + `ListMembers(ctx, projectID)`）；`app.NewListProjectMembers(members MemberLister, auth)`：`findAndDecide` → `ListMembers`，不开事务。列表只读 `project_members`：有效、未删除的成员关系，按建立的时刻、再按 id（spec 第 3 节第 4 条：工作区成员关系结束时项目成员关系由 P5 结束）。
- 矩阵：`seeded.account(c)`：`prepareMatrix` 按列登记每个账户的 id；不是列的账户让测试立刻失败；没有数据库时是 `uuid.Nil`，`matrixViolations` 照样能拼请求。P4b 的路径参数只有 `{project_id}`（brief 移交第 4 条），正文里的账户经 `s.account` 取（Task 12）。

**Tests:**
- `adapter/postgres/members_test.go`：`TestListMembers`（只列这个项目有效、未删除的成员关系，按建立的时刻、再按 id：erin 的最后建立而最先存入，排在最后；alice、bob 同一时刻建立，按 id；不列已结束的 carol、已删除的 dave、别的项目的 frank）。`TestMemberships` 的准备随之扩充。
- `app/list_members_test.go`：`TestListProjectMembers`（判定之后原样答存储的列表：web 的三个有效成员，不含已结束的 dave；不开事务）；`TestListProjectMembersRefuses`（没有调用者；没有、看不到 404；成员的 403 来自 `Authorizer`；每个端口的失败原样返回，之后的不调用）。
- `adapter/http/members_test.go`：`TestListProjectMembers`（列表照用例的顺序，没有成员时 `[]`）。
- 矩阵一行（12 格 `ofProject(200, 200, 200, 200, 403, 403)`；答案核对 `acme/public` 的五个有效成员和角色，按 `prepareMatrix` 建立的顺序：PA、PM、PG 的账户、PM+WA、被移出的成员（他的项目成员关系由替身留作有效），不含 `partingStates` 结束的 WG-）。

- [ ] **Step 1: 接口描述和查询**

`api/modules/project.yaml`（修改，2 处）：

````old api/modules/project.yaml
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Project'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/me/projects/{project_id}/preferences:
````
````new api/modules/project.yaml
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Project'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/projects/{project_id}/members:
    parameters:
      - $ref: '#/components/parameters/ProjectID'
    get:
      operationId: listProjectMembers
      tags: [project]
      summary: List a project's members
      description: >-
        For the project's active members: its active members, each with his
        role in it, in the order they became members. A project that does
        not exist, is deleted, or that the caller does not see answers
        project.not_found; one he sees but is not a member of, forbidden.
      security: [{bearer: []}]
      x-problem-codes: [project.not_found, forbidden]
      responses:
        '200':
          description: The project's active members.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ProjectMemberList'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/me/projects/{project_id}/preferences:
````

````old api/modules/project.yaml
      additionalProperties: false
      properties:
        navigation:
          $ref: '#/components/schemas/ProjectNavigation'
        sort_order:
          description: The project's place in the caller's sidebar, lowest first.
          type: number

````
````new api/modules/project.yaml
      additionalProperties: false
      properties:
        navigation:
          $ref: '#/components/schemas/ProjectNavigation'
        sort_order:
          description: The project's place in the caller's sidebar, lowest first.
          type: number
    ProjectMember:
      description: An active membership of a project.
      type: object
      additionalProperties: false
      required: [id, project_id, member_id, role, created_at]
      properties:
        id:
          description: The membership's id.
          type: string
          format: uuid
        project_id:
          type: string
          format: uuid
        member_id:
          description: The member's account.
          type: string
          format: uuid
        role:
          $ref: '#/components/schemas/ProjectRole'
        created_at:
          description: When the membership was made; one restored keeps its time.
          type: string
          format: date-time
    ProjectMemberList:
      type: object
      additionalProperties: false
      required: [data]
      properties:
        data:
          type: array
          items:
            $ref: '#/components/schemas/ProjectMember'

````

`api/openapi.yaml`（修改，1 处）：

````old api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1projects~1{project_id}~1unarchive'
````
````new api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1projects~1{project_id}~1unarchive'
  /api/v0/projects/{project_id}/members:
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1projects~1{project_id}~1members'
````

`server/internal/modules/project/adapter/postgres/queries/members.sql`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/queries/members.sql
WHERE project_id = sqlc.arg(project_id) AND member_id = ANY (sqlc.arg(member_ids)::uuid[]) AND deleted_at IS NULL;

````
````new server/internal/modules/project/adapter/postgres/queries/members.sql
WHERE project_id = sqlc.arg(project_id) AND member_id = ANY (sqlc.arg(member_ids)::uuid[]) AND deleted_at IS NULL;

-- name: ListMembers :many
-- listProjectMembers (M3 design 3.12, 5.2): the project's active undeleted memberships, in the order they were made,
-- then by id. It reads project_members alone: an active member of a project stays an active member of its workspace,
-- for every growth locks his workspace membership and every shrinking ends his project memberships (3.6 conventions 3
-- and 6).
SELECT id, project_id, member_id, role, created_at
FROM project_members
WHERE project_id = sqlc.arg(project_id) AND is_active AND deleted_at IS NULL
ORDER BY created_at, id;

````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `5a809af5565240469664b6ebcbf698f8a3ec8e3e3a17fa9491599effb251eff9` | 2300 | `api/dist/openapi.yaml` |
| `15951961f03682a96638db614bc59d48f002d421ee11741eb6c5fa14e0057338` | 1728 | `server/internal/modules/project/adapter/http/gen/server.gen.go` |
| `a72e84b0944f2967652b5a03cec85b0b0fa965f97f6afaaa2de8680ff2eb5095` | 132 | `server/internal/modules/project/adapter/postgres/gen/members.sql.go` |
| `03c552a83c82386ece60d1026414d73ff24eec453745d63a732b5cd5a7258fd5` | 2459 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/server.gen.go server/internal/modules/project/adapter/postgres/gen/members.sql.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: 领域、端口、存储**

`server/internal/modules/project/domain/member.go`（新文件，19 行）：

````file server/internal/modules/project/domain/member.go
package domain

import (
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Member is an active membership of a project, as the members' list shows
// it (M3 design 5.2): its id, the project, the member's account, his role
// in the project and when he became its member.
type Member struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	MemberID  uuid.UUID
	Role      shared.Role
	CreatedAt time.Time
}
````

`server/internal/modules/project/app/ports.go`（修改，1 处）：

````old server/internal/modules/project/app/ports.go
	ProjectSharer
````
````new server/internal/modules/project/app/ports.go
	ProjectSharer
}

// MemberLister is listProjectMembers' repository.
type MemberLister interface {
	ProjectFinder
	// ListMembers lists projectID's active memberships, in the order they
	// were made, then by id.
	ListMembers(ctx context.Context, projectID uuid.UUID) ([]domain.Member, error)
````

`server/internal/modules/project/adapter/postgres/members.go`（修改，2 处）：

````old server/internal/modules/project/adapter/postgres/members.go
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
````
````new server/internal/modules/project/adapter/postgres/members.go
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
````

````old server/internal/modules/project/adapter/postgres/members.go
	return out, nil
}

````
````new server/internal/modules/project/adapter/postgres/members.go
	return out, nil
}

// ListMembers lists projectID's active memberships, in the order they were
// made (app.MemberLister).
func (s *Store) ListMembers(ctx context.Context, projectID uuid.UUID) ([]domain.Member, error) {
	rows, err := s.queries(ctx).ListMembers(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list members of project %s: %w", projectID, err)
	}
	out := make([]domain.Member, len(rows))
	for i, r := range rows {
		out[i] = domain.Member{ID: r.ID, ProjectID: r.ProjectID, MemberID: r.MemberID, Role: shared.Role(r.Role), CreatedAt: r.CreatedAt}
	}
	return out, nil
}

````

`server/internal/modules/project/adapter/postgres/members_test.go`（修改，3 处）：

````old server/internal/modules/project/adapter/postgres/members_test.go
	"testing"
````
````new server/internal/modules/project/adapter/postgres/members_test.go
	"testing"
	"time"
````

````old server/internal/modules/project/adapter/postgres/members_test.go
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
````
````new server/internal/modules/project/adapter/postgres/members_test.go
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
````

````old server/internal/modules/project/adapter/postgres/members_test.go
	}
}

````
````new server/internal/modules/project/adapter/postgres/members_test.go
	}
}

// ListMembers lists the project's active undeleted memberships alone, by
// the time they were made, then by id: erin's, made last but stored first,
// comes last; alice's and bob's, made at the same time, by their ids. Not
// carol's, ended; not dave's, deleted; not frank's of another project.
func TestListMembers(t *testing.T) {
	s, pool := newStore(t)
	var ids []uuid.UUID
	for _, email := range []string{"alice@corp.com", "bob@corp.com", "carol@corp.com", "dave@corp.com", "erin@corp.com", "frank@corp.com"} {
		ids = append(ids, newAccount(t, pool, email))
	}
	alice, bob, carol, dave, erin, frank := ids[0], ids[1], ids[2], ids[3], ids[4], ids[5]
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	erins := seedMember(t, pool, acme, web, erin, 5, true)
	exec(t, pool, "UPDATE project_members SET created_at = $2 WHERE id = $1", erins, now.Add(time.Hour))
	seedMember(t, pool, acme, web, carol, 15, false)
	bobs, alices := seedMember(t, pool, acme, web, bob, 15, true), seedMember(t, pool, acme, web, alice, 20, true)
	exec(t, pool, "UPDATE project_members SET deleted_at = $2 WHERE id = $1", seedMember(t, pool, acme, web, dave, 15, true), now)
	seedMember(t, pool, acme, ops, frank, 20, true)
	first, second := domain.Member{ID: bobs, ProjectID: web, MemberID: bob, Role: shared.RoleMember, CreatedAt: now},
		domain.Member{ID: alices, ProjectID: web, MemberID: alice, Role: shared.RoleAdmin, CreatedAt: now}
	if alices.String() < bobs.String() {
		first, second = second, first
	}
	want := []domain.Member{first, second, {ID: erins, ProjectID: web, MemberID: erin, Role: shared.RoleGuest, CreatedAt: now.Add(time.Hour)}}
	got, err := s.ListMembers(context.Background(), web)
	if err != nil || len(got) != len(want) {
		t.Fatalf("ListMembers() = %+v, %v; want %+v", got, err, want)
	}
	for i := range want {
		if got[i].ID != want[i].ID || got[i].ProjectID != want[i].ProjectID || got[i].MemberID != want[i].MemberID || got[i].Role != want[i].Role ||
			!got[i].CreatedAt.Equal(want[i].CreatedAt) {
			t.Errorf("ListMembers()[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
	if got, err := s.ListMembers(context.Background(), uuid.NewV7()); err != nil || len(got) != 0 {
		t.Errorf("ListMembers() of no project = %v, %v; want none", got, err)
	}
}

````

`server/internal/modules/project/adapter/postgres/failures_test.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("SetArchived() = %v; want context.Canceled", err)
	}
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("SetArchived() = %v; want context.Canceled", err)
	}
	if m, err := s.ListMembers(cancelled, web); !failed(err) || m != nil {
		t.Errorf("ListMembers() = %v, %v; want context.Canceled, not none", m, err)
	}
````

- [ ] **Step 3: 操作名、规则、用例**

`server/internal/modules/project/domain/actions.go`（修改，2 处）：

````old server/internal/modules/project/domain/actions.go
	ActionPreferencesUpdate shared.Action = "project_preferences.update"
````
````new server/internal/modules/project/domain/actions.go
	ActionPreferencesUpdate shared.Action = "project_preferences.update"
	// ActionMemberList is listing a project's members: listProjectMembers.
	ActionMemberList shared.Action = "project_member.list"
````

````old server/internal/modules/project/domain/actions.go
		ActionUnarchive, ActionDelete, ActionPreferencesRead, ActionPreferencesUpdate}
````
````new server/internal/modules/project/domain/actions.go
		ActionUnarchive, ActionDelete, ActionPreferencesRead, ActionPreferencesUpdate,
		ActionMemberList}
````

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"project_preferences.update": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
````
````new server/internal/modules/access/domain/rules.go
	"project_preferences.update": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// Every active member of the project (M3 design 9.2).
	"project_member.list": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"project_preferences.update": {allowed, allowed, allowed, allowed, allowed, forbidden, forbidden, invisible, invisible, invisible,
		invisible, forbidden, invisible, invisible, invisible, forbidden, forbidden},
````
````new server/internal/modules/access/domain/rules_test.go
	"project_preferences.update": {allowed, allowed, allowed, allowed, allowed, forbidden, forbidden, invisible, invisible, invisible,
		invisible, forbidden, invisible, invisible, invisible, forbidden, forbidden},
	"project_member.list": {allowed, allowed, allowed, allowed, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
````

`server/internal/modules/project/app/list_members.go`（新文件，35 行）：

````file server/internal/modules/project/app/list_members.go
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListProjectMembers lists a project's members: GET
// /api/v0/projects/{project_id}/members.
type ListProjectMembers struct {
	members MemberLister
	auth    shared.Authorizer
}

// NewListProjectMembers returns the use case.
func NewListProjectMembers(members MemberLister, auth shared.Authorizer) *ListProjectMembers {
	return &ListProjectMembers{members: members, auth: auth}
}

// Execute lists the project's active members, in the order they became
// members (M3 design 3.12), after the decision on project_member.list. A
// read opens no transaction.
func (u *ListProjectMembers) Execute(ctx context.Context, projectID uuid.UUID) ([]domain.Member, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	if err := findAndDecide(ctx, u.members, u.auth, actor, projectID, domain.ActionMemberList); err != nil {
		return nil, err
	}
	return u.members.ListMembers(ctx, projectID)
}
````

`server/internal/modules/project/app/list_members_test.go`（新文件，94 行）：

````file server/internal/modules/project/app/list_members_test.go
package app_test

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListMembers is the project's active memberships, by account.
func (f *fakeStore) ListMembers(ctx context.Context, projectID uuid.UUID) ([]domain.Member, error) {
	f.log.add(ctx, "ListMembers %s", projectID)
	if err := f.fail("ListMembers"); err != nil {
		return nil, err
	}
	var out []domain.Member
	members := f.projects[projectID].members
	for _, user := range slices.SortedFunc(maps.Keys(members), func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) }) {
		if m := members[user]; m.Active {
			out = append(out, domain.Member{ID: m.ID, ProjectID: projectID, MemberID: user, Role: m.Role, CreatedAt: now})
		}
	}
	return out, nil
}

// listed are the calls of user's list of project's members: the project's
// workspace, the decision, the list; none in a transaction.
func listed(user, project uuid.UUID) []string {
	return []string{"ProjectWorkspace " + project.String() + " outside tx",
		fmt.Sprintf("Authorize %s %s on %s/%s outside tx", user, domain.ActionMemberList, acme.ID, project),
		fmt.Sprintf("ListMembers %s outside tx", project)}
}

// ListProjectMembers decides project_member.list on the project, then
// answers the store's list as it is: web's three active members, not
// dave, whose membership ended. A read opens no transaction.
func TestListProjectMembers(t *testing.T) {
	f := newWrites()
	got, err := app.NewListProjectMembers(f.store, f.auth).Execute(as(bob), webID)
	members := f.store.projects[webID].members
	var want []domain.Member
	for _, user := range []uuid.UUID{alice, bob, carol} {
		want = append(want, domain.Member{ID: members[user].ID, ProjectID: webID, MemberID: user, Role: members[user].Role, CreatedAt: now})
	}
	slices.SortFunc(want, func(a, b domain.Member) int { return strings.Compare(a.MemberID.String(), b.MemberID.String()) })
	if err != nil || !reflect.DeepEqual(got, want) || !slices.Equal(f.log.calls, listed(bob, webID)) {
		t.Errorf("Execute() = %+v, %v, calls %q; want %+v, calls %q", got, err, f.log.calls, want, listed(bob, webID))
	}
}

// Refusals, each in its place: no caller; a project not there, or not
// visible: 404; a member's 403 from the Authorizer. Every port's failure
// comes back as itself, after the calls before it and none after.
func TestListProjectMembersRefuses(t *testing.T) {
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
		{"not seen", as(erin), webID, nil, domain.ErrNotFound, listed(erin, webID)[:2]},
		{"forbidden", as(alice), webID, nil, shared.Forbidden(), listed(alice, webID)[:2]},
		{"the project failing", as(bob), webID, func(f *writeFixture) { f.store.errs = map[string]error{"ProjectWorkspace": errDisk} }, errDisk,
			listed(bob, webID)[:1]},
		{"the decision failing", as(bob), webID, func(f *writeFixture) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, errDisk,
			listed(bob, webID)[:2]},
		{"the list failing", as(bob), webID, func(f *writeFixture) { f.store.errs = map[string]error{"ListMembers": errDisk} }, errDisk,
			listed(bob, webID)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newWrites()
			if tt.fail != nil {
				tt.fail(f)
			}
			got, err := app.NewListProjectMembers(f.store, f.auth).Execute(tt.ctx, tt.id)
			if !refusedAs(err, tt.want) || got != nil || !slices.Equal(f.log.calls, tt.calls) {
				t.Errorf("Execute() = %+v, %v, calls %q; want %v, calls %q", got, err, f.log.calls, tt.want, tt.calls)
			}
		})
	}
}
````

- [ ] **Step 4: HTTP 和接线**

`server/internal/modules/project/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/project/adapter/http/handler.go
}

// UseCases are the use cases behind the module's operations.
````
````new server/internal/modules/project/adapter/http/handler.go
}

// ListMembersUseCase is app.ListProjectMembers.
type ListMembersUseCase interface {
	Execute(ctx context.Context, projectID uuid.UUID) ([]domain.Member, error)
}

// UseCases are the use cases behind the module's operations.
````

````old server/internal/modules/project/adapter/http/handler.go
	UpdatePreferences UpdatePreferencesUseCase
````
````new server/internal/modules/project/adapter/http/handler.go
	UpdatePreferences UpdatePreferencesUseCase
	ListMembers       ListMembersUseCase
````

`server/internal/modules/project/adapter/http/members.go`（新文件，26 行）：

````file server/internal/modules/project/adapter/http/members.go
package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// ListProjectMembers serves GET /api/v0/projects/{project_id}/members.
func (h handler) ListProjectMembers(ctx context.Context, req gen.ListProjectMembersRequestObject) (gen.ListProjectMembersResponseObject, error) {
	list, err := h.uc.ListMembers.Execute(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}
	return gen.ListProjectMembers200JSONResponse(members(list)), nil
}

// members is list as the API shows it: data an array, never null.
func members(list []domain.Member) gen.ProjectMemberList {
	out := gen.ProjectMemberList{Data: make([]gen.ProjectMember, len(list))}
	for i, m := range list {
		out.Data[i] = gen.ProjectMember{ID: m.ID, ProjectID: m.ProjectID, MemberID: m.MemberID, Role: gen.ProjectRole(m.Role), CreatedAt: m.CreatedAt}
	}
	return out
}
````

`server/internal/modules/project/adapter/http/handler_test.go`（修改，3 处）：

````old server/internal/modules/project/adapter/http/handler_test.go
	prefs     *fakePreferences
````
````new server/internal/modules/project/adapter/http/handler_test.go
	prefs     *fakePreferences
	members   *fakeMembers
````

````old server/internal/modules/project/adapter/http/handler_test.go
		f.prefs = &fakePreferences{}
	}
````
````new server/internal/modules/project/adapter/http/handler_test.go
		f.prefs = &fakePreferences{}
	}
	if f.members == nil {
		f.members = &fakeMembers{}
	}
````

````old server/internal/modules/project/adapter/http/handler_test.go
		GetPreferences: f.prefs, UpdatePreferences: fakeUpdatePreferences{f.prefs}})
````
````new server/internal/modules/project/adapter/http/handler_test.go
		GetPreferences: f.prefs, UpdatePreferences: fakeUpdatePreferences{f.prefs}, ListMembers: f.members})
````

`server/internal/modules/project/adapter/http/members_test.go`（新文件，69 行）：

````file server/internal/modules/project/adapter/http/members_test.go
package httpadapter_test

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeMembers is the members' use cases: each call is recorded as "caller
// id"; the list answers list, or err.
type fakeMembers struct {
	calls []string
	list  []domain.Member
	err   error
}

func (f *fakeMembers) Execute(ctx context.Context, projectID uuid.UUID) ([]domain.Member, error) {
	f.calls = append(f.calls, caller(ctx)+" "+projectID.String())
	return f.list, f.err
}

var (
	aliceInWeb = domain.Member{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000b1"), ProjectID: webID, MemberID: aliceID,
		Role: shared.RoleAdmin, CreatedAt: created}
	bobInWeb = domain.Member{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000b2"), ProjectID: webID, MemberID: bobID,
		Role: shared.RoleGuest, CreatedAt: created}
	membersJSON = `{"data":[{"created_at":"2026-10-01T10:00:00.123456Z","id":"0199a2b4-0000-7000-8000-0000000000b1",` +
		`"member_id":"0199a2b4-0000-7000-8000-000000000001","project_id":"0199a2b4-0000-7000-8000-0000000000a1","role":20},` +
		`{"created_at":"2026-10-01T10:00:00.123456Z","id":"0199a2b4-0000-7000-8000-0000000000b2",` +
		`"member_id":"0199a2b4-0000-7000-8000-000000000002","project_id":"0199a2b4-0000-7000-8000-0000000000a1","role":5}]}`
)

// GET goes to the use case for the caller and the path's project; the
// answer is 200 with its list, in its order, and [] for none.
func TestListProjectMembers(t *testing.T) {
	path := "/api/v0/projects/" + webID.String() + "/members"
	for _, tt := range []struct {
		list []domain.Member
		want string
	}{{[]domain.Member{aliceInWeb, bobInWeb}, membersJSON}, {nil, `{"data":[]}`}} {
		list := &fakeMembers{list: tt.list}
		h := newServer(t, fakes{members: list})
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
		{domain.ErrNotFound, http.StatusNotFound,
			`{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`},
		{shared.Forbidden(), http.StatusForbidden, `{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
	} {
		h := newServer(t, fakes{members: &fakeMembers{err: tt.err}})
		if res, body := do(t, h, request(http.MethodGet, path, "alice", "")); res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("GET = %d %s, want %d %s", res.StatusCode, body, tt.status, tt.want)
		}
	}
}
````

`server/internal/modules/project/module.go`（修改，2 处）：

````old server/internal/modules/project/module.go
// projects, checking an identifier and each member's display settings,
// carries out the workspace module's cascades on the projects
````
````new server/internal/modules/project/module.go
// projects, checking an identifier, listing the members and each member's
// display settings, carries out the workspace module's cascades on the projects
````

````old server/internal/modules/project/module.go
		UpdatePreferences: app.NewUpdateProjectPreferences(store, locks, d.Tx, d.Clock),
````
````new server/internal/modules/project/module.go
		UpdatePreferences: app.NewUpdateProjectPreferences(store, locks, d.Tx, d.Clock),
		ListMembers:       app.NewListProjectMembers(store, d.Authorizer),
````

- [ ] **Step 5: 矩阵：账户登记和成员列表的行**

`server/internal/bootstrap/permission_matrix_seeded_test.go`（修改，5 处）：

````old server/internal/bootstrap/permission_matrix_seeded_test.go
	"runtime"
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
	"runtime"
	"slices"
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
// slug and the address; and each project, by its key. t is the test that
// asks for them (in).
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
// slug and the address; each project, by its key; and each account, by
// its column, which prepareMatrix registers. t is the test that asks for
// them (in).
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
	projects    map[string]uuid.UUID
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
	projects    map[string]uuid.UUID
	accounts    map[caller]uuid.UUID
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
		projects: map[string]uuid.UUID{}}
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
		projects: map[string]uuid.UUID{}, accounts: map[caller]uuid.UUID{}}
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
}

// fatalOf runs f on a goroutine of its own with a testing.TB whose Fatalf
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
}

// account is the id of the account of matrixAccounts c, which prepareMatrix
// registers; uuid.Nil until then, when nothing is registered, so that a
// request built without a database names an account still (matrixViolations).
// An account that is no column's fails the test at once.
func (s seeded) account(c caller) uuid.UUID {
	if !slices.Contains(matrixAccounts, c) {
		s.t.Helper()
		s.t.Fatalf("no account %s is registered", c)
	}
	return s.accounts[c]
}

// fatalOf runs f on a goroutine of its own with a testing.TB whose Fatalf
````

`server/internal/bootstrap/permission_matrix_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_test.go
// matrixAccounts, registered for its token. Through the workspace store,
````
````new server/internal/bootstrap/permission_matrix_test.go
// matrixAccounts, registered for its token and its id. Through the workspace store,
````

````old server/internal/bootstrap/permission_matrix_test.go
			ids[c] = id
````
````new server/internal/bootstrap/permission_matrix_test.go
			ids[c] = id
			d.seeded.accounts[c] = id
````

`server/internal/bootstrap/permission_matrix_project_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_project_test.go
			cells:   ofProject(cellOK, cellOK, cellOK, cellOK, cellForbidden, cellForbidden), check: readsPreferences("cycles", `["views"]`)},
````
````new server/internal/bootstrap/permission_matrix_project_test.go
			cells:   ofProject(cellOK, cellOK, cellOK, cellOK, cellForbidden, cellForbidden), check: readsPreferences("cycles", `["views"]`)},
		{op: "listProjectMembers", columns: projectColumns, request: toProject(http.MethodGet, "/members", ""),
			cells: ofProject(cellOK, cellOK, cellOK, cellOK, cellForbidden, cellForbidden), check: listsTheProjectMembers},
````

````old server/internal/bootstrap/permission_matrix_project_test.go
			cells:   map[caller]cell{callerArchivedAdmin: cellOK}, check: readsPreferences("cycles", `["views"]`)},
````
````new server/internal/bootstrap/permission_matrix_project_test.go
			cells:   map[caller]cell{callerArchivedAdmin: cellOK}, check: readsPreferences("cycles", `["views"]`)},
	}
}

// listsTheProjectMembers: acme's public project's active members, each
// with his role, in the order prepareMatrix made them: PA, PM, the
// workspace's guest (PG's account), PM+WA and the removed member, whose
// membership of the project the stand-in left active (the list reads
// project_members alone, M3 design 5.2); not WG-, whose membership
// partingStates ended.
func listsTheProjectMembers(t *testing.T, c caller, s seeded, answer string) {
	var list struct {
		Data []struct {
			ProjectID uuid.UUID `json:"project_id"`
			MemberID  uuid.UUID `json:"member_id"`
			Role      int       `json:"role"`
		} `json:"data"`
	}
	decodeAnswer(t, answer, &list)
	want := []struct {
		c    caller
		role int
	}{{callerProjectAdmin, 20}, {callerProjectMember, 15}, {callerGuest, 5}, {callerMemberAndAdmin, 15}, {callerRemoved, 15}}
	ok := len(list.Data) == len(want)
	for i := 0; ok && i < len(want); i++ {
		m := list.Data[i]
		ok = m.ProjectID == s.project("acme/public") && m.MemberID == s.account(want[i].c) && m.Role == want[i].role
	}
	if !ok {
		t.Errorf("%s lists %s; want %+v of acme/public, in that order", c, answer, want)
````

- [ ] **Step 6: 测试、lint、前端检查**

Run: `go -C server test -count=1 ./internal/modules/project/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix$|TestThePermissionMatrixCoversEveryOperation|TestMatrixViolationsCatchesEach|TestEveryColumnCallsAsARegisteredAccount|TestEveryActionHasARuleAndEveryRuleAnAction|TestAPIRoutesAreTheContractsOperations|TestOperationsThatNeedATokenAnswer401WithoutOne' ./internal/bootstrap/`
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

- [ ] **Step 7: 提交**

```bash
git add api/modules/project.yaml api/openapi.yaml server/internal/bootstrap/permission_matrix_project_test.go server/internal/bootstrap/permission_matrix_seeded_test.go server/internal/bootstrap/permission_matrix_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/project/adapter/http/handler.go server/internal/modules/project/adapter/http/handler_test.go server/internal/modules/project/adapter/http/members.go server/internal/modules/project/adapter/http/members_test.go server/internal/modules/project/adapter/postgres/failures_test.go server/internal/modules/project/adapter/postgres/members.go server/internal/modules/project/adapter/postgres/members_test.go server/internal/modules/project/adapter/postgres/queries/members.sql server/internal/modules/project/app/list_members.go server/internal/modules/project/app/list_members_test.go server/internal/modules/project/app/ports.go server/internal/modules/project/domain/actions.go server/internal/modules/project/domain/member.go server/internal/modules/project/module.go api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/server.gen.go server/internal/modules/project/adapter/postgres/gen/members.sql.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P4b): listProjectMembers; the matrix registers its accounts

GET /api/v0/projects/{project_id}/members, for the project's active
members: its active undeleted memberships, by the time they were made,
then by id. A read decides without a transaction. The matrix registers
each column's account by its id, so that a row's body can name one.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| `ListMembers` 列别的项目的成员 | `TestListMembers`；P3（Task 15 起） | 存储；端到端 |
| `ListMembers` 列已结束的、已删除的 | `TestListMembers`（故事看不到：只有 P5 的移出和删除项目写得出这两种行，spec 第 3 节第 6 条） | 存储 |
| 规则给看得到项目的每个人；不给项目的访客 | `TestPermissionMatrix` | 组合 |
| `project.New` 给列表谁都当作管理员放行的 `Authorizer` | `TestPermissionMatrix` | 组合 |
| 没有调用者时也读 | `TestListProjectMembersRefuses` | 单元 |
| 判定、列表的失败被吞掉 | `TestListProjectMembersRefuses` | 单元 |
| handler 吞掉用例的失败 | `TestListProjectMembers`（HTTP） | 单元 |
| 添加的回答经连接池、在事务之外读列表 | `TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 14 起） | 组合 |

**Done when:** 矩阵一行（12 格）通过；列表只有有效、未删除的成员关系，顺序固定；矩阵能按列取账户的 id。

---

### Task 10: 成员关系的恢复与增长的领域、存储

**Files:**
- Create: `server/internal/modules/project/domain/member_test.go`
- Modify: `server/internal/modules/project/adapter/postgres/failures_test.go`、`server/internal/modules/project/adapter/postgres/members.go`、`server/internal/modules/project/adapter/postgres/members_test.go`、`server/internal/modules/project/adapter/postgres/preferences.go`、`server/internal/modules/project/adapter/postgres/preferences_test.go`、`server/internal/modules/project/adapter/postgres/queries/members.sql`、`server/internal/modules/project/adapter/postgres/queries/preferences.sql`、`server/internal/modules/project/app/ports.go`、`server/internal/modules/project/domain/member.go`
- Generate: `server/internal/modules/project/adapter/postgres/gen/members.sql.go`、`server/internal/modules/project/adapter/postgres/gen/preferences.sql.go`

**Interfaces:**
- Produces（spec 2.9，M3 设计 3.5、3.6 约定六、3.18、9.1）：`domain.NewMember{MemberID, Role}`、`MaxNewMembers = 100`、`CheckNewMembers([]NewMember) error`（1–100 个、角色是三种之一、每个账户只出现一次；全部问题一个 422，按位置）；`CanAdd(workspaceRole, role)`：工作区管理员只能作项目管理员，工作区访客只能作访客，工作区成员三种都可以（Plane `views/project/member.py:69-83`），按集合；`JoinRole(ended *Role, workspaceRole) Role`：新的成员关系取工作区角色，已结束的取它原来的角色与工作区角色中较低的一个，顺序按 `roleOrder`（访客、成员、管理员），不按数字。
- 端口：`app.MemberGrower`（`MembershipReader`、`CreateMember`、`RestoreMember`、`EnsurePreferences`；锁在 `Locks` 里）；存储：`RestoreMember(id, role, by, now)`：已结束的成员关系恢复为有效、取给的角色，`updated_by_id`、`updated_at`，id 和 `created_at` 保留；`EnsurePreferences(row)`：他在这个项目没有未删除的显示设置时插入（位置是给的，导航取列的默认值），有就不动（冲突目标是部分唯一键，已删除的不算）。
- 使用者：Task 11 的添加、Task 13 的加入（`growth`）。

**Tests:**
- `domain/member_test.go`：`TestCheckNewMembersAccepts`（一个、一百个，三种角色）；`TestCheckNewMembersReportsEveryProblem`（没有、一百零一个、三种以外的角色、同一个账户两次：一个 422 按位置）；`TestCanAdd`（三种工作区角色各自能加的项目角色；三种以外的角色什么都不能加，也不能被加为三种以外的角色）；`TestJoinRole`（M3 设计 9.1 表的四行和两个相等的情形）。
- `adapter/postgres/members_test.go`：`TestRestoreMember`（恢复为有效、取给的角色，由给的账户在给的时刻；id、`created_at` 不变，别的列不变；别的成员关系每一列不变）。`TestListMembers` 的准备随之扩充。
- `adapter/postgres/preferences_test.go`：`TestEnsurePreferences`（没有未删除的设置时插入，已删除的不算；有的照旧；别的行每一列不变）。`TestUpsertPreferences` 的准备随之扩充。
- `failures_test.go`：`RestoreMember`、`EnsurePreferences` 的失败是 `context.Canceled`。

- [ ] **Step 1: 查询**

`server/internal/modules/project/adapter/postgres/queries/members.sql`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/queries/members.sql
ORDER BY created_at, id;

````
````new server/internal/modules/project/adapter/postgres/queries/members.sql
ORDER BY created_at, id;

-- name: RestoreMember :exec
-- addProjectMembers and joinProject, under the project's FOR NO KEY UPDATE (M3 design 3.6 convention 6): an ended
-- membership active again, with the role the use case gives, at the moment and by the account given; it keeps its id
-- and its created_at.
UPDATE project_members
SET is_active = true, role = sqlc.arg(role), updated_by_id = sqlc.arg(updated_by)::uuid, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

````

`server/internal/modules/project/adapter/postgres/queries/preferences.sql`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/queries/preferences.sql
RETURNING preferences, sort_order;

````
````new server/internal/modules/project/adapter/postgres/queries/preferences.sql
RETURNING preferences, sort_order;

-- name: EnsurePreferences :exec
-- addProjectMembers and joinProject (M3 design 3.18): the account's display settings in the project, made with the
-- place given unless he has undeleted ones there already, which a restored membership keeps as they are. The conflict
-- target is the partial unique index, so a deleted row does not count.
INSERT INTO project_user_properties (id, workspace_id, project_id, user_id, sort_order, created_by_id, updated_by_id, created_at,
                                     updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(user_id), sqlc.arg(sort_order), sqlc.arg(created_by),
        sqlc.arg(created_by), sqlc.arg(now), sqlc.arg(now))
ON CONFLICT (project_id, user_id) WHERE deleted_at IS NULL DO NOTHING;

````

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `e85cb54f07295daa4821e7766732d55cfa41ef96d3cb0fde12c386da2210104a` | 158 | `server/internal/modules/project/adapter/postgres/gen/members.sql.go` |
| `775281de44b97b8019d08c1e4e6917a2610f4dc24fa559e5f79c3860626aee80` | 174 | `server/internal/modules/project/adapter/postgres/gen/preferences.sql.go` |

Run: `shasum -a 256 server/internal/modules/project/adapter/postgres/gen/members.sql.go server/internal/modules/project/adapter/postgres/gen/preferences.sql.go`
Expected: 与上表相同。

- [ ] **Step 2: 领域**

`server/internal/modules/project/domain/member.go`（修改，2 处）：

````old server/internal/modules/project/domain/member.go
import (
````
````new server/internal/modules/project/domain/member.go
import (
	"fmt"
	"slices"
````

````old server/internal/modules/project/domain/member.go
}

````
````new server/internal/modules/project/domain/member.go
}

// NewMember is an account to add to a project, with the role he is to
// have in it.
type NewMember struct {
	MemberID uuid.UUID
	Role     shared.Role
}

// MaxNewMembers is the most members one request adds (M3 design 5.1).
const MaxNewMembers = 100

// projectRoles are the three roles a member of a project can have.
var projectRoles = []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}

// CheckNewMembers checks what the request alone tells: 1 to MaxNewMembers
// members, each role one of the three, each account named once. Every
// problem is reported at once, as one 422 validation_failed; whether each
// account may be added is the use case's, under its locks (M3 design 3.6
// convention 3).
func CheckNewMembers(members []NewMember) error {
	var found []*shared.FieldError
	switch {
	case len(members) == 0:
		found = append(found, &shared.FieldError{Field: "members", Code: shared.FieldTooShort, Message: "must name a member"})
	case len(members) > MaxNewMembers:
		found = append(found, &shared.FieldError{Field: "members", Code: shared.FieldTooLong,
			Message: fmt.Sprintf("must name at most %d members", MaxNewMembers)})
	}
	for i, m := range members {
		if slices.ContainsFunc(members[:i], func(o NewMember) bool { return o.MemberID == m.MemberID }) {
			found = append(found, &shared.FieldError{Field: fmt.Sprintf("members[%d].member_id", i), Code: shared.FieldDuplicate,
				Message: "is listed before"})
		}
		if !slices.Contains(projectRoles, m.Role) {
			found = append(found, &shared.FieldError{Field: fmt.Sprintf("members[%d].role", i), Code: shared.FieldInvalidFormat,
				Message: "is not 5, 15 or 20"})
		}
	}
	return invalid(found...)
}

// addable are the project roles an active member of the workspace can be
// added to a project with, by his workspace role (M3 design 3.5; Plane
// views/project/member.py:69-83): a workspace admin as an admin alone, a
// workspace guest as a guest alone, a workspace member as any of the three.
var addable = map[shared.Role][]shared.Role{
	shared.RoleAdmin:  {shared.RoleAdmin},
	shared.RoleMember: projectRoles,
	shared.RoleGuest:  {shared.RoleGuest},
}

// CanAdd reports whether an active member of the workspace of workspace
// role workspaceRole may be added to a project as role. The sets are
// named, not bounds.
func CanAdd(workspaceRole, role shared.Role) bool {
	return slices.Contains(addable[workspaceRole], role)
}

// roleOrder is the project roles from the least to the most.
var roleOrder = []shared.Role{shared.RoleGuest, shared.RoleMember, shared.RoleAdmin}

// JoinRole is the project role of an account who joins a project, of
// workspace role workspaceRole (M3 design 3.5, 3.6 convention 6): his
// workspace role for a new membership; for his ended one, ended, the
// lesser of its role and his workspace role, so that a restored membership
// gives no more than a new one would, nor more than it had. The order is
// roleOrder's, not the numbers'.
func JoinRole(ended *shared.Role, workspaceRole shared.Role) shared.Role {
	if ended == nil || slices.Index(roleOrder, workspaceRole) < slices.Index(roleOrder, *ended) {
		return workspaceRole
	}
	return *ended
}

````

`server/internal/modules/project/domain/member_test.go`（新文件，107 行）：

````file server/internal/modules/project/domain/member_test.go
package domain

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// One member, and a hundred, each of the three roles: accepted.
func TestCheckNewMembersAccepts(t *testing.T) {
	hundred := make([]NewMember, MaxNewMembers)
	for i := range hundred {
		hundred[i] = NewMember{MemberID: uuid.NewV7(), Role: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}[i%3]}
	}
	for _, in := range [][]NewMember{{{MemberID: uuid.NewV7(), Role: shared.RoleGuest}}, hundred} {
		if err := CheckNewMembers(in); err != nil {
			t.Errorf("CheckNewMembers(%d members) = %v, want nil", len(in), err)
		}
	}
}

// None, a hundred and one, a role outside the three, an account named
// twice: each refused, all of them in one 422 that names each by its place
// in the request.
func TestCheckNewMembersReportsEveryProblem(t *testing.T) {
	a, b := uuid.NewV7(), uuid.NewV7()
	many := make([]NewMember, MaxNewMembers+1)
	for i := range many {
		many[i] = NewMember{MemberID: uuid.NewV7(), Role: shared.RoleMember}
	}
	tests := []struct {
		name string
		in   []NewMember
		want []shared.FieldError
	}{
		{"none", nil, []shared.FieldError{{Field: "members", Code: "too_short", Message: "must name a member"}}},
		{"a hundred and one", many, []shared.FieldError{{Field: "members", Code: "too_long", Message: "must name at most 100 members"}}},
		{"roles and repeats", []NewMember{{MemberID: a, Role: 10}, {MemberID: b, Role: shared.RoleAdmin}, {MemberID: a, Role: shared.RoleGuest},
			{MemberID: b, Role: 0}}, []shared.FieldError{
			{Field: "members[0].role", Code: "invalid_format", Message: "is not 5, 15 or 20"},
			{Field: "members[2].member_id", Code: "duplicate", Message: "is listed before"},
			{Field: "members[3].member_id", Code: "duplicate", Message: "is listed before"},
			{Field: "members[3].role", Code: "invalid_format", Message: "is not 5, 15 or 20"},
		}},
	}
	for _, tt := range tests {
		var e *shared.Error
		if err := CheckNewMembers(tt.in); !errors.As(err, &e) || e.Code != "validation_failed" || !reflect.DeepEqual(e.Fields, tt.want) {
			t.Errorf("%s: CheckNewMembers() = %v; want validation_failed with %+v", tt.name, err, tt.want)
		}
	}
}

// A workspace admin is added as an admin alone, a workspace guest as a
// guest alone, a workspace member as any of the three; nobody of a role
// outside the three, as nothing outside them (M3 design 3.5).
func TestCanAdd(t *testing.T) {
	roles := []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest, 10, 25}
	allowed := map[[2]shared.Role]bool{
		{shared.RoleAdmin, shared.RoleAdmin}: true, {shared.RoleMember, shared.RoleAdmin}: true, {shared.RoleMember, shared.RoleMember}: true,
		{shared.RoleMember, shared.RoleGuest}: true, {shared.RoleGuest, shared.RoleGuest}: true,
	}
	for _, ws := range roles {
		for _, role := range roles {
			if got := CanAdd(ws, role); got != allowed[[2]shared.Role{ws, role}] {
				t.Errorf("CanAdd(%d, %d) = %v, want %v", ws, role, got, !got)
			}
		}
	}
}

// A new membership takes the workspace role; an ended one the lesser of
// its role and the workspace role (M3 design 9.1's table, and the two
// equal cases).
func TestJoinRole(t *testing.T) {
	role := func(r shared.Role) *shared.Role { return &r }
	for _, tt := range []struct {
		ended     *shared.Role
		workspace shared.Role
		want      shared.Role
	}{
		{nil, shared.RoleMember, shared.RoleMember},
		{nil, shared.RoleAdmin, shared.RoleAdmin},
		{role(shared.RoleGuest), shared.RoleMember, shared.RoleGuest},
		{role(shared.RoleAdmin), shared.RoleMember, shared.RoleMember},
		{role(shared.RoleMember), shared.RoleAdmin, shared.RoleMember},
		{role(shared.RoleAdmin), shared.RoleGuest, shared.RoleGuest},
		{role(shared.RoleMember), shared.RoleMember, shared.RoleMember},
		{role(shared.RoleAdmin), shared.RoleAdmin, shared.RoleAdmin},
	} {
		if got := JoinRole(tt.ended, tt.workspace); got != tt.want {
			t.Errorf("JoinRole(%s, %d) = %d, want %d", show(tt.ended), tt.workspace, got, tt.want)
		}
	}
}

// show is r as a failure prints it.
func show(r *shared.Role) string {
	if r == nil {
		return "none"
	}
	return fmt.Sprint(*r)
}
````

- [ ] **Step 3: 端口和存储**

`server/internal/modules/project/app/ports.go`（修改，1 处）：

````old server/internal/modules/project/app/ports.go
}

// ProjectArchiver is archiveProject's and unarchiveProject's repository.
````
````new server/internal/modules/project/app/ports.go
}

// MemberGrower writes a project's new and restored memberships, each with
// its member's display settings (M3 design 3.6 convention 6, 3.18): the
// project side's growth, which addProjectMembers and joinProject share.
// Each method runs in the transaction ctx carries, under the project's
// FOR NO KEY UPDATE.
type MemberGrower interface {
	MembershipReader
	CreateMember(ctx context.Context, m MemberRow) error
	// RestoreMember makes the ended membership id active again with role,
	// by the account by at now.
	RestoreMember(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) error
	// EnsurePreferences inserts p unless its account has undeleted display
	// settings in its project already, which stay as they are.
	EnsurePreferences(ctx context.Context, p PreferencesRow) error
}

// ProjectArchiver is archiveProject's and unarchiveProject's repository.
````

`server/internal/modules/project/adapter/postgres/members.go`（修改，2 处）：

````old server/internal/modules/project/adapter/postgres/members.go
	"fmt"
````
````new server/internal/modules/project/adapter/postgres/members.go
	"fmt"
	"time"
````

````old server/internal/modules/project/adapter/postgres/members.go
}

// ListMembers lists projectID's active memberships, in the order they were
````
````new server/internal/modules/project/adapter/postgres/members.go
}

// RestoreMember makes the ended membership id active again with role, by
// the account by at now; it keeps its id and created_at
// (app.MemberGrower).
func (s *Store) RestoreMember(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) error {
	if err := s.queries(ctx).RestoreMember(ctx, gen.RestoreMemberParams{ID: id, Role: int16(role), UpdatedBy: by, Now: now}); err != nil {
		return fmt.Errorf("restore project membership %s: %w", id, err)
	}
	return nil
}

// ListMembers lists projectID's active memberships, in the order they were
````

`server/internal/modules/project/adapter/postgres/preferences.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/preferences.go
}

// preferences are the settings a row holds.
````
````new server/internal/modules/project/adapter/postgres/preferences.go
}

// EnsurePreferences inserts p unless its account has undeleted display
// settings in its project already, which stay as they are
// (app.MemberGrower).
func (s *Store) EnsurePreferences(ctx context.Context, p app.PreferencesRow) error {
	err := s.queries(ctx).EnsurePreferences(ctx, gen.EnsurePreferencesParams{
		ID: p.ID, WorkspaceID: p.WorkspaceID, ProjectID: p.ProjectID, UserID: p.UserID, SortOrder: p.SortOrder, CreatedBy: &p.CreatedBy, Now: p.Now,
	})
	if err != nil {
		return fmt.Errorf("ensure project preferences: %w", err)
	}
	return nil
}

// preferences are the settings a row holds.
````

`server/internal/modules/project/adapter/postgres/members_test.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/members_test.go
		t.Errorf("ListMembers() of no project = %v, %v; want none", got, err)
	}
}

````
````new server/internal/modules/project/adapter/postgres/members_test.go
		t.Errorf("ListMembers() of no project = %v, %v; want none", got, err)
	}
}

// RestoreMember makes the ended membership active again with the role
// given, at the moment and by the account given, and changes no other
// column of it, its id and created_at kept; every other membership keeps
// every column.
func TestRestoreMember(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	seedMember(t, pool, acme, ops, bob, 20, false)
	ended := seedMember(t, pool, acme, web, bob, 20, false)
	seedMember(t, pool, acme, web, alice, 15, true)
	others := tableRows(t, pool, "project_members", ended)
	before := columns(t, pool, "project_members", ended)
	later := now.Add(time.Hour)

	if err := s.RestoreMember(context.Background(), ended, shared.RoleGuest, alice, later); err != nil {
		t.Fatal(err)
	}

	want := changed(before, map[string]string{"is_active": "true", "role": "5", "updated_by_id": `"` + alice.String() + `"`,
		"updated_at": `"` + later.Format("2006-01-02T15:04:05.999999") + `+00:00"`})
	if got := columns(t, pool, "project_members", ended); !maps.Equal(got, want) {
		t.Errorf("the restored membership: %v\nwant %v", got, want)
	}
	if after := tableRows(t, pool, "project_members", ended); after != others {
		t.Errorf("the other memberships:\n%s\nwant\n%s", after, others)
	}
}

````

`server/internal/modules/project/adapter/postgres/preferences_test.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/preferences_test.go
		t.Errorf("the other rows:\n%s\nwant\n%s", after, others)
	}
}

````
````new server/internal/modules/project/adapter/postgres/preferences_test.go
		t.Errorf("the other rows:\n%s\nwant\n%s", after, others)
	}
}

// EnsurePreferences inserts the account's display settings at the place
// given, made by the account given at the moment given, the navigation the
// column's default, while he has no undeleted ones in the project, a
// deleted one not counting; undeleted ones stay as they are, and so does
// every other row.
func TestEnsurePreferences(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	bobs := preferencesRow(t, pool, acme, web, bob)
	exec(t, pool, `UPDATE project_user_properties SET preferences = '{"navigation": {"default_tab": "views", "hide_in_more_menu": []}}',
		sort_order = 7 WHERE id = $1`, bobs)
	exec(t, pool, "UPDATE project_user_properties SET deleted_at = $2 WHERE id = $1", preferencesRow(t, pool, acme, web, carol), now)
	preferencesRow(t, pool, acme, ops, carol)
	ensure := func(user uuid.UUID, at time.Time) uuid.UUID {
		t.Helper()
		id := uuid.NewV7()
		if err := s.EnsurePreferences(context.Background(), app.PreferencesRow{ID: id, WorkspaceID: acme, ProjectID: web, UserID: user, SortOrder: -3,
			CreatedBy: alice, Now: at}); err != nil {
			t.Fatal(err)
		}
		return id
	}
	before := tableRows(t, pool, "project_user_properties", uuid.Nil())

	ensure(bob, now.Add(time.Hour))
	if after := tableRows(t, pool, "project_user_properties", uuid.Nil()); after != before {
		t.Errorf("after bob's, who has his settings:\n%s\nwant\n%s", after, before)
	}
	carols := ensure(carol, now.Add(time.Hour))
	stamp := `"` + now.Add(time.Hour).Format("2006-01-02T15:04:05.999999") + `+00:00"`
	got := columns(t, pool, "project_user_properties", carols)
	for k, want := range map[string]string{"sort_order": "-3", "project_id": `"` + web.String() + `"`, "user_id": `"` + carol.String() + `"`,
		"created_by_id": `"` + alice.String() + `"`, "updated_by_id": `"` + alice.String() + `"`, "created_at": stamp, "updated_at": stamp,
		"deleted_at": "null", "preferences": `{"navigation": {"default_tab": "work_items", "hide_in_more_menu": []}}`} {
		if got[k] != want {
			t.Errorf("carol's new settings' %s = %s, want %s", k, got[k], want)
		}
	}
	if after := tableRows(t, pool, "project_user_properties", carols); after != before {
		t.Errorf("the other rows after carol's:\n%s\nwant\n%s", after, before)
	}
}

````

`server/internal/modules/project/adapter/postgres/failures_test.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("SetArchived() = %v; want context.Canceled", err)
	}
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("SetArchived() = %v; want context.Canceled", err)
	}
	if err := s.RestoreMember(cancelled, uuid.NewV7(), shared.RoleMember, alice, now); !failed(err) {
		t.Errorf("RestoreMember() = %v; want context.Canceled", err)
	}
	if err := s.EnsurePreferences(cancelled, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, UserID: alice,
		SortOrder: 1, CreatedBy: alice, Now: now}); !failed(err) {
		t.Errorf("EnsurePreferences() = %v; want context.Canceled", err)
	}
````

- [ ] **Step 4: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/project/...`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 5: 提交**

```bash
git add server/internal/modules/project/adapter/postgres/failures_test.go server/internal/modules/project/adapter/postgres/members.go server/internal/modules/project/adapter/postgres/members_test.go server/internal/modules/project/adapter/postgres/preferences.go server/internal/modules/project/adapter/postgres/preferences_test.go server/internal/modules/project/adapter/postgres/queries/members.sql server/internal/modules/project/adapter/postgres/queries/preferences.sql server/internal/modules/project/app/ports.go server/internal/modules/project/domain/member.go server/internal/modules/project/domain/member_test.go server/internal/modules/project/adapter/postgres/gen/members.sql.go server/internal/modules/project/adapter/postgres/gen/preferences.sql.go
```
```bash
git commit -m "feat(M3/P4b): restoring a membership, and the growth's domain and store

CheckNewMembers holds a request to 1-100 accounts, each once, each with
one of the three roles; CanAdd names the project roles each workspace
role may be added with, JoinRole the lesser of an ended membership's
role and the workspace role, by the roles' order. The store restores an
ended membership with a role, keeping its id and its time, and inserts
display settings unless the account has undeleted ones.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| `RestoreMember` 改每个成员关系（去掉 id） | `TestRestoreMember`（故事看不到：故事里没有已结束的成员关系，spec 第 3 节第 6 条） | 存储 |
| `RestoreMember` 也写 `created_at`、`created_by_id`；保留原来的角色；仍是已结束的 | `TestRestoreMember` | 存储 |
| `EnsurePreferences` 的冲突目标没有 `WHERE deleted_at IS NULL` | `TestEnsurePreferences`；P2、P3、P4、P8（Task 15 起） | 存储；端到端 |
| `EnsurePreferences` 覆盖已有的设置；行的最后修改者记成建行的人 | `TestEnsurePreferences` | 存储 |
| `JoinRole` 取较高的、取工作区角色、保留已结束的角色 | `TestJoinRole`；`TestARestoredMembershipGivesNoMoreThanItHad`（Task 13 起） | 单元；组合 |
| `CanAdd` 让工作区访客作成员、让工作区管理员作成员 | `TestCanAdd`；`TestPermissionMatrix`（Task 12 起，添加的两个变体行） | 单元；组合 |
| `RestoreMember`、`EnsurePreferences` 经连接池、在事务之外执行 | `TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 14 起） | 组合 |

**Done when:** 领域的四个测试、存储的两个测试通过；恢复保留成员关系的 id 和建立时刻；角色的规则都按集合或顺序表。

---

### Task 11: `addProjectMembers` 的用例；增长的一步

**Files:**
- Create: `server/internal/modules/project/app/add_members.go`、`server/internal/modules/project/app/add_members_test.go`、`server/internal/modules/project/app/fakes_growth_test.go`、`server/internal/modules/project/app/growth.go`
- Modify: `server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/project/app/clock_test.go`、`server/internal/modules/project/app/fakes_write_test.go`、`server/internal/modules/project/app/list_members_test.go`、`server/internal/modules/project/app/lock.go`、`server/internal/modules/project/app/ports.go`、`server/internal/modules/project/domain/actions.go`、`server/internal/modules/project/domain/member.go`、`server/internal/modules/project/domain/member_test.go`、`server/internal/modules/project/module.go`

**Interfaces:**
- Produces（spec 2.9，M3 设计 3.5、3.6 约定三和六、3.18、9.2）：`domain.Target{NewMember, WorkspaceRole *Role, Member bool}`、`CheckTargets([]Target) error`：每个目标对照锁下读到的：是工作区的有效成员（否则 `members[i].member_id` `not_allowed`）、还不是项目的有效成员（否则 `duplicate`）、角色是他的工作区角色允许的（`CanAdd`，否则 `members[i].role` `not_allowed`）；一个 422 按位置。
- `app/growth.go`：`growth`（一个账户成为项目成员：已结束的成员关系以 `growth.role` 恢复，否则新建；然后显示设置，已有的不动）与 `endedOf`，Task 13 共用。
- `app/lock.go`：`Locks` 加 `members WorkspaceMembers`（`NewLocks(projects, workspaces, members, auth)`，`project.New` 传 `d.Members`）；`write` 加 `targets`（写使之成为项目成员的账户），`held` 加 `roles`（他们在工作区的有效角色）：有目标的写在工作区 `FOR SHARE` 之后、项目之前经 `ShareMembers` 锁他们在工作区的成员关系（`FOR SHARE`，按 id 的顺序，M3 设计 3.6 约定三），锁之下读到的角色随 `held` 返回。
- `app.AddMembersDeps{Locks, Projects MemberAdder, Tx, Clock}`、`app.NewAddProjectMembers(d)`：`CheckNewMembers`（事务之前）→ 一个事务：`Locks.lockAndDecide(write{project, project_member.add, targets})`（不加锁读项目的工作区 → 工作区 `FOR SHARE` → 目标在工作区的成员关系 → 项目 `FOR NO KEY UPDATE` → 判定）→ `Memberships` → `CheckTargets`（判定之后：不能添加的人得不到目标的任何信息）→ 时钟 → 每个目标按请求的顺序：`LowestSortOrder`、`growth.apply`（恢复时取请求的角色，显示设置在他侧边栏的最前，`SortOrderFirst`）→ 回答经 `ListMembers` 读回，按请求的顺序。整批要么都写，要么都不写。
- `app.SortOrderReader` 从 `ProjectCreator` 拆出（`LowestSortOrder`），`MemberAdder` = `MemberGrower` + `MemberLister` + `SortOrderReader`。
- 操作名 `project_member.add`，规则同 `project.update`（项目管理员，和是工作区管理员的项目成员）。

**Tests:**
- `domain/member_test.go`：`TestCheckTargets`（不是工作区成员、已是项目成员、工作区角色不允许的角色：各按位置一个 422，已是成员的不看角色；别的通过）。`TestCanAdd` 随之扩充。
- `app/add_members_test.go`（假实现 `fakes_growth_test.go`：`newGrowth` 是 `newWrites` 加上工作区成员的角色，`fakeMembers` 进 `writeFixture`）：`TestAddProjectMembers`（完整的调用记录：项目的工作区、工作区的锁、目标的锁、项目的锁、判定、成员关系、时钟，再逐个目标：最低位置、恢复（dave，以前是成员，现在作管理员）或新建、显示设置在他最低位置之前 10000，没有时取默认值；回答按请求的顺序，恢复的保留 id）；`TestAddProjectMembersRefuses`（请求的值、没有调用者在事务之前；没有的项目在找工作区时 404；看不到、成员的 403 在判定，无论目标如何（矩阵的 PM、X 格）；判定之后每个目标的拒绝一个 422：erin 不是工作区成员，alice 已是项目成员，工作区管理员 gina 作成员，工作区访客 ivy 作成员；hank 作管理员通过；什么都不写）；`TestAddProjectMembersReturnsEachFailure`（每个端口、提交的失败原样返回，之前的调用都在、之后的都没有；写完读不到的成员关系是内部错误）；`TestAddProjectMembersToAnArchivedProject`。
- `app/clock_test.go`：`TestEachWriteReadsTheClockUnderItsLock` 加添加一行。

- [ ] **Step 1: 领域**

`server/internal/modules/project/domain/member.go`（修改，1 处）：

````old server/internal/modules/project/domain/member.go
}

// roleOrder is the project roles from the least to the most.
````
````new server/internal/modules/project/domain/member.go
}

// Target is an account a request adds to a project, as the use case reads
// him under its locks: his role in the workspace, nil while he is not its
// active member, and whether he is an active member of the project.
type Target struct {
	NewMember
	WorkspaceRole *shared.Role
	Member        bool
}

// CheckTargets checks each account to add against what the use case read
// under its locks, after the decision (M3 design 3.5, 3.6 convention 3):
// an active member of the workspace, not an active member of the project
// already, added with a role his workspace role allows (CanAdd). One 422
// names each one refused, by his place in the request.
func CheckTargets(targets []Target) error {
	var found []*shared.FieldError
	for i, t := range targets {
		switch {
		case t.WorkspaceRole == nil:
			found = append(found, &shared.FieldError{Field: fmt.Sprintf("members[%d].member_id", i), Code: shared.FieldNotAllowed,
				Message: "must be an active member of the workspace"})
		case t.Member:
			found = append(found, &shared.FieldError{Field: fmt.Sprintf("members[%d].member_id", i), Code: shared.FieldDuplicate,
				Message: "is an active member of the project already"})
		case !CanAdd(*t.WorkspaceRole, t.Role):
			found = append(found, &shared.FieldError{Field: fmt.Sprintf("members[%d].role", i), Code: shared.FieldNotAllowed,
				Message: "is not one his workspace role allows: a workspace admin joins as an admin, a guest as a guest"})
		}
	}
	return invalid(found...)
}

// roleOrder is the project roles from the least to the most.
````

`server/internal/modules/project/domain/member_test.go`（修改，1 处）：

````old server/internal/modules/project/domain/member_test.go
}

// A new membership takes the workspace role; an ended one the lesser of
````
````new server/internal/modules/project/domain/member_test.go
}

// Each target is held to what was read of him: no workspace role, a
// member already, a role his workspace role does not allow; each refused
// by his place, in one 422, a member's role not looked at; the others
// pass.
func TestCheckTargets(t *testing.T) {
	role := func(r shared.Role) *shared.Role { return &r }
	target := func(r shared.Role, ws *shared.Role, member bool) Target {
		return Target{NewMember: NewMember{MemberID: uuid.NewV7(), Role: r}, WorkspaceRole: ws, Member: member}
	}
	if err := CheckTargets([]Target{target(shared.RoleAdmin, role(shared.RoleAdmin), false), target(shared.RoleGuest, role(shared.RoleMember), false),
		target(shared.RoleGuest, role(shared.RoleGuest), false)}); err != nil {
		t.Errorf("CheckTargets() of allowed targets = %v, want nil", err)
	}
	roleProblem := "is not one his workspace role allows: a workspace admin joins as an admin, a guest as a guest"
	want := []shared.FieldError{
		{Field: "members[0].member_id", Code: "not_allowed", Message: "must be an active member of the workspace"},
		{Field: "members[1].member_id", Code: "duplicate", Message: "is an active member of the project already"},
		{Field: "members[2].role", Code: "not_allowed", Message: roleProblem},
		{Field: "members[4].role", Code: "not_allowed", Message: roleProblem},
	}
	err := CheckTargets([]Target{target(shared.RoleMember, nil, false), target(shared.RoleAdmin, role(shared.RoleGuest), true),
		target(shared.RoleMember, role(shared.RoleAdmin), false), target(shared.RoleAdmin, role(shared.RoleMember), false),
		target(shared.RoleMember, role(shared.RoleGuest), false)})
	var e *shared.Error
	if !errors.As(err, &e) || e.Code != "validation_failed" || !reflect.DeepEqual(e.Fields, want) {
		t.Errorf("CheckTargets() = %v; want validation_failed with %+v", err, want)
	}
}

// A new membership takes the workspace role; an ended one the lesser of
````

- [ ] **Step 2: 操作名和规则**

`server/internal/modules/project/domain/actions.go`（修改，2 处）：

````old server/internal/modules/project/domain/actions.go
	ActionMemberList shared.Action = "project_member.list"
````
````new server/internal/modules/project/domain/actions.go
	ActionMemberList shared.Action = "project_member.list"
	// ActionMemberAdd is adding members to a project: addProjectMembers.
	ActionMemberAdd shared.Action = "project_member.add"
````

````old server/internal/modules/project/domain/actions.go
		ActionMemberList}
````
````new server/internal/modules/project/domain/actions.go
		ActionMemberList, ActionMemberAdd}
````

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"project_member.list": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
````
````new server/internal/modules/access/domain/rules.go
	"project_member.list": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// The project's admins, and its members who are the workspace's admins
	// (M3 design 3.5, 9.2; Plane views/project/member.py:46).
	"project_member.add": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"project_member.list": {allowed, allowed, allowed, allowed, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
````
````new server/internal/modules/access/domain/rules_test.go
	"project_member.list": {allowed, allowed, allowed, allowed, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
	"project_member.add": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
````

- [ ] **Step 3: 端口、增长的一步、用例**

`server/internal/modules/project/app/ports.go`（修改，3 处）：

````old server/internal/modules/project/app/ports.go
type ProjectCreator interface {
	ProjectReader
	CreateProject(ctx context.Context, p ProjectRow) error
	CreateMember(ctx context.Context, m MemberRow) error
````
````new server/internal/modules/project/app/ports.go
type ProjectCreator interface {
	ProjectReader
	SortOrderReader
	CreateProject(ctx context.Context, p ProjectRow) error
	CreateMember(ctx context.Context, m MemberRow) error
	CreatePreferences(ctx context.Context, p PreferencesRow) error
	CreateStates(ctx context.Context, rows []StateRow) error
}

// SortOrderReader reads an account's places in his sidebar, for the place
// of a project he is made a member of (domain.SortOrderFirst).
type SortOrderReader interface {
````

````old server/internal/modules/project/app/ports.go
	LowestSortOrder(ctx context.Context, workspaceID, userID uuid.UUID) (*float64, error)
	CreatePreferences(ctx context.Context, p PreferencesRow) error
	CreateStates(ctx context.Context, rows []StateRow) error
````
````new server/internal/modules/project/app/ports.go
	LowestSortOrder(ctx context.Context, workspaceID, userID uuid.UUID) (*float64, error)
````

````old server/internal/modules/project/app/ports.go
}

// ProjectArchiver is archiveProject's and unarchiveProject's repository.
````
````new server/internal/modules/project/app/ports.go
}

// MemberAdder is addProjectMembers' repository.
type MemberAdder interface {
	MemberGrower
	MemberLister
	SortOrderReader
}

// ProjectArchiver is archiveProject's and unarchiveProject's repository.
````

`server/internal/modules/project/app/lock.go`（修改，7 处）：

````old server/internal/modules/project/app/lock.go
// workspace's row FOR SHARE, the write's first lock; the project's row, FOR
// NO KEY UPDATE, or FOR SHARE for a write under the project that leaves
// the row and its memberships as they are, still undeleted and of that
// workspace; then the decision, under them all. Every cascade over the
// workspace's projects runs under the workspace's FOR NO KEY UPDATE and
// reads its time after it (3.3): while a write holds the workspace FOR
// SHARE, no cascade touches the rows it writes. project.New builds one
// Locks for every write: a write holds no Authorizer of its own, so it
// decides only under these locks.
````
````new server/internal/modules/project/app/lock.go
// workspace's row FOR SHARE, the write's first lock; the memberships of the
// workspace of the accounts the write makes members of the project, FOR
// SHARE in id order (convention 3); the project's row, FOR NO KEY UPDATE,
// or FOR SHARE for a write under the project that leaves the row and its
// memberships as they are, still undeleted and of that workspace; then the
// decision, under them all. Every cascade over the workspace's projects
// runs under the workspace's FOR NO KEY UPDATE and reads its time after it
// (3.3): while a write holds the workspace FOR SHARE, no cascade touches
// the rows it writes. project.New builds one Locks for every write: a
// write holds no Authorizer of its own, so it decides only under these
// locks.
````

````old server/internal/modules/project/app/lock.go
	workspaces WorkspaceSharer
````
````new server/internal/modules/project/app/lock.go
	workspaces WorkspaceSharer
	members    WorkspaceMembers
````

````old server/internal/modules/project/app/lock.go
// module's directory and the Authorizer.
func NewLocks(projects ProjectLocks, workspaces WorkspaceSharer, auth shared.Authorizer) Locks {
	return Locks{projects: projects, workspaces: workspaces, auth: auth}
````
````new server/internal/modules/project/app/lock.go
// module's directory and members' lock, and the Authorizer.
func NewLocks(projects ProjectLocks, workspaces WorkspaceSharer, members WorkspaceMembers, auth shared.Authorizer) Locks {
	return Locks{projects: projects, workspaces: workspaces, members: members, auth: auth}
````

````old server/internal/modules/project/app/lock.go
	share bool
````
````new server/internal/modules/project/app/lock.go
	share bool
	// targets are the accounts the write makes members of the project.
	targets []uuid.UUID
````

````old server/internal/modules/project/app/lock.go
// lock read it and the caller's grant.
````
````new server/internal/modules/project/app/lock.go
// lock read it, the caller's grant, and the active roles in the workspace
// of the write's targets, by account.
````

````old server/internal/modules/project/app/lock.go
	grant   shared.Grant
````
````new server/internal/modules/project/app/lock.go
	grant   shared.Grant
	roles   map[uuid.UUID]shared.Role
````

````old server/internal/modules/project/app/lock.go
	var h held
````
````new server/internal/modules/project/app/lock.go
	var h held
	if len(w.targets) > 0 {
		if h.roles, err = l.members.ShareMembers(ctx, workspaceID, w.targets); err != nil {
			return held{}, err
		}
	}
````

`server/internal/modules/project/module.go`（修改，1 处）：

````old server/internal/modules/project/module.go
	locks := app.NewLocks(store, d.Workspaces, d.Authorizer)
````
````new server/internal/modules/project/module.go
	locks := app.NewLocks(store, d.Workspaces, d.Members, d.Authorizer)
````

`server/internal/modules/project/app/growth.go`（新文件，47 行）：

````file server/internal/modules/project/app/growth.go
package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// growth is one account made a member of a project, by addProjectMembers
// or joinProject, under the project's FOR NO KEY UPDATE and his workspace
// membership's FOR SHARE (M3 design 3.6 conventions 3 and 6).
type growth struct {
	workspaceID, projectID, user uuid.UUID
	// ended is his ended membership of the project, nil when he has none.
	ended     *Membership
	role      shared.Role
	sortOrder float64 // the project's place in his sidebar, if he has none there
	by        uuid.UUID
	now       time.Time
}

// apply writes g: his ended membership restored with g's role, or a new
// one; then his display settings in the project at g's place, unless he
// has undeleted ones there, which a restored membership keeps.
func (g growth) apply(ctx context.Context, p MemberGrower) error {
	if g.ended != nil {
		if err := p.RestoreMember(ctx, g.ended.ID, g.role, g.by, g.now); err != nil {
			return err
		}
	} else if err := p.CreateMember(ctx, MemberRow{ID: uuid.NewV7(), WorkspaceID: g.workspaceID, ProjectID: g.projectID, MemberID: g.user,
		Role: g.role, CreatedBy: g.by, Now: g.now}); err != nil {
		return err
	}
	return p.EnsurePreferences(ctx, PreferencesRow{ID: uuid.NewV7(), WorkspaceID: g.workspaceID, ProjectID: g.projectID, UserID: g.user,
		SortOrder: g.sortOrder, CreatedBy: g.by, Now: g.now})
}

// endedOf is the ended membership of memberships' account user, nil when
// he has none or his is active.
func endedOf(memberships map[uuid.UUID]Membership, user uuid.UUID) *Membership {
	if m, ok := memberships[user]; ok && !m.Active {
		return &m
	}
	return nil
}
````

`server/internal/modules/project/app/add_members.go`（新文件，114 行）：

````file server/internal/modules/project/app/add_members.go
package app

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// AddMembersDeps are addProjectMembers' ports.
type AddMembersDeps struct {
	Locks    Locks
	Projects MemberAdder
	Tx       shared.TxManager
	Clock    Clock
}

// AddProjectMembers adds workspace members to a project: POST
// /api/v0/projects/{project_id}/members.
type AddProjectMembers struct {
	d AddMembersDeps
}

// NewAddProjectMembers returns the use case.
func NewAddProjectMembers(d AddMembersDeps) *AddProjectMembers {
	return &AddProjectMembers{d: d}
}

// Execute checks the request, then in one transaction, in the order of M3
// design 3.6: the project's locks (Locks: its workspace FOR SHARE; the
// targets' memberships of the workspace FOR SHARE, which come before the
// project in the lock order, convention 3; the project FOR NO KEY UPDATE)
// and the decision on project_member.add; the targets' memberships of the
// project; the targets' check (domain.CheckTargets), after the decision, so
// that who may not add learns nothing of them; the clock; then each target,
// in the request's order: an ended membership restored with the role asked
// for, as the admin decides it now, or a new one; his display settings
// unless he has them, before his other projects in his sidebar (3.18). The
// answer is the targets' memberships as stored, in the request's order.
func (u *AddProjectMembers) Execute(ctx context.Context, projectID uuid.UUID, in []domain.NewMember) ([]domain.Member, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	if err := domain.CheckNewMembers(in); err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(in))
	for i, m := range in {
		ids[i] = m.MemberID
	}
	var added []domain.Member
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		h, err := u.d.Locks.lockAndDecide(ctx, actor, write{project: projectID, action: domain.ActionMemberAdd, targets: ids})
		if err != nil {
			return err
		}
		memberships, err := u.d.Projects.Memberships(ctx, projectID, ids)
		if err != nil {
			return err
		}
		targets := make([]domain.Target, len(in))
		for i, m := range in {
			targets[i] = domain.Target{NewMember: m, Member: memberships[m.MemberID].Active}
			if role, ok := h.roles[m.MemberID]; ok {
				targets[i].WorkspaceRole = &role
			}
		}
		if err := domain.CheckTargets(targets); err != nil {
			return err
		}
		now := u.d.Clock.Now()
		for _, m := range in {
			lowest, err := u.d.Projects.LowestSortOrder(ctx, h.project.WorkspaceID, m.MemberID)
			if err != nil {
				return err
			}
			if err := (growth{workspaceID: h.project.WorkspaceID, projectID: projectID, user: m.MemberID, ended: endedOf(memberships, m.MemberID),
				role: m.Role, sortOrder: domain.SortOrderFirst(lowest), by: actor.UserID, now: now}).apply(ctx, u.d.Projects); err != nil {
				return err
			}
		}
		added, err = storedMembers(ctx, u.d.Projects, projectID, ids)
		return err
	})
	if err != nil {
		return nil, err
	}
	return added, nil
}

// storedMembers are ids' active memberships of the project as stored, in
// ids' order; one missing is an internal error.
func storedMembers(ctx context.Context, members MemberLister, projectID uuid.UUID, ids []uuid.UUID) ([]domain.Member, error) {
	list, err := members.ListMembers(ctx, projectID)
	if err != nil {
		return nil, err
	}
	byAccount := make(map[uuid.UUID]domain.Member, len(list))
	for _, m := range list {
		byAccount[m.MemberID] = m
	}
	out := make([]domain.Member, len(ids))
	for i, id := range ids {
		m, ok := byAccount[id]
		if !ok {
			return nil, fmt.Errorf("the membership of %s in project %s is not there after its write", id, projectID)
		}
		out[i] = m
	}
	return out, nil
}
````

`server/internal/modules/project/app/fakes_growth_test.go`（新文件，87 行）：

````file server/internal/modules/project/app/fakes_growth_test.go
package app_test

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeStore's growth of a project's memberships: it logs each write and
// keeps it in the project, so that ListMembers and GetProject answer it.

func (f *fakeStore) CreateMember(ctx context.Context, m app.MemberRow) error {
	f.log.add(ctx, "CreateMember %s in %s/%s as %d by %s at %s", m.MemberID, m.WorkspaceID, m.ProjectID, m.Role, m.CreatedBy, m.Now.Format(timeFormat))
	if err := f.fail("CreateMember"); err != nil {
		return err
	}
	f.projects[m.ProjectID].members[m.MemberID] = app.Membership{ID: m.ID, Role: m.Role, Active: true}
	return nil
}

func (f *fakeStore) RestoreMember(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "RestoreMember %s as %d by %s at %s", id, role, by, now.Format(timeFormat))
	if err := f.fail("RestoreMember"); err != nil {
		return err
	}
	for _, p := range f.projects {
		for user, m := range p.members {
			if m.ID == id {
				p.members[user] = app.Membership{ID: id, Role: role, Active: true}
			}
		}
	}
	return nil
}

func (f *fakeStore) EnsurePreferences(ctx context.Context, p app.PreferencesRow) error {
	f.log.add(ctx, "EnsurePreferences %s in %s/%s at %v by %s at %s", p.UserID, p.WorkspaceID, p.ProjectID, p.SortOrder, p.CreatedBy,
		p.Now.Format(timeFormat))
	if err := f.fail("EnsurePreferences"); err != nil {
		return err
	}
	project := f.projects[p.ProjectID]
	if project.prefs == nil {
		project.prefs = map[uuid.UUID]domain.Preferences{}
	}
	if _, ok := project.prefs[p.UserID]; !ok {
		project.prefs[p.UserID] = domain.Preferences{Navigation: domain.DefaultPreferences().Navigation, SortOrder: p.SortOrder}
	}
	return nil
}

func (f *fakeStore) LowestSortOrder(ctx context.Context, workspaceID, userID uuid.UUID) (*float64, error) {
	f.log.add(ctx, "LowestSortOrder %s %s", workspaceID, userID)
	return f.lowest[userID], f.fail("LowestSortOrder")
}

var gina, hank, ivy = uuid.NewV7(), uuid.NewV7(), uuid.NewV7()

// newGrowth is newWrites with the workspace's members (fakeMembers) in
// acme, by account: bob, alice and dave members, carol a guest; gina an
// admin, hank a member and ivy a guest, of no project; erin none. hank's
// least place in his sidebar is -5.
func newGrowth() *writeFixture {
	f := newWrites()
	f.store.lowest = map[uuid.UUID]*float64{hank: ptr(-5.0)}
	f.members.roles = map[uuid.UUID]map[uuid.UUID]shared.Role{acme.ID: {
		bob: shared.RoleMember, alice: shared.RoleMember, dave: shared.RoleMember, carol: shared.RoleGuest,
		gina: shared.RoleAdmin, hank: shared.RoleMember, ivy: shared.RoleGuest,
	}}
	return f
}

// grown are the calls of one account's growth, by by: his membership,
// restored when ended is set, or made; his display settings at sortOrder.
func grown(user uuid.UUID, ended *app.Membership, role shared.Role, sortOrder float64, by uuid.UUID) []string {
	write := fmt.Sprintf("CreateMember %s in %s/%s as %d by %s at %s", user, acme.ID, webID, role, by, clockNow.Format(timeFormat))
	if ended != nil {
		write = fmt.Sprintf("RestoreMember %s as %d by %s at %s", ended.ID, role, by, clockNow.Format(timeFormat))
	}
	return []string{write, fmt.Sprintf("EnsurePreferences %s in %s/%s at %v by %s at %s", user, acme.ID, webID, sortOrder, by,
		clockNow.Format(timeFormat))}
}
````

`server/internal/modules/project/app/fakes_write_test.go`（修改，6 处）：

````old server/internal/modules/project/app/fakes_write_test.go
	workspaces *fakeWorkspaces
````
````new server/internal/modules/project/app/fakes_write_test.go
	workspaces *fakeWorkspaces
	members    *fakeMembers
````

````old server/internal/modules/project/app/fakes_write_test.go
// to lock, and the Authorizer's answers: bob's grant in acme, the project's
// admin; alice's 403, a project member; any other caller sees nothing.
````
````new server/internal/modules/project/app/fakes_write_test.go
// to lock, no workspace member to lock, and the Authorizer's answers: bob's
// grant in acme, the project's admin; alice's 403, a project member; any
// other caller sees nothing.
````

````old server/internal/modules/project/app/fakes_write_test.go
	return &writeFixture{log: log, tx: &fakeTx{log: log}, workspaces: &fakeWorkspaces{log: log},
````
````new server/internal/modules/project/app/fakes_write_test.go
	return &writeFixture{log: log, tx: &fakeTx{log: log}, workspaces: &fakeWorkspaces{log: log}, members: &fakeMembers{log: log},
````

````old server/internal/modules/project/app/fakes_write_test.go
	return app.NewLocks(f.store, f.workspaces, f.auth)
````
````new server/internal/modules/project/app/fakes_write_test.go
	return app.NewLocks(f.store, f.workspaces, f.members, f.auth)
````

````old server/internal/modules/project/app/fakes_write_test.go
// call; missing makes GetProject find nothing; moved, when set, is the
// workspace each project's lock reads, as if the project had moved there
// since ProjectWorkspace read it.
````
````new server/internal/modules/project/app/fakes_write_test.go
// call; missing makes GetProject and ListMembers find nothing, as if the
// writes were lost; lowest is each account's least place in his sidebar;
// moved, when set, is the workspace each project's lock reads, as if the
// project had moved there since ProjectWorkspace read it.
````

````old server/internal/modules/project/app/fakes_write_test.go
	missing  bool
````
````new server/internal/modules/project/app/fakes_write_test.go
	missing  bool
	lowest   map[uuid.UUID]*float64
````

`server/internal/modules/project/app/add_members_test.go`（新文件，176 行）：

````file server/internal/modules/project/app/add_members_test.go
package app_test

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newAdd is AddProjectMembers over newGrowth's fakes, its clock logged.
func newAdd() (*app.AddProjectMembers, *writeFixture) {
	f := newGrowth()
	return app.NewAddProjectMembers(app.AddMembersDeps{Locks: f.locks(), Projects: f.store, Tx: f.tx, Clock: clockAt{clockNow, f.log}}), f
}

// beforeTargets are the calls of user's addition of the accounts of in to
// project, before the targets' check: the transaction, the project's
// workspace, its lock, the targets' workspace memberships FOR SHARE, the
// project's lock, the decision, the targets' memberships of the project.
func beforeTargets(user, project uuid.UUID, in []domain.NewMember) []string {
	ids := make([]uuid.UUID, len(in))
	for i, m := range in {
		ids[i] = m.MemberID
	}
	return append(lockedTo(project), fmt.Sprintf("ShareMembers %s %v", acme.ID, ids), "LockProject "+project.String(),
		fmt.Sprintf("Authorize %s %s on %s/%s", user, domain.ActionMemberAdd, acme.ID, project), fmt.Sprintf("Memberships %s %v", project, ids))
}

// AddProjectMembers, in one transaction and in the order of M3 design 3.6,
// reads the project's workspace, locks it, the targets' memberships of it
// and the project, decides, reads the targets' memberships of the project,
// checks them, reads the clock, then for each target in the request's
// order reads his least place, restores his ended membership with the role
// asked for (dave, once a member, now an admin) or makes one, and makes
// his display settings at 10000 before his least place, or at the default
// (3.18); the answer is the targets as stored, in the request's order: a
// restored membership keeps its id.
func TestAddProjectMembers(t *testing.T) {
	uc, f := newAdd()
	in := []domain.NewMember{{MemberID: hank, Role: shared.RoleMember}, {MemberID: dave, Role: shared.RoleAdmin},
		{MemberID: gina, Role: shared.RoleAdmin}, {MemberID: ivy, Role: shared.RoleGuest}}
	daves := f.store.projects[webID].members[dave]

	got, err := uc.Execute(as(bob), webID, in)

	want := append(beforeTargets(bob, webID, in), "Now", "LowestSortOrder "+acme.ID.String()+" "+hank.String())
	want = append(want, grown(hank, nil, shared.RoleMember, -10005, bob)...)
	want = append(want, "LowestSortOrder "+acme.ID.String()+" "+dave.String())
	want = append(want, grown(dave, &daves, shared.RoleAdmin, 65535, bob)...)
	want = append(want, "LowestSortOrder "+acme.ID.String()+" "+gina.String())
	want = append(want, grown(gina, nil, shared.RoleAdmin, 65535, bob)...)
	want = append(want, "LowestSortOrder "+acme.ID.String()+" "+ivy.String())
	want = append(want, grown(ivy, nil, shared.RoleGuest, 65535, bob)...)
	want = append(want, "ListMembers "+webID.String())
	if err != nil || !slices.Equal(f.log.calls, want) {
		t.Fatalf("Execute() = %v, calls\n%q\nwant\n%q", err, f.log.calls, want)
	}
	members := f.store.projects[webID].members
	var answer []domain.Member
	for _, m := range in {
		answer = append(answer, domain.Member{ID: members[m.MemberID].ID, ProjectID: webID, MemberID: m.MemberID, Role: m.Role, CreatedAt: now})
	}
	if !reflect.DeepEqual(got, answer) || got[1].ID != daves.ID {
		t.Errorf("Execute() = %+v\nwant %+v, dave's membership %s restored", got, answer, daves.ID)
	}
}

// Refusals, each in its place, and nothing written: the request's values,
// and no caller, before the transaction; a project not there: 404 at its
// workspace; one not visible, or a member's 403, at the decision, whatever
// the targets (the matrix's PM and X cells, M3 design 9.2); each target
// refused after the decision, all of them in one 422 by their places: erin
// is no member of the workspace, alice is a member of the project already,
// gina, a workspace admin, is asked for as a member, ivy, a workspace
// guest, as a member; hank as an admin passes.
func TestAddProjectMembersRefuses(t *testing.T) {
	erins := []domain.NewMember{{MemberID: erin, Role: shared.RoleMember}}
	mixed := []domain.NewMember{{MemberID: erin, Role: shared.RoleMember}, {MemberID: alice, Role: shared.RoleMember},
		{MemberID: gina, Role: shared.RoleMember}, {MemberID: ivy, Role: shared.RoleMember}, {MemberID: hank, Role: shared.RoleAdmin}}
	tests := []struct {
		name  string
		ctx   context.Context
		id    uuid.UUID
		in    []domain.NewMember
		want  error
		calls []string
	}{
		{"none", as(bob), webID, nil, shared.Invalid(shared.FieldError{Field: "members", Code: "too_short"}), nil},
		{"no caller", context.Background(), webID, erins, shared.Unauthenticated(), nil},
		{"no project", as(bob), uuid.Nil(), erins, domain.ErrNotFound, noProject},
		{"not seen, an invalid target", as(erin), webID, erins, domain.ErrNotFound, beforeTargets(erin, webID, erins)[:6]},
		{"a project member, an invalid target", as(alice), webID, erins, shared.Forbidden(), beforeTargets(alice, webID, erins)[:6]},
		{"the targets", as(bob), webID, mixed, shared.Invalid(
			shared.FieldError{Field: "members[0].member_id", Code: "not_allowed"}, shared.FieldError{Field: "members[1].member_id", Code: "duplicate"},
			shared.FieldError{Field: "members[2].role", Code: "not_allowed"}, shared.FieldError{Field: "members[3].role", Code: "not_allowed"}),
			beforeTargets(bob, webID, mixed)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newAdd()
			before := maps.Clone(f.store.projects[webID].members)
			got, err := uc.Execute(tt.ctx, tt.id, tt.in)
			if !sameError(err, tt.want) || got != nil || !slices.Equal(f.log.calls, tt.calls) {
				t.Errorf("Execute() = %+v, %v, calls %q; want %v, calls %q", got, err, f.log.calls, tt.want, tt.calls)
			}
			if !maps.Equal(f.store.projects[webID].members, before) {
				t.Errorf("the memberships after the refusal: %v, want them as they were", f.store.projects[webID].members)
			}
		})
	}
}

// An archived project's members are added as any other's (M3 design 3.19).
func TestAddProjectMembersToAnArchivedProject(t *testing.T) {
	uc, f := newAdd()
	got, err := uc.Execute(as(bob), opsID, []domain.NewMember{{MemberID: hank, Role: shared.RoleGuest}})
	if err != nil || len(got) != 1 || got[0].MemberID != hank || got[0].ProjectID != opsID || !f.store.projects[opsID].members[hank].Active {
		t.Errorf("Execute() on ops = %+v, %v; want hank its guest", got, err)
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after; a membership the store cannot list after
// its write is an internal error.
func TestAddProjectMembersReturnsEachFailure(t *testing.T) {
	in := []domain.NewMember{{MemberID: hank, Role: shared.RoleMember}, {MemberID: dave, Role: shared.RoleMember}}
	fail := func(method string) func(f *writeFixture) {
		return func(f *writeFixture) { f.store.errs = map[string]error{method: errDisk} }
	}
	tests := []struct {
		name  string
		fail  func(f *writeFixture)
		calls int // how many of the calls ran
	}{
		{"the project's workspace", fail("ProjectWorkspace"), 2},
		{"the workspace's lock", func(f *writeFixture) { f.workspaces.err = errDisk }, 3},
		{"the targets' lock", func(f *writeFixture) { f.members.err = errDisk }, 4},
		{"the project's lock", fail("LockProject"), 5},
		{"the decision", func(f *writeFixture) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, 6},
		{"the memberships", fail("Memberships"), 7},
		{"the least place", fail("LowestSortOrder"), 9},
		{"a new membership", fail("CreateMember"), 10},
		{"the display settings", fail("EnsurePreferences"), 11},
		{"a restored membership", fail("RestoreMember"), 13},
		{"the answer", fail("ListMembers"), 15},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 15},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newAdd()
			tt.fail(f)
			daves := f.store.projects[webID].members[dave]
			all := slices.Concat(beforeTargets(bob, webID, in), []string{"Now", "LowestSortOrder " + acme.ID.String() + " " + hank.String()},
				grown(hank, nil, shared.RoleMember, -10005, bob), []string{"LowestSortOrder " + acme.ID.String() + " " + dave.String()},
				grown(dave, &daves, shared.RoleMember, 65535, bob), []string{"ListMembers " + webID.String()})
			got, err := uc.Execute(as(bob), webID, in)
			if !errors.Is(err, errDisk) || got != nil || !slices.Equal(f.log.calls, all[:tt.calls]) {
				t.Errorf("Execute() = %+v, %v, calls %q; want %v after %q", got, err, f.log.calls, errDisk, all[:tt.calls])
			}
		})
	}
	uc, f := newAdd()
	f.store.missing = true
	var e *shared.Error
	if got, err := uc.Execute(as(bob), webID, in); err == nil || errors.As(err, &e) || got != nil {
		t.Errorf("Execute() with the memberships not listed = %+v, %v; want an internal error", got, err)
	}
}
````

`server/internal/modules/project/app/list_members_test.go`（修改，1 处）：

````old server/internal/modules/project/app/list_members_test.go
	var out []domain.Member
````
````new server/internal/modules/project/app/list_members_test.go
	var out []domain.Member
	if f.missing {
		return out, nil
	}
````

`server/internal/modules/project/app/clock_test.go`（修改，2 处）：

````old server/internal/modules/project/app/clock_test.go
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
````
````new server/internal/modules/project/app/clock_test.go
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
````

````old server/internal/modules/project/app/clock_test.go
		}, changed(bob, webID, domain.PreferencesPatch{SortOrder: ptr(1.0)})},
````
````new server/internal/modules/project/app/clock_test.go
		}, changed(bob, webID, domain.PreferencesPatch{SortOrder: ptr(1.0)})},
		{"addProjectMembers", func() ([]string, error) {
			uc, f := newAdd()
			_, err := uc.Execute(as(bob), webID, []domain.NewMember{{MemberID: ivy, Role: shared.RoleGuest}})
			return f.log.calls, err
		}, slices.Concat(beforeTargets(bob, webID, []domain.NewMember{{MemberID: ivy, Role: shared.RoleGuest}}), []string{"Now",
			"LowestSortOrder " + acme.ID.String() + " " + ivy.String()}, grown(ivy, nil, shared.RoleGuest, 65535, bob),
			[]string{"ListMembers " + webID.String()})},
````

- [ ] **Step 4: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/project/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestEveryActionHasARuleAndEveryRuleAnAction|TestEveryModuleDeclaresItsActionsOrHasNone' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 5: 提交**

```bash
git add server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/project/app/add_members.go server/internal/modules/project/app/add_members_test.go server/internal/modules/project/app/clock_test.go server/internal/modules/project/app/fakes_growth_test.go server/internal/modules/project/app/fakes_write_test.go server/internal/modules/project/app/growth.go server/internal/modules/project/app/list_members_test.go server/internal/modules/project/app/lock.go server/internal/modules/project/app/ports.go server/internal/modules/project/domain/actions.go server/internal/modules/project/domain/member.go server/internal/modules/project/domain/member_test.go server/internal/modules/project/module.go
```
```bash
git commit -m "feat(M3/P4b): addProjectMembers' use case, and the growth's one step

Adding workspace members to a project checks the request, then in one
transaction takes Locks' path with targets: the workspace FOR SHARE, the
targets' workspace memberships FOR SHARE in id order, then the project
FOR NO KEY UPDATE; it decides, and checks each target after the
decision: an active member of the workspace, not of the project yet,
with a role his workspace role allows. Each is then restored with the
role asked for, or made a member, with display settings before his other
projects; the batch is written whole or not at all.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| 规则给项目成员、给不是成员的工作区管理员 | `TestPermissionMatrix`（Task 12 起） | 组合 |
| 不查目标是工作区的有效成员；已是有效成员的再加一次 | `TestPermissionMatrix`（Task 12 起，两个变体行） | 组合 |
| 目标在判定之前检查：不能添加的人得知目标的情况 | `TestPermissionMatrix`（Task 12 起，变体行的 PM、X 等格答 422） | 组合 |
| 一个目标被拒时别的照写：逐个检查、逐个写 | `TestPermissionMatrix`（Task 12 起） | 组合 |
| 恢复时不取请求的角色 | `TestARestoredMembershipGivesNoMoreThanItHad`（Task 13 起）；P3（Task 15 起） | 组合；端到端 |
| 显示设置放在默认位置，不在他别的项目之前 | `TestAddProjectMembers`；P3（Task 15 起） | 单元；端到端 |
| 先锁项目、后锁目标在工作区的成员关系（添加、加入共用的路径） | `TestAddProjectMembers`；`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 12 起：等 Ops 时不持有目标在工作区的成员关系）；`TestADemotionAndTheProjectSidesGrowthSerialize`（Task 14 起） | 单元；组合 |
| 工作区在目标之后锁 | `TestAddProjectMembers`；`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 12 起：等 acme 时已持有目标的成员关系）；`TestADemotionAndTheProjectSidesGrowthSerialize`（Task 14 起） | 单元；组合 |
| 只锁第一个目标在工作区的成员关系 | `TestAddProjectMembers`；`TestTheLeadAndTheDefaultAssigneeAreActiveMembersWhoAreNoGuests`（Task 12 起） | 单元；组合 |
| 没有调用者时也读 | `TestAddProjectMembersRefuses` | 单元 |
| 项目的工作区、工作区的锁、目标的锁、项目的锁和判定、成员关系、最低位置、写、读回、回答、提交的失败被吞掉；工作区的锁的失败答成找到；目标的锁重试一次 | `TestAddProjectMembersReturnsEachFailure` | 单元 |
| 请求里同一个账户出现两次被放过 | `TestCheckNewMembersReportsEveryProblem`（只在领域一层：放过之后第二次插入撞上部分唯一键，答 500，不是错的成功） | 单元 |
| `growth` 的恢复、新建、显示设置的失败被吞掉 | `TestAddProjectMembersReturnsEachFailure`；`TestJoinProjectReturnsEachFailure`（Task 13 起） | 单元 |

添加一类的安全性质（目标、重复、判定之前、部分写入）由 Task 12 的矩阵变体行在组合出的 app 上核对：这几行的变异从 Task 12 起才被发现。

**Done when:** 用例的四个测试和 `TestCheckTargets` 通过；目标在判定之后检查；锁的顺序是工作区、目标在工作区的成员关系、然后项目，都经 `Locks`。

---

### Task 12: 添加的接口和组合；矩阵的成员行

**Files:**
- Create: `server/internal/bootstrap/permission_matrix_members_test.go`、`server/internal/modules/project/adapter/http/add_test.go`
- Modify: `api/modules/project.yaml`、`server/internal/bootstrap/permission_matrix_project_test.go`、`server/internal/bootstrap/permission_matrix_seed_test.go`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/bootstrap/project_preferences_test.go`、`server/internal/bootstrap/project_write_locks_test.go`、`server/internal/bootstrap/project_writes_test.go`、`server/internal/modules/project/adapter/http/handler.go`、`server/internal/modules/project/adapter/http/handler_test.go`、`server/internal/modules/project/adapter/http/members.go`、`server/internal/modules/project/module.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/project/adapter/http/gen/bodyshape.gen.go`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.9，M3 设计 5.2、9.2）：`POST /api/v0/projects/{project_id}/members`，`ProjectMembersAdd{members: ProjectMemberNew[]}`（`ProjectMemberNew{member_id, role}`，`additionalProperties: false`），201 `ProjectMemberList`（请求的顺序）；码 `[validation_failed, project.not_found, forbidden]`；契约的说明不含守卫 `project-invitations` 的词序（brief 移交第 6 条）。
- 矩阵：`permission_matrix_members_test.go`：成员列表的行从 `permission_matrix_project_test.go` 移来；添加一行（`ofProject(201, 403, 403, 201, 403, 403)`，把工作区成员 WM-公 加为成员，答案核对他在列所指的项目里的成员关系）、四个无效目标的变体行（X 的账户不是工作区成员、WG- 作成员、WA- 作成员、PM 已是有效成员：`ofProject(422, 403, 403, 422, 403, 403)`）、已归档项目 1 格 201；`projectSeed.targets` 核对这些账户的前提（清扫 10，Task 13 加上 WA- 不是公开项目的成员）。
- `bootstrap/project_writes_test.go`：`inWorkspaceOf(t, pool, project, member, by, role)`（经工作区的存储让账户成为项目所在工作区的成员，返回工作区），`TestEachMemberHasHisOwnDisplaySettings` 也用它。

**Tests:**
- `adapter/http/add_test.go`：`TestAddProjectMembersPassesTheMembers`（成员按顺序传给用例，三种以外的角色也传；201 带用例答的列表）；`TestAddProjectMembersHoldsTheBodyToItsStructure`（九种结构错误 400，用例没被调用）；`TestAddProjectMembersRefusals`。
- `bootstrap/project_writes_test.go`：`TestTheLeadAndTheDefaultAssigneeAreActiveMembersWhoAreNoGuests`（新，组合出的 app 上：alice 把 carol 加为访客，bob 的成员关系已结束（SQL 代替 P5 的移出），dave 是 acme 的成员、不是 Web 的成员，三人作负责人、默认负责人各被拒，422 指名字段 `not_allowed`，项目不变；成员 erin 两者都可以）；`TestTheWritesOnAProjectStampTheirRequest` 加添加一行。
- `bootstrap/project_write_locks_test.go`：`projectWrite` 加 `targets`（写使之成为项目成员的账户，一段一个）；添加一行（在 Web 上加 bob、在 Ops 上加 carol，两人经 `inWorkspaceOf` 成为 acme 的成员）。第一段另核对等 acme 时不持有目标在 acme 的成员关系（`FOR UPDATE NOWAIT` 成功），第二段另核对等 Ops 时已持有它（`FOR UPDATE NOWAIT` 答 55P03）：目标在工作区之后、项目之前。
- 矩阵：六行 61 格。

- [ ] **Step 1: 接口描述**

`api/modules/project.yaml`（修改，2 处）：

````old api/modules/project.yaml
          description: The project's active members.
````
````new api/modules/project.yaml
          description: The project's active members.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ProjectMemberList'
        default:
          $ref: '#/components/responses/Problem'
    post:
      operationId: addProjectMembers
      tags: [project]
      summary: Add workspace members to a project
      description: >-
        For the project's admins, and its members who are the workspace's
        admins. Adds 1–100 accounts, each an active member of the workspace,
        with the role given: a workspace admin as an admin (20), a workspace
        guest as a guest (5), a workspace member as any of the three. One
        who was a member of the project before has his membership back, with
        the role given. Each is given display settings in the project, the
        project first in his sidebar, unless he has them. Refused, each by
        its place in members (validation_failed): an account that is not an
        active member of the workspace (member_id not_allowed), one that is
        a member of the project already, or named twice (member_id
        duplicate), and a role his workspace role does not allow (role
        not_allowed); the accounts are checked after the caller's role, so
        who may not add learns nothing of them. An archived project's
        members are added as any other's. A project that does not exist, is
        deleted, or that the caller does not see answers project.not_found;
        one he sees but may not add to, forbidden.
      security: [{bearer: []}]
      x-problem-codes: [validation_failed, project.not_found, forbidden]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ProjectMembersAdd'
      responses:
        '201':
          description: The accounts' memberships, in the request's order.
````

````old api/modules/project.yaml
            $ref: '#/components/schemas/ProjectMember'
````
````new api/modules/project.yaml
            $ref: '#/components/schemas/ProjectMember'
    ProjectMembersAdd:
      type: object
      additionalProperties: false
      required: [members]
      properties:
        members:
          description: 1–100 accounts, each named once.
          type: array
          items:
            $ref: '#/components/schemas/ProjectMemberNew'
    ProjectMemberNew:
      type: object
      additionalProperties: false
      required: [member_id, role]
      properties:
        member_id:
          description: An active member of the workspace.
          type: string
          format: uuid
        role:
          $ref: '#/components/schemas/ProjectRole'
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `b887e54ea3abda23687db413236ab4d1e5c7181c3d3eced28975f091940f10ed` | 2351 | `api/dist/openapi.yaml` |
| `75878eaf57948e09dbc4b4305ed6c679c8242e98e98c3b1ab90b5ca1369bcc13` | 60 | `server/internal/modules/project/adapter/http/gen/bodyshape.gen.go` |
| `4526b735083e802fc2ab715544207660457c4b912ddaca1485e8a0d8da95221f` | 1859 | `server/internal/modules/project/adapter/http/gen/server.gen.go` |
| `1ee85309d5f312011f5df7e8ec9eb045ca320345f053fbb3504e5309d3d599cb` | 2505 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/bodyshape.gen.go server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: HTTP 和接线**

`server/internal/modules/project/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/project/adapter/http/handler.go
}

// UseCases are the use cases behind the module's operations.
````
````new server/internal/modules/project/adapter/http/handler.go
}

// AddMembersUseCase is app.AddProjectMembers.
type AddMembersUseCase interface {
	Execute(ctx context.Context, projectID uuid.UUID, in []domain.NewMember) ([]domain.Member, error)
}

// UseCases are the use cases behind the module's operations.
````

````old server/internal/modules/project/adapter/http/handler.go
	ListMembers       ListMembersUseCase
````
````new server/internal/modules/project/adapter/http/handler.go
	ListMembers       ListMembersUseCase
	AddMembers        AddMembersUseCase
````

`server/internal/modules/project/adapter/http/members.go`（修改，2 处）：

````old server/internal/modules/project/adapter/http/members.go
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
````
````new server/internal/modules/project/adapter/http/members.go
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
````

````old server/internal/modules/project/adapter/http/members.go
}

// members is list as the API shows it: data an array, never null.
````
````new server/internal/modules/project/adapter/http/members.go
}

// AddProjectMembers serves POST /api/v0/projects/{project_id}/members: the
// members go to the use case as given, a role outside the three too, which
// the domain refuses (422).
func (h handler) AddProjectMembers(ctx context.Context, req gen.AddProjectMembersRequestObject) (gen.AddProjectMembersResponseObject, error) {
	in := make([]domain.NewMember, len(req.Body.Members))
	for i, m := range req.Body.Members {
		in[i] = domain.NewMember{MemberID: m.MemberID, Role: shared.Role(m.Role)}
	}
	list, err := h.uc.AddMembers.Execute(ctx, req.ProjectID, in)
	if err != nil {
		return nil, err
	}
	return gen.AddProjectMembers201JSONResponse(members(list)), nil
}

// members is list as the API shows it: data an array, never null.
````

`server/internal/modules/project/adapter/http/handler_test.go`（修改，3 处）：

````old server/internal/modules/project/adapter/http/handler_test.go
	members   *fakeMembers
````
````new server/internal/modules/project/adapter/http/handler_test.go
	members   *fakeMembers
	add       *fakeAdd
````

````old server/internal/modules/project/adapter/http/handler_test.go
		f.members = &fakeMembers{}
	}
````
````new server/internal/modules/project/adapter/http/handler_test.go
		f.members = &fakeMembers{}
	}
	if f.add == nil {
		f.add = &fakeAdd{}
	}
````

````old server/internal/modules/project/adapter/http/handler_test.go
		GetPreferences: f.prefs, UpdatePreferences: fakeUpdatePreferences{f.prefs}, ListMembers: f.members})
````
````new server/internal/modules/project/adapter/http/handler_test.go
		GetPreferences: f.prefs, UpdatePreferences: fakeUpdatePreferences{f.prefs}, ListMembers: f.members, AddMembers: f.add})
````

`server/internal/modules/project/adapter/http/add_test.go`（新文件，94 行）：

````file server/internal/modules/project/adapter/http/add_test.go
package httpadapter_test

import (
	"context"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeAdd is addProjectMembers: each call is recorded as "caller id" with
// what it got; it answers list, or err.
type fakeAdd struct {
	calls []string
	got   [][]domain.NewMember
	list  []domain.Member
	err   error
}

func (f *fakeAdd) Execute(ctx context.Context, projectID uuid.UUID, in []domain.NewMember) ([]domain.Member, error) {
	f.calls = append(f.calls, caller(ctx)+" "+projectID.String())
	f.got = append(f.got, in)
	return f.list, f.err
}

// The members go to the use case for the caller and the path's project, in
// their order, a role outside the three too; the answer is 201 with the
// list the use case answers.
func TestAddProjectMembersPassesTheMembers(t *testing.T) {
	add := &fakeAdd{list: []domain.Member{aliceInWeb, bobInWeb}}
	h := newServer(t, fakes{add: add})
	body := `{"members":[{"member_id":"0199a2b4-0000-7000-8000-000000000001","role":20},{"member_id":"0199a2b4-0000-7000-8000-000000000002","role":10}]}`
	res, got := do(t, h, request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/members", "bob", body))
	if res.StatusCode != http.StatusCreated || got != membersJSON+"\n" {
		t.Errorf("POST = %d %s, want 201 %s", res.StatusCode, got, membersJSON)
	}
	want := [][]domain.NewMember{{{MemberID: aliceID, Role: shared.RoleAdmin}, {MemberID: bobID, Role: 10}}}
	if !reflect.DeepEqual(add.got, want) || !slices.Equal(add.calls, []string{"bob " + webID.String()}) {
		t.Errorf("the use case got %q %+v, want bob's %+v", add.calls, add.got, want)
	}
}

// A body without members, with null ones, a member without his role or
// account, a field a member may not have, a value of another type: refused
// as bad_request before the use case.
func TestAddProjectMembersHoldsTheBodyToItsStructure(t *testing.T) {
	add := &fakeAdd{}
	h := newServer(t, fakes{add: add})
	for _, body := range []string{`{}`, `{"members":null}`, `{"members":[{"member_id":"0199a2b4-0000-7000-8000-000000000001"}]}`,
		`{"members":[{"role":15}]}`, `{"members":[{"member_id":"0199a2b4-0000-7000-8000-000000000001","role":15,"email":"a@b.c"}]}`,
		`{"members":[{"member_id":"alice","role":15}]}`, `{"members":[{"member_id":"0199a2b4-0000-7000-8000-000000000001","role":"15"}]}`,
		`{"members":{}}`, `{"members":[],"role":15}`} {
		if res, got := do(t, h, request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/members", "bob", body)); res.StatusCode != http.StatusBadRequest {
			t.Errorf("POST %s = %d %s, want 400", body, res.StatusCode, got)
		}
	}
	if len(add.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", add.calls)
	}
}

// The use case's refusals, as the contract declares them.
func TestAddProjectMembersRefusals(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"no project", domain.ErrNotFound, http.StatusNotFound,
			`{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`},
		{"a project member", shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"the targets", shared.Invalid(shared.FieldError{Field: "members[0].member_id", Code: "not_allowed",
			Message: "must be an active member of the workspace"}, shared.FieldError{Field: "members[1].role", Code: "not_allowed",
			Message: "is not one his workspace role allows: a workspace admin joins as an admin, a guest as a guest"}), http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"members[0].member_id","code":"not_allowed","message":"must be an active member of the workspace"},` +
				`{"field":"members[1].role","code":"not_allowed","message":"is not one his workspace role allows: a workspace admin joins as an admin, ` +
				`a guest as a guest"}]}`},
	}
	for _, tt := range tests {
		h := newServer(t, fakes{add: &fakeAdd{err: tt.err}})
		res, body := do(t, h, request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/members", "alice",
			`{"members":[{"member_id":"0199a2b4-0000-7000-8000-000000000002","role":15}]}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: POST = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
````

`server/internal/modules/project/module.go`（修改，2 处）：

````old server/internal/modules/project/module.go
// projects, checking an identifier, listing the members and each member's
// display settings, carries out the workspace module's cascades on the projects
````
````new server/internal/modules/project/module.go
// projects, checking an identifier, listing and adding the members and
// each member's display settings, carries out the workspace module's cascades on the projects
````

````old server/internal/modules/project/module.go
		ListMembers:       app.NewListProjectMembers(store, d.Authorizer),
````
````new server/internal/modules/project/module.go
		ListMembers:       app.NewListProjectMembers(store, d.Authorizer),
		AddMembers:        app.NewAddProjectMembers(app.AddMembersDeps{Locks: locks, Projects: store, Tx: d.Tx, Clock: d.Clock}),
````

- [ ] **Step 3: 矩阵的成员行**

`server/internal/bootstrap/permission_matrix_members_test.go`（新文件，103 行）：

````file server/internal/bootstrap/permission_matrix_members_test.go
package bootstrap

import (
	"fmt"
	"net/http"
	"testing"
	"uuid"
)

// The project members' rows of the permission matrix (M3 design 9.2):
// listing and adding them. prepareMatrix's preconditions hold the targets
// the rows add to what the rows say of them.

func memberMatrixRows() []matrixRow {
	return []matrixRow{
		{op: "listProjectMembers", columns: projectColumns, request: toProject(http.MethodGet, "/members", ""),
			cells: ofProject(cellOK, cellOK, cellOK, cellOK, cellForbidden, cellForbidden), check: listsTheProjectMembers},
		// The project's admins, and its members who are the workspace's
		// admins (M3 design 3.5): the workspace's member, of none of the
		// projects, is added as a member.
		{op: "addProjectMembers", write: true, columns: projectColumns, request: addsToProject(callerMember, 15),
			cells: ofProject(cellCreated, cellForbidden, cellForbidden, cellCreated, cellForbidden, cellForbidden), check: addsTheMember(callerMember, 15)},
		// A target refused after the decision (M3 design 3.6 convention 3,
		// 9.2): who may add gets the 422, and who may not his 403 or 404 as
		// with a valid target, learning nothing of it. X's account is no
		// member of acme; WG-'s is its guest, asked for as a member, and
		// WA-'s its admin, asked for as a member (M3 design 3.5: each joins
		// with his own role alone); PM's is the project's active member.
		{op: "addProjectMembers", variant: "a target who is no member of the workspace", write: true, columns: projectColumns,
			request: addsToProject(callerNever, 15),
			cells:   ofProject(cellValidationFailed, cellForbidden, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden)},
		{op: "addProjectMembers", variant: "a workspace guest as a member", write: true, columns: projectColumns,
			request: addsToProject(callerGuestOnly, 15),
			cells:   ofProject(cellValidationFailed, cellForbidden, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden)},
		{op: "addProjectMembers", variant: "a workspace admin as a member", write: true, columns: projectColumns,
			request: addsToProject(callerAdmin, 15),
			cells:   ofProject(cellValidationFailed, cellForbidden, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden)},
		{op: "addProjectMembers", variant: "an active member of the project", write: true, columns: projectColumns,
			request: addsToProject(callerProjectMember, 15),
			cells:   ofProject(cellValidationFailed, cellForbidden, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden)},
		// An archived project's members are added as any other's (M3 design
		// 3.19).
		{op: "addProjectMembers", variant: "archived", write: true, columns: archivedColumns, request: addsToProject(callerMember, 15),
			cells: map[caller]cell{callerArchivedAdmin: cellCreated}, check: addsTheMember(callerMember, 15)},
	}
}

// addsToProject is the request of a row whose callers each add target's
// account, as role, to the project their column targets.
func addsToProject(target caller, role int) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return http.MethodPost, "/api/v0/projects/" + s.project(projectOf(c)).String() + "/members",
			fmt.Sprintf(`{"members":[{"member_id":"%s","role":%d}]}`, s.account(target), role)
	}
}

// addsTheMember: the column's project's membership of target's account,
// with role.
func addsTheMember(target caller, role int) func(t *testing.T, c caller, s seeded, answer string) {
	return func(t *testing.T, c caller, s seeded, answer string) {
		var list struct {
			Data []struct {
				ProjectID uuid.UUID `json:"project_id"`
				MemberID  uuid.UUID `json:"member_id"`
				Role      int       `json:"role"`
			} `json:"data"`
		}
		decodeAnswer(t, answer, &list)
		if len(list.Data) != 1 || list.Data[0].ProjectID != s.project(projectOf(c)) || list.Data[0].MemberID != s.account(target) ||
			list.Data[0].Role != role {
			t.Errorf("%s adds %s; want %s's membership of %s as %d", c, answer, target, projectOf(c), role)
		}
	}
}

// listsTheProjectMembers: acme's public project's active members, each
// with his role, in the order prepareMatrix made them: PA, PM, the
// workspace's guest (PG's account), PM+WA and the removed member, whose
// membership of the project the stand-in left active (the list reads
// project_members alone, M3 design 5.2); not WG-, whose membership
// partingStates ended.
func listsTheProjectMembers(t *testing.T, c caller, s seeded, answer string) {
	var list struct {
		Data []struct {
			ProjectID uuid.UUID `json:"project_id"`
			MemberID  uuid.UUID `json:"member_id"`
			Role      int       `json:"role"`
		} `json:"data"`
	}
	decodeAnswer(t, answer, &list)
	want := []struct {
		c    caller
		role int
	}{{callerProjectAdmin, 20}, {callerProjectMember, 15}, {callerGuest, 5}, {callerMemberAndAdmin, 15}, {callerRemoved, 15}}
	ok := len(list.Data) == len(want)
	for i := 0; ok && i < len(want); i++ {
		m := list.Data[i]
		ok = m.ProjectID == s.project("acme/public") && m.MemberID == s.account(want[i].c) && m.Role == want[i].role
	}
	if !ok {
		t.Errorf("%s lists %s; want %+v of acme/public, in that order", c, answer, want)
	}
}
````

`server/internal/bootstrap/permission_matrix_project_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_project_test.go
			cells:   ofProject(cellOK, cellOK, cellOK, cellOK, cellForbidden, cellForbidden), check: readsPreferences("cycles", `["views"]`)},
		{op: "listProjectMembers", columns: projectColumns, request: toProject(http.MethodGet, "/members", ""),
			cells: ofProject(cellOK, cellOK, cellOK, cellOK, cellForbidden, cellForbidden), check: listsTheProjectMembers},
````
````new server/internal/bootstrap/permission_matrix_project_test.go
			cells:   ofProject(cellOK, cellOK, cellOK, cellOK, cellForbidden, cellForbidden), check: readsPreferences("cycles", `["views"]`)},
````

````old server/internal/bootstrap/permission_matrix_project_test.go
// listsTheProjectMembers: acme's public project's active members, each
// with his role, in the order prepareMatrix made them: PA, PM, the
// workspace's guest (PG's account), PM+WA and the removed member, whose
// membership of the project the stand-in left active (the list reads
// project_members alone, M3 design 5.2); not WG-, whose membership
// partingStates ended.
func listsTheProjectMembers(t *testing.T, c caller, s seeded, answer string) {
	var list struct {
		Data []struct {
			ProjectID uuid.UUID `json:"project_id"`
			MemberID  uuid.UUID `json:"member_id"`
			Role      int       `json:"role"`
		} `json:"data"`
	}
	decodeAnswer(t, answer, &list)
	want := []struct {
		c    caller
		role int
	}{{callerProjectAdmin, 20}, {callerProjectMember, 15}, {callerGuest, 5}, {callerMemberAndAdmin, 15}, {callerRemoved, 15}}
	ok := len(list.Data) == len(want)
	for i := 0; ok && i < len(want); i++ {
		m := list.Data[i]
		ok = m.ProjectID == s.project("acme/public") && m.MemberID == s.account(want[i].c) && m.Role == want[i].role
	}
	if !ok {
		t.Errorf("%s lists %s; want %+v of acme/public, in that order", c, answer, want)
	}
}

// toProjectPreferences is the request of a row whose callers each send method to
// their display settings in the project their column targets.
````
````new server/internal/bootstrap/permission_matrix_project_test.go
// toProjectPreferences is the request of a row whose callers each send
// method to their display settings in the project their column targets.
````

`server/internal/bootstrap/permission_matrix_seed_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_seed_test.go
			sd.workspace("other"), sd.workspace("acme"))
	}
}

````
````new server/internal/bootstrap/permission_matrix_seed_test.go
			sd.workspace("other"), sd.workspace("acme"))
	}
	s.targets(sd)
}

// targets checks the accounts addProjectMembers' rows add
// (permission_matrix_members_test.go): the workspace's member an active
// member of acme and of neither project his row adds him to, so that its
// 201 is an addition; X's account no active member of acme, WG-'s its
// active guest, WA-'s its active admin and PM's an active member of acme's
// public project, so that each 422 is the refusal its row names.
func (s projectSeed) targets(sd seeded) {
	s.t.Helper()
	ctx := context.Background()
	for _, tt := range []struct {
		c      caller
		role   shared.Role
		active bool
	}{{callerMember, shared.RoleMember, true}, {callerNever, 0, false}, {callerGuestOnly, shared.RoleGuest, true}, {callerAdmin, shared.RoleAdmin, true}} {
		if role, active, err := s.matrixSeed.store.ActiveRole(ctx, sd.workspace("acme"), s.ids[tt.c]); err != nil || active != tt.active ||
			active && role != tt.role {
			s.t.Fatalf("%s's role in acme = %d, %v, %v; want %d, %v", tt.c, role, active, err, tt.role, tt.active)
		}
	}
	for _, key := range []string{"acme/public", "acme/archived"} {
		if f, found, err := s.store.ProjectFacts(ctx, s.projects[key], s.ids[callerMember]); err != nil || !found || f.Member {
			s.t.Fatalf("the member's facts of %s = %+v, %v, %v; want him no active member of it", key, f, found, err)
		}
	}
	if f, found, err := s.store.ProjectFacts(ctx, s.projects["acme/public"], s.ids[callerProjectMember]); err != nil || !found || !f.Member {
		s.t.Fatalf("%s's facts of acme/public = %+v, %v, %v; want him its active member", callerProjectMember, f, found, err)
	}
}

````

`server/internal/bootstrap/permission_matrix_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_test.go
	return slices.Concat(workspaceMatrixRows(), projectMatrixRows())
````
````new server/internal/bootstrap/permission_matrix_test.go
	return slices.Concat(workspaceMatrixRows(), projectMatrixRows(), memberMatrixRows())
````

- [ ] **Step 4: 组合出的测试**

`server/internal/bootstrap/project_writes_test.go`（修改，7 处）：

````old server/internal/bootstrap/project_writes_test.go
	"context"
	"net/http"
````
````new server/internal/bootstrap/project_writes_test.go
	"context"
	"fmt"
	"net/http"
	"strings"
````

````old server/internal/bootstrap/project_writes_test.go
	"uuid"

````
````new server/internal/bootstrap/project_writes_test.go
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
````

````old server/internal/bootstrap/project_writes_test.go
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
````
````new server/internal/bootstrap/project_writes_test.go
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
````

````old server/internal/bootstrap/project_writes_test.go
// wrote, by $1 Web's id and $2 alice's.
````
````new server/internal/bootstrap/project_writes_test.go
// wrote, by $1 Web's id and $2 alice's. Bob, whom she adds, is acme's
// member.
````

````old server/internal/bootstrap/project_writes_test.go
	web := createdProject(t, contract, base, alice, "acme", "Web", "WEB")
````
````new server/internal/bootstrap/project_writes_test.go
	web := createdProject(t, contract, base, alice, "acme", "Web", "WEB")
	bobID := accountID(t, contract, base, registerAccount(t, contract, base, "bob@example.com").AccessToken)
	inWorkspaceOf(t, pool, web, bobID, aliceID, shared.RoleMember)
````

````old server/internal/bootstrap/project_writes_test.go
			"SELECT updated_at, updated_by_id = $2 FROM project_user_properties WHERE project_id = $1 AND user_id = $2 AND deleted_at IS NULL"},
````
````new server/internal/bootstrap/project_writes_test.go
			"SELECT updated_at, updated_by_id = $2 FROM project_user_properties WHERE project_id = $1 AND user_id = $2 AND deleted_at IS NULL"},
		// Bob's membership and his display settings, made.
		{"addProjectMembers", http.MethodPost, "/api/v0/projects/" + web.String() + "/members",
			`{"members":[{"member_id":"` + bobID.String() + `","role":15}]}`, http.StatusCreated,
			"SELECT created_at, created_by_id = $2 AND updated_by_id = $2 AND updated_at = created_at FROM project_members " +
				"WHERE project_id = $1 AND member_id <> $2 UNION ALL SELECT created_at, created_by_id = $2 AND updated_by_id = $2 AND " +
				"updated_at = created_at FROM project_user_properties WHERE project_id = $1 AND user_id <> $2"},
````

````old server/internal/bootstrap/project_writes_test.go
	return p.ID
}

````
````new server/internal/bootstrap/project_writes_test.go
	return p.ID
}

// inWorkspaceOf makes user a member of the workspace of project with role,
// by the account by, through the workspace store: no story of P4b invites
// anyone. It returns the workspace's id.
func inWorkspaceOf(t *testing.T, pool *pgxpool.Pool, project, user, by uuid.UUID, role shared.Role) uuid.UUID {
	t.Helper()
	var workspace uuid.UUID
	if err := pool.QueryRow(context.Background(), "SELECT workspace_id FROM projects WHERE id = $1", project).Scan(&workspace); err != nil {
		t.Fatal(err)
	}
	if err := workspacepg.New(pool).CreateMember(context.Background(), workspaceapp.MemberRow{ID: uuid.NewV7(), WorkspaceID: workspace,
		MemberID: user, Role: role, CreatedBy: by, Now: time.Now()}); err != nil {
		t.Fatal(err)
	}
	return workspace
}

// A project's lead and default assignee are active members of it who are
// not its guests (M3 design 3.19), on the wired app. Of alice's Web, carol,
// whom she adds as a guest, bob, whose membership ended (P5's removal ends
// one; SQL stands in), and dave, acme's member and none of Web's, are each
// refused as lead and as default assignee: 422 naming the field
// not_allowed, and the project as it was. erin, its member, is taken as
// both.
func TestTheLeadAndTheDefaultAssigneeAreActiveMembersWhoAreNoGuests(t *testing.T) {
	contract := apitest.Load(t)
	dbURL := pgtest.NewDatabase(t)
	base := startApp(t, testConfig(t, dbURL, false), migrations.FS())
	pool := openPool(t, dbURL)
	alice := registerAccount(t, contract, base, "alice@example.com").AccessToken
	aliceID := accountID(t, contract, base, alice)
	if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", alice, `{"name":"Acme","slug":"acme"}`); status != http.StatusCreated {
		t.Fatalf("creating acme = %d %s", status, body)
	}
	web := createdProject(t, contract, base, alice, "acme", "Web", "WEB")
	path := base + "/api/v0/projects/" + web.String()
	ids := map[string]uuid.UUID{}
	for _, name := range []string{"bob", "carol", "dave", "erin"} {
		ids[name] = accountID(t, contract, base, registerAccount(t, contract, base, name+"@example.com").AccessToken)
		inWorkspaceOf(t, pool, web, ids[name], aliceID, shared.RoleMember)
	}
	if status, body := call(t, contract, http.MethodPost, path+"/members", alice, fmt.Sprintf(
		`{"members":[{"member_id":"%s","role":15},{"member_id":"%s","role":5},{"member_id":"%s","role":15}]}`, ids["bob"], ids["carol"],
		ids["erin"])); status != http.StatusCreated {
		t.Fatalf("adding bob, carol and erin = %d %s", status, body)
	}
	if tag, err := pool.Exec(context.Background(), "UPDATE project_members SET is_active = false WHERE project_id = $1 AND member_id = $2",
		web, ids["bob"]); err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("ending bob's membership: %v, %v", tag, err)
	}
	stored := func() string {
		t.Helper()
		var s string
		if err := pool.QueryRow(context.Background(), `SELECT coalesce(project_lead_id::text, 'none') || ' ' ||
			coalesce(default_assignee_id::text, 'none') || ' ' || updated_at FROM projects WHERE id = $1`, web).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	before := stored()
	for _, name := range []string{"carol", "bob", "dave"} {
		for _, field := range []string{"project_lead_id", "default_assignee_id"} {
			status, body := call(t, contract, http.MethodPatch, path, alice, fmt.Sprintf(`{"%s":"%s"}`, field, ids[name]))
			if want := `"errors":[{"field":"` + field + `","code":"not_allowed"`; status != http.StatusUnprocessableEntity ||
				!strings.Contains(body, want) {
				t.Errorf("%s as %s = %d %s, want 422 with %s", name, field, status, body, want)
			}
		}
	}
	if got := stored(); got != before {
		t.Errorf("Web after the refusals: %s, want %s", got, before)
	}
	erin := ids["erin"].String()
	if status, body := call(t, contract, http.MethodPatch, path, alice,
		`{"project_lead_id":"`+erin+`","default_assignee_id":"`+erin+`"}`); status != http.StatusOK || !strings.HasPrefix(stored(), erin+" "+erin+" ") {
		t.Errorf("erin as both = %d %s, stored %s; want 200 and erin both", status, body, stored())
	}
}

````

`server/internal/bootstrap/project_preferences_test.go`（修改，2 处）：

````old server/internal/bootstrap/project_preferences_test.go
	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
````
````new server/internal/bootstrap/project_preferences_test.go
	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
````

````old server/internal/bootstrap/project_preferences_test.go
	var acme uuid.UUID
	if err := pool.QueryRow(context.Background(), "SELECT workspace_id FROM projects WHERE id = $1", web).Scan(&acme); err != nil {
		t.Fatal(err)
	}
	ctx, now := context.Background(), time.Now()
	if err := workspacepg.New(pool).CreateMember(ctx, workspaceapp.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, MemberID: bobID,
		Role: shared.RoleMember, CreatedBy: aliceID, Now: now}); err != nil {
		t.Fatal(err)
	}
````
````new server/internal/bootstrap/project_preferences_test.go
	acme := inWorkspaceOf(t, pool, web, bobID, aliceID, shared.RoleMember)
	ctx, now := context.Background(), time.Now()
````

`server/internal/bootstrap/project_write_locks_test.go`（修改，10 处）：

````old server/internal/bootstrap/project_write_locks_test.go
	"slices"
````
````new server/internal/bootstrap/project_write_locks_test.go
	"slices"
	"strings"
````

````old server/internal/bootstrap/project_write_locks_test.go
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
````
````new server/internal/bootstrap/project_write_locks_test.go
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
````

````old server/internal/bootstrap/project_write_locks_test.go
	op, method, path, body string // path: %s the project's id
	want                   int
````
````new server/internal/bootstrap/project_write_locks_test.go
	op, method, path, body string // path: %s the project's id; body: %s the target's id
	want                   int
	// targets are the accounts the write makes members of the project, one
	// a phase: acme's members, none of the project's.
	targets [2]string
````

````old server/internal/bootstrap/project_write_locks_test.go
		want: http.StatusOK},
````
````new server/internal/bootstrap/project_write_locks_test.go
		want: http.StatusOK},
	{op: "addProjectMembers", method: http.MethodPost, path: "/api/v0/projects/%s/members", body: `{"members":[{"member_id":"%s","role":15}]}`,
		want: http.StatusCreated, targets: [2]string{"bob", "carol"}},
````

````old server/internal/bootstrap/project_write_locks_test.go
// projects Web and Ops, each write once on each. Every write on a project
// of the contract has its row here: the matrix's rows that write at the
// project level are the list.
````
````new server/internal/bootstrap/project_write_locks_test.go
// projects Web and Ops, each write once on each, and a write's targets are
// made members of them. Every write on a project of the contract has its
// row here: the matrix's rows that write at the project level are the
// list.
````

````old server/internal/bootstrap/project_write_locks_test.go
//     Web waits for that row, and meanwhile does not hold Web's row: a FOR
//     UPDATE NOWAIT of it succeeds.
//   - In its transaction, FOR SHARE, before its project: another
//     transaction holds Ops's row FOR NO KEY UPDATE. The write on Ops waits
//     for it, and meanwhile holds acme's row at FOR SHARE, no stronger,
//     which another write on a project of acme shares (a FOR NO KEY UPDATE
//     NOWAIT of it fails, a FOR SHARE NOWAIT succeeds).
````
````new server/internal/bootstrap/project_write_locks_test.go
//     Web waits for that row, and meanwhile holds neither Web's row nor its
//     target's membership of acme: a FOR UPDATE NOWAIT of each succeeds.
//   - In its transaction, FOR SHARE, before its target and its project:
//     another transaction holds Ops's row FOR NO KEY UPDATE. The write on
//     Ops waits for it, and meanwhile holds acme's row at FOR SHARE, no
//     stronger, which another write on a project of acme shares (a FOR NO
//     KEY UPDATE NOWAIT of it fails, a FOR SHARE NOWAIT succeeds), and its
//     target's membership of acme (a FOR UPDATE NOWAIT fails, 55P03).
````

````old server/internal/bootstrap/project_write_locks_test.go
		createdProject(t, contract, base, alice, "acme", "Ops", "OPS")}
````
````new server/internal/bootstrap/project_write_locks_test.go
		createdProject(t, contract, base, alice, "acme", "Ops", "OPS")}
	ids, aliceID := map[string]uuid.UUID{}, accountID(t, contract, base, alice)
	for _, w := range projectWrites {
		for _, name := range w.targets {
			if name != "" {
				ids[name] = accountID(t, contract, base, registerAccount(t, contract, base, name+"@example.com").AccessToken)
				inWorkspaceOf(t, pool, projects[0], ids[name], aliceID, shared.RoleMember)
			}
		}
	}
	membership := "SELECT 1 FROM workspace_members WHERE workspace_id = $1 AND member_id = $2 AND deleted_at IS NULL FOR UPDATE NOWAIT"
````

````old server/internal/bootstrap/project_write_locks_test.go
			req := newRequest(t, w.method, base+fmt.Sprintf(w.path, project), alice, []byte(w.body))
````
````new server/internal/bootstrap/project_write_locks_test.go
			target, body := w.targets[phase], w.body
			if strings.Contains(body, "%s") {
				body = fmt.Sprintf(body, ids[target])
			}
			req := newRequest(t, w.method, base+fmt.Sprintf(w.path, project), alice, []byte(body))
````

````old server/internal/bootstrap/project_write_locks_test.go
					t.Errorf("%s holds its project while it waits for its workspace", w.op)
				}
````
````new server/internal/bootstrap/project_write_locks_test.go
					t.Errorf("%s holds its project while it waits for its workspace", w.op)
				}
				if target != "" && heldBy(t, pool, membership, acme.ID, ids[target]) {
					t.Errorf("%s holds %s's membership of acme while it waits for its workspace", w.op, target)
				}
````

````old server/internal/bootstrap/project_write_locks_test.go
					t.Errorf("%s does not hold its workspace FOR SHARE in its transaction while it waits for its project", w.op)
````
````new server/internal/bootstrap/project_write_locks_test.go
					t.Errorf("%s does not hold its workspace FOR SHARE in its transaction while it waits for its project", w.op)
				}
				if target != "" && !heldBy(t, pool, membership, acme.ID, ids[target]) {
					t.Errorf("%s does not hold %s's membership of acme while it waits for its project", w.op, target)
````

- [ ] **Step 5: 测试、lint、前端检查**

Run: `go -C server test -count=1 ./internal/modules/project/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix$|TestThePermissionMatrixCoversEveryOperation|TestMatrixViolationsCatchesEach|TestEveryColumnCallsAsARegisteredAccount|TestAPIRoutesAreTheContractsOperations|TestOperationsThatNeedATokenAnswer401WithoutOne|TestBodiesThatBreakTheStructureAnswer400|TestTheLeadAndTheDefaultAssigneeAreActiveMembersWhoAreNoGuests|TestEachMemberHasHisOwnDisplaySettings|TestTheWritesOnAProjectStampTheirRequest|TestEachWriteOnAProjectSharesItsWorkspaceFirst' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过（关键词守卫：生成的 `schema.gen.ts` 没有新的命中）。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

- [ ] **Step 6: 提交**

```bash
git add api/modules/project.yaml server/internal/bootstrap/permission_matrix_members_test.go server/internal/bootstrap/permission_matrix_project_test.go server/internal/bootstrap/permission_matrix_seed_test.go server/internal/bootstrap/permission_matrix_test.go server/internal/bootstrap/project_preferences_test.go server/internal/bootstrap/project_write_locks_test.go server/internal/bootstrap/project_writes_test.go server/internal/modules/project/adapter/http/add_test.go server/internal/modules/project/adapter/http/handler.go server/internal/modules/project/adapter/http/handler_test.go server/internal/modules/project/adapter/http/members.go server/internal/modules/project/module.go api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/bodyshape.gen.go server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P4b): addProjectMembers' operation; the members' rows of the matrix

POST /api/v0/projects/{project_id}/members answers 201 with the
memberships in the request's order. The matrix's members' rows add the
workspace's member, and four invalid targets that each column but the
adders' meets as its 403 or 404; the seed checks each target's state. On
the wired app, a lead or a default assignee who is a guest, an ended
member or no member of the project is refused.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| 契约声明 200 而不是答的 201 | `TestPermissionMatrix`（答案按契约核对） | 组合 |
| `project.New` 给添加不开事务的事务管理器 | `TestAGrowthRefusedAtItsCommitLeavesNoRow`、`TestADemotionAndTheProjectSidesGrowthSerialize`（Task 14 起） | 组合 |
| `project.New` 给添加固定在 2000 年的时钟 | `TestTheWritesOnAProjectStampTheirRequest` | 组合 |
| `project.New` 给添加谁都当作管理员放行的 `Authorizer`（经它自己的 `Locks`） | `TestPermissionMatrix` | 组合 |
| `project.New` 给添加一个 `Locks`，它的成员端口什么都不锁、答谁都是工作区成员 | `TestPermissionMatrix`；`TestADemotionAndTheProjectSidesGrowthSerialize`（Task 14 起） | 组合 |
| 只有添加不锁工作区 | `TestEachWriteOnAProjectSharesItsWorkspaceFirst`；`TestADemotionAndTheProjectSidesGrowthSerialize`（Task 14 起） | 组合 |
| 矩阵的准备：WA- 是 acme 的成员；PM 不是公开项目的成员；WG- 是成员；WM-公 是访客；X 是成员；WM-公 已是公开项目、已归档项目的成员；WA- 已是公开项目的成员 | `TestPermissionMatrix`（`targets` 的前提先失败） | 组合 |
| handler 吞掉用例的失败 | `TestAddProjectMembersRefusals` | 单元 |

**Done when:** 矩阵六行（61 格）通过，其中四个变体行让每个不能添加的列照样得到 403、404；负责人、默认负责人的规则在组合出的 app 上核对。

---

### Task 13: `joinProject`

**Files:**
- Create: `server/internal/bootstrap/project_members_test.go`、`server/internal/modules/project/adapter/http/join_test.go`、`server/internal/modules/project/app/join_project.go`、`server/internal/modules/project/app/join_project_test.go`
- Modify: `api/modules/project.yaml`、`api/openapi.yaml`、`server/internal/bootstrap/permission_matrix_members_test.go`、`server/internal/bootstrap/permission_matrix_seed_test.go`、`server/internal/bootstrap/project_write_locks_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/project/adapter/http/handler.go`、`server/internal/modules/project/adapter/http/handler_test.go`、`server/internal/modules/project/adapter/http/projects.go`、`server/internal/modules/project/app/clock_test.go`、`server/internal/modules/project/app/ports.go`、`server/internal/modules/project/domain/actions.go`、`server/internal/modules/project/domain/member.go`、`server/internal/modules/project/domain/member_test.go`、`server/internal/modules/project/module.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.10，M3 设计 3.5、3.6 约定三和六、3.18、9.1）：`POST /api/v0/projects/{project_id}/join`，200 `Project`；码 `[project.not_found, forbidden]`；操作名 `project.join`，规则 `{Level: LevelVisible}`：看得到项目的人；工作区角色由用例按集合要求。
- `domain.CanJoin(workspaceRole)`：工作区的管理员、成员（Plane `views/project/invite.py:131-189`），按集合。
- `app.MemberJoiner`（`MemberGrower` + `ProjectReader`）；`app.NewJoinProject(locks Locks, projects MemberJoiner, tx, clock)`：一个事务：`locks.lockAndDecide(write{project, project.join, targets: [调用者]})`（不加锁读项目的工作区 → 工作区 `FOR SHARE` → 他在工作区的成员关系 `FOR SHARE` → 项目 `FOR NO KEY UPDATE` → 判定）→ `CanJoin`（判定的 `Grant` 里他的工作区角色），否则 403，在读他的项目成员关系之前（G4：工作区访客即使是项目成员也一样被拒）→ `Memberships`：已是有效成员的什么都不写 → 时钟 → `growth.apply`（`JoinRole`；显示设置在 65535，`DefaultSortOrder`）→ `answer`。
- 矩阵：加入两行（`ofProject(200, 200, 403, 200, 200, 200)`，答案核对角色和 65535：已是成员的照旧，WA- 以 20、WM-公 以 15 加入；已归档 1 格 200）；`targets` 加上 WA- 不是公开项目的成员。

**Tests:**
- `domain/member_test.go`：`TestCanJoin`（工作区的管理员、成员可以；访客，和三种以外介于其间或高于它们的角色不可以）。
- `app/join_project_test.go`：`TestJoinProject`（完整的调用记录；新的成员关系取工作区角色，已结束的取较低的一个，按 9.1 的表；显示设置在 65535；回答是他看到的项目和他的角色）；`TestJoinProjectLeavesAnActiveMemberAsHeIs`；`TestJoinProjectJoinsAnArchivedProject`；`TestJoinProjectRefuses`（没有调用者在事务之前；没有的项目在找工作区时；看不到在判定；工作区访客 carol 是项目成员，在判定之后、读成员关系之前 403；高于三种的工作区角色也一样：按集合）；`TestJoinProjectReturnsEachFailure`（每个端口、提交的失败；hank 新加入，dave 恢复为成员）。
- `adapter/http/join_test.go`：`TestJoinProject`（只调加入的用例，200；拒绝照契约）。
- `bootstrap/project_write_locks_test.go`：`projectWrite` 加 `byTarget`（目标自己发出写：加入）；加入一行（dave 加入 Web、erin 加入 Ops，各自的 token），核对同添加。
- `bootstrap/project_members_test.go`：`TestARestoredMembershipGivesNoMoreThanItHad`（新，组合出的 app 上：bob 加入 Web 得到新的成员关系、他在请求的时刻、65535；然后每次以 SQL 代替 P5 的移出、以某个角色结束他的成员关系，他再加入或 alice 添加他：同一行恢复为 9.1 表那一行的角色，由调用者在那次请求的时刻，建立的时刻不变；最后一行之前经接口改他的工作区角色）。

- [ ] **Step 1: 接口描述**

`api/modules/project.yaml`（修改，1 处）：

````old api/modules/project.yaml
          description: The project as unarchived, as the caller sees it.
````
````new api/modules/project.yaml
          description: The project as unarchived, as the caller sees it.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Project'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/projects/{project_id}/join:
    parameters:
      - $ref: '#/components/parameters/ProjectID'
    post:
      operationId: joinProject
      tags: [project]
      summary: Join a project
      description: >-
        For the workspace's admins and members who see the project: its
        admins join any project, its members a public one. The caller
        becomes its member with his workspace role; one who was its member
        before has his membership back, with the lesser of the role it had
        and his workspace role. He is given display settings in the
        project unless he has them. One who is an active member already is
        left as he is. An archived project is joined as any other. A
        project that does not exist, is deleted, or that the caller does
        not see answers project.not_found; a workspace guest who sees it,
        as its member, forbidden.
      security: [{bearer: []}]
      x-problem-codes: [project.not_found, forbidden]
      responses:
        '200':
          description: The project as the caller sees it, now its member.
````

`api/openapi.yaml`（修改，1 处）：

````old api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1projects~1{project_id}~1unarchive'
````
````new api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1projects~1{project_id}~1unarchive'
  /api/v0/projects/{project_id}/join:
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1projects~1{project_id}~1join'
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `f92ea2c6e2c068d92f95d84bb117d5c68a4dbdee6f24376377579c9b0e995be3` | 2374 | `api/dist/openapi.yaml` |
| `83ec97947b89fd0cac9c6ef043ac097a45cc5142b8257dc2f132f92292ba0609` | 1964 | `server/internal/modules/project/adapter/http/gen/server.gen.go` |
| `8f925fda879c3ee20cbe8bef5d9ff83dcf2c38e84e4f8dbaf76b8e8ef586b335` | 2552 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: 领域、操作名、规则**

`server/internal/modules/project/domain/member.go`（修改，1 处）：

````old server/internal/modules/project/domain/member.go
}

// roleOrder is the project roles from the least to the most.
````
````new server/internal/modules/project/domain/member.go
}

// joiners are the workspace roles that may join a project they see (M3
// design 3.5; Plane views/project/invite.py:131-189): the workspace's
// admins and members, not its guests.
var joiners = []shared.Role{shared.RoleAdmin, shared.RoleMember}

// CanJoin reports whether an active member of the workspace of workspace
// role workspaceRole may join a project he sees. The set is named, not a
// bound.
func CanJoin(workspaceRole shared.Role) bool {
	return slices.Contains(joiners, workspaceRole)
}

// roleOrder is the project roles from the least to the most.
````

`server/internal/modules/project/domain/member_test.go`（修改，1 处）：

````old server/internal/modules/project/domain/member_test.go
}

// A new membership takes the workspace role; an ended one the lesser of
````
````new server/internal/modules/project/domain/member_test.go
}

// The workspace's admins and members may join; its guests, and a role
// outside the three, between them or above them, may not.
func TestCanJoin(t *testing.T) {
	for role, want := range map[shared.Role]bool{shared.RoleAdmin: true, shared.RoleMember: true, shared.RoleGuest: false, 0: false, 10: false,
		25: false} {
		if got := CanJoin(role); got != want {
			t.Errorf("CanJoin(%d) = %v, want %v", role, got, want)
		}
	}
}

// A new membership takes the workspace role; an ended one the lesser of
````

`server/internal/modules/project/domain/actions.go`（修改，2 处）：

````old server/internal/modules/project/domain/actions.go
	ActionMemberAdd shared.Action = "project_member.add"
````
````new server/internal/modules/project/domain/actions.go
	ActionMemberAdd shared.Action = "project_member.add"
	// ActionJoin is joining a project: joinProject.
	ActionJoin shared.Action = "project.join"
````

````old server/internal/modules/project/domain/actions.go
		ActionMemberList, ActionMemberAdd}
````
````new server/internal/modules/project/domain/actions.go
		ActionMemberList, ActionMemberAdd, ActionJoin}
````

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"project_member.add": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
````
````new server/internal/modules/access/domain/rules.go
	"project_member.add": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	// Whoever sees the project; the use case asks the workspace's admins
	// and members of him, by set, before it looks at his membership (M3
	// design 3.5, 6.4).
	"project.join": {Level: LevelVisible},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"project_member.add": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
````
````new server/internal/modules/access/domain/rules_test.go
	"project_member.add": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
	// As project.read: the guest's 403 is the use case's (M3 design 3.5).
	"project.join": {allowed, allowed, allowed, allowed, allowed, allowed, allowed, invisible, invisible, invisible, invisible, allowed,
		invisible, invisible, invisible, forbidden, forbidden},
````

- [ ] **Step 3: 用例**

`server/internal/modules/project/app/ports.go`（修改，1 处）：

````old server/internal/modules/project/app/ports.go
}

// ProjectArchiver is archiveProject's and unarchiveProject's repository.
````
````new server/internal/modules/project/app/ports.go
}

// MemberJoiner is joinProject's repository.
type MemberJoiner interface {
	MemberGrower
	ProjectReader
}

// ProjectArchiver is archiveProject's and unarchiveProject's repository.
````

`server/internal/modules/project/app/join_project.go`（新文件，75 行）：

````file server/internal/modules/project/app/join_project.go
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// JoinProject makes the caller a member of a project he sees: POST
// /api/v0/projects/{project_id}/join (M3 design 3.5).
type JoinProject struct {
	locks    Locks
	projects MemberJoiner
	tx       shared.TxManager
	clock    Clock
}

// NewJoinProject returns the use case.
func NewJoinProject(locks Locks, projects MemberJoiner, tx shared.TxManager, clock Clock) *JoinProject {
	return &JoinProject{locks: locks, projects: projects, tx: tx, clock: clock}
}

// Execute, in one transaction, in the order of M3 design 3.6: the
// project's locks (Locks: its workspace FOR SHARE; the caller's membership
// of the workspace FOR SHARE, which comes before the project in the lock
// order, convention 3; the project FOR NO KEY UPDATE) and the decision on
// project.join, which asks only that he see it; his workspace role, admin
// or member by set, else forbidden, before his membership of the project
// is looked at, so that a workspace guest who is its member is refused as
// any guest (3.5); his membership of the project. An active member is
// left as he is. Otherwise the clock, then his ended membership restored
// with the lesser of its role and his workspace role (domain.JoinRole,
// convention 6), or a new one with his workspace role; his display
// settings at the default place unless he has them (3.18). The answer is
// the project as he sees it.
func (u *JoinProject) Execute(ctx context.Context, id uuid.UUID) (domain.Project, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Project{}, err
	}
	var joined domain.Project
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		h, err := u.locks.lockAndDecide(ctx, actor, write{project: id, action: domain.ActionJoin, targets: []uuid.UUID{actor.UserID}})
		if err != nil {
			return err
		}
		if !domain.CanJoin(h.grant.WorkspaceRole) {
			return shared.Forbidden()
		}
		memberships, err := u.projects.Memberships(ctx, id, []uuid.UUID{actor.UserID})
		if err != nil {
			return err
		}
		if m, ok := memberships[actor.UserID]; !ok || !m.Active {
			ended := endedOf(memberships, actor.UserID)
			var was *shared.Role
			if ended != nil {
				was = &ended.Role
			}
			if err := (growth{workspaceID: h.project.WorkspaceID, projectID: id, user: actor.UserID, ended: ended,
				role: domain.JoinRole(was, h.grant.WorkspaceRole), sortOrder: domain.DefaultSortOrder, by: actor.UserID,
				now: u.clock.Now()}).apply(ctx, u.projects); err != nil {
				return err
			}
		}
		joined, err = answer(ctx, u.projects, id, actor.UserID)
		return err
	})
	if err != nil {
		return domain.Project{}, err
	}
	return joined, nil
}
````

`server/internal/modules/project/app/join_project_test.go`（新文件，202 行）：

````file server/internal/modules/project/app/join_project_test.go
package app_test

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newJoin is JoinProject over newGrowth's fakes, its clock logged, each
// caller deciding as his roles in acme: gina its admin, hank and dave its
// members, carol its guest and Web's.
func newJoin() (*app.JoinProject, *writeFixture) {
	f := newGrowth()
	for user, g := range map[uuid.UUID]shared.Grant{gina: {WorkspaceRole: shared.RoleAdmin}, hank: {WorkspaceRole: shared.RoleMember},
		dave: {WorkspaceRole: shared.RoleMember}, carol: {WorkspaceRole: shared.RoleGuest, ProjectRole: shared.RoleGuest}} {
		f.auth.grants[grantKey{user, acme.ID}] = g
	}
	return app.NewJoinProject(f.locks(), f.store, f.tx, clockAt{clockNow, f.log}), f
}

// beforeJoin are the calls of user's joining project before it writes:
// the transaction, the project's workspace, its lock, his workspace
// membership FOR SHARE, the project's lock, the decision, his membership
// of the project.
func beforeJoin(user, project uuid.UUID) []string {
	return append(lockedTo(project), fmt.Sprintf("ShareMembers %s [%s]", acme.ID, user), "LockProject "+project.String(),
		fmt.Sprintf("Authorize %s %s on %s/%s", user, domain.ActionJoin, acme.ID, project), fmt.Sprintf("Memberships %s [%s]", project, user))
}

// JoinProject, in one transaction and in the order of M3 design 3.6, reads
// the project's workspace, locks it, the caller's membership of it and the
// project, decides, reads his membership of the project and the clock,
// then makes his membership with his workspace role, or restores his ended
// one with the lesser of its role and his workspace role, by the order of
// the roles (9.1's table; its fourth row, a workspace guest, is refused
// before, TestJoinProjectRefuses), and makes his display settings at
// 65535 (3.18). The answer is the project as he sees it, his role in it.
func TestJoinProject(t *testing.T) {
	tests := []struct {
		name      string
		user      uuid.UUID
		ended     *shared.Role // his ended membership's role; nil: none
		workspace shared.Role  // his workspace role
		want      shared.Role
	}{
		{"a workspace member", hank, nil, shared.RoleMember, shared.RoleMember},
		{"a workspace admin", gina, nil, shared.RoleAdmin, shared.RoleAdmin},
		{"a guest before, a workspace member", dave, ptr(shared.RoleGuest), shared.RoleMember, shared.RoleGuest},
		{"an admin before, a workspace member", dave, ptr(shared.RoleAdmin), shared.RoleMember, shared.RoleMember},
		{"a member before, a workspace admin", dave, ptr(shared.RoleMember), shared.RoleAdmin, shared.RoleMember},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newJoin()
			f.auth.grants[grantKey{tt.user, acme.ID}] = shared.Grant{WorkspaceRole: tt.workspace}
			var ended *app.Membership
			if tt.ended != nil {
				m := f.store.projects[webID].members[tt.user]
				m.Role = *tt.ended
				f.store.projects[webID].members[tt.user] = m
				ended = &m
			}

			got, err := uc.Execute(as(tt.user), webID)

			want := slices.Concat(beforeJoin(tt.user, webID), []string{"Now"}, grown(tt.user, ended, tt.want, 65535, tt.user),
				[]string{fmt.Sprintf("GetProject %s for %s", webID, tt.user)})
			if err != nil || !slices.Equal(f.log.calls, want) {
				t.Fatalf("Execute() = %v, calls\n%q\nwant\n%q", err, f.log.calls, want)
			}
			if got.ID != webID || got.MemberRole == nil || *got.MemberRole != tt.want {
				t.Errorf("Execute() = %+v, want Web with his role %d", got, tt.want)
			}
		})
	}
}

// An active member of the project joining is left as he is: nothing is
// written, and the answer is the project as he sees it.
func TestJoinProjectLeavesAnActiveMemberAsHeIs(t *testing.T) {
	uc, f := newJoin()
	before := maps.Clone(f.store.projects[webID].members)
	got, err := uc.Execute(as(bob), webID)
	want := append(beforeJoin(bob, webID), fmt.Sprintf("GetProject %s for %s", webID, bob))
	if err != nil || !slices.Equal(f.log.calls, want) || got.MemberRole == nil || *got.MemberRole != shared.RoleAdmin {
		t.Errorf("Execute() = %+v, %v, calls %q; want Web, bob its admin, calls %q", got, err, f.log.calls, want)
	}
	if !maps.Equal(f.store.projects[webID].members, before) {
		t.Errorf("the memberships: %v, want them as they were", f.store.projects[webID].members)
	}
}

// Refusals, each in its place, and nothing written: no caller before the
// transaction; a project not there at its workspace; one not visible at
// the decision; a workspace guest, carol, who is the project's member,
// forbidden after the decision and before his membership is read (M3
// design 3.5, 9.2's PG cell), as is a workspace role outside the three,
// above them: the roles that may join are a set.
func TestJoinProjectRefuses(t *testing.T) {
	tests := []struct {
		name  string
		ctx   context.Context
		id    uuid.UUID
		setup func(f *writeFixture)
		want  error
		calls []string
	}{
		{"no caller", context.Background(), webID, nil, shared.Unauthenticated(), nil},
		{"no project", as(hank), uuid.Nil(), nil, domain.ErrNotFound, noProject},
		{"not seen", as(erin), webID, nil, domain.ErrNotFound, beforeJoin(erin, webID)[:6]},
		{"a workspace guest, the project's member", as(carol), webID, nil, shared.Forbidden(), beforeJoin(carol, webID)[:6]},
		{"a workspace role above the three", as(erin), webID, func(f *writeFixture) {
			f.auth.grants[grantKey{erin, acme.ID}] = shared.Grant{WorkspaceRole: 25}
		}, shared.Forbidden(), beforeJoin(erin, webID)[:6]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newJoin()
			if tt.setup != nil {
				tt.setup(f)
			}
			before := maps.Clone(f.store.projects[webID].members)
			got, err := uc.Execute(tt.ctx, tt.id)
			if !sameError(err, tt.want) || got.ID != uuid.Nil() || !slices.Equal(f.log.calls, tt.calls) {
				t.Errorf("Execute() = %+v, %v, calls %q; want %v, calls %q", got, err, f.log.calls, tt.want, tt.calls)
			}
			if !maps.Equal(f.store.projects[webID].members, before) {
				t.Errorf("the memberships after the refusal: %v, want them as they were", f.store.projects[webID].members)
			}
		})
	}
}

// An archived project is joined as any other (M3 design 3.19).
func TestJoinProjectJoinsAnArchivedProject(t *testing.T) {
	uc, f := newJoin()
	got, err := uc.Execute(as(hank), opsID)
	if err != nil || got.ID != opsID || got.MemberRole == nil || *got.MemberRole != shared.RoleMember || !f.store.projects[opsID].members[hank].Active {
		t.Errorf("Execute() on ops = %+v, %v; want hank its member", got, err)
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after; a project the store cannot read after
// the write is an internal error. hank, a workspace member, joins anew;
// dave, one too, restores his ended membership as a member's.
func TestJoinProjectReturnsEachFailure(t *testing.T) {
	all := func(f *writeFixture, user uuid.UUID) []string {
		var ended *app.Membership
		if m, ok := f.store.projects[webID].members[user]; ok {
			ended = &m
		}
		return slices.Concat(beforeJoin(user, webID), []string{"Now"}, grown(user, ended, shared.RoleMember, 65535, user),
			[]string{fmt.Sprintf("GetProject %s for %s", webID, user)})
	}
	fail := func(method string) func(f *writeFixture) {
		return func(f *writeFixture) { f.store.errs = map[string]error{method: errDisk} }
	}
	tests := []struct {
		name  string
		user  uuid.UUID
		fail  func(f *writeFixture)
		calls int // how many of the calls ran
	}{
		{"the project's workspace", hank, fail("ProjectWorkspace"), 2},
		{"the workspace's lock", hank, func(f *writeFixture) { f.workspaces.err = errDisk }, 3},
		{"his membership's lock", hank, func(f *writeFixture) { f.members.err = errDisk }, 4},
		{"the project's lock", hank, fail("LockProject"), 5},
		{"the decision", hank, func(f *writeFixture) { f.auth.errs[grantKey{hank, acme.ID}] = errDisk }, 6},
		{"his membership", hank, fail("Memberships"), 7},
		{"a new membership", hank, fail("CreateMember"), 9},
		{"the display settings", hank, fail("EnsurePreferences"), 10},
		{"a restored membership", dave, fail("RestoreMember"), 9},
		{"the answer", hank, fail("GetProject"), 11},
		{"the commit", hank, func(f *writeFixture) { f.tx.commitErr = errDisk }, 11},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newJoin()
			tt.fail(f)
			want := all(f, tt.user)[:tt.calls]
			got, err := uc.Execute(as(tt.user), webID)
			if !errors.Is(err, errDisk) || got.ID != uuid.Nil() || !slices.Equal(f.log.calls, want) {
				t.Errorf("Execute() = %+v, %v, calls %q; want %v after %q", got, err, f.log.calls, errDisk, want)
			}
		})
	}
	uc, f := newJoin()
	f.store.missing = true
	var e *shared.Error
	if got, err := uc.Execute(as(hank), webID); err == nil || errors.As(err, &e) || got.ID != uuid.Nil() {
		t.Errorf("Execute() with the project not read back = %+v, %v; want an internal error", got, err)
	}
}
````

`server/internal/modules/project/app/clock_test.go`（修改，1 处）：

````old server/internal/modules/project/app/clock_test.go
			[]string{"ListMembers " + webID.String()})},
````
````new server/internal/modules/project/app/clock_test.go
			[]string{"ListMembers " + webID.String()})},
		{"joinProject", func() ([]string, error) {
			uc, f := newJoin()
			_, err := uc.Execute(as(hank), webID)
			return f.log.calls, err
		}, slices.Concat(beforeJoin(hank, webID), []string{"Now"}, grown(hank, nil, shared.RoleMember, 65535, hank),
			[]string{"GetProject " + webID.String() + " for " + hank.String()})},
````

- [ ] **Step 4: HTTP 和接线**

`server/internal/modules/project/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/project/adapter/http/handler.go
}

// UseCases are the use cases behind the module's operations.
````
````new server/internal/modules/project/adapter/http/handler.go
}

// JoinProjectUseCase is app.JoinProject.
type JoinProjectUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) (domain.Project, error)
}

// UseCases are the use cases behind the module's operations.
````

````old server/internal/modules/project/adapter/http/handler.go
	AddMembers        AddMembersUseCase
````
````new server/internal/modules/project/adapter/http/handler.go
	AddMembers        AddMembersUseCase
	JoinProject       JoinProjectUseCase
````

`server/internal/modules/project/adapter/http/projects.go`（修改，1 处）：

````old server/internal/modules/project/adapter/http/projects.go
	return gen.UnarchiveProject200JSONResponse(project(p)), nil
````
````new server/internal/modules/project/adapter/http/projects.go
	return gen.UnarchiveProject200JSONResponse(project(p)), nil
}

// JoinProject serves POST /api/v0/projects/{project_id}/join.
func (h handler) JoinProject(ctx context.Context, req gen.JoinProjectRequestObject) (gen.JoinProjectResponseObject, error) {
	p, err := h.uc.JoinProject.Execute(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}
	return gen.JoinProject200JSONResponse(project(p)), nil
````

`server/internal/modules/project/adapter/http/handler_test.go`（修改，3 处）：

````old server/internal/modules/project/adapter/http/handler_test.go
	add       *fakeAdd
````
````new server/internal/modules/project/adapter/http/handler_test.go
	add       *fakeAdd
	join      *fakeOnProject
````

````old server/internal/modules/project/adapter/http/handler_test.go
		f.add = &fakeAdd{}
	}
````
````new server/internal/modules/project/adapter/http/handler_test.go
		f.add = &fakeAdd{}
	}
	if f.join == nil {
		f.join = &fakeOnProject{}
	}
````

````old server/internal/modules/project/adapter/http/handler_test.go
		GetPreferences: f.prefs, UpdatePreferences: fakeUpdatePreferences{f.prefs}, ListMembers: f.members, AddMembers: f.add})
````
````new server/internal/modules/project/adapter/http/handler_test.go
		GetPreferences: f.prefs, UpdatePreferences: fakeUpdatePreferences{f.prefs}, ListMembers: f.members, AddMembers: f.add,
		JoinProject: f.join})
````

`server/internal/modules/project/adapter/http/join_test.go`（新文件，41 行）：

````file server/internal/modules/project/adapter/http/join_test.go
package httpadapter_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// POST .../join goes to the use case for the caller and the path's project,
// and to no other; the answer is 200 with the project the use case
// answers. Its refusals, as the contract declares them.
func TestJoinProject(t *testing.T) {
	join, archive := &fakeOnProject{answer: web}, &fakeOnProject{answer: web}
	h := newServer(t, fakes{join: join, archive: archive})
	path := "/api/v0/projects/" + webID.String() + "/join"
	if res, body := do(t, h, request(http.MethodPost, path, "bob", "")); res.StatusCode != http.StatusOK || body != webJSON+"\n" {
		t.Errorf("POST %s = %d %s, want 200 %s", path, res.StatusCode, body, webJSON)
	}
	if want := []string{"bob " + webID.String()}; !slices.Equal(join.calls, want) || len(archive.calls) != 0 {
		t.Errorf("calls %q and %q; want %q, and none to archiving", join.calls, archive.calls, want)
	}
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"not seen", domain.ErrNotFound, http.StatusNotFound,
			`{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`},
		{"a workspace guest", shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
	} {
		h := newServer(t, fakes{join: &fakeOnProject{err: tt.err}})
		if res, body := do(t, h, request(http.MethodPost, path, "alice", "")); res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: POST = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
````

`server/internal/modules/project/module.go`（修改，2 处）：

````old server/internal/modules/project/module.go
// projects, checking an identifier, listing and adding the members and
// each member's display settings, carries out the workspace module's cascades on the projects
// (ProjectCascade), and offers the access module its reads of a project
// (ProjectAccess).
````
````new server/internal/modules/project/module.go
// projects, checking an identifier, listing, adding and joining the
// members and each member's display settings, carries out the workspace
// module's cascades on the projects (ProjectCascade), and offers the
// access module its reads of a project (ProjectAccess).
````

````old server/internal/modules/project/module.go
		AddMembers:        app.NewAddProjectMembers(app.AddMembersDeps{Locks: locks, Projects: store, Tx: d.Tx, Clock: d.Clock}),
````
````new server/internal/modules/project/module.go
		AddMembers:        app.NewAddProjectMembers(app.AddMembersDeps{Locks: locks, Projects: store, Tx: d.Tx, Clock: d.Clock}),
		JoinProject:       app.NewJoinProject(locks, store, d.Tx, d.Clock),
````

- [ ] **Step 5: 矩阵和组合出的测试**

`server/internal/bootstrap/permission_matrix_members_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_members_test.go
// listing and adding them. prepareMatrix's preconditions hold the targets
// the rows add to what the rows say of them.
````
````new server/internal/bootstrap/permission_matrix_members_test.go
// listing, adding and joining. prepareMatrix's preconditions hold the
// accounts the rows add, and the joiners, to what the rows say of them.
````

````old server/internal/bootstrap/permission_matrix_members_test.go
			cells: map[caller]cell{callerArchivedAdmin: cellCreated}, check: addsTheMember(callerMember, 15)},
````
````new server/internal/bootstrap/permission_matrix_members_test.go
			cells: map[caller]cell{callerArchivedAdmin: cellCreated}, check: addsTheMember(callerMember, 15)},
		// Whoever sees the project but a workspace guest, its member too
		// (M3 design 3.5): an active member is left as he is; the
		// workspace's admin and member, of no project, join with their
		// workspace roles.
		{op: "joinProject", write: true, columns: projectColumns, request: toProject(http.MethodPost, "/join", ""),
			cells: ofProject(cellOK, cellOK, cellForbidden, cellOK, cellOK, cellOK), check: joinsAs},
		{op: "joinProject", variant: "archived", write: true, columns: archivedColumns, request: toProject(http.MethodPost, "/join", ""),
			cells: map[caller]cell{callerArchivedAdmin: cellOK}, check: joinsAs},
	}
}

// joinsAs: the column's project as the caller sees it, his role in it and
// its place in his sidebar: PA's 20 and PM's and PM+WA's 15, the archived
// project's admin's 20, as they were; WA-'s 20 and WM-公's 15, their
// workspace roles; each at 65535, the joiners' place (M3 design 3.18) and
// the seeded members'.
func joinsAs(t *testing.T, c caller, s seeded, answer string) {
	var p struct {
		ID         uuid.UUID `json:"id"`
		MemberRole *int      `json:"member_role"`
		SortOrder  float64   `json:"sort_order"`
	}
	decodeAnswer(t, answer, &p)
	want := map[caller]int{callerProjectAdmin: 20, callerProjectMember: 15, callerMemberAndAdmin: 15, callerArchivedAdmin: 20,
		callerAdminOnly: 20, callerMemberPublic: 15}[c]
	if p.ID != s.project(projectOf(c)) || p.MemberRole == nil || *p.MemberRole != want || p.SortOrder != 65535 {
		t.Errorf("%s joins %s; want %s, his role %d, at 65535", c, answer, projectOf(c), want)
````

`server/internal/bootstrap/permission_matrix_seed_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_seed_test.go
// public project, so that each 422 is the refusal its row names.
````
````new server/internal/bootstrap/permission_matrix_seed_test.go
// public project, so that each 422 is the refusal its row names. The
// workspace's member, WM-公, and its admin, WA-, are no active members of
// acme's public project either, so that joinProject's row makes each a
// member with his workspace role, and the add of WA- as a member is refused
// for his role, not as a duplicate.
````

````old server/internal/bootstrap/permission_matrix_seed_test.go
	for _, key := range []string{"acme/public", "acme/archived"} {
		if f, found, err := s.store.ProjectFacts(ctx, s.projects[key], s.ids[callerMember]); err != nil || !found || f.Member {
			s.t.Fatalf("the member's facts of %s = %+v, %v, %v; want him no active member of it", key, f, found, err)
````
````new server/internal/bootstrap/permission_matrix_seed_test.go
	for _, tt := range []struct {
		key string
		c   caller
	}{{"acme/public", callerMember}, {"acme/archived", callerMember}, {"acme/public", callerAdmin}} {
		if f, found, err := s.store.ProjectFacts(ctx, s.projects[tt.key], s.ids[tt.c]); err != nil || !found || f.Member {
			s.t.Fatalf("%s's facts of %s = %+v, %v, %v; want him no active member of it", tt.c, tt.key, f, found, err)
````

`server/internal/bootstrap/project_members_test.go`（新文件，126 行）：

````file server/internal/bootstrap/project_members_test.go
package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// membership is an account's undeleted membership of a project as stored.
type membership struct {
	id          uuid.UUID
	role        shared.Role
	active      bool
	by          uuid.UUID // updated_by_id
	at, created time.Time // updated_at, created_at
}

func membershipOf(t *testing.T, pool *pgxpool.Pool, project, user uuid.UUID) membership {
	t.Helper()
	var m membership
	if err := pool.QueryRow(context.Background(), `SELECT id, role, is_active, updated_by_id, updated_at, created_at FROM project_members
		WHERE project_id = $1 AND member_id = $2 AND deleted_at IS NULL`, project, user).Scan(&m.id, &m.role, &m.active, &m.by, &m.at,
		&m.created); err != nil {
		t.Fatal(err)
	}
	return m
}

// A restored membership gives no more than it had, nor more than a new one
// would, when its member joins; when an admin adds him, it takes the role
// the admin asks for (M3 design 3.5, 3.6 convention 6, 9.1's table), on
// the wired app. bob, acme's member, joins alice's public project Web: a
// new membership as a member, by him at the time of his request, his
// display settings at 65535. Then, each time, his membership is ended with
// a role, as P5's removal will end it (SQL stands in), and he joins again,
// or alice adds him: the same row is active again with the role of 9.1's
// row, by the caller at the time of that request, still made when it was.
// His workspace role is changed through the API before the last row.
func TestARestoredMembershipGivesNoMoreThanItHad(t *testing.T) {
	contract, base, pool, alice, aliceID, web, _ := twoProjects(t)
	bob := registerAccount(t, contract, base, "bob@example.com").AccessToken
	bobID := accountID(t, contract, base, bob)
	acme := inWorkspaceOf(t, pool, web, bobID, aliceID, shared.RoleMember)
	join := func(want shared.Role) {
		t.Helper()
		status, body := call(t, contract, http.MethodPost, base+"/api/v0/projects/"+web.String()+"/join", bob, "")
		var p struct {
			MemberRole *shared.Role `json:"member_role"`
			SortOrder  float64      `json:"sort_order"`
		}
		decodeAnswer(t, body, &p)
		if status != http.StatusOK || p.MemberRole == nil || *p.MemberRole != want || p.SortOrder != 65535 {
			t.Fatalf("bob's joining = %d %s; want 200, his role %d, at 65535", status, body, want)
		}
	}

	before := time.Now().Truncate(time.Microsecond)
	join(shared.RoleMember)
	first := membershipOf(t, pool, web, bobID)
	var prefs struct {
		sortOrder float64
		by        uuid.UUID
		at        time.Time
	}
	if err := pool.QueryRow(context.Background(), `SELECT sort_order, created_by_id, created_at FROM project_user_properties
		WHERE project_id = $1 AND user_id = $2 AND deleted_at IS NULL`, web, bobID).Scan(&prefs.sortOrder, &prefs.by, &prefs.at); err != nil {
		t.Fatal(err)
	}
	if first.role != shared.RoleMember || !first.active || first.by != bobID || first.at != first.created || first.at.Before(before) ||
		first.at.After(time.Now()) || prefs.sortOrder != 65535 || prefs.by != bobID || prefs.at != first.at {
		t.Fatalf("bob's new membership %+v, display settings %+v; want him an active member and at 65535, both by him within the request",
			first, prefs)
	}

	for _, tt := range []struct {
		name      string
		was       shared.Role // the ended membership's role
		workspace shared.Role // his workspace role
		add       bool        // alice adds him as a guest, rather than he joins
		want      shared.Role
	}{
		{"a guest before joins", shared.RoleGuest, shared.RoleMember, false, shared.RoleGuest},
		{"an admin before joins", shared.RoleAdmin, shared.RoleMember, false, shared.RoleMember},
		{"an admin before is added as a guest", shared.RoleAdmin, shared.RoleMember, true, shared.RoleGuest},
		{"a member before, now a workspace admin, joins", shared.RoleMember, shared.RoleAdmin, false, shared.RoleMember},
	} {
		if tag, err := pool.Exec(context.Background(), "UPDATE project_members SET role = $3, is_active = false WHERE project_id = $1 AND member_id = $2",
			web, bobID, tt.was); err != nil || tag.RowsAffected() != 1 {
			t.Fatalf("%s: ending bob's membership = %v, %v", tt.name, tag, err)
		}
		if tt.workspace != shared.RoleMember {
			var id uuid.UUID
			if err := pool.QueryRow(context.Background(), "SELECT id FROM workspace_members WHERE workspace_id = $1 AND member_id = $2", acme,
				bobID).Scan(&id); err != nil {
				t.Fatal(err)
			}
			if status, body := call(t, contract, http.MethodPatch, base+"/api/v0/workspace-members/"+id.String(), alice,
				fmt.Sprintf(`{"role":%d}`, tt.workspace)); status != http.StatusOK {
				t.Fatalf("%s: bob's workspace role = %d %s", tt.name, status, body)
			}
		}
		before := time.Now().Truncate(time.Microsecond)
		by := bobID
		if tt.add {
			by = aliceID
			if status, body := call(t, contract, http.MethodPost, base+"/api/v0/projects/"+web.String()+"/members", alice,
				`{"members":[{"member_id":"`+bobID.String()+`","role":5}]}`); status != http.StatusCreated {
				t.Fatalf("%s: alice's adding = %d %s", tt.name, status, body)
			}
		} else {
			join(tt.want)
		}
		got := membershipOf(t, pool, web, bobID)
		if got.id != first.id || got.role != tt.want || !got.active || got.by != by || got.at.Before(before) || got.at.After(time.Now()) ||
			got.created != first.created {
			t.Errorf("%s: bob's membership %+v; want %s restored as %d, by %s within the request", tt.name, got, first.id, tt.want, by)
		}
	}
}
````

`server/internal/bootstrap/project_write_locks_test.go`（修改，7 处）：

````old server/internal/bootstrap/project_write_locks_test.go
// operationId: the request on the project, by alice.
````
````new server/internal/bootstrap/project_write_locks_test.go
// operationId: the request on the project, by alice unless byTarget.
````

````old server/internal/bootstrap/project_write_locks_test.go
	targets [2]string
````
````new server/internal/bootstrap/project_write_locks_test.go
	targets [2]string
	// byTarget is set when the target sends the write: a joining.
	byTarget bool
````

````old server/internal/bootstrap/project_write_locks_test.go
		want: http.StatusCreated, targets: [2]string{"bob", "carol"}},
````
````new server/internal/bootstrap/project_write_locks_test.go
		want: http.StatusCreated, targets: [2]string{"bob", "carol"}},
	{op: "joinProject", method: http.MethodPost, path: "/api/v0/projects/%s/join", want: http.StatusOK, targets: [2]string{"dave", "erin"},
		byTarget: true},
````

````old server/internal/bootstrap/project_write_locks_test.go
	ids, aliceID := map[string]uuid.UUID{}, accountID(t, contract, base, alice)
````
````new server/internal/bootstrap/project_write_locks_test.go
	tokens, ids, aliceID := map[string]string{}, map[string]uuid.UUID{}, accountID(t, contract, base, alice)
````

````old server/internal/bootstrap/project_write_locks_test.go
				ids[name] = accountID(t, contract, base, registerAccount(t, contract, base, name+"@example.com").AccessToken)
````
````new server/internal/bootstrap/project_write_locks_test.go
				tokens[name] = registerAccount(t, contract, base, name+"@example.com").AccessToken
				ids[name] = accountID(t, contract, base, tokens[name])
````

````old server/internal/bootstrap/project_write_locks_test.go
			target, body := w.targets[phase], w.body
````
````new server/internal/bootstrap/project_write_locks_test.go
			target, token, body := w.targets[phase], alice, w.body
````

````old server/internal/bootstrap/project_write_locks_test.go
			req := newRequest(t, w.method, base+fmt.Sprintf(w.path, project), alice, []byte(body))
````
````new server/internal/bootstrap/project_write_locks_test.go
			if w.byTarget {
				token = tokens[target]
			}
			req := newRequest(t, w.method, base+fmt.Sprintf(w.path, project), token, []byte(body))
````

- [ ] **Step 6: 测试、lint、前端检查**

Run: `go -C server test -count=1 ./internal/modules/project/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix$|TestThePermissionMatrixCoversEveryOperation|TestMatrixViolationsCatchesEach|TestEveryColumnCallsAsARegisteredAccount|TestEveryActionHasARuleAndEveryRuleAnAction|TestAPIRoutesAreTheContractsOperations|TestOperationsThatNeedATokenAnswer401WithoutOne|TestARestoredMembershipGivesNoMoreThanItHad|TestEachWriteOnAProjectSharesItsWorkspaceFirst' ./internal/bootstrap/`
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

- [ ] **Step 7: 提交**

```bash
git add api/modules/project.yaml api/openapi.yaml server/internal/bootstrap/permission_matrix_members_test.go server/internal/bootstrap/permission_matrix_seed_test.go server/internal/bootstrap/project_members_test.go server/internal/bootstrap/project_write_locks_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/project/adapter/http/handler.go server/internal/modules/project/adapter/http/handler_test.go server/internal/modules/project/adapter/http/join_test.go server/internal/modules/project/adapter/http/projects.go server/internal/modules/project/app/clock_test.go server/internal/modules/project/app/join_project.go server/internal/modules/project/app/join_project_test.go server/internal/modules/project/app/ports.go server/internal/modules/project/domain/actions.go server/internal/modules/project/domain/member.go server/internal/modules/project/domain/member_test.go server/internal/modules/project/module.go api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P4b): joinProject

POST /api/v0/projects/{project_id}/join, for the workspace's admins and
members who see the project. In one transaction it takes Locks' path
with the caller as its target: the workspace FOR SHARE, his workspace
membership FOR SHARE, then the project FOR NO KEY UPDATE; it decides, refuses a workspace guest by set before his membership of the
project is read, and makes him a member with his workspace role, or
restores his ended membership with the lesser of its role and that one,
his display settings at 65535. An active member is left as he is.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| 规则只给项目的成员 | `TestPermissionMatrix` | 组合 |
| 规则给工作区的每个有效成员：成员加入私密项目 | `TestPermissionMatrix`；P2（Task 15 起） | 组合；端到端 |
| 工作区访客加入公开项目 | `TestPermissionMatrix`；P2（Task 15 起） | 组合；端到端 |
| `CanJoin` 按大小比较 | `TestCanJoin`（只在单元一层：数据库的 CHECK 不容许三种以外的角色，spec 第 3 节第 7 条） | 单元 |
| 工作区角色在"已是有效成员"之后查：是项目成员的工作区访客加入 | `TestPermissionMatrix`（PG 一格）；P2（Task 15 起） | 组合；端到端 |
| 加入的显示设置不在 65535 | `TestARestoredMembershipGivesNoMoreThanItHad`；P2（Task 15 起） | 组合；端到端 |
| 契约不声明工作区访客得到的 `forbidden` | `TestPermissionMatrix`、`TestJoinProject`（HTTP，`apitest.Main`） | 组合；单元 |
| 不锁他在工作区的成员关系（`write` 没有 `targets`） | `TestJoinProject`（调用记录）；`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（等 Ops 时不持有 erin 在 acme 的成员关系）；`TestADemotionAndTheProjectSidesGrowthSerialize`（Task 14 起） | 单元；组合 |
| 只有加入不锁工作区 | `TestEachWriteOnAProjectSharesItsWorkspaceFirst`；`TestADemotionAndTheProjectSidesGrowthSerialize`、`TestAWorkspacesDeletionWaitsForTheWritesOnItsProjects`、`TestAddingSeveralMembersAndDeletingTheWorkspaceSerialize`（Task 14 起） | 组合 |
| `project.New` 给加入不开事务的事务管理器 | `TestAGrowthRefusedAtItsCommitLeavesNoRow`、`TestADemotionAndTheProjectSidesGrowthSerialize`（Task 14 起） | 组合 |
| `project.New` 给加入固定在 2000 年的时钟 | `TestARestoredMembershipGivesNoMoreThanItHad` | 组合 |
| `project.New` 给加入谁都当作管理员放行的 `Authorizer`（经它自己的 `Locks`） | `TestPermissionMatrix` | 组合 |
| `project.New` 给加入一个 `Locks`，它的成员端口什么都不锁、答谁都是工作区成员 | `TestADemotionAndTheProjectSidesGrowthSerialize`（Task 14 起） | 组合 |
| 没有调用者时也读 | `TestJoinProjectRefuses` | 单元 |
| 项目的工作区、工作区的锁、他的成员关系的锁、项目的锁和判定、成员关系、写、回答、提交的失败被吞掉；找工作区的失败答成 404；工作区的锁的失败答成找到；他的锁重试一次 | `TestJoinProjectReturnsEachFailure` | 单元 |
| 增长用不加锁读到的工作区、不用锁确认过的 | 没有测试能发现：`Locks` 在项目的锁之下确认工作区，没有写能把项目移到别的工作区，两者总相同（等价变异，spec 第 3 节第 10 条；确认本身由 `TestUpdateProjectRefuses` 的"moved"发现） | — |
| handler 吞掉用例的失败 | `TestJoinProject`（HTTP） | 单元 |

**Done when:** 矩阵两行（13 格）通过；加入的角色按 9.1 的表、按集合和顺序表；工作区访客在读成员关系之前被拒；七个项目级的写都在 `TestEachWriteOnAProjectSharesItsWorkspaceFirst` 里。

---

### Task 14: 交错 17；降级与项目级的写、与删除项目；删除工作区与项目级的写；写在事务的连接上

**Files:**
- Create: `server/internal/bootstrap/interleaving_deletion_test.go`、`server/internal/bootstrap/interleaving_growth_test.go`、`server/internal/bootstrap/interleaving_writes_test.go`、`server/internal/bootstrap/project_connection_test.go`
- Modify: `server/internal/bootstrap/demotion_test.go`、`server/internal/bootstrap/project_deletion_test.go`、`server/internal/bootstrap/project_members_test.go`、`server/internal/modules/project/adapter/postgres/demote_test.go`

**Interfaces:** 只有测试。
- `bootstrap/interleaving_growth_test.go`：`growthRace`（acme：管理员 alice、成员 bob；alice 的公开项目 Web，bob 在其中的成员关系按需已结束，SQL 代替 P5 的移出）；`gatedShares`（工作区的 `WorkspaceMembers`：工作区和目标的成员关系都已 `FOR SHARE` 之后在 gate 等待，增长在锁项目之前停下）；`demotedHolding`（降级在写了成员关系之后、项目一步之前停下）；`webFree`（`FOR UPDATE NOWAIT` 探测 Web 没有被持有）；`membershipTimes`。等待由 `pgtest.WaitForLockWaitOn` 证明在哪张表上（M3 设计 3.6 约定二，方案 E）：每个交错的第二方都等在 `workspaces`（工作区一侧的连带持有工作区行的 `FOR NO KEY UPDATE`，项目级的写持有它的 `FOR SHARE`），只有同一个项目上的两个写等在 `projects`。
- `bootstrap/interleaving_writes_test.go`：`gatedAuthorizer`（判定之后、在写的事务里、锁之后在 gate 等待）；`bobAdministersWeb`（bob 是 Web 的管理员，Ops 是 acme 的另一个项目）；四个写（修改、归档、恢复、改设置）各两个顺序，探测 `workspaces`；两个写在同一个项目上，探测 `projects`；两个项目上的写不互等。
- `bootstrap/interleaving_deletion_test.go`：`deleteAcme`（alice 经工作区的用例删除 acme，连带照 `bootstrap` 接上，系统的时钟）；`deletedWithAcme`（每一行都由删除在它的一个时刻、由 alice 最后写入，且不早于建立）；`answeredOrWaiting`。
- `bootstrap/project_connection_test.go`：`project.New` 和 `Authorizer` 照 `bootstrap` 接在只有一个连接的池上，请求 3 秒截止，每次等待 5 秒看门狗。
- `demotion_test.go`：`refusingCommits(t, pool, table)`：延迟的约束触发器，拒绝插入或修改了 `table` 的事务的提交（P4a 只拒绝 `workspace_members` 的修改）；`TestAProjectDeletionRefusedAtItsCommitChangesNoRow` 改用它。

**Tests:**
- `TestADemotionAndTheProjectSidesGrowthSerialize`（交错 17，八个子测试：加入、添加 × 新的、已结束的成员关系 × 两个顺序）：增长先：它持有 acme 和 bob 在 acme 的成员关系 `FOR SHARE`、在锁 Web 之前等待；降级等 acme 的行，什么都不持有；增长提交之后，降级的项目一步找到他在 Web 的成员关系并改为访客，时刻在降级持有 acme 之后读（gate 放开之后，M3 设计 3.3）。降级先：它持有 acme 的 `FOR NO KEY UPDATE` 和写过的成员关系行；增长等 acme 的行，然后读到他是 acme 的访客：加入答 404，添加作成员答 422 `members[0].role`；已结束的成员关系由降级改为访客的。两个顺序中，第二方等待时 `webFree`（P4a 的 F-M2 顺序：增长先取成员关系、后取 Web，降级先写成员关系、后取他的项目）；他在 Web 的成员关系最后写入的时刻不早于建立的时刻（预检的 L1：方案 E 之前增长先时这里倒退）。
- `TestADemotionAndAProjectsDeletionSerialize`（两个顺序）：删除先持有 acme `FOR SHARE` 和 Web，在判定之后等待；降级等 acme 的行，然后它的项目一步找到 Web 已删除、把它留在外面，他的成员关系照删除时的成员角色删除。降级先：它持有 acme 和（项目一步锁住的）Web；删除等 acme 的行，然后删除他已改为访客的成员关系。
- `TestAProjectWriteAndADemotionSerialize`（四个写 × 两个顺序）：bob 是 Web 的管理员，alice 把他改为 acme 的访客。写先：持有 acme `FOR SHARE` 和 Web（`FOR NO KEY UPDATE`，改设置是 `FOR SHARE`），判定之后等待；降级等 acme 的行，然后把他改为 Web 的访客。降级先：他的写等 acme 的行，然后按降级提交的事实判定：访客不能修改、归档、恢复（403），可以改自己的设置。
- `TestTwoWritesOnAProjectSerialize`（预检的 L2）：bob 归档 Web，判定之后等待；他对 Web 的修改与之共享 acme、等 Web 的行（探测 `projects`）；归档提交之后修改按已归档判定，答 409 `project.archived`。`FOR SHARE` 的项目锁下两者都持有 Web、各自的修改等对方（40P01）。
- `TestWritesOnTwoProjectsOfAWorkspaceDoNotWait`：bob 归档 Web、判定之后等待，持有 acme 和 Web；alice 同时修改 Ops，5 秒之内答 200。工作区的锁比 `FOR SHARE` 强时她的修改等归档。
- `TestAWorkspacesDeletionWaitsForTheWritesOnItsProjects`（`interleaving_deletion_test.go`，预检的 L1、F-M3 的两个窗口）：bob 修改 Web，或 bob 加入 Web（新建成员关系和显示设置），判定之后等待，持有 acme `FOR SHARE`；alice 删除 acme，等 acme 的行（探测 `workspaces`）。写提交之后，Web 和写写下的每一行都在删除的一个时刻、由 alice 删除和最后写入，且不早于建立；Web 的删除时刻不早于修改答复里的 `updated_at`。
- `TestAddingSeveralMembersAndDeletingTheWorkspaceSerialize`（`interleaving_deletion_test.go`，预检的 M1）：carol、dave 是 acme 的成员，carol 的成员关系 id 较小（添加按 id 先锁她），dave 的行在表里在前（删除工作区的成员关系一步先遇到他）。carol 加入 Ops、判定之后等待，持有 acme 和她的成员关系；alice 删除 acme，等 acme 的行；alice 把 carol、dave 添加进 Web：与加入共享 acme 和两人的成员关系，在删除等待时答 201。加入提交之后删除完成，三条新的成员关系在它的一个时刻删除；没有 40P01。添加若不先取 acme 的 `FOR SHARE`，删除会持有 acme、改了 dave 的行再等 carol 的，添加持有 carol 的、等 dave 的：环，40P01（测试让添加的那一次死锁检查在加入结束之前过去，由删除发现它）。
- `TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（P4a 的 L3、brief 移交第 1 条的 `az-project-outside-tx`）：一个连接的池上 alice 建 Ops、修改 Web、归档、恢复、改设置、添加 bob（恢复他已结束的成员关系），carol 新加入，alice 删除两个项目；经池而不是事务发出的语句会等第二个连接、请求在截止时失败。
- `TestAGrowthRefusedAtItsCommitLeavesNoRow`（`project_members_test.go`）：bob 先不是 Web 的成员，再是没有显示设置的已结束成员；每次先拒绝成员关系的提交、再拒绝显示设置的提交，alice 的添加和他的加入答 500，成员关系和显示设置都不变；放开之后他加入。
- `TestLockMemberProjectsLeavesOutAProjectDeletedWhileItWaited`（`adapter/postgres/demote_test.go`）：另一个事务持有 Web `FOR NO KEY UPDATE` 并软删除它；`LockMemberProjects` 等它，提交之后重新求值 `deleted_at`，只返回 Ops。

- [ ] **Step 1: 提交被拒的组合测试**

`server/internal/bootstrap/demotion_test.go`（修改，6 处）：

````old server/internal/bootstrap/demotion_test.go
// refusingCommits makes the commit of every transaction that updated a
// workspace membership fail, until restore runs, or the test ends: a
````
````new server/internal/bootstrap/demotion_test.go
// refusingCommits makes the commit of every transaction that inserted or
// updated a row of table fail, until restore runs, or the test ends: a
````

````old server/internal/bootstrap/demotion_test.go
func refusingCommits(t *testing.T, pool *pgxpool.Pool) (restore func()) {
````
````new server/internal/bootstrap/demotion_test.go
func refusingCommits(t *testing.T, pool *pgxpool.Pool, table string) (restore func()) {
````

````old server/internal/bootstrap/demotion_test.go
	exec(`CREATE CONSTRAINT TRIGGER refuse_commit AFTER UPDATE ON workspace_members DEFERRABLE INITIALLY DEFERRED
````
````new server/internal/bootstrap/demotion_test.go
	exec(`CREATE CONSTRAINT TRIGGER refuse_commit AFTER INSERT OR UPDATE ON ` + table + ` DEFERRABLE INITIALLY DEFERRED
````

````old server/internal/bootstrap/demotion_test.go
		exec("DROP TRIGGER IF EXISTS refuse_commit ON workspace_members")
````
````new server/internal/bootstrap/demotion_test.go
		exec("DROP TRIGGER IF EXISTS refuse_commit ON " + table)
````

````old server/internal/bootstrap/demotion_test.go

	restore = refusingCommits(t, pool)
	status, body = patch()
````
````new server/internal/bootstrap/demotion_test.go

	restore = refusingCommits(t, pool, "workspace_members")
	status, body = patch()
````

````old server/internal/bootstrap/demotion_test.go

	restore = refusingCommits(t, pool)
	status, body = accept()
````
````new server/internal/bootstrap/demotion_test.go

	restore = refusingCommits(t, pool, "workspace_members")
	status, body = accept()
````

`server/internal/bootstrap/project_deletion_test.go`（修改，1 处）：

````old server/internal/bootstrap/project_deletion_test.go
	for _, sql := range []string{
		`CREATE FUNCTION refuse_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'the commit is refused'; END $$`,
		`CREATE CONSTRAINT TRIGGER refuse_commit AFTER UPDATE ON states DEFERRABLE INITIALLY DEFERRED
			FOR EACH ROW EXECUTE FUNCTION refuse_commit()`,
	} {
		if _, err := pool.Exec(context.Background(), sql); err != nil {
			t.Fatal(err)
		}
	}
````
````new server/internal/bootstrap/project_deletion_test.go
	refusingCommits(t, pool, "states")
````

`server/internal/bootstrap/project_members_test.go`（修改，1 处）：

````old server/internal/bootstrap/project_members_test.go
	}
}

````
````new server/internal/bootstrap/project_members_test.go
	}
}

// The project side's growth is one transaction (M3 design 3.6), the one
// of the TxManager project.New is given: a growth refused at its commit,
// after every statement ran, leaves no row, which a statement run in a
// transaction of its own would have outlived. bob, acme's member, is first
// no member of alice's Web, then an ended one without display settings
// there (P5's removal ends it; SQL stands in): each time, with the commits
// of memberships refused, then those of display settings, which the growth
// writes last, alice's adding him and his joining answer 500 and change no
// membership and no display settings. Once commits are allowed again, he
// joins.
func TestAGrowthRefusedAtItsCommitLeavesNoRow(t *testing.T) {
	contract, base, pool, alice, aliceID, web, _ := twoProjects(t)
	bob := registerAccount(t, contract, base, "bob@example.com").AccessToken
	bobID := accountID(t, contract, base, bob)
	inWorkspaceOf(t, pool, web, bobID, aliceID, shared.RoleMember)
	rows := func() string {
		t.Helper()
		var s string
		if err := pool.QueryRow(context.Background(), `SELECT
			coalesce((SELECT string_agg(role || ' ' || is_active || ' ' || updated_at, ', ') FROM project_members
			          WHERE project_id = $1 AND member_id = $2), 'no membership') || '; ' ||
			coalesce((SELECT string_agg(sort_order::text, ', ') FROM project_user_properties WHERE project_id = $1 AND user_id = $2),
			         'no settings')`, web, bobID).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	add := func() (int, string) {
		return call(t, contract, http.MethodPost, base+"/api/v0/projects/"+web.String()+"/members", alice,
			`{"members":[{"member_id":"`+bobID.String()+`","role":15}]}`)
	}
	join := func() (int, string) {
		return call(t, contract, http.MethodPost, base+"/api/v0/projects/"+web.String()+"/join", bob, "")
	}
	grow := []struct {
		name string
		send func() (int, string)
	}{{"alice's adding", add}, {"bob's joining", join}}
	for _, ended := range []bool{false, true} {
		if ended {
			if _, err := pool.Exec(context.Background(), `INSERT INTO project_members (id, workspace_id, project_id, member_id, role,
				is_active, created_by_id, updated_by_id) SELECT $1, workspace_id, id, $2, 15, false, $3, $3 FROM projects WHERE id = $4`,
				uuid.NewV7(), bobID, aliceID, web); err != nil {
				t.Fatal(err)
			}
		}
		before := rows()
		for _, table := range []string{"project_members", "project_user_properties"} {
			restore := refusingCommits(t, pool, table)
			for _, g := range grow {
				if status, body := g.send(); status != http.StatusInternalServerError {
					t.Errorf("%s, ended %v, with the commits of %s refused = %d %s, want 500", g.name, ended, table, status, body)
				}
				if got := rows(); got != before {
					t.Errorf("%s, ended %v, refused at its commit of %s: bob's rows %s, want %s", g.name, ended, table, got, before)
				}
			}
			restore()
		}
	}
	if status, body := join(); status != http.StatusOK {
		t.Errorf("bob's joining once commits are allowed = %d %s, want 200", status, body)
	}
}

````

- [ ] **Step 2: 交错**

`server/internal/bootstrap/interleaving_growth_test.go`（新文件，398 行）：

````file server/internal/bootstrap/interleaving_growth_test.go
package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	projectpg "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	projectdomain "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The interleaving 17 of M3 design 9.3, and a demotion against deleteProject,
// on a real database: the project side through the project module as
// bootstrap wires it (project.New, behind the API), the demotion through
// workspace's use case, both with the Authorizer as bootstrap wires it. A
// gate inside the first side's transaction holds it open at a lock the
// second side needs; pgtest.WaitForLockWaitOn proves that the second side
// waits on that table's row before the gate opens: nothing else runs on
// the database, and the table is the one design 3.6 says the second side
// meets first. Every wait has a deadline.

// growthRace is a database with acme, whose admin is alice and whose
// member is bob, and alice's public project Web, of which bob has an ended
// membership as a member when ended is set: P5's removal ends one, and SQL
// stands in for it.
type growthRace struct {
	race
	bob, bobIn, web uuid.UUID
}

func newGrowthRace(t *testing.T, ended bool) growthRace {
	t.Helper()
	r := growthRace{race: newRace(t), bob: uuid.NewV7(), bobIn: uuid.NewV7(), web: uuid.NewV7()}
	ctx, now := context.Background(), time.Now()
	users := identitypg.New(r.pool)
	if err := users.CreateUser(ctx, identityapp.NewUser{ID: r.bob, Email: "bob@example.com", PasswordHash: "x", DisplayName: "bob", Now: now}); err != nil {
		t.Fatal(err)
	}
	if err := users.CreateDefaultProfile(ctx, uuid.NewV7(), r.bob, now); err != nil {
		t.Fatal(err)
	}
	workspaces := workspacepg.New(r.pool)
	w, err := workspaces.CreateWorkspace(ctx, workspaceapp.WorkspaceRow{ID: uuid.NewV7(), Name: "Acme", Slug: "acme", Timezone: "UTC",
		CreatedBy: r.alice, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	for id, m := range map[uuid.UUID]struct {
		user uuid.UUID
		role shared.Role
	}{uuid.NewV7(): {r.alice, shared.RoleAdmin}, r.bobIn: {r.bob, shared.RoleMember}} {
		if err := workspaces.CreateMember(ctx, workspaceapp.MemberRow{ID: id, WorkspaceID: w.ID, MemberID: m.user, Role: m.role, CreatedBy: r.alice,
			Now: now}); err != nil {
			t.Fatal(err)
		}
	}
	projects := projectpg.New(r.pool)
	if err := projects.CreateProject(ctx, projectapp.ProjectRow{ID: r.web, WorkspaceID: w.ID, Name: "Web", Identifier: "WEB",
		Network: projectdomain.NetworkPublic, Timezone: "UTC", CreatedBy: r.alice, Now: now}); err != nil {
		t.Fatal(err)
	}
	members := []uuid.UUID{r.alice}
	if ended {
		members = append(members, r.bob)
	}
	for _, user := range members {
		if err := projects.CreateMember(ctx, projectapp.MemberRow{ID: uuid.NewV7(), WorkspaceID: w.ID, ProjectID: r.web, MemberID: user,
			Role: map[uuid.UUID]shared.Role{r.alice: shared.RoleAdmin, r.bob: shared.RoleMember}[user], CreatedBy: r.alice, Now: now}); err != nil {
			t.Fatal(err)
		}
	}
	if ended {
		if tag, err := r.pool.Exec(ctx, "UPDATE project_members SET is_active = false WHERE project_id = $1 AND member_id = $2", r.web, r.bob); err != nil ||
			tag.RowsAffected() != 1 {
			t.Fatalf("ending bob's membership of Web: %v, %v", tag, err)
		}
	}
	return r
}

// authorizer is access's Authorizer as bootstrap wires it, on r's pool.
func (r growthRace) authorizer() shared.Authorizer {
	return authorizerOn(r.pool)
}

// authorizerOn is access's Authorizer as bootstrap wires it, on pool.
func authorizerOn(pool *pgxpool.Pool) shared.Authorizer {
	return access.New(access.Deps{WorkspaceRoles: workspace.Provide(pool).WorkspaceRoles,
		ProjectAccess: accessProjects{projects: project.Provide(pool).ProjectAccess}})
}

// gatedShares is workspace's WorkspaceMembers; with a gate, the growth
// waits at it once its workspace and its targets' memberships of it are
// held FOR SHARE, before it locks the project (M3 design 3.6 conventions 2
// and 3).
type gatedShares struct {
	projectapp.WorkspaceMembers
	gate *gate
}

func (m gatedShares) ShareMembers(ctx context.Context, workspaceID uuid.UUID, userIDs []uuid.UUID) (map[uuid.UUID]shared.Role, error) {
	roles, err := m.WorkspaceMembers.ShareMembers(ctx, workspaceID, userIDs)
	if err != nil || m.gate == nil {
		return roles, err
	}
	return roles, m.gate.wait(ctx)
}

// growth is bob's joining Web, or alice's adding him to it as a member,
// through the project module as bootstrap wires it, its WorkspaceMembers
// gated by g when g is not nil: a function that sends the request and
// answers it with its answer, which another goroutine may call.
func (r growthRace) growth(t *testing.T, add bool, g *gate) func() (*http.Request, *httptest.ResponseRecorder) {
	t.Helper()
	route := newProjectRoute(t, r.pool, r.authorizer(), gatedShares{workspace.Provide(r.pool).WorkspaceMembers, g})
	web := "/api/v0/projects/" + r.web.String()
	if add {
		return func() (*http.Request, *httptest.ResponseRecorder) {
			return route.send(http.MethodPost, web+"/members", r.alice, `{"members":[{"member_id":"`+r.bob.String()+`","role":15}]}`)
		}
	}
	return func() (*http.Request, *httptest.ResponseRecorder) {
		return route.send(http.MethodPost, web+"/join", r.bob, "")
	}
}

// join makes bob Web's member: his joining it, through the project module.
func (r growthRace) join(t *testing.T) {
	t.Helper()
	if _, rec := r.growth(t, false, nil)(); rec.Code != http.StatusOK {
		t.Fatalf("bob's joining Web = %d %s", rec.Code, rec.Body)
	}
}

// demote is alice's change of bob's role in acme to guest, over members
// and cascade, on the system's clock: its time is read when it reads it.
func (r growthRace) demote(ctx context.Context, members workspaceapp.MemberUpdater, cascade workspaceapp.ProjectCascade) error {
	_, err := workspaceapp.NewUpdateWorkspaceMember(members, cascade, workspaceProfiles{profiles: identity.Provide(r.pool).PublicProfiles},
		r.authorizer(), postgres.NewTxManager(r.pool, 2*time.Second), clock.System{}).
		Execute(shared.WithActor(ctx, shared.Actor{UserID: r.alice}), r.bobIn, shared.RoleGuest)
	return err
}

// standing is bob's role in acme, then his membership of Web: its role,
// "ended" when it is, "deleted" when Web is; "none" when he has none.
func (r growthRace) standing(t *testing.T) string {
	t.Helper()
	var s string
	if err := r.pool.QueryRow(context.Background(), `SELECT (SELECT role::text FROM workspace_members WHERE id = $1) || ', Web ' ||
		coalesce((SELECT role || CASE WHEN is_active THEN '' ELSE ' ended' END || CASE WHEN deleted_at IS NULL THEN '' ELSE ' deleted' END
		          FROM project_members WHERE project_id = $2 AND member_id = $3), 'none')`, r.bobIn, r.web, r.bob).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

// webFree reports whether no transaction holds Web's row: a FOR UPDATE
// NOWAIT of it answers lock_not_available (55P03) at once while one does.
func (r growthRace) webFree(t *testing.T) bool {
	t.Helper()
	_, err := r.pool.Exec(context.Background(), "SELECT 1 FROM projects WHERE id = $1 FOR UPDATE NOWAIT", r.web)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "55P03" {
		return false
	}
	if err != nil {
		t.Fatal(err)
	}
	return true
}

// demotedHolding stops a role change after its write of the membership,
// holding the membership's row, before the projects' step.
type demotedHolding struct {
	*workspacepg.Store
	gate *gate
}

func (m demotedHolding) UpdateMemberRole(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) (
	workspacedomain.Membership, error) {
	updated, err := m.Store.UpdateMemberRole(ctx, id, role, by, now)
	if err != nil {
		return updated, err
	}
	return updated, m.gate.wait(ctx)
}

// A demotion to guest and the project side's growth, bob's joining Web or
// alice's adding him to it, serialize on acme's row (M3 design 3.6
// conventions 2, 3 and 6, 9.3's interleaving 17), in both orders, for a
// new membership of Web and for his ended one. The growth first: it holds
// acme FOR SHARE and his membership of acme FOR SHARE, and waits at its
// gate before it locks Web; the demotion waits for acme's row, holding
// nothing. Once the growth has committed, the demotion's step over his
// projects, a statement run after its write of his membership, finds his
// membership of Web and makes it a guest's, at the demotion's time, read
// once it held acme: after the gate opened (design 3.3). The demotion
// first: it holds acme FOR NO KEY UPDATE and his membership's row after its
// write; the growth waits for acme's row, then reads him a guest of acme:
// his joining is refused as one who does not see Web (404), alice's adding
// him as a member as a role a guest may not have (422 members[0].role); an
// ended membership is a guest's, by the demotion. In either order, while
// the second side waits for acme's row, no transaction holds Web (FOR
// UPDATE NOWAIT): the growth takes his membership of acme before Web, and
// the demotion his membership before his projects (P4a's F-M2 order: a
// growth that locked Web first would hold it at its gate; a demotion that
// locked his projects first would hold Web, of which he has an ended
// membership). His membership of Web, once there, was last written no
// earlier than it was made.
func TestADemotionAndTheProjectSidesGrowthSerialize(t *testing.T) {
	contract := apitest.Load(t)
	for _, add := range []bool{false, true} {
		for _, ended := range []bool{false, true} {
			for _, growthFirst := range []bool{true, false} {
				name := fmt.Sprintf("join, ended %v, growth first %v", ended, growthFirst)
				if add {
					name = fmt.Sprintf("add, ended %v, growth first %v", ended, growthFirst)
				}
				t.Run(name, func(t *testing.T) {
					r := newGrowthRace(t, ended)
					before := "15, Web none"
					if ended {
						before = "15, Web 15 ended"
					}
					if got := r.standing(t); got != before {
						t.Fatalf("before: %s, want %s", got, before)
					}
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					g, cascade := newGate(), project.New(project.Deps{Pool: r.pool}).Cascade()
					var req *http.Request
					var rec *httptest.ResponseRecorder
					var grew, demoted <-chan error
					if growthFirst {
						grow := r.growth(t, add, g)
						grew = run(func() error { req, rec = grow(); return nil })
						held(t, ctx, g, grew, "the growth")
						demoted = run(func() error { return r.demote(ctx, workspacepg.New(r.pool), cascade) })
					} else {
						demoted = run(func() error { return r.demote(ctx, demotedHolding{workspacepg.New(r.pool), g}, cascade) })
						held(t, ctx, g, demoted, "the demotion")
						grow := r.growth(t, add, nil)
						grew = run(func() error { req, rec = grow(); return nil })
					}
					pgtest.WaitForLockWaitOn(t, r.pool, "workspaces", 5*time.Second)
					if !r.webFree(t) {
						t.Error("Web is held while the second side waits for acme's row; want it locked after that row")
					}
					opened := time.Now()
					close(g.open)

					demotion := result(t, ctx, demoted, "the demotion")
					if err := result(t, ctx, grew, "the growth"); err != nil {
						t.Fatal(err)
					}
					contract.CheckResponse(t, req, rec.Result())
					want, grown := "5, Web 5", rec.Code == map[bool]int{false: http.StatusOK, true: http.StatusCreated}[add]
					if !growthFirst {
						want, grown = map[bool]string{false: "5, Web none", true: "5, Web 5 ended"}[ended], refusedAsAGuest(rec, add)
					}
					if got := r.standing(t); demotion != nil || !grown || got != want {
						t.Errorf("the demotion = %v, the growth = %d %s, bob %s; want the demotion done, the growth %s, bob %s", demotion, rec.Code,
							rec.Body, got, map[bool]string{true: "done", false: "refused"}[growthFirst], want)
					}
					if made, written := r.membershipTimes(t); written.Before(made) || (growthFirst && written.Before(opened)) {
						t.Errorf("bob's membership of Web made at %v, last written at %v, the gate opened at %v; want it written last by the "+
							"demotion, at a time read once it held acme", made, written, opened)
					}
				})
			}
		}
	}
}

// membershipTimes are bob's membership of Web's created_at and updated_at.
func (r growthRace) membershipTimes(t *testing.T) (made, written time.Time) {
	t.Helper()
	if err := r.pool.QueryRow(context.Background(), "SELECT created_at, updated_at FROM project_members WHERE project_id = $1 AND member_id = $2",
		r.web, r.bob).Scan(&made, &written); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal(err)
	}
	return made, written
}

// refusedAsAGuest reports whether rec is the growth's refusal of bob as
// acme's guest: his joining, 404 project.not_found; alice's adding him as a
// member, 422 members[0].role not_allowed alone.
func refusedAsAGuest(rec *httptest.ResponseRecorder, add bool) bool {
	var problem struct {
		Code   string `json:"code"`
		Errors []struct {
			Field string `json:"field"`
			Code  string `json:"code"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		return false
	}
	if !add {
		return rec.Code == http.StatusNotFound && problem.Code == "project.not_found"
	}
	return rec.Code == http.StatusUnprocessableEntity && problem.Code == "validation_failed" && len(problem.Errors) == 1 &&
		problem.Errors[0].Field == "members[0].role" && problem.Errors[0].Code == "not_allowed"
}

// gatedDemoter stops the demotion's step over the projects after it has
// locked them, before its write.
type gatedDemoter struct {
	*projectpg.Store
	gate *gate
}

func (d gatedDemoter) DemoteMemberships(ctx context.Context, projectIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error {
	if err := d.gate.wait(ctx); err != nil {
		return err
	}
	return d.Store.DemoteMemberships(ctx, projectIDs, userID, by, now)
}

// A demotion to guest and the deletion of a project its member is in
// serialize on acme's row, in both orders (M3 design 3.6): bob is a member
// of Web. The deletion first holds acme FOR SHARE and Web FOR NO KEY
// UPDATE, and waits after its decision; the demotion waits for acme's row,
// then its step over his projects finds Web deleted and leaves it out, so
// his membership is deleted as it was, a member's. The demotion first holds
// acme and, after locking it, Web; the deletion waits for acme's row, then
// deletes his membership, a guest's by then.
func TestADemotionAndAProjectsDeletionSerialize(t *testing.T) {
	contract := apitest.Load(t)
	for _, deletionFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("deletion first %v", deletionFirst), func(t *testing.T) {
			r := newGrowthRace(t, false)
			r.join(t)
			if got := r.standing(t); got != "15, Web 15" {
				t.Fatalf("before: %s, want 15, Web 15", got)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g, store := newGate(), projectpg.New(r.pool)
			auth, cascade := r.authorizer(), projectapp.NewCascade(store, store)
			if deletionFirst {
				auth = gatedAuthorizer{Authorizer: auth, action: projectdomain.ActionDelete, gate: g}
			} else {
				cascade = projectapp.NewCascade(store, gatedDemoter{store, g})
			}
			route := newProjectRoute(t, r.pool, auth, workspace.Provide(r.pool).WorkspaceMembers)
			var req *http.Request
			var rec *httptest.ResponseRecorder
			deletion := func() error {
				req, rec = route.send(http.MethodDelete, "/api/v0/projects/"+r.web.String(), r.alice, "")
				return nil
			}
			demotion := func() error { return r.demote(ctx, workspacepg.New(r.pool), cascade) }
			first, second := deletion, demotion
			if !deletionFirst {
				first, second = demotion, deletion
			}
			firstDone := run(first)
			held(t, ctx, g, firstDone, "the first")
			secondDone := run(second)
			pgtest.WaitForLockWaitOn(t, r.pool, "workspaces", 5*time.Second)
			close(g.open)

			want := map[bool]string{true: "5, Web 15 deleted", false: "5, Web 5 deleted"}[deletionFirst]
			a, b := result(t, ctx, firstDone, "the first"), result(t, ctx, secondDone, "the second")
			contract.CheckResponse(t, req, rec.Result())
			if got := r.standing(t); a != nil || b != nil || rec.Code != http.StatusNoContent || got != want {
				t.Errorf("the first = %v, the second = %v, the deletion = %d %s, bob %s; want both done, the deletion 204, bob %s", a, b, rec.Code,
					rec.Body, got, want)
			}
		})
	}
}
````

`server/internal/bootstrap/interleaving_writes_test.go`（新文件，257 行）：

````file server/internal/bootstrap/interleaving_writes_test.go
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"uuid"

	projectpg "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	projectdomain "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A write on a project decides under its locks, its workspace's FOR SHARE
// and its project's, which it holds until it commits (M3 design 3.6
// convention 2, 6.7): against a demotion to guest, which holds the
// workspace FOR NO KEY UPDATE; against another write on the project, which
// holds the project; and beside a write on another project of the
// workspace, which shares the workspace. The writes run as bootstrap wires
// them (project.New), behind the API, so the locks are the ones of the
// transaction project.New is given.

// gatedAuthorizer is an Authorizer whose decision on action, once made,
// waits at the gate: inside the write's transaction, after its lock.
type gatedAuthorizer struct {
	shared.Authorizer
	action shared.Action
	gate   *gate
}

func (a gatedAuthorizer) Authorize(ctx context.Context, actor shared.Actor, action shared.Action, t shared.Target) (shared.Grant, error) {
	grant, err := a.Authorizer.Authorize(ctx, actor, action, t)
	if err == nil && action == a.action {
		if err := a.gate.wait(ctx); err != nil {
			return shared.Grant{}, err
		}
	}
	return grant, err
}

// Bob, acme's member, is the admin of Web; alice makes him acme's guest
// while he changes Web, archives it, unarchives it, or changes his display
// settings in it, in both orders. His write first: it holds acme FOR SHARE
// and Web, FOR NO KEY UPDATE or, for his settings, FOR SHARE, and waits
// after its decision; the demotion waits for acme's row, then makes him
// Web's guest. The demotion first: it holds acme FOR NO KEY UPDATE and, by
// its step over his projects, Web; his write waits for acme's row, then
// decides on what the demotion committed: Web's guest may not change,
// archive or unarchive it (403 forbidden), and may change his own
// settings.
func TestAProjectWriteAndADemotionSerialize(t *testing.T) {
	writes := []struct {
		name, method, path, body string
		action                   shared.Action // the write's decision, where its gate holds it
		archived                 bool          // Web is archived before the write
		done                     string        // the statement of whether the write is done, by $1 Web's id and $2 bob's
		guests                   bool          // Web's guest may write it
	}{
		{"updateProject", http.MethodPatch, "/api/v0/projects/%s", `{"name":"Site"}`, projectdomain.ActionUpdate, false,
			"SELECT name = 'Site' FROM projects WHERE id = $1 AND $2::uuid IS NOT NULL", false},
		{"archiveProject", http.MethodPost, "/api/v0/projects/%s/archive", "", projectdomain.ActionArchive, false,
			"SELECT archived_at IS NOT NULL FROM projects WHERE id = $1 AND $2::uuid IS NOT NULL", false},
		{"unarchiveProject", http.MethodPost, "/api/v0/projects/%s/unarchive", "", projectdomain.ActionUnarchive, true,
			"SELECT archived_at IS NULL FROM projects WHERE id = $1 AND $2::uuid IS NOT NULL", false},
		{"updateProjectPreferences", http.MethodPatch, "/api/v0/me/projects/%s/preferences", `{"sort_order":1}`,
			projectdomain.ActionPreferencesUpdate, false,
			"SELECT sort_order = 1 FROM project_user_properties WHERE project_id = $1 AND user_id = $2 AND deleted_at IS NULL", true},
	}
	contract := apitest.Load(t)
	for _, w := range writes {
		for _, writeFirst := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s, the write first %v", w.name, writeFirst), func(t *testing.T) {
				r := newGrowthRace(t, false)
				r.join(t)
				if _, err := r.pool.Exec(context.Background(), "UPDATE project_members SET role = 20 WHERE project_id = $1 AND member_id = $2",
					r.web, r.bob); err != nil {
					t.Fatal(err)
				}
				if _, err := r.pool.Exec(context.Background(), "UPDATE projects SET archived_at = CASE WHEN $2 THEN now() END WHERE id = $1",
					r.web, w.archived); err != nil {
					t.Fatal(err)
				}
				if got := r.standing(t); got != "15, Web 20" {
					t.Fatalf("before: %s, want 15, Web 20", got)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				g, store := newGate(), projectpg.New(r.pool)
				auth := r.authorizer()
				if writeFirst {
					auth = gatedAuthorizer{Authorizer: auth, action: w.action, gate: g}
				}
				route := newProjectRoute(t, r.pool, auth, workspace.Provide(r.pool).WorkspaceMembers)
				var req *http.Request
				var rec *httptest.ResponseRecorder
				write := func() error {
					req, rec = route.send(w.method, fmt.Sprintf(w.path, r.web), r.bob, w.body)
					return nil
				}
				var wrote, demoted <-chan error
				if writeFirst {
					wrote = run(write)
					held(t, ctx, g, wrote, "the write")
					demoted = run(func() error { return r.demote(ctx, workspacepg.New(r.pool), projectapp.NewCascade(store, store)) })
				} else {
					demoted = run(func() error {
						return r.demote(ctx, workspacepg.New(r.pool), projectapp.NewCascade(store, gatedDemoter{store, g}))
					})
					held(t, ctx, g, demoted, "the demotion")
					wrote = run(write)
				}
				pgtest.WaitForLockWaitOn(t, r.pool, "workspaces", 5*time.Second)
				close(g.open)

				demotion := result(t, ctx, demoted, "the demotion")
				if err := result(t, ctx, wrote, "the write"); err != nil {
					t.Fatal(err)
				}
				contract.CheckResponse(t, req, rec.Result())
				var done bool
				if err := r.pool.QueryRow(context.Background(), w.done, r.web, r.bob).Scan(&done); err != nil {
					t.Fatal(err)
				}
				wantDone, want := writeFirst || w.guests, http.StatusOK
				if !wantDone {
					want = http.StatusForbidden
				}
				if got := r.standing(t); demotion != nil || rec.Code != want || done != wantDone || got != "5, Web 5" {
					t.Errorf("the demotion = %v, the write = %d %s (done %v), bob %s; want the demotion done, the write %d (done %v), "+
						"bob 5, Web 5", demotion, rec.Code, rec.Body, done, got, want, wantDone)
				}
			})
		}
	}
}

// bobAdministersWeb is a growthRace in which bob, acme's member, has joined
// Web and is its admin (SQL stands in for P5's role change), and Ops is
// another project of acme, of which alice is the admin.
func bobAdministersWeb(t *testing.T) (r growthRace, ops uuid.UUID) {
	t.Helper()
	r = newGrowthRace(t, false)
	r.join(t)
	if _, err := r.pool.Exec(context.Background(), "UPDATE project_members SET role = 20 WHERE project_id = $1 AND member_id = $2", r.web,
		r.bob); err != nil {
		t.Fatal(err)
	}
	ops = uuid.NewV7()
	var acme uuid.UUID
	if err := r.pool.QueryRow(context.Background(), "SELECT workspace_id FROM projects WHERE id = $1", r.web).Scan(&acme); err != nil {
		t.Fatal(err)
	}
	store := projectpg.New(r.pool)
	if err := errors.Join(store.CreateProject(context.Background(), projectapp.ProjectRow{ID: ops, WorkspaceID: acme, Name: "Ops", Identifier: "OPS",
		Network: projectdomain.NetworkPublic, Timezone: "UTC", CreatedBy: r.alice, Now: time.Now()}),
		store.CreateMember(context.Background(), projectapp.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: ops, MemberID: r.alice,
			Role: shared.RoleAdmin, CreatedBy: r.alice, Now: time.Now()})); err != nil {
		t.Fatal(err)
	}
	return r, ops
}

// Two writes on one project serialize on its row (M3 design 3.6 convention
// 2: a write that changes the project row holds it FOR NO KEY UPDATE): bob
// archives Web and waits after his decision, holding acme FOR SHARE and
// Web; his change of Web takes acme's row beside it and waits for Web's.
// Once the archive commits, the change decides on Web archived and answers
// 409 project.archived. Under FOR SHARE both would hold Web, and each one's
// UPDATE would wait for the other (40P01).
func TestTwoWritesOnAProjectSerialize(t *testing.T) {
	contract := apitest.Load(t)
	r, _ := bobAdministersWeb(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	g := newGate()
	gated := newProjectRoute(t, r.pool, gatedAuthorizer{Authorizer: r.authorizer(), action: projectdomain.ActionArchive, gate: g},
		workspace.Provide(r.pool).WorkspaceMembers)
	route := newProjectRoute(t, r.pool, r.authorizer(), workspace.Provide(r.pool).WorkspaceMembers)
	web := "/api/v0/projects/" + r.web.String()
	var archiveReq, updateReq *http.Request
	var archiveRec, updateRec *httptest.ResponseRecorder
	archived := run(func() error {
		archiveReq, archiveRec = gated.send(http.MethodPost, web+"/archive", r.bob, "")
		return nil
	})
	held(t, ctx, g, archived, "the archive")
	updated := run(func() error {
		updateReq, updateRec = route.send(http.MethodPatch, web, r.bob, `{"name":"Site"}`)
		return nil
	})
	pgtest.WaitForLockWaitOn(t, r.pool, "projects", 5*time.Second)
	close(g.open)

	if err := errors.Join(result(t, ctx, archived, "the archive"), result(t, ctx, updated, "the change")); err != nil {
		t.Fatal(err)
	}
	contract.CheckResponse(t, archiveReq, archiveRec.Result())
	contract.CheckResponse(t, updateReq, updateRec.Result())
	if archiveRec.Code != http.StatusOK || updateRec.Code != http.StatusConflict || !strings.Contains(updateRec.Body.String(), `"project.archived"`) {
		t.Errorf("the archive = %d %s, the change = %d %s; want 200 and 409 project.archived", archiveRec.Code, archiveRec.Body, updateRec.Code,
			updateRec.Body)
	}
}

// Writes on two projects of one workspace do not wait for each other: each
// holds the workspace FOR SHARE, which the other shares (M3 design 3.6
// convention 2). Bob archives Web and waits after his decision, holding
// acme and Web; meanwhile alice changes Ops, and her change answers 200.
// Under a stronger lock of the workspace her change would wait for the
// archive.
func TestWritesOnTwoProjectsOfAWorkspaceDoNotWait(t *testing.T) {
	contract := apitest.Load(t)
	r, ops := bobAdministersWeb(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	g := newGate()
	gated := newProjectRoute(t, r.pool, gatedAuthorizer{Authorizer: r.authorizer(), action: projectdomain.ActionArchive, gate: g},
		workspace.Provide(r.pool).WorkspaceMembers)
	route := newProjectRoute(t, r.pool, r.authorizer(), workspace.Provide(r.pool).WorkspaceMembers)
	archived := run(func() error {
		gated.send(http.MethodPost, "/api/v0/projects/"+r.web.String()+"/archive", r.bob, "")
		return nil
	})
	held(t, ctx, g, archived, "the archive")
	type answer struct {
		req *http.Request
		rec *httptest.ResponseRecorder
	}
	changed := make(chan answer, 1)
	go func() {
		req, rec := route.send(http.MethodPatch, "/api/v0/projects/"+ops.String(), r.alice, `{"name":"Site"}`)
		changed <- answer{req, rec}
	}()
	select {
	case a := <-changed:
		contract.CheckResponse(t, a.req, a.rec.Result())
		if a.rec.Code != http.StatusOK {
			t.Errorf("alice's change of Ops while the archive holds acme = %d %s, want 200", a.rec.Code, a.rec.Body)
		}
	case <-time.After(5 * time.Second):
		t.Error("alice's change of Ops did not answer within 5s while the archive of Web held acme: it waited for it")
	}
	close(g.open)
	if err := result(t, ctx, archived, "the archive"); err != nil {
		t.Fatal(err)
	}
}
````

`server/internal/bootstrap/interleaving_deletion_test.go`（新文件，267 行）：

````file server/internal/bootstrap/interleaving_deletion_test.go
package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	projectdomain "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A workspace's deletion against the writes on its projects, on a real
// database: the writes through the project module as bootstrap wires it,
// behind the API, the deletion through workspace's use case with project's
// cascade as bootstrap wires it, each on the system's clock. Every write on
// a project holds its workspace FOR SHARE (M3 design 3.6 convention 2); the
// deletion holds it FOR NO KEY UPDATE and reads its time once it does, so
// it waits for the writes in flight and writes one time, after theirs, into
// every row it deletes (3.3). Every wait has a deadline.

// deleteAcme is alice's deletion of acme, on the system's clock.
func (r growthRace) deleteAcme(ctx context.Context) error {
	return workspaceapp.NewDeleteWorkspace(workspacepg.New(r.pool), project.New(project.Deps{Pool: r.pool}).Cascade(), r.authorizer(),
		postgres.NewTxManager(r.pool, 2*time.Second), clock.System{}, slog.New(slog.DiscardHandler)).
		Execute(shared.WithActor(ctx, shared.Actor{UserID: r.alice}), "acme")
}

// deletedWithAcme counts the rows statement reads, as deleted_at,
// updated_at, updated_by_id and created_at, and fails the test for each
// that acme's deletion did not write last: deleted and last written at the
// deletion's time, by alice, and not before it was made.
func (r growthRace) deletedWithAcme(t *testing.T, statement string, args ...any) int {
	t.Helper()
	ctx := context.Background()
	var acme *time.Time
	if err := r.pool.QueryRow(ctx, "SELECT deleted_at FROM workspaces WHERE slug = 'acme'").Scan(&acme); err != nil || acme == nil {
		t.Fatalf("acme's deletion time: %v, %v", acme, err)
	}
	rows, err := r.pool.Query(ctx, statement, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	n := 0
	for ; rows.Next(); n++ {
		var deleted *time.Time
		var written, made time.Time
		var by uuid.UUID
		if err := rows.Scan(&deleted, &written, &by, &made); err != nil {
			t.Fatal(err)
		}
		if deleted == nil || !deleted.Equal(*acme) || !written.Equal(*acme) || by != r.alice || deleted.Before(made) {
			t.Errorf("row %d: deleted at %v, written last at %v by %v, made at %v; want deleted and written last at acme's deletion's time %v, "+
				"by alice, not before it was made", n, deleted, written, by, made, *acme)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return n
}

// A workspace's deletion waits for the writes on its projects in flight and
// deletes their rows at its one time, after theirs (pre-flight L1): bob,
// Web's admin, changes Web; or bob, acme's member, joins it, which makes
// his membership of it and his display settings in it. His write waits
// after its decision, holding acme FOR SHARE; alice's deletion of acme
// waits for acme's row. Once his write commits, the deletion deletes Web
// and the rows under it: Web, and each row his write made, deleted and last
// written at acme's deletion's time, by alice, and not before it was made;
// Web's time no earlier than his change's, which his answer gives. A
// deletion that went past acme's row would wait for Web's, or for his
// membership of acme, with a time read before his write committed.
func TestAWorkspacesDeletionWaitsForTheWritesOnItsProjects(t *testing.T) {
	contract := apitest.Load(t)
	for _, w := range []struct {
		name, method, path, body string
		action                   shared.Action
		rows                     string // the rows the write wrote, by $1 Web's id and $2 bob's
		want                     int
	}{
		{"updateProject", http.MethodPatch, "", `{"name":"Site"}`, projectdomain.ActionUpdate,
			"SELECT deleted_at, updated_at, updated_by_id, created_at FROM projects WHERE id = $1 AND $2::uuid IS NOT NULL", 1},
		{"joinProject", http.MethodPost, "/join", "", projectdomain.ActionJoin, `
			SELECT deleted_at, updated_at, updated_by_id, created_at FROM project_members WHERE project_id = $1 AND member_id = $2
			UNION ALL
			SELECT deleted_at, updated_at, updated_by_id, created_at FROM project_user_properties WHERE project_id = $1 AND user_id = $2`, 2},
	} {
		t.Run(w.name, func(t *testing.T) {
			var r growthRace
			if w.action == projectdomain.ActionUpdate {
				r, _ = bobAdministersWeb(t)
			} else {
				r = newGrowthRace(t, false)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			route := newProjectRoute(t, r.pool, gatedAuthorizer{Authorizer: r.authorizer(), action: w.action, gate: g},
				workspace.Provide(r.pool).WorkspaceMembers)
			var req *http.Request
			var rec *httptest.ResponseRecorder
			wrote := run(func() error {
				req, rec = route.send(w.method, "/api/v0/projects/"+r.web.String()+w.path, r.bob, w.body)
				return nil
			})
			held(t, ctx, g, wrote, "bob's write")
			deleted := run(func() error { return r.deleteAcme(ctx) })
			pgtest.WaitForLockWaitOn(t, r.pool, "workspaces", 5*time.Second)
			close(g.open)

			if err := errors.Join(result(t, ctx, wrote, "bob's write"), result(t, ctx, deleted, "the deletion")); err != nil {
				t.Fatal(err)
			}
			contract.CheckResponse(t, req, rec.Result())
			if rec.Code != http.StatusOK {
				t.Fatalf("bob's write = %d %s, want 200", rec.Code, rec.Body)
			}
			if n := r.deletedWithAcme(t, w.rows, r.web, r.bob); n != w.want {
				t.Errorf("%d rows of bob's write, want %d", n, w.want)
			}
			if w.action != projectdomain.ActionUpdate {
				return
			}
			var changed struct {
				UpdatedAt time.Time `json:"updated_at"`
			}
			var deletedAt time.Time
			if err := errors.Join(json.Unmarshal(rec.Body.Bytes(), &changed),
				r.pool.QueryRow(context.Background(), "SELECT deleted_at FROM projects WHERE id = $1", r.web).Scan(&deletedAt)); err != nil {
				t.Fatal(err)
			}
			if deletedAt.Before(changed.UpdatedAt) {
				t.Errorf("Web deleted at %v, before bob's change at %v", deletedAt, changed.UpdatedAt)
			}
		})
	}
}

// answeredOrWaiting returns true once done yields, or false once n backends
// of pool's database wait for a lock; it fails the test when neither has
// happened within 5s. It suits a database on which only the test's sides
// can wait.
func answeredOrWaiting(t *testing.T, pool *pgxpool.Pool, n int, done <-chan error) bool {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(10 * time.Millisecond) {
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
			return true
		default:
		}
		var waiting int
		if err := pool.QueryRow(context.Background(),
			"SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock'").Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting >= n {
			return false
		}
		if time.Now().After(deadline) {
			t.Fatalf("no answer, and %d backends waiting, within 5s; want an answer or %d waiting", waiting, n)
		}
	}
}

// Adding several members to a project and deleting its workspace serialize
// on the workspace's row, with no 40P01, in the schedule of the P4b
// pre-flight's M1: carol and dave are acme's members; carol's membership
// has the lesser id, so the adding's lock of its targets (in id order)
// meets hers first; dave's row comes first in the table and in member_id
// order, so the deletion's update of acme's memberships meets his first.
// Carol joins Ops and waits after her decision, holding acme and her
// membership of it FOR SHARE (her own growth elsewhere, convention 3).
// Alice's deletion of acme waits for acme's row. Alice adds carol and dave
// to Web: the adding shares acme's row, and both memberships, with the
// joining, and answers 201 while the deletion waits. Once the joining
// commits, the deletion deletes the three new memberships at its one time.
// Without the workspace's share, the deletion would hold acme, update
// dave's membership and wait for carol's; the adding would share carol's
// and wait for dave's; once the joining ended, the deletion would wait for
// the adding: a cycle, 40P01. The test lets the adding's one deadlock check
// pass before the joining ends, so that the deletion is the one to find it.
func TestAddingSeveralMembersAndDeletingTheWorkspaceSerialize(t *testing.T) {
	contract := apitest.Load(t)
	r, ops := bobAdministersWeb(t)
	ctx, now := context.Background(), time.Now()
	dave, carol := uuid.NewV7(), uuid.NewV7()
	carolIn, daveIn := uuid.NewV7(), uuid.NewV7()
	var acme uuid.UUID
	if err := r.pool.QueryRow(ctx, "SELECT id FROM workspaces WHERE slug = 'acme'").Scan(&acme); err != nil {
		t.Fatal(err)
	}
	users := identitypg.New(r.pool)
	for _, m := range []struct{ id, user uuid.UUID }{{daveIn, dave}, {carolIn, carol}} {
		if err := errors.Join(users.CreateUser(ctx, identityapp.NewUser{ID: m.user, Email: m.user.String() + "@example.com", PasswordHash: "x",
			DisplayName: "x", Now: now}), users.CreateDefaultProfile(ctx, uuid.NewV7(), m.user, now),
			workspacepg.New(r.pool).CreateMember(ctx, workspaceapp.MemberRow{ID: m.id, WorkspaceID: acme, MemberID: m.user, Role: shared.RoleMember,
				CreatedBy: r.alice, Now: now})); err != nil {
			t.Fatal(err)
		}
	}
	if carolIn.String() >= daveIn.String() || dave.String() >= carol.String() {
		t.Fatal("carol's membership's id is not the lesser, or dave's account's is not")
	}
	tctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	g := newGate()
	joining := newProjectRoute(t, r.pool, gatedAuthorizer{Authorizer: r.authorizer(), action: projectdomain.ActionJoin, gate: g},
		workspace.Provide(r.pool).WorkspaceMembers)
	adding := newProjectRoute(t, r.pool, r.authorizer(), workspace.Provide(r.pool).WorkspaceMembers)
	var joinRec, addRec *httptest.ResponseRecorder
	var addReq *http.Request
	joined := run(func() error {
		_, joinRec = joining.send(http.MethodPost, "/api/v0/projects/"+ops.String()+"/join", carol, "")
		return nil
	})
	held(t, tctx, g, joined, "carol's joining")
	deleted := run(func() error { return r.deleteAcme(tctx) })
	pgtest.WaitForLockWait(t, r.pool, 5*time.Second)
	added := run(func() error {
		addReq, addRec = adding.send(http.MethodPost, "/api/v0/projects/"+r.web.String()+"/members", r.alice,
			`{"members":[{"member_id":"`+carol.String()+`","role":15},{"member_id":"`+dave.String()+`","role":15}]}`)
		return nil
	})
	answered := answeredOrWaiting(t, r.pool, 2, added)
	if !answered {
		time.Sleep(1200 * time.Millisecond) // past the adding's one deadlock check (deadlock_timeout, 1s)
	}
	close(g.open)

	errs := []error{result(t, tctx, joined, "carol's joining"), result(t, tctx, deleted, "the deletion")}
	if !answered {
		errs = append(errs, result(t, tctx, added, "the adding"))
	}
	if err := errors.Join(errs...); err != nil {
		t.Fatal(err)
	}
	contract.CheckResponse(t, addReq, addRec.Result())
	if joinRec.Code != http.StatusOK || addRec.Code != http.StatusCreated {
		t.Errorf("carol's joining = %d %s, the adding = %d %s; want 200 and 201", joinRec.Code, joinRec.Body, addRec.Code, addRec.Body)
	}
	if n := r.deletedWithAcme(t, "SELECT deleted_at, updated_at, updated_by_id, created_at FROM project_members WHERE member_id = ANY ($1::uuid[])",
		[]uuid.UUID{carol, dave}); n != 3 {
		t.Errorf("%d memberships of carol's and dave's, want 3: hers of Ops and Web, his of Web", n)
	}
}
````

- [ ] **Step 3: 写在事务的连接上**

`server/internal/bootstrap/project_connection_test.go`（新文件，169 行）：

````file server/internal/bootstrap/project_connection_test.go
package bootstrap

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/ratelimit"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// tokenIsAccount accepts an account's id as its bearer token.
type tokenIsAccount struct{}

func (tokenIsAccount) Authenticate(ctx context.Context, token string) (context.Context, string, error) {
	id, err := uuid.Parse(token)
	if err != nil {
		return nil, "", shared.Unauthenticated()
	}
	return shared.WithActor(ctx, shared.Actor{UserID: id, SessionID: uuid.NewV7()}), "account:" + token, nil
}

// projectRoute is the project module as bootstrap wires it (project.New),
// on pool, with auth as its Authorizer and members as its WorkspaceMembers,
// mounted behind an API whose bearer token is an account's id and whose
// requests end after 3 seconds.
type projectRoute struct {
	router http.Handler
}

func newProjectRoute(t *testing.T, pool *pgxpool.Pool, auth shared.Authorizer, members projectapp.WorkspaceMembers) projectRoute {
	t.Helper()
	module := project.New(project.Deps{Pool: pool, Tx: postgres.NewTxManager(pool, 2*time.Second), Clock: clock.System{}, Authorizer: auth,
		Workspaces: projectWorkspaces{directory: workspace.Provide(pool).WorkspaceDirectory}, Members: members})
	logger := slog.New(slog.DiscardHandler)
	router := httpserver.NewRouter(logger)
	limit := ratelimit.New(time.Now).Bucket("test", ratelimit.Rate{PerMinute: 600, Burst: 100})
	api, err := httpserver.NewAPI(httpserver.APIConfig{Logger: logger, Authenticator: tokenIsAccount{}, MaxBodyBytes: 1 << 20,
		RequestTimeout: 3 * time.Second, IPv6PrefixLen: 64, Anonymous: limit, Authenticated: limit, AuthFailure: limit})
	if err != nil {
		t.Fatal(err)
	}
	module.Register(router, api)
	return projectRoute{router: router}
}

// send sends method path as caller, with body when it is not empty, and
// returns the request and its answer. It touches no test, so another
// goroutine may send it.
func (p projectRoute) send(method, path string, caller uuid.UUID, body string) (*http.Request, *httptest.ResponseRecorder) {
	var b io.Reader
	if body != "" {
		b = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, b)
	req.Header.Set("Authorization", "Bearer "+caller.String())
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	p.router.ServeHTTP(rec, req)
	return req, rec
}

// Each write on a project runs every statement on its transaction's
// connection (M3 design 3.6 convention 2, 6.7; P4a's L3): its locks, its
// reads, the facts its decision reads (ProjectFacts and the workspace
// roles), its writes and its answer. The project module and the Authorizer
// are wired as bootstrap wires them, on a pool of one connection: a
// statement sent through the pool rather than the transaction would wait
// for a second connection that never comes, and its request fail at the
// request's deadline. alice, acme's admin, creates Ops, changes Web,
// archives and unarchives it, changes her display settings in it and adds
// bob, whose ended membership she restores; carol joins it anew; alice
// deletes both projects.
func TestTheWritesOnAProjectRunOnTheirTransactionsConnection(t *testing.T) {
	r := newGrowthRace(t, true)
	carol := uuid.NewV7()
	ctx, now := context.Background(), time.Now()
	if err := identitypg.New(r.pool).CreateUser(ctx, identityapp.NewUser{ID: carol, Email: "carol@example.com", PasswordHash: "x",
		DisplayName: "carol", Now: now}); err != nil {
		t.Fatal(err)
	}
	var acme uuid.UUID
	if err := r.pool.QueryRow(ctx, "SELECT workspace_id FROM projects WHERE id = $1", r.web).Scan(&acme); err != nil {
		t.Fatal(err)
	}
	if err := workspacepg.New(r.pool).CreateMember(ctx, workspaceapp.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, MemberID: carol,
		Role: shared.RoleMember, CreatedBy: r.alice, Now: now}); err != nil {
		t.Fatal(err)
	}
	one, err := postgres.NewPool(ctx, config.DatabaseConfig{URL: r.pool.Config().ConnString(), MaxConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	// A statement on a context without the request's deadline would wait for the connection for ever, and so would
	// closing the pool: each has a deadline of its own.
	t.Cleanup(func() {
		closed := make(chan struct{})
		go func() { one.Close(); close(closed) }()
		select {
		case <-closed:
		case <-time.After(5 * time.Second):
			t.Error("the pool of one connection did not close within 5s: a transaction still holds its connection")
		}
	})
	route := newProjectRoute(t, one, authorizerOn(one), workspace.Provide(one).WorkspaceMembers)
	contract := apitest.Load(t)
	send := func(method, path string, caller uuid.UUID, body string, want int) string {
		t.Helper()
		type answer struct {
			req *http.Request
			rec *httptest.ResponseRecorder
		}
		done := make(chan answer, 1)
		go func() {
			req, rec := route.send(method, path, caller, body)
			done <- answer{req, rec}
		}()
		var a answer
		select {
		case a = <-done:
		case <-time.After(5 * time.Second):
			t.Fatalf("%s %s did not answer within 5s: a statement waits for the pool's one connection without the request's deadline", method,
				path)
		}
		req, rec := a.req, a.rec
		contract.CheckResponse(t, req, rec.Result())
		if rec.Code != want {
			t.Fatalf("%s %s = %d %s, want %d", method, path, rec.Code, rec.Body, want)
		}
		return rec.Body.String()
	}

	var created struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, send(http.MethodPost, "/api/v0/workspaces/acme/projects", r.alice, `{"name":"Ops","identifier":"OPS"}`, http.StatusCreated),
		&created)
	web := "/api/v0/projects/" + r.web.String()
	send(http.MethodPatch, web, r.alice, `{"name":"Site"}`, http.StatusOK)
	send(http.MethodPost, web+"/archive", r.alice, "", http.StatusOK)
	send(http.MethodPost, web+"/unarchive", r.alice, "", http.StatusOK)
	send(http.MethodPatch, "/api/v0/me/projects/"+r.web.String()+"/preferences", r.alice, `{"sort_order":1}`, http.StatusOK)
	send(http.MethodPost, web+"/members", r.alice, `{"members":[{"member_id":"`+r.bob.String()+`","role":15}]}`, http.StatusCreated)
	send(http.MethodPost, web+"/join", carol, "", http.StatusOK)
	send(http.MethodDelete, web, r.alice, "", http.StatusNoContent)
	send(http.MethodDelete, "/api/v0/projects/"+created.ID.String(), r.alice, "", http.StatusNoContent)
}
````

- [ ] **Step 4: 删除项目之后的降级（存储）**

`server/internal/modules/project/adapter/postgres/demote_test.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/demote_test.go
		t.Fatal("LockMemberProjects() did not end within 10s")
	}
}

````
````new server/internal/modules/project/adapter/postgres/demote_test.go
		t.Fatal("LockMemberProjects() did not end within 10s")
	}
}

// A project deleted while LockMemberProjects waits for its row is left out
// (M3 design 3.6, P4a spec 2.12): another transaction holds Web FOR NO KEY
// UPDATE, as deleteProject does, and soft-deletes it; LockMemberProjects,
// which found Web undeleted, waits for it; once the deletion commits, it
// evaluates deleted_at again on the row's newest version and returns Ops
// alone.
func TestLockMemberProjectsLeavesOutAProjectDeletedWhileItWaited(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	for _, p := range []uuid.UUID{web, ops} {
		seedMember(t, pool, acme, p, bob, 15, true)
	}
	deletion, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = deletion.Rollback(context.Background()) }()
	if _, err := deletion.Exec(context.Background(), "SELECT 1 FROM projects WHERE id = $1 FOR NO KEY UPDATE", web); err != nil {
		t.Fatal(err)
	}
	if _, err := deletion.Exec(context.Background(), "UPDATE projects SET deleted_at = now() WHERE id = $1", web); err != nil {
		t.Fatal(err)
	}
	type locked struct {
		ids []uuid.UUID
		err error
	}
	done := make(chan locked, 1)
	go func() {
		var l locked
		l.err = postgres.NewTxManager(pool, 10*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
			var err error
			l.ids, err = s.LockMemberProjects(ctx, acme, bob)
			return err
		})
		done <- l
	}()
	pgtest.WaitForLockWaitOn(t, pool, "projects", 10*time.Second)
	if err := deletion.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case l := <-done:
		if l.err != nil || !slices.Equal(l.ids, []uuid.UUID{ops}) {
			t.Errorf("LockMemberProjects() = %v, %v; want Ops (%s) alone, Web (%s) deleted while it waited", l.ids, l.err, ops, web)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("LockMemberProjects() did not end within 10s")
	}
}

````

- [ ] **Step 5: 测试、lint**

Run: `go -C server test -count=1 -run 'TestLockMemberProjectsLeavesOutAProjectDeletedWhileItWaited' ./internal/modules/project/adapter/postgres/`
Expected: `ok`。

Run: `go -C server test -count=1 -run 'TestTheWritesOnAProjectRunOnTheirTransactionsConnection|TestAGrowthRefusedAtItsCommitLeavesNoRow|TestAProjectDeletionRefusedAtItsCommitChangesNoRow|TestDemotingToGuestDemotesInTheWorkspacesProjects|TestAcceptingAsAGuestAgainDemotesInTheWorkspacesProjects' ./internal/bootstrap/`
Expected: `ok`。

Run: `go -C server test -count=5 -race -run 'TestADemotionAndTheProjectSidesGrowthSerialize$|TestADemotionAndAProjectsDeletionSerialize$|TestAProjectWriteAndADemotionSerialize$|TestTwoWritesOnAProjectSerialize$|TestWritesOnTwoProjectsOfAWorkspaceDoNotWait$|TestAWorkspacesDeletionWaitsForTheWritesOnItsProjects$|TestAddingSeveralMembersAndDeletingTheWorkspaceSerialize$|TestEachWriteOnAProjectSharesItsWorkspaceFirst$' ./internal/bootstrap/`
Expected: `ok`；输出中没有 `40P01`（原型：25 秒，40 个顶层 PASS、100 个子测试，没有 40P01、没有 DATA RACE）。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 6: 提交**

```bash
git add server/internal/bootstrap/demotion_test.go server/internal/bootstrap/interleaving_deletion_test.go server/internal/bootstrap/interleaving_growth_test.go server/internal/bootstrap/interleaving_writes_test.go server/internal/bootstrap/project_connection_test.go server/internal/bootstrap/project_deletion_test.go server/internal/bootstrap/project_members_test.go server/internal/modules/project/adapter/postgres/demote_test.go
```
```bash
git commit -m "test(M3/P4b): interleaving 17, the project writes against a demotion, one connection

On a real database, as bootstrap wires the project module, every
cascade over a workspace's projects meets the writes on its projects at
the workspace's row (M3 design 3.6 convention 2): the project side's
growth and a demotion to guest, a demotion and a project's deletion, each
write on a project and a demotion serialize there in both orders, no
transaction holding the project while the second side waits; a
workspace's deletion waits for the writes in flight and deletes their
rows at its one time (pre-flight L1); adding several members and
deleting the workspace serialize with no 40P01 (pre-flight M1). Two
writes on one project serialize on its row; writes on two projects of a
workspace do not wait. On a pool of one connection every write runs each
statement on its transaction's connection; a growth refused at its
commit leaves no row.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| 写的回答（`GetProject`）、`CreateMember`、`LowestSortOrder`、`ShareMembers`（目标在工作区的成员关系）、`ActiveRole`（判定读的工作区角色）各经连接池、在事务之外执行 | `TestTheWritesOnAProjectRunOnTheirTransactionsConnection` | 组合 |
| `ProjectFacts`（判定读的项目事实）经连接池、在事务之外（`az-project-outside-tx`） | `TestTheWritesOnAProjectRunOnTheirTransactionsConnection`、P4a 的 `TestProjectFactsReadsInTheTransaction` | 组合；存储 |
| 降级先锁他的项目、再写他的成员关系（P4a 的 F-M2） | `TestADemotionAndTheProjectSidesGrowthSerialize` | 组合 |
| 降级的 `LockMemberProjects` 取 `FOR SHARE` | P4a 的 `TestLockMemberProjectsLocksInIDOrder`、`TestDemotingAMemberToGuest`（存储；方案 E 之下组合一层看不到：每个项目级的写先等降级持有的工作区行，不再在项目行上与它相遇，spec 第 3 节第 15 条） | 存储 |
| `Locks` 不锁工作区，在 M1 的排程里 | `TestAddingSeveralMembersAndDeletingTheWorkspaceSerialize`（删除工作区答 40P01） | 组合 |
| 删除工作区在它的事务和锁之前读时钟 | `TestAWorkspacesDeletionWaitsForTheWritesOnItsProjects`（Web 的删除时刻早于修改的） | 组合 |
| 降级在它的事务和锁之前读时钟 | `TestADemotionAndTheProjectSidesGrowthSerialize`（成员关系的最后写入早于 gate 放开） | 组合 |
| 测试改坏：交错 17、删除项目的交错、写的交错、删除工作区的交错的探测等在方案 E 之前的表（`workspace_members`、`projects`），两个写的探测等在 `workspaces` | 各自的测试自己失败（等待在指名的表上，探测超时）：每个探测不会被别的等待满足 | 组合 |
| 测试改坏：交错 17 的 gate 移到 `ShareMembers` 之前 | 不失败，等价：方案 E 之下增长在 gate 之前已持有 acme 的 `FOR SHARE`，第二方照样等在 `workspaces` | — |
| 测试改坏：交错 17 去掉 `webFree`，同时加入先锁项目 | 不失败：方案 E 之下第二方等在 acme 上，两种顺序都不形成环；先锁项目由 `TestJoinProject`、`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 13）发现，`webFree` 是交错 17 里唯一核对它的地方 | — |
| `LockMemberProjects` 留下等锁时被删除的项目 | `TestLockMemberProjectsLeavesOutAProjectDeletedWhileItWaited` | 存储 |
| 交错 17 的 bob 是 acme 的管理员；写的交错中 bob 是 Web 的成员而不是管理员；恢复的那一次 Web 事先没有归档 | 各自的测试 | 组合 |

前面各 Task 表中标着"（Task 14 起）"的变异（事务之外、先判定后锁、锁的顺序、不锁工作区、不开事务的事务管理器、什么都不锁的成员端口）从本 Task 起由这些测试发现。

**Done when:** 七个交错测试和 `TestEachWriteOnAProjectSharesItsWorkspaceFirst` `-count=5 -race` 通过、没有 40P01；删除工作区以它的一个时刻删除项目级的写写下的行；一个连接的池上每个写都完成；提交被拒的增长、删除一行不留。

---

### Task 15: 端到端：P2、P3、P4、P8 的接口版本；W3 先删除一个项目

**Files:**
- Create: `e2e/stories/project/p2-visibility.spec.ts`、`e2e/stories/project/p3-project-settings.spec.ts`、`e2e/stories/project/p4-archive.spec.ts`、`e2e/stories/project/p8-project-preferences.spec.ts`
- Modify: `e2e/fixtures/api.ts`、`e2e/fixtures/assert/project.ts`、`e2e/fixtures/assert/workspace.ts`、`e2e/stories/workspace/w3-workspace-settings.spec.ts`

**Interfaces:**
- Produces（spec 2.11，M3 设计 2、9.6）：`api.ts` 的 `ProjectUpdate`、`ProjectMember`、`ProjectMemberNew`、`ProjectPreferences`、`ProjectPreferencesUpdate` 类型（取自生成的客户端）、`addProjectMembers(api, token, projectId, members)`、`amidAnotherWorkspace`（在调用者另一个工作区的两个项目之间建故事的项目，一个在前、一个在后：丢掉项目 id 的查询无论数据库先读哪一行，都读到别的工作区的项目）；`assert/project.ts` 的 `Membership`、`expectMember(db, projectId, email, want)`（他在项目中未删除的成员关系、显示设置的位置、最后写的人；都是项目所在工作区的行，设置不晚于成员关系）、`expectProjectDeleted(db, projectId, adminEmail)`（项目由他删除，目录中每张指向 `projects` 的表的行都在同一时刻、由同一个账户删除，每张表都有这样的行，没有未删除的）；`expectProjectCreated` 只读未删除的项目（删除之后标识可以再用，brief 移交第 5 条）。
- `assert/workspace.ts`：`workspaceTables` 中项目的四张表 `deletedAlone: true`：W3 现在先删除 Old，它和它下面的行保留自己的删除时刻（brief 移交第 5 条）。
- 使用者：P10 的页面版本。

**Tests:**
- **P2 (API)**：管理员列出每个项目，成员列出公开的和自己的，访客只有自己的，已归档的只在要求时列出；成员以成员身份加入公开项目、位置 65535，再加入什么都不变；成员不能加入私密项目，访客不能加入公开项目，也不能加入他是成员的项目。
- **P3 (API)**：项目管理员添加一个成员和一个访客，成员列出他们和自己；管理员改每个设置（标识 `site` 存成 `SITE`），以成员为负责人和默认负责人；项目成员不能改；负责人或默认负责人是访客、不是成员，`archive_in` 13：都不改任何东西。
- **P4 (API)**：管理员归档项目：它离开列表、进入已归档的列表，不能修改（409）；恢复、再归档；然后删除：成员、显示设置、状态在同一时刻删除（`expectProjectDeleted`），之后每个操作都答 404，标识可以再用。
- **P8 (API)**：管理员把项目的默认标签页设为 modules，把 views 藏进"更多"，并把项目拖到侧边栏的第一位，设置保留；未知的标签页、藏起 work_items 都不改任何东西；他的成员的设置是成员自己的。
- **W3 (API)**：删除工作区之前先删除项目 Old（成员作负责人，四张表各有它的行）：Old 的行保留自己的时刻，其余的行带工作区的时刻。

- [ ] **Step 1: fixture**

`e2e/fixtures/api.ts`（修改，2 处）：

````old e2e/fixtures/api.ts
export type ProjectCreate = components["schemas"]["ProjectCreate"];
````
````new e2e/fixtures/api.ts
export type ProjectCreate = components["schemas"]["ProjectCreate"];
export type ProjectUpdate = components["schemas"]["ProjectUpdate"];
export type ProjectMember = components["schemas"]["ProjectMember"];
export type ProjectMemberNew = components["schemas"]["ProjectMemberNew"];
export type ProjectPreferences = components["schemas"]["ProjectPreferences"];
export type ProjectPreferencesUpdate = components["schemas"]["ProjectPreferencesUpdate"];
````

````old e2e/fixtures/api.ts
    throw new Error(`createProject ${body.identifier} answered 201 without the project`);
  }
  return data;
}

````
````new e2e/fixtures/api.ts
    throw new Error(`createProject ${body.identifier} answered 201 without the project`);
  }
  return data;
}

/**
 * Adds the accounts of members, each an active member of the project's workspace, to the project of projectId with
 * the bearer token given, an admin's of the project, and returns their memberships in that order. The one way a
 * project gets a member with a role of the admin's choice (M3 design 3.5).
 */
export async function addProjectMembers(
  api: Api,
  token: string,
  projectId: string,
  members: ProjectMemberNew[]
): Promise<ProjectMember[]> {
  const { data, error, response } = await api.POST("/api/v0/projects/{project_id}/members", {
    params: { path: { project_id: projectId } },
    body: { members },
    headers: bearer(token),
  });
  expect(response.status, `add members to ${projectId}: ${JSON.stringify(error)}`).toBe(201);
  if (!data) {
    throw new Error(`addProjectMembers to ${projectId} answered 201 without the members`);
  }
  return data.data;
}

/**
 * Runs make, which makes a story's projects, between two projects of another workspace of the caller of token, one
 * made before them and one after: a query that loses its project's id reads one of those two, whichever row the
 * database reads first, and a project of another workspace is not found. Returns what make returns.
 */
export async function amidAnotherWorkspace<T>(
  api: Api,
  token: string,
  testInfo: TestInfo,
  make: () => Promise<T>
): Promise<T> {
  const slug = slugFor(testInfo, "elsewhere");
  await createWorkspace(api, token, { name: "Elsewhere", slug });
  await createProject(api, token, slug, { name: "First", identifier: "FIRST" });
  const made = await make();
  await createProject(api, token, slug, { name: "Last", identifier: "LAST" });
  return made;
}

````

`e2e/fixtures/assert/project.ts`（修改，5 处）：

````old e2e/fixtures/assert/project.ts
 * P1, W3: the project of p.identifier in the workspace of slug holds p, is neither archived nor deleted, has no
 * work item numbered yet (last_issue_sequence 0) and is led by the account of leadEmail, or by no one when it is
 * null. Its members are exactly `members`, each an active admin (role 20) with his display settings in it at his
````
````new e2e/fixtures/assert/project.ts
 * P1, P4, W3: the undeleted project of p.identifier in the workspace of slug holds p, is not archived, has no work
 * item numbered yet (last_issue_sequence 0) and is led by the account of leadEmail, or by no one when it is null. A
 * deleted project's identifier is free again (M3 design 4.6), so a deleted project of it is not read. Its members are exactly `members`, each an active admin (role 20) with his display settings in it at his
````

````old e2e/fixtures/assert/project.ts
            p.archived_at, p.deleted_at, l.email AS lead, c.email AS creator,
````
````new e2e/fixtures/assert/project.ts
            p.archived_at, l.email AS lead, c.email AS creator,
````

````old e2e/fixtures/assert/project.ts
      WHERE w.slug = $1 AND p.identifier = $2`,
````
````new e2e/fixtures/assert/project.ts
      WHERE w.slug = $1 AND p.identifier = $2 AND p.deleted_at IS NULL`,
````

````old e2e/fixtures/assert/project.ts
    archived_at: null,
    deleted_at: null,
````
````new e2e/fixtures/assert/project.ts
    archived_at: null,
````

````old e2e/fixtures/assert/project.ts
  return counts;
}

````
````new e2e/fixtures/assert/project.ts
  return counts;
}

/** A membership of a project as expectMember reads it. */
export interface Membership {
  role: number;
  is_active: boolean;
  /** His place in his sidebar; null when he has no display settings in the project. */
  sort_order: number | null;
  /** The address of the account that wrote the membership last. */
  by: string;
}

/**
 * P2, P3: the account of email's undeleted membership of the project of projectId is want, or he has none when want
 * is null. The membership and his display settings in the project are rows of the project's workspace, and the
 * settings were written with the membership, when he became a member, or before it.
 */
export async function expectMember(
  db: Database,
  projectId: string,
  email: string,
  want: Membership | null
): Promise<void> {
  const rows = await db.query(
    `SELECT m.role, m.is_active, s.sort_order, b.email AS by,
            m.workspace_id = p.workspace_id AND (s.id IS NULL OR (s.workspace_id = p.workspace_id AND s.created_at <= m.updated_at))
              AS in_its_workspace
       FROM project_members m
       JOIN projects p ON p.id = m.project_id
       JOIN users u ON u.id = m.member_id
       JOIN users b ON b.id = m.updated_by_id
       LEFT JOIN project_user_properties s ON s.project_id = m.project_id AND s.user_id = m.member_id AND s.deleted_at IS NULL
      WHERE m.project_id = $1 AND u.email = $2 AND m.deleted_at IS NULL`,
    [projectId, email]
  );
  expect(rows, `the membership of ${email} in ${projectId}`).toEqual(
    want === null ? [] : [{ ...want, in_its_workspace: true }]
  );
}

/**
 * P4: the project of projectId is deleted by the account of adminEmail, and with it, at the same moment and by the
 * same account, every row under it: of each table whose foreign key names projects (the catalog's list, so a table a
 * later phase adds is read too), by its project_id. Each table has such a row, and none is left undeleted.
 */
export async function expectProjectDeleted(db: Database, projectId: string, adminEmail: string): Promise<void> {
  const [project] = await db.query(
    `SELECT p.deleted_at IS NOT NULL AS deleted, u.email AS by FROM projects p JOIN users u ON u.id = p.updated_by_id WHERE p.id = $1`,
    [projectId]
  );
  expect(project, `the project ${projectId}`).toEqual({ deleted: true, by: adminEmail });
  const tables = await db.query<{ name: string }>(
    `SELECT DISTINCT c.conrelid::regclass::text AS name FROM pg_constraint c
      WHERE c.contype = 'f' AND c.confrelid = 'projects'::regclass ORDER BY 1`
  );
  expect(tables.length, "the tables under projects").toBeGreaterThan(0);
  // The database compares the moments: a Date holds milliseconds, a timestamptz microseconds.
  const rows = await Promise.all(
    tables.map(async ({ name }) => {
      const [counts] = await db.query<{ with_it: number; other: number }>(
        `SELECT count(*) FILTER (WHERE t.deleted_at = p.deleted_at AND t.updated_by_id = p.updated_by_id)::int AS with_it,
                count(*) FILTER (WHERE t.deleted_at IS DISTINCT FROM p.deleted_at
                                    OR t.updated_by_id IS DISTINCT FROM p.updated_by_id)::int AS other
           FROM ${name} t JOIN projects p ON p.id = t.project_id WHERE p.id = $1`,
        [projectId]
      );
      return { table: name, deletedWithIt: (counts?.with_it ?? 0) > 0, other: counts?.other };
    })
  );
  expect(rows, `the rows under ${projectId}`).toEqual(
    tables.map(({ name }) => ({ table: name, deletedWithIt: true, other: 0 }))
  );
}

````

`e2e/fixtures/assert/workspace.ts`（修改，5 处）：

````old e2e/fixtures/assert/workspace.ts
 * deletedAlone tells whether a row of the table can be deleted on its own before the workspace, keeping that
 * moment: an invitation can, when it is accepted or deleted (M3 design 3.8); no row of the other tables can yet.
````
````new e2e/fixtures/assert/workspace.ts
 * deletedAlone tells whether W3 deletes rows of the table on their own before the workspace, which keep that
 * moment: an invitation, when it is accepted or deleted (M3 design 3.8); a project, its memberships, its members'
 * display settings and its states, when the project is deleted (P4b). No row of the other tables is deleted alone.
````

````old e2e/fixtures/assert/workspace.ts
  { table: "projects", deletedAlone: false },
  { table: "project_members", deletedAlone: false },
  { table: "project_user_properties", deletedAlone: false },
  { table: "states", deletedAlone: false },
````
````new e2e/fixtures/assert/workspace.ts
  { table: "projects", deletedAlone: true },
  { table: "project_members", deletedAlone: true },
  { table: "project_user_properties", deletedAlone: true },
  { table: "states", deletedAlone: true },
````

````old e2e/fixtures/assert/workspace.ts
 * such a row; none is left undeleted; and only a table whose rows can be deleted alone has rows deleted earlier
 * than the workspace, so every row of the others carries the workspace's moment.
````
````new e2e/fixtures/assert/workspace.ts
 * such a row; none is left undeleted; a table whose rows W3 deletes alone has rows deleted earlier, which kept
 * their moment, and every row of the others carries the workspace's.
````

````old e2e/fixtures/assert/workspace.ts
        deletedEarlier: counts?.earlier,
````
````new e2e/fixtures/assert/workspace.ts
        deletedEarlier: (counts?.earlier ?? 0) > 0,
````

````old e2e/fixtures/assert/workspace.ts
      deletedEarlier: deletedAlone ? expect.any(Number) : 0,
````
````new e2e/fixtures/assert/workspace.ts
      deletedEarlier: deletedAlone,
````

- [ ] **Step 2: 故事**

`e2e/stories/project/p2-visibility.spec.ts`（新文件，107 行）：

````file e2e/stories/project/p2-visibility.spec.ts
import {
  addProjectMembers,
  amidAnotherWorkspace,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
  type Project,
} from "../../fixtures/api";
import { expectMember } from "../../fixtures/assert/project";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// P2, the projects' list and who sees them, and joining (M3 design 2, 3.4,
// 3.5, 3.19). The page version, with the "join the project" screen, comes
// with the projects' pages (P10).

/** The names of the projects of slug that the caller of token lists, in their order; archived: the archived ones. */
async function listed(api: Api, token: string, slug: string, archived = false): Promise<string[]> {
  const { data, error, response } = await api.GET("/api/v0/workspaces/{slug}/projects", {
    params: { path: { slug }, query: { archived } },
    headers: bearer(token),
  });
  expect(response.status, `list ${slug}'s projects: ${JSON.stringify(error)}`).toBe(200);
  return (data?.data ?? []).map((p) => p.name);
}

/** The answer of POST /api/v0/projects/{project_id}/join: its status, and the project or the problem's code. */
async function join(
  api: Api,
  token: string,
  id: string
): Promise<{ status: number; project?: Project; code?: string }> {
  const { data, error, response } = await api.POST("/api/v0/projects/{project_id}/join", {
    params: { path: { project_id: id } },
    headers: bearer(token),
  });
  return data ? { status: response.status, project: data } : { status: response.status, code: error?.code };
}

test("P2 (API): the admin lists every project, a member the public ones and his own, a guest his own, none the archived ones unless asked; a member joins a public project as a member at 65535, and again changes nothing; a member cannot join a private project, nor a guest a public one, nor a guest the project he is a member of", async ({
  api,
  db,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
  const memberEmail = emailFor(testInfo, "member");
  const member = (await createPAT(api, (await register(api, memberEmail)).access_token)).token;
  const guestEmail = emailFor(testInfo, "guest");
  const guest = (await createPAT(api, (await register(api, guestEmail)).access_token)).token;
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin, { name: "Acme", slug });
  await inviteAndAccept(api, admin, slug, { email: memberEmail, token: member }, 15);
  await inviteAndAccept(api, admin, slug, { email: guestEmail, token: guest }, 5);
  const adminId = await accountId(api, admin);
  const memberId = await accountId(api, member);
  const guestId = await accountId(api, guest);
  // Each new project goes first in the admin's sidebar: Old, Docs, Secret, Web. Old is archived; the guest is
  // Docs' guest.
  const { web, secret, docs } = await amidAnotherWorkspace(api, admin, testInfo, async () => {
    const made = {
      web: await createProject(api, admin, slug, { name: "Web", identifier: "WEB", network: 2 }),
      secret: await createProject(api, admin, slug, { name: "Secret", identifier: "SEC", network: 0 }),
      docs: await createProject(api, admin, slug, { name: "Docs", identifier: "DOCS", network: 2 }),
    };
    const old = await createProject(api, admin, slug, { name: "Old", identifier: "OLD", network: 2 });
    const archived = await api.POST("/api/v0/projects/{project_id}/archive", {
      params: { path: { project_id: old.id } },
      headers: bearer(admin),
    });
    expect(archived.response.status, "archive Old").toBe(200);
    await addProjectMembers(api, admin, made.docs.id, [{ member_id: guestId, role: 5 }]);
    return made;
  });

  // The member has no place in any sidebar of his: by name.
  expect(await listed(api, admin, slug)).toEqual(["Docs", "Secret", "Web"]);
  expect(await listed(api, member, slug)).toEqual(["Docs", "Web"]);
  expect(await listed(api, guest, slug)).toEqual(["Docs"]);
  expect(await listed(api, admin, slug, true)).toEqual(["Old"]);
  expect(await listed(api, member, slug, true)).toEqual(["Old"]);
  expect(await listed(api, guest, slug, true)).toEqual([]);

  // The member joins Web: a member's membership, with his display settings at 65535, both by him.
  const joined = await join(api, member, web.id);
  expect(joined).toMatchObject({ status: 200, project: { id: web.id, member_role: 15, sort_order: 65535 } });
  expect(joined.project?.member_ids?.toSorted()).toEqual([adminId, memberId].toSorted());
  const membership = { role: 15, is_active: true, sort_order: 65535, by: memberEmail };
  await expectMember(db, web.id, memberEmail, membership);
  expect(await listed(api, member, slug)).toEqual(["Web", "Docs"]);
  // Joining again changes nothing.
  expect(await join(api, member, web.id)).toMatchObject({
    status: 200,
    project: { member_role: 15, sort_order: 65535 },
  });
  await expectMember(db, web.id, memberEmail, membership);

  // Refused, nothing written: the member does not see Secret, the guest does not see Web; the guest sees Docs, as
  // its guest, and is refused as one.
  expect(await join(api, member, secret.id)).toEqual({ status: 404, code: "project.not_found" });
  await expectMember(db, secret.id, memberEmail, null);
  expect(await join(api, guest, web.id)).toEqual({ status: 404, code: "project.not_found" });
  await expectMember(db, web.id, guestEmail, null);
  expect(await join(api, guest, docs.id)).toEqual({ status: 403, code: "forbidden" });
  await expectMember(db, docs.id, guestEmail, { role: 5, is_active: true, sort_order: 65535, by: adminEmail });
});
````

`e2e/stories/project/p3-project-settings.spec.ts`（新文件，163 行）：

````file e2e/stories/project/p3-project-settings.spec.ts
import {
  addProjectMembers,
  amidAnotherWorkspace,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
  type ProjectUpdate,
} from "../../fixtures/api";
import { expectMember } from "../../fixtures/assert/project";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import type { Database } from "../../fixtures/db";
import { expect, test } from "../../fixtures/test";

// P3, a project's settings (M3 design 2, 3.4, 3.5, 3.19). The page version
// comes with the project's settings pages (P10).

/** The answer of PATCH /api/v0/projects/{project_id}: its status, the project, or the problem's code and fields. */
async function change(api: Api, token: string, id: string, body: ProjectUpdate) {
  const { data, error, response } = await api.PATCH("/api/v0/projects/{project_id}", {
    params: { path: { project_id: id } },
    body,
    headers: bearer(token),
  });
  return data
    ? { status: response.status, project: data }
    : {
        status: response.status,
        code: error?.code,
        errors: error?.errors?.map((e) => ({ field: e.field, code: e.code })),
      };
}

/** The project's settings as stored, its lead and default assignee by address, and who changed it last. */
async function stored(db: Database, id: string): Promise<unknown> {
  const [row] = await db.query(
    `SELECT p.name, p.identifier, p.description, p.network, p.timezone, p.logo_props, p.cycle_view, p.module_view,
            p.issue_views_view, p.intake_view, p.guest_view_all_features, p.archive_in, l.email AS lead,
            a.email AS default_assignee, u.email AS by
       FROM projects p
       JOIN users u ON u.id = p.updated_by_id
       LEFT JOIN users l ON l.id = p.project_lead_id
       LEFT JOIN users a ON a.id = p.default_assignee_id
      WHERE p.id = $1`,
    [id]
  );
  return row;
}

test("P3 (API): the project's admin adds a member and a guest, whom the member lists with him, then changes every setting, the member its lead and default assignee; its member may not; a lead or default assignee who is its guest or no member, and archive_in 13, change nothing", async ({
  api,
  db,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
  const memberEmail = emailFor(testInfo, "member");
  const member = (await createPAT(api, (await register(api, memberEmail)).access_token)).token;
  const guestEmail = emailFor(testInfo, "guest");
  const guest = (await createPAT(api, (await register(api, guestEmail)).access_token)).token;
  const otherEmail = emailFor(testInfo, "other");
  const other = (await createPAT(api, (await register(api, otherEmail)).access_token)).token;
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin, { name: "Acme", slug });
  await inviteAndAccept(api, admin, slug, { email: memberEmail, token: member }, 15);
  await inviteAndAccept(api, admin, slug, { email: guestEmail, token: guest }, 5);
  await inviteAndAccept(api, admin, slug, { email: otherEmail, token: other }, 15);
  const memberId = await accountId(api, member);
  const guestId = await accountId(api, guest);
  const otherId = await accountId(api, other);
  // other is a member of Acme and of Ops, not of Web: a membership of another project makes no one Web's assignee.
  // The member is Ops' member too, at 65535 in his sidebar: Web, added later, goes before it.
  const web = await amidAnotherWorkspace(api, admin, testInfo, async () => {
    const ops = await createProject(api, admin, slug, { name: "Ops", identifier: "OPS" });
    await addProjectMembers(api, admin, ops.id, [
      { member_id: otherId, role: 15 },
      { member_id: memberId, role: 15 },
    ]);
    return createProject(api, admin, slug, { name: "Web", identifier: "WEB" });
  });

  const added = await addProjectMembers(api, admin, web.id, [
    { member_id: memberId, role: 15 },
    { member_id: guestId, role: 5 },
  ]);
  expect(added.map((m) => ({ project_id: m.project_id, member_id: m.member_id, role: m.role }))).toEqual([
    { project_id: web.id, member_id: memberId, role: 15 },
    { project_id: web.id, member_id: guestId, role: 5 },
  ]);
  await expectMember(db, web.id, memberEmail, { role: 15, is_active: true, sort_order: 55535, by: adminEmail });
  await expectMember(db, web.id, guestEmail, { role: 5, is_active: true, sort_order: 65535, by: adminEmail });
  // Web lists exactly its own three; other, Ops' member, is not among them.
  const members = await api.GET("/api/v0/projects/{project_id}/members", {
    params: { path: { project_id: web.id } },
    headers: bearer(member),
  });
  expect(members.response.status, `Web's members: ${JSON.stringify(members.error)}`).toBe(200);
  expect(
    members.data?.data.map((m) => ({ member_id: m.member_id, role: m.role })).toSorted((a, b) => b.role - a.role)
  ).toEqual([
    { member_id: await accountId(api, admin), role: 20 },
    { member_id: memberId, role: 15 },
    { member_id: guestId, role: 5 },
  ]);

  const logo = { in_use: "icon", icon: { name: "rocket", color: "#46A758" } } as const;
  const settings = {
    name: "Site",
    identifier: "SITE",
    description: "The site",
    network: 0 as const,
    timezone: "Asia/Shanghai",
    logo_props: logo,
    cycle_view: false,
    module_view: false,
    issue_views_view: false,
    intake_view: true,
    guest_view_all_features: true,
    archive_in: 3,
  };
  expect(
    await change(api, admin, web.id, {
      ...settings,
      identifier: "site",
      project_lead_id: memberId,
      default_assignee_id: memberId,
    })
  ).toMatchObject({ status: 200, project: { ...settings, project_lead_id: memberId, default_assignee_id: memberId } });
  const changed = { ...settings, lead: memberEmail, default_assignee: memberEmail, by: adminEmail };
  expect(await stored(db, web.id)).toEqual(changed);

  // Each refused, all at once: none writes.
  const refusals = [
    { token: member, body: { name: "Mine" }, want: { status: 403, code: "forbidden" } },
    {
      token: admin,
      body: { archive_in: 13 },
      want: { status: 422, code: "validation_failed", errors: [{ field: "archive_in", code: "out_of_range" }] },
    },
    {
      token: admin,
      body: { project_lead_id: guestId },
      want: { status: 422, code: "validation_failed", errors: [{ field: "project_lead_id", code: "not_allowed" }] },
    },
    {
      token: admin,
      body: { default_assignee_id: otherId },
      want: { status: 422, code: "validation_failed", errors: [{ field: "default_assignee_id", code: "not_allowed" }] },
    },
  ];
  expect(
    await Promise.all(refusals.map(({ token, body }) => change(api, token, web.id, body))),
    "the refusals"
  ).toEqual(refusals.map((r) => r.want));
  expect(await stored(db, web.id)).toEqual(changed);

  // Both cleared with null.
  expect(await change(api, admin, web.id, { project_lead_id: null, default_assignee_id: null })).toMatchObject({
    status: 200,
    project: { project_lead_id: null, default_assignee_id: null },
  });
  expect(await stored(db, web.id)).toEqual({ ...changed, lead: null, default_assignee: null });
});
````

`e2e/stories/project/p4-archive.spec.ts`（新文件，143 行）：

````file e2e/stories/project/p4-archive.spec.ts
import {
  addProjectMembers,
  amidAnotherWorkspace,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
} from "../../fixtures/api";
import { expectProjectCreated, expectProjectDeleted } from "../../fixtures/assert/project";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import type { Database } from "../../fixtures/db";
import { expect, test } from "../../fixtures/test";

// P4, archiving, unarchiving and deleting a project (M3 design 2, 3.6,
// 3.19). The page version comes with the project's settings pages (P10).

/** The names of the projects of slug that the caller of token lists; archived: the archived ones. */
async function listed(api: Api, token: string, slug: string, archived = false): Promise<string[]> {
  const { data, error, response } = await api.GET("/api/v0/workspaces/{slug}/projects", {
    params: { path: { slug }, query: { archived } },
    headers: bearer(token),
  });
  expect(response.status, `list ${slug}'s projects: ${JSON.stringify(error)}`).toBe(200);
  return (data?.data ?? []).map((p) => p.name);
}

/** POST /api/v0/projects/{project_id}/{action}, which must be 200: the project's archived_at. */
async function act(api: Api, token: string, id: string, action: "archive" | "unarchive"): Promise<string | null> {
  const init = { params: { path: { project_id: id } }, headers: bearer(token) };
  const { data, error, response } =
    action === "archive"
      ? await api.POST("/api/v0/projects/{project_id}/archive", init)
      : await api.POST("/api/v0/projects/{project_id}/unarchive", init);
  expect(response.status, `${action} ${id}: ${JSON.stringify(error)}`).toBe(200);
  return data?.archived_at ?? null;
}

/**
 * Every operation on the project of id by the caller of token, each once: reading, changing, archiving, unarchiving
 * and deleting it, listing and adding its members (the account of memberId), joining it, reading and changing his
 * display settings in it. Each answer's status and code, in that order.
 */
async function onProject(api: Api, token: string, id: string, memberId: string): Promise<unknown[]> {
  const init = { params: { path: { project_id: id } }, headers: bearer(token) };
  const answers = await Promise.all([
    api.GET("/api/v0/projects/{project_id}", init),
    api.PATCH("/api/v0/projects/{project_id}", { ...init, body: { name: "Site" } }),
    api.POST("/api/v0/projects/{project_id}/archive", init),
    api.POST("/api/v0/projects/{project_id}/unarchive", init),
    api.DELETE("/api/v0/projects/{project_id}", init),
    api.GET("/api/v0/projects/{project_id}/members", init),
    api.POST("/api/v0/projects/{project_id}/members", {
      ...init,
      body: { members: [{ member_id: memberId, role: 15 }] },
    }),
    api.POST("/api/v0/projects/{project_id}/join", init),
    api.GET("/api/v0/me/projects/{project_id}/preferences", init),
    api.PATCH("/api/v0/me/projects/{project_id}/preferences", { ...init, body: { sort_order: 1 } }),
  ]);
  return answers.map(({ error, response }) => ({ status: response.status, code: error?.code }));
}

/** Whether the project is archived as stored, and who changed it last. */
async function archiving(db: Database, id: string): Promise<unknown> {
  const [row] = await db.query(
    `SELECT p.archived_at IS NOT NULL AS archived, p.archived_at IS NOT DISTINCT FROM p.updated_at OR p.archived_at IS NULL AS at_its_change,
            u.email AS by
       FROM projects p JOIN users u ON u.id = p.updated_by_id WHERE p.id = $1`,
    [id]
  );
  return row;
}

test("P4 (API): the admin archives a project, which leaves the list for the archived ones and cannot be changed; unarchives it and archives it again; then deletes it with its members, settings and states at one moment, after which it is not found and its identifier is free", async ({
  api,
  db,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
  const memberEmail = emailFor(testInfo, "member");
  const member = (await createPAT(api, (await register(api, memberEmail)).access_token)).token;
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin, { name: "Acme", slug, timezone: "UTC" });
  await inviteAndAccept(api, admin, slug, { email: memberEmail, token: member }, 15);
  const memberId = await accountId(api, member);
  // Web goes before Ops in the admin's sidebar: Ops 65535, Web 55535. The member is Web's member, so each table under
  // it has his row too.
  const { ops, web } = await amidAnotherWorkspace(api, admin, testInfo, async () => {
    const made = {
      ops: await createProject(api, admin, slug, { name: "Ops", identifier: "OPS" }),
      web: await createProject(api, admin, slug, { name: "Web", identifier: "WEB" }),
    };
    await addProjectMembers(api, admin, made.web.id, [{ member_id: memberId, role: 15 }]);
    return made;
  });

  expect(await act(api, admin, web.id, "archive")).toEqual(expect.any(String));
  expect(await archiving(db, web.id)).toEqual({ archived: true, at_its_change: true, by: adminEmail });
  expect(await listed(api, admin, slug)).toEqual(["Ops"]);
  expect(await listed(api, admin, slug, true)).toEqual(["Web"]);
  const refused = await api.PATCH("/api/v0/projects/{project_id}", {
    params: { path: { project_id: web.id } },
    body: { name: "Site" },
    headers: bearer(admin),
  });
  expect({ status: refused.response.status, code: refused.error?.code }).toEqual({
    status: 409,
    code: "project.archived",
  });

  expect(await act(api, admin, web.id, "unarchive")).toBeNull();
  expect(await archiving(db, web.id)).toEqual({ archived: false, at_its_change: true, by: adminEmail });
  expect(await listed(api, admin, slug)).toEqual(["Web", "Ops"]);
  expect(await listed(api, admin, slug, true)).toEqual([]);
  expect(await act(api, admin, web.id, "archive")).toEqual(expect.any(String));

  // An archived project is deleted as any other.
  const deleted = await api.DELETE("/api/v0/projects/{project_id}", {
    params: { path: { project_id: web.id } },
    headers: bearer(admin),
  });
  expect(deleted.response.status, `delete Web: ${JSON.stringify(deleted.error)}`).toBe(204);
  await expectProjectDeleted(db, web.id, adminEmail);
  // Every operation on it is refused as on a project that does not exist, its admin's too.
  expect(await onProject(api, admin, web.id, memberId)).toEqual(
    Array(10).fill({ status: 404, code: "project.not_found" })
  );
  expect(await listed(api, admin, slug)).toEqual(["Ops"]);
  expect(await listed(api, admin, slug, true)).toEqual([]);

  // WEB is free again. The new Web goes before Ops, at 55535: the deleted Web's place, 55535, no longer counts.
  const again = await createProject(api, admin, slug, { name: "Web", identifier: "WEB" });
  const row = { name: "Web", identifier: "WEB", description: "", network: 2, timezone: "UTC", logo_props: {} };
  expect(await expectProjectCreated(db, slug, row, adminEmail, null, [{ email: adminEmail, sort_order: 55535 }])).toBe(
    again.id
  );
  expect(
    await expectProjectCreated(db, slug, { ...row, name: "Ops", identifier: "OPS" }, adminEmail, null, [
      { email: adminEmail, sort_order: 65535 },
    ])
  ).toBe(ops.id);
});
````

`e2e/stories/project/p8-project-preferences.spec.ts`（新文件，126 行）：

````file e2e/stories/project/p8-project-preferences.spec.ts
import {
  addProjectMembers,
  amidAnotherWorkspace,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
  type ProjectPreferences,
  type ProjectPreferencesUpdate,
} from "../../fixtures/api";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import type { Database } from "../../fixtures/db";
import { expect, test } from "../../fixtures/test";

// P8, a member's display settings in a project (M3 design 2, 3.18). The
// page version comes with the project's header and the sidebar (P10).

/** GET /api/v0/me/projects/{project_id}/preferences, which must be 200. */
async function read(api: Api, token: string, id: string): Promise<unknown> {
  const { data, error, response } = await api.GET("/api/v0/me/projects/{project_id}/preferences", {
    params: { path: { project_id: id } },
    headers: bearer(token),
  });
  expect(response.status, `read the settings in ${id}: ${JSON.stringify(error)}`).toBe(200);
  return data;
}

/** PATCH /api/v0/me/projects/{project_id}/preferences: its status, and the settings or the problem's fields. */
async function change(api: Api, token: string, id: string, body: ProjectPreferencesUpdate) {
  const { data, error, response } = await api.PATCH("/api/v0/me/projects/{project_id}/preferences", {
    params: { path: { project_id: id } },
    body,
    headers: bearer(token),
  });
  return data
    ? { status: response.status, preferences: data }
    : {
        status: response.status,
        code: error?.code,
        errors: error?.errors?.map((e) => ({ field: e.field, code: e.code })),
      };
}

/** The account of email's display settings in the project as stored, and who wrote them last. */
async function stored(db: Database, id: string, email: string): Promise<unknown[]> {
  return db.query(
    `SELECT s.preferences, s.sort_order, b.email AS by
       FROM project_user_properties s JOIN users u ON u.id = s.user_id JOIN users b ON b.id = s.updated_by_id
      WHERE s.project_id = $1 AND u.email = $2 AND s.deleted_at IS NULL`,
    [id, email]
  );
}

test("P8 (API): the admin opens a project on its modules tab, moves views under more and drags it first in his sidebar, which lasts; an unknown or the work items tab changes nothing; his member's settings stay his own", async ({
  api,
  db,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
  const memberEmail = emailFor(testInfo, "member");
  const member = (await createPAT(api, (await register(api, memberEmail)).access_token)).token;
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin, { name: "Acme", slug });
  await inviteAndAccept(api, admin, slug, { email: memberEmail, token: member }, 15);
  // The admin's sidebar: Wiki 35535, Docs 45535, Web 55535, Ops 65535. Web is neither the first nor the last made,
  // so that each settings row the admin reads or writes must be his own in Web, whichever row the database reads first.
  const web = await amidAnotherWorkspace(api, admin, testInfo, async () => {
    await createProject(api, admin, slug, { name: "Ops", identifier: "OPS" });
    const made = await createProject(api, admin, slug, { name: "Web", identifier: "WEB" });
    await createProject(api, admin, slug, { name: "Docs", identifier: "DOCS" });
    await createProject(api, admin, slug, { name: "Wiki", identifier: "WIKI" });
    await addProjectMembers(api, admin, made.id, [{ member_id: await accountId(api, member), role: 15 }]);
    return made;
  });
  const memberSettings: ProjectPreferences = {
    navigation: { default_tab: "work_items", hide_in_more_menu: [] },
    sort_order: 65535,
  };
  expect(await read(api, admin, web.id)).toEqual({ ...memberSettings, sort_order: 55535 });

  const tabs: ProjectPreferences = {
    navigation: { default_tab: "modules", hide_in_more_menu: ["views"] },
    sort_order: 55535,
  };
  expect(await change(api, admin, web.id, { navigation: tabs.navigation })).toEqual({ status: 200, preferences: tabs });
  // Dragged before Wiki (35535).
  const first = { ...tabs, sort_order: 25535 };
  expect(await change(api, admin, web.id, { sort_order: 25535 })).toEqual({ status: 200, preferences: first });
  // After a refresh the page reads them again, and the sidebar lists Web first.
  expect(await read(api, admin, web.id)).toEqual(first);
  const { data } = await api.GET("/api/v0/workspaces/{slug}/projects", {
    params: { path: { slug } },
    headers: bearer(admin),
  });
  expect(data?.data.map((p) => p.name)).toEqual(["Web", "Wiki", "Docs", "Ops"]);
  const row = [{ preferences: { navigation: first.navigation }, sort_order: 25535, by: adminEmail }];
  expect(await stored(db, web.id, adminEmail)).toEqual(row);

  // Each refused, all at once: none writes.
  const refusals: { body: ProjectPreferencesUpdate; field: string }[] = [
    {
      // A tab the contract does not name: the client's types would not send it.
      body: { navigation: { default_tab: "pages", hide_in_more_menu: [] } } as unknown as ProjectPreferencesUpdate,
      field: "navigation.default_tab",
    },
    {
      body: { navigation: { default_tab: "cycles", hide_in_more_menu: ["work_items"] } },
      field: "navigation.hide_in_more_menu[0]",
    },
  ];
  expect(await Promise.all(refusals.map(({ body }) => change(api, admin, web.id, body))), "the refusals").toEqual(
    refusals.map(({ field }) => ({
      status: 422,
      code: "validation_failed",
      errors: [{ field, code: "invalid_format" }],
    }))
  );
  expect(await stored(db, web.id, adminEmail)).toEqual(row);

  // The member's settings in Web are his own, as the admin's addition made them.
  expect(await read(api, member, web.id)).toEqual(memberSettings);
  expect(await stored(db, web.id, memberEmail)).toEqual([
    { preferences: { navigation: memberSettings.navigation }, sort_order: 65535, by: adminEmail },
  ]);
});
````

`e2e/stories/workspace/w3-workspace-settings.spec.ts`（修改，1 处）：

````old e2e/stories/workspace/w3-workspace-settings.spec.ts
  expect(memberDeletes.error?.code).toBe("forbidden");

````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
  expect(memberDeletes.error?.code).toBe("forbidden");

  // A project deleted before the workspace keeps its moment, and so do its rows: Old, with the member its lead, so
  // that each project table has a row of it.
  const old = await createProject(api, admin, slug, { name: "Old", identifier: "OLD", project_lead_id: memberId });
  const oldDeleted = await api.DELETE("/api/v0/projects/{project_id}", {
    params: { path: { project_id: old.id } },
    headers: bearer(admin),
  });
  expect(oldDeleted.response.status).toBe(204);

````

- [ ] **Step 3: 前端检查、端到端**

Run: `make lint-web`
Expected: 通过。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 62 个全部通过（此前的 58 个，P2、P3、P4、P8；W3 先删除 Old）。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add e2e/fixtures/api.ts e2e/fixtures/assert/project.ts e2e/fixtures/assert/workspace.ts e2e/stories/project/p2-visibility.spec.ts e2e/stories/project/p3-project-settings.spec.ts e2e/stories/project/p4-archive.spec.ts e2e/stories/project/p8-project-preferences.spec.ts e2e/stories/workspace/w3-workspace-settings.spec.ts
```
```bash
git commit -m "test(M3/P4b): the API versions of P2, P3, P4 and P8; W3 deletes a project first

P2 lists and joins projects, P3 adds members and changes a project's
settings, P4 archives, unarchives and deletes one, P8 changes a member's
display settings; each reads the rows it wrote amid another workspace's
project of the same name. expectProjectCreated reads the undeleted
project, a deleted one's identifier being free; W3 deletes Old before the
workspace, so the project tables have rows deleted alone.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**变异**（spec 附录 A；每个都只运行它的故事，`e2e_sweep1.py`、`e2e_mutants.py`）：

| 改坏 | 必须失败的故事 |
|---|---|
| `LockProject` 去掉项目的 id | 升序 P2、P3、P4、P8、W3；降序 P2、P3、P4 |
| `ShareProject` 去掉项目的 id（两个行序） | P8 |
| `ProjectWorkspace` 去掉项目的 id（两个行序） | P2、P3、P4、P8 |
| `UpdateProject`、`SetArchived` 去掉项目的 id | P3；P4 |
| `Memberships`、`ListMembers` 读别的项目的 | P3 |
| `Preferences` 读别的项目的、别的账户的（两个行序）；`UpsertPreferences` 的冲突目标没有条件 | P8 |
| `EnsurePreferences` 的冲突目标没有条件 | P2、P3、P4、P8 |
| 删除的四条语句各去掉项目 | P4、W3 |
| 删除的四条语句各去掉工作区、各去掉 `deleted_at IS NULL` | W3 |
| `ProjectWorkspace` 去掉 `deleted_at IS NULL`，同时 `ProjectFacts` 去掉 `p.deleted_at`；`LockProject`、`ShareProject` 各去掉 `deleted_at IS NULL`，同时 `ProjectWorkspace`、`ProjectFacts` 各去掉自己的 | P4（单独去掉、或锁的只与 `ProjectFacts` 的一起去掉时由别的遮住：方案 E 之下写先经 `ProjectWorkspace`，spec 第 3 节第 6 条） |
| 状态在另一个时刻删除；`LockProject` 不读已归档的项目；已归档的项目照改 | P4 |
| 工作区访客加入公开项目；工作区角色在"已是有效成员"之后查；加入的位置不在 65535；规则给工作区的每个有效成员 | P2 |
| 添加的位置在默认值；添加时不取请求的角色；负责人可以是访客；`archive_in` 收 13；标识不转大写 | P3 |
| 读的是谁都不是的设置；未知的标签页、藏起 work_items 被接受 | P8 |

故事看不到的谓词（`Memberships` 的账户、已删除，`ListMembers` 的已结束、已删除，`RestoreMember` 的 id，`Preferences` 的已删除，`ProjectFacts` 的 `m.deleted_at`、`p.deleted_at` 单独去掉）由存储测试发现；只有 P5 的移出或删除项目写得出这些行（spec 第 3 节第 6 条）。工作区按 id 的锁的 id（两个行序）、`deleted_at IS NULL` 故事也看不到：`Locks` 只看找到与否，故事里没有已删除的工作区上的写（spec 第 3 节第 15 条），由 Task 2 的存储测试发现。

**Done when:** 62 个故事全部通过；P4 删除项目之后每个操作都答 404；W3 中项目的四张表有单独删除的行。

---

### Task 16: 文档：差异清单、交接的处理结果

**Files:**
- Modify: `docs/v0/M3-workspace-project/handoffs/M1-P3-trim-platform.md`、`docs/v0/plane-diff.md`

**Interfaces:** 没有代码。M3 设计 3.20 中 P4b 的一行：差异清单第四节中项目负责人、默认负责人和项目标识两行补上修改的规则，加上修改和删除、归档和恢复、添加已是成员的人、加入时恢复的四行；M1-P3 交接的"处理结果（M3/P4b）"：项目成员只经 `addProjectMembers` 和 `joinProject`，没有按邮件加人的接口；前端改调新接口和守卫例外的删除留给 P8。

**Tests:** 没有新测试；`make lint-web` 的关键词守卫查文档。

- [ ] **Step 1: 差异清单和交接**

`docs/v0/plane-diff.md`（修改，2 处）：

````old docs/v0/plane-diff.md
| 项目负责人、默认负责人 | 外键 `CASCADE`；负责人可以是任何账户 | `ON DELETE SET NULL`；创建项目时，负责人须是工作区的有效管理员或成员，否则 422（`project_lead_id`，`not_allowed`），他与创建者都成为项目管理员（M3 设计 3.15、3.19）。修改项目时的规则由 M3/P4b 加入 |
````
````new docs/v0/plane-diff.md
| 项目负责人、默认负责人 | 外键 `CASCADE`；负责人可以是任何账户 | `ON DELETE SET NULL`；创建项目时，负责人须是工作区的有效管理员或成员，否则 422（`project_lead_id`，`not_allowed`），他与创建者都成为项目管理员（M3 设计 3.15、3.19）。修改项目时，负责人、默认负责人都须是项目中不是访客的有效成员，否则 422（`project_lead_id`、`default_assignee_id`，`not_allowed`），`null` 清空；修改不把谁加为成员（M3 设计 3.19） |
````

````old docs/v0/plane-diff.md
| 项目标识 | 最多 12 个字符，只禁一组符号 | 转成大写后 1–10 个，只能是 `A-Z`、`0-9` 和 `ÇŞĞİÖÜ`（M3 设计 3.19）；修改项目时同一规则由 M3/P4b 加入 |
````
````new docs/v0/plane-diff.md
| 项目标识 | 最多 12 个字符，只禁一组符号 | 转成大写后 1–10 个，只能是 `A-Z`、`0-9` 和 `ÇŞĞİÖÜ`（M3 设计 3.19）；修改项目时同一规则 |
| 修改、删除项目 | 不是项目成员的工作区管理员也能 | 项目级规则：项目管理员，或同时是工作区管理员的项目成员；不是成员的工作区管理员先加入（M3 设计 3.4） |
| 归档、恢复项目 | 项目管理员和成员 | 项目管理员，或同时是工作区管理员的项目成员（M3 设计 3.4） |
| 添加已是有效成员的人为项目成员 | 顺手改他的角色 | 422 `duplicate`（`members[i].member_id`），整批都不写（M3 设计 3.5） |
| 加入项目时恢复以前的成员行 | 只改 `is_active`，保留旧的角色：被移出的项目管理员自己加入就拿回管理员 | 角色取原来那一行的角色与他现在的工作区角色中较低的一个：不比新加入给得更多（被移出的项目管理员现在是工作区成员，回来是成员），也不比原来那一行更多（与 Plane 相同，被降为访客的人离开再加入仍是访客）（M3 设计 3.5） |
````

`docs/v0/M3-workspace-project/handoffs/M1-P3-trim-platform.md`（修改，1 处）：

````old docs/v0/M3-workspace-project/handoffs/M1-P3-trim-platform.md
来源：[M3/P4a spec](../specs/P4a-projects.md) 第 7 节。

````
````new docs/v0/M3-workspace-project/handoffs/M1-P3-trim-platform.md
来源：[M3/P4a spec](../specs/P4a-projects.md) 第 7 节。

## 处理结果（M3/P4b）

- **项目成员**（接口一侧完成）：只有一种方式：`addProjectMembers`（`POST /api/v0/projects/{project_id}/members`）从工作区的有效成员中添加，没有按邮件把人加进项目的接口；工作区的成员、管理员自己加入看得到的项目是 `joinProject`（`POST /api/v0/projects/{project_id}/join`，`api/modules/project.yaml`）。新接口的路径和说明都不含守卫 `project-invitations` 的字样，`schema.gen.ts` 里没有新的命中。

仍未处理，状态保持 `open`：`joinProject` 的页面改调新接口、守卫的 `project-invitations` 例外删除（P8）；`RESTRICTED_URLS` 与后端同源（前端一侧，P8）。

来源：[M3/P4b spec](../specs/P4b-project-members.md) 第 7 节。

````

- [ ] **Step 2: 检查**

Run: `make lint-web`
Expected: 通过。

- [ ] **Step 3: 提交**

```bash
git add docs/v0/M3-workspace-project/handoffs/M1-P3-trim-platform.md docs/v0/plane-diff.md
```
```bash
git commit -m "docs(M3/P4b): plane-diff's P4b rows; the M1-P3 handoff's project members

plane-diff gains the update's lead, default assignee and identifier
rules, who may change, delete, archive and unarchive a project, the
duplicate refused on add, and the role a restored membership takes on
join. The M1-P3 handoff records that project members are added from the
workspace's members or join, with no invitation by e-mail.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**变异**：文档没有可变异的代码；每一句由 spec 附录 A 的清扫 6 逐句对照代码或测试核对。

**Done when:** 差异清单和交接写好；`make lint-web` 通过。
