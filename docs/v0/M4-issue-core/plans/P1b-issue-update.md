# M4/P1b 工作项的取、改与写闸门 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `getIssue`（`GET /api/v0/issues/{issue_id}`）和 `updateIssue`（`PATCH /api/v0/issues/{issue_id}`：名称、状态、优先级、负责人、标签）；按资源寻址的写落在 P1a 的一条取锁路径上：读工作项（不加锁）→ 预检（凡判定拒绝的 404、403 在这里答，不取任何锁）→ 写闸门（不持连接、不持锁地等）→ 事务 → 项目行 `FOR NO KEY UPDATE` → 锁下重读工作项、确认仍在这个项目 → 锁下判定（最终的）→ 依赖行的检查 → 时刻 → 写（M4 设计 3.5、6.7）；修改写工作项、负责人和标签的差集（按 `id`）、新负责人的关注、动态的第 3、6、7、10–13 行，同一时刻、同一个写者；写闸门 `platform/gate` 和 `lock_timeout` 的兜底，排满、等不到、请求到期、等锁到期都答 503 `server_busy`（`Retry-After: 1`），只在 `bootstrap/gate.go` 一处换码（11.11）。交错 3–7 的修改一支、同一个工作项的两个修改（1u）和交错 20（经真实的连接池）在真实数据库上；在规模目标上量 P1a 的同步连带；I2 的接口版本通过。

**Architecture:** `issue` 模块：`domain`（`patch.go` 修改的规则和差集，`activity.go` 的修改动态）、`app`（`lock.go` 的 `Locks.run` 是每个工作项一侧的写共用的取锁路径，`see` 是读和预检共用的判定，`writes.go` 是写共用的行，`get_issue.go`、`update_issue.go`，端口 `IssueReader`、`IssueUpdater`、`WriteGate`）、`adapter/postgres`（`update.sql`）、`adapter/http`（`getIssue`、`updateIssue`）。`project`：状态、标签的目录回答名称。`platform/gate`（新，只用标准库）；`platform/postgres` 的 `TxManager` 选项 `LockTimeout` 和 `LockNotAvailable`；`platform/config` 的 `issue` 四个配置。`bootstrap`：`gate.go`（闸门和带锁的期限的事务，503 的唯一一处）、矩阵、取锁形状、盖戳、交错 1u 和 3u–7u、交错 20、规模的测量。`e2e`：`updateIssue`、`expectIssueUpdated`、I2。不加 Go 模块、npm 包；没有新表；模块之间只经端口。

**Tech Stack:** Go 1.27.1、pgx v5.11.0、sqlc v1.31.1（`CGO_ENABLED=0`）、oapi-codegen v2.8.0、goose v3.28.0、River v0.47.0、golangci-lint 2.13.2、PostgreSQL 18.6（testcontainers）；Node 24、pnpm 11.10.0、Playwright 1.63.0、TypeScript 5.8.3、oxlint 1.51.0、oxfmt 0.35.0。

**Spec:** `docs/v0/M4-issue-core/specs/P1b-issue-update.md`（上级：`docs/v0/M4-issue-core/M4-design.md`）

## Global Constraints

- **基点**：本 plan 和 spec 在 `5ce30a1a`（P1a 的合并提交）之上的一个提交里，M4 设计第 15 节 P1b 一行在同一个提交里改为"进行中"；Task 1 从这个提交开始。
- **Go 版本和依赖**：本 plan 不执行 `go get`，不加任何 Go 模块或 npm 包（`platform/gate` 只用标准库）。`server/go.mod` 和 `server/tools/go.mod` 保持 `go 1.27` 和 `toolchain go1.27.1`；`git diff --stat 5ce30a1a8974346485aa711a08f4848c3c7f2986 -- server/go.mod server/go.sum server/tools pnpm-lock.yaml` 在本 plan 的任何时刻都没有输出。
- **每个改了 `server/` 的 Task（Task 1–13）提交前**：`make lint-go` 两段都输出 `0 issues.`；除 Task 8 之外 `make test` 全部通过（没有 `FAIL`），Task 8 只加还没有使用者的 `platform/gate`，跑它自己的测试（含 `-race`）和 `archtest`。Task 14 只改 `e2e/`，跑 Web 的三个门禁和 `make e2e`；Task 15 只改文档，跑 `make lint-web`、`doclinks.py` 和本 Phase 最后一次全量的 `make test`。
- **生成的文件不手写、不从本 plan 复制**：有生成物的 Task 执行 Task 中的生成命令（只动了 sqlc 的 Task 2、3 执行 `make gen-go`，改了接口描述的 Task 4、6、10 执行 `make gen`），用 `shasum -a 256` 和 `wc -l` 核对 Task 末尾表中的 SHA-256 和行数，对不上时停下来：说明某个输入与本 plan 不一致。这些 Task **提交之后**执行 `make gen-check`（它用 `git status` 判断生成物是否已提交，提交之前必然报差异）。
- **前端检查和新的错误码**（M4 设计 12 节约束 4）：声明新码的 Task 在同一个 Task 里把它的文案加进 `web/apps/web/core/lib/error-messages.ts`（`PROBLEM_MESSAGES`）和两份 `errors.json`（`web/packages/i18n/src/locales/en/errors.json`、`zh-CN/errors.json`）：Task 4 `issue.not_found`。`server_busy` 是 M2 的平台码，三处已有它的文案，Task 10 不另加。改了接口描述、前端或 `e2e/` 的 Task 4、6、10、14 另执行 `make lint-web`、`make knip`、`make test-web`；改了故事经过的写的 Task 5（`createIssue` 改到 `Locks.run` 上）、Task 10（写闸门和锁的期限）和 Task 14 执行 `make e2e`；Task 15 只改文档，执行 `make lint-web` 和 `doclinks.py`（下面"一次性的脚本"）。
- **e2e 的数目**：在 worktree 中 `make e2e` 必须全部通过：开始时 121 个，Task 14 起 122 个（I2）。只在不是 git 仓库的副本里，S3 因为构建读不到提交而失败（M3/P4b spec F4）；没有装 Firefox 的机器上 A4 的两个 firefox 测试因为浏览器不在而失败；两者都与本 plan 无关。
- **容器**：`make test` 和 `make e2e` 用自己的 testcontainers；机器忙时偶尔起不来，等 Docker 空闲之后重跑一次再当作失败（容器起不来不算任何检查的失败）。容器测试一次只跑一套。开发库 `nerve-dev-db-1` 不要启动、停止或重建，不要执行 `make dev-db-down`、`make dev-db-reset`。不要碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`、`nervewiki-*`）和留下的 testcontainers。
- **git**：每次 Bash 调用只执行一个 git 命令，不用 `;`、`&&`、`|` 串联 git；不用 `git -C`、`stash`、`clean`、`reset --hard`。不用 `cd`。不碰 `plane/`、`refer/`。
- **安装**：除了 Docker、Go、Node 不做任何全局安装；不执行 `corepack enable`（pnpm 已在 PATH 上）。不把副本的 `node_modules` 链接到 worktree 的。
- **规则**：不写 `init()`，不用全局可变状态，构造函数显式传入依赖；不建 `utils`、`common`、`helpers` 包；一个文件只做一件事，不超过约 400 行；依赖只能向内；模块之间不互相导入，不跨模块的表 JOIN；模块的 SQL 只经 sqlc（`SET LOCAL lock_timeout` 由 `platform/postgres` 发，模块的 SQL 里没有 `SET`）；端口由使用方声明；角色只按集合判断，不按大小比较；不留没有使用者的代码。工作项一侧的每个写经 `issue/app/lock.go` 的 `Locks.run`（一条共用的取锁路径），不写近似的副本；测试的帮手共用，不复制（`issue_world_test.go`、`issue_interleaving_test.go`、`serializeOn`、`holding`、`assert/issue.ts`）。**例外**：接口描述按模块一个文件（M0-P3 交接 5），`api/modules/issue.yaml`（307 行）不受约 400 行的限制。本 plan 改的其余代码文件都在约 400 行以内（最终原型上量的）：最长的是 `platform/config/load_test.go`（414 行：基点已是 403 行，它的输入和期望是整份配置，Task 9 在两处加 `issue` 的四个键）、`bootstrap/permission_matrix_test.go`（399 行）、`bootstrap/issue_interleaving_test.go`（377 行）、`bootstrap/ports_test.go`（376 行）；新建的文件中最长的是 `issue/app/update_issue_test.go`（322 行）、`platform/gate/gate_test.go`（311 行）、`e2e/stories/issue/i2-update-issue.spec.ts`（303 行）、`bootstrap/gate_test.go`（278 行）、`bootstrap/issue_pool_test.go`（266 行）。
- **关键词守卫**：72 条规则、23 个例外，没有命中。本 plan 不改 `tools/keywords.json`：`phase` 照旧是 `M4/P1`（P1a、P1b 都写 `M4/P1`，裁定 5）。
- **注释**：Go、TS 代码、SQL 查询和接口描述用英文；配置文件的注释和中文文档照本 plan 原样。每条注释和说明在它所在的 Task 结束时都与代码相符。
- **代码块**：每个改动都写成四个反引号围起来的块，块的第一行写明种类和路径，照原样使用（原型中逐字节运行过）：
  - ````` ````file <路径> ````` 新文件，块的内容加一个结尾换行就是整个文件；
  - ````` ````whole <路径> ````` 已有文件的完整新内容（同样加结尾换行）；
  - ````` ````old <路径> ````` 与紧跟着的 ````` ````new <路径> `````：`old` 的文字在文件当前版本中恰好出现一次，把它换成 `new` 的文字；
  - ````` ````delete <路径> ````` 删除这个文件（块是空的）。

  一个文件的几个块按出现的顺序依次应用。拼 plan 的脚本已从基点起按顺序核对过全部块：每个 `old` 恰好出现一次（在它之前的块应用之后的文件中），每个新文件原来不存在，逐 Task 应用之后的文件与原型逐字节相同；之后在另一个副本上从基点逐 Task 复现，每个 Task 之后跑它的门禁，最后的树与原型逐个文件相同（spec 附录 A）。块写入之后不必再格式化（原型中每个 Task 都经过 `gofmt`、`oxfmt`）。
- **过渡版本**：一些文件先在较早的 Task 写成过渡版本，较晚的 Task 再修改（文件结构一节的"Task"一列列出每个文件的全部 Task）：`issue/app/lock.go`（Task 4 加 `see`，Task 5 整个文件，Task 10 加闸门）、`issue/module.go`（Task 2、4、5、6、10）、`issue/app/ports.go`（Task 2、4、5、10）、`create_issue.go`（Task 2、5、10）、`update_issue.go`（Task 5、10）、`create_issue_test.go`、`fakes_test.go`（Task 2、4、5、10）、`api/modules/issue.yaml`（Task 4、6、10）、`issue_world_test.go`（Task 6、7、10、11）、`issue_writes_test.go`（Task 6、7、10）、`issue_updates_test.go`（Task 7、11）、`rules.go`（Task 4、5）。每个过渡版本都在逐 Task 复现中运行过。
- **变异**：每个 Task 末尾的"变异"表列出：把代码改坏的方式、必须因此失败的检查和它所在的层（单元：假实现；存储：真实数据库；组合：`bootstrap` 组合出的 app；端到端：单独运行的故事；静态：`make lint-web` 的文案核对）。它们在各自 Task 的树上跑过，标"（Task n 起）"的检查在 Task n 的树上跑过，全部在最终的原型上再跑过，每个变异在一份新的副本里（`$M4TMP/p1btools/mutants_p1b.py`，`mut.py` 在它写的每个检查上各跑一次；端到端的"发现"必须是故事的断言失败，`e2ekills.py` 核对，不算构建失败或超时；容器起不来不算；spec 附录 A）。"（Task n 起）"的检查从那个 Task 起才发现这个变异：Task n 的实现者在 Task n 的树上重跑它（那个 Task 的表下面列出）。**安全或加锁的性质只由单元一层发现的，算缺口**：表中每一条这类性质都另有存储、组合或端到端一层的检查，例外（"按性质只在单元一层"）写在 spec 附录 A.3。
- **评审敏感**（M4 设计 12 节约束 3）：写闸门（取连接之前进入，不持连接地等，不在持数据库的锁时再等别的闸门；每个项目至多 K 个、`max_wait`、`max_queue`、请求的期限，503 `server_busy` 带 `Retry-After: 1`；每条路上离开）；`55P03` 换成 503 只在 `bootstrap/gate.go`；按资源寻址的写在锁下重读、再判定；修改的批量写按约定五（spec 第 3 节第 12 条）。改动这些代码或测试之前，先照"变异"表确认它在所说的性质去掉之后失败。
- **提交**：提交信息用英文，末尾加一行：`Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明。

## 一次性的脚本

`$M4TMP/p1btools/doclinks.py`（Task 15 的文档核对：给出的 markdown 文件中每个相对链接都指向仓库中存在的文件或目录；代码块和行内代码里的不算链接）：

```python
# M4/P1b plan, Task 15 (from M4/P1a's, from M3's closeout plan): every relative link of the markdown files given resolves, from the
# file's folder, to a file or folder of the repository (a link's #anchor is not checked). Code is not prose: a fenced
# block's lines and an inline code span hold no links (a Go generic call, New[K](l), reads like one). Prints one line
# per file and each link that does not resolve; exits 1 when one does not. usage: python3 doclinks.py <root> <markdown file>...
import os
import re
import sys

root = sys.argv[1]
LINK = re.compile(r'\]\(([^)\s]+)\)')
FENCE = re.compile(r'^(`{3,})')
SPAN = re.compile(r'(`+)(?:(?!\1).)+?\1')


def prose(text):
    # the text outside fenced code blocks, inline code spans removed
    kept, fence = [], ''
    for line in text.split('\n'):
        run = FENCE.match(line)
        if fence:
            if run and len(run.group(1)) >= len(fence) and not line[len(run.group(1)):].strip():
                fence = ''
            continue
        if run:
            fence = run.group(1)
            continue
        kept.append(SPAN.sub('', line))
    return '\n'.join(kept)


broken = 0
for rel in sys.argv[2:]:
    text = prose(open(os.path.join(root, rel), encoding='utf-8').read())
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
| `server/internal/modules/issue/domain/patch.go`（新）、`server/internal/modules/issue/domain/patch_test.go`（新） | `IssuePatch`、`CheckIssuePatch`、`Diff`、`Changes`、`ChangesOf`、`None`（Task 1）；`Unassignable`（Task 5） | 1、5 |
| `server/internal/modules/issue/domain/activity.go`、`server/internal/modules/issue/domain/activity_test.go`（修改） | 动态的第 3、6、7、10–13 行：`Named`、`Changed`、`Update`、`UpdateActivity` | 1 |
| `server/internal/modules/project/adapter/postgres/queries/issue_ports.sql`、`server/internal/modules/project/adapter/postgres/issue_ports.go`、`server/internal/modules/project/adapter/postgres/issue_ports_test.go`（修改） | 状态、标签的目录回答名称 | 2 |
| `server/internal/modules/project/app/issue_ports.go`、`server/internal/modules/project/module.go`（修改） | `IssueState.Name`、`IssueLabel` | 2 |
| `server/internal/bootstrap/ports.go`、`server/internal/bootstrap/ports_test.go`（修改） | 转换状态、标签的名称 | 2 |
| `server/internal/modules/issue/app/ports.go`（修改） | `StateRef.Name`、`LabelRef`（Task 2）；`IssueReader`（Task 4）；`IssueUpdater`（Task 5）；`WriteGate`（Task 10） | 2、4、5、10 |
| `server/internal/modules/issue/app/create_issue.go`（修改） | 按 `id` 找标签（Task 2）；改到 `Locks.run` 上，`stamp`、`readBack` 和 `pickState` 移到 `writes.go`（Task 5）；`Execute` 的说明写上写闸门（Task 10） | 2、5、10 |
| `server/internal/modules/issue/app/create_issue_test.go`、`server/internal/modules/issue/app/fakes_test.go`（修改） | 标签的目录（Task 2）；`Locks` 的新参数（Task 4、5、10）；修改要的假实现、创建和修改共用的拒绝和端口失败的检查（Task 5）；`fakeGate`，事务没有结束时的离开（Task 10） | 2、4、5、10 |
| `server/internal/modules/issue/module.go`（修改） | `LabelRef`（Task 2）；`GetIssue`（Task 4）；`Tx` 交给 `Locks`（Task 5）；`UpdateIssue`、包的说明（Task 6）；`WriteGate`（Task 10） | 2、4、5、6、10 |
| `server/internal/modules/issue/adapter/postgres/queries/update.sql`、`server/internal/modules/issue/adapter/postgres/update.go`、`server/internal/modules/issue/adapter/postgres/update_test.go`（新） | `UpdateIssue`、`RemoveAssignees`、`RemoveLabels` 和存储测试 | 3 |
| `server/internal/modules/issue/app/rows.go`（修改） | `IssueFields`、`UnlinkRows` | 3 |
| `server/internal/modules/issue/domain/errors.go`、`server/internal/modules/issue/domain/actions.go`（修改） | `ErrIssueNotFound`、`issue.read`（Task 4）；`issue.update`（Task 5） | 4、5 |
| `server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`（修改） | `issue.read`（Task 4）、`issue.update`（Task 5）的规则行 | 4、5 |
| `server/internal/modules/issue/app/lock.go`（修改） | `see`、`decide` 的 `notFound`（Task 4）；整个文件：`run`、按资源寻址的 `write`、`held`（Task 5）；写闸门（Task 10） | 4、5、10 |
| `server/internal/modules/issue/app/get_issue.go`（新）、`server/internal/modules/issue/app/get_issue_test.go`（新） | `getIssue` 的用例和测试（Task 4）；测试用 `Locks.run` 之后的假实现（Task 5）；测试的写闸门记下进出：读不进闸门（Task 10） | 4、5、10 |
| `api/modules/issue.yaml`（修改） | `getIssue`、`IssueID`（Task 4）；`updateIssue`、`IssueUpdate`（Task 6）；`server_busy`（Task 10） | 4、6、10 |
| `api/openapi.yaml`（修改） | `getIssue`、`updateIssue` 的路径 | 4 |
| `server/internal/modules/issue/adapter/http/handler.go`、`server/internal/modules/issue/adapter/http/issues.go`、`server/internal/modules/issue/adapter/http/handler_test.go`（修改） | `GetIssue`（Task 4）、`UpdateIssue`（Task 6） | 4、6 |
| `server/internal/modules/issue/adapter/http/issues_test.go`（修改） | `getIssue` 的测试（Task 4）；`server_busy` 的一行（Task 10） | 4、10 |
| `server/internal/modules/issue/adapter/http/update_test.go`（新） | `updateIssue` 的 HTTP 测试（Task 6）；`server_busy` 的一行（Task 10） | 6、10 |
| `web/apps/web/core/lib/error-messages.ts`、`web/packages/i18n/src/locales/en/errors.json`、`web/packages/i18n/src/locales/zh-CN/errors.json`（修改） | `issue.not_found` 的文案（约束 4） | 4 |
| `server/internal/bootstrap/permission_matrix_seed_issues_test.go`（新）、`server/internal/bootstrap/permission_matrix_seeded_test.go`、`server/internal/bootstrap/permission_matrix_targets_test.go`、`server/internal/bootstrap/permission_matrix_project_test.go`（修改） | 矩阵的种子：每个项目一个工作项，`{issue_id}`；`rowsByID` 带上工作项 | 4 |
| `server/internal/bootstrap/permission_matrix_issues_test.go`、`server/internal/bootstrap/permission_matrix_test.go`（修改） | `getIssue` 一行，`createIssue` 的编号（Task 4）；接上修改的行（Task 6） | 4、6 |
| `server/internal/modules/issue/app/writes.go`（新） | 每个写共用的 `pickState`、`stamp`、`assignees`、`readBack` | 5 |
| `server/internal/modules/issue/app/update_issue.go`（新） | `updateIssue` 的用例（Task 5）；`Execute` 的说明写上写闸门（Task 10） | 5、10 |
| `server/internal/modules/issue/app/update_issue_test.go`（新） | `updateIssue` 的单元测试 | 5 |
| `server/internal/modules/issue/domain/issue.go`（修改） | `unassignable` 与 `Assignees` 共用 | 5 |
| `server/internal/bootstrap/permission_matrix_issue_updates_test.go`（新） | `updateIssue` 的九行 | 6 |
| `server/internal/bootstrap/issue_world_test.go`（修改） | Done、`newWorkItem`（Task 6）；`workItemStamps`（Task 7）；`newIssueWorldWith`（Task 10）；`newWorkItemIn`、`activityBy`（Task 11） | 6、7、10、11 |
| `server/internal/bootstrap/issue_write_locks_test.go`（修改） | `issueWrites` 加 `updateIssue` 的一行；每行发的路径是它的操作的（`pathOf`） | 6 |
| `server/internal/bootstrap/issue_updates_test.go`（新） | 盖戳、回答如存（Task 7）；修改的盖戳测试用 `newWorkItemIn` 建它的工作项、用 `activityBy` 读修改者的动态（Task 11） | 7、11 |
| `server/internal/bootstrap/issue_writes_test.go`（修改） | 创建的盖戳测试用 `newWorkItem`（Task 6）、`workItemStamps`（Task 7）；事务的连接加修改（Task 7）；`newIssueRoute` 的闸门和配置（Task 10） | 6、7、10 |
| `server/internal/platform/gate/gate.go`、`server/internal/platform/gate/gate_test.go`（新） | `Keyed[K]` 和它的测试 | 8 |
| `server/internal/platform/postgres/tx.go`（修改）、`server/internal/platform/postgres/lock_timeout_test.go`（新） | `LockTimeout`、`LockNotAvailable` | 9 |
| `server/internal/platform/config/config.go`、`server/internal/platform/config/validate.go`、`server/internal/platform/config/config_test.go`、`server/internal/platform/config/load_test.go`、`server/internal/platform/config/validate_test.go`（修改） | `issue` 的四个配置、校验、日志 | 9 |
| `server/configs/config.yaml`、`server/configs/embed_test.go`（修改） | 四个配置的默认值和说明 | 9 |
| `server/internal/modules/issue/app/lock_test.go`（新） | 闸门在预检和事务之间、每条路上离开 | 10 |
| `server/internal/bootstrap/gate.go`、`server/internal/bootstrap/gate_test.go`（新） | `writeGate`、`busyTx`：503 `server_busy` 的唯一一处 | 10 |
| `server/internal/bootstrap/app.go`、`server/internal/bootstrap/app_test.go`（修改） | 接上闸门和带锁的期限的事务；`testIssueConfig` | 10 |
| `server/internal/archtest/composition_test.go`（修改） | 命令行不建写闸门 | 10 |
| `server/internal/bootstrap/issue_interleaving_test.go`、`server/internal/bootstrap/issue_interleaving_deletion_test.go`、`server/internal/bootstrap/issue_interleaving_catalog_test.go`（修改） | 交错 3u–7u：`itemWrite`、`itemWrites` | 11 |
| `server/internal/bootstrap/issue_interleaving_change_test.go`（新） | 交错 1 的修改一支（1u）：同一个工作项的两个修改；等锁期间工作项被删除或移走 | 11 |
| `server/internal/bootstrap/interleaving_writes_test.go`、`server/internal/bootstrap/interleaving_test.go`（修改） | `TestTwoWritesOnAProjectSerialize` 改用 `serializeOn`；`serializeOn` 的说明不再说第一个写在它的写之后等 | 11 |
| `server/internal/bootstrap/issue_pool_test.go`（新） | 交错 20 | 12 |
| `server/internal/bootstrap/issue_scale_test.go`（新） | 规模的测量 | 13 |
| `e2e/fixtures/issues.ts`、`e2e/fixtures/assert/issue.ts`（修改） | `updateIssue`、`expectIssueUpdated` | 14 |
| `e2e/stories/issue/i1-create-issue.spec.ts`（修改）、`e2e/stories/issue/i2-update-issue.spec.ts`（新） | I1 的动态行加 `was`；I2 的接口版本 | 14 |
| `docs/v0/v0-design.md`、`docs/v0/plane-diff.md`、`README.md`、`docs/v0/M4-issue-core/handoffs/M2-closeout.md`、`docs/v0/M4-issue-core/handoffs/M3-closeout.md`（修改） | 3.23、8.8 中 P1b 的行；交接的处理结果 | 15 |

---

### Task 1: 修改一侧的规则和动态的第 3、6、7、10–13 行

**Files:**
- Create: `server/internal/modules/issue/domain/patch.go`、`server/internal/modules/issue/domain/patch_test.go`
- Modify: `server/internal/modules/issue/domain/activity.go`、`server/internal/modules/issue/domain/activity_test.go`

**Interfaces:**（spec 2.3；M4 设计 3.7、3.13）
- `issue/domain/patch.go`（新）：
  - `IssuePatch{Name *string, StateID *uuid.UUID, Priority *Priority, AssigneeIDs *[]uuid.UUID, LabelIDs *[]uuid.UUID}`：`nil` 是没给，保持原值；给出的负责人、标签是新的集合，空的就是没有；
  - `CheckIssuePatch(p IssuePatch) (IssuePatch, error)`：与 `CheckNewIssue` 同一组规则（`checkName`、`checkPriority`、`checkIDs`），全部问题一次报出（422 `validation_failed`）；回答去掉空白的名称；
  - `Diff{Added, Removed}`、`diffOf(from, to)`：加的按 `id` 排序，去的按工作项的次序（`id` 的次序）；
  - `Changes{Name, Priority, StateID, Assignees, Labels}`、`ChangesOf(i Issue, p IssuePatch) Changes`：名称、优先级、状态与工作项的不同时才算改动；`(Changes) None() bool`。
- `issue/domain/activity.go`：`Named{ID, Name}`、`Changed[T]{Old, New}`、`Update{Name, Priority, State, LabelsAdded, LabelsRemoved, AssigneesAdded, AssigneesRemoved}`、`UpdateActivity(u Update) []Activity`（3.13 的表的次序，Plane 的 `comment`）；`Assignee` 的说明加"去掉的"。

**Tests:**（单元）
- `TestCheckIssuePatchAccepts`：什么都不给、一个和 255 个字符的名称、每个优先级、各 100 个负责人和标签、两个空的集合：接受，名称去掉空白，空的集合仍是给出的。
- `TestCheckIssuePatchReportsEveryProblem`：空白的名称、256 个字符、带 NUL、五个之外的优先级（大小写不对、空串）、给两次的账户或标签、各 101 个：每个都拒绝，一次报出，按字段和下标。
- `TestChangesOf`：没给或给出原样的字段不算改动；给出的集合与工作项的差集，加、去各按 `id`，不论给出的次序。
- `TestChangesNone`：每一项单独都算改动。
- `TestUpdateActivity`：每个改动一行，3.13 的次序，名称和 `id`，Plane 的句子；没改的不写。

- [ ] **Step 1: 修改的规则**

`server/internal/modules/issue/domain/patch.go`（新文件，119 行）：

````file server/internal/modules/issue/domain/patch.go
package domain

import (
	"slices"
	"strings"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// IssuePatch is what the caller asks to change in a work item (M4 design
// 5.2, IssueUpdate without the fields of later phases): each field nil
// when it is not given, and keeps its value. AssigneeIDs and LabelIDs,
// given, are the work item's new sets of accounts and labels; given empty,
// none.
type IssuePatch struct {
	Name        *string
	StateID     *uuid.UUID
	Priority    *Priority
	AssigneeIDs *[]uuid.UUID
	LabelIDs    *[]uuid.UUID
}

// CheckIssuePatch checks the fields p gives by createIssue's rules (M4
// design 3.7, CheckNewIssue) and returns p as it is stored: the name
// without its surrounding white space, of 1–255 characters and without
// NUL; the priority one of the five (given empty, it is refused, not left
// as it is); at most MaxAssignees accounts and MaxLabels labels, each
// named once. Every problem is reported at once, in one 422
// validation_failed; whether the state, each account and each label the
// change adds may be is the use case's, under the project's lock.
func CheckIssuePatch(p IssuePatch) (IssuePatch, error) {
	var found []*shared.FieldError
	if p.Name != nil {
		name := strings.TrimSpace(*p.Name)
		p.Name = &name
		found = append(found, checkName(name))
	}
	if p.Priority != nil {
		found = append(found, checkPriority(*p.Priority))
	}
	if p.AssigneeIDs != nil {
		found = append(found, checkIDs("assignee_ids", *p.AssigneeIDs, MaxAssignees)...)
	}
	if p.LabelIDs != nil {
		found = append(found, checkIDs("label_ids", *p.LabelIDs, MaxLabels)...)
	}
	if err := invalid(found...); err != nil {
		return IssuePatch{}, err
	}
	return p, nil
}

// Diff is how a set of ids becomes another: the ids it adds and those it
// removes, each in id order, nil for none.
type Diff struct {
	Added   []uuid.UUID
	Removed []uuid.UUID
}

// diffOf is the change of the set from, a work item's, in id order, into
// the set to, given in any order: the ids of to that from has not, and
// those of from that to has not.
func diffOf(from, to []uuid.UUID) Diff {
	var d Diff
	for _, id := range slices.SortedFunc(slices.Values(to), uuid.UUID.Compare) {
		if !slices.Contains(from, id) {
			d.Added = append(d.Added, id)
		}
	}
	for _, id := range from {
		if !slices.Contains(to, id) {
			d.Removed = append(d.Removed, id)
		}
	}
	return d
}

// Changes is what a checked patch changes in a work item (M4 design 3.7):
// the new name, priority and state where each differs from the work
// item's, nil where it is not given or is the same; and the accounts and
// the labels it adds and removes (Diff), where the patch gives the new
// set. An account or a label the work item has and the new set keeps is
// left as it is: its row is not written again.
type Changes struct {
	Name      *string
	Priority  *Priority
	StateID   *uuid.UUID
	Assignees Diff
	Labels    Diff
}

// ChangesOf is what p, checked (CheckIssuePatch), changes in i.
func ChangesOf(i Issue, p IssuePatch) Changes {
	var c Changes
	if p.Name != nil && *p.Name != i.Name {
		c.Name = p.Name
	}
	if p.Priority != nil && *p.Priority != i.Priority {
		c.Priority = p.Priority
	}
	if p.StateID != nil && *p.StateID != i.StateID {
		c.StateID = p.StateID
	}
	if p.AssigneeIDs != nil {
		c.Assignees = diffOf(i.AssigneeIDs, *p.AssigneeIDs)
	}
	if p.LabelIDs != nil {
		c.Labels = diffOf(i.LabelIDs, *p.LabelIDs)
	}
	return c
}

// None reports whether c changes nothing: then the update writes no row
// (M4/P1b spec 3).
func (c Changes) None() bool {
	return c.Name == nil && c.Priority == nil && c.StateID == nil && len(c.Assignees.Added) == 0 && len(c.Assignees.Removed) == 0 &&
		len(c.Labels.Added) == 0 && len(c.Labels.Removed) == 0
}
````

`server/internal/modules/issue/domain/patch_test.go`（新文件，143 行）：

````file server/internal/modules/issue/domain/patch_test.go
package domain

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Nothing given, a name of one character and of 255, each priority, a
// hundred accounts and labels, and the two sets given empty: accepted, and
// returned as stored: the name without its surrounding white space, a set
// given empty still given (it empties the work item's), a field not given
// still not given.
func TestCheckIssuePatchAccepts(t *testing.T) {
	state, none := uuid.NewV7(), []uuid.UUID{}
	many, labels := ids(MaxAssignees), ids(MaxLabels)
	tests := []struct {
		name string
		in   IssuePatch
		want IssuePatch
	}{
		{"nothing", IssuePatch{}, IssuePatch{}},
		{"one character, trimmed", IssuePatch{Name: ptr("  x\t")}, IssuePatch{Name: ptr("x")}},
		{"255 characters", IssuePatch{Name: ptr(strings.Repeat("é", 255))}, IssuePatch{Name: ptr(strings.Repeat("é", 255))}},
		{"everything", IssuePatch{Name: ptr("Fix"), StateID: &state, Priority: ptr(PriorityUrgent), AssigneeIDs: &many, LabelIDs: &labels},
			IssuePatch{Name: ptr("Fix"), StateID: &state, Priority: ptr(PriorityUrgent), AssigneeIDs: &many, LabelIDs: &labels}},
		{"both sets empty", IssuePatch{AssigneeIDs: &none, LabelIDs: &none}, IssuePatch{AssigneeIDs: &none, LabelIDs: &none}},
	}
	for _, p := range priorities {
		tests = append(tests, struct {
			name string
			in   IssuePatch
			want IssuePatch
		}{string(p), IssuePatch{Priority: ptr(p)}, IssuePatch{Priority: ptr(p)}})
	}
	for _, tt := range tests {
		if got, err := CheckIssuePatch(tt.in); err != nil || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: CheckIssuePatch() = %+v, %v; want %+v", tt.name, got, err, tt.want)
		}
	}
}

// A blank name, one of 256 characters, one with NUL, a priority outside the
// five, in another case or given empty, an account or a label named twice,
// a hundred and one of either: each refused, all of them in one 422 that
// names each by its field and place, as CheckNewIssue does (M4 design 3.7).
func TestCheckIssuePatchReportsEveryProblem(t *testing.T) {
	a, b := uuid.NewV7(), uuid.NewV7()
	repeats, twice := []uuid.UUID{a, b, a}, []uuid.UUID{b, b}
	many := append(ids(MaxAssignees-1), a, a)
	priority := shared.FieldError{Field: "priority", Code: "invalid_format", Message: "must be one of urgent, high, medium, low, none"}
	tests := []struct {
		name string
		in   IssuePatch
		want []shared.FieldError
	}{
		{"blank", IssuePatch{Name: ptr(" \n ")}, []shared.FieldError{{Field: "name", Code: "too_short", Message: "must not be empty"}}},
		{"256 characters", IssuePatch{Name: ptr(strings.Repeat("x", 256))},
			[]shared.FieldError{{Field: "name", Code: "too_long", Message: "must be at most 255 characters"}}},
		{"NUL", IssuePatch{Name: ptr("a\x00b")}, []shared.FieldError{{Field: "name", Code: "invalid_format", Message: "must not contain a NUL character"}}},
		{"another case", IssuePatch{Priority: ptr[Priority]("Urgent")}, []shared.FieldError{priority}},
		{"given empty", IssuePatch{Priority: ptr[Priority]("")}, []shared.FieldError{priority}},
		{"repeats", IssuePatch{AssigneeIDs: &repeats, LabelIDs: &twice}, []shared.FieldError{
			{Field: "assignee_ids[2]", Code: "duplicate", Message: "is listed before"},
			{Field: "label_ids[1]", Code: "duplicate", Message: "is listed before"},
		}},
		{"too many, and everything else", IssuePatch{Name: ptr(""), Priority: ptr[Priority]("high!"), AssigneeIDs: &many, LabelIDs: &many},
			[]shared.FieldError{
				{Field: "name", Code: "too_short", Message: "must not be empty"},
				priority,
				{Field: "assignee_ids", Code: "too_long", Message: "must name at most 100"},
				{Field: "label_ids", Code: "too_long", Message: "must name at most 100"},
			}},
	}
	for _, tt := range tests {
		var e *shared.Error
		if _, err := CheckIssuePatch(tt.in); !errors.As(err, &e) || e.Code != "validation_failed" || !reflect.DeepEqual(e.Fields, tt.want) {
			t.Errorf("%s: CheckIssuePatch() = %v; want validation_failed with %+v", tt.name, err, tt.want)
		}
	}
}

// What a patch changes in a work item, Fix in Todo, high, assigned to ann
// and ben, labelled Bug and UI (M4 design 3.7): a field not given, or given
// as it is, changes nothing; a set given is the new set, whose difference
// with the work item's is added and removed, each in id order, whatever the
// order given; a set not given leaves the work item's as it is, its
// accounts and labels too.
func TestChangesOf(t *testing.T) {
	// In id order: ann, ben, cat, dan; bug, ui, ux.
	people, tags := ids(4), ids(3)
	ann, ben, cat, dan := people[0], people[1], people[2], people[3]
	bug, ui, ux := tags[0], tags[1], tags[2]
	todo, done := uuid.NewV7(), uuid.NewV7()
	i := Issue{Name: "Fix", StateID: todo, Priority: PriorityHigh, AssigneeIDs: []uuid.UUID{ann, ben}, LabelIDs: []uuid.UUID{bug, ui}}
	set := func(ids ...uuid.UUID) *[]uuid.UUID { return &ids }
	tests := []struct {
		name string
		p    IssuePatch
		want Changes
	}{
		{"nothing given", IssuePatch{}, Changes{}},
		{"each given as it is", IssuePatch{Name: ptr("Fix"), StateID: &todo, Priority: ptr(PriorityHigh), AssigneeIDs: set(ben, ann),
			LabelIDs: set(ui, bug)}, Changes{}},
		{"the name", IssuePatch{Name: ptr("Mend")}, Changes{Name: ptr("Mend")}},
		{"the priority", IssuePatch{Priority: ptr(PriorityNone)}, Changes{Priority: ptr(PriorityNone)}},
		{"the state", IssuePatch{StateID: &done}, Changes{StateID: &done}},
		{"accounts added and removed, out of order", IssuePatch{AssigneeIDs: set(dan, ben, cat)},
			Changes{Assignees: Diff{Added: []uuid.UUID{cat, dan}, Removed: []uuid.UUID{ann}}}},
		{"every account removed", IssuePatch{AssigneeIDs: set()}, Changes{Assignees: Diff{Removed: []uuid.UUID{ann, ben}}}},
		{"labels added and removed, out of order", IssuePatch{LabelIDs: set(ux, bug)},
			Changes{Labels: Diff{Added: []uuid.UUID{ux}, Removed: []uuid.UUID{ui}}}},
		{"every label removed", IssuePatch{LabelIDs: set()}, Changes{Labels: Diff{Removed: []uuid.UUID{bug, ui}}}},
	}
	for _, tt := range tests {
		if got := ChangesOf(i, tt.p); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: ChangesOf() = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

// Changes change nothing only when each of them is empty: any one of them
// alone is a change.
func TestChangesNone(t *testing.T) {
	id := uuid.NewV7()
	one := []uuid.UUID{id}
	if !(Changes{}).None() {
		t.Error("Changes{}.None() = false, want true")
	}
	for name, c := range map[string]Changes{
		"the name": {Name: ptr("x")}, "the priority": {Priority: ptr(PriorityLow)}, "the state": {StateID: &id},
		"an account added": {Assignees: Diff{Added: one}}, "an account removed": {Assignees: Diff{Removed: one}},
		"a label added": {Labels: Diff{Added: one}}, "a label removed": {Labels: Diff{Removed: one}},
	} {
		if c.None() {
			t.Errorf("%s: None() = true, want false", name)
		}
	}
}
````

- [ ] **Step 2: 修改的动态**

`server/internal/modules/issue/domain/activity.go`（修改，3 处）：

````old server/internal/modules/issue/domain/activity.go
// Assignee is an account a work item is assigned to, as its activity names
// him: his id and his display name when the row is written.
type Assignee struct {
	ID          uuid.UUID
	DisplayName string
}
````
````new server/internal/modules/issue/domain/activity.go
// Assignee is an account a work item is assigned to, or unassigned from,
// as its activity names him: his id and his display name when the row is
// written.
type Assignee struct {
	ID          uuid.UUID
	DisplayName string
}

// assigneeAdded is the row of a, an account a work item is assigned to:
// row 2 at its creation and row 12 at an update, which M4 design 3.13
// gives the same columns: assignees, updated, from "" to his display name,
// his id the new identifier.
func assigneeAdded(a Assignee) Activity {
	return Activity{Field: ptr("assignees"), Verb: VerbUpdated, OldValue: ptr(""), NewValue: ptr(a.DisplayName),
		NewIdentifier: ptr(a.ID), Comment: "added assignee "}
}
````
````old server/internal/modules/issue/domain/activity.go
	for _, a := range assignees {
		rows = append(rows, Activity{Field: ptr("assignees"), Verb: VerbUpdated, OldValue: ptr(""), NewValue: ptr(a.DisplayName),
			NewIdentifier: ptr(a.ID), Comment: "added assignee "})
	}
````
````new server/internal/modules/issue/domain/activity.go
	for _, a := range assignees {
		rows = append(rows, assigneeAdded(a))
	}
````
````old server/internal/modules/issue/domain/activity.go
}

func ptr[T any](v T) *T {
````
````new server/internal/modules/issue/domain/activity.go
}

// Named is a row an activity names by its name, a state or a label: its
// id, and its name when the activity is written.
type Named struct {
	ID   uuid.UUID
	Name string
}

// Changed is a value an update changes: the old one and the new one.
type Changed[T any] struct {
	Old, New T
}

// Update is what an update of a work item changes, as its activity names
// it (M4 design 3.13): its name, its priority and its state, each nil when
// it is left as it is; the labels and the accounts it adds and removes,
// each in id order.
type Update struct {
	Name             *Changed[string]
	Priority         *Changed[Priority]
	State            *Changed[Named]
	LabelsAdded      []Named
	LabelsRemoved    []Named
	AssigneesAdded   []Assignee
	AssigneesRemoved []Assignee
}

// UpdateActivity is the rows an update writes, in the order of M4 design
// 3.13's table: row 3, the name; row 6, the priority; row 7, the state, by
// its names and its ids; then each label added (row 10) and removed (row
// 11), by its name and its id; then each account assigned (row 12) and
// unassigned (row 13), by his display name and his id; each list in the
// order given. A field left as it is writes no row (Plane the same).
func UpdateActivity(u Update) []Activity {
	var rows []Activity
	if u.Name != nil {
		rows = append(rows, Activity{Field: ptr("name"), Verb: VerbUpdated, OldValue: ptr(u.Name.Old), NewValue: ptr(u.Name.New),
			Comment: "updated the name to"})
	}
	if u.Priority != nil {
		rows = append(rows, Activity{Field: ptr("priority"), Verb: VerbUpdated, OldValue: ptr(string(u.Priority.Old)),
			NewValue: ptr(string(u.Priority.New)), Comment: "updated the priority to"})
	}
	if u.State != nil {
		rows = append(rows, Activity{Field: ptr("state"), Verb: VerbUpdated, OldValue: ptr(u.State.Old.Name), NewValue: ptr(u.State.New.Name),
			OldIdentifier: ptr(u.State.Old.ID), NewIdentifier: ptr(u.State.New.ID), Comment: "updated the state to"})
	}
	for _, l := range u.LabelsAdded {
		rows = append(rows, Activity{Field: ptr("labels"), Verb: VerbUpdated, OldValue: ptr(""), NewValue: ptr(l.Name),
			NewIdentifier: ptr(l.ID), Comment: "added label "})
	}
	for _, l := range u.LabelsRemoved {
		rows = append(rows, Activity{Field: ptr("labels"), Verb: VerbUpdated, OldValue: ptr(l.Name), NewValue: ptr(""),
			OldIdentifier: ptr(l.ID), Comment: "removed label "})
	}
	for _, a := range u.AssigneesAdded {
		rows = append(rows, assigneeAdded(a))
	}
	for _, a := range u.AssigneesRemoved {
		rows = append(rows, Activity{Field: ptr("assignees"), Verb: VerbUpdated, OldValue: ptr(a.DisplayName), NewValue: ptr(""),
			OldIdentifier: ptr(a.ID), Comment: "removed assignee "})
	}
	return rows
}

func ptr[T any](v T) *T {
````

`server/internal/modules/issue/domain/activity_test.go`（修改，3 处）：

````old server/internal/modules/issue/domain/activity_test.go
	"uuid"
)
````
````new server/internal/modules/issue/domain/activity_test.go
	"uuid"
)

// assigned is the row of the account id, of display name name, assigned to
// a work item: row 2 at its creation and row 12 at an update alike (M4
// design 3.13). Written out apart from assigneeAdded, it lets the tests of
// both check that one encoding.
func assigned(id uuid.UUID, name string) Activity {
	field, empty := "assignees", ""
	return Activity{Field: &field, Verb: "updated", OldValue: &empty, NewValue: &name, NewIdentifier: &id, Comment: "added assignee "}
}
````
````old server/internal/modules/issue/domain/activity_test.go
	created := Activity{Verb: "created", Comment: "created the work item"}
	assigned := func(id uuid.UUID, name string) Activity {
		field, empty := "assignees", ""
		return Activity{Field: &field, Verb: "updated", OldValue: &empty, NewValue: &name, NewIdentifier: &id, Comment: "added assignee "}
	}
````
````new server/internal/modules/issue/domain/activity_test.go
	created := Activity{Verb: "created", Comment: "created the work item"}
````
````old server/internal/modules/issue/domain/activity_test.go
	}
}

````
````new server/internal/modules/issue/domain/activity_test.go
	}
}

// An update writes a row for each field it changes, in the order of M4
// design 3.13's table, whatever else it changes: the name (row 3), the
// priority (row 6), the state by its names and ids (row 7), each label
// added and removed by its name and id (rows 10, 11), each account
// assigned and unassigned by his display name and id (rows 12, 13), each
// list in the order given; Plane's sentences go with them. A field left as
// it is writes none.
func TestUpdateActivity(t *testing.T) {
	todo, done, bug, ui, ann, ben := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	row := func(field string, oldValue, newValue string, oldID, newID *uuid.UUID, comment string) Activity {
		return Activity{Field: &field, Verb: "updated", OldValue: &oldValue, NewValue: &newValue, OldIdentifier: oldID, NewIdentifier: newID,
			Comment: comment}
	}
	name := row("name", "Fix", "Mend", nil, nil, "updated the name to")
	priority := row("priority", "high", "none", nil, nil, "updated the priority to")
	state := row("state", "Todo", "Done", &todo, &done, "updated the state to")
	labelled := func(id uuid.UUID, n string) Activity { return row("labels", "", n, nil, &id, "added label ") }
	unlabelled := func(id uuid.UUID, n string) Activity { return row("labels", n, "", &id, nil, "removed label ") }
	unassigned := func(id uuid.UUID, n string) Activity { return row("assignees", n, "", &id, nil, "removed assignee ") }
	tests := []struct {
		name string
		u    Update
		want []Activity
	}{
		{"nothing", Update{}, nil},
		{"everything", Update{
			AssigneesRemoved: []Assignee{{ID: ben, DisplayName: "ben"}}, AssigneesAdded: []Assignee{{ID: ann, DisplayName: "Ann L."}},
			LabelsRemoved: []Named{{ID: ui, Name: "UI"}}, LabelsAdded: []Named{{ID: bug, Name: "Bug"}},
			State:    &Changed[Named]{Old: Named{ID: todo, Name: "Todo"}, New: Named{ID: done, Name: "Done"}},
			Priority: &Changed[Priority]{Old: PriorityHigh, New: PriorityNone}, Name: &Changed[string]{Old: "Fix", New: "Mend"},
		}, []Activity{name, priority, state, labelled(bug, "Bug"), unlabelled(ui, "UI"), assigned(ann, "Ann L."), unassigned(ben, "ben")}},
		{"the state alone", Update{State: &Changed[Named]{Old: Named{ID: todo, Name: "Todo"}, New: Named{ID: done, Name: "Done"}}},
			[]Activity{state}},
		{"two labels and two accounts each way, in the order given", Update{
			LabelsAdded: []Named{{ID: ui, Name: "UI"}, {ID: bug, Name: "Bug"}}, LabelsRemoved: []Named{{ID: bug, Name: "Bug"}, {ID: ui, Name: "UI"}},
			AssigneesAdded:   []Assignee{{ID: ben, DisplayName: "ben"}, {ID: ann, DisplayName: "ann"}},
			AssigneesRemoved: []Assignee{{ID: ann, DisplayName: "ann"}, {ID: ben, DisplayName: "ben"}},
		}, []Activity{labelled(ui, "UI"), labelled(bug, "Bug"), unlabelled(bug, "Bug"), unlabelled(ui, "UI"), assigned(ben, "ben"),
			assigned(ann, "ann"), unassigned(ann, "ann"), unassigned(ben, "ben")}},
	}
	for _, tt := range tests {
		if got := UpdateActivity(tt.u); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: UpdateActivity() = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

````

- [ ] **Step 3: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/domain/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/modules/issue/domain/activity.go server/internal/modules/issue/domain/activity_test.go server/internal/modules/issue/domain/patch.go server/internal/modules/issue/domain/patch_test.go
```
```bash
git commit -m "feat(M4/P1): the rules of a work item's change and its activity's rows

A patch names the fields it changes; it is checked by createIssue's
rules and stored without the white space around its name. What it
changes in a work item is the name, the priority and the state where
each differs, and the assignees and labels it adds and removes, each in
id order. The activity writes a row for each change in the order of the
design's table, by names and ids.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p1b.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；静态是 `make lint-web` 的文案和键的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T1.1` | `CheckIssuePatch` 检查去掉空白的名称，存的却是原样的 | `internal/modules/issue/domain` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 单元；端到端 |
| `T1.2` | `CheckIssuePatch` 放过任何优先级 | `internal/modules/issue/domain` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 单元；端到端 |
| `T1.3` | `CheckIssuePatch` 不检查给出的标签（重复的、太多的） | `internal/modules/issue/domain` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 单元；端到端 |
| `T1.4` | 差集把新集合的每个 id 都算作加的，工作项已有的也算 | `internal/modules/issue/domain` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 单元；端到端 |
| `T1.5` | 差集按给出的次序加，不按 `id`（约定五） | `internal/modules/issue/domain` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 单元；端到端 |
| `T1.6` | 差集什么都不去 | `internal/modules/issue/domain` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 单元；端到端 |
| `T1.7` | 给出原样的名称也算改动 | `internal/modules/issue/domain` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 单元；端到端 |
| `T1.8` | 给出原样的状态也算改动 | `internal/modules/issue/domain` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 单元；端到端 |
| `T1.9` | `None` 不看去掉的标签 | `internal/modules/issue/domain` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 单元；端到端 |
| `T1.10` | 状态的一行排在优先级的之前（违反 3.13 的次序） | `internal/modules/issue/domain` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 单元；端到端 |
| `T1.11` | 去掉的标签的一行不带它的 `id` | `internal/modules/issue/domain` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 单元；端到端 |
| `T1.12` | 去掉的负责人的一行把他写成新值 | `internal/modules/issue/domain` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 单元；端到端 |
| `T1.13` | 状态的一行的两个 `id` 写反 | `internal/modules/issue/domain` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 单元；端到端 |
| `T1.14` | 给出原样的优先级也算改动 | `internal/modules/issue/domain` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 单元；端到端 |
| `T1.15` | `None` 不看去掉的负责人 | `internal/modules/issue/domain` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 单元；端到端 |

---

### Task 2: 状态、标签的目录回答名称

**Files:**
- Modify: `server/internal/bootstrap/ports.go`、`server/internal/bootstrap/ports_test.go`、`server/internal/modules/issue/app/create_issue.go`、`server/internal/modules/issue/app/create_issue_test.go`、`server/internal/modules/issue/app/fakes_test.go`、`server/internal/modules/issue/app/ports.go`、`server/internal/modules/issue/module.go`、`server/internal/modules/project/adapter/postgres/issue_ports.go`、`server/internal/modules/project/adapter/postgres/issue_ports_test.go`、`server/internal/modules/project/adapter/postgres/queries/issue_ports.sql`、`server/internal/modules/project/app/issue_ports.go`、`server/internal/modules/project/module.go`
- Generate: `server/internal/modules/project/adapter/postgres/gen/issue_ports.sql.go`

**Interfaces:**（spec 2.4、第 3 节第 9 条；M4 设计 3.13、6.5）
- `project`：
  - `queries/issue_ports.sql`：`IssueStates` 多读 `name`；`IssueLabels` 回答 `id, name`（说明写明动态按名称写它们）；
  - `app/issue_ports.go`：`IssueState.Name`，新的 `IssueLabel{ID, Name}`；`Store.IssueLabels` 回答 `[]IssueLabel`；`module.go` 的 `IssueProjects` 接口和别名。
- `issue/app/ports.go`：`StateRef.Name`；`LabelCatalog.Labels` 回答 `[]LabelRef`，`LabelRef{ID, Name}`；`issue/module.go` 的别名 `LabelRef`。
- `bootstrap/ports.go`：`issueProjects.States` 带上名称，`issueProjects.Labels` 转换成 `[]issue.LabelRef`（失败时没有标签）。
- `create_issue.go`：按 `id` 找给出的标签（`slices.ContainsFunc`）。

**Tests:**
- 存储：`TestIssueCatalogs`（状态带名称；标签带名称，多一个 Web 的 UI，按集合比较：写的一侧按 `id` 查找）。
- 端口的转换：`TestIssueProjectsConvertsProjectsAnswers`（状态、标签的每个字段；标签读失败时没有标签、原样的错误）。
- 单元：`issue/app` 的创建测试照旧通过（假实现的标签目录回答 `LabelRef`）。

- [ ] **Step 1: 查询，生成**

`server/internal/modules/project/adapter/postgres/queries/issue_ports.sql`（修改，2 处）：

````old server/internal/modules/project/adapter/postgres/queries/issue_ports.sql
-- StateCatalog (M4 design 6.5): the project's undeleted states, its triage state too, in id order. Read under the
-- project's FOR NO KEY UPDATE, which every write of a state holds too (M3 design 3.6), it stays as read (M4 design 3.5).
SELECT id, "group", "default"
````
````new server/internal/modules/project/adapter/postgres/queries/issue_ports.sql
-- StateCatalog (M4 design 6.5): the project's undeleted states, its triage state too, in id order, with their names,
-- which an update's activity names a state by (3.13). Read under the project's FOR NO KEY UPDATE, which every write of
-- a state holds too (M3 design 3.6), it stays as read (M4 design 3.5).
SELECT id, name, "group", "default"
````
````old server/internal/modules/project/adapter/postgres/queries/issue_ports.sql
-- LabelCatalog (M4 design 6.5): those of ids that are the project's undeleted labels, in id order; stable under the
-- project's FOR NO KEY UPDATE, as IssueStates.
SELECT id
````
````new server/internal/modules/project/adapter/postgres/queries/issue_ports.sql
-- LabelCatalog (M4 design 6.5): those of ids that are the project's undeleted labels, in id order, with their names,
-- which an update's activity names a label by (3.13); stable under the project's FOR NO KEY UPDATE, as IssueStates.
SELECT id, name
````

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `44b659b2fe2b93fe869d6696207cf57b052f6c2867ff44f3cb6ea35eee9dcfca` | 172 | `server/internal/modules/project/adapter/postgres/gen/issue_ports.sql.go` |

Run: `shasum -a 256 server/internal/modules/project/adapter/postgres/gen/issue_ports.sql.go`
Expected: 与上表相同。

- [ ] **Step 2: `project` 的存储和端口**

`server/internal/modules/project/adapter/postgres/issue_ports.go`（修改，4 处）：

````old server/internal/modules/project/adapter/postgres/issue_ports.go
// too, in id order (M4 design 6.5).
````
````new server/internal/modules/project/adapter/postgres/issue_ports.go
// too, in id order, with their names (M4 design 6.5).
````
````old server/internal/modules/project/adapter/postgres/issue_ports.go
		out[i] = app.IssueState{ID: r.ID, Group: r.Group, Default: r.Default}
````
````new server/internal/modules/project/adapter/postgres/issue_ports.go
		out[i] = app.IssueState{ID: r.ID, Name: r.Name, Group: r.Group, Default: r.Default}
````
````old server/internal/modules/project/adapter/postgres/issue_ports.go
// in id order (M4 design 6.5).
func (s *Store) IssueLabels(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	found, err := s.queries(ctx).IssueLabels(ctx, gen.IssueLabelsParams{ProjectID: projectID, Ids: ids})
````
````new server/internal/modules/project/adapter/postgres/issue_ports.go
// in id order, with their names (M4 design 6.5).
func (s *Store) IssueLabels(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) ([]app.IssueLabel, error) {
	rows, err := s.queries(ctx).IssueLabels(ctx, gen.IssueLabelsParams{ProjectID: projectID, Ids: ids})
````
````old server/internal/modules/project/adapter/postgres/issue_ports.go
	return found, nil
````
````new server/internal/modules/project/adapter/postgres/issue_ports.go
	out := make([]app.IssueLabel, len(rows))
	for i, r := range rows {
		out[i] = app.IssueLabel{ID: r.ID, Name: r.Name}
	}
	return out, nil
````

`server/internal/modules/project/adapter/postgres/issue_ports_test.go`（修改，4 处）：

````old server/internal/modules/project/adapter/postgres/issue_ports_test.go
// undeleted states, its triage state too, in id order, with their groups
// and the default; those of the labels asked for that are its undeleted
// labels; the roles of the accounts asked for who are its active members.
````
````new server/internal/modules/project/adapter/postgres/issue_ports_test.go
// undeleted states, its triage state too, in id order, with their names,
// their groups and the default; those of the labels asked for that are its
// undeleted labels, with their names, in whatever order (a work item's
// write looks each up by its id); the roles of the accounts asked for who
// are its active members.
````
````old server/internal/modules/project/adapter/postgres/issue_ports_test.go
	seedLabel(t, s, acme, web, nil, "Feature", alice)
````
````new server/internal/modules/project/adapter/postgres/issue_ports_test.go
	seedLabel(t, s, acme, web, nil, "Feature", alice)
	ui := seedLabel(t, s, acme, web, nil, "UI", alice)
````
````old server/internal/modules/project/adapter/postgres/issue_ports_test.go
		want = append(want, app.IssueState{ID: id, Group: groups[name], Default: name == "Backlog"})
````
````new server/internal/modules/project/adapter/postgres/issue_ports_test.go
		want = append(want, app.IssueState{ID: id, Name: name, Group: groups[name], Default: name == "Backlog"})
````
````old server/internal/modules/project/adapter/postgres/issue_ports_test.go
	if got, err := s.IssueLabels(ctx, web, []uuid.UUID{opsBug, oldBug, bug, uuid.NewV7()}); err != nil || !slices.Equal(got, []uuid.UUID{bug}) {
		t.Errorf("IssueLabels(Web, …) = %v, %v; want only Web's Bug", got, err)
````
````new server/internal/modules/project/adapter/postgres/issue_ports_test.go
	gotLabels, err := s.IssueLabels(ctx, web, []uuid.UUID{opsBug, ui, oldBug, bug, uuid.NewV7()})
	slices.SortFunc(gotLabels, func(a, b app.IssueLabel) int { return a.ID.Compare(b.ID) })
	if want := []app.IssueLabel{{ID: bug, Name: "Bug"}, {ID: ui, Name: "UI"}}; err != nil || !slices.Equal(gotLabels, want) {
		t.Errorf("IssueLabels(Web, …) = %+v, %v; want only Web's Bug and UI", gotLabels, err)
````

`server/internal/modules/project/app/issue_ports.go`（修改，3 处）：

````old server/internal/modules/project/app/issue_ports.go
// issue module reads it (M4 design 6.5, StateCatalog): its id, its group,
// and whether it is the project's default. bootstrap converts it into
// issue's value.
````
````new server/internal/modules/project/app/issue_ports.go
// issue module reads it (M4 design 6.5, StateCatalog): its id, its name,
// its group, and whether it is the project's default. bootstrap converts
// it into issue's value.
````
````old server/internal/modules/project/app/issue_ports.go
	ID      uuid.UUID
````
````new server/internal/modules/project/app/issue_ports.go
	ID      uuid.UUID
	Name    string
````
````old server/internal/modules/project/app/issue_ports.go
	Default bool
}

````
````new server/internal/modules/project/app/issue_ports.go
	Default bool
}

// IssueLabel is a label of a project as project.Provide's catalog for the
// issue module reads it (M4 design 6.5, LabelCatalog): its id and its
// name. bootstrap converts it into issue's value.
type IssueLabel struct {
	ID   uuid.UUID
	Name string
}

````

`server/internal/modules/project/module.go`（修改，3 处）：

````old server/internal/modules/project/module.go
	IssueLabels(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error)
````
````new server/internal/modules/project/module.go
	IssueLabels(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) ([]IssueLabel, error)
````
````old server/internal/modules/project/module.go
// IssueProject and IssueState are the values IssueProjects hands over:
// bootstrap converts them into issue's (M4 design 6.5).
````
````new server/internal/modules/project/module.go
// IssueProject, IssueState and IssueLabel are the values IssueProjects
// hands over: bootstrap converts them into issue's (M4 design 6.5).
````
````old server/internal/modules/project/module.go
	IssueState   = app.IssueState
````
````new server/internal/modules/project/module.go
	IssueState   = app.IssueState
	IssueLabel   = app.IssueLabel
````

- [ ] **Step 3: `issue` 的端口和转换**

`server/internal/bootstrap/ports.go`（修改，2 处）：

````old server/internal/bootstrap/ports.go
		out[i] = issue.StateRef{ID: s.ID, Group: issue.StateGroup(s.Group), Default: s.Default}
````
````new server/internal/bootstrap/ports.go
		out[i] = issue.StateRef{ID: s.ID, Name: s.Name, Group: issue.StateGroup(s.Group), Default: s.Default}
````
````old server/internal/bootstrap/ports.go
func (p issueProjects) Labels(ctx context.Context, id uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	return p.projects.IssueLabels(ctx, id, ids)
````
````new server/internal/bootstrap/ports.go
func (p issueProjects) Labels(ctx context.Context, id uuid.UUID, ids []uuid.UUID) ([]issue.LabelRef, error) {
	labels, err := p.projects.IssueLabels(ctx, id, ids)
	if err != nil {
		return nil, err
	}
	out := make([]issue.LabelRef, len(labels))
	for i, l := range labels {
		out[i] = issue.LabelRef(l)
	}
	return out, nil
````

`server/internal/bootstrap/ports_test.go`（修改，10 处）：

````old server/internal/bootstrap/ports_test.go
	labels []uuid.UUID
````
````new server/internal/bootstrap/ports_test.go
	labels []project.IssueLabel
````
````old server/internal/bootstrap/ports_test.go
func (f *fakeIssueProjects) IssueLabels(_ context.Context, _ uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
````
````new server/internal/bootstrap/ports_test.go
func (f *fakeIssueProjects) IssueLabels(_ context.Context, _ uuid.UUID, ids []uuid.UUID) ([]project.IssueLabel, error) {
````
````old server/internal/bootstrap/ports_test.go
// locked project and each state converted, every field; the workspace, the
// number, the labels and the roles as they came; found and the error as
// they came.
````
````new server/internal/bootstrap/ports_test.go
// locked project, each state and each label converted, every field; the
// workspace, the number and the roles as they came; found and the error as
// they came, and no state or label with an error.
````
````old server/internal/bootstrap/ports_test.go
	web, acme, ann, bug := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	backlog := project.IssueState{ID: uuid.NewV7(), Group: "backlog", Default: true}
	done := project.IssueState{ID: uuid.NewV7(), Group: "completed"}
````
````new server/internal/bootstrap/ports_test.go
	web, acme, ann, bug, ui := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	backlog := project.IssueState{ID: uuid.NewV7(), Name: "Backlog", Group: "backlog", Default: true}
	done := project.IssueState{ID: uuid.NewV7(), Name: "Done", Group: "completed"}
````
````old server/internal/bootstrap/ports_test.go
		states: []project.IssueState{backlog, done}, labels: []uuid.UUID{bug}, roles: map[uuid.UUID]shared.Role{ann: shared.RoleMember}}
````
````new server/internal/bootstrap/ports_test.go
		states: []project.IssueState{backlog, done}, labels: []project.IssueLabel{{ID: bug, Name: "Bug"}, {ID: ui, Name: "UI"}},
		roles: map[uuid.UUID]shared.Role{ann: shared.RoleMember}}
````
````old server/internal/bootstrap/ports_test.go
	want := []issue.StateRef{{ID: backlog.ID, Group: "backlog", Default: true}, {ID: done.ID, Group: "completed"}}
````
````new server/internal/bootstrap/ports_test.go
	want := []issue.StateRef{{ID: backlog.ID, Name: "Backlog", Group: "backlog", Default: true}, {ID: done.ID, Name: "Done", Group: "completed"}}
````
````old server/internal/bootstrap/ports_test.go
	if got, err := p.Labels(ctx, web, []uuid.UUID{bug}); err != nil || !slices.Equal(got, []uuid.UUID{bug}) {
		t.Errorf("Labels() = %v, %v; want Bug", got, err)
````
````new server/internal/bootstrap/ports_test.go
	if got, err := p.Labels(ctx, web, []uuid.UUID{bug, ui}); err != nil || !slices.Equal(got, []issue.LabelRef{{ID: bug, Name: "Bug"}, {ID: ui, Name: "UI"}}) {
		t.Errorf("Labels() = %+v, %v; want Bug and UI", got, err)
````
````old server/internal/bootstrap/ports_test.go
	if want := []string{"workspace", "lock", "lock", "next", "states", fmt.Sprint("labels ", []uuid.UUID{bug}),
````
````new server/internal/bootstrap/ports_test.go
	if want := []string{"workspace", "lock", "lock", "next", "states", fmt.Sprint("labels ", []uuid.UUID{bug, ui}),
````
````old server/internal/bootstrap/ports_test.go
		t.Errorf("States() = %+v, %v; want no states and %v", got, err, failure)
	}
````
````new server/internal/bootstrap/ports_test.go
		t.Errorf("States() = %+v, %v; want no states and %v", got, err, failure)
	}
	if got, err := p.Labels(ctx, web, nil); !errors.Is(err, failure) || got != nil {
		t.Errorf("Labels() = %+v, %v; want no labels and %v", got, err, failure)
	}
````
````old server/internal/bootstrap/ports_test.go
		"Find":   func() error { _, _, err := p.Find(ctx, web); return err },
		"Lock":   func() error { _, _, err := p.Lock(ctx, web); return err },
		"Next":   func() error { _, err := p.Next(ctx, web); return err },
		"Labels": func() error { _, err := p.Labels(ctx, web, nil); return err },
		"Roles":  func() error { _, err := p.Roles(ctx, web, nil); return err },
````
````new server/internal/bootstrap/ports_test.go
		"Find":  func() error { _, _, err := p.Find(ctx, web); return err },
		"Lock":  func() error { _, _, err := p.Lock(ctx, web); return err },
		"Next":  func() error { _, err := p.Next(ctx, web); return err },
		"Roles": func() error { _, err := p.Roles(ctx, web, nil); return err },
````

`server/internal/modules/issue/app/create_issue.go`（修改，1 处）：

````old server/internal/modules/issue/app/create_issue.go
			if !slices.Contains(known, id) {
````
````new server/internal/modules/issue/app/create_issue.go
			if !slices.ContainsFunc(known, func(l LabelRef) bool { return l.ID == id }) {
````

`server/internal/modules/issue/app/create_issue_test.go`（修改，1 处）：

````old server/internal/modules/issue/app/create_issue_test.go
		states: []app.StateRef{backlog, done, triage}, labels: []uuid.UUID{bug, ui},
````
````new server/internal/modules/issue/app/create_issue_test.go
		states: []app.StateRef{backlog, done, triage}, labels: []app.LabelRef{{ID: bug, Name: "Bug"}, {ID: ui, Name: "UI"}},
````

`server/internal/modules/issue/app/fakes_test.go`（修改，3 处）：

````old server/internal/modules/issue/app/fakes_test.go
	labels    []uuid.UUID
````
````new server/internal/modules/issue/app/fakes_test.go
	labels    []app.LabelRef
````
````old server/internal/modules/issue/app/fakes_test.go
func (f *fakeProject) Labels(ctx context.Context, id uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
````
````new server/internal/modules/issue/app/fakes_test.go
func (f *fakeProject) Labels(ctx context.Context, id uuid.UUID, ids []uuid.UUID) ([]app.LabelRef, error) {
````
````old server/internal/modules/issue/app/fakes_test.go
	return slices.DeleteFunc(slices.Clone(ids), func(l uuid.UUID) bool { return !slices.Contains(f.labels, l) }), nil
````
````new server/internal/modules/issue/app/fakes_test.go
	return slices.DeleteFunc(slices.Clone(f.labels), func(l app.LabelRef) bool { return !slices.Contains(ids, l.ID) }), nil
````

`server/internal/modules/issue/app/ports.go`（修改，3 处）：

````old server/internal/modules/issue/app/ports.go
// StateRef is a state as StateCatalog reads it: its id, its group, and
// whether it is the project's default.
````
````new server/internal/modules/issue/app/ports.go
// StateRef is a state as StateCatalog reads it: its id, its name, its
// group, and whether it is the project's default.
````
````old server/internal/modules/issue/app/ports.go
	ID      uuid.UUID
````
````new server/internal/modules/issue/app/ports.go
	ID      uuid.UUID
	Name    string
````
````old server/internal/modules/issue/app/ports.go
	Labels(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error)
````
````new server/internal/modules/issue/app/ports.go
	Labels(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) ([]LabelRef, error)
}

// LabelRef is a label as LabelCatalog reads it: its id and its name.
type LabelRef struct {
	ID   uuid.UUID
	Name string
````

`server/internal/modules/issue/module.go`（修改，2 处）：

````old server/internal/modules/issue/module.go
// LockedProject, StateRef and its StateGroup are the values ProjectPorts
// hand over: bootstrap converts project's into them (M4 design 6.5).
````
````new server/internal/modules/issue/module.go
// LockedProject, StateRef and its StateGroup, and LabelRef are the values
// ProjectPorts hand over: bootstrap converts project's into them (M4
// design 6.5).
````
````old server/internal/modules/issue/module.go
	StateGroup    = domain.StateGroup
````
````new server/internal/modules/issue/module.go
	StateGroup    = domain.StateGroup
	LabelRef      = app.LabelRef
````

- [ ] **Step 4: 测试和 lint**

Run: `go -C server test -count=1 -run 'TestIssueCatalogs$' ./internal/modules/project/adapter/postgres/`
Expected: `ok`。

Run: `go -C server test -count=1 ./internal/modules/issue/... ./internal/modules/project/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestIssueProjectsConvertsProjectsAnswers$' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 5: 提交**

```bash
git add server/internal/bootstrap/ports.go server/internal/bootstrap/ports_test.go server/internal/modules/issue/app/create_issue.go server/internal/modules/issue/app/create_issue_test.go server/internal/modules/issue/app/fakes_test.go server/internal/modules/issue/app/ports.go server/internal/modules/issue/module.go server/internal/modules/project/adapter/postgres/issue_ports.go server/internal/modules/project/adapter/postgres/issue_ports_test.go server/internal/modules/project/adapter/postgres/queries/issue_ports.sql server/internal/modules/project/app/issue_ports.go server/internal/modules/project/module.go server/internal/modules/project/adapter/postgres/gen/issue_ports.sql.go
```
```bash
git commit -m "feat(M4/P1): the state and label catalogs answer names

A change's activity names the states and labels it changes by their
names, so project's catalogs for the issue module answer each state's
and each label's name with its id, and bootstrap converts them.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p1b.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；静态是 `make lint-web` 的文案和键的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T2.1` | 状态的目录不回答名称 | `internal/modules/project/adapter/postgres` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestUpdatingAWorkItemStampsItsRows`（`make test`）（Task 7 起）、故事 `i2-update-issue`（Task 14 起） | 存储；组合；端到端 |
| `T2.2` | 标签的目录不回答名称 | `internal/modules/project/adapter/postgres` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 存储；端到端 |
| `T2.3` | `bootstrap` 的转换丢掉状态的名称 | `internal/bootstrap` 的 `TestIssueProjectsConvertsProjectsAnswers`（`make test`）、故事 `i2-update-issue`（Task 14 起） | 单元；端到端 |

---

### Task 3: 修改的存储

**Files:**
- Create: `server/internal/modules/issue/adapter/postgres/queries/update.sql`、`server/internal/modules/issue/adapter/postgres/update.go`、`server/internal/modules/issue/adapter/postgres/update_test.go`
- Modify: `server/internal/modules/issue/app/rows.go`
- Generate: `server/internal/modules/issue/adapter/postgres/gen/update.sql.go`

**Interfaces:**（spec 2.5、第 3 节第 12 条；M4 设计 3.5、3.7）
- `issue/adapter/postgres/queries/update.sql`（新）：
  - `UpdateIssue`：名称、状态、优先级、`completed_at`（给什么写什么，`NULL` 也写）、`updated_by_id`、`updated_at`，`WHERE id = $id`；
  - `RemoveAssignees`、`RemoveLabels`：这个工作项未删除的、给出的账户或标签的行，写 `deleted_at`、`updated_at`、`updated_by_id`，一条语句。说明写明三条都在项目 N 之下，不按 `id` 先锁（约定五）。
- `update.go`（新）：`Store.UpdateIssue(ctx, IssueFields)`、`RemoveAssignees(ctx, UnlinkRows)`、`RemoveLabels(ctx, UnlinkRows)`；失败照原样包一层返回。
- `issue/app/rows.go`：`IssueFields{ID, Name, StateID, Priority, CompletedAt, By, At}`、`UnlinkRows{IssueID, Targets, By, At}`。

**Tests:**（存储）
- `TestUpdatingAWorkItemWritesItsRows`：bob 在 `later` 改 alice 建的 7 号工作项：名称、状态（Done）、优先级、完成时间照给的写，编号、排序值、创建的戳不变；去掉 bob 的负责人行和 UI 的标签行（写下 bob、`later`）；另一个工作项的同一个账户、标签的行，这个工作项上没给的账户、标签的行，一小时之前已删除的同一个账户、标签的行，每一列都不变；改回 Todo、完成时间给空，就不再完成。
- `TestAFailedUpdateStatementIsAnError`：三条语句的失败（取消的 context）照原样返回。

- [ ] **Step 1: 查询，生成**

`server/internal/modules/issue/adapter/postgres/queries/update.sql`（新文件，24 行）：

````file server/internal/modules/issue/adapter/postgres/queries/update.sql
-- name: UpdateIssue :exec
-- updateIssue's work item (M4 design 3.7), under the project's FOR NO KEY UPDATE, read again under it: its name, state,
-- priority and completion as the use case gives them, those it changes and the others as they are, last changed at the
-- moment and by the account given. Every write of a work item's row holds its project's lock first (3.5), so the row is
-- as the use case read it; it is not locked by id first (M3 design 3.6, convention 5).
UPDATE issues
SET name = sqlc.arg(name), state_id = sqlc.arg(state_id), priority = sqlc.arg(priority), completed_at = sqlc.narg(completed_at),
    updated_by_id = sqlc.arg(by)::uuid, updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id);

-- name: RemoveAssignees :exec
-- The assignees updateIssue removes (M4 design 3.7): the work item's undeleted rows of the accounts, soft-deleted at
-- the moment and by the account given, in one statement. Every write of these rows holds the project's FOR NO KEY
-- UPDATE first (3.5): no other write reaches them meanwhile, so they are not locked by id first (M3 design 3.6,
-- convention 5). The partial unique key on (issue_id, assignee_id) serves it.
UPDATE issue_assignees
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(by)::uuid
WHERE issue_id = sqlc.arg(issue_id) AND assignee_id = ANY (sqlc.arg(assignee_ids)::uuid[]) AND deleted_at IS NULL;

-- name: RemoveLabels :exec
-- The labels updateIssue removes, as RemoveAssignees; the partial unique key on (issue_id, label_id) serves it.
UPDATE issue_labels
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(by)::uuid
WHERE issue_id = sqlc.arg(issue_id) AND label_id = ANY (sqlc.arg(label_ids)::uuid[]) AND deleted_at IS NULL;
````

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `89190635674568f5a8b9fa63f65ef94e7c884cf6b6bb61e3b0e8af19ab3a60bb` | 98 | `server/internal/modules/issue/adapter/postgres/gen/update.sql.go` |

Run: `shasum -a 256 server/internal/modules/issue/adapter/postgres/gen/update.sql.go`
Expected: 与上表相同。

- [ ] **Step 2: 存储和测试**

`server/internal/modules/issue/adapter/postgres/update.go`（新文件，46 行）：

````file server/internal/modules/issue/adapter/postgres/update.go
package postgresadapter

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// UpdateIssue writes the work item's fields f (M4 design 3.7). The use
// case read it under the project's lock and checked every value, so any
// violation is a bug: an internal error (500), not a domain error.
func (s *Store) UpdateIssue(ctx context.Context, f app.IssueFields) error {
	err := s.queries(ctx).UpdateIssue(ctx, gen.UpdateIssueParams{
		ID: f.ID, Name: f.Name, StateID: f.StateID, Priority: string(f.Priority), CompletedAt: f.CompletedAt, By: f.By, At: f.At,
	})
	if err != nil {
		return fmt.Errorf("update work item %s: %w", f.ID, err)
	}
	return nil
}

// RemoveAssignees soft-deletes the work item's undeleted assignees' rows
// of the accounts r names.
func (s *Store) RemoveAssignees(ctx context.Context, r app.UnlinkRows) error {
	err := s.queries(ctx).RemoveAssignees(ctx, gen.RemoveAssigneesParams{IssueID: r.IssueID, AssigneeIds: r.Targets, By: r.By, At: r.At})
	return unlinked(err, "assignees", r.IssueID)
}

// RemoveLabels soft-deletes the work item's undeleted labels' rows of the
// labels r names.
func (s *Store) RemoveLabels(ctx context.Context, r app.UnlinkRows) error {
	err := s.queries(ctx).RemoveLabels(ctx, gen.RemoveLabelsParams{IssueID: r.IssueID, LabelIds: r.Targets, By: r.By, At: r.At})
	return unlinked(err, "labels", r.IssueID)
}

// unlinked is err, a removal of the work item issue's rows of table, as
// the removal's error; nil is nil.
func unlinked(err error, table string, issue uuid.UUID) error {
	if err != nil {
		return fmt.Errorf("remove %s of work item %s: %w", table, issue, err)
	}
	return nil
}
````

`server/internal/modules/issue/adapter/postgres/update_test.go`（新文件，133 行）：

````file server/internal/modules/issue/adapter/postgres/update_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// updateIssue's rows through the store (M4 design 3.7): UpdateIssue writes
// the work item's name, state, priority and completion as given, last
// changed by bob at later, its number, its sort order and its creation as
// they were; RemoveAssignees and RemoveLabels soft-delete, by bob at
// later, the work item's undeleted rows of every account and label given,
// two of each in one call, every other column kept. Nothing else changes:
// not another work item, nor its rows of a given account and label, nor
// the work item's rows of an account or a label not given, nor its rows of
// a given account and label deleted an hour before, which keep their
// stamps. Moved back into a state of no completed group, its completion
// given nil, the work item is completed no more.
func TestUpdatingAWorkItemWritesItsRows(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	alice, bob, carol, dave := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"),
		newAccount(t, pool, "carol@corp.com"), newAccount(t, pool, "dave@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	done := uuid.NewV7()
	exec(t, pool, `INSERT INTO states (id, workspace_id, project_id, name, color, "group") VALUES ($1, $2, $3, 'Done', '#000', 'completed')`,
		done, web.workspace, web.id)
	bug, ui, docs := newLabel(t, pool, web, "Bug"), newLabel(t, pool, web, "UI"), newLabel(t, pool, web, "Docs")
	id, other := uuid.NewV7(), uuid.NewV7()
	createWebIssue(t, s, web, id, alice)
	newIssue(t, pool, web, other, 8, alice)
	assignees, labels := linkRows(web, id, alice, now, bob, carol, dave), linkRows(web, id, alice, now, bug, ui, docs)
	for _, step := range []struct {
		insert func(context.Context, app.LinkRows) error
		rows   app.LinkRows
	}{{s.InsertAssignees, assignees}, {s.InsertLabels, labels}, {s.InsertAssignees, linkRows(web, other, alice, now, bob)},
		{s.InsertLabels, linkRows(web, other, alice, now, ui)}} {
		if err := step.insert(ctx, step.rows); err != nil {
			t.Fatal(err)
		}
	}
	earlier, later := now.Add(-time.Hour), now.Add(time.Hour)
	for _, deleted := range []struct {
		table, column string
		target        uuid.UUID
	}{{"issue_assignees", "assignee_id", bob}, {"issue_labels", "label_id", ui}} {
		exec(t, pool, `INSERT INTO `+deleted.table+` (id, workspace_id, project_id, issue_id, `+deleted.column+`, updated_by_id, updated_at,
			deleted_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $7)`, uuid.NewV7(), web.workspace, web.id, id, deleted.target, alice, earlier)
	}
	tables := []string{"issues", "issue_assignees", "issue_labels"}
	before, stamps := map[string]map[uuid.UUID]string{}, map[string]map[uuid.UUID]string{}
	for _, table := range tables {
		before[table], stamps[table] = rowsAsText(t, pool, table), audits(t, pool, table)
	}

	if err := s.UpdateIssue(ctx, app.IssueFields{ID: id, Name: "Mend the login", StateID: done, Priority: domain.PriorityLow,
		CompletedAt: &later, By: bob, At: later}); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveAssignees(ctx, app.UnlinkRows{IssueID: id, Targets: []uuid.UUID{bob, dave}, By: bob, At: later}); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveLabels(ctx, app.UnlinkRows{IssueID: id, Targets: []uuid.UUID{ui, docs}, By: bob, At: later}); err != nil {
		t.Fatal(err)
	}

	got, found, err := s.GetIssue(ctx, id)
	want := domain.Issue{ID: id, WorkspaceID: web.workspace, ProjectID: web.id, SequenceID: 7, Name: "Mend the login", StateID: done,
		Priority: domain.PriorityLow, SortOrder: 75535, CompletedAt: &later, CreatedByID: &alice, UpdatedByID: &bob, CreatedAt: now,
		UpdatedAt: later, AssigneeIDs: []uuid.UUID{carol}, LabelIDs: []uuid.UUID{bug}}
	if err != nil || !found || !reflect.DeepEqual(got, want) {
		t.Errorf("GetIssue() = %+v, %v, %v; want %+v", got, found, err, want)
	}
	removed := map[string][]uuid.UUID{"issue_assignees": {assignees.IDs[0], assignees.IDs[2]}, "issue_labels": {labels.IDs[1], labels.IDs[2]}}
	for _, table := range tables {
		after, afterStamps := rowsAsText(t, pool, table), audits(t, pool, table)
		wantStamps := maps.Clone(stamps[table])
		if table == "issues" {
			delete(after, id)
			delete(before[table], id)
			delete(afterStamps, id)
			delete(wantStamps, id)
		}
		for _, row := range removed[table] {
			wantStamps[row] = deletion(later, bob)
		}
		if !maps.Equal(after, before[table]) {
			t.Errorf("%s changed:\n%v\nwas\n%v", table, after, before[table])
		}
		if !maps.Equal(afterStamps, wantStamps) {
			t.Errorf("%s's stamps:\n%v\nwant\n%v", table, afterStamps, wantStamps)
		}
	}

	if err := s.UpdateIssue(ctx, app.IssueFields{ID: id, Name: "Mend the login", StateID: web.state, Priority: domain.PriorityLow, By: bob,
		At: later}); err != nil {
		t.Fatal(err)
	}
	if got, _, err := s.GetIssue(ctx, id); err != nil || got.StateID != web.state || got.CompletedAt != nil {
		t.Errorf("GetIssue() moved back = %+v, %v; want it in its first state, completed_at NULL", got, err)
	}
}

// A write of an update that fails answers its error as itself (M4 design
// 3.5).
func TestAFailedUpdateStatementIsAnError(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	id := uuid.NewV7()
	newIssue(t, pool, web, id, 1, alice)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.UpdateIssue(cancelled, app.IssueFields{ID: id, Name: "a", StateID: web.state, Priority: domain.PriorityNone, By: alice,
		At: now}); !errors.Is(err, context.Canceled) {
		t.Errorf("UpdateIssue() = %v; want context.Canceled", err)
	}
	unlink := app.UnlinkRows{IssueID: id, Targets: []uuid.UUID{alice}, By: alice, At: now}
	for name, remove := range map[string]func(context.Context, app.UnlinkRows) error{"RemoveAssignees": s.RemoveAssignees,
		"RemoveLabels": s.RemoveLabels} {
		if err := remove(cancelled, unlink); !errors.Is(err, context.Canceled) {
			t.Errorf("%s() = %v; want context.Canceled", name, err)
		}
	}
}
````

`server/internal/modules/issue/app/rows.go`（修改，1 处）：

````old server/internal/modules/issue/app/rows.go
}

// ActivityRow is a new row of a work item's activity (M4 design 3.13):
````
````new server/internal/modules/issue/app/rows.go
}

// IssueFields are a work item's fields as an update writes them (M4 design
// 3.7): those it changes and the others as they are, and the moment and
// the account that change it last.
type IssueFields struct {
	ID          uuid.UUID
	Name        string
	StateID     uuid.UUID
	Priority    domain.Priority
	CompletedAt *time.Time
	By          uuid.UUID
	At          time.Time
}

// UnlinkRows are the rows of one of a work item's tables of accounts or
// labels, its assignees' or labels', that an update removes: its undeleted
// rows of Targets, soft-deleted at the moment and by the account given.
type UnlinkRows struct {
	IssueID uuid.UUID
	Targets []uuid.UUID
	By      uuid.UUID
	At      time.Time
}

// ActivityRow is a new row of a work item's activity (M4 design 3.13):
````

- [ ] **Step 3: 测试和 lint**

Run: `go -C server test -count=1 -run 'TestUpdatingAWorkItemWritesItsRows$|TestAFailedUpdateStatementIsAnError$' ./internal/modules/issue/adapter/postgres/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/modules/issue/adapter/postgres/queries/update.sql server/internal/modules/issue/adapter/postgres/update.go server/internal/modules/issue/adapter/postgres/update_test.go server/internal/modules/issue/app/rows.go server/internal/modules/issue/adapter/postgres/gen/update.sql.go
```
```bash
git commit -m "feat(M4/P1): store a work item's change and the rows it removes

UpdateIssue writes a work item's name, state, priority and completion
as given, changed last by the caller at the moment. RemoveAssignees and
RemoveLabels soft-delete the work item's undeleted rows of the accounts
and labels given, in one statement each, under the project's lock as
every writer of those rows is, so they lock nothing by id first.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p1b.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；静态是 `make lint-web` 的文案和键的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T3.1` | `UpdateIssue` 写每个工作项，不只是给出的那个 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 存储；端到端 |
| `T3.2` | `UpdateIssue` 保留工作项原来的最后修改者 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 存储；端到端 |
| `T3.3` | `UpdateIssue` 从不清空 `completed_at` | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 存储；端到端 |
| `T3.4` | `RemoveAssignees` 从每个工作项上去掉这些账户 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 存储；端到端 |
| `T3.5` | `RemoveAssignees` 再删一次早已删除的行，盖掉它们的戳 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 存储；端到端 |
| `T3.6` | `RemoveLabels` 去掉工作项的每个标签，不只是给出的 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 存储；端到端 |
| `T3.7` | `RemoveLabels` 再删一次早已删除的行，盖掉它们的戳 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 存储；端到端 |
| `T3.8` | `RemoveLabels` 保留行原来的最后修改者 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 存储；端到端 |
| `T3.9` | `RemoveAssignees` 把语句的失败答成没事 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`） | 存储 |
| `T3.10` | `RemoveAssignees` 去掉工作项的每个负责人，不只是给出的 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 存储；端到端 |
| `T3.11` | `RemoveLabels` 从每个工作项上去掉这些标签 | `internal/modules/issue/adapter/postgres` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 存储；端到端 |

---

### Task 4: `getIssue`

**Files:**
- Create: `server/internal/bootstrap/permission_matrix_seed_issues_test.go`、`server/internal/modules/issue/app/get_issue.go`、`server/internal/modules/issue/app/get_issue_test.go`
- Modify: `api/modules/issue.yaml`、`api/openapi.yaml`、`server/internal/bootstrap/permission_matrix_issues_test.go`、`server/internal/bootstrap/permission_matrix_project_test.go`、`server/internal/bootstrap/permission_matrix_seeded_test.go`、`server/internal/bootstrap/permission_matrix_targets_test.go`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/issue/adapter/http/handler.go`、`server/internal/modules/issue/adapter/http/handler_test.go`、`server/internal/modules/issue/adapter/http/issues.go`、`server/internal/modules/issue/adapter/http/issues_test.go`、`server/internal/modules/issue/app/create_issue_test.go`、`server/internal/modules/issue/app/lock.go`、`server/internal/modules/issue/app/ports.go`、`server/internal/modules/issue/domain/actions.go`、`server/internal/modules/issue/domain/errors.go`、`server/internal/modules/issue/module.go`、`web/apps/web/core/lib/error-messages.ts`、`web/packages/i18n/src/locales/en/errors.json`、`web/packages/i18n/src/locales/zh-CN/errors.json`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/issue/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**（spec 2.6、第 3 节第 8 条；M4 设计 3.4、3.5、3.6、5.1–5.3、9.2）
- `issue/domain`：`errors.go` 的 `ErrIssueNotFound`（404 `issue.not_found`）；`actions.go` 的 `ActionRead`（`issue.read`），`Actions()` 带上它。
- `access/domain/rules.go`：`issue.read` 一行（项目级，管理员、成员；访客是 P2 的）；`rules_test.go` 的一行。
- `issue/app`：
  - `ports.go`：`IssueReader`（`GetIssue(ctx, id) (domain.Issue, bool, error)`），`IssueCreator` 嵌入它；
  - `lock.go`：`Locks` 带上 `issues IssueReader`（`NewLocks(projects, issues, auth)`）；`see(ctx, actor, id, action)`：不加锁地读工作项、判定它的项目，没有或看不到 `issue.not_found`；`decide` 带上 `notFound`；
  - `get_issue.go`（新）：`GetIssue`、`NewGetIssue(locks)`、`Execute(ctx, id)`：`see` 一步，不取锁、不开事务。
- 契约：`api/modules/issue.yaml` 的 `GET /api/v0/issues/{issue_id}`（`getIssue`，`x-problem-codes: [issue.not_found, forbidden]`，回答 `IssueDetail`）、`IssueID` 参数；`api/openapi.yaml` 的路径。
- `issue/adapter/http`：`handler.go` 的 `GetIssueUseCase`、`UseCases.GetIssue`；`issues.go` 的 `GetIssue`；测试的 `newServer(t, UseCases)`。`issue/module.go` 接上 `GetIssue`。
- 前端：`error-messages.ts` 的 `"issue.not_found"`，两份 `errors.json` 的 `issue_not_found`（约束 4）。
- 矩阵：
  - `permission_matrix_seed_issues_test.go`（新）：`matrixIssue`（"Seeded"）、`issueSeed`：每个项目一个工作项，Todo、medium，指派项目管理员和一个已离开的账户，标签 Bug，经 `NextIssueSequence` 和 `issue` 的存储写；
  - `permission_matrix_seeded_test.go`、`permission_matrix_targets_test.go`：`seeded.issues`、`issue()`，`{issue_id}` 进 `underProject`；`permission_matrix_project_test.go`：`rowsByID` 的说明带上工作项和读的一行；
  - `permission_matrix_issues_test.go`：`getIssue` 一行（`readsTheIssue` 核对回答的工作项），`createIssue` 的格子的编号从 2 起；`permission_matrix_test.go` 接上种子。

**Tests:**
- 单元：`TestGetIssueReadsTheWorkItem`（不加锁、不开事务：读、判定；回答读到的）、`TestGetIssueRefusals`（不在、看不到 `issue.not_found`，角色不允许 403，读或判定的失败原样，没有调用者）；`create_issue_test.go` 的 `NewLocks` 多一个参数。
- 完整性和规则：`bootstrap` 的 `TestEveryActionHasARuleAndEveryRuleAnAction`、`TestEveryModuleDeclaresItsActionsOrHasNone`；`access/domain` 的规则表测试的 `issue.read` 一行。
- HTTP（`apitest.Main` 两个方向）：`TestGetIssue`（回答每个字段，没有的为 `null`，列表为 `[]`）、`TestGetIssueRefusals`。
- 组合：`TestPermissionMatrix` 的 `getIssue` 一行和 `createIssue` 的七行；`TestThePermissionMatrixCoversEveryOperation`。
- 前端：`error-messages.test.ts`（每个声明的码有文案）、`check:sync`（两份 `errors.json` 的键相同）。

- [ ] **Step 1: 契约，生成**

`api/modules/issue.yaml`（修改，2 处）：

````old api/modules/issue.yaml
          $ref: '#/components/responses/Problem'
````
````new api/modules/issue.yaml
          $ref: '#/components/responses/Problem'
  /api/v0/issues/{issue_id}:
    parameters:
      - $ref: '#/components/parameters/IssueID'
    get:
      operationId: getIssue
      tags: [issue]
      summary: Read a work item
      description: >-
        For the admins and members of its project, archived or not; an
        archived work item is read as any other. A work item that does not
        exist, is deleted, or whose project the caller does not see answers
        issue.not_found; one he sees the project of but may not read,
        forbidden.
      security: [{bearer: []}]
      x-problem-codes: [issue.not_found, forbidden]
      responses:
        '200':
          description: The work item.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IssueDetail'
        default:
          $ref: '#/components/responses/Problem'
````
````old api/modules/issue.yaml
      description: A project's id (Project.id).
````
````new api/modules/issue.yaml
      description: A project's id (Project.id).
      schema:
        type: string
        format: uuid
    IssueID:
      name: issue_id
      in: path
      required: true
      description: A work item's id (IssueDetail.id).
````

`api/openapi.yaml`（修改，1 处）：

````old api/openapi.yaml
    $ref: 'modules/issue.yaml#/paths/~1api~1v0~1projects~1{project_id}~1issues'
````
````new api/openapi.yaml
    $ref: 'modules/issue.yaml#/paths/~1api~1v0~1projects~1{project_id}~1issues'
  /api/v0/issues/{issue_id}:
    $ref: 'modules/issue.yaml#/paths/~1api~1v0~1issues~1{issue_id}'
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `c199fee32760c9e7ec66ce1e1314795ddec856cd7c0d71a217a703048355c348` | 3142 | `api/dist/openapi.yaml` |
| `9d513f0257e2b8a41606b37a235d9abfca698fee3dab459827cd17c67b63c22d` | 528 | `server/internal/modules/issue/adapter/http/gen/server.gen.go` |
| `71a5a28e0b9aca20bbf7f6341507ae6aa68a4d730c8a8520e490b54da41c2b64` | 3444 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/issue/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: 错误码、动作、规则和文案**

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"issue.create": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
````
````new server/internal/modules/access/domain/rules.go
	"issue.create": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
	// As issue.create; the project's guests are P2's (M4 design 3.4).
	"issue.read": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"issue.create": {allowed, allowed, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
````
````new server/internal/modules/access/domain/rules_test.go
	"issue.create": {allowed, allowed, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
	// As issue.create; the project's guests are P2's (M4 design 3.4).
	"issue.read": {allowed, allowed, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
````

`server/internal/modules/issue/domain/actions.go`（修改，2 处）：

````old server/internal/modules/issue/domain/actions.go
	ActionCreate shared.Action = "issue.create"
````
````new server/internal/modules/issue/domain/actions.go
	ActionCreate shared.Action = "issue.create"
	// ActionRead is reading a work item of a project.
	ActionRead shared.Action = "issue.read"
````
````old server/internal/modules/issue/domain/actions.go
	return []shared.Action{ActionCreate}
````
````new server/internal/modules/issue/domain/actions.go
	return []shared.Action{ActionCreate, ActionRead}
````

`server/internal/modules/issue/domain/errors.go`（修改，1 处）：

````old server/internal/modules/issue/domain/errors.go
	ErrProjectArchived = shared.NewError(shared.KindConflict, "project.archived", "The project is archived; unarchive it to change it.")
````
````new server/internal/modules/issue/domain/errors.go
	ErrProjectArchived = shared.NewError(shared.KindConflict, "project.archived", "The project is archived; unarchive it to change it.")
	// ErrIssueNotFound answers a work item that does not exist, is deleted,
	// or whose project the caller does not see (M4 design 5.3).
	ErrIssueNotFound = shared.NewError(shared.KindNotFound, "issue.not_found", "The work item does not exist, or you cannot see it.")
````

`web/apps/web/core/lib/error-messages.ts`（修改，1 处）：

````old web/apps/web/core/lib/error-messages.ts
  "project.label_not_found": "errors.project_label_not_found",
````
````new web/apps/web/core/lib/error-messages.ts
  "project.label_not_found": "errors.project_label_not_found",
  "issue.not_found": "errors.issue_not_found",
````

`web/packages/i18n/src/locales/en/errors.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/errors.json
    "project_label_not_found": "The label does not exist, or you cannot see its project.",
````
````new web/packages/i18n/src/locales/en/errors.json
    "project_label_not_found": "The label does not exist, or you cannot see its project.",
    "issue_not_found": "The work item does not exist, or you cannot see it.",
````

`web/packages/i18n/src/locales/zh-CN/errors.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/errors.json
    "project_label_not_found": "标签不存在，或者你看不到它所在的项目。",
````
````new web/packages/i18n/src/locales/zh-CN/errors.json
    "project_label_not_found": "标签不存在，或者你看不到它所在的项目。",
    "issue_not_found": "工作项不存在，或者你看不到它。",
````

- [ ] **Step 3: 读和用例**

`server/internal/modules/issue/app/create_issue_test.go`（修改，1 处）：

````old server/internal/modules/issue/app/create_issue_test.go
	uc := app.NewCreateIssue(app.CreateIssueDeps{Locks: app.NewLocks(w.project, w.auth), Issues: w.issues, Sequences: w.project,
````
````new server/internal/modules/issue/app/create_issue_test.go
	uc := app.NewCreateIssue(app.CreateIssueDeps{Locks: app.NewLocks(w.project, w.issues, w.auth), Issues: w.issues, Sequences: w.project,
````

`server/internal/modules/issue/app/get_issue.go`（新文件，33 行）：

````file server/internal/modules/issue/app/get_issue.go
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// GetIssue reads a work item: GET /api/v0/issues/{issue_id} (M4 design
// 5.1).
type GetIssue struct {
	locks Locks
}

// NewGetIssue returns the use case.
func NewGetIssue(locks Locks) *GetIssue {
	return &GetIssue{locks: locks}
}

// Execute reads the undeleted work item id, archived or not, with its
// assignees and labels, and decides issue.read on its project for the
// caller, without a lock or a transaction (Locks.see): a work item that is
// not there, or whose project the caller does not see, is
// issue.not_found; a role that may not read it, forbidden.
func (u *GetIssue) Execute(ctx context.Context, id uuid.UUID) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	return u.locks.see(ctx, actor, id, domain.ActionRead)
}
````

`server/internal/modules/issue/app/get_issue_test.go`（新文件，82 行）：

````file server/internal/modules/issue/app/get_issue_test.go
package app_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newGet is getIssue over web's world (newCreate's), with web's work item
// 7 stored in it: Fix, in Backlog, high, assigned to bob, labelled Bug,
// made and last written by carol, so that alice, who reads it, is neither.
func newGet() (*app.GetIssue, *world, uuid.UUID) {
	_, w := newCreate()
	id := uuid.NewV7()
	w.issues.issue = &app.IssueRow{ID: id, WorkspaceID: acme, ProjectID: web, SequenceID: 7, Name: "Fix", StateID: backlog.ID,
		Priority: domain.PriorityHigh, SortOrder: 65535, By: carol, At: clockNow}
	w.issues.assignees, w.issues.labels = []uuid.UUID{bob}, []uuid.UUID{bug}
	return app.NewGetIssue(app.NewLocks(w.project, w.issues, w.auth)), w, id
}

// A read, outside any transaction and without a lock (M4 design 3.5): the
// work item, then the decision on issue.read in its project, of its
// workspace; the answer is the work item as read.
func TestGetIssueReadsTheWorkItem(t *testing.T) {
	uc, w, id := newGet()
	got, err := uc.Execute(as(alice), id)
	want := domain.Issue{ID: id, WorkspaceID: acme, ProjectID: web, SequenceID: 7, Name: "Fix", StateID: backlog.ID, Priority: domain.PriorityHigh,
		SortOrder: 65535, AssigneeIDs: []uuid.UUID{bob}, LabelIDs: []uuid.UUID{bug}, CreatedByID: &carol, UpdatedByID: &carol, CreatedAt: clockNow,
		UpdatedAt: clockNow}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Execute() = %+v, %v; want %+v", got, err, want)
	}
	if calls := []string{line("GetIssue", id, "outside tx"), line("Authorize", alice, domain.ActionRead, acme, web, "outside tx")}; !slices.Equal(
		w.log.calls, calls) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
	}
}

// A work item that is not there, or whose project the caller does not
// see, is issue.not_found; a role that may not read it, forbidden; a read
// or a decision that fails, its failure as itself; a request without a
// caller, unauthenticated before any read.
func TestGetIssueRefusals(t *testing.T) {
	decided := line("Authorize", alice, domain.ActionRead, acme, web, "outside tx")
	for _, tt := range []struct {
		name  string
		set   func(w *world, id *uuid.UUID)
		ctx   context.Context
		want  error
		calls int // of read, decided
	}{
		{"no work item", func(_ *world, id *uuid.UUID) { *id = uuid.NewV7() }, as(alice), domain.ErrIssueNotFound, 1},
		{"a project not seen", func(w *world, _ *uuid.UUID) { w.auth.answers = []error{shared.ErrNotVisible} }, as(alice),
			domain.ErrIssueNotFound, 2},
		{"a role not allowed", func(w *world, _ *uuid.UUID) { w.auth.answers = []error{shared.Forbidden()} }, as(alice), shared.Forbidden(), 2},
		{"the read fails", func(w *world, _ *uuid.UUID) { w.log.fail = "GetIssue" }, as(alice), errDisk, 1},
		{"the decision fails", func(w *world, _ *uuid.UUID) { w.log.fail = "Authorize" }, as(alice), errDisk, 2},
		{"no caller", func(*world, *uuid.UUID) {}, context.Background(), shared.Unauthenticated(), 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id := newGet()
			tt.set(w, &id)
			got, err := uc.Execute(tt.ctx, id)
			var se, wantSE *shared.Error
			if !errors.Is(err, tt.want) && (!errors.As(err, &se) || !errors.As(tt.want, &wantSE) || se.Code != wantSE.Code) ||
				got.ID != (uuid.UUID{}) {
				t.Errorf("Execute() = %+v, %v; want no work item and %v", got, err, tt.want)
			}
			calls := []string{line("GetIssue", id, "outside tx"), decided}[:tt.calls]
			if !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
		})
	}
}
````

`server/internal/modules/issue/app/lock.go`（修改，7 处）：

````old server/internal/modules/issue/app/lock.go
	"uuid"

````
````new server/internal/modules/issue/app/lock.go
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
````
````old server/internal/modules/issue/app/lock.go
// issue.New builds one Locks for every write: a write holds no Authorizer
// of its own.
````
````new server/internal/modules/issue/app/lock.go
// A read of a work item takes the same decision on its project, without a
// lock (see). issue.New builds one Locks for every use case: none holds an
// Authorizer of its own.
````
````old server/internal/modules/issue/app/lock.go
	projects ProjectGate
````
````new server/internal/modules/issue/app/lock.go
	projects ProjectGate
	issues   IssueReader
````
````old server/internal/modules/issue/app/lock.go
// NewLocks returns the locks over project's gate and the Authorizer.
func NewLocks(projects ProjectGate, auth shared.Authorizer) Locks {
	return Locks{projects: projects, auth: auth}
````
````new server/internal/modules/issue/app/lock.go
// NewLocks returns the locks over project's gate, the work items' reads
// and the Authorizer.
func NewLocks(projects ProjectGate, issues IssueReader, auth shared.Authorizer) Locks {
	return Locks{projects: projects, issues: issues, auth: auth}
````
````old server/internal/modules/issue/app/lock.go
	return decide(ctx, l.auth, actor, w, workspaceID)
````
````new server/internal/modules/issue/app/lock.go
	return l.decide(ctx, actor, w.action, shared.Target{WorkspaceID: workspaceID, ProjectID: w.project}, w.notFound)
````
````old server/internal/modules/issue/app/lock.go
	if err := decide(ctx, l.auth, actor, w, p.WorkspaceID); err != nil {
````
````new server/internal/modules/issue/app/lock.go
	if err := l.decide(ctx, actor, w.action, shared.Target{WorkspaceID: p.WorkspaceID, ProjectID: w.project}, w.notFound); err != nil {
````
````old server/internal/modules/issue/app/lock.go
// decide asks the Authorizer for w's action on w's project of the
// workspace for actor. A project not visible to actor is w.notFound.
func decide(ctx context.Context, auth shared.Authorizer, actor shared.Actor, w write, workspaceID uuid.UUID) error {
	_, err := auth.Authorize(ctx, actor, w.action, shared.Target{WorkspaceID: workspaceID, ProjectID: w.project})
	if errors.Is(err, shared.ErrNotVisible) {
		return w.notFound
````
````new server/internal/modules/issue/app/lock.go
// see reads the undeleted work item id, archived or not, without a lock,
// and decides action on its project for actor (M4 design 3.4, 3.5): a work
// item that is not there, or whose project actor does not see, is
// issue.not_found; a role the rule does not allow is the Authorizer's
// shared.Forbidden. It answers the work item as read.
func (l Locks) see(ctx context.Context, actor shared.Actor, id uuid.UUID, action shared.Action) (domain.Issue, error) {
	i, found, err := l.issues.GetIssue(ctx, id)
	switch {
	case err != nil:
		return domain.Issue{}, err
	case !found:
		return domain.Issue{}, domain.ErrIssueNotFound
	}
	if err := l.decide(ctx, actor, action, shared.Target{WorkspaceID: i.WorkspaceID, ProjectID: i.ProjectID}, domain.ErrIssueNotFound); err != nil {
		return domain.Issue{}, err
	}
	return i, nil
}

// decide asks the Authorizer for action on the target for actor. A
// project not visible to actor is notFound.
func (l Locks) decide(ctx context.Context, actor shared.Actor, action shared.Action, target shared.Target, notFound error) error {
	_, err := l.auth.Authorize(ctx, actor, action, target)
	if errors.Is(err, shared.ErrNotVisible) {
		return notFound
````

`server/internal/modules/issue/app/ports.go`（修改，3 处）：

````old server/internal/modules/issue/app/ports.go
}

// IssueCreator is the work items' store as createIssue writes and reads
````
````new server/internal/modules/issue/app/ports.go
}

// IssueReader reads a work item, in the transaction ctx carries, or
// without one (M4 design 3.5, 5.2).
type IssueReader interface {
	// GetIssue reads the undeleted work item, archived or not, with its
	// assignees and labels; found is false when there is none.
	GetIssue(ctx context.Context, id uuid.UUID) (i domain.Issue, found bool, err error)
}

// IssueCreator is the work items' store as createIssue writes and reads
````
````old server/internal/modules/issue/app/ports.go
type IssueCreator interface {
````
````new server/internal/modules/issue/app/ports.go
type IssueCreator interface {
	IssueReader
````
````old server/internal/modules/issue/app/ports.go
	InsertActivities(ctx context.Context, rows []ActivityRow) error
	// GetIssue reads the undeleted work item with its assignees and
	// labels; found is false when there is none.
	GetIssue(ctx context.Context, id uuid.UUID) (i domain.Issue, found bool, err error)
````
````new server/internal/modules/issue/app/ports.go
	InsertActivities(ctx context.Context, rows []ActivityRow) error
````

`server/internal/modules/issue/module.go`（修改，2 处）：

````old server/internal/modules/issue/module.go
	store := postgresadapter.New(d.Pool)
````
````new server/internal/modules/issue/module.go
	store := postgresadapter.New(d.Pool)
	locks := app.NewLocks(d.Projects, store, d.Authorizer)
````
````old server/internal/modules/issue/module.go
			Locks: app.NewLocks(d.Projects, d.Authorizer), Issues: store, Sequences: d.Projects, States: d.Projects, Labels: d.Projects,
			Members: d.Projects, Profiles: d.Profiles, Tx: d.Tx, Clock: d.Clock,
		}),
````
````new server/internal/modules/issue/module.go
			Locks: locks, Issues: store, Sequences: d.Projects, States: d.Projects, Labels: d.Projects, Members: d.Projects,
			Profiles: d.Profiles, Tx: d.Tx, Clock: d.Clock,
		}),
		GetIssue: app.NewGetIssue(locks),
````

- [ ] **Step 4: HTTP**

`server/internal/modules/issue/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/issue/adapter/http/handler.go
}

// UseCases are the use cases behind the module's operations.
````
````new server/internal/modules/issue/adapter/http/handler.go
}

// GetIssueUseCase is app.GetIssue.
type GetIssueUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) (domain.Issue, error)
}

// UseCases are the use cases behind the module's operations.
````
````old server/internal/modules/issue/adapter/http/handler.go
	CreateIssue CreateIssueUseCase
````
````new server/internal/modules/issue/adapter/http/handler.go
	CreateIssue CreateIssueUseCase
	GetIssue    GetIssueUseCase
````

`server/internal/modules/issue/adapter/http/handler_test.go`（修改，2 处）：

````old server/internal/modules/issue/adapter/http/handler_test.go
// newServer serves the module's API over create, as bootstrap mounts it.
func newServer(t *testing.T, create *fakeCreate) http.Handler {
````
````new server/internal/modules/issue/adapter/http/handler_test.go
// fakeGet is getIssue: each call is recorded as "caller work item"; it
// answers answer, or err.
type fakeGet struct {
	calls  []string
	answer domain.Issue
	err    error
}

func (f *fakeGet) Execute(ctx context.Context, id uuid.UUID) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String())
	return f.answer, f.err
}

// newServer serves the module's API over the use cases uc, as bootstrap
// mounts it.
func newServer(t *testing.T, uc httpadapter.UseCases) http.Handler {
````
````old server/internal/modules/issue/adapter/http/handler_test.go
	httpadapter.Register(router, api, httpadapter.UseCases{CreateIssue: create})
````
````new server/internal/modules/issue/adapter/http/handler_test.go
	httpadapter.Register(router, api, uc)
````

`server/internal/modules/issue/adapter/http/issues.go`（修改，1 处）：

````old server/internal/modules/issue/adapter/http/issues.go
}

// issueDetail is i as the API shows it: each field that may be null null
````
````new server/internal/modules/issue/adapter/http/issues.go
}

// GetIssue serves GET /api/v0/issues/{issue_id}.
func (h handler) GetIssue(ctx context.Context, req gen.GetIssueRequestObject) (gen.GetIssueResponseObject, error) {
	i, err := h.uc.GetIssue.Execute(ctx, req.IssueID)
	if err != nil {
		return nil, err
	}
	return gen.GetIssue200JSONResponse(issueDetail(i)), nil
}

// issueDetail is i as the API shows it: each field that may be null null
````

`server/internal/modules/issue/adapter/http/issues_test.go`（修改，6 处）：

````old server/internal/modules/issue/adapter/http/issues_test.go
	"uuid"

````
````new server/internal/modules/issue/adapter/http/issues_test.go
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
````
````old server/internal/modules/issue/adapter/http/issues_test.go
		h := newServer(t, create)
````
````new server/internal/modules/issue/adapter/http/issues_test.go
		h := newServer(t, httpadapter.UseCases{CreateIssue: create})
````
````old server/internal/modules/issue/adapter/http/issues_test.go
	res, body := do(t, newServer(t, create), request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/issues", "alice",
````
````new server/internal/modules/issue/adapter/http/issues_test.go
	res, body := do(t, newServer(t, httpadapter.UseCases{CreateIssue: create}), request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/issues", "alice",
````
````old server/internal/modules/issue/adapter/http/issues_test.go
	create := &fakeCreate{}
	h := newServer(t, create)
````
````new server/internal/modules/issue/adapter/http/issues_test.go
	create := &fakeCreate{}
	h := newServer(t, httpadapter.UseCases{CreateIssue: create})
````
````old server/internal/modules/issue/adapter/http/issues_test.go
		h := newServer(t, &fakeCreate{err: tt.err})
````
````new server/internal/modules/issue/adapter/http/issues_test.go
		h := newServer(t, httpadapter.UseCases{CreateIssue: &fakeCreate{err: tt.err}})
````
````old server/internal/modules/issue/adapter/http/issues_test.go
			t.Errorf("%s: POST = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

````
````new server/internal/modules/issue/adapter/http/issues_test.go
			t.Errorf("%s: POST = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

// GET goes to the use case for the caller and the path's work item; the
// answer is 200 with the work item the use case answers, as POST shows it.
func TestGetIssue(t *testing.T) {
	for _, tt := range []struct {
		answer domain.Issue
		json   string
	}{{webSeven, webSevenJSON}, {webEight, webEightJSON}} {
		get := &fakeGet{answer: tt.answer}
		res, body := do(t, newServer(t, httpadapter.UseCases{GetIssue: get}), request(http.MethodGet, "/api/v0/issues/"+tt.answer.ID.String(),
			"alice", ""))
		if res.StatusCode != http.StatusOK || body != tt.json+"\n" {
			t.Errorf("GET = %d %s, want 200 %s", res.StatusCode, body, tt.json)
		}
		if want := []string{aliceID.String() + " " + tt.answer.ID.String()}; !slices.Equal(get.calls, want) {
			t.Errorf("GET: calls %q, want %q", get.calls, want)
		}
	}
}

// The use case's refusals, as the contract declares them, and its failure.
func TestGetIssueRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"no work item", domain.ErrIssueNotFound, http.StatusNotFound,
			`{"status":404,"code":"issue.not_found","title":"Not Found","detail":"The work item does not exist, or you cannot see it."}`},
		{"a project guest", shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"a failure", errGone, http.StatusInternalServerError,
			`{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
	} {
		h := newServer(t, httpadapter.UseCases{GetIssue: &fakeGet{err: tt.err}})
		res, body := do(t, h, request(http.MethodGet, "/api/v0/issues/"+webSeven.ID.String(), "alice", ""))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: GET = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

````

- [ ] **Step 5: 矩阵**

`server/internal/bootstrap/permission_matrix_issues_test.go`（修改，8 处）：

````old server/internal/bootstrap/permission_matrix_issues_test.go
// 9.2): creating one, under the project of each column. Each cell writes
// on a copy of its own, so each creation is its project's first.
````
````new server/internal/bootstrap/permission_matrix_issues_test.go
// 9.2): creating one, under the project of each column, and reading one,
// naming it by its id (/issues/{issue_id}): its column's project's seeded
// work item, matrixIssue. Each creating cell writes on a copy of its own,
// so each creation is its project's second, after the seeded one; the
// reading cells share one.

var cellIssueNotFound = cell{http.StatusNotFound, "issue.not_found"}

// issueRows are the work items the rows name by id: a column that does not
// see its project answers issue.not_found.
var issueRows = rowsByID{path: "issues", notFound: cellIssueNotFound, find: seeded.issue}
````
````old server/internal/bootstrap/permission_matrix_issues_test.go
			refusal: "assignee_ids[0] not_allowed"},
````
````new server/internal/bootstrap/permission_matrix_issues_test.go
			refusal: "assignee_ids[0] not_allowed"},
		// As createIssue; the project's guests are P2's (M4 design 3.4).
		{op: "getIssue", columns: projectColumns, request: issueRows.to(http.MethodGet, "", matrixIssue, ""),
			cells: issueRows.of(cellOK, cellOK, cellForbidden, cellOK, cellForbidden, cellForbidden), check: readsTheIssue},
		// An archived project's work items are read as any other's.
		{op: "getIssue", variant: "archived", columns: archivedColumns, request: issueRows.to(http.MethodGet, "", matrixIssue, ""),
			cells: issueRows.ofArchived(cellOK, cellForbidden), check: readsTheIssue},
````
````old server/internal/bootstrap/permission_matrix_issues_test.go
// createsTheIssue: the column's project's first work item, Fix the login,
````
````new server/internal/bootstrap/permission_matrix_issues_test.go
// createsTheIssue: the column's project's second work item, Fix the login,
````
````old server/internal/bootstrap/permission_matrix_issues_test.go
	if i.ProjectID != s.project(projectOf(c)) || i.SequenceID != 1 || i.Name != "Fix the login" || i.StateID != s.state(projectOf(c), "Backlog") ||
````
````new server/internal/bootstrap/permission_matrix_issues_test.go
	if i.ProjectID != s.project(projectOf(c)) || i.SequenceID != 2 || i.Name != "Fix the login" || i.StateID != s.state(projectOf(c), "Backlog") ||
````
````old server/internal/bootstrap/permission_matrix_issues_test.go
		t.Errorf("%s creates %s; want %s's work item 1, Fix the login, in Backlog, none, at 65535, by %s", c, answer, projectOf(c), me)
````
````new server/internal/bootstrap/permission_matrix_issues_test.go
		t.Errorf("%s creates %s; want %s's work item 2, Fix the login, in Backlog, none, at 65535, by %s", c, answer, projectOf(c), me)
````
````old server/internal/bootstrap/permission_matrix_issues_test.go
// createsTheGivenIssue: the column's project's first work item, Fix, in
````
````new server/internal/bootstrap/permission_matrix_issues_test.go
// createsTheGivenIssue: the column's project's second work item, Fix, in
````
````old server/internal/bootstrap/permission_matrix_issues_test.go
	if i.SequenceID != 1 || i.StateID != s.state(projectOf(c), "Done") || i.Priority != "high" || i.SortOrder != 65535 || i.CompletedAt == nil ||
````
````new server/internal/bootstrap/permission_matrix_issues_test.go
	if i.SequenceID != 2 || i.StateID != s.state(projectOf(c), "Done") || i.Priority != "high" || i.SortOrder != 65535 || i.CompletedAt == nil ||
````
````old server/internal/bootstrap/permission_matrix_issues_test.go
		t.Errorf("%s creates %s; want %s's work item 1 in Done, high, at 65535, completed, assigned to %s, labelled %s", c, answer,
			projectOf(c), pm, ui)
	}
}

````
````new server/internal/bootstrap/permission_matrix_issues_test.go
		t.Errorf("%s creates %s; want %s's work item 2 in Done, high, at 65535, completed, assigned to %s, labelled %s", c, answer,
			projectOf(c), pm, ui)
	}
}

// readsTheIssue: the column's project's seeded work item, its id, project,
// number, name, state, priority, assignees, labels and creator as
// issueSeed stored them: its first, named matrixIssue, in Todo, medium,
// assigned to PM's account and the removed member's, in id order, labelled
// Bug, made by acme's admin.
func readsTheIssue(t *testing.T, c caller, s seeded, answer string) {
	var i struct {
		ID          uuid.UUID   `json:"id"`
		ProjectID   uuid.UUID   `json:"project_id"`
		SequenceID  int         `json:"sequence_id"`
		Name        string      `json:"name"`
		StateID     uuid.UUID   `json:"state_id"`
		Priority    string      `json:"priority"`
		AssigneeIDs []uuid.UUID `json:"assignee_ids"`
		LabelIDs    []uuid.UUID `json:"label_ids"`
		CreatedByID *uuid.UUID  `json:"created_by_id"`
	}
	decodeAnswer(t, answer, &i)
	key := projectOf(c)
	assignees := slices.SortedFunc(slices.Values([]uuid.UUID{s.account(callerProjectMember), s.account(callerRemoved)}), uuid.UUID.Compare)
	if i.ID != s.issue(key, matrixIssue) || i.ProjectID != s.project(key) || i.SequenceID != 1 || i.Name != matrixIssue ||
		i.StateID != s.state(key, "Todo") || i.Priority != "medium" || !slices.Equal(i.AssigneeIDs, assignees) ||
		!slices.Equal(i.LabelIDs, []uuid.UUID{s.label(key, "Bug")}) || i.CreatedByID == nil || *i.CreatedByID != s.account(callerAdmin) {
		t.Errorf("%s reads %s; want %s's %s, its first, in Todo, medium, assigned to %v, labelled Bug, by acme's admin", c, answer, key,
			matrixIssue, assignees)
	}
}

````

`server/internal/bootstrap/permission_matrix_project_test.go`（修改，3 处）：

````old server/internal/bootstrap/permission_matrix_project_test.go
// rowsByID is a kind of row under a project that the writes of the matrix
// name by its id, /api/v0/<path>/{id}: a state (stateRows), a label
// (labelRows). notFound answers a caller who does not see the row's
// project, and find is the seeded row of a project by its name.
````
````new server/internal/bootstrap/permission_matrix_project_test.go
// rowsByID is a kind of row under a project that the matrix's rows name by
// its id, /api/v0/<path>/{id}: a state (stateRows), a label (labelRows), a
// work item (issueRows). notFound answers a caller who does not see the
// row's project, and find is the seeded row of a project by its name.
````
````old server/internal/bootstrap/permission_matrix_project_test.go
// of are the cells of a row of a write on one (projectCells), with
````
````new server/internal/bootstrap/permission_matrix_project_test.go
// of are the cells of a row on one, a write or a read (projectCells), with
````
````old server/internal/bootstrap/permission_matrix_project_test.go
// ofArchived are the cells of a row of a write on one of the archived
// project (archivedCells), with notFound.
````
````new server/internal/bootstrap/permission_matrix_project_test.go
// ofArchived are the cells of a row on one of the archived project, a write
// or a read (archivedCells), with notFound.
````

`server/internal/bootstrap/permission_matrix_seed_issues_test.go`（新文件，81 行）：

````file server/internal/bootstrap/permission_matrix_seed_issues_test.go
package bootstrap

import (
	"context"
	"slices"
	"strings"
	"uuid"

	issuepg "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	issuedomain "github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// matrixIssue is the work item prepareMatrix seeds in each project of
// matrixProjects through the issue store, by its workspace's admin, its
// project's first: in Todo, medium, assigned to PM's account and to the
// removed member's, whose membership of acme and of its public project
// ended since, and labelled Bug. gone's is deleted with it.
const matrixIssue = "Seeded"

// issueSeed writes the prepared work items through the issue store, each
// numbered through the project store, as createIssue numbers it.
type issueSeed struct {
	projectSeed
	store *issuepg.Store
}

// issues stores each project's matrixIssue with the id sd names: its
// number, its row, its assignees' rows in id order, and its label's.
func (s issueSeed) issues(sd seeded) {
	s.t.Helper()
	ctx := context.Background()
	assignees := slices.SortedFunc(slices.Values([]uuid.UUID{s.ids[callerProjectMember], s.ids[callerRemoved]}), uuid.UUID.Compare)
	for _, p := range matrixProjects {
		slug, _, _ := strings.Cut(p.key, "/")
		sequence, err := s.projectSeed.store.NextIssueSequence(ctx, s.projects[p.key])
		if err != nil {
			s.t.Fatal(err)
		}
		id, by := sd.issue(p.key, matrixIssue), s.ids[matrixAdmins[slug]]
		if err := s.store.InsertIssue(ctx, issueapp.IssueRow{ID: id, WorkspaceID: s.workspaces[slug], ProjectID: s.projects[p.key],
			SequenceID: sequence, Name: matrixIssue, StateID: sd.state(p.key, "Todo"), Priority: issuedomain.PriorityMedium, SortOrder: 65535,
			By: by, At: s.now}); err != nil {
			s.t.Fatal(err)
		}
		for _, link := range []struct {
			insert  func(context.Context, issueapp.LinkRows) error
			targets []uuid.UUID
		}{{s.store.InsertAssignees, assignees}, {s.store.InsertLabels, []uuid.UUID{sd.label(p.key, "Bug")}}} {
			rows := issueapp.LinkRows{WorkspaceID: s.workspaces[slug], ProjectID: s.projects[p.key], IssueID: id, Targets: link.targets, By: by,
				At: s.now}
			for range link.targets {
				rows.IDs = append(rows.IDs, uuid.NewV7())
			}
			if err := link.insert(ctx, rows); err != nil {
				s.t.Fatal(err)
			}
		}
	}
}

// seededIssues checks the work items the cells rest on, read back through
// the issue store by the ids sd names: each project's work item, in its
// project, its first, named matrixIssue, in Todo, medium, assigned to PM's
// account and the removed member's, in id order, labelled Bug; undeleted,
// but gone's, deleted with gone. One missing, or stored otherwise, would
// let a cell answer as it wants for another reason.
func (s issueSeed) seededIssues(sd seeded) {
	s.t.Helper()
	assignees := slices.SortedFunc(slices.Values([]uuid.UUID{s.ids[callerProjectMember], s.ids[callerRemoved]}), uuid.UUID.Compare)
	for _, p := range matrixProjects {
		gone := strings.HasPrefix(p.key, "gone/")
		i, found, err := s.store.GetIssue(context.Background(), sd.issue(p.key, matrixIssue))
		if err != nil || found == gone || !gone && (i.ProjectID != s.projects[p.key] || i.SequenceID != 1 || i.Name != matrixIssue ||
			i.StateID != sd.state(p.key, "Todo") || i.Priority != issuedomain.PriorityMedium || !slices.Equal(i.AssigneeIDs, assignees) ||
			!slices.Equal(i.LabelIDs, []uuid.UUID{sd.label(p.key, "Bug")})) {
			s.t.Fatalf("%s's work item = %+v, %v, %v; want %s, its first, in Todo, medium, assigned to %v, labelled Bug, deleted %v", p.key, i,
				found, err, matrixIssue, assignees, gone)
		}
	}
}
````

`server/internal/bootstrap/permission_matrix_seeded_test.go`（修改，6 处）：

````old server/internal/bootstrap/permission_matrix_seeded_test.go
// by the project's key and the column; each state and each label, by the
// project's key and its name; and each account, by its name in
// matrixAccounts, which prepareMatrix registers. t is the test that asks
// for them (in).
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
// by the project's key and the column; each state, each label and each
// work item, by the project's key and its name; and each account, by its
// name in matrixAccounts, which prepareMatrix registers. t is the test
// that asks for them (in).
````
````old server/internal/bootstrap/permission_matrix_seeded_test.go
	labels         map[string]uuid.UUID
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
	labels         map[string]uuid.UUID
	issues         map[string]uuid.UUID
````
````old server/internal/bootstrap/permission_matrix_seeded_test.go
// of matrixProjectMembers and each of matrixStates and matrixLabels in each
// project before prepareMatrix writes them, so that matrixViolations,
// without a database, sees the keys and the targets the cells will.
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
// of matrixProjectMembers, each of matrixStates and matrixLabels in each
// project and its work item, matrixIssue, before prepareMatrix writes them,
// so that matrixViolations, without a database, sees the keys and the
// targets the cells will.
````
````old server/internal/bootstrap/permission_matrix_seeded_test.go
		accounts: map[caller]uuid.UUID{}}
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
		issues: map[string]uuid.UUID{}, accounts: map[caller]uuid.UUID{}}
````
````old server/internal/bootstrap/permission_matrix_seeded_test.go
			s.labels[p.key+"|"+l.name] = uuid.NewV7()
		}
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
			s.labels[p.key+"|"+l.name] = uuid.NewV7()
		}
		s.issues[p.key+"|"+matrixIssue] = uuid.NewV7()
````
````old server/internal/bootstrap/permission_matrix_seeded_test.go
}

// projectOfRow is the key of the project of id, a row of rows, which are
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
}

// issue is the id of the work item name of the project key; one never
// seeded fails the test at once, as membership's does.
func (s seeded) issue(key, name string) uuid.UUID {
	id, ok := s.issues[key+"|"+name]
	if !ok {
		s.t.Helper()
		s.t.Fatalf("no work item %s of %s is seeded", name, key)
	}
	return id
}

// projectOfRow is the key of the project of id, a row of rows, which are
````

`server/internal/bootstrap/permission_matrix_targets_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_targets_test.go
	"{label_id}":          {"label", func(s seeded) map[string]uuid.UUID { return s.labels }},
````
````new server/internal/bootstrap/permission_matrix_targets_test.go
	"{label_id}":          {"label", func(s seeded) map[string]uuid.UUID { return s.labels }},
	"{issue_id}":          {"work item", func(s seeded) map[string]uuid.UUID { return s.issues }},
````
````old server/internal/bootstrap/permission_matrix_targets_test.go
// under a project (underProject: a project membership, a state, a label)
// one seeded in projectOf(c), each from a column of a project table
````
````new server/internal/bootstrap/permission_matrix_targets_test.go
// under a project (underProject: a project membership, a state, a label, a
// work item) one seeded in projectOf(c), each from a column of a project table
````

`server/internal/bootstrap/permission_matrix_test.go`（修改，5 处）：

````old server/internal/bootstrap/permission_matrix_test.go
	"uuid"

````
````new server/internal/bootstrap/permission_matrix_test.go
	"uuid"

	issuepg "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
````
````old server/internal/bootstrap/permission_matrix_test.go
// memberships through the workspace store, the projects, their
// memberships, their states and their labels through the project store,
// the deleted workspace through the API, and the membership states no
// store makes alone through SQL (prepareMatrix).
// The cells that only read share one copy of it, and each cell that writes
// gets a copy of its own (pgtest.NewDatabaseFrom), so no cell sees
// another's writes. Each module's rows are in a file of their own
// (permission_matrix_<module>_test.go): a phase that adds an operation adds
// its row there, and what the row needs prepared here and in
// permission_matrix_seeded_test.go; the labels' writer and check are
// beside their rows.
````
````new server/internal/bootstrap/permission_matrix_test.go
// memberships through the workspace store, the projects, their memberships,
// states and labels through the project store, their work items through
// the issue store, the deleted workspace through the API, and the
// membership states no store makes alone through SQL (prepareMatrix).
// The cells that only read share one copy of it, and each cell that writes
// gets a copy of its own (pgtest.NewDatabaseFrom), so no cell sees
// another's writes. Each module's rows are in a file of their own
// (permission_matrix_<module>_test.go): a phase that adds an operation adds
// its row there, and what the row needs prepared here and in
// permission_matrix_seeded_test.go; the labels' writer and check are beside
// their rows, the work items' in permission_matrix_seed_issues_test.go.
````
````old server/internal/bootstrap/permission_matrix_test.go
// matrixProjectMembers, matrixStates and matrixLabels, and acme's archived
// project archived. Through both stores, the removed member's removal;
// then, through the workspace store, the invitations of matrixInvitations.
// Through the project store, the memberships P5b's writes end (endings);
// through SQL, the membership states no store makes alone (partingStates).
// Through the API, gone deleted by its admin, which soft-deletes its
// memberships and its project, with its states and labels, with it; then
// the checks that the rows the cells rest on are there (preconditions), the
// states and the labels among them (seededStates, seededLabels).
````
````new server/internal/bootstrap/permission_matrix_test.go
// matrixProjectMembers, matrixStates and matrixLabels; through the issue
// store, each project's work item, matrixIssue (issueSeed); through the
// project store, acme's archived project archived. Through both stores,
// the removed member's removal; then, through the workspace store, the
// invitations of matrixInvitations. Through the project store, the
// memberships P5b's writes end (endings); through SQL, the membership
// states no store makes alone (partingStates). Through the API, gone
// deleted by its admin, which soft-deletes its memberships and its
// project, with its states, labels and work item, with it; then the checks
// that the rows the cells rest on are there (preconditions), the states,
// the labels and the work items among them (seededStates, seededLabels,
// seededIssues).
````
````old server/internal/bootstrap/permission_matrix_test.go
		projects.labels(s)
````
````new server/internal/bootstrap/permission_matrix_test.go
		projects.labels(s)
		issues := issueSeed{projectSeed: projects, store: issuepg.New(pool)}
		issues.issues(s)
````
````old server/internal/bootstrap/permission_matrix_test.go
		projects.seededLabels(pool, s)
````
````new server/internal/bootstrap/permission_matrix_test.go
		projects.seededLabels(pool, s)
		issues.seededIssues(s)
````

- [ ] **Step 6: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix|TestThePermissionMatrixCoversEveryOperation$|TestEveryModuleDeclaresItsActionsOrHasNone$|TestEveryActionHasARuleAndEveryRuleAnAction$' ./internal/bootstrap/`
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

- [ ] **Step 7: 提交**

```bash
git add api/modules/issue.yaml api/openapi.yaml server/internal/bootstrap/permission_matrix_issues_test.go server/internal/bootstrap/permission_matrix_project_test.go server/internal/bootstrap/permission_matrix_seed_issues_test.go server/internal/bootstrap/permission_matrix_seeded_test.go server/internal/bootstrap/permission_matrix_targets_test.go server/internal/bootstrap/permission_matrix_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/issue/adapter/http/handler.go server/internal/modules/issue/adapter/http/handler_test.go server/internal/modules/issue/adapter/http/issues.go server/internal/modules/issue/adapter/http/issues_test.go server/internal/modules/issue/app/create_issue_test.go server/internal/modules/issue/app/get_issue.go server/internal/modules/issue/app/get_issue_test.go server/internal/modules/issue/app/lock.go server/internal/modules/issue/app/ports.go server/internal/modules/issue/domain/actions.go server/internal/modules/issue/domain/errors.go server/internal/modules/issue/module.go web/apps/web/core/lib/error-messages.ts web/packages/i18n/src/locales/en/errors.json web/packages/i18n/src/locales/zh-CN/errors.json api/dist/openapi.yaml server/internal/modules/issue/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M4/P1): getIssue reads a work item for its project's admins and members

GET /api/v0/issues/{issue_id} reads the work item with every undeleted
assignee and label, without a lock or a transaction, and decides
issue.read on its project: a work item that is not there, or whose
project the caller does not see, is issue.not_found, with its copy; a
role that may not read it is forbidden. The matrix seeds a work item in
each project and holds the read of each column.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p1b.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；静态是 `make lint-web` 的文案和键的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T4.1` | `getIssue` 不判定 `issue.read` 就读 | `internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `getIssue` 各格（`make test`） | 单元；组合 |
| `T4.2` | `issue.read` 在 P2 的访客规则之前就放进项目的访客 | `internal/modules/access/domain` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `getIssue` 各格（`make test`） | 单元；组合 |
| `T4.3` | 工作项不在时答项目的 404，不是 `issue.not_found` | `internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `getIssue` 各格（`make test`） | 单元；组合 |
| `T4.4` | 页面没有 `issue.not_found` 的文案 | `error-messages.test.ts`（`make test-web`） | 单元 |
| `T4.5` | 中文的文案没有 `issue_not_found` | `web/packages/i18n` 的 `check:sync`（`make lint-web`） | 守卫 |
| `T4.6` | GET 回答的负责人总是空的列表 | `internal/modules/issue/adapter/http` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `getIssue` 各格（`make test`） | 单元；组合 |

---

### Task 5: 按资源寻址的写和 `updateIssue` 的用例

**Files:**
- Create: `server/internal/modules/issue/app/update_issue.go`、`server/internal/modules/issue/app/update_issue_test.go`、`server/internal/modules/issue/app/writes.go`
- Modify: `server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/issue/app/create_issue.go`、`server/internal/modules/issue/app/create_issue_test.go`、`server/internal/modules/issue/app/fakes_test.go`、`server/internal/modules/issue/app/get_issue_test.go`、`server/internal/modules/issue/app/lock.go`、`server/internal/modules/issue/app/ports.go`、`server/internal/modules/issue/domain/actions.go`、`server/internal/modules/issue/domain/issue.go`、`server/internal/modules/issue/domain/patch.go`、`server/internal/modules/issue/domain/patch_test.go`、`server/internal/modules/issue/module.go`

**Interfaces:**（spec 2.7、第 3 节第 3–7、12、16 条；M4 设计 3.5、3.6、3.7、3.12、3.13、6.7）
- `issue/domain`：
  - `patch.go`：`Unassignable(asked, added []uuid.UUID, roles map[uuid.UUID]shared.Role) []shared.FieldError`：给出的负责人中新加的、不能指派的（`CanAssign`），各在它的下标；留下的不查；
  - `issue.go`：`unassignable(i)` 与 `Assignees` 共用；
  - `actions.go`：`ActionUpdate`（`issue.update`）。
- `access/domain/rules.go`：`issue.update` 一行（项目级，管理员、成员）。
- `issue/app/lock.go`（整个文件）：
  - `Locks{projects, issues, auth, tx}`、`NewLocks(projects, issues, auth, tx)`：事务移进 `Locks`；
  - `write{project, issue *uuid.UUID, action, notFound}`、`held{project LockedProject, issue domain.Issue}`；
  - `run(ctx, actor, w, fn)`：`precheck` → `WithinTx`：`lockAndDecide` → `fn(ctx, h)`；
  - `precheck`：指名工作项的写 `see`（得到项目），否则读项目、判定；
  - `lockAndDecide`：项目 N → 指名工作项时锁下重读它（没有或不在这个项目：`w.notFound`）→ 判定（最终的）；
  - `see`、`decide` 照 Task 4。
- `writes.go`（新）：`stamp{workspace, project, issue, by, at}` 和它的 `links`、`unlinks`、`activity`；`assignees`（显示名，缺一个是错误）；`readBack`（写之后读不回来是错误）；`pickState`（设计 3.7 的状态规则：要的状态是项目的、不是分诊的，没要时默认的；创建和修改共用，从 `create_issue.go` 移来）。
- `create_issue.go`：改到 `Locks.run` 上（`CreateIssueDeps` 没有 `Tx`），`stamp`、`assignees`、`readBack`、`pickState` 来自 `writes.go`；调用的次序不变。
- `ports.go`：`IssueUpdater`（嵌入 `IssueReader`；`UpdateIssue`、`RemoveAssignees`、`RemoveLabels`，和 `IssueCreator` 的四个插入：负责人、标签、关注、动态）。
- `update_issue.go`（新）：`UpdateIssueDeps`、`UpdateIssue`、`NewUpdateIssue`、`Execute(ctx, id, p)`：`CheckIssuePatch` → `Locks.run(write{issue: &id, action: issue.update, notFound: issue.not_found})`：已归档 409 → `ChangesOf`，`None` 时不写、回答锁下读到的 → `check`（状态经 `pickState`，与创建同一条规则；新负责人的角色、加和去的标签，问题一次报出；读出动态要的名称）→ `Clock.Now()` → `write`（工作项，完成时间按新状态的组 → 负责人先去后加 → 标签先去后加 → 新负责人的关注 → 显示名 → 动态）→ `readBack`。
- `issue/module.go`：`New` 建 `Locks` 时带上 `Tx`，创建的用例不再带它（`UpdateIssue` 在 Task 6 接到 HTTP）。

**Tests:**（单元，假实现 `fakes_test.go` 按次序记下调用；`changed`、`writer()`、`unlink`、`fakeProject.waited`）
- `TestUpdateIssueWritesTheChangesAndTheirRows`：取锁的一步（锁之前读工作项、判定，事务里项目 N、重读、判定）→ 新状态、新负责人的角色、加和去的标签 → 时刻 → 工作项 → 去、加负责人 → 去、加标签（各按 `id`）→ 新负责人的关注 → 显示名 → 动态（3.13 的次序），全部 alice 在这一刻写；新加两个负责人（carol、dave：每个都指派、关注、写进动态，不只第一个）、去掉一个（bob）；工作项留着的标签 Docs 不读、它的行不再写（只去掉 Bug、只加 UI）；已离开的 out 留着、不查；名称去掉空白；到 Done 时完成。
- `TestUpdateIssueThatChangesNothingWritesNothing`：什么都不给、或每个字段原样（集合换了次序）：取锁和判定照旧，不写、不读时钟，回答锁下读到的。
- `TestUpdateIssueCompletesTheWorkItemByItsState`：完成组的状态（含从另一个完成组的状态）完成于这一刻，别的组不完成（已取消组的也不完成，从完成组移进去时不再完成），状态不变时完成时间不动。
- `TestUpdateIssueRefusesBeforeAnyLock`：不在、看不到 `issue.not_found`，角色不允许 403，都不取锁；不合规则的值在读工作项之前拒绝。
- `TestUpdateIssueRefusesUnderTheLock`：等锁期间项目被删除、工作项被删除或换了项目、项目看不到了：`issue.not_found`；角色不再允许 403；已归档 409（什么都不改的修改也是）；另一个项目的状态、分诊状态、新加的访客、另一个项目的标签：一个 422，各在它在修改中的位置（标签是它在 `label_ids` 中的下标，不是在加的标签中的）。
- `TestUpdateIssueStopsAtAFailingPort`：每个端口的失败原样返回，之后不再调用，预检通过之后锁下的重读、判定的失败也是；不可能的情况也是错误（工作项的状态不是项目的、去掉的标签不是项目的、没有显示名的负责人、写之后读不回来）。
- `TestUnassignable`（领域）；`issue.update` 的规则行；创建、读的测试照旧通过（假实现的 `Locks` 带事务）；创建和修改的测试共用拒绝的检查 `wantRefusal`（各自核对什么都没有写）和端口失败的检查 `wantStoppedAt`（`create_issue_test.go`）；状态的规则只有一处（`pickState`），去掉分诊的条件，创建和修改的 `RefusesUnderTheLock` 都失败。

- [ ] **Step 1: 领域、动作和规则**

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"issue.read": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
````
````new server/internal/modules/access/domain/rules.go
	"issue.read": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
	// As issue.create (M4 design 3.4).
	"issue.update": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"issue.read": {allowed, allowed, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
````
````new server/internal/modules/access/domain/rules_test.go
	"issue.read": {allowed, allowed, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
	// As issue.create (M4 design 3.4).
	"issue.update": {allowed, allowed, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
````

`server/internal/modules/issue/domain/actions.go`（修改，2 处）：

````old server/internal/modules/issue/domain/actions.go
	ActionRead shared.Action = "issue.read"
````
````new server/internal/modules/issue/domain/actions.go
	ActionRead shared.Action = "issue.read"
	// ActionUpdate is changing a work item of a project.
	ActionUpdate shared.Action = "issue.update"
````
````old server/internal/modules/issue/domain/actions.go
	return []shared.Action{ActionCreate, ActionRead}
````
````new server/internal/modules/issue/domain/actions.go
	return []shared.Action{ActionCreate, ActionRead, ActionUpdate}
````

`server/internal/modules/issue/domain/issue.go`（修改，2 处）：

````old server/internal/modules/issue/domain/issue.go
			refused = append(refused, shared.FieldError{Field: fmt.Sprintf("assignee_ids[%d]", i), Code: shared.FieldNotAllowed,
				Message: "must be an active member of the project who is not its guest"})
````
````new server/internal/modules/issue/domain/issue.go
			refused = append(refused, unassignable(i))
````
````old server/internal/modules/issue/domain/issue.go
	return slices.SortedFunc(slices.Values(asked), uuid.UUID.Compare), nil
````
````new server/internal/modules/issue/domain/issue.go
	return slices.SortedFunc(slices.Values(asked), uuid.UUID.Compare), nil
}

// unassignable is the problem of the i-th account asked for, who may not
// be assigned (CanAssign).
func unassignable(i int) shared.FieldError {
	return shared.FieldError{Field: fmt.Sprintf("assignee_ids[%d]", i), Code: shared.FieldNotAllowed,
		Message: "must be an active member of the project who is not its guest"}
````

`server/internal/modules/issue/domain/patch.go`（修改，1 处）：

````old server/internal/modules/issue/domain/patch.go
// None reports whether c changes nothing: then the update writes no row
// (M4/P1b spec 3).
````
````new server/internal/modules/issue/domain/patch.go
// Unassignable are the problems of the accounts of asked, a patch's
// assignee_ids, that it adds to the work item (added, Changes.Assignees'
// Added) and that may not be assigned (CanAssign), from roles, their
// active project roles, each at its index in asked (M4 design 3.6): an
// account who has left the project, or is its guest, is not assigned anew;
// one the work item has stays.
func Unassignable(asked, added []uuid.UUID, roles map[uuid.UUID]shared.Role) []shared.FieldError {
	var refused []shared.FieldError
	for i, id := range asked {
		if slices.Contains(added, id) && !CanAssign(roles[id]) {
			refused = append(refused, unassignable(i))
		}
	}
	return refused
}

// None reports whether c changes nothing: then the update writes no row.
````

`server/internal/modules/issue/domain/patch_test.go`（修改，2 处）：

````old server/internal/modules/issue/domain/patch_test.go
	"errors"
````
````new server/internal/modules/issue/domain/patch_test.go
	"errors"
	"fmt"
````
````old server/internal/modules/issue/domain/patch_test.go
}

// Changes change nothing only when each of them is empty: any one of them
````
````new server/internal/modules/issue/domain/patch_test.go
}

// Of the accounts asked for, only those added are checked (M4 design 3.6):
// an account added who is the project's guest, or no active member of it,
// is a problem at its index in what is asked; one the work item keeps is
// none, whatever his role, and nor is one added who may be assigned.
func TestUnassignable(t *testing.T) {
	people := ids(4) // in id order: ann, ben, gus, out
	ann, ben, gus, out := people[0], people[1], people[2], people[3]
	roles := map[uuid.UUID]shared.Role{ann: shared.RoleAdmin, ben: shared.RoleMember, gus: shared.RoleGuest}
	refused := func(i int) shared.FieldError {
		return shared.FieldError{Field: fmt.Sprintf("assignee_ids[%d]", i), Code: shared.FieldNotAllowed,
			Message: "must be an active member of the project who is not its guest"}
	}
	tests := []struct {
		name         string
		asked, added []uuid.UUID
		want         []shared.FieldError
	}{
		{"none added", []uuid.UUID{gus, out}, nil, nil},
		{"an admin and a member added", []uuid.UUID{ben, ann}, []uuid.UUID{ann, ben}, nil},
		{"a guest and one who left added, both kept too", []uuid.UUID{ann, out, gus, ben}, []uuid.UUID{gus, out},
			[]shared.FieldError{refused(1), refused(2)}},
		{"a guest kept, one who left added", []uuid.UUID{gus, out}, []uuid.UUID{out}, []shared.FieldError{refused(1)}},
	}
	for _, tt := range tests {
		if got := Unassignable(tt.asked, tt.added, roles); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: Unassignable() = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

// Changes change nothing only when each of them is empty: any one of them
````

- [ ] **Step 2: 取锁的一步和写共用的行**

`server/internal/modules/issue/app/lock.go`（整个文件，161 行）：

````whole server/internal/modules/issue/app/lock.go
package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Locks is the one way a write on the work-item side takes its lock and
// its decision (M4 design 3.5, decision point 1: B). Before any lock and
// outside the write's transaction, the precheck: the project, or the work
// item the write names and through it its project, read without a lock,
// and the decision on it, which answers every refusal there, so a caller
// who would be refused takes no lock at all. Then, in the write's
// transaction, the project's row FOR NO KEY UPDATE, its workspace's row
// not locked: the write's first lock, which every write on the project's
// work items, states, labels and memberships also takes, so they run one
// after another; the work item read again under it; and the decision again
// under it, which is final: a removal or a demotion that committed while
// the write waited is seen. A read of a work item takes the same decision
// on its project, without a lock (see). issue.New builds one Locks for
// every use case: none holds an Authorizer or a transaction of its own.
type Locks struct {
	projects ProjectGate
	issues   IssueReader
	auth     shared.Authorizer
	tx       shared.TxManager
}

// NewLocks returns the locks over project's gate, the work items' reads,
// the Authorizer and the transactions.
func NewLocks(projects ProjectGate, issues IssueReader, auth shared.Authorizer, tx shared.TxManager) Locks {
	return Locks{projects: projects, issues: issues, auth: auth, tx: tx}
}

// write is a write on the work-item side, as Locks takes its lock: the
// project it is on, or the work item it names, issue, whose project it is
// then on; the action it decides; and the 404 of what the caller named.
type write struct {
	project  uuid.UUID
	issue    *uuid.UUID
	action   shared.Action
	notFound error
}

// held is what a write holds under its lock: its project, locked, and the
// work item it names, read under the lock; zero for a write on a project.
type held struct {
	project LockedProject
	issue   domain.Issue
}

// run takes w's precheck, before any lock and outside any transaction,
// then in one transaction w's lock and decision, and runs fn in it with
// what they hold (M4 design 3.5): w's whole write. A refusal of either is
// its answer, and fn does not run.
func (l Locks) run(ctx context.Context, actor shared.Actor, w write, fn func(ctx context.Context, h held) error) error {
	w, err := l.precheck(ctx, actor, w)
	if err != nil {
		return err
	}
	return l.tx.WithinTx(ctx, func(ctx context.Context) error {
		h, err := l.lockAndDecide(ctx, actor, w)
		if err != nil {
			return err
		}
		return fn(ctx, h)
	})
}

// precheck decides w's action for actor before any lock and outside any
// transaction (M4 design 3.5), and answers w on its project. For a write
// that names a work item: the work item read without a lock and the
// decision on its project (see); a work item that is not there, or whose
// project actor does not see, is issue.not_found. For one on a project:
// the project read without a lock and the decision on it; a project that
// is not there, or not visible to actor, is w.notFound. A role the rule
// does not allow is the Authorizer's shared.Forbidden. What it lets
// through, lockAndDecide decides again.
func (l Locks) precheck(ctx context.Context, actor shared.Actor, w write) (write, error) {
	if w.issue != nil {
		i, err := l.see(ctx, actor, *w.issue, w.action)
		if err != nil {
			return write{}, err
		}
		w.project = i.ProjectID
		return w, nil
	}
	workspaceID, found, err := l.projects.Find(ctx, w.project)
	switch {
	case err != nil:
		return write{}, err
	case !found:
		return write{}, w.notFound
	}
	return w, l.decide(ctx, actor, w.action, shared.Target{WorkspaceID: workspaceID, ProjectID: w.project}, w.notFound)
}

// lockAndDecide takes w's lock and decides w's action under it, in the
// transaction ctx carries: the project's row FOR NO KEY UPDATE; the work
// item w names read again; then the decision on the locked project. It
// answers what they hold. A project deleted while the lock waited is
// w.notFound, and so is a work item deleted meanwhile, or no longer in the
// project, and a project no longer visible to actor; a role the rule no
// longer allows is the Authorizer's shared.Forbidden.
func (l Locks) lockAndDecide(ctx context.Context, actor shared.Actor, w write) (held, error) {
	p, found, err := l.projects.Lock(ctx, w.project)
	switch {
	case err != nil:
		return held{}, err
	case !found:
		return held{}, w.notFound
	}
	h := held{project: p}
	if w.issue != nil {
		i, found, err := l.issues.GetIssue(ctx, *w.issue)
		switch {
		case err != nil:
			return held{}, err
		case !found || i.ProjectID != w.project:
			return held{}, w.notFound
		}
		h.issue = i
	}
	if err := l.decide(ctx, actor, w.action, shared.Target{WorkspaceID: p.WorkspaceID, ProjectID: w.project}, w.notFound); err != nil {
		return held{}, err
	}
	return h, nil
}

// see reads the undeleted work item id, archived or not, without a lock,
// and decides action on its project for actor (M4 design 3.4, 3.5): a work
// item that is not there, or whose project actor does not see, is
// issue.not_found; a role the rule does not allow is the Authorizer's
// shared.Forbidden. It answers the work item as read.
func (l Locks) see(ctx context.Context, actor shared.Actor, id uuid.UUID, action shared.Action) (domain.Issue, error) {
	i, found, err := l.issues.GetIssue(ctx, id)
	switch {
	case err != nil:
		return domain.Issue{}, err
	case !found:
		return domain.Issue{}, domain.ErrIssueNotFound
	}
	if err := l.decide(ctx, actor, action, shared.Target{WorkspaceID: i.WorkspaceID, ProjectID: i.ProjectID}, domain.ErrIssueNotFound); err != nil {
		return domain.Issue{}, err
	}
	return i, nil
}

// decide asks the Authorizer for action on the target for actor. A
// project not visible to actor is notFound.
func (l Locks) decide(ctx context.Context, actor shared.Actor, action shared.Action, target shared.Target, notFound error) error {
	_, err := l.auth.Authorize(ctx, actor, action, target)
	if errors.Is(err, shared.ErrNotVisible) {
		return notFound
	}
	return err
}
````

`server/internal/modules/issue/app/ports.go`（修改，1 处）：

````old server/internal/modules/issue/app/ports.go
	InsertActivities(ctx context.Context, rows []ActivityRow) error
}

````
````new server/internal/modules/issue/app/ports.go
	InsertActivities(ctx context.Context, rows []ActivityRow) error
}

// IssueUpdater is the work items' store as updateIssue writes and reads
// them, in the transaction ctx carries (M4 design 3.5).
type IssueUpdater interface {
	IssueReader
	// UpdateIssue writes the work item's fields.
	UpdateIssue(ctx context.Context, f IssueFields) error
	InsertAssignees(ctx context.Context, r LinkRows) error
	// RemoveAssignees soft-deletes the work item's undeleted assignees'
	// rows of the accounts r names.
	RemoveAssignees(ctx context.Context, r UnlinkRows) error
	InsertLabels(ctx context.Context, r LinkRows) error
	// RemoveLabels soft-deletes the work item's undeleted labels' rows of
	// the labels r names.
	RemoveLabels(ctx context.Context, r UnlinkRows) error
	// InsertSubscribers inserts the rows, but of an account that
	// subscribes to the work item already.
	InsertSubscribers(ctx context.Context, r LinkRows) error
	// InsertActivities inserts the rows in their order.
	InsertActivities(ctx context.Context, rows []ActivityRow) error
}

````

`server/internal/modules/issue/app/writes.go`（新文件，101 行）：

````file server/internal/modules/issue/app/writes.go
package app

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// The pieces every write on a work item's rows shares (M4 design 3.5,
// 3.7): the state it may put the work item in, where its rows go, the
// accounts its activity names, and its answer.

// pickState is the state of states asked for, when it is there and not
// the triage state, or the default state when none is asked for (M4
// design 3.7); found is false when there is no such state. A creation's
// state and a change's new state are picked by it.
func pickState(states []StateRef, asked *uuid.UUID) (StateRef, bool) {
	for _, s := range states {
		if asked == nil && s.Default || asked != nil && s.ID == *asked && s.Group != domain.GroupTriage {
			return s, true
		}
	}
	return StateRef{}, false
}

// stamp is where a write's rows under a work item go, and whose and when
// they are: the work item's workspace, project and id; the caller, who
// writes them, at the write's one moment.
type stamp struct {
	workspace, project, issue uuid.UUID
	by                        uuid.UUID
	at                        time.Time
}

// links are new rows under the work item of each of targets, in their
// order, each with a fresh id.
func (s stamp) links(targets []uuid.UUID) LinkRows {
	r := LinkRows{WorkspaceID: s.workspace, ProjectID: s.project, IssueID: s.issue, Targets: targets, By: s.by, At: s.at}
	for range targets {
		r.IDs = append(r.IDs, uuid.NewV7())
	}
	return r
}

// unlinks are the work item's rows of each of targets that a write
// removes.
func (s stamp) unlinks(targets []uuid.UUID) UnlinkRows {
	return UnlinkRows{IssueID: s.issue, Targets: targets, By: s.by, At: s.at}
}

// activity are the rows of the work item's activity, in their order, each
// with a fresh id, so that their ids ascend in it.
func (s stamp) activity(activities []domain.Activity) []ActivityRow {
	rows := make([]ActivityRow, len(activities))
	for i, a := range activities {
		rows[i] = ActivityRow{ID: uuid.NewV7(), WorkspaceID: s.workspace, ProjectID: s.project, IssueID: s.issue, Activity: a, ActorID: s.by,
			At: s.at}
	}
	return rows
}

// assignees is the accounts ids with their display names, in their order,
// read through profiles; none for none. An account without one is an
// error: an assignee is, or was, a member of the project, and accounts are
// never deleted.
func assignees(ctx context.Context, profiles PublicProfiles, ids []uuid.UUID) ([]domain.Assignee, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	names, err := profiles.DisplayNames(ctx, ids)
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

// readBack reads the work item id back as stored, in the transaction ctx
// carries: a write's answer. A work item not there after its write is an
// internal error, not a 404.
func readBack(ctx context.Context, issues IssueReader, id uuid.UUID) (domain.Issue, error) {
	i, found, err := issues.GetIssue(ctx, id)
	switch {
	case err != nil:
		return domain.Issue{}, err
	case !found:
		return domain.Issue{}, errors.New("work item " + id.String() + " is not there after its write")
	}
	return i, nil
}
````

- [ ] **Step 3: 创建改到取锁的一步上**

`server/internal/modules/issue/app/create_issue.go`（修改，14 处）：

````old server/internal/modules/issue/app/create_issue.go
	"context"
	"errors"
````
````new server/internal/modules/issue/app/create_issue.go
	"context"
````
````old server/internal/modules/issue/app/create_issue.go
	Profiles  PublicProfiles
	Tx        shared.TxManager
````
````new server/internal/modules/issue/app/create_issue.go
	Profiles  PublicProfiles
````
````old server/internal/modules/issue/app/create_issue.go
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
````
````new server/internal/modules/issue/app/create_issue.go
// Execute checks in (domain.CheckNewIssue), before any read; then Locks
// runs the write (M4 design 3.5): its precheck, before any lock; then in
// one transaction, the project FOR NO KEY UPDATE and the decision on
// issue.create, and under them, in this order: an archived project refuses
// it (409 project.archived); the dependent rows (check), every problem in
// one 422; the project's next number; the clock; the sort order after the
// greatest of the state's work items; the work item; its assignees, the
// default one too; its labels; its subscribers, the caller and the
// assignees, in id order; its activity: the creation, then each assignee
// by his display name (domain.CreationActivity). All of it is the
// caller's at that moment. The answer is the work item as stored, read
// back in the transaction.
````
````old server/internal/modules/issue/app/create_issue.go
	w := write{project: projectID, action: domain.ActionCreate, notFound: domain.ErrProjectNotFound}
	if err := u.d.Locks.precheck(ctx, actor, w); err != nil {
		return domain.Issue{}, err
	}
````
````new server/internal/modules/issue/app/create_issue.go
	w := write{project: projectID, action: domain.ActionCreate, notFound: domain.ErrProjectNotFound}
````
````old server/internal/modules/issue/app/create_issue.go
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		project, err := u.d.Locks.lockAndDecide(ctx, actor, w)
		if err != nil {
			return err
		}
````
````new server/internal/modules/issue/app/create_issue.go
	err = u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		project := h.project
````
````old server/internal/modules/issue/app/create_issue.go
		created, err = u.answer(ctx, row.ID)
````
````new server/internal/modules/issue/app/create_issue.go
		created, err = readBack(ctx, u.d.Issues, row.ID)
````
````old server/internal/modules/issue/app/create_issue.go
	links := func(targets []uuid.UUID) LinkRows {
		r := LinkRows{WorkspaceID: row.WorkspaceID, ProjectID: row.ProjectID, IssueID: row.ID, Targets: targets, By: row.By, At: row.At}
		for range targets {
			r.IDs = append(r.IDs, uuid.NewV7())
		}
		return r
	}
````
````new server/internal/modules/issue/app/create_issue.go
	s := stamp{workspace: row.WorkspaceID, project: row.ProjectID, issue: row.ID, by: row.By, at: row.At}
````
````old server/internal/modules/issue/app/create_issue.go
		if err := u.d.Issues.InsertAssignees(ctx, links(c.assignees)); err != nil {
````
````new server/internal/modules/issue/app/create_issue.go
		if err := u.d.Issues.InsertAssignees(ctx, s.links(c.assignees)); err != nil {
````
````old server/internal/modules/issue/app/create_issue.go
		if err := u.d.Issues.InsertLabels(ctx, links(c.labels)); err != nil {
````
````new server/internal/modules/issue/app/create_issue.go
		if err := u.d.Issues.InsertLabels(ctx, s.links(c.labels)); err != nil {
````
````old server/internal/modules/issue/app/create_issue.go
	if err := u.d.Issues.InsertSubscribers(ctx, links(subscribers)); err != nil {
````
````new server/internal/modules/issue/app/create_issue.go
	if err := u.d.Issues.InsertSubscribers(ctx, s.links(subscribers)); err != nil {
````
````old server/internal/modules/issue/app/create_issue.go
	assignees, err := u.named(ctx, c.assignees)
````
````new server/internal/modules/issue/app/create_issue.go
	accounts, err := assignees(ctx, u.d.Profiles, c.assignees)
````
````old server/internal/modules/issue/app/create_issue.go
	var rows []ActivityRow
	for _, a := range domain.CreationActivity(assignees) {
		rows = append(rows, ActivityRow{ID: uuid.NewV7(), WorkspaceID: row.WorkspaceID, ProjectID: row.ProjectID, IssueID: row.ID, Activity: a,
			ActorID: row.By, At: row.At})
	}
	return u.d.Issues.InsertActivities(ctx, rows)
````
````new server/internal/modules/issue/app/create_issue.go
	return u.d.Issues.InsertActivities(ctx, s.activity(domain.CreationActivity(accounts)))
````
````old server/internal/modules/issue/app/create_issue.go

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
````new server/internal/modules/issue/app/create_issue.go

````
````old server/internal/modules/issue/app/create_issue.go
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
````
````new server/internal/modules/issue/app/create_issue.go
	return c, nil
}
````

`server/internal/modules/issue/app/create_issue_test.go`（修改，11 处）：

````old server/internal/modules/issue/app/create_issue_test.go
	backlog                     = app.StateRef{ID: uuid.NewV7(), Group: "backlog", Default: true}
	done                        = app.StateRef{ID: uuid.NewV7(), Group: domain.GroupCompleted}
	triage                      = app.StateRef{ID: uuid.NewV7(), Group: domain.GroupTriage}
````
````new server/internal/modules/issue/app/create_issue_test.go
	backlog                     = app.StateRef{ID: uuid.NewV7(), Name: "Backlog", Group: "backlog", Default: true}
	done                        = app.StateRef{ID: uuid.NewV7(), Name: "Done", Group: domain.GroupCompleted}
	triage                      = app.StateRef{ID: uuid.NewV7(), Name: "Triage", Group: domain.GroupTriage}
````
````old server/internal/modules/issue/app/create_issue_test.go
// world is web as createIssue's ports see it: its states, the greatest
````
````new server/internal/modules/issue/app/create_issue_test.go
// world is web as the use cases' ports see it: its states, the greatest
````
````old server/internal/modules/issue/app/create_issue_test.go
// call it is not told to refuse.
````
````new server/internal/modules/issue/app/create_issue_test.go
// call it is not told to refuse; and the locks over them.
````
````old server/internal/modules/issue/app/create_issue_test.go
	profiles fakeProfiles
````
````new server/internal/modules/issue/app/create_issue_test.go
	profiles fakeProfiles
	locks    app.Locks
````
````old server/internal/modules/issue/app/create_issue_test.go
	uc := app.NewCreateIssue(app.CreateIssueDeps{Locks: app.NewLocks(w.project, w.issues, w.auth), Issues: w.issues, Sequences: w.project,
		States: w.project, Labels: w.project, Members: w.project, Profiles: w.profiles, Tx: fakeTx{log: log}, Clock: clockAt{at: clockNow, log: log}})
````
````new server/internal/modules/issue/app/create_issue_test.go
	w.locks = app.NewLocks(w.project, w.issues, w.auth, fakeTx{log: log})
	uc := app.NewCreateIssue(app.CreateIssueDeps{Locks: w.locks, Issues: w.issues, Sequences: w.project, States: w.project, Labels: w.project,
		Members: w.project, Profiles: w.profiles, Clock: clockAt{at: clockNow, log: log}})
````
````old server/internal/modules/issue/app/create_issue_test.go
	activity := domain.CreationActivity([]domain.Assignee{{ID: bob, DisplayName: "bob"}, {ID: carol, DisplayName: "Carol C."}})
	if len(w.issues.activities) != len(activity) {
		t.Fatalf("the activity %+v, want %d rows", w.issues.activities, len(activity))
````
````new server/internal/modules/issue/app/create_issue_test.go
	wantActivity(t, w, id, domain.CreationActivity([]domain.Assignee{{ID: bob, DisplayName: "bob"}, {ID: carol, DisplayName: "Carol C."}}))
	if answer, _, _ := w.issues.GetIssue(context.Background(), id); !reflect.DeepEqual(got, answer) {
		t.Errorf("Execute() = %+v, want the work item as stored, %+v", got, answer)
	}
}

// wantActivity checks the activity w's store was given: a row of web's
// work item id for each of activities, in their order, alice's at the
// clock's moment, each id after the row's before.
func wantActivity(t *testing.T, w *world, id uuid.UUID, activities []domain.Activity) {
	t.Helper()
	if len(w.issues.activities) != len(activities) {
		t.Fatalf("the activity %+v, want %d rows", w.issues.activities, len(activities))
````
````old server/internal/modules/issue/app/create_issue_test.go
		wantRow := app.ActivityRow{ID: r.ID, WorkspaceID: acme, ProjectID: web, IssueID: id, Activity: activity[i], ActorID: alice, At: clockNow}
````
````new server/internal/modules/issue/app/create_issue_test.go
		wantRow := app.ActivityRow{ID: r.ID, WorkspaceID: acme, ProjectID: web, IssueID: id, Activity: activities[i], ActorID: alice, At: clockNow}
````
````old server/internal/modules/issue/app/create_issue_test.go
		}
	}
	if answer, _, _ := w.issues.GetIssue(context.Background(), id); !reflect.DeepEqual(got, answer) {
		t.Errorf("Execute() = %+v, want the work item as stored, %+v", got, answer)
````
````new server/internal/modules/issue/app/create_issue_test.go
		}
````
````old server/internal/modules/issue/app/create_issue_test.go
			t.Errorf("activity row %d: %+v, want %+v, its id after the row before", i, r, wantRow)
		}
	}
}
````
````new server/internal/modules/issue/app/create_issue_test.go
			t.Errorf("activity row %d: %+v, want %+v, its id after the row before", i, r, wantRow)
		}
	}
}

// wantRefusal checks a use case's refusal over w: err is want, of its code
// and with its fields, and the calls are calls. Each use case's test
// checks what its refusal leaves unwritten (refused, unchanged).
func wantRefusal(t *testing.T, w *world, err, want error, calls []string) {
	t.Helper()
	var se, wantSE *shared.Error
	if !errors.As(err, &se) || !errors.As(want, &wantSE) || se.Code != wantSE.Code || !reflect.DeepEqual(se.Fields, wantSE.Fields) {
		t.Errorf("Execute() = %v, want %v", err, want)
	}
	if !slices.Equal(w.log.calls, calls) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
	}
}

// wantStoppedAt checks a use case over w whose port failed: its answer is
// the failure, errDisk, as itself, and the port's first call is its last.
func wantStoppedAt(t *testing.T, w *world, err error, port string) {
	t.Helper()
	last := slices.IndexFunc(w.log.calls, func(c string) bool { return strings.HasPrefix(c, port+" ") })
	if !errors.Is(err, errDisk) || last != len(w.log.calls)-1 {
		t.Errorf("Execute() = %v after the calls\n%q\nwant %v, %s the last call", err, w.log.calls, errDisk, port)
	}
}
````
````old server/internal/modules/issue/app/create_issue_test.go
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
````
````new server/internal/modules/issue/app/create_issue_test.go
// refused checks a creation's refusal (wantRefusal), which writes nothing
// and takes no number.
func refused(t *testing.T, w *world, err, want error, calls []string) {
	t.Helper()
	wantRefusal(t, w, err, want, calls)
````
````old server/internal/modules/issue/app/create_issue_test.go
			last := slices.IndexFunc(w.log.calls, func(c string) bool { return strings.HasPrefix(c, port+" ") })
			if !errors.Is(err, errDisk) || last != len(w.log.calls)-1 {
				t.Errorf("Execute() = %v after the calls\n%q\nwant %v, %s the last call", err, w.log.calls, errDisk, port)
			}
````
````new server/internal/modules/issue/app/create_issue_test.go
			wantStoppedAt(t, w, err, port)
````

`server/internal/modules/issue/app/fakes_test.go`（修改，11 处）：

````old server/internal/modules/issue/app/fakes_test.go
// answers it, locked what Lock answers, nil for none.
````
````new server/internal/modules/issue/app/fakes_test.go
// answers it, locked what Lock answers, nil for none; waited, when set,
// runs as Lock waits, before it answers: what commits meanwhile.
````
````old server/internal/modules/issue/app/fakes_test.go
	locked    *app.LockedProject
````
````new server/internal/modules/issue/app/fakes_test.go
	locked    *app.LockedProject
	waited    func()
````
````old server/internal/modules/issue/app/fakes_test.go
func (f *fakeProject) Lock(ctx context.Context, id uuid.UUID) (app.LockedProject, bool, error) {
````
````new server/internal/modules/issue/app/fakes_test.go
func (f *fakeProject) Lock(ctx context.Context, id uuid.UUID) (app.LockedProject, bool, error) {
	if f.waited != nil {
		f.waited()
	}
````
````old server/internal/modules/issue/app/fakes_test.go
// item once its activity is written, so that it is not there to read back.
````
````new server/internal/modules/issue/app/fakes_test.go
// item once its activity is written, so that it is not there to read back.
// changed is the last update's fields, nil before any: the work item's
// last writer and moment.
````
````old server/internal/modules/issue/app/fakes_test.go
	issue      *app.IssueRow
````
````new server/internal/modules/issue/app/fakes_test.go
	issue      *app.IssueRow
	changed    *app.IssueFields
````
````old server/internal/modules/issue/app/fakes_test.go
}

// link logs an insert of name's rows r and answers their targets, or the
// failure: each row under the work item inserted, by its creator, then.
````
````new server/internal/modules/issue/app/fakes_test.go
}

// writer is the account and the moment of the work item's last write: its
// last update's, or its creation's.
func (f *fakeIssues) writer() (uuid.UUID, time.Time) {
	if f.changed != nil {
		return f.changed.By, f.changed.At
	}
	return f.issue.By, f.issue.At
}

func (f *fakeIssues) UpdateIssue(ctx context.Context, r app.IssueFields) error {
	if err := f.log.add(ctx, "UpdateIssue", r.ID); err != nil {
		return err
	}
	if f.issue == nil || r.ID != f.issue.ID {
		return fmt.Errorf("UpdateIssue: %+v is not the work item %+v", r, f.issue)
	}
	f.issue.Name, f.issue.StateID, f.issue.Priority, f.issue.CompletedAt = r.Name, r.StateID, r.Priority, r.CompletedAt
	f.changed = &r
	return nil
}

// link logs an insert of name's rows r and answers their targets, or the
// failure: each row under the work item inserted, by its last writer, then.
````
````old server/internal/modules/issue/app/fakes_test.go
		r.By != f.issue.By || !r.At.Equal(f.issue.At) || len(r.IDs) != len(r.Targets) || slices.Contains(r.IDs, uuid.UUID{}) {
		return nil, fmt.Errorf("%s: %+v not under the work item %+v", name, r, f.issue)
````
````new server/internal/modules/issue/app/fakes_test.go
		len(r.IDs) != len(r.Targets) || slices.Contains(r.IDs, uuid.UUID{}) {
		return nil, fmt.Errorf("%s: %+v not under the work item %+v", name, r, f.issue)
	}
	if by, at := f.writer(); r.By != by || !r.At.Equal(at) {
		return nil, fmt.Errorf("%s: %+v not by the work item's last writer %s at %s", name, r, by, at)
````
````old server/internal/modules/issue/app/fakes_test.go
func (f *fakeIssues) InsertAssignees(ctx context.Context, r app.LinkRows) (err error) {
	f.assignees, err = f.link(ctx, "InsertAssignees", r)
````
````new server/internal/modules/issue/app/fakes_test.go
// unlink logs a removal of name's rows r and answers the targets of from
// left, or the failure: each a row of the work item, by its last writer,
// then.
func (f *fakeIssues) unlink(ctx context.Context, name string, r app.UnlinkRows, from []uuid.UUID) ([]uuid.UUID, error) {
	if err := f.log.add(ctx, name, r.Targets); err != nil {
		return nil, err
	}
	if by, at := f.writer(); f.issue == nil || r.IssueID != f.issue.ID || r.By != by || !r.At.Equal(at) {
		return nil, fmt.Errorf("%s: %+v not of the work item %+v by its last writer", name, r, f.issue)
	}
	return slices.DeleteFunc(slices.Clone(from), func(id uuid.UUID) bool { return slices.Contains(r.Targets, id) }), nil
}

func (f *fakeIssues) InsertAssignees(ctx context.Context, r app.LinkRows) error {
	added, err := f.link(ctx, "InsertAssignees", r)
	f.assignees = append(f.assignees, added...)
````
````old server/internal/modules/issue/app/fakes_test.go
func (f *fakeIssues) InsertLabels(ctx context.Context, r app.LinkRows) (err error) {
	f.labels, err = f.link(ctx, "InsertLabels", r)
````
````new server/internal/modules/issue/app/fakes_test.go
func (f *fakeIssues) RemoveAssignees(ctx context.Context, r app.UnlinkRows) error {
	left, err := f.unlink(ctx, "RemoveAssignees", r, f.assignees)
	if err == nil {
		f.assignees = left
	}
	return err
}

func (f *fakeIssues) InsertLabels(ctx context.Context, r app.LinkRows) error {
	added, err := f.link(ctx, "InsertLabels", r)
	f.labels = append(f.labels, added...)
	return err
}

func (f *fakeIssues) RemoveLabels(ctx context.Context, r app.UnlinkRows) error {
	left, err := f.unlink(ctx, "RemoveLabels", r, f.labels)
	if err == nil {
		f.labels = left
	}
````
````old server/internal/modules/issue/app/fakes_test.go
	r := f.issue
````
````new server/internal/modules/issue/app/fakes_test.go
	r := f.issue
	by, at := f.writer()
````
````old server/internal/modules/issue/app/fakes_test.go
			uuid.UUID.Compare), CreatedByID: &r.By, UpdatedByID: &r.By, CreatedAt: r.At, UpdatedAt: r.At}, true, nil
````
````new server/internal/modules/issue/app/fakes_test.go
			uuid.UUID.Compare), CreatedByID: &r.By, UpdatedByID: &by, CreatedAt: r.At, UpdatedAt: at}, true, nil
````

`server/internal/modules/issue/app/get_issue_test.go`（修改，1 处）：

````old server/internal/modules/issue/app/get_issue_test.go
	return app.NewGetIssue(app.NewLocks(w.project, w.issues, w.auth)), w, id
````
````new server/internal/modules/issue/app/get_issue_test.go
	return app.NewGetIssue(w.locks), w, id
````

- [ ] **Step 4: 修改的用例**

`server/internal/modules/issue/app/update_issue.go`（新文件，233 行）：

````file server/internal/modules/issue/app/update_issue.go
package app

import (
	"context"
	"fmt"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateIssueDeps are what updateIssue reads and writes through.
type UpdateIssueDeps struct {
	Locks    Locks
	Issues   IssueUpdater
	States   StateCatalog
	Labels   LabelCatalog
	Members  ProjectMembers
	Profiles PublicProfiles
	Clock    Clock
}

// UpdateIssue changes a work item: PATCH /api/v0/issues/{issue_id} (M4
// design 3.5, 3.7, 5.1, 6.7).
type UpdateIssue struct {
	d UpdateIssueDeps
}

// NewUpdateIssue returns the use case.
func NewUpdateIssue(d UpdateIssueDeps) *UpdateIssue {
	return &UpdateIssue{d: d}
}

// named are the names an update's activity writes, read under the
// project's lock: the work item's state before and after, when the update
// changes it, and the new one's group; the labels it adds and removes,
// each in id order.
type named struct {
	state         *domain.Changed[domain.Named]
	group         domain.StateGroup
	labelsAdded   []domain.Named
	labelsRemoved []domain.Named
}

// Execute checks p (domain.CheckIssuePatch), before any read; then Locks
// runs the write on the work item id (M4 design 3.5, 6.7): its precheck,
// the work item read and the decision on issue.update in its project,
// before any lock; then in one transaction, the project FOR NO KEY UPDATE,
// the work item read again and the decision again, and under them, in
// this order: an archived project refuses it (409 project.archived); what
// p changes in the work item (domain.ChangesOf), and when it changes
// nothing, nothing is written and the answer is the work item as read;
// the dependent rows it adds (check), every problem in one 422; the clock;
// the work item's fields, its completion by its new state's group when its
// state changes (domain.CompletedAt); the accounts and the labels it
// removes and adds, in id order; the accounts it adds subscribed; its
// activity (domain.UpdateActivity), each account by his display name. All
// of it is the caller's at that moment. The answer is the work item as
// stored, read back in the transaction.
func (u *UpdateIssue) Execute(ctx context.Context, id uuid.UUID, p domain.IssuePatch) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	if p, err = domain.CheckIssuePatch(p); err != nil {
		return domain.Issue{}, err
	}
	w := write{issue: &id, action: domain.ActionUpdate, notFound: domain.ErrIssueNotFound}
	var updated domain.Issue
	err = u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		if h.project.Archived {
			return domain.ErrProjectArchived
		}
		c := domain.ChangesOf(h.issue, p)
		if c.None() {
			updated = h.issue
			return nil
		}
		n, err := u.check(ctx, h.issue, p, c)
		if err != nil {
			return err
		}
		s := stamp{workspace: h.issue.WorkspaceID, project: h.issue.ProjectID, issue: id, by: actor.UserID, at: u.d.Clock.Now()}
		if err := u.write(ctx, s, h.issue, c, n); err != nil {
			return err
		}
		updated, err = readBack(ctx, u.d.Issues, id)
		return err
	})
	if err != nil {
		return domain.Issue{}, err
	}
	return updated, nil
}

// check checks the dependent rows that c adds to the work item i under the
// project's lock (M4 design 3.6, 3.7), and reads the names its activity
// writes: the new state, when c changes it, is the project's and not its
// triage state, by a creation's rule (pickState); each account added may
// be assigned (domain.Unassignable), while one the work item keeps stays
// whatever his role; each label added is the project's. The labels the
// work item keeps are on it, so they are the project's undeleted labels:
// deleting a label takes it off every work item (DetachLabel). Every
// problem is reported at once, in one 422, each at its place in p.
func (u *UpdateIssue) check(ctx context.Context, i domain.Issue, p domain.IssuePatch, c domain.Changes) (named, error) {
	var n named
	var problems []shared.FieldError
	if c.StateID != nil {
		states, err := u.d.States.States(ctx, i.ProjectID)
		if err != nil {
			return named{}, err
		}
		old, found := stateOf(states, i.StateID)
		if !found {
			return named{}, fmt.Errorf("work item %s is in state %s, which is not its project's", i.ID, i.StateID)
		}
		if state, found := pickState(states, c.StateID); found {
			n.state = &domain.Changed[domain.Named]{Old: domain.Named{ID: old.ID, Name: old.Name}, New: domain.Named{ID: state.ID, Name: state.Name}}
			n.group = state.Group
		} else {
			problems = append(problems, domain.StateNotAllowed())
		}
	}
	if len(c.Assignees.Added) > 0 {
		roles, err := u.d.Members.Roles(ctx, i.ProjectID, c.Assignees.Added)
		if err != nil {
			return named{}, err
		}
		problems = append(problems, domain.Unassignable(*p.AssigneeIDs, c.Assignees.Added, roles)...)
	}
	var labels []LabelRef
	if len(c.Labels.Added) > 0 || len(c.Labels.Removed) > 0 {
		var err error
		asked := slices.SortedFunc(slices.Values(slices.Concat(c.Labels.Added, c.Labels.Removed)), uuid.UUID.Compare)
		if labels, err = u.d.Labels.Labels(ctx, i.ProjectID, asked); err != nil {
			return named{}, err
		}
		for at, id := range *p.LabelIDs {
			if slices.Contains(c.Labels.Added, id) && !slices.ContainsFunc(labels, func(l LabelRef) bool { return l.ID == id }) {
				problems = append(problems, domain.UnknownLabel(at))
			}
		}
	}
	if len(problems) > 0 {
		return named{}, shared.Invalid(problems...)
	}
	var err error
	if n.labelsAdded, err = labelNames(labels, c.Labels.Added); err != nil {
		return named{}, err
	}
	if n.labelsRemoved, err = labelNames(labels, c.Labels.Removed); err != nil {
		return named{}, err
	}
	return n, nil
}

// stateOf is the state id of states; found is false when none is.
func stateOf(states []StateRef, id uuid.UUID) (StateRef, bool) {
	i := slices.IndexFunc(states, func(s StateRef) bool { return s.ID == id })
	if i < 0 {
		return StateRef{}, false
	}
	return states[i], true
}

// labelNames are the labels ids, in their order, by their names in
// labels; none for none. A label not in labels is an error: the work
// item's labels, and those check let through, are the project's.
func labelNames(labels []LabelRef, ids []uuid.UUID) ([]domain.Named, error) {
	var out []domain.Named
	for _, id := range ids {
		i := slices.IndexFunc(labels, func(l LabelRef) bool { return l.ID == id })
		if i < 0 {
			return nil, fmt.Errorf("label %s is not the project's", id)
		}
		out = append(out, domain.Named{ID: id, Name: labels[i].Name})
	}
	return out, nil
}

// write writes what c changes in the work item i, all of it s's, in the
// order of M4 design 3.5: its fields; its assignees' rows, those removed
// then those added; its labels' rows, the same; its subscribers', the
// accounts added; its activity's.
func (u *UpdateIssue) write(ctx context.Context, s stamp, i domain.Issue, c domain.Changes, n named) error {
	f := IssueFields{ID: i.ID, Name: i.Name, StateID: i.StateID, Priority: i.Priority, CompletedAt: i.CompletedAt, By: s.by, At: s.at}
	activity := domain.Update{State: n.state, LabelsAdded: n.labelsAdded, LabelsRemoved: n.labelsRemoved}
	if c.Name != nil {
		f.Name, activity.Name = *c.Name, &domain.Changed[string]{Old: i.Name, New: *c.Name}
	}
	if c.Priority != nil {
		f.Priority, activity.Priority = *c.Priority, &domain.Changed[domain.Priority]{Old: i.Priority, New: *c.Priority}
	}
	if c.StateID != nil {
		f.StateID, f.CompletedAt = *c.StateID, domain.CompletedAt(n.group, s.at)
	}
	if err := u.d.Issues.UpdateIssue(ctx, f); err != nil {
		return err
	}
	if err := u.relink(ctx, s, c.Assignees, u.d.Issues.RemoveAssignees, u.d.Issues.InsertAssignees); err != nil {
		return err
	}
	if err := u.relink(ctx, s, c.Labels, u.d.Issues.RemoveLabels, u.d.Issues.InsertLabels); err != nil {
		return err
	}
	if len(c.Assignees.Added) > 0 {
		if err := u.d.Issues.InsertSubscribers(ctx, s.links(c.Assignees.Added)); err != nil {
			return err
		}
	}
	accounts, err := assignees(ctx, u.d.Profiles, slices.Concat(c.Assignees.Added, c.Assignees.Removed))
	if err != nil {
		return err
	}
	activity.AssigneesAdded, activity.AssigneesRemoved = accounts[:len(c.Assignees.Added)], accounts[len(c.Assignees.Added):]
	return u.d.Issues.InsertActivities(ctx, s.activity(domain.UpdateActivity(activity)))
}

// relink removes the rows of d's removed ids, then inserts those of its
// added ones, each when there are any.
func (u *UpdateIssue) relink(ctx context.Context, s stamp, d domain.Diff, remove func(context.Context, UnlinkRows) error,
	insert func(context.Context, LinkRows) error) error {
	if len(d.Removed) > 0 {
		if err := remove(ctx, s.unlinks(d.Removed)); err != nil {
			return err
		}
	}
	if len(d.Added) > 0 {
		return insert(ctx, s.links(d.Added))
	}
	return nil
}
````

`server/internal/modules/issue/app/update_issue_test.go`（新文件，322 行）：

````file server/internal/modules/issue/app/update_issue_test.go
package app_test

import (
	"errors"
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

// earlier is the moment web's work item 7 was made, an hour before the
// clock's; dave, drawn after every account of newCreate's world, is the
// member newUpdate adds to web; docs, drawn after its labels, is Docs, the
// label TestUpdateIssueWritesTheChangesAndTheirRows adds to web and to the
// work item.
var (
	earlier = clockNow.Add(-time.Hour)
	dave    = uuid.NewV7()
	docs    = uuid.NewV7()
)

// newUpdate is updateIssue over web's world (newCreate's), dave a member
// of web too, with web's work item 7 stored in it, made by carol at
// earlier: Fix, in Backlog, high, not completed, assigned to bob and to
// out, who is no member of web since, and labelled Bug.
func newUpdate() (*app.UpdateIssue, *world, uuid.UUID) {
	_, w := newCreate()
	id := uuid.NewV7()
	w.issues.issue = &app.IssueRow{ID: id, WorkspaceID: acme, ProjectID: web, SequenceID: 7, Name: "Fix", StateID: backlog.ID,
		Priority: domain.PriorityHigh, SortOrder: 65535, By: carol, At: earlier}
	w.issues.assignees, w.issues.labels = []uuid.UUID{bob, out}, []uuid.UUID{bug}
	w.profiles.names[out] = "out"
	w.project.roles[dave], w.profiles.names[dave] = shared.RoleMember, "dave"
	uc := app.NewUpdateIssue(app.UpdateIssueDeps{Locks: w.locks, Issues: w.issues, States: w.project, Labels: w.project, Members: w.project,
		Profiles: w.profiles, Clock: clockAt{at: clockNow, log: w.log}})
	return uc, w, id
}

// name is a patch's name.
func name(s string) *string {
	return &s
}

// set is a patch's set of ids.
func set(ids ...uuid.UUID) *[]uuid.UUID {
	return &ids
}

// locked are the calls of an update of id by alice up to the decision under
// the lock: the work item read and the decision, before any lock and
// outside the transaction; in it, web's lock, the work item read again and
// the decision again.
func locked(id uuid.UUID) []string {
	return []string{line("GetIssue", id, "outside tx"), line("Authorize", alice, domain.ActionUpdate, acme, web, "outside tx"), "Begin",
		line("Lock", web), line("GetIssue", id), line("Authorize", alice, domain.ActionUpdate, acme, web)}
}

// An update, in the order of M4 design 3.5 and 6.7: Locks' calls
// (locked); then the new state, the roles of the accounts added and the
// labels added and removed, checked or named; the clock; the work item's
// fields; the assignees removed and added, the labels the same, each in id
// order; the accounts added subscribed; the display names of the accounts
// added and removed; the activity: the name, the priority, the state by
// its names, each label added and removed, each account added and removed
// (domain.UpdateActivity), all of it alice's at the moment; the answer is
// the work item read back. The name is stored without its surrounding
// space; in Done the work item is completed at the moment. out, who is no
// member of web, stays assigned: only the accounts added, carol and dave,
// are checked, and each of them, not only the first, is assigned,
// subscribed and in the activity. Docs, which the work item has and
// keeps, is neither read nor written again (M4 design 3.7): Bug's row
// alone is removed and UI's alone added.
func TestUpdateIssueWritesTheChangesAndTheirRows(t *testing.T) {
	uc, w, id := newUpdate()
	w.project.labels = append(w.project.labels, app.LabelRef{ID: docs, Name: "Docs"})
	w.issues.labels = append(w.issues.labels, docs)
	low := domain.PriorityLow
	got, err := uc.Execute(as(alice), id, domain.IssuePatch{Name: name(" Mend "), StateID: &done.ID, Priority: &low,
		AssigneeIDs: set(out, dave, carol), LabelIDs: set(docs, ui)})
	want := domain.Issue{ID: id, WorkspaceID: acme, ProjectID: web, SequenceID: 7, Name: "Mend", StateID: done.ID, Priority: low,
		SortOrder: 65535, CompletedAt: &clockNow, AssigneeIDs: []uuid.UUID{carol, out, dave}, LabelIDs: []uuid.UUID{ui, docs}, CreatedByID: &carol,
		UpdatedByID: &alice, CreatedAt: earlier, UpdatedAt: clockNow}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("Execute() = %+v, %v; want %+v", got, err, want)
	}
	calls := append(locked(id), line("States", web), line("Roles", web, []uuid.UUID{carol, dave}), line("Labels", web, []uuid.UUID{bug, ui}), "Now",
		line("UpdateIssue", id), line("RemoveAssignees", []uuid.UUID{bob}), line("InsertAssignees", []uuid.UUID{carol, dave}),
		line("RemoveLabels", []uuid.UUID{bug}), line("InsertLabels", []uuid.UUID{ui}), line("InsertSubscribers", []uuid.UUID{carol, dave}),
		line("DisplayNames", []uuid.UUID{carol, dave, bob}), line("InsertActivities", 8), line("GetIssue", id))
	if !slices.Equal(w.log.calls, calls) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
	}
	if fields := (app.IssueFields{ID: id, Name: "Mend", StateID: done.ID, Priority: low, CompletedAt: &clockNow, By: alice,
		At: clockNow}); w.issues.changed == nil || !reflect.DeepEqual(*w.issues.changed, fields) {
		t.Errorf("the fields written %+v, want %+v", w.issues.changed, fields)
	}
	wantActivity(t, w, id, domain.UpdateActivity(domain.Update{
		Name: &domain.Changed[string]{Old: "Fix", New: "Mend"}, Priority: &domain.Changed[domain.Priority]{Old: domain.PriorityHigh, New: low},
		State:       &domain.Changed[domain.Named]{Old: domain.Named{ID: backlog.ID, Name: "Backlog"}, New: domain.Named{ID: done.ID, Name: "Done"}},
		LabelsAdded: []domain.Named{{ID: ui, Name: "UI"}}, LabelsRemoved: []domain.Named{{ID: bug, Name: "Bug"}},
		AssigneesAdded:   []domain.Assignee{{ID: carol, DisplayName: "Carol C."}, {ID: dave, DisplayName: "dave"}},
		AssigneesRemoved: []domain.Assignee{{ID: bob, DisplayName: "bob"}},
	}))
}

// A patch that changes nothing, given empty or with each field as it is,
// its sets in another order: the precheck, the lock and the decisions run
// as for any update; nothing is written, not even the clock read, and the
// answer is the work item as read under the lock, its last writer and
// moment as they were.
func TestUpdateIssueThatChangesNothingWritesNothing(t *testing.T) {
	high := domain.PriorityHigh
	for what, p := range map[string]domain.IssuePatch{
		"nothing given": {},
		"each field as it is": {Name: name(" Fix"), StateID: &backlog.ID, Priority: &high, AssigneeIDs: set(out, bob),
			LabelIDs: set(bug)},
	} {
		t.Run(what, func(t *testing.T) {
			uc, w, id := newUpdate()
			got, err := uc.Execute(as(alice), id, p)
			want := domain.Issue{ID: id, WorkspaceID: acme, ProjectID: web, SequenceID: 7, Name: "Fix", StateID: backlog.ID, Priority: high,
				SortOrder: 65535, AssigneeIDs: []uuid.UUID{bob, out}, LabelIDs: []uuid.UUID{bug}, CreatedByID: &carol, UpdatedByID: &carol,
				CreatedAt: earlier, UpdatedAt: earlier}
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Errorf("Execute() = %+v, %v; want %+v", got, err, want)
			}
			if calls := locked(id); !slices.Equal(w.log.calls, calls) || w.issues.changed != nil {
				t.Errorf("calls:\n%q\nwant\n%q; the fields written %+v, want none", w.log.calls, calls, w.issues.changed)
			}
		})
	}
}

// A state of the completed group completes the work item at the moment,
// also from another state of that group; any other state leaves it not
// completed, a cancelled one too; a change that leaves its state leaves its
// completion as it is (M4 design 3.7, 6.7).
func TestUpdateIssueCompletesTheWorkItemByItsState(t *testing.T) {
	shipped := app.StateRef{ID: uuid.NewV7(), Name: "Shipped", Group: domain.GroupCompleted}
	dropped := app.StateRef{ID: uuid.NewV7(), Name: "Dropped", Group: "cancelled"}
	for _, tt := range []struct {
		name      string
		from      app.StateRef
		p         domain.IssuePatch
		completed *time.Time
	}{
		{"into Done", backlog, domain.IssuePatch{StateID: &done.ID}, &clockNow},
		{"out of Done", done, domain.IssuePatch{StateID: &backlog.ID}, nil},
		{"from Done into Shipped", done, domain.IssuePatch{StateID: &shipped.ID}, &clockNow},
		{"into Dropped, cancelled", backlog, domain.IssuePatch{StateID: &dropped.ID}, nil},
		{"from Done into Dropped", done, domain.IssuePatch{StateID: &dropped.ID}, nil},
		{"in Done, renamed", done, domain.IssuePatch{Name: name("Mend")}, &earlier},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id := newUpdate()
			w.project.states = append(w.project.states, shipped, dropped)
			w.issues.issue.StateID = tt.from.ID
			if tt.from.Group == domain.GroupCompleted {
				w.issues.issue.CompletedAt = &earlier
			}
			got, err := uc.Execute(as(alice), id, tt.p)
			if err != nil || !reflect.DeepEqual(got.CompletedAt, tt.completed) {
				t.Errorf("Execute() = %+v, %v; want completed at %v", got, err, tt.completed)
			}
		})
	}
}

// unchanged checks an update's refusal (wantRefusal), which writes
// nothing: the work item, its assignees and its labels are as they were.
func unchanged(t *testing.T, w *world, err, want error, calls []string) {
	t.Helper()
	wantRefusal(t, w, err, want, calls)
	if w.issues.changed != nil || !slices.Equal(w.issues.assignees, []uuid.UUID{bob, out}) || !slices.Equal(w.issues.labels, []uuid.UUID{bug}) {
		t.Errorf("after the refusal: the fields written %+v, assignees %v, labels %v; want none, bob and out, Bug", w.issues.changed,
			w.issues.assignees, w.issues.labels)
	}
}

// Every refusal the decision makes is answered before any lock and outside
// any transaction (M4 design 3.5): a work item not there, or whose project
// the caller does not see, is issue.not_found, a role the rule does not
// allow is forbidden, and nothing is locked. Values that break the rules
// are refused before even the work item is read.
func TestUpdateIssueRefusesBeforeAnyLock(t *testing.T) {
	for _, tt := range []struct {
		name  string
		set   func(w *world, id *uuid.UUID)
		p     domain.IssuePatch
		want  error
		calls int // of locked's
	}{
		{"a blank name", func(*world, *uuid.UUID) {}, domain.IssuePatch{Name: name(" ")}, shared.Invalid(shared.FieldError{Field: "name",
			Code: "too_short", Message: "must not be empty"}), 0},
		{"no work item", func(_ *world, id *uuid.UUID) { *id = uuid.NewV7() }, domain.IssuePatch{}, domain.ErrIssueNotFound, 1},
		{"a project not seen", func(w *world, _ *uuid.UUID) { w.auth.answers = []error{shared.ErrNotVisible} }, domain.IssuePatch{},
			domain.ErrIssueNotFound, 2},
		{"a role not allowed", func(w *world, _ *uuid.UUID) { w.auth.answers = []error{shared.Forbidden()} }, domain.IssuePatch{},
			shared.Forbidden(), 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id := newUpdate()
			tt.set(w, &id)
			_, err := uc.Execute(as(alice), id, tt.p)
			unchanged(t, w, err, tt.want, locked(id)[:tt.calls])
		})
	}
}

// Under the lock the decision is made again, and is final: a project
// deleted while the lock waited, a work item deleted meanwhile or no longer
// in it, or a project no longer seen, is issue.not_found; a role no longer
// allowed forbidden; an archived project refuses the update, one that
// changes nothing too (409 project.archived). Then the dependent rows the
// update adds: a state not the project's, or its triage state; an account
// added who is its guest; a label added that is not its: every problem in
// one 422, each at its place in the patch (the label second in label_ids,
// though first among the labels added). out, kept, is not checked, though
// no member of web.
func TestUpdateIssueRefusesUnderTheLock(t *testing.T) {
	other := uuid.NewV7() // after every id of the world
	notAllowed := func(field, message string) shared.FieldError {
		return shared.FieldError{Field: field, Code: "not_allowed", Message: message}
	}
	stateNotAllowed := notAllowed("state_id", "must be a state of the project that is not its triage state")
	for _, tt := range []struct {
		name  string
		set   func(w *world)
		p     domain.IssuePatch
		want  error
		calls func(id uuid.UUID) []string
	}{
		{"the project deleted while the lock waited", func(w *world) { w.project.locked = nil }, domain.IssuePatch{}, domain.ErrIssueNotFound,
			func(id uuid.UUID) []string { return locked(id)[:4] }},
		{"the work item deleted meanwhile", func(w *world) { w.project.waited = func() { w.issues.issue.ID = uuid.NewV7() } },
			domain.IssuePatch{}, domain.ErrIssueNotFound, func(id uuid.UUID) []string { return locked(id)[:5] }},
		{"the work item in another project", func(w *world) { w.project.waited = func() { w.issues.issue.ProjectID = other } },
			domain.IssuePatch{}, domain.ErrIssueNotFound, func(id uuid.UUID) []string { return locked(id)[:5] }},
		{"no longer seen", func(w *world) { w.auth.answers = []error{nil, shared.ErrNotVisible} }, domain.IssuePatch{}, domain.ErrIssueNotFound,
			locked},
		{"no longer allowed", func(w *world) { w.auth.answers = []error{nil, shared.Forbidden()} }, domain.IssuePatch{}, shared.Forbidden(), locked},
		{"archived", func(w *world) { w.project.locked.Archived = true }, domain.IssuePatch{}, domain.ErrProjectArchived, locked},
		{"the triage state, a guest added, a label not the project's", func(*world) {},
			domain.IssuePatch{StateID: &triage.ID, AssigneeIDs: set(gus, bob, out, carol), LabelIDs: set(bug, other)},
			shared.Invalid(stateNotAllowed, notAllowed("assignee_ids[0]", "must be an active member of the project who is not its guest"),
				notAllowed("label_ids[1]", "must be a label of the project")),
			func(id uuid.UUID) []string {
				return append(locked(id), line("States", web), line("Roles", web, []uuid.UUID{carol, gus}), line("Labels", web, []uuid.UUID{other}))
			}},
		{"another project's state", func(*world) {}, domain.IssuePatch{StateID: &other}, shared.Invalid(stateNotAllowed),
			func(id uuid.UUID) []string { return append(locked(id), line("States", web)) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id := newUpdate()
			tt.set(w)
			_, err := uc.Execute(as(alice), id, tt.p)
			unchanged(t, w, err, tt.want, tt.calls(id))
		})
	}
}

// A port that fails ends the update with its failure, as itself, and no
// call after it runs, the read and the decision under the lock too once
// the precheck's have passed: the transaction rolls back whatever was
// written. What cannot be is an error too, before any write: a work item
// whose state is not its project's; a label it is removed from that is not
// its project's; an account without a display name, before the activity;
// and a work item not there to read back after its write.
func TestUpdateIssueStopsAtAFailingPort(t *testing.T) {
	low := domain.PriorityLow
	p := domain.IssuePatch{Name: name("Mend"), StateID: &done.ID, Priority: &low, AssigneeIDs: set(out, carol), LabelIDs: set(ui)}
	for _, port := range []string{"GetIssue", "Authorize", "Lock", "States", "Roles", "Labels", "UpdateIssue", "RemoveAssignees",
		"InsertAssignees", "RemoveLabels", "InsertLabels", "InsertSubscribers", "DisplayNames", "InsertActivities"} {
		t.Run(port, func(t *testing.T) {
			uc, w, id := newUpdate()
			w.log.fail = port
			_, err := uc.Execute(as(alice), id, p)
			wantStoppedAt(t, w, err, port)
		})
	}
	for _, tt := range []struct {
		name, port string
		calls      int // of locked's
	}{
		{"the work item read again under the lock", "GetIssue", 5},
		{"the decision under the lock", "Authorize", 6},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id := newUpdate()
			w.project.waited = func() { w.log.fail = tt.port }
			_, err := uc.Execute(as(alice), id, p)
			if calls := locked(id)[:tt.calls]; !errors.Is(err, errDisk) || !slices.Equal(w.log.calls, calls) {
				t.Errorf("Execute() = %v after the calls\n%q\nwant %v after\n%q", err, w.log.calls, errDisk, calls)
			}
		})
	}
	for _, tt := range []struct {
		name   string
		set    func(w *world)
		before string // a call that does not run
	}{
		{"a state not the project's", func(w *world) { w.project.states = []app.StateRef{done, triage} }, "Now"},
		{"a label not the project's", func(w *world) { w.project.labels = []app.LabelRef{{ID: ui, Name: "UI"}} }, "Now"},
		{"no display name", func(w *world) { delete(w.profiles.names, carol) }, "InsertActivities"},
		{"lost after its write", func(w *world) { w.issues.lost = true }, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id := newUpdate()
			tt.set(w)
			if got, err := uc.Execute(as(alice), id, p); err == nil ||
				tt.before != "" && slices.ContainsFunc(w.log.calls, func(c string) bool { return strings.HasPrefix(c, tt.before) }) {
				t.Errorf("Execute() = %+v, %v after %q; want an error before %s", got, err, w.log.calls, tt.before)
			}
		})
	}
}
````

`server/internal/modules/issue/module.go`（修改，2 处）：

````old server/internal/modules/issue/module.go
	locks := app.NewLocks(d.Projects, store, d.Authorizer)
````
````new server/internal/modules/issue/module.go
	locks := app.NewLocks(d.Projects, store, d.Authorizer, d.Tx)
````
````old server/internal/modules/issue/module.go
			Profiles: d.Profiles, Tx: d.Tx, Clock: d.Clock,
````
````new server/internal/modules/issue/module.go
			Profiles: d.Profiles, Clock: d.Clock,
````

- [ ] **Step 5: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestEveryModuleDeclaresItsActionsOrHasNone$|TestEveryActionHasARuleAndEveryRuleAnAction$|TestPermissionMatrix|TestEachIssueWriteLocksItsProjectFirst$' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make e2e`
Expected: 121 个通过（I1 经改过的创建）。

- [ ] **Step 6: 提交**

```bash
git add server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/issue/app/create_issue.go server/internal/modules/issue/app/create_issue_test.go server/internal/modules/issue/app/fakes_test.go server/internal/modules/issue/app/get_issue_test.go server/internal/modules/issue/app/lock.go server/internal/modules/issue/app/ports.go server/internal/modules/issue/app/update_issue.go server/internal/modules/issue/app/update_issue_test.go server/internal/modules/issue/app/writes.go server/internal/modules/issue/domain/actions.go server/internal/modules/issue/domain/issue.go server/internal/modules/issue/domain/patch.go server/internal/modules/issue/domain/patch_test.go server/internal/modules/issue/module.go
```
```bash
git commit -m "feat(M4/P1): updateIssue's use case on the one lock path, which now reads a work item

Locks runs every write on the work-item side: its precheck before any
lock, then in one transaction its project's lock and its decision, now
with the work item it names read again under the lock and found still in
that project. createIssue runs on it too. updateIssue checks the patch,
refuses an archived project, writes nothing when nothing changes, checks
the state, the accounts newly assigned and the labels added under the
lock, and writes the work item, the assignees and labels it removes and
adds, the new assignees' subscriptions and its activity at one moment.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p1b.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；静态是 `make lint-web` 的文案和键的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T5.1` | 已归档的项目里的修改照样通过 | `internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `updateIssue` 各格（`make test`）（Task 6 起）、`internal/bootstrap` 的 `TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`（`make test`）（Task 7 起） | 单元；组合 |
| `T5.2` | 什么都不改的修改照样写工作项 | `internal/modules/issue/app` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 单元；端到端 |
| `T5.3` | `updateIssue` 在锁之前读时刻 | `internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestTwoChangesOfAWorkItemSerialize`（`make test`）（Task 11 起） | 单元；组合 |
| `T5.4` | 改状态不改完成时间 | `internal/modules/issue/app` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 单元；端到端 |
| `T5.5` | 工作项可以移到项目的分诊状态 | `internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `updateIssue` 各格（`make test`）（Task 6 起） | 单元；组合 |
| `T5.6` | 不是工作项的项目的状态也接受 | `internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `updateIssue` 各格（`make test`）（Task 6 起）、`internal/bootstrap` 的 `TestDeletingAStateAndPuttingAWorkItemInItSerialize`（`make test`）（Task 11 起） | 单元；组合 |
| `T5.7` | 工作项留下的负责人也再查一遍，已离开项目的被拒绝 | `internal/modules/issue/domain` 的 全部测试（`make test`）、`internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `updateIssue` 各格（`make test`）（Task 6 起） | 单元；组合 |
| `T5.8` | 不能指派的账户照样指派 | `internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `updateIssue` 各格（`make test`）（Task 6 起） | 单元；组合 |
| `T5.9` | 不是工作项的项目的标签在写之前不拒绝（422） | `internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `updateIssue` 各格（`make test`）（Task 6 起）、`internal/bootstrap` 的 `TestDeletingALabelAndLabellingAWorkItemWithItSerialize`（`make test`）（Task 11 起）、故事 `i2-update-issue`（Task 14 起） | 单元；组合；端到端 |
| `T5.10` | 修改去掉的负责人也加关注 | `internal/modules/issue/app` 的 全部测试（`make test`） | 单元 |
| `T5.11` | 先写标签的行、再写负责人的（违反全局顺序） | `internal/modules/issue/app` 的 全部测试（`make test`） | 单元 |
| `T5.12` | 动态不写修改去掉的负责人 | `internal/modules/issue/app` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 单元；端到端 |
| `T5.13` | 工作项的写失败之后 `updateIssue` 照样往下走 | `internal/modules/issue/app` 的 全部测试（`make test`） | 单元 |
| `T5.14` | 读成员的角色失败之后 `updateIssue` 照样往下走 | `internal/modules/issue/app` 的 全部测试（`make test`） | 单元 |
| `T5.15` | 指名工作项的写在锁下不再判定 | `internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused`（`make test`）（Task 11 起） | 单元；组合 |
| `T5.16` | 等锁期间被删除的工作项照样被写 | `internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemGoneWhileItsChangeWaitsIsNotFound`（`make test`）（Task 11 起） | 单元；组合 |
| `T5.17` | 锁下重读的工作项不核对仍在锁住的项目里 | `internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemGoneWhileItsChangeWaitsIsNotFound`（`make test`）（Task 11 起） | 单元；组合 |
| `T5.18` | 锁下发现工作项的项目已不在时答 `project.not_found` | `internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemsWriteAndDeletingItsProjectOrWorkspaceSerialize`（`make test`）（Task 11 起） | 单元；组合 |
| `T5.19` | 修改只读加的标签的名称，不读去掉的 | `internal/modules/issue/app` 的 全部测试（`make test`）、故事 `i2-update-issue`（Task 14 起） | 单元；端到端 |
| `T5.20` | 锁住项目之前就重读工作项：等过锁的修改按它等锁之前的样子算差异 | `internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestTwoChangesOfAWorkItemSerialize`（`make test`）（Task 11 起） | 单元；组合 |
| `T5.21` | 修改把移进已取消组的状态的工作项当作完成 | `internal/modules/issue/app` 的 全部测试（`make test`） | 单元 |
| `T5.22` | 修改者因他的修改关注工作项 | `internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestUpdatingAWorkItemStampsItsRows`（`make test`）（Task 7 起） | 单元；组合 |

---

### Task 6: `updateIssue` 的接口、接线、矩阵和取锁形状的一行

**Files:**
- Create: `server/internal/bootstrap/permission_matrix_issue_updates_test.go`、`server/internal/modules/issue/adapter/http/update_test.go`
- Modify: `api/modules/issue.yaml`、`server/internal/bootstrap/issue_world_test.go`、`server/internal/bootstrap/issue_write_locks_test.go`、`server/internal/bootstrap/issue_writes_test.go`、`server/internal/bootstrap/permission_matrix_issues_test.go`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/modules/issue/adapter/http/handler.go`、`server/internal/modules/issue/adapter/http/handler_test.go`、`server/internal/modules/issue/adapter/http/issues.go`、`server/internal/modules/issue/module.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/issue/adapter/http/gen/bodyshape.gen.go`、`server/internal/modules/issue/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**（spec 2.8、第 3 节第 2、3 条；M4 设计 5.1–5.3、6.6、9.2、9.3）
- 契约 `api/modules/issue.yaml`：
  - `PATCH /api/v0/issues/{issue_id}`（`updateIssue`，`x-problem-codes: [validation_failed, issue.not_found, forbidden, project.archived]`），描述写明：给出的字段改、没给的不改；值按 `createIssue` 的规则、在读工作项之前检查；锁之前答的 404、403，锁下再判定；已归档 409；状态、新负责人、加的标签的检查一次报出；什么都不改时不写；完成时间的转换；新负责人关注；动态；一个时刻；
  - `IssueUpdate`（`additionalProperties: false`：名称、`state_id`、`priority`、`assignee_ids`、`label_ids`）。
- `issue/adapter/http`：`handler.go` 的 `UpdateIssueUseCase`、`UseCases.UpdateIssue`；`issues.go` 的 `UpdateIssue`（每个字段照原样交给用例）。`issue/module.go` 接上 `UpdateIssue`，包的说明写到取、改工作项。
- 矩阵：`permission_matrix_issue_updates_test.go`（新）的 `updateIssue` 九行和它们的核对（`answeredIssue`、`seededIssue`、`answered`；`readsTheIssue` 移到这里）；`permission_matrix_issues_test.go`、`permission_matrix_test.go` 接上。
- `issue_world_test.go`：Done、`newWorkItem`（alice 在 Web 建的 `fullIssue`）。
- `issue_write_locks_test.go`：`issueWrites` 加 `updateIssue` 一行（`path` 是契约的路径，`target(t, w)` 是发的路径、`body(w, n)`；`reads` 是 `states`：它在锁下最先读、在锁之前不读的表；`issues` 它在锁之前就读过）；`target` 不是它的 `path` 的一个路径（`pathOf`）的行在发之前失败，不在一个操作的名下测另一个操作。
- `issue_writes_test.go`：P1a 的 `TestCreatingAWorkItemStampsItsRows` 用 `newWorkItem` 建它的工作项。

**Tests:**
- HTTP（`apitest.Main` 两个方向）：`TestUpdateIssue`（只给的字段交给用例，回答 200 和工作项）、`TestUpdateIssueHoldsTheBodyToItsStructure`（编号、排序值、父工作项、描述、日期等 P1b 之外的字段，`null`、类型不对、不是 id 的 id：`bad_request`，不到用例）、`TestUpdateIssueRefusals`。
- 组合：`TestPermissionMatrix` 的 `updateIssue` 九行（只改名称、每个字段都给、什么都不改、已归档、另一个项目的状态、另一个项目的标签、分诊状态、指派访客、留下已离开的负责人）；`TestThePermissionMatrixCoversEveryOperation`；`TestEachIssueWriteLocksItsProjectFirst` 的 `updateIssue` 一行（每行发的是它的操作的路径；拒绝的不取锁；项目最先；停在 `states` 上时持 Web 的 `FOR NO KEY UPDATE`、工作区上没有锁，`writtenTables` 没有它的写；工作区行被持 N 时照样完成）；`TestCreatingAWorkItemStampsItsRows` 照旧（它的工作项由 `newWorkItem` 建）。

- [ ] **Step 1: 契约，生成**

`api/modules/issue.yaml`（修改，2 处）：

````old api/modules/issue.yaml
          description: The work item.
````
````new api/modules/issue.yaml
          description: The work item.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IssueDetail'
        default:
          $ref: '#/components/responses/Problem'
    patch:
      operationId: updateIssue
      tags: [issue]
      summary: Change a work item
      description: >-
        For the admins and members of its project. The fields given change
        and the others stay; a list given is the work item's new list, empty
        for none. The values follow createIssue's rules (validation_failed),
        checked before the work item is looked at. A work item that does not
        exist, is deleted, or whose project the caller does not see answers
        issue.not_found; one whose project he sees but whose work items he
        may not change, forbidden; both are answered before the project is
        locked, and with the project locked the role is decided again. A
        work item of an archived project cannot be changed
        (project.archived). Then the state, when it changes, is one of the
        project's, not its triage state; each account the work item is newly
        assigned to is an active member of the project who is not its guest,
        while one it keeps stays assigned whatever has become of him; each
        label added is one of the project's (validation_failed, every
        problem at once). A change that changes nothing writes nothing, and
        the work item is answered as it is. Otherwise the work item is
        changed last by the caller; moved into a state of the completed
        group, it is completed at the change, and moved into another, not
        completed; the accounts newly assigned subscribe to it; its
        activity says what changed: the name, the priority, the state, and
        each label and each assignee added and removed. Every row it writes
        is the caller's, at one moment.
      security: [{bearer: []}]
      x-problem-codes: [validation_failed, issue.not_found, forbidden, project.archived]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/IssueUpdate'
      responses:
        '200':
          description: The work item as changed.
````
````old api/modules/issue.yaml
          description: At most 100 of the project's labels, each named once.
          type: array
          items:
            type: string
            format: uuid

````
````new api/modules/issue.yaml
          description: At most 100 of the project's labels, each named once.
          type: array
          items:
            type: string
            format: uuid
    IssueUpdate:
      description: Changes the fields it names; a field left out keeps its value.
      type: object
      additionalProperties: false
      properties:
        name:
          description: 1–255 characters without the white space around it, which is not stored.
          type: string
        state_id:
          description: One of the project's states, not its triage state.
          type: string
          format: uuid
        priority:
          $ref: '#/components/schemas/IssuePriority'
        assignee_ids:
          description: >-
            The work item's new assignees, empty for none: at most 100
            accounts, each named once; each it is not assigned to yet an
            active member of the project who is not its guest.
          type: array
          items:
            type: string
            format: uuid
        label_ids:
          description: >-
            The work item's new labels, empty for none: at most 100 of the
            project's labels, each named once.
          type: array
          items:
            type: string
            format: uuid

````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `bb2390d24ea2d838bbf8260f1dfa04c448311a9d8630c772a7586ea4bf702e41` | 3196 | `api/dist/openapi.yaml` |
| `d70083285a6f319b9bd30ac513e42d76af057c872d8830d3e0af74a571d19072` | 34 | `server/internal/modules/issue/adapter/http/gen/bodyshape.gen.go` |
| `bc130a65e3fb7b5543a3fe25554ca35b48b81b9fe2338187bf84ee16bf6dec6e` | 662 | `server/internal/modules/issue/adapter/http/gen/server.gen.go` |
| `9b24370294427f335aa3d6553a8ec8cdcd9ed6975c48be08c979cda8033f62de` | 3492 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/issue/adapter/http/gen/bodyshape.gen.go server/internal/modules/issue/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: HTTP 和接线**

`server/internal/modules/issue/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/issue/adapter/http/handler.go
}

// UseCases are the use cases behind the module's operations.
````
````new server/internal/modules/issue/adapter/http/handler.go
}

// UpdateIssueUseCase is app.UpdateIssue.
type UpdateIssueUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, p domain.IssuePatch) (domain.Issue, error)
}

// UseCases are the use cases behind the module's operations.
````
````old server/internal/modules/issue/adapter/http/handler.go
	GetIssue    GetIssueUseCase
````
````new server/internal/modules/issue/adapter/http/handler.go
	GetIssue    GetIssueUseCase
	UpdateIssue UpdateIssueUseCase
````

`server/internal/modules/issue/adapter/http/handler_test.go`（修改，1 处）：

````old server/internal/modules/issue/adapter/http/handler_test.go
}

// newServer serves the module's API over the use cases uc, as bootstrap
````
````new server/internal/modules/issue/adapter/http/handler_test.go
}

// fakeUpdate is updateIssue: each call is recorded as "caller work item",
// with what it got; it answers answer, or err.
type fakeUpdate struct {
	calls  []string
	got    []domain.IssuePatch
	answer domain.Issue
	err    error
}

func (f *fakeUpdate) Execute(ctx context.Context, id uuid.UUID, p domain.IssuePatch) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String())
	f.got = append(f.got, p)
	return f.answer, f.err
}

// newServer serves the module's API over the use cases uc, as bootstrap
````

`server/internal/modules/issue/adapter/http/issues.go`（修改，1 处）：

````old server/internal/modules/issue/adapter/http/issues.go
}

// issueDetail is i as the API shows it: each field that may be null null
````
````new server/internal/modules/issue/adapter/http/issues.go
}

// UpdateIssue serves PATCH /api/v0/issues/{issue_id}: each field nil when
// not given; a value given goes to the use case as it came, which checks
// it.
func (h handler) UpdateIssue(ctx context.Context, req gen.UpdateIssueRequestObject) (gen.UpdateIssueResponseObject, error) {
	p := domain.IssuePatch{Name: req.Body.Name, StateID: req.Body.StateID, Priority: (*domain.Priority)(req.Body.Priority),
		AssigneeIDs: req.Body.AssigneeIds, LabelIDs: req.Body.LabelIds}
	i, err := h.uc.UpdateIssue.Execute(ctx, req.IssueID, p)
	if err != nil {
		return nil, err
	}
	return gen.UpdateIssue200JSONResponse(issueDetail(i)), nil
}

// issueDetail is i as the API shows it: each field that may be null null
````

`server/internal/modules/issue/adapter/http/update_test.go`（新文件，92 行）：

````file server/internal/modules/issue/adapter/http/update_test.go
package httpadapter_test

import (
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// PATCH goes to the use case for the caller and the path's work item, with
// the body's fields, each nil when not given, a list given empty given; the
// answer is 200 with the work item the use case answers, as POST shows it.
func TestUpdateIssue(t *testing.T) {
	path := "/api/v0/issues/" + webSeven.ID.String()
	name, high, none := " Mend ", domain.PriorityHigh, []uuid.UUID{}
	for _, tt := range []struct {
		body string
		want domain.IssuePatch
	}{
		{`{"name":" Mend ","state_id":"0199a2b4-0000-7000-8000-0000000000c1","priority":"high",` +
			`"assignee_ids":["0199a2b4-0000-7000-8000-000000000002"],"label_ids":["0199a2b4-0000-7000-8000-0000000000d1"]}`,
			domain.IssuePatch{Name: &name, StateID: &stateID, Priority: &high, AssigneeIDs: &[]uuid.UUID{bobID}, LabelIDs: &[]uuid.UUID{bugID}}},
		{`{}`, domain.IssuePatch{}},
		{`{"assignee_ids":[],"label_ids":[]}`, domain.IssuePatch{AssigneeIDs: &none, LabelIDs: &none}},
	} {
		update := &fakeUpdate{answer: webSeven}
		res, body := do(t, newServer(t, httpadapter.UseCases{UpdateIssue: update}), request(http.MethodPatch, path, "alice", tt.body))
		if res.StatusCode != http.StatusOK || body != webSevenJSON+"\n" {
			t.Errorf("PATCH %s = %d %s, want 200 %s", tt.body, res.StatusCode, body, webSevenJSON)
		}
		if want := []string{aliceID.String() + " " + webSeven.ID.String()}; !slices.Equal(update.calls, want) ||
			!reflect.DeepEqual(update.got, []domain.IssuePatch{tt.want}) {
			t.Errorf("PATCH %s: calls %q with %+v; want %q with %+v", tt.body, update.calls, update.got, want, tt.want)
		}
	}
}

// A body with a field it may not have (a number, which the project gives;
// a sort order, a parent, a description, a date, which IssueUpdate has not
// in P1b: M4/P1b spec 3), null for a field that is not nullable, a field of
// another type, or an id that is no id: refused as bad_request before the
// use case.
func TestUpdateIssueHoldsTheBodyToItsStructure(t *testing.T) {
	update := &fakeUpdate{}
	h := newServer(t, httpadapter.UseCases{UpdateIssue: update})
	for _, body := range []string{`{"sequence_id":3}`, `{"sort_order":1}`, `{"parent_id":null}`, `{"description_html":"<p>x</p>"}`,
		`{"start_date":"2026-10-18"}`, `{"name":null}`, `{"state_id":null}`, `{"assignee_ids":null}`, `{"name":1}`, `{"state_id":"Backlog"}`,
		`{"assignee_ids":"bob"}`, `{"label_ids":["Bug"]}`} {
		res, got := do(t, h, request(http.MethodPatch, "/api/v0/issues/"+webSeven.ID.String(), "alice", body))
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
func TestUpdateIssueRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"a label refused", shared.Invalid(domain.UnknownLabel(0)), http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"label_ids[0]","code":"not_allowed","message":"must be a label of the project"}]}`},
		{"no work item", domain.ErrIssueNotFound, http.StatusNotFound,
			`{"status":404,"code":"issue.not_found","title":"Not Found","detail":"The work item does not exist, or you cannot see it."}`},
		{"a project guest", shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"an archived project", domain.ErrProjectArchived, http.StatusConflict,
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
		{"a failure", errGone, http.StatusInternalServerError,
			`{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
	} {
		h := newServer(t, httpadapter.UseCases{UpdateIssue: &fakeUpdate{err: tt.err}})
		res, body := do(t, h, request(http.MethodPatch, "/api/v0/issues/"+webSeven.ID.String(), "alice", `{"name":"Mend"}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: PATCH = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
````

`server/internal/modules/issue/module.go`（修改，2 处）：

````old server/internal/modules/issue/module.go
// work items' tables and brings creating a work item. For the project
// module's deletions (its IssueCascade port) it soft-deletes a deleted
// project's rows of them, tells whether work items are in a state, and
// takes deleted labels off them.
````
````new server/internal/modules/issue/module.go
// work items' tables and brings creating, reading and changing a work
// item. For the project module's deletions (its IssueCascade port) it
// soft-deletes a deleted project's rows of them, tells whether work items
// are in a state, and takes deleted labels off them.
````
````old server/internal/modules/issue/module.go
		GetIssue: app.NewGetIssue(locks),
````
````new server/internal/modules/issue/module.go
		GetIssue: app.NewGetIssue(locks),
		UpdateIssue: app.NewUpdateIssue(app.UpdateIssueDeps{
			Locks: locks, Issues: store, States: d.Projects, Labels: d.Projects, Members: d.Projects, Profiles: d.Profiles, Clock: d.Clock,
		}),
````

- [ ] **Step 3: 矩阵和取锁形状的一行**

`server/internal/bootstrap/issue_world_test.go`（修改，4 处）：

````old server/internal/bootstrap/issue_world_test.go
// its public project Web, with Web's state Todo and its label Bug; bob,
// a member of acme and of Web; and three accounts who may not write on
// Web's work items: walt, a member of acme who is none of Web, and wanda,
// an admin of acme who is none of Web either, see Web (403); olga, no
// member of acme, does not (404).
````
````new server/internal/bootstrap/issue_world_test.go
// its public project Web, with Web's states Todo and Done and its label
// Bug, and no work item yet; bob, a member of acme and of Web; and three
// accounts who may not write on Web's work items: walt, a member of acme
// who is none of Web, and wanda, an admin of acme who is none of Web
// either, see Web (403); olga, no member of acme, does not (404).
````
````old server/internal/bootstrap/issue_world_test.go
	contract             *apitest.Contract
	base                 string
	pool                 *pgxpool.Pool
	tokens               map[string]string    // each account's, by its name
	ids                  map[string]uuid.UUID // each account's, by its name
	acme, web, todo, bug uuid.UUID
````
````new server/internal/bootstrap/issue_world_test.go
	contract                   *apitest.Contract
	base                       string
	pool                       *pgxpool.Pool
	tokens                     map[string]string    // each account's, by its name
	ids                        map[string]uuid.UUID // each account's, by its name
	acme, web, todo, done, bug uuid.UUID
````
````old server/internal/bootstrap/issue_world_test.go
	w.todo = stateID(t, w.pool, w.web, "Todo")
````
````new server/internal/bootstrap/issue_world_test.go
	w.todo, w.done = stateID(t, w.pool, w.web, "Todo"), stateID(t, w.pool, w.web, "Done")
````
````old server/internal/bootstrap/issue_world_test.go
}

// fullIssue is the body of a work item in Todo, assigned to bob and
````
````new server/internal/bootstrap/issue_world_test.go
}

// newWorkItem is the id of a work item alice creates in Web through the
// API: fullIssue's.
func (w issueWorld) newWorkItem(t *testing.T) uuid.UUID {
	t.Helper()
	var created struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", w.fullIssue(), http.StatusCreated),
		&created)
	return created.ID
}

// fullIssue is the body of a work item in Todo, assigned to bob and
````

`server/internal/bootstrap/issue_write_locks_test.go`（修改，8 处）：

````old server/internal/bootstrap/issue_write_locks_test.go
	"context"
	"fmt"
````
````new server/internal/bootstrap/issue_write_locks_test.go
	"context"
````
````old server/internal/bootstrap/issue_write_locks_test.go
	"time"
````
````new server/internal/bootstrap/issue_write_locks_test.go
	"time"
	"uuid"
````
````old server/internal/bootstrap/issue_write_locks_test.go
// method and its path as the contract has them, with %s for Web's id; its
// body, which names the world's rows; the answer it gets; and reads, the
// first table it reads under its project's lock that its precheck did not
// read, before any write of its own.
````
````new server/internal/bootstrap/issue_write_locks_test.go
// method and its path as the contract has them; target, the path it
// sends, a path of that pattern (pathOf), to Web of the world or to a row
// made in Web for it; body, its body when alice has sent it n times
// before, which names the world's rows and changes them anew; the answer
// it gets; and reads, the first table it reads under its project's lock
// that its precheck did not read, before any write of its own.
````
````old server/internal/bootstrap/issue_write_locks_test.go
	body             func(w issueWorld) string
````
````new server/internal/bootstrap/issue_write_locks_test.go
	target           func(t *testing.T, w issueWorld) string
	body             func(w issueWorld, n int) string
````
````old server/internal/bootstrap/issue_write_locks_test.go
	{op: "createIssue", method: http.MethodPost, path: "/api/v0/projects/%s/issues", body: issueWorld.fullIssue, want: http.StatusCreated,
		reads: "states"},
````
````new server/internal/bootstrap/issue_write_locks_test.go
	{op: "createIssue", method: http.MethodPost, path: "/api/v0/projects/{project_id}/issues",
		target: func(_ *testing.T, w issueWorld) string { return "/api/v0/projects/" + w.web.String() + "/issues" },
		body:   func(w issueWorld, _ int) string { return w.fullIssue() }, want: http.StatusCreated, reads: "states"},
	// A work item in Todo, which each of alice's sends moves to the other
	// state, Done first: a change of state, so the states are read under
	// the lock.
	{op: "updateIssue", method: http.MethodPatch, path: "/api/v0/issues/{issue_id}",
		target: func(t *testing.T, w issueWorld) string { return "/api/v0/issues/" + w.newWorkItem(t).String() },
		body: func(w issueWorld, n int) string {
			return `{"state_id":"` + []uuid.UUID{w.done, w.todo}[n%2].String() + `"}`
		},
		want: http.StatusOK, reads: "states"},
````
````old server/internal/bootstrap/issue_write_locks_test.go
// interleaving 21). Every write of the contract has its row here
// (issueWrites), which a write without one fails before any database.
````
````new server/internal/bootstrap/issue_write_locks_test.go
// interleaving 21). Every write of the contract has its row here
// (issueWrites), which a write without one fails before any database; a
// row whose target is not a path of its operation (pathOf) fails before
// it sends, so that no row tests another operation under its name.
````
````old server/internal/bootstrap/issue_write_locks_test.go
			return o.ID == iw.op && o.Method == iw.method && o.Path == fmt.Sprintf(iw.path, "{project_id}")
````
````new server/internal/bootstrap/issue_write_locks_test.go
			return o.ID == iw.op && o.Method == iw.method && o.Path == iw.path
````
````old server/internal/bootstrap/issue_write_locks_test.go
			path, body := w.base+fmt.Sprintf(iw.path, w.web), iw.body(w)
			send := func(name string) <-chan answer {
````
````new server/internal/bootstrap/issue_write_locks_test.go
			target := iw.target(t, w)
			if !pathOf(iw.path, target) {
				t.Fatalf("%s's row sends %s %s, not a path of %s", iw.op, iw.method, target, iw.path)
			}
			path, alices := w.base+target, 0
			send := func(name string) <-chan answer {
				body := iw.body(w, alices)
				if name == "alice" {
					alices++
				}
````

`server/internal/bootstrap/issue_writes_test.go`（修改，3 处）：

````old server/internal/bootstrap/issue_writes_test.go
	answer := w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", w.fullIssue(), http.StatusCreated)
	after := time.Now()
	var created struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, answer, &created)
````
````new server/internal/bootstrap/issue_writes_test.go
	item := w.newWorkItem(t)
	after := time.Now()
````
````old server/internal/bootstrap/issue_writes_test.go
			FROM issue_activities WHERE issue_id = $1`, created.ID, w.ids["alice"])
````
````new server/internal/bootstrap/issue_writes_test.go
			FROM issue_activities WHERE issue_id = $1`, item, w.ids["alice"])
````
````old server/internal/bootstrap/issue_writes_test.go
		WHERE a.issue_id = $1 AND a.field = 'assignees' AND a.new_identifier = $2 AND u.id = $2`, created.ID, w.ids["bob"]).Scan(&named,
````
````new server/internal/bootstrap/issue_writes_test.go
		WHERE a.issue_id = $1 AND a.field = 'assignees' AND a.new_identifier = $2 AND u.id = $2`, item, w.ids["bob"]).Scan(&named,
````

`server/internal/bootstrap/permission_matrix_issue_updates_test.go`（新文件，163 行）：

````file server/internal/bootstrap/permission_matrix_issue_updates_test.go
package bootstrap

import (
	"net/http"
	"reflect"
	"slices"
	"testing"
	"uuid"
)

// The rows of the permission matrix of changing a work item (M4 design
// 9.2), naming it by its id (/issues/{issue_id}): its column's project's
// seeded work item, matrixIssue. Each cell writes on a copy of its own.

// toIssue is the request of a row whose callers each change their column's
// project's seeded work item with the body body gives for their column.
func toIssue(body func(c caller, s seeded) string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return issueRows.to(http.MethodPatch, "", matrixIssue, body(c, s))(c, s)
	}
}

func issueUpdateMatrixRows() []matrixRow {
	ok := issueRows.of(cellOK, cellOK, cellForbidden, cellOK, cellForbidden, cellForbidden)
	refused := issueRows.of(cellValidationFailed, cellValidationFailed, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden)
	return []matrixRow{
		// The project's admins and members, as createIssue (M4 design 3.4).
		{op: "updateIssue", write: true, columns: projectColumns, request: issueRows.to(http.MethodPatch, "", matrixIssue, `{"name":"Mend"}`),
			cells: ok, check: renamesTheIssue},
		// Each field given: Done, high, PA's account for PM's and the removed
		// member's, UI for Bug.
		{op: "updateIssue", variant: "each field given", write: true, columns: projectColumns,
			request: toIssue(func(c caller, s seeded) string {
				return `{"name":"Mend","state_id":"` + s.state(projectOf(c), "Done").String() + `","priority":"high","assignee_ids":["` +
					s.account(callerProjectAdmin).String() + `"],"label_ids":["` + s.label(projectOf(c), "UI").String() + `"]}`
			}),
			cells: ok, check: changesEachField},
		// A change that changes nothing answers the work item as it is.
		{op: "updateIssue", variant: "nothing changed", write: true, columns: projectColumns,
			request: issueRows.to(http.MethodPatch, "", matrixIssue, `{}`), cells: ok, check: readsTheIssue},
		// An archived project's work items are not changed (M4 design 3.8),
		// after the decision: who may not change one learns nothing of its
		// project's state.
		{op: "updateIssue", variant: "archived", write: true, columns: archivedColumns,
			request: issueRows.to(http.MethodPatch, "", matrixIssue, `{"name":"Mend"}`), cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		// The archived project's Backlog, a state of another project of the
		// same workspace: the state is checked after the decision.
		{op: "updateIssue", variant: "another project's state", write: true, columns: projectColumns,
			request: toIssue(func(_ caller, s seeded) string {
				return `{"state_id":"` + s.state("acme/archived", "Backlog").String() + `"}`
			}),
			cells: refused, refusal: "state_id not_allowed"},
		// The archived project's UI, a label of another project of the same
		// workspace, checked after the decision too.
		{op: "updateIssue", variant: "another project's label", write: true, columns: projectColumns,
			request: toIssue(func(_ caller, s seeded) string {
				return `{"label_ids":["` + s.label("acme/archived", "UI").String() + `"]}`
			}),
			cells: refused, refusal: "label_ids[0] not_allowed"},
		// The project's triage state takes no work item (M4 design 3.7).
		{op: "updateIssue", variant: "the triage state", write: true, columns: projectColumns,
			request: toIssue(func(c caller, s seeded) string {
				return `{"state_id":"` + s.state(projectOf(c), "Triage").String() + `"}`
			}),
			cells: refused, refusal: "state_id not_allowed"},
		// PG's account, the project's guest, added to the work item's
		// assignees, who are kept: only he is checked, and refused at his
		// place (M4 design 3.6, 3.7).
		{op: "updateIssue", variant: "a guest assigned", write: true, columns: projectColumns,
			request: toIssue(func(_ caller, s seeded) string {
				return `{"assignee_ids":["` + s.account(callerProjectMember).String() + `","` + s.account(callerRemoved).String() + `","` +
					s.account(accountOf(callerProjectGuest)).String() + `"]}`
			}),
			cells: refused, refusal: "assignee_ids[2] not_allowed"},
		// The removed member, whose membership of acme and of its public
		// project ended, stays assigned while PM is unassigned: an assignee
		// who has left is kept (M4 design 3.6).
		{op: "updateIssue", variant: "an assignee who left kept", write: true, columns: projectColumns,
			request: toIssue(func(_ caller, s seeded) string {
				return `{"assignee_ids":["` + s.account(callerRemoved).String() + `"]}`
			}),
			cells: ok, check: keepsTheAssigneeWhoLeft},
	}
}

// answeredIssue is a work item as an answer shows it: the fields the
// checks read.
type answeredIssue struct {
	ID          uuid.UUID   `json:"id"`
	ProjectID   uuid.UUID   `json:"project_id"`
	SequenceID  int         `json:"sequence_id"`
	Name        string      `json:"name"`
	StateID     uuid.UUID   `json:"state_id"`
	Priority    string      `json:"priority"`
	CompletedAt *string     `json:"completed_at"`
	AssigneeIDs []uuid.UUID `json:"assignee_ids"`
	LabelIDs    []uuid.UUID `json:"label_ids"`
	CreatedByID *uuid.UUID  `json:"created_by_id"`
	UpdatedByID *uuid.UUID  `json:"updated_by_id"`
}

// seededIssue is the column's project's seeded work item as issueSeed
// stored it, as an answer shows it: its first, in Todo, medium, not
// completed, assigned to PM's account and the removed member's, in id
// order, labelled Bug, made and changed last by acme's admin.
func seededIssue(c caller, s seeded) answeredIssue {
	key, admin := projectOf(c), s.account(callerAdmin)
	return answeredIssue{ID: s.issue(key, matrixIssue), ProjectID: s.project(key), SequenceID: 1, Name: matrixIssue,
		StateID: s.state(key, "Todo"), Priority: "medium",
		AssigneeIDs: slices.SortedFunc(slices.Values([]uuid.UUID{s.account(callerProjectMember), s.account(callerRemoved)}), uuid.UUID.Compare),
		LabelIDs:    []uuid.UUID{s.label(key, "Bug")}, CreatedByID: &admin, UpdatedByID: &admin}
}

// answered checks that answer, the work item c's cell changes, is the
// seeded one with change made to it, the caller its last writer, and
// completed when completed; completedAt is not compared, but present or
// not.
func answered(t *testing.T, c caller, s seeded, answer string, completed bool, change func(i *answeredIssue)) {
	t.Helper()
	var got answeredIssue
	decodeAnswer(t, answer, &got)
	want := seededIssue(c, s)
	me := s.account(accountOf(c))
	want.UpdatedByID = &me
	change(&want)
	if (got.CompletedAt != nil) != completed {
		t.Errorf("%s changes %s; want it completed %v", c, answer, completed)
	}
	got.CompletedAt = nil
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s changes %s; want %+v", c, answer, want)
	}
}

// readsTheIssue: the column's project's seeded work item as issueSeed
// stored it, in each field answeredIssue holds (seededIssue).
func readsTheIssue(t *testing.T, c caller, s seeded, answer string) {
	var got answeredIssue
	decodeAnswer(t, answer, &got)
	if want := seededIssue(c, s); !reflect.DeepEqual(got, want) {
		t.Errorf("%s reads %s; want %+v", c, answer, want)
	}
}

// renamesTheIssue: the seeded work item named Mend, the rest as it was.
func renamesTheIssue(t *testing.T, c caller, s seeded, answer string) {
	answered(t, c, s, answer, false, func(i *answeredIssue) { i.Name = "Mend" })
}

// changesEachField: the seeded work item named Mend, in Done and so
// completed, high, assigned to PA's account alone, labelled UI alone.
func changesEachField(t *testing.T, c caller, s seeded, answer string) {
	answered(t, c, s, answer, true, func(i *answeredIssue) {
		i.Name, i.StateID, i.Priority = "Mend", s.state(projectOf(c), "Done"), "high"
		i.AssigneeIDs, i.LabelIDs = []uuid.UUID{s.account(callerProjectAdmin)}, []uuid.UUID{s.label(projectOf(c), "UI")}
	})
}

// keepsTheAssigneeWhoLeft: the seeded work item assigned to the removed
// member alone.
func keepsTheAssigneeWhoLeft(t *testing.T, c caller, s seeded, answer string) {
	answered(t, c, s, answer, false, func(i *answeredIssue) { i.AssigneeIDs = []uuid.UUID{s.account(callerRemoved)} })
}
````

`server/internal/bootstrap/permission_matrix_issues_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_issues_test.go

// readsTheIssue: the column's project's seeded work item, its id, project,
// number, name, state, priority, assignees, labels and creator as
// issueSeed stored them: its first, named matrixIssue, in Todo, medium,
// assigned to PM's account and the removed member's, in id order, labelled
// Bug, made by acme's admin.
func readsTheIssue(t *testing.T, c caller, s seeded, answer string) {
	var i struct {
		ID          uuid.UUID   `json:"id"`
		ProjectID   uuid.UUID   `json:"project_id"`
		SequenceID  int         `json:"sequence_id"`
		Name        string      `json:"name"`
		StateID     uuid.UUID   `json:"state_id"`
		Priority    string      `json:"priority"`
		AssigneeIDs []uuid.UUID `json:"assignee_ids"`
		LabelIDs    []uuid.UUID `json:"label_ids"`
		CreatedByID *uuid.UUID  `json:"created_by_id"`
	}
	decodeAnswer(t, answer, &i)
	key := projectOf(c)
	assignees := slices.SortedFunc(slices.Values([]uuid.UUID{s.account(callerProjectMember), s.account(callerRemoved)}), uuid.UUID.Compare)
	if i.ID != s.issue(key, matrixIssue) || i.ProjectID != s.project(key) || i.SequenceID != 1 || i.Name != matrixIssue ||
		i.StateID != s.state(key, "Todo") || i.Priority != "medium" || !slices.Equal(i.AssigneeIDs, assignees) ||
		!slices.Equal(i.LabelIDs, []uuid.UUID{s.label(key, "Bug")}) || i.CreatedByID == nil || *i.CreatedByID != s.account(callerAdmin) {
		t.Errorf("%s reads %s; want %s's %s, its first, in Todo, medium, assigned to %v, labelled Bug, by acme's admin", c, answer, key,
			matrixIssue, assignees)
	}
}

````
````new server/internal/bootstrap/permission_matrix_issues_test.go

````

`server/internal/bootstrap/permission_matrix_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_test.go
		labelMatrixRows(), issueMatrixRows())
````
````new server/internal/bootstrap/permission_matrix_test.go
		labelMatrixRows(), issueMatrixRows(), issueUpdateMatrixRows())
````

- [ ] **Step 4: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix|TestThePermissionMatrixCoversEveryOperation$|TestEachIssueWriteLocksItsProjectFirst$|TestCreatingAWorkItemStampsItsRows$' ./internal/bootstrap/`
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

- [ ] **Step 5: 提交**

```bash
git add api/modules/issue.yaml server/internal/bootstrap/issue_world_test.go server/internal/bootstrap/issue_write_locks_test.go server/internal/bootstrap/issue_writes_test.go server/internal/bootstrap/permission_matrix_issue_updates_test.go server/internal/bootstrap/permission_matrix_issues_test.go server/internal/bootstrap/permission_matrix_test.go server/internal/modules/issue/adapter/http/handler.go server/internal/modules/issue/adapter/http/handler_test.go server/internal/modules/issue/adapter/http/issues.go server/internal/modules/issue/adapter/http/update_test.go server/internal/modules/issue/module.go api/dist/openapi.yaml server/internal/modules/issue/adapter/http/gen/bodyshape.gen.go server/internal/modules/issue/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M4/P1): updateIssue in the API, wired, in the matrix and the lock shape

PATCH /api/v0/issues/{issue_id} takes the fields P1b changes, nothing
else, and answers the work item as changed. The matrix holds a change of
each column, with every field given, with nothing changed, in an
archived project, with another project's state or label, the triage
state, a guest assigned and an assignee who left kept. The lock shape's
row stops the change on the states it first reads under the lock: it
holds its project's row FOR NO KEY UPDATE and nothing of the workspace.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p1b.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；静态是 `make lint-web` 的文案和键的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T6.1` | PATCH 丢掉请求体的 `label_ids` | `internal/modules/issue/adapter/http` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `updateIssue` 各格（`make test`）、故事 `i2-update-issue`（Task 14 起） | 单元；组合；端到端 |
| `T6.2` | 契约让 `IssueUpdate` 带后面的 Phase 的字段（`sort_order`、`parent_id`、日期） | `internal/modules/issue/adapter/http` 的 全部测试（`make test`） | 单元 |
| `T6.3` | 写的预检以 `FOR SHARE` 读工作项：判定之前取了行锁 | `internal/bootstrap` 的 `TestEachIssueWriteLocksItsProjectFirst`（`make test`） | 组合 |

本 Task 的检查也发现较早的 Task 的变异 `T5.1`、`T5.5`、`T5.6`、`T5.7`、`T5.8`、`T5.9`（列在各自的 Task 的表里，标"（Task 6 起）"）：本 Task 做完之后，在本 Task 的树上重跑它们（`mutants_p1b.py`，`MUT_LATER=6`）。

---

### Task 7: 盖戳、回答如存和事务的连接

**Files:**
- Create: `server/internal/bootstrap/issue_updates_test.go`
- Modify: `server/internal/bootstrap/issue_world_test.go`、`server/internal/bootstrap/issue_writes_test.go`

**Interfaces:** 没有新的生产代码（spec 2.9；M4 设计 3.5、3.13、5.2；P1a 评审第 6 节）。`issue_world_test.go` 的 `workItemStamps`（一个工作项在工作项一侧五张表里的行，每行一句：谁、在哪个时刻建它和最后改它，删除的时刻，谁的；两个盖戳测试共用这一个查询）；`issue_updates_test.go` 的 `displayNames`、`storedIssue`；`issue_writes_test.go` 的创建的盖戳测试改用 `workItemStamps`，事务的连接加修改。

**Tests:**（组合）
- `TestUpdatingAWorkItemStampsItsRows`：alice 把 walt 加进 Web、建工作项，bob 改（名称、优先级、到 Done、负责人换成 walt、去掉 Bug）：工作项和 bob 写的每一行（去的负责人和标签、加的负责人和 walt 的关注、六行动态）由 bob 在请求之内的同一时刻写，bob 改了它也不关注它（3.12）；bob 删掉的两行保留 alice 的建，创建的其余的行（两行动态、alice 的关注）照旧；动态的值来自 `users.display_name`（经 `issueProfiles`）和目录的名称；回答的创建者、最后修改者和两个时刻。
- `TestAWorkItemReadsAsStored`：Web 的第二个工作项，在第一个之下、已归档（在数据库里写：P1b 没有归档的操作），由 bob 在 alice 建之后先改到 Done、再改优先级和标签：`getIssue` 回答每个字段如存（每个可空的字段有自己的值，每个戳是它自己的；建、完成、最后修改、归档是依次的四个时刻）。
- `TestCreatingAWorkItemStampsItsRows`（P1a）经 `workItemStamps` 读它的七行：都由 alice 在请求之内的同一时刻写，负责人的行是 bob 的，关注者是 alice 和 bob，两行动态的执行者是 alice。
- `TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`：一个连接的池上，修改（状态、负责人、标签）、什么都不改、锁下的拒绝，和创建一样，每个读写都在事务的连接上；读回在事务里。

- [ ] **Step 1: 盖戳和回答如存**

`server/internal/bootstrap/issue_world_test.go`（修改，2 处）：

````old server/internal/bootstrap/issue_world_test.go
import (
	"net/http"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
````
````new server/internal/bootstrap/issue_world_test.go
import (
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
````
````old server/internal/bootstrap/issue_world_test.go
		w.bug.String() + `"]}`
}
````
````new server/internal/bootstrap/issue_world_test.go
		w.bug.String() + `"]}`
}

// workItemStamps are the rows of the work item id in the work-item side's
// five tables, each a line, sorted: its table, who made it and changed it
// last, and when, when it was deleted if it was, and whose it is, for an
// assignee, a subscriber or an activity (the account assigned, subscribed
// or acting). An account goes by its name in the world, a moment by the
// work item's: made, its creation, or changed, its last change.
func (w issueWorld) workItemStamps(t *testing.T, id uuid.UUID) []string {
	t.Helper()
	var made, changed time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT created_at, updated_at FROM issues WHERE id = $1", id).Scan(&made, &changed); err != nil {
		t.Fatal(err)
	}
	when := func(at time.Time) string {
		switch {
		case at.Equal(made):
			return "made"
		case at.Equal(changed):
			return "changed"
		}
		return at.String()
	}
	names := make(map[uuid.UUID]string, len(w.ids))
	for name, account := range w.ids {
		names[account] = name
	}
	who := func(account *uuid.UUID) string {
		if account == nil {
			return "no one"
		}
		if name, ok := names[*account]; ok {
			return name
		}
		return account.String()
	}
	rows, err := w.pool.Query(pgtest.Soon(t), `
		SELECT 'issues', created_by_id, created_at, updated_by_id, updated_at, deleted_at, NULL::uuid FROM issues WHERE id = $1
		UNION ALL SELECT 'issue_assignees', created_by_id, created_at, updated_by_id, updated_at, deleted_at, assignee_id FROM issue_assignees
			WHERE issue_id = $1
		UNION ALL SELECT 'issue_labels', created_by_id, created_at, updated_by_id, updated_at, deleted_at, NULL FROM issue_labels
			WHERE issue_id = $1
		UNION ALL SELECT 'issue_subscribers', created_by_id, created_at, updated_by_id, updated_at, deleted_at, subscriber_id
			FROM issue_subscribers WHERE issue_id = $1
		UNION ALL SELECT 'issue_activities', created_by_id, created_at, updated_by_id, updated_at, deleted_at, actor_id FROM issue_activities
			WHERE issue_id = $1`, id)
	if err != nil {
		t.Fatal(err)
	}
	lines, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (string, error) {
		var table string
		var madeBy, changedBy, of *uuid.UUID
		var madeAt, changedAt time.Time
		var deletedAt *time.Time
		if err := row.Scan(&table, &madeBy, &madeAt, &changedBy, &changedAt, &deletedAt, &of); err != nil {
			return "", err
		}
		line := table + " made by " + who(madeBy) + " " + when(madeAt) + ", changed by " + who(changedBy) + " " + when(changedAt)
		if deletedAt != nil {
			line += ", deleted " + when(*deletedAt)
		}
		if of != nil {
			line += ", of " + who(of)
		}
		return line, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return slices.Sorted(slices.Values(lines))
}
````

`server/internal/bootstrap/issue_updates_test.go`（新文件，181 行）：

````file server/internal/bootstrap/issue_updates_test.go
package bootstrap

import (
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// updateIssue as bootstrap wires it stamps every row it writes at one
// moment within its request, from the clock issue.New takes, and as its
// caller (M4 design 3.7, M2 handoff 3): alice creates a work item in Web
// of the world, in Todo, assigned to herself and labelled Bug; bob renames
// it, makes it urgent, moves it to Done, assigns it to walt, a member of
// Web, alone and takes Bug off. The work item is changed last by bob then,
// and completed then, its creation alice's as it was; alice's assignment
// and Bug's row are deleted by bob then; walt's assignment and his
// subscription are bob's, then, and bob, who changed it, does not
// subscribe to it (3.12); its activity gains six rows of his, then: the
// name, the priority, the state by its names, Bug removed, walt added and
// alice removed, each account by his display name as identity has it.
// The rows bob deleted keep their making by alice, and the creation's
// other rows, its two activities and alice's subscription, are as they
// were. The answer shows the work item as stored: made by alice at its
// creation, changed last by bob then.
func TestUpdatingAWorkItemStampsItsRows(t *testing.T) {
	w := newIssueWorld(t)
	alice, bob, walt := w.ids["alice"], w.ids["bob"], w.ids["walt"]
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/members",
		`{"members":[{"member_id":"`+walt.String()+`","role":15}]}`, http.StatusCreated)
	var created struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", `{"name":"Fix","state_id":"`+
		w.todo.String()+`","assignee_ids":["`+alice.String()+`"],"label_ids":["`+w.bug.String()+`"]}`, http.StatusCreated), &created)
	before := time.Now().Truncate(time.Microsecond)
	answer := w.send(t, "bob", http.MethodPatch, "/api/v0/issues/"+created.ID.String(), `{"name":"Mend","priority":"urgent","state_id":"`+
		w.done.String()+`","assignee_ids":["`+walt.String()+`"],"label_ids":[]}`, http.StatusOK)
	after := time.Now()

	var made, changed time.Time
	var completed *time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT created_at, updated_at, completed_at FROM issues WHERE id = $1", created.ID).Scan(&made,
		&changed, &completed); err != nil {
		t.Fatal(err)
	}
	if changed.Equal(made) || changed.Before(before) || changed.After(after) || completed == nil || !completed.Equal(changed) {
		t.Fatalf("the work item made at %v, changed at %v, completed at %v; want it changed and completed at one moment within %v to %v",
			made, changed, completed, before, after)
	}
	creation := "issue_activities made by alice made, changed by alice made, of alice"
	update := "issue_activities made by bob changed, changed by bob changed, of bob"
	want := []string{creation, creation, update, update, update, update, update, update,
		"issue_assignees made by alice made, changed by bob changed, deleted changed, of alice",
		"issue_assignees made by bob changed, changed by bob changed, of walt",
		"issue_labels made by alice made, changed by bob changed, deleted changed",
		"issue_subscribers made by alice made, changed by alice made, of alice",
		"issue_subscribers made by bob changed, changed by bob changed, of walt",
		"issues made by alice made, changed by bob changed"}
	if got := w.workItemStamps(t, created.ID); !slices.Equal(got, want) {
		t.Errorf("the work item's rows:\n%q\nwant\n%q", got, want)
	}

	activity, err := w.pool.Query(pgtest.Soon(t), `SELECT a.field || ': ' || a.old_value || ' → ' || a.new_value FROM issue_activities a
		WHERE a.issue_id = $1 AND a.actor_id = $2 ORDER BY a.id`, created.ID, bob)
	if err != nil {
		t.Fatal(err)
	}
	said, err := pgx.CollectRows(activity, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	names := displayNames(t, w, alice, walt)
	if want := []string{"name: Fix → Mend", "priority: none → urgent", "state: Todo → Done", "labels: Bug → ", "assignees:  → " + names[1],
		"assignees: " + names[0] + " → "}; !slices.Equal(said, want) {
		t.Errorf("bob's activity says %q, want %q", said, want)
	}

	var shown struct {
		CreatedByID uuid.UUID `json:"created_by_id"`
		UpdatedByID uuid.UUID `json:"updated_by_id"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
	}
	if decodeAnswer(t, answer, &shown); shown.CreatedByID != alice || shown.UpdatedByID != bob || !shown.CreatedAt.Equal(made) ||
		!shown.UpdatedAt.Equal(changed) {
		t.Errorf("updateIssue answers %s; want made by alice at %v, changed last by bob at %v", answer, made, changed)
	}
}

// displayNames are the accounts' display names as identity has them, in
// their order.
func displayNames(t *testing.T, w issueWorld, ids ...uuid.UUID) []string {
	t.Helper()
	names := make([]string, len(ids))
	for i, id := range ids {
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT display_name FROM users WHERE id = $1", id).Scan(&names[i]); err != nil {
			t.Fatal(err)
		}
	}
	return names
}

// getIssue as bootstrap wires it answers the work item as stored, every
// field (M4 design 5.2): Web's second work item, under the first and
// archived (in the database: no operation of P1b archives a work item),
// moved into Done by bob after alice made it and changed by him again
// after that, so that each field that may be null has a value of its own,
// each of its stamps is its own, and it was made, completed, changed last
// and archived at four moments, in that order.
func TestAWorkItemReadsAsStored(t *testing.T) {
	w := newIssueWorld(t)
	first, second := w.newWorkItem(t), w.newWorkItem(t)
	path := "/api/v0/issues/" + second.String()
	w.send(t, "bob", http.MethodPatch, path, `{"state_id":"`+w.done.String()+`"}`, http.StatusOK)
	w.send(t, "bob", http.MethodPatch, path, `{"priority":"low","label_ids":[]}`, http.StatusOK)
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issues SET parent_id = $2, archived_at = updated_at + interval '1 minute' WHERE id = $1",
		second, first); err != nil {
		t.Fatal(err)
	}
	var shown storedIssue
	decodeAnswer(t, w.send(t, "bob", http.MethodGet, path, "", http.StatusOK), &shown)
	var stored storedIssue
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT id, workspace_id, project_id, sequence_id, name, state_id, priority, parent_id, sort_order,
		completed_at, archived_at, created_by_id, updated_by_id, created_at, updated_at,
		ARRAY(SELECT assignee_id FROM issue_assignees WHERE issue_id = i.id AND deleted_at IS NULL ORDER BY assignee_id),
		ARRAY(SELECT label_id FROM issue_labels WHERE issue_id = i.id AND deleted_at IS NULL ORDER BY label_id)
		FROM issues i WHERE id = $1`, second).Scan(&stored.ID, &stored.WorkspaceID, &stored.ProjectID, &stored.SequenceID, &stored.Name,
		&stored.StateID, &stored.Priority, &stored.ParentID, &stored.SortOrder, &stored.CompletedAt, &stored.ArchivedAt, &stored.CreatedByID,
		&stored.UpdatedByID, &stored.CreatedAt, &stored.UpdatedAt, &stored.AssigneeIDs, &stored.LabelIDs); err != nil {
		t.Fatal(err)
	}
	shown.inUTC()
	stored.inUTC()
	if !reflect.DeepEqual(shown, stored) || stored.ParentID == nil || stored.CompletedAt == nil || stored.ArchivedAt == nil ||
		*stored.UpdatedByID == *stored.CreatedByID || len(stored.LabelIDs) != 0 || !stored.CreatedAt.Before(*stored.CompletedAt) ||
		!stored.CompletedAt.Before(stored.UpdatedAt) || !stored.UpdatedAt.Before(*stored.ArchivedAt) {
		t.Errorf("getIssue answers %+v; want the work item as stored, %+v, under the first, made, completed, changed last by bob and "+
			"archived at four moments in that order, without labels", shown, stored)
	}
}

// storedIssue is every field of a work item, as getIssue answers it and as
// the database has it.
type storedIssue struct {
	ID          uuid.UUID   `json:"id"`
	WorkspaceID uuid.UUID   `json:"workspace_id"`
	ProjectID   uuid.UUID   `json:"project_id"`
	SequenceID  int         `json:"sequence_id"`
	Name        string      `json:"name"`
	StateID     uuid.UUID   `json:"state_id"`
	Priority    string      `json:"priority"`
	ParentID    *uuid.UUID  `json:"parent_id"`
	SortOrder   float64     `json:"sort_order"`
	CompletedAt *time.Time  `json:"completed_at"`
	ArchivedAt  *time.Time  `json:"archived_at"`
	CreatedByID *uuid.UUID  `json:"created_by_id"`
	UpdatedByID *uuid.UUID  `json:"updated_by_id"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	AssigneeIDs []uuid.UUID `json:"assignee_ids"`
	LabelIDs    []uuid.UUID `json:"label_ids"`
}

// inUTC puts each moment of s in UTC, where an answer and the database
// read it in zones of their own.
func (s *storedIssue) inUTC() {
	s.CreatedAt, s.UpdatedAt = s.CreatedAt.UTC(), s.UpdatedAt.UTC()
	for _, at := range []**time.Time{&s.CompletedAt, &s.ArchivedAt} {
		if *at != nil {
			utc := (*at).UTC()
			*at = &utc
		}
	}
}
````

`server/internal/bootstrap/issue_writes_test.go`（修改，2 处）：

````old server/internal/bootstrap/issue_writes_test.go
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
````
````new server/internal/bootstrap/issue_writes_test.go
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
````
````old server/internal/bootstrap/issue_writes_test.go
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
	item := w.newWorkItem(t)
	after := time.Now()

	rows, err := w.pool.Query(pgtest.Soon(t), `
		SELECT 'issues', created_at, updated_at, created_by_id = $2 AND updated_by_id = $2 FROM issues WHERE id = $1
		UNION ALL SELECT 'issue_assignees', created_at, updated_at, created_by_id = $2 AND updated_by_id = $2 FROM issue_assignees
			WHERE issue_id = $1
		UNION ALL SELECT 'issue_labels', created_at, updated_at, created_by_id = $2 AND updated_by_id = $2 FROM issue_labels
			WHERE issue_id = $1
		UNION ALL SELECT 'issue_subscribers', created_at, updated_at, created_by_id = $2 AND updated_by_id = $2 FROM issue_subscribers
			WHERE issue_id = $1
		UNION ALL SELECT 'issue_activities', created_at, updated_at, created_by_id = $2 AND updated_by_id = $2 AND actor_id = $2
			FROM issue_activities WHERE issue_id = $1`, item, w.ids["alice"])
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
````
````new server/internal/bootstrap/issue_writes_test.go
// createIssue as bootstrap wires it makes every row it writes at the time
// of its request, one moment, from the clock issue.New takes, and as its
// caller (M4 design 3.7): alice creates a work item in Web of the world,
// in Todo, assigned to bob and labelled Bug. The work item, its assignee,
// its label, its two subscribers, alice and bob, and its two activities,
// in both of which alice acts, are seven rows, each made and last changed
// by alice at one moment within the request. The second activity names
// bob by his display name as identity has it (bootstrap/ports.go's
// issueProfiles).
func TestCreatingAWorkItemStampsItsRows(t *testing.T) {
	w := newIssueWorld(t)
	before := time.Now().Truncate(time.Microsecond)
	item := w.newWorkItem(t)
	after := time.Now()

	var made time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT created_at FROM issues WHERE id = $1", item).Scan(&made); err != nil {
		t.Fatal(err)
	}
	if made.Before(before) || made.After(after) {
		t.Errorf("the work item made at %v; want a moment within %v to %v", made, before, after)
	}
	activity := "issue_activities made by alice made, changed by alice made, of alice"
	want := []string{activity, activity, "issue_assignees made by alice made, changed by alice made, of bob",
		"issue_labels made by alice made, changed by alice made", "issue_subscribers made by alice made, changed by alice made, of alice",
		"issue_subscribers made by alice made, changed by alice made, of bob", "issues made by alice made, changed by alice made"}
	if got := w.workItemStamps(t, item); !slices.Equal(got, want) {
		t.Errorf("the work item's rows:\n%q\nwant\n%q", got, want)
	}
````

- [ ] **Step 2: 事务的连接**

`server/internal/bootstrap/issue_writes_test.go`（修改，4 处）：

````old server/internal/bootstrap/issue_writes_test.go
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
````
````new server/internal/bootstrap/issue_writes_test.go
// work item read again under it, the decision under it, its checks of the
// states, the members and the labels, its number, its writes, the display
// names, and its answer. The issue module and the Authorizer are wired as
// bootstrap wires them, on a pool of one connection (poolOfOne): a
// statement sent through the pool rather than the transaction would wait
// for a second connection that never comes, and its request fail at the
// request's deadline. On Web of the world, alice creates a work item in
// Todo, assigned to bob and labelled Bug; with bob made Web's default
// assignee, one without assignees, which he takes; one labelled with a
// label no project has, which the check under the lock refuses (422); walt,
// who sees Web, is refused before the lock (403). She changes the first:
// into Done, assigned to herself instead of bob, without Bug; then its name
// as it is, which writes nothing; then with a label no project has (422),
// and walt is refused (403); and she reads it, without a transaction. With
// Web archived, her creation and her change are refused under the lock
````
````old server/internal/bootstrap/issue_writes_test.go
	send(alice, w.fullIssue(), http.StatusCreated)
````
````new server/internal/bootstrap/issue_writes_test.go
	var first struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, send(alice, w.fullIssue(), http.StatusCreated), &first)
````
````old server/internal/bootstrap/issue_writes_test.go
	send(w.ids["walt"], `{"name":"Fix"}`, http.StatusForbidden)
````
````new server/internal/bootstrap/issue_writes_test.go
	send(w.ids["walt"], `{"name":"Fix"}`, http.StatusForbidden)
	change := func(caller uuid.UUID, body string, want int) string {
		t.Helper()
		return route.answerWithin(t, w.contract, http.MethodPatch, "/api/v0/issues/"+first.ID.String(), caller, body, want)
	}
	var changed struct {
		AssigneeIDs []uuid.UUID `json:"assignee_ids"`
		LabelIDs    []uuid.UUID `json:"label_ids"`
	}
	if decodeAnswer(t, change(alice, `{"state_id":"`+w.done.String()+`","assignee_ids":["`+alice.String()+`"],"label_ids":[]}`,
		http.StatusOK), &changed); !slices.Equal(changed.AssigneeIDs, []uuid.UUID{alice}) || len(changed.LabelIDs) != 0 {
		t.Errorf("the work item changed is assigned to %v, labelled %v; want alice alone, no label", changed.AssigneeIDs, changed.LabelIDs)
	}
	change(alice, `{"name":"Fix"}`, http.StatusOK)
	change(alice, `{"label_ids":["`+uuid.NewV7().String()+`"]}`, http.StatusUnprocessableEntity)
	change(w.ids["walt"], `{"name":"Mend"}`, http.StatusForbidden)
	route.answerWithin(t, w.contract, http.MethodGet, "/api/v0/issues/"+first.ID.String(), alice, "", http.StatusOK)
````
````old server/internal/bootstrap/issue_writes_test.go
	send(alice, `{"name":"Fix"}`, http.StatusConflict)
````
````new server/internal/bootstrap/issue_writes_test.go
	send(alice, `{"name":"Fix"}`, http.StatusConflict)
	change(alice, `{"name":"Mend"}`, http.StatusConflict)
````

- [ ] **Step 3: 测试和 lint**

Run: `go -C server test -count=1 -run 'TestCreatingAWorkItemStampsItsRows$|TestUpdatingAWorkItemStampsItsRows$|TestAWorkItemReadsAsStored$|TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection$' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/bootstrap/issue_updates_test.go server/internal/bootstrap/issue_world_test.go server/internal/bootstrap/issue_writes_test.go
```
```bash
git commit -m "test(M4/P1): a work item's change stamps its rows, reads as stored, runs on its transaction

Every row a change writes is its caller's at one moment within the
request, and the rows it leaves keep their stamps; the activity names
accounts by their display names and states and labels by their names. A
work item reads back as stored, every field its own, so its last writer
and moment show above the HTTP tests too; and every statement of a
change runs on its transaction's connection.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p1b.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；静态是 `make lint-web` 的文案和键的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T7.1` | `updateIssue` 经池读回答，不经事务的连接 | `internal/bootstrap` 的 `TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`（`make test`） | 组合 |
| `T7.2` | 回答把创建者当作最后修改者 | `internal/modules/issue/adapter/http` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemReadsAsStored`（`make test`） | 单元；组合 |
| `T7.3` | 回答把完成时间当作归档时间 | `internal/modules/issue/adapter/http` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemReadsAsStored`（`make test`） | 单元；组合 |
| `T7.4` | `updateIssue` 写的行用工作项创建的时刻，不用自己的 | `internal/modules/issue/app` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestUpdatingAWorkItemStampsItsRows`（`make test`）、故事 `i2-update-issue`（Task 14 起） | 单元；组合；端到端 |

本 Task 的检查也发现较早的 Task 的变异 `T2.1`、`T5.1`、`T5.22`（列在各自的 Task 的表里，标"（Task 7 起）"）：本 Task 做完之后，在本 Task 的树上重跑它们（`mutants_p1b.py`，`MUT_LATER=7`）。

---

### Task 8: `platform/gate`：按键的有界、先来先得的闸门

**Files:**
- Create: `server/internal/platform/gate/gate.go`、`server/internal/platform/gate/gate_test.go`

**Interfaces:**（spec 2.10、第 3 节第 10、11 条；M4 设计 3.5、6.2、11.11）
- `Limits{PerKey, MaxQueue int; MaxWait time.Duration}`；`Keyed[K comparable]`、`New[K](l Limits) *Keyed[K]`；`Enter(ctx, key) (leave func(), err error)`；`ErrFull`、`ErrTimeout`。
- 一把互斥锁下的 `map[K]*queue{holders int; waiting []*waiter}`：
  - 有空位、没有人在等时立即进入；否则排在队尾，队列已有 `MaxQueue` 个时立即 `ErrFull`；
  - 等到 `MaxWait` 时 `ErrTimeout`，`ctx` 结束时 `ctx.Err()`，进入之前已结束的也不进；
  - 放弃的离开队列；正好在放弃时被交到位置的，把位置交给下一个；
  - `release` 把位置直接交给队首（有人等时没有空位）；没人持有、没人等的键从 map 里删去；
  - `leave` 只生效一次（`sync.Once`）。
- 不依赖任何模块，只用标准库（`archtest` 的平台规则照旧）。

**Tests:**（单元，表和 `-race`；`entry`、`enter`、`returned`、`held`、`waits`、`queued`、`forgotten`、`idle`）
- `TestAKeyAdmitsPerKeyCallersAtOnce`：`PerKey` 个立即进入，下一个等到有人离开；别的键不算；都离开之后不留键。
- `TestWaitersAreServedInTheOrderTheyCame`：等着的按到来的次序进入；排在中间的一个放弃之后，其余的照旧按次序进入。
- `TestAFullQueueRefusesAtOnce`：`MaxQueue` 0 和 2，满时立即 `ErrFull`，不等。
- `TestAWaiterGivesUp`：`MaxWait` 到 `ErrTimeout`；取消、期限各 `ctx.Err()`（都在看到它排着之后）；进入之前已取消的不进，哪怕有空位；放弃的不占位置，之后的照样进入。
- `TestLeavingTwiceLeavesOnce`：第二次 `leave` 不放出别人的位置。
- `TestUnderLoadAKeyIsNeverHeldByMoreThanPerKey`：32 个 goroutine 各 1000 次、三个键、随机的期限（不带 `-race` 每次也走到放弃时被交到位置的一支），任何时刻一个键至多 `PerKey` 个持有者，有人等的键没有空位（每次 `Enter` 之后在闸门的锁下查，`idle`）；最后不留键。

- [ ] **Step 1: 测试**

`server/internal/platform/gate/gate_test.go`（新文件，311 行）：

````file server/internal/platform/gate/gate_test.go
package gate

import (
	"context"
	"errors"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// long is a wait no test reaches: a caller that waits it has waited in
// vain.
const long = time.Minute

// entry is a caller's Enter, run on a goroutine of its own: done is closed
// when it returns, with its leave and its error.
type entry struct {
	done  chan struct{}
	leave func()
	err   error
}

// enter calls g.Enter(ctx, key) on a goroutine of its own.
func enter(g *Keyed[string], ctx context.Context, key string) *entry {
	e := &entry{done: make(chan struct{})}
	go func() {
		defer close(e.done)
		e.leave, e.err = g.Enter(ctx, key)
	}()
	return e
}

// returned waits a second at most for e to return, and fails the test when
// it does not.
func (e *entry) returned(t *testing.T) *entry {
	t.Helper()
	select {
	case <-e.done:
	case <-time.After(time.Second):
		t.Fatal("Enter did not return within 1s")
	}
	return e
}

// held waits like returned for e to return, and fails the test unless e
// holds the key.
func (e *entry) held(t *testing.T) *entry {
	t.Helper()
	if e.returned(t); e.err != nil {
		t.Fatalf("Enter() = %v, want the key", e.err)
	}
	return e
}

// waits checks that e has not returned a little while after it began.
func (e *entry) waits(t *testing.T) {
	t.Helper()
	select {
	case <-e.done:
		t.Fatalf("Enter returned %v, want it waiting", e.err)
	case <-time.After(20 * time.Millisecond):
	}
}

// waiting is how many callers wait for key, under g's lock; queued waits a
// second at most until n do.
func (g *Keyed[K]) waiting(key K) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	if q := g.keys[key]; q != nil {
		return len(q.waiting)
	}
	return 0
}

func queued(t *testing.T, g *Keyed[string], key string, n int) {
	t.Helper()
	for deadline := time.Now().Add(time.Second); g.waiting(key) != n; time.Sleep(time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("%d waiting for %s, want %d", g.waiting(key), key, n)
		}
	}
}

// forgotten checks that g keeps no key: none held, none waited for.
func forgotten(t *testing.T, g *Keyed[string]) {
	t.Helper()
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.keys) != 0 {
		t.Errorf("the gate keeps %d keys, want none", len(g.keys))
	}
}

// idle is how many places of g's keys are free while callers wait for them,
// under g's lock: none, since a holder who leaves hands his place to the
// first caller waiting.
func (g *Keyed[K]) idle() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	n := 0
	for _, q := range g.keys {
		if len(q.waiting) > 0 {
			n += g.limits.PerKey - q.holders
		}
	}
	return n
}

// PerKey callers hold a key at once, each entering at once; the next waits
// until one of them leaves, then holds it. Another key's holders do not
// count. Once every caller has left, the gate keeps nothing.
func TestAKeyAdmitsPerKeyCallersAtOnce(t *testing.T) {
	g := New[string](Limits{PerKey: 2, MaxQueue: 8, MaxWait: long})
	ctx := context.Background()
	a, b := enter(g, ctx, "web").held(t), enter(g, ctx, "web").held(t)
	ops := enter(g, ctx, "ops").held(t)
	c := enter(g, ctx, "web")
	c.waits(t)
	a.leave()
	c.held(t)
	b.leave()
	c.leave()
	ops.leave()
	forgotten(t, g)
}

// The callers waiting for a key are served in the order they came, each
// when the one before it leaves; one who gives up while others wait behind
// him leaves their order as it was.
func TestWaitersAreServedInTheOrderTheyCame(t *testing.T) {
	g := New[string](Limits{PerKey: 1, MaxQueue: 8, MaxWait: long})
	ctx := context.Background()
	holder := enter(g, ctx, "web").held(t)
	quitting, quit := context.WithCancel(ctx)
	defer quit()
	var waiters []*entry
	var quitter *entry
	for n := range 4 {
		if n == 1 {
			quitter = enter(g, quitting, "web")
		} else {
			waiters = append(waiters, enter(g, ctx, "web"))
		}
		queued(t, g, "web", n+1)
	}
	quit()
	if quitter.returned(t); !errors.Is(quitter.err, context.Canceled) || quitter.leave != nil {
		t.Fatalf("the second waiter's Enter() = %v, want context.Canceled and nothing to leave", quitter.err)
	}
	leave := holder.leave
	for i, w := range waiters {
		for _, later := range waiters[i+1:] {
			later.waits(t)
		}
		leave()
		leave = w.held(t).leave
	}
	leave()
	forgotten(t, g)
}

// A caller who finds MaxQueue others waiting is refused at once with
// ErrFull, holding nothing; with MaxQueue 0, none waits. The key is not
// forgotten while it is held.
func TestAFullQueueRefusesAtOnce(t *testing.T) {
	for _, maxQueue := range []int{0, 2} {
		g := New[string](Limits{PerKey: 1, MaxQueue: maxQueue, MaxWait: long})
		ctx := context.Background()
		holder := enter(g, ctx, "web").held(t)
		var waiters []*entry
		for n := range maxQueue {
			waiters = append(waiters, enter(g, ctx, "web"))
			queued(t, g, "web", n+1)
		}
		if refused := enter(g, ctx, "web").returned(t); !errors.Is(refused.err, ErrFull) || refused.leave != nil {
			t.Errorf("MaxQueue %d: Enter() with the queue full = %v, want ErrFull and nothing to leave", maxQueue, refused.err)
		}
		leave := holder.leave
		for _, w := range waiters {
			leave()
			leave = w.held(t).leave
		}
		leave()
		forgotten(t, g)
	}
}

// A waiting caller gives up after MaxWait with ErrTimeout, or with its
// context's error once the context is cancelled or past its deadline: the
// test sees it waiting first. One whose context is done already does not
// take even a free key. Each holds nothing and leaves the queue: the
// holder's leaving forgets the key, and the next caller holds it at once.
func TestAWaiterGivesUp(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, tt := range []struct {
		name    string
		maxWait time.Duration
		ctx     func() (context.Context, context.CancelFunc)
		held    bool // whether the key is held when it enters
		seen    bool // whether the test sees it waiting before its context ends
		cancels bool // whether the test then cancels its context
		want    error
	}{
		{"waited MaxWait", 30 * time.Millisecond, func() (context.Context, context.CancelFunc) { return context.Background(), func() {} }, true,
			false, false, ErrTimeout},
		{"cancelled while it waits", long, func() (context.Context, context.CancelFunc) { return context.WithCancel(context.Background()) },
			true, true, true, context.Canceled},
		// its deadline gives queued half a second to see it waiting first,
		// and leaves returned half of its second to see it give up
		{"past its deadline while it waits", long, func() (context.Context, context.CancelFunc) {
			return context.WithTimeout(context.Background(), 500*time.Millisecond)
		}, true, true, false, context.DeadlineExceeded},
		{"cancelled before", long, func() (context.Context, context.CancelFunc) { return cancelled, func() {} }, false, false, false,
			context.Canceled},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := New[string](Limits{PerKey: 1, MaxQueue: 8, MaxWait: tt.maxWait})
			var holder *entry
			if tt.held {
				holder = enter(g, context.Background(), "web").held(t)
			}
			ctx, cancel := tt.ctx()
			defer cancel()
			gaveUp := enter(g, ctx, "web")
			if tt.seen {
				queued(t, g, "web", 1)
			}
			if tt.cancels {
				cancel()
			}
			if gaveUp.returned(t); !errors.Is(gaveUp.err, tt.want) || gaveUp.leave != nil {
				t.Errorf("Enter() = %v, want %v and nothing to leave", gaveUp.err, tt.want)
			}
			if g.waiting("web") != 0 {
				t.Errorf("%d still waiting, want none", g.waiting("web"))
			}
			if holder != nil {
				holder.leave()
			}
			forgotten(t, g)
			enter(g, context.Background(), "web").held(t).leave()
		})
	}
}

// Leaving twice leaves once: the second call does not give up the key of
// the caller who holds it next.
func TestLeavingTwiceLeavesOnce(t *testing.T) {
	g := New[string](Limits{PerKey: 1, MaxQueue: 8, MaxWait: long})
	ctx := context.Background()
	first := enter(g, ctx, "web").held(t)
	first.leave()
	second := enter(g, ctx, "web").held(t)
	first.leave()
	third := enter(g, ctx, "web")
	third.waits(t)
	second.leave()
	third.held(t).leave()
	forgotten(t, g)
}

// Under load, on a few keys, with waits that time out and contexts
// cancelled at random, so that a key is handed to a caller as he gives up:
// never more than PerKey callers hold a key at once, no key has a free
// place while callers wait for it, every caller that enters leaves, and
// once all have, the gate keeps nothing (run with and without -race).
func TestUnderLoadAKeyIsNeverHeldByMoreThanPerKey(t *testing.T) {
	const perKey = 2
	g := New[string](Limits{PerKey: perKey, MaxQueue: 16, MaxWait: 2 * time.Millisecond})
	keys := []string{"web", "ops", "api"}
	var holders [3]atomic.Int32
	var most atomic.Int32
	var entered, refused, idle atomic.Int64
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			for range 1000 {
				k := rand.IntN(len(keys))
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rand.IntN(3000))*time.Microsecond)
				leave, err := g.Enter(ctx, keys[k])
				idle.Add(int64(g.idle()))
				if err != nil {
					refused.Add(1)
					cancel()
					continue
				}
				entered.Add(1)
				n := holders[k].Add(1)
				for m := most.Load(); n > m && !most.CompareAndSwap(m, n); m = most.Load() {
				}
				time.Sleep(time.Duration(rand.IntN(200)) * time.Microsecond)
				holders[k].Add(-1)
				leave()
				cancel()
			}
		})
	}
	wg.Wait()
	if idle.Load() != 0 {
		t.Errorf("keys with callers waiting had %d free places, summed over the checks; want none", idle.Load())
	}
	if most.Load() > perKey || entered.Load() == 0 || refused.Load() == 0 {
		t.Errorf("at most %d held a key at once, %d entered, %d refused; want at most %d, some of each", most.Load(), entered.Load(),
			refused.Load(), perKey)
	}
	forgotten(t, g)
}
````

- [ ] **Step 2: 闸门**

`server/internal/platform/gate/gate.go`（新文件，135 行）：

````file server/internal/platform/gate/gate.go
// Package gate bounds how many callers hold a key at once, within this
// process (M4 design 3.5, 11.11): a caller enters a key's gate, waits in
// the key's queue, first come first served, while as many others hold it
// as its bound allows, and leaves it once. A key nobody holds or waits for
// takes no memory. It uses only the standard library.
package gate

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"
)

var (
	// ErrFull answers a caller who finds as many callers waiting for the
	// key as the queue holds.
	ErrFull = errors.New("gate: the queue is full")
	// ErrTimeout answers a caller who waited for the key the longest wait.
	ErrTimeout = errors.New("gate: waited too long")
)

// Limits are a gate's bounds: at most PerKey callers hold a key at once,
// at least 1; at most MaxQueue others wait for it, 0 for none; each waits
// at most MaxWait (the configuration validates them).
type Limits struct {
	PerKey   int
	MaxQueue int
	MaxWait  time.Duration
}

// Keyed is a gate per key of type K, under one lock.
type Keyed[K comparable] struct {
	limits Limits
	mu     sync.Mutex
	keys   map[K]*queue
}

// queue is a key's state: how many callers hold it, and those waiting for
// it, in the order they came.
type queue struct {
	holders int
	waiting []*waiter
}

// waiter is a caller waiting for a key: ready is closed when it is handed
// the key, and granted set, under the gate's lock.
type waiter struct {
	ready   chan struct{}
	granted bool
}

// New returns a gate with the limits l.
func New[K comparable](l Limits) *Keyed[K] {
	return &Keyed[K]{limits: l, keys: map[K]*queue{}}
}

// Enter returns once the caller holds key, with the function that leaves
// it, which the caller calls once on every path, and which does nothing
// when called again. While PerKey callers hold key, or others wait for it
// already, the caller waits behind them, holding nothing; it is refused at
// once with ErrFull when MaxQueue others wait, and gives up with
// ErrTimeout after MaxWait, or with ctx's error once ctx is done, before
// or while it waits: then it holds nothing and leaves the queue.
func (g *Keyed[K]) Enter(ctx context.Context, key K) (leave func(), err error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	g.mu.Lock()
	q := g.keys[key]
	if q == nil {
		q = &queue{}
		g.keys[key] = q
	}
	if q.holders < g.limits.PerKey && len(q.waiting) == 0 {
		q.holders++
		g.mu.Unlock()
		return g.leaver(key), nil
	}
	if len(q.waiting) >= g.limits.MaxQueue {
		g.mu.Unlock()
		return nil, ErrFull
	}
	w := &waiter{ready: make(chan struct{})}
	q.waiting = append(q.waiting, w)
	g.mu.Unlock()

	timer := time.NewTimer(g.limits.MaxWait)
	defer timer.Stop()
	select {
	case <-w.ready:
		return g.leaver(key), nil
	case <-timer.C:
		err = ErrTimeout
	case <-ctx.Done():
		err = ctx.Err()
	}
	g.mu.Lock()
	if w.granted {
		// Handed the key as it gave up: it passes the key on.
		g.mu.Unlock()
		g.release(key)
		return nil, err
	}
	q.waiting = slices.DeleteFunc(q.waiting, func(other *waiter) bool { return other == w })
	g.mu.Unlock()
	return nil, err
}

// leaver is the function that leaves key once.
func (g *Keyed[K]) leaver(key K) func() {
	var once sync.Once
	return func() { once.Do(func() { g.release(key) }) }
}

// release gives key up: to the first caller waiting for it, who then holds
// it; when none waits, one holder fewer, and a key nobody holds is
// forgotten.
func (g *Keyed[K]) release(key K) {
	g.mu.Lock()
	defer g.mu.Unlock()
	q := g.keys[key]
	if len(q.waiting) > 0 {
		w := q.waiting[0]
		q.waiting = q.waiting[1:]
		w.granted = true
		close(w.ready)
		return
	}
	q.holders--
	if q.holders == 0 {
		delete(g.keys, key)
	}
}
````

- [ ] **Step 3: 测试和 lint**

Run: `go -C server test -count=1 ./internal/platform/gate/`
Expected: `ok`。

Run: `go -C server test -race -count=5 ./internal/platform/gate/`
Expected: `ok`，没有 `DATA RACE`。

Run: `go -C server test -count=1 ./internal/archtest/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

- [ ] **Step 4: 提交**

```bash
git add server/internal/platform/gate/gate.go server/internal/platform/gate/gate_test.go
```
```bash
git commit -m "feat(M4/P1): platform/gate, a bounded first-come gate per key

Keyed admits PerKey callers per key at once; the next ones wait in the
order they came, at most MaxQueue of them, each at most MaxWait or until
its context ends, and a full queue refuses at once. A leaving holder
hands its place to the first waiter; a waiter that gives up just as it
is handed one passes it on. Leaving twice leaves once, and a key no one
holds or waits for keeps no memory.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p1b.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；静态是 `make lint-web` 的文案和键的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T8.1` | 键交给最后来的等待者，不是最先来的 | `internal/platform/gate` 的 全部测试（`make test`） | 单元 |
| `T8.2` | 队列比 `MaxQueue` 多容一个 | `internal/platform/gate` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestAProjectTooBusyForAWriteAnswersServerBusy`（`make test`）（Task 10 起） | 单元；组合 |
| `T8.3` | 离开两次就让出两次，第二次让出的是下一个持有者的 | `internal/platform/gate` 的 全部测试（`make test`） | 单元 |
| `T8.4` | 放弃的等待者留在队列里 | `internal/platform/gate` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestAProjectTooBusyForAWriteAnswersServerBusy`（`make test`）（Task 10 起） | 单元；组合 |
| `T8.5` | 放弃时正好被交到键的人留着它，这个键再也不空 | `internal/platform/gate` 的 `TestUnderLoadAKeyIsNeverHeldByMoreThanPerKey`（`make test`）、`internal/platform/gate` 的 `TestUnderLoadAKeyIsNeverHeldByMoreThanPerKey`（`make test`，`-race -count=5`） | 单元 |
| `T8.6` | 等待者不看 `MaxWait`，一直等 | `internal/platform/gate` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestAProjectTooBusyForAWriteAnswersServerBusy`（`make test`）（Task 10 起） | 单元；组合 |
| `T8.7` | 等待者的 context 结束时不放弃 | `internal/platform/gate` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestAWriteWhoseRequestEndsWhileItWaitsAtTheGateIsServerBusy`（`make test`）（Task 10 起） | 单元；组合 |
| `T8.8` | context 已结束的调用者照样拿到空着的键 | `internal/platform/gate` 的 全部测试（`make test`） | 单元 |
| `T8.9` | 没人持有、没人等的键不被忘掉 | `internal/platform/gate` 的 全部测试（`make test`） | 单元 |
| `T8.10` | 闸门让比 `PerKey` 多一个的调用者持有键 | `internal/platform/gate` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestAHotProjectHoldsUpNoReadOfAnother`（`make test`）（Task 12 起） | 单元；组合 |
| `T8.11` | 有人等时新来的拿走空位（等价：有人等的键没有空位） | 没有：等价（spec 第 3 节第 11 条，附录 A.3） | — |

---

### Task 9: 锁的期限和 `issue` 的配置

**Files:**
- Create: `server/internal/platform/postgres/lock_timeout_test.go`
- Modify: `server/configs/config.yaml`、`server/configs/embed_test.go`、`server/internal/platform/config/config.go`、`server/internal/platform/config/config_test.go`、`server/internal/platform/config/load_test.go`、`server/internal/platform/config/validate.go`、`server/internal/platform/config/validate_test.go`、`server/internal/platform/postgres/tx.go`

**Interfaces:**（spec 2.11、第 3 节第 15 条；M4 设计 3.5、8.8、11.11）
- `platform/postgres/tx.go`：
  - `Option`、`LockTimeout(d time.Duration) Option`：`TxOptions.BeginQuery` 为 `BEGIN; SET LOCAL lock_timeout = N`，`N` 是向上取整的毫秒数（同一个往返，`fn` 的第一条语句之前；模块的 SQL 里没有 `SET`）；
  - `NewTxManager(pool, commitTimeout, opts ...Option)`：已有的调用不变；
  - `LockNotAvailable(err) bool`：`55P03`，包了几层也认，与 `UniqueViolation` 同一种写法。
- `platform/config`：
  - `config.go`：`Config.Issue IssueConfig`（`koanf:"issue"`）、`IssueConfig{WriteGate WriteGateConfig; WriteLockTimeout time.Duration}`、`WriteGateConfig{PerProject, MaxQueue int; MaxWait time.Duration}`；`LogValue` 记下四个键；
  - `validate.go`：`issue.write_gate.per_project` 至少 1、`max_wait` 为正、`max_queue` 不为负；`issue.write_lock_timeout` 为正，且加上 `issue.write_gate.max_wait` 短于 `server.request_timeout`（在闸门上等到 `max_wait`、再等锁到期的写仍要在请求到期之前答 503）。
- `configs/config.yaml`：四个键的默认值（2、2s、64、5s）和中文说明。

**Tests:**
- 存储（真实的 PostgreSQL）：
  - `TestLockTimeoutBoundsEachWaitOfTheTransaction`（表）：一个连接的池上，`LockTimeout(80ms)` 的事务里 `SHOW lock_timeout` 是 `80ms`，等另一个事务持的表锁约 80 毫秒之后 `55P03`、回滚（等不到期时在测试的 3 秒期限失败）；之后这个连接回到默认（`0`），提交之后也是；1.2 毫秒取整为 `2ms`（向下、就近取整都是 `1ms`）；不带选项的是 `0`；
  - `TestLockNotAvailable`：`55P03` 本身、包了一层的是；`40P01`、文本是 `55P03` 而不是 PostgreSQL 的错误、`nil` 不是。
- 单元：`TestValidateReportsEveryInvalidKey`（三个闸门的键、锁的期限）、`TestValidateCrossKeyRules`（闸门的等待加锁的期限等于、超过请求的期限）、`TestValidateAcceptsTheWriteGatesMinimums`（`per_project` 1、`max_queue` 0 可以）、`TestValidateAcceptsWriteWaitsJustUnderTheRequestTimeout`（两者之和略短于请求的期限可以）、`TestLoadAppliesLayersInOrder`（文件和环境变量的层）、`TestLogValueMasksDatabaseURL`（四个键在日志里；日志按文字比对，四个值各不同于默认值和别的键的值，也不是别的键的值的开头）、`configs` 的 `TestBuiltInProfiles`（默认值）。

- [ ] **Step 1: 锁的期限**

`server/internal/platform/postgres/lock_timeout_test.go`（新文件，103 行）：

````file server/internal/platform/postgres/lock_timeout_test.go
package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// lockTimeout is the transaction's lock_timeout, as SHOW states it.
func lockTimeout(t *testing.T, ctx context.Context, pool *pgxpool.Pool) string {
	t.Helper()
	var timeout string
	if err := postgres.DB(ctx, pool).QueryRow(ctx, "SHOW lock_timeout").Scan(&timeout); err != nil {
		t.Fatal(err)
	}
	return timeout
}

// A TxManager with LockTimeout begins each transaction with lock_timeout
// set for it alone, rounded up to whole milliseconds: a statement that
// waits longer for a lock another transaction holds fails with 55P03
// (LockNotAvailable) after about that long, and its transaction rolls
// back. Its connection, the only one of its pool, is back at the default,
// no timeout, after it, and after a transaction that commits; a TxManager
// without the option leaves the default. A wait that lock_timeout does not
// end ends at the test's deadline, a failure.
func TestLockTimeoutBoundsEachWaitOfTheTransaction(t *testing.T) {
	notes := newNotes(t, 2)
	one, err := postgres.NewPool(context.Background(), config.DatabaseConfig{URL: notes.Config().ConnString(), MaxConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(one.Close)
	holder, err := notes.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = holder.Rollback(context.Background()) })
	if _, err := holder.Exec(context.Background(), "LOCK TABLE notes IN ACCESS EXCLUSIVE MODE"); err != nil {
		t.Fatal(err)
	}

	var set string
	deadline, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	start := time.Now()
	err = postgres.NewTxManager(one, commitTimeout, postgres.LockTimeout(80*time.Millisecond)).WithinTx(deadline,
		func(ctx context.Context) error {
			set = lockTimeout(t, ctx, one)
			return insertNote(ctx, one, 1)
		})
	waited := time.Since(start)
	if !postgres.LockNotAvailable(err) || set != "80ms" || waited < 80*time.Millisecond || waited > 2*time.Second {
		t.Errorf("WithinTx() = %v after %v with lock_timeout %s; want 55P03 after about 80ms, with 80ms", err, waited, set)
	}
	if after := lockTimeout(t, context.Background(), one); after != "0" {
		t.Errorf("lock_timeout after the transaction = %s, want 0", after)
	}

	// 1.2ms: rounded up, 2ms; rounded down or to the nearest, 1ms.
	for _, tt := range []struct {
		opts []postgres.Option
		want string
	}{{[]postgres.Option{postgres.LockTimeout(1200 * time.Microsecond)}, "2ms"}, {nil, "0"}} {
		var got string
		if err := postgres.NewTxManager(one, commitTimeout, tt.opts...).WithinTx(context.Background(), func(ctx context.Context) error {
			got = lockTimeout(t, ctx, one)
			return nil
		}); err != nil || got != tt.want {
			t.Errorf("WithinTx() with %d options: lock_timeout %s, %v; want %s", len(tt.opts), got, err, tt.want)
		}
	}
}

func TestLockNotAvailable(t *testing.T) {
	timedOut := &pgconn.PgError{Code: "55P03"}
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"a lock waited for too long", timedOut, true},
		{"wrapped", fmt.Errorf("update thing: %w", timedOut), true},
		{"another SQLSTATE", &pgconn.PgError{Code: "40P01"}, false},
		{"an error that is not PostgreSQL's", errors.New("55P03"), false},
		{"no error", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := postgres.LockNotAvailable(tt.err); got != tt.want {
				t.Errorf("LockNotAvailable(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
````

`server/internal/platform/postgres/tx.go`（修改，4 处）：

````old server/internal/platform/postgres/tx.go
	"context"
````
````new server/internal/platform/postgres/tx.go
	"context"
	"errors"
````
````old server/internal/platform/postgres/tx.go
	commitTimeout time.Duration
````
````new server/internal/platform/postgres/tx.go
	commitTimeout time.Duration
	begin         pgx.TxOptions
````
````old server/internal/platform/postgres/tx.go
// NewTxManager returns a TxManager on pool.
func NewTxManager(pool *pgxpool.Pool, commitTimeout time.Duration) *TxManager {
	return &TxManager{pool: pool, commitTimeout: commitTimeout}
````
````new server/internal/platform/postgres/tx.go
// Option sets how a TxManager begins its transactions.
type Option func(m *TxManager)

// LockTimeout makes each transaction the TxManager begins wait at most d,
// rounded up to whole milliseconds, for each lock it waits for: its BEGIN
// sets lock_timeout for the transaction alone (SET LOCAL), in the same
// round trip, before any statement of fn. A statement that waits longer
// fails with SQLSTATE 55P03 (LockNotAvailable), and the transaction rolls
// back (M4 design 3.5, 11.11). d must be positive: PostgreSQL reads a
// lock_timeout of 0 as no limit.
func LockTimeout(d time.Duration) Option {
	return func(m *TxManager) {
		m.begin.BeginQuery = fmt.Sprintf("BEGIN; SET LOCAL lock_timeout = %d", (d + time.Millisecond - 1).Milliseconds())
	}
}

// NewTxManager returns a TxManager on pool, with opts.
func NewTxManager(pool *pgxpool.Pool, commitTimeout time.Duration, opts ...Option) *TxManager {
	m := &TxManager{pool: pool, commitTimeout: commitTimeout}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// LockNotAvailable reports whether err, or an error it wraps, is
// lock_not_available (SQLSTATE 55P03): e.g. a statement's that waited for
// a lock longer than its transaction's lock timeout (LockTimeout), or that
// asked with NOWAIT for a lock another transaction held.
func LockNotAvailable(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "55P03"
````
````old server/internal/platform/postgres/tx.go
	tx, err := m.pool.Begin(ctx)
````
````new server/internal/platform/postgres/tx.go
	tx, err := m.pool.BeginTx(ctx, m.begin)
````

- [ ] **Step 2: 配置**

`server/configs/config.yaml`（修改，1 处）：

````old server/configs/config.yaml
  creation_enabled: true

````
````new server/configs/config.yaml
  creation_enabled: true

# 工作项一侧的写（M4 设计 3.5、11.11）
issue:
  # 按项目的写闸门：本进程里同一个项目上同时持有数据库连接的工作项写至多 per_project 个；
  # 其余的写不持连接地排队，至多 max_queue 个，各等至多 max_wait。超出时 503 server_busy，带 Retry-After
  write_gate:
    per_project: 2
    max_wait: 2s
    max_queue: 64
  # 这些写的事务等每一把锁的上限（lock_timeout）；到期 503 server_busy。加上 write_gate.max_wait 必须短于 server.request_timeout
  write_lock_timeout: 5s

````

`server/configs/embed_test.go`（修改，1 处）：

````old server/configs/embed_test.go
				Files:     config.FilesConfig{SizeLimit: 5 << 20},
				Log:       config.LogConfig{Level: tt.level, Format: tt.format},
````
````new server/configs/embed_test.go
				Issue: config.IssueConfig{
					WriteGate:        config.WriteGateConfig{PerProject: 2, MaxWait: 2 * time.Second, MaxQueue: 64},
					WriteLockTimeout: 5 * time.Second,
				},
				Files: config.FilesConfig{SizeLimit: 5 << 20},
				Log:   config.LogConfig{Level: tt.level, Format: tt.format},
````

`server/internal/platform/config/config.go`（修改，3 处）：

````old server/internal/platform/config/config.go
	Workspace WorkspaceConfig `koanf:"workspace"`
````
````new server/internal/platform/config/config.go
	Workspace WorkspaceConfig `koanf:"workspace"`
	Issue     IssueConfig     `koanf:"issue"`
````
````old server/internal/platform/config/config.go
}

// FilesConfig configures uploads. The instance API reports it to clients;
````
````new server/internal/platform/config/config.go
}

// IssueConfig configures the writes on the work-item side (M4 design 3.5,
// 11.11).
type IssueConfig struct {
	// WriteGate bounds the writes on one project's work items that hold a
	// connection at once in this process.
	WriteGate WriteGateConfig `koanf:"write_gate"`
	// WriteLockTimeout bounds each wait of such a write's transaction for a
	// lock (lock_timeout): a write that waits longer answers 503
	// server_busy. It plus WriteGate.MaxWait is less than
	// server.request_timeout.
	WriteLockTimeout time.Duration `koanf:"write_lock_timeout"`
}

// WriteGateConfig is the work items' write gate (platform/gate): at most
// PerProject writes on a project's work items hold a connection at once in
// this process, at most MaxQueue others wait for it, holding none, each at
// most MaxWait; a write beyond answers 503 server_busy.
type WriteGateConfig struct {
	PerProject int           `koanf:"per_project"`
	MaxWait    time.Duration `koanf:"max_wait"`
	MaxQueue   int           `koanf:"max_queue"`
}

// FilesConfig configures uploads. The instance API reports it to clients;
````
````old server/internal/platform/config/config.go
			slog.Bool("creation_enabled", c.Workspace.CreationEnabled),
		),
````
````new server/internal/platform/config/config.go
			slog.Bool("creation_enabled", c.Workspace.CreationEnabled),
		),
		slog.Group("issue",
			slog.Group("write_gate",
				slog.Int("per_project", c.Issue.WriteGate.PerProject),
				slog.Duration("max_wait", c.Issue.WriteGate.MaxWait),
				slog.Int("max_queue", c.Issue.WriteGate.MaxQueue),
			),
			slog.Duration("write_lock_timeout", c.Issue.WriteLockTimeout),
		),
````

`server/internal/platform/config/config_test.go`（修改，1 处）：

````old server/internal/platform/config/config_test.go
		"config.workspace.creation_enabled=false",
````
````new server/internal/platform/config/config_test.go
		"config.workspace.creation_enabled=false",
		"config.issue.write_gate.per_project=9",
		"config.issue.write_gate.max_wait=1.5s",
		"config.issue.write_gate.max_queue=32",
		"config.issue.write_lock_timeout=3s",
````

`server/internal/platform/config/load_test.go`（修改，3 处）：

````old server/internal/platform/config/load_test.go
  creation_enabled: true
````
````new server/internal/platform/config/load_test.go
  creation_enabled: true
issue:
  write_gate:
    per_project: 2
    max_wait: 2s
    max_queue: 64
  write_lock_timeout: 5s
````
````old server/internal/platform/config/load_test.go
			"NERVE_WORKSPACE__CREATION_ENABLED=false",
````
````new server/internal/platform/config/load_test.go
			"NERVE_WORKSPACE__CREATION_ENABLED=false",
			"NERVE_ISSUE__WRITE_GATE__MAX_QUEUE=8",
````
````old server/internal/platform/config/load_test.go
		Files:     FilesConfig{SizeLimit: 1024},                 // environment
		Log:       LogConfig{Level: "debug", Format: "text"},
````
````new server/internal/platform/config/load_test.go
		Issue: IssueConfig{
			WriteGate:        WriteGateConfig{PerProject: 2, MaxWait: 2 * time.Second, MaxQueue: 8}, // max_queue: environment
			WriteLockTimeout: 5 * time.Second,
		},
		Files: FilesConfig{SizeLimit: 1024}, // environment
		Log:   LogConfig{Level: "debug", Format: "text"},
````

`server/internal/platform/config/validate.go`（修改，2 处）：

````old server/internal/platform/config/validate.go
	c.RateLimit.validate(fail)
````
````new server/internal/platform/config/validate.go
	c.RateLimit.validate(fail)
	c.Issue.validate(c.Server.RequestTimeout, fail)
````
````old server/internal/platform/config/validate.go
}

func (r RateLimitConfig) validate(fail func(key, format string, args ...any)) {
````
````new server/internal/platform/config/validate.go
}

func (i IssueConfig) validate(requestTimeout time.Duration, fail func(key, format string, args ...any)) {
	g := i.WriteGate
	if g.PerProject < 1 {
		fail("issue.write_gate.per_project", "must be at least 1, got %d", g.PerProject)
	}
	if g.MaxWait <= 0 {
		fail("issue.write_gate.max_wait", "must be positive, got %s", g.MaxWait)
	}
	if g.MaxQueue < 0 {
		fail("issue.write_gate.max_queue", "must not be negative, got %d", g.MaxQueue)
	}
	switch {
	case i.WriteLockTimeout <= 0:
		fail("issue.write_lock_timeout", "must be positive, got %s", i.WriteLockTimeout)
	case requestTimeout > 0 && g.MaxWait+i.WriteLockTimeout >= requestTimeout:
		// A write may wait up to max_wait at the gate and then up to its
		// lock timeout for a lock; one that waits both out must still
		// answer 503 before its request's deadline cancels it.
		fail("issue.write_lock_timeout", "plus issue.write_gate.max_wait (%s) must be less than server.request_timeout (%s), got %s",
			g.MaxWait, requestTimeout, i.WriteLockTimeout)
	}
}

func (r RateLimitConfig) validate(fail func(key, format string, args ...any)) {
````

`server/internal/platform/config/validate_test.go`（修改，5 处）：

````old server/internal/platform/config/validate_test.go
		Files:     FilesConfig{SizeLimit: 7340032},
		Log:       LogConfig{Level: "info", Format: "json"},
````
````new server/internal/platform/config/validate_test.go
		// Each unlike its default and every other key's value, and not the
		// start of one, so that a key logged from the wrong field shows.
		Issue: IssueConfig{WriteGate: WriteGateConfig{PerProject: 9, MaxWait: 1500 * time.Millisecond, MaxQueue: 32},
			WriteLockTimeout: 3 * time.Second},
		Files: FilesConfig{SizeLimit: 7340032},
		Log:   LogConfig{Level: "info", Format: "json"},
````
````old server/internal/platform/config/validate_test.go
		Database: DatabaseConfig{URL: "", MaxConns: 0},
````
````new server/internal/platform/config/validate_test.go
		Database: DatabaseConfig{URL: "", MaxConns: 0},
		Issue:    IssueConfig{WriteGate: WriteGateConfig{MaxQueue: -1}},
````
````old server/internal/platform/config/validate_test.go
		"ratelimit.password_user.burst: must be at least 1, got 0",
````
````new server/internal/platform/config/validate_test.go
		"ratelimit.password_user.burst: must be at least 1, got 0",
		"issue.write_gate.per_project: must be at least 1, got 0",
		"issue.write_gate.max_wait: must be positive, got 0s",
		"issue.write_gate.max_queue: must not be negative, got -1",
		"issue.write_lock_timeout: must be positive, got 0s",
````
````old server/internal/platform/config/validate_test.go
			want:   "ratelimit.ipv6_prefix_len: must be from 1 to 128, got 129",
		},
````
````new server/internal/platform/config/validate_test.go
			want:   "ratelimit.ipv6_prefix_len: must be from 1 to 128, got 129",
		},
		{
			// A write may wait max_wait at the gate and then its lock
			// timeout for a lock, and must still answer 503 before its
			// request's deadline (M4 design 3.5, 11.11).
			name:   "gate wait and lock timeout reaching the request timeout",
			change: func(c *Config) { c.Issue.WriteLockTimeout = 13500 * time.Millisecond },
			want:   "issue.write_lock_timeout: plus issue.write_gate.max_wait (1.5s) must be less than server.request_timeout (15s), got 13.5s",
		},
		{
			// Each wait alone is well under the request timeout.
			name: "gate wait and lock timeout past the request timeout",
			change: func(c *Config) {
				c.Issue.WriteGate.MaxWait = 11 * time.Second
				c.Issue.WriteLockTimeout = 5 * time.Second
			},
			want: "issue.write_lock_timeout: plus issue.write_gate.max_wait (11s) must be less than server.request_timeout (15s), got 5s",
		},
````
````old server/internal/platform/config/validate_test.go
// Only prod requires the signing key file; dev and test generate a key.
````
````new server/internal/platform/config/validate_test.go
// The write gate's minimums are inclusive: per_project 1 and max_queue 0
// (no queue) are allowed.
func TestValidateAcceptsTheWriteGatesMinimums(t *testing.T) {
	cfg := validConfig()
	cfg.Issue.WriteGate.PerProject = 1
	cfg.Issue.WriteGate.MaxQueue = 0
	if err := cfg.validate(); err != nil {
		t.Errorf("validate() = %v, want nil", err)
	}
}

// A write's gate wait and lock timeout may together come up to just under
// the request timeout.
func TestValidateAcceptsWriteWaitsJustUnderTheRequestTimeout(t *testing.T) {
	cfg := validConfig()
	cfg.Issue.WriteLockTimeout = cfg.Server.RequestTimeout - cfg.Issue.WriteGate.MaxWait - time.Millisecond
	if err := cfg.validate(); err != nil {
		t.Errorf("validate() = %v, want nil", err)
	}
}

// Only prod requires the signing key file; dev and test generate a key.
````

- [ ] **Step 3: 测试和 lint**

Run: `go -C server test -count=1 -run 'TestLockTimeoutBoundsEachWaitOfTheTransaction$|TestLockNotAvailable$' ./internal/platform/postgres/`
Expected: `ok`。

Run: `go -C server test -count=1 ./internal/platform/config/ ./configs/`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/configs/config.yaml server/configs/embed_test.go server/internal/platform/config/config.go server/internal/platform/config/config_test.go server/internal/platform/config/load_test.go server/internal/platform/config/validate.go server/internal/platform/config/validate_test.go server/internal/platform/postgres/lock_timeout_test.go server/internal/platform/postgres/tx.go
```
```bash
git commit -m "feat(M4/P1): a transaction's lock timeout, and the issue writes' configuration

LockTimeout has a TxManager begin each transaction with SET LOCAL
lock_timeout in the same round trip as its BEGIN, so a statement that
waits longer for a lock fails with 55P03 and the transaction rolls back,
and the connection goes back to the pool as it came; LockNotAvailable
tells that failure. issue.write_gate (per_project, max_wait, max_queue)
and issue.write_lock_timeout configure the writes on the work items; a
lock timeout no shorter than the request timeout is refused.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p1b.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；静态是 `make lint-web` 的文案和键的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T9.1` | 锁的期限设在连接的会话上，不只是这个事务 | `internal/platform/postgres` 的 `TestLockTimeoutBoundsEachWaitOfTheTransaction`、`TestLockNotAvailable`（`make test`） | 存储 |
| `T9.2` | 锁的期限向下取整到毫秒（1.5 毫秒等 1 毫秒，不到 1 毫秒的不等） | `internal/platform/postgres` 的 `TestLockTimeoutBoundsEachWaitOfTheTransaction`、`TestLockNotAvailable`（`make test`） | 存储 |
| `T9.3` | `TxManager` 开始事务时不带它的选项 | `internal/platform/postgres` 的 `TestLockTimeoutBoundsEachWaitOfTheTransaction`、`TestLockNotAvailable`（`make test`）、`internal/bootstrap` 的 `TestAProjectTooBusyForAWriteAnswersServerBusy`（`make test`）（Task 10 起） | 存储；组合 |
| `T9.4` | `LockNotAvailable` 把死锁（40P01）当作锁的期限 | `internal/platform/postgres` 的 `TestLockTimeoutBoundsEachWaitOfTheTransaction`、`TestLockNotAvailable`（`make test`） | 存储 |
| `T9.5` | `issue.write_gate.per_project` 可以是 0，谁也进不了的闸门 | `internal/platform/config` 的 全部测试（`make test`） | 单元 |
| `T9.6` | `issue.write_lock_timeout` 可以等于 `server.request_timeout` | `internal/platform/config` 的 全部测试（`make test`） | 单元 |
| `T9.7` | `issue` 的配置不校验 | `internal/platform/config` 的 全部测试（`make test`） | 单元 |
| `T9.8` | `issue.write_gate.max_queue` 可以是负数 | `internal/platform/config` 的 全部测试（`make test`） | 单元 |
| `T9.9` | 随附的配置让一个项目同时进三个写 | `configs` 的 全部测试（`make test`） | 单元 |
| `T9.10` | 日志把 `per_project` 写成 `max_queue` | `internal/platform/config` 的 全部测试（`make test`） | 单元 |

---

### Task 10: 写闸门和锁的期限接到工作项的写上

**Files:**
- Create: `server/internal/bootstrap/gate.go`、`server/internal/bootstrap/gate_test.go`、`server/internal/modules/issue/app/lock_test.go`
- Modify: `api/modules/issue.yaml`、`server/internal/archtest/composition_test.go`、`server/internal/bootstrap/app.go`、`server/internal/bootstrap/app_test.go`、`server/internal/bootstrap/issue_world_test.go`、`server/internal/bootstrap/issue_writes_test.go`、`server/internal/modules/issue/adapter/http/issues_test.go`、`server/internal/modules/issue/adapter/http/update_test.go`、`server/internal/modules/issue/app/create_issue.go`、`server/internal/modules/issue/app/create_issue_test.go`、`server/internal/modules/issue/app/fakes_test.go`、`server/internal/modules/issue/app/get_issue_test.go`、`server/internal/modules/issue/app/lock.go`、`server/internal/modules/issue/app/ports.go`、`server/internal/modules/issue/app/update_issue.go`、`server/internal/modules/issue/module.go`
- Generate: `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**（spec 2.12、第 3 节第 10、13 条；M4 设计 3.5、6.1、6.6、11.11）
- `issue/app`：
  - `ports.go`：`WriteGate`（`Enter(ctx, projectID) (leave func(), err error)`：一个项目一把，第 3 节第 10 条）；
  - `lock.go`：`Locks` 带上 `gate`（`NewLocks(projects, issues, auth, gate, tx)`）；`run`：预检 → `gate.Enter(ctx, w.project)` → `defer leave()` → 事务。预检拒绝的不进闸门；闸门拒绝的不开事务；离开在事务结束之后，每条路上都离开（拒绝、失败、panic）。
  - `create_issue.go`、`update_issue.go`：`Execute` 的说明在预检和事务之间写上项目的写闸门和它的 503。
- `issue/module.go`：`Deps.WriteGate`，交给 `NewLocks`。
- `bootstrap/gate.go`（新）：
  - `busyRetry`（1 秒）；
  - `writeGate`、`newWriteGate(config.WriteGateConfig)`：`platform/gate` 按项目；`ErrFull`、`ErrTimeout`、`context.DeadlineExceeded` 换成 `shared.ServerBusy(busyRetry)`，`context.Canceled` 照原样（没有人等它的回答）；
  - `busyTx`、`newIssueTx(pool, cfg)`：`postgres.NewTxManager` 带 `LockTimeout(issue.write_lock_timeout)`，`LockNotAvailable` 换成同一个 503。这是 `55P03` 换码的唯一一处（设计 3.5）。
- `bootstrap/app.go`：`issue.New` 拿 `newIssueTx(pool, cfg)` 和 `newWriteGate(cfg.Issue.WriteGate)`。
- 契约：`createIssue`、`updateIssue` 声明 `server_busy`，描述写明"太多写时 503，按 `Retry-After` 重试"；`server_busy` 是 M2 的平台码，文案已有（约束 4 不另加）。
- 测试的世界：`issue_writes_test.go` 的 `testIssueConfig`（默认值，锁的期限 4 秒，短于测试的请求期限）和 `newIssueRoute` 的闸门；`app_test.go` 的 `testConfig` 带上它；`issue_world_test.go` 的 `newIssueWorldWith(t, configure)`。
- `archtest/composition_test.go`：命令行不建写闸门（`platform/gate` 进 `TestCommandsComposeNoServerAndNoJobs` 的禁止表）。

**Tests:**
- 单元（`fakeGate` 记下进出，事务还没有结束时的离开记作 `in tx`：`fakeTx.WithinTx` 在事务里设 `callLog.inTx`、`defer` 复位；`fakes_test.go`）：`TestAWriteEntersItsProjectsGateBetweenItsPrecheckAndItsTransaction`、`TestAWriteLeavesTheGateOnEveryPath`（锁下的拒绝、端口的失败、`fn` 的 panic，每条路上都在事务结束之后离开；`fn` 的 panic 照样传给调用者，别的路上没有 panic）、`TestTheGateTakesOnlyWritesThePrecheckLetThrough`（预检拒绝的不进；闸门拒绝的原样、不开事务）；`get_issue_test.go` 的 `newGet` 用记下进出的闸门，`TestGetIssueReadsTheWorkItem` 核对读不进闸门；创建、读、修改的测试照旧通过。
- `bootstrap`：`TestTheWriteGateAnswersItsRefusalsAsServerBusy`（一个位置被持时，队列满、等到 `max_wait`、请求到期都是一秒后重试的 `server_busy`，请求取消的照原样；这时另一个项目的写立即进入；第一个离开之后能进）、`TestABusyTransactionIsServerBusy`（包了一层的 `55P03` 换，`40P01` 照原样，提交照旧）。
- 组合：`TestAProjectTooBusyForAWriteAnswersServerBusy`（一个项目一个位置、排队一个、`max_wait` 1 秒、锁的期限 3 秒：别的事务持 Web 的行时，alice 的修改进了闸门、等行；bob 和 alice 随后的两个创建，一个排队、到 `max_wait` 时 503，另一个在 `max_wait` 之前 503（队列满），都带 `Retry-After: 1`；olga、walt 的修改在位置被占时由预检答 404、403，都在 `max_wait` 之前（预检拒绝的不占闸门）；alice 在 Ops（acme 的另一个项目，测试先建好）的创建这时也在 `max_wait` 之前答 201（每个项目有自己的位置）；alice 的修改在约 3 秒（等锁到期）503，远早于请求的期限；行放开之后修改和创建都通过）；`TestAWriteWhoseRequestEndsWhileItWaitsAtTheGateIsServerBusy`（请求的期限 3 秒、锁的期限 2 秒、一个项目一个位置、`max_wait` 10 秒：别的事务以 `ACCESS EXCLUSIVE` 持 `issue_labels`，bob 的修改在预检里等它；2 秒之后 alice 在 Web 的创建（预检不读 `issue_labels`）占了位置、等 Web 的行；`issue_labels` 放开之后 bob 的修改在闸门上排在她后面，在他的请求的期限（3 秒）503，她的创建在等锁到期时（bob 的修改发出之后约 4 秒，比她自己的请求的期限早约 1 秒）503；bob 的修改在 3 到 4 秒之间：她最早在 4 秒离开闸门，T8.7、T10.16 的 bob 只在那之后得到回答）。
- HTTP（`apitest.Main` 两个方向）：`createIssue`、`updateIssue` 各一行 503 `server_busy`、`Retry-After: 1`。
- `archtest`：`TestCommandsComposeNoServerAndNoJobs`。

- [ ] **Step 1: 契约，生成**

`api/modules/issue.yaml`（修改，4 处）：

````old api/modules/issue.yaml
        assignee. Every row it writes is the caller's, at one moment.
````
````new api/modules/issue.yaml
        assignee. Every row it writes is the caller's, at one moment. A
        project with too many writes on its work items at once answers
        server_busy, with Retry-After: try again then.
````
````old api/modules/issue.yaml
      x-problem-codes: [validation_failed, project.not_found, forbidden, project.archived]
````
````new api/modules/issue.yaml
      x-problem-codes: [validation_failed, project.not_found, forbidden, project.archived, server_busy]
````
````old api/modules/issue.yaml
        is the caller's, at one moment.
````
````new api/modules/issue.yaml
        is the caller's, at one moment. A project with too many writes on
        its work items at once answers server_busy, with Retry-After: try
        again then.
````
````old api/modules/issue.yaml
      x-problem-codes: [validation_failed, issue.not_found, forbidden, project.archived]
````
````new api/modules/issue.yaml
      x-problem-codes: [validation_failed, issue.not_found, forbidden, project.archived, server_busy]
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `2a9bead7ada95eb685dacd36e263c93cc1fee71a69b9a3df26f83f8b3afbeb1d` | 3198 | `api/dist/openapi.yaml` |
| `5d4bef92f149830f81822434559995a3ae66054c4a713f27f2546383e8d14746` | 3492 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: 闸门进入取锁的一步**

`server/internal/modules/issue/adapter/http/issues_test.go`（修改，2 处）：

````old server/internal/modules/issue/adapter/http/issues_test.go
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
````
````new server/internal/modules/issue/adapter/http/issues_test.go
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
		{"a busy project", shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
````
````old server/internal/modules/issue/adapter/http/issues_test.go
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: POST = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

// GET goes to the use case for the caller and the path's work item; the
````
````new server/internal/modules/issue/adapter/http/issues_test.go
		if res.StatusCode != tt.status || body != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
			t.Errorf("%s: POST = %d %s, Retry-After %q; want %d %s", tt.name, res.StatusCode, body, res.Header.Get("Retry-After"), tt.status,
				tt.want)
		}
	}
}

// busy is the answer to a write the server is too busy for (shared's
// server_busy), which comes with Retry-After: 1.
const busy = `{"status":503,"code":"server_busy","title":"Service Unavailable","detail":"The server is busy; retry shortly."}`

// GET goes to the use case for the caller and the path's work item; the
````

`server/internal/modules/issue/adapter/http/update_test.go`（修改，3 处）：

````old server/internal/modules/issue/adapter/http/update_test.go
	"testing"
````
````new server/internal/modules/issue/adapter/http/update_test.go
	"testing"
	"time"
````
````old server/internal/modules/issue/adapter/http/update_test.go
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
````
````new server/internal/modules/issue/adapter/http/update_test.go
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
		{"a busy project", shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
````
````old server/internal/modules/issue/adapter/http/update_test.go
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: PATCH = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
````
````new server/internal/modules/issue/adapter/http/update_test.go
		if res.StatusCode != tt.status || body != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
			t.Errorf("%s: PATCH = %d %s, Retry-After %q; want %d %s", tt.name, res.StatusCode, body, res.Header.Get("Retry-After"), tt.status,
				tt.want)
````

`server/internal/modules/issue/app/create_issue.go`（修改，1 处）：

````old server/internal/modules/issue/app/create_issue.go
// Execute checks in (domain.CheckNewIssue), before any read; then Locks
// runs the write (M4 design 3.5): its precheck, before any lock; then in
// one transaction, the project FOR NO KEY UPDATE and the decision on
// issue.create, and under them, in this order: an archived project refuses
// it (409 project.archived); the dependent rows (check), every problem in
// one 422; the project's next number; the clock; the sort order after the
// greatest of the state's work items; the work item; its assignees, the
// default one too; its labels; its subscribers, the caller and the
// assignees, in id order; its activity: the creation, then each assignee
// by his display name (domain.CreationActivity). All of it is the
// caller's at that moment. The answer is the work item as stored, read
// back in the transaction.
````
````new server/internal/modules/issue/app/create_issue.go
// Execute checks in (domain.CheckNewIssue), before any read; then Locks
// runs the write (M4 design 3.5, 11.11): its precheck, before any lock;
// then the project's write gate, which refuses a write that may wait no
// longer (503 server_busy); then in one transaction, the project
// FOR NO KEY UPDATE and the decision on issue.create, and under them, in
// this order: an archived project refuses it (409 project.archived); the
// dependent rows (check), every problem in one 422; the project's next
// number; the clock; the sort order after the greatest of the state's work
// items; the work item; its assignees, the default one too; its labels;
// its subscribers, the caller and the assignees, in id order; its
// activity: the creation, then each assignee by his display name
// (domain.CreationActivity). All of it is the caller's at that moment. The
// answer is the work item as stored, read back in the transaction.
````

`server/internal/modules/issue/app/create_issue_test.go`（修改，1 处）：

````old server/internal/modules/issue/app/create_issue_test.go
	w.locks = app.NewLocks(w.project, w.issues, w.auth, fakeTx{log: log})
````
````new server/internal/modules/issue/app/create_issue_test.go
	w.locks = app.NewLocks(w.project, w.issues, w.auth, fakeGate{}, fakeTx{log: log})
````

`server/internal/modules/issue/app/fakes_test.go`（修改，3 处）：

````old server/internal/modules/issue/app/fakes_test.go
// it.
````
````new server/internal/modules/issue/app/fakes_test.go
// it. inTx is set while a fakeTx's transaction runs.
````
````old server/internal/modules/issue/app/fakes_test.go
	fail  string
````
````new server/internal/modules/issue/app/fakes_test.go
	fail  string
	inTx  bool
````
````old server/internal/modules/issue/app/fakes_test.go
	f.log.calls = append(f.log.calls, "Begin")
	return fn(context.WithValue(ctx, inTxKey{}, true))
````
````new server/internal/modules/issue/app/fakes_test.go
	f.log.calls = append(f.log.calls, "Begin")
	f.log.inTx = true
	defer func() { f.log.inTx = false }()
	return fn(context.WithValue(ctx, inTxKey{}, true))
}

// fakeGate is the write gate. Without a log, it lets every write in. With
// one, it logs each entry, " outside tx" when it is outside a transaction,
// then refuses the write with refuse, when set, or lets it in and logs its
// leaving, " in tx" when the transaction has not ended.
type fakeGate struct {
	log    *callLog
	refuse error
}

func (f fakeGate) Enter(ctx context.Context, projectID uuid.UUID) (func(), error) {
	if f.log == nil {
		return func() {}, nil
	}
	if err := f.log.add(ctx, "Enter", projectID); err != nil {
		return nil, err
	}
	if f.refuse != nil {
		return nil, f.refuse
	}
	return func() {
		call := line("Leave", projectID)
		if f.log.inTx {
			call += " in tx"
		}
		f.log.calls = append(f.log.calls, call)
	}, nil
````

`server/internal/modules/issue/app/get_issue_test.go`（修改，3 处）：

````old server/internal/modules/issue/app/get_issue_test.go
// newGet is getIssue over web's world (newCreate's), with web's work item
// 7 stored in it: Fix, in Backlog, high, assigned to bob, labelled Bug,
// made and last written by carol, so that alice, who reads it, is neither.
````
````new server/internal/modules/issue/app/get_issue_test.go
// newGet is getIssue over web's world (newCreate's), its write gate one
// that logs, with web's work item 7 stored in it: Fix, in Backlog, high,
// assigned to bob, labelled Bug, made and last written by carol, so that
// alice, who reads it, is neither.
````
````old server/internal/modules/issue/app/get_issue_test.go
	_, w := newCreate()
````
````new server/internal/modules/issue/app/get_issue_test.go
	_, w := newCreate()
	w.locks = app.NewLocks(w.project, w.issues, w.auth, fakeGate{log: w.log}, fakeTx{log: w.log})
````
````old server/internal/modules/issue/app/get_issue_test.go
// A read, outside any transaction and without a lock (M4 design 3.5): the
// work item, then the decision on issue.read in its project, of its
// workspace; the answer is the work item as read.
````
````new server/internal/modules/issue/app/get_issue_test.go
// A read, outside any transaction, without a lock and outside the write
// gate (M4 design 3.5, 11.11): the work item, then the decision on
// issue.read in its project, of its workspace; the answer is the work item
// as read.
````

`server/internal/modules/issue/app/lock.go`（修改，5 处）：

````old server/internal/modules/issue/app/lock.go
// who would be refused takes no lock at all. Then, in the write's
// transaction, the project's row FOR NO KEY UPDATE, its workspace's row
// not locked: the write's first lock, which every write on the project's
// work items, states, labels and memberships also takes, so they run one
// after another; the work item read again under it; and the decision again
// under it, which is final: a removal or a demotion that committed while
// the write waited is seen. A read of a work item takes the same decision
// on its project, without a lock (see). issue.New builds one Locks for
// every use case: none holds an Authorizer or a transaction of its own.
````
````new server/internal/modules/issue/app/lock.go
// who would be refused takes no lock at all, nor the project's place in
// the write gate. Then the write gate, waited for holding no connection
// and no lock (11.11). Then, in the write's transaction, the project's row
// FOR NO KEY UPDATE, its workspace's row not locked: the write's first
// lock, which every write on the project's work items, states, labels and
// memberships also takes, so they run one after another; the work item
// read again under it; and the decision again under it, which is final: a
// removal or a demotion that committed while the write waited is seen. A
// read of a work item takes the same decision on its project, without a
// lock (see). issue.New builds one Locks for every use case: none holds an
// Authorizer, a write gate or a transaction of its own.
````
````old server/internal/modules/issue/app/lock.go
	auth     shared.Authorizer
````
````new server/internal/modules/issue/app/lock.go
	auth     shared.Authorizer
	gate     WriteGate
````
````old server/internal/modules/issue/app/lock.go
// the Authorizer and the transactions.
func NewLocks(projects ProjectGate, issues IssueReader, auth shared.Authorizer, tx shared.TxManager) Locks {
	return Locks{projects: projects, issues: issues, auth: auth, tx: tx}
````
````new server/internal/modules/issue/app/lock.go
// the Authorizer, the write gate and the transactions.
func NewLocks(projects ProjectGate, issues IssueReader, auth shared.Authorizer, gate WriteGate, tx shared.TxManager) Locks {
	return Locks{projects: projects, issues: issues, auth: auth, gate: gate, tx: tx}
````
````old server/internal/modules/issue/app/lock.go
// run takes w's precheck, before any lock and outside any transaction,
// then in one transaction w's lock and decision, and runs fn in it with
// what they hold (M4 design 3.5): w's whole write. A refusal of either is
// its answer, and fn does not run.
````
````new server/internal/modules/issue/app/lock.go
// run takes w's precheck, before any lock and outside any transaction;
// then the write gate of w's project; then in one transaction w's lock and
// decision, and runs fn in it with what they hold (M4 design 3.5, 11.11):
// w's whole write. It leaves the gate once the transaction has ended, on
// every path, a panic's too. A refusal of the precheck, the gate or the
// decision is its answer, and fn does not run.
````
````old server/internal/modules/issue/app/lock.go
		return err
	}
````
````new server/internal/modules/issue/app/lock.go
		return err
	}
	leave, err := l.gate.Enter(ctx, w.project)
	if err != nil {
		return err
	}
	defer leave()
````

`server/internal/modules/issue/app/lock_test.go`（新文件，107 行）：

````file server/internal/modules/issue/app/lock_test.go
package app_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// gated rebuilds w's locks with a write gate that logs, and refuses each
// write with refuse, when set: the use cases' creation and update over
// them.
func gated(w *world, refuse error) (*app.CreateIssue, *app.UpdateIssue) {
	w.locks = app.NewLocks(w.project, w.issues, w.auth, fakeGate{log: w.log, refuse: refuse}, fakeTx{log: w.log})
	create := app.NewCreateIssue(app.CreateIssueDeps{Locks: w.locks, Issues: w.issues, Sequences: w.project, States: w.project,
		Labels: w.project, Members: w.project, Profiles: w.profiles, Clock: clockAt{at: clockNow, log: w.log}})
	update := app.NewUpdateIssue(app.UpdateIssueDeps{Locks: w.locks, Issues: w.issues, States: w.project, Labels: w.project, Members: w.project,
		Profiles: w.profiles, Clock: clockAt{at: clockNow, log: w.log}})
	return create, update
}

// A write enters its project's write gate after its precheck, outside any
// transaction, holding nothing, and begins its transaction only then (M4
// design 3.5, 11.11); it leaves the gate once its transaction has ended,
// after its last statement. A creation enters the gate of the project it
// names, an update that of its work item's project.
func TestAWriteEntersItsProjectsGateBetweenItsPrecheckAndItsTransaction(t *testing.T) {
	_, w, _ := newUpdate()
	create, _ := gated(w, nil)
	if _, err := create.Execute(as(alice), web, domain.NewIssue{Name: "Fix"}); err != nil {
		t.Fatal(err)
	}
	if want := []string{line("Find", web, "outside tx"), line("Authorize", alice, domain.ActionCreate, acme, web, "outside tx"),
		line("Enter", web, "outside tx"), "Begin", line("Lock", web)}; !slices.Equal(w.log.calls[:5], want) ||
		w.log.calls[len(w.log.calls)-1] != line("Leave", web) || slices.Index(w.log.calls, line("Leave", web)) != len(w.log.calls)-1 {
		t.Errorf("a creation's calls:\n%q\nwant them to begin %q and end with Leave web alone", w.log.calls, want)
	}

	_, w, id := newUpdate()
	_, update := gated(w, nil)
	if _, err := update.Execute(as(alice), id, domain.IssuePatch{Name: name("Mend")}); err != nil {
		t.Fatal(err)
	}
	want := slices.Concat(locked(id)[:2], []string{line("Enter", web, "outside tx")}, locked(id)[2:], []string{"Now", line("UpdateIssue", id),
		line("InsertActivities", 1), line("GetIssue", id), line("Leave", web)})
	if !slices.Equal(w.log.calls, want) {
		t.Errorf("an update's calls:\n%q\nwant\n%q", w.log.calls, want)
	}
}

// A write leaves the gate on every path once it has entered it: refused
// under the lock, ended by a failing port, or by a panic, which goes on.
func TestAWriteLeavesTheGateOnEveryPath(t *testing.T) {
	for _, tt := range []struct {
		name      string
		set       func(w *world)
		recovered any // what goes on past Execute: the write's panic; nil for none
	}{
		{"refused under the lock", func(w *world) { w.project.locked.Archived = true }, nil},
		{"a failing port", func(w *world) { w.log.fail = "UpdateIssue" }, nil},
		{"a panic", func(w *world) { w.project.waited = func() { panic("connection lost") } }, "connection lost"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, w, id := newUpdate()
			_, update := gated(w, nil)
			tt.set(w)
			func() {
				defer func() {
					if r := recover(); r != tt.recovered {
						t.Errorf("recovered %v past Execute(); want %v", r, tt.recovered)
					}
				}()
				if _, err := update.Execute(as(alice), id, domain.IssuePatch{Name: name("Mend")}); err == nil {
					t.Error("Execute() = nil, want its refusal or failure")
				}
			}()
			if last := w.log.calls[len(w.log.calls)-1]; !slices.Contains(w.log.calls, line("Enter", web, "outside tx")) || last != line("Leave", web) {
				t.Errorf("calls:\n%q\nwant the gate entered, and left last", w.log.calls)
			}
		})
	}
}

// A caller the precheck refuses takes no place in the gate; a write the
// gate refuses, its project too busy (503 server_busy), begins no
// transaction and has nothing to leave.
func TestTheGateTakesOnlyWritesThePrecheckLetThrough(t *testing.T) {
	_, w, id := newUpdate()
	_, update := gated(w, nil)
	w.auth.answers = []error{shared.Forbidden()}
	if _, err := update.Execute(as(alice), id, domain.IssuePatch{Name: name("Mend")}); !errors.Is(err, shared.Forbidden()) ||
		!slices.Equal(w.log.calls, locked(id)[:2]) {
		t.Errorf("Execute() = %v after\n%q\nwant forbidden after the precheck alone", err, w.log.calls)
	}

	_, w, id = newUpdate()
	busy := shared.ServerBusy(time.Second)
	_, update = gated(w, busy)
	if _, err := update.Execute(as(alice), id, domain.IssuePatch{Name: name("Mend")}); !errors.Is(err, busy) ||
		!slices.Equal(w.log.calls, append(locked(id)[:2], line("Enter", web, "outside tx"))) {
		t.Errorf("Execute() = %v after\n%q\nwant server_busy once the gate refuses it", err, w.log.calls)
	}
}
````

`server/internal/modules/issue/app/ports.go`（修改，1 处）：

````old server/internal/modules/issue/app/ports.go
	DefaultAssigneeID *uuid.UUID
````
````new server/internal/modules/issue/app/ports.go
	DefaultAssigneeID *uuid.UUID
}

// WriteGate bounds the writes on one project's work items that hold a
// connection at once, in this process (M4 design 3.5, 11.11):
// bootstrap/gate.go's, over platform/gate.
type WriteGate interface {
	// Enter returns once a write on the project's work items may begin
	// its transaction, with the function that leaves the gate, which the
	// write calls once its transaction has ended. It waits holding no
	// connection and no lock. A write that may wait no longer is refused
	// (503 server_busy); one whose request is cancelled while it waits
	// ends with context.Canceled. Either holds nothing.
	Enter(ctx context.Context, projectID uuid.UUID) (leave func(), err error)
````

`server/internal/modules/issue/app/update_issue.go`（修改，1 处）：

````old server/internal/modules/issue/app/update_issue.go
// Execute checks p (domain.CheckIssuePatch), before any read; then Locks
// runs the write on the work item id (M4 design 3.5, 6.7): its precheck,
// the work item read and the decision on issue.update in its project,
// before any lock; then in one transaction, the project FOR NO KEY UPDATE,
// the work item read again and the decision again, and under them, in
// this order: an archived project refuses it (409 project.archived); what
// p changes in the work item (domain.ChangesOf), and when it changes
// nothing, nothing is written and the answer is the work item as read;
// the dependent rows it adds (check), every problem in one 422; the clock;
// the work item's fields, its completion by its new state's group when its
// state changes (domain.CompletedAt); the accounts and the labels it
// removes and adds, in id order; the accounts it adds subscribed; its
// activity (domain.UpdateActivity), each account by his display name. All
// of it is the caller's at that moment. The answer is the work item as
// stored, read back in the transaction.
````
````new server/internal/modules/issue/app/update_issue.go
// Execute checks p (domain.CheckIssuePatch), before any read; then Locks
// runs the write on the work item id (M4 design 3.5, 6.7, 11.11): its
// precheck, the work item read and the decision on issue.update in its
// project, before any lock; then the project's write gate, which refuses a
// write that may wait no longer (503 server_busy); then in one
// transaction, the project FOR NO KEY UPDATE, the work item read again and
// the decision again, and under them, in this order: an archived project
// refuses it (409 project.archived); what p changes in the work item
// (domain.ChangesOf), and when it changes nothing, nothing is written and
// the answer is the work item as read; the dependent rows it adds (check),
// every problem in one 422; the clock; the work item's fields, its
// completion by its new state's group when its state changes
// (domain.CompletedAt); the accounts and the labels it removes and adds,
// in id order; the accounts it adds subscribed; its activity
// (domain.UpdateActivity), each account by his display name. All of it is
// the caller's at that moment. The answer is the work item as stored, read
// back in the transaction.
````

`server/internal/modules/issue/module.go`（修改，2 处）：

````old server/internal/modules/issue/module.go
	Pool       *pgxpool.Pool
	Tx         shared.TxManager
````
````new server/internal/modules/issue/module.go
	Pool *pgxpool.Pool
	// Tx are the transactions of the module's writes, each of which waits
	// for a lock at most issue.write_lock_timeout (bootstrap/gate.go).
	Tx shared.TxManager
	// WriteGate is the write gate of each project (bootstrap/gate.go).
	WriteGate  app.WriteGate
````
````old server/internal/modules/issue/module.go
	locks := app.NewLocks(d.Projects, store, d.Authorizer, d.Tx)
````
````new server/internal/modules/issue/module.go
	locks := app.NewLocks(d.Projects, store, d.Authorizer, d.WriteGate, d.Tx)
````

- [ ] **Step 3: `bootstrap` 的闸门、事务和接线**

`server/internal/bootstrap/app.go`（修改，1 处）：

````old server/internal/bootstrap/app.go
	issues := issue.New(issue.Deps{
		Pool: pool, Tx: tx, Clock: clock.System{}, Authorizer: authorizer,
````
````new server/internal/bootstrap/app.go
	issues := issue.New(issue.Deps{
		Pool: pool, Tx: newIssueTx(pool, cfg), WriteGate: newWriteGate(cfg.Issue.WriteGate), Clock: clock.System{}, Authorizer: authorizer,
````

`server/internal/bootstrap/app_test.go`（修改，1 处）：

````old server/internal/bootstrap/app_test.go
		Workspace: config.WorkspaceConfig{CreationEnabled: true},
````
````new server/internal/bootstrap/app_test.go
		Workspace: config.WorkspaceConfig{CreationEnabled: true},
		Issue:     testIssueConfig,
````

`server/internal/bootstrap/gate.go`（新文件，66 行）：

````file server/internal/bootstrap/gate.go
package bootstrap

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	platformgate "github.com/open-nerve/NerveProject/server/internal/platform/gate"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The work-item side's write gate and lock timeout (M4 design 3.5, 11.11),
// and the one place that answers either's refusal as 503 server_busy: a
// project too busy for one more write now, which the client tries again
// after Retry-After.

// busyRetry is the Retry-After of a write refused as server_busy.
const busyRetry = time.Second

// writeGate is platform/gate's gate per project as issue's WriteGate.
type writeGate struct {
	gate *platformgate.Keyed[uuid.UUID]
}

// newWriteGate is the write gate of issue.write_gate.
func newWriteGate(c config.WriteGateConfig) writeGate {
	limits := platformgate.Limits{PerKey: c.PerProject, MaxQueue: c.MaxQueue, MaxWait: c.MaxWait}
	return writeGate{gate: platformgate.New[uuid.UUID](limits)}
}

// Enter enters the project's gate: a write the gate refuses, its queue
// full or its wait at max_wait, or whose request reaches its deadline
// while it waits, is server_busy; one whose request is cancelled keeps
// context.Canceled, which no one waits for the answer of.
func (g writeGate) Enter(ctx context.Context, projectID uuid.UUID) (func(), error) {
	leave, err := g.gate.Enter(ctx, projectID)
	if errors.Is(err, platformgate.ErrFull) || errors.Is(err, platformgate.ErrTimeout) || errors.Is(err, context.DeadlineExceeded) {
		return nil, shared.ServerBusy(busyRetry)
	}
	return leave, err
}

// busyTx are the transactions of the work-item side's writes:
// platform/postgres's with issue.write_lock_timeout, each of whose waits
// for a lock ends there; a transaction ended so (55P03) is server_busy.
type busyTx struct {
	tx shared.TxManager
}

// newIssueTx is the work-item side's transactions on pool.
func newIssueTx(pool *pgxpool.Pool, cfg config.Config) busyTx {
	return busyTx{tx: postgres.NewTxManager(pool, cfg.Database.CommitTimeout, postgres.LockTimeout(cfg.Issue.WriteLockTimeout))}
}

func (b busyTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	err := b.tx.WithinTx(ctx, fn)
	if postgres.LockNotAvailable(err) {
		return shared.ServerBusy(busyRetry)
	}
	return err
}
````

`server/internal/bootstrap/gate_test.go`（新文件，278 行）：

````file server/internal/bootstrap/gate_test.go
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The write gate answers the gate's refusals as server_busy, to retry in a
// second (M4 design 3.5): while a write holds its project's one place, a
// second write on the project is refused at once when the queue is full,
// and when it has waited max_wait; one whose request reaches its deadline
// while it waits is server_busy too, and one whose request is cancelled
// keeps context.Canceled. A write on another project meanwhile enters at
// once: each project has places of its own (11.11). Once the first leaves,
// a write enters.
func TestTheWriteGateAnswersItsRefusalsAsServerBusy(t *testing.T) {
	project := uuid.NewV7()
	waiting := func(limit time.Duration, cancelled bool) func() (context.Context, context.CancelFunc) {
		return func() (context.Context, context.CancelFunc) {
			if !cancelled {
				return context.WithTimeout(context.Background(), limit)
			}
			ctx, cancel := context.WithCancel(context.Background())
			time.AfterFunc(limit, cancel)
			return ctx, cancel
		}
	}
	for _, tt := range []struct {
		name  string
		gate  config.WriteGateConfig
		ctx   func() (context.Context, context.CancelFunc)
		wants func(err error) bool
	}{
		{"the queue full", config.WriteGateConfig{PerProject: 1, MaxWait: time.Minute}, waiting(time.Minute, false), isBusy},
		{"at max_wait", config.WriteGateConfig{PerProject: 1, MaxWait: 30 * time.Millisecond, MaxQueue: 1}, waiting(time.Minute, false), isBusy},
		{"at the deadline", config.WriteGateConfig{PerProject: 1, MaxWait: time.Minute, MaxQueue: 1}, waiting(30*time.Millisecond, false), isBusy},
		{"cancelled", config.WriteGateConfig{PerProject: 1, MaxWait: time.Minute, MaxQueue: 1}, waiting(30*time.Millisecond, true),
			func(err error) bool { return errors.Is(err, context.Canceled) && !isBusy(err) }},
	} {
		g := newWriteGate(tt.gate)
		leave, err := g.Enter(context.Background(), project)
		if err != nil {
			t.Fatalf("%s: the first write: %v, want it in", tt.name, err)
		}
		ctx, cancel := tt.ctx()
		if _, err := g.Enter(ctx, project); !tt.wants(err) {
			t.Errorf("%s: the second write: %v", tt.name, err)
		}
		cancel()
		if other, err := g.Enter(context.Background(), uuid.NewV7()); err != nil {
			t.Fatalf("%s: a write on another project: %v, want it in at once", tt.name, err)
		} else {
			other()
		}
		leave()
		if leave, err := g.Enter(context.Background(), project); err != nil {
			t.Errorf("%s: a write after the first left: %v, want it in", tt.name, err)
		} else {
			leave()
		}
	}
}

// isBusy reports whether err is shared's server_busy to retry in a second.
func isBusy(err error) bool {
	var e *shared.Error
	return errors.As(err, &e) && e.Code == shared.CodeServerBusy && e.RetryDelay == time.Second
}

// A transaction of the work-item side that a wait for a lock ended
// (lock_not_available, 55P03), however wrapped, is server_busy; one that
// ended otherwise ends as it did.
func TestABusyTransactionIsServerBusy(t *testing.T) {
	deadlock := &pgconn.PgError{Code: "40P01"}
	for _, tt := range []struct {
		name  string
		end   error
		wants func(err error) bool
	}{
		{"at its lock timeout", fmt.Errorf("lock the project: %w", &pgconn.PgError{Code: "55P03"}), isBusy},
		{"in a deadlock", deadlock, func(err error) bool { return err == deadlock }},
		{"committed", nil, func(err error) bool { return err == nil }},
	} {
		ran := false
		err := busyTx{tx: endingIn{tt.end}}.WithinTx(context.Background(), func(context.Context) error {
			ran = true
			return nil
		})
		if !ran || !tt.wants(err) {
			t.Errorf("%s: ran %v, ended in %v", tt.name, ran, err)
		}
	}
}

// endingIn is a TxManager whose transactions run their work and end in
// err.
type endingIn struct{ err error }

func (e endingIn) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if err := fn(ctx); err != nil {
		return err
	}
	return e.err
}

// A project with more writes on its work items than the app's
// issue.write_gate and issue.write_lock_timeout allow answers the writes
// beyond them 503 server_busy with Retry-After: 1 (M4 design 3.5, 11.11);
// a write its precheck refuses takes no place in the gate, and a write on
// another project does not wait for it. In an app that lets one write a
// project in at once, queues one more for a second at most, and ends a
// write's wait for a lock at three seconds: a transaction holds Web's row,
// and alice's change of a work item of Web enters the gate and waits for
// the row. Of bob's creation in Web and alice's, sent then, one waits in
// the queue and is refused at max_wait, before alice's change is at its
// lock timeout; the other before it could have waited max_wait, the queue
// full. olga's change of the work item, which she does not see, and
// walt's, who may not change it, are refused by their prechecks, 404 and
// 403, while the gate's one place is taken, before a wait at the gate
// could have ended; alice's creation in Ops, another project of acme, goes
// through before one could have ended too: Ops has places of its own.
// alice's change is refused at the lock timeout, well before the
// request's. Once the row is free, a change and a creation go through.
func TestAProjectTooBusyForAWriteAnswersServerBusy(t *testing.T) {
	w := newIssueWorldWith(t, func(cfg *config.Config) {
		cfg.Issue = config.IssueConfig{WriteGate: config.WriteGateConfig{PerProject: 1, MaxWait: time.Second, MaxQueue: 1},
			WriteLockTimeout: 3 * time.Second}
	})
	item := w.newWorkItem(t)
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	path := "/api/v0/issues/" + item.String()
	change := newRequest(t, http.MethodPatch, w.base+path, w.tokens["alice"], []byte(`{"name":"Mend"}`))
	w.contract.CheckRequest(t, change)
	var creations []*http.Request
	for _, name := range []string{"bob", "alice"} {
		create := newRequest(t, http.MethodPost, w.base+"/api/v0/projects/"+w.web.String()+"/issues", w.tokens[name], []byte(`{"name":"Fix"}`))
		w.contract.CheckRequest(t, create)
		creations = append(creations, create)
	}

	held := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR UPDATE", w.web)
	start := time.Now()
	changed := sendInBackground(change)
	pgtest.WaitForLockWaitBehind(t, w.pool, held.Conn().PgConn(), 5*time.Second)
	sent := time.Now()
	pending := []<-chan answer{sendInBackground(creations[0]), sendInBackground(creations[1])}
	var after []time.Duration
	for range pending {
		var a answer
		select {
		case a = <-pending[0]:
			pending[0] = nil
		case a = <-pending[1]:
			pending[1] = nil
		case <-time.After(3 * time.Second):
			t.Fatal("a creation did not answer within 3s")
		}
		if a.err != nil {
			t.Fatal(a.err)
		}
		after = append(after, time.Since(sent))
		w.wantBusy(t, "a creation", a.res.Request, a.res, a.body)
	}
	if after[0] >= time.Second || after[1] < time.Second || after[1] >= 3*time.Second {
		t.Errorf("the creations answered after %v; want one at once, the queue full, and the other at max_wait, 1s", after)
	}
	for _, refused := range []struct {
		name string
		want int
	}{{"olga", http.StatusNotFound}, {"walt", http.StatusForbidden}} {
		begun := time.Now()
		w.send(t, refused.name, http.MethodPatch, path, `{"name":"Mend"}`, refused.want)
		if took := time.Since(begun); took >= time.Second {
			t.Errorf("%s's change answered %d after %v; want it at once, refused by its precheck before the gate", refused.name,
				refused.want, took)
		}
	}
	begun := time.Now()
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+ops.String()+"/issues", `{"name":"Fix"}`, http.StatusCreated)
	if took := time.Since(begun); took >= time.Second {
		t.Errorf("alice's creation in Ops answered after %v; want it at once: Ops has places of its own", took)
	}
	a := receiveWithin(t, changed, 4*time.Second, "answer to alice's change")
	if a.err != nil {
		t.Fatal(a.err)
	}
	if waited := time.Since(start); waited < 3*time.Second || waited >= 4*time.Second {
		t.Errorf("alice's change answered after %v; want at the lock timeout, 3s", waited)
	}
	w.wantBusy(t, "alice's change", change, a.res, a.body)

	if err := held.Rollback(pgtest.Soon(t)); err != nil {
		t.Fatal(err)
	}
	w.send(t, "alice", http.MethodPatch, path, `{"name":"Mend"}`, http.StatusOK)
	w.send(t, "bob", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", `{"name":"Fix"}`, http.StatusCreated)
}

// A write whose request reaches its deadline while it waits at the gate is
// refused 503 server_busy then (M4 design 3.5, 11.11): its wait ends with
// its request. In an app that lets one write a project in at once, queues
// others for ten seconds, ends a request at three seconds and a write's
// wait for a lock at two seconds: a transaction holds issue_labels
// IN ACCESS EXCLUSIVE MODE, so bob's change of a work item of Web waits in
// its precheck, which reads the work item's labels. alice's creation in
// Web, sent two seconds after his change, passes its precheck, which reads
// no issue_labels, takes the project's one place and waits for Web's row,
// which another transaction holds. Once issue_labels is released, bob's
// change waits at the gate behind her creation and is refused at its
// request's deadline, three seconds after it was sent, before her
// creation, at its lock timeout, four seconds after it at the earliest and
// about a second before her own request's deadline, could leave the gate.
// Once the row is free, a change and a creation go through.
func TestAWriteWhoseRequestEndsWhileItWaitsAtTheGateIsServerBusy(t *testing.T) {
	w := newIssueWorldWith(t, func(cfg *config.Config) {
		cfg.Server.RequestTimeout = 3 * time.Second
		cfg.Issue = config.IssueConfig{WriteGate: config.WriteGateConfig{PerProject: 1, MaxWait: 10 * time.Second, MaxQueue: 1},
			WriteLockTimeout: 2 * time.Second}
	})
	item := w.newWorkItem(t)
	path := "/api/v0/issues/" + item.String()
	change := newRequest(t, http.MethodPatch, w.base+path, w.tokens["bob"], []byte(`{"name":"Mend"}`))
	w.contract.CheckRequest(t, change)
	create := newRequest(t, http.MethodPost, w.base+"/api/v0/projects/"+w.web.String()+"/issues", w.tokens["alice"], []byte(`{"name":"Fix"}`))
	w.contract.CheckRequest(t, create)

	labels := lockedTables(t, w.pool, "issue_labels", "ACCESS EXCLUSIVE")
	sent := time.Now()
	changed := sendInBackground(change)
	pgtest.WaitForLockWaitBehind(t, w.pool, labels.Conn().PgConn(), 5*time.Second)
	time.Sleep(time.Until(sent.Add(2 * time.Second)))
	row := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR UPDATE", w.web)
	created := sendInBackground(create)
	pgtest.WaitForLockWaitBehind(t, w.pool, row.Conn().PgConn(), 5*time.Second)
	release(t, labels)

	a := receiveWithin(t, changed, 5*time.Second, "answer to bob's change")
	if a.err != nil {
		t.Fatal(a.err)
	}
	if took := time.Since(sent); took < 3*time.Second || took >= 4*time.Second {
		t.Errorf("bob's change answered after %v; want at its request's deadline, 3s, before alice's creation left the gate", took)
	}
	w.wantBusy(t, "bob's change", change, a.res, a.body)
	a = receiveWithin(t, created, 5*time.Second, "answer to alice's creation")
	if a.err != nil {
		t.Fatal(a.err)
	}
	if took := time.Since(sent); took < 4*time.Second {
		t.Errorf("alice's creation answered %v after bob's change was sent; want at its lock timeout, 4s", took)
	}
	w.wantBusy(t, "alice's creation", create, a.res, a.body)

	release(t, row)
	w.send(t, "bob", http.MethodPatch, path, `{"name":"Mend"}`, http.StatusOK)
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", `{"name":"Fix"}`, http.StatusCreated)
}

// wantBusy checks that res, the answer to req with body, is as the
// contract declares it and is 503 server_busy with Retry-After: 1.
func (w issueWorld) wantBusy(t *testing.T, what string, req *http.Request, res *http.Response, body []byte) {
	t.Helper()
	w.contract.CheckResponse(t, req, res)
	if res.StatusCode != http.StatusServiceUnavailable || problemCode(t, body) != shared.CodeServerBusy || res.Header.Get("Retry-After") != "1" {
		t.Errorf("%s = %d %s, Retry-After %q; want 503 server_busy, Retry-After 1", what, res.StatusCode, body, res.Header.Get("Retry-After"))
	}
}
````

- [ ] **Step 4: 测试的世界和 `archtest`**

`server/internal/archtest/composition_test.go`（修改，3 处）：

````old server/internal/archtest/composition_test.go
// Authorizer (a module's New), the HTTP server, a rate limiter or a jobs
// client. Each builds only what its commands use: Users, identity's
// NewAdmin and, for `nerve users deactivate`, workspace's NewDeactivator
// and project's NewCascade over issue's Provide (M4 design 6.6), never
// workspace's NewAdmin; Workspaces, workspace's NewAdmin, never the
// deactivation's three, whose cascade no command of it calls. The rule
// follows the static calls from each; the commands are func values it
// calls dynamically, so they are not followed: they only receive the
// composition. Reaching what each builds shows the walk sees the
// composition at all.
````
````new server/internal/archtest/composition_test.go
// Authorizer (a module's New), the HTTP server, a rate limiter, a write
// gate or a jobs client. Each builds only what its commands use: Users,
// identity's NewAdmin and, for `nerve users deactivate`, workspace's
// NewDeactivator and project's NewCascade over issue's Provide (M4 design
// 6.6), never workspace's NewAdmin; Workspaces, workspace's NewAdmin,
// never the deactivation's three, whose cascade no command of it calls.
// The rule follows the static calls from each; the commands are func
// values it calls dynamically, so they are not followed: they only
// receive the composition. Reaching what each builds shows the walk sees
// the composition at all.
````
````old server/internal/archtest/composition_test.go
// server (platform/httpserver), a rate limiter or a jobs client
// (platform/jobs, River).
````
````new server/internal/archtest/composition_test.go
// server (platform/httpserver), a rate limiter, a write gate
// (platform/gate) or a jobs client (platform/jobs, River).
````
````old server/internal/archtest/composition_test.go
		m("internal/platform/httpserver"), m("internal/platform/ratelimit"), m("internal/platform/jobs"), "github.com/riverqueue/river",
````
````new server/internal/archtest/composition_test.go
		m("internal/platform/httpserver"), m("internal/platform/ratelimit"), m("internal/platform/gate"), m("internal/platform/jobs"),
		"github.com/riverqueue/river",
````

`server/internal/bootstrap/issue_world_test.go`（修改，3 处）：

````old server/internal/bootstrap/issue_world_test.go
	"github.com/jackc/pgx/v5/pgxpool"

````
````new server/internal/bootstrap/issue_world_test.go
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
````
````old server/internal/bootstrap/issue_world_test.go
func newIssueWorld(t *testing.T) issueWorld {
	t.Helper()
````
````new server/internal/bootstrap/issue_world_test.go
func newIssueWorld(t *testing.T) issueWorld {
	t.Helper()
	return newIssueWorldWith(t, func(*config.Config) {})
}

// newIssueWorldWith is newIssueWorld in an app whose configuration,
// testConfig's, configure changes first.
func newIssueWorldWith(t *testing.T, configure func(cfg *config.Config)) issueWorld {
	t.Helper()
````
````old server/internal/bootstrap/issue_world_test.go
	w := issueWorld{contract: contract, base: startApp(t, testConfig(t, dbURL, false), migrations.FS()), pool: openPool(t, dbURL),
		tokens: map[string]string{}, ids: map[string]uuid.UUID{}}
````
````new server/internal/bootstrap/issue_world_test.go
	cfg := testConfig(t, dbURL, false)
	configure(&cfg)
	w := issueWorld{contract: contract, base: startApp(t, cfg, migrations.FS()), pool: openPool(t, dbURL), tokens: map[string]string{},
		ids: map[string]uuid.UUID{}}
````

`server/internal/bootstrap/issue_writes_test.go`（修改，3 处）：

````old server/internal/bootstrap/issue_writes_test.go
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
````
````new server/internal/bootstrap/issue_writes_test.go
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
````
````old server/internal/bootstrap/issue_writes_test.go
}

// newIssueRoute is the issue module (issue.New) on pool, with tx its
// transactions, the Authorizer on pool and project's and identity's ports
// converted, all as bootstrap wires them.
````
````new server/internal/bootstrap/issue_writes_test.go
}

// testIssueConfig is the work-item side's configuration of the tests'
// apps: the defaults, the lock timeout under testConfig's request timeout.
var testIssueConfig = config.IssueConfig{WriteGate: config.WriteGateConfig{PerProject: 2, MaxWait: 2 * time.Second, MaxQueue: 64},
	WriteLockTimeout: 4 * time.Second}

// newIssueRoute is the issue module (issue.New) on pool, with tx its
// transactions, the write gate of testIssueConfig, the Authorizer on pool
// and project's and identity's ports converted, all as bootstrap wires
// them.
````
````old server/internal/bootstrap/issue_writes_test.go
	return newRoute(t, issue.New(issue.Deps{Pool: pool, Tx: tx, Clock: clock.System{}, Authorizer: authorizerOn(pool),
		Projects: issueProjects{projects: project.Provide(pool).IssueProjects}, Profiles: issueProfiles{profiles: identity.Provide(pool).PublicProfiles}}).Register)
````
````new server/internal/bootstrap/issue_writes_test.go
	return newRoute(t, issue.New(issue.Deps{Pool: pool, Tx: tx, WriteGate: newWriteGate(testIssueConfig.WriteGate), Clock: clock.System{},
		Authorizer: authorizerOn(pool), Projects: issueProjects{projects: project.Provide(pool).IssueProjects},
		Profiles: issueProfiles{profiles: identity.Provide(pool).PublicProfiles}}).Register)
````

- [ ] **Step 5: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestTheWriteGateAnswersItsRefusalsAsServerBusy$|TestABusyTransactionIsServerBusy$|TestAProjectTooBusyForAWriteAnswersServerBusy$|TestAWriteWhoseRequestEndsWhileItWaitsAtTheGateIsServerBusy$' ./internal/bootstrap/`
Expected: `ok`。

Run: `go -C server test -count=1 ./internal/archtest/`
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
Expected: 121 个通过。

- [ ] **Step 6: 提交**

```bash
git add api/modules/issue.yaml server/internal/archtest/composition_test.go server/internal/bootstrap/app.go server/internal/bootstrap/app_test.go server/internal/bootstrap/gate.go server/internal/bootstrap/gate_test.go server/internal/bootstrap/issue_world_test.go server/internal/bootstrap/issue_writes_test.go server/internal/modules/issue/adapter/http/issues_test.go server/internal/modules/issue/adapter/http/update_test.go server/internal/modules/issue/app/create_issue.go server/internal/modules/issue/app/create_issue_test.go server/internal/modules/issue/app/fakes_test.go server/internal/modules/issue/app/get_issue_test.go server/internal/modules/issue/app/lock.go server/internal/modules/issue/app/lock_test.go server/internal/modules/issue/app/ports.go server/internal/modules/issue/app/update_issue.go server/internal/modules/issue/module.go api/dist/openapi.yaml web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M4/P1): the write gate and the lock timeout on the work items' writes

Every write on the work-item side passes its project's write gate after
its precheck and before its transaction, holding no connection while it
waits, and leaves it once the transaction has ended, on every path. Its
transaction waits at most issue.write_lock_timeout for each lock.
bootstrap answers a full gate, a wait at max_wait or at the request's
deadline, and a lock waited out, as 503 server_busy with Retry-After: 1,
in one place; the contract declares it on createIssue and updateIssue.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p1b.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；静态是 `make lint-web` 的文案和键的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T10.1` | 写在预检之前进闸门：会被拒绝的也占位置，指名工作项的写占的是不存在的项目的位置 | `internal/modules/issue/app` 的 `TestAWriteEntersItsProjectsGateBetweenItsPrecheckAndItsTransaction`、`TestAWriteLeavesTheGateOnEveryPath`、`TestTheGateTakesOnlyWritesThePrecheckLetThrough`（`make test`）、`internal/bootstrap` 的 `TestAProjectTooBusyForAWriteAnswersServerBusy`（`make test`） | 单元；组合 |
| `T10.2` | 写在事务之后才离开闸门，不用 `defer`：panic 留着位置 | `internal/modules/issue/app` 的 `TestAWriteEntersItsProjectsGateBetweenItsPrecheckAndItsTransaction`、`TestAWriteLeavesTheGateOnEveryPath`、`TestTheGateTakesOnlyWritesThePrecheckLetThrough`（`make test`） | 单元 |
| `T10.3` | 写在事务里进闸门，等的时候持着连接 | `internal/modules/issue/app` 的 `TestAWriteEntersItsProjectsGateBetweenItsPrecheckAndItsTransaction`、`TestAWriteLeavesTheGateOnEveryPath`、`TestTheGateTakesOnlyWritesThePrecheckLetThrough`（`make test`）、`internal/bootstrap` 的 `TestAHotProjectHoldsUpNoReadOfAnother`（`make test`）（Task 12 起） | 单元；组合 |
| `T10.4` | 等闸门时请求被取消的写答 `server_busy` | `internal/bootstrap` 的 `TestTheWriteGateAnswersItsRefusalsAsServerBusy`（`make test`） | 单元 |
| `T10.5` | 等闸门时请求到期的写不是 `server_busy`（500） | `internal/bootstrap` 的 `TestTheWriteGateAnswersItsRefusalsAsServerBusy`（`make test`）、`internal/bootstrap` 的 `TestAWriteWhoseRequestEndsWhileItWaitsAtTheGateIsServerBusy`（`make test`） | 单元；组合 |
| `T10.6` | 等闸门等到 `max_wait` 的写不是 `server_busy`（500） | `internal/bootstrap` 的 `TestTheWriteGateAnswersItsRefusalsAsServerBusy`（`make test`）、`internal/bootstrap` 的 `TestAProjectTooBusyForAWriteAnswersServerBusy`（`make test`） | 单元；组合 |
| `T10.7` | `server_busy` 让客户端两秒之后重试，不是一秒 | `internal/bootstrap` 的 `TestTheWriteGateAnswersItsRefusalsAsServerBusy`（`make test`）、`internal/bootstrap` 的 `TestABusyTransactionIsServerBusy`（`make test`）、`internal/bootstrap` 的 `TestAProjectTooBusyForAWriteAnswersServerBusy`（`make test`） | 单元；组合 |
| `T10.8` | 等锁到期（55P03）结束的事务不是 `server_busy`（500） | `internal/bootstrap` 的 `TestABusyTransactionIsServerBusy`（`make test`）、`internal/bootstrap` 的 `TestAProjectTooBusyForAWriteAnswersServerBusy`（`make test`） | 单元；组合 |
| `T10.9` | app 给 `issue` 普通的事务：写等锁没有期限 | `internal/bootstrap` 的 `TestAProjectTooBusyForAWriteAnswersServerBusy`（`make test`） | 组合 |
| `T10.10` | `issue` 的事务不带 `issue.write_lock_timeout` | `internal/bootstrap` 的 `TestAProjectTooBusyForAWriteAnswersServerBusy`（`make test`） | 组合 |
| `T10.11` | app 的写闸门用随附的默认值，不用配置的 | `internal/bootstrap` 的 `TestAProjectTooBusyForAWriteAnswersServerBusy`（`make test`） | 组合 |
| `T10.12` | 接上的闸门把 `max_queue` 当作每个项目的位置、`per_project` 当作队列 | `internal/bootstrap` 的 `TestTheWriteGateAnswersItsRefusalsAsServerBusy`（`make test`）、`internal/bootstrap` 的 `TestAHotProjectHoldsUpNoReadOfAnother`（`make test`）（Task 12 起） | 单元；组合 |
| `T10.13` | `updateIssue` 的契约不声明 `server_busy` | `internal/modules/issue/adapter/http` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestAProjectTooBusyForAWriteAnswersServerBusy`（`make test`） | 单元；组合 |
| `T10.14` | `createIssue` 的契约不声明 `server_busy` | `internal/modules/issue/adapter/http` 的 全部测试（`make test`）、`internal/bootstrap` 的 `TestAProjectTooBusyForAWriteAnswersServerBusy`（`make test`） | 单元；组合 |
| `T10.15` | 命令行建一个写闸门 | `internal/archtest` 的 `TestCommandsComposeNoServerAndNoJobs`（`make test`） | 组合 |
| `T10.16` | 写用自己的 context 进项目的闸门，不用请求的：请求到期、被取消都不结束它的等待 | `internal/bootstrap` 的 `TestAWriteWhoseRequestEndsWhileItWaitsAtTheGateIsServerBusy`（`make test`） | 组合 |
| `T10.17` | 离开闸门放进事务的函数里（提交、回滚之前就离开） | `internal/modules/issue/app` 的 `TestAWriteEntersItsProjectsGateBetweenItsPrecheckAndItsTransaction`、`TestAWriteLeavesTheGateOnEveryPath`（`make test`） | 单元 |
| `T10.18` | getIssue 进入它的项目的写闸门 | `internal/modules/issue/app` 的 `TestGetIssueReadsTheWorkItem`（`make test`）、`internal/bootstrap` 的 `TestAHotProjectHoldsUpNoReadOfAnother`（`make test`）（Task 12 起） | 单元；组合 |

本 Task 的检查也发现较早的 Task 的变异 `T8.2`、`T8.4`、`T8.6`、`T8.7`、`T9.3`（列在各自的 Task 的表里，标"（Task 10 起）"）：本 Task 做完之后，在本 Task 的树上重跑它们（`mutants_p1b.py`，`MUT_LATER=10`）。

---

### Task 11: 交错 1u、3u–7u：修改的一支

**Files:**
- Create: `server/internal/bootstrap/issue_interleaving_change_test.go`
- Modify: `server/internal/bootstrap/interleaving_test.go`、`server/internal/bootstrap/interleaving_writes_test.go`、`server/internal/bootstrap/issue_interleaving_catalog_test.go`、`server/internal/bootstrap/issue_interleaving_deletion_test.go`、`server/internal/bootstrap/issue_interleaving_test.go`、`server/internal/bootstrap/issue_updates_test.go`、`server/internal/bootstrap/issue_world_test.go`

**Interfaces:** 没有新的生产代码（spec 2.13、第 3 节第 16 条；M4 设计 3.5、9.3）。
- `issue_world_test.go`：`newWorkItemIn(t, project, body)`（alice 经接口在项目里建的工作项），`newWorkItem` 用它；`activityBy(t, item, actor)`（工作项中 actor 的动态，按 `id` 的次序，每行写成 `<field>: <old> → <new>`），1u 和修改的盖戳测试共用这一个读法。
- `issue_updates_test.go`：Task 7 的 `TestUpdatingAWorkItemStampsItsRows` 用 `newWorkItemIn` 建它的工作项、用 `activityBy` 读 bob 的动态。
- `issue_interleaving_test.go`：
  - `changesIssue`（改 Web 唯一的工作项）、`aWorkItem`（alice 的 Fix：三行）、`workItem`；
  - `itemWrite{name, prepare, rows, write, done, gone}`、`itemWrites`（"created"：Web 里的创建；"changed"：改 alice 的 Fix）、`after`、`wroteNothing`；
  - `workItemRows(t, projects, undeleted)`（工作项一侧五张表的行数，Task 13 的规模测试也用）。
- P1a 的交错测试各加修改的一支，换成两支共用的名字：`TestAWorkItemsWriteAndDeletingItsProjectOrWorkspaceSerialize`、`TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused`、`TestDeletingAStateAndPuttingAWorkItemInItSerialize`、`TestDeletingALabelAndLabellingAWorkItemWithItSerialize`（`issue_interleaving_deletion_test.go`、`issue_interleaving_catalog_test.go`）。
- `issue_interleaving_change_test.go`（新）：`TestTwoChangesOfAWorkItemSerialize`，交错 1 的修改一支（1u）；`TestAWorkItemGoneWhileItsChangeWaitsIsNotFound`，等锁期间工作项被删除或移走（spec 第 3 节第 16 条）。
- `interleaving_writes_test.go`：`TestTwoWritesOnAProjectSerialize` 改用 `serializeOn`（P1a 交给 P1b 的一条）。它的第一个写在判定之后、写之前等，所以 `interleaving_test.go` 中 `serializeOn` 的说明只说第一个写持着它的锁在门口等。

**Tests:**（组合；3u–7u 各两种顺序；`-count=5 -race` 没有 40P01）
- 1u：alice 的修改（改名 Mend、加 Bug）持 Web 的行；bob 的修改（改名 Repair、去掉每个标签）按改之前的工作项过了预检、等行；她提交之后他在锁下重读：动态写"Mend → Repair"和去掉她加的 Bug，Bug 不在了，时刻不早于门打开。
- 等锁期间 Fix 被删除、被移到 Ops（P1b 没有这两个操作，由持 Web 的行的事务在数据库里做）：bob 的修改过了预检、等行（探测只认 `projects` 的行锁：事务也持着 Fix 的行，锁了工作项的预检等的是它，之后由预检自己答，不经锁下重读），提交之后在锁下找不到 Web 里的 Fix：404 `issue.not_found`，Fix 的名称不变。
- 3u：bob 先改（加他自己为负责人、他的关注、标签和三行动态，共九行）时删除 Web 或 acme 等行，九行随它在一个时刻由 alice 删除（删除者不是这些行最后的写者），删除的时刻不早于门打开；先删时修改 404 `issue.not_found`，不写。
- 4u、5u：预检放行之后 bob 被移出 Web：404 `issue.not_found`；被降为访客：403；都不写。
- 6u：Review 先删：修改 422 `state_id`，不写；先改到 Review：删除 409 `project.state_in_use`，Review 保留。
- 7u：Bug 先删：修改 422 `label_ids[0]`，不写；先加：Bug 在删除的时刻由 alice 摘下。
- 创建的一支和 P1a 的一样，只是交错 2、3 的创建由 bob 做：六行（他作为创建者和负责人只关注一次），由 alice 删除；`TestTwoWritesOnAProjectSerialize` 照旧；`TestUpdatingAWorkItemStampsItsRows` 照旧（它的工作项由 `newWorkItemIn` 建）。

- [ ] **Step 1: 两支共用的写和世界**

`server/internal/bootstrap/interleaving_writes_test.go`（修改，3 处）：

````old server/internal/bootstrap/interleaving_writes_test.go
	var archiveRec, updateRec *httptest.ResponseRecorder
	archived := run(func() error {
````
````new server/internal/bootstrap/interleaving_writes_test.go
	var archiveRec, updateRec *httptest.ResponseRecorder
	_, archived, updated := serializeOn(t, ctx, r.pool, g, func() error {
````
````old server/internal/bootstrap/interleaving_writes_test.go
	})
	held(t, ctx, g, archived, "the archive")
	updated := run(func() error {
````
````new server/internal/bootstrap/interleaving_writes_test.go
	}, func() error {
````
````old server/internal/bootstrap/interleaving_writes_test.go
	pgtest.WaitForLockWaitOn(t, r.pool, "projects", 5*time.Second)
	close(g.open)

	if err := errors.Join(result(t, ctx, archived, "the archive"), result(t, ctx, updated, "the change")); err != nil {
````
````new server/internal/bootstrap/interleaving_writes_test.go
	if err := errors.Join(archived, updated); err != nil {
````

`server/internal/bootstrap/interleaving_test.go`（修改，1 处）：

````old server/internal/bootstrap/interleaving_test.go
// serializeOn runs first, a write that waits at g past its writes, holding
// its locks, then second, once the first is held there; once the second
// waits for a row lock of projects on pool's database
// (pgtest.WaitForLockWaitOn), it opens g. It answers the moment g opened,
// and each write's error. Every wait has a deadline: ctx's, g's or the
// probe's.
````
````new server/internal/bootstrap/interleaving_test.go
// serializeOn runs first, a write that waits at g holding its locks, then
// second, once the first is held there; once the second waits for a row
// lock of projects on pool's database (pgtest.WaitForLockWaitOn), it opens
// g. It answers the moment g opened, and each write's error. Every wait
// has a deadline: ctx's, g's or the probe's.
````

`server/internal/bootstrap/issue_interleaving_test.go`（修改，6 处）：

````old server/internal/bootstrap/issue_interleaving_test.go
import (
````
````new server/internal/bootstrap/issue_interleaving_test.go
import (
	"cmp"
````
````old server/internal/bootstrap/issue_interleaving_test.go
// The interleavings of a work item's creation (M4 design 3.5, 9.3), on
// issueWorld's real database: each write through its module's route, as
// bootstrap wires it. The first write's transaction is held open past its
// writes (heldTx); the second is sent once it is held, and the gate opens
// once the second waits for Web's row of projects, which each of the two
// takes FOR NO KEY UPDATE (M4 design 3.5, M3 design 3.6). Every wait has a
// deadline.
````
````new server/internal/bootstrap/issue_interleaving_test.go
// The interleavings of a work item's creation and of its change (M4 design
// 3.5, 9.3), on issueWorld's real database: each write through its
// module's route, as bootstrap wires it. The first write's transaction is
// held open past its writes (heldTx); the second is sent once it is held,
// and the gate opens once the second waits for Web's row of projects,
// which each of the two takes FOR NO KEY UPDATE (M4 design 3.5, M3 design
// 3.6). Every wait has a deadline.
````
````old server/internal/bootstrap/issue_interleaving_test.go
}

// named is the body of a work item named name, with no other field.
````
````new server/internal/bootstrap/issue_interleaving_test.go
}

// changesIssue is name's change of Web's one work item (workItem) with the
// body body gives.
func changesIssue(name string, body func(t *testing.T, w issueWorld) string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newIssueRoute(t, w.pool, tx), name, http.MethodPatch, "/api/v0/issues/"+w.workItem(t).String(), body(t, w))
	}
}

// named is the body of a work item named name, with no other field.
````
````old server/internal/bootstrap/issue_interleaving_test.go
	return func(*testing.T, issueWorld) string { return `{"name":"` + name + `"}` }
````
````new server/internal/bootstrap/issue_interleaving_test.go
	return func(*testing.T, issueWorld) string { return `{"name":"` + name + `"}` }
}

// aWorkItem is alice's work item Fix in Web, in Backlog, without assignees
// or labels, through the API: three rows, the work item, alice's
// subscription and its creation's activity.
func aWorkItem(t *testing.T, w issueWorld) {
	w.newWorkItemIn(t, w.web, `{"name":"Fix"}`)
}

// workItem is the id of Web's one work item.
func (w issueWorld) workItem(t *testing.T) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT id FROM issues WHERE project_id = $1", w.web).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// itemWrite is a write on Web's work items that an interleaving runs as a
// creation and as a change alike (M4 design 3.5: P1a's creation branch,
// P1b's change branch): prepare writes what it needs first, nil for
// nothing, leaving rows rows on Web's work-item side (workItemRows);
// write is name's write with the body body gives; done is its answer, and
// gone the code that refuses it once its project is gone to its caller.
type itemWrite struct {
	name    string
	prepare func(t *testing.T, w issueWorld)
	rows    int
	write   func(name string, body func(*testing.T, issueWorld) string) racingWrite
	done    int
	gone    string
}

// itemWrites are a creation in Web and a change of its one work item,
// alice's Fix (aWorkItem).
var itemWrites = []itemWrite{
	{"created", nil, 0, createsIssue, http.StatusCreated, "project.not_found"},
	{"changed", aWorkItem, 3, changesIssue, http.StatusOK, "issue.not_found"},
}

// after is prepare, when given, then iw's own preparation.
func (iw itemWrite) after(prepare func(t *testing.T, w issueWorld)) func(t *testing.T, w issueWorld) {
	return func(t *testing.T, w issueWorld) {
		for _, p := range []func(*testing.T, issueWorld){prepare, iw.prepare} {
			if p != nil {
				p(t, w)
			}
		}
	}
}

// wroteNothing checks that Web's work-item side still has iw's rows: a
// write refused under its lock wrote none.
func (w issueWorld) wroteNothing(t *testing.T, iw itemWrite) {
	t.Helper()
	if n := w.workItemRows(t, []uuid.UUID{w.web}, false); n != iw.rows {
		t.Errorf("Web's work-item side has %d rows after the refused write; want %d, as before it", n, iw.rows)
	}
````
````old server/internal/bootstrap/issue_interleaving_test.go
}

// createdIssue is what the interleavings read of a work item created.
````
````new server/internal/bootstrap/issue_interleaving_test.go
}

// workItemRows counts the rows of the work-item side of projects: their
// work items and the work items' assignees, labels, subscribers and
// activities; the deleted ones too, unless undeleted.
func (w issueWorld) workItemRows(t *testing.T, projects []uuid.UUID, undeleted bool) int {
	t.Helper()
	n := 0
	for _, table := range []string{"issues", "issue_assignees", "issue_labels", "issue_subscribers", "issue_activities"} {
		var rows int
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM "+table+" WHERE project_id = ANY ($1) AND (deleted_at IS NULL OR NOT $2)",
			projects, undeleted).Scan(&rows); err != nil {
			t.Fatal(err)
		}
		n += rows
	}
	return n
}

// createdIssue is what the interleavings read of a work item created.
````
````old server/internal/bootstrap/issue_interleaving_test.go
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
````
````new server/internal/bootstrap/issue_interleaving_test.go
// interleavings 4 and 5, the creation's branch and the change's): bob, a
// member of Web, made secret, creates a work item, or changes alice's
// Fix, while alice's removal of him, or her demotion of him to a guest,
// holds Web past its writes. His write passed the precheck, which read him
// a member, and waits for Web's row; once hers commits, the decision under
// the lock reads him removed, who no longer sees Web: 404
// project.not_found, or issue.not_found for the change; or a guest, who
// writes no work item: 403 forbidden. His write wrote nothing.
func TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused(t *testing.T) {
	for _, iw := range itemWrites {
		for _, tt := range []struct {
			name    string
			change  racingWrite
			changed int
			want    int
			code    string // "": iw.gone
		}{
			{"removed", removesBob, http.StatusNoContent, http.StatusNotFound, ""},
			{"made a guest", demotesBob, http.StatusOK, http.StatusForbidden, "forbidden"},
		} {
			t.Run(iw.name+", "+tt.name, func(t *testing.T) {
				w := newIssueWorld(t)
				w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"network":0}`, http.StatusOK)
				iw.after(nil)(t, w)
				var change, written exchange
				w.serialize(t, tt.change, iw.write("bob", named("Mend")), &change, &written)
				w.answered(t, change, tt.changed)
				w.refused(t, written, tt.want, cmp.Or(tt.code, iw.gone))
				w.wroteNothing(t, iw)
			})
		}
````

`server/internal/bootstrap/issue_world_test.go`（修改，3 处）：

````old server/internal/bootstrap/issue_world_test.go
func (w issueWorld) newWorkItem(t *testing.T) uuid.UUID {
	t.Helper()
````
````new server/internal/bootstrap/issue_world_test.go
func (w issueWorld) newWorkItem(t *testing.T) uuid.UUID {
	t.Helper()
	return w.newWorkItemIn(t, w.web, w.fullIssue())
}

// newWorkItemIn is the id of a work item alice creates in project through
// the API, with body.
func (w issueWorld) newWorkItemIn(t *testing.T, project uuid.UUID, body string) uuid.UUID {
	t.Helper()
````
````old server/internal/bootstrap/issue_world_test.go
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", w.fullIssue(), http.StatusCreated),
		&created)
````
````new server/internal/bootstrap/issue_world_test.go
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+project.String()+"/issues", body, http.StatusCreated), &created)
````
````old server/internal/bootstrap/issue_world_test.go
	return slices.Sorted(slices.Values(lines))
}
````
````new server/internal/bootstrap/issue_world_test.go
	return slices.Sorted(slices.Values(lines))
}

// activityBy is each row of the work item item's activity in which the
// account actor acts, in the order of their ids, as "<field>: <old value>
// → <new value>"; a row without either value fails the test.
func (w issueWorld) activityBy(t *testing.T, item, actor uuid.UUID) []string {
	t.Helper()
	rows, err := w.pool.Query(pgtest.Soon(t), `SELECT a.field || ': ' || a.old_value || ' → ' || a.new_value FROM issue_activities a
		WHERE a.issue_id = $1 AND a.actor_id = $2 ORDER BY a.id`, item, actor)
	if err != nil {
		t.Fatal(err)
	}
	said, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	return said
}
````

`server/internal/bootstrap/issue_updates_test.go`（修改，5 处）：

````old server/internal/bootstrap/issue_updates_test.go
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
````
````new server/internal/bootstrap/issue_updates_test.go
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
````
````old server/internal/bootstrap/issue_updates_test.go
	var created struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", `{"name":"Fix","state_id":"`+
		w.todo.String()+`","assignee_ids":["`+alice.String()+`"],"label_ids":["`+w.bug.String()+`"]}`, http.StatusCreated), &created)
	before := time.Now().Truncate(time.Microsecond)
	answer := w.send(t, "bob", http.MethodPatch, "/api/v0/issues/"+created.ID.String(), `{"name":"Mend","priority":"urgent","state_id":"`+
````
````new server/internal/bootstrap/issue_updates_test.go
	item := w.newWorkItemIn(t, w.web, `{"name":"Fix","state_id":"`+w.todo.String()+`","assignee_ids":["`+alice.String()+`"],"label_ids":["`+
		w.bug.String()+`"]}`)
	before := time.Now().Truncate(time.Microsecond)
	answer := w.send(t, "bob", http.MethodPatch, "/api/v0/issues/"+item.String(), `{"name":"Mend","priority":"urgent","state_id":"`+
````
````old server/internal/bootstrap/issue_updates_test.go
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT created_at, updated_at, completed_at FROM issues WHERE id = $1", created.ID).Scan(&made,
````
````new server/internal/bootstrap/issue_updates_test.go
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT created_at, updated_at, completed_at FROM issues WHERE id = $1", item).Scan(&made,
````
````old server/internal/bootstrap/issue_updates_test.go
	if got := w.workItemStamps(t, created.ID); !slices.Equal(got, want) {
````
````new server/internal/bootstrap/issue_updates_test.go
	if got := w.workItemStamps(t, item); !slices.Equal(got, want) {
````
````old server/internal/bootstrap/issue_updates_test.go
	activity, err := w.pool.Query(pgtest.Soon(t), `SELECT a.field || ': ' || a.old_value || ' → ' || a.new_value FROM issue_activities a
		WHERE a.issue_id = $1 AND a.actor_id = $2 ORDER BY a.id`, created.ID, bob)
	if err != nil {
		t.Fatal(err)
	}
	said, err := pgx.CollectRows(activity, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
````
````new server/internal/bootstrap/issue_updates_test.go
	said := w.activityBy(t, item, bob)
````

- [ ] **Step 2: 两个修改，删除和目录的交错**

`server/internal/bootstrap/issue_interleaving_catalog_test.go`（整个文件，86 行）：

````whole server/internal/bootstrap/issue_interleaving_catalog_test.go
package bootstrap

import (
	"net/http"
	"slices"
	"testing"
	"uuid"
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

// Deleting a state and putting a work item in it serialize on Web's row (M4
// design 3.6, 3.7; interleaving 6, the creation's branch and the change's),
// in both orders: Review deleted first, the creation, or the change of
// alice's Fix, finds no such state under its lock: 422 state_id, and it
// wrote nothing; written first, the deletion finds the work item in Review
// under its lock: 409 project.state_in_use, and Review is kept.
func TestDeletingAStateAndPuttingAWorkItemInItSerialize(t *testing.T) {
	for _, iw := range itemWrites {
		t.Run(iw.name, func(t *testing.T) {
			inBothOrders(t, iw.after(addsReview), "deleted", deletesReview, iw.name, iw.write("bob", inReview), func(t *testing.T,
				w issueWorld, deletedFirst bool, deletion, written exchange) {
				if deletedFirst {
					w.answered(t, deletion, http.StatusNoContent)
					w.refused(t, written, http.StatusUnprocessableEntity, "validation_failed state_id not_allowed")
					w.wroteNothing(t, iw)
					return
				}
				var put struct {
					StateID uuid.UUID `json:"state_id"`
				}
				decodeAnswer(t, w.answered(t, written, iw.done), &put)
				w.refused(t, deletion, http.StatusConflict, "project.state_in_use")
				w.stateKept(t, put.StateID, "Review, the work item's state")
			})
		})
	}
}

// Deleting a label and labelling a work item with it serialize on Web's row
// (M4 design 3.6, 3.7; interleaving 7, the creation's branch and the
// change's), in both orders: Bug deleted first, the creation, or the change
// of alice's Fix, finds no such label under its lock: 422 label_ids[0], and
// it wrote nothing; written first, the deletion takes Bug off the work item
// at its moment, by alice.
func TestDeletingALabelAndLabellingAWorkItemWithItSerialize(t *testing.T) {
	for _, iw := range itemWrites {
		t.Run(iw.name, func(t *testing.T) {
			inBothOrders(t, iw.prepare, "deleted", deletesBug, iw.name, iw.write("bob", labelledBug), func(t *testing.T, w issueWorld,
				deletedFirst bool, deletion, written exchange) {
				w.answered(t, deletion, http.StatusNoContent)
				if deletedFirst {
					w.refused(t, written, http.StatusUnprocessableEntity, "validation_failed label_ids[0] not_allowed")
					w.wroteNothing(t, iw)
					return
				}
				w.answered(t, written, iw.done)
				if got, want := w.workItemLabels(t, w.bug, "alice"), []string{"Bug taken off by alice with Bug"}; !slices.Equal(got, want) {
					t.Errorf("the work item's labels after Bug's deletion: %q; want %q", got, want)
				}
			})
		})
	}
}
````

`server/internal/bootstrap/issue_interleaving_change_test.go`（新文件，94 行）：

````file server/internal/bootstrap/issue_interleaving_change_test.go
package bootstrap

import (
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// Two changes of a work item serialize on its project's row (M4 design
// 3.5, 3.7; interleaving 1, the change's branch): alice's renames her Fix
// Mend and labels it Bug, and holds Web FOR NO KEY UPDATE and its rows
// past its writes; bob's, sent then, which renames it Repair and takes
// every label off, passes his precheck on the work item as it was and
// waits for Web's row. Once hers has committed he reads the work item
// again under his lock and changes it from what she left: his activity
// renames Mend, not Fix, and takes off the Bug she put on, which is gone,
// at a moment no earlier than the gate's opening, which he read under his
// lock.
func TestTwoChangesOfAWorkItemSerialize(t *testing.T) {
	w := newIssueWorld(t)
	aWorkItem(t, w)
	var hers, his exchange
	w.serialize(t, changesIssue("alice", func(_ *testing.T, w issueWorld) string {
		return `{"name":"Mend","label_ids":["` + w.bug.String() + `"]}`
	}), changesIssue("bob", func(*testing.T, issueWorld) string { return `{"name":"Repair","label_ids":[]}` }), &hers, &his)
	w.answered(t, hers, http.StatusOK)
	var changed struct {
		Name      string      `json:"name"`
		LabelIDs  []uuid.UUID `json:"label_ids"`
		UpdatedAt time.Time   `json:"updated_at"`
	}
	decodeAnswer(t, w.answered(t, his, http.StatusOK), &changed)
	if changed.Name != "Repair" || len(changed.LabelIDs) != 0 || changed.UpdatedAt.Before(hers.released) {
		t.Errorf("bob's change answers %+v; want Repair, no label, changed no earlier than %v", changed, hers.released)
	}
	said := w.activityBy(t, w.workItem(t), w.ids["bob"])
	if want := []string{"name: Mend → Repair", "labels: Bug → "}; !slices.Equal(said, want) {
		t.Errorf("bob's activity says %q, want %q", said, want)
	}
}

// A work item deleted, or moved into another project, while a change of it
// waits for its project's row is not found under the lock (M4 design 3.5:
// the work item read again under the lock, and found still in the project
// locked). No operation of P1b deletes or moves a work item, so a
// transaction does it in the database: it holds Web's row FOR NO KEY
// UPDATE and deletes alice's Fix, or moves it into Ops; bob's change,
// which passed his precheck on Fix as it was, waits for Web's row, and
// once the transaction commits finds no Fix in Web: 404 issue.not_found,
// and Fix keeps its name. The test probes for his wait on a row of
// projects: the transaction holds Fix's row too, and a wait for it, by a
// precheck that locked the work item, would be answered by the precheck,
// not under the lock.
func TestAWorkItemGoneWhileItsChangeWaitsIsNotFound(t *testing.T) {
	for _, moved := range []bool{false, true} {
		t.Run(map[bool]string{false: "deleted", true: "moved"}[moved], func(t *testing.T) {
			w := newIssueWorld(t)
			aWorkItem(t, w)
			item := w.workItem(t)
			gone, args := "UPDATE issues SET deleted_at = now() WHERE id = $1", []any{item}
			if moved {
				ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
				gone, args = "UPDATE issues SET project_id = $2 WHERE id = $1", []any{item, ops}
			}
			held := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR NO KEY UPDATE", w.web)
			if _, err := held.Exec(pgtest.Soon(t), gone, args...); err != nil {
				t.Fatal(err)
			}
			change := newRequest(t, http.MethodPatch, w.base+"/api/v0/issues/"+item.String(), w.tokens["bob"], []byte(`{"name":"Mend"}`))
			w.contract.CheckRequest(t, change)
			changed := sendInBackground(change)
			pgtest.WaitForLockWaitOn(t, w.pool, "projects", 5*time.Second)
			if err := held.Commit(pgtest.Soon(t)); err != nil {
				t.Fatal(err)
			}
			a := receiveWithin(t, changed, 5*time.Second, "answer to bob's change")
			if a.err != nil {
				t.Fatal(a.err)
			}
			w.contract.CheckResponse(t, change, a.res)
			if a.res.StatusCode != http.StatusNotFound || problemCode(t, a.body) != "issue.not_found" {
				t.Errorf("bob's change = %d %s; want 404 issue.not_found", a.res.StatusCode, a.body)
			}
			var name string
			if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT name FROM issues WHERE id = $1", item).Scan(&name); err != nil || name != "Fix" {
				t.Errorf("the work item is named %q (%v) after the refused change; want Fix", name, err)
			}
		})
	}
}
````

`server/internal/bootstrap/issue_interleaving_deletion_test.go`（修改，1 处）：

````old server/internal/bootstrap/issue_interleaving_deletion_test.go
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
````
````new server/internal/bootstrap/issue_interleaving_deletion_test.go
// A work item's write and its project's deletion, or its workspace's,
// serialize on Web's row (M4 design 3.5, 3.6; interleavings 2 and 3, the
// creation's branch and the change's), in both orders. bob writes and alice
// deletes: the rows he wrote name him until the deletion stamps them as
// hers. Written first, holding Web FOR NO KEY UPDATE: a work item bob
// creates assigned to himself and labelled Bug (fullIssue), six rows, his
// one subscription, as its creator and its assignee, among them; or alice's
// Fix (aWorkItem) he changes so, which adds his assignment, his
// subscription, a label and three activities to its three rows. The
// deletion waits for Web's row, then deletes every row of Web's work-item
// side with Web, or with acme, by alice, at its one moment, no earlier than
// the write's commit: it read its clock under the locks it took once the
// write had committed (M4 design 3.5). Deleted first: the write, which
// passed the precheck, finds Web deleted under its lock: 404
// project.not_found, or issue.not_found for the change, and it wrote
// nothing; the rows there were are deleted.
func TestAWorkItemsWriteAndDeletingItsProjectOrWorkspaceSerialize(t *testing.T) {
	for _, iw := range []struct {
		itemWrite
		written int // the rows of Web's work-item side once it has written
	}{{itemWrites[0], 6}, {itemWrites[1], 9}} {
		for _, deletion := range []struct {
			name  string
			write racingWrite
			table string                       // the deleted row's
			id    func(w issueWorld) uuid.UUID // its id
		}{
			{"Web", deletesWeb, "projects", func(w issueWorld) uuid.UUID { return w.web }},
			{"acme", deletesAcme, "workspaces", func(w issueWorld) uuid.UUID { return w.acme }},
		} {
			t.Run(iw.name+", "+deletion.name, func(t *testing.T) {
				inBothOrders(t, iw.prepare, "deleted", deletion.write, iw.name, iw.write("bob", fullBody), func(t *testing.T,
					w issueWorld, deletedFirst bool, deleted, written exchange) {
					w.answered(t, deleted, http.StatusNoContent)
					rows := iw.written
					if deletedFirst {
						w.refused(t, written, http.StatusNotFound, iw.gone)
						rows = iw.rows
					} else {
						w.answered(t, written, iw.done)
					}
					w.deletedWith(t, deletion.table, deletion.id(w), written.released, rows)
				})
			})
		}
````

- [ ] **Step 3: 测试和 lint**

Run: `go -C server test -count=1 -run 'TestTwoChangesOfAWorkItemSerialize$|TestAWorkItemGoneWhileItsChangeWaitsIsNotFound$|TestAWorkItemsWriteAndDeletingItsProjectOrWorkspaceSerialize$|TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused$|TestDeletingAStateAndPuttingAWorkItemInItSerialize$|TestDeletingALabelAndLabellingAWorkItemWithItSerialize$|TestTwoWritesOnAProjectSerialize$|TestUpdatingAWorkItemStampsItsRows$' ./internal/bootstrap/`
Expected: `ok`。

Run: `go -C server test -count=5 -race -run 'TestTwoCreationsOnAProjectSerialize$|TestTwoChangesOfAWorkItemSerialize$|TestAWorkItemGoneWhileItsChangeWaitsIsNotFound$|TestArchivingAProjectAndCreatingAWorkItemSerialize$|TestRemovingTheDefaultAssigneeAndCreatingAWorkItemSerialize$|TestAWorkItemsWriteAndDeletingItsProjectOrWorkspaceSerialize$|TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused$|TestDeletingAStateAndPuttingAWorkItemInItSerialize$|TestDeletingALabelAndLabellingAWorkItemWithItSerialize$|TestTwoWritesOnAProjectSerialize$' ./internal/bootstrap/`
Expected: `ok`；输出里没有 `40P01`、`deadlock detected`、`DATA RACE`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/bootstrap/interleaving_test.go server/internal/bootstrap/interleaving_writes_test.go server/internal/bootstrap/issue_interleaving_catalog_test.go server/internal/bootstrap/issue_interleaving_change_test.go server/internal/bootstrap/issue_interleaving_deletion_test.go server/internal/bootstrap/issue_interleaving_test.go server/internal/bootstrap/issue_updates_test.go server/internal/bootstrap/issue_world_test.go
```
```bash
git commit -m "test(M4/P1): interleavings 1 and 3 to 7 with a work item's change beside its creation

Two changes of one work item serialize on its project's row: the second
reads the work item again under its lock and changes it from what the
first left, at a moment after it; one whose work item was deleted or
moved meanwhile finds it no more. Each of P1a's interleavings runs a
change as well as a creation, through one itemWrite: a deletion of its
project or workspace, a removal or a demotion of its caller, a deletion
of the state or label it puts the work item in, each in both orders. A
change refused under its lock writes nothing; one written first is
deleted, or refuses the state's deletion, or has its label taken off.
TestTwoWritesOnAProjectSerialize uses serializeOn.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**：本 Task 没有自己的变异。

本 Task 的检查也发现较早的 Task 的变异 `T5.3`、`T5.6`、`T5.9`、`T5.15`、`T5.16`、`T5.17`、`T5.18`、`T5.20`（列在各自的 Task 的表里，标"（Task 11 起）"）：本 Task 做完之后，在本 Task 的树上重跑它们（`mutants_p1b.py`，`MUT_LATER=11`）。

---

### Task 12: 交错 20：饥饿与连接池

**Files:**
- Create: `server/internal/bootstrap/issue_pool_test.go`

**Interfaces:** 没有新的生产代码（spec 2.14、第 3 节第 13 条；M4 设计 3.5、9.3、11.11，决策点 1 的 B）。`issue_pool_test.go`（新）：`newPoolWorld`（池 10 个连接，`database.max_conns` 的默认值；闸门是 `testIssueConfig` 的默认值，`configure` 再改它）、`reply`、`replyTo`（不碰测试，同时发送的 goroutine 调用它）、`poolReadBound`（500 毫秒）。

**Tests:**（组合，`-count=5 -race` 没有 40P01）
- `TestChainedWorkItemWritesHoldOffNoWorkspaceWrite`（S5 的回归）：acme 的四个项目各一个工作项、20 个标签，16 串首尾相连的修改（每次换上或去掉 20 个标签，约四十行）；每串都有一次修改得到回答之后（5 秒为限），alice 改 acme 的名称、把 walt 改为管理员、删除 acme，各在 1 秒之内；每串的修改都答 200，直到 acme 删除之后答 404 `issue.not_found`。
- `TestAHotProjectHoldsUpNoReadOfAnother`（SP1 的回归）：这个测试的 `max_wait` 是一分钟（它断言闸门的位置和队列的容量，不断言等多久）。别的事务持 Web 的行至少一秒、直到 134 个 503 都答了（3 秒为限），期间 200 个修改同时打到 Web：闸门放两个进数据库（等行），64 个在进程里等，其余 134 个立即 503 `server_busy`（`Retry-After: 1`）；Ops 的工作项和 Web 的这个工作项每 20 毫秒各读一次，至少 20 次，每次 200、不超过 `poolReadBound`（500 毫秒，第 3 节第 13 条；读不进闸门）；行放开之后等着的 66 个都答 200。

- [ ] **Step 1: 交错 20**

`server/internal/bootstrap/issue_pool_test.go`（新文件，266 行）：

````file server/internal/bootstrap/issue_pool_test.go
package bootstrap

import (
	"context"
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// Interleaving 20 of M4 design 3.5 and 9.3, starvation and the connection
// pool (the regressions of the design's S5 and SP1): through the app as
// bootstrap wires it, on a pool of database.max_conns' default, 10
// connections, with the write gate of testIssueConfig, the defaults: two
// writes on a project in the database at once, 64 more waiting in the
// process, without a connection, for 2 seconds at most (a minute in the
// burst's test, which counts the gate's places and its queue's room, not
// the wait).

// newPoolWorld is the world in an app whose pool has database.max_conns'
// default, 10 connections, its configuration then changed by configure.
func newPoolWorld(t *testing.T, configure ...func(cfg *config.Config)) issueWorld {
	t.Helper()
	return newIssueWorldWith(t, func(cfg *config.Config) {
		cfg.Database.MaxConns = 10
		for _, c := range configure {
			c(cfg)
		}
	})
}

// reply is an answer to a request sent from a goroutine of its own: its
// status, its problem's code, its Retry-After, how long it took and when it
// came, or the error that ended it.
type reply struct {
	status int
	code   string
	retry  string
	took   time.Duration
	at     time.Time
	err    error
}

// String is the reply as the failures name it: its status, code and
// Retry-After, those it has.
func (r reply) String() string {
	if r.err != nil {
		return r.err.Error()
	}
	return strings.TrimSpace(strconv.Itoa(r.status) + " " + r.code + " " + r.retry)
}

// replyTo sends method path with body as alice. It touches no test: the
// goroutines that send at once call it.
func (w issueWorld) replyTo(method, path, body string) reply {
	req, err := http.NewRequest(method, w.base+path, strings.NewReader(body))
	if err != nil {
		return reply{err: err}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+w.tokens["alice"])
	start := time.Now()
	res, err := client.Do(req)
	if err != nil {
		return reply{took: time.Since(start), err: err}
	}
	defer func() { _ = res.Body.Close() }()
	got, err := io.ReadAll(res.Body)
	r := reply{status: res.StatusCode, retry: res.Header.Get("Retry-After"), took: time.Since(start), at: time.Now(), err: err}
	var problem struct{ Code string }
	if r.status >= 400 && json.Unmarshal(got, &problem) == nil {
		r.code = problem.Code
	}
	return r
}

// Chained writes on the work items of a workspace's projects hold off none
// of its workspace-level writes (M4 design 3.5, decision 1's B; S5's
// regression). Sixteen clients change the work items of four projects of
// acme back to back, each sending its next change once its last has
// answered; each change puts twenty labels on its work item or takes them
// off, some forty rows, so that the writes in the database overlap at
// every moment. Once each chain has had a change answered, alice renames
// acme and makes walt an admin of
// it, each within a second: neither waits for the chains, whose writes
// lock their projects' rows alone; under a lock of acme's row that each
// write shared, the chains' overlapping locks would hold either off to the
// request's deadline. Then she deletes acme, within a second too: it waits
// on each project only for the writes in the database, at most two (the
// gate's). Every change answered 200 until acme was deleted, and each
// chain's next 404 issue.not_found.
func TestChainedWorkItemWritesHoldOffNoWorkspaceWrite(t *testing.T) {
	w := newPoolWorld(t)
	projects := []uuid.UUID{w.web}
	for _, name := range []string{"Ops", "Api", "Docs"} {
		projects = append(projects, createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", name, strings.ToUpper(name)))
	}
	items := make([]uuid.UUID, len(projects))
	labelled := make([]string, len(projects))
	for i, p := range projects {
		items[i] = w.newWorkItemIn(t, p, `{"name":"Fix"}`)
		var ids []string
		for n := range 20 {
			var label struct {
				ID uuid.UUID `json:"id"`
			}
			decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+p.String()+"/labels", `{"name":"L`+strconv.Itoa(n)+`"}`,
				http.StatusCreated), &label)
			ids = append(ids, `"`+label.ID.String()+`"`)
		}
		labelled[i] = `{"label_ids":[` + strings.Join(ids, ",") + `]}`
	}
	walt := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id = $1 AND member_id = $2", w.acme, w.ids["walt"])[0]

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ends := make([]reply, 16)
	changes := make([]atomic.Int32, 16)
	var chains sync.WaitGroup
	for i := range ends {
		chains.Go(func() {
			for n := 0; ctx.Err() == nil; n++ {
				body := []string{labelled[i%len(items)], `{"label_ids":[]}`}[n%2]
				r := w.replyTo(http.MethodPatch, "/api/v0/issues/"+items[i%len(items)].String(), body)
				if r.status != http.StatusOK {
					ends[i] = r
					return
				}
				changes[i].Add(1)
			}
		})
	}
	underway := func() bool {
		for i := range changes {
			if changes[i].Load() == 0 {
				return false
			}
		}
		return true
	}
	for begun := time.Now(); !underway(); time.Sleep(10 * time.Millisecond) {
		if time.Since(begun) > 5*time.Second {
			cancel()
			chains.Wait()
			t.Fatal("not every chain had a change answered within 5s")
		}
	}
	for _, write := range []struct{ method, path, body string }{
		{http.MethodPatch, "/api/v0/workspaces/acme", `{"name":"Acme Two"}`},
		{http.MethodPatch, "/api/v0/workspace-members/" + walt.String(), `{"role":20}`},
		{http.MethodDelete, "/api/v0/workspaces/acme", ""},
	} {
		r := w.replyTo(write.method, write.path, write.body)
		t.Logf("%s %s: %v after %v", write.method, write.path, r, r.took)
		if r.err != nil || r.status >= 300 || r.took > time.Second {
			t.Errorf("%s %s = %v after %v; want it done within 1s", write.method, write.path, r, r.took)
		}
	}
	chains.Wait()
	for i, end := range ends {
		if end.status != http.StatusNotFound || end.code != "issue.not_found" || changes[i].Load() == 0 {
			t.Errorf("chain %d: %d changes, then %v; want changes, then 404 issue.not_found once acme was deleted", i, changes[i].Load(), end)
		}
	}
}

// A hot project holds up no read, of another project or of its own (M4
// design 3.5, 11.11; SP1's regression). A transaction holds Web's row while
// 200 changes of a work item of Web, a credential's burst, arrive at once:
// the write gate lets two into the database, which wait for the row, each
// holding a connection of the pool; 64 more wait in the process, without
// one, as long as the row is held; the other 134 are refused at once, the
// gate's queue full: 503 server_busy, Retry-After: 1. The row is held for a
// second at least, and until those 134 have answered. Meanwhile a read of a
// work item of Ops and one of the hot work item, every 20 ms, each take at
// most poolReadBound: without the gate the changes would hold all ten
// connections while they wait for Web's row, and the reads would queue
// behind them in the pool until it is free; a read takes no place in the
// gate. Once the row is free, the 66 changes that waited go through.
func TestAHotProjectHoldsUpNoReadOfAnother(t *testing.T) {
	w := newPoolWorld(t, func(cfg *config.Config) { cfg.Issue.WriteGate.MaxWait = time.Minute })
	hot := w.newWorkItem(t)
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	other := w.newWorkItemIn(t, ops, `{"name":"Fix"}`)

	held := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR UPDATE", w.web)
	writes := make([]reply, 200)
	var answered atomic.Int32
	var burst sync.WaitGroup
	start := make(chan struct{})
	for i := range writes {
		burst.Go(func() {
			<-start
			writes[i] = w.replyTo(http.MethodPatch, "/api/v0/issues/"+hot.String(), `{"name":"Fix `+strconv.Itoa(i)+`"}`)
			answered.Add(1)
		})
	}
	close(start)
	pgtest.WaitForLockWaitBehind(t, w.pool, held.Conn().PgConn(), 5*time.Second)
	waited := time.Now()
	var reads []reply
	probing, stop := context.WithCancel(context.Background())
	var probe sync.WaitGroup
	probe.Go(func() {
		for probing.Err() == nil {
			for _, item := range []uuid.UUID{other, hot} {
				reads = append(reads, w.replyTo(http.MethodGet, "/api/v0/issues/"+item.String(), ""))
			}
			time.Sleep(20 * time.Millisecond)
		}
	})
	for time.Since(waited) < time.Second || answered.Load() < 134 {
		if time.Since(waited) > 3*time.Second {
			t.Errorf("%d changes answered within 3s of the first wait for Web's row; want the 134 beyond the gate's room", answered.Load())
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	stop()
	released := time.Now()
	if err := held.Rollback(pgtest.Soon(t)); err != nil {
		t.Fatal(err)
	}
	probe.Wait()
	burst.Wait()

	slowest := slices.MaxFunc(reads, func(a, b reply) int { return int(a.took - b.took) })
	t.Logf("%d reads while Web's row was held, the slowest after %v", len(reads), slowest.took)
	if len(reads) < 20 || slowest.took > poolReadBound {
		t.Errorf("%d reads while Web's row was held, the slowest after %v; want 20 or more, each within %v", len(reads), slowest.took,
			poolReadBound)
	}
	for _, r := range reads {
		if r.status != http.StatusOK {
			t.Errorf("a read of a work item answered %v, want 200", r)
		}
	}
	before, after := map[string]int{}, map[string]int{}
	for _, r := range writes {
		if r.at.Before(released) {
			before[r.String()]++
		} else {
			after[r.String()]++
		}
	}
	if !maps.Equal(before, map[string]int{"503 server_busy 1": 134}) || !maps.Equal(after, map[string]int{"200": 66}) {
		t.Errorf("the 200 changes answered %v before Web's row was free, %v after; want 134 503 server_busy with Retry-After 1 before, "+
			"and 66 200 after", before, after)
	}
}

// poolReadBound is how long a read of another project may take while a
// project takes a burst of writes (M4/P1b spec 3: measured, with its
// margin).
const poolReadBound = 500 * time.Millisecond
````

- [ ] **Step 2: 测试和 lint**

Run: `go -C server test -count=1 -v -run 'TestChainedWorkItemWritesHoldOffNoWorkspaceWrite$|TestAHotProjectHoldsUpNoReadOfAnother$' ./internal/bootstrap/`
Expected: `ok`；日志写下读的次数和最慢的一次（原型上 2–49 毫秒，第 3 节第 13 条）。

Run: `go -C server test -count=5 -race -run 'TestChainedWorkItemWritesHoldOffNoWorkspaceWrite$|TestAHotProjectHoldsUpNoReadOfAnother$' ./internal/bootstrap/`
Expected: `ok`；输出里没有 `40P01`、`deadlock detected`、`DATA RACE`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 3: 提交**

```bash
git add server/internal/bootstrap/issue_pool_test.go
```
```bash
git commit -m "test(M4/P1): interleaving 20, starvation and the connection pool

Sixteen chains of changes on four projects of a workspace hold off
neither its rename, nor a member's promotion, nor its deletion, each
within a second, on a pool of ten connections. A burst of 200 changes on
one project whose row another transaction holds lets two into the
database and 64 wait in the process; the rest answer 503 server_busy at
once, and reads of another project's work item and of the hot one keep
answering within 500 ms.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p1b.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；静态是 `make lint-web` 的文案和键的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T12.1` | 工作项的写在项目行之外另取工作区行的 `FOR SHARE`（M3 的方案 E，决策点 1 不取） | `internal/bootstrap` 的 `TestChainedWorkItemWritesHoldOffNoWorkspaceWrite`（`make test`）、`internal/bootstrap` 的 `TestEachIssueWriteLocksItsProjectFirst`（`make test`） | 组合 |

本 Task 的检查也发现较早的 Task 的变异 `T8.10`、`T10.3`、`T10.12`、`T10.18`（列在各自的 Task 的表里，标"（Task 12 起）"）：本 Task 做完之后，在本 Task 的树上重跑它们（`mutants_p1b.py`，`MUT_LATER=12`）。

---

### Task 13: 规模目标上的删除的测量

**Files:**
- Create: `server/internal/bootstrap/issue_scale_test.go`

**Interfaces:** 没有新的生产代码（spec 2.15、第 3 节第 14 条；M4 设计决策点 7、3.16、9.3）。`issue_scale_test.go`（新）：`TestTheDeletionsAtTheScaleTarget`，只在设了 `NERVE_MEASURE_SCALE` 时运行（`make test` 和持续集成跳过它）；`seedAtScale`（一个事务，每张表的行按随机的次序写下，之后 `VACUUM ANALYZE`）、`rolledBack`（回滚的事务里量，之后 `VACUUM`）；行数用 Task 11 的 `workItemRows`。

**Tests:**（测量，不断言时间）
- Web 和另一个工作区 beta 的五个项目，每个项目 2 万个工作项，每个工作项一个负责人、一个标签、两个关注、20 行动态：每个项目 50 万行，共 300 万行。
- 先在回滚的事务里量工作项一侧的软删除（`DeleteProjectIssues`、`DeleteWorkspaceIssues`），再经接口删除 Web、删除 beta（请求的期限放长到一小时），量请求；最后核对六个项目没有未删除的行。
- 原型上的数字在 spec 附录 A.4：Web 一侧约 5–6 秒、请求约 4.5 秒；beta 一侧约 25–30 秒、请求约 23–25 秒，超过生产的 15 秒请求期限（spec 第 7 节：P9 换成任务）。

- [ ] **Step 1: 测量**

`server/internal/bootstrap/issue_scale_test.go`（新文件，174 行）：

````file server/internal/bootstrap/issue_scale_test.go
package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// The deletions of a project and of a workspace at the work-item side's
// scale target (M4 design decision point 7, 3.16: a project of some 20,000
// work items), measured, not asserted, and only with NERVE_MEASURE_SCALE
// set: `make test` and CI skip it. P1b measured P1a's synchronous cascade
// with it (M4/P1b spec 3); M4 design decision point 7 has P9 measure its
// jobs on the same data. Run it alone, with a long timeout:
//
//	NERVE_MEASURE_SCALE=1 go test ./internal/bootstrap -run TestTheDeletionsAtTheScaleTarget -v -timeout 60m
//
// The data: Web, acme's project, and five projects of a workspace beta,
// each with 20,000 work items in Backlog, each assigned to alice, labelled
// with its project's Bug, with alice and bob subscribed and 20 activities:
// 500,000 rows of the work-item side a project, 3,000,000 in all, the ids
// of each table interleaved across the six projects, as concurrent writes
// leave them. Each deletion is measured twice: its work-item side's
// soft-deletion alone (the issue module's cascade, issue.Provide), in a
// transaction rolled back, then vacuumed away; and its request through the
// API, which commits, in an app whose request deadline is long enough for
// it to finish.
func TestTheDeletionsAtTheScaleTarget(t *testing.T) {
	if os.Getenv("NERVE_MEASURE_SCALE") == "" {
		t.Skip("NERVE_MEASURE_SCALE is not set: the measurement at the scale target runs only when asked for")
	}
	w := newIssueWorldWith(t, func(cfg *config.Config) {
		cfg.Server.RequestTimeout, cfg.Server.WriteTimeout, cfg.Database.CommitTimeout = time.Hour, time.Hour, time.Hour
	})
	beta := w.newWorkspace(t, "Beta", "beta")
	projects := []uuid.UUID{w.web}
	for _, name := range []string{"One", "Two", "Three", "Four", "Five"} {
		project := createdProject(t, w.contract, w.base, w.tokens["alice"], "beta", name, strings.ToUpper(name))
		w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+project.String()+"/labels", `{"name":"Bug"}`, http.StatusCreated)
		projects = append(projects, project)
	}
	start := time.Now()
	w.seedAtScale(t, projects)
	t.Logf("seeded six projects, %d rows of the work-item side, in %v", w.workItemRows(t, projects, true), time.Since(start))

	cascade := issue.Provide(w.pool).IssueCascade
	t.Logf("deleteProject of Web: its work-item side's rows %v, rolled back", w.rolledBack(t, func(ctx context.Context) error {
		return cascade.DeleteProjectIssues(ctx, w.web, w.ids["alice"], time.Now())
	}))
	t.Logf("deleteWorkspace of beta: its five projects' work-item side's rows %v, rolled back", w.rolledBack(t,
		func(ctx context.Context) error {
			return cascade.DeleteWorkspaceIssues(ctx, projects[1:], w.ids["alice"], time.Now())
		}))
	t.Logf("deleteProject of Web: the request %v", w.deletion(t, "/api/v0/projects/"+w.web.String()))
	t.Logf("deleteWorkspace of beta: the request %v", w.deletion(t, "/api/v0/workspaces/"+beta))
	if n := w.workItemRows(t, projects, true); n != 0 {
		t.Errorf("%d rows of the six projects' work-item side left undeleted, want none", n)
	}
}

// newWorkspace is the slug of a workspace alice creates through the API.
func (w issueWorld) newWorkspace(t *testing.T, name, slug string) string {
	t.Helper()
	w.send(t, "alice", http.MethodPost, "/api/v0/workspaces", `{"name":"`+name+`","slug":"`+slug+`"}`, http.StatusCreated)
	return slug
}

// seedAtScale writes TestTheDeletionsAtTheScaleTarget's rows into each of
// the projects, in one transaction, and analyzes the tables.
func (w issueWorld) seedAtScale(t *testing.T, projects []uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	alice, bob := w.ids["alice"], w.ids["bob"]
	for _, statement := range []struct {
		sql  string
		args []any
	}{
		{`CREATE TEMP TABLE scale_projects ON COMMIT DROP AS
			SELECT p.id AS project_id, p.workspace_id, s.id AS state_id, l.id AS label_id
			FROM projects p JOIN states s ON s.project_id = p.id AND s.name = 'Backlog' JOIN labels l ON l.project_id = p.id AND l.name = 'Bug'
			WHERE p.id = ANY ($1)`, []any{projects}},
		{`INSERT INTO issues (id, workspace_id, project_id, state_id, name, sequence_id, created_by_id, updated_by_id)
			SELECT uuidv7(), workspace_id, project_id, state_id, 'Item ' || n, n, $1, $1
			FROM (SELECT p.*, n FROM scale_projects p, generate_series(1, 20000) n ORDER BY random()) s`, []any{alice}},
		{`UPDATE projects SET last_issue_sequence = 20000 WHERE id = ANY ($1)`, []any{projects}},
		{`INSERT INTO issue_assignees (id, workspace_id, project_id, issue_id, assignee_id, created_by_id, updated_by_id)
			SELECT uuidv7(), workspace_id, project_id, id, $2, $2, $2
			FROM (SELECT * FROM issues WHERE project_id = ANY ($1) ORDER BY random()) i`, []any{projects, alice}},
		{`INSERT INTO issue_labels (id, workspace_id, project_id, issue_id, label_id, created_by_id, updated_by_id)
			SELECT uuidv7(), workspace_id, project_id, id, label_id, $1, $1
			FROM (SELECT i.*, p.label_id FROM issues i JOIN scale_projects p USING (project_id) ORDER BY random()) i`, []any{alice}},
		{`INSERT INTO issue_subscribers (id, workspace_id, project_id, issue_id, subscriber_id, created_by_id, updated_by_id)
			SELECT uuidv7(), workspace_id, project_id, id, subscriber, $2, $2
			FROM (SELECT i.*, subscriber FROM issues i, unnest(ARRAY[$2::uuid, $3::uuid]) subscriber WHERE i.project_id = ANY ($1)
				ORDER BY random()) i`, []any{projects, alice, bob}},
		{`INSERT INTO issue_activities (id, workspace_id, project_id, issue_id, verb, field, new_value, comment, actor_id, created_by_id,
				updated_by_id)
			SELECT uuidv7(), workspace_id, project_id, id, 'updated', 'name', 'Item ' || k, 'updated the name to', $2, $2, $2
			FROM (SELECT i.*, k FROM issues i, generate_series(1, 20) k WHERE i.project_id = ANY ($1) ORDER BY random()) i`,
			[]any{projects, alice}},
	} {
		if _, err := tx.Exec(ctx, statement.sql, statement.args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	w.vacuum(t)
}

// vacuum vacuums and analyzes the work-item side's tables.
func (w issueWorld) vacuum(t *testing.T) {
	t.Helper()
	if _, err := w.pool.Exec(context.Background(),
		"VACUUM ANALYZE issues, issue_assignees, issue_labels, issue_subscribers, issue_activities"); err != nil {
		t.Fatal(err)
	}
}

// errRolledBack ends rolledBack's transaction.
var errRolledBack = errors.New("rolled back")

// rolledBack is how long fn took in a transaction that it then rolls back,
// and vacuums away.
func (w issueWorld) rolledBack(t *testing.T, fn func(ctx context.Context) error) time.Duration {
	t.Helper()
	var took time.Duration
	err := postgres.NewTxManager(w.pool, time.Hour).WithinTx(context.Background(), func(ctx context.Context) error {
		start := time.Now()
		if err := fn(ctx); err != nil {
			return err
		}
		took = time.Since(start)
		return errRolledBack
	})
	if !errors.Is(err, errRolledBack) {
		t.Fatal(err)
	}
	w.vacuum(t)
	return took
}

// deletion is how long alice's DELETE of path took, through the API, with
// no deadline of the client's.
func (w issueWorld) deletion(t *testing.T, path string) time.Duration {
	t.Helper()
	req := newRequest(t, http.MethodDelete, w.base+path, w.tokens["alice"], nil)
	start := time.Now()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	took := time.Since(start)
	_ = res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE %s = %d, want 204", path, res.StatusCode)
	}
	return took
}
````

- [ ] **Step 2: 测试和 lint**

Run: `go -C server test -count=1 -v -run 'TestTheDeletionsAtTheScaleTarget$' ./internal/bootstrap/`
Expected: `ok`；`--- SKIP: TestTheDeletionsAtTheScaleTarget`（没有设 `NERVE_MEASURE_SCALE`）。

Run: `NERVE_MEASURE_SCALE=1 go -C server test -count=1 -v -timeout 60m -run 'TestTheDeletionsAtTheScaleTarget$' ./internal/bootstrap/`
Expected: `ok`，约两分钟（种下约 40 秒）；日志写下种下的行数（3000000）和四个时间。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 3: 提交**

```bash
git add server/internal/bootstrap/issue_scale_test.go
```
```bash
git commit -m "test(M4/P1): the deletions at the scale target, measured when asked for

With NERVE_MEASURE_SCALE set, six projects of two workspaces get 20,000
work items each, 500,000 rows of the work-item side a project, and the
deletion of a project and of a workspace is measured on the issue
module's cascade alone, rolled back, and through the API. make test and
CI skip it; M4 design decision point 7 has P9 measure its jobs on the
same data.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**：本 Task 加的是一个只在设了 `NERVE_MEASURE_SCALE` 时运行的测量，它量耗时；最后的"没有未删除的行"一句由 P1a 的按目录的删除测试守着，没有自己的变异。

---

### Task 14: 端到端：I2 的接口版本

**Files:**
- Create: `e2e/stories/issue/i2-update-issue.spec.ts`
- Modify: `e2e/fixtures/assert/issue.ts`、`e2e/fixtures/issues.ts`、`e2e/stories/issue/i1-create-issue.spec.ts`

**Interfaces:**（spec 2.16；M4 设计第 2 节 I2、3.7、3.13）
- `e2e/fixtures/issues.ts`：`IssueUpdate`、`issueUpdated`（状态和工作项，或问题的码和字段）、`updateIssue`（要求 200）。
- `e2e/fixtures/assert/issue.ts`：
  - `ActivityRow` 带 `named`、`was`（新、旧的标识指的账户的地址，或状态、标签的名称）；`activityOf`、`rowsOf`（未删除的负责人、标签和关注，各按次序；动态）、`rowStamps`（创建和修改共用的戳的读）；
  - `UpdatedIssue`、`expectIssueUpdated`：名称、状态、优先级、是否完成、最后修改者，负责人、标签、关注、动态，以及最后一次修改写下的行（`updated_at` 等于工作项的那些）恰是 `wrote` 行，都是修改者在那一刻写的：新写的在那一刻创建，去掉的在那一刻软删除，动态在那一刻由他做出。
- `i1-create-issue.spec.ts`：动态的行加 `was: null`。
- `i2-update-issue.spec.ts`（新）：I2 的接口版本。

**Tests:**（`make e2e`）
- I2：ann 建工作项 Fix the login（Backlog、没有优先级，指派 bob 和她自己，标签 Bug 和 Docs），和不被改的 Fix the logout（指派 bob，标签 Bug）；
  - bob 改名称（名称去掉空白）；
  - ann 把优先级改为 high、移到 Done：完成，优先级的行在状态的行之前；
  - bob 把自己换成管理员、把 Bug 换成 UI，ann 和 Docs 留下：管理员关注、bob 仍关注；四行动态；去掉的两行是 ann 写的，软删除后是 bob 的；一次九行；
  - ann 给出原样的值（集合按回答的反序）：不写，仍是 bob 的最后修改；
  - ann 再指派 bob、再加 Bug 和 Login（两个标签按 `id` 的反序给出）：标签的动态按 `id` 的次序，之后是 bob 的一行；bob 仍关注，不再写关注；六行；
  - bob 只去掉 Bug：是一次修改，两行；先前去掉的 Bug 保留那次的戳；
  - ann 只去掉 bob：同上；
  - bob 移回 In Progress：不再完成；
  - 大小写不对的优先级、给两次的账户和标签：422，三个问题一次报出；Web 没有的状态和标签：422，两个问题一次报出；都不写；
  - Fix the logout 仍如创建时（`expectIssueCreated`）：与它共有 bob 和 Bug 的修改没有写到它的行。
- I1 照旧通过。

- [ ] **Step 1: 夹具和断言**

`e2e/fixtures/assert/issue.ts`（修改，5 处）：

````old e2e/fixtures/assert/issue.ts
/** A row of a work item's activity as expectIssueCreated reads it: the account a row names, by its address. */
````
````new e2e/fixtures/assert/issue.ts
/**
 * A row of a work item's activity as the assertions read it: what its identifiers name, an account by its address, a
 * state or a label by its name.
 */
````
````old e2e/fixtures/assert/issue.ts
  /** The address of the account the row names (new_identifier), for an assignee's row; else null. */
  named: string | null;
  comment: string;
}
````
````new e2e/fixtures/assert/issue.ts
  /** What the row's new_identifier names: the account assigned, the state or the label set; else null. */
  named: string | null;
  /** What its old_identifier names: the account unassigned, the state left or the label taken off; else null. */
  was: string | null;
  comment: string;
}

/** The activity of the work item of issueId, in its order. */
async function activityOf(db: Database, issueId: string): Promise<ActivityRow[]> {
  return db.query<ActivityRow>(
    `SELECT a.field, a.verb, a.old_value, a.new_value, COALESCE(nu.email, ns.name, nl.name) AS named,
            COALESCE(ou.email, os.name, ol.name) AS was, a.comment
       FROM issue_activities a
       LEFT JOIN users nu ON nu.id = a.new_identifier LEFT JOIN states ns ON ns.id = a.new_identifier
       LEFT JOIN labels nl ON nl.id = a.new_identifier
       LEFT JOIN users ou ON ou.id = a.old_identifier LEFT JOIN states os ON os.id = a.old_identifier
       LEFT JOIN labels ol ON ol.id = a.old_identifier
      WHERE a.issue_id = $1 ORDER BY a.created_at, a.id`,
    [issueId]
  );
}

/**
 * The undeleted assignees and labels and the subscribers of the work item of issueId, by address or name, each in
 * order, and its activity.
 */
async function rowsOf(db: Database, issueId: string) {
  const named = async (sql: string) => (await db.query<{ v: string }>(sql, [issueId])).map(({ v }) => v);
  return {
    assignees: await named(`SELECT u.email AS v FROM issue_assignees r JOIN users u ON u.id = r.assignee_id
                              WHERE r.issue_id = $1 AND r.deleted_at IS NULL ORDER BY u.email COLLATE "C"`),
    labels: await named(`SELECT l.name AS v FROM issue_labels r JOIN labels l ON l.id = r.label_id
                           WHERE r.issue_id = $1 AND r.deleted_at IS NULL ORDER BY l.name COLLATE "C"`),
    subscribers: await named(`SELECT u.email AS v FROM issue_subscribers r JOIN users u ON u.id = r.subscriber_id
                                WHERE r.issue_id = $1 AND r.deleted_at IS NULL ORDER BY u.email COLLATE "C"`),
    activity: await activityOf(db, issueId),
  };
}

/**
 * The work item's assignees, labels, subscribers and activity, each row with what the stamps' checks compare: its
 * workspace and project, who made it and wrote it last, who acted (the activity's actor, else who made it), and when.
 */
const rowStamps = `
              SELECT workspace_id, project_id, created_by_id, updated_by_id, created_by_id AS actor, created_at, updated_at, deleted_at
                FROM issue_assignees WHERE issue_id = i.id
              UNION ALL SELECT workspace_id, project_id, created_by_id, updated_by_id, created_by_id, created_at, updated_at, deleted_at
                FROM issue_labels WHERE issue_id = i.id
              UNION ALL SELECT workspace_id, project_id, created_by_id, updated_by_id, created_by_id, created_at, updated_at, deleted_at
                FROM issue_subscribers WHERE issue_id = i.id
              UNION ALL SELECT workspace_id, project_id, created_by_id, updated_by_id, actor_id, created_at, updated_at, deleted_at
                FROM issue_activities WHERE issue_id = i.id`;
````
````old e2e/fixtures/assert/issue.ts
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
````
````new e2e/fixtures/assert/issue.ts
  expect(await rowsOf(db, issueId), `the rows of the work item ${issueId}`).toEqual({
    assignees,
    labels,
    subscribers,
    activity,
  });
````
````old e2e/fixtures/assert/issue.ts
       FROM issues i, LATERAL (
              SELECT workspace_id, project_id, created_by_id, updated_by_id, created_by_id AS actor, created_at, updated_at, deleted_at
                FROM issue_assignees WHERE issue_id = i.id
              UNION ALL SELECT workspace_id, project_id, created_by_id, updated_by_id, created_by_id, created_at, updated_at, deleted_at
                FROM issue_labels WHERE issue_id = i.id
              UNION ALL SELECT workspace_id, project_id, created_by_id, updated_by_id, created_by_id, created_at, updated_at, deleted_at
                FROM issue_subscribers WHERE issue_id = i.id
              UNION ALL SELECT workspace_id, project_id, created_by_id, updated_by_id, actor_id, created_at, updated_at, deleted_at
                FROM issue_activities WHERE issue_id = i.id) r
````
````new e2e/fixtures/assert/issue.ts
       FROM issues i, LATERAL (${rowStamps}) r
````
````old e2e/fixtures/assert/issue.ts
  });
}

````
````new e2e/fixtures/assert/issue.ts
  });
}

/**
 * A work item as changed: its name, state (by name) and priority; whether it is completed; the address of the account
 * that changed it last; its undeleted assignees and its subscribers by address, its undeleted labels by name, each in
 * order; its activity, in its order; and how many rows of its assignees, labels, subscribers and activity its last
 * change wrote, those it took off among them.
 */
export interface UpdatedIssue {
  name: string;
  state: string;
  priority: string;
  completed: boolean;
  by: string;
  assignees: string[];
  labels: string[];
  subscribers: string[];
  activity: ActivityRow[];
  wrote: number;
}

/**
 * I2: the work item of issueId holds want after a change (M4 design 3.7, 3.13): undeleted, last written by the
 * account of want.by at a moment after its creation, completed (completed_at set) exactly when want.completed. Its
 * undeleted assignees and labels, its subscribers and its activity are want's. The rows its last change wrote, those
 * last written at the work item's updated_at, want.wrote of them, are of its project and workspace and want.by's at
 * that moment: those it made, made then; those it took off, soft-deleted then; its activity, acted then.
 */
export async function expectIssueUpdated(db: Database, issueId: string, want: UpdatedIssue): Promise<void> {
  const [issue] = await db.query(
    `SELECT i.name, s.name AS state, i.priority, i.completed_at IS NOT NULL AS completed, u.email AS by,
            i.updated_at > i.created_at AND i.deleted_at IS NULL AS as_changed
       FROM issues i JOIN states s ON s.id = i.state_id JOIN users u ON u.id = i.updated_by_id
      WHERE i.id = $1`,
    [issueId]
  );
  const { assignees, labels, subscribers, activity, wrote, ...row } = want;
  expect(issue, `the work item ${issueId}`).toEqual({ ...row, as_changed: true });
  expect(await rowsOf(db, issueId), `the rows of the work item ${issueId}`).toEqual({
    assignees,
    labels,
    subscribers,
    activity,
  });
  const [stamps] = await db.query<{ rows: number; stamped: number }>(
    `SELECT count(*)::int AS rows,
            count(*) FILTER (WHERE r.workspace_id = i.workspace_id AND r.project_id = i.project_id
                               AND r.updated_by_id = i.updated_by_id
                               AND (r.deleted_at = i.updated_at
                                    OR r.deleted_at IS NULL AND r.created_at = i.updated_at AND r.created_by_id = i.updated_by_id
                                       AND r.actor IS NOT DISTINCT FROM i.updated_by_id))::int AS stamped
       FROM issues i, LATERAL (${rowStamps}) r
      WHERE i.id = $1 AND r.updated_at = i.updated_at`,
    [issueId]
  );
  expect(stamps, `the rows the last change of the work item ${issueId} wrote`).toEqual({ rows: wrote, stamped: wrote });
}

````

`e2e/fixtures/issues.ts`（修改，2 处）：

````old e2e/fixtures/issues.ts
export type IssueCreate = components["schemas"]["IssueCreate"];
````
````new e2e/fixtures/issues.ts
export type IssueCreate = components["schemas"]["IssueCreate"];
export type IssueUpdate = components["schemas"]["IssueUpdate"];
````
````old e2e/fixtures/issues.ts
  return created.issue;
}

````
````new e2e/fixtures/issues.ts
  return created.issue;
}

/**
 * The answer of PATCH /api/v0/issues/{issue_id} by the caller of token: its status, the work item, or the problem's
 * code and fields.
 */
export async function issueUpdated(api: Api, token: string, issueId: string, body: IssueUpdate) {
  const { data, error, response } = await api.PATCH("/api/v0/issues/{issue_id}", {
    params: { path: { issue_id: issueId } },
    body,
    headers: bearer(token),
  });
  return data ? { status: response.status, issue: data } : answer(response, error);
}

/** Changes the work item of issueId with the bearer token given, a member's of its project, and returns it. */
export async function updateIssue(api: Api, token: string, issueId: string, body: IssueUpdate): Promise<Issue> {
  const updated = await issueUpdated(api, token, issueId, body);
  expect(updated.status, `change the work item ${issueId}: ${JSON.stringify(updated)}`).toBe(200);
  if (!("issue" in updated)) {
    throw new Error(`updateIssue ${issueId} answered 200 without the work item`);
  }
  return updated.issue;
}

````

- [ ] **Step 2: I1 和 I2**

`e2e/stories/issue/i1-create-issue.spec.ts`（修改，2 处）：

````old e2e/stories/issue/i1-create-issue.spec.ts
  named: null,
````
````new e2e/stories/issue/i1-create-issue.spec.ts
  named: null,
  was: null,
````
````old e2e/stories/issue/i1-create-issue.spec.ts
    named: email,
````
````new e2e/stories/issue/i1-create-issue.spec.ts
    named: email,
    was: null,
````

`e2e/stories/issue/i2-update-issue.spec.ts`（新文件，303 行）：

````file e2e/stories/issue/i2-update-issue.spec.ts
import {
  addProjectMembers,
  amidAnotherWorkspace,
  createLabel,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import { expectIssueCreated, expectIssueUpdated, type ActivityRow } from "../../fixtures/assert/issue";
import { newAccount, shownNameOf } from "../../fixtures/auth";
import { createIssue, issueUpdated, updateIssue } from "../../fixtures/issues";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";

// I2, changing a work item (M4 design 2, 3.7, 3.13): the API version, with P1b's assertions (its name, state,
// priority, assignees and labels, each change's rows of its activity, completed_at, who changed it last and when). The
// dates are P2's, the description P4's, the page version P14's.

/** The order of the ids, which a work item answers its accounts and labels in, and a write adds them in. */
function byId(a: { id: string }, b: { id: string }): number {
  return a.id < b.id ? -1 : 1;
}

/**
 * A row of a change's activity: field changed from old to now, with Plane's comment; named and was, what its new and
 * old identifiers name, for a state's, a label's or an account's row.
 */
function changed(
  field: string,
  old: string,
  now: string,
  comment: string,
  named: string | null = null,
  was: string | null = null
): ActivityRow {
  return { field, verb: "updated", old_value: old, new_value: now, named, was, comment };
}

/** The row of a work item's activity that assigns it to the account of email, by the display name nerve gave him. */
function assigned(email: string): ActivityRow {
  return changed("assignees", "", shownNameOf(email), "added assignee ", email);
}

/** The first rows of a work item's activity: its creation, and its assignment to each of accounts, in id order. */
function createdFor(...accounts: { id: string; email: string }[]): ActivityRow[] {
  const creation: ActivityRow = {
    field: null,
    verb: "created",
    old_value: null,
    new_value: null,
    named: null,
    was: null,
    comment: "created the work item",
  };
  return [creation, ...accounts.toSorted(byId).map(({ email }) => assigned(email))];
}

test("I2 (API): a member renames a work item; another makes it high and moves it to a completed state, which completes it; the first assigns it to another account instead of himself and labels it otherwise, the others kept, taking off rows the second wrote; its values sent as they are write nothing; an account and two labels added again, the labels out of their ids' order, are added in id order; a label, then an account, taken off alone each change it, the rows taken off before keeping their stamps; moving it back completes it no more; problems of the values are refused before any read, and a state and a label the project lacks under the lock, each set at once (422), writing nothing; a work item sharing an account and a label with it is left as created: each change one row of its activity a field, a label or an account added or taken off, all its rows the changer's at its moment", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  // ann and bob, Web's members, each a member of Acme.
  const [ann, bob] = await Promise.all(["ann", "bob"].map((label) => newAccount(api, testInfo, label)));
  if (!ann || !bob) {
    throw new Error("the accounts were not registered");
  }
  await Promise.all([ann, bob].map((who) => inviteAndAccept(api, admin.token, slug, who, 15)));
  const web = await amidAnotherWorkspace(api, admin.token, testInfo, () =>
    createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" })
  );
  await addProjectMembers(api, admin.token, web.id, [
    { member_id: ann.id, role: 15 },
    { member_id: bob.id, role: 15 },
  ]);
  const [bug, ui, docs, login] = await Promise.all(
    ["Bug", "UI", "Docs", "Login"].map((name) => createLabel(api, admin.token, web.id, { name }))
  );
  if (!bug || !ui || !docs || !login) {
    throw new Error("the labels were not created");
  }
  const { Done: done, "In Progress": inProgress } = await stateIds(api, ann.token, web.id);
  if (!done || !inProgress) {
    throw new Error("Web's states hold no Done or no In Progress");
  }
  // ann's work item, in Backlog, of no priority, assigned to bob and to her and labelled Bug and Docs; and her Fix the
  // logout, assigned to bob and labelled Bug, which none of the changes below names.
  const { id } = await createIssue(api, ann.token, web.id, {
    name: "Fix the login",
    assignee_ids: [bob.id, ann.id],
    label_ids: [bug.id, docs.id],
  });
  const logout = await createIssue(api, ann.token, web.id, {
    name: "Fix the logout",
    assignee_ids: [bob.id],
    label_ids: [bug.id],
  });
  const created = createdFor(bob, ann);

  // bob renames it.
  expect(await updateIssue(api, bob.token, id, { name: "  Fix the sign-in " })).toMatchObject({
    name: "Fix the sign-in",
  });
  const renamed = changed("name", "Fix the login", "Fix the sign-in", "updated the name to");
  await expectIssueUpdated(db, id, {
    name: "Fix the sign-in",
    state: "Backlog",
    priority: "none",
    completed: false,
    by: bob.email,
    assignees: [ann.email, bob.email].toSorted(),
    labels: ["Bug", "Docs"],
    subscribers: [ann.email, bob.email].toSorted(),
    activity: [...created, renamed],
    wrote: 1,
  });

  // ann makes it high and moves it to Done, of the completed group: completed, the priority's row before the state's.
  expect(await updateIssue(api, ann.token, id, { state_id: done, priority: "high" })).toMatchObject({
    state_id: done,
    priority: "high",
  });
  const toDone = [
    changed("priority", "none", "high", "updated the priority to"),
    changed("state", "Backlog", "Done", "updated the state to", "Done", "Backlog"),
  ];
  await expectIssueUpdated(db, id, {
    name: "Fix the sign-in",
    state: "Done",
    priority: "high",
    completed: true,
    by: ann.email,
    assignees: [ann.email, bob.email].toSorted(),
    labels: ["Bug", "Docs"],
    subscribers: [ann.email, bob.email].toSorted(),
    activity: [...created, renamed, ...toDone],
    wrote: 2,
  });

  // bob assigns it to the admin instead of him and labels it UI instead of Bug, ann and Docs kept: the admin
  // subscribed, bob kept subscribed; four rows of activity, the label added, the label taken off, the account added,
  // the account taken off; the two rows taken off, ann's, soft-deleted and bob's now; nine rows in all. The work item
  // answers its assignees and labels in id order.
  const assignedIds = [ann.id, admin.id].toSorted();
  const labelledIds = [docs.id, ui.id].toSorted();
  expect(
    await updateIssue(api, bob.token, id, { assignee_ids: [ann.id, admin.id], label_ids: [docs.id, ui.id] })
  ).toMatchObject({ assignee_ids: assignedIds, label_ids: labelledIds });
  const afterSwap = {
    name: "Fix the sign-in",
    state: "Done",
    priority: "high",
    completed: true,
    by: bob.email,
    assignees: [admin.email, ann.email].toSorted(),
    labels: ["Docs", "UI"],
    subscribers: [admin.email, ann.email, bob.email].toSorted(),
    activity: [
      ...created,
      renamed,
      ...toDone,
      changed("labels", "", "UI", "added label ", "UI"),
      changed("labels", "Bug", "", "removed label ", null, "Bug"),
      assigned(admin.email),
      changed("assignees", shownNameOf(bob.email), "", "removed assignee ", null, bob.email),
    ],
    wrote: 9,
  };
  await expectIssueUpdated(db, id, afterSwap);

  // ann sends its name, state, priority, assignees and labels as they are, the sets in the reverse of their order:
  // nothing changes, nothing is written, and it stays bob's last change.
  expect(
    await updateIssue(api, ann.token, id, {
      name: "Fix the sign-in",
      state_id: done,
      priority: "high",
      assignee_ids: assignedIds.toReversed(),
      label_ids: labelledIds.toReversed(),
    })
  ).toMatchObject({ name: "Fix the sign-in", state_id: done, assignee_ids: assignedIds, label_ids: labelledIds });
  await expectIssueUpdated(db, id, afterSwap);

  // ann assigns bob again and labels it Bug again and Login, the two in the reverse of their ids' order: their rows of
  // activity in id order, then bob's; bob, subscribed still, is not subscribed again; six rows.
  const again = [bug, login].toSorted(byId);
  expect(
    await updateIssue(api, ann.token, id, {
      assignee_ids: [ann.id, admin.id, bob.id],
      label_ids: [docs.id, ui.id, ...again.map((label) => label.id).toReversed()],
    })
  ).toMatchObject({
    assignee_ids: [ann.id, admin.id, bob.id].toSorted(),
    label_ids: [docs.id, ui.id, bug.id, login.id].toSorted(),
  });
  const addedAgain = {
    ...afterSwap,
    by: ann.email,
    assignees: [admin.email, ann.email, bob.email].toSorted(),
    labels: ["Bug", "Docs", "Login", "UI"],
    activity: [
      ...afterSwap.activity,
      ...again.map((label) => changed("labels", "", label.name, "added label ", label.name)),
      assigned(bob.email),
    ],
    wrote: 6,
  };
  await expectIssueUpdated(db, id, addedAgain);

  // bob takes Bug off it, nothing else: a change, its two rows; the Bug he took off before keeps that change's stamps.
  await updateIssue(api, bob.token, id, { label_ids: [docs.id, login.id, ui.id] });
  const bugOff = {
    ...addedAgain,
    by: bob.email,
    labels: ["Docs", "Login", "UI"],
    activity: [...addedAgain.activity, changed("labels", "Bug", "", "removed label ", null, "Bug")],
    wrote: 2,
  };
  await expectIssueUpdated(db, id, bugOff);

  // ann takes bob off it, nothing else: a change too; the bob taken off before keeps that change's stamps.
  await updateIssue(api, ann.token, id, { assignee_ids: [ann.id, admin.id] });
  const bobOff = {
    ...bugOff,
    by: ann.email,
    assignees: [admin.email, ann.email].toSorted(),
    activity: [
      ...bugOff.activity,
      changed("assignees", shownNameOf(bob.email), "", "removed assignee ", null, bob.email),
    ],
    wrote: 2,
  };
  await expectIssueUpdated(db, id, bobOff);

  // bob moves it back to In Progress: completed no more.
  await updateIssue(api, bob.token, id, { state_id: inProgress });
  const movedBack = {
    ...bobOff,
    state: "In Progress",
    completed: false,
    by: bob.email,
    activity: [
      ...bobOff.activity,
      changed("state", "Done", "In Progress", "updated the state to", "In Progress", "Done"),
    ],
    wrote: 1,
  };
  await expectIssueUpdated(db, id, movedBack);

  // A priority in another case (the contract's values are lower case), an account and a label named twice are refused
  // before the work item is read, the three at once; a state and a label Web lacks under the lock, the two at once.
  // Neither refused change writes anything: it stays bob's last change.
  expect(
    await issueUpdated(api, ann.token, id, {
      name: "Fix it",
      // @ts-expect-error -- a priority in another case breaks the contract on purpose
      priority: "High",
      assignee_ids: [ann.id, ann.id],
      label_ids: [ui.id, ui.id],
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
    await issueUpdated(api, ann.token, id, {
      name: "Fix it",
      state_id: crypto.randomUUID(),
      label_ids: [crypto.randomUUID()],
    })
  ).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [
      { field: "state_id", code: "not_allowed" },
      { field: "label_ids[0]", code: "not_allowed" },
    ],
  });
  await expectIssueUpdated(db, id, movedBack);

  // Fix the logout, which shares bob and Bug with it, is as ann created it: no change above wrote a row of it.
  await expectIssueCreated(db, logout.id, {
    sequence_id: 2,
    name: "Fix the logout",
    state: "Backlog",
    priority: "none",
    sort_order: logout.sort_order,
    completed: false,
    by: ann.email,
    assignees: [bob.email],
    labels: ["Bug"],
    subscribers: [ann.email, bob.email].toSorted(),
    activity: createdFor(bob),
  });
});
````

- [ ] **Step 3: 测试和 lint**

Run: `make lint-web`
Expected: 通过；关键词守卫 `keywords: 72 rules, 23 exceptions, no hits.`

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 122 个通过（此前的 121 个，加 I2）。

- [ ] **Step 4: 提交**

```bash
git add e2e/fixtures/assert/issue.ts e2e/fixtures/issues.ts e2e/stories/issue/i1-create-issue.spec.ts e2e/stories/issue/i2-update-issue.spec.ts
```
```bash
git commit -m "test(M4/P1): I2's API version, a work item changed field by field

A member renames a work item; another makes it high and moves it to
Done, which completes it; the first swaps one of its assignees and one
of its labels for others, the rest kept, and the rows he takes off are
his; values sent as they are write nothing; an account and two labels
added again are added in id order; a label, then an account, taken off
alone each make a change, and the rows taken off before keep their
stamps; moving it back completes it no more; problems of the values and
of the project are refused at once and write nothing; another work item
with the same assignee and label stays as created. Each change is
checked row by row: its activity, its assignees, labels and
subscribers, and every row it wrote its changer's at its moment.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**：本 Task 没有自己的变异。

本 Task 的检查也发现较早的 Task 的变异 `T1.1`、`T1.2`、`T1.3`、`T1.4`、`T1.5`、`T1.6`、`T1.7`、`T1.8`、`T1.9`、`T1.10`、`T1.11`、`T1.12`、`T1.13`、`T1.14`、`T1.15`、`T2.1`、`T2.2`、`T2.3`、`T3.1`、`T3.2`、`T3.3`、`T3.4`、`T3.5`、`T3.6`、`T3.7`、`T3.8`、`T3.10`、`T3.11`、`T5.2`、`T5.4`、`T5.9`、`T5.12`、`T5.19`、`T6.1`、`T7.4`（列在各自的 Task 的表里，标"（Task 14 起）"）：本 Task 做完之后，在本 Task 的树上重跑它们（`mutants_p1b.py`，`MUT_LATER=14`）。

---

### Task 15: 文档：写闸门、P1b 的差异和配置，交接的处理结果

**Files:**
- Modify: `README.md`、`docs/v0/M4-issue-core/handoffs/M2-closeout.md`、`docs/v0/M4-issue-core/handoffs/M3-closeout.md`、`docs/v0/plane-diff.md`、`docs/v0/v0-design.md`

**Interfaces:**（spec 2.17、第 7 节；M4 设计 3.5、3.23、8.8）
- 总体设计：
  - 4.2：工作项一侧的写在预检之后、取连接之前进入项目的写闸门；`issue` 的四个配置；503 `server_busy` 只在 `bootstrap/gate.go` 换码；连接池的代价（交错 20 的数字；闸门限的是每个项目的份额，同时热的项目达到池的大小除以 K 时池仍会被等锁的写占满）和多进程时的 K；
  - 6.2：`platform/gate`。
- 差异清单第四节 4.17：P1b 的三行（一个项目上的并发写、修改的响应、负责人的读取），接在 P1a 的取锁一行之后。
- README 的部署说明：写闸门和锁的期限一条（四个配置，8.8）。
- 两份交接（`M2-closeout.md`、`M3-closeout.md`）各写下"处理结果（M4/P1b）"，`status` 照旧 `open`（spec 第 7 节）。

**Tests:** 只改文档：`make lint-web`（关键词守卫读文档）和 `doclinks.py`（相对链接）。

- [ ] **Step 1: 文档**

`README.md`（修改，1 处）：

````old README.md
- **数据库的 locale**：用 UTF-8 的 locale 初始化数据库（官方的 PostgreSQL 镜像是 `en_US.utf8`）。标签的名称在项目内不分大小写，按数据库的 `LC_CTYPE` 折叠（`lower(name)`）：以 `C` 初始化的数据库只折叠 ASCII，`Ä` 与 `ä` 算两个名称（M3/P7b spec 第 7 节）。
````
````new README.md
- **数据库的 locale**：用 UTF-8 的 locale 初始化数据库（官方的 PostgreSQL 镜像是 `en_US.utf8`）。标签的名称在项目内不分大小写，按数据库的 `LC_CTYPE` 折叠（`lower(name)`）：以 `C` 初始化的数据库只折叠 ASCII，`Ä` 与 `ä` 算两个名称（M3/P7b spec 第 7 节）。
- **工作项的写闸门和锁的期限**：一个项目上的工作项的写，每个进程里至多 `issue.write_gate.per_project`（默认 2）个同时在数据库里；多的在进程里排队，至多 `issue.write_gate.max_queue`（默认 64）个，每个至多等 `issue.write_gate.max_wait`（默认 2 秒）；在数据库里每一次等锁至多 `issue.write_lock_timeout`（默认 5 秒；加上 `max_wait` 必须短于 `server.request_timeout`）。排满、等不到、请求到期或等锁到期的写得到 503 `server_busy`，带 `Retry-After: 1`，客户端隔这么久重试。闸门让一个热的项目至多占用连接池（`database.max_conns`，默认 10）里的这么多个连接：同时热的项目少于 `max_conns / per_project`（默认 5）个时，别的项目的读写不排在它们后面；更多的项目同时热时（比如删除一个大的工作区，它的每个项目上都有写在等行锁），等锁的写仍能占满连接池，别的请求在池的队列里等到这些写等锁到期。部署多个 nerve 进程时每个进程各有这么多个：连接池的大小怎样配合 `per_project`、同时热的项目数和进程数，由 M8 的部署说明定。
````

`docs/v0/M4-issue-core/handoffs/M2-closeout.md`（修改，1 处）：

````old docs/v0/M4-issue-core/handoffs/M2-closeout.md
来源：[M2 设计](../../M2-auth/M2-design.md) 13.2 中接收者含 M4 的各行；[M0-P6 的交接](../../M2-auth/handoffs/M0-P6-e2e-notes.md)；[M2 收尾 spec](../../M2-auth/specs/closeout.md) 第 3 节。

````
````new docs/v0/M4-issue-core/handoffs/M2-closeout.md
来源：[M2 设计](../../M2-auth/M2-design.md) 13.2 中接收者含 M4 的各行；[M0-P6 的交接](../../M2-auth/handoffs/M0-P6-e2e-notes.md)；[M2 收尾 spec](../../M2-auth/specs/closeout.md) 第 3 节。

## 处理结果（M4/P1b）

- **第 3 节，`updated_at`**（P1a、P1b 的写完成）：`createIssue`（P1a）和 `updateIssue`（P1b）把用例的时钟在项目 N 之下读到的一个时刻写进它写的每一行：修改写下工作项的 `updated_at`、`updated_by_id`，新加的负责人、标签、关注和每一行动态的 `created_at`、`updated_at`，去掉的负责人、标签的 `deleted_at`、`updated_at`。用例的测试用固定的时钟（`issue/app` 测试的 `clockAt`）断言这些列等于它；组合的测试断言它落在请求之内（`TestUpdatingAWorkItemStampsItsRows`）；端到端的 I2 断言修改写下的每一行是工作项的 `updated_at`、是修改者的。

仍未处理，状态保持 `open`：第 3 节其余写工作项的用例（M4 的 P2–P4、P7）和自动归档推进固定时钟的测试（M4 的 P6）；第 1、2、4–13 节，按 M4 设计 13.1 的落点。

来源：[M4/P1b spec](../specs/P1b-issue-update.md) 第 7 节。

````

`docs/v0/M4-issue-core/handoffs/M3-closeout.md`（修改，1 处）：

````old docs/v0/M4-issue-core/handoffs/M3-closeout.md
来源：[M4/P1a spec](../specs/P1a-issues.md) 第 7 节。

````
````new docs/v0/M4-issue-core/handoffs/M3-closeout.md
来源：[M4/P1a spec](../specs/P1a-issues.md) 第 7 节。

## 处理结果（M4/P1b）

- **第 2 节，连接池的代价和写闸门**（完成）：工作项一侧的写在预检之后、开始事务之前进入它的项目的写闸门（`platform/gate`；`issue/app` 声明的 `WriteGate`，`bootstrap/gate.go` 接线），不持连接、不持锁地等：每个进程里一个项目至多 `issue.write_gate.per_project`（默认 2）个写在数据库里，排队至多 `issue.write_gate.max_queue`（64）个、每个至多 `issue.write_gate.max_wait`（2 秒）；写的事务带 `issue.write_lock_timeout`（5 秒）的 `lock_timeout`。排满、等不到、请求到期、等锁到期都答 503 `server_busy`，带 `Retry-After: 1`，只在 `bootstrap/gate.go` 一处换码（M4 设计 3.5、11.11）。交错 20 经真实的连接池（10 个连接）核对：200 个写压在一个项目上、它的行被别的事务持着时，另一个项目的读最慢几十毫秒（没有闸门时一秒以上），超出闸门的 134 个写立即 503；首尾相连的工作项写之下，改工作区、改成员的角色、删除工作区各在 1 秒之内（M4/P1b spec 第 3 节）。闸门限的是每个项目的份额：一个进程里同时热的项目达到池的大小除以 K（默认 5 个）时池仍会被等锁的写占满；连接池的大小与 K、同时热的项目数、进程数交给 M8（M4 设计 13.2，收尾时写进 M8 的交接）。
- **第 2 节，约定五**（P1b 的批量写完成）：修改的两条批量写（`RemoveAssignees`、`RemoveLabels`：按谓词软删除一个工作项被去掉的负责人、标签）都在项目 N 之下，那些行只有持项目 N 的写会写到，不落在约定五的例外里；新加的负责人、标签、关注按 `id` 的顺序插入（M4/P1b spec 第 3 节）。

仍未处理，状态保持 `open`：第 2 节其余各项：加关联的另一端的预检（M4 的 P3）、P1b 之后各 Phase 的批量写按约定五核对（到 M4 的 P9）、物理删除账户（M4 的 P9）；第 4 节的 `guest_view_all_features`（M4 的 P2）、`archive_in`（M4 的 P6）的说明，`expectIssueDeleted`（M4 的 P2）和删除断言的限时轮询（M4 的 P9）；第 1、3、5–9 节，按 M4 设计 13.1 的落点。

来源：[M4/P1b spec](../specs/P1b-issue-update.md) 第 7 节。

````

`docs/v0/plane-diff.md`（修改，1 处）：

````old docs/v0/plane-diff.md
| 工作项的写的取锁 | 动态在 Celery 里异步写，写之间不串行 | 不加锁地预检，判定拒绝的在那里答（404、403），不取行锁；放行的写以 `FOR NO KEY UPDATE` 锁住项目行，在锁下重读、再判定，动态、关注与工作项在同一个事务里写（M4 设计 3.5，决策点 1） |
````
````new docs/v0/plane-diff.md
| 工作项的写的取锁 | 动态在 Celery 里异步写，写之间不串行 | 不加锁地预检，判定拒绝的在那里答（404、403），不取行锁；放行的写以 `FOR NO KEY UPDATE` 锁住项目行，在锁下重读、再判定，动态、关注与工作项在同一个事务里写（M4 设计 3.5，决策点 1） |
| 一个项目上的并发写 | 不限 | 每个进程里一个项目至多 K 个写同时在数据库里（`issue.write_gate.per_project`，默认 2），多的在进程里排队、不持连接；排满、等过 `issue.write_gate.max_wait`、等锁超过 `issue.write_lock_timeout` 时 503 `server_busy`，带 `Retry-After: 1`（M4 设计 3.5，架构问题 11） |
| 修改工作项的响应 | 204 | 200，回答修改之后的 `IssueDetail`；什么都没改的修改不写任何行，回答锁下读到的工作项（总体设计 3.1；M4 设计 3.7） |
| 工作项的负责人的读取 | 只返回"在某个项目里仍有效"的负责人 | 返回全部未删除的负责人行，离开了项目的负责人照样列出；修改只检查新加的负责人，留下的不查（M4 设计 3.6、3.7） |
````

`docs/v0/v0-design.md`（修改，2 处）：

````old docs/v0/v0-design.md
  工作项一侧的写从 M4/P1a 起落地（M4 设计 3.5，决策点 1 负责人裁定 B）。约定二对它们的延伸是**最先锁项目行，不锁工作区行**：不加锁地读出资源（按项目 id 的写读项目，按资源 id 的写读资源、得到工作项和项目，没有时答资源的 404），再做一次不加锁的预检，**凡是判定拒绝的都在这里答**（看不到答资源的 404，看得到而不能做答 403），不取行锁；放行的写开始事务，以 `FOR NO KEY UPDATE` 锁住项目行（带 `deleted_at IS NULL`），重读资源、在锁下再判定一次（等锁期间被移出、降级的人照样得到 404 或 403，锁下的判定是最终的），然后检查依赖的行、读时刻、写入。项目的 N 让同一个项目里的写一个接一个（取号、同一状态最大的 `sort_order`、动态按提交的顺序得到不减的时刻都靠它）；不锁工作区行，最频繁的写就不会像 M3 设计 17.4 的探针 B 那样让工作区一级的写一直等下去。删除工作区在锁住它的邀请之后、读时刻之前，按 `id` 升序以 `FOR NO KEY UPDATE` 锁住它的全部未删除项目（`ProjectCascade.LockWorkspaceProjects`），所以它的时刻不早于这些项目上已提交的每个写。B 的两个代价：一、授权与时刻的先后可以不一致：只改工作区角色（不降为访客）的写不锁项目，降为访客、结束成员关系的连带在锁项目之前读时刻，一个工作项的写可以在成员关系的变更读了时刻之后才写下更晚的时刻、按旧的角色生效；二、草稿的写、发布草稿（M4 的 P8）和 M3 的项目管理类的写仍先取工作区 S，M3 设计 17.4 的饥饿在这些较少的路径上仍在。
````
````new docs/v0/v0-design.md
  工作项一侧的写从 M4/P1a 起落地（M4 设计 3.5，决策点 1 负责人裁定 B）。约定二对它们的延伸是**最先锁项目行，不锁工作区行**：不加锁地读出资源（按项目 id 的写读项目，按资源 id 的写读资源、得到工作项和项目，没有时答资源的 404），再做一次不加锁的预检，**凡是判定拒绝的都在这里答**（看不到答资源的 404，看得到而不能做答 403），不取行锁；放行的写开始事务，以 `FOR NO KEY UPDATE` 锁住项目行（带 `deleted_at IS NULL`），重读资源、在锁下再判定一次（等锁期间被移出、降级的人照样得到 404 或 403，锁下的判定是最终的），然后检查依赖的行、读时刻、写入。项目的 N 让同一个项目里的写一个接一个（取号、同一状态最大的 `sort_order`、动态按提交的顺序得到不减的时刻都靠它）；不锁工作区行，最频繁的写就不会像 M3 设计 17.4 的探针 B 那样让工作区一级的写一直等下去。删除工作区在锁住它的邀请之后、读时刻之前，按 `id` 升序以 `FOR NO KEY UPDATE` 锁住它的全部未删除项目（`ProjectCascade.LockWorkspaceProjects`），所以它的时刻不早于这些项目上已提交的每个写。B 的两个代价：一、授权与时刻的先后可以不一致：只改工作区角色（不降为访客）的写不锁项目，降为访客、结束成员关系的连带在锁项目之前读时刻，一个工作项的写可以在成员关系的变更读了时刻之后才写下更晚的时刻、按旧的角色生效；二、草稿的写、发布草稿（M4 的 P8）和 M3 的项目管理类的写仍先取工作区 S，M3 设计 17.4 的饥饿在这些较少的路径上仍在。

  从 M4/P1b 起，工作项一侧的写在预检之后、开始事务（取连接）之前进入它的项目的**写闸门**（M4 设计 3.5，架构问题 11）：每个进程里一个项目至多 K 个写同时在数据库里（`issue.write_gate.per_project`，默认 2），多出的在进程里排队，不持连接、不持锁，排队至多 `issue.write_gate.max_queue` 个（默认 64），每个至多等 `issue.write_gate.max_wait`（默认 2 秒）；进入数据库的写，每一次等锁至多 `issue.write_lock_timeout`（默认 5 秒，事务开始时 `SET LOCAL lock_timeout`）。排满、等不到、请求到期和等锁到期都答 503 `server_busy`，带 `Retry-After: 1`，只在组合根（`bootstrap/gate.go`）一处换成这个码。连接池的代价：等项目行的写持着池里的一个连接，没有闸门时一个热的项目能占满整个池（`database.max_conns`，默认 10），别的项目上无关的读写排在池的等待者后面（M4 设计的 SP1：200 个写排在一个项目上时，别的项目的读等约 875 ms）；有了闸门，一个项目至多占 K 个连接（M4/P1b 的交错 20：同样的突发之下别的项目的读几十毫秒，没有闸门时一秒以上）；闸门限的是每个项目的份额，同时热的项目达到池的大小除以 K（默认 5 个）时，等锁的写仍能占满池。代价是热的项目上超出的写得到 503、要由客户端按 `Retry-After` 重试。多个进程时每个进程各有 K 个，池的大小与 K、同时热的项目数由 M8 的部署说明定。
````
````old docs/v0/v0-design.md
                            logging、httpserver（中间件、problem+json）、ratelimit、clock、idgen
````
````new docs/v0/v0-design.md
                            logging、httpserver（中间件、problem+json）、ratelimit、clock、idgen、
                            gate（带 context 的按键的信号量：工作项的写闸门，M4 设计 3.5）
````

- [ ] **Step 2: 检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 `keywords: 72 rules, 23 exceptions, no hits.`

Run: `python3 $M4TMP/p1btools/doclinks.py . README.md docs/v0/v0-design.md docs/v0/plane-diff.md docs/v0/M4-issue-core/handoffs/M2-closeout.md docs/v0/M4-issue-core/handoffs/M3-closeout.md docs/v0/M4-issue-core/specs/P1b-issue-update.md docs/v0/M4-issue-core/plans/P1b-issue-update.md`
Expected: 每个文件 `0 broken`，最后一行 `broken: 0`。

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`（文档不影响它；本 Phase 最后一次全量运行）。

- [ ] **Step 3: 提交**

```bash
git add README.md docs/v0/M4-issue-core/handoffs/M2-closeout.md docs/v0/M4-issue-core/handoffs/M3-closeout.md docs/v0/plane-diff.md docs/v0/v0-design.md
```
```bash
git commit -m "docs(M4/P1): the design documents and the handoffs record P1b

The overall design's lock order gains the write gate: a write on a
project's work items enters its project's gate after its precheck and
before it takes a connection, its waits for locks bounded, and what a
busy project answers; platform/gate joins the platform. The Plane diff
holds P1b's rows, the README the four settings, and the two handoffs'
outcomes say what P1b did.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**：本 Task 只改文档，没有变异（文档的相对链接由 `doclinks.py` 核对）。
