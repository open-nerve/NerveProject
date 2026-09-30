# M3/P4a 项目的建立、可见性与两个连带 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `project` 模块和项目的四张表出现：工作区的管理员和成员建项目（创建者和负责人成为项目管理员、各自侧边栏的第一位，六个默认状态），调用者按可见性列出、查看项目，检查标识是否可用；删除工作区在同一个事务里连带项目和项目之下的行，工作区成员改为访客、以访客的身份接受邀请恢复成员关系时，同一个事务里他在这个工作区的项目角色都改为访客。矩阵每行带自己的列（项目级 12 列），加 4 个操作的行（49 格）；每个模块的 HTTP 测试经 `apitest.Main` 的核对；P1 的接口版本和 W3 的项目连带通过；差异清单、交接中 P4a 的各行写好。

**Architecture:** 新迁移 `00010`–`00013`（`projects`、`project_members`、`project_user_properties`、`states`）和 `sqlc.yaml` 的 `project` 条目。新模块 `project`：domain（`CheckNewProject`、`CanLead`、`LogoProps`、`DefaultStates`、`SortOrderFirst`、`VisibilityOf`、操作名）；app（四个用例、`Cascade` 的 `DeleteWorkspaceProjects` 和 `DemoteToGuest`，按用例分的存储端口，`WorkspaceDirectory`、`WorkspaceMembers` 两个跨模块端口）；存储（sqlc）；HTTP（`api/modules/project.yaml`）。`project.Provide` 只交出 `ProjectAccess`（裁定 S2）；`project.New` 在 `workspace.New` 之前，`workspace` 经 `ProjectCascade` 端口拿到它的 `Cascade()`。`workspace`：`Provide` 加 `WorkspaceDirectory`、`WorkspaceMembers`（`adapter/postgres/directory.go`）；`cascade()` 的最后一步、`updateWorkspaceMember` 和接受邀请调 `ProjectCascade`。`access`：`ProjectAccess` 端口，`Authorize` 在目标带项目时读项目的事实；规则表加 4 行。`bootstrap`：两个转换（`projectWorkspaces`、`accessProjects`）、组合的顺序、注册。测试一侧：`workspace/app/fakes_test.go` 按用例拆开；矩阵每行带自己的列、`notTargets` 按路径参数列出；`apitest.Main` 的核对。

**Tech Stack:** Go 1.27.1、pgx v5.11.0、sqlc v1.31.1（`CGO_ENABLED=0`）、oapi-codegen v2.8.0、goose v3.28.0、River v0.47.0、golangci-lint 2.13.2、PostgreSQL 18.6（testcontainers）；Node 24、pnpm 11.10.0、Playwright 1.63.0。

**Spec:** `docs/v0/M3-workspace-project/specs/P4a-projects.md`（上级：`docs/v0/M3-workspace-project/M3-design.md`）

## Global Constraints

- **Go 版本和依赖**：本 plan 不执行 `go get`，不加任何 Go 模块或 npm 包。`server/go.mod` 和 `server/tools/go.mod` 保持 `go 1.27` 和 `toolchain go1.27.1`（golangci-lint 2.13.2 由 go1.27.0 构建，`go` 行更高就拒绝运行）。每个 Task 提交前执行 `grep -n "^go \|^toolchain" server/go.mod server/tools/go.mod`，四行必须是 `go 1.27` 和 `toolchain go1.27.1`；`git diff --stat d0796853 -- server/go.mod server/go.sum server/tools pnpm-lock.yaml` 在本 plan 的任何时刻都没有输出。
- **每个 Task 提交前**：`make lint-go` 两段都输出 `0 issues.`，`make test` 全部通过（没有 `FAIL`）。
- **生成的文件不手写、不从本 plan 复制**：有生成物的 Task（2、4、5、8–12）执行 Task 中的生成命令（改了接口描述的 Task 2、8–11 执行 `make gen`，只动了 sqlc 的 Task 4、5、12 执行 `make gen-go`），用 `shasum -a 256` 和 `wc -l` 核对表中的 SHA-256 和行数，对不上时停下来：说明某个输入与本 plan 不一致。这些 Task **提交之后**执行 `make gen-check`（它用 `git status` 判断生成物是否已提交，提交之前必然报差异）。
- **前端检查**：改了接口描述、前端或 `e2e/` 的 Task（2、8–11、14）另执行 `make lint-web`、`make knip`、`make test-web`；声明新错误码的 Task 8（`project.identifier_taken`、`project.name_taken`）、Task 9（`project.not_found`）在同一个 Task 里把码加进 `PROBLEM_MESSAGES` 和两份 `auth.json`（M3 设计 12 节约束 4）；Task 15 只改文档，执行 `make lint-web`（关键词守卫也查文档）；Task 14 执行 `make e2e`。
- **容器**：`make test` 和 `make e2e` 用自己的 testcontainers；机器忙时偶尔起不来，等 Docker 空闲之后重跑一次再当作失败。开发库 `nerve-dev-db-1` 可以用，但不要停止或重建它，不要执行 `make dev-db-down`、`make dev-db-reset`。不要碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`）。
- **git**：每次 Bash 调用只执行一个 git 命令，不用 `;`、`&&`、`|` 串联 git；不用 `git -C`、`stash`、`clean`、`reset --hard`。`cd` 不与别的命令组合，只读的命令也不行。不碰 `plane/`、`refer/`。
- **安装**：除了 Docker、Go、Node 不做任何全局安装；不执行 `corepack enable`（pnpm 已在 PATH 上）。
- **规则**：不写 `init()`，不用全局可变状态，构造函数显式传入依赖；不建 `utils`、`common`、`helpers` 包；一个文件只做一件事，不超过约 400 行；依赖只能向内；模块之间不互相导入，值在 `bootstrap` 中转换或直接接上（M3 设计 6.5、6.6），不跨模块的表 JOIN；模块的 SQL 只经 sqlc（`TestModulesRunSQLOnlyThroughSQLC`）；不留没有使用者的代码（`project.NewCascade` 不建，裁定 G2；`ProjectMembershipCounts` 不建，裁定 S2）。**例外**：接口描述按模块一个文件（M0-P3 交接 5），`api/modules/*.yaml` 不受约 400 行的限制。本 plan 的其余文件都在约 400 行以内：最长的是 `server/migrations/schema_test.go`（399 行，项目的名字和种类因此放进 `project_schema_test.go`）、`bootstrap/invitations_test.go`（386 行）和 `bootstrap/interleaving_answers_test.go`（375 行）；矩阵的格子指向的目标从 `permission_matrix_coverage_test.go` 移到 `permission_matrix_targets_test.go`（Task 6）；P3 留下的 `workspace/app/fakes_test.go`（401 行）在 Task 1 按用例拆成五个文件。
- **注释**：Go、TS 代码、SQL 查询和接口描述用英文；迁移文件的中文注释和中文文档照本 plan 原样。
- **代码块**：每个改动都写成四个反引号围起来的块，块的第一行写明种类和路径，照原样使用（原型中逐字节运行过）：
  - ````` ````file <路径> ````` 新文件，块的内容加一个结尾换行就是整个文件；
  - ````` ````whole <路径> ````` 已有文件的完整新内容（同样加结尾换行）；
  - ````` ````old <路径> ````` 与紧跟着的 ````` ````new <路径> `````：`old` 的文字在文件当前版本中恰好出现一次，把它换成 `new` 的文字；
  - ````` ````delete <路径> ````` 删除这个文件（块是空的）。

  一个文件的几个块按出现的顺序依次应用。拼 plan 的脚本已从 `d0796853` 起按顺序核对过全部块：每个 `old` 恰好出现一次（在它之前的块应用之后的文件中），每个新文件原来不存在，逐 Task 应用之后的文件与原型逐字节相同（spec 附录 A）。可以用 `node <planapply.mjs> <本 plan> apply <仓库根> <n>` 写入第 n 个 Task 的块，也可以手工照抄。
- **过渡版本**：一些文件先在较早的 Task 写成过渡版本，较晚的 Task 再修改（`api/modules/project.yaml`、`api/openapi.yaml`、`project/module.go`、`app/ports.go`、`domain/actions.go`、`domain/errors.go`、`domain/project.go`、`adapter/http/*.go`、`adapter/postgres/projects.go`、`queries/projects.sql`、`queries/cascade.sql`、`access/domain/rules.go`、`bootstrap/app.go`、`bootstrap/ports.go`、矩阵的文件、`workspace/module.go`、`workspace/app/ports.go`、`workspace/app/clock_test.go`、生成物）；每个过渡版本都在逐 Task 复现中运行过。
- **变异**：每个 Task 末尾的"变异"表列出：把代码改坏的方式，和必须因此失败的测试。它们在原型上逐个跑过（`$M3TMP/p4tools/mutants_*.py`、`e2e_mutants.py`、`e2e_sweep1.py`，spec 附录 A）；实现者可以照表抽查，改坏之后必须恢复。表中"（Task n 起）"标出的测试在较晚的 Task 才有。
- **评审敏感**（M3 设计 12 节约束 3）：两个连带第一次跨越模块边界，"谁看得到项目"第一次落地。改动连带、锁、可见性的测试之前，先照"变异"表确认它在所说的性质去掉之后失败。
- **提交**：提交信息用英文，末尾加一行：`Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明。

## 文件结构

路径相对于仓库根目录。"生成"表示由命令生成并提交；一个文件由多个 Task 修改时，"Task"一列都列出。

| 文件 | 职责 | Task |
|---|---|---|
| `server/internal/modules/workspace/app/fakes_test.go`（完整内容）；`server/internal/modules/workspace/app/fakes_accounts_test.go`、`server/internal/modules/workspace/app/fakes_members_test.go`、`server/internal/modules/workspace/app/fakes_preferences_test.go`、`server/internal/modules/workspace/app/fakes_workspaces_test.go` | P2、P3 的共用假实现按用例拆开（只移动） | 1 |
| `server/internal/modules/workspace/app/fakes_projects_test.go` | `ProjectCascade` 的假实现 | 2、12 |
| `server/migrations/sql/00010_project_projects.sql`、`server/migrations/sql/00011_project_project_members.sql`、`server/migrations/sql/00012_project_project_user_properties.sql`、`server/migrations/sql/00013_project_states.sql` | 项目的四张表（M3 设计 4.6–4.9） | 2 |
| `deploy/runtime-grants.sql`、`server/sqlc.yaml`（修改） | 运行时角色的权限；sqlc 的 `project` 条目 | 2 |
| `server/migrations/schema_test.go`（修改）；`server/migrations/project_schema_test.go` | 13 个迁移；四张表的名字和种类；CHECK 的反例；部分唯一键 | 2、3 |
| `server/internal/modules/project/adapter/postgres/queries/cascade.sql`、`server/internal/modules/project/adapter/postgres/cascade.go`、`server/internal/modules/project/adapter/postgres/cascade_test.go` | 删除工作区的四条语句；降为访客的锁和写 | 2、12 |
| `server/internal/modules/project/adapter/postgres/demote_test.go` | 降为访客的存储测试 | 12 |
| `server/internal/modules/project/adapter/postgres/store.go`、`server/internal/modules/project/adapter/postgres/store_test.go` | 存储：事务里的查询；测试的共用准备 | 2、4 |
| `server/internal/modules/project/adapter/postgres/gen/db.go`、`server/internal/modules/project/adapter/postgres/gen/models.go`、`server/internal/modules/project/adapter/postgres/gen/cascade.sql.go`（生成） | | 2、12 |
| `server/internal/modules/project/app/ports.go` | 用例的端口、跨模块端口、插入的行 | 2、4、7、9–12 |
| `server/internal/modules/project/app/cascade.go`、`server/internal/modules/project/app/cascade_test.go` | `Cascade`：`DeleteWorkspaceProjects`、`DemoteToGuest` | 2、12 |
| `server/internal/modules/project/domain/actions.go` | 操作名；`Actions()` 先是空的（裁定 S3） | 2、7、9–11 |
| `server/internal/modules/project/module.go` | `Provide`、`New`、`Register`、`Cascade()`、`Actions()` | 2、8–12 |
| `server/internal/modules/workspace/app/ports.go`、`server/internal/modules/workspace/app/delete_workspace.go`、`server/internal/modules/workspace/app/delete_workspace_test.go`、`server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go`（修改） | `ProjectCascade` 端口；`cascade()` 的最后一步 | 2、12、13 |
| `server/internal/modules/workspace/app/clock_test.go`（修改） | 写在锁之后读时钟，连带用同一个时刻 | 2、12、13 |
| `server/internal/modules/workspace/module.go`（修改） | `Deps.Projects`；`Provide` 的目录 | 2、5、12、13 |
| `api/modules/workspace.yaml`（修改） | `deleteWorkspace` 的说明写上项目 | 2 |
| `server/internal/bootstrap/app.go`（修改） | `project.Provide`、`project.New` 在 `workspace.New` 之前；注册；`ProjectAccess` 接进 `access` | 2、8、9 |
| `server/internal/bootstrap/actions_test.go`、`server/internal/bootstrap/workspace_deletion_test.go`、`server/internal/bootstrap/interleaving_answers_test.go`（修改） | `moduleActions` 加 `project`；组合的删除测试准备项目的行、项目一步失败时回滚；交错测试照新的构造 | 2、13 |
| `server/internal/modules/project/domain/project.go`、`server/internal/modules/project/domain/project_test.go` | `Project`、`NewProject`、`CheckNewProject`、`CanLead`、`Identifier`、`ValidIdentifier` | 3、4、10 |
| `server/internal/modules/project/domain/logo.go`、`server/internal/modules/project/domain/state.go`、`server/internal/modules/project/domain/state_test.go`、`server/internal/modules/project/domain/preferences.go`、`server/internal/modules/project/domain/preferences_test.go` | 图标；默认的 6 个状态；侧边栏的位置 | 3 |
| `server/internal/modules/project/domain/errors.go` | 模块的错误 | 4、7、9 |
| `server/internal/modules/project/adapter/postgres/queries/projects.sql`、`server/internal/modules/project/adapter/postgres/queries/members.sql`、`server/internal/modules/project/adapter/postgres/queries/preferences.sql`、`server/internal/modules/project/adapter/postgres/queries/states.sql` | 插入、`GetProject`、`LowestSortOrder`、`IdentifierTaken`、`ListProjects` | 4、10、11 |
| `server/internal/modules/project/adapter/postgres/projects.go`、`server/internal/modules/project/adapter/postgres/projects_test.go`、`server/internal/modules/project/adapter/postgres/rows.go`、`server/internal/modules/project/adapter/postgres/rows_test.go` | 项目的读写；项目之下的行 | 4、10、11 |
| `server/internal/modules/project/adapter/postgres/gen/projects.sql.go`、`server/internal/modules/project/adapter/postgres/gen/members.sql.go`、`server/internal/modules/project/adapter/postgres/gen/preferences.sql.go`、`server/internal/modules/project/adapter/postgres/gen/states.sql.go`（生成） | | 4、10、11 |
| `server/internal/modules/workspace/adapter/postgres/queries/directory.sql`、`server/internal/modules/workspace/adapter/postgres/directory.go`、`server/internal/modules/workspace/adapter/postgres/directory_test.go`、`server/internal/modules/workspace/adapter/postgres/store.go`（修改）、`server/internal/modules/workspace/app/directory.go` | `WorkspaceDirectory`、`WorkspaceMembers` | 5 |
| `server/internal/modules/workspace/adapter/postgres/gen/directory.sql.go`（生成） | | 5 |
| `server/internal/bootstrap/permission_matrix_columns_test.go`、`server/internal/bootstrap/permission_matrix_targets_test.go` | 项目级的列、账户和目标；格子指向的目标 | 6、9 |
| `server/internal/bootstrap/permission_matrix_test.go`、`server/internal/bootstrap/permission_matrix_coverage_test.go`、`server/internal/bootstrap/permission_matrix_seeded_test.go`、`server/internal/bootstrap/permission_matrix_invitations_test.go`、`server/internal/bootstrap/permission_matrix_workspace_test.go`（修改） | 每行带自己的列；准备的项目；按参数列出的"不是目标" | 6、8、10、11 |
| `server/internal/bootstrap/permission_matrix_project_test.go` | `project` 的矩阵行 | 8–11 |
| `server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`（修改） | 规则表的 4 行和它们的格子 | 7、9–11 |
| `server/internal/modules/project/app/create_project.go`、`server/internal/modules/project/app/create_project_test.go`、`server/internal/modules/project/app/clock_test.go`、`server/internal/modules/project/app/fakes_test.go`、`server/internal/modules/project/app/fakes_create_test.go` | `createProject`；假实现 | 7、10 |
| `api/modules/project.yaml`；`api/openapi.yaml`（修改） | 4 个操作和它们的结构 | 8–11（`openapi.yaml`：8–10） |
| `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`（生成） | | 2、8–11 |
| `server/internal/modules/project/adapter/http/gen/oapi-codegen.yaml` | 生成的配置 | 8 |
| `server/internal/modules/project/adapter/http/gen/server.gen.go`、`server/internal/modules/project/adapter/http/gen/bodyshape.gen.go`（生成） | | 8–11（`bodyshape.gen.go`：8） |
| `server/internal/modules/project/adapter/http/handler.go`、`server/internal/modules/project/adapter/http/handler_test.go`、`server/internal/modules/project/adapter/http/projects.go`、`server/internal/modules/project/adapter/http/projects_test.go` | 用例的接口；项目的 handler；`TestMain` 经 `apitest.Main` | 8–11 |
| `server/internal/platform/httpserver/apitest/main_callers_test.go` | 每个模块的 HTTP 测试经 `apitest.Main(m, "<模块>")` | 8 |
| `server/internal/bootstrap/ports.go`、`server/internal/bootstrap/ports_test.go`（修改） | `projectWorkspaces`、`accessProjects` 两个转换 | 8–10 |
| `server/internal/bootstrap/project_test.go` | 组合出的 app 上建项目 | 8 |
| `server/internal/bootstrap/project_wiring_test.go` | `project.New` 接的时钟和事务：建项目的时刻在请求之内，状态插入失败时 500、什么都不留 | 8 |
| `server/internal/bootstrap/project_access_test.go` | 真实的存储上 `Authorizer` 只认目标工作区的项目（第 8 条移交） | 9 |
| `web/apps/web/helpers/authentication.helper.ts`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json`（修改） | 新码的文案 | 8、9 |
| `server/internal/modules/access/app/ports.go`、`server/internal/modules/access/app/authorizer.go`、`server/internal/modules/access/app/authorizer_test.go`、`server/internal/modules/access/module.go`（修改） | `ProjectAccess` 端口；判定读项目的事实 | 9 |
| `server/internal/modules/project/adapter/postgres/queries/access.sql`、`server/internal/modules/project/adapter/postgres/access.go`、`server/internal/modules/project/adapter/postgres/access_test.go`、`server/internal/modules/project/app/access.go` | `ProjectFacts` | 9 |
| `server/internal/modules/project/adapter/postgres/gen/access.sql.go`（生成） | | 9 |
| `server/internal/modules/project/app/get_project.go`、`server/internal/modules/project/app/get_project_test.go` | `getProject` | 9 |
| `server/internal/modules/project/app/check_identifier.go`、`server/internal/modules/project/app/check_identifier_test.go` | `checkProjectIdentifier` | 10 |
| `server/internal/modules/project/domain/visibility.go`、`server/internal/modules/project/domain/visibility_test.go`、`server/internal/modules/project/app/list_projects.go`、`server/internal/modules/project/app/list_projects_test.go`、`server/internal/modules/project/adapter/postgres/list_test.go` | `listProjects` 和可见性 | 11 |
| `server/internal/bootstrap/project_visibility_test.go` | 列表等于逐个判定（9.3） | 11 |
| `server/internal/modules/workspace/app/update_member.go`、`server/internal/modules/workspace/app/update_member_test.go`、`server/internal/modules/workspace/app/list_members_test.go`（修改） | 改为访客时 `DemoteToGuest` | 12 |
| `server/internal/bootstrap/demotion_test.go` | 组合出的 app 上的两处降为访客 | 12、13 |
| `server/internal/bootstrap/interleaving_roles_test.go`（修改） | 交错测试照新的构造 | 12 |
| `server/internal/modules/workspace/app/accept_invitation.go`、`server/internal/modules/workspace/app/accept_invitation_test.go`、`server/internal/modules/workspace/app/list_invitations_test.go`、`server/internal/bootstrap/invitations_test.go`（修改） | 接受恢复为访客时 `DemoteToGuest` | 13 |
| `e2e/fixtures/api.ts`、`e2e/fixtures/auth.ts`、`e2e/fixtures/assert/workspace.ts`（修改）；`e2e/fixtures/assert/project.ts` | 建项目、账户的 id；项目的断言；删除工作区的表加上项目的四张 | 14 |
| `e2e/stories/project/p1-create-project.spec.ts`；`e2e/stories/workspace/w3-workspace-settings.spec.ts`（修改） | P1 的接口版本；W3 加上项目的连带 | 14 |
| `docs/v0/plane-diff.md`、`docs/v0/M3-workspace-project/handoffs/M2-closeout.md`、`docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md`、`docs/v0/M3-workspace-project/handoffs/M1-P3-trim-platform.md`（修改） | 3.20 中 P4a 的一行；交接的处理结果 | 15 |

---

### Task 1: `workspace/app/fakes_test.go` 按用例拆开

**Files:**
- Create: `server/internal/modules/workspace/app/fakes_accounts_test.go`、`server/internal/modules/workspace/app/fakes_members_test.go`、`server/internal/modules/workspace/app/fakes_preferences_test.go`、`server/internal/modules/workspace/app/fakes_workspaces_test.go`
- Modify（完整内容）: `server/internal/modules/workspace/app/fakes_test.go`

**Interfaces:** 没有新接口。P2 的 `fakeStore` 和它的方法、P3 的假账户和资料按它们服务的用例分进四个文件；`fakes_test.go` 留下共用的部分（调用记录、`fakeTx`、时钟、`fakeAuthorizer`）。只移动，不改写：`go vet` 和全部测试照旧通过（P3 review 第 6 节，P3 spec 第 3 节第 14 条）。

**Tests:** 没有新测试；`workspace/app` 的全部测试照旧通过。

- [ ] **Step 1: 拆开**

`server/internal/modules/workspace/app/fakes_test.go`（完整内容，101 行）：

````whole server/internal/modules/workspace/app/fakes_test.go
package app_test

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The write use cases' clock stands at clockNow, finer than a stored time:
// the fakes store a time as PostgreSQL's timestamptz does, to the
// microsecond, and answer the row as stored, at now. A use case that
// answered its own instant instead of the row as stored would answer
// clockNow.
var (
	clockNow = time.Date(2026, 9, 29, 10, 0, 0, 123456789, time.UTC)
	now      = stored(clockNow)
)

// stored is t as the database keeps it.
func stored(t time.Time) time.Time {
	return t.Truncate(time.Microsecond)
}

// clockAt stands at an instant, its nanoseconds kept, unlike
// clocktest.Fixed. With a log, each read is logged as "Now" among the
// fakes' calls, so a test sees when the use case reads it.
type clockAt struct {
	at  time.Time
	log *callLog
}

func (c clockAt) Now() time.Time {
	if c.log != nil {
		c.log.calls = append(c.log.calls, "Now")
	}
	return c.at
}

// fakeTx runs fn in a context marked as inside the transaction; the fakes
// record whether each call happened there. commitErr, when set, is the
// commit failing after fn succeeded.
type fakeTx struct {
	calls     int
	commitErr error
}

type inTxKey struct{}

func (f *fakeTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	f.calls++
	if err := fn(context.WithValue(ctx, inTxKey{}, true)); err != nil {
		return err
	}
	return f.commitErr
}

// callLog records the calls of the fakes that share it, in order, each with
// its arguments, and " outside tx" when it ran outside a transaction.
type callLog struct{ calls []string }

func (l *callLog) add(ctx context.Context, format string, args ...any) {
	call := fmt.Sprintf(format, args...)
	if ctx.Value(inTxKey{}) != true {
		call += " outside tx"
	}
	l.calls = append(l.calls, call)
}

// show is *s quoted, or <nil>.
func show(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%q", *s)
}

// grantKey is one (user, workspace) pair of fakeAuthorizer.
type grantKey struct{ user, workspace uuid.UUID }

// fakeAuthorizer answers each (user, workspace) pair its grant, or its
// error, and ErrNotVisible for any other; it logs every call.
type fakeAuthorizer struct {
	log    *callLog
	grants map[grantKey]shared.Grant
	errs   map[grantKey]error
}

func (f *fakeAuthorizer) Authorize(ctx context.Context, actor shared.Actor, action shared.Action, t shared.Target) (shared.Grant, error) {
	f.log.add(ctx, "Authorize %s %s on %s/%s", actor.UserID, action, t.WorkspaceID, t.ProjectID)
	key := grantKey{actor.UserID, t.WorkspaceID}
	if err := f.errs[key]; err != nil {
		return shared.Grant{}, err
	}
	if g, ok := f.grants[key]; ok {
		return g, nil
	}
	return shared.Grant{}, shared.ErrNotVisible
}
````

`server/internal/modules/workspace/app/fakes_workspaces_test.go`（新文件，163 行）：

````file server/internal/modules/workspace/app/fakes_workspaces_test.go
package app_test

import (
	"context"
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// fakeWorkspaces is the repositories: it logs every call with its
// arguments, answers from the workspaces it holds, and fails a call with
// the error set for it; a read fails for the argument its error is set
// for, wrapped as the store wraps it, so the use case must match with
// errors.Is. The workspaces' methods are here; the members' and the display
// settings' are in fakes_members_test.go and fakes_preferences_test.go.
type fakeWorkspaces struct {
	log         *callLog
	workspaces  []domain.Workspace // by slug for WorkspaceBySlug, SlugTaken and the locks; by id for UpdateWorkspace
	lists       map[uuid.UUID][]domain.Workspace
	createErr   error
	memberErr   error               // for CreateMember, which then stores nothing
	updateErr   error               // for UpdateWorkspace
	deleteErrs  map[string]error    // by step, e.g. "DeleteWorkspaceMembers"
	listErrs    map[uuid.UUID]error // by user, for ListWorkspaces
	slugErrs    map[string]error    // by slug, for WorkspaceBySlug and SlugTaken
	lockErrs    map[string]error    // by slug, for the locks
	members     []app.MemberRow
	memberships map[uuid.UUID][]domain.Membership // by workspace, for ListMembers and MemberByID
	membersErr  error                             // for ListMembers and MemberByID
	roleErr     error                             // for UpdateMemberRole
	onLock      func()                            // run by LockWorkspace once it has locked: what changed while it waited
	prefs       map[prefsKey]domain.Preferences
	prefIDs     map[prefsKey]uuid.UUID // the id each row UpsertPreferences inserted took
	prefsErr    error                  // for Preferences and UpsertPreferences
	upserts     []app.PreferencesRow
}

func (f *fakeWorkspaces) CreateWorkspace(ctx context.Context, w app.WorkspaceRow) (domain.Workspace, error) {
	size := "<nil>"
	if w.OrganizationSize != nil {
		size = *w.OrganizationSize
	}
	f.log.add(ctx, "CreateWorkspace %s %q %s %s %s by %s at %s", w.ID, w.Name, w.Slug, size, w.Timezone, w.CreatedBy, w.Now.Format(time.RFC3339Nano))
	if f.createErr != nil {
		return domain.Workspace{}, f.createErr
	}
	return domain.Workspace{ID: w.ID, Name: w.Name, Slug: w.Slug, OrganizationSize: w.OrganizationSize, Timezone: w.Timezone,
		CreatedAt: stored(w.Now), UpdatedAt: stored(w.Now)}, nil
}

func (f *fakeWorkspaces) CreateMember(ctx context.Context, m app.MemberRow) error {
	f.log.add(ctx, "CreateMember %s in %s as %d by %s at %s", m.MemberID, m.WorkspaceID, m.Role, m.CreatedBy, m.Now.Format(time.RFC3339Nano))
	if f.memberErr != nil {
		return fmt.Errorf("create workspace member: %w", f.memberErr)
	}
	f.members = append(f.members, m)
	return nil
}

func (f *fakeWorkspaces) ListWorkspaces(ctx context.Context, userID uuid.UUID) ([]domain.Workspace, error) {
	f.log.add(ctx, "ListWorkspaces %s", userID)
	if err := f.listErrs[userID]; err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	return f.lists[userID], nil
}

func (f *fakeWorkspaces) WorkspaceBySlug(ctx context.Context, slug string) (domain.Workspace, error) {
	f.log.add(ctx, "WorkspaceBySlug %s", slug)
	if err := f.slugErrs[slug]; err != nil {
		return domain.Workspace{}, fmt.Errorf("workspace by slug: %w", err)
	}
	i := slices.IndexFunc(f.workspaces, func(w domain.Workspace) bool { return w.Slug == slug })
	if i < 0 {
		return domain.Workspace{}, app.ErrNotFound
	}
	return f.workspaces[i], nil
}

func (f *fakeWorkspaces) SlugTaken(ctx context.Context, slug string) (bool, error) {
	f.log.add(ctx, "SlugTaken %s", slug)
	if err := f.slugErrs[slug]; err != nil {
		return false, fmt.Errorf("check slug: %w", err)
	}
	return slices.ContainsFunc(f.workspaces, func(w domain.Workspace) bool { return w.Slug == slug }), nil
}

func (f *fakeWorkspaces) LockWorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, error) {
	f.log.add(ctx, "LockWorkspaceBySlug %s", slug)
	return f.lock(slug)
}

// lock answers the id of the workspace with slug, app.ErrNotFound when it
// holds none, and the error set for slug wrapped as the store wraps it.
func (f *fakeWorkspaces) lock(slug string) (uuid.UUID, error) {
	if err := f.lockErrs[slug]; err != nil {
		return uuid.UUID{}, fmt.Errorf("lock the workspace row: %w", err)
	}
	i := slices.IndexFunc(f.workspaces, func(w domain.Workspace) bool { return w.Slug == slug })
	if i < 0 {
		return uuid.UUID{}, app.ErrNotFound
	}
	return f.workspaces[i].ID, nil
}

func (f *fakeWorkspaces) UpdateWorkspace(ctx context.Context, id uuid.UUID, p domain.WorkspacePatch, by uuid.UUID, now time.Time) (domain.Workspace, error) {
	f.log.add(ctx, "UpdateWorkspace %s name=%s size=%s timezone=%s by %s at %s", id, show(p.Name), show(p.OrganizationSize), show(p.Timezone),
		by, now.Format(time.RFC3339Nano))
	if f.updateErr != nil {
		return domain.Workspace{}, fmt.Errorf("update workspace: %w", f.updateErr)
	}
	i := slices.IndexFunc(f.workspaces, func(w domain.Workspace) bool { return w.ID == id })
	if i < 0 {
		return domain.Workspace{}, fmt.Errorf("update workspace %s: no such row", id)
	}
	w := f.workspaces[i]
	if p.Name != nil {
		w.Name = *p.Name
	}
	if p.OrganizationSize != nil {
		w.OrganizationSize = p.OrganizationSize
	}
	if p.Timezone != nil {
		w.Timezone = *p.Timezone
	}
	w.UpdatedAt = stored(now)
	return w, nil
}

func (f *fakeWorkspaces) ShareWorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, error) {
	f.log.add(ctx, "ShareWorkspaceBySlug %s", slug)
	return f.lock(slug)
}

func (f *fakeWorkspaces) DeleteWorkspace(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	return f.deleteStep(ctx, "DeleteWorkspace", id, by, now)
}

func (f *fakeWorkspaces) DeleteWorkspaceInvitations(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return f.deleteStep(ctx, "DeleteWorkspaceInvitations", workspaceID, by, now)
}

func (f *fakeWorkspaces) DeleteWorkspaceMembers(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return f.deleteStep(ctx, "DeleteWorkspaceMembers", workspaceID, by, now)
}

func (f *fakeWorkspaces) DeleteWorkspacePreferences(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return f.deleteStep(ctx, "DeleteWorkspacePreferences", workspaceID, by, now)
}

// deleteStep logs a step of the deletion and fails it with the error set
// for it, wrapped as the store wraps it.
func (f *fakeWorkspaces) deleteStep(ctx context.Context, step string, id, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "%s %s by %s at %s", step, id, by, now.Format(time.RFC3339Nano))
	if err := f.deleteErrs[step]; err != nil {
		return fmt.Errorf("%s: %w", step, err)
	}
	return nil
}
````

`server/internal/modules/workspace/app/fakes_members_test.go`（新文件，73 行）：

````file server/internal/modules/workspace/app/fakes_members_test.go
package app_test

import (
	"context"
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The members' repositories of fakeWorkspaces: MemberLister and
// MemberUpdater.

func (f *fakeWorkspaces) ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]domain.Membership, error) {
	f.log.add(ctx, "ListMembers %s", workspaceID)
	if f.membersErr != nil {
		return nil, fmt.Errorf("list workspace members: %w", f.membersErr)
	}
	return f.memberships[workspaceID], nil
}

func (f *fakeWorkspaces) MemberByID(ctx context.Context, id uuid.UUID) (domain.Membership, error) {
	f.log.add(ctx, "MemberByID %s", id)
	if f.membersErr != nil {
		return domain.Membership{}, fmt.Errorf("read workspace member: %w", f.membersErr)
	}
	for _, list := range f.memberships {
		if i := slices.IndexFunc(list, func(m domain.Membership) bool { return m.ID == id }); i >= 0 {
			return list[i], nil
		}
	}
	return domain.Membership{}, app.ErrNotFound
}

// LockWorkspace answers as the slug locks do, for the workspace with id.
func (f *fakeWorkspaces) LockWorkspace(ctx context.Context, id uuid.UUID) error {
	f.log.add(ctx, "LockWorkspace %s", id)
	return f.lockByID(id)
}

// lockByID answers as lock does, for the workspace with id; once locked, it
// runs onLock.
func (f *fakeWorkspaces) lockByID(id uuid.UUID) error {
	i := slices.IndexFunc(f.workspaces, func(w domain.Workspace) bool { return w.ID == id })
	if i < 0 {
		return app.ErrNotFound
	}
	if _, err := f.lock(f.workspaces[i].Slug); err != nil {
		return err
	}
	if f.onLock != nil {
		f.onLock()
	}
	return nil
}

func (f *fakeWorkspaces) UpdateMemberRole(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) (domain.Membership, error) {
	f.log.add(ctx, "UpdateMemberRole %s to %d by %s at %s", id, role, by, now.Format(time.RFC3339Nano))
	if f.roleErr != nil {
		return domain.Membership{}, fmt.Errorf("update workspace member: %w", f.roleErr)
	}
	for _, list := range f.memberships {
		if i := slices.IndexFunc(list, func(m domain.Membership) bool { return m.ID == id }); i >= 0 {
			list[i].Role = role
			return list[i], nil
		}
	}
	return domain.Membership{}, fmt.Errorf("update workspace member %s: no such row", id)
}
````

`server/internal/modules/workspace/app/fakes_preferences_test.go`（新文件，53 行）：

````file server/internal/modules/workspace/app/fakes_preferences_test.go
package app_test

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// prefsKey is one (workspace, user) pair of fakeWorkspaces' preferences.
type prefsKey struct{ workspace, user uuid.UUID }

func (f *fakeWorkspaces) Preferences(ctx context.Context, workspaceID, userID uuid.UUID) (domain.Preferences, bool, error) {
	f.log.add(ctx, "Preferences %s %s", workspaceID, userID)
	if f.prefsErr != nil {
		return domain.Preferences{}, false, fmt.Errorf("read workspace preferences: %w", f.prefsErr)
	}
	p, found := f.prefs[prefsKey{workspaceID, userID}]
	return p, found, nil
}

// UpsertPreferences logs the row without its id, which the use case makes
// anew each time; upserts keeps it whole. A row it inserts takes r's id and
// keeps it through later changes, as the store's does (prefIDs).
func (f *fakeWorkspaces) UpsertPreferences(ctx context.Context, r app.PreferencesRow) (domain.Preferences, error) {
	var limit *string
	if r.Patch.NavigationProjectLimit != nil {
		limit = ptr(fmt.Sprint(*r.Patch.NavigationProjectLimit))
	}
	f.log.add(ctx, "UpsertPreferences %s %s mode=%s limit=%s at %s", r.WorkspaceID, r.UserID, show(r.Patch.NavigationControl), show(limit),
		r.Now.Format(time.RFC3339Nano))
	f.upserts = append(f.upserts, r)
	if f.prefsErr != nil {
		return domain.Preferences{}, fmt.Errorf("write workspace preferences: %w", f.prefsErr)
	}
	key := prefsKey{r.WorkspaceID, r.UserID}
	p, found := f.prefs[key]
	if !found {
		p = domain.DefaultPreferences()
		if f.prefIDs == nil {
			f.prefIDs = map[prefsKey]uuid.UUID{}
		}
		f.prefIDs[key] = r.ID
	}
	if f.prefs == nil {
		f.prefs = map[prefsKey]domain.Preferences{}
	}
	f.prefs[key] = p.Apply(r.Patch)
	return f.prefs[key], nil
}
````

`server/internal/modules/workspace/app/fakes_accounts_test.go`（新文件，58 行）：

````file server/internal/modules/workspace/app/fakes_accounts_test.go
package app_test

import (
	"context"
	"fmt"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
)

// fakeAccounts answers the state of the accounts it holds, by id and by
// address, and logs each lock.
type fakeAccounts struct {
	log      *callLog
	accounts []app.AccountState
	err      error
}

func (f *fakeAccounts) ShareAccount(ctx context.Context, id uuid.UUID) (app.AccountState, bool, error) {
	f.log.add(ctx, "ShareAccount %s", id)
	i := slices.IndexFunc(f.accounts, func(a app.AccountState) bool { return a.ID == id })
	if f.err != nil || i < 0 {
		return app.AccountState{}, false, f.err
	}
	return f.accounts[i], true, nil
}

func (f *fakeAccounts) ShareAccountByEmail(ctx context.Context, email string) (app.AccountState, bool, error) {
	f.log.add(ctx, "ShareAccountByEmail %s", email)
	i := slices.IndexFunc(f.accounts, func(a app.AccountState) bool { return a.Email == email })
	if f.err != nil || i < 0 {
		return app.AccountState{}, false, f.err
	}
	return f.accounts[i], true, nil
}

// fakeProfiles answers the profiles it holds of the ids asked for, in the
// order it holds them, and logs each call with its ids.
type fakeProfiles struct {
	log      *callLog
	profiles []app.PublicProfile
	err      error
}

func (f *fakeProfiles) PublicProfiles(ctx context.Context, ids []uuid.UUID) ([]app.PublicProfile, error) {
	f.log.add(ctx, "PublicProfiles %v", ids)
	if f.err != nil {
		return nil, fmt.Errorf("read public profiles: %w", f.err)
	}
	var out []app.PublicProfile
	for _, p := range f.profiles {
		if slices.Contains(ids, p.ID) {
			out = append(out, p)
		}
	}
	return out, nil
}
````

- [ ] **Step 2: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/workspace/...`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 3: 提交**

```bash
git add server/internal/modules/workspace/app/fakes_accounts_test.go server/internal/modules/workspace/app/fakes_members_test.go server/internal/modules/workspace/app/fakes_preferences_test.go server/internal/modules/workspace/app/fakes_test.go server/internal/modules/workspace/app/fakes_workspaces_test.go
```
```bash
git commit -m "refactor(M3/P4a): split workspace/app's shared fakes by use case

The 401-line fakes_test.go (P3 spec 3.14, P3 review 6) becomes the shared
part (the call log, the transaction, the clock, the authorizer) and one
file each for the workspaces, the members, the display settings and the
accounts. Only moved: P4a adds the projects' cascade beside them.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**变异**：只移动代码，没有要变异的性质；`make test` 通过即可。

**Done when:** 五个文件都在约 400 行以内，`workspace/app` 的测试一个不少地通过。

---

### Task 2: 迁移 `00010`–`00013`；`project` 模块的骨架；删除工作区连带项目

**Files:**
- Create: `server/internal/modules/project/adapter/postgres/cascade.go`、`server/internal/modules/project/adapter/postgres/cascade_test.go`、`server/internal/modules/project/adapter/postgres/queries/cascade.sql`、`server/internal/modules/project/adapter/postgres/store.go`、`server/internal/modules/project/adapter/postgres/store_test.go`、`server/internal/modules/project/app/cascade.go`、`server/internal/modules/project/app/cascade_test.go`、`server/internal/modules/project/app/ports.go`、`server/internal/modules/project/domain/actions.go`、`server/internal/modules/project/module.go`、`server/internal/modules/workspace/app/fakes_projects_test.go`、`server/migrations/project_schema_test.go`、`server/migrations/sql/00010_project_projects.sql`、`server/migrations/sql/00011_project_project_members.sql`、`server/migrations/sql/00012_project_project_user_properties.sql`、`server/migrations/sql/00013_project_states.sql`
- Modify: `api/modules/workspace.yaml`、`deploy/runtime-grants.sql`、`server/internal/bootstrap/actions_test.go`、`server/internal/bootstrap/app.go`、`server/internal/bootstrap/interleaving_answers_test.go`、`server/internal/bootstrap/workspace_deletion_test.go`、`server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go`、`server/internal/modules/workspace/app/clock_test.go`、`server/internal/modules/workspace/app/delete_workspace.go`、`server/internal/modules/workspace/app/delete_workspace_test.go`、`server/internal/modules/workspace/app/ports.go`、`server/internal/modules/workspace/module.go`、`server/migrations/schema_test.go`、`server/sqlc.yaml`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/project/adapter/postgres/gen/cascade.sql.go`、`server/internal/modules/project/adapter/postgres/gen/db.go`、`server/internal/modules/project/adapter/postgres/gen/models.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.3，M3 设计 3.3、3.6、4.6–4.9、6.6）：四张表；`workspace/app.ProjectCascade`（`DeleteWorkspaceProjects(ctx, workspaceID, by uuid.UUID, now time.Time) error`，Task 12 加 `DemoteToGuest`）；`NewDeleteWorkspace(workspaces, projects ProjectCascade, auth, tx, clock, logger)`，`cascade()` 的最后一步是 `u.projects.DeleteWorkspaceProjects`；`project/app.Cascade`（`NewCascade(projects WorkspaceProjectsDeleter)`，Task 12 加第二个参数），`deletion()` 依次是项目、项目成员、项目显示设置、状态（P7 在最后加标签）；`project/app.WorkspaceProjectsDeleter` 的四个方法；`project.New(Deps{Pool})`、`(*Module).Cascade()`、`project.Actions()`（空的，裁定 S3）；`workspace.Deps.Projects`。
- 组合（6.6）：`bootstrap` 在 `workspace.New` 之前建 `project.New`，把它的 `Cascade()` 交给 `workspace.Deps.Projects`；`moduleActions` 加 `"project": project.Actions()`。
- 使用者：Task 12 的 `DemoteToGuest`；Task 4 起项目的存储；P7 在 `deletion()` 的最后加标签。

**Tests:**
- `server/migrations/schema_test.go`：迁移数 13（`TestMigrationsGoUpDownAndUpAgain`）；`TestConstraintAndIndexNames` 的期望加上 `project_schema_test.go` 的 `projectNames`（四张表的 52 个名字和种类：`iuw` 的部分唯一键、`iw` 的 `project_members_member_id_idx`、每条外键的 `f c`、`f n`）。
- `project/adapter/postgres/cascade_test.go`：`TestDeletingAWorkspaceSoftDeletesItsProjects`（两个工作区；四张表的行在同一时刻、由同一个账户删除，已归档的项目、已结束的成员关系、分诊状态一起；此前删除的行保持原来的时间；另一个工作区一行不动；再跑一次什么都不变）。
- `project/app/cascade_test.go`：`TestDeleteWorkspaceProjects`（四步的顺序、同一个工作区、账户、时刻；一步失败原样返回，之后的不运行）。
- `workspace/app`：`TestDeleteWorkspaceLocksThenDecidesThenCascades` 的调用记录最后多一步 `DeleteWorkspaceProjects`；`TestDeleteWorkspaceRefusals` 加项目一步的失败；`TestEachWriteReadsTheClockUnderItsLock` 的删除用同一个时刻。`workspace/adapter/postgres`：`TestAFailedDeletionLeavesTheWorkspace` 经不删除任何东西的假连带。
- `bootstrap/workspace_deletion_test.go`：两个工作区各准备项目、项目成员、显示设置、状态的行（照表的 SQL：存储的插入在 Task 4）；`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt` 从目录找到四张新表，不加豁免；`TestAFailedProjectsStepRollsTheDeletionBack`（组合出的 app 上 `states` 表在请求中改名，最后一条语句失败：500，两个工作区的每一行都不变）。
- `bootstrap/actions_test.go`：`project` 的空 `Actions()` 在 `moduleActions` 中，`TestEveryModuleDeclaresItsActionsOrHasNone`、`TestEveryActionHasARuleAndEveryRuleAnAction` 通过（裁定 S3）。

- [ ] **Step 1: 迁移、运行时权限、sqlc**

`server/migrations/sql/00010_project_projects.sql`（新文件，62 行）：

````file server/migrations/sql/00010_project_projects.sql
-- projects：Plane 的 projects 表（36 列）按 M3 设计 4.6 保留 22 列，另加计数列 last_issue_sequence，共 23 列。
-- 负责人、默认负责人改为 ON DELETE SET NULL（3.15）；last_issue_sequence 是工作项编号的计数列，M4 取号，M3 不读写它。
-- 删除工作区时随之软删除（DeleteWorkspaceProjects，3.3）。

-- +goose Up
CREATE TABLE projects (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    -- Plane 的禁用字符（serializers/project.py:39-44，3.19）
    name varchar(255) NOT NULL CHECK (name <> '' AND name !~ '[&+,:;$^}{*=?@#\|''<>.()%!-]'),
    description text NOT NULL DEFAULT '',
    -- 转成大写后 1–10 个，只能是 A-Z、0-9 和 ÇŞĞİÖÜ（3.19）；列类型仍是 varchar(12)
    identifier varchar(12) NOT NULL CHECK (identifier ~ '^[A-Z0-9ÇŞĞİÖÜ]{1,10}$'),
    -- 0 私密、2 公开（快照的 >= 0 收紧）
    network smallint NOT NULL DEFAULT 2 CHECK (network IN (0, 2)),
    project_lead_id uuid REFERENCES users ON DELETE SET NULL,
    default_assignee_id uuid REFERENCES users ON DELETE SET NULL,
    cycle_view boolean NOT NULL DEFAULT false,
    module_view boolean NOT NULL DEFAULT false,
    issue_views_view boolean NOT NULL DEFAULT false,
    intake_view boolean NOT NULL DEFAULT false,
    guest_view_all_features boolean NOT NULL DEFAULT false,
    -- 模型的验证器 0–12
    archive_in integer NOT NULL DEFAULT 0 CHECK (archive_in BETWEEN 0 AND 12),
    -- {} 表示没有图标；键和值类型与接口的结构相同，字段都可选（3.19、5.2）
    logo_props jsonb NOT NULL DEFAULT '{}',
    -- 取值在领域层校验（3.13）
    timezone varchar(255) NOT NULL DEFAULT 'UTC',
    last_issue_sequence integer NOT NULL DEFAULT 0 CHECK (last_issue_sequence >= 0),
    archived_at timestamptz,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    -- 是对象、键的集合、出现的每个键的值类型，嵌套的 emoji、icon 也查到底（M2 设计 3.13；M3 设计 4.6）。
    -- 嵌套的 -> 要加括号：- 的优先级高于 ->
    CONSTRAINT projects_logo_props_check CHECK (CASE WHEN jsonb_typeof(logo_props) = 'object' THEN
            (logo_props - array['in_use', 'emoji', 'icon']) = '{}'::jsonb
        AND (NOT logo_props ? 'in_use' OR logo_props -> 'in_use' IN ('"emoji"'::jsonb, '"icon"'::jsonb))
        AND (NOT logo_props ? 'emoji' OR CASE WHEN jsonb_typeof(logo_props -> 'emoji') = 'object' THEN
                ((logo_props -> 'emoji') - array['value', 'url']) = '{}'::jsonb
            AND (NOT (logo_props -> 'emoji') ? 'value' OR jsonb_typeof(logo_props -> 'emoji' -> 'value') = 'string')
            AND (NOT (logo_props -> 'emoji') ? 'url' OR jsonb_typeof(logo_props -> 'emoji' -> 'url') = 'string')
          ELSE false END)
        AND (NOT logo_props ? 'icon' OR CASE WHEN jsonb_typeof(logo_props -> 'icon') = 'object' THEN
                ((logo_props -> 'icon') - array['name', 'color', 'background_color']) = '{}'::jsonb
            AND (NOT (logo_props -> 'icon') ? 'name' OR jsonb_typeof(logo_props -> 'icon' -> 'name') = 'string')
            AND (NOT (logo_props -> 'icon') ? 'color' OR jsonb_typeof(logo_props -> 'icon' -> 'color') = 'string')
            AND (NOT (logo_props -> 'icon') ? 'background_color'
                 OR jsonb_typeof(logo_props -> 'icon' -> 'background_color') = 'string')
          ELSE false END)
      ELSE false END)
);
-- 标识、名称在工作区的未删除项目中唯一（3.19）；删除之后可以再用。按工作区列出项目也用后一个
CREATE UNIQUE INDEX projects_workspace_id_identifier_key ON projects (workspace_id, identifier) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX projects_workspace_id_name_key ON projects (workspace_id, name) WHERE deleted_at IS NULL;
-- 物理级联（M4 的 60 天清理）按它找子行（4 节开头）
CREATE INDEX projects_workspace_id_idx ON projects (workspace_id);

-- +goose Down
DROP TABLE projects;
````

`server/migrations/sql/00011_project_project_members.sql`（新文件，28 行）：

````file server/migrations/sql/00011_project_project_members.sql
-- project_members：Plane 的 project_members 表（16 列）按 M3 设计 4.7 保留 11 列。
-- 移出、离开、停用都只把 is_active 改为假，行不删除；删除项目、删除工作区时随之软删除。

-- +goose Up
CREATE TABLE project_members (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    -- Plane 可空，改为非空：没有成员的成员关系没有意义
    member_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    -- 访客 5、成员 15、管理员 20，同 workspace_members.role
    role smallint NOT NULL DEFAULT 5 CHECK (role IN (5, 15, 20)),
    is_active boolean NOT NULL DEFAULT true,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE UNIQUE INDEX project_members_project_id_member_id_key ON project_members (project_id, member_id) WHERE deleted_at IS NULL;
-- 连带结束、降为访客、停用、我的项目角色（4.7）
CREATE INDEX project_members_member_id_idx ON project_members (member_id) WHERE deleted_at IS NULL;
-- 物理级联（M4 的 60 天清理）按它们找子行（4 节开头）
CREATE INDEX project_members_workspace_id_idx ON project_members (workspace_id);
CREATE INDEX project_members_project_id_idx ON project_members (project_id);

-- +goose Down
DROP TABLE project_members;
````

`server/migrations/sql/00012_project_project_user_properties.sql`（新文件，38 行）：

````file server/migrations/sql/00012_project_project_user_properties.sql
-- project_user_properties：Plane 的 project_user_properties 表（15 列）按 M3 设计 4.8 保留 11 列。
-- 一个账户在一个项目的显示设置（3.18）：导航偏好和侧边栏里项目的顺序，随项目成员关系一起建出。
-- 工作项列表的筛选和显示列（filters、display_filters、display_properties、rich_filters）由它们的使用者 M4 按自己的格式加回（3.18）。

-- +goose Up
CREATE TABLE project_user_properties (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    -- 外层恰好一个键 navigation，它恰好有 default_tab（字符串）和 hide_in_more_menu（数组）；外层的 CASE 让数组、
    -- 标量和 JSON null 也得到 check_violation（M2 设计 3.13）。默认值来自模型，去掉 pages（文档页已砍）
    preferences jsonb NOT NULL DEFAULT '{"navigation": {"default_tab": "work_items", "hide_in_more_menu": []}}'
        CHECK (CASE WHEN jsonb_typeof(preferences) = 'object' THEN
                (preferences - 'navigation') = '{}'::jsonb
            AND CASE WHEN jsonb_typeof(preferences -> 'navigation') = 'object' THEN
                    (preferences -> 'navigation') ?& array['default_tab', 'hide_in_more_menu']
                AND ((preferences -> 'navigation') - array['default_tab', 'hide_in_more_menu']) = '{}'::jsonb
                AND jsonb_typeof(preferences -> 'navigation' -> 'default_tab') = 'string'
                AND jsonb_typeof(preferences -> 'navigation' -> 'hide_in_more_menu') = 'array'
              ELSE false END
          ELSE false END),
    -- 侧边栏里项目的顺序（3.18）
    sort_order double precision NOT NULL DEFAULT 65535,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE UNIQUE INDEX project_user_properties_project_id_user_id_key ON project_user_properties (project_id, user_id)
    WHERE deleted_at IS NULL;
-- 物理级联（M4 的 60 天清理）按它们找子行（4 节开头）
CREATE INDEX project_user_properties_workspace_id_idx ON project_user_properties (workspace_id);
CREATE INDEX project_user_properties_project_id_idx ON project_user_properties (project_id);

-- +goose Down
DROP TABLE project_user_properties;
````

`server/migrations/sql/00013_project_states.sql`（新文件，33 行）：

````file server/migrations/sql/00013_project_states.sql
-- states：Plane 的 states 表（18 列）按 M3 设计 4.9 保留 14 列。
-- 分诊状态只由 group = 'triage' 识别，删除 is_triage；slug 没有读取者，删除（3.17）。建项目时建出默认的 6 个状态。

-- +goose Up
CREATE TABLE states (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    name varchar(255) NOT NULL CHECK (name <> ''),
    description text NOT NULL DEFAULT '',
    color varchar(255) NOT NULL,
    sequence double precision NOT NULL DEFAULT 65535,
    -- Plane 的 StateGroup；列名照搬，SQL 中带引号
    "group" varchar(20) NOT NULL DEFAULT 'backlog'
        CHECK ("group" IN ('backlog', 'unstarted', 'started', 'completed', 'cancelled', 'triage')),
    "default" boolean NOT NULL DEFAULT false,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
-- 名称在项目内唯一，区分大小写（照搬）
CREATE UNIQUE INDEX states_project_id_name_key ON states (project_id, name) WHERE deleted_at IS NULL;
-- 每个项目至多一个默认状态、至多一个分诊状态（3.17；名字按 4 节开头的补充）
CREATE UNIQUE INDEX states_project_id_default_key ON states (project_id) WHERE "default" AND deleted_at IS NULL;
CREATE UNIQUE INDEX states_project_id_triage_key ON states (project_id) WHERE "group" = 'triage' AND deleted_at IS NULL;
-- 物理级联（M4 的 60 天清理）按它们找子行（4 节开头）
CREATE INDEX states_workspace_id_idx ON states (workspace_id);
CREATE INDEX states_project_id_idx ON states (project_id);

-- +goose Down
DROP TABLE states;
````

`deploy/runtime-grants.sql`（修改，1 处）：

````old deploy/runtime-grants.sql
    TO nerve_runtime;
````
````new deploy/runtime-grants.sql
    TO nerve_runtime;
GRANT SELECT, INSERT, UPDATE, DELETE ON projects, project_members, project_user_properties, states TO nerve_runtime;
````

`server/sqlc.yaml`（修改，1 处）：

````old server/sqlc.yaml
        out: internal/modules/workspace/adapter/postgres/gen
        sql_package: pgx/v5
        emit_pointers_for_null_types: true
        overrides:
          - db_type: uuid
            go_type: {import: uuid, type: UUID}
          - db_type: uuid
            nullable: true
            go_type: {import: uuid, type: UUID, pointer: true}
          - db_type: timestamptz
            go_type: {import: time, type: Time}
          - db_type: timestamptz
            nullable: true
            go_type: {import: time, type: Time, pointer: true}

````
````new server/sqlc.yaml
        out: internal/modules/workspace/adapter/postgres/gen
        sql_package: pgx/v5
        emit_pointers_for_null_types: true
        overrides:
          - db_type: uuid
            go_type: {import: uuid, type: UUID}
          - db_type: uuid
            nullable: true
            go_type: {import: uuid, type: UUID, pointer: true}
          - db_type: timestamptz
            go_type: {import: time, type: Time}
          - db_type: timestamptz
            nullable: true
            go_type: {import: time, type: Time, pointer: true}
  - engine: postgresql
    schema:
      - migrations/sql/00010_project_projects.sql
      - migrations/sql/00011_project_project_members.sql
      - migrations/sql/00012_project_project_user_properties.sql
      - migrations/sql/00013_project_states.sql
    queries: internal/modules/project/adapter/postgres/queries
    gen:
      go:
        package: gen
        out: internal/modules/project/adapter/postgres/gen
        sql_package: pgx/v5
        emit_pointers_for_null_types: true
        overrides:
          - db_type: uuid
            go_type: {import: uuid, type: UUID}
          - db_type: uuid
            nullable: true
            go_type: {import: uuid, type: UUID, pointer: true}
          - db_type: timestamptz
            go_type: {import: time, type: Time}
          - db_type: timestamptz
            nullable: true
            go_type: {import: time, type: Time, pointer: true}

````

- [ ] **Step 2: 连带的查询；`deleteWorkspace` 的说明**

`server/internal/modules/project/adapter/postgres/queries/cascade.sql`（新文件，25 行）：

````file server/internal/modules/project/adapter/postgres/queries/cascade.sql
-- The steps of deleting a workspace's projects (M3 design 3.3, 3.6), each one statement under the workspace's FOR NO
-- KEY UPDATE, which deleteWorkspace took (convention 5): the rows of the workspace not deleted before, at the moment
-- and by the account of the workspace's deletion. Rows deleted before keep their moment.

-- name: DeleteWorkspaceProjects :exec
UPDATE projects
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL;

-- name: DeleteWorkspaceProjectMembers :exec
-- Active memberships and ended ones alike.
UPDATE project_members
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL;

-- name: DeleteWorkspaceProjectPreferences :exec
UPDATE project_user_properties
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL;

-- name: DeleteWorkspaceStates :exec
-- The triage states too.
UPDATE states
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL;
````

`api/modules/workspace.yaml`（修改，1 处）：

````old api/modules/workspace.yaml
        memberships and the members' display settings are soft-deleted in one
````
````new api/modules/workspace.yaml
        memberships, the members' display settings, and its projects with
        their memberships, display settings and states are soft-deleted in one
````

- [ ] **Step 3: 生成**

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `0906eeb83b15f0a672415da7822011c2c21e2a3517e7ed6f0c2cad7c15b53e79` | 1684 | `api/dist/openapi.yaml` |
| `fac3a4e597208da1fe81b247756fa20d243f6a92cab93b3a7e1b9871adfe8298` | 87 | `server/internal/modules/project/adapter/postgres/gen/cascade.sql.go` |
| `1dfb2b6312c3c3db25a8cad84c031995c3b0546c73b3585a0282c98e11397676` | 32 | `server/internal/modules/project/adapter/postgres/gen/db.go` |
| `f05678d2125c386344b67a38b3ae89c003e8768a726a1a1ae48dc4432522dcb1` | 82 | `server/internal/modules/project/adapter/postgres/gen/models.go` |
| `0d7dbda9dc299181a8c700481a804a285dcf7ea2fe08b3457f8e1ec95e096f63` | 1803 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/project/adapter/postgres/gen/cascade.sql.go server/internal/modules/project/adapter/postgres/gen/db.go server/internal/modules/project/adapter/postgres/gen/models.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 4: 迁移的测试**

`server/migrations/schema_test.go`（修改，5 处）：

````old server/migrations/schema_test.go
	if err != nil || len(up) != 9 {
		t.Fatalf("Up() = %d migrations, %v; want 9", len(up), err)
````
````new server/migrations/schema_test.go
	if err != nil || len(up) != 13 {
		t.Fatalf("Up() = %d migrations, %v; want 13", len(up), err)
````

````old server/migrations/schema_test.go
		{tablesQuery, []string{"api_tokens", "auth_sessions", "profiles", "river_job", "river_leader", "river_notification", "river_queue", "users",
			"workspace_member_invites", "workspace_members", "workspace_user_properties", "workspaces"}},
````
````new server/migrations/schema_test.go
		{tablesQuery, []string{"api_tokens", "auth_sessions", "profiles", "project_members", "project_user_properties", "projects", "river_job",
			"river_leader", "river_notification", "river_queue", "states", "users", "workspace_member_invites", "workspace_members",
			"workspace_user_properties", "workspaces"}},
````

````old server/migrations/schema_test.go
	if again, err := m.Up(ctx); err != nil || len(again) != 9 {
		t.Errorf("Up() again = %d migrations, %v; want 9", len(again), err)
````
````new server/migrations/schema_test.go
	if again, err := m.Up(ctx); err != nil || len(again) != 13 {
		t.Errorf("Up() again = %d migrations, %v; want 13", len(again), err)
````

````old server/migrations/schema_test.go
	// u when it is unique and w when it is partial (has a WHERE).
	want := []string{
````
````new server/migrations/schema_test.go
	// u when it is unique and w when it is partial (has a WHERE). The project
	// module's tables are in projectNames (project_schema_test.go).
	want := slices.Concat([]string{
````

````old server/migrations/schema_test.go
		"workspaces_updated_by_id_fkey f n",
	}
````
````new server/migrations/schema_test.go
		"workspaces_updated_by_id_fkey f n",
	}, projectNames)
	slices.Sort(want)
````

`server/migrations/project_schema_test.go`（新文件，59 行）：

````file server/migrations/project_schema_test.go
package migrations_test

// projectNames are the constraints and indexes of the project module's
// tables (M3 design 4.6–4.9), as TestConstraintAndIndexNames reads them:
// the part of its want that the migrations 00010–00013 add.
var projectNames = []string{
	"project_members_created_by_id_fkey f n",
	"project_members_member_id_fkey f c",
	"project_members_member_id_idx iw",
	"project_members_pkey iu",
	"project_members_pkey p",
	"project_members_project_id_fkey f c",
	"project_members_project_id_idx i",
	"project_members_project_id_member_id_key iuw",
	"project_members_role_check c",
	"project_members_updated_by_id_fkey f n",
	"project_members_workspace_id_fkey f c",
	"project_members_workspace_id_idx i",
	"project_user_properties_created_by_id_fkey f n",
	"project_user_properties_pkey iu",
	"project_user_properties_pkey p",
	"project_user_properties_preferences_check c",
	"project_user_properties_project_id_fkey f c",
	"project_user_properties_project_id_idx i",
	"project_user_properties_project_id_user_id_key iuw",
	"project_user_properties_updated_by_id_fkey f n",
	"project_user_properties_user_id_fkey f c",
	"project_user_properties_workspace_id_fkey f c",
	"project_user_properties_workspace_id_idx i",
	"projects_archive_in_check c",
	"projects_created_by_id_fkey f n",
	"projects_default_assignee_id_fkey f n",
	"projects_identifier_check c",
	"projects_last_issue_sequence_check c",
	"projects_logo_props_check c",
	"projects_name_check c",
	"projects_network_check c",
	"projects_pkey iu",
	"projects_pkey p",
	"projects_project_lead_id_fkey f n",
	"projects_updated_by_id_fkey f n",
	"projects_workspace_id_fkey f c",
	"projects_workspace_id_identifier_key iuw",
	"projects_workspace_id_idx i",
	"projects_workspace_id_name_key iuw",
	"states_created_by_id_fkey f n",
	"states_group_check c",
	"states_name_check c",
	"states_pkey iu",
	"states_pkey p",
	"states_project_id_default_key iuw",
	"states_project_id_fkey f c",
	"states_project_id_idx i",
	"states_project_id_name_key iuw",
	"states_project_id_triage_key iuw",
	"states_updated_by_id_fkey f n",
	"states_workspace_id_fkey f c",
	"states_workspace_id_idx i",
}
````

- [ ] **Step 5: `project` 模块：操作名、端口、连带、存储**

`server/internal/modules/project/domain/actions.go`（新文件，13 行）：

````file server/internal/modules/project/domain/actions.go
// Package domain holds the project module's rules (M3 design 6.3): pure
// functions and values.
package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// Actions lists the module's actions: the keys of its rows in the access
// module's rule table (M3 design 3.4). bootstrap's test holds the union of
// every module's Actions equal to the rule table's keys. Each operation adds
// its action here with its row.
func Actions() []shared.Action {
	return nil
}
````

`server/internal/modules/project/app/ports.go`（新文件，21 行）：

````file server/internal/modules/project/app/ports.go
// Package app holds the project module's use cases (M3 design 6.3) and the
// ports they need: small repository interfaces per use case, the clock, and
// the other modules' adapters as bootstrap converts them.
package app

import (
	"context"
	"time"
	"uuid"
)

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

`server/internal/modules/project/app/cascade.go`（新文件，48 行）：

````file server/internal/modules/project/app/cascade.go
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
	projects WorkspaceProjectsDeleter
}

// NewCascade returns the cascade over projects.
func NewCascade(projects WorkspaceProjectsDeleter) *Cascade {
	return &Cascade{projects: projects}
}

// DeleteWorkspaceProjects soft-deletes the workspace's projects and the
// rows under them (M3 design 3.6): the last step of deleteWorkspace's
// cascade, under its FOR NO KEY UPDATE of the workspace, each step one
// statement (convention 5). Every step runs, in order, at by and now.
func (c *Cascade) DeleteWorkspaceProjects(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	for _, step := range c.deletion() {
		if err := step(ctx, workspaceID, by, now); err != nil {
			return err
		}
	}
	return nil
}

// deletion is what deleting a workspace deletes of the projects, in the
// global order of M3 design 3.6: the projects, their memberships, the
// members' display settings, the states. P7 adds the labels at the end.
func (c *Cascade) deletion() []func(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return []func(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error{
		c.projects.DeleteWorkspaceProjects,
		c.projects.DeleteWorkspaceProjectMembers,
		c.projects.DeleteWorkspaceProjectPreferences,
		c.projects.DeleteWorkspaceStates,
	}
}
````

`server/internal/modules/project/app/cascade_test.go`（新文件，77 行）：

````file server/internal/modules/project/app/cascade_test.go
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
)

// fakeDeleter is the projects' repository: it records each step with its
// arguments, and fails the step named in fail.
type fakeDeleter struct {
	calls []string
	fail  string
}

var errDisk = errors.New("disk full")

func (f *fakeDeleter) step(name string, workspaceID, by uuid.UUID, now time.Time) error {
	f.calls = append(f.calls, fmt.Sprintf("%s %s by %s at %s", name, workspaceID, by, now.Format(time.RFC3339Nano)))
	if name == f.fail {
		return errDisk
	}
	return nil
}

func (f *fakeDeleter) DeleteWorkspaceProjects(_ context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return f.step("DeleteWorkspaceProjects", workspaceID, by, now)
}

func (f *fakeDeleter) DeleteWorkspaceProjectMembers(_ context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return f.step("DeleteWorkspaceProjectMembers", workspaceID, by, now)
}

func (f *fakeDeleter) DeleteWorkspaceProjectPreferences(_ context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return f.step("DeleteWorkspaceProjectPreferences", workspaceID, by, now)
}

func (f *fakeDeleter) DeleteWorkspaceStates(_ context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return f.step("DeleteWorkspaceStates", workspaceID, by, now)
}

// DeleteWorkspaceProjects runs the four steps in the order of M3 design
// 3.6, each with the caller's workspace, account and moment; a failing step
// comes back as itself and the steps after it do not run.
func TestDeleteWorkspaceProjects(t *testing.T) {
	workspace, by := uuid.NewV7(), uuid.NewV7()
	now := time.Date(2026, 10, 1, 10, 0, 0, 123456000, time.UTC)
	var steps []string
	for _, name := range []string{"DeleteWorkspaceProjects", "DeleteWorkspaceProjectMembers", "DeleteWorkspaceProjectPreferences",
		"DeleteWorkspaceStates"} {
		steps = append(steps, fmt.Sprintf("%s %s by %s at %s", name, workspace, by, now.Format(time.RFC3339Nano)))
	}
	tests := []struct {
		fail    string
		wantErr error
		want    []string
	}{
		{"", nil, steps},
		{"DeleteWorkspaceProjects", errDisk, steps[:1]},
		{"DeleteWorkspaceProjectMembers", errDisk, steps[:2]},
		{"DeleteWorkspaceProjectPreferences", errDisk, steps[:3]},
		{"DeleteWorkspaceStates", errDisk, steps},
	}
	for _, tt := range tests {
		f := &fakeDeleter{fail: tt.fail}
		err := app.NewCascade(f).DeleteWorkspaceProjects(context.Background(), workspace, by, now)
		if !errors.Is(err, tt.wantErr) || (tt.wantErr == nil) != (err == nil) || !slices.Equal(f.calls, tt.want) {
			t.Errorf("failing %q: DeleteWorkspaceProjects() = %v, the steps\n%q\nwant %v,\n%q", tt.fail, err, f.calls, tt.wantErr, tt.want)
		}
	}
}
````

`server/internal/modules/project/adapter/postgres/store.go`（新文件，28 行）：

````file server/internal/modules/project/adapter/postgres/store.go
// Package postgresadapter is the project module's repository adapter: sqlc
// queries (queries/, generated into gen/) over the transaction that the
// context carries, or the pool.
package postgresadapter

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// Store implements the project repository ports of app.
type Store struct {
	pool *pgxpool.Pool
}

// New returns the store over pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// queries runs in the context's transaction when there is one.
func (s *Store) queries(ctx context.Context) *gen.Queries {
	return gen.New(postgres.DB(ctx, s.pool))
}
````

`server/internal/modules/project/adapter/postgres/cascade.go`（新文件，56 行）：

````file server/internal/modules/project/adapter/postgres/cascade.go
package postgresadapter

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres/gen"
)

// The steps of deleting a workspace's projects (app.WorkspaceProjectsDeleter):
// each soft-deletes the workspace's undeleted rows of one table, at now, by
// the account by, in one statement.

// DeleteWorkspaceProjects soft-deletes the workspace's projects, archived
// ones too.
func (s *Store) DeleteWorkspaceProjects(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	err := s.queries(ctx).DeleteWorkspaceProjects(ctx, gen.DeleteWorkspaceProjectsParams{WorkspaceID: workspaceID, DeletedBy: by, Now: now})
	if err != nil {
		return fmt.Errorf("delete the workspace's projects: %w", err)
	}
	return nil
}

// DeleteWorkspaceProjectMembers soft-deletes the memberships of the
// workspace's projects, active or not.
func (s *Store) DeleteWorkspaceProjectMembers(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	err := s.queries(ctx).DeleteWorkspaceProjectMembers(ctx, gen.DeleteWorkspaceProjectMembersParams{WorkspaceID: workspaceID, DeletedBy: by, Now: now})
	if err != nil {
		return fmt.Errorf("delete the workspace's project members: %w", err)
	}
	return nil
}

// DeleteWorkspaceProjectPreferences soft-deletes the display settings in
// the workspace's projects.
func (s *Store) DeleteWorkspaceProjectPreferences(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	err := s.queries(ctx).DeleteWorkspaceProjectPreferences(ctx, gen.DeleteWorkspaceProjectPreferencesParams{
		WorkspaceID: workspaceID, DeletedBy: by, Now: now,
	})
	if err != nil {
		return fmt.Errorf("delete the workspace's project preferences: %w", err)
	}
	return nil
}

// DeleteWorkspaceStates soft-deletes the states of the workspace's
// projects, the triage states too.
func (s *Store) DeleteWorkspaceStates(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	err := s.queries(ctx).DeleteWorkspaceStates(ctx, gen.DeleteWorkspaceStatesParams{WorkspaceID: workspaceID, DeletedBy: by, Now: now})
	if err != nil {
		return fmt.Errorf("delete the workspace's states: %w", err)
	}
	return nil
}
````

`server/internal/modules/project/adapter/postgres/store_test.go`（新文件，55 行）：

````file server/internal/modules/project/adapter/postgres/store_test.go
package postgresadapter_test

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// now is the fixed clock's time: whole microseconds, as timestamptz stores
// them, so the audit columns read back equal to it (M2 design 3.13).
var now = clocktest.At(time.Date(2026, 10, 1, 10, 0, 0, 123456789, time.UTC)).Now()

func newStore(t *testing.T) (*postgresadapter.Store, *pgxpool.Pool) {
	t.Helper()
	pool, err := postgres.NewPool(context.Background(), config.DatabaseConfig{URL: pgtest.NewDatabase(t), MaxConns: 4})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return postgresadapter.New(pool), pool
}

func exec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatal(err)
	}
}

// newAccount inserts the users row that the project tables reference.
// identity owns the table: the test writes it directly, as a fixture.
func newAccount(t *testing.T, pool *pgxpool.Pool, email string) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	exec(t, pool, "INSERT INTO users (id, email, password, display_name) VALUES ($1, $2, 'x', 'x')", id, email)
	return id
}

// newWorkspace inserts the workspaces row the project tables reference.
// workspace owns the table: the test writes it directly, as a fixture.
func newWorkspace(t *testing.T, pool *pgxpool.Pool, slug string) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	exec(t, pool, "INSERT INTO workspaces (id, name, slug) VALUES ($1, $2, $2)", id, slug)
	return id
}
````

`server/internal/modules/project/adapter/postgres/cascade_test.go`（新文件，166 行）：

````file server/internal/modules/project/adapter/postgres/cascade_test.go
package postgresadapter_test

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// deletion is a row's audit columns, as the tests read them.
type deletion struct {
	deletedAt *time.Time
	updatedAt time.Time
	updatedBy *uuid.UUID
}

// deletedAtBy reports whether d was deleted at when by by.
func (d deletion) deletedAtBy(when time.Time, by uuid.UUID) bool {
	return d.deletedAt != nil && d.deletedAt.Equal(when) && d.updatedAt.Equal(when) && d.updatedBy != nil && *d.updatedBy == by
}

// projectTables are the tables a workspace's deletion deletes the projects'
// rows of, in its order.
var projectTables = []string{"projects", "project_members", "project_user_properties", "states"}

// deletions reads the audit columns of the workspace's rows of each project
// table, keyed by table then id.
func deletions(t *testing.T, pool *pgxpool.Pool, workspace uuid.UUID) map[string]map[uuid.UUID]deletion {
	t.Helper()
	out := map[string]map[uuid.UUID]deletion{}
	for _, table := range projectTables {
		rows, err := pool.Query(context.Background(),
			"SELECT id, deleted_at, updated_at, updated_by_id FROM "+table+" WHERE workspace_id = $1", workspace)
		if err != nil {
			t.Fatal(err)
		}
		out[table] = map[uuid.UUID]deletion{}
		for rows.Next() {
			var id uuid.UUID
			var d deletion
			if err := rows.Scan(&id, &d.deletedAt, &d.updatedAt, &d.updatedBy); err != nil {
				t.Fatal(err)
			}
			out[table][id] = d
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

// seedProjects writes two projects of workspace (seedProject), the second
// archived and bob's membership of it inactive.
func seedProjects(t *testing.T, pool *pgxpool.Pool, workspace, alice, bob uuid.UUID) {
	t.Helper()
	seedProject(t, pool, workspace, "Web", alice, bob)
	ops := seedProject(t, pool, workspace, "Ops", alice, bob)
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", ops, now)
	exec(t, pool, "UPDATE project_members SET is_active = false WHERE project_id = $1 AND member_id = $2", ops, bob)
}

// The four steps, in one transaction, soft-delete the workspace's
// projects, archived ones too, every membership of them, active or not,
// every member's display settings in them and every state, the triage ones
// too, at the same moment and by the same account; a row deleted before
// keeps its time, and another workspace keeps everything. Running the
// steps again changes nothing.
func TestDeletingAWorkspaceSoftDeletesItsProjects(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	seedProjects(t, pool, acme, alice, bob)
	seedProjects(t, pool, beta, alice, bob)
	// A project of acme deleted before, with the rows under it.
	earlier, later := now.Add(-time.Hour), now.Add(time.Hour)
	old := seedProject(t, pool, acme, "Old", alice, bob)
	for _, table := range projectTables {
		column := map[string]string{"projects": "id"}[table]
		if column == "" {
			column = "project_id"
		}
		exec(t, pool, "UPDATE "+table+" SET deleted_at = $2 WHERE "+column+" = $1", old, earlier)
	}
	steps := []func(ctx context.Context, id, by uuid.UUID, now time.Time) error{
		s.DeleteWorkspaceProjects, s.DeleteWorkspaceProjectMembers, s.DeleteWorkspaceProjectPreferences, s.DeleteWorkspaceStates,
	}
	run := func(ctx context.Context, by uuid.UUID, at time.Time) error {
		for _, step := range steps {
			if err := step(ctx, acme, by, at); err != nil {
				return err
			}
		}
		return nil
	}

	if err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		return run(ctx, bob, later)
	}); err != nil {
		t.Fatal(err)
	}
	// Once more, by alice an hour later: nothing is left undeleted.
	if err := run(context.Background(), alice, later.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	perProject := map[string]int{"projects": 1, "project_members": 2, "project_user_properties": 2, "states": 2}
	for table, rows := range deletions(t, pool, acme) {
		deleted, before := 0, 0
		for id, d := range rows {
			switch {
			case d.deletedAtBy(later, bob):
				deleted++
			case d.deletedAt != nil && d.deletedAt.Equal(earlier) && d.updatedAt.Equal(now):
				before++
			default:
				t.Errorf("acme's %s %s: %+v, want deleted at %v by bob, or at %v as before", table, id, d, later, earlier)
			}
		}
		if deleted != 2*perProject[table] || before != perProject[table] {
			t.Errorf("acme's %s: %d deleted by bob, %d deleted before; want %d, %d", table, deleted, before, 2*perProject[table], perProject[table])
		}
	}
	var active []bool
	if err := pool.QueryRow(context.Background(), `SELECT array_agg(is_active ORDER BY is_active) FROM project_members
		WHERE workspace_id = $1 AND deleted_at = $2`, acme, later).Scan(&active); err != nil || len(active) != 4 || active[0] || !active[1] {
		t.Errorf("is_active of acme's deleted memberships: %v, %v; want one false, three true, as they were", active, err)
	}
	for table, rows := range deletions(t, pool, beta) {
		if len(rows) != 2*perProject[table] {
			t.Errorf("beta's %s: %d rows, want %d", table, len(rows), 2*perProject[table])
		}
		for id, d := range rows {
			if d.deletedAt != nil || !d.updatedAt.Equal(now) {
				t.Errorf("beta's %s %s: %+v, want it untouched", table, id, d)
			}
		}
	}
}
````

`server/internal/modules/project/module.go`（新文件，58 行）：

````file server/internal/modules/project/module.go
// Package project is the projects module (M3 design 3.3, 6.3): projects,
// their members, their states and each member's display settings. It
// carries out the workspace module's cascades on the projects
// (ProjectCascade).
package project

import (
	"context"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Cascade is what the workspace module's writes ask of the projects:
// workspace's ProjectCascade port (M3 design 3.3). Each method runs in the
// caller's transaction, which ctx carries, writes by and now into the rows
// it changes, and returns a failure as itself, so the caller's whole write
// rolls back.
type Cascade interface {
	// DeleteWorkspaceProjects soft-deletes the workspace's projects and the
	// rows under them: deleteWorkspace's last step.
	DeleteWorkspaceProjects(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error
}

// Deps are what bootstrap gives the module (M3 design 6.6, step 4).
type Deps struct {
	Pool *pgxpool.Pool
}

// Module is the wired project module.
type Module struct {
	cascade *app.Cascade
}

// New wires the module from d; nothing is registered or injected after it
// (M3 design 6.6). bootstrap builds it before workspace, which takes its
// Cascade.
func New(d Deps) *Module {
	store := postgresadapter.New(d.Pool)
	return &Module{cascade: app.NewCascade(store)}
}

// Cascade is the module's implementation of workspace's ProjectCascade.
func (m *Module) Cascade() Cascade {
	return m.cascade
}

// Actions lists the module's actions: bootstrap's test holds the union of
// every module's actions equal to access's rule table (M3 design 3.4).
func Actions() []shared.Action {
	return domain.Actions()
}
````

- [ ] **Step 6: `workspace` 一侧：端口和 `cascade()` 的最后一步**

`server/internal/modules/workspace/app/ports.go`（修改，1 处）：

````old server/internal/modules/workspace/app/ports.go
}

// PreferencesRow is a change of an account's display settings in a
````
````new server/internal/modules/workspace/app/ports.go
}

// ProjectCascade is what the workspace's writes ask of the projects (M3
// design 3.3): the project module implements it (project.New's Cascade).
// Each method runs in the transaction ctx carries, writes by and now into
// the rows it changes, and returns a failure as itself, so the caller's
// whole write rolls back.
type ProjectCascade interface {
	// DeleteWorkspaceProjects soft-deletes the workspace's projects and the
	// rows under them, at the moment and by the account of the workspace's
	// deletion: its last step.
	DeleteWorkspaceProjects(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error
}

// PreferencesRow is a change of an account's display settings in a
````

`server/internal/modules/workspace/app/delete_workspace.go`（修改，4 处）：

````old server/internal/modules/workspace/app/delete_workspace.go
	workspaces WorkspaceDeleter
````
````new server/internal/modules/workspace/app/delete_workspace.go
	workspaces WorkspaceDeleter
	projects   ProjectCascade
````

````old server/internal/modules/workspace/app/delete_workspace.go
func NewDeleteWorkspace(workspaces WorkspaceDeleter, auth shared.Authorizer, tx shared.TxManager, clock Clock, logger *slog.Logger) *DeleteWorkspace {
	return &DeleteWorkspace{workspaces: workspaces, auth: auth, tx: tx, clock: clock, logger: logger}
````
````new server/internal/modules/workspace/app/delete_workspace.go
func NewDeleteWorkspace(workspaces WorkspaceDeleter, projects ProjectCascade, auth shared.Authorizer, tx shared.TxManager, clock Clock,
	logger *slog.Logger) *DeleteWorkspace {
	return &DeleteWorkspace{workspaces: workspaces, projects: projects, auth: auth, tx: tx, clock: clock, logger: logger}
````

````old server/internal/modules/workspace/app/delete_workspace.go
// statement at the same moment. P4 adds the projects at the end, through
// ProjectCascade (M3 design 3.3).
````
````new server/internal/modules/workspace/app/delete_workspace.go
// statement at the same moment, and last the projects and the rows under
// them, through ProjectCascade (M3 design 3.3).
````

````old server/internal/modules/workspace/app/delete_workspace.go
		u.workspaces.DeleteWorkspacePreferences,
````
````new server/internal/modules/workspace/app/delete_workspace.go
		u.workspaces.DeleteWorkspacePreferences,
		u.projects.DeleteWorkspaceProjects,
````

`server/internal/modules/workspace/app/fakes_projects_test.go`（新文件，24 行）：

````file server/internal/modules/workspace/app/fakes_projects_test.go
package app_test

import (
	"context"
	"fmt"
	"time"
	"uuid"
)

// fakeProjects is the project module's cascade (app.ProjectCascade): it
// logs each call with its arguments, and fails a method with the error set
// for it, wrapped as the module wraps it.
type fakeProjects struct {
	log  *callLog
	errs map[string]error // by method, e.g. "DeleteWorkspaceProjects"
}

func (f *fakeProjects) DeleteWorkspaceProjects(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "DeleteWorkspaceProjects %s by %s at %s", workspaceID, by, now.Format(time.RFC3339Nano))
	if err := f.errs["DeleteWorkspaceProjects"]; err != nil {
		return fmt.Errorf("delete the workspace's projects: %w", err)
	}
	return nil
}
````

`server/internal/modules/workspace/app/delete_workspace_test.go`（修改，7 处）：

````old server/internal/modules/workspace/app/delete_workspace_test.go
	workspaces *fakeWorkspaces
````
````new server/internal/modules/workspace/app/delete_workspace_test.go
	workspaces *fakeWorkspaces
	projects   *fakeProjects
````

````old server/internal/modules/workspace/app/delete_workspace_test.go
		auth: &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{
````
````new server/internal/modules/workspace/app/delete_workspace_test.go
		projects: &fakeProjects{log: log}, auth: &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{
````

````old server/internal/modules/workspace/app/delete_workspace_test.go
	return app.NewDeleteWorkspace(f.workspaces, f.auth, f.tx, ticking{clocktest.At(now)}, slog.New(slog.NewTextHandler(f.logs, nil))), f
````
````new server/internal/modules/workspace/app/delete_workspace_test.go
	return app.NewDeleteWorkspace(f.workspaces, f.projects, f.auth, f.tx, ticking{clocktest.At(now)}, slog.New(slog.NewTextHandler(f.logs, nil))), f
````

````old server/internal/modules/workspace/app/delete_workspace_test.go
// cascadeCalls are the deletion's steps on w by user, in order.
````
````new server/internal/modules/workspace/app/delete_workspace_test.go
// cascadeCalls are the deletion's steps on w by user, in order: the
// workspace's own, then the projects', through the project module.
````

````old server/internal/modules/workspace/app/delete_workspace_test.go
	for _, step := range []string{"DeleteWorkspace", "DeleteWorkspaceInvitations", "DeleteWorkspaceMembers", "DeleteWorkspacePreferences"} {
````
````new server/internal/modules/workspace/app/delete_workspace_test.go
	for _, step := range []string{"DeleteWorkspace", "DeleteWorkspaceInvitations", "DeleteWorkspaceMembers", "DeleteWorkspacePreferences",
		"DeleteWorkspaceProjects"} {
````

````old server/internal/modules/workspace/app/delete_workspace_test.go
// settings, by the caller at one moment, in one transaction (M3 design
// 3.6), and logs it. Two callers, two workspaces.
````
````new server/internal/modules/workspace/app/delete_workspace_test.go
// settings, and last its projects through the project module, by the
// caller at one moment, in one transaction (M3 design 3.6), and logs it.
// Two callers, two workspaces.
````

````old server/internal/modules/workspace/app/delete_workspace_test.go
				f.workspaces.deleteErrs = map[string]error{"DeleteWorkspacePreferences": failure}
			},
````
````new server/internal/modules/workspace/app/delete_workspace_test.go
				f.workspaces.deleteErrs = map[string]error{"DeleteWorkspacePreferences": failure}
			},
			failure, append(slices.Clone(decided), steps[:4]...)},
		{"the projects failed", alice, "acme",
			func(f *deleteFixture) { f.projects.errs = map[string]error{"DeleteWorkspaceProjects": failure} },
````

`server/internal/modules/workspace/app/clock_test.go`（修改，1 处）：

````old server/internal/modules/workspace/app/clock_test.go
			err := app.NewDeleteWorkspace(f.workspaces, f.auth, f.tx, clockAt{now, f.log}, slog.New(slog.DiscardHandler)).
````
````new server/internal/modules/workspace/app/clock_test.go
			err := app.NewDeleteWorkspace(f.workspaces, f.projects, f.auth, f.tx, clockAt{now, f.log}, slog.New(slog.DiscardHandler)).
````

`server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go`（修改，3 处）：

````old server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
}

// allowAll allows every action, as the workspace's admin.
````
````new server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
}

// noProjects is the project module's cascade, the last step, which the
// failure comes before.
type noProjects struct{}

func (noProjects) DeleteWorkspaceProjects(context.Context, uuid.UUID, uuid.UUID, time.Time) error {
	return errors.New("the projects' step ran after the failed one")
}

// allowAll allows every action, as the workspace's admin.
````

````old server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
// The use case's cascade is one transaction on the database: when its last
// step fails, the workspace, its invitations and its members are not
// deleted either.
````
````new server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
// The use case's cascade is one transaction on the database: when its
// last step of the workspace's own fails, the workspace, its invitations
// and its members are not deleted either.
````

````old server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
	uc := app.NewDeleteWorkspace(failingSettings{s}, allowAll{}, postgres.NewTxManager(pool, 2*time.Second), clocktest.At(now),
````
````new server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
	uc := app.NewDeleteWorkspace(failingSettings{s}, noProjects{}, allowAll{}, postgres.NewTxManager(pool, 2*time.Second), clocktest.At(now),
````

`server/internal/modules/workspace/module.go`（修改，2 处）：

````old server/internal/modules/workspace/module.go
	CallerLock app.CallerLock
````
````new server/internal/modules/workspace/module.go
	CallerLock app.CallerLock
	// Projects is the project module's Cascade (project.New).
	Projects app.ProjectCascade
````

````old server/internal/modules/workspace/module.go
		DeleteWorkspace:   app.NewDeleteWorkspace(store, d.Authorizer, d.Tx, d.Clock, d.Logger),
````
````new server/internal/modules/workspace/module.go
		DeleteWorkspace:   app.NewDeleteWorkspace(store, d.Projects, d.Authorizer, d.Tx, d.Clock, d.Logger),
````

- [ ] **Step 7: 组合根和组合出的测试**

`server/internal/bootstrap/app.go`（修改，3 处）：

````old server/internal/bootstrap/app.go
	"github.com/open-nerve/NerveProject/server/internal/modules/instance"
````
````new server/internal/bootstrap/app.go
	"github.com/open-nerve/NerveProject/server/internal/modules/instance"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
````

````old server/internal/bootstrap/app.go
	authorizer := access.New(access.Deps{WorkspaceRoles: workspacePorts.WorkspaceRoles})
````
````new server/internal/bootstrap/app.go
	authorizer := access.New(access.Deps{WorkspaceRoles: workspacePorts.WorkspaceRoles})
	// project before workspace: workspace's writes take its cascade.
	proj := project.New(project.Deps{Pool: pool})
````

````old server/internal/bootstrap/app.go
		CallerLock:      identityPorts.CredentialLock,
````
````new server/internal/bootstrap/app.go
		CallerLock:      identityPorts.CredentialLock,
		Projects:        proj.Cascade(),
````

`server/internal/bootstrap/actions_test.go`（修改，2 处）：

````old server/internal/bootstrap/actions_test.go
	"github.com/open-nerve/NerveProject/server/internal/modules/access"
````
````new server/internal/bootstrap/actions_test.go
	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
````

````old server/internal/bootstrap/actions_test.go
		"workspace": workspace.Actions(),
````
````new server/internal/bootstrap/actions_test.go
		"workspace": workspace.Actions(),
		"project":   project.Actions(),
````

`server/internal/bootstrap/workspace_deletion_test.go`（修改，3 处）：

````old server/internal/bootstrap/workspace_deletion_test.go
// the cascade deletes it (P4 the projects' tables through ProjectCascade).
````
````new server/internal/bootstrap/workspace_deletion_test.go
// the cascade deletes it: the projects' tables through ProjectCascade.
````

````old server/internal/bootstrap/workspace_deletion_test.go
// settings, through the API. A phase that adds a table under workspaces
// seeds a row of it here. It returns the workspace's id.
````
````new server/internal/bootstrap/workspace_deletion_test.go
// settings, through the API; a project with the member's membership, his
// display settings in it and a state (seedProject). A phase that adds a
// table under workspaces seeds a row of it here. It returns the workspace's
// id.
````

````old server/internal/bootstrap/workspace_deletion_test.go
		t.Fatalf("the settings in %s = %d %s, want 200", slug, status, body)
	}
	return id
````
````new server/internal/bootstrap/workspace_deletion_test.go
		t.Fatalf("the settings in %s = %d %s, want 200", slug, status, body)
	}
	seedProject(t, pool, id, admin, member)
	return id
}

// seedProject writes a project of the workspace id, created by admin,
// with member's membership, his display settings in it and one state,
// directly: the seed does not depend on which of the project module's
// writes exist, nor on what they write besides.
func seedProject(t *testing.T, pool *pgxpool.Pool, id, admin, member uuid.UUID) {
	t.Helper()
	ctx, project := context.Background(), uuid.NewV7()
	for _, stmt := range []struct {
		sql  string
		args []any
	}{
		{"INSERT INTO projects (id, workspace_id, name, identifier, created_by_id) VALUES ($1, $2, 'Web', 'WEB', $3)", []any{project, id, admin}},
		{"INSERT INTO project_members (id, workspace_id, project_id, member_id, role) VALUES ($1, $2, $3, $4, 15)",
			[]any{uuid.NewV7(), id, project, member}},
		{"INSERT INTO project_user_properties (id, workspace_id, project_id, user_id) VALUES ($1, $2, $3, $4)",
			[]any{uuid.NewV7(), id, project, member}},
		{`INSERT INTO states (id, workspace_id, project_id, name, color, "default") VALUES ($1, $2, $3, 'Backlog', '#60646C', true)`,
			[]any{uuid.NewV7(), id, project}},
	} {
		if _, err := pool.Exec(ctx, stmt.sql, stmt.args...); err != nil {
			t.Fatal(err)
		}
	}
}

// The projects' step runs in the deletion's transaction (M3 design 3.3,
// 9.3): when it fails on the wired app, deleteWorkspace answers the failure
// and no row changes, under either workspace, the projects' nor the
// workspace's own. The states table, renamed while the request runs, fails
// the last statement of the cascade.
func TestAFailedProjectsStepRollsTheDeletionBack(t *testing.T) {
	contract := apitest.Load(t)
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
		t.Helper()
		if _, err := pool.Exec(context.Background(), sql); err != nil {
			t.Fatal(err)
		}
	}
	exec("ALTER TABLE states RENAME TO states_away")
	status, body := call(t, contract, http.MethodDelete, base+"/api/v0/workspaces/deleted", admin, "")
	exec("ALTER TABLE states_away RENAME TO states")
	if status != http.StatusInternalServerError {
		t.Fatalf("deleting the workspace with the states' step failing = %d %s, want 500", status, body)
	}
	if after := rowsOf(); !slices.Equal(after, before) {
		t.Errorf("the rows after the failed deletion:\n%q\nwant\n%q", after, before)
	}
````

`server/internal/bootstrap/interleaving_answers_test.go`（修改，2 处）：

````old server/internal/bootstrap/interleaving_answers_test.go
	identitydomain "github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
````
````new server/internal/bootstrap/interleaving_answers_test.go
	identitydomain "github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
````

````old server/internal/bootstrap/interleaving_answers_test.go
	return workspaceapp.NewDeleteWorkspace(workspaces, access.New(access.Deps{WorkspaceRoles: workspace.Provide(r.pool).WorkspaceRoles}),
		r.tx(), clocktest.At(time.Now()), slog.New(slog.DiscardHandler)).Execute(shared.WithActor(ctx, shared.Actor{UserID: r.alice}), "acme")
````
````new server/internal/bootstrap/interleaving_answers_test.go
	return workspaceapp.NewDeleteWorkspace(workspaces, project.New(project.Deps{Pool: r.pool}).Cascade(),
		access.New(access.Deps{WorkspaceRoles: workspace.Provide(r.pool).WorkspaceRoles}), r.tx(), clocktest.At(time.Now()),
		slog.New(slog.DiscardHandler)).Execute(shared.WithActor(ctx, shared.Actor{UserID: r.alice}), "acme")
````

- [ ] **Step 8: 测试、lint、前端检查**

Run: `go -C server test -count=1 ./migrations/ ./internal/modules/project/... ./internal/modules/workspace/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt|TestAFailedProjectsStepRollsTheDeletionBack|TestTheGrantsFileCoversEveryRelationAndFunction|TestEveryModuleDeclaresItsActionsOrHasNone|TestEveryActionHasARuleAndEveryRuleAnAction' ./internal/bootstrap/`
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

- [ ] **Step 9: 提交**

```bash
git add api/modules/workspace.yaml deploy/runtime-grants.sql server/internal/bootstrap/actions_test.go server/internal/bootstrap/app.go server/internal/bootstrap/interleaving_answers_test.go server/internal/bootstrap/workspace_deletion_test.go server/internal/modules/project/adapter/postgres/cascade.go server/internal/modules/project/adapter/postgres/cascade_test.go server/internal/modules/project/adapter/postgres/queries/cascade.sql server/internal/modules/project/adapter/postgres/store.go server/internal/modules/project/adapter/postgres/store_test.go server/internal/modules/project/app/cascade.go server/internal/modules/project/app/cascade_test.go server/internal/modules/project/app/ports.go server/internal/modules/project/domain/actions.go server/internal/modules/project/module.go server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go server/internal/modules/workspace/app/clock_test.go server/internal/modules/workspace/app/delete_workspace.go server/internal/modules/workspace/app/delete_workspace_test.go server/internal/modules/workspace/app/fakes_projects_test.go server/internal/modules/workspace/app/ports.go server/internal/modules/workspace/module.go server/migrations/project_schema_test.go server/migrations/schema_test.go server/migrations/sql/00010_project_projects.sql server/migrations/sql/00011_project_project_members.sql server/migrations/sql/00012_project_project_user_properties.sql server/migrations/sql/00013_project_states.sql server/sqlc.yaml api/dist/openapi.yaml server/internal/modules/project/adapter/postgres/gen/cascade.sql.go server/internal/modules/project/adapter/postgres/gen/db.go server/internal/modules/project/adapter/postgres/gen/models.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P4a): the project tables; deleting a workspace deletes its projects

Migrations 00010-00013 keep Plane's projects, project_members,
project_user_properties and states as M3 design 4.6-4.9 rule them, with
the logo_props, identifier and name CHECKs and the partial unique keys.
The project module starts with its cascade: deleteWorkspace's last step
calls ProjectCascade.DeleteWorkspaceProjects, which soft-deletes the
projects, their memberships, their display settings and their states in
the caller's transaction, at its moment and by its account. bootstrap
builds project.New before workspace.New; project joins the module list
with no action yet.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 |
|---|---|
| 四条删除语句各去掉 `workspace_id`；各去掉 `deleted_at IS NULL`（再删已删除的） | `TestDeletingAWorkspaceSoftDeletesItsProjects`（前者另有 `TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`；Task 14 起 W3 单独运行也失败） |
| 四条语句各不写 `updated_by_id`、`updated_at` | `TestDeletingAWorkspaceSoftDeletesItsProjects` |
| `deletion()` 少状态一步；顺序不是设计的；一步的失败被吞掉；各步用别的时刻 | `TestDeleteWorkspaceProjects`（第一个另有 `TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`） |
| `cascade()` 没有项目一步；项目一步在工作区行之前 | `TestDeleteWorkspaceLocksThenDecidesThenCascades`（第一个另有 `TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`） |
| 组合根给 `workspace` 一个什么都不删的连带 | `TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`、`TestAFailedProjectsStepRollsTheDeletionBack` |
| 项目的存储在调用者的事务之外执行 | `TestAFailedProjectsStepRollsTheDeletionBack` |
| `moduleActions` 不列 `project`（裁定 S3） | `TestEveryModuleDeclaresItsActionsOrHasNone`、`TestEveryActionHasARuleAndEveryRuleAnAction` |

**Done when:** 13 个迁移 up、down、再 up；四张表的名字和种类由测试核对；删除工作区在同一个事务、同一时刻连带项目和项目之下的三张表，组合的删除测试不加豁免而通过，项目一步失败时整个删除回滚。

---

### Task 3: 项目的领域；CHECK 的反例与部分唯一键

**Files:**
- Create: `server/internal/modules/project/domain/logo.go`、`server/internal/modules/project/domain/preferences.go`、`server/internal/modules/project/domain/preferences_test.go`、`server/internal/modules/project/domain/project.go`、`server/internal/modules/project/domain/project_test.go`、`server/internal/modules/project/domain/state.go`、`server/internal/modules/project/domain/state_test.go`
- Modify: `server/migrations/project_schema_test.go`

**Interfaces:**
- Produces（spec 2.4，M3 设计 3.17–3.19、4.6）：`domain.Project`、`Network`（`NetworkPrivate = 0`、`NetworkPublic = 2`）、`NewProject`、`CheckNewProject(NewProject) (NewProject, error)`、`CanLead(shared.Role) bool`（按集合：管理员或成员）、`Identifier(s)`；`LogoProps`、`Emoji`、`Icon`；`StateGroup`、`NewState`、`DefaultStates()`；`DefaultSortOrder = 65535`、`SortOrderFirst(lowest *float64) float64`。
- 使用者：Task 4 的存储，Task 7 的用例。

**Tests:**
- `domain/project_test.go`：`TestCheckNewProjectAcceptsValidProjects`（标识转成大写、不给网络时公开、各边界值）；`TestCheckNewProjectReportsEveryField`（每个字段的每种问题，一个 422 列出全部；名称全是空白、256 个字符、每个禁用字符、NUL；标识的空串、11 个字符、`-`、别的字母；说明的 NUL；网络 1；未知的时区；图标的 `in_use` 和五个文本的 NUL）；`TestCanLead`（管理员、成员可以，访客和三种以外的角色不可以）。
- `domain/state_test.go`：`TestDefaultStates`；`domain/preferences_test.go`：`TestSortOrderFirst`。
- `server/migrations/project_schema_test.go`：`TestProjectChecksRejectCounterexamples`（`logo_props` 的四个合法值和十五个反例：M3 设计 4.6、9.3 的十个，含 Codex S5 的两个，另有那十个没有试到的每个条件一个：表情的键、`url`，图标是对象、`name`、`background_color`；名称、标识、网络、`archive_in`、`last_issue_sequence`、项目角色、显示设置、状态的反例和边界）；`TestProjectUniqueKeysHoldAmongUndeletedRowsOnly`（五个部分唯一键：第二个未删除的行被拒绝，软删除第一行之后可以再用；别的工作区、项目、账户有自己的键）。

- [ ] **Step 1: 领域**

`server/internal/modules/project/domain/project.go`（新文件，158 行）：

````file server/internal/modules/project/domain/project.go
package domain

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Network is who of the workspace sees a project besides its members (M3
// design 3.4, 4.6).
type Network int

// The two networks, the values of projects.network.
const (
	// NetworkPrivate: the project's members and the workspace's admins.
	NetworkPrivate Network = 0
	// NetworkPublic: also the workspace's members.
	NetworkPublic Network = 2
)

// NewProject is what the caller asks for when creating a project (M3 design
// 5.1). A nil Network asks for a public project, the column's default; a
// nil Timezone asks for the workspace's (3.19).
type NewProject struct {
	Name        string
	Identifier  string
	Description string
	Network     *Network
	LeadID      *uuid.UUID
	LogoProps   LogoProps
	Timezone    *string
}

// The lengths of projects.name, varchar(255), and of an identifier, in
// characters (M3 design 3.19).
const (
	maxNameLength       = 255
	maxIdentifierLength = 10
)

// forbiddenNameCharacters may not occur in a name: Plane's
// FORBIDDEN_IDENTIFIER_CHARS_PATTERN (db/models/project.py:143), which the
// column's CHECK holds too (M3 design 3.19, 4.6).
const forbiddenNameCharacters = `&+,:;$^}{*=?@#|'<>.()%!-`

// identifierPattern is an identifier's spelling, in upper case: the web
// form's (core/components/project/create/common-attributes.tsx:96-107),
// which the column's CHECK holds too.
var identifierPattern = regexp.MustCompile(`^[A-Z0-9ÇŞĞİÖÜ]+$`)

// Identifier is the identifier that s stands for: s in upper case, as the
// web form sends it and the column stores it (M3 design 3.19).
func Identifier(s string) string {
	return strings.ToUpper(s)
}

// CheckNewProject checks p (M3 design 3.19) and returns it as it is stored:
// the identifier in upper case, the network given or public.
//   - a name of 1–255 characters, not blank, without NUL, which the
//     database cannot store, and without any of & + , : ; $ ^ } { * = ? @ #
//     | ' < > . ( ) % ! - (not_allowed);
//   - an identifier that is 1–10 of A-Z, 0-9 and ÇŞĞİÖÜ in upper case;
//   - a description without NUL; a network of 0 or 2; a time zone that
//     shared.ValidTimezone accepts; logo_props by checkLogoProps.
//
// Every problem is reported at once, as one 422 validation_failed. Whether
// the name or the identifier is taken is the database's to say; whether the
// lead may lead, the use case's, under its locks (M3 design 3.6).
func CheckNewProject(p NewProject) (NewProject, error) {
	p.Identifier = Identifier(p.Identifier)
	if p.Network == nil {
		public := NetworkPublic
		p.Network = &public
	}
	found := []*shared.FieldError{checkName(p.Name), checkIdentifier(p.Identifier), checkText("description", p.Description),
		checkNetwork(*p.Network), checkTimezone(p.Timezone)}
	if err := invalid(append(found, checkLogoProps(p.LogoProps)...)...); err != nil {
		return NewProject{}, err
	}
	return p, nil
}

// CanLead reports whether an account of workspace role role, an active
// member of the workspace, may lead a new project: an admin or a member,
// not a guest (M3 design 3.19). The set is named, not a bound.
func CanLead(role shared.Role) bool {
	return role == shared.RoleAdmin || role == shared.RoleMember
}

// invalid is the 422 of the problems found, in order, or nil when there is
// none.
func invalid(found ...*shared.FieldError) error {
	var fields []shared.FieldError
	for _, f := range found {
		if f != nil {
			fields = append(fields, *f)
		}
	}
	if len(fields) > 0 {
		return shared.Invalid(fields...)
	}
	return nil
}

func checkName(name string) *shared.FieldError {
	field := "name"
	switch {
	case strings.TrimSpace(name) == "":
		return &shared.FieldError{Field: field, Code: shared.FieldTooShort, Message: "must not be empty"}
	case utf8.RuneCountInString(name) > maxNameLength:
		return &shared.FieldError{Field: field, Code: shared.FieldTooLong, Message: fmt.Sprintf("must be at most %d characters", maxNameLength)}
	case strings.ContainsAny(name, forbiddenNameCharacters):
		return &shared.FieldError{Field: field, Code: shared.FieldNotAllowed,
			Message: "must not contain any of & + , : ; $ ^ } { * = ? @ # | ' < > . ( ) % ! -"}
	}
	return checkText(field, name)
}

func checkIdentifier(id string) *shared.FieldError {
	field := "identifier"
	switch {
	case id == "":
		return &shared.FieldError{Field: field, Code: shared.FieldTooShort, Message: "must not be empty"}
	case utf8.RuneCountInString(id) > maxIdentifierLength:
		return &shared.FieldError{Field: field, Code: shared.FieldTooLong,
			Message: fmt.Sprintf("must be at most %d characters", maxIdentifierLength)}
	case !identifierPattern.MatchString(id):
		return &shared.FieldError{Field: field, Code: shared.FieldInvalidFormat, Message: "may hold only A-Z, 0-9 and ÇŞĞİÖÜ"}
	}
	return nil
}

// checkText refuses NUL in the text of field, which Postgres cannot store
// in text nor in jsonb.
func checkText(field, s string) *shared.FieldError {
	if strings.ContainsRune(s, 0) {
		return &shared.FieldError{Field: field, Code: shared.FieldInvalidFormat, Message: "must not contain a NUL character"}
	}
	return nil
}

func checkNetwork(n Network) *shared.FieldError {
	if n != NetworkPrivate && n != NetworkPublic {
		return &shared.FieldError{Field: "network", Code: shared.FieldInvalidFormat, Message: "must be 0, private, or 2, public"}
	}
	return nil
}

func checkTimezone(zone *string) *shared.FieldError {
	if zone != nil && !shared.ValidTimezone(*zone) {
		return &shared.FieldError{Field: "timezone", Code: shared.FieldInvalidFormat, Message: "is not a known time zone"}
	}
	return nil
}
````

`server/internal/modules/project/domain/logo.go`（新文件，59 行）：

````file server/internal/modules/project/domain/logo.go
package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// LogoProps is a project's icon (M3 design 3.19, 5.2): a closed structure,
// every field optional; the zero value, {} as JSON, is no icon. It is the
// web app's TLogoProps (web/packages/types/src/common.ts:21-32), and
// projects.logo_props holds it as this JSON: its CHECK takes the same keys
// and value types (M3 design 4.6).
type LogoProps struct {
	InUse *string `json:"in_use,omitempty"`
	Emoji *Emoji  `json:"emoji,omitempty"`
	Icon  *Icon   `json:"icon,omitempty"`
}

// Emoji is an emoji icon: its value, or the address of a custom one, which
// the web app does not request (M3 design 8.5).
type Emoji struct {
	Value *string `json:"value,omitempty"`
	URL   *string `json:"url,omitempty"`
}

// Icon is a named icon and its colors.
type Icon struct {
	Name            *string `json:"name,omitempty"`
	Color           *string `json:"color,omitempty"`
	BackgroundColor *string `json:"background_color,omitempty"`
}

// The values of LogoProps.InUse: which of the two icons the project shows.
const (
	LogoEmoji = "emoji"
	LogoIcon  = "icon"
)

// checkLogoProps refuses an in_use other than emoji or icon, which the
// column's CHECK refuses too, and NUL in any of its texts, which jsonb
// cannot store. Its keys and their types are the contract's to hold.
func checkLogoProps(l LogoProps) []*shared.FieldError {
	var found []*shared.FieldError
	if l.InUse != nil && *l.InUse != LogoEmoji && *l.InUse != LogoIcon {
		found = append(found, &shared.FieldError{Field: "logo_props.in_use", Code: shared.FieldInvalidFormat, Message: "must be emoji or icon"})
	}
	texts := map[string]*string{}
	if l.Emoji != nil {
		texts["logo_props.emoji.value"], texts["logo_props.emoji.url"] = l.Emoji.Value, l.Emoji.URL
	}
	if l.Icon != nil {
		texts["logo_props.icon.name"], texts["logo_props.icon.color"] = l.Icon.Name, l.Icon.Color
		texts["logo_props.icon.background_color"] = l.Icon.BackgroundColor
	}
	for _, field := range []string{"logo_props.emoji.value", "logo_props.emoji.url", "logo_props.icon.name", "logo_props.icon.color",
		"logo_props.icon.background_color"} {
		if s := texts[field]; s != nil {
			found = append(found, checkText(field, *s))
		}
	}
	return found
}
````

`server/internal/modules/project/domain/state.go`（新文件，39 行）：

````file server/internal/modules/project/domain/state.go
package domain

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

// NewState is a state to create: its values (M3 design 3.17).
type NewState struct {
	Name     string
	Color    string
	Sequence float64
	Group    StateGroup
	Default  bool
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
````

`server/internal/modules/project/domain/preferences.go`（新文件，18 行）：

````file server/internal/modules/project/domain/preferences.go
package domain

// DefaultSortOrder is a project's place in a member's sidebar when he has
// no other place in the workspace's projects:
// project_user_properties.sort_order's default (M3 design 3.18).
const DefaultSortOrder = 65535

// SortOrderFirst is the place in a member's sidebar of a project he is made
// a member of by its creation: before the others, 10000 less than the least
// of his places in the workspace's projects, lowest, or DefaultSortOrder
// when he has none (Plane's ProjectMember.save, db/models/project.py:226-
// 239; M3 design 3.18).
func SortOrderFirst(lowest *float64) float64 {
	if lowest == nil {
		return DefaultSortOrder
	}
	return *lowest - 10000
}
````

`server/internal/modules/project/domain/project_test.go`（新文件，128 行）：

````file server/internal/modules/project/domain/project_test.go
package domain

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func ptr[T any](v T) *T { return &v }

// CheckNewProject accepts these and returns each as it is stored: the
// identifier in upper case, the network given or public.
func TestCheckNewProjectAcceptsValidProjects(t *testing.T) {
	tests := []struct {
		p          NewProject
		identifier string
		network    Network
	}{
		{NewProject{Name: "Web", Identifier: "WEB"}, "WEB", NetworkPublic},
		{NewProject{Name: "w", Identifier: "w"}, "W", NetworkPublic},
		{NewProject{Name: strings.Repeat("项", 255), Identifier: "abcdefghij"}, "ABCDEFGHIJ", NetworkPublic},
		{NewProject{Name: "研发 Web_2 [beta] \\ /", Identifier: "çşğiöü09", Network: ptr(NetworkPrivate), Timezone: ptr("Asia/Shanghai"),
			Description: "多行\n说明", LogoProps: LogoProps{InUse: ptr(LogoIcon), Icon: &Icon{Name: ptr("home")}}}, "ÇŞĞIÖÜ09", NetworkPrivate},
		{NewProject{Name: "Ops", Identifier: "İ1", Network: ptr(NetworkPublic), LogoProps: LogoProps{InUse: ptr(LogoEmoji)}}, "İ1", NetworkPublic},
	}
	for _, tt := range tests {
		got, err := CheckNewProject(tt.p)
		want := tt.p
		want.Identifier, want.Network = tt.identifier, &tt.network
		if err != nil || got.Identifier != want.Identifier || *got.Network != *want.Network || got.Name != want.Name ||
			got.Description != want.Description || got.Timezone != want.Timezone {
			t.Errorf("CheckNewProject(%+v) = %+v, %v; want %+v", tt.p, got, err, want)
		}
	}
}

func TestCheckNewProjectReportsEveryField(t *testing.T) {
	valid := NewProject{Name: "Web", Identifier: "WEB"}
	with := func(change func(*NewProject)) NewProject {
		p := valid
		change(&p)
		return p
	}
	name := func(n string) NewProject { return with(func(p *NewProject) { p.Name = n }) }
	identifier := func(s string) NewProject { return with(func(p *NewProject) { p.Identifier = s }) }
	logo := func(l LogoProps) NewProject { return with(func(p *NewProject) { p.LogoProps = l }) }
	field := func(f, code, message string) []shared.FieldError {
		return []shared.FieldError{{Field: f, Code: code, Message: message}}
	}
	const forbidden = "must not contain any of & + , : ; $ ^ } { * = ? @ # | ' < > . ( ) % ! -"
	const nul = "must not contain a NUL character"
	tests := []struct {
		name string
		p    NewProject
		want []shared.FieldError
	}{
		{"empty name", name(""), field("name", "too_short", "must not be empty")},
		{"blank name", name(" \t\n"), field("name", "too_short", "must not be empty")},
		{"name of 256 characters", name(strings.Repeat("项", 256)), field("name", "too_long", "must be at most 255 characters")},
		{"name with NUL", name("W\x00eb"), field("name", "invalid_format", nul)},
		{"empty identifier", identifier(""), field("identifier", "too_short", "must not be empty")},
		{"identifier of 11 characters", identifier("abcdefghijk"), field("identifier", "too_long", "must be at most 10 characters")},
		{"identifier with -", identifier("WEB-2"), field("identifier", "invalid_format", "may hold only A-Z, 0-9 and ÇŞĞİÖÜ")},
		{"identifier with a space", identifier("WEB 2"), field("identifier", "invalid_format", "may hold only A-Z, 0-9 and ÇŞĞİÖÜ")},
		{"identifier with _", identifier("WEB_2"), field("identifier", "invalid_format", "may hold only A-Z, 0-9 and ÇŞĞİÖÜ")},
		{"identifier with another letter", identifier("ÉQUIPE"), field("identifier", "invalid_format", "may hold only A-Z, 0-9 and ÇŞĞİÖÜ")},
		{"identifier with a trailing newline", identifier("WEB\n"), field("identifier", "invalid_format", "may hold only A-Z, 0-9 and ÇŞĞİÖÜ")},
		{"description with NUL", with(func(p *NewProject) { p.Description = "a\x00" }), field("description", "invalid_format", nul)},
		{"network 1", with(func(p *NewProject) { p.Network = ptr(Network(1)) }), field("network", "invalid_format", "must be 0, private, or 2, public")},
		{"network 3", with(func(p *NewProject) { p.Network = ptr(Network(3)) }), field("network", "invalid_format", "must be 0, private, or 2, public")},
		{"unknown time zone", with(func(p *NewProject) { p.Timezone = ptr("Mars/Olympus") }),
			field("timezone", "invalid_format", "is not a known time zone")},
		{"the host's zone", with(func(p *NewProject) { p.Timezone = ptr("Local") }), field("timezone", "invalid_format", "is not a known time zone")},
		{"empty time zone", with(func(p *NewProject) { p.Timezone = ptr("") }), field("timezone", "invalid_format", "is not a known time zone")},
		{"in_use other", logo(LogoProps{InUse: ptr("other")}), field("logo_props.in_use", "invalid_format", "must be emoji or icon")},
		{"in_use empty", logo(LogoProps{InUse: ptr("")}), field("logo_props.in_use", "invalid_format", "must be emoji or icon")},
		{"emoji value with NUL", logo(LogoProps{Emoji: &Emoji{Value: ptr("\x00")}}), field("logo_props.emoji.value", "invalid_format", nul)},
		{"emoji url with NUL", logo(LogoProps{Emoji: &Emoji{URL: ptr("a\x00")}}), field("logo_props.emoji.url", "invalid_format", nul)},
		{"icon name with NUL", logo(LogoProps{Icon: &Icon{Name: ptr("\x00")}}), field("logo_props.icon.name", "invalid_format", nul)},
		{"icon color with NUL", logo(LogoProps{Icon: &Icon{Color: ptr("\x00")}}), field("logo_props.icon.color", "invalid_format", nul)},
		{"icon background with NUL", logo(LogoProps{Icon: &Icon{BackgroundColor: ptr("\x00")}}),
			field("logo_props.icon.background_color", "invalid_format", nul)},
		{"all at once", NewProject{Name: "", Identifier: "web-2", Description: "\x00", Network: ptr(Network(1)), Timezone: ptr(""),
			LogoProps: LogoProps{InUse: ptr("x"), Emoji: &Emoji{Value: ptr("\x00"), URL: ptr("\x00")},
				Icon: &Icon{Name: ptr("\x00"), Color: ptr("\x00"), BackgroundColor: ptr("\x00")}}}, []shared.FieldError{
			{Field: "name", Code: "too_short", Message: "must not be empty"},
			{Field: "identifier", Code: "invalid_format", Message: "may hold only A-Z, 0-9 and ÇŞĞİÖÜ"},
			{Field: "description", Code: "invalid_format", Message: nul},
			{Field: "network", Code: "invalid_format", Message: "must be 0, private, or 2, public"},
			{Field: "timezone", Code: "invalid_format", Message: "is not a known time zone"},
			{Field: "logo_props.in_use", Code: "invalid_format", Message: "must be emoji or icon"},
			{Field: "logo_props.emoji.value", Code: "invalid_format", Message: nul},
			{Field: "logo_props.emoji.url", Code: "invalid_format", Message: nul},
			{Field: "logo_props.icon.name", Code: "invalid_format", Message: nul},
			{Field: "logo_props.icon.color", Code: "invalid_format", Message: nul},
			{Field: "logo_props.icon.background_color", Code: "invalid_format", Message: nul},
		}},
	}
	// Each of Plane's forbidden characters, alone in a name (M3 design 3.19).
	for _, c := range "&+,:;$^}{*=?@#|'<>.()%!-" {
		tests = append(tests, struct {
			name string
			p    NewProject
			want []shared.FieldError
		}{"name with " + string(c), name("Web" + string(c) + "2"), field("name", "not_allowed", forbidden)})
	}
	for _, tt := range tests {
		_, err := CheckNewProject(tt.p)
		var e *shared.Error
		if !errors.As(err, &e) || e.Code != shared.CodeValidationFailed || !slices.Equal(e.Fields, tt.want) {
			t.Errorf("%s: CheckNewProject() = %v, want the fields %+v", tt.name, err, tt.want)
		}
	}
}

// Only a workspace's admins and members may lead a new project, not its
// guests nor a role outside the three (M3 design 3.19).
func TestCanLead(t *testing.T) {
	for role, want := range map[shared.Role]bool{shared.RoleAdmin: true, shared.RoleMember: true, shared.RoleGuest: false, 0: false, 10: false,
		25: false} {
		if got := CanLead(role); got != want {
			t.Errorf("CanLead(%d) = %v, want %v", role, got, want)
		}
	}
}
````

`server/internal/modules/project/domain/state_test.go`（新文件，22 行）：

````file server/internal/modules/project/domain/state_test.go
package domain

import (
	"slices"
	"testing"
)

// The six default states are Plane's, in its order: one per group, Backlog
// the only default, Triage the only triage state (M3 design 3.17).
func TestDefaultStates(t *testing.T) {
	want := []NewState{
		{"Backlog", "#60646C", 15000, "backlog", true},
		{"Todo", "#60646C", 25000, "unstarted", false},
		{"In Progress", "#F59E0B", 35000, "started", false},
		{"Done", "#46A758", 45000, "completed", false},
		{"Cancelled", "#9AA4BC", 55000, "cancelled", false},
		{"Triage", "#4E5355", 65000, "triage", false},
	}
	if got := DefaultStates(); !slices.Equal(got, want) {
		t.Errorf("DefaultStates() =\n%+v\nwant\n%+v", got, want)
	}
}
````

`server/internal/modules/project/domain/preferences_test.go`（新文件，16 行）：

````file server/internal/modules/project/domain/preferences_test.go
package domain

import "testing"

// SortOrderFirst puts a new project before every other of the member's in
// the workspace, or at the default when he has none (M3 design 3.18).
func TestSortOrderFirst(t *testing.T) {
	for _, tt := range []struct {
		lowest *float64
		want   float64
	}{{nil, 65535}, {ptr(65535.0), 55535}, {ptr(-2.5), -10002.5}, {ptr(0.0), -10000}} {
		if got := SortOrderFirst(tt.lowest); got != tt.want {
			t.Errorf("SortOrderFirst(%v) = %v, want %v", tt.lowest, got, tt.want)
		}
	}
}
````

- [ ] **Step 2: CHECK 的反例、部分唯一键**

`server/migrations/project_schema_test.go`（修改，2 处）：

````old server/migrations/project_schema_test.go
package migrations_test
````
````new server/migrations/project_schema_test.go
package migrations_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)
````

````old server/migrations/project_schema_test.go
}

````
````new server/migrations/project_schema_test.go
}

// The project tables' CHECKs accept what the domain writes and reject what
// bypasses it (M3 design 3.17, 3.19, 4.6–4.9). projects_logo_props_check
// takes the four valid values and refuses fifteen counterexamples: the ten
// of 4.6, Codex S5's two among them, and one for each conjunct those ten
// leave untried (the emoji's keys and url; the icon an object, its name
// and background color).
func TestProjectChecksRejectCounterexamples(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t, pgtest.NewDatabase(t))
	const (
		user      = "'0199a2b4-0000-7000-8000-000000000001'"
		workspace = "'0199a2b4-0000-7000-8000-000000000002'"
		project   = "'0199a2b4-0000-7000-8000-000000000003'"
	)
	for _, stmt := range []string{
		"INSERT INTO users (id, email, password, display_name) VALUES (" + user + ", 'alice@corp.com', 'x', 'alice')",
		"INSERT INTO workspaces (id, name, slug) VALUES (" + workspace + ", 'Acme', 'acme')",
		"INSERT INTO projects (id, workspace_id, name, identifier) VALUES (" + project + ", " + workspace + ", 'Web', 'WEB')",
		"INSERT INTO project_members (id, workspace_id, project_id, member_id) VALUES (gen_random_uuid(), " + workspace + ", " + project + ", " + user + ")",
		"INSERT INTO project_user_properties (id, workspace_id, project_id, user_id) VALUES (gen_random_uuid(), " + workspace + ", " + project +
			", " + user + ")",
		"INSERT INTO states (id, workspace_id, project_id, name, color) VALUES (gen_random_uuid(), " + workspace + ", " + project + ", 'Backlog', '#60646C')",
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	// What the domain writes, and the edges of each rule, all accepted: the
	// four logo_props values of 4.6, a name with a backslash (the CHECK's
	// \| is the bar, not a backslash), the upper-case letters of the
	// identifier's set.
	for _, stmt := range []string{
		"UPDATE projects SET logo_props = '{}'",
		`UPDATE projects SET logo_props = '{"emoji": {"value": "128640"}}'`,
		`UPDATE projects SET logo_props = '{"icon": {"name": "home", "color": "#6d7b8a"}}'`,
		`UPDATE projects SET logo_props = '{"in_use": "icon", "emoji": {"value": "128640", "url": "https://example.com/e.png"}, ` +
			`"icon": {"name": "home", "color": "#6d7b8a", "background_color": "#ffffff"}}'`,
		`UPDATE projects SET name = E'研发 Web_2 [beta] \\ /'`,
		"UPDATE projects SET identifier = 'ÇŞĞİÖÜ0129'",
		"UPDATE projects SET identifier = 'A'",
		"UPDATE projects SET network = 0",
		"UPDATE projects SET archive_in = 12",
		"UPDATE project_members SET role = 20",
		"UPDATE project_members SET role = 15",
		`UPDATE project_user_properties SET preferences = '{"navigation": {"default_tab": "cycles", "hide_in_more_menu": ["intake"]}}'`,
		`UPDATE states SET "group" = 'triage'`,
		`UPDATE states SET "group" = 'cancelled'`,
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Errorf("%s: %v, want it accepted", stmt, err)
		}
	}
	prefs := func(json string) string { return "UPDATE project_user_properties SET preferences = '" + json + "'" }
	tests := []struct{ name, stmt, constraint string }{
		{"empty project name", "UPDATE projects SET name = ''", "projects_name_check"},
		{"lower-case identifier", "UPDATE projects SET identifier = 'web'", "projects_identifier_check"},
		{"empty identifier", "UPDATE projects SET identifier = ''", "projects_identifier_check"},
		{"identifier of 11 characters", "UPDATE projects SET identifier = 'ABCDEFGHIJK'", "projects_identifier_check"},
		{"identifier with -", "UPDATE projects SET identifier = 'WEB-2'", "projects_identifier_check"},
		{"identifier with a space", "UPDATE projects SET identifier = 'WEB 2'", "projects_identifier_check"},
		{"identifier with another letter", "UPDATE projects SET identifier = 'ÉQUIPE'", "projects_identifier_check"},
		{"network 1", "UPDATE projects SET network = 1", "projects_network_check"},
		{"network -1", "UPDATE projects SET network = -1", "projects_network_check"},
		{"archive_in 13", "UPDATE projects SET archive_in = 13", "projects_archive_in_check"},
		{"archive_in -1", "UPDATE projects SET archive_in = -1", "projects_archive_in_check"},
		{"negative last_issue_sequence", "UPDATE projects SET last_issue_sequence = -1", "projects_last_issue_sequence_check"},
		{"logo_props with an unknown key", `UPDATE projects SET logo_props = '{"unexpected": true}'`, "projects_logo_props_check"},
		{"logo_props' in_use a number, emoji an array", `UPDATE projects SET logo_props = '{"in_use": 17, "emoji": []}'`, "projects_logo_props_check"},
		{"logo_props' in_use other", `UPDATE projects SET logo_props = '{"in_use": "other"}'`, "projects_logo_props_check"},
		{"logo_props' emoji value a number", `UPDATE projects SET logo_props = '{"emoji": {"value": 1}}'`, "projects_logo_props_check"},
		{"logo_props' emoji a string", `UPDATE projects SET logo_props = '{"emoji": "x"}'`, "projects_logo_props_check"},
		{"logo_props' icon with an unknown key", `UPDATE projects SET logo_props = '{"icon": {"shape": "round"}}'`, "projects_logo_props_check"},
		{"logo_props' icon color null", `UPDATE projects SET logo_props = '{"icon": {"color": null}}'`, "projects_logo_props_check"},
		{"logo_props' emoji with an unknown key", `UPDATE projects SET logo_props = '{"emoji": {"shape": "x"}}'`, "projects_logo_props_check"},
		{"logo_props' emoji url a number", `UPDATE projects SET logo_props = '{"emoji": {"url": 1}}'`, "projects_logo_props_check"},
		{"logo_props' icon a string", `UPDATE projects SET logo_props = '{"icon": "x"}'`, "projects_logo_props_check"},
		{"logo_props' icon name a number", `UPDATE projects SET logo_props = '{"icon": {"name": 1}}'`, "projects_logo_props_check"},
		{"logo_props' icon background_color a number", `UPDATE projects SET logo_props = '{"icon": {"background_color": 1}}'`,
			"projects_logo_props_check"},
		{"logo_props an array", `UPDATE projects SET logo_props = '[]'`, "projects_logo_props_check"},
		{"logo_props a string", `UPDATE projects SET logo_props = '"x"'`, "projects_logo_props_check"},
		{"logo_props JSON null", `UPDATE projects SET logo_props = 'null'`, "projects_logo_props_check"},
		{"project role 10", "UPDATE project_members SET role = 10", "project_members_role_check"},
		{"project role 0", "UPDATE project_members SET role = 0", "project_members_role_check"},
		{"preferences an array", prefs(`[]`), "project_user_properties_preferences_check"},
		{"preferences a scalar", prefs(`5`), "project_user_properties_preferences_check"},
		{"preferences JSON null", prefs(`null`), "project_user_properties_preferences_check"},
		{"preferences without navigation", prefs(`{}`), "project_user_properties_preferences_check"},
		{"preferences with pages", prefs(`{"navigation": {"default_tab": "work_items", "hide_in_more_menu": []}, "pages": {}}`),
			"project_user_properties_preferences_check"},
		{"navigation an array", prefs(`{"navigation": []}`), "project_user_properties_preferences_check"},
		{"navigation without default_tab", prefs(`{"navigation": {"hide_in_more_menu": []}}`), "project_user_properties_preferences_check"},
		{"navigation without hide_in_more_menu", prefs(`{"navigation": {"default_tab": "work_items"}}`), "project_user_properties_preferences_check"},
		{"navigation with another key", prefs(`{"navigation": {"default_tab": "work_items", "hide_in_more_menu": [], "x": 1}}`),
			"project_user_properties_preferences_check"},
		{"default_tab a number", prefs(`{"navigation": {"default_tab": 1, "hide_in_more_menu": []}}`), "project_user_properties_preferences_check"},
		{"hide_in_more_menu a string", prefs(`{"navigation": {"default_tab": "work_items", "hide_in_more_menu": "cycles"}}`),
			"project_user_properties_preferences_check"},
		{"empty state name", "UPDATE states SET name = ''", "states_name_check"},
		{"unknown group", `UPDATE states SET "group" = 'other'`, "states_group_check"},
		{"upper-case group", `UPDATE states SET "group" = 'Backlog'`, "states_group_check"},
	}
	// Each of Plane's forbidden characters in a name (M3 design 3.19).
	for _, c := range "&+,:;$^}{*=?@#|'<>.()%!-" {
		name := strings.ReplaceAll("Web"+string(c)+"2", "'", "''")
		tests = append(tests, struct{ name, stmt, constraint string }{"name with " + string(c), "UPDATE projects SET name = '" + name + "'",
			"projects_name_check"})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, tt.stmt)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "23514" || pgErr.ConstraintName != tt.constraint {
				t.Errorf("%s = %v, want check_violation (23514) of %s", tt.stmt, err, tt.constraint)
			}
		})
	}
}

// The project tables' partial unique keys hold among undeleted rows only:
// a second undeleted row with the key is refused, and soft-deleting the
// first frees the key (M3 design 3.17, 3.19, 4.6–4.9). Another workspace,
// project or account holds keys of its own: a key short of a column
// refuses one of the seeds.
func TestProjectUniqueKeysHoldAmongUndeletedRowsOnly(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t, pgtest.NewDatabase(t))
	const (
		alice = "'0199a2b4-0000-7000-8000-000000000001'"
		bob   = "'0199a2b4-0000-7000-8000-000000000002'"
		acme  = "'0199a2b4-0000-7000-8000-000000000003'"
		beta  = "'0199a2b4-0000-7000-8000-000000000004'"
		web   = "'0199a2b4-0000-7000-8000-000000000005'"
		ops   = "'0199a2b4-0000-7000-8000-000000000006'"
		other = "'0199a2b4-0000-7000-8000-000000000007'"
	)
	project := func(id, workspace, name, identifier string) string {
		return "INSERT INTO projects (id, workspace_id, name, identifier) VALUES (" + id + ", " + workspace + ", '" + name + "', '" + identifier + "')"
	}
	member := func(table, column, project, user string) string {
		return "INSERT INTO " + table + " (id, workspace_id, project_id, " + column + ") VALUES (gen_random_uuid(), " + acme + ", " + project + ", " + user + ")"
	}
	state := func(project, name, group string, isDefault bool) string {
		return fmt.Sprintf(`INSERT INTO states (id, workspace_id, project_id, name, color, "group", "default") VALUES (gen_random_uuid(), %s, %s, '%s', '#60646C', '%s', %t)`,
			acme, project, name, group, isDefault)
	}
	for _, stmt := range []string{
		"INSERT INTO users (id, email, password, display_name) VALUES (" + alice + ", 'alice@corp.com', 'x', 'alice'), (" + bob + ", 'bob@corp.com', 'x', 'bob')",
		"INSERT INTO workspaces (id, name, slug) VALUES (" + acme + ", 'Acme', 'acme'), (" + beta + ", 'Beta', 'beta')",
		project(web, acme, "Web", "WEB"), project(ops, acme, "Ops", "OPS"),
		// Another workspace holds the same name and identifier.
		project(other, beta, "Web", "WEB"),
		// alice in web; bob in web and alice in ops hold keys of their own.
		member("project_members", "member_id", web, alice), member("project_members", "member_id", web, bob),
		member("project_members", "member_id", ops, alice),
		member("project_user_properties", "user_id", web, alice), member("project_user_properties", "user_id", web, bob),
		member("project_user_properties", "user_id", ops, alice),
		// web's default, triage and Todo; ops has its own.
		state(web, "Backlog", "backlog", true), state(web, "Triage", "triage", false), state(web, "Todo", "unstarted", false),
		state(ops, "Backlog", "backlog", true), state(ops, "Triage", "triage", false), state(ops, "Todo", "unstarted", false),
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	tests := []struct{ name, insert, softDelete, index string }{
		{"a project's identifier in a workspace", project("gen_random_uuid()", acme, "Web 2", "WEB"),
			"UPDATE projects SET deleted_at = now() WHERE identifier = 'WEB' AND workspace_id = " + acme, "projects_workspace_id_identifier_key"},
		{"a project's name in a workspace", project("gen_random_uuid()", acme, "Ops", "OPS2"),
			"UPDATE projects SET deleted_at = now() WHERE name = 'Ops' AND workspace_id = " + acme, "projects_workspace_id_name_key"},
		{"an account's membership of a project", member("project_members", "member_id", web, alice),
			"UPDATE project_members SET deleted_at = now()", "project_members_project_id_member_id_key"},
		{"an account's display settings in a project", member("project_user_properties", "user_id", web, alice),
			"UPDATE project_user_properties SET deleted_at = now()", "project_user_properties_project_id_user_id_key"},
		{"a state's name in a project", state(web, "Todo", "started", false),
			"UPDATE states SET deleted_at = now() WHERE name = 'Todo' AND project_id = " + web, "states_project_id_name_key"},
		{"a project's default state", state(web, "Other", "backlog", true),
			`UPDATE states SET deleted_at = now() WHERE "default" AND project_id = ` + web, "states_project_id_default_key"},
		{"a project's triage state", state(web, "Intake", "triage", false),
			`UPDATE states SET deleted_at = now() WHERE "group" = 'triage' AND project_id = ` + web, "states_project_id_triage_key"},
	}
	// Any case can run first: each soft-deletes rows of its own table only,
	// and none of those rows is another case's key.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, tt.insert)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "23505" || pgErr.ConstraintName != tt.index {
				t.Errorf("%s = %v, want unique_violation (23505) of %s", tt.insert, err, tt.index)
			}
			if _, err := pool.Exec(ctx, tt.softDelete); err != nil {
				t.Fatalf("%s: %v", tt.softDelete, err)
			}
			if _, err := pool.Exec(ctx, tt.insert); err != nil {
				t.Errorf("after %s, %s = %v; want the key free", tt.softDelete, tt.insert, err)
			}
		})
	}
}

````

- [ ] **Step 3: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/project/domain/ ./migrations/`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/modules/project/domain/logo.go server/internal/modules/project/domain/preferences.go server/internal/modules/project/domain/preferences_test.go server/internal/modules/project/domain/project.go server/internal/modules/project/domain/project_test.go server/internal/modules/project/domain/state.go server/internal/modules/project/domain/state_test.go server/migrations/project_schema_test.go
```
```bash
git commit -m "feat(M3/P4a): the project's rules; the tables' CHECKs refuse their counterexamples

CheckNewProject holds a new project to M3 design 3.19: a name without
Plane's forbidden characters, an identifier of 1-10 of A-Z, 0-9 and
ÇŞĞİÖÜ once upper-cased, a network of 0 or 2, a known time zone, no NUL
anywhere; every problem in one 422. CanLead is a set. The six default
states and a new project's place in a sidebar are Plane's. The CHECKs
take logo_props' four valid values and refuse fifteen counterexamples,
the ten of 4.6 among them and one for each conjunct those leave untried,
and each partial unique key holds among undeleted rows only.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 |
|---|---|
| 标识不转大写；不给网络时私密 | `TestCheckNewProjectAcceptsValidProjects` |
| 标识接受 `-`、11 个字符、空串；名称接受 `-`、全是空白、256 个字符、NUL；说明接受 NUL；网络接受 1；任何时区；任何 `in_use`；图标的背景色、表情的地址接受 NUL | `TestCheckNewProjectReportsEveryField` |
| `CanLead` 按大小；访客可以领导 | `TestCanLead` |
| 默认状态的颜色改了；Todo 也是默认 | `TestDefaultStates` |
| 新项目排在别的之前 1000；没有别的位置时 0 | `TestSortOrderFirst` |
| 名称的 CHECK 去掉禁用字符；禁止反斜杠；标识的 CHECK 接受 12 个；`logo_props` 的 `in_use` 任何字符串、`icon.color` 任何类型、外层任何键、任何 JSON；表情任何键、`emoji.url`、`icon.name`、`icon.background_color` 任何类型、图标不是对象 | `TestProjectChecksRejectCounterexamples` |
| 标识的唯一键也管已删除的；名称的唯一键不带工作区；默认状态的唯一键也管已删除的；状态名的唯一键不带项目 | `TestProjectUniqueKeysHoldAmongUndeletedRowsOnly`（第一个另有 `TestConstraintAndIndexNames`） |

**Done when:** 领域的每条规则由表驱动的测试核对，四张表的 CHECK（`projects_logo_props_check` 的每个条件）和五个部分唯一键由真实数据库上的反例核对。

---

### Task 4: `project` 的存储：插入、`GetProject`、`LowestSortOrder`

**Files:**
- Create: `server/internal/modules/project/adapter/postgres/projects.go`、`server/internal/modules/project/adapter/postgres/projects_test.go`、`server/internal/modules/project/adapter/postgres/queries/members.sql`、`server/internal/modules/project/adapter/postgres/queries/preferences.sql`、`server/internal/modules/project/adapter/postgres/queries/projects.sql`、`server/internal/modules/project/adapter/postgres/queries/states.sql`、`server/internal/modules/project/adapter/postgres/rows.go`、`server/internal/modules/project/adapter/postgres/rows_test.go`、`server/internal/modules/project/domain/errors.go`
- Modify: `server/internal/modules/project/adapter/postgres/store.go`、`server/internal/modules/project/adapter/postgres/store_test.go`、`server/internal/modules/project/app/ports.go`、`server/internal/modules/project/domain/project.go`
- Generate: `server/internal/modules/project/adapter/postgres/gen/members.sql.go`、`server/internal/modules/project/adapter/postgres/gen/preferences.sql.go`、`server/internal/modules/project/adapter/postgres/gen/projects.sql.go`、`server/internal/modules/project/adapter/postgres/gen/states.sql.go`

**Interfaces:**
- Produces（spec 2.5）：`app.ProjectRow`、`MemberRow`、`PreferencesRow`、`StateRow`；`ProjectReader.GetProject(ctx, id, userID) (domain.Project, bool, error)`；`ProjectCreator`（`CreateProject`、`CreateMember`、`LowestSortOrder`、`CreatePreferences`、`CreateStates`）；`domain.ErrIdentifierTaken`、`ErrNameTaken`（409，存储把 `projects_workspace_id_identifier_key`、`projects_workspace_id_name_key` 的 23505 翻译成它们）。
- 使用者：Task 7 的用例；Task 6 的矩阵准备数据（经存储写项目和成员关系）。

**Tests:**
- `adapter/postgres/projects_test.go`：`TestCreateProjectStoresTheRow`（用例的值、其余列的默认值、审计列是时钟的时刻和创建者；别的项目不变；`GetProject` 读回，UTC，精确到微秒）；`TestCreateProjectKeepsTheLogo`（四个合法的图标原样读回）；`TestCreateProjectIdentifierOrNameTaken`（两个码；别的工作区的、已删除的项目不占）；`TestGetProject`（调用者的角色和位置只在成员关系有效时；有效成员按成为成员的顺序、再按成员关系的 id；夹具让丢了谓词的连接先读到别的行）。
- `adapter/postgres/rows_test.go`：`TestCreateTheRowsUnderAProject`（成员关系、显示设置、状态各存用例的值，别的项目、别的账户的行不变）；`TestCreateStatesStopsAtTheFirstFailure`；`TestLowestSortOrder`（别的工作区的、别的账户的、已删除的行都比答案低）。

- [ ] **Step 1: 查询**

`server/internal/modules/project/adapter/postgres/queries/projects.sql`（新文件，25 行）：

````file server/internal/modules/project/adapter/postgres/queries/projects.sql
-- name: CreateProject :exec
-- createProject (M3 design 3.6): the audit columns come from the use case's clock (M2 design 3.13); the columns the
-- insert does not name take their defaults.
INSERT INTO projects (id, workspace_id, name, description, identifier, network, project_lead_id, logo_props, timezone,
                      created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(name), sqlc.arg(description), sqlc.arg(identifier), sqlc.arg(network),
        sqlc.narg(project_lead_id), sqlc.arg(logo_props), sqlc.arg(timezone), sqlc.arg(created_by), sqlc.arg(created_by),
        sqlc.arg(now), sqlc.arg(now));

-- name: GetProject :one
-- The undeleted project, archived or not, as the user sees it (M3 design 3.19, 5.2): his project role and his place in
-- his sidebar while his membership is active, null otherwise; and the active members' accounts, in the order they
-- became members (3.12).
SELECT p.id, p.workspace_id, p.name, p.description, p.identifier, p.network, p.project_lead_id, p.default_assignee_id,
       p.cycle_view, p.module_view, p.issue_views_view, p.intake_view, p.guest_view_all_features, p.archive_in,
       p.archived_at, p.logo_props, p.timezone, p.created_at, p.updated_at, m.role AS member_role, u.sort_order,
       ARRAY(SELECT a.member_id FROM project_members a
             WHERE a.project_id = p.id AND a.is_active AND a.deleted_at IS NULL
             ORDER BY a.created_at, a.id)::uuid[] AS member_ids
FROM projects p
LEFT JOIN project_members m
       ON m.project_id = p.id AND m.member_id = sqlc.arg(user_id) AND m.is_active AND m.deleted_at IS NULL
LEFT JOIN project_user_properties u
       ON u.project_id = m.project_id AND u.user_id = m.member_id AND u.deleted_at IS NULL
WHERE p.id = sqlc.arg(id) AND p.deleted_at IS NULL;
````

`server/internal/modules/project/adapter/postgres/queries/members.sql`（新文件，4 行）：

````file server/internal/modules/project/adapter/postgres/queries/members.sql
-- name: CreateMember :exec
INSERT INTO project_members (id, workspace_id, project_id, member_id, role, created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(member_id), sqlc.arg(role), sqlc.arg(created_by),
        sqlc.arg(created_by), sqlc.arg(now), sqlc.arg(now));
````

`server/internal/modules/project/adapter/postgres/queries/preferences.sql`（新文件，15 行）：

````file server/internal/modules/project/adapter/postgres/queries/preferences.sql
-- name: CreatePreferences :exec
-- The navigation takes the column's default (M3 design 4.8); the place in the sidebar is the use case's (3.18).
INSERT INTO project_user_properties (id, workspace_id, project_id, user_id, sort_order, created_by_id, updated_by_id, created_at,
                                     updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(user_id), sqlc.arg(sort_order), sqlc.arg(created_by),
        sqlc.arg(created_by), sqlc.arg(now), sqlc.arg(now));

-- name: LowestSortOrder :one
-- The least place of the user's in his sidebar among the workspace's projects, over his undeleted display settings,
-- the ones of projects he left too (Plane's ProjectMember.save, M3 design 3.18); no row when he has none.
SELECT sort_order
FROM project_user_properties
WHERE workspace_id = sqlc.arg(workspace_id) AND user_id = sqlc.arg(user_id) AND deleted_at IS NULL
ORDER BY sort_order
LIMIT 1;
````

`server/internal/modules/project/adapter/postgres/queries/states.sql`（新文件，5 行）：

````file server/internal/modules/project/adapter/postgres/queries/states.sql
-- name: CreateState :exec
INSERT INTO states (id, workspace_id, project_id, name, color, sequence, "group", "default", created_by_id, updated_by_id,
                    created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(name), sqlc.arg(color), sqlc.arg(sequence),
        sqlc.arg(state_group), sqlc.arg(is_default), sqlc.arg(created_by), sqlc.arg(created_by), sqlc.arg(now), sqlc.arg(now));
````

- [ ] **Step 2: 生成**

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `6d7b3386b04161683c7d478b6273ac25482452f10a6ba6b41d7054c00c075c88` | 42 | `server/internal/modules/project/adapter/postgres/gen/members.sql.go` |
| `971b36adfebc7386951da967dd71e74c36b05b4ee95a521af5777febb003a573` | 66 | `server/internal/modules/project/adapter/postgres/gen/preferences.sql.go` |
| `ff94f1140b432f8b80502968542b8363639fe4349a80fa01bc98e49e32abd02d` | 132 | `server/internal/modules/project/adapter/postgres/gen/projects.sql.go` |
| `b455f15f07dec890c00294eb39178bf3a0f78d3ee3ed7e6c6b3d05ca52c1e9c1` | 49 | `server/internal/modules/project/adapter/postgres/gen/states.sql.go` |

Run: `shasum -a 256 server/internal/modules/project/adapter/postgres/gen/members.sql.go server/internal/modules/project/adapter/postgres/gen/preferences.sql.go server/internal/modules/project/adapter/postgres/gen/projects.sql.go server/internal/modules/project/adapter/postgres/gen/states.sql.go`
Expected: 与上表相同。

- [ ] **Step 3: 领域的错误、端口、存储**

`server/internal/modules/project/domain/errors.go`（新文件，12 行）：

````file server/internal/modules/project/domain/errors.go
package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The project module's errors (M3 design 5.3).
var (
	// ErrIdentifierTaken answers an identifier an undeleted project of the
	// workspace has.
	ErrIdentifierTaken = shared.NewError(shared.KindConflict, "project.identifier_taken", "A project of the workspace has this identifier.")
	// ErrNameTaken answers a name an undeleted project of the workspace has.
	ErrNameTaken = shared.NewError(shared.KindConflict, "project.name_taken", "A project of the workspace has this name.")
)
````

`server/internal/modules/project/domain/project.go`（修改，2 处）：

````old server/internal/modules/project/domain/project.go
	"strings"
````
````new server/internal/modules/project/domain/project.go
	"strings"
	"time"
````

````old server/internal/modules/project/domain/project.go
	"github.com/open-nerve/NerveProject/server/internal/shared"
)
````
````new server/internal/modules/project/domain/project.go
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Project is a project as a caller sees it (M3 design 5.2): its columns,
// and the caller's view of it.
type Project struct {
	ID                   uuid.UUID
	WorkspaceID          uuid.UUID
	Name                 string
	Description          string
	Identifier           string
	Network              Network
	LeadID               *uuid.UUID
	DefaultAssigneeID    *uuid.UUID
	CycleView            bool
	ModuleView           bool
	IssueViewsView       bool
	IntakeView           bool
	GuestViewAllFeatures bool
	ArchiveIn            int
	ArchivedAt           *time.Time
	LogoProps            LogoProps
	Timezone             string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	// The caller's view (M3 design 3.19): his project role and his place in
	// his sidebar, nil unless his membership is active; and the accounts of
	// the active members, in the order they became members.
	MemberRole *shared.Role
	SortOrder  *float64
	MemberIDs  []uuid.UUID
}
````

`server/internal/modules/project/app/ports.go`（修改，1 处）：

````old server/internal/modules/project/app/ports.go
	"uuid"
)
````
````new server/internal/modules/project/app/ports.go
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ProjectRow is a project to insert: checked values, its id, its creator
// and the time of the use case's clock. Every column it does not name takes
// its default.
type ProjectRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Name        string
	Description string
	Identifier  string
	Network     domain.Network
	LeadID      *uuid.UUID
	LogoProps   domain.LogoProps
	Timezone    string
	CreatedBy   uuid.UUID
	Now         time.Time
}

// MemberRow is a project membership to insert, active.
type MemberRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	MemberID    uuid.UUID
	Role        shared.Role
	CreatedBy   uuid.UUID
	Now         time.Time
}

// PreferencesRow is an account's display settings in a project to insert:
// the default navigation, and the place in his sidebar given (M3 design
// 3.18).
type PreferencesRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	UserID      uuid.UUID
	SortOrder   float64
	CreatedBy   uuid.UUID
	Now         time.Time
}

// StateRow is a state to insert.
type StateRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	State       domain.NewState
	CreatedBy   uuid.UUID
	Now         time.Time
}
````

`server/internal/modules/project/adapter/postgres/store.go`（修改，2 处）：

````old server/internal/modules/project/adapter/postgres/store.go
	"context"

````
````new server/internal/modules/project/adapter/postgres/store.go
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
````

````old server/internal/modules/project/adapter/postgres/store.go
	return gen.New(postgres.DB(ctx, s.pool))
}

````
````new server/internal/modules/project/adapter/postgres/store.go
	return gen.New(postgres.DB(ctx, s.pool))
}

// uniqueViolation reports whether err broke the unique constraint name.
func uniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == constraint
}

````

`server/internal/modules/project/adapter/postgres/projects.go`（新文件，70 行）：

````file server/internal/modules/project/adapter/postgres/projects.go
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
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CreateProject inserts p. An identifier or a name an undeleted project of
// the workspace has is domain.ErrIdentifierTaken or domain.ErrNameTaken.
// The domain checked every value, so a CHECK violation is a bug: an
// internal error (500), not a domain error.
func (s *Store) CreateProject(ctx context.Context, p app.ProjectRow) error {
	logo, err := json.Marshal(p.LogoProps)
	if err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	err = s.queries(ctx).CreateProject(ctx, gen.CreateProjectParams{
		ID: p.ID, WorkspaceID: p.WorkspaceID, Name: p.Name, Description: p.Description, Identifier: p.Identifier,
		Network: int16(p.Network), ProjectLeadID: p.LeadID, LogoProps: logo, Timezone: p.Timezone, CreatedBy: &p.CreatedBy, Now: p.Now,
	})
	switch {
	case uniqueViolation(err, "projects_workspace_id_identifier_key"):
		return domain.ErrIdentifierTaken
	case uniqueViolation(err, "projects_workspace_id_name_key"):
		return domain.ErrNameTaken
	case err != nil:
		return fmt.Errorf("create project: %w", err)
	}
	return nil
}

// GetProject returns the undeleted project id, archived or not, as userID
// sees it: his role and his place in his sidebar while his membership is
// active, and the active members' accounts. found is false when there is no
// such project. It reads in the transaction ctx carries.
func (s *Store) GetProject(ctx context.Context, id, userID uuid.UUID) (p domain.Project, found bool, err error) {
	r, err := s.queries(ctx).GetProject(ctx, gen.GetProjectParams{ID: id, UserID: userID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Project{}, false, nil
	case err != nil:
		return domain.Project{}, false, fmt.Errorf("read project %s: %w", id, err)
	}
	var logo domain.LogoProps
	if err := json.Unmarshal(r.LogoProps, &logo); err != nil {
		return domain.Project{}, false, fmt.Errorf("read logo_props of project %s: %w", id, err)
	}
	p = domain.Project{
		ID: r.ID, WorkspaceID: r.WorkspaceID, Name: r.Name, Description: r.Description, Identifier: r.Identifier,
		Network: domain.Network(r.Network), LeadID: r.ProjectLeadID, DefaultAssigneeID: r.DefaultAssigneeID,
		CycleView: r.CycleView, ModuleView: r.ModuleView, IssueViewsView: r.IssueViewsView, IntakeView: r.IntakeView,
		GuestViewAllFeatures: r.GuestViewAllFeatures, ArchiveIn: int(r.ArchiveIn), ArchivedAt: r.ArchivedAt, LogoProps: logo,
		Timezone: r.Timezone, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, SortOrder: r.SortOrder, MemberIDs: r.MemberIds,
	}
	if r.MemberRole != nil {
		role := shared.Role(*r.MemberRole)
		p.MemberRole = &role
	}
	return p, true, nil
}
````

`server/internal/modules/project/adapter/postgres/rows.go`（新文件，69 行）：

````file server/internal/modules/project/adapter/postgres/rows.go
package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
)

// The rows under a project: its memberships, its members' display settings
// and its states, one statement a row.

// CreateMember inserts m, active.
func (s *Store) CreateMember(ctx context.Context, m app.MemberRow) error {
	err := s.queries(ctx).CreateMember(ctx, gen.CreateMemberParams{
		ID: m.ID, WorkspaceID: m.WorkspaceID, ProjectID: m.ProjectID, MemberID: m.MemberID, Role: int16(m.Role),
		CreatedBy: &m.CreatedBy, Now: m.Now,
	})
	if err != nil {
		return fmt.Errorf("create project member: %w", err)
	}
	return nil
}

// CreatePreferences inserts p, the navigation at its default.
func (s *Store) CreatePreferences(ctx context.Context, p app.PreferencesRow) error {
	err := s.queries(ctx).CreatePreferences(ctx, gen.CreatePreferencesParams{
		ID: p.ID, WorkspaceID: p.WorkspaceID, ProjectID: p.ProjectID, UserID: p.UserID, SortOrder: p.SortOrder,
		CreatedBy: &p.CreatedBy, Now: p.Now,
	})
	if err != nil {
		return fmt.Errorf("create project preferences: %w", err)
	}
	return nil
}

// LowestSortOrder returns the least place of userID's in his sidebar among
// workspaceID's projects, over his undeleted display settings, those of
// projects he left too; nil when he has none (M3 design 3.18).
func (s *Store) LowestSortOrder(ctx context.Context, workspaceID, userID uuid.UUID) (*float64, error) {
	lowest, err := s.queries(ctx).LowestSortOrder(ctx, gen.LowestSortOrderParams{WorkspaceID: workspaceID, UserID: userID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("read the lowest sort order: %w", err)
	}
	return &lowest, nil
}

// CreateStates inserts rows in the order given, and stops at the first
// that fails.
func (s *Store) CreateStates(ctx context.Context, rows []app.StateRow) error {
	for _, r := range rows {
		err := s.queries(ctx).CreateState(ctx, gen.CreateStateParams{
			ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, Name: r.State.Name, Color: r.State.Color,
			Sequence: r.State.Sequence, StateGroup: string(r.State.Group), IsDefault: r.State.Default, CreatedBy: &r.CreatedBy, Now: r.Now,
		})
		if err != nil {
			return fmt.Errorf("create state %q: %w", r.State.Name, err)
		}
	}
	return nil
}
````

- [ ] **Step 4: 存储的测试**

`server/internal/modules/project/adapter/postgres/store_test.go`（修改，2 处）：

````old server/internal/modules/project/adapter/postgres/store_test.go
	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
````
````new server/internal/modules/project/adapter/postgres/store_test.go
	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
````

````old server/internal/modules/project/adapter/postgres/store_test.go
	exec(t, pool, "INSERT INTO workspaces (id, name, slug) VALUES ($1, $2, $2)", id, slug)
	return id
}

````
````new server/internal/modules/project/adapter/postgres/store_test.go
	exec(t, pool, "INSERT INTO workspaces (id, name, slug) VALUES ($1, $2, $2)", id, slug)
	return id
}

// newProject stores a public project of workspace named name, with the
// identifier identifier, created by by at now, and returns its id.
func newProject(t *testing.T, s *postgresadapter.Store, workspace uuid.UUID, name, identifier string, by uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	if err := s.CreateProject(context.Background(), app.ProjectRow{
		ID: id, WorkspaceID: workspace, Name: name, Identifier: identifier, Network: domain.NetworkPublic, Timezone: "UTC",
		CreatedBy: by, Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	return id
}

// tableRows is every row of table as text but the row id, in order: what
// an insert of the row id must leave as it was.
func tableRows(t *testing.T, pool *pgxpool.Pool, table string, id uuid.UUID) string {
	t.Helper()
	var s string
	if err := pool.QueryRow(context.Background(), "SELECT coalesce(string_agg(r::text, E'\\n' ORDER BY r::text), '') FROM "+table+
		" r WHERE r.id <> $1", id).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

````

`server/internal/modules/project/adapter/postgres/projects_test.go`（新文件，213 行）：

````file server/internal/modules/project/adapter/postgres/projects_test.go
package postgresadapter_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func ptr[T any](v T) *T { return &v }

// jsonOf is v as JSON, to compare values that hold pointers.
func jsonOf(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// CreateProject stores the use case's values, the other columns at their
// defaults, and the audit columns at the clock's time, by the creator; it
// leaves every other project as it was. GetProject reads it back as
// stored, in UTC, to the microsecond.
func TestCreateProjectStoresTheRow(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	newProject(t, s, acme, "Ops", "OPS", bob)
	newProject(t, s, beta, "Web", "WEB", bob)
	row := app.ProjectRow{
		ID: uuid.NewV7(), WorkspaceID: acme, Name: "研发 Web", Description: "The web app", Identifier: "WEBÇ", Network: domain.NetworkPrivate,
		LeadID: &bob, LogoProps: domain.LogoProps{InUse: ptr("emoji"), Emoji: &domain.Emoji{Value: ptr("128640")}}, Timezone: "Asia/Shanghai",
		CreatedBy: alice, Now: now,
	}
	others := tableRows(t, pool, "projects", row.ID)

	if err := s.CreateProject(context.Background(), row); err != nil {
		t.Fatal(err)
	}

	got, found, err := s.GetProject(context.Background(), row.ID, alice)
	want := domain.Project{
		ID: row.ID, WorkspaceID: acme, Name: "研发 Web", Description: "The web app", Identifier: "WEBÇ", Network: domain.NetworkPrivate,
		LeadID: &bob, LogoProps: row.LogoProps, Timezone: "Asia/Shanghai", CreatedAt: now, UpdatedAt: now, MemberIDs: []uuid.UUID{},
	}
	if g, w := jsonOf(t, got), jsonOf(t, want); err != nil || !found || g != w || got.CreatedAt.Location() != time.UTC {
		t.Errorf("GetProject() = %s, %v, %v; want\n%s", g, found, err, w)
	}
	var createdBy, updatedBy uuid.UUID
	var sequence int
	var deleted *time.Time
	if err := pool.QueryRow(context.Background(),
		"SELECT created_by_id, updated_by_id, last_issue_sequence, deleted_at FROM projects WHERE id = $1", row.ID).
		Scan(&createdBy, &updatedBy, &sequence, &deleted); err != nil {
		t.Fatal(err)
	}
	if createdBy != alice || updatedBy != alice || sequence != 0 || deleted != nil {
		t.Errorf("row: created_by %s updated_by %s, last_issue_sequence %d, deleted %v; want alice, 0, not deleted", createdBy, updatedBy, sequence, deleted)
	}
	if after := tableRows(t, pool, "projects", row.ID); after != others {
		t.Errorf("the other projects:\n%s\nwant\n%s", after, others)
	}
}

// The four values of logo_props the CHECK accepts (M3 design 4.6) come
// back as they were stored: no icon, an emoji only, an icon only, every
// key.
func TestCreateProjectKeepsTheLogo(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	for i, logo := range []domain.LogoProps{
		{},
		{Emoji: &domain.Emoji{Value: ptr("128640")}},
		{Icon: &domain.Icon{Name: ptr("home"), Color: ptr("#6d7b8a")}},
		{InUse: ptr("icon"), Emoji: &domain.Emoji{Value: ptr("128640"), URL: ptr("https://example.com/e.png")},
			Icon: &domain.Icon{Name: ptr("home"), Color: ptr("#6d7b8a"), BackgroundColor: ptr("#ffffff")}},
	} {
		id, name := uuid.NewV7(), "P"+string(rune('A'+i))
		err := s.CreateProject(context.Background(), app.ProjectRow{
			ID: id, WorkspaceID: acme, Name: name, Identifier: name, LogoProps: logo, Timezone: "UTC", CreatedBy: alice, Now: now,
		})
		if err != nil {
			t.Fatal(err)
		}
		got, _, err := s.GetProject(context.Background(), id, alice)
		if g, w := jsonOf(t, got.LogoProps), jsonOf(t, logo); err != nil || g != w {
			t.Errorf("the logo %s came back as %s, %v", w, g, err)
		}
	}
}

// An identifier and a name are unique among the workspace's undeleted
// projects only: another workspace's project, and a deleted one, leave
// them free (M3 design 3.19).
func TestCreateProjectIdentifierOrNameTaken(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	first := newProject(t, s, acme, "Web", "WEB", alice)
	create := func(workspace uuid.UUID, name, identifier string) error {
		return s.CreateProject(context.Background(), app.ProjectRow{
			ID: uuid.NewV7(), WorkspaceID: workspace, Name: name, Identifier: identifier, Timezone: "UTC", CreatedBy: alice, Now: now,
		})
	}

	if err := create(acme, "Web 2", "WEB"); !errors.Is(err, domain.ErrIdentifierTaken) {
		t.Errorf("a taken identifier: %v, want project.identifier_taken", err)
	}
	if err := create(acme, "Web", "WEB2"); !errors.Is(err, domain.ErrNameTaken) {
		t.Errorf("a taken name: %v, want project.name_taken", err)
	}
	if err := create(beta, "Web", "WEB"); err != nil {
		t.Errorf("another workspace's name and identifier: %v, want them free", err)
	}
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", first, now)
	if err := create(acme, "Web", "WEB"); err != nil {
		t.Errorf("a deleted project's name and identifier: %v, want them free", err)
	}
}

// GetProject answers the caller's view (M3 design 3.19): his role and his
// place in his sidebar only while his membership is active, and the active
// members in the order they became members, then by the membership's id.
// The fixture has each state once, next to the rows another predicate
// would let in: another project's rows of the same accounts, an inactive
// and a deleted membership, a deleted display settings row; and rows that
// lie in the table in another order than the answer's.
func TestGetProject(t *testing.T) {
	s, pool := newStore(t)
	var ids []uuid.UUID
	for _, email := range []string{"alice@corp.com", "bob@corp.com", "carol@corp.com", "dave@corp.com", "erin@corp.com", "frank@corp.com",
		"gina@corp.com"} {
		ids = append(ids, newAccount(t, pool, email))
	}
	alice, bob, carol, dave, erin, frank, gina := ids[0], ids[1], ids[2], ids[3], ids[4], ids[5], ids[6]
	acme := newWorkspace(t, pool, "acme")
	// ops first: its rows come first in the tables, so a join that loses a
	// predicate reads them before web's.
	ops := newProject(t, s, acme, "Ops", "OPS", alice)
	web := newProject(t, s, acme, "Web", "WEB", alice)
	member := func(id, project, user uuid.UUID, role shared.Role, at time.Time, sortOrder float64) {
		t.Helper()
		ctx := context.Background()
		if err := s.CreateMember(ctx, app.MemberRow{ID: id, WorkspaceID: acme, ProjectID: project, MemberID: user, Role: role,
			CreatedBy: alice, Now: at}); err != nil {
			t.Fatal(err)
		}
		if err := s.CreatePreferences(ctx, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: project, UserID: user,
			SortOrder: sortOrder, CreatedBy: alice, Now: at}); err != nil {
			t.Fatal(err)
		}
	}
	for _, u := range []uuid.UUID{alice, bob, carol, dave, frank, gina} {
		member(uuid.NewV7(), ops, u, shared.RoleAdmin, now, -1)
	}
	// In web: carol's membership is stored first but began last; gina's
	// began with dave's and has the smaller id, stored after his; alice's
	// began between. bob's ended, frank's was deleted; carol's display
	// settings were deleted; erin was never a member.
	ginasID := uuid.NewV7()
	member(uuid.NewV7(), web, carol, shared.RoleGuest, now.Add(2*time.Minute), 30)
	member(uuid.NewV7(), web, dave, shared.RoleMember, now, 40)
	member(ginasID, web, gina, shared.RoleMember, now, 60)
	member(uuid.NewV7(), web, alice, shared.RoleAdmin, now.Add(time.Minute), 10)
	member(uuid.NewV7(), web, bob, shared.RoleMember, now, 20)
	member(uuid.NewV7(), web, frank, shared.RoleMember, now, 50)
	exec(t, pool, "UPDATE project_members SET is_active = false WHERE project_id = $1 AND member_id = $2", web, bob)
	exec(t, pool, "UPDATE project_members SET deleted_at = $3 WHERE project_id = $1 AND member_id = $2", web, frank, now)
	exec(t, pool, "UPDATE project_user_properties SET deleted_at = $3 WHERE project_id = $1 AND user_id = $2", web, carol, now)
	members := []uuid.UUID{gina, dave, alice, carol}
	tests := []struct {
		name      string
		user      uuid.UUID
		role      *shared.Role
		sortOrder *float64
	}{
		{"alice, admin", alice, ptr(shared.RoleAdmin), ptr(10.0)},
		{"dave, member", dave, ptr(shared.RoleMember), ptr(40.0)},
		{"carol, guest without display settings", carol, ptr(shared.RoleGuest), nil},
		{"bob, membership ended", bob, nil, nil},
		{"frank, membership deleted", frank, nil, nil},
		{"erin, never a member", erin, nil, nil},
	}
	for _, tt := range tests {
		got, found, err := s.GetProject(context.Background(), web, tt.user)
		if err != nil || !found || got.ID != web || got.Name != "Web" || jsonOf(t, got.MemberRole) != jsonOf(t, tt.role) ||
			jsonOf(t, got.SortOrder) != jsonOf(t, tt.sortOrder) || !slices.Equal(got.MemberIDs, members) {
			t.Errorf("%s: GetProject() = %+v, %v, %v; want role %v, sort order %v, members %v", tt.name, got, found, err,
				jsonOf(t, tt.role), jsonOf(t, tt.sortOrder), members)
		}
	}
	// Archived: found as it is; deleted, or no project: not found.
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", web, now)
	if got, found, err := s.GetProject(context.Background(), web, alice); err != nil || !found || got.ArchivedAt == nil || !got.ArchivedAt.Equal(now) {
		t.Errorf("the archived project: %+v, %v, %v; want it, archived at %v", got, found, err, now)
	}
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", web, now)
	for _, id := range []uuid.UUID{web, uuid.NewV7()} {
		if got, found, err := s.GetProject(context.Background(), id, alice); err != nil || found {
			t.Errorf("GetProject(%s) = %+v, %v, %v; want not found", id, got, found, err)
		}
	}
}
````

`server/internal/modules/project/adapter/postgres/rows_test.go`（新文件，166 行）：

````file server/internal/modules/project/adapter/postgres/rows_test.go
package postgresadapter_test

import (
	"context"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The rows under a project store the use case's values, their other
// columns at their defaults: an active membership, the default navigation,
// a state's empty description. Each insert leaves the table's other rows,
// of another project and another account, as they were.
func TestCreateTheRowsUnderAProject(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, pool, "acme")
	ops, web := newProject(t, s, acme, "Ops", "OPS", alice), newProject(t, s, acme, "Web", "WEB", alice)
	ctx := context.Background()
	// Rows of another project and of another account, before each insert.
	for _, row := range []struct {
		project, user uuid.UUID
	}{{ops, bob}, {web, alice}} {
		if err := s.CreateMember(ctx, app.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: row.project, MemberID: row.user,
			Role: shared.RoleAdmin, CreatedBy: alice, Now: now}); err != nil {
			t.Fatal(err)
		}
		if err := s.CreatePreferences(ctx, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: row.project, UserID: row.user,
			SortOrder: 1, CreatedBy: alice, Now: now}); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.CreateStates(ctx, []app.StateRow{{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: ops, CreatedBy: alice, Now: now,
		State: domain.NewState{Name: "Backlog", Color: "#000000", Sequence: 1, Group: "backlog", Default: true}}}); err != nil {
		t.Fatal(err)
	}
	member, prefs := uuid.NewV7(), uuid.NewV7()
	states := []app.StateRow{
		{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, CreatedBy: bob, Now: now,
			State: domain.NewState{Name: "Backlog", Color: "#60646C", Sequence: 15000, Group: "backlog", Default: true}},
		{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, CreatedBy: bob, Now: now,
			State: domain.NewState{Name: "Triage", Color: "#4E5355", Sequence: 65000, Group: "triage"}},
	}
	before := map[string]string{"project_members": tableRows(t, pool, "project_members", member),
		"project_user_properties": tableRows(t, pool, "project_user_properties", prefs), "states": tableRows(t, pool, "states", uuid.UUID{})}

	if err := s.CreateMember(ctx, app.MemberRow{
		ID: member, WorkspaceID: acme, ProjectID: web, MemberID: bob, Role: shared.RoleMember, CreatedBy: alice, Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreatePreferences(ctx, app.PreferencesRow{
		ID: prefs, WorkspaceID: acme, ProjectID: web, UserID: bob, SortOrder: -9999.5, CreatedBy: alice, Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateStates(ctx, states); err != nil {
		t.Fatal(err)
	}

	var got string
	if err := pool.QueryRow(ctx, `SELECT concat_ws(' | ',
		(SELECT concat_ws(' ', workspace_id, project_id, member_id, role, is_active, created_by_id, updated_by_id, created_at = $2,
			updated_at = $2, deleted_at IS NULL) FROM project_members WHERE id = $3),
		(SELECT concat_ws(' ', workspace_id, project_id, user_id, preferences, sort_order, created_by_id, updated_by_id, created_at = $2,
			updated_at = $2, deleted_at IS NULL) FROM project_user_properties WHERE id = $4),
		(SELECT string_agg(concat_ws(' ', workspace_id, project_id, name, description = '', color, sequence, "group", "default",
			created_by_id, updated_by_id, created_at = $2, updated_at = $2, deleted_at IS NULL), ' / ' ORDER BY sequence)
			FROM states WHERE project_id = $1))`, web, now, member, prefs).Scan(&got); err != nil {
		t.Fatal(err)
	}
	under := func(s string) string { return acme.String() + " " + web.String() + " " + s }
	a, b := alice.String(), bob.String()
	want := under(b+" 15 t "+a+" "+a+" t t t") + " | " +
		under(b+` {"navigation": {"default_tab": "work_items", "hide_in_more_menu": []}} -9999.5 `+a+" "+a+" t t t") + " | " +
		under("Backlog t #60646C 15000 backlog t "+b+" "+b+" t t t") + " / " +
		under("Triage t #4E5355 65000 triage f "+b+" "+b+" t t t")
	if got != want {
		t.Errorf("the rows =\n%s\nwant\n%s", got, want)
	}
	for table, rows := range map[string]string{"project_members": tableRows(t, pool, "project_members", member),
		"project_user_properties": tableRows(t, pool, "project_user_properties", prefs)} {
		if rows != before[table] {
			t.Errorf("the other rows of %s:\n%s\nwant\n%s", table, rows, before[table])
		}
	}
	var others string
	if err := pool.QueryRow(ctx, "SELECT string_agg(r::text, E'\\n' ORDER BY r::text) FROM states r WHERE project_id <> $1", web).Scan(&others); err != nil ||
		others != before["states"] {
		t.Errorf("the other states: %s, %v; want %s", others, err, before["states"])
	}
}

// CreateStates stops at the first state that fails, and names it: a
// second default state breaks states_project_id_default_key.
func TestCreateStatesStopsAtTheFirstFailure(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web := newProject(t, s, acme, "Web", "WEB", alice)
	row := func(name string) app.StateRow {
		return app.StateRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, CreatedBy: alice, Now: now,
			State: domain.NewState{Name: name, Color: "#60646C", Group: "backlog", Default: true}}
	}

	err := s.CreateStates(context.Background(), []app.StateRow{row("One"), row("Two"), row("Three")})

	var names []string
	if err := pool.QueryRow(context.Background(), "SELECT coalesce(array_agg(name ORDER BY name), '{}') FROM states").Scan(&names); err != nil {
		t.Fatal(err)
	}
	if err == nil || !strings.Contains(err.Error(), `"Two"`) || !strings.Contains(err.Error(), "states_project_id_default_key") ||
		len(names) != 1 || names[0] != "One" {
		t.Errorf("CreateStates() = %v, the states %q; want Two's failure and only One stored", err, names)
	}
}

// LowestSortOrder is the least place of the account's in the workspace's
// projects, over his undeleted display settings, those of a project he
// left too; nil when he has none. Each row the query must pass over has a
// lower place than the answer: another workspace's, another account's, a
// deleted one.
func TestLowestSortOrder(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web, ops, old := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice), newProject(t, s, acme, "Old", "OLD", alice)
	other := newProject(t, s, beta, "Web", "WEB", alice)
	for _, row := range []struct {
		project, user uuid.UUID
		sortOrder     float64
	}{{web, alice, 100}, {ops, alice, 50}, {old, alice, -5}, {other, alice, -1000}, {web, bob, -2000}} {
		workspace := acme
		if row.project == other {
			workspace = beta
		}
		if err := s.CreatePreferences(context.Background(), app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: workspace, ProjectID: row.project,
			UserID: row.user, SortOrder: row.sortOrder, CreatedBy: alice, Now: now}); err != nil {
			t.Fatal(err)
		}
	}
	// ops stands for a project alice left: its display settings stay
	// undeleted. old's were deleted.
	exec(t, pool, "UPDATE project_user_properties SET deleted_at = $2 WHERE project_id = $1", old, now)

	for _, tt := range []struct {
		name            string
		workspace, user uuid.UUID
		want            *float64
	}{
		{"alice in acme", acme, alice, ptr(50.0)},
		{"bob in acme", acme, bob, ptr(-2000.0)},
		{"alice in beta", beta, alice, ptr(-1000.0)},
		{"carol, none", acme, carol, nil},
		{"bob in beta, none", beta, bob, nil},
	} {
		got, err := s.LowestSortOrder(context.Background(), tt.workspace, tt.user)
		if err != nil || jsonOf(t, got) != jsonOf(t, tt.want) {
			t.Errorf("%s: LowestSortOrder() = %v, %v; want %v", tt.name, jsonOf(t, got), err, jsonOf(t, tt.want))
		}
	}
}
````

- [ ] **Step 5: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/project/...`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 6: 提交**

```bash
git add server/internal/modules/project/adapter/postgres/projects.go server/internal/modules/project/adapter/postgres/projects_test.go server/internal/modules/project/adapter/postgres/queries/members.sql server/internal/modules/project/adapter/postgres/queries/preferences.sql server/internal/modules/project/adapter/postgres/queries/projects.sql server/internal/modules/project/adapter/postgres/queries/states.sql server/internal/modules/project/adapter/postgres/rows.go server/internal/modules/project/adapter/postgres/rows_test.go server/internal/modules/project/adapter/postgres/store.go server/internal/modules/project/adapter/postgres/store_test.go server/internal/modules/project/app/ports.go server/internal/modules/project/domain/errors.go server/internal/modules/project/domain/project.go server/internal/modules/project/adapter/postgres/gen/members.sql.go server/internal/modules/project/adapter/postgres/gen/preferences.sql.go server/internal/modules/project/adapter/postgres/gen/projects.sql.go server/internal/modules/project/adapter/postgres/gen/states.sql.go
```
```bash
git commit -m "feat(M3/P4a): the project store writes a project and reads it as a user sees it

CreateProject, CreateMember, CreatePreferences and CreateStates write
the checked values at the use case's time, the other columns at their
defaults; a taken identifier or name is its own 409. GetProject reads
an undeleted project with the user's role and place while his membership
is active, and its active members in the order they joined.
LowestSortOrder is the user's least place in the workspace's projects.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`GetProject` 的五个关键谓词另按两个行序各跑一次，`mutants_rev.py`）：

| 改坏 | 必须失败的测试 |
|---|---|
| 标识被占答成名称被占；不存图标；不存负责人 | `TestCreateProjectIdentifierOrNameTaken`；`TestCreateProjectKeepsTheLogo`、`TestCreateProjectStoresTheRow`；`TestCreateProjectStoresTheRow` |
| 成员关系存成别的角色；每个状态都存成默认 | `TestCreateTheRowsUnderAProject` |
| `CreateStates` 失败之后继续 | `TestCreateStatesStopsAtTheFirstFailure` |
| `GetProject` 去掉 12 个谓词中的任何一个（成员列表的项目、有效、未删除；调用者的成员关系的项目、账户、有效、未删除；显示设置的项目、账户、未删除；项目的 id、未删除），成员列表的顺序只按时间、只按 id | `TestGetProject`（关键的五个在 id 升序、降序两个行序下都失败） |
| `LowestSortOrder` 不看工作区、账户、`deleted_at`；不排序；倒序 | `TestLowestSortOrder` |

**Done when:** 插入和读取的每个值、每个默认值、每个谓词由真实数据库上的测试核对，每个写的测试都有别的项目和别的账户的行。

---

### Task 5: `workspace.Provide` 的 `WorkspaceDirectory`、`WorkspaceMembers`

**Files:**
- Create: `server/internal/modules/workspace/adapter/postgres/directory.go`、`server/internal/modules/workspace/adapter/postgres/directory_test.go`、`server/internal/modules/workspace/adapter/postgres/queries/directory.sql`、`server/internal/modules/workspace/app/directory.go`
- Modify: `server/internal/modules/workspace/adapter/postgres/store.go`、`server/internal/modules/workspace/module.go`
- Generate: `server/internal/modules/workspace/adapter/postgres/gen/directory.sql.go`

**Interfaces:**
- Produces（spec 2.6，M3 设计 3.6 约定二、三，6.5）：`workspace.WorkspaceDirectory`（`WorkspaceBySlug`：不加锁；`ShareWorkspaceBySlug`：`FOR SHARE`，等锁期间删除的工作区读不到）、`workspace.WorkspaceMembers`（`ShareMembers(ctx, workspaceID, userIDs) (map[uuid.UUID]shared.Role, error)`：锁住这些账户未删除的成员行，有效的、已结束的都锁，按 id 的顺序 `FOR SHARE`，答有效成员的角色）、`workspace.DirectoryEntry`（`app.DirectoryEntry{ID, Timezone}`）；`Provided` 加两个字段，都由 `postgresadapter.NewDirectory(pool)` 实现。
- 使用者：Task 7、10、11 的用例（经 Task 8 的 `projectWorkspaces` 转换）。

**Tests:**（`adapter/postgres/directory_test.go`）
- `TestWorkspaceDirectoryFindsTheUndeletedWorkspace`（两种读都按 slug 找到未删除的工作区和它的时区，不是已删除的、不是大小写不同的，每个工作区各是自己的）。
- `TestTheDirectorysLockIsForShare`（等工作区的 `FOR NO KEY UPDATE`、让它等，不与另一个 `FOR SHARE` 互等，不锁别的工作区；`lock_timeout` 下答 `lock_not_available`）；`TestTheDirectorysLockSeesADeletionItWaitedFor`。
- `TestShareMembersAnswersTheActiveMembersRoles`（不是已结束的、已删除的、别的工作区的、没问到的）；`TestShareMembersLocksTheRowsAskedFor`（已结束的也锁，别的账户、别的工作区、已删除的行不锁）；`TestShareMembersLocksInIDOrder`（行在表里、在索引里的顺序都与 id 相反时仍按 id 取锁）。

- [ ] **Step 1: 查询**

`server/internal/modules/workspace/adapter/postgres/queries/directory.sql`（新文件，25 行）：

````file server/internal/modules/workspace/adapter/postgres/queries/directory.sql
-- What workspace.Provide offers the project module (M3 design 6.5).

-- name: DirectoryWorkspace :one
-- WorkspaceDirectory: the undeleted workspace with the slug, read without a lock.
SELECT id, timezone
FROM workspaces
WHERE slug = sqlc.arg(slug) AND deleted_at IS NULL;

-- name: ShareDirectoryWorkspace :one
-- WorkspaceDirectory's lock: the parent lock of a write that adds a project under the workspace (M3 design 3.6
-- convention 2), as ShareWorkspaceBySlug takes it. After a wait, Postgres evaluates deleted_at IS NULL again on the
-- row's newest version, so a workspace deleted meanwhile reads no row.
SELECT id, timezone
FROM workspaces
WHERE slug = sqlc.arg(slug) AND deleted_at IS NULL
FOR SHARE;

-- name: ShareMembers :many
-- WorkspaceMembers (M3 design 3.6 convention 3): the users' undeleted memberships of the workspace, active or not,
-- locked FOR SHARE in id order. The lock is taken as the sorted rows come, so the order is the ids'.
SELECT member_id, role, is_active
FROM workspace_members
WHERE workspace_id = sqlc.arg(workspace_id) AND member_id = ANY (sqlc.arg(user_ids)::uuid[]) AND deleted_at IS NULL
ORDER BY id
FOR SHARE;
````

- [ ] **Step 2: 生成**

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `be607abc0e16ddee253ed5dcaf497b45ee80f85abb58585b1b9d00c792531a04` | 96 | `server/internal/modules/workspace/adapter/postgres/gen/directory.sql.go` |

Run: `shasum -a 256 server/internal/modules/workspace/adapter/postgres/gen/directory.sql.go`
Expected: 与上表相同。

- [ ] **Step 3: 目录**

`server/internal/modules/workspace/app/directory.go`（新文件，12 行）：

````file server/internal/modules/workspace/app/directory.go
package app

import "uuid"

// DirectoryEntry is a workspace as the project module finds it by its slug
// through workspace.Provide's WorkspaceDirectory (M3 design 6.5): its id,
// and its time zone, a new project's unless the caller gives one (3.19).
// bootstrap converts it into project's value.
type DirectoryEntry struct {
	ID       uuid.UUID
	Timezone string
}
````

`server/internal/modules/workspace/adapter/postgres/directory.go`（新文件，76 行）：

````file server/internal/modules/workspace/adapter/postgres/directory.go
package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Directory is what workspace.Provide offers the project module (M3 design
// 6.5): WorkspaceDirectory, the undeleted workspace a slug names, and
// WorkspaceMembers, the memberships a project write makes members from,
// locked (3.6 convention 3). It reads the store's tables under names of its
// own: the store's WorkspaceBySlug and ShareWorkspaceBySlug answer the
// workspace module's use cases.
type Directory struct {
	store *Store
}

// NewDirectory returns the directory over pool.
func NewDirectory(pool *pgxpool.Pool) *Directory {
	return &Directory{store: New(pool)}
}

// WorkspaceBySlug returns the undeleted workspace with slug, read without a
// lock; found is false when there is none.
func (d *Directory) WorkspaceBySlug(ctx context.Context, slug string) (w app.DirectoryEntry, found bool, err error) {
	r, err := d.store.queries(ctx).DirectoryWorkspace(ctx, slug)
	return directoryEntry(r.ID, r.Timezone, err)
}

// ShareWorkspaceBySlug returns the undeleted workspace with slug and locks
// its row FOR SHARE until the transaction ctx carries ends: the parent lock
// of a write that adds a project under it (M3 design 3.6 convention 2).
// found is false when there is none, also when it was deleted while the
// lock waited.
func (d *Directory) ShareWorkspaceBySlug(ctx context.Context, slug string) (w app.DirectoryEntry, found bool, err error) {
	r, err := d.store.queries(ctx).ShareDirectoryWorkspace(ctx, slug)
	return directoryEntry(r.ID, r.Timezone, err)
}

func directoryEntry(id uuid.UUID, timezone string, err error) (app.DirectoryEntry, bool, error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return app.DirectoryEntry{}, false, nil
	case err != nil:
		return app.DirectoryEntry{}, false, fmt.Errorf("find the workspace: %w", err)
	}
	return app.DirectoryEntry{ID: id, Timezone: timezone}, true, nil
}

// ShareMembers locks the undeleted memberships of userIDs in workspaceID,
// active or not, FOR SHARE in id order until the transaction ctx carries
// ends, and returns the roles of the active ones by account; an account
// without an active membership is not in the map (M3 design 3.6 convention
// 3).
func (d *Directory) ShareMembers(ctx context.Context, workspaceID uuid.UUID, userIDs []uuid.UUID) (map[uuid.UUID]shared.Role, error) {
	rows, err := d.store.queries(ctx).ShareMembers(ctx, gen.ShareMembersParams{WorkspaceID: workspaceID, UserIds: userIDs})
	if err != nil {
		return nil, fmt.Errorf("lock the workspace members: %w", err)
	}
	roles := map[uuid.UUID]shared.Role{}
	for _, r := range rows {
		if r.IsActive {
			roles[r.MemberID] = shared.Role(r.Role)
		}
	}
	return roles, nil
}
````

`server/internal/modules/workspace/adapter/postgres/store.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/store.go
// modules make of workspaces through ports (M3 design 6.5): WorkspaceRoles.
````
````new server/internal/modules/workspace/adapter/postgres/store.go
// modules make of workspaces through ports (M3 design 6.5): WorkspaceRoles,
// and the Directory's WorkspaceDirectory and WorkspaceMembers.
````

`server/internal/modules/workspace/module.go`（修改，3 处）：

````old server/internal/modules/workspace/module.go
}

// Provided are the adapters workspace offers the other modules. They depend
````
````new server/internal/modules/workspace/module.go
}

// WorkspaceDirectory finds an undeleted workspace by its slug for the
// project module (M3 design 6.5); found is false when there is none.
type WorkspaceDirectory interface {
	// WorkspaceBySlug reads it without a lock.
	WorkspaceBySlug(ctx context.Context, slug string) (w DirectoryEntry, found bool, err error)
	// ShareWorkspaceBySlug locks its row FOR SHARE until the transaction
	// ctx carries ends: the parent lock of a write that adds a project
	// (M3 design 3.6 convention 2). A workspace deleted while the lock
	// waited is not found.
	ShareWorkspaceBySlug(ctx context.Context, slug string) (w DirectoryEntry, found bool, err error)
}

// WorkspaceMembers locks the memberships that a write of the project
// module makes project members from (M3 design 3.6 convention 3).
type WorkspaceMembers interface {
	// ShareMembers locks userIDs' undeleted memberships of workspaceID,
	// active or not, FOR SHARE in id order until the transaction ctx
	// carries ends, and returns the roles of the active ones by account.
	ShareMembers(ctx context.Context, workspaceID uuid.UUID, userIDs []uuid.UUID) (map[uuid.UUID]shared.Role, error)
}

// DirectoryEntry is the workspace WorkspaceDirectory finds: bootstrap
// converts it into project's value (M3 design 6.5).
type DirectoryEntry = app.DirectoryEntry

// Provided are the adapters workspace offers the other modules. They depend
````

````old server/internal/modules/workspace/module.go
	WorkspaceRoles WorkspaceRoles
````
````new server/internal/modules/workspace/module.go
	WorkspaceRoles     WorkspaceRoles
	WorkspaceDirectory WorkspaceDirectory
	WorkspaceMembers   WorkspaceMembers
````

````old server/internal/modules/workspace/module.go
	return Provided{WorkspaceRoles: postgresadapter.New(pool)}
````
````new server/internal/modules/workspace/module.go
	directory := postgresadapter.NewDirectory(pool)
	return Provided{WorkspaceRoles: postgresadapter.New(pool), WorkspaceDirectory: directory, WorkspaceMembers: directory}
````

`server/internal/modules/workspace/adapter/postgres/directory_test.go`（新文件，283 行）：

````file server/internal/modules/workspace/adapter/postgres/directory_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Both of the directory's reads find the undeleted workspace a slug names,
// with its id and its time zone: not a deleted one, not by another case of
// the slug, and each workspace its own.
func TestWorkspaceDirectoryFindsTheUndeletedWorkspace(t *testing.T) {
	s, pool := newStore(t)
	d := postgresadapter.NewDirectory(pool)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	beta := newWorkspace(t, s, "Beta", "beta", alice)
	gone := newWorkspace(t, s, "Gone", "gone", alice)
	exec(t, pool, "UPDATE workspaces SET timezone = 'Asia/Shanghai' WHERE id = $1", beta.ID)
	exec(t, pool, "UPDATE workspaces SET deleted_at = $2 WHERE id = $1", gone.ID, now)
	finds := map[string]func(context.Context, string) (app.DirectoryEntry, bool, error){
		"WorkspaceBySlug": d.WorkspaceBySlug, "ShareWorkspaceBySlug": d.ShareWorkspaceBySlug,
	}
	for name, find := range finds {
		tx := postgres.NewTxManager(pool, 2*time.Second)
		err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
			for slug, want := range map[string]app.DirectoryEntry{"acme": {ID: acme.ID, Timezone: "UTC"}, "beta": {ID: beta.ID, Timezone: "Asia/Shanghai"}} {
				if got, found, err := find(ctx, slug); err != nil || !found || got != want {
					t.Errorf("%s(%s) = %+v, %v, %v; want %+v", name, slug, got, found, err, want)
				}
			}
			for _, slug := range []string{"gone", "ACME", "acm", "nothing"} {
				if got, found, err := find(ctx, slug); err != nil || found || got != (app.DirectoryEntry{}) {
					t.Errorf("%s(%q) = %+v, %v, %v; want not found", name, slug, got, found, err)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// The directory's lock is convention 2's FOR SHARE: it waits for the
// workspace's FOR NO KEY UPDATE and makes it wait, not another FOR SHARE,
// and no lock of another workspace. A lock that waits ends with
// lock_not_available under a lock_timeout.
func TestTheDirectorysLockIsForShare(t *testing.T) {
	const directory = "Directory.ShareWorkspaceBySlug"
	tests := []struct {
		held, then string
		slug       string // then's
		waits      bool
	}{
		{directory, noKeyUpdate.name, "acme", true},
		{noKeyUpdate.name, directory, "acme", true},
		{directory, forShare.name, "acme", false},
		{forShare.name, directory, "acme", false},
		{directory, noKeyUpdate.name, "beta", false},
		{noKeyUpdate.name, directory, "beta", false},
	}
	for _, tt := range tests {
		t.Run(tt.held+" held, "+tt.then+" of "+tt.slug, func(t *testing.T) {
			s, pool := newStore(t)
			d := postgresadapter.NewDirectory(pool)
			locks := map[string]lock{noKeyUpdate.name: noKeyUpdate, forShare.name: forShare, directory: {directory,
				func(ctx context.Context, _ *postgresadapter.Store, w named) (uuid.UUID, error) {
					e, found, err := d.ShareWorkspaceBySlug(ctx, w.slug)
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

// A workspace deleted while the directory's lock waits is not found: the
// lock's statement has deleted_at IS NULL, which Postgres evaluates again
// on the row's newest version after the wait.
func TestTheDirectorysLockSeesADeletionItWaitedFor(t *testing.T) {
	s, pool := newStore(t)
	d := postgresadapter.NewDirectory(pool)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	end := hold(t, tx, func(ctx context.Context) error {
		if _, err := s.LockWorkspaceBySlug(ctx, "acme"); err != nil {
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
			a.w, a.found, err = d.ShareWorkspaceBySlug(ctx, "acme")
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
			t.Errorf("ShareWorkspaceBySlug() after the deletion = %+v, %v, %v; want not found", a.w, a.found, a.err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("ShareWorkspaceBySlug() did not end within 10s")
	}
}

// ShareMembers answers the active members' roles among the accounts asked
// for, in the workspace asked for: not an ended membership, not a deleted
// one, not another workspace's, not an account not asked for.
func TestShareMembersAnswersTheActiveMembersRoles(t *testing.T) {
	s, pool := newStore(t)
	d := postgresadapter.NewDirectory(pool)
	var ids []uuid.UUID
	for _, email := range []string{"alice@corp.com", "bob@corp.com", "carol@corp.com", "dave@corp.com", "erin@corp.com", "frank@corp.com", "gina@corp.com"} {
		ids = append(ids, newAccount(t, pool, email))
	}
	alice, bob, carol, dave, erin, frank, gina := ids[0], ids[1], ids[2], ids[3], ids[4], ids[5], ids[6]
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", frank)
	join(t, s, acme.ID, bob, shared.RoleMember)
	join(t, s, acme.ID, carol, shared.RoleGuest)
	join(t, s, acme.ID, dave, shared.RoleMember)
	join(t, s, acme.ID, erin, shared.RoleMember)
	join(t, s, acme.ID, gina, shared.RoleMember)
	join(t, s, beta.ID, bob, shared.RoleAdmin)
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE workspace_id = $1 AND member_id = $2", acme.ID, dave)
	exec(t, pool, "UPDATE workspace_members SET deleted_at = $3 WHERE workspace_id = $1 AND member_id = $2", acme.ID, erin, now)

	var got map[uuid.UUID]shared.Role
	err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		var err error
		got, err = d.ShareMembers(ctx, acme.ID, []uuid.UUID{alice, bob, carol, dave, erin, frank, uuid.NewV7()})
		return err
	})

	want := map[uuid.UUID]shared.Role{alice: shared.RoleAdmin, bob: shared.RoleMember, carol: shared.RoleGuest}
	if err != nil || !maps.Equal(got, want) {
		t.Errorf("ShareMembers() = %v, %v; want %v", got, err, want)
	}
}

// ShareMembers locks FOR SHARE the undeleted memberships asked for in the
// workspace, the ended one too, and no other row: an update of a locked row
// waits; one of another account's, of the same account in another
// workspace, or of a deleted row does not.
func TestShareMembersLocksTheRowsAskedFor(t *testing.T) {
	s, pool := newStore(t)
	d := postgresadapter.NewDirectory(pool)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	dave, erin := newAccount(t, pool, "dave@corp.com"), newAccount(t, pool, "erin@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", bob)
	join(t, s, acme.ID, bob, shared.RoleMember)
	join(t, s, acme.ID, carol, shared.RoleMember)
	join(t, s, acme.ID, dave, shared.RoleMember)
	join(t, s, acme.ID, erin, shared.RoleMember)
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE workspace_id = $1 AND member_id = $2", acme.ID, carol)
	exec(t, pool, "UPDATE workspace_members SET deleted_at = $3 WHERE workspace_id = $1 AND member_id = $2", acme.ID, erin, now)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	hold(t, tx, func(ctx context.Context) error {
		_, err := d.ShareMembers(ctx, acme.ID, []uuid.UUID{bob, carol, erin})
		return err
	})
	for _, tt := range []struct {
		name            string
		workspace, user uuid.UUID
		waits           bool
	}{
		{"bob's in acme, asked for", acme.ID, bob, true},
		{"carol's ended one, asked for", acme.ID, carol, true},
		{"dave's, not asked for", acme.ID, dave, false},
		{"erin's deleted one", acme.ID, erin, false},
		{"bob's in beta", beta.ID, bob, false},
	} {
		err := withLockTimeout(tx, pool, func(ctx context.Context) error {
			_, err := postgres.DB(ctx, pool).Exec(ctx, "UPDATE workspace_members SET role = role WHERE workspace_id = $1 AND member_id = $2",
				tt.workspace, tt.user)
			return err
		})
		var pgErr *pgconn.PgError
		if waited := errors.As(err, &pgErr) && pgErr.Code == "55P03"; waited != tt.waits || (!waited && err != nil) {
			t.Errorf("%s: updating it = %v; want a wait %v", tt.name, err, tt.waits)
		}
	}
}

// ShareMembers takes its locks in the memberships' id order, whatever
// order the rows lie in, in the table or in an index: bob's membership has
// the smaller id, but lies after carol's in the table, and bob's account
// has the greater id, so the (workspace_id, member_id) index lists carol's
// first too. carol's row is held. ShareMembers waits for carol's holding
// bob's, which an update of bob's row then waits for; in any other order it
// would reach carol's first and wait holding nothing.
func TestShareMembersLocksInIDOrder(t *testing.T) {
	s, pool := newStore(t)
	d := postgresadapter.NewDirectory(pool)
	alice, carol, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "carol@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	bobsID := uuid.NewV7() // drawn first: the smaller id
	exec(t, pool, "INSERT INTO workspace_members (id, workspace_id, member_id, role) VALUES ($1, $2, $3, 15)", uuid.NewV7(), acme.ID, carol)
	exec(t, pool, "INSERT INTO workspace_members (id, workspace_id, member_id, role) VALUES ($1, $2, $3, 15)", bobsID, acme.ID, bob)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	end := hold(t, tx, func(ctx context.Context) error {
		_, err := postgres.DB(ctx, pool).Exec(ctx, "SELECT id FROM workspace_members WHERE member_id = $1 FOR NO KEY UPDATE", carol)
		return err
	})
	done := make(chan error, 1)
	go func() {
		done <- tx.WithinTx(context.Background(), func(ctx context.Context) error {
			_, err := d.ShareMembers(ctx, acme.ID, []uuid.UUID{carol, bob})
			return err
		})
	}()
	pgtest.WaitForLockWaitOn(t, pool, "workspace_members", 10*time.Second)

	err := withLockTimeout(tx, pool, func(ctx context.Context) error {
		_, err := postgres.DB(ctx, pool).Exec(ctx, "UPDATE workspace_members SET role = role WHERE id = $1", bobsID)
		return err
	})

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "55P03" {
		t.Errorf("updating bob's row while ShareMembers waits for carol's = %v; want lock_not_available: bob's is locked first", err)
	}
	if err := end(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("ShareMembers() = %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("ShareMembers() did not end within 10s")
	}
}
````

- [ ] **Step 4: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/workspace/...`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 5: 提交**

```bash
git add server/internal/modules/workspace/adapter/postgres/directory.go server/internal/modules/workspace/adapter/postgres/directory_test.go server/internal/modules/workspace/adapter/postgres/queries/directory.sql server/internal/modules/workspace/adapter/postgres/store.go server/internal/modules/workspace/app/directory.go server/internal/modules/workspace/module.go server/internal/modules/workspace/adapter/postgres/gen/directory.sql.go
```
```bash
git commit -m "feat(M3/P4a): workspace.Provide offers the project module its directory

WorkspaceDirectory finds the undeleted workspace a slug names, without
a lock for a read or FOR SHARE as a new project's parent lock, which
sees a deletion it waited for. WorkspaceMembers locks the memberships a
project write makes members from FOR SHARE in id order, ended ones too,
and answers the active members' roles (M3 design 3.6 conventions 2, 3).

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 |
|---|---|
| 两种读各去掉 `slug`、`deleted_at IS NULL` | `TestWorkspaceDirectoryFindsTheUndeletedWorkspace`（锁的 `deleted_at` 另有 `TestTheDirectorysLockSeesADeletionItWaitedFor`） |
| 目录的锁用 `FOR KEY SHARE`、`FOR NO KEY UPDATE`、不加锁 | `TestTheDirectorysLockIsForShare` |
| `ShareMembers` 去掉 `workspace_id`、`member_id`、`deleted_at IS NULL` | `TestShareMembersAnswersTheActiveMembersRoles`、`TestShareMembersLocksTheRowsAskedFor` |
| `ShareMembers` 不排序；倒序 | `TestShareMembersLocksInIDOrder` |
| `ShareMembers` 用 `FOR KEY SHARE`；不加锁 | `TestShareMembersLocksTheRowsAskedFor`（另有 `TestShareMembersLocksInIDOrder`） |
| `ShareMembers` 答已结束的成员关系 | `TestShareMembersAnswersTheActiveMembersRoles` |

**Done when:** 目录的两种读、锁的强度、锁到的行和锁的顺序都由真实数据库上的测试核对；`workspace` 不导入 `project`。

---

### Task 6: 矩阵的形状：每行带自己的列；项目级的列、账户和项目；按参数列出的"不是目标"

**Files:**
- Create: `server/internal/bootstrap/permission_matrix_columns_test.go`、`server/internal/bootstrap/permission_matrix_targets_test.go`
- Modify: `server/internal/bootstrap/permission_matrix_coverage_test.go`、`server/internal/bootstrap/permission_matrix_invitations_test.go`、`server/internal/bootstrap/permission_matrix_seeded_test.go`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/bootstrap/permission_matrix_workspace_test.go`

**Interfaces:**
- Produces（spec 2.7，M3 设计 9.2；P1 review 第 6 节 M8，P2 re-review Minor 2，裁定 G3）：`matrixRow.columns`（`nil` 是工作区级的六列）和 `columnsOf()`；`projectColumns`（9.2 的十列，X 展开为从来不是成员、已被移出、工作区已删除三个账户，共 12 列）、`archivedColumns`（已归档项目的一列）；`matrixAccounts`（11 个账户）、`accountOf(c)`（一列以哪个账户调用）、`projectOf(c)`（一列指向哪个项目）；`seeded.project(key)`，准备数据 `matrixProjects`、`matrixProjectMembers`，经项目的存储写入（`projectSeed`）；`notTarget{path, param}`：按路径参数列出，不是整条路径；`targetViolation(pattern, path, c, s, passOver)` 加 `{project_id}` 的规则（必须是这一列的项目），`pathOf`、`targetViolation` 移到 `permission_matrix_targets_test.go`。
- 矩阵的运行：一行的每个格子都运行，不按列的清单（按工作区级的清单会悄悄跳过项目级的格子）；完整性核对要求一行恰好有它每一列的格子，没有别的列的格子。
- 使用者：Task 8–11 的项目行。

**Tests:**
- `permission_matrix_columns_test.go`：`TestEveryColumnCallsAsARegisteredAccount`（每一列的账户都已注册，每个注册的账户都是某一列的，共 11 个）；`TestMatrixViolationsCatchesEachColumnGap`（项目行少 PA 的格子；项目行有工作区列的格子；工作区行有项目列的格子；格子指向别的列的项目；列出的 `{identifier}` 旁边的 `{slug}` 指向别的列的工作区（裁定 G3 的反例）；`{identifier}` 没有列出；列出的参数不在路径里；从来没有准备的项目让测试立即失败并指名）。
- `TestMatrixViolationsCatchesEachGap`（P1–P3 的反例）改为按参数列出；`TestPermissionMatrix` 照新的运行方式，工作区级的格子不变（132 格）。

- [ ] **Step 1: 项目级的列和格子的目标**

`server/internal/bootstrap/permission_matrix_columns_test.go`（新文件，183 行）：

````file server/internal/bootstrap/permission_matrix_columns_test.go
package bootstrap

import (
	"fmt"
	"maps"
	"net/http"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
)

// The columns of the project level (M3 design 9.2), the accounts that call
// from them, and the projects they target.

// The project level's columns that are not the workspace level's. PA, PM
// and the member before are workspace members; PG, and WG-, the workspace's
// guests; PM+WA and WA-, its admins. X is the workspace level's three
// outsiders: never a member, removed, and the deleted workspace's.
const (
	callerProjectAdmin   caller = "project admin"                      // PA
	callerProjectMember  caller = "project member"                     // PM
	callerProjectGuest   caller = "project guest"                      // PG: the workspace's guest
	callerMemberAndAdmin caller = "project member and workspace admin" // PM+WA
	callerAdminOnly      caller = "workspace admin only"               // WA-: the workspace's admin
	callerMemberPublic   caller = "workspace member only, public"      // WM-公: the workspace's member
	callerMemberPrivate  caller = "workspace member only, private"     // WM-私: the workspace's member
	callerGuestOnly      caller = "workspace guest only"               // WG-
	callerBefore         caller = "project member before"              // P-前: ended in the private project
)

// projectColumns are the columns of the project level, in the order of
// 9.2's table, X as its three accounts.
var projectColumns = []caller{callerProjectAdmin, callerProjectMember, callerProjectGuest, callerMemberAndAdmin, callerAdminOnly,
	callerMemberPublic, callerMemberPrivate, callerGuestOnly, callerBefore, callerNever, callerRemoved, callerDeleted}

// matrixAccounts are the accounts prepareMatrix registers: each workspace
// column's, and each project column's that is none of those (M3 design
// 9.2: eleven).
var matrixAccounts = append(slices.Clone(workspaceColumns), callerProjectAdmin, callerProjectMember, callerMemberAndAdmin, callerGuestOnly,
	callerBefore)

// accountOf is the account a column's cells call as: a project column that
// is a workspace column's account under the project level's name is that
// account.
func accountOf(c caller) caller {
	switch c {
	case callerProjectGuest:
		return callerGuest
	case callerAdminOnly:
		return callerAdmin
	case callerMemberPublic, callerMemberPrivate:
		return callerMember
	}
	return c
}

// projectOf is the key of the project a project column's cells target:
// acme's private project for the columns about it, gone's for the deleted
// workspace's, acme's public one for every other; PA, PM, PG and PM+WA
// answer alike in both, WG- too (9.2).
func projectOf(c caller) string {
	switch c {
	case callerMemberPrivate, callerBefore:
		return "acme/private"
	case callerDeleted:
		return "gone/project"
	}
	return "acme/public"
}

// Every account of a column is registered, and every account registered is
// some column's: a column whose account prepareMatrix did not register would
// call with no token, and its cells answer 401 whatever the rule.
func TestEveryColumnCallsAsARegisteredAccount(t *testing.T) {
	var used []caller
	for _, c := range slices.Concat(workspaceColumns, projectColumns) {
		if !slices.Contains(matrixAccounts, accountOf(c)) {
			t.Errorf("column %s calls as %s, which prepareMatrix does not register", c, accountOf(c))
		}
		used = append(used, accountOf(c))
	}
	for _, a := range matrixAccounts {
		if !slices.Contains(used, a) {
			t.Errorf("the account %s is no column's", a)
		}
	}
	if len(matrixAccounts) != 11 {
		t.Errorf("%d accounts, want 9.2's 11", len(matrixAccounts))
	}
}

// Each check of the project level's columns and of the not-target
// parameters fails on its counterexample: a project row's columns are
// projectColumns, each cell aims at its column's project, and a parameter
// listed as not a target leaves the path's others checked.
func TestMatrixViolationsCatchesEachColumnGap(t *testing.T) {
	s := newSeeded().in(t)
	getProject := apitest.Operation{ID: "getProject", Tags: []string{"project"}, Method: http.MethodGet, Path: "/api/v0/projects/{project_id}"}
	cells := map[caller]cell{}
	for _, c := range projectColumns {
		cells[c] = cellOK
	}
	toProject := func(c caller, s seeded) (string, string, string) {
		return http.MethodGet, "/api/v0/projects/" + s.project(projectOf(c)).String(), ""
	}
	row := matrixRow{op: getProject.ID, columns: projectColumns, request: toProject, cells: cells}
	// with is row with change made to a copy of it.
	with := func(change func(r *matrixRow)) matrixRow {
		r := row
		r.cells = maps.Clone(cells)
		change(&r)
		return r
	}
	identifiers := apitest.Operation{ID: "checkProjectIdentifier", Tags: []string{"project"}, Method: http.MethodGet,
		Path: "/api/v0/workspaces/{slug}/project-identifiers/{identifier}"}
	checks := matrixRow{op: identifiers.ID, request: toWorkspace(http.MethodGet, "/project-identifiers/WEB", ""), cells: every(cellOK)}
	listed := matrixExemptions{notTargets: map[notTarget]string{{identifiers.Path, "{identifier}"}: "an identifier asked about"}}
	ops := []apitest.Operation{getProject, identifiers}
	if got := matrixViolations(ops, listed, []matrixRow{row, checks}, s, nil); len(got) != 0 {
		t.Fatalf("a matching matrix: %q, want none", got)
	}
	public, private := s.project("acme/public").String(), s.project("acme/private").String()
	tests := []struct {
		name   string
		exempt matrixExemptions
		rows   []matrixRow
		want   []string
	}{
		{"a project row without a cell for PA", listed, []matrixRow{with(func(r *matrixRow) { delete(r.cells, callerProjectAdmin) }), checks},
			[]string{"row getProject has no cell for project admin"}},
		{"a project row with a workspace column's cell", listed, []matrixRow{with(func(r *matrixRow) { r.cells[callerAdmin] = cellOK }), checks},
			[]string{"row getProject has a cell for admin, which is none of its columns"}},
		{"a workspace row with a project column's cell", listed, []matrixRow{row, func() matrixRow {
			r := checks
			r.cells = maps.Clone(checks.cells)
			r.cells[callerBefore] = cellOK
			return r
		}()}, []string{"row checkProjectIdentifier has a cell for project member before, which is none of its columns"}},
		{"a cell of another column's project", listed, []matrixRow{with(func(r *matrixRow) {
			r.request = func(c caller, s seeded) (string, string, string) {
				if c == callerMemberPrivate {
					return http.MethodGet, "/api/v0/projects/" + public, ""
				}
				return toProject(c, s)
			}
		}), checks}, []string{fmt.Sprintf("row getProject, %s: {project_id} %s is not its column's project acme/private, %s",
			callerMemberPrivate, public, private)}},
		{"a {slug} of another column's workspace beside a listed {identifier}", listed, []matrixRow{row, func() matrixRow {
			r := checks
			r.request = sameRequest(http.MethodGet, "/api/v0/workspaces/acme/project-identifiers/WEB", "")
			return r
		}()}, []string{"row checkProjectIdentifier, workspace deleted: targets the workspace acme, not its column's gone"}},
		{"an {identifier} not listed", matrixExemptions{}, []matrixRow{row, checks}, func() []string {
			var want []string
			for _, c := range workspaceColumns {
				want = append(want, fmt.Sprintf("row checkProjectIdentifier, %s: {identifier} is no target the matrix knows: "+
					"list /api/v0/workspaces/{slug}/project-identifiers/{identifier} as not a target, with its reason", c))
			}
			return want
		}()},
		{"a not-target parameter its path does not have", matrixExemptions{notTargets: map[notTarget]string{
			{identifiers.Path, "{identifier}"}: "an identifier asked about", {identifiers.Path, "{project_id}"}: "a mistake"}},
			[]matrixRow{row, checks},
			[]string{"the not-target {project_id} of /api/v0/workspaces/{slug}/project-identifiers/{identifier} is no parameter of an operation's path"}},
	}
	for _, tt := range tests {
		if got := matrixViolations(ops, tt.exempt, tt.rows, s, nil); !slices.Equal(got, tt.want) {
			t.Errorf("%s: %q, want %q", tt.name, got, tt.want)
		}
	}
	// A project never seeded fails the test at once, and names it.
	failed := fatalOf(func(tb testing.TB) {
		matrixViolations(ops, listed, []matrixRow{with(func(r *matrixRow) {
			r.request = func(c caller, s seeded) (string, string, string) {
				return http.MethodGet, "/api/v0/projects/" + s.project("acme/nothing").String(), ""
			}
		})}, newSeeded().in(tb), nil)
	})
	if want := "no project acme/nothing is seeded"; failed != want {
		t.Errorf("a project never seeded: failed with %q, want %q", failed, want)
	}
}
````

`server/internal/bootstrap/permission_matrix_targets_test.go`（新文件，70 行）：

````file server/internal/bootstrap/permission_matrix_targets_test.go
package bootstrap

import (
	"fmt"
	"strings"
	"uuid"
)

// Where a cell's request points: the path of its operation, and the target
// of its column.

// pathOf reports whether path, its query left out, is a path of the
// contract's pattern: each {parameter} one segment that is not empty, every
// other segment the same.
func pathOf(pattern, path string) bool {
	path, _, _ = strings.Cut(path, "?")
	want, got := strings.Split(pattern, "/"), strings.Split(path, "/")
	if len(want) != len(got) {
		return false
	}
	for i, segment := range want {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			if got[i] == "" {
				return false
			}
		} else if segment != got[i] {
			return false
		}
	}
	return true
}

// targetViolation is what is wrong with where path, a path of pattern that
// a cell of the column c sends, points; "" when nothing. A workspace named
// by its slug ({slug} right after workspaces) must be workspaceOf(c), a
// project named by its id ({project_id}) must be projectOf(c)'s, and any
// other row named by its id (a parameter ending in _id) must be a row of s
// under workspaceOf(c): a cell of the deleted workspace's column that named
// acme would get the 404 of a workspace its caller is not in, and pass
// whether deleted workspaces are hidden or not. A parameter passOver
// reports is passed over, each other checked. Any other parameter is
// reported: a parameter that is no column's target is listed as such, with
// its reason (matrixExemptions.notTargets), and not given here.
func targetViolation(pattern, path string, c caller, s seeded, passOver func(param string) bool) string {
	path, _, _ = strings.Cut(path, "?")
	want, got := strings.Split(pattern, "/"), strings.Split(path, "/")
	for i, segment := range want {
		switch {
		case !strings.HasPrefix(segment, "{") || !strings.HasSuffix(segment, "}"):
		case passOver(segment):
		case segment == "{project_id}":
			if id := s.project(projectOf(c)).String(); got[i] != id {
				return fmt.Sprintf("{project_id} %s is not its column's project %s, %s", got[i], projectOf(c), id)
			}
		case segment == "{slug}" && i > 0 && want[i-1] == "workspaces":
			if got[i] != workspaceOf(c) {
				return fmt.Sprintf("targets the workspace %s, not its column's %s", got[i], workspaceOf(c))
			}
		case strings.HasSuffix(segment, "_id}"):
			id, err := uuid.Parse(got[i])
			slug, isRow := s.workspaceOfRow(id)
			if err != nil || !isRow || slug != workspaceOf(c) {
				return fmt.Sprintf("%s %s is no row seeded under its column's workspace %s", segment, got[i], workspaceOf(c))
			}
		default:
			return fmt.Sprintf("%s is no target the matrix knows: list %s as not a target, with its reason", segment, pattern)
		}
	}
	return ""
}
````

- [ ] **Step 2: 每行带自己的列；准备的账户和项目；按参数列出**

`server/internal/bootstrap/permission_matrix_test.go`（修改，19 处）：

````old server/internal/bootstrap/permission_matrix_test.go
	"fmt"
````
````new server/internal/bootstrap/permission_matrix_test.go
	"fmt"
	"maps"
````

````old server/internal/bootstrap/permission_matrix_test.go
	"uuid"

````
````new server/internal/bootstrap/permission_matrix_test.go
	"uuid"

	projectpg "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
````

````old server/internal/bootstrap/permission_matrix_test.go
// memberships through the workspace store, the deleted workspace through
// the API, and the state no store writes yet through SQL (prepareMatrix).
````
````new server/internal/bootstrap/permission_matrix_test.go
// memberships through the workspace store, the projects and their
// memberships through the project store, the deleted workspace through the
// API, and the state no store writes yet through SQL (prepareMatrix).
````

````old server/internal/bootstrap/permission_matrix_test.go
	notTargets: map[string]string{
		"/api/v0/workspace-slugs/{slug}": "a slug asked about, not a workspace: the answer is the same for every caller",
		"/api/v0/workspace-invitations/{invitation_id}/accept": "account level: each column answers an invitation to its own " +
			"address (ownInvitation), or acme's newcomer's, whatever workspace its column targets",
		"/api/v0/workspace-invitations/{invitation_id}/decline": "account level, as accept",
````
````new server/internal/bootstrap/permission_matrix_test.go
	notTargets: map[notTarget]string{
		{"/api/v0/workspace-slugs/{slug}", "{slug}"}: "a slug asked about, not a workspace: the answer is the same for every caller",
		{"/api/v0/workspace-invitations/{invitation_id}/accept", "{invitation_id}"}: "account level: each column answers an invitation " +
			"to its own address (ownInvitation), or acme's newcomer's, whatever workspace its column targets",
		{"/api/v0/workspace-invitations/{invitation_id}/decline", "{invitation_id}"}: "account level, as accept",
````

````old server/internal/bootstrap/permission_matrix_test.go
// notTargets are the paths whose parameters name nothing a column's cell
// must aim at its workspace, each with its reason: targetViolation passes
// them over, and reports any other parameter it does not know.
````
````new server/internal/bootstrap/permission_matrix_test.go
// notTargets are the parameters of paths that name nothing a column's cell
// must aim at, each with its reason: targetViolation passes over the
// parameter listed, still checks the path's other parameters, and reports
// any parameter it does not know.
````

````old server/internal/bootstrap/permission_matrix_test.go
	public     map[string]string // operationId → the test that stands for its row
	notTargets map[string]string // path → why its parameters are no column's target
}
````
````new server/internal/bootstrap/permission_matrix_test.go
	public     map[string]string    // operationId → the test that stands for its row
	notTargets map[notTarget]string // a path's parameter → why it is no column's target
}

// notTarget is a parameter of a path, both as the contract spells them.
type notTarget struct{ path, param string }
````

````old server/internal/bootstrap/permission_matrix_test.go
// name a row prepareMatrix seeded, and the answer each gets.
````
````new server/internal/bootstrap/permission_matrix_test.go
// name a row prepareMatrix seeded, and the answer each gets, by the name of
// the caller's column.
````

````old server/internal/bootstrap/permission_matrix_test.go
	config  func(*config.Config)
````
````new server/internal/bootstrap/permission_matrix_test.go
	config  func(*config.Config)
	// columns are the row's columns: nil for the workspace level's
	// (workspaceColumns), projectColumns for a row of the project level.
	columns []caller
````

````old server/internal/bootstrap/permission_matrix_test.go
	return r.op + ", " + r.variant
````
````new server/internal/bootstrap/permission_matrix_test.go
	return r.op + ", " + r.variant
}

// columnsOf are the columns r has cells for: its own, or the workspace
// level's.
func (r matrixRow) columnsOf() []caller {
	if r.columns == nil {
		return workspaceColumns
	}
	return r.columns
````

````old server/internal/bootstrap/permission_matrix_test.go
// of it shares, each column's access token, and the seeded ids.
````
````new server/internal/bootstrap/permission_matrix_test.go
// of it shares, each account's access token (accountOf a column), and the
// seeded ids.
````

````old server/internal/bootstrap/permission_matrix_test.go
// prepareMatrix fills a database for the matrix. Through the API, an
// account for each column, registered for its token. Through the workspace
// store, the workspaces, memberships and invitations of matrixMemberships
// and matrixInvitations, with the ids newSeeded named, and acme's admin's
````
````new server/internal/bootstrap/permission_matrix_test.go
// prepareMatrix fills a database for the matrix. Through the API, each of
// matrixAccounts, registered for its token. Through the workspace store,
// the workspaces, memberships and invitations of matrixMemberships and
// matrixInvitations, with the ids newSeeded named, and acme's admin's
````

````old server/internal/bootstrap/permission_matrix_test.go
// role read in the wrong workspace lets either into acme. Through the API,
// gone deleted by its admin, which soft-deletes its memberships with it.
// Through SQL, until P5's store replaces it, the removed member's
// membership of acme ended. Everything that connected to the database is
// closed when it returns, so that it can be copied. A -run that leaves out
// prepare fails here, not with a 401 in every cell.
````
````new server/internal/bootstrap/permission_matrix_test.go
// role read in the wrong workspace lets either into acme. Through the
// project store, the projects and project memberships of matrixProjects
// and matrixProjectMembers. Through the API, gone deleted by its admin,
// which soft-deletes its memberships and its project with it. Through SQL,
// until the stores of P4b and P5 replace it, acme's archived project
// archived, the member before's membership of the private project ended,
// and the removed member's membership of acme ended. Everything that
// connected to the database is closed when it returns, so that it can be
// copied. A -run that leaves out prepare fails here, not with a 401 in
// every cell.
````

````old server/internal/bootstrap/permission_matrix_test.go
		ids := map[caller]uuid.UUID{}
		for _, c := range workspaceColumns {
````
````new server/internal/bootstrap/permission_matrix_test.go
		ids := map[caller]uuid.UUID{}
		for _, c := range matrixAccounts {
````

````old server/internal/bootstrap/permission_matrix_test.go
		// No store removes a member yet (P5), so SQL stands in until that
		// phase replaces it: it ends the removed member's membership of acme.
````
````new server/internal/bootstrap/permission_matrix_test.go
		projects := projectSeed{matrixSeed: seed, store: projectpg.New(pool), projects: map[string]uuid.UUID{}}
		for _, p := range matrixProjects {
			projects.project(s.project(p.key), p.key, p.name, p.identifier, p.network)
		}
		for _, pm := range matrixProjectMembers {
			projects.join(pm.key, pm.c, pm.role)
		}
		// No store archives a project (P4b), ends a project membership (P5)
		// or removes a member (P5) yet, so SQL stands in until those phases
		// replace it.
		seed.exec(pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", s.project("acme/archived"), seed.now)
		seed.exec(pool, "UPDATE project_members SET is_active = false WHERE project_id = $1 AND member_id = $2",
			s.project("acme/private"), ids[callerBefore])
````

````old server/internal/bootstrap/permission_matrix_test.go
		// membership and invitations go with the workspace row, so every
		// cell of the column is asked about a workspace deleted the one way
		// there is. A membership left active in a deleted workspace is
````
````new server/internal/bootstrap/permission_matrix_test.go
		// memberships, invitations and project go with the workspace row, so
		// every cell of the column is asked about a workspace deleted the one
		// way there is. A membership left active in a deleted workspace is
````

````old server/internal/bootstrap/permission_matrix_test.go
	if len(d.tokens) != len(workspaceColumns) {
````
````new server/internal/bootstrap/permission_matrix_test.go
	if len(d.tokens) != len(matrixAccounts) {
````

````old server/internal/bootstrap/permission_matrix_test.go
// config) on its copy, at most matrixApps of those at once. Every answer a
// row's check is for is checked, and counted: a harness that skipped the
// checks would fail.
````
````new server/internal/bootstrap/permission_matrix_test.go
// config) on its copy, at most matrixApps of those at once. Every cell a
// row has runs, of whichever level: a run over a list of columns would skip
// the cells of the other level's in silence, and
// TestThePermissionMatrixCoversEveryOperation holds a row's cells to its
// own columns. Every answer a row's check is for is checked, and counted: a
// harness that skipped the checks would fail.
````

````old server/internal/bootstrap/permission_matrix_test.go
		for _, c := range workspaceColumns {
			want, ok := r.cells[c]
			if !ok {
				continue // TestThePermissionMatrixCoversEveryOperation reports it
			}
````
````new server/internal/bootstrap/permission_matrix_test.go
		for _, c := range slices.Sorted(maps.Keys(r.cells)) {
			want := r.cells[c]
````

````old server/internal/bootstrap/permission_matrix_test.go
				status, answer := call(t, contract, method, base+path, d.tokens[c], body)
````
````new server/internal/bootstrap/permission_matrix_test.go
				status, answer := call(t, contract, method, base+path, d.tokens[accountOf(c)], body)
````

`server/internal/bootstrap/permission_matrix_coverage_test.go`（修改，11 处）：

````old server/internal/bootstrap/permission_matrix_coverage_test.go
// path that is no operation's; a row that names no operation; a row without a cell for a column; a cell
// whose request is not the operation its row names, so that no row tests
// another operation under its name; a cell that does not target its
// column's workspace (targetViolation), so that no column quietly tests
// another's case; a row that sends anything but GET without write, whose
// cells could write on the copy the reading cells share. The requests name
// the rows of s.
````
````new server/internal/bootstrap/permission_matrix_coverage_test.go
// parameter that no operation's path has; a row that names no operation; a
// row without a cell for one of its columns, or with a cell for a column it
// does not have; a cell whose request is not the operation its row names,
// so that no row tests another operation under its name; a cell that does
// not target its column's workspace or project (targetViolation), so that
// no column quietly tests another's case; a row that sends anything but GET
// without write, whose cells could write on the copy the reading cells
// share. The requests name the rows of s.
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		for _, c := range workspaceColumns {
````
````new server/internal/bootstrap/permission_matrix_coverage_test.go
		for _, c := range r.columnsOf() {
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
				if _, listed := exempt.notTargets[op.Path]; listed {
					break
````
````new server/internal/bootstrap/permission_matrix_coverage_test.go
				listed := func(param string) bool {
					_, ok := exempt.notTargets[notTarget{op.Path, param}]
					return ok
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
				if v := targetViolation(op.Path, path, c, s); v != "" {
````
````new server/internal/bootstrap/permission_matrix_coverage_test.go
				if v := targetViolation(op.Path, path, c, s, listed); v != "" {
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
			found = append(found, fmt.Sprintf("row %s sends %s without write: its cells could run on the reads' copy", r.name(), unsafe))
````
````new server/internal/bootstrap/permission_matrix_coverage_test.go
			found = append(found, fmt.Sprintf("row %s sends %s without write: its cells could run on the reads' copy", r.name(), unsafe))
		}
		for _, c := range slices.Sorted(maps.Keys(r.cells)) {
			if !slices.Contains(r.columnsOf(), c) {
				found = append(found, fmt.Sprintf("row %s has a cell for %s, which is none of its columns", r.name(), c))
			}
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
	for _, path := range slices.Sorted(maps.Keys(exempt.notTargets)) {
		if !slices.ContainsFunc(ops, func(op apitest.Operation) bool { return op.Path == path }) {
			found = append(found, fmt.Sprintf("the not-target path %s is no operation's", path))
````
````new server/internal/bootstrap/permission_matrix_coverage_test.go
	for _, n := range slices.SortedFunc(maps.Keys(exempt.notTargets), func(a, b notTarget) int {
		return strings.Compare(a.path+" "+a.param, b.path+" "+b.param)
	}) {
		if !slices.ContainsFunc(ops, func(op apitest.Operation) bool {
			return op.Path == n.path && slices.Contains(strings.Split(n.path, "/"), n.param)
		}) {
			found = append(found, fmt.Sprintf("the not-target %s of %s is no parameter of an operation's path", n.param, n.path))
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
	return found
}

// pathOf reports whether path, its query left out, is a path of the
// contract's pattern: each {parameter} one segment that is not empty, every
// other segment the same.
func pathOf(pattern, path string) bool {
	path, _, _ = strings.Cut(path, "?")
	want, got := strings.Split(pattern, "/"), strings.Split(path, "/")
	if len(want) != len(got) {
		return false
	}
	for i, segment := range want {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			if got[i] == "" {
				return false
			}
		} else if segment != got[i] {
			return false
		}
	}
	return true
}

// targetViolation is what is wrong with where path, a path of pattern that
// a cell of the column c sends, points; "" when nothing. A workspace named
// by its slug ({slug} right after workspaces) must be workspaceOf(c), and a
// row named by its id (a parameter ending in _id) must be a row of s under
// workspaceOf(c): a cell of the deleted workspace's column that named acme
// would get the 404 of a workspace its caller is not in, and pass whether
// deleted workspaces are hidden or not. Any other parameter is reported: a
// path whose parameters are no column's target is listed as such, with its
// reason (matrixExemptions.notTargets), and not given here.
func targetViolation(pattern, path string, c caller, s seeded) string {
	path, _, _ = strings.Cut(path, "?")
	want, got := strings.Split(pattern, "/"), strings.Split(path, "/")
	for i, segment := range want {
		switch {
		case !strings.HasPrefix(segment, "{") || !strings.HasSuffix(segment, "}"):
		case segment == "{slug}" && i > 0 && want[i-1] == "workspaces":
			if got[i] != workspaceOf(c) {
				return fmt.Sprintf("targets the workspace %s, not its column's %s", got[i], workspaceOf(c))
			}
		case strings.HasSuffix(segment, "_id}"):
			id, err := uuid.Parse(got[i])
			slug, isRow := s.workspaceOfRow(id)
			if err != nil || !isRow || slug != workspaceOf(c) {
				return fmt.Sprintf("%s %s is no row seeded under its column's workspace %s", segment, got[i], workspaceOf(c))
			}
		default:
			return fmt.Sprintf("%s is no target the matrix knows: list %s as not a target, with its reason", segment, pattern)
		}
	}
	return ""
````
````new server/internal/bootstrap/permission_matrix_coverage_test.go
	return found
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
	// workspace; notTarget is exempt with its path listed as not a target.
````
````new server/internal/bootstrap/permission_matrix_coverage_test.go
	// workspace; slugListed is exempt with its {slug} listed as not a target.
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
	notTarget := exemptPublic("getWorkspaceInvitation")
	notTarget.notTargets = map[string]string{checkSlug.Path: "a slug asked about"}
````
````new server/internal/bootstrap/permission_matrix_coverage_test.go
	slugListed := exemptPublic("getWorkspaceInvitation")
	slugListed.notTargets = map[notTarget]string{{checkSlug.Path, "{slug}"}: "a slug asked about"}
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
	answers.notTargets = map[string]string{accept.Path: "account level"}
````
````new server/internal/bootstrap/permission_matrix_coverage_test.go
	answers.notTargets = map[notTarget]string{{accept.Path, "{invitation_id}"}: "account level"}
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		{"a path listed as not a target", append(ops, checkSlug), notTarget, []matrixRow{row, checks}, nil},
		{"a not-target path of no operation", ops, notTarget, []matrixRow{row},
			[]string{"the not-target path /api/v0/workspace-slugs/{slug} is no operation's"}},
````
````new server/internal/bootstrap/permission_matrix_coverage_test.go
		{"a path listed as not a target", append(ops, checkSlug), slugListed, []matrixRow{row, checks}, nil},
		{"a not-target path of no operation", ops, slugListed, []matrixRow{row},
			[]string{"the not-target {slug} of /api/v0/workspace-slugs/{slug} is no parameter of an operation's path"}},
````

`server/internal/bootstrap/permission_matrix_seeded_test.go`（修改，14 处）：

````old server/internal/bootstrap/permission_matrix_seeded_test.go
	"github.com/jackc/pgx/v5/pgxpool"

````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
	"github.com/jackc/pgx/v5/pgxpool"

	projectpg "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	projectdomain "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
// guest and the member later removed; gone with its admin and the member;
// other, whose admin was never a member of acme and where the removed
// member is still active.
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
// guest and the member later removed, and the project level's own accounts
// (matrixAccounts); gone with its admin and the member; other, whose admin
// was never a member of acme and where the removed member is still active.
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
	{"acme", callerRemoved, shared.RoleMember},
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
	{"acme", callerRemoved, shared.RoleMember}, {"acme", callerProjectAdmin, shared.RoleMember},
	{"acme", callerProjectMember, shared.RoleMember}, {"acme", callerMemberAndAdmin, shared.RoleAdmin},
	{"acme", callerGuestOnly, shared.RoleGuest}, {"acme", callerBefore, shared.RoleMember},
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
	{"acme", emailOf(callerDeleted), shared.RoleMember},
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
	{"acme", emailOf(callerDeleted), shared.RoleMember},
}

// matrixProjects are the projects prepareMatrix seeds through the project
// store, each by its workspace's admin: acme's public and private ones,
// and one archived; gone's, deleted with it.
var matrixProjects = []struct {
	key, name, identifier string // key: the workspace's slug / which project
	network               projectdomain.Network
}{
	{"acme/public", "Web", "WEB", projectdomain.NetworkPublic},
	{"acme/private", "Secret", "SEC", projectdomain.NetworkPrivate},
	{"acme/archived", "Old", "OLD", projectdomain.NetworkPublic},
	{"gone/project", "Web", "WEB", projectdomain.NetworkPublic},
}

// matrixProjectMembers are the project memberships prepareMatrix seeds,
// each with its display settings: in acme's public and private projects,
// the project level's members (projectColumns); the removed member, still
// an active member of the public one, so that only his membership of acme
// keeps him out; the member before in the private one, ended; the archived
// project's admin; gone's admin in gone's project.
var matrixProjectMembers = []struct {
	key  string
	c    caller
	role shared.Role
}{
	{"acme/public", callerProjectAdmin, shared.RoleAdmin}, {"acme/public", callerProjectMember, shared.RoleMember},
	{"acme/public", callerGuest, shared.RoleGuest}, {"acme/public", callerMemberAndAdmin, shared.RoleMember},
	{"acme/public", callerRemoved, shared.RoleMember},
	{"acme/private", callerProjectAdmin, shared.RoleAdmin}, {"acme/private", callerProjectMember, shared.RoleMember},
	{"acme/private", callerGuest, shared.RoleGuest}, {"acme/private", callerMemberAndAdmin, shared.RoleMember},
	{"acme/private", callerBefore, shared.RoleMember},
	{"acme/archived", callerProjectAdmin, shared.RoleAdmin},
	{"gone/project", callerDeleted, shared.RoleAdmin},
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
// workspace's slug and the column; and each invitation, by the workspace's
// slug and the address. t is the test that asks for them (in).
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
// workspace's slug and the column; each invitation, by the workspace's
// slug and the address; and each project, by its key. t is the test that
// asks for them (in).
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
	invitations map[string]uuid.UUID
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
	invitations map[string]uuid.UUID
	projects    map[string]uuid.UUID
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
// matrixMemberships and each of matrixInvitations before prepareMatrix
// writes them, so that matrixViolations, without a database, sees the keys
// and the workspaces the cells will.
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
// matrixMemberships, each of matrixInvitations and each of matrixProjects
// before prepareMatrix writes them, so that matrixViolations, without a
// database, sees the keys and the targets the cells will.
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
	s := seeded{workspaces: map[string]uuid.UUID{}, memberships: map[string]uuid.UUID{}, invitations: map[string]uuid.UUID{}}
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
	s := seeded{workspaces: map[string]uuid.UUID{}, memberships: map[string]uuid.UUID{}, invitations: map[string]uuid.UUID{},
		projects: map[string]uuid.UUID{}}
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
		s.invitations[i.slug+"/"+i.email] = uuid.NewV7()
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
		s.invitations[i.slug+"/"+i.email] = uuid.NewV7()
	}
	for _, p := range matrixProjects {
		s.projects[p.key] = uuid.NewV7()
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
		s.t.Fatalf("no invitation of %s to %s is seeded", email, slug)
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
		s.t.Fatalf("no invitation of %s to %s is seeded", email, slug)
	}
	return id
}

// project is the id of the project key; one never seeded fails the test at
// once, as membership's does.
func (s seeded) project(key string) uuid.UUID {
	id, ok := s.projects[key]
	if !ok {
		s.t.Helper()
		s.t.Fatalf("no project %s is seeded", key)
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
}

// invite stores the invitation id of email to the workspace slug, by its
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
}

// matrixAdmins are the creators of the prepared workspaces, their admins.
var matrixAdmins = map[string]caller{"acme": callerAdmin, "gone": callerDeleted, "other": callerNever}

// invite stores the invitation id of email to the workspace slug, by its
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
	s.t.Helper()
	admin := map[string]caller{"acme": callerAdmin, "gone": callerDeleted, "other": callerNever}[slug]
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
	s.t.Helper()
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
		{ID: id, WorkspaceID: s.workspaces[slug], Email: email, Role: role, CreatedBy: s.ids[admin], Now: s.now},
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
		{ID: id, WorkspaceID: s.workspaces[slug], Email: email, Role: role, CreatedBy: s.ids[matrixAdmins[slug]], Now: s.now},
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		s.t.Fatal(err)
	}
}

````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		s.t.Fatal(err)
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

````

`server/internal/bootstrap/permission_matrix_invitations_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_invitations_test.go
// member; acme: its admin, member and guest).
````
````new server/internal/bootstrap/permission_matrix_invitations_test.go
// member; acme: its admin, member and guest, and the project level's five
// accounts).
````

````old server/internal/bootstrap/permission_matrix_invitations_test.go
	want := map[string]int{"other": 3, "acme": 4}[ownInvitation(c)]
````
````new server/internal/bootstrap/permission_matrix_invitations_test.go
	want := map[string]int{"other": 3, "acme": 9}[ownInvitation(c)]
````

`server/internal/bootstrap/permission_matrix_workspace_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_workspace_test.go
// listsTheMembers: acme's four memberships, the removed member's ended; the
````
````new server/internal/bootstrap/permission_matrix_workspace_test.go
// listsTheMembers: acme's nine memberships, the removed member's ended; the
````

````old server/internal/bootstrap/permission_matrix_workspace_test.go
	for _, name := range []string{"admin", "member", "guest", "removed"} {
````
````new server/internal/bootstrap/permission_matrix_workspace_test.go
	for _, name := range []string{"admin", "member", "guest", "removed", "project-admin", "project-member", "project-member-and-workspace-admin",
		"workspace-guest-only", "project-member-before"} {
````

- [ ] **Step 3: 测试、lint**

Run: `go -C server test -count=1 -run 'TestPermissionMatrix$|TestThePermissionMatrixCoversEveryOperation|TestMatrixViolationsCatchesEachGap|TestMatrixViolationsCatchesEachColumnGap|TestEveryColumnCallsAsARegisteredAccount' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/bootstrap/permission_matrix_columns_test.go server/internal/bootstrap/permission_matrix_coverage_test.go server/internal/bootstrap/permission_matrix_invitations_test.go server/internal/bootstrap/permission_matrix_seeded_test.go server/internal/bootstrap/permission_matrix_targets_test.go server/internal/bootstrap/permission_matrix_test.go server/internal/bootstrap/permission_matrix_workspace_test.go
```
```bash
git commit -m "test(M3/P4a): each matrix row has its own columns; the project level's accounts and projects

A row names its columns, the workspace level's six unless it says
otherwise, and every cell it has runs. The project level's twelve
columns (M3 design 9.2, X as its three accounts) and the archived
project's call as eleven registered accounts, each cell aimed at its
column's project, seeded through the project store under ids named
before seeding. A parameter that is no target is listed by path and
parameter, so {identifier} leaves its path's {slug} checked.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 |
|---|---|
| 完整性核对按工作区的列核对每一行；放过别的列的格子；每行的列都是工作区级的 | `TestMatrixViolationsCatchesEachColumnGap` |
| `{project_id}` 指向别的列的项目也通过；按任何准备过的行核对 | `TestMatrixViolationsCatchesEachColumnGap` |
| 列出的参数放过整条路径（裁定 G3 的反例：`{identifier}` 旁边的 `{slug}`）；列出的参数照样报告；不在路径里的参数也能列出 | `TestMatrixViolationsCatchesEachColumnGap`（第二个另有 `TestMatrixViolationsCatchesEachGap`） |
| 以前的成员不注册；项目访客用自己的账户 | `TestEveryColumnCallsAsARegisteredAccount` |
| 只注册工作区列的账户；以前的成员不是 `acme` 的成员 | `TestPermissionMatrix` |

**Done when:** 工作区级的 132 格照旧通过；项目级的列、账户和项目的准备由它们的核对和反例证明；按参数列出的"不是目标"不隐藏同一路径上的别的参数。

---

### Task 7: `createProject` 的用例

**Files:**
- Create: `server/internal/modules/project/app/clock_test.go`、`server/internal/modules/project/app/create_project.go`、`server/internal/modules/project/app/create_project_test.go`、`server/internal/modules/project/app/fakes_create_test.go`、`server/internal/modules/project/app/fakes_test.go`
- Modify: `server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/project/app/ports.go`、`server/internal/modules/project/domain/errors.go`
- Modify（完整内容）: `server/internal/modules/project/domain/actions.go`

**Interfaces:**
- Produces（spec 2.8，M3 设计 3.6 约定二、三，3.17–3.19）：`domain.ActionCreate = "project.create"`（`Actions()` 从这里起列出它）；规则表 `project.create`：工作区的管理员和成员（`LevelWorkspace`）；`domain.ErrWorkspaceNotFound`（`workspace.not_found`）、`LeadNotAllowed()`（422 `project_lead_id` `not_allowed`）；端口 `app.WorkspaceDirectory`（`ShareWorkspaceBySlug`）、`app.WorkspaceMembers`、`app.Workspace`、`app.Clock`；`app.NewCreateProject(CreateProjectDeps{Workspaces, Members, Projects, Auth, Tx, Clock})`、`Execute(ctx, slug, domain.NewProject) (domain.Project, error)`。
- 顺序：`CheckNewProject`（事务之前）→ 读时钟、取新 id（事务之前，裁定 (c) 的写法，第 7 条移交）→ 一个事务：`ShareWorkspaceBySlug`（没有是 `workspace.not_found`）→ 判定 `project.create`（看不到是 `workspace.not_found`，访客是 `forbidden`）→ `ShareMembers(创建者[, 负责人])` → 负责人不是有效的管理员或成员：422 → `CreateProject` → 每个管理员（创建者、与他不同的负责人）：`CreateMember`（角色 20）、`LowestSortOrder`、`CreatePreferences(SortOrderFirst)` → 六个默认状态 → `GetProject(id, 调用者)` 作回答。
- 使用者：Task 8 的 HTTP。

**Tests:**
- `app/create_project_test.go`：`TestCreateProject`（调用记录按 3.6 的顺序；创建者也是负责人时只插一次；时区取请求的或工作区的；答案是存储读回的、不是用例拼的）；`TestCreateProjectStoresTheDefaultStates`；`TestCreateProjectRefuses`（请求的问题在时钟和事务之前；工作区没有或看不到是 404 且在成员行的锁之前；访客 403 且在锁之前；负责人是访客、已结束、不是成员：422，在判定和锁之后）；`TestCreateProjectReturnsEachFailure`（每个端口的失败、提交的失败原样返回，不答任何项目）；`TestCreateProjectAnswersTheStoresConflicts`；`TestCreateProjectWithoutACaller`。
- `app/clock_test.go`：`TestCreatingAProjectReadsTheClockBeforeItsTransaction`（读一次，在事务的第一个调用之前；每一行都是这个时刻）。
- `access/domain/rules_test.go`：`project.create` 的格子；`TestEveryActionHasARuleAndEveryRuleAnAction` 照新的操作名。

- [ ] **Step 1: 操作名、规则、错误**

`server/internal/modules/project/domain/actions.go`（完整内容，19 行）：

````whole server/internal/modules/project/domain/actions.go
// Package domain holds the project module's rules (M3 design 6.3): pure
// functions and values.
package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The project module's actions: the keys of its rows in the access module's
// rule table (M3 design 3.4).
const (
	// ActionCreate is creating a project in a workspace: createProject.
	ActionCreate shared.Action = "project.create"
)

// Actions lists the module's actions. bootstrap's test holds the union of
// every module's Actions equal to the rule table's keys (M3 design 3.4).
// Each operation adds its action here with its row.
func Actions() []shared.Action {
	return []shared.Action{ActionCreate}
}
````

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"workspace_invitation.delete": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
````
````new server/internal/modules/access/domain/rules.go
	"workspace_invitation.delete": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
	// The workspace's admins and members, not its guests (M3 design 9.2;
	// Plane views/project/base.py:257).
	"project.create": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"workspace_invitation.delete":  {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
````
````new server/internal/modules/access/domain/rules_test.go
	"workspace_invitation.delete":  {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
	"project.create":               {allowed, allowed, forbidden, invisible, invisible, invisible, forbidden},
````

`server/internal/modules/project/domain/errors.go`（修改，2 处）：

````old server/internal/modules/project/domain/errors.go
var (
````
````new server/internal/modules/project/domain/errors.go
var (
	// ErrWorkspaceNotFound answers a workspace that does not exist, is
	// deleted, or of which the caller is not an active member: the
	// workspace module's code, which the project's operations under a
	// workspace declare too (M3 design 5.1).
	ErrWorkspaceNotFound = shared.NewError(shared.KindNotFound, "workspace.not_found", "The workspace does not exist, or you are not a member of it.")
````

````old server/internal/modules/project/domain/errors.go
	ErrNameTaken = shared.NewError(shared.KindConflict, "project.name_taken", "A project of the workspace has this name.")
)

````
````new server/internal/modules/project/domain/errors.go
	ErrNameTaken = shared.NewError(shared.KindConflict, "project.name_taken", "A project of the workspace has this name.")
)

// LeadNotAllowed is the 422 of a lead who is not an active admin or member
// of the workspace (M3 design 3.19).
func LeadNotAllowed() error {
	return shared.Invalid(shared.FieldError{Field: "project_lead_id", Code: shared.FieldNotAllowed,
		Message: "must be an active admin or member of the workspace"})
}

````

- [ ] **Step 2: 端口和用例**

`server/internal/modules/project/app/ports.go`（修改，1 处）：

````old server/internal/modules/project/app/ports.go
	"github.com/open-nerve/NerveProject/server/internal/shared"
)
````
````new server/internal/modules/project/app/ports.go
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Clock is the use cases' time.
type Clock interface {
	Now() time.Time
}

// Workspace is a workspace as WorkspaceDirectory finds it: bootstrap
// converts the workspace module's answer into it (M3 design 6.5).
type Workspace struct {
	ID       uuid.UUID
	Timezone string
}

// WorkspaceDirectory is the workspace module's directory
// (workspace.Provide): the undeleted workspace a slug names; found is false
// when there is none (M3 design 6.5).
type WorkspaceDirectory interface {
	// ShareWorkspaceBySlug also locks the workspace's row FOR SHARE until
	// the transaction ctx carries ends: the parent lock of a write that adds
	// a project (M3 design 3.6 convention 2). A workspace deleted while the
	// lock waited is not found.
	ShareWorkspaceBySlug(ctx context.Context, slug string) (w Workspace, found bool, err error)
}

// WorkspaceMembers is the workspace module's lock of the memberships a
// write makes project members from (workspace.Provide; M3 design 3.6
// convention 3).
type WorkspaceMembers interface {
	// ShareMembers locks userIDs' undeleted memberships of workspaceID,
	// active or not, FOR SHARE in id order until the transaction ctx
	// carries ends, and returns the roles of the active ones by account.
	ShareMembers(ctx context.Context, workspaceID uuid.UUID, userIDs []uuid.UUID) (map[uuid.UUID]shared.Role, error)
}

// ProjectCreator is createProject's repository. Each method runs in the
// transaction ctx carries.
type ProjectCreator interface {
	CreateProject(ctx context.Context, p ProjectRow) error
	CreateMember(ctx context.Context, m MemberRow) error
	// LowestSortOrder is the least place of userID's in his sidebar among
	// workspaceID's projects, nil when he has none.
	LowestSortOrder(ctx context.Context, workspaceID, userID uuid.UUID) (*float64, error)
	CreatePreferences(ctx context.Context, p PreferencesRow) error
	CreateStates(ctx context.Context, rows []StateRow) error
	// GetProject is the undeleted project id as userID sees it; found is
	// false when there is none.
	GetProject(ctx context.Context, id, userID uuid.UUID) (p domain.Project, found bool, err error)
}
````

`server/internal/modules/project/app/create_project.go`（新文件，142 行）：

````file server/internal/modules/project/app/create_project.go
package app

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CreateProject creates a project in a workspace: POST
// /api/v0/workspaces/{slug}/projects (M3 design 3.6, 3.17–3.19).
type CreateProject struct {
	d CreateProjectDeps
}

// CreateProjectDeps are the use case's ports.
type CreateProjectDeps struct {
	Workspaces WorkspaceDirectory
	Members    WorkspaceMembers
	Projects   ProjectCreator
	Auth       shared.Authorizer
	Tx         shared.TxManager
	Clock      Clock
}

// NewCreateProject returns the use case.
func NewCreateProject(d CreateProjectDeps) *CreateProject {
	return &CreateProject{d: d}
}

// Execute checks in (domain.CheckNewProject), then, in one transaction, in
// the order of M3 design 3.6: the workspace FOR SHARE; the decision on
// project.create; the creator's and the lead's memberships of the workspace
// FOR SHARE in id order (convention 3); the lead's check, after the
// decision, so that a caller who may not create learns nothing of the lead
// (422 project_lead_id not_allowed unless the lead is an active admin or
// member); then the project, the creator's and the lead's memberships as
// its admins, each one's display settings first in his sidebar
// (SortOrderFirst), and the six default states. The project's time zone is
// the one given, or the workspace's. The rows are new, so the clock is read
// once, before the transaction: no row it reads under the locks has a time
// the new rows must follow (P3's ruling (c), as createWorkspaceInvitations).
// The answer is the project as stored, as the caller sees it.
func (u *CreateProject) Execute(ctx context.Context, slug string, in domain.NewProject) (domain.Project, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Project{}, err
	}
	in, err = domain.CheckNewProject(in)
	if err != nil {
		return domain.Project{}, err
	}
	now := u.d.Clock.Now()
	id := uuid.NewV7()
	var created domain.Project
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		ws, found, err := u.d.Workspaces.ShareWorkspaceBySlug(ctx, slug)
		switch {
		case err != nil:
			return err
		case !found:
			return domain.ErrWorkspaceNotFound
		}
		_, err = u.d.Auth.Authorize(ctx, actor, domain.ActionCreate, shared.Target{WorkspaceID: ws.ID})
		switch {
		case errors.Is(err, shared.ErrNotVisible):
			return domain.ErrWorkspaceNotFound
		case err != nil:
			return err
		}
		admins := []uuid.UUID{actor.UserID}
		if in.LeadID != nil && *in.LeadID != actor.UserID {
			admins = append(admins, *in.LeadID)
		}
		roles, err := u.d.Members.ShareMembers(ctx, ws.ID, admins)
		if err != nil {
			return err
		}
		if in.LeadID != nil {
			if role, active := roles[*in.LeadID]; !active || !domain.CanLead(role) {
				return domain.LeadNotAllowed()
			}
		}
		if err := u.insert(ctx, u.row(id, ws, in, actor.UserID, now), admins); err != nil {
			return err
		}
		created, found, err = u.d.Projects.GetProject(ctx, id, actor.UserID)
		switch {
		case err != nil:
			return err
		case !found:
			return fmt.Errorf("create project: project %s is not there after its insert", id)
		}
		return nil
	})
	if err != nil {
		return domain.Project{}, err
	}
	return created, nil
}

// row is the project to insert: in's values, in the workspace ws, its time
// zone the workspace's unless in gives one.
func (u *CreateProject) row(id uuid.UUID, ws Workspace, in domain.NewProject, by uuid.UUID, now time.Time) ProjectRow {
	timezone := ws.Timezone
	if in.Timezone != nil {
		timezone = *in.Timezone
	}
	return ProjectRow{ID: id, WorkspaceID: ws.ID, Name: in.Name, Description: in.Description, Identifier: in.Identifier, Network: *in.Network,
		LeadID: in.LeadID, LogoProps: in.LogoProps, Timezone: timezone, CreatedBy: by, Now: now}
}

// insert writes p, each of admins its admin with his display settings, and
// the default states.
func (u *CreateProject) insert(ctx context.Context, p ProjectRow, admins []uuid.UUID) error {
	if err := u.d.Projects.CreateProject(ctx, p); err != nil {
		return err
	}
	for _, user := range admins {
		if err := u.d.Projects.CreateMember(ctx, MemberRow{ID: uuid.NewV7(), WorkspaceID: p.WorkspaceID, ProjectID: p.ID, MemberID: user,
			Role: shared.RoleAdmin, CreatedBy: p.CreatedBy, Now: p.Now}); err != nil {
			return err
		}
		lowest, err := u.d.Projects.LowestSortOrder(ctx, p.WorkspaceID, user)
		if err != nil {
			return err
		}
		if err := u.d.Projects.CreatePreferences(ctx, PreferencesRow{ID: uuid.NewV7(), WorkspaceID: p.WorkspaceID, ProjectID: p.ID, UserID: user,
			SortOrder: domain.SortOrderFirst(lowest), CreatedBy: p.CreatedBy, Now: p.Now}); err != nil {
			return err
		}
	}
	var states []StateRow
	for _, s := range domain.DefaultStates() {
		states = append(states, StateRow{ID: uuid.NewV7(), WorkspaceID: p.WorkspaceID, ProjectID: p.ID, State: s, CreatedBy: p.CreatedBy, Now: p.Now})
	}
	return u.d.Projects.CreateStates(ctx, states)
}
````

- [ ] **Step 3: 假实现和测试**

`server/internal/modules/project/app/fakes_test.go`（新文件，93 行）：

````file server/internal/modules/project/app/fakes_test.go
package app_test

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The use cases' clock stands at clockNow, finer than a stored time: the
// fakes store a time as PostgreSQL's timestamptz does, to the microsecond,
// and answer the row as stored. A use case that answered its own instant
// instead of the row as stored would answer clockNow.
var (
	clockNow = time.Date(2026, 10, 1, 10, 0, 0, 123456789, time.UTC)
	now      = clockNow.Truncate(time.Microsecond)
)

// clockAt stands at an instant; with a log, each read is logged as "Now"
// among the fakes' calls.
type clockAt struct {
	at  time.Time
	log *callLog
}

func (c clockAt) Now() time.Time {
	if c.log != nil {
		c.log.calls = append(c.log.calls, "Now")
	}
	return c.at
}

// fakeTx runs fn in a context marked as inside the transaction; the fakes
// record whether each call happened there. commitErr, when set, is the
// commit failing after fn succeeded.
type fakeTx struct {
	log       *callLog
	commitErr error
}

type inTxKey struct{}

func (f *fakeTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if f.log != nil {
		f.log.calls = append(f.log.calls, "Begin")
	}
	if err := fn(context.WithValue(ctx, inTxKey{}, true)); err != nil {
		return err
	}
	return f.commitErr
}

// callLog records the calls of the fakes that share it, in order, each with
// its arguments, and " outside tx" when it ran outside a transaction.
type callLog struct{ calls []string }

func (l *callLog) add(ctx context.Context, format string, args ...any) {
	call := fmt.Sprintf(format, args...)
	if ctx.Value(inTxKey{}) != true {
		call += " outside tx"
	}
	l.calls = append(l.calls, call)
}

// grantKey is one (user, workspace) pair of fakeAuthorizer.
type grantKey struct{ user, workspace uuid.UUID }

// fakeAuthorizer answers each (user, workspace) pair its grant, or its
// error, and ErrNotVisible for any other; it logs every call.
type fakeAuthorizer struct {
	log    *callLog
	grants map[grantKey]shared.Grant
	errs   map[grantKey]error
}

func (f *fakeAuthorizer) Authorize(ctx context.Context, actor shared.Actor, action shared.Action, t shared.Target) (shared.Grant, error) {
	f.log.add(ctx, "Authorize %s %s on %s/%s", actor.UserID, action, t.WorkspaceID, t.ProjectID)
	key := grantKey{actor.UserID, t.WorkspaceID}
	if err := f.errs[key]; err != nil {
		return shared.Grant{}, err
	}
	if g, ok := f.grants[key]; ok {
		return g, nil
	}
	return shared.Grant{}, shared.ErrNotVisible
}

// as is a context carrying the caller id.
func as(id uuid.UUID) context.Context {
	return shared.WithActor(context.Background(), shared.Actor{UserID: id})
}
````

`server/internal/modules/project/app/fakes_create_test.go`（新文件，141 行）：

````file server/internal/modules/project/app/fakes_create_test.go
package app_test

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeDirectory finds the workspaces it holds by slug, logs each call and
// fails with err.
type fakeDirectory struct {
	log        *callLog
	workspaces map[string]app.Workspace
	err        error
}

func (f *fakeDirectory) ShareWorkspaceBySlug(ctx context.Context, slug string) (app.Workspace, bool, error) {
	f.log.add(ctx, "ShareWorkspaceBySlug %s", slug)
	if f.err != nil {
		return app.Workspace{}, false, f.err
	}
	w, ok := f.workspaces[slug]
	return w, ok, nil
}

// fakeMembers answers the active members' roles it holds by workspace,
// only of the accounts asked for; it logs each call and fails with err.
type fakeMembers struct {
	log   *callLog
	roles map[uuid.UUID]map[uuid.UUID]shared.Role // workspace → account → role
	err   error
}

func (f *fakeMembers) ShareMembers(ctx context.Context, workspaceID uuid.UUID, userIDs []uuid.UUID) (map[uuid.UUID]shared.Role, error) {
	f.log.add(ctx, "ShareMembers %s %v", workspaceID, userIDs)
	if f.err != nil {
		return nil, f.err
	}
	out := map[uuid.UUID]shared.Role{}
	for _, id := range userIDs {
		if role, ok := f.roles[workspaceID][id]; ok {
			out[id] = role
		}
	}
	return out, nil
}

// fakeProjects is createProject's repository: it logs each call with its
// arguments, keeps what it is given, and answers GetProject from it, the
// times as stored (to the microsecond); errs fails a method by its name.
type fakeProjects struct {
	log     *callLog
	lowest  map[uuid.UUID]*float64 // by account
	project *app.ProjectRow
	members []app.MemberRow
	prefs   []app.PreferencesRow
	states  []app.StateRow
	errs    map[string]error
	missing bool // GetProject finds nothing
}

func (f *fakeProjects) fail(name string) error {
	if err := f.errs[name]; err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func (f *fakeProjects) CreateProject(ctx context.Context, p app.ProjectRow) error {
	logo, _ := json.Marshal(p.LogoProps)
	f.log.add(ctx, "CreateProject %s in %s %q %q %q network %d lead %v logo %s timezone %s by %s at %s", p.ID, p.WorkspaceID, p.Name,
		p.Identifier, p.Description, p.Network, lead(p.LeadID), logo, p.Timezone, p.CreatedBy, p.Now.Format(timeFormat))
	f.project = &p
	return f.fail("CreateProject")
}

func (f *fakeProjects) CreateMember(ctx context.Context, m app.MemberRow) error {
	f.log.add(ctx, "CreateMember %s in %s/%s as %d by %s at %s", m.MemberID, m.WorkspaceID, m.ProjectID, m.Role, m.CreatedBy, m.Now.Format(timeFormat))
	f.members = append(f.members, m)
	return f.fail("CreateMember")
}

func (f *fakeProjects) LowestSortOrder(ctx context.Context, workspaceID, userID uuid.UUID) (*float64, error) {
	f.log.add(ctx, "LowestSortOrder %s %s", workspaceID, userID)
	return f.lowest[userID], f.fail("LowestSortOrder")
}

func (f *fakeProjects) CreatePreferences(ctx context.Context, p app.PreferencesRow) error {
	f.log.add(ctx, "CreatePreferences %s in %s/%s at %v by %s at %s", p.UserID, p.WorkspaceID, p.ProjectID, p.SortOrder, p.CreatedBy,
		p.Now.Format(timeFormat))
	f.prefs = append(f.prefs, p)
	return f.fail("CreatePreferences")
}

func (f *fakeProjects) CreateStates(ctx context.Context, rows []app.StateRow) error {
	var names []string
	for _, r := range rows {
		names = append(names, fmt.Sprintf("%s/%s %s by %s at %s", r.WorkspaceID, r.ProjectID, r.State.Name, r.CreatedBy, r.Now.Format(timeFormat)))
	}
	f.log.add(ctx, "CreateStates %q", names)
	f.states = append(f.states, rows...)
	return f.fail("CreateStates")
}

func (f *fakeProjects) GetProject(ctx context.Context, id, userID uuid.UUID) (domain.Project, bool, error) {
	f.log.add(ctx, "GetProject %s for %s", id, userID)
	if err := f.fail("GetProject"); err != nil || f.missing || f.project == nil || f.project.ID != id {
		return domain.Project{}, false, err
	}
	p := domain.Project{ID: f.project.ID, WorkspaceID: f.project.WorkspaceID, Name: f.project.Name, Description: f.project.Description,
		Identifier: f.project.Identifier, Network: f.project.Network, LeadID: f.project.LeadID, LogoProps: f.project.LogoProps,
		Timezone: f.project.Timezone, CreatedAt: f.project.Now.Truncate(time.Microsecond), UpdatedAt: f.project.Now.Truncate(time.Microsecond)}
	for _, m := range f.members {
		p.MemberIDs = append(p.MemberIDs, m.MemberID)
		if m.MemberID == userID {
			role := m.Role
			p.MemberRole = &role
		}
	}
	if i := slices.IndexFunc(f.prefs, func(r app.PreferencesRow) bool { return r.UserID == userID }); i >= 0 {
		p.SortOrder = &f.prefs[i].SortOrder
	}
	return p, true, nil
}

const timeFormat = "2006-01-02T15:04:05.999999999"

// lead is the lead's id, or <nil>.
func lead(id *uuid.UUID) string {
	if id == nil {
		return "<nil>"
	}
	return id.String()
}
````

`server/internal/modules/project/app/create_project_test.go`（新文件，290 行）：

````file server/internal/modules/project/app/create_project_test.go
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

// The accounts and workspaces of createProject's tests: alice is acme's
// member, bob its admin, carol its guest, dave not an active member; erin
// is beta's admin.
var (
	alice, bob, carol, dave, erin = uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	acme                          = app.Workspace{ID: uuid.NewV7(), Timezone: "Asia/Shanghai"}
	beta                          = app.Workspace{ID: uuid.NewV7(), Timezone: "UTC"}
)

// createFixture is CreateProject over fakes sharing one log.
type createFixture struct {
	log        *callLog
	tx         *fakeTx
	workspaces *fakeDirectory
	members    *fakeMembers
	projects   *fakeProjects
	auth       *fakeAuthorizer
}

func newCreate() (*app.CreateProject, *createFixture) {
	log := &callLog{}
	f := &createFixture{log: log, tx: &fakeTx{log: log},
		workspaces: &fakeDirectory{log: log, workspaces: map[string]app.Workspace{"acme": acme, "beta": beta}},
		members: &fakeMembers{log: log, roles: map[uuid.UUID]map[uuid.UUID]shared.Role{
			acme.ID: {alice: shared.RoleMember, bob: shared.RoleAdmin, carol: shared.RoleGuest},
			beta.ID: {erin: shared.RoleAdmin, dave: shared.RoleMember},
		}},
		projects: &fakeProjects{log: log, lowest: map[uuid.UUID]*float64{bob: ptr(100.0)}},
		auth: &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{
			{alice, acme.ID}: {WorkspaceRole: shared.RoleMember}, {bob, acme.ID}: {WorkspaceRole: shared.RoleAdmin},
			{erin, beta.ID}: {WorkspaceRole: shared.RoleAdmin},
		}, errs: map[grantKey]error{{carol, acme.ID}: shared.Forbidden()}},
	}
	return app.NewCreateProject(app.CreateProjectDeps{Workspaces: f.workspaces, Members: f.members, Projects: f.projects, Auth: f.auth, Tx: f.tx,
		Clock: clockAt{clockNow, log}}), f
}

func ptr[T any](v T) *T { return &v }

// created are the calls of a creation of the project id in w by user, lead
// the other admin or none: its locks and decision, then the inserts.
func created(id uuid.UUID, w app.Workspace, user uuid.UUID, lead *uuid.UUID, row string, sortOrders map[uuid.UUID]string) []string {
	at := clockNow.Format(timeFormat)
	admins := []uuid.UUID{user}
	if lead != nil && *lead != user {
		admins = append(admins, *lead)
	}
	calls := []string{"Now", "Begin", "ShareWorkspaceBySlug " + slugOf(w),
		fmt.Sprintf("Authorize %s project.create on %s/%s", user, w.ID, uuid.UUID{}), fmt.Sprintf("ShareMembers %s %v", w.ID, admins),
		"CreateProject " + id.String() + " in " + w.ID.String() + " " + row + " by " + user.String() + " at " + at}
	for _, a := range admins {
		calls = append(calls, fmt.Sprintf("CreateMember %s in %s/%s as 20 by %s at %s", a, w.ID, id, user, at),
			fmt.Sprintf("LowestSortOrder %s %s", w.ID, a),
			fmt.Sprintf("CreatePreferences %s in %s/%s at %s by %s at %s", a, w.ID, id, sortOrders[a], user, at))
	}
	var states []string
	for _, s := range []string{"Backlog", "Todo", "In Progress", "Done", "Cancelled", "Triage"} {
		states = append(states, fmt.Sprintf("%s/%s %s by %s at %s", w.ID, id, s, user, at))
	}
	return append(calls, fmt.Sprintf("CreateStates %q", states), fmt.Sprintf("GetProject %s for %s", id, user))
}

func slugOf(w app.Workspace) string {
	if w == acme {
		return "acme"
	}
	return "beta"
}

// createdID is the id of the project the fixture's store was given.
func (f *createFixture) createdID(t *testing.T) uuid.UUID {
	t.Helper()
	if f.projects.project == nil {
		t.Fatal("no project was inserted")
	}
	return f.projects.project.ID
}

// CreateProject checks the request, then, in one transaction and in the
// order of M3 design 3.6, locks the workspace, decides, locks the creator's
// and the lead's memberships, and inserts the project, each of them its
// admin with his display settings first in his sidebar, and the six
// default states, by the caller at the clock's time read once before the
// transaction; the answer is the project as stored, read back as the
// caller sees it. The project takes the workspace's time zone unless the
// request gives one.
func TestCreateProject(t *testing.T) {
	tests := []struct {
		name       string
		user       uuid.UUID
		slug       string
		in         domain.NewProject
		w          app.Workspace
		lead       *uuid.UUID
		row        string
		sortOrders map[uuid.UUID]string
	}{
		{"a member, the admin as the lead", alice, "acme",
			domain.NewProject{Name: "Web", Identifier: "web", Description: "The app", LeadID: &bob,
				LogoProps: domain.LogoProps{InUse: ptr("emoji"), Emoji: &domain.Emoji{Value: ptr("128640")}}}, acme, &bob,
			`"Web" "WEB" "The app" network 2 lead ` + bob.String() + ` logo {"in_use":"emoji","emoji":{"value":"128640"}} timezone Asia/Shanghai`,
			map[uuid.UUID]string{alice: "65535", bob: "-9900"}},
		{"the admin, himself the lead, private, in his time zone", bob, "acme",
			domain.NewProject{Name: "Ops", Identifier: "OPS", LeadID: &bob, Network: ptr(domain.NetworkPrivate), Timezone: ptr("Europe/Paris")},
			acme, &bob, `"Ops" "OPS" "" network 0 lead ` + bob.String() + ` logo {} timezone Europe/Paris`, map[uuid.UUID]string{bob: "-9900"}},
		{"another workspace's admin, no lead", erin, "beta", domain.NewProject{Name: "Web", Identifier: "WEB"}, beta, nil,
			`"Web" "WEB" "" network 2 lead <nil> logo {} timezone UTC`, map[uuid.UUID]string{erin: "65535"}},
		{"a member of another workspace as the lead", erin, "beta", domain.NewProject{Name: "Web", Identifier: "WEB", LeadID: &dave}, beta, &dave,
			`"Web" "WEB" "" network 2 lead ` + dave.String() + ` logo {} timezone UTC`, map[uuid.UUID]string{erin: "65535", dave: "65535"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newCreate()

			got, err := uc.Execute(as(tt.user), tt.slug, tt.in)

			if err != nil {
				t.Fatalf("Execute() = %v", err)
			}
			id := f.createdID(t)
			if want := created(id, tt.w, tt.user, tt.lead, tt.row, tt.sortOrders); !slices.Equal(f.log.calls, want) {
				t.Errorf("calls\n%q\nwant\n%q", f.log.calls, want)
			}
			if got.ID != id || !got.CreatedAt.Equal(now) || got.CreatedAt != now || got.MemberRole == nil || *got.MemberRole != shared.RoleAdmin {
				t.Errorf("Execute() = %+v; want the project %s as stored at %v, the caller its admin", got, id, now)
			}
		})
	}
}

// The six default states are stored as the domain gives them, each its own
// row of the project.
func TestCreateProjectStoresTheDefaultStates(t *testing.T) {
	uc, f := newCreate()
	if _, err := uc.Execute(as(alice), "acme", domain.NewProject{Name: "Web", Identifier: "WEB"}); err != nil {
		t.Fatal(err)
	}
	var got []domain.NewState
	ids := map[uuid.UUID]bool{}
	for _, r := range f.projects.states {
		got = append(got, r.State)
		ids[r.ID] = true
	}
	if !slices.Equal(got, domain.DefaultStates()) || len(ids) != 6 {
		t.Errorf("the states %+v, %d ids; want the six defaults, each its own id", got, len(ids))
	}
}

// Refusals, each in its place, and nothing written:
//   - the request's values, before the clock or the transaction;
//   - a workspace not there, or not visible: 404 workspace.not_found,
//     before the members' lock;
//   - a guest: the Authorizer's 403, before the members' lock;
//   - a lead who is not an active admin or member of the workspace: 422,
//     after the decision and the members' lock.
func TestCreateProjectRefuses(t *testing.T) {
	leadIs := func(lead uuid.UUID) domain.NewProject {
		return domain.NewProject{Name: "Web", Identifier: "WEB", LeadID: &lead}
	}
	decided := func(user uuid.UUID, w app.Workspace, slug string) []string {
		return []string{"Now", "Begin", "ShareWorkspaceBySlug " + slug, fmt.Sprintf("Authorize %s project.create on %s/%s", user, w.ID, uuid.UUID{})}
	}
	tests := []struct {
		name  string
		user  uuid.UUID
		slug  string
		in    domain.NewProject
		want  error
		calls []string
	}{
		{"an invalid identifier", alice, "acme", domain.NewProject{Name: "Web", Identifier: "WEB-2"},
			shared.Invalid(shared.FieldError{Field: "identifier", Code: "invalid_format"}), nil},
		{"no workspace", alice, "gone", domain.NewProject{Name: "Web", Identifier: "WEB"}, domain.ErrWorkspaceNotFound,
			[]string{"Now", "Begin", "ShareWorkspaceBySlug gone"}},
		{"a workspace he is not in", dave, "acme", domain.NewProject{Name: "Web", Identifier: "WEB"}, domain.ErrWorkspaceNotFound,
			decided(dave, acme, "acme")},
		{"a guest", carol, "acme", domain.NewProject{Name: "Web", Identifier: "WEB"}, shared.Forbidden(), decided(carol, acme, "acme")},
		{"a guest as the lead", alice, "acme", leadIs(carol), domain.LeadNotAllowed(),
			append(decided(alice, acme, "acme"), fmt.Sprintf("ShareMembers %s %v", acme.ID, []uuid.UUID{alice, carol}))},
		{"no active member as the lead", alice, "acme", leadIs(dave), domain.LeadNotAllowed(),
			append(decided(alice, acme, "acme"), fmt.Sprintf("ShareMembers %s %v", acme.ID, []uuid.UUID{alice, dave}))},
		{"another workspace's admin as the lead", alice, "acme", leadIs(erin), domain.LeadNotAllowed(),
			append(decided(alice, acme, "acme"), fmt.Sprintf("ShareMembers %s %v", acme.ID, []uuid.UUID{alice, erin}))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newCreate()

			got, err := uc.Execute(as(tt.user), tt.slug, tt.in)

			if !sameError(err, tt.want) || got.ID != (uuid.UUID{}) {
				t.Errorf("Execute() = %+v, %v; want %v", got, err, tt.want)
			}
			if !slices.Equal(f.log.calls, tt.calls) || f.projects.project != nil {
				t.Errorf("calls %q, a project inserted %v; want %q and none", f.log.calls, f.projects.project != nil, tt.calls)
			}
		})
	}
}

// sameError reports whether err is want: the same code, and for a 422 the
// same fields, their messages left out.
func sameError(err, want error) bool {
	var e, w *shared.Error
	if !errors.As(err, &e) || !errors.As(want, &w) || e.Kind != w.Kind || e.Code != w.Code || len(e.Fields) != len(w.Fields) {
		return false
	}
	for i := range e.Fields {
		if e.Fields[i].Field != w.Fields[i].Field || e.Fields[i].Code != w.Fields[i].Code {
			return false
		}
	}
	return true
}

// Every port's failure comes back as itself, the transaction's commit's
// too, and nothing is answered.
func TestCreateProjectReturnsEachFailure(t *testing.T) {
	failure := errors.New("disk full")
	type failing struct {
		name string
		fail func(f *createFixture)
	}
	tests := []failing{
		{"the workspace's lock", func(f *createFixture) { f.workspaces.err = failure }},
		{"the decision", func(f *createFixture) { f.auth.errs = map[grantKey]error{{alice, acme.ID}: failure} }},
		{"the members' lock", func(f *createFixture) { f.members.err = failure }},
		{"the commit", func(f *createFixture) { f.tx.commitErr = failure }},
	}
	for _, method := range []string{"CreateProject", "CreateMember", "LowestSortOrder", "CreatePreferences", "CreateStates", "GetProject"} {
		tests = append(tests, failing{method, func(f *createFixture) { f.projects.errs = map[string]error{method: failure} }})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newCreate()
			tt.fail(f)
			got, err := uc.Execute(as(alice), "acme", domain.NewProject{Name: "Web", Identifier: "WEB", LeadID: &bob})
			if !errors.Is(err, failure) || got.ID != (uuid.UUID{}) {
				t.Errorf("Execute() = %+v, %v; want %v", got, err, failure)
			}
		})
	}
	// A project the store cannot read back is an internal error, not an
	// answer.
	uc, f := newCreate()
	f.projects.missing = true
	var e *shared.Error
	if got, err := uc.Execute(as(alice), "acme", domain.NewProject{Name: "Web", Identifier: "WEB"}); err == nil || errors.As(err, &e) ||
		got.ID != (uuid.UUID{}) {
		t.Errorf("Execute() with the project gone = %+v, %v; want an internal error", got, err)
	}
}

// The store's conflicts come back as themselves: 409 of the identifier or
// the name.
func TestCreateProjectAnswersTheStoresConflicts(t *testing.T) {
	for _, conflict := range []error{domain.ErrIdentifierTaken, domain.ErrNameTaken} {
		uc, f := newCreate()
		f.projects.errs = map[string]error{"CreateProject": conflict}
		if _, err := uc.Execute(as(alice), "acme", domain.NewProject{Name: "Web", Identifier: "WEB"}); !errors.Is(err, conflict) {
			t.Errorf("Execute() = %v, want %v", err, conflict)
		}
	}
}

// Without a caller nothing is read: 401, from the use case on its own.
func TestCreateProjectWithoutACaller(t *testing.T) {
	uc, f := newCreate()
	if _, err := uc.Execute(context.Background(), "acme", domain.NewProject{Name: "Web", Identifier: "WEB"}); !errors.Is(err, shared.Unauthenticated()) {
		t.Errorf("Execute() without an actor = %v, want 401 unauthorized", err)
	}
	if len(f.log.calls) != 0 {
		t.Errorf("calls = %q, want none", f.log.calls)
	}
}
````

`server/internal/modules/project/app/clock_test.go`（新文件，36 行）：

````file server/internal/modules/project/app/clock_test.go
package app_test

import (
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// createProject only inserts, so it reads the clock once, before its
// transaction and its locks (P3's ruling (c)): no row it reads under them
// has a time its rows must follow. Every row it writes has that one time.
func TestCreatingAProjectReadsTheClockBeforeItsTransaction(t *testing.T) {
	uc, f := newCreate()
	if _, err := uc.Execute(as(alice), "acme", domain.NewProject{Name: "Web", Identifier: "WEB", LeadID: &bob}); err != nil {
		t.Fatal(err)
	}
	if len(f.log.calls) < 2 || f.log.calls[0] != "Now" || f.log.calls[1] != "Begin" || slices.Contains(f.log.calls[1:], "Now") {
		t.Errorf("calls %q; want the clock read once, first, before the transaction", f.log.calls)
	}
	for _, r := range f.projects.members {
		if r.Now != clockNow {
			t.Errorf("a membership at %v, want %v", r.Now, clockNow)
		}
	}
	for _, r := range f.projects.prefs {
		if r.Now != clockNow {
			t.Errorf("display settings at %v, want %v", r.Now, clockNow)
		}
	}
	for _, r := range f.projects.states {
		if r.Now != clockNow {
			t.Errorf("a state at %v, want %v", r.Now, clockNow)
		}
	}
}
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
git add server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/project/app/clock_test.go server/internal/modules/project/app/create_project.go server/internal/modules/project/app/create_project_test.go server/internal/modules/project/app/fakes_create_test.go server/internal/modules/project/app/fakes_test.go server/internal/modules/project/app/ports.go server/internal/modules/project/domain/actions.go server/internal/modules/project/domain/errors.go
```
```bash
git commit -m "feat(M3/P4a): createProject's use case

The request is checked first, the clock read and the id drawn before
the transaction; then, in the order of M3 design 3.6: the workspace
FOR SHARE, the decision on project.create, the creator's and the lead's
memberships FOR SHARE, the lead's check after the decision, the project,
each of its admins with his display settings first in his sidebar, and
the six default states. The answer is the project read back as the
caller sees it. project.create is the workspace's admins' and members'.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 |
|---|---|
| 没有调用者照常进行；判定别的操作；看不到照原样答（不是 `workspace.not_found`）；忽略判定的拒绝 | `TestCreateProjectWithoutACaller`；`TestCreateProject`、`TestCreateProjectRefuses`（后两个另有 `TestCreateProjectReturnsEachFailure`） |
| 请求不检查；时钟在检查之前读；时钟在锁之下读（第 7 条移交） | `TestCreateProject`、`TestCreateProjectRefuses`；`TestCreateProjectRefuses`；`TestCreatingAProjectReadsTheClockBeforeItsTransaction` |
| 成员行的锁和负责人的检查在判定之前（约定三）；负责人的成员行不锁 | `TestCreateProject`、`TestCreateProjectRefuses`；`TestCreateProject` |
| 负责人是任何有效的角色；不检查负责人 | `TestCreateProjectRefuses` |
| 创建者也是负责人时插两次；忽略请求的时区；不给时区时用 UTC | `TestCreateProject` |
| 管理员存成成员；成员关系由各自的账户创建；按创建者的侧边栏放每个管理员；放在默认的位置；不建状态；每个状态用项目的 id | `TestCreateProject`（后两个另有 `TestCreateProjectStoresTheDefaultStates`） |
| 答用例拼的值；答"谁都不是"的视角；在事务之外读回答 | `TestCreateProject` |
| 工作区的锁、成员行的锁、项目、成员关系、位置、显示设置、回答的读、提交，每一个的失败被吞掉或答成别的 | `TestCreateProjectReturnsEachFailure`（项目的另有 `TestCreateProjectAnswersTheStoresConflicts`） |
| 没有工作区照常建 | `TestCreateProjectRefuses` |
| 规则让访客建；只让管理员建 | `TestEveryRuleDecidesItsCells`（Task 8 起另有矩阵） |
| `Actions()` 不列 `project.create` | `TestEveryActionHasARuleAndEveryRuleAnAction` |

**Done when:** 用例的顺序、每种拒绝的位置、每个端口的失败、时钟的一次读由假实现的测试核对，假实现按参数回答、答存储的时刻而不是时钟的。

---

### Task 8: `createProject` 的接口：`project.yaml`、HTTP、注册与接线；每个模块的 HTTP 测试经 `apitest.Main`

**Files:**
- Create: `api/modules/project.yaml`、`server/internal/bootstrap/permission_matrix_project_test.go`、`server/internal/bootstrap/project_test.go`、`server/internal/bootstrap/project_wiring_test.go`、`server/internal/modules/project/adapter/http/gen/oapi-codegen.yaml`、`server/internal/modules/project/adapter/http/handler.go`、`server/internal/modules/project/adapter/http/handler_test.go`、`server/internal/modules/project/adapter/http/projects.go`、`server/internal/modules/project/adapter/http/projects_test.go`、`server/internal/platform/httpserver/apitest/main_callers_test.go`
- Modify: `api/openapi.yaml`、`server/internal/bootstrap/app.go`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/bootstrap/ports.go`、`server/internal/bootstrap/ports_test.go`、`server/internal/modules/project/module.go`、`web/apps/web/helpers/authentication.helper.ts`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/project/adapter/http/gen/bodyshape.gen.go`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.8、2.13，M3 设计 5.1–5.3、6.6、9.4；P1 review 第 6 节 M6）：`POST /api/v0/workspaces/{slug}/projects`，`ProjectCreate{name, identifier, description?, network?, project_lead_id?, logo_props?, timezone?}`（`additionalProperties: false`，`LogoProps` 的三层都是封闭的结构），201 `Project`；码 `[validation_failed, workspace.not_found, forbidden, project.identifier_taken, project.name_taken]`；`Project` 的 23 个字段都是必有的，`cover_image_url` 总是 `null`，`member_ids` 是数组。新码 `project.identifier_taken`、`project.name_taken`（409）的文案。
- `project.Deps{Pool, Tx, Clock, Authorizer, Workspaces, Members}`、`(*Module).Register`；`bootstrap/ports.go` 的 `projectWorkspaces`（把 `workspace.DirectoryEntry` 转成 `project.Workspace`）；`app.go` 把它和 `workspacePorts.WorkspaceMembers` 交给 `project.New`，注册 `proj.Register`。
- `apitest`：`TestEveryModuleRunsItsHTTPTestsThroughMain` 对 `api/modules/` 下的每个模块解析它的 `adapter/http/*_test.go`，要求有 `TestMain`，函数体恰好是 `apitest.Main(m, "<模块>")`，用它自己的 `*testing.M`。
- 矩阵：`createProject` 两行（12 格）：`inWorkspace(201, 201, 403)` 和负责人不是成员时 `inWorkspace(422, 422, 403)`（约定三：访客照样是 403，得不到负责人的任何信息）。

**Tests:**
- `adapter/http/projects_test.go`：`TestCreateProjectAnswers201`（请求体的每个字段交给用例，不给的是 `nil` 或空；201 答每个字段，空的是 `null`、`[]`、`{}`）；`TestCreateProjectRefusals`（契约声明的每个码，含 `workspace.not_found`，9.4）；`TestCreateProjectHoldsTheLogoToItsStructure`（任何一层多出的键、类型不对的值在用例之前 400；类型对但不是两个值之一的 `in_use` 由领域拒绝）。`handler_test.go` 的 `TestMain` 经 `apitest.Main(m, "project")`。
- `apitest/main_callers_test.go`：`TestEveryModuleRunsItsHTTPTestsThroughMain`、`TestMainViolationsCatchesEachGap`（没有 `TestMain`、普通的 `TestMain`、别的模块名、函数体多做别的事、传别的 `*testing.M`，五个反例；照做的包通过）。
- `bootstrap/project_test.go`：`TestCreatingAProject`（组合出的 app：alice 建上海时区的 `acme`，邀请 bob 加入，以他为负责人建项目；答案、四张表的行；同一个标识的别的大小写、同一个名称各答自己的 409，什么都不写）；`ports_test.go`：`TestProjectWorkspacesConvertsWorkspacesAnswer`。
- `bootstrap/project_wiring_test.go`：`TestCreateProjectRunsOnTheWiredClockAndTransaction`（`project.New` 接的时钟和事务：项目的 `created_at` 在请求的前后之间；状态的每次插入都失败时建项目答 500，四张表的行数不变）。

- [ ] **Step 1: 接口描述**

`api/modules/project.yaml`（新文件，240 行）：

````file api/modules/project.yaml
# project 模块的接口（M3 设计 5.1、5.2）。生成的 Go 代码在
# server/internal/modules/project/adapter/http/gen（make gen-go）。
openapi: 3.1.0
info:
  title: Nerve project API
  version: v0
paths:
  /api/v0/workspaces/{slug}/projects:
    parameters:
      - $ref: '#/components/parameters/Slug'
    post:
      operationId: createProject
      tags: [project]
      summary: Create a project in a workspace
      description: >-
        For the workspace's admins and members. The project is created with
        the caller, and the lead when one is given, as its admins, the
        project first in each one's sidebar, and with its six states:
        Backlog, the default, Todo, In Progress, Done, Cancelled and Triage.
        The name has 1–255 characters, not all spaces, without NUL and
        without any of & + , : ; $ ^ } { * = ? @ # | ' < > . ( ) % ! -; the
        identifier, upper-cased, has 1–10 of A-Z, 0-9 and ÇŞĞİÖÜ; the
        network is 0 or 2, public when not given; the time zone is an IANA
        name, the workspace's when not given (validation_failed). The values
        are checked before the workspace is looked at. A workspace that does
        not exist, is deleted, or of which the caller is not an active member
        answers workspace.not_found; a guest, forbidden. The lead must be an
        active admin or member of the workspace (project_lead_id
        not_allowed), which is checked after the caller's role. Neither the
        name nor the identifier may be another undeleted project's of the
        workspace (project.name_taken, project.identifier_taken).
      security: [{bearer: []}]
      x-problem-codes: [validation_failed, workspace.not_found, forbidden, project.identifier_taken, project.name_taken]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ProjectCreate'
      responses:
        '201':
          description: The new project, as the caller, its admin, sees it.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Project'
        default:
          $ref: '#/components/responses/Problem'
components:
  # 与 api/openapi.yaml 的相同：打包时丢掉模块文件的 securitySchemes，oapi-codegen 读这一份（M0-P3 交接 3）
  securitySchemes:
    bearer:
      type: http
      scheme: bearer
  parameters:
    Slug:
      name: slug
      in: path
      required: true
      description: A workspace's slug, as in the web app's address.
      schema:
        type: string
  responses:
    Problem:
      description: Error (RFC 9457 problem details).
      headers:
        Retry-After:
          description: >-
            Whole seconds to wait before trying again, rounded up; sent with
            rate_limited and server_busy.
          schema:
            type: integer
            minimum: 1
        WWW-Authenticate:
          description: >-
            Sent with every 401 (RFC 9110 15.5.2): Bearer error="invalid_token"
            (RFC 6750 3) when the bearer token sent is refused before the
            operation runs, being invalid or expired; plain Bearer for every
            other 401, including a credential that the operation finds revoked
            while it runs.
          schema:
            type: string
      content:
        application/problem+json:
          schema:
            $ref: '../common.yaml#/components/schemas/Problem'
  schemas:
    ProjectRole:
      description: A member's role in a project, 5 guest, 15 member, 20 admin.
      type: integer
      enum: [5, 15, 20]
    ProjectNetwork:
      description: >-
        Who sees the project besides its members and the workspace's admins:
        0 private, nobody; 2 public, the workspace's members too.
      type: integer
      enum: [0, 2]
    LogoProps:
      description: >-
        A project's icon, the web app's TLogoProps: every field optional, and
        {} no icon.
      type: object
      additionalProperties: false
      properties:
        in_use:
          description: Which of the two icons the project shows.
          type: string
          enum: [emoji, icon]
        emoji:
          $ref: '#/components/schemas/LogoEmoji'
        icon:
          $ref: '#/components/schemas/LogoIcon'
    LogoEmoji:
      type: object
      additionalProperties: false
      properties:
        value:
          description: The emoji, as the web app's picker gives it.
          type: string
        url:
          description: The address of a custom emoji.
          type: string
    LogoIcon:
      type: object
      additionalProperties: false
      properties:
        name:
          type: string
        color:
          type: string
        background_color:
          type: string
    Project:
      description: >-
        A project, as the caller sees it: member_role and sort_order are his,
        null when he is not an active member.
      type: object
      additionalProperties: false
      required: [id, workspace_id, name, description, identifier, network, project_lead_id, default_assignee_id, cycle_view, module_view,
        issue_views_view, intake_view, guest_view_all_features, archive_in, archived_at, logo_props, timezone, cover_image_url, member_role,
        sort_order, member_ids, created_at, updated_at]
      properties:
        id:
          type: string
          format: uuid
        workspace_id:
          type: string
          format: uuid
        name:
          type: string
        description:
          type: string
        identifier:
          description: Upper case; it prefixes the numbers of the project's work items.
          type: string
        network:
          $ref: '#/components/schemas/ProjectNetwork'
        project_lead_id:
          type: [string, 'null']
          format: uuid
        default_assignee_id:
          description: Whom a new work item is assigned to when nobody is given.
          type: [string, 'null']
          format: uuid
        cycle_view:
          description: Whether the project shows cycles.
          type: boolean
        module_view:
          description: Whether the project shows modules.
          type: boolean
        issue_views_view:
          description: Whether the project shows views.
          type: boolean
        intake_view:
          description: Whether the project shows intake.
          type: boolean
        guest_view_all_features:
          description: Whether the project's guests see every work item, not only their own.
          type: boolean
        archive_in:
          description: After how many months a closed work item is archived, 0–12; 0 never.
          type: integer
        archived_at:
          description: When the project was archived; null while it is not.
          type: [string, 'null']
          format: date-time
        logo_props:
          $ref: '#/components/schemas/LogoProps'
        timezone:
          description: An IANA time zone name.
          type: string
        cover_image_url:
          description: Null until uploads arrive (M5).
          type: [string, 'null']
        member_role:
          description: The caller's role in the project; null when he is not an active member.
          oneOf:
            - $ref: '#/components/schemas/ProjectRole'
            - type: 'null'
        sort_order:
          description: The project's place in the caller's sidebar, lowest first; null when he is not an active member.
          type: [number, 'null']
        member_ids:
          description: The accounts of the active members, in the order they joined, then by id.
          type: array
          items:
            type: string
            format: uuid
        created_at:
          type: string
          format: date-time
        updated_at:
          type: string
          format: date-time
    ProjectCreate:
      type: object
      additionalProperties: false
      required: [name, identifier]
      properties:
        name:
          description: >-
            1–255 characters, not all spaces, without any of
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
          description: An active admin or member of the workspace; he becomes an admin of the project.
          type: string
          format: uuid
        logo_props:
          $ref: '#/components/schemas/LogoProps'
        timezone:
          description: An IANA time zone name, e.g. from GET /api/v0/timezones; the workspace's when not given.
          type: string
````

`api/openapi.yaml`（修改，2 处）：

````old api/openapi.yaml
    description: Workspaces and their members.
````
````new api/openapi.yaml
    description: Workspaces and their members.
  - name: project
    description: Projects, their members and their states.
````

````old api/openapi.yaml
    $ref: 'modules/workspace.yaml#/paths/~1api~1v0~1me~1workspaces~1{slug}~1preferences'
````
````new api/openapi.yaml
    $ref: 'modules/workspace.yaml#/paths/~1api~1v0~1me~1workspaces~1{slug}~1preferences'
  /api/v0/workspaces/{slug}/projects:
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1workspaces~1{slug}~1projects'
````

`server/internal/modules/project/adapter/http/gen/oapi-codegen.yaml`（新文件，33 行）：

````file server/internal/modules/project/adapter/http/gen/oapi-codegen.yaml
# oapi-codegen 配置：api/modules/project.yaml → server.gen.go。
# 由 make gen-go 在 server/ 下执行，路径相对于 server/。
package: gen
output: internal/modules/project/adapter/http/gen/server.gen.go
generate:
  models: true
  std-http-server: true
  strict-server: true
compatibility:
  # 枚举常量总是带类型名前缀：以后别的枚举出现同名的值，也不会改掉已有常量的名字
  always-prefix-enum-values: true
# output-options 是每个模块照抄的模板（M2 设计 3.12），bodyshapegen 读同一份 type-mapping
output-options:
  name-normalizer: ToCamelCaseWithInitialisms
  # 可为空又可省略的字段生成 nullable.Nullable[T]：PATCH 能区分"没传"和"传 null"
  nullable-type: true
  type-mapping:
    # 不带 format 的 number 默认生成 float32，bodyshape 查不了它的范围；bodyshapegen 只接受 int、int64 和 float64
    number:
      default:
        type: float64
    string:
      formats:
        # 标准库的 uuid：默认的 openapi_types.UUID 会带进 github.com/google/uuid
        uuid:
          type: uuid.UUID
          import: uuid
        # 邮箱的格式由领域层校验（422）；默认的 openapi_types.Email 在解码时自己校验，绕过这一层
        email:
          type: string
import-mapping:
  # common.yaml 的组件生成在共享包 apigen 里，这里只引用
  ../common.yaml: github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apigen
````

- [ ] **Step 2: 生成**

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `85b31191655fce3a2fcccad0b25995eab691c348e9c9bc314bafb5f80da5ab6d` | 1903 | `api/dist/openapi.yaml` |
| `37a70b20bb528248df55fa7098db89b8e1ba993d16e536bbef4a848ed22059b2` | 34 | `server/internal/modules/project/adapter/http/gen/bodyshape.gen.go` |
| `b2d5efe5b6fb835e243305ecd4cf83c366e9be41e8ceb0397425d19036bd706f` | 504 | `server/internal/modules/project/adapter/http/gen/server.gen.go` |
| `6288716d3410acf132d8da97cb6c085c66dd2c74455b56d35708ebf3ad4e762f` | 1959 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/bodyshape.gen.go server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 3: HTTP 和模块**

`server/internal/modules/project/adapter/http/handler.go`（新文件，47 行）：

````file server/internal/modules/project/adapter/http/handler.go
// Package httpadapter serves the project module's API: it implements the
// strict server that oapi-codegen generates from api/modules/project.yaml
// into the gen package, and translates between the generated types and the
// use cases.
package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

// CreateProjectUseCase is app.CreateProject.
type CreateProjectUseCase interface {
	Execute(ctx context.Context, slug string, in domain.NewProject) (domain.Project, error)
}

// UseCases are the use cases behind the module's operations.
type UseCases struct {
	CreateProject CreateProjectUseCase
}

// Register mounts the module's routes on router behind api's per-route
// middlewares; api.Errors answers binding, decoding and handler errors.
// Every operation needs a token.
func Register(router *httpserver.Router, api *httpserver.API, uc UseCases) {
	strict := gen.NewStrictHandlerWithOptions(handler{uc: uc}, nil, gen.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  api.Errors.BodyError,
		ResponseErrorHandlerFunc: api.Errors.Write,
	})
	var middlewares []gen.MiddlewareFunc
	for _, m := range api.Middlewares(gen.BodyShapes()) {
		middlewares = append(middlewares, m)
	}
	gen.HandlerWithOptions(strict, gen.StdHTTPServerOptions{
		BaseRouter:       router,
		Middlewares:      middlewares,
		ErrorHandlerFunc: api.Errors.BadRequest,
	})
}

// handler implements gen.StrictServerInterface.
type handler struct {
	uc UseCases
}
````

`server/internal/modules/project/adapter/http/projects.go`（新文件，108 行）：

````file server/internal/modules/project/adapter/http/projects.go
package httpadapter

import (
	"context"
	"uuid"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// CreateProject serves POST /api/v0/workspaces/{slug}/projects.
func (h handler) CreateProject(ctx context.Context, req gen.CreateProjectRequestObject) (gen.CreateProjectResponseObject, error) {
	b := req.Body
	in := domain.NewProject{Name: b.Name, Identifier: b.Identifier, LeadID: b.ProjectLeadID, Timezone: b.Timezone}
	if b.Description != nil {
		in.Description = *b.Description
	}
	if b.Network != nil {
		n := domain.Network(*b.Network)
		in.Network = &n
	}
	if b.LogoProps != nil {
		in.LogoProps = logoIn(*b.LogoProps)
	}
	p, err := h.uc.CreateProject.Execute(ctx, req.Slug, in)
	if err != nil {
		return nil, err
	}
	return gen.CreateProject201JSONResponse(project(p)), nil
}

// project is p as the API shows it.
func project(p domain.Project) gen.Project {
	out := gen.Project{
		ID:                   p.ID,
		WorkspaceID:          p.WorkspaceID,
		Name:                 p.Name,
		Description:          p.Description,
		Identifier:           p.Identifier,
		Network:              gen.ProjectNetwork(p.Network),
		ProjectLeadID:        orNull(p.LeadID),
		DefaultAssigneeID:    orNull(p.DefaultAssigneeID),
		CycleView:            p.CycleView,
		ModuleView:           p.ModuleView,
		IssueViewsView:       p.IssueViewsView,
		IntakeView:           p.IntakeView,
		GuestViewAllFeatures: p.GuestViewAllFeatures,
		ArchiveIn:            p.ArchiveIn,
		ArchivedAt:           orNull(p.ArchivedAt),
		LogoProps:            logoOut(p.LogoProps),
		Timezone:             p.Timezone,
		// Required and null until M5.
		CoverImageURL: nullable.NewNullNullable[string](),
		MemberRole:    nullable.NewNullNullable[gen.ProjectRole](),
		SortOrder:     orNull(p.SortOrder),
		// An array, never null: a project without members has [].
		MemberIds: append([]uuid.UUID{}, p.MemberIDs...),
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
	if p.MemberRole != nil {
		out.MemberRole = nullable.NewNullableWithValue(gen.ProjectRole(*p.MemberRole))
	}
	return out
}

// orNull is v as a required field that can be null: null when v is nil.
// The zero Nullable is "unspecified" and would marshal as the zero value.
func orNull[T any](v *T) nullable.Nullable[T] {
	if v == nil {
		return nullable.NewNullNullable[T]()
	}
	return nullable.NewNullableWithValue(*v)
}

// logoIn is the request's icon as the domain takes it.
func logoIn(l gen.LogoProps) domain.LogoProps {
	var out domain.LogoProps
	if l.InUse != nil {
		inUse := string(*l.InUse)
		out.InUse = &inUse
	}
	if e := l.Emoji; e != nil {
		out.Emoji = &domain.Emoji{Value: e.Value, URL: e.URL}
	}
	if i := l.Icon; i != nil {
		out.Icon = &domain.Icon{Name: i.Name, Color: i.Color, BackgroundColor: i.BackgroundColor}
	}
	return out
}

// logoOut is a project's icon as the API shows it: {} for none.
func logoOut(l domain.LogoProps) gen.LogoProps {
	var out gen.LogoProps
	if l.InUse != nil {
		inUse := gen.LogoPropsInUse(*l.InUse)
		out.InUse = &inUse
	}
	if e := l.Emoji; e != nil {
		out.Emoji = &gen.LogoEmoji{Value: e.Value, URL: e.URL}
	}
	if i := l.Icon; i != nil {
		out.Icon = &gen.LogoIcon{Name: i.Name, Color: i.Color, BackgroundColor: i.BackgroundColor}
	}
	return out
}
````

`server/internal/modules/project/adapter/http/handler_test.go`（新文件，129 行）：

````file server/internal/modules/project/adapter/http/handler_test.go
package httpadapter_test

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

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/ratelimit"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Every problem code the module's operations declare must be answered by a
// test here (M2 design 3.11), whichever module produces it (M3 design 9.4).
func TestMain(m *testing.M) { apitest.Main(m, "project") }

var (
	aliceID = uuid.MustParse("0199a2b4-0000-7000-8000-000000000001")
	bobID   = uuid.MustParse("0199a2b4-0000-7000-8000-000000000002")
	created = time.Date(2026, 10, 1, 10, 0, 0, 123456000, time.UTC)
)

// fakeAuth accepts the tokens "alice" and "bob" as those accounts.
type fakeAuth struct{}

func (fakeAuth) Authenticate(ctx context.Context, token string) (context.Context, string, error) {
	switch token {
	case "alice":
		return shared.WithActor(ctx, shared.Actor{UserID: aliceID, SessionID: uuid.NewV7()}), "session:alice", nil
	case "bob":
		return shared.WithActor(ctx, shared.Actor{UserID: bobID, SessionID: uuid.NewV7()}), "session:bob", nil
	}
	return nil, "", shared.Unauthenticated()
}

// caller is the account the context's actor is, by name.
func caller(ctx context.Context) string {
	actor, err := shared.RequireActor(ctx)
	switch {
	case err != nil:
		return "nobody"
	case actor.UserID == aliceID:
		return "alice"
	case actor.UserID == bobID:
		return "bob"
	}
	return "someone else"
}

// fakes are the use cases behind a test server: each records who called it
// with what, and answers what it is given.
type fakes struct {
	create *fakeCreate
}

type fakeCreate struct {
	calls  []string // "caller slug"
	got    []domain.NewProject
	answer domain.Project
	err    error
}

func (f *fakeCreate) Execute(ctx context.Context, slug string, in domain.NewProject) (domain.Project, error) {
	f.calls = append(f.calls, caller(ctx)+" "+slug)
	f.got = append(f.got, in)
	return f.answer, f.err
}

// newServer serves the module with f; a fake left nil is an idle one.
func newServer(t *testing.T, f fakes) http.Handler {
	t.Helper()
	logger := slog.New(slog.DiscardHandler)
	router := httpserver.NewRouter(logger)
	limit := ratelimit.New(time.Now).Bucket("test", ratelimit.Rate{PerMinute: 600, Burst: 100})
	api, err := httpserver.NewAPI(httpserver.APIConfig{
		Logger:         logger,
		Authenticator:  fakeAuth{},
		MaxBodyBytes:   1024,
		RequestTimeout: 5 * time.Second,
		IPv6PrefixLen:  64,
		Anonymous:      limit,
		Authenticated:  limit,
		AuthFailure:    limit,
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.create == nil {
		f.create = &fakeCreate{}
	}
	httpadapter.Register(router, api, httpadapter.UseCases{CreateProject: f.create})
	return router
}

// do serves req and checks the answer against the contract.
func do(t *testing.T, h http.Handler, req *http.Request) (*http.Response, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	res := rec.Result()
	apitest.Load(t).CheckResponse(t, req, res)
	body, _ := io.ReadAll(res.Body)
	return res, string(body)
}

// request is a request with token as its bearer and body as JSON, if any.
func request(method, path, token, body string) *http.Request {
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}
````

`server/internal/modules/project/adapter/http/projects_test.go`（新文件，137 行）：

````file server/internal/modules/project/adapter/http/projects_test.go
package httpadapter_test

import (
	"net/http"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var (
	acmeID = uuid.MustParse("0199a2b4-0000-7000-8000-00000000000a")
	webID  = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000a1")
	// web is a project as its admin sees it, every field set.
	web = domain.Project{ID: webID, WorkspaceID: acmeID, Name: "Web", Description: "The app", Identifier: "WEB", Network: domain.NetworkPrivate,
		LeadID: &bobID, DefaultAssigneeID: &aliceID, CycleView: true, ModuleView: true, IssueViewsView: true, IntakeView: true,
		GuestViewAllFeatures: true, ArchiveIn: 3, ArchivedAt: &created,
		LogoProps: domain.LogoProps{InUse: ptr("icon"), Emoji: &domain.Emoji{Value: ptr("128640"), URL: ptr("https://example.com/e.png")},
			Icon: &domain.Icon{Name: ptr("home"), Color: ptr("#ffffff"), BackgroundColor: ptr("#000000")}},
		Timezone: "Asia/Shanghai", CreatedAt: created, UpdatedAt: created, MemberRole: ptr(shared.RoleAdmin), SortOrder: ptr(-9900.5),
		MemberIDs: []uuid.UUID{aliceID, bobID}}
	// bare is a project with every optional field empty, as a caller who is
	// not its member sees it.
	bare = domain.Project{ID: webID, WorkspaceID: acmeID, Name: "Web", Identifier: "WEB", Network: domain.NetworkPublic, Timezone: "UTC",
		CreatedAt: created, UpdatedAt: created}
)

const (
	webJSON = `{"archive_in":3,"archived_at":"2026-10-01T10:00:00.123456Z","cover_image_url":null,"created_at":"2026-10-01T10:00:00.123456Z",` +
		`"cycle_view":true,"default_assignee_id":"0199a2b4-0000-7000-8000-000000000001","description":"The app","guest_view_all_features":true,` +
		`"id":"0199a2b4-0000-7000-8000-0000000000a1","identifier":"WEB","intake_view":true,"issue_views_view":true,` +
		`"logo_props":{"emoji":{"url":"https://example.com/e.png","value":"128640"},"icon":{"background_color":"#000000","color":"#ffffff","name":"home"},"in_use":"icon"},` +
		`"member_ids":["0199a2b4-0000-7000-8000-000000000001","0199a2b4-0000-7000-8000-000000000002"],"member_role":20,"module_view":true,` +
		`"name":"Web","network":0,"project_lead_id":"0199a2b4-0000-7000-8000-000000000002","sort_order":-9900.5,"timezone":"Asia/Shanghai",` +
		`"updated_at":"2026-10-01T10:00:00.123456Z","workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
	bareJSON = `{"archive_in":0,"archived_at":null,"cover_image_url":null,"created_at":"2026-10-01T10:00:00.123456Z","cycle_view":false,` +
		`"default_assignee_id":null,"description":"","guest_view_all_features":false,"id":"0199a2b4-0000-7000-8000-0000000000a1",` +
		`"identifier":"WEB","intake_view":false,"issue_views_view":false,"logo_props":{},"member_ids":[],"member_role":null,"module_view":false,` +
		`"name":"Web","network":2,"project_lead_id":null,"sort_order":null,"timezone":"UTC","updated_at":"2026-10-01T10:00:00.123456Z",` +
		`"workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
)

func ptr[T any](v T) *T { return &v }

// The body becomes the use case's input for the caller and the path's
// workspace, each optional field nil or empty when absent; the answer is 201
// with the project the use case answers, every field shown, the empty ones
// as null, [] and {}.
func TestCreateProjectAnswers201(t *testing.T) {
	create := &fakeCreate{answer: web}
	h := newServer(t, fakes{create: create})

	res, body := do(t, h, request(http.MethodPost, "/api/v0/workspaces/acme/projects", "alice",
		`{"name":"Web","identifier":"web","description":"The app","network":0,"project_lead_id":"0199a2b4-0000-7000-8000-000000000002",`+
			`"logo_props":{"in_use":"icon","emoji":{"value":"128640","url":"https://example.com/e.png"},`+
			`"icon":{"name":"home","color":"#ffffff","background_color":"#000000"}},"timezone":"Asia/Shanghai"}`))
	if res.StatusCode != http.StatusCreated || body != webJSON+"\n" {
		t.Errorf("POST = %d %s, want 201 %s", res.StatusCode, body, webJSON)
	}
	create.answer = bare
	res, body = do(t, h, request(http.MethodPost, "/api/v0/workspaces/beta/projects", "bob", `{"name":"Web","identifier":"WEB"}`))
	if res.StatusCode != http.StatusCreated || body != bareJSON+"\n" {
		t.Errorf("POST without the optional fields = %d %s, want 201 %s", res.StatusCode, body, bareJSON)
	}

	private := domain.NetworkPrivate
	want := []domain.NewProject{
		{Name: "Web", Identifier: "web", Description: "The app", Network: &private, LeadID: &bobID, Timezone: ptr("Asia/Shanghai"),
			LogoProps: web.LogoProps},
		{Name: "Web", Identifier: "WEB"},
	}
	if !reflect.DeepEqual(create.got, want) {
		t.Errorf("inputs = %+v, want %+v", create.got, want)
	}
	if want := []string{"alice acme", "bob beta"}; !slices.Equal(create.calls, want) {
		t.Errorf("calls = %q, want %q", create.calls, want)
	}
}

// The use case's refusals, as the contract declares them.
func TestCreateProjectRefusals(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"no workspace", domain.ErrWorkspaceNotFound, http.StatusNotFound,
			`{"status":404,"code":"workspace.not_found","title":"Not Found","detail":"The workspace does not exist, or you are not a member of it."}`},
		{"a guest", shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"a lead who may not lead", domain.LeadNotAllowed(), http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"project_lead_id","code":"not_allowed","message":"must be an active admin or member of the workspace"}]}`},
		{"a taken identifier", domain.ErrIdentifierTaken, http.StatusConflict,
			`{"status":409,"code":"project.identifier_taken","title":"Conflict","detail":"A project of the workspace has this identifier."}`},
		{"a taken name", domain.ErrNameTaken, http.StatusConflict,
			`{"status":409,"code":"project.name_taken","title":"Conflict","detail":"A project of the workspace has this name."}`},
	}
	for _, tt := range tests {
		h := newServer(t, fakes{create: &fakeCreate{err: tt.err}})
		res, body := do(t, h, request(http.MethodPost, "/api/v0/workspaces/acme/projects", "alice", `{"name":"Web","identifier":"WEB"}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: POST = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

// logo_props is a closed structure (M3 design 5.2): a key it does not have,
// at any depth, or a value of another type is refused as bad_request before
// the use case; an in_use of the right type but no known value is the
// domain's to refuse, and reaches it.
func TestCreateProjectHoldsTheLogoToItsStructure(t *testing.T) {
	create := &fakeCreate{answer: bare}
	h := newServer(t, fakes{create: create})
	for _, logo := range []string{
		`{"shape":"round"}`, `{"emoji":{"value":"1","size":2}}`, `{"icon":{"name":"home","shape":"round"}}`,
		`{"in_use":17}`, `{"emoji":"x"}`, `{"emoji":{"value":1}}`, `{"icon":{"color":null}}`, `[]`, `"x"`,
	} {
		res, body := do(t, h, request(http.MethodPost, "/api/v0/workspaces/acme/projects", "alice",
			`{"name":"Web","identifier":"WEB","logo_props":`+logo+`}`))
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("logo_props %s: POST = %d %s, want 400", logo, res.StatusCode, body)
		}
	}
	if len(create.got) != 0 {
		t.Fatalf("the use case got %+v, want nothing", create.got)
	}
	res, _ := do(t, h, request(http.MethodPost, "/api/v0/workspaces/acme/projects", "alice",
		`{"name":"Web","identifier":"WEB","logo_props":{"in_use":"other"}}`))
	if res.StatusCode != http.StatusCreated || len(create.got) != 1 || create.got[0].LogoProps.InUse == nil || *create.got[0].LogoProps.InUse != "other" {
		t.Errorf("in_use other: POST = %d, inputs %+v; want it passed to the use case", res.StatusCode, create.got)
	}
}
````

`server/internal/modules/project/module.go`（修改，8 处）：

````old server/internal/modules/project/module.go
// carries out the workspace module's cascades on the projects
// (ProjectCascade).
````
````new server/internal/modules/project/module.go
// brings creating projects, and carries out the workspace module's cascades
// on the projects (ProjectCascade).
````

````old server/internal/modules/project/module.go
	"github.com/jackc/pgx/v5/pgxpool"

````
````new server/internal/modules/project/module.go
	"github.com/jackc/pgx/v5/pgxpool"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/http"
````

````old server/internal/modules/project/module.go
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
````
````new server/internal/modules/project/module.go
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
````

````old server/internal/modules/project/module.go
}

// Deps are what bootstrap gives the module (M3 design 6.6, step 4).
````
````new server/internal/modules/project/module.go
}

// Workspace is a workspace as the WorkspaceDirectory port hands it over:
// bootstrap converts workspace's DirectoryEntry into it (M3 design 6.5).
type Workspace = app.Workspace

// Deps are what bootstrap gives the module (M3 design 6.6, step 4).
````

````old server/internal/modules/project/module.go
	Pool *pgxpool.Pool
````
````new server/internal/modules/project/module.go
	Pool       *pgxpool.Pool
	Tx         shared.TxManager
	Clock      app.Clock
	Authorizer shared.Authorizer
	// Workspaces is workspace's WorkspaceDirectory, converted
	// (bootstrap/ports.go).
	Workspaces app.WorkspaceDirectory
	// Members is workspace's WorkspaceMembers (workspace.Provide).
	Members app.WorkspaceMembers
````

````old server/internal/modules/project/module.go
	cascade *app.Cascade
````
````new server/internal/modules/project/module.go
	cascade *app.Cascade
	uc      httpadapter.UseCases
````

````old server/internal/modules/project/module.go
// New wires the module from d; nothing is registered or injected after it
// (M3 design 6.6). bootstrap builds it before workspace, which takes its
// Cascade.
````
````new server/internal/modules/project/module.go
// New wires the module's use cases and its HTTP side from d; nothing is
// registered or injected after it (M3 design 6.6). bootstrap builds it
// before workspace, which takes its Cascade.
````

````old server/internal/modules/project/module.go
	return &Module{cascade: app.NewCascade(store)}
````
````new server/internal/modules/project/module.go
	return &Module{cascade: app.NewCascade(store), uc: httpadapter.UseCases{
		CreateProject: app.NewCreateProject(app.CreateProjectDeps{
			Workspaces: d.Workspaces, Members: d.Members, Projects: store, Auth: d.Authorizer, Tx: d.Tx, Clock: d.Clock,
		}),
	}}
}

// Register mounts the module's API on router behind api's middlewares.
func (m *Module) Register(router *httpserver.Router, api *httpserver.API) {
	httpadapter.Register(router, api, m.uc)
````

- [ ] **Step 4: 每个模块的 HTTP 测试经 `apitest.Main`**

`server/internal/platform/httpserver/apitest/main_callers_test.go`（新文件，131 行）：

````file server/internal/platform/httpserver/apitest/main_callers_test.go
package apitest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
)

// Every module with a file in api/modules/ runs the tests of its HTTP
// adapter through Main, under its own name (M3/P1 review, M6): a module
// whose tests ran without it, or with another module's name, would never
// have its declared problem codes checked.
func TestEveryModuleRunsItsHTTPTestsThroughMain(t *testing.T) {
	names, err := moduleNames()
	if err != nil || len(names) == 0 {
		t.Fatalf("module files = %q, %v; want at least one", names, err)
	}
	for _, name := range names {
		for _, v := range mainViolations(adapterDir(name), name) {
			t.Error(v)
		}
	}
}

// adapterDir is server/internal/modules/<module>/adapter/http, where Main's
// doc comment says the module's HTTP tests are.
func adapterDir(module string) string {
	return filepath.Join(apiDir(), "..", "server", "internal", "modules", module, "adapter", "http")
}

// mainViolations reports what keeps the tests in dir from running through
// Main for module: no TestMain in its _test.go files, or one whose body is
// not the one call Main(m, "<module>") with its own *testing.M.
func mainViolations(dir, module string) []string {
	paths, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		return []string{err.Error()}
	}
	var found []string
	mains := 0
	for _, path := range paths {
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
		if err != nil {
			return []string{err.Error()}
		}
		for _, d := range f.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.Name == "TestMain" {
				mains++
				if !callsMain(fn, module) {
					found = append(found, fmt.Sprintf("%s: TestMain is not apitest.Main(m, %q)", path, module))
				}
			}
		}
	}
	if mains == 0 {
		found = append(found, fmt.Sprintf("%s has no TestMain: write func TestMain(m *testing.M) { apitest.Main(m, %q) }", dir, module))
	}
	return found
}

// callsMain reports whether fn's body is the one statement
// apitest.Main(<its parameter>, "<module>").
func callsMain(fn *ast.FuncDecl, module string) bool {
	params := fn.Type.Params.List
	if len(params) != 1 || len(params[0].Names) != 1 || fn.Body == nil || len(fn.Body.List) != 1 {
		return false
	}
	stmt, ok := fn.Body.List[0].(*ast.ExprStmt)
	if !ok {
		return false
	}
	call, ok := stmt.X.(*ast.CallExpr)
	if !ok || len(call.Args) != 2 {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	arg, isIdent := call.Args[0].(*ast.Ident)
	name, isLit := call.Args[1].(*ast.BasicLit)
	return ok && pkg.Name == "apitest" && sel.Sel.Name == "Main" && isIdent && arg.Name == params[0].Names[0].Name && isLit &&
		name.Kind == token.STRING && name.Value == strconv.Quote(module)
}

// Each check of mainViolations fails on its counterexample, and a package
// that runs through Main passes.
func TestMainViolationsCatchesEachGap(t *testing.T) {
	const head = "package p_test\n\nimport (\n\t\"os\"\n\t\"testing\"\n\n\t\"x/apitest\"\n)\n\nvar _ = os.Exit\n\n"
	tests := []struct {
		name  string
		files map[string]string
		want  []string
	}{
		{"through Main", map[string]string{"handler_test.go": head + "func TestMain(m *testing.M) { apitest.Main(m, \"project\") }\n",
			"other_test.go": head + "func TestOther(t *testing.T) {}\n"}, nil},
		{"no TestMain", map[string]string{"handler_test.go": head + "func TestOther(t *testing.T) {}\n"},
			[]string{"DIR has no TestMain: write func TestMain(m *testing.M) { apitest.Main(m, \"project\") }"}},
		{"a plain TestMain", map[string]string{"handler_test.go": head + "func TestMain(m *testing.M) { os.Exit(m.Run()) }\n"},
			[]string{"DIR/handler_test.go: TestMain is not apitest.Main(m, \"project\")"}},
		{"another module's name", map[string]string{"handler_test.go": head + "func TestMain(m *testing.M) { apitest.Main(m, \"workspace\") }\n"},
			[]string{"DIR/handler_test.go: TestMain is not apitest.Main(m, \"project\")"}},
		{"Main and more", map[string]string{"handler_test.go": head +
			"func TestMain(m *testing.M) {\n\tapitest.Main(m, \"project\")\n\tos.Exit(0)\n}\n"},
			[]string{"DIR/handler_test.go: TestMain is not apitest.Main(m, \"project\")"}},
		{"another M", map[string]string{"handler_test.go": head + "var other *testing.M\n\nfunc TestMain(m *testing.M) { apitest.Main(other, \"project\") }\n"},
			[]string{"DIR/handler_test.go: TestMain is not apitest.Main(m, \"project\")"}},
	}
	for _, tt := range tests {
		dir := t.TempDir()
		for name, src := range tt.files {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		var want []string
		for _, w := range tt.want {
			want = append(want, filepath.Clean(dir)+w[len("DIR"):])
		}
		if got := mainViolations(dir, "project"); !slices.Equal(got, want) {
			t.Errorf("%s: %q, want %q", tt.name, got, want)
		}
	}
}
````

- [ ] **Step 5: 组合根、组合出的测试、矩阵**

`server/internal/bootstrap/ports.go`（修改，2 处）：

````old server/internal/bootstrap/ports.go
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
````
````new server/internal/bootstrap/ports.go
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
````

````old server/internal/bootstrap/ports.go
	return out, nil
}

````
````new server/internal/bootstrap/ports.go
	return out, nil
}

// projectWorkspaces is workspace's WorkspaceDirectory as project's port:
// the same reads and locks, each workspace converted.
type projectWorkspaces struct {
	directory workspace.WorkspaceDirectory
}

func (d projectWorkspaces) ShareWorkspaceBySlug(ctx context.Context, slug string) (project.Workspace, bool, error) {
	w, found, err := d.directory.ShareWorkspaceBySlug(ctx, slug)
	return project.Workspace(w), found, err
}

````

`server/internal/bootstrap/ports_test.go`（修改，2 处）：

````old server/internal/bootstrap/ports_test.go
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
````
````new server/internal/bootstrap/ports_test.go
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
````

````old server/internal/bootstrap/ports_test.go
		t.Errorf("PublicProfiles() = %+v, %v; want no profiles and %v", got, err, failure)
	}
}

````
````new server/internal/bootstrap/ports_test.go
		t.Errorf("PublicProfiles() = %+v, %v; want no profiles and %v", got, err, failure)
	}
}

// fakeWorkspaceDirectory answers the workspaces it holds by slug, and
// records what it was asked: "read" without a lock, "share" with one.
type fakeWorkspaceDirectory struct {
	workspaces map[string]workspace.DirectoryEntry
	err        error
	asked      []string
}

func (f *fakeWorkspaceDirectory) WorkspaceBySlug(_ context.Context, slug string) (workspace.DirectoryEntry, bool, error) {
	f.asked = append(f.asked, "read "+slug)
	w, found := f.workspaces[slug]
	return w, found, f.err
}

func (f *fakeWorkspaceDirectory) ShareWorkspaceBySlug(_ context.Context, slug string) (workspace.DirectoryEntry, bool, error) {
	f.asked = append(f.asked, "share "+slug)
	w, found := f.workspaces[slug]
	return w, found, f.err
}

// projectWorkspaces hands project workspace's answer to the same question,
// through the same lock: the workspace converted, found and the error as
// they came.
func TestProjectWorkspacesConvertsWorkspacesAnswer(t *testing.T) {
	acme := workspace.DirectoryEntry{ID: uuid.NewV7(), Timezone: "Asia/Shanghai"}
	fake := &fakeWorkspaceDirectory{workspaces: map[string]workspace.DirectoryEntry{"acme": acme}}
	d := projectWorkspaces{directory: fake}
	ctx := context.Background()

	w, found, err := d.ShareWorkspaceBySlug(ctx, "acme")
	if want := (project.Workspace{ID: acme.ID, Timezone: "Asia/Shanghai"}); err != nil || !found || w != want {
		t.Errorf("ShareWorkspaceBySlug(acme) = %+v, %v, %v; want %+v, found", w, found, err, want)
	}
	if w, found, err := d.ShareWorkspaceBySlug(ctx, "gone"); err != nil || found || w != (project.Workspace{}) {
		t.Errorf("ShareWorkspaceBySlug(gone) = %+v, %v, %v; want not found", w, found, err)
	}
	if want := []string{"share acme", "share gone"}; !slices.Equal(fake.asked, want) {
		t.Errorf("workspace was asked %q, want %q", fake.asked, want)
	}
	failure := errors.New("connection reset")
	fake.err = failure
	if _, _, err := d.ShareWorkspaceBySlug(ctx, "acme"); !errors.Is(err, failure) {
		t.Errorf("ShareWorkspaceBySlug() = %v, want %v", err, failure)
	}
}

````

`server/internal/bootstrap/app.go`（修改，2 处）：

````old server/internal/bootstrap/app.go
	proj := project.New(project.Deps{Pool: pool})
````
````new server/internal/bootstrap/app.go
	proj := project.New(project.Deps{
		Pool: pool, Tx: tx, Clock: clock.System{}, Authorizer: authorizer,
		Workspaces: projectWorkspaces{directory: workspacePorts.WorkspaceDirectory},
		Members:    workspacePorts.WorkspaceMembers,
	})
````

````old server/internal/bootstrap/app.go
	ws.Register(a.router, api)
````
````new server/internal/bootstrap/app.go
	ws.Register(a.router, api)
	proj.Register(a.router, api)
````

`server/internal/bootstrap/project_test.go`（新文件，112 行）：

````file server/internal/bootstrap/project_test.go
package bootstrap

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// The project module as bootstrap wires it (M3 design 6.6): alice creates
// acme, in Shanghai time, and invites bob, who joins it as a member. She
// creates a project with bob as its lead: the answer is the project as
// stored, as she sees it, her role 20, first in her sidebar, the two of
// them its members, in acme's time zone; the rows it wrote are there, each
// of them its admin with his display settings, and the six states, Backlog
// the default. Another project with its identifier, in any case, or its
// name answers the code of each and writes nothing.
func TestCreatingAProject(t *testing.T) {
	contract := apitest.Load(t)
	dbURL := pgtest.NewDatabase(t)
	base := startApp(t, testConfig(t, dbURL, false), migrations.FS())
	alice := registerAccount(t, contract, base, "alice@example.com").AccessToken
	bob := registerAccount(t, contract, base, "bob@example.com").AccessToken
	if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", alice,
		`{"name":"Acme","slug":"acme","timezone":"Asia/Shanghai"}`); status != http.StatusCreated {
		t.Fatalf("creating acme = %d %s", status, body)
	}
	answerInvitation(t, contract, base, bob, "accept", invite(t, contract, base, alice, "acme", "bob@example.com"), http.StatusOK)
	aliceID, bobID := accountID(t, contract, base, alice), accountID(t, contract, base, bob)

	status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces/acme/projects", alice,
		`{"name":"Web","identifier":"web","project_lead_id":"`+bobID.String()+`"}`)

	var p struct {
		ID         uuid.UUID   `json:"id"`
		Identifier string      `json:"identifier"`
		Network    int         `json:"network"`
		LeadID     *uuid.UUID  `json:"project_lead_id"`
		Timezone   string      `json:"timezone"`
		MemberRole *int        `json:"member_role"`
		SortOrder  *float64    `json:"sort_order"`
		MemberIDs  []uuid.UUID `json:"member_ids"`
	}
	decodeAnswer(t, body, &p)
	if status != http.StatusCreated || p.Identifier != "WEB" || p.Network != 2 || p.LeadID == nil || *p.LeadID != bobID ||
		p.Timezone != "Asia/Shanghai" || p.MemberRole == nil || *p.MemberRole != 20 || p.SortOrder == nil || *p.SortOrder != 65535 ||
		!slices.Equal(p.MemberIDs, []uuid.UUID{aliceID, bobID}) {
		t.Fatalf("create = %d %s; want 201 with WEB, public, bob its lead, in Asia/Shanghai, alice its admin at 65535, alice and bob its members",
			status, body)
	}
	pool := openPool(t, dbURL)
	want := aliceID.String() + " 20 65535, " + bobID.String() + " 20 65535 | " +
		"Backlog backlog default, Todo unstarted, In Progress started, Done completed, Cancelled cancelled, Triage triage | 1 project"
	if got := projectRows(t, pool, p.ID); got != want {
		t.Errorf("the rows are %s, want %s", got, want)
	}

	for _, tt := range []struct{ body, code string }{
		{`{"name":"Web 2","identifier":"Web"}`, "project.identifier_taken"},
		{`{"name":"Web","identifier":"WEB2"}`, "project.name_taken"},
	} {
		if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces/acme/projects", bob, tt.body); status != http.StatusConflict ||
			problemCode(t, []byte(body)) != tt.code {
			t.Errorf("POST %s = %d %s, want 409 %s", tt.body, status, body, tt.code)
		}
	}
	if got := projectRows(t, pool, p.ID); got != want {
		t.Errorf("after the refusals the rows are %s, want %s", got, want)
	}
}

// accountID is the id of the account whose access token is token.
func accountID(t *testing.T, contract *apitest.Contract, base, token string) uuid.UUID {
	t.Helper()
	status, body := call(t, contract, http.MethodGet, base+"/api/v0/me", token, "")
	var me struct {
		ID uuid.UUID `json:"id"`
	}
	if status != http.StatusOK {
		t.Fatalf("GET /api/v0/me = %d %s", status, body)
	}
	decodeAnswer(t, body, &me)
	return me.ID
}

// projectRows are the undeleted rows under the project id: each
// membership's account, role and place in his sidebar; each state's name,
// group and whether it is the default, by sequence; and the number of the
// workspace's projects.
func projectRows(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) string {
	t.Helper()
	var rows string
	if err := pool.QueryRow(context.Background(), `
SELECT (SELECT string_agg(m.member_id || ' ' || m.role || ' ' || u.sort_order, ', ' ORDER BY m.id)
          FROM project_members m JOIN project_user_properties u ON u.project_id = m.project_id AND u.user_id = m.member_id
         WHERE m.project_id = $1 AND m.deleted_at IS NULL AND u.deleted_at IS NULL)
    || ' | ' || (SELECT string_agg(s.name || ' ' || s."group" || CASE WHEN s."default" THEN ' default' ELSE '' END, ', ' ORDER BY s.sequence)
                   FROM states s WHERE s.project_id = $1 AND s.deleted_at IS NULL)
    || ' | ' || (SELECT count(*) FROM projects p
                  WHERE p.workspace_id = (SELECT workspace_id FROM projects WHERE id = $1) AND p.deleted_at IS NULL) || ' project'`,
		id).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	return rows
}
````

`server/internal/bootstrap/project_wiring_test.go`（新文件，65 行）：

````file server/internal/bootstrap/project_wiring_test.go
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

// createProject as bootstrap wires it (M3 design 3.6, 6.6): its rows carry
// the time of the request, from the clock project.New takes; and it writes
// in the one transaction of the TxManager project.New takes, so while every
// insert of a state fails, a create answers 500 and leaves no project,
// membership or display setting behind.
func TestCreateProjectRunsOnTheWiredClockAndTransaction(t *testing.T) {
	contract := apitest.Load(t)
	dbURL := pgtest.NewDatabase(t)
	base := startApp(t, testConfig(t, dbURL, false), migrations.FS())
	pool := openPool(t, dbURL)
	alice := registerAccount(t, contract, base, "alice@example.com").AccessToken
	if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", alice,
		`{"name":"Acme","slug":"acme"}`); status != http.StatusCreated {
		t.Fatalf("creating acme = %d %s", status, body)
	}
	before := time.Now().Truncate(time.Microsecond)
	status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces/acme/projects", alice, `{"name":"Web","identifier":"WEB"}`)
	after := time.Now()
	if status != http.StatusCreated {
		t.Fatalf("creating Web = %d %s", status, body)
	}
	var web struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, body, &web)
	var createdAt time.Time
	if err := pool.QueryRow(context.Background(), "SELECT created_at FROM projects WHERE id = $1", web.ID).Scan(&createdAt); err != nil {
		t.Fatal(err)
	}
	if createdAt.Before(before) || createdAt.After(after) {
		t.Errorf("Web was created at %v, want within the request, %v to %v", createdAt, before, after)
	}

	rows := func() string {
		t.Helper()
		var n string
		if err := pool.QueryRow(context.Background(), `SELECT (SELECT count(*) FROM projects) || ' ' || (SELECT count(*) FROM project_members)
			|| ' ' || (SELECT count(*) FROM project_user_properties) || ' ' || (SELECT count(*) FROM states)`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	want := rows()
	if _, err := pool.Exec(context.Background(), "ALTER TABLE states ADD CONSTRAINT no_states CHECK (false) NOT VALID"); err != nil {
		t.Fatal(err)
	}
	status, body = call(t, contract, http.MethodPost, base+"/api/v0/workspaces/acme/projects", alice, `{"name":"Ops","identifier":"OPS"}`)
	if got := rows(); status != http.StatusInternalServerError || got != want {
		t.Errorf("creating Ops while the states fail = %d %s, rows %s; want 500 and the rows as they were, %s", status, body, got, want)
	}
}
````

`server/internal/bootstrap/permission_matrix_project_test.go`（新文件，36 行）：

````file server/internal/bootstrap/permission_matrix_project_test.go
package bootstrap

import (
	"net/http"
	"testing"
	"uuid"
)

// The project module's rows of the permission matrix (M3 design 9.2).

func projectMatrixRows() []matrixRow {
	return []matrixRow{
		{op: "createProject", write: true, request: toWorkspace(http.MethodPost, "/projects", `{"name":"New","identifier":"new"}`),
			cells: inWorkspace(cellCreated, cellCreated, cellForbidden), check: createsItsProject},
		// A lead who is no member of the workspace is refused after the
		// decision (M3 design 3.6 convention 3): the guest is still refused
		// as a guest, and learns nothing of the lead.
		{op: "createProject", variant: "a lead who is no member", write: true,
			request: toWorkspace(http.MethodPost, "/projects", `{"name":"New","identifier":"NEW","project_lead_id":"`+uuid.Nil().String()+`"}`),
			cells:   inWorkspace(cellValidationFailed, cellValidationFailed, cellForbidden)},
	}
}

// createsItsProject: the project is created as asked, with its caller as
// its admin and only member.
func createsItsProject(t *testing.T, c caller, _ seeded, answer string) {
	var p struct {
		Identifier string      `json:"identifier"`
		MemberRole *int        `json:"member_role"`
		MemberIDs  []uuid.UUID `json:"member_ids"`
	}
	decodeAnswer(t, answer, &p)
	if p.Identifier != "NEW" || p.MemberRole == nil || *p.MemberRole != 20 || len(p.MemberIDs) != 1 {
		t.Errorf("%s creates %s; want NEW, with him its admin and only member", c, answer)
	}
}
````

`server/internal/bootstrap/permission_matrix_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_test.go
	return slices.Concat(workspaceMatrixRows())
````
````new server/internal/bootstrap/permission_matrix_test.go
	return slices.Concat(workspaceMatrixRows(), projectMatrixRows())
````

- [ ] **Step 6: 新码的文案**

`web/apps/web/helpers/authentication.helper.ts`（修改，1 处）：

````old web/apps/web/helpers/authentication.helper.ts
  "workspace.invitation_email_mismatch": "auth.errors.workspace_invitation_email_mismatch",
````
````new web/apps/web/helpers/authentication.helper.ts
  "workspace.invitation_email_mismatch": "auth.errors.workspace_invitation_email_mismatch",
  "project.identifier_taken": "auth.errors.project_identifier_taken",
  "project.name_taken": "auth.errors.project_name_taken",
````

`web/packages/i18n/src/locales/en/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/auth.json
      "workspace_invitation_email_mismatch": "This invitation was sent to another email address.",
````
````new web/packages/i18n/src/locales/en/auth.json
      "workspace_invitation_email_mismatch": "This invitation was sent to another email address.",
      "project_identifier_taken": "A project of this workspace already has this identifier.",
      "project_name_taken": "A project of this workspace already has this name.",
````

`web/packages/i18n/src/locales/zh-CN/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/auth.json
      "workspace_invitation_email_mismatch": "这份邀请发给了另一个邮箱。",
````
````new web/packages/i18n/src/locales/zh-CN/auth.json
      "workspace_invitation_email_mismatch": "这份邀请发给了另一个邮箱。",
      "project_identifier_taken": "这个工作区已有项目使用这个标识。",
      "project_name_taken": "这个工作区已有项目使用这个名称。",
````

- [ ] **Step 7: 测试、lint、前端检查**

Run: `go -C server test -count=1 ./internal/modules/project/... ./internal/platform/httpserver/apitest/`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestCreatingAProject|TestCreateProjectRunsOnTheWiredClockAndTransaction|TestProjectWorkspacesConvertsWorkspacesAnswer|TestPermissionMatrix$|TestThePermissionMatrixCoversEveryOperation|TestAPIRoutesAreTheContractsOperations|TestBodiesThatBreakTheStructureAnswer400' ./internal/bootstrap/`
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
git add api/modules/project.yaml api/openapi.yaml server/internal/bootstrap/app.go server/internal/bootstrap/permission_matrix_project_test.go server/internal/bootstrap/permission_matrix_test.go server/internal/bootstrap/ports.go server/internal/bootstrap/ports_test.go server/internal/bootstrap/project_test.go server/internal/bootstrap/project_wiring_test.go server/internal/modules/project/adapter/http/gen/oapi-codegen.yaml server/internal/modules/project/adapter/http/handler.go server/internal/modules/project/adapter/http/handler_test.go server/internal/modules/project/adapter/http/projects.go server/internal/modules/project/adapter/http/projects_test.go server/internal/modules/project/module.go server/internal/platform/httpserver/apitest/main_callers_test.go web/apps/web/helpers/authentication.helper.ts web/packages/i18n/src/locales/en/auth.json web/packages/i18n/src/locales/zh-CN/auth.json api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/bodyshape.gen.go server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P4a): POST /api/v0/workspaces/{slug}/projects; every module's HTTP tests run through apitest.Main

The project module's contract starts with createProject and the Project
every answer shows, cover_image_url null until M5; logo_props is a
closed structure the contract holds. bootstrap converts workspace's
directory for project, wires project.New's use case with the Authorizer
and registers its routes. The matrix gains createProject's rows, and a
test holds every module's HTTP tests to apitest.Main under its own name
(M3/P1 review, M6).

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 |
|---|---|
| handler 不传路径的工作区；丢掉说明、网络、负责人、时区、图标或图标的某个字段 | `TestCreateProjectAnswers201` |
| 答案的负责人是默认负责人；`archived_at`、`member_role`、`sort_order` 总是 `null`；丢掉表情；`intake_view` 总是假；没有成员答成 `null` | `TestCreateProjectAnswers201` |
| 用例的拒绝答成 201 | `TestCreateProjectRefusals` |
| 生成的 `bodyshape` 让 `logo_props`、`emoji`、`icon` 接受任何键；`icon.color` 接受 `null` | `TestCreateProjectHoldsTheLogoToItsStructure` |
| 契约少 `project.name_taken`；模块文件声明一个没有测试回答的码；答案的 `project_lead_id` 不可为空 | `TestCreateProjectRefusals`；`apitest.Main`（`declares problem codes that no test answered`）；`TestCreateProjectAnswers201` |
| 组合根不注册项目的路由；目录的锁转成不加锁的读；转换丢掉时区；`createProject` 不经 `Authorizer` | `TestAPIRoutesAreTheContractsOperations`、`TestCreatingAProject`；`TestProjectWorkspacesConvertsWorkspacesAnswer`；同前另有 `TestCreatingAProject`；`TestPermissionMatrix` |
| `project.New` 的 `Tx` 换成不开事务的；`Clock` 换成固定的时刻（`mutants_j.py`） | `TestCreateProjectRunsOnTheWiredClockAndTransaction` |
| 矩阵少项目的行 | `TestThePermissionMatrixCoversEveryOperation` |
| 核对不看模块名、函数体、`*testing.M`；没有 `TestMain` 的包通过；不读真实的目录 | `TestMainViolationsCatchesEachGap`；最后一个 `TestEveryModuleRunsItsHTTPTestsThroughMain` |
| `project` 的 HTTP 测试不经 `apitest.Main`（`guard_inplace.py`，在文件上改） | `TestEveryModuleRunsItsHTTPTestsThroughMain` |

**Done when:** 组合出的 app 建项目、答案和四张表的行、两个 409 由测试核对，`project.New` 接的时钟和事务也由组合出的测试核对；矩阵 144 格；`project` 的 `apitest.Main` 两个方向通过，每个模块的 HTTP 测试都经它；前端检查通过。

---

### Task 9: `ProjectAccess`（`project.Provide`）与 `getProject`

**Files:**
- Create: `server/internal/bootstrap/project_access_test.go`、`server/internal/modules/project/adapter/postgres/access.go`、`server/internal/modules/project/adapter/postgres/access_test.go`、`server/internal/modules/project/adapter/postgres/queries/access.sql`、`server/internal/modules/project/app/access.go`、`server/internal/modules/project/app/get_project.go`、`server/internal/modules/project/app/get_project_test.go`
- Modify: `api/modules/project.yaml`、`api/openapi.yaml`、`server/internal/bootstrap/app.go`、`server/internal/bootstrap/permission_matrix_columns_test.go`、`server/internal/bootstrap/permission_matrix_project_test.go`、`server/internal/bootstrap/ports.go`、`server/internal/bootstrap/ports_test.go`、`server/internal/modules/access/app/authorizer.go`、`server/internal/modules/access/app/authorizer_test.go`、`server/internal/modules/access/app/ports.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/access/module.go`、`server/internal/modules/project/adapter/http/handler.go`、`server/internal/modules/project/adapter/http/handler_test.go`、`server/internal/modules/project/adapter/http/projects.go`、`server/internal/modules/project/adapter/http/projects_test.go`、`server/internal/modules/project/app/ports.go`、`server/internal/modules/project/domain/actions.go`、`server/internal/modules/project/domain/errors.go`、`server/internal/modules/project/module.go`、`web/apps/web/helpers/authentication.helper.ts`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`server/internal/modules/project/adapter/postgres/gen/access.sql.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.9，M3 设计 3.4、3.19、6.5、8.2；裁定 S2）：`access.ProjectAccess`（`ProjectFacts(ctx, projectID, userID) (ProjectFacts, found bool, err)`）、`access.ProjectFacts{WorkspaceID, Public, Member, Role}`、`access.Deps.ProjectAccess`；`Authorize` 在 `Target.ProjectID` 不为零时读项目的事实：没有、或不是目标的工作区的项目算作没有（项目级的规则看不到它）。`project.Provide(pool)` 只交出 `ProjectAccess`（`ProjectMembershipCounts` 随它的第一个使用者 P5 加入）；`project.AccessFacts`；`bootstrap/ports.go` 的 `accessProjects`。
- `GET /api/v0/projects/{project_id}`，200 `Project`；码 `[project.not_found]`；规则 `project.read`：`LevelVisible`（看得到即可）；`app.NewGetProject(projects, auth)`：先按 id 读（带调用者的视角），再对它的工作区和项目判定；没有和看不到是同一个 404（8.2）；不开事务（6.7）。新码 `project.not_found` 的文案。
- 矩阵：`getProject` 的项目级 12 格和已归档项目的 1 格（`ofProject`：PA、PM、PG、PM+WA、WA-、WM-公 200，WM-私、WG-、P-前 和 X 是 `project.not_found`；答案核对项目、角色和 `archived_at`）。

**Tests:**
- `access/app/authorizer_test.go`：`TestAuthorizeReadsTheTargetsProject`（带项目的目标读它的事实，为调用者、在调用者的 ctx 里；项目角色进 `Grant`；工作区管理员看得到他不在的私密项目；别的工作区的、找不到的项目谁都看不到）；P1 的四个测试照新的构造。
- `project/adapter/postgres/access_test.go`：`TestProjectFacts`（公开项目的访客、私密项目的管理员、已归档项目的成员、已结束的、已删除的成员关系、没有成员关系、已删除的项目、没有项目）。
- `app/get_project_test.go`：`TestGetProject`、`TestGetProjectRefuses`；`adapter/http/projects_test.go`：`TestGetProject`；`bootstrap/ports_test.go`：`TestAccessProjectsConvertsProjectsAnswer`。
- `bootstrap/project_access_test.go`：`TestTheAuthorizerSeesNoProjectOfAnotherWorkspace`（`bootstrap` 接的 `Authorizer`，真实的存储：alice 是 `acme`、`beta` 的管理员、`acme` 的私密项目 Web 的管理员；经 `acme` 读 Web 得到她的项目管理员角色，经 `beta` 看不到它；第 8 条移交）。

- [ ] **Step 1: 接口描述和查询**

`api/modules/project.yaml`（修改，2 处）：

````old api/modules/project.yaml
          $ref: '#/components/responses/Problem'
````
````new api/modules/project.yaml
          $ref: '#/components/responses/Problem'
  /api/v0/projects/{project_id}:
    parameters:
      - $ref: '#/components/parameters/ProjectID'
    get:
      operationId: getProject
      tags: [project]
      summary: Read a project
      description: >-
        For whoever sees the project: its active members, the workspace's
        admins, and for a public project the workspace's members too; a
        caller who is not its member reads it with member_role and sort_order
        null. An archived project is read as any other, with its archived_at.
        A project that does not exist, is deleted, or that the caller does
        not see answers project.not_found; so does one whose workspace he is
        not an active member of.
      security: [{bearer: []}]
      x-problem-codes: [project.not_found]
      responses:
        '200':
          description: The project, as the caller sees it.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Project'
        default:
          $ref: '#/components/responses/Problem'
````

````old api/modules/project.yaml
      schema:
        type: string
````
````new api/modules/project.yaml
      schema:
        type: string
    ProjectID:
      name: project_id
      in: path
      required: true
      description: A project's id (Project.id).
      schema:
        type: string
        format: uuid
````

`api/openapi.yaml`（修改，1 处）：

````old api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1workspaces~1{slug}~1projects'
````
````new api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1workspaces~1{slug}~1projects'
  /api/v0/projects/{project_id}:
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1projects~1{project_id}'
````

`server/internal/modules/project/adapter/postgres/queries/access.sql`（新文件，8 行）：

````file server/internal/modules/project/adapter/postgres/queries/access.sql
-- name: ProjectFacts :one
-- access's ProjectAccess (M3 design 6.5): the undeleted project, archived or not, and the user's active membership of
-- it, if any.
SELECT p.workspace_id, p.network, m.role AS member_role
FROM projects p
LEFT JOIN project_members m
       ON m.project_id = p.id AND m.member_id = sqlc.arg(user_id) AND m.is_active AND m.deleted_at IS NULL
WHERE p.id = sqlc.arg(project_id) AND p.deleted_at IS NULL;
````

- [ ] **Step 2: 生成**

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `306d797a9b1b886107df8727f81a4184e299a9e1d2608530900ce3b545df6bca` | 1933 | `api/dist/openapi.yaml` |
| `6bfeae7299f88dd365a877cafe829a8cbbefa158ac5286d87876683dabe47e11` | 612 | `server/internal/modules/project/adapter/http/gen/server.gen.go` |
| `5a645efd0444e56a6d5cb27671e8cd449469bb0e02a5b1822e82603a276207a2` | 40 | `server/internal/modules/project/adapter/postgres/gen/access.sql.go` |
| `eefef639d6b839411c4963eeb34516c471746c4c121a5dcc3a5b160b4f0eb075` | 2009 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/server.gen.go server/internal/modules/project/adapter/postgres/gen/access.sql.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 3: `access` 读项目的事实；规则**

`server/internal/modules/access/app/ports.go`（修改，1 处）：

````old server/internal/modules/access/app/ports.go
}

````
````new server/internal/modules/access/app/ports.go
}

// ProjectFacts are what ProjectAccess reads of a project for a decision:
// its workspace, whether it is public, and whether the user asked about is
// its active member, with his project role then.
type ProjectFacts struct {
	WorkspaceID uuid.UUID
	Public      bool
	Member      bool
	Role        shared.Role
}

// ProjectAccess reads a project for a decision (M3 design 6.5). The project
// module implements it (project.Provide); bootstrap wires it, converting the
// facts.
type ProjectAccess interface {
	// ProjectFacts returns the facts of the undeleted project projectID,
	// archived or not, for userID; found is false for a project that does
	// not exist or is deleted. It reads in the transaction ctx carries, as
	// ActiveRole does.
	ProjectFacts(ctx context.Context, projectID, userID uuid.UUID) (f ProjectFacts, found bool, err error)
}

````

`server/internal/modules/access/app/authorizer.go`（修改，5 处）：

````old server/internal/modules/access/app/authorizer.go
	"fmt"
````
````new server/internal/modules/access/app/authorizer.go
	"fmt"
	"uuid"
````

````old server/internal/modules/access/app/authorizer.go
	roles WorkspaceRoles
````
````new server/internal/modules/access/app/authorizer.go
	roles    WorkspaceRoles
	projects ProjectAccess
````

````old server/internal/modules/access/app/authorizer.go
// NewAuthorizer returns the Authorizer that reads through roles.
func NewAuthorizer(roles WorkspaceRoles) *Authorizer {
	return &Authorizer{roles: roles}
````
````new server/internal/modules/access/app/authorizer.go
// NewAuthorizer returns the Authorizer that reads through roles and
// projects.
func NewAuthorizer(roles WorkspaceRoles, projects ProjectAccess) *Authorizer {
	return &Authorizer{roles: roles, projects: projects}
````

````old server/internal/modules/access/app/authorizer.go
// the caller's workspace membership; no project is read, since the rule
// table has no project-level row yet, so a project-level rule would see no
// project and answer shared.ErrNotVisible. The ProjectAccess port adds the
// project's facts with the projects (M3 design 6.5).
````
````new server/internal/modules/access/app/authorizer.go
// the caller's membership of t's workspace and, when t names a project, the
// project's: a project that is not there, or is another workspace's than
// t's, is none, and a project-level rule sees nothing (M3 design 3.4).
````

````old server/internal/modules/access/app/authorizer.go
	return domain.Decide(rule, domain.Facts{Workspace: domain.Membership{Active: active, Role: role}})
````
````new server/internal/modules/access/app/authorizer.go
	facts := domain.Facts{Workspace: domain.Membership{Active: active, Role: role}}
	if t.ProjectID != (uuid.UUID{}) {
		p, found, err := a.projects.ProjectFacts(ctx, t.ProjectID, actor.UserID)
		if err != nil {
			return shared.Grant{}, fmt.Errorf("access: read the project: %w", err)
		}
		if found && p.WorkspaceID == t.WorkspaceID {
			facts.Project = &domain.Project{Public: p.Public, Member: domain.Membership{Active: p.Member, Role: p.Role}}
		}
	}
	return domain.Decide(rule, facts)
````

`server/internal/modules/access/app/authorizer_test.go`（修改，9 处）：

````old server/internal/modules/access/app/authorizer_test.go
}

var (
````
````new server/internal/modules/access/app/authorizer_test.go
}

// projectKey is one (project, user) pair of fakeProjects.
type projectKey struct{ project, user uuid.UUID }

// fakeProjects answers the facts of each (project, user) pair it holds, and
// records every call as fakeRoles does.
type fakeProjects struct {
	facts map[projectKey]app.ProjectFacts
	err   error
	calls []string
}

func (f *fakeProjects) ProjectFacts(ctx context.Context, projectID, userID uuid.UUID) (app.ProjectFacts, bool, error) {
	value, ok := ctx.Value(ctxKey{}).(string)
	if !ok {
		value = "(none)"
	}
	f.calls = append(f.calls, projectID.String()+" "+userID.String()+" "+value)
	if f.err != nil {
		return app.ProjectFacts{}, false, f.err
	}
	p, ok := f.facts[projectKey{projectID, userID}]
	return p, ok, nil
}

var (
````

````old server/internal/modules/access/app/authorizer_test.go
	}}
	auth := app.NewAuthorizer(roles)
````
````new server/internal/modules/access/app/authorizer_test.go
	}}
	projects := &fakeProjects{}
	auth := app.NewAuthorizer(roles, projects)
````

````old server/internal/modules/access/app/authorizer_test.go
			t.Errorf("ActiveRole calls = %q, want [%q]", roles.calls, want)
		}
````
````new server/internal/modules/access/app/authorizer_test.go
			t.Errorf("ActiveRole calls = %q, want [%q]", roles.calls, want)
		}
		if len(projects.calls) != 0 {
			t.Errorf("a workspace-level target read the projects %q, want none", projects.calls)
		}
````

````old server/internal/modules/access/app/authorizer_test.go
func TestAuthorizeReadsOnEveryCall(t *testing.T) {
	roles := &fakeRoles{roles: map[membership]shared.Role{{w1, a}: shared.RoleAdmin}}
	auth := app.NewAuthorizer(roles)
	ctx := context.WithValue(context.Background(), ctxKey{}, "request")
````
````new server/internal/modules/access/app/authorizer_test.go
func TestAuthorizeReadsOnEveryCall(t *testing.T) {
	roles := &fakeRoles{roles: map[membership]shared.Role{{w1, a}: shared.RoleAdmin}}
	auth := app.NewAuthorizer(roles, &fakeProjects{})
	ctx := context.WithValue(context.Background(), ctxKey{}, "request")
````

````old server/internal/modules/access/app/authorizer_test.go
func TestAuthorizeRefusesAnActionWithoutARule(t *testing.T) {
	roles := &fakeRoles{roles: map[membership]shared.Role{{w1, a}: shared.RoleAdmin}}
	auth := app.NewAuthorizer(roles)
	ctx := context.WithValue(context.Background(), ctxKey{}, "request")
	grant, err := auth.Authorize(ctx, shared.Actor{UserID: a}, "no.such.action", shared.Target{WorkspaceID: w1})
````
````new server/internal/modules/access/app/authorizer_test.go
func TestAuthorizeRefusesAnActionWithoutARule(t *testing.T) {
	roles := &fakeRoles{roles: map[membership]shared.Role{{w1, a}: shared.RoleAdmin}}
	projects := &fakeProjects{}
	auth := app.NewAuthorizer(roles, projects)
	ctx := context.WithValue(context.Background(), ctxKey{}, "request")
	grant, err := auth.Authorize(ctx, shared.Actor{UserID: a}, "no.such.action", shared.Target{WorkspaceID: w1, ProjectID: uuid.NewV7()})
````

````old server/internal/modules/access/app/authorizer_test.go
	if len(roles.calls) != 0 {
		t.Errorf("ActiveRole calls = %q, want none", roles.calls)
````
````new server/internal/modules/access/app/authorizer_test.go
	if len(roles.calls)+len(projects.calls) != 0 {
		t.Errorf("ActiveRole calls = %q, ProjectFacts calls %q; want none", roles.calls, projects.calls)
````

````old server/internal/modules/access/app/authorizer_test.go
// The port's failure is Authorize's.
````
````new server/internal/modules/access/app/authorizer_test.go
// Each port's failure is Authorize's.
````

````old server/internal/modules/access/app/authorizer_test.go
	failure := errors.New("connection reset")
	auth := app.NewAuthorizer(&fakeRoles{err: failure})
	ctx := context.WithValue(context.Background(), ctxKey{}, "request")
````
````new server/internal/modules/access/app/authorizer_test.go
	failure := errors.New("connection reset")
	ctx := context.WithValue(context.Background(), ctxKey{}, "request")
	auth := app.NewAuthorizer(&fakeRoles{err: failure}, &fakeProjects{})
````

````old server/internal/modules/access/app/authorizer_test.go
		t.Errorf("Authorize() = %v, want %v", err, failure)
	}
}

````
````new server/internal/modules/access/app/authorizer_test.go
		t.Errorf("Authorize() with the roles failing = %v, want %v", err, failure)
	}
	auth = app.NewAuthorizer(&fakeRoles{roles: map[membership]shared.Role{{w1, a}: shared.RoleAdmin}}, &fakeProjects{err: failure})
	if _, err := auth.Authorize(ctx, shared.Actor{UserID: a}, "project.read", shared.Target{WorkspaceID: w1, ProjectID: uuid.NewV7()}); !errors.Is(err, failure) {
		t.Errorf("Authorize() with the projects failing = %v, want %v", err, failure)
	}
}

// A target that names a project has the project's facts read too, for the
// caller and in the caller's context, and the decision takes them: the
// caller's project role is in the Grant, a workspace admin sees a private
// project he is not in. A project of another workspace than the target's,
// or one not found, is seen by no one: its facts do not count.
func TestAuthorizeReadsTheTargetsProject(t *testing.T) {
	p1, p2, gone := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	roles := &fakeRoles{roles: map[membership]shared.Role{{w1, a}: shared.RoleMember, {w1, b}: shared.RoleAdmin}}
	projects := &fakeProjects{facts: map[projectKey]app.ProjectFacts{
		{p1, a}: {WorkspaceID: w1, Member: true, Role: shared.RoleGuest},
		{p1, b}: {WorkspaceID: w1},
		{p2, a}: {WorkspaceID: w2, Public: true, Member: true, Role: shared.RoleAdmin},
	}}
	auth := app.NewAuthorizer(roles, projects)
	tests := []struct {
		user, project uuid.UUID
		want          *shared.Grant // nil: not visible
	}{
		{a, p1, &shared.Grant{WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleGuest}},
		{b, p1, &shared.Grant{WorkspaceRole: shared.RoleAdmin}},
		{a, p2, nil},
		{a, gone, nil},
	}
	for _, tt := range tests {
		projects.calls = nil
		ctx := context.WithValue(context.Background(), ctxKey{}, "tx")
		grant, err := auth.Authorize(ctx, shared.Actor{UserID: tt.user}, "project.read", shared.Target{WorkspaceID: w1, ProjectID: tt.project})
		if want := tt.project.String() + " " + tt.user.String() + " tx"; len(projects.calls) != 1 || projects.calls[0] != want {
			t.Errorf("ProjectFacts calls = %q, want [%q]", projects.calls, want)
		}
		if tt.want == nil {
			if !errors.Is(err, shared.ErrNotVisible) {
				t.Errorf("user %s, project %s: Authorize() = %+v, %v; want ErrNotVisible", tt.user, tt.project, grant, err)
			}
			continue
		}
		if err != nil || grant != *tt.want {
			t.Errorf("user %s, project %s: Authorize() = %+v, %v; want %+v", tt.user, tt.project, grant, err, *tt.want)
		}
	}
}

````

`server/internal/modules/access/module.go`（修改，2 处）：

````old server/internal/modules/access/module.go
	WorkspaceRoles app.WorkspaceRoles
}
````
````new server/internal/modules/access/module.go
	WorkspaceRoles app.WorkspaceRoles
	// ProjectAccess is project's ProjectAccess, converted
	// (bootstrap/ports.go).
	ProjectAccess app.ProjectAccess
}

// ProjectFacts are the facts the ProjectAccess port hands over: bootstrap
// converts project's into them (M3 design 6.5).
type ProjectFacts = app.ProjectFacts
````

````old server/internal/modules/access/module.go
	return app.NewAuthorizer(d.WorkspaceRoles)
````
````new server/internal/modules/access/module.go
	return app.NewAuthorizer(d.WorkspaceRoles, d.ProjectAccess)
````

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"project.create": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
````
````new server/internal/modules/access/domain/rules.go
	"project.create": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
	// Whoever sees the project (M3 design 3.4, 3.19).
	"project.read": {Level: LevelVisible},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"project.create":               {allowed, allowed, forbidden, invisible, invisible, invisible, forbidden},
````
````new server/internal/modules/access/domain/rules_test.go
	"project.create":               {allowed, allowed, forbidden, invisible, invisible, invisible, forbidden},
	// PA, PM, PG, WM demoted to PG, PM+WA, WA- private, WM- public, WM- private, WG- public, WG- private, P-before,
	// P-before public, X, workspace role outside the three public, above the three public, project role outside the
	// three, workspace role outside the three project admin (projectIdentities)
	"project.read": {allowed, allowed, allowed, allowed, allowed, allowed, allowed, invisible, invisible, invisible, invisible, allowed,
		invisible, invisible, invisible, forbidden, forbidden},
````

- [ ] **Step 4: `project` 的事实和 `getProject`**

`server/internal/modules/project/domain/actions.go`（修改，2 处）：

````old server/internal/modules/project/domain/actions.go
	ActionCreate shared.Action = "project.create"
````
````new server/internal/modules/project/domain/actions.go
	ActionCreate shared.Action = "project.create"
	// ActionRead is reading a project: getProject.
	ActionRead shared.Action = "project.read"
````

````old server/internal/modules/project/domain/actions.go
	return []shared.Action{ActionCreate}
````
````new server/internal/modules/project/domain/actions.go
	return []shared.Action{ActionCreate, ActionRead}
````

`server/internal/modules/project/domain/errors.go`（修改，1 处）：

````old server/internal/modules/project/domain/errors.go
	ErrNameTaken = shared.NewError(shared.KindConflict, "project.name_taken", "A project of the workspace has this name.")
````
````new server/internal/modules/project/domain/errors.go
	ErrNameTaken = shared.NewError(shared.KindConflict, "project.name_taken", "A project of the workspace has this name.")
	// ErrNotFound answers a project that does not exist, is deleted, or that
	// the caller does not see (M3 design 3.4, 8.2).
	ErrNotFound = shared.NewError(shared.KindNotFound, "project.not_found", "The project does not exist, or you cannot see it.")
````

`server/internal/modules/project/app/access.go`（新文件，18 行）：

````file server/internal/modules/project/app/access.go
package app

import (
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// AccessFacts are what project.Provide's ProjectAccess answers the access
// module (M3 design 6.5): the project's workspace, whether it is public, and
// whether the user asked about is its active member, with his project role
// then. bootstrap converts them into access's value.
type AccessFacts struct {
	WorkspaceID uuid.UUID
	Public      bool
	Member      bool
	Role        shared.Role
}
````

`server/internal/modules/project/app/ports.go`（修改，3 处）：

````old server/internal/modules/project/app/ports.go
}

// ProjectCreator is createProject's repository. Each method runs in the
````
````new server/internal/modules/project/app/ports.go
}

// ProjectReader reads a project as a user sees it.
type ProjectReader interface {
	// GetProject is the undeleted project id as userID sees it; found is
	// false when there is none.
	GetProject(ctx context.Context, id, userID uuid.UUID) (p domain.Project, found bool, err error)
}

// ProjectCreator is createProject's repository. Each method runs in the
````

````old server/internal/modules/project/app/ports.go
type ProjectCreator interface {
````
````new server/internal/modules/project/app/ports.go
type ProjectCreator interface {
	ProjectReader
````

````old server/internal/modules/project/app/ports.go
	CreateStates(ctx context.Context, rows []StateRow) error
	// GetProject is the undeleted project id as userID sees it; found is
	// false when there is none.
	GetProject(ctx context.Context, id, userID uuid.UUID) (p domain.Project, found bool, err error)
````
````new server/internal/modules/project/app/ports.go
	CreateStates(ctx context.Context, rows []StateRow) error
````

`server/internal/modules/project/app/get_project.go`（新文件，49 行）：

````file server/internal/modules/project/app/get_project.go
package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// GetProject reads a project: GET /api/v0/projects/{project_id} (M3 design
// 3.19).
type GetProject struct {
	projects ProjectReader
	auth     shared.Authorizer
}

// NewGetProject returns the use case.
func NewGetProject(projects ProjectReader, auth shared.Authorizer) *GetProject {
	return &GetProject{projects: projects, auth: auth}
}

// Execute answers the undeleted project id, archived or not, as the caller
// sees it, when he may read it. It reads the project first, which names its
// workspace, then decides project.read on it; a project that is not there
// and one the caller does not see answer the same domain.ErrNotFound (M3
// design 8.2). A read opens no transaction (M3 design 6.7).
func (u *GetProject) Execute(ctx context.Context, id uuid.UUID) (domain.Project, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Project{}, err
	}
	p, found, err := u.projects.GetProject(ctx, id, actor.UserID)
	switch {
	case err != nil:
		return domain.Project{}, err
	case !found:
		return domain.Project{}, domain.ErrNotFound
	}
	_, err = u.auth.Authorize(ctx, actor, domain.ActionRead, shared.Target{WorkspaceID: p.WorkspaceID, ProjectID: p.ID})
	switch {
	case errors.Is(err, shared.ErrNotVisible):
		return domain.Project{}, domain.ErrNotFound
	case err != nil:
		return domain.Project{}, err
	}
	return p, nil
}
````

`server/internal/modules/project/app/get_project_test.go`（新文件，90 行）：

````file server/internal/modules/project/app/get_project_test.go
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

// fakeReader holds projects by id, logs each read and fails with err.
type fakeReader struct {
	log      *callLog
	projects map[uuid.UUID]domain.Project
	err      error
}

func (f *fakeReader) GetProject(ctx context.Context, id, userID uuid.UUID) (domain.Project, bool, error) {
	f.log.add(ctx, "GetProject %s for %s", id, userID)
	p, found := f.projects[id]
	return p, found && f.err == nil, f.err
}

// newGet is GetProject over fakes sharing one log, and web, acme's
// project, which alice, acme's member, sees, and dave does not.
func newGet() (*app.GetProject, *fakeReader, *fakeAuthorizer, domain.Project) {
	log := &callLog{}
	web := domain.Project{ID: uuid.NewV7(), WorkspaceID: acme.ID, Name: "Web", MemberRole: ptr(shared.RoleAdmin)}
	reader := &fakeReader{log: log, projects: map[uuid.UUID]domain.Project{web.ID: web}}
	auth := &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{{alice, acme.ID}: {WorkspaceRole: shared.RoleMember}}}
	return app.NewGetProject(reader, auth), reader, auth, web
}

// GetProject reads the project as the caller sees it, outside any
// transaction, then decides project.read on it, in its workspace: the
// answer is the project as read.
func TestGetProject(t *testing.T) {
	uc, reader, _, web := newGet()

	got, err := uc.Execute(as(alice), web.ID)

	want := []string{fmt.Sprintf("GetProject %s for %s outside tx", web.ID, alice),
		fmt.Sprintf("Authorize %s project.read on %s/%s outside tx", alice, acme.ID, web.ID)}
	if err != nil || got.ID != web.ID || got.MemberRole == nil || *got.MemberRole != shared.RoleAdmin || !slices.Equal(reader.log.calls, want) {
		t.Errorf("Execute() = %+v, %v, calls %q; want the project as read, calls %q", got, err, reader.log.calls, want)
	}
}

// A project not there and one the caller does not see answer the same
// project.not_found; the Authorizer's other refusals and every failure
// come back as themselves; without a caller nothing is read.
func TestGetProjectRefuses(t *testing.T) {
	failure := errors.New("connection reset")
	tests := []struct {
		name     string
		ctx      context.Context
		another  bool  // asks for another id than web's
		readErr  error // the read's failure
		alicesTo error // the Authorizer's answer to alice
		want     error
		calls    int
	}{
		{"no project", as(alice), true, nil, nil, domain.ErrNotFound, 1},
		{"not seen", as(dave), false, nil, nil, domain.ErrNotFound, 2},
		{"forbidden", as(alice), false, nil, shared.Forbidden(), shared.Forbidden(), 2},
		{"the decision failing", as(alice), false, nil, failure, failure, 2},
		{"the read failing", as(alice), false, failure, nil, failure, 1},
		{"no caller", context.Background(), false, nil, nil, shared.Unauthenticated(), 0},
	}
	for _, tt := range tests {
		uc, reader, auth, web := newGet()
		reader.err = tt.readErr
		if tt.alicesTo != nil {
			auth.errs = map[grantKey]error{{alice, acme.ID}: tt.alicesTo}
		}
		id := web.ID
		if tt.another {
			id = uuid.NewV7()
		}
		got, err := uc.Execute(tt.ctx, id)
		if !errors.Is(err, tt.want) || got.ID != (uuid.UUID{}) || len(reader.log.calls) != tt.calls {
			t.Errorf("%s: Execute() = %+v, %v, calls %q; want %v after %d calls", tt.name, got, err, reader.log.calls, tt.want, tt.calls)
		}
	}
}
````

`server/internal/modules/project/adapter/postgres/access.go`（新文件，34 行）：

````file server/internal/modules/project/adapter/postgres/access.go
package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ProjectFacts is what project.Provide offers the access module (M3 design
// 6.5): the facts of the undeleted project projectID, archived or not, for
// userID; found is false for a project that does not exist or is deleted.
// It reads in the transaction ctx carries.
func (s *Store) ProjectFacts(ctx context.Context, projectID, userID uuid.UUID) (f app.AccessFacts, found bool, err error) {
	r, err := s.queries(ctx).ProjectFacts(ctx, gen.ProjectFactsParams{ProjectID: projectID, UserID: userID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return app.AccessFacts{}, false, nil
	case err != nil:
		return app.AccessFacts{}, false, fmt.Errorf("read project %s for a decision: %w", projectID, err)
	}
	f = app.AccessFacts{WorkspaceID: r.WorkspaceID, Public: domain.Network(r.Network) == domain.NetworkPublic}
	if r.MemberRole != nil {
		f.Member, f.Role = true, shared.Role(*r.MemberRole)
	}
	return f, true, nil
}
````

`server/internal/modules/project/adapter/postgres/access_test.go`（新文件，62 行）：

````file server/internal/modules/project/adapter/postgres/access_test.go
package postgresadapter_test

import (
	"context"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ProjectFacts reads the undeleted project, archived or not, and the user's
// membership of it while it is active: a membership ended or deleted is
// none, a project deleted is not found. Each case is one fact changed from
// a project where the user is its member.
func TestProjectFacts(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, pool, "acme")
	public := newProject(t, s, acme, "Web", "WEB", alice)
	private := newProject(t, s, acme, "Secret", "SEC", alice)
	exec(t, pool, "UPDATE projects SET network = 0 WHERE id = $1", private)
	archived := newProject(t, s, acme, "Old", "OLD", alice)
	exec(t, pool, "UPDATE projects SET archived_at = now() WHERE id = $1", archived)
	deleted := newProject(t, s, acme, "Gone", "GONE", alice)
	exec(t, pool, "UPDATE projects SET deleted_at = now() WHERE id = $1", deleted)
	for _, m := range []struct {
		project, user uuid.UUID
		role          shared.Role
	}{{public, alice, shared.RoleGuest}, {private, alice, shared.RoleAdmin}, {archived, alice, shared.RoleMember},
		{deleted, alice, shared.RoleAdmin}, {public, bob, shared.RoleAdmin}, {private, bob, shared.RoleMember}} {
		if err := s.CreateMember(ctx, app.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: m.project, MemberID: m.user, Role: m.role,
			CreatedBy: alice, Now: now}); err != nil {
			t.Fatal(err)
		}
	}
	exec(t, pool, "UPDATE project_members SET is_active = false WHERE project_id = $1 AND member_id = $2", public, bob)
	exec(t, pool, "UPDATE project_members SET deleted_at = now() WHERE project_id = $1 AND member_id = $2", private, bob)

	tests := []struct {
		name          string
		project, user uuid.UUID
		want          app.AccessFacts
		found         bool
	}{
		{"a public project's guest", public, alice, app.AccessFacts{WorkspaceID: acme, Public: true, Member: true, Role: shared.RoleGuest}, true},
		{"a private project's admin", private, alice, app.AccessFacts{WorkspaceID: acme, Member: true, Role: shared.RoleAdmin}, true},
		{"an archived project's member", archived, alice, app.AccessFacts{WorkspaceID: acme, Public: true, Member: true, Role: shared.RoleMember}, true},
		{"a membership ended", public, bob, app.AccessFacts{WorkspaceID: acme, Public: true}, true},
		{"a membership deleted", private, bob, app.AccessFacts{WorkspaceID: acme}, true},
		{"no membership", archived, bob, app.AccessFacts{WorkspaceID: acme, Public: true}, true},
		{"a project deleted", deleted, alice, app.AccessFacts{}, false},
		{"no project", uuid.NewV7(), alice, app.AccessFacts{}, false},
	}
	for _, tt := range tests {
		got, found, err := s.ProjectFacts(ctx, tt.project, tt.user)
		if err != nil || found != tt.found || got != tt.want {
			t.Errorf("%s: ProjectFacts() = %+v, %v, %v; want %+v, %v", tt.name, got, found, err, tt.want, tt.found)
		}
	}
}
````

`server/internal/modules/project/adapter/http/handler.go`（修改，3 处）：

````old server/internal/modules/project/adapter/http/handler.go
	"context"
````
````new server/internal/modules/project/adapter/http/handler.go
	"context"
	"uuid"
````

````old server/internal/modules/project/adapter/http/handler.go
}

// UseCases are the use cases behind the module's operations.
````
````new server/internal/modules/project/adapter/http/handler.go
}

// GetProjectUseCase is app.GetProject.
type GetProjectUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) (domain.Project, error)
}

// UseCases are the use cases behind the module's operations.
````

````old server/internal/modules/project/adapter/http/handler.go
	CreateProject CreateProjectUseCase
````
````new server/internal/modules/project/adapter/http/handler.go
	CreateProject CreateProjectUseCase
	GetProject    GetProjectUseCase
````

`server/internal/modules/project/adapter/http/projects.go`（修改，1 处）：

````old server/internal/modules/project/adapter/http/projects.go
	return gen.CreateProject201JSONResponse(project(p)), nil
````
````new server/internal/modules/project/adapter/http/projects.go
	return gen.CreateProject201JSONResponse(project(p)), nil
}

// GetProject serves GET /api/v0/projects/{project_id}.
func (h handler) GetProject(ctx context.Context, req gen.GetProjectRequestObject) (gen.GetProjectResponseObject, error) {
	p, err := h.uc.GetProject.Execute(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}
	return gen.GetProject200JSONResponse(project(p)), nil
````

`server/internal/modules/project/adapter/http/handler_test.go`（修改，3 处）：

````old server/internal/modules/project/adapter/http/handler_test.go
	create *fakeCreate
````
````new server/internal/modules/project/adapter/http/handler_test.go
	create *fakeCreate
	get    *fakeGet
````

````old server/internal/modules/project/adapter/http/handler_test.go
	return f.answer, f.err
````
````new server/internal/modules/project/adapter/http/handler_test.go
	return f.answer, f.err
}

type fakeGet struct {
	calls    []string // "caller id"
	projects map[string]domain.Project
}

// Execute answers the project the caller's key names, project.not_found
// for any other.
func (f *fakeGet) Execute(ctx context.Context, id uuid.UUID) (domain.Project, error) {
	key := caller(ctx) + " " + id.String()
	f.calls = append(f.calls, key)
	p, ok := f.projects[key]
	if !ok {
		return domain.Project{}, domain.ErrNotFound
	}
	return p, nil
````

````old server/internal/modules/project/adapter/http/handler_test.go
	httpadapter.Register(router, api, httpadapter.UseCases{CreateProject: f.create})
````
````new server/internal/modules/project/adapter/http/handler_test.go
	if f.get == nil {
		f.get = &fakeGet{}
	}
	httpadapter.Register(router, api, httpadapter.UseCases{CreateProject: f.create, GetProject: f.get})
````

`server/internal/modules/project/adapter/http/projects_test.go`（修改，1 处）：

````old server/internal/modules/project/adapter/http/projects_test.go
		t.Errorf("in_use other: POST = %d, inputs %+v; want it passed to the use case", res.StatusCode, create.got)
	}
}

````
````new server/internal/modules/project/adapter/http/projects_test.go
		t.Errorf("in_use other: POST = %d, inputs %+v; want it passed to the use case", res.StatusCode, create.got)
	}
}

// The path's id goes to the use case for the caller; a project it does not
// find, or the caller does not see, is project.not_found.
func TestGetProject(t *testing.T) {
	get := &fakeGet{projects: map[string]domain.Project{"alice " + webID.String(): web, "bob " + webID.String(): bare}}
	h := newServer(t, fakes{get: get})
	tests := []struct {
		token, id string
		status    int
		want      string
	}{
		{"alice", webID.String(), http.StatusOK, webJSON},
		{"bob", webID.String(), http.StatusOK, bareJSON},
		{"alice", acmeID.String(), http.StatusNotFound,
			`{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`},
	}
	for _, tt := range tests {
		res, body := do(t, h, request(http.MethodGet, "/api/v0/projects/"+tt.id, tt.token, ""))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s GET %s = %d %s, want %d %s", tt.token, tt.id, res.StatusCode, body, tt.status, tt.want)
		}
	}
	if want := []string{"alice " + webID.String(), "bob " + webID.String(), "alice " + acmeID.String()}; !slices.Equal(get.calls, want) {
		t.Errorf("calls = %q, want %q", get.calls, want)
	}
}

````

`server/internal/modules/project/module.go`（修改，3 处）：

````old server/internal/modules/project/module.go
// brings creating projects, and carries out the workspace module's cascades
// on the projects (ProjectCascade).
````
````new server/internal/modules/project/module.go
// brings creating and reading projects, carries out the workspace module's
// cascades on the projects (ProjectCascade), and offers the access module
// its reads of a project (ProjectAccess).
````

````old server/internal/modules/project/module.go
	DeleteWorkspaceProjects(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error
````
````new server/internal/modules/project/module.go
	DeleteWorkspaceProjects(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error
}

// ProjectAccess reads a project for the access module's decision (M3
// design 6.5): the facts of the undeleted project projectID, archived or
// not, for userID; found is false for a project that does not exist or is
// deleted. It reads in the transaction ctx carries.
type ProjectAccess interface {
	ProjectFacts(ctx context.Context, projectID, userID uuid.UUID) (f AccessFacts, found bool, err error)
}

// AccessFacts are the facts ProjectAccess hands over: bootstrap converts
// them into access's value (M3 design 6.5).
type AccessFacts = app.AccessFacts

// Provided are the adapters project offers the other modules. They depend
// on the pool alone, so bootstrap builds them before any module (M3 design
// 6.6, step 2).
type Provided struct {
	ProjectAccess ProjectAccess
}

// Provide builds project's adapters for the other modules.
func Provide(pool *pgxpool.Pool) Provided {
	return Provided{ProjectAccess: postgresadapter.New(pool)}
````

````old server/internal/modules/project/module.go
		}),
````
````new server/internal/modules/project/module.go
		}),
		GetProject: app.NewGetProject(store, d.Authorizer),
````

- [ ] **Step 5: 组合根和矩阵**

`server/internal/bootstrap/ports.go`（修改，2 处）：

````old server/internal/bootstrap/ports.go
	"uuid"

````
````new server/internal/bootstrap/ports.go
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
````

````old server/internal/bootstrap/ports.go
	return project.Workspace(w), found, err
}

````
````new server/internal/bootstrap/ports.go
	return project.Workspace(w), found, err
}

// accessProjects is project's ProjectAccess as access's port: the same read,
// the facts converted.
type accessProjects struct {
	projects project.ProjectAccess
}

func (a accessProjects) ProjectFacts(ctx context.Context, projectID, userID uuid.UUID) (access.ProjectFacts, bool, error) {
	f, found, err := a.projects.ProjectFacts(ctx, projectID, userID)
	return access.ProjectFacts(f), found, err
}

````

`server/internal/bootstrap/ports_test.go`（修改，2 处）：

````old server/internal/bootstrap/ports_test.go
	"uuid"

````
````new server/internal/bootstrap/ports_test.go
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
````

````old server/internal/bootstrap/ports_test.go
		t.Errorf("ShareWorkspaceBySlug() = %v, want %v", err, failure)
	}
}

````
````new server/internal/bootstrap/ports_test.go
		t.Errorf("ShareWorkspaceBySlug() = %v, want %v", err, failure)
	}
}

// fakeProjectAccess answers the facts it holds by project, and records what
// it was asked.
type fakeProjectAccess struct {
	facts map[uuid.UUID]project.AccessFacts
	err   error
	asked []string
}

func (f *fakeProjectAccess) ProjectFacts(_ context.Context, projectID, userID uuid.UUID) (project.AccessFacts, bool, error) {
	f.asked = append(f.asked, projectID.String()+" "+userID.String())
	p, found := f.facts[projectID]
	return p, found, f.err
}

// accessProjects hands access project's answer to the same question: the
// facts converted, found and the error as they came.
func TestAccessProjectsConvertsProjectsAnswer(t *testing.T) {
	web, acme, alice := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	fake := &fakeProjectAccess{facts: map[uuid.UUID]project.AccessFacts{web: {WorkspaceID: acme, Public: true, Member: true, Role: 15}}}
	a := accessProjects{projects: fake}
	ctx := context.Background()

	f, found, err := a.ProjectFacts(ctx, web, alice)
	if want := (access.ProjectFacts{WorkspaceID: acme, Public: true, Member: true, Role: 15}); err != nil || !found || f != want {
		t.Errorf("ProjectFacts(web) = %+v, %v, %v; want %+v, found", f, found, err, want)
	}
	gone := uuid.NewV7()
	if f, found, err := a.ProjectFacts(ctx, gone, alice); err != nil || found || f != (access.ProjectFacts{}) {
		t.Errorf("ProjectFacts(gone) = %+v, %v, %v; want not found", f, found, err)
	}
	if want := []string{web.String() + " " + alice.String(), gone.String() + " " + alice.String()}; !slices.Equal(fake.asked, want) {
		t.Errorf("project was asked %q, want %q", fake.asked, want)
	}
	failure := errors.New("connection reset")
	fake.err = failure
	if _, _, err := a.ProjectFacts(ctx, web, alice); !errors.Is(err, failure) {
		t.Errorf("ProjectFacts() = %v, want %v", err, failure)
	}
}

````

`server/internal/bootstrap/app.go`（修改，1 处）：

````old server/internal/bootstrap/app.go
	authorizer := access.New(access.Deps{WorkspaceRoles: workspacePorts.WorkspaceRoles})
````
````new server/internal/bootstrap/app.go
	projectPorts := project.Provide(pool)
	authorizer := access.New(access.Deps{
		WorkspaceRoles: workspacePorts.WorkspaceRoles,
		ProjectAccess:  accessProjects{projects: projectPorts.ProjectAccess},
	})
````

`server/internal/bootstrap/project_access_test.go`（新文件，64 行）：

````file server/internal/bootstrap/project_access_test.go
package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// The Authorizer takes a project's facts within the target's workspace only
// (M3 design 3.4; carry 8), as bootstrap wires it on the real stores: alice
// is the admin of acme and of beta, and the admin of acme's private project
// Web. Through acme she reads Web as its admin; through beta, where every
// project is hers to see, she does not see it.
func TestTheAuthorizerSeesNoProjectOfAnotherWorkspace(t *testing.T) {
	contract := apitest.Load(t)
	dbURL := pgtest.NewDatabase(t)
	base := startApp(t, testConfig(t, dbURL, false), migrations.FS())
	alice := registerAccount(t, contract, base, "alice@example.com").AccessToken
	for _, slug := range []string{"acme", "beta"} {
		if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", alice,
			`{"name":"`+slug+`","slug":"`+slug+`"}`); status != http.StatusCreated {
			t.Fatalf("creating %s = %d %s", slug, status, body)
		}
	}
	status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces/acme/projects", alice,
		`{"name":"Web","identifier":"WEB","network":0}`)
	if status != http.StatusCreated {
		t.Fatalf("creating Web = %d %s", status, body)
	}
	var web struct {
		ID          uuid.UUID `json:"id"`
		WorkspaceID uuid.UUID `json:"workspace_id"`
	}
	decodeAnswer(t, body, &web)
	pool := openPool(t, dbURL)
	var beta uuid.UUID
	if err := pool.QueryRow(context.Background(), "SELECT id FROM workspaces WHERE slug = 'beta'").Scan(&beta); err != nil {
		t.Fatal(err)
	}
	authorizer := access.New(access.Deps{
		WorkspaceRoles: workspace.Provide(pool).WorkspaceRoles,
		ProjectAccess:  accessProjects{projects: project.Provide(pool).ProjectAccess},
	})
	actor, read := shared.Actor{UserID: accountID(t, contract, base, alice)}, shared.Action("project.read")

	g, err := authorizer.Authorize(context.Background(), actor, read, shared.Target{WorkspaceID: web.WorkspaceID, ProjectID: web.ID})
	if err != nil || g.ProjectRole != shared.RoleAdmin {
		t.Errorf("project.read on Web through acme = %+v, %v; want her grant as its admin", g, err)
	}
	g, err = authorizer.Authorize(context.Background(), actor, read, shared.Target{WorkspaceID: beta, ProjectID: web.ID})
	if !errors.Is(err, shared.ErrNotVisible) {
		t.Errorf("project.read on Web through beta = %+v, %v; want %v", g, err, shared.ErrNotVisible)
	}
}
````

`server/internal/bootstrap/permission_matrix_columns_test.go`（修改，6 处）：

````old server/internal/bootstrap/permission_matrix_columns_test.go
	callerBefore         caller = "project member before"              // P-前: ended in the private project
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
	callerBefore         caller = "project member before"              // P-前: ended in the private project
	// callerArchivedAdmin is PA on acme's archived project: the column of
	// 9.2's small table of the archived project.
	callerArchivedAdmin caller = "archived project admin"
````

````old server/internal/bootstrap/permission_matrix_columns_test.go
	callerMemberPublic, callerMemberPrivate, callerGuestOnly, callerBefore, callerNever, callerRemoved, callerDeleted}
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
	callerMemberPublic, callerMemberPrivate, callerGuestOnly, callerBefore, callerNever, callerRemoved, callerDeleted}

// archivedColumns are the columns of the archived project's table (9.2).
var archivedColumns = []caller{callerArchivedAdmin}
````

````old server/internal/bootstrap/permission_matrix_columns_test.go
func accountOf(c caller) caller {
	switch c {
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
func accountOf(c caller) caller {
	switch c {
	case callerArchivedAdmin:
		return callerProjectAdmin
````

````old server/internal/bootstrap/permission_matrix_columns_test.go
// acme's private project for the columns about it, gone's for the deleted
// workspace's, acme's public one for every other; PA, PM, PG and PM+WA
// answer alike in both, WG- too (9.2).
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
// acme's private project for the columns about it, its archived one for
// the archived project's, gone's for the deleted workspace's, acme's public
// one for every other; PA, PM, PG and PM+WA answer alike in both, WG- too
// (9.2).
````

````old server/internal/bootstrap/permission_matrix_columns_test.go
		return "acme/private"
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
		return "acme/private"
	case callerArchivedAdmin:
		return "acme/archived"
````

````old server/internal/bootstrap/permission_matrix_columns_test.go
	for _, c := range slices.Concat(workspaceColumns, projectColumns) {
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
	for _, c := range slices.Concat(workspaceColumns, projectColumns, archivedColumns) {
````

`server/internal/bootstrap/permission_matrix_project_test.go`（修改，3 处）：

````old server/internal/bootstrap/permission_matrix_project_test.go
	"testing"
````
````new server/internal/bootstrap/permission_matrix_project_test.go
	"testing"
	"time"
````

````old server/internal/bootstrap/permission_matrix_project_test.go
// The project module's rows of the permission matrix (M3 design 9.2).
````
````new server/internal/bootstrap/permission_matrix_project_test.go
// The project module's rows of the permission matrix (M3 design 9.2).

var cellProjectNotFound = cell{http.StatusNotFound, "project.not_found"}

// ofProject are the cells of a project-level row: the answers of PA, PM,
// PG, PM+WA, WA- and WM-公, and project.not_found for the columns that do
// not see their project: WM-私, WG-, P-前 and X.
func ofProject(pa, pm, pg, pmwa, wa, wm cell) map[caller]cell {
	return map[caller]cell{callerProjectAdmin: pa, callerProjectMember: pm, callerProjectGuest: pg, callerMemberAndAdmin: pmwa,
		callerAdminOnly: wa, callerMemberPublic: wm, callerMemberPrivate: cellProjectNotFound, callerGuestOnly: cellProjectNotFound,
		callerBefore: cellProjectNotFound, callerNever: cellProjectNotFound, callerRemoved: cellProjectNotFound,
		callerDeleted: cellProjectNotFound}
}

// toProject is the request of a row whose callers each send method to the
// path under the project their column targets.
func toProject(method, path, body string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return method, "/api/v0/projects/" + s.project(projectOf(c)).String() + path, body
	}
}
````

````old server/internal/bootstrap/permission_matrix_project_test.go
			cells:   inWorkspace(cellValidationFailed, cellValidationFailed, cellForbidden)},
````
````new server/internal/bootstrap/permission_matrix_project_test.go
			cells:   inWorkspace(cellValidationFailed, cellValidationFailed, cellForbidden)},
		{op: "getProject", columns: projectColumns, request: toProject(http.MethodGet, "", ""),
			cells: ofProject(cellOK, cellOK, cellOK, cellOK, cellOK, cellOK), check: readsItsProject},
		{op: "getProject", variant: "archived", columns: archivedColumns, request: toProject(http.MethodGet, "", ""),
			cells: map[caller]cell{callerArchivedAdmin: cellOK}, check: readsItsProject},
	}
}

// readsItsProject: the column's project, with the caller's role in it, null
// for who is not its member, and its archived_at, set for the archived one
// alone.
func readsItsProject(t *testing.T, c caller, s seeded, answer string) {
	var p struct {
		ID         uuid.UUID  `json:"id"`
		MemberRole *int       `json:"member_role"`
		ArchivedAt *time.Time `json:"archived_at"`
	}
	decodeAnswer(t, answer, &p)
	roles := map[caller]int{callerProjectAdmin: 20, callerProjectMember: 15, callerProjectGuest: 5, callerMemberAndAdmin: 15,
		callerArchivedAdmin: 20}
	role, member := roles[c]
	if p.ID != s.project(projectOf(c)) || (p.MemberRole != nil) != member || (member && *p.MemberRole != role) ||
		(p.ArchivedAt != nil) != (c == callerArchivedAdmin) {
		t.Errorf("%s reads %s; want %s, his role %d (0: none), archived only for the archived project", c, answer, projectOf(c), role)
````

- [ ] **Step 6: 新码的文案**

`web/apps/web/helpers/authentication.helper.ts`（修改，1 处）：

````old web/apps/web/helpers/authentication.helper.ts
  "project.name_taken": "auth.errors.project_name_taken",
````
````new web/apps/web/helpers/authentication.helper.ts
  "project.name_taken": "auth.errors.project_name_taken",
  "project.not_found": "auth.errors.project_not_found",
````

`web/packages/i18n/src/locales/en/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/auth.json
      "project_name_taken": "A project of this workspace already has this name.",
````
````new web/packages/i18n/src/locales/en/auth.json
      "project_name_taken": "A project of this workspace already has this name.",
      "project_not_found": "The project does not exist, or you cannot see it.",
````

`web/packages/i18n/src/locales/zh-CN/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/auth.json
      "project_name_taken": "这个工作区已有项目使用这个名称。",
````
````new web/packages/i18n/src/locales/zh-CN/auth.json
      "project_name_taken": "这个工作区已有项目使用这个名称。",
      "project_not_found": "项目不存在，或你看不到它。",
````

- [ ] **Step 7: 测试、lint、前端检查**

Run: `go -C server test -count=1 ./internal/modules/access/... ./internal/modules/project/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestAccessProjectsConvertsProjectsAnswer|TestTheAuthorizerSeesNoProjectOfAnotherWorkspace|TestPermissionMatrix$|TestThePermissionMatrixCoversEveryOperation|TestEveryColumnCallsAsARegisteredAccount|TestEveryActionHasARuleAndEveryRuleAnAction' ./internal/bootstrap/`
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
git add api/modules/project.yaml api/openapi.yaml server/internal/bootstrap/app.go server/internal/bootstrap/permission_matrix_columns_test.go server/internal/bootstrap/permission_matrix_project_test.go server/internal/bootstrap/ports.go server/internal/bootstrap/ports_test.go server/internal/bootstrap/project_access_test.go server/internal/modules/access/app/authorizer.go server/internal/modules/access/app/authorizer_test.go server/internal/modules/access/app/ports.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/access/module.go server/internal/modules/project/adapter/http/handler.go server/internal/modules/project/adapter/http/handler_test.go server/internal/modules/project/adapter/http/projects.go server/internal/modules/project/adapter/http/projects_test.go server/internal/modules/project/adapter/postgres/access.go server/internal/modules/project/adapter/postgres/access_test.go server/internal/modules/project/adapter/postgres/queries/access.sql server/internal/modules/project/app/access.go server/internal/modules/project/app/get_project.go server/internal/modules/project/app/get_project_test.go server/internal/modules/project/app/ports.go server/internal/modules/project/domain/actions.go server/internal/modules/project/domain/errors.go server/internal/modules/project/module.go web/apps/web/helpers/authentication.helper.ts web/packages/i18n/src/locales/en/auth.json web/packages/i18n/src/locales/zh-CN/auth.json api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/server.gen.go server/internal/modules/project/adapter/postgres/gen/access.sql.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P4a): GET /api/v0/projects/{project_id}; the Authorizer reads a target's project

project.Provide offers access its ProjectAccess, and Authorize reads a
target project's facts for the caller: a project of another workspace
than the target's, or none, is seen by no one. getProject reads the
project as the caller sees it, then decides project.read, which whoever
sees it passes; not there and not seen answer the same project.not_found
(M3 design 3.4, 8.2). The matrix gains getProject across the project
level's columns and the archived project's.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 |
|---|---|
| 读了项目的事实而不用于判定；别的工作区的项目也算；项目的成员关系从不算；每个项目都私密；为"谁都不是"读事实 | `TestAuthorizeReadsTheTargetsProject`、`TestPermissionMatrix`（按改法各一个或两个；别的工作区的项目另有真实存储上的 `TestTheAuthorizerSeesNoProjectOfAnotherWorkspace`） |
| 在调用者的事务之外读事实；工作区级的目标也读项目 | `TestAuthorizeReadsTheTargetsProject`；`TestAuthorizeReadsTheCallersRoleInTheTargetsWorkspace` |
| 读事实的失败被吞掉 | `TestAuthorizeReturnsThePortsError` |
| `project.read` 只给项目成员；判定时不要求有效的工作区成员关系 | `TestEveryRuleDecidesItsCells`、`TestPermissionMatrix`；`TestDecideAtTheProjectLevels`、`TestPermissionMatrix` |
| `ProjectFacts` 找到已删除的项目；不看项目的 id（按 id 升序、降序两个行序各一次）；已结束的、已删除的、任何人的、任何项目的成员关系都算；公开读成私密；每个成员都是管理员 | `TestProjectFacts` |
| `getProject` 谁问都答；只按工作区判定；看不到不答 `project.not_found`；忽略判定的别的回答；读的失败答成 404；读成"谁都不是"的视角；没有调用者照常进行 | `TestGetProject`、`TestGetProjectRefuses`（按改法，另有 `TestPermissionMatrix`） |
| handler 不传路径的 id | `TestGetProject`（HTTP） |
| 转换丢掉调用者的成员关系；`Authorizer` 找不到任何项目 | `TestAccessProjectsConvertsProjectsAnswer`、`TestPermissionMatrix`；`TestPermissionMatrix` |
| 已归档项目的列以 `acme` 的管理员调用；WM-私 指向公开项目；准备数据里 PA 是公开项目的成员、PG 不是它的成员、私密项目公开、以前的成员仍有效、已归档的项目没有归档 | `TestPermissionMatrix` |

**Done when:** 判定读项目的事实只为带项目的目标，不跨工作区（假实现上和真实的存储上各有测试）；`getProject` 的 13 格（矩阵共 157 格）和它的用例、存储、HTTP 测试通过；`project.Provide` 只交出 `ProjectAccess`；前端检查通过。

---

### Task 10: `checkProjectIdentifier`

**Files:**
- Create: `server/internal/modules/project/app/check_identifier.go`、`server/internal/modules/project/app/check_identifier_test.go`
- Modify: `api/modules/project.yaml`、`api/openapi.yaml`、`server/internal/bootstrap/permission_matrix_project_test.go`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/bootstrap/ports.go`、`server/internal/bootstrap/ports_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/project/adapter/http/handler.go`、`server/internal/modules/project/adapter/http/handler_test.go`、`server/internal/modules/project/adapter/http/projects.go`、`server/internal/modules/project/adapter/http/projects_test.go`、`server/internal/modules/project/adapter/postgres/projects.go`、`server/internal/modules/project/adapter/postgres/projects_test.go`、`server/internal/modules/project/adapter/postgres/queries/projects.sql`、`server/internal/modules/project/app/fakes_create_test.go`、`server/internal/modules/project/app/ports.go`、`server/internal/modules/project/domain/actions.go`、`server/internal/modules/project/domain/project.go`、`server/internal/modules/project/domain/project_test.go`、`server/internal/modules/project/module.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`server/internal/modules/project/adapter/postgres/gen/projects.sql.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.10，M3 设计 3.19、5.1；M1-P3 交接的地址）：`GET /api/v0/workspaces/{slug}/project-identifiers/{identifier}`（不带结尾 `/`），200 `IdentifierAvailability{available}`；码 `[workspace.not_found, forbidden]`；规则 `project_identifier.check`：工作区的管理员和成员（与建项目相同）；`domain.ValidIdentifier(s)`；`app.IdentifierReader.IdentifierTaken`；`app.NewCheckProjectIdentifier(workspaces, projects, auth)`：不加锁、不开事务地按 slug 找工作区 → 判定 → 标识不合规答 `available: false`、不问存储 → 否则转大写问存储。`app.WorkspaceDirectory` 加 `WorkspaceBySlug`（不加锁），`projectWorkspaces` 转换它。
- 矩阵：两行（12 格，`inWorkspace(200, 200, 403)`，被占的 `web` 和空着的 `NEW`）；`{identifier}` 按参数列为"不是目标"（问的是一个标识），`{slug}` 照常核对（裁定 G3）。

**Tests:**
- `app/check_identifier_test.go`：`TestCheckProjectIdentifier`（不加锁、不开事务；判定之后转大写问存储；不合规的标识不问存储、答不可用）；`TestCheckProjectIdentifierRefuses`（工作区没有、看不到；访客；每个失败原样返回；没有调用者什么都不读；都不问标识）。
- `adapter/postgres/projects_test.go`：`TestIdentifierTaken`（别的工作区的、已删除的项目的标识不算）；`adapter/http/projects_test.go`：`TestCheckProjectIdentifier`（路径的工作区和标识解码后交给用例）；`domain/project_test.go`：`TestValidIdentifier`；`TestProjectWorkspacesConvertsWorkspacesAnswer` 加不加锁的读。

- [ ] **Step 1: 接口描述和查询**

`api/modules/project.yaml`（修改，2 处）：

````old api/modules/project.yaml
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Project'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/projects/{project_id}:
````
````new api/modules/project.yaml
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Project'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/workspaces/{slug}/project-identifiers/{identifier}:
    parameters:
      - $ref: '#/components/parameters/Slug'
      - name: identifier
        in: path
        required: true
        description: An identifier asked about, in any case.
        schema:
          type: string
    get:
      operationId: checkProjectIdentifier
      tags: [project]
      summary: Check whether a project identifier is available in a workspace
      description: >-
        For the workspace's admins and members, as createProject. The
        identifier is available when createProject would take it: once
        upper-cased, 1–10 of A-Z, 0-9 and ÇŞĞİÖÜ, and no undeleted project
        of the workspace's. A workspace that does not exist, is deleted, or of
        which the caller is not an active member answers workspace.not_found;
        a guest, forbidden.
      security: [{bearer: []}]
      x-problem-codes: [workspace.not_found, forbidden]
      responses:
        '200':
          description: Whether the identifier is available.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IdentifierAvailability'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/projects/{project_id}:
````

````old api/modules/project.yaml
          type: string
          format: date-time
    ProjectCreate:
````
````new api/modules/project.yaml
          type: string
          format: date-time
    IdentifierAvailability:
      type: object
      additionalProperties: false
      required: [available]
      properties:
        available:
          type: boolean
    ProjectCreate:
````

`api/openapi.yaml`（修改，1 处）：

````old api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1workspaces~1{slug}~1projects'
````
````new api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1workspaces~1{slug}~1projects'
  /api/v0/workspaces/{slug}/project-identifiers/{identifier}:
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1workspaces~1{slug}~1project-identifiers~1{identifier}'
````

`server/internal/modules/project/adapter/postgres/queries/projects.sql`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/queries/projects.sql
        sqlc.arg(now), sqlc.arg(now));
````
````new server/internal/modules/project/adapter/postgres/queries/projects.sql
        sqlc.arg(now), sqlc.arg(now));

-- name: IdentifierTaken :one
-- checkProjectIdentifier: whether an undeleted project of the workspace has the identifier.
SELECT EXISTS (SELECT 1 FROM projects
               WHERE workspace_id = sqlc.arg(workspace_id) AND identifier = sqlc.arg(identifier) AND deleted_at IS NULL);
````

- [ ] **Step 2: 生成**

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `184e3b131f60265aff23cc1f44ac9bbc29e54447179d9190fb47012d6dd6c87f` | 1970 | `api/dist/openapi.yaml` |
| `1fc54bcccf9e55100e07328f082b22be8ff57d22c2dd9e60cbbb70fabfc89db5` | 733 | `server/internal/modules/project/adapter/http/gen/server.gen.go` |
| `e90e5fd1bd25110a5abc57c4f5861685c50c3876f566077838a69d7cb0deeeac` | 150 | `server/internal/modules/project/adapter/postgres/gen/projects.sql.go` |
| `0198cf1bf19672e08aceed3a78f4223a26585b4021fbad0c505c53ecca4961a6` | 2064 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/server.gen.go server/internal/modules/project/adapter/postgres/gen/projects.sql.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 3: 领域、用例、存储、HTTP、规则**

`server/internal/modules/project/domain/project.go`（修改，1 处）：

````old server/internal/modules/project/domain/project.go
	return strings.ToUpper(s)
````
````new server/internal/modules/project/domain/project.go
	return strings.ToUpper(s)
}

// ValidIdentifier reports whether s, upper-cased, is an identifier that
// CheckNewProject accepts.
func ValidIdentifier(s string) bool {
	return checkIdentifier(Identifier(s)) == nil
````

`server/internal/modules/project/domain/project_test.go`（修改，1 处）：

````old server/internal/modules/project/domain/project_test.go
}

// Only a workspace's admins and members may lead a new project, not its
````
````new server/internal/modules/project/domain/project_test.go
}

// ValidIdentifier is CheckNewProject's rule on the identifier, in any case.
func TestValidIdentifier(t *testing.T) {
	for s, want := range map[string]bool{"WEB": true, "web": true, "çay1": true, "ABCDEFGHIJ": true, "": false, "ABCDEFGHIJK": false,
		"WEB-2": false, "WE B": false, "Ä": false} {
		if got := ValidIdentifier(s); got != want {
			t.Errorf("ValidIdentifier(%q) = %v, want %v", s, got, want)
		}
	}
}

// Only a workspace's admins and members may lead a new project, not its
````

`server/internal/modules/project/domain/actions.go`（修改，2 处）：

````old server/internal/modules/project/domain/actions.go
	ActionRead shared.Action = "project.read"
````
````new server/internal/modules/project/domain/actions.go
	ActionRead shared.Action = "project.read"
	// ActionCheckIdentifier is asking whether an identifier is available
	// in a workspace: checkProjectIdentifier.
	ActionCheckIdentifier shared.Action = "project_identifier.check"
````

````old server/internal/modules/project/domain/actions.go
	return []shared.Action{ActionCreate, ActionRead}
````
````new server/internal/modules/project/domain/actions.go
	return []shared.Action{ActionCreate, ActionRead, ActionCheckIdentifier}
````

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"project.read": {Level: LevelVisible},
````
````new server/internal/modules/access/domain/rules.go
	"project.read": {Level: LevelVisible},
	// Who may create a project (M3 design 9.2).
	"project_identifier.check": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"project.create":               {allowed, allowed, forbidden, invisible, invisible, invisible, forbidden},
````
````new server/internal/modules/access/domain/rules_test.go
	"project.create":               {allowed, allowed, forbidden, invisible, invisible, invisible, forbidden},
	"project_identifier.check":     {allowed, allowed, forbidden, invisible, invisible, invisible, forbidden},
````

`server/internal/modules/project/app/ports.go`（修改，2 处）：

````old server/internal/modules/project/app/ports.go
type WorkspaceDirectory interface {
````
````new server/internal/modules/project/app/ports.go
type WorkspaceDirectory interface {
	// WorkspaceBySlug reads it without a lock: for a read.
	WorkspaceBySlug(ctx context.Context, slug string) (w Workspace, found bool, err error)
````

````old server/internal/modules/project/app/ports.go
	GetProject(ctx context.Context, id, userID uuid.UUID) (p domain.Project, found bool, err error)
````
````new server/internal/modules/project/app/ports.go
	GetProject(ctx context.Context, id, userID uuid.UUID) (p domain.Project, found bool, err error)
}

// IdentifierReader is checkProjectIdentifier's repository.
type IdentifierReader interface {
	// IdentifierTaken reports whether an undeleted project of workspaceID
	// has identifier.
	IdentifierTaken(ctx context.Context, workspaceID uuid.UUID, identifier string) (bool, error)
````

`server/internal/modules/project/app/check_identifier.go`（新文件，57 行）：

````file server/internal/modules/project/app/check_identifier.go
package app

import (
	"context"
	"errors"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CheckProjectIdentifier answers whether an identifier is available in a
// workspace: GET /api/v0/workspaces/{slug}/project-identifiers/{identifier}
// (M3 design 5.1).
type CheckProjectIdentifier struct {
	workspaces WorkspaceDirectory
	projects   IdentifierReader
	auth       shared.Authorizer
}

// NewCheckProjectIdentifier returns the use case.
func NewCheckProjectIdentifier(workspaces WorkspaceDirectory, projects IdentifierReader, auth shared.Authorizer) *CheckProjectIdentifier {
	return &CheckProjectIdentifier{workspaces: workspaces, projects: projects, auth: auth}
}

// Execute finds the workspace slug, decides project_identifier.check in it,
// then answers whether createProject would take identifier: valid once
// upper-cased (domain.ValidIdentifier), and no undeleted project of the
// workspace's. A workspace not there, or not visible to the caller, is
// domain.ErrWorkspaceNotFound. A read opens no transaction (M3 design 6.7).
func (u *CheckProjectIdentifier) Execute(ctx context.Context, slug, identifier string) (bool, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return false, err
	}
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
		return false, err
	}
	if !domain.ValidIdentifier(identifier) {
		return false, nil
	}
	taken, err := u.projects.IdentifierTaken(ctx, ws.ID, domain.Identifier(identifier))
	if err != nil {
		return false, err
	}
	return !taken, nil
}
````

`server/internal/modules/project/app/check_identifier_test.go`（新文件，102 行）：

````file server/internal/modules/project/app/check_identifier_test.go
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

// fakeIdentifiers holds the identifiers taken, by workspace; it logs each
// question and fails with err.
type fakeIdentifiers struct {
	log   *callLog
	taken map[uuid.UUID][]string
	err   error
}

func (f *fakeIdentifiers) IdentifierTaken(ctx context.Context, workspaceID uuid.UUID, identifier string) (bool, error) {
	f.log.add(ctx, "IdentifierTaken %s %s", workspaceID, identifier)
	return slices.Contains(f.taken[workspaceID], identifier), f.err
}

// newCheck is CheckProjectIdentifier over fakes sharing one log: acme has
// WEB, beta OPS; alice is acme's member, carol its guest.
func newCheck() (*app.CheckProjectIdentifier, *fakeDirectory, *fakeIdentifiers, *fakeAuthorizer) {
	log := &callLog{}
	workspaces := &fakeDirectory{log: log, workspaces: map[string]app.Workspace{"acme": acme, "beta": beta}}
	projects := &fakeIdentifiers{log: log, taken: map[uuid.UUID][]string{acme.ID: {"WEB"}, beta.ID: {"OPS"}}}
	auth := &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{{alice, acme.ID}: {WorkspaceRole: shared.RoleMember}},
		errs: map[grantKey]error{{carol, acme.ID}: shared.Forbidden()}}
	return app.NewCheckProjectIdentifier(workspaces, projects, auth), workspaces, projects, auth
}

// The use case finds the workspace without a lock and outside any
// transaction, decides project_identifier.check in it, then asks for the
// identifier upper-cased: available unless the workspace has it. An
// identifier createProject would refuse is not available, and the store is
// not asked.
func TestCheckProjectIdentifier(t *testing.T) {
	decided := []string{"WorkspaceBySlug acme outside tx",
		fmt.Sprintf("Authorize %s project_identifier.check on %s/%s outside tx", alice, acme.ID, uuid.UUID{})}
	asked := func(id string) []string {
		return append(slices.Clone(decided), fmt.Sprintf("IdentifierTaken %s %s outside tx", acme.ID, id))
	}
	tests := []struct {
		identifier string
		want       bool
		calls      []string
	}{
		{"WEB", false, asked("WEB")},
		{"web", false, asked("WEB")},
		{"OPS", true, asked("OPS")},
		{"çay1", true, asked("ÇAY1")},
		{"WEB-2", false, decided},
		{"ABCDEFGHIJK", false, decided},
	}
	for _, tt := range tests {
		uc, workspaces, _, _ := newCheck()
		got, err := uc.Execute(as(alice), "acme", tt.identifier)
		if err != nil || got != tt.want || !slices.Equal(workspaces.log.calls, tt.calls) {
			t.Errorf("Execute(%q) = %v, %v, calls %q; want %v, calls %q", tt.identifier, got, err, workspaces.log.calls, tt.want, tt.calls)
		}
	}
}

// A workspace not there, or not visible, is workspace.not_found; a guest
// is refused; every failure comes back as itself; without a caller nothing
// is read. None of them asks about the identifier.
func TestCheckProjectIdentifierRefuses(t *testing.T) {
	failure := errors.New("connection reset")
	tests := []struct {
		name       string
		ctx        context.Context
		slug       string
		dirErr     error
		storeErr   error
		want       error
		identifier bool // the store was asked
	}{
		{"no workspace", as(alice), "gone", nil, nil, domain.ErrWorkspaceNotFound, false},
		{"a workspace he is not in", as(dave), "acme", nil, nil, domain.ErrWorkspaceNotFound, false},
		{"a guest", as(carol), "acme", nil, nil, shared.Forbidden(), false},
		{"the directory failing", as(alice), "acme", failure, nil, failure, false},
		{"the store failing", as(alice), "acme", nil, failure, failure, true},
		{"no caller", context.Background(), "acme", nil, nil, shared.Unauthenticated(), false},
	}
	for _, tt := range tests {
		uc, workspaces, projects, _ := newCheck()
		workspaces.err, projects.err = tt.dirErr, tt.storeErr
		got, err := uc.Execute(tt.ctx, tt.slug, "NEW")
		asked := slices.ContainsFunc(workspaces.log.calls, func(c string) bool { return c == fmt.Sprintf("IdentifierTaken %s NEW outside tx", acme.ID) })
		if !errors.Is(err, tt.want) || got || asked != tt.identifier {
			t.Errorf("%s: Execute() = %v, %v, calls %q; want false, %v", tt.name, got, err, workspaces.log.calls, tt.want)
		}
	}
}
````

`server/internal/modules/project/app/fakes_create_test.go`（修改，1 处）：

````old server/internal/modules/project/app/fakes_create_test.go
	err        error
````
````new server/internal/modules/project/app/fakes_create_test.go
	err        error
}

func (f *fakeDirectory) WorkspaceBySlug(ctx context.Context, slug string) (app.Workspace, bool, error) {
	f.log.add(ctx, "WorkspaceBySlug %s", slug)
	if f.err != nil {
		return app.Workspace{}, false, f.err
	}
	w, ok := f.workspaces[slug]
	return w, ok, nil
````

`server/internal/modules/project/adapter/postgres/projects.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/projects.go
}

// GetProject returns the undeleted project id, archived or not, as userID
````
````new server/internal/modules/project/adapter/postgres/projects.go
}

// IdentifierTaken reports whether an undeleted project of workspaceID has
// identifier.
func (s *Store) IdentifierTaken(ctx context.Context, workspaceID uuid.UUID, identifier string) (bool, error) {
	taken, err := s.queries(ctx).IdentifierTaken(ctx, gen.IdentifierTakenParams{WorkspaceID: workspaceID, Identifier: identifier})
	if err != nil {
		return false, fmt.Errorf("check identifier %q: %w", identifier, err)
	}
	return taken, nil
}

// GetProject returns the undeleted project id, archived or not, as userID
````

`server/internal/modules/project/adapter/postgres/projects_test.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/projects_test.go
}

// GetProject answers the caller's view (M3 design 3.19): his role and his
````
````new server/internal/modules/project/adapter/postgres/projects_test.go
}

// IdentifierTaken: the workspace's undeleted projects' identifiers, as
// stored, in upper case; another workspace's and a deleted project's do not
// count.
func TestIdentifierTaken(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	alice := newAccount(t, pool, "alice@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	newProject(t, s, acme, "Web", "WEB", alice)
	gone := newProject(t, s, acme, "Gone", "GONE", alice)
	exec(t, pool, "UPDATE projects SET deleted_at = now() WHERE id = $1", gone)
	newProject(t, s, beta, "Ops", "OPS", alice)
	for _, tt := range []struct {
		identifier string
		want       bool
	}{{"WEB", true}, {"web", false}, {"GONE", false}, {"OPS", false}, {"NEW", false}} {
		if got, err := s.IdentifierTaken(ctx, acme, tt.identifier); err != nil || got != tt.want {
			t.Errorf("IdentifierTaken(acme, %s) = %v, %v; want %v", tt.identifier, got, err, tt.want)
		}
	}
}

// GetProject answers the caller's view (M3 design 3.19): his role and his
````

`server/internal/modules/project/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/project/adapter/http/handler.go
}

// UseCases are the use cases behind the module's operations.
````
````new server/internal/modules/project/adapter/http/handler.go
}

// CheckIdentifierUseCase is app.CheckProjectIdentifier.
type CheckIdentifierUseCase interface {
	Execute(ctx context.Context, slug, identifier string) (bool, error)
}

// UseCases are the use cases behind the module's operations.
````

````old server/internal/modules/project/adapter/http/handler.go
	CreateProject CreateProjectUseCase
	GetProject    GetProjectUseCase
````
````new server/internal/modules/project/adapter/http/handler.go
	CreateProject   CreateProjectUseCase
	GetProject      GetProjectUseCase
	CheckIdentifier CheckIdentifierUseCase
````

`server/internal/modules/project/adapter/http/projects.go`（修改，1 处）：

````old server/internal/modules/project/adapter/http/projects.go
	return gen.GetProject200JSONResponse(project(p)), nil
````
````new server/internal/modules/project/adapter/http/projects.go
	return gen.GetProject200JSONResponse(project(p)), nil
}

// CheckProjectIdentifier serves GET
// /api/v0/workspaces/{slug}/project-identifiers/{identifier}.
func (h handler) CheckProjectIdentifier(ctx context.Context, req gen.CheckProjectIdentifierRequestObject) (gen.CheckProjectIdentifierResponseObject, error) {
	available, err := h.uc.CheckIdentifier.Execute(ctx, req.Slug, req.Identifier)
	if err != nil {
		return nil, err
	}
	return gen.CheckProjectIdentifier200JSONResponse{Available: available}, nil
````

`server/internal/modules/project/adapter/http/handler_test.go`（修改，3 处）：

````old server/internal/modules/project/adapter/http/handler_test.go
	get    *fakeGet
````
````new server/internal/modules/project/adapter/http/handler_test.go
	get    *fakeGet
	check  *fakeCheck
````

````old server/internal/modules/project/adapter/http/handler_test.go
}

// newServer serves the module with f; a fake left nil is an idle one.
````
````new server/internal/modules/project/adapter/http/handler_test.go
}

type fakeCheck struct {
	calls     []string // "caller slug identifier"
	available map[string]bool
	err       error
}

func (f *fakeCheck) Execute(ctx context.Context, slug, identifier string) (bool, error) {
	f.calls = append(f.calls, caller(ctx)+" "+slug+" "+identifier)
	return f.available[identifier], f.err
}

// newServer serves the module with f; a fake left nil is an idle one.
````

````old server/internal/modules/project/adapter/http/handler_test.go
	httpadapter.Register(router, api, httpadapter.UseCases{CreateProject: f.create, GetProject: f.get})
````
````new server/internal/modules/project/adapter/http/handler_test.go
	if f.check == nil {
		f.check = &fakeCheck{}
	}
	httpadapter.Register(router, api, httpadapter.UseCases{CreateProject: f.create, GetProject: f.get, CheckIdentifier: f.check})
````

`server/internal/modules/project/adapter/http/projects_test.go`（修改，1 处）：

````old server/internal/modules/project/adapter/http/projects_test.go
		t.Errorf("calls = %q, want %q", get.calls, want)
	}
}

````
````new server/internal/modules/project/adapter/http/projects_test.go
		t.Errorf("calls = %q, want %q", get.calls, want)
	}
}

// The path's workspace and identifier, decoded, go to the use case for the
// caller; the answer is its availability, and its refusals as the contract
// declares them.
func TestCheckProjectIdentifier(t *testing.T) {
	check := &fakeCheck{available: map[string]bool{"NEW": true, "ÇAY": true}}
	h := newServer(t, fakes{check: check})
	for path, want := range map[string]string{
		"/api/v0/workspaces/acme/project-identifiers/NEW":      `{"available":true}`,
		"/api/v0/workspaces/acme/project-identifiers/WEB":      `{"available":false}`,
		"/api/v0/workspaces/acme/project-identifiers/%C3%87AY": `{"available":true}`,
	} {
		res, body := do(t, h, request(http.MethodGet, path, "alice", ""))
		if res.StatusCode != http.StatusOK || body != want+"\n" {
			t.Errorf("GET %s = %d %s, want 200 %s", path, res.StatusCode, body, want)
		}
	}
	if want := []string{"alice acme NEW", "alice acme WEB", "alice acme ÇAY"}; !slices.Equal(slices.Sorted(slices.Values(check.calls)),
		slices.Sorted(slices.Values(want))) {
		t.Errorf("calls = %q, want %q", check.calls, want)
	}
	for _, tt := range []struct {
		err    error
		status int
		want   string
	}{
		{domain.ErrWorkspaceNotFound, http.StatusNotFound,
			`{"status":404,"code":"workspace.not_found","title":"Not Found","detail":"The workspace does not exist, or you are not a member of it."}`},
		{shared.Forbidden(), http.StatusForbidden, `{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
	} {
		h := newServer(t, fakes{check: &fakeCheck{err: tt.err}})
		res, body := do(t, h, request(http.MethodGet, "/api/v0/workspaces/acme/project-identifiers/NEW", "bob", ""))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("GET = %d %s, want %d %s", res.StatusCode, body, tt.status, tt.want)
		}
	}
}

````

`server/internal/modules/project/module.go`（修改，2 处）：

````old server/internal/modules/project/module.go
// brings creating and reading projects, carries out the workspace module's
// cascades on the projects (ProjectCascade), and offers the access module
// its reads of a project (ProjectAccess).
````
````new server/internal/modules/project/module.go
// brings creating and reading projects and checking an identifier, carries
// out the workspace module's cascades on the projects (ProjectCascade), and
// offers the access module its reads of a project (ProjectAccess).
````

````old server/internal/modules/project/module.go
		GetProject: app.NewGetProject(store, d.Authorizer),
````
````new server/internal/modules/project/module.go
		GetProject:      app.NewGetProject(store, d.Authorizer),
		CheckIdentifier: app.NewCheckProjectIdentifier(d.Workspaces, store, d.Authorizer),
````

- [ ] **Step 4: 组合根和矩阵**

`server/internal/bootstrap/ports.go`（修改，1 处）：

````old server/internal/bootstrap/ports.go
}

func (d projectWorkspaces) ShareWorkspaceBySlug(ctx context.Context, slug string) (project.Workspace, bool, error) {
````
````new server/internal/bootstrap/ports.go
}

func (d projectWorkspaces) WorkspaceBySlug(ctx context.Context, slug string) (project.Workspace, bool, error) {
	w, found, err := d.directory.WorkspaceBySlug(ctx, slug)
	return project.Workspace(w), found, err
}

func (d projectWorkspaces) ShareWorkspaceBySlug(ctx context.Context, slug string) (project.Workspace, bool, error) {
````

`server/internal/bootstrap/ports_test.go`（修改，3 处）：

````old server/internal/bootstrap/ports_test.go
// through the same lock: the workspace converted, found and the error as
// they came.
````
````new server/internal/bootstrap/ports_test.go
// through the same lock or without one: the workspace converted, found and
// the error as they came.
````

````old server/internal/bootstrap/ports_test.go
	d := projectWorkspaces{directory: fake}
	ctx := context.Background()
````
````new server/internal/bootstrap/ports_test.go
	d := projectWorkspaces{directory: fake}
	ctx := context.Background()
	want := project.Workspace{ID: acme.ID, Timezone: "Asia/Shanghai"}
````

````old server/internal/bootstrap/ports_test.go
	w, found, err := d.ShareWorkspaceBySlug(ctx, "acme")
	if want := (project.Workspace{ID: acme.ID, Timezone: "Asia/Shanghai"}); err != nil || !found || w != want {
		t.Errorf("ShareWorkspaceBySlug(acme) = %+v, %v, %v; want %+v, found", w, found, err, want)
	}
	if w, found, err := d.ShareWorkspaceBySlug(ctx, "gone"); err != nil || found || w != (project.Workspace{}) {
		t.Errorf("ShareWorkspaceBySlug(gone) = %+v, %v, %v; want not found", w, found, err)
	}
	if want := []string{"share acme", "share gone"}; !slices.Equal(fake.asked, want) {
		t.Errorf("workspace was asked %q, want %q", fake.asked, want)
	}
	failure := errors.New("connection reset")
	fake.err = failure
	if _, _, err := d.ShareWorkspaceBySlug(ctx, "acme"); !errors.Is(err, failure) {
		t.Errorf("ShareWorkspaceBySlug() = %v, want %v", err, failure)
````
````new server/internal/bootstrap/ports_test.go
	for name, find := range map[string]func(context.Context, string) (project.Workspace, bool, error){
		"share": d.ShareWorkspaceBySlug, "read": d.WorkspaceBySlug,
	} {
		fake.asked, fake.err = nil, nil
		if w, found, err := find(ctx, "acme"); err != nil || !found || w != want {
			t.Errorf("%s acme = %+v, %v, %v; want %+v, found", name, w, found, err, want)
		}
		if w, found, err := find(ctx, "gone"); err != nil || found || w != (project.Workspace{}) {
			t.Errorf("%s gone = %+v, %v, %v; want not found", name, w, found, err)
		}
		if want := []string{name + " acme", name + " gone"}; !slices.Equal(fake.asked, want) {
			t.Errorf("workspace was asked %q, want %q", fake.asked, want)
		}
		failure := errors.New("connection reset")
		fake.err = failure
		if _, _, err := find(ctx, "acme"); !errors.Is(err, failure) {
			t.Errorf("%s = %v, want %v", name, err, failure)
		}
````

`server/internal/bootstrap/permission_matrix_project_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_project_test.go
			cells:   inWorkspace(cellValidationFailed, cellValidationFailed, cellForbidden)},
````
````new server/internal/bootstrap/permission_matrix_project_test.go
			cells:   inWorkspace(cellValidationFailed, cellValidationFailed, cellForbidden)},
		// acme has WEB, which the identifier asked about is in any case; gone
		// has it too, deleted with gone.
		{op: "checkProjectIdentifier", variant: "taken", request: toWorkspace(http.MethodGet, "/project-identifiers/web", ""),
			cells: inWorkspace(cellOK, cellOK, cellForbidden), check: identifierAvailable(false)},
		{op: "checkProjectIdentifier", variant: "free", request: toWorkspace(http.MethodGet, "/project-identifiers/NEW", ""),
			cells: inWorkspace(cellOK, cellOK, cellForbidden), check: identifierAvailable(true)},
````

````old server/internal/bootstrap/permission_matrix_project_test.go
		t.Errorf("%s creates %s; want NEW, with him its admin and only member", c, answer)
	}
}

````
````new server/internal/bootstrap/permission_matrix_project_test.go
		t.Errorf("%s creates %s; want NEW, with him its admin and only member", c, answer)
	}
}

// identifierAvailable: the answer is want.
func identifierAvailable(want bool) func(t *testing.T, c caller, _ seeded, answer string) {
	return func(t *testing.T, c caller, _ seeded, answer string) {
		var a struct {
			Available bool `json:"available"`
		}
		decodeAnswer(t, answer, &a)
		if a.Available != want {
			t.Errorf("%s is answered %s, want available %v", c, answer, want)
		}
	}
}

````

`server/internal/bootstrap/permission_matrix_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_test.go
		{"/api/v0/workspace-invitations/{invitation_id}/decline", "{invitation_id}"}: "account level, as accept",
````
````new server/internal/bootstrap/permission_matrix_test.go
		{"/api/v0/workspace-invitations/{invitation_id}/decline", "{invitation_id}"}: "account level, as accept",
		{"/api/v0/workspaces/{slug}/project-identifiers/{identifier}", "{identifier}"}: "an identifier asked about, not a row: " +
			"its {slug} is still its column's workspace",
````

- [ ] **Step 5: 测试、lint、前端检查**

Run: `go -C server test -count=1 ./internal/modules/project/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestProjectWorkspacesConvertsWorkspacesAnswer|TestPermissionMatrix$|TestThePermissionMatrixCoversEveryOperation|TestEveryActionHasARuleAndEveryRuleAnAction' ./internal/bootstrap/`
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
git add api/modules/project.yaml api/openapi.yaml server/internal/bootstrap/permission_matrix_project_test.go server/internal/bootstrap/permission_matrix_test.go server/internal/bootstrap/ports.go server/internal/bootstrap/ports_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/project/adapter/http/handler.go server/internal/modules/project/adapter/http/handler_test.go server/internal/modules/project/adapter/http/projects.go server/internal/modules/project/adapter/http/projects_test.go server/internal/modules/project/adapter/postgres/projects.go server/internal/modules/project/adapter/postgres/projects_test.go server/internal/modules/project/adapter/postgres/queries/projects.sql server/internal/modules/project/app/check_identifier.go server/internal/modules/project/app/check_identifier_test.go server/internal/modules/project/app/fakes_create_test.go server/internal/modules/project/app/ports.go server/internal/modules/project/domain/actions.go server/internal/modules/project/domain/project.go server/internal/modules/project/domain/project_test.go server/internal/modules/project/module.go api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/server.gen.go server/internal/modules/project/adapter/postgres/gen/projects.sql.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P4a): GET /api/v0/workspaces/{slug}/project-identifiers/{identifier}

For the workspace's admins and members, as createProject: an identifier
is available when createProject would take it, valid once upper-cased
and no undeleted project's of the workspace. A read, without a lock or
a transaction; an invalid identifier is not available and the store is
not asked. The matrix lists {identifier} as no target and still checks
the path's {slug}.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 |
|---|---|
| 任何调用者都可以问；看不到不答 `workspace.not_found`；没有调用者照常进行；规则让访客问 | `TestCheckProjectIdentifierRefuses`（前两个、最后一个另有 `TestPermissionMatrix`，最后一个另有 `TestEveryRuleDecidesItsCells`） |
| 问不合规的标识；不转大写；被占答成可用；不带工作区问 | `TestCheckProjectIdentifier`（后三个另有 `TestPermissionMatrix`） |
| 读加锁；转换把不加锁的读转成加锁的 | `TestCheckProjectIdentifier`；`TestProjectWorkspacesConvertsWorkspacesAnswer` |
| 目录的失败答成 404；存储的失败答成不可用 | `TestCheckProjectIdentifierRefuses` |
| `IdentifierTaken` 算已删除的、别的工作区的；不看标识 | `TestIdentifierTaken`（后两个 Task 14 起 P1 单独运行也失败） |
| `ValidIdentifier` 按原样的大小写 | `TestValidIdentifier` |
| handler 把 slug 当标识 | `TestCheckProjectIdentifier`（HTTP） |
| `{identifier}` 不列为"不是目标"；各格都在 `acme` 问 | `TestThePermissionMatrixCoversEveryOperation` |

**Done when:** 检查的判定、大小写、不合规的标识、被占与空着由测试核对；矩阵 169 格；前端检查通过。

---

### Task 11: `listProjects` 与可见性一致

**Files:**
- Create: `server/internal/bootstrap/project_visibility_test.go`、`server/internal/modules/project/adapter/postgres/list_test.go`、`server/internal/modules/project/app/list_projects.go`、`server/internal/modules/project/app/list_projects_test.go`、`server/internal/modules/project/domain/visibility.go`、`server/internal/modules/project/domain/visibility_test.go`
- Modify: `api/modules/project.yaml`、`server/internal/bootstrap/permission_matrix_project_test.go`、`server/internal/bootstrap/permission_matrix_seeded_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/project/adapter/http/handler.go`、`server/internal/modules/project/adapter/http/handler_test.go`、`server/internal/modules/project/adapter/http/projects.go`、`server/internal/modules/project/adapter/http/projects_test.go`、`server/internal/modules/project/adapter/postgres/projects.go`、`server/internal/modules/project/adapter/postgres/projects_test.go`、`server/internal/modules/project/adapter/postgres/queries/projects.sql`、`server/internal/modules/project/app/ports.go`、`server/internal/modules/project/domain/actions.go`、`server/internal/modules/project/module.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`server/internal/modules/project/adapter/postgres/gen/projects.sql.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.11，M3 设计 3.4、3.12、3.19、9.3）：`GET /api/v0/workspaces/{slug}/projects?archived=`，200 `ProjectList{data}`；码 `[workspace.not_found]`；规则 `project.list`：工作区的每个有效成员（`LevelWorkspace`，管理员、成员、访客）；`domain.Visibility{All, Public}`、`VisibilityOf(role)`（管理员看全部，成员另看公开的，访客只看自己加入的；按集合）；`app.ProjectLister.ListProjects(ctx, workspaceID, userID, v, archived)`；`app.NewListProjects(workspaces, projects, auth)`：不加锁、不开事务地按 slug 找工作区 → 判定 → 按判定读到的工作区角色的 `Visibility` 列出。`ListProjects` 与 `GetProject` 读同样的列（行可以互相转换），按调用者的侧边栏位置、不是成员的在最后、再按名称（在工作区的未删除项目中唯一）排序。
- 矩阵：两行（12 格，`inWorkspace(200, 200, 200)`，未归档的和 `?archived=true` 的，答案核对每一列看到的项目名）；准备数据加 `other` 的项目（`acme` 的列表不能列出它）。

**Tests:**
- `bootstrap/project_visibility_test.go`：`TestListingProjectsIsReadingEach`（矩阵的每个账户，未归档、已归档两种：列表列出的 `acme` 的项目恰好是 `getProject` 让他读到的；有的账户两边都空、有的都满，两边不会都是空的或都是满的）。
- `adapter/postgres/list_test.go`：`TestListProjects`（可见性的三种、已归档的和别的、已删除的、别的工作区的、已结束的成员关系；排序；夹具的存储顺序与答案不同）、`TestListProjectsAnswersEachAsGetProjectDoes`（在 `TestGetProject` 的夹具上，每个账户列出的每一行等于 `GetProject` 给他的那一行：成员列表、角色、侧边栏位置；夹具从 `TestGetProject` 移到 `projects_test.go` 的 `newWebFixture`，两个测试共用，`TestGetProject` 的断言不变）；`app/list_projects_test.go`：`TestListProjects`、`TestListProjectsRefuses`；`adapter/http/projects_test.go`：`TestListProjects`（`archived` 不给时是假，空列表是 `[]`）；`domain/visibility_test.go`：`TestVisibilityOf`。

- [ ] **Step 1: 接口描述和查询**

`api/modules/project.yaml`（修改，2 处）：

````old api/modules/project.yaml
    parameters:
      - $ref: '#/components/parameters/Slug'
    post:
````
````new api/modules/project.yaml
    parameters:
      - $ref: '#/components/parameters/Slug'
    get:
      operationId: listProjects
      tags: [project]
      summary: List a workspace's projects that the caller sees
      description: >-
        The workspace's projects that the caller sees, each as he sees it:
        every one, to its admins; to its members, the public ones and those
        they are members of; to its guests, those they are members of. The
        archived projects are left out, unless archived is true, which lists
        them alone. By the caller's place of each in his sidebar, the
        projects he is not a member of last, then by name. A workspace that
        does not exist, is deleted, or of which the caller is not an active
        member answers workspace.not_found. The whole collection at once:
        collections are not paginated.
      security: [{bearer: []}]
      x-problem-codes: [workspace.not_found]
      parameters:
        - name: archived
          in: query
          required: false
          description: true lists the archived projects alone; false, or no value, the others.
          schema:
            type: boolean
      responses:
        '200':
          description: The projects.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ProjectList'
        default:
          $ref: '#/components/responses/Problem'
    post:
````

````old api/modules/project.yaml
          type: string
          format: date-time
    IdentifierAvailability:
````
````new api/modules/project.yaml
          type: string
          format: date-time
    ProjectList:
      type: object
      additionalProperties: false
      required: [data]
      properties:
        data:
          type: array
          items:
            $ref: '#/components/schemas/Project'
    IdentifierAvailability:
````

`server/internal/modules/project/adapter/postgres/queries/projects.sql`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/queries/projects.sql
WHERE p.id = sqlc.arg(id) AND p.deleted_at IS NULL;

````
````new server/internal/modules/project/adapter/postgres/queries/projects.sql
WHERE p.id = sqlc.arg(id) AND p.deleted_at IS NULL;

-- name: ListProjects :many
-- listProjects (M3 design 3.4, 3.12, 3.19): the workspace's undeleted projects that the user sees, the archived ones or
-- the others, each as GetProject reads it (the same columns, so the rows convert); sees_all and sees_public are his
-- workspace role's domain.Visibility. By his place in his sidebar, the projects he is not a member of last, then by
-- name, which is unique among the workspace's undeleted projects.
SELECT p.id, p.workspace_id, p.name, p.description, p.identifier, p.network, p.project_lead_id, p.default_assignee_id,
       p.cycle_view, p.module_view, p.issue_views_view, p.intake_view, p.guest_view_all_features, p.archive_in,
       p.archived_at, p.logo_props, p.timezone, p.created_at, p.updated_at, m.role AS member_role, u.sort_order,
       ARRAY(SELECT a.member_id FROM project_members a
             WHERE a.project_id = p.id AND a.is_active AND a.deleted_at IS NULL
             ORDER BY a.created_at, a.id)::uuid[] AS member_ids
FROM projects p
LEFT JOIN project_members m
       ON m.project_id = p.id AND m.member_id = sqlc.arg(user_id) AND m.is_active AND m.deleted_at IS NULL
LEFT JOIN project_user_properties u
       ON u.project_id = m.project_id AND u.user_id = m.member_id AND u.deleted_at IS NULL
WHERE p.workspace_id = sqlc.arg(workspace_id) AND p.deleted_at IS NULL
  AND (p.archived_at IS NOT NULL) = sqlc.arg(archived)::boolean
  AND (sqlc.arg(sees_all)::boolean OR m.id IS NOT NULL OR (sqlc.arg(sees_public)::boolean AND p.network = 2))
ORDER BY u.sort_order NULLS LAST, p.name;

````

- [ ] **Step 2: 生成**

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `bc86e79f89962db19fec2d329c715db991d68b2b27dcf39bc9b1884ea91394ef` | 2006 | `api/dist/openapi.yaml` |
| `00f8f69e03dfefe665f66b2b268227749f2c14856502c074acaeb1f2742a32f0` | 868 | `server/internal/modules/project/adapter/http/gen/server.gen.go` |
| `bad76ab051f78512718e7b7849f468fcb5ec6a72a3c7e9db9ec22a86890b46f7` | 254 | `server/internal/modules/project/adapter/postgres/gen/projects.sql.go` |
| `7f2edf8912cdbde7f477014e69fa904c33dfa901b1a635c003a6584f0392c0c2` | 2099 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/server.gen.go server/internal/modules/project/adapter/postgres/gen/projects.sql.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 3: 可见性、用例、存储、HTTP、规则**

`server/internal/modules/project/domain/visibility.go`（新文件，26 行）：

````file server/internal/modules/project/domain/visibility.go
package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// Visibility is which of a workspace's projects an active member sees
// besides those he is an active member of (M3 design 3.4): every one, or
// the public ones, or none. It is the access module's rule of seeing a
// project, which a list applies in its query; bootstrap's test holds the
// list equal to reading each project (M3 design 9.3).
type Visibility struct {
	All    bool
	Public bool
}

// VisibilityOf is the Visibility of an active member of workspace role
// role: the workspace's admin sees every project, its member the public
// ones too, its guest no other; by set, as access decides it.
func VisibilityOf(role shared.Role) Visibility {
	switch role {
	case shared.RoleAdmin:
		return Visibility{All: true, Public: true}
	case shared.RoleMember:
		return Visibility{Public: true}
	}
	return Visibility{}
}
````

`server/internal/modules/project/domain/visibility_test.go`（新文件，18 行）：

````file server/internal/modules/project/domain/visibility_test.go
package domain

import (
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Each workspace role sees what access lets it see (M3 design 3.4); a role
// outside the three sees nothing more than its own projects.
func TestVisibilityOf(t *testing.T) {
	for role, want := range map[shared.Role]Visibility{shared.RoleAdmin: {All: true, Public: true}, shared.RoleMember: {Public: true},
		shared.RoleGuest: {}, 10: {}, 25: {}} {
		if got := VisibilityOf(role); got != want {
			t.Errorf("VisibilityOf(%d) = %+v, want %+v", role, got, want)
		}
	}
}
````

`server/internal/modules/project/domain/actions.go`（修改，2 处）：

````old server/internal/modules/project/domain/actions.go
const (
````
````new server/internal/modules/project/domain/actions.go
const (
	// ActionList is listing a workspace's projects: listProjects.
	ActionList shared.Action = "project.list"
````

````old server/internal/modules/project/domain/actions.go
	return []shared.Action{ActionCreate, ActionRead, ActionCheckIdentifier}
````
````new server/internal/modules/project/domain/actions.go
	return []shared.Action{ActionList, ActionCreate, ActionRead, ActionCheckIdentifier}
````

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"workspace_invitation.delete": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
````
````new server/internal/modules/access/domain/rules.go
	"workspace_invitation.delete": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
	// Every active member; each sees in the list what project.read lets
	// him read (M3 design 3.4).
	"project.list": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"workspace_invitation.delete":  {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
````
````new server/internal/modules/access/domain/rules_test.go
	"workspace_invitation.delete":  {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
	"project.list":                 {allowed, allowed, allowed, invisible, invisible, invisible, forbidden},
````

`server/internal/modules/project/app/ports.go`（修改，1 处）：

````old server/internal/modules/project/app/ports.go
	GetProject(ctx context.Context, id, userID uuid.UUID) (p domain.Project, found bool, err error)
````
````new server/internal/modules/project/app/ports.go
	GetProject(ctx context.Context, id, userID uuid.UUID) (p domain.Project, found bool, err error)
}

// ProjectLister is listProjects' repository.
type ProjectLister interface {
	// ListProjects lists workspaceID's undeleted projects that userID sees
	// with v, the archived ones alone when archived is true and the others
	// otherwise, each as he sees it: by his place in his sidebar, the
	// projects he is not an active member of last, then by name (M3 design
	// 3.12).
	ListProjects(ctx context.Context, workspaceID, userID uuid.UUID, v domain.Visibility, archived bool) ([]domain.Project, error)
````

`server/internal/modules/project/app/list_projects.go`（新文件，50 行）：

````file server/internal/modules/project/app/list_projects.go
package app

import (
	"context"
	"errors"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListProjects lists a workspace's projects that the caller sees: GET
// /api/v0/workspaces/{slug}/projects (M3 design 3.4, 3.12, 3.19).
type ListProjects struct {
	workspaces WorkspaceDirectory
	projects   ProjectLister
	auth       shared.Authorizer
}

// NewListProjects returns the use case.
func NewListProjects(workspaces WorkspaceDirectory, projects ProjectLister, auth shared.Authorizer) *ListProjects {
	return &ListProjects{workspaces: workspaces, projects: projects, auth: auth}
}

// Execute finds the workspace slug, decides project.list in it, then lists
// the projects that the caller's workspace role, as the decision read it,
// sees (domain.VisibilityOf) besides those he is a member of: the archived
// ones alone when archived is true, the others otherwise. A workspace not
// there, or not visible to the caller, is domain.ErrWorkspaceNotFound. A
// read opens no transaction (M3 design 6.7).
func (u *ListProjects) Execute(ctx context.Context, slug string, archived bool) ([]domain.Project, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
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
		return nil, err
	}
	return u.projects.ListProjects(ctx, ws.ID, actor.UserID, domain.VisibilityOf(grant.WorkspaceRole), archived)
}
````

`server/internal/modules/project/app/list_projects_test.go`（新文件，101 行）：

````file server/internal/modules/project/app/list_projects_test.go
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

// fakeLister answers its list, logs each call with the visibility asked for,
// and fails with err, answering nothing then, as the store does.
type fakeLister struct {
	log  *callLog
	list []domain.Project
	err  error
}

func (f *fakeLister) ListProjects(ctx context.Context, workspaceID, userID uuid.UUID, v domain.Visibility, archived bool) ([]domain.Project, error) {
	f.log.add(ctx, "ListProjects %s for %s %+v archived %v", workspaceID, userID, v, archived)
	if f.err != nil {
		return nil, f.err
	}
	return f.list, nil
}

// newList is ListProjects over fakes sharing one log: in acme, alice is a
// member, bob an admin, carol a guest.
func newList() (*app.ListProjects, *fakeDirectory, *fakeLister, *fakeAuthorizer) {
	log := &callLog{}
	workspaces := &fakeDirectory{log: log, workspaces: map[string]app.Workspace{"acme": acme}}
	projects := &fakeLister{log: log, list: []domain.Project{{Name: "Web"}}}
	auth := &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{{alice, acme.ID}: {WorkspaceRole: shared.RoleMember},
		{bob, acme.ID}: {WorkspaceRole: shared.RoleAdmin}, {carol, acme.ID}: {WorkspaceRole: shared.RoleGuest}}}
	return app.NewListProjects(workspaces, projects, auth), workspaces, projects, auth
}

// The use case finds the workspace without a lock, outside any
// transaction, decides project.list in it, then lists what the role the
// decision read sees, archived or not as asked; the answer is the list.
func TestListProjects(t *testing.T) {
	for _, tt := range []struct {
		user     uuid.UUID
		archived bool
		v        domain.Visibility
	}{
		{alice, false, domain.Visibility{Public: true}},
		{bob, true, domain.Visibility{All: true, Public: true}},
		{carol, false, domain.Visibility{}},
	} {
		uc, workspaces, _, _ := newList()
		got, err := uc.Execute(as(tt.user), "acme", tt.archived)
		want := []string{"WorkspaceBySlug acme outside tx",
			fmt.Sprintf("Authorize %s project.list on %s/%s outside tx", tt.user, acme.ID, uuid.UUID{}),
			fmt.Sprintf("ListProjects %s for %s %+v archived %v outside tx", acme.ID, tt.user, tt.v, tt.archived)}
		if err != nil || len(got) != 1 || got[0].Name != "Web" || !slices.Equal(workspaces.log.calls, want) {
			t.Errorf("Execute() as %s = %+v, %v, calls %q; want the list, calls %q", tt.user, got, err, workspaces.log.calls, want)
		}
	}
}

// A workspace not there, or not visible, is workspace.not_found and lists
// nothing; every failure comes back as itself; without a caller nothing is
// read.
func TestListProjectsRefuses(t *testing.T) {
	failure := errors.New("connection reset")
	tests := []struct {
		name     string
		ctx      context.Context
		slug     string
		dirErr   error
		authErr  error
		listErr  error
		want     error
		listRead bool
	}{
		{"no workspace", as(alice), "gone", nil, nil, nil, domain.ErrWorkspaceNotFound, false},
		{"a workspace he is not in", as(dave), "acme", nil, nil, nil, domain.ErrWorkspaceNotFound, false},
		{"the directory failing", as(alice), "acme", failure, nil, nil, failure, false},
		{"the decision failing", as(alice), "acme", nil, failure, nil, failure, false},
		{"the list failing", as(alice), "acme", nil, nil, failure, failure, true},
		{"no caller", context.Background(), "acme", nil, nil, nil, shared.Unauthenticated(), false},
	}
	for _, tt := range tests {
		uc, workspaces, projects, auth := newList()
		workspaces.err, projects.err = tt.dirErr, tt.listErr
		if tt.authErr != nil {
			auth.errs = map[grantKey]error{{alice, acme.ID}: tt.authErr}
		}
		got, err := uc.Execute(tt.ctx, tt.slug, false)
		listed := slices.ContainsFunc(workspaces.log.calls, func(c string) bool { return len(c) > 12 && c[:12] == "ListProjects" })
		if !errors.Is(err, tt.want) || got != nil || listed != tt.listRead {
			t.Errorf("%s: Execute() = %+v, %v, calls %q; want nothing, %v", tt.name, got, err, workspaces.log.calls, tt.want)
		}
	}
}
````

`server/internal/modules/project/adapter/postgres/projects.go`（修改，4 处）：

````old server/internal/modules/project/adapter/postgres/projects.go
		return domain.Project{}, false, fmt.Errorf("read project %s: %w", id, err)
	}
````
````new server/internal/modules/project/adapter/postgres/projects.go
		return domain.Project{}, false, fmt.Errorf("read project %s: %w", id, err)
	}
	p, err = projectOf(r)
	return p, err == nil, err
}

// ListProjects lists workspaceID's undeleted projects that userID sees
// with v, the archived ones alone when archived is true and the others
// otherwise, each as he sees it, in the order of M3 design 3.12.
func (s *Store) ListProjects(ctx context.Context, workspaceID, userID uuid.UUID, v domain.Visibility, archived bool) ([]domain.Project, error) {
	rows, err := s.queries(ctx).ListProjects(ctx, gen.ListProjectsParams{UserID: userID, WorkspaceID: workspaceID, Archived: archived,
		SeesAll: v.All, SeesPublic: v.Public})
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	list := make([]domain.Project, len(rows))
	for i, r := range rows {
		if list[i], err = projectOf(gen.GetProjectRow(r)); err != nil {
			return nil, err
		}
	}
	return list, nil
}

// projectOf is the project a row of GetProject, or of ListProjects, holds.
func projectOf(r gen.GetProjectRow) (domain.Project, error) {
````

````old server/internal/modules/project/adapter/postgres/projects.go
		return domain.Project{}, false, fmt.Errorf("read logo_props of project %s: %w", id, err)
````
````new server/internal/modules/project/adapter/postgres/projects.go
		return domain.Project{}, fmt.Errorf("read logo_props of project %s: %w", r.ID, err)
````

````old server/internal/modules/project/adapter/postgres/projects.go
	p = domain.Project{
````
````new server/internal/modules/project/adapter/postgres/projects.go
	p := domain.Project{
````

````old server/internal/modules/project/adapter/postgres/projects.go
	return p, true, nil
````
````new server/internal/modules/project/adapter/postgres/projects.go
	return p, nil
````

`server/internal/modules/project/adapter/postgres/projects_test.go`（修改，8 处）：

````old server/internal/modules/project/adapter/postgres/projects_test.go
	"uuid"

````
````new server/internal/modules/project/adapter/postgres/projects_test.go
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
````

````old server/internal/modules/project/adapter/postgres/projects_test.go
// GetProject answers the caller's view (M3 design 3.19): his role and his
// place in his sidebar only while his membership is active, and the active
// members in the order they became members, then by the membership's id.
// The fixture has each state once, next to the rows another predicate
````
````new server/internal/modules/project/adapter/postgres/projects_test.go
// webFixture is the workspace acme with its projects ops and web, which
// has each state of a membership once, next to the rows another predicate
````

````old server/internal/modules/project/adapter/postgres/projects_test.go
func TestGetProject(t *testing.T) {
````
````new server/internal/modules/project/adapter/postgres/projects_test.go
type webFixture struct {
	s                                          *postgresadapter.Store
	pool                                       *pgxpool.Pool
	acme, ops, web                             uuid.UUID
	alice, bob, carol, dave, erin, frank, gina uuid.UUID
}

// newWebFixture stores webFixture's rows, alice's. ops first: its rows come
// first in the tables, so a join that loses a predicate reads them before
// web's; everyone but erin is its admin. In web: carol's membership is
// stored first but began last; gina's began with dave's and has the
// smaller id, stored after his; alice's began between. bob's ended,
// frank's was deleted; carol's display settings were deleted; erin was
// never a member.
func newWebFixture(t *testing.T) webFixture {
	t.Helper()
````

````old server/internal/modules/project/adapter/postgres/projects_test.go
	alice, bob, carol, dave, erin, frank, gina := ids[0], ids[1], ids[2], ids[3], ids[4], ids[5], ids[6]
	acme := newWorkspace(t, pool, "acme")
	// ops first: its rows come first in the tables, so a join that loses a
	// predicate reads them before web's.
	ops := newProject(t, s, acme, "Ops", "OPS", alice)
	web := newProject(t, s, acme, "Web", "WEB", alice)
````
````new server/internal/modules/project/adapter/postgres/projects_test.go
	f := webFixture{s: s, pool: pool, alice: ids[0], bob: ids[1], carol: ids[2], dave: ids[3], erin: ids[4], frank: ids[5], gina: ids[6]}
	f.acme = newWorkspace(t, pool, "acme")
	f.ops = newProject(t, s, f.acme, "Ops", "OPS", f.alice)
	f.web = newProject(t, s, f.acme, "Web", "WEB", f.alice)
````

````old server/internal/modules/project/adapter/postgres/projects_test.go
		if err := s.CreateMember(ctx, app.MemberRow{ID: id, WorkspaceID: acme, ProjectID: project, MemberID: user, Role: role,
			CreatedBy: alice, Now: at}); err != nil {
````
````new server/internal/modules/project/adapter/postgres/projects_test.go
		if err := s.CreateMember(ctx, app.MemberRow{ID: id, WorkspaceID: f.acme, ProjectID: project, MemberID: user, Role: role,
			CreatedBy: f.alice, Now: at}); err != nil {
````

````old server/internal/modules/project/adapter/postgres/projects_test.go
		if err := s.CreatePreferences(ctx, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: project, UserID: user,
			SortOrder: sortOrder, CreatedBy: alice, Now: at}); err != nil {
````
````new server/internal/modules/project/adapter/postgres/projects_test.go
		if err := s.CreatePreferences(ctx, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: f.acme, ProjectID: project, UserID: user,
			SortOrder: sortOrder, CreatedBy: f.alice, Now: at}); err != nil {
````

````old server/internal/modules/project/adapter/postgres/projects_test.go
	for _, u := range []uuid.UUID{alice, bob, carol, dave, frank, gina} {
		member(uuid.NewV7(), ops, u, shared.RoleAdmin, now, -1)
	}
	// In web: carol's membership is stored first but began last; gina's
	// began with dave's and has the smaller id, stored after his; alice's
	// began between. bob's ended, frank's was deleted; carol's display
	// settings were deleted; erin was never a member.
````
````new server/internal/modules/project/adapter/postgres/projects_test.go
	for _, u := range []uuid.UUID{f.alice, f.bob, f.carol, f.dave, f.frank, f.gina} {
		member(uuid.NewV7(), f.ops, u, shared.RoleAdmin, now, -1)
	}
````

````old server/internal/modules/project/adapter/postgres/projects_test.go
	member(uuid.NewV7(), web, carol, shared.RoleGuest, now.Add(2*time.Minute), 30)
	member(uuid.NewV7(), web, dave, shared.RoleMember, now, 40)
	member(ginasID, web, gina, shared.RoleMember, now, 60)
	member(uuid.NewV7(), web, alice, shared.RoleAdmin, now.Add(time.Minute), 10)
	member(uuid.NewV7(), web, bob, shared.RoleMember, now, 20)
	member(uuid.NewV7(), web, frank, shared.RoleMember, now, 50)
	exec(t, pool, "UPDATE project_members SET is_active = false WHERE project_id = $1 AND member_id = $2", web, bob)
	exec(t, pool, "UPDATE project_members SET deleted_at = $3 WHERE project_id = $1 AND member_id = $2", web, frank, now)
	exec(t, pool, "UPDATE project_user_properties SET deleted_at = $3 WHERE project_id = $1 AND user_id = $2", web, carol, now)
	members := []uuid.UUID{gina, dave, alice, carol}
````
````new server/internal/modules/project/adapter/postgres/projects_test.go
	member(uuid.NewV7(), f.web, f.carol, shared.RoleGuest, now.Add(2*time.Minute), 30)
	member(uuid.NewV7(), f.web, f.dave, shared.RoleMember, now, 40)
	member(ginasID, f.web, f.gina, shared.RoleMember, now, 60)
	member(uuid.NewV7(), f.web, f.alice, shared.RoleAdmin, now.Add(time.Minute), 10)
	member(uuid.NewV7(), f.web, f.bob, shared.RoleMember, now, 20)
	member(uuid.NewV7(), f.web, f.frank, shared.RoleMember, now, 50)
	exec(t, pool, "UPDATE project_members SET is_active = false WHERE project_id = $1 AND member_id = $2", f.web, f.bob)
	exec(t, pool, "UPDATE project_members SET deleted_at = $3 WHERE project_id = $1 AND member_id = $2", f.web, f.frank, now)
	exec(t, pool, "UPDATE project_user_properties SET deleted_at = $3 WHERE project_id = $1 AND user_id = $2", f.web, f.carol, now)
	return f
}

// GetProject answers the caller's view (M3 design 3.19): his role and his
// place in his sidebar only while his membership is active, and the active
// members in the order they became members, then by the membership's id;
// on webFixture's web.
func TestGetProject(t *testing.T) {
	f := newWebFixture(t)
	s, pool, web := f.s, f.pool, f.web
	alice, bob, carol, dave, erin, frank := f.alice, f.bob, f.carol, f.dave, f.erin, f.frank
	members := []uuid.UUID{f.gina, dave, alice, carol}
````

`server/internal/modules/project/adapter/postgres/list_test.go`（新文件，115 行）：

````file server/internal/modules/project/adapter/postgres/list_test.go
package postgresadapter_test

import (
	"context"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListProjects answers each project as GetProject reads it (M3 design
// 3.12, 3.19): the members, the caller's role and his place in his
// sidebar. On webFixture, every account lists ops and web, each the row
// GetProject answers him.
func TestListProjectsAnswersEachAsGetProjectDoes(t *testing.T) {
	f := newWebFixture(t)
	ctx := context.Background()
	for _, user := range []uuid.UUID{f.alice, f.bob, f.carol, f.dave, f.erin, f.frank, f.gina} {
		list, err := f.s.ListProjects(ctx, f.acme, user, domain.Visibility{All: true, Public: true}, false)
		if err != nil || len(list) != 2 {
			t.Errorf("ListProjects(%s) = %d projects, %v; want ops and web", user, len(list), err)
			continue
		}
		for _, p := range list {
			want, found, err := f.s.GetProject(ctx, p.ID, user)
			if err != nil || !found || jsonOf(t, p) != jsonOf(t, want) {
				t.Errorf("ListProjects(%s) answers %s; GetProject answers %s, %v, %v", user, jsonOf(t, p), jsonOf(t, want), found, err)
			}
		}
	}
}

// ListProjects: of the workspace's undeleted projects, the archived ones
// or the others, those the visibility lets the user see besides those he
// is an active member of, each as he sees it; by his place in his sidebar,
// the projects he is not a member of last, then by name. The fixture's
// projects are stored in another order than the answer's.
func TestListProjects(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	private := func(id uuid.UUID) { exec(t, pool, "UPDATE projects SET network = 0 WHERE id = $1", id) }
	join := func(project, user uuid.UUID, sortOrder float64) {
		t.Helper()
		if err := s.CreateMember(ctx, app.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: project, MemberID: user,
			Role: shared.RoleMember, CreatedBy: alice, Now: now}); err != nil {
			t.Fatal(err)
		}
		if err := s.CreatePreferences(ctx, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: project, UserID: user,
			SortOrder: sortOrder, CreatedBy: alice, Now: now}); err != nil {
			t.Fatal(err)
		}
	}
	// Alice is a member of mid (place -5) and zeta (10), both private, and
	// of kilo (10), public, and was one of ended, whose membership ended;
	// apple and pear are public, secret private; archived, deleted and
	// beta's are not in this list. A name is unique in a workspace, so the
	// id decides no order here.
	zeta := newProject(t, s, acme, "Zeta", "ZETA", alice)
	private(zeta)
	join(zeta, alice, 10)
	pear := newProject(t, s, acme, "Pear", "PEAR", alice)
	mid := newProject(t, s, acme, "Mid", "MID", alice)
	private(mid)
	join(mid, alice, -5)
	secret := newProject(t, s, acme, "Secret", "SEC", alice)
	private(secret)
	ended := newProject(t, s, acme, "Ended", "END", alice)
	private(ended)
	join(ended, alice, 1)
	exec(t, pool, "UPDATE project_members SET is_active = false WHERE project_id = $1 AND member_id = $2", ended, alice)
	kilo := newProject(t, s, acme, "Kilo", "KILO", alice)
	join(kilo, alice, 10)
	apple := newProject(t, s, acme, "Apple", "APPLE", alice)
	archived := newProject(t, s, acme, "Old", "OLD", alice)
	exec(t, pool, "UPDATE projects SET archived_at = now() WHERE id = $1", archived)
	deleted := newProject(t, s, acme, "Gone", "GONE", alice)
	exec(t, pool, "UPDATE projects SET deleted_at = now() WHERE id = $1", deleted)
	newProject(t, s, beta, "Beta", "BETA", alice)

	tests := []struct {
		name     string
		user     uuid.UUID
		v        domain.Visibility
		archived bool
		want     []uuid.UUID
	}{
		{"everything", alice, domain.Visibility{All: true, Public: true}, false, []uuid.UUID{mid, kilo, zeta, apple, ended, pear, secret}},
		{"the public ones", alice, domain.Visibility{Public: true}, false, []uuid.UUID{mid, kilo, zeta, apple, pear}},
		{"hers alone", alice, domain.Visibility{}, false, []uuid.UUID{mid, kilo, zeta}},
		{"another's", bob, domain.Visibility{Public: true}, false, []uuid.UUID{apple, kilo, pear}},
		{"the archived ones", alice, domain.Visibility{All: true, Public: true}, true, []uuid.UUID{archived}},
	}
	for _, tt := range tests {
		list, err := s.ListProjects(ctx, acme, tt.user, tt.v, tt.archived)
		var got []uuid.UUID
		for _, p := range list {
			got = append(got, p.ID)
		}
		if err != nil || !slices.Equal(got, tt.want) {
			t.Errorf("%s: ListProjects() = %v, %v; want %v", tt.name, got, err, tt.want)
		}
	}
	// Each as the user sees it: his role and place in his own, none in the
	// others; the members.
	list, err := s.ListProjects(ctx, acme, alice, domain.Visibility{Public: true}, false)
	if err != nil || len(list) != 5 || list[0].MemberRole == nil || *list[0].MemberRole != shared.RoleMember || list[0].SortOrder == nil ||
		*list[0].SortOrder != -5 || !slices.Equal(list[0].MemberIDs, []uuid.UUID{alice}) || list[3].MemberRole != nil || list[3].SortOrder != nil {
		t.Errorf("ListProjects() = %+v, %v; want mid with her role and place, apple without", list, err)
	}
}
````

`server/internal/modules/project/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/project/adapter/http/handler.go
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)
````
````new server/internal/modules/project/adapter/http/handler.go
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

// ListProjectsUseCase is app.ListProjects.
type ListProjectsUseCase interface {
	Execute(ctx context.Context, slug string, archived bool) ([]domain.Project, error)
}
````

````old server/internal/modules/project/adapter/http/handler.go
type UseCases struct {
````
````new server/internal/modules/project/adapter/http/handler.go
type UseCases struct {
	ListProjects    ListProjectsUseCase
````

`server/internal/modules/project/adapter/http/projects.go`（修改，1 处）：

````old server/internal/modules/project/adapter/http/projects.go
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)
````
````new server/internal/modules/project/adapter/http/projects.go
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// ListProjects serves GET /api/v0/workspaces/{slug}/projects.
func (h handler) ListProjects(ctx context.Context, req gen.ListProjectsRequestObject) (gen.ListProjectsResponseObject, error) {
	archived := req.Params.Archived != nil && *req.Params.Archived
	list, err := h.uc.ListProjects.Execute(ctx, req.Slug, archived)
	if err != nil {
		return nil, err
	}
	out := gen.ListProjects200JSONResponse{Data: make([]gen.Project, len(list))}
	for i, p := range list {
		out.Data[i] = project(p)
	}
	return out, nil
}
````

`server/internal/modules/project/adapter/http/handler_test.go`（修改，5 处）：

````old server/internal/modules/project/adapter/http/handler_test.go
	"context"
````
````new server/internal/modules/project/adapter/http/handler_test.go
	"context"
	"fmt"
````

````old server/internal/modules/project/adapter/http/handler_test.go
type fakes struct {
````
````new server/internal/modules/project/adapter/http/handler_test.go
type fakes struct {
	list   *fakeList
````

````old server/internal/modules/project/adapter/http/handler_test.go
	check  *fakeCheck
````
````new server/internal/modules/project/adapter/http/handler_test.go
	check  *fakeCheck
}

type fakeList struct {
	calls []string // "caller slug archived"
	lists map[string][]domain.Project
	err   error
}

func (f *fakeList) Execute(ctx context.Context, slug string, archived bool) ([]domain.Project, error) {
	f.calls = append(f.calls, fmt.Sprintf("%s %s %v", caller(ctx), slug, archived))
	return f.lists[caller(ctx)], f.err
````

````old server/internal/modules/project/adapter/http/handler_test.go
		t.Fatal(err)
	}
````
````new server/internal/modules/project/adapter/http/handler_test.go
		t.Fatal(err)
	}
	if f.list == nil {
		f.list = &fakeList{}
	}
````

````old server/internal/modules/project/adapter/http/handler_test.go
	httpadapter.Register(router, api, httpadapter.UseCases{CreateProject: f.create, GetProject: f.get, CheckIdentifier: f.check})
````
````new server/internal/modules/project/adapter/http/handler_test.go
	httpadapter.Register(router, api, httpadapter.UseCases{ListProjects: f.list, CreateProject: f.create, GetProject: f.get,
		CheckIdentifier: f.check})
````

`server/internal/modules/project/adapter/http/projects_test.go`（修改，1 处）：

````old server/internal/modules/project/adapter/http/projects_test.go
			t.Errorf("GET = %d %s, want %d %s", res.StatusCode, body, tt.status, tt.want)
		}
	}
}

````
````new server/internal/modules/project/adapter/http/projects_test.go
			t.Errorf("GET = %d %s, want %d %s", res.StatusCode, body, tt.status, tt.want)
		}
	}
}

// The caller, the path's workspace and archived, false when absent, go to
// the use case; the answer is its list, empty as [].
func TestListProjects(t *testing.T) {
	list := &fakeList{lists: map[string][]domain.Project{"alice": {web, bare}}}
	h := newServer(t, fakes{list: list})
	for _, tt := range []struct {
		token, query string
		want         string
	}{
		{"alice", "", `{"data":[` + webJSON + `,` + bareJSON + `]}`},
		{"bob", "?archived=true", `{"data":[]}`},
		{"bob", "?archived=false", `{"data":[]}`},
	} {
		res, body := do(t, h, request(http.MethodGet, "/api/v0/workspaces/acme/projects"+tt.query, tt.token, ""))
		if res.StatusCode != http.StatusOK || body != tt.want+"\n" {
			t.Errorf("%s GET %s = %d %s, want 200 %s", tt.token, tt.query, res.StatusCode, body, tt.want)
		}
	}
	if want := []string{"alice acme false", "bob acme true", "bob acme false"}; !slices.Equal(list.calls, want) {
		t.Errorf("calls = %q, want %q", list.calls, want)
	}
	h = newServer(t, fakes{list: &fakeList{err: domain.ErrWorkspaceNotFound}})
	res, body := do(t, h, request(http.MethodGet, "/api/v0/workspaces/acme/projects", "alice", ""))
	if want := `{"status":404,"code":"workspace.not_found","title":"Not Found","detail":"The workspace does not exist, or you are not a member of it."}`; res.StatusCode != http.StatusNotFound || body != want+"\n" {
		t.Errorf("GET = %d %s, want 404 %s", res.StatusCode, body, want)
	}
}

````

`server/internal/modules/project/module.go`（修改，2 处）：

````old server/internal/modules/project/module.go
// brings creating and reading projects and checking an identifier, carries
// out the workspace module's cascades on the projects (ProjectCascade), and
// offers the access module its reads of a project (ProjectAccess).
````
````new server/internal/modules/project/module.go
// brings listing, creating and reading projects and checking an identifier,
// carries out the workspace module's cascades on the projects
// (ProjectCascade), and offers the access module its reads of a project
// (ProjectAccess).
````

````old server/internal/modules/project/module.go
		}),
````
````new server/internal/modules/project/module.go
		}),
		ListProjects:    app.NewListProjects(d.Workspaces, store, d.Authorizer),
````

- [ ] **Step 4: 矩阵和可见性一致**

`server/internal/bootstrap/permission_matrix_seeded_test.go`（修改，4 处）：

````old server/internal/bootstrap/permission_matrix_seeded_test.go
// and one archived; gone's, deleted with it.
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
// and one archived; gone's, deleted with it; other's, which no list of
// acme's may show.
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
	{"gone/project", "Web", "WEB", projectdomain.NetworkPublic},
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
	{"gone/project", "Web", "WEB", projectdomain.NetworkPublic},
	{"other/project", "Other", "OTH", projectdomain.NetworkPublic},
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
// project's admin; gone's admin in gone's project.
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
// project's admin; each other workspace's admin in its project.
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
	{"gone/project", callerDeleted, shared.RoleAdmin},
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
	{"gone/project", callerDeleted, shared.RoleAdmin}, {"other/project", callerNever, shared.RoleAdmin},
````

`server/internal/bootstrap/permission_matrix_project_test.go`（修改，3 处）：

````old server/internal/bootstrap/permission_matrix_project_test.go
	"net/http"
````
````new server/internal/bootstrap/permission_matrix_project_test.go
	"net/http"
	"slices"
````

````old server/internal/bootstrap/permission_matrix_project_test.go
	return []matrixRow{
````
````new server/internal/bootstrap/permission_matrix_project_test.go
	return []matrixRow{
		{op: "listProjects", request: toWorkspace(http.MethodGet, "/projects", ""), cells: inWorkspace(cellOK, cellOK, cellOK),
			check: listsTheProjects(false)},
		{op: "listProjects", variant: "archived", request: toWorkspace(http.MethodGet, "/projects?archived=true", ""),
			cells: inWorkspace(cellOK, cellOK, cellOK), check: listsTheProjects(true)},
````

````old server/internal/bootstrap/permission_matrix_project_test.go
}

// identifierAvailable: the answer is want.
````
````new server/internal/bootstrap/permission_matrix_project_test.go
}

// listsTheProjects: acme's projects the column's caller sees, the archived
// one alone or the others (9.2): every one to the admin, the public one to
// the member, those he is a member of to the guest; by name, as every
// place in a sidebar is the same.
func listsTheProjects(archived bool) func(t *testing.T, c caller, _ seeded, answer string) {
	return func(t *testing.T, c caller, _ seeded, answer string) {
		var list struct {
			Data []struct {
				Name string `json:"name"`
			} `json:"data"`
		}
		decodeAnswer(t, answer, &list)
		var got []string
		for _, p := range list.Data {
			got = append(got, p.Name)
		}
		want := map[caller][]string{callerAdmin: {"Secret", "Web"}, callerMember: {"Web"}, callerGuest: {"Secret", "Web"}}[c]
		if archived {
			want = map[caller][]string{callerAdmin: {"Old"}, callerMember: {"Old"}}[c]
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s lists %q, want %q", c, got, want)
		}
	}
}

// identifierAvailable: the answer is want.
````

`server/internal/bootstrap/project_visibility_test.go`（新文件，63 行）：

````file server/internal/bootstrap/project_visibility_test.go
package bootstrap

import (
	"maps"
	"net/http"
	"slices"
	"strconv"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// The list and the decision agree (M3 design 3.4, 9.3): for every account
// of the matrix, of each kind of 9.2, acme's projects that listProjects
// lists, archived or not, are exactly those that getProject lets him read
// in the same state. The list applies the visibility in its query
// (domain.Visibility), the decision in access's rule: here the two cannot
// part. Neither side is empty for every account, nor full.
func TestListingProjectsIsReadingEach(t *testing.T) {
	d := prepareMatrix(t)
	contract := apitest.Load(t)
	base := startApp(t, d.config(t, pgtest.NewDatabaseFrom(t, d.url), nil), migrations.FS())
	s := d.seeded.in(t)
	projects := map[string]bool{"acme/public": false, "acme/private": false, "acme/archived": true} // key: archived
	var sizes []int
	for _, account := range matrixAccounts {
		token := d.tokens[account]
		for _, archived := range []bool{false, true} {
			status, body := call(t, contract, http.MethodGet, base+"/api/v0/workspaces/acme/projects?archived="+strconv.FormatBool(archived), token, "")
			var list struct {
				Data []struct {
					ID uuid.UUID `json:"id"`
				} `json:"data"`
			}
			if status == http.StatusOK {
				decodeAnswer(t, body, &list)
			}
			var listed, read []uuid.UUID
			for _, p := range list.Data {
				listed = append(listed, p.ID)
			}
			for _, key := range slices.Sorted(maps.Keys(projects)) {
				id := s.project(key)
				if status, _ := call(t, contract, http.MethodGet, base+"/api/v0/projects/"+id.String(), token, ""); status == http.StatusOK &&
					projects[key] == archived {
					read = append(read, id)
				}
			}
			slices.SortFunc(listed, func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) })
			slices.SortFunc(read, func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) })
			if !slices.Equal(listed, read) {
				t.Errorf("%s, archived %v: lists %v, reads %v", account, archived, listed, read)
			}
			sizes = append(sizes, len(read))
		}
	}
	if slices.Min(sizes) != 0 || slices.Max(sizes) != 2 {
		t.Errorf("the accounts read %v projects; want some none and some both of acme's", sizes)
	}
}
````

- [ ] **Step 5: 测试、lint、前端检查**

Run: `go -C server test -count=1 ./internal/modules/project/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -v -run 'TestPermissionMatrix$' ./internal/bootstrap/`
Expected: `ok`；记下 `TestPermissionMatrix` 和 `TestPermissionMatrix/prepare` 的耗时（9.2 的预算是 20–30 秒）。

Run: `go -C server test -count=1 -run 'TestListingProjectsIsReadingEach|TestThePermissionMatrixCoversEveryOperation|TestEveryActionHasARuleAndEveryRuleAnAction' ./internal/bootstrap/`
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
git add api/modules/project.yaml server/internal/bootstrap/permission_matrix_project_test.go server/internal/bootstrap/permission_matrix_seeded_test.go server/internal/bootstrap/project_visibility_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/project/adapter/http/handler.go server/internal/modules/project/adapter/http/handler_test.go server/internal/modules/project/adapter/http/projects.go server/internal/modules/project/adapter/http/projects_test.go server/internal/modules/project/adapter/postgres/list_test.go server/internal/modules/project/adapter/postgres/projects.go server/internal/modules/project/adapter/postgres/projects_test.go server/internal/modules/project/adapter/postgres/queries/projects.sql server/internal/modules/project/app/list_projects.go server/internal/modules/project/app/list_projects_test.go server/internal/modules/project/app/ports.go server/internal/modules/project/domain/actions.go server/internal/modules/project/domain/visibility.go server/internal/modules/project/domain/visibility_test.go server/internal/modules/project/module.go api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/server.gen.go server/internal/modules/project/adapter/postgres/gen/projects.sql.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P4a): GET /api/v0/workspaces/{slug}/projects lists what project.read lets its caller read

Every active member of the workspace may list; each sees in the list
what the decision lets him read (M3 design 3.4): the admins every
project, the members the public ones too, the guests their own; the
archived ones alone with archived=true. The query applies the workspace
role the decision read. A test on the wired app holds the list equal to
getProject for every account of the matrix (9.3).

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 |
|---|---|
| 查询不看 `archived`；看反了；忽略管理员的可见性；成员看得到私密项目；已结束的成员关系算；列出已删除的项目；列出每个工作区的 | `TestListProjects`（存储；除已删除的一条，另有 `TestListingProjectsIsReadingEach`，多数另有 `TestPermissionMatrix`） |
| 不是成员的项目排在前面；只按名称 | `TestListProjects`（存储） |
| 与 `GetProject` 相同的列：成员列表不看项目、有效、未删除，只按 id、只按时刻排；调用者的成员关系不看项目、账户、未删除；显示设置不看项目、账户、未删除 | `TestListProjectsAnswersEachAsGetProjectDoes` |
| 每个调用者都按工作区管理员列出；不传 `archived`；不判定 `project.list` | `TestListProjects`（用例）、`TestPermissionMatrix`（前两个另有 `TestListingProjectsIsReadingEach`，最后一个另有 `TestListProjectsRefuses`） |
| 看不到不答 `workspace.not_found`；判定的失败被吞掉；目录的失败答成 404；读加锁；没有调用者照常进行 | `TestListProjectsRefuses`（读加锁：`TestListProjects`） |
| `VisibilityOf` 让成员看全部；让访客看公开的 | `TestVisibilityOf`、`TestListingProjectsIsReadingEach`（前者另有 `TestPermissionMatrix`） |
| `project.list` 只给管理员和成员 | `TestEveryRuleDecidesItsCells`、`TestPermissionMatrix` |
| handler 不读 `archived`；空列表答成 `null` | `TestListProjects`（HTTP） |

**Done when:** 列表的每个谓词、排序由存储测试核对，列表与逐个判定在组合出的 app 上对矩阵的每个账户一致；矩阵 181 格，耗时记下；前端检查通过。

---

### Task 12: `DemoteToGuest`：存储、连带；`updateWorkspaceMember` 改为访客时调用

**Files:**
- Create: `server/internal/bootstrap/demotion_test.go`、`server/internal/modules/project/adapter/postgres/demote_test.go`
- Modify: `server/internal/bootstrap/interleaving_roles_test.go`、`server/internal/modules/project/adapter/postgres/cascade.go`、`server/internal/modules/project/adapter/postgres/queries/cascade.sql`、`server/internal/modules/project/app/cascade.go`、`server/internal/modules/project/app/cascade_test.go`、`server/internal/modules/project/app/ports.go`、`server/internal/modules/project/module.go`、`server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go`、`server/internal/modules/workspace/app/clock_test.go`、`server/internal/modules/workspace/app/fakes_projects_test.go`、`server/internal/modules/workspace/app/list_members_test.go`、`server/internal/modules/workspace/app/ports.go`、`server/internal/modules/workspace/app/update_member.go`、`server/internal/modules/workspace/app/update_member_test.go`、`server/internal/modules/workspace/module.go`
- Generate: `server/internal/modules/project/adapter/postgres/gen/cascade.sql.go`

**Interfaces:**
- Produces（spec 2.12，M3 设计 3.3、3.6 约定五，裁定 G1）：查询 `LockMemberProjects`（工作区的未删除项目中他有未删除成员关系的，有效的、已结束的、已是访客的都算，已归档的项目也算，按 id `FOR NO KEY UPDATE`）、`DemoteMemberships`（这些项目里他未删除的、不是访客的成员关系改为 5，写 `updated_at`、`updated_by_id`；访客的行不动）；`app.MemberDemoter`；`app.NewCascade(projects WorkspaceProjectsDeleter, members MemberDemoter)`；`(*Cascade).DemoteToGuest(ctx, workspaceID, userID, by, now) error`：先锁，没有锁到项目就什么都不写，否则一条语句写；`project.Cascade`、`workspace/app.ProjectCascade` 加 `DemoteToGuest`。
- `NewUpdateWorkspaceMember(members, projects ProjectCascade, profiles, auth, tx, clock)`：锁之后读一次时钟，改角色；角色是访客时 `DemoteToGuest(工作区, 成员, 管理员, 同一个时刻)`，在写之后、读资料之前（P2 review 第 6 节）；失败原样返回，整个事务回滚。
- 使用者：Task 13 的接受；P5 的 `EndMemberships`（同一个端口的第三个方法）。

**Tests:**
- `project/adapter/postgres/demote_test.go`：`TestDemotingAMemberToGuest`（锁按 id 顺序答 `acme` 的五个项目中他有未删除成员关系的三个，已归档的一个在内；锁到事务结束，`FOR SHARE` 要等、外键检查的 `FOR KEY SHARE` 不等，别的项目不锁；写让这三行成为访客，已结束的仍结束，时刻、账户是给的；他已是访客的行、alice 的行、他已删除的行、已删除项目的行、`beta` 的行每一列都不变）；`TestLockMemberProjectsLocksInIDOrder`（行在表里、在名称和标识的索引里的顺序都与 id 相反时仍按 id 取锁）。
- `project/app/cascade_test.go`：`TestDemoteToGuest`（先锁再写，账户、时刻是调用者的，两步都在调用者的事务里：`fakeDemoter` 经 Task 7 的 `callLog` 记下在事务之外的调用；没有锁到就不写；失败原样返回，之后什么都不运行）。
- `workspace/app`：`TestUpdateWorkspaceMemberLocksThenDecidesThenWrites` 对访客、成员、管理员三种新角色，只有访客调 `DemoteToGuest`，位置在改角色之后、读资料之前；`TestUpdateWorkspaceMemberFailsWithinTheTransaction` 加项目一步的失败；`TestEachWriteReadsTheClockUnderItsLock` 的改为访客用同一个时刻。
- `bootstrap/demotion_test.go`：`TestDemotingToGuestDemotesInTheWorkspacesProjects`（组合出的 app：bob 是 `acme`、`beta` 的成员，各领导一个项目；项目一步失败时 alice 改他的角色答 500，什么都不变；之后成功：他是 `acme` 的访客、`acme` 项目的访客，由 alice 在改角色的时刻写入，`beta` 的一切不变）。

- [ ] **Step 1: 查询**

`server/internal/modules/project/adapter/postgres/queries/cascade.sql`（修改，2 处）：

````old server/internal/modules/project/adapter/postgres/queries/cascade.sql
-- The steps of deleting a workspace's projects (M3 design 3.3, 3.6), each one statement under the workspace's FOR NO
-- KEY UPDATE, which deleteWorkspace took (convention 5): the rows of the workspace not deleted before, at the moment
-- and by the account of the workspace's deletion. Rows deleted before keep their moment.
````
````new server/internal/modules/project/adapter/postgres/queries/cascade.sql
-- ProjectCascade's statements (M3 design 3.3, 3.6), each under the workspace's FOR NO KEY UPDATE, which the caller
-- took. The steps of deleting a workspace's projects are one statement each (convention 5): the rows of the workspace
-- not deleted before, at the moment and by the account of the workspace's deletion. Rows deleted before keep their
-- moment.
````

````old server/internal/modules/project/adapter/postgres/queries/cascade.sql
UPDATE states
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL;

````
````new server/internal/modules/project/adapter/postgres/queries/cascade.sql
UPDATE states
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL;

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

- [ ] **Step 2: 生成**

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `5cd634e0b224c82b7e9ce21a21d5ae6b8d59a9994d6c3f3c93e4bbf10a8ad055` | 153 | `server/internal/modules/project/adapter/postgres/gen/cascade.sql.go` |

Run: `shasum -a 256 server/internal/modules/project/adapter/postgres/gen/cascade.sql.go`
Expected: 与上表相同。

- [ ] **Step 3: `project` 一侧**

`server/internal/modules/project/app/ports.go`（修改，1 处）：

````old server/internal/modules/project/app/ports.go
}

// WorkspaceProjectsDeleter soft-deletes a workspace's projects and the rows
````
````new server/internal/modules/project/app/ports.go
}

// MemberDemoter makes an account a guest in a workspace's projects (M3
// design 3.3, 3.6): his projects locked first, then his memberships of them
// in one statement. Each method runs in the transaction ctx carries.
type MemberDemoter interface {
	// LockMemberProjects locks FOR NO KEY UPDATE, in id order, workspaceID's
	// undeleted projects in which userID has an undeleted membership, active
	// or not, and returns their ids.
	LockMemberProjects(ctx context.Context, workspaceID, userID uuid.UUID) ([]uuid.UUID, error)
	// DemoteMemberships sets role 5, updated_at now and updated_by_id by on
	// userID's undeleted memberships of projectIDs, active or not, that are
	// not a guest's already.
	DemoteMemberships(ctx context.Context, projectIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error
}

// WorkspaceProjectsDeleter soft-deletes a workspace's projects and the rows
````

`server/internal/modules/project/app/cascade.go`（修改，3 处）：

````old server/internal/modules/project/app/cascade.go
	projects WorkspaceProjectsDeleter
````
````new server/internal/modules/project/app/cascade.go
	projects WorkspaceProjectsDeleter
	members  MemberDemoter
````

````old server/internal/modules/project/app/cascade.go
// NewCascade returns the cascade over projects.
func NewCascade(projects WorkspaceProjectsDeleter) *Cascade {
	return &Cascade{projects: projects}
````
````new server/internal/modules/project/app/cascade.go
// NewCascade returns the cascade over projects and members.
func NewCascade(projects WorkspaceProjectsDeleter, members MemberDemoter) *Cascade {
	return &Cascade{projects: projects, members: members}
````

````old server/internal/modules/project/app/cascade.go
}

// deletion is what deleting a workspace deletes of the projects, in the
````
````new server/internal/modules/project/app/cascade.go
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

// deletion is what deleting a workspace deletes of the projects, in the
````

`server/internal/modules/project/app/cascade_test.go`（修改，2 处）：

````old server/internal/modules/project/app/cascade_test.go
		err := app.NewCascade(f).DeleteWorkspaceProjects(context.Background(), workspace, by, now)
````
````new server/internal/modules/project/app/cascade_test.go
		err := app.NewCascade(f, &fakeDemoter{}).DeleteWorkspaceProjects(context.Background(), workspace, by, now)
````

````old server/internal/modules/project/app/cascade_test.go
	}
}

````
````new server/internal/modules/project/app/cascade_test.go
	}
}

// fakeDemoter is the memberships' repository: it records each call with its
// arguments, and " outside tx" when it ran outside the caller's
// transaction (callLog), answers LockMemberProjects with locked, and fails
// the call named in fail.
type fakeDemoter struct {
	locked []uuid.UUID
	log    callLog
	fail   string
}

func (f *fakeDemoter) call(ctx context.Context, name, format string, args ...any) error {
	f.log.add(ctx, name+" "+format, args...)
	if name == f.fail {
		return errDisk
	}
	return nil
}

func (f *fakeDemoter) LockMemberProjects(ctx context.Context, workspaceID, userID uuid.UUID) ([]uuid.UUID, error) {
	if err := f.call(ctx, "LockMemberProjects", "%s %s", workspaceID, userID); err != nil {
		return nil, err
	}
	return f.locked, nil
}

func (f *fakeDemoter) DemoteMemberships(ctx context.Context, projectIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error {
	return f.call(ctx, "DemoteMemberships", "%v %s by %s at %s", projectIDs, userID, by, now.Format(time.RFC3339Nano))
}

// DemoteToGuest locks the account's projects in the workspace, then
// demotes his memberships of the ones locked, by the caller's account at
// the caller's moment, both in the caller's transaction; with none locked
// it writes nothing. A failing call comes back as itself, and nothing runs
// after it.
func TestDemoteToGuest(t *testing.T) {
	workspace, user, by := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	now := time.Date(2026, 10, 1, 10, 0, 0, 123456000, time.UTC)
	projects := []uuid.UUID{uuid.NewV7(), uuid.NewV7()}
	lock := fmt.Sprintf("LockMemberProjects %s %s", workspace, user)
	demote := fmt.Sprintf("DemoteMemberships %v %s by %s at %s", projects, user, by, now.Format(time.RFC3339Nano))
	tests := []struct {
		name    string
		locked  []uuid.UUID
		fail    string
		wantErr error
		want    []string
	}{
		{"two projects", projects, "", nil, []string{lock, demote}},
		{"none", nil, "", nil, []string{lock}},
		{"the lock failing", projects, "LockMemberProjects", errDisk, []string{lock}},
		{"the write failing", projects, "DemoteMemberships", errDisk, []string{lock, demote}},
	}
	for _, tt := range tests {
		f := &fakeDemoter{locked: tt.locked, fail: tt.fail}
		inTx := context.WithValue(context.Background(), inTxKey{}, true)
		err := app.NewCascade(&fakeDeleter{}, f).DemoteToGuest(inTx, workspace, user, by, now)
		if !errors.Is(err, tt.wantErr) || (tt.wantErr == nil) != (err == nil) || !slices.Equal(f.log.calls, tt.want) {
			t.Errorf("%s: DemoteToGuest() = %v, the calls\n%q\nwant %v,\n%q", tt.name, err, f.log.calls, tt.wantErr, tt.want)
		}
	}
}

````

`server/internal/modules/project/adapter/postgres/cascade.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/cascade.go
		return fmt.Errorf("delete the workspace's states: %w", err)
	}
	return nil
}

````
````new server/internal/modules/project/adapter/postgres/cascade.go
		return fmt.Errorf("delete the workspace's states: %w", err)
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

`server/internal/modules/project/adapter/postgres/demote_test.go`（新文件，181 行）：

````file server/internal/modules/project/adapter/postgres/demote_test.go
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
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// seedMember writes user's membership of project, of role, active or not,
// written by user at now, and returns its id.
func seedMember(t *testing.T, pool *pgxpool.Pool, workspace, project, user uuid.UUID, role int, active bool) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	exec(t, pool, `INSERT INTO project_members (id, workspace_id, project_id, member_id, role, is_active, created_by_id, updated_by_id,
		created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $4, $4, $7, $7)`, id, workspace, project, user, role, active, now)
	return id
}

// waits reports whether a statement of another transaction on project's
// row, lock, waits for a lock held on it: NOWAIT answers lock_not_available
// (55P03) at once instead of waiting.
func waits(t *testing.T, pool *pgxpool.Pool, project uuid.UUID, lock string) bool {
	t.Helper()
	_, err := pool.Exec(context.Background(), "SELECT 1 FROM projects WHERE id = $1 "+lock+" NOWAIT", project)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "55P03" {
		return true
	}
	if err != nil {
		t.Fatal(err)
	}
	return false
}

// DemoteToGuest's two steps in one transaction (M3 design 3.3, 3.6). The
// lock returns, in id order, acme's undeleted projects, the archived one
// too, in which bob has an undeleted membership, active, ended, or a
// guest's already; until the transaction ends each of them is held FOR NO
// KEY UPDATE, which a FOR SHARE waits for and a foreign key's FOR KEY SHARE
// does not, and no other project is: not HR, where alice is a member and
// bob's membership is deleted. The write makes bob's memberships of those a
// guest's, at the moment and by the account given, the ended one still
// ended; his guest's one, alice's, his deleted ones, his one of a deleted
// project and his one in beta keep every column.
func TestDemotingAMemberToGuest(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	arch, docs := newProject(t, s, acme, "Arch", "ARCH", alice), newProject(t, s, acme, "Docs", "DOCS", alice)
	hr, gone := newProject(t, s, acme, "HR", "HR", alice), newProject(t, s, acme, "Gone", "GONE", alice)
	betas := newProject(t, s, beta, "Web", "WEB", alice)
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", arch, now)
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", gone, now)
	// bob's deleted memberships: of Web, before the one he has now, and of HR.
	for _, project := range []uuid.UUID{web, hr} {
		exec(t, pool, "UPDATE project_members SET deleted_at = $2 WHERE id = $1", seedMember(t, pool, acme, project, bob, 15, true), now)
	}
	demoted := map[uuid.UUID]bool{ // by id, whether active
		seedMember(t, pool, acme, web, bob, 20, true): true, seedMember(t, pool, acme, ops, bob, 15, false): false,
		seedMember(t, pool, acme, arch, bob, 15, true): true,
	}
	seedMember(t, pool, acme, docs, bob, 5, true)
	seedMember(t, pool, acme, web, alice, 20, true)
	seedMember(t, pool, acme, hr, alice, 20, true)
	seedMember(t, pool, acme, gone, bob, 20, true)
	seedMember(t, pool, beta, betas, bob, 20, true)
	kept := func() string {
		t.Helper()
		var rows string
		if err := pool.QueryRow(context.Background(), `SELECT string_agg(r::text, E'\n' ORDER BY r.id) FROM project_members r
			WHERE NOT r.id = ANY ($1)`, slices.Collect(maps.Keys(demoted))).Scan(&rows); err != nil {
			t.Fatal(err)
		}
		return rows
	}
	before := kept()
	later := now.Add(time.Hour)

	var locked []uuid.UUID
	err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		var err error
		if locked, err = s.LockMemberProjects(ctx, acme, bob); err != nil {
			return err
		}
		for name, p := range map[string]struct {
			id   uuid.UUID
			held bool
		}{"Web": {web, true}, "Ops": {ops, true}, "Arch": {arch, true}, "Docs": {docs, true}, "HR": {hr, false}, "Gone": {gone, false},
			"beta's Web": {betas, false}} {
			if share, keyShare := waits(t, pool, p.id, "FOR SHARE"), waits(t, pool, p.id, "FOR KEY SHARE"); share != p.held || keyShare {
				t.Errorf("%s while locked: a FOR SHARE waits %v, a FOR KEY SHARE waits %v; want %v, false", name, share, keyShare, p.held)
			}
		}
		return s.DemoteMemberships(ctx, locked, bob, alice, later)
	})
	if err != nil {
		t.Fatal(err)
	}

	if want := slices.SortedFunc(slices.Values([]uuid.UUID{web, ops, arch, docs}), uuid.UUID.Compare); !slices.Equal(locked, want) {
		t.Errorf("LockMemberProjects() = %v, want %v", locked, want)
	}
	for id, active := range demoted {
		var role int
		var isActive bool
		var updatedAt time.Time
		var updatedBy uuid.UUID
		if err := pool.QueryRow(context.Background(), "SELECT role, is_active, updated_at, updated_by_id FROM project_members WHERE id = $1", id).
			Scan(&role, &isActive, &updatedAt, &updatedBy); err != nil {
			t.Fatal(err)
		}
		if role != 5 || isActive != active || !updatedAt.Equal(later) || updatedBy != alice {
			t.Errorf("membership %s: role %d, active %v, at %v by %s; want 5, %v, at %v by alice", id, role, isActive, updatedAt, updatedBy, active, later)
		}
	}
	if after := kept(); after != before {
		t.Errorf("the other memberships:\n%s\nwant them as they were:\n%s", after, before)
	}
}

// LockMemberProjects takes its locks in the projects' id order, whatever
// order the rows lie in: Web has the smaller id, but lies after Alpha in
// the table and in the indexes on the name and on the identifier. Alpha's
// row is held. LockMemberProjects waits for it holding Web's, which a FOR
// SHARE then waits for; in any other order it would reach Alpha first and
// wait holding nothing.
func TestLockMemberProjectsLocksInIDOrder(t *testing.T) {
	s, pool := newStore(t)
	bob := newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, alpha := uuid.NewV7(), uuid.NewV7() // web drawn first: the smaller id
	for _, p := range []struct {
		id   uuid.UUID
		name string
	}{{alpha, "Alpha"}, {web, "Web"}} {
		exec(t, pool, "INSERT INTO projects (id, workspace_id, name, identifier) VALUES ($1, $2, $3::text, upper($3::text))", p.id, acme, p.name)
		seedMember(t, pool, acme, p.id, bob, 15, true)
	}
	held, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = held.Rollback(context.Background()) }()
	if _, err := held.Exec(context.Background(), "SELECT 1 FROM projects WHERE id = $1 FOR NO KEY UPDATE", alpha); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- postgres.NewTxManager(pool, 10*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
			_, err := s.LockMemberProjects(ctx, acme, bob)
			return err
		})
	}()
	pgtest.WaitForLockWaitOn(t, pool, "projects", 10*time.Second)

	if !waits(t, pool, web, "FOR SHARE") {
		t.Error("a FOR SHARE of Web while LockMemberProjects waits for Alpha does not wait; want Web locked first")
	}
	if err := held.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("LockMemberProjects() = %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("LockMemberProjects() did not end within 10s")
	}
}
````

`server/internal/modules/project/module.go`（修改，2 处）：

````old server/internal/modules/project/module.go
	DeleteWorkspaceProjects(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error
````
````new server/internal/modules/project/module.go
	DeleteWorkspaceProjects(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error
	// DemoteToGuest makes userID a guest in each of the workspace's projects
	// he has a membership of, ended ones too.
	DemoteToGuest(ctx context.Context, workspaceID, userID, by uuid.UUID, now time.Time) error
````

````old server/internal/modules/project/module.go
	return &Module{cascade: app.NewCascade(store), uc: httpadapter.UseCases{
````
````new server/internal/modules/project/module.go
	return &Module{cascade: app.NewCascade(store, store), uc: httpadapter.UseCases{
````

- [ ] **Step 4: `workspace` 一侧**

`server/internal/modules/workspace/app/ports.go`（修改，1 处）：

````old server/internal/modules/workspace/app/ports.go
	DeleteWorkspaceProjects(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error
````
````new server/internal/modules/workspace/app/ports.go
	DeleteWorkspaceProjects(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error
	// DemoteToGuest makes userID a guest in each of the workspace's projects
	// he has a membership of, ended ones too, at the moment and by the
	// account of the change of his workspace role to guest.
	DemoteToGuest(ctx context.Context, workspaceID, userID, by uuid.UUID, now time.Time) error
````

`server/internal/modules/workspace/app/update_member.go`（修改，5 处）：

````old server/internal/modules/workspace/app/update_member.go
	members  MemberUpdater
````
````new server/internal/modules/workspace/app/update_member.go
	members  MemberUpdater
	projects ProjectCascade
````

````old server/internal/modules/workspace/app/update_member.go
func NewUpdateWorkspaceMember(members MemberUpdater, profiles MemberProfiles, auth shared.Authorizer, tx shared.TxManager,
	clock Clock) *UpdateWorkspaceMember {
	return &UpdateWorkspaceMember{members: members, profiles: profiles, auth: auth, tx: tx, clock: clock}
````
````new server/internal/modules/workspace/app/update_member.go
func NewUpdateWorkspaceMember(members MemberUpdater, projects ProjectCascade, profiles MemberProfiles, auth shared.Authorizer,
	tx shared.TxManager, clock Clock) *UpdateWorkspaceMember {
	return &UpdateWorkspaceMember{members: members, projects: projects, profiles: profiles, auth: auth, tx: tx, clock: clock}
````

````old server/internal/modules/workspace/app/update_member.go
// then the change, at the time the clock gives under the lock.
````
````new server/internal/modules/workspace/app/update_member.go
// then the change, at the time the clock gives under the lock. A change to
// guest then makes him a guest in each of the workspace's projects he has a
// membership of, ended ones too (ProjectCascade.DemoteToGuest, M3 design
// 3.3), at the same time; a failure there rolls the change back.
````

````old server/internal/modules/workspace/app/update_member.go
// 3.6 convention 1) before the commit, so a failed read changes nothing.
// Demoting to guest does not touch projects yet: P4 adds that cascade here.
````
````new server/internal/modules/workspace/app/update_member.go
// 3.6 convention 1) before the commit, so a failed read changes nothing.
````

````old server/internal/modules/workspace/app/update_member.go
		if m, err = u.members.UpdateMemberRole(ctx, m.ID, role, actor.UserID, u.clock.Now()); err != nil {
			return err
````
````new server/internal/modules/workspace/app/update_member.go
		now := u.clock.Now()
		if m, err = u.members.UpdateMemberRole(ctx, m.ID, role, actor.UserID, now); err != nil {
			return err
		}
		if role == shared.RoleGuest {
			if err := u.projects.DemoteToGuest(ctx, m.WorkspaceID, m.MemberID, actor.UserID, now); err != nil {
				return err
			}
````

`server/internal/modules/workspace/app/fakes_projects_test.go`（修改，1 处）：

````old server/internal/modules/workspace/app/fakes_projects_test.go
	return nil
}

````
````new server/internal/modules/workspace/app/fakes_projects_test.go
	return nil
}

func (f *fakeProjects) DemoteToGuest(ctx context.Context, workspaceID, userID, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "DemoteToGuest %s %s by %s at %s", workspaceID, userID, by, now.Format(time.RFC3339Nano))
	if err := f.errs["DemoteToGuest"]; err != nil {
		return fmt.Errorf("demote the member's project memberships: %w", err)
	}
	return nil
}

````

`server/internal/modules/workspace/app/list_members_test.go`（修改，3 处）：

````old server/internal/modules/workspace/app/list_members_test.go
// membersFixture is ListWorkspaceMembers over fakes sharing one log. acme's
````
````new server/internal/modules/workspace/app/list_members_test.go
// membersFixture is ListWorkspaceMembers over fakes sharing one log, with
// the projects' cascade that UpdateWorkspaceMember takes besides. acme's
````

````old server/internal/modules/workspace/app/list_members_test.go
	workspaces *fakeWorkspaces
````
````new server/internal/modules/workspace/app/list_members_test.go
	workspaces *fakeWorkspaces
	projects   *fakeProjects
````

````old server/internal/modules/workspace/app/list_members_test.go
		}},
````
````new server/internal/modules/workspace/app/list_members_test.go
		}},
		projects: &fakeProjects{log: log},
````

`server/internal/modules/workspace/app/update_member_test.go`（修改，10 处）：

````old server/internal/modules/workspace/app/update_member_test.go
	return app.NewUpdateWorkspaceMember(f.workspaces, f.profiles, f.auth, tx, clockAt{at: clockNow}), f, tx
````
````new server/internal/modules/workspace/app/update_member_test.go
	return app.NewUpdateWorkspaceMember(f.workspaces, f.projects, f.profiles, f.auth, tx, clockAt{at: clockNow}), f, tx
````

````old server/internal/modules/workspace/app/update_member_test.go
// member's profile, all in one transaction (M3 design 3.6); the admin sees
// the address.
````
````new server/internal/modules/workspace/app/update_member_test.go
// member's profile, all in one transaction (M3 design 3.6); a change to
// guest makes him a guest in acme's projects in between, by alice at the
// role's time (M3 design 3.3), a change to another role leaves them. The
// admin sees the address.
````

````old server/internal/modules/workspace/app/update_member_test.go
	for _, role := range []shared.Role{shared.RoleGuest, shared.RoleAdmin} {
````
````new server/internal/modules/workspace/app/update_member_test.go
	at := clockNow.Format(time.RFC3339Nano)
	for _, tt := range []struct {
		role    shared.Role
		cascade []string
	}{
		{shared.RoleGuest, []string{fmt.Sprintf("DemoteToGuest %s %s by %s at %s", acme.ID, bob.ID, alice.ID, at)}},
		{shared.RoleMember, nil},
		{shared.RoleAdmin, nil},
	} {
````

````old server/internal/modules/workspace/app/update_member_test.go
		got, err := uc.Execute(as(alice), bobInAcme.ID, role)
````
````new server/internal/modules/workspace/app/update_member_test.go
		got, err := uc.Execute(as(alice), bobInAcme.ID, tt.role)
````

````old server/internal/modules/workspace/app/update_member_test.go
		want.Role = role
````
````new server/internal/modules/workspace/app/update_member_test.go
		want.Role = tt.role
````

````old server/internal/modules/workspace/app/update_member_test.go
			t.Errorf("to %d: Execute() = %+v, %v; want %+v", role, got, err, withUser(want, true))
````
````new server/internal/modules/workspace/app/update_member_test.go
			t.Errorf("to %d: Execute() = %+v, %v; want %+v", tt.role, got, err, withUser(want, true))
````

````old server/internal/modules/workspace/app/update_member_test.go
		wantCalls := append(lockedMemberCalls(alice, bobInAcme),
			fmt.Sprintf("UpdateMemberRole %s to %d by %s at %s", bobInAcme.ID, role, alice.ID, clockNow.Format(time.RFC3339Nano)),
			fmt.Sprintf("PublicProfiles %v", []uuid.UUID{bob.ID}))
````
````new server/internal/modules/workspace/app/update_member_test.go
		wantCalls := slices.Concat(lockedMemberCalls(alice, bobInAcme),
			[]string{fmt.Sprintf("UpdateMemberRole %s to %d by %s at %s", bobInAcme.ID, tt.role, alice.ID, at)}, tt.cascade,
			[]string{fmt.Sprintf("PublicProfiles %v", []uuid.UUID{bob.ID})})
````

````old server/internal/modules/workspace/app/update_member_test.go
			t.Errorf("to %d: calls = %q in %d transactions, want %q in one", role, f.log.calls, tx.calls, wantCalls)
````
````new server/internal/modules/workspace/app/update_member_test.go
			t.Errorf("to %d: calls = %q in %d transactions, want %q in one", tt.role, f.log.calls, tx.calls, wantCalls)
````

````old server/internal/modules/workspace/app/update_member_test.go
// A failed write, a failed read of the profile, and a member without an
// account each fail the transaction, which the database then rolls back:
````
````new server/internal/modules/workspace/app/update_member_test.go
// A failed write, a failed step of the projects, a failed read of the
// profile, and a member without an account each fail the transaction, which
// the database then rolls back, the role's change with it:
````

````old server/internal/modules/workspace/app/update_member_test.go
		{"the write", func(f *membersFixture) { f.workspaces.roleErr = failure }, failure},
````
````new server/internal/modules/workspace/app/update_member_test.go
		{"the write", func(f *membersFixture) { f.workspaces.roleErr = failure }, failure},
		{"the projects' step", func(f *membersFixture) { f.projects.errs = map[string]error{"DemoteToGuest": failure} }, failure},
````

`server/internal/modules/workspace/app/clock_test.go`（修改，3 处）：

````old server/internal/modules/workspace/app/clock_test.go
// every step. The clock logs its read among the fakes' calls.
````
````new server/internal/modules/workspace/app/clock_test.go
// every step, and a change to guest for the projects' step. The clock logs
// its read among the fakes' calls.
````

````old server/internal/modules/workspace/app/clock_test.go
			_, err := app.NewUpdateWorkspaceMember(f.workspaces, f.profiles, f.auth, &fakeTx{}, clockAt{clockNow, f.log}).
````
````new server/internal/modules/workspace/app/clock_test.go
			_, err := app.NewUpdateWorkspaceMember(f.workspaces, f.projects, f.profiles, f.auth, &fakeTx{}, clockAt{clockNow, f.log}).
````

````old server/internal/modules/workspace/app/clock_test.go
			fmt.Sprintf("UpdateMemberRole %s to %d by %s at %s", bobInAcme.ID, shared.RoleGuest, alice.ID, at),
````
````new server/internal/modules/workspace/app/clock_test.go
			fmt.Sprintf("UpdateMemberRole %s to %d by %s at %s", bobInAcme.ID, shared.RoleGuest, alice.ID, at),
			fmt.Sprintf("DemoteToGuest %s %s by %s at %s", acme.ID, bob.ID, alice.ID, at),
````

`server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go`（修改，2 处）：

````old server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
// noProjects is the project module's cascade, the last step, which the
// failure comes before.
````
````new server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
// noProjects is the project module's cascade: the deletion's last step,
// which the failure comes before; a deletion demotes no one.
````

````old server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
	return errors.New("the projects' step ran after the failed one")
````
````new server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
	return errors.New("the projects' step ran after the failed one")
}

func (noProjects) DemoteToGuest(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, time.Time) error {
	return errors.New("a deletion demoted a member")
````

`server/internal/modules/workspace/module.go`（修改，1 处）：

````old server/internal/modules/workspace/module.go
		UpdateMember:      app.NewUpdateWorkspaceMember(store, d.Profiles, d.Authorizer, d.Tx, d.Clock),
````
````new server/internal/modules/workspace/module.go
		UpdateMember:      app.NewUpdateWorkspaceMember(store, d.Projects, d.Profiles, d.Authorizer, d.Tx, d.Clock),
````

- [ ] **Step 5: 组合出的测试**

`server/internal/bootstrap/interleaving_roles_test.go`（修改，3 处）：

````old server/internal/bootstrap/interleaving_roles_test.go
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
````
````new server/internal/bootstrap/interleaving_roles_test.go
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
````

````old server/internal/bootstrap/interleaving_roles_test.go
// change is updateWorkspaceMember over members, with identity's profiles and
// the Authorizer as bootstrap wires them.
````
````new server/internal/bootstrap/interleaving_roles_test.go
// change is updateWorkspaceMember over members, with project's cascade,
// identity's profiles and the Authorizer as bootstrap wires them.
````

````old server/internal/bootstrap/interleaving_roles_test.go
	return workspaceapp.NewUpdateWorkspaceMember(members, workspaceProfiles{profiles: identity.Provide(r.pool).PublicProfiles},
````
````new server/internal/bootstrap/interleaving_roles_test.go
	return workspaceapp.NewUpdateWorkspaceMember(members, project.New(project.Deps{Pool: r.pool}).Cascade(),
		workspaceProfiles{profiles: identity.Provide(r.pool).PublicProfiles},
````

`server/internal/bootstrap/demotion_test.go`（新文件，108 行）：

````file server/internal/bootstrap/demotion_test.go
package bootstrap

import (
	"context"
	"net/http"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// withBobLeadingWeb has alice create the workspace slug and invite bob,
// who accepts as a member, then create its project Web with bob its lead,
// and so its admin; alice and bob are the access tokens.
func withBobLeadingWeb(t *testing.T, contract *apitest.Contract, base, alice, bob string, bobID uuid.UUID, slug string) {
	t.Helper()
	if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", alice,
		`{"name":"`+slug+`","slug":"`+slug+`"}`); status != http.StatusCreated {
		t.Fatalf("creating %s = %d %s", slug, status, body)
	}
	answerInvitation(t, contract, base, bob, "accept", invite(t, contract, base, alice, slug, "bob@example.com"), http.StatusOK)
	if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces/"+slug+"/projects", alice,
		`{"name":"Web","identifier":"WEB","project_lead_id":"`+bobID.String()+`"}`); status != http.StatusCreated {
		t.Fatalf("creating %s's project = %d %s", slug, status, body)
	}
}

// rolesOf are member's roles in each workspace and in its project, each
// marked when ended, and whether his membership of acme's project was last
// written by the account by when his membership of acme was.
func rolesOf(t *testing.T, pool *pgxpool.Pool, member, by uuid.UUID) string {
	t.Helper()
	var roles string
	if err := pool.QueryRow(context.Background(), `
SELECT string_agg(w.slug || ' ' || wm.role || CASE WHEN wm.is_active THEN '' ELSE ' ended' END
                  || ', project ' || pm.role || CASE WHEN pm.is_active THEN '' ELSE ' ended' END, '; ' ORDER BY w.slug)
       || ' | ' || bool_and(w.slug <> 'acme' OR (pm.updated_by_id = $2 AND pm.updated_at = wm.updated_at))
FROM workspace_members wm
JOIN workspaces w ON w.id = wm.workspace_id
JOIN project_members pm ON pm.workspace_id = wm.workspace_id AND pm.member_id = wm.member_id
WHERE wm.member_id = $1`, member, by).Scan(&roles); err != nil {
		t.Fatal(err)
	}
	return roles
}

// failingDemotions makes every write of a guest's project membership fail,
// the projects' step of a demotion among them, until restore runs: a CHECK
// that the rows already there need not pass.
func failingDemotions(t *testing.T, pool *pgxpool.Pool) (restore func()) {
	t.Helper()
	exec := func(sql string) {
		t.Helper()
		if _, err := pool.Exec(context.Background(), sql); err != nil {
			t.Fatal(err)
		}
	}
	exec("ALTER TABLE project_members ADD CONSTRAINT no_guests CHECK (role <> 5) NOT VALID")
	return func() { exec("ALTER TABLE project_members DROP CONSTRAINT no_guests") }
}

// A change of a member's role to guest makes him a guest in the
// workspace's projects in the same transaction (M3 design 3.3, 9.3), on the
// wired app: bob, a member of acme and of beta, leads a project in each,
// and so is its admin. While the projects' step fails, alice's change of
// his role in acme answers 500 and changes nothing; once it runs, he is
// acme's guest and a guest in acme's project, by alice at the time of his
// workspace role's change, and still beta's member and the admin of its
// project.
func TestDemotingToGuestDemotesInTheWorkspacesProjects(t *testing.T) {
	contract := apitest.Load(t)
	dbURL := pgtest.NewDatabase(t)
	base := startApp(t, testConfig(t, dbURL, false), migrations.FS())
	pool := openPool(t, dbURL)
	alice := registerAccount(t, contract, base, "alice@example.com").AccessToken
	bob := registerAccount(t, contract, base, "bob@example.com").AccessToken
	aliceID, bobID := accountID(t, contract, base, alice), accountID(t, contract, base, bob)
	withBobLeadingWeb(t, contract, base, alice, bob, bobID, "acme")
	withBobLeadingWeb(t, contract, base, alice, bob, bobID, "beta")
	var membership uuid.UUID
	if err := pool.QueryRow(context.Background(), `SELECT m.id FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id
		WHERE w.slug = 'acme' AND m.member_id = $1`, bobID).Scan(&membership); err != nil {
		t.Fatal(err)
	}
	patch := func() (int, string) {
		return call(t, contract, http.MethodPatch, base+"/api/v0/workspace-members/"+membership.String(), alice, `{"role":5}`)
	}
	before := rolesOf(t, pool, bobID, aliceID)
	if want := "acme 15, project 20; beta 15, project 20 | false"; before != want {
		t.Fatalf("bob's roles before = %s, want %s", before, want)
	}

	restore := failingDemotions(t, pool)
	status, body := patch()
	restore()
	if got := rolesOf(t, pool, bobID, aliceID); status != http.StatusInternalServerError || got != before {
		t.Errorf("the change with the projects' step failing = %d %s, roles %s; want 500 and %s", status, body, got, before)
	}

	status, body = patch()
	if got, want := rolesOf(t, pool, bobID, aliceID), "acme 5, project 5; beta 15, project 20 | true"; status != http.StatusOK || got != want {
		t.Errorf("the change = %d %s, roles %s; want 200 and %s", status, body, got, want)
	}
}
````

- [ ] **Step 6: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/project/... ./internal/modules/workspace/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestDemotingToGuestDemotesInTheWorkspacesProjects|TestTwoAdminsDemotingEachOtherLeaveAnAdmin' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 7: 提交**

```bash
git add server/internal/bootstrap/demotion_test.go server/internal/bootstrap/interleaving_roles_test.go server/internal/modules/project/adapter/postgres/cascade.go server/internal/modules/project/adapter/postgres/demote_test.go server/internal/modules/project/adapter/postgres/queries/cascade.sql server/internal/modules/project/app/cascade.go server/internal/modules/project/app/cascade_test.go server/internal/modules/project/app/ports.go server/internal/modules/project/module.go server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go server/internal/modules/workspace/app/clock_test.go server/internal/modules/workspace/app/fakes_projects_test.go server/internal/modules/workspace/app/list_members_test.go server/internal/modules/workspace/app/ports.go server/internal/modules/workspace/app/update_member.go server/internal/modules/workspace/app/update_member_test.go server/internal/modules/workspace/module.go server/internal/modules/project/adapter/postgres/gen/cascade.sql.go
```
```bash
git commit -m "feat(M3/P4a): a change of a member's role to guest makes him a guest in the workspace's projects

ProjectCascade.DemoteToGuest locks the workspace's projects in which the
account has a membership, ended ones and archived projects too, FOR NO
KEY UPDATE in id order, then makes those memberships a guest's in one
statement, by the account and at the moment of the change (M3 design
3.3, 3.6 convention 5). updateWorkspaceMember calls it after writing a
guest's role and before reading the profile, in its transaction: a
failure rolls the whole change back.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 |
|---|---|
| 锁按行来的顺序；用 `FOR SHARE`；用 `FOR UPDATE`（外键检查也要等）；不加锁 | `TestLockMemberProjectsLocksInIDOrder`；`TestDemotingAMemberToGuest`；同左；两个都失败 |
| 锁到已删除的项目；已删除的成员关系也算；每个工作区的项目；任何人的成员关系都算；任何项目的成员关系都算 | `TestDemotingAMemberToGuest`（每个工作区的另有 `TestDemotingToGuestDemotesInTheWorkspacesProjects`） |
| 写成员的角色；让已结束的恢复有效；改每个项目的、每个账户的、已删除的成员关系；重写访客的行；不写 `updated_at`；`updated_by_id` 写成员自己 | `TestDemotingAMemberToGuest`（角色、项目、时刻、账户另有 `TestDemotingToGuestDemotesInTheWorkspacesProjects`） |
| 存储以工作区的 id 当成员锁；写由成员写 | `TestDemotingAMemberToGuest` |
| 没有锁到项目也写；锁的失败被忽略；由成员写；锁了不写 | `TestDemoteToGuest`（最后一个另有 `TestDemotingToGuestDemotesInTheWorkspacesProjects`） |
| `Cascade.DemoteToGuest` 的两步在调用者的事务之外（`mutants_j.py` 的 `pf07c`） | `TestDemoteToGuest` |
| 改为访客时不调；每次改角色都调；项目一步的失败被忽略；为项目再读一次时钟；由成员写；把成员关系的 id 当账户；在改角色之前调；在读资料之后调 | `TestUpdateWorkspaceMemberLocksThenDecidesThenWrites`、`TestUpdateWorkspaceMemberFailsWithinTheTransaction`、`TestEachWriteReadsTheClockUnderItsLock`（按改法；第一、六个另有 `TestDemotingToGuestDemotesInTheWorkspacesProjects`） |
| 项目的连带不接成员关系的存储；`updateWorkspaceMember` 不接项目的连带 | `TestDemotingToGuestDemotesInTheWorkspacesProjects` |

存储把失败吞掉的变异（锁、写各一个）在真实数据库上等价：失败的语句让 Postgres 中止事务，之后的读和提交都失败，写不可能提交；错误原样返回由用例的测试（`TestDemoteToGuest`、`TestUpdateWorkspaceMemberFailsWithinTheTransaction`）和组合出的 app 的 500 核对（spec 第 3 节）。

**Done when:** 降为访客的锁、锁的顺序、写到的行和不动的行由真实数据库核对；连带的两步在调用者的事务里，`updateWorkspaceMember` 的调用位置、时刻、账户由用例测试核对；组合出的 app 上项目一步失败时整个改角色回滚。

---

### Task 13: 接受邀请恢复为访客时 `DemoteToGuest`

**Files:**
- Modify: `server/internal/bootstrap/demotion_test.go`、`server/internal/bootstrap/interleaving_answers_test.go`、`server/internal/bootstrap/invitations_test.go`、`server/internal/modules/workspace/app/accept_invitation.go`、`server/internal/modules/workspace/app/accept_invitation_test.go`、`server/internal/modules/workspace/app/clock_test.go`、`server/internal/modules/workspace/app/list_invitations_test.go`、`server/internal/modules/workspace/app/ports.go`、`server/internal/modules/workspace/module.go`

**Interfaces:**
- Produces（spec 2.12，M3 设计 3.8、9.1；P3 review 第 6 节）：`AcceptInvitationDeps.Projects ProjectCascade`；接受的 `switch` 中恢复已结束的成员关系改由 `restore(ctx, m, role, now)`：`RestoreMember` 之后，邀请的角色是访客时 `DemoteToGuest(工作区, 他, 他自己, 同一个时刻)`，在 `AcceptInvitation` 之前、同一个事务里（它让加入的上限成立，3.8）；有效成员不变、新成员没有项目可降，都不调。
- 使用者：P5 的移出、离开之后再次接受（W12）。

**Tests:**
- `workspace/app/accept_invitation_test.go`：`TestAcceptWorkspaceInvitation`（erin 已结束的成员关系以邀请的角色恢复，恢复为访客时也成为 `acme` 各项目的访客，由她自己在同一个时刻；"以前的管理员，以成员被邀请"不调；frank 新建为访客也不调；bob 的有效成员关系不变）；`TestAcceptWorkspaceInvitationRefusals` 加项目一步的失败；`TestEachWriteReadsTheClockUnderItsLock` 加"恢复为访客"的一行。
- `bootstrap/demotion_test.go`：`TestAcceptingAsAGuestAgainDemotesInTheWorkspacesProjects`（组合出的 app：bob 领导过 `acme` 的 Web，照 P5 的离开结束了两个成员关系；alice 以访客再邀请他；项目一步失败时接受答 500，什么都不变，邀请仍待接受；之后成功：他是 `acme` 的访客、Web 的访客（仍结束），由他自己在恢复的时刻写入，邀请已消费）。

- [ ] **Step 1: 接受的恢复**

`server/internal/modules/workspace/app/ports.go`（修改，1 处）：

````old server/internal/modules/workspace/app/ports.go
	// account of the change of his workspace role to guest.
````
````new server/internal/modules/workspace/app/ports.go
	// account of the change that made him the workspace's guest: an admin's
	// change of his role (updateWorkspaceMember), or his own acceptance of
	// an invitation as a guest that restores his ended membership
	// (acceptWorkspaceInvitation).
````

`server/internal/modules/workspace/app/accept_invitation.go`（修改，7 处）：

````old server/internal/modules/workspace/app/accept_invitation.go
	"context"
````
````new server/internal/modules/workspace/app/accept_invitation.go
	"context"
	"time"
````

````old server/internal/modules/workspace/app/accept_invitation.go
	Invitations InvitationAccepter
````
````new server/internal/modules/workspace/app/accept_invitation.go
	Invitations InvitationAccepter
	Projects    ProjectCascade
````

````old server/internal/modules/workspace/app/accept_invitation.go
	invitations InvitationAccepter
````
````new server/internal/modules/workspace/app/accept_invitation.go
	invitations InvitationAccepter
	projects    ProjectCascade
````

````old server/internal/modules/workspace/app/accept_invitation.go
	return &AcceptWorkspaceInvitation{invitations: d.Invitations, tx: d.Tx, clock: d.Clock,
````
````new server/internal/modules/workspace/app/accept_invitation.go
	return &AcceptWorkspaceInvitation{invitations: d.Invitations, projects: d.Projects, tx: d.Tx, clock: d.Clock,
````

````old server/internal/modules/workspace/app/accept_invitation.go
// former member's row is restored with the invitation's role; P4 adds, when
// that role is a guest's, DemoteToGuest of his project memberships here, in
// the same transaction. Anyone else is inserted with it. Then the
// invitation is accepted, and deleted. The answer is the workspace with the
// caller's role as it now is.
````
````new server/internal/modules/workspace/app/accept_invitation.go
// former member's row is restored with the invitation's role (restore).
// Anyone else is inserted with it. Then the invitation is accepted, and
// deleted. The answer is the workspace with the caller's role as it now
// is.
````

````old server/internal/modules/workspace/app/accept_invitation.go
			err = u.invitations.RestoreMember(ctx, m.ID, inv.Role, account.ID, now)
````
````new server/internal/modules/workspace/app/accept_invitation.go
			err = u.restore(ctx, m, inv.Role, now)
````

````old server/internal/modules/workspace/app/accept_invitation.go
	return joined, nil
}

````
````new server/internal/modules/workspace/app/accept_invitation.go
	return joined, nil
}

// restore makes the ended membership m active again with role, by its
// member at now. Restored as a guest, he is made a guest in each of the
// workspace's projects he has a membership of, ended ones too
// (ProjectCascade.DemoteToGuest, M3 design 3.8), in the same transaction:
// joining a project again restores his membership there at no more than
// its role before, so without it a former guest made a member later would
// get back the project roles he had before he was a guest.
func (u *AcceptWorkspaceInvitation) restore(ctx context.Context, m domain.Membership, role shared.Role, now time.Time) error {
	if err := u.invitations.RestoreMember(ctx, m.ID, role, m.MemberID, now); err != nil {
		return err
	}
	if role != shared.RoleGuest {
		return nil
	}
	return u.projects.DemoteToGuest(ctx, m.WorkspaceID, m.MemberID, m.MemberID, now)
}

````

`server/internal/modules/workspace/module.go`（修改，1 处）：

````old server/internal/modules/workspace/module.go
			Accounts: d.Accounts, Invitations: store, Tx: d.Tx, Clock: d.Clock, MAC: d.InvitationMAC,
````
````new server/internal/modules/workspace/module.go
			Accounts: d.Accounts, Invitations: store, Projects: d.Projects, Tx: d.Tx, Clock: d.Clock, MAC: d.InvitationMAC,
````

- [ ] **Step 2: 用例的测试**

`server/internal/modules/workspace/app/list_invitations_test.go`（修改，1 处）：

````old server/internal/modules/workspace/app/list_invitations_test.go
	accounts    *fakeAccounts // responding's
````
````new server/internal/modules/workspace/app/list_invitations_test.go
	accounts    *fakeAccounts // responding's
	projects    *fakeProjects // responding's
````

`server/internal/modules/workspace/app/accept_invitation_test.go`（修改，12 处）：

````old server/internal/modules/workspace/app/accept_invitation_test.go
// of acme, and the accounts: alice, bob, carol (deactivated), dave, erin
// and frank.
````
````new server/internal/modules/workspace/app/accept_invitation_test.go
// of acme, the accounts: alice, bob, carol (deactivated), dave, erin and
// frank, and the projects' cascade.
````

````old server/internal/modules/workspace/app/accept_invitation_test.go
	f.accounts = &fakeAccounts{log: f.log, accounts: []app.AccountState{alice, bob, carol, dave, erin, frank}}
````
````new server/internal/modules/workspace/app/accept_invitation_test.go
	f.accounts = &fakeAccounts{log: f.log, accounts: []app.AccountState{alice, bob, carol, dave, erin, frank}}
	f.projects = &fakeProjects{log: f.log}
````

````old server/internal/modules/workspace/app/accept_invitation_test.go
	return app.NewAcceptWorkspaceInvitation(app.AcceptInvitationDeps{Accounts: f.accounts, Invitations: f.invitations, Tx: f.tx,
		Clock: clockAt{at: clockNow}, MAC: f.mac})
````
````new server/internal/modules/workspace/app/accept_invitation_test.go
	return app.NewAcceptWorkspaceInvitation(app.AcceptInvitationDeps{Accounts: f.accounts, Invitations: f.invitations, Projects: f.projects,
		Tx: f.tx, Clock: clockAt{at: clockNow}, MAC: f.mac})
````

````old server/internal/modules/workspace/app/accept_invitation_test.go
// ended membership is restored with the invitation's role; frank is
// inserted with it. Each in one transaction, after the account's, the
// workspace's and the invitation's locks; the answer is the workspace with
// the caller's role as it now is, and the invitation is gone.
func TestAcceptWorkspaceInvitation(t *testing.T) {
	at := clockNow.Format(time.RFC3339Nano)
````
````new server/internal/modules/workspace/app/accept_invitation_test.go
// ended membership is restored with the invitation's role, and restored as
// a guest's she is made a guest in acme's projects too, by herself at the
// same time; frank is inserted with it, a guest's too without a project to
// demote in. Each in one transaction, after the account's, the workspace's
// and the invitation's locks; the answer is the workspace with the
// caller's role as it now is, and the invitation is gone.
func TestAcceptWorkspaceInvitation(t *testing.T) {
	at := clockNow.Format(time.RFC3339Nano)
	restored := func(role shared.Role) string {
		return fmt.Sprintf("RestoreMember %s as %d by %s at %s", erinInAcme.ID, role, erin.ID, at)
	}
	created := func(user app.AccountState, w domain.Workspace, role shared.Role) string {
		return fmt.Sprintf("CreateMember %s in %s as %d by %s at %s", user.ID, w.ID, role, user.ID, at)
	}
````

````old server/internal/modules/workspace/app/accept_invitation_test.go
		name  string
		user  app.AccountState
		inv   domain.Invitation
		w     domain.Workspace
		role  shared.Role // the answer's
		write string      // the membership's, "" for none
````
````new server/internal/modules/workspace/app/accept_invitation_test.go
		name   string
		user   app.AccountState
		inv    domain.Invitation
		w      domain.Workspace
		role   shared.Role // the answer's
		writes []string    // the membership's and the projects'
````

````old server/internal/modules/workspace/app/accept_invitation_test.go
		{"an active member, invited higher", bob, invitationTo(bob, acme, shared.RoleAdmin), acme, shared.RoleMember, ""},
		{"an active member, invited lower", bob, invitationTo(bob, acme, shared.RoleGuest), acme, shared.RoleMember, ""},
		{"an active member, invited the same", bob, invitationTo(bob, acme, shared.RoleMember), acme, shared.RoleMember, ""},
		{"an active guest of another workspace, invited higher", bob, invitationTo(bob, beta, shared.RoleAdmin), beta, shared.RoleGuest, ""},
````
````new server/internal/modules/workspace/app/accept_invitation_test.go
		{"an active member, invited higher", bob, invitationTo(bob, acme, shared.RoleAdmin), acme, shared.RoleMember, nil},
		{"an active member, invited lower", bob, invitationTo(bob, acme, shared.RoleGuest), acme, shared.RoleMember, nil},
		{"an active member, invited the same", bob, invitationTo(bob, acme, shared.RoleMember), acme, shared.RoleMember, nil},
		{"an active guest of another workspace, invited higher", bob, invitationTo(bob, beta, shared.RoleAdmin), beta, shared.RoleGuest, nil},
````

````old server/internal/modules/workspace/app/accept_invitation_test.go
			fmt.Sprintf("RestoreMember %s as %d by %s at %s", erinInAcme.ID, shared.RoleGuest, erin.ID, at)},
````
````new server/internal/modules/workspace/app/accept_invitation_test.go
			[]string{restored(shared.RoleGuest), fmt.Sprintf("DemoteToGuest %s %s by %s at %s", acme.ID, erin.ID, erin.ID, at)}},
		{"a former admin, invited as a member", erin, invitationTo(erin, acme, shared.RoleMember), acme, shared.RoleMember,
			[]string{restored(shared.RoleMember)}},
````

````old server/internal/modules/workspace/app/accept_invitation_test.go
			fmt.Sprintf("CreateMember %s in %s as %d by %s at %s", erin.ID, beta.ID, shared.RoleMember, erin.ID, at)},
````
````new server/internal/modules/workspace/app/accept_invitation_test.go
			[]string{created(erin, beta, shared.RoleMember)}},
````

````old server/internal/modules/workspace/app/accept_invitation_test.go
			fmt.Sprintf("CreateMember %s in %s as %d by %s at %s", frank.ID, acme.ID, shared.RoleAdmin, frank.ID, at)},
````
````new server/internal/modules/workspace/app/accept_invitation_test.go
			[]string{created(frank, acme, shared.RoleAdmin)}},
		{"never a member, invited as a guest", frank, invitationTo(frank, acme, shared.RoleGuest), acme, shared.RoleGuest,
			[]string{created(frank, acme, shared.RoleGuest)}},
````

````old server/internal/modules/workspace/app/accept_invitation_test.go
		wantCalls := append(respondedCalls(tt.user, tt.inv, "LockWorkspace"), "MemberOf "+tt.w.ID.String()+" "+tt.user.ID.String())
		if tt.write != "" {
			wantCalls = append(wantCalls, tt.write)
		}
		wantCalls = append(wantCalls, fmt.Sprintf("AcceptInvitation %s by %s at %s", tt.inv.ID, tt.user.ID, at), "WorkspaceByID "+tt.w.ID.String())
````
````new server/internal/modules/workspace/app/accept_invitation_test.go
		wantCalls := slices.Concat(respondedCalls(tt.user, tt.inv, "LockWorkspace"), []string{"MemberOf " + tt.w.ID.String() + " " + tt.user.ID.String()},
			tt.writes, []string{fmt.Sprintf("AcceptInvitation %s by %s at %s", tt.inv.ID, tt.user.ID, at), "WorkspaceByID " + tt.w.ID.String()})
````

````old server/internal/modules/workspace/app/accept_invitation_test.go
// restore among them, of the answer's read or of the commit is itself too,
// and no workspace is answered. Without a caller it is 401 and nothing is
````
````new server/internal/modules/workspace/app/accept_invitation_test.go
// restore and her projects' step among them, of the answer's read or of the
// commit is itself too, and no workspace is answered. Without a caller it is 401 and nothing is
````

````old server/internal/modules/workspace/app/accept_invitation_test.go
			fmt.Sprintf("RestoreMember %s as %d by %s at %s", erinInAcme.ID, shared.RoleGuest, erin.ID, at))},
````
````new server/internal/modules/workspace/app/accept_invitation_test.go
			fmt.Sprintf("RestoreMember %s as %d by %s at %s", erinInAcme.ID, shared.RoleGuest, erin.ID, at))},
		responseRefusal{"DemoteToGuest failed", erin, erinToAcme.ID, "", func(f *invitationsFixture) {
			f.invitations.invitations = append(f.invitations.invitations, erinToAcme)
			f.projects.errs = map[string]error{"DemoteToGuest": failure}
		}, failure, append(respondedCalls(erin, erinToAcme, "LockWorkspace"), "MemberOf "+acme.ID.String()+" "+erin.ID.String(),
			fmt.Sprintf("RestoreMember %s as %d by %s at %s", erinInAcme.ID, shared.RoleGuest, erin.ID, at),
			fmt.Sprintf("DemoteToGuest %s %s by %s at %s", acme.ID, erin.ID, erin.ID, at))},
````

`server/internal/modules/workspace/app/clock_test.go`（修改，3 处）：

````old server/internal/modules/workspace/app/clock_test.go
// every step, and a change to guest for the projects' step. The clock logs
// its read among the fakes' calls.
````
````new server/internal/modules/workspace/app/clock_test.go
// every step, and a change to guest or a restoring as a guest for the
// projects' step. The clock logs its read among the fakes' calls.
````

````old server/internal/modules/workspace/app/clock_test.go
	at := clockNow.Format(time.RFC3339Nano)
````
````new server/internal/modules/workspace/app/clock_test.go
	at := clockNow.Format(time.RFC3339Nano)
	erinToAcme := invitationTo(erin, acme, shared.RoleGuest)
````

````old server/internal/modules/workspace/app/clock_test.go
			fmt.Sprintf("AcceptInvitation %s by %s at %s", frankToAcme.ID, frank.ID, at), "WorkspaceByID "+acme.ID.String())},
````
````new server/internal/modules/workspace/app/clock_test.go
			fmt.Sprintf("AcceptInvitation %s by %s at %s", frankToAcme.ID, frank.ID, at), "WorkspaceByID "+acme.ID.String())},
		{"acceptWorkspaceInvitation, restoring a guest", func() ([]string, error) {
			f := responding(erinToAcme)
			_, err := app.NewAcceptWorkspaceInvitation(app.AcceptInvitationDeps{Accounts: f.accounts, Invitations: f.invitations,
				Projects: f.projects, Tx: f.tx, Clock: clockAt{clockNow, f.log}, MAC: f.mac}).
				Execute(as(erin), erinToAcme.ID, tokenOf(f.mac, erinToAcme.ID))
			return f.log.calls, err
		}, append(respondedCalls(erin, erinToAcme, "LockWorkspace"), "Now", "MemberOf "+acme.ID.String()+" "+erin.ID.String(),
			fmt.Sprintf("RestoreMember %s as %d by %s at %s", erinInAcme.ID, shared.RoleGuest, erin.ID, at),
			fmt.Sprintf("DemoteToGuest %s %s by %s at %s", acme.ID, erin.ID, erin.ID, at),
			fmt.Sprintf("AcceptInvitation %s by %s at %s", erinToAcme.ID, erin.ID, at), "WorkspaceByID "+acme.ID.String())},
````

- [ ] **Step 3: 组合出的测试**

`server/internal/bootstrap/invitations_test.go`（修改，2 处）：

````old server/internal/bootstrap/invitations_test.go
func invite(t *testing.T, contract *apitest.Contract, base, admin, slug, email string) invitationLink {
	t.Helper()
````
````new server/internal/bootstrap/invitations_test.go
func invite(t *testing.T, contract *apitest.Contract, base, admin, slug, email string) invitationLink {
	t.Helper()
	return inviteAs(t, contract, base, admin, slug, email, shared.RoleMember)
}

// inviteAs is invite with the invitation's role.
func inviteAs(t *testing.T, contract *apitest.Contract, base, admin, slug, email string, role shared.Role) invitationLink {
	t.Helper()
````

````old server/internal/bootstrap/invitations_test.go
		`{"invitations":[{"email":"`+email+`","role":15}]}`)
````
````new server/internal/bootstrap/invitations_test.go
		fmt.Sprintf(`{"invitations":[{"email":%q,"role":%d}]}`, email, role))
````

`server/internal/bootstrap/interleaving_answers_test.go`（修改，3 处）：

````old server/internal/bootstrap/interleaving_answers_test.go
}

// accept is bob's acceptance of his invitation, over invitations.
````
````new server/internal/bootstrap/interleaving_answers_test.go
}

// projects is project's cascade as bootstrap wires it.
func (r answerRace) projects() workspaceapp.ProjectCascade {
	return project.New(project.Deps{Pool: r.pool}).Cascade()
}

// accept is bob's acceptance of his invitation, over invitations.
````

````old server/internal/bootstrap/interleaving_answers_test.go
		Accounts: r.accounts(), Invitations: invitations, Tx: r.tx(), Clock: clocktest.At(time.Now()), MAC: r.mac,
````
````new server/internal/bootstrap/interleaving_answers_test.go
		Accounts: r.accounts(), Invitations: invitations, Projects: r.projects(), Tx: r.tx(), Clock: clocktest.At(time.Now()), MAC: r.mac,
````

````old server/internal/bootstrap/interleaving_answers_test.go
	return workspaceapp.NewDeleteWorkspace(workspaces, project.New(project.Deps{Pool: r.pool}).Cascade(),
````
````new server/internal/bootstrap/interleaving_answers_test.go
	return workspaceapp.NewDeleteWorkspace(workspaces, r.projects(),
````

`server/internal/bootstrap/demotion_test.go`（修改，2 处）：

````old server/internal/bootstrap/demotion_test.go
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
````
````new server/internal/bootstrap/demotion_test.go
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
````

````old server/internal/bootstrap/demotion_test.go
		t.Errorf("the change = %d %s, roles %s; want 200 and %s", status, body, got, want)
	}
}

````
````new server/internal/bootstrap/demotion_test.go
		t.Errorf("the change = %d %s, roles %s; want 200 and %s", status, body, got, want)
	}
}

// Accepting an invitation as a guest that restores an ended membership
// makes its member a guest in the workspace's projects in the same
// transaction (M3 design 3.8, 9.3), on the wired app: bob led acme's
// project Web, so was its admin, and has left acme and Web, both his
// memberships ended as P5's leave ends them. alice invites him again, as a
// guest. While the projects' step fails, his acceptance answers 500 and
// changes nothing, the invitation still pending; once it runs, he is
// acme's guest again and a guest in Web, still ended there, by himself at
// the time of the restoring, and the invitation is consumed.
func TestAcceptingAsAGuestAgainDemotesInTheWorkspacesProjects(t *testing.T) {
	contract := apitest.Load(t)
	dbURL := pgtest.NewDatabase(t)
	base := startApp(t, testConfig(t, dbURL, false), migrations.FS())
	pool := openPool(t, dbURL)
	alice := registerAccount(t, contract, base, "alice@example.com").AccessToken
	bob := registerAccount(t, contract, base, "bob@example.com").AccessToken
	bobID := accountID(t, contract, base, bob)
	withBobLeadingWeb(t, contract, base, alice, bob, bobID, "acme")
	for _, table := range []string{"workspace_members", "project_members"} {
		if _, err := pool.Exec(context.Background(), "UPDATE "+table+" SET is_active = false WHERE member_id = $1", bobID); err != nil {
			t.Fatal(err)
		}
	}
	link := inviteAs(t, contract, base, alice, "acme", "bob@example.com", shared.RoleGuest)
	accept := func() (int, string) {
		return call(t, contract, http.MethodPost, base+"/api/v0/workspace-invitations/"+link.id.String()+"/accept", bob,
			`{"token":"`+link.token+`"}`)
	}
	pending := func() bool {
		t.Helper()
		var pending bool
		if err := pool.QueryRow(context.Background(),
			"SELECT EXISTS (SELECT 1 FROM workspace_member_invites WHERE id = $1 AND deleted_at IS NULL AND NOT accepted)", link.id).
			Scan(&pending); err != nil {
			t.Fatal(err)
		}
		return pending
	}
	before := rolesOf(t, pool, bobID, bobID)
	if want := "acme 15 ended, project 20 ended | false"; before != want {
		t.Fatalf("bob's roles before = %s, want %s", before, want)
	}

	restore := failingDemotions(t, pool)
	status, body := accept()
	restore()
	if got := rolesOf(t, pool, bobID, bobID); status != http.StatusInternalServerError || got != before || !pending() {
		t.Errorf("the acceptance with the projects' step failing = %d %s, roles %s; want 500, %s and the invitation pending", status, body, got,
			before)
	}

	status, body = accept()
	if got, want := rolesOf(t, pool, bobID, bobID), "acme 5, project 5 ended | true"; status != http.StatusOK || got != want || pending() {
		t.Errorf("the acceptance = %d %s, roles %s; want 200, %s and the invitation consumed", status, body, got, want)
	}
}

````

- [ ] **Step 4: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/workspace/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestAcceptingAsAGuestAgainDemotesInTheWorkspacesProjects|TestDemotingToGuestDemotesInTheWorkspacesProjects|TestAcceptingAndDeletingTheWorkspace|TestAcceptingAndChangingTheAddress' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 5: 提交**

```bash
git add server/internal/bootstrap/demotion_test.go server/internal/bootstrap/interleaving_answers_test.go server/internal/bootstrap/invitations_test.go server/internal/modules/workspace/app/accept_invitation.go server/internal/modules/workspace/app/accept_invitation_test.go server/internal/modules/workspace/app/clock_test.go server/internal/modules/workspace/app/list_invitations_test.go server/internal/modules/workspace/app/ports.go server/internal/modules/workspace/module.go
```
```bash
git commit -m "feat(M3/P4a): accepting an invitation as a guest again makes the account a guest in the projects

When acceptWorkspaceInvitation restores an ended membership with a
guest's role, it calls ProjectCascade.DemoteToGuest after RestoreMember
and before AcceptInvitation, in the same transaction, by the accepting
account at the moment of the restoring (M3 design 3.8): no restored
guest keeps a higher role in a project he may rejoin. An active member
and a new one have nothing to demote.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 |
|---|---|
| 恢复为访客时不调 | `TestAcceptWorkspaceInvitation`、`TestAcceptWorkspaceInvitationRefusals`、`TestEachWriteReadsTheClockUnderItsLock`、`TestAcceptingAsAGuestAgainDemotesInTheWorkspacesProjects` |
| 每次恢复都调；新成员也调；有效成员以访客被邀请也调；在恢复之前调；在接受邀请之后调 | `TestAcceptWorkspaceInvitation`（第四、五个另有 `TestEachWriteReadsTheClockUnderItsLock`） |
| 项目一步的失败被忽略 | `TestAcceptWorkspaceInvitationRefusals` |
| 以成员关系的 id 当账户；为项目再读一次时钟 | `TestAcceptWorkspaceInvitation`、`TestEachWriteReadsTheClockUnderItsLock`（前者另有 `TestAcceptingAsAGuestAgainDemotesInTheWorkspacesProjects`）；`TestEachWriteReadsTheClockUnderItsLock` |
| 接受不接项目的连带 | `TestAcceptingAsAGuestAgainDemotesInTheWorkspacesProjects` |

**Done when:** 接受的三种情形只在恢复为访客时降级，位置、时刻、账户由用例测试核对；组合出的 app 上项目一步失败时整个接受回滚、邀请仍待接受。

---

### Task 14: 端到端：P1 的接口版本；W3 加上项目的连带

**Files:**
- Create: `e2e/fixtures/assert/project.ts`、`e2e/stories/project/p1-create-project.spec.ts`
- Modify: `e2e/fixtures/api.ts`、`e2e/fixtures/assert/workspace.ts`、`e2e/fixtures/auth.ts`、`e2e/stories/workspace/w3-workspace-settings.spec.ts`

**Interfaces:**
- Produces（spec 2.15，M3 设计 2、9.6；裁定 S4）：`api.ts` 的 `Project`、`ProjectCreate` 类型（取自生成的客户端）和 `createProject(api, token, slug, body)`；`auth.ts` 的 `accountId(api, token)`（`GET /api/v0/me`）；`assert/project.ts` 的 `NewProject`、`NewMember`、`expectProjectCreated(db, slug, p, creatorEmail, leadEmail, members)`（项目、成员关系和显示设置、六个状态，每一行都由创建者在创建的时刻写入、之后没有变）、`countProjects`；`assert/workspace.ts` 的 `workspaceTables` 加上项目的四张表（标签由 P7 加入）。
- 使用者：P4b 的 P2–P4、P8 的接口版本；P10 的 P1 的页面版本。

**Tests:**
- **P1 (API)**：成员以管理员为负责人建 Web（标识小写、带图标）：答案、四张表的行；标识的检查（之前可用、`WE-B` 不可用；之后 `Web` 不可用、`ops` 可用，陌生人自己工作区的 `web` 可用）；七种拒绝一起发出、每一种都不写（标识、名称被占，名称含 `-`、`.`，访客，负责人是访客、是陌生人：他是自己工作区的管理员也不行）；之后管理员建私密的 Ops（他侧边栏的第一位 55535），成员以管理员为负责人建 Docs（成员 55535、管理员 45535），各按自己的位置；最后读以前的项目（裁定第 25 条）：管理员读 Ops（私密，他侧边栏中间的 55535）和 Docs（他是负责人不是创建者，45535），成员读 Ops 得 404 `project.not_found`。
- **W3 (API)**：两个工作区各由管理员以成员为负责人建 Web，每个答案是这个工作区自己的项目（`workspace_id`、角色、位置、成员）；删除 `acme` 之后它的项目、项目成员、显示设置、状态与工作区在同一时刻删除（`expectWorkspaceDeleted` 从 `workspaceTables` 取表），`other` 的项目照 `expectProjectCreated` 不变。

- [ ] **Step 1: fixture**

`e2e/fixtures/api.ts`（修改，2 处）：

````old e2e/fixtures/api.ts
export type InvitationCreate = components["schemas"]["InvitationCreate"];
````
````new e2e/fixtures/api.ts
export type InvitationCreate = components["schemas"]["InvitationCreate"];
export type Project = components["schemas"]["Project"];
export type ProjectCreate = components["schemas"]["ProjectCreate"];
````

````old e2e/fixtures/api.ts
  return accept(api, member.token, invitation);
}

````
````new e2e/fixtures/api.ts
  return accept(api, member.token, invitation);
}

/** Creates a project in the workspace of slug with the bearer token given, an admin's or a member's, and returns it. */
export async function createProject(api: Api, token: string, slug: string, body: ProjectCreate): Promise<Project> {
  const { data, error, response } = await api.POST("/api/v0/workspaces/{slug}/projects", {
    params: { path: { slug } },
    body,
    headers: bearer(token),
  });
  expect(response.status, `create the project ${body.identifier} in ${slug}: ${JSON.stringify(error)}`).toBe(201);
  if (!data) {
    throw new Error(`createProject ${body.identifier} answered 201 without the project`);
  }
  return data;
}

````

`e2e/fixtures/auth.ts`（修改，1 处）：

````old e2e/fixtures/auth.ts
}

/** Creates a personal access token with the bearer token given, and returns it with its token. */
````
````new e2e/fixtures/auth.ts
}

/** The id of the account of the bearer token given. */
export async function accountId(api: Api, token: string): Promise<string> {
  const { data, error, response } = await api.GET("/api/v0/me", { headers: bearer(token) });
  expect(response.status, `read the account: ${JSON.stringify(error)}`).toBe(200);
  if (!data) {
    throw new Error("GET /api/v0/me answered 200 without the account");
  }
  return data.id;
}

/** Creates a personal access token with the bearer token given, and returns it with its token. */
````

`e2e/fixtures/assert/project.ts`（新文件，131 行）：

````file e2e/fixtures/assert/project.ts
import { expect } from "@playwright/test";

import type { Database } from "../db";

// Database assertions of the project stories. The page version and the
// API version of a story call the same function (M3 design 2).

/**
 * What a project holds once created: what its creator gave, the identifier
 * upper-cased, the workspace's time zone when he gave none.
 */
export interface NewProject {
  name: string;
  identifier: string;
  description: string;
  network: number;
  timezone: string;
  logo_props: Record<string, unknown>;
}

/** A member a new project has, an active admin, and his place in his sidebar (M3 design 3.18). */
export interface NewMember {
  email: string;
  sort_order: number;
}

/**
 * The six states of a new project, in their order, as expectProjectCreated reads them: Plane's DEFAULT_STATES,
 * Backlog the default (M3 design 3.17), each written with the project.
 */
const newStates = [
  { name: "Backlog", color: "#60646C", sequence: 15000, group: "backlog", default: true, with_the_project: true },
  { name: "Todo", color: "#60646C", sequence: 25000, group: "unstarted", default: false, with_the_project: true },
  { name: "In Progress", color: "#F59E0B", sequence: 35000, group: "started", default: false, with_the_project: true },
  { name: "Done", color: "#46A758", sequence: 45000, group: "completed", default: false, with_the_project: true },
  { name: "Cancelled", color: "#9AA4BC", sequence: 55000, group: "cancelled", default: false, with_the_project: true },
  { name: "Triage", color: "#4E5355", sequence: 65000, group: "triage", default: false, with_the_project: true },
];

/**
 * P1, W3: the project of p.identifier in the workspace of slug holds p, is neither archived nor deleted, has no
 * work item numbered yet (last_issue_sequence 0) and is led by the account of leadEmail, or by no one when it is
 * null. Its members are members, each an active admin (role 20) with his display settings in it at his place; its
 * states are the six of a new project. The account of creatorEmail wrote every row, at the project's creation, and
 * none has changed since. Returns the project's id.
 */
export async function expectProjectCreated(
  db: Database,
  slug: string,
  p: NewProject,
  creatorEmail: string,
  leadEmail: string | null,
  members: NewMember[]
): Promise<string> {
  const [project] = await db.query<{ id: string }>(
    `SELECT p.id, p.name, p.identifier, p.description, p.network, p.timezone, p.logo_props, p.last_issue_sequence,
            p.archived_at, p.deleted_at, l.email AS lead, c.email AS creator,
            p.updated_by_id = p.created_by_id AND p.updated_at = p.created_at AS unchanged
       FROM projects p
       JOIN workspaces w ON w.id = p.workspace_id
       JOIN users c ON c.id = p.created_by_id
       LEFT JOIN users l ON l.id = p.project_lead_id
      WHERE w.slug = $1 AND p.identifier = $2`,
    [slug, p.identifier]
  );
  expect(project, `the project ${p.identifier} of ${slug}`).toEqual({
    ...p,
    id: expect.any(String),
    last_issue_sequence: 0,
    archived_at: null,
    deleted_at: null,
    lead: leadEmail,
    creator: creatorEmail,
    unchanged: true,
  });
  const id = project?.id as string;
  // The database compares the moments: a Date holds milliseconds, a timestamptz microseconds.
  expect(
    await db.query(
      `SELECT u.email, m.role, m.is_active, s.sort_order,
              m.created_by_id = p.created_by_id AND m.updated_by_id = p.created_by_id AND m.created_at = p.created_at
                AND m.updated_at = p.created_at AND m.deleted_at IS NULL
              AND s.created_by_id = p.created_by_id AND s.updated_by_id = p.created_by_id AND s.created_at = p.created_at
                AND s.updated_at = p.created_at AND s.deleted_at IS NULL AS with_the_project
         FROM project_members m
         JOIN projects p ON p.id = m.project_id
         JOIN users u ON u.id = m.member_id
         LEFT JOIN project_user_properties s ON s.project_id = m.project_id AND s.user_id = m.member_id
        WHERE m.project_id = $1 ORDER BY u.email COLLATE "C"`,
      [id]
    ),
    `the members of ${p.identifier} and their display settings`
  ).toEqual(
    members
      .toSorted((a, b) => (a.email < b.email ? -1 : 1))
      .map((m) => ({ email: m.email, role: 20, is_active: true, sort_order: m.sort_order, with_the_project: true }))
  );
  expect(
    await db.query(
      `SELECT s.name, s.color, s.sequence, s."group", s."default",
              s.created_by_id = p.created_by_id AND s.updated_by_id = p.created_by_id AND s.created_at = p.created_at
                AND s.updated_at = p.created_at AND s.deleted_at IS NULL AS with_the_project
         FROM states s JOIN projects p ON p.id = s.project_id
        WHERE s.project_id = $1 ORDER BY s.sequence`,
      [id]
    ),
    `the states of ${p.identifier}`
  ).toEqual(newStates);
  return id;
}

/** How many rows the project tables have. */
export interface ProjectCounts {
  projects: number;
  members: number;
  preferences: number;
  states: number;
}

export async function countProjects(db: Database): Promise<ProjectCounts> {
  const [counts] = await db.query<ProjectCounts>(
    `SELECT (SELECT count(*)::int FROM projects) AS projects,
            (SELECT count(*)::int FROM project_members) AS members,
            (SELECT count(*)::int FROM project_user_properties) AS preferences,
            (SELECT count(*)::int FROM states) AS states`
  );
  if (!counts) {
    throw new Error("the counts query returned no row");
  }
  return counts;
}
````

`e2e/fixtures/assert/workspace.ts`（修改，2 处）：

````old e2e/fixtures/assert/workspace.ts
/** The tables whose rows belong to a workspace and are deleted with it (M3 design 4.12); P4 adds the projects'. */
const workspaceTables = ["workspace_members", "workspace_member_invites", "workspace_user_properties"];
````
````new e2e/fixtures/assert/workspace.ts
/** The tables whose rows belong to a workspace and are deleted with it (M3 design 3.6, 4.12); P7 adds the labels. */
const workspaceTables = [
  "workspace_members",
  "workspace_member_invites",
  "workspace_user_properties",
  "projects",
  "project_members",
  "project_user_properties",
  "states",
];
````

````old e2e/fixtures/assert/workspace.ts
 * settings. Each table has such a row; none is left undeleted.
````
````new e2e/fixtures/assert/workspace.ts
 * settings, its projects, their memberships, their members' display settings and their states. Each table has
 * such a row; none is left undeleted.
````

- [ ] **Step 2: 故事**

`e2e/stories/project/p1-create-project.spec.ts`（新文件，189 行）：

````file e2e/stories/project/p1-create-project.spec.ts
import {
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
  type ProjectCreate,
} from "../../fixtures/api";
import { countProjects, expectProjectCreated } from "../../fixtures/assert/project";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// P1, create a project (M3 design 2, 3.17, 3.18). The page version comes
// with the projects' pages (P10).

/** The answer of GET /api/v0/workspaces/{slug}/project-identifiers/{identifier}. */
async function availability(api: Api, token: string, slug: string, identifier: string): Promise<unknown> {
  const { data, error, response } = await api.GET("/api/v0/workspaces/{slug}/project-identifiers/{identifier}", {
    params: { path: { slug, identifier } },
    headers: bearer(token),
  });
  expect(response.status, `check ${identifier}: ${JSON.stringify(error)}`).toBe(200);
  return data;
}

/** The answer of GET /api/v0/projects/{project_id}: its status, and the project or the problem's code. */
async function read(
  api: Api,
  token: string,
  id: string
): Promise<{ status: number; project?: unknown; code?: string }> {
  const { data, error, response } = await api.GET("/api/v0/projects/{project_id}", {
    params: { path: { project_id: id } },
    headers: bearer(token),
  });
  return data ? { status: response.status, project: data } : { status: response.status, code: error?.code };
}

/** The answer to a createProject whose field the rules do not allow. */
function notAllowed(field: string) {
  return { status: 422, code: "validation_failed", errors: [{ field, code: "not_allowed" }] };
}

test("P1 (API): a member creates a project with the admin its lead, both its admins, first in their sidebars, with its six states; a taken identifier or name, a name with a forbidden character, a guest, or a lead who is a guest or no member adds nothing", async ({
  api,
  db,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
  const memberEmail = emailFor(testInfo, "member");
  const member = (await createPAT(api, (await register(api, memberEmail)).access_token)).token;
  const guestEmail = emailFor(testInfo, "guest");
  const guest = (await createPAT(api, (await register(api, guestEmail)).access_token)).token;
  const stranger = (await register(api, emailFor(testInfo, "stranger"))).access_token;
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin, { name: "Acme", slug, timezone: "Asia/Shanghai" });
  // The stranger admins a workspace of his own: his role there makes him no lead of Acme's, and each workspace
  // answers for its own identifiers.
  const strangers = slugFor(testInfo, "stranger");
  await createWorkspace(api, stranger, { name: "Stranger", slug: strangers });
  await inviteAndAccept(api, admin, slug, { email: memberEmail, token: member }, 15);
  await inviteAndAccept(api, admin, slug, { email: guestEmail, token: guest }, 5);
  const [adminId, memberId, guestId, strangerId] = await Promise.all(
    [admin, member, guest, stranger].map((token) => accountId(api, token))
  );
  expect(await availability(api, member, slug, "web")).toEqual({ available: true });
  expect(await availability(api, member, slug, "WE-B")).toEqual({ available: false });

  const logo = { in_use: "emoji", emoji: { value: "128640" } } as const;
  const created = await createProject(api, member, slug, {
    name: "Web",
    identifier: "web",
    description: "The site",
    network: 2,
    project_lead_id: adminId,
    logo_props: logo,
  });

  const web = {
    name: "Web",
    identifier: "WEB",
    description: "The site",
    network: 2,
    timezone: "Asia/Shanghai",
    logo_props: logo,
  };
  expect(created).toMatchObject({
    ...web,
    project_lead_id: adminId,
    archived_at: null,
    member_role: 20,
    sort_order: 65535,
    member_ids: [memberId, adminId],
  });
  const members = [
    { email: memberEmail, sort_order: 65535 },
    { email: adminEmail, sort_order: 65535 },
  ];
  expect(await expectProjectCreated(db, slug, web, memberEmail, adminEmail, members)).toBe(created.id);
  expect(await availability(api, admin, slug, "Web")).toEqual({ available: false });
  expect(await availability(api, member, slug, "ops")).toEqual({ available: true });
  expect(await availability(api, stranger, strangers, "web")).toEqual({ available: true });

  const before = await countProjects(db);
  // Each refused, all at once: none writes.
  const refusals: { token: string; body: ProjectCreate; want: { status: number; code: string } }[] = [
    {
      token: member,
      body: { name: "Web 2", identifier: "Web" },
      want: { status: 409, code: "project.identifier_taken" },
    },
    { token: member, body: { name: "Web", identifier: "WEB2" }, want: { status: 409, code: "project.name_taken" } },
    { token: member, body: { name: "Web-2", identifier: "WEB2" }, want: notAllowed("name") },
    { token: member, body: { name: "Web.2", identifier: "WEB2" }, want: notAllowed("name") },
    { token: guest, body: { name: "Mine", identifier: "MINE" }, want: { status: 403, code: "forbidden" } },
    {
      token: member,
      body: { name: "Ops", identifier: "OPS", project_lead_id: guestId },
      want: notAllowed("project_lead_id"),
    },
    {
      token: member,
      body: { name: "Ops", identifier: "OPS", project_lead_id: strangerId },
      want: notAllowed("project_lead_id"),
    },
  ];
  const answers = await Promise.all(
    refusals.map(async ({ token, body }) => {
      const { error, response } = await api.POST("/api/v0/workspaces/{slug}/projects", {
        params: { path: { slug } },
        body,
        headers: bearer(token),
      });
      return {
        status: response.status,
        code: error?.code,
        errors: error?.errors?.map((e) => ({ field: e.field, code: e.code })),
      };
    })
  );
  expect(answers, "the refusals").toEqual(refusals.map((r) => r.want));
  expect(await countProjects(db)).toEqual(before);
  // No refusal changed the project created first.
  expect(await expectProjectCreated(db, slug, web, memberEmail, adminEmail, members)).toBe(created.id);

  // A new project goes first in the sidebar of each of its admins (M3 design 3.18), each by his own places: the
  // admin's Ops before his Web; then the member's Docs, led by the admin, before the member's Web and the admin's Ops.
  const ops = await createProject(api, admin, slug, { name: "Ops", identifier: "ops", network: 0 });
  expect(ops).toMatchObject({
    identifier: "OPS",
    network: 0,
    member_role: 20,
    sort_order: 55535,
    member_ids: [adminId],
  });
  const docs = await createProject(api, member, slug, { name: "Docs", identifier: "docs", project_lead_id: adminId });
  expect(docs).toMatchObject({
    identifier: "DOCS",
    member_role: 20,
    sort_order: 55535,
    member_ids: [memberId, adminId],
  });
  const docsRow = {
    name: "Docs",
    identifier: "DOCS",
    description: "",
    network: 2,
    timezone: "Asia/Shanghai",
    logo_props: {},
  };
  expect(
    await expectProjectCreated(db, slug, docsRow, memberEmail, adminEmail, [
      { email: memberEmail, sort_order: 55535 },
      { email: adminEmail, sort_order: 45535 },
    ])
  ).toBe(docs.id);

  // An older project reads as its reader sees it (M3 design 3.19): the admin's private Ops, between his Web and his
  // Docs in his sidebar; Docs, which the member created with the admin its lead; the member does not see Ops.
  expect(await read(api, admin, ops.id)).toMatchObject({
    status: 200,
    project: { identifier: "OPS", network: 0, member_role: 20, sort_order: 55535, member_ids: [adminId] },
  });
  expect(await read(api, admin, docs.id)).toMatchObject({
    status: 200,
    project: { identifier: "DOCS", member_role: 20, sort_order: 45535, member_ids: [memberId, adminId] },
  });
  expect(await read(api, member, ops.id)).toEqual({ status: 404, code: "project.not_found" });
});
````

`e2e/stories/workspace/w3-workspace-settings.spec.ts`（修改，9 处）：

````old e2e/stories/workspace/w3-workspace-settings.spec.ts
import { createWorkspace, invite, inviteAndAccept, slugFor } from "../../fixtures/api";
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
import { createProject, createWorkspace, invite, inviteAndAccept, slugFor, type Workspace } from "../../fixtures/api";
import { expectProjectCreated } from "../../fixtures/assert/project";
````

````old e2e/stories/workspace/w3-workspace-settings.spec.ts
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
````

````old e2e/stories/workspace/w3-workspace-settings.spec.ts
// session switch of 7.1, comes with the general page (P9); P4 adds the
// projects to the deletion's assertions.
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
// session switch of 7.1, comes with the general page (P9).
````

````old e2e/stories/workspace/w3-workspace-settings.spec.ts
test("W3 (API): the admin changes the workspace and deletes it with its members, invitations and settings at one moment; a member may do neither, and the slug never changes", async ({
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
test("W3 (API): the admin changes the workspace and deletes it with its members, invitations, settings and projects at one moment; a member may do neither, and the slug never changes", async ({
````

````old e2e/stories/workspace/w3-workspace-settings.spec.ts
  // Two workspaces alike, each with the member, a pending and a declined invitation and the member's settings;
  // only Acme is deleted. Both exist before either is furnished, and the member is Other's admin and Acme's
  // member: each answer, role and row must be the workspace's own, whichever row the database reads first.
  await createWorkspace(api, admin, { name: "Other", slug: other });
  await createWorkspace(api, admin, { name: "Acme", slug });
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
  // Two workspaces alike, each with the member, a pending and a declined invitation, the member's settings and a
  // project the admin created with the member its lead; only Acme is deleted. Both exist before either is
  // furnished, and the member is Other's admin and Acme's member: each answer, role and row must be the
  // workspace's own, whichever row the database reads first.
  const [adminId, memberId] = await Promise.all([admin, member].map((token) => accountId(api, token)));
  const web = { name: "Web", identifier: "WEB", description: "", network: 2, timezone: "UTC", logo_props: {} };
  const otherWorkspace = await createWorkspace(api, admin, { name: "Other", slug: other });
  const acme = await createWorkspace(api, admin, { name: "Acme", slug });
````

````old e2e/stories/workspace/w3-workspace-settings.spec.ts
  const furnish = async (target: string, memberRole: 15 | 20) => {
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
  const furnish = async ({ slug: target, id }: Workspace, memberRole: 15 | 20) => {
````

````old e2e/stories/workspace/w3-workspace-settings.spec.ts
    expect(settings.response.status).toBe(200);
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
    expect(settings.response.status).toBe(200);
    // The answer is the workspace's own new project as its creator sees it, first in both admins' sidebars.
    expect(
      await createProject(api, admin, target, { name: "Web", identifier: "web", project_lead_id: memberId })
    ).toMatchObject({
      workspace_id: id,
      identifier: "WEB",
      member_role: 20,
      sort_order: 65535,
      member_ids: [adminId, memberId],
    });
````

````old e2e/stories/workspace/w3-workspace-settings.spec.ts
  await furnish(other, 20);
  await furnish(slug, 15);
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
  await furnish(otherWorkspace, 20);
  await furnish(acme, 15);
````

````old e2e/stories/workspace/w3-workspace-settings.spec.ts
  await expectMembership(db, other, memberEmail, { role: 20, is_active: true });
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
  await expectMembership(db, other, memberEmail, { role: 20, is_active: true });
  await expectProjectCreated(db, other, web, adminEmail, memberEmail, [
    { email: adminEmail, sort_order: 65535 },
    { email: memberEmail, sort_order: 65535 },
  ]);
````

- [ ] **Step 3: 测试、lint、前端检查、端到端**

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
Expected: 58 个全部通过（此前的 57 个，P1；W3 加了项目）。

- [ ] **Step 4: 提交**

```bash
git add e2e/fixtures/api.ts e2e/fixtures/assert/project.ts e2e/fixtures/assert/workspace.ts e2e/fixtures/auth.ts e2e/stories/project/p1-create-project.spec.ts e2e/stories/workspace/w3-workspace-settings.spec.ts
```
```bash
git commit -m "test(M3/P4a): story P1's API version; W3 asserts the projects' cascade

P1: a member creates a project with the admin its lead; both are its
admins, first in their sidebars, with the six states; each refusal
writes nothing; each new project goes first in its admins' sidebars;
older projects read as each reader sees them, a private one not at all
by a member who is not in it. W3: each workspace's project is its own,
and deleting the workspace deletes its projects, memberships, display
settings and states at its moment, the other workspace's untouched.
Labels join both in P7.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**变异**（spec 附录 A；每个都只运行它的故事）：

| 改坏 | 必须失败的故事 |
|---|---|
| 删除工作区的连带少显示设置一步 | W3 |
| 新项目的 Backlog 不是默认状态；负责人不成为项目成员 | P1 |
| 目录的两种读去掉 `slug`（两个行序） | P1（读）；W3（锁，P1 在一个行序） |
| `ShareMembers` 去掉 `workspace_id`（两个行序） | P1（陌生人是自己工作区的管理员） |
| `LowestSortOrder` 去掉 `workspace_id`；去掉 `user_id` | W3；P1、W3 |
| `IdentifierTaken` 去掉 `workspace_id`；去掉 `identifier` | P1 |
| `GetProject` 的成员列表去掉项目 | P1、W3 |
| `GetProject` 去掉项目的 id（两个行序） | P1（W3 在 id 升序） |
| `GetProject` 的调用者成员关系、显示设置的四个连接谓词（两个行序） | P1（管理员读 Ops、Docs：他在三个项目里的位置各不相同，Docs 他不是创建者） |
| `ProjectFacts` 去掉项目的 id、成员关系的项目、账户（两个行序） | P1（成员读私密的 Ops 得 404，管理员读得到） |
| 删除的四条语句去掉 `workspace_id` | W3 |

**Done when:** 58 个故事通过；P1、W3 单独运行时看得到上表的每个谓词；故事只断言本 Phase 已有的表（裁定 S4）。

---

### Task 15: 文档：差异清单、交接的处理结果

**Files:**
- Modify: `docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md`、`docs/v0/M3-workspace-project/handoffs/M1-P3-trim-platform.md`、`docs/v0/M3-workspace-project/handoffs/M2-closeout.md`、`docs/v0/plane-diff.md`

**Interfaces:** 没有代码。M3 设计 3.20 中 P4a 的一行：差异清单二·按表的四张表逐列（`projects` 的计数列定名为 `last_issue_sequence`），第四节中标 P4a 的行（含标着"P4a"的一半）；M2 收尾交接、M1-P2、M1-P3 交接的"处理结果（M3/P4a）"。

**Tests:** 没有新测试；`make lint-web` 的关键词守卫查文档。

- [ ] **Step 1: 差异清单和交接**

`docs/v0/plane-diff.md`（修改，4 处）：

````old docs/v0/plane-diff.md
| `workspace_member_invites` | 删除 `token`、`message` | 不存令牌：链接里的令牌是由签名密钥派生的 MAC 从邀请的 id 算出的，数据库泄露时待接受的链接不泄露（M3 设计 3.8、8.1）；`message` 没有写入方 |
````
````new docs/v0/plane-diff.md
| `workspace_member_invites` | 删除 `token`、`message` | 不存令牌：链接里的令牌是由签名密钥派生的 MAC 从邀请的 id 算出的，数据库泄露时待接受的链接不泄露（M3 设计 3.8、8.1）；`message` 没有写入方 |
| `projects` | 36 列保留 22 列，另加 `last_issue_sequence`（见下），共 23 列（M3/P4a，`00010_project_projects.sql`） | M3 设计 4.6 |
| `projects` | `workspace_id`：加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL` | 二·全局（模型的 `on_delete`） |
| `projects` | `project_lead_id`、`default_assignee_id`：模型的 `CASCADE` 改为 `ON DELETE SET NULL` | 物理删除一个账户不连带删除他负责的项目（M3 设计 3.15，删除关系图见 M3 设计 4.12） |
| `projects` | `name`：新加 `CHECK (name <> '' AND name !~ '[&+,:;$^}{*=?@#\|''<>.()%!-]')` | Plane 的禁用字符只在序列化器中检查（`serializers/project.py:39-44`，M3 设计 3.19） |
| `projects` | `identifier`：新加 `CHECK (identifier ~ '^[A-Z0-9ÇŞĞİÖÜ]{1,10}$')`，列类型仍是 `varchar(12)` | 见第四节"项目标识"（M3 设计 3.19） |
| `projects` | `network`：`CHECK (network >= 0)` 收紧为 `CHECK (network IN (0, 2))`，新加 `DEFAULT 2`；`description`：新加 `DEFAULT ''`；`cycle_view`、`module_view`、`issue_views_view`、`intake_view`、`guest_view_all_features`：新加 `DEFAULT false`；`archive_in`：新加 `DEFAULT 0`、`CHECK (archive_in BETWEEN 0 AND 12)`；`timezone`：新加 `DEFAULT 'UTC'`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 模型的默认值、choices 和验证器（0 私密、2 公开） |
| `projects` | `logo_props`：新加 `DEFAULT '{}'` 和 `projects_logo_props_check`：是对象，键只能是 `in_use`、`emoji`、`icon`，出现的每个键的值类型对，嵌套的 `emoji`、`icon` 两个对象也查到底；十五个反例各试一个条件，M3 设计 4.6 的十个在内（`TestProjectChecksRejectCounterexamples`） | 二·全局（对象型的 `jsonb` 列）；`{}` 表示没有图标（M3 设计 4.6） |
| `projects` | **新增** `last_issue_sequence integer NOT NULL DEFAULT 0 CHECK (last_issue_sequence >= 0)`：工作项编号的计数列，M4 取号；M3 不读写它，它不进入接口 | 替代 `issue_sequences`（一 B；v0-design 5.3） |
| `projects` | 两个部分唯一索引照搬，改名为 `projects_workspace_id_identifier_key`、`projects_workspace_id_name_key`（`ON (workspace_id, …) WHERE deleted_at IS NULL`；Plane 是 `project_unique_identifier_workspace_when_deleted_at_null`、`project_unique_name_workspace_when_deleted_at_null`，列的顺序相反）；新加不带条件的 `projects_workspace_id_idx ON (workspace_id)` | 按工作区列出项目用它们；物理级联要不带条件的索引（M3 设计 4） |
````

````old docs/v0/plane-diff.md
| `projects` | **新增**工作项编号计数列（列名在 M3 建表时确定） | 替代 `issue_sequences` |
| `project_members` | 删除 `view_props`、`default_props`、`preferences` | 和 `project_user_properties` 重复 |
````
````new docs/v0/plane-diff.md
| `projects` | 删除 `default_state_id`；`cover_image_asset_id` 暂不建，由 M5 随文件存储加入 | 新工作项的默认状态来自 `states."default"`（M1/P2 的交接）；在 M5 之前接口的 `cover_image_url` 是 `null` |
| `project_members` | 16 列保留 11 列（M3/P4a，`00011_project_project_members.sql`） | M3 设计 4.7 |
| `project_members` | `workspace_id`、`project_id`：加上 `ON DELETE CASCADE`；`member_id`：改为 `NOT NULL`（Plane 可为空），加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL` | 二·全局（模型的 `on_delete`）；没有成员的成员关系没有意义，Plane 也从不写入空值 |
| `project_members` | `role`：`CHECK (role >= 0)` 收紧为 `CHECK (role IN (5, 15, 20))`，新加 `DEFAULT 5`；`is_active`：新加 `DEFAULT true`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 只有三种角色（M3 设计 3.4）；模型的默认值 |
| `project_members` | 部分唯一索引照搬，改名为 `project_members_project_id_member_id_key ON (project_id, member_id) WHERE deleted_at IS NULL`（Plane 是 `project_member_unique_project_member_when_deleted_at_null`）；新加 `project_members_member_id_idx ON (member_id) WHERE deleted_at IS NULL`，不带条件的 `project_members_workspace_id_idx`、`project_members_project_id_idx` | 降为访客、结束成员关系按账户查；物理级联要不带条件的索引（M3 设计 4） |
| `project_members` | 删除 `view_props`、`default_props`、`preferences` | 和 `project_user_properties` 重复 |
| `project_members` | 删除 `sort_order`、`comment` | 侧边栏的顺序在 `project_user_properties.sort_order`，前端不读这一列；`comment` 没有写入方 |
| `project_user_properties` | 15 列保留 11 列（M3/P4a，`00012_project_project_user_properties.sql`） | M3 设计 4.8 |
| `project_user_properties` | `workspace_id`、`project_id`、`user_id`：加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL` | 二·全局（模型的 `on_delete`） |
| `project_user_properties` | `preferences`：新加默认值 `{"navigation": {"default_tab": "work_items", "hide_in_more_menu": []}}` 和 CHECK（外层恰好一个键 `navigation`，它恰好有字符串 `default_tab` 和数组 `hide_in_more_menu`）；`sort_order`：新加 `DEFAULT 65535`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 模型的默认值，去掉 `pages`（文档页已砍）；二·全局（对象型的 `jsonb` 列） |
| `project_user_properties` | 部分唯一索引照搬，改名为 `project_user_properties_project_id_user_id_key ON (project_id, user_id) WHERE deleted_at IS NULL`（Plane 是 `project_user_property_unique_user_project_when_deleted_at_null ON (user_id, project_id)`）；新加不带条件的 `project_user_properties_workspace_id_idx`、`project_user_properties_project_id_idx` | 物理级联要不带条件的索引（M3 设计 4） |
| `project_user_properties` | 删除 `filters`、`display_filters`、`display_properties`、`rich_filters` | 工作项列表的筛选和显示列由它们的使用者 M4 按自己的格式加回（M3 设计 3.18） |
| `states` | 18 列保留 14 列（M3/P4a，`00013_project_states.sql`） | M3 设计 4.9 |
| `states` | `workspace_id`、`project_id`：加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL` | 二·全局（模型的 `on_delete`） |
| `states` | `name`：新加 `CHECK (name <> '')`；`"group"`：新加 `DEFAULT 'backlog'` 和 `CHECK ("group" IN ('backlog', 'unstarted', 'started', 'completed', 'cancelled', 'triage'))`；`description`：新加 `DEFAULT ''`；`sequence`：新加 `DEFAULT 65535`；`"default"`：新加 `DEFAULT false`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 模型的默认值和 `StateGroup` |
| `states` | 部分唯一索引照搬，改名为 `states_project_id_name_key ON (project_id, name) WHERE deleted_at IS NULL`（Plane 是 `state_unique_name_project_when_deleted_at_null ON (name, project_id)`）；新加 `states_project_id_default_key ON (project_id) WHERE "default" AND deleted_at IS NULL`、`states_project_id_triage_key ON (project_id) WHERE "group" = 'triage' AND deleted_at IS NULL`，不带条件的 `states_workspace_id_idx`、`states_project_id_idx` | 见第四节"默认状态、分诊状态"（M3 设计 3.17）；物理级联要不带条件的索引（M3 设计 4） |
| `states` | 删除 `slug`、`is_triage` | `slug` 没有读取者；分诊状态只由 `"group" = 'triage'` 识别（M3 设计 3.17） |
````

````old docs/v0/plane-diff.md
| 删除工作区 | 清掉别人的 `last_workspace_id`；成员、显示设置等的连带软删除由 Celery 异步完成 | 不清 `last_workspace_id`（登录后的落点规则让它无害，M3 设计 3.14）；工作区、成员、邀请、显示设置在同一个事务里、同一时刻软删除，删除之后 slug 立即可以重用。项目由 M3/P4、标签由 M3/P7 加入这个事务 |
````
````new docs/v0/plane-diff.md
| 删除工作区 | 清掉别人的 `last_workspace_id`；成员、显示设置等的连带软删除由 Celery 异步完成 | 不清 `last_workspace_id`（登录后的落点规则让它无害，M3 设计 3.14）；工作区、成员、邀请、显示设置，项目和它们的成员关系、成员在项目里的显示设置、状态，在同一个事务里、同一时刻软删除，删除之后 slug 立即可以重用。标签由 M3/P7 加入这个事务 |
````

````old docs/v0/plane-diff.md
| 接受邀请时已有成员行 | 不分有效还是已离开，都把角色改为邀请的角色 | 已是有效成员：只消费邀请，成员关系和角色不变；以前的成员行：恢复，角色取邀请的（访客时项目角色的连带由 M3/P4 加入）（M3 设计 3.8） |
````
````new docs/v0/plane-diff.md
| 接受邀请时已有成员行 | 不分有效还是已离开，都把角色改为邀请的角色 | 已是有效成员：只消费邀请，成员关系和角色不变；以前的成员行：恢复，角色取邀请的；取的是访客时，同一个事务里他在这个工作区的项目角色都改为访客，含已离开的项目（M3 设计 3.8） |
| 项目负责人、默认负责人 | 外键 `CASCADE`；负责人可以是任何账户 | `ON DELETE SET NULL`；创建项目时，负责人须是工作区的有效管理员或成员，否则 422（`project_lead_id`，`not_allowed`），他与创建者都成为项目管理员（M3 设计 3.15、3.19）。修改项目时的规则由 M3/P4b 加入 |
| 看得到而不是成员时取项目 | 409（公开项目）或 403（私密项目） | 200，`member_role`、`sort_order` 为 `null`（M3 设计 3.19） |
| 工作区访客取没加入的公开项目 | 409 | 404 `project.not_found`：看不到（M3 设计 3.19） |
| 已归档的项目 | 取单个 404；列表里与未归档的混在一起 | 取单个照常返回；列表默认不含，`?archived=true` 只列它们（M3 设计 3.19） |
| 项目标识 | 最多 12 个字符，只禁一组符号 | 转成大写后 1–10 个，只能是 `A-Z`、`0-9` 和 `ÇŞĞİÖÜ`（M3 设计 3.19）；修改项目时同一规则由 M3/P4b 加入 |
| 默认状态、分诊状态 | 在代码里维持唯一；`is_triage` 可以与 `group` 不一致 | 数据库保证每个项目各至多一个（部分唯一索引）；分诊状态只看 `group`（M3 设计 3.17） |
````

`docs/v0/M3-workspace-project/handoffs/M2-closeout.md`（修改，1 处）：

````old docs/v0/M3-workspace-project/handoffs/M2-closeout.md
来源：[M3/P3 spec](../specs/P3-invitations.md) 第 7 节。

````
````new docs/v0/M3-workspace-project/handoffs/M2-closeout.md
来源：[M3/P3 spec](../specs/P3-invitations.md) 第 7 节。

## 处理结果（M3/P4a）

- **第 7 节 可空的引用字段**（部分）：`Project.cover_image_url` 在接口中必有、可为 `null`，M5 之前总是 `null`（`api/modules/project.yaml`；`listProjects`、`createProject`、`getProject` 的答复）；`IUserLite` 随 P8，本节保持 `open`。
- **第 9 节 物理删除与跨模块外键的关系图**（完成）：图在 M3 设计 4.12，每条指向 `users` 的外键写明去向；项目负责人、默认负责人由 Plane 的 `CASCADE` 改为 `ON DELETE SET NULL`（`server/migrations/sql/00010_project_projects.sql`），登记在差异清单二·按表的 `projects` 各行和第四节"项目负责人、默认负责人"。

仍未处理，状态保持 `open`：第 1 节的页面一侧（P9）；第 2、3、6 节，第 7 节的其余部分，第 10、11、13、14 节，随 M3 设计 13.1 中各自的 Phase；第 12 节等 M3 的收尾。

来源：[M3/P4a spec](../specs/P4a-projects.md) 第 7 节。

````

`docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md`（修改，1 处）：

````old docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md
来源：[M3/P2 spec](../specs/P2-workspaces.md) 第 7 节。

````
````new docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md
来源：[M3/P2 spec](../specs/P2-workspaces.md) 第 7 节。

## 处理结果（M3/P4a）

- **项目字段**（完成）：项目的接口（`api/modules/project.yaml` 的 `Project`、`ProjectCreate`）没有 `close_in`、`default_state`、`page_view`、`estimate_id`，表里也没有这几列（`server/migrations/sql/00010_project_projects.sql`；差异清单二·按表）。新工作项的默认状态来自 `states."default"`，每个项目至多一个（`states_project_id_default_key`）；项目的"自动化"只剩 `archive_in`。

仍未处理，状态保持 `open`：保留名单的前端一侧（P8）；个人主页的页面（P9）。

来源：[M3/P4a spec](../specs/P4a-projects.md) 第 7 节。

````

`docs/v0/M3-workspace-project/handoffs/M1-P3-trim-platform.md`（修改，1 处）：

````old docs/v0/M3-workspace-project/handoffs/M1-P3-trim-platform.md
来源：[M3/P1 spec](../specs/P1-platform.md) 第 7 节。

````
````new docs/v0/M3-workspace-project/handoffs/M1-P3-trim-platform.md
来源：[M3/P1 spec](../specs/P1-platform.md) 第 7 节。

## 处理结果（M3/P4a）

- **不再读的字段**（项目一侧完成）：项目的接口没有 `anchor` 和发布设置（`api/modules/project.yaml`）；Nerve 没有项目动态，不产生本文件列出的几类记录。视图、收集箱的接口不在 M3，由 M7 的同名交接（`docs/v0/M7-collaboration/handoffs/M1-P3-trim-platform.md`）约束，M3 的 review 写明。
- **地址**（`project-identifiers` 一条完成）：检查标识是 `GET /api/v0/workspaces/{slug}/project-identifiers/{identifier}`，不带结尾 `/`（`api/modules/project.yaml`）；页面改调它随项目的页面（P10）。

仍未处理，状态保持 `open`：项目成员、`RESTRICTED_URLS` 与后端同源（前端一侧，P8）和本文件的其余几条，随 M3 设计 13.1 中各自的 Phase。

来源：[M3/P4a spec](../specs/P4a-projects.md) 第 7 节。

````

- [ ] **Step 2: 检查**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过。

- [ ] **Step 3: 提交**

```bash
git add docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md docs/v0/M3-workspace-project/handoffs/M1-P3-trim-platform.md docs/v0/M3-workspace-project/handoffs/M2-closeout.md docs/v0/plane-diff.md
```
```bash
git commit -m "docs(M3/P4a): plane-diff for the project tables by column; the handoffs P4a closes

plane-diff lists projects, project_members, project_user_properties and
states column by column, and the 4.11 rows P4a lands: the lead and the
default assignee SET NULL and the lead's rule at creation, reading a
project one only sees, a guest's 404, archived projects, the identifier,
the default and triage states, and both cascades. M2-closeout 7 and 9,
M1-P2's project fields and M1-P3's fields and address record what P4a
closed.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**变异**：文档没有可变异的代码；每一句由 spec 附录 A 的清扫 6 逐句对照代码或测试核对。

**Done when:** 3.20 中 P4a 的一行写好；三份交接有"处理结果（M3/P4a）"；`make lint-web` 通过。
