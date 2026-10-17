# M4/P1a 工作项 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 工作项的五张表（迁移 `00015`–`00019`：`issues`、`issue_assignees`、`issue_labels`、`issue_subscribers`、`issue_activities`）和新模块 `issue`；工作项的写的取锁约定第一次落地：不加锁地读项目 → 预检（凡判定拒绝的 404、403 在这里答，不取任何锁）→ 事务 → 项目行 `FOR NO KEY UPDATE`（不锁工作区行）→ 锁下再判定（最终的）→ 依赖行的检查 → 编号 → 时刻 → 写（M4 设计 3.5，决策点 1 的 B）；`createIssue`（默认状态、默认负责人、编号、排序值、完成时间、关注、动态的第 1、2 行）；删除工作区改为工作区 N → 判定 → 邀请 N → 项目 N（`id` 升序）→ 读时刻；删除项目、删除工作区经过渡的 `IssueCascade` 在请求的事务里分批软删除五张表的行，删除在用的状态 409 `project.state_in_use`，删除标签从工作项上摘下它和它的子标签。交错 1、2、3、8、21、22 和交错 4–7 的创建一支在真实数据库上两种顺序串行；I1 的接口版本通过，M3 的 P4、W2、W3 带着工作项一侧的行删除。

**Architecture:** `server/migrations`：`00015`–`00019`。`issue` 模块：`domain`（`issue.go`、`activity.go`、`errors.go`、`actions.go`）、`app`（`ports.go` 由使用方声明的端口，`lock.go` 的 `Locks` 是每个工作项的写共用的一条取锁路径，`create_issue.go`，`cascade.go` 和 `delete_project_rows.go` 是连带的编排，`rows.go`）、`adapter/postgres`（`delete_rows.sql`、`issues.sql`、`cascade.sql` 的语句和存储）、`adapter/http`（`createIssue`）、`module.go`（`Provide` 交出 `IssueCascade`，`New` 接出用例）。`project`：`IssueProjects`（`issue_ports.sql`：项目行的锁、编号、状态、标签、成员的目录）、`IssueCascade` 端口（删除项目、删除工作区的最后一步，`deleteState` 的 `StateInUse`，`deleteLabel` 的 `DetachLabel`）、`Cascade.LockWorkspaceProjects`。`workspace`：删除工作区的新顺序，邀请的锁与删除分开。`bootstrap`：`issue.Provide` 先于各模块，`ports.go` 转换 `project`、`identity` 的端口给 `issue`；矩阵、取锁的形状、盖戳、连接、交错。`e2e`：`issues.ts`、`furnish.ts`、`assert/issue.ts`、`assert/catalog.ts`，I1，M3 的 P4、W2、W3、W11。不加 Go 模块、npm 包；模块之间只经端口。

**Tech Stack:** Go 1.27.1、pgx v5.11.0、sqlc v1.31.1（`CGO_ENABLED=0`）、oapi-codegen v2.8.0、goose v3.28.0、River v0.47.0、golangci-lint 2.13.2、PostgreSQL 18.6（testcontainers）；Node 24、pnpm 11.10.0、Playwright 1.63.0、TypeScript 5.8.3、oxlint 1.51.0、oxfmt 0.35.0。

**Spec:** `docs/v0/M4-issue-core/specs/P1a-issues.md`（上级：`docs/v0/M4-issue-core/M4-design.md`）

## Global Constraints

- **基点**：本 plan 和 spec 在 `d693f539`（设计的拆分提交）之上的一个提交里，M4 设计第 15 节 P1a 一行在同一个提交里改为"进行中"；Task 1 从这个提交开始。
- **Go 版本和依赖**：本 plan 不执行 `go get`，不加任何 Go 模块或 npm 包。`server/go.mod` 和 `server/tools/go.mod` 保持 `go 1.27` 和 `toolchain go1.27.1`（golangci-lint 2.13.2 由 go1.27.0 构建，`go` 行更高就拒绝运行）。每个 Task 提交前执行 `grep -n "^go \|^toolchain" server/go.mod server/tools/go.mod`，四行必须是 `go 1.27` 和 `toolchain go1.27.1`；`git diff --stat d693f5395b37ea975e0c09d56e50be40c0205a87 -- server/go.mod server/go.sum server/tools pnpm-lock.yaml` 在本 plan 的任何时刻都没有输出。
- **每个改了 `server/` 的 Task（Task 1–13）提交前**：`make lint-go` 两段都输出 `0 issues.`，`make test` 全部通过（没有 `FAIL`）。Task 14 只改 `e2e/`，跑 Web 的三个门禁和 `make e2e`；Task 15 只改文档，跑 `make lint-web`、`doclinks.py` 和本 Phase 最后一次全量的 `make test`。
- **生成的文件不手写、不从本 plan 复制**：有生成物的 Task 执行 Task 中的生成命令（只动了 sqlc 的 Task 1、2、3、6、7 执行 `make gen-go`，改了接口描述的 Task 9、11 执行 `make gen`），用 `shasum -a 256` 和 `wc -l` 核对 Task 末尾表中的 SHA-256 和行数，对不上时停下来：说明某个输入与本 plan 不一致。这些 Task **提交之后**执行 `make gen-check`（它用 `git status` 判断生成物是否已提交，提交之前必然报差异）。
- **前端检查和新的错误码**（M4 设计 12 节约束 4）：声明新码的 Task 在同一个 Task 里把它的文案加进 `web/apps/web/core/lib/error-messages.ts`（`PROBLEM_MESSAGES`）和两份 `errors.json`（`web/packages/i18n/src/locales/en/errors.json`、`zh-CN/errors.json`）：Task 11 `project.state_in_use`。改了 `tools/keywords.json` 的 Task 1，改了接口描述、前端或 `e2e/` 的 Task 2、3、9、11、14 另执行 `make lint-web`、`make knip`、`make test-web`；Task 1、2、3、11、14 执行 `make e2e`；Task 15 只改文档，执行 `make lint-web` 和 `doclinks.py`（下面"一次性的脚本"）。
- **e2e 的数目**：在 worktree 中 `make e2e` 必须全部通过：开始时 120 个，Task 14 起 121 个（I1）。只在不是 git 仓库的副本里，S3 因为构建读不到提交而失败（M3/P4b spec F4）；没有装 Firefox 的机器上 A4 的两个 firefox 测试因为浏览器不在而失败；两者都与本 plan 无关。
- **容器**：`make test` 和 `make e2e` 用自己的 testcontainers；机器忙时偶尔起不来，等 Docker 空闲之后重跑一次再当作失败。容器测试一次只跑一套。开发库 `nerve-dev-db-1` 不要启动、停止或重建，不要执行 `make dev-db-down`、`make dev-db-reset`。不要碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`、`nervewiki-*`）和留下的 testcontainers。
- **git**：每次 Bash 调用只执行一个 git 命令，不用 `;`、`&&`、`|` 串联 git；不用 `git -C`、`stash`、`clean`、`reset --hard`。`cd` 不与别的命令组合，只读的命令也不行。不碰 `plane/`、`refer/`。
- **安装**：除了 Docker、Go、Node 不做任何全局安装；不执行 `corepack enable`（pnpm 已在 PATH 上）。不把副本的 `node_modules` 链接到 worktree 的。
- **规则**：不写 `init()`，不用全局可变状态，构造函数显式传入依赖；不建 `utils`、`common`、`helpers` 包；一个文件只做一件事，不超过约 400 行；依赖只能向内；模块之间不互相导入，不跨模块的表 JOIN；模块的 SQL 只经 sqlc；端口由使用方声明；角色只按集合判断，不按大小比较；不留没有使用者的代码。工作项一侧的每个写经 `issue/app/lock.go` 的 `Locks`（一条共用的取锁路径），不写近似的副本；测试的帮手共用，不复制（`issue_world_test.go`、`issue_interleaving_test.go`、`assert/catalog.ts`）。**例外**：接口描述按模块一个文件（M0-P3 交接 5），`api/modules/project.yaml`（1358 行）、`api/modules/workspace.yaml`（872 行）不受约 400 行的限制；`tools/keywords.json` 是数据。本 plan 的其余代码文件都在约 400 行以内（最终原型上量的）：最长的是 `e2e/fixtures/api.ts`（403 行，基点 423 行，本 plan 移出 `furnishWorkspace`）、`server/migrations/schema_test.go`（400 行，本 plan 只改迁移的数目、表名和名字的清单）、`bootstrap/permission_matrix_test.go`（392 行）、`bootstrap/project_write_locks_test.go`（389 行）、`bootstrap/interleaving_growth_test.go`（389 行）、`e2e/fixtures/assert/project.ts`（385 行）、`bootstrap/interleaving_answers_test.go`（375 行）、`bootstrap/ports_test.go`（373 行）、`e2e/fixtures/assert/workspace.ts`（367 行）、`project/app/ports.go`（361 行）。W3 在基点已有 399 行，本 plan 把它的两个页面测试原样移到 `w3-workspace-settings-page.spec.ts`（spec 第 3 节第 12 条）。
- **关键词守卫**：72 条规则、23 个例外，没有命中。Task 1 把 `phase` 改为 `M4/P1`、两个 `/user-properties/` 例外的 `until` 改为 `M4/P11`（P1a、P1b 都写 `M4/P1`，裁定 5）。
- **注释**：Go、TS 代码、SQL 查询和接口描述用英文；迁移的注释和中文文档照本 plan 原样。每条注释和说明在它所在的 Task 结束时都与代码相符。
- **代码块**：每个改动都写成四个反引号围起来的块，块的第一行写明种类和路径，照原样使用（原型中逐字节运行过）：
  - ````` ````file <路径> ````` 新文件，块的内容加一个结尾换行就是整个文件；
  - ````` ````whole <路径> ````` 已有文件的完整新内容（同样加结尾换行）；
  - ````` ````old <路径> ````` 与紧跟着的 ````` ````new <路径> `````：`old` 的文字在文件当前版本中恰好出现一次，把它换成 `new` 的文字；
  - ````` ````delete <路径> ````` 删除这个文件（块是空的）。

  一个文件的几个块按出现的顺序依次应用。拼 plan 的脚本已从基点起按顺序核对过全部块：每个 `old` 恰好出现一次（在它之前的块应用之后的文件中），每个新文件原来不存在，逐 Task 应用之后的文件与原型逐字节相同；之后在另一个副本上从基点逐 Task 复现，每个 Task 之后跑它的门禁，最后的树与原型逐个文件相同（spec 附录 A）。块写入之后不必再格式化（原型中每个 Task 都经过 `gofmt`、`oxfmt`）。
- **过渡版本**：一些文件先在较早的 Task 写成过渡版本，较晚的 Task 再修改（文件结构一节的"Task"一列列出每个文件的全部 Task）：`issue/module.go`（Task 2、8、9、11）、`issue/app/cascade.go`（Task 2、11）、`issue/app/cascade_test.go`（Task 3、8、11）、`delete_rows.sql`、`delete_rows.go`、`delete_project_rows.go`（Task 2、3）、`project/module.go`（Task 1、2、6、11）、`project/app/issue_ports.go`（Task 2、6、11）、`project/app/cascade.go` 及测试（Task 1、2）、`workspace/app/delete_workspace.go` 及测试（Task 1、2）、`server/sqlc.yaml`、授权、迁移的测试（Task 2、3、4）、`bootstrap/app.go`（Task 2、9）、按目录的删除测试（Task 2、3）、`issue_writes_test.go`、`permission_matrix_issues_test.go`、`project_connection_test.go`（Task 9–12）、`api/modules/project.yaml`（Task 9、11）、`e2e/fixtures/assert/project.ts`（Task 2 暂时把 `issues` 写成没有随项目删除的行，Task 3 扩到五张表，Task 14 删去，spec 第 3 节第 11 条）。每个过渡版本都在逐 Task 复现中运行过。
- **变异**：每个 Task 末尾的"变异"表列出：把代码改坏的方式、必须因此失败的检查和它所在的层（单元：假实现；结构：迁移的测试；存储：真实数据库；组合：`bootstrap` 组合出的 app；端到端：单独运行的故事；守卫：关键词守卫；静态：`make lint-web` 的文案核对）。它们在各自 Task 的树上跑过，也在最终的原型上逐个跑过，每个变异在一份新的副本里（`$M4TMP/p1tools/mutants_p1a.py`，`mut.py` 在它写的每个检查上各跑一次；端到端的"发现"必须是故事的断言失败，`e2ekills.py` 核对，不算构建失败或超时；spec 附录 A）。表中"（Task n 起）"标出的检查从那个 Task 起才发现这个变异：在变异自己的 Task 的树上它不失败，在 Task n 的树上失败（`fromruns.py` 在那份快照上跑过）。**安全或加锁的性质只由单元一层发现的，算缺口**（brief 的缺陷类别）：表中每一条这类性质都另有存储、组合或端到端一层的检查，例外（"按性质只在单元一层"）写在 spec 附录 A。
- **评审敏感**（M4 设计 12 节约束 3）：取锁的一步（读 → 预检 → 项目 N → 重读 → 判定），预检拒绝的写不取任何锁，锁下的判定仍是最终的；删除工作区的新顺序与 M3 设计 3.6 的取锁表中的每个写没有新的等待环；新表进入两个连带，同一时刻、同一个删除者，按目录的删除测试不开豁免；每个新的批量写按约定五（按 `id` 的键集，`FOR NO KEY UPDATE`）。改动这些锁、检查或测试之前，先照"变异"表确认它在所说的性质去掉之后失败。
- **提交**：提交信息用英文，末尾加一行：`Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明。

## 一次性的脚本

`$M4TMP/p1tools/doclinks.py`（Task 15 的文档核对：给出的 markdown 文件中每个相对链接都指向仓库中存在的文件或目录）：

```python
# M4/P1a plan, Task 15 (from M3's closeout plan): every relative link of the markdown files given resolves, from the
# file's folder, to a file or folder of the repository (a link's #anchor is not checked). Prints one line per file and
# each link that does not resolve; exits 1 when one does not. usage: python3 doclinks.py <root> <markdown file>...
import os
import re
import sys

root = sys.argv[1]
LINK = re.compile(r'\]\(([^)\s]+)\)')
broken = 0
for rel in sys.argv[2:]:
    text = open(os.path.join(root, rel), encoding='utf-8').read()
    base = os.path.dirname(os.path.join(root, rel))
    links = [m.group(1) for m in LINK.finditer(text) if not re.match(r'[a-z]+:', m.group(1))]
    missing = [link for link in links
               if link.split('#')[0] and not os.path.exists(os.path.normpath(os.path.join(base, link.split('#')[0])))]
    print(f'{rel}: {len(links)} links, {len(missing)} broken')
    for link in missing:
        print('   ', link)
    broken += len(missing)
print('broken:', broken)
sys.exit(1 if broken else 0)
```

## 文件结构

路径相对于仓库根目录。一个文件由多个 Task 修改时，"Task"一列都列出（"新"是新文件）。生成的文件不在表中，钉在各自 Task 的末尾。

| 文件 | 职责 | Task |
|---|---|---|
| `tools/keywords.json`（修改） | `phase` 为 `M4/P1`；两个 `/user-properties/` 例外到 `M4/P11` | 1 |
| `server/internal/modules/project/adapter/postgres/queries/cascade.sql`、`server/internal/modules/project/adapter/postgres/cascade.go`（修改） | `LockWorkspaceProjects`：这个工作区未删除的项目，`id` 升序 `FOR NO KEY UPDATE` | 1 |
| `server/internal/modules/project/adapter/postgres/lock_workspace_projects_test.go`（新） | 存储测试：回答、按 `id` 加锁、锁的强度 | 1 |
| `server/internal/modules/project/adapter/postgres/failures_test.go`（修改） | 新方法的失败测试：`LockWorkspaceProjects`（Task 1）、`IssueProjects` 的五个方法（Task 6）、`DeleteLabel` 回答的 `id`（Task 11） | 1、6、11 |
| `server/internal/modules/project/app/ports.go`（修改） | `WorkspaceProjectsLocker` | 1 |
| `server/internal/modules/project/app/cascade.go`、`server/internal/modules/project/app/cascade_test.go`（修改） | `Cascade.LockWorkspaceProjects`（Task 1）；`DeleteWorkspaceProjects` 的最后一步是锁住的项目的工作项一侧（Task 2） | 1、2 |
| `server/internal/modules/project/module.go`（修改） | `Cascade` 接口加 `LockWorkspaceProjects`（Task 1）；`Deps.Issues`、`CascadeDeps.Issues`（Task 2）；`IssueProjects` 和它的类型（Task 6）；包说明（Task 11） | 1、2、6、11 |
| `server/internal/modules/workspace/adapter/postgres/queries/invitations.sql`、`server/internal/modules/workspace/adapter/postgres/invitations.go`、`server/internal/modules/workspace/adapter/postgres/failures_test.go`（修改） | 邀请的锁与删除分开：`LockWorkspaceInvitations`、`DeleteInvitations` | 1 |
| `server/internal/modules/workspace/adapter/postgres/delete_workspace_invitations_test.go`、`server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go`（修改） | 邀请按 `id` 加锁的存储测试（Task 1）；连带的调用多一个参数（Task 2） | 1、2 |
| `server/internal/modules/workspace/app/ports.go`、`server/internal/modules/workspace/app/delete_workspace.go`、`server/internal/modules/workspace/app/delete_workspace_test.go`、`server/internal/modules/workspace/app/fakes_projects_test.go`（修改） | 删除工作区的新顺序：工作区 N → 判定 → 邀请 N → 项目 N → 时刻 → 写（Task 1）；锁住的项目交给 `DeleteWorkspaceProjects`（Task 2） | 1、2 |
| `server/internal/modules/workspace/app/fakes_workspaces_test.go`、`server/internal/modules/workspace/app/clock_test.go`（修改） | 假实现的两个邀请方法；时钟在两步锁之后 | 1 |
| `server/internal/bootstrap/interleaving_answers_test.go`、`server/internal/bootstrap/interleaving_deletion_test.go`、`server/internal/bootstrap/interleaving_growth_test.go`、`server/internal/bootstrap/interleaving_writes_test.go`（修改） | `NewCascade` 多一个参数（Task 1、2）；`TestAWorkspacesDeletionWaitsForAWriteHoldingOneOfItsProjects`（Task 1） | 1、2 |
| `server/migrations/sql/00015_issue_issues.sql`（新） | `issues` 表（20 列） | 2 |
| `server/migrations/sql/00016_issue_issue_assignees.sql`、`server/migrations/sql/00017_issue_issue_labels.sql`、`server/migrations/sql/00018_issue_issue_subscribers.sql`、`server/migrations/sql/00019_issue_issue_activities.sql`（新） | 其余四张表 | 3 |
| `server/sqlc.yaml`、`deploy/runtime-grants.sql`、`server/migrations/schema_test.go`（修改） | `issue` 的 sqlc 条目和它的迁移；运行时的角色读写新表；迁移的数目和表名 | 2、3 |
| `server/migrations/issue_schema_test.go`（新） | `issueNames`（Task 2、3）；CHECK 的反例、唯一键（Task 4） | 2、3、4 |
| `server/internal/modules/issue/module.go`（新） | `Provide`、`Cascade`（Task 2）；`Actions`（Task 8）；`ProjectPorts`、`Deps`、`New`、`Register`（Task 9）；`Cascade` 的两个方法、包说明（Task 11） | 2、8、9、11 |
| `server/internal/modules/issue/app/cascade.go`（新） | `NewCascade`、`DeleteProjectIssues`、`DeleteWorkspaceIssues`（Task 2）；`StateInUse`、`DetachLabel`（Task 11） | 2、11 |
| `server/internal/modules/issue/app/delete_project_rows.go`、`server/internal/modules/issue/adapter/postgres/delete_rows.go`、`server/internal/modules/issue/adapter/postgres/queries/delete_rows.sql`（新） | 删除项目的工作项一侧：编排（按全局顺序逐表、按 `id` 的键集分批）和每张表一条语句（Task 2 一张表，Task 3 五张） | 2、3 |
| `server/internal/modules/issue/adapter/postgres/store.go`（新） | `Store`、`New`、事务里的查询 | 2 |
| `server/internal/modules/issue/app/cascade_test.go`（新） | 编排的单元测试（Task 3）；假实现随端口（Task 8、11） | 3、8、11 |
| `server/internal/modules/project/app/issue_ports.go`（新） | `IssueCascade`（Task 2、11）；`IssueProject`、`IssueState`（Task 6） | 2、6、11 |
| `server/internal/modules/project/app/fakes_issue_test.go`（新） | `IssueCascade` 的假实现 | 2、11 |
| `server/internal/modules/project/app/delete_project.go`、`server/internal/modules/project/app/delete_project_test.go`、`server/internal/modules/project/app/fakes_write_test.go`（修改） | 删除项目的最后一步 `DeleteProjectIssues` | 2 |
| `server/internal/bootstrap/users.go`、`server/internal/archtest/composition_test.go`、`server/internal/bootstrap/deactivating_test.go`、`server/internal/bootstrap/interleaving_accepting_test.go`、`server/internal/bootstrap/interleaving_endings_test.go`、`server/internal/bootstrap/interleaving_leaving_test.go`、`server/internal/bootstrap/interleaving_removal_test.go`、`server/internal/bootstrap/interleaving_roles_test.go`（修改） | 命令行的组合先 `issue.Provide`；测试的组合随之 | 2 |
| `server/internal/bootstrap/app.go`（修改） | `issue.Provide` 先于各模块，`IssueCascade` 交给 `project`（Task 2）；`issue.New` 和它的端口（Task 9） | 2、9 |
| `server/internal/bootstrap/actions_test.go`（修改） | 模块的动作清单加 `issue`（Task 2：没有动作；Task 8：`issue.create`） | 2、8 |
| `server/internal/bootstrap/project_deletion_test.go`、`server/internal/bootstrap/workspace_deletion_test.go`（修改） | 按目录的删除测试给工作项的表种下行（`seedIssues`，Task 2 一张表、Task 3 五张、两个项目）；工作项一侧失败时整个回滚 | 2、3 |
| `server/internal/bootstrap/ending_connection_test.go`（修改） | 组合随 `issue.Provide`（Task 2）；`newWorkspaceRoute` 带自己的 `TxManager`（Task 12） | 2、12 |
| `server/internal/bootstrap/project_connection_test.go`（修改） | 组合随 `issue.Provide`（Task 2）；说明提到 `issue` 的连带（Task 11）；`newProjectRouteOn`（Task 12） | 2、11、12 |
| `e2e/fixtures/assert/project.ts`（修改） | 过渡：工作项的表没有随项目删除的行（Task 2 `issues`，Task 3 五张）；Task 14 删去，读 `projectTables` | 2、3、14 |
| `server/internal/modules/issue/adapter/postgres/delete_rows_test.go`、`server/internal/modules/issue/adapter/postgres/store_test.go`（新） | 每条分批语句的存储测试；失败；存储测试的帮手 | 4 |
| `server/internal/modules/issue/domain/issue.go`、`server/internal/modules/issue/domain/issue_test.go`、`server/internal/modules/issue/domain/activity.go`、`server/internal/modules/issue/domain/activity_test.go`、`server/internal/modules/issue/domain/errors.go`（新） | 创建一侧的规则；动态的第 1、2 行；两个项目码 | 5 |
| `server/internal/modules/project/adapter/postgres/queries/issue_ports.sql`、`server/internal/modules/project/adapter/postgres/issue_ports.go`、`server/internal/modules/project/adapter/postgres/issue_ports_test.go`（新） | 项目行的锁、编号、状态、标签、成员的目录 | 6 |
| `server/internal/modules/issue/adapter/postgres/queries/issues.sql`、`server/internal/modules/issue/adapter/postgres/issues.go`、`server/internal/modules/issue/adapter/postgres/issues_test.go`、`server/internal/modules/issue/app/rows.go`（新） | 创建的存储：最大的 `sort_order`、五种插入、详情的重读 | 7 |
| `server/internal/modules/issue/app/ports.go`、`server/internal/modules/issue/app/lock.go`、`server/internal/modules/issue/app/create_issue.go`、`server/internal/modules/issue/app/create_issue_test.go`、`server/internal/modules/issue/app/fakes_test.go`、`server/internal/modules/issue/domain/actions.go`（新） | 端口（使用方声明）、`Locks`（预检、锁下的判定）、`createIssue`；`issue.create` | 8 |
| `server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`（修改） | `issue.create`：项目级，管理员、成员 | 8 |
| `api/modules/issue.yaml`（新）；`api/openapi.yaml`（修改） | `createIssue`、`IssuePriority`、`IssueDetail`、`IssueCreate`；`issue` 标签 | 9 |
| `api/modules/project.yaml`（修改） | `Project.identifier`、`default_assignee_id`、`State.default` 的说明（Task 9）；`deleteState` 的 `project.state_in_use`、`deleteProject` 的描述（Task 11） | 9、11 |
| `server/internal/modules/issue/adapter/http/gen/oapi-codegen.yaml`、`server/internal/modules/issue/adapter/http/handler.go`、`server/internal/modules/issue/adapter/http/handler_test.go`、`server/internal/modules/issue/adapter/http/issues.go`、`server/internal/modules/issue/adapter/http/issues_test.go`（新） | 生成的配置；`UseCases`、`Register`；`createIssue` 的处理函数和它的测试（`apitest.Main`） | 9 |
| `server/internal/bootstrap/ports.go`、`server/internal/bootstrap/ports_test.go`（修改） | `issueProjects`、`issueProfiles` 和它们的转换测试 | 9 |
| `server/internal/bootstrap/permission_matrix_issues_test.go`（新）；`server/internal/bootstrap/permission_matrix_test.go`（修改） | `createIssue` 的矩阵行（Task 9 五行，Task 12 加分诊状态）；`prepareMatrix` 接上它们 | 9、12（`permission_matrix_test.go`：9） |
| `server/internal/bootstrap/project_write_locks_test.go`（修改） | `writesOnAProject` 去掉标签为 `issue` 的写；形状测试的操作带真实的标签 | 9 |
| `server/internal/bootstrap/issue_world_test.go`、`server/internal/bootstrap/issue_write_locks_test.go`（新） | 工作项的测试世界；取锁的形状（交错 21）、`issueWrites` | 10 |
| `server/internal/bootstrap/issue_writes_test.go`（新） | 盖戳、事务的连接（Task 10）；`newIssueRoute` 带自己的 `TxManager`（Task 12） | 10、12 |
| `server/internal/modules/issue/adapter/postgres/queries/cascade.sql`、`server/internal/modules/issue/adapter/postgres/cascade.go`、`server/internal/modules/issue/adapter/postgres/cascade_test.go`（新） | `StateInUse`、`DetachLabels` 和它们的存储测试 | 11 |
| `server/internal/modules/project/app/delete_state.go`、`server/internal/modules/project/app/delete_state_test.go`、`server/internal/modules/project/app/delete_label.go`、`server/internal/modules/project/app/delete_label_test.go`、`server/internal/modules/project/app/clock_test.go`、`server/internal/modules/project/app/fakes_label_test.go`、`server/internal/modules/project/app/label_ports.go`、`server/internal/modules/project/domain/errors.go`（修改） | 删除状态问 `StateInUse`（409 `project.state_in_use`）；删除标签回答删除的 `id`，同一时刻 `DetachLabel` | 11 |
| `server/internal/modules/project/adapter/postgres/queries/labels.sql`、`server/internal/modules/project/adapter/postgres/labels.go`、`server/internal/modules/project/adapter/postgres/label_writes_test.go`、`server/internal/modules/project/adapter/http/state_writes_test.go`（修改） | `DeleteLabel` 回答删除的 `id`；HTTP 答 409 `project.state_in_use` | 11 |
| `api/modules/workspace.yaml`（修改） | `deleteWorkspace` 的描述提到工作项 | 11 |
| `web/apps/web/core/lib/error-messages.ts`、`web/packages/i18n/src/locales/en/errors.json`、`web/packages/i18n/src/locales/zh-CN/errors.json`（修改） | `project.state_in_use` 的文案 | 11 |
| `server/internal/bootstrap/issue_cascade_test.go`（新）；`server/internal/bootstrap/interleaving_states_test.go`、`server/internal/bootstrap/interleaving_labels_test.go`（修改） | 删除有工作项的状态 409、删除标签从工作项上摘下；M3 的交错用 `issue.Provide` 的连带 | 11 |
| `server/internal/bootstrap/issue_interleaving_test.go`、`server/internal/bootstrap/issue_interleaving_deletion_test.go`（新） | 交错的工具；交错 1、8、22，交错 4、5 的创建一支；交错 2、3 | 12 |
| `server/internal/bootstrap/issue_interleaving_catalog_test.go`（新） | 交错 6、7 的创建一支 | 13 |
| `e2e/fixtures/issues.ts`、`e2e/fixtures/furnish.ts`、`e2e/fixtures/assert/issue.ts`、`e2e/fixtures/assert/catalog.ts`（新）；`e2e/fixtures/api.ts`、`e2e/fixtures/assert/workspace.ts`（修改） | `createIssue` 的夹具；`furnishIssueRows`、移来的 `furnishWorkspace`；`expectIssueCreated`；`tablesUnder`、`projectTables`、`workspaceTables` | 14 |
| `e2e/stories/issue/i1-create-issue.spec.ts`、`e2e/stories/workspace/w3-workspace-settings-page.spec.ts`（新）；`e2e/stories/project/p4-archive.spec.ts`、`e2e/stories/workspace/w2-landing.spec.ts`、`e2e/stories/workspace/w3-workspace-settings.spec.ts`、`e2e/stories/workspace/w11-guest-bounds.spec.ts`（修改） | I1 的接口版本；P4、W2、W3 带着工作项一侧的行；W3 的页面版本移到自己的文件；W11 的表从目录读 | 14 |
| `docs/v0/v0-design.md`、`docs/v0/plane-diff.md`、`docs/v0/M3-workspace-project/M3-design.md`（修改） | 3.23 中 P1a 的行 | 15 |
| `docs/v0/M4-issue-core/handoffs/M1-P2-trim-content.md`、`docs/v0/M4-issue-core/handoffs/M1-P3-trim-platform.md`、`docs/v0/M4-issue-core/handoffs/M3-closeout.md`（修改） | 处理结果（M4/P1a） | 15 |

---

### Task 1: 本 Phase 的第一个提交：关键词守卫；删除工作区的新顺序（先锁邀请、再锁项目、再读时刻）

**Files:**
- Create: `server/internal/modules/project/adapter/postgres/lock_workspace_projects_test.go`
- Modify: `server/internal/bootstrap/interleaving_answers_test.go`、`server/internal/bootstrap/interleaving_deletion_test.go`、`server/internal/bootstrap/interleaving_growth_test.go`、`server/internal/bootstrap/interleaving_writes_test.go`、`server/internal/modules/project/adapter/postgres/cascade.go`、`server/internal/modules/project/adapter/postgres/failures_test.go`、`server/internal/modules/project/adapter/postgres/queries/cascade.sql`、`server/internal/modules/project/app/cascade.go`、`server/internal/modules/project/app/cascade_test.go`、`server/internal/modules/project/app/ports.go`、`server/internal/modules/project/module.go`、`server/internal/modules/workspace/adapter/postgres/delete_workspace_invitations_test.go`、`server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go`、`server/internal/modules/workspace/adapter/postgres/failures_test.go`、`server/internal/modules/workspace/adapter/postgres/invitations.go`、`server/internal/modules/workspace/adapter/postgres/queries/invitations.sql`、`server/internal/modules/workspace/app/clock_test.go`、`server/internal/modules/workspace/app/delete_workspace.go`、`server/internal/modules/workspace/app/delete_workspace_test.go`、`server/internal/modules/workspace/app/fakes_projects_test.go`、`server/internal/modules/workspace/app/fakes_workspaces_test.go`、`server/internal/modules/workspace/app/ports.go`、`tools/keywords.json`
- Generate: `server/internal/modules/project/adapter/postgres/gen/cascade.sql.go`、`server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go`

**Interfaces:**（spec 2.3；M4 设计 3.5、7.9）
- `tools/keywords.json`：`phase` 为 `M4/P1`，两个 `/user-properties/` 例外的 `until` 为 `M4/P11`。
- `project`：
  - `queries/cascade.sql` 的 `LockWorkspaceProjects`（这个工作区未删除的项目，含已归档的，`ORDER BY id FOR NO KEY UPDATE`，回答 `id`）；
  - `Store.LockWorkspaceProjects(ctx, workspaceID) ([]uuid.UUID, error)`；
  - 端口 `app.WorkspaceProjectsLocker`；`NewCascade(projects, locker, members, enders)`（`locker` 是新的第二个参数）和 `Cascade.LockWorkspaceProjects`；
  - `module.go` 的 `Cascade` 接口加这个方法。
- `workspace`：
  - 邀请的锁与删除分开：`queries/invitations.sql` 的 `LockWorkspaceInvitations`（`id` 升序 `FOR NO KEY UPDATE`，回答 `id`）、`DeleteInvitations`（给出的 `id`）；`Store.DeleteWorkspaceInvitations` 删去；
  - `app.WorkspaceDeleter` 的 `LockWorkspaceInvitations`、`DeleteInvitations(ctx, ids, by, now)`；
  - `app.ProjectCascade` 加 `LockWorkspaceProjects`；
  - `DeleteWorkspace.Execute`：工作区 N → 判定 → 邀请 N → 项目 N → `u.clock.Now()` → `cascade(ctx, id, invitations, by, now)`（工作区、锁住的邀请、成员、显示设置、项目一侧）。

**Tests:**
- 单元：
  - `delete_workspace_test.go` 的 `deletionLocks`：判定之后是两步锁，之后才有写；
  - 失败的表加"邀请的锁失败""项目的锁失败"两行；
  - `clock_test.go`：时钟在两步锁之后。
- 存储：
  - `TestLockWorkspaceProjects`（回答 acme 未删除的项目，含已归档的；锁住期间 `FOR SHARE` 等它而外键的 `FOR KEY SHARE` 不等；已删除的项目、beta 的项目不被锁）；
  - `TestLockWorkspaceProjectsLocksInIDOrder`（两个连接：Alpha 被持住时，它持着 Web 等 Alpha）；
  - `TestLockWorkspaceInvitationsLocksInIDOrder`（M3 的 `TestDeleteWorkspaceInvitationsLocksInIDOrder` 改名，测锁的一步）；
  - 两个失败测试的行。
- 组合：
  - `TestAWorkspacesDeletionWaitsForAWriteHoldingOneOfItsProjects`：一个事务持 Web 的 `FOR NO KEY UPDATE`（工作项的写将来就这样持项目）；删除 acme 在 `projects` 上等它；它提交之后，删除的时刻不早于提交。
  - M3 的交错测试照旧通过（`NewCascade` 多一个参数，`gatedDeleter` 在邀请的锁上开门）。

- [ ] **Step 1: 关键词守卫**

`tools/keywords.json`（修改，2 处）：

````old tools/keywords.json
{
  "phase": "M3/closeout",
````
````new tools/keywords.json
{
  "phase": "M4/P1",
````
````old tools/keywords.json
      "until": "M4"
````
````new tools/keywords.json
      "until": "M4/P11"
````

- [ ] **Step 2: 两条锁的查询、邀请的删除，生成**

`server/internal/modules/project/adapter/postgres/queries/cascade.sql`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/queries/cascade.sql
WHERE project_id = ANY (sqlc.arg(project_ids)::uuid[]) AND member_id = sqlc.arg(member_id) AND is_active AND deleted_at IS NULL;

````
````new server/internal/modules/project/adapter/postgres/queries/cascade.sql
WHERE project_id = ANY (sqlc.arg(project_ids)::uuid[]) AND member_id = sqlc.arg(member_id) AND is_active AND deleted_at IS NULL;

-- name: LockWorkspaceProjects :many
-- deleteWorkspace's lock of its projects (M4 design 3.5), under the workspace's FOR NO KEY UPDATE and after the lock
-- of its invitations, before it reads its moment: the workspace's undeleted projects, archived ones too, FOR NO KEY
-- UPDATE in id order: the deletion waits here for every write that holds one of them, so its moment is later than
-- theirs, also of a write that took no lock of the workspace row. The lock is taken as the sorted rows come, so the
-- order is the ids'. No project of the workspace is created or deleted while it waits: both take the workspace's row
-- FOR SHARE first.
SELECT id
FROM projects
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL
ORDER BY id
FOR NO KEY UPDATE;

````

`server/internal/modules/workspace/adapter/postgres/queries/invitations.sql`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/queries/invitations.sql
-- deleteWorkspace's cascade, under the workspace's FOR NO KEY UPDATE: every undeleted invitation of the workspace,
-- pending or declined, FOR NO KEY UPDATE in id order, which the store then deletes by these ids (DeleteInvitations).
````
````new server/internal/modules/workspace/adapter/postgres/queries/invitations.sql
-- deleteWorkspace's lock of its invitations, under the workspace's FOR NO KEY UPDATE and before the lock of its projects
-- (M4 design 3.5): every undeleted invitation of the workspace, pending or declined, FOR NO KEY UPDATE in id order,
-- which the deletion then deletes by these ids (DeleteInvitations).
````

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `84473a9fd034954e5fc78311d130aa82ccf7edfa8056356cc8227c262afd7cc0` | 331 | `server/internal/modules/project/adapter/postgres/gen/cascade.sql.go` |
| `9840ef36cf4a7e3f170451ef1f456a4d384da458942d80250f092e9e683f39c6` | 342 | `server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go` |

Run: `shasum -a 256 server/internal/modules/project/adapter/postgres/gen/cascade.sql.go server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go`
Expected: 与上表相同。

- [ ] **Step 3: `project` 一侧：存储方法、端口、`Cascade.LockWorkspaceProjects` 和它们的测试**

`server/internal/modules/project/adapter/postgres/cascade.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/cascade.go
}

// LockMemberProjects locks FOR NO KEY UPDATE, in id order, the workspace's
````
````new server/internal/modules/project/adapter/postgres/cascade.go
}

// LockWorkspaceProjects locks FOR NO KEY UPDATE, in id order, the
// workspace's undeleted projects, archived ones too, and returns their ids
// (app.WorkspaceProjectsLocker).
func (s *Store) LockWorkspaceProjects(ctx context.Context, workspaceID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).LockWorkspaceProjects(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("lock the workspace's projects: %w", err)
	}
	return ids, nil
}

// LockMemberProjects locks FOR NO KEY UPDATE, in id order, the workspace's
````

`server/internal/modules/project/adapter/postgres/failures_test.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("LockMemberProjects() = %v, %v; want context.Canceled, not no project", ids, err)
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("LockMemberProjects() = %v, %v; want context.Canceled, not no project", ids, err)
	}
	if ids, err := s.LockWorkspaceProjects(cancelled, acme); !failed(err) || ids != nil {
		t.Errorf("LockWorkspaceProjects() = %v, %v; want context.Canceled, not no project", ids, err)
````

`server/internal/modules/project/adapter/postgres/lock_workspace_projects_test.go`（新文件，100 行）：

````file server/internal/modules/project/adapter/postgres/lock_workspace_projects_test.go
package postgresadapter_test

import (
	"context"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// deleteWorkspace's lock of its projects (M4 design 3.5). It returns, in id
// order, acme's undeleted projects, the archived one too; until the
// transaction ends each of them is held FOR NO KEY UPDATE, which a FOR
// SHARE waits for and a foreign key's FOR KEY SHARE does not, and no other
// project is: not acme's deleted one, not beta's.
func TestLockWorkspaceProjects(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	arch, gone := newProject(t, s, acme, "Arch", "ARCH", alice), newProject(t, s, acme, "Gone", "GONE", alice)
	betas := newProject(t, s, beta, "Web", "WEB", alice)
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", arch, now)
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", gone, now)

	var locked []uuid.UUID
	err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		var err error
		if locked, err = s.LockWorkspaceProjects(ctx, acme); err != nil {
			return err
		}
		for name, p := range map[string]struct {
			id   uuid.UUID
			held bool
		}{"Web": {web, true}, "Ops": {ops, true}, "Arch": {arch, true}, "Gone": {gone, false}, "beta's Web": {betas, false}} {
			if share, keyShare := waits(t, pool, p.id, "FOR SHARE"), waits(t, pool, p.id, "FOR KEY SHARE"); share != p.held || keyShare {
				t.Errorf("%s while locked: a FOR SHARE waits %v, a FOR KEY SHARE waits %v; want %v, false", name, share, keyShare, p.held)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := slices.SortedFunc(slices.Values([]uuid.UUID{web, ops, arch}), uuid.UUID.Compare); !slices.Equal(locked, want) {
		t.Errorf("LockWorkspaceProjects() = %v, want %v", locked, want)
	}
}

// LockWorkspaceProjects takes its locks in the projects' id order, whatever
// order the rows lie in: Web has the smaller id, but lies after Alpha in
// the table and in the indexes on the name and on the identifier. Alpha's
// row is held. LockWorkspaceProjects waits for it holding Web's, which a
// FOR SHARE then waits for; in any other order it would reach Alpha first
// and wait holding nothing.
func TestLockWorkspaceProjectsLocksInIDOrder(t *testing.T) {
	s, pool := newStore(t)
	acme := newWorkspace(t, pool, "acme")
	web, alpha := uuid.NewV7(), uuid.NewV7() // web drawn first: the smaller id
	for _, p := range []struct {
		id   uuid.UUID
		name string
	}{{alpha, "Alpha"}, {web, "Web"}} {
		exec(t, pool, "INSERT INTO projects (id, workspace_id, name, identifier) VALUES ($1, $2, $3::text, upper($3::text))", p.id, acme, p.name)
	}
	held, err := pool.Begin(pgtest.Soon(t))
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
			_, err := s.LockWorkspaceProjects(ctx, acme)
			return err
		})
	}()
	pgtest.WaitForLockWaitOn(t, pool, "projects", 10*time.Second)

	if !waits(t, pool, web, "FOR SHARE") {
		t.Error("a FOR SHARE of Web while LockWorkspaceProjects waits for Alpha does not wait; want Web locked first")
	}
	if err := held.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("LockWorkspaceProjects() = %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("LockWorkspaceProjects() did not end within 10s")
	}
}
````

`server/internal/modules/project/app/cascade.go`（修改，2 处）：

````old server/internal/modules/project/app/cascade.go
	projects ProjectsDeleter
````
````new server/internal/modules/project/app/cascade.go
	projects ProjectsDeleter
	locker   WorkspaceProjectsLocker
````
````old server/internal/modules/project/app/cascade.go
// NewCascade returns the cascade over projects, members and enders.
func NewCascade(projects ProjectsDeleter, members MemberDemoter, enders MembershipEnder) *Cascade {
	return &Cascade{projects: projects, members: members, enders: enders}
````
````new server/internal/modules/project/app/cascade.go
// NewCascade returns the cascade over projects, locker, members and
// enders.
func NewCascade(projects ProjectsDeleter, locker WorkspaceProjectsLocker, members MemberDemoter, enders MembershipEnder) *Cascade {
	return &Cascade{projects: projects, locker: locker, members: members, enders: enders}
}

// LockWorkspaceProjects locks the workspace's undeleted projects, archived
// ones too, FOR NO KEY UPDATE in id order, and returns their ids in that
// order (M4 design 3.5): deleteWorkspace takes it under its FOR NO KEY
// UPDATE of the workspace and after the lock of the invitations, before it
// reads its moment, so that the moment is later than that of every write
// that holds one of these projects.
func (c *Cascade) LockWorkspaceProjects(ctx context.Context, workspaceID uuid.UUID) ([]uuid.UUID, error) {
	return c.locker.LockWorkspaceProjects(ctx, workspaceID)
````

`server/internal/modules/project/app/cascade_test.go`（修改，4 处）：

````old server/internal/modules/project/app/cascade_test.go
		err := app.NewCascade(f.store, &fakeDemoter{}, &fakeEnder{}).DeleteWorkspaceProjects(inTx, workspace, by, clockNow)
````
````new server/internal/modules/project/app/cascade_test.go
		err := app.NewCascade(f.store, &fakeLocker{}, &fakeDemoter{}, &fakeEnder{}).DeleteWorkspaceProjects(inTx, workspace, by, clockNow)
````
````old server/internal/modules/project/app/cascade_test.go
			t.Errorf("failing %q: DeleteWorkspaceProjects() = %v, the steps\n%q\nwant %v,\n%q", tt.fail, err, f.log.calls, tt.wantErr, tt.want)
````
````new server/internal/modules/project/app/cascade_test.go
			t.Errorf("failing %q: DeleteWorkspaceProjects() = %v, the steps\n%q\nwant %v,\n%q", tt.fail, err, f.log.calls, tt.wantErr, tt.want)
		}
	}
}

// fakeLocker is the projects' lock of a workspace's deletion: it records
// each call with its arguments, and " outside tx" when it ran outside the
// caller's transaction (callLog), answers with locked, and fails with
// errDisk when fail is set.
type fakeLocker struct {
	locked []uuid.UUID
	log    callLog
	fail   bool
}

func (f *fakeLocker) LockWorkspaceProjects(ctx context.Context, workspaceID uuid.UUID) ([]uuid.UUID, error) {
	f.log.add(ctx, "LockWorkspaceProjects %s", workspaceID)
	if f.fail {
		return nil, errDisk
	}
	return f.locked, nil
}

// LockWorkspaceProjects locks the workspace's projects in the caller's
// transaction and answers the ids as the lock returned them; a failing
// lock comes back as itself.
func TestLockWorkspaceProjects(t *testing.T) {
	workspace := uuid.NewV7()
	projects := []uuid.UUID{uuid.NewV7(), uuid.NewV7()}
	lock := []string{fmt.Sprintf("LockWorkspaceProjects %s", workspace)}
	tests := []struct {
		name    string
		locked  []uuid.UUID
		fail    bool
		wantErr error
		want    []uuid.UUID
	}{
		{"two projects", projects, false, nil, projects},
		{"none", nil, false, nil, nil},
		{"the lock failing", projects, true, errDisk, nil},
	}
	for _, tt := range tests {
		f := &fakeLocker{locked: tt.locked, fail: tt.fail}
		inTx := context.WithValue(context.Background(), inTxKey{}, true)
		got, err := app.NewCascade(newWrites().store, f, &fakeDemoter{}, &fakeEnder{}).LockWorkspaceProjects(inTx, workspace)
		if !errors.Is(err, tt.wantErr) || (tt.wantErr == nil) != (err == nil) || !slices.Equal(got, tt.want) || !slices.Equal(f.log.calls, lock) {
			t.Errorf("%s: LockWorkspaceProjects() = %v, %v, the calls %q; want %v, %v, %q", tt.name, got, err, f.log.calls, tt.want, tt.wantErr, lock)
````
````old server/internal/modules/project/app/cascade_test.go
		err := app.NewCascade(newWrites().store, f, &fakeEnder{}).DemoteToGuest(inTx, workspace, user, by, now)
````
````new server/internal/modules/project/app/cascade_test.go
		err := app.NewCascade(newWrites().store, &fakeLocker{}, f, &fakeEnder{}).DemoteToGuest(inTx, workspace, user, by, now)
````
````old server/internal/modules/project/app/cascade_test.go
		err := app.NewCascade(newWrites().store, &fakeDemoter{}, f).EndMemberships(inTx, workspaces, user, by, now)
````
````new server/internal/modules/project/app/cascade_test.go
		err := app.NewCascade(newWrites().store, &fakeLocker{}, &fakeDemoter{}, f).EndMemberships(inTx, workspaces, user, by, now)
````

`server/internal/modules/project/app/ports.go`（修改，1 处）：

````old server/internal/modules/project/app/ports.go
}

// MemberDemoter makes an account a guest in a workspace's projects (M3
````
````new server/internal/modules/project/app/ports.go
}

// WorkspaceProjectsLocker takes deleteWorkspace's lock of the workspace's
// projects (M4 design 3.5), in the transaction ctx carries.
type WorkspaceProjectsLocker interface {
	// LockWorkspaceProjects locks FOR NO KEY UPDATE, in id order, the
	// workspace's undeleted projects, archived ones too, and returns their
	// ids in that order.
	LockWorkspaceProjects(ctx context.Context, workspaceID uuid.UUID) ([]uuid.UUID, error)
}

// MemberDemoter makes an account a guest in a workspace's projects (M3
````

`server/internal/modules/project/module.go`（修改，2 处）：

````old server/internal/modules/project/module.go
type Cascade interface {
````
````new server/internal/modules/project/module.go
type Cascade interface {
	// LockWorkspaceProjects locks the workspace's undeleted projects FOR NO
	// KEY UPDATE in id order and returns their ids: deleteWorkspace's lock
	// of its projects, before it reads its moment (M4 design 3.5).
	LockWorkspaceProjects(ctx context.Context, workspaceID uuid.UUID) ([]uuid.UUID, error)
````
````old server/internal/modules/project/module.go
	return app.NewCascade(store, store, store)
````
````new server/internal/modules/project/module.go
	return app.NewCascade(store, store, store, store)
````

- [ ] **Step 4: `workspace` 一侧：邀请的两个方法、删除的新顺序和它们的测试**

`server/internal/modules/workspace/adapter/postgres/delete_workspace_invitations_test.go`（修改，5 处）：

````old server/internal/modules/workspace/adapter/postgres/delete_workspace_invitations_test.go
type noProjectRows struct{}

````
````new server/internal/modules/workspace/adapter/postgres/delete_workspace_invitations_test.go
type noProjectRows struct{}

func (noProjectRows) LockWorkspaceProjects(context.Context, uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

````
````old server/internal/modules/workspace/adapter/postgres/delete_workspace_invitations_test.go
// DeleteWorkspaceInvitations takes the workspace's undeleted invitations
// in id order, whatever order the rows lie in, FOR NO KEY UPDATE, before it
// deletes them, and takes no other row (M3 design 3.6 convention 5; ruling
// G-1). acme's undeleted invitations, to zoe, yan and xia, have ids in that
// order, but lie the other way round, in the table as in each index but the
// primary key's: xia's was inserted first, zoe's last, and the addresses
// sort the other way. beta's invitation and acme's deleted one, to wes, have
// the smallest ids. Three transactions hold zoe's, yan's and xia's FOR
// SHARE, one each, and let go one at a time, each step's probe naming the
// holder the deletion then waits behind: zoe's first, holding no row; then
// yan's, holding zoe's; then xia's, holding zoe's and yan's. In the table's
// order, or that of an index other than the primary key's, it would wait
// behind xia's first. Once xia's lets go it deletes the three, at its moment
// and by its account; wes's keeps its deletion, and beta's is not deleted.
func TestDeleteWorkspaceInvitationsLocksInIDOrder(t *testing.T) {
````
````new server/internal/modules/workspace/adapter/postgres/delete_workspace_invitations_test.go
// The deletion's lock of the workspace's invitations
// (LockWorkspaceInvitations) takes its undeleted ones in id order, whatever
// order the rows lie in, FOR NO KEY UPDATE, and no other row, and the
// deletion then deletes those (DeleteInvitations; M3 design 3.6 convention
// 5; ruling G-1). acme's undeleted invitations, to zoe, yan and xia, have
// ids in that order, but lie the other way round, in the table as in each
// index but the primary key's: xia's was inserted first, zoe's last, and
// the addresses sort the other way. beta's invitation and acme's deleted
// one, to wes, have the smallest ids. Three transactions hold zoe's, yan's
// and xia's FOR SHARE, one each, and let go one at a time, each step's
// probe naming the holder the deletion then waits behind: zoe's first,
// holding no row; then yan's, holding zoe's; then xia's, holding zoe's and
// yan's. In the table's order, or that of an index other than the primary
// key's, it would wait behind xia's first. Once xia's lets go it deletes
// the three, at its moment and by its account; wes's keeps its deletion,
// and beta's is not deleted.
func TestLockWorkspaceInvitationsLocksInIDOrder(t *testing.T) {
````
````old server/internal/modules/workspace/adapter/postgres/delete_workspace_invitations_test.go
			return s.DeleteWorkspaceInvitations(ctx, acme, alice, later)
````
````new server/internal/modules/workspace/adapter/postgres/delete_workspace_invitations_test.go
			ids, err := s.LockWorkspaceInvitations(ctx, acme)
			if err != nil {
				return err
			}
			return s.DeleteInvitations(ctx, ids, alice, later)
````
````old server/internal/modules/workspace/adapter/postgres/delete_workspace_invitations_test.go
			t.Fatalf("DeleteWorkspaceInvitations() = %v", err)
````
````new server/internal/modules/workspace/adapter/postgres/delete_workspace_invitations_test.go
			t.Fatalf("LockWorkspaceInvitations(), then DeleteInvitations() = %v", err)
````
````old server/internal/modules/workspace/adapter/postgres/delete_workspace_invitations_test.go
		t.Fatal("DeleteWorkspaceInvitations() did not end within 20s")
````
````new server/internal/modules/workspace/adapter/postgres/delete_workspace_invitations_test.go
		t.Fatal("LockWorkspaceInvitations(), then DeleteInvitations() did not end within 20s")
````

`server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go`（修改，4 处）：

````old server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
// deleteRows runs the store's four steps of a workspace's deletion, of the
// workspace's own rows, in the cascade's order (M3 design 3.6), on the
// workspace id, by the account by at now; the first that fails stops it.
````
````new server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
// deleteRows runs the store's part of a workspace's deletion, of the
// workspace's own rows, in the use case's order (M3 design 3.6, M4 design
// 3.5), on the workspace id, by the account by at now: the lock of its
// invitations, then the four steps, the invitations' on those it locked;
// the first that fails stops it.
````
````old server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
	for _, step := range []func(ctx context.Context, id, by uuid.UUID, now time.Time) error{
		s.DeleteWorkspace, s.DeleteWorkspaceInvitations, s.DeleteWorkspaceMembers, s.DeleteWorkspacePreferences,
````
````new server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
	invitations, err := s.LockWorkspaceInvitations(ctx, id)
	if err != nil {
		return err
	}
	for _, step := range []func() error{
		func() error { return s.DeleteWorkspace(ctx, id, by, now) },
		func() error { return s.DeleteInvitations(ctx, invitations, by, now) },
		func() error { return s.DeleteWorkspaceMembers(ctx, id, by, now) },
		func() error { return s.DeleteWorkspacePreferences(ctx, id, by, now) },
````
````old server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
		if err := step(ctx, id, by, now); err != nil {
````
````new server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
		if err := step(); err != nil {
````
````old server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
// noProjects is the project module's cascade: the deletion's last step,
// which the failure comes before; a deletion demotes no one.
type noProjects struct{}
````
````new server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
// noProjects is the project module's cascade over a workspace without
// projects: its lock finds none, and the deletion's last step comes after
// the failure; a deletion demotes no one.
type noProjects struct{}

func (noProjects) LockWorkspaceProjects(context.Context, uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}
````

`server/internal/modules/workspace/adapter/postgres/failures_test.go`（修改，2 处）：

````old server/internal/modules/workspace/adapter/postgres/failures_test.go
		t.Errorf("LockInvitationsToDelete() = %v, %v; want context.Canceled, no invitation", ids, err)
	}
````
````new server/internal/modules/workspace/adapter/postgres/failures_test.go
		t.Errorf("LockInvitationsToDelete() = %v, %v; want context.Canceled, no invitation", ids, err)
	}
	if ids, err := s.LockWorkspaceInvitations(cancelled, w.ID); !failed(err) || ids != nil {
		t.Errorf("LockWorkspaceInvitations() = %v, %v; want context.Canceled, no invitation", ids, err)
	}
````
````old server/internal/modules/workspace/adapter/postgres/failures_test.go
		"DeleteWorkspace": s.DeleteWorkspace, "DeleteWorkspaceInvitations": s.DeleteWorkspaceInvitations,
		"DeleteWorkspaceMembers": s.DeleteWorkspaceMembers, "DeleteWorkspacePreferences": s.DeleteWorkspacePreferences,
````
````new server/internal/modules/workspace/adapter/postgres/failures_test.go
		"DeleteWorkspace": s.DeleteWorkspace, "DeleteWorkspaceMembers": s.DeleteWorkspaceMembers,
		"DeleteWorkspacePreferences": s.DeleteWorkspacePreferences,
````

`server/internal/modules/workspace/adapter/postgres/invitations.go`（修改，3 处）：

````old server/internal/modules/workspace/adapter/postgres/invitations.go
// DeleteWorkspaceInvitations soft-deletes the undeleted invitations of the
// workspace, pending or declined, by the account by at now. It locks them
// FOR NO KEY UPDATE in id order, then deletes exactly those, in the
// caller's transaction (M3 design 3.6 convention 5; ruling G-1).
func (s *Store) DeleteWorkspaceInvitations(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	q := s.queries(ctx)
	ids, err := q.LockWorkspaceInvitations(ctx, workspaceID)
````
````new server/internal/modules/workspace/adapter/postgres/invitations.go
// LockWorkspaceInvitations locks the undeleted invitations of the
// workspace, pending or declined, FOR NO KEY UPDATE in id order, in the
// caller's transaction, and returns their ids in that order; the deletion
// deletes exactly those (DeleteInvitations; M3 design 3.6 convention 5,
// ruling G-1).
func (s *Store) LockWorkspaceInvitations(ctx context.Context, workspaceID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).LockWorkspaceInvitations(ctx, workspaceID)
````
````old server/internal/modules/workspace/adapter/postgres/invitations.go
		return fmt.Errorf("lock workspace invitations: %w", err)
````
````new server/internal/modules/workspace/adapter/postgres/invitations.go
		return nil, fmt.Errorf("lock workspace invitations: %w", err)
````
````old server/internal/modules/workspace/adapter/postgres/invitations.go
	if err := q.DeleteInvitations(ctx, gen.DeleteInvitationsParams{Ids: ids, DeletedBy: by, Now: now}); err != nil {
		return fmt.Errorf("delete workspace invitations: %w", err)
	}
	return nil
````
````new server/internal/modules/workspace/adapter/postgres/invitations.go
	return ids, nil
````

`server/internal/modules/workspace/app/clock_test.go`（修改，2 处）：

````old server/internal/modules/workspace/app/clock_test.go
// the one it waited for. The deletion's cascade uses that one read for
// every step, a change to guest or a restoring as a guest for the
// projects' step, and a removal and a leaving for each step of the ending.
````
````new server/internal/modules/workspace/app/clock_test.go
// the one it waited for. The deletion reads it after the locks of its
// invitations and its projects too (M4 design 3.5), and its cascade uses
// that one read for every step, a change to guest or a restoring as a
// guest for the projects' step, and a removal and a leaving for each step
// of the ending.
````
````old server/internal/modules/workspace/app/clock_test.go
		}, slices.Concat(lockedDecision(alice, acme, "LockWorkspaceBySlug", domain.ActionDelete), []string{"Now"}, cascadeCalls(alice, acme))},
````
````new server/internal/modules/workspace/app/clock_test.go
		}, slices.Concat(lockedDecision(alice, acme, "LockWorkspaceBySlug", domain.ActionDelete), deletionLocks(acme), []string{"Now"},
			cascadeCalls(alice, acme))},
````

`server/internal/modules/workspace/app/delete_workspace.go`（修改，3 处）：

````old server/internal/modules/workspace/app/delete_workspace.go
// cascade is what deleting a workspace soft-deletes, in the order of M3
// design 3.6: the workspace row, then the rows under it, each step at the
// same moment (the invitations locked in id order first, convention 5's
// exception), and last the projects and the rows under them, through
// ProjectCascade (M3 design 3.3).
func (u *DeleteWorkspace) cascade() []func(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return []func(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error{
		u.workspaces.DeleteWorkspace,
		u.workspaces.DeleteWorkspaceInvitations,
		u.workspaces.DeleteWorkspaceMembers,
		u.workspaces.DeleteWorkspacePreferences,
		u.projects.DeleteWorkspaceProjects,
	}
````
````new server/internal/modules/workspace/app/delete_workspace.go
// cascade soft-deletes, by by at now, in the order of M3 design 3.6: the
// workspace row, the invitations it locked, the memberships, the display
// settings, and last the projects and the rows under them, through
// ProjectCascade (M3 design 3.3). A failing step comes back as itself and
// the steps after it do not run.
func (u *DeleteWorkspace) cascade(ctx context.Context, id uuid.UUID, invitations []uuid.UUID, by uuid.UUID, now time.Time) error {
	for _, step := range []func() error{
		func() error { return u.workspaces.DeleteWorkspace(ctx, id, by, now) },
		func() error { return u.workspaces.DeleteInvitations(ctx, invitations, by, now) },
		func() error { return u.workspaces.DeleteWorkspaceMembers(ctx, id, by, now) },
		func() error { return u.workspaces.DeleteWorkspacePreferences(ctx, id, by, now) },
		func() error { return u.projects.DeleteWorkspaceProjects(ctx, id, by, now) },
	} {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
````
````old server/internal/modules/workspace/app/delete_workspace.go
// Execute deletes the workspace in one transaction (M3 design 3.6): the
// workspace row FOR NO KEY UPDATE, the decision on workspace.delete, then
// the cascade, every step at the one moment the clock gives under the lock.
// Nobody's last_workspace_id is cleared (M3 design 3.14).
````
````new server/internal/modules/workspace/app/delete_workspace.go
// Execute deletes the workspace in one transaction (M3 design 3.6, M4
// design 3.5): the workspace row FOR NO KEY UPDATE, the decision on
// workspace.delete, then its invitations and then its projects FOR NO KEY
// UPDATE, each in id order, then the cascade, every step at the one moment
// the clock gives after those locks: a write that held one of the projects
// has committed by then, so the deletion's moment is never earlier than
// its. Nobody's last_workspace_id is cleared (M3 design 3.14).
````
````old server/internal/modules/workspace/app/delete_workspace.go
		now := u.clock.Now()
		for _, step := range u.cascade() {
			if err := step(ctx, id, actor.UserID, now); err != nil {
				return err
			}
````
````new server/internal/modules/workspace/app/delete_workspace.go
		invitations, err := u.workspaces.LockWorkspaceInvitations(ctx, id)
		if err != nil {
			return err
		}
		if _, err := u.projects.LockWorkspaceProjects(ctx, id); err != nil {
			return err
		}
		if err := u.cascade(ctx, id, invitations, actor.UserID, u.clock.Now()); err != nil {
			return err
````

`server/internal/modules/workspace/app/delete_workspace_test.go`（修改，16 处）：

````old server/internal/modules/workspace/app/delete_workspace_test.go
	"errors"
````
````new server/internal/modules/workspace/app/delete_workspace_test.go
	"errors"
	"fmt"
````
````old server/internal/modules/workspace/app/delete_workspace_test.go
	"time"
````
````new server/internal/modules/workspace/app/delete_workspace_test.go
	"time"
	"uuid"
````
````old server/internal/modules/workspace/app/delete_workspace_test.go
}

func newDelete() (*app.DeleteWorkspace, *deleteFixture) {
````
````new server/internal/modules/workspace/app/delete_workspace_test.go
}

// lockedInvitations and lockedProjects are what the deletion's two locks
// answer: the ids, in id order, of the invitations and the projects it
// deletes.
var lockedInvitations, lockedProjects = []uuid.UUID{uuid.NewV7(), uuid.NewV7()}, []uuid.UUID{uuid.NewV7(), uuid.NewV7()}

func newDelete() (*app.DeleteWorkspace, *deleteFixture) {
````
````old server/internal/modules/workspace/app/delete_workspace_test.go
	f := &deleteFixture{log: log, tx: &fakeTx{}, workspaces: &fakeWorkspaces{log: log, workspaces: []domain.Workspace{acme, beta}},
		projects: &fakeProjects{log: log}, auth: &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{
````
````new server/internal/modules/workspace/app/delete_workspace_test.go
	f := &deleteFixture{log: log, tx: &fakeTx{}, workspaces: &fakeWorkspaces{log: log, workspaces: []domain.Workspace{acme, beta},
		invitations: lockedInvitations}, projects: &fakeProjects{log: log, locked: lockedProjects},
		auth: &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{
````
````old server/internal/modules/workspace/app/delete_workspace_test.go
}

// cascadeCalls are the deletion's steps on w by user, in order: the
// workspace's own, then the projects', through the project module.
````
````new server/internal/modules/workspace/app/delete_workspace_test.go
}

// deletionLocks are the deletion's locks of w's rows after its decision, in
// order: its invitations, then its projects, through the project module.
func deletionLocks(w domain.Workspace) []string {
	return []string{"LockWorkspaceInvitations " + w.ID.String(), "LockWorkspaceProjects " + w.ID.String()}
}

// cascadeCalls are the deletion's steps on w by user, in order: the
// workspace's own, the invitations locked among them, then the projects',
// through the project module.
````
````old server/internal/modules/workspace/app/delete_workspace_test.go
	for _, step := range []string{"DeleteWorkspace", "DeleteWorkspaceInvitations", "DeleteWorkspaceMembers", "DeleteWorkspacePreferences",
````
````new server/internal/modules/workspace/app/delete_workspace_test.go
	at := " by " + user.ID.String() + " at " + now.Format(time.RFC3339Nano)
	for _, step := range []string{"DeleteWorkspace", "DeleteInvitations", "DeleteWorkspaceMembers", "DeleteWorkspacePreferences",
````
````old server/internal/modules/workspace/app/delete_workspace_test.go
		calls = append(calls, step+" "+w.ID.String()+" by "+user.ID.String()+" at "+now.Format(time.RFC3339Nano))
````
````new server/internal/modules/workspace/app/delete_workspace_test.go
		if step == "DeleteInvitations" {
			calls = append(calls, fmt.Sprintf("%s %v%s", step, lockedInvitations, at))
			continue
		}
		calls = append(calls, step+" "+w.ID.String()+at)
````
````old server/internal/modules/workspace/app/delete_workspace_test.go
// DeleteWorkspace locks the workspace FOR NO KEY UPDATE, decides, then
// soft-deletes the workspace, its invitations, its members and their
// settings, and last its projects through the project module, by the
// caller at one moment, in one transaction (M3 design 3.6), and logs it.
// Two callers, two workspaces.
````
````new server/internal/modules/workspace/app/delete_workspace_test.go
// DeleteWorkspace locks the workspace FOR NO KEY UPDATE, decides, locks its
// invitations and then its projects through the project module, then
// soft-deletes the workspace, the invitations it locked, its members and
// their settings, and last its projects through the project module, by
// the caller at one moment, in one transaction (M3 design 3.6, M4 design
// 3.5), and logs it. Two callers, two workspaces.
````
````old server/internal/modules/workspace/app/delete_workspace_test.go
		want := append(lockedDecision(tt.user, tt.w, "LockWorkspaceBySlug", domain.ActionDelete), cascadeCalls(tt.user, tt.w)...)
````
````new server/internal/modules/workspace/app/delete_workspace_test.go
		want := slices.Concat(lockedDecision(tt.user, tt.w, "LockWorkspaceBySlug", domain.ActionDelete), deletionLocks(tt.w),
			cascadeCalls(tt.user, tt.w))
````
````old server/internal/modules/workspace/app/delete_workspace_test.go
	decided := lockedDecision(alice, acme, "LockWorkspaceBySlug", domain.ActionDelete)
````
````new server/internal/modules/workspace/app/delete_workspace_test.go
	decided := lockedDecision(alice, acme, "LockWorkspaceBySlug", domain.ActionDelete)
	locks := deletionLocks(acme)
	locked := slices.Concat(decided, locks)
````
````old server/internal/modules/workspace/app/delete_workspace_test.go
			failure, decided},
````
````new server/internal/modules/workspace/app/delete_workspace_test.go
			failure, decided},
		{"the invitations' lock failed", alice, "acme",
			func(f *deleteFixture) {
				f.workspaces.deleteErrs = map[string]error{"LockWorkspaceInvitations": failure}
			},
			failure, append(slices.Clone(decided), locks[0])},
		{"the projects' lock failed", alice, "acme",
			func(f *deleteFixture) { f.projects.errs = map[string]error{"LockWorkspaceProjects": failure} },
			failure, locked},
````
````old server/internal/modules/workspace/app/delete_workspace_test.go
			failure, append(slices.Clone(decided), steps[0])},
		{"the invitations failed", alice, "acme",
			func(f *deleteFixture) {
				f.workspaces.deleteErrs = map[string]error{"DeleteWorkspaceInvitations": failure}
			},
			failure, append(slices.Clone(decided), steps[:2]...)},
````
````new server/internal/modules/workspace/app/delete_workspace_test.go
			failure, append(slices.Clone(locked), steps[0])},
		{"the invitations failed", alice, "acme", func(f *deleteFixture) { f.workspaces.endErrs = map[string]error{"DeleteInvitations": failure} },
			failure, append(slices.Clone(locked), steps[:2]...)},
````
````old server/internal/modules/workspace/app/delete_workspace_test.go
			failure, append(slices.Clone(decided), steps[:3]...)},
````
````new server/internal/modules/workspace/app/delete_workspace_test.go
			failure, append(slices.Clone(locked), steps[:3]...)},
````
````old server/internal/modules/workspace/app/delete_workspace_test.go
			failure, append(slices.Clone(decided), steps[:4]...)},
````
````new server/internal/modules/workspace/app/delete_workspace_test.go
			failure, append(slices.Clone(locked), steps[:4]...)},
````
````old server/internal/modules/workspace/app/delete_workspace_test.go
			failure, append(slices.Clone(decided), steps...)},
````
````new server/internal/modules/workspace/app/delete_workspace_test.go
			failure, append(slices.Clone(locked), steps...)},
````
````old server/internal/modules/workspace/app/delete_workspace_test.go
			append(slices.Clone(decided), steps...)},
````
````new server/internal/modules/workspace/app/delete_workspace_test.go
			append(slices.Clone(locked), steps...)},
````

`server/internal/modules/workspace/app/fakes_projects_test.go`（修改，3 处）：

````old server/internal/modules/workspace/app/fakes_projects_test.go
	"fmt"
````
````new server/internal/modules/workspace/app/fakes_projects_test.go
	"fmt"
	"slices"
````
````old server/internal/modules/workspace/app/fakes_projects_test.go
// logs each call with its arguments, and fails a method with the error set
// for it, wrapped as the module wraps it.
````
````new server/internal/modules/workspace/app/fakes_projects_test.go
// logs each call with its arguments, answers LockWorkspaceProjects with
// locked, and fails a method with the error set for it, wrapped as the
// module wraps it.
````
````old server/internal/modules/workspace/app/fakes_projects_test.go
	log  *callLog
	errs map[string]error // by method, e.g. "DeleteWorkspaceProjects"
````
````new server/internal/modules/workspace/app/fakes_projects_test.go
	log    *callLog
	locked []uuid.UUID
	errs   map[string]error // by method, e.g. "DeleteWorkspaceProjects"
}

func (f *fakeProjects) LockWorkspaceProjects(ctx context.Context, workspaceID uuid.UUID) ([]uuid.UUID, error) {
	f.log.add(ctx, "LockWorkspaceProjects %s", workspaceID)
	if err := f.errs["LockWorkspaceProjects"]; err != nil {
		return nil, fmt.Errorf("lock the workspace's projects: %w", err)
	}
	return slices.Clone(f.locked), nil
````

`server/internal/modules/workspace/app/fakes_workspaces_test.go`（修改，2 处）：

````old server/internal/modules/workspace/app/fakes_workspaces_test.go
	invitations []uuid.UUID                       // what LockInvitationsToDelete returns: the ids it locked
````
````new server/internal/modules/workspace/app/fakes_workspaces_test.go
	invitations []uuid.UUID                       // what LockInvitationsToDelete and LockWorkspaceInvitations return: the ids they locked
````
````old server/internal/modules/workspace/app/fakes_workspaces_test.go
func (f *fakeWorkspaces) DeleteWorkspaceInvitations(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return f.deleteStep(ctx, "DeleteWorkspaceInvitations", workspaceID, by, now)
````
````new server/internal/modules/workspace/app/fakes_workspaces_test.go
// LockWorkspaceInvitations logs its call and returns the ids the fixture
// gave it (invitations), a copy, or fails with the error set for it in
// deleteErrs, wrapped as the store wraps it.
func (f *fakeWorkspaces) LockWorkspaceInvitations(ctx context.Context, workspaceID uuid.UUID) ([]uuid.UUID, error) {
	f.log.add(ctx, "LockWorkspaceInvitations %s", workspaceID)
	if err := f.deleteErrs["LockWorkspaceInvitations"]; err != nil {
		return nil, fmt.Errorf("lock workspace invitations: %w", err)
	}
	return slices.Clone(f.invitations), nil
````

`server/internal/modules/workspace/app/ports.go`（修改，3 处）：

````old server/internal/modules/workspace/app/ports.go
type WorkspaceDeleter interface {
	WorkspaceLocker
````
````new server/internal/modules/workspace/app/ports.go
type WorkspaceDeleter interface {
	WorkspaceLocker
	// LockWorkspaceInvitations locks the workspace's undeleted invitations,
	// pending or declined, FOR NO KEY UPDATE in id order, until the
	// transaction ends, and returns their ids in that order (M3 design 3.6
	// convention 5's exception).
	LockWorkspaceInvitations(ctx context.Context, workspaceID uuid.UUID) ([]uuid.UUID, error)
````
````old server/internal/modules/workspace/app/ports.go
	// DeleteWorkspaceInvitations soft-deletes its invitations, pending or
	// declined.
	DeleteWorkspaceInvitations(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error
````
````new server/internal/modules/workspace/app/ports.go
	// DeleteInvitations soft-deletes the undeleted invitations of ids, and
	// no other: those LockWorkspaceInvitations returned.
	DeleteInvitations(ctx context.Context, ids []uuid.UUID, by uuid.UUID, now time.Time) error
````
````old server/internal/modules/workspace/app/ports.go
type ProjectCascade interface {
````
````new server/internal/modules/workspace/app/ports.go
type ProjectCascade interface {
	// LockWorkspaceProjects locks the workspace's undeleted projects,
	// archived ones too, FOR NO KEY UPDATE in id order, until the
	// transaction ends, and returns their ids in that order: the deletion's
	// lock of its projects, after the lock of its invitations and before it
	// reads its moment (M4 design 3.5).
	LockWorkspaceProjects(ctx context.Context, workspaceID uuid.UUID) ([]uuid.UUID, error)
````

- [ ] **Step 5: 组合的测试**

`server/internal/bootstrap/interleaving_answers_test.go`（修改，2 处）：

````old server/internal/bootstrap/interleaving_answers_test.go
// first write.
````
````new server/internal/bootstrap/interleaving_answers_test.go
// locks of the rows under the workspace (M4 design 3.5) and its first
// write.
````
````old server/internal/bootstrap/interleaving_answers_test.go
func (d gatedDeleter) DeleteWorkspace(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	if err := d.gate.wait(ctx); err != nil {
		return err
	}
	return d.Store.DeleteWorkspace(ctx, id, by, now)
````
````new server/internal/bootstrap/interleaving_answers_test.go
func (d gatedDeleter) LockWorkspaceInvitations(ctx context.Context, workspaceID uuid.UUID) ([]uuid.UUID, error) {
	if err := d.gate.wait(ctx); err != nil {
		return nil, err
	}
	return d.Store.LockWorkspaceInvitations(ctx, workspaceID)
````

`server/internal/bootstrap/interleaving_deletion_test.go`（修改，2 处）：

````old server/internal/bootstrap/interleaving_deletion_test.go
// deletion holds it FOR NO KEY UPDATE and reads its time once it does, so
// it waits for the writes in flight and writes one time, after theirs, into
````
````new server/internal/bootstrap/interleaving_deletion_test.go
// deletion holds it FOR NO KEY UPDATE, then its invitations and its
// projects (M4 design 3.5), and reads its time once it holds them, so it
// waits for the writes in flight and writes one time, after theirs, into
````
````old server/internal/bootstrap/interleaving_deletion_test.go
		t.Errorf("%d memberships of carol's and dave's, want 3: hers of Ops and Web, his of Web", n)
	}
}

````
````new server/internal/bootstrap/interleaving_deletion_test.go
		t.Errorf("%d memberships of carol's and dave's, want 3: hers of Ops and Web, his of Web", n)
	}
}

// A workspace's deletion waits for a write that holds one of its projects
// FOR NO KEY UPDATE and not the workspace's row, and reads its time after
// that write ends (M4 design 3.5): a transaction holds Web so, as a work
// item's write will, which locks the project's row alone. Alice's deletion
// of acme locks acme, its invitations, then its projects in id order, and
// waits for Web's row; once the holder commits, it deletes Web at its one
// time, no earlier than the commit. A deletion that read its time before
// it locked the projects would delete Web at a time before the holder's
// commit, which the holder's rows would come after.
func TestAWorkspacesDeletionWaitsForAWriteHoldingOneOfItsProjects(t *testing.T) {
	r := newGrowthRace(t, false)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	holder, err := r.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = holder.Rollback(context.Background()) }()
	if _, err := holder.Exec(ctx, "SELECT 1 FROM projects WHERE id = $1 FOR NO KEY UPDATE", r.web); err != nil {
		t.Fatal(err)
	}
	deleted := run(func() error { return r.deleteAcme(ctx, workspacepg.New(r.pool)) })
	pgtest.WaitForLockWaitOn(t, r.pool, "projects", 5*time.Second)
	released := time.Now().Truncate(time.Microsecond)
	if err := holder.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := result(t, ctx, deleted, "the deletion"); err != nil {
		t.Fatal(err)
	}
	if n := r.deletedWithAcme(t, "SELECT deleted_at, updated_at, updated_by_id, created_at FROM projects WHERE id = $1", r.web); n != 1 {
		t.Fatalf("%d rows of Web, want 1", n)
	}
	var at time.Time
	if err := r.pool.QueryRow(ctx, "SELECT deleted_at FROM workspaces WHERE slug = 'acme'").Scan(&at); err != nil {
		t.Fatal(err)
	}
	if at.Before(released) {
		t.Errorf("acme deleted at %v, before the holder of Web let go at %v", at, released)
	}
}

````

`server/internal/bootstrap/interleaving_growth_test.go`（修改，2 处）：

````old server/internal/bootstrap/interleaving_growth_test.go
			auth, cascade := r.authorizer(), projectapp.NewCascade(store, store, store)
````
````new server/internal/bootstrap/interleaving_growth_test.go
			auth, cascade := r.authorizer(), projectapp.NewCascade(store, store, store, store)
````
````old server/internal/bootstrap/interleaving_growth_test.go
				cascade = projectapp.NewCascade(store, gatedDemoter{store, g}, store)
````
````new server/internal/bootstrap/interleaving_growth_test.go
				cascade = projectapp.NewCascade(store, store, gatedDemoter{store, g}, store)
````

`server/internal/bootstrap/interleaving_writes_test.go`（修改，2 处）：

````old server/internal/bootstrap/interleaving_writes_test.go
						return r.demote(ctx, workspacepg.New(r.pool), projectapp.NewCascade(store, store, store))
````
````new server/internal/bootstrap/interleaving_writes_test.go
						return r.demote(ctx, workspacepg.New(r.pool), projectapp.NewCascade(store, store, store, store))
````
````old server/internal/bootstrap/interleaving_writes_test.go
						return r.demote(ctx, workspacepg.New(r.pool), projectapp.NewCascade(store, gatedDemoter{store, g}, store))
````
````new server/internal/bootstrap/interleaving_writes_test.go
						return r.demote(ctx, workspacepg.New(r.pool), projectapp.NewCascade(store, store, gatedDemoter{store, g}, store))
````

- [ ] **Step 6: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/project/... ./internal/modules/workspace/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestAWorkspacesDeletionWaitsForAWriteHoldingOneOfItsProjects$|TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt$' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过；关键词守卫 `keywords: 72 rules, 23 exceptions, no hits.`

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 120 个全部通过（W3 删除工作区走新的顺序）。

- [ ] **Step 7: 提交**

```bash
git add server/internal/bootstrap/interleaving_answers_test.go server/internal/bootstrap/interleaving_deletion_test.go server/internal/bootstrap/interleaving_growth_test.go server/internal/bootstrap/interleaving_writes_test.go server/internal/modules/project/adapter/postgres/cascade.go server/internal/modules/project/adapter/postgres/failures_test.go server/internal/modules/project/adapter/postgres/lock_workspace_projects_test.go server/internal/modules/project/adapter/postgres/queries/cascade.sql server/internal/modules/project/app/cascade.go server/internal/modules/project/app/cascade_test.go server/internal/modules/project/app/ports.go server/internal/modules/project/module.go server/internal/modules/workspace/adapter/postgres/delete_workspace_invitations_test.go server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go server/internal/modules/workspace/adapter/postgres/failures_test.go server/internal/modules/workspace/adapter/postgres/invitations.go server/internal/modules/workspace/adapter/postgres/queries/invitations.sql server/internal/modules/workspace/app/clock_test.go server/internal/modules/workspace/app/delete_workspace.go server/internal/modules/workspace/app/delete_workspace_test.go server/internal/modules/workspace/app/fakes_projects_test.go server/internal/modules/workspace/app/fakes_workspaces_test.go server/internal/modules/workspace/app/ports.go tools/keywords.json server/internal/modules/project/adapter/postgres/gen/cascade.sql.go server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go
```
```bash
git commit -m "feat(M4/P1): a workspace's deletion locks its invitations and then its projects before it reads its moment

The keyword guard's phase is M4/P1, and the two user-properties
exceptions last until M4/P11. Deleting a workspace now takes, under the
workspace's row and after the decision, its invitations and then its
projects FOR NO KEY UPDATE, each in id order, and reads the clock only
then: a work item's write holds its project's row alone, and the
deletion waits for it and comes after it. The invitations' lock is a
step of its own; the cascade deletes the invitations it locked. The
project module's Cascade answers the projects it locked, for the work
items' side of the deletion to come.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p1a.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试；结构是迁移的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是关键词守卫；静态是 `make lint-web` 的文案和键的核对）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T1.1` | 删除工作区在判定之后就读时刻，在锁邀请和项目之前 | `internal/modules/workspace/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkspacesDeletion`（`make test`）、`internal/bootstrap` 的 `TestCreatingAWorkItemAndDeletingItsProjectOrWorkspaceSerialize`（`make test`）（Task 12 起） | 单元；组合 |
| `T1.2` | 删除工作区不锁它的项目 | `internal/modules/workspace/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkspacesDeletion`（`make test`） | 单元；组合 |
| `T1.3` | 删除工作区先锁项目、再锁邀请（与全局顺序相反） | `internal/modules/workspace/app` 的 全部测试（`make test`） | 单元 |
| `T1.4` | `LockWorkspaceProjects` 按行的存放次序加锁，不按 `id` | `internal/modules/project/adapter/postgres` 的 全部测试（`make test`） | 存储 |
| `T1.5` | `LockWorkspaceProjects` 连工作区里已删除的项目也锁 | `internal/modules/project/adapter/postgres` 的 全部测试（`make test`） | 存储 |
| `T1.6` | `LockWorkspaceProjects` 取 `FOR SHARE`：项目上的写不等它，它也不等那个写 | `internal/modules/project/adapter/postgres` 的 全部测试（`make test`） | 存储 |
| `T1.7` | `LockWorkspaceProjects` 取 `FOR UPDATE`：外键的 `FOR KEY SHARE` 要等它 | `internal/modules/project/adapter/postgres` 的 全部测试（`make test`） | 存储 |
| `T1.8` | `project` 的 `Cascade` 回答 `LockWorkspaceProjects` 而不加锁 | `internal/modules/project/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkspacesDeletion`（`make test`） | 单元；组合 |
| `T1.9` | 删除工作区不删除它锁住的邀请 | `internal/modules/workspace/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkspace`、`TestAFailedProjectsStep`、`TestADeletionRefused`（`make test`） | 单元；组合 |
| `T1.10` | 存储的 `LockWorkspaceProjects` 读失败时回答没有项目 | `internal/modules/project/adapter/postgres` 的 全部测试（`make test`） | 存储 |
| `T1.11` | 存储的 `LockWorkspaceInvitations` 读失败时回答没有邀请 | `internal/modules/workspace/adapter/postgres` 的 全部测试（`make test`） | 存储 |
| `T1.12` | `/user-properties/` 的例外在 `phase` 为 `M4/P1` 时仍到 `M4` 为止 | 关键词守卫（`make lint-web` 的第一步） | 守卫 |
| `T1.13` | 删除工作区锁邀请失败时照样往下做 | `internal/modules/workspace/app` 的 全部测试（`make test`） | 单元 |
| `T1.14` | `LockWorkspaceInvitations` 按行的存放次序加锁，不按 `id` | `internal/modules/workspace/adapter/postgres` 的 全部测试（`make test`） | 存储 |
| `T1.15` | `LockWorkspaceInvitations` 连已删除的邀请也锁（删除的语句自己跳过它们） | `internal/modules/workspace/adapter/postgres` 的 全部测试（`make test`） | 存储 |
| `T1.16` | `LockWorkspaceProjects` 锁每个工作区的项目 | `internal/modules/project/adapter/postgres` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkspace`、`TestAFailedProjectsStep`、`TestADeletionRefused`（`make test`）（Task 2 起） | 存储；组合 |

---

### Task 2: `issues` 表、`issue` 模块的骨架和两个连带的第一张表

**Files:**
- Create: `server/internal/modules/issue/adapter/postgres/delete_rows.go`、`server/internal/modules/issue/adapter/postgres/queries/delete_rows.sql`、`server/internal/modules/issue/adapter/postgres/store.go`、`server/internal/modules/issue/app/cascade.go`、`server/internal/modules/issue/app/delete_project_rows.go`、`server/internal/modules/issue/module.go`、`server/internal/modules/project/app/fakes_issue_test.go`、`server/internal/modules/project/app/issue_ports.go`、`server/migrations/issue_schema_test.go`、`server/migrations/sql/00015_issue_issues.sql`
- Modify: `deploy/runtime-grants.sql`、`e2e/fixtures/assert/project.ts`、`server/internal/archtest/composition_test.go`、`server/internal/bootstrap/actions_test.go`、`server/internal/bootstrap/app.go`、`server/internal/bootstrap/deactivating_test.go`、`server/internal/bootstrap/ending_connection_test.go`、`server/internal/bootstrap/interleaving_accepting_test.go`、`server/internal/bootstrap/interleaving_answers_test.go`、`server/internal/bootstrap/interleaving_deletion_test.go`、`server/internal/bootstrap/interleaving_endings_test.go`、`server/internal/bootstrap/interleaving_growth_test.go`、`server/internal/bootstrap/interleaving_leaving_test.go`、`server/internal/bootstrap/interleaving_removal_test.go`、`server/internal/bootstrap/interleaving_roles_test.go`、`server/internal/bootstrap/interleaving_writes_test.go`、`server/internal/bootstrap/project_connection_test.go`、`server/internal/bootstrap/project_deletion_test.go`、`server/internal/bootstrap/users.go`、`server/internal/bootstrap/workspace_deletion_test.go`、`server/internal/modules/project/app/cascade.go`、`server/internal/modules/project/app/cascade_test.go`、`server/internal/modules/project/app/delete_project.go`、`server/internal/modules/project/app/delete_project_test.go`、`server/internal/modules/project/app/fakes_write_test.go`、`server/internal/modules/project/module.go`、`server/internal/modules/workspace/adapter/postgres/delete_workspace_invitations_test.go`、`server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go`、`server/internal/modules/workspace/app/delete_workspace.go`、`server/internal/modules/workspace/app/delete_workspace_test.go`、`server/internal/modules/workspace/app/fakes_projects_test.go`、`server/internal/modules/workspace/app/ports.go`、`server/migrations/schema_test.go`、`server/sqlc.yaml`
- Generate: `server/internal/modules/issue/adapter/postgres/gen/db.go`、`server/internal/modules/issue/adapter/postgres/gen/delete_rows.sql.go`、`server/internal/modules/issue/adapter/postgres/gen/models.go`

**Interfaces:**（spec 2.4；M4 设计 3.3、3.6、4.2、6.6）
- 迁移 `00015_issue_issues.sql`：`issues`，20 列。
  - 约束：`issues_not_own_parent_check`、`issues_dates_check`、全表唯一的 `issues_project_id_sequence_id_key`；
  - 索引：外键列各一个不带条件的索引，加 `issues_project_id_state_id_idx ... WHERE deleted_at IS NULL`；Down 删表。
  - `server/sqlc.yaml` 加 `issue` 的条目（只读它自己的迁移）；`deploy/runtime-grants.sql` 给运行时的角色 `issues` 的读写。
- `issue` 模块：
  - `module.go`：`Cascade` 接口（`DeleteProjectIssues`、`DeleteWorkspaceIssues(ctx, projectIDs, by, now)`）、`Provided{IssueCascade}`、`Provide(pool)`、`deleteBatch = 1000`；
  - `app/cascade.go`：`NewCascade(rows, batch)` 和两个方法；
  - `app/delete_project_rows.go`：`RowBatch`、`ProjectRowsDeleter`、`rowSteps`、`deleteProjectRows`。每张表从 `after` 起每批 `limit` 行，下一批从这一批的最大 `id` 之后，直到一批为空；这个 Task 只有 `issues` 一张表；
  - `adapter/postgres`：`store.go`（`Store`、`New`、事务里的查询）、`delete_rows.go`（`DeleteProjectIssueRows`）、`queries/delete_rows.sql`（`DeleteProjectIssues`：`UPDATE ... WHERE id IN (SELECT ... ORDER BY id LIMIT ... FOR NO KEY UPDATE) RETURNING id`）。
  - `issue` 这时没有动作（`bootstrap/actions_test.go` 的清单写明为什么），动作随 Task 8 的用例。
- `project`：
  - `app/issue_ports.go` 的 `IssueCascade`（使用方声明）；
  - `NewDeleteProject(projects, issues, locks, tx, clock)`，最后一步 `DeleteProjectIssues`；
  - `NewCascade(projects, locker, members, enders, issues)`；`DeleteWorkspaceProjects(ctx, ws, projectIDs, by, now)` 的最后一步 `DeleteWorkspaceIssues(projectIDs, ...)`；
  - `module.go` 的 `CascadeDeps.Issues`、`Deps.Issues`，`Cascade` 接口的签名。
- `workspace`：`ProjectCascade.DeleteWorkspaceProjects` 多一个参数，传入 Task 1 锁住的项目。
- 组合：
  - `bootstrap/app.go` 先 `issue.Provide(pool)`，把 `IssueCascade` 交给 `project`；`users.go`（命令行的组合）同样；
  - `archtest` 的 `Users` 也建 `issue.Provide`，`Workspaces` 不建；
  - 测试的组合（交错、连接的测试）用 `issue.Provide(pool).IssueCascade`，`interleaving_growth_test.go` 的 `cascadeOn`。
- 端到端的过渡（spec 第 3 节第 11 条）：`expectProjectDeleted` 把 `issues` 写成"没有随项目删除的行，也没有没删的行"（`withoutRows`），注释写明为什么；Task 3 扩到五张表，Task 14 删去。

**Tests:**
- 迁移：
  - `TestMigrationsGoUpDownAndUpAgain`：15 个迁移，`issues` 在表的列表里；
  - `TestConstraintAndIndexNames`：`issueNames`（`issue_schema_test.go`）的 20 个名字和种类。
- 单元：
  - `project` 的 `TestDeleteProject`、`TestDeleteProjectReturnsEachFailure`：最后一步是 `DeleteProjectIssues`，同一时刻、同一个删除者；它的失败原样返回；
  - `TestDeleteWorkspaceProjects`：工作项一侧在最后，带着锁住的项目；
  - `workspace` 的 `cascadeCalls`：`DeleteWorkspaceProjects` 带着锁住的项目。
- 组合：
  - `TestDeletingAProjectLeavesNoUndeletedRowUnderIt`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`（目录驱动）：`keysTo` 读出 `issues` 的外键；`seedIssues` 由不是删除者的账户直接写一个工作项，它随项目、工作区在同一时刻由删除者删除；
  - `TestAProjectDeletionRefusedAtItsCommitChangesNoRow`、`TestADeletionRefusedAtItsCommitChangesNoRow` 照旧通过；
  - `TestCommandsComposeNoServerAndNoJobs`；`TestEveryModuleDeclaresItsActionsOrHasNone`（`issue` 没有动作）。

- [ ] **Step 1: 迁移、sqlc 配置、授权和迁移的测试**

`deploy/runtime-grants.sql`（修改，1 处）：

````old deploy/runtime-grants.sql
GRANT SELECT, INSERT, UPDATE, DELETE ON projects, project_members, project_user_properties, states, labels TO nerve_runtime;
````
````new deploy/runtime-grants.sql
GRANT SELECT, INSERT, UPDATE, DELETE ON projects, project_members, project_user_properties, states, labels TO nerve_runtime;
GRANT SELECT, INSERT, UPDATE, DELETE ON issues TO nerve_runtime;
````

`server/migrations/issue_schema_test.go`（新文件，27 行）：

````file server/migrations/issue_schema_test.go
package migrations_test

// issueNames are the constraints and indexes of the issue module's tables
// (M4 design 4.2–4.5, 4.11), as TestConstraintAndIndexNames reads them: the
// part of its want that the migration 00015 adds.
var issueNames = []string{
	"issues_created_by_id_fkey f n",
	"issues_dates_check c",
	"issues_name_check c",
	"issues_not_own_parent_check c",
	"issues_parent_id_fkey f c",
	"issues_parent_id_idx i",
	"issues_pkey iu",
	"issues_pkey p",
	"issues_priority_check c",
	"issues_project_id_fkey f c",
	"issues_project_id_idx i",
	"issues_project_id_sequence_id_key iu",
	"issues_project_id_sequence_id_key u",
	"issues_project_id_state_id_idx iw",
	"issues_sequence_id_check c",
	"issues_state_id_fkey f c",
	"issues_state_id_idx i",
	"issues_updated_by_id_fkey f n",
	"issues_workspace_id_fkey f c",
	"issues_workspace_id_idx i",
}
````

`server/migrations/schema_test.go`（修改，5 处）：

````old server/migrations/schema_test.go
	if err != nil || len(up) != 14 {
		t.Fatalf("Up() = %d migrations, %v; want 14", len(up), err)
````
````new server/migrations/schema_test.go
	if err != nil || len(up) != 15 {
		t.Fatalf("Up() = %d migrations, %v; want 15", len(up), err)
````
````old server/migrations/schema_test.go
		{tablesQuery, []string{"api_tokens", "auth_sessions", "labels", "profiles", "project_members", "project_user_properties", "projects", "river_job",
			"river_leader", "river_notification", "river_queue", "states", "users", "workspace_member_invites", "workspace_members",
````
````new server/migrations/schema_test.go
		{tablesQuery, []string{"api_tokens", "auth_sessions", "issues", "labels", "profiles", "project_members", "project_user_properties", "projects",
			"river_job", "river_leader", "river_notification", "river_queue", "states", "users", "workspace_member_invites", "workspace_members",
````
````old server/migrations/schema_test.go
	if again, err := m.Up(ctx); err != nil || len(again) != 14 {
		t.Errorf("Up() again = %d migrations, %v; want 14", len(again), err)
````
````new server/migrations/schema_test.go
	if again, err := m.Up(ctx); err != nil || len(again) != 15 {
		t.Errorf("Up() again = %d migrations, %v; want 15", len(again), err)
````
````old server/migrations/schema_test.go
	// module's tables are in projectNames (project_schema_test.go).
````
````new server/migrations/schema_test.go
	// module's tables are in projectNames (project_schema_test.go), the issue
	// module's in issueNames (issue_schema_test.go).
````
````old server/migrations/schema_test.go
	}, projectNames)
````
````new server/migrations/schema_test.go
	}, projectNames, issueNames)
````

`server/migrations/sql/00015_issue_issues.sql`（新文件，44 行）：

````file server/migrations/sql/00015_issue_issues.sql
-- issues：Plane 的 issues 表（28 列）按 M4 设计 4.2 保留 20 列。
-- 编号取自项目行的计数列 last_issue_sequence（总体设计 5.3）；描述、起止日期的列随表建出，进入接口在 P4、P2（3.7）。

-- +goose Up
CREATE TABLE issues (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    -- 父子在同一项目（3.9）；自己不能做自己的父工作项
    parent_id uuid REFERENCES issues ON DELETE CASCADE,
    -- Plane 可空，改为非空：创建总有状态（3.7）。删除在用的状态被拒绝（3.6），CASCADE 只删掉已软删除的工作项
    state_id uuid NOT NULL REFERENCES states ON DELETE CASCADE,
    name varchar(255) NOT NULL CHECK (name <> ''),
    description_html text NOT NULL DEFAULT '<p></p>',
    description_stripped text,
    priority varchar(30) NOT NULL DEFAULT 'none' CHECK (priority IN ('urgent', 'high', 'medium', 'low', 'none')),
    start_date date,
    target_date date,
    -- 没有默认值：取自项目的计数列
    sequence_id integer NOT NULL CHECK (sequence_id > 0),
    sort_order double precision NOT NULL DEFAULT 65535,
    completed_at timestamptz,
    -- Plane 是 date
    archived_at timestamptz,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    CONSTRAINT issues_not_own_parent_check CHECK (parent_id <> id),
    CONSTRAINT issues_dates_check CHECK (start_date <= target_date),
    -- 全表唯一，不是部分唯一：计数器从不回退，编号从不重用（3.7）；也服务按编号取
    CONSTRAINT issues_project_id_sequence_id_key UNIQUE (project_id, sequence_id)
);
-- 物理级联、子工作项、在用的状态按它们找行，所以不带条件（4.2）
CREATE INDEX issues_workspace_id_idx ON issues (workspace_id);
CREATE INDEX issues_project_id_idx ON issues (project_id);
CREATE INDEX issues_parent_id_idx ON issues (parent_id);
CREATE INDEX issues_state_id_idx ON issues (state_id);
-- 同一状态的 sort_order 的最大值、按状态筛选（3.7）
CREATE INDEX issues_project_id_state_id_idx ON issues (project_id, state_id) WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE issues;
````

`server/sqlc.yaml`（修改，1 处）：

````old server/sqlc.yaml
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
````new server/sqlc.yaml
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
  - engine: postgresql
    schema:
      - migrations/sql/00015_issue_issues.sql
    queries: internal/modules/issue/adapter/postgres/queries
    gen:
      go:
        package: gen
        out: internal/modules/issue/adapter/postgres/gen
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

- [ ] **Step 2: 删除的语句，生成**

`server/internal/modules/issue/adapter/postgres/queries/delete_rows.sql`（新文件，14 行）：

````file server/internal/modules/issue/adapter/postgres/queries/delete_rows.sql
-- name: DeleteProjectIssueRows :many
-- The deletion of a project's work-item side (M4 design 3.6), the work items' table: a batch of the project's
-- undeleted rows after the id `after`, in id order, locked FOR NO KEY UPDATE as they come and soft-deleted at the
-- moment and by the account given (convention 5: two batches over the same rows take them in one order). The caller
-- runs it again from the batch's greatest id until a batch finds none (a row deleted while the batch waits for it is
-- passed over and the next one taken; each batch reads a new snapshot). The primary key's index serves the id order
-- and the bound. It returns the ids it deleted.
UPDATE issues AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issues r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;
````

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `1dfb2b6312c3c3db25a8cad84c031995c3b0546c73b3585a0282c98e11397676` | 32 | `server/internal/modules/issue/adapter/postgres/gen/db.go` |
| `3dc81983d38c58d226751d066619cb37a21f6e492f95c494e3eabc576b3160b4` | 63 | `server/internal/modules/issue/adapter/postgres/gen/delete_rows.sql.go` |
| `2efad096aca0eaea33ce6550b369dadc129d4b972e8128c0167bc7551f990192` | 35 | `server/internal/modules/issue/adapter/postgres/gen/models.go` |

Run: `shasum -a 256 server/internal/modules/issue/adapter/postgres/gen/db.go server/internal/modules/issue/adapter/postgres/gen/delete_rows.sql.go server/internal/modules/issue/adapter/postgres/gen/models.go`
Expected: 与上表相同。

- [ ] **Step 3: `issue` 模块的骨架**

`server/internal/modules/issue/adapter/postgres/delete_rows.go`（新文件，34 行）：

````file server/internal/modules/issue/adapter/postgres/delete_rows.go
package postgresadapter

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// The batches of a deleted project's work-item side (app.ProjectRowsDeleter,
// delete_rows.sql): each soft-deletes the batch's rows of one table and
// returns their ids.

// batch is b as the statement takes it.
func batch(b app.RowBatch) gen.DeleteProjectIssueRowsParams {
	return gen.DeleteProjectIssueRowsParams{At: b.At, DeletedBy: b.By, ProjectID: b.ProjectID, After: b.After, Batch: int32(b.Limit)}
}

// deleted returns the batch's ids, or its failure named by what.
func deleted(ids []uuid.UUID, err error, what string) ([]uuid.UUID, error) {
	if err != nil {
		return nil, fmt.Errorf("delete a batch of the project's %s: %w", what, err)
	}
	return ids, nil
}

// DeleteProjectIssueRows soft-deletes a batch of the project's work items,
// archived ones too.
func (s *Store) DeleteProjectIssueRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteProjectIssueRows(ctx, batch(b))
	return deleted(ids, err, "work items")
}
````

`server/internal/modules/issue/adapter/postgres/store.go`（新文件，28 行）：

````file server/internal/modules/issue/adapter/postgres/store.go
// Package postgresadapter is the issue module's repository adapter: sqlc
// queries (queries/, generated into gen/) over the transaction that the
// context carries, or the pool.
package postgresadapter

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// Store implements the issue repository ports of app.
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

`server/internal/modules/issue/app/cascade.go`（新文件，39 行）：

````file server/internal/modules/issue/app/cascade.go
package app

import (
	"context"
	"time"
	"uuid"
)

// Cascade carries out the project module's deletions on the work items
// (project's IssueCascade port, M4 design 3.3): it soft-deletes a deleted
// project's work-item-side rows in the deletion's transaction, at its
// moment and by its account, through the statements' orchestration
// (deleteProjectRows).
type Cascade struct {
	rows  ProjectRowsDeleter
	batch int
}

// NewCascade returns the cascade over rows, batch rows a statement.
func NewCascade(rows ProjectRowsDeleter, batch int) *Cascade {
	return &Cascade{rows: rows, batch: batch}
}

// DeleteProjectIssues soft-deletes the project's work-item-side rows:
// deleteProject's last step.
func (c *Cascade) DeleteProjectIssues(ctx context.Context, projectID, by uuid.UUID, now time.Time) error {
	return deleteProjectRows(ctx, c.rows, c.batch, projectID, by, now)
}

// DeleteWorkspaceIssues soft-deletes the work-item-side rows of each of the
// projects, in their order: those deleteWorkspace locked and deletes.
func (c *Cascade) DeleteWorkspaceIssues(ctx context.Context, projectIDs []uuid.UUID, by uuid.UUID, now time.Time) error {
	for _, project := range projectIDs {
		if err := deleteProjectRows(ctx, c.rows, c.batch, project, by, now); err != nil {
			return err
		}
	}
	return nil
}
````

`server/internal/modules/issue/app/delete_project_rows.go`（新文件，56 行）：

````file server/internal/modules/issue/app/delete_project_rows.go
// Package app holds the issue module's use cases and the ports they need
// (M4 design 6.2).
package app

import (
	"context"
	"slices"
	"time"
	"uuid"
)

// RowBatch is one batch of a deleted project's work-item-side rows: at
// most Limit of its undeleted rows whose ids are greater than After, in id
// order, soft-deleted at At by By (M4 design 3.6).
type RowBatch struct {
	ProjectID, After uuid.UUID
	Limit            int
	By               uuid.UUID
	At               time.Time
}

// ProjectRowsDeleter soft-deletes a deleted project's rows of one table a
// batch at a time, in the transaction ctx carries; each method locks the
// batch's rows FOR NO KEY UPDATE in id order before it writes them
// (convention 5) and returns the ids it deleted.
type ProjectRowsDeleter interface {
	DeleteProjectIssueRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
}

// rowSteps are d's statements in the global order of M4 design 3.5: the
// work items.
func rowSteps(d ProjectRowsDeleter) []func(context.Context, RowBatch) ([]uuid.UUID, error) {
	return []func(context.Context, RowBatch) ([]uuid.UUID, error){d.DeleteProjectIssueRows}
}

// deleteProjectRows soft-deletes every undeleted work-item-side row of the
// project at at by by, in the transaction ctx carries: table by table in
// the global order, each in batches of limit rows, the next batch after
// the greatest id of the last, until a batch finds none (M4 design 3.6).
// A failing batch comes back as itself, and nothing runs after it.
func deleteProjectRows(ctx context.Context, d ProjectRowsDeleter, limit int, projectID, by uuid.UUID, at time.Time) error {
	for _, step := range rowSteps(d) {
		b := RowBatch{ProjectID: projectID, Limit: limit, By: by, At: at}
		for {
			ids, err := step(ctx, b)
			if err != nil {
				return err
			}
			if len(ids) == 0 {
				break
			}
			b.After = slices.MaxFunc(ids, uuid.UUID.Compare)
		}
	}
	return nil
}
````

`server/internal/modules/issue/module.go`（新文件，44 行）：

````file server/internal/modules/issue/module.go
// Package issue is the work items module (M4 design 3.3, 6.2): it owns the
// work items' tables, and soft-deletes a deleted project's rows of them for
// the project module's deletions (its IssueCascade port).
package issue

import (
	"context"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// deleteBatch is the rows a statement of a project's deletion soft-deletes
// at most (M4 design 3.6).
const deleteBatch = 1000

// Cascade is what the project module's deletions ask of the work items:
// project's IssueCascade port (M4 design 3.3). Each method runs in the
// caller's transaction, which ctx carries, writes by and now into the rows
// it deletes, and returns a failure as itself, so the caller's whole write
// rolls back.
type Cascade interface {
	// DeleteProjectIssues soft-deletes the project's work-item-side rows.
	DeleteProjectIssues(ctx context.Context, projectID, by uuid.UUID, now time.Time) error
	// DeleteWorkspaceIssues soft-deletes the work-item-side rows of each
	// of the projects.
	DeleteWorkspaceIssues(ctx context.Context, projectIDs []uuid.UUID, by uuid.UUID, now time.Time) error
}

// Provided are the adapters issue offers the other modules. They depend on
// the pool alone, so bootstrap builds them before any module (M4 design
// 6.6).
type Provided struct {
	IssueCascade Cascade
}

// Provide builds issue's adapters for the other modules.
func Provide(pool *pgxpool.Pool) Provided {
	return Provided{IssueCascade: app.NewCascade(postgresadapter.New(pool), deleteBatch)}
}
````

- [ ] **Step 4: `project` 的 `IssueCascade` 端口和两个连带的最后一步**

`server/internal/modules/project/app/cascade.go`（修改，3 处）：

````old server/internal/modules/project/app/cascade.go
	enders   MembershipEnder
````
````new server/internal/modules/project/app/cascade.go
	enders   MembershipEnder
	issues   IssueCascade
````
````old server/internal/modules/project/app/cascade.go
// NewCascade returns the cascade over projects, locker, members and
// enders.
func NewCascade(projects ProjectsDeleter, locker WorkspaceProjectsLocker, members MemberDemoter, enders MembershipEnder) *Cascade {
	return &Cascade{projects: projects, locker: locker, members: members, enders: enders}
````
````new server/internal/modules/project/app/cascade.go
// NewCascade returns the cascade over projects, locker, members, enders
// and the issue module's cascade.
func NewCascade(projects ProjectsDeleter, locker WorkspaceProjectsLocker, members MemberDemoter, enders MembershipEnder,
	issues IssueCascade) *Cascade {
	return &Cascade{projects: projects, locker: locker, members: members, enders: enders, issues: issues}
````
````old server/internal/modules/project/app/cascade.go
// cascade, under its FOR NO KEY UPDATE of the workspace, each step one
// statement (convention 5), the steps deleteProject runs too
// (deleteProjects), at by and now.
func (c *Cascade) DeleteWorkspaceProjects(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return deleteProjects(ctx, c.projects, Deletion{WorkspaceID: workspaceID, By: by, Now: now})
````
````new server/internal/modules/project/app/cascade.go
// cascade, under its FOR NO KEY UPDATE of the workspace and of the projects
// (LockWorkspaceProjects), each step one statement (convention 5), the
// steps deleteProject runs too (deleteProjects), then the work-item side of
// the projects it locked, through the issue module (M4 design 3.5), all at
// by and now.
func (c *Cascade) DeleteWorkspaceProjects(ctx context.Context, workspaceID uuid.UUID, projectIDs []uuid.UUID, by uuid.UUID, now time.Time) error {
	if err := deleteProjects(ctx, c.projects, Deletion{WorkspaceID: workspaceID, By: by, Now: now}); err != nil {
		return err
	}
	return c.issues.DeleteWorkspaceIssues(ctx, projectIDs, by, now)
````

`server/internal/modules/project/app/cascade_test.go`（修改，7 处）：

````old server/internal/modules/project/app/cascade_test.go
// 3.6, each on every project of the caller's workspace, with the caller's
// account and moment, in the caller's transaction; a failing step comes
// back as itself and the steps after it do not run. Each step fails in
// turn, one row each of deletionSteps.
````
````new server/internal/modules/project/app/cascade_test.go
// 3.6, each on every project of the caller's workspace, then deletes the
// work-item side of the projects the caller locked through the issue
// module (M4 design 3.5), with the caller's account and moment, in the
// caller's transaction; a failing step comes back as itself and the steps
// after it do not run. Each step fails in turn.
````
````old server/internal/modules/project/app/cascade_test.go
	steps := deletionSteps(workspace, "*", by, clockNow)
````
````new server/internal/modules/project/app/cascade_test.go
	projects := []uuid.UUID{uuid.NewV7(), uuid.NewV7()}
	steps := append(deletionSteps(workspace, "*", by, clockNow),
		fmt.Sprintf("DeleteWorkspaceIssues %v by %s at %s", projects, by, clockNow.Format(timeFormat)))
````
````old server/internal/modules/project/app/cascade_test.go
		f.store.errs = map[string]error{tt.fail: errDisk}
````
````new server/internal/modules/project/app/cascade_test.go
		f.store.errs, f.issues.errs = map[string]error{tt.fail: errDisk}, map[string]error{tt.fail: errDisk}
````
````old server/internal/modules/project/app/cascade_test.go
		err := app.NewCascade(f.store, &fakeLocker{}, &fakeDemoter{}, &fakeEnder{}).DeleteWorkspaceProjects(inTx, workspace, by, clockNow)
````
````new server/internal/modules/project/app/cascade_test.go
		err := app.NewCascade(f.store, &fakeLocker{}, &fakeDemoter{}, &fakeEnder{}, f.issues).DeleteWorkspaceProjects(inTx, workspace, projects, by, clockNow)
````
````old server/internal/modules/project/app/cascade_test.go
		got, err := app.NewCascade(newWrites().store, f, &fakeDemoter{}, &fakeEnder{}).LockWorkspaceProjects(inTx, workspace)
````
````new server/internal/modules/project/app/cascade_test.go
		got, err := app.NewCascade(newWrites().store, f, &fakeDemoter{}, &fakeEnder{}, &fakeIssues{}).LockWorkspaceProjects(inTx, workspace)
````
````old server/internal/modules/project/app/cascade_test.go
		err := app.NewCascade(newWrites().store, &fakeLocker{}, f, &fakeEnder{}).DemoteToGuest(inTx, workspace, user, by, now)
````
````new server/internal/modules/project/app/cascade_test.go
		err := app.NewCascade(newWrites().store, &fakeLocker{}, f, &fakeEnder{}, &fakeIssues{}).DemoteToGuest(inTx, workspace, user, by, now)
````
````old server/internal/modules/project/app/cascade_test.go
		err := app.NewCascade(newWrites().store, &fakeLocker{}, &fakeDemoter{}, f).EndMemberships(inTx, workspaces, user, by, now)
````
````new server/internal/modules/project/app/cascade_test.go
		err := app.NewCascade(newWrites().store, &fakeLocker{}, &fakeDemoter{}, f, &fakeIssues{}).EndMemberships(inTx, workspaces, user, by, now)
````

`server/internal/modules/project/app/delete_project.go`（修改，4 处）：

````old server/internal/modules/project/app/delete_project.go
	projects ProjectsDeleter
````
````new server/internal/modules/project/app/delete_project.go
	projects ProjectsDeleter
	issues   IssueCascade
````
````old server/internal/modules/project/app/delete_project.go
func NewDeleteProject(projects ProjectsDeleter, locks Locks, tx shared.TxManager, clock Clock) *DeleteProject {
	return &DeleteProject{projects: projects, locks: locks, tx: tx, clock: clock}
````
````new server/internal/modules/project/app/delete_project.go
func NewDeleteProject(projects ProjectsDeleter, issues IssueCascade, locks Locks, tx shared.TxManager, clock Clock) *DeleteProject {
	return &DeleteProject{projects: projects, issues: issues, locks: locks, tx: tx, clock: clock}
````
````old server/internal/modules/project/app/delete_project.go
// the caller (deleteProjects). An archived project is deleted as any
````
````new server/internal/modules/project/app/delete_project.go
// the caller (deleteProjects), and last its work-item side, through the
// issue module (M4 design 3.5). An archived project is deleted as any
````
````old server/internal/modules/project/app/delete_project.go
		return deleteProjects(ctx, u.projects, Deletion{WorkspaceID: h.project.WorkspaceID, ProjectID: &id, By: actor.UserID, Now: u.clock.Now()})
````
````new server/internal/modules/project/app/delete_project.go
		now := u.clock.Now()
		if err := deleteProjects(ctx, u.projects, Deletion{WorkspaceID: h.project.WorkspaceID, ProjectID: &id, By: actor.UserID, Now: now}); err != nil {
			return err
		}
		return u.issues.DeleteProjectIssues(ctx, id, actor.UserID, now)
````

`server/internal/modules/project/app/delete_project_test.go`（修改，6 处）：

````old server/internal/modules/project/app/delete_project_test.go
	"errors"
````
````new server/internal/modules/project/app/delete_project_test.go
	"errors"
	"fmt"
````
````old server/internal/modules/project/app/delete_project_test.go
	return app.NewDeleteProject(f.store, f.locks(), f.tx, clockAt{clockNow, f.log}), f
````
````new server/internal/modules/project/app/delete_project_test.go
	return app.NewDeleteProject(f.store, f.issues, f.locks(), f.tx, clockAt{clockNow, f.log}), f
````
````old server/internal/modules/project/app/delete_project_test.go
// decision, the clock, the five steps on that project alone.
````
````new server/internal/modules/project/app/delete_project_test.go
// decision, the clock, the five steps on that project alone, then its
// work-item side.
````
````old server/internal/modules/project/app/delete_project_test.go
		deletionSteps(acme.ID, project.String(), bob, clockNow))
````
````new server/internal/modules/project/app/delete_project_test.go
		deletionSteps(acme.ID, project.String(), bob, clockNow),
		[]string{fmt.Sprintf("DeleteProjectIssues %s by %s at %s", project, bob, clockNow.Format(timeFormat))})
````
````old server/internal/modules/project/app/delete_project_test.go
// lock read, by the caller at that one moment. An archived project is
````
````new server/internal/modules/project/app/delete_project_test.go
// lock read, and last deletes its work-item side through the issue module
// (M4 design 3.5), by the caller at that one moment. An archived project is
````
````old server/internal/modules/project/app/delete_project_test.go
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 11},
````
````new server/internal/modules/project/app/delete_project_test.go
		{"the work items", func(f *writeFixture) { f.issues.errs = map[string]error{"DeleteProjectIssues": errDisk} }, 12},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 12},
````

`server/internal/modules/project/app/fakes_issue_test.go`（新文件，33 行）：

````file server/internal/modules/project/app/fakes_issue_test.go
package app_test

import (
	"context"
	"fmt"
	"time"
	"uuid"
)

// fakeIssues is the issue module's cascade (app.IssueCascade): it logs each
// call with its arguments, and " outside tx" when it ran outside the
// caller's transaction (callLog), and fails the method errs names, wrapped
// as the module wraps it.
type fakeIssues struct {
	log  *callLog
	errs map[string]error
}

func (f *fakeIssues) call(ctx context.Context, name, format string, args ...any) error {
	f.log.add(ctx, name+" "+format, args...)
	if err := f.errs[name]; err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func (f *fakeIssues) DeleteProjectIssues(ctx context.Context, projectID, by uuid.UUID, now time.Time) error {
	return f.call(ctx, "DeleteProjectIssues", "%s by %s at %s", projectID, by, now.Format(timeFormat))
}

func (f *fakeIssues) DeleteWorkspaceIssues(ctx context.Context, projectIDs []uuid.UUID, by uuid.UUID, now time.Time) error {
	return f.call(ctx, "DeleteWorkspaceIssues", "%v by %s at %s", projectIDs, by, now.Format(timeFormat))
}
````

`server/internal/modules/project/app/fakes_write_test.go`（修改，2 处）：

````old server/internal/modules/project/app/fakes_write_test.go
	auth       *fakeAuthorizer
````
````new server/internal/modules/project/app/fakes_write_test.go
	auth       *fakeAuthorizer
	issues     *fakeIssues
````
````old server/internal/modules/project/app/fakes_write_test.go
	return &writeFixture{log: log, tx: &fakeTx{log: log}, workspaces: &fakeWorkspaces{log: log}, members: &fakeMembers{log: log},
````
````new server/internal/modules/project/app/fakes_write_test.go
	return &writeFixture{log: log, tx: &fakeTx{log: log}, workspaces: &fakeWorkspaces{log: log}, members: &fakeMembers{log: log},
		issues: &fakeIssues{log: log},
````

`server/internal/modules/project/app/issue_ports.go`（新文件，23 行）：

````file server/internal/modules/project/app/issue_ports.go
package app

import (
	"context"
	"time"
	"uuid"
)

// IssueCascade is what the project's deletions ask of the work items (M4
// design 3.3): the issue module implements it (issue.Provide). Each method
// runs in the transaction ctx carries, which the caller began and holds
// the project's or the workspace's lock in, writes by and now into the
// rows it deletes, and returns a failure as itself, so the caller's whole
// write rolls back.
type IssueCascade interface {
	// DeleteProjectIssues soft-deletes the project's work-item-side rows:
	// deleteProject's last step.
	DeleteProjectIssues(ctx context.Context, projectID, by uuid.UUID, now time.Time) error
	// DeleteWorkspaceIssues soft-deletes the work-item-side rows of each of
	// the projects: deleteWorkspace's last step, on the projects it locked
	// (LockWorkspaceProjects).
	DeleteWorkspaceIssues(ctx context.Context, projectIDs []uuid.UUID, by uuid.UUID, now time.Time) error
}
````

`server/internal/modules/project/module.go`（修改，6 处）：

````old server/internal/modules/project/module.go
	// rows under them: deleteWorkspace's last step.
	DeleteWorkspaceProjects(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error
````
````new server/internal/modules/project/module.go
	// rows under them, the work-item side of projectIDs, those it locked,
	// too: deleteWorkspace's last step.
	DeleteWorkspaceProjects(ctx context.Context, workspaceID uuid.UUID, projectIDs []uuid.UUID, by uuid.UUID, now time.Time) error
````
````old server/internal/modules/project/module.go
	Pool *pgxpool.Pool
````
````new server/internal/modules/project/module.go
	Pool *pgxpool.Pool
	// Issues is issue's IssueCascade (issue.Provide): deleting a
	// workspace's projects deletes their work-item side (M4 design 6.6).
	Issues app.IssueCascade
````
````old server/internal/modules/project/module.go
	return app.NewCascade(store, store, store, store)
````
````new server/internal/modules/project/module.go
	return app.NewCascade(store, store, store, store, d.Issues)
````
````old server/internal/modules/project/module.go
	Members app.WorkspaceMembers
````
````new server/internal/modules/project/module.go
	Members app.WorkspaceMembers
	// Issues is issue's IssueCascade (issue.Provide).
	Issues app.IssueCascade
````
````old server/internal/modules/project/module.go
	return &Module{cascade: NewCascade(CascadeDeps{Pool: d.Pool}), uc: httpadapter.UseCases{
````
````new server/internal/modules/project/module.go
	return &Module{cascade: NewCascade(CascadeDeps{Pool: d.Pool, Issues: d.Issues}), uc: httpadapter.UseCases{
````
````old server/internal/modules/project/module.go
		DeleteProject:       app.NewDeleteProject(store, locks, d.Tx, d.Clock),
````
````new server/internal/modules/project/module.go
		DeleteProject:       app.NewDeleteProject(store, d.Issues, locks, d.Tx, d.Clock),
````

- [ ] **Step 5: `workspace` 把锁住的项目交给连带**

`server/internal/modules/workspace/adapter/postgres/delete_workspace_invitations_test.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/delete_workspace_invitations_test.go
func (noProjectRows) DeleteWorkspaceProjects(context.Context, uuid.UUID, uuid.UUID, time.Time) error {
````
````new server/internal/modules/workspace/adapter/postgres/delete_workspace_invitations_test.go
func (noProjectRows) DeleteWorkspaceProjects(context.Context, uuid.UUID, []uuid.UUID, uuid.UUID, time.Time) error {
````

`server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
func (noProjects) DeleteWorkspaceProjects(context.Context, uuid.UUID, uuid.UUID, time.Time) error {
````
````new server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
func (noProjects) DeleteWorkspaceProjects(context.Context, uuid.UUID, []uuid.UUID, uuid.UUID, time.Time) error {
````

`server/internal/modules/workspace/app/delete_workspace.go`（修改，4 处）：

````old server/internal/modules/workspace/app/delete_workspace.go
// settings, and last the projects and the rows under them, through
// ProjectCascade (M3 design 3.3). A failing step comes back as itself and
// the steps after it do not run.
func (u *DeleteWorkspace) cascade(ctx context.Context, id uuid.UUID, invitations []uuid.UUID, by uuid.UUID, now time.Time) error {
````
````new server/internal/modules/workspace/app/delete_workspace.go
// settings, and last the projects and the rows under them, the work-item
// side of the projects it locked too, through ProjectCascade (M3 design
// 3.3, M4 design 3.5). A failing step comes back as itself and the steps
// after it do not run.
func (u *DeleteWorkspace) cascade(ctx context.Context, id uuid.UUID, invitations, projects []uuid.UUID, by uuid.UUID, now time.Time) error {
````
````old server/internal/modules/workspace/app/delete_workspace.go
		func() error { return u.projects.DeleteWorkspaceProjects(ctx, id, by, now) },
````
````new server/internal/modules/workspace/app/delete_workspace.go
		func() error { return u.projects.DeleteWorkspaceProjects(ctx, id, projects, by, now) },
````
````old server/internal/modules/workspace/app/delete_workspace.go
		if _, err := u.projects.LockWorkspaceProjects(ctx, id); err != nil {
````
````new server/internal/modules/workspace/app/delete_workspace.go
		projects, err := u.projects.LockWorkspaceProjects(ctx, id)
		if err != nil {
````
````old server/internal/modules/workspace/app/delete_workspace.go
		if err := u.cascade(ctx, id, invitations, actor.UserID, u.clock.Now()); err != nil {
````
````new server/internal/modules/workspace/app/delete_workspace.go
		if err := u.cascade(ctx, id, invitations, projects, actor.UserID, u.clock.Now()); err != nil {
````

`server/internal/modules/workspace/app/delete_workspace_test.go`（修改，3 处）：

````old server/internal/modules/workspace/app/delete_workspace_test.go
// through the project module.
````
````new server/internal/modules/workspace/app/delete_workspace_test.go
// through the project module, with the projects locked.
````
````old server/internal/modules/workspace/app/delete_workspace_test.go
		if step == "DeleteInvitations" {
````
````new server/internal/modules/workspace/app/delete_workspace_test.go
		switch step {
		case "DeleteInvitations":
````
````old server/internal/modules/workspace/app/delete_workspace_test.go
			continue
		}
		calls = append(calls, step+" "+w.ID.String()+at)
````
````new server/internal/modules/workspace/app/delete_workspace_test.go
		case "DeleteWorkspaceProjects":
			calls = append(calls, fmt.Sprintf("%s %s %v%s", step, w.ID, lockedProjects, at))
		default:
			calls = append(calls, step+" "+w.ID.String()+at)
		}
````

`server/internal/modules/workspace/app/fakes_projects_test.go`（修改，1 处）：

````old server/internal/modules/workspace/app/fakes_projects_test.go
func (f *fakeProjects) DeleteWorkspaceProjects(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "DeleteWorkspaceProjects %s by %s at %s", workspaceID, by, now.Format(time.RFC3339Nano))
````
````new server/internal/modules/workspace/app/fakes_projects_test.go
func (f *fakeProjects) DeleteWorkspaceProjects(ctx context.Context, workspaceID uuid.UUID, projectIDs []uuid.UUID, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "DeleteWorkspaceProjects %s %v by %s at %s", workspaceID, projectIDs, by, now.Format(time.RFC3339Nano))
````

`server/internal/modules/workspace/app/ports.go`（修改，1 处）：

````old server/internal/modules/workspace/app/ports.go
	// rows under them, at the moment and by the account of the workspace's
	// deletion: its last step.
	DeleteWorkspaceProjects(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error
````
````new server/internal/modules/workspace/app/ports.go
	// rows under them, the work-item side of projectIDs, those
	// LockWorkspaceProjects locked, too, at the moment and by the account of
	// the workspace's deletion: its last step.
	DeleteWorkspaceProjects(ctx context.Context, workspaceID uuid.UUID, projectIDs []uuid.UUID, by uuid.UUID, now time.Time) error
````

- [ ] **Step 6: 组合和它的测试**

`server/internal/archtest/composition_test.go`（修改，3 处）：

````old server/internal/archtest/composition_test.go
// and project's NewCascade, never workspace's NewAdmin; Workspaces,
// workspace's NewAdmin, never the deactivation's two, whose cascade no
// command of it calls. The rule follows the static calls from each; the
// commands are func values it calls dynamically, so they are not followed:
// they only receive the composition. Reaching what each builds shows the
// walk sees the composition at all.
````
````new server/internal/archtest/composition_test.go
// and project's NewCascade over issue's Provide (M4 design 6.6), never
// workspace's NewAdmin; Workspaces, workspace's NewAdmin, never the
// deactivation's three, whose cascade no command of it calls. The rule
// follows the static calls from each; the commands are func values it
// calls dynamically, so they are not followed: they only receive the
// composition. Reaching what each builds shows the walk sees the
// composition at all.
````
````old server/internal/archtest/composition_test.go
	identity, workspace, project := m("internal/modules/identity"), m("internal/modules/workspace"), m("internal/modules/project")
````
````new server/internal/archtest/composition_test.go
	identity, workspace, project := m("internal/modules/identity"), m("internal/modules/workspace"), m("internal/modules/project")
	issue := m("internal/modules/issue")
````
````old server/internal/archtest/composition_test.go
		{"Users", []string{identity + ".NewAdmin", workspace + ".NewDeactivator", project + ".NewCascade"}, []string{workspace + ".NewAdmin"}},
		{"Workspaces", []string{workspace + ".NewAdmin"}, []string{workspace + ".NewDeactivator", project + ".NewCascade"}},
````
````new server/internal/archtest/composition_test.go
		{"Users", []string{identity + ".NewAdmin", workspace + ".NewDeactivator", project + ".NewCascade", issue + ".Provide"},
			[]string{workspace + ".NewAdmin"}},
		{"Workspaces", []string{workspace + ".NewAdmin"}, []string{workspace + ".NewDeactivator", project + ".NewCascade", issue + ".Provide"}},
````

`server/internal/bootstrap/actions_test.go`（修改，1 处）：

````old server/internal/bootstrap/actions_test.go
	"access",
````
````new server/internal/bootstrap/actions_test.go
	"access",
	// It answers no operation: the project module's deletions call it, and
	// no rule decides that.
	"issue",
````

`server/internal/bootstrap/app.go`（修改，3 处）：

````old server/internal/bootstrap/app.go
	"github.com/open-nerve/NerveProject/server/internal/modules/instance"
````
````new server/internal/bootstrap/app.go
	"github.com/open-nerve/NerveProject/server/internal/modules/instance"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
````
````old server/internal/bootstrap/app.go
	projectPorts := project.Provide(pool)
````
````new server/internal/bootstrap/app.go
	projectPorts := project.Provide(pool)
	issuePorts := issue.Provide(pool)
````
````old server/internal/bootstrap/app.go
		Members:    workspacePorts.WorkspaceMembers,
````
````new server/internal/bootstrap/app.go
		Members:    workspacePorts.WorkspaceMembers,
		Issues:     issuePorts.IssueCascade,
````

`server/internal/bootstrap/deactivating_test.go`（修改，2 处）：

````old server/internal/bootstrap/deactivating_test.go
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
````
````new server/internal/bootstrap/deactivating_test.go
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
````
````old server/internal/bootstrap/deactivating_test.go
		Memberships: workspaceapp.NewDeactivator(memberships, project.NewCascade(project.CascadeDeps{Pool: pool}), clock.System{}),
````
````new server/internal/bootstrap/deactivating_test.go
		Memberships: workspaceapp.NewDeactivator(memberships, cascadeOn(pool), clock.System{}),
````

`server/internal/bootstrap/ending_connection_test.go`（修改，3 处）：

````old server/internal/bootstrap/ending_connection_test.go
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
````
````new server/internal/bootstrap/ending_connection_test.go
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
````
````old server/internal/bootstrap/ending_connection_test.go
// Accounts and MemberProfiles, converted, and the project module's cascade,
// each on pool.
````
````new server/internal/bootstrap/ending_connection_test.go
// Accounts and MemberProfiles, converted, and the project module's cascade
// over issue's, each on pool.
````
````old server/internal/bootstrap/ending_connection_test.go
		Workspaces: projectWorkspaces{directory: provided.WorkspaceDirectory}, Members: provided.WorkspaceMembers})
````
````new server/internal/bootstrap/ending_connection_test.go
		Workspaces: projectWorkspaces{directory: provided.WorkspaceDirectory}, Members: provided.WorkspaceMembers,
		Issues: issue.Provide(pool).IssueCascade})
````

`server/internal/bootstrap/interleaving_accepting_test.go`（修改，2 处）：

````old server/internal/bootstrap/interleaving_accepting_test.go
	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
````
````new server/internal/bootstrap/interleaving_accepting_test.go
	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
````
````old server/internal/bootstrap/interleaving_accepting_test.go
			cascade := project.NewCascade(project.CascadeDeps{Pool: r.pool})
````
````new server/internal/bootstrap/interleaving_accepting_test.go
			cascade := cascadeOn(r.pool)
````

`server/internal/bootstrap/interleaving_answers_test.go`（修改，2 处）：

````old server/internal/bootstrap/interleaving_answers_test.go
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
````
````new server/internal/bootstrap/interleaving_answers_test.go
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
````
````old server/internal/bootstrap/interleaving_answers_test.go
	return project.NewCascade(project.CascadeDeps{Pool: r.pool})
````
````new server/internal/bootstrap/interleaving_answers_test.go
	return cascadeOn(r.pool)
````

`server/internal/bootstrap/interleaving_deletion_test.go`（修改，2 处）：

````old server/internal/bootstrap/interleaving_deletion_test.go
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
````
````new server/internal/bootstrap/interleaving_deletion_test.go
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
````
````old server/internal/bootstrap/interleaving_deletion_test.go
	return workspaceapp.NewDeleteWorkspace(workspaces, project.NewCascade(project.CascadeDeps{Pool: r.pool}), authorizerOn(r.pool),
````
````new server/internal/bootstrap/interleaving_deletion_test.go
	return workspaceapp.NewDeleteWorkspace(workspaces, cascadeOn(r.pool), authorizerOn(r.pool),
````

`server/internal/bootstrap/interleaving_endings_test.go`（修改，3 处）：

````old server/internal/bootstrap/interleaving_endings_test.go
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
````
````new server/internal/bootstrap/interleaving_endings_test.go
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
````
````old server/internal/bootstrap/interleaving_endings_test.go
		project.NewCascade(project.CascadeDeps{Pool: r.pool}), authorizerOn(r.pool), postgres.NewTxManager(r.pool, 2*time.Second), clock.System{}).
````
````new server/internal/bootstrap/interleaving_endings_test.go
		cascadeOn(r.pool), authorizerOn(r.pool), postgres.NewTxManager(r.pool, 2*time.Second), clock.System{}).
````
````old server/internal/bootstrap/interleaving_endings_test.go
		project.NewCascade(project.CascadeDeps{Pool: r.pool}), r.authorizer(), postgres.NewTxManager(r.pool, 2*time.Second), clock.System{}).
````
````new server/internal/bootstrap/interleaving_endings_test.go
		cascadeOn(r.pool), r.authorizer(), postgres.NewTxManager(r.pool, 2*time.Second), clock.System{}).
````

`server/internal/bootstrap/interleaving_growth_test.go`（修改，5 处）：

````old server/internal/bootstrap/interleaving_growth_test.go
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
````
````new server/internal/bootstrap/interleaving_growth_test.go
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
````
````old server/internal/bootstrap/interleaving_growth_test.go
		ProjectAccess: accessProjects{projects: project.Provide(pool).ProjectAccess}})
````
````new server/internal/bootstrap/interleaving_growth_test.go
		ProjectAccess: accessProjects{projects: project.Provide(pool).ProjectAccess}})
}

// cascadeOn is project's cascade as bootstrap wires it, on pool, over
// issue's (M4 design 6.6).
func cascadeOn(pool *pgxpool.Pool) project.Cascade {
	return project.NewCascade(project.CascadeDeps{Pool: pool, Issues: issue.Provide(pool).IssueCascade})
````
````old server/internal/bootstrap/interleaving_growth_test.go
					g, cascade := newGate(), project.NewCascade(project.CascadeDeps{Pool: r.pool})
````
````new server/internal/bootstrap/interleaving_growth_test.go
					g, cascade := newGate(), cascadeOn(r.pool)
````
````old server/internal/bootstrap/interleaving_growth_test.go
			auth, cascade := r.authorizer(), projectapp.NewCascade(store, store, store, store)
````
````new server/internal/bootstrap/interleaving_growth_test.go
			auth, cascade := r.authorizer(), projectapp.NewCascade(store, store, store, store, issue.Provide(r.pool).IssueCascade)
````
````old server/internal/bootstrap/interleaving_growth_test.go
				cascade = projectapp.NewCascade(store, store, gatedDemoter{store, g}, store)
````
````new server/internal/bootstrap/interleaving_growth_test.go
				cascade = projectapp.NewCascade(store, store, gatedDemoter{store, g}, store, issue.Provide(r.pool).IssueCascade)
````

`server/internal/bootstrap/interleaving_leaving_test.go`（修改，2 处）：

````old server/internal/bootstrap/interleaving_leaving_test.go
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
````
````new server/internal/bootstrap/interleaving_leaving_test.go
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
````
````old server/internal/bootstrap/interleaving_leaving_test.go
		project.NewCascade(project.CascadeDeps{Pool: w.pool}), authorizerOn(w.pool), w.tx(), clock.System{}).
````
````new server/internal/bootstrap/interleaving_leaving_test.go
		cascadeOn(w.pool), authorizerOn(w.pool), w.tx(), clock.System{}).
````

`server/internal/bootstrap/interleaving_removal_test.go`（修改，2 处）：

````old server/internal/bootstrap/interleaving_removal_test.go
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
````
````new server/internal/bootstrap/interleaving_removal_test.go
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
````
````old server/internal/bootstrap/interleaving_removal_test.go
		project.NewCascade(project.CascadeDeps{Pool: r.pool}), authorizerOn(r.pool), postgres.NewTxManager(r.pool, 2*time.Second), clock.System{}).
````
````new server/internal/bootstrap/interleaving_removal_test.go
		cascadeOn(r.pool), authorizerOn(r.pool), postgres.NewTxManager(r.pool, 2*time.Second), clock.System{}).
````

`server/internal/bootstrap/interleaving_roles_test.go`（修改，2 处）：

````old server/internal/bootstrap/interleaving_roles_test.go
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
````
````new server/internal/bootstrap/interleaving_roles_test.go
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
````
````old server/internal/bootstrap/interleaving_roles_test.go
	return workspaceapp.NewUpdateWorkspaceMember(members, project.NewCascade(project.CascadeDeps{Pool: r.pool}),
````
````new server/internal/bootstrap/interleaving_roles_test.go
	return workspaceapp.NewUpdateWorkspaceMember(members, cascadeOn(r.pool),
````

`server/internal/bootstrap/interleaving_writes_test.go`（修改，3 处）：

````old server/internal/bootstrap/interleaving_writes_test.go
	"uuid"

````
````new server/internal/bootstrap/interleaving_writes_test.go
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
````
````old server/internal/bootstrap/interleaving_writes_test.go
						return r.demote(ctx, workspacepg.New(r.pool), projectapp.NewCascade(store, store, store, store))
````
````new server/internal/bootstrap/interleaving_writes_test.go
						return r.demote(ctx, workspacepg.New(r.pool), projectapp.NewCascade(store, store, store, store, issue.Provide(r.pool).IssueCascade))
````
````old server/internal/bootstrap/interleaving_writes_test.go
						return r.demote(ctx, workspacepg.New(r.pool), projectapp.NewCascade(store, store, gatedDemoter{store, g}, store))
````
````new server/internal/bootstrap/interleaving_writes_test.go
						return r.demote(ctx, workspacepg.New(r.pool), projectapp.NewCascade(store, store, gatedDemoter{store, g}, store, issue.Provide(r.pool).IssueCascade))
````

`server/internal/bootstrap/project_connection_test.go`（修改，3 处）：

````old server/internal/bootstrap/project_connection_test.go
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
````
````new server/internal/bootstrap/project_connection_test.go
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
````
````old server/internal/bootstrap/project_connection_test.go
// its Authorizer and members as its WorkspaceMembers.
````
````new server/internal/bootstrap/project_connection_test.go
// its Authorizer, members as its WorkspaceMembers and issue's cascade.
````
````old server/internal/bootstrap/project_connection_test.go
		Workspaces: projectWorkspaces{directory: workspace.Provide(pool).WorkspaceDirectory}, Members: members}).Register)
````
````new server/internal/bootstrap/project_connection_test.go
		Workspaces: projectWorkspaces{directory: workspace.Provide(pool).WorkspaceDirectory}, Members: members,
		Issues: issue.Provide(pool).IssueCascade}).Register)
````

`server/internal/bootstrap/project_deletion_test.go`（修改，3 处）：

````old server/internal/bootstrap/project_deletion_test.go
// settings in it and its states, through createProject, and a label of
// hers, Bug, with its child, UI, written directly (seedLabels).
````
````new server/internal/bootstrap/project_deletion_test.go
// settings in it and its states, through createProject, a label of hers,
// Bug, with its child, UI, and a work item of hers, written directly
// (seedLabels, seedIssues).
````
````old server/internal/bootstrap/project_deletion_test.go
	seedLabels(t, pool, web, aliceID)
	seedLabels(t, pool, ops, aliceID)
````
````new server/internal/bootstrap/project_deletion_test.go
	for _, project := range []uuid.UUID{web, ops} {
		seedLabels(t, pool, project, aliceID)
		seedIssues(t, pool, project, aliceID)
	}
````
````old server/internal/bootstrap/project_deletion_test.go
		t.Fatalf("seeding the labels of %s: %d rows, %v; want 2", project, tag.RowsAffected(), err)
````
````new server/internal/bootstrap/project_deletion_test.go
		t.Fatalf("seeding the labels of %s: %d rows, %v; want 2", project, tag.RowsAffected(), err)
	}
}

// seedIssues writes a work item of project in its default state, by by,
// directly: the seed does not depend on which of the issue module's writes
// exist. Its number is the project's counter, moved on, as a creation
// moves it.
func seedIssues(t *testing.T, pool *pgxpool.Pool, project, by uuid.UUID) {
	t.Helper()
	issue := uuid.NewV7()
	for _, stmt := range []struct {
		sql  string
		args []any
	}{
		{`UPDATE projects SET last_issue_sequence = last_issue_sequence + 1 WHERE id = $1`, []any{project}},
		{`INSERT INTO issues (id, workspace_id, project_id, state_id, name, sequence_id, created_by_id, updated_by_id)
			SELECT $1, p.workspace_id, p.id, s.id, 'Seeded', p.last_issue_sequence, $3, $3
			FROM projects p JOIN states s ON s.project_id = p.id AND s."default" AND s.deleted_at IS NULL WHERE p.id = $2`,
			[]any{issue, project, by}},
	} {
		if tag, err := pool.Exec(pgtest.Soon(t), stmt.sql, stmt.args...); err != nil || tag.RowsAffected() != 1 {
			t.Fatalf("seeding the work items of %s: %s: %d rows, %v; want 1", project, stmt.sql, tag.RowsAffected(), err)
		}
````

`server/internal/bootstrap/users.go`（修改，3 处）：

````old server/internal/bootstrap/users.go
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
````
````new server/internal/bootstrap/users.go
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
````
````old server/internal/bootstrap/users.go
// Deactivator over project's cascade, each built on the pool alone (M3
// design 6.6); no HTTP server, no Authorizer, no jobs client. The
````
````new server/internal/bootstrap/users.go
// Deactivator over project's cascade, itself over issue's, each built on
// the pool alone (M3 design 6.6, M4 design 6.6); no HTTP server, no
// Authorizer, no jobs client. The
````
````old server/internal/bootstrap/users.go
			Pool: pool, Clock: clock.System{}, Projects: project.NewCascade(project.CascadeDeps{Pool: pool}),
````
````new server/internal/bootstrap/users.go
			Pool: pool, Clock: clock.System{}, Projects: project.NewCascade(project.CascadeDeps{Pool: pool, Issues: issue.Provide(pool).IssueCascade}),
````

`server/internal/bootstrap/workspace_deletion_test.go`（修改，3 处）：

````old server/internal/bootstrap/workspace_deletion_test.go
// display settings in it, a state and two labels (seedProject). A phase
// that adds a table under workspaces seeds a row of it here. It returns the
// workspace's id.
````
````new server/internal/bootstrap/workspace_deletion_test.go
// display settings in it, a state, two labels and a work item
// (seedProject). A phase that adds a table under workspaces seeds a row of
// it here. It returns the workspace's id.
````
````old server/internal/bootstrap/workspace_deletion_test.go
// with member as its admin, his display settings in it, one state, and a
// label with its child by member (seedLabels), directly: the seed does not
// depend on which of the project module's writes exist, nor on what they
// write besides. member, a project admin who is a member of the workspace
// (PA in M3 design 9.2), may write the project's labels. No row under the
````
````new server/internal/bootstrap/workspace_deletion_test.go
// with member as its admin, his display settings in it, one state, a label
// with its child and a work item by member (seedLabels, seedIssues),
// directly: the seed does not depend on which of the project module's
// writes exist, nor on what they write besides. member, a project admin
// who is a member of the workspace (PA in M3 design 9.2), may write the
// project's labels. No row under the
````
````old server/internal/bootstrap/workspace_deletion_test.go
	seedLabels(t, pool, project, member)
````
````new server/internal/bootstrap/workspace_deletion_test.go
	seedLabels(t, pool, project, member)
	seedIssues(t, pool, project, member)
````

- [ ] **Step 7: 端到端的过渡**

`e2e/fixtures/assert/project.ts`（修改，3 处）：

````old e2e/fixtures/assert/project.ts
 * by its project_id. Each table has such a row, and none is left undeleted; a row deleted before the project, as a
 * label a story deleted, is not counted.
````
````new e2e/fixtures/assert/project.ts
 * by its project_id. Each table but the work items' has such a row, and none is left undeleted; a row deleted before
 * the project, as a label a story deleted, is not counted.
````
````old e2e/fixtures/assert/project.ts
    })
  );
````
````new e2e/fixtures/assert/project.ts
    })
  );
  // No story writes the work items' tables yet (M4 P1a spec, section 3): they have no row to be deleted with the
  // project, and none left undeleted either.
  const withoutRows = new Set(["issues"]);
````
````old e2e/fixtures/assert/project.ts
    tables.map(({ name }) => ({ table: name, deletedWithIt: true, other: 0 }))
````
````new e2e/fixtures/assert/project.ts
    tables.map(({ name }) => ({ table: name, deletedWithIt: !withoutRows.has(name), other: 0 }))
````

- [ ] **Step 8: 测试和 lint**

Run: `go -C server test -count=1 ./migrations/ ./internal/modules/issue/... ./internal/modules/project/... ./internal/modules/workspace/...`
Expected: 全部 `ok`（`issue` 的包在 Task 3 起才有测试，这里是 `no test files`）。

Run: `go -C server test -count=1 -run 'TestDeletingAProjectLeavesNoUndeletedRowUnderIt$|TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt$|TestAProjectDeletionRefusedAtItsCommitChangesNoRow$|TestADeletionRefusedAtItsCommitChangesNoRow$' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过；关键词守卫 `keywords: 72 rules, 23 exceptions, no hits.`

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 120 个全部通过（P4、W3 删除项目、工作区时，`expectProjectDeleted` 要求 `issues` 没有行）。

- [ ] **Step 9: 提交**

```bash
git add deploy/runtime-grants.sql e2e/fixtures/assert/project.ts server/internal/archtest/composition_test.go server/internal/bootstrap/actions_test.go server/internal/bootstrap/app.go server/internal/bootstrap/deactivating_test.go server/internal/bootstrap/ending_connection_test.go server/internal/bootstrap/interleaving_accepting_test.go server/internal/bootstrap/interleaving_answers_test.go server/internal/bootstrap/interleaving_deletion_test.go server/internal/bootstrap/interleaving_endings_test.go server/internal/bootstrap/interleaving_growth_test.go server/internal/bootstrap/interleaving_leaving_test.go server/internal/bootstrap/interleaving_removal_test.go server/internal/bootstrap/interleaving_roles_test.go server/internal/bootstrap/interleaving_writes_test.go server/internal/bootstrap/project_connection_test.go server/internal/bootstrap/project_deletion_test.go server/internal/bootstrap/users.go server/internal/bootstrap/workspace_deletion_test.go server/internal/modules/issue/adapter/postgres/delete_rows.go server/internal/modules/issue/adapter/postgres/queries/delete_rows.sql server/internal/modules/issue/adapter/postgres/store.go server/internal/modules/issue/app/cascade.go server/internal/modules/issue/app/delete_project_rows.go server/internal/modules/issue/module.go server/internal/modules/project/app/cascade.go server/internal/modules/project/app/cascade_test.go server/internal/modules/project/app/delete_project.go server/internal/modules/project/app/delete_project_test.go server/internal/modules/project/app/fakes_issue_test.go server/internal/modules/project/app/fakes_write_test.go server/internal/modules/project/app/issue_ports.go server/internal/modules/project/module.go server/internal/modules/workspace/adapter/postgres/delete_workspace_invitations_test.go server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go server/internal/modules/workspace/app/delete_workspace.go server/internal/modules/workspace/app/delete_workspace_test.go server/internal/modules/workspace/app/fakes_projects_test.go server/internal/modules/workspace/app/ports.go server/migrations/issue_schema_test.go server/migrations/schema_test.go server/migrations/sql/00015_issue_issues.sql server/sqlc.yaml server/internal/modules/issue/adapter/postgres/gen/db.go server/internal/modules/issue/adapter/postgres/gen/delete_rows.sql.go server/internal/modules/issue/adapter/postgres/gen/models.go
```
```bash
git commit -m "feat(M4/P1): the issues table, the issue module's skeleton, and the deletions of projects and workspaces take their work items

Migration 00015 adds issues: a project's work items, each numbered once
in its project for good, never its own parent, in a state. The issue
module offers the project module its IssueCascade: deleting a project,
and deleting a workspace, soft-delete the work items of each project
last, at the deletion's moment and by its account, in batches of a
thousand in id order, each batch locked as it comes. A workspace's
deletion hands the cascade the projects it locked. The end-to-end check
of a deleted project expects no work item until the stories can write
one.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p1a.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试；结构是迁移的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是关键词守卫；静态是 `make lint-web` 的文案和键的核对）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T2.1` | 删除项目留下它的工作项一侧 | `internal/modules/project/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAProject`、`TestAProjectDeletionRefused`（`make test`） | 单元；组合 |
| `T2.2` | 删除工作区的项目一步留下它们的工作项一侧 | `internal/modules/project/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkspace`、`TestAFailedProjectsStep`、`TestADeletionRefused`（`make test`）、故事 `w3-workspace-settings`（Task 14 起） | 单元；组合；端到端 |
| `T2.3` | 删除工作区不把锁住的项目交给 `project` 模块 | `internal/modules/workspace/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkspace`、`TestAFailedProjectsStep`、`TestADeletionRefused`（`make test`） | 单元；组合 |
| `T2.4` | 工作项的分批语句保留行的最后写者 | `internal/bootstrap` 的 `TestDeletingAProject`、`TestAProjectDeletionRefused`（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkspace`、`TestAFailedProjectsStep`、`TestADeletionRefused`（`make test`） | 组合 |
| `T2.5` | 授权文件漏掉工作项的表 | `internal/bootstrap` 的 `TestTheGrantsFileCoversEveryRelationAndFunction`、`TestTheRuntimeRoleServesWithTheGrantsFile`（`make test`） | 组合 |
| `T2.6` | 服务端的组合不给 `project` 模块工作项的连带 | `internal/bootstrap` 的 `TestDeletingAProject`、`TestAProjectDeletionRefused`（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkspace`、`TestAFailedProjectsStep`、`TestADeletionRefused`（`make test`） | 组合 |
| `T2.7` | 命令行的连带不经 `issue` 的建出 | `internal/archtest` 的 `TestCommandsComposeNoServerAndNoJobs`（`make test`） | 组合 |
| `T2.8` | 工作项的编号只在未删除的工作项中唯一 | `migrations` 的 `TestConstraintAndIndexNames`（`make test`） | 结构 |
| `T2.9` | 删除项目吞掉工作项一侧的失败 | `internal/modules/project/app` 的 全部测试（`make test`） | 单元 |
| `T2.10` | 删除工作区的项目一步吞掉工作项一侧的失败 | `internal/modules/project/app` 的 全部测试（`make test`） | 单元 |

---

### Task 3: 其余四张表进入两个连带

**Files:**
- Create: `server/internal/modules/issue/app/cascade_test.go`、`server/migrations/sql/00016_issue_issue_assignees.sql`、`server/migrations/sql/00017_issue_issue_labels.sql`、`server/migrations/sql/00018_issue_issue_subscribers.sql`、`server/migrations/sql/00019_issue_issue_activities.sql`
- Modify: `deploy/runtime-grants.sql`、`e2e/fixtures/assert/project.ts`、`server/internal/bootstrap/project_deletion_test.go`、`server/internal/bootstrap/workspace_deletion_test.go`、`server/internal/modules/issue/adapter/postgres/delete_rows.go`、`server/internal/modules/issue/adapter/postgres/queries/delete_rows.sql`、`server/internal/modules/issue/app/delete_project_rows.go`、`server/migrations/issue_schema_test.go`、`server/migrations/schema_test.go`、`server/sqlc.yaml`
- Generate: `server/internal/modules/issue/adapter/postgres/gen/delete_rows.sql.go`、`server/internal/modules/issue/adapter/postgres/gen/models.go`

**Interfaces:**（spec 2.4；M4 设计 3.5、3.6、4.3–4.5、4.11）
- 迁移：
  - `00016_issue_issue_assignees.sql`、`00017_issue_issue_labels.sql`、`00018_issue_issue_subscribers.sql`：各 10 列，每张一个部分唯一索引 `(issue_id, assignee_id | label_id | subscriber_id) WHERE deleted_at IS NULL`；
  - `00019_issue_issue_activities.sql`：17 列，`issue_activities_verb_check`，`issue_activities_issue_id_created_at_idx ON (issue_id, created_at, id)`。
  - 每张表带 `workspace_id`、`project_id`，外键 `ON DELETE CASCADE`；`server/sqlc.yaml`、授权加这四张表。
- `queries/delete_rows.sql`（整个文件）：五条语句，每张表一条，同一个形状。
- `delete_rows.go` 加四个方法；`ProjectRowsDeleter` 加四个方法；`rowSteps` 按全局顺序 `issues` → `issue_assignees` → `issue_labels` → `issue_subscribers` → `issue_activities`。
- 端到端的过渡扩到五张表。

**Tests:**
- 迁移：`TestMigrationsGoUpDownAndUpAgain`（19 个迁移，五张表）；`TestConstraintAndIndexNames`（`issueNames` 加四张表的名字）。
- 单元（`issue/app/cascade_test.go`，假实现 `fakeRows`：每批回答的 `id` 最大的在前，取最后一个做下一批的起点的调用方会重读删过的行）：
  - `TestTheCascadeDeletesEachProjectsRowsInBatches`：`limit` 2；五条语句的次序；每批从上一批的最大 `id` 之后；一批为空时结束；另一个项目的行不动；都在调用方的事务里，同一时刻、同一个删除者；`DeleteWorkspaceIssues` 按给出的项目依次进行；
  - `TestTheCascadeStopsAtAFailingBatch`：失败的一批原样返回，之后不再有批次。
- 组合：
  - 按目录的删除测试：`seedIssues` 给工作项下的四张表各写一行（负责人、标签 Bug、关注、动态的第一行）；
  - 删除工作区的种子是两个项目 Web、Ops（工作项一侧按项目逐个删除）；
  - `TestAFailedProjectsStepRollsTheDeletionBack`：请求期间把 `issue_activities` 改名，连带的最后一条语句失败，500，之前的步骤都回滚。

- [ ] **Step 1: 四个迁移、sqlc 配置、授权和迁移的测试**

`deploy/runtime-grants.sql`（修改，1 处）：

````old deploy/runtime-grants.sql
GRANT SELECT, INSERT, UPDATE, DELETE ON issues TO nerve_runtime;
````
````new deploy/runtime-grants.sql
GRANT SELECT, INSERT, UPDATE, DELETE ON issues, issue_assignees, issue_labels, issue_subscribers, issue_activities TO nerve_runtime;
````

`server/migrations/issue_schema_test.go`（修改，1 处）：

````old server/migrations/issue_schema_test.go
// part of its want that the migration 00015 adds.
var issueNames = []string{
````
````new server/migrations/issue_schema_test.go
// part of its want that the migrations 00015–00019 add.
var issueNames = []string{
	"issue_activities_actor_id_fkey f n",
	"issue_activities_created_by_id_fkey f n",
	"issue_activities_issue_id_created_at_idx i",
	"issue_activities_issue_id_fkey f c",
	"issue_activities_pkey iu",
	"issue_activities_pkey p",
	"issue_activities_project_id_fkey f c",
	"issue_activities_project_id_idx i",
	"issue_activities_updated_by_id_fkey f n",
	"issue_activities_verb_check c",
	"issue_activities_workspace_id_fkey f c",
	"issue_activities_workspace_id_idx i",
	"issue_assignees_assignee_id_fkey f c",
	"issue_assignees_assignee_id_idx iw",
	"issue_assignees_created_by_id_fkey f n",
	"issue_assignees_issue_id_assignee_id_key iuw",
	"issue_assignees_issue_id_fkey f c",
	"issue_assignees_issue_id_idx i",
	"issue_assignees_pkey iu",
	"issue_assignees_pkey p",
	"issue_assignees_project_id_fkey f c",
	"issue_assignees_project_id_idx i",
	"issue_assignees_updated_by_id_fkey f n",
	"issue_assignees_workspace_id_fkey f c",
	"issue_assignees_workspace_id_idx i",
	"issue_labels_created_by_id_fkey f n",
	"issue_labels_issue_id_fkey f c",
	"issue_labels_issue_id_idx i",
	"issue_labels_issue_id_label_id_key iuw",
	"issue_labels_label_id_fkey f c",
	"issue_labels_label_id_idx i",
	"issue_labels_pkey iu",
	"issue_labels_pkey p",
	"issue_labels_project_id_fkey f c",
	"issue_labels_project_id_idx i",
	"issue_labels_updated_by_id_fkey f n",
	"issue_labels_workspace_id_fkey f c",
	"issue_labels_workspace_id_idx i",
	"issue_subscribers_created_by_id_fkey f n",
	"issue_subscribers_issue_id_fkey f c",
	"issue_subscribers_issue_id_idx i",
	"issue_subscribers_issue_id_subscriber_id_key iuw",
	"issue_subscribers_pkey iu",
	"issue_subscribers_pkey p",
	"issue_subscribers_project_id_fkey f c",
	"issue_subscribers_project_id_idx i",
	"issue_subscribers_subscriber_id_fkey f c",
	"issue_subscribers_subscriber_id_idx iw",
	"issue_subscribers_updated_by_id_fkey f n",
	"issue_subscribers_workspace_id_fkey f c",
	"issue_subscribers_workspace_id_idx i",
````

`server/migrations/schema_test.go`（修改，3 处）：

````old server/migrations/schema_test.go
	if err != nil || len(up) != 15 {
		t.Fatalf("Up() = %d migrations, %v; want 15", len(up), err)
````
````new server/migrations/schema_test.go
	if err != nil || len(up) != 19 {
		t.Fatalf("Up() = %d migrations, %v; want 19", len(up), err)
````
````old server/migrations/schema_test.go
		{tablesQuery, []string{"api_tokens", "auth_sessions", "issues", "labels", "profiles", "project_members", "project_user_properties", "projects",
			"river_job", "river_leader", "river_notification", "river_queue", "states", "users", "workspace_member_invites", "workspace_members",
			"workspace_user_properties", "workspaces"}},
````
````new server/migrations/schema_test.go
		{tablesQuery, []string{"api_tokens", "auth_sessions", "issue_activities", "issue_assignees", "issue_labels", "issue_subscribers", "issues",
			"labels", "profiles", "project_members", "project_user_properties", "projects", "river_job", "river_leader", "river_notification",
			"river_queue", "states", "users", "workspace_member_invites", "workspace_members", "workspace_user_properties", "workspaces"}},
````
````old server/migrations/schema_test.go
	if again, err := m.Up(ctx); err != nil || len(again) != 15 {
		t.Errorf("Up() again = %d migrations, %v; want 15", len(again), err)
````
````new server/migrations/schema_test.go
	if again, err := m.Up(ctx); err != nil || len(again) != 19 {
		t.Errorf("Up() again = %d migrations, %v; want 19", len(again), err)
````

`server/migrations/sql/00016_issue_issue_assignees.sql`（新文件，27 行）：

````file server/migrations/sql/00016_issue_issue_assignees.sql
-- issue_assignees：Plane 的 issue_assignees 表（10 列）按 M4 设计 4.3 照搬 10 列。
-- 负责人是项目的有效成员、角色不低于成员（3.7）；离开项目的人仍是已有工作项的负责人（3.6）。

-- +goose Up
CREATE TABLE issue_assignees (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    issue_id uuid NOT NULL REFERENCES issues ON DELETE CASCADE,
    -- 照搬；v0 不物理删除账户，不触发
    assignee_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE UNIQUE INDEX issue_assignees_issue_id_assignee_id_key ON issue_assignees (issue_id, assignee_id) WHERE deleted_at IS NULL;
-- 个人主页的"已分配"、按负责人筛选（4.3）
CREATE INDEX issue_assignees_assignee_id_idx ON issue_assignees (assignee_id) WHERE deleted_at IS NULL;
-- 物理级联按它们找子行（4 节开头）
CREATE INDEX issue_assignees_issue_id_idx ON issue_assignees (issue_id);
CREATE INDEX issue_assignees_workspace_id_idx ON issue_assignees (workspace_id);
CREATE INDEX issue_assignees_project_id_idx ON issue_assignees (project_id);

-- +goose Down
DROP TABLE issue_assignees;
````

`server/migrations/sql/00017_issue_issue_labels.sql`（新文件，25 行）：

````file server/migrations/sql/00017_issue_issue_labels.sql
-- issue_labels：Plane 的 issue_labels 表（10 列）按 M4 设计 4.4 照搬 10 列。
-- Plane 没有任何唯一约束，同一个标签可以重复；这里新加部分唯一（差异清单）。删除标签软删除它的行（3.6）。

-- +goose Up
CREATE TABLE issue_labels (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    issue_id uuid NOT NULL REFERENCES issues ON DELETE CASCADE,
    label_id uuid NOT NULL REFERENCES labels ON DELETE CASCADE,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE UNIQUE INDEX issue_labels_issue_id_label_id_key ON issue_labels (issue_id, label_id) WHERE deleted_at IS NULL;
-- 物理级联按它们找子行（4 节开头），标签的也是
CREATE INDEX issue_labels_issue_id_idx ON issue_labels (issue_id);
CREATE INDEX issue_labels_label_id_idx ON issue_labels (label_id);
CREATE INDEX issue_labels_workspace_id_idx ON issue_labels (workspace_id);
CREATE INDEX issue_labels_project_id_idx ON issue_labels (project_id);

-- +goose Down
DROP TABLE issue_labels;
````

`server/migrations/sql/00018_issue_issue_subscribers.sql`（新文件，27 行）：

````file server/migrations/sql/00018_issue_issue_subscribers.sql
-- issue_subscribers：Plane 的 issue_subscribers 表（10 列）按 M4 设计 4.5 照搬 10 列。
-- 写入只有自动关注（3.12）。

-- +goose Up
CREATE TABLE issue_subscribers (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    issue_id uuid NOT NULL REFERENCES issues ON DELETE CASCADE,
    -- 照搬；v0 不物理删除账户，不触发
    subscriber_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE UNIQUE INDEX issue_subscribers_issue_id_subscriber_id_key ON issue_subscribers (issue_id, subscriber_id) WHERE deleted_at IS NULL;
-- "已关注"、按关注者筛选（4.5）
CREATE INDEX issue_subscribers_subscriber_id_idx ON issue_subscribers (subscriber_id) WHERE deleted_at IS NULL;
-- 物理级联按它们找子行（4 节开头）
CREATE INDEX issue_subscribers_issue_id_idx ON issue_subscribers (issue_id);
CREATE INDEX issue_subscribers_workspace_id_idx ON issue_subscribers (workspace_id);
CREATE INDEX issue_subscribers_project_id_idx ON issue_subscribers (project_id);

-- +goose Down
DROP TABLE issue_subscribers;
````

`server/migrations/sql/00019_issue_issue_activities.sql`（新文件，35 行）：

````file server/migrations/sql/00019_issue_issue_activities.sql
-- issue_activities：Plane 的 issue_activities 表（20 列）按 M4 设计 4.11 保留 17 列。
-- 删除 issue_comment_id（评论不写动态）、epoch（只写不读）、attachments（从不写）（4.11）。

-- +goose Up
CREATE TABLE issue_activities (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    -- 这一行的工作项的项目（3.13）
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    -- Plane 可空、DO_NOTHING：动态随工作项删除（3.6）
    issue_id uuid NOT NULL REFERENCES issues ON DELETE CASCADE,
    verb varchar(255) NOT NULL DEFAULT 'created' CHECK (verb IN ('created', 'updated', 'deleted')),
    field varchar(255),
    old_value text,
    new_value text,
    -- 不是外键：指向状态、标签、账户、链接、工作项
    old_identifier uuid,
    new_identifier uuid,
    comment text NOT NULL DEFAULT '',
    -- 为空表示 nerve 自己（决策点 4）
    actor_id uuid REFERENCES users ON DELETE SET NULL,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
-- 列表的游标、找"最新的一行"、物理级联（4.11），所以不带条件
CREATE INDEX issue_activities_issue_id_created_at_idx ON issue_activities (issue_id, created_at, id);
-- 物理级联按它们找子行（4 节开头）
CREATE INDEX issue_activities_workspace_id_idx ON issue_activities (workspace_id);
CREATE INDEX issue_activities_project_id_idx ON issue_activities (project_id);

-- +goose Down
DROP TABLE issue_activities;
````

`server/sqlc.yaml`（修改，1 处）：

````old server/sqlc.yaml
      - migrations/sql/00015_issue_issues.sql
````
````new server/sqlc.yaml
      - migrations/sql/00015_issue_issues.sql
      - migrations/sql/00016_issue_issue_assignees.sql
      - migrations/sql/00017_issue_issue_labels.sql
      - migrations/sql/00018_issue_issue_subscribers.sql
      - migrations/sql/00019_issue_issue_activities.sql
````

- [ ] **Step 2: 五条删除的语句，生成**

`server/internal/modules/issue/adapter/postgres/queries/delete_rows.sql`（整个文件，54 行）：

````whole server/internal/modules/issue/adapter/postgres/queries/delete_rows.sql
-- name: DeleteProjectIssueRows :many
-- The deletion of a project's work-item side (M4 design 3.6), the work items' table: a batch of the project's
-- undeleted rows after the id `after`, in id order, locked FOR NO KEY UPDATE as they come and soft-deleted at the
-- moment and by the account given (convention 5: two batches over the same rows take them in one order). The caller
-- runs the tables' statements in the global order (3.5) and, for each table, again from the batch's greatest id until
-- a batch finds none (a row deleted while the batch waits for it is passed over and the next one taken; each batch
-- reads a new snapshot). The primary key's index serves the id order and the bound. It returns the ids it deleted.
UPDATE issues AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issues r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;

-- name: DeleteProjectAssigneeRows :many
-- DeleteProjectIssueRows' batch, of the work items' assignees.
UPDATE issue_assignees AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issue_assignees r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;

-- name: DeleteProjectLabelRows :many
-- DeleteProjectIssueRows' batch, of the work items' labels.
UPDATE issue_labels AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issue_labels r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;

-- name: DeleteProjectSubscriberRows :many
-- DeleteProjectIssueRows' batch, of the work items' subscribers.
UPDATE issue_subscribers AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issue_subscribers r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;

-- name: DeleteProjectActivityRows :many
-- DeleteProjectIssueRows' batch, of the work items' activity.
UPDATE issue_activities AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issue_activities r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;
````

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `f0baebccc3984ae921fd1d502a74b66cbb1602827199c99b3f3091f790298e51` | 243 | `server/internal/modules/issue/adapter/postgres/gen/delete_rows.sql.go` |
| `354335adbeb7af6b7725ee7fb4cfdbd4b48594a364a3577f98ebb50294c31d76` | 94 | `server/internal/modules/issue/adapter/postgres/gen/models.go` |

Run: `shasum -a 256 server/internal/modules/issue/adapter/postgres/gen/delete_rows.sql.go server/internal/modules/issue/adapter/postgres/gen/models.go`
Expected: 与上表相同（`gen/db.go` 不变）。

- [ ] **Step 3: 存储方法、编排和它的单元测试**

`server/internal/modules/issue/adapter/postgres/delete_rows.go`（修改，2 处）：

````old server/internal/modules/issue/adapter/postgres/delete_rows.go
// batch is b as the statement takes it.
````
````new server/internal/modules/issue/adapter/postgres/delete_rows.go
// batch is b as the statements take it; their parameters are of one shape.
````
````old server/internal/modules/issue/adapter/postgres/delete_rows.go
	return deleted(ids, err, "work items")
}

````
````new server/internal/modules/issue/adapter/postgres/delete_rows.go
	return deleted(ids, err, "work items")
}

// DeleteProjectAssigneeRows soft-deletes a batch of the project's work
// items' assignees.
func (s *Store) DeleteProjectAssigneeRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteProjectAssigneeRows(ctx, gen.DeleteProjectAssigneeRowsParams(batch(b)))
	return deleted(ids, err, "assignees")
}

// DeleteProjectLabelRows soft-deletes a batch of the labels on the
// project's work items.
func (s *Store) DeleteProjectLabelRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteProjectLabelRows(ctx, gen.DeleteProjectLabelRowsParams(batch(b)))
	return deleted(ids, err, "work items' labels")
}

// DeleteProjectSubscriberRows soft-deletes a batch of the project's work
// items' subscribers.
func (s *Store) DeleteProjectSubscriberRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteProjectSubscriberRows(ctx, gen.DeleteProjectSubscriberRowsParams(batch(b)))
	return deleted(ids, err, "subscribers")
}

// DeleteProjectActivityRows soft-deletes a batch of the project's work
// items' activity.
func (s *Store) DeleteProjectActivityRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteProjectActivityRows(ctx, gen.DeleteProjectActivityRowsParams(batch(b)))
	return deleted(ids, err, "activity")
}

````

`server/internal/modules/issue/app/cascade_test.go`（新文件，176 行）：

````file server/internal/modules/issue/app/cascade_test.go
package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

var errDisk = errors.New("disk full")

// inTxKey marks the context of the caller's transaction: a call that does
// not carry it ran outside it.
type inTxKey struct{}

// fakeRows is the deleted project's rows, by statement and project, as
// ProjectRowsDeleter's statements see them: each batch takes the
// undeleted rows of the project after the batch's id, at most its limit,
// in id order, marks them deleted, and answers their ids, the greatest
// first, so that a caller who took the last of them for the next batch's
// bound would read again what it deleted. It logs each batch, and fails
// the statement named in fail.
type fakeRows struct {
	rows    map[string]map[uuid.UUID][]uuid.UUID // statement → project → ids, ascending
	deleted map[uuid.UUID]string                 // id → "by at"
	calls   []string
	fail    string
}

func (f *fakeRows) batch(ctx context.Context, statement string, b app.RowBatch) ([]uuid.UUID, error) {
	call := fmt.Sprintf("%s %s after %s limit %d", statement, b.ProjectID, b.After, b.Limit)
	if ctx.Value(inTxKey{}) != true {
		call += " outside tx"
	}
	f.calls = append(f.calls, call)
	if statement == f.fail {
		return nil, errDisk
	}
	var ids []uuid.UUID
	for _, id := range f.rows[statement][b.ProjectID] {
		if _, gone := f.deleted[id]; !gone && id.Compare(b.After) > 0 && len(ids) < b.Limit {
			ids = append(ids, id)
			f.deleted[id] = fmt.Sprintf("%s at %s", b.By, b.At.Format(time.RFC3339Nano))
		}
	}
	slices.Reverse(ids)
	return ids, nil
}

func (f *fakeRows) DeleteProjectIssueRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.batch(ctx, "issues", b)
}

func (f *fakeRows) DeleteProjectAssigneeRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.batch(ctx, "assignees", b)
}

func (f *fakeRows) DeleteProjectLabelRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.batch(ctx, "labels", b)
}

func (f *fakeRows) DeleteProjectSubscriberRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.batch(ctx, "subscribers", b)
}

func (f *fakeRows) DeleteProjectActivityRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.batch(ctx, "activities", b)
}

// ids are n ids drawn in order: ascending.
func ids(n int) []uuid.UUID {
	out := make([]uuid.UUID, n)
	for i := range out {
		out[i] = uuid.NewV7()
	}
	return out
}

// twoProjects are the rows of web and ops: web has three work items, an
// assignee and two activity rows, ops a work item, a label on it and a
// subscriber.
func twoProjects(web, ops uuid.UUID) *fakeRows {
	w, o := ids(6), ids(3)
	return &fakeRows{deleted: map[uuid.UUID]string{}, rows: map[string]map[uuid.UUID][]uuid.UUID{
		"issues":      {web: w[0:3], ops: o[0:1]},
		"assignees":   {web: w[3:4]},
		"labels":      {ops: o[1:2]},
		"subscribers": {ops: o[2:3]},
		"activities":  {web: w[4:6]},
	}}
}

// batches are the calls deleteProjectRows makes on project's rows of f,
// in batches of two: per statement in the global order, a batch after the
// zero id, then after the greatest id of each batch, until one finds none.
func batches(f *fakeRows, project uuid.UUID) []string {
	var calls []string
	for _, statement := range []string{"issues", "assignees", "labels", "subscribers", "activities"} {
		rows := f.rows[statement][project]
		after := uuid.UUID{}
		for i := 0; ; i += 2 {
			calls = append(calls, fmt.Sprintf("%s %s after %s limit 2", statement, project, after))
			if i >= len(rows) {
				break
			}
			after = rows[min(i+2, len(rows))-1]
		}
	}
	return calls
}

// DeleteProjectIssues soft-deletes every row of the project's work-item
// side, and none of another project's, in the caller's transaction, at its
// moment and by its account: statement by statement in the global order of
// M4 design 3.5, each in batches of the limit, the next after the greatest
// id of the last, until a batch finds none (M4 design 3.6).
// DeleteWorkspaceIssues does it for each project it is given, in that
// order.
func TestTheCascadeDeletesEachProjectsRowsInBatches(t *testing.T) {
	web, ops, by := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	now := time.Date(2026, 10, 17, 10, 0, 0, 123456000, time.UTC)
	inTx := context.WithValue(context.Background(), inTxKey{}, true)
	for _, tt := range []struct {
		name     string
		run      func(c *app.Cascade) error
		projects []uuid.UUID
	}{
		{"one project", func(c *app.Cascade) error { return c.DeleteProjectIssues(inTx, web, by, now) }, []uuid.UUID{web}},
		{"a workspace's two", func(c *app.Cascade) error { return c.DeleteWorkspaceIssues(inTx, []uuid.UUID{ops, web}, by, now) },
			[]uuid.UUID{ops, web}},
		{"a workspace without projects", func(c *app.Cascade) error { return c.DeleteWorkspaceIssues(inTx, nil, by, now) }, nil},
	} {
		f := twoProjects(web, ops)
		if err := tt.run(app.NewCascade(f, 2)); err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		var want []string
		deleted := map[uuid.UUID]string{}
		for _, p := range tt.projects {
			want = append(want, batches(f, p)...)
			for _, rows := range f.rows {
				for _, id := range rows[p] {
					deleted[id] = fmt.Sprintf("%s at %s", by, now.Format(time.RFC3339Nano))
				}
			}
		}
		if !slices.Equal(f.calls, want) {
			t.Errorf("%s: the batches\n%q\nwant\n%q", tt.name, f.calls, want)
		}
		if fmt.Sprint(f.deleted) != fmt.Sprint(deleted) {
			t.Errorf("%s: deleted %v, want %v", tt.name, f.deleted, deleted)
		}
	}
}

// A failing batch comes back as itself, and no batch runs after it.
func TestTheCascadeStopsAtAFailingBatch(t *testing.T) {
	web, ops, by := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for _, statement := range []string{"issues", "assignees", "labels", "subscribers", "activities"} {
		f := twoProjects(web, ops)
		f.fail = statement
		want := batches(f, web)
		want = want[:slices.IndexFunc(want, func(c string) bool { return strings.HasPrefix(c, statement+" ") })+1]
		err := app.NewCascade(f, 2).DeleteWorkspaceIssues(context.WithValue(context.Background(), inTxKey{}, true), []uuid.UUID{web, ops}, by,
			time.Now())
		if !errors.Is(err, errDisk) || !slices.Equal(f.calls, want) {
			t.Errorf("failing %s: %v, the batches\n%q\nwant %v after\n%q", statement, err, f.calls, errDisk, want)
		}
	}
}
````

`server/internal/modules/issue/app/delete_project_rows.go`（修改，3 处）：

````old server/internal/modules/issue/app/delete_project_rows.go
	DeleteProjectIssueRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
````
````new server/internal/modules/issue/app/delete_project_rows.go
	DeleteProjectIssueRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	DeleteProjectAssigneeRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	DeleteProjectLabelRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	DeleteProjectSubscriberRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	DeleteProjectActivityRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
````
````old server/internal/modules/issue/app/delete_project_rows.go
// work items.
````
````new server/internal/modules/issue/app/delete_project_rows.go
// work items, then the rows under them.
````
````old server/internal/modules/issue/app/delete_project_rows.go
	return []func(context.Context, RowBatch) ([]uuid.UUID, error){d.DeleteProjectIssueRows}
````
````new server/internal/modules/issue/app/delete_project_rows.go
	return []func(context.Context, RowBatch) ([]uuid.UUID, error){
		d.DeleteProjectIssueRows, d.DeleteProjectAssigneeRows, d.DeleteProjectLabelRows, d.DeleteProjectSubscriberRows,
		d.DeleteProjectActivityRows,
	}
````

- [ ] **Step 4: 组合的删除测试**

`server/internal/bootstrap/project_deletion_test.go`（修改，3 处）：

````old server/internal/bootstrap/project_deletion_test.go
// Bug, with its child, UI, and a work item of hers, written directly
// (seedLabels, seedIssues).
````
````new server/internal/bootstrap/project_deletion_test.go
// Bug, with its child, UI, and a work item of hers with a row of each
// table under it, written directly (seedLabels, seedIssues).
````
````old server/internal/bootstrap/project_deletion_test.go
// directly: the seed does not depend on which of the issue module's writes
// exist. Its number is the project's counter, moved on, as a creation
// moves it.
````
````new server/internal/bootstrap/project_deletion_test.go
// and a row of each of the issue module's tables under it: by its assignee
// and its subscriber, the project's label Bug (seedLabels) on it, and its
// activity's first row, directly: the seed does not depend on which of the
// issue module's writes exist. Its number is the project's counter, moved
// on, as a creation moves it.
````
````old server/internal/bootstrap/project_deletion_test.go
			[]any{issue, project, by}},
````
````new server/internal/bootstrap/project_deletion_test.go
			[]any{issue, project, by}},
		{`INSERT INTO issue_assignees (id, workspace_id, project_id, issue_id, assignee_id, created_by_id, updated_by_id)
			SELECT $1, workspace_id, project_id, id, $3, $3, $3 FROM issues WHERE id = $2`, []any{uuid.NewV7(), issue, by}},
		{`INSERT INTO issue_labels (id, workspace_id, project_id, issue_id, label_id, created_by_id, updated_by_id)
			SELECT $1, i.workspace_id, i.project_id, i.id, l.id, $3, $3 FROM issues i JOIN labels l ON l.project_id = i.project_id AND l.name = 'Bug'
			WHERE i.id = $2`, []any{uuid.NewV7(), issue, by}},
		{`INSERT INTO issue_subscribers (id, workspace_id, project_id, issue_id, subscriber_id, created_by_id, updated_by_id)
			SELECT $1, workspace_id, project_id, id, $3, $3, $3 FROM issues WHERE id = $2`, []any{uuid.NewV7(), issue, by}},
		{`INSERT INTO issue_activities (id, workspace_id, project_id, issue_id, comment, actor_id, created_by_id, updated_by_id)
			SELECT $1, workspace_id, project_id, id, 'created the work item', $3, $3, $3 FROM issues WHERE id = $2`, []any{uuid.NewV7(), issue, by}},
````

`server/internal/bootstrap/workspace_deletion_test.go`（修改，8 处）：

````old server/internal/bootstrap/workspace_deletion_test.go
// settings, through the API; a project with the member as its admin, his
// display settings in it, a state, two labels and a work item
// (seedProject). A phase that adds a table under workspaces seeds a row of
// it here. It returns the workspace's id.
````
````new server/internal/bootstrap/workspace_deletion_test.go
// settings, through the API; two projects, Web and Ops, each with the
// member as its admin, his display settings in it, a state, two labels and
// a work item with a row of each table under it (seedProject): the
// work-item side is deleted project by project (M4 design 3.5). A phase
// that adds a table under workspaces seeds a row of it here. It returns
// the workspace's id.
````
````old server/internal/bootstrap/workspace_deletion_test.go
	seedProject(t, pool, id, admin, member)
````
````new server/internal/bootstrap/workspace_deletion_test.go
	seedProject(t, pool, id, admin, member, "Web")
	seedProject(t, pool, id, admin, member, "Ops")
````
````old server/internal/bootstrap/workspace_deletion_test.go
// seedProject writes a project of the workspace id, created by admin,
// with member as its admin, his display settings in it, one state, a label
// with its child and a work item by member (seedLabels, seedIssues),
// directly: the seed does not depend on which of the project module's
// writes exist, nor on what they write besides. member, a project admin
// who is a member of the workspace (PA in M3 design 9.2), may write the
// project's labels. No row under the
// project is the last writing of admin, the workspace's deleter (the test
// reads it back), so a deletion by admin that kept a row's writer would
// show.
func seedProject(t *testing.T, pool *pgxpool.Pool, id, admin, member uuid.UUID) {
````
````new server/internal/bootstrap/workspace_deletion_test.go
// seedProject writes a project of the workspace id named name, its
// identifier the name in upper case, created by admin, with member as its
// admin, his display settings in it, one state, a label with its child and
// a work item with a row of each table under it, by member (seedLabels,
// seedIssues), directly: the seed does not depend on which of the project
// module's writes exist, nor on what they write besides. member, a project
// admin who is a member of the workspace (PA in M3 design 9.2), may write
// the project's labels. No row under the project is the last writing of
// admin, the workspace's deleter (the test reads it back), so a deletion
// by admin that kept a row's writer would show.
func seedProject(t *testing.T, pool *pgxpool.Pool, id, admin, member uuid.UUID, name string) {
````
````old server/internal/bootstrap/workspace_deletion_test.go
		{"INSERT INTO projects (id, workspace_id, name, identifier, created_by_id) VALUES ($1, $2, 'Web', 'WEB', $3)", []any{project, id, admin}},
````
````new server/internal/bootstrap/workspace_deletion_test.go
		{"INSERT INTO projects (id, workspace_id, name, identifier, created_by_id) VALUES ($1, $2, $4::text, upper($4::text), $3)",
			[]any{project, id, admin, name}},
````
````old server/internal/bootstrap/workspace_deletion_test.go
// workspace's own. The labels table, renamed while the request runs, fails
// the last statement of the cascade. It does not show that the statements
// share the deletion's one transaction: the projects' statements in a
// transaction of their own roll back here too, with the failing one.
````
````new server/internal/bootstrap/workspace_deletion_test.go
// workspace's own. The work items' activity table, renamed while the
// request runs, fails the last statement of the cascade (M4 design 3.5).
// It does not show that the statements share the deletion's one
// transaction: the projects' statements in a transaction of their own roll
// back here too, with the failing one.
````
````old server/internal/bootstrap/workspace_deletion_test.go
	exec("ALTER TABLE labels RENAME TO labels_away")
````
````new server/internal/bootstrap/workspace_deletion_test.go
	exec("ALTER TABLE issue_activities RENAME TO issue_activities_away")
````
````old server/internal/bootstrap/workspace_deletion_test.go
	exec("ALTER TABLE labels_away RENAME TO labels")
````
````new server/internal/bootstrap/workspace_deletion_test.go
	exec("ALTER TABLE issue_activities_away RENAME TO issue_activities")
````
````old server/internal/bootstrap/workspace_deletion_test.go
		t.Fatalf("deleting the workspace with the labels' step failing = %d %s, want 500", status, body)
````
````new server/internal/bootstrap/workspace_deletion_test.go
		t.Fatalf("deleting the workspace with the activity's step failing = %d %s, want 500", status, body)
````

- [ ] **Step 5: 端到端的过渡**

`e2e/fixtures/assert/project.ts`（修改，1 处）：

````old e2e/fixtures/assert/project.ts
  const withoutRows = new Set(["issues"]);
````
````new e2e/fixtures/assert/project.ts
  const withoutRows = new Set(["issue_activities", "issue_assignees", "issue_labels", "issue_subscribers", "issues"]);
````

- [ ] **Step 6: 测试和 lint**

Run: `go -C server test -count=1 ./migrations/ ./internal/modules/issue/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestDeletingAProjectLeavesNoUndeletedRowUnderIt$|TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt$|TestAFailedProjectsStepRollsTheDeletionBack$|TestAProjectDeletionRefusedAtItsCommitChangesNoRow$|TestADeletionRefusedAtItsCommitChangesNoRow$' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过；关键词守卫 `keywords: 72 rules, 23 exceptions, no hits.`

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 120 个全部通过（`expectProjectDeleted` 要求五张表没有行）。

- [ ] **Step 7: 提交**

```bash
git add deploy/runtime-grants.sql e2e/fixtures/assert/project.ts server/internal/bootstrap/project_deletion_test.go server/internal/bootstrap/workspace_deletion_test.go server/internal/modules/issue/adapter/postgres/delete_rows.go server/internal/modules/issue/adapter/postgres/queries/delete_rows.sql server/internal/modules/issue/app/cascade_test.go server/internal/modules/issue/app/delete_project_rows.go server/migrations/issue_schema_test.go server/migrations/schema_test.go server/migrations/sql/00016_issue_issue_assignees.sql server/migrations/sql/00017_issue_issue_labels.sql server/migrations/sql/00018_issue_issue_subscribers.sql server/migrations/sql/00019_issue_issue_activities.sql server/sqlc.yaml server/internal/modules/issue/adapter/postgres/gen/delete_rows.sql.go server/internal/modules/issue/adapter/postgres/gen/models.go
```
```bash
git commit -m "feat(M4/P1): the assignees', labels', subscribers' and activity tables join the deletions

Migrations 00016 to 00019 add the rows under a work item: its
assignees, labels and subscribers, each once among the undeleted, and
its activity. Each carries its workspace and project, so the deletions'
catalog-driven tests find it. Deleting a project, or a workspace,
soft-deletes the five tables in the global order, each in batches in id
order from the greatest id of the last batch until one finds none; a
failing statement rolls the whole deletion back.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p1a.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试；结构是迁移的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是关键词守卫；静态是 `make lint-web` 的文案和键的核对）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T3.1` | 编排漏掉动态的表 | `internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAProject`、`TestAProjectDeletionRefused`（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkspace`、`TestAFailedProjectsStep`、`TestADeletionRefused`（`make test`）、故事 `p4-archive`、故事 `w3-workspace-settings`（Task 14 起） | 单元；组合；端到端 |
| `T3.2` | 编排先删动态、再删工作项（与全局顺序相反） | `internal/modules/issue/app` 的 全部测试（`make test`） | 单元 |
| `T3.3` | 一张表的分批在第一个不满的批次就停，不等到空的批次 | `internal/modules/issue/app` 的 全部测试（`make test`） | 单元 |
| `T3.4` | 下一批从上一批回答的最小的 `id` 之后开始，不是最大的 | `internal/modules/issue/app` 的 全部测试（`make test`） | 单元 |
| `T3.5` | 删除工作区只删它的第一个项目的工作项一侧 | `internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkspace`、`TestAFailedProjectsStep`、`TestADeletionRefused`（`make test`） | 单元；组合 |
| `T3.6` | 授权文件漏掉 `issue_activities` | `internal/bootstrap` 的 `TestTheGrantsFileCoversEveryRelationAndFunction`、`TestTheRuntimeRoleServesWithTheGrantsFile`（`make test`） | 组合 |
| `T3.7` | 失败的一批当作没有行：这张表的分批结束，删除照样往下做 | `internal/modules/issue/app` 的 全部测试（`make test`） | 单元 |

---

### Task 4: 删除语句的存储测试；CHECK 的反例和唯一键

**Files:**
- Create: `server/internal/modules/issue/adapter/postgres/delete_rows_test.go`、`server/internal/modules/issue/adapter/postgres/store_test.go`
- Modify: `server/migrations/issue_schema_test.go`

**Interfaces:** 没有新的生产代码。
- `issue/adapter/postgres/store_test.go`：存储测试共用的帮手，后面的存储测试照用：
  - `newStore`（迁移过的数据库上的 `Store` 和连接池）、固定时刻 `now`、`exec`；
  - 直接写的夹具（这些表属于别的模块，测试直接写）：`newAccount`、`newWorkspace`、`newProject`（带默认状态）、`newIssue`。

**Tests:**
- 存储（`delete_rows_test.go`）：
  - `TestTheBatchStatementsDeleteAProjectsRowsInIDOrder`：五条语句各一个子测试（`batchStatements` 用方法表达式，`child` 写工作项下的行），每条都核对：
    - 一条语句取这个项目 `after` 之后未删除的行，`id` 升序，至多 `limit` 行，不管行在表中的次序；
    - 在删除的时刻、由删除者软删除它们，此外什么都不改：被删除的行的其余列、此前已删除的行、另一个项目的行都不变；
    - 一批从 c 之后取 d、e（回滚掉）；
    - 从每批的最大 `id` 重跑：两行一批，直到为空；
    - 读失败时回答失败，不当作没有行。
- 迁移（`issue_schema_test.go`）：
  - `TestIssueChecksRejectCounterexamples`：
    - 接受领域写的：每个优先级和动词；开始日期等于目标日期，或没有目标日期；没有父工作项；编号 1；
    - 拒绝：空的标题、五个之外的优先级（大小写不同也拒绝）、编号 0、自己做自己的父工作项、开始晚于目标、三个之外的动词。
  - `TestIssueUniqueKeys`：
    - 编号在项目内全表唯一：第二个在第一个软删除之前和之后都被拒绝，另一个项目有自己的 1；
    - 三张关联表的键只在未删除的行之间唯一：同一对的第二行被拒绝，软删除第一行之后可以；另一个工作项有它自己的同一对。

- [ ] **Step 1: 存储测试**

`server/internal/modules/issue/adapter/postgres/delete_rows_test.go`（新文件，225 行）：

````file server/internal/modules/issue/adapter/postgres/delete_rows_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// inserter inserts the row id of a table under the work item issue of p,
// by by, the n-th of the test, so each takes an account or a label of its
// own where the table's key asks for one.
type inserter func(t *testing.T, pool *pgxpool.Pool, p project, issue, id, by uuid.UUID, n int)

// batchStatement is one of a deleted project's batch statements: the table
// it writes, the store's method, and how a row of the table is inserted.
type batchStatement struct {
	table  string
	method func(*postgresadapter.Store, context.Context, app.RowBatch) ([]uuid.UUID, error)
	insert inserter
}

// child inserts a row of table under the work item, its column the value
// value takes.
func child(table, column string, value func(t *testing.T, pool *pgxpool.Pool, p project, n int) uuid.UUID) inserter {
	return func(t *testing.T, pool *pgxpool.Pool, p project, issue, id, by uuid.UUID, n int) {
		t.Helper()
		exec(t, pool, `INSERT INTO `+table+` (id, workspace_id, project_id, issue_id, `+column+`, created_by_id, updated_by_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $6, $7, $7)`, id, p.workspace, p.id, issue, value(t, pool, p, n), by, now)
	}
}

// anAccount is an account of its own for the n-th row.
func anAccount(t *testing.T, pool *pgxpool.Pool, _ project, n int) uuid.UUID {
	return newAccount(t, pool, fmt.Sprintf("u%d@corp.com", n))
}

// aLabel is a label of p of its own for the n-th row.
func aLabel(t *testing.T, pool *pgxpool.Pool, p project, n int) uuid.UUID {
	id := uuid.NewV7()
	exec(t, pool, "INSERT INTO labels (id, workspace_id, project_id, name) VALUES ($1, $2, $3, $4)", id, p.workspace, p.id, fmt.Sprint("L", n))
	return id
}

var batchStatements = []batchStatement{
	{"issues", (*postgresadapter.Store).DeleteProjectIssueRows, func(t *testing.T, pool *pgxpool.Pool, p project, _, id, by uuid.UUID, n int) {
		newIssue(t, pool, p, id, 100+n, by)
	}},
	{"issue_assignees", (*postgresadapter.Store).DeleteProjectAssigneeRows, child("issue_assignees", "assignee_id", anAccount)},
	{"issue_labels", (*postgresadapter.Store).DeleteProjectLabelRows, child("issue_labels", "label_id", aLabel)},
	{"issue_subscribers", (*postgresadapter.Store).DeleteProjectSubscriberRows, child("issue_subscribers", "subscriber_id", anAccount)},
	{"issue_activities", (*postgresadapter.Store).DeleteProjectActivityRows, func(t *testing.T, pool *pgxpool.Pool, p project, issue, id,
		by uuid.UUID, _ int) {
		exec(t, pool, `INSERT INTO issue_activities (id, workspace_id, project_id, issue_id, actor_id, created_by_id, updated_by_id, created_at,
			updated_at) VALUES ($1, $2, $3, $4, $5, $5, $5, $6, $6)`, id, p.workspace, p.id, issue, by, now)
	}},
}

// rowsAsText is every row of table but its audit columns of a deletion, by
// id, as text.
func rowsAsText(t *testing.T, pool *pgxpool.Pool, table string) map[uuid.UUID]string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `SELECT id, (to_jsonb(t) - 'deleted_at' - 'updated_at' - 'updated_by_id')::text FROM `+table+` t`)
	if err != nil {
		t.Fatal(err)
	}
	out := map[uuid.UUID]string{}
	for rows.Next() {
		var id uuid.UUID
		var text string
		if err := rows.Scan(&id, &text); err != nil {
			t.Fatal(err)
		}
		out[id] = text
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// Each batch statement of a deleted project (M4 design 3.6) takes, in one
// statement, at most the limit of the project's undeleted rows of its
// table whose ids are after the bound, in id order, whatever order the
// rows lie in, and soft-deletes them at the moment and by the account
// given, and nothing else: a batch after c takes d and e only (rolled
// back); run again from the greatest id of each batch, its batches are the
// project's undeleted rows in id order, two at a time, then none. web's
// rows lie out of their id order in the table, a row of ops lies among
// them, and one of web's was deleted before, by alice, an hour earlier.
// Every other column of the rows deleted is kept, and every column of the
// earlier deleted row and of ops' rows. Each statement answers a failed
// read as its failure, never as no rows.
func TestTheBatchStatementsDeleteAProjectsRowsInIDOrder(t *testing.T) {
	for _, st := range batchStatements {
		t.Run(st.table, func(t *testing.T) {
			s, pool := newStore(t)
			alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
			acme := newWorkspace(t, pool, "acme")
			web, ops := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS")
			webIssue, opsIssue := uuid.NewV7(), uuid.NewV7()
			newIssue(t, pool, web, webIssue, 1, alice)
			newIssue(t, pool, ops, opsIssue, 1, alice)
			// drawn in this order: ascending ids
			a, gone, inOps, b, c, d, e := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
			for n, row := range []uuid.UUID{e, c, inOps, a, d, gone, b} {
				if row == inOps {
					st.insert(t, pool, ops, opsIssue, row, alice, n)
				} else {
					st.insert(t, pool, web, webIssue, row, alice, n)
				}
			}
			earlier, later := now.Add(-time.Hour), now.Add(time.Hour)
			exec(t, pool, "UPDATE "+st.table+" SET deleted_at = $2, updated_at = $2, updated_by_id = $3 WHERE id = $1", gone, earlier, alice)
			var webRows []uuid.UUID // web's undeleted rows of the table, in id order
			for _, id := range []uuid.UUID{webIssue, a, b, c, d, e} {
				if st.table == "issues" || id != webIssue {
					webRows = append(webRows, id)
				}
			}
			before := rowsAsText(t, pool, st.table)
			auditBefore := audits(t, pool, st.table)
			tx := postgres.NewTxManager(pool, 2*time.Second)

			rollBack := errors.New("roll back")
			err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
				ids, err := st.method(s, ctx, app.RowBatch{ProjectID: web.id, After: c, Limit: 10, By: bob, At: later})
				if got := slices.SortedFunc(slices.Values(ids), uuid.UUID.Compare); err != nil || fmt.Sprint(got) != fmt.Sprint([]uuid.UUID{d, e}) {
					t.Errorf("a batch after c = %v, %v; want d and e only", got, err)
				}
				return rollBack
			})
			if !errors.Is(err, rollBack) {
				t.Fatal(err)
			}

			var got [][]uuid.UUID
			err = tx.WithinTx(context.Background(), func(ctx context.Context) error {
				for after := (uuid.UUID{}); ; {
					ids, err := st.method(s, ctx, app.RowBatch{ProjectID: web.id, After: after, Limit: 2, By: bob, At: later})
					if err != nil {
						return err
					}
					got = append(got, slices.SortedFunc(slices.Values(ids), uuid.UUID.Compare))
					if len(ids) == 0 || len(got) > 10 {
						return nil
					}
					after = slices.MaxFunc(ids, uuid.UUID.Compare)
				}
			})
			if err != nil {
				t.Fatal(err)
			}

			var want [][]uuid.UUID
			for chunk := range slices.Chunk(webRows, 2) {
				want = append(want, chunk)
			}
			want = append(want, nil)
			if fmt.Sprint(got) != fmt.Sprint(want) {
				t.Errorf("the batches %v, want %v", got, want)
			}
			if after := rowsAsText(t, pool, st.table); fmt.Sprint(after) != fmt.Sprint(before) {
				t.Errorf("the rows but their audit columns:\n%v\nwant them as they were:\n%v", after, before)
			}
			auditAfter := audits(t, pool, st.table)
			for id, was := range auditBefore {
				want := was
				if slices.Contains(webRows, id) {
					want = fmt.Sprintf("deleted at %s, written at %s by %s", later.Format(time.RFC3339Nano), later.Format(time.RFC3339Nano), bob)
				}
				if auditAfter[id] != want {
					t.Errorf("row %s: %s, want %s", id, auditAfter[id], want)
				}
			}

			cancelled, cancel := context.WithCancel(context.Background())
			cancel()
			if ids, err := st.method(s, cancelled, app.RowBatch{ProjectID: ops.id, Limit: 2, By: bob, At: later}); !errors.Is(err, context.Canceled) ||
				ids != nil {
				t.Errorf("a batch on a cancelled context = %v, %v; want context.Canceled, not no rows", ids, err)
			}
		})
	}
}

// audits are table's rows' audit columns of a deletion, by id, as text.
func audits(t *testing.T, pool *pgxpool.Pool, table string) map[uuid.UUID]string {
	t.Helper()
	rows, err := pool.Query(context.Background(), "SELECT id, deleted_at, updated_at, updated_by_id FROM "+table)
	if err != nil {
		t.Fatal(err)
	}
	out := map[uuid.UUID]string{}
	for rows.Next() {
		var id uuid.UUID
		var deleted *time.Time
		var written time.Time
		var by *uuid.UUID
		if err := rows.Scan(&id, &deleted, &written, &by); err != nil {
			t.Fatal(err)
		}
		d, writer := "undeleted", "nobody"
		if deleted != nil {
			d = "deleted at " + deleted.UTC().Format(time.RFC3339Nano)
		}
		if by != nil {
			writer = by.String()
		}
		out[id] = fmt.Sprintf("%s, written at %s by %s", d, written.UTC().Format(time.RFC3339Nano), writer)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}
````

`server/internal/modules/issue/adapter/postgres/store_test.go`（新文件，81 行）：

````file server/internal/modules/issue/adapter/postgres/store_test.go
package postgresadapter_test

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// now is the fixed clock's time: whole microseconds, as timestamptz stores
// them, so the audit columns read back equal to it (M2 design 3.13).
var now = clocktest.At(time.Date(2026, 10, 17, 10, 0, 0, 123456789, time.UTC)).Now()

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

// newAccount inserts the users row that the issue tables reference.
// identity owns the table: the test writes it directly, as a fixture.
func newAccount(t *testing.T, pool *pgxpool.Pool, email string) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	exec(t, pool, "INSERT INTO users (id, email, password, display_name) VALUES ($1, $2, 'x', 'x')", id, email)
	return id
}

// newWorkspace inserts the workspaces row the issue tables reference.
// workspace owns the table: the test writes it directly, as a fixture.
func newWorkspace(t *testing.T, pool *pgxpool.Pool, slug string) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	exec(t, pool, "INSERT INTO workspaces (id, name, slug) VALUES ($1, $2, $2)", id, slug)
	return id
}

// project is a project the issue tables reference, with its default
// state: project owns the tables, the test writes them directly, as a
// fixture.
type project struct {
	workspace, id, state uuid.UUID
}

// newProject inserts a project of workspace with the identifier, and its
// default state.
func newProject(t *testing.T, pool *pgxpool.Pool, workspace uuid.UUID, identifier string) project {
	t.Helper()
	p := project{workspace: workspace, id: uuid.NewV7(), state: uuid.NewV7()}
	exec(t, pool, "INSERT INTO projects (id, workspace_id, name, identifier) VALUES ($1, $2, $3, $3)", p.id, workspace, identifier)
	exec(t, pool, `INSERT INTO states (id, workspace_id, project_id, name, color, "default") VALUES ($1, $2, $3, 'Backlog', '#60646C', true)`,
		p.state, workspace, p.id)
	return p
}

// newIssue inserts the work item id of p, numbered sequence, in p's
// default state, by by at now.
func newIssue(t *testing.T, pool *pgxpool.Pool, p project, id uuid.UUID, sequence int, by uuid.UUID) {
	t.Helper()
	exec(t, pool, `INSERT INTO issues (id, workspace_id, project_id, state_id, name, sequence_id, created_by_id, updated_by_id, created_at,
		updated_at) VALUES ($1, $2, $3, $4, 'Work', $5, $6, $6, $7, $7)`, id, p.workspace, p.id, p.state, sequence, by, now)
}
````

- [ ] **Step 2: 迁移的测试**

`server/migrations/issue_schema_test.go`（修改，2 处）：

````old server/migrations/issue_schema_test.go
package migrations_test
````
````new server/migrations/issue_schema_test.go
package migrations_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)
````
````old server/migrations/issue_schema_test.go
}

````
````new server/migrations/issue_schema_test.go
}

// The ids the issue tables' tests write, as SQL literals: two accounts, a
// workspace, two projects with a state each, a label of web, and a work
// item of each project.
const (
	issueAlice = "'0199a2b4-0000-7000-8000-000000000001'"
	issueBob   = "'0199a2b4-0000-7000-8000-000000000002'"
	issueAcme  = "'0199a2b4-0000-7000-8000-000000000003'"
	issueWeb   = "'0199a2b4-0000-7000-8000-000000000004'"
	issueOps   = "'0199a2b4-0000-7000-8000-000000000005'"
	webState   = "'0199a2b4-0000-7000-8000-000000000006'"
	opsState   = "'0199a2b4-0000-7000-8000-000000000007'"
	webBug     = "'0199a2b4-0000-7000-8000-000000000008'"
	webIssue   = "'0199a2b4-0000-7000-8000-000000000009'"
	opsIssue   = "'0199a2b4-0000-7000-8000-00000000000a'"
)

// newIssueDatabase is a database with the rows of the ids above, each work
// item numbered 1, and an activity row of web's.
func newIssueDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := newPool(t, pgtest.NewDatabase(t))
	for _, stmt := range []string{
		"INSERT INTO users (id, email, password, display_name) VALUES (" + issueAlice + ", 'alice@corp.com', 'x', 'alice'), (" + issueBob +
			", 'bob@corp.com', 'x', 'bob')",
		"INSERT INTO workspaces (id, name, slug) VALUES (" + issueAcme + ", 'Acme', 'acme')",
		"INSERT INTO projects (id, workspace_id, name, identifier) VALUES (" + issueWeb + ", " + issueAcme + ", 'Web', 'WEB'), (" + issueOps +
			", " + issueAcme + ", 'Ops', 'OPS')",
		"INSERT INTO states (id, workspace_id, project_id, name, color) VALUES (" + webState + ", " + issueAcme + ", " + issueWeb +
			", 'Backlog', '#60646C'), (" + opsState + ", " + issueAcme + ", " + issueOps + ", 'Backlog', '#60646C')",
		"INSERT INTO labels (id, workspace_id, project_id, name) VALUES (" + webBug + ", " + issueAcme + ", " + issueWeb + ", 'Bug')",
		"INSERT INTO issues (id, workspace_id, project_id, state_id, name, sequence_id) VALUES (" + webIssue + ", " + issueAcme + ", " +
			issueWeb + ", " + webState + ", 'Work', 1), (" + opsIssue + ", " + issueAcme + ", " + issueOps + ", " + opsState + ", 'Work', 1)",
		"INSERT INTO issue_activities (id, workspace_id, project_id, issue_id) VALUES (gen_random_uuid(), " + issueAcme + ", " + issueWeb +
			", " + webIssue + ")",
	} {
		if _, err := pool.Exec(context.Background(), stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	return pool
}

// The issue tables' CHECKs accept what the domain writes and reject what
// bypasses it (M4 design 4.2, 4.11): every priority and verb, a start on
// its target date or with no target, a top-level work item, the number 1;
// and refuse an empty title, a priority outside the five, in another case
// too, the number 0, a work item its own parent, a start after its target,
// a verb outside the three.
func TestIssueChecksRejectCounterexamples(t *testing.T) {
	ctx := context.Background()
	pool := newIssueDatabase(t)
	for _, stmt := range []string{
		"UPDATE issues SET priority = 'urgent'", "UPDATE issues SET priority = 'high'", "UPDATE issues SET priority = 'medium'",
		"UPDATE issues SET priority = 'low'", "UPDATE issues SET priority = 'none'",
		"UPDATE issues SET start_date = '2026-10-17', target_date = '2026-10-17'",
		"UPDATE issues SET start_date = '2026-10-18', target_date = NULL",
		"UPDATE issues SET parent_id = NULL", "UPDATE issues SET sequence_id = 1",
		"UPDATE issue_activities SET verb = 'updated'", "UPDATE issue_activities SET verb = 'deleted'",
		"UPDATE issue_activities SET verb = 'created'",
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Errorf("%s: %v, want it accepted", stmt, err)
		}
	}
	for _, tt := range []struct{ name, stmt, constraint string }{
		{"empty title", "UPDATE issues SET name = ''", "issues_name_check"},
		{"unknown priority", "UPDATE issues SET priority = 'other'", "issues_priority_check"},
		{"upper-case priority", "UPDATE issues SET priority = 'Urgent'", "issues_priority_check"},
		{"number 0", "UPDATE issues SET sequence_id = 0", "issues_sequence_id_check"},
		{"its own parent", "UPDATE issues SET parent_id = id", "issues_not_own_parent_check"},
		{"a start after the target", "UPDATE issues SET start_date = '2026-10-18', target_date = '2026-10-17'", "issues_dates_check"},
		{"unknown verb", "UPDATE issue_activities SET verb = 'commented'", "issue_activities_verb_check"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, tt.stmt)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "23514" || pgErr.ConstraintName != tt.constraint {
				t.Errorf("%s = %v, want check_violation (23514) of %s", tt.stmt, err, tt.constraint)
			}
		})
	}
}

// A work item's number is unique in its project among every work item,
// deleted ones too (M4 design 3.7: a number is never used again): a second
// one is refused before and after the first is soft-deleted; another
// project holds its own 1. The assignees', labels' and subscribers' keys
// hold among undeleted rows only (4.3–4.5): a second undeleted row of the
// pair is refused, and soft-deleting the first frees the pair; another
// work item holds the same pair of its own.
func TestIssueUniqueKeys(t *testing.T) {
	ctx := context.Background()
	pool := newIssueDatabase(t)
	refused := func(t *testing.T, stmt, index string) {
		t.Helper()
		_, err := pool.Exec(ctx, stmt)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23505" || pgErr.ConstraintName != index {
			t.Errorf("%s = %v, want unique_violation (23505) of %s", stmt, err, index)
		}
	}
	second := "INSERT INTO issues (id, workspace_id, project_id, state_id, name, sequence_id) VALUES (gen_random_uuid(), " + issueAcme + ", " +
		issueWeb + ", " + webState + ", 'Again', 1)"
	refused(t, second, "issues_project_id_sequence_id_key")
	if _, err := pool.Exec(ctx, "UPDATE issues SET deleted_at = now() WHERE id = "+webIssue); err != nil {
		t.Fatal(err)
	}
	refused(t, second, "issues_project_id_sequence_id_key")

	for _, k := range []struct{ table, column, value, index string }{
		{"issue_assignees", "assignee_id", issueBob, "issue_assignees_issue_id_assignee_id_key"},
		{"issue_labels", "label_id", webBug, "issue_labels_issue_id_label_id_key"},
		{"issue_subscribers", "subscriber_id", issueBob, "issue_subscribers_issue_id_subscriber_id_key"},
	} {
		t.Run(k.table, func(t *testing.T) {
			insert := func(project, issue string) string {
				return "INSERT INTO " + k.table + " (id, workspace_id, project_id, issue_id, " + k.column + ") VALUES (gen_random_uuid(), " +
					issueAcme + ", " + project + ", " + issue + ", " + k.value + ")"
			}
			for _, stmt := range []string{insert(issueWeb, webIssue), insert(issueOps, opsIssue)} {
				if _, err := pool.Exec(ctx, stmt); err != nil {
					t.Fatalf("%s: %v", stmt, err)
				}
			}
			refused(t, insert(issueWeb, webIssue), k.index)
			if _, err := pool.Exec(ctx, "UPDATE "+k.table+" SET deleted_at = now() WHERE issue_id = "+webIssue); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, insert(issueWeb, webIssue)); err != nil {
				t.Errorf("after the first is deleted, %s = %v; want the pair free", insert(issueWeb, webIssue), err)
			}
		})
	}
}

````

- [ ] **Step 3: 测试和 lint**

Run: `go -C server test -count=1 ./migrations/ ./internal/modules/issue/...`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/modules/issue/adapter/postgres/delete_rows_test.go server/internal/modules/issue/adapter/postgres/store_test.go server/migrations/issue_schema_test.go
```
```bash
git commit -m "test(M4/P1): the deletion's batch statements, and the work item tables' checks and keys

Each of the five batch statements takes, in id order, at most the limit
of a project's undeleted rows after the bound, whatever order the rows
lie in, and soft-deletes them at the moment and by the account given,
nothing else; a failed read is a failure, never no rows. The tables'
checks refuse what the domain never writes, a work item's number is
unique in its project for good, and the assignees', labels' and
subscribers' pairs are unique among the undeleted rows only.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p1a.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试；结构是迁移的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是关键词守卫；静态是 `make lint-web` 的文案和键的核对）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T4.1` | 工作项的批次按行的存放次序取行，不按 `id` | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`） | 存储 |
| `T4.2` | 负责人的批次连之前删除的行也取，再盖一次戳 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`） | 存储 |
| `T4.3` | 关注的批次取每个项目的行 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAProject`、`TestAProjectDeletionRefused`（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkspace`、`TestAFailedProjectsStep`、`TestADeletionRefused`（`make test`） | 存储；组合 |
| `T4.4` | 标签的批次保留行的最后写者 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAProject`、`TestAProjectDeletionRefused`（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkspace`、`TestAFailedProjectsStep`、`TestADeletionRefused`（`make test`） | 存储；组合 |
| `T4.5` | 动态的批次不看下界，每次从第一行开始 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`） | 存储 |
| `T4.6` | 工作项的开始日期必须早于目标日期，不能相同 | `migrations` 的 `TestIssueChecksRejectCounterexamples`、`TestIssueUniqueKeys`（`make test`） | 结构 |
| `T4.7` | 动态的动词检查漏掉 `deleted` | `migrations` 的 `TestIssueChecksRejectCounterexamples`、`TestIssueUniqueKeys`（`make test`） | 结构 |
| `T4.8` | 一个标签在一个工作项上只能有一行，已删除的行也算 | `migrations` 的 `TestIssueChecksRejectCounterexamples`、`TestIssueUniqueKeys`（`make test`） | 结构 |
| `T4.9` | 别的大小写的优先级被接受（数据库） | `migrations` 的 `TestIssueChecksRejectCounterexamples`、`TestIssueUniqueKeys`（`make test`） | 结构 |
| `T4.10` | 分批的语句读失败时回答没有行 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`） | 存储 |

---

### Task 5: 创建一侧的规则和动态的第 1、2 行

**Files:**
- Create: `server/internal/modules/issue/domain/activity.go`、`server/internal/modules/issue/domain/activity_test.go`、`server/internal/modules/issue/domain/errors.go`、`server/internal/modules/issue/domain/issue.go`、`server/internal/modules/issue/domain/issue_test.go`

**Interfaces:**（spec 2.5；M4 设计 3.7、3.12、3.13）
- `issue/domain/issue.go`（包说明在这里）：
  - 类型：`Priority`（五个值）、`StateGroup`（`GroupCompleted`、`GroupTriage`）、`Issue`、`NewIssue`；
  - `CheckNewIssue(n) (NewIssue, error)`：名称去掉首尾空白之后 1–255 个字符、没有 NUL；优先级是五个之一；负责人、标签各至多 100 个、不重复（`duplicate`）；全部问题在一个 422 `validation_failed` 里一次报出；
  - `CanAssign(role)`：项目的管理员、成员；
  - `Assignees(asked, defaultAssignee, roles)`：
    - 请求的负责人按 `id` 排序，各须能指派，否则 `assignee_ids[i]` `not_allowed`（`i` 是请求中的位置）；
    - 不带负责人时，项目的默认负责人在他能指派时成为负责人，否则没有；
  - `UnknownLabel(i)`、`StateNotAllowed()`：依赖行的问题；
  - `SortOrderAfter(greatest)`：最大值加 10000，没有时 65535；
  - `CompletedAt(group, now)`："已完成"组为当前时刻，否则没有。
- `issue/domain/activity.go`：
  - `Verb`（`created`、`updated`、`deleted`）、`Activity`、`Assignee`；
  - `CreationActivity(assignees)`：第 1 行 `created`，"created the work item"；第 2 行每个负责人一行：`field` 为 `assignees`、`old_value` 为空串、`new_value` 是他的显示名、`new_identifier` 是他的 `id`，"added assignee "。
- `issue/domain/errors.go`：`ErrProjectNotFound`、`ErrProjectArchived`（码 `project.not_found`、`project.archived`；P1a 没有新的 `issue.*` 码）。

**Tests:**（单元）
- `TestCheckNewIssueAccepts`：去掉首尾空白，边界的长度、个数。
- `TestCheckNewIssueReportsEveryProblem`：每个问题和它的字段、码，几个问题一次报出。
- `TestCanAssign`：每个角色。
- `TestAssignees`：请求的负责人排序；不能指派的报出位置；默认负责人能指派、不能指派、不是成员。
- `TestSortOrderAfter`、`TestCompletedAt`。
- `TestDependentRowProblems`：两个问题的字段和码。
- `TestCreationActivity`：两行的每一列，负责人的次序。

- [ ] **Step 1: 规则、动态、码和测试**

`server/internal/modules/issue/domain/activity.go`（新文件，52 行）：

````file server/internal/modules/issue/domain/activity.go
package domain

import "uuid"

// Verb is what a row of a work item's activity says was done: Plane's
// values, which the page's renderer reads (M4 design 3.13).
type Verb string

// The verbs, the values of issue_activities.verb.
const (
	VerbCreated Verb = "created"
	VerbUpdated Verb = "updated"
	VerbDeleted Verb = "deleted"
)

// Activity is a row of a work item's activity as M4 design 3.13's table
// gives it: the field it is on, its verb, the old and new values and the
// ids they refer to, and Plane's English sentence without its brand, which
// the API does not show (3.13). A nil field, value or id is NULL.
type Activity struct {
	Field         *string
	Verb          Verb
	OldValue      *string
	NewValue      *string
	OldIdentifier *uuid.UUID
	NewIdentifier *uuid.UUID
	Comment       string
}

// Assignee is an account a work item is assigned to, as its activity names
// him: his id and his display name when the row is written.
type Assignee struct {
	ID          uuid.UUID
	DisplayName string
}

// CreationActivity is the rows a work item's creation writes, in order (M4
// design 3.13): row 1, the creation, then row 2 for each of assignees in
// the order given, the accounts it is assigned to, the default assignee
// too. The labels it is created with write none (Plane the same).
func CreationActivity(assignees []Assignee) []Activity {
	rows := []Activity{{Verb: VerbCreated, Comment: "created the work item"}}
	for _, a := range assignees {
		rows = append(rows, Activity{Field: ptr("assignees"), Verb: VerbUpdated, OldValue: ptr(""), NewValue: ptr(a.DisplayName),
			NewIdentifier: ptr(a.ID), Comment: "added assignee "})
	}
	return rows
}

func ptr[T any](v T) *T {
	return &v
}
````

`server/internal/modules/issue/domain/activity_test.go`（新文件，35 行）：

````file server/internal/modules/issue/domain/activity_test.go
package domain

import (
	"reflect"
	"testing"
	"uuid"
)

// A creation writes row 1, the creation, alone with no field, value or
// id; then row 2 for each account it is assigned to, in the order given:
// assignees, updated, from "" to his display name, his id the new
// identifier (M4 design 3.13). Plane's sentences go with them, its brand
// left out.
func TestCreationActivity(t *testing.T) {
	ann, ben := uuid.NewV7(), uuid.NewV7()
	created := Activity{Verb: "created", Comment: "created the work item"}
	assigned := func(id uuid.UUID, name string) Activity {
		field, empty := "assignees", ""
		return Activity{Field: &field, Verb: "updated", OldValue: &empty, NewValue: &name, NewIdentifier: &id, Comment: "added assignee "}
	}
	tests := []struct {
		name      string
		assignees []Assignee
		want      []Activity
	}{
		{"no assignee", nil, []Activity{created}},
		{"two", []Assignee{{ID: ben, DisplayName: "ben"}, {ID: ann, DisplayName: "Ann L."}},
			[]Activity{created, assigned(ben, "ben"), assigned(ann, "Ann L.")}},
	}
	for _, tt := range tests {
		if got := CreationActivity(tt.assignees); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: CreationActivity() = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}
````

`server/internal/modules/issue/domain/errors.go`（新文件，15 行）：

````file server/internal/modules/issue/domain/errors.go
package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The issue module's errors (M4 design 5.3). Its operations on a project
// answer the project module's codes for the project, which they declare
// too (M4 design 5.1), with the project module's details.
var (
	// ErrProjectNotFound answers a project that does not exist, is deleted,
	// or that the caller does not see (M3 design 3.4, 8.2).
	ErrProjectNotFound = shared.NewError(shared.KindNotFound, "project.not_found", "The project does not exist, or you cannot see it.")
	// ErrProjectArchived answers a write on the work items of an archived
	// project (M4 design 3.8).
	ErrProjectArchived = shared.NewError(shared.KindConflict, "project.archived", "The project is archived; unarchive it to change it.")
)
````

`server/internal/modules/issue/domain/issue.go`（新文件，241 行）：

````file server/internal/modules/issue/domain/issue.go
// Package domain holds the issue module's rules (M4 design 6.2): pure
// functions and values.
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

// Priority is a work item's priority, a value of issues.priority (M4
// design 3.7).
type Priority string

// The priorities, from the most to the least urgent.
const (
	PriorityUrgent Priority = "urgent"
	PriorityHigh   Priority = "high"
	PriorityMedium Priority = "medium"
	PriorityLow    Priority = "low"
	PriorityNone   Priority = "none"
)

// priorities are the values issues.priority takes.
var priorities = []Priority{PriorityUrgent, PriorityHigh, PriorityMedium, PriorityLow, PriorityNone}

// StateGroup is the group of a work item's state, as the project module's
// states.group stores it (M3 design 4.9).
type StateGroup string

// The groups the work items' rules name: a work item in the completed
// group is completed (M4 design 3.7); the triage group's state takes no
// work item through the API.
const (
	GroupCompleted StateGroup = "completed"
	GroupTriage    StateGroup = "triage"
)

// Issue is a work item as stored and as the API shows it (M4 design 5.2),
// with the accounts of its undeleted assignees' rows and the labels of its
// undeleted labels' rows, in the accounts' and the labels' id order. The
// fields of later phases come with them (M4 design 12, constraint 5).
type Issue struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	SequenceID  int
	Name        string
	StateID     uuid.UUID
	Priority    Priority
	ParentID    *uuid.UUID
	SortOrder   float64
	CompletedAt *time.Time
	ArchivedAt  *time.Time
	AssigneeIDs []uuid.UUID
	LabelIDs    []uuid.UUID
	CreatedByID *uuid.UUID
	UpdatedByID *uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewIssue is what the caller asks for when creating a work item (M4
// design 5.2, IssueCreate without the fields of later phases): StateID nil
// for the project's default state, Priority empty for none.
type NewIssue struct {
	Name        string
	StateID     *uuid.UUID
	Priority    Priority
	AssigneeIDs []uuid.UUID
	LabelIDs    []uuid.UUID
}

// maxName is the length of issues.name, varchar(255), in characters.
const maxName = 255

// MaxAssignees and MaxLabels are the most accounts and labels a request
// sets on a work item (M4 design 3.7).
const (
	MaxAssignees = 100
	MaxLabels    = 100
)

// sortOrderStep is the gap between the greatest sort order of a state's
// work items and a new one's, and firstSortOrder a state's first work
// item's: Plane's Issue.save() and the column's default (M4 design 3.7).
const (
	sortOrderStep  = 10000
	firstSortOrder = 65535
)

// CheckNewIssue checks n (M4 design 3.7) and returns it as it is stored:
// the name without its surrounding white space, of 1–255 characters and
// without NUL; the priority one of the five, none when it is not given; at
// most MaxAssignees accounts and MaxLabels labels, each named once. Every
// problem is reported at once, in one 422 validation_failed; whether the
// state, each assignee and each label may be is the use case's, under the
// project's lock.
func CheckNewIssue(n NewIssue) (NewIssue, error) {
	n.Name = strings.TrimSpace(n.Name)
	if n.Priority == "" {
		n.Priority = PriorityNone
	}
	found := []*shared.FieldError{checkName(n.Name), checkPriority(n.Priority)}
	found = append(found, checkIDs("assignee_ids", n.AssigneeIDs, MaxAssignees)...)
	found = append(found, checkIDs("label_ids", n.LabelIDs, MaxLabels)...)
	if err := invalid(found...); err != nil {
		return NewIssue{}, err
	}
	return n, nil
}

// assignableRoles are the project roles of the members a work item is
// assigned to: not its guests (M4 design 3.7).
var assignableRoles = []shared.Role{shared.RoleAdmin, shared.RoleMember}

// CanAssign reports whether an active member of a work item's project, of
// project role role, may be assigned to it, as an assignee asked for or as
// the project's default assignee (M4 design 3.7). The set is named, not a
// bound.
func CanAssign(role shared.Role) bool {
	return slices.Contains(assignableRoles, role)
}

// Assignees are the accounts a new work item is assigned to, in id order
// (M4 design 3.7), from roles, the active project roles of the accounts
// asked for and of the project's default assignee, nil for none: those
// asked for, each of whom CanAssign; when none is asked for, the default
// assignee if he may be, else no one, without a problem. Each account asked
// for who may not be is a problem at its index, and then no one is.
func Assignees(asked []uuid.UUID, defaultAssignee *uuid.UUID, roles map[uuid.UUID]shared.Role) ([]uuid.UUID, []shared.FieldError) {
	if len(asked) == 0 {
		if defaultAssignee != nil && CanAssign(roles[*defaultAssignee]) {
			return []uuid.UUID{*defaultAssignee}, nil
		}
		return nil, nil
	}
	var refused []shared.FieldError
	for i, id := range asked {
		if !CanAssign(roles[id]) {
			refused = append(refused, shared.FieldError{Field: fmt.Sprintf("assignee_ids[%d]", i), Code: shared.FieldNotAllowed,
				Message: "must be an active member of the project who is not its guest"})
		}
	}
	if len(refused) > 0 {
		return nil, refused
	}
	return slices.SortedFunc(slices.Values(asked), uuid.UUID.Compare), nil
}

// UnknownLabel is the problem of the i-th label asked for, which is not an
// undeleted label of the project (M4 design 3.7).
func UnknownLabel(i int) shared.FieldError {
	return shared.FieldError{Field: fmt.Sprintf("label_ids[%d]", i), Code: shared.FieldNotAllowed, Message: "must be a label of the project"}
}

// StateNotAllowed is the problem of a state asked for that is not an
// undeleted state of the project, or is its triage state (M4 design 3.7).
func StateNotAllowed() shared.FieldError {
	return shared.FieldError{Field: "state_id", Code: shared.FieldNotAllowed, Message: "must be a state of the project that is not its triage state"}
}

// SortOrderAfter is the sort order of a new work item in a state whose
// work items, archived ones too, have greatest as their greatest sort
// order, nil when it has none: one step after it, or the column's default
// (M4 design 3.7).
func SortOrderAfter(greatest *float64) float64 {
	if greatest == nil {
		return firstSortOrder
	}
	return *greatest + sortOrderStep
}

// CompletedAt is the completed_at of a work item put at now in a state of
// group (M4 design 3.7): now in the completed group, none in any other,
// the cancelled group too.
func CompletedAt(group StateGroup, now time.Time) *time.Time {
	if group != GroupCompleted {
		return nil
	}
	return &now
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

// checkName refuses a name, without its surrounding white space, that is
// empty, longer than issues.name, or with NUL, which Postgres cannot store.
func checkName(name string) *shared.FieldError {
	switch {
	case name == "":
		return &shared.FieldError{Field: "name", Code: shared.FieldTooShort, Message: "must not be empty"}
	case utf8.RuneCountInString(name) > maxName:
		return &shared.FieldError{Field: "name", Code: shared.FieldTooLong, Message: fmt.Sprintf("must be at most %d characters", maxName)}
	case strings.ContainsRune(name, 0):
		return &shared.FieldError{Field: "name", Code: shared.FieldInvalidFormat, Message: "must not contain a NUL character"}
	}
	return nil
}

func checkPriority(p Priority) *shared.FieldError {
	if !slices.Contains(priorities, p) {
		return &shared.FieldError{Field: "priority", Code: shared.FieldInvalidFormat, Message: "must be one of urgent, high, medium, low, none"}
	}
	return nil
}

// checkIDs refuses a list of field longer than limit, as its one problem:
// no id of a list too long is looked at; otherwise each id named before in
// the list, at its index.
func checkIDs(field string, ids []uuid.UUID, limit int) []*shared.FieldError {
	if len(ids) > limit {
		return []*shared.FieldError{{Field: field, Code: shared.FieldTooLong, Message: fmt.Sprintf("must name at most %d", limit)}}
	}
	var found []*shared.FieldError
	named := make(map[uuid.UUID]bool, len(ids))
	for i, id := range ids {
		if named[id] {
			found = append(found, &shared.FieldError{Field: fmt.Sprintf("%s[%d]", field, i), Code: shared.FieldDuplicate, Message: "is listed before"})
		}
		named[id] = true
	}
	return found
}
````

`server/internal/modules/issue/domain/issue_test.go`（新文件，177 行）：

````file server/internal/modules/issue/domain/issue_test.go
package domain

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ids are n fresh ids, in the order drawn: ascending.
func ids(n int) []uuid.UUID {
	out := make([]uuid.UUID, n)
	for i := range out {
		out[i] = uuid.NewV7()
	}
	return out
}

// A name of one character and of 255, each priority, none given, a hundred
// assignees and labels: accepted, and returned as stored: the name without
// its surrounding white space, the priority none when none is given.
func TestCheckNewIssueAccepts(t *testing.T) {
	state := uuid.NewV7()
	tests := []struct {
		name string
		in   NewIssue
		want NewIssue
	}{
		{"one character, trimmed, none", NewIssue{Name: "  x\t"}, NewIssue{Name: "x", Priority: PriorityNone}},
		{"255 characters", NewIssue{Name: strings.Repeat("é", 255), Priority: PriorityLow}, NewIssue{Name: strings.Repeat("é", 255), Priority: PriorityLow}},
		{"everything", NewIssue{Name: "Fix", StateID: &state, Priority: PriorityUrgent, AssigneeIDs: ids(100), LabelIDs: ids(100)},
			NewIssue{Name: "Fix", StateID: &state, Priority: PriorityUrgent}},
	}
	for _, p := range priorities {
		tests = append(tests, struct {
			name string
			in   NewIssue
			want NewIssue
		}{string(p), NewIssue{Name: "a", Priority: p}, NewIssue{Name: "a", Priority: p}})
	}
	for _, tt := range tests {
		got, err := CheckNewIssue(tt.in)
		tt.want.AssigneeIDs, tt.want.LabelIDs = tt.in.AssigneeIDs, tt.in.LabelIDs
		if err != nil || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: CheckNewIssue() = %+v, %v; want %+v", tt.name, got, err, tt.want)
		}
	}
}

// A blank name, one of 256 characters, one with NUL, a priority outside the
// five or in another case, an account or a label named twice, more than a
// hundred of either: each refused, all of them in one 422 that names each by
// its field and place. A list too long is reported alone: its ids are not
// looked at.
func TestCheckNewIssueReportsEveryProblem(t *testing.T) {
	a, b := uuid.NewV7(), uuid.NewV7()
	many := append(ids(MaxAssignees), a)
	many = append(many, a)
	tests := []struct {
		name string
		in   NewIssue
		want []shared.FieldError
	}{
		{"blank", NewIssue{Name: " \n "}, []shared.FieldError{{Field: "name", Code: "too_short", Message: "must not be empty"}}},
		{"256 characters", NewIssue{Name: strings.Repeat("x", 256)},
			[]shared.FieldError{{Field: "name", Code: "too_long", Message: "must be at most 255 characters"}}},
		{"NUL", NewIssue{Name: "a\x00b"}, []shared.FieldError{{Field: "name", Code: "invalid_format", Message: "must not contain a NUL character"}}},
		{"another case", NewIssue{Name: "a", Priority: "Urgent"},
			[]shared.FieldError{{Field: "priority", Code: "invalid_format", Message: "must be one of urgent, high, medium, low, none"}}},
		{"repeats", NewIssue{Name: "a", AssigneeIDs: []uuid.UUID{a, b, a, b, a}, LabelIDs: []uuid.UUID{b, b}}, []shared.FieldError{
			{Field: "assignee_ids[2]", Code: "duplicate", Message: "is listed before"},
			{Field: "assignee_ids[3]", Code: "duplicate", Message: "is listed before"},
			{Field: "assignee_ids[4]", Code: "duplicate", Message: "is listed before"},
			{Field: "label_ids[1]", Code: "duplicate", Message: "is listed before"},
		}},
		{"too many, with repeats, and everything else", NewIssue{Name: "", Priority: "high!", AssigneeIDs: many, LabelIDs: many}, []shared.FieldError{
			{Field: "name", Code: "too_short", Message: "must not be empty"},
			{Field: "priority", Code: "invalid_format", Message: "must be one of urgent, high, medium, low, none"},
			{Field: "assignee_ids", Code: "too_long", Message: "must name at most 100"},
			{Field: "label_ids", Code: "too_long", Message: "must name at most 100"},
		}},
	}
	for _, tt := range tests {
		var e *shared.Error
		if _, err := CheckNewIssue(tt.in); !errors.As(err, &e) || e.Code != "validation_failed" || !reflect.DeepEqual(e.Fields, tt.want) {
			t.Errorf("%s: CheckNewIssue() = %v; want validation_failed with %+v", tt.name, err, tt.want)
		}
	}
}

// A project's admins and members may be assigned; its guests, and accounts
// of any other role or of none, may not (M4 design 3.7).
func TestCanAssign(t *testing.T) {
	for role, want := range map[shared.Role]bool{shared.RoleAdmin: true, shared.RoleMember: true, shared.RoleGuest: false, 0: false, 10: false,
		25: false} {
		if got := CanAssign(role); got != want {
			t.Errorf("CanAssign(%d) = %v, want %v", role, got, want)
		}
	}
}

// The accounts asked for, in id order, when each may be assigned; each who
// may not be, a guest or no member, is a problem at his index, and then no
// one is assigned. None asked for: the default assignee if he is an active
// admin or member, else no one, without a problem (M4 design 3.7).
func TestAssignees(t *testing.T) {
	ann, ben, gus, out := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	roles := map[uuid.UUID]shared.Role{ann: shared.RoleAdmin, ben: shared.RoleMember, gus: shared.RoleGuest}
	notAllowed := func(field string) shared.FieldError {
		return shared.FieldError{Field: field, Code: "not_allowed", Message: "must be an active member of the project who is not its guest"}
	}
	tests := []struct {
		name     string
		asked    []uuid.UUID
		fallback *uuid.UUID
		want     []uuid.UUID
		problems []shared.FieldError
	}{
		{"asked for, in id order; the default is not used", []uuid.UUID{ben, ann}, &gus, []uuid.UUID{ann, ben}, nil},
		{"a guest and no member", []uuid.UUID{ann, gus, ben, out}, &ann, nil, []shared.FieldError{notAllowed("assignee_ids[1]"),
			notAllowed("assignee_ids[3]")}},
		{"none asked for: the default member", nil, &ben, []uuid.UUID{ben}, nil},
		{"none asked for: the default admin", []uuid.UUID{}, &ann, []uuid.UUID{ann}, nil},
		{"none asked for: the default is a guest", nil, &gus, nil, nil},
		{"none asked for: the default left", nil, &out, nil, nil},
		{"none asked for, no default", nil, nil, nil, nil},
	}
	for _, tt := range tests {
		got, problems := Assignees(tt.asked, tt.fallback, roles)
		if !reflect.DeepEqual(got, tt.want) || !reflect.DeepEqual(problems, tt.problems) {
			t.Errorf("%s: Assignees() = %v, %+v; want %v, %+v", tt.name, got, problems, tt.want, tt.problems)
		}
	}
}

// A state's first work item takes 65535, the next one step after the
// greatest (M4 design 3.7).
func TestSortOrderAfter(t *testing.T) {
	greatest := 70000.5
	if got := SortOrderAfter(nil); got != 65535 {
		t.Errorf("SortOrderAfter(nil) = %v, want 65535", got)
	}
	if got := SortOrderAfter(&greatest); got != 80000.5 {
		t.Errorf("SortOrderAfter(70000.5) = %v, want 80000.5", got)
	}
}

// A work item is completed at now in the completed group, and in no other,
// the cancelled group neither (M4 design 3.7).
func TestCompletedAt(t *testing.T) {
	now := time.Date(2026, 10, 17, 10, 0, 0, 0, time.UTC)
	if got := CompletedAt(GroupCompleted, now); got == nil || !got.Equal(now) {
		t.Errorf("CompletedAt(completed) = %v, want %v", got, now)
	}
	for _, g := range []StateGroup{"backlog", "unstarted", "started", "cancelled", GroupTriage} {
		if got := CompletedAt(g, now); got != nil {
			t.Errorf("CompletedAt(%s) = %v, want none", g, got)
		}
	}
}

// A state asked for that is not the project's, or is its triage state, is
// state_id not_allowed; a label that is not the project's, label_ids[i]
// not_allowed (M4 design 3.7).
func TestDependentRowProblems(t *testing.T) {
	if got, want := StateNotAllowed(), (shared.FieldError{Field: "state_id", Code: "not_allowed",
		Message: "must be a state of the project that is not its triage state"}); got != want {
		t.Errorf("StateNotAllowed() = %+v, want %+v", got, want)
	}
	if got, want := UnknownLabel(3), (shared.FieldError{Field: "label_ids[3]", Code: "not_allowed", Message: "must be a label of the project"}); got != want {
		t.Errorf("UnknownLabel(3) = %+v, want %+v", got, want)
	}
}
````

- [ ] **Step 2: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/domain/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`（规则在 Task 8 起才有使用者；它们是导出的，lint 不报未使用）

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 3: 提交**

```bash
git add server/internal/modules/issue/domain/activity.go server/internal/modules/issue/domain/activity_test.go server/internal/modules/issue/domain/errors.go server/internal/modules/issue/domain/issue.go server/internal/modules/issue/domain/issue_test.go
```
```bash
git commit -m "feat(M4/P1): a work item's creation rules and its first two activity rows

A new work item's title is trimmed and 1 to 255 characters, its
priority one of five, its assignees and labels at most a hundred each
and never twice, every problem reported at once. Its assignees, in id
order, must each be able to be assigned: an admin or a member of the
project; without any, the project's default assignee is assigned when
he still can be. Its place is after the greatest of its state's, and a
completed state completes it. Its activity records its creation and
each assignee added, by display name.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p1a.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试；结构是迁移的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是关键词守卫；静态是 `make lint-web` 的文案和键的核对）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T5.1` | 工作项的名称连同首尾的空白存下 | `internal/modules/issue/domain` 的 全部测试（`make test`）、故事 `i1-create-issue`（Task 14 起） | 单元；端到端 |
| `T5.2` | 别的大小写的优先级被接受（领域） | `internal/modules/issue/domain` 的 全部测试（`make test`）、故事 `i1-create-issue`（Task 14 起） | 单元；端到端 |
| `T5.3` | 给两次的账户或标签被接受 | `internal/modules/issue/domain` 的 全部测试（`make test`）、故事 `i1-create-issue`（Task 14 起） | 单元；端到端 |
| `T5.4` | 项目的访客可以被指派 | `internal/modules/issue/domain` 的 全部测试（`make test`）、故事 `i1-create-issue`（Task 14 起） | 单元；端到端 |
| `T5.5` | 默认负责人不论角色都被指派 | `internal/modules/issue/domain` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestRemovingTheDefaultAssigneeAndCreatingAWorkItemSerialize`（`make test`）（Task 12 起）、故事 `i1-create-issue`（Task 14 起） | 单元；组合；端到端 |
| `T5.6` | 指派的账户按请求的次序保存，不按 `id` | `internal/modules/issue/domain` 的 全部测试（`make test`）、故事 `i1-create-issue`（Task 14 起） | 单元；端到端 |
| `T5.7` | "已取消"组里的工作项也算完成 | `internal/modules/issue/domain` 的 全部测试（`make test`）、故事 `i1-create-issue`（Task 14 起） | 单元；端到端 |
| `T5.8` | 一个状态的第一个工作项取步长，不是 65535 | `internal/modules/issue/domain` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `createIssue` 各格（`make test`）（Task 9 起）、故事 `i1-create-issue`（Task 14 起） | 单元；组合；端到端 |
| `T5.9` | 第 2 行的旧值是 NULL，不是渲染器认作"添加"的空串 | `internal/modules/issue/domain` 的 全部测试（`make test`）、故事 `i1-create-issue`（Task 14 起） | 单元；端到端 |
| `T5.10` | 不认识的标签的问题指名整个列表，不指名它在列表中的位置 | `internal/modules/issue/domain` 的 全部测试（`make test`）、`internal/modules/issue/app` 的 全部测试（`make test`）（Task 8 起）、`internal/bootstrap` 的 `TestDeletingALabelAndCreatingAWorkItemWithItSerialize`（`make test`）（Task 13 起）、故事 `i1-create-issue`（Task 14 起） | 单元；组合；端到端 |

---

### Task 6: `project.Provide` 的 `IssueProjects`：项目行的锁、编号、状态、标签、成员的目录

**Files:**
- Create: `server/internal/modules/project/adapter/postgres/issue_ports.go`、`server/internal/modules/project/adapter/postgres/issue_ports_test.go`、`server/internal/modules/project/adapter/postgres/queries/issue_ports.sql`
- Modify: `server/internal/modules/project/adapter/postgres/failures_test.go`、`server/internal/modules/project/app/issue_ports.go`、`server/internal/modules/project/module.go`
- Generate: `server/internal/modules/project/adapter/postgres/gen/issue_ports.sql.go`

**Interfaces:**（spec 2.6；M4 设计 3.5、6.5）
- `queries/issue_ports.sql`：
  - `LockIssueProject`：未删除的项目 `FOR NO KEY UPDATE`，不锁工作区行；回答工作区、是否归档、默认负责人。等锁之后 Postgres 在行的最新版本上重新判断 `deleted_at IS NULL`，所以等锁期间被删除的项目不回答。
  - `NextIssueSequence`：`UPDATE projects SET last_issue_sequence = last_issue_sequence + 1 ... RETURNING`。
  - `IssueStates`：未删除的状态，含分诊，`id` 升序。
  - `IssueLabels`：给出的 `id` 中这个项目未删除的标签，`id` 升序。
  - `ActiveRoles`：给出的账户中这个项目有效成员的角色。
- `Store` 的同名方法（`issue_ports.go`）。
- `project/app/issue_ports.go` 加 `IssueProject`（工作区、`Archived`、`DefaultAssigneeID`）、`IssueState`（`id`、组、`Default`）。
- `module.go`：`IssueProjects` 接口（不加锁的 `ProjectWorkspace` 是 M3 已有的读）、类型别名、`Provided.IssueProjects`。
- 端口只带 P1a 读的东西（spec 第 3 节第 5 条）。

**Tests:**（存储）
- `TestLockIssueProject`：
  - 回答未删除的项目的工作区、是否归档、默认负责人（已归档的也回答），已删除的不回答；
  - 锁住期间，`FOR SHARE` 等它，外键的 `FOR KEY SHARE` 不等；工作区行没有被锁（它的 `FOR UPDATE` 不等）。
- `TestNextIssueSequence`：一个项目的编号 1、2、3，另一个项目从 1 起；已删除的项目答错误，不答编号。
- `TestIssueCatalogs`：
  - 状态含分诊、按 `id`，带组和默认状态；
  - 标签只回答请求中这个项目未删除的；角色只回答请求中有效成员的；
  - 别的项目的状态、标签、成员，已删除的状态、标签，结束或删除的成员关系都不在其中。
- `failures_test.go`：五个方法的失败原样返回。

- [ ] **Step 1: 查询，生成**

`server/internal/modules/project/adapter/postgres/queries/issue_ports.sql`（新文件，41 行）：

````file server/internal/modules/project/adapter/postgres/queries/issue_ports.sql
-- name: LockIssueProject :one
-- ProjectGate.Lock, what project.Provide offers the issue module (M4 design 6.5): the first lock of a write on the
-- work-item side (3.5, decision point 1: B), the project's row FOR NO KEY UPDATE, its workspace's row not locked.
-- After a wait Postgres evaluates deleted_at IS NULL again on the row's newest version, so a project deleted meanwhile
-- reads no row.
SELECT workspace_id, (archived_at IS NOT NULL)::boolean AS archived, default_assignee_id
FROM projects
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
FOR NO KEY UPDATE;

-- name: NextIssueSequence :one
-- Sequences.Next (M4 design 6.5), under the project's FOR NO KEY UPDATE: the project's next work-item number. The
-- counter never goes back, so a number is never used again (3.7).
UPDATE projects
SET last_issue_sequence = last_issue_sequence + 1
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
RETURNING last_issue_sequence;

-- name: IssueStates :many
-- StateCatalog (M4 design 6.5): the project's undeleted states, its triage state too, in id order. Read under the
-- project's FOR NO KEY UPDATE, which every write of a state holds too (M3 design 3.6), it stays as read (M4 design 3.5).
SELECT id, "group", "default"
FROM states
WHERE project_id = sqlc.arg(project_id) AND deleted_at IS NULL
ORDER BY id;

-- name: IssueLabels :many
-- LabelCatalog (M4 design 6.5): those of ids that are the project's undeleted labels, in id order; stable under the
-- project's FOR NO KEY UPDATE, as IssueStates.
SELECT id
FROM labels
WHERE project_id = sqlc.arg(project_id) AND deleted_at IS NULL AND id = ANY (sqlc.arg(ids)::uuid[])
ORDER BY id;

-- name: ActiveRoles :many
-- ProjectMembers (M4 design 6.5): the active memberships of the project of those of the accounts ids that have one,
-- and their roles; stable under the project's FOR NO KEY UPDATE, as IssueStates.
SELECT member_id, role
FROM project_members
WHERE project_id = sqlc.arg(project_id) AND member_id = ANY (sqlc.arg(ids)::uuid[]) AND is_active AND deleted_at IS NULL
ORDER BY member_id;
````

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `ab38ae0fc3fef101ce9ff3018d7a93790b58311b64bbae3d387e6435398ea038` | 160 | `server/internal/modules/project/adapter/postgres/gen/issue_ports.sql.go` |

Run: `shasum -a 256 server/internal/modules/project/adapter/postgres/gen/issue_ports.sql.go`
Expected: 与上表相同。

- [ ] **Step 2: 存储方法、端口的类型和接口**

`server/internal/modules/project/adapter/postgres/issue_ports.go`（新文件，79 行）：

````file server/internal/modules/project/adapter/postgres/issue_ports.go
package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// LockIssueProject locks the undeleted project id FOR NO KEY UPDATE, its
// workspace not locked, and answers it: the first lock of a write on the
// work-item side (M4 design 3.5, 6.5); found is false for a project that
// is not there or is deleted, also when it was deleted while the lock
// waited. It locks in the transaction ctx carries.
func (s *Store) LockIssueProject(ctx context.Context, id uuid.UUID) (p app.IssueProject, found bool, err error) {
	r, err := s.queries(ctx).LockIssueProject(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return app.IssueProject{}, false, nil
	case err != nil:
		return app.IssueProject{}, false, fmt.Errorf("lock project %s for its work items: %w", id, err)
	}
	return app.IssueProject{WorkspaceID: r.WorkspaceID, Archived: r.Archived, DefaultAssigneeID: r.DefaultAssigneeID}, true, nil
}

// NextIssueSequence takes the undeleted project id's next work-item number
// (M4 design 3.7), under its lock: a project that is not there is an
// error, never a number.
func (s *Store) NextIssueSequence(ctx context.Context, id uuid.UUID) (int, error) {
	n, err := s.queries(ctx).NextIssueSequence(ctx, id)
	if err != nil {
		return 0, fmt.Errorf("take the next work-item number of project %s: %w", id, err)
	}
	return int(n), nil
}

// IssueStates reads the undeleted states of the project, its triage state
// too, in id order (M4 design 6.5).
func (s *Store) IssueStates(ctx context.Context, projectID uuid.UUID) ([]app.IssueState, error) {
	rows, err := s.queries(ctx).IssueStates(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("read the states of project %s: %w", projectID, err)
	}
	out := make([]app.IssueState, len(rows))
	for i, r := range rows {
		out[i] = app.IssueState{ID: r.ID, Group: r.Group, Default: r.Default}
	}
	return out, nil
}

// IssueLabels reads those of ids that are undeleted labels of the project,
// in id order (M4 design 6.5).
func (s *Store) IssueLabels(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	found, err := s.queries(ctx).IssueLabels(ctx, gen.IssueLabelsParams{ProjectID: projectID, Ids: ids})
	if err != nil {
		return nil, fmt.Errorf("read labels of project %s: %w", projectID, err)
	}
	return found, nil
}

// ActiveRoles reads the project roles of those of the accounts ids that
// are active members of the project, by account (M4 design 6.5).
func (s *Store) ActiveRoles(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]shared.Role, error) {
	rows, err := s.queries(ctx).ActiveRoles(ctx, gen.ActiveRolesParams{ProjectID: projectID, Ids: ids})
	if err != nil {
		return nil, fmt.Errorf("read the roles of members of project %s: %w", projectID, err)
	}
	roles := make(map[uuid.UUID]shared.Role, len(rows))
	for _, r := range rows {
		roles[r.MemberID] = shared.Role(r.Role)
	}
	return roles, nil
}
````

`server/internal/modules/project/app/issue_ports.go`（修改，1 处）：

````old server/internal/modules/project/app/issue_ports.go
}

````
````new server/internal/modules/project/app/issue_ports.go
}

// IssueProject is a project as project.Provide's lock for the issue
// module answers it (M4 design 6.5, ProjectGate.Lock): its workspace,
// whether it is archived, and its default assignee, nil for none.
// bootstrap converts it into issue's value.
type IssueProject struct {
	WorkspaceID       uuid.UUID
	Archived          bool
	DefaultAssigneeID *uuid.UUID
}

// IssueState is a state of a project as project.Provide's catalog for the
// issue module reads it (M4 design 6.5, StateCatalog): its id, its group,
// and whether it is the project's default. bootstrap converts it into
// issue's value.
type IssueState struct {
	ID      uuid.UUID
	Group   string
	Default bool
}

````

`server/internal/modules/project/module.go`（修改，4 处）：

````old server/internal/modules/project/module.go
// of a project (ProjectAccess) and the workspace module its count of an
// account's ended project memberships (ProjectMembershipCounts).
````
````new server/internal/modules/project/module.go
// of a project (ProjectAccess), the workspace module its count of an
// account's ended project memberships (ProjectMembershipCounts), and the
// issue module the project's lock, numbers and catalogs (IssueProjects).
````
````old server/internal/modules/project/module.go
}

// Provided are the adapters project offers the other modules. They depend
````
````new server/internal/modules/project/module.go
}

// IssueProjects is what the issue module's writes ask of a project (M4
// design 6.5): its gate (ProjectGate's Find and Lock), its work items'
// numbers (Sequences), and its states', labels' and members' catalogs
// (StateCatalog, LabelCatalog, ProjectMembers). Each reads, or locks, in
// the transaction ctx carries; the catalogs are read under the project's
// lock, which every write of a state, a label or a membership holds too,
// so they stay as read until the write commits (M4 design 3.5).
type IssueProjects interface {
	// ProjectWorkspace is the workspace of the undeleted project id,
	// archived or not, read without a lock; found is false when there is
	// none.
	ProjectWorkspace(ctx context.Context, id uuid.UUID) (workspaceID uuid.UUID, found bool, err error)
	// LockIssueProject locks the undeleted project id FOR NO KEY UPDATE,
	// its workspace not locked, and answers it; found is false when there
	// is none, also when it was deleted while the lock waited.
	LockIssueProject(ctx context.Context, id uuid.UUID) (p IssueProject, found bool, err error)
	// NextIssueSequence takes the project's next work-item number.
	NextIssueSequence(ctx context.Context, id uuid.UUID) (int, error)
	// IssueStates reads the project's undeleted states, its triage state
	// too, in id order.
	IssueStates(ctx context.Context, projectID uuid.UUID) ([]IssueState, error)
	// IssueLabels reads those of ids that are the project's undeleted
	// labels, in id order.
	IssueLabels(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error)
	// ActiveRoles reads the project roles of those of the accounts ids
	// that are its active members, by account.
	ActiveRoles(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]shared.Role, error)
}

// IssueProject and IssueState are the values IssueProjects hands over:
// bootstrap converts them into issue's (M4 design 6.5).
type (
	IssueProject = app.IssueProject
	IssueState   = app.IssueState
)

// Provided are the adapters project offers the other modules. They depend
````
````old server/internal/modules/project/module.go
	ProjectMembershipCounts ProjectMembershipCounts
````
````new server/internal/modules/project/module.go
	ProjectMembershipCounts ProjectMembershipCounts
	IssueProjects           IssueProjects
````
````old server/internal/modules/project/module.go
	return Provided{ProjectAccess: store, ProjectMembershipCounts: store}
````
````new server/internal/modules/project/module.go
	return Provided{ProjectAccess: store, ProjectMembershipCounts: store, IssueProjects: store}
````

- [ ] **Step 3: 存储测试和失败测试**

`server/internal/modules/project/adapter/postgres/failures_test.go`（修改，3 处）：

````old server/internal/modules/project/adapter/postgres/failures_test.go
// for a label free to take a parent. Each read runs on a cancelled context
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
// for a label free to take a parent; not "no such project" from the issue
// module's lock, which createIssue would answer as project.not_found, nor
// no states, labels or members, which it would answer as a state, labels
// or assignees not allowed. Each read runs on a cancelled context
````
````old server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("HasChildren() = %v, %v; want context.Canceled, not an answer", has, err)
	}
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("HasChildren() = %v, %v; want context.Canceled, not an answer", has, err)
	}
	if p, found, err := s.LockIssueProject(cancelled, web); !failed(err) || found || p != (app.IssueProject{}) {
		t.Errorf("LockIssueProject() = %+v, %v, %v; want context.Canceled, not no project", p, found, err)
	}
	if list, err := s.IssueStates(cancelled, web); !failed(err) || list != nil {
		t.Errorf("IssueStates() = %+v, %v; want context.Canceled, not none", list, err)
	}
	if list, err := s.IssueLabels(cancelled, web, []uuid.UUID{bug}); !failed(err) || list != nil {
		t.Errorf("IssueLabels() = %v, %v; want context.Canceled, not none", list, err)
	}
	if roles, err := s.ActiveRoles(cancelled, web, []uuid.UUID{alice, bob}); !failed(err) || roles != nil {
		t.Errorf("ActiveRoles() = %v, %v; want context.Canceled, not no member", roles, err)
	}
````
````old server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("UpsertPreferences() = %v; want context.Canceled", err)
	}
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("UpsertPreferences() = %v; want context.Canceled", err)
	}
	if n, err := s.NextIssueSequence(cancelled, web); !failed(err) || n != 0 {
		t.Errorf("NextIssueSequence() = %d, %v; want context.Canceled, not a number", n, err)
	}
````

`server/internal/modules/project/adapter/postgres/issue_ports_test.go`（新文件，155 行）：

````file server/internal/modules/project/adapter/postgres/issue_ports_test.go
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

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The issue module's lock of a project (M4 design 3.5, 6.5). It answers
// the undeleted project's workspace, whether it is archived, and its
// default assignee, an archived one too, none for a deleted one. Until the
// transaction ends the project is held FOR NO KEY UPDATE, which a FOR SHARE
// waits for and a foreign key's FOR KEY SHARE does not, and its
// workspace's row is not locked at all: a FOR UPDATE of it does not wait.
func TestLockIssueProject(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, arch, gone := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Arch", "ARCH", alice), newProject(t, s, acme,
		"Gone", "GONE", alice)
	exec(t, pool, "UPDATE projects SET default_assignee_id = $2 WHERE id = $1", web, alice)
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", arch, now)
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", gone, now)

	for name, tt := range map[string]struct {
		id    uuid.UUID
		want  app.IssueProject
		found bool
	}{
		"Web":  {web, app.IssueProject{WorkspaceID: acme, DefaultAssigneeID: &alice}, true},
		"Arch": {arch, app.IssueProject{WorkspaceID: acme, Archived: true}, true},
		"Gone": {gone, app.IssueProject{}, false},
	} {
		err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
			p, found, err := s.LockIssueProject(ctx, tt.id)
			if err != nil {
				return err
			}
			if found != tt.found || p.WorkspaceID != tt.want.WorkspaceID || p.Archived != tt.want.Archived ||
				!slices.Equal(ids(p.DefaultAssigneeID), ids(tt.want.DefaultAssigneeID)) {
				t.Errorf("LockIssueProject(%s) = %+v, %v; want %+v, %v", name, p, found, tt.want, tt.found)
			}
			if share, keyShare := waits(t, pool, tt.id, "FOR SHARE"), waits(t, pool, tt.id, "FOR KEY SHARE"); share != tt.found || keyShare {
				t.Errorf("%s while locked: a FOR SHARE waits %v, a FOR KEY SHARE waits %v; want %v, false", name, share, keyShare, tt.found)
			}
			_, err = pool.Exec(pgtest.Soon(t), "SELECT 1 FROM workspaces WHERE id = $1 FOR UPDATE NOWAIT", acme)
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "55P03" {
				t.Errorf("%s while locked: a FOR UPDATE of acme waits; want the workspace's row not locked", name)
			} else if err != nil {
				t.Fatal(err)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// ids is the id p points to, alone, or none for nil.
func ids(p *uuid.UUID) []uuid.UUID {
	if p == nil {
		return nil
	}
	return []uuid.UUID{*p}
}

// A project's work items are numbered 1, 2, 3, another project's from 1
// again (M4 design 3.7); a deleted project's number is an error, never a
// number.
func TestNextIssueSequence(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops, gone := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice), newProject(t, s, acme,
		"Gone", "GONE", alice)
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", gone, now)
	ctx := context.Background()
	var got []int
	for _, p := range []uuid.UUID{web, web, ops, web} {
		n, err := s.NextIssueSequence(ctx, p)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, n)
	}
	if want := []int{1, 2, 1, 3}; !slices.Equal(got, want) {
		t.Errorf("NextIssueSequence() of Web, Web, Ops, Web = %v, want %v", got, want)
	}
	if n, err := s.NextIssueSequence(ctx, gone); err == nil || n != 0 {
		t.Errorf("NextIssueSequence() of a deleted project = %d, %v; want an error", n, err)
	}
}

// The catalogs of a project for the issue module (M4 design 6.5): its
// undeleted states, its triage state too, in id order, with their groups
// and the default; those of the labels asked for that are its undeleted
// labels; the roles of the accounts asked for who are its active members.
// Another project's states, labels and members, a deleted state or label,
// an ended or deleted membership are not in them.
func TestIssueCatalogs(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	dave, erin := newAccount(t, pool, "dave@corp.com"), newAccount(t, pool, "erin@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	states := seedDefaultStates(t, s, acme, web, alice)
	seedDefaultStates(t, s, acme, ops, alice)
	gone := uuid.NewV7()
	exec(t, pool, `INSERT INTO states (id, workspace_id, project_id, name, color, "group", deleted_at) VALUES ($1, $2, $3, 'Gone', '#000', 'started', $4)`,
		gone, acme, web, now)
	bug, opsBug, oldBug := seedLabel(t, s, acme, web, nil, "Bug", alice), seedLabel(t, s, acme, ops, nil, "Bug", alice), seedLabel(t, s, acme,
		web, nil, "Old", alice)
	exec(t, pool, "UPDATE labels SET deleted_at = $2 WHERE id = $1", oldBug, now)
	seedMember(t, pool, acme, web, alice, 20, true)
	seedMember(t, pool, acme, web, bob, 15, true)
	seedMember(t, pool, acme, web, carol, 5, true)
	seedMember(t, pool, acme, web, dave, 15, false)
	exec(t, pool, "UPDATE project_members SET deleted_at = $2 WHERE id = $1", seedMember(t, pool, acme, web, erin, 15, true), now)
	seedMember(t, pool, acme, ops, erin, 20, true)
	ctx := context.Background()

	got, err := s.IssueStates(ctx, web)
	groups := map[string]string{"Backlog": "backlog", "Todo": "unstarted", "In Progress": "started", "Done": "completed", "Cancelled": "cancelled",
		"Triage": "triage"}
	var want []app.IssueState
	for name, id := range states {
		want = append(want, app.IssueState{ID: id, Group: groups[name], Default: name == "Backlog"})
	}
	slices.SortFunc(want, func(a, b app.IssueState) int { return a.ID.Compare(b.ID) })
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("IssueStates(Web) = %+v, %v; want %+v", got, err, want)
	}

	if got, err := s.IssueLabels(ctx, web, []uuid.UUID{opsBug, oldBug, bug, uuid.NewV7()}); err != nil || !slices.Equal(got, []uuid.UUID{bug}) {
		t.Errorf("IssueLabels(Web, …) = %v, %v; want only Web's Bug", got, err)
	}
	roles, err := s.ActiveRoles(ctx, web, []uuid.UUID{alice, bob, carol, dave, erin, uuid.NewV7()})
	if want := map[uuid.UUID]shared.Role{alice: shared.RoleAdmin, bob: shared.RoleMember, carol: shared.RoleGuest}; err != nil ||
		!maps.Equal(roles, want) {
		t.Errorf("ActiveRoles(Web, …) = %v, %v; want alice, bob and carol's roles", roles, err)
	}
}
````

- [ ] **Step 4: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/project/...`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`（`IssueProjects` 在 Task 9 起才接给 `issue`；它是导出的）

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 5: 提交**

```bash
git add server/internal/modules/project/adapter/postgres/failures_test.go server/internal/modules/project/adapter/postgres/issue_ports.go server/internal/modules/project/adapter/postgres/issue_ports_test.go server/internal/modules/project/adapter/postgres/queries/issue_ports.sql server/internal/modules/project/app/issue_ports.go server/internal/modules/project/module.go server/internal/modules/project/adapter/postgres/gen/issue_ports.sql.go
```
```bash
git commit -m "feat(M4/P1): the project module offers the work items its lock, numbers and catalogs

IssueProjects is what a work item's write asks of its project: the
project's row FOR NO KEY UPDATE with the workspace's row left alone,
answering its workspace, whether it is archived and its default
assignee, and nothing once it is deleted, also while the lock waited;
the next number from the project's counter; and its states, its labels
among those named, and the roles of its active members among those
named, each read under that lock.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p1a.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试；结构是迁移的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是关键词守卫；静态是 `make lint-web` 的文案和键的核对）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T6.1` | `issue` 模块锁项目取 `FOR SHARE` | `internal/modules/project/adapter/postgres` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestEachIssueWriteLocksItsProjectFirst`（`make test`）（Task 10 起） | 存储；组合 |
| `T6.2` | `issue` 模块锁项目时也锁它的工作区行（sqlc 拒绝：`project` 模块的 schema 没有 `workspaces`） | `make gen-go`（sqlc 拒绝这条查询） | 结构 |
| `T6.3` | `issue` 模块的锁回答已删除的项目 | `internal/modules/project/adapter/postgres` 的 全部测试（`make test`） | 存储 |
| `T6.4` | 项目的下一个编号还是上一个 | `internal/modules/project/adapter/postgres` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestTwoCreationsOnAProjectSerialize`（`make test`）（Task 12 起）、故事 `i1-create-issue`（Task 14 起） | 存储；组合；端到端 |
| `T6.5` | 状态的目录回答已删除的状态 | `internal/modules/project/adapter/postgres` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAStateAndCreatingAWorkItemInItSerialize`（`make test`）（Task 13 起） | 存储；组合 |
| `T6.6` | 标签的目录回答别的项目的标签 | `internal/modules/project/adapter/postgres` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `createIssue` 各格（`make test`）（Task 9 起） | 存储；组合 |
| `T6.7` | 成员的目录回答已结束的成员关系 | `internal/modules/project/adapter/postgres` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestRemovingTheDefaultAssigneeAndCreatingAWorkItemSerialize`（`make test`）（Task 12 起） | 存储；组合 |
| `T6.8` | `issue` 模块的锁失败时回答没有项目 | `internal/modules/project/adapter/postgres` 的 全部测试（`make test`） | 存储 |

---

### Task 7: 创建的存储：排序值、五种插入、详情的重读

**Files:**
- Create: `server/internal/modules/issue/adapter/postgres/issues.go`、`server/internal/modules/issue/adapter/postgres/issues_test.go`、`server/internal/modules/issue/adapter/postgres/queries/issues.sql`、`server/internal/modules/issue/app/rows.go`
- Generate: `server/internal/modules/issue/adapter/postgres/gen/issues.sql.go`

**Interfaces:**（spec 2.7；M4 设计 3.7、3.12、3.13、5.2）
- `queries/issues.sql`：
  - `GreatestSortOrder`：这个项目、这个状态未删除的工作项（含已归档的）的最大值，没有时不回答；
  - 写：`InsertIssue`、`InsertAssignees`、`InsertLabels`、`InsertSubscribers`、`InsertActivity`；
    - 关联表一条语句写全部的行（`unnest`），按给出的次序；
    - `InsertSubscribers` 用 `ON CONFLICT ... DO NOTHING`：已关注的账户保留他的行，名字出现两次的关注一次（部分唯一键跳过这一行）；
  - `GetIssue`：未删除的工作项，带它未删除的负责人、标签，各按 `id` 升序；没有时数组为 NULL（spec 第 3 节第 9 条：`[]` 由 HTTP 的 `orEmpty` 一处负责）。
- `issue/app/rows.go`：`IssueRow`、`LinkRows`、`ActivityRow`（存储写的行；时刻、写者由用例给出）。
- `issue/adapter/postgres/issues.go`：
  - `Store` 的 `GreatestSortOrder`、`InsertIssue`、`InsertAssignees`、`InsertLabels`、`InsertSubscribers`；
  - `InsertActivities`（每行一条 `InsertActivity`）；
  - `GetIssue(ctx, id) (domain.Issue, bool, error)`。

**Tests:**（存储）
- `TestCreatingAWorkItemWritesItsRows`（`createWebIssue`：每一列都不是默认值）：
  - 工作项和它的负责人、标签、关注、动态的行，每一行在工作项下、它的项目和工作区里，由 alice 在 `now`；
  - `GetIssue` 读回负责人、标签，按 `id`，不含已删除的负责人行，不含别的工作项的行。
- `TestInsertSubscribersSubscribesEachAccountOnce`：
  - 已关注的保留原来的行；名字出现两次的只关注一次，取给出的第一行；
  - 关注被删除过的在旁边重新关注。
- `TestGreatestSortOrder`：已归档的计入；已删除的、别的状态的、别的项目的不计入；没有工作项时不回答。
- `TestAFailedIssueStatementIsAnError`：已删除的、不存在的工作项读作没有；读写失败时原样回答失败，从不当作工作项、"没有"或排序值。

- [ ] **Step 1: 查询，生成**

`server/internal/modules/issue/adapter/postgres/queries/issues.sql`（新文件，62 行）：

````file server/internal/modules/issue/adapter/postgres/queries/issues.sql
-- name: GreatestSortOrder :one
-- The greatest sort order of the project's undeleted work items in the state, archived ones too; no row when it has
-- none: a new work item's place after them (M4 design 3.7). Read under the project's FOR NO KEY UPDATE, which every
-- write of a work item holds, so it stays as read until the write commits (3.5).
SELECT sort_order
FROM issues
WHERE project_id = sqlc.arg(project_id) AND state_id = sqlc.arg(state_id) AND deleted_at IS NULL
ORDER BY sort_order DESC
LIMIT 1;

-- name: InsertIssue :exec
-- createIssue's work item (M4 design 3.5), under the project's FOR NO KEY UPDATE; created and last changed at the
-- moment and by the account given. The fields of later phases keep their defaults.
INSERT INTO issues (id, workspace_id, project_id, state_id, name, priority, sequence_id, sort_order, completed_at,
                    created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(state_id), sqlc.arg(name), sqlc.arg(priority),
        sqlc.arg(sequence_id), sqlc.arg(sort_order), sqlc.narg(completed_at), sqlc.arg(by)::uuid, sqlc.arg(by),
        sqlc.arg(at)::timestamptz, sqlc.arg(at));

-- name: InsertAssignees :exec
-- A work item's new assignees' rows, one statement: ids[i] the row of assignee_ids[i], in the order given (the
-- accounts' id order, convention 5's second half), at the moment and by the account given.
INSERT INTO issue_assignees (id, workspace_id, project_id, issue_id, assignee_id, created_by_id, updated_by_id, created_at,
                             updated_at)
SELECT unnest(sqlc.arg(ids)::uuid[]), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(issue_id),
       unnest(sqlc.arg(assignee_ids)::uuid[]), sqlc.arg(by)::uuid, sqlc.arg(by), sqlc.arg(at)::timestamptz, sqlc.arg(at);

-- name: InsertLabels :exec
-- A work item's new labels' rows, one statement, as InsertAssignees.
INSERT INTO issue_labels (id, workspace_id, project_id, issue_id, label_id, created_by_id, updated_by_id, created_at, updated_at)
SELECT unnest(sqlc.arg(ids)::uuid[]), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(issue_id),
       unnest(sqlc.arg(label_ids)::uuid[]), sqlc.arg(by)::uuid, sqlc.arg(by), sqlc.arg(at)::timestamptz, sqlc.arg(at);

-- name: InsertSubscribers :exec
-- A work item's new subscribers' rows, one statement, as InsertAssignees; an account that already subscribes to it, or
-- that the accounts name twice, is subscribed once: the partial unique key skips the row (M4 design 3.12).
INSERT INTO issue_subscribers (id, workspace_id, project_id, issue_id, subscriber_id, created_by_id, updated_by_id, created_at,
                               updated_at)
SELECT unnest(sqlc.arg(ids)::uuid[]), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(issue_id),
       unnest(sqlc.arg(subscriber_ids)::uuid[]), sqlc.arg(by)::uuid, sqlc.arg(by), sqlc.arg(at)::timestamptz, sqlc.arg(at)
ON CONFLICT (issue_id, subscriber_id) WHERE deleted_at IS NULL DO NOTHING;

-- name: InsertActivity :exec
-- A row of a work item's activity (M4 design 3.13): its actor, who also creates it, at the moment given.
INSERT INTO issue_activities (id, workspace_id, project_id, issue_id, verb, field, old_value, new_value, old_identifier,
                              new_identifier, comment, actor_id, created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(issue_id), sqlc.arg(verb), sqlc.narg(field),
        sqlc.narg(old_value), sqlc.narg(new_value), sqlc.narg(old_identifier), sqlc.narg(new_identifier), sqlc.arg(comment),
        sqlc.arg(actor_id)::uuid, sqlc.arg(actor_id), sqlc.arg(actor_id), sqlc.arg(at)::timestamptz, sqlc.arg(at));

-- name: GetIssue :one
-- The undeleted work item, archived or not, as the API shows it (M4 design 5.2): with the accounts of its undeleted
-- assignees' rows and the labels of its undeleted labels' rows, in the accounts' and the labels' id order, NULL for
-- none; an assignee who left the project is still one (3.6).
SELECT i.id, i.workspace_id, i.project_id, i.sequence_id, i.name, i.state_id, i.priority, i.parent_id, i.sort_order,
       i.completed_at, i.archived_at, i.created_by_id, i.updated_by_id, i.created_at, i.updated_at,
       (SELECT array_agg(a.assignee_id ORDER BY a.assignee_id) FROM issue_assignees a
        WHERE a.issue_id = i.id AND a.deleted_at IS NULL)::uuid[] AS assignee_ids,
       (SELECT array_agg(l.label_id ORDER BY l.label_id) FROM issue_labels l
        WHERE l.issue_id = i.id AND l.deleted_at IS NULL)::uuid[] AS label_ids
FROM issues i
WHERE i.id = sqlc.arg(id) AND i.deleted_at IS NULL;
````

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `0047b5a1b7c4505aabc84fda900b9111a8e571fda1556adc74ef39388ddea1f6` | 275 | `server/internal/modules/issue/adapter/postgres/gen/issues.sql.go` |

Run: `shasum -a 256 server/internal/modules/issue/adapter/postgres/gen/issues.sql.go`
Expected: 与上表相同。

- [ ] **Step 2: 行的类型和存储方法**

`server/internal/modules/issue/adapter/postgres/issues.go`（新文件，112 行）：

````file server/internal/modules/issue/adapter/postgres/issues.go
package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// GreatestSortOrder reads the greatest sort order of the project's
// undeleted work items in the state, archived ones too, nil when it has
// none (M4 design 3.7).
func (s *Store) GreatestSortOrder(ctx context.Context, projectID, stateID uuid.UUID) (*float64, error) {
	greatest, err := s.queries(ctx).GreatestSortOrder(ctx, gen.GreatestSortOrderParams{ProjectID: projectID, StateID: stateID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("read the greatest sort order of state %s: %w", stateID, err)
	}
	return &greatest, nil
}

// InsertIssue inserts the work item r. The use case checked every value
// and took its number under the project's lock, so any violation is a bug:
// an internal error (500), not a domain error.
func (s *Store) InsertIssue(ctx context.Context, r app.IssueRow) error {
	err := s.queries(ctx).InsertIssue(ctx, gen.InsertIssueParams{
		ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, StateID: r.StateID, Name: r.Name, Priority: string(r.Priority),
		SequenceID: int32(r.SequenceID), SortOrder: r.SortOrder, CompletedAt: r.CompletedAt, By: &r.By, At: r.At,
	})
	if err != nil {
		return fmt.Errorf("insert work item %s: %w", r.ID, err)
	}
	return nil
}

// InsertAssignees inserts the work item's assignees' rows r.
func (s *Store) InsertAssignees(ctx context.Context, r app.LinkRows) error {
	err := s.queries(ctx).InsertAssignees(ctx, gen.InsertAssigneesParams{
		Ids: r.IDs, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, IssueID: r.IssueID, AssigneeIds: r.Targets, By: &r.By, At: r.At,
	})
	return linked(err, "assignees", r.IssueID)
}

// InsertLabels inserts the work item's labels' rows r.
func (s *Store) InsertLabels(ctx context.Context, r app.LinkRows) error {
	err := s.queries(ctx).InsertLabels(ctx, gen.InsertLabelsParams{
		Ids: r.IDs, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, IssueID: r.IssueID, LabelIds: r.Targets, By: &r.By, At: r.At,
	})
	return linked(err, "labels", r.IssueID)
}

// InsertSubscribers inserts the work item's subscribers' rows r, but of an
// account that subscribes to it already or that r names before (M4 design
// 3.12).
func (s *Store) InsertSubscribers(ctx context.Context, r app.LinkRows) error {
	err := s.queries(ctx).InsertSubscribers(ctx, gen.InsertSubscribersParams{
		Ids: r.IDs, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, IssueID: r.IssueID, SubscriberIds: r.Targets, By: &r.By, At: r.At,
	})
	return linked(err, "subscribers", r.IssueID)
}

// linked is err, an insert of the work item issue's rows of table, as the
// insert's error; nil is nil.
func linked(err error, table string, issue uuid.UUID) error {
	if err != nil {
		return fmt.Errorf("insert the %s of work item %s: %w", table, issue, err)
	}
	return nil
}

// InsertActivities inserts rows, a statement each, in their order: a
// creation writes one more than it has assignees (M4 design 3.13).
func (s *Store) InsertActivities(ctx context.Context, rows []app.ActivityRow) error {
	for _, r := range rows {
		a := r.Activity
		err := s.queries(ctx).InsertActivity(ctx, gen.InsertActivityParams{
			ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, IssueID: r.IssueID, Verb: string(a.Verb), Field: a.Field,
			OldValue: a.OldValue, NewValue: a.NewValue, OldIdentifier: a.OldIdentifier, NewIdentifier: a.NewIdentifier, Comment: a.Comment,
			ActorID: &r.ActorID, At: r.At,
		})
		if err != nil {
			return fmt.Errorf("insert activity of work item %s: %w", r.IssueID, err)
		}
	}
	return nil
}

// GetIssue reads the undeleted work item id, archived or not, with its
// assignees and labels (M4 design 5.2); found is false when there is none.
func (s *Store) GetIssue(ctx context.Context, id uuid.UUID) (domain.Issue, bool, error) {
	r, err := s.queries(ctx).GetIssue(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Issue{}, false, nil
	case err != nil:
		return domain.Issue{}, false, fmt.Errorf("read work item %s: %w", id, err)
	}
	return domain.Issue{
		ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, SequenceID: int(r.SequenceID), Name: r.Name, StateID: r.StateID,
		Priority: domain.Priority(r.Priority), ParentID: r.ParentID, SortOrder: r.SortOrder, CompletedAt: r.CompletedAt,
		ArchivedAt: r.ArchivedAt, AssigneeIDs: r.AssigneeIds, LabelIDs: r.LabelIds, CreatedByID: r.CreatedByID,
		UpdatedByID: r.UpdatedByID, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}, true, nil
}
````

`server/internal/modules/issue/app/rows.go`（新文件，51 行）：

````file server/internal/modules/issue/app/rows.go
package app

import (
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// IssueRow is a work item a write inserts (M4 design 3.5): its values,
// and the moment and the account that create it.
type IssueRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	SequenceID  int
	Name        string
	StateID     uuid.UUID
	Priority    domain.Priority
	SortOrder   float64
	CompletedAt *time.Time
	By          uuid.UUID
	At          time.Time
}

// LinkRows are new rows of one of a work item's tables of accounts or
// labels, its assignees', labels' or subscribers': IDs[i] is the row of
// Targets[i], the account or the label, under the work item, at the moment
// and by the account given.
type LinkRows struct {
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	IssueID     uuid.UUID
	IDs         []uuid.UUID
	Targets     []uuid.UUID
	By          uuid.UUID
	At          time.Time
}

// ActivityRow is a new row of a work item's activity (M4 design 3.13):
// its id, the work item's, the row's values, and its actor, who writes it,
// at the moment given.
type ActivityRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	IssueID     uuid.UUID
	Activity    domain.Activity
	ActorID     uuid.UUID
	At          time.Time
}
````

- [ ] **Step 3: 存储测试**

`server/internal/modules/issue/adapter/postgres/issues_test.go`（新文件，287 行）：

````file server/internal/modules/issue/adapter/postgres/issues_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// newLabel inserts a label of p named name and returns its id. project
// owns the table: the test writes it directly, as a fixture.
func newLabel(t *testing.T, pool *pgxpool.Pool, p project, name string) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	exec(t, pool, "INSERT INTO labels (id, workspace_id, project_id, name) VALUES ($1, $2, $3, $4)", id, p.workspace, p.id, name)
	return id
}

// linkRows are the rows of a link table under the work item issue of p,
// one per target, each with a fresh id, by by at at.
func linkRows(p project, issue uuid.UUID, by uuid.UUID, at time.Time, targets ...uuid.UUID) app.LinkRows {
	r := app.LinkRows{WorkspaceID: p.workspace, ProjectID: p.id, IssueID: issue, Targets: targets, By: by, At: at}
	for range targets {
		r.IDs = append(r.IDs, uuid.NewV7())
	}
	return r
}

// linkTable is every row of table, in id order, as text: its id, its work
// item's, its target in column, its audit columns, whether it is deleted.
func linkTable(t *testing.T, pool *pgxpool.Pool, table, column string) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `SELECT format('%s %s/%s/%s %s by %s/%s at %s/%s deleted %s', id, workspace_id, project_id, issue_id, `+
		column+`, created_by_id, updated_by_id, created_at, updated_at, deleted_at IS NOT NULL) FROM `+table+` ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// linkLine is linkTable's line of a row id of table under the work item
// issue of p, of the target, by by at at, undeleted.
func linkLine(t *testing.T, pool *pgxpool.Pool, id uuid.UUID, p project, issue, target, by uuid.UUID, at time.Time) string {
	t.Helper()
	var stamp string
	if err := pool.QueryRow(context.Background(), "SELECT $1::timestamptz::text", at).Scan(&stamp); err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%s %s/%s/%s %s by %s/%s at %s/%s deleted f", id, p.workspace, p.id, issue, target, by, by, stamp, stamp)
}

// createWebIssue inserts web's work item 7 through the store, by alice at
// now, high, in web's default state, at 75535, completed at an hour
// before: every column createIssue writes, none at its default.
func createWebIssue(t *testing.T, s *postgresadapter.Store, web project, id, alice uuid.UUID) app.IssueRow {
	t.Helper()
	done := now.Add(-time.Hour)
	r := app.IssueRow{ID: id, WorkspaceID: web.workspace, ProjectID: web.id, SequenceID: 7, Name: "Fix the login", StateID: web.state,
		Priority: domain.PriorityHigh, SortOrder: 75535, CompletedAt: &done, By: alice, At: now}
	if err := s.InsertIssue(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	return r
}

// createIssue's rows through the store (M4 design 3.5): the work item with
// every value given, its assignees' and labels' rows, one per target in
// the order given, its subscribers', one per account however often named,
// and its activity's, each row's own; all of them by alice at now, under
// the work item, in its project and workspace. GetIssue reads the work
// item back with its undeleted assignees and labels in their id order,
// but not a deleted assignee's row, nor another work item's rows.
func TestCreatingAWorkItemWritesItsRows(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	bug, ui := newLabel(t, pool, web, "Bug"), newLabel(t, pool, web, "UI")
	id, other := uuid.NewV7(), uuid.NewV7()
	r := createWebIssue(t, s, web, id, alice)
	newIssue(t, pool, web, other, 8, alice)

	assignees, labels := linkRows(web, id, alice, now, carol, bob), linkRows(web, id, alice, now, ui, bug)
	subscribers := linkRows(web, id, alice, now, alice, bob, alice)
	for _, step := range []struct {
		insert func(context.Context, app.LinkRows) error
		rows   app.LinkRows
	}{{s.InsertAssignees, assignees}, {s.InsertLabels, labels}, {s.InsertSubscribers, subscribers},
		{s.InsertAssignees, linkRows(web, other, alice, now, alice)}} {
		if err := step.insert(ctx, step.rows); err != nil {
			t.Fatal(err)
		}
	}
	exec(t, pool, `INSERT INTO issue_assignees (id, workspace_id, project_id, issue_id, assignee_id, deleted_at) VALUES ($1, $2, $3, $4, $5,
		$6)`, uuid.NewV7(), web.workspace, web.id, id, alice, now)
	created, assigned := uuid.NewV7(), uuid.NewV7()
	rows := []app.ActivityRow{
		{ID: created, WorkspaceID: web.workspace, ProjectID: web.id, IssueID: id, ActorID: alice, At: now,
			Activity: domain.CreationActivity(nil)[0]},
		{ID: assigned, WorkspaceID: web.workspace, ProjectID: web.id, IssueID: id, ActorID: alice, At: now,
			Activity: domain.CreationActivity([]domain.Assignee{{ID: bob, DisplayName: "Bob"}})[1]},
	}
	if err := s.InsertActivities(ctx, rows); err != nil {
		t.Fatal(err)
	}

	got, found, err := s.GetIssue(ctx, id)
	want := domain.Issue{ID: id, WorkspaceID: web.workspace, ProjectID: web.id, SequenceID: 7, Name: "Fix the login", StateID: web.state,
		Priority: domain.PriorityHigh, SortOrder: 75535, CompletedAt: r.CompletedAt, CreatedByID: &alice, UpdatedByID: &alice, CreatedAt: now,
		UpdatedAt: now, AssigneeIDs: slices.SortedFunc(slices.Values([]uuid.UUID{bob, carol}), uuid.UUID.Compare),
		LabelIDs: slices.SortedFunc(slices.Values([]uuid.UUID{bug, ui}), uuid.UUID.Compare)}
	if err != nil || !found || !reflect.DeepEqual(got, want) {
		t.Errorf("GetIssue() = %+v, %v, %v; want %+v", got, found, err, want)
	}

	for _, tt := range []struct {
		table, column string
		rows          app.LinkRows
		lines         int // of the rows, those written
	}{{"issue_labels", "label_id", labels, 2}, {"issue_subscribers", "subscriber_id", subscribers, 2}} {
		var want []string
		for i := range tt.lines {
			want = append(want, linkLine(t, pool, tt.rows.IDs[i], web, id, tt.rows.Targets[i], alice, now))
		}
		if got := linkTable(t, pool, tt.table, tt.column); !slices.Equal(got, want) {
			t.Errorf("%s:\n%v\nwant\n%v", tt.table, got, want)
		}
	}
	if got := linkTable(t, pool, "issue_assignees", "assignee_id"); len(got) != 4 || got[0] != linkLine(t, pool, assignees.IDs[0], web, id, carol,
		alice, now) || got[1] != linkLine(t, pool, assignees.IDs[1], web, id, bob, alice, now) {
		t.Errorf("issue_assignees:\n%v\nwant carol's and bob's rows first, as given", got)
	}
	var activity []string
	q, err := pool.Query(ctx, `SELECT format('%s %s/%s/%s %s %L %L→%L %L→%L %L %s/%s/%s %s/%s %L', id, workspace_id, project_id, issue_id, verb,
		field, old_value, new_value, old_identifier, new_identifier, comment, actor_id, created_by_id, updated_by_id, created_at = $1,
		updated_at = $1, deleted_at) FROM issue_activities ORDER BY id`, now)
	if err != nil {
		t.Fatal(err)
	}
	for q.Next() {
		var line string
		if err := q.Scan(&line); err != nil {
			t.Fatal(err)
		}
		activity = append(activity, line)
	}
	if err := q.Err(); err != nil {
		t.Fatal(err)
	}
	place := fmt.Sprintf("%s/%s/%s", web.workspace, web.id, id)
	by := fmt.Sprintf("%s/%s/%s", alice, alice, alice)
	wantActivity := []string{
		fmt.Sprintf("%s %s created NULL NULL→NULL NULL→NULL 'created the work item' %s t/t NULL", created, place, by),
		fmt.Sprintf("%s %s updated 'assignees' ''→'Bob' NULL→'%s' 'added assignee ' %s t/t NULL", assigned, place, bob, by),
	}
	if !slices.Equal(activity, wantActivity) {
		t.Errorf("issue_activities:\n%v\nwant\n%v", activity, wantActivity)
	}
}

// An account that subscribes to the work item already keeps his row and is
// not subscribed again; one named twice is subscribed once, in the first
// row given; one whose subscription was deleted is subscribed anew beside
// it (M4 design 3.12).
func TestInsertSubscribersSubscribesEachAccountOnce(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	id := uuid.NewV7()
	newIssue(t, pool, web, id, 1, alice)
	earlier := now.Add(-time.Hour)
	before := linkRows(web, id, alice, earlier, bob, carol)
	if err := s.InsertSubscribers(context.Background(), before); err != nil {
		t.Fatal(err)
	}
	exec(t, pool, "UPDATE issue_subscribers SET deleted_at = $2 WHERE id = $1", before.IDs[1], earlier)
	again := linkRows(web, id, alice, now, carol, bob, carol)
	if err := s.InsertSubscribers(context.Background(), again); err != nil {
		t.Fatal(err)
	}
	got := linkTable(t, pool, "issue_subscribers", "subscriber_id")
	deleted := linkLine(t, pool, before.IDs[1], web, id, carol, alice, earlier)
	want := []string{linkLine(t, pool, before.IDs[0], web, id, bob, alice, earlier), deleted[:len(deleted)-1] + "t",
		linkLine(t, pool, again.IDs[0], web, id, carol, alice, now)}
	if !slices.Equal(got, want) {
		t.Errorf("issue_subscribers:\n%v\nwant\n%v", got, want)
	}
}

// The greatest sort order of a state's undeleted work items, an archived
// one's too, is where a new one goes after (M4 design 3.7); none for a
// state without work items. A deleted work item, another state's and
// another project's do not count.
func TestGreatestSortOrder(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS")
	other := uuid.NewV7()
	exec(t, pool, "INSERT INTO states (id, workspace_id, project_id, name, color) VALUES ($1, $2, $3, 'Todo', '#000')", other, acme, web.id)
	ctx := context.Background()
	if greatest, err := s.GreatestSortOrder(ctx, web.id, web.state); err != nil || greatest != nil {
		t.Errorf("GreatestSortOrder() of a state without work items = %v, %v; want none", greatest, err)
	}
	for n, w := range []struct {
		p     project
		state uuid.UUID
		sort  float64
		set   string
	}{{web, web.state, 100, ""}, {web, web.state, 300, "archived_at = now()"}, {web, web.state, 900, "deleted_at = now()"},
		{web, other, 500, ""}, {ops, ops.state, 700, ""}} {
		id := uuid.NewV7()
		newIssue(t, pool, w.p, id, n+1, alice)
		exec(t, pool, "UPDATE issues SET state_id = $2, sort_order = $3 WHERE id = $1", id, w.state, w.sort)
		if w.set != "" {
			exec(t, pool, "UPDATE issues SET "+w.set+" WHERE id = $1", id)
		}
	}
	if greatest, err := s.GreatestSortOrder(ctx, web.id, web.state); err != nil || greatest == nil || *greatest != 300 {
		t.Errorf("GreatestSortOrder() = %v, %v; want 300", greatest, err)
	}
}

// A deleted work item, and one that is not there, read as none; a read or
// a write that fails answers its error as itself: never a work item, nor
// "none", nor a sort order (M4 design 3.5).
func TestAFailedIssueStatementIsAnError(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	id := uuid.NewV7()
	newIssue(t, pool, web, id, 1, alice)
	ctx := context.Background()
	if got, found, err := s.GetIssue(ctx, uuid.NewV7()); err != nil || found || got.ID != (uuid.UUID{}) {
		t.Errorf("GetIssue() of no work item = %+v, %v, %v; want none", got, found, err)
	}
	exec(t, pool, "UPDATE issues SET deleted_at = now() WHERE id = $1", id)
	if got, found, err := s.GetIssue(ctx, id); err != nil || found || got.ID != (uuid.UUID{}) {
		t.Errorf("GetIssue() of a deleted work item = %+v, %v, %v; want none", got, found, err)
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if got, found, err := s.GetIssue(cancelled, id); !errors.Is(err, context.Canceled) || found || got.ID != (uuid.UUID{}) {
		t.Errorf("GetIssue() = %+v, %v, %v; want context.Canceled", got, found, err)
	}
	if greatest, err := s.GreatestSortOrder(cancelled, web.id, web.state); !errors.Is(err, context.Canceled) || greatest != nil {
		t.Errorf("GreatestSortOrder() = %v, %v; want context.Canceled", greatest, err)
	}
	if err := s.InsertIssue(cancelled, app.IssueRow{ID: uuid.NewV7(), WorkspaceID: web.workspace, ProjectID: web.id, SequenceID: 2, Name: "a",
		StateID: web.state, Priority: domain.PriorityNone, By: alice, At: now}); !errors.Is(err, context.Canceled) {
		t.Errorf("InsertIssue() = %v; want context.Canceled", err)
	}
	links := linkRows(web, id, alice, now, alice)
	for name, insert := range map[string]func(context.Context, app.LinkRows) error{"InsertAssignees": s.InsertAssignees,
		"InsertLabels": s.InsertLabels, "InsertSubscribers": s.InsertSubscribers} {
		if err := insert(cancelled, links); !errors.Is(err, context.Canceled) {
			t.Errorf("%s() = %v; want context.Canceled", name, err)
		}
	}
	if err := s.InsertActivities(cancelled, []app.ActivityRow{{ID: uuid.NewV7(), WorkspaceID: web.workspace, ProjectID: web.id, IssueID: id,
		ActorID: alice, At: now, Activity: domain.CreationActivity(nil)[0]}}); !errors.Is(err, context.Canceled) {
		t.Errorf("InsertActivities() = %v; want context.Canceled", err)
	}
}
````

- [ ] **Step 4: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/...`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`（存储的这些方法在 Task 8 起才有使用者；它们是导出的）

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 5: 提交**

```bash
git add server/internal/modules/issue/adapter/postgres/issues.go server/internal/modules/issue/adapter/postgres/issues_test.go server/internal/modules/issue/adapter/postgres/queries/issues.sql server/internal/modules/issue/app/rows.go server/internal/modules/issue/adapter/postgres/gen/issues.sql.go
```
```bash
git commit -m "feat(M4/P1): the storage of a work item's creation

The store reads the greatest sort order of a state's work items,
archived ones too, and writes a work item with its assignees', labels',
subscribers' and activity's rows, each link table in one statement; an
account already subscribed keeps his row, and one named twice is
subscribed once. GetIssue reads an undeleted work item back with its
undeleted assignees and labels in id order.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p1a.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试；结构是迁移的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是关键词守卫；静态是 `make lint-web` 的文案和键的核对）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T7.1` | 已删除的工作项的排序值也算进新工作项的 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`） | 存储 |
| `T7.2` | 每个状态的工作项都算进新工作项的排序值 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`）、故事 `i1-create-issue`（Task 14 起） | 存储；端到端 |
| `T7.3` | 给两次的账户、已关注的账户再关注一次 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`） | 存储 |
| `T7.4` | 负责人的行没有写者 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestCreatingAWorkItemStampsItsRows`（`make test`）（Task 10 起） | 存储；组合 |
| `T7.5` | 工作项的读回答已删除的负责人行 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`） | 存储 |
| `T7.6` | 工作项的读按行的次序回答标签，不按标签的 `id` | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`） | 存储 |
| `T7.7` | 动态行的那句话漏掉 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`）、故事 `i1-create-issue`（Task 14 起） | 存储；端到端 |
| `T7.8` | 读到已删除的工作项 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`） | 存储 |
| `T7.9` | 创建的动态只写第一行 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestCreatingAWorkItemStampsItsRows`（`make test`）（Task 10 起）、故事 `i1-create-issue`（Task 14 起） | 存储；组合；端到端 |
| `T7.10` | `GetIssue` 读失败时回答没有这个工作项 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`） | 存储 |

---

### Task 8: 取锁的一步、`createIssue` 的用例和 `issue.create`

**Files:**
- Create: `server/internal/modules/issue/app/create_issue.go`、`server/internal/modules/issue/app/create_issue_test.go`、`server/internal/modules/issue/app/fakes_test.go`、`server/internal/modules/issue/app/lock.go`、`server/internal/modules/issue/app/ports.go`、`server/internal/modules/issue/domain/actions.go`
- Modify: `server/internal/bootstrap/actions_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/issue/app/cascade_test.go`、`server/internal/modules/issue/module.go`

**Interfaces:**（spec 2.8；M4 设计 3.4、3.5、3.7、3.12、3.13）
- `issue/domain/actions.go`：`ActionCreate = "issue.create"`、`Actions()`；`issue/module.go` 的 `Actions()`。
- `access` 的规则表加 `issue.create`（项目级，管理员、成员，不含访客）；`bootstrap/actions_test.go` 的模块清单加 `issue`，"没有动作"的说明删去（设计 12 节约束 4：动作随它的操作）。
- `issue/app/ports.go`（端口由使用方声明）：
  - 项目一侧：`Clock`、`ProjectGate`（`Find`、`Lock`）、`LockedProject`、`Sequences`、`StateCatalog`、`StateRef`、`LabelCatalog`、`ProjectMembers`；
  - `PublicProfiles`；
  - 存储：`IssueCreator`。
- `issue/app/lock.go`：每个工作项的写共用的一条取锁路径。
  - `Locks`（`NewLocks(projects, auth)`）、`write{project, action, notFound}`；
  - `precheck`：`Find` → `decide`；看不到项目时答 `notFound`，角色不允许时答 `shared.Forbidden`；不取锁，不开事务；
  - `lockAndDecide`：`Lock` → `decide`（最终的），回答锁住的项目；
  - `decide`。
- `issue/app/create_issue.go`：`CreateIssueDeps`、`NewCreateIssue`。`Execute` 的次序：
  1. `CheckNewIssue`，再 `precheck`；
  2. 事务里：`lockAndDecide`；已归档答 409；
  3. `check`：
     - 状态：给出的须是项目的、不是分诊的，否则 `state_id` `not_allowed`；不给时用默认状态；
     - 负责人经 `Assignees`，角色经 `ProjectMembers`；
     - 标签经 `LabelCatalog`；
     - 问题一次报出。
  4. `Sequences.Next` → `Clock.Now()` → `GreatestSortOrder`；
  5. 写：工作项、负责人、标签、关注（创建者和负责人，`id` 升序）、动态（负责人的显示名经 `PublicProfiles`，在事务里）；
  6. 重读回答。

**Tests:**（单元，`fakes_test.go` 的假实现把调用按次序记进 `callLog`，带上是否在事务里）
- `TestCreateIssueWritesTheWorkItemAndItsRows`：一次创建的全部调用和它们的次序（设计 3.5）、写下的行。
- `TestCreateIssueDefaults`：
  - 默认状态、优先级 `none`、没有工作项的状态的第一个排序值、不完成；
  - 默认负责人是有效的管理员或成员时被指派、关注、写进动态；是访客、已离开、没有时不指派，不报问题；给出了负责人时不用他。
- `TestCreateIssueRefusesBeforeAnyLock`：
  - 项目不在、看不到时答 `project.not_found`，角色不允许时答 `forbidden`：都在任何锁、事务之前；
  - 违反规则的值在读项目之前就被拒绝。
- `TestCreateIssueRefusesUnderTheLock`：
  - 预检放行，锁下等锁期间被删除、看不到时答 `project.not_found`，角色不再允许时答 `forbidden`；已归档答 409；
  - 依赖行的每个问题在一个 422 里报出；不取编号。
- `TestCreateIssueStopsAtAFailingPort`：每个端口的失败原样返回，之后不再调用；不可能的情况也是错误：没有显示名的负责人（在写动态之前）、没有默认状态的项目（在取号之前）、写入之后读不回来的工作项（假实现的 `lost`）。
- `rules_test.go` 的 `issue.create` 一行；`TestEveryModuleDeclaresItsActionsOrHasNone`。

- [ ] **Step 1: 动作和规则行**

`server/internal/bootstrap/actions_test.go`（修改，3 处）：

````old server/internal/bootstrap/actions_test.go
	"github.com/open-nerve/NerveProject/server/internal/modules/access"
````
````new server/internal/bootstrap/actions_test.go
	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
````
````old server/internal/bootstrap/actions_test.go
		"project":   project.Actions(),
````
````new server/internal/bootstrap/actions_test.go
		"project":   project.Actions(),
		"issue":     issue.Actions(),
````
````old server/internal/bootstrap/actions_test.go
	"access",
	// It answers no operation: the project module's deletions call it, and
	// no rule decides that.
	"issue",
````
````new server/internal/bootstrap/actions_test.go
	"access",
````

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"label.delete": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
````
````new server/internal/modules/access/domain/rules.go
	"label.delete": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	// The project's admins and members, not its guests (M4 design 3.4).
	"issue.create": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"label.delete": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
````
````new server/internal/modules/access/domain/rules_test.go
	"label.delete": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
	// The project's admins and members, not its guests (M4 design 3.4).
	"issue.create": {allowed, allowed, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
````

`server/internal/modules/issue/domain/actions.go`（新文件，16 行）：

````file server/internal/modules/issue/domain/actions.go
package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The issue module's actions: the keys of its rows in the access module's
// rule table (M4 design 3.4).
const (
	// ActionCreate is creating a work item in a project.
	ActionCreate shared.Action = "issue.create"
)

// Actions lists the module's actions. bootstrap's test holds the union of
// every module's Actions equal to the rule table's keys (M3 design 3.4).
func Actions() []shared.Action {
	return []shared.Action{ActionCreate}
}
````

`server/internal/modules/issue/module.go`（修改，2 处）：

````old server/internal/modules/issue/module.go
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
````
````new server/internal/modules/issue/module.go
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
````
````old server/internal/modules/issue/module.go
	return Provided{IssueCascade: app.NewCascade(postgresadapter.New(pool), deleteBatch)}
}

````
````new server/internal/modules/issue/module.go
	return Provided{IssueCascade: app.NewCascade(postgresadapter.New(pool), deleteBatch)}
}

// Actions lists the module's actions: bootstrap's test holds the union of
// every module's actions equal to access's rule table (M3 design 3.4).
func Actions() []shared.Action {
	return domain.Actions()
}

````

- [ ] **Step 2: 端口、取锁的一步和用例**

`server/internal/modules/issue/app/create_issue.go`（新文件，245 行）：

````file server/internal/modules/issue/app/create_issue.go
package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CreateIssueDeps are what createIssue reads and writes through.
type CreateIssueDeps struct {
	Locks     Locks
	Issues    IssueCreator
	Sequences Sequences
	States    StateCatalog
	Labels    LabelCatalog
	Members   ProjectMembers
	Profiles  PublicProfiles
	Tx        shared.TxManager
	Clock     Clock
}

// CreateIssue creates a work item in a project: POST
// /api/v0/projects/{project_id}/issues (M4 design 3.5, 3.7, 5.1).
type CreateIssue struct {
	d CreateIssueDeps
}

// NewCreateIssue returns the use case.
func NewCreateIssue(d CreateIssueDeps) *CreateIssue {
	return &CreateIssue{d: d}
}

// checked is what createIssue's checks of the dependent rows found, under
// the project's lock: the state, the accounts the work item is assigned
// to and its labels, each in id order.
type checked struct {
	state     StateRef
	assignees []uuid.UUID
	labels    []uuid.UUID
}

// Execute checks in (domain.CheckNewIssue) and takes Locks' precheck,
// before any lock; then in one transaction, in the order of M4 design 3.5:
// the project FOR NO KEY UPDATE and the decision on issue.create; an
// archived project refuses it (409 project.archived); the dependent rows
// (check), every problem in one 422; the project's next number; the clock;
// the sort order after the greatest of the state's work items; the work
// item; its assignees, the default one too; its labels; its subscribers,
// the caller and the assignees, in id order; its activity: the creation,
// then each assignee by his display name (domain.CreationActivity). All
// of it is the caller's at that moment. The answer is the work item as
// stored, read back in the transaction.
func (u *CreateIssue) Execute(ctx context.Context, projectID uuid.UUID, in domain.NewIssue) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	if in, err = domain.CheckNewIssue(in); err != nil {
		return domain.Issue{}, err
	}
	w := write{project: projectID, action: domain.ActionCreate, notFound: domain.ErrProjectNotFound}
	if err := u.d.Locks.precheck(ctx, actor, w); err != nil {
		return domain.Issue{}, err
	}
	var created domain.Issue
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		project, err := u.d.Locks.lockAndDecide(ctx, actor, w)
		if err != nil {
			return err
		}
		if project.Archived {
			return domain.ErrProjectArchived
		}
		c, err := u.check(ctx, projectID, project, in)
		if err != nil {
			return err
		}
		sequence, err := u.d.Sequences.Next(ctx, projectID)
		if err != nil {
			return err
		}
		now := u.d.Clock.Now()
		greatest, err := u.d.Issues.GreatestSortOrder(ctx, projectID, c.state.ID)
		if err != nil {
			return err
		}
		row := IssueRow{ID: uuid.NewV7(), WorkspaceID: project.WorkspaceID, ProjectID: projectID, SequenceID: sequence, Name: in.Name,
			StateID: c.state.ID, Priority: in.Priority, SortOrder: domain.SortOrderAfter(greatest),
			CompletedAt: domain.CompletedAt(c.state.Group, now), By: actor.UserID, At: now}
		if err := u.write(ctx, row, c); err != nil {
			return err
		}
		created, err = u.answer(ctx, row.ID)
		return err
	})
	if err != nil {
		return domain.Issue{}, err
	}
	return created, nil
}

// check checks the dependent rows of in under the project's lock (M4
// design 3.7): the state asked for is the project's and not its triage
// state, or the project's default state when none is; the assignees by
// domain.Assignees, from the roles of the accounts asked for, or of the
// project's default assignee when none is; each label asked for is the
// project's. Every problem is reported at once, in one 422.
func (u *CreateIssue) check(ctx context.Context, projectID uuid.UUID, p LockedProject, in domain.NewIssue) (checked, error) {
	var c checked
	var problems []shared.FieldError
	states, err := u.d.States.States(ctx, projectID)
	if err != nil {
		return checked{}, err
	}
	var found bool
	c.state, found = pickState(states, in.StateID)
	switch {
	case !found && in.StateID != nil:
		problems = append(problems, domain.StateNotAllowed())
	case !found:
		return checked{}, fmt.Errorf("project %s has no default state", projectID)
	}
	asked := in.AssigneeIDs
	if len(asked) == 0 && p.DefaultAssigneeID != nil {
		asked = []uuid.UUID{*p.DefaultAssigneeID}
	}
	roles := map[uuid.UUID]shared.Role{}
	if len(asked) > 0 {
		if roles, err = u.d.Members.Roles(ctx, projectID, asked); err != nil {
			return checked{}, err
		}
	}
	var refused []shared.FieldError
	c.assignees, refused = domain.Assignees(in.AssigneeIDs, p.DefaultAssigneeID, roles)
	problems = append(problems, refused...)
	if len(in.LabelIDs) > 0 {
		known, err := u.d.Labels.Labels(ctx, projectID, in.LabelIDs)
		if err != nil {
			return checked{}, err
		}
		for i, id := range in.LabelIDs {
			if !slices.Contains(known, id) {
				problems = append(problems, domain.UnknownLabel(i))
			}
		}
		c.labels = slices.SortedFunc(slices.Values(in.LabelIDs), uuid.UUID.Compare)
	}
	if len(problems) > 0 {
		return checked{}, shared.Invalid(problems...)
	}
	return c, nil
}

// pickState is the state of states asked for, when it is there and not
// the triage state, or the default state when none is asked for; found is
// false when there is no such state.
func pickState(states []StateRef, asked *uuid.UUID) (StateRef, bool) {
	for _, s := range states {
		if asked == nil && s.Default || asked != nil && s.ID == *asked && s.Group != domain.GroupTriage {
			return s, true
		}
	}
	return StateRef{}, false
}

// write writes the work item row and the rows under it, in the order of M4
// design 3.5: the work item, its assignees, its labels, its subscribers,
// its activity.
func (u *CreateIssue) write(ctx context.Context, row IssueRow, c checked) error {
	if err := u.d.Issues.InsertIssue(ctx, row); err != nil {
		return err
	}
	links := func(targets []uuid.UUID) LinkRows {
		r := LinkRows{WorkspaceID: row.WorkspaceID, ProjectID: row.ProjectID, IssueID: row.ID, Targets: targets, By: row.By, At: row.At}
		for range targets {
			r.IDs = append(r.IDs, uuid.NewV7())
		}
		return r
	}
	if len(c.assignees) > 0 {
		if err := u.d.Issues.InsertAssignees(ctx, links(c.assignees)); err != nil {
			return err
		}
	}
	if len(c.labels) > 0 {
		if err := u.d.Issues.InsertLabels(ctx, links(c.labels)); err != nil {
			return err
		}
	}
	subscribers := slices.Compact(slices.SortedFunc(slices.Values(append([]uuid.UUID{row.By}, c.assignees...)), uuid.UUID.Compare))
	if err := u.d.Issues.InsertSubscribers(ctx, links(subscribers)); err != nil {
		return err
	}
	assignees, err := u.named(ctx, c.assignees)
	if err != nil {
		return err
	}
	var rows []ActivityRow
	for _, a := range domain.CreationActivity(assignees) {
		rows = append(rows, ActivityRow{ID: uuid.NewV7(), WorkspaceID: row.WorkspaceID, ProjectID: row.ProjectID, IssueID: row.ID, Activity: a,
			ActorID: row.By, At: row.At})
	}
	return u.d.Issues.InsertActivities(ctx, rows)
}

// named is the accounts ids with their display names, in their order. An
// account without one is an error: an assignee is a member of the project,
// and accounts are never deleted.
func (u *CreateIssue) named(ctx context.Context, ids []uuid.UUID) ([]domain.Assignee, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	names, err := u.d.Profiles.DisplayNames(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Assignee, len(ids))
	for i, id := range ids {
		name, ok := names[id]
		if !ok {
			return nil, fmt.Errorf("assignee %s has no profile", id)
		}
		out[i] = domain.Assignee{ID: id, DisplayName: name}
	}
	return out, nil
}

// answer reads the work item id back as stored, in the transaction ctx
// carries: the write's answer. A work item not there after its write is an
// internal error, not a 404.
func (u *CreateIssue) answer(ctx context.Context, id uuid.UUID) (domain.Issue, error) {
	i, found, err := u.d.Issues.GetIssue(ctx, id)
	switch {
	case err != nil:
		return domain.Issue{}, err
	case !found:
		return domain.Issue{}, errors.New("work item " + id.String() + " is not there after its write")
	}
	return i, nil
}
````

`server/internal/modules/issue/app/lock.go`（新文件，86 行）：

````file server/internal/modules/issue/app/lock.go
package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Locks is the one way a write on the work-item side takes its lock and
// its decision (M4 design 3.5, decision point 1: B). Before any lock and
// outside the write's transaction, the precheck: the project read without
// a lock, and the decision on it, which answers every refusal there, so a
// caller who would be refused takes no lock at all. Then, in the write's
// transaction, the project's row FOR NO KEY UPDATE, its workspace's row
// not locked: the write's first lock, which every write on the project's
// work items, states, labels and memberships also takes, so they run one
// after another; and the decision again under it, which is final: a
// removal or a demotion that committed while the write waited is seen.
// issue.New builds one Locks for every write: a write holds no Authorizer
// of its own.
type Locks struct {
	projects ProjectGate
	auth     shared.Authorizer
}

// NewLocks returns the locks over project's gate and the Authorizer.
func NewLocks(projects ProjectGate, auth shared.Authorizer) Locks {
	return Locks{projects: projects, auth: auth}
}

// write is a write on the work-item side of a project, as Locks takes its
// lock: the project, the action it decides, and the 404 of what the caller
// named.
type write struct {
	project  uuid.UUID
	action   shared.Action
	notFound error
}

// precheck reads w's project without a lock and decides w's action on it
// for actor, before any lock and outside any transaction (M4 design 3.5):
// a project that is not there or not visible to actor is w.notFound; a
// role the rule does not allow is the Authorizer's shared.Forbidden. What
// it lets through, lockAndDecide decides again.
func (l Locks) precheck(ctx context.Context, actor shared.Actor, w write) error {
	workspaceID, found, err := l.projects.Find(ctx, w.project)
	switch {
	case err != nil:
		return err
	case !found:
		return w.notFound
	}
	return decide(ctx, l.auth, actor, w, workspaceID)
}

// lockAndDecide takes w's lock and decides w's action under it, in the
// transaction ctx carries: the project's row FOR NO KEY UPDATE, then the
// decision on the locked project, which it answers. A project deleted
// while the lock waited is w.notFound, and so is one no longer visible to
// actor; a role the rule no longer allows is the Authorizer's
// shared.Forbidden.
func (l Locks) lockAndDecide(ctx context.Context, actor shared.Actor, w write) (LockedProject, error) {
	p, found, err := l.projects.Lock(ctx, w.project)
	switch {
	case err != nil:
		return LockedProject{}, err
	case !found:
		return LockedProject{}, w.notFound
	}
	if err := decide(ctx, l.auth, actor, w, p.WorkspaceID); err != nil {
		return LockedProject{}, err
	}
	return p, nil
}

// decide asks the Authorizer for w's action on w's project of the
// workspace for actor. A project not visible to actor is w.notFound.
func decide(ctx context.Context, auth shared.Authorizer, actor shared.Actor, w write, workspaceID uuid.UUID) error {
	_, err := auth.Authorize(ctx, actor, w.action, shared.Target{WorkspaceID: workspaceID, ProjectID: w.project})
	if errors.Is(err, shared.ErrNotVisible) {
		return w.notFound
	}
	return err
}
````

`server/internal/modules/issue/app/ports.go`（新文件，102 行）：

````file server/internal/modules/issue/app/ports.go
package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Clock reads the moment a write stamps on its rows, under its lock (M4
// design 3.5).
type Clock interface {
	Now() time.Time
}

// ProjectGate is a project's gate of every write on the work-item side (M4
// design 3.5, 6.5): project.Provide's, converted (bootstrap/ports.go).
type ProjectGate interface {
	// Find reads the workspace of the undeleted project, archived or not,
	// without a lock; found is false when there is none.
	Find(ctx context.Context, projectID uuid.UUID) (workspaceID uuid.UUID, found bool, err error)
	// Lock locks the undeleted project FOR NO KEY UPDATE, its workspace's
	// row not locked, in the transaction ctx carries, and answers it;
	// found is false when there is none, also when it was deleted while the
	// lock waited.
	Lock(ctx context.Context, projectID uuid.UUID) (p LockedProject, found bool, err error)
}

// LockedProject is a project as ProjectGate's lock answers it: its
// workspace, whether it is archived, and its default assignee, nil for
// none.
type LockedProject struct {
	WorkspaceID       uuid.UUID
	Archived          bool
	DefaultAssigneeID *uuid.UUID
}

// Sequences numbers a project's work items (M4 design 3.7, 6.5):
// project.Provide's.
type Sequences interface {
	// Next takes the project's next work-item number, under its lock.
	Next(ctx context.Context, projectID uuid.UUID) (int, error)
}

// StateCatalog reads a project's states (M4 design 6.5), under its lock:
// project.Provide's.
type StateCatalog interface {
	// States reads the project's undeleted states, its triage state too.
	States(ctx context.Context, projectID uuid.UUID) ([]StateRef, error)
}

// StateRef is a state as StateCatalog reads it: its id, its group, and
// whether it is the project's default.
type StateRef struct {
	ID      uuid.UUID
	Group   domain.StateGroup
	Default bool
}

// LabelCatalog reads a project's labels (M4 design 6.5), under its lock:
// project.Provide's.
type LabelCatalog interface {
	// Labels reads those of ids that are the project's undeleted labels.
	Labels(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error)
}

// ProjectMembers reads a project's memberships (M4 design 6.5), under its
// lock: project.Provide's.
type ProjectMembers interface {
	// Roles reads the project roles of those of the accounts ids that are
	// its active members, by account.
	Roles(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]shared.Role, error)
}

// PublicProfiles reads accounts' display names (M4 design 6.5): identity's
// PublicProfiles, converted (bootstrap/ports.go).
type PublicProfiles interface {
	// DisplayNames reads the display name of each of the accounts ids that
	// exists, by account.
	DisplayNames(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error)
}

// IssueCreator is the work items' store as createIssue writes and reads
// them, in the transaction ctx carries (M4 design 3.5).
type IssueCreator interface {
	// GreatestSortOrder reads the greatest sort order of the project's
	// undeleted work items in the state, archived ones too, nil for none.
	GreatestSortOrder(ctx context.Context, projectID, stateID uuid.UUID) (*float64, error)
	InsertIssue(ctx context.Context, r IssueRow) error
	InsertAssignees(ctx context.Context, r LinkRows) error
	InsertLabels(ctx context.Context, r LinkRows) error
	// InsertSubscribers inserts the rows, but of an account that
	// subscribes to the work item already.
	InsertSubscribers(ctx context.Context, r LinkRows) error
	// InsertActivities inserts the rows in their order.
	InsertActivities(ctx context.Context, rows []ActivityRow) error
	// GetIssue reads the undeleted work item with its assignees and
	// labels; found is false when there is none.
	GetIssue(ctx context.Context, id uuid.UUID) (i domain.Issue, found bool, err error)
}
````

- [ ] **Step 3: 假实现和单元测试**

`server/internal/modules/issue/app/cascade_test.go`（修改，1 处）：

````old server/internal/modules/issue/app/cascade_test.go
)

var errDisk = errors.New("disk full")

// inTxKey marks the context of the caller's transaction: a call that does
// not carry it ran outside it.
type inTxKey struct{}
````
````new server/internal/modules/issue/app/cascade_test.go
)
````

`server/internal/modules/issue/app/create_issue_test.go`（新文件，317 行）：

````file server/internal/modules/issue/app/create_issue_test.go
package app_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The accounts, drawn in this order, ascending: alice, the caller, and
// carol are web's admins, bob its member, gus its guest; out is not its
// member. acme is web's workspace. web's states are Backlog, the default,
// Done, completed, and the triage state; its labels Bug and UI.
var (
	alice, bob, carol, gus, out = uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	acme, web                   = uuid.NewV7(), uuid.NewV7()
	backlog                     = app.StateRef{ID: uuid.NewV7(), Group: "backlog", Default: true}
	done                        = app.StateRef{ID: uuid.NewV7(), Group: domain.GroupCompleted}
	triage                      = app.StateRef{ID: uuid.NewV7(), Group: domain.GroupTriage}
	bug, ui                     = uuid.NewV7(), uuid.NewV7()
	clockNow                    = time.Date(2026, 10, 17, 10, 0, 0, 123456789, time.UTC)
)

// world is web as createIssue's ports see it: its states, the greatest
// sort order of Done's work items 85535, its labels, its last number 6,
// its members' roles and display names; alice's grant, a project admin's.
type world struct {
	log      *callLog
	project  *fakeProject
	issues   *fakeIssues
	auth     *fakeAuthorizer
	profiles fakeProfiles
}

func newCreate() (*app.CreateIssue, *world) {
	log := &callLog{}
	w := &world{log: log}
	w.project = &fakeProject{log: log, id: web, workspace: acme, found: true, locked: &app.LockedProject{WorkspaceID: acme}, last: 6,
		states: []app.StateRef{backlog, done, triage}, labels: []uuid.UUID{bug, ui},
		roles: map[uuid.UUID]shared.Role{alice: shared.RoleAdmin, bob: shared.RoleMember, carol: shared.RoleAdmin, gus: shared.RoleGuest}}
	w.issues = &fakeIssues{log: log, greatest: map[uuid.UUID]float64{done.ID: 85535}}
	w.auth = &fakeAuthorizer{log: log, grant: shared.Grant{WorkspaceRole: shared.RoleAdmin, ProjectRole: shared.RoleAdmin, ProjectAdmin: true}}
	w.profiles = fakeProfiles{log: log, names: map[uuid.UUID]string{alice: "Alice", bob: "bob", carol: "Carol C.", gus: "gus"}}
	uc := app.NewCreateIssue(app.CreateIssueDeps{Locks: app.NewLocks(w.project, w.auth), Issues: w.issues, Sequences: w.project,
		States: w.project, Labels: w.project, Members: w.project, Profiles: w.profiles, Tx: fakeTx{log: log}, Clock: clockAt{at: clockNow, log: log}})
	return uc, w
}

// as is a context carrying the caller id.
func as(id uuid.UUID) context.Context {
	return shared.WithActor(context.Background(), shared.Actor{UserID: id})
}

// line is a logged call, as callLog writes it.
func line(name string, args ...any) string {
	for _, a := range args {
		name += fmt.Sprint(" ", a)
	}
	return name
}

// created are the calls of a creation by alice in web, the work item id,
// with assignees, labels and names asked for, the state's id, and the
// accounts and labels it is created with, in id order, its subscribers,
// and its activity's rows.
func created(id, state uuid.UUID, asked, labelsAsked, assignees, labels, subscribers []uuid.UUID, rows int) []string {
	calls := []string{
		line("Find", web, "outside tx"), line("Authorize", alice, domain.ActionCreate, acme, web, "outside tx"), "Begin", line("Lock", web),
		line("Authorize", alice, domain.ActionCreate, acme, web), line("States", web),
	}
	if len(asked) > 0 {
		calls = append(calls, line("Roles", web, asked))
	}
	if len(labelsAsked) > 0 {
		calls = append(calls, line("Labels", web, labelsAsked))
	}
	calls = append(calls, line("Next", web), "Now", line("GreatestSortOrder", web, state), line("InsertIssue", id))
	if len(assignees) > 0 {
		calls = append(calls, line("InsertAssignees", assignees))
	}
	if len(labels) > 0 {
		calls = append(calls, line("InsertLabels", labels))
	}
	calls = append(calls, line("InsertSubscribers", subscribers))
	if len(assignees) > 0 {
		calls = append(calls, line("DisplayNames", assignees))
	}
	return append(calls, line("InsertActivities", rows), line("GetIssue", id))
}

// A creation, in the order of M4 design 3.5: before any lock and outside
// the transaction, the project read and the decision; in it, web's lock,
// the decision again, the state, the assignees' roles and the labels
// checked, the next number, then the clock, the sort order after the
// greatest of the state's, the work item, its assignees and labels in id
// order, its subscribers, the caller and the assignees, in id order, then
// the assignees' display names and its activity: the creation and each
// assignee in id order; the answer is the work item read back. The name is
// stored without its surrounding space; the work item in the completed
// state is completed at the moment, which is also every row's, the
// caller's.
func TestCreateIssueWritesTheWorkItemAndItsRows(t *testing.T) {
	uc, w := newCreate()
	got, err := uc.Execute(as(alice), web, domain.NewIssue{Name: "  Fix the login ", StateID: &done.ID, Priority: domain.PriorityHigh,
		AssigneeIDs: []uuid.UUID{carol, bob}, LabelIDs: []uuid.UUID{ui, bug}})
	if err != nil || w.issues.issue == nil {
		t.Fatalf("Execute() = %v, %v; want the work item", got, err)
	}
	id := w.issues.issue.ID
	want := created(id, done.ID, []uuid.UUID{carol, bob}, []uuid.UUID{ui, bug}, []uuid.UUID{bob, carol}, []uuid.UUID{bug, ui},
		[]uuid.UUID{alice, bob, carol}, 3)
	if !slices.Equal(w.log.calls, want) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, want)
	}
	row := app.IssueRow{ID: id, WorkspaceID: acme, ProjectID: web, SequenceID: 7, Name: "Fix the login", StateID: done.ID,
		Priority: domain.PriorityHigh, SortOrder: 95535, CompletedAt: &clockNow, By: alice, At: clockNow}
	if !reflect.DeepEqual(*w.issues.issue, row) {
		t.Errorf("the work item %+v, want %+v", *w.issues.issue, row)
	}
	activity := domain.CreationActivity([]domain.Assignee{{ID: bob, DisplayName: "bob"}, {ID: carol, DisplayName: "Carol C."}})
	if len(w.issues.activities) != len(activity) {
		t.Fatalf("the activity %+v, want %d rows", w.issues.activities, len(activity))
	}
	for i, r := range w.issues.activities {
		wantRow := app.ActivityRow{ID: r.ID, WorkspaceID: acme, ProjectID: web, IssueID: id, Activity: activity[i], ActorID: alice, At: clockNow}
		if !reflect.DeepEqual(r, wantRow) || i > 0 && r.ID.Compare(w.issues.activities[i-1].ID) <= 0 {
			t.Errorf("activity row %d: %+v, want %+v, its id after the row before", i, r, wantRow)
		}
	}
	if answer, _, _ := w.issues.GetIssue(context.Background(), id); !reflect.DeepEqual(got, answer) {
		t.Errorf("Execute() = %+v, want the work item as stored, %+v", got, answer)
	}
}

// What the caller leaves out (M4 design 3.7): the project's default state,
// the priority none, the first sort order of a state without work items,
// no completion; and the assignees: the default assignee when he is an
// active admin or member, assigned, subscribed and in the activity like
// one asked for; no one when he is a guest, has left, or there is none,
// without a problem; the ones asked for, not him, when some are.
func TestCreateIssueDefaults(t *testing.T) {
	for _, tt := range []struct {
		name      string
		fallback  *uuid.UUID
		asked     []uuid.UUID
		read      []uuid.UUID // the roles read
		assignees []uuid.UUID
	}{
		{"no default assignee", nil, nil, nil, nil},
		{"the default assignee, a member", &bob, nil, []uuid.UUID{bob}, []uuid.UUID{bob}},
		{"the default assignee, an admin", &carol, nil, []uuid.UUID{carol}, []uuid.UUID{carol}},
		{"the default assignee, a guest", &gus, nil, []uuid.UUID{gus}, nil},
		{"the default assignee left", &out, nil, []uuid.UUID{out}, nil},
		{"assignees asked for", &bob, []uuid.UUID{carol}, []uuid.UUID{carol}, []uuid.UUID{carol}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w := newCreate()
			w.project.locked.DefaultAssigneeID = tt.fallback
			got, err := uc.Execute(as(alice), web, domain.NewIssue{Name: "Fix", AssigneeIDs: tt.asked})
			if err != nil || w.issues.issue == nil {
				t.Fatalf("Execute() = %v, %v; want the work item", got, err)
			}
			subscribers := slices.SortedFunc(slices.Values(append([]uuid.UUID{alice}, tt.assignees...)), uuid.UUID.Compare)
			if want := created(w.issues.issue.ID, backlog.ID, tt.read, nil, tt.assignees, nil, subscribers, 1+len(tt.assignees)); !slices.Equal(
				w.log.calls, want) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, want)
			}
			r := w.issues.issue
			if r.StateID != backlog.ID || r.Priority != domain.PriorityNone || r.SortOrder != 65535 || r.CompletedAt != nil ||
				!slices.Equal(got.AssigneeIDs, tt.assignees) {
				t.Errorf("the work item %+v, answered %+v; want in Backlog, none, at 65535, not completed, assigned to %v", r, got, tt.assignees)
			}
		})
	}
}

// refused checks a refusal: err is want, the calls are those of a creation
// up to the last one given, and nothing is written, no number taken.
func refused(t *testing.T, w *world, err, want error, calls []string) {
	t.Helper()
	var se, wantSE *shared.Error
	if !errors.As(err, &se) || !errors.As(want, &wantSE) || se.Code != wantSE.Code || !reflect.DeepEqual(se.Fields, wantSE.Fields) {
		t.Errorf("Execute() = %v, want %v", err, want)
	}
	if !slices.Equal(w.log.calls, calls) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
	}
	if w.issues.issue != nil || w.project.last != 6 {
		t.Errorf("after the refusal: the work item %+v, the last number %d; want none and 6", w.issues.issue, w.project.last)
	}
}

// Every refusal the decision makes is answered before any lock and outside
// any transaction (M4 design 3.5): a project not there, or that the
// caller does not see, is project.not_found, a role the rule does not
// allow is forbidden, and nothing is locked. Values that break the rules
// are refused before even the project is read.
func TestCreateIssueRefusesBeforeAnyLock(t *testing.T) {
	precheck := []string{line("Find", web, "outside tx"), line("Authorize", alice, domain.ActionCreate, acme, web, "outside tx")}
	for _, tt := range []struct {
		name  string
		set   func(w *world)
		in    domain.NewIssue
		want  error
		calls []string
	}{
		{"a blank name", func(*world) {}, domain.NewIssue{Name: " "}, shared.Invalid(shared.FieldError{Field: "name", Code: "too_short",
			Message: "must not be empty"}), nil},
		{"no project", func(w *world) { w.project.found = false }, domain.NewIssue{Name: "a"}, domain.ErrProjectNotFound, precheck[:1]},
		{"a project not seen", func(w *world) { w.auth.answers = []error{shared.ErrNotVisible} }, domain.NewIssue{Name: "a"},
			domain.ErrProjectNotFound, precheck},
		{"a role not allowed", func(w *world) { w.auth.answers = []error{shared.Forbidden()} }, domain.NewIssue{Name: "a"}, shared.Forbidden(),
			precheck},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w := newCreate()
			tt.set(w)
			_, err := uc.Execute(as(alice), web, tt.in)
			refused(t, w, err, tt.want, tt.calls)
		})
	}
}

// Under the lock the decision is made again, and is final: a project
// deleted while the lock waited, or no longer seen, is project.not_found,
// a role no longer allowed forbidden; an archived project refuses the
// creation (409 project.archived). Then the dependent rows: a state not
// the project's, or its triage state; an assignee who is its guest or not
// its member; a label not its: every problem in one 422, each at its
// place, and no number taken.
func TestCreateIssueRefusesUnderTheLock(t *testing.T) {
	locked := []string{line("Find", web, "outside tx"), line("Authorize", alice, domain.ActionCreate, acme, web, "outside tx"), "Begin",
		line("Lock", web)}
	decided := append(slices.Clone(locked), line("Authorize", alice, domain.ActionCreate, acme, web))
	other := uuid.NewV7()
	notAllowed := func(field, message string) shared.FieldError {
		return shared.FieldError{Field: field, Code: "not_allowed", Message: message}
	}
	unassignable := "must be an active member of the project who is not its guest"
	for _, tt := range []struct {
		name  string
		set   func(w *world)
		in    domain.NewIssue
		want  error
		calls []string
	}{
		{"deleted while the lock waited", func(w *world) { w.project.locked = nil }, domain.NewIssue{Name: "a"}, domain.ErrProjectNotFound, locked},
		{"no longer seen", func(w *world) { w.auth.answers = []error{nil, shared.ErrNotVisible} }, domain.NewIssue{Name: "a"},
			domain.ErrProjectNotFound, decided},
		{"no longer allowed", func(w *world) { w.auth.answers = []error{nil, shared.Forbidden()} }, domain.NewIssue{Name: "a"}, shared.Forbidden(),
			decided},
		{"archived", func(w *world) { w.project.locked.Archived = true }, domain.NewIssue{Name: "a"}, domain.ErrProjectArchived, decided},
		{"the triage state, a guest, no member, a label not the project's", func(*world) {},
			domain.NewIssue{Name: "a", StateID: &triage.ID, AssigneeIDs: []uuid.UUID{gus, bob, out}, LabelIDs: []uuid.UUID{bug, other}},
			shared.Invalid(notAllowed("state_id", "must be a state of the project that is not its triage state"),
				notAllowed("assignee_ids[0]", unassignable), notAllowed("assignee_ids[2]", unassignable),
				notAllowed("label_ids[1]", "must be a label of the project")),
			append(slices.Clone(decided), line("States", web), line("Roles", web, []uuid.UUID{gus, bob, out}),
				line("Labels", web, []uuid.UUID{bug, other}))},
		{"another project's state", func(*world) {}, domain.NewIssue{Name: "a", StateID: &other},
			shared.Invalid(notAllowed("state_id", "must be a state of the project that is not its triage state")),
			append(slices.Clone(decided), line("States", web))},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w := newCreate()
			tt.set(w)
			_, err := uc.Execute(as(alice), web, tt.in)
			refused(t, w, err, tt.want, tt.calls)
		})
	}
}

// A port that fails ends the creation with its failure, as itself, and no
// call after it runs: the transaction rolls back whatever was written. What
// cannot be is an error too: an assignee without a display name, before the
// activity; a project without a default state, before its number is taken;
// a work item not there to read back after its write.
func TestCreateIssueStopsAtAFailingPort(t *testing.T) {
	for _, port := range []string{"Find", "Authorize", "Lock", "States", "Roles", "Labels", "Next", "GreatestSortOrder", "InsertIssue",
		"InsertAssignees", "InsertLabels", "InsertSubscribers", "DisplayNames", "InsertActivities", "GetIssue"} {
		t.Run(port, func(t *testing.T) {
			uc, w := newCreate()
			w.log.fail = port
			_, err := uc.Execute(as(alice), web, domain.NewIssue{Name: "a", AssigneeIDs: []uuid.UUID{bob}, LabelIDs: []uuid.UUID{bug}})
			last := slices.IndexFunc(w.log.calls, func(c string) bool { return strings.HasPrefix(c, port+" ") })
			if !errors.Is(err, errDisk) || last != len(w.log.calls)-1 {
				t.Errorf("Execute() = %v after the calls\n%q\nwant %v, %s the last call", err, w.log.calls, errDisk, port)
			}
		})
	}
	uc, w := newCreate()
	delete(w.profiles.names, bob)
	if _, err := uc.Execute(as(alice), web, domain.NewIssue{Name: "a", AssigneeIDs: []uuid.UUID{bob}}); err == nil ||
		slices.ContainsFunc(w.log.calls, func(c string) bool { return strings.HasPrefix(c, "InsertActivities") }) {
		t.Errorf("Execute() without bob's display name = %v after %q; want an error before the activity", err, w.log.calls)
	}
	uc, w = newCreate()
	w.project.states = []app.StateRef{done, triage}
	if _, err := uc.Execute(as(alice), web, domain.NewIssue{Name: "a"}); err == nil ||
		slices.ContainsFunc(w.log.calls, func(c string) bool { return strings.HasPrefix(c, "Next") }) {
		t.Errorf("Execute() in a project without a default state = %v after %q; want an error before the number", err, w.log.calls)
	}
	uc, w = newCreate()
	w.issues.lost = true
	if got, err := uc.Execute(as(alice), web, domain.NewIssue{Name: "a"}); err == nil {
		t.Errorf("Execute() whose work item is not there after its write = %+v; want an error", got)
	}
}
````

`server/internal/modules/issue/app/fakes_test.go`（新文件，250 行）：

````file server/internal/modules/issue/app/fakes_test.go
package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var errDisk = errors.New("disk full")

// inTxKey marks the context of the caller's transaction: a call that does
// not carry it ran outside it.
type inTxKey struct{}

// callLog records the calls of the fakes that share it, in order, each with
// its arguments, and " outside tx" when it ran outside a transaction. A
// fake fails the call named in fail with errDisk, after logging it.
type callLog struct {
	calls []string
	fail  string
}

func (l *callLog) add(ctx context.Context, name string, args ...any) error {
	call := name
	for _, a := range args {
		call += fmt.Sprint(" ", a)
	}
	if ctx.Value(inTxKey{}) != true {
		call += " outside tx"
	}
	l.calls = append(l.calls, call)
	if name == l.fail {
		return errDisk
	}
	return nil
}

// fakeTx runs fn in a context marked as inside the transaction.
type fakeTx struct{ log *callLog }

func (f fakeTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	f.log.calls = append(f.log.calls, "Begin")
	return fn(context.WithValue(ctx, inTxKey{}, true))
}

// clockAt stands at an instant; each read is logged as "Now".
type clockAt struct {
	at  time.Time
	log *callLog
}

func (c clockAt) Now() time.Time {
	c.log.calls = append(c.log.calls, "Now")
	return c.at
}

// fakeAuthorizer answers its n-th call answers[n], nil or past the end
// for allowed, with grant; it logs every call.
type fakeAuthorizer struct {
	log     *callLog
	grant   shared.Grant
	answers []error
	calls   int
}

func (f *fakeAuthorizer) Authorize(ctx context.Context, actor shared.Actor, action shared.Action, t shared.Target) (shared.Grant, error) {
	n := f.calls
	f.calls++
	if err := f.log.add(ctx, "Authorize", actor.UserID, action, t.WorkspaceID, t.ProjectID); err != nil {
		return shared.Grant{}, err
	}
	if n < len(f.answers) && f.answers[n] != nil {
		return shared.Grant{}, f.answers[n]
	}
	return f.grant, nil
}

// fakeProject is a project's side of the issue module's ports: its gate,
// its numbers and its catalogs, each call logged. found is whether Find
// answers it, locked what Lock answers, nil for none.
type fakeProject struct {
	log       *callLog
	id        uuid.UUID
	workspace uuid.UUID
	found     bool
	locked    *app.LockedProject
	last      int
	states    []app.StateRef
	labels    []uuid.UUID
	roles     map[uuid.UUID]shared.Role
}

func (f *fakeProject) Find(ctx context.Context, id uuid.UUID) (uuid.UUID, bool, error) {
	if err := f.log.add(ctx, "Find", id); err != nil || id != f.id || !f.found {
		return uuid.UUID{}, false, err
	}
	return f.workspace, true, nil
}

func (f *fakeProject) Lock(ctx context.Context, id uuid.UUID) (app.LockedProject, bool, error) {
	if err := f.log.add(ctx, "Lock", id); err != nil || id != f.id || f.locked == nil {
		return app.LockedProject{}, false, err
	}
	return *f.locked, true, nil
}

func (f *fakeProject) Next(ctx context.Context, id uuid.UUID) (int, error) {
	if err := f.log.add(ctx, "Next", id); err != nil {
		return 0, err
	}
	f.last++
	return f.last, nil
}

func (f *fakeProject) States(ctx context.Context, id uuid.UUID) ([]app.StateRef, error) {
	if err := f.log.add(ctx, "States", id); err != nil {
		return nil, err
	}
	return f.states, nil
}

func (f *fakeProject) Labels(ctx context.Context, id uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	if err := f.log.add(ctx, "Labels", id, ids); err != nil {
		return nil, err
	}
	return slices.DeleteFunc(slices.Clone(ids), func(l uuid.UUID) bool { return !slices.Contains(f.labels, l) }), nil
}

func (f *fakeProject) Roles(ctx context.Context, id uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]shared.Role, error) {
	if err := f.log.add(ctx, "Roles", id, ids); err != nil {
		return nil, err
	}
	out := map[uuid.UUID]shared.Role{}
	for _, a := range ids {
		if r, ok := f.roles[a]; ok {
			out[a] = r
		}
	}
	return out, nil
}

// fakeProfiles answers the display names it has, each call logged.
type fakeProfiles struct {
	log   *callLog
	names map[uuid.UUID]string
}

func (f fakeProfiles) DisplayNames(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	if err := f.log.add(ctx, "DisplayNames", ids); err != nil {
		return nil, err
	}
	out := map[uuid.UUID]string{}
	for _, id := range ids {
		if n, ok := f.names[id]; ok {
			out[id] = n
		}
	}
	return out, nil
}

// fakeIssues is the work items' store: it keeps what it is given, each
// call logged, and answers the work item from it, its assignees and labels
// in id order. greatest is the greatest sort order of each state; lost,
// when set, loses the work item once its activity is written, so that it is
// not there to read back.
type fakeIssues struct {
	log         *callLog
	greatest    map[uuid.UUID]float64
	issue       *app.IssueRow
	assignees   []uuid.UUID
	labels      []uuid.UUID
	subscribers []uuid.UUID
	activities  []app.ActivityRow
	lost        bool
}

func (f *fakeIssues) GreatestSortOrder(ctx context.Context, projectID, stateID uuid.UUID) (*float64, error) {
	if err := f.log.add(ctx, "GreatestSortOrder", projectID, stateID); err != nil {
		return nil, err
	}
	if g, ok := f.greatest[stateID]; ok {
		return &g, nil
	}
	return nil, nil
}

func (f *fakeIssues) InsertIssue(ctx context.Context, r app.IssueRow) error {
	if err := f.log.add(ctx, "InsertIssue", r.ID); err != nil {
		return err
	}
	f.issue = &r
	return nil
}

// link logs an insert of name's rows r and answers their targets, or the
// failure: each row under the work item inserted, by its creator, then.
func (f *fakeIssues) link(ctx context.Context, name string, r app.LinkRows) ([]uuid.UUID, error) {
	if err := f.log.add(ctx, name, r.Targets); err != nil {
		return nil, err
	}
	if f.issue == nil || r.IssueID != f.issue.ID || r.WorkspaceID != f.issue.WorkspaceID || r.ProjectID != f.issue.ProjectID ||
		r.By != f.issue.By || !r.At.Equal(f.issue.At) || len(r.IDs) != len(r.Targets) || slices.Contains(r.IDs, uuid.UUID{}) {
		return nil, fmt.Errorf("%s: %+v not under the work item %+v", name, r, f.issue)
	}
	return r.Targets, nil
}

func (f *fakeIssues) InsertAssignees(ctx context.Context, r app.LinkRows) (err error) {
	f.assignees, err = f.link(ctx, "InsertAssignees", r)
	return err
}

func (f *fakeIssues) InsertLabels(ctx context.Context, r app.LinkRows) (err error) {
	f.labels, err = f.link(ctx, "InsertLabels", r)
	return err
}

func (f *fakeIssues) InsertSubscribers(ctx context.Context, r app.LinkRows) (err error) {
	f.subscribers, err = f.link(ctx, "InsertSubscribers", r)
	return err
}

func (f *fakeIssues) InsertActivities(ctx context.Context, rows []app.ActivityRow) error {
	if err := f.log.add(ctx, "InsertActivities", len(rows)); err != nil {
		return err
	}
	f.activities = rows
	if f.lost {
		f.issue = nil
	}
	return nil
}

func (f *fakeIssues) GetIssue(ctx context.Context, id uuid.UUID) (domain.Issue, bool, error) {
	if err := f.log.add(ctx, "GetIssue", id); err != nil || f.issue == nil || f.issue.ID != id {
		return domain.Issue{}, false, err
	}
	r := f.issue
	return domain.Issue{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, SequenceID: r.SequenceID, Name: r.Name,
		StateID: r.StateID, Priority: r.Priority, SortOrder: r.SortOrder, CompletedAt: r.CompletedAt,
		AssigneeIDs: slices.SortedFunc(slices.Values(f.assignees), uuid.UUID.Compare), LabelIDs: slices.SortedFunc(slices.Values(f.labels),
			uuid.UUID.Compare), CreatedByID: &r.By, UpdatedByID: &r.By, CreatedAt: r.At, UpdatedAt: r.At}, true, nil
}
````

- [ ] **Step 4: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestEveryModuleDeclaresItsActionsOrHasNone$' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`（`NewCreateIssue` 在 Task 9 起才接线；它是导出的）

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 5: 提交**

```bash
git add server/internal/bootstrap/actions_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/issue/app/cascade_test.go server/internal/modules/issue/app/create_issue.go server/internal/modules/issue/app/create_issue_test.go server/internal/modules/issue/app/fakes_test.go server/internal/modules/issue/app/lock.go server/internal/modules/issue/app/ports.go server/internal/modules/issue/domain/actions.go server/internal/modules/issue/module.go
```
```bash
git commit -m "feat(M4/P1): the one way a work item's write takes its lock, and createIssue's use case

Every write on the work-item side goes through Locks: the project read
without a lock and the decision on it first, so every refusal is
answered before any lock is taken; then, in the write's transaction,
the project's row FOR NO KEY UPDATE, the workspace's row left alone,
and the decision again, which is final. Creating a work item then
refuses an archived project, checks its state, assignees and labels
under that lock, takes the next number, reads the clock, and writes the
work item after the greatest of its state's, its assignees, labels and
subscribers, and its activity. issue.create is the project's admins'
and members'.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p1a.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试；结构是迁移的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是关键词守卫；静态是 `make lint-web` 的文案和键的核对）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T8.1` | `createIssue` 没有预检：会被拒绝的调用者先锁住项目 | `internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestEachIssueWriteLocksItsProjectFirst`（`make test`）（Task 10 起） | 单元；组合 |
| `T8.2` | 锁下不再判定一次 | `internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestACallerRemovedOrDemotedWhileHisCreationWaitsIsRefused`（`make test`）（Task 12 起） | 单元；组合 |
| `T8.3` | 已归档的项目接受新的工作项 | `internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestArchivingAProjectAndCreatingAWorkItemSerialize`（`make test`）（Task 12 起） | 单元；组合 |
| `T8.4` | `createIssue` 在锁项目之前读时刻 | `internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestTwoCreationsOnAProjectSerialize`（`make test`）（Task 12 起） | 单元；组合 |
| `T8.5` | 分诊状态接受工作项 | `internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `createIssue, the triage state` 各格（`make test`）（Task 12 起） | 单元；组合 |
| `T8.6` | 创建者不关注他建的工作项 | `internal/modules/issue/app` 的 全部测试（`make test`）、故事 `i1-create-issue`（Task 14 起） | 单元；端到端 |
| `T8.7` | 别的项目的标签被接受 | `internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingALabelAndCreatingAWorkItemWithItSerialize`（`make test`）（Task 13 起）、故事 `i1-create-issue`（Task 14 起）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `createIssue` 各格（`make test`）（Task 9 起） | 单元；组合；端到端 |
| `T8.8` | 在检查依赖的行之前取编号（拒绝时事务回滚，编号随之回滚） | `internal/modules/issue/app` 的 全部测试（`make test`） | 单元 |
| `T8.9` | 调用者看不到的项目答 403，透露它存在 | `internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `createIssue` 各格（`make test`）（Task 9 起）、`internal/bootstrap` 的 `TestEachIssueWriteLocksItsProjectFirst`（`make test`）（Task 10 起） | 单元；组合 |
| `T8.10` | 不读默认负责人的角色：他从不被指派 | `internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestRemovingTheDefaultAssigneeAndCreatingAWorkItemSerialize`（`make test`）（Task 12 起） | 单元；组合 |
| `T8.11` | 依赖的行的问题一次只报一个 | `internal/modules/issue/app` 的 全部测试（`make test`）、故事 `i1-create-issue`（Task 14 起） | 单元；端到端 |
| `T8.12` | 先写动态、再写关注（与全局顺序相反） | `internal/modules/issue/app` 的 全部测试（`make test`） | 单元 |
| `T8.13` | `issue.create` 放进项目的访客 | `internal/modules/access/domain` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `createIssue` 各格（`make test`）（Task 9 起）、故事 `i1-create-issue`（Task 14 起） | 单元；组合；端到端 |
| `T8.14` | `createIssue` 写关注失败时照样往下做 | `internal/modules/issue/app` 的 全部测试（`make test`） | 单元 |
| `T8.15` | 显示名的端口没有回答的负责人，以空的名字写进动态 | `internal/modules/issue/app` 的 全部测试（`make test`） | 单元 |
| `T8.16` | 没有默认状态的项目收下一个不在任何状态里的工作项 | `internal/modules/issue/app` 的 全部测试（`make test`） | 单元 |
| `T8.17` | 写入之后读不回来的工作项被答成一个空的工作项，像是建成了 | `internal/modules/issue/app` 的 全部测试（`make test`） | 单元 |

---

### Task 9: `createIssue` 的接口、接线和矩阵；M3 的最先锁工作区的测试收窄

**Files:**
- Create: `api/modules/issue.yaml`、`server/internal/bootstrap/permission_matrix_issues_test.go`、`server/internal/modules/issue/adapter/http/gen/oapi-codegen.yaml`、`server/internal/modules/issue/adapter/http/handler.go`、`server/internal/modules/issue/adapter/http/handler_test.go`、`server/internal/modules/issue/adapter/http/issues.go`、`server/internal/modules/issue/adapter/http/issues_test.go`
- Modify: `api/modules/project.yaml`、`api/openapi.yaml`、`server/internal/bootstrap/app.go`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/bootstrap/ports.go`、`server/internal/bootstrap/ports_test.go`、`server/internal/bootstrap/project_write_locks_test.go`、`server/internal/modules/issue/module.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/issue/adapter/http/gen/bodyshape.gen.go`、`server/internal/modules/issue/adapter/http/gen/server.gen.go`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**（spec 2.9；M4 设计 5.1–5.3、6.5、6.6、9.2、9.3）
- 契约：
  - `api/modules/issue.yaml`：`createIssue`（`POST /api/v0/projects/{project_id}/issues`）。`x-problem-codes`：`validation_failed`、`project.not_found`、`forbidden`、`project.archived`。描述写明默认状态、默认负责人（"仍是项目中不是访客的有效成员时"）、编号、排序值、完成时间、关注、动态；
  - 结构：`IssuePriority`、`IssueDetail`、`IssueCreate`（不许多余的字段）；
  - `api/openapi.yaml` 的 `issue` 标签和路径；
  - `api/modules/project.yaml`：`Project.identifier`、`default_assignee_id`、`State.default` 的说明改成它们对工作项的作用（M3 收尾交接第 4 节）。
- `issue/adapter/http`：
  - `gen/oapi-codegen.yaml`（生成 `server.gen.go`、`bodyshape.gen.go`）；
  - `handler.go`：`UseCases`、`Register`；
  - `issues.go`：`createIssue`、`issueDetail`、`orEmpty`（"列表从不为 null"的一处）。
- `issue/module.go`：`ProjectPorts`（五个端口）、`LockedProject`、`StateRef`、`StateGroup` 别名、`Deps`、`Module`、`New`、`Register`。
- `bootstrap`：
  - `app.go`：`issue.New(issue.Deps{Pool, Tx, Clock, Authorizer, Projects, Profiles})` 在 `project` 之前，`issues.Register`；
  - `ports.go`：`issueProjects`（转换 `project.IssueProjects`）、`issueProfiles`（转换 `identity` 的 `PublicProfiles`，每个账户的显示名）。
- 矩阵：`permission_matrix_issues_test.go` 的 `issueMatrixRows`（`createIssue` 六行：不带字段、每个字段都给、已归档、另一个项目的状态、另一个项目的标签、指派访客）；`permission_matrix_test.go` 接上它。
- `project_write_locks_test.go`：
  - `writesOnAProject` 去掉标签为 `issue` 的写：它们最先锁项目行，不锁工作区行；M3 的 `TestEachWriteOnAProjectSharesItsWorkspaceFirst` 收窄到项目管理类的写；
  - 形状测试的操作带上真实的标签：`issue` 的写被留在外面时，形状测试失败。

**Tests:**
- HTTP（`apitest.Main("issue")`：两个方向核对每个声明的码都被返回过、返回的码都声明过）：
  - `TestCreateIssue`：请求交给用例；201 带工作项的每个字段，没有的为 `null`，列表为 `[]`；
  - `TestCreateIssueHoldsTheBodyToItsStructure`：缺名称、多余的字段（编号、排序值、描述）、类型不对、不是 id 的 id 都答 `bad_request`，不到用例；
  - `TestCreateIssueRefusals`：用例的每个拒绝和失败。
- 端口的转换：`TestIssueProjectsConvertsProjectsAnswers`、`TestIssueProfilesConvertsIdentitysAnswer`。
- 组合：
  - `TestPermissionMatrix` 的 `createIssue` 六行，每格各在自己的副本上建项目的第一个工作项；"每个字段都给"一行经 `identity` 写动态的显示名；
  - `TestThePermissionMatrixCoversEveryOperation`；
  - `TestEachWriteOnAProjectSharesItsWorkspaceFirst`（收窄后）。

- [ ] **Step 1: 契约和生成的配置，生成**

`api/modules/issue.yaml`（新文件，196 行）：

````file api/modules/issue.yaml
# issue 模块的接口（M4 设计 5.1、5.2）。生成的 Go 代码在
# server/internal/modules/issue/adapter/http/gen（make gen-go）。
openapi: 3.1.0
info:
  title: Nerve issue API
  version: v0
paths:
  /api/v0/projects/{project_id}/issues:
    parameters:
      - $ref: '#/components/parameters/ProjectID'
    post:
      operationId: createIssue
      tags: [issue]
      summary: Create a work item in a project
      description: >-
        For the project's admins and members; not in an archived project
        (project.archived). The name, without the white space around it,
        has 1–255 characters, without NUL, and is stored so; at most 100
        assignees and 100 labels, each named once (validation_failed),
        checked before the project is looked at. A project that does not
        exist, is deleted, or that the caller does not see answers
        project.not_found; one he sees but may not create work items in,
        forbidden; both are answered before the project is locked, and with
        the project locked the role is decided again. Then the state, when
        it is given, is one of the project's, not its triage state; each
        assignee is an active member of the project who is not its guest;
        each label is one of the project's (validation_failed, every
        problem at once). Left out, the state is the project's default, the
        priority none, and the assignee the project's default assignee
        while he is still an active member of it who is not its guest, else
        no one. The work item takes the project's next number, never used
        again, also after a deletion; it comes after its state's work items,
        archived ones too: its sort order is their greatest plus 10000, or
        65535 for the state's first; in a state of the completed group it is
        completed at its creation. The caller and the assignees subscribe
        to it; its activity says it was created, and assigned to each
        assignee. Every row it writes is the caller's, at one moment.
      security: [{bearer: []}]
      x-problem-codes: [validation_failed, project.not_found, forbidden, project.archived]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/IssueCreate'
      responses:
        '201':
          description: The new work item.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IssueDetail'
        default:
          $ref: '#/components/responses/Problem'
components:
  # 与 api/openapi.yaml 的相同：打包时丢掉模块文件的 securitySchemes，oapi-codegen 读这一份（M0-P3 交接 3）
  securitySchemes:
    bearer:
      type: http
      scheme: bearer
  parameters:
    ProjectID:
      name: project_id
      in: path
      required: true
      description: A project's id (Project.id).
      schema:
        type: string
        format: uuid
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
    IssuePriority:
      description: A work item's priority, from the most to the least urgent.
      type: string
      enum: [urgent, high, medium, low, none]
    IssueDetail:
      description: A work item of a project.
      type: object
      additionalProperties: false
      required: [id, workspace_id, project_id, sequence_id, name, state_id, priority, parent_id, sort_order, completed_at, archived_at,
        assignee_ids, label_ids, created_by_id, updated_by_id, created_at, updated_at]
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
        sequence_id:
          description: Its number in its project, from 1; with the project's identifier, as in WEB-12.
          type: integer
        name:
          type: string
        state_id:
          type: string
          format: uuid
        priority:
          $ref: '#/components/schemas/IssuePriority'
        parent_id:
          description: The work item it is under, in the same project; null for none.
          type: [string, 'null']
          format: uuid
        sort_order:
          description: Its place among its state's work items, the lowest first.
          type: number
        completed_at:
          description: When it was put in a state of the completed group; null in any other state.
          type: [string, 'null']
          format: date-time
        archived_at:
          description: When it was archived; null when it is not.
          type: [string, 'null']
          format: date-time
        assignee_ids:
          description: >-
            The accounts it is assigned to, by id; one who has left the
            project since is still one.
          type: array
          items:
            type: string
            format: uuid
        label_ids:
          description: Its labels, by id.
          type: array
          items:
            type: string
            format: uuid
        created_by_id:
          description: The account that created it; null for none.
          type: [string, 'null']
          format: uuid
        updated_by_id:
          description: The account that changed it last; null for none.
          type: [string, 'null']
          format: uuid
        created_at:
          type: string
          format: date-time
        updated_at:
          type: string
          format: date-time
    IssueCreate:
      type: object
      additionalProperties: false
      required: [name]
      properties:
        name:
          description: 1–255 characters without the white space around it, which is not stored.
          type: string
        state_id:
          description: One of the project's states, not its triage state; left out for the project's default state.
          type: string
          format: uuid
        priority:
          $ref: '#/components/schemas/IssuePriority'
        assignee_ids:
          description: >-
            At most 100 accounts, each named once, each an active member of
            the project who is not its guest; left out, or empty, for the
            project's default assignee while he is one, else no one.
          type: array
          items:
            type: string
            format: uuid
        label_ids:
          description: At most 100 of the project's labels, each named once.
          type: array
          items:
            type: string
            format: uuid
````

`api/modules/project.yaml`（修改，3 处）：

````old api/modules/project.yaml
          description: Upper case; unique among the workspace's projects.
````
````new api/modules/project.yaml
          description: Upper case; unique among the workspace's projects; the prefix of its work items' numbers, as in WEB-12.
````
````old api/modules/project.yaml
          description: The account set as the project's default assignee, kept as set; null for none.
````
````new api/modules/project.yaml
          description: >-
            The account set as the project's default assignee, kept as set;
            null for none. A work item created without assignees is assigned
            to him while he is an active member of the project who is not its
            guest.
````
````old api/modules/project.yaml
          description: Whether the state is its project's default state; a project has exactly one.
````
````new api/modules/project.yaml
          description: >-
            Whether the state is its project's default state, which a work
            item created without a state takes; a project has exactly one.
````

`api/openapi.yaml`（修改，2 处）：

````old api/openapi.yaml
    description: Projects, their members, their states and their labels.
````
````new api/openapi.yaml
    description: Projects, their members, their states and their labels.
  - name: issue
    description: Work items.
````
````old api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1labels~1{label_id}'
````
````new api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1labels~1{label_id}'
  /api/v0/projects/{project_id}/issues:
    $ref: 'modules/issue.yaml#/paths/~1api~1v0~1projects~1{project_id}~1issues'
````

`server/internal/modules/issue/adapter/http/gen/oapi-codegen.yaml`（新文件，33 行）：

````file server/internal/modules/issue/adapter/http/gen/oapi-codegen.yaml
# oapi-codegen 配置：api/modules/issue.yaml → server.gen.go。
# 由 make gen-go 在 server/ 下执行，路径相对于 server/。
package: gen
output: internal/modules/issue/adapter/http/gen/server.gen.go
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

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `dd362bd437168d48f72f74bf32066613fcb11e97d859e05ae251d26986c82348` | 3110 | `api/dist/openapi.yaml` |
| `4e15f358c670a732718a4dca773cc63e8ce258d21fc434f202012906b51c3a55` | 26 | `server/internal/modules/issue/adapter/http/gen/bodyshape.gen.go` |
| `af59e9e1eaf4172009a31d71c599824f3a8209f8dc43da3866cadd4c60d2da48` | 420 | `server/internal/modules/issue/adapter/http/gen/server.gen.go` |
| `ab6ad21888db60bf8471ab35593472bdd5dd1a4daac47d3ee87cb685c4cab5ad` | 3505 | `server/internal/modules/project/adapter/http/gen/server.gen.go` |
| `05a8be09f11fabba2cbb374222631e254040643d504b5e3ad990c5adb33e8b96` | 3394 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/issue/adapter/http/gen/bodyshape.gen.go server/internal/modules/issue/adapter/http/gen/server.gen.go server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同（`project` 的 `server.gen.go` 的字段注释来自 `project.yaml` 的三处说明）。

- [ ] **Step 2: HTTP 一侧和模块**

`server/internal/modules/issue/adapter/http/handler.go`（新文件，48 行）：

````file server/internal/modules/issue/adapter/http/handler.go
// Package httpadapter serves the issue module's API: it implements the
// strict server that oapi-codegen generates from api/modules/issue.yaml
// into the gen package, and translates between the generated types and the
// use cases.
package httpadapter

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

// CreateIssueUseCase is app.CreateIssue.
type CreateIssueUseCase interface {
	Execute(ctx context.Context, projectID uuid.UUID, in domain.NewIssue) (domain.Issue, error)
}

// UseCases are the use cases behind the module's operations.
type UseCases struct {
	CreateIssue CreateIssueUseCase
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

`server/internal/modules/issue/adapter/http/handler_test.go`（新文件，106 行）：

````file server/internal/modules/issue/adapter/http/handler_test.go
package httpadapter_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/ratelimit"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Every problem code the module's operations declare must be answered by a
// test here (M2 design 3.11), whichever module produces it (M3 design 9.4).
func TestMain(m *testing.M) { apitest.Main(m, "issue") }

var (
	aliceID = uuid.MustParse("0199a2b4-0000-7000-8000-000000000001")
	acmeID  = uuid.MustParse("0199a2b4-0000-7000-8000-00000000000a")
	webID   = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000a1")
	created = time.Date(2026, 10, 17, 10, 0, 0, 123456000, time.UTC)
	errGone = errors.New("connection reset")
)

// fakeAuth accepts the token "alice" as her account.
type fakeAuth struct{}

func (fakeAuth) Authenticate(ctx context.Context, token string) (context.Context, string, error) {
	if token == "alice" {
		return shared.WithActor(ctx, shared.Actor{UserID: aliceID, SessionID: uuid.NewV7()}), "session:alice", nil
	}
	return nil, "", shared.Unauthenticated()
}

// fakeCreate is createIssue: each call is recorded as "caller project",
// with what it got; it answers answer, or err.
type fakeCreate struct {
	calls  []string
	got    []domain.NewIssue
	answer domain.Issue
	err    error
}

func (f *fakeCreate) Execute(ctx context.Context, projectID uuid.UUID, in domain.NewIssue) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+projectID.String())
	f.got = append(f.got, in)
	return f.answer, f.err
}

// newServer serves the module's API over create, as bootstrap mounts it.
func newServer(t *testing.T, create *fakeCreate) http.Handler {
	t.Helper()
	logger := slog.New(slog.DiscardHandler)
	router := httpserver.NewRouter(logger)
	limit := ratelimit.New(time.Now).Bucket("test", ratelimit.Rate{PerMinute: 600, Burst: 100})
	api, err := httpserver.NewAPI(httpserver.APIConfig{
		Logger: logger, Authenticator: fakeAuth{}, MaxBodyBytes: 1024, RequestTimeout: 5 * time.Second, IPv6PrefixLen: 64,
		Anonymous: limit, Authenticated: limit, AuthFailure: limit,
	})
	if err != nil {
		t.Fatal(err)
	}
	httpadapter.Register(router, api, httpadapter.UseCases{CreateIssue: create})
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

`server/internal/modules/issue/adapter/http/issues.go`（新文件，55 行）：

````file server/internal/modules/issue/adapter/http/issues.go
package httpadapter

import (
	"context"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// CreateIssue serves POST /api/v0/projects/{project_id}/issues: the state
// nil, the priority empty and the lists empty when not given.
func (h handler) CreateIssue(ctx context.Context, req gen.CreateIssueRequestObject) (gen.CreateIssueResponseObject, error) {
	in := domain.NewIssue{Name: req.Body.Name, StateID: req.Body.StateID}
	if req.Body.Priority != nil {
		in.Priority = domain.Priority(*req.Body.Priority)
	}
	if req.Body.AssigneeIds != nil {
		in.AssigneeIDs = *req.Body.AssigneeIds
	}
	if req.Body.LabelIds != nil {
		in.LabelIDs = *req.Body.LabelIds
	}
	i, err := h.uc.CreateIssue.Execute(ctx, req.ProjectID, in)
	if err != nil {
		return nil, err
	}
	return gen.CreateIssue201JSONResponse(issueDetail(i)), nil
}

// issueDetail is i as the API shows it: each field that may be null null
// when it is nil, each list [] when it is empty.
func issueDetail(i domain.Issue) gen.IssueDetail {
	return gen.IssueDetail{ID: i.ID, WorkspaceID: i.WorkspaceID, ProjectID: i.ProjectID, SequenceID: i.SequenceID, Name: i.Name,
		StateID: i.StateID, Priority: gen.IssuePriority(i.Priority), ParentID: orNull(i.ParentID), SortOrder: i.SortOrder,
		CompletedAt: orNull(i.CompletedAt), ArchivedAt: orNull(i.ArchivedAt), AssigneeIds: orEmpty(i.AssigneeIDs), LabelIds: orEmpty(i.LabelIDs),
		CreatedByID: orNull(i.CreatedByID), UpdatedByID: orNull(i.UpdatedByID), CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt}
}

// orNull is v as a nullable field: null for nil.
func orNull[T any](v *T) nullable.Nullable[T] {
	if v == nil {
		return nullable.NewNullNullable[T]()
	}
	return nullable.NewNullableWithValue(*v)
}

// orEmpty is s, an empty list for nil: a list is never null.
func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
````

`server/internal/modules/issue/adapter/http/issues_test.go`（新文件，125 行）：

````file server/internal/modules/issue/adapter/http/issues_test.go
package httpadapter_test

import (
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The work items the use case answers: WEB-7, completed, with an assignee
// and a label, made by alice; WEB-8, new, with none of them, made by no one
// now (its creator's account is gone).
var (
	stateID   = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000c1")
	bobID     = uuid.MustParse("0199a2b4-0000-7000-8000-000000000002")
	bugID     = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000d1")
	completed = created.Add(-1)
	webSeven  = domain.Issue{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000e7"), WorkspaceID: acmeID, ProjectID: webID, SequenceID: 7,
		Name: "Fix the login", StateID: stateID, Priority: domain.PriorityHigh, SortOrder: 75535.5, CompletedAt: &completed,
		AssigneeIDs: []uuid.UUID{bobID}, LabelIDs: []uuid.UUID{bugID}, CreatedByID: &aliceID, UpdatedByID: &aliceID, CreatedAt: created,
		UpdatedAt: created}
	webSevenJSON = `{"archived_at":null,"assignee_ids":["0199a2b4-0000-7000-8000-000000000002"],` +
		`"completed_at":"2026-10-17T10:00:00.123455999Z","created_at":"2026-10-17T10:00:00.123456Z",` +
		`"created_by_id":"0199a2b4-0000-7000-8000-000000000001","id":"0199a2b4-0000-7000-8000-0000000000e7",` +
		`"label_ids":["0199a2b4-0000-7000-8000-0000000000d1"],"name":"Fix the login","parent_id":null,"priority":"high",` +
		`"project_id":"0199a2b4-0000-7000-8000-0000000000a1","sequence_id":7,"sort_order":75535.5,` +
		`"state_id":"0199a2b4-0000-7000-8000-0000000000c1","updated_at":"2026-10-17T10:00:00.123456Z",` +
		`"updated_by_id":"0199a2b4-0000-7000-8000-000000000001","workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
	webEight = domain.Issue{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000e8"), WorkspaceID: acmeID, ProjectID: webID, SequenceID: 8,
		Name: "Fix", StateID: stateID, Priority: domain.PriorityNone, SortOrder: 65535, CreatedAt: created, UpdatedAt: created}
	webEightJSON = `{"archived_at":null,"assignee_ids":[],"completed_at":null,"created_at":"2026-10-17T10:00:00.123456Z",` +
		`"created_by_id":null,"id":"0199a2b4-0000-7000-8000-0000000000e8","label_ids":[],"name":"Fix","parent_id":null,"priority":"none",` +
		`"project_id":"0199a2b4-0000-7000-8000-0000000000a1","sequence_id":8,"sort_order":65535,` +
		`"state_id":"0199a2b4-0000-7000-8000-0000000000c1","updated_at":"2026-10-17T10:00:00.123456Z","updated_by_id":null,` +
		`"workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
)

// POST goes to the use case for the caller and the path's project, with
// the body's fields: the state nil, the priority empty and the lists empty
// when not given; the answer is 201 with the work item the use case
// answers, every field of it, null for each it has none of, [] for a list
// without ids.
func TestCreateIssue(t *testing.T) {
	path := "/api/v0/projects/" + webID.String() + "/issues"
	high := domain.PriorityHigh
	for _, tt := range []struct {
		body   string
		want   domain.NewIssue
		answer domain.Issue
		json   string
	}{
		{`{"name":" Fix the login ","state_id":"0199a2b4-0000-7000-8000-0000000000c1","priority":"high",` +
			`"assignee_ids":["0199a2b4-0000-7000-8000-000000000002"],"label_ids":["0199a2b4-0000-7000-8000-0000000000d1"]}`,
			domain.NewIssue{Name: " Fix the login ", StateID: &stateID, Priority: high, AssigneeIDs: []uuid.UUID{bobID},
				LabelIDs: []uuid.UUID{bugID}}, webSeven, webSevenJSON},
		{`{"name":"Fix"}`, domain.NewIssue{Name: "Fix"}, webEight, webEightJSON},
		{`{"name":"Fix","assignee_ids":[],"label_ids":[]}`, domain.NewIssue{Name: "Fix", AssigneeIDs: []uuid.UUID{}, LabelIDs: []uuid.UUID{}},
			webEight, webEightJSON},
	} {
		create := &fakeCreate{answer: tt.answer}
		h := newServer(t, create)
		if res, body := do(t, h, request(http.MethodPost, path, "alice", tt.body)); res.StatusCode != http.StatusCreated || body != tt.json+"\n" {
			t.Errorf("POST %s = %d %s, want 201 %s", tt.body, res.StatusCode, body, tt.json)
		}
		if want := []string{aliceID.String() + " " + webID.String()}; !slices.Equal(create.calls, want) ||
			!reflect.DeepEqual(create.got, []domain.NewIssue{tt.want}) {
			t.Errorf("POST %s: calls %q with %+v; want %q with %+v", tt.body, create.calls, create.got, want, tt.want)
		}
	}
}

// A body without its name, with a field it may not have (a number or a
// sort order, which the project gives: M4 design 3.7; a description, which
// comes in P4), a field of another type, or an id that is no id: refused as
// bad_request before the use case.
func TestCreateIssueHoldsTheBodyToItsStructure(t *testing.T) {
	create := &fakeCreate{}
	h := newServer(t, create)
	for _, body := range []string{`{}`, `{"priority":"high"}`, `{"name":1}`, `{"name":"A","sequence_id":3}`, `{"name":"A","sort_order":1}`,
		`{"name":"A","description_html":"<p>x</p>"}`, `{"name":"A","state_id":"Backlog"}`, `{"name":"A","state_id":null}`,
		`{"name":"A","assignee_ids":"bob"}`, `{"name":"A","label_ids":["Bug"]}`} {
		res, got := do(t, h, request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/issues", "alice", body))
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
func TestCreateIssueRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"an assignee refused", shared.Invalid(shared.FieldError{Field: "assignee_ids[1]", Code: shared.FieldNotAllowed,
			Message: "must be an active member of the project who is not its guest"}), http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"assignee_ids[1]","code":"not_allowed","message":"must be an active member of the project who is not its guest"}]}`},
		{"no project", domain.ErrProjectNotFound, http.StatusNotFound,
			`{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`},
		{"a project guest", shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"an archived project", domain.ErrProjectArchived, http.StatusConflict,
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
		{"a failure", errGone, http.StatusInternalServerError,
			`{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
	} {
		h := newServer(t, &fakeCreate{err: tt.err})
		res, body := do(t, h, request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/issues", "alice", `{"name":"Fix"}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: POST = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
````

`server/internal/modules/issue/module.go`（修改，4 处）：

````old server/internal/modules/issue/module.go
// work items' tables, and soft-deletes a deleted project's rows of them for
// the project module's deletions (its IssueCascade port).
````
````new server/internal/modules/issue/module.go
// work items' tables, brings creating a work item, and soft-deletes a
// deleted project's rows of them for the project module's deletions (its
// IssueCascade port).
````
````old server/internal/modules/issue/module.go
	"github.com/jackc/pgx/v5/pgxpool"

````
````new server/internal/modules/issue/module.go
	"github.com/jackc/pgx/v5/pgxpool"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
````
````old server/internal/modules/issue/module.go
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
````
````new server/internal/modules/issue/module.go
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
````
````old server/internal/modules/issue/module.go
}

// Actions lists the module's actions: bootstrap's test holds the union of
````
````new server/internal/modules/issue/module.go
}

// ProjectPorts are what the module's writes ask of a project (M4 design
// 6.5): its gate, its numbers, its states', labels' and members' catalogs.
// project.Provide's IssueProjects is them, converted (bootstrap/ports.go).
type ProjectPorts interface {
	app.ProjectGate
	app.Sequences
	app.StateCatalog
	app.LabelCatalog
	app.ProjectMembers
}

// LockedProject, StateRef and its StateGroup are the values ProjectPorts
// hand over: bootstrap converts project's into them (M4 design 6.5).
type (
	LockedProject = app.LockedProject
	StateRef      = app.StateRef
	StateGroup    = domain.StateGroup
)

// Deps are what bootstrap gives the module (M4 design 6.6, step 6).
type Deps struct {
	Pool       *pgxpool.Pool
	Tx         shared.TxManager
	Clock      app.Clock
	Authorizer shared.Authorizer
	// Projects are project's IssueProjects, converted (bootstrap/ports.go).
	Projects ProjectPorts
	// Profiles is identity's PublicProfiles, converted (bootstrap/ports.go).
	Profiles app.PublicProfiles
}

// Module is the wired issue module.
type Module struct {
	uc httpadapter.UseCases
}

// New wires the module's use cases and its HTTP side from d; nothing is
// registered or injected after it (M4 design 6.6).
func New(d Deps) *Module {
	store := postgresadapter.New(d.Pool)
	return &Module{uc: httpadapter.UseCases{
		CreateIssue: app.NewCreateIssue(app.CreateIssueDeps{
			Locks: app.NewLocks(d.Projects, d.Authorizer), Issues: store, Sequences: d.Projects, States: d.Projects, Labels: d.Projects,
			Members: d.Projects, Profiles: d.Profiles, Tx: d.Tx, Clock: d.Clock,
		}),
	}}
}

// Register mounts the module's API on router behind api's middlewares.
func (m *Module) Register(router *httpserver.Router, api *httpserver.API) {
	httpadapter.Register(router, api, m.uc)
}

// Actions lists the module's actions: bootstrap's test holds the union of
````

- [ ] **Step 3: 接线和端口的转换**

`server/internal/bootstrap/app.go`（修改，2 处）：

````old server/internal/bootstrap/app.go
		ProjectAccess:  accessProjects{projects: projectPorts.ProjectAccess},
	})
````
````new server/internal/bootstrap/app.go
		ProjectAccess:  accessProjects{projects: projectPorts.ProjectAccess},
	})
	issues := issue.New(issue.Deps{
		Pool: pool, Tx: tx, Clock: clock.System{}, Authorizer: authorizer,
		Projects: issueProjects{projects: projectPorts.IssueProjects},
		Profiles: issueProfiles{profiles: identityPorts.PublicProfiles},
	})
````
````old server/internal/bootstrap/app.go
	proj.Register(a.router, api)
````
````new server/internal/bootstrap/app.go
	proj.Register(a.router, api)
	issues.Register(a.router, api)
````

`server/internal/bootstrap/ports.go`（修改，3 处）：

````old server/internal/bootstrap/ports.go
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
````
````new server/internal/bootstrap/ports.go
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
````
````old server/internal/bootstrap/ports.go
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
````
````new server/internal/bootstrap/ports.go
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/shared"
````
````old server/internal/bootstrap/ports.go
	return access.ProjectFacts(f), found, err
}

````
````new server/internal/bootstrap/ports.go
	return access.ProjectFacts(f), found, err
}

// issueProjects is project's IssueProjects as issue's ProjectPorts: the
// same reads and lock, each value converted.
type issueProjects struct {
	projects project.IssueProjects
}

func (p issueProjects) Find(ctx context.Context, id uuid.UUID) (uuid.UUID, bool, error) {
	return p.projects.ProjectWorkspace(ctx, id)
}

func (p issueProjects) Lock(ctx context.Context, id uuid.UUID) (issue.LockedProject, bool, error) {
	locked, found, err := p.projects.LockIssueProject(ctx, id)
	return issue.LockedProject(locked), found, err
}

func (p issueProjects) Next(ctx context.Context, id uuid.UUID) (int, error) {
	return p.projects.NextIssueSequence(ctx, id)
}

func (p issueProjects) States(ctx context.Context, id uuid.UUID) ([]issue.StateRef, error) {
	states, err := p.projects.IssueStates(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]issue.StateRef, len(states))
	for i, s := range states {
		out[i] = issue.StateRef{ID: s.ID, Group: issue.StateGroup(s.Group), Default: s.Default}
	}
	return out, nil
}

func (p issueProjects) Labels(ctx context.Context, id uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	return p.projects.IssueLabels(ctx, id, ids)
}

func (p issueProjects) Roles(ctx context.Context, id uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]shared.Role, error) {
	return p.projects.ActiveRoles(ctx, id, ids)
}

// issueProfiles is identity's PublicProfiles as issue's: each account's
// display name, by account.
type issueProfiles struct {
	profiles identity.PublicProfiles
}

func (p issueProfiles) DisplayNames(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	profiles, err := p.profiles.PublicProfiles(ctx, ids)
	if err != nil {
		return nil, err
	}
	names := make(map[uuid.UUID]string, len(profiles))
	for _, profile := range profiles {
		names[profile.ID] = profile.DisplayName
	}
	return names, nil
}

````

`server/internal/bootstrap/ports_test.go`（修改，4 处）：

````old server/internal/bootstrap/ports_test.go
	"errors"
````
````new server/internal/bootstrap/ports_test.go
	"errors"
	"fmt"
	"maps"
````
````old server/internal/bootstrap/ports_test.go
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
````
````new server/internal/bootstrap/ports_test.go
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
````
````old server/internal/bootstrap/ports_test.go
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
````
````new server/internal/bootstrap/ports_test.go
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/shared"
````
````old server/internal/bootstrap/ports_test.go
		t.Errorf("ProjectFacts() = %v, want %v", err, failure)
	}
}

````
````new server/internal/bootstrap/ports_test.go
		t.Errorf("ProjectFacts() = %v, want %v", err, failure)
	}
}

// fakeIssueProjects answers project's IssueProjects of one project, web,
// and records each question; err, when set, is every answer's failure.
type fakeIssueProjects struct {
	web    uuid.UUID
	locked project.IssueProject
	states []project.IssueState
	labels []uuid.UUID
	roles  map[uuid.UUID]shared.Role
	err    error
	asked  []string
}

func (f *fakeIssueProjects) ProjectWorkspace(_ context.Context, id uuid.UUID) (uuid.UUID, bool, error) {
	f.asked = append(f.asked, "workspace")
	return f.locked.WorkspaceID, id == f.web, f.err
}

func (f *fakeIssueProjects) LockIssueProject(_ context.Context, id uuid.UUID) (project.IssueProject, bool, error) {
	f.asked = append(f.asked, "lock")
	return f.locked, id == f.web, f.err
}

func (f *fakeIssueProjects) NextIssueSequence(context.Context, uuid.UUID) (int, error) {
	f.asked = append(f.asked, "next")
	return 7, f.err
}

func (f *fakeIssueProjects) IssueStates(context.Context, uuid.UUID) ([]project.IssueState, error) {
	f.asked = append(f.asked, "states")
	return f.states, f.err
}

func (f *fakeIssueProjects) IssueLabels(_ context.Context, _ uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	f.asked = append(f.asked, fmt.Sprint("labels ", ids))
	return f.labels, f.err
}

func (f *fakeIssueProjects) ActiveRoles(_ context.Context, _ uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]shared.Role, error) {
	f.asked = append(f.asked, fmt.Sprint("roles ", ids))
	return f.roles, f.err
}

// issueProjects hands issue project's answers to the same questions: the
// locked project and each state converted, every field; the workspace, the
// number, the labels and the roles as they came; found and the error as
// they came.
func TestIssueProjectsConvertsProjectsAnswers(t *testing.T) {
	web, acme, ann, bug := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	backlog := project.IssueState{ID: uuid.NewV7(), Group: "backlog", Default: true}
	done := project.IssueState{ID: uuid.NewV7(), Group: "completed"}
	fake := &fakeIssueProjects{web: web, locked: project.IssueProject{WorkspaceID: acme, Archived: true, DefaultAssigneeID: &ann},
		states: []project.IssueState{backlog, done}, labels: []uuid.UUID{bug}, roles: map[uuid.UUID]shared.Role{ann: shared.RoleMember}}
	p := issueProjects{projects: fake}
	ctx := context.Background()

	if w, found, err := p.Find(ctx, web); err != nil || !found || w != acme {
		t.Errorf("Find() = %s, %v, %v; want acme, found", w, found, err)
	}
	if l, found, err := p.Lock(ctx, web); err != nil || !found || l != (issue.LockedProject{WorkspaceID: acme, Archived: true, DefaultAssigneeID: &ann}) {
		t.Errorf("Lock() = %+v, %v, %v; want web as locked, found", l, found, err)
	}
	if _, found, _ := p.Lock(ctx, uuid.NewV7()); found {
		t.Error("Lock() of another project found it")
	}
	if n, err := p.Next(ctx, web); err != nil || n != 7 {
		t.Errorf("Next() = %d, %v; want 7", n, err)
	}
	want := []issue.StateRef{{ID: backlog.ID, Group: "backlog", Default: true}, {ID: done.ID, Group: "completed"}}
	if got, err := p.States(ctx, web); err != nil || !slices.Equal(got, want) {
		t.Errorf("States() = %+v, %v; want %+v", got, err, want)
	}
	if got, err := p.Labels(ctx, web, []uuid.UUID{bug}); err != nil || !slices.Equal(got, []uuid.UUID{bug}) {
		t.Errorf("Labels() = %v, %v; want Bug", got, err)
	}
	if got, err := p.Roles(ctx, web, []uuid.UUID{ann}); err != nil || !maps.Equal(got, fake.roles) {
		t.Errorf("Roles() = %v, %v; want ann's", got, err)
	}
	if want := []string{"workspace", "lock", "lock", "next", "states", fmt.Sprint("labels ", []uuid.UUID{bug}),
		fmt.Sprint("roles ", []uuid.UUID{ann})}; !slices.Equal(fake.asked, want) {
		t.Errorf("project was asked %q, want %q", fake.asked, want)
	}

	failure := errors.New("connection reset")
	fake.err = failure
	if got, err := p.States(ctx, web); !errors.Is(err, failure) || got != nil {
		t.Errorf("States() = %+v, %v; want no states and %v", got, err, failure)
	}
	for name, call := range map[string]func() error{
		"Find":   func() error { _, _, err := p.Find(ctx, web); return err },
		"Lock":   func() error { _, _, err := p.Lock(ctx, web); return err },
		"Next":   func() error { _, err := p.Next(ctx, web); return err },
		"Labels": func() error { _, err := p.Labels(ctx, web, nil); return err },
		"Roles":  func() error { _, err := p.Roles(ctx, web, nil); return err },
	} {
		if err := call(); !errors.Is(err, failure) {
			t.Errorf("%s() = %v, want %v", name, err, failure)
		}
	}
}

// issueProfiles asks identity for the ids issue asks for and hands over
// each account's display name, by account; an error as it came, without
// names.
func TestIssueProfilesConvertsIdentitysAnswer(t *testing.T) {
	alice := identityapp.PublicProfile{ID: uuid.NewV7(), Email: "alice@corp.com", FirstName: "Alice", DisplayName: "al"}
	bob := identityapp.PublicProfile{ID: uuid.NewV7(), Email: "bob@corp.com", DisplayName: "bob"}
	fake := &fakeIdentityProfiles{profiles: []identityapp.PublicProfile{alice, bob}}
	p := issueProfiles{profiles: fake}
	ids := []uuid.UUID{bob.ID, uuid.NewV7(), alice.ID}

	got, err := p.DisplayNames(context.Background(), ids)
	if want := map[uuid.UUID]string{alice.ID: "al", bob.ID: "bob"}; err != nil || !maps.Equal(got, want) {
		t.Errorf("DisplayNames() = %v, %v; want %v", got, err, want)
	}
	if len(fake.asked) != 1 || !slices.Equal(fake.asked[0], ids) {
		t.Errorf("identity was asked %v, want %v", fake.asked, ids)
	}
	failure := errors.New("connection reset")
	fake.err = failure
	if got, err := p.DisplayNames(context.Background(), ids); !errors.Is(err, failure) || got != nil {
		t.Errorf("DisplayNames() = %v, %v; want no names and %v", got, err, failure)
	}
}

````

- [ ] **Step 4: 矩阵和最先锁工作区的测试的收窄**

`server/internal/bootstrap/permission_matrix_issues_test.go`（新文件，119 行）：

````file server/internal/bootstrap/permission_matrix_issues_test.go
package bootstrap

import (
	"net/http"
	"slices"
	"testing"
	"uuid"
)

// The rows of the permission matrix of a project's work items (M4 design
// 9.2): creating one, under the project of each column. Each cell writes
// on a copy of its own, so each creation is its project's first.

// newIssue is the body of the work item the rows create.
const newIssue = `{"name":"Fix the login"}`

// withIssueBody is the request of a row whose callers each create a work
// item with the body body gives for their column.
func withIssueBody(body func(c caller, s seeded) string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return toProject(http.MethodPost, "/issues", body(c, s))(c, s)
	}
}

func issueMatrixRows() []matrixRow {
	return []matrixRow{
		// The project's admins and members, not its guests (M4 design 3.4):
		// PM+WA as the member he is; a workspace admin or member who is no
		// member of the project sees it and may not.
		{op: "createIssue", write: true, columns: projectColumns, request: toProject(http.MethodPost, "/issues", newIssue),
			cells: ofProject(cellCreated, cellCreated, cellForbidden, cellCreated, cellForbidden, cellForbidden), check: createsTheIssue},
		// Each field given: in Done, high, assigned to PM's account, a
		// member of the project, and labelled UI; the activity names him by
		// his display name, which identity gives (bootstrap/ports.go).
		{op: "createIssue", variant: "each field given", write: true, columns: projectColumns,
			request: withIssueBody(func(c caller, s seeded) string {
				return `{"name":"Fix","state_id":"` + s.state(projectOf(c), "Done").String() + `","priority":"high","assignee_ids":["` +
					s.account(callerProjectMember).String() + `"],"label_ids":["` + s.label(projectOf(c), "UI").String() + `"]}`
			}),
			cells: ofProject(cellCreated, cellCreated, cellForbidden, cellCreated, cellForbidden, cellForbidden), check: createsTheGivenIssue},
		// An archived project takes no work items (M4 design 3.8), after the
		// decision: who may not create one learns nothing of its state.
		{op: "createIssue", variant: "archived", write: true, columns: archivedColumns, request: toProject(http.MethodPost, "/issues", newIssue),
			cells: ofArchived(cellProjectArchived, cellForbidden)},
		// The archived project's Backlog, a state of another project of the
		// same workspace: the state is checked after the decision.
		{op: "createIssue", variant: "another project's state", write: true, columns: projectColumns,
			request: withIssueBody(func(_ caller, s seeded) string {
				return `{"name":"Fix","state_id":"` + s.state("acme/archived", "Backlog").String() + `"}`
			}),
			cells:   ofProject(cellValidationFailed, cellValidationFailed, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden),
			refusal: "state_id not_allowed"},
		// The archived project's Bug, a label of another project of the same
		// workspace, checked after the decision too.
		{op: "createIssue", variant: "another project's label", write: true, columns: projectColumns,
			request: withIssueBody(func(_ caller, s seeded) string {
				return `{"name":"Fix","label_ids":["` + s.label("acme/archived", "Bug").String() + `"]}`
			}),
			cells:   ofProject(cellValidationFailed, cellValidationFailed, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden),
			refusal: "label_ids[0] not_allowed"},
		// The project's guest, PG's account, as the assignee: a guest is not
		// assigned (M4 design 3.7), and it is checked after the decision.
		{op: "createIssue", variant: "a guest assigned", write: true, columns: projectColumns,
			request: withIssueBody(func(_ caller, s seeded) string {
				return `{"name":"Fix","assignee_ids":["` + s.account(accountOf(callerProjectGuest)).String() + `"]}`
			}),
			cells:   ofProject(cellValidationFailed, cellValidationFailed, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden),
			refusal: "assignee_ids[0] not_allowed"},
	}
}

// createsTheIssue: the column's project's first work item, Fix the login,
// in its default state Backlog, of no priority, at a state's first sort
// order, not completed, assigned to no one, without labels, made and
// last changed by the caller.
func createsTheIssue(t *testing.T, c caller, s seeded, answer string) {
	var i struct {
		ProjectID   uuid.UUID   `json:"project_id"`
		SequenceID  int         `json:"sequence_id"`
		Name        string      `json:"name"`
		StateID     uuid.UUID   `json:"state_id"`
		Priority    string      `json:"priority"`
		SortOrder   float64     `json:"sort_order"`
		CompletedAt *string     `json:"completed_at"`
		AssigneeIDs []uuid.UUID `json:"assignee_ids"`
		LabelIDs    []uuid.UUID `json:"label_ids"`
		CreatedByID *uuid.UUID  `json:"created_by_id"`
		UpdatedByID *uuid.UUID  `json:"updated_by_id"`
	}
	decodeAnswer(t, answer, &i)
	me := s.account(accountOf(c))
	if i.ProjectID != s.project(projectOf(c)) || i.SequenceID != 1 || i.Name != "Fix the login" || i.StateID != s.state(projectOf(c), "Backlog") ||
		i.Priority != "none" || i.SortOrder != 65535 || i.CompletedAt != nil || len(i.AssigneeIDs) != 0 || len(i.LabelIDs) != 0 ||
		i.CreatedByID == nil || *i.CreatedByID != me || i.UpdatedByID == nil || *i.UpdatedByID != me {
		t.Errorf("%s creates %s; want %s's work item 1, Fix the login, in Backlog, none, at 65535, by %s", c, answer, projectOf(c), me)
	}
}

// createsTheGivenIssue: the column's project's first work item, Fix, in
// Done, high, at a state's first sort order, completed at its creation,
// assigned to PM's account, labelled UI.
func createsTheGivenIssue(t *testing.T, c caller, s seeded, answer string) {
	var i struct {
		SequenceID  int         `json:"sequence_id"`
		StateID     uuid.UUID   `json:"state_id"`
		Priority    string      `json:"priority"`
		SortOrder   float64     `json:"sort_order"`
		CompletedAt *string     `json:"completed_at"`
		AssigneeIDs []uuid.UUID `json:"assignee_ids"`
		LabelIDs    []uuid.UUID `json:"label_ids"`
	}
	decodeAnswer(t, answer, &i)
	pm, ui := s.account(callerProjectMember), s.label(projectOf(c), "UI")
	if i.SequenceID != 1 || i.StateID != s.state(projectOf(c), "Done") || i.Priority != "high" || i.SortOrder != 65535 || i.CompletedAt == nil ||
		!slices.Equal(i.AssigneeIDs, []uuid.UUID{pm}) || !slices.Equal(i.LabelIDs, []uuid.UUID{ui}) {
		t.Errorf("%s creates %s; want %s's work item 1 in Done, high, at 65535, completed, assigned to %s, labelled %s", c, answer,
			projectOf(c), pm, ui)
	}
}
````

`server/internal/bootstrap/permission_matrix_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_test.go
		labelMatrixRows())
````
````new server/internal/bootstrap/permission_matrix_test.go
		labelMatrixRows(), issueMatrixRows())
````

`server/internal/bootstrap/project_write_locks_test.go`（修改，6 处）：

````old server/internal/bootstrap/project_write_locks_test.go
// write that the only admin's table asks (leaveWorkspace).
````
````new server/internal/bootstrap/project_write_locks_test.go
// write that the only admin's table asks (leaveWorkspace); not a write on
// the work-item side (tagged issue), which locks its project's row first
// and its workspace's not at all (M4 design 3.5, decision point 1: B).
````
````old server/internal/bootstrap/project_write_locks_test.go
		if op.Method != http.MethodGet && (slices.Contains(strings.Split(op.Path, "/"), "{project_id}") || projectLevel[op.ID]) {
````
````new server/internal/bootstrap/project_write_locks_test.go
		if op.Method != http.MethodGet && !slices.Contains(op.Tags, "issue") &&
			(slices.Contains(strings.Split(op.Path, "/"), "{project_id}") || projectLevel[op.ID]) {
````
````old server/internal/bootstrap/project_write_locks_test.go
// asks it.
````
````new server/internal/bootstrap/project_write_locks_test.go
// asks it, nor a write on the work-item side, also when its path names
// the project and its rows a project table's columns.
````
````old server/internal/bootstrap/project_write_locks_test.go
		{ID: "getProject", Method: http.MethodGet, Path: "/api/v0/projects/{project_id}"},
		{ID: "createProject", Method: http.MethodPost, Path: "/api/v0/workspaces/{slug}/projects"},
		{ID: "leaveProject", Method: http.MethodPost, Path: "/api/v0/projects/{project_id}/leave"},
		{ID: "updateProjectMember", Method: http.MethodPatch, Path: "/api/v0/project-members/{project_member_id}"},
		{ID: "leaveProjectSelf", Method: http.MethodPost, Path: "/api/v0/project-members/{project_member_id}/leave"},
		{ID: "leaveWorkspace", Method: http.MethodPost, Path: "/api/v0/workspaces/{slug}/leave"},
````
````new server/internal/bootstrap/project_write_locks_test.go
		{ID: "createIssue", Tags: []string{"issue"}, Method: http.MethodPost, Path: "/api/v0/projects/{project_id}/issues"},
		{ID: "getProject", Tags: []string{"project"}, Method: http.MethodGet, Path: "/api/v0/projects/{project_id}"},
		{ID: "createProject", Tags: []string{"project"}, Method: http.MethodPost, Path: "/api/v0/workspaces/{slug}/projects"},
		{ID: "leaveProject", Tags: []string{"project"}, Method: http.MethodPost, Path: "/api/v0/projects/{project_id}/leave"},
		{ID: "updateProjectMember", Tags: []string{"project"}, Method: http.MethodPatch, Path: "/api/v0/project-members/{project_member_id}"},
		{ID: "leaveProjectSelf", Tags: []string{"project"}, Method: http.MethodPost, Path: "/api/v0/project-members/{project_member_id}/leave"},
		{ID: "leaveWorkspace", Tags: []string{"workspace"}, Method: http.MethodPost, Path: "/api/v0/workspaces/{slug}/leave"},
````
````old server/internal/bootstrap/project_write_locks_test.go
	rows := []matrixRow{
````
````new server/internal/bootstrap/project_write_locks_test.go
	rows := []matrixRow{
		{op: "createIssue", write: true, columns: projectColumns},
````
````old server/internal/bootstrap/project_write_locks_test.go
// lock table), as bootstrap wires it: alice, acme's admin, writes on her
````
````new server/internal/bootstrap/project_write_locks_test.go
// lock table), as bootstrap wires it; the work-item side's writes are not
// among them (writesOnAProject): alice, acme's admin, writes on her
````

- [ ] **Step 5: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix|TestIssueProjectsConvertsProjectsAnswers$|TestIssueProfilesConvertsIdentitysAnswer$|TestEachWriteOnAProjectSharesItsWorkspaceFirst$' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过；关键词守卫 `keywords: 72 rules, 23 exceptions, no hits.`

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

- [ ] **Step 6: 提交**

```bash
git add api/modules/issue.yaml api/modules/project.yaml api/openapi.yaml server/internal/bootstrap/app.go server/internal/bootstrap/permission_matrix_issues_test.go server/internal/bootstrap/permission_matrix_test.go server/internal/bootstrap/ports.go server/internal/bootstrap/ports_test.go server/internal/bootstrap/project_write_locks_test.go server/internal/modules/issue/adapter/http/gen/oapi-codegen.yaml server/internal/modules/issue/adapter/http/handler.go server/internal/modules/issue/adapter/http/handler_test.go server/internal/modules/issue/adapter/http/issues.go server/internal/modules/issue/adapter/http/issues_test.go server/internal/modules/issue/module.go api/dist/openapi.yaml server/internal/modules/issue/adapter/http/gen/bodyshape.gen.go server/internal/modules/issue/adapter/http/gen/server.gen.go server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M4/P1): createIssue in the API, wired, in the permission matrix

POST /api/v0/projects/{project_id}/issues creates a work item and
answers it whole, every list an array. bootstrap builds the issue
module over project's lock, numbers and catalogs and identity's display
names. The matrix holds a creation of each column, with every field
given, in an archived project, with another project's state and with a
guest assigned. The work-item side's writes lock their project's row
first and the workspace's not at all, so the test that every write on a
project shares its workspace first leaves them out by their tag.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p1a.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试；结构是迁移的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是关键词守卫；静态是 `make lint-web` 的文案和键的核对）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T9.1` | `createIssue` 的处理函数丢掉请求的优先级 | `internal/modules/issue/adapter/http` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `createIssue` 各格（`make test`） | 单元；组合 |
| `T9.2` | 没有负责人或标签的工作项把它们答成 null，不是 [] | `internal/modules/issue/adapter/http` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `createIssue` 各格（`make test`） | 单元；组合 |
| `T9.3` | 回答把归档的时刻当作完成的时刻 | `internal/modules/issue/adapter/http` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `createIssue` 各格（`make test`） | 单元；组合 |
| `T9.4` | `createIssue` 的契约不声明它回答的 `project.archived` | `internal/modules/issue/adapter/http` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `createIssue` 各格（`make test`） | 单元；组合 |
| `T9.5` | `writesOnAProject` 留着工作项一侧的写，它们不锁工作区 | `internal/bootstrap` 的 `TestWritesOnAProjectAreEachShape`（`make test`）、`internal/bootstrap` 的 `TestEachWriteOnAProjectSharesItsWorkspaceFirst`（`make test`） | 单元；组合 |
| `T9.6` | `writesOnAProject` 去掉每个模块带标签的写，不只是工作项一侧的 | `internal/bootstrap` 的 `TestWritesOnAProjectAreEachShape`（`make test`） | 单元 |
| `T9.7` | `bootstrap` 不给 `issue` 显示名：带负责人的工作项失败 | `internal/bootstrap` 的 `TestPermissionMatrix` 的 `createIssue` 各格（`make test`） | 组合 |
| `T9.8` | `issueProjects` 丢掉哪个状态是默认的 | `internal/bootstrap` 的 `TestIssueProjectsConvertsProjectsAnswers`、`TestIssueProfilesConvertsIdentitysAnswer`（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `createIssue` 各格（`make test`） | 单元；组合 |
| `T9.9` | `issueProfiles` 用名而不是显示名称呼账户 | `internal/bootstrap` 的 `TestIssueProjectsConvertsProjectsAnswers`、`TestIssueProfilesConvertsIdentitysAnswer`（`make test`）、`internal/bootstrap` 的 `TestCreatingAWorkItemStampsItsRows`（`make test`）（Task 10 起） | 单元；组合 |
| `T9.10` | `bootstrap` 不挂 `issue` 的接口 | `internal/bootstrap` 的 `TestPermissionMatrix` 的 `createIssue` 各格（`make test`） | 组合 |
| `T9.11` | `IssueCreate` 收下它没有的字段：请求里的编号、排序值被忽略，不被拒绝 | `internal/modules/issue/adapter/http` 的 全部测试（`make test`） | 单元 |

---

### Task 10: 取锁的形状（交错 21）、盖戳和事务的连接

**Files:**
- Create: `server/internal/bootstrap/issue_world_test.go`、`server/internal/bootstrap/issue_write_locks_test.go`、`server/internal/bootstrap/issue_writes_test.go`

**Interfaces:** 没有新的生产代码（spec 2.10；M4 设计 3.5 交错 21，9.3）。
- `issue_world_test.go`：工作项的组合测试共用的世界，Task 11–13 照用。
  - `newIssueWorld`：bootstrap 组合出的 app；acme、Web、状态 Todo、标签 Bug；
  - 人：alice 是 acme 的管理员；bob 是 Web 的成员；walt 是工作区成员而不是 Web 的；wanda 是工作区管理员而不是 Web 的；olga 是外人；
  - `send`、`fullIssue`。
- `issue_write_locks_test.go`：
  - `issueWrite`、`issueWrites`：契约中工作项一侧的每个写各一行；
  - `issueTables`、`writesOnIssues`（按标签 `issue` 从契约读出的写）；
  - 帮手 `lockedTables`、`release`、`waitedTable`、`lockOn`。
- `issue_writes_test.go`：`newIssueRoute`（`issue` 模块经 bootstrap 的接法装在一个路由上）。

**Tests:**（组合）
- `TestEachIssueWriteLocksItsProjectFirst`：先核对 `issueWrites` 与契约中工作项一侧的写一一对应（完整性：P1b 的写不加行时在任何数据库之前失败）。然后对每个写：
  - 拒绝的不取锁：另一个事务以 `EXCLUSIVE` 持有它读写的每张表（行锁和写都等它，普通的读不等）；walt、wanda 答 403，olga 答 404；
  - 项目最先：同样的锁下，alice 的写只在 `projects` 上等；
  - 它的锁就是 `FOR NO KEY UPDATE`，工作区上没有锁：另一个事务以 `ACCESS EXCLUSIVE` 持写在锁下最先读的表（`reads`，`createIssue` 是 `states`；普通的读也等它）时，alice 的写在它自己的任何写之前停在那里（编号的 `UPDATE` 还没有做），持着 Web 的 `FOR NO KEY UPDATE`（不更弱、不更强），acme 的行上没有锁；
  - 工作区行被持 `FOR NO KEY UPDATE`（每个项目连带都这样持）时，写照样完成。
- `TestCreatingAWorkItemStampsItsRows`：
  - alice 在 Web 的 Todo 建一个工作项，指派 bob、标签 Bug；
  - 七行（工作项、负责人、标签、两个关注、两行动态）都由 alice 在请求之内的同一时刻写下；
  - 第二行动态用 `users.display_name` 写 bob 的名字（经 `issueProfiles`）。
- `TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`：一个连接的池（`poolOfOne`）上，经池而不经事务的语句会等第二个连接到超时。逐个发出：
  - 完整的创建；
  - bob 是默认负责人时不带负责人的创建；
  - 标签不存在的 422；
  - walt 的 403（锁之前）；
  - Web 归档之后的 409（锁之下）。

- [ ] **Step 1: 工作项的世界**

`server/internal/bootstrap/issue_world_test.go`（新文件，79 行）：

````file server/internal/bootstrap/issue_world_test.go
package bootstrap

import (
	"net/http"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// issueWorld is the wired app on a database of its own, as the tests of
// the work-item side's writes find it: acme, whose admin alice created
// its public project Web, with Web's state Todo and its label Bug; bob,
// a member of acme and of Web; and three accounts who may not write on
// Web's work items: walt, a member of acme who is none of Web, and wanda,
// an admin of acme who is none of Web either, see Web (403); olga, no
// member of acme, does not (404).
type issueWorld struct {
	contract             *apitest.Contract
	base                 string
	pool                 *pgxpool.Pool
	tokens               map[string]string    // each account's, by its name
	ids                  map[string]uuid.UUID // each account's, by its name
	acme, web, todo, bug uuid.UUID
}

// newIssueWorld makes the world: each account registered through the API,
// acme, Web, bob's membership of Web and Bug through the API too, and the
// memberships of acme through the workspace store (inWorkspaceOf).
func newIssueWorld(t *testing.T) issueWorld {
	t.Helper()
	contract := apitest.Load(t)
	dbURL := pgtest.NewDatabase(t)
	w := issueWorld{contract: contract, base: startApp(t, testConfig(t, dbURL, false), migrations.FS()), pool: openPool(t, dbURL),
		tokens: map[string]string{}, ids: map[string]uuid.UUID{}}
	for _, name := range []string{"alice", "bob", "walt", "wanda", "olga"} {
		w.tokens[name] = registerAccount(t, contract, w.base, name+"@example.com").AccessToken
		w.ids[name] = accountID(t, contract, w.base, w.tokens[name])
	}
	var created struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/workspaces", `{"name":"Acme","slug":"acme"}`, http.StatusCreated), &created)
	w.acme = created.ID
	w.web = createdProject(t, contract, w.base, w.tokens["alice"], "acme", "Web", "WEB")
	for name, role := range map[string]shared.Role{"bob": shared.RoleMember, "walt": shared.RoleMember, "wanda": shared.RoleAdmin} {
		inWorkspaceOf(t, w.pool, w.web, w.ids[name], w.ids["alice"], role)
	}
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/members",
		`{"members":[{"member_id":"`+w.ids["bob"].String()+`","role":15}]}`, http.StatusCreated)
	w.todo = stateID(t, w.pool, w.web, "Todo")
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/labels", `{"name":"Bug"}`, http.StatusCreated),
		&created)
	w.bug = created.ID
	return w
}

// send sends method path with body as the account name, checks the answer
// against the contract and that it is want, and returns its body.
func (w issueWorld) send(t *testing.T, name, method, path, body string, want int) string {
	t.Helper()
	status, answer := call(t, w.contract, method, w.base+path, w.tokens[name], body)
	if status != want {
		t.Fatalf("%s %s as %s = %d %s, want %d", method, path, name, status, answer, want)
	}
	return answer
}

// fullIssue is the body of a work item in Todo, assigned to bob and
// labelled Bug.
func (w issueWorld) fullIssue() string {
	return `{"name":"Fix","state_id":"` + w.todo.String() + `","assignee_ids":["` + w.ids["bob"].String() + `"],"label_ids":["` +
		w.bug.String() + `"]}`
}
````

- [ ] **Step 2: 取锁的形状**

`server/internal/bootstrap/issue_write_locks_test.go`（新文件，183 行）：

````file server/internal/bootstrap/issue_write_locks_test.go
package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// issueWrite is a write on the work-item side as
// TestEachIssueWriteLocksItsProjectFirst sends it: its operation, its
// method and its path as the contract has them, with %s for Web's id; its
// body, which names the world's rows; the answer it gets; and reads, the
// table it reads first under its project's lock, before any write of its
// own.
type issueWrite struct {
	op, method, path string
	body             func(w issueWorld) string
	want             int
	reads            string
}

// issueWrites are the work-item side's writes, each with its row: every
// operation of the contract tagged issue but GET (writesOnIssues).
var issueWrites = []issueWrite{
	{op: "createIssue", method: http.MethodPost, path: "/api/v0/projects/%s/issues", body: issueWorld.fullIssue, want: http.StatusCreated,
		reads: "states"},
}

// issueTables are the tables a write on the work-item side reads or
// writes: its project's, the memberships its decision reads, the states
// and labels it checks, and the work-item side's own.
const issueTables = "workspaces, workspace_members, projects, project_members, states, labels, issues, issue_assignees, issue_labels, " +
	"issue_subscribers, issue_activities"

// writesOnIssues lists the operations of ops tagged issue, but GET, sorted.
func writesOnIssues(ops []apitest.Operation) []string {
	var writes []string
	for _, op := range ops {
		if op.Method != http.MethodGet && slices.Contains(op.Tags, "issue") {
			writes = append(writes, op.ID)
		}
	}
	return slices.Sorted(slices.Values(writes))
}

// Each write on the work-item side, as bootstrap wires it, takes its
// project's row FOR NO KEY UPDATE as its first lock, and its workspace's
// row no more than a foreign key's FOR KEY SHARE; a caller it refuses
// takes no lock at all (M4 design 3.5, decision point 1: B; 9.3,
// interleaving 21). Every write of the contract has its row here
// (issueWrites), which a write without one fails before any database.
// Each write, on Web of the world (newIssueWorld):
//   - Refused, no lock: another transaction holds every table it reads or
//     writes (issueTables) IN EXCLUSIVE MODE, which a row lock and a write
//     wait for and a plain read does not. walt and wanda, who see Web but
//     may not write on its work items, are answered 403, and olga, who
//     does not see it, 404, while it holds them.
//   - Its project first: alice's write, under the same locks, waits for
//     the projects table, no other.
//   - FOR NO KEY UPDATE by its lock, and nothing of the workspace: another
//     transaction holds the table the write reads first under its lock
//     (reads) IN ACCESS EXCLUSIVE MODE, which a plain read waits for too.
//     alice's write waits for it before any write of its own, holding
//     Web's row FOR NO KEY UPDATE, no weaker and no stronger, and acme's
//     row not at all (lockOn).
//   - No lock of the workspace that a cascade's waits for: another
//     transaction holds acme's row FOR NO KEY UPDATE, as every cascade over
//     its projects does (M3 design 3.3); alice's write answers meanwhile.
//
// Each wait has a deadline; a write that waited where it may not fails
// there.
func TestEachIssueWriteLocksItsProjectFirst(t *testing.T) {
	contract := apitest.Load(t)
	ops := make([]string, len(issueWrites))
	for i, iw := range issueWrites {
		ops[i] = iw.op
		if !slices.ContainsFunc(contract.Operations(), func(o apitest.Operation) bool {
			return o.ID == iw.op && o.Method == iw.method && o.Path == fmt.Sprintf(iw.path, "{project_id}")
		}) {
			t.Fatalf("%s's row sends %s %s, not the contract's %s", iw.op, iw.method, iw.path, iw.op)
		}
	}
	if want := writesOnIssues(contract.Operations()); !slices.Equal(slices.Sorted(slices.Values(ops)), want) {
		t.Fatalf("the writes here are %q; the writes on the work-item side are %q: each has its row, in issueWrites", ops, want)
	}
	w := newIssueWorld(t)
	for _, iw := range issueWrites {
		t.Run(iw.op, func(t *testing.T) {
			path, body := w.base+fmt.Sprintf(iw.path, w.web), iw.body(w)
			send := func(name string) <-chan answer {
				req := newRequest(t, iw.method, path, w.tokens[name], []byte(body))
				w.contract.CheckRequest(t, req)
				return sendInBackground(req)
			}
			answers := func(name string, answered <-chan answer, want int) {
				t.Helper()
				a := receiveWithin(t, answered, 10*time.Second, "the answer to "+iw.op+" as "+name)
				if a.err != nil {
					t.Fatal(a.err)
				}
				if a.res.StatusCode != want {
					t.Errorf("%s as %s = %d %s, want %d", iw.op, name, a.res.StatusCode, a.body, want)
				}
			}

			tables := lockedTables(t, w.pool, issueTables, "EXCLUSIVE")
			for name, want := range map[string]int{"walt": http.StatusForbidden, "wanda": http.StatusForbidden, "olga": http.StatusNotFound} {
				answers(name, send(name), want)
			}
			answered := send("alice")
			pgtest.WaitForLockWaitBehind(t, w.pool, tables.Conn().PgConn(), 10*time.Second)
			if table := waitedTable(t, w.pool, tables.Conn().PgConn()); table != "projects" {
				t.Errorf("%s first waits for %s, not for projects", iw.op, table)
			}
			release(t, tables)
			answers("alice", answered, iw.want)

			read := lockedTables(t, w.pool, iw.reads, "ACCESS EXCLUSIVE")
			answered = send("alice")
			pgtest.WaitForLockWaitBehind(t, w.pool, read.Conn().PgConn(), 10*time.Second)
			if table := waitedTable(t, w.pool, read.Conn().PgConn()); table != iw.reads {
				t.Fatalf("%s waits for %s, not for %s", iw.op, table, iw.reads)
			}
			if lock := lockOn(t, w.pool, "projects WHERE id = $1", w.web); lock != "FOR NO KEY UPDATE" {
				t.Errorf("%s holds Web %s when it reads %s, not FOR NO KEY UPDATE", iw.op, lock, iw.reads)
			}
			if lock := lockOn(t, w.pool, "workspaces WHERE id = $1", w.acme); lock != "no lock" {
				t.Errorf("%s holds acme %s when it reads %s", iw.op, lock, iw.reads)
			}
			release(t, read)
			answers("alice", answered, iw.want)

			cascade := holding(t, w.pool, "SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", w.acme)
			answers("alice", send("alice"), iw.want)
			release(t, cascade)
		})
	}
}

// lockedTables begins a transaction on pool that holds tables, a list,
// in mode until it is released or the test ends.
func lockedTables(t *testing.T, pool *pgxpool.Pool, tables, mode string) pgx.Tx {
	t.Helper()
	tx, err := pool.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	if _, err := tx.Exec(pgtest.Soon(t), "LOCK TABLE "+tables+" IN "+mode+" MODE"); err != nil {
		t.Fatal(err)
	}
	return tx
}

// release rolls tx back.
func release(t *testing.T, tx pgx.Tx) {
	t.Helper()
	if err := tx.Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
}

// waitedTable is the table whose lock the backend blocked behind holder
// waits for.
func waitedTable(t *testing.T, pool *pgxpool.Pool, holder *pgconn.PgConn) string {
	t.Helper()
	var table string
	if err := pool.QueryRow(pgtest.Soon(t), `SELECT l.relation::regclass::text FROM pg_locks l
		WHERE NOT l.granted AND l.locktype = 'relation' AND $1::int = ANY (pg_blocking_pids(l.pid))`, int64(holder.PID())).Scan(&table); err != nil {
		t.Fatalf("the table waited for behind backend %d: %v", holder.PID(), err)
	}
	return table
}
````

- [ ] **Step 3: 盖戳和事务的连接**

`server/internal/bootstrap/issue_writes_test.go`（新文件，145 行）：

````file server/internal/bootstrap/issue_writes_test.go
package bootstrap

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// issueStamp is a row createIssue writes, as TestCreatingAWorkItemStampsItsRows
// reads it: its table, when it was made and last changed, and whether
// alice made it and changed it last, and acted, for an activity.
type issueStamp struct {
	table            string
	created, updated time.Time
	hers             bool
}

// createIssue as bootstrap wires it makes every row it writes at the time
// of its request, one moment, from the clock issue.New takes, and as its
// caller (M4 design 3.7): alice creates a work item in Web of the world,
// in Todo, assigned to bob and labelled Bug. The work item, its assignee,
// its label, its two subscribers, alice and bob, and its two activities
// are seven rows, each made and last changed by alice at one moment within
// the request. The second activity names bob by his display name as
// identity has it (bootstrap/ports.go's issueProfiles).
func TestCreatingAWorkItemStampsItsRows(t *testing.T) {
	w := newIssueWorld(t)
	before := time.Now().Truncate(time.Microsecond)
	answer := w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", w.fullIssue(), http.StatusCreated)
	after := time.Now()
	var created struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, answer, &created)

	rows, err := w.pool.Query(pgtest.Soon(t), `
		SELECT 'issues', created_at, updated_at, created_by_id = $2 AND updated_by_id = $2 FROM issues WHERE id = $1
		UNION ALL SELECT 'issue_assignees', created_at, updated_at, created_by_id = $2 AND updated_by_id = $2 FROM issue_assignees
			WHERE issue_id = $1
		UNION ALL SELECT 'issue_labels', created_at, updated_at, created_by_id = $2 AND updated_by_id = $2 FROM issue_labels
			WHERE issue_id = $1
		UNION ALL SELECT 'issue_subscribers', created_at, updated_at, created_by_id = $2 AND updated_by_id = $2 FROM issue_subscribers
			WHERE issue_id = $1
		UNION ALL SELECT 'issue_activities', created_at, updated_at, created_by_id = $2 AND updated_by_id = $2 AND actor_id = $2
			FROM issue_activities WHERE issue_id = $1`, created.ID, w.ids["alice"])
	if err != nil {
		t.Fatal(err)
	}
	stamps, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (issueStamp, error) {
		var s issueStamp
		return s, row.Scan(&s.table, &s.created, &s.updated, &s.hers)
	})
	if err != nil {
		t.Fatal(err)
	}
	var tables []string
	for _, s := range stamps {
		tables = append(tables, s.table)
		if !s.created.Equal(stamps[0].created) || !s.updated.Equal(s.created) || s.created.Before(before) || s.created.After(after) || !s.hers {
			t.Errorf("a row of %s made at %v, changed at %v, by alice %v; want made and changed by alice at one moment within %v to %v",
				s.table, s.created, s.updated, s.hers, before, after)
		}
	}
	if want := []string{"issue_activities", "issue_activities", "issue_assignees", "issue_labels", "issue_subscribers", "issue_subscribers",
		"issues"}; !slices.Equal(slices.Sorted(slices.Values(tables)), want) {
		t.Errorf("createIssue wrote rows of %q, want %q", tables, want)
	}

	var named, displayName string
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT a.new_value, u.display_name FROM issue_activities a, users u
		WHERE a.issue_id = $1 AND a.field = 'assignees' AND a.new_identifier = $2 AND u.id = $2`, created.ID, w.ids["bob"]).Scan(&named,
		&displayName); err != nil {
		t.Fatal(err)
	}
	if named != displayName {
		t.Errorf("the activity assigning bob names him %q; want his display name %q", named, displayName)
	}
}

// newIssueRoute is the issue module (issue.New) on pool, with the
// Authorizer on pool and project's and identity's ports converted, all as
// bootstrap wires them.
func newIssueRoute(t *testing.T, pool *pgxpool.Pool) moduleRoute {
	t.Helper()
	return newRoute(t, issue.New(issue.Deps{Pool: pool, Tx: postgres.NewTxManager(pool, 2*time.Second), Clock: clock.System{},
		Authorizer: authorizerOn(pool), Projects: issueProjects{projects: project.Provide(pool).IssueProjects},
		Profiles: issueProfiles{profiles: identity.Provide(pool).PublicProfiles}}).Register)
}

// Each write on the work-item side runs every statement of its transaction
// on that transaction's connection (M4 design 3.5, 6.7): its lock, the
// decision under it, its checks of the states, the members and the labels,
// its number, its writes, the display names, and its answer. The issue
// module and the Authorizer are wired as bootstrap wires them, on a pool
// of one connection (poolOfOne): a statement sent through the pool rather
// than the transaction would wait for a second connection that never
// comes, and its request fail at the request's deadline. On Web of the
// world, alice creates a work item in Todo, assigned to bob and labelled
// Bug; with bob made Web's default assignee, one without assignees, which
// he takes; one labelled with a label no project has, which the check
// under the lock refuses (422); walt, who sees Web, is refused before the
// lock (403); and with Web archived, alice's is refused under the lock
// (409).
func TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection(t *testing.T) {
	w := newIssueWorld(t)
	one := poolOfOne(t, w.pool.Config().ConnString())
	route := newIssueRoute(t, one)
	path := "/api/v0/projects/" + w.web.String() + "/issues"
	alice := w.ids["alice"]
	send := func(caller uuid.UUID, body string, want int) string {
		t.Helper()
		return route.answerWithin(t, w.contract, http.MethodPost, path, caller, body, want)
	}

	send(alice, w.fullIssue(), http.StatusCreated)
	if _, err := w.pool.Exec(context.Background(), "UPDATE projects SET default_assignee_id = $2 WHERE id = $1", w.web, w.ids["bob"]); err != nil {
		t.Fatal(err)
	}
	var defaulted struct {
		AssigneeIDs []uuid.UUID `json:"assignee_ids"`
	}
	if decodeAnswer(t, send(alice, `{"name":"Defaulted"}`, http.StatusCreated), &defaulted); !slices.Equal(defaulted.AssigneeIDs,
		[]uuid.UUID{w.ids["bob"]}) {
		t.Errorf("a work item without assignees is assigned to %v; want bob, Web's default assignee", defaulted.AssigneeIDs)
	}
	send(alice, `{"name":"Fix","label_ids":["`+uuid.NewV7().String()+`"]}`, http.StatusUnprocessableEntity)
	send(w.ids["walt"], `{"name":"Fix"}`, http.StatusForbidden)
	if _, err := w.pool.Exec(context.Background(), "UPDATE projects SET archived_at = now() WHERE id = $1", w.web); err != nil {
		t.Fatal(err)
	}
	send(alice, `{"name":"Fix"}`, http.StatusConflict)
}
````

- [ ] **Step 4: 测试和 lint**

Run: `go -C server test -count=1 -run 'TestEachIssueWriteLocksItsProjectFirst$|TestCreatingAWorkItemStampsItsRows$|TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection$' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 5: 提交**

```bash
git add server/internal/bootstrap/issue_world_test.go server/internal/bootstrap/issue_write_locks_test.go server/internal/bootstrap/issue_writes_test.go
```
```bash
git commit -m "test(M4/P1): a work item's write locks its project first, stamps its rows, and runs on its transaction

Each write on the work-item side has its row: a caller it refuses takes
no lock at all; one it lets through waits first for the projects table,
holds the project's row FOR NO KEY UPDATE and nothing stronger, and
locks nothing of the workspace, so a cascade holding the workspace's
row does not stop it. A creation's seven rows carry one moment within
the request and its caller, the activity the assignee's display name;
and every statement of a write runs on its transaction's connection.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p1a.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试；结构是迁移的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是关键词守卫；静态是 `make lint-web` 的文案和键的核对）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T10.1` | `issue` 模块锁项目取 `FOR UPDATE`：外键对项目的 `FOR KEY SHARE` 要等它 | `internal/modules/project/adapter/postgres` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestEachIssueWriteLocksItsProjectFirst`（`make test`） | 存储；组合 |
| `T10.2` | `issue` 模块锁项目之前先以 `FOR SHARE` 锁工作区行，像 M3 的项目级的写 | `internal/bootstrap` 的 `TestEachIssueWriteLocksItsProjectFirst`（`make test`） | 组合 |
| `T10.3` | 预检找到项目却不判定：被拒绝的调用者先锁住项目 | `internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestEachIssueWriteLocksItsProjectFirst`（`make test`） | 单元；组合 |
| `T10.4` | 锁下的判定经连接池读，不经写的事务 | `internal/bootstrap` 的 `TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`（`make test`） | 组合 |
| `T10.5` | `createIssue` 经连接池读负责人的显示名，不经它的事务 | `internal/bootstrap` 的 `TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`（`make test`） | 组合 |
| `T10.6` | `createIssue` 的动态取自己的时刻，不是工作项的 | `internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestCreatingAWorkItemStampsItsRows`（`make test`） | 单元；组合 |
| `T10.7` | 工作项一侧的写在写入之后以 `FOR SHARE` 锁工作区行，连带的 N 要等它 | `internal/bootstrap` 的 `TestEachIssueWriteLocksItsProjectFirst`（`make test`） | 组合 |
| `T10.8` | 契约中工作项一侧的一个写在 `issueWrites` 里没有一行 | `internal/bootstrap` 的 `TestEachIssueWriteLocksItsProjectFirst`（`make test`） | 组合 |

---

### Task 11: 删除状态问有没有工作项（409 `project.state_in_use`），删除标签从工作项上摘下它

**Files:**
- Create: `server/internal/bootstrap/issue_cascade_test.go`、`server/internal/modules/issue/adapter/postgres/cascade.go`、`server/internal/modules/issue/adapter/postgres/cascade_test.go`、`server/internal/modules/issue/adapter/postgres/queries/cascade.sql`
- Modify: `api/modules/project.yaml`、`api/modules/workspace.yaml`、`server/internal/bootstrap/interleaving_labels_test.go`、`server/internal/bootstrap/interleaving_states_test.go`、`server/internal/bootstrap/project_connection_test.go`、`server/internal/modules/issue/app/cascade.go`、`server/internal/modules/issue/app/cascade_test.go`、`server/internal/modules/issue/module.go`、`server/internal/modules/project/adapter/http/state_writes_test.go`、`server/internal/modules/project/adapter/postgres/failures_test.go`、`server/internal/modules/project/adapter/postgres/label_writes_test.go`、`server/internal/modules/project/adapter/postgres/labels.go`、`server/internal/modules/project/adapter/postgres/queries/labels.sql`、`server/internal/modules/project/app/clock_test.go`、`server/internal/modules/project/app/delete_label.go`、`server/internal/modules/project/app/delete_label_test.go`、`server/internal/modules/project/app/delete_state.go`、`server/internal/modules/project/app/delete_state_test.go`、`server/internal/modules/project/app/fakes_issue_test.go`、`server/internal/modules/project/app/fakes_label_test.go`、`server/internal/modules/project/app/issue_ports.go`、`server/internal/modules/project/app/label_ports.go`、`server/internal/modules/project/domain/errors.go`、`server/internal/modules/project/module.go`、`web/apps/web/core/lib/error-messages.ts`、`web/packages/i18n/src/locales/en/errors.json`、`web/packages/i18n/src/locales/zh-CN/errors.json`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/issue/adapter/postgres/gen/cascade.sql.go`、`server/internal/modules/project/adapter/postgres/gen/labels.sql.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**（spec 2.11；M4 设计 3.5、3.6；M3 设计 3.17）
- `issue`：
  - `queries/cascade.sql`：
    - `StateInUse`：这个状态有没有未删除的工作项，含已归档的；
    - `DetachLabels`：这些标签未删除的 `issue_labels` 行，在给出的时刻、由给出的账户软删除。只有一条语句：删除标签持项目的 `FOR NO KEY UPDATE`，写这些行的每个写都先取它。
  - `adapter/postgres/cascade.go` 的 `Store.StateInUse`、`DetachLabels`；
  - `app/cascade.go`（整个文件）：端口 `StateLabelUses`（`StateInUse`、`DetachLabels`），`NewCascade(rows, uses, batch)`，加 `StateInUse`、`DetachLabel`；
  - `module.go` 的 `Cascade` 接口加这两个方法，包说明。
- `project`：
  - `domain.ErrStateInUse`（409 `project.state_in_use`）；
  - `IssueCascade` 加 `StateInUse`、`DetachLabel`；
  - `DeleteState`（`NewDeleteState(locks, states, issues, tx, clock)`）：`lockRowAndDecide` 之后、守卫的删除之前问 `StateInUse`。所以默认状态、组里唯一的状态有工作项时也答 `state_in_use`；
  - `DeleteLabel`（整个文件）：存储的 `DeleteLabel` 回答删除的 `id`（这个标签和它下面的标签，`queries/labels.sql` 改为 `:many`），用例在同一时刻 `DetachLabel(ids, by, now)`；
  - `module.go` 接上 `d.Issues`。
- 契约：`deleteState` 的 `x-problem-codes` 加 `project.state_in_use`，描述写明；`deleteProject`、`deleteWorkspace` 的描述提到工作项。
- 前端（设计 12 节约束 4）：`web/apps/web/core/lib/error-messages.ts` 和两份 `errors.json` 的 `project_state_in_use`。
- 组合：M3 的状态、标签的交错测试用 `issue.Provide(pool).IssueCascade`；`project_connection_test.go` 的说明提到 `issue` 的连带。

**Tests:**
- 单元：
  - `delete_state_test.go`：`stateDeleted` 带 `StateInUse`；新的两行（Review、默认的 Backlog 有工作项时答 409）；失败的位置；
  - `delete_label_test.go`：`labelDeleted` 带 `DetachLabel`，同一时刻、同一个删除者，删除的 `id`；
  - `clock_test.go`：时钟仍在最后一把锁之后。
- 存储：
  - `TestStateInUse`：已归档的工作项算，已删除的不算，别的状态的不算；
  - `TestDetachLabels`：只摘这些标签未删除的行，写下时刻和删除者；
  - `label_writes_test.go`：`DeleteLabel` 回答删除的 `id`；`failures_test.go`。
- HTTP：`state_writes_test.go` 的 409 `project.state_in_use`（`apitest.Main` 两个方向）。
- 组合：
  - `TestDeletingAStateAWorkItemIsInIsRefused`：
    - Review 有工作项时答 409，Review 保留；工作项归档之后仍是 409，删除之后 Review 可以删除；
    - Todo（组里唯一的）、Backlog（默认的）有工作项时也答 `project.state_in_use`：先查工作项，再查默认状态和组；
  - `TestDeletingALabelTakesItOffTheWorkItems`：删除 Bug 时，Bug 和它下面的 UI 在 Bug 的时刻、由 alice 从工作项上摘下；Feature 的行保留。
- 静态：`make lint-web` 的文案和键的核对。

- [ ] **Step 1: 两条工作项一侧的查询、删除标签回答 `id`、契约，生成**

`api/modules/project.yaml`（修改，4 处）：

````old api/modules/project.yaml
        states and its labels, all at one moment: it is no longer read,
````
````new api/modules/project.yaml
        states, its labels and its work items (with their assignees, labels,
        subscribers and activity), all at one moment: it is no longer read,
````
````old api/modules/project.yaml
        that is both is project.state_default. The state is deleted at the
        moment of the request, by the caller, and its name is free again in
        the project. The role is decided after the workspace and project
        rows are locked.
````
````new api/modules/project.yaml
        that is both is project.state_default. Nor is a state a work item is
        in, an archived one too (project.state_in_use): move the work items
        first; it is checked before the default and the group. The state is
        deleted at the moment of the request, by the caller, and its name is
        free again in the project. The role is decided after the workspace
        and project rows are locked.
````
````old api/modules/project.yaml
      x-problem-codes: [project.state_not_found, forbidden, project.state_default, project.state_last_in_group]
````
````new api/modules/project.yaml
      x-problem-codes: [project.state_not_found, forbidden, project.state_in_use, project.state_default, project.state_last_in_group]
````
````old api/modules/project.yaml
        project. The role is decided after the workspace and project rows
        are locked.
````
````new api/modules/project.yaml
        project; at the same moment and by the same caller they are taken
        off every work item they are on. The role is decided after the
        workspace and project rows are locked.
````

`api/modules/workspace.yaml`（修改，1 处）：

````old api/modules/workspace.yaml
        their memberships, display settings, states and labels are
        soft-deleted in one transaction, at the same moment; the members'
        accounts stay. The slug can name a new workspace at once. Nobody's
````
````new api/modules/workspace.yaml
        their memberships, display settings, states, labels and work items
        (with the work items' assignees, labels, subscribers and activity)
        are soft-deleted in one transaction, at the same moment; the
        members' accounts stay. The slug can name a new workspace at once. Nobody's
````

`server/internal/modules/issue/adapter/postgres/queries/cascade.sql`（新文件，14 行）：

````file server/internal/modules/issue/adapter/postgres/queries/cascade.sql
-- name: StateInUse :one
-- Cascade.StateInUse (M4 design 3.6): whether an undeleted work item, archived or not, is in the state. deleteState
-- reads it under the project's FOR NO KEY UPDATE, which every write of the project's work items takes first (3.5), so
-- it stays as read until the deletion ends. The index on state_id serves it.
SELECT EXISTS (SELECT 1 FROM issues WHERE state_id = sqlc.arg(state_id) AND deleted_at IS NULL)::boolean AS in_use;

-- name: DetachLabels :exec
-- Cascade.DetachLabel (M4 design 3.6): the undeleted rows that put the labels on work items, soft-deleted at the moment
-- and by the account given, in one statement. deleteLabel runs it under the project's FOR NO KEY UPDATE, which every
-- write of these rows takes first (3.5): no other write reaches them meanwhile, so they are not locked by id first. The
-- index on label_id serves it.
UPDATE issue_labels
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE label_id = ANY (sqlc.arg(label_ids)::uuid[]) AND deleted_at IS NULL;
````

`server/internal/modules/project/adapter/postgres/queries/labels.sql`（修改，3 处）：

````old server/internal/modules/project/adapter/postgres/queries/labels.sql
-- name: DeleteLabel :exec
````
````new server/internal/modules/project/adapter/postgres/queries/labels.sql
-- name: DeleteLabel :many
````
````old server/internal/modules/project/adapter/postgres/queries/labels.sql
-- project (3.16), so that lock covers every row the statement writes.
````
````new server/internal/modules/project/adapter/postgres/queries/labels.sql
-- project (3.16), so that lock covers every row the statement writes. It answers their ids, which the work items'
-- rows of them are detached by (M4 design 3.6).
````
````old server/internal/modules/project/adapter/postgres/queries/labels.sql
WHERE (id = sqlc.arg(id) OR parent_id = sqlc.arg(id)) AND deleted_at IS NULL;
````
````new server/internal/modules/project/adapter/postgres/queries/labels.sql
WHERE (id = sqlc.arg(id) OR parent_id = sqlc.arg(id)) AND deleted_at IS NULL
RETURNING id;
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `e8e6fb8819b4dd93fbc0b6c9a9fc3a91c62c101837c1e6a08cc964a64e711e84` | 3111 | `api/dist/openapi.yaml` |
| `f26b516b6e6be107369c5c208c8ed2ff3a741ff6130565f6d4874e1bacb7af09` | 48 | `server/internal/modules/issue/adapter/postgres/gen/cascade.sql.go` |
| `767d070340dbc2a0659ea0be54e074008b47bd8dcd9c37af1d259c3e859ed005` | 292 | `server/internal/modules/project/adapter/postgres/gen/labels.sql.go` |
| `1cd33944a477a63a3dd90b30cb3a129901fe4881537d8d43ebdaf67e565babf8` | 3394 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/issue/adapter/postgres/gen/cascade.sql.go server/internal/modules/project/adapter/postgres/gen/labels.sql.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: `issue` 一侧：存储、`Cascade` 的两个方法和测试**

`server/internal/modules/issue/adapter/postgres/cascade.go`（新文件，29 行）：

````file server/internal/modules/issue/adapter/postgres/cascade.go
package postgresadapter

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
)

// StateInUse reports whether an undeleted work item, archived or not, is
// in the state (M4 design 3.6).
func (s *Store) StateInUse(ctx context.Context, stateID uuid.UUID) (bool, error) {
	inUse, err := s.queries(ctx).StateInUse(ctx, stateID)
	if err != nil {
		return false, fmt.Errorf("read whether state %s is in use: %w", stateID, err)
	}
	return inUse, nil
}

// DetachLabels soft-deletes the undeleted rows that put the labels on
// work items, at now by by (M4 design 3.6).
func (s *Store) DetachLabels(ctx context.Context, labelIDs []uuid.UUID, by uuid.UUID, now time.Time) error {
	if err := s.queries(ctx).DetachLabels(ctx, gen.DetachLabelsParams{At: now, DeletedBy: by, LabelIds: labelIDs}); err != nil {
		return fmt.Errorf("detach labels %v from their work items: %w", labelIDs, err)
	}
	return nil
}
````

`server/internal/modules/issue/adapter/postgres/cascade_test.go`（新文件，142 行）：

````file server/internal/modules/issue/adapter/postgres/cascade_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// A state is in use while an undeleted work item is in it, an archived one
// too; a deleted work item, and one of another state, do not count (M4
// design 3.6). A failed read is its failure, never "not in use".
func TestStateInUse(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	todo := uuid.NewV7()
	exec(t, pool, "INSERT INTO states (id, workspace_id, project_id, name, color) VALUES ($1, $2, $3, 'Todo', '#000')", todo, web.workspace, web.id)
	ctx := context.Background()
	inUse := func(want bool, with string) {
		t.Helper()
		if got, err := s.StateInUse(ctx, web.state); err != nil || got != want {
			t.Errorf("StateInUse() with %s = %v, %v; want %v", with, got, err, want)
		}
	}
	inUse(false, "no work items")
	for n, set := range []string{"deleted_at = now()", "state_id = '" + todo.String() + "'"} {
		id := uuid.NewV7()
		newIssue(t, pool, web, id, n+1, alice)
		exec(t, pool, "UPDATE issues SET "+set+" WHERE id = $1", id)
	}
	inUse(false, "a deleted work item and another state's")
	archived := uuid.NewV7()
	newIssue(t, pool, web, archived, 3, alice)
	exec(t, pool, "UPDATE issues SET archived_at = now() WHERE id = $1", archived)
	inUse(true, "an archived work item")

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if got, err := s.StateInUse(cancelled, web.state); !errors.Is(err, context.Canceled) || got {
		t.Errorf("StateInUse() = %v, %v; want context.Canceled", got, err)
	}
}

// DetachLabels soft-deletes the undeleted rows of the labels named, on
// every work item, at the moment and by the account given, and keeps every
// other column; another label's rows, and a row of a named label deleted
// before, keep every column (M4 design 3.6). A failed write is its
// failure.
func TestDetachLabels(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	bug, ui, feature := newLabel(t, pool, web, "Bug"), newLabel(t, pool, web, "UI"), newLabel(t, pool, web, "Feature")
	one, two := uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, web, one, 1, alice)
	newIssue(t, pool, web, two, 2, alice)
	ctx := context.Background()
	gone := linkRows(web, one, alice, now.Add(-time.Hour), bug)
	for _, r := range []app.LinkRows{gone, linkRows(web, one, alice, now, bug, feature), linkRows(web, two, alice, now, ui, feature)} {
		if err := s.InsertLabels(ctx, r); err != nil {
			t.Fatal(err)
		}
		if r.At.Before(now) {
			exec(t, pool, "UPDATE issue_labels SET deleted_at = updated_at WHERE id = $1", r.IDs[0])
		}
	}
	before := labelRows(t, pool)
	at := now.Add(time.Hour)
	if err := s.DetachLabels(ctx, []uuid.UUID{bug, ui}, bob, at); err != nil {
		t.Fatal(err)
	}
	after := labelRows(t, pool)
	for id, row := range before {
		if row.deleted == nil && (row.label == bug || row.label == ui) {
			row.updatedBy, row.updatedAt, row.deleted = bob, at, &at
		}
		if got := after[id]; got.String() != row.String() {
			t.Errorf("issue_labels %s = %s; want %s", id, got, row)
		}
	}
	if len(after) != len(before) {
		t.Errorf("issue_labels has %d rows, want %d", len(after), len(before))
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := s.DetachLabels(cancelled, []uuid.UUID{feature}, bob, at); !errors.Is(err, context.Canceled) {
		t.Errorf("DetachLabels() = %v; want context.Canceled", err)
	}
	if !maps.EqualFunc(labelRows(t, pool), after, func(a, b labelRow) bool { return a.String() == b.String() }) {
		t.Error("a failed DetachLabels() changed issue_labels")
	}
}

// labelRow is a row of issue_labels as TestDetachLabels reads it: its work
// item and label, who made it and when, who changed it last and when, and
// when it was deleted, nil for never.
type labelRow struct {
	issue, label, createdBy, updatedBy uuid.UUID
	createdAt, updatedAt               time.Time
	deleted                            *time.Time
}

func (r labelRow) String() string {
	deleted := "undeleted"
	if r.deleted != nil {
		deleted = "deleted at " + r.deleted.UTC().String()
	}
	return r.issue.String() + " " + r.label.String() + " made by " + r.createdBy.String() + " at " + r.createdAt.UTC().String() +
		", changed by " + r.updatedBy.String() + " at " + r.updatedAt.UTC().String() + ", " + deleted
}

// labelRows are issue_labels' rows, by id.
func labelRows(t *testing.T, pool *pgxpool.Pool) map[uuid.UUID]labelRow {
	t.Helper()
	rows, err := pool.Query(context.Background(),
		"SELECT id, issue_id, label_id, created_by_id, updated_by_id, created_at, updated_at, deleted_at FROM issue_labels")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[uuid.UUID]labelRow{}
	for rows.Next() {
		var id uuid.UUID
		var r labelRow
		if err := rows.Scan(&id, &r.issue, &r.label, &r.createdBy, &r.updatedBy, &r.createdAt, &r.updatedAt, &r.deleted); err != nil {
			t.Fatal(err)
		}
		out[id] = r
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}
````

`server/internal/modules/issue/app/cascade.go`（整个文件，66 行）：

````whole server/internal/modules/issue/app/cascade.go
package app

import (
	"context"
	"time"
	"uuid"
)

// StateLabelUses are the work items' uses of a project's states and
// labels, as the project module's deletions of them ask about them (M4
// design 3.6), each in the transaction ctx carries.
type StateLabelUses interface {
	// StateInUse reports whether an undeleted work item, archived or not,
	// is in the state.
	StateInUse(ctx context.Context, stateID uuid.UUID) (bool, error)
	// DetachLabels soft-deletes the undeleted rows that put the labels on
	// work items, at now by by.
	DetachLabels(ctx context.Context, labelIDs []uuid.UUID, by uuid.UUID, now time.Time) error
}

// Cascade carries out the project module's deletions on the work items
// (project's IssueCascade port, M4 design 3.3): it soft-deletes a deleted
// project's work-item-side rows in the deletion's transaction, at its
// moment and by its account, through the statements' orchestration
// (deleteProjectRows); it tells a state's deletion whether work items are
// in the state, and takes a deleted label off the work items.
type Cascade struct {
	rows  ProjectRowsDeleter
	uses  StateLabelUses
	batch int
}

// NewCascade returns the cascade over rows and uses, batch rows a
// statement.
func NewCascade(rows ProjectRowsDeleter, uses StateLabelUses, batch int) *Cascade {
	return &Cascade{rows: rows, uses: uses, batch: batch}
}

// DeleteProjectIssues soft-deletes the project's work-item-side rows:
// deleteProject's last step.
func (c *Cascade) DeleteProjectIssues(ctx context.Context, projectID, by uuid.UUID, now time.Time) error {
	return deleteProjectRows(ctx, c.rows, c.batch, projectID, by, now)
}

// DeleteWorkspaceIssues soft-deletes the work-item-side rows of each of the
// projects, in their order: those deleteWorkspace locked and deletes.
func (c *Cascade) DeleteWorkspaceIssues(ctx context.Context, projectIDs []uuid.UUID, by uuid.UUID, now time.Time) error {
	for _, project := range projectIDs {
		if err := deleteProjectRows(ctx, c.rows, c.batch, project, by, now); err != nil {
			return err
		}
	}
	return nil
}

// StateInUse reports whether an undeleted work item, archived or not, is
// in the state: deleteState's check.
func (c *Cascade) StateInUse(ctx context.Context, stateID uuid.UUID) (bool, error) {
	return c.uses.StateInUse(ctx, stateID)
}

// DetachLabel soft-deletes the rows that put the labels on work items, at
// now by by: deleteLabel's last step.
func (c *Cascade) DetachLabel(ctx context.Context, labelIDs []uuid.UUID, by uuid.UUID, now time.Time) error {
	return c.uses.DetachLabels(ctx, labelIDs, by, now)
}
````

`server/internal/modules/issue/app/cascade_test.go`（修改，2 处）：

````old server/internal/modules/issue/app/cascade_test.go
		if err := tt.run(app.NewCascade(f, 2)); err != nil {
````
````new server/internal/modules/issue/app/cascade_test.go
		if err := tt.run(app.NewCascade(f, nil, 2)); err != nil {
````
````old server/internal/modules/issue/app/cascade_test.go
		err := app.NewCascade(f, 2).DeleteWorkspaceIssues(context.WithValue(context.Background(), inTxKey{}, true), []uuid.UUID{web, ops}, by,
````
````new server/internal/modules/issue/app/cascade_test.go
		err := app.NewCascade(f, nil, 2).DeleteWorkspaceIssues(context.WithValue(context.Background(), inTxKey{}, true), []uuid.UUID{web, ops}, by,
````

`server/internal/modules/issue/module.go`（修改，4 处）：

````old server/internal/modules/issue/module.go
// work items' tables, brings creating a work item, and soft-deletes a
// deleted project's rows of them for the project module's deletions (its
// IssueCascade port).
````
````new server/internal/modules/issue/module.go
// work items' tables and brings creating a work item. For the project
// module's deletions (its IssueCascade port) it soft-deletes a deleted
// project's rows of them, tells whether work items are in a state, and
// takes deleted labels off them.
````
````old server/internal/modules/issue/module.go
// caller's transaction, which ctx carries, writes by and now into the rows
// it deletes, and returns a failure as itself, so the caller's whole write
// rolls back.
````
````new server/internal/modules/issue/module.go
// caller's transaction, which ctx carries, and returns a failure as
// itself, so the caller's whole write rolls back; one that deletes writes
// by and now into the rows it deletes.
````
````old server/internal/modules/issue/module.go
	DeleteWorkspaceIssues(ctx context.Context, projectIDs []uuid.UUID, by uuid.UUID, now time.Time) error
````
````new server/internal/modules/issue/module.go
	DeleteWorkspaceIssues(ctx context.Context, projectIDs []uuid.UUID, by uuid.UUID, now time.Time) error
	// StateInUse reports whether an undeleted work item, archived or not,
	// is in the state (M4 design 3.6).
	StateInUse(ctx context.Context, stateID uuid.UUID) (bool, error)
	// DetachLabel soft-deletes the rows that put the labels on work items.
	DetachLabel(ctx context.Context, labelIDs []uuid.UUID, by uuid.UUID, now time.Time) error
````
````old server/internal/modules/issue/module.go
	return Provided{IssueCascade: app.NewCascade(postgresadapter.New(pool), deleteBatch)}
````
````new server/internal/modules/issue/module.go
	store := postgresadapter.New(pool)
	return Provided{IssueCascade: app.NewCascade(store, store, deleteBatch)}
````

- [ ] **Step 3: `project` 一侧：删除状态、删除标签和测试**

`server/internal/modules/project/adapter/http/state_writes_test.go`（修改，2 处）：

````old server/internal/modules/project/adapter/http/state_writes_test.go
		`"detail":"The default state cannot be deleted; make another state the default first."}`
````
````new server/internal/modules/project/adapter/http/state_writes_test.go
		`"detail":"The default state cannot be deleted; make another state the default first."}`
	stateInUseJSON = `{"status":409,"code":"project.state_in_use","title":"Conflict",` +
		`"detail":"Work items are in the state, archived ones too; move them to another state first."}`
````
````old server/internal/modules/project/adapter/http/state_writes_test.go
		{"its group's only state", domain.ErrStateLastInGroup, http.StatusConflict, stateLastInGroupJSON},
````
````new server/internal/modules/project/adapter/http/state_writes_test.go
		{"its group's only state", domain.ErrStateLastInGroup, http.StatusConflict, stateLastInGroupJSON},
		{"a state work items are in", domain.ErrStateInUse, http.StatusConflict, stateInUseJSON},
````

`server/internal/modules/project/adapter/postgres/failures_test.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/failures_test.go
	if err := s.DeleteLabel(cancelled, bug, alice, now); !failed(err) {
		t.Errorf("DeleteLabel() = %v; want context.Canceled", err)
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
	if ids, err := s.DeleteLabel(cancelled, bug, alice, now); !failed(err) || ids != nil {
		t.Errorf("DeleteLabel() = %v, %v; want context.Canceled", ids, err)
````

`server/internal/modules/project/adapter/postgres/label_writes_test.go`（修改，5 处）：

````old server/internal/modules/project/adapter/postgres/label_writes_test.go
	"maps"
````
````new server/internal/modules/project/adapter/postgres/label_writes_test.go
	"maps"
	"slices"
````
````old server/internal/modules/project/adapter/postgres/label_writes_test.go
// account given, deleted_at and updated_at alike, and touches no other
// column: Gone, under Bug and deleted before, keeps its moment; Feature,
// Feature's Docs, and Ops's Bug keep every column. Deleted again, at a
// later moment, Bug changes nothing. Deleting a child, Docs, deletes it
// alone.
````
````new server/internal/modules/project/adapter/postgres/label_writes_test.go
// account given, deleted_at and updated_at alike, touches no other column,
// and answers their ids, which their rows on work items are taken off by
// (M4 design 3.6): Gone, under Bug and deleted before, keeps its moment
// and is not answered; Feature, Feature's Docs, and Ops's Bug keep every
// column. Deleted again, at a later moment, Bug changes nothing and
// answers no id. Deleting a child, Docs, deletes it alone.
````
````old server/internal/modules/project/adapter/postgres/label_writes_test.go
	if err := w.s.DeleteLabel(context.Background(), bug.ID, w.alice, now); err != nil {
		t.Errorf("DeleteLabel(Bug) = %v", err)
````
````new server/internal/modules/project/adapter/postgres/label_writes_test.go
	sorted := func(ids ...uuid.UUID) []uuid.UUID { return slices.SortedFunc(slices.Values(ids), uuid.UUID.Compare) }
	if ids, err := w.s.DeleteLabel(context.Background(), bug.ID, w.alice, now); err != nil || !slices.Equal(sorted(ids...), sorted(bug.ID, ui.ID)) {
		t.Errorf("DeleteLabel(Bug) = %v, %v; want Bug's and UI's ids", ids, err)
````
````old server/internal/modules/project/adapter/postgres/label_writes_test.go
	if err := w.s.DeleteLabel(context.Background(), bug.ID, w.alice, now.Add(time.Minute)); err != nil {
		t.Errorf("DeleteLabel(Bug) again = %v", err)
````
````new server/internal/modules/project/adapter/postgres/label_writes_test.go
	if ids, err := w.s.DeleteLabel(context.Background(), bug.ID, w.alice, now.Add(time.Minute)); err != nil || len(ids) != 0 {
		t.Errorf("DeleteLabel(Bug) again = %v, %v; want no id", ids, err)
````
````old server/internal/modules/project/adapter/postgres/label_writes_test.go
	if err := w.s.DeleteLabel(context.Background(), docs.ID, w.alice, now); err != nil {
		t.Errorf("DeleteLabel(Docs) = %v", err)
````
````new server/internal/modules/project/adapter/postgres/label_writes_test.go
	if ids, err := w.s.DeleteLabel(context.Background(), docs.ID, w.alice, now); err != nil || !slices.Equal(ids, []uuid.UUID{docs.ID}) {
		t.Errorf("DeleteLabel(Docs) = %v, %v; want Docs's id", ids, err)
````

`server/internal/modules/project/adapter/postgres/labels.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/labels.go
// under it, by the account by at now, in one statement (app.LabelDeleter).
func (s *Store) DeleteLabel(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	if err := s.queries(ctx).DeleteLabel(ctx, gen.DeleteLabelParams{ID: id, DeletedBy: by, Now: now}); err != nil {
		return fmt.Errorf("delete label %s: %w", id, err)
	}
	return nil
````
````new server/internal/modules/project/adapter/postgres/labels.go
// under it, by the account by at now, in one statement, and answers their
// ids (app.LabelDeleter).
func (s *Store) DeleteLabel(ctx context.Context, id, by uuid.UUID, now time.Time) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteLabel(ctx, gen.DeleteLabelParams{ID: id, DeletedBy: by, Now: now})
	if err != nil {
		return nil, fmt.Errorf("delete label %s: %w", id, err)
	}
	return ids, nil
````

`server/internal/modules/project/app/clock_test.go`（修改，1 处）：

````old server/internal/modules/project/app/clock_test.go
		}, labelDeleted(bob, webBug, webID)},
````
````new server/internal/modules/project/app/clock_test.go
		}, labelDeleted(bob, webBug, webID, webBug, webUI)},
````

`server/internal/modules/project/app/delete_label.go`（整个文件，53 行）：

````whole server/internal/modules/project/app/delete_label.go
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DeleteLabel deletes a label and the labels under it: DELETE
// /api/v0/labels/{label_id} (M3 design 3.16, M4 design 3.6).
type DeleteLabel struct {
	locks  Locks
	labels LabelDeleter
	issues IssueCascade
	tx     shared.TxManager
	clock  Clock
}

// NewDeleteLabel returns the use case.
func NewDeleteLabel(locks Locks, labels LabelDeleter, issues IssueCascade, tx shared.TxManager, clock Clock) *DeleteLabel {
	return &DeleteLabel{locks: locks, labels: labels, issues: issues, tx: tx, clock: clock}
}

// Execute, in one transaction, in the order of M3 design 3.6 and M4 design
// 3.5: the label's locks (lockRowAndDecide: the label read for its project
// and workspace, the workspace FOR SHARE, the project FOR NO KEY UPDATE,
// which every write of its labels and of the work items' labels takes,
// the label read again) and the decision on label.delete; an archived
// project's labels are deleted as any other's (M3 design 3.19). Then the
// label and the labels under it, in one statement, by the caller at the
// time the clock gives under the locks; then, at that time and by the
// caller, the rows that put them on work items (IssueCascade.DetachLabel,
// M4 design 3.6).
func (u *DeleteLabel) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	return u.tx.WithinTx(ctx, func(ctx context.Context) error {
		_, l, err := lockRowAndDecide(ctx, u.locks, actor, labelWrite(id, domain.ActionLabelDelete, u.labels))
		if err != nil {
			return err
		}
		now := u.clock.Now()
		ids, err := u.labels.DeleteLabel(ctx, l.ID, actor.UserID, now)
		if err != nil {
			return err
		}
		return u.issues.DetachLabel(ctx, ids, actor.UserID, now)
	})
}
````

`server/internal/modules/project/app/delete_label_test.go`（修改，6 处）：

````old server/internal/modules/project/app/delete_label_test.go
	return app.NewDeleteLabel(f.locks(), l, f.tx, clockAt{clockNow, f.log}), f, l
````
````new server/internal/modules/project/app/delete_label_test.go
	return app.NewDeleteLabel(f.locks(), l, f.issues, f.tx, clockAt{clockNow, f.log}), f, l
````
````old server/internal/modules/project/app/delete_label_test.go
// project: its locks and decision, the clock, the deletion by user at that
// time.
func labelDeleted(user, id, project uuid.UUID) []string {
	return append(labelLocked(id, project, user, domain.ActionLabelDelete), "Now",
		fmt.Sprintf("DeleteLabel %s by %s at %s", id, user, clockNow.Format(timeFormat)))
````
````new server/internal/modules/project/app/delete_label_test.go
// project, which takes the labels gone: its locks and decision, the clock,
// the deletion by user at that time, then gone, in id order, taken off the
// work items by user at that time.
func labelDeleted(user, id, project uuid.UUID, gone ...uuid.UUID) []string {
	at := clockNow.Format(timeFormat)
	return append(labelLocked(id, project, user, domain.ActionLabelDelete), "Now", fmt.Sprintf("DeleteLabel %s by %s at %s", id, user, at),
		fmt.Sprintf("DetachLabel %v by %s at %s", slices.SortedFunc(slices.Values(gone), uuid.UUID.Compare), user, at))
````
````old server/internal/modules/project/app/delete_label_test.go
// 3.16, locks the label's project, decides, reads the clock, then deletes
// the label and the labels under it by the caller at that time: Bug takes
// UI with it; UI, under Bug, and Feature go alone; and ops's Docs, of an
// archived project, goes as any other (3.19).
````
````new server/internal/modules/project/app/delete_label_test.go
// 3.16 and M4 design 3.5, locks the label's project, decides, reads the
// clock, deletes the label and the labels under it by the caller at that
// time, then takes them off the work items, in the transaction, by the
// caller at that time: Bug takes UI with it; UI, under Bug, and Feature go
// alone; and ops's Docs, of an archived project, goes as any other (M3
// design 3.19).
````
````old server/internal/modules/project/app/delete_label_test.go
			if calls := labelDeleted(bob, tt.id, tt.project); !slices.Equal(f.log.calls, calls) {
````
````new server/internal/modules/project/app/delete_label_test.go
			if calls := labelDeleted(bob, tt.id, tt.project, tt.gone...); !slices.Equal(f.log.calls, calls) {
````
````old server/internal/modules/project/app/delete_label_test.go
	all := labelDeleted(bob, webBug, webID)
````
````new server/internal/modules/project/app/delete_label_test.go
	all := labelDeleted(bob, webBug, webID, webBug, webUI)
````
````old server/internal/modules/project/app/delete_label_test.go
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 8},
````
````new server/internal/modules/project/app/delete_label_test.go
		{"taking them off the work items", func(f *writeFixture) { f.issues.errs = map[string]error{"DetachLabel": errDisk} }, 9},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 9},
````

`server/internal/modules/project/app/delete_state.go`（修改，5 处）：

````old server/internal/modules/project/app/delete_state.go
// design 3.17).
````
````new server/internal/modules/project/app/delete_state.go
// design 3.17, M4 design 3.6).
````
````old server/internal/modules/project/app/delete_state.go
	states StateDeleter
````
````new server/internal/modules/project/app/delete_state.go
	states StateDeleter
	issues IssueCascade
````
````old server/internal/modules/project/app/delete_state.go
func NewDeleteState(locks Locks, states StateDeleter, tx shared.TxManager, clock Clock) *DeleteState {
	return &DeleteState{locks: locks, states: states, tx: tx, clock: clock}
````
````new server/internal/modules/project/app/delete_state.go
func NewDeleteState(locks Locks, states StateDeleter, issues IssueCascade, tx shared.TxManager, clock Clock) *DeleteState {
	return &DeleteState{locks: locks, states: states, issues: issues, tx: tx, clock: clock}
````
````old server/internal/modules/project/app/delete_state.go
// Execute, in one transaction and in the order of M3 design 3.6: the
// state's locks (lockRowAndDecide: the state read for its project and
// workspace, the workspace FOR SHARE, the project FOR NO KEY UPDATE, the
// state read again) and the decision on state.delete; an archived
// project's states are deleted as any other's (3.19). Then, at the time the
// clock gives under the locks, the guarded deletion (3.17), which passes
// over the project's default state: the state read again answers why
// (deletedNothing). Then the states its group keeps without it
// (domain.CheckGroupKept): a refusal rolls the deletion back.
````
````new server/internal/modules/project/app/delete_state.go
// Execute, in one transaction and in the order of M3 design 3.6 and M4
// design 3.5: the state's locks (lockRowAndDecide: the state read for its
// project and workspace, the workspace FOR SHARE, the project FOR NO KEY
// UPDATE, the state read again) and the decision on state.delete; an
// archived project's states are deleted as any other's (M3 design 3.19).
// Then whether a work item is in the state, an archived one too
// (IssueCascade.StateInUse): one is project.state_in_use (M4 design 3.6),
// and none enters or leaves the state while the project's lock is held.
// Then, at the time the clock gives under the locks, the guarded deletion
// (M3 design 3.17), which passes over the project's default state: the
// state read again answers why (deletedNothing). Then the states its group
// keeps without it (domain.CheckGroupKept): a refusal rolls the deletion
// back.
````
````old server/internal/modules/project/app/delete_state.go
		_, s, err := lockRowAndDecide(ctx, u.locks, actor, rw)
		if err != nil {
			return err
		}
````
````new server/internal/modules/project/app/delete_state.go
		_, s, err := lockRowAndDecide(ctx, u.locks, actor, rw)
		if err != nil {
			return err
		}
		inUse, err := u.issues.StateInUse(ctx, s.ID)
		switch {
		case err != nil:
			return err
		case inUse:
			return domain.ErrStateInUse
		}
````

`server/internal/modules/project/app/delete_state_test.go`（修改，11 处）：

````old server/internal/modules/project/app/delete_state_test.go
	return app.NewDeleteState(f.locks(), s, f.tx, clockAt{clockNow, f.log}), f, s
````
````new server/internal/modules/project/app/delete_state_test.go
	return app.NewDeleteState(f.locks(), s, f.issues, f.tx, clockAt{clockNow, f.log}), f, s
````
````old server/internal/modules/project/app/delete_state_test.go
// up to the deletion: its locks and decision, the clock, the deletion by
// user at that time.
````
````new server/internal/modules/project/app/delete_state_test.go
// up to the deletion: its locks and decision, whether a work item is in
// the state, the clock, the deletion by user at that time.
````
````old server/internal/modules/project/app/delete_state_test.go
	return append(stateLocked(id, project, user, domain.ActionStateDelete), "Now",
````
````new server/internal/modules/project/app/delete_state_test.go
	return append(stateLocked(id, project, user, domain.ActionStateDelete), "StateInUse "+id.String(), "Now",
````
````old server/internal/modules/project/app/delete_state_test.go
// 3.17, locks the state's project, decides, reads the clock, deletes the
// state by the caller at that time, then counts the states its group keeps:
// Review leaves In Progress in the started group. ops, archived, has its
// states deleted as any other's (3.19): Icebox leaves Backlog in the
// backlog group.
````
````new server/internal/modules/project/app/delete_state_test.go
// 3.17 and M4 design 3.5, locks the state's project, decides, asks whether
// a work item is in the state, in the transaction, reads the clock,
// deletes the state by the caller at that time, then counts the states its
// group keeps: Review leaves In Progress in the started group. ops,
// archived, has its states deleted as any other's (M3 design 3.19): Icebox
// leaves Backlog in the backlog group.
````
````old server/internal/modules/project/app/delete_state_test.go
// past the decision: Backlog, the default and its group's only state,
// which the deletion passes over and the state read again answers
````
````new server/internal/modules/project/app/delete_state_test.go
// past the decision: a state a work item is in, project.state_in_use
// before the deletion, Backlog's too, though it is the default and its
// group's only state: the work items are checked first (M4 design 3.5).
// Backlog, which the deletion passes over and the state read again answers
````
````old server/internal/modules/project/app/delete_state_test.go
// gone when read again,
// project.state_not_found; Todo, the only state of its group, deleted and
// project.state_last_in_group, which the transaction rolls back. A
// deletion that passed over a state neither gone nor the default is the
// write's own error. A refusal before the deletion leaves the states as
// they were.
````
````new server/internal/modules/project/app/delete_state_test.go
// gone when read again, project.state_not_found; Todo, the only state of
// its group, deleted and project.state_last_in_group, which the
// transaction rolls back. A deletion that passed over a state neither gone
// nor the default is the write's own error. A refusal before the deletion
// leaves the states as they were.
````
````old server/internal/modules/project/app/delete_state_test.go
	readAgain := func(id uuid.UUID) []string { return append(stateDeleted(bob, id, webID), "StateByID "+id.String()) }
````
````new server/internal/modules/project/app/delete_state_test.go
	readAgain := func(id uuid.UUID) []string { return append(stateDeleted(bob, id, webID), "StateByID "+id.String()) }
	inUse := func(id uuid.UUID) func(f *writeFixture, _ *fakeStates) {
		return func(f *writeFixture, _ *fakeStates) { f.issues.inUse = map[uuid.UUID]bool{id: true} }
	}
	checked := func(id uuid.UUID) []string {
		return append(stateLocked(id, webID, bob, domain.ActionStateDelete), "StateInUse "+id.String())
	}
````
````old server/internal/modules/project/app/delete_state_test.go
		{"a member", as(alice), webReview, nil, shared.Forbidden(), stateLocked(webReview, webID, alice, domain.ActionStateDelete)},
````
````new server/internal/modules/project/app/delete_state_test.go
		{"a member", as(alice), webReview, nil, shared.Forbidden(), stateLocked(webReview, webID, alice, domain.ActionStateDelete)},
		{"Review, which a work item is in", as(bob), webReview, inUse(webReview), domain.ErrStateInUse, checked(webReview)},
		{"Backlog, which a work item is in", as(bob), webBacklog, inUse(webBacklog), domain.ErrStateInUse, checked(webBacklog)},
````
````old server/internal/modules/project/app/delete_state_test.go
			beforeTheDeletion := len(tt.calls) <= len(stateLocked(tt.id, webID, bob, domain.ActionStateDelete))
````
````new server/internal/modules/project/app/delete_state_test.go
			beforeTheDeletion := len(tt.calls) < len(stateDeleted(bob, tt.id, webID))
````
````old server/internal/modules/project/app/delete_state_test.go
		{"the deletion", fail("DeleteState"), 8},
		{"the count of the group", fail("CountGroupStates"), 9},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 9},
````
````new server/internal/modules/project/app/delete_state_test.go
		{"the work items' check", func(f *writeFixture) { f.issues.errs = map[string]error{"StateInUse": errDisk} }, 7},
		{"the deletion", fail("DeleteState"), 9},
		{"the count of the group", fail("CountGroupStates"), 10},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 10},
````
````old server/internal/modules/project/app/delete_state_test.go
			err := app.NewDeleteState(f.locks(), afterDeletion{s, tt.set}, f.tx, clockAt{clockNow, f.log}).Execute(as(bob), webBacklog)
````
````new server/internal/modules/project/app/delete_state_test.go
			err := app.NewDeleteState(f.locks(), afterDeletion{s, tt.set}, f.issues, f.tx, clockAt{clockNow, f.log}).Execute(as(bob), webBacklog)
````

`server/internal/modules/project/app/fakes_issue_test.go`（修改，3 处）：

````old server/internal/modules/project/app/fakes_issue_test.go
// as the module wraps it.
````
````new server/internal/modules/project/app/fakes_issue_test.go
// as the module wraps it. The states in inUse have work items.
````
````old server/internal/modules/project/app/fakes_issue_test.go
	log  *callLog
	errs map[string]error
````
````new server/internal/modules/project/app/fakes_issue_test.go
	log   *callLog
	errs  map[string]error
	inUse map[uuid.UUID]bool
````
````old server/internal/modules/project/app/fakes_issue_test.go
	return f.call(ctx, "DeleteWorkspaceIssues", "%v by %s at %s", projectIDs, by, now.Format(timeFormat))
}

````
````new server/internal/modules/project/app/fakes_issue_test.go
	return f.call(ctx, "DeleteWorkspaceIssues", "%v by %s at %s", projectIDs, by, now.Format(timeFormat))
}

func (f *fakeIssues) StateInUse(ctx context.Context, stateID uuid.UUID) (bool, error) {
	if err := f.call(ctx, "StateInUse", "%s", stateID); err != nil {
		return false, err
	}
	return f.inUse[stateID], nil
}

func (f *fakeIssues) DetachLabel(ctx context.Context, labelIDs []uuid.UUID, by uuid.UUID, now time.Time) error {
	return f.call(ctx, "DetachLabel", "%v by %s at %s", labelIDs, by, now.Format(timeFormat))
}

````

`server/internal/modules/project/app/fakes_label_test.go`（修改，4 处）：

````old server/internal/modules/project/app/fakes_label_test.go
// DeleteLabel deletes the label id and the labels under it.
func (f *fakeLabels) DeleteLabel(ctx context.Context, id, by uuid.UUID, now time.Time) error {
````
````new server/internal/modules/project/app/fakes_label_test.go
// DeleteLabel deletes the label id and the labels under it, and answers
// their ids, in id order.
func (f *fakeLabels) DeleteLabel(ctx context.Context, id, by uuid.UUID, now time.Time) ([]uuid.UUID, error) {
````
````old server/internal/modules/project/app/fakes_label_test.go
		return err
	}
````
````new server/internal/modules/project/app/fakes_label_test.go
		return nil, err
	}
	var ids []uuid.UUID
````
````old server/internal/modules/project/app/fakes_label_test.go
		if key == id || (l.ParentID != nil && *l.ParentID == id) {
````
````new server/internal/modules/project/app/fakes_label_test.go
		if key == id || (l.ParentID != nil && *l.ParentID == id) {
			ids = append(ids, key)
````
````old server/internal/modules/project/app/fakes_label_test.go
	}
	return nil
````
````new server/internal/modules/project/app/fakes_label_test.go
	}
	return slices.SortedFunc(slices.Values(ids), uuid.UUID.Compare), nil
````

`server/internal/modules/project/app/issue_ports.go`（修改，2 处）：

````old server/internal/modules/project/app/issue_ports.go
// the project's or the workspace's lock in, writes by and now into the
// rows it deletes, and returns a failure as itself, so the caller's whole
// write rolls back.
````
````new server/internal/modules/project/app/issue_ports.go
// the project's or the workspace's lock in, and returns a failure as
// itself, so the caller's whole write rolls back; one that deletes writes
// by and now into the rows it deletes.
````
````old server/internal/modules/project/app/issue_ports.go
	DeleteWorkspaceIssues(ctx context.Context, projectIDs []uuid.UUID, by uuid.UUID, now time.Time) error
````
````new server/internal/modules/project/app/issue_ports.go
	DeleteWorkspaceIssues(ctx context.Context, projectIDs []uuid.UUID, by uuid.UUID, now time.Time) error
	// StateInUse reports whether an undeleted work item, archived or not,
	// is in the state: deleteState's check, under the project's lock.
	StateInUse(ctx context.Context, stateID uuid.UUID) (bool, error)
	// DetachLabel soft-deletes the rows that put the labels on work items:
	// deleteLabel's last step, on the label and the labels under it.
	DetachLabel(ctx context.Context, labelIDs []uuid.UUID, by uuid.UUID, now time.Time) error
````

`server/internal/modules/project/app/label_ports.go`（修改，1 处）：

````old server/internal/modules/project/app/label_ports.go
	// under it, by the account by at now, in one statement.
	DeleteLabel(ctx context.Context, id, by uuid.UUID, now time.Time) error
````
````new server/internal/modules/project/app/label_ports.go
	// under it, by the account by at now, in one statement, and answers
	// their ids.
	DeleteLabel(ctx context.Context, id, by uuid.UUID, now time.Time) ([]uuid.UUID, error)
````

`server/internal/modules/project/domain/errors.go`（修改，1 处）：

````old server/internal/modules/project/domain/errors.go
		"The state is the only one of its group, and every group keeps one; add another to the group first.")
````
````new server/internal/modules/project/domain/errors.go
		"The state is the only one of its group, and every group keeps one; add another to the group first.")
	// ErrStateInUse answers the deletion of a state an undeleted work item,
	// archived or not, is in (M4 design 3.6).
	ErrStateInUse = shared.NewError(shared.KindConflict, "project.state_in_use",
		"Work items are in the state, archived ones too; move them to another state first.")
````

`server/internal/modules/project/module.go`（修改，2 处）：

````old server/internal/modules/project/module.go
		DeleteState:         app.NewDeleteState(locks, store, d.Tx, d.Clock),
````
````new server/internal/modules/project/module.go
		DeleteState:         app.NewDeleteState(locks, store, d.Issues, d.Tx, d.Clock),
````
````old server/internal/modules/project/module.go
		DeleteLabel:         app.NewDeleteLabel(locks, store, d.Tx, d.Clock),
````
````new server/internal/modules/project/module.go
		DeleteLabel:         app.NewDeleteLabel(locks, store, d.Issues, d.Tx, d.Clock),
````

- [ ] **Step 4: 新码的文案**

`web/apps/web/core/lib/error-messages.ts`（修改，1 处）：

````old web/apps/web/core/lib/error-messages.ts
  "project.state_default": "errors.project_state_default",
````
````new web/apps/web/core/lib/error-messages.ts
  "project.state_default": "errors.project_state_default",
  "project.state_in_use": "errors.project_state_in_use",
````

`web/packages/i18n/src/locales/en/errors.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/errors.json
    "project_state_default": "The default state cannot be deleted. Make another state the default first.",
````
````new web/packages/i18n/src/locales/en/errors.json
    "project_state_default": "The default state cannot be deleted. Make another state the default first.",
    "project_state_in_use": "Work items are in this state, archived ones too. Move them to another state first.",
````

`web/packages/i18n/src/locales/zh-CN/errors.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/errors.json
    "project_state_default": "默认状态不能删除。请先把另一个状态设为默认。",
````
````new web/packages/i18n/src/locales/zh-CN/errors.json
    "project_state_default": "默认状态不能删除。请先把另一个状态设为默认。",
    "project_state_in_use": "这个状态下还有工作项（包括已归档的）。请先把它们移到别的状态。",
````

- [ ] **Step 5: 组合的测试**

`server/internal/bootstrap/interleaving_labels_test.go`（修改，4 处）：

````old server/internal/bootstrap/interleaving_labels_test.go
	"uuid"

````
````new server/internal/bootstrap/interleaving_labels_test.go
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
````
````old server/internal/bootstrap/interleaving_labels_test.go
func (s labelWrittenHolding) DeleteLabel(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	if err := s.Store.DeleteLabel(ctx, id, by, now); err != nil {
		return err
````
````new server/internal/bootstrap/interleaving_labels_test.go
func (s labelWrittenHolding) DeleteLabel(ctx context.Context, id, by uuid.UUID, now time.Time) ([]uuid.UUID, error) {
	ids, err := s.Store.DeleteLabel(ctx, id, by, now)
	if err != nil {
		return nil, err
````
````old server/internal/bootstrap/interleaving_labels_test.go
	return s.gate.wait(ctx)
````
````new server/internal/bootstrap/interleaving_labels_test.go
	return ids, s.gate.wait(ctx)
````
````old server/internal/bootstrap/interleaving_labels_test.go
		return projectapp.NewDeleteLabel(w.projectLocks(), labels, w.tx(), clock.System{}).Execute(w.asBob(ctx), ids[name])
````
````new server/internal/bootstrap/interleaving_labels_test.go
		return projectapp.NewDeleteLabel(w.projectLocks(), labels, issue.Provide(w.pool).IssueCascade, w.tx(), clock.System{}).Execute(w.asBob(ctx),
			ids[name])
````

`server/internal/bootstrap/interleaving_states_test.go`（修改，2 处）：

````old server/internal/bootstrap/interleaving_states_test.go
	"uuid"

````
````new server/internal/bootstrap/interleaving_states_test.go
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
````
````old server/internal/bootstrap/interleaving_states_test.go
		return projectapp.NewDeleteState(w.projectLocks(), states, w.tx(), clock.System{}).Execute(w.asBob(ctx), ids[name])
````
````new server/internal/bootstrap/interleaving_states_test.go
		return projectapp.NewDeleteState(w.projectLocks(), states, issue.Provide(w.pool).IssueCascade, w.tx(), clock.System{}).Execute(w.asBob(ctx),
			ids[name])
````

`server/internal/bootstrap/issue_cascade_test.go`（新文件，97 行）：

````file server/internal/bootstrap/issue_cascade_test.go
package bootstrap

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// deleteState refuses a state a work item is in, an archived one too, as
// bootstrap wires it (M4 design 3.6): in Web of the world, alice adds
// Review to the started group, and bob creates a work item in it, one in
// Todo and one in the default state, Backlog. Deleting Review answers 409
// project.state_in_use and keeps it, and so it does once its work item is
// archived; once the work item is deleted, Review is deleted. Todo, its
// group's only state, and Backlog, the default and its group's only state,
// each answer project.state_in_use: the work items are checked before the
// default and the group (M4 design 3.5).
func TestDeletingAStateAWorkItemIsInIsRefused(t *testing.T) {
	w := newIssueWorld(t)
	web := "/api/v0/projects/" + w.web.String()
	var review, fix struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, web+"/states", `{"name":"Review","color":"#F59E0B","group":"started"}`,
		http.StatusCreated), &review)
	decodeAnswer(t, w.send(t, "bob", http.MethodPost, web+"/issues", `{"name":"Fix","state_id":"`+review.ID.String()+`"}`,
		http.StatusCreated), &fix)
	w.send(t, "bob", http.MethodPost, web+"/issues", `{"name":"Plan","state_id":"`+w.todo.String()+`"}`, http.StatusCreated)
	w.send(t, "bob", http.MethodPost, web+"/issues", `{"name":"Idea"}`, http.StatusCreated)
	refused := func(state uuid.UUID, what string) {
		t.Helper()
		answer := w.send(t, "alice", http.MethodDelete, "/api/v0/states/"+state.String(), "", http.StatusConflict)
		if got := refusalOf(t, []byte(answer), true); got != "project.state_in_use" {
			t.Errorf("deleting %s = %s; want project.state_in_use", what, answer)
		}
		var kept bool
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at IS NULL FROM states WHERE id = $1", state).Scan(&kept); err != nil || !kept {
			t.Errorf("%s after its refused deletion: kept %v, %v; want kept", what, kept, err)
		}
	}
	fixed := func(set string) {
		t.Helper()
		if _, err := w.pool.Exec(context.Background(), "UPDATE issues SET "+set+" WHERE id = $1", fix.ID); err != nil {
			t.Fatal(err)
		}
	}
	refused(review.ID, "Review, with a work item in it")
	refused(w.todo, "Todo, its group's only state, with a work item in it")
	refused(stateID(t, w.pool, w.web, "Backlog"), "Backlog, the default and its group's only state, with a work item in it")
	fixed("archived_at = now()")
	refused(review.ID, "Review, with its work item archived")
	fixed("deleted_at = now()")
	w.send(t, "alice", http.MethodDelete, "/api/v0/states/"+review.ID.String(), "", http.StatusNoContent)
}

// deleteLabel takes its label, and the labels under it, off every work
// item, at the moment of the deletion and by its caller, as bootstrap
// wires it (M4 design 3.6): in Web of the world, alice adds UI under Bug,
// and Feature; bob creates a work item labelled Bug and Feature, and one
// labelled UI. Once alice deletes Bug, the rows that put Bug and UI on
// them are deleted at Bug's moment, by her; Feature's row is kept.
func TestDeletingALabelTakesItOffTheWorkItems(t *testing.T) {
	w := newIssueWorld(t)
	web := "/api/v0/projects/" + w.web.String()
	var ui, feature struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, web+"/labels", `{"name":"UI","parent_id":"`+w.bug.String()+`"}`, http.StatusCreated), &ui)
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, web+"/labels", `{"name":"Feature"}`, http.StatusCreated), &feature)
	for _, labels := range [][]uuid.UUID{{w.bug, feature.ID}, {ui.ID}} {
		quoted := make([]string, len(labels))
		for i, id := range labels {
			quoted[i] = `"` + id.String() + `"`
		}
		w.send(t, "bob", http.MethodPost, web+"/issues", `{"name":"Fix","label_ids":[`+strings.Join(quoted, ",")+`]}`, http.StatusCreated)
	}
	w.send(t, "alice", http.MethodDelete, "/api/v0/labels/"+w.bug.String(), "", http.StatusNoContent)

	rows, err := w.pool.Query(pgtest.Soon(t), `SELECT format('%s %s', l.name, CASE WHEN il.deleted_at IS NULL THEN 'kept'
		WHEN il.deleted_at = b.deleted_at AND il.updated_at = b.deleted_at AND il.updated_by_id = $2 THEN 'taken off by alice with Bug'
		ELSE 'taken off otherwise' END)
		FROM issue_labels il JOIN labels l ON l.id = il.label_id, labels b WHERE b.id = $1 ORDER BY l.name`, w.bug, w.ids["alice"])
	if err != nil {
		t.Fatal(err)
	}
	got, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if want := []string{"Bug taken off by alice with Bug", "Feature kept", "UI taken off by alice with Bug"}; err != nil || !slices.Equal(got, want) {
		t.Errorf("the work items' labels after Bug's deletion: %q, %v; want %q", got, err, want)
	}
}
````

`server/internal/bootstrap/project_connection_test.go`（修改，1 处）：

````old server/internal/bootstrap/project_connection_test.go
// its writes and its answer. The project module and the Authorizer are
// wired as bootstrap wires them, on a pool of one connection: a statement
// sent through the pool rather than the transaction would wait for a second
// connection that never comes, and its request fail at the request's
// deadline. alice, acme's admin, creates Ops, changes Web, archives and
// unarchives it, changes her display settings in it and adds bob, whose
// ended membership she restores; carol joins it anew; alice makes carol an
// admin of Web and removes bob; she creates QA in it, alone in the
// completed group, and moves it to the started group, which the count of
// its group refuses (409 project.state_last_in_group); she renames it
// Checked, creates Done in the completed group, deletes Checked, which
// counts the group again, and makes Done Web's default; she creates the
// label Bug in it, UI under Bug, and Icons under UI, which the parent's
// read refuses (422: labels have two levels); she moves Bug under UI,
// which the same read refuses, renames UI Widgets at the top, moves it
// back under Bug, and deletes Bug, Widgets with it; carol, an admin,
// leaves Web; alice deletes both projects.
````
````new server/internal/bootstrap/project_connection_test.go
// what it asks of the work items (issue's cascade), its writes and its
// answer. The project module and the Authorizer are wired as bootstrap
// wires them, on a pool of one connection: a statement sent through the
// pool rather than the transaction would wait for a second connection that
// never comes, and its request fail at the request's deadline. alice,
// acme's admin, creates Ops, changes Web, archives and unarchives it,
// changes her display settings in it and adds bob, whose ended membership
// she restores; carol joins it anew; alice makes carol an admin of Web and
// removes bob; she creates QA in it, alone in the completed group, and
// moves it to the started group, which the count of its group refuses (409
// project.state_last_in_group); she renames it Checked, creates Done in the
// completed group, deletes Checked, which asks whether a work item is in
// it and counts the group again, and makes Done Web's default; she creates
// the label Bug in it, UI under Bug, and Icons under UI, which the
// parent's read refuses (422: labels have two levels); she moves Bug under
// UI, which the same read refuses, renames UI Widgets at the top, moves it
// back under Bug, and deletes Bug, Widgets with it, which takes them off
// the work items; carol, an admin, leaves Web; alice deletes both
// projects.
````

- [ ] **Step 6: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/... ./internal/modules/project/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestDeletingAStateAWorkItemIsInIsRefused$|TestDeletingALabelTakesItOffTheWorkItems$' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过；关键词守卫 `keywords: 72 rules, 23 exceptions, no hits.`

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 120 个全部通过（P6、P7 删除状态、标签时还没有工作项）。

- [ ] **Step 7: 提交**

```bash
git add api/modules/project.yaml api/modules/workspace.yaml server/internal/bootstrap/interleaving_labels_test.go server/internal/bootstrap/interleaving_states_test.go server/internal/bootstrap/issue_cascade_test.go server/internal/bootstrap/project_connection_test.go server/internal/modules/issue/adapter/postgres/cascade.go server/internal/modules/issue/adapter/postgres/cascade_test.go server/internal/modules/issue/adapter/postgres/queries/cascade.sql server/internal/modules/issue/app/cascade.go server/internal/modules/issue/app/cascade_test.go server/internal/modules/issue/module.go server/internal/modules/project/adapter/http/state_writes_test.go server/internal/modules/project/adapter/postgres/failures_test.go server/internal/modules/project/adapter/postgres/label_writes_test.go server/internal/modules/project/adapter/postgres/labels.go server/internal/modules/project/adapter/postgres/queries/labels.sql server/internal/modules/project/app/clock_test.go server/internal/modules/project/app/delete_label.go server/internal/modules/project/app/delete_label_test.go server/internal/modules/project/app/delete_state.go server/internal/modules/project/app/delete_state_test.go server/internal/modules/project/app/fakes_issue_test.go server/internal/modules/project/app/fakes_label_test.go server/internal/modules/project/app/issue_ports.go server/internal/modules/project/app/label_ports.go server/internal/modules/project/domain/errors.go server/internal/modules/project/module.go web/apps/web/core/lib/error-messages.ts web/packages/i18n/src/locales/en/errors.json web/packages/i18n/src/locales/zh-CN/errors.json api/dist/openapi.yaml server/internal/modules/issue/adapter/postgres/gen/cascade.sql.go server/internal/modules/project/adapter/postgres/gen/labels.sql.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M4/P1): a state a work item is in cannot be deleted, and a deleted label leaves the work items

Deleting a state asks the issue module, under the project's lock,
whether a work item is in it, an archived one too: one is 409
project.state_in_use, checked before the default state and the group,
so the answer says what to do first. Deleting a label answers the ids
it deleted, the label and those under it, and the issue module takes
them off every work item at the deletion's moment and by its caller.
The contract and the web's messages carry the new code.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p1a.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试；结构是迁移的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是关键词守卫；静态是 `make lint-web` 的文案和键的核对）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T11.1` | 删除状态不问工作项：有工作项在里面的状态被删除 | `internal/modules/project/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAStateAWorkItemIsInIsRefused`（`make test`）、`internal/bootstrap` 的 `TestDeletingAStateAndCreatingAWorkItemInItSerialize`（`make test`）（Task 13 起） | 单元；组合 |
| `T11.2` | `StateInUse` 不算已归档的工作项 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAStateAWorkItemIsInIsRefused`（`make test`） | 存储；组合 |
| `T11.3` | `StateInUse` 连已删除的工作项也算 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAStateAWorkItemIsInIsRefused`（`make test`） | 存储；组合 |
| `T11.4` | 删除状态在带守卫的删除之后才问工作项：默认的 Backlog 答 `project.state_default` | `internal/modules/project/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAStateAWorkItemIsInIsRefused`（`make test`） | 单元；组合 |
| `T11.5` | 删除状态经连接池问工作项，不经它的事务 | `internal/modules/project/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（`make test`） | 单元；组合 |
| `T11.6` | 删除标签把标签留在工作项上 | `internal/modules/project/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingALabelTakesItOffTheWorkItems`（`make test`）、`internal/bootstrap` 的 `TestDeletingALabelAndCreatingAWorkItemWithItSerialize`（`make test`）（Task 13 起） | 单元；组合 |
| `T11.7` | 删除标签只从工作项上摘下这个标签，不摘它下面的标签 | `internal/modules/project/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingALabelTakesItOffTheWorkItems`（`make test`） | 单元；组合 |
| `T11.8` | 存储的 `DeleteLabel` 只回答这个标签的 `id`，不回答它下面的标签的 | `internal/modules/project/adapter/postgres` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingALabelTakesItOffTheWorkItems`（`make test`） | 存储；组合 |
| `T11.9` | 删除标签在自己的时刻从工作项上摘下标签，不是删除的时刻 | `internal/modules/project/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingALabelTakesItOffTheWorkItems`（`make test`） | 单元；组合 |
| `T11.10` | 删除标签经连接池从工作项上摘下标签，不经它的事务 | `internal/modules/project/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（`make test`） | 单元；组合 |
| `T11.11` | `DetachLabels` 保留行的最后写者 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingALabelTakesItOffTheWorkItems`（`make test`） | 存储；组合 |
| `T11.12` | `DetachLabels` 把数据库的时刻写进 `updated_at`，不是删除的时刻 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingALabelTakesItOffTheWorkItems`（`make test`） | 存储；组合 |
| `T11.13` | `DetachLabels` 把之前删除的行在自己的时刻再删一次 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`） | 存储 |
| `T11.14` | `issue` 的连带不问存储，回答每个状态都没有工作项 | `internal/bootstrap` 的 `TestDeletingAStateAWorkItemIsInIsRefused`（`make test`） | 组合 |
| `T11.15` | `issue` 的连带不经存储，什么标签也不摘 | `internal/bootstrap` 的 `TestDeletingALabelTakesItOffTheWorkItems`（`make test`） | 组合 |
| `T11.16` | `deleteState` 的契约不声明它回答的 `project.state_in_use` | `internal/modules/project/adapter/http` 的 全部测试（`make test`） | 单元 |
| `T11.17` | `project.state_in_use` 没有文案：页面会显示服务端的英文说明 | `error-messages.test.ts`（`make test-web`） | 单元 |
| `T11.18` | 中文的文案没有 `project_state_in_use` | `web/packages/i18n` 的 `check:sync`（`make lint-web`） | 守卫 |
| `T11.19` | 删除状态问工作项失败时照样往下做 | `internal/modules/project/app` 的 全部测试（`make test`） | 单元 |
| `T11.20` | 删除标签吞掉从工作项上摘下标签的失败 | `internal/modules/project/app` 的 全部测试（`make test`） | 单元 |
| `T11.21` | 存储的 `StateInUse` 读失败时回答没有工作项 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`） | 存储 |

---

### Task 12: 交错 1、2、3、8、22 和交错 4、5 的创建一支；分诊状态的矩阵行

**Files:**
- Create: `server/internal/bootstrap/issue_interleaving_deletion_test.go`、`server/internal/bootstrap/issue_interleaving_test.go`
- Modify: `server/internal/bootstrap/ending_connection_test.go`、`server/internal/bootstrap/issue_writes_test.go`、`server/internal/bootstrap/permission_matrix_issues_test.go`、`server/internal/bootstrap/project_connection_test.go`

**Interfaces:** 没有新的生产代码（spec 2.12；M4 设计 3.5、9.3）。
- 各模块的路由带自己的 `TxManager`，交错的第一个写才能让门扣住它的事务：
  - `newIssueRoute(t, pool, tx)`（`issue_writes_test.go`）；
  - `newProjectRouteOn(t, pool, tx, auth, members)`（`project_connection_test.go`，`newProjectRoute` 用它）；
  - `newWorkspaceRoute(t, pool, tx)`（`ending_connection_test.go`）。
- `issue_interleaving_test.go` 的工具，P1b 的交错照用（spec 第 5 节）：
  - `heldTx`：写的事务在它的写之后、提交之前停在门上；
  - `exchange{req, rec, released}`；
  - `racingWrite`、`createsIssue`、`named`；
  - `aliceOnWeb`、`webPath`、`bobsMembership` 和 alice 在 Web 上的四个写（归档、删除、移出 bob、把 bob 降为访客）；
  - `serialize`：第一个写停在门上时发出第二个，第二个在 `projects` 上等 Web 的行时（`pgtest.WaitForLockWaitOn`）开门；
  - `inBothOrders`：两种顺序各在自己的世界上跑；
  - 断言的帮手：`answered`、`refused`、`workItems`、`createdIssue`。
- `issue_interleaving_deletion_test.go`：`deletesAcme`、`fullBody`、`deletedWith`。

**Tests:**（组合，各两种顺序，`-count=5 -race` 没有 40P01）
- `TestTwoCreationsOnAProjectSerialize`（交错 1）：
  - 编号 1、2，第二个的 `sort_order` 在第一个之后，第二个的时刻不早于开门；
  - 两次创建都要 `NextIssueSequence` 的 `UPDATE`，所以这个交错看不出缺少显式的锁（spec 第 3 节第 14 条）。
- `TestCreatingAWorkItemAndDeletingItsProjectOrWorkspaceSerialize`（交错 2、3：Web、acme）：
  - 先建：七行随项目或工作区删除，时刻不早于开门；
  - 先删：404，没有行。
- `TestACallerRemovedOrDemotedWhileHisCreationWaitsIsRefused`（交错 4、5 的创建一支）：预检放行之后被移出答 404 `project.not_found`，被降为访客答 403。
- `TestArchivingAProjectAndCreatingAWorkItemSerialize`（交错 8）：先归档答 409；先建则建成，之后归档。
- `TestRemovingTheDefaultAssigneeAndCreatingAWorkItemSerialize`（交错 22）：先移出则没有负责人；先建则指派给他，移出不改这一行。
- `TestPermissionMatrix` 的"the triage state"一行：Triage 是矩阵经存储种下的分诊状态（接口的状态列表不含它）；能建工作项的格子答 422 `state_id`，其余的格子在判定时答 403（spec 第 3 节第 10 条）。

- [ ] **Step 1: 各模块的路由带自己的事务**

`server/internal/bootstrap/ending_connection_test.go`（修改，3 处）：

````old server/internal/bootstrap/ending_connection_test.go
// newWorkspaceRoute is the workspace module (workspace.New) on pool, as
// bootstrap wires it for the endings: access's Authorizer, identity's
// Accounts and MemberProfiles, converted, and the project module's cascade
// over issue's, each on pool.
func newWorkspaceRoute(t *testing.T, pool *pgxpool.Pool) moduleRoute {
````
````new server/internal/bootstrap/ending_connection_test.go
// newWorkspaceRoute is the workspace module (workspace.New) on pool, with
// tx its transactions, as bootstrap wires it: access's Authorizer,
// identity's Accounts and MemberProfiles, converted, and the project
// module's cascade over issue's, each on pool.
func newWorkspaceRoute(t *testing.T, pool *pgxpool.Pool, tx shared.TxManager) moduleRoute {
````
````old server/internal/bootstrap/ending_connection_test.go
	tx, auth, identityPorts, provided := postgres.NewTxManager(pool, 2*time.Second), authorizerOn(pool), identity.Provide(pool), workspace.Provide(pool)
````
````new server/internal/bootstrap/ending_connection_test.go
	auth, identityPorts, provided := authorizerOn(pool), identity.Provide(pool), workspace.Provide(pool)
````
````old server/internal/bootstrap/ending_connection_test.go
	route := newWorkspaceRoute(t, poolOfOne(t, r.pool.Config().ConnString()))
````
````new server/internal/bootstrap/ending_connection_test.go
	one := poolOfOne(t, r.pool.Config().ConnString())
	route := newWorkspaceRoute(t, one, postgres.NewTxManager(one, 2*time.Second))
````

`server/internal/bootstrap/issue_writes_test.go`（修改，4 处）：

````old server/internal/bootstrap/issue_writes_test.go
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
````
````new server/internal/bootstrap/issue_writes_test.go
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
````
````old server/internal/bootstrap/issue_writes_test.go
// newIssueRoute is the issue module (issue.New) on pool, with the
// Authorizer on pool and project's and identity's ports converted, all as
// bootstrap wires them.
func newIssueRoute(t *testing.T, pool *pgxpool.Pool) moduleRoute {
````
````new server/internal/bootstrap/issue_writes_test.go
// newIssueRoute is the issue module (issue.New) on pool, with tx its
// transactions, the Authorizer on pool and project's and identity's ports
// converted, all as bootstrap wires them.
func newIssueRoute(t *testing.T, pool *pgxpool.Pool, tx shared.TxManager) moduleRoute {
````
````old server/internal/bootstrap/issue_writes_test.go
	return newRoute(t, issue.New(issue.Deps{Pool: pool, Tx: postgres.NewTxManager(pool, 2*time.Second), Clock: clock.System{},
		Authorizer: authorizerOn(pool), Projects: issueProjects{projects: project.Provide(pool).IssueProjects},
		Profiles: issueProfiles{profiles: identity.Provide(pool).PublicProfiles}}).Register)
````
````new server/internal/bootstrap/issue_writes_test.go
	return newRoute(t, issue.New(issue.Deps{Pool: pool, Tx: tx, Clock: clock.System{}, Authorizer: authorizerOn(pool),
		Projects: issueProjects{projects: project.Provide(pool).IssueProjects}, Profiles: issueProfiles{profiles: identity.Provide(pool).PublicProfiles}}).Register)
````
````old server/internal/bootstrap/issue_writes_test.go
	route := newIssueRoute(t, one)
````
````new server/internal/bootstrap/issue_writes_test.go
	route := newIssueRoute(t, one, postgres.NewTxManager(one, 2*time.Second))
````

`server/internal/bootstrap/project_connection_test.go`（修改，1 处）：

````old server/internal/bootstrap/project_connection_test.go
	return newRoute(t, project.New(project.Deps{Pool: pool, Tx: postgres.NewTxManager(pool, 2*time.Second), Clock: clock.System{}, Authorizer: auth,
````
````new server/internal/bootstrap/project_connection_test.go
	return newProjectRouteOn(t, pool, postgres.NewTxManager(pool, 2*time.Second), auth, members)
}

// newProjectRouteOn is newProjectRoute with tx its transactions.
func newProjectRouteOn(t *testing.T, pool *pgxpool.Pool, tx shared.TxManager, auth shared.Authorizer,
	members projectapp.WorkspaceMembers) moduleRoute {
	t.Helper()
	return newRoute(t, project.New(project.Deps{Pool: pool, Tx: tx, Clock: clock.System{}, Authorizer: auth,
````

- [ ] **Step 2: 交错的工具和交错 1、4、5、8、22**

`server/internal/bootstrap/issue_interleaving_test.go`（新文件，294 行）：

````file server/internal/bootstrap/issue_interleaving_test.go
package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The interleavings of a work item's creation (M4 design 3.5, 9.3), on
// issueWorld's real database: each write through its module's route, as
// bootstrap wires it. The first write's transaction is held open past its
// writes (heldTx); the second is sent once it is held, and the gate opens
// once the second waits for Web's row of projects, which each of the two
// takes FOR NO KEY UPDATE (M4 design 3.5, M3 design 3.6). Every wait has a
// deadline.

// heldTx holds each transaction it runs open past its writes: once fn has
// done them, before the commit, it waits at its gate.
type heldTx struct {
	shared.TxManager
	gate *gate
}

func (h heldTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return h.TxManager.WithinTx(ctx, func(ctx context.Context) error {
		if err := fn(ctx); err != nil {
			return err
		}
		return h.gate.wait(ctx)
	})
}

// exchange is a request through a route and its answer; released, for the
// first of two writes (serialize), is the moment its gate let its
// transaction go on to its commit.
type exchange struct {
	req      *http.Request
	rec      *httptest.ResponseRecorder
	released time.Time
}

// racingWrite is a write on Web of the world through its module's route,
// built on the world's pool with tx its transactions: the sending of it
// into e, for run.
type racingWrite func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error

// sending is the sending of method path with body as the account name
// through route, into e, for run: it touches no test (moduleRoute.send).
func (w issueWorld) sending(e *exchange, route moduleRoute, name, method, path, body string) func() error {
	return func() error {
		e.req, e.rec = route.send(method, path, w.ids[name], body)
		return nil
	}
}

// createsIssue is name's creation of a work item in Web with the body body
// gives.
func createsIssue(name string, body func(t *testing.T, w issueWorld) string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newIssueRoute(t, w.pool, tx), name, http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", body(t, w))
	}
}

// named is the body of a work item named name, with no other field.
func named(name string) func(*testing.T, issueWorld) string {
	return func(*testing.T, issueWorld) string { return `{"name":"` + name + `"}` }
}

// aliceOnWeb is alice's write method path, with body, through the
// project module's route.
func aliceOnWeb(method string, path func(t *testing.T, w issueWorld) string, body string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		route := newProjectRouteOn(t, w.pool, tx, authorizerOn(w.pool), workspace.Provide(w.pool).WorkspaceMembers)
		return w.sending(e, route, "alice", method, path(t, w), body)
	}
}

// webPath is the path of Web, then rest.
func webPath(rest string) func(*testing.T, issueWorld) string {
	return func(_ *testing.T, w issueWorld) string { return "/api/v0/projects/" + w.web.String() + rest }
}

// bobsMembership is the path of bob's membership of Web.
func bobsMembership(t *testing.T, w issueWorld) string {
	return "/api/v0/project-members/" + projectMemberships(t, w.pool, w.ids["bob"], w.web)[0].String()
}

// alice's writes on Web: its archiving, its deletion, bob's removal from
// it and his demotion to a guest of it.
var (
	archivesWeb = aliceOnWeb(http.MethodPost, webPath("/archive"), "")
	deletesWeb  = aliceOnWeb(http.MethodDelete, webPath(""), "")
	removesBob  = aliceOnWeb(http.MethodDelete, bobsMembership, "")
	demotesBob  = aliceOnWeb(http.MethodPatch, bobsMembership, `{"role":5}`)
)

// serialize sends first, through a route whose transaction a gate holds
// open past its writes, then, once it is held, second; once the second
// waits for a row of projects, Web's, it opens the gate, at a's released.
// Each answers into its exchange within 10 seconds.
func (w issueWorld) serialize(t *testing.T, first, second racingWrite, a, b *exchange) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	g := newGate()
	firstWrite, secondWrite := first(t, w, heldTx{postgres.NewTxManager(w.pool, 2*time.Second), g}, a), second(t, w,
		postgres.NewTxManager(w.pool, 2*time.Second), b)
	done := run(firstWrite)
	held(t, ctx, g, done, "the first write")
	waiting := run(secondWrite)
	pgtest.WaitForLockWaitOn(t, w.pool, "projects", 5*time.Second)
	opened := time.Now().Truncate(time.Microsecond)
	close(g.open)
	if err := errors.Join(result(t, ctx, done, "the first write"), result(t, ctx, waiting, "the second write")); err != nil {
		t.Fatal(err)
	}
	a.released = opened
}

// inBothOrders runs a, then b, and b, then a, each on a world of its own
// that prepare, when given, prepares first (serialize), and checks each
// order's answers with check. aName and bName name the writes in the
// subtests' names.
func inBothOrders(t *testing.T, prepare func(t *testing.T, w issueWorld), aName string, a racingWrite, bName string, b racingWrite,
	check func(t *testing.T, w issueWorld, aFirst bool, ea, eb exchange)) {
	for _, aFirst := range []bool{true, false} {
		t.Run(map[bool]string{true: aName, false: bName}[aFirst]+" first", func(t *testing.T) {
			w := newIssueWorld(t)
			if prepare != nil {
				prepare(t, w)
			}
			var ea, eb exchange
			if aFirst {
				w.serialize(t, a, b, &ea, &eb)
			} else {
				w.serialize(t, b, a, &eb, &ea)
			}
			check(t, w, aFirst, ea, eb)
		})
	}
}

// answered checks e's answer against the contract and that it is want,
// and returns its body.
func (w issueWorld) answered(t *testing.T, e exchange, want int) string {
	t.Helper()
	w.contract.CheckResponse(t, e.req, e.rec.Result())
	if e.rec.Code != want {
		t.Errorf("%s %s = %d %s, want %d", e.req.Method, e.req.URL.Path, e.rec.Code, e.rec.Body, want)
	}
	return e.rec.Body.String()
}

// refused checks that e answered want with the problem code.
func (w issueWorld) refused(t *testing.T, e exchange, want int, code string) {
	t.Helper()
	if answer := w.answered(t, e, want); refusalOf(t, []byte(answer), false) != code {
		t.Errorf("%s %s = %s; want %s", e.req.Method, e.req.URL.Path, answer, code)
	}
}

// workItems counts Web's rows of issues, the deleted ones too.
func (w issueWorld) workItems(t *testing.T) int {
	t.Helper()
	var n int
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM issues WHERE project_id = $1", w.web).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// createdIssue is what the interleavings read of a work item created.
type createdIssue struct {
	SequenceID  int         `json:"sequence_id"`
	SortOrder   float64     `json:"sort_order"`
	AssigneeIDs []uuid.UUID `json:"assignee_ids"`
	CreatedAt   time.Time   `json:"created_at"`
}

// Two creations on Web serialize on its row (M4 design 3.5, 3.7;
// interleaving 1): alice's holds Web FOR NO KEY UPDATE and its rows past
// its writes; bob's, sent then, waits for Web's row, and once hers has
// committed takes the next number, 2, after her 1, and the sort order
// after hers in Backlog, at a moment no earlier than the gate's opening:
// he read the number, the greatest sort order and the clock under the
// lock he took once hers had committed.
func TestTwoCreationsOnAProjectSerialize(t *testing.T) {
	w := newIssueWorld(t)
	var hers, his exchange
	w.serialize(t, createsIssue("alice", named("First")), createsIssue("bob", named("Second")), &hers, &his)
	var first, second createdIssue
	decodeAnswer(t, w.answered(t, hers, http.StatusCreated), &first)
	decodeAnswer(t, w.answered(t, his, http.StatusCreated), &second)
	if first.SequenceID != 1 || second.SequenceID != 2 || second.SortOrder <= first.SortOrder || second.CreatedAt.Before(hers.released) {
		t.Errorf("created %+v, then %+v; want 1, then 2 after it in sort order, made no earlier than %v", first, second, hers.released)
	}
}

// Archiving Web and creating a work item in it serialize on Web's row (M4
// design 3.8; interleaving 8), in both orders: archived first, the
// creation, which passed the precheck, finds Web archived under its lock:
// 409 project.archived, and no work item; created first, the archiving
// waits for it and archives Web with the work item in it.
func TestArchivingAProjectAndCreatingAWorkItemSerialize(t *testing.T) {
	inBothOrders(t, nil, "archived", archivesWeb, "created", createsIssue("bob", named("Fix")), func(t *testing.T, w issueWorld,
		archivedFirst bool, archive, create exchange) {
		w.answered(t, archive, http.StatusOK)
		want := 1
		if archivedFirst {
			w.refused(t, create, http.StatusConflict, "project.archived")
			want = 0
		} else {
			w.answered(t, create, http.StatusCreated)
		}
		if n := w.workItems(t); n != want {
			t.Errorf("Web has %d work items, want %d", n, want)
		}
	})
}

// bobDefaultAssignee makes bob Web's default assignee, through the API.
func bobDefaultAssignee(t *testing.T, w issueWorld) {
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"default_assignee_id":"`+w.ids["bob"].String()+`"}`,
		http.StatusOK)
}

// Removing Web's default assignee and creating a work item without
// assignees serialize on Web's row (M4 design 3.7; interleaving 22), in
// both orders: bob, the default assignee, removed first, the creation reads
// him no member under its lock and assigns no one; created first, he is
// assigned, and his removal, which waited, leaves that row as it was.
func TestRemovingTheDefaultAssigneeAndCreatingAWorkItemSerialize(t *testing.T) {
	inBothOrders(t, bobDefaultAssignee, "removed", removesBob, "created", createsIssue("alice", named("Fix")), func(t *testing.T,
		w issueWorld, removedFirst bool, removal, create exchange) {
		w.answered(t, removal, http.StatusNoContent)
		var created createdIssue
		decodeAnswer(t, w.answered(t, create, http.StatusCreated), &created)
		want := []uuid.UUID{w.ids["bob"]}
		if removedFirst {
			want = nil
		}
		var kept int
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM issue_assignees WHERE assignee_id = $1 AND deleted_at IS NULL",
			w.ids["bob"]).Scan(&kept); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(created.AssigneeIDs, want) || kept != len(want) {
			t.Errorf("the work item assigned to %v, %d of bob's rows kept; want %v, each kept", created.AssigneeIDs, kept, want)
		}
	})
}

// The decision under the lock outlives the precheck (M4 design 3.5;
// interleavings 4 and 5, the creation's branch): bob, a member of Web,
// made secret, creates a work item while alice's removal of him, or her
// demotion of him to a guest, holds Web past its writes. His creation
// passed the precheck, which read him a member, and waits for Web's row;
// once hers commits, the decision under the lock reads him removed, who
// no longer sees Web: 404 project.not_found; or a guest, who creates no
// work item: 403 forbidden. No work item.
func TestACallerRemovedOrDemotedWhileHisCreationWaitsIsRefused(t *testing.T) {
	for _, tt := range []struct {
		name    string
		change  racingWrite
		changed int
		want    int
		code    string
	}{
		{"removed", removesBob, http.StatusNoContent, http.StatusNotFound, "project.not_found"},
		{"made a guest", demotesBob, http.StatusOK, http.StatusForbidden, "forbidden"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newIssueWorld(t)
			w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"network":0}`, http.StatusOK)
			var change, create exchange
			w.serialize(t, tt.change, createsIssue("bob", named("Fix")), &change, &create)
			w.answered(t, change, tt.changed)
			w.refused(t, create, tt.want, tt.code)
			if n := w.workItems(t); n != 0 {
				t.Errorf("Web has %d work items, want none", n)
			}
		})
	}
}
````

- [ ] **Step 3: 交错 2、3**

`server/internal/bootstrap/issue_interleaving_deletion_test.go`（新文件，85 行）：

````file server/internal/bootstrap/issue_interleaving_deletion_test.go
package bootstrap

import (
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// deletesAcme is alice's deletion of acme through the workspace module's
// route.
func deletesAcme(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
	return w.sending(e, newWorkspaceRoute(t, w.pool, tx), "alice", http.MethodDelete, "/api/v0/workspaces/acme", "")
}

// fullBody is the world's fullIssue, as createsIssue takes a body.
func fullBody(_ *testing.T, w issueWorld) string {
	return w.fullIssue()
}

// deletedWith checks the deletion of the row id of table, the deleted
// project's or workspace's: at a moment no earlier than since, with the
// rows of Web's work-item side, n of them, each deleted and last written
// at that moment, by alice, and not before it was made.
func (w issueWorld) deletedWith(t *testing.T, table string, id uuid.UUID, since time.Time, n int) {
	t.Helper()
	var at time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at FROM "+table+" WHERE id = $1", id).Scan(&at); err != nil || at.Before(since) {
		t.Errorf("the row of %s deleted at %v (%v); want no earlier than %v", table, at, err, since)
	}
	var all, hers int
	if err := w.pool.QueryRow(pgtest.Soon(t), `
		SELECT count(*), count(*) FILTER (WHERE r.deleted_at = d.at AND r.updated_at = d.at AND r.updated_by_id = $3 AND r.created_at <= d.at)
		FROM (SELECT deleted_at, updated_at, updated_by_id, created_at FROM issues WHERE project_id = $2
			UNION ALL SELECT deleted_at, updated_at, updated_by_id, created_at FROM issue_assignees WHERE project_id = $2
			UNION ALL SELECT deleted_at, updated_at, updated_by_id, created_at FROM issue_labels WHERE project_id = $2
			UNION ALL SELECT deleted_at, updated_at, updated_by_id, created_at FROM issue_subscribers WHERE project_id = $2
			UNION ALL SELECT deleted_at, updated_at, updated_by_id, created_at FROM issue_activities WHERE project_id = $2) r,
			(SELECT deleted_at AS at FROM `+table+` WHERE id = $1) d`, id, w.web, w.ids["alice"]).Scan(&all, &hers); err != nil {
		t.Fatal(err)
	}
	if all != n || hers != n {
		t.Errorf("Web's work-item side has %d rows, %d of them deleted with the row of %s; want %d, each deleted with it", all, hers,
			table, n)
	}
}

// A work item's creation and its project's deletion, or its workspace's,
// serialize on Web's row (M4 design 3.5, 3.6; interleavings 2 and 3), in
// both orders. Created first, holding Web FOR NO KEY UPDATE, assigned to
// bob and labelled Bug (fullIssue): the deletion waits for Web's row, then
// deletes the work item's seven rows with Web, or with acme, by alice, at
// its one moment, no earlier than the creation's commit: it read its clock
// under the locks it took once the creation had committed (M4 design 3.5).
// Deleted first: the creation, which passed the precheck, finds Web
// deleted under its lock: 404 project.not_found, and no work item.
func TestCreatingAWorkItemAndDeletingItsProjectOrWorkspaceSerialize(t *testing.T) {
	for _, deletion := range []struct {
		name  string
		write racingWrite
		table string                       // the deleted row's
		id    func(w issueWorld) uuid.UUID // its id
	}{
		{"Web", deletesWeb, "projects", func(w issueWorld) uuid.UUID { return w.web }},
		{"acme", deletesAcme, "workspaces", func(w issueWorld) uuid.UUID { return w.acme }},
	} {
		t.Run(deletion.name, func(t *testing.T) {
			inBothOrders(t, nil, "deleted", deletion.write, "created", createsIssue("alice", fullBody), func(t *testing.T,
				w issueWorld, deletedFirst bool, deleted, create exchange) {
				w.answered(t, deleted, http.StatusNoContent)
				rows := 7
				if deletedFirst {
					w.refused(t, create, http.StatusNotFound, "project.not_found")
					rows = 0
				} else {
					w.answered(t, create, http.StatusCreated)
				}
				w.deletedWith(t, deletion.table, deletion.id(w), create.released, rows)
			})
		})
	}
}
````

- [ ] **Step 4: 分诊状态的矩阵行**

`server/internal/bootstrap/permission_matrix_issues_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_issues_test.go
			refusal: "label_ids[0] not_allowed"},
````
````new server/internal/bootstrap/permission_matrix_issues_test.go
			refusal: "label_ids[0] not_allowed"},
		// The project's triage state, which the matrix seeds through the
		// project store, as createProject makes it, and which the API's lists
		// of states leave out (M4 design 9.3, M17): it takes no work item
		// before the intake (M7), checked after the decision too.
		{op: "createIssue", variant: "the triage state", write: true, columns: projectColumns,
			request: withIssueBody(func(c caller, s seeded) string {
				return `{"name":"Fix","state_id":"` + s.state(projectOf(c), "Triage").String() + `"}`
			}),
			cells:   ofProject(cellValidationFailed, cellValidationFailed, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden),
			refusal: "state_id not_allowed"},
````

- [ ] **Step 5: 测试和 lint**

Run: `go -C server test -count=5 -race -run 'TestTwoCreationsOnAProjectSerialize$|TestCreatingAWorkItemAndDeletingItsProjectOrWorkspaceSerialize$|TestACallerRemovedOrDemotedWhileHisCreationWaitsIsRefused$|TestArchivingAProjectAndCreatingAWorkItemSerialize$|TestRemovingTheDefaultAssigneeAndCreatingAWorkItemSerialize$' ./internal/bootstrap/`
Expected: `ok`；输出中没有 `40P01`、`deadlock detected`、`DATA RACE`。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 6: 提交**

```bash
git add server/internal/bootstrap/ending_connection_test.go server/internal/bootstrap/issue_interleaving_deletion_test.go server/internal/bootstrap/issue_interleaving_test.go server/internal/bootstrap/issue_writes_test.go server/internal/bootstrap/permission_matrix_issues_test.go server/internal/bootstrap/project_connection_test.go
```
```bash
git commit -m "test(M4/P1): a work item's creation serializes with the writes on its project

On a real database, each write through its module's route, the first
held open past its writes until the second waits for the project's
row: two creations take numbers 1 and 2 and places one after the other;
a creation and the deletion of its project or workspace, either order,
leave its rows deleted with them or answer 404; a caller removed or
made a guest while his creation waited is refused under the lock;
archiving and creating, and removing the default assignee and
creating, serialize either way. The matrix's triage state takes no
work item.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p1a.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试；结构是迁移的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是关键词守卫；静态是 `make lint-web` 的文案和键的核对）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T12.1` | `issue` 模块锁项目只是普通的读：判定和检查在取编号的 `UPDATE` 锁住项目之前进行 | `internal/bootstrap` 的 `TestArchivingAProjectAndCreatingAWorkItemSerialize`（`make test`）、`internal/bootstrap` 的 `TestACallerRemovedOrDemotedWhileHisCreationWaitsIsRefused`（`make test`）、`internal/bootstrap` 的 `TestEachIssueWriteLocksItsProjectFirst`（`make test`） | 组合 |
| `T12.2` | 交错的门在第一个写写入之前就扣住它，那时它还没有锁 Web | `internal/bootstrap` 的 `TestTwoCreationsOnAProjectSerialize`（`make test`） | 组合 |
| `T12.3` | 交错的探测等的是 `workspaces` 上的等待，第二个创建从不在那里等 | `internal/bootstrap` 的 `TestTwoCreationsOnAProjectSerialize`（`make test`） | 组合 |
| `T12.4` | 删除项目在自己的时刻删除工作项一侧，不是项目的时刻 | `internal/modules/project/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestCreatingAWorkItemAndDeletingItsProjectOrWorkspaceSerialize`（`make test`） | 单元；组合 |

---

### Task 13: 交错 6、7 的创建一支

**Files:**
- Create: `server/internal/bootstrap/issue_interleaving_catalog_test.go`

**Interfaces:** 没有新的生产代码（spec 2.12；M4 设计 3.6、3.7、9.3）。`issue_interleaving_catalog_test.go` 照 Task 12 的工具写：`addsReview`、`inReview`、`labelledBug`，alice 删除 Review、Bug 的两个写。

**Tests:**（组合，两种顺序，`-count=5 -race` 没有 40P01）
- `TestDeletingAStateAndCreatingAWorkItemInItSerialize`（交错 6 的创建一支）：先删 Review，创建在锁下找不到它，答 422 `state_id`，没有工作项；先建，删除在锁下看到 Review 里的工作项，答 409 `project.state_in_use`，Review 保留。
- `TestDeletingALabelAndCreatingAWorkItemWithItSerialize`（交错 7 的创建一支）：先删 Bug，创建在锁下找不到它，答 422 `label_ids[0]`，没有工作项；先建，删除在它的时刻、由 alice 把 Bug 从工作项上摘下。

- [ ] **Step 1: 交错 6、7**

`server/internal/bootstrap/issue_interleaving_catalog_test.go`（新文件，85 行）：

````file server/internal/bootstrap/issue_interleaving_catalog_test.go
package bootstrap

import (
	"net/http"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// addsReview adds Review to Web's started group, through the API.
func addsReview(t *testing.T, w issueWorld) {
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/states", `{"name":"Review","color":"#F59E0B","group":"started"}`,
		http.StatusCreated)
}

// inReview is the body of a work item in Web's Review.
func inReview(t *testing.T, w issueWorld) string {
	return `{"name":"Fix","state_id":"` + stateID(t, w.pool, w.web, "Review").String() + `"}`
}

// labelledBug is the body of a work item labelled Web's Bug.
func labelledBug(_ *testing.T, w issueWorld) string {
	return `{"name":"Fix","label_ids":["` + w.bug.String() + `"]}`
}

// alice's deletions of Web's Review and of its Bug.
var (
	deletesReview = aliceOnWeb(http.MethodDelete, func(t *testing.T, w issueWorld) string {
		return "/api/v0/states/" + stateID(t, w.pool, w.web, "Review").String()
	}, "")
	deletesBug = aliceOnWeb(http.MethodDelete, func(_ *testing.T, w issueWorld) string { return "/api/v0/labels/" + w.bug.String() }, "")
)

// Deleting a state and creating a work item in it serialize on Web's row
// (M4 design 3.6, 3.7; interleaving 6, the creation's branch), in both
// orders: Review deleted first, the creation finds no such state under its
// lock: 422 state_id, and no work item; created first, the deletion finds
// the work item in Review under its lock: 409 project.state_in_use, and
// Review is kept.
func TestDeletingAStateAndCreatingAWorkItemInItSerialize(t *testing.T) {
	inBothOrders(t, addsReview, "deleted", deletesReview, "created", createsIssue("bob", inReview), func(t *testing.T, w issueWorld,
		deletedFirst bool, deletion, create exchange) {
		if deletedFirst {
			w.answered(t, deletion, http.StatusNoContent)
			w.refused(t, create, http.StatusUnprocessableEntity, "validation_failed state_id not_allowed")
			if n := w.workItems(t); n != 0 {
				t.Errorf("Web has %d work items, want none", n)
			}
			return
		}
		w.answered(t, create, http.StatusCreated)
		w.refused(t, deletion, http.StatusConflict, "project.state_in_use")
		var kept bool
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at IS NULL FROM states WHERE project_id = $1 AND name = 'Review'",
			w.web).Scan(&kept); err != nil || !kept {
			t.Errorf("Review kept: %v, %v; want kept", kept, err)
		}
	})
}

// Deleting a label and creating a work item labelled with it serialize on
// Web's row (M4 design 3.6, 3.7; interleaving 7, the creation's branch),
// in both orders: Bug deleted first, the creation finds no such label
// under its lock: 422 label_ids[0], and no work item; created first, the
// deletion takes Bug off the work item at its moment, by alice.
func TestDeletingALabelAndCreatingAWorkItemWithItSerialize(t *testing.T) {
	inBothOrders(t, nil, "deleted", deletesBug, "created", createsIssue("bob", labelledBug), func(t *testing.T, w issueWorld,
		deletedFirst bool, deletion, create exchange) {
		w.answered(t, deletion, http.StatusNoContent)
		if deletedFirst {
			w.refused(t, create, http.StatusUnprocessableEntity, "validation_failed label_ids[0] not_allowed")
			if n := w.workItems(t); n != 0 {
				t.Errorf("Web has %d work items, want none", n)
			}
			return
		}
		w.answered(t, create, http.StatusCreated)
		var takenOff bool
		if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT il.deleted_at = l.deleted_at AND il.updated_at = l.deleted_at AND il.updated_by_id = $2
			FROM issue_labels il JOIN labels l ON l.id = il.label_id WHERE l.id = $1`, w.bug, w.ids["alice"]).Scan(&takenOff); err != nil ||
			!takenOff {
			t.Errorf("Bug taken off the work item with its deletion, by alice: %v, %v; want it so", takenOff, err)
		}
	})
}
````

- [ ] **Step 2: 测试和 lint**

Run: `go -C server test -count=5 -race -run 'TestDeletingAStateAndCreatingAWorkItemInItSerialize$|TestDeletingALabelAndCreatingAWorkItemWithItSerialize$' ./internal/bootstrap/`
Expected: `ok`；输出中没有 `40P01`、`deadlock detected`、`DATA RACE`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 3: 提交**

```bash
git add server/internal/bootstrap/issue_interleaving_catalog_test.go
```
```bash
git commit -m "test(M4/P1): deleting a state or a label serializes with creating a work item in it

Either order, on the project's row: a state or label deleted first is
not found by the creation under its lock (422), and no work item is
made; a work item created first keeps its state from being deleted
(409 project.state_in_use), or loses the label at the deletion's moment.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p1a.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试；结构是迁移的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是关键词守卫；静态是 `make lint-web` 的文案和键的核对）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T13.1` | 标签的目录回答已删除的标签：等锁期间被删除的标签被挂到工作项上 | `internal/modules/project/adapter/postgres` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingALabelAndCreatingAWorkItemWithItSerialize`（`make test`） | 存储；组合 |

---

### Task 14: 端到端：工作项的夹具、I1 的接口版本，M3 的删除故事带着工作项一侧的行；过渡结束

**Files:**
- Create: `e2e/fixtures/assert/catalog.ts`、`e2e/fixtures/assert/issue.ts`、`e2e/fixtures/furnish.ts`、`e2e/fixtures/issues.ts`、`e2e/stories/issue/i1-create-issue.spec.ts`、`e2e/stories/workspace/w3-workspace-settings-page.spec.ts`
- Modify: `e2e/fixtures/api.ts`、`e2e/fixtures/assert/project.ts`、`e2e/fixtures/assert/workspace.ts`、`e2e/stories/project/p4-archive.spec.ts`、`e2e/stories/workspace/w11-guest-bounds.spec.ts`、`e2e/stories/workspace/w2-landing.spec.ts`、`e2e/stories/workspace/w3-workspace-settings.spec.ts`

**Interfaces:**（spec 2.13；M4 设计第 2 节 I1、P4、W2、W3，3.6）
- `e2e/fixtures/issues.ts`：
  - 类型 `Issue`、`IssueCreate` 来自 `schema.gen.ts`；
  - `issueCreated`：回答状态和工作项，或问题的码和字段；
  - `createIssue`：要求 201。
- `e2e/fixtures/furnish.ts`：
  - `furnishIssueRows(api, token, projectId, assigneeId, labelId)`：一个工作项，指派一人、带一个标签，五张表各至少一行（spec 第 3 节第 12 条）；
  - `furnishWorkspace`：从 `api.ts` 移来，另给 Web 一个标签 Bug 和这样一个工作项。
- `e2e/fixtures/assert/issue.ts` 的 `expectIssueCreated`：
  - 工作项的行；编号等于项目的 `last_issue_sequence`；负责人、标签、关注、动态；
  - 每一行的工作区、项目、写者、执行者都是创建的，同一时刻。
- `e2e/fixtures/assert/catalog.ts`（新）：`tablesUnder(db, parent, column)` 从目录读出外键指向 `parent` 的表，并核对它就是有 `column` 列的每张表。`projectTables`、`workspaceTables` 是它的两个用法，不写两份。
- `assert/project.ts`：`expectProjectDeleted` 读 `projectTables`，过渡的 `withoutRows` 删去（裁定 7 的过渡在这里结束）。
- `assert/workspace.ts`：手写的清单 `workspaceTables`、`deletedAloneTables` 删去；`expectWorkspaceDeleted` 读 `workspaceTables`，它的 `deletedAlone`（删除之前已删除的行，如接受时删除的邀请）由故事给出。

**Tests:**（端到端）
- I1（接口版本，新）：
  - ann 在 In Progress 建一个工作项，优先级 high，指派 bob 和管理员，标签 UI、Bug：编号 1，排在它的状态第一；负责人、标签按 `id`；创建者和负责人关注；创建和每个负责人的动态；同一时刻（`expectIssueCreated`）；
  - ann 是默认负责人时，bob 建的下一个在 Backlog、指派给 ann；ann 离开之后再建：没有负责人，排在前一个之后；
  - 在 Cancelled（"已取消"组）建的不算完成；
  - 同一个账户、同一个标签给两次和大小写不对的优先级（`High`：契约的值是小写的，体的形状不核对枚举的值），422，三个问题一次报出；Web 没有的状态、Web 没有的标签和指派访客，422，三个问题一次报出；
  - 访客建 403，已归档的项目 409。
- P4（接口版本）：删除之前暂时恢复，成员建一个工作项，再归档；删除时工作项一侧的行随项目在同一时刻删除。页面版本：wes 被指派。
- W2：`furnishWorkspace` 带着工作项。
- W3：
  - 拆成两个文件（spec 第 3 节第 12 条）：接口版本 `w3-workspace-settings.spec.ts`（整个文件）；页面版本 `w3-workspace-settings-page.spec.ts`，两个页面测试原样移过去；
  - 接口版本里 Acme 的 Web、Old 各一个工作项，Other 的 Web 没有。
- W11：表的清单从目录读。
- `make e2e`：121 个（此前的 120 个，加 I1）。

- [ ] **Step 1: 工作项的夹具，`furnishWorkspace` 移到 `furnish.ts`**

`e2e/fixtures/api.ts`（修改，1 处）：

````old e2e/fixtures/api.ts

/**
 * Gives the workspace of slug, as its admin of adminToken, a row of each table its deletion writes
 * (expectWorkspaceDeleted): the membership of member, by an invitation he accepts (deleted as he accepts it); a pending
 * invitation of inviteeEmail; the admin's display settings; and the project Web, with its own rows, its label Bug
 * among them.
 */
export async function furnishWorkspace(
  api: Api,
  adminToken: string,
  slug: string,
  member: { email: string; token: string },
  inviteeEmail: string
): Promise<void> {
  await inviteAndAccept(api, adminToken, slug, member, 15);
  await invite(api, adminToken, slug, [{ email: inviteeEmail, role: 15 }]);
  await changeWorkspacePreferences(api, adminToken, slug, { navigation_project_limit: 3 });
  const web = await createProject(api, adminToken, slug, { name: "Web", identifier: "WEB" });
  await createLabel(api, adminToken, web.id, { name: "Bug" });
}

````
````new e2e/fixtures/api.ts

````

`e2e/fixtures/furnish.ts`（新文件，47 行）：

````file e2e/fixtures/furnish.ts
import { changeWorkspacePreferences, createLabel, createProject, invite, inviteAndAccept, type Api } from "./api";
import { accountId } from "./auth";
import { createIssue, type Issue } from "./issues";

// The rows a deletion story furnishes a workspace or a project with first, so that its deletion has a row of each
// table to write (expectWorkspaceDeleted, expectProjectDeleted).

/**
 * Gives the project of projectId a row of each of the work items' tables: a work item that the caller of token, a
 * member of it, creates, assigned to the account of assigneeId, a member of it who is not its guest, and labelled the
 * label of labelId, one of its labels; the two accounts its subscribers, and its activity the creation and the
 * assignee. Returns it.
 */
export async function furnishIssueRows(
  api: Api,
  token: string,
  projectId: string,
  assigneeId: string,
  labelId: string
): Promise<Issue> {
  return createIssue(api, token, projectId, {
    name: "Fix the login",
    assignee_ids: [assigneeId],
    label_ids: [labelId],
  });
}

/**
 * Gives the workspace of slug, as its admin of adminToken, a row of each table its deletion writes
 * (expectWorkspaceDeleted): the membership of member, by an invitation he accepts (deleted as he accepts it); a pending
 * invitation of inviteeEmail; the admin's display settings; and the project Web, with its own rows: its label Bug
 * among them, and a work item assigned to the admin and labelled Bug, with its own (furnishIssueRows).
 */
export async function furnishWorkspace(
  api: Api,
  adminToken: string,
  slug: string,
  member: { email: string; token: string },
  inviteeEmail: string
): Promise<void> {
  await inviteAndAccept(api, adminToken, slug, member, 15);
  await invite(api, adminToken, slug, [{ email: inviteeEmail, role: 15 }]);
  await changeWorkspacePreferences(api, adminToken, slug, { navigation_project_limit: 3 });
  const web = await createProject(api, adminToken, slug, { name: "Web", identifier: "WEB" });
  const bug = await createLabel(api, adminToken, web.id, { name: "Bug" });
  await furnishIssueRows(api, adminToken, web.id, await accountId(api, adminToken), bug.id);
}
````

`e2e/fixtures/issues.ts`（新文件，33 行）：

````file e2e/fixtures/issues.ts
import type { components } from "@nerve/api-client";
import { expect } from "@playwright/test";

import { answer, type Api } from "./api";
import { bearer } from "./auth";

// The work items' calls of the stories (M4 design 2).

export type Issue = components["schemas"]["IssueDetail"];
export type IssueCreate = components["schemas"]["IssueCreate"];

/**
 * The answer of POST /api/v0/projects/{project_id}/issues by the caller of token: its status, the work item, or the
 * problem's code and fields.
 */
export async function issueCreated(api: Api, token: string, projectId: string, body: IssueCreate) {
  const { data, error, response } = await api.POST("/api/v0/projects/{project_id}/issues", {
    params: { path: { project_id: projectId } },
    body,
    headers: bearer(token),
  });
  return data ? { status: response.status, issue: data } : answer(response, error);
}

/** Creates a work item in the project of projectId with the bearer token given, a member's of it, and returns it. */
export async function createIssue(api: Api, token: string, projectId: string, body: IssueCreate): Promise<Issue> {
  const created = await issueCreated(api, token, projectId, body);
  expect(created.status, `create the work item ${body.name} in ${projectId}: ${JSON.stringify(created)}`).toBe(201);
  if (!("issue" in created)) {
    throw new Error(`createIssue ${body.name} answered 201 without the work item`);
  }
  return created.issue;
}
````

- [ ] **Step 2: 断言：`expectIssueCreated`、目录驱动的表，过渡结束**

`e2e/fixtures/assert/catalog.ts`（新文件，38 行）：

````file e2e/fixtures/assert/catalog.ts
import { expect } from "@playwright/test";

import type { Database } from "../db";

// The tables a deletion writes, read from the database's catalog (M3 design 3.6; M4 design 3.6), so that a table a
// later phase adds is read too.

/**
 * The tables under parent: each table whose foreign key names parent. Every table with a column named column is one
 * of them, so a table that has the column without the foreign key fails here. Both lists sort their names under "C":
 * a regclass's text would sort under the database's collation and an information_schema name under "C", two orders
 * that differ (issues, issue_views).
 */
async function tablesUnder(db: Database, parent: string, column: string): Promise<string[]> {
  const tables = await db.query<{ name: string }>(
    `SELECT DISTINCT c.conrelid::regclass::text COLLATE "C" AS name FROM pg_constraint c
      WHERE c.contype = 'f' AND c.confrelid = $1::regclass ORDER BY 1`,
    [parent]
  );
  expect(tables.length, `the tables under ${parent}`).toBeGreaterThan(0);
  const withColumn = await db.query<{ name: string }>(
    `SELECT DISTINCT c.table_name::text COLLATE "C" AS name FROM information_schema.columns c
      WHERE c.table_schema = current_schema() AND c.column_name = $1 ORDER BY 1`,
    [column]
  );
  expect(tables, `the tables under ${parent}, as their ${column} columns name them`).toEqual(withColumn);
  return tables.map(({ name }) => name);
}

/** The tables under projects, whose rows a project's deletion writes: P4, W3 (expectProjectDeleted). */
export function projectTables(db: Database): Promise<string[]> {
  return tablesUnder(db, "projects", "project_id");
}

/** The tables under workspaces, whose rows a workspace's deletion writes: W2, W3 (expectWorkspaceDeleted), W11. */
export function workspaceTables(db: Database): Promise<string[]> {
  return tablesUnder(db, "workspaces", "workspace_id");
}
````

`e2e/fixtures/assert/issue.ts`（新文件，103 行）：

````file e2e/fixtures/assert/issue.ts
import { expect } from "@playwright/test";

import type { Database } from "../db";

// Database assertions of the work-item stories. The page version and the
// API version of a story call the same function (M4 design 2).

/** A row of a work item's activity as expectIssueCreated reads it: the account a row names, by its address. */
export interface ActivityRow {
  field: string | null;
  verb: string;
  old_value: string | null;
  new_value: string | null;
  /** The address of the account the row names (new_identifier), for an assignee's row; else null. */
  named: string | null;
  comment: string;
}

/**
 * A work item as created: its number, name, state (by name), priority and sort order; whether it is completed; the
 * address of the account that created it; its assignees and subscribers by address, its labels by name, each in
 * order; and its activity, in its order.
 */
export interface CreatedIssue {
  sequence_id: number;
  name: string;
  state: string;
  priority: string;
  sort_order: number;
  completed: boolean;
  by: string;
  assignees: string[];
  labels: string[];
  subscribers: string[];
  activity: ActivityRow[];
}

/**
 * I1: the work item of issueId holds want, as created (M4 design 3.7, 3.13): undeleted, not archived, numbered
 * want.sequence_id, which its project's last_issue_sequence is now. It and every row of its assignees, labels,
 * subscribers and activity are of its project and workspace, made and last written by the account of want.by, the
 * activity's actor too, at one moment.
 */
export async function expectIssueCreated(db: Database, issueId: string, want: CreatedIssue): Promise<void> {
  const [issue] = await db.query(
    `SELECT i.sequence_id, i.name, s.name AS state, i.priority, i.sort_order, i.completed_at IS NOT NULL AS completed,
            c.email AS by, p.last_issue_sequence,
            i.workspace_id = p.workspace_id AND i.updated_by_id = i.created_by_id AND i.updated_at = i.created_at
              AND i.deleted_at IS NULL AND i.archived_at IS NULL AS as_created
       FROM issues i JOIN projects p ON p.id = i.project_id JOIN states s ON s.id = i.state_id
       JOIN users c ON c.id = i.created_by_id
      WHERE i.id = $1`,
    [issueId]
  );
  const { assignees, labels, subscribers, activity, ...row } = want;
  expect(issue, `the work item ${issueId}`).toEqual({
    ...row,
    last_issue_sequence: want.sequence_id,
    as_created: true,
  });
  const named = async (sql: string) => (await db.query<{ v: string }>(sql, [issueId])).map(({ v }) => v);
  expect(
    {
      assignees: await named(`SELECT u.email AS v FROM issue_assignees r JOIN users u ON u.id = r.assignee_id
                                WHERE r.issue_id = $1 ORDER BY u.email COLLATE "C"`),
      labels: await named(`SELECT l.name AS v FROM issue_labels r JOIN labels l ON l.id = r.label_id
                             WHERE r.issue_id = $1 ORDER BY l.name COLLATE "C"`),
      subscribers: await named(`SELECT u.email AS v FROM issue_subscribers r JOIN users u ON u.id = r.subscriber_id
                                  WHERE r.issue_id = $1 ORDER BY u.email COLLATE "C"`),
      activity: await db.query(
        `SELECT a.field, a.verb, a.old_value, a.new_value, u.email AS named, a.comment
           FROM issue_activities a LEFT JOIN users u ON u.id = a.new_identifier
          WHERE a.issue_id = $1 ORDER BY a.created_at, a.id`,
        [issueId]
      ),
    },
    `the rows of the work item ${issueId}`
  ).toEqual({ assignees, labels, subscribers, activity });
  // The database compares the moments: a Date holds milliseconds, a timestamptz microseconds.
  const [stamps] = await db.query<{ rows: number; stamped: number }>(
    `SELECT count(*)::int AS rows,
            count(*) FILTER (WHERE r.workspace_id = i.workspace_id AND r.project_id = i.project_id
                               AND r.created_by_id = i.created_by_id AND r.updated_by_id = i.created_by_id
                               AND r.actor IS NOT DISTINCT FROM i.created_by_id
                               AND r.created_at = i.created_at AND r.updated_at = i.created_at AND r.deleted_at IS NULL)::int
              AS stamped
       FROM issues i, LATERAL (
              SELECT workspace_id, project_id, created_by_id, updated_by_id, created_by_id AS actor, created_at, updated_at, deleted_at
                FROM issue_assignees WHERE issue_id = i.id
              UNION ALL SELECT workspace_id, project_id, created_by_id, updated_by_id, created_by_id, created_at, updated_at, deleted_at
                FROM issue_labels WHERE issue_id = i.id
              UNION ALL SELECT workspace_id, project_id, created_by_id, updated_by_id, created_by_id, created_at, updated_at, deleted_at
                FROM issue_subscribers WHERE issue_id = i.id
              UNION ALL SELECT workspace_id, project_id, created_by_id, updated_by_id, actor_id, created_at, updated_at, deleted_at
                FROM issue_activities WHERE issue_id = i.id) r
      WHERE i.id = $1`,
    [issueId]
  );
  expect(stamps, `the rows of the work item ${issueId}, each made with it`).toEqual({
    rows: assignees.length + labels.length + subscribers.length + activity.length,
    stamped: assignees.length + labels.length + subscribers.length + activity.length,
  });
}
````

`e2e/fixtures/assert/project.ts`（修改，6 处）：

````old e2e/fixtures/assert/project.ts
import type { Database } from "../db";
````
````new e2e/fixtures/assert/project.ts
import type { Database } from "../db";
import { projectTables } from "./catalog";
````
````old e2e/fixtures/assert/project.ts
 * same account, every row under it that was not deleted before: of each table whose foreign key names projects (the
 * catalog's list, so a table a later phase adds is read too, and the list is every table with a project_id column),
 * by its project_id. Each table but the work items' has such a row, and none is left undeleted; a row deleted before
 * the project, as a label a story deleted, is not counted.
````
````new e2e/fixtures/assert/project.ts
 * same account, every row under it that was not deleted before, of each table under projects (projectTables), by its
 * project_id. Each table has such a row, and none is left undeleted; a row deleted before the project, as a label a
 * story deleted, is not counted.
````
````old e2e/fixtures/assert/project.ts
  // Both lists sort their names under "C": a regclass's text would sort under the database's collation and an
  // information_schema name under "C", two orders that differ (issues, issue_views).
  const tables = await db.query<{ name: string }>(
    `SELECT DISTINCT c.conrelid::regclass::text COLLATE "C" AS name FROM pg_constraint c
      WHERE c.contype = 'f' AND c.confrelid = 'projects'::regclass ORDER BY 1`
  );
  expect(tables.length, "the tables under projects").toBeGreaterThan(0);
  // Every table with a project_id is read: a table left out of the list above fails here.
  const withProjectId = await db.query<{ name: string }>(
    `SELECT DISTINCT c.table_name::text COLLATE "C" AS name FROM information_schema.columns c
      WHERE c.table_schema = current_schema() AND c.column_name = 'project_id' ORDER BY 1`
  );
  expect(tables, "the tables under projects, as their project_id columns name them").toEqual(withProjectId);
````
````new e2e/fixtures/assert/project.ts
  const tables = await projectTables(db);
````
````old e2e/fixtures/assert/project.ts
    tables.map(async ({ name }) => {
````
````new e2e/fixtures/assert/project.ts
    tables.map(async (name) => {
````
````old e2e/fixtures/assert/project.ts
  );
  // No story writes the work items' tables yet (M4 P1a spec, section 3): they have no row to be deleted with the
  // project, and none left undeleted either.
  const withoutRows = new Set(["issue_activities", "issue_assignees", "issue_labels", "issue_subscribers", "issues"]);
````
````new e2e/fixtures/assert/project.ts
  );
````
````old e2e/fixtures/assert/project.ts
    tables.map(({ name }) => ({ table: name, deletedWithIt: !withoutRows.has(name), other: 0 }))
````
````new e2e/fixtures/assert/project.ts
    tables.map((name) => ({ table: name, deletedWithIt: true, other: 0 }))
````

`e2e/fixtures/assert/workspace.ts`（修改，7 处）：

````old e2e/fixtures/assert/workspace.ts
import type { Database } from "../db";
````
````new e2e/fixtures/assert/workspace.ts
import type { Database } from "../db";
import { workspaceTables } from "./catalog";
````
````old e2e/fixtures/assert/workspace.ts

/** The tables whose rows belong to a workspace and are deleted with it (M3 design 3.6, 4.12). */
export const workspaceTables = [
  "workspace_members",
  "workspace_member_invites",
  "workspace_user_properties",
  "projects",
  "project_members",
  "project_user_properties",
  "states",
  "labels",
] as const;

/**
 * The tables whose rows a story can delete on their own before their workspace, which keep that moment: an
 * invitation, when it is accepted or deleted, or its address's membership ends (M3 design 3.8); a project, its
 * memberships, its members' display settings, its states and its labels, when the project is deleted (P4b, P7b).
 */
export const deletedAloneTables: (typeof workspaceTables)[number][] = [
  "workspace_member_invites",
  "projects",
  "project_members",
  "project_user_properties",
  "states",
  "labels",
];

````
````new e2e/fixtures/assert/workspace.ts

````
````old e2e/fixtures/assert/workspace.ts
 * the same account, every row under it that was not deleted before: its memberships, invitations and display
 * settings, its projects, their memberships, their members' display settings, their states and their labels. Each
 * table has such a row; none is left undeleted; each table of deletedAlone, whose rows the story deleted alone, has
 * rows deleted earlier, which kept their moment, and every row of the others carries the workspace's.
````
````new e2e/fixtures/assert/workspace.ts
 * the same account, every row under it that was not deleted before (workspaceTables): its memberships, invitations
 * and display settings, its projects, their memberships, their members' display settings, their states, their labels
 * and their work items' rows. Each table has such a row; none is left undeleted; each table of deletedAlone, whose
 * rows the story deleted alone, has rows deleted earlier, which kept their moment, and every row of the others carries
 * the workspace's.
````
````old e2e/fixtures/assert/workspace.ts
  deletedAlone: (typeof workspaceTables)[number][]
````
````new e2e/fixtures/assert/workspace.ts
  deletedAlone: string[]
````
````old e2e/fixtures/assert/workspace.ts
  expect(w?.updated_by_id, `${slug} deleted by ${adminEmail}`).toBe(w?.admin);
  // The database compares the moments: a Date holds milliseconds, a timestamptz microseconds.
````
````new e2e/fixtures/assert/workspace.ts
  expect(w?.updated_by_id, `${slug} deleted by ${adminEmail}`).toBe(w?.admin);
  // The database compares the moments: a Date holds milliseconds, a timestamptz microseconds.
  const names = await workspaceTables(db);
````
````old e2e/fixtures/assert/workspace.ts
    workspaceTables.map(async (table) => {
````
````new e2e/fixtures/assert/workspace.ts
    names.map(async (table) => {
````
````old e2e/fixtures/assert/workspace.ts
    workspaceTables.map((table) => ({
````
````new e2e/fixtures/assert/workspace.ts
    names.map((table) => ({
````

- [ ] **Step 3: I1 的接口版本**

`e2e/stories/issue/i1-create-issue.spec.ts`（新文件，207 行）：

````file e2e/stories/issue/i1-create-issue.spec.ts
import {
  addProjectMembers,
  amidAnotherWorkspace,
  archiveProject,
  changeProject,
  createLabel,
  createProject,
  createWorkspace,
  inviteAndAccept,
  projectMemberWrites,
  slugFor,
} from "../../fixtures/api";
import { expectIssueCreated, type ActivityRow } from "../../fixtures/assert/issue";
import { bearer, newAccount, shownNameOf } from "../../fixtures/auth";
import { createIssue, issueCreated } from "../../fixtures/issues";
import { expect, test } from "../../fixtures/test";

// I1, creating a work item (M4 design 2, 3.7, 3.13): the API version, with P1a's assertions (the work item's rows,
// its number, the default state and assignee, its assignees, labels and subscribers, the activity of its creation and
// assignees, the refusals). The page version is P14's.

/** The first row of a work item's activity: its creation. */
const created: ActivityRow = {
  field: null,
  verb: "created",
  old_value: null,
  new_value: null,
  named: null,
  comment: "created the work item",
};

/** The order of the ids, which the answers give accounts and labels in. */
function byId(a: { id: string }, b: { id: string }): number {
  return a.id < b.id ? -1 : 1;
}

/** The row of a work item's activity that assigns it to the account of email, by the display name nerve gave him. */
function assigned(email: string): ActivityRow {
  return {
    field: "assignees",
    verb: "updated",
    old_value: "",
    new_value: shownNameOf(email),
    named: email,
    comment: "added assignee ",
  };
}

test("I1 (API): a member creates a work item in a state, of a priority, assigned to two members and labelled twice: numbered 1, first in its state, with its assignees and labels, the creator and the assignees its subscribers, its creation and each assignee in its activity, all at one moment; without a state or assignees, the next is in the default state and assigned to the project's default assignee while he is its member, the one after to no one once he has left; one in a cancelled state is not completed; an account or a label named twice and a priority in another case are refused at once (422), and so are a state, a label the project lacks and a guest assigned (422); a guest creates none (403), nor does anyone in an archived project (409)", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  // ann and bob, Web's members; gus, its guest; each a member of Acme.
  const [ann, bob, gus] = await Promise.all(["ann", "bob", "gus"].map((label) => newAccount(api, testInfo, label)));
  if (!ann || !bob || !gus) {
    throw new Error("the accounts were not registered");
  }
  await Promise.all([ann, bob, gus].map((who) => inviteAndAccept(api, admin.token, slug, who, 15)));
  const web = await amidAnotherWorkspace(api, admin.token, testInfo, () =>
    createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" })
  );
  await addProjectMembers(api, admin.token, web.id, [
    { member_id: ann.id, role: 15 },
    { member_id: bob.id, role: 15 },
    { member_id: gus.id, role: 5 },
  ]);
  const bug = await createLabel(api, admin.token, web.id, { name: "Bug" });
  const ui = await createLabel(api, admin.token, web.id, { name: "UI" });
  const states = await api.GET("/api/v0/projects/{project_id}/states", {
    params: { path: { project_id: web.id } },
    headers: bearer(ann.token),
  });
  const inProgress = states.data?.data.find((s) => s.name === "In Progress")?.id;
  const cancelled = states.data?.data.find((s) => s.name === "Cancelled")?.id;
  if (!inProgress || !cancelled) {
    throw new Error(`Web's states hold no In Progress or no Cancelled: ${JSON.stringify(states.error)}`);
  }

  // ann creates Fix the login in In Progress, high, assigned to bob and the admin, labelled UI and Bug: the answer
  // gives its accounts and labels in their ids' order.
  const assignees = [bob, admin].toSorted(byId);
  const first = await createIssue(api, ann.token, web.id, {
    name: "  Fix the login ",
    state_id: inProgress,
    priority: "high",
    assignee_ids: [bob.id, admin.id],
    label_ids: [ui.id, bug.id],
  });
  expect(first).toMatchObject({
    project_id: web.id,
    sequence_id: 1,
    name: "Fix the login",
    state_id: inProgress,
    assignee_ids: assignees.map((a) => a.id),
    label_ids: [ui, bug].toSorted(byId).map((l) => l.id),
  });
  await expectIssueCreated(db, first.id, {
    sequence_id: 1,
    name: "Fix the login",
    state: "In Progress",
    priority: "high",
    sort_order: 65535,
    completed: false,
    by: ann.email,
    assignees: [admin.email, bob.email].toSorted(),
    labels: ["Bug", "UI"],
    subscribers: [admin.email, ann.email, bob.email].toSorted(),
    activity: [created, ...assignees.map((a) => assigned(a.email))],
  });

  // The admin makes ann Web's default assignee; bob's next work item, without a state or assignees, is in Backlog,
  // the default state, assigned to her.
  expect((await changeProject(api, admin.token, web.id, { default_assignee_id: ann.id })).status).toBe(200);
  const second = await createIssue(api, bob.token, web.id, { name: "Plan the release" });
  await expectIssueCreated(db, second.id, {
    sequence_id: 2,
    name: "Plan the release",
    state: "Backlog",
    priority: "none",
    sort_order: 65535,
    completed: false,
    by: bob.email,
    assignees: [ann.email],
    labels: [],
    subscribers: [ann.email, bob.email].toSorted(),
    activity: [created, assigned(ann.email)],
  });
  // Once she has left Web, the next is assigned to no one, after the second in Backlog.
  expect((await projectMemberWrites(api, web.id).leave(ann.token)).status).toBe(204);
  const third = await createIssue(api, bob.token, web.id, { name: "Write the notes" });
  await expectIssueCreated(db, third.id, {
    sequence_id: 3,
    name: "Write the notes",
    state: "Backlog",
    priority: "none",
    sort_order: 75535,
    completed: false,
    by: bob.email,
    assignees: [],
    labels: [],
    subscribers: [bob.email],
    activity: [created],
  });

  // One in Cancelled, a state of the cancelled group, is not completed (M4 design 3.7).
  const fourth = await createIssue(api, bob.token, web.id, { name: "Drop the beta", state_id: cancelled });
  await expectIssueCreated(db, fourth.id, {
    sequence_id: 4,
    name: "Drop the beta",
    state: "Cancelled",
    priority: "none",
    sort_order: 65535,
    completed: false,
    by: bob.email,
    assignees: [],
    labels: [],
    subscribers: [bob.email],
    activity: [created],
  });

  // An account and a label named twice and a priority in another case (the contract's values are lower case) are
  // refused, the three at once; so are a state and a label Web does not have and gus, Web's guest, as an assignee:
  // every problem in one answer.
  expect(
    await issueCreated(api, bob.token, web.id, {
      name: "Fix",
      // @ts-expect-error -- a priority in another case breaks the contract on purpose
      priority: "High",
      assignee_ids: [bob.id, bob.id],
      label_ids: [bug.id, bug.id],
    })
  ).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [
      { field: "priority", code: "invalid_format" },
      { field: "assignee_ids[1]", code: "duplicate" },
      { field: "label_ids[1]", code: "duplicate" },
    ],
  });
  expect(
    await issueCreated(api, bob.token, web.id, {
      name: "Fix",
      state_id: crypto.randomUUID(),
      assignee_ids: [gus.id],
      label_ids: [crypto.randomUUID()],
    })
  ).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [
      { field: "state_id", code: "not_allowed" },
      { field: "assignee_ids[0]", code: "not_allowed" },
      { field: "label_ids[0]", code: "not_allowed" },
    ],
  });
  // gus creates none; in archived Web nobody creates one.
  expect(await issueCreated(api, gus.token, web.id, { name: "Fix" })).toEqual({ status: 403, code: "forbidden" });
  await archiveProject(api, admin.token, web.id);
  expect(await issueCreated(api, bob.token, web.id, { name: "Fix" })).toEqual({
    status: 409,
    code: "project.archived",
  });
});
````

- [ ] **Step 4: M3 的 P4、W2、W3、W11 带着工作项一侧的行**

`e2e/stories/project/p4-archive.spec.ts`（修改，4 处）：

````old e2e/stories/project/p4-archive.spec.ts
} from "../../fixtures/dialogs";
````
````new e2e/stories/project/p4-archive.spec.ts
} from "../../fixtures/dialogs";
import { furnishIssueRows } from "../../fixtures/furnish";
````
````old e2e/stories/project/p4-archive.spec.ts
test("P4 (API): the admin archives a project, which leaves the list for the archived ones and whose fields cannot be updated (409) until it is unarchived; unarchives it and archives it again; then labels it and deletes it with its members, settings, states and labels at one moment, a label deleted before keeping its own, after which it is not found and its identifier is free", async ({
````
````new e2e/stories/project/p4-archive.spec.ts
test("P4 (API): the admin archives a project, which leaves the list for the archived ones and whose fields cannot be updated (409) until it is unarchived; unarchives it and archives it again; then labels it and, unarchived for a moment, the member creates a work item in it; it is deleted with its members, settings, states, labels and work item at one moment, a label deleted before keeping its own, after which it is not found and its identifier is free", async ({
````
````old e2e/stories/project/p4-archive.spec.ts
  ]);

  // An archived project is deleted as any other.
````
````new e2e/stories/project/p4-archive.spec.ts
  ]);

  // The member creates a work item in Web, assigned to him and labelled Bug, while it is unarchived: an archived
  // project takes none (M4 design 3.8). Its rows are his own writing, which the deletion must write again as the
  // admin's.
  expect(await act(api, admin, web.id, "unarchive")).toBeNull();
  await furnishIssueRows(api, member, web.id, memberId, bug.id);
  expect(await act(api, admin, web.id, "archive")).toEqual(expect.any(String));
  // An archived project is deleted as any other.
````
````old e2e/stories/project/p4-archive.spec.ts
  // Web has a row in each table its deletion writes: its label Bug too (expectProjectDeleted).
  await createLabel(api, admin.access_token, web.id, { name: "Bug" });
````
````new e2e/stories/project/p4-archive.spec.ts
  // Web has a row in each table its deletion writes (expectProjectDeleted): its label Bug too, and a work item
  // assigned to wes and labelled Bug.
  const bug = await createLabel(api, admin.access_token, web.id, { name: "Bug" });
  await furnishIssueRows(api, admin.access_token, web.id, wesId, bug.id);
````

`e2e/stories/workspace/w11-guest-bounds.spec.ts`（修改，2 处）：

````old e2e/stories/workspace/w11-guest-bounds.spec.ts
import { expectMembership, expectMembershipEnded, workspaceTables } from "../../fixtures/assert/workspace";
````
````new e2e/stories/workspace/w11-guest-bounds.spec.ts
import { workspaceTables } from "../../fixtures/assert/catalog";
import { expectMembership, expectMembershipEnded } from "../../fixtures/assert/workspace";
````
````old e2e/stories/workspace/w11-guest-bounds.spec.ts
        ["workspaces", ...workspaceTables].map(async (table) => [
````
````new e2e/stories/workspace/w11-guest-bounds.spec.ts
        ["workspaces", ...(await workspaceTables(db))].map(async (table) => [
````

`e2e/stories/workspace/w2-landing.spec.ts`（修改，3 处）：

````old e2e/stories/workspace/w2-landing.spec.ts
  createWorkspace,
  furnishWorkspace,
````
````new e2e/stories/workspace/w2-landing.spec.ts
  createWorkspace,
````
````old e2e/stories/workspace/w2-landing.spec.ts
import { EMOJI_CHECK_WARNING, expectQuietConsole } from "../../fixtures/browser";
````
````new e2e/stories/workspace/w2-landing.spec.ts
import { EMOJI_CHECK_WARNING, expectQuietConsole } from "../../fixtures/browser";
import { furnishWorkspace } from "../../fixtures/furnish";
````
````old e2e/stories/workspace/w2-landing.spec.ts
  // the workspace); her display settings; and her project Web with its own rows, its label Bug among them.
````
````new e2e/stories/workspace/w2-landing.spec.ts
  // the workspace); her display settings; and her project Web with its own rows, its label Bug and a work item
  // labelled Bug among them.
````

`e2e/stories/workspace/w3-workspace-settings-page.spec.ts`（新文件，166 行）：

````file e2e/stories/workspace/w3-workspace-settings-page.spec.ts
import { changeRole, createWorkspace, slugFor } from "../../fixtures/api";
import { expectWorkspaceDeleted } from "../../fixtures/assert/workspace";
import { accountId, anotherTabSignsIn, emailFor } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { closedByEscape } from "../../fixtures/dialogs";
import { furnishWorkspace } from "../../fixtures/furnish";
import { answerTo, holdAnswer, registerOnboarded, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { anotherBrowser, confirmDeletion, deleteFromGeneralPage, pickTimeZone } from "../../fixtures/workspace-pages";

// W3, the workspace's settings (M3 design 2): the page versions, with the session switch of 7.1. The API version is
// in w3-workspace-settings.spec.ts.

test("W3 (page): the admin changes the name, size and time zone, which hold after a reload; a member sees them and can change nothing; demoted, the admin is refused and told why; the admin deletes the workspace by its name and lands on his other one", async ({
  api,
  baseURL,
  browser,
  db,
  signedInPage,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = await registerOnboarded(api, adminEmail);
  const memberEmail = emailFor(testInfo, "member");
  const member = await registerOnboarded(api, memberEmail);
  const slug = slugFor(testInfo);
  const other = slugFor(testInfo, "other");
  await createWorkspace(api, admin.access_token, { name: "Acme", slug });
  await createWorkspace(api, admin.access_token, { name: "Zeta", slug: other });
  const memberAccount = { email: memberEmail, token: member.access_token };
  await furnishWorkspace(api, admin.access_token, slug, memberAccount, emailFor(testInfo, "invitee"));
  const memberId = await accountId(api, member.access_token);

  const page = await signedInPage(admin);
  const watch = await watchPage(page);
  await page.goto(`/${slug}/settings`);
  await expect(page.locator("#name")).toHaveValue("Acme");
  // Sent empty, the name's rule fails: the name takes the focus, its message its description.
  await page.locator("#name").fill("");
  await page.getByRole("button", { name: "Update workspace" }).click();
  await expect(page.locator("#name")).toBeFocused();
  await expect(page.locator("#name")).toHaveAccessibleDescription("Workspace name is required");
  await page.locator("#name").fill("Acme Corp");
  await page.getByRole("button", { name: "Select organization size" }).click();
  await page.getByRole("option", { name: "11-50" }).click();
  // The time zone by the keyboard alone (M3 design 7.7): the button then names the zone picked, and keeps the focus.
  await pickTimeZone(page, "UTC", "Asia/Shanghai");
  await expect(page.getByRole("button", { name: "Beijing" })).toBeFocused();
  // The page sends the fields its form edits, the size as nerve's OrganizationSize names it.
  const updated = await sentTo(page, "PATCH", `/api/v0/workspaces/${slug}`, () =>
    page.getByRole("button", { name: "Update workspace" }).click()
  );
  expect([updated.answer.status(), updated.body]).toEqual([
    200,
    { name: "Acme Corp", organization_size: "11-50", timezone: "Asia/Shanghai" },
  ]);
  await expect(page.getByText("Workspace updated successfully")).toBeVisible();
  expect(
    await db.query(
      `SELECT w.name, w.organization_size, w.timezone, w.updated_by_id = u.id AS by_the_admin
         FROM workspaces w JOIN users u ON u.email = $2 WHERE w.slug = $1`,
      [slug, adminEmail]
    )
  ).toEqual([{ name: "Acme Corp", organization_size: "11-50", timezone: "Asia/Shanghai", by_the_admin: true }]);
  await page.reload();
  await expect(page.locator("#name")).toHaveValue("Acme Corp");
  await expect(page.getByRole("button", { name: "11-50" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Beijing" })).toBeVisible();

  // The member sees what nerve holds, in a form he cannot change; he has no update and no deletion.
  const theMember = await anotherBrowser(browser, baseURL ?? "", member);
  const memberWatch = await watchPage(theMember.page);
  await theMember.page.goto(`/${slug}/settings`);
  await expect(theMember.page.locator("#name")).toHaveValue("Acme Corp");
  await expect(theMember.page.locator("#name")).toBeDisabled();
  await expect(theMember.page.getByRole("button", { name: "Update workspace" })).toHaveCount(0);
  await expect(theMember.page.getByRole("button", { name: "Delete", exact: true })).toHaveCount(0);
  expect([memberWatch.apiFailures, memberWatch.oldApiRequests, memberWatch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(theMember.page, memberWatch, { warnings: [EMOJI_CHECK_WARNING] });
  await theMember.close();

  // Made an admin, the member demotes the admin while his page is open: nerve refuses his next update, and the page
  // says why and lets him try again. Made an admin again, he goes on.
  const adminId = await accountId(api, admin.access_token);
  await changeRole(api, admin.access_token, slug, memberId, 20);
  await changeRole(api, member.access_token, slug, adminId, 15);
  const refused = await answerTo(page, "PATCH", `/api/v0/workspaces/${slug}`, () =>
    page.getByRole("button", { name: "Update workspace" }).click()
  );
  expect(refused.status()).toBe(403);
  await expect(page.getByText("Your role does not allow this.")).toBeVisible();
  await expect(page.getByRole("button", { name: "Update workspace" })).toBeEnabled();
  await changeRole(api, member.access_token, slug, adminId, 20);

  // The admin deletes it; the root lands him on his other workspace, and the page says so.
  expect((await deleteFromGeneralPage(page, slug, "Acme Corp")).status()).toBe(204);
  await expect(page).toHaveURL(`/${other}`);
  await expect(page.getByText("Workspace deleted.")).toBeVisible();
  await expectWorkspaceDeleted(db, slug, adminEmail, ["workspace_member_invites"]);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`403 PATCH /api/v0/workspaces/${slug}`],
    [],
    [],
  ]);
  // Two loads of the settings: the first, and the reload; the deletion's landing navigates within the app. The
  // browser's report of the refusal.
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING],
    errors: ["Failed to load resource: the server responded with a status of 403 (Forbidden)"],
  });
});

// The session switch (M3 design 7.1, 9.6): a change the page sent before another tab moved it to another account may
// still succeed after, and the page is that account's then. The request reaches nerve before the switch, as X's: sent
// after it, it would be Y's, refused, and the page would stay put without any check of the session.
test("W3 (page): a deletion nerve made before another tab signed another account in, and answered after, neither moves the page nor says so: the page is that account's", async ({
  api,
  context,
  db,
  signedInPage,
}, testInfo) => {
  const x = emailFor(testInfo, "x");
  const y = emailFor(testInfo, "y");
  const xTokens = await registerOnboarded(api, x);
  const yTokens = await registerOnboarded(api, y);
  const slug = slugFor(testInfo);
  await createWorkspace(api, xTokens.access_token, { name: "Doomed", slug });
  await createWorkspace(api, xTokens.access_token, { name: "Kept", slug: slugFor(testInfo, "kept") });
  await createWorkspace(api, yTokens.access_token, { name: "Yours", slug: slugFor(testInfo, "yours") });
  const tabA = await signedInPage(xTokens);
  const watch = await watchPage(tabA);
  await tabA.goto(`/${slug}/settings`);
  await expect(tabA.locator("#name")).toHaveValue("Doomed");

  // X deletes Doomed: nerve deletes it at once, and its answer waits (holdAnswer: route.fetch, later route.fulfill).
  const release = await holdAnswer(tabA, "DELETE", `/api/v0/workspaces/${slug}`);
  await confirmDeletion(tabA, "Doomed");
  // While it is out the dialog cannot be dismissed: Cancel is disabled, and Escape leaves it open.
  await expect(tabA.getByRole("dialog").getByRole("button", { name: "Cancel" })).toBeDisabled();
  expect(await closedByEscape(tabA)).toBe(false);
  const deletedAt = async () =>
    (await db.query<{ deleted_at: Date | null }>(`SELECT deleted_at FROM workspaces WHERE slug = $1`, [slug]))[0]
      ?.deleted_at ?? null;
  await expect.poll(deletedAt).toBeInstanceOf(Date);

  // Tab B keeps a sign-in of Y as the token manager does; tab A follows, and Doomed is not one of Y's.
  await anotherTabSignsIn(context, api, y);
  await expect(tabA.getByText("Workspace not found")).toBeVisible();

  // nerve's answer reaches tab A, waited for from its release on (answerTo), so that the wait's deadline is the
  // answer's alone. A move would come with the deletion's continuation, as its answer settles it: the window of 2 s
  // after the release is orders of magnitude longer than the moment that takes.
  const moved = tabA
    .waitForURL((url) => url.pathname !== `/${slug}/settings`, { timeout: 2_000 })
    .then(
      () => true,
      () => false
    );
  expect((await answerTo(tabA, "DELETE", `/api/v0/workspaces/${slug}`, release)).status()).toBe(204);
  expect(await moved).toBe(false);
  // counted at once, as a retrying check would pass once a toast had gone
  expect(await tabA.getByText("Workspace deleted.").count()).toBe(0);
  await expect(tabA.getByText("Workspace not found")).toBeVisible();
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  // One load of tab A.
  await expectQuietConsole(tabA, watch, { warnings: [EMOJI_CHECK_WARNING] });
});
````

`e2e/stories/workspace/w3-workspace-settings.spec.ts`（整个文件，252 行）：

````whole e2e/stories/workspace/w3-workspace-settings.spec.ts
import {
  changeWorkspacePreferences,
  createLabel,
  createProject,
  createWorkspace,
  invite,
  inviteAndAccept,
  slugFor,
  type Workspace,
} from "../../fixtures/api";
import { projectTables } from "../../fixtures/assert/catalog";
import { expectLabels, expectProjectCreated, expectProjectDeleted, type LabelRow } from "../../fixtures/assert/project";
import {
  expectInvitations,
  expectMembership,
  expectPreferences,
  expectWorkspaceDeleted,
} from "../../fixtures/assert/workspace";
import { accountId, bearer, emailFor, register, registeredPAT } from "../../fixtures/auth";
import { furnishIssueRows } from "../../fixtures/furnish";
import { expect, test } from "../../fixtures/test";

// W3, the workspace's settings (M3 design 2): the API version. The page versions, with the session switch of 7.1,
// are in w3-workspace-settings-page.spec.ts.

test("W3 (API): the admin changes the workspace and deletes it with its members, invitations, settings and projects at one moment; a member may do neither, and the slug never changes", async ({
  api,
  db,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await registeredPAT(api, adminEmail)).token;
  const memberEmail = emailFor(testInfo, "member");
  const member = (await registeredPAT(api, memberEmail)).token;
  const declinerEmail = emailFor(testInfo, "decliner");
  const decliner = (await register(api, declinerEmail)).access_token;
  const inviteeEmail = emailFor(testInfo, "invitee");
  const slug = slugFor(testInfo);
  const other = slugFor(testInfo, "other");
  // Two workspaces alike, each with the member, a pending and a declined invitation, the member's settings and a
  // project the admin created with the member its lead, which the member labelled; only Acme is deleted. Both exist
  // before either is furnished, and the member is Other's admin and Acme's member: each answer, role and row must be
  // the workspace's own, whichever row the database reads first.
  const [adminId, memberId] = await Promise.all([accountId(api, admin), accountId(api, member)]);
  const web = { name: "Web", identifier: "WEB", description: "", network: 2, timezone: "UTC", logo_props: {} };
  const otherWorkspace = await createWorkspace(api, admin, { name: "Other", slug: other });
  const acme = await createWorkspace(api, admin, { name: "Acme", slug });
  const tokens: string[] = [];
  const furnish = async ({ slug: target, id }: Workspace, memberRole: 15 | 20) => {
    expect(await inviteAndAccept(api, admin, target, { email: memberEmail, token: member }, memberRole)).toMatchObject({
      slug: target,
      role: memberRole,
      total_members: 2,
    });
    const created = await invite(api, admin, target, [
      { email: inviteeEmail, role: 5 },
      { email: declinerEmail, role: 15 },
    ]);
    tokens.push(...created.map((i) => i.token));
    const [, toDecline] = created;
    if (!toDecline) {
      throw new Error(`the invitations to ${target} were not created`);
    }
    const declined = await api.POST("/api/v0/workspace-invitations/{invitation_id}/decline", {
      params: { path: { invitation_id: toDecline.id } },
      body: { token: toDecline.token },
      headers: bearer(decliner),
    });
    expect(declined.response.status).toBe(204);
    await changeWorkspacePreferences(api, member, target, { navigation_project_limit: 3 });
    // The answer is the workspace's own new project as its creator sees it, first in both admins' sidebars.
    const project = await createProject(api, admin, target, {
      name: "Web",
      identifier: "web",
      project_lead_id: memberId,
    });
    expect(project).toMatchObject({
      workspace_id: id,
      identifier: "WEB",
      member_role: 20,
      sort_order: 65535,
      member_ids: [adminId, memberId],
    });
    // The member, Web's lead and so its admin, labels it: Bug, and UI under Bug.
    const bug = await createLabel(api, member, project.id, { name: "Bug", color: "#EF4444" });
    await createLabel(api, member, project.id, { name: "UI", parent_id: bug.id });
    return { project, bug };
  };
  const { project: otherWeb } = await furnish(otherWorkspace, 20);
  const { project: acmeWeb, bug: acmeBug } = await furnish(acme, 15);
  // Acme's Web has a work item too, the member's, assigned to him and labelled Bug, whose rows Acme's deletion must
  // write again as the admin's; Other's Web has none, as made (expectProjectCreated).
  await furnishIssueRows(api, member, acmeWeb.id, memberId, acmeBug.id);
  /** Web's labels, as the member made them; deleted by the admin, as Acme's deletion writes them. */
  const webLabels = (deleted: boolean): LabelRow[] => [
    { name: "Bug", color: "#EF4444", parent: null, sort_order: 65535, deleted, by: deleted ? adminEmail : memberEmail },
    { name: "UI", color: "", parent: "Bug", sort_order: 75535, deleted, by: deleted ? adminEmail : memberEmail },
  ];
  await expectLabels(db, acmeWeb.id, webLabels(false));
  // The member reads Other as its admin, whatever becomes of Acme.
  const readOther = async () => {
    const read = await api.GET("/api/v0/workspaces/{slug}", {
      params: { path: { slug: other } },
      headers: bearer(member),
    });
    expect([read.response.status, read.data]).toEqual([
      200,
      expect.objectContaining({ name: "Other", slug: other, role: 20, total_members: 2 }),
    ]);
  };

  // A second admin changes it, so that the deletion must record its own author.
  const coAdminEmail = emailFor(testInfo, "co-admin");
  const coAdmin = (await registeredPAT(api, coAdminEmail)).token;
  await inviteAndAccept(api, admin, slug, { email: coAdminEmail, token: coAdmin }, 20);
  const renamed = await api.PATCH("/api/v0/workspaces/{slug}", {
    params: { path: { slug } },
    body: { name: "Acme Corp", organization_size: "11-50", timezone: "Europe/Berlin" },
    headers: bearer(coAdmin),
  });
  expect(renamed.response.status).toBe(200);
  expect(renamed.data).toMatchObject({
    slug,
    name: "Acme Corp",
    organization_size: "11-50",
    timezone: "Europe/Berlin",
    role: 20,
    total_members: 3,
  });
  expect(
    await db.query(
      `SELECT w.name, w.organization_size, w.timezone, w.updated_by_id = u.id AS updated_by_the_co_admin
         FROM workspaces w JOIN users u ON u.email = $2 WHERE w.slug = $1`,
      [slug, coAdminEmail]
    )
  ).toEqual([
    { name: "Acme Corp", organization_size: "11-50", timezone: "Europe/Berlin", updated_by_the_co_admin: true },
  ]);
  await readOther();

  // A member may not change it; a slug in the body is refused before anything is looked at.
  const byMember = await api.PATCH("/api/v0/workspaces/{slug}", {
    params: { path: { slug } },
    body: { name: "Mine" },
    headers: bearer(member),
  });
  expect(byMember.response.status).toBe(403);
  expect(byMember.error?.code).toBe("forbidden");
  const withSlug = await api.PATCH("/api/v0/workspaces/{slug}", {
    params: { path: { slug } },
    body: { slug: "other" } as never,
    headers: bearer(admin),
  });
  expect(withSlug.response.status).toBe(400);
  const memberDeletes = await api.DELETE("/api/v0/workspaces/{slug}", {
    params: { path: { slug } },
    headers: bearer(member),
  });
  expect(memberDeletes.response.status).toBe(403);
  expect(memberDeletes.error?.code).toBe("forbidden");

  // A project deleted before the workspace keeps its moment, and so do its rows: Old, with the member its lead, so
  // that each project table has a row of it.
  const old = await createProject(api, admin, slug, { name: "Old", identifier: "OLD", project_lead_id: memberId });
  // The member moves Old in his sidebar, labels it and creates a work item in it, assigned to him and labelled Bug: his
  // display settings in it, its label and the work item's rows are his own writing, which Old's deletion must write
  // again as the admin's.
  const oldBug = await createLabel(api, member, old.id, { name: "Bug" });
  await furnishIssueRows(api, member, old.id, memberId, oldBug.id);
  const moved = await api.PATCH("/api/v0/me/projects/{project_id}/preferences", {
    params: { path: { project_id: old.id } },
    body: { sort_order: 75535 },
    headers: bearer(member),
  });
  expect(moved.response.status).toBe(200);
  expect(
    await db.query(
      `SELECT s.sort_order, b.email AS by
         FROM project_user_properties s JOIN users u ON u.id = s.user_id JOIN users b ON b.id = s.updated_by_id
        WHERE s.project_id = $1 AND u.email = $2 AND s.deleted_at IS NULL`,
      [old.id, memberEmail]
    )
  ).toEqual([{ sort_order: 75535, by: memberEmail }]);
  await expectLabels(db, old.id, [
    { name: "Bug", color: "", parent: null, sort_order: 65535, deleted: false, by: memberEmail },
  ]);
  const oldDeleted = await api.DELETE("/api/v0/projects/{project_id}", {
    params: { path: { project_id: old.id } },
    headers: bearer(admin),
  });
  expect(oldDeleted.response.status).toBe(204);

  const deleted = await api.DELETE("/api/v0/workspaces/{slug}", { params: { path: { slug } }, headers: bearer(admin) });
  expect(deleted.response.status).toBe(204);
  // The rows a story can delete before their workspace keep that moment: an invitation, when it is accepted (M3
  // design 3.8); Old and each row under it, with Old.
  await expectWorkspaceDeleted(db, slug, adminEmail, [
    "workspace_member_invites",
    "projects",
    ...(await projectTables(db)),
  ]);
  await expectLabels(db, acmeWeb.id, webLabels(true));
  // Each of Web's labels carries Acme's deletion, its moment and its author: expectWorkspaceDeleted's counts cannot
  // tell a label stamped earlier, as Old's Bug, deleted before, gives the labels an earlier row.
  expect(
    await db.query(
      `SELECT l.name, l.deleted_at = w.deleted_at AS at_acmes_moment, l.updated_by_id = w.updated_by_id AS by_acmes_deleter
         FROM labels l JOIN workspaces w ON w.id = l.workspace_id WHERE l.project_id = $1 ORDER BY l.sort_order`,
      [acmeWeb.id]
    ),
    "when and by whom Web's labels were deleted"
  ).toEqual([
    { name: "Bug", at_acmes_moment: true, by_acmes_deleter: true },
    { name: "UI", at_acmes_moment: true, by_acmes_deleter: true },
  ]);
  // Old and every row under it keep its deletion: its moment and its author, the admin.
  await expectProjectDeleted(db, old.id, adminEmail);
  // The invitations accepted before keep the moment they were answered; the declined one goes with the pending
  // one; the other workspace keeps everything.
  const invitations = (memberRole: number, withTheWorkspace: boolean) => [
    { email: memberEmail, role: memberRole, accepted: true, responded: true, deleted: true },
    { email: inviteeEmail, role: 5, accepted: false, responded: false, deleted: withTheWorkspace },
    { email: declinerEmail, role: 15, accepted: false, responded: true, deleted: withTheWorkspace },
  ];
  await expectInvitations(
    db,
    slug,
    adminEmail,
    [...invitations(15, true), { email: coAdminEmail, role: 20, accepted: true, responded: true, deleted: true }],
    tokens
  );
  await expectInvitations(db, other, adminEmail, invitations(20, false), tokens);
  await expectMembership(db, other, memberEmail, { role: 20, is_active: true });
  await expectProjectCreated(db, other, web, adminEmail, memberEmail, [
    { email: adminEmail, sort_order: 65535 },
    { email: memberEmail, sort_order: 65535 },
  ]);
  await expectPreferences(db, other, memberEmail, {
    navigation_control_preference: "ACCORDION",
    navigation_project_limit: 3,
  });
  await expectLabels(db, otherWeb.id, webLabels(false));
  await readOther();
  const gone = await Promise.all(
    [admin, member].map((token) =>
      api.GET("/api/v0/workspaces/{slug}", { params: { path: { slug } }, headers: bearer(token) })
    )
  );
  expect(gone.map((g) => [g.response.status, g.error?.code])).toEqual([
    [404, "workspace.not_found"],
    [404, "workspace.not_found"],
  ]);
});
````

- [ ] **Step 5: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 `keywords: 72 rules, 23 exceptions, no hits.`

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 121 个全部通过（`withoutRows` 和手写的 `deletedAloneTables` 都已删去）。

- [ ] **Step 6: 提交**

```bash
git add e2e/fixtures/api.ts e2e/fixtures/assert/catalog.ts e2e/fixtures/assert/issue.ts e2e/fixtures/assert/project.ts e2e/fixtures/assert/workspace.ts e2e/fixtures/furnish.ts e2e/fixtures/issues.ts e2e/stories/issue/i1-create-issue.spec.ts e2e/stories/project/p4-archive.spec.ts e2e/stories/workspace/w11-guest-bounds.spec.ts e2e/stories/workspace/w2-landing.spec.ts e2e/stories/workspace/w3-workspace-settings-page.spec.ts e2e/stories/workspace/w3-workspace-settings.spec.ts
```
```bash
git commit -m "test(M4/P1): I1 creates work items through the API, and M3's deletion stories delete theirs

The stories create work items through createIssue, and a deletion
story furnishes a project with one, assigned and labelled, so every
work-item table has a row to delete. I1 checks a creation's rows,
number, place, assignees, labels, subscribers, activity and moment, the
project's default state and assignee, and the refusals. P4, W2 and W3
delete projects and workspaces with their work items. The tables a
deleted project or workspace must have written are read from the
catalog and checked against the columns that name them, so the
temporary exclusion of the work items' tables ends. W3's page tests
move to a file of their own.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p1a.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试；结构是迁移的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是关键词守卫；静态是 `make lint-web` 的文案和键的核对）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T14.1` | `issue_subscribers` 的 `project_id` 不指向项目：按外键读出的项目下的表漏掉它 | `migrations` 的 `TestConstraintAndIndexNames`（`make test`）、故事 `p4-archive`、故事 `w3-workspace-settings` | 结构；端到端 |
| `T14.2` | `issue_activities` 的 `workspace_id` 不指向工作区：按外键读出的工作区下的表漏掉它 | `migrations` 的 `TestConstraintAndIndexNames`（`make test`）、故事 `w3-workspace-settings`、故事 `w11-guest-bounds` | 结构；端到端 |
| `T14.3` | `expectProjectDeleted` 留着过渡的排除（Task 2–13）：工作项的表不必有随项目删除的行 | 故事 `p4-archive`、故事 `w3-workspace-settings` | 端到端 |

---

### Task 15: 文档：3.23 中 P1a 的行，交接的处理结果

**Files:**
- Modify: `docs/v0/M3-workspace-project/M3-design.md`、`docs/v0/M4-issue-core/handoffs/M1-P2-trim-content.md`、`docs/v0/M4-issue-core/handoffs/M1-P3-trim-platform.md`、`docs/v0/M4-issue-core/handoffs/M3-closeout.md`、`docs/v0/plane-diff.md`、`docs/v0/v0-design.md`

**Interfaces:**（spec 2.14、第 7 节；M4 设计 3.23）
- 总体设计：
  - 4.2：加锁顺序延伸到 M4 的表；工作项的写最先锁项目行，有预检；删除工作区的新顺序；方案 B 的两个代价；
  - 6.2：`issue` 模块和它的表。
- M3 设计：
  - 3.6 的取锁表：删除工作区、删除项目、删除状态、删除标签四行；
  - 3.17：`project.state_in_use`；
  - 17.4：M4 的选择。
- 差异清单：二·按表的五张表逐列；第四节中 4.17 标 P1a 的行。
- 三份交接（`M1-P2-trim-content.md`、`M1-P3-trim-platform.md`、`M3-closeout.md`）各写下"处理结果（M4/P1a）"，`status` 照旧 `open`（spec 第 7 节）。

**Tests:** 只改文档：`make lint-web`（关键词守卫读文档）和 `doclinks.py`（相对链接）。

- [ ] **Step 1: 文档**

`docs/v0/M3-workspace-project/M3-design.md`（修改，6 处）：

````old docs/v0/M3-workspace-project/M3-design.md
    - **一个改变一个时刻，对连带改写的每一列都成立**（负责人 2026-10-02 的裁定，17.4）：调用方在持有工作区的 `FOR NO KEY UPDATE` 之后才读 `now`（删除工作区、改成员的角色、接受邀请都是这样），而每个项目级的写最先以 `FOR SHARE` 锁住它的工作区行、在全部锁之后读自己的时刻（3.6 约定二）。两把锁互斥：连带读 `now` 时，在它要改的行上没有还未提交的项目级的写，先到的写都已提交、时刻不晚于 `now`，后到的写等连带提交、时刻晚于 `now`。所以连带写下的 `updated_at` 不早于行上原有的，`deleted_at`、`updated_at` 也不早于 `created_at`。这是连带与项目级的写之间的保证；两个只持共享锁的写在同一行上相遇时不成立：同一个账户并发改自己的显示设置，两个写都持项目（或工作区）的 S、各自在锁之后读时刻，到 `UPSERT` 里才相遇，后提交的可以带较早的时刻（P4b review 第 7 节，只涉及调用者自己那一行的审计列）。
````
````new docs/v0/M3-workspace-project/M3-design.md
    - **一个改变一个时刻，对连带改写的每一列都成立**（负责人 2026-10-02 的裁定，17.4）：调用方在持有工作区的 `FOR NO KEY UPDATE` 之后才读 `now`（删除工作区、改成员的角色、接受邀请都是这样），而每个项目级的写最先以 `FOR SHARE` 锁住它的工作区行、在全部锁之后读自己的时刻（3.6 约定二）。两把锁互斥：连带读 `now` 时，在它要改的行上没有还未提交的项目级的写，先到的写都已提交、时刻不晚于 `now`，后到的写等连带提交、时刻晚于 `now`。所以连带写下的 `updated_at` 不早于行上原有的，`deleted_at`、`updated_at` 也不早于 `created_at`。M4 的工作项一侧的写不锁工作区行、最先锁项目行（M4 设计 3.5，决策点 1 的 B）：删除项目、删除工作区持这些项目的 N 之后才读时刻，同一个保证对工作项一侧的行经项目行成立（M4/P1a 起）。这是连带与项目级的写之间的保证；两个只持共享锁的写在同一行上相遇时不成立：同一个账户并发改自己的显示设置，两个写都持项目（或工作区）的 S、各自在锁之后读时刻，到 `UPSERT` 里才相遇，后提交的可以带较早的时刻（P4b review 第 7 节，只涉及调用者自己那一行的审计列）。
````
````old docs/v0/M3-workspace-project/M3-design.md
  | 删除工作区 | 工作区 N → 判定 → 软删除工作区 → 邀请（N，`id` 升序，一条语句，回答它们的 `id`；再删除这些 `id` 的，约定五的例外）→ 成员 → 显示设置 → `DeleteWorkspaceProjects`：项目 → 项目成员 → 项目显示设置 → 状态 → 标签（批量，约定五） |
````
````new docs/v0/M3-workspace-project/M3-design.md
  | 删除工作区 | 工作区 N → 判定 → 邀请（N，`id` 升序，一条语句，回答它们的 `id`，约定五的例外）→ 这个工作区的全部未删除项目（N，`id` 升序，一条语句，`ProjectCascade.LockWorkspaceProjects`）→ 读时刻 → 软删除工作区 → 删除锁住的邀请 → 成员 → 显示设置 → `DeleteWorkspaceProjects`：项目 → 项目成员 → 项目显示设置 → 状态 → 标签（批量，约定五）→ 锁住的项目的工作项一侧的行。锁项目、在它之后读时刻是 M4/P1a 起的（M4 设计 3.5）：工作项的写只锁项目行、不锁工作区行，删除的时刻要不早于这些项目上已提交的每个写 |
````
````old docs/v0/M3-workspace-project/M3-design.md
  | 删除项目 | 工作区 S → 项目 N → 判定 → 软删除项目 → 项目成员 → 显示设置 → 状态 → 标签（批量） |
````
````new docs/v0/M3-workspace-project/M3-design.md
  | 删除项目 | 工作区 S → 项目 N → 判定 → 软删除项目 → 项目成员 → 显示设置 → 状态 → 标签（批量）→ 工作项一侧的行（M4/P1a 起，M4 设计 3.6） |
````
````old docs/v0/M3-workspace-project/M3-design.md
  | 删除状态 | 工作区 S → 项目 N → 重读状态 → 判定 → 守卫的写（守卫就是默认的检查）→ 写之后数它原来的组，组空了时拒绝、事务回滚（3.17） |
  | 新建、修改、删除标签 | 工作区 S → 项目 N →（修改、删除）重读标签 → 判定 → 层级检查 → 写（删除连带子标签，批量） |
````
````new docs/v0/M3-workspace-project/M3-design.md
  | 删除状态 | 工作区 S → 项目 N → 重读状态 → 判定 → 有工作项在它里面时拒绝（M4/P1a 起，3.17）→ 守卫的写（守卫就是默认的检查）→ 写之后数它原来的组，组空了时拒绝、事务回滚（3.17） |
  | 新建、修改、删除标签 | 工作区 S → 项目 N →（修改、删除）重读标签 → 判定 → 层级检查 → 写（删除连带子标签，批量；M4/P1a 起再从工作项上摘下它们，批量，M4 设计 3.6） |
````
````old docs/v0/M3-workspace-project/M3-design.md
- 删除默认状态 409 `project.state_default`（Plane `views/state/base.py:113` 起）。"状态下还有工作项时不能删"由 M4 加入（13.2）。
````
````new docs/v0/M3-workspace-project/M3-design.md
- 删除默认状态 409 `project.state_default`（Plane `views/state/base.py:113` 起）。有工作项（已归档的也算，已删除的不算）在这个状态里时 409 `project.state_in_use`（Plane 答 400）：由 M4/P1a 加入，在项目 N 之下、守卫的写之前经 `IssueCascade.StateInUse` 数（M4 设计 3.6、3.7），所以既是默认、又有工作项的状态答 `project.state_in_use`。
````
````old docs/v0/M3-workspace-project/M3-design.md
- 范围（负责人 2026-10-02 确认）：本约定覆盖项目管理类的写，即上面列的 P4b、P5b、P7a、P7b 的写。M4 的工作项写入是最频繁的路径（agent 高频调用），**不默认沿用**：M4 的设计单独决定它取不取工作区的 S，并把上面的代价作为设计输入；取的话要说明工作区一级的写在那样的负载下怎样不被饿死。
````
````new docs/v0/M3-workspace-project/M3-design.md
- 范围（负责人 2026-10-02 确认）：本约定覆盖项目管理类的写，即上面列的 P4b、P5b、P7a、P7b 的写。M4 的工作项写入是最频繁的路径（agent 高频调用），**不默认沿用**：M4 的设计单独决定它取不取工作区的 S，并把上面的代价作为设计输入；取的话要说明工作区一级的写在那样的负载下怎样不被饿死。M4 的选择（M4 设计 3.5，决策点 1 负责人裁定 B，2026-10-17）：工作项一侧的写不取工作区的 S，最先以 `FOR NO KEY UPDATE` 锁住项目行，取锁之前做不加锁的预检（判定拒绝的在那里答，不取锁）；删除工作区在读时刻之前按 `id` 升序锁住它的全部项目（M4/P1a 起）。
````

`docs/v0/M4-issue-core/handoffs/M1-P2-trim-content.md`（修改，1 处）：

````old docs/v0/M4-issue-core/handoffs/M1-P2-trim-content.md
来源：[M1/P2 评审记录](../../M1-frontend-trim/reviews/P2-trim-content-review.md)第 7 节。

````
````new docs/v0/M4-issue-core/handoffs/M1-P2-trim-content.md
来源：[M1/P2 评审记录](../../M1-frontend-trim/reviews/P2-trim-content-review.md)第 7 节。

## 处理结果（M4/P1a）

- **接口的字段**（工作项的第一个操作一侧完成）：`createIssue`（`POST /api/v0/projects/{project_id}/issues`，`api/modules/issue.yaml`）的 `IssueCreate`、`IssueDetail` 没有 `estimate_point`、`description_binary`；没有 `/issue-dates/` 的接口，`issues` 表也没有 `estimate_point_id`、`description_binary` 两列（M4 设计 4.2）。以后的工作项操作照此，搜索结果的类型在 M4 的 P8。

仍未处理，状态保持 `open`：`skip_activity`（M4 的 P4，随描述）；描述保存的"撤销再重做"（M4 的 P14）。

来源：[M4/P1a spec](../specs/P1a-issues.md) 第 7 节。

````

`docs/v0/M4-issue-core/handoffs/M1-P3-trim-platform.md`（修改，1 处）：

````old docs/v0/M4-issue-core/handoffs/M1-P3-trim-platform.md
来源：[M1/P3 评审记录](../../M1-frontend-trim/reviews/P3-trim-platform-review.md)第 7 节。

````
````new docs/v0/M4-issue-core/handoffs/M1-P3-trim-platform.md
来源：[M1/P3 评审记录](../../M1-frontend-trim/reviews/P3-trim-platform-review.md)第 7 节。

## 处理结果（M4/P1a）

- **工作项的字段**（工作项的第一个操作一侧完成）：`createIssue` 的 `IssueCreate`、`IssueDetail` 没有 `is_epic`、`type_id`，`issues` 表也没有 `type_id`（M4 设计 4.2）；动态的两类（`WORKLOG`、`ISSUE_ADDITIONAL_PROPERTIES_ACTIVITY`）不在 `issue_activities.verb` 的取值里（`created`、`updated`、`deleted`）。没有批量的接口。

仍未处理，状态保持 `open`：动态、评论的接口（M4 的 P4）；描述历史和按编号查找的地址（M4 的 P4、P2）；富文本筛选的显示层（M4 的 P13）；收集箱的来源（M7）。

来源：[M4/P1a spec](../specs/P1a-issues.md) 第 7 节。

````

`docs/v0/M4-issue-core/handoffs/M3-closeout.md`（修改，1 处）：

````old docs/v0/M4-issue-core/handoffs/M3-closeout.md
来源：[M3 设计](../../M3-workspace-project/M3-design.md) 13.2 的 M4 一行；M3 各 Phase 的 spec 第 5 节和评审第 6 节中交给 M4 的各条；[M3 收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 2 节。

````
````new docs/v0/M4-issue-core/handoffs/M3-closeout.md
来源：[M3 设计](../../M3-workspace-project/M3-design.md) 13.2 的 M4 一行；M3 各 Phase 的 spec 第 5 节和评审第 6 节中交给 M4 的各条；[M3 收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 2 节。

## 处理结果（M4/P1a）

- **第 2 节，先锁后判定的副作用**（完成）：工作项一侧的写在第一把锁之前做一次不加锁的预检，判定拒绝的（看不到答 404，看得到而不能写答 403）都在那里答，不取任何锁；放行的写才锁项目行（M4 设计 3.5）。`TestEachIssueWriteLocksItsProjectFirst` 核对看不到项目、看得到而不能写的账户的写不出现在 `pg_locks` 里。方案 E 的选择（不取工作区的 S，最先锁项目行，决策点 1 的 B）写在 M4 设计 3.5、M3 设计 17.4 和总体设计 4.2。
- **第 4 节，状态**（完成）：删除有工作项（已归档的也算）的状态 409 `project.state_in_use`，在项目 N 之下经 `IssueCascade.StateInUse` 数，在默认状态的检查之前（M4 设计 3.6，M3 设计 3.17）；新工作项取项目的默认状态，`State.default` 的说明补上这一句。
- **第 4 节，标签**（完成）：删除标签在同一个事务里、同一时刻从工作项上摘下它和它的子标签（`IssueCascade.DetachLabel`）；`issue_labels` 的写都持项目的 N，不落在约定五的例外里（M4 设计 3.5）。
- **第 4 节，项目**（完成一部分）：编号取 `projects.last_issue_sequence`（项目 N 之下加一）；`identifier`、`default_assignee_id` 的说明改成它们的作用（编号的前缀；新工作项的默认负责人，他仍是项目中不是访客的有效成员时才指派，否则不指派、不报错，`createIssue` 的说明写明）。`guest_view_all_features`（M4 的 P2）、`archive_in`（M4 的 P6）的说明随它们的读者改。
- **第 4 节，表和删除**（完成）：五张表都带 `workspace_id`、`project_id`，进入删除项目、删除工作区的连带，同一时刻、同一个删除者（`IssueCascade.DeleteProjectIssues`、`DeleteWorkspaceIssues`，P9 换成任务）；按目录的删除测试给每张表准备一行，不开豁免；端到端的 `workspaceTables` 从目录读出，并像 `expectProjectDeleted` 一样核对它就是有 `workspace_id` 列的每张表，两个列表保持 `COLLATE "C"`；M3 的 P4、W2、W3 经 `furnishIssueRows` 带着工作项一侧的行。

仍未处理，状态保持 `open`：第 1、3、5–9 节和第 2 节其余各项，按 M4 设计 13.1 的落点。

来源：[M4/P1a spec](../specs/P1a-issues.md) 第 7 节。

````

`docs/v0/plane-diff.md`（修改，5 处）：

````old docs/v0/plane-diff.md
| `issues` | 删除 `point`、`is_draft`、`estimate_point_id`、`type_id`、`description_binary` | 遗留列或对应功能已砍掉 |
````
````new docs/v0/plane-diff.md
| `issues` | 28 列保留 20 列（M4/P1a，`00015_issue_issues.sql`）；主键名 `issues_pkey`（Plane 是 `issue_pkey`） | M4 设计 4.2 |
| `issues` | `workspace_id`、`project_id`、`parent_id`：加上 `ON DELETE CASCADE`；`state_id`：改为 `NOT NULL`（Plane 可为空），加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL` | 二·全局（模型的 `on_delete`）；创建总有状态（项目恰有一个默认状态），删除有工作项在里面的状态被拒绝（第四节"删除状态"），`state_id` 的 `CASCADE` 只会删掉已软删除的工作项（M4 设计 3.6、3.7） |
| `issues` | `name`：新加 `CHECK (name <> '')`；`description_html`：新加 `DEFAULT '<p></p>'`；`priority`：新加 `DEFAULT 'none'` 和 `CHECK (priority IN ('urgent', 'high', 'medium', 'low', 'none'))`；`sort_order`：新加 `DEFAULT 65535`；`sequence_id`：没有默认值（Plane 的模型默认 1），新加 `CHECK (sequence_id > 0)`；新加 `issues_not_own_parent_check CHECK (parent_id <> id)`、`issues_dates_check CHECK (start_date <= target_date)`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 模型的默认值和 choices；编号只取自项目的计数列 `last_issue_sequence`；工作项不能做自己的父工作项，开始日期不晚于目标日期（M4 设计 3.7、3.9）。描述和起止日期的列随表建出，进入接口在 M4 的 P4、P2 |
````
````old docs/v0/plane-diff.md
| `issues` | **新增**唯一约束 `(project_id, sequence_id)` | Plane 只靠咨询锁保证编号不重复 |
````
````new docs/v0/plane-diff.md
| `issues` | 删除 `point`、`estimate_point_id`、`type_id`、`is_draft`、`description_binary`、`description_json` | 估算、工作项类型已砍掉；草稿是另一张表（`draft_issues`）；`description_binary` 是协同编辑的遗留；`description_json` 没有读取者（M4 设计 3.7） |
| `issues` | **新增**唯一约束 `issues_project_id_sequence_id_key UNIQUE (project_id, sequence_id)`（全表唯一，不是部分唯一）；外键列的索引只留不带条件的 `issues_workspace_id_idx`、`issues_project_id_idx`、`issues_parent_id_idx`、`issues_state_id_idx`；新加 `issues_project_id_state_id_idx ON (project_id, state_id) WHERE deleted_at IS NULL` | Plane 只靠咨询锁保证编号不重复；计数列从不回退、编号从不重用，所以全表唯一（M4 设计 3.7）；物理级联、子工作项、在用的状态按外键列找行；同一状态最大的 `sort_order`、按状态筛选（M4 设计 4.2） |
````
````old docs/v0/plane-diff.md
| `issue_labels` | **新增**部分唯一约束 `(issue_id, label_id)` | Plane 在数据库层没有这个约束 |
````
````new docs/v0/plane-diff.md
| `issue_assignees` | 10 列照搬（M4/P1a，`00016_issue_issue_assignees.sql`）；主键名 `issue_assignees_pkey`（Plane 是 `issue_assignee_pkey`） | M4 设计 4.3 |
| `issue_assignees` | `workspace_id`、`project_id`、`issue_id`、`assignee_id`：加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 二·全局（模型的 `on_delete`）；v0 不物理删除账户，`assignee_id` 的 `CASCADE` 不会触发 |
| `issue_assignees` | 部分唯一索引照搬，改名为 `issue_assignees_issue_id_assignee_id_key ON (issue_id, assignee_id) WHERE deleted_at IS NULL`（Plane 是 `issue_assignee_unique_issue_assignee_when_deleted_at_null`），去掉 `(issue_id, assignee_id, deleted_at)` 的唯一约束（二·全局）；新加 `issue_assignees_assignee_id_idx ON (assignee_id) WHERE deleted_at IS NULL`，外键列的索引只留不带条件的 `issue_assignees_issue_id_idx`、`issue_assignees_workspace_id_idx`、`issue_assignees_project_id_idx` | 个人主页的"已分配"、按负责人筛选；物理级联要不带条件的索引（M4 设计 4.3） |
| `issue_labels` | 10 列照搬（M4/P1a，`00017_issue_issue_labels.sql`）；主键名 `issue_labels_pkey`（Plane 是 `issue_label_pkey`） | M4 设计 4.4 |
| `issue_labels` | `workspace_id`、`project_id`、`issue_id`、`label_id`：加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 二·全局（模型的 `on_delete`） |
| `issue_labels` | **新增**部分唯一索引 `issue_labels_issue_id_label_id_key ON (issue_id, label_id) WHERE deleted_at IS NULL`；外键列的索引只留不带条件的 `issue_labels_issue_id_idx`、`issue_labels_label_id_idx`、`issue_labels_workspace_id_idx`、`issue_labels_project_id_idx` | Plane 在数据库层没有这个约束，同一个标签可以挂两次（M4 设计 4.4）；物理级联要不带条件的索引，标签的也是 |
| `issue_subscribers` | 10 列照搬（M4/P1a，`00018_issue_issue_subscribers.sql`） | M4 设计 4.5 |
| `issue_subscribers` | `workspace_id`、`project_id`、`issue_id`、`subscriber_id`：加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 二·全局（模型的 `on_delete`）；v0 不物理删除账户，`subscriber_id` 的 `CASCADE` 不会触发 |
| `issue_subscribers` | 部分唯一索引照搬，改名为 `issue_subscribers_issue_id_subscriber_id_key ON (issue_id, subscriber_id) WHERE deleted_at IS NULL`（Plane 是 `issue_subscriber_unique_issue_subscriber_when_deleted_at_null`），去掉 `(issue_id, subscriber_id, deleted_at)` 的唯一约束（二·全局）；新加 `issue_subscribers_subscriber_id_idx ON (subscriber_id) WHERE deleted_at IS NULL`，外键列的索引只留不带条件的 `issue_subscribers_issue_id_idx`、`issue_subscribers_workspace_id_idx`、`issue_subscribers_project_id_idx` | "已关注"、按关注者筛选；物理级联要不带条件的索引（M4 设计 4.5） |
| `issue_activities` | 20 列保留 17 列（M4/P1a，`00019_issue_issue_activities.sql`）；主键名 `issue_activities_pkey`（Plane 是 `issue_activity_pkey`） | M4 设计 4.11 |
| `issue_activities` | `issue_id`：改为 `NOT NULL`（Plane 可为空），`DO_NOTHING` 改为 `ON DELETE CASCADE`；`workspace_id`、`project_id`：加上 `ON DELETE CASCADE`；`actor_id`、`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL`；`project_id` 是这一行的工作项的项目 | 动态随工作项物理删除，不挡住清理（M4 设计 4.11、3.21）；`actor_id` 为空表示 nerve 自己（M4 设计决策点 4） |
| `issue_activities` | `verb`：新加 `DEFAULT 'created'` 和 `CHECK (verb IN ('created', 'updated', 'deleted'))`；`comment`：新加 `DEFAULT ''`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局）；`created_by_id`、`updated_by_id` 写执行者，删除时写删除者 | 模型的默认值；动态只有这三种动词（M7 的收集箱随它的迁移放宽）；Plane 在 Celery 里写动态，这两列实际总为空（M4 设计 4.11） |
| `issue_activities` | 删除 `issue_comment_id`、`epoch`、`attachments` | 评论不写动态，这个 `DO_NOTHING` 的外键本会挡住物理清理；`epoch` 只写不读；`attachments` 从不写（M4 设计 4.11、3.13） |
| `issue_activities` | 外键列的索引只留不带条件的 `issue_activities_workspace_id_idx`、`issue_activities_project_id_idx`；新加 `issue_activities_issue_id_created_at_idx ON (issue_id, created_at, id)`（不带条件） | 动态列表的游标、找"最新的一行"、物理级联都按工作项找行（M4 设计 4.11） |
````
````old docs/v0/plane-diff.md
| 工作项编号 | 咨询锁加 `issue_sequences` 表 | 在项目行上用计数列原子自增 |
````
````new docs/v0/plane-diff.md
| 工作项编号 | 咨询锁加 `issue_sequences` 表；序列化器允许客户端写 `sequence_id` | 在项目行上用计数列原子自增；客户端不能写它；`(project_id, sequence_id)` 全表唯一，编号从不重用（M4 设计 3.7） |
| 工作项的写的取锁 | 动态在 Celery 里异步写，写之间不串行 | 不加锁地预检，判定拒绝的在那里答（404、403），不取锁；放行的写以 `FOR NO KEY UPDATE` 锁住项目行，在锁下重读、再判定，动态、关注与工作项在同一个事务里写（M4 设计 3.5，决策点 1） |
| 工作项的负责人、标签不合格 | 静默去掉 | 负责人不是项目中不是访客的有效成员、标签不是这个项目未删除的标签：422 `not_allowed`（`assignee_ids[i]`、`label_ids[i]`）；同一个 id 给两次：422 `duplicate`（M4 设计 3.7） |
| 默认负责人 | 指派但不关注 | 指派并关注；建工作项时现查他仍是项目中不是访客的有效成员，不是就不指派、不报错（M4 设计 3.7、3.12） |
| 负责人的动态 | 按请求里的 id 写，含被去掉的 | 只为实际指派的写（M4 设计 3.13） |
| 自动关注 | 每个产生通知的动态的执行者；被提及的人都关注 | 创建者、负责人；评论者、被提及的成员中能读这个工作项的随 M4 的 P4 加入（M4 设计 3.12） |
| 删除状态 | 有工作项时 400 | 有工作项在它里面（已归档的也算）时 409 `project.state_in_use`，这一检查在默认状态的检查之前（M4 设计 3.6） |
````
````old docs/v0/plane-diff.md
| 删除工作区 | 清掉别人的 `last_workspace_id`；成员、显示设置等的连带软删除由 Celery 异步完成 | 不清 `last_workspace_id`（登录后的落点规则让它无害，M3 设计 3.14）；工作区、成员、邀请、显示设置，项目和它们的成员关系、成员在项目里的显示设置、状态和标签，在同一个事务里、同一时刻软删除，删除之后 slug 立即可以重用 |
````
````new docs/v0/plane-diff.md
| 删除工作区 | 清掉别人的 `last_workspace_id`；成员、显示设置等的连带软删除由 Celery 异步完成 | 不清 `last_workspace_id`（登录后的落点规则让它无害，M3 设计 3.14）；工作区、成员、邀请、显示设置，项目和它们的成员关系、成员在项目里的显示设置、状态和标签，项目的工作项和它们的负责人、标签、关注、动态（M4/P1a 起），在同一个事务里、同一时刻软删除，删除之后 slug 立即可以重用 |
````

`docs/v0/v0-design.md`（修改，3 处）：

````old docs/v0/v0-design.md
- **加锁顺序**（M3 设计 3.6）：全局顺序是 `users` → `profiles` → `auth_sessions` → `api_tokens` → `workspaces` → `workspace_member_invites` → `workspace_members` → `workspace_user_properties` → `projects` → `project_members` → `project_user_properties` → `states` → `labels`，同一张表的多行按 `id` 升序。另有六条约定，后续里程碑的表照做：
````
````new docs/v0/v0-design.md
- **加锁顺序**（M3 设计 3.6；M4 设计 3.5）：全局顺序是 `users` → `profiles` → `auth_sessions` → `api_tokens` → `workspaces` → `workspace_member_invites` → `workspace_members` → `workspace_user_properties` → `projects` → `project_members` → `project_user_properties` → `states` → `labels` → `draft_issues` → `issues` → `issue_assignees` → `issue_labels` → `issue_relations` → `issue_links` → `issue_subscribers` → `issue_mentions` → `issue_comments` → `issue_reactions` → `comment_reactions` → `issue_activities` → `issue_description_versions`（M4 的表随各自的 Phase 建出），同一张表的多行按 `id` 升序。另有六条约定，后续里程碑的表照做：
````
````old docs/v0/v0-design.md
  "先锁父行，再判定"从 M3/P2 起落地（修改、删除工作区，改成员的角色，修改显示设置）：两位管理员互相降级时，先拿到工作区锁的一方成功，后到的一方判定时已是成员而被拒绝。约定六的接受邀请一段从 M3/P3 起落地：接受、忽略邀请最先以 `FOR SHARE` 锁住调用者的账户行，在锁下重读 `is_active` 和邮箱，再锁工作区（接受 `FOR NO KEY UPDATE`，忽略 `FOR SHARE`）和邀请行（`FOR UPDATE`）；接受比较的是在账户行锁下读到的邮箱，与改邮箱在账户行上串行：改邮箱先持有这一行时，接受等它提交，比较的是改后的邮箱；接受先持有时，比较的是原来的邮箱，改邮箱等接受结束（M3 设计 9.3 交错 12）。约定六的停用一段从 M3/P6 起落地：停用（自助的和服务器管理员的命令）最先以 `FOR NO KEY UPDATE` 锁住账户行，在锁下读邮箱，写完账户、新手引导和会话之后，按 `id` 锁住他有效成员关系所在的全部工作区（`FOR NO KEY UPDATE`，上锁时已删除的跳过），以一条语句按 `id` 锁住要删除的邀请（发给这个邮箱的全部邀请和他是唯一有效成员的工作区的待接受邀请，`FOR NO KEY UPDATE`：发给他邮箱的邀请多在他不锁的工作区里，两个停用不在这样锁住的邀请上互相等成环），在这之后读时刻，按锁回答的 `id` 删除这些邀请、不写别的（控制者的裁定 F-1），结束他的工作区成员行，再列举他的项目成员关系、按 `id` 一次锁住那些项目并结束它们；工作区一侧的增长在账户行上、项目一侧的增长在工作区行上与它串行（M3 设计 3.9，9.3 交错 7、8、13–16、19）。
````
````new docs/v0/v0-design.md
  "先锁父行，再判定"从 M3/P2 起落地（修改、删除工作区，改成员的角色，修改显示设置）：两位管理员互相降级时，先拿到工作区锁的一方成功，后到的一方判定时已是成员而被拒绝。约定六的接受邀请一段从 M3/P3 起落地：接受、忽略邀请最先以 `FOR SHARE` 锁住调用者的账户行，在锁下重读 `is_active` 和邮箱，再锁工作区（接受 `FOR NO KEY UPDATE`，忽略 `FOR SHARE`）和邀请行（`FOR UPDATE`）；接受比较的是在账户行锁下读到的邮箱，与改邮箱在账户行上串行：改邮箱先持有这一行时，接受等它提交，比较的是改后的邮箱；接受先持有时，比较的是原来的邮箱，改邮箱等接受结束（M3 设计 9.3 交错 12）。约定六的停用一段从 M3/P6 起落地：停用（自助的和服务器管理员的命令）最先以 `FOR NO KEY UPDATE` 锁住账户行，在锁下读邮箱，写完账户、新手引导和会话之后，按 `id` 锁住他有效成员关系所在的全部工作区（`FOR NO KEY UPDATE`，上锁时已删除的跳过），以一条语句按 `id` 锁住要删除的邀请（发给这个邮箱的全部邀请和他是唯一有效成员的工作区的待接受邀请，`FOR NO KEY UPDATE`：发给他邮箱的邀请多在他不锁的工作区里，两个停用不在这样锁住的邀请上互相等成环），在这之后读时刻，按锁回答的 `id` 删除这些邀请、不写别的（控制者的裁定 F-1），结束他的工作区成员行，再列举他的项目成员关系、按 `id` 一次锁住那些项目并结束它们；工作区一侧的增长在账户行上、项目一侧的增长在工作区行上与它串行（M3 设计 3.9，9.3 交错 7、8、13–16、19）。

  工作项一侧的写从 M4/P1a 起落地（M4 设计 3.5，决策点 1 负责人裁定 B）。约定二对它们的延伸是**最先锁项目行，不锁工作区行**：不加锁地读出资源（按项目 id 的写读项目，按资源 id 的写读资源、得到工作项和项目，没有时答资源的 404），再做一次不加锁的预检，**凡是判定拒绝的都在这里答**（看不到答资源的 404，看得到而不能做答 403），不取任何锁；放行的写开始事务，以 `FOR NO KEY UPDATE` 锁住项目行（带 `deleted_at IS NULL`），重读资源、在锁下再判定一次（等锁期间被移出、降级的人照样得到 404 或 403，锁下的判定是最终的），然后检查依赖的行、读时刻、写入。项目的 N 让同一个项目里的写一个接一个（取号、同一状态最大的 `sort_order`、动态按提交的顺序得到不减的时刻都靠它）；不锁工作区行，最频繁的写就不会像 M3 设计 17.4 的探针 B 那样让工作区一级的写一直等下去。删除工作区在锁住它的邀请之后、读时刻之前，按 `id` 升序以 `FOR NO KEY UPDATE` 锁住它的全部未删除项目（`ProjectCascade.LockWorkspaceProjects`），所以它的时刻不早于这些项目上已提交的每个写。B 的两个代价：一、授权与时刻的先后可以不一致：只改工作区角色（不降为访客）的写不锁项目，降为访客、结束成员关系的连带在锁项目之前读时刻，一个工作项的写可以在成员关系的变更读了时刻之后才写下更晚的时刻、按旧的角色生效；二、草稿的写和 M3 的项目管理类的写仍先取工作区 S，M3 设计 17.4 的饥饿在这些较少的路径上仍在。
````
````old docs/v0/v0-design.md
      issue/                工作项、评论、关联、链接、表情回应、操作动态、历史版本、草稿、归档、列表查询
````
````new docs/v0/v0-design.md
      issue/                工作项、评论、关联、链接、表情回应、操作动态、历史版本、草稿、归档、列表查询；
                            拥有以工作项为根的 13 张表（`issues`、`issue_assignees`、`issue_labels`、
                            `issue_relations`、`issue_links`、`issue_subscribers`、`issue_mentions`、
                            `issue_comments`、`issue_reactions`、`comment_reactions`、`issue_activities`、
                            `issue_description_versions`、`draft_issues`，随各自的 Phase 建出），
                            经端口读写 project、workspace、identity（M4 设计 3.3）
````

- [ ] **Step 2: 检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 `keywords: 72 rules, 23 exceptions, no hits.`

Run: `python3 $M4TMP/p1tools/doclinks.py . docs/v0/v0-design.md docs/v0/plane-diff.md docs/v0/M3-workspace-project/M3-design.md docs/v0/M4-issue-core/handoffs/M1-P2-trim-content.md docs/v0/M4-issue-core/handoffs/M1-P3-trim-platform.md docs/v0/M4-issue-core/handoffs/M3-closeout.md docs/v0/M4-issue-core/specs/P1a-issues.md docs/v0/M4-issue-core/plans/P1a-issues.md`
Expected: 每个文件 `0 broken`，最后一行 `broken: 0`。

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`（文档不影响它；本 Phase 最后一次全量运行）。

- [ ] **Step 3: 提交**

```bash
git add docs/v0/M3-workspace-project/M3-design.md docs/v0/M4-issue-core/handoffs/M1-P2-trim-content.md docs/v0/M4-issue-core/handoffs/M1-P3-trim-platform.md docs/v0/M4-issue-core/handoffs/M3-closeout.md docs/v0/plane-diff.md docs/v0/v0-design.md
```
```bash
git commit -m "docs(M4/P1): the design documents and the handoffs record P1a

The overall design's lock order reaches the work items' tables: a work
item's write prechecks, then locks its project's row first; a
workspace's deletion locks its invitations and projects before its
moment; and what choosing B costs. The M3 design's lock table, codes
and choices, the Plane diff's five tables and P1a's rows, and the three
handoffs' outcomes say what P1a did.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**：本 Task 只改文档，没有变异（文档的相对链接由 `doclinks.py` 核对）。
