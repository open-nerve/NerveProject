# M4/P2 访客规则、创建者、删除、按编号取、归档、起止日期 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 访客只看自己的工作项（`OwnOnly`，按项目的角色），一条访客规则用在每个指名工作项的读和写上：预检和锁下的判定都先答 404、再答角色的 403；创建者的权限（`AllowCreator`：仍是项目的有效成员、角色不低于成员，锁下再判定）；`deleteIssue`（工作项和 P1a 的四张子表，同一时刻、同一个删除者，在 P1b 的取锁路径上）；`getIssueByIdentifier`（`/workspaces/{slug}/issues/{identifier}`，折叠比较，各种 404 是同一个回答）；`archiveIssue`、`unarchiveIssue`（409 `issue.not_archivable`，动态第 14、16 行）；已归档的工作项只读（409 `issue.archived`）；起止日期（`shared.Date`、`bodyshape` 的 `date`、生成器、类型映射，修改按合并之后的值检查，动态第 8、9 行）。交错 12、交错 5 的访客一支、删除与修改、等锁期间被降级或移出的创建者在真实数据库上；矩阵的访客三列、"管理员·访客"、建了它的项目成员；I15 的接口版本、I1 和 I2 的日期、I2 的已归档 409 通过（M4 设计 12 节 P2）。

**Architecture:** `shared`：`Target.CreatorID`、`Grant.OwnOnly`、`date.go`。`access`：读 `guest_view_all_features`，`AllowCreator`，拒绝时也交回 `Grant`；规则表的 `issue.read`（加访客）、`issue.delete`、`issue.archive`、`issue.search`。`issue` 模块：`domain`（`HiddenFrom`、`Archivable`、`identifier.go`、`dates.go`，动态第 8、9、14、16 行）、`app`（`lock.go` 的 `decide` 先执行访客规则，`seeWorkspace`；`delete_issue.go`、`get_issue_by_identifier.go`、`archive_issue.go`；每个写仍经 `Locks.run`）、`adapter/postgres`（`delete_issue.sql`、`SetArchivedAt`、`IssueByNumber`、日期的两列）、`adapter/http`（四个新操作，日期）。`project`：`ProjectIdentifiers`；`state_in_use` 的补救。`platform/httpserver`：`bodyshape` 的 `FormatDate`、`paths.go`（下标按数值排序）、开放节点的数；`apitest` 的 `date`。`server/tools/bodyshapegen`：`shared.Date` 的检查器。`bootstrap`：接线（`issueWorkspaces`、`Identifiers`）、矩阵的工作项表和 `prepare`、按目录的删除测试、取锁形状、交错。`e2e`：I15、I1、I2。不加 Go 模块、npm 包；没有新表、新迁移；模块之间只经端口。

**Tech Stack:** Go 1.27.1、pgx v5.11.0、sqlc v1.31.1（`CGO_ENABLED=0`）、oapi-codegen v2.8.0、goose v3.28.0、River v0.47.0、golangci-lint 2.13.2、PostgreSQL 18.6（testcontainers）；Node 24、pnpm 11.10.0、Playwright 1.63.0、TypeScript 5.8.3、oxlint 1.51.0、oxfmt 0.35.0。

**Spec:** `docs/v0/M4-issue-core/specs/P2-issue-access.md`（上级：`docs/v0/M4-issue-core/M4-design.md`）

## Global Constraints

- **基点**：本 plan 和 spec 在 `7db78788`（P1b 的合并提交）之上的一个提交里，M4 设计第 15 节 P2 一行在同一个提交里改为"进行中"；Task 1 从这个提交开始。
- **Go 版本和依赖**：本 plan 不执行 `go get`，不加任何 Go 模块或 npm 包（`shared/date.go` 只用标准库）。`server/go.mod` 和 `server/tools/go.mod` 保持 `go 1.27` 和 `toolchain go1.27.1`；`git diff --stat 7db78788e3b836c01134b02c0deb56e26d72851e -- server/go.mod server/go.sum server/tools/go.mod server/tools/go.sum pnpm-lock.yaml` 在本 plan 的任何时刻都没有输出（Task 9 改 `server/tools/bodyshapegen` 的代码和测试数据，不改那个模块的依赖）。
- **每个改了 `server/` 的 Task（Task 1–12）提交前**：`make lint-go` 两段都输出 `0 issues.`；`make test` 全部通过（没有 `FAIL`；它也跑 `server/tools` 模块的测试，Task 9 的生成器的测试在那里）。Task 13 只改 `e2e/`，跑 Web 的三个门禁和 `make e2e`；Task 14 只改文档，跑 `make lint-web`、`doclinks.py` 和本 Phase 最后一次全量的 `make test`。
- **生成的文件不手写、不从本 plan 复制**：有生成物的 Task 执行 Task 中的生成命令（只动了 sqlc 的 Task 1、5、10 执行 `make gen-go`，改了契约的 Task 2、4、6、7、8、11 执行 `make gen`；Task 9 执行 `make gen`，它只改映射和生成器，生成物不变），用 `shasum -a 256` 和 `wc -l` 核对生成命令下面的表中的 SHA-256 和行数（Task 9 没有表：生成物与之前逐字节相同），对不上时停下来：说明某个输入与本 plan 不一致。这些 Task **提交之后**执行 `make gen-check`（它用 `git status` 判断生成物是否已提交，提交之前必然报差异）。
- **前端检查和新的错误码**（M4 设计 12 节约束 4）：声明新码的 Task 在同一个 Task 里把它的文案加进 `web/apps/web/core/lib/error-messages.ts`（`PROBLEM_MESSAGES`）和两份 `errors.json`（`web/packages/i18n/src/locales/en/errors.json`、`web/packages/i18n/src/locales/zh-CN/errors.json`）：Task 7 `issue.not_archivable`，Task 8 `issue.archived`（Task 8 同时改 `project.state_in_use` 的文案）。`workspace.not_found`（Task 5、6 的 `getIssueByIdentifier`）是 M3 的码，三处已有它的文案，不另加。改了契约、前端或 `e2e/` 的 Task 2、4、6、7、8、11、13 另执行 `make lint-web`、`make knip`、`make test-web`；改了故事经过的读写的 Task 2（每个工作项的判定先执行访客规则）、Task 11（日期进契约和创建、修改）和 Task 13 执行 `make e2e`；Task 14 只改文档，执行 `make lint-web` 和 `doclinks.py`（下面"一次性的脚本"）。
- **e2e 的数目**：在 worktree 中 `make e2e` 必须全部通过：开始时 122 个，Task 13 起 123 个（I15）。只在不是 git 仓库的副本里，S3 因为构建读不到提交而失败（M3/P4b spec F4）；没有装 Firefox 的机器上 A4 的两个 firefox 测试因为浏览器不在而失败；两者都与本 plan 无关。
- **容器**：`make test` 和 `make e2e` 用自己的 testcontainers；机器忙时偶尔起不来，等 Docker 空闲之后重跑一次再当作失败（容器起不来不算任何检查的失败）。容器测试一次只跑一套。开发库 `nerve-dev-db-1` 不要启动、停止或重建，不要执行 `make dev-db-down`、`make dev-db-reset`。不要碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`、`nervewiki-*`）和留下的 testcontainers。
- **git**：每次 Bash 调用只执行一个 git 命令，不用 `;`、`&&`、`|` 串联 git；不用 `git -C`、`stash`、`clean`、`reset --hard`。不用 `cd`。不碰 `plane/`、`refer/`。
- **安装**：除了 Docker、Go、Node 不做任何全局安装；不执行 `corepack enable`（pnpm 已在 PATH 上）。不把副本的 `node_modules` 链接到 worktree 的。
- **规则**：不写 `init()`，不用全局可变状态，构造函数显式传入依赖；不建 `utils`、`common`、`helpers` 包；一个文件只做一件事，不超过约 400 行；依赖只能向内；模块之间不互相导入，不跨模块的表 JOIN；模块的 SQL 只经 sqlc；端口由使用方声明；角色只按集合判断，不按大小比较；不留没有使用者的代码。工作项一侧的每个写经 `issue/app/lock.go` 的 `Locks.run`（一条共用的取锁路径），每个指名工作项的判定经 `Locks.decide`（访客规则在那里），不写近似的副本；测试的帮手共用、扩展，不复制、不重写（`issue_world_test.go` 的 `issueWorld`、`newWorkItem`、`newWorkItemIn`、`workItemStamps`、`activityBy`，`issue_interleaving_test.go` 的 `itemWrite`、`wroteNothing`，`serializeOn`、`heldTx`、`holding`，矩阵的帮手，`e2e/fixtures/assert/issue.ts` 的 `byId`、`changed`、`assigned`、`createdFor`；一种动态的行一个构造）。**例外**：契约按模块一个文件（M0-P3 交接 5），`api/modules/issue.yaml`（481 行）和 `api/modules/project.yaml`（1363 行）不受约 400 行的限制；关键词守卫的规则表 `tools/keywords.json`（2670 行，数据，本 plan 只改它的 `phase`）同样；`bootstrap/permission_matrix_test.go`（406 行：基点已是 399 行，Task 7 加 `matrixRow.prepare` 的字段和它在格子的副本上的一步）。本 plan 改的其余代码文件都在约 400 行以内（最终原型上量的）：最长的是 `issue/app/fakes_test.go`（400 行）、`bootstrap/issue_interleaving_test.go`（394 行）；新建的文件中最长的是 `issue/app/archive_issue_test.go`（195 行）、`issue/app/get_issue_by_identifier_test.go`（170 行）、`bootstrap/permission_matrix_issue_columns_test.go`（162 行）。
- **关键词守卫**：72 条规则、23 个例外，没有命中。Task 1 把 `tools/keywords.json` 的 `phase` 改为 `M4/P2`，因此也执行 `make lint-web`。
- **注释**：Go、TS 代码、SQL 查询和接口描述用英文；配置文件的注释和中文文档照本 plan 原样。每条注释和说明在它所在的 Task 结束时都与代码相符（spec 附录 A 的清扫 59）。
- **代码块**：每个改动都写成四个反引号围起来的块，块的第一行写明种类和路径，照原样使用（原型中逐字节运行过）：
  - ````` ````file <路径> ````` 新文件，块的内容加一个结尾换行就是整个文件；
  - ````` ````whole <路径> ````` 已有文件的完整新内容（同样加结尾换行）；
  - ````` ````old <路径> ````` 与紧跟着的 ````` ````new <路径> `````：`old` 的文字在文件当前版本中恰好出现一次，把它换成 `new` 的文字；
  - ````` ````delete <路径> ````` 删除这个文件（块是空的）。

  一个文件的几个块按出现的顺序依次应用。拼 plan 的脚本已从基点起按顺序核对过全部块：每个 `old` 恰好出现一次（在它之前的块应用之后的文件中），每个新文件原来不存在，逐 Task 应用之后的文件与原型逐字节相同；之后在另一个副本上从基点逐 Task 复现，每个 Task 之后跑它的门禁，最后的树与原型逐个文件相同（spec 附录 A）。块写入之后不必再格式化（原型中每个 Task 都经过 `gofmt`、`oxfmt`）。
- **plan 是"照执行的"**（P1b 的裁定）：执行中的每个修复轮在提交它的同时改本 plan 里它的 Task 的块，plan 始终从基点复现到 HEAD。一个 Task 的块由**一条命令**从两个固定的树重新生成：`python3 $M4TMP/p2tools/taskblocks.py <n> <之前的树> <之后的树>`（树是目录，或 `git archive -o <文件>.tar <提交>` 得到的 tar，每个 git 命令单独执行；它把两个树放进 `$M4TMP/p2exec/T<n-1>`、`T<n>`，写出 `plan-blocks/T<n>.md`、`T<n>.files` 和生成物的表 `gen-T<n>.txt`，并从基点核对块，`problems: 0`）；然后 `python3 $M4TMP/p2tools/compose_plan.py <plan>` 重新拼出本 plan（变异表照结果文件写出，执行中加的变异行在修复轮里加进 `mutants_p2*.py` 和结果文件）。
- **过渡版本**：一些文件先在较早的 Task 写成过渡版本，较晚的 Task 再修改（文件结构一节的"Task"一列列出每个文件的全部 Task）：`api/modules/issue.yaml`（Task 2、4、6、7、8、11）、`issue/app/lock.go`（Task 2、5）、`issue/module.go`（Task 4、5、6、7）、`issue/app/fakes_test.go`（Task 2、4、7、10）、`issue/app/lock_test.go` 和 `bootstrap/gate_test.go`（Task 4、7）、`issue/app/update_issue.go`（Task 2、7、8、10）、`issue/domain/issue.go`（Task 2、7、10）、`issue/adapter/http/issues.go`（Task 4、6、7、11）、`access/domain/rules.go`（Task 1、2、5）、`bootstrap/issue_writes_test.go`（Task 4、5、7）、`bootstrap/permission_matrix_issue_updates_test.go`（Task 2、3、4、7、8、11）、`permission_matrix_issues_test.go`（Task 2、3、6、11）。每个过渡版本都在逐 Task 复现中运行过。
- **变异**：每个 Task 末尾的"变异"表列出：把代码改坏的方式、必须因此失败的检查和它所在的层（单元：假实现，或没有替身的平台、领域的测试；存储：真实数据库；组合：`bootstrap` 组合出的 app；端到端：单独运行的故事；守卫：生成器、文案的键的核对）。它们在各自 Task 的树上跑过，标"（Task n 起）"的检查在 Task n 的树上跑过，Go 的全部在最终的原型上再跑过，每个变异在一份新的副本里（`$M4TMP/p2tools/mutants_p2.py`，`mut.py` 在它写的每个检查上各跑一次；端到端的"发现"必须是故事的断言失败，`e2ekills.py` 核对，不算构建失败或超时；容器起不来不算；spec 附录 A）。"（Task n 起）"的检查从那个 Task 起才发现这个变异：Task n 的实现者在 Task n 的树上重跑它（那个 Task 的表下面列出）。**安全或加锁的性质只由单元一层发现的，算缺口**：表中每一条这类性质都另有存储、组合或端到端一层的检查，例外（"按性质只在单元一层"）写在 spec 附录 A.3。
- **评审敏感**（M4 设计 12 节约束 3：权限的扩展这一段）：访客规则（预检和锁下的判定各执行一次，404 优先，`OwnOnly` 按项目的角色）；`AllowCreator`（创建者仍是有效成员、角色不低于成员，锁下再判定）；`deleteIssue` 在取锁路径上，约定五和死锁（spec 第 3 节第 7 条）；归档的拒绝次序（第 5 条）；合并之后的日期（第 11 条）。改动这些代码或测试之前，先照"变异"表确认它在所说的性质去掉之后失败。
- **提交**：提交信息用英文，末尾加一行：`Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明。

## 一次性的脚本

`$M4TMP/p2tools/doclinks.py`（Task 14 的文档核对：给出的 markdown 文件中每个相对链接都指向仓库中存在的文件或目录；代码块和行内代码里的不算链接）：

```python
# M4/P2 plan, Task 14 (from M4/P1b's, from M4/P1a's, from M3's closeout plan): every relative link of the markdown files
# given resolves, from the file's folder, to a file or folder of the repository (a link's #anchor is not checked). Code is
# not prose: a fenced block's lines and an inline code span hold no links (a Go generic call, New[K](l), reads like one).
# Prints one line per file and each link that does not resolve; exits 1 when one does not.
# usage: python3 doclinks.py <root> <markdown file>...
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
| `server/internal/shared/authorize.go`（修改） | `Target.CreatorID`、`Grant.OwnOnly`；拒绝时也交回 `Grant` 的说明 | 1 |
| `server/internal/modules/access/domain/decide.go`、`server/internal/modules/access/domain/decide_test.go`（修改）、`server/internal/modules/access/domain/creator_test.go`（新） | `Project.GuestViewAll`、`Facts.Creator`、`creatorRoles`；`OwnOnly`；拒绝时交回 `Grant`；`AllowCreator` 的判定和它的测试 | 1 |
| `server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`（修改） | `Rule.AllowCreator`、`issue.delete`、`issue.archive`（Task 1）；`issue.read` 加访客（Task 2）；`issue.search`（Task 5） | 1、2、5 |
| `server/internal/modules/access/app/authorizer.go`、`server/internal/modules/access/app/authorizer_test.go`、`server/internal/modules/access/app/ports.go`（修改） | `Creator`、`GuestViewAll` 进判定的事实 | 1 |
| `server/internal/modules/project/adapter/postgres/queries/access.sql`、`server/internal/modules/project/adapter/postgres/access.go`、`server/internal/modules/project/adapter/postgres/access_test.go`、`server/internal/modules/project/app/access.go`（修改） | `ProjectFacts` 读 `guest_view_all_features` | 1 |
| `server/internal/modules/issue/domain/actions.go`（修改） | `ActionDelete`、`ActionArchive`（Task 1）；`ActionSearch`（Task 5） | 1、5 |
| `server/internal/bootstrap/ports_test.go`（修改） | `GuestViewAll` 的转换（Task 1）；假的 `IssueProjects` 回答标识（Task 5） | 1、5 |
| `tools/keywords.json`（修改） | `phase` 改为 `M4/P2` | 1 |
| `server/internal/modules/issue/domain/issue.go`、`server/internal/modules/issue/domain/issue_test.go`（修改） | `HiddenFrom`（Task 2）；`GroupCancelled`、`Archivable`（Task 7）；`StartDate`、`TargetDate`、`checkNewDates` 的接入（Task 10） | 2、7、10 |
| `server/internal/modules/issue/app/lock.go`（修改） | `decide` 带上工作项、先执行访客规则（Task 2）；`seeWorkspace`（Task 5） | 2、5 |
| `server/internal/modules/issue/app/guest_rule_test.go`（新） | 访客规则在预检和锁下的单元测试（Task 2）；删除的判定（Task 4）；回答逐值比较，隐藏的与不存在的是同一个值（Task 5） | 2、4、5 |
| `server/internal/modules/issue/app/fakes_test.go`（修改） | 判定记下目标、工作项的创建者，创建者不明的工作项（`creatorOf`）（Task 2）；删除的存储（Task 4）；归档的存储（Task 7）；日期（Task 10） | 2、4、7、10 |
| `server/internal/modules/issue/app/get_issue.go`（修改） | 说明写上访客规则 | 2 |
| `server/internal/modules/issue/app/update_issue.go`（修改） | 说明写上访客规则（Task 2）；`issueState`：工作项的状态，不是项目的是错误，`stateOf` 改成它，归档共用（Task 7）；已归档 409（Task 8）；合并之后的日期、写日期（Task 10） | 2、7、8、10 |
| `api/modules/issue.yaml`（修改） | `getIssue`、`updateIssue` 的访客规则（Task 2）；`deleteIssue`（Task 4）；`getIssueByIdentifier`、`Slug`、`IssueIdentifier`（Task 6）；`archiveIssue`、`unarchiveIssue`（Task 7）；`issue.archived`（Task 8）；三个模式的日期（Task 11） | 2、4、6、7、8、11 |
| `api/modules/project.yaml`（修改） | `guest_view_all_features` 的说明（Task 2）；`deleteState` 的补救（Task 8） | 2、8 |
| `api/openapi.yaml`（修改） | 新的路径 | 6、7 |
| `server/internal/modules/issue/adapter/http/issues_test.go`（修改） | `getIssue` 的访客的拒绝（Task 2）；创建的日期、体的形状（Task 11） | 2、11 |
| `server/internal/bootstrap/issue_interleaving_test.go`（修改） | 交错 5 的访客一支（Task 2）；删除他自己的、`wroteNothing` 数未删除的行（Task 12） | 2、12 |
| `server/internal/bootstrap/issue_world_test.go`（修改） | `newWorkItemBy`：别的账户建的工作项，交错用的 `bobsWorkItem`、`waltsWorkItem`（Task 2）；`doneWorkItem`：Done 里的工作项（Task 7） | 2、7 |
| `server/internal/bootstrap/permission_matrix_issue_columns_test.go`（新） | 工作项表的列、各列的工作项、`onAWorkItem`、`onTheIssues`（Task 3）；按编号取的格子（Task 6） | 3、6 |
| `server/internal/bootstrap/permission_matrix_seed_issues_test.go`（修改） | `ownIssues`、`guestViewAll`、`seededItems`、`numberOf`、`nextNumber`，种子和它的核对 | 3 |
| `server/internal/bootstrap/permission_matrix_seeded_test.go`、`server/internal/bootstrap/permission_matrix_members_test.go`、`server/internal/bootstrap/permission_matrix_invitations_test.go`、`server/internal/bootstrap/permission_matrix_workspace_test.go`（修改） | 新身份的账户、它们在别的表里的格子 | 3 |
| `server/internal/bootstrap/permission_matrix_columns_test.go`（修改） | 新身份的账户、它们在别的表里的格子（Task 3）；`noProjectTable`：矩阵的自检里出自不是项目表的列的违规（Task 6） | 3、6 |
| `server/internal/bootstrap/permission_matrix_targets_test.go`（修改） | 格子指名它的列的工作项（Task 3）；按编号取的目标（Task 6） | 3、6 |
| `server/internal/bootstrap/permission_matrix_coverage_test.go`（修改） | 工作项表的列（Task 3）；`prepare` 只给写的行（Task 7） | 3、7 |
| `server/internal/bootstrap/permission_matrix_test.go`（修改） | 种子的说明（Task 3）；`matrixRow.prepare`、归档的行（Task 7） | 3、7 |
| `server/internal/bootstrap/permission_matrix_issues_test.go`（修改） | 访客读得到（Task 2）；换到工作项表（Task 3）；`getIssueByIdentifier`（Task 6）；创建带日期（Task 11） | 2、3、6、11 |
| `server/internal/bootstrap/permission_matrix_issue_updates_test.go`（修改） | 修改的访客格子（Task 2）；换到工作项表（Task 3）；`deleteIssue`（Task 4）；`answered` 的说明：最后修改者由 `change` 另说，归档的矩阵用它（Task 7）；已归档的工作项（Task 8）；日期（Task 11） | 2、3、4、7、8、11 |
| `server/internal/modules/issue/app/delete_issue.go`（新） | `IssueDeletion`、`IssueRowsDeleter`、`issueRowSteps`、`DeleteIssue` | 4 |
| `server/internal/modules/issue/app/delete_issue_test.go`（新） | `DeleteIssue` 的单元测试（Task 4）；也删除已归档的工作项（Task 8） | 4、8 |
| `server/internal/modules/issue/app/update_issue_test.go`（修改） | `lockedAs`：锁下判定之前的调用，删除共用（Task 4）；已归档的拒绝（Task 8）；已归档先于日期的次序，什么都不改的修改给出原样的日期（Task 11） | 4、8、11 |
| `server/internal/modules/issue/adapter/postgres/queries/delete_issue.sql`、`server/internal/modules/issue/adapter/postgres/delete_issue.go`、`server/internal/modules/issue/adapter/postgres/delete_issue_test.go`（新） | 删除工作项的五条语句和存储测试 | 4 |
| `server/internal/modules/issue/adapter/http/handler.go`（修改） | `DeleteIssueUseCase`（Task 4）；`GetIssueByIdentifierUseCase`（Task 5）；`ArchiveIssueUseCase`（Task 7） | 4、5、7 |
| `server/internal/modules/issue/adapter/http/issues.go`（修改） | `DeleteIssue`（Task 4）；`GetIssueByIdentifier`（Task 6）；`ArchiveIssue`、`UnarchiveIssue`（Task 7）；日期、`given`（Task 11） | 4、6、7、11 |
| `server/internal/modules/issue/adapter/http/handler_test.go`（修改） | 测试的用例集 | 4、6 |
| `server/internal/modules/issue/adapter/http/delete_test.go`（新） | `TestDeleteIssue`、`TestDeleteIssueRefusals` | 4 |
| `server/internal/modules/issue/module.go`（修改） | `DeleteIssue`（Task 4）；`Workspaces`、`GetIssueByIdentifier`（Task 5、6）；`ArchiveIssue`、`UnarchiveIssue`（Task 7） | 4、5、6、7 |
| `server/internal/bootstrap/issue_deletion_test.go`（新） | 按目录的删除测试 | 4 |
| `server/internal/bootstrap/issue_write_locks_test.go`（修改） | `waits`、`mode`，删除一行（Task 4）；归档、恢复两行（Task 7） | 4、7 |
| `server/internal/modules/issue/app/lock_test.go`、`server/internal/bootstrap/gate_test.go`（修改） | P2 的写进项目的写闸门：单元的调用次序，组合的 503 `server_busy`；删除一行（Task 4）；归档、恢复两行（Task 7） | 4、7 |
| `server/internal/bootstrap/project_deletion_test.go`、`server/internal/bootstrap/workspace_deletion_test.go`、`server/internal/bootstrap/workspace_deletion_catalog_test.go`（修改） | 在项目之前已删除的工作项（PF.3）；`noRowLastWrittenBy` 取外键的列表；`foreignKey.deleted`、`asText` | 4 |
| `server/internal/modules/issue/domain/identifier.go`、`server/internal/modules/issue/domain/identifier_test.go`（新）、`api/testdata/identifier-fold.json`（新） | `ParseIdentifier`、`MatchIdentifier`、`foldIdentifier` 和样例 | 5 |
| `server/internal/modules/issue/domain/errors.go`（修改） | `ErrWorkspaceNotFound`（Task 5）；`ErrNotArchivable`（Task 7）；`ErrIssueArchived`（Task 8） | 5、7、8 |
| `server/internal/modules/issue/app/get_issue_by_identifier.go`、`server/internal/modules/issue/app/get_issue_by_identifier_test.go`（新） | `GetIssueByIdentifier` 和单元测试 | 5 |
| `server/internal/modules/issue/app/get_issue_test.go`（修改） | `readIssue`（两个用例共用）；拒绝的回答逐值比较 | 5 |
| `server/internal/modules/issue/app/ports.go`（修改） | `WorkspaceDirectory`、`ProjectIdentifiers`、`ProjectIdentifier`、`IssueNumbers`（Task 5）；`IssueArchiver`（Task 7） | 5、7 |
| `server/internal/modules/issue/adapter/postgres/queries/issues.sql`、`server/internal/modules/issue/adapter/postgres/issues.go`、`server/internal/modules/issue/adapter/postgres/issues_test.go`（修改） | `IssueByNumber`（Task 5）；日期的两列（Task 10） | 5、10 |
| `server/internal/modules/project/adapter/postgres/queries/issue_ports.sql`、`server/internal/modules/project/adapter/postgres/issue_ports.go`、`server/internal/modules/project/adapter/postgres/issue_ports_test.go`、`server/internal/modules/project/adapter/postgres/failures_test.go`（修改） | `ProjectIdentifiers` 和存储测试 | 5 |
| `server/internal/modules/project/app/issue_ports.go`、`server/internal/modules/project/module.go`（修改） | `IssueProjectIdentifier`、`IssueProjects.ProjectIdentifiers` | 5 |
| `server/internal/modules/workspace/module.go`（修改） | `WorkspaceDirectory` 的说明写上 `issue` | 5 |
| `server/internal/bootstrap/ports.go`、`server/internal/bootstrap/app.go`（修改）、`server/internal/bootstrap/issue_ports_test.go`（新） | `issueProjects.Identifiers`、`issueWorkspaces` 和转换的测试 | 5 |
| `server/internal/bootstrap/issue_writes_test.go`（修改） | 一个连接的池上删除第二个工作项（Task 4）；世界的 `issue.New` 带上 `Workspaces`（Task 5）；一个连接的池上归档、恢复（Task 7） | 4、5、7 |
| `server/internal/modules/issue/adapter/http/by_identifier_test.go`（新）、`server/internal/bootstrap/issue_identifier_test.go`（新） | 按编号取的 HTTP 和组合的测试 | 6 |
| `server/internal/modules/issue/domain/activity.go`、`server/internal/modules/issue/domain/activity_test.go`（修改） | `ArchiveActivity`、`UnarchiveActivity`（Task 7）；第 8、9 行（Task 10） | 7、10 |
| `server/internal/modules/issue/app/archive_issue.go`、`server/internal/modules/issue/app/archive_issue_test.go`（新） | `ArchiveIssue`、`UnarchiveIssue` 和单元测试 | 7 |
| `server/internal/modules/issue/app/rows.go`（修改） | `IssueArchival`（Task 7）；`IssueRow`、`IssueFields` 的日期（Task 10） | 7、10 |
| `server/internal/modules/issue/adapter/postgres/queries/update.sql`、`server/internal/modules/issue/adapter/postgres/update.go`、`server/internal/modules/issue/adapter/postgres/update_test.go`（修改） | `SetArchivedAt`（Task 7）；修改写日期（Task 10） | 7、10 |
| `server/internal/modules/issue/adapter/http/archive_test.go`（新） | 归档、恢复的 HTTP 测试 | 7 |
| `server/internal/bootstrap/issue_archive_test.go`（新）、`server/internal/bootstrap/permission_matrix_issue_archive_test.go`（新） | 归档的盖戳；矩阵的归档、恢复的行 | 7 |
| `server/internal/bootstrap/issue_updates_test.go`（修改） | 回答如存的工作项的归档时刻的说明（Task 7）；日期（Task 11） | 7、11 |
| `web/apps/web/core/lib/error-messages.ts`、`web/packages/i18n/src/locales/en/errors.json`、`web/packages/i18n/src/locales/zh-CN/errors.json`（修改） | `issue.not_archivable`（Task 7）；`issue.archived`、`project.state_in_use` 的补救（Task 8） | 7、8 |
| `server/internal/modules/issue/adapter/http/update_test.go`（修改） | `issue.archived` 的拒绝（Task 8）；修改的日期、体的形状（Task 11） | 8、11 |
| `server/internal/modules/project/domain/errors.go`、`server/internal/modules/project/adapter/http/state_writes_test.go`（修改） | `project.state_in_use` 的补救 | 8 |
| `server/internal/bootstrap/issue_cascade_test.go`（修改） | `deleteLabel` 的组合测试：已归档的工作项的标签行照样删除 | 8 |
| `server/internal/shared/date.go`、`server/internal/shared/date_test.go`（新） | `shared.Date` | 9 |
| `server/internal/platform/httpserver/bodyshape/bodyshape.go`（修改） | `FormatDate`、`checkDate`、开放节点的数，路径的代码移出（Task 9）；`checkDate` 的说明（Task 11） | 9、11 |
| `server/internal/platform/httpserver/bodyshape/paths.go`、`server/internal/platform/httpserver/bodyshape/paths_test.go`（新） | `problems` 和路径；`comparePaths` | 9 |
| `server/internal/platform/httpserver/bodyshape/ambiguity.go`、`server/internal/platform/httpserver/bodyshape/bodyshape_test.go`、`server/internal/platform/httpserver/bodyshape/cost_test.go`（修改） | `numbers`；测试和代价测试 | 9 |
| `server/tools/bodyshapegen/generate.go`、`server/tools/bodyshapegen/generate_test.go`、`server/tools/bodyshapegen/testdata/oapi-codegen.yaml`、`server/tools/bodyshapegen/testdata/things.yaml`、`server/tools/bodyshapegen/testdata/things.golden`（修改） | `shared.Date` 的检查器和测试数据 | 9 |
| `server/internal/modules/identity/adapter/http/gen/oapi-codegen.yaml`、`server/internal/modules/instance/adapter/http/gen/oapi-codegen.yaml`、`server/internal/modules/issue/adapter/http/gen/oapi-codegen.yaml`、`server/internal/modules/project/adapter/http/gen/oapi-codegen.yaml`、`server/internal/modules/workspace/adapter/http/gen/oapi-codegen.yaml`（修改） | `format: date` → `shared.Date` | 9 |
| `server/internal/platform/httpserver/apitest/operations.go`、`server/internal/platform/httpserver/apitest/bodycases_test.go`（修改） | `date` 是检查的格式 | 9 |
| `server/internal/modules/issue/domain/dates.go`、`server/internal/modules/issue/domain/dates_test.go`（新） | `DateChange`、`checkNewDates`、`DatesOutOfOrder`、`sameDay`、`dayText` | 10 |
| `server/internal/modules/issue/domain/patch.go`、`server/internal/modules/issue/domain/patch_test.go`（修改） | `IssuePatch`、`Changes` 的日期 | 10 |
| `server/internal/modules/issue/app/create_issue.go`、`server/internal/modules/issue/app/create_issue_test.go`（修改）、`server/internal/modules/issue/app/dates_test.go`（新） | 创建、修改写日期，合并之后的检查 | 10 |
| `server/sqlc.yaml`（修改） | `issue` 的 `date` → `time.Time` | 10 |
| `server/internal/modules/issue/adapter/http/dates_test.go`（新）、`server/internal/bootstrap/issue_dates_test.go`（新） | 体的检查与解码一致；组合的日期 | 11 |
| `server/internal/bootstrap/issue_interleaving_archive_test.go`（新）、`server/internal/bootstrap/issue_interleaving_deletion_test.go`（修改） | 交错 12；删除与修改 | 12 |
| `e2e/fixtures/issues.ts`、`e2e/fixtures/assert/issue.ts`（修改） | `issueArchived`、`issueUnarchived`、`issueDeleted`；`archivedRow`、`restoredRow`、`expectIssueArchival`、`expectIssueDeleted`，创建和修改的日期 | 13 |
| `e2e/stories/issue/i15-archive-issue.spec.ts`（新）、`e2e/stories/issue/i1-create-issue.spec.ts`、`e2e/stories/issue/i2-update-issue.spec.ts`（修改） | I15 的接口版本；I1、I2 的日期；I2 的已归档 409 | 13 |
| `docs/v0/v0-design.md`、`docs/v0/M2-auth/M2-design.md`、`docs/v0/plane-diff.md`、`docs/v0/M4-issue-core/handoffs/M1-P3-trim-platform.md`、`docs/v0/M4-issue-core/handoffs/M2-closeout.md`、`docs/v0/M4-issue-core/handoffs/M3-closeout.md`（修改） | 3.23 中 P2 的行；交接的处理结果 | 14 |

---

### Task 1: 规则的扩展：创建者、`OwnOnly`、`guest_view_all_features`

**Files:**
- Create: `server/internal/modules/access/domain/creator_test.go`
- Modify: `server/internal/bootstrap/ports_test.go`、`server/internal/modules/access/app/authorizer.go`、`server/internal/modules/access/app/authorizer_test.go`、`server/internal/modules/access/app/ports.go`、`server/internal/modules/access/domain/decide.go`、`server/internal/modules/access/domain/decide_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/issue/domain/actions.go`、`server/internal/modules/project/adapter/postgres/access.go`、`server/internal/modules/project/adapter/postgres/access_test.go`、`server/internal/modules/project/adapter/postgres/queries/access.sql`、`server/internal/modules/project/app/access.go`、`server/internal/shared/authorize.go`、`tools/keywords.json`
- Generate: `server/internal/modules/project/adapter/postgres/gen/access.sql.go`

**Interfaces:**（spec 2.3、第 3 节第 2 条；M4 设计 3.4、6.5）
- `shared/authorize.go`：`Target.CreatorID uuid.UUID`（目标工作项的创建者，别的动作是零值）；`Grant.OwnOnly bool`（项目角色是访客、项目的 `guest_view_all_features` 为假）；`Authorizer` 的说明：角色不允许时回答 `shared.Forbidden` 和这次判定读到的 `Grant`。
- `access/domain/decide.go`：`Project{Public, Member, GuestViewAll}`、`Facts{Workspace, Project, Creator}`；`creatorRoles`（管理员、成员）；`decideProject` 先算出 `Grant`（`OwnOnly` 按项目的角色），再判定，拒绝时交回它；`decideWorkspace` 同样；规则带 `AllowCreator`、`Facts.Creator`、调用者是项目的有效成员、项目角色在 `creatorRoles` 里时允许。
- `access/domain/rules.go`：`Rule.AllowCreator`；`issue.delete`（项目级，管理员，`AllowCreator`）、`issue.archive`（项目级，管理员、成员）。`issue/domain/actions.go`：`ActionDelete`、`ActionArchive`，`Actions()` 带上它们。
- `access/app`：`ports.go` 的 `ProjectFacts.GuestViewAll`；`authorizer.go`：`Creator: t.CreatorID == actor.UserID`，`GuestViewAll` 进 `domain.Project`。
- `project`：`queries/access.sql` 多读 `p.guest_view_all_features`；`adapter/postgres/access.go`、`app/access.go` 带上它；`bootstrap/ports_test.go` 的转换。
- `tools/keywords.json`：`phase` 改为 `M4/P2`。

**Tests:**
- 单元：`TestEveryRuleDecidesItsCells`（每一行逐格，对没建和建了目标工作项的调用者各一遍：带 `AllowCreator` 的行用它的 `creatorCells`）、`TestTheGrantCarriesTheRoles`（`OwnOnly` 按项目角色）、`TestARefusalCarriesTheGrant`、`TestAllowCreatorAllowsAMemberWhoCreatedIt`（`creator_test.go`）、`TestAuthorizeKnowsTheCreator`、`TestAuthorizeReadsTheTargetsProject`、`bootstrap` 的 `TestAccessProjectsConvertsProjectsAnswer`。
- 存储：`TestProjectFacts`（读出标志：Web 让访客看每个工作项，别的项目不让）。
- 完整性：`TestEveryActionHasARuleAndEveryRuleAnAction`、`TestEveryModuleDeclaresItsActionsOrHasNone`。

- [ ] **Step 1: 项目的事实读标志，生成**

`server/internal/modules/project/adapter/postgres/access.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/access.go
	f = app.AccessFacts{WorkspaceID: r.WorkspaceID, Public: domain.Network(r.Network) == domain.NetworkPublic}
````
````new server/internal/modules/project/adapter/postgres/access.go
	f = app.AccessFacts{WorkspaceID: r.WorkspaceID, Public: domain.Network(r.Network) == domain.NetworkPublic,
		GuestViewAll: r.GuestViewAllFeatures}
````

`server/internal/modules/project/adapter/postgres/access_test.go`（修改，4 处）：

````old server/internal/modules/project/adapter/postgres/access_test.go
// ProjectFacts reads the undeleted project, archived or not, and the user's
// membership of it while it is active: a membership ended or deleted is
// none, and so is an active membership of another project; a project
// deleted is not found. Each case is one fact changed from a project where
// the user is its member.
````
````new server/internal/modules/project/adapter/postgres/access_test.go
// ProjectFacts reads the undeleted project, archived or not, whether it
// shows its guests every work item, and the user's membership of it while
// it is active: a membership ended or deleted is none, and so is an active
// membership of another project; a project deleted is not found. Each case
// is one fact changed from a project where the user is its member. Web
// shows its guests every work item, the others do not.
````
````old server/internal/modules/project/adapter/postgres/access_test.go
	public := newProject(t, s, acme, "Web", "WEB", alice)
````
````new server/internal/modules/project/adapter/postgres/access_test.go
	public := newProject(t, s, acme, "Web", "WEB", alice)
	exec(t, pool, "UPDATE projects SET guest_view_all_features = true WHERE id = $1", public)
````
````old server/internal/modules/project/adapter/postgres/access_test.go
		{"a public project's guest", public, alice, app.AccessFacts{WorkspaceID: acme, Public: true, Member: true, Role: shared.RoleGuest}, true},
````
````new server/internal/modules/project/adapter/postgres/access_test.go
		{"a public project's guest", public, alice, app.AccessFacts{WorkspaceID: acme, Public: true, Member: true, Role: shared.RoleGuest,
			GuestViewAll: true}, true},
````
````old server/internal/modules/project/adapter/postgres/access_test.go
		{"a membership ended", public, bob, app.AccessFacts{WorkspaceID: acme, Public: true}, true},
````
````new server/internal/modules/project/adapter/postgres/access_test.go
		{"a membership ended", public, bob, app.AccessFacts{WorkspaceID: acme, Public: true, GuestViewAll: true}, true},
````

`server/internal/modules/project/adapter/postgres/queries/access.sql`（整个文件，8 行）：

````whole server/internal/modules/project/adapter/postgres/queries/access.sql
-- name: ProjectFacts :one
-- access's ProjectAccess (M3 design 6.5): the undeleted project, archived or not, whether it shows its guests every
-- work item (M4 design 3.4), and the user's active membership of it, if any.
SELECT p.workspace_id, p.network, p.guest_view_all_features, m.role AS member_role
FROM projects p
LEFT JOIN project_members m
       ON m.project_id = p.id AND m.member_id = sqlc.arg(user_id) AND m.is_active AND m.deleted_at IS NULL
WHERE p.id = sqlc.arg(project_id) AND p.deleted_at IS NULL;
````

`server/internal/modules/project/app/access.go`（整个文件，21 行）：

````whole server/internal/modules/project/app/access.go
package app

import (
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// AccessFacts are what project.Provide's ProjectAccess answers the access
// module (M3 design 6.5): the project's workspace, whether it is public,
// whether the user asked about is its active member, with his project role
// then, and whether the project shows its guests every work item
// (guest_view_all_features, M4 design 3.4). bootstrap converts them into
// access's value.
type AccessFacts struct {
	WorkspaceID  uuid.UUID
	Public       bool
	Member       bool
	Role         shared.Role
	GuestViewAll bool
}
````

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `5f7076830007c54589ea46600c8284dbd4f3025a5e4f73544b2266dc942b1202` | 46 | `server/internal/modules/project/adapter/postgres/gen/access.sql.go` |

Run: `shasum -a 256 server/internal/modules/project/adapter/postgres/gen/access.sql.go`
Expected: 与上表相同。

- [ ] **Step 2: 判定**

`server/internal/modules/access/app/authorizer.go`（修改，3 处）：

````old server/internal/modules/access/app/authorizer.go
// t's, is none, and a project-level rule sees nothing (M3 design 3.4).
````
````new server/internal/modules/access/app/authorizer.go
// t's, is none, and a project-level rule sees nothing (M3 design 3.4). The
// caller is the creator when t names him (M4 design 3.4).
````
````old server/internal/modules/access/app/authorizer.go
	facts := domain.Facts{Workspace: domain.Membership{Active: active, Role: role}}
````
````new server/internal/modules/access/app/authorizer.go
	facts := domain.Facts{
		Workspace: domain.Membership{Active: active, Role: role},
		Creator:   t.CreatorID == actor.UserID,
	}
````
````old server/internal/modules/access/app/authorizer.go
			facts.Project = &domain.Project{Public: p.Public, Member: domain.Membership{Active: p.Member, Role: p.Role}}
````
````new server/internal/modules/access/app/authorizer.go
			facts.Project = &domain.Project{Public: p.Public, Member: domain.Membership{Active: p.Member, Role: p.Role},
				GuestViewAll: p.GuestViewAll}
````

`server/internal/modules/access/app/authorizer_test.go`（修改，5 处）：

````old server/internal/modules/access/app/authorizer_test.go
// caller's project role is in the Grant, a workspace admin sees a private
// project he is not in. A project of another workspace than the target's,
// or one not found, is seen by no one: its facts do not count.
````
````new server/internal/modules/access/app/authorizer_test.go
// caller's project role is in the Grant, and whether the project shows its
// guests every work item (OwnOnly, M4 design 3.4); a workspace admin sees a
// private project he is not in. A project of another workspace than the
// target's, or one not found, is seen by no one: its facts do not count.
````
````old server/internal/modules/access/app/authorizer_test.go
	p1, p2, gone := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
````
````new server/internal/modules/access/app/authorizer_test.go
	p1, p2, p3, gone := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
````
````old server/internal/modules/access/app/authorizer_test.go
		{p2, a}: {WorkspaceID: w2, Public: true, Member: true, Role: shared.RoleAdmin},
````
````new server/internal/modules/access/app/authorizer_test.go
		{p2, a}: {WorkspaceID: w2, Public: true, Member: true, Role: shared.RoleAdmin},
		{p3, a}: {WorkspaceID: w1, Member: true, Role: shared.RoleGuest, GuestViewAll: true},
````
````old server/internal/modules/access/app/authorizer_test.go
		{a, p1, &shared.Grant{WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleGuest}},
		{b, p1, &shared.Grant{WorkspaceRole: shared.RoleAdmin}},
````
````new server/internal/modules/access/app/authorizer_test.go
		{a, p1, &shared.Grant{WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleGuest, OwnOnly: true}},
		{b, p1, &shared.Grant{WorkspaceRole: shared.RoleAdmin}},
		{a, p3, &shared.Grant{WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleGuest}},
````
````old server/internal/modules/access/app/authorizer_test.go
			t.Errorf("user %s, project %s: Authorize() = %+v, %v; want %+v", tt.user, tt.project, grant, err, *tt.want)
		}
	}
}

````
````new server/internal/modules/access/app/authorizer_test.go
			t.Errorf("user %s, project %s: Authorize() = %+v, %v; want %+v", tt.user, tt.project, grant, err, *tt.want)
		}
	}
}

// The caller created the target's work item when its CreatorID is his
// account (M4 design 3.4): issue.delete, the project's admins and the
// creator, allows the project's member who created it and refuses him a
// work item another created, or one whose creator is not known. Each answer,
// a refusal too, hands on the Grant the decision read.
func TestAuthorizeKnowsTheCreator(t *testing.T) {
	p := uuid.NewV7()
	roles := &fakeRoles{roles: map[membership]shared.Role{{w1, a}: shared.RoleMember}}
	projects := &fakeProjects{facts: map[projectKey]app.ProjectFacts{{p, a}: {WorkspaceID: w1, Member: true, Role: shared.RoleMember}}}
	auth := app.NewAuthorizer(roles, projects)
	grant := shared.Grant{WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleMember}
	for _, tt := range []struct {
		name    string
		creator uuid.UUID
		want    error
	}{
		{"his own", a, nil},
		{"another's", b, shared.Forbidden()},
		{"a creator not known", uuid.UUID{}, shared.Forbidden()},
	} {
		got, err := auth.Authorize(context.Background(), shared.Actor{UserID: a}, "issue.delete",
			shared.Target{WorkspaceID: w1, ProjectID: p, CreatorID: tt.creator})
		if (tt.want == nil) != (err == nil) || (tt.want != nil && !errors.Is(err, tt.want)) || got != grant {
			t.Errorf("%s: Authorize() = %+v, %v; want %+v, %v", tt.name, got, err, grant, tt.want)
		}
	}
}

````

`server/internal/modules/access/app/ports.go`（修改，2 处）：

````old server/internal/modules/access/app/ports.go
// its workspace, whether it is public, and whether the user asked about is
// its active member, with his project role then.
````
````new server/internal/modules/access/app/ports.go
// its workspace, whether it is public, whether the user asked about is its
// active member, with his project role then, and whether it shows its
// guests every work item (guest_view_all_features, M4 design 3.4).
````
````old server/internal/modules/access/app/ports.go
	WorkspaceID uuid.UUID
	Public      bool
	Member      bool
	Role        shared.Role
````
````new server/internal/modules/access/app/ports.go
	WorkspaceID  uuid.UUID
	Public       bool
	Member       bool
	Role         shared.Role
	GuestViewAll bool
````

`server/internal/modules/access/domain/creator_test.go`（新文件，50 行）：

````file server/internal/modules/access/domain/creator_test.go
package domain_test

import (
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/access/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// AllowCreator lets a project-level rule allow the work item's creator too,
// while he is an active member of the project of the role member or admin
// (M4 design 3.4): not as its guest, the project's flag for guests
// whichever, not once his membership has ended, and not with a project role
// outside the three. A caller who did not create it is decided by his role
// alone, and a rule without AllowCreator does not look at the creator.
func TestAllowCreatorAllowsAMemberWhoCreatedIt(t *testing.T) {
	admins := domain.Rule{Level: domain.LevelProject, Roles: []shared.Role{shared.RoleAdmin}}
	byCreator := domain.Rule{Level: domain.LevelProject, Roles: []shared.Role{shared.RoleAdmin}, AllowCreator: true}
	left := domain.Membership{Active: false, Role: shared.RoleMember}
	tests := []struct {
		name    string
		rule    domain.Rule
		ws      domain.Membership
		p       domain.Project
		creator bool
		want    outcome
	}{
		{"a member who created it", byCreator, member, domain.Project{Member: member}, true, allowed},
		{"a member who did not", byCreator, member, domain.Project{Member: member}, false, forbidden},
		{"a member who created it, by a rule without AllowCreator", admins, member, domain.Project{Member: member}, true, forbidden},
		{"the project's admin who did not create it", byCreator, member, domain.Project{Member: admin}, false, allowed},
		{"the project's admin who created it, by a rule of its creator alone", domain.Rule{Level: domain.LevelProject, AllowCreator: true},
			member, domain.Project{Member: admin}, true, allowed},
		{"a guest who created it", byCreator, guest, domain.Project{Member: guest}, true, forbidden},
		{"a guest who created it, the project showing its guests every work item", byCreator, guest,
			domain.Project{Member: guest, GuestViewAll: true}, true, forbidden},
		{"a member demoted to the project's guest who created it", byCreator, member, domain.Project{Member: guest}, true, forbidden},
		{"a creator who left the private project", byCreator, member, domain.Project{Member: left}, true, invisible},
		{"a creator who left the public project", byCreator, member, domain.Project{Public: true, Member: left}, true, forbidden},
		{"a creator whose project role is outside the three", byCreator, member,
			domain.Project{Member: domain.Membership{Active: true, Role: 10}}, true, forbidden},
	}
	for _, tt := range tests {
		p := tt.p
		_, err := domain.Decide(tt.rule, domain.Facts{Workspace: tt.ws, Project: &p, Creator: tt.creator})
		if got := outcomeOf(t, err); got != tt.want {
			t.Errorf("%s: Decide() = %s, want %s", tt.name, got, tt.want)
		}
	}
}
````

`server/internal/modules/access/domain/decide.go`（整个文件，114 行）：

````whole server/internal/modules/access/domain/decide.go
package domain

import (
	"fmt"
	"slices"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Membership is the caller's membership of a workspace or a project as a
// port read it. Active is false when there is none that counts: no row,
// is_active false, the row deleted, or the workspace or project deleted.
type Membership struct {
	Active bool
	Role   shared.Role
}

// Project is what the ports read about the target project: whether it is
// public, the caller's membership of it, and whether it shows its guests
// every work item (guest_view_all_features, M4 design 3.4).
type Project struct {
	Public       bool // network = 2
	Member       Membership
	GuestViewAll bool
}

// Facts are what the ports read about the caller and the target (M3 design
// 3.4). Project is nil when the target has no project that counts: a
// workspace-level target, or a project that does not exist, is deleted or
// belongs to another workspace. An archived project counts (M3 design 3.19).
// Creator is true when the caller created the work item the target names
// (shared.Target.CreatorID, M4 design 3.4).
type Facts struct {
	Workspace Membership
	Project   *Project
	Creator   bool
}

// Decide applies rule to f (M3 design 3.4): the Grant when the caller may
// act, shared.ErrNotVisible when the caller cannot see the target, and
// shared.Forbidden when the caller sees it but the rule does not allow the
// caller's role, with the Grant of what it read, so that a use case can
// answer first that the guest rule hides the target from him (M4 design 3.4).
// A caller who is not an active member of the workspace sees nothing in it.
// Roles are compared by membership in a set, never by order: a role outside
// the three is allowed nothing and sees no project it is not a member of. A
// rule of no known level is an error: nothing is allowed.
func Decide(rule Rule, f Facts) (shared.Grant, error) {
	switch rule.Level {
	case LevelWorkspace:
		return decideWorkspace(rule, f.Workspace)
	case LevelProject, LevelVisible:
		return decideProject(rule, f)
	}
	return shared.Grant{}, fmt.Errorf("access: a rule of unknown level %d", rule.Level)
}

func decideWorkspace(rule Rule, ws Membership) (shared.Grant, error) {
	if !ws.Active {
		return shared.Grant{}, shared.ErrNotVisible
	}
	grant := shared.Grant{WorkspaceRole: ws.Role}
	if !slices.Contains(rule.Roles, ws.Role) {
		return grant, shared.Forbidden()
	}
	return grant, nil
}

// knownRoles are the three roles; any other is allowed nothing.
var knownRoles = []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}

// creatorRoles are the project roles of a creator AllowCreator allows: an
// active member of the project, of the role member or above, by set (M4
// design 3.4).
var creatorRoles = []shared.Role{shared.RoleAdmin, shared.RoleMember}

func decideProject(rule Rule, f Facts) (shared.Grant, error) {
	ws, p := f.Workspace, f.Project
	if !ws.Active || p == nil || !sees(ws.Role, *p) {
		return shared.Grant{}, shared.ErrNotVisible
	}
	grant := shared.Grant{WorkspaceRole: ws.Role}
	if p.Member.Active {
		grant.ProjectRole = p.Member.Role
		grant.ProjectAdmin = p.Member.Role == shared.RoleAdmin || ws.Role == shared.RoleAdmin
		// By the project role alone: the workspace's admin who is only a
		// guest in the project sees only his own work items too (M4 design
		// 3.4).
		grant.OwnOnly = p.Member.Role == shared.RoleGuest && !p.GuestViewAll
	}
	// A role outside the three, the workspace's or the project's, is allowed
	// nothing, as at the workspace level.
	if !slices.Contains(knownRoles, ws.Role) || (p.Member.Active && !slices.Contains(knownRoles, p.Member.Role)) {
		return grant, shared.Forbidden()
	}
	if rule.Level == LevelVisible {
		return grant, nil
	}
	if p.Member.Active && (slices.Contains(rule.Roles, p.Member.Role) || ws.Role == shared.RoleAdmin ||
		rule.AllowCreator && f.Creator && slices.Contains(creatorRoles, p.Member.Role)) {
		return grant, nil
	}
	return grant, shared.Forbidden()
}

// sees reports whether an active workspace member with role wsRole sees p:
// the workspace's admin sees every project, an active project member sees
// the project, and a workspace member or admin sees a public project. A
// guest sees only the projects he is a member of, public or not (Plane
// views/project/base.py:198-222).
func sees(wsRole shared.Role, p Project) bool {
	return wsRole == shared.RoleAdmin || p.Member.Active ||
		(p.Public && slices.Contains([]shared.Role{shared.RoleMember, shared.RoleAdmin}, wsRole))
}
````

`server/internal/modules/access/domain/decide_test.go`（修改，4 处）：

````old server/internal/modules/access/domain/decide_test.go
// The Grant carries the roles the decision read, and ProjectAdmin for a
// project member who is its admin or the workspace's; a membership that is
// not active gives no project role. It is the same Grant at every level that
// allows the caller: LevelVisible, and for a project member LevelProject,
// by his project role or as the workspace's admin.
````
````new server/internal/modules/access/domain/decide_test.go
// The Grant carries the roles the decision read, ProjectAdmin for a project
// member who is its admin or the workspace's, and OwnOnly for the project's
// guest in a project that does not show its guests every work item, by his
// project role whatever his workspace role (M4 design 3.4); a membership
// that is not active gives no project role. It is the same Grant at every
// level that allows the caller: LevelVisible, and for a project member
// LevelProject, by his project role or as the workspace's admin.
````
````old server/internal/modules/access/domain/decide_test.go
			shared.Grant{WorkspaceRole: 5, ProjectRole: 5}},
````
````new server/internal/modules/access/domain/decide_test.go
			shared.Grant{WorkspaceRole: 5, ProjectRole: 5, OwnOnly: true}},
		{"PG, the project showing its guests every work item", guest, domain.Project{Member: guest, GuestViewAll: true},
			[]domain.Rule{visible, everyRole}, shared.Grant{WorkspaceRole: 5, ProjectRole: 5}},
		{"WM demoted to PG", member, domain.Project{Member: guest}, []domain.Rule{visible, everyRole},
			shared.Grant{WorkspaceRole: 15, ProjectRole: 5, OwnOnly: true}},
		// Every rule allows the workspace's admin who is the project's
		// guest, and he still sees only his own work items.
		{"PG+WA", admin, domain.Project{Member: guest}, []domain.Rule{visible, everyRole, admins},
			shared.Grant{WorkspaceRole: 20, ProjectRole: 5, ProjectAdmin: true, OwnOnly: true}},
		{"PG+WA, the project showing its guests every work item", admin, domain.Project{Member: guest, GuestViewAll: true},
			[]domain.Rule{visible, everyRole, admins}, shared.Grant{WorkspaceRole: 20, ProjectRole: 5, ProjectAdmin: true}},
		{"PA, the project showing its guests every work item", member, domain.Project{Member: admin, GuestViewAll: true},
			[]domain.Rule{visible, everyRole, admins}, shared.Grant{WorkspaceRole: 15, ProjectRole: 20, ProjectAdmin: true}},
````
````old server/internal/modules/access/domain/decide_test.go
		{"P-before public", member, domain.Project{Public: true, Member: domain.Membership{Active: false, Role: shared.RoleAdmin}}, []domain.Rule{visible},
			shared.Grant{WorkspaceRole: 15}},
````
````new server/internal/modules/access/domain/decide_test.go
		{"P-before public", member, domain.Project{Public: true, Member: domain.Membership{Active: false, Role: shared.RoleAdmin}}, []domain.Rule{visible},
			shared.Grant{WorkspaceRole: 15}},
		{"a guest who left the public project", member, domain.Project{Public: true, Member: domain.Membership{Active: false, Role: shared.RoleGuest}},
			[]domain.Rule{visible}, shared.Grant{WorkspaceRole: 15}},
````
````old server/internal/modules/access/domain/decide_test.go
				t.Errorf("%s, level %d, roles %v: Decide() = %+v, %v; want %+v", tt.name, rule.Level, rule.Roles, grant, err, tt.want)
			}
````
````new server/internal/modules/access/domain/decide_test.go
				t.Errorf("%s, level %d, roles %v: Decide() = %+v, %v; want %+v", tt.name, rule.Level, rule.Roles, grant, err, tt.want)
			}
		}
	}
}

// A refusal of the caller's role, a role outside the three too, comes with
// the Grant of what the decision read, OwnOnly among it, so a use case can
// answer first that the guest rule hides the target from him (M4 design 3.4,
// 6.7); a caller who does not see the target gets no Grant.
func TestARefusalCarriesTheGrant(t *testing.T) {
	admins := domain.Rule{Level: domain.LevelProject, Roles: []shared.Role{shared.RoleAdmin}}
	everyRole := domain.Rule{Level: domain.LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}}
	tests := []struct {
		name  string
		rule  domain.Rule
		facts domain.Facts
		want  shared.Grant
		err   outcome
	}{
		{"PG", admins, domain.Facts{Workspace: guest, Project: &domain.Project{Member: guest}},
			shared.Grant{WorkspaceRole: 5, ProjectRole: 5, OwnOnly: true}, forbidden},
		{"PG, the project showing its guests every work item", admins,
			domain.Facts{Workspace: guest, Project: &domain.Project{Member: guest, GuestViewAll: true}},
			shared.Grant{WorkspaceRole: 5, ProjectRole: 5}, forbidden},
		{"PM", admins, domain.Facts{Workspace: member, Project: &domain.Project{Member: member}},
			shared.Grant{WorkspaceRole: 15, ProjectRole: 15}, forbidden},
		{"WA- private", everyRole, domain.Facts{Workspace: admin, Project: &domain.Project{}}, shared.Grant{WorkspaceRole: 20}, forbidden},
		{"WM- public", everyRole, domain.Facts{Workspace: member, Project: &domain.Project{Public: true}}, shared.Grant{WorkspaceRole: 15},
			forbidden},
		{"a workspace role outside the three, the project's guest", everyRole,
			domain.Facts{Workspace: domain.Membership{Active: true, Role: 10}, Project: &domain.Project{Member: guest}},
			shared.Grant{WorkspaceRole: 10, ProjectRole: 5, OwnOnly: true}, forbidden},
		{"a member, at the workspace level", domain.Rule{Level: domain.LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
			domain.Facts{Workspace: member}, shared.Grant{WorkspaceRole: 15}, forbidden},
		{"WM- private", everyRole, domain.Facts{Workspace: member, Project: &domain.Project{}}, shared.Grant{}, invisible},
		{"never a member", everyRole, domain.Facts{Workspace: none, Project: &domain.Project{Public: true}}, shared.Grant{}, invisible},
	}
	for _, tt := range tests {
		grant, err := domain.Decide(tt.rule, tt.facts)
		if got := outcomeOf(t, err); got != tt.err || grant != tt.want {
			t.Errorf("%s: Decide() = %+v, %s; want %+v, %s", tt.name, grant, got, tt.want, tt.err)
````

`server/internal/modules/access/domain/rules.go`（修改，4 处）：

````old server/internal/modules/access/domain/rules.go
	// workspace's admin.
````
````new server/internal/modules/access/domain/rules.go
	// workspace's admin; and, by a rule with AllowCreator, the creator of the
	// target work item when his project role is member or admin.
````
````old server/internal/modules/access/domain/rules.go
// LevelVisible has none.
````
````new server/internal/modules/access/domain/rules.go
// LevelVisible has none. AllowCreator lets a project-level rule allow the
// creator of the target work item besides those roles, while he is an
// active member of the project of the role member or above (M4 design 3.4:
// a guest who created it is not allowed; Plane's creator=True looks at the
// workspace membership alone).
````
````old server/internal/modules/access/domain/rules.go
	Level Level
	Roles []shared.Role
````
````new server/internal/modules/access/domain/rules.go
	Level        Level
	Roles        []shared.Role
	AllowCreator bool
````
````old server/internal/modules/access/domain/rules.go
	"issue.update": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
````
````new server/internal/modules/access/domain/rules.go
	"issue.update": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
	// The project's admins, and the work item's creator while he is a member
	// or an admin of the project, not a guest who created it (M4 design 3.4).
	"issue.delete": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}, AllowCreator: true},
	// As issue.create: archiving and unarchiving a work item (M4 design 3.4,
	// 3.8).
	"issue.archive": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，6 处）：

````old server/internal/modules/access/domain/rules_test.go
	"issue.update": {allowed, allowed, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
````
````new server/internal/modules/access/domain/rules_test.go
	"issue.update": {allowed, allowed, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
	// As project.update for a caller who did not create the work item;
	// creatorCells has its creators' (M4 design 3.4).
	"issue.delete": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
	// As issue.create (M4 design 3.4).
	"issue.archive": {allowed, allowed, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
````
````old server/internal/modules/access/domain/rules_test.go
// cells decides rule for each identity of its level.
func cells(t *testing.T, rule domain.Rule) []outcome {
````
````new server/internal/modules/access/domain/rules_test.go
// creatorCells is every row with AllowCreator decided for every identity of
// projectIdentities who created the target work item (M4 design 3.4): the
// project's member who created it is allowed too, and no one else whom
// tableCells refuses: not the project's guest who created it (WM demoted to
// PG among them), not a creator who is not an active member of the project,
// and not a project role outside the three.
var creatorCells = map[shared.Action][]outcome{
	"issue.delete": {allowed, allowed, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
}

// cells decides rule for each identity of its level, who created the target
// work item when creator is true.
func cells(t *testing.T, rule domain.Rule, creator bool) []outcome {
````
````old server/internal/modules/access/domain/rules_test.go
			_, err := domain.Decide(rule, domain.Facts{Workspace: id.ws})
````
````new server/internal/modules/access/domain/rules_test.go
			_, err := domain.Decide(rule, domain.Facts{Workspace: id.ws, Creator: creator})
````
````old server/internal/modules/access/domain/rules_test.go
		_, err := domain.Decide(rule, domain.Facts{Workspace: id.ws, Project: &domain.Project{Public: id.public, Member: id.member}})
````
````new server/internal/modules/access/domain/rules_test.go
		_, err := domain.Decide(rule, domain.Facts{Workspace: id.ws, Project: &domain.Project{Public: id.public, Member: id.member},
			Creator: creator})
````
````old server/internal/modules/access/domain/rules_test.go
}

func TestEveryRuleDecidesItsCells(t *testing.T) {
````
````new server/internal/modules/access/domain/rules_test.go
}

// Each row decides its cells for a caller who did not create the target
// work item, and for one who did: a row with AllowCreator its creatorCells,
// any other the same cells, the creator counting for nothing.
func TestEveryRuleDecidesItsCells(t *testing.T) {
````
````old server/internal/modules/access/domain/rules_test.go
		if got := cells(t, rule); !slices.Equal(got, tableCells[action]) {
			t.Errorf("%s decides %q, want %q", action, got, tableCells[action])
````
````new server/internal/modules/access/domain/rules_test.go
		if got := cells(t, rule, false); !slices.Equal(got, tableCells[action]) {
			t.Errorf("%s decides %q, want %q", action, got, tableCells[action])
		}
		want, byCreator := creatorCells[action]
		if !byCreator {
			want = tableCells[action]
		}
		if byCreator != rule.AllowCreator {
			t.Errorf("%s: AllowCreator %t, and creatorCells has a row for it %t", action, rule.AllowCreator, byCreator)
		}
		if got := cells(t, rule, true); !slices.Equal(got, want) {
			t.Errorf("%s decides %q for the work item's creator, want %q", action, got, want)
````

`server/internal/shared/authorize.go`（修改，5 处）：

````old server/internal/shared/authorize.go
// action.
````
````new server/internal/shared/authorize.go
// action. CreatorID is the account that created the work item the action is
// on, or the work item a row it is on hangs from (M4 design 3.4); the zero
// UUID for every other action, and for a work item whose creator is not
// known.
````
````old server/internal/shared/authorize.go
	ProjectID   uuid.UUID
````
````new server/internal/shared/authorize.go
	ProjectID   uuid.UUID
	CreatorID   uuid.UUID
````
````old server/internal/shared/authorize.go
// again.
````
````new server/internal/shared/authorize.go
// again. OwnOnly is true for the project's guest (project role 5) when the
// project does not show its guests every work item (guest_view_all_features
// false): he sees only the work items he created (M4 design 3.4). It follows
// the project role, not the workspace's: a workspace admin who is only a
// guest in the project has it too.
````
````old server/internal/shared/authorize.go
	ProjectAdmin  bool
````
````new server/internal/shared/authorize.go
	ProjectAdmin  bool
	OwnOnly       bool
````
````old server/internal/shared/authorize.go
// rule does not allow gets Forbidden. An action without a rule is an
// internal error: nothing is allowed.
````
````new server/internal/shared/authorize.go
// rule does not allow gets Forbidden, with the Grant of what the decision
// read, so a use case can answer first that the guest rule hides the target
// from him (M4 design 3.4, 6.7). An action without a rule is an internal
// error: nothing is allowed.
````

- [ ] **Step 3: 动作、转换和关键词守卫的 phase**

`server/internal/bootstrap/ports_test.go`（修改，2 处）：

````old server/internal/bootstrap/ports_test.go
	fake := &fakeProjectAccess{facts: map[uuid.UUID]project.AccessFacts{web: {WorkspaceID: acme, Public: true, Member: true, Role: 15}}}
````
````new server/internal/bootstrap/ports_test.go
	fake := &fakeProjectAccess{facts: map[uuid.UUID]project.AccessFacts{web: {WorkspaceID: acme, Public: true, Member: true, Role: 5,
		GuestViewAll: true}}}
````
````old server/internal/bootstrap/ports_test.go
	if want := (access.ProjectFacts{WorkspaceID: acme, Public: true, Member: true, Role: 15}); err != nil || !found || f != want {
````
````new server/internal/bootstrap/ports_test.go
	if want := (access.ProjectFacts{WorkspaceID: acme, Public: true, Member: true, Role: 5, GuestViewAll: true}); err != nil || !found || f != want {
````

`server/internal/modules/issue/domain/actions.go`（修改，2 处）：

````old server/internal/modules/issue/domain/actions.go
	ActionUpdate shared.Action = "issue.update"
````
````new server/internal/modules/issue/domain/actions.go
	ActionUpdate shared.Action = "issue.update"
	// ActionDelete is deleting a work item of a project.
	ActionDelete shared.Action = "issue.delete"
	// ActionArchive is archiving and unarchiving a work item of a project.
	ActionArchive shared.Action = "issue.archive"
````
````old server/internal/modules/issue/domain/actions.go
	return []shared.Action{ActionCreate, ActionRead, ActionUpdate}
````
````new server/internal/modules/issue/domain/actions.go
	return []shared.Action{ActionCreate, ActionRead, ActionUpdate, ActionDelete, ActionArchive}
````

`tools/keywords.json`（修改，1 处）：

````old tools/keywords.json
  "phase": "M4/P1",
````
````new tools/keywords.json
  "phase": "M4/P2",
````

- [ ] **Step 4: 测试和 lint**

Run: `go -C server test -count=1 ./internal/shared/... ./internal/modules/access/... ./internal/modules/project/adapter/postgres/`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestAccessProjectsConvertsProjectsAnswer$|TestEveryActionHasARuleAndEveryRuleAnAction$|TestEveryModuleDeclaresItsActionsOrHasNone$' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过；关键词守卫 `keywords: 72 rules, 23 exceptions, no hits.`

- [ ] **Step 5: 提交**

```bash
git add server/internal/bootstrap/ports_test.go server/internal/modules/access/app/authorizer.go server/internal/modules/access/app/authorizer_test.go server/internal/modules/access/app/ports.go server/internal/modules/access/domain/creator_test.go server/internal/modules/access/domain/decide.go server/internal/modules/access/domain/decide_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/issue/domain/actions.go server/internal/modules/project/adapter/postgres/access.go server/internal/modules/project/adapter/postgres/access_test.go server/internal/modules/project/adapter/postgres/queries/access.sql server/internal/modules/project/app/access.go server/internal/shared/authorize.go tools/keywords.json server/internal/modules/project/adapter/postgres/gen/access.sql.go
```
```bash
git commit -m "feat(M4/P2): access decides the creator and a guest's own work items

A decision on a work item knows its creator (Target.CreatorID), and a
project-level rule with AllowCreator allows him while he is an active
member of the project of the role member or admin. The Grant says
whether the caller sees only his own work items (OwnOnly): the project's
guest when the project does not show its guests every one, by his
project role; a refusal of his role carries it, so that the use case
answers the guest rule first. The rule table gains issue.delete and
issue.archive; access reads guest_view_all_features.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p2.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试，或没有替身的平台、领域、转换的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是生成器（`make gen-go`）和中英文翻译键（`make lint-web`）的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过，端到端的检查只在标出的 Task 的树上）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T1.1` | `AllowCreator` 也允许成了项目访客的创建者 | `internal/modules/access/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `deleteIssue` 各格（`make test`）（Task 4 起）、`internal/bootstrap` 的 `TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused`（`make test`）（Task 12 起） | 单元；组合 |
| `T1.2` | 没有 `AllowCreator` 的规则也允许创建者 | `internal/modules/access/domain` 的全部测试（`make test`） | 单元 |
| `T1.3` | `AllowCreator` 允许项目的每个成员，不看是不是创建者 | `internal/modules/access/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `deleteIssue` 各格（`make test`）（Task 4 起） | 单元；组合 |
| `T1.4` | `Authorizer` 从不把调用者当作创建者 | `internal/modules/access/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `deleteIssue` 各格（`make test`）（Task 4 起）、`internal/bootstrap` 的 `TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused`（`make test`）（Task 12 起） | 单元；组合 |
| `T1.5` | 访客只看自己的工作项，不看 `guest_view_all_features` | `internal/modules/access/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `getIssue` 各格（`make test`）（Task 3 起）、故事 `i15-archive-issue`（Task 13 起） | 单元；组合；端到端 |
| `T1.6` | 没有一个判定是 `OwnOnly`：访客看得到每个工作项 | `internal/modules/access/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `getIssue` 各格（`make test`）（Task 2 起）、`internal/bootstrap` 的 `TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused`（`make test`）（Task 2 起） | 单元；组合 |
| `T1.7` | `OwnOnly` 按工作区的角色：在项目里是访客的工作区管理员看得到每个工作项 | `internal/modules/access/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `getIssue` 各格（`make test`）（Task 3 起） | 单元；组合 |
| `T1.8` | `Authorizer` 丢掉项目的 `guest_view_all_features` | `internal/modules/access/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `getIssue` 各格（`make test`）（Task 3 起）、故事 `i15-archive-issue`（Task 13 起） | 单元；组合；端到端 |
| `T1.9` | `ProjectFacts` 把每个项目都答成不让访客看全部的 | `internal/modules/project/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `getIssue` 各格（`make test`）（Task 3 起）、故事 `i15-archive-issue`（Task 13 起） | 存储；组合；端到端 |
| `T1.10` | `issue.delete` 没有 `AllowCreator` | `internal/modules/access/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `deleteIssue` 各格（`make test`）（Task 4 起） | 单元；组合 |
| `T1.11` | `issue.archive` 只给项目的管理员 | `internal/modules/access/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `archiveIssue`、`unarchiveIssue` 各格（`make test`）（Task 7 起）、故事 `i15-archive-issue`（Task 13 起） | 单元；组合；端到端 |
| `T1.12` | 角色的拒绝丢掉这次判定读到的 `Grant`（`OwnOnly` 在内）：别人的工作项给访客答 403 | `internal/modules/access/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `updateIssue` 各格（`make test`）（Task 2 起）、`internal/bootstrap` 的 `TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused`（`make test`）（Task 2 起） | 单元；组合 |
| `T1.13` | 三个之外的角色的拒绝丢掉 `Grant`（按性质只在单元一层：表的 `role` 有 `CHECK (role IN (5, 15, 20))`，这样的角色进不了表） | `internal/modules/access/domain` 的全部测试（`make test`） | 单元 |
| `T1.14` | `creatorRoles` 没有管理员：建了工作项的项目管理员不算 `AllowCreator` 的创建者（按性质只在单元一层，同 `T1.2`：到 P2 为止带 `AllowCreator` 的规则都列了管理员） | `internal/modules/access/domain` 的全部测试（`make test`） | 单元 |
| `T1.15` | `Authorizer` 拒绝时丢掉判定读到的 `Grant`：别人的工作项给访客答 403 | `internal/modules/access/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `updateIssue` 各格（`make test`）（Task 2 起）、`internal/bootstrap` 的 `TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused`（`make test`）（Task 2 起） | 单元；组合 |

---

### Task 2: 访客规则：`getIssue`、`updateIssue` 走它，404 优先

**Files:**
- Create: `server/internal/modules/issue/app/guest_rule_test.go`
- Modify: `api/modules/issue.yaml`、`api/modules/project.yaml`、`server/internal/bootstrap/issue_interleaving_test.go`、`server/internal/bootstrap/issue_world_test.go`、`server/internal/bootstrap/permission_matrix_issue_updates_test.go`、`server/internal/bootstrap/permission_matrix_issues_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/issue/adapter/http/issues_test.go`、`server/internal/modules/issue/app/fakes_test.go`、`server/internal/modules/issue/app/get_issue.go`、`server/internal/modules/issue/app/lock.go`、`server/internal/modules/issue/app/update_issue.go`、`server/internal/modules/issue/domain/issue.go`、`server/internal/modules/issue/domain/issue_test.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**（spec 2.4、第 3 节第 2 条；M4 设计 3.4、3.5、6.7、8.4）
- `access/domain/rules.go`：`issue.read` 加项目的访客。
- `issue/domain/issue.go`：`Issue.HiddenFrom(actor uuid.UUID, ownOnly bool) bool`：`ownOnly` 的调用者看不到别人建的工作项，也看不到创建者不明的；别的调用者什么都不藏。
- `issue/app/lock.go`：`decide(ctx, actor, action, workspaceID, projectID uuid.UUID, i *domain.Issue, notFound error) error`：`i` 不为空时 `Target.CreatorID` 是它的创建者；`ErrNotVisible` 答 `notFound`；`i.HiddenFrom(actor.UserID, grant.OwnOnly)` 也答 `notFound`，先于角色的拒绝；`see` 传读到的工作项，`lockAndDecide` 传锁下重读的工作项（`h.issue, on = i, &i`），`precheck` 的按项目的一支传 `nil`。`get_issue.go`、`update_issue.go` 的说明写上访客规则。
- 契约：`getIssue`、`updateIssue` 的描述（访客规则，404 不论角色）；`api/modules/project.yaml` 的 `guest_view_all_features` 的说明（设计 12 节 P2 的第 2 个任务）。
- 测试的帮手：`fakes_test.go` 的假判定记下每次的目标，按次回答 `grants[n]`（拒绝时也带它），假存储的读把零值的创建者答成没有（`creatorOf`：创建者的账户被清除之后，P9）；`issue_world_test.go` 的 `newWorkItemBy`（别的账户建的工作项，`newWorkItemIn` 用它）和交错用的 `bobsWorkItem`、`waltsWorkItem`（walt 在 bob 之后注册，id 在他之后；alice 先让他成为 Web 的成员）。

**Tests:**
- 领域：`TestTheGuestRuleHidesAnothersWorkItem`。
- 单元（`guest_rule_test.go`，新）：`TestADecisionOnAWorkItemNamesItsCreator`（预检和锁下各把创建者交给判定，按项目的判定不指名谁）、`TestTheGuestRuleHidesAnothersWorkItemFromAReader`（创建者不明的工作项对只看自己的访客也是 404）、`TestTheGuestRuleHidesAnothersWorkItemFromAWriter`（预检答 404，在闸门和锁之前；等锁期间成了这样的访客的，锁下 404，不写；对自己的，角色的 403）；规则表的 `issue.read` 一行。
- HTTP：`TestGetIssueRefusals`（访客规则的 404）。
- 组合：交错 5 的访客一支（`TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused`：被降为访客的 bob 改 alice 的 Fix、walt 的 Fix 404（创建者的 id 一个在他之前、一个在他之后），改他自己的和创建 403，都不写）；矩阵的 `getIssue`、`updateIssue` 的 PG 格子（"只看自己"的项目里别人的工作项：404）。

- [ ] **Step 1: 契约，生成**

`api/modules/issue.yaml`（修改，2 处）：

````old api/modules/issue.yaml
        For the admins and members of its project, archived or not; an
        archived work item is read as any other. A work item that does not
        exist, is deleted, or whose project the caller does not see answers
        issue.not_found; one he sees the project of but may not read,
        forbidden.
````
````new api/modules/issue.yaml
        For the admins, members and guests of its project, archived or not;
        an archived work item is read as any other. A guest of a project
        that does not show its guests every work item
        (guest_view_all_features false) reads only the work items he
        created. A work item that does not exist, is deleted, whose project
        the caller does not see, or that he, such a guest, did not create,
        answers issue.not_found; one he sees the project of but may not
        read, forbidden.
````
````old api/modules/issue.yaml
        exist, is deleted, or whose project the caller does not see answers
        issue.not_found; one whose project he sees but whose work items he
        may not change, forbidden; both are answered before the project is
        locked, and with the project locked the role is decided again. A
        work item of an archived project cannot be changed
````
````new api/modules/issue.yaml
        exist, is deleted, whose project the caller does not see, or that he
        did not create as a guest who sees only his own (getIssue) answers
        issue.not_found, whatever his role may do; one whose project he sees
        but whose work items he may not change, forbidden; both are answered
        before the project is locked, and with the project locked the role
        is decided again. A work item of an archived project cannot be changed
````

`api/modules/project.yaml`（修改，1 处）：

````old api/modules/project.yaml
          description: A setting of the project for its guests, kept as set.
````
````new api/modules/project.yaml
          description: >-
            Whether the project's guests see every work item of it. When
            false, a guest of the project sees only the work items he
            created, and every other answers him as one that does not exist
            (issue.not_found).
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `5432103262a314da890ec4d10d4b27cdd971294f7bdc70597fa9f5a77ef538b0` | 3198 | `api/dist/openapi.yaml` |
| `61cbb8a6e416fad860bf023f12edefbef654bab237c22b8a0026fc0393d72973` | 3505 | `server/internal/modules/project/adapter/http/gen/server.gen.go` |
| `917f7685c774ddcdb32117559c72d7d1db60dcb97e55d9a6f2381c799ff1fe9e` | 3492 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: 规则和领域**

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	// As issue.create; the project's guests are P2's (M4 design 3.4).
	"issue.read": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
````
````new server/internal/modules/access/domain/rules.go
	// The project's admins, members and guests; a guest who sees only his
	// own work items finds no other, the issue module's guest rule (M4
	// design 3.4).
	"issue.read": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	// As issue.create; the project's guests are P2's (M4 design 3.4).
	"issue.read": {allowed, allowed, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
````
````new server/internal/modules/access/domain/rules_test.go
	// As project_member.list: every active member of the project, the guest
	// rule the issue module's (M4 design 3.4).
	"issue.read": {allowed, allowed, allowed, allowed, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
````

`server/internal/modules/issue/domain/issue.go`（修改，1 处）：

````old server/internal/modules/issue/domain/issue.go
	UpdatedAt   time.Time
````
````new server/internal/modules/issue/domain/issue.go
	UpdatedAt   time.Time
}

// HiddenFrom reports whether the guest rule hides i from the account actor,
// whose grant on i's project is ownOnly (M4 design 3.4): the project's
// guest who sees only his own work items does not see one another created,
// nor one whose creator is not known. It hides nothing from any other
// caller.
func (i Issue) HiddenFrom(actor uuid.UUID, ownOnly bool) bool {
	return ownOnly && (i.CreatedByID == nil || *i.CreatedByID != actor)
````

`server/internal/modules/issue/domain/issue_test.go`（修改，1 处）：

````old server/internal/modules/issue/domain/issue_test.go
		t.Errorf("UnknownLabel(3) = %+v, want %+v", got, want)
	}
}

````
````new server/internal/modules/issue/domain/issue_test.go
		t.Errorf("UnknownLabel(3) = %+v, want %+v", got, want)
	}
}

// The guest rule hides from a caller who sees only his own work items each
// work item another created, the other's id drawn before his or after it,
// and one whose creator is not known; from any other caller, none (M4
// design 3.4): it tells his own by its creator, not by the ids' order.
func TestTheGuestRuleHidesAnothersWorkItem(t *testing.T) {
	creator := ids(3)
	earlier, gus, other := creator[0], creator[1], creator[2]
	tests := []struct {
		name    string
		by      *uuid.UUID
		ownOnly bool
		want    bool
	}{
		{"his own", &gus, true, false},
		{"another's", &other, true, true},
		{"another's, whose id is drawn before his", &earlier, true, true},
		{"one whose creator is not known", nil, true, true},
		{"another's, to a caller who sees every work item", &other, false, false},
		{"one whose creator is not known, to a caller who sees every work item", nil, false, false},
	}
	for _, tt := range tests {
		if got := (Issue{CreatedByID: tt.by}).HiddenFrom(gus, tt.ownOnly); got != tt.want {
			t.Errorf("%s: HiddenFrom() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

````

- [ ] **Step 3: 判定先执行访客规则**

`server/internal/modules/issue/app/fakes_test.go`（修改，7 处）：

````old server/internal/modules/issue/app/fakes_test.go
// for allowed, with an empty grant, which Locks does not read; it logs
// every call.
````
````new server/internal/modules/issue/app/fakes_test.go
// for allowed, with grants[n], the Grant of what it read, which a refusal
// of the role carries too (shared.Authorizer), empty past the end; it logs
// every call, and keeps each call's target.
````
````old server/internal/modules/issue/app/fakes_test.go
	answers []error
````
````new server/internal/modules/issue/app/fakes_test.go
	answers []error
	grants  []shared.Grant
	targets []shared.Target
````
````old server/internal/modules/issue/app/fakes_test.go
	f.calls++
````
````new server/internal/modules/issue/app/fakes_test.go
	f.calls++
	f.targets = append(f.targets, t)
````
````old server/internal/modules/issue/app/fakes_test.go
		return shared.Grant{}, err
	}
````
````new server/internal/modules/issue/app/fakes_test.go
		return shared.Grant{}, err
	}
	var grant shared.Grant
	if n < len(f.grants) {
		grant = f.grants[n]
	}
````
````old server/internal/modules/issue/app/fakes_test.go
		return shared.Grant{}, f.answers[n]
````
````new server/internal/modules/issue/app/fakes_test.go
		return grant, f.answers[n]
````
````old server/internal/modules/issue/app/fakes_test.go
	return shared.Grant{}, nil
````
````new server/internal/modules/issue/app/fakes_test.go
	return grant, nil
````
````old server/internal/modules/issue/app/fakes_test.go
			uuid.UUID.Compare), CreatedByID: &r.By, UpdatedByID: &by, CreatedAt: r.At, UpdatedAt: at}, true, nil
}

````
````new server/internal/modules/issue/app/fakes_test.go
			uuid.UUID.Compare), CreatedByID: creatorOf(r.By), UpdatedByID: &by, CreatedAt: r.At, UpdatedAt: at}, true, nil
}

// creatorOf is a stored work item's creator as a read answers it: none for
// the zero id, as for a work item whose creator's account is purged (P9).
func creatorOf(by uuid.UUID) *uuid.UUID {
	if by == (uuid.UUID{}) {
		return nil
	}
	return &by
}

````

`server/internal/modules/issue/app/get_issue.go`（修改，1 处）：

````old server/internal/modules/issue/app/get_issue.go
// assignees and labels, and decides issue.read on its project for the
// caller, without a lock or a transaction (Locks.see): a work item that is
// not there, or whose project the caller does not see, is
// issue.not_found; a role that may not read it, forbidden.
````
````new server/internal/modules/issue/app/get_issue.go
// assignees and labels, and decides issue.read on it for the caller,
// without a lock or a transaction (Locks.see): a work item that is not
// there, whose project the caller does not see, or that the guest rule
// hides from him, is issue.not_found; a role that may not read it,
// forbidden.
````

`server/internal/modules/issue/app/guest_rule_test.go`（新文件，143 行）：

````file server/internal/modules/issue/app/guest_rule_test.go
package app_test

import (
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The guest rule in Locks' decisions (M4 design 3.4, 6.7): a decision on a
// work item names its creator, and a caller who sees only his own work
// items (Grant.OwnOnly) finds none another created, before his role is
// looked at: in the precheck, and again under the lock.

var (
	// ownOnly is the grant of web's guest when web does not show its
	// guests every work item: he sees only those he created.
	ownOnly = shared.Grant{WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleGuest, OwnOnly: true}
	// everyItem is the same guest's when web shows its guests every work
	// item.
	everyItem = shared.Grant{WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleGuest}
)

// A guest who sees only his own work items reads his own, and not
// another's, carol's Fix or bob's, which is issue.not_found to alice as
// such a guest, the work item read and the one decision taken; carol's id
// is after hers and bob's before it, so the rule tells her own by its
// creator, not by the ids' order. When web shows its guests every work
// item, she reads carol's. A refusal of her role on another's work item
// is issue.not_found too, the guest rule first; on her own, forbidden. A
// work item whose creator is not known, as one whose creator's account is
// purged (P9), is no guest's own: issue.not_found.
func TestTheGuestRuleHidesAnothersWorkItemFromAReader(t *testing.T) {
	for _, tt := range []struct {
		name    string
		creator uuid.UUID
		grant   shared.Grant
		refused error
		want    error
	}{
		{"another's", carol, ownOnly, nil, domain.ErrIssueNotFound},
		{"another's, bob's, whose id is before hers", bob, ownOnly, nil, domain.ErrIssueNotFound},
		{"her own", alice, ownOnly, nil, nil},
		{"another's, web showing its guests every work item", carol, everyItem, nil, nil},
		{"another's, her role refused", carol, ownOnly, shared.Forbidden(), domain.ErrIssueNotFound},
		{"her own, her role refused", alice, ownOnly, shared.Forbidden(), shared.Forbidden()},
		{"no known creator's", uuid.UUID{}, ownOnly, nil, domain.ErrIssueNotFound},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id := newGet()
			w.issues.issue.By = tt.creator
			w.auth.grants, w.auth.answers = []shared.Grant{tt.grant}, []error{tt.refused}
			got, err := uc.Execute(as(alice), id)
			if (tt.want == nil) != (err == nil) || tt.want != nil && !errors.Is(err, tt.want) || (err == nil) != (got.ID == id) {
				t.Errorf("Execute() = %+v, %v; want %v", got, err, tt.want)
			}
			if calls := []string{line("GetIssue", id, "outside tx"), line("Authorize", alice, domain.ActionRead, acme, web, "outside tx")}; !slices.Equal(
				w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
		})
	}
}

// A write on another's work item by a guest who sees only his own is
// issue.not_found in the precheck, his role refused or not, before the
// gate and any lock; on his own, his refused role is forbidden there. A
// write the precheck let through, whose caller became such a guest while
// it waited, is issue.not_found under the lock, the decision there final,
// and writes nothing; forbidden on his own.
func TestTheGuestRuleHidesAnothersWorkItemFromAWriter(t *testing.T) {
	member := shared.Grant{WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleMember}
	for _, tt := range []struct {
		name    string
		creator uuid.UUID
		grants  []shared.Grant
		answers []error
		want    error
		calls   int // of locked(id)
	}{
		{"another's, in the precheck", carol, []shared.Grant{ownOnly}, []error{shared.Forbidden()}, domain.ErrIssueNotFound, 2},
		{"another's, in the precheck, his role allowed", carol, []shared.Grant{ownOnly}, nil, domain.ErrIssueNotFound, 2},
		{"his own, in the precheck", alice, []shared.Grant{ownOnly}, []error{shared.Forbidden()}, shared.Forbidden(), 2},
		{"another's, under the lock", carol, []shared.Grant{member, ownOnly}, []error{nil, shared.Forbidden()}, domain.ErrIssueNotFound, 6},
		{"his own, under the lock", alice, []shared.Grant{member, ownOnly}, []error{nil, shared.Forbidden()}, shared.Forbidden(), 6},
		{"another's, under the lock, his role allowed", carol, []shared.Grant{member, ownOnly}, nil, domain.ErrIssueNotFound, 6},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, w, id := newUpdate()
			_, update := gated(w, nil)
			w.issues.issue.By = tt.creator
			w.auth.grants, w.auth.answers = tt.grants, tt.answers
			if _, err := update.Execute(as(alice), id, domain.IssuePatch{Name: name("Mend")}); !errors.Is(err, tt.want) {
				t.Errorf("Execute() = %v, want %v", err, tt.want)
			}
			want := locked(id)[:tt.calls]
			if tt.calls > 2 {
				want = slices.Concat(locked(id)[:2], []string{line("Enter", web, "outside tx")}, locked(id)[2:], []string{line("Leave", web)})
			}
			if !slices.Equal(w.log.calls, want) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, want)
			}
		})
	}
}

// Each decision on a work item names its creator, the precheck's and the
// one under the lock, so that a rule may allow him (AllowCreator); a
// decision on a project names no one.
func TestADecisionOnAWorkItemNamesItsCreator(t *testing.T) {
	get, w, id := newGet()
	if _, err := get.Execute(as(alice), id); err != nil {
		t.Fatal(err)
	}
	update, uw, uid := newUpdate()
	if _, err := update.Execute(as(alice), uid, domain.IssuePatch{Name: name("Mend")}); err != nil {
		t.Fatal(err)
	}
	create, cw := newCreate()
	if _, err := create.Execute(as(alice), web, domain.NewIssue{Name: "Fix"}); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name    string
		targets []shared.Target
		want    []uuid.UUID
	}{
		{"a read", w.auth.targets, []uuid.UUID{carol}},
		{"an update", uw.auth.targets, []uuid.UUID{carol, carol}},
		{"a creation", cw.auth.targets, []uuid.UUID{{}, {}}},
	} {
		var got []uuid.UUID
		for _, target := range tt.targets {
			got = append(got, target.CreatorID)
		}
		if !slices.Equal(got, tt.want) {
			t.Errorf("%s: the decisions' creators %v, want %v", tt.name, got, tt.want)
		}
	}
}
````

`server/internal/modules/issue/app/lock.go`（修改，10 处）：

````old server/internal/modules/issue/app/lock.go
// lock (see). issue.New builds one Locks for every use case: none holds an
````
````new server/internal/modules/issue/app/lock.go
// lock (see). Every decision on a work item applies the guest rule first
// (decide). issue.New builds one Locks for every use case: none holds an
````
````old server/internal/modules/issue/app/lock.go
// decision on its project (see); a work item that is not there, or whose
// project actor does not see, is issue.not_found. For one on a project:
// the project read without a lock and the decision on it; a project that
// is not there, or not visible to actor, is w.notFound. A role the rule
// does not allow is the Authorizer's shared.Forbidden. What it lets
// through, lockAndDecide decides again.
````
````new server/internal/modules/issue/app/lock.go
// decision on it (see); a work item that is not there, whose project actor
// does not see, or that the guest rule hides from him, is
// issue.not_found. For one on a project: the project read without a lock
// and the decision on it; a project that is not there, or not visible to
// actor, is w.notFound. A role the rule does not allow is the Authorizer's
// shared.Forbidden. What it lets through, lockAndDecide decides again.
````
````old server/internal/modules/issue/app/lock.go
	return w, l.decide(ctx, actor, w.action, shared.Target{WorkspaceID: workspaceID, ProjectID: w.project}, w.notFound)
````
````new server/internal/modules/issue/app/lock.go
	return w, l.decide(ctx, actor, w.action, workspaceID, w.project, nil, w.notFound)
````
````old server/internal/modules/issue/app/lock.go
// item w names read again; then the decision on the locked project. It
// answers what they hold. A project deleted while the lock waited is
// w.notFound, and so is a work item deleted meanwhile, or no longer in the
// project, and a project no longer visible to actor; a role the rule no
// longer allows is the Authorizer's shared.Forbidden.
````
````new server/internal/modules/issue/app/lock.go
// item w names read again; then the decision on the locked project, and on
// the work item as read again. It answers what they hold. A project
// deleted while the lock waited is w.notFound, and so is a work item
// deleted meanwhile, or no longer in the project, a project no longer
// visible to actor, and a work item the guest rule now hides from him; a
// role the rule no longer allows is the Authorizer's shared.Forbidden.
````
````old server/internal/modules/issue/app/lock.go
	h := held{project: p}
````
````new server/internal/modules/issue/app/lock.go
	h := held{project: p}
	var on *domain.Issue
````
````old server/internal/modules/issue/app/lock.go
		h.issue = i
````
````new server/internal/modules/issue/app/lock.go
		h.issue, on = i, &i
````
````old server/internal/modules/issue/app/lock.go
	if err := l.decide(ctx, actor, w.action, shared.Target{WorkspaceID: p.WorkspaceID, ProjectID: w.project}, w.notFound); err != nil {
````
````new server/internal/modules/issue/app/lock.go
	if err := l.decide(ctx, actor, w.action, p.WorkspaceID, w.project, on, w.notFound); err != nil {
````
````old server/internal/modules/issue/app/lock.go
// and decides action on its project for actor (M4 design 3.4, 3.5): a work
// item that is not there, or whose project actor does not see, is
// issue.not_found; a role the rule does not allow is the Authorizer's
// shared.Forbidden. It answers the work item as read.
````
````new server/internal/modules/issue/app/lock.go
// and decides action on it for actor (M4 design 3.4, 3.5): a work item that
// is not there, whose project actor does not see, or that the guest rule
// hides from him, is issue.not_found; a role the rule does not allow is
// the Authorizer's shared.Forbidden. It answers the work item as read.
````
````old server/internal/modules/issue/app/lock.go
	if err := l.decide(ctx, actor, action, shared.Target{WorkspaceID: i.WorkspaceID, ProjectID: i.ProjectID}, domain.ErrIssueNotFound); err != nil {
````
````new server/internal/modules/issue/app/lock.go
	if err := l.decide(ctx, actor, action, i.WorkspaceID, i.ProjectID, &i, domain.ErrIssueNotFound); err != nil {
````
````old server/internal/modules/issue/app/lock.go
// decide asks the Authorizer for action on the target for actor. A
// project not visible to actor is notFound.
func (l Locks) decide(ctx context.Context, actor shared.Actor, action shared.Action, target shared.Target, notFound error) error {
	_, err := l.auth.Authorize(ctx, actor, action, target)
	if errors.Is(err, shared.ErrNotVisible) {
````
````new server/internal/modules/issue/app/lock.go
// decide asks the Authorizer for action on the project projectID of the
// workspace workspaceID for actor, and, for an action on a work item, i, on
// it: the decision knows its creator (shared.Target.CreatorID), whom a rule
// may allow (AllowCreator). A project not visible to actor is notFound, and
// so is a work item the guest rule hides from him (M4 design 3.4, 6.7):
// the project's guest who sees only his own work items (Grant.OwnOnly), a
// workspace admin among them, does not see one another created, before
// what the rule says of his role, so another's work item is as unknown to
// him as one that is not there (8.4). i is nil for an action on the
// project.
func (l Locks) decide(ctx context.Context, actor shared.Actor, action shared.Action, workspaceID, projectID uuid.UUID, i *domain.Issue,
	notFound error) error {
	target := shared.Target{WorkspaceID: workspaceID, ProjectID: projectID}
	if i != nil && i.CreatedByID != nil {
		target.CreatorID = *i.CreatedByID
	}
	grant, err := l.auth.Authorize(ctx, actor, action, target)
	switch {
	case errors.Is(err, shared.ErrNotVisible):
		return notFound
	case i != nil && i.HiddenFrom(actor.UserID, grant.OwnOnly):
````

`server/internal/modules/issue/app/update_issue.go`（修改，1 处）：

````old server/internal/modules/issue/app/update_issue.go
// precheck, the work item read and the decision on issue.update in its
// project, before any lock; then the project's write gate, which refuses a
// write that may wait no longer (503 server_busy); then in one
````
````new server/internal/modules/issue/app/update_issue.go
// precheck, the work item read and the decision on issue.update on it, the
// guest rule first, before any lock; then the project's write gate, which
// refuses a write that may wait no longer (503 server_busy); then in one
````

- [ ] **Step 4: HTTP、交错和矩阵**

`server/internal/bootstrap/issue_interleaving_test.go`（修改，5 处）：

````old server/internal/bootstrap/issue_interleaving_test.go
// member of Web, made secret, creates a work item, or changes alice's
// Fix, while alice's removal of him, or her demotion of him to a guest,
// holds Web past its writes. His write passed the precheck, which read him
// a member, and waits for Web's row; once hers commits, the decision under
// the lock reads him removed, who no longer sees Web: 404
// project.not_found, or issue.not_found for the change; or a guest, who
// writes no work item: 403 forbidden. His write wrote nothing.
````
````new server/internal/bootstrap/issue_interleaving_test.go
// member of Web, made secret, creates a work item, or changes alice's Fix,
// walt's or his own, while alice's removal of him, or her demotion of him
// to a guest, holds Web past its writes. His write passed the precheck,
// which read him a member, and waits for Web's row; once hers commits, the
// decision under the lock reads him removed, who no longer sees Web: 404
// project.not_found, or issue.not_found for a change; or a guest of a
// project that does not show its guests every work item, who finds no work
// item another created, alice's Fix or walt's, their creators' ids before
// his and after it, as one not there: 404 issue.not_found (3.4); and who
// writes no work item, his own or a new one: 403 forbidden. His write
// wrote nothing.
````
````old server/internal/bootstrap/issue_interleaving_test.go
	for _, iw := range itemWrites {
````
````new server/internal/bootstrap/issue_interleaving_test.go
	// guest is a guest's answer to each write: issue.not_found for alice's
	// Fix and walt's, forbidden for the others.
	guest := map[string]cell{"created": cellForbidden, "changed": cellIssueNotFound, "changed his own": cellForbidden,
		"changed walt's": cellIssueNotFound}
	for _, iw := range append(slices.Clone(itemWrites),
		itemWrite{"changed his own", bobsWorkItem, 3, changesIssue, http.StatusOK, "issue.not_found"},
		itemWrite{"changed walt's", waltsWorkItem, 3, changesIssue, http.StatusOK, "issue.not_found"}) {
````
````old server/internal/bootstrap/issue_interleaving_test.go
			want    int
			code    string // "": iw.gone
````
````new server/internal/bootstrap/issue_interleaving_test.go
			want    cell // a zero status: 404 iw.gone
````
````old server/internal/bootstrap/issue_interleaving_test.go
			{"removed", removesBob, http.StatusNoContent, http.StatusNotFound, ""},
			{"made a guest", demotesBob, http.StatusOK, http.StatusForbidden, "forbidden"},
````
````new server/internal/bootstrap/issue_interleaving_test.go
			{"removed", removesBob, http.StatusNoContent, cell{}},
			{"made a guest", demotesBob, http.StatusOK, guest[iw.name]},
````
````old server/internal/bootstrap/issue_interleaving_test.go
				w.refused(t, written, tt.want, cmp.Or(tt.code, iw.gone))
````
````new server/internal/bootstrap/issue_interleaving_test.go
				want := cmp.Or(tt.want, cell{http.StatusNotFound, iw.gone})
				w.refused(t, written, want.status, want.code)
````

`server/internal/bootstrap/issue_world_test.go`（修改，2 处）：

````old server/internal/bootstrap/issue_world_test.go
func (w issueWorld) newWorkItemIn(t *testing.T, project uuid.UUID, body string) uuid.UUID {
	t.Helper()
````
````new server/internal/bootstrap/issue_world_test.go
func (w issueWorld) newWorkItemIn(t *testing.T, project uuid.UUID, body string) uuid.UUID {
	t.Helper()
	return w.newWorkItemBy(t, "alice", project, body)
}

// newWorkItemBy is the id of a work item the account name creates in
// project through the API, with body.
func (w issueWorld) newWorkItemBy(t *testing.T, name string, project uuid.UUID, body string) uuid.UUID {
	t.Helper()
````
````old server/internal/bootstrap/issue_world_test.go
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+project.String()+"/issues", body, http.StatusCreated), &created)
	return created.ID
````
````new server/internal/bootstrap/issue_world_test.go
	decodeAnswer(t, w.send(t, name, http.MethodPost, "/api/v0/projects/"+project.String()+"/issues", body, http.StatusCreated), &created)
	return created.ID
}

// bobsWorkItem is aWorkItem made by bob, Web's member: three rows, his.
func bobsWorkItem(t *testing.T, w issueWorld) {
	w.newWorkItemBy(t, "bob", w.web, `{"name":"Fix"}`)
}

// waltsWorkItem is aWorkItem made by walt, registered after bob and so of
// an id after his, once alice has made him a member of Web: three rows,
// his.
func waltsWorkItem(t *testing.T, w issueWorld) {
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/members",
		`{"members":[{"member_id":"`+w.ids["walt"].String()+`","role":15}]}`, http.StatusCreated)
	w.newWorkItemBy(t, "walt", w.web, `{"name":"Fix"}`)
````

`server/internal/bootstrap/permission_matrix_issue_updates_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
	ok := issueRows.of(cellOK, cellOK, cellForbidden, cellOK, cellForbidden, cellForbidden)
	refused := issueRows.of(cellValidationFailed, cellValidationFailed, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden)
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
	ok := issueRows.of(cellOK, cellOK, cellIssueNotFound, cellOK, cellForbidden, cellForbidden)
	refused := issueRows.of(cellValidationFailed, cellValidationFailed, cellIssueNotFound, cellValidationFailed, cellForbidden, cellForbidden)
````
````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
		// The project's admins and members, as createIssue (M4 design 3.4).
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
		// The project's admins and members, as createIssue (M4 design 3.4);
		// PG, a guest who sees only his own work items, finds no other's,
		// whatever his role may do: issue.not_found, not forbidden.
````

`server/internal/bootstrap/permission_matrix_issues_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_issues_test.go
		// As createIssue; the project's guests are P2's (M4 design 3.4).
````
````new server/internal/bootstrap/permission_matrix_issues_test.go
		// The project's admins, members and guests; PG, a guest who sees only
		// his own work items, finds no other's, as one not there (M4 design
		// 3.4).
````
````old server/internal/bootstrap/permission_matrix_issues_test.go
			cells: issueRows.of(cellOK, cellOK, cellForbidden, cellOK, cellForbidden, cellForbidden), check: readsTheIssue},
````
````new server/internal/bootstrap/permission_matrix_issues_test.go
			cells: issueRows.of(cellOK, cellOK, cellIssueNotFound, cellOK, cellForbidden, cellForbidden), check: readsTheIssue},
````

`server/internal/modules/issue/adapter/http/issues_test.go`（修改，1 处）：

````old server/internal/modules/issue/adapter/http/issues_test.go
			`{"status":404,"code":"issue.not_found","title":"Not Found","detail":"The work item does not exist, or you cannot see it."}`},
		{"a project guest", shared.Forbidden(), http.StatusForbidden,
````
````new server/internal/modules/issue/adapter/http/issues_test.go
			`{"status":404,"code":"issue.not_found","title":"Not Found","detail":"The work item does not exist, or you cannot see it."}`},
		{"a workspace admin who is no member of the project", shared.Forbidden(), http.StatusForbidden,
````

- [ ] **Step 5: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused$|TestPermissionMatrix' ./internal/bootstrap/`
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
Expected: 122 个全部通过（每个工作项的读写的判定都经访客规则）。

- [ ] **Step 6: 提交**

```bash
git add api/modules/issue.yaml api/modules/project.yaml server/internal/bootstrap/issue_interleaving_test.go server/internal/bootstrap/issue_world_test.go server/internal/bootstrap/permission_matrix_issue_updates_test.go server/internal/bootstrap/permission_matrix_issues_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/issue/adapter/http/issues_test.go server/internal/modules/issue/app/fakes_test.go server/internal/modules/issue/app/get_issue.go server/internal/modules/issue/app/guest_rule_test.go server/internal/modules/issue/app/lock.go server/internal/modules/issue/app/update_issue.go server/internal/modules/issue/domain/issue.go server/internal/modules/issue/domain/issue_test.go api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M4/P2): the guest rule hides another's work item, before any role

issue.read takes the project's guests. Every decision on a work item, the
precheck's and the one under the lock, names its creator and applies the
guest rule first: a guest who sees only his own work items finds another's
as one not there (404 issue.not_found), whatever his role may do; on his
own, his role decides. getIssue and updateIssue go through it, and a
member demoted to such a guest while his change waits finds another's
work item gone under the lock (interleaving 5).

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p2.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试，或没有替身的平台、领域、转换的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是生成器（`make gen-go`）和中英文翻译键（`make lint-web`）的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过，端到端的检查只在标出的 Task 的树上）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T2.1` | 访客规则什么都不藏 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `getIssue` 各格（`make test`）、`internal/bootstrap` 的 `TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused`（`make test`） | 单元；组合 |
| `T2.2` | 规则不允许的角色先于访客规则答：别人的工作项给访客答 403 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `updateIssue` 各格（`make test`）、`internal/bootstrap` 的 `TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused`（`make test`） | 单元；组合 |
| `T2.3` | 访客规则把创建者不明的工作项给访客看 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/modules/issue/app` 的全部测试（`make test`） | 单元 |
| `T2.4` | 对工作项的判定不知道它的创建者 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `deleteIssue` 各格（`make test`）（Task 4 起）、`internal/bootstrap` 的 `TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused`（`make test`）（Task 12 起） | 单元；组合 |
| `T2.5` | 锁下的判定看不到工作项：那里没有访客规则、没有创建者 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused`（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `deleteIssue` 各格（`make test`）（Task 4 起） | 单元；组合 |
| `T2.6` | 预检和读看不到工作项：锁之前没有访客规则 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `getIssue` 各格（`make test`） | 单元；组合 |
| `T2.7` | `issue.read` 没有项目的访客 | `internal/modules/access/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `getIssue` 各格（`make test`）（Task 3 起） | 单元；组合 |
| `T2.8` | `getIssue` 判定 `issue.create` 而不是 `issue.read`（P1b 评审第 6 节：访客让两者分得出） | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `getIssue` 各格（`make test`）（Task 3 起） | 单元；组合 |
| `T2.9` | 创建者不明的工作项在 `AllowCreator` 上算调用者的 | `internal/modules/issue/app` 的全部测试（`make test`）（Task 4 起） | 单元 |
| `T2.10` | 访客规则按 id 的次序而不是相等认调用者自己的工作项：创建者的 id 在调用者之前的别人的工作项给访客看 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `getIssue` 各格（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `updateIssue` 各格（`make test`）、`internal/bootstrap` 的 `TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused`（`make test`） | 单元；组合 |
| `T2.11` | 访客规则按 id 的次序而不是相等认调用者自己的工作项：创建者的 id 在调用者之后的别人的工作项给访客看（`T2.10` 的镜像） | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused`（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `getIssue` 各格（`make test`）（Task 3 起）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `updateIssue` 各格（`make test`）（Task 3 起） | 单元；组合 |
| `T2.12` | 访客规则隐藏的工作项答一个文字不同的 `issue.not_found`：别人的工作项不再与不存在的是同一个回答（M4 设计 8.4） | `internal/modules/issue/app` 的全部测试（`make test`）（Task 5 起）、`internal/bootstrap` 的 `TestAWorkItemIsFoundByItsIdentifier`（`make test`）（Task 6 起） | 单元；组合 |

本 Task 的检查也发现较早的 Task 的变异 `T1.6`、`T1.12`、`T1.15`（列在各自的 Task 的表里，标"（Task 2 起）"）：本 Task 做完之后，在本 Task 的树上重跑它们（`mutants_p2.py`，`MUT_LATER=2`）。

---

### Task 3: 矩阵的工作项表：访客的三列、"管理员·访客"、建了它的项目成员

**Files:**
- Create: `server/internal/bootstrap/permission_matrix_issue_columns_test.go`
- Modify: `server/internal/bootstrap/permission_matrix_columns_test.go`、`server/internal/bootstrap/permission_matrix_coverage_test.go`、`server/internal/bootstrap/permission_matrix_invitations_test.go`、`server/internal/bootstrap/permission_matrix_issue_updates_test.go`、`server/internal/bootstrap/permission_matrix_issues_test.go`、`server/internal/bootstrap/permission_matrix_members_test.go`、`server/internal/bootstrap/permission_matrix_seed_issues_test.go`、`server/internal/bootstrap/permission_matrix_seeded_test.go`、`server/internal/bootstrap/permission_matrix_targets_test.go`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/bootstrap/permission_matrix_workspace_test.go`

**Interfaces:**（spec 2.5、第 3 节第 15 条；M4 设计 9.2）
- `permission_matrix_issue_columns_test.go`（新）：列 `callerCreator`、`callerGuestAll`、`callerGuestOwn`、`callerGuestOthers`、`callerAdminGuest`、`callerAdminGuestOwn`；`issueColumns`（17 列，9.2 的次序）；`issueOf(c)`（列的格子指名的工作项）、`toWorkItem(method, after, body)`；`onAWorkItem(admin, member, creator, guest)`、`onTheIssues(admin, member, guest)`（项目一级的一行的格子（`issueRows.of`、`ofProject`）去掉 PG 的，放进工作项表自己的六列的：`issueTable`）。
- `permission_matrix_seed_issues_test.go`：`ownIssues`（PM 的；种子经存储记在工作区访客名下的，访客不能创建（`createIssue`），所以访客自己的工作项是他从前作为成员建的：判定只读创建者和他现在的角色，不读它怎样来的；在公开项目里是访客的工作区管理员作为管理员建的；已离开私有项目的成员离开之前建的）；`guestViewAll`（acme 的私有项目让访客看每个工作项，由它的管理员 PA 经项目的存储设：acme 的管理员不是 acme 的任何项目的成员，规则不让他设；公开项目照默认不让）；`seededItem`、`seededItems()`、`numberOf`、`nextNumber`；`issueSeed.issues` 写全部，`seededIssues` 读回每个工作项的编号、名称、创建者，以及工作项表自己的每一列靠的事实（列的账户在它的项目里是有效成员，建了它的项目成员是成员、其余是访客，只有访客·全部的项目"可见全部"；管理员·访客在 acme 是有效的管理员）。
- `permission_matrix_seeded_test.go`、`permission_matrix_targets_test.go`：每个种下的工作项一个 `id`，`targetViolation` 核对 `{issue_id}` 是它的列的工作项；新身份的账户（`permission_matrix_columns_test.go`，它最先注册，在 acme 的管理员之前：账户的 id 随注册增大，两格"别人的"的创建者因此一格在调用者之后、一格在之前）和它们在工作区、成员、邀请的表里的格子；`permission_matrix_coverage_test.go` 认工作项表；`permission_matrix_test.go` 的种子的说明。
- `permission_matrix_issues_test.go`、`permission_matrix_issue_updates_test.go`：`getIssue`、`createIssue`、`updateIssue` 的行换到工作项表上（`createIssue` 的编号是 `nextNumber`）。

**Tests:**
- 组合：`TestPermissionMatrix`（工作项表的 `getIssue`、`createIssue`、`updateIssue` 各行的 17 列）、`TestMatrixViolationsCatchesAnotherCasesWorkItem`（一格指名别的情形的工作项是违规）、`TestEveryColumnCallsAsARegisteredAccount`、`TestThePermissionMatrixCoversEveryOperation`、`TestMatrixViolationsCatchesEachGap`。

- [ ] **Step 1: 列、种子和目标**

`server/internal/bootstrap/permission_matrix_columns_test.go`（修改，5 处）：

````old server/internal/bootstrap/permission_matrix_columns_test.go
// matrixAccounts are the accounts prepareMatrix registers: each workspace
// column's, and each project column's that is none of those (M3 design
// 9.2: eleven).
var matrixAccounts = append(slices.Clone(workspaceColumns), callerProjectAdmin, callerProjectMember, callerMemberAndAdmin, callerGuestOnly,
	callerBefore)
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
// matrixAccounts are the accounts prepareMatrix registers, in this order:
// the issue table's workspace admin who is only a guest of acme's public
// project (M4 design 9.2), then each workspace column's, and each project
// column's that is none of those (M3 design 9.2: eleven): twelve. He comes
// before acme's admin, who made the work item that the issue table's two
// columns on another's name (issueOf), and the guest after acme's admin:
// an account's id grows as accounts register (uuid.NewV7), so that the
// maker's id is after one caller's and before the other's, and a guest
// rule that told another's work item by the ids' order, not by its maker,
// fails one of the two.
var matrixAccounts = slices.Concat([]caller{callerAdminGuest}, workspaceColumns, []caller{callerProjectAdmin, callerProjectMember,
	callerMemberAndAdmin, callerGuestOnly, callerBefore})
````
````old server/internal/bootstrap/permission_matrix_columns_test.go
	case callerProjectGuest:
		return callerGuest
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
	case callerProjectGuest, callerGuestAll, callerGuestOwn, callerGuestOthers:
		return callerGuest
	case callerCreator:
		return callerProjectMember
	case callerAdminGuestOwn:
		return callerAdminGuest
````
````old server/internal/bootstrap/permission_matrix_columns_test.go
// acme's private project for the columns about it, its archived one for
// the archived project's, gone's for the deleted workspace's, acme's public
// one for every other; PA, PM, PG and PM+WA answer alike in both, WG- too
// (9.2).
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
// acme's private project for the columns about it, and for the guest who
// sees every work item, which it shows its guests (guestViewAll); its
// archived one for the archived project's, gone's for the deleted
// workspace's, acme's public one for every other; PA, PM and PM+WA
// answer alike in both, PG too but on another's work item, which only
// the private one shows him (callerGuestAll), and WG- too (9.2).
````
````old server/internal/bootstrap/permission_matrix_columns_test.go
	case callerMemberPrivate, callerBefore:
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
	case callerMemberPrivate, callerBefore, callerGuestAll:
````
````old server/internal/bootstrap/permission_matrix_columns_test.go
	if len(matrixAccounts) != 11 {
		t.Errorf("%d accounts, want 9.2's 11", len(matrixAccounts))
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
	if len(matrixAccounts) != 12 {
		t.Errorf("%d accounts, want M3 9.2's 11 and M4 9.2's workspace admin who is a project guest", len(matrixAccounts))
````

`server/internal/bootstrap/permission_matrix_coverage_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_coverage_test.go
// projectTables are the tables of the project level (M3 design 9.2), each
// whole: a row that names a project names one of them.
var projectTables = [][]caller{projectColumns, archivedColumns}
````
````new server/internal/bootstrap/permission_matrix_coverage_test.go
// projectTables are the tables of the project level (M3 design 9.2, and
// M4 design 9.2's of the work items, issueColumns), each whole: a row that
// names a project names one of them.
var projectTables = [][]caller{projectColumns, archivedColumns, issueColumns}
````

`server/internal/bootstrap/permission_matrix_invitations_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_invitations_test.go
// member; acme: its admin, member and guest, and the project level's five
````
````new server/internal/bootstrap/permission_matrix_invitations_test.go
// member; acme: its admin, member and guest, and the project level's six
````
````old server/internal/bootstrap/permission_matrix_invitations_test.go
	want := map[string]int{"other": 3, "acme": 9}[ownInvitation(c)]
````
````new server/internal/bootstrap/permission_matrix_invitations_test.go
	want := map[string]int{"other": 3, "acme": 10}[ownInvitation(c)]
````

`server/internal/bootstrap/permission_matrix_issue_columns_test.go`（新文件，130 行）：

````file server/internal/bootstrap/permission_matrix_issue_columns_test.go
package bootstrap

import (
	"fmt"
	"maps"
	"net/http"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
)

// The issue table of the permission matrix (M4 design 9.2): the project
// level's columns, its PG as the project's guest in each of his three
// cases, beside the member who made the work item and the workspace admin
// who is only a guest of the project; and the work item each column's
// cells name.

// The issue table's columns that are no other table's. Each calls as an
// account of matrixAccounts (accountOf), and names the work item its case
// is about (issueOf): acme's public project does not show its guests every
// work item, as a project starts, and its private one does (guestViewAll),
// so that the guest's three cases are one account's.
const (
	callerCreator       caller = "project member who made it"                 // 建了它的项目成员: PM, on his own
	callerGuestAll      caller = "project guest, every work item"             // 访客·全部: PG, in the private project
	callerGuestOwn      caller = "project guest, his own"                     // 访客·自己: PG, on his own
	callerGuestOthers   caller = "project guest, another's"                   // 访客·别人: PG, on acme's admin's
	callerAdminGuest    caller = "workspace admin and project guest"          // 管理员·访客: on acme's admin's
	callerAdminGuestOwn caller = "workspace admin and project guest, his own" // the same account, on his own
)

// issueColumns are the columns of the issue table, in the order of 9.2's
// table: PA, PM and the member who made the work item; the guest who sees
// every work item, and the guest who sees only his own, on his own and on
// another's; the workspace admin who is a guest of the project, on
// another's and on his own; then the project level's others, as
// projectColumns has them.
var issueColumns = []caller{callerProjectAdmin, callerProjectMember, callerCreator, callerGuestAll, callerGuestOwn, callerGuestOthers,
	callerAdminGuest, callerAdminGuestOwn, callerMemberAndAdmin, callerAdminOnly, callerMemberPublic, callerMemberPrivate, callerGuestOnly,
	callerBefore, callerNever, callerRemoved, callerDeleted}

// issueOf is the name of the work item of its project (projectOf) that a
// column's cells name: its own, made by its account, for a column of
// ownIssues; its project's matrixIssue, made by the workspace's admin,
// for every other.
func issueOf(c caller) string {
	for _, o := range ownIssues {
		if o.c == c {
			return o.name
		}
	}
	return matrixIssue
}

// toWorkItem is the request of a row whose callers each send method, with
// body, to their column's work item (issueOf), after its id.
func toWorkItem(method, after, body string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return issueRows.to(method, after, issueOf(c), body)(c, s)
	}
}

// onAWorkItem are the cells of a row of the issue table on a work item: a
// project-level row's on one (issueRows.of), but PG's, whose cases are
// the issue table's own columns (issueTable). The answers of a caller who
// acts on it as an admin of its project (PA; PM+WA, and the workspace
// admin who is a guest, on his own: a workspace admin with a membership
// passes each rule of the project, Grant.ProjectAdmin), of PM, of the
// member who made it, and of the project's guest who sees it, every work
// item or his own; issue.not_found for each column that does not see it:
// the guest and the workspace admin who is a guest, on another's, whom
// the guest rule hides it from before any role (M4 design 3.4), and
// WM-私, WG-, P-前 and X; forbidden for WA- and WM-公, who see the project
// and are none of its members.
func onAWorkItem(admin, member, creator, guest cell) map[caller]cell {
	return issueTable(issueRows.of(admin, member, guest, admin, cellForbidden, cellForbidden), map[caller]cell{callerCreator: creator,
		callerGuestAll: guest, callerGuestOwn: guest, callerGuestOthers: cellIssueNotFound, callerAdminGuest: cellIssueNotFound,
		callerAdminGuestOwn: admin})
}

// onTheIssues are the cells of a row of the issue table on its project's
// work items, none named (createIssue): a project-level row's on the
// project (ofProject), but PG's, whose cases are the issue table's own
// columns (issueTable). The answers of a caller who acts as an admin of
// the project (PA, PM+WA, and the workspace admin who is a guest, in
// either column), of its members (PM, and the member who made a work
// item) and of its guests, in each of their columns; project.not_found
// for WM-私, WG-, P-前 and X, who do not see the project; forbidden for
// WA- and WM-公.
func onTheIssues(admin, member, guest cell) map[caller]cell {
	return issueTable(ofProject(admin, member, guest, admin, cellForbidden, cellForbidden), map[caller]cell{callerCreator: member,
		callerGuestAll: guest, callerGuestOwn: guest, callerGuestOthers: guest, callerAdminGuest: admin, callerAdminGuestOwn: admin})
}

// issueTable is a row of the issue table made from project's, a
// project-level row's cells: PG's taken out, and own's, the cells of the
// issue table's own columns, put in.
func issueTable(project, own map[caller]cell) map[caller]cell {
	delete(project, callerProjectGuest)
	maps.Copy(project, own)
	return project
}

// A cell names its column's own work item (issueOf), not only one of its
// column's project: the guest's own column naming acme's admin's work item
// would ask the guest rule's other case under its name, and its 404 could
// pass for his own.
func TestMatrixViolationsCatchesAnotherCasesWorkItem(t *testing.T) {
	s := newSeeded().in(t)
	getIssue := apitest.Operation{ID: "getIssue", Tags: []string{"issue"}, Method: http.MethodGet, Path: "/api/v0/issues/{issue_id}"}
	row := matrixRow{op: getIssue.ID, columns: issueColumns, request: toWorkItem(http.MethodGet, "", ""),
		cells: onAWorkItem(cellOK, cellOK, cellOK, cellOK)}
	ops := []apitest.Operation{getIssue}
	if got := matrixViolations(ops, matrixExemptions{}, []matrixRow{row}, s, nil); len(got) != 0 {
		t.Fatalf("a matching matrix: %q, want none", got)
	}
	another := row
	another.request = func(c caller, s seeded) (string, string, string) {
		if c == callerGuestOwn {
			return row.request(callerGuestOthers, s)
		}
		return row.request(c, s)
	}
	want := []string{fmt.Sprintf("row getIssue, %s: {issue_id} %s is not its column's work item Guest's, %s", callerGuestOwn,
		s.issue("acme/public", matrixIssue), s.issue("acme/public", "Guest's"))}
	if got := matrixViolations(ops, matrixExemptions{}, []matrixRow{another}, s, nil); !slices.Equal(got, want) {
		t.Errorf("the guest's own column at acme's admin's work item: %q, want %q", got, want)
	}
}
````

`server/internal/bootstrap/permission_matrix_members_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_members_test.go
// workspace's guest (PG's account) and PM+WA. Not the removed member,
````
````new server/internal/bootstrap/permission_matrix_members_test.go
// workspace's guest (PG's account), PM+WA and the workspace admin who is
// only its guest (issueColumns). Not the removed member,
````
````old server/internal/bootstrap/permission_matrix_members_test.go
	}{{callerProjectAdmin, 20}, {callerProjectMember, 15}, {callerGuest, 5}, {callerMemberAndAdmin, 15}}
````
````new server/internal/bootstrap/permission_matrix_members_test.go
	}{{callerProjectAdmin, 20}, {callerProjectMember, 15}, {callerGuest, 5}, {callerMemberAndAdmin, 15}, {callerAdminGuest, 5}}
````

`server/internal/bootstrap/permission_matrix_seed_issues_test.go`（整个文件，191 行）：

````whole server/internal/bootstrap/permission_matrix_seed_issues_test.go
package bootstrap

import (
	"context"
	"slices"
	"strings"
	"uuid"

	issuepg "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	issuedomain "github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	projectdomain "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// matrixIssue is the work item prepareMatrix seeds in each project of
// matrixProjects through the issue store, by its workspace's admin, its
// project's first. gone's is deleted with it.
const matrixIssue = "Seeded"

// ownIssues are the work items prepareMatrix seeds besides, each the one
// a column's cells name (issueOf), whose maker the column's case is about:
// made by the column's account (accountOf) in its project (projectOf), in
// this order, after its project's matrixIssue. PM's, the member who made
// it; the workspace guest's, stored with him as its maker: a guest makes
// none (createIssue), so a guest's own work item is one he made as a
// member before; no decision reads how it came to be, only its creator and
// his role now; the workspace admin's who is the project's guest, who may
// make one as an admin of it (M4 design 9.2); and, in the private project,
// the member before's, made before he left it (endings): 9.2's member who
// made it and has left the project.
var ownIssues = []struct {
	c    caller
	name string
}{{callerCreator, "Member's"}, {callerGuestOwn, "Guest's"}, {callerAdminGuestOwn, "Admin guest's"}, {callerBefore, "Before's"}}

// guestViewAll is the project that shows its guests every work item
// (guest_view_all_features), as its admin, PA, chose: acme's private one,
// for the guest who sees them all (callerGuestAll); acme's public one
// does not, as a project starts (M4 design 3.4).
const guestViewAll = "acme/private"

// seededItem is a work item issueSeed stores: its project's key, its name
// and its maker.
type seededItem struct {
	key, name string
	by        caller
}

// seededItems are the work items issueSeed stores, in order: each
// project's matrixIssue, by its workspace's admin, then ownIssues.
func seededItems() []seededItem {
	var items []seededItem
	for _, p := range matrixProjects {
		slug, _, _ := strings.Cut(p.key, "/")
		items = append(items, seededItem{p.key, matrixIssue, matrixAdmins[slug]})
	}
	for _, o := range ownIssues {
		items = append(items, seededItem{projectOf(o.c), o.name, accountOf(o.c)})
	}
	return items
}

// numberOf is the number of the seeded work item name of the project key,
// its place among its project's in seededItems' order, and its maker; 0
// and none for a name never seeded.
func numberOf(key, name string) (int, caller) {
	n := 0
	for _, i := range seededItems() {
		if i.key == key {
			n++
			if i.name == name {
				return n, i.by
			}
		}
	}
	return 0, ""
}

// nextNumber is the number of the project key's next work item, the one a
// creation makes: one past the seeded ones.
func nextNumber(key string) int {
	n := 1
	for _, i := range seededItems() {
		if i.key == key {
			n++
		}
	}
	return n
}

// issueSeed writes the prepared work items through the issue store, each
// numbered through the project store, as createIssue numbers it.
type issueSeed struct {
	projectSeed
	store *issuepg.Store
}

// issues makes guestViewAll show its guests every work item, through the
// project store, by its admin, PA (updateProject; acme's admin, a member
// of no project of acme, may not); then stores each of seededItems with
// the id sd names, by its maker: its number, its row, its assignees' rows
// in id order, and its label's. Each is in Todo, medium, assigned to PM's
// account and to the removed member's, whose membership of acme and of
// its public project ended since, and labelled Bug.
func (s issueSeed) issues(sd seeded) {
	s.t.Helper()
	ctx := context.Background()
	on := true
	if err := s.projectSeed.store.UpdateProject(ctx, s.projects[guestViewAll], projectdomain.ProjectPatch{GuestViewAllFeatures: &on},
		s.ids[callerProjectAdmin], s.now); err != nil {
		s.t.Fatal(err)
	}
	assignees := slices.SortedFunc(slices.Values([]uuid.UUID{s.ids[callerProjectMember], s.ids[callerRemoved]}), uuid.UUID.Compare)
	for _, item := range seededItems() {
		slug, _, _ := strings.Cut(item.key, "/")
		sequence, err := s.projectSeed.store.NextIssueSequence(ctx, s.projects[item.key])
		if err != nil {
			s.t.Fatal(err)
		}
		id, by := sd.issue(item.key, item.name), s.ids[item.by]
		if err := s.store.InsertIssue(ctx, issueapp.IssueRow{ID: id, WorkspaceID: s.workspaces[slug], ProjectID: s.projects[item.key],
			SequenceID: sequence, Name: item.name, StateID: sd.state(item.key, "Todo"), Priority: issuedomain.PriorityMedium, SortOrder: 65535,
			By: by, At: s.now}); err != nil {
			s.t.Fatal(err)
		}
		for _, link := range []struct {
			insert  func(context.Context, issueapp.LinkRows) error
			targets []uuid.UUID
		}{{s.store.InsertAssignees, assignees}, {s.store.InsertLabels, []uuid.UUID{sd.label(item.key, "Bug")}}} {
			rows := issueapp.LinkRows{WorkspaceID: s.workspaces[slug], ProjectID: s.projects[item.key], IssueID: id, Targets: link.targets,
				By: by, At: s.now}
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
// the issue store by the ids sd names: each of seededItems in its project,
// of its number and name, made by its maker, in Todo, medium, assigned to
// PM's account and the removed member's, in id order, labelled Bug;
// undeleted, but gone's, deleted with gone. Then, through the project
// store, the facts each of the issue table's own columns (issueColumns
// but projectColumns) rests on, as the guest rule and AllowCreator read
// them: its account an active member of its project (accountOf,
// projectOf), the member who made a work item as a member, every other as
// a guest; and the project showing its guests every work item for the
// guest who sees them all alone (guestViewAll). Last, through the
// workspace store, the workspace admin who is a guest acme's active admin.
// One missing, or stored otherwise, would let a cell answer as it wants
// for another reason, or a column call as another case's account.
func (s issueSeed) seededIssues(sd seeded) {
	s.t.Helper()
	ctx := context.Background()
	assignees := slices.SortedFunc(slices.Values([]uuid.UUID{s.ids[callerProjectMember], s.ids[callerRemoved]}), uuid.UUID.Compare)
	for _, item := range seededItems() {
		gone := strings.HasPrefix(item.key, "gone/")
		number, _ := numberOf(item.key, item.name)
		i, found, err := s.store.GetIssue(ctx, sd.issue(item.key, item.name))
		if err != nil || found == gone || !gone && (i.ProjectID != s.projects[item.key] || i.SequenceID != number || i.Name != item.name ||
			i.CreatedByID == nil || *i.CreatedByID != s.ids[item.by] || i.StateID != sd.state(item.key, "Todo") ||
			i.Priority != issuedomain.PriorityMedium || !slices.Equal(i.AssigneeIDs, assignees) ||
			!slices.Equal(i.LabelIDs, []uuid.UUID{sd.label(item.key, "Bug")})) {
			s.t.Fatalf("%s's work item %s = %+v, %v, %v; want its %d, by %s, in Todo, medium, assigned to %v, labelled Bug, deleted %v", item.key,
				item.name, i, found, err, number, item.by, assignees, gone)
		}
	}
	for _, c := range issueColumns {
		if slices.Contains(projectColumns, c) {
			continue
		}
		role, viewAll := shared.RoleGuest, c == callerGuestAll
		if c == callerCreator {
			role = shared.RoleMember
		}
		f, found, err := s.projectSeed.store.ProjectFacts(ctx, s.projects[projectOf(c)], s.ids[accountOf(c)])
		if err != nil || !found || !f.Member || f.Role != role || f.GuestViewAll != viewAll {
			s.t.Fatalf("%s's facts of %s = %+v, %v, %v; want its account an active member of role %d, every work item shown %v", c,
				projectOf(c), f, found, err, role, viewAll)
		}
	}
	if role, active, err := s.matrixSeed.store.ActiveRole(ctx, sd.workspace("acme"), s.ids[accountOf(callerAdminGuest)]); err != nil ||
		!active || role != shared.RoleAdmin {
		s.t.Fatalf("%s's role in acme = %d, %v, %v; want its active admin", callerAdminGuest, role, active, err)
	}
}
````

`server/internal/bootstrap/permission_matrix_seeded_test.go`（修改，5 处）：

````old server/internal/bootstrap/permission_matrix_seeded_test.go
	{"acme", callerGuestOnly, shared.RoleGuest}, {"acme", callerBefore, shared.RoleMember},
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
	{"acme", callerGuestOnly, shared.RoleGuest}, {"acme", callerBefore, shared.RoleMember}, {"acme", callerAdminGuest, shared.RoleAdmin},
````
````old server/internal/bootstrap/permission_matrix_seeded_test.go
// private one, for partingStates to delete; the archived project's admin,
// PM, a member whose role he changes and whom he removes, and PM+WA, its
// other admin, so that he may leave it; each other workspace's admin in its
// project, and gone's member in gone's, a member of the project the X
// columns' writes on a membership name there.
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
// private one, for partingStates to delete; the workspace admin who is
// only a guest of the public one (issueColumns); the archived project's
// admin, PM, a member whose role he changes and whom he removes, and
// PM+WA, its other admin, so that he may leave it; each other workspace's
// admin in its project, and gone's member in gone's, a member of the
// project the X columns' writes on a membership name there.
````
````old server/internal/bootstrap/permission_matrix_seeded_test.go
	{"acme/public", callerRemoved, shared.RoleMember}, {"acme/public", callerGuestOnly, shared.RoleGuest},
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
	{"acme/public", callerRemoved, shared.RoleMember}, {"acme/public", callerGuestOnly, shared.RoleGuest},
	{"acme/public", callerAdminGuest, shared.RoleGuest},
````
````old server/internal/bootstrap/permission_matrix_seeded_test.go
// project and its work item, matrixIssue, before prepareMatrix writes them,
// so that matrixViolations, without a database, sees the keys and the
// targets the cells will.
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
// project, and each work item of seededItems, before prepareMatrix writes
// them, so that matrixViolations, without a database, sees the keys and
// the targets the cells will.
````
````old server/internal/bootstrap/permission_matrix_seeded_test.go
		s.issues[p.key+"|"+matrixIssue] = uuid.NewV7()
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
	}
	for _, i := range seededItems() {
		s.issues[i.key+"|"+i.name] = uuid.NewV7()
````

`server/internal/bootstrap/permission_matrix_targets_test.go`（修改，5 处）：

````old server/internal/bootstrap/permission_matrix_targets_test.go
// which it must be, keyed by their project's key.
````
````new server/internal/bootstrap/permission_matrix_targets_test.go
// which it must be, keyed by their project's key; and, when set, the one
// row of them a column's cells may name, and its name: a work item is its
// column's (issueOf), whose maker the column's case may be about.
````
````old server/internal/bootstrap/permission_matrix_targets_test.go
	rows func(s seeded) map[string]uuid.UUID
````
````new server/internal/bootstrap/permission_matrix_targets_test.go
	rows func(s seeded) map[string]uuid.UUID
	one  func(c caller, s seeded) (string, uuid.UUID)
````
````old server/internal/bootstrap/permission_matrix_targets_test.go
	"{project_member_id}": {"membership", func(s seeded) map[string]uuid.UUID { return s.projectMembers }},
	"{state_id}":          {"state", func(s seeded) map[string]uuid.UUID { return s.states }},
	"{label_id}":          {"label", func(s seeded) map[string]uuid.UUID { return s.labels }},
	"{issue_id}":          {"work item", func(s seeded) map[string]uuid.UUID { return s.issues }},
````
````new server/internal/bootstrap/permission_matrix_targets_test.go
	"{project_member_id}": {"membership", func(s seeded) map[string]uuid.UUID { return s.projectMembers }, nil},
	"{state_id}":          {"state", func(s seeded) map[string]uuid.UUID { return s.states }, nil},
	"{label_id}":          {"label", func(s seeded) map[string]uuid.UUID { return s.labels }, nil},
	"{issue_id}": {"work item", func(s seeded) map[string]uuid.UUID { return s.issues }, func(c caller, s seeded) (string, uuid.UUID) {
		return issueOf(c), s.issue(projectOf(c), issueOf(c))
	}},
````
````old server/internal/bootstrap/permission_matrix_targets_test.go
// work item) one seeded in projectOf(c), each from a column of a project table
// (projectTables): a project's operation in a workspace-level row, or the
// only admin's, would leave the project level's own columns unasked.
````
````new server/internal/bootstrap/permission_matrix_targets_test.go
// work item) one seeded in projectOf(c), a work item the column's own
// (issueOf), each from a column of a project table (projectTables): a
// project's operation in a workspace-level row, or the only admin's, would
// leave the project level's own columns unasked.
````
````old server/internal/bootstrap/permission_matrix_targets_test.go
			if key, isRow := projectOfRow(underProject[segment].rows(s), id); err != nil || !isRow || key != projectOf(c) {
				return fmt.Sprintf("%s %s is no %s seeded in its column's project %s", segment, got[i], underProject[segment].what, projectOf(c))
````
````new server/internal/bootstrap/permission_matrix_targets_test.go
			under := underProject[segment]
			if key, isRow := projectOfRow(under.rows(s), id); err != nil || !isRow || key != projectOf(c) {
				return fmt.Sprintf("%s %s is no %s seeded in its column's project %s", segment, got[i], under.what, projectOf(c))
			}
			if under.one != nil {
				if name, one := under.one(c, s); id != one {
					return fmt.Sprintf("%s %s is not its column's %s %s, %s", segment, got[i], under.what, name, one)
				}
````

`server/internal/bootstrap/permission_matrix_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_test.go
	// (workspaceColumns), projectColumns for a row of the project level.
````
````new server/internal/bootstrap/permission_matrix_test.go
	// (workspaceColumns), or a table of matrixTables, as projectColumns.
````
````old server/internal/bootstrap/permission_matrix_test.go
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
````new server/internal/bootstrap/permission_matrix_test.go
// store, the work items of seededItems, once guestViewAll shows its guests
// every one (issueSeed); through the project store, acme's archived project
// archived. Through both stores, the removed member's removal; then,
// through the workspace store, the invitations of matrixInvitations.
// Through the project store, the memberships P5b's writes end (endings);
// through SQL, the membership states no store makes alone (partingStates).
// Through the API, gone deleted by its admin, which soft-deletes its
// memberships and its project, with its states, labels and work item, with
// it; then the checks that the rows the cells rest on are there
// (preconditions), the states, the labels and the work items among them
// (seededStates, seededLabels, seededIssues).
````

`server/internal/bootstrap/permission_matrix_workspace_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_workspace_test.go
// listsTheMembers: acme's nine memberships, the removed member's ended; the
````
````new server/internal/bootstrap/permission_matrix_workspace_test.go
// listsTheMembers: acme's ten memberships, the removed member's ended; the
````
````old server/internal/bootstrap/permission_matrix_workspace_test.go
		"workspace-guest-only", "project-member-before"} {
````
````new server/internal/bootstrap/permission_matrix_workspace_test.go
		"workspace-guest-only", "project-member-before", "workspace-admin-and-project-guest"} {
````

- [ ] **Step 2: 工作项的行换到工作项表上**

`server/internal/bootstrap/permission_matrix_issue_updates_test.go`（修改，16 处）：

````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
// 9.2), naming it by its id (/issues/{issue_id}): its column's project's
// seeded work item, matrixIssue. Each cell writes on a copy of its own.
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
// 9.2), naming it by its id (/issues/{issue_id}): its column's work item
// (issueOf). Each cell writes on a copy of its own. The row of 9.2's table
// has the issue table's columns; its variants, the project level's, as
// createIssue's (permission_matrix_issues_test.go).
````
````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
// project's seeded work item with the body body gives for their column.
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
// work item with the body body gives for their column.
````
````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
		return issueRows.to(http.MethodPatch, "", matrixIssue, body(c, s))(c, s)
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
		return toWorkItem(http.MethodPatch, "", body(c, s))(c, s)
````
````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
func issueUpdateMatrixRows() []matrixRow {
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
func issueUpdateMatrixRows() []matrixRow {
	// PG, the guest of a project that does not show its guests every work
	// item, on acme's admin's: issue.not_found, whatever his role may do.
````
````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
		// The project's admins and members, as createIssue (M4 design 3.4);
		// PG, a guest who sees only his own work items, finds no other's,
		// whatever his role may do: issue.not_found, not forbidden.
		{op: "updateIssue", write: true, columns: projectColumns, request: issueRows.to(http.MethodPatch, "", matrixIssue, `{"name":"Mend"}`),
			cells: ok, check: renamesTheIssue},
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
		// The project's admins and members, as createIssue (M4 design 3.4),
		// not a guest who sees the work item; a guest of a project that does
		// not show its guests every work item, the workspace admin who is
		// only its guest too, finds none another made, whatever his role may
		// do: issue.not_found, not forbidden.
		{op: "updateIssue", write: true, columns: issueColumns, request: toWorkItem(http.MethodPatch, "", `{"name":"Mend"}`),
			cells: onAWorkItem(cellOK, cellOK, cellOK, cellForbidden), check: renamesTheIssue},
````
````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
			request: issueRows.to(http.MethodPatch, "", matrixIssue, `{}`), cells: ok, check: readsTheIssue},
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
			request: toWorkItem(http.MethodPatch, "", `{}`), cells: ok, check: readsTheIssue},
````
````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
			request: issueRows.to(http.MethodPatch, "", matrixIssue, `{}`), cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
			request: toWorkItem(http.MethodPatch, "", `{}`), cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
````
````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
// seededIssue is the column's project's seeded work item as issueSeed
// stored it, as an answer shows it: its first, in Todo, medium, not
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
// seededIssue is the column's work item (issueOf) as issueSeed stored it,
// as an answer shows it: of its number (numberOf), in Todo, medium, not
````
````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
// order, labelled Bug, made and changed last by acme's admin.
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
// order, labelled Bug, made and changed last by its maker.
````
````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
	key, admin := projectOf(c), s.account(callerAdmin)
	return answeredIssue{ID: s.issue(key, matrixIssue), ProjectID: s.project(key), SequenceID: 1, Name: matrixIssue,
		StateID: s.state(key, "Todo"), Priority: "medium",
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
	key, name := projectOf(c), issueOf(c)
	number, by := numberOf(key, name)
	maker := s.account(by)
	return answeredIssue{ID: s.issue(key, name), ProjectID: s.project(key), SequenceID: number, Name: name, StateID: s.state(key, "Todo"),
		Priority:    "medium",
````
````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
		LabelIDs:    []uuid.UUID{s.label(key, "Bug")}, CreatedByID: &admin, UpdatedByID: &admin}
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
		LabelIDs:    []uuid.UUID{s.label(key, "Bug")}, CreatedByID: &maker, UpdatedByID: &maker}
````
````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
// answered checks that answer, the work item c's cell changes, is the
// seeded one with change made to it, the caller its last writer, and
// completed when completed; completedAt is not compared, but present or
// not.
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
// answered checks that answer, the work item c's cell changes, is the one
// seeded (seededIssue) with change made to it, the caller its last writer,
// and completed when completed; completedAt is not compared, but present
// or not.
````
````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
// readsTheIssue: the column's project's seeded work item as issueSeed
// stored it, in each field answeredIssue holds (seededIssue).
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
// readsTheIssue: the column's work item as issueSeed stored it, in each
// field answeredIssue holds (seededIssue).
````
````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
// renamesTheIssue: the seeded work item named Mend, the rest as it was.
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
// renamesTheIssue: the column's work item named Mend, the rest as it was.
````
````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
// changesEachField: the seeded work item named Mend, in Done and so
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
// changesEachField: the column's work item named Mend, in Done and so
````
````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
// keepsTheAssigneeWhoLeft: the seeded work item assigned to the removed
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
// keepsTheAssigneeWhoLeft: the column's work item assigned to the removed
````

`server/internal/bootstrap/permission_matrix_issues_test.go`（修改，8 处）：

````old server/internal/bootstrap/permission_matrix_issues_test.go
// naming it by its id (/issues/{issue_id}): its column's project's seeded
// work item, matrixIssue. Each creating cell writes on a copy of its own,
// so each creation is its project's second, after the seeded one; the
// reading cells share one.
````
````new server/internal/bootstrap/permission_matrix_issues_test.go
// naming it by its id (/issues/{issue_id}): its column's work item
// (issueOf). Each creating cell writes on a copy of its own, so each
// creation is its project's next work item after the seeded ones
// (nextNumber); the reading cells share one. A row of 9.2's table has the
// issue table's columns (issueColumns); a variant that asks what is
// checked after the decision, the project level's (projectColumns), whose
// PG, a guest, may create none; one on the archived project, its table's
// (archivedColumns).
````
````old server/internal/bootstrap/permission_matrix_issues_test.go
		// PM+WA as the member he is; a workspace admin or member who is no
		// member of the project sees it and may not.
		{op: "createIssue", write: true, columns: projectColumns, request: toProject(http.MethodPost, "/issues", newIssue),
			cells: ofProject(cellCreated, cellCreated, cellForbidden, cellCreated, cellForbidden, cellForbidden), check: createsTheIssue},
````
````new server/internal/bootstrap/permission_matrix_issues_test.go
		// PM+WA as the member he is, and the workspace admin who is only a
		// guest of the project as its admin (Grant.ProjectAdmin, 9.2); a
		// workspace admin or member who is no member of the project sees it
		// and may not.
		{op: "createIssue", write: true, columns: issueColumns, request: toProject(http.MethodPost, "/issues", newIssue),
			cells: onTheIssues(cellCreated, cellCreated, cellForbidden), check: createsTheIssue},
````
````old server/internal/bootstrap/permission_matrix_issues_test.go
		// The project's admins, members and guests; PG, a guest who sees only
		// his own work items, finds no other's, as one not there (M4 design
		// 3.4).
		{op: "getIssue", columns: projectColumns, request: issueRows.to(http.MethodGet, "", matrixIssue, ""),
			cells: issueRows.of(cellOK, cellOK, cellIssueNotFound, cellOK, cellForbidden, cellForbidden), check: readsTheIssue},
````
````new server/internal/bootstrap/permission_matrix_issues_test.go
		// The project's admins, members and guests (M4 design 3.4); a guest
		// of a project that does not show its guests every work item, the
		// workspace admin who is only its guest too, finds none another made,
		// as one not there.
		{op: "getIssue", columns: issueColumns, request: toWorkItem(http.MethodGet, "", ""), cells: onAWorkItem(cellOK, cellOK, cellOK, cellOK),
			check: readsTheIssue},
````
````old server/internal/bootstrap/permission_matrix_issues_test.go
		{op: "getIssue", variant: "archived", columns: archivedColumns, request: issueRows.to(http.MethodGet, "", matrixIssue, ""),
````
````new server/internal/bootstrap/permission_matrix_issues_test.go
		{op: "getIssue", variant: "archived", columns: archivedColumns, request: toWorkItem(http.MethodGet, "", ""),
````
````old server/internal/bootstrap/permission_matrix_issues_test.go
// createsTheIssue: the column's project's second work item, Fix the login,
// in its default state Backlog, of no priority, at a state's first sort
// order, not completed, assigned to no one, without labels, made and
// last changed by the caller.
````
````new server/internal/bootstrap/permission_matrix_issues_test.go
// createsTheIssue: the column's project's next work item (nextNumber), Fix
// the login, in its default state Backlog, of no priority, at a state's
// first sort order, not completed, assigned to no one, without labels,
// made and last changed by the caller.
````
````old server/internal/bootstrap/permission_matrix_issues_test.go
	me := s.account(accountOf(c))
	if i.ProjectID != s.project(projectOf(c)) || i.SequenceID != 2 || i.Name != "Fix the login" || i.StateID != s.state(projectOf(c), "Backlog") ||
		i.Priority != "none" || i.SortOrder != 65535 || i.CompletedAt != nil || len(i.AssigneeIDs) != 0 || len(i.LabelIDs) != 0 ||
		i.CreatedByID == nil || *i.CreatedByID != me || i.UpdatedByID == nil || *i.UpdatedByID != me {
		t.Errorf("%s creates %s; want %s's work item 2, Fix the login, in Backlog, none, at 65535, by %s", c, answer, projectOf(c), me)
````
````new server/internal/bootstrap/permission_matrix_issues_test.go
	me, next := s.account(accountOf(c)), nextNumber(projectOf(c))
	if i.ProjectID != s.project(projectOf(c)) || i.SequenceID != next || i.Name != "Fix the login" ||
		i.StateID != s.state(projectOf(c), "Backlog") || i.Priority != "none" || i.SortOrder != 65535 || i.CompletedAt != nil ||
		len(i.AssigneeIDs) != 0 || len(i.LabelIDs) != 0 || i.CreatedByID == nil || *i.CreatedByID != me || i.UpdatedByID == nil ||
		*i.UpdatedByID != me {
		t.Errorf("%s creates %s; want %s's work item %d, Fix the login, in Backlog, none, at 65535, by %s", c, answer, projectOf(c), next, me)
````
````old server/internal/bootstrap/permission_matrix_issues_test.go
// createsTheGivenIssue: the column's project's second work item, Fix, in
````
````new server/internal/bootstrap/permission_matrix_issues_test.go
// createsTheGivenIssue: the column's project's next work item, Fix, in
````
````old server/internal/bootstrap/permission_matrix_issues_test.go
	pm, ui := s.account(callerProjectMember), s.label(projectOf(c), "UI")
	if i.SequenceID != 2 || i.StateID != s.state(projectOf(c), "Done") || i.Priority != "high" || i.SortOrder != 65535 || i.CompletedAt == nil ||
		!slices.Equal(i.AssigneeIDs, []uuid.UUID{pm}) || !slices.Equal(i.LabelIDs, []uuid.UUID{ui}) {
		t.Errorf("%s creates %s; want %s's work item 2 in Done, high, at 65535, completed, assigned to %s, labelled %s", c, answer,
			projectOf(c), pm, ui)
````
````new server/internal/bootstrap/permission_matrix_issues_test.go
	pm, ui, next := s.account(callerProjectMember), s.label(projectOf(c), "UI"), nextNumber(projectOf(c))
	if i.SequenceID != next || i.StateID != s.state(projectOf(c), "Done") || i.Priority != "high" || i.SortOrder != 65535 ||
		i.CompletedAt == nil || !slices.Equal(i.AssigneeIDs, []uuid.UUID{pm}) || !slices.Equal(i.LabelIDs, []uuid.UUID{ui}) {
		t.Errorf("%s creates %s; want %s's work item %d in Done, high, at 65535, completed, assigned to %s, labelled %s", c, answer,
			projectOf(c), next, pm, ui)
````

- [ ] **Step 3: 测试和 lint**

Run: `go -C server test -count=1 -run 'TestPermissionMatrix|TestMatrixViolations|TestThePermissionMatrixCoversEveryOperation$|TestEveryColumnCallsAsARegisteredAccount$' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/bootstrap/permission_matrix_columns_test.go server/internal/bootstrap/permission_matrix_coverage_test.go server/internal/bootstrap/permission_matrix_invitations_test.go server/internal/bootstrap/permission_matrix_issue_columns_test.go server/internal/bootstrap/permission_matrix_issue_updates_test.go server/internal/bootstrap/permission_matrix_issues_test.go server/internal/bootstrap/permission_matrix_members_test.go server/internal/bootstrap/permission_matrix_seed_issues_test.go server/internal/bootstrap/permission_matrix_seeded_test.go server/internal/bootstrap/permission_matrix_targets_test.go server/internal/bootstrap/permission_matrix_test.go server/internal/bootstrap/permission_matrix_workspace_test.go
```
```bash
git commit -m "test(M4/P2): the matrix's work-item table, with the guest's three cases

The work items' rows take the issue table of M4 design 9.2: the member
who made the work item, the guest who sees every work item, and the
guest who sees only his own, on his own and on another's; the workspace
admin who is only a guest of the project, on another's and on his own;
the member who made one and has left the project. Each column's cells
name the work item its case is about, seeded by its account, and acme's
private project shows its guests every work item.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p2.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试，或没有替身的平台、领域、转换的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是生成器（`make gen-go`）和中英文翻译键（`make lint-web`）的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过，端到端的检查只在标出的 Task 的树上）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T3.1` | 矩阵的一格可以指名它的列的项目里任何种下的工作项，不必是它的列自己的 | `internal/bootstrap` 的 `TestMatrixViolationsCatchesAnotherCasesWorkItem`（`make test`） | 单元 |
| `T3.2` | 工作项表的"建了它的项目成员"一列用 PA 的账户调用，不是 PM 的 | `internal/bootstrap` 的 `TestPermissionMatrix`（`make test`） | 组合 |

本 Task 的检查也发现较早的 Task 的变异 `T1.5`、`T1.7`、`T1.8`、`T1.9`、`T2.7`、`T2.8`、`T2.11`（列在各自的 Task 的表里，标"（Task 3 起）"）：本 Task 做完之后，在本 Task 的树上重跑它们（`mutants_p2.py`，`MUT_LATER=3`）。

---

### Task 4: `deleteIssue`

**Files:**
- Create: `server/internal/bootstrap/issue_deletion_test.go`、`server/internal/modules/issue/adapter/http/delete_test.go`、`server/internal/modules/issue/adapter/postgres/delete_issue.go`、`server/internal/modules/issue/adapter/postgres/delete_issue_test.go`、`server/internal/modules/issue/adapter/postgres/queries/delete_issue.sql`、`server/internal/modules/issue/app/delete_issue.go`、`server/internal/modules/issue/app/delete_issue_test.go`
- Modify: `api/modules/issue.yaml`、`server/internal/bootstrap/gate_test.go`、`server/internal/bootstrap/issue_write_locks_test.go`、`server/internal/bootstrap/issue_writes_test.go`、`server/internal/bootstrap/permission_matrix_issue_updates_test.go`、`server/internal/bootstrap/project_deletion_test.go`、`server/internal/bootstrap/workspace_deletion_catalog_test.go`、`server/internal/bootstrap/workspace_deletion_test.go`、`server/internal/modules/issue/adapter/http/handler.go`、`server/internal/modules/issue/adapter/http/handler_test.go`、`server/internal/modules/issue/adapter/http/issues.go`、`server/internal/modules/issue/app/fakes_test.go`、`server/internal/modules/issue/app/guest_rule_test.go`、`server/internal/modules/issue/app/lock_test.go`、`server/internal/modules/issue/app/update_issue_test.go`、`server/internal/modules/issue/module.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/issue/adapter/http/gen/server.gen.go`、`server/internal/modules/issue/adapter/postgres/gen/delete_issue.sql.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**（spec 2.6、第 3 节第 6、7、13、14 条；M4 设计 3.5、3.6、3.8、5.1）
- 契约：`DELETE /api/v0/issues/{issue_id}`（`deleteIssue`，204，`x-problem-codes: [issue.not_found, forbidden, project.archived, server_busy]`），描述写明谁能删、访客规则、锁之前答的和锁下再判定的、已归档的项目和已归档的工作项、删掉的行。
- `issue/adapter/postgres/queries/delete_issue.sql`（新）：`DeleteIssueRow`、`DeleteIssueAssigneeRows`、`DeleteIssueLabelRows`、`DeleteIssueSubscriberRows`、`DeleteIssueActivityRows`：`deleted_at`、`updated_at` 是给的时刻，`updated_by_id` 是删除者，只写未删除的行（`deleted_at IS NULL`）。`delete_issue.go`（新）：`Store` 的五个方法（`app.IssueDeletion`）。
- `issue/app/delete_issue.go`（新）：`IssueDeletion{IssueID, By, At}`；`IssueRowsDeleter`；`issueRowSteps(d)`（全局的表序）；`DeleteIssueDeps{Locks, Rows, Clock}`；`DeleteIssue`、`NewDeleteIssue`、`Execute(ctx, id) error`：`Locks.run` 的 `write{issue: &id, action: domain.ActionDelete, notFound: domain.ErrIssueNotFound}`，锁下项目已归档 409，`Clock.Now()`，逐张表一条语句。
- `issue/adapter/http`：`handler.go` 的 `DeleteIssueUseCase`、`UseCases.DeleteIssue`；`issues.go` 的 `DeleteIssue`（204）。`issue/module.go` 接上 `DeleteIssue`。
- 测试的帮手：`fakes_test.go` 的 `fakeIssues` 记下删除的调用；`update_issue_test.go` 的 `lockedAs(action, id)`（锁下判定之前的调用，删除共用）；`guest_rule_test.go` 的 `TestADecisionOnAWorkItemNamesItsCreator` 带上删除（访客规则的两个表经修改核对 `Locks.run`，删除走同一条路），和创建者不明的工作项的删除；`lock_test.go` 的 `TestEachWriteOnAWorkItemEntersItsProjectsGate`，先只有删除一行（Task 7 加归档、恢复）。
- `bootstrap`：`issue_deletion_test.go`（新：`subItems`、`workItemKeys`、`deletedWorkItem`）；`workspace_deletion_catalog_test.go` 的 `foreignKey.deleted`、`idsUnder`、`asText`；`project_deletion_test.go` 的种子加一个在项目之前删除的工作项（PF.3），`noRowLastWrittenBy(t, pool, keys, id, by)`（`workspace_deletion_test.go` 照改）；`issue_write_locks_test.go` 的 `waits`、`mode`、每次发送新建目标，`deleteIssue` 一行；`issue_writes_test.go` 的连接测试在一个连接的池上删除第二个工作项；`gate_test.go` 的 `TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy`，先只有删除一行（Task 7 加归档、恢复）；矩阵的 `deleteIssue` 行。

**Tests:**
- 单元：`TestDeleteIssueDeletesTheWorkItemAndItsRows`、`TestDeleteIssueRefusals`、`TestDeleteIssueStopsAtAFailingPort`；`TestADecisionOnAWorkItemNamesItsCreator` 的删除两行（创建者不明的工作项：两次判定都不指名创建者，`AllowCreator` 不把调用者当作它的创建者）；`TestEachWriteOnAWorkItemEntersItsProjectsGate` 的删除一行（预检之后、任何事务之外进项目的写闸门，事务结束之后最后离开）。
- 存储：`TestTheStatementsOfADeletedWorkItemDeleteItsRows`（每条语句只删这个工作项未删除的行；更早的删除、别的工作项的行和每一列不动；失败的写答它的失败）。
- HTTP（`apitest.Main` 两个方向）：`TestDeleteIssue`、`TestDeleteIssueRefusals`。
- 组合：`TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`（按目录，`subItems` 豁免到 P3）、`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`（PF.3）、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`、`TestEachIssueWriteLocksItsProjectFirst` 的 `deleteIssue` 一行（等 `issues`，`SHARE`）、`TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`（删除的每条语句在它的事务的连接上：一个连接的池）、`TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy` 的删除一行（项目的一个位置被占、不排队时，删除立刻答 503 `server_busy`、`Retry-After: 1`，在等项目的行到锁超时之前）、`TestPermissionMatrix` 的 `deleteIssue` 行（9.2 的一行：管理员、建了它的成员 204，成员、访客 403，访客·别人、管理员·访客 404……；已归档的项目 409）、`TestThePermissionMatrixCoversEveryOperation`。

- [ ] **Step 1: 契约和语句，生成**

`api/modules/issue.yaml`（修改，1 处）：

````old api/modules/issue.yaml
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IssueDetail'
        default:
          $ref: '#/components/responses/Problem'
components:
````
````new api/modules/issue.yaml
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IssueDetail'
        default:
          $ref: '#/components/responses/Problem'
    delete:
      operationId: deleteIssue
      tags: [issue]
      summary: Delete a work item
      description: >-
        For the admins of its project, and for the member of it who created
        the work item, while he is an admin or a member of the project. A
        work item that does not exist, is deleted, whose project the caller
        does not see, or that he did not create as a guest who sees only
        his own (getIssue) answers issue.not_found, whatever his role may
        do; one whose project he sees but that he may not delete,
        forbidden; both are answered before the project is locked, and with
        the project locked the role is decided again. A work item of an
        archived project cannot be deleted (project.archived); an archived
        work item is deleted as any other. The work item, its assignees,
        labels, subscribers and activity are deleted by the caller at one
        moment; its number is not used again. A project with too many
        writes on its work items at once answers server_busy, with
        Retry-After: try again then.
      security: [{bearer: []}]
      x-problem-codes: [issue.not_found, forbidden, project.archived, server_busy]
      responses:
        '204':
          description: The work item is deleted.
        default:
          $ref: '#/components/responses/Problem'
components:
````

`server/internal/modules/issue/adapter/postgres/queries/delete_issue.sql`（新文件，35 行）：

````file server/internal/modules/issue/adapter/postgres/queries/delete_issue.sql
-- name: DeleteIssueRow :exec
-- deleteIssue's work item (M4 design 3.6), under its project's FOR NO KEY UPDATE, read again under it, archived or not:
-- soft-deleted at the moment and by the account given. Every write of a work item's rows holds its project's lock first
-- (3.5), so the row is as the use case read it; it is not locked by id first (M3 design 3.6, convention 5).
UPDATE issues
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE id = sqlc.arg(issue_id) AND deleted_at IS NULL;

-- name: DeleteIssueAssigneeRows :exec
-- The deleted work item's assignees (M4 design 3.6): its undeleted rows, soft-deleted at the moment and by the account
-- given, in one statement, which locks them in its scan's order. Every write of these rows holds the project's FOR NO
-- KEY UPDATE first (3.5): no other write reaches them meanwhile, so they are not locked by id first (M3 design 3.6,
-- convention 5). A row deleted before keeps its deletion. The index on issue_id serves it.
UPDATE issue_assignees
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE issue_id = sqlc.arg(issue_id) AND deleted_at IS NULL;

-- name: DeleteIssueLabelRows :exec
-- DeleteIssueAssigneeRows' statement, of the deleted work item's labels.
UPDATE issue_labels
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE issue_id = sqlc.arg(issue_id) AND deleted_at IS NULL;

-- name: DeleteIssueSubscriberRows :exec
-- DeleteIssueAssigneeRows' statement, of the deleted work item's subscribers.
UPDATE issue_subscribers
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE issue_id = sqlc.arg(issue_id) AND deleted_at IS NULL;

-- name: DeleteIssueActivityRows :exec
-- DeleteIssueAssigneeRows' statement, of the deleted work item's activity (3.6: it goes with its work item); the index
-- on (issue_id, created_at, id) serves it.
UPDATE issue_activities
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE issue_id = sqlc.arg(issue_id) AND deleted_at IS NULL;
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `28dc326bed4a35724732bd8c90511cd19701ba6a5eae4acf0bd9ed3f5cfa4d29` | 3216 | `api/dist/openapi.yaml` |
| `53adfce82114ae1a7045431e7d52227dc1b98b41a277bc70b034af16586788d8` | 761 | `server/internal/modules/issue/adapter/http/gen/server.gen.go` |
| `5867354950b81604c6785ba428ce15a4abe02018f6251645e4729e4b7d3c782f` | 109 | `server/internal/modules/issue/adapter/postgres/gen/delete_issue.sql.go` |
| `ce7acf7ba41ae6677c2e57e12396197a72c45b3ad89d27af6a4486ccc504234a` | 3518 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/issue/adapter/http/gen/server.gen.go server/internal/modules/issue/adapter/postgres/gen/delete_issue.sql.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: 用例**

`server/internal/modules/issue/app/delete_issue.go`（新文件，92 行）：

````file server/internal/modules/issue/app/delete_issue.go
package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// IssueDeletion is a work item's deletion as its rows take it: the work
// item, and the moment and the account that every row it deletes is
// stamped with (M4 design 3.6).
type IssueDeletion struct {
	IssueID uuid.UUID
	By      uuid.UUID
	At      time.Time
}

// IssueRowsDeleter soft-deletes a deleted work item's rows, one table a
// call, in the transaction ctx carries, under its project's lock (M4
// design 3.5, 3.6): each method soft-deletes the work item's undeleted
// rows of its table at d.At by d.By, and leaves a row deleted before as it
// is.
type IssueRowsDeleter interface {
	DeleteIssueRow(ctx context.Context, d IssueDeletion) error
	DeleteIssueAssigneeRows(ctx context.Context, d IssueDeletion) error
	DeleteIssueLabelRows(ctx context.Context, d IssueDeletion) error
	DeleteIssueSubscriberRows(ctx context.Context, d IssueDeletion) error
	DeleteIssueActivityRows(ctx context.Context, d IssueDeletion) error
}

// issueRowSteps are d's statements of a work item's deletion in the global
// order of M4 design 3.5, as a project's (rowSteps): the work item, then
// the rows under it.
func issueRowSteps(d IssueRowsDeleter) []func(context.Context, IssueDeletion) error {
	return []func(context.Context, IssueDeletion) error{
		d.DeleteIssueRow, d.DeleteIssueAssigneeRows, d.DeleteIssueLabelRows, d.DeleteIssueSubscriberRows, d.DeleteIssueActivityRows,
	}
}

// DeleteIssueDeps are what deleteIssue writes through.
type DeleteIssueDeps struct {
	Locks Locks
	Rows  IssueRowsDeleter
	Clock Clock
}

// DeleteIssue deletes a work item: DELETE /api/v0/issues/{issue_id} (M4
// design 3.5, 3.6, 5.1).
type DeleteIssue struct {
	d DeleteIssueDeps
}

// NewDeleteIssue returns the use case.
func NewDeleteIssue(d DeleteIssueDeps) *DeleteIssue {
	return &DeleteIssue{d: d}
}

// Execute runs, through Locks, the deletion of the work item id (M4 design
// 3.5, 3.6, 11.11): its precheck, the work item read and the decision on
// issue.delete on it, the guest rule first, before any lock: the
// project's admins may, and the member who made it (AllowCreator); then
// the project's write gate (503 server_busy); then in one transaction, the
// project FOR NO KEY UPDATE, the work item read again and the decision
// again, and under them: an archived project refuses it (409
// project.archived); the clock; the work item and each row under it
// soft-deleted, table by table in the global order (issueRowSteps), all of
// it the caller's at that moment. An archived work item is deleted as any
// other (3.8). It deletes no other work item: its sub-items come with P3
// (3.9). A failing statement ends it with its failure, and none after it
// runs.
func (u *DeleteIssue) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	w := write{issue: &id, action: domain.ActionDelete, notFound: domain.ErrIssueNotFound}
	return u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		if h.project.Archived {
			return domain.ErrProjectArchived
		}
		d := IssueDeletion{IssueID: id, By: actor.UserID, At: u.d.Clock.Now()}
		for _, step := range issueRowSteps(u.d.Rows) {
			if err := step(ctx, d); err != nil {
				return err
			}
		}
		return nil
	})
}
````

`server/internal/modules/issue/app/delete_issue_test.go`（新文件，91 行）：

````file server/internal/modules/issue/app/delete_issue_test.go
package app_test

import (
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newDelete is deleteIssue over newUpdate's world: web's work item 7, made
// by carol, assigned to bob and out, labelled Bug.
func newDelete() (*app.DeleteIssue, *world, uuid.UUID) {
	_, w, id := newUpdate()
	return app.NewDeleteIssue(app.DeleteIssueDeps{Locks: w.locks, Rows: w.issues, Clock: clockAt{at: clockNow, log: w.log}}), w, id
}

// deletingRows are the calls of the deletion of the work item id's rows,
// in the global order of M4 design 3.5: the work item, its assignees, its
// labels, its subscribers, its activity.
func deletingRows(id uuid.UUID) []string {
	return []string{line("DeleteIssueRow", id), line("DeleteIssueAssigneeRows", id), line("DeleteIssueLabelRows", id),
		line("DeleteIssueSubscriberRows", id), line("DeleteIssueActivityRows", id)}
}

// A deletion, in the order of M4 design 3.5: Locks' calls, its decisions
// on issue.delete (lockedAs); the clock; the work item's rows, table by
// table in the global order (deletingRows), each alice's at the moment.
func TestDeleteIssueDeletesTheWorkItemAndItsRows(t *testing.T) {
	uc, w, id := newDelete()
	if err := uc.Execute(as(alice), id); err != nil {
		t.Fatal(err)
	}
	if calls := slices.Concat(lockedAs(domain.ActionDelete, id), []string{"Now"}, deletingRows(id)); !slices.Equal(w.log.calls, calls) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
	}
	deletion := app.IssueDeletion{IssueID: id, By: alice, At: clockNow}
	if want := slices.Repeat([]app.IssueDeletion{deletion}, 5); !slices.Equal(w.issues.deletions, want) {
		t.Errorf("the deletions %+v, want %+v", w.issues.deletions, want)
	}
}

// Every refusal of the decision is answered before any lock (M4 design
// 3.5), and the decision under the lock is final: a work item not there,
// or whose project the caller does not see, before or under the lock, is
// issue.not_found; a role that may not delete it forbidden; an archived
// project refuses it under the lock (409 project.archived). None deletes
// a row.
func TestDeleteIssueRefusals(t *testing.T) {
	for _, tt := range []struct {
		name  string
		set   func(w *world, id *uuid.UUID)
		want  error
		calls int // of lockedAs'
	}{
		{"no work item", func(_ *world, id *uuid.UUID) { *id = uuid.NewV7() }, domain.ErrIssueNotFound, 1},
		{"a project not seen", func(w *world, _ *uuid.UUID) { w.auth.answers = []error{shared.ErrNotVisible} }, domain.ErrIssueNotFound, 2},
		{"a role not allowed", func(w *world, _ *uuid.UUID) { w.auth.answers = []error{shared.Forbidden()} }, shared.Forbidden(), 2},
		{"the project deleted while the lock waited", func(w *world, _ *uuid.UUID) { w.project.locked = nil }, domain.ErrIssueNotFound, 4},
		{"the work item deleted meanwhile", func(w *world, _ *uuid.UUID) { w.project.waited = func() { w.issues.issue.ID = uuid.NewV7() } },
			domain.ErrIssueNotFound, 5},
		{"no longer seen", func(w *world, _ *uuid.UUID) { w.auth.answers = []error{nil, shared.ErrNotVisible} }, domain.ErrIssueNotFound, 6},
		{"no longer allowed", func(w *world, _ *uuid.UUID) { w.auth.answers = []error{nil, shared.Forbidden()} }, shared.Forbidden(), 6},
		{"archived", func(w *world, _ *uuid.UUID) { w.project.locked.Archived = true }, domain.ErrProjectArchived, 6},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id := newDelete()
			tt.set(w, &id)
			err := uc.Execute(as(alice), id)
			wantRefusal(t, w, err, tt.want, lockedAs(domain.ActionDelete, id)[:tt.calls])
			if len(w.issues.deletions) != 0 {
				t.Errorf("the deletions %+v, want none", w.issues.deletions)
			}
		})
	}
}

// A port that fails ends the deletion with its failure, as itself, and no
// call after it runs: the transaction rolls back whatever was deleted.
func TestDeleteIssueStopsAtAFailingPort(t *testing.T) {
	for _, port := range []string{"GetIssue", "Authorize", "Lock", "DeleteIssueRow", "DeleteIssueAssigneeRows", "DeleteIssueLabelRows",
		"DeleteIssueSubscriberRows", "DeleteIssueActivityRows"} {
		t.Run(port, func(t *testing.T) {
			uc, w, id := newDelete()
			w.log.fail = port
			wantStoppedAt(t, w, uc.Execute(as(alice), id), port)
		})
	}
}
````

`server/internal/modules/issue/app/fakes_test.go`（修改，3 处）：

````old server/internal/modules/issue/app/fakes_test.go
// last writer and moment.
````
````new server/internal/modules/issue/app/fakes_test.go
// last writer and moment. deletions are the deletion's calls' arguments,
// one a table, in order.
````
````old server/internal/modules/issue/app/fakes_test.go
	activities []app.ActivityRow
````
````new server/internal/modules/issue/app/fakes_test.go
	activities []app.ActivityRow
	deletions  []app.IssueDeletion
````
````old server/internal/modules/issue/app/fakes_test.go
}

func (f *fakeIssues) GetIssue(ctx context.Context, id uuid.UUID) (domain.Issue, bool, error) {
````
````new server/internal/modules/issue/app/fakes_test.go
}

// deleteRows logs a deletion of name's rows, d's work item's, and keeps d,
// or answers the failure.
func (f *fakeIssues) deleteRows(ctx context.Context, name string, d app.IssueDeletion) error {
	if err := f.log.add(ctx, name, d.IssueID); err != nil {
		return err
	}
	f.deletions = append(f.deletions, d)
	return nil
}

func (f *fakeIssues) DeleteIssueRow(ctx context.Context, d app.IssueDeletion) error {
	return f.deleteRows(ctx, "DeleteIssueRow", d)
}

func (f *fakeIssues) DeleteIssueAssigneeRows(ctx context.Context, d app.IssueDeletion) error {
	return f.deleteRows(ctx, "DeleteIssueAssigneeRows", d)
}

func (f *fakeIssues) DeleteIssueLabelRows(ctx context.Context, d app.IssueDeletion) error {
	return f.deleteRows(ctx, "DeleteIssueLabelRows", d)
}

func (f *fakeIssues) DeleteIssueSubscriberRows(ctx context.Context, d app.IssueDeletion) error {
	return f.deleteRows(ctx, "DeleteIssueSubscriberRows", d)
}

func (f *fakeIssues) DeleteIssueActivityRows(ctx context.Context, d app.IssueDeletion) error {
	return f.deleteRows(ctx, "DeleteIssueActivityRows", d)
}

func (f *fakeIssues) GetIssue(ctx context.Context, id uuid.UUID) (domain.Issue, bool, error) {
````

`server/internal/modules/issue/app/guest_rule_test.go`（修改，3 处）：

````old server/internal/modules/issue/app/guest_rule_test.go
// decision on a project names no one.
````
````new server/internal/modules/issue/app/guest_rule_test.go
// decision on a project names no one, and so does a decision on a work
// item whose creator is not known: AllowCreator takes no caller for him.
````
````old server/internal/modules/issue/app/guest_rule_test.go
		t.Fatal(err)
	}
	for _, tt := range []struct {
````
````new server/internal/modules/issue/app/guest_rule_test.go
		t.Fatal(err)
	}
	deletion, dw, did := newDelete()
	if err := deletion.Execute(as(alice), did); err != nil {
		t.Fatal(err)
	}
	unknown, nw, nid := newDelete()
	nw.issues.issue.By = uuid.UUID{}
	if err := unknown.Execute(as(alice), nid); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
````
````old server/internal/modules/issue/app/guest_rule_test.go
		{"a creation", cw.auth.targets, []uuid.UUID{{}, {}}},
````
````new server/internal/modules/issue/app/guest_rule_test.go
		{"a creation", cw.auth.targets, []uuid.UUID{{}, {}}},
		{"a deletion", dw.auth.targets, []uuid.UUID{carol, carol}},
		{"a deletion of a work item of no known creator", nw.auth.targets, []uuid.UUID{{}, {}}},
````

`server/internal/modules/issue/app/lock_test.go`（修改，2 处）：

````old server/internal/modules/issue/app/lock_test.go
	"time"
````
````new server/internal/modules/issue/app/lock_test.go
	"time"
	"uuid"
````
````old server/internal/modules/issue/app/lock_test.go
		t.Errorf("an update's calls:\n%q\nwant\n%q", w.log.calls, want)
````
````new server/internal/modules/issue/app/lock_test.go
		t.Errorf("an update's calls:\n%q\nwant\n%q", w.log.calls, want)
	}
}

// Each write P2 adds on a work item enters its project's write gate as an
// update does (TestAWriteEntersItsProjectsGateBetweenItsPrecheckAndItsTransaction):
// after its precheck, outside any transaction, before its transaction
// begins; and leaves it last, once its transaction has ended.
func TestEachWriteOnAWorkItemEntersItsProjectsGate(t *testing.T) {
	for _, tt := range []struct {
		name   string
		action shared.Action
		write  func(w *world, id uuid.UUID) error
	}{
		{"a deletion", domain.ActionDelete, func(w *world, id uuid.UUID) error {
			return app.NewDeleteIssue(app.DeleteIssueDeps{Locks: w.locks, Rows: w.issues, Clock: clockAt{at: clockNow, log: w.log}}).
				Execute(as(alice), id)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, w, id := newUpdate()
			gated(w, nil)
			if err := tt.write(w, id); err != nil {
				t.Fatal(err)
			}
			want := slices.Concat(lockedAs(tt.action, id)[:2], []string{line("Enter", web, "outside tx")}, lockedAs(tt.action, id)[2:])
			if len(w.log.calls) < len(want) || !slices.Equal(w.log.calls[:len(want)], want) ||
				slices.Index(w.log.calls, line("Leave", web)) != len(w.log.calls)-1 {
				t.Errorf("calls:\n%q\nwant them to begin\n%q\nand end with Leave web alone", w.log.calls, want)
			}
		})
````

`server/internal/modules/issue/app/update_issue_test.go`（修改，2 处）：

````old server/internal/modules/issue/app/update_issue_test.go
// the lock: the work item read and the decision, before any lock and
// outside the transaction; in it, web's lock, the work item read again and
// the decision again.
````
````new server/internal/modules/issue/app/update_issue_test.go
// the lock (lockedAs).
````
````old server/internal/modules/issue/app/update_issue_test.go
	return []string{line("GetIssue", id, "outside tx"), line("Authorize", alice, domain.ActionUpdate, acme, web, "outside tx"), "Begin",
		line("Lock", web), line("GetIssue", id), line("Authorize", alice, domain.ActionUpdate, acme, web)}
````
````new server/internal/modules/issue/app/update_issue_test.go
	return lockedAs(domain.ActionUpdate, id)
}

// lockedAs are the calls of a write of action on the work item id by alice
// up to the decision under the lock: the work item read and the decision,
// before any lock and outside the transaction; in it, web's lock, the work
// item read again and the decision again.
func lockedAs(action shared.Action, id uuid.UUID) []string {
	return []string{line("GetIssue", id, "outside tx"), line("Authorize", alice, action, acme, web, "outside tx"), "Begin", line("Lock", web),
		line("GetIssue", id), line("Authorize", alice, action, acme, web)}
````

- [ ] **Step 3: 存储**

`server/internal/modules/issue/adapter/postgres/delete_issue.go`（新文件，55 行）：

````file server/internal/modules/issue/adapter/postgres/delete_issue.go
package postgresadapter

import (
	"context"
	"fmt"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// The statements of a deleted work item (app.IssueRowsDeleter,
// delete_issue.sql): each soft-deletes the work item's undeleted rows of
// one table.

// deletion is d as the statements take it; their parameters are of one
// shape.
func deletion(d app.IssueDeletion) gen.DeleteIssueRowParams {
	return gen.DeleteIssueRowParams{At: d.At, DeletedBy: d.By, IssueID: d.IssueID}
}

// deletedRows is err, the deletion of the work item d's rows of what, as
// the deletion's error; nil is nil.
func deletedRows(err error, what string, d app.IssueDeletion) error {
	if err != nil {
		return fmt.Errorf("delete %s of work item %s: %w", what, d.IssueID, err)
	}
	return nil
}

// DeleteIssueRow soft-deletes the work item, archived or not.
func (s *Store) DeleteIssueRow(ctx context.Context, d app.IssueDeletion) error {
	return deletedRows(s.queries(ctx).DeleteIssueRow(ctx, deletion(d)), "the row", d)
}

// DeleteIssueAssigneeRows soft-deletes the work item's assignees' rows.
func (s *Store) DeleteIssueAssigneeRows(ctx context.Context, d app.IssueDeletion) error {
	return deletedRows(s.queries(ctx).DeleteIssueAssigneeRows(ctx, gen.DeleteIssueAssigneeRowsParams(deletion(d))), "the assignees", d)
}

// DeleteIssueLabelRows soft-deletes the rows of the labels on the work
// item.
func (s *Store) DeleteIssueLabelRows(ctx context.Context, d app.IssueDeletion) error {
	return deletedRows(s.queries(ctx).DeleteIssueLabelRows(ctx, gen.DeleteIssueLabelRowsParams(deletion(d))), "the labels", d)
}

// DeleteIssueSubscriberRows soft-deletes the work item's subscribers'
// rows.
func (s *Store) DeleteIssueSubscriberRows(ctx context.Context, d app.IssueDeletion) error {
	return deletedRows(s.queries(ctx).DeleteIssueSubscriberRows(ctx, gen.DeleteIssueSubscriberRowsParams(deletion(d))), "the subscribers", d)
}

// DeleteIssueActivityRows soft-deletes the work item's activity.
func (s *Store) DeleteIssueActivityRows(ctx context.Context, d app.IssueDeletion) error {
	return deletedRows(s.queries(ctx).DeleteIssueActivityRows(ctx, gen.DeleteIssueActivityRowsParams(deletion(d))), "the activity", d)
}
````

`server/internal/modules/issue/adapter/postgres/delete_issue_test.go`（新文件，98 行）：

````file server/internal/modules/issue/adapter/postgres/delete_issue_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"
	"time"
	"uuid"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// insertOf is how a row of table is inserted under a work item, as the
// batch statements' test inserts it (batchStatements).
func insertOf(t *testing.T, table string) inserter {
	t.Helper()
	i := slices.IndexFunc(batchStatements, func(st batchStatement) bool { return st.table == table })
	if i < 0 {
		t.Fatalf("no batch statement writes %s", table)
	}
	return batchStatements[i].insert
}

// Each statement of a deleted work item (M4 design 3.6, delete_issue.sql)
// soft-deletes the work item's undeleted rows of its table at the moment
// and by the account given, and nothing else. In web, the work item item
// has the rows a and b of the table (of the work items' table, item
// itself) and c, which alice deleted an hour before; the work item gone
// has its row g, deleted then too (of the work items' table, gone itself);
// the work item other has its row o, other's id after gone's and before
// item's, so that a statement that took its work item by the ids' order,
// not by equality, would reach o on item or on gone. The statement on
// item, then on gone, by bob at later: a and b (item) are deleted at later
// by bob; c and g (gone) keep alice's deletion, and o its row as it was;
// every other column of every row is kept. Each statement answers a failed
// write as its failure.
func TestTheStatementsOfADeletedWorkItemDeleteItsRows(t *testing.T) {
	for _, st := range []struct {
		table  string
		method func(*postgresadapter.Store, context.Context, app.IssueDeletion) error
	}{
		{"issues", (*postgresadapter.Store).DeleteIssueRow},
		{"issue_assignees", (*postgresadapter.Store).DeleteIssueAssigneeRows},
		{"issue_labels", (*postgresadapter.Store).DeleteIssueLabelRows},
		{"issue_subscribers", (*postgresadapter.Store).DeleteIssueSubscriberRows},
		{"issue_activities", (*postgresadapter.Store).DeleteIssueActivityRows},
	} {
		t.Run(st.table, func(t *testing.T) {
			s, pool := newStore(t)
			alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
			web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
			gone, other, item := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
			for n, id := range []uuid.UUID{item, gone, other} {
				newIssue(t, pool, web, id, n+1, alice)
			}
			deleted, earlier := []uuid.UUID{item}, []uuid.UUID{gone}
			if st.table != "issues" {
				a, b, c, g, o := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
				insert := insertOf(t, st.table)
				for n, row := range []struct{ issue, id uuid.UUID }{{item, a}, {item, b}, {item, c}, {gone, g}, {other, o}} {
					insert(t, pool, web, row.issue, row.id, alice, n)
				}
				deleted, earlier = []uuid.UUID{a, b}, []uuid.UUID{c, g}
			}
			before, later := now.Add(-time.Hour), now.Add(time.Hour)
			for _, id := range earlier {
				exec(t, pool, "UPDATE "+st.table+" SET deleted_at = $2, updated_at = $2, updated_by_id = $3 WHERE id = $1", id, before, alice)
			}
			rows, stamps := rowsAsText(t, pool, st.table), audits(t, pool, st.table)

			for _, id := range []uuid.UUID{item, gone} {
				if err := st.method(s, context.Background(), app.IssueDeletion{IssueID: id, By: bob, At: later}); err != nil {
					t.Fatal(err)
				}
			}

			if after := rowsAsText(t, pool, st.table); !maps.Equal(after, rows) {
				t.Errorf("the rows but their audit columns:\n%v\nwant them as they were:\n%v", after, rows)
			}
			want := maps.Clone(stamps)
			for _, id := range deleted {
				want[id] = deletion(later, bob)
			}
			if after := audits(t, pool, st.table); !maps.Equal(after, want) {
				t.Errorf("the audit columns:\n%v\nwant\n%v", after, want)
			}

			cancelled, cancel := context.WithCancel(context.Background())
			cancel()
			if err := st.method(s, cancelled, app.IssueDeletion{IssueID: other, By: bob, At: later}); !errors.Is(err, context.Canceled) {
				t.Errorf("the statement on a cancelled context = %v; want context.Canceled", err)
			}
		})
	}
}
````

- [ ] **Step 4: HTTP 和接线**

`server/internal/modules/issue/adapter/http/delete_test.go`（新文件，53 行）：

````file server/internal/modules/issue/adapter/http/delete_test.go
package httpadapter_test

import (
	"net/http"
	"slices"
	"testing"
	"time"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DELETE goes to the use case for the caller and the path's work item; the
// answer is 204, without a body.
func TestDeleteIssue(t *testing.T) {
	deletion := &fakeDelete{}
	res, body := do(t, newServer(t, httpadapter.UseCases{DeleteIssue: deletion}),
		request(http.MethodDelete, "/api/v0/issues/"+webSeven.ID.String(), "alice", ""))
	if res.StatusCode != http.StatusNoContent || body != "" {
		t.Errorf("DELETE = %d %q, want 204 without a body", res.StatusCode, body)
	}
	if want := []string{aliceID.String() + " " + webSeven.ID.String()}; !slices.Equal(deletion.calls, want) {
		t.Errorf("calls %q, want %q", deletion.calls, want)
	}
}

// The use case's refusals, as the contract declares them, and its failure.
func TestDeleteIssueRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"no work item", domain.ErrIssueNotFound, http.StatusNotFound,
			`{"status":404,"code":"issue.not_found","title":"Not Found","detail":"The work item does not exist, or you cannot see it."}`},
		{"a project member, another's work item", shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"an archived project", domain.ErrProjectArchived, http.StatusConflict,
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
		{"a busy project", shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
		{"a failure", errGone, http.StatusInternalServerError,
			`{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
	} {
		h := newServer(t, httpadapter.UseCases{DeleteIssue: &fakeDelete{err: tt.err}})
		res, body := do(t, h, request(http.MethodDelete, "/api/v0/issues/"+webSeven.ID.String(), "alice", ""))
		if res.StatusCode != tt.status || body != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
			t.Errorf("%s: DELETE = %d %s, Retry-After %q; want %d %s", tt.name, res.StatusCode, body, res.Header.Get("Retry-After"), tt.status,
				tt.want)
		}
	}
}
````

`server/internal/modules/issue/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/issue/adapter/http/handler.go
}

// UseCases are the use cases behind the module's operations.
````
````new server/internal/modules/issue/adapter/http/handler.go
}

// DeleteIssueUseCase is app.DeleteIssue.
type DeleteIssueUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) error
}

// UseCases are the use cases behind the module's operations.
````
````old server/internal/modules/issue/adapter/http/handler.go
	UpdateIssue UpdateIssueUseCase
````
````new server/internal/modules/issue/adapter/http/handler.go
	UpdateIssue UpdateIssueUseCase
	DeleteIssue DeleteIssueUseCase
````

`server/internal/modules/issue/adapter/http/handler_test.go`（修改，1 处）：

````old server/internal/modules/issue/adapter/http/handler_test.go
}

// newServer serves the module's API over the use cases uc, as bootstrap
````
````new server/internal/modules/issue/adapter/http/handler_test.go
}

// fakeDelete is deleteIssue: each call is recorded as "caller work item";
// it answers err.
type fakeDelete struct {
	calls []string
	err   error
}

func (f *fakeDelete) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String())
	return f.err
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

// DeleteIssue serves DELETE /api/v0/issues/{issue_id}.
func (h handler) DeleteIssue(ctx context.Context, req gen.DeleteIssueRequestObject) (gen.DeleteIssueResponseObject, error) {
	if err := h.uc.DeleteIssue.Execute(ctx, req.IssueID); err != nil {
		return nil, err
	}
	return gen.DeleteIssue204Response{}, nil
}

// issueDetail is i as the API shows it: each field that may be null null
````

`server/internal/modules/issue/module.go`（修改，2 处）：

````old server/internal/modules/issue/module.go
// work items' tables and brings creating, reading and changing a work
// item. For the project module's deletions (its IssueCascade port) it
````
````new server/internal/modules/issue/module.go
// work items' tables and brings creating, reading, changing and deleting a
// work item. For the project module's deletions (its IssueCascade port) it
````
````old server/internal/modules/issue/module.go
			Locks: locks, Issues: store, States: d.Projects, Labels: d.Projects, Members: d.Projects, Profiles: d.Profiles, Clock: d.Clock,
		}),
````
````new server/internal/modules/issue/module.go
			Locks: locks, Issues: store, States: d.Projects, Labels: d.Projects, Members: d.Projects, Profiles: d.Profiles, Clock: d.Clock,
		}),
		DeleteIssue: app.NewDeleteIssue(app.DeleteIssueDeps{Locks: locks, Rows: store, Clock: d.Clock}),
````

- [ ] **Step 5: 按目录的删除、取锁形状、写闸门和矩阵**

`server/internal/bootstrap/gate_test.go`（修改，1 处）：

````old server/internal/bootstrap/gate_test.go
}

// A write whose request reaches its deadline while it waits at the gate is
````
````new server/internal/bootstrap/gate_test.go
}

// Each write P2 adds on a work item enters its project's write gate, as a
// change does (TestAProjectTooBusyForAWriteAnswersServerBusy; M4 design
// 3.5, 11.11): while a write holds Web's one place, each of them on a work
// item of Web is refused at once, the queue full, 503 server_busy with
// Retry-After: 1, before a wait for Web's row could have ended, at the
// lock timeout at the earliest. In an app that lets one write a project in
// at once, queues none and ends a write's wait for a lock at three
// seconds: a transaction holds Web's row, and alice's change of a work
// item of Web enters the gate and waits for the row. Once the row is free,
// her change goes through, and so does each write refused.
func TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy(t *testing.T) {
	w := newIssueWorldWith(t, func(cfg *config.Config) {
		cfg.Issue = config.IssueConfig{WriteGate: config.WriteGateConfig{PerProject: 1, MaxWait: time.Minute},
			WriteLockTimeout: 3 * time.Second}
	})
	item := w.newWorkItem(t)
	writes := []struct {
		name, method, path string
		want               int // its answer once Web's row is free
	}{
		{"a deletion", http.MethodDelete, "/api/v0/issues/" + w.newWorkItem(t).String(), http.StatusNoContent},
	}
	change := newRequest(t, http.MethodPatch, w.base+"/api/v0/issues/"+item.String(), w.tokens["alice"], []byte(`{"name":"Mend"}`))
	w.contract.CheckRequest(t, change)

	held := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR UPDATE", w.web)
	changed := sendInBackground(change)
	pgtest.WaitForLockWaitBehind(t, w.pool, held.Conn().PgConn(), 5*time.Second)
	for _, write := range writes {
		req := newRequest(t, write.method, w.base+write.path, w.tokens["alice"], nil)
		w.contract.CheckRequest(t, req)
		begun := time.Now()
		a := receiveWithin(t, sendInBackground(req), 5*time.Second, "answer to "+write.name)
		if a.err != nil {
			t.Fatal(a.err)
		}
		if took := time.Since(begun); took >= time.Second {
			t.Errorf("%s answered after %v; want at once, the queue full, before a wait for Web's row could end at 3s", write.name, took)
		}
		w.wantBusy(t, write.name, req, a.res, a.body)
	}

	release(t, held)
	a := receiveWithin(t, changed, 4*time.Second, "answer to alice's change")
	if a.err != nil {
		t.Fatal(a.err)
	}
	if a.res.StatusCode != http.StatusOK {
		t.Errorf("alice's change = %d %s, want 200 once Web's row is free", a.res.StatusCode, a.body)
	}
	for _, write := range writes {
		w.send(t, "alice", write.method, write.path, "", write.want)
	}
}

// A write whose request reaches its deadline while it waits at the gate is
````

`server/internal/bootstrap/issue_deletion_test.go`（新文件，119 行）：

````file server/internal/bootstrap/issue_deletion_test.go
package bootstrap

import (
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The deletion of a work item against the catalog (M4 design 3.6), as a
// project's (project_deletion_test.go): a work item deleted through the
// wired app leaves no undeleted row under it, in any table with a foreign
// key to issues (keysTo); deletes each at the work item's deleted_at, a
// time within the request, by its deleter; leaves its rows deleted before
// as they were; and changes no row of another work item of its project.
// The tables come from pg_constraint: a phase that adds a table under
// issues fails this test until a creation or a change writes a row of it
// under the work item, and the deletion deletes it.

// subItems is the key of a work item's sub-items, which come with P3, as
// does their deletion with their parent's (M4 design 3.6, 3.9): no work
// item has one before.
const subItems = "issues.parent_id"

// workItemKeys are the foreign keys to issues (keysTo), the work item's
// own row included, but its sub-items' (subItems).
func workItemKeys(t testing.TB, pool *pgxpool.Pool) []foreignKey {
	t.Helper()
	return slices.DeleteFunc(keysTo(t, pool, "issues"), func(k foreignKey) bool { return k.String() == subItems })
}

// deletedWorkItem is a work item that the caller of token, account, makes
// in project through the API, assigned to him and labelled the project's
// Bug, and deletes through the API: a row of each table under it,
// deleted.
func deletedWorkItem(t *testing.T, contract *apitest.Contract, base string, pool *pgxpool.Pool, token string, account,
	project uuid.UUID) uuid.UUID {
	t.Helper()
	var bug uuid.UUID
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT id FROM labels WHERE project_id = $1 AND name = 'Bug'", project).Scan(&bug); err != nil {
		t.Fatal(err)
	}
	status, body := call(t, contract, http.MethodPost, base+"/api/v0/projects/"+project.String()+"/issues", token,
		`{"name":"Gone","assignee_ids":["`+account.String()+`"],"label_ids":["`+bug.String()+`"]}`)
	if status != http.StatusCreated {
		t.Fatalf("creating the work item = %d %s, want 201", status, body)
	}
	var created struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, body, &created)
	if status, body := call(t, contract, http.MethodDelete, base+"/api/v0/issues/"+created.ID.String(), token, ""); status != http.StatusNoContent {
		t.Fatalf("deleting the work item = %d %s, want 204", status, body)
	}
	return created.ID
}

// Deleting a work item through the API soft-deletes every row under it in
// every table the catalog ties to issues, and the work item's row, each at
// its deleted_at, a time within the request, and by the account that
// deleted it; its rows deleted before keep their deletion, and nothing
// under its project's other work item changes. The work item is bob's,
// assigned to him and labelled Bug (fullIssue), whom he unassigned and
// unlabelled, then assigned and labelled again: a row of each table under
// it, and of its assignees and its labels one deleted before, all of
// bob's writing. alice, Web's admin, deletes it: no row under it is of her
// writing before, so a deletion that kept a row's writer would show. The
// deleted work item is not found any more.
func TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt(t *testing.T) {
	w := newIssueWorld(t)
	deleted, kept := w.newWorkItemBy(t, "bob", w.web, w.fullIssue()), w.newWorkItemBy(t, "bob", w.web, w.fullIssue())
	for _, body := range []string{`{"assignee_ids":[],"label_ids":[]}`,
		`{"assignee_ids":["` + w.ids["bob"].String() + `"],"label_ids":["` + w.bug.String() + `"]}`} {
		w.send(t, "bob", http.MethodPatch, "/api/v0/issues/"+deleted.String(), body, http.StatusOK)
	}
	keys := workItemKeys(t, w.pool)
	under, recorded := make([]rowsUnder, len(keys)), make([][]uuid.UUID, len(keys))
	earlier, earlierRows := make([][]uuid.UUID, len(keys)), make([]string, len(keys))
	for i, k := range keys {
		recorded[i], earlier[i] = k.undeleted(t, w.pool, deleted), k.deleted(t, w.pool, deleted)
		earlierRows[i] = k.asText(t, w.pool, earlier[i])
		under[i] = rowsUnder{key: k.String(), deletedBefore: len(recorded[i]), keptBefore: k.rows(t, w.pool, kept)}
		if (k.table == "issue_assignees" || k.table == "issue_labels") && len(earlier[i]) == 0 {
			t.Fatalf("%s: no row of the work item deleted before: bob's change removes one", k)
		}
	}
	noRowLastWrittenBy(t, w.pool, keys, deleted, w.ids["alice"])

	before := time.Now().Truncate(time.Microsecond)
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+deleted.String(), "", http.StatusNoContent)
	after := time.Now()

	for i, k := range keys {
		under[i].deletedAfter, under[i].keptAfter = len(k.undeleted(t, w.pool, deleted)), k.rows(t, w.pool, kept)
	}
	for _, v := range deletionViolations("issues", under, nil) {
		t.Error(v)
	}
	for i, k := range keys {
		if rows := k.unstamped(t, w.pool, recorded[i], deleted, w.ids["alice"]); rows != "" {
			t.Errorf("%s: rows not deleted at the work item's deleted_at by its deleter alice:\n%s", k, rows)
		}
		if rows := k.asText(t, w.pool, earlier[i]); rows != earlierRows[i] {
			t.Errorf("%s: the rows deleted before:\n%s\nwant them as they were:\n%s", k, rows, earlierRows[i])
		}
	}
	var at time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at FROM issues WHERE id = $1", deleted).Scan(&at); err != nil ||
		at.Before(before) || at.After(after) {
		t.Errorf("the work item's deleted_at = %v, %v; want within the request, %v to %v", at, err, before, after)
	}
	w.send(t, "alice", http.MethodGet, "/api/v0/issues/"+deleted.String(), "", http.StatusNotFound)
}
````

`server/internal/bootstrap/issue_write_locks_test.go`（修改，17 处）：

````old server/internal/bootstrap/issue_write_locks_test.go
	"time"
	"uuid"
````
````new server/internal/bootstrap/issue_write_locks_test.go
	"time"
````
````old server/internal/bootstrap/issue_write_locks_test.go
// made in Web for it; body, its body when alice has sent it n times
// before, which names the world's rows and changes them anew; the answer
// it gets; and reads, the first table it reads under its project's lock
// that its precheck did not read, before any write of its own.
````
````new server/internal/bootstrap/issue_write_locks_test.go
// made in Web for it, made anew for each of alice's sends, before any lock
// is held, so that a write that ends what it names finds it there; body,
// its body, which names the world's rows; the answer it gets; waits, the
// table it waits for under its lock before any write of its own: the
// first it reads there that its precheck did not (held ACCESS EXCLUSIVE),
// or, when it reads none, the first it writes (held SHARE, which its
// precheck's plain read does not wait for); and mode, the lock another
// transaction holds of that table: ACCESS EXCLUSIVE, which a plain read
// waits for too, or SHARE, which a write waits for and a plain read does
// not, so that the write waits there and its precheck does not.
````
````old server/internal/bootstrap/issue_write_locks_test.go
	body             func(w issueWorld, n int) string
````
````new server/internal/bootstrap/issue_write_locks_test.go
	body             func(w issueWorld) string
````
````old server/internal/bootstrap/issue_write_locks_test.go
	reads            string
````
````new server/internal/bootstrap/issue_write_locks_test.go
	waits, mode      string
````
````old server/internal/bootstrap/issue_write_locks_test.go
		body:   func(w issueWorld, _ int) string { return w.fullIssue() }, want: http.StatusCreated, reads: "states"},
	// A work item in Todo, which each of alice's sends moves to the other
	// state, Done first: a change of state, so the states are read under
	// the lock.
````
````new server/internal/bootstrap/issue_write_locks_test.go
		body:   func(w issueWorld) string { return w.fullIssue() }, want: http.StatusCreated, waits: "states", mode: "ACCESS EXCLUSIVE"},
	// A work item in Todo, which the send moves to Done: a change of
	// state, so the states are read under the lock.
````
````old server/internal/bootstrap/issue_write_locks_test.go
		body: func(w issueWorld, n int) string {
			return `{"state_id":"` + []uuid.UUID{w.done, w.todo}[n%2].String() + `"}`
		},
		want: http.StatusOK, reads: "states"},
````
````new server/internal/bootstrap/issue_write_locks_test.go
		body:   func(w issueWorld) string { return `{"state_id":"` + w.done.String() + `"}` }, want: http.StatusOK, waits: "states",
		mode: "ACCESS EXCLUSIVE"},
	// A work item the send deletes: it reads nothing under the lock that
	// its precheck did not, so it waits for the work items' table, which
	// it writes first.
	{op: "deleteIssue", method: http.MethodDelete, path: "/api/v0/issues/{issue_id}",
		target: func(t *testing.T, w issueWorld) string { return "/api/v0/issues/" + w.newWorkItem(t).String() },
		body:   func(issueWorld) string { return "" }, want: http.StatusNoContent, waits: "issues", mode: "SHARE"},
````
````old server/internal/bootstrap/issue_write_locks_test.go
//     transaction holds the first table the write reads under its lock
//     that its precheck did not (reads) IN ACCESS EXCLUSIVE MODE, which a
//     plain read waits for too. alice's write waits for it before any
//     write of its own: it holds no table's ROW EXCLUSIVE lock, which
//     every write takes (writtenTables), so what it holds of Web is its
//     lock's, not an UPDATE's. It holds Web's row FOR NO KEY UPDATE, no
//     weaker and no stronger, and acme's row not at all (lockOn).
````
````new server/internal/bootstrap/issue_write_locks_test.go
//     transaction holds the table the write waits for under its lock
//     (waits), in its mode (mode): the first it reads there that its
//     precheck did not, or, when it reads none, the first it writes.
//     alice's write waits for it before any write of its own: it holds no
//     table's ROW EXCLUSIVE lock, which every write takes (writtenTables),
//     so what it holds of Web is its lock's, not an UPDATE's. It holds
//     Web's row FOR NO KEY UPDATE, no weaker and no stronger, and acme's
//     row not at all (lockOn).
````
````old server/internal/bootstrap/issue_write_locks_test.go
			target := iw.target(t, w)
			if !pathOf(iw.path, target) {
				t.Fatalf("%s's row sends %s %s, not a path of %s", iw.op, iw.method, target, iw.path)
````
````new server/internal/bootstrap/issue_write_locks_test.go
			// alice's three sends' targets; the callers refused send her
			// first's.
			var targets []string
			for range 3 {
				target := iw.target(t, w)
				if !pathOf(iw.path, target) {
					t.Fatalf("%s's row sends %s %s, not a path of %s", iw.op, iw.method, target, iw.path)
				}
				targets = append(targets, target)
````
````old server/internal/bootstrap/issue_write_locks_test.go
			path, alices := w.base+target, 0
````
````new server/internal/bootstrap/issue_write_locks_test.go
			alices := 0
````
````old server/internal/bootstrap/issue_write_locks_test.go
				body := iw.body(w, alices)
				if name == "alice" {
````
````new server/internal/bootstrap/issue_write_locks_test.go
				target := targets[0]
				if name == "alice" {
					target = targets[alices]
````
````old server/internal/bootstrap/issue_write_locks_test.go
				req := newRequest(t, iw.method, path, w.tokens[name], []byte(body))
````
````new server/internal/bootstrap/issue_write_locks_test.go
				req := newRequest(t, iw.method, w.base+target, w.tokens[name], []byte(iw.body(w)))
````
````old server/internal/bootstrap/issue_write_locks_test.go
			read := lockedTables(t, w.pool, iw.reads, "ACCESS EXCLUSIVE")
````
````new server/internal/bootstrap/issue_write_locks_test.go
			waited := lockedTables(t, w.pool, iw.waits, iw.mode)
````
````old server/internal/bootstrap/issue_write_locks_test.go
			pgtest.WaitForLockWaitBehind(t, w.pool, read.Conn().PgConn(), 10*time.Second)
			if table := waitedTable(t, w.pool, read.Conn().PgConn()); table != iw.reads {
				t.Fatalf("%s waits for %s, not for %s", iw.op, table, iw.reads)
````
````new server/internal/bootstrap/issue_write_locks_test.go
			pgtest.WaitForLockWaitBehind(t, w.pool, waited.Conn().PgConn(), 10*time.Second)
			if table := waitedTable(t, w.pool, waited.Conn().PgConn()); table != iw.waits {
				t.Fatalf("%s waits for %s, not for %s", iw.op, table, iw.waits)
````
````old server/internal/bootstrap/issue_write_locks_test.go
			if written := writtenTables(t, w.pool, read.Conn().PgConn()); len(written) > 0 {
				t.Errorf("%s has written %q when it reads %s: what it holds of Web is no longer its lock's alone", iw.op, written, iw.reads)
````
````new server/internal/bootstrap/issue_write_locks_test.go
			if written := writtenTables(t, w.pool, waited.Conn().PgConn()); len(written) > 0 {
				t.Errorf("%s has written %q when it waits for %s: what it holds of Web is no longer its lock's alone", iw.op, written, iw.waits)
````
````old server/internal/bootstrap/issue_write_locks_test.go
				t.Errorf("%s holds Web %s when it reads %s, not FOR NO KEY UPDATE", iw.op, lock, iw.reads)
````
````new server/internal/bootstrap/issue_write_locks_test.go
				t.Errorf("%s holds Web %s when it waits for %s, not FOR NO KEY UPDATE", iw.op, lock, iw.waits)
````
````old server/internal/bootstrap/issue_write_locks_test.go
				t.Errorf("%s holds acme %s when it reads %s", iw.op, lock, iw.reads)
````
````new server/internal/bootstrap/issue_write_locks_test.go
				t.Errorf("%s holds acme %s when it waits for %s", iw.op, lock, iw.waits)
````
````old server/internal/bootstrap/issue_write_locks_test.go
			release(t, read)
````
````new server/internal/bootstrap/issue_write_locks_test.go
			release(t, waited)
````

`server/internal/bootstrap/issue_writes_test.go`（修改，3 处）：

````old server/internal/bootstrap/issue_writes_test.go
// and walt is refused (403); and she reads it, without a transaction. With
// Web archived, her creation and her change are refused under the lock
// (409).
````
````new server/internal/bootstrap/issue_writes_test.go
// and walt is refused (403); and she reads it, without a transaction. She
// deletes the second. With Web archived, her creation and her change are
// refused under the lock (409).
````
````old server/internal/bootstrap/issue_writes_test.go
	var defaulted struct {
````
````new server/internal/bootstrap/issue_writes_test.go
	var defaulted struct {
		ID          uuid.UUID   `json:"id"`
````
````old server/internal/bootstrap/issue_writes_test.go
	route.answerWithin(t, w.contract, http.MethodGet, "/api/v0/issues/"+first.ID.String(), alice, "", http.StatusOK)
````
````new server/internal/bootstrap/issue_writes_test.go
	route.answerWithin(t, w.contract, http.MethodGet, "/api/v0/issues/"+first.ID.String(), alice, "", http.StatusOK)
	route.answerWithin(t, w.contract, http.MethodDelete, "/api/v0/issues/"+defaulted.ID.String(), alice, "", http.StatusNoContent)
````

`server/internal/bootstrap/permission_matrix_issue_updates_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
// The rows of the permission matrix of changing a work item (M4 design
// 9.2), naming it by its id (/issues/{issue_id}): its column's work item
// (issueOf). Each cell writes on a copy of its own. The row of 9.2's table
// has the issue table's columns; its variants, the project level's, as
// createIssue's (permission_matrix_issues_test.go).
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
// The rows of the permission matrix of changing and deleting a work item
// (M4 design 9.2), naming it by its id (/issues/{issue_id}): its column's
// work item (issueOf). Each cell writes on a copy of its own. A row of
// 9.2's table has the issue table's columns; its variants, the project
// level's, as createIssue's (permission_matrix_issues_test.go).
````
````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
			cells: ok, check: keepsTheAssigneeWhoLeft},
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
			cells: ok, check: keepsTheAssigneeWhoLeft},
		// The project's admins, and the member who made the work item
		// (AllowCreator, M4 design 3.4), not another member; a guest of the
		// project may not, also on his own, which he made as the member he
		// was (9.2: made it, then a guest, 403); the member before, who made
		// his and has left the project, does not see it (made it, then left,
		// 404); a guest who sees only his own finds none another made.
		{op: "deleteIssue", write: true, columns: issueColumns, request: toWorkItem(http.MethodDelete, "", ""),
			cells: onAWorkItem(cellNoContent, cellForbidden, cellNoContent, cellForbidden)},
		// An archived project's work items are not deleted (M4 design 3.8),
		// after the decision, as updateIssue's.
		{op: "deleteIssue", variant: "archived", write: true, columns: archivedColumns, request: toWorkItem(http.MethodDelete, "", ""),
			cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
````

`server/internal/bootstrap/project_deletion_test.go`（修改，7 处）：

````old server/internal/bootstrap/project_deletion_test.go
// noRowLastWrittenBy fails the test when a row under project, in a table
// the catalog ties to projects (keysTo), the project row itself included,
// was last written by by: a deletion by by that kept a row's writer would
// not show on that row. The deletion tests read it before they delete.
func noRowLastWrittenBy(t *testing.T, pool *pgxpool.Pool, project, by uuid.UUID) {
````
````new server/internal/bootstrap/project_deletion_test.go
// noRowLastWrittenBy fails the test when a row under the parent row id, by
// one of keys, the foreign keys to its table (keysTo), the row itself
// included, was last written by by: a deletion by by that kept a row's
// writer would not show on that row. The deletion tests read it before
// they delete.
func noRowLastWrittenBy(t *testing.T, pool *pgxpool.Pool, keys []foreignKey, id, by uuid.UUID) {
````
````old server/internal/bootstrap/project_deletion_test.go
	for _, k := range keysTo(t, pool, "projects") {
````
````new server/internal/bootstrap/project_deletion_test.go
	for _, k := range keys {
````
````old server/internal/bootstrap/project_deletion_test.go
			" = $1 AND updated_by_id = $2", project, by).Scan(&n); err != nil || n != 0 {
			t.Fatalf("%s: %d rows under the project %s last written by %s, %v; want none", k, n, project, by, err)
````
````new server/internal/bootstrap/project_deletion_test.go
			" = $1 AND updated_by_id = $2", id, by).Scan(&n); err != nil || n != 0 {
			t.Fatalf("%s: %d rows under the %s %s last written by %s, %v; want none", k, n, rowOf(k.parent), id, by, err)
````
````old server/internal/bootstrap/project_deletion_test.go
// The deleted project is not found any more. Web is archived first: an
// archived project is deleted as any other. Its deleter is dave, acme's
// admin, whom alice adds as Web's admin: no row under Web is of his
// writing before, so a deletion that kept a row's writer would show.
````
````new server/internal/bootstrap/project_deletion_test.go
// The deleted project is not found any more. A work item of Web that alice
// deleted before keeps its rows as they were, their deletion hers (M4/P1a
// review §6: the statements pass over a row deleted before). Web is
// archived then: an archived project is deleted as any other. Its deleter
// is dave, acme's admin, whom alice adds as Web's admin: no row under Web
// is of his writing before, so a deletion that kept a row's writer would
// show.
````
````old server/internal/bootstrap/project_deletion_test.go
	contract, base, pool, alice, aliceID, web, ops := twoProjects(t)
````
````new server/internal/bootstrap/project_deletion_test.go
	contract, base, pool, alice, aliceID, web, ops := twoProjects(t)
	gone := deletedWorkItem(t, contract, base, pool, alice, aliceID, web)
	goneKeys := workItemKeys(t, pool)
	goneRows := make([]string, len(goneKeys))
	for i, k := range goneKeys {
		if goneRows[i] = k.rows(t, pool, gone); goneRows[i] == "" {
			t.Fatalf("%s: no row of the work item deleted before Web: deletedWorkItem writes one", k)
		}
	}
````
````old server/internal/bootstrap/project_deletion_test.go
	noRowLastWrittenBy(t, pool, web, daveID)
````
````new server/internal/bootstrap/project_deletion_test.go
	noRowLastWrittenBy(t, pool, keys, web, daveID)
````
````old server/internal/bootstrap/project_deletion_test.go
			t.Errorf("%s: rows not deleted at the project's deleted_at by its deleter %s:\n%s", k, daveID, rows)
````
````new server/internal/bootstrap/project_deletion_test.go
			t.Errorf("%s: rows not deleted at the project's deleted_at by its deleter %s:\n%s", k, daveID, rows)
		}
	}
	for i, k := range goneKeys {
		if rows := k.rows(t, pool, gone); rows != goneRows[i] {
			t.Errorf("%s: the rows of the work item deleted before Web:\n%s\nwant them as they were:\n%s", k, rows, goneRows[i])
````

`server/internal/bootstrap/workspace_deletion_catalog_test.go`（修改，5 处）：

````old server/internal/bootstrap/workspace_deletion_catalog_test.go
// project_deletion_test.go): the foreign keys to the deleted row's table,
// workspaces or projects, which fail on a table under it without one of its
// own (keysTo, with its counterexample), and each key's rows under a
// deleted row: undeleted, unstamped by the deletion, or all of them as
````
````new server/internal/bootstrap/workspace_deletion_catalog_test.go
// project_deletion_test.go, issue_deletion_test.go): the foreign keys to
// the deleted row's table, workspaces, projects or issues, which fail on a
// table under it without one of its own (keysTo, with its counterexample);
// each key's rows under a deleted row: undeleted or deleted, unstamped by
// the deletion, or all of them as text; and its rows of the ids given, as
````
````old server/internal/bootstrap/workspace_deletion_catalog_test.go
// "workspace" for workspaces, "project" for projects. A parent is spelt as
// its table everywhere else.
````
````new server/internal/bootstrap/workspace_deletion_catalog_test.go
// "workspace" for workspaces, "project" for projects, "issue" for issues.
// A parent is spelt as its table everywhere else.
````
````old server/internal/bootstrap/workspace_deletion_catalog_test.go
func (k foreignKey) undeleted(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) []uuid.UUID {
	t.Helper()
````
````new server/internal/bootstrap/workspace_deletion_catalog_test.go
func (k foreignKey) undeleted(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) []uuid.UUID {
	t.Helper()
	return k.idsUnder(t, pool, id, "IS NULL")
}

// deleted are the ids of k's deleted rows under the parent row id.
func (k foreignKey) deleted(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) []uuid.UUID {
	t.Helper()
	return k.idsUnder(t, pool, id, "IS NOT NULL")
}

// idsUnder are the ids of k's rows under the parent row id whose
// deleted_at is as state says, in order.
func (k foreignKey) idsUnder(t *testing.T, pool *pgxpool.Pool, id uuid.UUID, state string) []uuid.UUID {
	t.Helper()
````
````old server/internal/bootstrap/workspace_deletion_catalog_test.go
		" = $1 AND deleted_at IS NULL ORDER BY id", id)
````
````new server/internal/bootstrap/workspace_deletion_catalog_test.go
		" = $1 AND deleted_at "+state+" ORDER BY id", id)
````
````old server/internal/bootstrap/workspace_deletion_catalog_test.go
	return ids
````
````new server/internal/bootstrap/workspace_deletion_catalog_test.go
	return ids
}

// asText is k's rows ids, each as text, in order; "" when there is none.
func (k foreignKey) asText(t *testing.T, pool *pgxpool.Pool, ids []uuid.UUID) string {
	t.Helper()
	var s string
	if err := pool.QueryRow(context.Background(), "SELECT coalesce(string_agg(r::text, E'\\n' ORDER BY r::text), '') FROM "+k.table+
		" r WHERE r.id = ANY ($1)", ids).Scan(&s); err != nil {
		t.Fatalf("%s: %v", k, err)
	}
	return s
````

`server/internal/bootstrap/workspace_deletion_test.go`（修改，3 处）：

````old server/internal/bootstrap/workspace_deletion_test.go
// itself as its id) held under the test's two parent rows, a workspace or
// a project each, before and after the deletion: the deleted one's
// undeleted rows, counted, and the kept one's rows, as text.
````
````new server/internal/bootstrap/workspace_deletion_test.go
// itself as its id) held under the test's two parent rows, two workspaces,
// two projects or two work items, before and after the deletion: the
// deleted one's undeleted rows, counted, and the kept one's rows, as text.
````
````old server/internal/bootstrap/workspace_deletion_test.go
// table ("workspaces" or "projects", as keysTo takes it): no row under
// either parent row before the deletion, which would leave its checks
// nothing to see; an undeleted row left under the deleted one, or, for a
// key on exempt, a row deleted that must survive; a row of the kept one
// changed. An exempt key that no foreign key matches is reported too.
````
````new server/internal/bootstrap/workspace_deletion_test.go
// table ("workspaces", "projects" or "issues", as keysTo takes it): no row
// under either parent row before the deletion, which would leave its
// checks nothing to see; an undeleted row left under the deleted one, or,
// for a key on exempt, a row deleted that must survive; a row of the kept
// one changed. An exempt key that no foreign key matches is reported too.
````
````old server/internal/bootstrap/workspace_deletion_test.go
		noRowLastWrittenBy(t, pool, project, adminID)
````
````new server/internal/bootstrap/workspace_deletion_test.go
		noRowLastWrittenBy(t, pool, keysTo(t, pool, "projects"), project, adminID)
````

- [ ] **Step 6: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt$|TestDeletingAProjectLeavesNoUndeletedRowUnderIt$|TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt$|TestEachIssueWriteLocksItsProjectFirst$|TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection$|TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy$|TestPermissionMatrix|TestThePermissionMatrixCoversEveryOperation$' ./internal/bootstrap/`
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
git add api/modules/issue.yaml server/internal/bootstrap/gate_test.go server/internal/bootstrap/issue_deletion_test.go server/internal/bootstrap/issue_write_locks_test.go server/internal/bootstrap/issue_writes_test.go server/internal/bootstrap/permission_matrix_issue_updates_test.go server/internal/bootstrap/project_deletion_test.go server/internal/bootstrap/workspace_deletion_catalog_test.go server/internal/bootstrap/workspace_deletion_test.go server/internal/modules/issue/adapter/http/delete_test.go server/internal/modules/issue/adapter/http/handler.go server/internal/modules/issue/adapter/http/handler_test.go server/internal/modules/issue/adapter/http/issues.go server/internal/modules/issue/adapter/postgres/delete_issue.go server/internal/modules/issue/adapter/postgres/delete_issue_test.go server/internal/modules/issue/adapter/postgres/queries/delete_issue.sql server/internal/modules/issue/app/delete_issue.go server/internal/modules/issue/app/delete_issue_test.go server/internal/modules/issue/app/fakes_test.go server/internal/modules/issue/app/guest_rule_test.go server/internal/modules/issue/app/lock_test.go server/internal/modules/issue/app/update_issue_test.go server/internal/modules/issue/module.go api/dist/openapi.yaml server/internal/modules/issue/adapter/http/gen/server.gen.go server/internal/modules/issue/adapter/postgres/gen/delete_issue.sql.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M4/P2): deleteIssue deletes a work item and its rows at one moment

DELETE /api/v0/issues/{issue_id} runs on the one lock path: the work item
read and issue.delete decided before any lock (the project's admins, and
the member who made it), the gate, the project FOR NO KEY UPDATE, the
work item read again and the decision again. An archived project refuses
it; an archived work item is deleted as any other. The work item and its
assignees, labels, subscribers and activity are soft-deleted table by
table, in the global order, by the caller at one moment. The deletion is
checked against the catalog, and the project's deletion passes over a
work item deleted before it.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p2.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试，或没有替身的平台、领域、转换的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是生成器（`make gen-go`）和中英文翻译键（`make lint-web`）的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过，端到端的检查只在标出的 Task 的树上）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T4.1` | 已归档的项目里的工作项照样删除 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `deleteIssue` 各格（`make test`） | 单元；组合 |
| `T4.2` | 删除留下工作项的负责人 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`（`make test`）、故事 `i15-archive-issue`（Task 13 起） | 单元；组合；端到端 |
| `T4.3` | 删除留下工作项的动态 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`（`make test`）、故事 `i15-archive-issue`（Task 13 起） | 单元；组合；端到端 |
| `T4.4` | 删除在取锁之前读时刻 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkItemAndChangingItSerialize`（`make test`）（Task 12 起） | 单元；组合 |
| `T4.5` | 删除保留工作项原来的最后修改者 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`（`make test`）、故事 `i15-archive-issue`（Task 13 起） | 存储；组合；端到端 |
| `T4.6` | 删除再删一次早已删除的负责人行，盖掉它们的戳 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`（`make test`） | 存储；组合 |
| `T4.7` | 删除把每个工作项的标签行都删掉 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`（`make test`） | 存储；组合 |
| `T4.8` | 删除先删负责人的行、再删工作项的 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestEachIssueWriteLocksItsProjectFirst`（`make test`） | 单元；组合 |
| `T4.9` | 删除在预检之前、在项目的锁之外删掉工作项的动态 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestEachIssueWriteLocksItsProjectFirst`（`make test`） | 单元；组合 |
| `T4.10` | 契约不声明 `deleteIssue` 答 `project.archived` | `internal/modules/issue/adapter/http` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `deleteIssue` 各格（`make test`） | 单元；组合 |
| `T4.11` | `DELETE` 不问用例就答 204 | `internal/modules/issue/adapter/http` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`（`make test`）、故事 `i15-archive-issue`（Task 13 起） | 单元；组合；端到端 |
| `T4.12` | 删除把每个未删除的工作项都删掉 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`（`make test`） | 存储；组合 |
| `T4.13` | 删除再删一次早已删除的工作项，盖掉它的戳 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`） | 存储 |
| `T4.14` | 删除把每个工作项的负责人行都删掉 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`（`make test`） | 存储；组合 |
| `T4.15` | 删除再删一次早已删除的标签行，盖掉它们的戳 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`（`make test`） | 存储；组合 |
| `T4.16` | 删除把每个工作项的关注行都删掉 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`（`make test`） | 存储；组合 |
| `T4.17` | 删除再删一次早已删除的关注行，盖掉它们的戳 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`） | 存储 |
| `T4.18` | 删除把每个工作项的动态都删掉 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`（`make test`） | 存储；组合 |
| `T4.19` | 删除再删一次早已删除的动态，盖掉它们的戳 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`） | 存储 |
| `T4.20` | 删除的语句经连接池发出，不在它的事务的连接上 | `internal/bootstrap` 的 `TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`（`make test`） | 组合 |
| `T4.21` | 删除不进项目的写闸门 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy`（`make test`） | 单元；组合 |
| `T4.22` | 删除按 id 的次序而不是相等取工作项：id 不在它之后的每个工作项的负责人行都删掉 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`） | 存储 |

本 Task 的检查也发现较早的 Task 的变异 `T1.1`、`T1.3`、`T1.4`、`T1.10`、`T2.4`、`T2.5`、`T2.9`（列在各自的 Task 的表里，标"（Task 4 起）"）：本 Task 做完之后，在本 Task 的树上重跑它们（`mutants_p2.py`，`MUT_LATER=4`）。

---

### Task 5: 按编号取：解析、折叠、`issue.search`、用例、端口和查询

**Files:**
- Create: `api/testdata/identifier-fold.json`、`server/internal/bootstrap/issue_ports_test.go`、`server/internal/modules/issue/app/get_issue_by_identifier.go`、`server/internal/modules/issue/app/get_issue_by_identifier_test.go`、`server/internal/modules/issue/domain/identifier.go`、`server/internal/modules/issue/domain/identifier_test.go`
- Modify: `server/internal/bootstrap/app.go`、`server/internal/bootstrap/issue_writes_test.go`、`server/internal/bootstrap/ports.go`、`server/internal/bootstrap/ports_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/issue/adapter/http/handler.go`、`server/internal/modules/issue/adapter/postgres/issues.go`、`server/internal/modules/issue/adapter/postgres/issues_test.go`、`server/internal/modules/issue/adapter/postgres/queries/issues.sql`、`server/internal/modules/issue/app/get_issue_test.go`、`server/internal/modules/issue/app/guest_rule_test.go`、`server/internal/modules/issue/app/lock.go`、`server/internal/modules/issue/app/ports.go`、`server/internal/modules/issue/domain/actions.go`、`server/internal/modules/issue/domain/errors.go`、`server/internal/modules/issue/module.go`、`server/internal/modules/project/adapter/postgres/failures_test.go`、`server/internal/modules/project/adapter/postgres/issue_ports.go`、`server/internal/modules/project/adapter/postgres/issue_ports_test.go`、`server/internal/modules/project/adapter/postgres/queries/issue_ports.sql`、`server/internal/modules/project/app/issue_ports.go`、`server/internal/modules/project/module.go`、`server/internal/modules/workspace/module.go`
- Generate: `server/internal/modules/issue/adapter/postgres/gen/issues.sql.go`、`server/internal/modules/project/adapter/postgres/gen/issue_ports.sql.go`

**Interfaces:**（spec 2.7、第 3 节第 4 条；M4 设计 3.4、3.7、6.5）
- `issue/domain/identifier.go`（新）：`ParseIdentifier(s) (project string, number int, ok bool)`（最后一个连字符；项目的标识不空；编号是不带符号、不以 0 开头的十进制，1 到 2147483647）；`foldIdentifier`（`a`–`z` 成 `A`–`Z`，`çşğöü` 成 `ÇŞĞÖÜ`，`i`、`ı`、`I`、`İ` 成 `I`）；`MatchIdentifier(asked, identifiers) (index int, ok bool)`：折叠之后相同的只有一个时取它，两个以上时取逐字等于 `strings.ToUpper(asked)` 的那个，没有就没有。`api/testdata/identifier-fold.json`（新，`issue-order.json` 的写法）：样例。
- `issue/domain`：`actions.go` 的 `ActionSearch`（`issue.search`），`Actions()` 带上它；`errors.go` 的 `ErrWorkspaceNotFound`（404 `workspace.not_found`，`workspace` 模块的码）。`access/domain/rules.go`：`issue.search`（工作区级，管理员、成员、访客）。
- `issue/app`：`ports.go` 的 `WorkspaceDirectory`（`WorkspaceBySlug(ctx, slug) (id uuid.UUID, found bool, err error)`）、`ProjectIdentifiers`（`Identifiers(ctx, workspaceID) ([]ProjectIdentifier, error)`）、`ProjectIdentifier{ID, Identifier}`、`IssueNumbers`（`IssueByNumber(ctx, projectID, number) (id uuid.UUID, found bool, err error)`）；`lock.go` 的 `Locks.seeWorkspace(ctx, actor, action, workspaceID) error`（不取锁；`ErrNotVisible` 答 `workspace.not_found`）；`get_issue_by_identifier.go`（新）：`GetIssueByIdentifierDeps{Locks, Workspaces, Projects, Issues}`、`NewGetIssueByIdentifier`、`Execute(ctx, slug, identifier) (domain.Issue, error)`：`WorkspaceBySlug` → `seeWorkspace(issue.search)` → `find`（`ParseIdentifier` → `Identifiers` → `MatchIdentifier` → `IssueByNumber`，找不到都是 `issue.not_found`）→ `Locks.see(issue.read)`；不取锁、不开事务、不进闸门。
- 查询：`project` 的 `queries/issue_ports.sql` 的 `ProjectIdentifiers`（工作区未删除的项目，归档的也在，`id` 的次序）；`issue` 的 `queries/issues.sql` 的 `IssueByNumber`（项目的、未删除的，归档的也在）。存储：`project/adapter/postgres/issue_ports.go` 的 `ProjectIdentifiers`，`issue/adapter/postgres/issues.go` 的 `IssueByNumber`。
- 端口和接线：`project/app/issue_ports.go` 的 `IssueProjectIdentifier{ID, Identifier}`；`project/module.go` 的 `IssueProjects` 加 `ProjectIdentifiers`；`issue/module.go` 的 `ProjectPorts` 嵌入 `app.ProjectIdentifiers`，`Deps.Workspaces`，`UseCases.GetIssueByIdentifier`；`issue/adapter/http/handler.go` 的 `GetIssueByIdentifierUseCase`（路由在 Task 6）；`bootstrap/ports.go` 的 `issueProjects.Identifiers` 和 `issueWorkspaces`（`workspace` 交给 `project` 的同一个目录，只取 `id`），`app.go` 和 `issue_writes_test.go` 的组合接上它；`workspace/module.go` 的 `WorkspaceDirectory` 的说明写上 `issue`。

**Tests:**
- 领域：`TestParseIdentifier`、`TestMatchIdentifierFindsWhatTheSamplesSay`（读 `identifier-fold.json`）。
- 单元：`TestGetIssueByIdentifierReadsTheWorkItem`（项目的标识用任何大小写问，土耳其字母也是）、`TestGetIssueByIdentifierRefusals`（工作区不在、调用者不是成员：`workspace.not_found`，在找工作项之前；`issue.search` 之外的角色 403；之后，不是标识、不是 acme 的项目、没有这个编号、看不到项目、访客规则隐藏的：同一个 `issue.not_found`；不能读的角色 403；读或判定的失败照原样；没有调用者，在任何读之前）；`TestGetIssueReadsTheWorkItem`（`readIssue`，两个用例共用）；`TestGetIssueByIdentifierRefusals`、`TestGetIssueRefusals` 和 `guest_rule_test.go` 的两个表逐值比较回答（`reflect.DeepEqual`：`shared.Error` 的 `Is` 只比较种类和码），访客规则隐藏的工作项与不存在的是同一个回答（M4 设计 8.4）；规则表的 `issue.search` 一行（`TestEveryRuleDecidesItsCells`）；`bootstrap` 的 `TestIssueProjectsConvertsProjectIdentifiers`、`TestIssueWorkspacesConvertsWorkspacesAnswer`（转换，不用数据库）。
- 存储：`TestIssueByNumber`（web 的 1、2 和 ops 的 1 各是自己的，归档的也在；已删除的不算；项目里没有的编号不算，即使别的项目有）、`TestAFailedIssueStatementIsAnError`；`TestProjectIdentifiers`（未删除的项目，归档的也在，`id` 的次序；已删除的、别的工作区的不在；先建的 Web 归档、Ops 的名字和标识在前，`id` 的次序既不是行的次序，也不是计划器可能读的 `(workspace_id, name)` 或 `(workspace_id, identifier)` 索引的次序）、`TestAFailedReadIsAnErrorNotAnAnswer`。
- 完整性：`TestEveryActionHasARuleAndEveryRuleAnAction`、`TestEveryModuleDeclaresItsActionsOrHasNone`。

- [ ] **Step 1: 两条查询，生成**

`server/internal/modules/issue/adapter/postgres/queries/issues.sql`（修改，1 处）：

````old server/internal/modules/issue/adapter/postgres/queries/issues.sql
WHERE i.id = sqlc.arg(id) AND i.deleted_at IS NULL;

````
````new server/internal/modules/issue/adapter/postgres/queries/issues.sql
WHERE i.id = sqlc.arg(id) AND i.deleted_at IS NULL;

-- name: IssueByNumber :one
-- The id of the project's undeleted work item, archived or not, of the number: getIssueByIdentifier's (M4 design 3.7),
-- read without a lock. A number is its project's once (issues_project_id_sequence_id_key), deleted work items' too.
SELECT id
FROM issues
WHERE project_id = sqlc.arg(project_id) AND sequence_id = sqlc.arg(sequence_id) AND deleted_at IS NULL;

````

`server/internal/modules/project/adapter/postgres/queries/issue_ports.sql`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/queries/issue_ports.sql
ORDER BY id;

-- name: ActiveRoles :many
````
````new server/internal/modules/project/adapter/postgres/queries/issue_ports.sql
ORDER BY id;

-- name: ProjectIdentifiers :many
-- ProjectIdentifiers (M4 design 3.7): the workspace's undeleted projects, archived or not, with their identifiers, in id
-- order, read without a lock: the projects a work item's identifier may name. A deleted project's identifier may be
-- another's since (M3 design 3.19), so it names none.
SELECT id, identifier
FROM projects
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL
ORDER BY id;

-- name: ActiveRoles :many
````

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `896cd1586bdbb710c07d860a3083aea64350f925779c5c4c17f26b51d4ae0444` | 295 | `server/internal/modules/issue/adapter/postgres/gen/issues.sql.go` |
| `b3a2acdc569ea80fe6cae9d1d640506e7f6e3acd60d38549153e45415ccf346b` | 207 | `server/internal/modules/project/adapter/postgres/gen/issue_ports.sql.go` |

Run: `shasum -a 256 server/internal/modules/issue/adapter/postgres/gen/issues.sql.go server/internal/modules/project/adapter/postgres/gen/issue_ports.sql.go`
Expected: 与上表相同。

- [ ] **Step 2: 解析和折叠**

`api/testdata/identifier-fold.json`（新文件，24 行）：

````file api/testdata/identifier-fold.json
{
  "description": "How getIssueByIdentifier finds a project by the identifier asked (M4 design 3.7): each case is a workspace's projects' identifiers, as they are stored, upper-cased; the project's identifier asked, as a path gives it; and the identifier of the project found, or null for none (404 issue.not_found). Identifiers compare folded: a-z as A-Z, çşğöü as ÇŞĞÖÜ, i, ı, I and İ as one letter. Of two or more that fold as the one asked does, the one equal to it in upper case letter for letter (Go's strings.ToUpper: i and ı are I, İ is İ) is found; when none is, none is found. The issue module's domain test reads this file.",
  "cases": [
    {"name": "as stored", "identifiers": ["WEB", "OPS"], "asked": "WEB", "want": "WEB"},
    {"name": "in lower case", "identifiers": ["WEB", "OPS"], "asked": "web", "want": "WEB"},
    {"name": "in mixed case", "identifiers": ["WEB", "OPS"], "asked": "wEb", "want": "WEB"},
    {"name": "with digits", "identifiers": ["W3B"], "asked": "w3b", "want": "W3B"},
    {"name": "Turkish letters in lower case", "identifiers": ["ÇŞĞÖÜ"], "asked": "çşğöü", "want": "ÇŞĞÖÜ"},
    {"name": "İ stored, i asked", "identifiers": ["İŞ"], "asked": "iş", "want": "İŞ"},
    {"name": "İ stored, ı asked", "identifiers": ["İŞ"], "asked": "ış", "want": "İŞ"},
    {"name": "I stored, ı asked", "identifiers": ["IŞ"], "asked": "ış", "want": "IŞ"},
    {"name": "I stored, İ asked", "identifiers": ["IŞ"], "asked": "İŞ", "want": "IŞ"},
    {"name": "I and İ stored, i asked: I, its upper case", "identifiers": ["IŞ", "İŞ"], "asked": "iş", "want": "IŞ"},
    {"name": "İ and I stored, i asked: I, whichever comes first", "identifiers": ["İŞ", "IŞ"], "asked": "iş", "want": "IŞ"},
    {"name": "I and İ stored, ı asked: I, its upper case", "identifiers": ["IŞ", "İŞ"], "asked": "ış", "want": "IŞ"},
    {"name": "I and İ stored, İ asked", "identifiers": ["IŞ", "İŞ"], "asked": "İş", "want": "İŞ"},
    {"name": "I and İ stored, I asked", "identifiers": ["IŞ", "İŞ"], "asked": "IŞ", "want": "IŞ"},
    {"name": "two that fold alike, neither the one asked in upper case", "identifiers": ["Iİ", "İI"], "asked": "ii", "want": null},
    {"name": "two that fold alike, one the one asked in upper case", "identifiers": ["Iİ", "II"], "asked": "ii", "want": "II"},
    {"name": "S is not Ş", "identifiers": ["IŞ"], "asked": "IS", "want": null},
    {"name": "another project's identifier", "identifiers": ["WEB"], "asked": "WEBS", "want": null},
    {"name": "no project", "identifiers": [], "asked": "WEB", "want": null}
  ]
}
````

`server/internal/modules/issue/domain/actions.go`（修改，2 处）：

````old server/internal/modules/issue/domain/actions.go
	ActionArchive shared.Action = "issue.archive"
````
````new server/internal/modules/issue/domain/actions.go
	ActionArchive shared.Action = "issue.archive"
	// ActionSearch is looking for a work item in a workspace: finding one
	// by its identifier, which then reads it as ActionRead.
	ActionSearch shared.Action = "issue.search"
````
````old server/internal/modules/issue/domain/actions.go
	return []shared.Action{ActionCreate, ActionRead, ActionUpdate, ActionDelete, ActionArchive}
````
````new server/internal/modules/issue/domain/actions.go
	return []shared.Action{ActionCreate, ActionRead, ActionUpdate, ActionDelete, ActionArchive, ActionSearch}
````

`server/internal/modules/issue/domain/errors.go`（修改，1 处）：

````old server/internal/modules/issue/domain/errors.go
// answer the project module's codes for the project, which they declare
// too (M4 design 5.1), with the project module's details.
var (
````
````new server/internal/modules/issue/domain/errors.go
// answer the project module's codes for the project, and its operations in
// a workspace the workspace module's code for the workspace, which they
// declare too (M4 design 5.1), with those modules' details.
var (
	// ErrWorkspaceNotFound answers a workspace that does not exist, is
	// deleted, or of which the caller is not an active member (M3 design
	// 5.1, 8.2).
	ErrWorkspaceNotFound = shared.NewError(shared.KindNotFound, "workspace.not_found", "The workspace does not exist, or you are not a member of it.")
````

`server/internal/modules/issue/domain/identifier.go`（新文件，81 行）：

````file server/internal/modules/issue/domain/identifier.go
package domain

import (
	"strconv"
	"strings"
)

// A work item's identifier is its project's identifier, a hyphen, and its
// number: WEB-12 (M4 design 3.7). The project's identifier is compared
// folded (foldIdentifier), so that one asked for in any case, Turkish
// letters too, finds its project.

// ParseIdentifier splits the identifier s of a work item into its
// project's identifier and its number: at its last hyphen, the project's
// identifier not empty, the number in decimal digits without a sign or a
// leading zero, from 1 to 2147483647, the range of a work item's number.
// ok is false for anything else.
func ParseIdentifier(s string) (project string, number int, ok bool) {
	project, digits, found := strings.CutLast(s, "-")
	if !found || project == "" || digits == "" || digits[0] == '0' || strings.TrimLeft(digits, "0123456789") != "" {
		return "", 0, false
	}
	n, err := strconv.ParseInt(digits, 10, 32)
	if err != nil {
		return "", 0, false
	}
	return project, int(n), true
}

// MatchIdentifier is the index among identifiers, a workspace's projects'
// identifiers, of the project's identifier asked, compared folded
// (foldIdentifier): the only one that folds as asked does; of two or more,
// the one that is asked in upper case letter for letter (strings.ToUpper),
// as a project's identifier is stored. ok is false when none folds as
// asked does, or when two or more do and none is asked in upper case: the
// identifiers I and İ tell apart, which strings.ToUpper keeps apart (i is
// I, ı is I, İ is İ) and folding does not (M4 design 3.7).
func MatchIdentifier(asked string, identifiers []string) (index int, ok bool) {
	folded, upper := foldIdentifier(asked), strings.ToUpper(asked)
	index, matches := -1, 0
	for i, id := range identifiers {
		if foldIdentifier(id) != folded {
			continue
		}
		matches++
		if index < 0 || id == upper {
			index = i
		}
	}
	if matches > 1 && identifiers[index] != upper {
		return 0, false
	}
	return index, matches > 0
}

// foldIdentifier is s as identifiers compare: a–z as A–Z, çşğöü as
// ÇŞĞÖÜ, and i, ı, I and İ as one letter, I; every other character as it
// is. A project's identifier holds A–Z, 0–9 and ÇŞĞİÖÜ (M3 design 3.19).
func foldIdentifier(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r - 'a' + 'A'
		case r == 'ı' || r == 'İ':
			return 'I'
		}
		switch r {
		case 'ç':
			return 'Ç'
		case 'ş':
			return 'Ş'
		case 'ğ':
			return 'Ğ'
		case 'ö':
			return 'Ö'
		case 'ü':
			return 'Ü'
		}
		return r
	}, s)
}
````

`server/internal/modules/issue/domain/identifier_test.go`（新文件，74 行）：

````file server/internal/modules/issue/domain/identifier_test.go
package domain

import (
	"encoding/json"
	"os"
	"testing"
)

// Each case of api/testdata/identifier-fold.json (M4 design 3.7): the
// project's identifier asked finds the project it says, or none.
func TestMatchIdentifierFindsWhatTheSamplesSay(t *testing.T) {
	data, err := os.ReadFile("../../../../../api/testdata/identifier-fold.json")
	if err != nil {
		t.Fatal(err)
	}
	var samples struct {
		Cases []struct {
			Name        string   `json:"name"`
			Identifiers []string `json:"identifiers"`
			Asked       string   `json:"asked"`
			Want        *string  `json:"want"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &samples); err != nil {
		t.Fatal(err)
	}
	if len(samples.Cases) == 0 {
		t.Fatal("identifier-fold.json has no case")
	}
	for _, c := range samples.Cases {
		i, ok := MatchIdentifier(c.Asked, c.Identifiers)
		switch {
		case c.Want == nil && ok:
			t.Errorf("%s: MatchIdentifier(%q, %q) = %s, want none", c.Name, c.Asked, c.Identifiers, c.Identifiers[i])
		case c.Want != nil && (!ok || c.Identifiers[i] != *c.Want):
			t.Errorf("%s: MatchIdentifier(%q, %q) = %d, %v; want %s", c.Name, c.Asked, c.Identifiers, i, ok, *c.Want)
		}
	}
}

// A work item's identifier is its project's identifier, a hyphen and its
// number, split at the last hyphen; the number is decimal digits without a
// sign or a leading zero, from 1 to the greatest number a work item may
// have. Anything else is no identifier.
func TestParseIdentifier(t *testing.T) {
	for _, tt := range []struct {
		s       string
		project string
		number  int
		ok      bool
	}{
		{"WEB-12", "WEB", 12, true},
		{"web-1", "web", 1, true},
		{"İŞ-7", "İŞ", 7, true},
		{"A-B-3", "A-B", 3, true},
		{"WEB-2147483647", "WEB", 2147483647, true},
		{"WEB-2147483648", "", 0, false},
		{"WEB-0", "", 0, false},
		{"WEB-012", "", 0, false},
		{"WEB-+12", "", 0, false},
		{"WEB--12", "WEB-", 12, true},
		{"WEB-1x", "", 0, false},
		{"WEB-", "", 0, false},
		{"-12", "", 0, false},
		{"WEB12", "", 0, false},
		{"WEB-١٢", "", 0, false},
		{"", "", 0, false},
	} {
		project, number, ok := ParseIdentifier(tt.s)
		if project != tt.project || number != tt.number || ok != tt.ok {
			t.Errorf("ParseIdentifier(%q) = %q, %d, %v; want %q, %d, %v", tt.s, project, number, ok, tt.project, tt.number, tt.ok)
		}
	}
}
````

- [ ] **Step 3: `issue.search`**

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"issue.archive": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
````
````new server/internal/modules/access/domain/rules.go
	"issue.archive": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
	// Every active member of the workspace: getIssueByIdentifier's decision
	// in the workspace, before issue.read on the project of the work item
	// it finds, which the guest rule follows (M4 design 3.4, 3.7).
	"issue.search": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"workspace_state.list":         {allowed, allowed, allowed, invisible, invisible, invisible, forbidden},
````
````new server/internal/modules/access/domain/rules_test.go
	"workspace_state.list":         {allowed, allowed, allowed, invisible, invisible, invisible, forbidden},
	"issue.search":                 {allowed, allowed, allowed, invisible, invisible, invisible, forbidden},
````

- [ ] **Step 4: 用例**

`server/internal/modules/issue/app/get_issue_by_identifier.go`（新文件，91 行）：

````file server/internal/modules/issue/app/get_issue_by_identifier.go
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// GetIssueByIdentifier reads a work item by its identifier, WEB-12, in a
// workspace (M4 design 3.7, 5.1).
type GetIssueByIdentifier struct {
	d GetIssueByIdentifierDeps
}

// GetIssueByIdentifierDeps are the use case's ports.
type GetIssueByIdentifierDeps struct {
	Locks      Locks
	Workspaces WorkspaceDirectory
	Projects   ProjectIdentifiers
	Issues     IssueNumbers
}

// NewGetIssueByIdentifier returns the use case.
func NewGetIssueByIdentifier(d GetIssueByIdentifierDeps) *GetIssueByIdentifier {
	return &GetIssueByIdentifier{d: d}
}

// Execute finds the workspace slug names and decides issue.search in it:
// one that is not there, or of which the caller is not an active member,
// is workspace.not_found. Then the work item identifier names: its
// project's identifier, compared folded with the workspace's undeleted
// projects' (domain.MatchIdentifier), and its number in that project. It
// reads the work item as getIssue does, without a lock or a transaction
// (Locks.see): an identifier that is no identifier, that names no
// project, or a number no undeleted work item of it has, is
// issue.not_found as one whose project the caller does not see, or that
// the guest rule hides from him, with the same answer; a role that may
// not read it, forbidden. An archived work item is read as any other.
func (u *GetIssueByIdentifier) Execute(ctx context.Context, slug, identifier string) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	workspaceID, found, err := u.d.Workspaces.WorkspaceBySlug(ctx, slug)
	switch {
	case err != nil:
		return domain.Issue{}, err
	case !found:
		return domain.Issue{}, domain.ErrWorkspaceNotFound
	}
	if err := u.d.Locks.seeWorkspace(ctx, actor, domain.ActionSearch, workspaceID); err != nil {
		return domain.Issue{}, err
	}
	id, err := u.find(ctx, workspaceID, identifier)
	if err != nil {
		return domain.Issue{}, err
	}
	return u.d.Locks.see(ctx, actor, id, domain.ActionRead)
}

// find is the id of the undeleted work item identifier names among the
// workspace's undeleted projects; domain.ErrIssueNotFound when it names
// none.
func (u *GetIssueByIdentifier) find(ctx context.Context, workspaceID uuid.UUID, identifier string) (uuid.UUID, error) {
	asked, number, ok := domain.ParseIdentifier(identifier)
	if !ok {
		return uuid.UUID{}, domain.ErrIssueNotFound
	}
	projects, err := u.d.Projects.Identifiers(ctx, workspaceID)
	if err != nil {
		return uuid.UUID{}, err
	}
	identifiers := make([]string, len(projects))
	for i, p := range projects {
		identifiers[i] = p.Identifier
	}
	i, ok := domain.MatchIdentifier(asked, identifiers)
	if !ok {
		return uuid.UUID{}, domain.ErrIssueNotFound
	}
	id, found, err := u.d.Issues.IssueByNumber(ctx, projects[i].ID, number)
	switch {
	case err != nil:
		return uuid.UUID{}, err
	case !found:
		return uuid.UUID{}, domain.ErrIssueNotFound
	}
	return id, nil
}
````

`server/internal/modules/issue/app/get_issue_by_identifier_test.go`（新文件，168 行）：

````file server/internal/modules/issue/app/get_issue_by_identifier_test.go
package app_test

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ops is acme's other project.
var ops = uuid.NewV7()

// fakeWorkspaces is the workspace directory: slug names id, each read
// logged.
type fakeWorkspaces struct {
	log  *callLog
	slug string
	id   uuid.UUID
}

func (f *fakeWorkspaces) WorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, bool, error) {
	if err := f.log.add(ctx, "WorkspaceBySlug", slug); err != nil || slug != f.slug {
		return uuid.UUID{}, false, err
	}
	return f.id, true, nil
}

// fakeIdentifiers are a workspace's projects with their identifiers, each
// read logged.
type fakeIdentifiers struct {
	log      *callLog
	projects []app.ProjectIdentifier
}

func (f *fakeIdentifiers) Identifiers(ctx context.Context, workspaceID uuid.UUID) ([]app.ProjectIdentifier, error) {
	if err := f.log.add(ctx, "Identifiers", workspaceID); err != nil {
		return nil, err
	}
	return f.projects, nil
}

// fakeNumbers is the work item id, its project's of the number, each read
// logged.
type fakeNumbers struct {
	log     *callLog
	project uuid.UUID
	number  int
	id      uuid.UUID
}

func (f *fakeNumbers) IssueByNumber(ctx context.Context, projectID uuid.UUID, number int) (uuid.UUID, bool, error) {
	if err := f.log.add(ctx, "IssueByNumber", projectID, number); err != nil || projectID != f.project || number != f.number {
		return uuid.UUID{}, false, err
	}
	return f.id, true, nil
}

// byIdentifier is getIssueByIdentifier's world: getIssue's (newGet), its
// work item web's 7, in acme, whose slug is acme, beside ops; their
// identifiers OPS and WEB.
type byIdentifier struct {
	*world
	projects *fakeIdentifiers
}

func newGetByIdentifier() (*app.GetIssueByIdentifier, byIdentifier, uuid.UUID) {
	_, w, id := newGet()
	b := byIdentifier{world: w, projects: &fakeIdentifiers{log: w.log, projects: []app.ProjectIdentifier{{ID: ops, Identifier: "OPS"},
		{ID: web, Identifier: "WEB"}}}}
	uc := app.NewGetIssueByIdentifier(app.GetIssueByIdentifierDeps{Locks: w.locks, Workspaces: &fakeWorkspaces{log: w.log, slug: "acme", id: acme},
		Projects: b.projects, Issues: &fakeNumbers{log: w.log, project: web, number: 7, id: id}})
	return uc, b, id
}

// identified are the calls of alice's read of web's work item number in
// acme, read as id: acme by its slug and the decision in it, outside any
// transaction and without a lock; acme's projects' identifiers; web's work
// item of the number; then the work item, and the decision on it, as
// getIssue reads it.
func identified(number int, id uuid.UUID) []string {
	return []string{line("WorkspaceBySlug", "acme", "outside tx"), line("Authorize", alice, domain.ActionSearch, acme, uuid.UUID{}, "outside tx"),
		line("Identifiers", acme, "outside tx"), line("IssueByNumber", web, number, "outside tx"), line("GetIssue", id, "outside tx"),
		line("Authorize", alice, domain.ActionRead, acme, web, "outside tx")}
}

// The work item, as getIssue answers it, whichever case its project's
// identifier is asked in, Turkish letters too: WEB-7 is web's 7, not
// ops's, and so is iş-7 once web's identifier is İŞ (M4 design 3.7).
func TestGetIssueByIdentifierReadsTheWorkItem(t *testing.T) {
	for _, tt := range []struct{ identifier, stored string }{{"WEB-7", "WEB"}, {"web-7", "WEB"}, {"wEb-7", "WEB"}, {"iş-7", "İŞ"}} {
		uc, b, id := newGetByIdentifier()
		b.projects.projects[1].Identifier = tt.stored
		got, err := uc.Execute(as(alice), "acme", tt.identifier)
		if want := readIssue(id); err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("Execute(%s) = %+v, %v; want %+v", tt.identifier, got, err, want)
		}
		if calls := identified(7, id); !slices.Equal(b.log.calls, calls) {
			t.Errorf("%s: calls:\n%q\nwant\n%q", tt.identifier, b.log.calls, calls)
		}
	}
}

// A workspace that is not there, or of which the caller is no member, is
// workspace.not_found, before any work item is looked for; a role outside
// issue.search, forbidden. Then an identifier that is no identifier, of no
// project of acme, of a number no work item of web has, of a work item
// whose project the caller does not see, or that the guest rule hides
// from him, is issue.not_found, one answer for each, detail and all; a
// role that may not read it, forbidden. A read or a decision that fails,
// its failure as itself; a request without a caller, unauthenticated
// before any read. Each answer is compared whole, not by its code alone.
func TestGetIssueByIdentifierRefusals(t *testing.T) {
	for _, tt := range []struct {
		name       string
		set        func(b byIdentifier)
		ctx        context.Context
		slug       string
		identifier string
		want       error
		calls      int // of identified's
	}{
		{"no workspace", func(byIdentifier) {}, as(alice), "acne", "WEB-7", domain.ErrWorkspaceNotFound, 1},
		{"a workspace he is no member of", func(b byIdentifier) { b.auth.answers = []error{shared.ErrNotVisible} }, as(alice), "acme", "WEB-7",
			domain.ErrWorkspaceNotFound, 2},
		{"a workspace role outside the rule", func(b byIdentifier) { b.auth.answers = []error{shared.Forbidden()} }, as(alice), "acme", "WEB-7",
			shared.Forbidden(), 2},
		{"no identifier", func(byIdentifier) {}, as(alice), "acme", "WEB7", domain.ErrIssueNotFound, 2},
		{"no project's identifier", func(byIdentifier) {}, as(alice), "acme", "WEBS-7", domain.ErrIssueNotFound, 3},
		{"no work item of the number", func(byIdentifier) {}, as(alice), "acme", "WEB-8", domain.ErrIssueNotFound, 4},
		{"a project not seen", func(b byIdentifier) { b.auth.answers = []error{nil, shared.ErrNotVisible} }, as(alice), "acme", "WEB-7",
			domain.ErrIssueNotFound, 6},
		{"another's, to a guest who sees only his own", func(b byIdentifier) { b.auth.grants = []shared.Grant{{}, ownOnly} }, as(alice), "acme",
			"WEB-7", domain.ErrIssueNotFound, 6},
		{"a role not allowed", func(b byIdentifier) { b.auth.answers = []error{nil, shared.Forbidden()} }, as(alice), "acme", "WEB-7",
			shared.Forbidden(), 6},
		{"the workspace's read fails", func(b byIdentifier) { b.log.fail = "WorkspaceBySlug" }, as(alice), "acme", "WEB-7", errDisk, 1},
		{"the decision in the workspace fails", func(b byIdentifier) { b.log.fail = "Authorize" }, as(alice), "acme", "WEB-7", errDisk, 2},
		{"the identifiers' read fails", func(b byIdentifier) { b.log.fail = "Identifiers" }, as(alice), "acme", "WEB-7", errDisk, 3},
		{"the number's read fails", func(b byIdentifier) { b.log.fail = "IssueByNumber" }, as(alice), "acme", "WEB-7", errDisk, 4},
		{"the work item's read fails", func(b byIdentifier) { b.log.fail = "GetIssue" }, as(alice), "acme", "WEB-7", errDisk, 5},
		{"no caller", func(byIdentifier) {}, context.Background(), "acme", "WEB-7", shared.Unauthenticated(), 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, b, id := newGetByIdentifier()
			tt.set(b)
			got, err := uc.Execute(tt.ctx, tt.slug, tt.identifier)
			if !reflect.DeepEqual(err, tt.want) || got.ID != (uuid.UUID{}) {
				t.Errorf("Execute(%s, %s) = %+v, %v; want no work item and %v", tt.slug, tt.identifier, got, err, tt.want)
			}
			number := 7
			if tt.identifier == "WEB-8" {
				number = 8
			}
			calls := identified(number, id)[:tt.calls]
			if tt.calls > 0 {
				calls[0] = line("WorkspaceBySlug", tt.slug, "outside tx")
			}
			if !slices.Equal(b.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", b.log.calls, calls)
			}
		})
	}
}
````

`server/internal/modules/issue/app/get_issue_test.go`（修改，5 处）：

````old server/internal/modules/issue/app/get_issue_test.go
	"context"
	"errors"
````
````new server/internal/modules/issue/app/get_issue_test.go
	"context"
````
````old server/internal/modules/issue/app/get_issue_test.go
}

// A read, outside any transaction, without a lock and outside the write
````
````new server/internal/modules/issue/app/get_issue_test.go
}

// readIssue is newGet's work item id as a read answers it.
func readIssue(id uuid.UUID) domain.Issue {
	return domain.Issue{ID: id, WorkspaceID: acme, ProjectID: web, SequenceID: 7, Name: "Fix", StateID: backlog.ID, Priority: domain.PriorityHigh,
		SortOrder: 65535, AssigneeIDs: []uuid.UUID{bob}, LabelIDs: []uuid.UUID{bug}, CreatedByID: &carol, UpdatedByID: &carol, CreatedAt: clockNow,
		UpdatedAt: clockNow}
}

// A read, outside any transaction, without a lock and outside the write
````
````old server/internal/modules/issue/app/get_issue_test.go
	want := domain.Issue{ID: id, WorkspaceID: acme, ProjectID: web, SequenceID: 7, Name: "Fix", StateID: backlog.ID, Priority: domain.PriorityHigh,
		SortOrder: 65535, AssigneeIDs: []uuid.UUID{bob}, LabelIDs: []uuid.UUID{bug}, CreatedByID: &carol, UpdatedByID: &carol, CreatedAt: clockNow,
		UpdatedAt: clockNow}
	if err != nil || !reflect.DeepEqual(got, want) {
````
````new server/internal/modules/issue/app/get_issue_test.go
	if want := readIssue(id); err != nil || !reflect.DeepEqual(got, want) {
````
````old server/internal/modules/issue/app/get_issue_test.go
// caller, unauthenticated before any read.
````
````new server/internal/modules/issue/app/get_issue_test.go
// caller, unauthenticated before any read. Each answer is compared whole,
// not by its code alone.
````
````old server/internal/modules/issue/app/get_issue_test.go
			var se, wantSE *shared.Error
			if !errors.Is(err, tt.want) && (!errors.As(err, &se) || !errors.As(tt.want, &wantSE) || se.Code != wantSE.Code) ||
				got.ID != (uuid.UUID{}) {
````
````new server/internal/modules/issue/app/get_issue_test.go
			if !reflect.DeepEqual(err, tt.want) || got.ID != (uuid.UUID{}) {
````

`server/internal/modules/issue/app/guest_rule_test.go`（修改，4 处）：

````old server/internal/modules/issue/app/guest_rule_test.go
	"errors"
````
````new server/internal/modules/issue/app/guest_rule_test.go
	"reflect"
````
````old server/internal/modules/issue/app/guest_rule_test.go
// looked at: in the precheck, and again under the lock.
````
````new server/internal/modules/issue/app/guest_rule_test.go
// looked at: in the precheck, and again under the lock. Each answer is
// compared whole: another's work item is the very issue.not_found of one
// that is not there, detail and all (8.4).
````
````old server/internal/modules/issue/app/guest_rule_test.go
			if (tt.want == nil) != (err == nil) || tt.want != nil && !errors.Is(err, tt.want) || (err == nil) != (got.ID == id) {
````
````new server/internal/modules/issue/app/guest_rule_test.go
			if !reflect.DeepEqual(err, tt.want) || (err == nil) != (got.ID == id) {
````
````old server/internal/modules/issue/app/guest_rule_test.go
			if _, err := update.Execute(as(alice), id, domain.IssuePatch{Name: name("Mend")}); !errors.Is(err, tt.want) {
````
````new server/internal/modules/issue/app/guest_rule_test.go
			if _, err := update.Execute(as(alice), id, domain.IssuePatch{Name: name("Mend")}); !reflect.DeepEqual(err, tt.want) {
````

`server/internal/modules/issue/app/lock.go`（修改，2 处）：

````old server/internal/modules/issue/app/lock.go
// lock (see). Every decision on a work item applies the guest rule first
// (decide). issue.New builds one Locks for every use case: none holds an
// Authorizer, a write gate or a transaction of its own.
````
````new server/internal/modules/issue/app/lock.go
// lock (see), and a read in a workspace its decision in the workspace
// (seeWorkspace). Every decision on a work item applies the guest rule
// first (decide). issue.New builds one Locks for every use case: none
// holds an Authorizer, a write gate or a transaction of its own.
````
````old server/internal/modules/issue/app/lock.go
}

// decide asks the Authorizer for action on the project projectID of the
// workspace workspaceID for actor, and, for an action on a work item, i, on
// it: the decision knows its creator (shared.Target.CreatorID), whom a rule
// may allow (AllowCreator). A project not visible to actor is notFound, and
// so is a work item the guest rule hides from him (M4 design 3.4, 6.7):
// the project's guest who sees only his own work items (Grant.OwnOnly), a
// workspace admin among them, does not see one another created, before
// what the rule says of his role, so another's work item is as unknown to
// him as one that is not there (8.4). i is nil for an action on the
// project.
````
````new server/internal/modules/issue/app/lock.go
}

// seeWorkspace decides action in the workspace workspaceID for actor,
// without a lock (M4 design 3.4): a workspace of which actor is not an
// active member, or deleted, is workspace.not_found; a role the rule does
// not allow is the Authorizer's shared.Forbidden.
func (l Locks) seeWorkspace(ctx context.Context, actor shared.Actor, action shared.Action, workspaceID uuid.UUID) error {
	return l.decide(ctx, actor, action, workspaceID, uuid.UUID{}, nil, domain.ErrWorkspaceNotFound)
}

// decide asks the Authorizer for action on the project projectID of the
// workspace workspaceID for actor, or in the workspace for a zero
// projectID, and, for an action on a work item, i, on it: the decision
// knows its creator (shared.Target.CreatorID), whom a rule may allow
// (AllowCreator). A project, or a workspace, not visible to actor is
// notFound, and so is a work item the guest rule hides from him (M4 design
// 3.4, 6.7): the project's guest who sees only his own work items
// (Grant.OwnOnly), a workspace admin among them, does not see one another
// created, before what the rule says of his role, so another's work item
// is as unknown to him as one that is not there (8.4). i is nil for an
// action on the project or in the workspace.
````

`server/internal/modules/issue/app/ports.go`（修改，1 处）：

````old server/internal/modules/issue/app/ports.go
}

// IssueCreator is the work items' store as createIssue writes and reads
````
````new server/internal/modules/issue/app/ports.go
}

// WorkspaceDirectory finds a workspace by its slug (M4 design 3.3): the
// workspace module's directory (workspace.Provide), converted
// (bootstrap/ports.go).
type WorkspaceDirectory interface {
	// WorkspaceBySlug reads the id of the undeleted workspace slug names,
	// without a lock; found is false when there is none.
	WorkspaceBySlug(ctx context.Context, slug string) (id uuid.UUID, found bool, err error)
}

// ProjectIdentifiers reads the identifiers of a workspace's projects, the
// projects a work item's identifier may name (M4 design 3.7):
// project.Provide's, converted (bootstrap/ports.go).
type ProjectIdentifiers interface {
	// Identifiers reads the workspace's undeleted projects, archived or
	// not, each with its identifier, without a lock.
	Identifiers(ctx context.Context, workspaceID uuid.UUID) ([]ProjectIdentifier, error)
}

// ProjectIdentifier is a project as ProjectIdentifiers reads it: its id
// and its identifier, as stored.
type ProjectIdentifier struct {
	ID         uuid.UUID
	Identifier string
}

// IssueNumbers finds a work item by its number in its project (M4 design
// 3.7).
type IssueNumbers interface {
	// IssueByNumber reads the id of the project's undeleted work item,
	// archived or not, of the number, without a lock; found is false when
	// there is none.
	IssueByNumber(ctx context.Context, projectID uuid.UUID, number int) (id uuid.UUID, found bool, err error)
}

// IssueCreator is the work items' store as createIssue writes and reads
````

- [ ] **Step 5: 存储**

`server/internal/modules/issue/adapter/postgres/issues.go`（修改，1 处）：

````old server/internal/modules/issue/adapter/postgres/issues.go
}

// GetIssue reads the undeleted work item id, archived or not, with its
````
````new server/internal/modules/issue/adapter/postgres/issues.go
}

// IssueByNumber reads the id of the project's undeleted work item, archived
// or not, of the number (M4 design 3.7); found is false when there is
// none.
func (s *Store) IssueByNumber(ctx context.Context, projectID uuid.UUID, number int) (uuid.UUID, bool, error) {
	id, err := s.queries(ctx).IssueByNumber(ctx, gen.IssueByNumberParams{ProjectID: projectID, SequenceID: int32(number)})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return uuid.UUID{}, false, nil
	case err != nil:
		return uuid.UUID{}, false, fmt.Errorf("find work item %d of project %s: %w", number, projectID, err)
	}
	return id, true, nil
}

// GetIssue reads the undeleted work item id, archived or not, with its
````

`server/internal/modules/issue/adapter/postgres/issues_test.go`（修改，2 处）：

````old server/internal/modules/issue/adapter/postgres/issues_test.go
}

// A work item without assignees or labels reads with neither, nil and not
````
````new server/internal/modules/issue/adapter/postgres/issues_test.go
}

// A work item is found by its number in its project, archived or not
// (M4 design 3.7): web's 1 and 2, and ops's 1, each its own; not a
// deleted one, nor a number of no work item of the project, though
// another project's work item has it.
func TestIssueByNumber(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS")
	one, two, gone, opsOne, opsThree := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, web, one, 1, alice)
	newIssue(t, pool, web, two, 2, alice)
	newIssue(t, pool, web, gone, 3, alice)
	newIssue(t, pool, ops, opsOne, 1, alice)
	newIssue(t, pool, ops, opsThree, 3, alice)
	exec(t, pool, "UPDATE issues SET archived_at = now() WHERE id = $1", two)
	exec(t, pool, "UPDATE issues SET deleted_at = now() WHERE id = $1", gone)
	for _, tt := range []struct {
		p      project
		number int
		want   uuid.UUID
		found  bool
	}{{web, 1, one, true}, {web, 2, two, true}, {ops, 1, opsOne, true}, {web, 3, uuid.UUID{}, false}, {web, 4, uuid.UUID{}, false},
		{ops, 2, uuid.UUID{}, false}} {
		if id, found, err := s.IssueByNumber(context.Background(), tt.p.id, tt.number); err != nil || id != tt.want || found != tt.found {
			t.Errorf("IssueByNumber(%s, %d) = %s, %v, %v; want %s, %v", tt.p.id, tt.number, id, found, err, tt.want, tt.found)
		}
	}
}

// A work item without assignees or labels reads with neither, nil and not
````
````old server/internal/modules/issue/adapter/postgres/issues_test.go
		t.Errorf("GreatestSortOrder() = %v, %v; want context.Canceled", greatest, err)
````
````new server/internal/modules/issue/adapter/postgres/issues_test.go
		t.Errorf("GreatestSortOrder() = %v, %v; want context.Canceled", greatest, err)
	}
	if got, found, err := s.IssueByNumber(cancelled, web.id, 1); !errors.Is(err, context.Canceled) || found || got != (uuid.UUID{}) {
		t.Errorf("IssueByNumber() = %s, %v, %v; want context.Canceled", got, found, err)
````

`server/internal/modules/project/adapter/postgres/failures_test.go`（修改，2 处）：

````old server/internal/modules/project/adapter/postgres/failures_test.go
// or assignees not allowed. Each read runs on a cancelled context
// against a project alice is the only admin of, beside bob, a member, and
// has display settings in, with its six default states and a label with a
// child, bob's membership of another project ended, so that the right
// answer is none of the zero values.
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
// or assignees not allowed, nor no projects' identifiers, which
// getIssueByIdentifier would answer as issue.not_found. Each read runs on
// a cancelled context against a project alice is the only admin of,
// beside bob, a member, and has display settings in, with its six default
// states and a label with a child, bob's membership of another project
// ended, so that the right answer is none of the zero values.
````
````old server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("ActiveRoles() = %v, %v; want context.Canceled, not no member", roles, err)
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("ActiveRoles() = %v, %v; want context.Canceled, not no member", roles, err)
	}
	if list, err := s.ProjectIdentifiers(cancelled, acme); !failed(err) || list != nil {
		t.Errorf("ProjectIdentifiers() = %+v, %v; want context.Canceled, not none", list, err)
````

`server/internal/modules/project/adapter/postgres/issue_ports.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/issue_ports.go
}

// ActiveRoles reads the project roles of those of the accounts ids that
````
````new server/internal/modules/project/adapter/postgres/issue_ports.go
}

// ProjectIdentifiers reads the workspace's undeleted projects, archived or
// not, each with its identifier, in id order, without a lock (M4 design
// 3.7).
func (s *Store) ProjectIdentifiers(ctx context.Context, workspaceID uuid.UUID) ([]app.IssueProjectIdentifier, error) {
	rows, err := s.queries(ctx).ProjectIdentifiers(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("read the identifiers of the projects of workspace %s: %w", workspaceID, err)
	}
	out := make([]app.IssueProjectIdentifier, len(rows))
	for i, r := range rows {
		out[i] = app.IssueProjectIdentifier{ID: r.ID, Identifier: r.Identifier}
	}
	return out, nil
}

// ActiveRoles reads the project roles of those of the accounts ids that
````

`server/internal/modules/project/adapter/postgres/issue_ports_test.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/issue_ports_test.go
}

// The catalogs of a project for the issue module (M4 design 6.5): its
````
````new server/internal/modules/project/adapter/postgres/issue_ports_test.go
}

// A workspace's projects' identifiers for the issue module (M4 design
// 3.7): its undeleted projects', an archived one's too, in id order; not a
// deleted project's, nor another workspace's. Web, drawn first, is the
// archived one, so its row is written after Ops's, and Ops comes first by
// name and by identifier: the id order is neither the rows' order nor that
// of the (workspace, name) or (workspace, identifier) index the planner may
// read, so that a read without its ORDER BY is seen whether the database
// scans the rows or one of those indexes.
func TestProjectIdentifiers(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web, ops, gone := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice), newProject(t, s, acme,
		"Gone", "GONE", alice)
	newProject(t, s, beta, "Web", "BETA", alice)
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", web, now)
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", gone, now)
	got, err := s.ProjectIdentifiers(context.Background(), acme)
	want := []app.IssueProjectIdentifier{{ID: web, Identifier: "WEB"}, {ID: ops, Identifier: "OPS"}}
	slices.SortFunc(want, func(a, b app.IssueProjectIdentifier) int { return a.ID.Compare(b.ID) })
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("ProjectIdentifiers(acme) = %+v, %v; want %+v", got, err, want)
	}
}

// The catalogs of a project for the issue module (M4 design 6.5): its
````

- [ ] **Step 6: 端口和接线**

`server/internal/bootstrap/app.go`（修改，1 处）：

````old server/internal/bootstrap/app.go
		Projects: issueProjects{projects: projectPorts.IssueProjects},
		Profiles: issueProfiles{profiles: identityPorts.PublicProfiles},
````
````new server/internal/bootstrap/app.go
		Projects:   issueProjects{projects: projectPorts.IssueProjects},
		Workspaces: issueWorkspaces{directory: workspacePorts.WorkspaceDirectory},
		Profiles:   issueProfiles{profiles: identityPorts.PublicProfiles},
````

`server/internal/bootstrap/issue_ports_test.go`（新文件，67 行）：

````file server/internal/bootstrap/issue_ports_test.go
package bootstrap

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
)

// The conversions of a read by identifier (M4 design 3.7): the ports
// getIssueByIdentifier asks, as bootstrap/ports.go hands them to issue.

// ProjectIdentifiers answers fakeIssueProjects' identifiers, of whatever
// workspace it is asked.
func (f *fakeIssueProjects) ProjectIdentifiers(_ context.Context, workspaceID uuid.UUID) ([]project.IssueProjectIdentifier, error) {
	f.asked = append(f.asked, "identifiers "+workspaceID.String())
	return f.identifiers, f.err
}

// issueProjects asks project for the workspace issue asks about and hands
// over each project's id and identifier, in their order; an error as it
// came, without identifiers.
func TestIssueProjectsConvertsProjectIdentifiers(t *testing.T) {
	acme, web, ops := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	fake := &fakeIssueProjects{identifiers: []project.IssueProjectIdentifier{{ID: web, Identifier: "WEB"}, {ID: ops, Identifier: "İŞ"}}}
	p := issueProjects{projects: fake}
	got, err := p.Identifiers(context.Background(), acme)
	if want := []issue.ProjectIdentifier{{ID: web, Identifier: "WEB"}, {ID: ops, Identifier: "İŞ"}}; err != nil || !slices.Equal(got, want) {
		t.Errorf("Identifiers() = %+v, %v; want %+v", got, err, want)
	}
	if want := []string{"identifiers " + acme.String()}; !slices.Equal(fake.asked, want) {
		t.Errorf("project was asked %q, want %q", fake.asked, want)
	}
	failure := errors.New("connection reset")
	fake.err = failure
	if got, err := p.Identifiers(context.Background(), acme); !errors.Is(err, failure) || got != nil {
		t.Errorf("Identifiers() = %+v, %v; want no identifiers and %v", got, err, failure)
	}
}

// issueWorkspaces hands issue workspace's answer to its read without a
// lock: the workspace's id, found and the error as they came.
func TestIssueWorkspacesConvertsWorkspacesAnswer(t *testing.T) {
	acme := workspace.DirectoryEntry{ID: uuid.NewV7(), Timezone: "Asia/Shanghai"}
	fake := &fakeWorkspaceDirectory{workspaces: map[string]workspace.DirectoryEntry{"acme": acme}}
	d := issueWorkspaces{directory: fake}
	ctx := context.Background()
	if id, found, err := d.WorkspaceBySlug(ctx, "acme"); err != nil || !found || id != acme.ID {
		t.Errorf("WorkspaceBySlug(acme) = %s, %v, %v; want %s, found", id, found, err, acme.ID)
	}
	if id, found, err := d.WorkspaceBySlug(ctx, "gone"); err != nil || found || id != (uuid.UUID{}) {
		t.Errorf("WorkspaceBySlug(gone) = %s, %v, %v; want not found", id, found, err)
	}
	if want := []string{"read acme", "read gone"}; !slices.Equal(fake.asked, want) {
		t.Errorf("workspace was asked %q, want %q", fake.asked, want)
	}
	failure := errors.New("connection reset")
	fake.err = failure
	if _, _, err := d.WorkspaceBySlug(ctx, "acme"); !errors.Is(err, failure) {
		t.Errorf("WorkspaceBySlug() = %v, want %v", err, failure)
	}
}
````

`server/internal/bootstrap/issue_writes_test.go`（修改，3 处）：

````old server/internal/bootstrap/issue_writes_test.go
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
````
````new server/internal/bootstrap/issue_writes_test.go
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
````
````old server/internal/bootstrap/issue_writes_test.go
// and project's and identity's ports converted, all as bootstrap wires
// them.
````
````new server/internal/bootstrap/issue_writes_test.go
// and project's, workspace's and identity's ports converted, all as
// bootstrap wires them.
````
````old server/internal/bootstrap/issue_writes_test.go
		Profiles: issueProfiles{profiles: identity.Provide(pool).PublicProfiles}}).Register)
````
````new server/internal/bootstrap/issue_writes_test.go
		Workspaces: issueWorkspaces{directory: workspace.Provide(pool).WorkspaceDirectory},
		Profiles:   issueProfiles{profiles: identity.Provide(pool).PublicProfiles}}).Register)
````

`server/internal/bootstrap/ports.go`（修改，1 处）：

````old server/internal/bootstrap/ports.go
}

// issueProfiles is identity's PublicProfiles as issue's: each account's
````
````new server/internal/bootstrap/ports.go
}

func (p issueProjects) Identifiers(ctx context.Context, workspaceID uuid.UUID) ([]issue.ProjectIdentifier, error) {
	projects, err := p.projects.ProjectIdentifiers(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	out := make([]issue.ProjectIdentifier, len(projects))
	for i, pr := range projects {
		out[i] = issue.ProjectIdentifier(pr)
	}
	return out, nil
}

// issueWorkspaces is workspace's WorkspaceDirectory as issue's: the same
// read without a lock, the workspace's id alone.
type issueWorkspaces struct {
	directory workspace.WorkspaceDirectory
}

func (d issueWorkspaces) WorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, bool, error) {
	w, found, err := d.directory.WorkspaceBySlug(ctx, slug)
	return w.ID, found, err
}

// issueProfiles is identity's PublicProfiles as issue's: each account's
````

`server/internal/bootstrap/ports_test.go`（修改，1 处）：

````old server/internal/bootstrap/ports_test.go
	web    uuid.UUID
	locked project.IssueProject
	states []project.IssueState
	labels []project.IssueLabel
	roles  map[uuid.UUID]shared.Role
	err    error
	asked  []string
````
````new server/internal/bootstrap/ports_test.go
	web         uuid.UUID
	locked      project.IssueProject
	states      []project.IssueState
	labels      []project.IssueLabel
	roles       map[uuid.UUID]shared.Role
	identifiers []project.IssueProjectIdentifier
	err         error
	asked       []string
````

`server/internal/modules/issue/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/issue/adapter/http/handler.go
}

// UpdateIssueUseCase is app.UpdateIssue.
````
````new server/internal/modules/issue/adapter/http/handler.go
}

// GetIssueByIdentifierUseCase is app.GetIssueByIdentifier.
type GetIssueByIdentifierUseCase interface {
	Execute(ctx context.Context, slug, identifier string) (domain.Issue, error)
}

// UpdateIssueUseCase is app.UpdateIssue.
````
````old server/internal/modules/issue/adapter/http/handler.go
	CreateIssue CreateIssueUseCase
	GetIssue    GetIssueUseCase
	UpdateIssue UpdateIssueUseCase
	DeleteIssue DeleteIssueUseCase
````
````new server/internal/modules/issue/adapter/http/handler.go
	CreateIssue          CreateIssueUseCase
	GetIssue             GetIssueUseCase
	GetIssueByIdentifier GetIssueByIdentifierUseCase
	UpdateIssue          UpdateIssueUseCase
	DeleteIssue          DeleteIssueUseCase
````

`server/internal/modules/issue/module.go`（修改，6 处）：

````old server/internal/modules/issue/module.go
// ProjectPorts are what the module's writes ask of a project (M4 design
// 6.5): its gate, its numbers, its states', labels' and members' catalogs.
````
````new server/internal/modules/issue/module.go
// ProjectPorts are what the module asks of a project (M4 design 6.5): its
// gate, its numbers, its states', labels' and members' catalogs, for its
// writes; a workspace's projects' identifiers, for a read by identifier.
````
````old server/internal/modules/issue/module.go
	app.ProjectMembers
````
````new server/internal/modules/issue/module.go
	app.ProjectMembers
	app.ProjectIdentifiers
````
````old server/internal/modules/issue/module.go
// LockedProject, StateRef and its StateGroup, and LabelRef are the values
// ProjectPorts hand over: bootstrap converts project's into them (M4
// design 6.5).
````
````new server/internal/modules/issue/module.go
// LockedProject, StateRef and its StateGroup, LabelRef and
// ProjectIdentifier are the values ProjectPorts hand over: bootstrap
// converts project's into them (M4 design 6.5).
````
````old server/internal/modules/issue/module.go
	LockedProject = app.LockedProject
	StateRef      = app.StateRef
	StateGroup    = domain.StateGroup
	LabelRef      = app.LabelRef
````
````new server/internal/modules/issue/module.go
	LockedProject     = app.LockedProject
	StateRef          = app.StateRef
	StateGroup        = domain.StateGroup
	LabelRef          = app.LabelRef
	ProjectIdentifier = app.ProjectIdentifier
````
````old server/internal/modules/issue/module.go
	Projects ProjectPorts
````
````new server/internal/modules/issue/module.go
	Projects ProjectPorts
	// Workspaces is workspace's WorkspaceDirectory, converted
	// (bootstrap/ports.go).
	Workspaces app.WorkspaceDirectory
````
````old server/internal/modules/issue/module.go
		GetIssue: app.NewGetIssue(locks),
````
````new server/internal/modules/issue/module.go
		GetIssue: app.NewGetIssue(locks),
		GetIssueByIdentifier: app.NewGetIssueByIdentifier(app.GetIssueByIdentifierDeps{
			Locks: locks, Workspaces: d.Workspaces, Projects: d.Projects, Issues: store,
		}),
````

`server/internal/modules/project/app/issue_ports.go`（修改，1 处）：

````old server/internal/modules/project/app/issue_ports.go
	Name string
}

````
````new server/internal/modules/project/app/issue_ports.go
	Name string
}

// IssueProjectIdentifier is a project as project.Provide reads it for the
// issue module's getIssueByIdentifier (M4 design 3.7): its id and its
// identifier. bootstrap converts it into issue's value.
type IssueProjectIdentifier struct {
	ID         uuid.UUID
	Identifier string
}

````

`server/internal/modules/project/module.go`（修改，4 处）：

````old server/internal/modules/project/module.go
// IssueProjects is what the issue module's writes ask of a project (M4
// design 6.5): its gate (ProjectGate's Find and Lock), its work items'
// numbers (Sequences), and its states', labels' and members' catalogs
// (StateCatalog, LabelCatalog, ProjectMembers). Each reads, or locks, in
// the transaction ctx carries; the catalogs are read under the project's
// lock, which every write of a state, a label or a membership holds too,
// so they stay as read until the write commits (M4 design 3.5).
````
````new server/internal/modules/project/module.go
// IssueProjects is what the issue module asks of a project (M4 design
// 6.5): its gate (ProjectGate's Find and Lock), its work items' numbers
// (Sequences), and its states', labels' and members' catalogs
// (StateCatalog, LabelCatalog, ProjectMembers), for its writes; and a
// workspace's projects' identifiers (ProjectIdentifiers), for a read by
// identifier. Each reads, or locks, in the transaction ctx carries; the
// catalogs are read under the project's lock, which every write of a
// state, a label or a membership holds too, so they stay as read until the
// write commits (M4 design 3.5).
````
````old server/internal/modules/project/module.go
	ActiveRoles(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]shared.Role, error)
````
````new server/internal/modules/project/module.go
	ActiveRoles(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]shared.Role, error)
	// ProjectIdentifiers reads the workspace's undeleted projects, archived
	// or not, each with its identifier, in id order, without a lock.
	ProjectIdentifiers(ctx context.Context, workspaceID uuid.UUID) ([]IssueProjectIdentifier, error)
````
````old server/internal/modules/project/module.go
// IssueProject, IssueState and IssueLabel are the values IssueProjects
// hands over: bootstrap converts them into issue's (M4 design 6.5).
````
````new server/internal/modules/project/module.go
// IssueProject, IssueState, IssueLabel and IssueProjectIdentifier are the
// values IssueProjects hands over: bootstrap converts them into issue's
// (M4 design 6.5).
````
````old server/internal/modules/project/module.go
	IssueProject = app.IssueProject
	IssueState   = app.IssueState
	IssueLabel   = app.IssueLabel
````
````new server/internal/modules/project/module.go
	IssueProject           = app.IssueProject
	IssueState             = app.IssueState
	IssueLabel             = app.IssueLabel
	IssueProjectIdentifier = app.IssueProjectIdentifier
````

`server/internal/modules/workspace/module.go`（修改，1 处）：

````old server/internal/modules/workspace/module.go
// project module, or locks one by its id (M3 design 6.5); found is false
// when there is none.
````
````new server/internal/modules/workspace/module.go
// project and issue modules, or locks one by its id (M3 design 6.5, M4
// design 3.3); found is false when there is none.
````

- [ ] **Step 7: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/... ./internal/modules/access/... ./internal/modules/project/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestIssueProjectsConvertsProjectIdentifiers$|TestIssueWorkspacesConvertsWorkspacesAnswer$|TestEveryActionHasARuleAndEveryRuleAnAction$|TestEveryModuleDeclaresItsActionsOrHasNone$' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 8: 提交**

```bash
git add api/testdata/identifier-fold.json server/internal/bootstrap/app.go server/internal/bootstrap/issue_ports_test.go server/internal/bootstrap/issue_writes_test.go server/internal/bootstrap/ports.go server/internal/bootstrap/ports_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/issue/adapter/http/handler.go server/internal/modules/issue/adapter/postgres/issues.go server/internal/modules/issue/adapter/postgres/issues_test.go server/internal/modules/issue/adapter/postgres/queries/issues.sql server/internal/modules/issue/app/get_issue_by_identifier.go server/internal/modules/issue/app/get_issue_by_identifier_test.go server/internal/modules/issue/app/get_issue_test.go server/internal/modules/issue/app/guest_rule_test.go server/internal/modules/issue/app/lock.go server/internal/modules/issue/app/ports.go server/internal/modules/issue/domain/actions.go server/internal/modules/issue/domain/errors.go server/internal/modules/issue/domain/identifier.go server/internal/modules/issue/domain/identifier_test.go server/internal/modules/issue/module.go server/internal/modules/project/adapter/postgres/failures_test.go server/internal/modules/project/adapter/postgres/issue_ports.go server/internal/modules/project/adapter/postgres/issue_ports_test.go server/internal/modules/project/adapter/postgres/queries/issue_ports.sql server/internal/modules/project/app/issue_ports.go server/internal/modules/project/module.go server/internal/modules/workspace/module.go server/internal/modules/issue/adapter/postgres/gen/issues.sql.go server/internal/modules/project/adapter/postgres/gen/issue_ports.sql.go
```
```bash
git commit -m "feat(M4/P2): a work item is found by its identifier, folded

getIssueByIdentifier finds the workspace by its slug and decides
issue.search in it (workspace.not_found for one the caller is no active
member of), then the project whose identifier folds as the one asked
does (i, ı, I and İ as one letter; of two, the one that is it upper-cased
letter for letter), the work item of the number, and reads it as
getIssue does. Whatever is not found after the workspace is the one
issue.not_found. The project module reads a workspace's projects'
identifiers for it, and bootstrap hands the issue module the workspace
directory; the route comes in the next task.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p2.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试，或没有替身的平台、领域、转换的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是生成器（`make gen-go`）和中英文翻译键（`make lint-web`）的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过，端到端的检查只在标出的 Task 的树上）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T5.1` | `ı` 不与 `i`、`I`、`İ` 折叠成同一个字母 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsFoundByItsIdentifier`（`make test`）（Task 6 起） | 单元；组合 |
| `T5.2` | `ş` 不与 `Ş` 折叠 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsFoundByItsIdentifier`（`make test`）（Task 6 起） | 单元；组合 |
| `T5.3` | 折叠之后相同、却没有一个等于所问的大写的几个标识，仍找到其中一个 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsFoundByItsIdentifier`（`make test`）（Task 6 起） | 单元；组合 |
| `T5.4` | 折叠之后相同的几个标识取最后一个，不取等于所问的大写的那个 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsFoundByItsIdentifier`（`make test`）（Task 6 起） | 单元；组合 |
| `T5.5` | 前面带 0 的编号指它的值的那个工作项 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsFoundByItsIdentifier`（`make test`）（Task 6 起） | 单元；组合 |
| `T5.6` | 不先判定调用者在工作区里的成员关系：`workspace.not_found` 变成 `issue.not_found` | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsFoundByItsIdentifier`（`make test`）（Task 6 起）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `getIssueByIdentifier` 各格（`make test`）（Task 6 起） | 单元；组合 |
| `T5.7` | 找到的工作项按工作区的规则 `issue.search` 判定，不按它的项目上的 `issue.read` | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsFoundByItsIdentifier`（`make test`）（Task 6 起）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `getIssueByIdentifier` 各格（`make test`）（Task 6 起） | 单元；组合 |
| `T5.8` | 标识在每个工作区的项目里找 | `internal/modules/project/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsFoundByItsIdentifier`（`make test`）（Task 6 起） | 存储；组合 |
| `T5.9` | 标识也在工作区已删除的项目里找 | `internal/modules/project/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsFoundByItsIdentifier`（`make test`）（Task 6 起） | 存储；组合 |
| `T5.10` | 编号也指已删除的工作项 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`） | 存储 |
| `T5.11` | `issue.search` 没有工作区的访客 | `internal/modules/access/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `getIssueByIdentifier` 各格（`make test`）（Task 6 起） | 单元；组合 |
| `T5.12` | `issueProjects.Identifiers` 把项目的存储的失败答成没有项目 | `internal/bootstrap` 的 `TestIssueProjectsConvertsProjectIdentifiers`（`make test`） | 单元 |
| `T5.13` | `issueWorkspaces` 丢掉工作区的存储的失败 | `internal/bootstrap` 的 `TestIssueWorkspacesConvertsWorkspacesAnswer`（`make test`） | 单元 |
| `T5.14` | 编号在每个项目里找，不只在它的项目里 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsFoundByItsIdentifier`（`make test`）（Task 6 起） | 存储；组合 |
| `T5.15` | 项目里不论编号的任何一个工作项都找得到 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsFoundByItsIdentifier`（`make test`）（Task 6 起） | 存储；组合 |
| `T5.16` | 标识只按大写逐个字母比较，不折叠 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsFoundByItsIdentifier`（`make test`）（Task 6 起） | 单元；组合 |
| `T5.17` | `ProjectIdentifiers` 不按 `id` 的次序回答工作区的项目（去掉 `ORDER BY id`；按性质只在存储一层：`MatchIdentifier` 不靠次序） | `internal/modules/project/adapter/postgres` 的全部测试（`make test`） | 存储 |
| `T5.18` | 没有项目的标识对得上时答一个文字不同的 `issue.not_found`：工作区之后的各种找不到不再是同一个回答 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsFoundByItsIdentifier`（`make test`）（Task 6 起） | 单元；组合 |

本 Task 的检查也发现较早的 Task 的变异 `T2.12`（列在各自的 Task 的表里，标"（Task 5 起）"）：本 Task 做完之后，在本 Task 的树上重跑它们（`mutants_p2.py`，`MUT_LATER=5`）。

---

### Task 6: 按编号取：接口、矩阵和组合

**Files:**
- Create: `server/internal/bootstrap/issue_identifier_test.go`、`server/internal/modules/issue/adapter/http/by_identifier_test.go`
- Modify: `api/modules/issue.yaml`、`api/openapi.yaml`、`server/internal/bootstrap/permission_matrix_columns_test.go`、`server/internal/bootstrap/permission_matrix_issue_columns_test.go`、`server/internal/bootstrap/permission_matrix_issues_test.go`、`server/internal/bootstrap/permission_matrix_targets_test.go`、`server/internal/modules/issue/adapter/http/handler_test.go`、`server/internal/modules/issue/adapter/http/issues.go`、`server/internal/modules/issue/module.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/issue/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**（spec 2.7、第 3 节第 4、15 条；M4 设计 3.7、5.1、9.2）
- 契约：`GET /api/v0/workspaces/{slug}/issues/{identifier}`（`getIssueByIdentifier`，200 `IssueDetail`，`x-problem-codes: [workspace.not_found, issue.not_found, forbidden]`），描述写明先按工作区、再照 `getIssue`，标识的写法（编号是 0–9 的十进制，不带符号、不以 0 开头，1 到 2147483647，`ParseIdentifier`）和折叠比较（多于一个项目的标识对得上时取逐字等于大写的那个），各种 404 是同一个回答，已归档的照样读；参数 `Slug`、`IssueIdentifier`；`api/openapi.yaml` 的路径。
- `issue/adapter/http/issues.go`：`GetIssueByIdentifier`（标识原样交给用例）；`issue/module.go` 的说明。
- 测试的帮手：`handler_test.go` 的 `fakeGetByIdentifier`（记下"调用者 slug 标识"）；`permission_matrix_columns_test.go` 的 `noProjectTable`（矩阵的自检里"出自不是项目表的列"的违规，`TestMatrixViolationsCatchesEachColumnGap` 的三处和 `TestMatrixViolationsCatchesAnotherCasesIdentifier` 共用）。
- 矩阵：`permission_matrix_issues_test.go` 的 `getIssueByIdentifier` 两行（照常、已归档的项目），`identifierOf(c)`（列的工作项的标识）、`byIdentifier`、`inItsWorkspace`（X 和已归档的项目的"从未是 acme 的成员"一列答 `workspace.not_found`）；`permission_matrix_targets_test.go` 核对 `{identifier}` 是它的列的工作项、出自项目的表。

**Tests:**
- HTTP（`apitest.Main` 两个方向）：`TestGetIssueByIdentifier`（调用者、路径的工作区（acme、beta）和标识原样交给用例，转义的土耳其字母也是）、`TestGetIssueByIdentifierRefusals`。
- 组合：`TestAWorkItemIsFoundByItsIdentifier`（acme 里 Web、IŞ、İŞ、İI、Iİ、ŞI、已删除的 Şİ、Sec（私有）；Web 的 1、已删除的 2、Done 里已归档的 3；walt 是 Web 的"只看自己"的访客；另一个工作区 beta 里也有 WEB 和 İŞ，各有它的 1：只在调用的工作区的未删除的项目里找，任何大小写、土耳其字母，IŞ 和 İŞ、İI 和 Iİ 各取逐字等于大写的那个；已删除的项目的标识与活着的折叠相同、又正是所问的大写（Şİ）时不指名任何项目，找到的是 ŞI 的；工作区里唯一带点的项目（beta 的 İŞ）用不带点的标识（`iş`）找到：比较的是折叠，不是大写；不是标识、不是 acme 的项目、与 İI 和 Iİ 折叠相同而大写不是其中任何一个的（`ii-1`）、没有这个未删除的编号、看不到的项目、访客规则隐藏的：同一个 `issue.not_found`，回答逐字节相同；wanda 403；olga 和不存在的 slug：`workspace.not_found`，逐字节相同）、`TestPermissionMatrix` 的 `getIssueByIdentifier` 两行、`TestMatrixViolationsCatchesAnotherCasesIdentifier`、`TestThePermissionMatrixCoversEveryOperation`；契约的整程序测试（`TestAPIRoutesAreTheContractsOperations`、`TestOperationsThatNeedATokenAnswer401WithoutOne`、`TestParametersThatDoNotBindAnswer400`）。

- [ ] **Step 1: 契约，生成**

`api/modules/issue.yaml`（修改，2 处）：

````old api/modules/issue.yaml
        default:
          $ref: '#/components/responses/Problem'
components:
````
````new api/modules/issue.yaml
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/workspaces/{slug}/issues/{identifier}:
    parameters:
      - $ref: '#/components/parameters/Slug'
      - $ref: '#/components/parameters/IssueIdentifier'
    get:
      operationId: getIssueByIdentifier
      tags: [issue]
      summary: Read a work item by its identifier
      description: >-
        For the workspace's active members, then as getIssue. The
        identifier is the project's identifier, a hyphen and the work
        item's number: WEB-12. The number is written in decimal with the
        digits 0 to 9, without a sign or a leading zero, from 1 to
        2147483647. The project's identifier is compared with those of the
        workspace's undeleted projects in any case, i, ı, I and İ as one
        letter; when more than one project's identifier matches it, the one
        that is it upper-cased letter for letter (i and ı as I, İ as İ) is
        taken, and when none is, none. A workspace that does not exist, is
        deleted, or of which the caller is not an active member answers
        workspace.not_found. An identifier that is not one, that names no
        project of the workspace, or a number of no undeleted work item of
        the project answers issue.not_found, as does a work item whose
        project the caller does not see, or that he did not create as a
        guest who sees only his own (getIssue): the same answer for each.
        One whose project he sees but may not read answers forbidden. An
        archived work item is read as any other.
      security: [{bearer: []}]
      x-problem-codes: [workspace.not_found, issue.not_found, forbidden]
      responses:
        '200':
          description: The work item.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IssueDetail'
        default:
          $ref: '#/components/responses/Problem'
components:
````
````old api/modules/issue.yaml
      scheme: bearer
  parameters:
````
````new api/modules/issue.yaml
      scheme: bearer
  parameters:
    Slug:
      name: slug
      in: path
      required: true
      description: A workspace's slug, as in the web app's address.
      schema:
        type: string
    IssueIdentifier:
      name: identifier
      in: path
      required: true
      description: >-
        A work item's identifier: its project's identifier, in any case, a
        hyphen and its number (IssueDetail.sequence_id), as WEB-12.
      schema:
        type: string
````

`api/openapi.yaml`（修改，1 处）：

````old api/openapi.yaml
    $ref: 'modules/issue.yaml#/paths/~1api~1v0~1issues~1{issue_id}'
````
````new api/openapi.yaml
    $ref: 'modules/issue.yaml#/paths/~1api~1v0~1issues~1{issue_id}'
  /api/v0/workspaces/{slug}/issues/{identifier}:
    $ref: 'modules/issue.yaml#/paths/~1api~1v0~1workspaces~1{slug}~1issues~1{identifier}'
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `d7ef3af4fcb935e9eb6ddad4d91fcf43c017c2589bbbc7502321ada68389ae7d` | 3248 | `api/dist/openapi.yaml` |
| `1419a422920c5dab18fda8c96b16e81502240bc91088ba0bc8ce73e458230f02` | 883 | `server/internal/modules/issue/adapter/http/gen/server.gen.go` |
| `eaba4cc339519bc395eccf3188ffe65e36260741f9fef024115f55b3bb4a7b04` | 3572 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/issue/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: HTTP 和接线**

`server/internal/modules/issue/adapter/http/by_identifier_test.go`（新文件，57 行）：

````file server/internal/modules/issue/adapter/http/by_identifier_test.go
package httpadapter_test

import (
	"net/http"
	"slices"
	"testing"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// GET goes to the use case for the caller, the path's workspace, and the
// identifier as the path gives it, in its case, a Turkish letter sent
// escaped as it came; the answer is 200 with the work item the use case
// answers, as getIssue shows it.
func TestGetIssueByIdentifier(t *testing.T) {
	for _, tt := range []struct{ path, slug, identifier string }{
		{"/api/v0/workspaces/acme/issues/WEB-7", "acme", "WEB-7"},
		{"/api/v0/workspaces/beta/issues/web-7", "beta", "web-7"},
		{"/api/v0/workspaces/acme/issues/i%C5%9F-7", "acme", "iş-7"},
	} {
		get := &fakeGetByIdentifier{answer: webSeven}
		res, body := do(t, newServer(t, httpadapter.UseCases{GetIssueByIdentifier: get}), request(http.MethodGet, tt.path, "alice", ""))
		if res.StatusCode != http.StatusOK || body != webSevenJSON+"\n" {
			t.Errorf("GET %s = %d %s, want 200 %s", tt.path, res.StatusCode, body, webSevenJSON)
		}
		if want := []string{aliceID.String() + " " + tt.slug + " " + tt.identifier}; !slices.Equal(get.calls, want) {
			t.Errorf("GET %s: calls %q, want %q", tt.path, get.calls, want)
		}
	}
}

// The use case's refusals, as the contract declares them, and its failure.
func TestGetIssueByIdentifierRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"no workspace", domain.ErrWorkspaceNotFound, http.StatusNotFound,
			`{"status":404,"code":"workspace.not_found","title":"Not Found","detail":"The workspace does not exist, or you are not a member of it."}`},
		{"no work item", domain.ErrIssueNotFound, http.StatusNotFound,
			`{"status":404,"code":"issue.not_found","title":"Not Found","detail":"The work item does not exist, or you cannot see it."}`},
		{"a workspace admin who is no member of the project", shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"a failure", errGone, http.StatusInternalServerError,
			`{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
	} {
		h := newServer(t, httpadapter.UseCases{GetIssueByIdentifier: &fakeGetByIdentifier{err: tt.err}})
		res, body := do(t, h, request(http.MethodGet, "/api/v0/workspaces/acme/issues/WEB-7", "alice", ""))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: GET = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
````

`server/internal/modules/issue/adapter/http/handler_test.go`（修改，1 处）：

````old server/internal/modules/issue/adapter/http/handler_test.go
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String())
	return f.answer, f.err
````
````new server/internal/modules/issue/adapter/http/handler_test.go
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String())
	return f.answer, f.err
}

// fakeGetByIdentifier is getIssueByIdentifier: each call is recorded as
// "caller slug identifier"; it answers answer, or err.
type fakeGetByIdentifier struct {
	calls  []string
	answer domain.Issue
	err    error
}

func (f *fakeGetByIdentifier) Execute(ctx context.Context, slug, identifier string) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+slug+" "+identifier)
	return f.answer, f.err
````

`server/internal/modules/issue/adapter/http/issues.go`（修改，1 处）：

````old server/internal/modules/issue/adapter/http/issues.go
	return gen.GetIssue200JSONResponse(issueDetail(i)), nil
````
````new server/internal/modules/issue/adapter/http/issues.go
	return gen.GetIssue200JSONResponse(issueDetail(i)), nil
}

// GetIssueByIdentifier serves GET
// /api/v0/workspaces/{slug}/issues/{identifier}: the identifier goes to
// the use case as it came, which reads it.
func (h handler) GetIssueByIdentifier(ctx context.Context, req gen.GetIssueByIdentifierRequestObject) (gen.GetIssueByIdentifierResponseObject,
	error) {
	i, err := h.uc.GetIssueByIdentifier.Execute(ctx, req.Slug, req.Identifier)
	if err != nil {
		return nil, err
	}
	return gen.GetIssueByIdentifier200JSONResponse(issueDetail(i)), nil
````

`server/internal/modules/issue/module.go`（修改，1 处）：

````old server/internal/modules/issue/module.go
// work items' tables and brings creating, reading, changing and deleting a
// work item. For the project module's deletions (its IssueCascade port) it
// soft-deletes a deleted project's rows of them, tells whether work items
// are in a state, and takes deleted labels off them.
````
````new server/internal/modules/issue/module.go
// work items' tables and brings creating, reading, by id or by identifier,
// changing and deleting a work item. For the project module's deletions
// (its IssueCascade port) it soft-deletes a deleted project's rows of
// them, tells whether work items are in a state, and takes deleted labels
// off them.
````

- [ ] **Step 3: 组合和矩阵**

`server/internal/bootstrap/issue_identifier_test.go`（新文件，112 行）：

````file server/internal/bootstrap/issue_identifier_test.go
package bootstrap

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"uuid"
)

// getIssueByIdentifier as bootstrap wires it (M4 design 3.7). In acme of
// the world, beside Web, alice makes the projects IŞ, as she gives "iş",
// which createProject upper-cases letter for letter, İŞ, İI, Iİ, ŞI, Şİ,
// deleted since, and Sec, which is private; Web's work items 1, 2, deleted
// since, and 3, archived in Done (in the database); IŞ's, İŞ's, İI's,
// Iİ's, ŞI's and Sec's 1; and walt Web's guest, whom Web shows only his
// own. Then she makes the workspace beta, and in it a project WEB too,
// after acme's, and İŞ, each with its work item 1. A work item is found by
// its identifier among its workspace's projects alone, in any case,
// Turkish letters too: of IŞ and İŞ, which fold alike, and of İI and Iİ,
// the one that is the identifier asked upper-cased letter for letter; an
// archived one as any other. A project deleted since whose identifier
// folds as a live one's and is the identifier asked upper-cased, Şİ, names
// nothing: ŞI's work item is found; a workspace's one dotted project,
// beta's İŞ, is found by a dotless identifier, iş. An identifier that is
// not one, one that names no project of acme, one that folds as İI and Iİ
// do and is neither of them upper-cased, a number of no undeleted work
// item, a project its caller does not see, and another's work item to a
// guest who sees only his own are one answer, issue.not_found, body for
// body. wanda, who sees Web and is none of its members, may not read it;
// olga, no member of acme, finds no workspace, and nor does a slug of
// none, body for body.
func TestAWorkItemIsFoundByItsIdentifier(t *testing.T) {
	w := newIssueWorld(t)
	is := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Is", "iş")
	dotted := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Dotted", "İŞ")
	dotFirst := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Dot first", "İI")
	dotLast := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Dot last", "Iİ")
	shi := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Shi", "ŞI")
	gone := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Shi dotted", "Şİ")
	w.send(t, "alice", http.MethodDelete, "/api/v0/projects/"+gone.String(), "", http.StatusNoContent)
	var created struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/workspaces/acme/projects", `{"name":"Sec","identifier":"SEC","network":0}`,
		http.StatusCreated), &created)
	web1, web2, web3 := w.newWorkItem(t), w.newWorkItem(t), w.newWorkItem(t)
	isOne, dottedOne := w.newWorkItemIn(t, is, `{"name":"Fix"}`), w.newWorkItemIn(t, dotted, `{"name":"Fix"}`)
	secOne := w.newWorkItemIn(t, created.ID, `{"name":"Fix"}`)
	dotFirstOne, dotLastOne := w.newWorkItemIn(t, dotFirst, `{"name":"Fix"}`), w.newWorkItemIn(t, dotLast, `{"name":"Fix"}`)
	shiOne := w.newWorkItemIn(t, shi, `{"name":"Fix"}`)
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+web2.String(), "", http.StatusNoContent)
	if _, err := w.pool.Exec(context.Background(), "UPDATE issues SET state_id = $2, completed_at = updated_at, archived_at = updated_at WHERE id = $1",
		web3, w.done); err != nil {
		t.Fatal(err)
	}
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/members",
		`{"members":[{"member_id":"`+w.ids["walt"].String()+`","role":5}]}`, http.StatusCreated)
	w.send(t, "alice", http.MethodPost, "/api/v0/workspaces", `{"name":"Beta","slug":"beta"}`, http.StatusCreated)
	betaOne := w.newWorkItemIn(t, createdProject(t, w.contract, w.base, w.tokens["alice"], "beta", "Web", "WEB"), `{"name":"Fix"}`)
	betaDotted := w.newWorkItemIn(t, createdProject(t, w.contract, w.base, w.tokens["alice"], "beta", "Dotted", "İŞ"), `{"name":"Fix"}`)
	read := func(name, slug, identifier string) (int, string) {
		return call(t, w.contract, http.MethodGet, w.base+"/api/v0/workspaces/"+slug+"/issues/"+url.PathEscape(identifier), w.tokens[name], "")
	}

	for _, tt := range []struct {
		slug, identifier string
		want             uuid.UUID
		archived         bool
	}{{"acme", "WEB-1", web1, false}, {"acme", "web-1", web1, false}, {"acme", "wEb-1", web1, false}, {"acme", "WEB-3", web3, true},
		{"acme", "iş-1", isOne, false}, {"acme", "ış-1", isOne, false}, {"acme", "IŞ-1", isOne, false}, {"acme", "İş-1", dottedOne, false},
		{"acme", "İŞ-1", dottedOne, false}, {"acme", "İi-1", dotFirstOne, false}, {"acme", "iİ-1", dotLastOne, false},
		{"acme", "sec-1", secOne, false}, {"beta", "web-1", betaOne, false}, {"acme", "Şİ-1", shiOne, false},
		{"beta", "iş-1", betaDotted, false}} {
		status, body := read("alice", tt.slug, tt.identifier)
		var got struct {
			ID         uuid.UUID `json:"id"`
			ArchivedAt *string   `json:"archived_at"`
		}
		if status == http.StatusOK {
			decodeAnswer(t, body, &got)
		}
		if status != http.StatusOK || got.ID != tt.want || (got.ArchivedAt != nil) != tt.archived {
			t.Errorf("%s in %s = %d %s; want 200, the work item %s, archived %v", tt.identifier, tt.slug, status, body, tt.want, tt.archived)
		}
	}

	refusals := map[string]string{}
	for _, tt := range []struct{ name, slug, identifier, code string }{
		{"alice", "acme", "WEB1", "issue.not_found"}, {"alice", "acme", "WEB-0", "issue.not_found"},
		{"alice", "acme", "WEB-01", "issue.not_found"}, {"alice", "acme", "API-1", "issue.not_found"},
		{"alice", "acme", "IS-1", "issue.not_found"}, {"alice", "acme", "ii-1", "issue.not_found"},
		{"alice", "acme", "WEB-4", "issue.not_found"}, {"alice", "acme", "WEB-2", "issue.not_found"},
		{"walt", "acme", "SEC-1", "issue.not_found"}, {"walt", "acme", "WEB-1", "issue.not_found"},
		{"wanda", "acme", "WEB-1", "forbidden"}, {"olga", "acme", "WEB-1", "workspace.not_found"},
		{"alice", "acne", "WEB-1", "workspace.not_found"},
	} {
		status, body := read(tt.name, tt.slug, tt.identifier)
		if refusals[tt.code] == "" {
			refusals[tt.code] = body
		}
		want := http.StatusNotFound
		if tt.code == "forbidden" {
			want = http.StatusForbidden
		}
		if status != want || !strings.Contains(body, `"code":"`+tt.code+`"`) || body != refusals[tt.code] {
			t.Errorf("%s as %s in %s = %d %s; want %d %s, as each such refusal: %s", tt.identifier, tt.name, tt.slug, status, body, want, tt.code,
				refusals[tt.code])
		}
	}
}
````

`server/internal/bootstrap/permission_matrix_columns_test.go`（修改，4 处）：

````old server/internal/bootstrap/permission_matrix_columns_test.go
}

// Each check of the project level's columns and of the not-target
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
}

// noProjectTable is what matrixViolations reports of a row of op whose
// cells name param, a project or a row under one, from columns of no
// project table (projectTables): one violation for each, in the order
// given.
func noProjectTable(op, param string, columns ...caller) []string {
	var want []string
	for _, c := range columns {
		want = append(want, fmt.Sprintf("row %s, %s: %s from a column of no project table (projectTables): "+
			"a project's row names its columns", op, c, param))
	}
	return want
}

// Each check of the project level's columns and of the not-target
````
````old server/internal/bootstrap/permission_matrix_columns_test.go
		}), checks}, func() []string {
			var want []string
			for _, c := range []caller{callerAdmin, callerMember, callerGuest} {
				want = append(want, fmt.Sprintf("row getProject, %s: {project_id} from a column of no project table (projectTables): "+
					"a project's row names its columns", c))
			}
			return want
		}()},
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
		}), checks}, noProjectTable("getProject", "{project_id}", callerAdmin, callerMember, callerGuest)},
````
````old server/internal/bootstrap/permission_matrix_columns_test.go
		}), checks}, []string{fmt.Sprintf("row getProject, %s: {project_id} from a column of no project table (projectTables): "+
			"a project's row names its columns", callerSoleAdmin)}},
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
		}), checks}, noProjectTable("getProject", "{project_id}", callerSoleAdmin)},
````
````old server/internal/bootstrap/permission_matrix_columns_test.go
		var noTable []string
		for _, c := range []caller{callerAdmin, callerMember, callerGuest} {
			noTable = append(noTable, fmt.Sprintf("row %s, %s: %s from a column of no project table (projectTables): "+
				"a project's row names its columns", under.op.ID, c, param))
		}
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
		noTable := noProjectTable(under.op.ID, param, callerAdmin, callerMember, callerGuest)
````

`server/internal/bootstrap/permission_matrix_issue_columns_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_issue_columns_test.go
		t.Errorf("the guest's own column at acme's admin's work item: %q, want %q", got, want)
	}
}

````
````new server/internal/bootstrap/permission_matrix_issue_columns_test.go
		t.Errorf("the guest's own column at acme's admin's work item: %q, want %q", got, want)
	}
}

// A cell that names a work item by its identifier names its column's own
// (identifierOf), from a column of a project table: the guest's own column
// asking for WEB-1, acme's admin's, would ask the guest rule's other case
// under its name; a workspace-level row's admin, member and guest would
// ask no project's column (X are the project level's too).
func TestMatrixViolationsCatchesAnotherCasesIdentifier(t *testing.T) {
	s := newSeeded().in(t)
	byID := apitest.Operation{ID: "getIssueByIdentifier", Tags: []string{"issue"}, Method: http.MethodGet,
		Path: "/api/v0/workspaces/{slug}/issues/{identifier}"}
	row := matrixRow{op: byID.ID, columns: issueColumns, request: byIdentifier,
		cells: inItsWorkspace(onAWorkItem(cellOK, cellOK, cellOK, cellOK))}
	ops := []apitest.Operation{byID}
	if got := matrixViolations(ops, matrixExemptions{}, []matrixRow{row}, s, nil); len(got) != 0 {
		t.Fatalf("a matching matrix: %q, want none", got)
	}
	another := row
	another.request = func(c caller, s seeded) (string, string, string) {
		if c == callerGuestOwn {
			return byIdentifier(callerGuestOthers, s)
		}
		return byIdentifier(c, s)
	}
	want := []string{fmt.Sprintf("row getIssueByIdentifier, %s: {identifier} WEB-1 is not its column's work item Guest's, WEB-3",
		callerGuestOwn)}
	if got := matrixViolations(ops, matrixExemptions{}, []matrixRow{another}, s, nil); !slices.Equal(got, want) {
		t.Errorf("the guest's own column at WEB-1: %q, want %q", got, want)
	}
	inWorkspaceRow := matrixRow{op: byID.ID, request: func(c caller, _ seeded) (string, string, string) {
		return http.MethodGet, "/api/v0/workspaces/" + workspaceOf(c) + "/issues/WEB-1", ""
	}, cells: inWorkspace(cellOK, cellOK, cellOK)}
	refused := noProjectTable(byID.ID, "{identifier}", callerAdmin, callerMember, callerGuest)
	if got := matrixViolations(ops, matrixExemptions{}, []matrixRow{inWorkspaceRow}, s, nil); !slices.Equal(got, refused) {
		t.Errorf("a workspace-level row naming a work item: %q, want %q", got, refused)
	}
}

````

`server/internal/bootstrap/permission_matrix_issues_test.go`（修改，4 处）：

````old server/internal/bootstrap/permission_matrix_issues_test.go
import (
````
````new server/internal/bootstrap/permission_matrix_issues_test.go
import (
	"maps"
````
````old server/internal/bootstrap/permission_matrix_issues_test.go
	"slices"
````
````new server/internal/bootstrap/permission_matrix_issues_test.go
	"slices"
	"strconv"
````
````old server/internal/bootstrap/permission_matrix_issues_test.go
// naming it by its id (/issues/{issue_id}): its column's work item
// (issueOf). Each creating cell writes on a copy of its own, so each
// creation is its project's next work item after the seeded ones
// (nextNumber); the reading cells share one. A row of 9.2's table has the
// issue table's columns (issueColumns); a variant that asks what is
````
````new server/internal/bootstrap/permission_matrix_issues_test.go
// naming it by its id (/issues/{issue_id}), or by its identifier in its
// column's workspace (/workspaces/{slug}/issues/{identifier}): its
// column's work item (issueOf). Each creating cell writes on a copy of its
// own, so each creation is its project's next work item after the seeded
// ones (nextNumber); the reading cells share one. A row of 9.2's table has
// the issue table's columns (issueColumns); a variant that asks what is
````
````old server/internal/bootstrap/permission_matrix_issues_test.go
			cells: issueRows.ofArchived(cellOK, cellForbidden), check: readsTheIssue},
	}
````
````new server/internal/bootstrap/permission_matrix_issues_test.go
			cells: issueRows.ofArchived(cellOK, cellForbidden), check: readsTheIssue},
		// Every active member of the workspace finds it, then as getIssue;
		// X, who is none, finds no workspace (M4 design 3.4, 3.7).
		{op: "getIssueByIdentifier", columns: issueColumns, request: byIdentifier,
			cells: inItsWorkspace(onAWorkItem(cellOK, cellOK, cellOK, cellOK)), check: readsTheIssue},
		// An archived project's work items are found as any other's.
		{op: "getIssueByIdentifier", variant: "archived", columns: archivedColumns, request: byIdentifier,
			cells: inItsWorkspace(issueRows.ofArchived(cellOK, cellForbidden)), check: readsTheIssue},
	}
}

// identifierOf is the identifier of the column's work item (issueOf): its
// project's identifier, a hyphen and its number, as WEB-2.
func identifierOf(c caller) string {
	n, _ := numberOf(projectOf(c), issueOf(c))
	for _, p := range matrixProjects {
		if p.key == projectOf(c) {
			return p.identifier + "-" + strconv.Itoa(n)
		}
	}
	return ""
}

// byIdentifier is the request of a row whose callers each read their
// column's work item by its identifier, in their column's workspace.
func byIdentifier(c caller, _ seeded) (string, string, string) {
	return http.MethodGet, "/api/v0/workspaces/" + workspaceOf(c) + "/issues/" + identifierOf(c), ""
}

// inItsWorkspace are cells of a row that names its column's workspace by
// its slug: workspace.not_found for X, and for the archived project's
// column of an account never a member of acme, whom it does not let in;
// every other cell as cells has it.
func inItsWorkspace(cells map[caller]cell) map[caller]cell {
	out := maps.Clone(cells)
	for _, c := range []caller{callerNever, callerRemoved, callerDeleted, callerArchivedNever} {
		if _, ok := out[c]; ok {
			out[c] = cellWorkspaceNotFound
		}
	}
	return out
````

`server/internal/bootstrap/permission_matrix_targets_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_targets_test.go
// (issueOf), each from a column of a project table (projectTables): a
// project's operation in a workspace-level row, or the only admin's, would
// leave the project level's own columns unasked.
````
````new server/internal/bootstrap/permission_matrix_targets_test.go
// (issueOf), as must a work item named by its identifier ({identifier}
// right after issues: identifierOf), each from a column of a project table
// (projectTables): a project's operation in a workspace-level row, or the
// only admin's, would leave the project level's own columns unasked.
````
````old server/internal/bootstrap/permission_matrix_targets_test.go
				}
			}
````
````new server/internal/bootstrap/permission_matrix_targets_test.go
				}
			}
		case segment == "{identifier}" && i > 0 && want[i-1] == "issues":
			if !ofAProjectTable(c) {
				return "{identifier} from a column of no project table (projectTables): a project's row names its columns"
			}
			if got[i] != identifierOf(c) {
				return fmt.Sprintf("{identifier} %s is not its column's work item %s, %s", got[i], issueOf(c), identifierOf(c))
			}
````

- [ ] **Step 4: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestAWorkItemIsFoundByItsIdentifier$|TestPermissionMatrix|TestMatrixViolations|TestThePermissionMatrixCoversEveryOperation$|TestAPIRoutesAreTheContractsOperations$|TestOperationsThatNeedATokenAnswer401WithoutOne$|TestParametersThatDoNotBindAnswer400$' ./internal/bootstrap/`
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
git add api/modules/issue.yaml api/openapi.yaml server/internal/bootstrap/issue_identifier_test.go server/internal/bootstrap/permission_matrix_columns_test.go server/internal/bootstrap/permission_matrix_issue_columns_test.go server/internal/bootstrap/permission_matrix_issues_test.go server/internal/bootstrap/permission_matrix_targets_test.go server/internal/modules/issue/adapter/http/by_identifier_test.go server/internal/modules/issue/adapter/http/handler_test.go server/internal/modules/issue/adapter/http/issues.go server/internal/modules/issue/module.go api/dist/openapi.yaml server/internal/modules/issue/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M4/P2): GET a work item by its identifier in a workspace

GET /api/v0/workspaces/{slug}/issues/{identifier} serves
getIssueByIdentifier: workspace.not_found for a workspace the caller is
no active member of, then one issue.not_found, body for body, for an
identifier that names no undeleted work item of the workspace's
projects, a project he does not see, or another's work item to a guest
who sees only his own. The matrix reads each column's own work item by
its identifier in its column's workspace, and the composition test finds
work items in any case, Turkish letters too, in their own workspace
alone and never in a project deleted since.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p2.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试，或没有替身的平台、领域、转换的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是生成器（`make gen-go`）和中英文翻译键（`make lint-web`）的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过，端到端的检查只在标出的 Task 的树上）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T6.1` | 契约不声明 `getIssueByIdentifier` 答 `workspace.not_found` | `internal/modules/issue/adapter/http` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsFoundByItsIdentifier`（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `getIssueByIdentifier` 各格（`make test`） | 单元；组合 |
| `T6.2` | 矩阵的一格可以按任何标识指名工作项，不必是它的列自己的 | `internal/bootstrap` 的 `TestMatrixViolationsCatchesAnotherCasesIdentifier`（`make test`） | 单元 |
| `T6.3` | 处理函数不论路径的工作区是哪个，都把 `acme` 交给用例 | `internal/modules/issue/adapter/http` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsFoundByItsIdentifier`（`make test`） | 单元；组合 |

本 Task 的检查也发现较早的 Task 的变异 `T2.12`、`T5.1`、`T5.2`、`T5.3`、`T5.4`、`T5.5`、`T5.6`、`T5.7`、`T5.8`、`T5.9`、`T5.11`、`T5.14`、`T5.15`、`T5.16`、`T5.18`（列在各自的 Task 的表里，标"（Task 6 起）"）：本 Task 做完之后，在本 Task 的树上重跑它们（`mutants_p2.py`，`MUT_LATER=6`）。

---

### Task 7: 归档、恢复

**Files:**
- Create: `server/internal/bootstrap/issue_archive_test.go`、`server/internal/bootstrap/permission_matrix_issue_archive_test.go`、`server/internal/modules/issue/adapter/http/archive_test.go`、`server/internal/modules/issue/app/archive_issue.go`、`server/internal/modules/issue/app/archive_issue_test.go`
- Modify: `api/modules/issue.yaml`、`api/openapi.yaml`、`server/internal/bootstrap/gate_test.go`、`server/internal/bootstrap/issue_updates_test.go`、`server/internal/bootstrap/issue_world_test.go`、`server/internal/bootstrap/issue_write_locks_test.go`、`server/internal/bootstrap/issue_writes_test.go`、`server/internal/bootstrap/permission_matrix_coverage_test.go`、`server/internal/bootstrap/permission_matrix_issue_updates_test.go`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/modules/issue/adapter/http/handler.go`、`server/internal/modules/issue/adapter/http/issues.go`、`server/internal/modules/issue/adapter/postgres/queries/update.sql`、`server/internal/modules/issue/adapter/postgres/update.go`、`server/internal/modules/issue/adapter/postgres/update_test.go`、`server/internal/modules/issue/app/fakes_test.go`、`server/internal/modules/issue/app/lock_test.go`、`server/internal/modules/issue/app/ports.go`、`server/internal/modules/issue/app/rows.go`、`server/internal/modules/issue/app/update_issue.go`、`server/internal/modules/issue/domain/activity.go`、`server/internal/modules/issue/domain/activity_test.go`、`server/internal/modules/issue/domain/errors.go`、`server/internal/modules/issue/domain/issue.go`、`server/internal/modules/issue/domain/issue_test.go`、`server/internal/modules/issue/module.go`、`web/apps/web/core/lib/error-messages.ts`、`web/packages/i18n/src/locales/en/errors.json`、`web/packages/i18n/src/locales/zh-CN/errors.json`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/issue/adapter/http/gen/server.gen.go`、`server/internal/modules/issue/adapter/postgres/gen/update.sql.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**（spec 2.8、第 3 节第 5、8、9、13、15 条；M4 设计 3.5、3.8、3.13、5.1、9.2）
- 契约：`POST /api/v0/issues/{issue_id}/archive`（`archiveIssue`，200 `IssueDetail`，`x-problem-codes: [issue.not_found, forbidden, project.archived, issue.not_archivable, server_busy]`）、`POST …/unarchive`（`unarchiveIssue`，同上，没有 `issue.not_archivable`），描述写明谁能做、访客规则、锁下的次序、已是所求的样子时不写、只归档已完成或已取消的；`api/openapi.yaml` 的两条路径。
- `issue/domain`：`GroupCancelled`、`Archivable(group) bool`（已完成、已取消）；`ErrNotArchivable`（409 `issue.not_archivable`）；`ArchiveActivity()`（第 14 行：`archived_at`、`updated`，`NULL` → `manual_archive`）、`UnarchiveActivity()`（第 16 行：`archive` → `restore`），`comment` 是"archived the work item""restored the work item"（第 3 节第 8 条）。
- `issue/app`：`ports.go` 的 `IssueArchiver`（嵌入 `IssueReader`；`SetArchivedAt`、`InsertActivities`）；`rows.go` 的 `IssueArchival{ID, ArchivedAt *time.Time, By, At}`；`archive_issue.go`（新）：`ArchiveIssueDeps{Locks, Issues, States, Clock}`，`ArchiveIssue`、`UnarchiveIssue`（`NewArchiveIssue`、`NewUnarchiveIssue`，`Execute(ctx, id) (domain.Issue, error)`），共用 `archive(ctx, id, on)`：`Locks.run` 的 `write{issue: &id, action: issue.archive}` → 锁下项目已归档 409 → 已是所求的样子：回答锁下读到的工作项，不写 → 归档时 `archivable`（锁下读项目的状态，由 `issueState` 取出工作项的状态；不是项目的是错误）→ 时刻 → `SetArchivedAt` → 一行动态 → 在事务里读回（`readBack`）。`update_issue.go` 的 `stateOf` 改成 `issueState(states, i) (StateRef, error)`（带上那个错误），修改的 `check` 和归档共用。
- 存储：`queries/update.sql` 的 `SetArchivedAt`（按 `id` 写 `archived_at`、`updated_at`、`updated_by_id`）；`update.go` 的 `SetArchivedAt`。
- `issue/adapter/http`：`handler.go` 的 `ArchiveIssueUseCase`，`UseCases.ArchiveIssue`、`UnarchiveIssue`（两个都是它）；`issues.go` 的 `ArchiveIssue`、`UnarchiveIssue`。`issue/module.go` 接上两个用例。
- 前端：`error-messages.ts` 的 `issue.not_archivable`，两份 `errors.json` 的 `issue_not_archivable`（措辞第 3 节第 9 条）。
- 测试的帮手：`fakes_test.go` 的 `fakeIssues` 记下最后一次 `SetArchivedAt`（`archival`），读出的工作项带上它（`archivedAt`、最后修改者和时刻）；`issue_world_test.go` 的 `doneWorkItem`；`issue_updates_test.go` 的说明（`archiveIssue` 改最后修改者）；`lock_test.go` 的 `TestEachWriteOnAWorkItemEntersItsProjectsGate` 加归档、恢复两行（在 `newArchive` 的世界上）；`permission_matrix_issue_updates_test.go` 的 `answered` 的说明（调用者是最后修改者，除非 `change` 另说）。
- 矩阵：`matrixRow.prepare func(t, c, s, url string)`：在写的格子的副本上、它的 app 启动之前运行；没有 `write` 的行带 `prepare` 是违规（`permission_matrix_coverage_test.go`）；`permission_matrix_issue_archive_test.go`（新）的九行：`archiveIssue`（照常、Todo 里的、已归档的、已归档的项目、已归档的项目里已归档的）、`unarchiveIssue`（照常、没归档的、已归档的项目、已归档的项目里已归档的），`prepare` 把格子的列的工作项移到 Done（和归档）；写的行的检查 `asArchived` 建在 `answered` 上（Done 里、完成了，没写的行最后修改者是建它的人）。
- 取锁形状：`archiveIssue` 一行（锁下读状态：等 `states`，`ACCESS EXCLUSIVE`）、`unarchiveIssue` 一行（等 `issues`，`SHARE`）。`issue_writes_test.go` 的连接测试在一个连接的池上归档、恢复第一个工作项。`gate_test.go` 的 `TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy` 加归档、恢复两行。

**Tests:**
- 领域：`TestArchivable`、`TestArchiveActivity`。
- 单元：`TestArchiveIssueArchivesTheWorkItem`（3.5、3.8 的次序：`Locks` 的调用、锁下读状态、时刻、alice 在那一刻归档、第 14 行、读回；Cancelled 里的同 Done）、`TestUnarchiveIssueUnarchivesTheWorkItem`（不看状态：Backlog 里已归档的也恢复；第 16 行）、`TestArchivingAsItIsWritesNothing`（不读时刻、不写，回答锁下读到的）、`TestArchiveIssueRefusals`（锁下已归档的项目，已归档的工作项也是，409 `project.archived`；Backlog 里的 409 `issue.not_archivable`；角色在预检；读写的失败照原样；没有调用者）、`TestArchivingAWorkItemWhoseStateIsNotItsProjectsFails`（锁下读到的状态里没有工作项的状态：是错误，不是 409 `issue.not_archivable`，读完就不再写）；`TestEachWriteOnAWorkItemEntersItsProjectsGate` 的归档、恢复两行（预检之后、任何事务之外进项目的写闸门，事务结束之后最后离开）。
- 存储：`TestArchivingAWorkItemWritesItsRow`（归档、再恢复：只改这三列，别的列、别的工作项不动）、`TestAFailedUpdateStatementIsAnError`。
- HTTP（`apitest.Main` 两个方向）：`TestArchiveAndUnarchiveIssue`、`TestArchiveIssueRefusals`（`issue.not_archivable` 只是 `archiveIssue` 的）。
- 组合：`TestArchivingAWorkItemStampsItsRows`（alice 在 Done 里建，bob 归档、再归档，alice 恢复、再恢复：第一次的各是做的人在请求之内的一个时刻，工作项和它的一行动态同一时刻；第二次什么都不写；一开始就建好的 Todo 里的工作项 409 `issue.not_archivable`，它自己被拒绝的归档和别的工作项的归档、恢复都不写它；移到 Cancelled 之后 bob 归档它；Cancelled 改到"进行中"组之后（另一个已取消的状态 Dropped 留住那一组），bob 再归档它照样回答、什么都不写：已归档的工作项先于状态组回答；alice 恢复它：恢复不看状态组）、`TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy` 的归档、恢复两行（项目的一个位置被占、不排队时立刻答 503 `server_busy`、`Retry-After: 1`）、`TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`（归档、恢复的每条语句在它的事务的连接上）、`TestEachIssueWriteLocksItsProjectFirst` 的两行、`TestPermissionMatrix` 的九行、`TestMatrixViolationsCatchesEachGap`（没有写的行带 `prepare`）、`TestThePermissionMatrixCoversEveryOperation`、`TestAWorkItemReadsAsStored`。

- [ ] **Step 1: 契约和语句，生成**

`api/modules/issue.yaml`（修改，1 处）：

````old api/modules/issue.yaml
          description: The work item is deleted.
````
````new api/modules/issue.yaml
          description: The work item is deleted.
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/issues/{issue_id}/archive:
    parameters:
      - $ref: '#/components/parameters/IssueID'
    post:
      operationId: archiveIssue
      tags: [issue]
      summary: Archive a work item
      description: >-
        For the admins and members of its project. A work item that does not
        exist, is deleted, whose project the caller does not see, or that he
        did not create as a guest who sees only his own (getIssue) answers
        issue.not_found, whatever his role may do; one whose project he sees
        but whose work items he may not archive, forbidden; both are
        answered before the project is locked, and with the project locked
        the role is decided again. A work item of an archived project
        cannot be archived (project.archived). An archived work item is
        answered as it is, and nothing is written. Only a work item in a
        state of the completed or the cancelled group is archived
        (issue.not_archivable). It is archived at the moment of the
        request, and changed last by the caller then; its activity says it
        was archived, by the caller then. A project with too many writes on
        its work items at once answers server_busy, with Retry-After: try
        again then.
      security: [{bearer: []}]
      x-problem-codes: [issue.not_found, forbidden, project.archived, issue.not_archivable, server_busy]
      responses:
        '200':
          description: The work item, archived.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IssueDetail'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/issues/{issue_id}/unarchive:
    parameters:
      - $ref: '#/components/parameters/IssueID'
    post:
      operationId: unarchiveIssue
      tags: [issue]
      summary: Unarchive a work item
      description: >-
        For the admins and members of its project, as archiveIssue. A work
        item of an archived project cannot be unarchived
        (project.archived). A work item that is not archived is answered as
        it is, and nothing is written. Whatever its state, it is unarchived
        at the moment of the request, and changed last by the caller then;
        its activity says it was restored, by the caller then. A project
        with too many writes on its work items at once answers server_busy,
        with Retry-After: try again then.
      security: [{bearer: []}]
      x-problem-codes: [issue.not_found, forbidden, project.archived, server_busy]
      responses:
        '200':
          description: The work item, not archived.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IssueDetail'
````

`api/openapi.yaml`（修改，1 处）：

````old api/openapi.yaml
    $ref: 'modules/issue.yaml#/paths/~1api~1v0~1issues~1{issue_id}'
````
````new api/openapi.yaml
    $ref: 'modules/issue.yaml#/paths/~1api~1v0~1issues~1{issue_id}'
  /api/v0/issues/{issue_id}/archive:
    $ref: 'modules/issue.yaml#/paths/~1api~1v0~1issues~1{issue_id}~1archive'
  /api/v0/issues/{issue_id}/unarchive:
    $ref: 'modules/issue.yaml#/paths/~1api~1v0~1issues~1{issue_id}~1unarchive'
````

`server/internal/modules/issue/adapter/postgres/queries/update.sql`（修改，1 处）：

````old server/internal/modules/issue/adapter/postgres/queries/update.sql
    updated_by_id = sqlc.arg(by)::uuid, updated_at = sqlc.arg(at)::timestamptz
````
````new server/internal/modules/issue/adapter/postgres/queries/update.sql
    updated_by_id = sqlc.arg(by)::uuid, updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id);

-- name: SetArchivedAt :exec
-- archiveIssue's and unarchiveIssue's work item (M4 design 3.8), under the project's FOR NO KEY UPDATE, read again under
-- it: its archived_at, the moment or NULL, last changed at the moment and by the account given; every other column as
-- it is. It is not locked by id first, as UpdateIssue.
UPDATE issues
SET archived_at = sqlc.narg(archived_at), updated_by_id = sqlc.arg(by)::uuid, updated_at = sqlc.arg(at)::timestamptz
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `59ced5c7261164b92b84538c1aed1f82bf3e3bb2e7c58d1f45be04c60acfe0ad` | 3299 | `api/dist/openapi.yaml` |
| `a0c6e1fb65f25f8aebd7e9949b8b7599867a45569000a5123f943a2f626f11d7` | 1093 | `server/internal/modules/issue/adapter/http/gen/server.gen.go` |
| `d1e8d6caab269d042c2ecf9032533c765f04e8c0ae10f102550af7685a3a9b9d` | 124 | `server/internal/modules/issue/adapter/postgres/gen/update.sql.go` |
| `23d47e1db27950d2ebd10e57e88d1f13205b982c2f8a75d048c6f77effbbfdd8` | 3666 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/issue/adapter/http/gen/server.gen.go server/internal/modules/issue/adapter/postgres/gen/update.sql.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: 领域**

`server/internal/modules/issue/domain/activity.go`（修改，1 处）：

````old server/internal/modules/issue/domain/activity.go
}

func ptr[T any](v T) *T {
````
````new server/internal/modules/issue/domain/activity.go
}

// ArchiveActivity is the row a work item's archiving by a caller writes:
// row 14 of M4 design 3.13, archived_at, updated, from NULL to
// manual_archive.
func ArchiveActivity() Activity {
	return Activity{Field: ptr("archived_at"), Verb: VerbUpdated, NewValue: ptr("manual_archive"), Comment: "archived the work item"}
}

// UnarchiveActivity is the row a work item's unarchiving writes: row 16 of
// M4 design 3.13, archived_at, updated, from archive to restore, after a
// caller's archiving too (Plane the same).
func UnarchiveActivity() Activity {
	return Activity{Field: ptr("archived_at"), Verb: VerbUpdated, OldValue: ptr("archive"), NewValue: ptr("restore"),
		Comment: "restored the work item"}
}

func ptr[T any](v T) *T {
````

`server/internal/modules/issue/domain/activity_test.go`（修改，1 处）：

````old server/internal/modules/issue/domain/activity_test.go
			t.Errorf("%s: UpdateActivity() = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

````
````new server/internal/modules/issue/domain/activity_test.go
			t.Errorf("%s: UpdateActivity() = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

// Archiving by a caller writes row 14 of M4 design 3.13, and unarchiving
// row 16: Plane's values of archived_at, the old one NULL for an
// archiving.
func TestArchiveActivity(t *testing.T) {
	field, archived, archive, restore := "archived_at", "manual_archive", "archive", "restore"
	if got, want := ArchiveActivity(), (Activity{Field: &field, Verb: "updated", NewValue: &archived,
		Comment: "archived the work item"}); !reflect.DeepEqual(got, want) {
		t.Errorf("ArchiveActivity() = %+v, want %+v", got, want)
	}
	if got, want := UnarchiveActivity(), (Activity{Field: &field, Verb: "updated", OldValue: &archive, NewValue: &restore,
		Comment: "restored the work item"}); !reflect.DeepEqual(got, want) {
		t.Errorf("UnarchiveActivity() = %+v, want %+v", got, want)
	}
}

````

`server/internal/modules/issue/domain/errors.go`（修改，1 处）：

````old server/internal/modules/issue/domain/errors.go
	ErrIssueNotFound = shared.NewError(shared.KindNotFound, "issue.not_found", "The work item does not exist, or you cannot see it.")
````
````new server/internal/modules/issue/domain/errors.go
	ErrIssueNotFound = shared.NewError(shared.KindNotFound, "issue.not_found", "The work item does not exist, or you cannot see it.")
	// ErrNotArchivable answers the archiving of a work item in a state of
	// neither the completed nor the cancelled group (M4 design 3.8).
	ErrNotArchivable = shared.NewError(shared.KindConflict, "issue.not_archivable",
		"Only a completed or cancelled work item can be archived; move it to such a state first.")
````

`server/internal/modules/issue/domain/issue.go`（修改，2 处）：

````old server/internal/modules/issue/domain/issue.go
// group is completed (M4 design 3.7); the triage group's state takes no
// work item through the API.
````
````new server/internal/modules/issue/domain/issue.go
// group is completed (M4 design 3.7), and one in it or in the cancelled
// group may be archived (3.8); the triage group's state takes no work item
// through the API.
````
````old server/internal/modules/issue/domain/issue.go
	GroupCompleted StateGroup = "completed"
	GroupTriage    StateGroup = "triage"
)
````
````new server/internal/modules/issue/domain/issue.go
	GroupCompleted StateGroup = "completed"
	GroupCancelled StateGroup = "cancelled"
	GroupTriage    StateGroup = "triage"
)

// Archivable reports whether a work item in a state of group may be
// archived (M4 design 3.8): in the completed or the cancelled group, not
// in any other.
func Archivable(group StateGroup) bool {
	return group == GroupCompleted || group == GroupCancelled
}
````

`server/internal/modules/issue/domain/issue_test.go`（修改，1 处）：

````old server/internal/modules/issue/domain/issue_test.go
}

// A state asked for that is not the project's, or is its triage state, is
````
````new server/internal/modules/issue/domain/issue_test.go
}

// A work item is archived in the completed and the cancelled groups, and
// in no other (M4 design 3.8).
func TestArchivable(t *testing.T) {
	for group, want := range map[StateGroup]bool{"backlog": false, "unstarted": false, "started": false, GroupCompleted: true,
		GroupCancelled: true, GroupTriage: false} {
		if got := Archivable(group); got != want {
			t.Errorf("Archivable(%s) = %v, want %v", group, got, want)
		}
	}
}

// A state asked for that is not the project's, or is its triage state, is
````

- [ ] **Step 3: 用例**

`server/internal/modules/issue/app/archive_issue.go`（新文件，124 行）：

````file server/internal/modules/issue/app/archive_issue.go
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ArchiveIssueDeps are what archiveIssue and unarchiveIssue read and write
// through.
type ArchiveIssueDeps struct {
	Locks  Locks
	Issues IssueArchiver
	States StateCatalog
	Clock  Clock
}

// ArchiveIssue archives a work item: POST
// /api/v0/issues/{issue_id}/archive (M4 design 3.8, 5.1).
type ArchiveIssue struct {
	d ArchiveIssueDeps
}

// NewArchiveIssue returns the use case.
func NewArchiveIssue(d ArchiveIssueDeps) *ArchiveIssue {
	return &ArchiveIssue{d: d}
}

// Execute archives the work item id (archive), or answers it as it is when
// it is archived already; a work item in a state of neither the completed
// nor the cancelled group is refused (409 issue.not_archivable), its state
// read under the project's lock.
func (u *ArchiveIssue) Execute(ctx context.Context, id uuid.UUID) (domain.Issue, error) {
	return u.d.archive(ctx, id, true)
}

// UnarchiveIssue unarchives a work item: POST
// /api/v0/issues/{issue_id}/unarchive (M4 design 3.8, 5.1).
type UnarchiveIssue struct {
	d ArchiveIssueDeps
}

// NewUnarchiveIssue returns the use case.
func NewUnarchiveIssue(d ArchiveIssueDeps) *UnarchiveIssue {
	return &UnarchiveIssue{d: d}
}

// Execute unarchives the work item id (archive), whatever its state, or
// answers it as it is when it is not archived.
func (u *UnarchiveIssue) Execute(ctx context.Context, id uuid.UUID) (domain.Issue, error) {
	return u.d.archive(ctx, id, false)
}

// archive archives the work item id, or unarchives it when on is false (M4
// design 3.5, 3.8, 3.13): Locks runs the write, issue.archive decided on
// it, the guest rule first, before any lock; then the project's write
// gate, which refuses a write that may wait no longer (503 server_busy);
// then in one transaction, the project FOR NO KEY UPDATE, the work item
// read again and the decision again, and under them, in this order: an
// archived project refuses it (409 project.archived); a work item already
// as on would make it is answered as read, and nothing is written; an
// archiving checks the work item's state (archivable); then the clock; the
// work item's archived_at, the moment or none, and its last writer and
// moment, the caller's; its activity's row, 14 or 16. The answer is the
// work item as stored, read back in the transaction.
func (d ArchiveIssueDeps) archive(ctx context.Context, id uuid.UUID, on bool) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	w := write{issue: &id, action: domain.ActionArchive, notFound: domain.ErrIssueNotFound}
	var answer domain.Issue
	err = d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		switch {
		case h.project.Archived:
			return domain.ErrProjectArchived
		case (h.issue.ArchivedAt != nil) == on:
			answer = h.issue
			return nil
		}
		if on {
			if err := d.archivable(ctx, h.issue); err != nil {
				return err
			}
		}
		s := stamp{workspace: h.issue.WorkspaceID, project: h.issue.ProjectID, issue: id, by: actor.UserID, at: d.Clock.Now()}
		a, activity := IssueArchival{ID: id, By: s.by, At: s.at}, domain.UnarchiveActivity()
		if on {
			a.ArchivedAt, activity = &s.at, domain.ArchiveActivity()
		}
		if err := d.Issues.SetArchivedAt(ctx, a); err != nil {
			return err
		}
		if err := d.Issues.InsertActivities(ctx, s.activity([]domain.Activity{activity})); err != nil {
			return err
		}
		answer, err = readBack(ctx, d.Issues, id)
		return err
	})
	if err != nil {
		return domain.Issue{}, err
	}
	return answer, nil
}

// archivable refuses to archive the work item i unless its state, one of
// its project's read under the project's lock (issueState), is of a group a
// work item is archived in (domain.Archivable): 409 issue.not_archivable.
func (d ArchiveIssueDeps) archivable(ctx context.Context, i domain.Issue) error {
	states, err := d.States.States(ctx, i.ProjectID)
	if err != nil {
		return err
	}
	state, err := issueState(states, i)
	switch {
	case err != nil:
		return err
	case !domain.Archivable(state.Group):
		return domain.ErrNotArchivable
	}
	return nil
}
````

`server/internal/modules/issue/app/archive_issue_test.go`（新文件，195 行）：

````file server/internal/modules/issue/app/archive_issue_test.go
package app_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// cancelled is web's state of the cancelled group, which newArchive adds.
var cancelled = app.StateRef{ID: uuid.NewV7(), Name: "Cancelled", Group: domain.GroupCancelled}

func (f *fakeIssues) SetArchivedAt(ctx context.Context, a app.IssueArchival) error {
	if err := f.log.add(ctx, "SetArchivedAt", a.ID); err != nil {
		return err
	}
	if f.issue == nil || a.ID != f.issue.ID {
		return fmt.Errorf("SetArchivedAt: %+v is not the work item %+v", a, f.issue)
	}
	f.archival, f.archivedAt = &a, a.ArchivedAt
	return nil
}

// newArchive is archiveIssue and unarchiveIssue over updateIssue's world
// (newUpdate), web's states with Cancelled, its work item 7 moved into
// Done: made by carol at earlier, not archived.
func newArchive() (*app.ArchiveIssue, *app.UnarchiveIssue, *world, uuid.UUID) {
	_, w, id := newUpdate()
	w.project.states = append(w.project.states, cancelled)
	w.issues.issue.StateID = done.ID
	d := app.ArchiveIssueDeps{Locks: w.locks, Issues: w.issues, States: w.project, Clock: clockAt{at: clockNow, log: w.log}}
	return app.NewArchiveIssue(d), app.NewUnarchiveIssue(d), w, id
}

// stored is newArchive's work item as a read answers it, archived at
// archivedAt, or not for nil, last written by by at at.
func stored(id uuid.UUID, state uuid.UUID, archivedAt *time.Time, by uuid.UUID, at time.Time) domain.Issue {
	return domain.Issue{ID: id, WorkspaceID: acme, ProjectID: web, SequenceID: 7, Name: "Fix", StateID: state, Priority: domain.PriorityHigh,
		SortOrder: 65535, ArchivedAt: archivedAt, AssigneeIDs: []uuid.UUID{bob, out}, LabelIDs: []uuid.UUID{bug}, CreatedByID: &carol,
		UpdatedByID: &by, CreatedAt: earlier, UpdatedAt: at}
}

// An archiving, in the order of M4 design 3.5 and 3.8: Locks' calls on
// issue.archive (lockedAs); the work item's state read under the lock; the
// clock; the work item archived at the moment, by alice; its activity's
// row 14, alice's at the moment; the answer read back. A work item in
// Cancelled is archived as one in Done.
func TestArchiveIssueArchivesTheWorkItem(t *testing.T) {
	for _, state := range []app.StateRef{done, cancelled} {
		t.Run(state.Name, func(t *testing.T) {
			uc, _, w, id := newArchive()
			w.issues.issue.StateID = state.ID
			got, err := uc.Execute(as(alice), id)
			if want := stored(id, state.ID, &clockNow, alice, clockNow); err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("Execute() = %+v, %v; want %+v", got, err, want)
			}
			calls := append(lockedAs(domain.ActionArchive, id), line("States", web), "Now", line("SetArchivedAt", id), line("InsertActivities", 1),
				line("GetIssue", id))
			if !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
			if want := (app.IssueArchival{ID: id, ArchivedAt: &clockNow, By: alice, At: clockNow}); w.issues.archival == nil ||
				!reflect.DeepEqual(*w.issues.archival, want) {
				t.Errorf("the archiving written %+v, want %+v", w.issues.archival, want)
			}
			wantActivity(t, w, id, []domain.Activity{domain.ArchiveActivity()})
		})
	}
}

// An unarchiving, as an archiving, without the state, which it does not
// check: the work item, archived in Backlog, unarchived at the moment, by
// alice; its activity's row 16.
func TestUnarchiveIssueUnarchivesTheWorkItem(t *testing.T) {
	_, uc, w, id := newArchive()
	w.issues.issue.StateID, w.issues.archivedAt = backlog.ID, &earlier
	got, err := uc.Execute(as(alice), id)
	if want := stored(id, backlog.ID, nil, alice, clockNow); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("Execute() = %+v, %v; want %+v", got, err, want)
	}
	calls := append(lockedAs(domain.ActionArchive, id), "Now", line("SetArchivedAt", id), line("InsertActivities", 1), line("GetIssue", id))
	if !slices.Equal(w.log.calls, calls) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
	}
	if want := (app.IssueArchival{ID: id, By: alice, At: clockNow}); w.issues.archival == nil || !reflect.DeepEqual(*w.issues.archival, want) {
		t.Errorf("the unarchiving written %+v, want %+v", w.issues.archival, want)
	}
	wantActivity(t, w, id, []domain.Activity{domain.UnarchiveActivity()})
}

// Archiving an archived work item, in Backlog too, whose state is not
// read, and unarchiving one that is not archived: the precheck, the lock
// and the decisions run as for any; nothing is written, not even the
// clock read, and the answer is the work item as read under the lock, its
// last writer and moment as they were (M4 design 3.8).
func TestArchivingAsItIsWritesNothing(t *testing.T) {
	for _, tt := range []struct {
		name       string
		archive    bool
		archivedAt *time.Time
	}{{"archiving an archived work item", true, &earlier}, {"unarchiving a work item not archived", false, nil}} {
		t.Run(tt.name, func(t *testing.T) {
			archive, unarchive, w, id := newArchive()
			w.issues.issue.StateID, w.issues.archivedAt = backlog.ID, tt.archivedAt
			execute := unarchive.Execute
			if tt.archive {
				execute = archive.Execute
			}
			got, err := execute(as(alice), id)
			if want := stored(id, backlog.ID, tt.archivedAt, carol, earlier); err != nil || !reflect.DeepEqual(got, want) {
				t.Errorf("Execute() = %+v, %v; want %+v", got, err, want)
			}
			if calls := lockedAs(domain.ActionArchive, id); !slices.Equal(w.log.calls, calls) || w.issues.archival != nil ||
				w.issues.activities != nil {
				t.Errorf("calls:\n%q\nwant\n%q; written %+v and %+v, want nothing", w.log.calls, calls, w.issues.archival, w.issues.activities)
			}
		})
	}
}

// What refuses an archiving or an unarchiving: an archived project, under
// the lock, before the work item is looked at, an archived one too (409
// project.archived); a work item in Backlog, its state read, when it is
// archived (409 issue.not_archivable); a role that may not, in the
// precheck; a read or a write that fails, its failure as itself; a request
// without a caller, unauthenticated before any read.
func TestArchiveIssueRefusals(t *testing.T) {
	for _, tt := range []struct {
		name    string
		archive bool
		set     func(w *world)
		ctx     context.Context
		want    error
		calls   int // of the archiving's calls (TestArchiveIssueArchivesTheWorkItem), the work item's state read in Done
	}{
		{"an archived project", true, func(w *world) { w.project.locked.Archived = true }, as(alice), domain.ErrProjectArchived, 6},
		{"an archived project, the work item archived", true, func(w *world) {
			w.project.locked.Archived, w.issues.archivedAt = true, &earlier
		}, as(alice), domain.ErrProjectArchived, 6},
		{"an archived project, unarchiving", false, func(w *world) {
			w.project.locked.Archived, w.issues.archivedAt = true, &earlier
		}, as(alice), domain.ErrProjectArchived, 6},
		{"a work item in Backlog", true, func(w *world) { w.issues.issue.StateID = backlog.ID }, as(alice), domain.ErrNotArchivable, 7},
		{"a role that may not", true, func(w *world) { w.auth.answers = []error{shared.Forbidden()} }, as(alice), shared.Forbidden(), 2},
		{"the state's read fails", true, func(w *world) { w.log.fail = "States" }, as(alice), errDisk, 7},
		{"the archiving's write fails", true, func(w *world) { w.log.fail = "SetArchivedAt" }, as(alice), errDisk, 9},
		{"the activity's write fails", true, func(w *world) { w.log.fail = "InsertActivities" }, as(alice), errDisk, 10},
		{"no caller", true, func(*world) {}, context.Background(), shared.Unauthenticated(), 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			archive, unarchive, w, id := newArchive()
			tt.set(w)
			execute := unarchive.Execute
			if tt.archive {
				execute = archive.Execute
			}
			got, err := execute(tt.ctx, id)
			var se, wantSE *shared.Error
			if !errors.Is(err, tt.want) && (!errors.As(err, &se) || !errors.As(tt.want, &wantSE) || se.Code != wantSE.Code) ||
				got.ID != (uuid.UUID{}) {
				t.Errorf("Execute() = %+v, %v; want no work item and %v", got, err, tt.want)
			}
			calls := slices.Concat(lockedAs(domain.ActionArchive, id), []string{line("States", web), "Now", line("SetArchivedAt", id),
				line("InsertActivities", 1)})[:tt.calls]
			if !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
		})
	}
}

// The states read under the lock answer what the archiving asked: a work
// item in a state that is none of its project's, as the store answers
// them, is an error, not a refusal (409 issue.not_archivable), and nothing
// is written after the read.
func TestArchivingAWorkItemWhoseStateIsNotItsProjectsFails(t *testing.T) {
	archive, _, w, id := newArchive()
	w.issues.issue.StateID = uuid.NewV7()
	_, err := archive.Execute(as(alice), id)
	var se *shared.Error
	if err == nil || errors.As(err, &se) {
		t.Errorf("Execute() = %v; want an error that is no refusal", err)
	}
	if calls := slices.Concat(lockedAs(domain.ActionArchive, id), []string{line("States", web)}); !slices.Equal(w.log.calls, calls) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
	}
}
````

`server/internal/modules/issue/app/fakes_test.go`（修改，4 处）：

````old server/internal/modules/issue/app/fakes_test.go
// last writer and moment. deletions are the deletion's calls' arguments,
// one a table, in order.
````
````new server/internal/modules/issue/app/fakes_test.go
// last writer and moment; archival, when set, is the last archiving's or
// unarchiving's, and its last writer and moment instead; archivedAt the
// work item's archived_at, nil for none. deletions are the deletion's
// calls' arguments, one a table, in order.
````
````old server/internal/modules/issue/app/fakes_test.go
	changed    *app.IssueFields
````
````new server/internal/modules/issue/app/fakes_test.go
	changed    *app.IssueFields
	archival   *app.IssueArchival
	archivedAt *time.Time
````
````old server/internal/modules/issue/app/fakes_test.go
	if f.changed != nil {
````
````new server/internal/modules/issue/app/fakes_test.go
	switch {
	case f.archival != nil:
		return f.archival.By, f.archival.At
	case f.changed != nil:
````
````old server/internal/modules/issue/app/fakes_test.go
		StateID: r.StateID, Priority: r.Priority, SortOrder: r.SortOrder, CompletedAt: r.CompletedAt,
````
````new server/internal/modules/issue/app/fakes_test.go
		StateID: r.StateID, Priority: r.Priority, SortOrder: r.SortOrder, CompletedAt: r.CompletedAt, ArchivedAt: f.archivedAt,
````

`server/internal/modules/issue/app/lock_test.go`（修改，2 处）：

````old server/internal/modules/issue/app/lock_test.go
		}},
````
````new server/internal/modules/issue/app/lock_test.go
		}},
		{"an archiving", domain.ActionArchive, func(w *world, id uuid.UUID) error {
			_, err := app.NewArchiveIssue(app.ArchiveIssueDeps{Locks: w.locks, Issues: w.issues, States: w.project,
				Clock: clockAt{at: clockNow, log: w.log}}).Execute(as(alice), id)
			return err
		}},
		{"an unarchiving", domain.ActionArchive, func(w *world, id uuid.UUID) error {
			w.issues.archivedAt = &earlier
			_, err := app.NewUnarchiveIssue(app.ArchiveIssueDeps{Locks: w.locks, Issues: w.issues, States: w.project,
				Clock: clockAt{at: clockNow, log: w.log}}).Execute(as(alice), id)
			return err
		}},
````
````old server/internal/modules/issue/app/lock_test.go
		t.Run(tt.name, func(t *testing.T) {
			_, w, id := newUpdate()
			gated(w, nil)
````
````new server/internal/modules/issue/app/lock_test.go
		t.Run(tt.name, func(t *testing.T) {
			_, _, w, id := newArchive()
			gated(w, nil)
````

`server/internal/modules/issue/app/ports.go`（修改，1 处）：

````old server/internal/modules/issue/app/ports.go
}

// WorkspaceDirectory finds a workspace by its slug (M4 design 3.3): the
````
````new server/internal/modules/issue/app/ports.go
}

// IssueArchiver is the work items' store as archiveIssue and
// unarchiveIssue write and read them, in the transaction ctx carries (M4
// design 3.5).
type IssueArchiver interface {
	IssueReader
	// SetArchivedAt writes the work item's archived_at and its last writer
	// and moment.
	SetArchivedAt(ctx context.Context, a IssueArchival) error
	// InsertActivities inserts the rows in their order.
	InsertActivities(ctx context.Context, rows []ActivityRow) error
}

// WorkspaceDirectory finds a workspace by its slug (M4 design 3.3): the
````

`server/internal/modules/issue/app/rows.go`（修改，1 处）：

````old server/internal/modules/issue/app/rows.go
}

// UnlinkRows are the rows of one of a work item's tables of accounts or
````
````new server/internal/modules/issue/app/rows.go
}

// IssueArchival is a work item's archiving or unarchiving as it is written
// (M4 design 3.8): its archived_at, the moment, or nil for none, and the
// moment and the account that change it last.
type IssueArchival struct {
	ID         uuid.UUID
	ArchivedAt *time.Time
	By         uuid.UUID
	At         time.Time
}

// UnlinkRows are the rows of one of a work item's tables of accounts or
````

`server/internal/modules/issue/app/update_issue.go`（修改，3 处）：

````old server/internal/modules/issue/app/update_issue.go
		old, found := stateOf(states, i.StateID)
		if !found {
			return named{}, fmt.Errorf("work item %s is in state %s, which is not its project's", i.ID, i.StateID)
````
````new server/internal/modules/issue/app/update_issue.go
		old, err := issueState(states, i)
		if err != nil {
			return named{}, err
````
````old server/internal/modules/issue/app/update_issue.go
// stateOf is the state id of states; found is false when none is.
func stateOf(states []StateRef, id uuid.UUID) (StateRef, bool) {
	i := slices.IndexFunc(states, func(s StateRef) bool { return s.ID == id })
	if i < 0 {
		return StateRef{}, false
````
````new server/internal/modules/issue/app/update_issue.go
// issueState is the state of states, its project's, that the work item i is
// in. A state that is none of them is an error, not a refusal: a work
// item's state is its project's.
func issueState(states []StateRef, i domain.Issue) (StateRef, error) {
	at := slices.IndexFunc(states, func(s StateRef) bool { return s.ID == i.StateID })
	if at < 0 {
		return StateRef{}, fmt.Errorf("work item %s is in state %s, which is not its project's", i.ID, i.StateID)
````
````old server/internal/modules/issue/app/update_issue.go
	return states[i], true
````
````new server/internal/modules/issue/app/update_issue.go
	return states[at], nil
````

- [ ] **Step 4: 存储**

`server/internal/modules/issue/adapter/postgres/update.go`（修改，1 处）：

````old server/internal/modules/issue/adapter/postgres/update.go
		return fmt.Errorf("update work item %s: %w", f.ID, err)
````
````new server/internal/modules/issue/adapter/postgres/update.go
		return fmt.Errorf("update work item %s: %w", f.ID, err)
	}
	return nil
}

// SetArchivedAt writes the work item's archived_at, the moment or none,
// and its last writer and moment, a (M4 design 3.8).
func (s *Store) SetArchivedAt(ctx context.Context, a app.IssueArchival) error {
	err := s.queries(ctx).SetArchivedAt(ctx, gen.SetArchivedAtParams{ID: a.ID, ArchivedAt: a.ArchivedAt, By: a.By, At: a.At})
	if err != nil {
		return fmt.Errorf("archive work item %s: %w", a.ID, err)
````

`server/internal/modules/issue/adapter/postgres/update_test.go`（修改，3 处）：

````old server/internal/modules/issue/adapter/postgres/update_test.go
	"reflect"
````
````new server/internal/modules/issue/adapter/postgres/update_test.go
	"reflect"
	"slices"
````
````old server/internal/modules/issue/adapter/postgres/update_test.go
}

// A write of an update that fails answers its error as itself (M4 design
````
````new server/internal/modules/issue/adapter/postgres/update_test.go
}

// archiveIssue's and unarchiveIssue's row through the store (M4 design
// 3.8): SetArchivedAt archives the work item at later, changed last by bob
// then, and unarchives it, changed last by alice at latest; each time every
// other column of it is as it was, and so are the work items older and
// other, drawn before and after it, so that a statement that took its work
// item by the ids' order, not by equality, would reach one of them.
func TestArchivingAWorkItemWritesItsRow(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	older, id, other := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	createWebIssue(t, s, web, id, alice)
	newIssue(t, pool, web, other, 8, alice)
	newIssue(t, pool, web, older, 9, alice)
	// row is the work item's row as text, but the columns SetArchivedAt writes.
	row := `SELECT (to_jsonb(i) - 'archived_at' - 'updated_at' - 'updated_by_id')::text FROM issues i WHERE id = $1`
	before, stamps, columns := rowsAsText(t, pool, "issues"), audits(t, pool, "issues"), textRows(t, pool, row, id)
	created, _, err := s.GetIssue(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	later, latest := now.Add(time.Hour), now.Add(2*time.Hour)
	for _, tt := range []struct {
		archivedAt *time.Time
		by         uuid.UUID
		at         time.Time
	}{{&later, bob, later}, {nil, alice, latest}} {
		if err := s.SetArchivedAt(ctx, app.IssueArchival{ID: id, ArchivedAt: tt.archivedAt, By: tt.by, At: tt.at}); err != nil {
			t.Fatal(err)
		}
		got, found, err := s.GetIssue(ctx, id)
		want := created
		want.ArchivedAt, want.UpdatedByID, want.UpdatedAt = tt.archivedAt, &tt.by, tt.at
		if err != nil || !found || !reflect.DeepEqual(got, want) {
			t.Errorf("GetIssue() archived at %v = %+v, %v, %v; want %+v", tt.archivedAt, got, found, err, want)
		}
		if kept := textRows(t, pool, row, id); !slices.Equal(kept, columns) {
			t.Errorf("archived at %v, the work item's other columns:\n%s\nwere\n%s", tt.archivedAt, kept, columns)
		}
		after, afterStamps := rowsAsText(t, pool, "issues"), audits(t, pool, "issues")
		for _, unwritten := range []uuid.UUID{older, other} {
			if after[unwritten] != before[unwritten] || afterStamps[unwritten] != stamps[unwritten] {
				t.Errorf("the work item %s:\n%s %s\nwas\n%s %s", unwritten, after[unwritten], afterStamps[unwritten], before[unwritten],
					stamps[unwritten])
			}
		}
	}
}

// A write of an update that fails answers its error as itself (M4 design
````
````old server/internal/modules/issue/adapter/postgres/update_test.go
		t.Errorf("UpdateIssue() = %v; want context.Canceled", err)
	}
````
````new server/internal/modules/issue/adapter/postgres/update_test.go
		t.Errorf("UpdateIssue() = %v; want context.Canceled", err)
	}
	if err := s.SetArchivedAt(cancelled, app.IssueArchival{ID: id, ArchivedAt: &now, By: alice, At: now}); !errors.Is(err, context.Canceled) {
		t.Errorf("SetArchivedAt() = %v; want context.Canceled", err)
	}
````

- [ ] **Step 5: HTTP、接线和文案**

`server/internal/modules/issue/adapter/http/archive_test.go`（新文件，96 行）：

````file server/internal/modules/issue/adapter/http/archive_test.go
package httpadapter_test

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeArchive is archiveIssue, or unarchiveIssue: each call is recorded as
// "caller work item"; it answers answer, or err.
type fakeArchive struct {
	calls  []string
	answer domain.Issue
	err    error
}

func (f *fakeArchive) Execute(ctx context.Context, id uuid.UUID) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String())
	return f.answer, f.err
}

// POST archive goes to archiveIssue, and POST unarchive to unarchiveIssue,
// for the caller and the path's work item; the answer is 200 with the work
// item the use case answers, as getIssue shows it.
func TestArchiveAndUnarchiveIssue(t *testing.T) {
	for _, tt := range []struct {
		path   string
		answer domain.Issue
		json   string
	}{{"/archive", webSeven, webSevenJSON}, {"/unarchive", webEight, webEightJSON}} {
		archive, unarchive := &fakeArchive{answer: webSeven}, &fakeArchive{answer: webEight}
		h := newServer(t, httpadapter.UseCases{ArchiveIssue: archive, UnarchiveIssue: unarchive})
		res, body := do(t, h, request(http.MethodPost, "/api/v0/issues/"+tt.answer.ID.String()+tt.path, "alice", ""))
		if res.StatusCode != http.StatusOK || body != tt.json+"\n" {
			t.Errorf("POST %s = %d %s, want 200 %s", tt.path, res.StatusCode, body, tt.json)
		}
		called, other := archive, unarchive
		if tt.path == "/unarchive" {
			called, other = unarchive, archive
		}
		if want := []string{aliceID.String() + " " + tt.answer.ID.String()}; !slices.Equal(called.calls, want) || len(other.calls) != 0 {
			t.Errorf("POST %s: calls %q and %q, want %q and none", tt.path, called.calls, other.calls, want)
		}
	}
}

// The use cases' refusals, as the contract declares them, and their
// failure, each through both operations; issue.not_archivable is
// archiveIssue's alone.
func TestArchiveIssueRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		paths  []string
		err    error
		status int
		want   string
	}{
		{"no work item", both, domain.ErrIssueNotFound, http.StatusNotFound,
			`{"status":404,"code":"issue.not_found","title":"Not Found","detail":"The work item does not exist, or you cannot see it."}`},
		{"a project guest", both, shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"an archived project", both, domain.ErrProjectArchived, http.StatusConflict,
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
		{"a work item in progress", []string{"/archive"}, domain.ErrNotArchivable, http.StatusConflict,
			`{"status":409,"code":"issue.not_archivable","title":"Conflict","detail":"Only a completed or cancelled work item can be archived; ` +
				`move it to such a state first."}`},
		{"a busy project", both, shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
		{"a failure", both, errGone, http.StatusInternalServerError,
			`{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
	} {
		for _, path := range tt.paths {
			uc := &fakeArchive{err: tt.err}
			h := newServer(t, httpadapter.UseCases{ArchiveIssue: uc, UnarchiveIssue: uc})
			res, body := do(t, h, request(http.MethodPost, "/api/v0/issues/"+webSeven.ID.String()+path, "alice", ""))
			if res.StatusCode != tt.status || body != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
				t.Errorf("%s: POST %s = %d %s, Retry-After %q; want %d %s", tt.name, path, res.StatusCode, body, res.Header.Get("Retry-After"),
					tt.status, tt.want)
			}
		}
	}
}

// both are the paths of archiveIssue and unarchiveIssue, after a work
// item's.
var both = []string{"/archive", "/unarchive"}
````

`server/internal/modules/issue/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/issue/adapter/http/handler.go
}

// UseCases are the use cases behind the module's operations.
````
````new server/internal/modules/issue/adapter/http/handler.go
}

// ArchiveIssueUseCase is app.ArchiveIssue, and app.UnarchiveIssue.
type ArchiveIssueUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) (domain.Issue, error)
}

// UseCases are the use cases behind the module's operations.
````
````old server/internal/modules/issue/adapter/http/handler.go
	DeleteIssue          DeleteIssueUseCase
````
````new server/internal/modules/issue/adapter/http/handler.go
	DeleteIssue          DeleteIssueUseCase
	ArchiveIssue         ArchiveIssueUseCase
	UnarchiveIssue       ArchiveIssueUseCase
````

`server/internal/modules/issue/adapter/http/issues.go`（修改，1 处）：

````old server/internal/modules/issue/adapter/http/issues.go
}

// issueDetail is i as the API shows it: each field that may be null null
````
````new server/internal/modules/issue/adapter/http/issues.go
}

// ArchiveIssue serves POST /api/v0/issues/{issue_id}/archive.
func (h handler) ArchiveIssue(ctx context.Context, req gen.ArchiveIssueRequestObject) (gen.ArchiveIssueResponseObject, error) {
	i, err := h.uc.ArchiveIssue.Execute(ctx, req.IssueID)
	if err != nil {
		return nil, err
	}
	return gen.ArchiveIssue200JSONResponse(issueDetail(i)), nil
}

// UnarchiveIssue serves POST /api/v0/issues/{issue_id}/unarchive.
func (h handler) UnarchiveIssue(ctx context.Context, req gen.UnarchiveIssueRequestObject) (gen.UnarchiveIssueResponseObject, error) {
	i, err := h.uc.UnarchiveIssue.Execute(ctx, req.IssueID)
	if err != nil {
		return nil, err
	}
	return gen.UnarchiveIssue200JSONResponse(issueDetail(i)), nil
}

// issueDetail is i as the API shows it: each field that may be null null
````

`server/internal/modules/issue/module.go`（修改，3 处）：

````old server/internal/modules/issue/module.go
// changing and deleting a work item. For the project module's deletions
// (its IssueCascade port) it soft-deletes a deleted project's rows of
// them, tells whether work items are in a state, and takes deleted labels
// off them.
````
````new server/internal/modules/issue/module.go
// changing, archiving, unarchiving and deleting a work item. For the
// project module's deletions (its IssueCascade port) it soft-deletes a
// deleted project's rows of them, tells whether work items are in a state,
// and takes deleted labels off them.
````
````old server/internal/modules/issue/module.go
	locks := app.NewLocks(d.Projects, store, d.Authorizer, d.WriteGate, d.Tx)
````
````new server/internal/modules/issue/module.go
	locks := app.NewLocks(d.Projects, store, d.Authorizer, d.WriteGate, d.Tx)
	archive := app.ArchiveIssueDeps{Locks: locks, Issues: store, States: d.Projects, Clock: d.Clock}
````
````old server/internal/modules/issue/module.go
		DeleteIssue: app.NewDeleteIssue(app.DeleteIssueDeps{Locks: locks, Rows: store, Clock: d.Clock}),
````
````new server/internal/modules/issue/module.go
		DeleteIssue:    app.NewDeleteIssue(app.DeleteIssueDeps{Locks: locks, Rows: store, Clock: d.Clock}),
		ArchiveIssue:   app.NewArchiveIssue(archive),
		UnarchiveIssue: app.NewUnarchiveIssue(archive),
````

`web/apps/web/core/lib/error-messages.ts`（修改，1 处）：

````old web/apps/web/core/lib/error-messages.ts
  "issue.not_found": "errors.issue_not_found",
````
````new web/apps/web/core/lib/error-messages.ts
  "issue.not_found": "errors.issue_not_found",
  "issue.not_archivable": "errors.issue_not_archivable",
````

`web/packages/i18n/src/locales/en/errors.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/errors.json
    "issue_not_found": "The work item does not exist, or you cannot see it.",
````
````new web/packages/i18n/src/locales/en/errors.json
    "issue_not_found": "The work item does not exist, or you cannot see it.",
    "issue_not_archivable": "Only a completed or cancelled work item can be archived. Move it to such a state first.",
````

`web/packages/i18n/src/locales/zh-CN/errors.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/errors.json
    "issue_not_found": "工作项不存在，或者你看不到它。",
````
````new web/packages/i18n/src/locales/zh-CN/errors.json
    "issue_not_found": "工作项不存在，或者你看不到它。",
    "issue_not_archivable": "只有已完成或已取消的工作项才能归档。请先把它移到已完成或已取消的状态。",
````

- [ ] **Step 6: 组合、取锁形状、写闸门和矩阵**

`server/internal/bootstrap/gate_test.go`（修改，2 处）：

````old server/internal/bootstrap/gate_test.go
	})
	item := w.newWorkItem(t)
	writes := []struct {
````
````new server/internal/bootstrap/gate_test.go
	})
	item, archived := w.newWorkItem(t), w.doneWorkItem(t)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+archived.String()+"/archive", "", http.StatusOK)
	writes := []struct {
````
````old server/internal/bootstrap/gate_test.go
		{"a deletion", http.MethodDelete, "/api/v0/issues/" + w.newWorkItem(t).String(), http.StatusNoContent},
````
````new server/internal/bootstrap/gate_test.go
		{"a deletion", http.MethodDelete, "/api/v0/issues/" + w.newWorkItem(t).String(), http.StatusNoContent},
		{"an archiving", http.MethodPost, "/api/v0/issues/" + w.doneWorkItem(t).String() + "/archive", http.StatusOK},
		{"an unarchiving", http.MethodPost, "/api/v0/issues/" + archived.String() + "/unarchive", http.StatusOK},
````

`server/internal/bootstrap/issue_archive_test.go`（新文件，154 行）：

````file server/internal/bootstrap/issue_archive_test.go
package bootstrap

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// archiveIssue and unarchiveIssue as bootstrap wires them write the work
// item and their activity's row at one moment within their request, as
// their caller (M4 design 3.8, 3.13): alice makes a work item in Web's
// Done; bob archives it, then archives it again; alice unarchives it, then
// unarchives it again. The first archiving archives it, changed last by
// bob at one moment within his request, and writes his row 14 then:
// archived_at, updated, from NULL to manual_archive. The first
// unarchiving unarchives it, changed last by alice at one moment within
// her request, and writes her row 16 then: archive to restore. Each second
// one writes nothing; the creation's rows stay as they were. A work item
// in Todo, made at the start, is not archived (409 issue.not_archivable),
// and nothing of it is written, neither by its refused archiving nor by
// the other work item's archivings; moved to Cancelled, bob archives it.
// Cancelled moved to the started group since (Dropped, a second cancelled
// state, keeps that group in Web), bob's archiving it again is answered as
// it is and writes nothing: an archived work item is answered before its
// state group is asked; and alice restores it: a restoring asks no state
// group.
func TestArchivingAWorkItemStampsItsRows(t *testing.T) {
	w := newIssueWorld(t)
	item, todo := w.doneWorkItem(t), w.newWorkItem(t)
	path := "/api/v0/issues/" + item.String()
	created, todoStamps := w.workItemStamps(t, item), w.workItemStamps(t, todo)

	var moments []time.Time
	for _, step := range []struct {
		name, verb string
		archived   bool
	}{{"bob", "archive", true}, {"alice", "unarchive", false}} {
		before := time.Now().Truncate(time.Microsecond)
		w.send(t, step.name, http.MethodPost, path+"/"+step.verb, "", http.StatusOK)
		after := time.Now()
		archivedAt, changed, by := archivalOf(t, w, item)
		if changed.Before(before) || changed.After(after) || by != w.ids[step.name] || (archivedAt != nil) != step.archived ||
			archivedAt != nil && !archivedAt.Equal(changed) {
			t.Fatalf("after %s's %s: archived at %v, changed last by %s at %v; want archived %v, at one moment within %v to %v", step.name,
				step.verb, archivedAt, by, changed, step.archived, before, after)
		}
		stamps := w.workItemStamps(t, item)
		w.send(t, step.name, http.MethodPost, path+"/"+step.verb, "", http.StatusOK)
		if again := w.workItemStamps(t, item); !slices.Equal(again, stamps) {
			t.Errorf("%s's second %s wrote:\n%q\nwas\n%q", step.name, step.verb, again, stamps)
		}
		moments = append(moments, changed)
	}

	want := []string{
		fmt.Sprintf("updated archived_at <nil> → manual_archive, archived the work item, by %s at %s", w.ids["bob"], moments[0]),
		fmt.Sprintf("updated archived_at archive → restore, restored the work item, by %s at %s", w.ids["alice"], moments[1]),
	}
	if got := archivalRows(t, w, item); !slices.Equal(got, want) {
		t.Errorf("the work item's archived_at rows:\n%q\nwant\n%q", got, want)
	}
	stamps := w.workItemStamps(t, item)
	for _, line := range created {
		if !slices.Contains(stamps, line) && !strings.HasPrefix(line, "issues ") {
			t.Errorf("the creation's row %q changed: the rows are\n%q", line, stamps)
		}
	}

	refused := w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+todo.String()+"/archive", "", http.StatusConflict)
	if code := problemCode(t, []byte(refused)); code != "issue.not_archivable" {
		t.Errorf("archiving the work item in Todo answered %s; want issue.not_archivable", refused)
	}
	if after := w.workItemStamps(t, todo); !slices.Equal(after, todoStamps) {
		t.Errorf("the work item in Todo, after the other's archivings and its own refused one:\n%q\nwas\n%q", after, todoStamps)
	}
	cancelled := stateID(t, w.pool, w.web, "Cancelled")
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+todo.String(), `{"state_id":"`+cancelled.String()+`"}`, http.StatusOK)
	w.send(t, "bob", http.MethodPost, "/api/v0/issues/"+todo.String()+"/archive", "", http.StatusOK)
	if archivedAt, _, by := archivalOf(t, w, todo); archivedAt == nil || by != w.ids["bob"] {
		t.Errorf("the work item moved to Cancelled: archived at %v, changed last by %s; want archived by bob", archivedAt, by)
	}
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/states", `{"name":"Dropped","color":"#9AA4BC","group":"cancelled"}`,
		http.StatusCreated)
	w.send(t, "alice", http.MethodPatch, "/api/v0/states/"+cancelled.String(), `{"group":"started"}`, http.StatusOK)
	archived := w.workItemStamps(t, todo)
	w.send(t, "bob", http.MethodPost, "/api/v0/issues/"+todo.String()+"/archive", "", http.StatusOK)
	if again := w.workItemStamps(t, todo); !slices.Equal(again, archived) {
		t.Errorf("bob's archiving again of the work item archived in Cancelled, of the started group since, wrote:\n%q\nwas\n%q", again,
			archived)
	}
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+todo.String()+"/unarchive", "", http.StatusOK)
	if archivedAt, _, by := archivalOf(t, w, todo); archivedAt != nil || by != w.ids["alice"] {
		t.Errorf("the work item in Cancelled, of the started group since: archived at %v, changed last by %s; want it restored by alice",
			archivedAt, by)
	}
}

// archivalOf is the work item id's archived_at, nil for none, and its last
// change's moment and account.
func archivalOf(t *testing.T, w issueWorld, id uuid.UUID) (*time.Time, time.Time, uuid.UUID) {
	t.Helper()
	var archivedAt *time.Time
	var changed time.Time
	var by uuid.UUID
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT archived_at, updated_at, updated_by_id FROM issues WHERE id = $1", id).Scan(&archivedAt,
		&changed, &by); err != nil {
		t.Fatal(err)
	}
	return archivedAt, changed, by
}

// archivalRows are the work item id's activity rows on archived_at, in
// their ids' order, each as "<verb> archived_at <old> → <new>, <comment>,
// by <actor> at <moment>", each of whose made and last-changed stamps is
// its actor's and its moment's; one that is not fails the test.
func archivalRows(t *testing.T, w issueWorld, id uuid.UUID) []string {
	t.Helper()
	rows, err := w.pool.Query(pgtest.Soon(t), `SELECT verb, old_value, new_value, comment, actor_id, created_at, created_by_id, updated_by_id,
		updated_at FROM issue_activities WHERE issue_id = $1 AND field = 'archived_at' ORDER BY id`, id)
	if err != nil {
		t.Fatal(err)
	}
	lines, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (string, error) {
		var verb, comment string
		var oldValue, newValue *string
		var actor, madeBy, changedBy uuid.UUID
		var at, changedAt time.Time
		if err := row.Scan(&verb, &oldValue, &newValue, &comment, &actor, &at, &madeBy, &changedBy, &changedAt); err != nil {
			return "", err
		}
		if madeBy != actor || changedBy != actor || !changedAt.Equal(at) {
			t.Errorf("an archived_at row by %s at %v was made by %s, changed by %s at %v", actor, at, madeBy, changedBy, changedAt)
		}
		value := func(v *string) string {
			if v == nil {
				return "<nil>"
			}
			return *v
		}
		return fmt.Sprintf("%s archived_at %s → %s, %s, by %s at %s", verb, value(oldValue), value(newValue), comment, actor, at), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return lines
}
````

`server/internal/bootstrap/issue_updates_test.go`（修改，1 处）：

````old server/internal/bootstrap/issue_updates_test.go
// archived (in the database: no operation of P1b archives a work item),
// moved into Done by bob after alice made it and changed by him again
// after that, so that each field that may be null has a value of its own,
// each of its stamps is its own, and it was made, completed, changed last
// and archived at four moments, in that order.
````
````new server/internal/bootstrap/issue_updates_test.go
// archived (in the database, a minute after its last change: archiveIssue
// changes it last as it archives it), moved into Done by bob after alice
// made it and changed by him again after that, so that each field that
// may be null has a value of its own, each of its stamps is its own, and
// it was made, completed, changed last and archived at four moments, in
// that order.
````

`server/internal/bootstrap/issue_world_test.go`（修改，1 处）：

````old server/internal/bootstrap/issue_world_test.go
}

// fullIssue is the body of a work item in Todo, assigned to bob and
````
````new server/internal/bootstrap/issue_world_test.go
}

// doneWorkItem is the id of a work item alice creates in Web's Done
// through the API, completed at its creation.
func (w issueWorld) doneWorkItem(t *testing.T) uuid.UUID {
	t.Helper()
	return w.newWorkItemIn(t, w.web, `{"name":"Fix","state_id":"`+w.done.String()+`"}`)
}

// fullIssue is the body of a work item in Todo, assigned to bob and
````

`server/internal/bootstrap/issue_write_locks_test.go`（修改，1 处）：

````old server/internal/bootstrap/issue_write_locks_test.go
		body:   func(issueWorld) string { return "" }, want: http.StatusNoContent, waits: "issues", mode: "SHARE"},
````
````new server/internal/bootstrap/issue_write_locks_test.go
		body:   func(issueWorld) string { return "" }, want: http.StatusNoContent, waits: "issues", mode: "SHARE"},
	// A work item in Done, which the send archives: its state is read
	// under the lock.
	{op: "archiveIssue", method: http.MethodPost, path: "/api/v0/issues/{issue_id}/archive",
		target: func(t *testing.T, w issueWorld) string {
			return "/api/v0/issues/" + w.doneWorkItem(t).String() + "/archive"
		},
		body: func(issueWorld) string { return "" }, want: http.StatusOK, waits: "states", mode: "ACCESS EXCLUSIVE"},
	// An archived work item, which the send unarchives: it reads nothing
	// under the lock that its precheck did not, so it waits for the work
	// items' table, which it writes first.
	{op: "unarchiveIssue", method: http.MethodPost, path: "/api/v0/issues/{issue_id}/unarchive",
		target: func(t *testing.T, w issueWorld) string {
			id := w.doneWorkItem(t)
			w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+id.String()+"/archive", "", http.StatusOK)
			return "/api/v0/issues/" + id.String() + "/unarchive"
		},
		body: func(issueWorld) string { return "" }, want: http.StatusOK, waits: "issues", mode: "SHARE"},
````

`server/internal/bootstrap/issue_writes_test.go`（修改，2 处）：

````old server/internal/bootstrap/issue_writes_test.go
// deletes the second. With Web archived, her creation and her change are
// refused under the lock (409).
````
````new server/internal/bootstrap/issue_writes_test.go
// archives it, in Done, and unarchives it; she deletes the second. With
// Web archived, her creation and her change are refused under the lock
// (409).
````
````old server/internal/bootstrap/issue_writes_test.go
	route.answerWithin(t, w.contract, http.MethodGet, "/api/v0/issues/"+first.ID.String(), alice, "", http.StatusOK)
````
````new server/internal/bootstrap/issue_writes_test.go
	route.answerWithin(t, w.contract, http.MethodGet, "/api/v0/issues/"+first.ID.String(), alice, "", http.StatusOK)
	for _, verb := range []string{"archive", "unarchive"} {
		route.answerWithin(t, w.contract, http.MethodPost, "/api/v0/issues/"+first.ID.String()+"/"+verb, alice, "", http.StatusOK)
	}
````

`server/internal/bootstrap/permission_matrix_coverage_test.go`（修改，4 处）：

````old server/internal/bootstrap/permission_matrix_coverage_test.go
// that sends anything but GET without write, whose cells could write on the
// copy the reading cells share. The requests name the rows of s.
````
````new server/internal/bootstrap/permission_matrix_coverage_test.go
// that sends anything but GET without write, or prepares its cells without
// it, whose cells could write on the copy the reading cells share. The
// requests name the rows of s.
````
````old server/internal/bootstrap/permission_matrix_coverage_test.go
			found = append(found, fmt.Sprintf("row %s sends %s without write: its cells could run on the reads' copy", r.name(), unsafe))
````
````new server/internal/bootstrap/permission_matrix_coverage_test.go
			found = append(found, fmt.Sprintf("row %s sends %s without write: its cells could run on the reads' copy", r.name(), unsafe))
		}
		if r.prepare != nil && !r.write {
			found = append(found, fmt.Sprintf("row %s prepares its cells without write: it would change the reads' copy", r.name()))
````
````old server/internal/bootstrap/permission_matrix_coverage_test.go
	posting.write = true
````
````new server/internal/bootstrap/permission_matrix_coverage_test.go
	posting.write = true
	prepared := row
	prepared.prepare = func(*testing.T, caller, seeded, string) {}
````
````old server/internal/bootstrap/permission_matrix_coverage_test.go
			[]string{"row acceptWorkspaceInvitation sends POST without write: its cells could run on the reads' copy"}},
````
````new server/internal/bootstrap/permission_matrix_coverage_test.go
			[]string{"row acceptWorkspaceInvitation sends POST without write: its cells could run on the reads' copy"}},
		{"a prepare without write", ops, exempt, []matrixRow{prepared},
			[]string{"row getWorkspace prepares its cells without write: it would change the reads' copy"}},
````

`server/internal/bootstrap/permission_matrix_issue_archive_test.go`（新文件，94 行）：

````file server/internal/bootstrap/permission_matrix_issue_archive_test.go
package bootstrap

import (
	"net/http"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The rows of the permission matrix of archiving and unarchiving a work
// item (M4 design 3.8, 9.2), naming it by its id: its column's work item
// (issueOf), which each cell's prepare moves into Done, completed, or into
// Done and archived, on the cell's copy, for the cases that need it. A row
// of 9.2's table has the issue table's columns; a variant on an archived
// project, the archived project's.

var cellNotArchivable = cell{http.StatusConflict, "issue.not_archivable"}

func issueArchiveMatrixRows() []matrixRow {
	archive, unarchive := toWorkItem(http.MethodPost, "/archive", ""), toWorkItem(http.MethodPost, "/unarchive", "")
	return []matrixRow{
		// The project's admins and members, the member who made it too, not
		// its guests (M4 design 3.4, 9.2).
		{op: "archiveIssue", write: true, columns: issueColumns, prepare: inDone(false), request: archive,
			cells: onAWorkItem(cellOK, cellOK, cellOK, cellForbidden), check: asArchived(true, true)},
		// A work item in Todo is not archived (M4 design 3.8), after the
		// decision: who may not archive it learns nothing of its state.
		{op: "archiveIssue", variant: "in Todo", write: true, columns: issueColumns, request: archive,
			cells: onAWorkItem(cellNotArchivable, cellNotArchivable, cellNotArchivable, cellForbidden)},
		// An archived work item is answered as it is, after the decision.
		{op: "archiveIssue", variant: "archived work item", write: true, columns: issueColumns, prepare: inDone(true), request: archive,
			cells: onAWorkItem(cellOK, cellOK, cellOK, cellForbidden), check: asArchived(true, false)},
		// An archived project's work items are not archived, after the
		// decision, before their state is looked at.
		{op: "archiveIssue", variant: "archived", write: true, columns: archivedColumns, request: archive,
			cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		// Nor are they answered as they are when archived: the project's 409
		// comes first.
		{op: "archiveIssue", variant: "archived, its work item archived", write: true, columns: archivedColumns, prepare: inDone(true),
			request: archive, cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		// As archiveIssue (M4 design 3.4, 9.2), on an archived work item.
		{op: "unarchiveIssue", write: true, columns: issueColumns, prepare: inDone(true), request: unarchive,
			cells: onAWorkItem(cellOK, cellOK, cellOK, cellForbidden), check: asArchived(false, true)},
		// A work item that is not archived is answered as it is, after the
		// decision.
		{op: "unarchiveIssue", variant: "not archived", write: true, columns: issueColumns, request: unarchive,
			cells: onAWorkItem(cellOK, cellOK, cellOK, cellForbidden), check: readsTheIssue},
		// An archived project's work items are not unarchived, after the
		// decision, before the work item is looked at.
		{op: "unarchiveIssue", variant: "archived", write: true, columns: archivedColumns, request: unarchive,
			cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		{op: "unarchiveIssue", variant: "archived, its work item archived", write: true, columns: archivedColumns, prepare: inDone(true),
			request: unarchive, cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
	}
}

// inDone is a cell's prepare: its column's work item moved into its
// project's Done, completed when it was last changed, and archived then
// too when archived, as archiveIssue leaves one, through SQL on the cell's
// copy at url, over a pool it closes before the cell's app starts.
func inDone(archived bool) func(t *testing.T, c caller, s seeded, url string) {
	return func(t *testing.T, c caller, s seeded, url string) {
		t.Helper()
		pool := openPool(t, url)
		defer pool.Close()
		tag, err := pool.Exec(pgtest.Soon(t), `UPDATE issues SET state_id = $2, completed_at = updated_at,
			archived_at = CASE WHEN $3 THEN updated_at END WHERE id = $1`, s.issue(projectOf(c), issueOf(c)), s.state(projectOf(c), "Done"), archived)
		if err != nil || tag.RowsAffected() != 1 {
			t.Fatalf("moving %s's work item into Done: %v, %d rows", c, err, tag.RowsAffected())
		}
	}
}

// asArchived is the check of a row whose prepare moved the column's work
// item into Done (inDone): the answer is the work item so moved, as
// seeded otherwise and completed (answered), archived when archived;
// changed last by the caller when written, else by its maker.
func asArchived(archived, written bool) func(t *testing.T, c caller, s seeded, answer string) {
	return func(t *testing.T, c caller, s seeded, answer string) {
		var got struct {
			ArchivedAt *string `json:"archived_at"`
		}
		decodeAnswer(t, answer, &got)
		if (got.ArchivedAt != nil) != archived {
			t.Errorf("%s answers %s; want it archived %v", c, answer, archived)
		}
		answered(t, c, s, answer, true, func(i *answeredIssue) {
			i.StateID = s.state(projectOf(c), "Done")
			if !written {
				i.UpdatedByID = i.CreatedByID
			}
		})
	}
}
````

`server/internal/bootstrap/permission_matrix_issue_updates_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
// seeded (seededIssue) with change made to it, the caller its last writer,
// and completed when completed; completedAt is not compared, but present
// or not.
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
// seeded (seededIssue) with change made to it, the caller its last writer
// unless change says otherwise, and completed when completed; completedAt
// is not compared, but present or not.
````

`server/internal/bootstrap/permission_matrix_test.go`（修改，3 处）：

````old server/internal/bootstrap/permission_matrix_test.go
	refusal string
````
````new server/internal/bootstrap/permission_matrix_test.go
	refusal string
	// prepare, when set, changes each cell's copy, at url, before its app
	// starts: what the row's case needs that the prepared data lacks.
	prepare func(t *testing.T, c caller, s seeded, url string)
````
````old server/internal/bootstrap/permission_matrix_test.go
		labelMatrixRows(), issueMatrixRows(), issueUpdateMatrixRows())
````
````new server/internal/bootstrap/permission_matrix_test.go
		labelMatrixRows(), issueMatrixRows(), issueUpdateMatrixRows(), issueArchiveMatrixRows())
````
````old server/internal/bootstrap/permission_matrix_test.go
					base = startApp(t, d.config(t, pgtest.NewDatabaseFrom(t, d.url), r.config), migrations.FS())
````
````new server/internal/bootstrap/permission_matrix_test.go
					url := pgtest.NewDatabaseFrom(t, d.url)
					if r.prepare != nil {
						r.prepare(t, c, d.seeded.in(t), url)
					}
					base = startApp(t, d.config(t, url, r.config), migrations.FS())
````

- [ ] **Step 7: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestArchivingAWorkItemStampsItsRows$|TestEachIssueWriteLocksItsProjectFirst$|TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy$|TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection$|TestAWorkItemReadsAsStored$|TestPermissionMatrix|TestMatrixViolations|TestThePermissionMatrixCoversEveryOperation$' ./internal/bootstrap/`
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

- [ ] **Step 8: 提交**

```bash
git add api/modules/issue.yaml api/openapi.yaml server/internal/bootstrap/gate_test.go server/internal/bootstrap/issue_archive_test.go server/internal/bootstrap/issue_updates_test.go server/internal/bootstrap/issue_world_test.go server/internal/bootstrap/issue_write_locks_test.go server/internal/bootstrap/issue_writes_test.go server/internal/bootstrap/permission_matrix_coverage_test.go server/internal/bootstrap/permission_matrix_issue_archive_test.go server/internal/bootstrap/permission_matrix_issue_updates_test.go server/internal/bootstrap/permission_matrix_test.go server/internal/modules/issue/adapter/http/archive_test.go server/internal/modules/issue/adapter/http/handler.go server/internal/modules/issue/adapter/http/issues.go server/internal/modules/issue/adapter/postgres/queries/update.sql server/internal/modules/issue/adapter/postgres/update.go server/internal/modules/issue/adapter/postgres/update_test.go server/internal/modules/issue/app/archive_issue.go server/internal/modules/issue/app/archive_issue_test.go server/internal/modules/issue/app/fakes_test.go server/internal/modules/issue/app/lock_test.go server/internal/modules/issue/app/ports.go server/internal/modules/issue/app/rows.go server/internal/modules/issue/app/update_issue.go server/internal/modules/issue/domain/activity.go server/internal/modules/issue/domain/activity_test.go server/internal/modules/issue/domain/errors.go server/internal/modules/issue/domain/issue.go server/internal/modules/issue/domain/issue_test.go server/internal/modules/issue/module.go web/apps/web/core/lib/error-messages.ts web/packages/i18n/src/locales/en/errors.json web/packages/i18n/src/locales/zh-CN/errors.json api/dist/openapi.yaml server/internal/modules/issue/adapter/http/gen/server.gen.go server/internal/modules/issue/adapter/postgres/gen/update.sql.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M4/P2): archiveIssue and unarchiveIssue, on the one lock path

POST /api/v0/issues/{issue_id}/archive and /unarchive decide
issue.archive (the project's admins and members) and run on the lock
path. Under the lock an archived project refuses them; a work item
already as asked is answered as read and nothing is written; an
archiving needs the work item's state, read under the lock, of the
completed or the cancelled group (409 issue.not_archivable, with its
copy). The work item is stamped by the caller at one moment with its
activity's row 14 or 16, and answered as stored. The matrix gains
prepare, run on a writing cell's copy before its app starts.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p2.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试，或没有替身的平台、领域、转换的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是生成器（`make gen-go`）和中英文翻译键（`make lint-web`）的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过，端到端的检查只在标出的 Task 的树上）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T7.1` | 已归档的项目里的工作项照样归档、恢复 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `archiveIssue`、`unarchiveIssue` 各格（`make test`） | 单元；组合 |
| `T7.2` | 已经是所求样子的工作项先于它的项目已归档的 409 答 200 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `archiveIssue`、`unarchiveIssue` 各格（`make test`） | 单元；组合 |
| `T7.3` | 再归档已归档的、再恢复没归档的，都再写一次 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `archiveIssue`、`unarchiveIssue` 各格（`make test`）、`internal/bootstrap` 的 `TestArchivingAWorkItemStampsItsRows`（`make test`）、故事 `i15-archive-issue`（Task 13 起） | 单元；组合；端到端 |
| `T7.4` | 任何状态的工作项都能归档 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `archiveIssue`、`unarchiveIssue` 各格（`make test`）、`internal/bootstrap` 的 `TestArchivingAWorkItemAndMovingItSerialize`（`make test`）（Task 12 起）、故事 `i15-archive-issue`（Task 13 起） | 单元；组合；端到端 |
| `T7.5` | 已取消的工作项不能归档 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestArchivingAWorkItemStampsItsRows`（`make test`） | 单元；组合 |
| `T7.6` | 工作项的归档时刻是它上一次修改的时刻，不是归档的时刻 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestArchivingAWorkItemStampsItsRows`（`make test`）、故事 `i15-archive-issue`（Task 13 起） | 单元；组合；端到端 |
| `T7.7` | 恢复写归档的那一行动态 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestArchivingAWorkItemStampsItsRows`（`make test`）、故事 `i15-archive-issue`（Task 13 起） | 单元；组合；端到端 |
| `T7.8` | 恢复的一行从 `manual_archive` 起，而不是从 `archive` | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestArchivingAWorkItemStampsItsRows`（`make test`）、故事 `i15-archive-issue`（Task 13 起） | 单元；组合；端到端 |
| `T7.9` | 归档保留工作项原来的最后修改者 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestArchivingAWorkItemStampsItsRows`（`make test`） | 存储；组合 |
| `T7.10` | `POST …/unarchive` 归档 | `internal/bootstrap` 的 `TestPermissionMatrix` 的 `archiveIssue`、`unarchiveIssue` 各格（`make test`）、`internal/bootstrap` 的 `TestArchivingAWorkItemStampsItsRows`（`make test`）、故事 `i15-archive-issue`（Task 13 起） | 组合；端到端 |
| `T7.11` | 已经是所求样子的工作项答一个空的工作项 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `archiveIssue`、`unarchiveIssue` 各格（`make test`）、故事 `i15-archive-issue`（Task 13 起） | 单元；组合；端到端 |
| `T7.12` | 页面没有 `issue.not_archivable` 的文案 | `error-messages.test.ts`（`make test-web`） | 单元 |
| `T7.13` | 中文的一份没有 `issue_not_archivable` | `web/packages/i18n` 的 `check:sync`（`make lint-web`） | 守卫 |
| `T7.14` | 契约不声明 `archiveIssue` 答 `issue.not_archivable` | `internal/modules/issue/adapter/http` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `archiveIssue`、`unarchiveIssue` 各格（`make test`）、`internal/bootstrap` 的 `TestArchivingAWorkItemStampsItsRows`（`make test`） | 单元；组合 |
| `T7.15` | 归档把每个工作项都归档 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestArchivingAWorkItemStampsItsRows`（`make test`） | 存储；组合 |
| `T7.16` | `archiveIssue` 经连接池读回答，不在它的事务的连接上 | `internal/bootstrap` 的 `TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`（`make test`） | 组合 |
| `T7.17` | 归档经连接池读项目的状态，不在它的事务的连接上 | `internal/bootstrap` 的 `TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`（`make test`） | 组合 |
| `T7.18` | 归档把不是项目的状态当作不能归档的状态（409），而不是错误 | `internal/modules/issue/app` 的全部测试（`make test`） | 单元 |
| `T7.19` | 归档、恢复不进项目的写闸门 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy`（`make test`） | 单元；组合 |
| `T7.20` | 恢复也检查状态组 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestArchivingAWorkItemStampsItsRows`（`make test`） | 单元；组合 |
| `T7.21` | 归档按 id 的次序而不是相等取工作项：id 不在它之后的每个工作项都归档 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy`（`make test`）（Task 8 起） | 存储；组合 |
| `T7.22` | 归档也写工作项的一列 `GetIssue` 不读的（`description_html`） | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`） | 存储 |
| `T7.23` | 矩阵的自检放过没有 `write` 而带 `prepare` 的行：它的 `prepare` 会改动读的格子共用的副本 | `internal/bootstrap` 的 `TestMatrixViolationsCatchesEachGap`（`make test`） | 单元 |
| `T7.24` | 状态不能归档的工作项答项目已归档的 409，而不是 `issue.not_archivable` | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `archiveIssue`、`unarchiveIssue` 各格（`make test`）、`internal/bootstrap` 的 `TestArchivingAWorkItemStampsItsRows`（`make test`） | 单元；组合 |
| `T7.25` | 归档先看状态组、再答已归档的工作项：在不能归档的组里的已归档的工作项答 409 `issue.not_archivable`，而不是照样答 200 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestArchivingAWorkItemStampsItsRows`（`make test`） | 单元；组合 |

本 Task 的检查也发现较早的 Task 的变异 `T1.11`（列在各自的 Task 的表里，标"（Task 7 起）"）：本 Task 做完之后，在本 Task 的树上重跑它们（`mutants_p2.py`，`MUT_LATER=7`）。

---

### Task 8: 已归档的工作项只读；`state_in_use` 的补救

**Files:**
- Modify: `api/modules/issue.yaml`、`api/modules/project.yaml`、`server/internal/bootstrap/issue_cascade_test.go`、`server/internal/bootstrap/permission_matrix_issue_updates_test.go`、`server/internal/modules/issue/adapter/http/update_test.go`、`server/internal/modules/issue/app/delete_issue_test.go`、`server/internal/modules/issue/app/update_issue.go`、`server/internal/modules/issue/app/update_issue_test.go`、`server/internal/modules/issue/domain/errors.go`、`server/internal/modules/project/adapter/http/state_writes_test.go`、`server/internal/modules/project/domain/errors.go`、`web/apps/web/core/lib/error-messages.ts`、`web/packages/i18n/src/locales/en/errors.json`、`web/packages/i18n/src/locales/zh-CN/errors.json`
- Generate: `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**（spec 2.9、第 3 节第 5、9 条；M4 设计 3.5、3.8）
- `issue/domain/errors.go`：`ErrIssueArchived`（409 `issue.archived`：已归档的工作项的修改，什么都不改的也是）。
- `issue/app/update_issue.go`：锁下的次序：项目已归档 409 → 工作项已归档 409 `issue.archived`（什么都不改的修改也是）→ `ChangesOf`；说明照改。
- 契约：`updateIssue` 的描述和 `x-problem-codes` 加 `issue.archived`；`api/modules/project.yaml` 的 `deleteState` 的描述加上补救。
- `project/domain/errors.go`：`ErrStateInUse` 的说明和文字加上补救（"unarchive the archived ones and move them all to another state first"）。
- 前端：`error-messages.ts` 的 `issue.archived`，两份 `errors.json` 的 `issue_archived` 和 `project_state_in_use` 的补救（措辞第 3 节第 9 条）。
- 矩阵：`permission_matrix_issue_updates_test.go` 的 `cellIssueArchived`；`updateIssue` 的"已归档的项目里已归档的工作项"（项目的 409 在先）、"已归档的工作项"（判定之后 409，什么都不改也是）；`deleteIssue` 的同样两行（已归档的工作项照样删除）；格子的 `prepare` 是 Task 7 的 `inDone(true)`。

**Tests:**
- 单元：`TestUpdateIssueRefusesUnderTheLock`（已归档的项目里已归档的工作项答项目的 409；已归档的工作项，什么都不改的也 409 `issue.archived`，在读它要移进的状态之前）；`TestDeleteIssueDeletesTheWorkItemAndItsRows` 也删除已归档的工作项（同样的调用和删除：已归档的工作项照样删除）。
- HTTP：`TestUpdateIssueRefusals`（`issue.archived`）；`project` 的 `TestDeleteState`（`stateInUseJSON` 带补救）。
- 组合：`TestPermissionMatrix` 的 `updateIssue`、`deleteIssue` 各两行；`TestDeletingALabelTakesItOffTheWorkItems`（alice 删除 Bug 之前，带 Bug 的工作项已归档：它的标签行照样删除）。

- [ ] **Step 1: 契约，生成**

`api/modules/issue.yaml`（修改，2 处）：

````old api/modules/issue.yaml
        is decided again. A work item of an archived project cannot be changed
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
        is the caller's, at one moment. A project with too many writes on
        its work items at once answers server_busy, with Retry-After: try
        again then.
````
````new api/modules/issue.yaml
        is decided again. A work item of an archived project cannot be
        changed (project.archived), nor then an archived work item
        (issue.archived), even by a change that changes nothing. Then the
        state, when it changes, is one of the project's, not its triage
        state; each account the work item is newly assigned to is an active
        member of the project who is not its guest, while one it keeps stays
        assigned whatever has become of him; each label added is one of the
        project's (validation_failed, every problem at once). A change that
        changes nothing writes nothing, and the work item is answered as it
        is. Otherwise the work item is changed last by the caller; moved
        into a state of the completed group, it is completed at the change,
        and moved into another, not completed; the accounts newly assigned
        subscribe to it; its activity says what changed: the name, the
        priority, the state, and each label and each assignee added and
        removed. Every row it writes is the caller's, at one moment. A
        project with too many writes on its work items at once answers
        server_busy, with Retry-After: try again then.
````
````old api/modules/issue.yaml
      x-problem-codes: [validation_failed, issue.not_found, forbidden, project.archived, server_busy]
````
````new api/modules/issue.yaml
      x-problem-codes: [validation_failed, issue.not_found, forbidden, project.archived, issue.archived, server_busy]
````

`api/modules/project.yaml`（修改，1 处）：

````old api/modules/project.yaml
        in, an archived one too (project.state_in_use); it is checked before
        the default and the group. The state is deleted at the moment of the
        request, by the caller, and its name is free again in the project.
        The role is decided after the workspace and project rows are locked.
````
````new api/modules/project.yaml
        in, an archived one too (project.state_in_use): unarchive the
        archived ones and move them all to another state first. It is
        checked before the default and the group. The state is deleted at
        the moment of the request, by the caller, and its name is free again
        in the project. The role is decided after the workspace and project
        rows are locked.
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `ba0ab50a1ff0b2af2f69bed7ab3d8b770b4bf486524aff191a4c7ab780754c40` | 3300 | `api/dist/openapi.yaml` |
| `bbcb0639cef4a6b87d246c4c20be430e9ecb4c6fea1ec009652531d1c8ca5ae6` | 3666 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: 已归档只读和补救**

`server/internal/modules/issue/adapter/http/update_test.go`（修改，1 处）：

````old server/internal/modules/issue/adapter/http/update_test.go
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
````
````new server/internal/modules/issue/adapter/http/update_test.go
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
		{"an archived work item", domain.ErrIssueArchived, http.StatusConflict,
			`{"status":409,"code":"issue.archived","title":"Conflict","detail":"The work item is archived; unarchive it to change it."}`},
````

`server/internal/modules/issue/app/delete_issue_test.go`（修改，3 处）：

````old server/internal/modules/issue/app/delete_issue_test.go
	"testing"
````
````new server/internal/modules/issue/app/delete_issue_test.go
	"testing"
	"time"
````
````old server/internal/modules/issue/app/delete_issue_test.go
// table in the global order (deletingRows), each alice's at the moment.
````
````new server/internal/modules/issue/app/delete_issue_test.go
// table in the global order (deletingRows), each alice's at the moment. An
// archived work item is deleted as any other (M4 design 3.8).
````
````old server/internal/modules/issue/app/delete_issue_test.go
	uc, w, id := newDelete()
	if err := uc.Execute(as(alice), id); err != nil {
		t.Fatal(err)
	}
	if calls := slices.Concat(lockedAs(domain.ActionDelete, id), []string{"Now"}, deletingRows(id)); !slices.Equal(w.log.calls, calls) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
	}
	deletion := app.IssueDeletion{IssueID: id, By: alice, At: clockNow}
	if want := slices.Repeat([]app.IssueDeletion{deletion}, 5); !slices.Equal(w.issues.deletions, want) {
		t.Errorf("the deletions %+v, want %+v", w.issues.deletions, want)
````
````new server/internal/modules/issue/app/delete_issue_test.go
	for what, archivedAt := range map[string]*time.Time{"a work item": nil, "an archived work item": &earlier} {
		t.Run(what, func(t *testing.T) {
			uc, w, id := newDelete()
			w.issues.archivedAt = archivedAt
			if err := uc.Execute(as(alice), id); err != nil {
				t.Fatal(err)
			}
			if calls := slices.Concat(lockedAs(domain.ActionDelete, id), []string{"Now"}, deletingRows(id)); !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
			deletion := app.IssueDeletion{IssueID: id, By: alice, At: clockNow}
			if want := slices.Repeat([]app.IssueDeletion{deletion}, 5); !slices.Equal(w.issues.deletions, want) {
				t.Errorf("the deletions %+v, want %+v", w.issues.deletions, want)
			}
		})
````

`server/internal/modules/issue/app/update_issue.go`（修改，2 处）：

````old server/internal/modules/issue/app/update_issue.go
// refuses it (409 project.archived); what p changes in the work item
// (domain.ChangesOf), and when it changes nothing, nothing is written and
// the answer is the work item as read; the dependent rows it adds (check),
// every problem in one 422; the clock; the work item's fields, its
// completion by its new state's group when its state changes
````
````new server/internal/modules/issue/app/update_issue.go
// refuses it (409 project.archived), then an archived work item (409
// issue.archived), a change of nothing too; what p changes in the work
// item (domain.ChangesOf), and when it changes nothing, nothing is written
// and the answer is the work item as read; the dependent rows it adds
// (check), every problem in one 422; the clock; the work item's fields,
// its completion by its new state's group when its state changes
````
````old server/internal/modules/issue/app/update_issue.go
		if h.project.Archived {
			return domain.ErrProjectArchived
````
````new server/internal/modules/issue/app/update_issue.go
		switch {
		case h.project.Archived:
			return domain.ErrProjectArchived
		case h.issue.ArchivedAt != nil:
			return domain.ErrIssueArchived
````

`server/internal/modules/issue/app/update_issue_test.go`（修改，2 处）：

````old server/internal/modules/issue/app/update_issue_test.go
// changes nothing too (409 project.archived). Then the dependent rows the
// update adds: a state not the project's, or its triage state; an account
// added who is its guest; a label added that is not its: every problem in
// one 422, each at its place in the patch (the label second in label_ids,
// though first among the labels added). out, kept, is not checked, though
// no member of web.
````
````new server/internal/modules/issue/app/update_issue_test.go
// changes nothing too (409 project.archived), its archived work items too;
// then an archived work item refuses it, one that changes nothing too,
// before the state it moves into is read (409 issue.archived). Then the
// dependent rows the update adds: a state not the project's, or its triage
// state; an account added who is its guest; a label added that is not its:
// every problem in one 422, each at its place in the patch (the label
// second in label_ids, though first among the labels added). out, kept, is
// not checked, though no member of web.
````
````old server/internal/modules/issue/app/update_issue_test.go
		{"archived", func(w *world) { w.project.locked.Archived = true }, domain.IssuePatch{}, domain.ErrProjectArchived, locked},
````
````new server/internal/modules/issue/app/update_issue_test.go
		{"archived", func(w *world) { w.project.locked.Archived = true }, domain.IssuePatch{}, domain.ErrProjectArchived, locked},
		{"archived, the work item archived", func(w *world) { w.project.locked.Archived, w.issues.archivedAt = true, &earlier },
			domain.IssuePatch{}, domain.ErrProjectArchived, locked},
		{"an archived work item", func(w *world) { w.issues.archivedAt = &earlier }, domain.IssuePatch{}, domain.ErrIssueArchived, locked},
		{"an archived work item moved into Done", func(w *world) { w.issues.archivedAt = &earlier }, domain.IssuePatch{StateID: &done.ID},
			domain.ErrIssueArchived, locked},
````

`server/internal/modules/issue/domain/errors.go`（修改，1 处）：

````old server/internal/modules/issue/domain/errors.go
	ErrIssueNotFound = shared.NewError(shared.KindNotFound, "issue.not_found", "The work item does not exist, or you cannot see it.")
````
````new server/internal/modules/issue/domain/errors.go
	ErrIssueNotFound = shared.NewError(shared.KindNotFound, "issue.not_found", "The work item does not exist, or you cannot see it.")
	// ErrIssueArchived answers a change of an archived work item, even one
	// that changes nothing (M4 design 3.8).
	ErrIssueArchived = shared.NewError(shared.KindConflict, "issue.archived", "The work item is archived; unarchive it to change it.")
````

`server/internal/modules/project/adapter/http/state_writes_test.go`（修改，1 处）：

````old server/internal/modules/project/adapter/http/state_writes_test.go
		`"detail":"Work items are in the state, archived ones too, so it cannot be deleted."}`
````
````new server/internal/modules/project/adapter/http/state_writes_test.go
		`"detail":"Work items are in the state, archived ones too, so it cannot be deleted; unarchive the archived ones and move them all ` +
		`to another state first."}`
````

`server/internal/modules/project/domain/errors.go`（修改，2 处）：

````old server/internal/modules/project/domain/errors.go
	// archived or not, is in (M4 design 3.6).
````
````new server/internal/modules/project/domain/errors.go
	// archived or not, is in (M4 design 3.6): an archived one is moved out
	// once unarchived (3.8).
````
````old server/internal/modules/project/domain/errors.go
		"Work items are in the state, archived ones too, so it cannot be deleted.")
````
````new server/internal/modules/project/domain/errors.go
		"Work items are in the state, archived ones too, so it cannot be deleted; unarchive the archived ones and move them all to "+
			"another state first.")
````

- [ ] **Step 3: 文案、矩阵和删除标签的组合测试**

`server/internal/bootstrap/issue_cascade_test.go`（修改，2 处）：

````old server/internal/bootstrap/issue_cascade_test.go
// labelled UI. Once alice deletes Bug, the rows that put Bug and UI on
// them are deleted at Bug's moment, by her; Feature's row is kept.
````
````new server/internal/bootstrap/issue_cascade_test.go
// labelled UI; the first is archived. Once alice deletes Bug, the rows
// that put Bug and UI on them are deleted at Bug's moment, by her, the
// archived work item's too: deleting a label is a write on the project,
// not on a work item. Feature's row is kept.
````
````old server/internal/bootstrap/issue_cascade_test.go
		w.send(t, "bob", http.MethodPost, web+"/issues", `{"name":"Fix","label_ids":[`+strings.Join(quoted, ",")+`]}`, http.StatusCreated)
````
````new server/internal/bootstrap/issue_cascade_test.go
		w.send(t, "bob", http.MethodPost, web+"/issues", `{"name":"Fix","label_ids":[`+strings.Join(quoted, ",")+`]}`, http.StatusCreated)
	}
	tag, err := w.pool.Exec(pgtest.Soon(t), `UPDATE issues SET archived_at = now()
		WHERE id = (SELECT issue_id FROM issue_labels WHERE label_id = $1)`, w.bug)
	if err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("archiving the work item labelled Bug: %v, %d rows", err, tag.RowsAffected())
````

`server/internal/bootstrap/permission_matrix_issue_updates_test.go`（修改，3 处）：

````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
// 9.2's table has the issue table's columns; its variants, the project
// level's, as createIssue's (permission_matrix_issues_test.go).
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
// 9.2's table has the issue table's columns, and so has a variant on an
// archived work item, which each cell's prepare archives on its copy
// (inDone); the other variants, the project level's, as createIssue's
// (permission_matrix_issues_test.go), or the archived project's.

var cellIssueArchived = cell{http.StatusConflict, "issue.archived"}
````
````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
			request: toWorkItem(http.MethodPatch, "", `{}`), cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
			request: toWorkItem(http.MethodPatch, "", `{}`), cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		// Nor are its archived work items: the project's 409 comes first.
		{op: "updateIssue", variant: "archived, its work item archived", write: true, columns: archivedColumns, prepare: inDone(true),
			request: toWorkItem(http.MethodPatch, "", `{}`), cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		// An archived work item is not changed either (M4 design 3.8), after
		// the decision, by a change of nothing too.
		{op: "updateIssue", variant: "archived work item", write: true, columns: issueColumns, prepare: inDone(true),
			request: toWorkItem(http.MethodPatch, "", `{}`), cells: onAWorkItem(cellIssueArchived, cellIssueArchived, cellIssueArchived, cellForbidden)},
````
````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
			cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
			cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		{op: "deleteIssue", variant: "archived, its work item archived", write: true, columns: archivedColumns, prepare: inDone(true),
			request: toWorkItem(http.MethodDelete, "", ""), cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		// An archived work item is deleted as any other (M4 design 3.8).
		{op: "deleteIssue", variant: "archived work item", write: true, columns: issueColumns, prepare: inDone(true),
			request: toWorkItem(http.MethodDelete, "", ""), cells: onAWorkItem(cellNoContent, cellForbidden, cellNoContent, cellForbidden)},
````

`web/apps/web/core/lib/error-messages.ts`（修改，1 处）：

````old web/apps/web/core/lib/error-messages.ts
  "issue.not_found": "errors.issue_not_found",
````
````new web/apps/web/core/lib/error-messages.ts
  "issue.not_found": "errors.issue_not_found",
  "issue.archived": "errors.issue_archived",
````

`web/packages/i18n/src/locales/en/errors.json`（修改，2 处）：

````old web/packages/i18n/src/locales/en/errors.json
    "project_state_in_use": "Work items are in this state, archived ones too, so it cannot be deleted.",
````
````new web/packages/i18n/src/locales/en/errors.json
    "project_state_in_use": "Work items are in this state, archived ones too, so it cannot be deleted. Restore the archived ones and move them all to another state first.",
````
````old web/packages/i18n/src/locales/en/errors.json
    "issue_not_found": "The work item does not exist, or you cannot see it.",
````
````new web/packages/i18n/src/locales/en/errors.json
    "issue_not_found": "The work item does not exist, or you cannot see it.",
    "issue_archived": "The work item is archived. Restore it to change it.",
````

`web/packages/i18n/src/locales/zh-CN/errors.json`（修改，2 处）：

````old web/packages/i18n/src/locales/zh-CN/errors.json
    "project_state_in_use": "这个状态下还有工作项（包括已归档的），不能删除。",
````
````new web/packages/i18n/src/locales/zh-CN/errors.json
    "project_state_in_use": "这个状态下还有工作项（包括已归档的），不能删除。请先恢复已归档的，再把它们都移到别的状态。",
````
````old web/packages/i18n/src/locales/zh-CN/errors.json
    "issue_not_found": "工作项不存在，或者你看不到它。",
````
````new web/packages/i18n/src/locales/zh-CN/errors.json
    "issue_not_found": "工作项不存在，或者你看不到它。",
    "issue_archived": "工作项已归档，恢复之后才能修改。",
````

- [ ] **Step 4: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/... ./internal/modules/project/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix' ./internal/bootstrap/`
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
git add api/modules/issue.yaml api/modules/project.yaml server/internal/bootstrap/issue_cascade_test.go server/internal/bootstrap/permission_matrix_issue_updates_test.go server/internal/modules/issue/adapter/http/update_test.go server/internal/modules/issue/app/delete_issue_test.go server/internal/modules/issue/app/update_issue.go server/internal/modules/issue/app/update_issue_test.go server/internal/modules/issue/domain/errors.go server/internal/modules/project/adapter/http/state_writes_test.go server/internal/modules/project/domain/errors.go web/apps/web/core/lib/error-messages.ts web/packages/i18n/src/locales/en/errors.json web/packages/i18n/src/locales/zh-CN/errors.json api/dist/openapi.yaml web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M4/P2): an archived work item is read-only, and state_in_use says what to do

updateIssue refuses an archived work item under the lock, after the
archived project's 409 and before what the change changes, a change of
nothing too (409 issue.archived, with its copy); deleteIssue deletes one
as any other, which the matrix's rows on an archived work item show.
project.state_in_use now says how to free the state: restore the
archived work items in it and move them all to another state first.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p2.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试，或没有替身的平台、领域、转换的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是生成器（`make gen-go`）和中英文翻译键（`make lint-web`）的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过，端到端的检查只在标出的 Task 的树上）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T8.1` | 已归档的工作项照样修改 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `updateIssue` 各格（`make test`）、`internal/bootstrap` 的 `TestArchivingAWorkItemAndMovingItSerialize`（`make test`）（Task 12 起）、故事 `i2-update-issue`（Task 13 起）、故事 `i15-archive-issue`（Task 13 起） | 单元；组合；端到端 |
| `T8.2` | 已归档的工作项的什么都不改的修改先于它的 409 答 200 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `updateIssue` 各格（`make test`） | 单元；组合 |
| `T8.3` | 已归档的项目里已归档的工作项答 `issue.archived`，先于项目的 409 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `updateIssue` 各格（`make test`） | 单元；组合 |
| `T8.4` | 页面没有 `issue.archived` 的文案 | `error-messages.test.ts`（`make test-web`） | 单元 |
| `T8.5` | 中文的一份没有 `issue_archived` | `web/packages/i18n` 的 `check:sync`（`make lint-web`） | 守卫 |
| `T8.6` | 契约不声明 `updateIssue` 答 `issue.archived` | `internal/modules/issue/adapter/http` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `updateIssue` 各格（`make test`） | 单元；组合 |
| `T8.7` | `project.state_in_use` 不再说怎样才能删除状态 | `internal/modules/project/adapter/http` 的全部测试（`make test`） | 单元 |
| `T8.8` | 已归档的工作项不能删除：`deleteIssue` 在已归档的项目的 409 之后答 409 `issue.archived` | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `deleteIssue` 各格（`make test`） | 单元；组合 |
| `T8.9` | 删除标签只从没归档的工作项上摘下它：已归档的工作项留着已删除的标签 | `internal/bootstrap` 的 `TestDeletingALabelTakesItOffTheWorkItems`（`make test`） | 组合 |

本 Task 的检查也发现较早的 Task 的变异 `T7.21`（列在各自的 Task 的表里，标"（Task 8 起）"）：本 Task 做完之后，在本 Task 的树上重跑它们（`mutants_p2.py`，`MUT_LATER=8`）。

---

### Task 9: 日期的平台：`shared.Date`、`bodyshape`、生成器、类型映射

**Files:**
- Create: `server/internal/platform/httpserver/bodyshape/paths.go`、`server/internal/platform/httpserver/bodyshape/paths_test.go`、`server/internal/shared/date.go`、`server/internal/shared/date_test.go`
- Modify: `server/internal/modules/identity/adapter/http/gen/oapi-codegen.yaml`、`server/internal/modules/instance/adapter/http/gen/oapi-codegen.yaml`、`server/internal/modules/issue/adapter/http/gen/oapi-codegen.yaml`、`server/internal/modules/project/adapter/http/gen/oapi-codegen.yaml`、`server/internal/modules/workspace/adapter/http/gen/oapi-codegen.yaml`、`server/internal/platform/httpserver/apitest/bodycases_test.go`、`server/internal/platform/httpserver/apitest/operations.go`、`server/internal/platform/httpserver/bodyshape/ambiguity.go`、`server/internal/platform/httpserver/bodyshape/bodyshape.go`、`server/internal/platform/httpserver/bodyshape/bodyshape_test.go`、`server/internal/platform/httpserver/bodyshape/cost_test.go`、`server/tools/bodyshapegen/generate.go`、`server/tools/bodyshapegen/generate_test.go`、`server/tools/bodyshapegen/testdata/oapi-codegen.yaml`、`server/tools/bodyshapegen/testdata/things.golden`、`server/tools/bodyshapegen/testdata/things.yaml`

**Interfaces:**（spec 2.10、第 3 节第 10、12 条；M4 设计 6.1；M2 收尾交接第 4 节；M2 设计 3.11）
- `shared/date.go`（新，只用标准库）：`Date{year, month, day}`（零值不是任何一天）；`ParseDate(s string) (Date, error)`（`time.DateOnly`，年份从 0001 起；别的写法、带时间或时区、前后的空白、符号、非 ASCII 的数字都是错误）；`DateOf(t time.Time) Date`（`t` 所在时区的那一天）；`Time()`（UTC 的零点）；`String()`；`Compare(e Date) int`；`MarshalText`、`UnmarshalText`（JSON 是字符串，`null` 留下空指针）。
- `bodyshape`：`FormatDate`，`checkDate` 照 `shared.ParseDate` 的规则读（平台不能导入 `shared`；两者一致在 Task 11 由 `issue` 的 HTTP 适配器钉住）；`paths.go`（新）：`problems` 和路径的代码从 `bodyshape.go` 移来，`comparePaths`（下标按数值：`a[9].z` 在 `a[10]` 之前，全序）；`numbers()`：不限类型的节点（`Any` 的值、开放对象的未声明成员、没有 `items` 的数组的元素）里的数一遍读完，放不进 `float64` 的报 `invalid_format`；`ambiguity.go` 的扫描带上它。
- 生成器（`server/tools/bodyshapegen/generate.go`）：`checkers` 加 `shared.Date` → `FormatDate`；`testdata` 的 `things.yaml`、`oapi-codegen.yaml` 加日期的字段和映射，`things.golden` 整个重写。
- 五个模块的 `adapter/http/gen/oapi-codegen.yaml`（`identity`、`instance`、`issue`、`project`、`workspace`）：`format: date` 映射到 `shared.Date`（今天只有 `issue` 的契约会有日期，Task 11；第 3 节第 10 条）。生成物不变。
- `apitest/operations.go`：`date` 是检查的格式（`checkedFormat`），`validValue` 给它一个合法的日期。

**Tests:**
- 单元：`TestParseDate`、`TestDateCompare`、`TestDateAndTime`（UTC-5 的 23:30 是那一天）、`TestDateInJSON`；`TestFormatDate`（`TestParseDate` 的样例和一个转义）、`TestComparePaths`、`TestCheck`（`ids[2]` 在 `ids[10]` 之前，开放节点的数）、`TestCheckCostsAboutTheBody`（加三个：字符串的数组、日期、不限类型的值里的数）；`TestBodyCasesOfDates`（合法的体带一个合法的日期，别的用例都留着它，每个日期的字段各有一个写法不对的用例）。
- 生成器（`server/tools`，`make test` 的第二段）：`TestGenerateFollowsTheTypeMapping`（没有映射时 `date` 生成 `openapi_types.Date`，没有检查器：映射之外的 `date` 仍被拒绝）、`TestGenerateAcceptsTheStringsItCanCheck`、`TestGenerateRejectsWhatItCannotCheck`。

- [ ] **Step 1: `shared.Date`**

`server/internal/shared/date.go`（新文件，67 行）：

````file server/internal/shared/date.go
package shared

import (
	"cmp"
	"fmt"
	"time"
)

// Date is a day of the calendar, without a time of day or a time zone
// (M4 design 3.7, 6.1). Its text is YYYY-MM-DD, of a day from 0001-01-01
// to 9999-12-31; in JSON, a string. The zero Date is no day.
type Date struct {
	year  int
	month time.Month
	day   int
}

// ParseDate reads s, a day as YYYY-MM-DD: four digits of a year from 0001,
// two of a month, and two of a day that month has. Anything else is an
// error: another layout, a time or a zone after the day, white space
// around it. The platform's body check reads a date by the same rule
// (bodyshape's FormatDate), which may not import this package.
func ParseDate(s string) (Date, error) {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil || t.Year() < 1 {
		return Date{}, fmt.Errorf("%q is not a day as YYYY-MM-DD", s)
	}
	return DateOf(t), nil
}

// DateOf is the day t falls on, where t is: a date column, as the database
// driver reads it at midnight UTC.
func DateOf(t time.Time) Date {
	y, m, d := t.Date()
	return Date{year: y, month: m, day: d}
}

// Time is d's midnight UTC, as a date column is written.
func (d Date) Time() time.Time {
	return time.Date(d.year, d.month, d.day, 0, 0, 0, 0, time.UTC)
}

// String is d as YYYY-MM-DD.
func (d Date) String() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.year, d.month, d.day)
}

// Compare is -1 when d is before e, +1 when it is after, and 0 when they are
// the same day.
func (d Date) Compare(e Date) int {
	return cmp.Or(cmp.Compare(d.year, e.year), cmp.Compare(d.month, e.month), cmp.Compare(d.day, e.day))
}

// MarshalText writes d as YYYY-MM-DD, its JSON string.
func (d Date) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

// UnmarshalText reads text as ParseDate does.
func (d *Date) UnmarshalText(text []byte) error {
	parsed, err := ParseDate(string(text))
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}
````

`server/internal/shared/date_test.go`（新文件，92 行）：

````file server/internal/shared/date_test.go
package shared_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ParseDate reads a day as YYYY-MM-DD, of a year from 0001 to 9999, and
// nothing else (M4 design 3.7): the first and the last days, the 29th of
// February of a leap year; not one of another year, nor a month's day it
// has not, nor a month 0 or 13, nor the year 0; not one digit for two, nor
// two for four; not another separator, a time or a zone after the day,
// white space around it, a sign, digits that are not ASCII, nor nothing.
func TestParseDate(t *testing.T) {
	for _, s := range []string{"2026-10-19", "0001-01-01", "9999-12-31", "2024-02-29"} {
		d, err := shared.ParseDate(s)
		if err != nil || d.String() != s {
			t.Errorf("ParseDate(%q) = %v, %v; want the day, written back as it was", s, d, err)
		}
	}
	for _, s := range []string{"2026-02-29", "2026-04-31", "2026-00-10", "2026-13-01", "0000-12-31", "2026-1-19", "2026-10-9", "26-10-19",
		"2026/10/19", "2026-10-19T00:00:00Z", "2026-10-19+08:00", " 2026-10-19", "2026-10-19\n", "+2026-10-19", "10000-01-01",
		"２０２６-10-19", ""} {
		if d, err := shared.ParseDate(s); err == nil {
			t.Errorf("ParseDate(%q) = %v, want an error", s, d)
		}
	}
}

// A day is at midnight UTC as a date column is written, and a moment is
// the day it falls on where it is: 23:30 at UTC-5 is that day, though it
// is the next one in UTC.
func TestDateAndTime(t *testing.T) {
	d, _ := shared.ParseDate("2026-10-19")
	if got, want := d.Time(), time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC); !got.Equal(want) || got.Location() != time.UTC {
		t.Errorf("Time() = %v, want %v", got, want)
	}
	late := time.Date(2026, 10, 19, 23, 30, 0, 0, time.FixedZone("UTC-5", -5*3600))
	if got := shared.DateOf(late); got != d {
		t.Errorf("DateOf(%v) = %v, want %v", late, got, d)
	}
}

// Days compare by year, then month, then day.
func TestDateCompare(t *testing.T) {
	day := func(s string) shared.Date {
		d, err := shared.ParseDate(s)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	for _, tt := range []struct {
		a, b string
		want int
	}{
		{"2026-10-19", "2026-10-19", 0},
		{"2026-10-19", "2026-10-20", -1},
		{"2026-10-19", "2026-09-30", 1},
		{"2025-12-31", "2026-01-01", -1},
	} {
		if got := day(tt.a).Compare(day(tt.b)); got != tt.want {
			t.Errorf("%s.Compare(%s) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

// In JSON a day is its string, read back by ParseDate's rule: a day the
// month has not, or a number, is an error, and null leaves a pointer nil.
func TestDateInJSON(t *testing.T) {
	type body struct {
		On  shared.Date  `json:"on"`
		Due *shared.Date `json:"due"`
	}
	d, _ := shared.ParseDate("2026-10-19")
	out, err := json.Marshal(body{On: d, Due: &d})
	if want := `{"on":"2026-10-19","due":"2026-10-19"}`; err != nil || string(out) != want {
		t.Errorf("Marshal = %s, %v; want %s", out, err, want)
	}
	var in body
	if err := json.Unmarshal([]byte(`{"on":"2026-10-19","due":null}`), &in); err != nil || in.On != d || in.Due != nil {
		t.Errorf("Unmarshal = %+v, %v; want %v and no due", in, err, d)
	}
	for _, text := range []string{`{"on":"2026-02-30"}`, `{"on":20261019}`} {
		if err := json.Unmarshal([]byte(text), new(body)); err == nil {
			t.Errorf("Unmarshal(%s) = nil, want an error", text)
		}
	}
}
````

- [ ] **Step 2: `bodyshape`**

`server/internal/platform/httpserver/bodyshape/ambiguity.go`（修改，4 处）：

````old server/internal/platform/httpserver/bodyshape/ambiguity.go
}

// scanner reads a valid JSON value byte by byte; it does not check the
````
````new server/internal/platform/httpserver/bodyshape/ambiguity.go
}

// numbers reports to p each number in value, one valid JSON value read
// where the schema takes any value, that no float64 holds (invalid_format),
// at its JSON path (Table.walk). It reads value once, as ambiguities does,
// however deep it is; its ambiguities, ambiguities has found none of.
func numbers(value []byte, p *problems) {
	s := scanner{data: value, p: p, numbers: true}
	s.value()
}

// scanner reads a valid JSON value byte by byte; it does not check the
````
````old server/internal/platform/httpserver/bodyshape/ambiguity.go
// its bytes.
````
````new server/internal/platform/httpserver/bodyshape/ambiguity.go
// its bytes. With numbers, a number no float64 holds is reported too.
````
````old server/internal/platform/httpserver/bodyshape/ambiguity.go
	data []byte
	pos  int
	p    *problems
````
````new server/internal/platform/httpserver/bodyshape/ambiguity.go
	data    []byte
	pos     int
	p       *problems
	numbers bool
````
````old server/internal/platform/httpserver/bodyshape/ambiguity.go
	default: // a number, true, false or null
		for s.pos < len(s.data) && !isEnd(s.data[s.pos]) {
			s.pos++
````
````new server/internal/platform/httpserver/bodyshape/ambiguity.go
	default: // a number, true, false or null
		start := s.pos
		for s.pos < len(s.data) && !isEnd(s.data[s.pos]) {
			s.pos++
		}
		if s.numbers && kindOf(s.data[start:s.pos]) == 0 {
			s.p.report(codeInvalidFormat)
````

`server/internal/platform/httpserver/bodyshape/bodyshape.go`（修改，14 处）：

````old server/internal/platform/httpserver/bodyshape/bodyshape.go
	"time"
	"unicode/utf8"
````
````new server/internal/platform/httpserver/bodyshape/bodyshape.go
	"time"
````
````old server/internal/platform/httpserver/bodyshape/bodyshape.go
	FormatUUID        // the standard library's uuid.UUID (uuid)
````
````new server/internal/platform/httpserver/bodyshape/bodyshape.go
	FormatUUID        // the standard library's uuid.UUID (uuid)
	FormatDate        // shared.Date (date), read by its rule (checkDate)
````
````old server/internal/platform/httpserver/bodyshape/bodyshape.go
	// Open does not check them, or the items of an array without an item
	// schema: additionalProperties: true, or no schema.
````
````new server/internal/platform/httpserver/bodyshape/bodyshape.go
	// Open checks them, or the items of an array without an item schema,
	// only for numbers no float64 holds (numbers): additionalProperties: true,
	// or no schema.
````
````old server/internal/platform/httpserver/bodyshape/bodyshape.go
// meets. The scan reads the body in document order; the walk reads an
// object's members by name, depth first, then its missing required members.
// The problems are then sorted by path. So the same body always gets the same
// problems, but not necessarily those with the smallest paths: 16 undeclared
// members can crowd out a missing email.
````
````new server/internal/platform/httpserver/bodyshape/bodyshape.go
// meets. The scan reads the body in document order. The walk reads an
// object's members by name, depth first, then its missing required members;
// what it reads only for numbers (numbers), it reads in document order. The
// problems are then sorted by path, an index by its number (comparePaths).
// So the same body always gets the same problems, but not necessarily those
// with the smallest paths: 16 undeclared members can crowd out a missing
// email.
````
````old server/internal/platform/httpserver/bodyshape/bodyshape.go
		return cmp.Or(cmp.Compare(a.Field, b.Field), cmp.Compare(a.Code, b.Code))
````
````new server/internal/platform/httpserver/bodyshape/bodyshape.go
		return cmp.Or(comparePaths(a.Field, b.Field), cmp.Compare(a.Code, b.Code))
````
````old server/internal/platform/httpserver/bodyshape/bodyshape.go

// maxProblems bounds the problems of one body. A body can have a problem
// every few bytes: listing them all could cost the square of the body (M2
// design 3.11).
const maxProblems = 16

// maxPath bounds a reported path, in bytes. A name is the client's and can be
// as long as the body, and every problem under it repeats it: a path longer
// than this is cut short and ends with an ellipsis, so the paths of one
// answer are at most maxProblems × maxPath bytes, whatever the body (M2
// design 3.11). The contract's paths are tens of bytes.
const maxPath = 256

// ellipsis ends a path that was cut short.
const ellipsis = "…"

// problems collects the problems of one body while it is read. It keeps the
// path of the value being read as text, cut after maxPath+1 bytes, and where
// each segment began. Writing a path for every value would cost the square of
// the depth, and the whole path for every problem the problems times the
// name. This way a report copies at most maxPath bytes and compares them with
// the paths already reported, at most maxProblems of them, however long or
// deep the path, so a body can repeat one problem as often as it likes.
type problems struct {
	path  []byte // names joined by dots, indexes in brackets, e.g. tags[1].name
	marks []int  // len(path) before each segment, for leave
	shown []byte // at's buffer for a path cut short
	errs  []FieldError
}

// enter adds a member's name to the path.
func (p *problems) enter(name string) {
	p.marks = append(p.marks, len(p.path))
	if len(p.path) > 0 {
		p.path = append(p.path, '.')
	}
	p.path = append(p.path, name[:min(len(name), maxPath+1)]...) // no more than clip keeps
	p.clip()
}

// enterItem adds an item's index to the path.
func (p *problems) enterItem(i int) {
	p.marks = append(p.marks, len(p.path))
	p.path = append(strconv.AppendInt(append(p.path, '['), int64(i), 10), ']')
	p.clip()
}

// leave takes the last segment off the path.
func (p *problems) leave() {
	p.path = p.path[:p.marks[len(p.marks)-1]]
	p.marks = p.marks[:len(p.marks)-1]
}

// clip keeps the first maxPath+1 bytes of the path: one more than a reported
// path has, which tells at that the path is longer.
func (p *problems) clip() { p.path = p.path[:min(len(p.path), maxPath+1)] }

// full reports whether maxProblems have been reported: the rest is not read.
func (p *problems) full() bool { return len(p.errs) >= maxProblems }

// report adds a problem at the current path, unless p is full or has it
// already. A problem is reported once: a name repeated in its object can
// break the same rule each time, and two long paths can be cut short to the
// same text, which a client could not tell apart anyway. A repeat does not
// count against maxProblems.
func (p *problems) report(code string) {
	if p.full() {
		return
	}
	path := p.at()
	for _, f := range p.errs {
		if f.Field == string(path) && f.Code == code {
			return
		}
	}
	p.errs = append(p.errs, FieldError{string(path), code})
}

// at returns the current path as it is reported; the body itself is the
// empty path. A path longer than maxPath is cut short: its first bytes, cut
// at a rune boundary, then an ellipsis, maxPath bytes in all at most. The
// path's names are decoded, so it is UTF-8 up to the cut. The result is valid
// until the path changes.
func (p *problems) at() []byte {
	if len(p.path) <= maxPath {
		return p.path
	}
	cut := maxPath - len(ellipsis)
	for !utf8.RuneStart(p.path[cut]) {
		cut--
	}
	p.shown = append(append(p.shown[:0], p.path[:cut]...), ellipsis...)
	return p.shown
}

````
````new server/internal/platform/httpserver/bodyshape/bodyshape.go

````
````old server/internal/platform/httpserver/bodyshape/bodyshape.go
// goes only where the schema goes: declared properties, the values of an
// open map, the items of an array with an item schema. It reads the members
// of an object in name order, so a body with more than maxProblems problems
// gets the same ones every time.
````
````new server/internal/platform/httpserver/bodyshape/bodyshape.go
// goes where the schema goes: declared properties, the values of a map,
// the items of an array with an item schema. Where the schema takes any
// value, it checks only that each number fits a float64 (M2 handoff 4): a
// value of a node of any type, an undeclared member of an open object, an
// item of an array without an item schema, read once however deep
// (numbers). Where the generated decoder keeps such a value, as for a
// schema without a type, additionalProperties: true or an array without an
// item schema, it reads every number in it into a float64. walk reads the
// members of an object in name order, and numbers reads a value in document
// order, so a body with more than maxProblems problems gets the same ones
// every time.
````
````old server/internal/platform/httpserver/bodyshape/bodyshape.go
	if n.Types != Any && kind&n.Types == 0 {
````
````new server/internal/platform/httpserver/bodyshape/bodyshape.go
	if kind == 0 || n.Types != Any && kind&n.Types == 0 {
````
````old server/internal/platform/httpserver/bodyshape/bodyshape.go
				t.walk(n.Extra, props[name], p)
````
````new server/internal/platform/httpserver/bodyshape/bodyshape.go
				t.walk(n.Extra, props[name], p)
			default:
				numbers(props[name], p)
````
````old server/internal/platform/httpserver/bodyshape/bodyshape.go
	case Array:
		if n.Items < 0 {
			return
		}
````
````new server/internal/platform/httpserver/bodyshape/bodyshape.go
	case Array:
````
````old server/internal/platform/httpserver/bodyshape/bodyshape.go
			t.walk(n.Items, item, p)
````
````new server/internal/platform/httpserver/bodyshape/bodyshape.go
			if n.Items >= 0 {
				t.walk(n.Items, item, p)
			} else {
				numbers(item, p)
			}
````
````old server/internal/platform/httpserver/bodyshape/bodyshape.go
// the two accept the same values by construction (M2 design 3.11).
````
````new server/internal/platform/httpserver/bodyshape/bodyshape.go
// the two accept the same values by construction (M2 design 3.11). A date's
// Go type, shared.Date, is the business's, which the platform does not
// import: its string is read by its rule (checkDate).
````
````old server/internal/platform/httpserver/bodyshape/bodyshape.go
		return json.Unmarshal(raw, new(uuid.UUID))
````
````new server/internal/platform/httpserver/bodyshape/bodyshape.go
		return json.Unmarshal(raw, new(uuid.UUID))
	case FormatDate:
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return err
		}
		return checkDate(s)
````
````old server/internal/platform/httpserver/bodyshape/bodyshape.go
	return fmt.Errorf("bodyshape: unknown format %d", f)
}

````
````new server/internal/platform/httpserver/bodyshape/bodyshape.go
	return fmt.Errorf("bodyshape: unknown format %d", f)
}

// checkDate reads s by shared.ParseDate's rule (M4 design 6.1): a day as
// YYYY-MM-DD, of a year from 0001.
func checkDate(s string) error {
	t, err := time.Parse(time.DateOnly, s)
	if err == nil && t.Year() < 1 {
		err = fmt.Errorf("%q is before the year 1", s)
	}
	return err
}

````

`server/internal/platform/httpserver/bodyshape/bodyshape_test.go`（修改，7 处）：

````old server/internal/platform/httpserver/bodyshape/bodyshape_test.go
// nullable scalars, nesting, an array of objects, an open map, formats.
````
````new server/internal/platform/httpserver/bodyshape/bodyshape_test.go
// nullable scalars, nesting, an array of objects and one of strings, a
// map, formats; and what takes any value: a schema without a type, an
// open object, an array without an item schema.
````
````old server/internal/platform/httpserver/bodyshape/bodyshape_test.go
				Props: map[string]int{"count": 2, "id": 4, "labels": 10, "maybe": 5, "name": 1, "nested": 6, "note": 11, "tags": 8, "when": 3}},
````
````new server/internal/platform/httpserver/bodyshape/bodyshape_test.go
				Props: map[string]int{"anything": 13, "count": 2, "free": 14, "id": 4, "ids": 15, "labels": 10, "list": 17, "maybe": 5, "name": 1,
					"nested": 6, "note": 11, "on": 12, "tags": 8, "when": 3}},
````
````old server/internal/platform/httpserver/bodyshape/bodyshape_test.go
			/* 11 */ {Types: Number, Extra: Open, Items: Open},
````
````new server/internal/platform/httpserver/bodyshape/bodyshape_test.go
			/* 11 */ {Types: Number, Extra: Open, Items: Open},
			/* 12 */ {Types: String | Null, Format: FormatDate, Extra: Open, Items: Open},
			/* 13 */ {Types: Any, Extra: Open, Items: Open},
			/* 14 */ {Types: Object, Extra: Open, Items: Open, Props: map[string]int{}},
			/* 15 */ {Types: Array, Extra: Open, Items: 16},
			/* 16 */ {Types: String, Format: FormatUUID, Extra: Open, Items: Open},
			/* 17 */ {Types: Array, Extra: Open, Items: Open},
````
````old server/internal/platform/httpserver/bodyshape/bodyshape_test.go
func TestCheck(t *testing.T) {
````
````new server/internal/platform/httpserver/bodyshape/bodyshape_test.go
func TestCheck(t *testing.T) {
	const id = `"0199a2b4-7c3e-7d2a-9f10-2b3c4d5e6f70"`
	ids := func(wrong ...int) string {
		items := slices.Repeat([]string{id}, 12)
		for _, i := range wrong {
			items[i] = `"x"`
		}
		return `{"name":"a","nested":{"a":"y"},"ids":[` + strings.Join(items, ",") + `]}`
	}
````
````old server/internal/platform/httpserver/bodyshape/bodyshape_test.go
			`"maybe":{"a":"x"},"nested":{"a":"y","b":2},"tags":[{"name":"t"}],"labels":{"k":"v"},"note":1.5}`, nil},
````
````new server/internal/platform/httpserver/bodyshape/bodyshape_test.go
			`"maybe":{"a":"x"},"nested":{"a":"y","b":2},"tags":[{"name":"t"}],"labels":{"k":"v"},"note":1.5,"on":"2026-10-19",` +
			`"ids":[` + id + `],"anything":{"a":[1.5e300,"x"]},"free":{"k":-7},"list":[1e-400,true]}`, nil},
````
````old server/internal/platform/httpserver/bodyshape/bodyshape_test.go
			[]FieldError{{"id", "invalid_format"}}},
````
````new server/internal/platform/httpserver/bodyshape/bodyshape_test.go
			[]FieldError{{"id", "invalid_format"}}},
		{"null for a nullable date", `{"name":"a","nested":{"a":"y"},"on":null}`, nil},
		{"a day the month has not", `{"name":"a","nested":{"a":"y"},"on":"2026-02-30"}`,
			[]FieldError{{"on", "invalid_format"}}},
		{"a date-time is not a date", `{"name":"a","nested":{"a":"y"},"on":"2026-10-19T00:00:00Z"}`,
			[]FieldError{{"on", "invalid_format"}}},
		{"a wrong item of an array of strings", ids(1), []FieldError{{"ids[1]", "invalid_format"}}},
		{"indexes in the order of their numbers", ids(10, 2), []FieldError{{"ids[2]", "invalid_format"}, {"ids[10]", "invalid_format"}}},
		{"a number no float64 holds where any value goes", `{"name":"a","nested":{"a":"y"},"anything":1e400}`,
			[]FieldError{{"anything", "invalid_format"}}},
		{"deep in a value of any type", `{"name":"a","nested":{"a":"y"},"anything":{"a":[1,{"b":-1e400}]}}`,
			[]FieldError{{"anything.a[1].b", "invalid_format"}}},
		{"in an undeclared member of an open object", `{"name":"a","nested":{"a":"y"},"free":{"j":1,"k":1e400}}`,
			[]FieldError{{"free.k", "invalid_format"}}},
		{"in an item of an array without an item schema", `{"name":"a","nested":{"a":"y"},"list":[1,"x",[1e400]]}`,
			[]FieldError{{"list[2][0]", "invalid_format"}}},
````
````old server/internal/platform/httpserver/bodyshape/bodyshape_test.go
}

// A number passes the check exactly when the generated decoder can decode it
````
````new server/internal/platform/httpserver/bodyshape/bodyshape_test.go
}

// A date passes the check exactly when shared.ParseDate reads it, which
// this package may not import: the samples of its test (shared's
// TestParseDate), and an escape, which the decoder reads first.
func TestFormatDate(t *testing.T) {
	for raw, want := range map[string]bool{
		`"2026-10-19"`: true, `"0001-01-01"`: true, `"9999-12-31"`: true, `"2024-02-29"`: true, `"2026-10-1\u0039"`: true,
		`"2026-02-29"`: false, `"2026-04-31"`: false, `"2026-00-10"`: false, `"2026-13-01"`: false, `"0000-12-31"`: false,
		`"2026-1-19"`: false, `"2026-10-9"`: false, `"26-10-19"`: false, `"2026/10/19"`: false, `"2026-10-19T00:00:00Z"`: false,
		`"2026-10-19+08:00"`: false, `" 2026-10-19"`: false, `"2026-10-19\n"`: false, `"+2026-10-19"`: false, `"10000-01-01"`: false,
		`"２０２６-10-19"`: false, `""`: false,
	} {
		if got := checkFormat(FormatDate, []byte(raw)) == nil; got != want {
			t.Errorf("checkFormat(FormatDate, %s) = %v, want %v", raw, got, want)
		}
	}
}

// A number passes the check exactly when the generated decoder can decode it
````

`server/internal/platform/httpserver/bodyshape/cost_test.go`（修改，3 处）：

````old server/internal/platform/httpserver/bodyshape/cost_test.go
// answers 16 copies of a path of about 1 MiB. The last repeats one problem,
// which is reported once and not counted, so each repeat must cost little
// (M2 design 3.11). Not parallel: TotalAlloc counts the whole process.
````
````new server/internal/platform/httpserver/bodyshape/cost_test.go
// answers 16 copies of a path of about 1 MiB. The eighth repeats one
// problem, which is reported once and not counted, so each repeat must cost
// little (M2 design 3.11). The last three are M4's (M2 handoff 4): an array
// of strings, a date, and a number in a value of any type. The number is
// under 9,990 arrays: a check that read the value once a level would cost
// the square of the depth. Not parallel: TotalAlloc counts the whole
// process.
````
````old server/internal/platform/httpserver/bodyshape/cost_test.go
	repeated := `{"` + angles[:300] + `":[` + strings.Repeat("\"\xff\",", 200000) + "\"\xff\"]}"
````
````new server/internal/platform/httpserver/bodyshape/cost_test.go
	repeated := `{"` + angles[:300] + `":[` + strings.Repeat("\"\xff\",", 200000) + "\"\xff\"]}"
	// 200,000 strings that are not uuids, of which 16 are reported and the
	// rest cost little; a wrong date of 1 MiB, read once; a number no float64
	// holds in a value of any type, under 9,990 arrays.
	notUUIDs := `{"name":"a","nested":{"a":"y"},"ids":[` + strings.Repeat(`"x",`, 199999) + `"x"]}`
	longDate := `{"name":"a","nested":{"a":"y"},"on":"` + strings.Repeat("9", 1<<20) + `"}`
	deepNumber := `{"name":"a","nested":{"a":"y"},"anything":` + strings.Repeat("[", 9990) + "1e400" + strings.Repeat("]", 9990) + "}"
````
````old server/internal/platform/httpserver/bodyshape/cost_test.go
		{"a string that is not UTF-8, 200,000 times under a name of 300 bytes", things(), repeated, 6 * len(repeated)},
````
````new server/internal/platform/httpserver/bodyshape/cost_test.go
		{"a string that is not UTF-8, 200,000 times under a name of 300 bytes", things(), repeated, 6 * len(repeated)},
		// 64 MiB as for the first four, 6 times the body as for the next four.
		{"200,000 strings that are not uuids", things(), notUUIDs, 64 << 20},
		{"a wrong date of 1 MiB", things(), longDate, 6 * len(longDate)},
		{"a number no float64 holds under 9,990 arrays of any type", things(), deepNumber, 64 << 20},
````

`server/internal/platform/httpserver/bodyshape/paths.go`（新文件，138 行）：

````file server/internal/platform/httpserver/bodyshape/paths.go
package bodyshape

import (
	"cmp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// The paths of the problems: kept while a body is read, cut short when
// they are long, each problem reported once, and the problems put in order.

// maxProblems bounds the problems of one body. A body can have a problem
// every few bytes: listing them all could cost the square of the body (M2
// design 3.11).
const maxProblems = 16

// maxPath bounds a reported path, in bytes. A name is the client's and can be
// as long as the body, and every problem under it repeats it: a path longer
// than this is cut short and ends with an ellipsis, so the paths of one
// answer are at most maxProblems × maxPath bytes, whatever the body (M2
// design 3.11). The contract's paths are tens of bytes.
const maxPath = 256

// ellipsis ends a path that was cut short.
const ellipsis = "…"

// problems collects the problems of one body while it is read. It keeps the
// path of the value being read as text, cut after maxPath+1 bytes, and where
// each segment began. Writing a path for every value would cost the square of
// the depth, and the whole path for every problem the problems times the
// name. This way a report copies at most maxPath bytes and compares them with
// the paths already reported, at most maxProblems of them, however long or
// deep the path, so a body can repeat one problem as often as it likes.
type problems struct {
	path  []byte // names joined by dots, indexes in brackets, e.g. tags[1].name
	marks []int  // len(path) before each segment, for leave
	shown []byte // at's buffer for a path cut short
	errs  []FieldError
}

// enter adds a member's name to the path.
func (p *problems) enter(name string) {
	p.marks = append(p.marks, len(p.path))
	if len(p.path) > 0 {
		p.path = append(p.path, '.')
	}
	p.path = append(p.path, name[:min(len(name), maxPath+1)]...) // no more than clip keeps
	p.clip()
}

// enterItem adds an item's index to the path.
func (p *problems) enterItem(i int) {
	p.marks = append(p.marks, len(p.path))
	p.path = append(strconv.AppendInt(append(p.path, '['), int64(i), 10), ']')
	p.clip()
}

// leave takes the last segment off the path.
func (p *problems) leave() {
	p.path = p.path[:p.marks[len(p.marks)-1]]
	p.marks = p.marks[:len(p.marks)-1]
}

// clip keeps the first maxPath+1 bytes of the path: one more than a reported
// path has, which tells at that the path is longer.
func (p *problems) clip() { p.path = p.path[:min(len(p.path), maxPath+1)] }

// full reports whether maxProblems have been reported: the rest is not read.
func (p *problems) full() bool { return len(p.errs) >= maxProblems }

// report adds a problem at the current path, unless p is full or has it
// already. A problem is reported once: a name repeated in its object can
// break the same rule each time, and two long paths can be cut short to the
// same text, which a client could not tell apart anyway. A repeat does not
// count against maxProblems.
func (p *problems) report(code string) {
	if p.full() {
		return
	}
	path := p.at()
	for _, f := range p.errs {
		if f.Field == string(path) && f.Code == code {
			return
		}
	}
	p.errs = append(p.errs, FieldError{string(path), code})
}

// at returns the current path as it is reported; the body itself is the
// empty path. A path longer than maxPath is cut short: its first bytes, cut
// at a rune boundary, then an ellipsis, maxPath bytes in all at most. The
// path's names are decoded, so it is UTF-8 up to the cut. The result is valid
// until the path changes.
func (p *problems) at() []byte {
	if len(p.path) <= maxPath {
		return p.path
	}
	cut := maxPath - len(ellipsis)
	for !utf8.RuneStart(p.path[cut]) {
		cut--
	}
	p.shown = append(append(p.shown[:0], p.path[:cut]...), ellipsis...)
	return p.shown
}

// comparePaths orders two reported paths as a reader counts: byte by byte,
// but an index against an index by its number, so tags[2] comes before
// tags[10] (M2 handoff 4). Each '[' starts an index, the digits after it:
// a member name with a '[' and digits in it is ordered the same way, which
// keeps the order total.
func comparePaths(a, b string) int {
	for a != "" && b != "" {
		if a[0] == '[' && b[0] == '[' {
			var x, y string
			x, a = cutDigits(a[1:])
			y, b = cutDigits(b[1:])
			if c := cmp.Or(cmp.Compare(len(x), len(y)), strings.Compare(x, y)); c != 0 {
				return c
			}
			continue
		}
		if a[0] != b[0] {
			return cmp.Compare(a[0], b[0])
		}
		a, b = a[1:], b[1:]
	}
	return cmp.Compare(len(a), len(b))
}

// cutDigits is the digits s starts with, and the rest of s.
func cutDigits(s string) (digits, rest string) {
	i := 0
	for i < len(s) && '0' <= s[i] && s[i] <= '9' {
		i++
	}
	return s[:i], s[i:]
}
````

`server/internal/platform/httpserver/bodyshape/paths_test.go`（新文件，31 行）：

````file server/internal/platform/httpserver/bodyshape/paths_test.go
package bodyshape

import (
	"slices"
	"testing"
)

// Paths sort byte by byte, but an index against an index by its number
// (M2 handoff 4): a[1].z before a[2].a, though the rest of the path would
// order them the other way, and a[9].z before a[10], before a[10].name; a
// member before an index of the same value ('.' before '['), the shorter
// path first; a name with a '[' and digits in it is ordered as an index
// is. The order is the same from any order of the paths.
func TestComparePaths(t *testing.T) {
	want := []string{"", "[2]", "[10]", "a", "a.b", "a[0]", "a[1].z", "a[2].a", "a[9].z", "a[10]", "a[10].name", "a[10][3]", "a[10][20]",
		"k[9", "k[10", "z"}
	reversed := slices.Clone(want)
	slices.Reverse(reversed)
	for _, from := range [][]string{slices.Clone(want), reversed, {"a[10]", "a[2].a", "z", "k[10", "", "a[0]", "[10]", "a.b", "a[10][20]",
		"a[9].z", "k[9", "a", "a[1].z", "a[10][3]", "[2]", "a[10].name"}} {
		got := slices.SortedFunc(slices.Values(from), comparePaths)
		if !slices.Equal(got, want) {
			t.Errorf("sorted from %q = %q, want %q", from, got, want)
		}
	}
	for _, p := range want {
		if c := comparePaths(p, p); c != 0 {
			t.Errorf("comparePaths(%q, %q) = %d, want 0", p, p, c)
		}
	}
}
````

- [ ] **Step 3: 生成器和类型映射**

`server/internal/modules/identity/adapter/http/gen/oapi-codegen.yaml`（修改，1 处）：

````old server/internal/modules/identity/adapter/http/gen/oapi-codegen.yaml
          type: string
````
````new server/internal/modules/identity/adapter/http/gen/oapi-codegen.yaml
          type: string
        # 日期用 shared.Date（M4 设计 6.1）：默认的 openapi_types.Date 所在的包带进 github.com/google/uuid（M2 交接第 4 节），bodyshape 也查不了它
        date:
          type: shared.Date
          import: github.com/open-nerve/NerveProject/server/internal/shared
````

`server/internal/modules/instance/adapter/http/gen/oapi-codegen.yaml`（修改，1 处）：

````old server/internal/modules/instance/adapter/http/gen/oapi-codegen.yaml
          type: string
````
````new server/internal/modules/instance/adapter/http/gen/oapi-codegen.yaml
          type: string
        # 日期用 shared.Date（M4 设计 6.1）：默认的 openapi_types.Date 所在的包带进 github.com/google/uuid（M2 交接第 4 节），bodyshape 也查不了它
        date:
          type: shared.Date
          import: github.com/open-nerve/NerveProject/server/internal/shared
````

`server/internal/modules/issue/adapter/http/gen/oapi-codegen.yaml`（修改，1 处）：

````old server/internal/modules/issue/adapter/http/gen/oapi-codegen.yaml
          type: string
````
````new server/internal/modules/issue/adapter/http/gen/oapi-codegen.yaml
          type: string
        # 日期用 shared.Date（M4 设计 6.1）：默认的 openapi_types.Date 所在的包带进 github.com/google/uuid（M2 交接第 4 节），bodyshape 也查不了它
        date:
          type: shared.Date
          import: github.com/open-nerve/NerveProject/server/internal/shared
````

`server/internal/modules/project/adapter/http/gen/oapi-codegen.yaml`（修改，1 处）：

````old server/internal/modules/project/adapter/http/gen/oapi-codegen.yaml
          type: string
````
````new server/internal/modules/project/adapter/http/gen/oapi-codegen.yaml
          type: string
        # 日期用 shared.Date（M4 设计 6.1）：默认的 openapi_types.Date 所在的包带进 github.com/google/uuid（M2 交接第 4 节），bodyshape 也查不了它
        date:
          type: shared.Date
          import: github.com/open-nerve/NerveProject/server/internal/shared
````

`server/internal/modules/workspace/adapter/http/gen/oapi-codegen.yaml`（修改，1 处）：

````old server/internal/modules/workspace/adapter/http/gen/oapi-codegen.yaml
          type: string
````
````new server/internal/modules/workspace/adapter/http/gen/oapi-codegen.yaml
          type: string
        # 日期用 shared.Date（M4 设计 6.1）：默认的 openapi_types.Date 所在的包带进 github.com/google/uuid（M2 交接第 4 节），bodyshape 也查不了它
        date:
          type: shared.Date
          import: github.com/open-nerve/NerveProject/server/internal/shared
````

`server/tools/bodyshapegen/generate.go`（修改，1 处）：

````old server/tools/bodyshapegen/generate.go
	{Type: "uuid.UUID", Import: "uuid"}: "FormatUUID",
````
````new server/tools/bodyshapegen/generate.go
	{Type: "uuid.UUID", Import: "uuid"}: "FormatUUID",
	{Type: "shared.Date", Import: "github.com/open-nerve/NerveProject/server/internal/shared"}: "FormatDate",
````

`server/tools/bodyshapegen/generate_test.go`（修改，6 处）：

````old server/tools/bodyshapegen/generate_test.go
// type-mapping (uuid checked, email left to the domain, numbers as int, int64
// and float64), a reference into another file, and one root per operation
// with a JSON body.
````
````new server/tools/bodyshapegen/generate_test.go
// type-mapping (uuid checked, date checked as shared.Date, email left to the
// domain, numbers as int, int64 and float64), a reference into another file,
// and one root per operation with a JSON body.
````
````old server/tools/bodyshapegen/generate_test.go
// Without the module template's type-mapping, uuid is generated as
// openapi_types.UUID, which has no checker: the table follows the mapping.
````
````new server/tools/bodyshapegen/generate_test.go
// Without the module template's type-mapping, email is generated as
// openapi_types.Email and date as openapi_types.Date, which have no
// checker: the table follows the mapping.
````
````old server/tools/bodyshapegen/generate_test.go

	_, err := generate(thingsSpec, conf)

	if err == nil || !strings.Contains(err.Error(), `format "email" is generated as openapi_types.Email`) {
		t.Errorf("generate() = %v, want the default email mapping rejected", err)
````
````new server/tools/bodyshapegen/generate_test.go
	for spec, want := range map[string]string{
		thingsSpec: `format "email" is generated as openapi_types.Email`,
		bodySpec(t, "{type: string, format: date}"): `due: format "date" is generated as openapi_types.Date, which has no bodyshape checker`,
	} {
		if _, err := generate(spec, conf); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("generate(%s) = %v, want an error containing %q", spec, err, want)
		}
````
````old server/tools/bodyshapegen/generate_test.go
	tests := []struct{ name, property, want string }{
		{"date", "{type: string, format: date}", `due: format "date" is generated as openapi_types.Date`},
````
````new server/tools/bodyshapegen/generate_test.go
	tests := []struct{ name, property, want string }{
````
````old server/tools/bodyshapegen/generate_test.go
		{"nested", "{type: object, properties: {at: {type: array, items: {type: string, format: date}}}}", `due.at[]: format "date"`},
````
````new server/tools/bodyshapegen/generate_test.go
		{"nested", "{type: object, properties: {at: {type: array, items: {type: string, format: byte}}}}", `due.at[]: format "byte"`},
````
````old server/tools/bodyshapegen/generate_test.go
		"{type: string, format: date-time, contentMediaType: text/plain}",
````
````new server/tools/bodyshapegen/generate_test.go
		"{type: string, format: date-time, contentMediaType: text/plain}",
		"{type: [string, 'null'], format: date}",
````

`server/tools/bodyshapegen/testdata/oapi-codegen.yaml`（修改，1 处）：

````old server/tools/bodyshapegen/testdata/oapi-codegen.yaml
          type: string
````
````new server/tools/bodyshapegen/testdata/oapi-codegen.yaml
          type: string
        date:
          type: shared.Date
          import: github.com/open-nerve/NerveProject/server/internal/shared
````

`server/tools/bodyshapegen/testdata/things.golden`（整个文件，44 行）：

````whole server/tools/bodyshapegen/testdata/things.golden
// Code generated by bodyshapegen from things.yaml. DO NOT EDIT.

package gen

import "github.com/open-nerve/NerveProject/server/internal/platform/httpserver/bodyshape"

// BodyShapes returns the structure of every JSON request body of this module,
// by route pattern. The platform's bodyshape middleware checks each request
// body against it before the strict handler decodes it.
func BodyShapes() *bodyshape.Table {
	return &bodyshape.Table{
		Nodes: []bodyshape.Node{
			/* 0 */ {Types: bodyshape.Object, Extra: bodyshape.Closed, Items: bodyshape.Open, Props: map[string]int{"anything": 1, "count": 2, "due": 3, "email": 4, "extra": 5, "ids": 6, "labels": 8, "name": 10, "ratio": 11, "settings": 12, "starts": 14, "tags": 15, "total": 19, "weight": 20}, Required: []string{"email", "name", "settings"}},
			/* 1 */ {Types: bodyshape.Any, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 2 */ {Types: bodyshape.Integer, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 3 */ {Types: bodyshape.String, Format: bodyshape.FormatTime, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 4 */ {Types: bodyshape.String, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 5 */ {Types: bodyshape.Object, Extra: bodyshape.Open, Items: bodyshape.Open, Props: map[string]int{}},
			/* 6 */ {Types: bodyshape.Array, Extra: bodyshape.Open, Items: 7},
			/* 7 */ {Types: bodyshape.String, Format: bodyshape.FormatUUID, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 8 */ {Types: bodyshape.Object, Extra: 9, Items: bodyshape.Open, Props: map[string]int{}},
			/* 9 */ {Types: bodyshape.String, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 10 */ {Types: bodyshape.String, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 11 */ {Types: bodyshape.Number, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 12 */ {Types: bodyshape.Object, Extra: bodyshape.Closed, Items: bodyshape.Open, Props: map[string]int{"notify": 13}},
			/* 13 */ {Types: bodyshape.Boolean, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 14 */ {Types: bodyshape.Null | bodyshape.String, Format: bodyshape.FormatDate, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 15 */ {Types: bodyshape.Array, Extra: bodyshape.Open, Items: 16},
			/* 16 */ {Types: bodyshape.Object, Extra: bodyshape.Closed, Items: bodyshape.Open, Props: map[string]int{"color": 17, "name": 18}, Required: []string{"name"}},
			/* 17 */ {Types: bodyshape.Null | bodyshape.String, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 18 */ {Types: bodyshape.String, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 19 */ {Types: bodyshape.Integer, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 20 */ {Types: bodyshape.Number, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 21 */ {Types: bodyshape.Object, Extra: bodyshape.Closed, Items: bodyshape.Open, Props: map[string]int{"name": 22, "owner_id": 23, "settings": 24}},
			/* 22 */ {Types: bodyshape.String, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 23 */ {Types: bodyshape.Null | bodyshape.String, Format: bodyshape.FormatUUID, Extra: bodyshape.Open, Items: bodyshape.Open},
			/* 24 */ {Types: bodyshape.Null | bodyshape.Object, Extra: bodyshape.Closed, Items: bodyshape.Open, Props: map[string]int{"notify": 13}},
		},
		Roots: map[string]int{
			"PATCH /api/v0/things/{thing_id}": 21,
			"POST /api/v0/things":             0,
		},
	}
}
````

`server/tools/bodyshapegen/testdata/things.yaml`（修改，2 处）：

````old server/tools/bodyshapegen/testdata/things.yaml
          additionalProperties: true
````
````new server/tools/bodyshapegen/testdata/things.yaml
          additionalProperties: true
        ids:
          type: array
          items:
            type: string
            format: uuid
````
````old server/tools/bodyshapegen/testdata/things.yaml
          $ref: '#/components/schemas/Settings'
````
````new server/tools/bodyshapegen/testdata/things.yaml
          $ref: '#/components/schemas/Settings'
        starts:
          type: [string, 'null']
          format: date
````

Run: `make gen`
Expected: 成功；生成的文件一个都不变（还没有契约用 `format: date`，映射和检查器从 Task 11 起才用到）。

- [ ] **Step 4: `apitest` 的 `date`**

`server/internal/platform/httpserver/apitest/bodycases_test.go`（修改，2 处）：

````old server/internal/platform/httpserver/apitest/bodycases_test.go
import (
````
````new server/internal/platform/httpserver/apitest/bodycases_test.go
import (
	"fmt"
````
````old server/internal/platform/httpserver/apitest/bodycases_test.go
		t.Errorf("every problem at once expects %v, want %v", last.Fields, wantFields)
````
````new server/internal/platform/httpserver/apitest/bodycases_test.go
		t.Errorf("every problem at once expects %v, want %v", last.Fields, wantFields)
	}
}

// A date is a checked format (M4 design 6.1): the valid body has a valid
// one, which every case but its own keeps, and each date gets a wrong one.
func TestBodyCasesOfDates(t *testing.T) {
	op := bodyOperation(t, "{type: object, required: [on], properties: {on: {type: string, format: date}, "+
		"until: {type: [string, 'null'], format: date}}}")

	var got []string
	for _, c := range op.BodyCases() {
		got = append(got, c.Name+" "+string(c.Body)+" "+fmt.Sprint(c.Fields))
	}
	want := []string{
		`undeclared property {"nerve_undeclared":1,"on":"2026-01-01"} [{nerve_undeclared not_allowed}]`,
		`null for nullable until {"on":"2026-01-01","until":null} []`,
		`missing on {} [{on required}]`,
		`wrong date in on {"on":"not-a-date"} [{on invalid_format}]`,
		`wrong date in until {"on":"2026-01-01","until":"not-a-date"} [{until invalid_format}]`,
		`on twice {"on":"2026-01-01","on":"2026-01-01"} [{on duplicate}]`,
		"not UTF-8 in on {\"on\":\"\xff\"} [{on invalid_format}]",
		`every problem at once {"nerve_undeclared":1,"until":"not-a-date"} [{nerve_undeclared not_allowed} {on required} ` +
			`{until invalid_format}]`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("BodyCases() =\n%q\nwant\n%q", got, want)
````

`server/internal/platform/httpserver/apitest/operations.go`（修改，3 处）：

````old server/internal/platform/httpserver/apitest/operations.go
		return "2026-01-01T00:00:00Z"
````
````new server/internal/platform/httpserver/apitest/operations.go
		return "2026-01-01T00:00:00Z"
	case "date":
		return "2026-01-01"
````
````old server/internal/platform/httpserver/apitest/operations.go
// module template generates as Go types (M2 design 3.12).
````
````new server/internal/platform/httpserver/apitest/operations.go
// module template generates as Go types (M2 design 3.12; date, M4 design
// 6.1).
````
````old server/internal/platform/httpserver/apitest/operations.go
	return s.Type.Includes("string") && (s.Format == "date-time" || s.Format == "uuid")
````
````new server/internal/platform/httpserver/apitest/operations.go
	return s.Type.Includes("string") && (s.Format == "date-time" || s.Format == "date" || s.Format == "uuid")
````

- [ ] **Step 5: 测试和 lint**

Run: `go -C server test -count=1 ./internal/shared/... ./internal/platform/httpserver/...`
Expected: 全部 `ok`。

Run: `go -C server/tools test -count=1 ./bodyshapegen/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 6: 提交**

```bash
git add server/internal/modules/identity/adapter/http/gen/oapi-codegen.yaml server/internal/modules/instance/adapter/http/gen/oapi-codegen.yaml server/internal/modules/issue/adapter/http/gen/oapi-codegen.yaml server/internal/modules/project/adapter/http/gen/oapi-codegen.yaml server/internal/modules/workspace/adapter/http/gen/oapi-codegen.yaml server/internal/platform/httpserver/apitest/bodycases_test.go server/internal/platform/httpserver/apitest/operations.go server/internal/platform/httpserver/bodyshape/ambiguity.go server/internal/platform/httpserver/bodyshape/bodyshape.go server/internal/platform/httpserver/bodyshape/bodyshape_test.go server/internal/platform/httpserver/bodyshape/cost_test.go server/internal/platform/httpserver/bodyshape/paths.go server/internal/platform/httpserver/bodyshape/paths_test.go server/internal/shared/date.go server/internal/shared/date_test.go server/tools/bodyshapegen/generate.go server/tools/bodyshapegen/generate_test.go server/tools/bodyshapegen/testdata/oapi-codegen.yaml server/tools/bodyshapegen/testdata/things.golden server/tools/bodyshapegen/testdata/things.yaml
```
```bash
git commit -m "feat(M4/P2): shared.Date, the date format in bodyshape and its generator

shared.Date is a day: read as YYYY-MM-DD from the year 0001 and nothing
else, its JSON its string, midnight UTC as a time. bodyshape checks the
date format by the same rule, which it may not import; sorts its
problems' paths with indexes by number; and checks the numbers in values
of any type in one pass. The generator maps shared.Date to FormatDate,
and the five modules' templates map format: date to shared.Date, so that
no date is generated as openapi_types.Date. apitest counts date as a
checked format. Nothing generated changes yet.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p2.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试，或没有替身的平台、领域、转换的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是生成器（`make gen-go`）和中英文翻译键（`make lint-web`）的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过，端到端的检查只在标出的 Task 的树上）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T9.1` | `shared.ParseDate` 读年份 0 | `internal/shared` 的全部测试（`make test`） | 单元 |
| `T9.2` | 请求体检查的日期读年份 0，`shared.ParseDate` 不读 | `internal/platform/httpserver/bodyshape` 的全部测试（`make test`）、`internal/modules/issue/adapter/http` 的全部测试（`make test`）（Task 11 起） | 单元 |
| `T9.3` | 请求体检查按日期的原始字节读，不先解开转义 | `internal/platform/httpserver/bodyshape` 的全部测试（`make test`）、`internal/modules/issue/adapter/http` 的全部测试（`make test`）（Task 11 起） | 单元 |
| `T9.4` | `shared.Date` 只比较年和日，不比较月 | `internal/shared` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDatesAreKeptInOrder`（`make test`）（Task 11 起） | 单元；组合 |
| `T9.5` | 问题按路径逐字节排序：`tags[10]` 在 `tags[2]` 之前 | `internal/platform/httpserver/bodyshape` 的全部测试（`make test`） | 单元 |
| `T9.6` | 开放对象的未声明成员不查数 | `internal/platform/httpserver/bodyshape` 的全部测试（`make test`） | 单元 |
| `T9.7` | 没有 `items` 的数组的元素不查数 | `internal/platform/httpserver/bodyshape` 的全部测试（`make test`） | 单元 |
| `T9.8` | 查数的一遍什么都不报 | `internal/platform/httpserver/bodyshape` 的全部测试（`make test`） | 单元 |
| `T9.9` | 生成器没有 `shared.Date` 的检查器 | `server/tools` 的 `bodyshapegen` 的全部测试（`make test`）、`make gen-go`（生成器拒绝没有映射的 `date`）（Task 11 起） | 单元；守卫 |
| `T9.10` | `issue` 模块的模板把 `date` 留给 `openapi_types.Date` | `make gen-go`（生成器拒绝没有映射的 `date`）（Task 11 起） | 守卫 |
| `T9.11` | `apitest` 的请求体用例不把 `date` 算作检查的格式 | `internal/platform/httpserver/apitest` 的全部测试（`make test`） | 单元 |
| `T9.12` | `shared.DateOf` 取那个时刻在 UTC 的那一天，不取它所在的时区的 | `internal/shared` 的全部测试（`make test`） | 单元 |
| `T9.13` | `Date` 的 JSON 是它的零点的时刻（RFC 3339），不是 `YYYY-MM-DD` | `internal/shared` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDatesAreKeptInOrder`（`make test`）（Task 11 起） | 单元；组合 |
| `T9.14` | 下标与下标只按位数比较，位数相同时由路径其余的部分决定：`tags[2].a` 在 `tags[1].x` 之前 | `internal/platform/httpserver/bodyshape` 的全部测试（`make test`） | 单元 |

---

### Task 10: 起止日期：领域、用例、存储

**Files:**
- Create: `server/internal/modules/issue/app/dates_test.go`、`server/internal/modules/issue/domain/dates.go`、`server/internal/modules/issue/domain/dates_test.go`
- Modify: `server/internal/modules/issue/adapter/postgres/issues.go`、`server/internal/modules/issue/adapter/postgres/issues_test.go`、`server/internal/modules/issue/adapter/postgres/queries/issues.sql`、`server/internal/modules/issue/adapter/postgres/queries/update.sql`、`server/internal/modules/issue/adapter/postgres/update.go`、`server/internal/modules/issue/adapter/postgres/update_test.go`、`server/internal/modules/issue/app/create_issue.go`、`server/internal/modules/issue/app/create_issue_test.go`、`server/internal/modules/issue/app/fakes_test.go`、`server/internal/modules/issue/app/rows.go`、`server/internal/modules/issue/app/update_issue.go`、`server/internal/modules/issue/domain/activity.go`、`server/internal/modules/issue/domain/activity_test.go`、`server/internal/modules/issue/domain/issue.go`、`server/internal/modules/issue/domain/issue_test.go`、`server/internal/modules/issue/domain/patch.go`、`server/internal/modules/issue/domain/patch_test.go`、`server/sqlc.yaml`
- Generate: `server/internal/modules/issue/adapter/postgres/gen/issues.sql.go`、`server/internal/modules/issue/adapter/postgres/gen/models.go`、`server/internal/modules/issue/adapter/postgres/gen/update.sql.go`

**Interfaces:**（spec 2.11、第 3 节第 10、11 条；M4 设计 3.7、3.13、6.1）
- `server/sqlc.yaml`：`issue` 的 `date` → `time.Time`（UTC 的零点），可为空的是指针。查询：`queries/issues.sql` 的 `InsertIssue` 写、`GetIssue` 读 `start_date`、`target_date`；`queries/update.sql` 的 `UpdateIssue` 写它们。
- `issue/domain`：`Issue.StartDate`、`TargetDate *shared.Date`，`NewIssue` 的两个日期（`issue.go`）；`dates.go`（新）：`DateChange{To}`、`outOfOrder`、`checkNewDates`（创建：只看请求，开始晚于截止时在 `target_date`，`not_allowed`）、`DatesOutOfOrder(i, p) *shared.FieldError`（修改：合并之后的值；只给了开始日期时在 `start_date`，否则在 `target_date`；同一天允许）、`sameDay`、`dayText`；`patch.go`：`IssuePatch.SetStartDate`、`StartDate`、`SetTargetDate`、`TargetDate`（`Set…` 说给没给，给了 `nil` 是清空），`Changes.StartDate`、`TargetDate *DateChange`（与工作项的相同时不算改动）；`activity.go`：`Update.StartDate`、`TargetDate` 和动态第 8、9 行（`start_date`、`target_date`，`YYYY-MM-DD` 或 `""`，`comment` 照 Plane："updated the start date to "（带结尾的空格）、"updated the target date to"）。
- `issue/app`：`create_issue.go` 的 `CheckNewIssue` 在任何读之前查创建的两个日期；`update_issue.go` 的 `check` 在锁下、在状态、负责人、标签之后查合并之后的日期（`domain.DatesOutOfOrder`），与它们一次报出；`rows.go` 的 `IssueRow`、`IssueFields` 带两个日期。
- 存储：`issues.go` 的 `dayOf`（读出的 `time.Time` 是 UTC 的零点 → `shared.Date`）、`timeOf`（反过来）；`InsertIssue`、`UpdateIssue`、`GetIssue` 用它们（`update.go`）。
- 测试的帮手：`fakes_test.go` 的假存储记下并读出两个日期。

**Tests:**
- 领域：`TestDatesOutOfOrder`（给的日期，没给的用工作项的；清空的、没有的日期不定次序；同一天是对的）、`TestCheckNewIssueAccepts`、`TestCheckNewIssueReportsEveryProblem`（开始晚于截止在 `target_date`）、`TestChangesOf`（与工作项相同的日期不算改动）、`TestChangesNone`（日期的改动单独也算）、`TestCheckIssuePatchAccepts`、`TestUpdateActivity`（第 8、9 行）。
- 单元：`TestUpdateIssueHoldsTheMergedDatesInOrder`（工作项从 12 日到 19 日：只给晚于留着的截止日期的开始日期，只看请求会放过、合并之后不对，在 `start_date`；只给早于留着的开始日期的截止日期，在 `target_date`；两个都给、各自与原来的另一个日期次序对而彼此不对，在 `target_date`；都在依赖行的问题之后、同一个 422，什么都不写）、`TestUpdateIssueWritesTheDates`（两个都移过原来的截止日期：与原来的值比较会拒绝、合并之后对的，写下；清空开始日期并给早于原来开始日期的截止日期；只给与留着的截止日期同一天的开始日期；清空截止日期；每个改了的日期一行动态）、`TestCreateIssueRefusesBeforeAnyLock`（开始晚于截止，在任何读之前）、`TestCreateIssueWritesTheWorkItemAndItsRows`（两个日期写下）。
- 存储：`TestCreatingAWorkItemWritesItsRows`（给的每个值，两个日期也是，读回）、`TestUpdatingAWorkItemWritesItsRows`（日期照给的写下，开始日期给 `nil` 是 `NULL`）。

- [ ] **Step 1: 映射和查询，生成**

`server/internal/modules/issue/adapter/postgres/queries/issues.sql`（修改，3 处）：

````old server/internal/modules/issue/adapter/postgres/queries/issues.sql
INSERT INTO issues (id, workspace_id, project_id, state_id, name, priority, sequence_id, sort_order, completed_at,
                    created_by_id, updated_by_id, created_at, updated_at)
````
````new server/internal/modules/issue/adapter/postgres/queries/issues.sql
INSERT INTO issues (id, workspace_id, project_id, state_id, name, priority, start_date, target_date, sequence_id, sort_order,
                    completed_at, created_by_id, updated_by_id, created_at, updated_at)
````
````old server/internal/modules/issue/adapter/postgres/queries/issues.sql
        sqlc.arg(sequence_id), sqlc.arg(sort_order), sqlc.narg(completed_at), sqlc.arg(by)::uuid, sqlc.arg(by),
        sqlc.arg(at)::timestamptz, sqlc.arg(at));
````
````new server/internal/modules/issue/adapter/postgres/queries/issues.sql
        sqlc.narg(start_date), sqlc.narg(target_date), sqlc.arg(sequence_id), sqlc.arg(sort_order), sqlc.narg(completed_at),
        sqlc.arg(by)::uuid, sqlc.arg(by), sqlc.arg(at)::timestamptz, sqlc.arg(at));
````
````old server/internal/modules/issue/adapter/postgres/queries/issues.sql
SELECT i.id, i.workspace_id, i.project_id, i.sequence_id, i.name, i.state_id, i.priority, i.parent_id, i.sort_order,
       i.completed_at, i.archived_at, i.created_by_id, i.updated_by_id, i.created_at, i.updated_at,
````
````new server/internal/modules/issue/adapter/postgres/queries/issues.sql
SELECT i.id, i.workspace_id, i.project_id, i.sequence_id, i.name, i.state_id, i.priority, i.start_date, i.target_date,
       i.parent_id, i.sort_order, i.completed_at, i.archived_at, i.created_by_id, i.updated_by_id, i.created_at, i.updated_at,
````

`server/internal/modules/issue/adapter/postgres/queries/update.sql`（修改，2 处）：

````old server/internal/modules/issue/adapter/postgres/queries/update.sql
-- priority and completion as the use case gives them, those it changes and the others as they are, last changed at the
-- moment and by the account given. Every write of a work item's row holds its project's lock first (3.5), so the row is
-- as the use case read it; it is not locked by id first (M3 design 3.6, convention 5).
````
````new server/internal/modules/issue/adapter/postgres/queries/update.sql
-- priority, dates and completion as the use case gives them, those it changes and the others as they are, last changed
-- at the moment and by the account given. Every write of a work item's row holds its project's lock first (3.5), so the
-- row is as the use case read it; it is not locked by id first (M3 design 3.6, convention 5).
````
````old server/internal/modules/issue/adapter/postgres/queries/update.sql
SET name = sqlc.arg(name), state_id = sqlc.arg(state_id), priority = sqlc.arg(priority), completed_at = sqlc.narg(completed_at),
    updated_by_id = sqlc.arg(by)::uuid, updated_at = sqlc.arg(at)::timestamptz
````
````new server/internal/modules/issue/adapter/postgres/queries/update.sql
SET name = sqlc.arg(name), state_id = sqlc.arg(state_id), priority = sqlc.arg(priority), start_date = sqlc.narg(start_date),
    target_date = sqlc.narg(target_date), completed_at = sqlc.narg(completed_at), updated_by_id = sqlc.arg(by)::uuid,
    updated_at = sqlc.arg(at)::timestamptz
````

`server/sqlc.yaml`（修改，2 处）：

````old server/sqlc.yaml
# 类型覆盖：uuid 用标准库的 uuid.UUID，timestamptz 用 time.Time（db_type 写 timestamptz，写 pg_catalog.timestamptz 不匹配）。
````
````new server/sqlc.yaml
# 类型覆盖：uuid 用标准库的 uuid.UUID，timestamptz 用 time.Time（db_type 写 timestamptz，写 pg_catalog.timestamptz 不匹配）；
# issue 的 date 也用 time.Time（UTC 的零点），适配器换成 shared.Date（M4 设计 6.1）。
````
````old server/sqlc.yaml
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
````new server/sqlc.yaml
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
          - db_type: date
            go_type: {import: time, type: Time}
          - db_type: date
            nullable: true
            go_type: {import: time, type: Time, pointer: true}

````

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `235bd7f300ea1988ff2d53af319b52b8fe8f1e5998bf59535f41f3958a467764` | 303 | `server/internal/modules/issue/adapter/postgres/gen/issues.sql.go` |
| `59e0fe885115707e17713f7cb7f2d4db82588d61ff88026c268b4f501ef930cd` | 93 | `server/internal/modules/issue/adapter/postgres/gen/models.go` |
| `5beaddd300fa8db852b4c6972ae0e32d3a794726d5928892cb2aa5c37576dc09` | 129 | `server/internal/modules/issue/adapter/postgres/gen/update.sql.go` |

Run: `shasum -a 256 server/internal/modules/issue/adapter/postgres/gen/issues.sql.go server/internal/modules/issue/adapter/postgres/gen/models.go server/internal/modules/issue/adapter/postgres/gen/update.sql.go`
Expected: 与上表相同。

- [ ] **Step 2: 领域**

`server/internal/modules/issue/domain/activity.go`（修改，5 处）：

````old server/internal/modules/issue/domain/activity.go
import "uuid"
````
````new server/internal/modules/issue/domain/activity.go
import (
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)
````
````old server/internal/modules/issue/domain/activity.go
// it (M4 design 3.13): its name, its priority and its state, each nil when
// it is left as it is; the labels and the accounts it adds and removes,
// each in id order.
````
````new server/internal/modules/issue/domain/activity.go
// it (M4 design 3.13): its name, its priority, its state and its dates,
// each nil when it is left as it is; the labels and the accounts it adds
// and removes, each in id order.
````
````old server/internal/modules/issue/domain/activity.go
	State            *Changed[Named]
````
````new server/internal/modules/issue/domain/activity.go
	State            *Changed[Named]
	StartDate        *Changed[*shared.Date]
	TargetDate       *Changed[*shared.Date]
````
````old server/internal/modules/issue/domain/activity.go
// its names and its ids; then each label added (row 10) and removed (row
// 11), by its name and its id; then each account assigned (row 12) and
// unassigned (row 13), by his display name and his id; each list in the
// order given. A field left as it is writes no row (Plane the same).
````
````new server/internal/modules/issue/domain/activity.go
// its names and its ids; rows 8 and 9, the start and the target dates, as
// YYYY-MM-DD or "" for none; then each label added (row 10) and removed
// (row 11), by its name and its id; then each account assigned (row 12)
// and unassigned (row 13), by his display name and his id; each list in
// the order given. A field left as it is writes no row (Plane the same).
````
````old server/internal/modules/issue/domain/activity.go
			OldIdentifier: ptr(u.State.Old.ID), NewIdentifier: ptr(u.State.New.ID), Comment: "updated the state to"})
````
````new server/internal/modules/issue/domain/activity.go
			OldIdentifier: ptr(u.State.Old.ID), NewIdentifier: ptr(u.State.New.ID), Comment: "updated the state to"})
	}
	if u.StartDate != nil {
		rows = append(rows, Activity{Field: ptr("start_date"), Verb: VerbUpdated, OldValue: ptr(dayText(u.StartDate.Old)),
			NewValue: ptr(dayText(u.StartDate.New)), Comment: "updated the start date to "})
	}
	if u.TargetDate != nil {
		rows = append(rows, Activity{Field: ptr("target_date"), Verb: VerbUpdated, OldValue: ptr(dayText(u.TargetDate.Old)),
			NewValue: ptr(dayText(u.TargetDate.New)), Comment: "updated the target date to"})
````

`server/internal/modules/issue/domain/activity_test.go`（修改，5 处）：

````old server/internal/modules/issue/domain/activity_test.go
	"uuid"
````
````new server/internal/modules/issue/domain/activity_test.go
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
````
````old server/internal/modules/issue/domain/activity_test.go
// priority (row 6), the state by its names and ids (row 7), each label
// added and removed by its name and id (rows 10, 11), each account
// assigned and unassigned by his display name and id (rows 12, 13), each
// list in the order given; Plane's sentences go with them. A field left as
// it is writes none.
````
````new server/internal/modules/issue/domain/activity_test.go
// priority (row 6), the state by its names and ids (row 7), the start and
// the target dates as YYYY-MM-DD, "" for none (rows 8, 9), each label added
// and removed by its name and id (rows 10, 11), each account assigned and
// unassigned by his display name and id (rows 12, 13), each list in the
// order given; Plane's sentences go with them, the start date's with its
// space. A field left as it is writes none.
````
````old server/internal/modules/issue/domain/activity_test.go
	state := row("state", "Todo", "Done", &todo, &done, "updated the state to")
````
````new server/internal/modules/issue/domain/activity_test.go
	state := row("state", "Todo", "Done", &todo, &done, "updated the state to")
	start := row("start_date", "2026-10-12", "", nil, nil, "updated the start date to ")
	target := row("target_date", "", "2026-10-19", nil, nil, "updated the target date to")
````
````old server/internal/modules/issue/domain/activity_test.go
			LabelsRemoved: []Named{{ID: ui, Name: "UI"}}, LabelsAdded: []Named{{ID: bug, Name: "Bug"}},
````
````new server/internal/modules/issue/domain/activity_test.go
			LabelsRemoved: []Named{{ID: ui, Name: "UI"}}, LabelsAdded: []Named{{ID: bug, Name: "Bug"}},
			TargetDate: &Changed[*shared.Date]{New: october(19)}, StartDate: &Changed[*shared.Date]{Old: october(12)},
````
````old server/internal/modules/issue/domain/activity_test.go
		}, []Activity{name, priority, state, labelled(bug, "Bug"), unlabelled(ui, "UI"), assigned(ann, "Ann L."), unassigned(ben, "ben")}},
````
````new server/internal/modules/issue/domain/activity_test.go
		}, []Activity{name, priority, state, start, target, labelled(bug, "Bug"), unlabelled(ui, "UI"), assigned(ann, "Ann L."),
			unassigned(ben, "ben")}},
		{"a start date set, the target date moved", Update{StartDate: &Changed[*shared.Date]{New: october(19)},
			TargetDate: &Changed[*shared.Date]{Old: october(19), New: october(26)}}, []Activity{
			row("start_date", "", "2026-10-19", nil, nil, "updated the start date to "),
			row("target_date", "2026-10-19", "2026-10-26", nil, nil, "updated the target date to")}},
````

`server/internal/modules/issue/domain/dates.go`（新文件，70 行）：

````file server/internal/modules/issue/domain/dates.go
package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// A work item's start and target dates (M4 design 3.7): each a day, or
// none; the start not after the target when it has both, which the
// database's issues_dates_check holds too. The same day for both is in
// order.

// DateChange is a date a change sets: To, the new day, nil when it clears
// the date.
type DateChange struct {
	To *shared.Date
}

// outOfOrder reports whether start comes after target, either nil for
// none: a date missing orders nothing.
func outOfOrder(start, target *shared.Date) bool {
	return start != nil && target != nil && start.Compare(*target) > 0
}

// checkNewDates refuses the dates of a new work item, both as the request
// gives them, when the start comes after the target: at target_date.
func checkNewDates(start, target *shared.Date) *shared.FieldError {
	if outOfOrder(start, target) {
		return targetBeforeStart()
	}
	return nil
}

// DatesOutOfOrder is the problem of the dates the work item i has once p,
// a patch, changes it (M4 design 3.7): the dates p gives, each else i's.
// A start after the target is refused at target_date, or at start_date
// when p gives the start date alone; nil when they are in order. So a
// start date given alone is held to the target date the work item keeps,
// and one given with a target date to that date, not to the one it had.
func DatesOutOfOrder(i Issue, p IssuePatch) *shared.FieldError {
	start, target := i.StartDate, i.TargetDate
	if p.SetStartDate {
		start = p.StartDate
	}
	if p.SetTargetDate {
		target = p.TargetDate
	}
	switch {
	case !outOfOrder(start, target):
		return nil
	case p.SetStartDate && !p.SetTargetDate:
		return &shared.FieldError{Field: "start_date", Code: shared.FieldNotAllowed, Message: "must not be after the target date"}
	}
	return targetBeforeStart()
}

// targetBeforeStart is the problem of a target date before the start date.
func targetBeforeStart() *shared.FieldError {
	return &shared.FieldError{Field: "target_date", Code: shared.FieldNotAllowed, Message: "must not be before the start date"}
}

// sameDay reports whether a and b, either nil for none, are the same day.
func sameDay(a, b *shared.Date) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

// dayText is d as an activity's value: YYYY-MM-DD, "" for none.
func dayText(d *shared.Date) string {
	if d == nil {
		return ""
	}
	return d.String()
}
````

`server/internal/modules/issue/domain/dates_test.go`（新文件，53 行）：

````file server/internal/modules/issue/domain/dates_test.go
package domain

import (
	"reflect"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// october is the day d of October 2026, a work item's date.
func october(d int) *shared.Date {
	day := shared.DateOf(time.Date(2026, time.October, d, 0, 0, 0, 0, time.UTC))
	return &day
}

// The dates a work item has once a patch changes it are held in order (M4
// design 3.7): each date given, else the work item's. A start date given
// alone is held to the target date the work item keeps, and refused at
// start_date; given with a target date, to that date, not to the one it
// had, and refused at target_date, as a target date given alone is. A date
// cleared, or missing, orders nothing; the same day for both is in order.
func TestDatesOutOfOrder(t *testing.T) {
	start := &shared.FieldError{Field: "start_date", Code: "not_allowed", Message: "must not be after the target date"}
	target := &shared.FieldError{Field: "target_date", Code: "not_allowed", Message: "must not be before the start date"}
	from12To19 := Issue{StartDate: october(12), TargetDate: october(19)}
	tests := []struct {
		name string
		i    Issue
		p    IssuePatch
		want *shared.FieldError
	}{
		{"nothing given", from12To19, IssuePatch{}, nil},
		{"a start date alone, on the target date kept", from12To19, IssuePatch{SetStartDate: true, StartDate: october(19)}, nil},
		{"a start date alone, after the target date kept", from12To19, IssuePatch{SetStartDate: true, StartDate: october(20)}, start},
		{"a target date alone, on the start date kept", from12To19, IssuePatch{SetTargetDate: true, TargetDate: october(12)}, nil},
		{"a target date alone, before the start date kept", from12To19, IssuePatch{SetTargetDate: true, TargetDate: october(11)}, target},
		{"both in order, past the target date it had", from12To19,
			IssuePatch{SetStartDate: true, StartDate: october(20), SetTargetDate: true, TargetDate: october(25)}, nil},
		{"both out of order, each in order with the date it had", from12To19,
			IssuePatch{SetStartDate: true, StartDate: october(15), SetTargetDate: true, TargetDate: october(14)}, target},
		{"a start date after the target date cleared", from12To19, IssuePatch{SetStartDate: true, StartDate: october(30), SetTargetDate: true}, nil},
		{"the start date cleared, a target date before it", from12To19, IssuePatch{SetStartDate: true, SetTargetDate: true, TargetDate: october(1)},
			nil},
		{"a start date alone, no target date", Issue{}, IssuePatch{SetStartDate: true, StartDate: october(30)}, nil},
		{"a target date alone, no start date", Issue{TargetDate: october(19)}, IssuePatch{SetTargetDate: true, TargetDate: october(1)}, nil},
	}
	for _, tt := range tests {
		if got := DatesOutOfOrder(tt.i, tt.p); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: DatesOutOfOrder() = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}
````

`server/internal/modules/issue/domain/issue.go`（修改，5 处）：

````old server/internal/modules/issue/domain/issue.go
	Priority    Priority
````
````new server/internal/modules/issue/domain/issue.go
	Priority    Priority
	StartDate   *shared.Date
	TargetDate  *shared.Date
````
````old server/internal/modules/issue/domain/issue.go
// for the project's default state, Priority nil for none. A field given is
// checked as given, a priority given empty too.
````
````new server/internal/modules/issue/domain/issue.go
// for the project's default state, Priority nil for none, a date nil for
// none. A field given is checked as given, a priority given empty too.
````
````old server/internal/modules/issue/domain/issue.go
	AssigneeIDs []uuid.UUID
	LabelIDs    []uuid.UUID
}
````
````new server/internal/modules/issue/domain/issue.go
	AssigneeIDs []uuid.UUID
	LabelIDs    []uuid.UUID
	StartDate   *shared.Date
	TargetDate  *shared.Date
}
````
````old server/internal/modules/issue/domain/issue.go
// MaxLabels labels, each named once. Every problem is reported at once, in
// one 422 validation_failed; whether the state, each assignee and each
// label may be is the use case's, under the project's lock.
````
````new server/internal/modules/issue/domain/issue.go
// MaxLabels labels, each named once; the start date not after the target
// date, as the request gives both (at target_date). Every problem is
// reported at once, in one 422 validation_failed; whether the state, each
// assignee and each label may be is the use case's, under the project's
// lock.
````
````old server/internal/modules/issue/domain/issue.go
	found = append(found, checkIDs("label_ids", n.LabelIDs, MaxLabels)...)
````
````new server/internal/modules/issue/domain/issue.go
	found = append(found, checkIDs("label_ids", n.LabelIDs, MaxLabels)...)
	found = append(found, checkNewDates(n.StartDate, n.TargetDate))
````

`server/internal/modules/issue/domain/issue_test.go`（修改，6 处）：

````old server/internal/modules/issue/domain/issue_test.go
// assignees and labels: accepted, and returned as stored: the name without
// its surrounding white space, the priority none when none is given.
````
````new server/internal/modules/issue/domain/issue_test.go
// assignees and labels, the dates in order, on the same day, or one alone:
// accepted, and returned as stored: the name without its surrounding white
// space, the priority none when none is given, the dates as given.
````
````old server/internal/modules/issue/domain/issue_test.go
		{"everything", NewIssue{Name: "Fix", StateID: &state, Priority: ptr(PriorityUrgent), AssigneeIDs: ids(100), LabelIDs: ids(100)},
			NewIssue{Name: "Fix", StateID: &state, Priority: ptr(PriorityUrgent)}},
````
````new server/internal/modules/issue/domain/issue_test.go
		{"everything", NewIssue{Name: "Fix", StateID: &state, Priority: ptr(PriorityUrgent), AssigneeIDs: ids(100), LabelIDs: ids(100),
			StartDate: october(12), TargetDate: october(19)},
			NewIssue{Name: "Fix", StateID: &state, Priority: ptr(PriorityUrgent), StartDate: october(12), TargetDate: october(19)}},
		{"the same day", NewIssue{Name: "a", StartDate: october(19), TargetDate: october(19)},
			NewIssue{Name: "a", Priority: ptr(PriorityNone), StartDate: october(19), TargetDate: october(19)}},
		{"a start date alone", NewIssue{Name: "a", StartDate: october(30)}, NewIssue{Name: "a", Priority: ptr(PriorityNone), StartDate: october(30)}},
		{"a target date alone", NewIssue{Name: "a", TargetDate: october(1)}, NewIssue{Name: "a", Priority: ptr(PriorityNone), TargetDate: october(1)}},
````
````old server/internal/modules/issue/domain/issue_test.go
// a hundred and one of either: each refused, all of them in one 422 that
// names each by its field and place. A list too long is reported alone: its
// ids, a repeat among them, are not looked at.
````
````new server/internal/modules/issue/domain/issue_test.go
// a hundred and one of either, a start date after the target date (at
// target_date): each refused, all of them in one 422 that names each by its
// field and place. A list too long is reported alone: its ids, a repeat
// among them, are not looked at.
````
````old server/internal/modules/issue/domain/issue_test.go
	priority := []shared.FieldError{{Field: "priority", Code: "invalid_format", Message: "must be one of urgent, high, medium, low, none"}}
````
````new server/internal/modules/issue/domain/issue_test.go
	priority := []shared.FieldError{{Field: "priority", Code: "invalid_format", Message: "must be one of urgent, high, medium, low, none"}}
	dates := shared.FieldError{Field: "target_date", Code: "not_allowed", Message: "must not be before the start date"}
````
````old server/internal/modules/issue/domain/issue_test.go
		{"too many, with repeats, and everything else", NewIssue{Name: "", Priority: ptr[Priority]("high!"), AssigneeIDs: many, LabelIDs: many},
````
````new server/internal/modules/issue/domain/issue_test.go
		{"a start date after the target date", NewIssue{Name: "a", StartDate: october(20), TargetDate: october(19)}, []shared.FieldError{dates}},
		{"too many, with repeats, and everything else", NewIssue{Name: "", Priority: ptr[Priority]("high!"), AssigneeIDs: many, LabelIDs: many,
			StartDate: october(20), TargetDate: october(19)},
````
````old server/internal/modules/issue/domain/issue_test.go
				{Field: "label_ids", Code: "too_long", Message: "must name at most 100"},
````
````new server/internal/modules/issue/domain/issue_test.go
				{Field: "label_ids", Code: "too_long", Message: "must name at most 100"},
				dates,
````

`server/internal/modules/issue/domain/patch.go`（修改，7 处）：

````old server/internal/modules/issue/domain/patch.go
// none.
````
````new server/internal/modules/issue/domain/patch.go
// none. A date may be cleared: SetStartDate and SetTargetDate tell whether
// the date is given, StartDate and TargetDate its day, nil for none.
````
````old server/internal/modules/issue/domain/patch.go
	Name        *string
	StateID     *uuid.UUID
	Priority    *Priority
	AssigneeIDs *[]uuid.UUID
	LabelIDs    *[]uuid.UUID
````
````new server/internal/modules/issue/domain/patch.go
	Name          *string
	StateID       *uuid.UUID
	Priority      *Priority
	AssigneeIDs   *[]uuid.UUID
	LabelIDs      *[]uuid.UUID
	SetStartDate  bool
	StartDate     *shared.Date
	SetTargetDate bool
	TargetDate    *shared.Date
````
````old server/internal/modules/issue/domain/patch.go
// change adds may be is the use case's, under the project's lock.
````
````new server/internal/modules/issue/domain/patch.go
// change adds may be is the use case's, under the project's lock, and so
// is the order of the dates the change leaves (DatesOutOfOrder): they are
// the work item's, merged with those given.
````
````old server/internal/modules/issue/domain/patch.go
// the new name, priority and state where each differs from the work
// item's, nil where it is not given or is the same; and the accounts and
// the labels it adds and removes (Diff), where the patch gives the new
// set. An account or a label the work item has and the new set keeps is
// left as it is: its row is not written again.
````
````new server/internal/modules/issue/domain/patch.go
// the new name, priority, state and dates where each differs from the
// work item's, nil where it is not given or is the same; and the accounts
// and the labels it adds and removes (Diff), where the patch gives the
// new set. An account or a label the work item has and the new set keeps
// is left as it is: its row is not written again.
````
````old server/internal/modules/issue/domain/patch.go
	Name      *string
	Priority  *Priority
	StateID   *uuid.UUID
	Assignees Diff
	Labels    Diff
````
````new server/internal/modules/issue/domain/patch.go
	Name       *string
	Priority   *Priority
	StateID    *uuid.UUID
	StartDate  *DateChange
	TargetDate *DateChange
	Assignees  Diff
	Labels     Diff
````
````old server/internal/modules/issue/domain/patch.go
		c.StateID = p.StateID
````
````new server/internal/modules/issue/domain/patch.go
		c.StateID = p.StateID
	}
	if p.SetStartDate && !sameDay(p.StartDate, i.StartDate) {
		c.StartDate = &DateChange{To: p.StartDate}
	}
	if p.SetTargetDate && !sameDay(p.TargetDate, i.TargetDate) {
		c.TargetDate = &DateChange{To: p.TargetDate}
````
````old server/internal/modules/issue/domain/patch.go
	return c.Name == nil && c.Priority == nil && c.StateID == nil && len(c.Assignees.Added) == 0 && len(c.Assignees.Removed) == 0 &&
		len(c.Labels.Added) == 0 && len(c.Labels.Removed) == 0
````
````new server/internal/modules/issue/domain/patch.go
	return c.Name == nil && c.Priority == nil && c.StateID == nil && c.StartDate == nil && c.TargetDate == nil && len(c.Assignees.Added) == 0 &&
		len(c.Assignees.Removed) == 0 && len(c.Labels.Added) == 0 && len(c.Labels.Removed) == 0
````

`server/internal/modules/issue/domain/patch_test.go`（修改，7 处）：

````old server/internal/modules/issue/domain/patch_test.go
// hundred accounts and labels, and the two sets given empty: accepted, and
// returned as stored: the name without its surrounding white space, a set
// given empty still given (it empties the work item's), a field not given
// still not given.
````
````new server/internal/modules/issue/domain/patch_test.go
// hundred accounts and labels, the two sets given empty, and the dates
// given out of order or cleared: accepted, and returned as stored: the name
// without its surrounding white space, a set given empty still given (it
// empties the work item's), a date cleared still given, a field not given
// still not given. The order of the dates is the work item's once merged,
// which the use case holds under the lock (DatesOutOfOrder).
````
````old server/internal/modules/issue/domain/patch_test.go
		{"both sets empty", IssuePatch{AssigneeIDs: &none, LabelIDs: &none}, IssuePatch{AssigneeIDs: &none, LabelIDs: &none}},
````
````new server/internal/modules/issue/domain/patch_test.go
		{"both sets empty", IssuePatch{AssigneeIDs: &none, LabelIDs: &none}, IssuePatch{AssigneeIDs: &none, LabelIDs: &none}},
		{"the dates out of order", IssuePatch{SetStartDate: true, StartDate: october(20), SetTargetDate: true, TargetDate: october(19)},
			IssuePatch{SetStartDate: true, StartDate: october(20), SetTargetDate: true, TargetDate: october(19)}},
		{"both dates cleared", IssuePatch{SetStartDate: true, SetTargetDate: true}, IssuePatch{SetStartDate: true, SetTargetDate: true}},
````
````old server/internal/modules/issue/domain/patch_test.go
// What a patch changes in a work item, Fix in Todo, high, assigned to ann
// and ben, labelled Bug and UI (M4 design 3.7): a field not given, or given
// as it is, changes nothing; a set given is the new set, whose difference
````
````new server/internal/modules/issue/domain/patch_test.go
// What a patch changes in a work item, Fix in Todo, high, from the 12th
// with no target date, assigned to ann and ben, labelled Bug and UI (M4
// design 3.7): a field not given, or given as it is, changes nothing, a
// date the same day or none again too; a date given another day, or
// cleared, is the new one; a set given is the new set, whose difference
````
````old server/internal/modules/issue/domain/patch_test.go
	i := Issue{Name: "Fix", StateID: todo, Priority: PriorityHigh, AssigneeIDs: []uuid.UUID{ann, ben}, LabelIDs: []uuid.UUID{bug, ui}}
````
````new server/internal/modules/issue/domain/patch_test.go
	i := Issue{Name: "Fix", StateID: todo, Priority: PriorityHigh, StartDate: october(12), AssigneeIDs: []uuid.UUID{ann, ben},
		LabelIDs: []uuid.UUID{bug, ui}}
````
````old server/internal/modules/issue/domain/patch_test.go
		{"each given as it is", IssuePatch{Name: ptr("Fix"), StateID: &todo, Priority: ptr(PriorityHigh), AssigneeIDs: set(ben, ann),
			LabelIDs: set(ui, bug)}, Changes{}},
````
````new server/internal/modules/issue/domain/patch_test.go
		{"each given as it is", IssuePatch{Name: ptr("Fix"), StateID: &todo, Priority: ptr(PriorityHigh), SetStartDate: true,
			StartDate: october(12), SetTargetDate: true, AssigneeIDs: set(ben, ann), LabelIDs: set(ui, bug)}, Changes{}},
````
````old server/internal/modules/issue/domain/patch_test.go
		{"the state", IssuePatch{StateID: &done}, Changes{StateID: &done}},
````
````new server/internal/modules/issue/domain/patch_test.go
		{"the state", IssuePatch{StateID: &done}, Changes{StateID: &done}},
		{"the start date", IssuePatch{SetStartDate: true, StartDate: october(13)}, Changes{StartDate: &DateChange{To: october(13)}}},
		{"the start date cleared", IssuePatch{SetStartDate: true}, Changes{StartDate: &DateChange{}}},
		{"a target date", IssuePatch{SetTargetDate: true, TargetDate: october(19)}, Changes{TargetDate: &DateChange{To: october(19)}}},
````
````old server/internal/modules/issue/domain/patch_test.go
		"the name": {Name: ptr("x")}, "the priority": {Priority: ptr(PriorityLow)}, "the state": {StateID: &id},
````
````new server/internal/modules/issue/domain/patch_test.go
		"the name": {Name: ptr("x")}, "the priority": {Priority: ptr(PriorityLow)}, "the state": {StateID: &id},
		"the start date cleared": {StartDate: &DateChange{}}, "a target date": {TargetDate: &DateChange{To: october(19)}},
````

- [ ] **Step 3: 用例**

`server/internal/modules/issue/app/create_issue.go`（修改，1 处）：

````old server/internal/modules/issue/app/create_issue.go
			StateID: c.state.ID, Priority: *in.Priority, SortOrder: domain.SortOrderAfter(greatest),
			CompletedAt: domain.CompletedAt(c.state.Group, now), By: actor.UserID, At: now}
````
````new server/internal/modules/issue/app/create_issue.go
			StateID: c.state.ID, Priority: *in.Priority, StartDate: in.StartDate, TargetDate: in.TargetDate,
			SortOrder: domain.SortOrderAfter(greatest), CompletedAt: domain.CompletedAt(c.state.Group, now), By: actor.UserID, At: now}
````

`server/internal/modules/issue/app/create_issue_test.go`（修改，5 处）：

````old server/internal/modules/issue/app/create_issue_test.go
// stored without its surrounding space; the work item in the completed
// state is completed at the moment, which is also every row's, the
// caller's.
````
````new server/internal/modules/issue/app/create_issue_test.go
// stored without its surrounding space, the dates as given; the work item
// in the completed state is completed at the moment, which is also every
// row's, the caller's.
````
````old server/internal/modules/issue/app/create_issue_test.go
		AssigneeIDs: []uuid.UUID{carol, bob}, LabelIDs: []uuid.UUID{ui, bug}})
````
````new server/internal/modules/issue/app/create_issue_test.go
		AssigneeIDs: []uuid.UUID{carol, bob}, LabelIDs: []uuid.UUID{ui, bug}, StartDate: october(12), TargetDate: october(19)})
````
````old server/internal/modules/issue/app/create_issue_test.go
		Priority: domain.PriorityHigh, SortOrder: 95535, CompletedAt: &clockNow, By: alice, At: clockNow}
````
````new server/internal/modules/issue/app/create_issue_test.go
		Priority: domain.PriorityHigh, StartDate: october(12), TargetDate: october(19), SortOrder: 95535, CompletedAt: &clockNow, By: alice,
		At: clockNow}
````
````old server/internal/modules/issue/app/create_issue_test.go
// allow is forbidden, and nothing is locked. Values that break the rules
// are refused before even the project is read.
````
````new server/internal/modules/issue/app/create_issue_test.go
// allow is forbidden, and nothing is locked. Values that break the rules,
// the dates out of order as given too, are refused before even the project
// is read.
````
````old server/internal/modules/issue/app/create_issue_test.go
			Message: "must not be empty"}), nil},
````
````new server/internal/modules/issue/app/create_issue_test.go
			Message: "must not be empty"}), nil},
		{"a start date after the target date", func(*world) {}, domain.NewIssue{Name: "a", StartDate: october(20), TargetDate: october(19)},
			shared.Invalid(shared.FieldError{Field: "target_date", Code: "not_allowed", Message: "must not be before the start date"}), nil},
````

`server/internal/modules/issue/app/dates_test.go`（新文件，103 行）：

````file server/internal/modules/issue/app/dates_test.go
package app_test

import (
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// october is the day d of October 2026, a work item's date.
func october(d int) *shared.Date {
	day := shared.DateOf(time.Date(2026, time.October, d, 0, 0, 0, 0, time.UTC))
	return &day
}

// An update writes the dates the work item then has (M4 design 3.7, 3.13):
// each date given, the other kept; each date it changes is a row of its
// activity, from and to YYYY-MM-DD or "" for none (rows 8, 9), alice's at
// the moment, after the clock and the work item's fields; the answer is the
// work item read back. Web's work item 7 runs from the 12th to the 19th:
// both dates moved past the target date it had, which a start date given
// with the target date is not held to; the start date cleared, and a
// target date before the start date it had; a start date alone, on the
// target date it keeps; the target date cleared.
func TestUpdateIssueWritesTheDates(t *testing.T) {
	for _, tt := range []struct {
		name          string
		p             domain.IssuePatch
		start, target *shared.Date
		rows          domain.Update
	}{
		{"both past the target date", domain.IssuePatch{SetStartDate: true, StartDate: october(20), SetTargetDate: true, TargetDate: october(25)},
			october(20), october(25), domain.Update{StartDate: &domain.Changed[*shared.Date]{Old: october(12), New: october(20)},
				TargetDate: &domain.Changed[*shared.Date]{Old: october(19), New: october(25)}}},
		{"the start date cleared, the target date before it", domain.IssuePatch{SetStartDate: true, SetTargetDate: true, TargetDate: october(10)},
			nil, october(10), domain.Update{StartDate: &domain.Changed[*shared.Date]{Old: october(12)},
				TargetDate: &domain.Changed[*shared.Date]{Old: october(19), New: october(10)}}},
		{"a start date alone, on the target date", domain.IssuePatch{SetStartDate: true, StartDate: october(19)}, october(19), october(19),
			domain.Update{StartDate: &domain.Changed[*shared.Date]{Old: october(12), New: october(19)}}},
		{"the target date cleared", domain.IssuePatch{SetTargetDate: true}, october(12), nil,
			domain.Update{TargetDate: &domain.Changed[*shared.Date]{Old: october(19)}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id := newUpdate()
			w.issues.issue.StartDate, w.issues.issue.TargetDate = october(12), october(19)
			got, err := uc.Execute(as(alice), id, tt.p)
			if err != nil || !reflect.DeepEqual(got.StartDate, tt.start) || !reflect.DeepEqual(got.TargetDate, tt.target) {
				t.Fatalf("Execute() = %+v, %v; want it from %v to %v", got, err, tt.start, tt.target)
			}
			rows := domain.UpdateActivity(tt.rows)
			if calls := append(locked(id), "Now", line("UpdateIssue", id), line("InsertActivities", len(rows)), line("GetIssue", id)); !slices.Equal(
				w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
			if fields := (app.IssueFields{ID: id, Name: "Fix", StateID: backlog.ID, Priority: domain.PriorityHigh, StartDate: tt.start,
				TargetDate: tt.target, By: alice, At: clockNow}); w.issues.changed == nil || !reflect.DeepEqual(*w.issues.changed, fields) {
				t.Errorf("the fields written %+v, want %+v", w.issues.changed, fields)
			}
			wantActivity(t, w, id, rows)
		})
	}
}

// The dates an update leaves the work item with are held in order under the
// lock, merged with those it keeps (M4 design 3.7): of web's work item 7,
// from the 12th to the 19th, a start date alone after the target date kept
// is refused at start_date; a target date alone before the start date kept
// at target_date; both given out of order at target_date, though each is
// in order with the date it had. The dates come after the dependent rows'
// problems, in the one 422, and nothing is written.
func TestUpdateIssueHoldsTheMergedDatesInOrder(t *testing.T) {
	other := uuid.NewV7() // after every id of the world
	start := shared.FieldError{Field: "start_date", Code: "not_allowed", Message: "must not be after the target date"}
	target := shared.FieldError{Field: "target_date", Code: "not_allowed", Message: "must not be before the start date"}
	for _, tt := range []struct {
		name  string
		p     domain.IssuePatch
		want  []shared.FieldError
		calls []string // after locked's
	}{
		{"a start date alone after the target date", domain.IssuePatch{SetStartDate: true, StartDate: october(20)}, []shared.FieldError{start}, nil},
		{"a target date alone before the start date", domain.IssuePatch{SetTargetDate: true, TargetDate: october(11)}, []shared.FieldError{target},
			nil},
		{"both out of order, each in order with the date it had",
			domain.IssuePatch{SetStartDate: true, StartDate: october(15), SetTargetDate: true, TargetDate: october(14)}, []shared.FieldError{target},
			nil},
		{"a label not the project's, and a start date alone after the target date",
			domain.IssuePatch{LabelIDs: set(bug, other), SetStartDate: true, StartDate: october(20)},
			[]shared.FieldError{domain.UnknownLabel(1), start}, []string{line("Labels", web, []uuid.UUID{other})}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id := newUpdate()
			w.issues.issue.StartDate, w.issues.issue.TargetDate = october(12), october(19)
			_, err := uc.Execute(as(alice), id, tt.p)
			unchanged(t, w, err, shared.Invalid(tt.want...), append(locked(id), tt.calls...))
		})
	}
}
````

`server/internal/modules/issue/app/fakes_test.go`（修改，2 处）：

````old server/internal/modules/issue/app/fakes_test.go
	f.issue.Name, f.issue.StateID, f.issue.Priority, f.issue.CompletedAt = r.Name, r.StateID, r.Priority, r.CompletedAt
````
````new server/internal/modules/issue/app/fakes_test.go
	f.issue.Name, f.issue.StateID, f.issue.Priority, f.issue.CompletedAt = r.Name, r.StateID, r.Priority, r.CompletedAt
	f.issue.StartDate, f.issue.TargetDate = r.StartDate, r.TargetDate
````
````old server/internal/modules/issue/app/fakes_test.go
		StateID: r.StateID, Priority: r.Priority, SortOrder: r.SortOrder, CompletedAt: r.CompletedAt, ArchivedAt: f.archivedAt,
````
````new server/internal/modules/issue/app/fakes_test.go
		StateID: r.StateID, Priority: r.Priority, StartDate: r.StartDate, TargetDate: r.TargetDate, SortOrder: r.SortOrder,
		CompletedAt: r.CompletedAt, ArchivedAt: f.archivedAt,
````

`server/internal/modules/issue/app/rows.go`（修改，3 处）：

````old server/internal/modules/issue/app/rows.go
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
````
````new server/internal/modules/issue/app/rows.go
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
````
````old server/internal/modules/issue/app/rows.go
	StateID     uuid.UUID
	Priority    domain.Priority
	SortOrder   float64
````
````new server/internal/modules/issue/app/rows.go
	StateID     uuid.UUID
	Priority    domain.Priority
	StartDate   *shared.Date
	TargetDate  *shared.Date
	SortOrder   float64
````
````old server/internal/modules/issue/app/rows.go
	StateID     uuid.UUID
	Priority    domain.Priority
	CompletedAt *time.Time
````
````new server/internal/modules/issue/app/rows.go
	StateID     uuid.UUID
	Priority    domain.Priority
	StartDate   *shared.Date
	TargetDate  *shared.Date
	CompletedAt *time.Time
````

`server/internal/modules/issue/app/update_issue.go`（修改，5 处）：

````old server/internal/modules/issue/app/update_issue.go
// issue.archived), a change of nothing too; what p changes in the work
// item (domain.ChangesOf), and when it changes nothing, nothing is written
// and the answer is the work item as read; the dependent rows it adds
// (check), every problem in one 422; the clock; the work item's fields,
// its completion by its new state's group when its state changes
// (domain.CompletedAt); the accounts and the labels it removes and adds,
// in id order; the accounts it adds subscribed; its activity
// (domain.UpdateActivity), each account by his display name. All of it is
// the caller's at that moment. The answer is the work item as stored, read
// back in the transaction.
````
````new server/internal/modules/issue/app/update_issue.go
// issue.archived), a change of nothing too; what p changes in the work item
// (domain.ChangesOf), and when it changes nothing, nothing is written and
// the answer is the work item as read; the dependent rows it adds and the
// order of the dates it leaves (check), every problem in one 422; the
// clock; the work item's fields, its completion by its new state's group
// when its state changes (domain.CompletedAt); the accounts and the labels
// it removes and adds, in id order; the accounts it adds subscribed; its
// activity (domain.UpdateActivity), each account by his display name. All
// of it is the caller's at that moment. The answer is the work item as
// stored, read back in the transaction.
````
````old server/internal/modules/issue/app/update_issue.go
// deleting a label takes it off every work item (DetachLabel). Every
// problem is reported at once, in one 422, each at its place in p.
````
````new server/internal/modules/issue/app/update_issue.go
// deleting a label takes it off every work item (DetachLabel). Then the
// dates p leaves the work item with are in order, merged with those it
// keeps (domain.DatesOutOfOrder). Every problem is reported at once, in
// one 422, each at its place in p.
````
````old server/internal/modules/issue/app/update_issue.go
			}
		}
````
````new server/internal/modules/issue/app/update_issue.go
			}
		}
	}
	if f := domain.DatesOutOfOrder(i, p); f != nil {
		problems = append(problems, *f)
````
````old server/internal/modules/issue/app/update_issue.go
	f := IssueFields{ID: i.ID, Name: i.Name, StateID: i.StateID, Priority: i.Priority, CompletedAt: i.CompletedAt, By: s.by, At: s.at}
````
````new server/internal/modules/issue/app/update_issue.go
	f := IssueFields{ID: i.ID, Name: i.Name, StateID: i.StateID, Priority: i.Priority, StartDate: i.StartDate, TargetDate: i.TargetDate,
		CompletedAt: i.CompletedAt, By: s.by, At: s.at}
````
````old server/internal/modules/issue/app/update_issue.go
		f.StateID, f.CompletedAt = *c.StateID, domain.CompletedAt(n.group, s.at)
````
````new server/internal/modules/issue/app/update_issue.go
		f.StateID, f.CompletedAt = *c.StateID, domain.CompletedAt(n.group, s.at)
	}
	if c.StartDate != nil {
		f.StartDate, activity.StartDate = c.StartDate.To, &domain.Changed[*shared.Date]{Old: i.StartDate, New: c.StartDate.To}
	}
	if c.TargetDate != nil {
		f.TargetDate, activity.TargetDate = c.TargetDate.To, &domain.Changed[*shared.Date]{Old: i.TargetDate, New: c.TargetDate.To}
````

- [ ] **Step 4: 存储**

`server/internal/modules/issue/adapter/postgres/issues.go`（修改，4 处）：

````old server/internal/modules/issue/adapter/postgres/issues.go
	"fmt"
````
````new server/internal/modules/issue/adapter/postgres/issues.go
	"fmt"
	"time"
````
````old server/internal/modules/issue/adapter/postgres/issues.go
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
````
````new server/internal/modules/issue/adapter/postgres/issues.go
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
````
````old server/internal/modules/issue/adapter/postgres/issues.go
		SequenceID: int32(r.SequenceID), SortOrder: r.SortOrder, CompletedAt: r.CompletedAt, By: &r.By, At: r.At,
````
````new server/internal/modules/issue/adapter/postgres/issues.go
		StartDate: timeOf(r.StartDate), TargetDate: timeOf(r.TargetDate), SequenceID: int32(r.SequenceID), SortOrder: r.SortOrder,
		CompletedAt: r.CompletedAt, By: &r.By, At: r.At,
````
````old server/internal/modules/issue/adapter/postgres/issues.go
		Priority: domain.Priority(r.Priority), ParentID: r.ParentID, SortOrder: r.SortOrder, CompletedAt: r.CompletedAt,
		ArchivedAt: r.ArchivedAt, AssigneeIDs: r.AssigneeIds, LabelIDs: r.LabelIds, CreatedByID: r.CreatedByID,
		UpdatedByID: r.UpdatedByID, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}, true, nil
}

````
````new server/internal/modules/issue/adapter/postgres/issues.go
		Priority: domain.Priority(r.Priority), StartDate: dayOf(r.StartDate), TargetDate: dayOf(r.TargetDate), ParentID: r.ParentID,
		SortOrder: r.SortOrder, CompletedAt: r.CompletedAt, ArchivedAt: r.ArchivedAt, AssigneeIDs: r.AssigneeIds, LabelIDs: r.LabelIds,
		CreatedByID: r.CreatedByID, UpdatedByID: r.UpdatedByID, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}, true, nil
}

// dayOf is the day of a date column as pgx reads it, midnight UTC; nil for
// NULL.
func dayOf(t *time.Time) *shared.Date {
	if t == nil {
		return nil
	}
	d := shared.DateOf(*t)
	return &d
}

// timeOf is d as a date column is written, its midnight UTC; nil for none.
func timeOf(d *shared.Date) *time.Time {
	if d == nil {
		return nil
	}
	t := d.Time()
	return &t
}

````

`server/internal/modules/issue/adapter/postgres/issues_test.go`（修改，6 处）：

````old server/internal/modules/issue/adapter/postgres/issues_test.go
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
````
````new server/internal/modules/issue/adapter/postgres/issues_test.go
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
````
````old server/internal/modules/issue/adapter/postgres/issues_test.go
// now, high, in web's default state, at 75535, completed at an hour
// before: every column createIssue writes, none at its default.
````
````new server/internal/modules/issue/adapter/postgres/issues_test.go
// now, high, in web's default state, from the 12th to the 19th of October,
// at 75535, completed at an hour before: every column createIssue writes,
// none at its default.
````
````old server/internal/modules/issue/adapter/postgres/issues_test.go
	done := now.Add(-time.Hour)
````
````new server/internal/modules/issue/adapter/postgres/issues_test.go
	done := now.Add(-time.Hour)
	start, target := day(12), day(19)
````
````old server/internal/modules/issue/adapter/postgres/issues_test.go
		Priority: domain.PriorityHigh, SortOrder: 75535, CompletedAt: &done, By: alice, At: now}
````
````new server/internal/modules/issue/adapter/postgres/issues_test.go
		Priority: domain.PriorityHigh, StartDate: &start, TargetDate: &target, SortOrder: 75535, CompletedAt: &done, By: alice, At: now}
````
````old server/internal/modules/issue/adapter/postgres/issues_test.go
		t.Fatal(err)
	}
	return r
}
````
````new server/internal/modules/issue/adapter/postgres/issues_test.go
		t.Fatal(err)
	}
	return r
}

// day is the day d of October 2026, a work item's date.
func day(d int) shared.Date {
	return shared.DateOf(time.Date(2026, time.October, d, 0, 0, 0, 0, time.UTC))
}
````
````old server/internal/modules/issue/adapter/postgres/issues_test.go
		Priority: domain.PriorityHigh, SortOrder: 75535, CompletedAt: r.CompletedAt, CreatedByID: &alice, UpdatedByID: &alice, CreatedAt: now,
		UpdatedAt: now, AssigneeIDs: slices.SortedFunc(slices.Values([]uuid.UUID{bob, carol}), uuid.UUID.Compare),
		LabelIDs: slices.SortedFunc(slices.Values([]uuid.UUID{bug, ui}), uuid.UUID.Compare)}
````
````new server/internal/modules/issue/adapter/postgres/issues_test.go
		Priority: domain.PriorityHigh, StartDate: r.StartDate, TargetDate: r.TargetDate, SortOrder: 75535, CompletedAt: r.CompletedAt,
		CreatedByID: &alice, UpdatedByID: &alice, CreatedAt: now, UpdatedAt: now,
		AssigneeIDs: slices.SortedFunc(slices.Values([]uuid.UUID{bob, carol}), uuid.UUID.Compare),
		LabelIDs:    slices.SortedFunc(slices.Values([]uuid.UUID{bug, ui}), uuid.UUID.Compare)}
````

`server/internal/modules/issue/adapter/postgres/update.go`（修改，1 处）：

````old server/internal/modules/issue/adapter/postgres/update.go
		ID: f.ID, Name: f.Name, StateID: f.StateID, Priority: string(f.Priority), CompletedAt: f.CompletedAt, By: f.By, At: f.At,
````
````new server/internal/modules/issue/adapter/postgres/update.go
		ID: f.ID, Name: f.Name, StateID: f.StateID, Priority: string(f.Priority), StartDate: timeOf(f.StartDate),
		TargetDate: timeOf(f.TargetDate), CompletedAt: f.CompletedAt, By: f.By, At: f.At,
````

`server/internal/modules/issue/adapter/postgres/update_test.go`（修改，3 处）：

````old server/internal/modules/issue/adapter/postgres/update_test.go
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
````
````new server/internal/modules/issue/adapter/postgres/update_test.go
// the work item's name, state, priority, dates and completion as given, its
// start date given nil NULL, last changed by bob at later, its number, its
// sort order and its creation as they were; RemoveAssignees and
// RemoveLabels soft-delete, by bob at later, the work item's undeleted rows
// of every account and label given, two of each in one call, every other
// column kept. Nothing else changes: not another work item, nor its rows of
// a given account and label, nor the work item's rows of an account or a
// label not given, nor its rows of a given account and label deleted an
// hour before, which keep their stamps. Moved back into a state of no
// completed group, its completion given nil, the work item is completed no
// more.
````
````old server/internal/modules/issue/adapter/postgres/update_test.go
	if err := s.UpdateIssue(ctx, app.IssueFields{ID: id, Name: "Mend the login", StateID: done, Priority: domain.PriorityLow,
````
````new server/internal/modules/issue/adapter/postgres/update_test.go
	target := day(26)
	if err := s.UpdateIssue(ctx, app.IssueFields{ID: id, Name: "Mend the login", StateID: done, Priority: domain.PriorityLow, TargetDate: &target,
````
````old server/internal/modules/issue/adapter/postgres/update_test.go
		Priority: domain.PriorityLow, SortOrder: 75535, CompletedAt: &later, CreatedByID: &alice, UpdatedByID: &bob, CreatedAt: now,
		UpdatedAt: later, AssigneeIDs: []uuid.UUID{carol}, LabelIDs: []uuid.UUID{bug}}
````
````new server/internal/modules/issue/adapter/postgres/update_test.go
		Priority: domain.PriorityLow, TargetDate: &target, SortOrder: 75535, CompletedAt: &later, CreatedByID: &alice, UpdatedByID: &bob,
		CreatedAt: now, UpdatedAt: later, AssigneeIDs: []uuid.UUID{carol}, LabelIDs: []uuid.UUID{bug}}
````

- [ ] **Step 5: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/...`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 6: 提交**

```bash
git add server/internal/modules/issue/adapter/postgres/issues.go server/internal/modules/issue/adapter/postgres/issues_test.go server/internal/modules/issue/adapter/postgres/queries/issues.sql server/internal/modules/issue/adapter/postgres/queries/update.sql server/internal/modules/issue/adapter/postgres/update.go server/internal/modules/issue/adapter/postgres/update_test.go server/internal/modules/issue/app/create_issue.go server/internal/modules/issue/app/create_issue_test.go server/internal/modules/issue/app/dates_test.go server/internal/modules/issue/app/fakes_test.go server/internal/modules/issue/app/rows.go server/internal/modules/issue/app/update_issue.go server/internal/modules/issue/domain/activity.go server/internal/modules/issue/domain/activity_test.go server/internal/modules/issue/domain/dates.go server/internal/modules/issue/domain/dates_test.go server/internal/modules/issue/domain/issue.go server/internal/modules/issue/domain/issue_test.go server/internal/modules/issue/domain/patch.go server/internal/modules/issue/domain/patch_test.go server/sqlc.yaml server/internal/modules/issue/adapter/postgres/gen/issues.sql.go server/internal/modules/issue/adapter/postgres/gen/models.go server/internal/modules/issue/adapter/postgres/gen/update.sql.go
```
```bash
git commit -m "feat(M4/P2): a work item keeps its start and target dates, in order

The domain, the use cases and the store carry the start and target
dates as shared.Date (sqlc reads a date as midnight UTC, the store turns
it into the day). A creation checks its two dates as given before any
read; an update checks the dates the work item will have, the patch's
over the work item's, under the lock after the dependent rows, in the
same 422: at start_date when the start date is given alone, else at
target_date. Each date changed writes its activity row, 8 or 9, with
Plane's comment.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p2.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试，或没有替身的平台、领域、转换的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是生成器（`make gen-go`）和中英文翻译键（`make lint-web`）的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过，端到端的检查只在标出的 Task 的树上）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T10.1` | 新工作项不查日期 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDatesAreKeptInOrder`（`make test`）（Task 11 起）、故事 `i1-create-issue`（Task 13 起） | 单元；组合；端到端 |
| `T10.2` | 两个日期同一天算顺序不对 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDatesAreKeptInOrder`（`make test`）（Task 11 起） | 单元；组合 |
| `T10.3` | 修改只看它给出的日期，不与工作项留着的合并 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDatesAreKeptInOrder`（`make test`）（Task 11 起）、故事 `i2-update-issue`（Task 13 起） | 单元；组合；端到端 |
| `T10.4` | 给出的截止日期不合并：修改与工作项原来的截止日期比较 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDatesAreKeptInOrder`（`make test`）（Task 11 起）、故事 `i2-update-issue`（Task 13 起） | 单元；组合；端到端 |
| `T10.5` | 只给开始日期时也答在 `target_date` | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDatesAreKeptInOrder`（`make test`）（Task 11 起）、故事 `i2-update-issue`（Task 13 起） | 单元；组合；端到端 |
| `T10.6` | 合并之后的日期只在修改两个都给时检查（照 Plane） | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDatesAreKeptInOrder`（`make test`）（Task 11 起）、故事 `i2-update-issue`（Task 13 起） | 单元；组合；端到端 |
| `T10.7` | 给出原样的开始日期也算改动 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDatesAreKeptInOrder`（`make test`）（Task 11 起） | 单元；组合 |
| `T10.8` | 任何两天都算同一天：移动了的日期不算改动 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDatesAreKeptInOrder`（`make test`）（Task 11 起）、故事 `i2-update-issue`（Task 13 起） | 单元；组合；端到端 |
| `T10.9` | 改了的开始日期只写进动态，不写进工作项 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDatesAreKeptInOrder`（`make test`）（Task 11 起）、故事 `i2-update-issue`（Task 13 起） | 单元；组合；端到端 |
| `T10.10` | 第 8 行的 `comment` 没有结尾的空格（与 Plane 不同） | `internal/modules/issue/domain` 的全部测试（`make test`）、故事 `i2-update-issue`（Task 13 起） | 单元；端到端 |
| `T10.11` | 清空的日期在它的一行里是 `None`，不是 `""` | `internal/modules/issue/domain` 的全部测试（`make test`）、故事 `i2-update-issue`（Task 13 起） | 单元；端到端 |
| `T10.12` | 修改把开始日期写成截止日期 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemReadsAsStored`（`make test`）（Task 11 起）、`internal/bootstrap` 的 `TestDatesAreKeptInOrder`（`make test`）（Task 11 起）、故事 `i2-update-issue`（Task 13 起） | 存储；组合；端到端 |
| `T10.13` | 工作项把截止日期也读成开始日期 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemReadsAsStored`（`make test`）（Task 11 起）、故事 `i1-create-issue`（Task 13 起） | 存储；组合；端到端 |
| `T10.14` | 新工作项把截止日期也存成开始日期 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDatesAreKeptInOrder`（`make test`）（Task 11 起）、故事 `i1-create-issue`（Task 13 起） | 存储；组合；端到端 |

---

### Task 11: 起止日期：接口、组合和矩阵

**Files:**
- Create: `server/internal/bootstrap/issue_dates_test.go`、`server/internal/modules/issue/adapter/http/dates_test.go`
- Modify: `api/modules/issue.yaml`、`server/internal/bootstrap/issue_updates_test.go`、`server/internal/bootstrap/permission_matrix_issue_updates_test.go`、`server/internal/bootstrap/permission_matrix_issues_test.go`、`server/internal/modules/issue/adapter/http/issues.go`、`server/internal/modules/issue/adapter/http/issues_test.go`、`server/internal/modules/issue/adapter/http/update_test.go`、`server/internal/modules/issue/app/update_issue_test.go`、`server/internal/platform/httpserver/bodyshape/bodyshape.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/issue/adapter/http/gen/bodyshape.gen.go`、`server/internal/modules/issue/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**（spec 2.12、第 3 节第 10、11 条；M4 设计 3.7、5.2、6.1）
- 契约：`IssueDetail` 的 `start_date`、`target_date`（必有，可为 `null`，`format: date`）；`IssueCreate` 的（可省略，不可为 `null`）；`IssueUpdate` 的（可为 `null`：清空）；`createIssue`、`updateIssue` 的描述写上日期的规则（创建只看请求；修改按合并之后的值，在锁下；`updateIssue` 的"值照 `createIssue` 的规则"一句除去日期的次序）。生成的 `IssueCreate`、`IssueUpdate`、`IssueDetail` 的日期是 `shared.Date`（Task 9 的映射），`bodyshape.gen.go` 用 `FormatDate` 检查它们。
- `issue/adapter/http/issues.go`：`given[T](nullable.Nullable[T]) (bool, *T)`（没给、给 `null`、给值）；创建、修改、回答的映射（`orNull`）。
- `bodyshape.go` 的 `checkDate` 的说明指向钉住两者一致的测试。

**Tests:**
- HTTP：`TestDatesPassTheBodyCheckAsTheyDecode`（`dates_test.go`，新：`bodyshape` 放过的与生成的类型解码得了的是同一组，作为创建的和修改的开始日期各发一次：2026-10-19、0001-01-01、9999-12-31、闰年的 2 月 29 日、用转义写的日期，与平年的 2 月 29 日、4 月 31 日、0 月、13 月、0000 年、少一位的月或日、两位的年、斜线、带时间、带时区、前后的空白、带符号、五位的年、全角数字、空串）；`TestCreateIssue`、`TestUpdateIssue`（日期交给用例，修改给 `null` 是清空；回答的两个日期，没有的是 `null`）；`TestCreateIssueHoldsTheBodyToItsStructure`、`TestUpdateIssueHoldsTheBodyToItsStructure`（不是日期的、带时间的 400；创建给 `null` 的日期 400）。
- 用例（单元）：`TestUpdateIssueRefusesUnderTheLock` 加"已归档的工作项给次序不对的日期"一行（409 `issue.archived`，先于日期的次序；调用正是 `locked`）；`TestUpdateIssueThatChangesNothingWritesNothing` 的"每个字段原样"也给出两个日期，照工作项的（没有）。
- 组合：`TestDatesAreKeptInOrder`（`issue_dates_test.go`，新：alice 建 10 月 12 日到 19 日的工作项；bob 只给晚于截止的开始日期（11 月 1 日：月份晚、日子早）、只给早于开始的截止日期，各 422，在 `start_date`、`target_date`，什么都不写；两个都移过它原来的截止日期、再清空截止日期，各写一行动态；再给它已有的日期，什么都不改、不写一行；alice 把它移进 Done、归档之后，bob 给早于开始日期的截止日期，409 `issue.archived`（先于日期的次序），什么都不写；每个回答的日期与数据库的相同；开始晚于截止的创建（11 月 1 日到 10 月 19 日）422、不占编号，同一天的允许）；`TestAWorkItemReadsAsStored`（加两个日期）；`TestPermissionMatrix` 的 `createIssue`、`updateIssue` 的"每个字段都给"带日期，`updateIssue` 加"日期的次序不对"一行；`TestBodiesThatBreakTheStructureAnswer400`。
- 端到端：现有的 122 个照样通过（日期进了契约、创建和修改的映射）。

- [ ] **Step 1: 契约，生成**

`api/modules/issue.yaml`（修改，7 处）：

````old api/modules/issue.yaml
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
        assignee. Every row it writes is the caller's, at one moment. A
        project with too many writes on its work items at once answers
        server_busy, with Retry-After: try again then.
````
````new api/modules/issue.yaml
        (project.archived). The name, without the white space around it, has
        1–255 characters, without NUL, and is stored so; at most 100
        assignees and 100 labels, each named once; the start date not after
        the target date (validation_failed), checked before the project is
        looked at. A project that does not exist, is deleted, or that the
        caller does not see answers project.not_found; one he sees but may
        not create work items in, forbidden; both are answered before the
        project is locked, and with the project locked the role is decided
        again. Then the state, when it is given, is one of the project's,
        not its triage state; each assignee is an active member of the
        project who is not its guest; each label is one of the project's
        (validation_failed, every problem at once). Left out, the state is
        the project's default, the priority none, and the assignee the
        project's default assignee while he is still an active member of it
        who is not its guest, else no one. The work item takes the project's
        next number, never used again, also after a deletion; it comes after
        its state's work items, archived ones too: its sort order is their
        greatest plus 10000, or 65535 for the state's first; in a state of
        the completed group it is completed at its creation. The caller and
        the assignees subscribe to it; its activity says it was created, and
        assigned to each assignee. Every row it writes is the caller's, at
        one moment. A project with too many writes on its work items at once
        answers server_busy, with Retry-After: try again then.
````
````old api/modules/issue.yaml
        for none. The values follow createIssue's rules (validation_failed),
        checked before the work item is looked at. A work item that does not
        exist, is deleted, whose project the caller does not see, or that he
        did not create as a guest who sees only his own (getIssue) answers
        issue.not_found, whatever his role may do; one whose project he sees
        but whose work items he may not change, forbidden; both are answered
        before the project is locked, and with the project locked the role
        is decided again. A work item of an archived project cannot be
        changed (project.archived), nor then an archived work item
        (issue.archived), even by a change that changes nothing. Then the
        state, when it changes, is one of the project's, not its triage
        state; each account the work item is newly assigned to is an active
        member of the project who is not its guest, while one it keeps stays
        assigned whatever has become of him; each label added is one of the
        project's (validation_failed, every problem at once). A change that
        changes nothing writes nothing, and the work item is answered as it
        is. Otherwise the work item is changed last by the caller; moved
        into a state of the completed group, it is completed at the change,
        and moved into another, not completed; the accounts newly assigned
````
````new api/modules/issue.yaml
        for none. The values follow createIssue's rules, but for the order of
        the dates (validation_failed), checked before the work item is looked
        at. A work item that does not exist, is deleted, whose project the
        caller does not see, or that he did not create as a guest who sees
        only his own (getIssue) answers issue.not_found, whatever his role
        may do; one whose project he sees but whose work items he may not
        change, forbidden; both are answered before the project is locked,
        and with the project locked the role is decided again. A work item of
        an archived project cannot be changed (project.archived), nor then an
        archived work item (issue.archived), even by a change that changes
        nothing. Then the state, when it changes, is one of the project's,
        not its triage state; each account the work item is newly assigned to
        is an active member of the project who is not its guest, while one it
        keeps stays assigned whatever has become of him; each label added is
        one of the project's; the start date the work item then has is not
        after its target date, each as given or, not given, as it is: at
        start_date when the start date alone is given, else at target_date
        (validation_failed, every problem at once). A change that changes
        nothing writes nothing, and the work item is answered as it is.
        Otherwise the work item is changed last by the caller; moved into a
        state of the completed group, it is completed at the change, and
        moved into another, not completed; the accounts newly assigned
````
````old api/modules/issue.yaml
        priority, the state, and each label and each assignee added and
        removed. Every row it writes is the caller's, at one moment. A
        project with too many writes on its work items at once answers
        server_busy, with Retry-After: try again then.
````
````new api/modules/issue.yaml
        priority, the state, the start and the target dates, and each label
        and each assignee added and removed. Every row it writes is the
        caller's, at one moment. A project with too many writes on its work
        items at once answers server_busy, with Retry-After: try again then.
````
````old api/modules/issue.yaml
      required: [id, workspace_id, project_id, sequence_id, name, state_id, priority, parent_id, sort_order, completed_at, archived_at,
        assignee_ids, label_ids, created_by_id, updated_by_id, created_at, updated_at]
````
````new api/modules/issue.yaml
      required: [id, workspace_id, project_id, sequence_id, name, state_id, priority, start_date, target_date, parent_id, sort_order,
        completed_at, archived_at, assignee_ids, label_ids, created_by_id, updated_by_id, created_at, updated_at]
````
````old api/modules/issue.yaml
        priority:
          $ref: '#/components/schemas/IssuePriority'
        parent_id:
````
````new api/modules/issue.yaml
        priority:
          $ref: '#/components/schemas/IssuePriority'
        start_date:
          description: The day it starts; null for none.
          type: [string, 'null']
          format: date
        target_date:
          description: The day it is due, not before its start date; null for none.
          type: [string, 'null']
          format: date
        parent_id:
````
````old api/modules/issue.yaml
            type: string
            format: uuid
    IssueUpdate:
````
````new api/modules/issue.yaml
            type: string
            format: uuid
        start_date:
          description: The day it starts, as YYYY-MM-DD; left out for none.
          type: string
          format: date
        target_date:
          description: The day it is due, as YYYY-MM-DD, not before start_date; left out for none.
          type: string
          format: date
    IssueUpdate:
````
````old api/modules/issue.yaml
            project's labels, each named once.
          type: array
          items:
            type: string
            format: uuid

````
````new api/modules/issue.yaml
            project's labels, each named once.
          type: array
          items:
            type: string
            format: uuid
        start_date:
          description: >-
            The day it starts, as YYYY-MM-DD; null for none. Not after the
            target date the work item then has.
          type: [string, 'null']
          format: date
        target_date:
          description: >-
            The day it is due, as YYYY-MM-DD; null for none. Not before the
            start date the work item then has.
          type: [string, 'null']
          format: date

````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `0ddf0747329150a2323774dc11f1414f5f5af576439dd7686391a124ae421e66` | 3334 | `api/dist/openapi.yaml` |
| `132fca761648f96c2b192144536e767bf75a327e06758c95fdbf39364a336f40` | 38 | `server/internal/modules/issue/adapter/http/gen/bodyshape.gen.go` |
| `4607cfafc7f95dda5fec3fefa39152bee1444226657f4af444f1bbb66b69f7d4` | 1112 | `server/internal/modules/issue/adapter/http/gen/server.gen.go` |
| `9b409bb526b4ba3083483f44fc6e607a49990079b8b91dc717077fc3949697bb` | 3696 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/issue/adapter/http/gen/bodyshape.gen.go server/internal/modules/issue/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: HTTP 和用例的测试**

`server/internal/modules/issue/adapter/http/dates_test.go`（新文件，39 行）：

````file server/internal/modules/issue/adapter/http/dates_test.go
package httpadapter_test

import (
	"encoding/json"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
)

// A date passes the body check exactly when the generated types decode it
// (M4 design 6.1): bodyshape reads a date by a rule of its own (checkDate),
// since the platform may not import shared, whose Date the generated
// IssueCreate and IssueUpdate decode. Each sample is a day, one that is no
// day, or one in another layout, with a time or a zone, with white space,
// or written with an escape; each goes as a new work item's start date and
// as a change's.
func TestDatesPassTheBodyCheckAsTheyDecode(t *testing.T) {
	shapes := gen.BodyShapes()
	for raw, want := range map[string]bool{
		`"2026-10-19"`: true, `"0001-01-01"`: true, `"9999-12-31"`: true, `"2024-02-29"`: true, `"2026-10-1\u0039"`: true,
		`"2026-02-29"`: false, `"2026-04-31"`: false, `"2026-00-10"`: false, `"2026-13-01"`: false, `"0000-12-31"`: false,
		`"2026-1-19"`: false, `"2026-10-9"`: false, `"26-10-19"`: false, `"2026/10/19"`: false, `"2026-10-19T00:00:00Z"`: false,
		`"2026-10-19+08:00"`: false, `" 2026-10-19"`: false, `"2026-10-19\n"`: false, `"+2026-10-19"`: false, `"10000-01-01"`: false,
		`"２０２６-10-19"`: false, `""`: false,
	} {
		for _, tt := range []struct {
			route, body string
			into        any
		}{
			{"POST /api/v0/projects/{project_id}/issues", `{"name":"A","start_date":` + raw + `}`, new(gen.IssueCreate)},
			{"PATCH /api/v0/issues/{issue_id}", `{"start_date":` + raw + `}`, new(gen.IssueUpdate)},
		} {
			checked, decoded := shapes.Check(tt.route, []byte(tt.body)), json.Unmarshal([]byte(tt.body), tt.into)
			if (checked == nil) != want || (decoded == nil) != want {
				t.Errorf("%s %s: the body check %v, the decoder %v; want both to accept it: %v", tt.route, tt.body, checked, decoded, want)
			}
		}
	}
}
````

`server/internal/modules/issue/adapter/http/issues.go`（修改，6 处）：

````old server/internal/modules/issue/adapter/http/issues.go
// CreateIssue serves POST /api/v0/projects/{project_id}/issues: the state
// and the priority nil and the lists empty when not given; a value given
// goes to the use case as it came, which checks it.
````
````new server/internal/modules/issue/adapter/http/issues.go
// CreateIssue serves POST /api/v0/projects/{project_id}/issues: the state,
// the priority and the dates nil and the lists empty when not given; a
// value given goes to the use case as it came, which checks it.
````
````old server/internal/modules/issue/adapter/http/issues.go
	in := domain.NewIssue{Name: req.Body.Name, StateID: req.Body.StateID, Priority: (*domain.Priority)(req.Body.Priority)}
````
````new server/internal/modules/issue/adapter/http/issues.go
	in := domain.NewIssue{Name: req.Body.Name, StateID: req.Body.StateID, Priority: (*domain.Priority)(req.Body.Priority),
		StartDate: req.Body.StartDate, TargetDate: req.Body.TargetDate}
````
````old server/internal/modules/issue/adapter/http/issues.go
// not given; a value given goes to the use case as it came, which checks
// it.
````
````new server/internal/modules/issue/adapter/http/issues.go
// not given, a date given null given as none; a value given goes to the
// use case as it came, which checks it.
````
````old server/internal/modules/issue/adapter/http/issues.go
		AssigneeIDs: req.Body.AssigneeIds, LabelIDs: req.Body.LabelIds}
````
````new server/internal/modules/issue/adapter/http/issues.go
		AssigneeIDs: req.Body.AssigneeIds, LabelIDs: req.Body.LabelIds}
	p.SetStartDate, p.StartDate = given(req.Body.StartDate)
	p.SetTargetDate, p.TargetDate = given(req.Body.TargetDate)
````
````old server/internal/modules/issue/adapter/http/issues.go
		StateID: i.StateID, Priority: gen.IssuePriority(i.Priority), ParentID: orNull(i.ParentID), SortOrder: i.SortOrder,
````
````new server/internal/modules/issue/adapter/http/issues.go
		StateID: i.StateID, Priority: gen.IssuePriority(i.Priority), StartDate: orNull(i.StartDate), TargetDate: orNull(i.TargetDate),
		ParentID: orNull(i.ParentID), SortOrder: i.SortOrder,
````
````old server/internal/modules/issue/adapter/http/issues.go
		CreatedByID: orNull(i.CreatedByID), UpdatedByID: orNull(i.UpdatedByID), CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt}
````
````new server/internal/modules/issue/adapter/http/issues.go
		CreatedByID: orNull(i.CreatedByID), UpdatedByID: orNull(i.UpdatedByID), CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt}
}

// given reports whether a body gives v, and its value: nil when it gives
// null.
func given[T any](v nullable.Nullable[T]) (bool, *T) {
	switch {
	case !v.IsSpecified():
		return false, nil
	case v.IsNull():
		return true, nil
	}
	x := v.MustGet()
	return true, &x
````

`server/internal/modules/issue/adapter/http/issues_test.go`（修改，10 处）：

````old server/internal/modules/issue/adapter/http/issues_test.go
// The work items the use case answers. WEB-7, under WEB-6, completed, with
// an assignee and a label, made by alice, archived half a minute later and
// changed last by bob a minute after its creation: each field that may be
// null has a value of its own, so each shows as its own. WEB-8, new, with
// none of them, without a creator or a last writer (created_by_id and
// updated_by_id null).
````
````new server/internal/modules/issue/adapter/http/issues_test.go
// The work items the use case answers. WEB-7, under WEB-6, from the 12th
// to the 19th of October, completed, with an assignee and a label, made by
// alice, archived half a minute later and changed last by bob a minute
// after its creation: each field that may be null has a value of its own,
// so each shows as its own. WEB-8, new, with none of them, without a
// creator or a last writer (created_by_id and updated_by_id null).
````
````old server/internal/modules/issue/adapter/http/issues_test.go
	webSixID  = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000e6")
````
````new server/internal/modules/issue/adapter/http/issues_test.go
	webSixID  = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000e6")
	startDay  = shared.DateOf(time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC))
	targetDay = shared.DateOf(time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC))
````
````old server/internal/modules/issue/adapter/http/issues_test.go
		Name: "Fix the login", StateID: stateID, Priority: domain.PriorityHigh, ParentID: &webSixID, SortOrder: 75535.5,
		CompletedAt: &completed, ArchivedAt: &archived, AssigneeIDs: []uuid.UUID{bobID}, LabelIDs: []uuid.UUID{bugID},
````
````new server/internal/modules/issue/adapter/http/issues_test.go
		Name: "Fix the login", StateID: stateID, Priority: domain.PriorityHigh, StartDate: &startDay, TargetDate: &targetDay, ParentID: &webSixID,
		SortOrder: 75535.5, CompletedAt: &completed, ArchivedAt: &archived, AssigneeIDs: []uuid.UUID{bobID}, LabelIDs: []uuid.UUID{bugID},
````
````old server/internal/modules/issue/adapter/http/issues_test.go
		`"priority":"high","project_id":"0199a2b4-0000-7000-8000-0000000000a1","sequence_id":7,"sort_order":75535.5,` +
		`"state_id":"0199a2b4-0000-7000-8000-0000000000c1","updated_at":"2026-10-17T10:01:00.123456Z",` +
````
````new server/internal/modules/issue/adapter/http/issues_test.go
		`"priority":"high","project_id":"0199a2b4-0000-7000-8000-0000000000a1","sequence_id":7,"sort_order":75535.5,"start_date":"2026-10-12",` +
		`"state_id":"0199a2b4-0000-7000-8000-0000000000c1","target_date":"2026-10-19","updated_at":"2026-10-17T10:01:00.123456Z",` +
````
````old server/internal/modules/issue/adapter/http/issues_test.go
		`"project_id":"0199a2b4-0000-7000-8000-0000000000a1","sequence_id":8,"sort_order":65535,` +
		`"state_id":"0199a2b4-0000-7000-8000-0000000000c1","updated_at":"2026-10-17T10:00:00.123456Z","updated_by_id":null,` +
````
````new server/internal/modules/issue/adapter/http/issues_test.go
		`"project_id":"0199a2b4-0000-7000-8000-0000000000a1","sequence_id":8,"sort_order":65535,"start_date":null,` +
		`"state_id":"0199a2b4-0000-7000-8000-0000000000c1","target_date":null,"updated_at":"2026-10-17T10:00:00.123456Z","updated_by_id":null,` +
````
````old server/internal/modules/issue/adapter/http/issues_test.go
// the body's fields: the state and the priority nil and the lists empty
// when not given; the answer is 201 with the work item the use case
// answers, every field of it, null for each it has none of, [] for a list
// without ids.
````
````new server/internal/modules/issue/adapter/http/issues_test.go
// the body's fields: the state, the priority and the dates nil and the
// lists empty when not given; the answer is 201 with the work item the use
// case answers, every field of it, null for each it has none of, [] for a
// list without ids.
````
````old server/internal/modules/issue/adapter/http/issues_test.go
			`"assignee_ids":["0199a2b4-0000-7000-8000-000000000002"],"label_ids":["0199a2b4-0000-7000-8000-0000000000d1"]}`,
````
````new server/internal/modules/issue/adapter/http/issues_test.go
			`"assignee_ids":["0199a2b4-0000-7000-8000-000000000002"],"label_ids":["0199a2b4-0000-7000-8000-0000000000d1"],` +
			`"start_date":"2026-10-12","target_date":"2026-10-19"}`,
````
````old server/internal/modules/issue/adapter/http/issues_test.go
				LabelIDs: []uuid.UUID{bugID}}, webSeven, webSevenJSON},
````
````new server/internal/modules/issue/adapter/http/issues_test.go
				LabelIDs: []uuid.UUID{bugID}, StartDate: &startDay, TargetDate: &targetDay}, webSeven, webSevenJSON},
````
````old server/internal/modules/issue/adapter/http/issues_test.go
// comes in P4), a field of another type, or an id that is no id: refused as
// bad_request before the use case.
````
````new server/internal/modules/issue/adapter/http/issues_test.go
// comes in P4), null for a field that is not nullable (a date left out for
// none), a field of another type, an id that is no id, or a date that is
// no day or has a time: refused as bad_request before the use case.
````
````old server/internal/modules/issue/adapter/http/issues_test.go
		`{"name":"A","assignee_ids":"bob"}`, `{"name":"A","label_ids":["Bug"]}`} {
````
````new server/internal/modules/issue/adapter/http/issues_test.go
		`{"name":"A","target_date":null}`, `{"name":"A","assignee_ids":"bob"}`, `{"name":"A","label_ids":["Bug"]}`,
		`{"name":"A","start_date":"2026-02-30"}`, `{"name":"A","target_date":"2026-10-19T00:00:00Z"}`} {
````

`server/internal/modules/issue/adapter/http/update_test.go`（修改，4 处）：

````old server/internal/modules/issue/adapter/http/update_test.go
// the body's fields, each nil when not given, a list given empty given; the
// answer is 200 with the work item the use case answers, as POST shows it.
````
````new server/internal/modules/issue/adapter/http/update_test.go
// the body's fields, each nil when not given, a list given empty given, a
// date given null given as none; the answer is 200 with the work item the
// use case answers, as POST shows it.
````
````old server/internal/modules/issue/adapter/http/update_test.go
		{`{"assignee_ids":[],"label_ids":[]}`, domain.IssuePatch{AssigneeIDs: &none, LabelIDs: &none}},
````
````new server/internal/modules/issue/adapter/http/update_test.go
		{`{"assignee_ids":[],"label_ids":[]}`, domain.IssuePatch{AssigneeIDs: &none, LabelIDs: &none}},
		{`{"start_date":"2026-10-12","target_date":null}`, domain.IssuePatch{SetStartDate: true, StartDate: &startDay, SetTargetDate: true}},
		{`{"start_date":null,"target_date":"2026-10-19"}`, domain.IssuePatch{SetStartDate: true, SetTargetDate: true, TargetDate: &targetDay}},
````
````old server/internal/modules/issue/adapter/http/update_test.go
// a sort order, a parent, a description, a date, which IssueUpdate has not
// in P1b: M4/P1b spec 3), null for a field that is not nullable, a field of
// another type, or an id that is no id: refused as bad_request before the
// use case.
````
````new server/internal/modules/issue/adapter/http/update_test.go
// a sort order, a parent, a description, which IssueUpdate has not in P2:
// M4 design 12, constraint 5), null for a field that is not nullable, a
// field of another type, an id that is no id, or a date that is no day or
// has a time: refused as bad_request before the use case.
````
````old server/internal/modules/issue/adapter/http/update_test.go
		`{"start_date":"2026-10-18"}`, `{"name":null}`, `{"state_id":null}`, `{"assignee_ids":null}`, `{"name":1}`, `{"state_id":"Backlog"}`,
		`{"assignee_ids":"bob"}`, `{"label_ids":["Bug"]}`} {
````
````new server/internal/modules/issue/adapter/http/update_test.go
		`{"name":null}`, `{"state_id":null}`, `{"assignee_ids":null}`, `{"name":1}`, `{"state_id":"Backlog"}`, `{"assignee_ids":"bob"}`,
		`{"label_ids":["Bug"]}`, `{"start_date":"2026-02-30"}`, `{"target_date":"2026-10-19T00:00:00Z"}`} {
````

`server/internal/modules/issue/app/update_issue_test.go`（修改，4 处）：

````old server/internal/modules/issue/app/update_issue_test.go
// A patch that changes nothing, given empty or with each field as it is,
// its sets in another order: the precheck, the lock and the decisions run
// as for any update; nothing is written, not even the clock read, and the
// answer is the work item as read under the lock, its last writer and
// moment as they were.
````
````new server/internal/modules/issue/app/update_issue_test.go
// A patch that changes nothing, given empty or with each field as it is:
// its sets in another order, its start and target dates given as the work
// item has them, none. The precheck, the lock and the decisions run as for
// any update; nothing is written, not even the clock read, and the answer
// is the work item as read under the lock, its last writer and moment as
// they were.
````
````old server/internal/modules/issue/app/update_issue_test.go
			LabelIDs: set(bug)},
````
````new server/internal/modules/issue/app/update_issue_test.go
			LabelIDs: set(bug), SetStartDate: true, SetTargetDate: true},
````
````old server/internal/modules/issue/app/update_issue_test.go
// before the state it moves into is read (409 issue.archived). Then the
// dependent rows the update adds: a state not the project's, or its triage
// state; an account added who is its guest; a label added that is not its:
// every problem in one 422, each at its place in the patch (the label
// second in label_ids, though first among the labels added). out, kept, is
// not checked, though no member of web.
````
````new server/internal/modules/issue/app/update_issue_test.go
// before the state it moves into is read, and before the order of the
// dates it gives is checked (409 issue.archived). Then the dependent rows
// the update adds: a state not the project's, or its triage state; an
// account added who is its guest; a label added that is not its: every
// problem in one 422, each at its place in the patch (the label second in
// label_ids, though first among the labels added). out, kept, is not
// checked, though no member of web.
````
````old server/internal/modules/issue/app/update_issue_test.go
			domain.ErrIssueArchived, locked},
````
````new server/internal/modules/issue/app/update_issue_test.go
			domain.ErrIssueArchived, locked},
		{"an archived work item given dates out of order", func(w *world) { w.issues.archivedAt = &earlier },
			domain.IssuePatch{SetStartDate: true, StartDate: october(20), SetTargetDate: true, TargetDate: october(19)}, domain.ErrIssueArchived,
			locked},
````

`server/internal/platform/httpserver/bodyshape/bodyshape.go`（修改，1 处）：

````old server/internal/platform/httpserver/bodyshape/bodyshape.go
// YYYY-MM-DD, of a year from 0001.
````
````new server/internal/platform/httpserver/bodyshape/bodyshape.go
// YYYY-MM-DD, of a year from 0001. The issue module's HTTP adapter tests
// that the two agree (TestDatesPassTheBodyCheckAsTheyDecode).
````

- [ ] **Step 3: 组合和矩阵**

`server/internal/bootstrap/issue_dates_test.go`（新文件，114 行）：

````file server/internal/bootstrap/issue_dates_test.go
package bootstrap

import (
	"net/http"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The dates as bootstrap wires them (M4 design 3.7, 3.13): alice creates a
// work item in Web from the 12th to the 19th of October. bob gives it a
// start date alone after the target date it keeps, the 1st of November (a
// later month, an earlier day), then a target date alone before the start
// date it keeps: each refused (422), at start_date and at target_date,
// nothing written. He moves both past the target date it had, then clears
// its target date: each written, each date changed a row of his activity
// (rows 8, 9). Then he gives it the dates it has, which changes nothing
// and writes no row. Each answer shows the dates as the database has them.
// Once alice has moved it into Done and archived it, bob's target date
// before its start date is refused (409 issue.archived) before the order
// of its dates is checked, and nothing is written. A creation with its
// start date after its target date, the 1st of November and the 19th of
// October, is refused (422 at target_date) and takes no number; one with
// both on one day is made, with the next number.
func TestDatesAreKeptInOrder(t *testing.T) {
	w := newIssueWorld(t)
	item := w.newWorkItemIn(t, w.web, `{"name":"Fix","start_date":"2026-10-12","target_date":"2026-10-19"}`)
	path := "/api/v0/issues/" + item.String()
	for _, step := range []struct {
		body, refusal, stored string // refusal "" for a change answered 200
		writes                bool   // whether it writes the work item
	}{
		{`{"start_date":"2026-11-01"}`, "validation_failed start_date not_allowed: must not be after the target date", "2026-10-12 2026-10-19",
			false},
		{`{"target_date":"2026-10-11"}`, "validation_failed target_date not_allowed: must not be before the start date", "2026-10-12 2026-10-19",
			false},
		{`{"start_date":"2026-10-20","target_date":"2026-10-25"}`, "", "2026-10-20 2026-10-25", true},
		{`{"target_date":null}`, "", "2026-10-20 none", true},
		{`{"start_date":"2026-10-20","target_date":null}`, "", "2026-10-20 none", false},
	} {
		_, changed, by := archivalOf(t, w, item)
		if step.refusal != "" {
			answer := w.send(t, "bob", http.MethodPatch, path, step.body, http.StatusUnprocessableEntity)
			if got := refusalOf(t, []byte(answer), true); got != step.refusal {
				t.Errorf("PATCH %s answers %s, want %s", step.body, got, step.refusal)
			}
		} else if got := answeredDates(t, w.send(t, "bob", http.MethodPatch, path, step.body, http.StatusOK)); got != step.stored {
			t.Errorf("PATCH %s answers the dates %s, want %s", step.body, got, step.stored)
		}
		if got := w.datesOf(t, item); got != step.stored {
			t.Errorf("after PATCH %s the dates are %s, want %s", step.body, got, step.stored)
		}
		if _, at, who := archivalOf(t, w, item); (at.Equal(changed) && who == by) == step.writes {
			t.Errorf("after PATCH %s the work item was changed last by %s at %v, before it by %s at %v; want it written: %v", step.body, who,
				at, by, changed, step.writes)
		}
	}
	w.send(t, "alice", http.MethodPatch, path, `{"state_id":"`+w.done.String()+`"}`, http.StatusOK)
	w.send(t, "alice", http.MethodPost, path+"/archive", "", http.StatusOK)
	_, archived, by := archivalOf(t, w, item)
	refused := w.send(t, "bob", http.MethodPatch, path, `{"target_date":"2026-10-11"}`, http.StatusConflict)
	if got := refusalOf(t, []byte(refused), true); got != "issue.archived" {
		t.Errorf("PATCH of the archived work item answers %s, want issue.archived", got)
	}
	if got := w.datesOf(t, item); got != "2026-10-20 none" {
		t.Errorf("after PATCH of the archived work item the dates are %s, want 2026-10-20 none", got)
	}
	if _, at, who := archivalOf(t, w, item); !at.Equal(archived) || who != by {
		t.Errorf("after PATCH of the archived work item it was changed last by %s at %v; want it as archived, by %s at %v", who, at, by,
			archived)
	}
	if said, want := w.activityBy(t, item, w.ids["bob"]), []string{"start_date: 2026-10-12 → 2026-10-20",
		"target_date: 2026-10-19 → 2026-10-25", "target_date: 2026-10-25 → "}; !slices.Equal(said, want) {
		t.Errorf("bob's activity says %q, want %q", said, want)
	}

	create := "/api/v0/projects/" + w.web.String() + "/issues"
	answer := w.send(t, "alice", http.MethodPost, create, `{"name":"Fix","start_date":"2026-11-01","target_date":"2026-10-19"}`,
		http.StatusUnprocessableEntity)
	if got, want := refusalOf(t, []byte(answer), true), "validation_failed target_date not_allowed: must not be before the start date"; got != want {
		t.Errorf("POST answers %s, want %s", got, want)
	}
	sameDay := w.newWorkItemIn(t, w.web, `{"name":"Fix","start_date":"2026-10-19","target_date":"2026-10-19"}`)
	var number int
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT sequence_id FROM issues WHERE id = $1", sameDay).Scan(&number); err != nil {
		t.Fatal(err)
	}
	if got := w.datesOf(t, sameDay); got != "2026-10-19 2026-10-19" || number != 2 {
		t.Errorf("the work item made on one day: from %s, number %d; want 2026-10-19 2026-10-19, number 2", got, number)
	}
}

// datesOf are the dates of the work item id as the database has them:
// "<start> <target>", each YYYY-MM-DD, or none.
func (w issueWorld) datesOf(t *testing.T, id uuid.UUID) string {
	t.Helper()
	var dates string
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT coalesce(to_char(start_date, 'YYYY-MM-DD'), 'none') || ' ' ||
		coalesce(to_char(target_date, 'YYYY-MM-DD'), 'none') FROM issues WHERE id = $1`, id).Scan(&dates); err != nil {
		t.Fatal(err)
	}
	return dates
}

// answeredDates are the dates of the work item an answer shows, as datesOf
// writes them.
func answeredDates(t *testing.T, answer string) string {
	t.Helper()
	var i storedIssue
	decodeAnswer(t, answer, &i)
	return i.dates()
}
````

`server/internal/bootstrap/issue_updates_test.go`（修改，7 处）：

````old server/internal/bootstrap/issue_updates_test.go
// made it and changed by him again after that, so that each field that
// may be null has a value of its own, each of its stamps is its own, and
// it was made, completed, changed last and archived at four moments, in
// that order.
````
````new server/internal/bootstrap/issue_updates_test.go
// made it and changed by him again after that, from the 12th to the 19th
// of October, so that each field that may be null has a value of its own,
// each of its stamps is its own, and it was made, completed, changed last
// and archived at four moments, in that order.
````
````old server/internal/bootstrap/issue_updates_test.go
	w.send(t, "bob", http.MethodPatch, path, `{"priority":"low","label_ids":[]}`, http.StatusOK)
````
````new server/internal/bootstrap/issue_updates_test.go
	w.send(t, "bob", http.MethodPatch, path, `{"priority":"low","label_ids":[],"start_date":"2026-10-12","target_date":"2026-10-19"}`,
		http.StatusOK)
````
````old server/internal/bootstrap/issue_updates_test.go
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT id, workspace_id, project_id, sequence_id, name, state_id, priority, parent_id, sort_order,
		completed_at, archived_at, created_by_id, updated_by_id, created_at, updated_at,
````
````new server/internal/bootstrap/issue_updates_test.go
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT id, workspace_id, project_id, sequence_id, name, state_id, priority,
		to_char(start_date, 'YYYY-MM-DD'), to_char(target_date, 'YYYY-MM-DD'), parent_id, sort_order, completed_at, archived_at, created_by_id,
		updated_by_id, created_at, updated_at,
````
````old server/internal/bootstrap/issue_updates_test.go
		&stored.StateID, &stored.Priority, &stored.ParentID, &stored.SortOrder, &stored.CompletedAt, &stored.ArchivedAt, &stored.CreatedByID,
		&stored.UpdatedByID, &stored.CreatedAt, &stored.UpdatedAt, &stored.AssigneeIDs, &stored.LabelIDs); err != nil {
````
````new server/internal/bootstrap/issue_updates_test.go
		&stored.StateID, &stored.Priority, &stored.StartDate, &stored.TargetDate, &stored.ParentID, &stored.SortOrder, &stored.CompletedAt,
		&stored.ArchivedAt, &stored.CreatedByID, &stored.UpdatedByID, &stored.CreatedAt, &stored.UpdatedAt, &stored.AssigneeIDs,
		&stored.LabelIDs); err != nil {
````
````old server/internal/bootstrap/issue_updates_test.go
	if !reflect.DeepEqual(shown, stored) || stored.ParentID == nil || stored.CompletedAt == nil || stored.ArchivedAt == nil ||
		*stored.UpdatedByID == *stored.CreatedByID || len(stored.LabelIDs) != 0 || !stored.CreatedAt.Before(*stored.CompletedAt) ||
		!stored.CompletedAt.Before(stored.UpdatedAt) || !stored.UpdatedAt.Before(*stored.ArchivedAt) {
		t.Errorf("getIssue answers %+v; want the work item as stored, %+v, under the first, made, completed, changed last by bob and "+
			"archived at four moments in that order, without labels", shown, stored)
````
````new server/internal/bootstrap/issue_updates_test.go
	if !reflect.DeepEqual(shown, stored) || stored.StartDate == nil || stored.TargetDate == nil || *stored.StartDate == *stored.TargetDate ||
		stored.ParentID == nil || stored.CompletedAt == nil || stored.ArchivedAt == nil || *stored.UpdatedByID == *stored.CreatedByID ||
		len(stored.LabelIDs) != 0 || !stored.CreatedAt.Before(*stored.CompletedAt) || !stored.CompletedAt.Before(stored.UpdatedAt) ||
		!stored.UpdatedAt.Before(*stored.ArchivedAt) {
		t.Errorf("getIssue answers %+v, dated %s; want the work item as stored, %+v, dated %s, from one day to another, under the first, "+
			"made, completed, changed last by bob and archived at four moments in that order, without labels", shown, shown.dates(), stored,
			stored.dates())
````
````old server/internal/bootstrap/issue_updates_test.go
	Priority    string      `json:"priority"`
````
````new server/internal/bootstrap/issue_updates_test.go
	Priority    string      `json:"priority"`
	StartDate   *string     `json:"start_date"`
	TargetDate  *string     `json:"target_date"`
````
````old server/internal/bootstrap/issue_updates_test.go
	LabelIDs    []uuid.UUID `json:"label_ids"`
````
````new server/internal/bootstrap/issue_updates_test.go
	LabelIDs    []uuid.UUID `json:"label_ids"`
}

// dates are the start and the target dates of s as datesOf writes them:
// "<start> <target>", each YYYY-MM-DD, or none.
func (s storedIssue) dates() string {
	dates := []string{"none", "none"}
	for n, d := range []*string{s.StartDate, s.TargetDate} {
		if d != nil {
			dates[n] = *d
		}
	}
	return dates[0] + " " + dates[1]
````

`server/internal/bootstrap/permission_matrix_issue_updates_test.go`（修改，7 处）：

````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
		// member's, UI for Bug.
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
		// member's, UI for Bug, the 12th to the 19th of October for no dates.
````
````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
					s.account(callerProjectAdmin).String() + `"],"label_ids":["` + s.label(projectOf(c), "UI").String() + `"]}`
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
					s.account(callerProjectAdmin).String() + `"],"label_ids":["` + s.label(projectOf(c), "UI").String() +
					`"],"start_date":"2026-10-12","target_date":"2026-10-19"}`
````
````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
			cells: refused, refusal: "assignee_ids[2] not_allowed"},
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
			cells: refused, refusal: "assignee_ids[2] not_allowed"},
		// Dates out of order, checked after the decision with the dependent
		// rows (M4 design 3.7): who may not change the work item learns
		// nothing of its dates.
		{op: "updateIssue", variant: "dates out of order", write: true, columns: projectColumns,
			request: toWorkItem(http.MethodPatch, "", `{"start_date":"2026-10-20","target_date":"2026-10-19"}`), cells: refused,
			refusal: "target_date not_allowed"},
````
````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
	Priority    string      `json:"priority"`
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
	Priority    string      `json:"priority"`
	StartDate   *string     `json:"start_date"`
	TargetDate  *string     `json:"target_date"`
````
````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
// as an answer shows it: of its number (numberOf), in Todo, medium, not
// completed, assigned to PM's account and the removed member's, in id
// order, labelled Bug, made and changed last by its maker.
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
// as an answer shows it: of its number (numberOf), in Todo, medium, without
// dates, not completed, assigned to PM's account and the removed member's,
// in id order, labelled Bug, made and changed last by its maker.
````
````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
// completed, high, assigned to PA's account alone, labelled UI alone.
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
// completed, high, assigned to PA's account alone, labelled UI alone, from
// the 12th to the 19th of October.
````
````old server/internal/bootstrap/permission_matrix_issue_updates_test.go
		i.Name, i.StateID, i.Priority = "Mend", s.state(projectOf(c), "Done"), "high"
````
````new server/internal/bootstrap/permission_matrix_issue_updates_test.go
		start, target := "2026-10-12", "2026-10-19"
		i.Name, i.StateID, i.Priority, i.StartDate, i.TargetDate = "Mend", s.state(projectOf(c), "Done"), "high", &start, &target
````

`server/internal/bootstrap/permission_matrix_issues_test.go`（修改，5 处）：

````old server/internal/bootstrap/permission_matrix_issues_test.go
		// member of the project, and labelled UI; the activity names him by
		// his display name, which identity gives (bootstrap/ports.go).
````
````new server/internal/bootstrap/permission_matrix_issues_test.go
		// member of the project, labelled UI, from the 12th to the 19th of
		// October; the activity names him by his display name, which
		// identity gives (bootstrap/ports.go).
````
````old server/internal/bootstrap/permission_matrix_issues_test.go
					s.account(callerProjectMember).String() + `"],"label_ids":["` + s.label(projectOf(c), "UI").String() + `"]}`
````
````new server/internal/bootstrap/permission_matrix_issues_test.go
					s.account(callerProjectMember).String() + `"],"label_ids":["` + s.label(projectOf(c), "UI").String() +
					`"],"start_date":"2026-10-12","target_date":"2026-10-19"}`
````
````old server/internal/bootstrap/permission_matrix_issues_test.go
// assigned to PM's account, labelled UI.
````
````new server/internal/bootstrap/permission_matrix_issues_test.go
// assigned to PM's account, labelled UI, from the 12th to the 19th of
// October.
````
````old server/internal/bootstrap/permission_matrix_issues_test.go
		AssigneeIDs []uuid.UUID `json:"assignee_ids"`
		LabelIDs    []uuid.UUID `json:"label_ids"`
	}
````
````new server/internal/bootstrap/permission_matrix_issues_test.go
		AssigneeIDs []uuid.UUID `json:"assignee_ids"`
		LabelIDs    []uuid.UUID `json:"label_ids"`
		StartDate   string      `json:"start_date"`
		TargetDate  string      `json:"target_date"`
	}
````
````old server/internal/bootstrap/permission_matrix_issues_test.go
		i.CompletedAt == nil || !slices.Equal(i.AssigneeIDs, []uuid.UUID{pm}) || !slices.Equal(i.LabelIDs, []uuid.UUID{ui}) {
		t.Errorf("%s creates %s; want %s's work item %d in Done, high, at 65535, completed, assigned to %s, labelled %s", c, answer,
			projectOf(c), next, pm, ui)
````
````new server/internal/bootstrap/permission_matrix_issues_test.go
		i.CompletedAt == nil || !slices.Equal(i.AssigneeIDs, []uuid.UUID{pm}) || !slices.Equal(i.LabelIDs, []uuid.UUID{ui}) ||
		i.StartDate != "2026-10-12" || i.TargetDate != "2026-10-19" {
		t.Errorf("%s creates %s; want %s's work item %d in Done, high, at 65535, completed, assigned to %s, labelled %s, from 2026-10-12 to "+
			"2026-10-19", c, answer, projectOf(c), next, pm, ui)
````

- [ ] **Step 4: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/... ./internal/platform/httpserver/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestDatesAreKeptInOrder$|TestAWorkItemReadsAsStored$|TestPermissionMatrix|TestBodiesThatBreakTheStructureAnswer400$' ./internal/bootstrap/`
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
Expected: 122 个全部通过。

- [ ] **Step 5: 提交**

```bash
git add api/modules/issue.yaml server/internal/bootstrap/issue_dates_test.go server/internal/bootstrap/issue_updates_test.go server/internal/bootstrap/permission_matrix_issue_updates_test.go server/internal/bootstrap/permission_matrix_issues_test.go server/internal/modules/issue/adapter/http/dates_test.go server/internal/modules/issue/adapter/http/issues.go server/internal/modules/issue/adapter/http/issues_test.go server/internal/modules/issue/adapter/http/update_test.go server/internal/modules/issue/app/update_issue_test.go server/internal/platform/httpserver/bodyshape/bodyshape.go api/dist/openapi.yaml server/internal/modules/issue/adapter/http/gen/bodyshape.gen.go server/internal/modules/issue/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M4/P2): the start and target dates in the API

IssueDetail answers start_date and target_date, null for none;
IssueCreate may give them, and IssueUpdate may give or clear them, as
YYYY-MM-DD (format: date, decoded as shared.Date and checked by
bodyshape's FormatDate). The descriptions say which order is checked
when: a creation's dates as given, an update's as the work item will
have them. A test holds bodyshape's date check to what the generated
types decode, and the composition test and the matrix keep the dates in
order through the API.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p2.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试，或没有替身的平台、领域、转换的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是生成器（`make gen-go`）和中英文翻译键（`make lint-web`）的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过，端到端的检查只在标出的 Task 的树上）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T11.1` | `PATCH` 把给成 `null` 的日期当作没给 | `internal/modules/issue/adapter/http` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDatesAreKeptInOrder`（`make test`）、故事 `i2-update-issue`（Task 13 起） | 单元；组合；端到端 |
| `T11.2` | `POST` 丢掉请求体的截止日期 | `internal/modules/issue/adapter/http` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDatesAreKeptInOrder`（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `createIssue` 各格（`make test`）、故事 `i1-create-issue`（Task 13 起） | 单元；组合；端到端 |
| `T11.3` | 回答把截止日期也给成开始日期 | `internal/modules/issue/adapter/http` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemReadsAsStored`（`make test`）、故事 `i1-create-issue`（Task 13 起） | 单元；组合；端到端 |
| `T11.4` | `PATCH` 丢掉请求体的开始日期 | `internal/modules/issue/adapter/http` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDatesAreKeptInOrder`（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `updateIssue` 各格（`make test`）、故事 `i2-update-issue`（Task 13 起） | 单元；组合；端到端 |
| `T11.5` | 什么也不改但给了日期的修改照样重写工作项（它的最后修改者和时刻），不写动态 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDatesAreKeptInOrder`（`make test`） | 单元；组合 |
| `T11.6` | 先查日期的次序、再拒绝已归档的工作项：给已归档的工作项次序不对的日期答 422，而不是 409 `issue.archived` | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDatesAreKeptInOrder`（`make test`） | 单元；组合 |

本 Task 的检查也发现较早的 Task 的变异 `T9.2`、`T9.3`、`T9.4`、`T9.9`、`T9.10`、`T9.13`、`T10.1`、`T10.2`、`T10.3`、`T10.4`、`T10.5`、`T10.6`、`T10.7`、`T10.8`、`T10.9`、`T10.12`、`T10.13`、`T10.14`（列在各自的 Task 的表里，标"（Task 11 起）"）：本 Task 做完之后，在本 Task 的树上重跑它们（`mutants_p2.py`，`MUT_LATER=11`）。

---

### Task 12: 交错：12、删除与修改、等锁期间被降级或移出的创建者

**Files:**
- Create: `server/internal/bootstrap/issue_interleaving_archive_test.go`
- Modify: `server/internal/bootstrap/issue_interleaving_deletion_test.go`、`server/internal/bootstrap/issue_interleaving_test.go`

**Interfaces:**（spec 2.13、第 3 节第 7、16、18 条；M4 设计 3.5、3.6、3.8、9.3）只改测试，用 P1b 的交错的帮手（`inBothOrders`、`itemWrite`、`racingWrite`、`changesIssue`、`answered`、`refused`、`deletedWith`），不另写：
- `issue_interleaving_archive_test.go`（新）：`archivesIssue(name)`、`inProgress`；交错 12。
- `issue_interleaving_deletion_test.go`：`deletesIssue(name, _)`（不发请求体，作为 `itemWrite` 的写不取体）；删除与修改同一个工作项。
- `issue_interleaving_test.go`：`wroteNothing` 也数未删除的行（被拒绝的写既没写也没删）；交错 4、5 加"删除他自己的"一支（`AllowCreator` 在锁下再判定）。

**Tests:**（组合；两种顺序；`-race -count=5` 没有 40P01）
- `TestArchivingAWorkItemAndMovingItSerialize`（交错 12）：alice 归档 Done 里的 Fix，bob 把它移到 In Progress。先移：归档过了预检，锁下读到开始组，409 `issue.not_archivable`，没有归档；先归档：修改过了预检，锁下读到已归档，409 `issue.archived`，它留在 Done。动态里只有过了锁的那个写的一行。
- `TestDeletingAWorkItemAndChangingItSerialize`：alice 删除她的 Fix，bob 改名。先删：修改锁下找不到 Fix，404 `issue.not_found`，没写，三行已删除；先改：删除等 Web 的行，然后把工作项和它的四行（含 bob 的一行动态）由 alice 在同一时刻删除，时刻不早于修改的提交。
- `TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused` 加"删除他自己的"：预检放行之后被移出 404 `issue.not_found`；被降为"只看自己"的项目里的访客，删他自己的 403；都没有写、没有删。

- [ ] **Step 1: 交错 12**

`server/internal/bootstrap/issue_interleaving_archive_test.go`（新文件，60 行）：

````file server/internal/bootstrap/issue_interleaving_archive_test.go
package bootstrap

import (
	"net/http"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// archivesIssue is name's archiving of Web's one work item (workItem).
func archivesIssue(name string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newIssueRoute(t, w.pool, tx), name, http.MethodPost, "/api/v0/issues/"+w.workItem(t).String()+"/archive", "")
	}
}

// inProgress is the body of a change into Web's In Progress, a state of
// the started group.
func inProgress(t *testing.T, w issueWorld) string {
	return `{"state_id":"` + stateID(t, w.pool, w.web, "In Progress").String() + `"}`
}

// Archiving a work item and moving it out of the completed group serialize
// on Web's row (M4 design 3.8; interleaving 12), in both orders: alice
// archives her Fix in Done (doneWorkItem), bob moves it into In Progress.
// Moved first, holding Web FOR NO KEY UPDATE: the archiving, which passed
// its precheck, finds it in the started group under its lock: 409
// issue.not_archivable, and it is not archived. Archived first: the
// change, which passed its precheck, finds it archived under its lock: 409
// issue.archived, and it stays in Done. Only the write that went through
// is in the activity: bob's move, or the archiving.
func TestArchivingAWorkItemAndMovingItSerialize(t *testing.T) {
	inBothOrders(t, func(t *testing.T, w issueWorld) { w.doneWorkItem(t) }, "archived", archivesIssue("alice"), "moved",
		changesIssue("bob", inProgress), func(t *testing.T, w issueWorld, archivedFirst bool, archived, moved exchange) {
			state, said, archivals := "In Progress", []string{"state: Done → In Progress"}, 0
			if archivedFirst {
				w.answered(t, archived, http.StatusOK)
				w.refused(t, moved, http.StatusConflict, "issue.archived")
				state, said, archivals = "Done", nil, 1
			} else {
				w.answered(t, moved, http.StatusOK)
				w.refused(t, archived, http.StatusConflict, "issue.not_archivable")
			}
			item := w.workItem(t)
			var in uuid.UUID
			var isArchived bool
			if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT state_id, archived_at IS NOT NULL FROM issues WHERE id = $1", item).Scan(&in,
				&isArchived); err != nil {
				t.Fatal(err)
			}
			if got := w.activityBy(t, item, w.ids["bob"]); in != stateID(t, w.pool, w.web, state) || isArchived != archivedFirst ||
				!slices.Equal(got, said) || len(archivalRows(t, w, item)) != archivals {
				t.Errorf("the work item in %s, archived %v, bob's activity %q, %d archival rows; want in %s (%s), archived %v, %q, %d", in,
					isArchived, got, len(archivalRows(t, w, item)), stateID(t, w.pool, w.web, state), state, archivedFirst, said, archivals)
			}
		})
}
````

- [ ] **Step 2: 删除与修改；创建者的删除**

`server/internal/bootstrap/issue_interleaving_deletion_test.go`（修改，4 处）：

````old server/internal/bootstrap/issue_interleaving_deletion_test.go
	"net/http"
````
````new server/internal/bootstrap/issue_interleaving_deletion_test.go
	"net/http"
	"slices"
````
````old server/internal/bootstrap/issue_interleaving_deletion_test.go
}

// fullBody is the world's fullIssue, as createsIssue takes a body.
````
````new server/internal/bootstrap/issue_interleaving_deletion_test.go
}

// deletesIssue is name's deletion of Web's one work item (workItem), which
// sends no body: as an itemWrite's write, it ignores the body it is given.
func deletesIssue(name string, _ func(*testing.T, issueWorld) string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newIssueRoute(t, w.pool, tx), name, http.MethodDelete, "/api/v0/issues/"+w.workItem(t).String(), "")
	}
}

// fullBody is the world's fullIssue, as createsIssue takes a body.
````
````old server/internal/bootstrap/issue_interleaving_deletion_test.go
// deletedWith checks the deletion of the row id of table, the deleted
// project's or workspace's: at a moment no earlier than since, with the
// rows of Web's work-item side, n of them, each deleted and last written
// at that moment, by alice, and not before it was made.
````
````new server/internal/bootstrap/issue_interleaving_deletion_test.go
// deletedWith checks the deletion of the row id of table, the deleted work
// item's, project's or workspace's: at a moment no earlier than since,
// with the rows of Web's work-item side, n of them, each deleted and last
// written at that moment, by alice, and not before it was made.
````
````old server/internal/bootstrap/issue_interleaving_deletion_test.go
		}
	}
}

````
````new server/internal/bootstrap/issue_interleaving_deletion_test.go
		}
	}
}

// Deleting a work item and changing it serialize on Web's row (M4 design
// 3.5, 3.6), in both orders. alice deletes her Fix (aWorkItem), whose
// three rows are hers; bob renames it Mend. Deleted first, holding Web FOR
// NO KEY UPDATE: the change, which passed its precheck on Fix as it was,
// finds no Fix under its lock: 404 issue.not_found, and it wrote nothing;
// the three rows are deleted. Changed first: the deletion waits for Web's
// row, then deletes the work item and its rows, bob's activity among them,
// four, by alice, at its one moment, no earlier than the change's commit:
// it read its clock under the lock it took once the change had committed.
func TestDeletingAWorkItemAndChangingItSerialize(t *testing.T) {
	inBothOrders(t, aWorkItem, "deleted", deletesIssue("alice", nil), "changed", changesIssue("bob", named("Mend")), func(t *testing.T,
		w issueWorld, deletedFirst bool, deleted, changed exchange) {
		w.answered(t, deleted, http.StatusNoContent)
		rows, said := 4, []string{"name: Fix → Mend"}
		if deletedFirst {
			w.refused(t, changed, http.StatusNotFound, "issue.not_found")
			rows, said = 3, nil
		} else {
			w.answered(t, changed, http.StatusOK)
		}
		if got := w.activityBy(t, w.workItem(t), w.ids["bob"]); !slices.Equal(got, said) {
			t.Errorf("bob's activity says %q, want %q", got, said)
		}
		w.deletedWith(t, "issues", w.workItem(t), changed.released, rows)
	})
}

````

`server/internal/bootstrap/issue_interleaving_test.go`（修改，6 处）：

````old server/internal/bootstrap/issue_interleaving_test.go
// creation and as a change alike (M4 design 3.5: P1a's creation branch,
// P1b's change branch): prepare writes what it needs first, nil for
// nothing, leaving rows rows on Web's work-item side (workItemRows);
// write is name's write with the body body gives; done is its answer, and
// gone the code that refuses it once its project is gone to its caller.
````
````new server/internal/bootstrap/issue_interleaving_test.go
// creation, a change and a deletion alike (M4 design 3.5: P1a's creation
// branch, P1b's change branch, P2's deletion by its creator): prepare
// writes what it needs first, nil for nothing, leaving rows rows on Web's
// work-item side (workItemRows); write is name's write, with the body body
// gives when it sends one; done is its answer, and gone the code that
// refuses it once its project is gone to its caller.
````
````old server/internal/bootstrap/issue_interleaving_test.go
// wroteNothing checks that Web's work-item side still has iw's rows: a
// write refused under its lock wrote none.
````
````new server/internal/bootstrap/issue_interleaving_test.go
// wroteNothing checks that Web's work-item side still has iw's rows, each
// undeleted: a write refused under its lock wrote none and deleted none.
````
````old server/internal/bootstrap/issue_interleaving_test.go
	if n := w.workItemRows(t, []uuid.UUID{w.web}, false); n != iw.rows {
		t.Errorf("Web's work-item side has %d rows after the refused write; want %d, as before it", n, iw.rows)
````
````new server/internal/bootstrap/issue_interleaving_test.go
	web := []uuid.UUID{w.web}
	if n, undeleted := w.workItemRows(t, web, false), w.workItemRows(t, web, true); n != iw.rows || undeleted != iw.rows {
		t.Errorf("Web's work-item side has %d rows, %d undeleted, after the refused write; want %d, each undeleted, as before it", n,
			undeleted, iw.rows)
````
````old server/internal/bootstrap/issue_interleaving_test.go
// interleavings 4 and 5, the creation's branch and the change's): bob, a
// member of Web, made secret, creates a work item, or changes alice's Fix,
// walt's or his own, while alice's removal of him, or her demotion of him
// to a guest, holds Web past its writes. His write passed the precheck,
// which read him a member, and waits for Web's row; once hers commits, the
// decision under the lock reads him removed, who no longer sees Web: 404
// project.not_found, or issue.not_found for a change; or a guest of a
// project that does not show its guests every work item, who finds no work
// item another created, alice's Fix or walt's, their creators' ids before
// his and after it, as one not there: 404 issue.not_found (3.4); and who
// writes no work item, his own or a new one: 403 forbidden. His write
// wrote nothing.
````
````new server/internal/bootstrap/issue_interleaving_test.go
// interleavings 4 and 5, the creation's branch and the change's, and the
// deletion of a work item by its creator, whom AllowCreator lets: 3.4):
// bob, a member of Web, made secret, creates a work item, changes alice's
// Fix, walt's or his own, or deletes his own, while alice's removal of
// him, or her demotion of him to a guest, holds Web past its writes. His
// write passed the precheck, which read him a member, and waits for Web's
// row; once hers commits, the decision under the lock reads him removed,
// who no longer sees Web: 404 project.not_found, or issue.not_found for a
// change or a deletion; or a guest of a project that does not show its
// guests every work item, who finds no work item another created, alice's
// Fix or walt's, their creators' ids before his and after it, as one not
// there: 404 issue.not_found (3.4); and who writes no work item, his own
// or a new one, nor deletes his own: 403 forbidden. His write wrote
// nothing and deleted nothing.
````
````old server/internal/bootstrap/issue_interleaving_test.go
		"changed walt's": cellIssueNotFound}
````
````new server/internal/bootstrap/issue_interleaving_test.go
		"changed walt's": cellIssueNotFound, "deleted his own": cellForbidden}
````
````old server/internal/bootstrap/issue_interleaving_test.go
		itemWrite{"changed walt's", waltsWorkItem, 3, changesIssue, http.StatusOK, "issue.not_found"}) {
````
````new server/internal/bootstrap/issue_interleaving_test.go
		itemWrite{"changed walt's", waltsWorkItem, 3, changesIssue, http.StatusOK, "issue.not_found"},
		itemWrite{"deleted his own", bobsWorkItem, 3, deletesIssue, http.StatusNoContent, "issue.not_found"}) {
````

- [ ] **Step 3: 测试和 lint**

Run: `go -C server test -count=1 -run 'TestArchivingAWorkItemAndMovingItSerialize$|TestDeletingAWorkItemAndChangingItSerialize$|TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused$' ./internal/bootstrap/`
Expected: `ok`。

Run: `go -C server test -count=5 -race -timeout 30m -run 'TestArchivingAWorkItemAndMovingItSerialize$|TestDeletingAWorkItemAndChangingItSerialize$|TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused$|TestTwoCreationsOnAProjectSerialize$|TestTwoChangesOfAWorkItemSerialize$|TestAWorkItemGoneWhileItsChangeWaitsIsNotFound$|TestArchivingAProjectAndCreatingAWorkItemSerialize$|TestRemovingTheDefaultAssigneeAndCreatingAWorkItemSerialize$|TestAWorkItemsWriteAndDeletingItsProjectOrWorkspaceSerialize$|TestDeletingAStateAndPuttingAWorkItemInItSerialize$|TestDeletingALabelAndLabellingAWorkItemWithItSerialize$|TestTwoWritesOnAProjectSerialize$|TestChainedWorkItemWritesHoldOffNoWorkspaceWrite$|TestAHotProjectHoldsUpNoReadOfAnother$|TestEachIssueWriteLocksItsProjectFirst$' ./internal/bootstrap/`
Expected: `ok`；输出中没有 `40P01`、`deadlock detected`、`DATA RACE`（工作项一侧的每个交错、闸门和连接池的测试、取锁形状，各五遍）。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/bootstrap/issue_interleaving_archive_test.go server/internal/bootstrap/issue_interleaving_deletion_test.go server/internal/bootstrap/issue_interleaving_test.go
```
```bash
git commit -m "test(M4/P2): interleaving 12, a deletion against a change, and the creator's deletion

Archiving a work item and moving it out of Done serialize on its
project's row in both orders: whichever comes second finds under the
lock what the first did (issue.not_archivable, or issue.archived), and
only the first is in the activity. Deleting a work item and changing it
serialize too: the change finds it gone (issue.not_found), or the
deletion stamps the change's rows with the rest. A creator removed or
demoted to a guest while his deletion waits is refused under the lock.
wroteNothing counts undeleted rows too.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p2.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试，或没有替身的平台、领域、转换的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是生成器（`make gen-go`）和中英文翻译键（`make lint-web`）的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过，端到端的检查只在标出的 Task 的树上）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T12.1` | 删除不在锁下再判定：等锁期间被降级或移出的创建者照样删除 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused`（`make test`） | 单元；组合 |
| `T12.2` | 在锁下被拒绝的删除照样删除并提交，回答仍是那个拒绝 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused`（`make test`） | 单元；组合 |

本 Task 的检查也发现较早的 Task 的变异 `T1.1`、`T1.4`、`T2.4`、`T4.4`、`T7.4`、`T8.1`（列在各自的 Task 的表里，标"（Task 12 起）"）：本 Task 做完之后，在本 Task 的树上重跑它们（`mutants_p2.py`，`MUT_LATER=12`）。

---

### Task 13: 端到端：I15 的接口版本，I1、I2 的日期，I2 的已归档 409

**Files:**
- Create: `e2e/stories/issue/i15-archive-issue.spec.ts`
- Modify: `e2e/fixtures/assert/issue.ts`、`e2e/fixtures/issues.ts`、`e2e/stories/issue/i1-create-issue.spec.ts`、`e2e/stories/issue/i2-update-issue.spec.ts`

**Interfaces:**（spec 2.14、第 3 节第 17 条；M4 设计第 2 节 I1、I2、I15，3.6、3.8、3.13）只改 `e2e/`，扩展共用的夹具，不复制：
- `e2e/fixtures/issues.ts`：`issueArchived`、`issueUnarchived`（状态和工作项，或问题的码）、`issueDeleted`（状态或码）。
- `e2e/fixtures/assert/issue.ts`：`archivedRow`、`restoredRow`（第 14、16 行，经共用的 `changed` 写；`changed` 的说明照改）；`CreatedIssue`、`UpdatedIssue` 加 `start_date`、`target_date`（`YYYY-MM-DD`，没有是 `null`）；`Archival`、`expectIssueArchival`；`expectIssueDeleted`：从目录读出外键图，自 `issues` 向下递归（两层以下的行也算在它下面），每个外键下都要有一行（`subItems`：`issues.parent_id` 豁免到 P3），删除者和时刻是那一次删除的；在工作项之前已删除的行不算，也不在这里核对（它保留原来的删除由 `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt` 核对）。
- `i15-archive-issue.spec.ts`（新）：ann 在 In Progress 的工作项 409 `issue.not_archivable`；移到 Done 后归档（一行动态），再归档不写；已归档时修改 409 `issue.archived`；gus（项目的访客，项目"可见全部"）恢复、归档都 403；恢复（一行动态），再恢复不写；再归档之后，管理员删除它：`expectIssueDeleted`。
- `i1-create-issue.spec.ts`：起止日期；开始晚于截止的 422 与别的值的问题一次报出（`target_date`）。`i2-update-issue.spec.ts`：日期的修改（两行动态）；只给晚于留着的截止日期的开始日期、只给早于留着的开始日期的截止日期各 422；两个都移过原来的截止日期（只看请求会拒绝的一步不被拒绝）；清空；已归档的工作项 409 `issue.archived`。

**Tests:** 端到端：I15（新）、I1、I2；`make e2e` 123 个。访客取别人的工作项的 404 在本 Phase 由矩阵核对，端到端的那一格随 I20 在 P5。

- [ ] **Step 1: 夹具**

`e2e/fixtures/assert/issue.ts`（修改，9 处）：

````old e2e/fixtures/assert/issue.ts
 * An "updated" row of a work item's activity: field changed from old to now, with Plane's comment; named and was,
 * what its new and old identifiers name, for a state's, a label's or an account's row.
````
````new e2e/fixtures/assert/issue.ts
 * An "updated" row of a work item's activity: field changed from old to now, with its comment, Plane's but for the
 * archiving's and the restoring's rows; named and was, what its new and old identifiers name, for a state's, a
 * label's or an account's row.
````
````old e2e/fixtures/assert/issue.ts
  return changed("assignees", "", shownNameOf(email), "added assignee ", email);
}
````
````new e2e/fixtures/assert/issue.ts
  return changed("assignees", "", shownNameOf(email), "added assignee ", email);
}

/** The rows of a work item's activity that archive it and that restore it (M4 design 3.13, rows 14 and 16). */
export const archivedRow: ActivityRow = {
  field: "archived_at",
  verb: "updated",
  old_value: null,
  new_value: "manual_archive",
  named: null,
  was: null,
  comment: "archived the work item",
};
export const restoredRow = changed("archived_at", "archive", "restore", "restored the work item");
````
````old e2e/fixtures/assert/issue.ts
 * A work item as created: its number, name, state (by name), priority and sort order; whether it is completed; the
 * address of the account that created it; its assignees and subscribers by address, its labels by name, each in
 * order; and its activity, in its order.
````
````new e2e/fixtures/assert/issue.ts
 * A work item as created: its number, name, state (by name), priority, dates (YYYY-MM-DD, null for none) and sort
 * order; whether it is completed; the address of the account that created it; its assignees and subscribers by
 * address, its labels by name, each in order; and its activity, in its order.
````
````old e2e/fixtures/assert/issue.ts
  state: string;
  priority: string;
  sort_order: number;
````
````new e2e/fixtures/assert/issue.ts
  state: string;
  priority: string;
  start_date: string | null;
  target_date: string | null;
  sort_order: number;
````
````old e2e/fixtures/assert/issue.ts
    `SELECT i.sequence_id, i.name, s.name AS state, i.priority, i.sort_order, i.completed_at IS NOT NULL AS completed,
````
````new e2e/fixtures/assert/issue.ts
    `SELECT i.sequence_id, i.name, s.name AS state, i.priority, to_char(i.start_date, 'YYYY-MM-DD') AS start_date,
            to_char(i.target_date, 'YYYY-MM-DD') AS target_date, i.sort_order, i.completed_at IS NOT NULL AS completed,
````
````old e2e/fixtures/assert/issue.ts
 * A work item as changed: its name, state (by name) and priority; whether it is completed; the address of the account
 * that changed it last; its undeleted assignees and subscribers by address, its undeleted labels by name, each in
 * order; its activity, in its order; and how many rows of its assignees, labels, subscribers and activity its last
 * change wrote, those it took off among them.
````
````new e2e/fixtures/assert/issue.ts
 * A work item as changed: its name, state (by name), priority and dates (YYYY-MM-DD, null for none); whether it is
 * completed; the address of the account that changed it last; its undeleted assignees and subscribers by address, its
 * undeleted labels by name, each in order; its activity, in its order; and how many rows of its assignees, labels,
 * subscribers and activity its last change wrote, those it took off among them.
````
````old e2e/fixtures/assert/issue.ts
  state: string;
  priority: string;
  completed: boolean;
````
````new e2e/fixtures/assert/issue.ts
  state: string;
  priority: string;
  start_date: string | null;
  target_date: string | null;
  completed: boolean;
````
````old e2e/fixtures/assert/issue.ts
    `SELECT i.name, s.name AS state, i.priority, i.completed_at IS NOT NULL AS completed, u.email AS by,
````
````new e2e/fixtures/assert/issue.ts
    `SELECT i.name, s.name AS state, i.priority, to_char(i.start_date, 'YYYY-MM-DD') AS start_date,
            to_char(i.target_date, 'YYYY-MM-DD') AS target_date, i.completed_at IS NOT NULL AS completed, u.email AS by,
````
````old e2e/fixtures/assert/issue.ts
  expect(stamps, `the rows the last change of the work item ${issueId} wrote`).toEqual({ rows: wrote, stamped: wrote });
}

````
````new e2e/fixtures/assert/issue.ts
  expect(stamps, `the rows the last change of the work item ${issueId} wrote`).toEqual({ rows: wrote, stamped: wrote });
}

/** A work item's archival: whether it is archived, the address of the account that wrote it last, and its activity. */
export interface Archival {
  archived: boolean;
  by: string;
  activity: ActivityRow[];
}

/**
 * I15: the work item of issueId is archived exactly when want.archived (M4 design 3.8): undeleted, written last by the
 * account of want.by, archived at that last write when it is archived. Its activity is want.activity, the last row of
 * it want.by's, made at that last write.
 */
export async function expectIssueArchival(db: Database, issueId: string, want: Archival): Promise<void> {
  const [issue] = await db.query(
    `SELECT i.archived_at IS NOT NULL AS archived, u.email AS by,
            i.deleted_at IS NULL AND (i.archived_at IS NULL OR i.archived_at = i.updated_at) AND a.actor_id = i.updated_by_id
              AND a.created_by_id = i.updated_by_id AND a.created_at = i.updated_at AS stamped
       FROM issues i JOIN users u ON u.id = i.updated_by_id,
            LATERAL (SELECT actor_id, created_by_id, created_at FROM issue_activities WHERE issue_id = i.id
                      ORDER BY created_at DESC, id DESC LIMIT 1) a
      WHERE i.id = $1`,
    [issueId]
  );
  const { activity, ...row } = want;
  expect(issue, `the work item ${issueId}`).toEqual({ ...row, stamped: true });
  expect(await activityOf(db, issueId), `the activity of the work item ${issueId}`).toEqual(activity);
}

/** A foreign key of one column: the table that holds it, the column, and the table it names. */
interface ForeignKey {
  child: string;
  column: string;
  parent: string;
}

/** The key of a work item's sub-items, which come with P3, as their deletion with their parent's (M4 design 3.6, 3.9). */
const subItems = "issues.parent_id";

/**
 * I15 (I17 from P3): the work item of issueId is deleted by the account of email, and with it, at the same moment and
 * by the same account, every row under it that was not deleted before (M4 design 3.6). A row is under it when a
 * foreign key of its table names the work item or a row under it: the catalog's graph of foreign keys is walked down
 * from issues, so that a row two keys below (a reaction to a comment, which names the comment) is under it too. Each
 * key of the graph has such a row, but its sub-items' (subItems), which no work item has before P3; a row deleted
 * before the work item is neither held to its deletion nor checked here (that it keeps its deletion is
 * `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`'s).
 */
export async function expectIssueDeleted(db: Database, issueId: string, email: string): Promise<void> {
  const [issue] = await db.query(
    `SELECT i.deleted_at IS NOT NULL AS deleted, u.email AS by FROM issues i JOIN users u ON u.id = i.updated_by_id WHERE i.id = $1`,
    [issueId]
  );
  expect(issue, `the work item ${issueId}`).toEqual({ deleted: true, by: email });
  const all = await db.query<ForeignKey>(
    `SELECT c.conrelid::regclass::text AS child, a.attname::text AS column, c.confrelid::regclass::text AS parent
       FROM pg_constraint c JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = c.conkey[1]
      WHERE c.contype = 'f' AND cardinality(c.conkey) = 1`
  );
  // The keys below issues, in the order the walk reaches them: tables grows as the loop goes, each table once.
  const tables = ["issues"];
  const keys: ForeignKey[] = [];
  for (let i = 0; i < tables.length; i++) {
    for (const key of all.filter((k) => k.parent === tables[i])) {
      keys.push(key);
      if (!tables.includes(key.child)) {
        tables.push(key.child);
      }
    }
  }
  expect(keys.length, "the keys below issues").toBeGreaterThan(1);
  // The rows under the work item, by table, and how each key's were deleted: a round reads each key's rows under the
  // rows found before it, and the walk goes on while a round finds a row that is new.
  const under = new Map<string, Set<string>>([["issues", new Set([issueId])]]);
  const counts = new Map<string, { withIt: number; other: number }>();
  const round = async (): Promise<void> => {
    const found = await Promise.all(
      keys.map((key) =>
        db.query<{ id: string; with_it: boolean; other: boolean }>(
          `SELECT t.id, t.deleted_at = i.deleted_at AND t.updated_by_id = i.updated_by_id AS with_it,
                  (t.deleted_at IS NULL OR t.deleted_at >= i.deleted_at)
                    AND (t.deleted_at IS DISTINCT FROM i.deleted_at OR t.updated_by_id IS DISTINCT FROM i.updated_by_id) AS other
             FROM ${key.child} t, issues i WHERE t.${key.column} = ANY($1::uuid[]) AND i.id = $2`,
          [[...(under.get(key.parent) ?? [])], issueId]
        )
      )
    );
    let grew = false;
    keys.forEach((key, i) => {
      const rows = found[i] ?? [];
      counts.set(`${key.child}.${key.column}`, {
        withIt: rows.filter((r) => r.with_it).length,
        other: rows.filter((r) => r.other).length,
      });
      const ids = under.get(key.child) ?? new Set<string>();
      for (const { id } of rows) {
        grew ||= !ids.has(id);
        ids.add(id);
      }
      under.set(key.child, ids);
    });
    if (grew) {
      await round();
    }
  };
  await round();
  const names = keys.map((k) => `${k.child}.${k.column}`);
  expect(
    names.map((key) => ({ key, deletedWithIt: (counts.get(key)?.withIt ?? 0) > 0, other: counts.get(key)?.other })),
    `the rows under the work item ${issueId}`
  ).toEqual(names.map((key) => ({ key, deletedWithIt: key !== subItems, other: 0 })));
}

````

`e2e/fixtures/issues.ts`（修改，1 处）：

````old e2e/fixtures/issues.ts
  return updated.issue;
}

````
````new e2e/fixtures/issues.ts
  return updated.issue;
}

/**
 * The answer of POST /api/v0/issues/{issue_id}/archive by the caller of token: its status, the work item, or the
 * problem's code.
 */
export async function issueArchived(api: Api, token: string, issueId: string) {
  const { data, error, response } = await api.POST("/api/v0/issues/{issue_id}/archive", {
    params: { path: { issue_id: issueId } },
    headers: bearer(token),
  });
  return data ? { status: response.status, issue: data } : answer(response, error);
}

/** The answer of POST /api/v0/issues/{issue_id}/unarchive by the caller of token, as issueArchived's. */
export async function issueUnarchived(api: Api, token: string, issueId: string) {
  const { data, error, response } = await api.POST("/api/v0/issues/{issue_id}/unarchive", {
    params: { path: { issue_id: issueId } },
    headers: bearer(token),
  });
  return data ? { status: response.status, issue: data } : answer(response, error);
}

/** The answer of DELETE /api/v0/issues/{issue_id} by the caller of token: its status, or the problem's code. */
export async function issueDeleted(api: Api, token: string, issueId: string) {
  const { error, response } = await api.DELETE("/api/v0/issues/{issue_id}", {
    params: { path: { issue_id: issueId } },
    headers: bearer(token),
  });
  return answer(response, error);
}

````

- [ ] **Step 2: I15**

`e2e/stories/issue/i15-archive-issue.spec.ts`（新文件，113 行）：

````file e2e/stories/issue/i15-archive-issue.spec.ts
import {
  addProjectMembers,
  amidAnotherWorkspace,
  changeProject,
  createLabel,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import {
  archivedRow,
  changed,
  createdFor,
  expectIssueArchival,
  expectIssueDeleted,
  restoredRow,
} from "../../fixtures/assert/issue";
import { newAccount } from "../../fixtures/auth";
import {
  createIssue,
  issueArchived,
  issueDeleted,
  issueUnarchived,
  issueUpdated,
  updateIssue,
} from "../../fixtures/issues";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";

// I15, archiving and restoring a work item (M4 design 2, 3.8, 3.13): the API version, with P2's assertions (archiving
// and restoring, their two rows of activity, issue.not_archivable, an archived work item not changed, a guest
// refused). The archived list is P5's, the page version P13's. The archived work item's deletion at the end is P2's
// deleteIssue (3.6), whose rows expectIssueDeleted reads from the catalog.

test("I15 (API): a member's work item in a started state is not archived (409); moved to a completed state, she archives it, and archiving it again writes nothing; it is not changed while it is archived (409); a guest who sees it neither restores nor archives it (403); she restores it, and restoring it again writes nothing; each archiving and restoring one row of its activity, hers at its moment; archived again, the admin deletes it, with every row under it, at one moment", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  // ann, Web's member; gus, its guest; each a member of Acme.
  const [ann, gus] = await Promise.all(["ann", "gus"].map((label) => newAccount(api, testInfo, label)));
  if (!ann || !gus) {
    throw new Error("the accounts were not registered");
  }
  await Promise.all([ann, gus].map((who) => inviteAndAccept(api, admin.token, slug, who, 15)));
  const web = await amidAnotherWorkspace(api, admin.token, testInfo, () =>
    createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" })
  );
  await addProjectMembers(api, admin.token, web.id, [
    { member_id: ann.id, role: 15 },
    { member_id: gus.id, role: 5 },
  ]);
  // Web shows its guests every work item: gus sees ann's, and is refused as a guest (403), not as one who sees no
  // such work item (404).
  expect((await changeProject(api, admin.token, web.id, { guest_view_all_features: true })).status).toBe(200);
  const bug = await createLabel(api, admin.token, web.id, { name: "Bug" });
  const { "In Progress": inProgress, Done: done } = await stateIds(api, ann.token, web.id);
  if (!inProgress || !done) {
    throw new Error("Web's states hold no In Progress or no Done");
  }

  // ann's Fix the login, in In Progress, assigned to her and labelled Bug: a row of each table under it. In a state of
  // the started group it is not archived.
  const { id } = await createIssue(api, ann.token, web.id, {
    name: "Fix the login",
    state_id: inProgress,
    assignee_ids: [ann.id],
    label_ids: [bug.id],
  });
  const created = createdFor(ann);
  expect(await issueArchived(api, ann.token, id)).toEqual({ status: 409, code: "issue.not_archivable" });
  await expectIssueArchival(db, id, { archived: false, by: ann.email, activity: created });

  // Moved to Done, she archives it: archived at its moment, one row of its activity; archived again, nothing is
  // written, and the answer is the work item as it is.
  await updateIssue(api, ann.token, id, { state_id: done });
  const toDone = changed("state", "In Progress", "Done", "updated the state to", "Done", "In Progress");
  expect(await issueArchived(api, ann.token, id)).toMatchObject({
    status: 200,
    issue: { id, archived_at: expect.any(String) },
  });
  const archived = { archived: true, by: ann.email, activity: [...created, toDone, archivedRow] };
  await expectIssueArchival(db, id, archived);
  expect(await issueArchived(api, ann.token, id)).toMatchObject({
    status: 200,
    issue: { id, archived_at: expect.any(String) },
  });
  await expectIssueArchival(db, id, archived);

  // While it is archived, she does not change it, and gus, who may not, does not restore it.
  expect(await issueUpdated(api, ann.token, id, { name: "Fix the sign-in" })).toEqual({
    status: 409,
    code: "issue.archived",
  });
  expect(await issueUnarchived(api, gus.token, id)).toEqual({ status: 403, code: "forbidden" });
  await expectIssueArchival(db, id, archived);

  // She restores it: one row; restored again, nothing is written. gus does not archive it either.
  expect(await issueUnarchived(api, ann.token, id)).toMatchObject({ status: 200, issue: { id, archived_at: null } });
  const restored = { archived: false, by: ann.email, activity: [...archived.activity, restoredRow] };
  await expectIssueArchival(db, id, restored);
  expect(await issueUnarchived(api, ann.token, id)).toMatchObject({ status: 200, issue: { id, archived_at: null } });
  expect(await issueArchived(api, gus.token, id)).toEqual({ status: 403, code: "forbidden" });
  await expectIssueArchival(db, id, restored);

  // Archived again, it is deleted as any other: the admin deletes it, and with it every row under it.
  expect((await issueArchived(api, ann.token, id)).status).toBe(200);
  expect(await issueDeleted(api, admin.token, id)).toEqual({ status: 204 });
  await expectIssueDeleted(db, id, admin.email);
});
````

- [ ] **Step 3: I1、I2**

`e2e/stories/issue/i1-create-issue.spec.ts`（修改，12 处）：

````old e2e/stories/issue/i1-create-issue.spec.ts
// assignees, the refusals). The page version is P14's.
````
````new e2e/stories/issue/i1-create-issue.spec.ts
// assignees, the refusals) and P2's (its dates). The page version is P14's.
````
````old e2e/stories/issue/i1-create-issue.spec.ts
test("I1 (API): a member creates a work item in a state, of a priority, assigned to two members and labelled twice: numbered 1, first in its state, with its assignees and labels, the creator and the assignees its subscribers, its creation and each assignee in its activity, all at one moment; without a state or assignees, the next is in the default state and assigned to the project's default assignee while he is its member, the one after to no one once he has left; one in a cancelled state is not completed; an account or a label named twice and a priority in another case are refused at once (422), and so are a state, a label the project lacks and a guest assigned (422); a guest creates none (403), nor does anyone in an archived project (409)", async ({
````
````new e2e/stories/issue/i1-create-issue.spec.ts
test("I1 (API): a member creates a work item in a state, of a priority, assigned to two members, labelled twice, from a start date to a target date: numbered 1, first in its state, with its assignees, labels and dates, the creator and the assignees its subscribers, its creation and each assignee in its activity, all at one moment; without a state or assignees, the next is in the default state and assigned to the project's default assignee while he is its member, the one after to no one once he has left; one in a cancelled state is not completed; an account or a label named twice, a priority in another case and a start date after the target date are refused at once (422), and so are a state, a label the project lacks and a guest assigned (422); a guest creates none (403), nor does anyone in an archived project (409)", async ({
````
````old e2e/stories/issue/i1-create-issue.spec.ts
  // ann creates Fix the login in In Progress, high, assigned to bob and the admin, labelled UI and Bug: the answer
  // gives its accounts and labels in their ids' order.
````
````new e2e/stories/issue/i1-create-issue.spec.ts
  // ann creates Fix the login in In Progress, high, assigned to bob and the admin, labelled UI and Bug, from the 12th
  // to the 19th of October: the answer gives its accounts and labels in their ids' order.
````
````old e2e/stories/issue/i1-create-issue.spec.ts
    label_ids: [ui.id, bug.id],
````
````new e2e/stories/issue/i1-create-issue.spec.ts
    label_ids: [ui.id, bug.id],
    start_date: "2026-10-12",
    target_date: "2026-10-19",
````
````old e2e/stories/issue/i1-create-issue.spec.ts
    name: "Fix the login",
    state_id: inProgress,
````
````new e2e/stories/issue/i1-create-issue.spec.ts
    name: "Fix the login",
    state_id: inProgress,
    start_date: "2026-10-12",
    target_date: "2026-10-19",
````
````old e2e/stories/issue/i1-create-issue.spec.ts
    state: "In Progress",
    priority: "high",
````
````new e2e/stories/issue/i1-create-issue.spec.ts
    state: "In Progress",
    priority: "high",
    start_date: "2026-10-12",
    target_date: "2026-10-19",
````
````old e2e/stories/issue/i1-create-issue.spec.ts
    state: "Backlog",
    priority: "none",
    sort_order: 65535,
````
````new e2e/stories/issue/i1-create-issue.spec.ts
    state: "Backlog",
    priority: "none",
    start_date: null,
    target_date: null,
    sort_order: 65535,
````
````old e2e/stories/issue/i1-create-issue.spec.ts
    state: "Backlog",
    priority: "none",
    sort_order: 75535,
````
````new e2e/stories/issue/i1-create-issue.spec.ts
    state: "Backlog",
    priority: "none",
    start_date: null,
    target_date: null,
    sort_order: 75535,
````
````old e2e/stories/issue/i1-create-issue.spec.ts
    state: "Cancelled",
    priority: "none",
````
````new e2e/stories/issue/i1-create-issue.spec.ts
    state: "Cancelled",
    priority: "none",
    start_date: null,
    target_date: null,
````
````old e2e/stories/issue/i1-create-issue.spec.ts
  // An account and a label named twice and a priority in another case (the contract's values are lower case) are
  // refused, the three at once; so are a state and a label Web does not have and gus, Web's guest, as an assignee:
  // every problem in one answer.
````
````new e2e/stories/issue/i1-create-issue.spec.ts
  // An account and a label named twice, a priority in another case (the contract's values are lower case) and a start
  // date after the target date are refused, the four at once; so are a state and a label Web does not have and gus,
  // Web's guest, as an assignee: every problem in one answer.
````
````old e2e/stories/issue/i1-create-issue.spec.ts
      label_ids: [bug.id, bug.id],
````
````new e2e/stories/issue/i1-create-issue.spec.ts
      label_ids: [bug.id, bug.id],
      start_date: "2026-10-20",
      target_date: "2026-10-19",
````
````old e2e/stories/issue/i1-create-issue.spec.ts
      { field: "label_ids[1]", code: "duplicate" },
````
````new e2e/stories/issue/i1-create-issue.spec.ts
      { field: "label_ids[1]", code: "duplicate" },
      { field: "target_date", code: "not_allowed" },
````

`e2e/stories/issue/i2-update-issue.spec.ts`（修改，12 处）：

````old e2e/stories/issue/i2-update-issue.spec.ts
} from "../../fixtures/api";
import {
````
````new e2e/stories/issue/i2-update-issue.spec.ts
} from "../../fixtures/api";
import {
  archivedRow,
````
````old e2e/stories/issue/i2-update-issue.spec.ts
  createdFor,
````
````new e2e/stories/issue/i2-update-issue.spec.ts
  createdFor,
  expectIssueArchival,
````
````old e2e/stories/issue/i2-update-issue.spec.ts
import { createIssue, issueUpdated, updateIssue, type IssueUpdate } from "../../fixtures/issues";
````
````new e2e/stories/issue/i2-update-issue.spec.ts
import { createIssue, issueArchived, issueUpdated, updateIssue, type IssueUpdate } from "../../fixtures/issues";
````
````old e2e/stories/issue/i2-update-issue.spec.ts
// priority, assignees and labels, each change's rows of its activity, completed_at, who changed it last and when). The
// dates are P2's, the description P4's, the page version P14's.
````
````new e2e/stories/issue/i2-update-issue.spec.ts
// priority, assignees and labels, each change's rows of its activity, completed_at, who changed it last and when) and
// P2's (its dates and their rows, the dates held in order once merged, an archived work item not changed). The
// description is P4's, the page version P14's.
````
````old e2e/stories/issue/i2-update-issue.spec.ts
test("I2 (API): a member renames a work item; another makes it high and moves it to a completed state, which completes it; the first assigns it to another account instead of himself and labels it otherwise, the others kept, taking off rows the second wrote; its values sent as they are write nothing; an account and a label added again and a new label, the two labels out of their ids' order, are added in id order; a label, then an account, taken off alone each change it, the rows taken off before keeping their stamps; moving it back completes it no more; problems of the values are refused before any read, alike for a work item that is not there, and a state and a label the project lacks under the lock, each set at once (422), writing nothing; a work item sharing an account and a label with it is left as created: each change one row of its activity a field, a label or an account added or taken off, all its rows the changer's at its moment", async ({
````
````new e2e/stories/issue/i2-update-issue.spec.ts
test("I2 (API): a member renames a work item; another makes it high and moves it to a completed state, which completes it; the first assigns it to another account instead of himself and labels it otherwise, the others kept, taking off rows the second wrote; its values sent as they are write nothing; an account and a label added again and a new label, the two labels out of their ids' order, are added in id order; a label, then an account, taken off alone each change it, the rows taken off before keeping their stamps; moving it back completes it no more; its dates are set, then a start date alone after the target date kept and a target date alone before the start date kept are refused (422), both moved past the target date it had are taken, and the target date is cleared; problems of the values are refused before any read, alike for a work item that is not there, and a state and a label the project lacks under the lock, each set at once (422), writing nothing; a work item sharing an account and a label with it is left as created; an archived work item is not changed (409): each change one row of its activity a field, a label or an account added or taken off, all its rows the changer's at its moment", async ({
````
````old e2e/stories/issue/i2-update-issue.spec.ts
    state: "Backlog",
    priority: "none",
    completed: false,
````
````new e2e/stories/issue/i2-update-issue.spec.ts
    state: "Backlog",
    priority: "none",
    start_date: null,
    target_date: null,
    completed: false,
````
````old e2e/stories/issue/i2-update-issue.spec.ts
    name: "Fix the sign-in",
    state: "Done",
    priority: "high",
    completed: true,
    by: ann.email,
````
````new e2e/stories/issue/i2-update-issue.spec.ts
    name: "Fix the sign-in",
    state: "Done",
    priority: "high",
    start_date: null,
    target_date: null,
    completed: true,
    by: ann.email,
````
````old e2e/stories/issue/i2-update-issue.spec.ts
    name: "Fix the sign-in",
    state: "Done",
    priority: "high",
    completed: true,
    by: bob.email,
````
````new e2e/stories/issue/i2-update-issue.spec.ts
    name: "Fix the sign-in",
    state: "Done",
    priority: "high",
    start_date: null,
    target_date: null,
    completed: true,
    by: bob.email,
````
````old e2e/stories/issue/i2-update-issue.spec.ts
  await expectIssueUpdated(db, id, movedBack);

  // A priority in another case (the contract's values are lower case), an account and a label named twice are refused
````
````new e2e/stories/issue/i2-update-issue.spec.ts
  await expectIssueUpdated(db, id, movedBack);

  // ann gives it a start date and a target date, the 12th and the 19th of October: a row each, the start date's first.
  await updateIssue(api, ann.token, id, { start_date: "2026-10-12", target_date: "2026-10-19" });
  const dated = {
    ...movedBack,
    start_date: "2026-10-12",
    target_date: "2026-10-19",
    by: ann.email,
    activity: [
      ...movedBack.activity,
      changed("start_date", "", "2026-10-12", "updated the start date to "),
      changed("target_date", "", "2026-10-19", "updated the target date to"),
    ],
    wrote: 2,
  };
  await expectIssueUpdated(db, id, dated);
  // The dates it then has are held in order, each given one merged with the one it keeps: a start date alone after the
  // target date is refused at start_date, a target date alone before the start date at target_date, writing nothing;
  // both moved past the target date it had, in order, are taken.
  expect(await issueUpdated(api, bob.token, id, { start_date: "2026-10-20" })).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [{ field: "start_date", code: "not_allowed" }],
  });
  expect(await issueUpdated(api, bob.token, id, { target_date: "2026-10-11" })).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [{ field: "target_date", code: "not_allowed" }],
  });
  await expectIssueUpdated(db, id, dated);
  await updateIssue(api, bob.token, id, { start_date: "2026-10-20", target_date: "2026-10-26" });
  const redated = {
    ...dated,
    start_date: "2026-10-20",
    target_date: "2026-10-26",
    by: bob.email,
    activity: [
      ...dated.activity,
      changed("start_date", "2026-10-12", "2026-10-20", "updated the start date to "),
      changed("target_date", "2026-10-19", "2026-10-26", "updated the target date to"),
    ],
    wrote: 2,
  };
  await expectIssueUpdated(db, id, redated);
  // bob clears its target date: its row's new value is "".
  await updateIssue(api, bob.token, id, { target_date: null });
  const cleared = {
    ...redated,
    target_date: null,
    activity: [...redated.activity, changed("target_date", "2026-10-26", "", "updated the target date to")],
    wrote: 1,
  };
  await expectIssueUpdated(db, id, cleared);

  // A priority in another case (the contract's values are lower case), an account and a label named twice are refused
````
````old e2e/stories/issue/i2-update-issue.spec.ts
  });
  await expectIssueUpdated(db, id, movedBack);
````
````new e2e/stories/issue/i2-update-issue.spec.ts
  });
  await expectIssueUpdated(db, id, cleared);
````
````old e2e/stories/issue/i2-update-issue.spec.ts
    state: "Backlog",
    priority: "none",
    sort_order: logout.sort_order,
````
````new e2e/stories/issue/i2-update-issue.spec.ts
    state: "Backlog",
    priority: "none",
    start_date: null,
    target_date: null,
    sort_order: logout.sort_order,
````
````old e2e/stories/issue/i2-update-issue.spec.ts
    activity: createdFor(bob),
  });
````
````new e2e/stories/issue/i2-update-issue.spec.ts
    activity: createdFor(bob),
  });

  // An archived work item is not changed (409 issue.archived): ann's Ship it, in Done, which she archives; bob's
  // change writes nothing, so it stays ann's archiving.
  const shipped = await createIssue(api, ann.token, web.id, { name: "Ship it", state_id: done });
  expect((await issueArchived(api, ann.token, shipped.id)).status).toBe(200);
  expect(await issueUpdated(api, bob.token, shipped.id, { target_date: "2026-10-19" })).toEqual({
    status: 409,
    code: "issue.archived",
  });
  await expectIssueArchival(db, shipped.id, {
    archived: true,
    by: ann.email,
    activity: [...createdFor(), archivedRow],
  });
````

- [ ] **Step 4: 检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 `keywords: 72 rules, 23 exceptions, no hits.`

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 123 个全部通过（新的是 I15）。

- [ ] **Step 5: 提交**

```bash
git add e2e/fixtures/assert/issue.ts e2e/fixtures/issues.ts e2e/stories/issue/i1-create-issue.spec.ts e2e/stories/issue/i15-archive-issue.spec.ts e2e/stories/issue/i2-update-issue.spec.ts
```
```bash
git commit -m "test(M4/P2): the stories of archiving, deleting and dating a work item

I15 drives the API: a work item in progress is not archived; moved into
Done it is, once (the second archiving writes nothing); archived, it is
not changed; the project's guest neither restores nor archives it; it is
restored, once; archived again, its project's admin deletes it, and every
row under it, by the catalog's foreign keys, is deleted with it at that
moment. I1 and I2 give, change and clear the start and target dates, and
are refused dates out of order, the update's as merged; I2 changes an
archived work item in vain.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**：本 Task 只加故事和夹具，没有自己的变异：它的故事是较早的 Task 的变异的端到端一层。

本 Task 的检查也发现较早的 Task 的变异 `T1.5`、`T1.8`、`T1.9`、`T1.11`、`T4.2`、`T4.3`、`T4.5`、`T4.11`、`T7.3`、`T7.4`、`T7.6`、`T7.7`、`T7.8`、`T7.10`、`T7.11`、`T8.1`、`T10.1`、`T10.3`、`T10.4`、`T10.5`、`T10.6`、`T10.8`、`T10.9`、`T10.10`、`T10.11`、`T10.12`、`T10.13`、`T10.14`、`T11.1`、`T11.2`、`T11.3`、`T11.4`（列在各自的 Task 的表里，标"（Task 13 起）"）：本 Task 做完之后，在本 Task 的树上重跑它们（`mutants_p2.py`，`MUT_LATER=13`）。

---

### Task 14: 文档：3.23 中 P2 的行，三份交接的处理结果

**Files:**
- Modify: `docs/v0/M2-auth/M2-design.md`、`docs/v0/M4-issue-core/handoffs/M1-P3-trim-platform.md`、`docs/v0/M4-issue-core/handoffs/M2-closeout.md`、`docs/v0/M4-issue-core/handoffs/M3-closeout.md`、`docs/v0/plane-diff.md`、`docs/v0/v0-design.md`

**Interfaces:**（spec 2.15、第 7 节；M4 设计 3.23、4.17）
- 总体设计 6.5：`AllowCreator` 一句改写（创建者要是项目的有效成员、项目角色不低于成员，锁下再判定）；`OwnOnly` 和一条访客规则（预检和锁下的判定各执行一次，404 先于角色的 403）；Plane 的两条照原样。
- M2 设计 3.11：字段错误的路径按下标的数值排序；不限类型的节点里的数；`shared.Date` 的检查器（`FormatDate`）；`date` 从生成器拒绝的例子里去掉（映射之外的仍被拒绝）。
- 差异清单第四节 4.17：P2 的七行（起止日期、创建者的权限、访客只看自己的工作项、访客删除自己建的工作项、已归档的工作项、归档和恢复、按编号取），接在"删除状态"一行之后；归档和恢复一行写明 Plane 恢复没归档的也答 404，动态的 `comment` 与 Plane 不同（spec 第 3 节第 8 条）。
- 三份交接（`M1-P3-trim-platform.md`、`M2-closeout.md`、`M3-closeout.md`）各写下"处理结果（M4/P2）"，`status` 照旧 `open`（spec 第 7 节）。

**Tests:** 只改文档：`make lint-web`（关键词守卫读文档）和 `doclinks.py`（相对链接，本 spec 和 plan 也在内）。

- [ ] **Step 1: 设计和差异清单**

`docs/v0/M2-auth/M2-design.md`（修改，4 处）：

````old docs/v0/M2-auth/M2-design.md
      - **报哪 16 个**：按固定的顺序读，遇到的前 16 个。扫描按文档的顺序读；按表检查按名字的顺序读一个对象的成员，深度优先，读完成员再报它缺少的必填成员。然后按路径排序。所以同一个请求体每次得到同样的 16 个，但不一定是路径最小的 16 个：16 个未声明的成员可以把缺少的 `email` 挤出去。
````
````new docs/v0/M2-auth/M2-design.md
      - **报哪 16 个**：按固定的顺序读，遇到的前 16 个。扫描按文档的顺序读；按表检查按名字的顺序读一个对象的成员，深度优先，读完成员再报它缺少的必填成员。然后按路径排序，数组的下标按数值比较（`tags[2]` 在 `tags[10]` 之前，M4/P2 起；之前按字典序）。所以同一个请求体每次得到同样的 16 个，但不一定是路径最小的 16 个：16 个未声明的成员可以把缺少的 `email` 挤出去。
````
````old docs/v0/M2-auth/M2-design.md
    - 格式检查器按**生成的 Go 类型**登记：`time.Time`、标准库的 `uuid.UUID` 各一个，都是上面那一行 `json.Unmarshal`。`bodyshape` 只依赖标准库。
      - 上一稿还为 `format: date` 登记了 `openapi_types.Date`，让 `bodyshape` 依赖 `oapi-codegen/runtime/types`，而那个包导入 `github.com/google/uuid`（核验 F1，3.12）。M2 没有 `date` 字段，这个检查器删去；生成器遇到 `date` 就失败（见下）。以后第一个用到 `date` 的 M 选定它的 Go 类型，带着测试登记检查器。
````
````new docs/v0/M2-auth/M2-design.md
    - 格式检查器按**生成的 Go 类型**登记：`time.Time`、标准库的 `uuid.UUID` 各一个，都是上面那一行 `json.Unmarshal`；M4/P2 起加 `shared.Date`（`format: date`）。`bodyshape` 只依赖标准库。
      - 上一稿还为 `format: date` 登记了 `openapi_types.Date`，让 `bodyshape` 依赖 `oapi-codegen/runtime/types`，而那个包导入 `github.com/google/uuid`（核验 F1，3.12）。M2 没有 `date` 字段，这个检查器删去；生成器遇到 `date` 就失败（见下）。以后第一个用到 `date` 的 M 选定它的 Go 类型，带着测试登记检查器。
      - M4/P2 选定 `shared.Date`（不带时区的一天，`YYYY-MM-DD`）：各模块的 `oapi-codegen.yaml` 把 `date` 映射过去（M4 设计 6.1）。它是业务的类型，平台不导入 `internal/shared`，所以 `date` 的检查器先把字符串解码出来，再照 `shared.ParseDate` 的规则读它（`time.DateOnly`，年份不小于 1）；`issue` 的 HTTP 适配器的测试核对两者对同一组写法（含 JSON 转义）的结论与生成的解码器相同。
````
````old docs/v0/M2-auth/M2-design.md
      - **格式的清单取自同一份 `type-mapping`**：生成器读这个模块的 `oapi-codegen.yaml` 中的 `output-options.type-mapping`，与 oapi-codegen 的默认映射合并（`codegen.DefaultTypeMapping.Merge`，与 `oapi-codegen/v2@v2.8.0/pkg/codegen/codegen.go:162-165` 的做法相同），得到每个字段实际生成的 Go 类型。生成为 `string` 的格式（`email`）不检查，交给领域层；生成为已登记检查器的类型（`time.Time`、`uuid.UUID`）的，表中记下检查器；生成为其他类型的，生成失败并说明原因。数值同理：`bodyshape` 的 `Integer` 是 int64 容纳得下的字面量、`Number` 是 float64 容纳得下的，所以 `integer` 只能生成为 `int` 或 `int64`，`number` 只能生成为 `float64`（模板把 `number` 的默认映射改为 `float64`，3.12）；生成为更窄的整数、无符号整数、`float32`，或 `format` 不在映射里的，生成失败。模板改了映射，表随之改变，两边不会走样。
````
````new docs/v0/M2-auth/M2-design.md
      - **格式的清单取自同一份 `type-mapping`**：生成器读这个模块的 `oapi-codegen.yaml` 中的 `output-options.type-mapping`，与 oapi-codegen 的默认映射合并（`codegen.DefaultTypeMapping.Merge`，与 `oapi-codegen/v2@v2.8.0/pkg/codegen/codegen.go:162-165` 的做法相同），得到每个字段实际生成的 Go 类型。生成为 `string` 的格式（`email`）不检查，交给领域层；生成为已登记检查器的类型（`time.Time`、`uuid.UUID`）的，表中记下检查器；生成为其他类型的，生成失败并说明原因。数值同理：`bodyshape` 的 `Integer` 是 int64 容纳得下的字面量、`Number` 是 float64 容纳得下的，所以 `integer` 只能生成为 `int` 或 `int64`，`number` 只能生成为 `float64`（模板把 `number` 的默认映射改为 `float64`，3.12）；生成为更窄的整数、无符号整数、`float32`，或 `format` 不在映射里的，生成失败。模板改了映射，表随之改变，两边不会走样。不限类型的节点（`{}`、开放对象的成员、没有 `items` 的数组的元素）里的数也看范围：float64 放不下的数字面量（`1e400`）在它的路径上报 `invalid_format`，一遍读完那个值（M4/P2 起；之前由解码器笼统地答 400）。
````
````old docs/v0/M2-auth/M2-design.md
      - 支持的写法：对象（属性、必填、`additionalProperties` 为 `false`、`true` 或一个 schema）、数组、标量、`type: [X, 'null']` 和 `anyOf: [X, {type: 'null'}]`，以及上一条的格式。遇到其他 `anyOf`/`oneOf`/`allOf`，或生成为没有检查器的类型的格式（例如 `date`、`byte`、`binary`、`duration`），生成失败并说明原因。
````
````new docs/v0/M2-auth/M2-design.md
      - 支持的写法：对象（属性、必填、`additionalProperties` 为 `false`、`true` 或一个 schema）、数组、标量、`type: [X, 'null']` 和 `anyOf: [X, {type: 'null'}]`，以及上一条的格式。遇到其他 `anyOf`/`oneOf`/`allOf`，或生成为没有检查器的类型的格式（例如 `byte`、`binary`、`duration`；`date` 从 M4/P2 起映射为 `shared.Date`），生成失败并说明原因。
````

`docs/v0/plane-diff.md`（修改，1 处）：

````old docs/v0/plane-diff.md
| 删除状态 | 有工作项时 400 | 有工作项在它里面（已归档的也算）时 409 `project.state_in_use`，这一检查在默认状态的检查之前（M4 设计 3.6） |
````
````new docs/v0/plane-diff.md
| 删除状态 | 有工作项时 400 | 有工作项在它里面（已归档的也算）时 409 `project.state_in_use`，这一检查在默认状态的检查之前（M4 设计 3.6） |
| 工作项的起止日期 | 两个都在请求里才检查开始不晚于截止 | 按合并之后的值检查：只给一个日期时与存着的另一个比较，开始晚于截止 422 `not_allowed`（在 `target_date`，只给了 `start_date` 时在 `start_date`）；同一天允许；数据库另有 `issues_dates_check`（M4 设计 3.7） |
| 工作项的创建者的权限 | 只看工作区成员关系，已离开项目也能读、改、删 | 创建者只在删除上另有权限（`issue.delete` 的 `AllowCreator`）：他须仍是项目的有效成员、角色不低于成员；离开项目或成了访客之后不能删（看不到项目时 404，看得到时 403）（M4 设计 3.4） |
| 访客只看自己的工作项 | 只在约 6 处执行 | 项目没开 `guest_view_all_features` 时，它的访客（按项目角色算，工作区管理员在项目里是访客时同样）只看得到自己创建的工作项：取、按编号取、修改、删除、归档、恢复别人的都答 404 `issue.not_found`；读在判定之后执行这条规则，写在不加锁的预检和锁下的判定中各执行一次（M4 设计 3.4；列表、子资源在各自的 Phase） |
| 访客删除自己建的工作项 | 服务端允许，页面不显示 | 403 `forbidden`（M4 设计 3.4） |
| 已归档的工作项 | 只挡修改；评论、链接、表情、关联不挡；能成为父工作项 | 修改 409 `issue.archived`，什么都没改的修改也是；恢复、删除照常；在已归档的项目里先答 409 `project.archived`（M4 设计 3.8；子资源的写、成为父工作项和关联的对方在 M4 的各自的 Phase） |
| 工作项的归档、恢复 | 归档答 `{archived_at}`，恢复 204；再归档、恢复没归档的 404；状态不在"已完成""已取消"组时 400 | 都答 `IssueDetail`；已归档的再归档、未归档的再恢复不写、200；状态不在"已完成""已取消"组时 409 `issue.not_archivable`；各写一条动态（`archived_at` 的 `manual_archive`、`restore`），动态的 `comment` 不同（M4 设计 3.8、3.13） |
| 按编号取工作项 | 格式不对 400，不是成员 403 | `GET /api/v0/workspaces/{slug}/issues/{identifier}`：不是工作区的有效成员 404 `workspace.not_found`；之后格式不对、项目或编号不存在、看不到、访客规则都是同一个 404 `issue.not_found`；标识经同一个折叠比较（`identifier-fold.json`，`i`、`ı`、`I`、`İ` 一样）（M4 设计 3.7） |
````

`docs/v0/v0-design.md`（修改，1 处）：

````old docs/v0/v0-design.md
  "创建者本人可以修改或删除自己创建的对象"由 M4 随第一条用到它的规则在规则表加入（`AllowCreator`）。
````
````new docs/v0/v0-design.md
  创建者本人的规则（`AllowCreator`）随第一条用到它的规则 `issue.delete`（管理员 + `AllowCreator`）在 M4/P2 加入规则表：判定读目标的创建者（`Target.CreatorID`），调用者是它、仍是项目的有效成员、角色不低于成员时同样允许，离开了项目或成了访客的创建者不行（Plane 只看工作区成员关系，M4 设计 3.4）。访客的规则是判定交回的 `Grant.OwnOnly`：调用者是项目的访客（按项目角色算，工作区管理员在项目里是访客时同样）、项目没开 `guest_view_all_features` 时为真。同一条访客规则用在工作项的每个读、写和列表上，看不到的工作项和挂在它下面的一切都答 404（Plane 只在约 6 处执行；单个工作项的读写从 M4/P2 起，列表随 M4 的各 Phase，M4 设计 3.4）。
````

- [ ] **Step 2: 交接**

`docs/v0/M4-issue-core/handoffs/M1-P3-trim-platform.md`（修改，1 处）：

````old docs/v0/M4-issue-core/handoffs/M1-P3-trim-platform.md
来源：[M4/P1a spec](../specs/P1a-issues.md) 第 7 节。

````
````new docs/v0/M4-issue-core/handoffs/M1-P3-trim-platform.md
来源：[M4/P1a spec](../specs/P1a-issues.md) 第 7 节。

## 处理结果（M4/P2）

- **地址，按编号查找的一半**（接口一侧完成）：按编号取工作项是 `GET /api/v0/workspaces/{slug}/issues/{identifier}`（`getIssueByIdentifier`，M4 设计 3.7、3.18），与其余工作项的接口同在 `/api/v0` 下；前端的 `/work-items/` 字面量随前端改接新的 service 删除（关键词守卫 `old-issue-urls`，M4 的 P10a 起）。

仍未处理，状态保持 `open`：动态的字段名里没有 `WORKLOG`、`ISSUE_ADDITIONAL_PROPERTIES_ACTIVITY`（M4 设计 3.13 的表）、评论没有 `access`（都在 M4 的 P4）；描述历史的地址（M4 的 P4）和前端的两处 `/work-items/` 字面量（M4 的前端各 Phase）；富文本筛选的显示层（M4 的 P10b，关键词守卫的终态在 P16）；收集箱的来源（以转交关闭：收尾写进 M7 的交接；M4 删除工作项动态中显示来源的一支在 P15，M4 设计 13.1）。

来源：[M4/P2 spec](../specs/P2-issue-access.md) 第 7 节。

````

`docs/v0/M4-issue-core/handoffs/M2-closeout.md`（修改，1 处）：

````old docs/v0/M4-issue-core/handoffs/M2-closeout.md
来源：[M4/P1b spec](../specs/P1b-issue-update.md) 第 7 节。

````
````new docs/v0/M4-issue-core/handoffs/M2-closeout.md
来源：[M4/P1b spec](../specs/P1b-issue-update.md) 第 7 节。

## 处理结果（M4/P2）

- **第 3 节，`updated_at`**（P2 的写完成）：`deleteIssue`、`archiveIssue`、`unarchiveIssue` 和带日期的 `updateIssue` 把用例的时钟在项目 N 之下读到的一个时刻写进它写的每一行：删除写下工作项和它的负责人、标签、关注、动态的 `deleted_at`、`updated_at`，归档、恢复写下工作项的 `archived_at`（恢复为空）、`updated_at` 和那一行动态的 `created_at`、`updated_at`。用例的测试用固定的时钟（`clockAt`）断言交给仓储的时刻等于它，仓储的测试断言这些列等于交给它的时刻；组合的测试断言它落在请求之内（`TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`、`TestArchivingAWorkItemStampsItsRows`）；端到端的 I15 断言归档、恢复的那一行动态是工作项的 `updated_at`、是写它的人的，删除的每一行同一时刻、同一个删除者（`expectIssueDeleted`）。
- **第 4 节，路径的排序**（完成）：字段错误的路径按数组下标的数值排序（`bodyshape` 的 `comparePaths`，`tags[2]` 在 `tags[10]` 之前，一个全序）；单元测试 `TestComparePaths` 和 `Check` 的表测试（`ids[2]` 在 `ids[10]` 之前）；M2 设计 3.11 改写。
- **第 4 节，数的范围**（完成）：不限类型的节点（`{}`、开放对象的成员、没有 `items` 的数组的元素）里 float64 放不下的数在它的路径上报 `invalid_format`（`ambiguity.go` 的 `numbers`，一遍读完那个值）；`Check` 的表测试在这三种节点里各放一个 `1e400`；M2 设计 3.11 改写。
- **第 4 节，第一个 `format: date` 的字段**（完成）：Go 类型是 `shared.Date`，五个模块的 `oapi-codegen.yaml` 都映射过去；`bodyshape` 登记 `FormatDate`（检查器照 `shared.ParseDate` 的规则读，平台不导入 `shared`），生成器认它；`issue` 的 HTTP 适配器的 `TestDatesPassTheBodyCheckAsTheyDecode` 核对检查器与生成的解码器对同一组写法结论相同。不合格的日期得到带路径的 400：`bodyshape` 的 `TestFormatDate`；`apitest` 为契约的每个日期字段生成一个不合格的请求体用例（`TestBodyCasesOfDates`），整程序测试 `TestBodiesThatBreakTheStructureAnswer400` 把它们发给 `createIssue`、`updateIssue`。
- **第 4 节，结构检查的代价**（数组完成）：`TestCheckCostsAboutTheBody` 加上字符串的数组（每个元素一个格式问题、一个很长的日期字符串）和开放节点里很深的数；第一个数组的请求体（P1a 的 `assignee_ids`）的代价由它守住。开放 map 和递归的请求体 schema：M4/P2 没有用到。
- **第 4 节，map 型对象**：M4/P2 没有用到（契约里没有 `additionalProperties` 为 schema 的对象），`closedObject` 不改。

仍未处理，状态保持 `open`：第 3 节其余写工作项的用例（M4 的 P3、P4、P7）和自动归档推进固定时钟的测试（M4 的 P6）；第 4 节的草稿发布（M4 的 P8）、map 型对象和递归的请求体 schema（第一个用到它们的 Phase，或 M4 的 review 写明没有用到）；第 1、2、5–13 节，按 M4 设计 13.1 的落点。

来源：[M4/P2 spec](../specs/P2-issue-access.md) 第 7 节。

````

`docs/v0/M4-issue-core/handoffs/M3-closeout.md`（修改，1 处）：

````old docs/v0/M4-issue-core/handoffs/M3-closeout.md
来源：[M4/P1b spec](../specs/P1b-issue-update.md) 第 7 节。

````
````new docs/v0/M4-issue-core/handoffs/M3-closeout.md
来源：[M4/P1b spec](../specs/P1b-issue-update.md) 第 7 节。

## 处理结果（M4/P2）

- **第 3 节，`AllowCreator`**（完成）：随第一条用到它的规则 `issue.delete`（管理员 + `AllowCreator`）加入规则表；判定读目标的创建者（`shared.Target.CreatorID`），调用者是它、在项目里的有效角色不低于成员时允许，已离开项目、成了访客的创建者不行（M4 设计 3.4）；判定表测试逐格核对。
- **第 3 节，`guest_view_all_features`**（工作项的取、改、删、按编号取、归档、恢复完成；列表在各自的 Phase）：`access` 读项目的 `guest_view_all_features`，判定交回 `Grant.OwnOnly`（项目的访客、项目没开它时为真，按项目角色算：工作区管理员在项目里是访客时同样）；一条访客规则在不加锁的预检和锁下的判定中各执行一次，别人建的工作项答 404 `issue.not_found`，不是 403（M4 设计 3.4、3.5）；交错 5 加上访客规则下 404 的一支。
- **第 3 节，矩阵和码**（工作项的操作到 P2 为止完成）：P2 的 `getIssueByIdentifier`、`deleteIssue`、`archiveIssue`、`unarchiveIssue` 在规则表、操作名、矩阵中各有一行；矩阵有了工作项表（M4 设计 9.2），P1a、P1b 的 `createIssue`、`getIssue`、`updateIssue` 的行移到它上面：项目级的各列之外，建了它的项目成员，项目访客的三种情形（看得到全部的；只看自己的，对自己建的、对别人建的），只是项目访客的工作区管理员（对别人建的、对自己建的）；设计 9.2 另列的"建了它、后来被降为访客"（403）和"建了它、已离开项目"（404）是"访客·自己"（种子记在他名下的）和"已离开项目"（他离开之前建的）两列上 `deleteIssue` 的格子。新码 `issue.archived`、`issue.not_archivable` 由 `issue` 的 HTTP 测试返回过（`apitest.Main` 两个方向），带前端文案。决策点 3 的个人主页一行在 M4 的 P5。
- **第 4 节，`guest_view_all_features` 的说明**（完成）：契约的 `Project.guest_view_all_features` 改写为它的作用（项目的访客是否看得到项目的每个工作项；不看时别人建的答 `issue.not_found`）。
- **第 4 节，`expectIssueDeleted`**（完成）：`e2e/fixtures/assert/issue.ts` 照 `expectProjectDeleted` 从目录读表，但沿外键图从 `issues` 往下走：一行的外键指向工作项或它下面的一行，它就在工作项下面（评论的表情指向评论，P4 起同样断言到），每个外键在工作项下面都有一行与它同一时刻、同一个删除者删除，删除之前已删除的不算（它保留原来的删除由按目录的删除测试核对）；子工作项的外键（`issues.parent_id`）在 P3 之前没有行，豁免到 P3。第一个使用者是 I15 的接口版本（删除已归档的工作项）。

仍未处理，状态保持 `open`：第 2 节其余各项：加关联的另一端的预检（M4 的 P3）、P2 之后各 Phase 的批量写按约定五核对（到 M4 的 P9）、物理删除账户（M4 的 P9）；第 3 节决策点 3 的个人主页一行（M4 的 P5）和工作项列表的访客规则（各列表的 Phase）；第 4 节的 `archive_in`（M4 的 P6）的说明、删除断言的限时轮询（M4 的 P9）和 `expectIssueDeleted` 中子工作项的豁免（M4 的 P3）；第 1、5–9 节，按 M4 设计 13.1 的落点。

来源：[M4/P2 spec](../specs/P2-issue-access.md) 第 7 节。

````

- [ ] **Step 3: 检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 `keywords: 72 rules, 23 exceptions, no hits.`

Run: `python3 $M4TMP/p2tools/doclinks.py . docs/v0/v0-design.md docs/v0/M2-auth/M2-design.md docs/v0/plane-diff.md docs/v0/M4-issue-core/handoffs/M1-P3-trim-platform.md docs/v0/M4-issue-core/handoffs/M2-closeout.md docs/v0/M4-issue-core/handoffs/M3-closeout.md docs/v0/M4-issue-core/specs/P2-issue-access.md docs/v0/M4-issue-core/plans/P2-issue-access.md`
Expected: 每个文件 `0 broken`，最后一行 `broken: 0`。

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`（文档不影响它；本 Phase 最后一次全量运行）。

- [ ] **Step 4: 提交**

```bash
git add docs/v0/M2-auth/M2-design.md docs/v0/M4-issue-core/handoffs/M1-P3-trim-platform.md docs/v0/M4-issue-core/handoffs/M2-closeout.md docs/v0/M4-issue-core/handoffs/M3-closeout.md docs/v0/plane-diff.md docs/v0/v0-design.md
```
```bash
git commit -m "docs(M4/P2): the design documents and the handoffs record P2

The overall design's AllowCreator sentence says whom it lets (an active
member of the project, of the role member or above, decided again under
the lock), beside OwnOnly and the one guest rule. The M2 design's body
check orders indexes by number, checks numbers in values of any type and
reads shared.Date. The Plane diff holds P2's rows, and the three
handoffs' outcomes say what P2 did.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**：本 Task 只改文档，没有变异（文档的相对链接由 `doclinks.py` 核对）。
